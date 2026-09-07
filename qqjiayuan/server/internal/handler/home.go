package handler

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

type HomeHandler struct{ DB *gorm.DB }

// addHomeNews 写新鲜事（诺哈 wap_news）
func addHomeNews(db *gorm.DB, uid, nid uint, ntype int, refID uint, content string) {
	db.Create(&model.HomeNews{UserID: uid, NID: nid, NType: ntype, RefID: refID, Content: content})
}

// applyInvite 使用邀请码：标记邀请关系 + 双方各 50 G币奖励（诺哈 promo）
func applyInvite(db *gorm.DB, user *model.User, code string) {
	var inv model.Invite
	if err := db.Where("code = ? AND status = 0", code).First(&inv).Error; err != nil {
		return
	}
	// 禁止自我邀请
	if inv.UserID == user.ID {
		return
	}
	now := time.Now()
	db.Model(&inv).Updates(map[string]interface{}{"status": 1, "used_uid": user.ID, "used_at": now})
	db.Model(user).Update("invited_by", inv.UserID)
	db.Model(&model.User{}).Where("id = ?", user.ID).Update("coins", gorm.Expr("coins + 50"))
	db.Model(&model.User{}).Where("id = ?", inv.UserID).Update("coins", gorm.Expr("coins + 50"))
	addWalletLog(db, user.ID, "invite", "受邀注册奖励", "coins", 50)
	addWalletLog(db, inv.UserID, "invite", "邀请新人奖励", "coins", 50)
	db.Create(&model.Notification{UserID: inv.UserID, Type: "system", Title: "邀请成功",
		Content: "你邀请的「" + user.Nickname + "」已注册家园，50 G币奖励已到账！"})
	db.Create(&model.Notification{UserID: user.ID, Type: "system", Title: "受邀注册",
		Content: "通过邀请码注册成功，50 G币见面礼已到账，快去逛广场吧！"})
	inviteLink := "invite_" + code
	db.Where(&model.Setting{Key: inviteLink}).FirstOrCreate(&model.Setting{Key: inviteLink, Value: "1"})
}

// ensureHome 取/建家园统计行；当日首登活跃点+1（诺哈 my_home 每日+1活跃）
func (h *HomeHandler) ensureHome(uid uint) model.Home {
	var home model.Home
	if err := h.DB.Where("user_id = ?", uid).First(&home).Error; err != nil {
		home = model.Home{UserID: uid, Name: "我的家园"}
		h.DB.Create(&home)
	}
	today := time.Now().Format("2006-01-02")
	if home.LastActiveDate != today {
		h.DB.Model(&home).Updates(map[string]interface{}{"point": gorm.Expr("point + 1"), "last_active_date": today})
		home.Point++
		home.LastActiveDate = today
	}
	return home
}

func (h *HomeHandler) newsBrief(list []model.HomeNews) []gin.H {
	out := make([]gin.H, 0, len(list))
	for _, n := range list {
		var u model.User
		h.DB.Select("id,nickname,color").First(&u, n.UserID)
		out = append(out, gin.H{
			"id": n.ID, "user_id": n.UserID, "nickname": u.Nickname, "color": u.Color,
			"ntype": n.NType, "ref_id": n.RefID, "content": n.Content, "created_at": n.CreatedAt,
		})
	}
	return out
}

// NewsList 新鲜事列表（mine=我的 friend=好友的）
func (h *HomeHandler) NewsList(c *gin.Context) {
	uid := middleware.GetUID(c)
	scope := c.DefaultQuery("scope", "mine")
	page, size := pageParams(c, 10)
	q := h.DB.Model(&model.HomeNews{})
	if scope == "friend" {
		q = q.Where("user_id IN (SELECT oid FROM friendships WHERE uid = ? AND status = 1)", uid)
	} else {
		q = q.Where("user_id = ?", uid)
	}
	var total int64
	q.Count(&total)
	var list []model.HomeNews
	q.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&list)
	resp.OK(c, gin.H{"total": total, "page": page, "list": h.newsBrief(list)})
}

