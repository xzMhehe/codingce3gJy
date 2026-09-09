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

// 我的好友：返回好友列表（含备注/分组/在线）、分组、待处理申请、黑名单数、验证设置
func (h *FriendHandler) List(c *gin.Context) {
	uid := middleware.GetUID(c)

	type friendInfo struct {
		ID        uint   `json:"id"`
		Nickname  string `json:"nickname"`
		Remark    string `json:"remark"`
		Color     string `json:"color"`
		Level     int    `json:"level"`
		Sign      string `json:"signature"`
		GroupID   uint   `json:"group_id"`
		GroupName string `json:"group_name"`
		Degree    int    `json:"degree"`
		Online    bool   `json:"online"`
		CreatedAt string `json:"created_at"`
	}
	var friends []friendInfo
	h.DB.Raw(`
SELECT f.id, u.id AS ID, u.nickname, u.color, u.level, u.signature, f.remark, f.group_id, f.degree,
       IFNULL(g.name,'') AS group_name, f.created_at,
       CASE WHEN u.last_active_at IS NOT NULL AND u.last_active_at > (NOW() - INTERVAL 10 MINUTE) THEN 1 ELSE 0 END AS online
FROM friendships f
JOIN users u ON u.id = f.friend_id
LEFT JOIN friend_groups g ON g.id = f.group_id AND g.user_id = f.user_id
WHERE f.user_id = ? AND f.status = 1
ORDER BY f.degree DESC, f.id DESC`, uid).Scan(&friends)

	// 分组（含数量）
	var groups []model.FriendGroup
	h.DB.Where("user_id = ?", uid).Order("sort ASC, id ASC").Find(&groups)

	// 待处理的申请（诺哈 wap_friend_apply：uid=接收方=我）
	var applies []model.FriendApply
	h.DB.Where("user_id = ?", uid).Order("id ASC").Find(&applies)
	requests := []gin.H{}
	for _, a := range applies {
		var u model.User
		if err := h.DB.First(&u, a.FriendID).Error; err == nil {
			requests = append(requests, gin.H{"apply_id": a.ID, "id": u.ID, "nickname": u.Nickname,
				"color": u.Color, "level": u.Level, "remark": a.Remark, "created_at": a.CreatedAt})
		}
	}

	// 验证设置
	var me model.User
	h.DB.First(&me, uid)

	// 黑名单
	var blackCount int64
	h.DB.Model(&model.FriendBlack{}).Where("user_id = ? AND end_time > ?", uid, time.Now()).Count(&blackCount)

	resp.OK(c, gin.H{
		"friends": friends, "requests": requests, "groups": groups,
		"friend_policy": me.FriendPolicy, "black_count": blackCount,
	})
}

// 查询单个好友关系（供资料页判断是否已是好友/申请状态）
func (h *FriendHandler) Status(c *gin.Context) {
	uid := middleware.GetUID(c)
	peerID, _ := strconv.Atoi(c.Param("id"))
	var fr model.Friendship
	err := h.DB.Where("user_id = ? AND friend_id = ?", uid, peerID).First(&fr).Error
	if err == nil {
		if fr.Status == 1 {
			resp.OK(c, gin.H{"state": "friend", "remark": fr.Remark, "group_id": fr.GroupID, "degree": fr.Degree})
			return
		}
	}
	// 是否收到对方申请
	var n int64
	h.DB.Model(&model.FriendApply{}).Where("user_id = ? AND friend_id = ?", uid, peerID).Count(&n)
	if n > 0 {
		resp.OK(c, gin.H{"state": "incoming"})
		return
	}
	var n2 int64
	h.DB.Model(&model.FriendApply{}).Where("user_id = ? AND friend_id = ?", peerID, uid).Count(&n2)
	if n2 > 0 {
		resp.OK(c, gin.H{"state": "applied"})
		return
	}
	var bl int64
	h.DB.Model(&model.FriendBlack{}).Where("user_id = ? AND friend_id = ? AND end_time > ?", uid, peerID, time.Now()).Count(&bl)
	if bl > 0 {
		resp.OK(c, gin.H{"state": "blacked"})
		return
	}
	resp.OK(c, gin.H{"state": "none"})
}

