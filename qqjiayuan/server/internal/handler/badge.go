package handler

import (
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

type BadgeHandler struct{ DB *gorm.DB }

// cleanupExpired 自动清理已过期会员勋章（复刻诺哈 wap_medal 过期清除）
func (h *BadgeHandler) cleanupExpired() {
	h.DB.Exec("DELETE FROM user_badges WHERE expire_at IS NOT NULL AND expire_at < NOW()")
}

// 公开：勋章商店列表（上架中，按排序）—— 用于勋章大全/资料页
func (h *BadgeHandler) List(c *gin.Context) {
	h.cleanupExpired()
	var badges []model.Badge
	h.DB.Where("status = 1").Order("sort ASC, id ASC").Find(&badges)
	resp.OK(c, badges)
}

// 我的勋章（用户端：展示我获得的勋章及其到期时间）
func (h *BadgeHandler) MyMedals(c *gin.Context) {
	h.cleanupExpired()
	uid := middleware.GetUID(c)
	if uid == 0 {
		resp.Unauthorized(c, "请先登录")
		return
	}
	var ubs []model.UserBadge
	h.DB.Where("user_id = ?", uid).Order("sort ASC, id ASC").Find(&ubs)
	badges := map[uint]model.Badge{}
	var all []model.Badge
	h.DB.Where("status = 1").Find(&all)
	for _, b := range all {
		badges[b.ID] = b
	}
	out := []gin.H{}
	for _, ub := range ubs {
		b, ok := badges[ub.BadgeID]
		if !ok {
			continue
		}
		out = append(out, gin.H{
			"id": b.ID, "name": b.Name, "icon": b.Icon, "remark": b.Remark,
			"price": b.Price, "period": b.Period, "sort": ub.Sort,
			"granted_at": ub.GrantedAt, "expire_at": ub.ExpireAt,
		})
	}
	resp.OK(c, gin.H{"list": out})
}

// 公开：可用的图标/头像/logo 素材（从资源库读取启用的条目）
func (h *BadgeHandler) Presets(c *gin.Context) {
	files := func(cat string) []string {
		var rs []model.Resource
		h.DB.Where("category = ? AND status = 1", cat).Order("id ASC").Find(&rs)
		out := make([]string, 0, len(rs))
		for _, r := range rs {
			out = append(out, filepath.Base(r.File))
		}
		return out
	}
	resp.OK(c, gin.H{
		"badge_icons": files("badge"),
		"avatars":     files("avatar"),
		"game_logos":  files("game"),
	})
}

// 管理端：勋章商店分页列表
func (h *BadgeHandler) AdminList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 10
	}
	q := h.DB.Model(&model.Badge{})
	var total int64
	q.Count(&total)
	if maxPage := int(total+int64(size)-1) / size; maxPage < 1 {
		page = 1
	} else if page > maxPage {
		page = maxPage
	}
	var badges []model.Badge
	q.Order("sort ASC, id ASC").Offset((page - 1) * size).Limit(size).Find(&badges)
	resp.OK(c, gin.H{"list": badges, "total": total, "page": page, "size": size})
}

func (h *BadgeHandler) Create(c *gin.Context) {
	var req struct {
		Name   string `json:"name" binding:"required,min=1,max=30"`
		Icon   string `json:"icon" binding:"required"`
		Remark string `json:"remark" binding:"max=100"`
		Price  int    `json:"price"`
		Period int    `json:"period"`
		Sort   int    `json:"sort"`
		Status int    `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "名称和图标必填")
		return
	}
	if !h.validBadgeIcon(req.Icon) {
		resp.ParamError(c, "图标不在素材库中")
		return
	}
	badge := model.Badge{Name: req.Name, Icon: req.Icon, Remark: req.Remark, Price: req.Price, Period: req.Period, Sort: req.Sort, Status: boolToInt(req.Status != 0)}
	if badge.Status == 0 {
		badge.Status = 1
	}
	h.DB.Create(&badge)
	resp.OK(c, badge)
}

func (h *BadgeHandler) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Name   string `json:"name" binding:"required,min=1,max=30"`
		Icon   string `json:"icon" binding:"required"`
		Remark string `json:"remark" binding:"max=100"`
		Price  int    `json:"price"`
		Period int    `json:"period"`
		Sort   int    `json:"sort"`
		Status int    `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "名称和图标必填")
		return
	}
	if !h.validBadgeIcon(req.Icon) {
		resp.ParamError(c, "图标不在素材库中")
		return
	}
	var badge model.Badge
	if err := h.DB.First(&badge, id).Error; err != nil {
		resp.NotFound(c, "勋章不存在")
		return
	}
	h.DB.Model(&badge).Updates(map[string]interface{}{
		"name": req.Name, "icon": req.Icon, "remark": req.Remark,
		"price": req.Price, "period": req.Period, "sort": req.Sort,
		"status": boolToInt(req.Status != 0),
	})
	resp.OK(c, badge)
}

func (h *BadgeHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var badge model.Badge
	if err := h.DB.First(&badge, id).Error; err != nil {
		resp.NotFound(c, "勋章不存在")
		return
	}
	h.DB.Exec("DELETE FROM user_badges WHERE badge_id = ?", id)
	h.DB.Delete(&badge)
	resp.OK(c, nil)
}

