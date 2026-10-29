// 风云榜（每周玩家对战战绩，★ 2026-10-14；★ 2026-10-14 按积分排名）
//
// 按**自然周**（周一 00:00 ~ 周日 23:59）统计**玩家 vs 玩家**战斗：
// 灭敌(击杀敌军数) / 战损(己方阵亡数) / 参战场次 / 胜场 / 积分 → 按**积分**排名。
// 只记打玩家城（target_type=3），野地/寇城/活动目标一律不算 PVP。
// 每个玩家每周一行（uk_pvp_week_user），战斗结算时按(week,user_id)累加。
package ezfy

import (
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// ezfyPvpWeekOf 取时间 t 所属自然周的周串："20060102-20060102"（周一 00:00 ~ 周日 23:59）。
func ezfyPvpWeekOf(t time.Time) string {
	loc := t.Location()
	// 周一 = 本周起点：Weekday() 里 Sunday=0，把 Monday 视为 0 则偏移 = (wd+6)%7
	offset := (int(t.Weekday()) + 6) % 7
	mon := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -offset)
	sun := mon.AddDate(0, 0, 6)
	return mon.Format("20060102") + "-" + sun.Format("20060102")
}

// pvpCorpsNameOf 玩家当前军团名（ezfy_corps_member 唯一，无则空串）。
func (h *EzfyHandler) pvpCorpsNameOf(uid uint) string {
	var mb model.EzfyCorpsMember
	if err := h.DB.Where("user_id = ?", uid).First(&mb).Error; err != nil {
		return ""
	}
	var cp model.EzfyCorps
	if err := h.DB.Select("name").First(&cp, mb.CorpsId).Error; err != nil {
		return ""
	}
	return cp.Name
}

// 风云榜防刷阈值（★ 2026-10-14 上线后发现 1 兵刷 / 小号互刷，加三条防刷规则）
const (
	// ezfyPvpMinTroop 本场己方投入兵力（加权口径）低于此值 → 本场**不计积分**（★ 2026-10-16 1000→10000→20000）。
	//	卡控小号/垃圾兵刷分：约「1 万步兵 / 70 艘航母」才够门槛，即**满一场常规装载的司令部出击**。
	//	能带多少兵本来就受「司令部等级 + 集结令」的携带上限约束 —— 带上集结令把上限拉高、多带兵，
	//	分自然更高（兵越多分越多）；靠几千垃圾兵互点无效。攻守同规则，任一方不足整仗 0 分。
	ezfyPvpMinTroop = int64(20000)
	// ezfyPvpFoeCapDef 同一自然周内，与**同一对手**对战的积分累计封顶**默认值**（防小号互刷）。
	// ★ 2026-10-16 8000 → 20000：加权兵种分后单仗更能打了，封顶同步放宽。
	// ★ 2026-10-29 改为可配置（ezfy_cfg_limit.pvp_foe_cap，见 ezfyPvpFoeCap）。
	// 达上限后与该对手再交战：场次/灭敌/战损照记，积分不再计入榜单。
	ezfyPvpFoeCapDef = int64(20000)
)

// ezfyPvpFoeCap 同一周内与同一对手的积分封顶（读 ezfy_cfg_limit.pvp_foe_cap，默认 20000）。
//
// ★ 2026-10-29 用户需求「pvp 对战同一个人积分上限也可以配置，默认 20000，可灵活调整」。
// ⚠️ 0/负数无意义（= 与任何人交手都不计分）→ 未加载/非法值一律回落默认 20000。
func ezfyPvpFoeCap() int64 {
	if !ezfyCfg.ready() || ezfyCfg.limit.PvpFoeCap <= 0 {
		return ezfyPvpFoeCapDef
	}
	v := int64(ezfyCfg.limit.PvpFoeCap)
	if v > 100000000 { // 防呆：上限 1 亿
		v = 100000000
	}
	return v
}

// pvpUnitCost 兵种分权重（★ 2026-10-16 用户要求「按建造资源多少算分」）。
//
//	权重 = 建造资源总成本(粮/钢/油/稀有) ÷100，向下取整、至少 1、封顶 300；
//	步兵≈2，特种兵≈48，火箭≈67，战列舰/航母等封顶 300 —— 同数量下贵兵种分远高于步兵。
//	侦察机(9)/卡车(3) 权重=0：运输/侦察队不算战力，灭敌/战损/兵力统统不计入积分（堵刷分）。
//	懒构建一次（配置整表已进缓存），并发安全。
var (
	pvpCostOnce sync.Once
	pvpCost     map[int]int64
)