type addFriendReq struct {
	TargetID uint   `json:"target_id" binding:"required"`
	Remark   string `json:"remark"` // 验证信息（需要验证时）
}

// 添加好友（对齐诺哈 friend_add_ok.asp）
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
	if err := h.DB.First(&target, req.TargetID).Error; err != nil || target.Status == 0 {
		resp.NotFound(c, "这位友友不存在")
		return
	}
	var me model.User
	h.DB.First(&me, uid)

	// 已在黑名单
	var meBlack int64
	h.DB.Model(&model.FriendBlack{}).Where("user_id = ? AND friend_id = ? AND end_time > ?", uid, req.TargetID, time.Now()).Count(&meBlack)
	if meBlack > 0 {
		resp.Forbidden(c, "你已将该友友拉黑")
		return
	}
	var targetBlack int64
	h.DB.Model(&model.FriendBlack{}).Where("user_id = ? AND friend_id = ? AND end_time > ?", req.TargetID, uid, time.Now()).Count(&targetBlack)
	if targetBlack > 0 {
		resp.Forbidden(c, "对方已将你拉黑")
		return
	}

	// 已是好友
	var exist model.Friendship
	if err := h.DB.Where("user_id = ? AND friend_id = ?", uid, req.TargetID).First(&exist).Error; err == nil {
		if exist.Status == 1 {
			resp.ParamError(c, "你们已经是好友啦")
			return
		}
	}
	// 已申请过（对侧存在我的申请）
	var mine int64
	h.DB.Model(&model.FriendApply{}).Where("user_id = ? AND friend_id = ?", req.TargetID, uid).Count(&mine)
	if mine > 0 {
		resp.ParamError(c, "申请已在路上，等待对方处理")
		return
	}

	// 加好友策略（诺哈 wap_user.friend：0允许 1需要验证 2拒绝）
	if target.FriendPolicy == 2 {
		resp.Forbidden(c, "对方设置了不接受好友申请")
		return
	}
	if target.FriendPolicy == 1 {
		// 需要验证：写入申请记录（uid=接收方，friend_id=我）
		h.DB.Create(&model.FriendApply{UserID: req.TargetID, FriendID: uid, Remark: left(req.Remark, 100)})
		h.DB.Create(&model.Notification{
			UserID: req.TargetID, Type: "friend", RefID: uid,
			Title:   me.Nickname + " 请求加你为好友",
			Content: left(req.Remark, 80) + "（到「好友」页处理）",
		})
		resp.OK(c, "验证信息已发送，等待对方确认")
		return
	}

	// 允许所有人添加：直接建立好友关系（我→TA），并通知对方可互加
	h.becomeFriend(uid, req.TargetID)
	h.DB.Create(&model.Notification{
		UserID: req.TargetID, Type: "friend", RefID: uid,
		Title:   me.Nickname + " 添加你为好友",
		Content: "你多了一位新朋友啦，去打个招呼吧",
	})
	resp.OK(c, "添加好友成功")
}

// becomeFriend 单向建立好友关系（对齐诺哈：一方添加，仅在本方向插入一条）
func (h *FriendHandler) becomeFriend(uid, target uint) {
	var n int64
	h.DB.Model(&model.Friendship{}).Where("user_id = ? AND friend_id = ?", uid, target).Count(&n)
	if n > 0 {
		h.DB.Model(&model.Friendship{}).Where("user_id = ? AND friend_id = ?", uid, target).Update("status", 1)
		return
	}
	h.DB.Create(&model.Friendship{UserID: uid, FriendID: target, Status: 1})
	syncOneGroupAmount(h.DB, uid)
}