// 授予/重设会员勋章（复刻诺哈 medal_add：uid + 勋章编号 + 排序 + 有效期）
func (h *BadgeHandler) UserBadges(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		BadgeIDs []uint `json:"badge_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请提交 badge_ids 数组")
		return
	}
	var user model.User
	if err := h.DB.First(&user, id).Error; err != nil {
		resp.NotFound(c, "用户不存在")
		return
	}
	// 先清除旧的会员勋章
	h.DB.Exec("DELETE FROM user_badges WHERE user_id = ?", id)
	now := time.Now()
	for i, bid := range req.BadgeIDs {
		var b model.Badge
		if err := h.DB.First(&b, bid).Error; err != nil {
			continue
		}
		ub := model.UserBadge{UserID: uint(id), BadgeID: b.ID, Sort: i + 1, GrantedAt: now}
		if b.Period > 0 {
			exp := now.AddDate(0, 0, b.Period)
			ub.ExpireAt = &exp
		}
		h.DB.Create(&ub)
	}
	resp.OK(c, nil)
}

// 会员勋章分页列表（管理端：复刻诺哈 medal_list/medal_search —— 可按号码 uid/勋章 badge_id 筛选）
func (h *BadgeHandler) AdminUserBadges(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 10
	}
	uid, _ := strconv.Atoi(c.DefaultQuery("uid", "0"))
	bid, _ := strconv.Atoi(c.DefaultQuery("badge_id", "0"))
	h.cleanupExpired()
	q := h.DB.Model(&model.UserBadge{})
	if uid > 0 {
		q = q.Where("user_id = ?", uid)
	}
	if bid > 0 {
		q = q.Where("badge_id = ?", bid)
	}
	var total int64
	q.Count(&total)
	if maxPage := int(total+int64(size)-1) / size; maxPage < 1 {
		page = 1
	} else if page > maxPage {
		page = maxPage
	}
	var list []model.UserBadge
	q.Order("user_id ASC, sort ASC, id ASC").Offset((page - 1) * size).Limit(size).Find(&list)
	// 勋章商店索引
	badges := map[uint]model.Badge{}
	var all []model.Badge
	h.DB.Find(&all)
	for _, b := range all {
		badges[b.ID] = b
	}
	// 用户索引
	users := map[uint]model.User{}
	var allU []model.User
	h.DB.Find(&allU)
	for _, u := range allU {
		users[u.ID] = u
	}
	out := []gin.H{}
	for _, ub := range list {
		b, _ := badges[ub.BadgeID]
		u, _ := users[ub.UserID]
		out = append(out, gin.H{
			"id": ub.ID, "user_id": ub.UserID, "badge_id": ub.BadgeID,
			"sort": ub.Sort, "granted_at": ub.GrantedAt, "expire_at": ub.ExpireAt,
			"badge_name": b.Name, "badge_icon": b.Icon, "badge_remark": b.Remark,
			"nickname": u.Nickname, "color": u.Color,
		})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// 回收会员勋章（管理端：复刻诺哈 medal_del）
func (h *BadgeHandler) AdminUserBadgeDel(c *gin.Context) {
	ubID, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.UserBadge{}, ubID)
	resp.OK(c, nil)
}

// 编辑会员勋章（管理端：复刻诺哈 medal_edit —— 改排序 / 到期时间，空到期=永久）
func (h *BadgeHandler) AdminUserBadgeUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Sort     int    `json:"sort"`
		ExpireAt string `json:"expire_at"` // "2006-01-02 15:04:05" 或 "2006-01-02"，空=永久
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	var ub model.UserBadge
	if err := h.DB.First(&ub, id).Error; err != nil {
		resp.NotFound(c, "勋章记录不存在")
		return
	}
	updates := map[string]interface{}{"sort": req.Sort}
	if req.ExpireAt == "" {
		updates["expire_at"] = nil
	} else if t, err := time.ParseInLocation("2006-01-02 15:04:05", req.ExpireAt, time.Local); err == nil {
		updates["expire_at"] = t
	} else if t, err := time.ParseInLocation("2006-01-02", req.ExpireAt, time.Local); err == nil {
		updates["expire_at"] = t
	} else {
		resp.ParamError(c, "到期时间格式错误（空=永久，或 yyyy-MM-dd HH:mm:ss）")
		return
	}
	h.DB.Model(&ub).Updates(updates)
	resp.OK(c, gin.H{"id": ub.ID, "sort": req.Sort, "expire_at": updates["expire_at"]})
}

// 给用户授予单个勋章（管理端：指定勋章+有效天数）
func (h *BadgeHandler) GrantOne(c *gin.Context) {
	var req struct {
		UserID  uint `json:"user_id" binding:"required"`
		BadgeID uint `json:"badge_id" binding:"required"`
		Days    int  `json:"days"` // 有效天数，0=永久
		Sort    int  `json:"sort"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请选择用户和勋章")
		return
	}
	var u model.User
	if err := h.DB.First(&u, req.UserID).Error; err != nil {
		resp.NotFound(c, "用户不存在")
		return
	}
	var b model.Badge
	if err := h.DB.First(&b, req.BadgeID).Error; err != nil {
		resp.NotFound(c, "勋章不存在")
		return
	}
	days := req.Days
	if days <= 0 {
		days = b.Period
	}
	now := time.Now()
	ub := model.UserBadge{UserID: req.UserID, BadgeID: req.BadgeID, Sort: req.Sort, GrantedAt: now}
	if days > 0 {
		exp := now.AddDate(0, 0, days)
		ub.ExpireAt = &exp
	}
	h.DB.Create(&ub)
	resp.OK(c, gin.H{"id": ub.ID, "name": b.Name, "expire_at": ub.ExpireAt})
}

// validBadgeIcon 校验图标是否在资源库（badge 类，启用中）
func (h *BadgeHandler) validBadgeIcon(icon string) bool {
	var n int64
	h.DB.Model(&model.Resource{}).
		Where("category = ? AND status = 1 AND file LIKE ?", "badge", "%/"+icon).
		Count(&n)
	if n > 0 {
		return true
	}
	// 兼容：老勋章引用的演示站图标（资源库未登记也可用）
	for _, p := range model.BadgeIconPresets {
		if p == icon {
			return true
		}
	}
	return false
}