func pvpUnitWeight(id int) int64 {
	pvpCostOnce.Do(func() {
		m := make(map[int]int64)
		for _, t := range ezfyCfg.sortedTroops() {
			w := (t.Food + t.Steel + t.Oil + t.Rare) / 100
			if w < 1 {
				w = 1
			}
			if w > 300 {
				w = 300
			}
			m[t.ID] = w
		}
		m[3] = 0 // 卡车  不参与积分
		m[9] = 0 // 侦察机 不参与积分
		pvpCost = m
	})
	if w, ok := pvpCost[id]; ok {
		return w
	}
	return 1
}

// pvpUnitsOf 统计 ezfyUnitGroup 总兵力（风云榜口径：按兵种分权重加权，侦察机/卡车不计）。
//
// ★ 2026-10-16 原先是数人头——侦察机/卡车/步兵堆数量刷分、没阶梯；
// 改为「Σ 权重×数量」：同数量下航母/战列舰等昂贵兵种远高于步兵，堵无差别刷步兵。
// 同时承担加权灭敌 / 加权战损两个角色（格式相同，省一次遍历）。
func pvpUnitsOf(gs []ezfyUnitGroup) int64 {
	var n int64
	for _, g := range gs {
		w := pvpUnitWeight(g.TroopId)
		if w > 0 {
			n += w * g.Count
		}
	}
	return n
}

// pvpBattlePoints 单场 PVP 积分（风云榜，★ 2026-10-14 规则；杀敌分同日本轮补充）。
//
//		积分 = ((行动分 + 胜负分) × 兵力劣势系数 + 杀敌分 + 战损比分) × 兵力规模系数，
//		单场**封顶按规模分档**（★ 2026-10-16 用户要求「鼓励大仗、小仗卡 400 以下、大兵团趋近封顶」；
//		★ 2026-10-29 上限改为可配置 pvp_battle_cap，默认 800）：
//		  本场投入 < 大兵团线(300 万加权 ≈ 10000 航母 / 150 万步兵) → 单场最多 **上限的一半**（默认 400）；
//		  达到大兵团线（集结令拉满级别的会战）→ 单场最多 **上限**（默认 800）。
//
//	  - 行动分：进攻(掠夺/征服) 20 分，防守 15 分（进攻稍高，鼓励主动出击）
//	  - 胜负分：胜 40 / 平 15 / 负 10（参战即有，鼓励对抗）
//	  - 兵力劣势系数：己方本场投入兵力 < 对方时，以少打多奖励：
//	    系数 = min(3.0, 对方兵力÷己方兵力)；攻方出征少、守方兵力多时攻击方按比例**多拿分**；
//	    系数对双方都生效（谁兵力劣势谁加成），≥1 时系数=1
//	  - 兵力规模系数（★ 2026-10-16）：按己方投入兵力取 0.3×~3.0×（见 pvpArmyScale），
//	    大兵团分越多、少兵碰瓷的白拿分被压低
//	  - 杀敌分（★ 2026-10-14 用户反馈「大战积分太少」补上；★ 2026-10-16 灭敌改按兵种价值加权后
//	    同数量高级兵种 > 低级兵种）：按**加权灭敌量**对数缩放，
//	    20×log10(灭敌) 封顶 200（★ 2026-10-29 可在管理端关掉上限 → 无上限）——
//	    千万级大战与百万级小仗拉开差距、大仗有分量；
//	    对数压缩 → 灭敌量翻 10 倍只多 20 分，不会被天文数字刷爆。1 兵刷/空城刷由门槛拦住。
//	  - 战损比分：按 战损比 = 灭敌÷战损 给分，kdr×10 封顶 30；无损(战损=0)按满 30 给
const (
	// ezfyPvpKillPtCap 杀敌分封顶
	ezfyPvpKillPtCap = int64(200)
	// ezfyPvpMaxPointDef 单场分数上限**默认值**（大兵团会战）。
	// ★ 2026-10-29 用户需求「风云榜加个单场分数上限配置，默认 800，可灵活调整」：
	//   实际值读 ezfy_cfg_limit.pvp_battle_cap（见 ezfyPvpBattleCap），这里只是默认/兜底。
	ezfyPvpMaxPointDef = int64(800)
	// ezfyPvpSmallMaxDef 小仗单场上限**默认值**（★ 2026-10-16 用户要求「小仗卡控在 400 以下」）
	//   实际值 = 单场上限的一半（见 ezfyPvpSmallCap），默认 800 → 400。
	ezfyPvpSmallMaxDef = int64(400)
	// ezfyPvpBigArmyPower 大兵团线（加权口径）：本场投入达到该值才用「全额」单场上限
	// ≈ 1 万航母 / 150 万步兵 —— 不带集结令拉上限的话基本到不了，天然只属于大兵团会战
	ezfyPvpBigArmyPower = int64(3000000)
)

