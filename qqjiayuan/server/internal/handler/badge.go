package handler

import (
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

type BadgeHandler struct{ DB *gorm.DB }

// 公开：勋章列表（资料页/选择器用）
func (h *BadgeHandler) List(c *gin.Context) {
	var badges []model.Badge
	h.DB.Order("id ASC").Find(&badges)
	resp.OK(c, badges)
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

// 管理端：勋章分页列表
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
	q.Order("id ASC").Offset((page - 1) * size).Limit(size).Find(&badges)
	resp.OK(c, gin.H{"list": badges, "total": total, "page": page, "size": size})
}

func (h *BadgeHandler) Create(c *gin.Context) {
	var req struct {
		Name   string `json:"name" binding:"required,min=1,max=30"`
		Icon   string `json:"icon" binding:"required"`
		Remark string `json:"remark" binding:"max=100"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "名称和图标必填")
		return
	}
	if !h.validBadgeIcon(req.Icon) {
		resp.ParamError(c, "图标不在素材库中")
		return
	}
	badge := model.Badge{Name: req.Name, Icon: req.Icon, Remark: req.Remark}
	h.DB.Create(&badge)
	resp.OK(c, badge)
}

func (h *BadgeHandler) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Name   string `json:"name" binding:"required,min=1,max=30"`
		Icon   string `json:"icon" binding:"required"`
		Remark string `json:"remark" binding:"max=100"`
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
	h.DB.Model(&badge).Updates(map[string]interface{}{"name": req.Name, "icon": req.Icon, "remark": req.Remark})
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

// 给用户授予/重设马甲
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
	var badges []model.Badge
	if len(req.BadgeIDs) > 0 {
		h.DB.Where("id IN ?", req.BadgeIDs).Find(&badges)
	}
	h.DB.Model(&user).Association("Badges").Replace(&badges)
	resp.OK(c, nil)
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
