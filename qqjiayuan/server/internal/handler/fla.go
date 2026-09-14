package handler

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// FlaHandler 福利院·捐款慈善基金（每日捐款上榜，榜首受膜拜）
type FlaHandler struct{ DB *gorm.DB }

func (h *FlaHandler) todayTop() model.FlaDonation {
	var top model.FlaDonation
	h.DB.Preload("User").Where("DATE(created_at) = ?", time.Now().Format("2006-01-02")).
		Order("amount DESC, id ASC").First(&top)
	return top
}

// Index 今日上榜列表（公开）+ 我的今日捐款（登录才有）
func (h *FlaHandler) Index(c *gin.Context) {
	uid := middleware.GetUID(c)
	today := time.Now().Format("2006-01-02")
	var list []model.FlaDonation
	h.DB.Preload("User").Where("DATE(created_at) = ?", today).
		Order("amount DESC, id ASC").Limit(20).Find(&list)
	out := make([]gin.H, 0, len(list))
	for _, d := range list {
		nickname, color := "神秘友友", ""
		if d.User != nil {
			nickname = d.User.Nickname
			color = d.User.Color
		}
		out = append(out, gin.H{
			"id": d.ID, "user_id": d.UserID, "nickname": nickname, "color": color,
			"amount": d.Amount, "word": d.Word, "worships": d.Worships, "created_at": d.CreatedAt,
		})
	}
	var mineOut gin.H
	if uid > 0 {
		var mine model.FlaDonation
		h.DB.Where("user_id = ? AND DATE(created_at) = ?", uid, today).Order("id DESC").First(&mine)
		if mine.ID > 0 {
			mineOut = gin.H{"amount": mine.Amount, "word": mine.Word}
		}
	}
	resp.OK(c, gin.H{"list": out, "mine": mineOut})
}

// Donate 捐款上榜（每人每日一次，价高者上位）
func (h *FlaHandler) Donate(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Amount int    `json:"amount" binding:"required,min=1"`
		Word   string `json:"word" binding:"max=30"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "捐款金额需大于 0，宣言最多 30 字")
		return
	}
	req.Word = strings.TrimSpace(req.Word)
	today := time.Now().Format("2006-01-02")
	var cnt int64
	h.DB.Model(&model.FlaDonation{}).Where("user_id = ? AND DATE(created_at) = ?", uid, today).Count(&cnt)
	if cnt > 0 {
		resp.ParamError(c, "每人每日仅可捐献一次，明天再来吧")
		return
	}
	var u model.User
	if err := h.DB.First(&u, uid).Error; err != nil {
		resp.Unauthorized(c, "请先登录")
		return
	}
	if u.Coins < req.Amount {
		resp.ParamError(c, "G币不足")
		return
	}
	h.DB.Model(&u).Update("coins", gorm.Expr("coins - ?", req.Amount))
	h.DB.Create(&model.FlaDonation{UserID: uid, Amount: req.Amount, Word: req.Word})
	addWalletLog(h.DB, uid, "fla", "福利院捐款上榜", "coins", -req.Amount)
	var total int64
	h.DB.Model(&model.FlaDonation{}).Where("DATE(created_at) = ?", today).Count(&total)
	resp.OK(c, gin.H{"coins": u.Coins - req.Amount, "rank": total})
}

// Worship 膜拜今日榜首（每人每日一次）
func (h *FlaHandler) Worship(c *gin.Context) {
	uid := middleware.GetUID(c)
	top := h.todayTop()
	if top.ID == 0 {
		resp.ParamError(c, "今日还没有捐款榜首")
		return
	}
	if top.UserID == uid {
		resp.ParamError(c, "不能膜拜自己哦")
		return
	}
	today := time.Now().Format("2006-01-02")
	var cnt int64
	h.DB.Model(&model.FlaWorship{}).Where("user_id = ? AND DATE(created_at) = ?", uid, today).Count(&cnt)
	if cnt > 0 {
		resp.ParamError(c, "今天已经膜拜过啦，明天再来~")
		return
	}
	h.DB.Create(&model.FlaWorship{DonID: top.ID, UserID: uid})
	h.DB.Model(&model.FlaDonation{}).Where("id = ?", top.ID).UpdateColumn("worships", gorm.Expr("worships + 1"))
	var me model.User
	h.DB.First(&me, uid)
	h.DB.Create(&model.Notification{
		UserID: top.UserID, Type: "system", Title: "福利院膜拜",
		Content: fmt.Sprintf("友友 %s（%s）膜拜了今日慈善榜首的你！记得保持低调~", me.Nickname, me.Username),
	})
	resp.OK(c, gin.H{"worships": top.Worships + 1})
}
