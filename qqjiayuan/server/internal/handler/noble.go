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

type NobleHandler struct{ DB *gorm.DB }

// 等级门槛（1-8级成长值）与图标
var nobleLevels = []struct {
	Lv   int
	Exp  int
	Blue string // 蓝钻图标
	QQ   string // 超Q图标
}{
	{1, 0, "noble_2_1.gif", "noble_1_1.gif"},
	{2, 600, "noble_2_2.gif", "noble_1_2.gif"},
	{3, 1800, "noble_2_3.gif", "noble_1_3.gif"},
	{4, 3600, "noble_2_4.gif", "noble_1_4.gif"},
	{5, 6000, "noble_2_5.gif", "noble_1_5.gif"},
	{6, 10800, "noble_2_6.gif", "noble_1_6.gif"},
	{7, 32400, "noble_2_7.gif", "noble_1_7.gif"},
	{8, 46800, "noble_2_8.gif", "noble_1_8.gif"},
}

// 补成长最多补几天（防刷）
const nobleCatchUpDays = 7

func lvOf(exp int) int {
	lv := 1
	for _, l := range nobleLevels {
		if exp >= l.Exp {
			lv = l.Lv
		}
	}
	return lv
}

func blueIcon(lv int) string {
	for _, l := range nobleLevels {
		if l.Lv == lv {
			return l.Blue
		}
	}
	return nobleLevels[0].Blue
}

func qqIcon(lv int) string {
	for _, l := range nobleLevels {
		if l.Lv == lv {
			return l.QQ
		}
	}
	return nobleLevels[0].QQ
}

// dailyGrow 每日成长值：会员有效期内每天按方案速度成长（补至多 nobleCatchUpDays 天）
func (h *NobleHandler) dailyGrow(uid uint, u *model.User) {
	now := time.Now()
	today := now.Format("2006-01-02")
	var blueSpeed, qqSpeed int
	// 取用户当前方案的成长速度（按最近开通的方案）
	if u.BlueEnd != nil && u.BlueEnd.After(now) {
		blueSpeed = h.userSpeed(u.ID, "blue", u.BlueEnd)
		if u.BluePtime == nil {
			u.BluePtime = u.BlueStart
		}
		if days := growDays(u.BluePtime, today); days > 0 {
			if days > nobleCatchUpDays {
				days = nobleCatchUpDays
			}
			add := blueSpeed * days
			if add > 0 {
				h.DB.Model(&model.User{}).Where("id = ?", uid).
					Updates(map[string]interface{}{
						"blue_exp":  gorm.Expr("blue_exp + ?", add),
						"blue_lv":   lvOf(u.BlueExp + add),
						"blue_ptime": now,
					})
				addWalletLog(h.DB, uid, "blue_grow", "蓝钻每日成长（"+strconv.Itoa(blueSpeed)+"点/天）", "blue_grow", add)
			}
		}
	}
	if u.QqEnd != nil && u.QqEnd.After(now) {
		qqSpeed = h.userSpeed(u.ID, "qq", u.QqEnd)
		if u.QqPtime == nil {
			u.QqPtime = u.QqStart
		}
		if days := growDays(u.QqPtime, today); days > 0 {
			if days > nobleCatchUpDays {
				days = nobleCatchUpDays
			}
			add := qqSpeed * days
			if add > 0 {
				h.DB.Model(&model.User{}).Where("id = ?", uid).
					Updates(map[string]interface{}{
						"qq_exp":  gorm.Expr("qq_exp + ?", add),
						"qq_lv":   lvOf(u.QqExp + add),
						"qq_ptime": now,
					})
				addWalletLog(h.DB, uid, "qq_grow", "超Q每日成长（"+strconv.Itoa(qqSpeed)+"点/天）", "qq_grow", add)
			}
		}
	}
}

