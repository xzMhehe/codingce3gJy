package handler

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// WelfareHandler 福利院·慈善基金（池内 G币供友友领取，友友捐献 补池+得财富值）
type WelfareHandler struct{ DB *gorm.DB }

const welfareDefaultPool = 500845400

// fund 保证存在单行基金池并返回
func (h *WelfareHandler) fund() model.WelfareFund {
	var f model.WelfareFund
	if err := h.DB.First(&f, 1).Error; err != nil || f.ID == 0 {
		f = model.WelfareFund{ID: 1, Pool: welfareDefaultPool}
		h.DB.Create(&f)
		h.DB.First(&f, 1)
	}
	return f
}

func outUser(u *model.User) (nickname, username, color string) {
	nickname, username, color = "神秘友友", "", ""
	if u != nil {
		nickname = u.Nickname
		username = u.Username
		color = u.Color
	}
	return
}

// Index 福利院主页（公开，登录可见个人状态）
func (h *WelfareHandler) Index(c *gin.Context) {
	uid := middleware.GetUID(c)
	today := time.Now().Format("2006-01-02")
	f := h.fund()

	// 今日统计
	var claimSum, donSum int64
	h.DB.Model(&model.WelfareClaim{}).Where("DATE(created_at) = ?", today).
		Select("COALESCE(SUM(amount),0)").Scan(&claimSum)
	h.DB.Model(&model.WelfareDonate{}).Where("DATE(created_at) = ?", today).
		Select("COALESCE(SUM(amount),0)").Scan(&donSum)

	// 领取动态（最近5条）
	var claims []model.WelfareClaim
	h.DB.Preload("User").Where("DATE(created_at) = ?", today).
		Order("id DESC").Limit(5).Find(&claims)
	claimOut := make([]gin.H, 0, len(claims))
	for _, cl := range claims {
		nickname, username, color := outUser(cl.User)
		claimOut = append(claimOut, gin.H{
			"id": cl.ID, "user_id": cl.UserID, "nickname": nickname, "username": username, "color": color,
			"amount": cl.Amount, "created_at": cl.CreatedAt,
		})
	}

	// 捐献动态（最近5条）
	var dons []model.WelfareDonate
	h.DB.Preload("User").Where("DATE(created_at) = ?", today).
		Order("id DESC").Limit(5).Find(&dons)
	donOut := make([]gin.H, 0, len(dons))
	for _, d := range dons {
		nickname, username, color := outUser(d.User)
		donOut = append(donOut, gin.H{
			"id": d.ID, "user_id": d.UserID, "nickname": nickname, "username": username, "color": color,
			"amount": d.Amount, "points": d.Amount / 100, "created_at": d.CreatedAt,
		})
	}

	// 竞价排行：今日捐款榜首（联动捐款上榜 fla 模块）
	var flTop model.FlaDonation
	h.DB.Preload("User").Where("DATE(created_at) = ?", today).
		Order("amount DESC, id ASC").First(&flTop)
	var flTopOut gin.H
	flWorshipped := false
	if flTop.ID > 0 {
		nickname, username, color := outUser(flTop.User)
		avatar, avatarB64 := "", ""
		if flTop.User != nil {
			avatar = flTop.User.Avatar
			avatarB64 = flTop.User.AvatarBase64
		}
		flTopOut = gin.H{
			"id": flTop.ID, "user_id": flTop.UserID, "nickname": nickname, "username": username, "color": color,
			"avatar": avatar, "avatar_base64": avatarB64,
			"amount": flTop.Amount, "word": flTop.Word, "worships": flTop.Worships,
		}
	}
	if uid > 0 {
		var wcnt int64
		h.DB.Model(&model.FlaWorship{}).Where("user_id = ? AND DATE(created_at) = ?", uid, today).Count(&wcnt)
		flWorshipped = wcnt > 0
	}

	// 财富排行（累计捐献折算财富值 TOP3）
	type rankRow struct {
		UserID uint
		Total  int64
	}
	var ranks []rankRow
	h.DB.Model(&model.WelfareDonate{}).
		Select("user_id, COALESCE(SUM(amount),0) AS total").
		Group("user_id").Order("total DESC").Limit(3).Scan(&ranks)
	rankOut := make([]gin.H, 0, len(ranks))
	for _, r := range ranks {
		var u model.User
		h.DB.First(&u, r.UserID)
		nickname, username, color := outUser(&u)
		rankOut = append(rankOut, gin.H{
			"user_id": r.UserID, "nickname": nickname, "username": username, "color": color,
			"points": r.Total / 100,
		})
	}

	// 个人状态（任务为累计发帖/回帖数量，领取可多次）
	var mine gin.H
	if uid > 0 {
		var u model.User
		if h.DB.First(&u, uid).Error == nil {
			var tCnt, rCnt int64
			h.DB.Model(&model.Thread{}).Where("user_id = ? AND status = 1", uid).Count(&tCnt)
			h.DB.Model(&model.Reply{}).Where("user_id = ? AND status = 1", uid).Count(&rCnt)
			var cCnt, dCnt int64
			h.DB.Model(&model.WelfareClaim{}).Where("user_id = ? AND DATE(created_at) = ?", uid, today).Count(&cCnt)
			h.DB.Model(&model.WelfareDonate{}).Where("user_id = ? AND DATE(created_at) = ?", uid, today).Count(&dCnt)
			var claimSumToday int64
			h.DB.Model(&model.WelfareClaim{}).Where("user_id = ? AND DATE(created_at) = ?", uid, today).
				Select("COALESCE(SUM(amount),0)").Scan(&claimSumToday)
			claimAmt, donateAmt := 0, 0
			if cCnt > 0 {
				var cl model.WelfareClaim
				h.DB.Where("user_id = ? AND DATE(created_at) = ?", uid, today).Order("id DESC").First(&cl)
				claimAmt = cl.Amount
			}
			if dCnt > 0 {
				var d model.WelfareDonate
				h.DB.Where("user_id = ? AND DATE(created_at) = ?", uid, today).Order("id DESC").First(&d)
				donateAmt = d.Amount
			}
			mine = gin.H{
				"coins":           u.Coins,
				"task_thread":     int(tCnt),
				"task_reply":      int(rCnt),
				"task_done":       tCnt >= 1 && rCnt >= 5,
				"claimed_today":   cCnt > 0,
				"claims_today":    int(cCnt),
				"claim_amount":    claimAmt,
				"claim_today_sum": claimSumToday,
				"donated_today":   dCnt > 0,
				"donate_amount":   donateAmt,
				"donate_points":   donateAmt / 100,
				"achieve":         u.Achieve,
			}
		}
	}

	resp.OK(c, gin.H{
		"pool":          f.Pool,
		"today_out":     claimSum,
		"today_in":      donSum,
		"claims":        claimOut,
		"donations":     donOut,
		"rank":          rankOut,
		"mine":          mine,
		"fl_top":        flTopOut,
		"fl_worshipped": flWorshipped,
	})
}

