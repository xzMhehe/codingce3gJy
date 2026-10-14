// 风云榜（每周玩家对战战绩，★ 2026-10-14；★ 2026-10-14 按积分排名）
//
// 按**自然周**（周一 00:00 ~ 周日 23:59）统计**玩家 vs 玩家**战斗：
// 灭敌(击杀敌军数) / 战损(己方阵亡数) / 参战场次 / 胜场 / 积分 → 按**积分**排名。
// 只记打玩家城（target_type=3），野地/寇城/活动目标一律不算 PVP。
// 每个玩家每周一行（uk_pvp_week_user），战斗结算时按(week,user_id)累加。
package ezfy

import (
	"math"
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
	// ezfyPvpMinTroop 本场己方投入兵力低于此值 → 本场**不计积分**（1 兵刷无效；攻守双方同规则）。
	// 正常出击/布防都是上万兵，「1 兵碰瓷」直接 0 分。
	ezfyPvpMinTroop = int64(1000)
	// ezfyPvpFoeCap 同一自然周内，与**同一对手**对战的积分累计封顶（防小号互刷）。
	// 达上限后与该对手再交战：场次/灭敌/战损照记，积分不再计入榜单。
	ezfyPvpFoeCap = int64(8000)
)

// pvpUnitsOf 统计 ezfyUnitGroup 总兵力。
func pvpUnitsOf(gs []ezfyUnitGroup) int64 {
	var n int64
	for _, g := range gs {
		n += g.Count
	}
	return n
}

// pvpBattlePoints 单场 PVP 积分（风云榜，★ 2026-10-14 规则；杀敌分同日本轮补充）。
//
//	积分 = (行动分 + 胜负分) × 兵力劣势系数 + 杀敌分 + 战损比分，单场**上限 400**（防刷）
//
//   - 行动分：进攻(掠夺/征服) 20 分，防守 15 分（进攻稍高，鼓励主动出击）
//   - 胜负分：胜 40 / 平 15 / 负 10（参战即有，鼓励对抗）
//   - 兵力劣势系数：己方本场投入兵力 < 对方时，以少打多奖励：
//     系数 = min(3.0, 对方兵力÷己方兵力)；攻方出征少、守方兵力多时攻击方按比例**多拿分**；
//     系数对双方都生效（谁兵力劣势谁加成），≥1 时系数=1
//   - 杀敌分（★ 2026-10-14 用户反馈「大战积分太少」补上）：按**灭敌绝对量**对数缩放，
//     20×log10(灭敌) 封顶 200 —— 千万级大战与百万级小仗拉开差距、大仗有分量；
//     对数压缩 → 灭敌量翻 10 倍只多 20 分，不会被天文数字刷爆。1 兵刷/空城刷由门槛拦住。
//   - 战损比分：按 战损比 = 灭敌÷战损 给分，kdr×10 封顶 30；无损(战损=0)按满 30 给
const (
	// ezfyPvpKillPtCap 杀敌分封顶
	ezfyPvpKillPtCap = int64(200)
	// ezfyPvpMaxPoint 单场上限（防极端刷分）
	ezfyPvpMaxPoint = int64(400)
)