// userSpeed 取用户当前生效方案的成长速度
func (h *NobleHandler) userSpeed(uid uint, typ string, end *time.Time) int {
	// 找该类型最近一次开通的日志（按 wallet_log kind）
	kind := "blue_open"
	if typ == "qq" {
		kind = "qq_open"
	}
	var log model.WalletLog
	if err := h.DB.Where("user_id = ? AND kind = ?", uid, kind).Order("id DESC").First(&log).Error; err == nil {
		if speed, err2 := strconv.Atoi(log.Remark); err2 == nil && speed > 0 {
			return speed
		}
	}
	return 10
}

// growDays 从上次成长时间到今天的天数
func growDays(ptime *time.Time, today string) int {
	if ptime == nil {
		return 1
	}
	p := ptime.Format("2006-01-02")
	if p >= today {
		return 0
	}
	pt, _ := time.Parse("2006-01-02", p)
	tt, _ := time.Parse("2006-01-02", today)
	d := int(tt.Sub(pt).Hours() / 24)
	if d < 0 {
		return 0
	}
	return d
}

// 到期处理：到期后保留成长值 30 天，超过则清空
func (h *NobleHandler) expireCheck(uid uint, u *model.User) {
	now := time.Now()
	grace := now.AddDate(0, 0, -30)
	if u.BlueEnd != nil && u.BlueEnd.Before(now) {
		if u.BlueEnd.Before(grace) {
			h.DB.Model(&model.User{}).Where("id = ?", uid).
				Updates(map[string]interface{}{"blue_exp": 0, "blue_lv": 0, "blue_start": nil, "blue_end": nil, "blue_ptime": nil})
		}
	}
	if u.QqEnd != nil && u.QqEnd.Before(now) {
		if u.QqEnd.Before(grace) {
			h.DB.Model(&model.User{}).Where("id = ?", uid).
				Updates(map[string]interface{}{"qq_exp": 0, "qq_lv": 0, "qq_start": nil, "qq_end": nil, "qq_ptime": nil})
		}
	}
}

// 用户端：特权中心聚合（首页/成长体系/开通/个人中心所需数据）
func (h *NobleHandler) View(c *gin.Context) {
	uid := middleware.GetUID(c)
	var u model.User
	h.DB.First(&u, uid)

	// 到期处理 + 每日成长
	h.expireCheck(uid, &u)
	h.dailyGrow(uid, &u)
	h.DB.First(&u, uid)

	var plans []model.NoblePlan
	h.DB.Order("sort ASC").Find(&plans)

	type row struct {
		UserID   uint   `json:"user_id"`
		Nickname string `json:"nickname"`
		Color    string `json:"color"`
		Exp      int    `json:"exp"`
	}
	var blueRank, qqRank []row
	h.DB.Raw(`SELECT id AS user_id, nickname, color, blue_exp AS exp FROM users WHERE blue_exp > 0 ORDER BY blue_exp DESC LIMIT 10`).Scan(&blueRank)
	h.DB.Raw(`SELECT id AS user_id, nickname, color, qq_exp AS exp FROM users WHERE qq_exp > 0 ORDER BY qq_exp DESC LIMIT 10`).Scan(&qqRank)

	resp.OK(c, gin.H{
		"coins": u.Coins, "nickname": u.Nickname,
		"blue": gin.H{"lv": lvOf(u.BlueExp), "exp": u.BlueExp, "active": u.BlueEnd != nil && u.BlueEnd.After(time.Now()),
			"icon": blueIcon(lvOf(u.BlueExp)), "start": u.BlueStart, "end": u.BlueEnd,
			"speed": h.userSpeed(u.ID, "blue", u.BlueEnd), "days_left": daysLeft(u.BlueEnd)},
		"qq": gin.H{"lv": lvOf(u.QqExp), "exp": u.QqExp, "active": u.QqEnd != nil && u.QqEnd.After(time.Now()),
			"icon": qqIcon(lvOf(u.QqExp)), "start": u.QqStart, "end": u.QqEnd,
			"speed": h.userSpeed(u.ID, "qq", u.QqEnd), "days_left": daysLeft(u.QqEnd)},
		"levels": nobleLevels, "plans": plans, "blue_rank": blueRank, "qq_rank": qqRank,
	})
}