// ezfyPvpBattleCap 风云榜单场分数上限（读 ezfy_cfg_limit.pvp_battle_cap，默认 800）。
//
// ★ 2026-10-29 用户需求「加个单场分数上限配置，默认 800，可灵活调整」：
//
//	大兵团会战（投入 ≥ 大兵团线）按此封顶；小仗按**一半**封顶（800 → 400，保持原平衡）。
//	⚠️ 0/负数无意义（= 每场 0 分）→ 未加载/非法值一律回落默认 800，不会把榜单打成全 0。
func ezfyPvpBattleCap() int64 {
	if !ezfyCfg.ready() || ezfyCfg.limit.PvpBattleCap <= 0 {
		return ezfyPvpMaxPointDef
	}
	v := int64(ezfyCfg.limit.PvpBattleCap)
	if v > 1000000 { // 防呆：上限 100 万，别让手滑写成天文数字
		v = 1000000
	}
	return v
}

// ezfyPvpSmallCap 小仗单场上限 = 单场上限的一半（默认 800 → 400）。
func ezfyPvpSmallCap() int64 {
	v := ezfyPvpBattleCap() / 2
	if v < 1 {
		v = 1
	}
	return v
}

// pvpArmyScale 兵力规模系数（★ 2026-10-16 用户要求「大兵团作战分越多，少兵刷不了」）。
//
//	按本场**实际投入兵力(加权口径)**取规模档，百分比返回（100 = ×1.0）：
//	投入越大系数越高（0 ~ 3.5×）。
//	目的：把「少兵碰瓷/小号垃圾兵互刷」彻底压死 ——
//	门槛（≈1 万步兵/70 航母，一场常规装载）以下整仗 0 分；贴着门槛的部队白拿分只给四成；
//	大兵团会战（≥300 万加权，99 集结令+满级司令部级别）直接 ×3.5，顶满单场 800。
//	与「以少打多加成」独立叠加：大兵团打大兵团互吃高系数，小部队打大兵团仍以少胜多有余地。
func pvpArmyScale(power int64) int64 {
	switch {
	case power < 20000:
		return 0 // 门槛以下（≈1 万步兵/70 航母 = 一场常规装载）：整仗不计分（★ 2026-10-16 小号/垃圾兵刷分卡控）
	case power < 60000:
		return 40 // ×0.4  小部队：白拿分（行动+胜负+战损比）压到极低
	case power < 200000:
		return 80 // ×0.8
	case power < 600000:
		return 120 // ×1.2  正常一仗
	case power < 2000000:
		return 160 // ×1.6
	case power < 3000000:
		return 220 // ×2.2
	default:
		return 350 // ×3.5  大兵团会战（≥300 万加权=大兵团线，99 集结令满级司令部可到）
	}
}