// 处理好友申请（对齐诺哈 apply_ok.asp：互加/通过/拒绝/忽略）
func (h *FriendHandler) Handle(c *gin.Context) {
	uid := middleware.GetUID(c)
	id, _ := strconv.Atoi(c.Param("id"))
	act := c.DefaultQuery("act", "add")
	var fr model.FriendApply
	if err := h.DB.First(&fr, id).Error; err != nil {
		resp.NotFound(c, "申请不存在")
		return
	}
	if fr.UserID != uid {
		resp.Forbidden(c, "这不是发给你的申请")
		return
	}
	var me model.User
	h.DB.First(&me, uid)
	h.DB.Delete(&fr) // 无论哪种处理，申请即失效

	// 对方是否已把我拉黑
	var targetBlack int64
	h.DB.Model(&model.FriendBlack{}).Where("user_id = ? AND friend_id = ? AND end_time > ?", fr.FriendID, uid, time.Now()).Count(&targetBlack)

	switch act {
	case "reject": // 拒绝
		h.DB.Create(&model.Notification{UserID: fr.FriendID, Type: "friend", RefID: uid,
			Title: me.Nickname + " 拒绝了你的好友请求", Content: "不要太伤心，可以再试试"})
		resp.OK(c, "已拒绝")
	case "ignore": // 忽略
		resp.OK(c, "已忽略")
	case "pass": // 仅通过（对方加我为好友），不再互加
		if targetBlack > 0 {
			resp.Forbidden(c, "对方已将你拉黑，无法建立好友关系")
			return
		}
		h.becomeFriend(fr.FriendID, uid)
		h.DB.Create(&model.Notification{UserID: fr.FriendID, Type: "friend", RefID: uid,
			Title: me.Nickname + " 通过了你的好友请求", Content: "你们已成为好友"})
		resp.OK(c, "已通过")
	default: // 互加：既通过又添加对方
		if targetBlack > 0 {
			resp.Forbidden(c, "对方已将你拉黑，无法建立好友关系")
			return
		}
		h.becomeFriend(fr.FriendID, uid) // 对方→我
		h.becomeFriend(uid, fr.FriendID) // 我→对方（互加）
		h.DB.Create(&model.Notification{UserID: fr.FriendID, Type: "friend", RefID: uid,
			Title: me.Nickname + " 通过了你的好友请求，并添加你为好友", Content: "你们已成为好友"})
		resp.OK(c, "已通过并添加对方为好友")
	}
}

// 设置好友备注（对齐诺哈 wap_friend.name）
func (h *FriendHandler) SetRemark(c *gin.Context) {
	uid := middleware.GetUID(c)
	peerID, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Remark string `json:"remark" binding:"required,max=30"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "备注 1-30 个字")
		return
	}
	var fr model.Friendship
	if err := h.DB.Where("user_id = ? AND friend_id = ? AND status = 1", uid, peerID).First(&fr).Error; err != nil {
		resp.NotFound(c, "你们还不是好友")
		return
	}
	h.DB.Model(&fr).Update("remark", req.Remark)
	resp.OK(c, "备注已更新")
}

// 移动好友到分组（对齐诺哈 friend_group_move_ok.asp）
func (h *FriendHandler) MoveGroup(c *gin.Context) {
	uid := middleware.GetUID(c)
	peerID, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		GroupID uint `json:"group_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请选择分组")
		return
	}
	var fr model.Friendship
	if err := h.DB.Where("user_id = ? AND friend_id = ? AND status = 1", uid, peerID).First(&fr).Error; err != nil {
		resp.NotFound(c, "你们还不是好友")
		return
	}
	if req.GroupID != 0 {
		var g model.FriendGroup
		if err := h.DB.Where("id = ? AND user_id = ?", req.GroupID, uid).First(&g).Error; err != nil {
			resp.NotFound(c, "分组不存在")
			return
		}
	}
	oldGroup := fr.GroupID
	h.DB.Model(&fr).Update("group_id", req.GroupID)
	// 更新两个分组数量
	dbUpdateGroupAmount(h.DB, uid, oldGroup)
	dbUpdateGroupAmount(h.DB, uid, req.GroupID)
	resp.OK(c, "移动成功")
}

