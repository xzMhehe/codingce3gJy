package ezfy

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// AdminEzfyMonitor 二战「活动监测」：统计最近时间段内玩家的
// 造兵次数 / 征服次数 / 掠夺次数 / 召集人口次数，并按玩家排行。
//
// 参数：
//
//	start / end  : 时间范围，支持 "2006-01-02" 或毫秒时间戳（都空=今日）
//	page / size  : 分页（pageOf）
//	word         : 账号/昵称模糊（纯数字=uid/账号名，先经家园库反查）
//	sort         : train|conquer|raid|convene|total（默认 total 降序）
//
// 返回：
//
//	totals: { train, conquer, raid, convene, players }（窗口内全库计数，不分组）
//	list:   [{ user_id, player_name, account, train, conquer, raid, convene, total }]
//
// ★ 性能：四类聚合全部下推数据库（GROUP BY），内存只收「每用户一行」的聚合结果；
//
//	名字用 ezfyAdminNamesBatch 批量补齐（2 条 SQL，无 N+1）；word 过滤先进 SQL 窄化。
func (h *EzfyAdmin) AdminEzfyMonitor(c *gin.Context) {
	page, offset, size := pageOf(c, 15)
	word := strings.TrimSpace(c.Query("word"))
	sortKey := strings.TrimSpace(c.Query("sort"))
	if sortKey == "" {
		sortKey = "total"
	}
	switch sortKey {
	case "train", "conquer", "raid", "convene", "total":
	default:
		sortKey = "total"
	}

	start, end, ok := monitorRange(c.Query("start"), c.Query("end"))
	if !ok {
		resp.ParamError(c, "时间范围错误")
		return
	}

	// word → 玩家 uid 集合（双库口径，与 AdminEzfyReports 一致）
	uids := monitorWordUids(h, word)

	// ---------- 四类聚合（各一条 GROUP BY SQL，全部下推） ----------

	// 征服(3) + 掠夺(2)：一条 SQL 出两个指标
	type orderAgg struct {
		UserID    uint  `gorm:"column:user_id"`
		OrderType int   `gorm:"column:order_type"`
		Cnt       int64 `gorm:"column:cnt"`
	}
	oq := h.DB.Model(&model.EzfyOrder{}).
		Select("user_id, order_type, COUNT(*) AS cnt").
		Where("order_type IN ?", []int{2, 3}).
		Where("created_at >= ? AND created_at < ?", start, end)
	if len(uids) > 0 {
		oq = oq.Where("user_id IN ?", uids)
	}
	var orders []orderAgg
	oq.Group("user_id, order_type").Scan(&orders)

	// 造兵：train_queue 无 user_id → JOIN 城市反查玩家（走 idx_city）
	type trainAgg struct {
		UserID uint  `gorm:"column:user_id"`
		Cnt    int64 `gorm:"column:cnt"`
	}
	tq := h.DB.Table("ezfy_train_queue t").
		Select("c.user_id AS user_id, COUNT(*) AS cnt").
		Joins("JOIN ezfy_city c ON c.id = t.city_id").
		Where("t.start_time >= ? AND t.start_time < ?", start.UnixMilli(), end.UnixMilli())
	if len(uids) > 0 {
		tq = tq.Where("c.user_id IN ?", uids)
	}
	var trains []trainAgg
	tq.Group("c.user_id").Scan(&trains)

	// 召集：新埋点表 ezfy_convene_log
	cq := h.DB.Model(&model.EzfyConveneLog{}).
		Select("user_id, COUNT(*) AS cnt").
		Where("created_at >= ? AND created_at < ?", start, end)
	if len(uids) > 0 {
		cq = cq.Where("user_id IN ?", uids)
	}
	var convenes []trainAgg
	cq.Group("user_id").Scan(&convenes)

	// ---------- 内存合并 + 排序 + 分页 ----------
	type monRow struct {
		UserID         uint
		Train, Conquer int64
		Raid, Convene  int64
	}
	m := map[uint]*monRow{}
	for _, o := range orders {
		r := m[o.UserID]
		if r == nil {
			r = &monRow{UserID: o.UserID}
			m[o.UserID] = r
		}
		if o.OrderType == 2 {
			r.Raid = o.Cnt
		} else {
			r.Conquer = o.Cnt
		}
	}
	for _, t := range trains {
		r := m[t.UserID]
		if r == nil {
			r = &monRow{UserID: t.UserID}
			m[t.UserID] = r
		}
		r.Train = t.Cnt
	}
	for _, v := range convenes {
		r := m[v.UserID]
		if r == nil {
			r = &monRow{UserID: v.UserID}
			m[v.UserID] = r
		}
		r.Convene = v.Cnt
	}
	rows := make([]*monRow, 0, len(m))
	for _, r := range m {
		rows = append(rows, r)
	}
	// 排序：sort 字段降序，tie-break user_id 升序（稳定）
	keyOf := func(r *monRow) int64 {
		switch sortKey {
		case "train":
			return r.Train
		case "conquer":
			return r.Conquer
		case "raid":
			return r.Raid
		case "convene":
			return r.Convene
		default:
			return r.Train + r.Conquer + r.Raid + r.Convene
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if keyOf(rows[i]) != keyOf(rows[j]) {
			return keyOf(rows[i]) > keyOf(rows[j])
		}
		return rows[i].UserID < rows[j].UserID
	})

	// ---------- 窗口内全库计数（totals，不分组） ----------
	var totalTrain, totalConquer, totalRaid, totalConvene int64
	h.DB.Table("ezfy_train_queue t").
		Joins("JOIN ezfy_city c ON c.id = t.city_id").
		Where("t.start_time >= ? AND t.start_time < ?", start.UnixMilli(), end.UnixMilli()).
		Count(&totalTrain)
	h.DB.Model(&model.EzfyOrder{}).Where("order_type = ?", 3).
		Where("created_at >= ? AND created_at < ?", start, end).Count(&totalConquer)
	h.DB.Model(&model.EzfyOrder{}).Where("order_type = ?", 2).
		Where("created_at >= ? AND created_at < ?", start, end).Count(&totalRaid)
	h.DB.Model(&model.EzfyConveneLog{}).
		Where("created_at >= ? AND created_at < ?", start, end).Count(&totalConvene)

	// ---------- 分页切片 + 名字补齐 ----------
	pageUids := make([]uint, 0, size)
	for i := offset; i < len(rows) && i < offset+size; i++ {
		pageUids = append(pageUids, rows[i].UserID)
	}
	nick, num := h.ezfyAdminNamesBatch(pageUids)
	out := make([]gin.H, 0, len(pageUids))
	for i := offset; i < len(rows) && i < offset+size; i++ {
		r := rows[i]
		total := r.Train + r.Conquer + r.Raid + r.Convene
		out = append(out, gin.H{
			"user_id":     r.UserID,
			"player_name": nick[r.UserID],
			"account":     num[r.UserID],
			"train":       r.Train,
			"conquer":     r.Conquer,
			"raid":        r.Raid,
			"convene":     r.Convene,
			"total":       total,
		})
	}
	resp.OK(c, gin.H{
		"totals": gin.H{
			"train": totalTrain, "conquer": totalConquer,
			"raid": totalRaid, "convene": totalConvene,
			"players": len(rows),
		},
		"list": out, "page": page, "size": size,
	})
}

