package ezfy

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/model"
)

// parseActWildTroops 解析活动野地配置的守军 JSON [[兵种id,count],...]→ 战斗兵组列表
//
// 失败/空串返回 nil。count 不合法（<=0）的行丢弃。
func parseActWildTroops(raw string) []ezfyUnitGroup {
	var rows [][2]int
	if err := json.Unmarshal([]byte(raw), &rows); err != nil {
		return nil
	}
	out := []ezfyUnitGroup{}
	for _, r := range rows {
		if r[0] <= 0 || r[1] <= 0 {
			continue
		}
		out = append(out, ezfyUnitGroup{TroopId: r[0], Count: int64(r[1])})
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
	if g != nil && generalHasSkill(g, "绝地反击") {
		return generalSkillLevel(g)
	}
	return 0
}

// generalSkillDefBonus 军官池守将「防御类技能的攻击加成拆解段」（弧形防御 30 / 弹幕支援 10，
// 随技能等级 ×N）。活动守军无城墙/无科技 → 守方攻击加成整体来自守将（defBonus），
// 这里取下其中的技能段，供战报拆解单独展示「军官技能+N%」。
func generalSkillDefBonus(g *model.EzfyCfgGeneral) int {
	if g == nil {
		return 0
	}
	scale := generalSkillScale(g)
	bonus := 0
	for _, s := range generalSkillList(g) {
		switch s {
		case "弧形防御":
			bonus += 30 * scale
		case "弹幕支援":
			bonus += 10 * scale
		}
	}
	return bonus
}

// generalSkillDefBreak 军官池守将「防御类技能」的**逐项**明细（战报展示「军官技能·弧形防御+N%」）。
// 与 generalSkillDefBonus 同一口径，明细之和 = 技能总加成。
func generalSkillDefBreak(g *model.EzfyCfgGeneral) []ezfyBonusItem {
	if g == nil {
		return nil
	}
	scale := generalSkillScale(g)
	out := []ezfyBonusItem{}
	for _, s := range generalSkillList(g) {
		switch s {
		case "弧形防御":
			out = append(out, ezfyBonusItem{Name: s, Value: 30 * scale})
		case "弹幕支援":
			out = append(out, ezfyBonusItem{Name: s, Value: 10 * scale})
		}
	}
	return out
}

// ezfyActWildDefBonus 活动野地配置了守将时的守方加成（复刻玩家城「城守」口径）。
//
// ★ 2026-09-29 修复：活动野地原来「守方加成恒为 0」，导致配了守将也看不出守方厉害
//
//	（战斗加成里 守方 防御+0% 速度+0%）。现在按守将有效学识给防御、守将速度技能给速度。
//	无科技/城墙/装备：防御 = (有效学识+1)/2 + 弧形防御+30 / 弹幕支援+10；
//	速度 = 命中 坦克突袭 / 闪电袭击 / 越岛战术 任一 +10。
//
// ★ 2026-10-06 守将技能随等级自动升级：弧形防御/弹幕支援/速度 均 ×技能倍率
//
//	（属性部分 ezfyAttrToBonus(g.Learning) 不加倍）。
//
// 返回 (防御加成, 速度加成)。
func ezfyActWildDefBonus(g *model.EzfyCfgGeneral) (int, int) {
	if g == nil {
		return 0, 0
	}
	scale := generalSkillScale(g)
	def := 0
	speed := 0
	hasSpeed := false
	for _, s := range generalSkillList(g) {
		switch s {
		case "弧形防御":
			def += 30 * scale
		case "弹幕支援":
			def += 10 * scale
		case "坦克突袭", "闪电袭击", "越岛战术":
			hasSpeed = true
		}
	}
	def += ezfyAttrToBonus(g.Learning)
	if hasSpeed {
		speed = 10 * scale
	}
	return def, speed
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

// playerOwnsActWildGeneral 该玩家是否已拥有此坐标活动野地的守将（名将野地按玩家判定）。
//
// ★ 2026-10-05 用户规则：名将野地对没抓到守将的玩家仍是名将野地；只有**已抓到该守将的玩家**，
//
//	该坐标才是普通野地。判定口径 = 该玩家名下任一城市的军官里存在 general_id == 守将ID
//	（收编后 IsCaptive=0 也算已拥有）。快速路径：非活动野地/没配守将 → 直接 false，零查询。
func (h *EzfyHandler) playerOwnsActWildGeneral(uid uint, x, y int) bool {
	aw := ezfyActWildAt(x, y)
	if aw == nil || aw.Enabled != 1 || aw.OfficerId <= 0 {
		return false
	}
	g := ezfyCfg.general(aw.OfficerId)
	if g == nil {
		return false
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
	leadOfficer := h.officerByName(city.ID, order.Officer)
	// ★ 2026-10-06 弹道学(8) 改为**射程加成**（用户要求：弹道学=射程，不参与攻击加成）
	atkBonus := h.officerBattleBonus(leadOfficer) +
		atkTech[5]*2 + atkTech[6]*3 + atkTech[9]*2
	// 速度：燃烧引擎(10)「部队速度」= 通用；喷气引擎(19)「空军速度」= 兵种专属（见下方 atkType）
	atkSpeedBonus := atkTech[10] * 2
	// ★ 2026-10-08 兵种专属加成（与普通出征同口径）：喷气引擎只加空军 + 军官兵种技能
	atkType := ezfyTypeBonus{Speed: map[int]int{}}
	if v := atkTech[19] * 3; v > 0 {
		atkType.Speed[ezfyTroopTypeAir] += v
	}
	atkType = h.officerTypeBonus(leadOfficer, atkType)
	// ★ 2026-10-08 与普通出征同口径：军官行同时展示攻击加成与防御加成（属性部分，技能单列）
	atkOfficerDesc := h.officerBattleDesc(leadOfficer, h.officerBaseBonus(leadOfficer), "攻击加成",
		h.officerGuardAttrBonus(leadOfficer), "防御加成")
	// ★ 2026-10-06 战报拆解逐项明细：攻方科技/技能逐项（与普通出征同一口径）
	// ★ 2026-10-08 补全影响 攻击/防御/射程/速度 的全部攻方科技（与普通出征同口径）
	atkTechs := ezfyBonusItems(
		ezfyTechItem("军训艺术", atkTech[5]*2),
		ezfyTechItem("武器科技", atkTech[6]*3),
		ezfyTechItem("装甲科技", atkTech[7]*3),
		ezfyTechItem("弹道学", atkTech[8]*3),
		ezfyTechItem("重工技术", atkTech[9]*2),
		ezfyTechItem("燃烧引擎", atkTech[10]*2),
		ezfyTechItem("掩体防御", atkTech[16]*2),
		ezfyTechItem("喷气引擎", atkTech[19]*3),
	)
	atkSkillBreak := h.officerSkillsBreak(leadOfficer)
	// ★ 2026-10-07 攻方「防御加成」（出征军官属性+防御技能(弧形防御/弹幕支援)+装备 Def，
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
	// ★ 2026-09-29 修复「守方军官加成恒为 0」：活动野地配置了守将时，把守将的
	//   防御加成 / 速度加成 / 战报描述 接进战斗引擎（指挥室和战报都从这里生成）。
	var defGeneral *model.EzfyCfgGeneral
	if aw != nil && aw.Enabled == 1 && aw.OfficerId > 0 {
		defGeneral = ezfyCfg.general(aw.OfficerId)
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
	//   被打行展示「防御加成+N%(军官·名+X% 军官技能·弧形防御+Y% …)」，与 defBonus 构成同口径）
	defDefBreak := []ezfyBonusItem{}
	if defGeneral != nil {
		defDefBreak = append(defDefBreak, ezfyBonusItem{Name: "军官·" + defGeneral.Name, Value: ezfyAttrToBonus(defGeneral.Learning)})
		for _, s := range generalSkillDefBreak(defGeneral) {
			defDefBreak = append(defDefBreak, ezfyBonusItem{Name: "军官技能·" + s.Name, Value: s.Value})
		}
	}
	if done, ok := ezfyBattleResultDecode(order.BattleResult); ok {
		br = done
	} else {
		// 活动守军无城墙/无科技 → 只有守将的加成；攻方装备六项加成照常生效；
		// 守方阵营: 活动野地=盟军、活动寇/特殊城市=轴心国
		// ★ 2026-10-06 攻方射程加成 = 弹道学(8)*3；活动守军无科技，射程/守方攻击加成取守将加成(0 兜底)
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
			h.buildMoveMap(city.ID, true), map[int]int{},
			h.officerCounterRounds(leadOfficer),
			generalCounterRounds(defGeneral),
			h.ensureProfile(uid).Camp, defCamp,
			// ★ 2026-10-08 兵种专属加成（攻方喷气引擎=空军速度+军官兵种技能；活动守军无科技 → 空表）
			atkType, ezfyTypeBonus{Atk: map[int]int{}, Speed: map[int]int{}})
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
	report += fmt.Sprintf("[%s]攻方:%s\n", atkTag, city.Name)
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
	detail += fmt.Sprintf("[%s]攻方:%s\n", atkTag, city.Name)
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

	// 攻方战损入伤兵营：兵种修复率% + 治愈伤兵科技 2%/级 + 机械改造 10%
	healTech := atkTech[21] * 2
	if h.officerHasSkill(leadOfficer, "机械改造") {
		healTech += 10 * h.officerSkillScale(leadOfficer)
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
		// ★ 2026-09-29 活动野地配置：奖励可配，缺省回退默认
		res := ezfyActRewardBase(actType, level)
		gold := ezfyActGoldReward(actType, level)
		prestigeGain = ezfyActPrestigeReward(actType, level)
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
		if loot := h.wildlandLoot(city, level*3, terrain, true); loot != "" {
			report += "\n" + loot
		}
		// ★ 2026-09-29 必掉宝物多行配置 [[cfg_id,数量]]：胜利后按配置掉落多件（增强宝，保证进背包）
		if aw != nil && aw.Enabled == 1 && strings.TrimSpace(aw.Treasures) != "" {
			for _, tr := range parseActWildTreasures(aw.Treasures) {
				if e := ezfyCfg.equipment(tr[0]); e != nil {
					for k := 0; k < tr[1]; k++ {
						h.addEquipment(city, e)
					}
					report += fmt.Sprintf("\n掉落宝物: %s ×%d", e.Name, tr[1])
					h.ezfySysChat("恭喜玩家 %s 缴获宝物：%s", h.ezfyProfileName(city.UserID), e.Name)
				}
			}
		}
		h.addPrestige(uid, prestigeGain)
		report += fmt.Sprintf("\n军功声望+%d", prestigeGain)
		// ★ 2026-09-29 活动野地守将：是否被俘虏只看 **CaptureRate**。
		//   aw.CaptureRate=0 → 不俘虏；>0 → 按该百分比（100=必俘虏）。
		// ★ 2026-09-30 同一玩家可抓次数上限（aw.MaxCapture，默认 1）：
		//   玩家已经抓到过该守将 ≥ 上限次数 → 本次概率强制 0（不再俘虏）。
		//   判定口径：该玩家名下所有城里、general_id 为该守将、且是「真俘虏」（IsCaptive=1 或
		//   已收编的也得算 —— 收编后 IsCaptive=0 但玩家已拥有，不能再抓）。
		//   所以直接用「该玩家已拥有该守将的军官实例数」来算（含俘虏与收编，一次发放=1）。
		if aw != nil && aw.Enabled == 1 && aw.OfficerId > 0 && aw.CaptureRate > 0 {
			if g := ezfyCfg.general(aw.OfficerId); g != nil {
				captures := 0
				if aw.MaxCapture > 0 {
					var had int64
					// ★ 2026-10-06：军校招募的军官现在也写 general_id（= 池子 id，便于丢官追溯），
					//   所以这里必须**排掉「军校招募」来源**，否则「从军校招过同一个池子军官」会被
					//   误判成「已经抓过这个守将」而抓不了 —— 保持本次改动前的口径不变。
					h.DB.Model(&model.EzfyOfficer{}).
						Where("general_id = ? AND source <> ? AND city_id IN (?)", g.ID,
							model.EzfyOfficerSourceRecruit,
							h.DB.Model(&model.EzfyCity{}).Select("id").Where("user_id = ?", uid)).
						Count(&had)
					captures = int(had)
				}
				rate := aw.CaptureRate
				if captures >= aw.MaxCapture {
					rate = 0 // 已达上限 → 概率 0
					report += "\n" + g.Name + "已被你捕获达到上限，无法再次俘虏"
				}
				if rate > 0 {
					if c := h.createCaptiveOfficer(city, g, level, true, rate); c != "" {
						report += "\n" + c
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
	report += ezfyWoundedReportLine(repairedTotal, br.AttackerLosses, healTech)
	report += h.battleStatsTail(uid, prestigeGain, 50)

	// 返航时长与去程一致
	order.ReturnTime = now + ezfyAbs64(order.ArriveTime-order.StartTime)
	h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).Updates(map[string]interface{}{
		"status": order.Status, "result": order.Result, "return_time": order.ReturnTime,
	})
	// ★ 用户反馈：标题「活动野地3级战斗报告: 活动野地」冒号后面没带等级，看着像缺了目标。
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

// ezfyActWildlandView 活动目标详情（守军预览 + 奖励预览 + 说明）
func (h *EzfyHandler) ezfyActWildlandView(uid uint, camp, x, y, actType int) gin.H {
	level := ezfyActivityLevel(x, y)
	terrain := ezfyTerrain(x, y)
	// ★ 2026-09-29 活动野地配置：等级/守军/奖励 优先用 ezfy_act_wild（enabled=1）
	aw := ezfyActWildAt(x, y)
	if aw != nil && aw.Enabled == 1 && aw.Level >= 1 && aw.Level <= 3 {
		level = aw.Level
	}
	defs := ezfyActivityDefender(actType, level, terrain)
	if aw != nil && aw.Enabled == 1 && strings.TrimSpace(aw.Troops) != "" {
		if cfg := parseActWildTroops(aw.Troops); cfg != nil && len(cfg) > 0 {
			defs = cfg
		}
	}
	troops := []gin.H{}
	var total int64
	for _, g := range defs {
		total += g.Count
		troops = append(troops, gin.H{
			"troop_id": g.TroopId, "name": ezfyCfg.troopName(g.TroopId, camp),
			"min": g.Count, "max": g.Count,
		})
	}
	res := ezfyActRewardBase(actType, level)
	gold := ezfyActGoldReward(actType, level)
	prestige := ezfyActPrestigeReward(actType, level)
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
	officerName := ""
	officerKind := 0
	officerStar := 0
	if aw != nil && aw.Enabled == 1 && aw.OfficerId > 0 {
		if g := ezfyCfg.general(aw.OfficerId); g != nil {
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
		"act_desc":  ezfyActTargetDesc(actType),
		"act_total": total,
		"troops":    troops,
		"res_min":   res,
		"res_max":   res,
		"gold":      gold,
		"prestige":  prestige,
		"terrain":   terrain,
		// ★ 2026-10-05：活动野地本身有野地 ⇒ knownWild=true（海里的叫「海底森林」，岛屿仍是「岛屿」）
		"terrain_name": ezfyWildTerrainDisplayName(x, y, true),
		"continent":    ezfyContinentName(x, y),
		"jewel":        jewelName,
		// ★ 2026-09-29 活动野地守将（配置的军官池军官）
		"officer_id": func() int {
			if aw != nil && aw.Enabled == 1 {
				return aw.OfficerId
			}
			return 0
		}(),
		"officer_name": officerName,
		"officer_kind": officerKind, // ★ 2026-09-30 区分名将(2)/普通(1)
		"officer_star": officerStar,
		"is_starred":   isStarred, // ★ 2026-09-30 收藏态（前端收藏按钮直接用）
		"owner":        "",
	}
}