// 删除好友（对齐诺哈 friend_del_ok.asp：仅解除本方向）
func (h *FriendHandler) Remove(c *gin.Context) {
	uid := middleware.GetUID(c)
	peerID, _ := strconv.Atoi(c.Param("id"))
	var fr model.Friendship
	if err := h.DB.Where("user_id = ? AND friend_id = ? AND status = 1", uid, peerID).First(&fr).Error; err != nil {
		resp.NotFound(c, "你们还不是好友")
		return
	}
	oldGroup := fr.GroupID
	h.DB.Delete(&fr)
	dbUpdateGroupAmount(h.DB, uid, oldGroup)
	resp.OK(c, "已删除")
}

// 设置加好友策略（对齐诺哈 chat/config.asp）
func (h *FriendHandler) SetPolicy(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Policy int `json:"friend_policy" binding:"oneof=0 1 2"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "friend_policy 必须是 0/1/2")
		return
	}
	h.DB.Model(&model.User{}).Where("id = ?", uid).Update("friend_policy", req.Policy)
	resp.OK(c, "设置成功")
}

// dbUpdateGroupAmount 重新计算某分组好友数
func dbUpdateGroupAmount(db *gorm.DB, uid, gid uint) {
	var n int64
	db.Model(&model.Friendship{}).Where("user_id = ? AND group_id = ? AND status = 1", uid, gid).Count(&n)
	db.Model(&model.FriendGroup{}).Where("id = ? AND user_id = ?", gid, uid).Update("amount", n)
}

// syncOneGroupAmount 同步某个用户所有分组数量
func syncOneGroupAmount(db *gorm.DB, uid uint) {
	var groups []model.FriendGroup
	db.Where("user_id = ?", uid).Find(&groups)
	for _, g := range groups {
		var n int64
		db.Model(&model.Friendship{}).Where("user_id = ? AND group_id = ? AND status = 1", uid, g.ID).Count(&n)
		db.Model(&model.FriendGroup{}).Where("id = ?", g.ID).Update("amount", n)
	}
}

func left(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
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

// 收信箱（对齐诺哈 inbox.asp）：收到的私信列表，点击进入往来
func (h *MessageHandler) Inbox(c *gin.Context) {
	uid := middleware.GetUID(c)
	page, offset, size := pageOf(c, 10)
	var total int64
	h.DB.Model(&model.PrivateMessage{}).Where("receiver_id = ?", uid).Count(&total)
	var msgs []model.PrivateMessage
	h.DB.Preload("Sender").Where("receiver_id = ?", uid).Order("id DESC").Offset(offset).Limit(size).Find(&msgs)
	out := []gin.H{}
	for _, m := range msgs {
		nick, color := "系统信息", ""
		if m.Sender != nil {
			nick = m.Sender.Nickname
			color = m.Sender.Color
		}
		out = append(out, gin.H{"id": m.ID, "sender_id": m.SenderID, "sender": nick, "color": color,
			"content": m.Content, "is_read": m.IsRead, "created_at": m.CreatedAt})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// 发信箱（对齐诺哈 outbox.asp）：发出的私信列表
func (h *MessageHandler) Outbox(c *gin.Context) {
	uid := middleware.GetUID(c)
	page, offset, size := pageOf(c, 10)
	var total int64
	h.DB.Model(&model.PrivateMessage{}).Where("sender_id = ?", uid).Count(&total)
	var msgs []model.PrivateMessage
	h.DB.Where("sender_id = ?", uid).Order("id DESC").Offset(offset).Limit(size).Find(&msgs)
	out := []gin.H{}
	for _, m := range msgs {
		nick, color := "—", ""
		var r model.User
		if err := h.DB.First(&r, m.ReceiverID).Error; err == nil {
			nick = r.Nickname
			color = r.Color
		}
		out = append(out, gin.H{"id": m.ID, "receiver_id": m.ReceiverID, "receiver": nick, "color": color,
			"content": m.Content, "is_read": m.IsRead, "created_at": m.CreatedAt})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
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
	// 对齐诺哈 send.asp：黑名单屏蔽家信
	var black int64
	h.DB.Model(&model.FriendBlack{}).Where("user_id = ? AND friend_id = ? AND end_time > ?", req.To, uid, time.Now()).Count(&black)
	if black > 0 {
		resp.Forbidden(c, "对方不接受您的家信")
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
