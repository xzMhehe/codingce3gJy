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

type FriendHandler struct{ DB *gorm.DB }

// 我的好友 + 待处理申请
func (h *FriendHandler) List(c *gin.Context) {
	uid := middleware.GetUID(c)
	var rows []model.Friendship
	h.DB.Preload("User").Where("friend_id = ? AND status = 0", uid).Find(&rows) // 收到的申请
	var friends []model.Friendship
	h.DB.Where("(user_id = ? OR friend_id = ?) AND status = 1", uid, uid).Find(&friends)

	type friendInfo struct {
		ID       uint   `json:"id"`
		Nickname string `json:"nickname"`
		Color    string `json:"color"`
		Level    int    `json:"level"`
		Sign     string `json:"signature"`
		Online   bool   `json:"online"`
	}
	list := []friendInfo{}
	for _, f := range friends {
		otherID := f.FriendID
		if f.UserID == uid {
			otherID = f.UserID
		}
		var u model.User
		if err := h.DB.First(&u, otherID).Error; err == nil {
			online := u.LastActiveAt != nil && time.Since(*u.LastActiveAt) < 10*time.Minute
			list = append(list, friendInfo{ID: u.ID, Nickname: u.Nickname, Color: u.Color, Level: u.Level, Sign: u.Signature, Online: online})
		}
	}
	requests := []gin.H{}
	for _, r := range rows {
		var u model.User
		if err := h.DB.First(&u, r.UserID).Error; err == nil {
			requests = append(requests, gin.H{"apply_id": r.ID, "id": u.ID, "nickname": u.Nickname,
				"color": u.Color, "level": u.Level, "created_at": r.CreatedAt})
		}
	}
	resp.OK(c, gin.H{"friends": list, "requests": requests})
}

type addFriendReq struct {
	TargetID uint `json:"target_id"`
}

func (h *FriendHandler) Add(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req addFriendReq
	if err := c.ShouldBindJSON(&req); err != nil || req.TargetID == 0 {
		resp.ParamError(c, "请指定要添加的友友")
		return
	}
	if req.TargetID == uid {
		resp.ParamError(c, "自己就是自己最好的朋友")
		return
	}
	var target model.User
	if err := h.DB.First(&target, req.TargetID).Error; err != nil {
		resp.NotFound(c, "这位友友不存在")
		return
	}
	var exist model.Friendship
	if err := h.DB.Where("(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)",
		uid, req.TargetID, req.TargetID, uid).First(&exist).Error; err == nil {
		if exist.Status == 1 {
			resp.ParamError(c, "你们已经是好友啦")
		}
		if exist.Status == 0 && exist.FriendID == uid {
			// 对方也申请过我，直接成为好友
			h.DB.Model(&exist).Update("status", 1)
			resp.OK(c, "你们已成为好友")
			return
		}
		resp.ParamError(c, "申请已在路上，等待对方处理")
		return
	}
	h.DB.Create(&model.Friendship{UserID: uid, FriendID: req.TargetID, Status: 0})
	var me model.User
	h.DB.First(&me, uid)
	h.DB.Create(&model.Notification{
		UserID: req.TargetID, Type: "friend", RefID: uid,
		Title: me.Nickname + " 请求加你为好友",
		Content: "到「好友」页面处理这条申请吧",
	})
	resp.OK(c, "好友申请已发送")
}

func (h *FriendHandler) Handle(c *gin.Context) {
	uid := middleware.GetUID(c)
	id, _ := strconv.Atoi(c.Param("id"))
	var act struct {
		Action string `json:"action" binding:"required"` // accept / reject
	}
	if err := c.ShouldBindJSON(&act); err != nil {
		resp.ParamError(c, "action 必须是 accept 或 reject")
		return
	}
	var fr model.Friendship
	if err := h.DB.First(&fr, id).Error; err != nil {
		resp.NotFound(c, "申请不存在")
		return
	}
	if fr.FriendID != uid {
		resp.Forbidden(c, "这不是发给你的申请")
		return
	}
	if act.Action == "accept" {
		h.DB.Model(&fr).Update("status", 1)
		var me, other model.User
		h.DB.First(&me, uid)
		h.DB.First(&other, fr.UserID)
		h.DB.Create(&model.Notification{
			UserID: fr.UserID, Type: "friend", RefID: uid,
			Title:   me.Nickname + " 同意了你的好友申请",
			Content: "你们已成为好友，去打个招呼吧",
		})
		resp.OK(c, "已同意")
		return
	}
	h.DB.Model(&fr).Update("status", 2)
	resp.OK(c, "已拒绝")
}

func (h *FriendHandler) Remove(c *gin.Context) {
	uid := middleware.GetUID(c)
	id, _ := strconv.Atoi(c.Param("id"))
	var fr model.Friendship
	if err := h.DB.First(&fr, id).Error; err != nil {
		resp.NotFound(c, "好友关系不存在")
		return
	}
	if fr.UserID != uid && fr.FriendID != uid {
		resp.Forbidden(c, "只能删除自己的好友")
		return
	}
	h.DB.Delete(&fr)
	resp.OK(c, "已删除")
}

// 私信
type MessageHandler struct{ DB *gorm.DB }