// Claim 领取慈善福利（每日一次；任务为累计发帖 1 / 回帖 5；池子见底需等捐献）
func (h *WelfareHandler) Claim(c *gin.Context) {
	uid := middleware.GetUID(c)
	if uid == 0 {
		resp.Unauthorized(c, "请先登录")
		return
	}
	today := time.Now().Format("2006-01-02")
	var cnt int64
	h.DB.Model(&model.WelfareClaim{}).Where("user_id = ? AND DATE(created_at) = ?", uid, today).Count(&cnt)
	if cnt > 0 {
		resp.ParamError(c, "每天只能领取一次，明天再来吧")
		return
	}
	var tCnt, rCnt int64
	h.DB.Model(&model.Thread{}).Where("user_id = ? AND status = 1", uid).Count(&tCnt)
	h.DB.Model(&model.Reply{}).Where("user_id = ? AND status = 1", uid).Count(&rCnt)
	if tCnt < 1 || rCnt < 5 {
		msg := ""
		if tCnt < 1 {
			msg = "领取失败：需先完成以下任务——发布一篇帖子（未完成）"
		} else {
			msg = fmt.Sprintf("领取失败：需先完成以下任务——完成五条回帖（%d/5）", rCnt)
		}
		resp.ParamError(c, msg)
		return
	}

	// 领取金额：与池子挂钩，约万分之五 ~ 百分之零点九
	f := h.fund()
	if f.Pool <= 0 {
		resp.ParamError(c, "福利池已见底，等待友友捐献吧")
		return
	}
	minAmt := 200
	maxAmt := f.Pool / 1000
	if maxAmt < minAmt {
		maxAmt = minAmt
	}
	if maxAmt > 500000 {
		maxAmt = 500000
	}
	amount := minAmt + rand.Intn(maxAmt-minAmt+1)

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		// 扣池（不允许为负）
		res := tx.Model(&model.WelfareFund{}).Where("id = 1 AND pool >= ?", amount).
			Update("pool", gorm.Expr("pool - ?", amount))
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return fmt.Errorf("福利池已见底，等待友友捐献吧")
		}
		if err := tx.Model(&model.User{}).Where("id = ?", uid).
			Update("coins", gorm.Expr("coins + ?", amount)).Error; err != nil {
			return err
		}
		return tx.Create(&model.WelfareClaim{UserID: uid, Amount: amount}).Error
	})
	if err != nil {
		resp.ParamError(c, err.Error())
		return
	}
	addWalletLog(h.DB, uid, "welfare", "领取慈善福利", "coins", amount)
	var u model.User
	h.DB.First(&u, uid)
	resp.OK(c, gin.H{"amount": amount, "coins": u.Coins})
}

