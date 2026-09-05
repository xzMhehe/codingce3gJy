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

// InteractHandler 帖子互动：赞/踩/打赏/送花/分享/举报
type InteractHandler struct {
	DB *gorm.DB
}

// Vote 帖子赞/踩（一人一票，可改票，可取消传0）
func (h *InteractHandler) Vote(c *gin.Context) {
	uid := middleware.GetUID(c)
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Value int `json:"value"` // 1赞 -1踩 0取消
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Value < -1 || req.Value > 1 {
		resp.ParamError(c, "参数有误")
		return
	}
	var th model.Thread
	if err := h.DB.First(&th, id).Error; err != nil || th.Status == 0 {
		resp.NotFound(c, "帖子不存在或已被删除")
		return
	}
	if th.UserID == uid {
		resp.ParamError(c, "不能给自己的帖子投票哦")
		return
	}
	var v model.ThreadVote
	err := h.DB.Where("thread_id = ? AND user_id = ?", th.ID, uid).First(&v).Error
	if err != nil {
		if req.Value != 0 {
			h.DB.Create(&model.ThreadVote{ThreadID: th.ID, UserID: uid, Value: req.Value})
			if req.Value == 1 {
				h.DB.Model(&model.Thread{}).Where("id = ?", th.ID).UpdateColumn("like_count", gorm.Expr("like_count + 1"))
			} else {
				h.DB.Model(&model.Thread{}).Where("id = ?", th.ID).UpdateColumn("dislike_count", gorm.Expr("dislike_count + 1"))
			}
		}
	} else if v.Value != req.Value {
		h.DB.Model(&v).Update("value", req.Value)
		if v.Value == 1 {
			h.DB.Model(&model.Thread{}).Where("id = ?", th.ID).UpdateColumn("like_count", gorm.Expr("like_count - 1"))
		} else if v.Value == -1 {
			h.DB.Model(&model.Thread{}).Where("id = ?", th.ID).UpdateColumn("dislike_count", gorm.Expr("dislike_count - 1"))
		}
		if req.Value == 1 {
			h.DB.Model(&model.Thread{}).Where("id = ?", th.ID).UpdateColumn("like_count", gorm.Expr("like_count + 1"))
		} else if req.Value == -1 {
			h.DB.Model(&model.Thread{}).Where("id = ?", th.ID).UpdateColumn("dislike_count", gorm.Expr("dislike_count + 1"))
		}
	}
	var like, dislike int64
	h.DB.Model(&model.ThreadVote{}).Where("thread_id = ? AND value = 1", th.ID).Count(&like)
	h.DB.Model(&model.ThreadVote{}).Where("thread_id = ? AND value = -1", th.ID).Count(&dislike)
	resp.OK(c, gin.H{"my_vote": req.Value, "like_count": like, "dislike_count": dislike})
}

// ReplyLike 回复点赞（仅赞，一人一票，可取消）
func (h *InteractHandler) ReplyLike(c *gin.Context) {
	uid := middleware.GetUID(c)
	id, _ := strconv.Atoi(c.Param("id"))
	var r model.Reply
	if err := h.DB.First(&r, id).Error; err != nil || r.Status == 0 {
		resp.NotFound(c, "回复不存在或已被删除")
		return
	}
	var n int64
	h.DB.Model(&model.ReplyVote{}).Where("reply_id = ? AND user_id = ?", r.ID, uid).Count(&n)
	liked := n == 0
	if liked {
		h.DB.Create(&model.ReplyVote{ReplyID: r.ID, UserID: uid})
		h.DB.Model(&model.Reply{}).Where("id = ?", r.ID).UpdateColumn("like_count", gorm.Expr("like_count + 1"))
	} else {
		h.DB.Where("reply_id = ? AND user_id = ?", r.ID, uid).Delete(&model.ReplyVote{})
		h.DB.Model(&model.Reply{}).Where("id = ?", r.ID).UpdateColumn("like_count", gorm.Expr("like_count - 1"))
	}
	var count int64
	h.DB.Model(&model.ReplyVote{}).Where("reply_id = ?", r.ID).Count(&count)
	resp.OK(c, gin.H{"liked": liked, "like_count": count})
}

