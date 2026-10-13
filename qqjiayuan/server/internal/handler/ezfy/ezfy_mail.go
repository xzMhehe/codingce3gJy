package ezfy

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 二战风云 —— 游戏内邮件 / 好友私聊
//
// ★ 2026-10-13 邮箱改造（用户规则）：
//   邮箱四个分类：收件箱 / 系统邮件 / 发件箱 / 写邮件；**非好友之间用邮箱联系**；
//   好友之间的私聊放好友页，**强制仅好友**（后端校验，非好友只能走邮箱）。
//
// 接口：
//   GET  /games/ezfy/mail?tab=inbox|system|sent&page&size   邮件列表 + 未读数
//   POST /games/ezfy/mail/send   {to, subject, content}      发邮件（收件人昵称/游戏ID）
//   POST /games/ezfy/mail/read   {id}                        标记已读
//   POST /games/ezfy/mail/readall {tab}                      该分类全部已读
//   POST /games/ezfy/mail/delete {id}                        删除（只删自己视角）
//   POST /games/ezfy/privchat/send {to, content}             好友私聊发送（强制好友）

// ezfyMailPageMax 邮件列表每页最大条数
const ezfyMailPageMax = 50

// ezfyMailDailyLimit 24 小时内最多发送的邮件数（用户规则「24小时内最多50封」）
const ezfyMailDailyLimit = 50

// ezfyMailPager 邮件列表分页参数（页码/每页条数，做上限）
func ezfyMailPager(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > ezfyMailPageMax {
		size = 20
	}
	return page, size
}

// ezfyResolvePlayer 按「游戏ID 或 昵称」解析玩家
//
// 用户规则「收件人昵称须完全一致」：昵称必须**精确匹配**（不像好友搜索那样模糊 LIKE）。
// 数字 → 先按游戏ID（ezfy_profile.game_uid）；非数字 → 昵称精确匹配。
// 返回目标 user_id；0 = 没找到。
func (h *EzfyHandler) ezfyResolvePlayer(kw string) uint {
	kw = strings.TrimSpace(kw)
	if kw == "" {
		return 0
	}
	// 游戏ID（二战档案的 game_uid）
	if n, err := strconv.ParseInt(kw, 10, 64); err == nil {
		var prof model.EzfyProfile
		if err := h.DB.Where("game_uid = ?", n).First(&prof).Error; err == nil {
			return prof.UserID
		}
	}
	// 昵称精确匹配：二战档案昵称优先，兜底家园昵称
	var prof model.EzfyProfile
	if err := h.DB.Where("nickname = ?", kw).First(&prof).Error; err == nil {
		return prof.UserID
	}
	var u model.User
	if err := h.home().Where("nickname = ?", kw).First(&u).Error; err == nil {
		return u.ID
	}
	return 0
}

// ezfyMailCreate 写一封游戏内邮件（系统邮件 sender=0；内容非空才写，长度按字段上限截断）
func (h *EzfyHandler) ezfyMailCreate(sender, receiver uint, senderName, subject, content string) {
	content = trimStr(strings.TrimSpace(content), 500)
	if content == "" {
		return
	}
	if senderName == "" {
		senderName = "系统"
	}
	h.DB.Create(&model.EzfyMail{
		SenderId: sender, ReceiverId: receiver, SenderName: trimStr(senderName, 30),
		Subject: trimStr(strings.TrimSpace(subject), 20), Content: content,
	})
}