func daysLeft(end *time.Time) int {
	if end == nil {
		return 0
	}
	d := int(end.Sub(time.Now()).Hours()/24) + 1
	if d < 0 {
		return 0
	}
	return d
}

type nobleActReq struct {
	PlanID uint `json:"plan_id" binding:"required"`
	Num    int  `json:"num"` // 购买数量（月），默认1
}

// 开通/续费（蓝钻或超Q），支持多月数量；记录销量
func (h *NobleHandler) Activate(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req nobleActReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请选择开通方案")
		return
	}
	if req.Num < 1 {
		req.Num = 1
	}
	if req.Num > 12 {
		req.Num = 12
	}
	var plan model.NoblePlan
	if err := h.DB.First(&plan, req.PlanID).Error; err != nil {
		resp.NotFound(c, "开通方案不存在")
		return
	}
	// 限购
	if plan.Limit > 0 {
		var sold int64
		h.DB.Model(&model.WalletLog{}).Where("user_id = ? AND kind = ? AND title LIKE ?", uid, plan.Type+"_open", "%"+plan.Name+"%").Count(&sold)
		if int(sold)+req.Num > plan.Limit {
			resp.ParamError(c, "该方案每号限购 "+strconv.Itoa(plan.Limit)+" 个月")
			return
		}
	}
	// 库存
	if plan.Stock > 0 && plan.Sales+req.Num > plan.Stock {
		resp.ParamError(c, "库存不足")
		return
	}
	var u model.User
	h.DB.First(&u, uid)
	totalCost := plan.Cost * req.Num
	if u.Coins < totalCost {
		resp.ParamError(c, "G币不足，开通需要 "+strconv.Itoa(totalCost)+" G币")
		return
	}
	now := time.Now()
	days := plan.Days * req.Num
	if days <= 0 {
		days = 30 * req.Num
	}
	totalGain := plan.Gain * req.Num
	kind := plan.Type + "_open"
	if plan.Type == "blue" {
		start := now
		end := now.AddDate(0, 0, days)
		ptime := now
		if u.BlueEnd != nil && u.BlueEnd.After(now) {
			start = *u.BlueStart
			end = u.BlueEnd.AddDate(0, 0, days)
			ptime = now
		}
		h.DB.Model(&u).Updates(map[string]interface{}{
			"coins":     gorm.Expr("coins - ?", totalCost),
			"blue_exp":  gorm.Expr("blue_exp + ?", totalGain),
			"blue_lv":   lvOf(u.BlueExp + totalGain),
			"blue_start": start, "blue_end": end, "blue_ptime": ptime,
		})
	} else {
		start := now
		end := now.AddDate(0, 0, days)
		ptime := now
		if u.QqEnd != nil && u.QqEnd.After(now) {
			start = *u.QqStart
			end = u.QqEnd.AddDate(0, 0, days)
			ptime = now
		}
		h.DB.Model(&u).Updates(map[string]interface{}{
			"coins":    gorm.Expr("coins - ?", totalCost),
			"qq_exp":   gorm.Expr("qq_exp + ?", totalGain),
			"qq_lv":    lvOf(u.QqExp + totalGain),
			"qq_start": start, "qq_end": end, "qq_ptime": ptime,
		})
	}
	// 销量 +1 方案
	h.DB.Model(&plan).UpdateColumn("sales", gorm.Expr("sales + ?", req.Num))
	// 记流水（remark 存成长速度，用于每日成长）
	speedStr := strconv.Itoa(plan.Speed)
	title := plan.Name
	if req.Num > 1 {
		title += "×" + strconv.Itoa(req.Num)
	}
	addWalletLog(h.DB, uid, kind, title, "coins", -totalCost)
	var lg model.WalletLog
	h.DB.Where("user_id = ? AND kind = ?", uid, kind).Order("id DESC").First(&lg)
	if lg.ID > 0 {
		h.DB.Model(&lg).Update("remark", speedStr)
	}
	resp.OK(c, gin.H{"type": plan.Type, "cost": totalCost, "gain": totalGain, "speed": plan.Speed})
}