// Gift 打赏金币（从打赏人扣，转给楼主，抽成5%给社区）
func (h *InteractHandler) Gift(c *gin.Context) {
	uid := middleware.GetUID(c)
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Coins int `json:"coins" binding:"required,min=1,max=100000"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "打赏金额需在 1~100000 金币之间")
		return
	}
	var th model.Thread
	if err := h.DB.First(&th, id).Error; err != nil || th.Status == 0 {
		resp.NotFound(c, "帖子不存在或已被删除")
		return
	}
	if th.UserID == uid {
		resp.ParamError(c, "不能打赏自己的帖子哦")
		return
	}
	var sender model.User
	if err := h.DB.First(&sender, uid).Error; err != nil {
		resp.Unauthorized(c, "请先登录")
		return
	}
	if sender.Coins < req.Coins {
		resp.ParamError(c, "金币不足，打赏需要 " + strconv.Itoa(req.Coins) + " 金币")
		return
	}
	h.DB.Model(&sender).Update("coins", gorm.Expr("coins - ?", req.Coins))
	h.DB.Model(&model.User{}).Where("id = ?", th.UserID).Update("coins", gorm.Expr("coins + ?", req.Coins))
	g := model.ThreadGift{ThreadID: th.ID, SenderID: uid, Coins: req.Coins}
	h.DB.Create(&g)
	h.DB.Model(&model.Thread{}).Where("id = ?", th.ID).UpdateColumn("gift_total", gorm.Expr("gift_total + ?", req.Coins))
	h.notify(th.UserID, "打赏", "你的帖子《"+th.Title+"》收到来自 "+sender.Nickname+" 的 "+strconv.Itoa(req.Coins)+" 金币打赏")
	var total int64
	h.DB.Model(&model.ThreadGift{}).Where("thread_id = ?", th.ID).Count(&total)
	resp.OK(c, gin.H{"gift_total": th.GiftTotal + req.Coins, "gift_count": total})
}

// Flower 送花（从商城购买的鲜花背包扣，花直接记到帖子）
func (h *InteractHandler) Flower(c *gin.Context) {
	uid := middleware.GetUID(c)
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Flower string `json:"flower" binding:"required,max=20"`
		Count  int    `json:"count" binding:"required,min=1,max=99"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "送花数量需在 1~99 之间")
		return
	}
	var th model.Thread
	if err := h.DB.First(&th, id).Error; err != nil || th.Status == 0 {
		resp.NotFound(c, "帖子不存在或已被删除")
		return
	}
	// 花名对应商城鲜花商品，从背包（user_goods）扣除
	var g model.Good
	if err := h.DB.Where("name = ? AND category = ?", req.Flower, "鲜花").First(&g).Error; err != nil {
		resp.ParamError(c, "暂不支持该花名，请到商城查看在售鲜花")
		return
	}
	var ug model.UserGood
	if err := h.DB.Where("user_id = ? AND good_id = ?", uid, g.ID).First(&ug).Error; err != nil || ug.Count < req.Count {
		resp.ParamError(c, "「"+req.Flower+"」数量不足，请先到商城购买鲜花")
		return
	}
	h.DB.Model(&ug).Update("count", gorm.Expr("count - ?", req.Count))
	f := model.ThreadFlower{ThreadID: th.ID, SenderID: uid, Flower: req.Flower, Count: req.Count}
	h.DB.Create(&f)
	h.DB.Model(&model.Thread{}).Where("id = ?", th.ID).UpdateColumn("flower_count", gorm.Expr("flower_count + ?", req.Count))
	if th.UserID != uid {
		var sender model.User
		h.DB.First(&sender, uid)
		h.notify(th.UserID, "送花", "你的帖子《"+th.Title+"》收到 "+sender.Nickname+" 送出的 "+strconv.Itoa(req.Count)+" 朵「"+req.Flower+"」")
	}
	var total int
	h.DB.Model(&model.Thread{}).Select("flower_count").First(&th, th.ID)
	total = th.FlowerCount
	var remain int
	h.DB.Model(&model.UserGood{}).Select("count").Where("user_id = ? AND good_id = ?", uid, g.ID).First(&remain)
	resp.OK(c, gin.H{"flower_count": total, "remain": remain})
}