// 会话列表：按联系人分组，含未读数
func (h *MessageHandler) Conversations(c *gin.Context) {
	uid := middleware.GetUID(c)
	type convRow struct {
		UserID      uint   `json:"user_id"`
		Nickname    string `json:"nickname"`
		Color       string `json:"color"`
		LastContent string `json:"last_content"`
		LastAt      string `json:"last_at"`
		Unread      int64  `json:"unread"`
	}
	var convs []convRow
	h.DB.Raw(`
SELECT u.id AS user_id, u.nickname, u.color, m.content AS last_content, m.created_at AS last_at,
  (SELECT COUNT(*) FROM private_messages x WHERE x.sender_id = u.id AND x.receiver_id = ? AND x.is_read = 0) AS unread
FROM private_messages m
JOIN users u ON u.id = IF(m.sender_id = ?, m.receiver_id, m.sender_id)
JOIN (SELECT IF(sender_id = ?, receiver_id, sender_id) AS peer, MAX(id) AS max_id
      FROM private_messages WHERE sender_id = ? OR receiver_id = ? GROUP BY peer) t
  ON t.peer = u.id AND t.max_id = m.id
ORDER BY m.created_at DESC`, uid, uid, uid, uid, uid).Scan(&convs)
	resp.OK(c, convs)
}

// 与某人的私信往来
func (h *MessageHandler) With(c *gin.Context) {
	uid := middleware.GetUID(c)
	peerID, _ := strconv.Atoi(c.Param("id"))
	var peer model.User
	if err := h.DB.First(&peer, peerID).Error; err != nil {
		resp.NotFound(c, "这位友友不存在")
		return
	}
	var msgs []model.PrivateMessage
	h.DB.Preload("Sender").Where("(sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)",
		uid, peerID, peerID, uid).Order("created_at ASC").Limit(200).Find(&msgs)
	h.DB.Model(&model.PrivateMessage{}).
		Where("sender_id = ? AND receiver_id = ? AND is_read = 0", peerID, uid).Update("is_read", 1)
	resp.OK(c, gin.H{"peer": gin.H{"id": peer.ID, "nickname": peer.Nickname, "color": peer.Color}, "list": msgs})
}

type sendMsgReq struct {
	To      uint   `json:"to" binding:"required"`
	Content string `json:"content" binding:"required,min=1,max=500"`
}

func (h *MessageHandler) Send(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req sendMsgReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请选择接收人并填写内容（500字以内）")
		return
	}
	if req.To == uid {
		resp.ParamError(c, "不能给自己发私信哦")
		return
	}
	var target model.User
	if err := h.DB.First(&target, req.To).Error; err != nil {
		resp.NotFound(c, "这位友友不存在")
		return
	}
	msg := model.PrivateMessage{SenderID: uid, ReceiverID: req.To, Content: req.Content}
	h.DB.Create(&msg)
	resp.OK(c, gin.H{"id": msg.ID})
}

// 聊天室（family_id=0 公共 / >0 家族聊室，轮询）
type ChatHandler struct{ DB *gorm.DB }

func (h *ChatHandler) List(c *gin.Context) {
	after, _ := strconv.Atoi(c.DefaultQuery("after", "0"))
	famID, _ := strconv.Atoi(c.DefaultQuery("family_id", "0"))
	var msgs []model.ChatMessage
	q := h.DB.Preload("User").Preload("User.Badges").Where("family_id = ?", famID)
	q.Where("id > ?", after).Order("id ASC").Limit(50).Find(&msgs)
	resp.OK(c, msgs)
}

type chatReq struct {
	Content string `json:"content" binding:"required,min=1,max=500"`
}

func (h *ChatHandler) Send(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req chatReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "说点什么吧（500字以内）")
		return
	}
	famID, _ := strconv.Atoi(c.DefaultQuery("family_id", "0"))
	if famID > 0 {
		// 家族聊室：仅成员可发言
		var n int64
		h.DB.Model(&model.FamilyMember{}).Where("family_id = ? AND user_id = ?", famID, uid).Count(&n)
		if n == 0 {
			resp.Forbidden(c, "只有家族成员才能在家族聊室发言")
			return
		}
	}
	msg := model.ChatMessage{UserID: uid, FamilyID: uint(famID), Content: req.Content}
	h.DB.Create(&msg)
	h.DB.Preload("User").Preload("User.Badges").First(&msg, msg.ID)
	resp.OK(c, msg)
}

// 通知
type NotifyHandler struct{ DB *gorm.DB }

func (h *NotifyHandler) List(c *gin.Context) {
	uid := middleware.GetUID(c)
	page, offset, size := pageOf(c, 20)
	var total int64
	h.DB.Model(&model.Notification{}).Where("user_id = ?", uid).Count(&total)
	var list []model.Notification
	h.DB.Where("user_id = ?", uid).Order("created_at DESC").Offset(offset).Limit(size).Find(&list)
	var unread int64
	h.DB.Model(&model.Notification{}).Where("user_id = ? AND is_read = 0", uid).Count(&unread)
	resp.OK(c, gin.H{"list": list, "total": total, "page": page, "unread": unread})
}

func (h *NotifyHandler) ReadAll(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.DB.Model(&model.Notification{}).Where("user_id = ? AND is_read = 0", uid).Update("is_read", 1)
	resp.OK(c, nil)
}