func pvpBattlePoints(kills, losses, myPower, foePower int64, actBase int64, win, draw bool) int64 {
	// 杀敌分：20×log10(灭敌)，0~200（灭敌 0 不给分）
	killPts := int64(0)
	if kills > 0 {
		killPts = int64(20 * math.Log10(float64(kills)))
		if killPts > ezfyPvpKillPtCap {
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
	pts := (actBase+outcome)*mult/100 + killPts + kdrPts
	if pts > ezfyPvpMaxPoint {
		pts = ezfyPvpMaxPoint
	}
	return pts
}

// accumPvpWeekOnce 把一位玩家的本周战绩累加进风云榜（按 (week,user_id) 唯一键 upsert）。
//   - battles: 本场 +1；wins: 本场胜 1，否则 0（平局双方都不算胜）；
//   - kills: 本场灭敌（击杀敌军数）；losses: 本场战损（己方阵亡数）；
//   - points: 本场风云积分（周累计，按积分排名）；foeID 为对手玩家ID，做同对手封顶。
//
// 昵称/军团随写榜时刻刷新（玩家改名/换军团后历史周保留当时名字）。
func (h *EzfyHandler) accumPvpWeekOnce(week string, uid, foeID uint, nick, corps string, wins, kills, losses, points int64) {
	if uid == 0 || week == "" {
		return
	}
	// ★ 防刷：同一周与同一对手对战的积分封顶（小号互刷只能刷到固定上限）。
	//   已计分 ≥ 上限 → 本场积分 0；未达上限 → 最多补到上限。场次/灭敌/战损不受影响。
	if points > 0 && foeID > 0 {
		var cur int64
		h.DB.Model(&model.EzfyPvpFoe{}).
			Where("week = ? AND user_id = ? AND foe_id = ?", week, uid, foeID).
			Select("points").Scan(&cur)
		if cur >= ezfyPvpFoeCap {
			points = 0
		} else if cur+points > ezfyPvpFoeCap {
			points = ezfyPvpFoeCap - cur
		}
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
}

// ezfyPvpRankOn 风云榜统计开关（读 ezfy_cfg_limit.pvp_rank_on，管理端「排行榜 → 风云榜」可开关）。
//
// ★ 2026-10-14 用户需求「风云榜 tab 能够选择开启关闭，开启就是现在这样，关闭就是不进行统计」。
//   · 开（1）= 战斗结算写周榜（现状）；
//   · 关（0）= **完全不做统计**：不写 ezfy_pvp_week / ezfy_pvp_foe，玩家端榜单为空。
//   历史数据保留，重新开启即继续累计（不清历史）。
//   ⚠️ 读的是缓存里的 limit，开关是「0 合法」的值 → 必须确认配置已加载（ready()），
//     否则未加载时会读到零值 0 → 把「开」误判成「关」（/war/status 踩过同款坑）。
func ezfyPvpRankOn() bool {
	if !ezfyCfg.ready() {
		return true // 配置未加载 → 按默认「开」，避免误判成关
	}
	return ezfyCfg.limit.PvpRankOn != 0
}

// pvpExcludedSet 取「不参与排名」名单（管理端操作列维护），返回 user_id 集合。
//
// ★ 2026-10-14：榜单查询时把名单内玩家过滤掉（统计照常不入榜 → 「标记排除，不显示在榜」）。
func pvpExcludedSet(db *gorm.DB) map[uint]bool {
	var ids []uint
	db.Model(&model.EzfyPvpExclude{}).Pluck("user_id", &ids)
	if len(ids) == 0 {
		return nil
	}
	set := make(map[uint]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return set
}

// pvpIsExcluded 单个玩家是否在「不参与排名」名单里。
func pvpIsExcluded(db *gorm.DB, uid uint) bool {
	if uid == 0 {
		return false
	}
	var n int64
	db.Model(&model.EzfyPvpExclude{}).Where("user_id = ?", uid).Count(&n)
	return n > 0
}

// accumPvpWeek 一次玩家对战结算 → 攻守双方各写一条风云榜战绩。
// br 为已定局的战斗结果，win=攻方是否获胜，draw=是否平局。
func (h *EzfyHandler) accumPvpWeek(now int64, atkUID, defUID uint, br ezfyBattleResult, win, draw bool) {
	// ★ 2026-10-14 风云榜统计开关关闭 → 完全不做统计（不写榜、不记对手封顶）
	if !ezfyPvpRankOn() {
		return
	}
	if atkUID == 0 || defUID == 0 || atkUID == defUID {
		return
	}
	week := ezfyPvpWeekOf(time.UnixMilli(now))
	atkKills := enemyDeadOf(br.DefenderLosses) // 攻方灭敌 = 守方阵亡
	atkLosses := enemyDeadOf(br.AttackerLosses)
	defKills := enemyDeadOf(br.AttackerLosses) // 守方灭敌 = 攻方阵亡
	defLosses := enemyDeadOf(br.DefenderLosses)
	atkPower := pvpUnitsOf(br.AttackerLosses) + pvpUnitsOf(br.AttackerLeft) // 攻方本场投入
	defPower := pvpUnitsOf(br.DefenderLosses) + pvpUnitsOf(br.DefenderLeft) // 守方本场投入
	// 积分：攻方行动分 20（掠夺/征服），守方 15（防守）
	atkPts := pvpBattlePoints(atkKills, atkLosses, atkPower, defPower, 20, win, draw)
	defPts := pvpBattlePoints(defKills, defLosses, defPower, atkPower, 15, !win, draw)
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
	h.accumPvpWeekOnce(week, atkUID, defUID, atkP.Nickname, h.pvpCorpsNameOf(atkUID), boolInt64(win), atkKills, atkLosses, atkPts)
	// 守方：攻方败且非平局 = 守方胜；平局双方都不算胜
	h.accumPvpWeekOnce(week, defUID, atkUID, defP.Nickname, h.pvpCorpsNameOf(defUID), boolInt64(!win && !draw), defKills, defLosses, defPts)
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
	if !pvpIsExcluded(h.DB, uid) {
		var me model.EzfyPvpWeek
		if err := h.DB.Where("week = ? AND user_id = ?", week, uid).First(&me).Error; err == nil {
			mine = gin.H{
				"user_id": me.UserID, "name": me.Nickname, "corps": me.Corps,
				"points": me.Points, "battles": me.Battles, "wins": me.Wins, "kills": me.Kills, "losses": me.Losses,
				"win_rate": pvpWinRate(me.Wins, me.Battles),
			}
		}
	}
	resp.OK(c, gin.H{"week": week, "weeks": weeks, "ranks": ranks, "mine": mine, "enabled": true})
}

// ezfyPvpRankRows 查询风云榜指定周（管理端/玩家端共用），**按积分降序**。
//
// ★ 2026-10-14：自动跳过「不参与排名」名单里的玩家（管理端操作列维护），
// 名单内玩家即使有战绩也不出现在榜上。limit > 0 时在**过滤后**截断。
func ezfyPvpRankRows(db *gorm.DB, week string, limit int) []model.EzfyPvpWeek {
	q := db.Where("week = ?", week)
	if ex := pvpExcludedSet(db); len(ex) > 0 {
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

// pvpExcludedRows 当前「不参与排名」名单（带昵称，管理端展示用）。
func pvpExcludedRows(db *gorm.DB) []gin.H {
	var rows []model.EzfyPvpExclude
	db.Order("id DESC").Find(&rows)
	out := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		out = append(out, gin.H{
			"id": r.ID, "user_id": r.UserID, "name": r.Nickname, "reason": r.Reason,
			"created_at": r.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return out
}