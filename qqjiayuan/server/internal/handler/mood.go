package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

type MoodHandler struct{ DB *gorm.DB }

// List 我的心情列表（分页，不依赖空间是否开通）
func (h *MoodHandler) List(c *gin.Context) {
	uid := middleware.GetUID(c)
	page, size := pageParams(c, 10)
	q := h.DB.Model(&model.Mood{}).Where("user_id = ? AND status = 1", uid)
	var total int64
	q.Count(&total)
	var moods []model.Mood
	q.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&moods)
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": moods})
}

// Add 发表心情（maxlength 120，同参考页输入框）
func (h *MoodHandler) Add(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Content string `json:"content" binding:"required,max=120"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "心情内容不能为空，最多120字")
		return
	}
	m := model.Mood{UserID: uid, Content: req.Content, Status: 1}
	h.DB.Create(&m)
	resp.OK(c, m)
}

// Del 删除自己的心情
func (h *MoodHandler) Del(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	uid := middleware.GetUID(c)
	h.DB.Model(&model.Mood{}).Where("id = ? AND user_id = ?", id, uid).Update("status", 0)
	resp.OK(c, nil)
}

// Latest 我的最新一条心情（用于我的家园顶部展示）
func (h *MoodHandler) Latest(c *gin.Context) {
	uid := middleware.GetUID(c)
	var m model.Mood
	err := h.DB.Where("user_id = ? AND status = 1", uid).Order("created_at DESC").First(&m).Error
	if err != nil {
		resp.OK(c, nil)
		return
	}
	resp.OK(c, m)
}