// Donate 捐献慈善基金（每日一次，捐献金额全部入池，并按 1% 换算财富值）
func (h *WelfareHandler) Donate(c *gin.Context) {
	uid := middleware.GetUID(c)
	if uid == 0 {
		resp.Unauthorized(c, "请先登录")
		return
	}
	var req struct {
		Amount int `json:"amount" binding:"required,min=100"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Amount < 100 {
		resp.ParamError(c, "捐献金额最低 100 G币")
		return
	}
	today := time.Now().Format("2006-01-02")
	var cnt int64
	h.DB.Model(&model.WelfareDonate{}).Where("user_id = ? AND DATE(created_at) = ?", uid, today).Count(&cnt)
	if cnt > 0 {
		resp.ParamError(c, "每日仅可捐献一次，明天再来吧")
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
	points := req.Amount / 100
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&model.User{}).Where("id = ? AND coins >= ?", uid, req.Amount).
			Update("coins", gorm.Expr("coins - ?", req.Amount))
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return fmt.Errorf("G币不足")
		}
		if err := tx.Model(&model.WelfareFund{}).Where("id = 1").
			Update("pool", gorm.Expr("pool + ?", req.Amount)).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.User{}).Where("id = ?", uid).
			Update("achieve", gorm.Expr("achieve + ?", points)).Error; err != nil {
			return err
		}
		return tx.Create(&model.WelfareDonate{UserID: uid, Amount: req.Amount}).Error
	})
	if err != nil {
		resp.ParamError(c, err.Error())
		return
	}
	addWalletLog(h.DB, uid, "welfare", "捐献慈善基金", "coins", -req.Amount)
	resp.OK(c, gin.H{"points": points})
}

// Claims 领取名单（公开，分页；不传 date 只显示今日）
func (h *WelfareHandler) Claims(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	date := c.DefaultQuery("date", time.Now().Format("2006-01-02"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 50 {
		size = 10
	}
	q := h.DB.Model(&model.WelfareClaim{})
	if date != "all" {
		q = q.Where("DATE(created_at) = ?", date)
	}
	var total, sum int64
	q.Count(&total)
	dq := h.DB.Model(&model.WelfareClaim{})
	if date != "all" {
		dq = dq.Where("DATE(created_at) = ?", date)
	}
	dq.Select("COALESCE(SUM(amount),0)").Scan(&sum)
	var list []model.WelfareClaim
	lq := h.DB.Preload("User").Order("id DESC")
	if date != "all" {
		lq = lq.Where("DATE(created_at) = ?", date)
	}
	lq.Offset((page - 1) * size).Limit(size).Find(&list)
	out := make([]gin.H, 0, len(list))
	for _, cl := range list {
		nickname, _, color := outUser(cl.User)
		out = append(out, gin.H{
			"id": cl.ID, "user_id": cl.UserID, "nickname": nickname, "color": color,
			"amount": cl.Amount, "created_at": cl.CreatedAt,
		})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "sum": sum, "page": page, "size": size})
}

// Donations 捐赠名单（公开，分页）
func (h *WelfareHandler) Donations(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	date := c.DefaultQuery("date", time.Now().Format("2006-01-02"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 50 {
		size = 10
	}
	dq := h.DB.Model(&model.WelfareDonate{})
	if date != "all" {
		dq = dq.Where("DATE(created_at) = ?", date)
	}
	var total, sum int64
	dq.Count(&total)
	dq.Select("COALESCE(SUM(amount),0)").Scan(&sum)
	var list []model.WelfareDonate
	lq := h.DB.Preload("User").Order("id DESC")
	if date != "all" {
		lq = lq.Where("DATE(created_at) = ?", date)
	}
	lq.Offset((page - 1) * size).Limit(size).Find(&list)
	out := make([]gin.H, 0, len(list))
	for _, d := range list {
		nickname, _, color := outUser(d.User)
		out = append(out, gin.H{
			"id": d.ID, "user_id": d.UserID, "nickname": nickname, "color": color,
			"amount": d.Amount, "points": d.Amount / 100, "created_at": d.CreatedAt,
		})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "sum": sum, "page": page, "size": size})
}

// ============ 管理端 ============

// AdminStats 福利院统计 + 基金池
func (h *WelfareHandler) AdminStats(c *gin.Context) {
	today := time.Now().Format("2006-01-02")
	var totalOut, totalIn, claimCnt, donCnt, todayOut, todayIn int64
	h.DB.Model(&model.WelfareClaim{}).Select("COALESCE(SUM(amount),0)").Scan(&totalOut)
	h.DB.Model(&model.WelfareDonate{}).Select("COALESCE(SUM(amount),0)").Scan(&totalIn)
	h.DB.Model(&model.WelfareClaim{}).Count(&claimCnt)
	h.DB.Model(&model.WelfareDonate{}).Count(&donCnt)
	h.DB.Model(&model.WelfareClaim{}).Where("DATE(created_at) = ?", today).Select("COALESCE(SUM(amount),0)").Scan(&todayOut)
	h.DB.Model(&model.WelfareDonate{}).Where("DATE(created_at) = ?", today).Select("COALESCE(SUM(amount),0)").Scan(&todayIn)
	f := h.fund()
	resp.OK(c, gin.H{
		"pool": f.Pool, "total_out": totalOut, "total_in": totalIn,
		"claim_count": claimCnt, "donate_count": donCnt,
		"today_out": todayOut, "today_in": todayIn,
	})
}

// AdminSetPool 管理端调整福利池（正数充值 / 负数扣减）
func (h *WelfareHandler) AdminSetPool(c *gin.Context) {
	var req struct {
		Delta int  `json:"delta"`
		Set   *int `json:"set"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	h.fund()
	var f model.WelfareFund
	h.DB.First(&f, 1)
	if req.Set != nil {
		if *req.Set < 0 {
			resp.ParamError(c, "池金额不能为负")
			return
		}
		h.DB.Model(&f).Update("pool", *req.Set)
	} else {
		if req.Delta == 0 {
			resp.ParamError(c, "调整金额不能为 0")
			return
		}
		if f.Pool+req.Delta < 0 {
			resp.ParamError(c, "扣减后不能为负")
			return
		}
		h.DB.Model(&f).Update("pool", gorm.Expr("pool + ?", req.Delta))
	}
	h.DB.First(&f, 1)
	resp.OK(c, gin.H{"pool": f.Pool})
}

func (h *WelfareHandler) adminFilter(c *gin.Context, db *gorm.DB) *gorm.DB {
	user := c.Query("user")
	date := c.Query("date")
	if user != "" {
		var ids []uint
		h.DB.Model(&model.User{}).Where("nickname LIKE ? OR username LIKE ?", "%"+user+"%", "%"+user+"%").Pluck("id", &ids)
		if len(ids) == 0 {
			return db.Where("1 = 0")
		}
		db = db.Where("user_id IN ?", ids)
	}
	if date != "" {
		db = db.Where("DATE(created_at) = ?", date)
	}
	return db
}

// AdminClaims 管理端·领取记录
func (h *WelfareHandler) AdminClaims(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 10
	}
	q := h.adminFilter(c, h.DB.Model(&model.WelfareClaim{}))
	var total, sum int64
	q.Count(&total)
	sq := h.adminFilter(c, h.DB.Model(&model.WelfareClaim{}))
	sq.Select("COALESCE(SUM(amount),0)").Scan(&sum)
	var list []model.WelfareClaim
	lq := h.adminFilter(c, h.DB.Preload("User"))
	lq.Order("created_at DESC, id DESC").Offset((page - 1) * size).Limit(size).Find(&list)
	out := make([]gin.H, 0, len(list))
	for _, cl := range list {
		nickname, username, color := outUser(cl.User)
		out = append(out, gin.H{
			"id": cl.ID, "user_id": cl.UserID, "nickname": nickname, "username": username, "color": color,
			"amount": cl.Amount, "created_at": cl.CreatedAt,
		})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "sum": sum, "page": page, "size": size})
}

