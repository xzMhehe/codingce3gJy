// 统帅称号（★ 2026-10-17）
//
// 用户需求（原话）：「二战管理端加个模块 统帅称号，称号管理员可添加、称号名称、称号类型、
// 称号有效期、称号创建日期、已有称号玩家。我的灵感来自率土之滨的三英战吕布 天下无双 称号（有效期一天）。
// 现在咱们不是有风云榜了 每天的第一可以也给个称号 天下无双。
// 放到 games/ezfy?cur=playerinfo 页面 阵营上面加个……称号样式好看些 简约好看 能激起玩家的战胜欲」。
//
// 本文件负责**玩家端读取 + 发放 + 风云榜每日第一自动结算**；
// 管理端的增删改查见 `ezfy_admin_title.go`。
//
// 三个口径必须一致，改任何一处都要同步：
//  1. 「生效中」= 称号启用(status=1) 且（expire_at=0 永久 或 expire_at > now）；
//  2. 授予 = 同 (title_id,user_id) 唯一行 upsert，**续期取较晚**，永久不被缩短；
//  3. 每日冠军 = 风云榜「我的战绩」明细（ezfy_pvp_detail）按天聚合积分第一，
//     与周榜同口径（points 是本场实际入账分，已含门槛/封顶/同对手上限）。
package ezfy

import (
	"time"

	"gorm.io/gorm"

	"qqjiayuan/server/internal/model"
)

// 称号类型（与前端徽章配色一一对应，改这里要同步 Ezfy.vue 的 .ezfy-title-t1~t4）
const (
	ezfyTitleTypeMerit = 1 // 战功（赤金）
	ezfyTitleTypeRank  = 2 // 榜单（紫金）——「天下无双」属这类
	ezfyTitleTypeEvent = 3 // 活动（靛青）
	ezfyTitleTypeHonor = 4 // 荣誉（银灰）
)

// ezfyTitleTypeName 称号类型中文名（管理端/前端展示用）。
func ezfyTitleTypeName(t int) string {
	switch t {
	case ezfyTitleTypeMerit:
		return "战功"
	case ezfyTitleTypeRank:
		return "榜单"
	case ezfyTitleTypeEvent:
		return "活动"
	case ezfyTitleTypeHonor:
		return "荣誉"
	}
	return "其他"
}

// ezfyTitleTypes 称号类型下拉选项（管理端新增/编辑用）。
func ezfyTitleTypes() []map[string]interface{} {
	return []map[string]interface{}{
		{"value": ezfyTitleTypeMerit, "label": ezfyTitleTypeName(ezfyTitleTypeMerit)},
		{"value": ezfyTitleTypeRank, "label": ezfyTitleTypeName(ezfyTitleTypeRank)},
		{"value": ezfyTitleTypeEvent, "label": ezfyTitleTypeName(ezfyTitleTypeEvent)},
		{"value": ezfyTitleTypeHonor, "label": ezfyTitleTypeName(ezfyTitleTypeHonor)},
	}
}

// ezfyTitleBadgeRow 玩家端徽章行（一次 JOIN 取齐，避免 N+1）。
type ezfyTitleBadgeRow struct {
	ID       uint   `gorm:"column:id"`
	TitleId  uint   `gorm:"column:title_id"`
	Name     string `gorm:"column:name"`
	Type     int    `gorm:"column:type"`
	ExpireAt int64  `gorm:"column:expire_at"`
	Source   string `gorm:"column:source"`
}

// ezfyTitlesOf 某玩家**生效中**的称号（按类型、到期先后排序，最多 6 个防刷屏）。
//
// 返回字段：id/name/type/type_name/expire_at/expire_fmt/permanent/source。
// 前端直接用 type 选配色类（.ezfy-title-t1~t4），不需要再判空。
func ezfyTitlesOf(db *gorm.DB, uid uint) []map[string]interface{} {
	out := []map[string]interface{}{}
	if uid == 0 {
		return out
	}
	var rows []ezfyTitleBadgeRow
	db.Table("ezfy_title_grant AS g").
		Select("g.id AS id, g.title_id AS title_id, t.name AS name, t.type AS type, "+
			"g.expire_at AS expire_at, g.source AS source").
		Joins("JOIN ezfy_title AS t ON t.id = g.title_id").
		Where("g.user_id = ? AND t.status = 1 AND (g.expire_at = 0 OR g.expire_at > ?)",
			uid, time.Now().UnixMilli()).
		// 永久(expire_at=0) 排最前，其次按剩余时间短的排前（快过期的靠前，玩家更在意）
		Order("t.type ASC, g.expire_at ASC, g.id ASC").
		Limit(6).
		Scan(&rows)
	for _, r := range rows {
		permanent := r.ExpireAt == 0
		expireFmt := ""
		if !permanent {
			expireFmt = time.UnixMilli(r.ExpireAt).Format("2006-01-02 15:04")
		}
		out = append(out, map[string]interface{}{
			"id":         r.ID,
			"title_id":   r.TitleId,
			"name":       r.Name,
			"type":       r.Type,
			"type_name":  ezfyTitleTypeName(r.Type),
			"expire_at":  r.ExpireAt,
			"expire_fmt": expireFmt,
			"permanent":  permanent,
			"source":     r.Source,
		})
	}
	return out
}

