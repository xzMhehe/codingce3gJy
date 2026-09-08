package handler

import (
	"math/rand"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// YouQuanHandler 友友券中心（复刻诺哈财务中心 + 家园友友券玩法）
type YouQuanHandler struct{ DB *gorm.DB }

// 友友券 → G币 兑换汇率：1 友友券 = 100 G币
const youquanExchangeRate = 100

// View 友友券中心首页：余额 + 今日领取状态 + 收支记录 + 最近连续签到
func (h *YouQuanHandler) View(c *gin.Context) {
	uid := middleware.GetUID(c)
	var u model.User
	if err := h.DB.First(&u, uid).Error; err != nil {
		resp.Unauthorized(c, "请先登录")
		return
	}
	// 今日是否已免费领取
	claimedToday := false
	var cnt int64
	h.DB.Model(&model.WalletLog{}).
		Where("user_id = ? AND kind = 'youquan_daily' AND created_at >= ?", uid, todayStart()).
		Count(&cnt)
	if cnt > 0 {
		claimedToday = true
	}
	// 连续签到天数
	consec := 0
	var last model.SignIn
	if err := h.DB.Where("user_id = ?", uid).Order("sign_date DESC").First(&last).Error; err == nil {
		if last.SignDate == todayStr() || last.SignDate == time.Now().AddDate(0, 0, -1).Format("2006-01-02") {
			consec = last.Consec
		}
	}
	// 友友券收支记录（最近 30 条）
	var logs []model.WalletLog
	h.DB.Where("user_id = ? AND currency = 'youquan'", uid).Order("id DESC").Limit(30).Find(&logs)
	out := []gin.H{}
	for _, l := range logs {
		out = append(out, gin.H{"id": l.ID, "kind": l.Kind, "title": l.Title, "delta": l.Delta, "created_at": l.CreatedAt})
	}
	// 友友券可购商品数量
	var youquanGoods int64
	h.DB.Model(&model.Good{}).Where("status = 1 AND youquan_price > 0").Count(&youquanGoods)
	resp.OK(c, gin.H{
		"youquan": u.YouQuan, "claimed_today": claimedToday, "consec": consec,
		"exchange_rate": youquanExchangeRate, "youquan_goods": youquanGoods,
		"logs": out,
	})
}

// Daily 每日免费领取 1~3 张友友券（每天一次）
func (h *YouQuanHandler) Daily(c *gin.Context) {
	uid := middleware.GetUID(c)
	var cnt int64
	h.DB.Model(&model.WalletLog{}).
		Where("user_id = ? AND kind = 'youquan_daily' AND created_at >= ?", uid, todayStart()).
		Count(&cnt)
	if cnt > 0 {
		resp.ParamError(c, "今天已经领取过友友券啦，明天再来吧")
		return
	}
	n := 1 + rand.Intn(3) // 1~3 张
	h.DB.Model(&model.User{}).Where("id = ?", uid).Update("youquan", gorm.Expr("youquan + ?", n))
	addWalletLog(h.DB, uid, "youquan_daily", "每日免费领取友友券", "youquan", n)
	var u model.User
	h.DB.First(&u, uid)
	resp.OK(c, gin.H{"gain": n, "youquan": u.YouQuan})
}

// Exchange 友友券 → G币 兑换
func (h *YouQuanHandler) Exchange(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Num int `json:"num" binding:"required,min=1,max=100000"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "兑换数量需在 1~100000 之间")
		return
	}
	var u model.User
	if err := h.DB.First(&u, uid).Error; err != nil {
		resp.Unauthorized(c, "请先登录")
		return
	}
	if u.YouQuan < req.Num {
		resp.ParamError(c, "友友券不足，需要 "+strconv.Itoa(req.Num)+" 张")
		return
	}
	gain := req.Num * youquanExchangeRate
	h.DB.Model(&u).Updates(map[string]interface{}{
		"youquan": gorm.Expr("youquan - ?", req.Num),
		"coins":   gorm.Expr("coins + ?", gain),
	})
	addWalletLog(h.DB, uid, "youquan_exchange", "友友券兑换G币", "youquan", -req.Num)
	addWalletLog(h.DB, uid, "youquan_exchange", "友友券兑换G币", "coins", gain)
	resp.OK(c, gin.H{"gain": gain, "youquan": u.YouQuan - req.Num, "coins": u.Coins + gain})
}

// Transfer 友友券转账给好友（1% 手续费，取整）
func (h *YouQuanHandler) Transfer(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		To     string `json:"to" binding:"required"`
		Amount int    `json:"amount" binding:"required,min=1,max=1000000"`
		Remark string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请输入对方号码和转账数量（1~1000000）")
		return
	}
	var me model.User
	if err := h.DB.First(&me, uid).Error; err != nil {
		resp.Unauthorized(c, "请先登录")
		return
	}
	var to model.User
	if err := h.DB.Where("username = ?", req.To).First(&to).Error; err != nil {
		resp.ParamError(c, "对方号码不存在")
		return
	}
	if to.ID == uid {
		resp.ParamError(c, "不能转给自己哦")
		return
	}
	fee := req.Amount / 100
	if fee < 1 {
		fee = 0
	}
	total := req.Amount + fee
	if me.YouQuan < total {
		resp.ParamError(c, "友友券不足（含1%手续费），需要 "+strconv.Itoa(total)+" 张")
		return
	}
	h.DB.Model(&me).Update("youquan", gorm.Expr("youquan - ?", total))
	h.DB.Model(&to).Update("youquan", gorm.Expr("youquan + ?", req.Amount))
	remark := req.Remark
	if remark == "" {
		remark = "-"
	}
	addWalletLog(h.DB, uid, "youquan_transfer", "转账友友券给"+to.Nickname+"("+to.Username+")", "youquan", -total)
	addWalletLog(h.DB, to.ID, "youquan_transfer", "收到"+me.Nickname+"("+me.Username+")友友券", "youquan", req.Amount)
	var u model.User
	h.DB.First(&u, uid)
	resp.OK(c, gin.H{"youquan": u.YouQuan, "fee": fee})
}

// todayStart 今日零点（用于判断今日是否已领取）
func todayStart() string {
	return time.Now().Format("2006-01-02") + " 00:00:00"
}
