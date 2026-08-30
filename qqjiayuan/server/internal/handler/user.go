package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

type UserHandler struct {
	DB *gorm.DB
}

// 通用分页参数：page 从 1 起，size 默认 10
func pageOf(c *gin.Context, defSize int) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", strconv.Itoa(defSize)))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = defSize
	}
	return page, (page - 1) * size
}

// 他人主页：资料 + 最新发帖/回帖统计
func (h *UserHandler) Profile(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var user model.User
	if err := h.DB.First(&user, id).Error; err != nil {
		resp.NotFound(c, "这位友友不见了")
		return
	}
	var threadCount, replyCount, signDays int64
	h.DB.Model(&model.Thread{}).Where("user_id = ? AND status = 1", user.ID).Count(&threadCount)
	h.DB.Model(&model.Reply{}).Where("user_id = ? AND status = 1", user.ID).Count(&replyCount)
	h.DB.Model(&model.SignIn{}).Where("user_id = ?", user.ID).Count(&signDays)

	var threads []model.Thread
	h.DB.Preload("Board").Where("user_id = ? AND status = 1", user.ID).
		Order("created_at DESC").Limit(10).Find(&threads)

	var badges []model.Badge
	h.DB.Model(&user).Association("Badges").Find(&badges)
	var roles []model.Role
	h.DB.Model(&user).Association("Roles").Find(&roles)
	online := user.LastActiveAt != nil && time.Since(*user.LastActiveAt) < 10*time.Minute

	// 婚恋状态：伴侣昵称
	partnerName := ""
	if user.PartnerID > 0 {
		var p model.User
		if err := h.DB.First(&p, user.PartnerID).Error; err == nil {
			partnerName = p.Nickname
		}
	}
	// 社区职务：职务类马甲（演示站：1.<图标>公坛协管员 2.家族版主）
	dutyIcons := map[string]bool{"706.jpg": true, "704.gif": true, "3.gif": true, "501.gif": true}
	duties := []gin.H{}
	for _, b := range badges {
		if dutyIcons[b.Icon] {
			duties = append(duties, gin.H{"name": b.Name, "icon": b.Icon})
		}
	}

	resp.OK(c, gin.H{
		"id": user.ID, "username": user.Username, "nickname": user.Nickname,
		"gender": user.Gender, "signature": user.Signature, "color": user.Color,
		"avatar": user.Avatar, "level": user.Level, "exp": user.Exp, "coins": user.Coins,
		"level_icon": user.LevelIcon, "level_title": user.LevelTitle,
		"noble": user.Noble, "partner_id": user.PartnerID, "partner_name": partnerName,
		"baby_name": user.BabyName, "achieve": user.Achieve, "achieve_level": user.AchieveLevel,
		"priv_id": user.PrivID, "priv": user.Priv,
		"online":     online,
		"created_at": user.CreatedAt, "last_login_at": user.LastLoginAt,
		"thread_count": threadCount, "reply_count": replyCount, "sign_days": signDays,
		"badges": badges, "roles": roles, "duties": duties, "threads": threads,
	})
}

type profileReq struct {
	Signature string `json:"signature" binding:"max=50"`
	Gender    int    `json:"gender"`
	Color     string `json:"color" binding:"max=10"`
	Nickname  string `json:"nickname" binding:"min=1,max=20"`
	Avatar    string `json:"avatar" binding:"max=100"`
}

func (h *UserHandler) UpdateMe(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req profileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "资料格式不对：昵称1-20字，签名50字以内")
		return
	}
	if req.Gender != 1 && req.Gender != 2 {
		req.Gender = 1
	}
	var user model.User
	h.DB.First(&user, uid)
	// 想改名？家园传统：改名要谨慎
	if req.Nickname != user.Nickname {
		var n int64
		h.DB.Model(&model.User{}).Where("nickname = ? AND id <> ?", req.Nickname, uid).Count(&n)
		if n > 0 {
			resp.ParamError(c, "这个昵称已经有友友用啦")
			return
		}
	}
	h.DB.Model(&user).Updates(map[string]interface{}{
		"nickname": req.Nickname, "signature": req.Signature,
		"gender": req.Gender, "color": req.Color, "avatar": req.Avatar,
	})
	resp.OK(c, nil)
}