// MailList GET /games/ezfy/mail?tab=inbox|system|sent&page&size
func (h *EzfyHandler) MailList(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	tab := c.DefaultQuery("tab", "inbox")
	page, size := ezfyMailPager(c)

	var total int64
	var rows []model.EzfyMail
	base := h.DB.Model(&model.EzfyMail{})
	switch tab {
	case "system":
		base.Where("receiver_id = ? AND sender_id = 0 AND recv_del = 0", uid).Count(&total)
		h.DB.Where("receiver_id = ? AND sender_id = 0 AND recv_del = 0", uid).
			Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&rows)
	case "sent":
		base.Where("sender_id = ? AND sender_del = 0", uid).Count(&total)
		h.DB.Where("sender_id = ? AND sender_del = 0", uid).
			Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&rows)
	default: // inbox
		tab = "inbox"
		base.Where("receiver_id = ? AND sender_id <> 0 AND recv_del = 0", uid).Count(&total)
		h.DB.Where("receiver_id = ? AND sender_id <> 0 AND recv_del = 0", uid).
			Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&rows)
	}

	// 发件箱需要展示「发给谁」（按 receiver_id 批量取昵称，避免 N+1）
	receiverNames := map[uint]string{}
	if tab == "sent" && len(rows) > 0 {
		ids := []uint{}
		for _, m := range rows {
			if m.ReceiverId > 0 {
				ids = append(ids, m.ReceiverId)
			}
		}
		for _, id := range ids {
			b := h.ezfyFriendBrief(id)
			receiverNames[id] = b["nickname"].(string)
		}
	}

	views := make([]gin.H, 0, len(rows))
	for _, m := range rows {
		v := gin.H{"id": m.ID, "sender_id": m.SenderId, "sender_name": m.SenderName,
			"subject": m.Subject, "content": m.Content, "is_read": m.IsRead,
			"created_at": m.CreatedAt}
		if tab == "sent" {
			v["receiver_id"] = m.ReceiverId
			v["receiver_name"] = receiverNames[m.ReceiverId]
		}
		views = append(views, v)
	}

	// 未读数（收件箱 / 系统邮件 / 合计）
	var unIn, unSys int64
	h.DB.Model(&model.EzfyMail{}).
		Where("receiver_id = ? AND sender_id <> 0 AND is_read = 0 AND recv_del = 0", uid).Count(&unIn)
	h.DB.Model(&model.EzfyMail{}).
		Where("receiver_id = ? AND sender_id = 0 AND is_read = 0 AND recv_del = 0", uid).Count(&unSys)
	resp.OK(c, gin.H{"tab": tab, "list": views, "total": total, "page": page, "size": size,
		"unread": gin.H{"inbox": unIn, "system": unSys, "total": unIn + unSys}})
}