// View 我的家园聚合（诺哈 my_home.asp 区块）
func (h *HomeHandler) View(c *gin.Context) {
	uid := middleware.GetUID(c)
	home := h.ensureHome(uid)
	var u model.User
	h.DB.First(&u, uid)

	// 最新心情
	var mood model.Mood
	h.DB.Where("user_id = ? AND status = 1", uid).Order("created_at DESC").First(&mood)

	// 我的新鲜事 TOP5
	var myNews []model.HomeNews
	h.DB.Where("user_id = ?", uid).Order("created_at DESC").Limit(5).Find(&myNews)

	// 好友新鲜事 TOP5
	var friendNews []model.HomeNews
	h.DB.Where("user_id IN (SELECT oid FROM friendships WHERE uid = ? AND status = 1)", uid).
		Order("created_at DESC").Limit(5).Find(&friendNews)

	// 访客 TOP5
	var visitors []gin.H
	var visitorTotal int64
	h.DB.Model(&model.Visitor{}).Where("owner_id = ?", uid).Count(&visitorTotal)
	var vs []model.Visitor
	h.DB.Where("owner_id = ?", uid).Order("created_at DESC").Limit(5).Find(&vs)
	for _, v := range vs {
		var vu model.User
		h.DB.Select("id,nickname,color").First(&vu, v.UserID)
		visitors = append(visitors, gin.H{"user_id": v.UserID, "nickname": vu.Nickname, "color": vu.Color, "time": v.CreatedAt})
	}

	// 留言 TOP2
	var msgs []model.SpaceMessage
	h.DB.Where("to_user_id = ? AND status = 1", uid).Order("created_at DESC").Limit(2).Find(&msgs)
	var msgTotal int64
	h.DB.Model(&model.SpaceMessage{}).Where("to_user_id = ? AND status = 1", uid).Count(&msgTotal)

	// 收藏数
	var favCount int64
	h.DB.Model(&model.HomeFavorite{}).Where("user_id = ?", uid).Count(&favCount)

	lv, need := homePointLevel(home.Point)
	resp.OK(c, gin.H{
		"home":     gin.H{"point": home.Point, "level": lv, "next_need": need, "visitors": home.Visitors, "messages": home.Messages, "moods": home.Moods},
		"today_first": home.LastActiveDate == time.Now().Format("2006-01-02"),
		"user":     gin.H{"id": u.ID, "username": u.Username, "nickname": u.Nickname, "color": u.Color, "level": u.Level, "noble": u.Noble, "coins": u.Coins},
		"mood":     gin.H{"id": mood.ID, "content": mood.Content, "created_at": mood.CreatedAt},
		"my_news":  h.newsBrief(myNews),
		"friend_news": h.newsBrief(friendNews),
		"visitors": visitors, "visitor_total": visitorTotal,
		"messages": msgs, "message_total": msgTotal,
		"favorite_count": favCount,
	})
}

// Other 他人家园（诺哈 home.asp）
func (h *HomeHandler) Other(c *gin.Context) {
	id, ok := positiveParam(c, "userId")
	if !ok {
		return
	}
	var u model.User
	if err := h.DB.First(&u, id).Error; err != nil {
		resp.NotFound(c, "该号码的主人还不存在")
		return
	}
	home := model.Home{}
	h.DB.Where("user_id = ?", id).First(&home)

	var mood model.Mood
	h.DB.Where("user_id = ? AND status = 1", id).Order("created_at DESC").First(&mood)

	var news []model.HomeNews
	h.DB.Where("user_id = ?", id).Order("created_at DESC").Limit(3).Find(&news)

	var visitors []gin.H
	var visitorTotal int64
	h.DB.Model(&model.Visitor{}).Where("owner_id = ?", id).Count(&visitorTotal)
	var vs []model.Visitor
	h.DB.Where("owner_id = ?", id).Order("created_at DESC").Limit(3).Find(&vs)
	for _, v := range vs {
		var vu model.User
		h.DB.Select("id,nickname,color").First(&vu, v.UserID)
		visitors = append(visitors, gin.H{"user_id": v.UserID, "nickname": vu.Nickname, "color": vu.Color, "time": v.CreatedAt})
	}

	var msgs []model.SpaceMessage
	h.DB.Where("to_user_id = ? AND status = 1 AND (mtype = 0 OR from_user_id = ?)", id, middleware.GetUID(c)).
		Order("created_at DESC").Limit(2).Find(&msgs)
	var msgTotal int64
	h.DB.Model(&model.SpaceMessage{}).Where("to_user_id = ? AND status = 1", id).Count(&msgTotal)

	lv, need := homePointLevel(home.Point)
	uid := middleware.GetUID(c)
	isFriend := false
	if uid > 0 && uid != u.ID {
		var cnt int64
		h.DB.Model(&model.Friendship{}).Where("uid = ? AND oid = ? AND status = 1", uid, u.ID).Count(&cnt)
		isFriend = cnt > 0
	}
	online := u.LastActiveAt != nil && time.Since(*u.LastActiveAt) < 30*time.Minute
	resp.OK(c, gin.H{
		"home":  gin.H{"point": home.Point, "level": lv, "next_need": need, "visitors": home.Visitors, "messages": home.Messages},
		"user":  gin.H{"id": u.ID, "username": u.Username, "nickname": u.Nickname, "color": u.Color, "level": u.Level, "noble": u.Noble, "signature": u.Signature, "introduction": u.Introduction, "gender": u.Gender, "online": online, "is_friend": isFriend},
		"mood":  gin.H{"id": mood.ID, "content": mood.Content, "created_at": mood.CreatedAt},
		"news":  h.newsBrief(news),
		"visitors": visitors, "visitor_total": visitorTotal,
		"messages": msgs, "message_total": msgTotal,
	})
}

