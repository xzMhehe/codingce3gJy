package ezfy

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/model"
)

// parseActWildTroops 解析活动野地配置的守军 JSON → 战斗兵组列表
//
// 支持两种格式（★ 2026-10-14 与野地类型同口径，改成范围配）：
//   - [[兵种id,数量],...]：固定数量（兼容旧数据）
//   - [[兵种id,最小,最大],...]：范围内随机数量
//
// 失败/空串返回 nil。兵种id<=0 或 数量<=0 的行丢弃。
func parseActWildTroops(raw string) []ezfyUnitGroup {
	var rows [][]int64
	if err := json.Unmarshal([]byte(raw), &rows); err != nil {
		return nil
	}
	out := []ezfyUnitGroup{}
	for _, r := range rows {
		if len(r) < 2 || r[0] <= 0 {
			continue
		}
		count := r[1]
		if len(r) >= 3 {
			lo, hi := r[1], r[2]
			if hi < lo {
				hi = lo
			}
			count = lo
			if hi > lo {
				count = lo + rand.Int63n(hi-lo+1)
			}
		}
		if count <= 0 {
			continue
		}
		out = append(out, ezfyUnitGroup{TroopId: int(r[0]), Count: count})
	}
	return out
}

// parseActWildTroopRanges 解析活动野地守军配置为范围预览 [[兵种id,最小,最大],...]，
// 兼容旧格式 [兵种id,数量]（min=max=数量）。失败/空 → nil。
func parseActWildTroopRanges(raw string) [][]int64 {
	var rows [][]int64
	if err := json.Unmarshal([]byte(raw), &rows); err != nil {
		return nil
	}
	out := [][]int64{}
	for _, r := range rows {
		if len(r) < 2 || r[0] <= 0 {
			continue
		}
		lo, hi := r[1], r[1]
		if len(r) >= 3 {
			hi = r[2]
			if hi < lo {
				hi = lo
			}
		}
		if lo <= 0 {
			continue
		}
		out = append(out, []int64{r[0], lo, hi})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// parseActWildTreasures 解析活动野地配置的必掉宝物 JSON [[cfg_id,count],...]
// 失败/空串返回 nil。id<=0 或 count<=0 的行丢弃。
func parseActWildTreasures(raw string) [][2]int {
	var rows [][2]int
	if err := json.Unmarshal([]byte(raw), &rows); err != nil {
		return nil
	}
	out := [][2]int{}
	for _, r := range rows {
		if r[0] <= 0 || r[1] <= 0 {
			continue
		}
		out = append(out, r)
	}
	return out
}

// ezfyActDropReady 活动目标掉落冷却是否已过：距该玩家上次该类掉落 >= cooldownSec 秒（或从未掉过）→ true。
//
// ★ 冷却按**玩家**维度算（不分活动目标类型/等级）。上次掉落时间戳(ms)存在 ezfy_profile 的 col 列
// （宝物=act_treasure_at / 商城道具=act_item_at），0 = 从未掉过。
// cooldownSec <= 0 由调用方直接放行（= 关闭冷却），这里只处理 > 0 的判定。
func (h *EzfyHandler) ezfyActDropReady(uid uint, col string, cooldownSec int) bool {
	var last []int64
	h.DB.Model(&model.EzfyProfile{}).Where("user_id = ?", uid).Pluck(col, &last)
	if len(last) == 0 || last[0] <= 0 {
		return true // 从未掉过 / 列尚未补出（老库）→ 放行，避免误伤
	}
	return time.Now().UnixMilli()-last[0] >= int64(cooldownSec)*1000
}

// generalSkillList 解析军官池守将(general)的技能名列表（g.Skill 是 JSON 数组）
func generalSkillList(g *model.EzfyCfgGeneral) []string {
	out := []string{}
	if g == nil || g.Skill == "" {
		return out
	}
	_ = json.Unmarshal([]byte(g.Skill), &out)
	return out
}

// generalHasSkill 军官池守将是否带某个技能
func generalHasSkill(g *model.EzfyCfgGeneral, name string) bool {
	for _, s := range generalSkillList(g) {
		if s == name {
			return true
		}
	}
	return false
}

// ★ 2026-10-06 军官池守将技能随等级自动升级（与玩家军官同口径，见 ezfy_officer.go）：
//
//	每 30 级 +1 级；名将(kind=2)最高 6 级，普通军官(kind=1)最高 5 级。
func generalSkillLevel(g *model.EzfyCfgGeneral) int {
	if g == nil {
		return 1
	}
	cap := 5
	if g.Kind == 2 {
		cap = 6
	}
	return ezfySkillLevelOf(g.Level, cap)
}
func generalSkillScale(g *model.EzfyCfgGeneral) int { return generalSkillLevel(g) }
func generalCounterRounds(g *model.EzfyCfgGeneral) int {
	if g != nil && generalHasSkill(g, "绝地逆袭") {
		return generalSkillLevel(g)
	}
	return 0
}

// generalSkillDefBonus 军官池守将「防御类技能的攻击加成拆解段」（弧形壁垒 30，随技能等级 ×N）。
// 活动守军无城墙/无科技 → 守方攻击加成整体来自守将（defBonus），
// 这里取下其中的技能段，供战报拆解单独展示「军官技能+N%」。
//
// ★ 2026-10-31 弹幕覆盖移出：它按文案是「城防攻击范围+10%」，不是防御加成
//   （原来错算 +10% 防御且全兵种生效，用户反馈「兵种也生效了」）。
func generalSkillDefBonus(g *model.EzfyCfgGeneral) int {
	if g == nil {
		return 0
	}
	scale := generalSkillScale(g)
	bonus := 0
	for _, s := range generalSkillList(g) {
		switch s {
		case "弧形壁垒":
			bonus += 30 * scale
		}
	}
	return bonus
}

// generalSkillDefBreak 军官池守将「防御类技能」的**逐项**明细（战报展示「军官技能·弧形壁垒+N%」）。
// 与 generalSkillDefBonus 同一口径，明细之和 = 技能总加成。
func generalSkillDefBreak(g *model.EzfyCfgGeneral) []ezfyBonusItem {
	if g == nil {
		return nil
	}
	scale := generalSkillScale(g)
	out := []ezfyBonusItem{}
	for _, s := range generalSkillList(g) {
		switch s {
		case "弧形壁垒":
			out = append(out, ezfyBonusItem{Name: s, Value: 30 * scale})
		}
	}
	return out
}

// generalTypeBonus 军官池守将的**兵种专属**技能加成（与 officerTypeBonus 同一口径）。
//
// ★ 2026-10-31 新增：活动守军/野地守将原来没有兵种专属通道 ——
// 坦克冲锋（只轻坦/重坦**攻速**）、狼群围猎（海军打海军）、弹幕覆盖（城防射程%）
// 在守将身上全被丢掉。现在按与玩家军官完全相同的规则下发。
func generalTypeBonus(g *model.EzfyCfgGeneral, b ezfyTypeBonus) ezfyTypeBonus {
	if b.Atk == nil {
		b.Atk = map[int]int{}
	}
	if b.Speed == nil {
		b.Speed = map[int]int{}
	}
	if b.AtkVsType == nil {
		b.AtkVsType = map[int]map[int]int{}
	}
	if b.AtkFlat == nil {
		b.AtkFlat = map[int]int{}
	}
	if b.AtkFlatTroop == nil {
		b.AtkFlatTroop = map[int]int{}
	}
	if b.SpeedTroop == nil {
		b.SpeedTroop = map[int]int{}
	}
	if b.AtkSpeed == nil {
		b.AtkSpeed = map[int]int{}
	}
	if b.AtkSpeedTroop == nil {
		b.AtkSpeedTroop = map[int]int{}
	}
	if b.Range == nil {
		b.Range = map[int]int{}
	}
	if g == nil {
		return b
	}
	scale := generalSkillScale(g)
	for _, s := range generalSkillList(g) {
		switch s {
		case "火炮压制":
			for _, id := range ezfyArmorTroopIds {
				b.AtkFlatTroop[id] += 10 * scale
			}
		case "四指阵列":
			b.addVsType(ezfyTroopTypeAir, ezfyTroopTypeAir, 15*scale)
		case "狼群围猎": // ★★ 2026-10-31 最终口径：海军打海军（目标也必须是海军兵种）
			b.addVsType(ezfyTroopTypeNavy, ezfyTroopTypeNavy, 15*scale)
		case "坦克冲锋": // ★ 2026-10-31 改为**攻速**（只轻坦/重坦），与玩家军官同口径
			for _, id := range ezfyTankRushTroopIds {
				b.AtkSpeedTroop[id] += 10 * scale
			}
		case "闪电疾攻": // ★ 2026-10-31 改为**空军攻速**
			b.AtkSpeed[ezfyTroopTypeAir] += 10 * scale
		case "越岛登陆": // ★ 2026-10-31 改为**海军攻速**
			b.AtkSpeed[ezfyTroopTypeNavy] += 10 * scale
		case "弹幕覆盖":
			b.Range[ezfyTroopTypeCity] += 10 * scale
		}
	}
	return b
}

// ezfyActWildDefBonus 活动野地配置了守将时的守方**防御**加成（复刻玩家城「城守」口径）。
//
// ★ 2026-09-29 修复：活动野地原来「守方加成恒为 0」，导致配了守将也看不出守方厉害
//
//	（战斗加成里 守方 防御+0% 速度+0%）。现在按守将有效学识给防御。
//	无科技/城墙/装备：防御 = (有效学识+1)/2 + 弧形壁垒+30（每3回合发动1回合）。
//
// ★ 2026-10-06 守将技能随等级自动升级：弧形壁垒 ×技能倍率（属性部分 ezfyAttrToBonus 不加倍）。
//
// ★★ 2026-10-31 两处口径修正（用户要求）：
//
//	① 弹幕覆盖不再算防御 —— 它是「城防攻击范围+10%」，改走 generalTypeBonus 的射程档；
//	② 速度技能（坦克冲锋/闪电疾攻/越岛登陆）不再返回「全兵种 +10」标量，
//	   改由 generalTypeBonus 按兵种下发（坦克冲锋只轻坦/重坦）→ 第二个返回值恒为 0，
//	   保留签名只为不动调用点（真正速度走 defType）。
//
// 返回 (防御加成, 速度加成)。
func ezfyActWildDefBonus(g *model.EzfyCfgGeneral) (int, int) {
	if g == nil {
		return 0, 0
	}
	scale := generalSkillScale(g)
	def := 0
	for _, s := range generalSkillList(g) {
		switch s {
		case "弧形壁垒":
			def += 30 * scale
		}
	}
	def += ezfyAttrToBonus(g.Learning)
	return def, 0
}

// 二战风云 活动目标：活动野地 / 活动寇城 / 特殊城市
//
// 复刻 GameServiceImpl：
//   isActivityWildland / isActivityKouCity / isSpecialCity / getActivityLevel
//   buildActivityDefender / processActivityBattle
// 以及 MapController 里给格子打的 actWild / actKou / actCity 标记。
//
// 与普通野地/寇城的区别：
//   1. 位置由坐标哈希固定生成（不是随机刷、不落库），全服一致；
//   2. 守军按「活动等级 1~3」成倍增长，最强是特殊城市（100万/200万/500万）；
//   3. 打赢只结算奖励（资源/黄金/必掉宝物/大量声望），**不占领**，也不占用附属野地上限；
//   4. 战报统一是「战斗报告」，标题形如「特殊城市3级战斗报告: 特殊城市」。

// 活动目标类型
const (
	ezfyActNone = 0 // 非活动目标
	ezfyActWild = 1 // 活动野地
	ezfyActKou  = 2 // 活动寇城
	ezfyActCity = 3 // 特殊城市
)

// ============ 坐标哈希判定（复刻 GameServiceImpl） ============

// ezfyActivityWildland 活动野地：哈希命中 499 且距世界中心 > 100
func ezfyActivityWildland(x, y int) bool {
	if ezfyAbs(x*1729^y*3803)%499 != 0 {
		return false
	}
	return ezfyAbs(x-250)+ezfyAbs(y-250) > 100
}

// ezfyActivityKouCity 活动寇城：哈希命中 1499 且距世界中心 > 120
func ezfyActivityKouCity(x, y int) bool {
	if ezfyAbs(x*9157^y*2657)%1499 != 0 {
		return false
	}
	return ezfyAbs(x-250)+ezfyAbs(y-250) > 120
}

// ezfySpecialCity 特殊城市：哈希命中 2999 且距世界中心 > 150
func ezfySpecialCity(x, y int) bool {
	if ezfyAbs(x*6689^y*5347)%2999 != 0 {
		return false
	}
	return ezfyAbs(x-250)+ezfyAbs(y-250) > 150
}

// ezfyActivityLevel 活动目标等级 1~3（复刻 getActivityLevel）
func ezfyActivityLevel(x, y int) int {
	return ezfyAbs(x*817^y*1013)%3 + 1
}

// ezfyActTargetType 判定某格的「活动目标」类型（0=无 1=活动野地 2=活动寇城 3=特殊城市）
// 复刻 MapController 的三条互斥规则：
//
//	actKou  = isKou      && isActivityKouCity
//	actWild = !isKou                 && isActivityWildland
//	actCity = !isKou && !actWild     && isSpecialCity
func (h *EzfyHandler) ezfyActTargetType(x, y int) int {
	// ★ 2026-09-29 活动野地「按坐标配置」优先：某格有 ezfy_act_wild 记录时，
	//   enabled=1 → 强制活动野地；enabled=0 → 强制**不是**活动目标（区别于普通野地）。
	//   无记录才回落 mark 覆盖 / 坐标哈希判定。
	if aw := ezfyActWildAt(x, y); aw != nil {
		if aw.Enabled == 1 {
			return ezfyActWild
		}
		return ezfyActNone
	}
	// ★ 管理端「地图格子覆盖」优先：某格被显式标成活动目标就用它，
	//   否则照旧按坐标哈希推导。
	switch ezfyMarkKindAt(x, y) {
	case model.EzfyMarkActKou:
		return ezfyActKou
	case model.EzfyMarkActWild:
		return ezfyActWild
	case model.EzfyMarkActCity:
		return ezfyActCity
	case model.EzfyMarkKou:
		return ezfyActNone // 普通寇城不算活动目标
	}
	return ezfyActTypeFor(x, y, h.ezfyIsKouCity(x, y))
}

// ezfyActTypeFor 纯函数版本（寇城判定由调用方传入，避免地图逐格重复查库）
func ezfyActTypeFor(x, y int, kou bool) int {
	if kou {
		if ezfyActivityKouCity(x, y) {
			return ezfyActKou
		}
		return ezfyActNone
	}
	if ezfyActivityWildland(x, y) {
		return ezfyActWild
	}
	if ezfySpecialCity(x, y) {
		return ezfyActCity
	}
	return ezfyActNone
}

// ezfyActTargetName 活动目标中文名
func ezfyActTargetName(actType int) string {
	switch actType {
	case ezfyActWild:
		return "活动野地"
	case ezfyActKou:
		return "活动寇城"
	case ezfyActCity:
		return "特殊城市"
	}
	return ""
}

// ezfyActTargetDesc 活动目标详情页说明文案（照 activityIndex.html）
func ezfyActTargetDesc(actType int) string {
	switch actType {
	case ezfyActWild:
		return "陆/海随机刷新, 10万~30万守军, 胜利获得大量资源+黄金(元宝)+必定掉落宝物+大量声望"
	case ezfyActKou:
		return "20万~60万守军, 胜利获得巨大资源+黄金+宝物+声望"
	case ezfyActCity:
		return "100万~500万守军, 全服最强活动目标, 需要强力的部队! 胜利必定获得高级/特殊宝物, 巨额黄金与资源"
	}
	return ""
}

// ezfyActTargetLabel 「活动野地2级」这样的完整标签
func ezfyActTargetLabel(actType, level int) string {
	return ezfyActTargetName(actType) + strconv.Itoa(level) + "级"
}

// ============ 活动目标每日攻打上限（★ 2026-10-30）============
//
// 用户规则：每个城市每天最多打 50 次活动目标（管理端可调）；超过后该活动目标
// 对这座城按普通野地展示/结算，避免脚本反复刷活动资源。
// 口径：按**出发城市** + **自然日**统计掠夺(2)/征服(3)订单；侦查(1)不占次数。

// ezfyActDailyLimit 每座城市每天的活动目标攻击上限（默认 50，0 = 不限）。
// ★ 0 是合法值（= 不限），不能用 ezfyLimitOr（它把 0 当未配置回落默认）。
func ezfyActDailyLimit() int {
	if !ezfyCfg.ready() {
		return ezfyActDailyLimitDef
	}
	if n := ezfyCfg.limit.ActDailyLimit; n >= 0 {
		return n
	}
	return ezfyActDailyLimitDef
}

// ezfyActDailyRange 返回 now 所在自然日的 [起, 止) 毫秒时间戳（服务器时区）。
func ezfyActDailyRange(now int64) (int64, int64) {
	start := ezfyDayStart(now)
	end := time.UnixMilli(start).In(time.Local).AddDate(0, 0, 1).UnixMilli()
	return start, end
}

// ezfyActDailyAllow 第 rank 次（从 1 开始）是否还在当日配额内。
func ezfyActDailyAllow(limit int, rank int64) bool {
	return limit <= 0 || rank <= int64(limit)
}

// ezfyActDailyUsed 某城今日已创建的活动目标攻击订单数。
//
// ★★ 2026-10-30 名将活动目标（配了守将的）**不计入**配额 —— 它们既不占次数，
// 也不受上限约束（见 ezfyActNamedTargetAt）。否则打名将会把普通活动目标的次数挤掉。
// 订单量很小（每城每天上限几十条），取坐标后在内存里过滤，不额外打库。
func (h *EzfyHandler) ezfyActDailyUsed(cityID int64) int64 {
	start, end := ezfyActDailyRange(time.Now().UnixMilli())
	var rows []model.EzfyOrder
	h.DB.Model(&model.EzfyOrder{}).Select("target_x", "target_y").
		Where("city_id = ? AND act_type > 0 AND order_type IN (?, ?) AND start_time >= ? AND start_time < ?",
			cityID, 2, 3, start, end).
		Find(&rows)
	n := int64(0)
	for i := range rows {
		if h.ezfyActNamedTargetAt(rows[i].TargetX, rows[i].TargetY) {
			continue
		}
		n++
	}
	return n
}

// ezfyActNamedTargetAt 该坐标是否为「名将活动目标」—— 活动目标且守将**是名将**（general.Kind=2）。
//
// 守将来源与 ezfyActDefConfigOf 同口径（坐标级 aw 优先，其次活动目标等级配置）；
// ★ 只认**名将**（kind=2）：配普通军官（kind=1）的活动野地仍按普通活动目标受每日上限约束。
//
// ★★ 2026-10-30 用户规则：**名将活动目标不卡「活动目标每日上限(次/城)」**。
// 名将只受原有「可抓次数 max_capture」约束 —— 抓满后该坐标对该玩家变普通地形
// （playerOwnsActWildGeneral）。所以名将订单既不计入当日配额，也不被配额拦下。
func (h *EzfyHandler) ezfyActNamedTargetAt(x, y int) bool {
	act := h.ezfyActTargetType(x, y)
	if act <= 0 {
		return false
	}
	aw := ezfyActWildAt(x, y)
	lv := ezfyMarkLevelAt(x, y)
	if aw != nil && aw.Enabled == 1 && aw.Level >= 1 && aw.Level <= 3 {
		lv = aw.Level
	}
	oid := ezfyActDefConfigOf(aw, act, lv).OfficerId
	return oid > 0 && ezfyCfg.isGeneral(oid) // isGeneral = 唯一实现（kind==2 名将，普通军官不算）
}

// ezfyActDailyLimitReached 该城今日活动目标攻击次数是否已用完。
func (h *EzfyHandler) ezfyActDailyLimitReached(cityID int64) bool {
	limit := ezfyActDailyLimit()
	if limit <= 0 {
		return false
	}
	return h.ezfyActDailyUsed(cityID) >= int64(limit)
}

// ezfyActDailyOrderRank 该订单是出发城市当日第几支活动攻击队（含自己，从 1 开始）。
// 用订单 id 排序做「到达时二次卡控」，防止并发下单同时穿过创建时的读校验。
// ★ 名将活动目标的订单不计入排名（它们不占配额）。
func (h *EzfyHandler) ezfyActDailyOrderRank(order *model.EzfyOrder) int64 {
	if order == nil {
		return 0
	}
	start, end := ezfyActDailyRange(order.StartTime)
	var rows []model.EzfyOrder
	h.DB.Model(&model.EzfyOrder{}).Select("target_x", "target_y").
		Where("city_id = ? AND act_type > 0 AND order_type IN (?, ?) AND start_time >= ? AND start_time < ? AND id <= ?",
			order.CityId, 2, 3, start, end, order.ID).
		Order("id").
		Find(&rows)
	n := int64(0)
	for i := range rows {
		if h.ezfyActNamedTargetAt(rows[i].TargetX, rows[i].TargetY) {
			continue
		}
		n++
	}
	return n
}

// ezfyActOrderWithinDailyLimit 该订单到达时是否仍可用活动目标结算。
// ★ 存量订单（ActType = -1，补列前创建）不做配额追溯，保持原有在途战斗行为。
// ★★ 名将活动目标（配了守将）不受每日上限约束，直接放行（只受「可抓次数」约束）。
func (h *EzfyHandler) ezfyActOrderWithinDailyLimit(order *model.EzfyOrder) bool {
	if order == nil || order.ActType < 0 {
		return true
	}
	if h.ezfyActNamedTargetAt(order.TargetX, order.TargetY) {
		return true
	}
	limit := ezfyActDailyLimit()
	if limit <= 0 {
		return true
	}
	return ezfyActDailyAllow(limit, h.ezfyActDailyOrderRank(order))
}

// ezfyOrderActType 订单实际走哪类活动目标结算/展示。
// 新订单以 ActType 为准（0 = 当日次数用完后按普通野地处理）；
// 补列前的存量订单（-1）才按坐标兜底推断，避免部署瞬间让在途活动单变成普通野地。
func (h *EzfyHandler) ezfyOrderActType(uid uint, order *model.EzfyOrder) int {
	if order == nil {
		return 0
	}
	if order.ActType >= 0 {
		return order.ActType
	}
	if order.OrderType != 1 && order.OrderType != 2 && order.OrderType != 3 {
		return 0
	}
	act := h.ezfyActTargetType(order.TargetX, order.TargetY)
	if act > 0 && !h.playerOwnsActWildGeneral(uid, order.TargetX, order.TargetY) {
		return act
	}
	return 0
}

// playerOwnsActWildGeneral 该玩家是否已拥有此坐标活动野地的守将（名将野地按玩家判定）。
//
// ★ 2026-10-05 用户规则：名将野地对没抓到守将的玩家仍是名将野地；只有**已抓到该守将的玩家**，
//
//	该坐标才是普通野地。判定口径 = 该玩家名下任一城市的军官里存在 general_id == 守将ID
//	（收编后 IsCaptive=0 也算已拥有）。快速路径：非活动野地/没配守将 → 直接 false，零查询。
//
// ★★ 2026-10-30 用户规则「名将只卡原来的 —— 抓到了最大次数才会变成普通地形」：
//
//	守将是**名将**且「可抓次数 max_capture > 0」时，改按**已抓次数是否抓满**判定
//	（ezfy_act_capture_log 只增流水）。原来「拥有即变」会让 max_capture=3 时
//	抓到第 1 只坐标就变普通地形，剩下的 2 次永远用不掉。
//	· max_capture <= 0（不限）→ 仍按「拥有即变」（否则可以无限刷名将野地资源）；
//	· 普通军官守将 → 仍按「拥有即变」（本次只收口名将）。
func (h *EzfyHandler) playerOwnsActWildGeneral(uid uint, x, y int) bool {
	aw := ezfyActWildAt(x, y)
	if aw == nil || aw.Enabled != 1 || aw.OfficerId <= 0 {
		return false
	}
	g := ezfyCfg.general(aw.OfficerId)
	if g == nil {
		return false
	}
	if g.Kind == 2 && aw.MaxCapture > 0 {
		// ★ 只读流水计数（不能调 ezfyActCaptureUsed：它在无流水时会**补记流水**，
		//   而这个函数在地图渲染里逐格调用，绝不能写库）。
		var used int64
		h.DB.Model(&model.EzfyActCaptureLog{}).
			Where("user_id = ? AND general_id = ?", uid, g.ID).
			Count(&used)
		if used > 0 {
			return ezfyActCaptureFull(aw.MaxCapture, used)
		}
		// 没有抓取流水（老数据 / 非野地来源）→ 落到下面「拥有即变」兜底，避免凭空多出机会
	}
	var n int64
	h.DB.Model(&model.EzfyOfficer{}).
		Where("general_id = ? AND city_id IN (?)", g.ID,
			h.DB.Model(&model.EzfyCity{}).Select("id").Where("user_id = ?", uid)).
		Count(&n)
	return n > 0
}

// ============ 活动守军（复刻 buildActivityDefender） ============

// ezfyActivityDefender 活动目标守军
// 陆地：装甲车 50% / 轻型坦克 30% / 重型坦克 20%
// 海洋：驱逐舰 40% / 潜艇 30% / 战列舰 20% / 航母 10%
func ezfyActivityDefender(actType, level, terrain int) []ezfyUnitGroup {
	// ★ 2026-10-22 管理端「活动目标配置」：配了守军就优先用配置（范围随机），空则走内置默认。
	if cfg := ezfyActLevelCfg(actType, level); cfg != nil && strings.TrimSpace(cfg.Troops) != "" {
		if groups := parseActWildTroops(cfg.Troops); len(groups) > 0 {
			return groups
		}
	}
	var base int64
	switch actType {
	case ezfyActKou:
		base = 200000 // 活动寇 20万/40万/60万
	case ezfyActCity:
		base = 1000000 // 特殊城市 100万/200万/500万
	default:
		base = 100000 // 活动野地 10万/20万/30万
	}
	total := base * int64(level)
	if actType == ezfyActCity && level == 3 {
		total = base * 5
	}
	ratio := [][2]int{{4, 50}, {5, 30}, {6, 20}} // 装甲车 / 轻型坦克 / 重型坦克
	// ★ 2026-10-05 岛屿也属于海野 → 活动目标在岛屿/海底森林上守军按海军配比
	if ezfyIsSeaWildTerrain(terrain) {
		ratio = [][2]int{{13, 40}, {14, 30}, {15, 20}, {16, 10}} // 驱逐舰 / 潜艇 / 战列舰 / 航母
	}
	out := []ezfyUnitGroup{}
	for _, r := range ratio {
		out = append(out, ezfyUnitGroup{TroopId: r[0], Count: total * int64(r[1]) / 100})
	}
	return out
}

// ============ 活动战斗结算（复刻 processActivityBattle） ============

// ezfyActDefConfig 活动目标「守将 / 守军微调 / 被俘」配置：
// 坐标级 aw（enabled=1）优先，否则回退「活动目标配置」按 (ActType, Level)。
//
// ★ 2026-10-24 活动目标等级配置（al）也支持与活动野地同款的守将配置；
//   个别坐标想特调 → 在活动野地 tab 配 aw，启用后优先于本表。
type ezfyActDefConfig struct {
	OfficerId      int    // 守将军官（军官池 ID，0 = 不设）
	DefLevel       int    // 守将战斗等级（0 = 用军官池默认）
	DefMoveDir     string // 守军行为 JSON map {兵种ID:1前进/0停止/-1后退}
	CaptureRate    int    // 被俘虏概率%（0 = 不俘虏）
	MaxCapture     int    // 同一玩家可抓次数（默认 1，0 = 不限）
	MinCaptureWins int    // ★ 2026-10-25 兜底被俘次数：至少打赢多少次才可能被俘（0 = 第一次就按概率）
	CaptiveLevel   int    // 被俘虏默认等级（0 = 用军官池默认）
}

func ezfyActDefConfigOf(aw *model.EzfyActWild, actType, level int) ezfyActDefConfig {
	if aw != nil && aw.Enabled == 1 {
		return ezfyActDefConfig{
			OfficerId:      aw.OfficerId,
			DefLevel:       aw.DefLevel,
			DefMoveDir:     aw.DefMoveDir,
			CaptureRate:    aw.CaptureRate,
			MaxCapture:     aw.MaxCapture,
			MinCaptureWins: aw.MinCaptureWins,
			CaptiveLevel:   aw.CaptiveLevel,
		}
	}
	if cfg := ezfyActLevelCfg(actType, level); cfg != nil {
		return ezfyActDefConfig{
			OfficerId:      cfg.OfficerId,
			DefLevel:       cfg.DefLevel,
			DefMoveDir:     cfg.DefMoveDir,
			CaptureRate:    cfg.CaptureRate,
			MaxCapture:     cfg.MaxCapture,
			MinCaptureWins: cfg.MinCaptureWins,
			CaptiveLevel:   cfg.CaptiveLevel,
		}
	}
	return ezfyActDefConfig{MaxCapture: 1}
}

// ezfyActCaptureAllowed 兜底被俘次数是否已达标（★ 唯一实现，便于测试）。
//
//	minWins = 管理端配置的「兜底被俘次数」（0 = 不设门槛，第一次打赢就按概率）；
//	wins    = 该玩家已打赢该守将的次数。
//
//	达标线只在服务端判定，**不写进战报、不下发用户端**（用户要求「不要展示到用户端」）。
func ezfyActCaptureAllowed(minWins int, wins int64) bool {
	return minWins <= 0 || wins >= int64(minWins)
}

// ezfyActCaptureWinAdd 记一次「打赢该守将」，返回该玩家累计胜场数（★ 兜底被俘次数进度）。
//
// ★ 2026-10-25：口径 = **打赢次数**（战败/平局不算）—— 俘虏只在胜利时判定，
//   按胜场计才不会被「1 兵自杀式碰瓷刷次数」钻空子。
//   进度按 (玩家, 守将) 存 ezfy_act_capture_try，坐标级 aw 与等级配置共用同一份。
func (h *EzfyHandler) ezfyActCaptureWinAdd(uid uint, generalId int) int64 {
	if uid == 0 || generalId <= 0 {
		return 0
	}
	h.DB.Exec(`INSERT INTO ezfy_act_capture_try (user_id, general_id, wins, updated_at)
VALUES (?, ?, 1, NOW())
ON DUPLICATE KEY UPDATE wins = wins + 1, updated_at = VALUES(updated_at)`, uid, generalId)
	var n int64
	// ★ 与 ezfy_pvp_rank.go 同款写法：Select(单列).Scan(&标量) 取一行一列（Pluck 只吃 slice）
	h.DB.Model(&model.EzfyActCaptureTry{}).
		Where("user_id = ? AND general_id = ?", uid, generalId).
		Select("wins").Scan(&n)
	return n
}

// ezfyActCaptureUsed 该玩家对某守将的「已抓取次数」= 抓取流水条数（★ 2026-10-28 用户规则）。
//
// ★★ 口径变更（用户反馈）：「可抓次数配 1 次」是**针对玩家**的 —— 他抓住过一次，
//
//	就再也不能抓了；即使他抓到的那个守将后来被其他玩家抢走，也**不能**再抓。
//	配置 N 次同理：抓 N 次（不管现在手里还剩几个）就封顶。
//
// ★ 旧实现为什么不对：原来数的是「该玩家当前拥有的该守将军官实例数」，
//
//	军官被抢走（PvP 俘虏/策反）后实例数归零 → 抓过的人又能再抓一次。
//	现在数 ezfy_act_capture_log 的流水条数，**只增不减**（流水只插不删）。
//
// ★ 老数据兜底：流水表刚建时是空的（历史抓取没记过），此时按「当前拥有的
//
//	军官实例数」补记同样条数的 kind=2 流水，避免老玩家凭空多出一次机会；
//	补记之后即使军官被抢走，流水仍在 → 不会再回退（这正是本次要修的点）。
func (h *EzfyHandler) ezfyActCaptureUsed(uid uint, generalId int) int64 {
	if uid == 0 || generalId <= 0 {
		return 0
	}
	var used int64
	h.DB.Model(&model.EzfyActCaptureLog{}).
		Where("user_id = ? AND general_id = ?", uid, generalId).
		Count(&used)
	if used > 0 {
		return used
	}
	// —— 老数据兜底：只在该玩家对该守将「一条流水都没有」时补记 ——
	var had int64
	// ★★ 2026-10-28 用户反馈「抓取错了吧，我的意思是抓住的记录呀」后收紧口径：
	//   只认**野地俘虏**（source=3）与历史未标注（source=0）；
	//   **PvP 抢来的（4）/ 管理端发放（1）/ 军校招募（2）都不算「抓过」** ——
	//   抢来的守将不是自己抓的，不该消耗「可抓次数」。
	h.DB.Model(&model.EzfyOfficer{}).
		Where("general_id = ? AND source IN ? AND city_id IN (?)", generalId,
			[]int{model.EzfyOfficerSourceUnknown, model.EzfyOfficerSourceWildland},
			h.DB.Model(&model.EzfyCity{}).Select("id").Where("user_id = ?", uid)).
		Count(&had)
	for i := int64(0); i < had; i++ {
		h.ezfyActCaptureLogAdd(uid, generalId, 0, 0, 0, model.EzfyActCaptureKindOfficer)
	}
	return had
}

// ezfyActCaptureLogAdd 记一条抓取流水（★ 只增不删，这就是「可抓次数」的扣减）。
//
// kind：model.EzfyActCaptureKindBattle（战斗抓到）/ KindReport（战报补记）/ KindOfficer（军官补记）。
func (h *EzfyHandler) ezfyActCaptureLogAdd(uid uint, generalId, x, y int, cityId int64, kind int) {
	if uid == 0 || generalId <= 0 {
		return
	}
	if kind <= 0 {
		kind = model.EzfyActCaptureKindBattle
	}
	// ★ 写入必须查 error：这张表要是没建出来（老库没跑建表），静默失败会让「可抓次数」
	//   永远停在 0 → 玩家可以无限抓。宁可留日志也别无声无息。
	if err := h.DB.Create(&model.EzfyActCaptureLog{
		UserId: uid, GeneralId: generalId, X: x, Y: y, CityId: cityId, Kind: kind,
	}).Error; err != nil {
		log.Printf("【严重】活动守将抓取流水写入失败 user=%d general=%d: %v", uid, generalId, err)
	}
}

// ezfyActCaptureFull 是否已达「同一玩家可抓次数」上限（maxCapture <= 0 = 不限）。
//
// ★ 唯一判定实现，便于测试；调用方不要再自己写 `used >= max`。
func ezfyActCaptureFull(maxCapture int, used int64) bool {
	return maxCapture > 0 && used >= int64(maxCapture)
}

// processActivityBattle 活动目标战斗结算（独立于普通野地/寇城流程，打赢不占领）
func (h *EzfyHandler) processActivityBattle(uid uint, city *model.EzfyCity, order *model.EzfyOrder,
	now int64, actType int) {

	level := ezfyActivityLevel(order.TargetX, order.TargetY)
	terrain := ezfyTerrain(order.TargetX, order.TargetY)
	label := ezfyActTargetLabel(actType, level)

	// ★ 2026-09-29 活动野地「按坐标配置」：等级/守军/奖励 优先用 ezfy_act_wild 配置，缺省回退默认
	aw := ezfyActWildAt(order.TargetX, order.TargetY)
	if aw != nil && aw.Enabled == 1 {
		if aw.Level >= 1 && aw.Level <= 3 {
			level = aw.Level
			label = ezfyActTargetLabel(actType, level)
		}
	}

	defender := ezfyActivityDefender(actType, level, terrain)
	// 配置里的守军优先（JSON [[兵种id,count],...]）
	if aw != nil && aw.Enabled == 1 && strings.TrimSpace(aw.Troops) != "" {
		if cfg := parseActWildTroops(aw.Troops); cfg != nil && len(cfg) > 0 {
			defender = cfg
		}
	}
	attacker := parseGroups(order.Troops)

	// 攻方加成与普通出征完全一致（军官 + 科技 + 技能）
	atkTech := h.techMap(city.ID)
	// ★ 2026-10-24 优先按命令里的军官 id 精确取（同名军官互不误伤）；老数据无 id 按名字兜底
	leadOfficer := h.officerByName(city.ID, order.Officer)
	if order.OfficerId > 0 {
		if o := h.officerOf(city.ID, int64(order.OfficerId)); o != nil {
			leadOfficer = o
		}
	}
	// ★ 2026-10-06 增程弹药(8) 改为**射程加成**（：增程弹药=射程，不参与攻击加成）
	// ★ 2026-10-21 动员训练(5) 改为造兵速度科技，不再参与战斗攻击加成。
	atkBonus := h.officerBattleBonus(leadOfficer) +
		atkTech[6]*3 + atkTech[9]*2
	// 速度：机械增压(10)「部队速度」= 通用；喷气推进(19)「空军速度」= 兵种专属（见下方 atkType）
	// ★ 2026-10-31 满级 50%：读配置（不再写死每级 2%）
	atkSpeedBonus := ezfyTechEffectValueAt(10, atkTech[10])
	// ★ 2026-10-08 兵种专属加成（与普通出征同口径）：喷气推进只加空军 + 军官兵种技能
	atkType := ezfyTypeBonus{Speed: map[int]int{}}
	if v := ezfyTechEffectValueAt(19, atkTech[19]); v > 0 {
		atkType.Speed[ezfyTroopTypeAir] += v
	}
	atkType = h.officerTypeBonus(leadOfficer, atkType)
	// ★ 2026-10-08 与普通出征同口径：军官行同时展示攻击加成与防御加成（属性部分，技能单列）
	atkOfficerDesc := h.officerBattleDesc(leadOfficer, h.officerBaseBonus(leadOfficer), "攻击加成",
		h.officerGuardAttrBonus(leadOfficer), "防御加成")
	// ★ 2026-10-06 战报拆解逐项明细：攻方科技/技能逐项（与普通出征同一口径）
	// ★ 2026-10-08 补全影响 攻击/防御/射程/速度 的全部攻方科技（与普通出征同口径）
	atkTechs := ezfyBonusItems(
		ezfyTechItem("枪械改良", atkTech[6]*3),
		ezfyTechItem("均质装甲", atkTech[7]*3),
		ezfyTechItem("增程弹药", atkTech[8]*3),
		ezfyTechItem("重型机床", atkTech[9]*2),
		ezfyTechItem("机械增压", ezfyTechEffectValueAt(10, atkTech[10])),
		ezfyTechItem("永备工事", atkTech[16]*2),
		ezfyTechItem("喷气推进", ezfyTechEffectValueAt(19, atkTech[19])),
	)
	atkSkillBreak := h.officerSkillsBreak(leadOfficer)
	// ★ 2026-10-07 攻方「防御加成」（出征军官属性+防御技能(弧形壁垒/弹幕覆盖)+装备 Def，
	//   与普通出征同口径；活动守军无装备 → 守方无此段）
	atkEquip := h.officerBattleEquipBonus(leadOfficer)
	atkDefBonus := 0
	var atkDefBreak []ezfyBonusItem
	if leadOfficer != nil {
		attr := h.officerGuardAttrBonus(leadOfficer)
		atkDefBonus += attr
		atkDefBreak = append(atkDefBreak, ezfyBonusItem{Name: "军官·" + leadOfficer.Name, Value: attr})
		for _, s := range h.officerGuardSkillsBreak(leadOfficer) {
			atkDefBonus += s.Value
			atkDefBreak = append(atkDefBreak, ezfyBonusItem{Name: "军官技能·" + s.Name, Value: s.Value})
		}
	}
	atkDefBonus += atkEquip.Def
	atkDefBreak = append(atkDefBreak, ezfyTechItem("装备套装", atkEquip.Def)...)

	// ★★ 指挥室（2026-09-22 ）：活动目标也是战斗，同样先开战场等玩家指挥，
	//   与普通野地/寇城/玩家城保持一致（否则打活动城不能指挥，玩家会困惑）。
	//   BattleResult 非空 = 已在指挥室里打完，直接用结果结算。
	// ★ 守方阵营: 活动野地=盟军、活动寇/特殊城市=轴心国(战报兵种名按阵营显示)
	defCamp := 1
	if actType == ezfyActKou || actType == ezfyActCity {
		defCamp = 2
	}
	// ★ 2026-09-29 修复「守方军官加成恒为 0」：活动目标配置了守将时，把守将的
	//   防御加成 / 速度加成 / 战报描述 接进战斗引擎（指挥室和战报都从这里生成）。
	// ★ 2026-10-24 守将配置 aw 启用优先，否则回退「活动目标配置」按 (ActType, Level)。
	defCfg := ezfyActDefConfigOf(aw, actType, level)
	var defGeneral *model.EzfyCfgGeneral
	if defCfg.OfficerId > 0 {
		defGeneral = ezfyCfg.general(defCfg.OfficerId)
	}
	// ★ 2026-10-24 「守军等级」：0 = 用军官池默认；>0 覆盖守将等级（浅拷贝，
	//   不污染 ezfyCfg 缓存里其他请求共用的对象）。等级影响守将技能 / 组合技等级。
	if defGeneral != nil && defCfg.DefLevel > 0 {
		g := *defGeneral
		g.Level = defCfg.DefLevel
		defGeneral = &g
	}
	defBonus, defSpeedBonus := ezfyActWildDefBonus(defGeneral)
	defOfficerDesc := ""
	if defGeneral != nil {
		// ★ 2026-10-08 与玩家城城守同口径：**同时**列出「攻击加成」与「守军防御」（属性部分），
		//   技能逐项单列。活动守军无城墙/无科技 → 守方的攻防加成整体都来自守将（defBonus），
		//   原来只写「守军防御」，守军那半攻击加成在军官行里看不到（用户反馈）。
		//   ★ 顺带补上「Lv.N」：原来活动守将行缺等级，且 ezfyOfficerShortName 只能靠第一个
		//     空格取名字（名字里带空格时会截错）。
		lv := generalSkillLevel(defGeneral)
		attr := ezfyAttrToBonus(defGeneral.Learning)
		parts := []string{}
		if attr > 0 {
			parts = append(parts, "攻击加成+"+strconv.Itoa(attr)+"%", "守军防御+"+strconv.Itoa(attr)+"%")
		}
		for _, s := range generalSkillList(defGeneral) {
			if eff := ezfySkillEffectTextAt(s, lv); eff != "" {
				parts = append(parts, s+"(Lv."+strconv.Itoa(lv)+" "+eff+")")
			}
		}
		defOfficerDesc = defGeneral.Name + " Lv." + strconv.Itoa(defGeneral.Level)
		if len(parts) > 0 {
			defOfficerDesc += " " + strings.Join(parts, " ")
		}
	}
	var br ezfyBattleResult
	// ★ 2026-10-06 守方「防御加成」逐项明细（活动守军无城墙/无科技 → 只有守将属性+技能，
	//   被打行展示「防御加成+N%(军官·名+X% 军官技能·弧形壁垒+Y% …)」，与 defBonus 构成同口径）
	defDefBreak := []ezfyBonusItem{}
	if defGeneral != nil {
		defDefBreak = append(defDefBreak, ezfyBonusItem{Name: "军官·" + defGeneral.Name, Value: ezfyAttrToBonus(defGeneral.Learning)})
		for _, s := range generalSkillDefBreak(defGeneral) {
			defDefBreak = append(defDefBreak, ezfyBonusItem{Name: "军官技能·" + s.Name, Value: s.Value})
		}
	}
	// ★ 2026-10-24 守军行为（**按兵种**配置）：JSON map {"兵种ID":1前进/0停止/-1后退}，
	//   直接转成引擎的 defMoves map[int]int；空/{} = 全部兵种默认前进（引擎缺省行为）。
	//   配置来源与守将同款：aw 启用优先，否则回退活动目标配置。
	defMoves := map[int]int{}
	if raw := strings.TrimSpace(defCfg.DefMoveDir); raw != "" && raw != "{}" {
		if json.Unmarshal([]byte(raw), &defMoves) != nil {
			defMoves = map[int]int{} // 脏数据兜底：当没配
		}
	}
	if done, ok := ezfyBattleResultDecode(order.BattleResult); ok {
		br = done
	} else {
		// 活动守军无城墙/无科技 → 只有守将的加成；攻方装备六项加成照常生效；
		// 守方阵营: 活动野地=盟军、活动寇/特殊城市=轴心国
		// ★ 2026-10-06 攻方射程加成 = 增程弹药(8)*3；活动守军无科技，射程/守方攻击加成取守将加成(0 兜底)
		st := ezfyNewBattleState(attacker, defender, atkBonus, defBonus, defBonus, atkSpeedBonus, defSpeedBonus,
			atkTech[8]*3, 0,
			atkEquip, ezfyBattleBonus{},
			h.officerSetEquipBonus(leadOfficer), ezfyBattleBonus{},
			h.officerSetsDesc(leadOfficer), "",
			atkOfficerDesc, defOfficerDesc,
			// 活动守军无城墙/无科技 → 守方攻击加成整体都来自守将；攻方军官加成照常拆解展示
			// ★ 2026-10-06 军官加成里「技能」占的百分点（拆解单独展示「军官技能+N%」）
			h.officerBattleBonus(leadOfficer), defBonus,
			h.officerSkillBattleBonus(leadOfficer), generalSkillDefBonus(defGeneral),
			// ★ 2026-10-06 技能/科技逐项明细：攻方=带队军官技能+科技；守方=守将技能（活动守军无科技 → nil）
			atkSkillBreak, generalSkillDefBreak(defGeneral), atkTechs, nil, defDefBreak,
			atkDefBonus, atkDefBreak,
			h.buildTargetMap(city.ID, true), map[int]int{},
			h.buildMoveMap(city.ID, true), defMoves,
			h.officerCounterRounds(leadOfficer),
			generalCounterRounds(defGeneral),
			h.ensureProfile(uid).Camp, defCamp,
			// ★ 2026-10-08 兵种专属加成（攻方喷气推进=空军速度+军官兵种技能）
			// ★★ 2026-10-31 守方（活动守将）也要下发兵种专属技能：狼群围猎=海军打海军 /
			//   弹幕覆盖=城防射程% / 坦克冲锋·闪电疾攻·越岛登陆=对应兵种速度。
			//   原来这里传空表 → 守将的兵种技能全被丢掉，且 ezfyActWildDefBonus 已不再
			//   返回「全兵种速度」标量（改按兵种下发）→ 不接这一档守将速度会变 0（回归）。
			atkType, generalTypeBonus(defGeneral, ezfyTypeBonus{}))
		// ★★ 2026-10-31 普通军官「每3回合发动1回合」技能（弧形壁垒 / 火炮压制）：
		//   与普通出征同口径 —— 快照存满额，引擎按回合扣减（见 ezfy_skill_timed.go）。
		st.AtkTimed = h.officerTimedSkills(leadOfficer)
		st.DefTimed = generalTimedSkills(defGeneral)
		// ★ 2026-10-23 名将组合技能：攻方带队名将 / 活动守将（池子配置，按星级定级）
		//   ★ 2026-10-24 多技能：名将可配多个，comboRuntimes 包成运行时数组，每个技能独立掷骰
		st.AtkCombos = comboRuntimes(h.comboSkillsOfOfficer(leadOfficer))
		if defGeneral != nil {
			st.DefCombos = comboRuntimes(ezfyComboOfGeneralCfg(defGeneral, defGeneral.Level))
		}
		st.appendComboHead()
		// ★★ 2026-10-07 「自动战斗」配置（出征页可配，见 EzfyOrder.AutoBattle）：
		//   活动目标（活动野地 / 活动寇城 / 特殊城市）**按「野地 / NPC」口径处理** ——
		//   守军是配置数据、没有真人，所以默认「是」：抵达即自动打完，无需指挥。
		//   · 玩家想指挥 → 在出征页把该订单改成「否」，抵达后照旧进指挥室。
		//   ★ 2026-10-08 移除了原「攻方不在线 → 自动结算」的覆盖：活动目标是 NPC，
		//     玩家在出征页明确选了「否」就要进指挥室，绝不因 180 秒在线窗口过期而
		//     偷偷自动打完（与打野地 / AI 寇城同口径；用户反馈「选了否也不进指挥」）。
		//   （原注释里的「同格后续部队全堵在等待(6)」防堵只对打真人守方的玩家城有意义。）
		autoBattle := order.AutoBattle == 1
		if autoBattle {
			br = h.ezfyBattleAutoFinish(uid, order, st, label, now)
		} else if ezfyOrderTargetBusy(h, order, int64(order.ID)) {
			// 「目标被抢先指挥 → 等待」：上一场打完(该订单不再战斗中)后 processOrders 自动放行重进
			order.Status = ezfyOrderStatusWaiting
			h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
				Update("status", ezfyOrderStatusWaiting)
			return
		} else if b := h.ezfyBattleStart(uid, order, st, label, now); b != nil {
			order.Status = ezfyOrderStatusBattle
			h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
				Update("status", ezfyOrderStatusBattle)
			return
		} else {
			// 开战场失败（极端情况）→ 兜底直接模拟，绝不让部队卡住
			for !st.Done {
				st.Step(nil, nil)
			}
			br = st.Result()
		}
	}
	win := br.AttackerWin
	draw := br.Draw

	// ★ 带队军官战功经验（2026-09-27 用户反馈「战败军官经验是 0」）：
	//   活动流程原来完全没给军官经验（打赢打输都是 0）。统一走普通战斗同款口径
	//   ezfyOfficerBattleExp —— 基础经验 = 击杀数/10 + 30，**即便战败、0 击杀也有 30 点**，
	//   胜利再多拿 50%。写在函数开头只算数值，战报正文与写库放在结果判定后统一处理。
	atkExp := int64(0)
	if leadOfficer != nil {
		atkExp = ezfyOfficerBattleExp(enemyDeadOf(br.DefenderLosses), win)
	}

	profile := h.ensureProfile(uid)
	// ★ 2026-10-21 战报 [攻方] 显示玩家昵称（活动目标守方仍是野地/目标名+等级）。
	atkLabel := strings.TrimSpace(profile.Nickname)
	if atkLabel == "" {
		atkLabel = city.Name
	}
	report := fmt.Sprintf("主题:战斗报告\n出发地:%s(%d,%d)\n目的地:%s(%d,%d)\n时间:%s\n公文报告:战斗报告\n我方一支部队对%s[ %d，%d ]发起了进攻。战斗共持续 %d 回合，我方战斗%s\n",
		city.Name, city.X, city.Y, label, order.TargetX, order.TargetY,
		time.UnixMilli(now).Format("2006-01-02 15:04"), label,
		order.TargetX, order.TargetY, br.Rounds, battleOutcomeText(win, draw))
	report += fmt.Sprintf("统帅声望:%d\n", profile.Prestige)
	if leadOfficer != nil {
		report += "军官:" + officerReportDesc(leadOfficer) + "\n"
	}

	atkBefore := groupCounts(attacker)
	atkAfter := groupCounts(br.AttackerLeft)
	// ★ 2026-09-24 40 回合平局时双方标签都显示 [平]
	atkTag, defTag := winText(win), winText(!win)
	if draw {
		atkTag, defTag = "平", "平"
	}
	report += fmt.Sprintf("[%s]攻方:%s\n", atkTag, atkLabel)
	report += troopChangeText(atkBefore, atkAfter)
	report += fmt.Sprintf("--------------------\n[%s]守方:%s\n", defTag, label)
	// ★ 2026-09-29 配置了守将时展示守方军官（否则守方只有兵种看不到是谁）
	if defGeneral != nil {
		report += "军官:" + defGeneral.Name + "\n"
	}
	defBefore := groupCounts(defender)
	defAfter := map[int]int64{}
	for _, g := range br.DefenderLosses {
		defAfter[g.TroopId] = maxInt64(0, defBefore[g.TroopId]-g.Count)
	}
	for tid, cnt := range defBefore {
		if _, ok := defAfter[tid]; !ok {
			defAfter[tid] = cnt
		}
	}
	report += troopChangeText(defBefore, defAfter)

	// 详细战报：逐回合过程 + 双方兵力变化
	detail := ""
	for _, a := range br.Actions {
		detail += a + "\n"
	}
	detail += "\n[双方兵力]\n"
	detail += fmt.Sprintf("[%s]攻方:%s\n", atkTag, atkLabel)
	if leadOfficer != nil {
		detail += "军官:" + officerReportDesc(leadOfficer) + "\n"
	}
	detail += troopChangeText(atkBefore, atkAfter)
	detail += fmt.Sprintf("--------------------\n[%s]守方:%s\n", defTag, label)
	if defGeneral != nil {
		detail += "军官:" + defGeneral.Name + "\n"
	}
	detail += troopChangeText(defBefore, defAfter)
	detail += "[双方兵力]"

	// 攻方战损入伤兵营：兵种修复率% + 战地医疗科技（管理端「效果数值」，未配置按 2%/级）+ 机械改装 5%/级
	// ★ 2026-10-15 机械改装回收率由 10%/级 → **5%/级**（名将满级最高 30%）
	healTech := ezfyTechEffectValueAt(21, atkTech[21])
	if h.officerHasSkill(leadOfficer, "机械改装") {
		healTech += 5 * h.officerSkillScale(leadOfficer)
	}
	var repairedTotal int64
	for _, g := range br.AttackerLosses {
		if g.Count <= 0 {
			continue
		}
		rate := 10
		if cfg := ezfyCfg.troop(g.TroopId); cfg != nil {
			rate = cfg.RepairRate
		}
		rate += healTech
		wounded := g.Count * int64(rate) / 100
		if wounded > g.Count {
			wounded = g.Count
		}
		if wounded > 0 {
			h.addWounded(city.ID, g.TroopId, 0, wounded)
			repairedTotal += wounded
		}
	}

	// 剩余部队（返航后归还）
	resultStr := ""
	for _, g := range br.AttackerLeft {
		if g.Count > 0 {
			resultStr += strconv.Itoa(g.TroopId) + ":" + strconv.FormatInt(g.Count, 10) + ","
		}
	}
	if len(resultStr) > 0 {
		resultStr = resultStr[:len(resultStr)-1]
	}
	order.Result = resultStr

	prestigeGain := 0
	if win {
		// 奖励：资源（档位基础值 × 类型倍数）+ 黄金（元宝）+ 必定掉宝 + 大量声望
		// ★ 2026-10-22 优先用「活动目标配置」里按类型+等级维护的奖励；缺省回退内置默认。
		res, gold, actPrestige := ezfyActRewardOf(actType, level)
		prestigeGain = actPrestige
		// ★ 2026-09-29 活动野地「按坐标配置」：奖励可配，优先于等级配置。
		if aw != nil && aw.Enabled == 1 {
			if aw.Res > 0 {
				res = aw.Res
			}
			if aw.Gold > 0 {
				gold = aw.Gold
			}
			if aw.Prestige > 0 {
				prestigeGain = aw.Prestige
			}
		}
		// ★ 2026-09-27 活动城奖励必须「累加」到「资源最大值」，不能按仓储上限 clamp：
		//   旧写法 min64(city.XxxCap*3, old+res) 会把「已超过仓储 3 倍」的存量直接拉低，
		//   导致打一次活动城资源反而变少（用户反馈「资源会掉」）。
		//   统一走 ezfyAddResMax（结果 = max(现值, min(资源最大值, 现值+增量))），与生产/入库同口径：
		//   无条件累加、只受资源最大值约束、老值超出也不被拉低。
		city.Food = ezfyAddResMax("food", city.Food, res)
		city.Steel = ezfyAddResMax("steel", city.Steel, res)
		city.Oil = ezfyAddResMax("oil", city.Oil, res)
		city.Rare = ezfyAddResMax("rare", city.Rare, res)
		city.Gold = ezfyAddResMax("gold", city.Gold, gold)
		h.saveCityRes(city)

		report += fmt.Sprintf("\n战利品: 粮%d 钢%d 油%d 稀矿%d 黄金%d", res, res, res, res, gold)
		// ★ 2026-10-09 「掉落 都走手动配置的」：活动目标**不再走旧的随机掉宝**
		//   （原 `wildlandLoot` + 全局概率 drop_t2/t3/t4 + drop_act_pct，已废弃），
		//   只按活动中手动配置的必掉宝物发放：
		//   · 活动野地坐标级配置 `aw.Treasures` 优先（个别坐标特调）；
		//   · 否则走「活动目标配置」按类型+等级维护的宝物（2026-10-22）。
		treasuresRaw := ""
		if aw != nil && aw.Enabled == 1 && strings.TrimSpace(aw.Treasures) != "" {
			treasuresRaw = aw.Treasures
		} else if cfg := ezfyActLevelCfg(actType, level); cfg != nil {
			treasuresRaw = cfg.Treasures
		}
		// ★ 2026-10-24 活动目标「必掉宝物」冷却（二战系统配置可配，默认 7200 秒 = 2 小时；0 = 关闭）：
		//   按**玩家**维度：距上次掉到宝物不足冷却秒数 → 本场不掉宝物；掉到才刷新冷却，没掉到不占冷却。
		actTreasureCd := ezfyActTreasureCooldownSec()
		treasureDropped := false
		if strings.TrimSpace(treasuresRaw) == "" {
			// 没配必掉宝物 → 不掉
		} else if actTreasureCd > 0 && !h.ezfyActDropReady(uid, "act_treasure_at", actTreasureCd) {
			// 冷却中 → 本场不掉宝物；冷却只对管理员可见，**不写进玩家战报**
		} else {
			// ★ 2026-10-11 宝物库存上限（默认 5000，二战系统配置可配）：装不下的不入包，战报给简约提示。
			left := h.ezfyTreasureLeft(city.UserID)
			// ★ 2026-10-24 掉落宝物多条合并成一条战报（用户要求）：如「翡翠项链 ×1、蓝宝石戒指 ×1」
			treasureNames := []string{}
			for _, tr := range parseActWildTreasures(treasuresRaw) {
				if e := ezfyCfg.equipment(tr[0]); e != nil {
					// 套装件只能开宝箱获得（与野地类型掉落同口径），配置误选时跳过。
					if e.SetId > 0 {
						continue
					}
					give := tr[1]
					if give > left {
						give = left
					}
					for k := 0; k < give; k++ {
						h.addEquipment(city, e)
					}
					left -= give
					if give > 0 {
						treasureNames = append(treasureNames, fmt.Sprintf("%s ×%d", e.Name, give))
						treasureDropped = true
						h.ezfySysChat("恭喜玩家 %s 缴获宝物: %s", h.ezfyProfileName(city.UserID), e.Name)
					}
					if give < tr[1] {
						report += fmt.Sprintf("\n宝物已达上限(%d), %s×%d 未能入包", ezfyTreasureMax(), e.Name, tr[1]-give)
					}
				}
			}
			if len(treasureNames) > 0 {
				report += fmt.Sprintf("\n掉落宝物: %s", strings.Join(treasureNames, "、"))
			}
			if treasureDropped {
				// 本次掉到宝物 → 记录掉宝时间，冷却从此刻起算（下次要等冷却过去才会再掉）
				h.DB.Model(&model.EzfyProfile{}).Where("user_id = ?", uid).
					Update("act_treasure_at", time.Now().UnixMilli())
			}
		}
		// ★ 2026-10-22 「活动目标配置 → 掉落道具」：逐条按概率掉落商城道具，
		//   受背包单件上限约束；掉到后写战报并全服播报（与野地道具掉落同口径）。
		// ★ 2026-10-24 「掉落道具」冷却（二战系统配置可配，默认 7200 秒 = 2 小时；0 = 关闭）：
		//   与必掉宝物冷却各自独立，按玩家维度；掉到才刷新冷却，没掉到不占冷却。
		actItemCd := ezfyActItemCooldownSec()
		if cfg := ezfyActLevelCfg(actType, level); cfg != nil && strings.TrimSpace(cfg.DropItems) != "" {
			if actItemCd > 0 && !h.ezfyActDropReady(uid, "act_item_at", actItemCd) {
				// 冷却中 → 本场不掉道具；冷却只对管理员可见，**不写进玩家战报**
			} else {
				lootNames := []string{}
				bagMax := ezfyBagItemMax()
				for _, d := range parseWildlandItemDrops(cfg.DropItems) {
					if rand.Intn(100) >= d[2] {
						continue
					}
					it := ezfyCfg.item(d[0])
					if it == nil {
						continue
					}
					if have := h.itemCount(uid, d[0]); have+d[1] > bagMax {
						report += fmt.Sprintf("\n背包已满, %s×%d 未能入包", it.Name, d[1])
						continue
					}
					h.addItem(uid, d[0], d[1])
					lootNames = append(lootNames, fmt.Sprintf("%s×%d", it.Name, d[1]))
				}
				if len(lootNames) > 0 {
					// ★ 2026-10-24 掉落道具多条合并成一条战报（用户要求，与掉落宝物同口径）
					report += fmt.Sprintf("\n掉落道具: %s", strings.Join(lootNames, "、"))
					// 本次掉到商城道具 → 记录掉道具时间，冷却从此刻起算
					h.DB.Model(&model.EzfyProfile{}).Where("user_id = ?", uid).
						Update("act_item_at", time.Now().UnixMilli())
					h.ezfySysChat("恭喜玩家 %s 在 %s(%d,%d) 缴获: %s",
						h.ezfyProfileName(uid), label, order.TargetX, order.TargetY,
						strings.Join(lootNames, "、"))
				}
			}
		}
		h.addPrestige(uid, prestigeGain)
		report += fmt.Sprintf("\n军功声望+%d", prestigeGain)
		// ★ 2026-09-29 活动目标守将：是否被俘虏只看 **CaptureRate**。
		//   CaptureRate=0 → 不俘虏；>0 → 按该百分比（100=必俘虏）。
		// ★ 2026-09-30 同一玩家可抓次数上限（MaxCapture，默认 1）：
		//   玩家已经抓到过该守将 ≥ 上限次数 → 本次概率强制 0（不再俘虏）。
		// ★★ 2026-10-28 计数口径改为「累计抓到过几次」（ezfy_act_capture_log 流水条数，只增）：
		//   用户规则 = 可抓次数是**针对玩家**的，抓住一次就计一次，即使之后被其他玩家抢走
		//   也不能再抓。旧实现数「当前拥有的军官实例数」，被抢走会归零 → 抓过的人又能再抓。
		//   同一个守将在哪个坐标打都算同一份计数（与「兜底被俘次数」同维度）。
		// ★ 2026-10-24 配置来源 aw 启用优先，否则回退活动目标配置（同一套口径，按 general_id 统计与坐标无关）。
		// ★ 2026-10-25 兜底被俘次数（min_capture_wins）：至少打赢该守将 N 次之后才可能被俘虏；
		//   0 = 第一次打赢就按概率（默认）。未达标**静默不俘**，战报与用户端都不出现这个门槛。
		//   进度只在「配了守将且被俘概率 > 0」时累计（没开俘虏就无需记进度）。
		if defCfg.OfficerId > 0 && defCfg.CaptureRate > 0 {
			if g := ezfyCfg.general(defCfg.OfficerId); g != nil {
				// ★ 2026-10-25 兜底被俘次数：先记一次「打赢该守将」，再判是否达标。
				//   达标线（min_capture_wins）**只给运营看**，未达标时静默不俘 —— 战报里不提示，
				//   用户端任何接口也不下发这个门槛（用户要求「不要展示到用户端」）。
				wins := h.ezfyActCaptureWinAdd(uid, g.ID)
				// ★ 2026-10-28 已抓次数（只增累计）。上限 = 0（不限）时**也要读**：
				//   保证计数一直是真实的 —— 运营之后把「可抓次数」从 0 改成 N 时，
				//   历史抓取次数不会从 0 重新算（玩家凭空多出机会）。
				captures := h.ezfyActCaptureUsed(uid, g.ID)
				rate := defCfg.CaptureRate
				// ★ 2026-10-24 顺带修复：MaxCapture=0（不限）时 captures=0，旧写法 `captures >= 0` 恒真
				//   会把概率强制归 0，等于「不限」反而永远抓不到 —— 只在上限 >0 时判达上限。
				if ezfyActCaptureFull(defCfg.MaxCapture, captures) {
					rate = 0 // 已达上限 → 概率 0
					report += "\n" + g.Name + "已被你捕获达到上限，无法再次俘虏"
				}
				// ★ 2026-10-25 兜底被俘次数未达标 → 本场不判俘虏（静默，不写战报）
				if rate > 0 && !ezfyActCaptureAllowed(defCfg.MinCaptureWins, wins) {
					rate = 0
				}
				if rate > 0 {
					// ★ 2026-10-24 被俘虏默认等级：配置了 CaptiveLevel 就按该等级发放俘虏（0 = 用军官池默认 1 级）
					captiveLv := 0
					if defCfg.CaptiveLevel > 0 {
						captiveLv = defCfg.CaptiveLevel
					}
					if c := h.createCaptiveOfficer(city, g, level, true, rate, captiveLv); c != "" {
						report += "\n" + c
						// ★ 2026-10-28 抓到即记一条流水（**只增不删**）：这就是「可抓次数」的扣减，
						//   即使之后这个守将被其他玩家抢走，本玩家的流水仍在、次数不会回退。
						h.ezfyActCaptureLogAdd(uid, g.ID, order.TargetX, order.TargetY,
							int64(city.ID), model.EzfyActCaptureKindBattle)
						// ★ 2026-10-05 名将野地**按玩家判定**（用户纠正）：只有**已抓到该守将的玩家**
						//   该坐标才是普通野地；对没抓到的玩家仍是名将野地。所以这里**不再全局禁用**——
						//   全局禁用会害得其他没抓到的玩家也打不到名将野地。
						//   已抓到该守将的玩家在 出征结算/地图/详情 处都会按普通野地处理（见
						//   playerOwnsActWildGeneral / processArrive / MapView / WildlandView）。
					}
				}
			}
		}
		order.Status = 2
		report = "我军胜利!\n" + report
	} else if draw {
		// ★ 2026-09-24 40 回合平局——残部返航（不是阵亡，不能 status=4 吃掉幸存部队）
		order.Status = 2
		report = "我军与敌军打成平局!\n" + report
	} else {
		order.Status = 4 // 全队阵亡
		report = "我军战败!\n" + report
	}
	// ★ 带队军官经验：胜负平局**都给**（「战败也有经验」），与普通战斗同款口径。
	//   即便战败/0 击杀也有基数 30 点；写库放这里统一处理（函数开头只算了数值）。
	if leadOfficer != nil && atkExp > 0 {
		h.addOfficerExp(city, leadOfficer.ID, atkExp)
		report += fmt.Sprintf("\n军官经验+%d", atkExp)
	}
	// ★ 2026-10-09 回收比例行（谁看展示谁的 → 这里只有攻方视角，算攻方自己那份）
	report += h.battleStatsTail(uid, prestigeGain, 50,
		ezfyHealPctLine(repairedTotal, br.AttackerLosses))

	// 返航时长与去程一致
	order.ReturnTime = now + ezfyAbs64(order.ArriveTime-order.StartTime)
	h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).Updates(map[string]interface{}{
		"status": order.Status, "result": order.Result, "return_time": order.ReturnTime,
	})
	// ★ 标题「活动野地3级战斗报告: 活动野地」冒号后面没带等级，看着像缺了目标。
	//   统一成「活动野地3级战斗报告: 活动野地3级(323,69)」，与普通战报「地形N级(坐标)」一致。
	h.addReport(uid, 3, label+"战斗报告: "+label+"("+strconv.Itoa(order.TargetX)+","+strconv.Itoa(order.TargetY)+")",
		report, detail, order.ID)
}

// ezfyActRewardBase 活动目标基础奖励（复刻 processActivityBattle 的 resBase 与类型倍数）
func ezfyActRewardBase(actType, level int) int64 {
	resBase := []int64{100000, 250000, 600000}
	res := resBase[minInt(2, level-1)]
	switch actType {
	case ezfyActKou:
		res *= 2
	case ezfyActCity:
		res *= 5
	}
	return res
}

// ezfyActGoldReward 活动目标黄金奖励（复刻 5000/10000/20000 × 等级）
func ezfyActGoldReward(actType, level int) int64 {
	base := int64(5000)
	switch actType {
	case ezfyActKou:
		base = 10000
	case ezfyActCity:
		base = 20000
	}
	return base * int64(level)
}

// ezfyActPrestigeReward 活动目标声望奖励（复刻 200 × 等级 × 类型倍数）
func ezfyActPrestigeReward(actType, level int) int {
	p := 200 * level
	switch actType {
	case ezfyActKou:
		p *= 3
	case ezfyActCity:
		p *= 5
	}
	return p
}

// ezfyActRewardOf 活动目标胜利奖励：优先取「活动目标配置」按类型+等级维护的值，
// 未配置（行不存在或值为 0）回退代码内置默认。
//
// ★ 2026-10-22 用户需求：三类活动目标的资源 / 黄金 / 声望要能在
// 二战管理 → 地图管理 → 活动目标配置 里维护。
func ezfyActRewardOf(actType, level int) (res int64, gold int64, prestige int) {
	res = ezfyActRewardBase(actType, level)
	gold = ezfyActGoldReward(actType, level)
	prestige = ezfyActPrestigeReward(actType, level)
	if cfg := ezfyActLevelCfg(actType, level); cfg != nil {
		if cfg.Res > 0 {
			res = cfg.Res
		}
		if cfg.Gold > 0 {
			gold = cfg.Gold
		}
		if cfg.Prestige > 0 {
			prestige = cfg.Prestige
		}
	}
	return
}

// ezfyActWildlandView 活动目标详情（守军预览 + 奖励预览 + 说明）
func (h *EzfyHandler) ezfyActWildlandView(uid uint, camp, x, y, actType int) gin.H {
	level := ezfyActivityLevel(x, y)
	terrain := ezfyTerrain(x, y)
	// ★ 2026-09-29 活动野地配置：等级优先用 ezfy_act_wild（enabled=1）
	aw := ezfyActWildAt(x, y)
	if aw != nil && aw.Enabled == 1 && aw.Level >= 1 && aw.Level <= 3 {
		level = aw.Level
	}
	// ★ 2026-10-22 活动目标等级配置（三类通用）
	actCfg := ezfyActLevelCfg(actType, level)
	defs := ezfyActivityDefender(actType, level, terrain)
	troops := []gin.H{}
	var total int64
	// ★ 2026-10-14 配置守军改为范围格式：预览按 [[兵种id,最小,最大]] 展示 min~max
	//   （不再随机取值，保证玩家看到的范围 = 实际打的随机区间；act_total 按「最大守军」算）
	// ★ 2026-10-22 先看坐标级配置（活动野地特调），再看等级配置，最后才用内置默认。
	troopRaw := ""
	if aw != nil && aw.Enabled == 1 && strings.TrimSpace(aw.Troops) != "" {
		troopRaw = aw.Troops
	} else if actCfg != nil {
		troopRaw = actCfg.Troops
	}
	if strings.TrimSpace(troopRaw) != "" {
		if rgs := parseActWildTroopRanges(troopRaw); rgs != nil {
			for _, rg := range rgs {
				tid, lo, hi := int(rg[0]), rg[1], rg[2]
				total += hi
				troops = append(troops, gin.H{
					"troop_id": tid, "name": ezfyCfg.troopName(tid, camp),
					"min": lo, "max": hi,
				})
			}
		}
	}
	if len(troops) == 0 {
		for _, g := range defs {
			total += g.Count
			troops = append(troops, gin.H{
				"troop_id": g.TroopId, "name": ezfyCfg.troopName(g.TroopId, camp),
				"min": g.Count, "max": g.Count,
			})
		}
	}
	res, gold, prestige := ezfyActRewardOf(actType, level)
	if aw != nil && aw.Enabled == 1 {
		if aw.Res > 0 {
			res = aw.Res
		}
		if aw.Gold > 0 {
			gold = aw.Gold
		}
		if aw.Prestige > 0 {
			prestige = aw.Prestige
		}
	}
	jewelName := ""
	if j := h.randomJewel(terrain); j != nil {
		jewelName = j.Name
	}
	actDesc := ezfyActTargetDesc(actType)
	if actCfg != nil && strings.TrimSpace(actCfg.Des) != "" {
		actDesc = actCfg.Des
	}
	officerName := ""
	officerKind := 0
	officerStar := 0
	// ★ 2026-10-24 守将来源与战斗同口径：aw 启用优先，否则回退活动目标配置
	if dc := ezfyActDefConfigOf(aw, actType, level); dc.OfficerId > 0 {
		if g := ezfyCfg.general(dc.OfficerId); g != nil {
			officerName = g.Name
			officerKind = g.Kind // 1普通军官 2名将
			officerStar = g.Star
		}
	}
	// ★ 2026-09-30 详情直接下发收藏态，避免前端依赖异步 loadStars 的时序导致初始态错
	var starCnt int64
	h.DB.Model(&model.EzfyMapStar{}).Where("user_id = ? AND x = ? AND y = ?", uid, x, y).Count(&starCnt)
	isStarred := starCnt > 0
	return gin.H{
		"x": x, "y": y, "type": 1, "level": level,
		"name":     ezfyActTargetLabel(actType, level),
		"act_type": actType, "act_level": level,
		"act_name":  ezfyActTargetName(actType),
		"act_desc":  actDesc,
		"act_total": total,
		"troops":    troops,
		"res_min":   res,
		"res_max":   res,
		"gold":      gold,
		"prestige":  prestige,
		"terrain":   terrain,
		// ★ 2026-10-05：活动野地本身有野地 ⇒ knownWild=true（海里的叫「海底森林」，岛屿仍是「岛屿」）
		"terrain_name": ezfyWildTerrainDisplayName(x, y, true),
		// ★ 2026-10-25 用户「活动野地所属区域显示大海，应该是具体的洲」：
		//   ezfyContinentName 只认陆地，海洋坐标一律回落「大海」；改用 ezfyRegionName
		//   （陆地给洲名、海洋给大洋名），与 /view、地图格子、野地弹窗同口径。
		"continent":    ezfyRegionName(x, y),
		"jewel":        jewelName,
		// ★ 2026-09-29 活动目标守将（配置的军官池军官；★ 2026-10-24 aw 优先，否则回退等级配置）
		"officer_id":   ezfyActDefConfigOf(aw, actType, level).OfficerId,
		"officer_name": officerName,
		"officer_kind": officerKind, // ★ 2026-09-30 区分名将(2)/普通(1)
		"officer_star": officerStar,
		"is_starred":   isStarred, // ★ 2026-09-30 收藏态（前端收藏按钮直接用）
		"owner":        "",
	}
}
