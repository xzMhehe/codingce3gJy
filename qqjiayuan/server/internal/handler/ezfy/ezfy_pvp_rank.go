// 风云榜（每周玩家对战战绩，★ 2026-10-14；★ 2026-10-14 按积分排名）
//
// 按**自然周**（周一 00:00 ~ 周日 23:59）统计**玩家 vs 玩家**战斗：
// 灭敌(击杀敌军数) / 战损(己方阵亡数) / 参战场次 / 胜场 / 积分 → 按**积分**排名。
// 只记打玩家城（target_type=3），野地/寇城/活动目标一律不算 PVP。
// 每个玩家每周一行（uk_pvp_week_user），战斗结算时按(week,user_id)累加。
package ezfy

import (
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

// pvpUnitsOf 统计 ezfyUnitGroup 总兵力。
func pvpUnitsOf(gs []ezfyUnitGroup) int64 {
	var n int64
	for _, g := range gs {
		n += g.Count
	}
	return n
}

// pvpBattlePoints 单场 PVP 积分（风云榜，★ 2026-10-14 积分规则设计）。
//
//	积分 = (行动分 + 胜负分) × 兵力劣势系数 + 战损比分，单场**上限 300**（防刷）
//
//   - 行动分：进攻(掠夺/征服) 20 分，防守 15 分（进攻稍高，鼓励主动出击）
//   - 胜负分：胜 40 / 平 15 / 负 10（参战即有，鼓励对抗）
//   - 战损比分：按 战损比 = 灭敌÷战损 给分，kdr×10 封顶 30；无损(战损=0)按满 30 给
//   - 兵力劣势系数：己方本场投入兵力 < 对方时，以少打多奖励：
//     系数 = min(3.0, 对方兵力÷己方兵力)；攻方出征少、守方兵力多时攻击方按比例**多拿分**；
//     系数对双方都生效（谁兵力劣势谁加成），≥1 时系数=1
func pvpBattlePoints(kills, losses, myPower, foePower int64, actBase int64, win, draw bool) int64 {
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
	pts := (actBase+outcome)*mult/100 + kdrPts
	if pts > 300 {
		pts = 300
	}
	return pts
}

// accumPvpWeekOnce 把一位玩家的本周战绩累加进风云榜（按 (week,user_id) 唯一键 upsert）。
//   - battles: 本场 +1；wins: 本场胜 1，否则 0（平局双方都不算胜）；
//   - kills: 本场灭敌（击杀敌军数）；losses: 本场战损（己方阵亡数）；
//   - points: 本场风云积分（周累计，按积分排名）。
//
// 昵称/军团随写榜时刻刷新（玩家改名/换军团后历史周保留当时名字）。
func (h *EzfyHandler) accumPvpWeekOnce(week string, uid uint, nick, corps string, wins, kills, losses, points int64) {
	if uid == 0 || week == "" {
		return
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

// accumPvpWeek 一次玩家对战结算 → 攻守双方各写一条风云榜战绩。
// br 为已定局的战斗结果，win=攻方是否获胜，draw=是否平局。
func (h *EzfyHandler) accumPvpWeek(now int64, atkUID, defUID uint, br ezfyBattleResult, win, draw bool) {
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
	atkP := h.ensureProfile(atkUID)
	defP := h.ensureProfile(defUID)
	// 攻方
	h.accumPvpWeekOnce(week, atkUID, atkP.Nickname, h.pvpCorpsNameOf(atkUID), boolInt64(win), atkKills, atkLosses, atkPts)
	// 守方：攻方败且非平局 = 守方胜；平局双方都不算胜
	h.accumPvpWeekOnce(week, defUID, defP.Nickname, h.pvpCorpsNameOf(defUID), boolInt64(!win && !draw), defKills, defLosses, defPts)
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
	mine := gin.H{}
	var me model.EzfyPvpWeek
	if err := h.DB.Where("week = ? AND user_id = ?", week, uid).First(&me).Error; err == nil {
		mine = gin.H{
			"user_id": me.UserID, "name": me.Nickname, "corps": me.Corps,
			"points": me.Points, "battles": me.Battles, "wins": me.Wins, "kills": me.Kills, "losses": me.Losses,
			"win_rate": pvpWinRate(me.Wins, me.Battles),
		}
	}
	resp.OK(c, gin.H{"week": week, "weeks": weeks, "ranks": ranks, "mine": mine})
}

// EzfyPvpRankRows 查询风云榜指定周（管理端/玩家端共用），**按积分降序**。
func ezfyPvpRankRows(db *gorm.DB, week string, limit int) []model.EzfyPvpWeek {
	var rows []model.EzfyPvpWeek
	q := db.Where("week = ?", week).
		Order("points DESC, kills DESC, wins DESC, losses ASC, user_id ASC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	q.Find(&rows)
	return rows
}