// Visit 串门：按家园号码找人（诺哈“串门”表单）
func (h *HomeHandler) Visit(c *gin.Context) {
	no := c.Query("no")
	if no == "" {
		resp.ParamError(c, "请输入要串门的家园号码")
		return
	}
	var u model.User
	if err := h.DB.Where("username = ?", no).First(&u).Error; err != nil {
		resp.ParamError(c, "找不到号码为 " + no + " 的居民，检查一下再试")
		return
	}
	resp.OK(c, gin.H{"user_id": u.ID, "username": u.Username, "nickname": u.Nickname})
}

// ============ 我的收藏（诺哈 wap_bbs_favor 泛化） ============

// FavList 收藏列表
func (h *HomeHandler) FavList(c *gin.Context) {
	uid := middleware.GetUID(c)
	q := h.DB.Where("user_id = ?", uid)
	if t := c.Query("f_type"); t != "" {
		q = q.Where("f_type = ?", t)
	}
	var list []model.HomeFavorite
	q.Order("created_at DESC").Limit(100).Find(&list)
	resp.OK(c, list)
}

// FavAdd 添加收藏
func (h *HomeHandler) FavAdd(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		FType int    `json:"f_type" binding:"required,min=1,max=6"`
		RefID uint   `json:"ref_id" binding:"required,min=1"`
		Name  string `json:"name" binding:"required,max=100"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "收藏参数不完整")
		return
	}
	var cnt int64
	h.DB.Model(&model.HomeFavorite{}).Where("user_id = ? AND f_type = ? AND ref_id = ?", uid, req.FType, req.RefID).Count(&cnt)
	if cnt > 0 {
		resp.OK(c, nil)
		return
	}
	h.DB.Create(&model.HomeFavorite{UserID: uid, FType: req.FType, RefID: req.RefID, Name: req.Name})
	resp.OK(c, nil)
}

// FavDel 删除收藏
func (h *HomeHandler) FavDel(c *gin.Context) {
	uid := middleware.GetUID(c)
	id, ok := positiveParam(c, "id")
	if !ok {
		return
	}
	h.DB.Where("id = ? AND user_id = ?", id, uid).Delete(&model.HomeFavorite{})
	resp.OK(c, nil)
}

// ============ 邀请开通家园（诺哈 invite.asp + promo 推荐奖励） ============

// InviteInfo 我的邀请码与邀请记录
func (h *HomeHandler) InviteInfo(c *gin.Context) {
	uid := middleware.GetUID(c)
	code := fmt.Sprintf("JY%d", uid)
	// 懒创建邀请码记录，保证 register 可凭码反查
	var inv model.Invite
	if err := h.DB.Where("user_id = ?", uid).First(&inv).Error; err != nil {
		h.DB.Create(&model.Invite{UserID: uid, Code: code, Status: 0})
	}
	var used []model.User
	h.DB.Select("id,username,nickname,created_at").Where("invited_by = ?", uid).Order("created_at DESC").Limit(50).Find(&used)
	out := make([]gin.H, 0, len(used))
	for _, u := range used {
		out = append(out, gin.H{"id": u.ID, "username": u.Username, "nickname": u.Nickname, "created_at": u.CreatedAt})
	}
	// 奖励规则文案（诺哈：被邀请人经验达标后发 preward）
	var total int64
	h.DB.Model(&model.User{}).Where("invited_by = ?", uid).Count(&total)
	resp.OK(c, gin.H{
		"code": code, "link": "/register?invite=" + code,
		"invited":   out, "invited_count": total,
		"reward":    "每成功邀请 1 位新居民注册，你获得 50 G币 奖励；对方也额外得 50 G币 见面礼。",
		"rule_text": "邀请你开通家园……用说说分享心情，用日志记录感悟，用照片记录生活。",
	})
}