// monitorRange 解析时间范围（"2006-01-02" 或毫秒时间戳）。
// end 按「含整天末」取排他（< end+1day）；两者都空 → 今日。
func monitorRange(startStr, endStr string) (start, end time.Time, ok bool) {
	loc := time.Local
	parse := func(s string, isEnd bool) (time.Time, bool) {
		s = strings.TrimSpace(s)
		if s == "" {
			return time.Time{}, false
		}
		if _, err := strconv.ParseInt(s, 10, 64); err == nil {
			ms, _ := strconv.ParseInt(s, 10, 64)
			return time.UnixMilli(ms), true
		}
		if t, err := time.ParseInLocation("2006-01-02", s, loc); err == nil {
			if isEnd {
				return t.AddDate(0, 0, 1), true
			}
			return t, true
		}
		return time.Time{}, false
	}
	now := time.Now()
	if startStr == "" && endStr == "" {
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
		end = start.AddDate(0, 0, 1)
		return start, end, true
	}
	if startStr == "" || endStr == "" {
		return time.Time{}, time.Time{}, false
	}
	var ok1, ok2 bool
	start, ok1 = parse(startStr, false)
	end, ok2 = parse(endStr, true)
	if !ok1 || !ok2 || !end.After(start) {
		return time.Time{}, time.Time{}, false
	}
	return start, end, true
}

// monitorWordUids 解析 word → 玩家 uid 集合（空 word 返回 nil = 不过滤）。
// 纯数字：先按账号名反查 uid，再合并 user_id/game_uid/账号名三口径；
// 非数字：昵称 LIKE + 账号 LIKE 双库合并。
func monitorWordUids(h *EzfyAdmin, word string) []uint {
	if word == "" {
		return nil
	}
	if uid, err := strconv.Atoi(word); err == nil {
		byName := []uint{}
		var uu model.User
		if e := h.home().Select("id").Where("username = ?", word).First(&uu).Error; e == nil {
			byName = append(byName, uu.ID)
		}
		ids := []uint{}
		h.DB.Model(&model.EzfyProfile{}).Select("user_id").
			Where("user_id = ? OR game_uid = ?", uid, uid).Scan(&ids)
		return append(ids, byName...)
	}
	var ids []uint
	h.DB.Model(&model.EzfyProfile{}).Select("user_id").
		Where("nickname LIKE ?", "%"+word+"%").Scan(&ids)
	var uids []uint
	h.home().Model(&model.User{}).Select("id").
		Where("username LIKE ?", "%"+word+"%").Scan(&uids)
	return append(ids, uids...)
}