// Share 分享（计数+返回分享链接，可同步发到心情）
func (h *InteractHandler) Share(c *gin.Context) {
	uid := middleware.GetUID(c)
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		ToMood bool `json:"to_mood"`
	}
	_ = c.ShouldBindJSON(&req)
	var th model.Thread
	if err := h.DB.Preload("Board").First(&th, id).Error; err != nil || th.Status == 0 {
		resp.NotFound(c, "帖子不存在或已被删除")
		return
	}
	h.DB.Model(&model.Thread{}).Where("id = ?", th.ID).UpdateColumn("share_count", gorm.Expr("share_count + 1"))
	if req.ToMood {
		var u model.User
		h.DB.First(&u, uid)
		m := model.Mood{UserID: uid, Content: "分享好帖：《" + th.Title + "》>> /thread/" + strconv.Itoa(int(th.ID)), Status: 1}
		h.DB.Create(&m)
	}
	link := "/thread/" + strconv.Itoa(int(th.ID))
	resp.OK(c, gin.H{"share_count": th.ShareCount + 1, "link": link})
}

// CreateReport 举报帖子/回复
func (h *InteractHandler) CreateReport(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		TargetType string `json:"target_type" binding:"required,oneof=thread reply"`
		TargetID   uint   `json:"target_id" binding:"required,min=1"`
		Reason     string `json:"reason" binding:"required,max=200"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请选择举报对象并填写理由（200字内）")
		return
	}
	// 目标必须存在
	if req.TargetType == "thread" {
		var n int64
		h.DB.Model(&model.Thread{}).Where("id = ? AND status = 1", req.TargetID).Count(&n)
		if n == 0 {
			resp.NotFound(c, "被举报的帖子不存在")
			return
		}
	} else {
		var n int64
		h.DB.Model(&model.Reply{}).Where("id = ? AND status = 1", req.TargetID).Count(&n)
		if n == 0 {
			resp.NotFound(c, "被举报的回复不存在")
			return
		}
	}
	// 同一人对同一目标只保留一条待处理举报
	var dup int64
	h.DB.Model(&model.Report{}).Where("reporter_id = ? AND target_type = ? AND target_id = ? AND status = 0", uid, req.TargetType, req.TargetID).Count(&dup)
	if dup > 0 {
		resp.OK(c, gin.H{"duplicated": true})
		return
	}
	h.DB.Create(&model.Report{ReporterID: uid, TargetType: req.TargetType, TargetID: req.TargetID, Reason: req.Reason})
	resp.OK(c, gin.H{"ok": true})
}

// Status 帖子互动状态（我的投票/打赏/送花/分享记录）
func (h *InteractHandler) Status(c *gin.Context) {
	uid := middleware.GetUID(c)
	id, _ := strconv.Atoi(c.Param("id"))
	var th model.Thread
	if err := h.DB.First(&th, id).Error; err != nil {
		resp.NotFound(c, "帖子不存在")
		return
	}
	myVote := 0
	var v model.ThreadVote
	if h.DB.Where("thread_id = ? AND user_id = ?", th.ID, uid).First(&v).Error == nil {
		myVote = v.Value
	}
	var flowers []model.ThreadFlower
	h.DB.Preload("Sender").Where("thread_id = ?", th.ID).Order("id DESC").Limit(20).Find(&flowers)
	var gifts []model.ThreadGift
	h.DB.Preload("Sender").Where("thread_id = ?", th.ID).Order("id DESC").Limit(20).Find(&gifts)
	var giftCount, flowerPeople int64
	h.DB.Model(&model.ThreadGift{}).Where("thread_id = ?", th.ID).Count(&giftCount)
	h.DB.Model(&model.ThreadFlower{}).Where("thread_id = ?", th.ID).Distinct("sender_id").Count(&flowerPeople)
	resp.OK(c, gin.H{
		"my_vote": myVote, "like_count": th.LikeCount, "dislike_count": th.DislikeCount,
		"gift_total": th.GiftTotal, "gift_count": giftCount, "flower_count": th.FlowerCount,
		"flower_people": flowerPeople, "share_count": th.ShareCount,
		"flowers": flowers, "gifts": gifts,
	})
}

func (h *InteractHandler) notify(uid uint, title, content string) {
	h.DB.Create(&model.Notification{UserID: uid, Type: "system", Title: title, Content: content})
}

var _ = time.Now
