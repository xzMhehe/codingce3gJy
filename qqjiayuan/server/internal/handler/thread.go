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

type ThreadHandler struct {
	DB *gorm.DB
}

// 发帖/回帖/签到收益：经验 + 金币 + 社区成就点
func addExpAndCoins(db *gorm.DB, uid uint, exp, coins, achieve int) {
	db.Model(&model.User{}).Where("id = ?", uid).
		Updates(map[string]interface{}{
			"exp":     gorm.Expr("exp + ?", exp),
			"coins":   gorm.Expr("coins + ?", coins),
			"achieve": gorm.Expr("achieve + ?", achieve),
		})
	var u model.User
	db.First(&u, uid)
	db.Model(&u).Update("level", model.CalcLevel(u.Exp))
}

// 帖子详情 + 分页楼层（1楼=楼主，回复从2楼起）
func (h *ThreadHandler) Detail(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	page, _ := pageOf(c, 10)

	var th model.Thread
	if err := h.DB.Preload("User").Preload("User.Badges").Preload("Board").First(&th, id).Error; err != nil || th.Status == 0 {
		resp.NotFound(c, "帖子不存在或已被删除")
		return
	}
	go func() {
		h.DB.Model(&model.Thread{}).Where("id = ?", th.ID).
			UpdateColumn("view_count", gorm.Expr("view_count + 1"))
	}()

	var replies []model.Reply
	var total int64
	h.DB.Model(&model.Reply{}).Where("thread_id = ? AND status = 1", th.ID).Count(&total)
	// 页码夹紧到有效范围
	if maxPage := int(total+9) / 10; maxPage < 1 {
		page = 1
	} else if page > maxPage {
		page = maxPage
	}
	h.DB.Preload("User").Preload("User.Badges").Where("thread_id = ? AND status = 1", th.ID).
		Order("floor ASC").Offset((page - 1) * 10).Limit(10).Find(&replies)

	// 互动统计（未登录也可看）
	var giftCount, flowerPeople int64
	h.DB.Model(&model.ThreadGift{}).Where("thread_id = ?", th.ID).Count(&giftCount)
	h.DB.Model(&model.ThreadFlower{}).Where("thread_id = ?", th.ID).Distinct("sender_id").Count(&flowerPeople)
	var flowers []model.ThreadFlower
	h.DB.Preload("Sender").Where("thread_id = ?", th.ID).Order("id DESC").Limit(20).Find(&flowers)
	var gifts []model.ThreadGift
	h.DB.Preload("Sender").Where("thread_id = ?", th.ID).Order("id DESC").Limit(20).Find(&gifts)

	resp.OK(c, gin.H{"thread": th, "replies": replies, "total": total, "page": page, "size": 10,
		"like_count": th.LikeCount, "dislike_count": th.DislikeCount,
		"gift_total": th.GiftTotal, "gift_count": giftCount, "flower_count": th.FlowerCount,
		"flower_people": flowerPeople, "share_count": th.ShareCount,
		"flowers": flowers, "gifts": gifts})
}

type replyReq struct {
	Content string `json:"content" binding:"required,min=1,max=2000"`
}

// 回复盖楼
func (h *ThreadHandler) Reply(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	uid := middleware.GetUID(c)
	var req replyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "回复内容不能为空，2000字以内")
		return
	}

	var th model.Thread
	if err := h.DB.First(&th, id).Error; err != nil || th.Status == 0 {
		resp.NotFound(c, "帖子不存在或已被删除")
		return
	}
	// 家族专属论坛板块：仅家族成员可回帖
	var board model.Board
	if h.DB.First(&board, th.BoardID).Error == nil {
		if famID, ok := familyBoardOwner(h.DB, board); ok && !isFamilyMember(h.DB, famID, uid) {
			resp.Forbidden(c, "只有家族成员才能在家族论坛回帖")
			return
		}
	}

	var reply model.Reply
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		var count int64
		tx.Model(&model.Reply{}).Where("thread_id = ?", id).Count(&count)
		reply = model.Reply{ThreadID: uint(id), UserID: uid, Content: req.Content, Floor: int(count) + 2}
		if err := tx.Create(&reply).Error; err != nil {
			return err
		}
		return tx.Model(&model.Thread{}).Where("id = ?", id).
			Updates(map[string]interface{}{
				"reply_count":   gorm.Expr("reply_count + 1"),
				"last_reply_at": time.Now(),
			}).Error
	})
	if err != nil {
		resp.ServerError(c, err)
		return
	}
	addExpAndCoins(h.DB, uid, 5, 2, 1)

	if th.UserID != uid {
		var me model.User
		h.DB.First(&me, uid)
		h.DB.Create(&model.Notification{
			UserID: th.UserID, Type: "reply", RefID: th.ID,
			Title:   me.Nickname + " 回复了你的帖子",
			Content: "《" + th.Title + "》来了新回复，快去看看吧",
		})
	}
	resp.OK(c, gin.H{"id": reply.ID, "floor": reply.Floor})
}

// 编辑帖子：作者本人，或持有 thread:manage 权限的管理员
func (h *ThreadHandler) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	uid := middleware.GetUID(c)
	var req threadReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "标题1-100字，内容不能为空")
		return
	}
	var th model.Thread
	if err := h.DB.First(&th, id).Error; err != nil || th.Status == 0 {
		resp.NotFound(c, "帖子不存在或已被删除")
		return
	}
	if th.UserID != uid {
		var perm int64
		h.DB.Raw(`SELECT COUNT(DISTINCT p.id)
FROM permissions p
JOIN role_permissions rp ON rp.permission_id = p.id
JOIN user_roles ur ON ur.role_id = rp.role_id
WHERE ur.user_id = ? AND p.code = ?`, uid, "thread:manage").Scan(&perm)
		if perm == 0 {
			resp.Forbidden(c, "只能编辑自己的帖子")
			return
		}
	}
	h.DB.Model(&th).Updates(map[string]interface{}{"title": req.Title, "content": req.Content})
	resp.OK(c, nil)
}

// 删除自己的帖子
func (h *ThreadHandler) DeleteThread(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	uid := middleware.GetUID(c)
	var th model.Thread
	if err := h.DB.First(&th, id).Error; err != nil {
		resp.NotFound(c, "帖子不存在")
		return
	}
	if th.UserID != uid {
		resp.Forbidden(c, "只能删除自己的帖子")
		return
	}
	h.DB.Model(&th).Update("status", 0)
	resp.OK(c, nil)
}

// 删除自己的回复
func (h *ThreadHandler) DeleteReply(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	uid := middleware.GetUID(c)
	var reply model.Reply
	if err := h.DB.First(&reply, id).Error; err != nil {
		resp.NotFound(c, "回复不存在")
		return
	}
	if reply.UserID != uid {
		resp.Forbidden(c, "只能删除自己的回复")
		return
	}
	h.DB.Model(&reply).Update("status", 0)
	resp.OK(c, nil)
}
