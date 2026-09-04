package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

type FavoriteHandler struct{ DB *gorm.DB }

// 我收藏的帖子
func (h *FavoriteHandler) MyFavorites(c *gin.Context) {
	uid := middleware.GetUID(c)
	ids := []uint{}
	var favs []model.ThreadFavorite
	h.DB.Where("user_id = ?", uid).Order("created_at DESC").Find(&favs)
	for _, f := range favs {
		ids = append(ids, f.ThreadID)
	}
	var threads []model.Thread
	if len(ids) > 0 {
		h.DB.Preload("User").Preload("Board").
			Where("id IN ? AND status = 1", ids).Order("created_at DESC").Find(&threads)
	}
	resp.OK(c, threads)
}

// 我的回复
func (h *FavoriteHandler) MyReplies(c *gin.Context) {
	uid := middleware.GetUID(c)
	var replies []model.Reply
	h.DB.Preload("User").Preload("Thread").Preload("Thread.User").
		Where("user_id = ? AND status = 1", uid).Order("created_at DESC").Limit(30).Find(&replies)
	resp.OK(c, replies)
}

// 收藏 / 取消收藏（切换）
func (h *FavoriteHandler) Toggle(c *gin.Context) {
	uid := middleware.GetUID(c)
	id, _ := strconv.Atoi(c.Param("id"))
	var th model.Thread
	if err := h.DB.First(&th, id).Error; err != nil {
		resp.NotFound(c, "帖子不存在")
		return
	}
	var fav model.ThreadFavorite
	if err := h.DB.Where("user_id = ? AND thread_id = ?", uid, id).First(&fav).Error; err == nil {
		h.DB.Delete(&fav)
		resp.OK(c, gin.H{"favored": false})
		return
	}
	h.DB.Create(&model.ThreadFavorite{UserID: uid, ThreadID: uint(id)})
	resp.OK(c, gin.H{"favored": true})
}

// 我是否已收藏（帖子详情用）
func (h *FavoriteHandler) Status(c *gin.Context) {
	uid := middleware.GetUID(c)
	id, _ := strconv.Atoi(c.Param("id"))
	var n int64
	h.DB.Model(&model.ThreadFavorite{}).Where("user_id = ? AND thread_id = ?", uid, id).Count(&n)
	resp.OK(c, gin.H{"favored": n > 0})
}