func pvpBattlePoints(kills, losses, myPower, foePower int64, actBase int64, win, draw bool) int64 {
	// 杀敌分：20×log10(灭敌)，0~200（灭敌 0 不给分）
	// ★ 2026-10-29 上限可关（管理端「排行榜 → 风云榜 · 灭敌分上限」）：关掉后不封顶。
	killPts := int64(0)
	if kills > 0 {
		killPts = int64(20 * math.Log10(float64(kills)))
		if ezfyPvpKillCapOn() && killPts > ezfyPvpKillPtCap {
			killPts = ezfyPvpKillPtCap
		}
	}
	// 战损比奖励分 0~30
	kdrPts := int64(30)
	if losses > 0 {
		kdrPts = 10 * kills / losses
		if kdrPts > 30 {
			kdrPts = 30
		}
	}
	// 胜负分
	outcome := int64(10)
	if draw {
		outcome = 15
	} else if win {
		outcome = 40
	}
	// 兵力劣势系数（百分比，1.0=100，最高 300=3.0）
	mult := int64(100)
	if foePower > myPower && myPower > 0 {
		mult = int64(float64(100) * float64(foePower) / float64(myPower))
		if mult > 300 {
			mult = 300
		}
	}
	// ★ 2026-10-16 单场封顶按规模分档：小仗 400，大兵团会战（≥300 万加权）800 —— 鼓励大仗
	// ★ 2026-10-29 两个值改为读配置：上限 = pvp_battle_cap（默认 800），小仗 = 上限的一半
	cap := ezfyPvpSmallCap()
	if myPower >= ezfyPvpBigArmyPower {
		cap = ezfyPvpBattleCap()
	}
	pts := ((actBase+outcome)*mult/100 + killPts + kdrPts) * pvpArmyScale(myPower) / 100
	if pts > cap {
		pts = cap
	}
	return pts
}

// accumPvpWeekOnce 把一位玩家的本周战绩累加进风云榜（按 (week,user_id) 唯一键 upsert）。
//   - battles: 本场 +1；wins: 本场胜 1，否则 0（平局双方都不算胜）；
//   - kills: 本场灭敌（击杀敌军数）；losses: 本场战损（己方阵亡数）；
//   - points: 本场风云积分（周累计，按积分排名）；foeID 为对手玩家ID，做同对手封顶。
//   - 另写一条「我的战绩」场次明细（★ 2026-10-16 战报会定期删，明细表独立留存）：
//     记的是**实际入账分**（封顶/门槛/同对手上限之后），0 分也记——玩家每场加没加、加了多少一目了然。
//
// 昵称/军团随写榜时刻刷新（玩家改名/换军团后历史周保留当时名字）。
//
// ★ 2026-10-26 返回**明细行 id**：结算流程末尾要把「本场战报 id」回填到这条明细
// （管理端「积分明细 → 查看战报」），战报是在明细之后才写的，所以只能先拿 id 后补。
func (h *EzfyHandler) accumPvpWeekOnce(week string, uid, foeID uint, nick, foeNick, corps string, ts int64, wins, kills, losses, points int64, draw bool) uint {
	if uid == 0 || week == "" {
		return 0
	}
	// 本场实际入账分（同对手封顶截断后），先保存再写明细
	credited := points
	// ★ 防刷：同一周与同一对手对战的积分封顶（小号互刷只能刷到固定上限）。
	//   已计分 ≥ 上限 → 本场积分 0；未达上限 → 最多补到上限。场次/灭敌/战损不受影响。
	if points > 0 && foeID > 0 {
		foeCap := ezfyPvpFoeCap() // ★ 2026-10-29 改为读配置（默认 20000，管理端可调）
		var cur int64
		h.DB.Model(&model.EzfyPvpFoe{}).
			Where("week = ? AND user_id = ? AND foe_id = ?", week, uid, foeID).
			Select("points").Scan(&cur)
		if cur >= foeCap {
			points = 0
		} else if cur+points > foeCap {
			points = foeCap - cur
		}
		credited = points
		h.DB.Exec(`INSERT INTO ezfy_pvp_foe (week, user_id, foe_id, points, battles, updated_at)
VALUES (?, ?, ?, ?, 1, NOW())
ON DUPLICATE KEY UPDATE
  points = points + VALUES(points),
  battles = battles + VALUES(battles),
  updated_at = VALUES(updated_at)`,
			week, uid, foeID, points)
	}
	h.DB.Exec(`INSERT INTO ezfy_pvp_week (week, user_id, nickname, corps, battles, wins, kills, losses, points, updated_at)
VALUES (?, ?, ?, ?, 1, ?, ?, ?, ?, NOW())
ON DUPLICATE KEY UPDATE
  battles = battles + VALUES(battles),
  wins = wins + VALUES(wins),
  kills = kills + VALUES(kills),
  losses = losses + VALUES(losses),
  points = points + VALUES(points),
  nickname = VALUES(nickname),
  corps = VALUES(corps),
  updated_at = VALUES(updated_at)`,
		week, uid, nick, corps, wins, kills, losses, points)
	// ★ 2026-10-16 「我的战绩」场次明细：独立表，战报清理不影响
	//   ★ 2026-10-26 改走 GORM Create（原来 raw INSERT）→ 拿到明细 id 供回填战报 id。
	d := model.EzfyPvpDetail{
		Week: week, UserID: uid, FoeID: foeID, FoeNick: foeNick, Ts: ts,
		Kills: kills, Losses: losses, Points: credited, IsWin: wins > 0, IsDraw: draw,
	}
	if err := h.DB.Create(&d).Error; err != nil {
		return 0
	}
	return d.ID
}

