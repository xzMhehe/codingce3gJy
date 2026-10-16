package ezfy

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// ============ 管理端 · 免战模块 ============
//
// ★ 2026-10-12 新增（用户：「二战征途管理端加个免战模块，能管理免战用户数据，比如开始/结束免战，
//   且管理端不受冷却限制」）。
//
// 免战的底层机制与玩家端完全一致：`ezfy_city_effect` 表里 `effect_type=2`（免战保护令）的行，
// 行上有 `until_time` 到期时间戳、作用在某座城；玩家本人还有 `ezfy_profile.peace_cool_until`
// 使用冷却（时长 = ezfyPeaceCoolH，2026-10-16 起 8 小时）。管理端只管**数据本身**：
//
//	· 开始免战 → 给该玩家**全部城池**打上 effect_type=2（until = now + 小时数），
//	  直接写 `until_time`（覆盖旧的），**不受玩家的 peace_cool_until 冷却限制**；
//	· 结束免战 → 删掉该玩家全部城池的 effect_type=2 行。
//
// 权限：module:ezfyPeace（独立菜单「免战管理」）。

// AdminEzfyPeace GET /admin/ezfy-peace —— 玩家免战列表
//
// 参数：page / size / word（昵称·家园号·用户ID·游戏ID）/ only（1=只看当前有免战保护的在免战玩家）。
//
//	每行返回：基础档案 + 是否在免战 + 生效中的免战截止时间（取名下所有城的最新 until，0=无）+ 在免战的城池数。
func (h *EzfyAdmin) AdminEzfyPeace(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	word := strings.TrimSpace(c.Query("word"))
	only := strings.TrimSpace(c.DefaultQuery("only", "0")) == "1"
	now := time.Now().UnixMilli()

	q := h.DB.Model(&model.EzfyProfile{})
	// only=1：只看当前有免战保护的玩家（有任一城 effect_type=2 且未过期）
	if only {
		q = q.Where(`user_id IN (
			SELECT c.user_id FROM ezfy_city_effect e JOIN ezfy_city c ON c.id = e.city_id
			WHERE e.effect_type = 2 AND e.until_time > ?)`, now)
	}
	// word：昵称 / 用户ID / 游戏号
	//
	// ★★ 2026-10-15 用户规则：家园号条件查询 → 游戏号（不再按家园库 users.username 反查）。
	// ★ 顺带修一个**条件串味**的老 bug：原来用 `q.Where(...)` + 多次 `q.Or(...)` 拼，
	//   而上面 only=1 时已经有一个 `user_id IN (...)` 条件 —— SQL 里 AND 优先于 OR，
	//   拼出来是 `(only条件 AND user_id=?) OR user_id=? OR game_uid=?`，
	//   于是「只看在免战的玩家 + 搜索」会把不在免战的玩家也带出来。
	//   现在改成**一个 Where 内的括号条件**，与 only 条件天然 AND。
	if word != "" {
		if uid, err := strconv.Atoi(word); err == nil {
			q = q.Where("user_id = ? OR game_uid = ?", uid, uid)
		} else {
			var wu model.User
			h.home().Select("id").Where("nickname = ?", word).First(&wu)
			if wu.ID > 0 {
				q = q.Where("nickname LIKE ? OR user_id = ?", "%"+word+"%", wu.ID)
			} else {
				q = q.Where("nickname LIKE ?", "%"+word+"%")
			}
		}
	}
	var total int64
	q.Count(&total)
	var rows []model.EzfyProfile
	q.Order("prestige DESC, id ASC").Offset(offset).Limit(size).Find(&rows)

	// 批量取每个玩家「生效中的免战」信息（each 1 SQL，替代逐行查）
	peaceOf := map[uint]struct{ CityCount int64; Until int64 }{}
	{
		uids := make([]uint, 0, len(rows))
		for _, p := range rows {
			uids = append(uids, p.UserID)
		}
		if len(uids) > 0 {
			var agg []struct {
				UserID uint
				Cities int64
				Until  int64
			}
			h.DB.Table("ezfy_city_effect e").
				Select("c.user_id AS user_id, COUNT(*) AS cities, MAX(e.until_time) AS until").
				Joins("JOIN ezfy_city c ON c.id = e.city_id").
				Where("e.effect_type = 2 AND e.until_time > ? AND c.user_id IN ?", now, uids).
				Group("c.user_id").Scan(&agg)
			for _, a := range agg {
				peaceOf[a.UserID] = struct{ CityCount int64; Until int64 }{CityCount: a.Cities, Until: a.Until}
			}
		}
	}

	// 批量取昵称/家园号
	uids := make([]uint, 0, len(rows))
	for _, p := range rows {
		uids = append(uids, p.UserID)
	}
	nick, num := h.ezfyAdminNamesBatch(uids)

	// 批量取每个玩家**总城池数**（开始免战弹窗展示「全部 N 城」用）
	totalCityOf := map[uint]int64{}
	{
		var agg []struct {
			UserID uint
			Cities int64
		}
		if len(uids) > 0 {
			h.DB.Model(&model.EzfyCity{}).
				Select("user_id, COUNT(*) AS cities").
				Where("user_id IN ?", uids).Group("user_id").Scan(&agg)
			for _, a := range agg {
				totalCityOf[a.UserID] = a.Cities
			}
		}
	}

	out := []gin.H{}
	for _, p := range rows {
		pe := peaceOf[p.UserID]
		active := pe.Until > 0
		out = append(out, gin.H{
			"user_id":           p.UserID,
			"nickname":          p.Nickname,
			"home_nick":         nick[p.UserID],
			"home_num":          num[p.UserID],
			"game_uid":          p.GameUID,
			"camp":              p.Camp,
			"camp_name":         ezfyCampName(p.Camp),
			"prestige":          p.Prestige,
			"city_count":        totalCityOf[p.UserID],
			"peace_active":      active,
			"peace_expire_at":   pe.Until,
			"peace_city_count":  pe.CityCount,
			"peace_cool_until":  p.PeaceCoolUntil,
		})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// AdminEzfyPeaceStart POST /admin/ezfy-peace/:id/start?hours=8 —— 给该玩家**全部城池**开始免战
//
// id = user_id。hours 默认 8（与玩家端免战道具时长一致）。直接覆盖每座城现有的免战到期时间（= now + hours）。
// ★ 不受玩家 peace_cool_until 冷却限制（管理端覆盖）。
func (h *EzfyAdmin) AdminEzfyPeaceStart(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	hours, _ := strconv.Atoi(c.DefaultQuery("hours", "8"))
	if hours <= 0 {
		hours = 8
	}
	var prof model.EzfyProfile
	if err := h.DB.Where("user_id = ?", id).First(&prof).Error; err != nil {
		resp.NotFound(c, "玩家不存在")
		return
	}
	var cities []model.EzfyCity
	h.DB.Select("id").Where("user_id = ?", id).Find(&cities)
	if len(cities) == 0 {
		resp.ParamError(c, "该玩家名下还没有城池")
		return
	}
	until := time.Now().UnixMilli() + int64(hours)*3600000
	applied := 0
	for _, ct := range cities {
		// 覆盖式：先删旧的 effect_type=2，再插入新到期时间（保证精确到 now+hours，不做累加）
		h.DB.Where("city_id = ? AND effect_type = ?", int64(ct.ID), 2).
			Delete(&model.EzfyCityEffect{})
		if err := h.DB.Create(&model.EzfyCityEffect{
			CityId: int64(ct.ID), EffectType: 2, Param1: hours, UntilTime: until,
		}).Error; err == nil {
			applied++
		}
	}
	// 管理端不受冷却限制：不写 peace_cool_until（保持玩家自己道具的冷却节奏不受扰动）
	resp.OK(c, gin.H{
		"msg":  fmt.Sprintf("已为「%s」的 %d 座城池开启免战 %d 小时（到期 %s）",
			prof.Nickname, applied, hours, time.UnixMilli(until).Format("2006-01-02 15:04")),
		"applied": applied, "until": until,
	})
}

// AdminEzfyPeaceEnd DELETE /admin/ezfy-peace/:id —— 结束该玩家的免战（清掉全部城池的免战保护）
func (h *EzfyAdmin) AdminEzfyPeaceEnd(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var prof model.EzfyProfile
	if err := h.DB.Where("user_id = ?", id).First(&prof).Error; err != nil {
		resp.NotFound(c, "玩家不存在")
		return
	}
	var cities []model.EzfyCity
	h.DB.Select("id").Where("user_id = ?", id).Find(&cities)
	ids := make([]int64, 0, len(cities))
	for _, ct := range cities {
		ids = append(ids, int64(ct.ID))
	}
	var cnt int64
	if len(ids) > 0 {
		res := h.DB.Where("effect_type = ? AND until_time > ? AND city_id IN ?", 2, time.Now().UnixMilli(), ids).
			Delete(&model.EzfyCityEffect{})
		cnt = res.RowsAffected
	}
	resp.OK(c, gin.H{"msg": fmt.Sprintf("已结束「%s」的免战，清除了 %d 座城池的免战保护", prof.Nickname, cnt), "removed": cnt})
}