// ezfyTitleExpireAt 按有效期算到期时间戳（ms）；days<=0 返回 0 = 永久。
func ezfyTitleExpireAt(days int, from time.Time) int64 {
	if days <= 0 {
		return 0
	}
	return from.AddDate(0, 0, days).UnixMilli()
}

// ezfyGrantTitle 授予称号（(title_id,user_id) 唯一行 upsert）。
//
// 续期规则（★ 只延长不缩短）：
//   - 已有行是**永久**（expire_at=0）→ 保持永久，只刷新昵称/来源；
//   - 新授予是永久 → 直接改成永久；
//   - 否则到期时间取较晚的那个。
//
// 这样「管理端补发 30 天」不会把玩家已拿到的 100 天砍成 30 天，反之亦然。
func ezfyGrantTitle(db *gorm.DB, titleID, uid uint, nick, source string, expireAt int64) error {
	var g model.EzfyTitleGrant
	err := db.Where("title_id = ? AND user_id = ?", titleID, uid).First(&g).Error
	if err == nil {
		up := map[string]interface{}{"source": source}
		if nick != "" {
			up["nickname"] = nick
		}
		switch {
		case g.ExpireAt == 0:
			// 已是永久 → 不动到期时间
		case expireAt == 0:
			up["expire_at"] = int64(0)
		case expireAt > g.ExpireAt:
			up["expire_at"] = expireAt
		}
		return db.Model(&model.EzfyTitleGrant{}).Where("id = ?", g.ID).Updates(up).Error
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}
	return db.Create(&model.EzfyTitleGrant{
		TitleId: titleID, UserID: uid, Nickname: nick,
		Source: source, ExpireAt: expireAt,
	}).Error
}

// ============ 风云榜「每日第一」自动发放「天下无双」============

// ezfyDailyTitleName 风云榜每日第一自动获得的称号名（管理端可改有效期，但名字是约定）。
const ezfyDailyTitleName = "天下无双"

// ezfyTitleEnsureDaily 取（不存在则建）「天下无双」称号定义。
//
// 有效期默认 1 天（用户要求）；管理员事后在管理端改 valid_days 即生效（这里只补建、不改已有行）。
func ezfyTitleEnsureDaily(db *gorm.DB) (model.EzfyTitle, bool) {
	var t model.EzfyTitle
	if err := db.Where("name = ?", ezfyDailyTitleName).First(&t).Error; err == nil {
		return t, t.Status == 1
	}
	t = model.EzfyTitle{
		Name:      ezfyDailyTitleName,
		Type:      ezfyTitleTypeRank,
		ValidDays: 1,
		Remark:    "风云榜每日积分第一自动获得",
		Status:    1,
	}
	if err := db.Create(&t).Error; err != nil {
		// 并发下另一个实例先建了 → 回读一次
		if e2 := db.Where("name = ?", ezfyDailyTitleName).First(&t).Error; e2 == nil {
			return t, t.Status == 1
		}
		return t, false
	}
	return t, true
}