// 赠送超Q/蓝钻给好友（复刻诺哈 shop_send：扣自己G币，好友获特权）
func (h *NobleHandler) Gift(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		PlanID uint   `json:"plan_id" binding:"required"`
		To     string `json:"to" binding:"required"`
		Num    int    `json:"num"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请选择方案和对方号码")
		return
	}
	if req.Num < 1 {
		req.Num = 1
	}
	var plan model.NoblePlan
	if err := h.DB.First(&plan, req.PlanID).Error; err != nil {
		resp.NotFound(c, "开通方案不存在")
		return
	}
	var me model.User
	h.DB.First(&me, uid)
	var to model.User
	if err := h.DB.Where("username = ?", req.To).First(&to).Error; err != nil {
		resp.ParamError(c, "对方号码不存在")
		return
	}
	if to.ID == uid {
		resp.ParamError(c, "不能送给自己哦")
		return
	}
	totalCost := plan.Cost * req.Num
	if me.Coins < totalCost {
		resp.ParamError(c, "G币不足，需要 "+strconv.Itoa(totalCost)+" G币")
		return
	}
	now := time.Now()
	days := plan.Days * req.Num
	totalGain := plan.Gain * req.Num
	if plan.Type == "blue" {
		start, end := now, now.AddDate(0, 0, days)
		if to.BlueEnd != nil && to.BlueEnd.After(now) {
			start = *to.BlueStart
			end = to.BlueEnd.AddDate(0, 0, days)
		}
		h.DB.Model(&to).Updates(map[string]interface{}{
			"blue_exp": gorm.Expr("blue_exp + ?", totalGain), "blue_lv": lvOf(to.BlueExp + totalGain),
			"blue_start": start, "blue_end": end, "blue_ptime": now,
		})
	} else {
		start, end := now, now.AddDate(0, 0, days)
		if to.QqEnd != nil && to.QqEnd.After(now) {
			start = *to.QqStart
			end = to.QqEnd.AddDate(0, 0, days)
		}
		h.DB.Model(&to).Updates(map[string]interface{}{
			"qq_exp": gorm.Expr("qq_exp + ?", totalGain), "qq_lv": lvOf(to.QqExp + totalGain),
			"qq_start": start, "qq_end": end, "qq_ptime": now,
		})
	}
	h.DB.Model(&me).Update("coins", gorm.Expr("coins - ?", totalCost))
	h.DB.Model(&plan).UpdateColumn("sales", gorm.Expr("sales + ?", req.Num))
	addWalletLog(h.DB, uid, plan.Type+"_gift", "赠送"+plan.Name+"给"+to.Nickname+"("+to.Username+")", "coins", -totalCost)
	// 通知好友
	h.DB.Create(&model.Notification{
		UserID: to.ID, Type: "system", RefID: 0,
		Title: me.Nickname + " 送了你一份厚礼", Content: "恭喜你获得「" + plan.Name + "」，快去看看吧",
	})
	resp.OK(c, gin.H{"type": plan.Type, "cost": totalCost, "to": to.Nickname})
}

// 后台：方案管理
func (h *NobleHandler) AdminPlans(c *gin.Context) {
	var list []model.NoblePlan
	h.DB.Order("sort ASC").Find(&list)
	resp.OK(c, list)
}

type planReq struct {
	Type  string `json:"type" binding:"required,oneof=blue qq"`
	Name  string `json:"name" binding:"required"`
	Cost  int    `json:"cost"`
	Gain  int    `json:"gain"`
	Speed int    `json:"speed"`
	Days  int    `json:"days"`
	Limit int    `json:"limit"`
	Stock int    `json:"stock"`
	Sales int    `json:"sales"`
	Sort  int    `json:"sort"`
}

func (h *NobleHandler) AdminPlanCreate(c *gin.Context) {
	var req planReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请填写方案名称与类型")
		return
	}
	h.DB.Create(&model.NoblePlan{Type: req.Type, Name: req.Name, Cost: req.Cost, Gain: req.Gain, Speed: req.Speed, Days: req.Days, Limit: req.Limit, Stock: req.Stock, Sales: req.Sales, Sort: req.Sort})
	resp.OK(c, nil)
}

func (h *NobleHandler) AdminPlanUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req planReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数有误")
		return
	}
	h.DB.Model(&model.NoblePlan{}).Where("id = ?", id).Updates(map[string]interface{}{
		"type": req.Type, "name": req.Name, "cost": req.Cost, "gain": req.Gain, "speed": req.Speed,
		"days": req.Days, "limit": req.Limit, "stock": req.Stock, "sales": req.Sales, "sort": req.Sort,
	})
	resp.OK(c, nil)
}

func (h *NobleHandler) AdminPlanDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.NoblePlan{}, id)
	resp.OK(c, nil)
}

// 后台：用户特权列表
func (h *NobleHandler) AdminUsers(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	var total int64
	h.DB.Model(&model.User{}).Count(&total)
	var users []model.User
	h.DB.Order("blue_exp DESC, qq_exp DESC, id ASC").Offset(offset).Limit(size).Find(&users)
	out := []gin.H{}
	for _, u := range users {
		out = append(out, gin.H{"id": u.ID, "nickname": u.Nickname, "color": u.Color, "blue_lv": lvOf(u.BlueExp), "blue_exp": u.BlueExp, "qq_lv": lvOf(u.QqExp), "qq_exp": u.QqExp})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// 一键给所有用户开通蓝钻/超Q
func (h *NobleHandler) AdminBatch(c *gin.Context) {
	var req struct {
		Type string `json:"type" binding:"required,oneof=blue qq"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请选择开通类型")
		return
	}
	if req.Type == "blue" {
		h.DB.Exec("UPDATE users SET blue_exp = GREATEST(blue_exp,100), blue_lv = 1, blue_start = NOW(), blue_end = DATE_ADD(NOW(), INTERVAL 30 DAY), blue_ptime = NOW()")
	} else {
		h.DB.Exec("UPDATE users SET qq_exp = GREATEST(qq_exp,100), qq_lv = 1, qq_start = NOW(), qq_end = DATE_ADD(NOW(), INTERVAL 30 DAY), qq_ptime = NOW()")
	}
	resp.OK(c, nil)
}

