// 风云榜（每周玩家对战战绩，★ 2026-10-14）
//
// 按**自然周**（周一 00:00 ~ 周日 23:59）统计**玩家 vs 玩家**战斗：
// 灭敌(击杀敌军数) / 战损(己方阵亡数) / 参战场次 / 胜场 → 排名展示。
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

// accumPvpWeekOnce 把一位玩家的本周战绩累加进风云榜（按 (week,user_id) 唯一键 upsert）。
//   - battles: 本场 +1；wins: 本场胜 1，否则 0（平局双方都不算胜）；
//   - kills: 本场灭敌（击杀敌军数）；losses: 本场战损（己方阵亡数）。
//
// 昵称/军团随写榜时刻刷新（玩家改名/换军团后历史周保留当时名字）。
func (h *EzfyHandler) accumPvpWeekOnce(week string, uid uint, nick, corps string, wins int64, kills, losses int64) {
	if uid == 0 || week == "" {
		return
	}
	h.DB.Exec(`INSERT INTO ezfy_pvp_week (week, user_id, nickname, corps, battles, wins, kills, losses, updated_at)
VALUES (?, ?, ?, ?, 1, ?, ?, ?, NOW())
ON DUPLICATE KEY UPDATE
  battles = battles + VALUES(battles),
  wins = wins + VALUES(wins),
  kills = kills + VALUES(kills),
  losses = losses + VALUES(losses),
  nickname = VALUES(nickname),
  corps = VALUES(corps),
  updated_at = VALUES(updated_at)`,
		week, uid, nick, corps, wins, kills, losses)
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
	atkP := h.ensureProfile(atkUID)
	defP := h.ensureProfile(defUID)
	// 攻方
	h.accumPvpWeekOnce(week, atkUID, atkP.Nickname, h.pvpCorpsNameOf(atkUID), boolInt64(win), atkKills, atkLosses)
	// 守方：攻方败且非平局 = 守方胜；平局双方都不算胜
	h.accumPvpWeekOnce(week, defUID, defP.Nickname, h.pvpCorpsNameOf(defUID), boolInt64(!win && !draw), defKills, defLosses)
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
//   - ranks: 本周灭敌榜前 20（rank/nick/corps/battles/wins/kills/losses）；
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

	// 本周榜单：灭敌降序（同灭敌按胜场多排前，再按战损少排前）
	var rows []model.EzfyPvpWeek
	h.DB.Where("week = ?", week).Order("kills DESC, wins DESC, losses ASC, user_id ASC").
		Limit(20).Find(&rows)
	ranks := make([]gin.H, 0, len(rows))
	for i, r := range rows {
		ranks = append(ranks, gin.H{
			"rank": i + 1, "user_id": r.UserID, "name": r.Nickname, "corps": r.Corps,
			"battles": r.Battles, "wins": r.Wins, "kills": r.Kills, "losses": r.Losses,
			"win_rate": pvpWinRate(r.Wins, r.Battles),
		})
	}
	// 本人本周战绩（未上榜也返回，供前端「我的战绩」展示）
	mine := gin.H{}
	var me model.EzfyPvpWeek
	if err := h.DB.Where("week = ? AND user_id = ?", week, uid).First(&me).Error; err == nil {
		mine = gin.H{
			"user_id": me.UserID, "name": me.Nickname, "corps": me.Corps,
			"battles": me.Battles, "wins": me.Wins, "kills": me.Kills, "losses": me.Losses,
			"win_rate": pvpWinRate(me.Wins, me.Battles),
		}
	}
	resp.OK(c, gin.H{"week": week, "weeks": weeks, "ranks": ranks, "mine": mine})
}

// EzfyPvpRankRows 查询风云榜指定周（管理端/玩家端共用）。
func ezfyPvpRankRows(db *gorm.DB, week string, limit int) []model.EzfyPvpWeek {
	var rows []model.EzfyPvpWeek
	q := db.Where("week = ?", week).Order("kills DESC, wins DESC, losses ASC, user_id ASC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	q.Find(&rows)
	return rows
}