// AdminClaims 管理端·捐献记录
func (h *WelfareHandler) AdminDonations(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 10
	}
	q := h.adminFilter(c, h.DB.Model(&model.WelfareDonate{}))
	var total, sum int64
	q.Count(&total)
	sq := h.adminFilter(c, h.DB.Model(&model.WelfareDonate{}))
	sq.Select("COALESCE(SUM(amount),0)").Scan(&sum)
	var list []model.WelfareDonate
	lq := h.adminFilter(c, h.DB.Preload("User"))
	lq.Order("created_at DESC, id DESC").Offset((page - 1) * size).Limit(size).Find(&list)
	out := make([]gin.H, 0, len(list))
	for _, d := range list {
		nickname, username, color := outUser(d.User)
		out = append(out, gin.H{
			"id": d.ID, "user_id": d.UserID, "nickname": nickname, "username": username, "color": color,
			"amount": d.Amount, "points": d.Amount / 100, "created_at": d.CreatedAt,
		})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "sum": sum, "page": page, "size": size})
}

// AdminCreateClaim 管理端手工补发一条领取记录（并直接加 G币给该用户）
func (h *WelfareHandler) AdminCreateClaim(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Nickname string `json:"nickname"`
		Amount   int    `json:"amount" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "金额需大于 0")
		return
	}
	var u model.User
	q := h.DB
	switch {
	case req.Username != "":
		q = q.Where("username = ?", req.Username)
	case req.Nickname != "":
		q = q.Where("nickname = ?", req.Nickname)
	default:
		resp.ParamError(c, "请填写家园号码或昵称")
		return
	}
	if err := q.First(&u).Error; err != nil {
		resp.ParamError(c, "用户不存在，请核对号码/昵称")
		return
	}
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.User{}).Where("id = ?", u.ID).
			Update("coins", gorm.Expr("coins + ?", req.Amount)).Error; err != nil {
			return err
		}
		return tx.Create(&model.WelfareClaim{UserID: u.ID, Amount: req.Amount}).Error
	})
	if err != nil {
		resp.ServerError(c, err)
		return
	}
	addWalletLog(h.DB, u.ID, "welfare", "管理员补发慈善福利", "coins", req.Amount)
	resp.OK(c, gin.H{"id": u.ID, "nickname": u.Nickname, "username": u.Username, "amount": req.Amount})
}

// AdminDeleteClaim 管理端删除领取记录（该用户 G币不足时余额扣至 0 为止）
func (h *WelfareHandler) AdminDeleteClaim(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var cl model.WelfareClaim
	if err := h.DB.First(&cl, id).Error; err != nil {
		resp.ParamError(c, "记录不存在")
		return
	}
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.User{}).Where("id = ? AND coins >= ?", cl.UserID, cl.Amount).
			Update("coins", gorm.Expr("coins - ?", cl.Amount)).Error; err != nil {
			return err
		}
		return tx.Delete(&cl).Error
	})
	if err != nil {
		resp.ServerError(c, err)
		return
	}
	addWalletLog(h.DB, cl.UserID, "welfare", "管理员作废领取记录", "coins", -cl.Amount)
	resp.OK(c, nil)
}

// AdminDeleteDonate 管理端删除捐献记录（从福利池扣回）
func (h *WelfareHandler) AdminDeleteDonate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var d model.WelfareDonate
	if err := h.DB.First(&d, id).Error; err != nil {
		resp.ParamError(c, "记录不存在")
		return
	}
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.WelfareFund{}).Where("id = 1").
			Update("pool", gorm.Expr("pool - ?", d.Amount)).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.User{}).Where("id = ?", d.UserID).
			Update("achieve", gorm.Expr("achieve - ?", d.Amount/100)).Error; err != nil {
			return err
		}
		return tx.Delete(&d).Error
	})
	if err != nil {
		resp.ServerError(c, err)
		return
	}
	resp.OK(c, nil)
}
