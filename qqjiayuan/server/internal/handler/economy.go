package handler

import (
	"crypto/md5"
	"math/rand"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

type EconomyHandler struct {
	DB *gorm.DB
}

// 社区银行：年化日息 0.5%（向上取整，最少 1 金币），每日可领一次
const bankDailyRate = 0.005

func (h *EconomyHandler) BankView(c *gin.Context) {
	uid := middleware.GetUID(c)
	acc := h.getAccount(uid)
	var u model.User
	h.DB.First(&u, uid)

	balance, lastInterest := 0, (*time.Time)(nil)
	if acc.ID > 0 {
		balance = acc.Balance
		lastInterest = acc.LastInterestAt
	}
	// 今日是否已领利息
	interestToday := false
	if lastInterest != nil && lastInterest.Format("2006-01-02") == time.Now().Format("2006-01-02") {
		interestToday = true
	}
	// 今日应得利息
	rate := int(float64(balance) * bankDailyRate)
	if rate < 1 {
		rate = 1
	}
	if balance <= 0 {
		rate = 0
	}
	resp.OK(c, gin.H{
		"balance":       balance,
		"coins":         u.Coins,
		"rate":          rate,
		"interest_today": interestToday,
		"last_interest":  lastInterest,
	})
}

func (h *EconomyHandler) BankDeposit(c *gin.Context) {
	uid := middleware.GetUID(c)
	amount, ok := amountOf(c)
	if !ok {
		return
	}
	if amount <= 0 {
		resp.ParamError(c, "存入金额需大于 0")
		return
	}
	var u model.User
	if err := h.DB.First(&u, uid).Error; err != nil {
		resp.ParamError(c, "用户不存在")
		return
	}
	if u.Coins < amount {
		resp.ParamError(c, "金币不足")
		return
	}
	var ack model.BankAccount
	h.DB.Where("user_id = ?", uid).FirstOrCreate(&ack, model.BankAccount{UserID: uid})
	h.DB.Model(&u).Update("coins", gorm.Expr("coins - ?", amount))
	h.DB.Model(&ack).Update("balance", gorm.Expr("balance + ?", amount))
	resp.OK(c, h.bankAfter(uid))
}

func (h *EconomyHandler) BankWithdraw(c *gin.Context) {
	uid := middleware.GetUID(c)
	amount, ok := amountOf(c)
	if !ok {
		return
	}
	if amount <= 0 {
		resp.ParamError(c, "取出金额需大于 0")
		return
	}
	acc := h.getAccount(uid)
	if acc.ID == 0 || acc.Balance < amount {
		resp.ParamError(c, "存款不足")
		return
	}
	var u model.User
	h.DB.First(&u, uid)
	h.DB.Model(&u).Update("coins", gorm.Expr("coins + ?", amount))
	h.DB.Model(&acc).Update("balance", gorm.Expr("balance - ?", amount))
	resp.OK(c, h.bankAfter(uid))
}

// 领取每日利息
func (h *EconomyHandler) BankInterest(c *gin.Context) {
	uid := middleware.GetUID(c)
	acc := h.getAccount(uid)
	if acc.ID == 0 || acc.Balance <= 0 {
		resp.ParamError(c, "账户里还没有存款")
		return
	}
	today := time.Now().Format("2006-01-02")
	if acc.LastInterestAt != nil && acc.LastInterestAt.Format("2006-01-02") == today {
		resp.ParamError(c, "今天已经领过利息啦，明天再来吧")
		return
	}
	rate := int(float64(acc.Balance) * bankDailyRate)
	if rate < 1 {
		rate = 1
	}
	now := time.Now()
	h.DB.Model(&acc).Updates(map[string]interface{}{
		"balance":         gorm.Expr("balance + ?", rate),
		"last_interest_at": now,
	})
	var u model.User
	h.DB.First(&u, uid)
	h.DB.Model(&u).Update("coins", gorm.Expr("coins + ?", rate))
	resp.OK(c, gin.H{"rate": rate, "balance": acc.Balance + rate, "coins": u.Coins + rate})
}

// 挖宝：花金币挖，随机得金币
const digCost = 30

func (h *EconomyHandler) Dig(c *gin.Context) {
	uid := middleware.GetUID(c)
	var u model.User
	h.DB.First(&u, uid)
	if u.Coins < digCost {
		resp.ParamError(c, "金币不足，挖宝需要 "+strconv.Itoa(digCost)+" 金币")
		return
	}
	r := rand.Intn(100)
	reward := 0
	tip := ""
	switch {
	case r < 35:
		tip = "挖到了泥土，什么也没有"
	case r < 80:
		reward = 10 + rand.Intn(31)
		tip = "挖到了宝物！"
	default:
		reward = 100
		tip = "大丰收！挖到稀有宝藏"
	}
	h.DB.Model(&u).Update("coins", gorm.Expr("coins + ? - ?", reward, digCost))
	h.DB.First(&u, uid)
	resp.OK(c, gin.H{"tip": tip, "reward": reward, "coins": u.Coins})
}

type charityReq struct {
	Amount int `json:"amount" binding:"required"`
}

// 慈善基金：捐款
func (h *EconomyHandler) Charity(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req charityReq
	if err := c.ShouldBindJSON(&req); err != nil || req.Amount < 1 {
		resp.ParamError(c, "捐款金额需大于 0")
		return
	}
	var u model.User
	h.DB.First(&u, uid)
	if u.Coins < req.Amount {
		resp.ParamError(c, "金币不足")
		return
	}
	h.DB.Model(&u).Update("coins", gorm.Expr("coins - ?", req.Amount))
	h.DB.Create(&model.Donation{UserID: uid, Amount: req.Amount})
	resp.OK(c, gin.H{"coins": u.Coins - req.Amount})
}

// 慈善基金排行：按累计捐款排序
func (h *EconomyHandler) CharityRank(c *gin.Context) {
	type row struct {
		UserID     uint   `json:"user_id"`
		Nickname   string `json:"nickname"`
		Color      string `json:"color"`
		TotalDonat int64  `json:"total"`
	}
	var rows []row
	h.DB.Raw(`SELECT d.user_id, u.nickname, u.color, SUM(d.amount) AS total
FROM donations d JOIN users u ON u.id = d.user_id
GROUP BY d.user_id, u.nickname, u.color ORDER BY total DESC LIMIT 10`).Scan(&rows)
	resp.OK(c, rows)
}

// 打工：每次随机奖励金币，每日最多 3 次
const workDailyLimit = 3

func (h *EconomyHandler) WorkStatus(c *gin.Context) {
	uid := middleware.GetUID(c)
	done := h.workDoneToday(uid)
	resp.OK(c, gin.H{"done": done, "limit": workDailyLimit, "left": workDailyLimit - done})
}

func (h *EconomyHandler) WorkDo(c *gin.Context) {
	uid := middleware.GetUID(c)
	done := h.workDoneToday(uid)
	if done >= workDailyLimit {
		resp.ParamError(c, "今天已经打满工了，明天再来吧")
		return
	}
	reward := 5 + rand.Intn(11) // 5~15 金币
	if err := h.DB.Create(&model.WorkRecord{UserID: uid}).Error; err != nil {
		resp.ServerError(c, err)
		return
	}
	h.DB.Model(&model.User{}).Where("id = ?", uid).Update("coins", gorm.Expr("coins + ?", reward))
	var u model.User
	h.DB.First(&u, uid)
	resp.OK(c, gin.H{"reward": reward, "coins": u.Coins, "done": done + 1, "limit": workDailyLimit})
}

// 每日星运：按 用户+日期 锚定，同一天内结果稳定
func (h *EconomyHandler) Fortune(c *gin.Context) {
	uid := middleware.GetUID(c)
	date := time.Now().Format("2006-01-02")
	seed := md5.Sum([]byte(string(rune(uid)) + date))
	n := int(seed[0])
	stars := []string{"★", "★★", "★★★", "★★★★", "★★★★★"}
	luck := []string{"大吉", "中吉", "小吉", "平", "凶", "大凶"}
	opts := []string{"宜交友", "宜灌水", "宜签到", "宜发帖", "宜养崽", "宜挖宝", "宜低调", "宜出游", "宜表白"}
	words := []string{
		"运势上扬，好友互动带来意外之喜。",
		"今天适合盖楼灌水，人气值爆棚。",
		"低调攒人品，静待花开。",
		"财运不错，邻里互赠或有好消息。",
		"收好小脾气，温和待人会换来真心。",
		"适合整理心情，写一篇美文。",
		"别宅着啦，去同城看看热闹吧。",
		"疲惫时去看看花园，一切都会好起来。",
		"遇见旧友，记得好好打个招呼。",
	}
	idx := n % 9
	luckIdx := seed[1] % uint8(len(luck))
	resp.OK(c, gin.H{
		"star":  stars[seed[2]%uint8(len(stars))],
		"luck":  luck[luckIdx],
		"opts":  opts[idx],
		"word":  words[idx],
		"lucky": seed[3]%9 + 1,
		"date":  date,
	})
}

// 幸运猜数字：下注猜大/小/单/双，结果随机 1~9，加倍返还
func (h *EconomyHandler) Lottery(c *gin.Context) {
	uid := middleware.GetUID(c)
	bet := c.PostForm("bet")
	amount, ok := amountOf(c)
	if !ok {
		return
	}
	bets := map[string]bool{"big": true, "small": true, "odd": true, "even": true}
	if !bets[bet] {
		resp.ParamError(c, "请选择 大/小/单/双")
		return
	}
	if amount <= 0 {
		resp.ParamError(c, "下注金额需大于 0")
		return
	}
	var u model.User
	if err := h.DB.First(&u, uid).Error; err != nil {
		resp.ParamError(c, "用户不存在")
		return
	}
	if u.Coins < amount {
		resp.ParamError(c, "金币不足")
		return
	}
	num := 1 + rand.Intn(9) // 1~9
	win := false
	switch bet {
	case "big":
		win = num >= 5
	case "small":
		win = num <= 4
	case "odd":
		win = num%2 == 1
	case "even":
		win = num%2 == 0
	}
	if win {
		h.DB.Model(&u).Update("coins", gorm.Expr("coins + ?", amount))
	} else {
		h.DB.Model(&u).Update("coins", gorm.Expr("coins - ?", amount))
	}
	h.DB.First(&u, uid)
	resp.OK(c, gin.H{"num": num, "win": win, "coins": u.Coins})
}

// ---- 内部工具 ----

func (h *EconomyHandler) getAccount(uid uint) model.BankAccount {
	var acc model.BankAccount
	h.DB.Where("user_id = ?", uid).First(&acc)
	return acc
}

func (h *EconomyHandler) workDoneToday(uid uint) int {
	var count int64
	h.DB.Model(&model.WorkRecord{}).
		Where("user_id = ? AND created_at >= ?", uid, time.Now().Format("2006-01-02")+" 00:00:00").
		Count(&count)
	return int(count)
}

func (h *EconomyHandler) bankAfter(uid uint) gin.H {
	acc := h.getAccount(uid)
	var u model.User
	h.DB.First(&u, uid)
	balance := 0
	if acc.ID > 0 {
		balance = acc.Balance
	}
	return gin.H{"balance": balance, "coins": u.Coins}
}

// 解析并校验 amount 表单参数
func amountOf(c *gin.Context) (int, bool) {
	s := c.PostForm("amount")
	amount, err := strconv.Atoi(s)
	if err != nil || amount <= 0 {
		resp.ParamError(c, "金额不合法")
		return 0, false
	}
	return amount, true
}