// MailSend POST /games/ezfy/mail/send {to, subject, content}
//
// 用户规则：收件人填「玩家昵称或游戏ID」；昵称须完全一致；24小时内最多50封。
func (h *EzfyHandler) MailSend(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	var req struct {
		To      string `json:"to"`
		Subject string `json:"subject"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	to := strings.TrimSpace(req.To)
	content := strings.TrimSpace(req.Content)
	if to == "" {
		resp.ParamError(c, "请填写收件人（玩家昵称或游戏ID）")
		return
	}
	if content == "" {
		resp.ParamError(c, "请填写邮件内容")
		return
	}
	if r := []rune(content); len(r) > 500 {
		resp.ParamError(c, "邮件内容最多 500 字")
		return
	}
	if r := []rune(req.Subject); len(r) > 20 {
		resp.ParamError(c, "邮件主题最多 20 字")
		return
	}
	// 敏感词过滤（复用聊天敏感词表，拦截类直接拒绝）
	if filtered, blocked := ezfyFilterChat(content); blocked {
		resp.Forbidden(c, "邮件内容包含敏感词，请修改后再发")
		return
	} else {
		content = filtered
	}
	target := h.ezfyResolvePlayer(to)
	if target == 0 {
		resp.NotFound(c, "收件人不存在，请确认昵称（须完全一致）或游戏ID")
		return
	}
	if target == uid {
		resp.ParamError(c, "不能给自己发邮件")
		return
	}
	// 24 小时最多 50 封（按发件人统计）
	var sent int64
	h.DB.Model(&model.EzfyMail{}).
		Where("sender_id = ? AND created_at >= ?", uid, time.Now().Add(-24*time.Hour)).Count(&sent)
	if sent >= ezfyMailDailyLimit {
		resp.ParamError(c, fmt.Sprintf("24小时内最多发送 %d 封邮件，请稍后再试", ezfyMailDailyLimit))
		return
	}
	me := h.ensureProfile(uid)
	myNick := me.Nickname
	if myNick == "" {
		var u model.User
		if err := h.home().First(&u, uid).Error; err == nil {
			myNick = u.Nickname
		}
	}
	h.ezfyMailCreate(uid, target, myNick, req.Subject, content)
	resp.OK(c, gin.H{"msg": "邮件已发送"})
}

// MailRead POST /games/ezfy/mail/read {id}
func (h *EzfyHandler) MailRead(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		ID uint `json:"id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.ID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	h.DB.Model(&model.EzfyMail{}).
		Where("id = ? AND receiver_id = ?", req.ID, uid).Update("is_read", 1)
	resp.OK(c, gin.H{})
}

// MailReadAll POST /games/ezfy/mail/readall {tab: inbox|system}
func (h *EzfyHandler) MailReadAll(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Tab string `json:"tab"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	q := h.DB.Model(&model.EzfyMail{}).Where("receiver_id = ? AND is_read = 0", uid)
	if req.Tab == "system" {
		q.Where("sender_id = 0")
	} else {
		q.Where("sender_id <> 0")
	}
	q.Update("is_read", 1)
	resp.OK(c, gin.H{"msg": "已全部标记为已读"})
}

// MailDelete POST /games/ezfy/mail/delete {id}
//
// 用户规则「删除只删自己视角的副本」：收件人视角置 recv_del，发件人视角置 sender_del，互不影响。
func (h *EzfyHandler) MailDelete(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		ID uint `json:"id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.ID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var m model.EzfyMail
	if err := h.DB.First(&m, req.ID).Error; err != nil {
		resp.NotFound(c, "邮件不存在")
		return
	}
	switch {
	case m.ReceiverId == uid:
		h.DB.Model(&model.EzfyMail{}).Where("id = ?", m.ID).Update("recv_del", 1)
		resp.OK(c, gin.H{"msg": "邮件已删除"})
	case m.SenderId == uid:
		h.DB.Model(&model.EzfyMail{}).Where("id = ?", m.ID).Update("sender_del", 1)
		resp.OK(c, gin.H{"msg": "邮件已删除"})
	default:
		resp.ParamError(c, "这不是你的邮件")
	}
}

// PrivChatSend POST /games/ezfy/privchat/send {to, content}
//
// ★ 2026-10-13 用户规则「好友之间私聊放好友页、非好友用邮箱联系」：
//   私聊**强制仅好友**——非好友之间不能私聊，请走邮箱。
//   私信仍落在 PrivateMessage（与聊天/会话历史同一张表），只加好友校验。
func (h *EzfyHandler) PrivChatSend(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		To      string `json:"to"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		resp.ParamError(c, "请填写内容")
		return
	}
	if r := []rune(content); len(r) > 500 {
		content = string(r[:500])
	}
	if filtered, blocked := ezfyFilterChat(content); blocked {
		resp.Forbidden(c, "消息包含敏感词，请修改后再发")
		return
	} else {
		content = filtered
	}
	target := h.ezfyResolvePlayer(strings.TrimSpace(req.To))
	if target == 0 {
		resp.NotFound(c, "找不到这位玩家（可填游戏ID或昵称）")
		return
	}
	if target == uid {
		resp.ParamError(c, "不能和自己私聊")
		return
	}
	if !h.ezfyIsFriend(uid, target) {
		resp.Forbidden(c, "只有游戏好友之间才能私聊，非好友请用「邮箱」联系")
		return
	}
	// 对方把我拉黑（复用家园黑名单）则发不出去
	var black int64
	h.DB.Model(&model.FriendBlack{}).
		Where("user_id = ? AND friend_id = ? AND end_time > ?", target, uid, time.Now()).Count(&black)
	if black > 0 {
		resp.Forbidden(c, "对方不接受你的私聊")
		return
	}
	h.DB.Create(&model.PrivateMessage{SenderID: uid, ReceiverID: target, Content: content})
	resp.OK(c, gin.H{"msg": "已发送"})
}
