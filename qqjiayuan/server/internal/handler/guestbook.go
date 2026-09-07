package handler

import (
	"crypto/md5"
	"encoding/hex"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

type GuestHandler struct{ DB *gorm.DB }

type guestItem struct {
	model.GuestBook
	Nickname string  `json:"nickname"`
	Replies  []gin.H `json:"replies"`
	Locked   bool    `json:"locked"`
}

func hashGuestPass(pass string) string {
	sum := md5.Sum([]byte(pass + "jiayuan-guest"))
	return hex.EncodeToString(sum[:])
}

func (h *GuestHandler) isAdmin(uid uint) bool {
	for _, code := range middleware.UserPermissionCodes(h.DB, uid) {
		if code == "admin:access" {
			return true
		}
	}
	return false
}

// List 留言本列表（私密留言凭密码解锁后可见，未解锁显示占位）
func (h *GuestHandler) List(c *gin.Context) {
	page, size := pageParams(c, 10)
	q := h.DB.Model(&model.GuestBook{}).Where("status = 1")
	var total int64
	q.Count(&total)
	var list []model.GuestBook
	q.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&list)

	uid := middleware.GetUID(c)
	unlocked := c.Query("unlocked")
	out := make([]guestItem, 0, len(list))
	for _, g := range list {
		item := guestItem{GuestBook: g, Replies: []gin.H{}}
		if g.UserID > 0 {
			var u model.User
			h.DB.Select("nickname").First(&u, g.UserID)
			item.Nickname = u.Nickname
		}
		if g.Pass != "" {
			item.Locked = true
			if uid > 0 && (h.isAdmin(uid) || g.UserID == uid) {
				item.Locked = false
			} else if containsID(unlocked, int(g.ID)) {
				item.Locked = false
			}
		}
		if item.Locked {
			item.Content = "【私密留言，输入留言密码查看】"
		}
		var replies []model.GuestReply
		h.DB.Where("guest_id = ? AND status = 1", g.ID).Order("created_at ASC").Limit(10).Find(&replies)
		for _, r := range replies {
			var ru model.User
			h.DB.Select("nickname").First(&ru, r.UserID)
			item.Replies = append(item.Replies, gin.H{"id": r.ID, "content": r.Content, "nickname": ru.Nickname, "created_at": r.CreatedAt})
		}
		out = append(out, item)
	}
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": out})
}

// Add 留言（游客可留：name+content；登录用户自动带昵称；pass 非空为私密留言）
func (h *GuestHandler) Add(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Name    string `json:"name"`
		Pass    string `json:"pass"`
		Content string `json:"content" binding:"required,max=500"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "留言内容不能为空且不超过 500 字")
		return
	}
	if !verifyTime("guest_"+strconv.FormatUint(uint64(uid), 10), 5) {
		resp.ParamError(c, "留言太频繁了，歇 5 秒再来")
		return
	}
	g := model.GuestBook{UserID: uid, Content: req.Content, Status: 1}
	if uid > 0 {
		var u model.User
		h.DB.Select("nickname").First(&u, uid)
		g.Name = u.Nickname
	} else {
		g.Name = strings.TrimSpace(req.Name)
		if g.Name == "" {
			resp.ParamError(c, "游客留言请留下你的名字")
			return
		}
	}
	pass := strings.TrimSpace(req.Pass)
	if pass != "" {
		g.Pass = hashGuestPass(pass)
	}
	h.DB.Create(&g)
	resp.OK(c, gin.H{"id": g.ID, "private": pass != ""})
}

// Unlock 校验私密留言密码
func (h *GuestHandler) Unlock(c *gin.Context) {
	id, ok := positiveParam(c, "id")
	if !ok {
		return
	}
	var req struct {
		Pass string `json:"pass" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请输入留言密码")
		return
	}
	var g model.GuestBook
	if err := h.DB.Where("id = ? AND status = 1", id).First(&g).Error; err != nil {
		resp.NotFound(c, "留言不存在")
		return
	}
	if g.Pass == "" || g.Pass != hashGuestPass(req.Pass) {
		resp.ParamError(c, "留言密码不对")
		return
	}
	resp.OK(c, gin.H{"id": g.ID, "content": g.Content, "name": g.Name, "created_at": g.CreatedAt})
}

// Reply 回复留言（仅管理员）
func (h *GuestHandler) Reply(c *gin.Context) {
	uid := middleware.GetUID(c)
	if !h.isAdmin(uid) {
		resp.Forbidden(c, "只有站长才能回复留言本")
		return
	}
	id, ok := positiveParam(c, "id")
	if !ok {
		return
	}
	var req struct {
		Content string `json:"content" binding:"required,max=300"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "回复内容不能为空")
		return
	}
	var g model.GuestBook
	if err := h.DB.First(&g, id).Error; err != nil {
		resp.NotFound(c, "留言不存在")
		return
	}
	h.DB.Create(&model.GuestReply{GuestID: g.ID, UserID: uid, Content: req.Content, Status: 1})
	resp.OK(c, nil)
}

// Del 删除留言（管理员或留言者本人）
func (h *GuestHandler) Del(c *gin.Context) {
	uid := middleware.GetUID(c)
	id, ok := positiveParam(c, "id")
	if !ok {
		return
	}
	q := h.DB.Model(&model.GuestBook{}).Where("id = ?", id)
	if !h.isAdmin(uid) {
		q = q.Where("user_id = ?", uid)
	}
	q.Update("status", 0)
	resp.OK(c, nil)
}

func containsID(s string, id int) bool {
	if s == "" {
		return false
	}
	target := strconv.Itoa(id)
	for _, p := range strings.Split(s, ",") {
		if p == target {
			return true
		}
	}
	return false
}