// 后台：给单个用户开通蓝钻/超Q
func (h *NobleHandler) AdminUserOpen(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Type string `json:"type" binding:"required,oneof=blue qq"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请选择开通类型")
		return
	}
	if req.Type == "blue" {
		h.DB.Exec("UPDATE users SET blue_exp = GREATEST(blue_exp,100), blue_lv = 1, blue_start = NOW(), blue_end = DATE_ADD(NOW(), INTERVAL 30 DAY), blue_ptime = NOW() WHERE id = ?", id)
	} else {
		h.DB.Exec("UPDATE users SET qq_exp = GREATEST(qq_exp,100), qq_lv = 1, qq_start = NOW(), qq_end = DATE_ADD(NOW(), INTERVAL 30 DAY), qq_ptime = NOW() WHERE id = ?", id)
	}
	resp.OK(c, nil)
}

type privUserReq struct {
	BlueExp int `json:"blue_exp"`
	QqExp   int `json:"qq_exp"`
}

// 后台：设置用户蓝钻/超Q成长
func (h *NobleHandler) AdminUserUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req privUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数有误")
		return
	}
	if req.BlueExp < 0 {
		req.BlueExp = 0
	}
	if req.QqExp < 0 {
		req.QqExp = 0
	}
	h.DB.Model(&model.User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"blue_exp": req.BlueExp, "blue_lv": lvOf(req.BlueExp),
		"qq_exp": req.QqExp, "qq_lv": lvOf(req.QqExp),
	})
	resp.OK(c, nil)
}