// ezfyPvpRankOn 风云榜统计开关（读 ezfy_cfg_limit.pvp_rank_on，管理端「排行榜 → 风云榜」可开关）。
//
// ★ 2026-10-14 用户需求「风云榜 tab 能够选择开启关闭，开启就是现在这样，关闭就是不进行统计」。
//
//	· 开（1）= 战斗结算写周榜（现状）；
//	· 关（0）= **完全不做统计**：不写 ezfy_pvp_week / ezfy_pvp_foe，玩家端榜单为空。
//	历史数据保留，重新开启即继续累计（不清历史）。
//	⚠️ 读的是缓存里的 limit，开关是「0 合法」的值 → 必须确认配置已加载（ready()），
//	  否则未加载时会读到零值 0 → 把「开」误判成「关」（/war/status 踩过同款坑）。
func ezfyPvpRankOn() bool {
	if !ezfyCfg.ready() {
		return true // 配置未加载 → 按默认「开」，避免误判成关
	}
	return ezfyCfg.limit.PvpRankOn != 0
}

// ezfyPvpKillCapOn 风云榜「单场灭敌加分上限」开关（读 ezfy_cfg_limit.pvp_kill_cap_on）。
//
// ★ 2026-10-29 用户需求「单灭敌加分上限加个开关，默认开有上限，关掉期间灭敌无上限」。
//
//	· 开（1，默认）= 杀敌分 20×log10(加权灭敌) 最多 ezfyPvpKillPtCap(200) 分；
//	· 关（0）      = 灭敌分不封顶（只受单场规模上限 400/800 约束）。
//	⚠️ 与统计开关同理：0 是合法值，配置未加载时会读到零值 0 → 必须先 ready()，
//	  否则重启后头几次结算会误判成「无上限」。
func ezfyPvpKillCapOn() bool {
	if !ezfyCfg.ready() {
		return true // 配置未加载 → 按默认「有上限」，避免误判成无上限
	}
	return ezfyCfg.limit.PvpKillCapOn != 0
}

// ezfyPvpRankLocked 该周风云榜是否被「锁分」（★ 2026-10-18 用户需求）：
//
//	锁住的周 = 终止当周竞争，战绩不再累计积分（榜单冻结在锁定前）。
//	与统计开关的区别：开关是全局开/关，锁分只冻**指定周**，其他周照常累计。
func ezfyPvpRankLocked(week string) bool {
	if !ezfyCfg.ready() || week == "" {
		return false
	}
	return ezfyCfg.limit.PvpRankLockWeek == week
}

// ezfyPvpRankLockWeek 当前锁定的周（空 = 未锁定），管理端展示用。
//
// ★ 2026-10-26 用户规则「锁分操作只锁当期」：只有存的周**就是本周**才算锁定；
// 跨周后库里仍留着上周的串，但这里返回空 → 管理端不再显示「已锁分」，
// 本周战绩照常累计（等价于「锁分只锁当期、过周自动失效」）。
func ezfyPvpRankLockWeek() string {
	if !ezfyCfg.ready() {
		return ""
	}
	w := ezfyCfg.limit.PvpRankLockWeek
	if w == "" || w != ezfyPvpWeekOf(time.Now()) {
		return ""
	}
	return w
}

// ★ 2026-10-16 榜单「不参与排名」名单按榜单维度隔离（用户需求：
// 「风云榜和战力榜、军衔声望榜都可以单独不参与排名，现在公用一个不对」）。
// 这三个常量同时是 ezfy_pvp_exclude.board 的取值 —— 改动务必同步前端 tab 名。
const (
	ezfyBoardPvp      = "pvp"      // 风云榜
	ezfyBoardPower    = "power"    // 战力榜
	ezfyBoardPrestige = "prestige" // 军衔声望榜
)