// ezfyDailyChampion 取指定自然日（本地时区）风云榜积分第一的玩家。
//
// 口径：ezfy_pvp_detail 按天聚合 SUM(points)，积分必须 > 0；
// 自动跳过「风云榜不参与排名」名单（与周榜一致，否则被排除的人照样拿称号）。
// 并列时按「灭敌多、场次少」排（同分但打得更狠的优先），再按 user_id 兜底保证确定性。
func ezfyDailyChampion(db *gorm.DB, dayStart, dayEnd time.Time) (uint, string, int64) {
	type row struct {
		UserID uint
		Points int64
	}
	var r row
	q := db.Table("ezfy_pvp_detail").
		Select("user_id, SUM(points) AS points").
		Where("ts >= ? AND ts < ?", dayStart.UnixMilli(), dayEnd.UnixMilli()).
		Group("user_id").
		Having("SUM(points) > 0")
	if ex := pvpExcludedSet(db, ezfyBoardPvp); len(ex) > 0 {
		ids := make([]uint, 0, len(ex))
		for id := range ex {
			ids = append(ids, id)
		}
		q = q.Where("user_id NOT IN ?", ids)
	}
	q.Order("points DESC, user_id ASC").Limit(1).Scan(&r)
	if r.UserID == 0 {
		return 0, "", 0
	}
	// 昵称：二战档案优先，家园昵称兜底
	var p model.EzfyProfile
	db.Select("nickname").Where("user_id = ?", r.UserID).First(&p)
	return r.UserID, p.Nickname, r.Points
}

// ezfySettleDailyTitle 结算「指定自然日」的风云榜日冠军 → 发放称号。
//
// 幂等：先在 ezfy_title_daily 抢一条 day 记录（INSERT IGNORE，day 唯一），
// 抢到才真的发；抢不到说明别的实例/上一次 tick 已经处理过。
// 当天无冠军也落一条 UserID=0 的记录，避免每轮 tick 反复重算。
func (h *EzfyHandler) ezfySettleDailyTitle(day time.Time) {
	dayKey := day.Format("20060102")
	// 抢锁（INSERT IGNORE：唯一键冲突时 RowsAffected=0）
	res := h.DB.Exec("INSERT IGNORE INTO ezfy_title_daily (day, user_id, nickname, points, title_id, created_at) "+
		"VALUES (?, 0, '', 0, 0, NOW())", dayKey)
	if res.Error != nil || res.RowsAffected == 0 {
		return
	}
	// 称号被停用 / 建不出来 → 当天不发放（记录已落，不会再试）
	t, enabled := ezfyTitleEnsureDaily(h.DB)
	if !enabled {
		return
	}
	dayStart := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
	dayEnd := dayStart.AddDate(0, 0, 1)
	uid, nick, pts := ezfyDailyChampion(h.DB, dayStart, dayEnd)
	if uid == 0 {
		return
	}
	if nick == "" {
		nick = ezfyNickByUid(h.DB, h.HomeDB, uid)
	}
	expireAt := ezfyTitleExpireAt(t.ValidDays, time.Now())
	if err := ezfyGrantTitle(h.DB, t.ID, uid, nick, "pvp_daily", expireAt); err != nil {
		// 发放失败 → 把抢到的锁行删掉，下一轮 tick 重试（否则这一天永久漏发）
		h.DB.Exec("DELETE FROM ezfy_title_daily WHERE day = ? AND user_id = 0", dayKey)
		return
	}
	h.DB.Model(&model.EzfyTitleDaily{}).Where("day = ?", dayKey).Updates(map[string]interface{}{
		"user_id": uid, "nickname": nick, "points": pts, "title_id": t.ID,
	})
}

// ezfyNickByUid 取玩家展示昵称（二战档案优先，家园昵称兜底）。
//
// db = 二战库、home = 家园库；home 为空（单库模式）时回落 db。
func ezfyNickByUid(db, home *gorm.DB, uid uint) string {
	if db == nil || uid == 0 {
		return ""
	}
	var p model.EzfyProfile
	db.Select("nickname").Where("user_id = ?", uid).First(&p)
	if p.Nickname != "" {
		return p.Nickname
	}
	if home == nil {
		home = db
	}
	var u model.User
	home.Select("nickname").First(&u, uid)
	return u.Nickname
}

// BgTickTitles 后台定时器：结算「昨天」的风云榜日冠军并发放称号（常驻 goroutine）。
//
// 为什么每 5 分钟跑一次而不是掐着 00:00：
//   - 不依赖进程正好在午夜活着（重启/维护窗口都不会漏），反正 day 唯一 + INSERT IGNORE 幂等；
//   - 5 分钟一次的单条 SQL 开销可忽略（ts 有索引）。
//
// 启动时先跑一次，补上「服务停了一整天」的情况（只补昨天，不回溯更早——更早的称号已无意义）。
func (h *EzfyHandler) BgTickTitles() {
	h.ezfySettleDailyTitle(time.Now().AddDate(0, 0, -1))
	for range time.Tick(5 * time.Minute) {
		h.ezfySettleDailyTitle(time.Now().AddDate(0, 0, -1))
	}
}
