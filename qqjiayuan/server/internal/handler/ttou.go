package handler

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

type TtouHandler struct{ DB *gorm.DB }

// currentTTou 当前 T台秀秀主（后台指定优先，否则经验最高）
func (h *TtouHandler) currentTTou() uint {
	var ttouID string
	h.DB.Raw("SELECT value FROM settings WHERE `key` = 'ttou_user_id'").Scan(&ttouID)
	if ttouID != "" {
		var id uint
		h.DB.Raw("SELECT id FROM users WHERE username = ?", ttouID).Scan(&id)
		if id > 0 {
			return id
		}
	}
	var u model.User
	h.DB.Where("status = 1").Order("exp DESC").First(&u)
	return u.ID
}

// 我要上榜：提交/更新上榜宣言申请
func (h *TtouHandler) Apply(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Slogan string `json:"slogan"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请填写上榜宣言")
		return
	}
	req.Slogan = strings.TrimSpace(req.Slogan)
	if req.Slogan == "" {
		resp.ParamError(c, "请填写上榜宣言")
		return
	}
	if len(req.Slogan) > 100 {
		resp.ParamError(c, "宣言最多 100 字")
		return
	}
	var cnt int64
	h.DB.Model(&model.TtouApply{}).Where("user_id = ? AND status = 0", uid).Count(&cnt)
	if cnt > 0 {
		h.DB.Model(&model.TtouApply{}).Where("user_id = ? AND status = 0", uid).Update("slogan", req.Slogan)
	} else {
		h.DB.Create(&model.TtouApply{UserID: uid, Slogan: req.Slogan})
	}
	resp.OK(c, nil)
}

// 我要膜拜：每天对当前秀主限一次
func (h *TtouHandler) Worship(c *gin.Context) {
	uid := middleware.GetUID(c)
	target := h.currentTTou()
	if target == 0 {
		resp.ParamError(c, "暂无T台秀秀主")
		return
	}
	if target == uid {
		resp.ParamError(c, "不能膜拜自己哦")
		return
	}
	today := time.Now().Format("2006-01-02")
	var cnt int64
	h.DB.Raw("SELECT COUNT(*) FROM ttou_worships WHERE user_id = ? AND target_id = ? AND DATE(created_at) = ?",
		uid, target, today).Scan(&cnt)
	if cnt > 0 {
		resp.ParamError(c, "今天已经膜拜过啦，明天再来~")
		return
	}
	h.DB.Create(&model.TtouWorship{TargetID: target, UserID: uid})
	var total int64
	h.DB.Model(&model.TtouWorship{}).Where("target_id = ?", target).Count(&total)
	resp.OK(c, gin.H{"count": total, "target_id": target})
}