// ezfyBoardName 榜单中文名（提示文案用）。
func ezfyBoardName(board string) string {
	switch board {
	case ezfyBoardPvp:
		return "风云榜"
	case ezfyBoardPower:
		return "战力榜"
	case ezfyBoardPrestige:
		return "军衔声望榜"
	}
	return board
}

// pvpExcludedSet 取**指定榜单**的「不参与排名」名单（管理端操作列维护），返回 user_id 集合。
//
// ★ 2026-10-14：榜单查询时把名单内玩家过滤掉（统计照常不入榜 → 「标记排除，不显示在榜」）。
// ★ 2026-10-16：必须带 board —— 三个榜各有一份独立名单，混用会让「只在风云榜排除」的人
//
//	在战力榜/军衔声望榜上也消失（用户反馈的「公用一个不对」）。
func pvpExcludedSet(db *gorm.DB, board string) map[uint]bool {
	var ids []uint
	db.Model(&model.EzfyPvpExclude{}).Where("board = ?", board).Pluck("user_id", &ids)
	if len(ids) == 0 {
		return nil
	}
	set := make(map[uint]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return set
}

// pvpIsExcluded 单个玩家是否在**指定榜单**的「不参与排名」名单里。
func pvpIsExcluded(db *gorm.DB, uid uint, board string) bool {
	if uid == 0 {
		return false
	}
	var n int64
	db.Model(&model.EzfyPvpExclude{}).Where("user_id = ? AND board = ?", uid, board).Count(&n)
	return n > 0
}

// accumPvpWeek 一次玩家对战结算 → 攻守双方各写一条风云榜战绩。
// br 为已定局的战斗结果，win=攻方是否获胜，draw=是否平局。
//
// ★ 2026-10-26 返回 (攻方明细id, 守方明细id)：调用方在战报落库后把战报 id 回填进去
// （管理端「积分风云榜 → 积分明细 → 查看本场战报」）。0 = 没写/被门槛拦下。
func (h *EzfyHandler) accumPvpWeek(now int64, atkUID, defUID uint, br ezfyBattleResult, win, draw bool) (uint, uint) {
	// ★ 2026-10-14 风云榜统计开关关闭 → 完全不做统计（不写榜、不记对手封顶）
	if !ezfyPvpRankOn() {
		return 0, 0
	}
	if atkUID == 0 || defUID == 0 || atkUID == defUID {
		return 0, 0
	}
	week := ezfyPvpWeekOf(time.UnixMilli(now))
	// ★ 2026-10-18 锁分：该周已被锁定 → 终止当周竞争，不再累计积分（榜单冻结在锁定前）
	if ezfyPvpRankLocked(week) {
		return 0, 0
	}
	// ★ 2026-10-16 积分口径（加权）：灭敌/战损/兵力都按「兵种分权重」算（侦察机/卡车不计）——
	//   同数量下高级兵种(航母/战列舰)分远高于步兵，堵「无差别刷步兵」。
	//   展示口径（原始数量）另存，写榜/战报里的灭敌/战损列仍是玩家看到的真实杀敌数。
	atkKillsV := pvpUnitsOf(br.DefenderLosses) // 攻方有效灭敌（加权）
	atkLossesV := pvpUnitsOf(br.AttackerLosses)
	defKillsV := pvpUnitsOf(br.AttackerLosses)
	defLossesV := pvpUnitsOf(br.DefenderLosses)
	atkPower := atkKillsV + pvpUnitsOf(br.AttackerLeft)  // 攻方本场投入（加权）
	defPower := defLossesV + pvpUnitsOf(br.DefenderLeft) // 守方本场投入（加权）
	// 榜单展示列：原始灭敌/战损
	atkKills := enemyDeadOf(br.DefenderLosses) // 攻方灭敌 = 守方阵亡
	atkLosses := enemyDeadOf(br.AttackerLosses)
	defKills := enemyDeadOf(br.AttackerLosses) // 守方灭敌 = 攻方阵亡
	defLosses := enemyDeadOf(br.DefenderLosses)
	// 积分：攻方行动分 20（掠夺/征服），守方 15（防守）；按加权口径计
	atkPts := pvpBattlePoints(atkKillsV, atkLossesV, atkPower, defPower, 20, win, draw)
	defPts := pvpBattlePoints(defKillsV, defLossesV, defPower, atkPower, 15, !win, draw)
	// ★ 防刷：
	// 1) 兵力下限——1 兵碰瓷（无论胜负）**攻守双方**本场都 0 分：
	//    攻方 1 兵送，守方轻易全灭也不给防御分（防「小号送人头给大号刷分」）。
	if atkPower < ezfyPvpMinTroop {
		atkPts = 0
		defPts = 0
	}
	// 2) 防守方自己兵力也少（<1000）→ 守方 0 分（守方同规则，防守方 1 兵挂分）。
	if defPower < ezfyPvpMinTroop {
		defPts = 0
	}
	// 3) 空城刷——守方零兵力（空城）时进攻方不得分（打空城不算战绩）。
	if defPower == 0 {
		atkPts = 0
	}
	// 4) 同对手封顶在 accumPvpWeekOnce 内按 foeID 处理。
	atkP := h.ensureProfile(atkUID)
	defP := h.ensureProfile(defUID)
	// 攻方
	atkDetailID := h.accumPvpWeekOnce(week, atkUID, defUID, atkP.Nickname, defP.Nickname, h.pvpCorpsNameOf(atkUID), now, boolInt64(win), atkKills, atkLosses, atkPts, draw)
	// 守方：攻方败且非平局 = 守方胜；平局双方都不算胜
	defDetailID := h.accumPvpWeekOnce(week, defUID, atkUID, defP.Nickname, atkP.Nickname, h.pvpCorpsNameOf(defUID), now, boolInt64(!win && !draw), defKills, defLosses, defPts, draw)
	return atkDetailID, defDetailID
}

// ezfyPvpAttachReport 把「本场战报 id」回填到风云榜积分明细行。
//
// ★ 2026-10-26 管理端「积分明细 → 查看本场战报」用：明细先写、战报后写，
// 所以结算末尾补一次 UPDATE；detailID / reportID 任一为 0 直接跳过（老数据 / 写战报失败）。
func (h *EzfyHandler) ezfyPvpAttachReport(detailID, reportID uint) {
	if detailID == 0 || reportID == 0 {
		return
	}
	h.DB.Model(&model.EzfyPvpDetail{}).Where("id = ?", detailID).Update("report_id", reportID)
}

func boolInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// pvpWinRate 胜率（%），保留 1 位小数；无场次返回 0。
func pvpWinRate(wins, battles int64) float64 {
	if battles <= 0 {
		return 0
	}
	return float64(100*wins) / float64(battles)
}

// PvpRankDetail GET /games/ezfy/rank/pvp/detail?week=&page=&size= 玩家端「我的战绩」场次明细（分页倒序）。
//
// ★ 2026-10-16：战报会定期清理，明细存独立表 ezfy_pvp_detail，历史照查。
// 返回 list（按时间倒序）：ts/ts_fmt/foe_nick/points(本场实际入账)/kills/losses/is_win/is_draw，
// 外加 total/page/size 供前端分页。
func (h *EzfyHandler) PvpRankDetail(c *gin.Context) {
	h.cfgs()
	uid := middleware.GetUID(c)
	week := c.Query("week")
	if week == "" {
		week = ezfyPvpWeekOf(time.Now())
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 50 {
		size = 20
	}
	q := h.DB.Model(&model.EzfyPvpDetail{}).Where("week = ? AND user_id = ?", week, uid)
	var total int64
	q.Count(&total)
	var rows []model.EzfyPvpDetail
	q.Order("ts DESC, id DESC").Offset((page - 1) * size).Limit(size).Find(&rows)
	list := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		list = append(list, gin.H{
			"ts": r.Ts, "ts_fmt": time.UnixMilli(r.Ts).Format("01-02 15:04"),
			"foe_nick": r.FoeNick, "points": r.Points,
			"kills": r.Kills, "losses": r.Losses,
			"is_win": r.IsWin, "is_draw": r.IsDraw,
		})
	}
	resp.OK(c, gin.H{"list": list, "total": total, "page": page, "size": size})
}

// PvpRank GET /games/ezfy/rank/pvp?week=… 玩家端风云榜（周榜）。
//
// 返回：
//   - week: 生效周串（没传就取本周）；
//   - weeks: 最近 12 个有数据的周（倒序）；
//   - ranks: 本周**积分榜**前 20（rank/nick/corps/points/battles/wins/kills/losses）；
//   - mine: 本人本周战绩（没参战过则空对象）。
func (h *EzfyHandler) PvpRank(c *gin.Context) {
	h.cfgs()
	uid := middleware.GetUID(c)
	week := c.Query("week")
	if week == "" {
		week = ezfyPvpWeekOf(time.Now())
	}
	enabled := ezfyPvpRankOn()
	// ★ 2026-10-14 统计关闭时不展示榜单（前端显示「暂未开启」），但接口仍正常返回不报错。
	if !enabled {
		resp.OK(c, gin.H{"week": week, "weeks": []string{}, "ranks": []gin.H{}, "mine": gin.H{}, "enabled": false})
		return
	}
	// 最近 12 个有数据的周（倒序，周串定长 → 字典序即时间序）
	var weeks []string
	h.DB.Model(&model.EzfyPvpWeek{}).Distinct("week").Order("week DESC").Limit(12).Pluck("week", &weeks)

	// 本周榜单：**积分降序**（同积分按灭敌多、胜场多、战损少排前）
	rows := ezfyPvpRankRows(h.DB, week, 20)
	ranks := make([]gin.H, 0, len(rows))
	for i, r := range rows {
		ranks = append(ranks, gin.H{
			"rank": i + 1, "user_id": r.UserID, "name": r.Nickname, "corps": r.Corps,
			"points": r.Points, "battles": r.Battles, "wins": r.Wins, "kills": r.Kills, "losses": r.Losses,
			"win_rate": pvpWinRate(r.Wins, r.Battles),
		})
	}
	// 本人本周战绩（未上榜也返回，供前端「我的战绩」展示）
	// ★ 2026-10-14 被设为「不参与排名」的玩家不再返回榜单名次，但自己的战绩仍展示（提示已排除）。
	mine := gin.H{}
	if !pvpIsExcluded(h.DB, uid, ezfyBoardPvp) {
		var me model.EzfyPvpWeek
		if err := h.DB.Where("week = ? AND user_id = ?", week, uid).First(&me).Error; err == nil {
			mine = gin.H{
				"user_id": me.UserID, "name": me.Nickname, "corps": me.Corps,
				"points": me.Points, "battles": me.Battles, "wins": me.Wins, "kills": me.Kills, "losses": me.Losses,
				"win_rate": pvpWinRate(me.Wins, me.Battles),
			}
		}
	}
	resp.OK(c, gin.H{"week": week, "weeks": weeks, "ranks": ranks, "mine": mine, "enabled": true,
		// ★ 2026-10-29 玩家端说明文案用：灭敌分是否有上限、单场分数上限、同对手周封顶
		"kill_cap_on": ezfyPvpKillCapOn(), "battle_cap": ezfyPvpBattleCap(), "foe_cap": ezfyPvpFoeCap()})
}

// ezfyPvpRankRows 查询风云榜指定周（管理端/玩家端共用），**按积分降序**。
//
// ★ 2026-10-14：自动跳过「不参与排名」名单里的玩家（管理端操作列维护），
// 名单内玩家即使有战绩也不出现在榜上。limit > 0 时在**过滤后**截断。
// ★ 2026-10-16：只跳过**风云榜那一份**名单（三榜名单互相独立）。
func ezfyPvpRankRows(db *gorm.DB, week string, limit int) []model.EzfyPvpWeek {
	q := db.Where("week = ?", week)
	if ex := pvpExcludedSet(db, ezfyBoardPvp); len(ex) > 0 {
		ids := make([]uint, 0, len(ex))
		for id := range ex {
			ids = append(ids, id)
		}
		q = q.Where("user_id NOT IN ?", ids)
	}
	q = q.Order("points DESC, kills DESC, wins DESC, losses ASC, user_id ASC")
	if limit > 0 {
		// 过滤已在 SQL 层完成 → 直接 LIMIT 即「过滤后的前 N 名」
		q = q.Limit(limit)
	}
	var rows []model.EzfyPvpWeek
	q.Find(&rows)
	return rows
}

// pvpExcludedRows 指定榜单当前的「不参与排名」名单（带昵称，管理端展示用）。
func pvpExcludedRows(db *gorm.DB, board string) []gin.H {
	var rows []model.EzfyPvpExclude
	db.Where("board = ?", board).Order("id DESC").Find(&rows)
	out := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		out = append(out, gin.H{
			"id": r.ID, "user_id": r.UserID, "name": r.Nickname, "reason": r.Reason,
			"board": r.Board, "board_name": ezfyBoardName(r.Board),
			"created_at": r.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return out
}
