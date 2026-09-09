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

// 补成长最多补几天（防刷）
const nobleCatchUpDays = 7

// nobleLevels 读取贵宾等级配置（复刻诺哈 wap_vip_config：id=等级, point=升级经验, 图标）
func (h *NobleHandler) nobleLevels() []model.NobleLevel {
	var ls []model.NobleLevel
	h.DB.Order("id ASC").Find(&ls)
	return ls
}

// nobleLv 按等级配置算贵宾等级：成长值<=0 为 0 级（未开通/无成长），否则取满足门槛的最高级
func nobleLv(ls []model.NobleLevel, exp int) int {
	if exp <= 0 {
		return 0
	}
	return model.NobleLvOf(ls, exp)
}

// dailyGrow 每日成长值：会员有效期内每天按成长速度成长（补至多 nobleCatchUpDays 天）
func (h *NobleHandler) dailyGrow(uid uint, u *model.User) {
	if u.BlueEnd == nil && u.QqEnd == nil {
		return
	}
	now := time.Now()
	today := now.Format("2006-01-02")
	levels := h.nobleLevels()
	if u.BlueEnd != nil && u.BlueEnd.After(now) {
		blueSpeed := u.BlueSpeed
		if blueSpeed <= 0 {
			blueSpeed = h.userSpeed(u.ID, "blue")
		}
		if u.BluePtime == nil {
			u.BluePtime = u.BlueStart
		}
		if days := growDays(u.BluePtime, today); days > 0 {
			if days > nobleCatchUpDays {
				days = nobleCatchUpDays
			}
			add := blueSpeed * days
			if add > 0 {
				newExp := u.BlueExp + add
				h.DB.Model(&model.User{}).Where("id = ?", uid).
					Updates(map[string]interface{}{
						"blue_exp":   gorm.Expr("blue_exp + ?", add),
						"blue_lv":    nobleLv(levels, newExp),
						"blue_ptime": now,
					})
				addWalletLog(h.DB, uid, "blue_grow", "蓝钻每日成长（"+strconv.Itoa(blueSpeed)+"点/天）", "blue_grow", add)
			}
		}
	}
	if u.QqEnd != nil && u.QqEnd.After(now) {
		qqSpeed := u.QqSpeed
		if qqSpeed <= 0 {
			qqSpeed = h.userSpeed(u.ID, "qq")
		}
		if u.QqPtime == nil {
			u.QqPtime = u.QqStart
		}
		if days := growDays(u.QqPtime, today); days > 0 {
			if days > nobleCatchUpDays {
				days = nobleCatchUpDays
			}
			add := qqSpeed * days
			if add > 0 {
				newExp := u.QqExp + add
				h.DB.Model(&model.User{}).Where("id = ?", uid).
					Updates(map[string]interface{}{
						"qq_exp":   gorm.Expr("qq_exp + ?", add),
						"qq_lv":    nobleLv(levels, newExp),
						"qq_ptime": now,
					})
				addWalletLog(h.DB, uid, "qq_grow", "超Q每日成长（"+strconv.Itoa(qqSpeed)+"点/天）", "qq_grow", add)
			}
		}
	}
}

// userSpeed 取用户当前生效方案的成长速度（优先用户列，其次最近一次开通日志备注）
func (h *NobleHandler) userSpeed(uid uint, typ string) int {
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
				Updates(map[string]interface{}{"blue_exp": 0, "blue_lv": 0, "blue_speed": 0, "blue_start": nil, "blue_end": nil, "blue_ptime": nil})
		}
	}
	if u.QqEnd != nil && u.QqEnd.Before(now) {
		if u.QqEnd.Before(grace) {
			h.DB.Model(&model.User{}).Where("id = ?", uid).
				Updates(map[string]interface{}{"qq_exp": 0, "qq_lv": 0, "qq_speed": 0, "qq_start": nil, "qq_end": nil, "qq_ptime": nil})
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

	levels := h.nobleLevels()

	var plans []model.NoblePlan
	h.DB.Order("sort ASC, id ASC").Find(&plans)

	type row struct {
		UserID   uint   `json:"user_id"`
		Nickname string `json:"nickname"`
		Color    string `json:"color"`
		Exp      int    `json:"exp"`
	}
	var blueRank, qqRank []row
	h.DB.Raw(`SELECT id AS user_id, nickname, color, blue_exp AS exp FROM users WHERE blue_exp > 0 ORDER BY blue_exp DESC LIMIT 10`).Scan(&blueRank)
	h.DB.Raw(`SELECT id AS user_id, nickname, color, qq_exp AS exp FROM users WHERE qq_exp > 0 ORDER BY qq_exp DESC LIMIT 10`).Scan(&qqRank)

	// 等级表（复刻诺哈 vip/index.asp 会员介绍：等级/图标/成长值）
	levelOut := make([]gin.H, 0, len(levels))
	for _, l := range levels {
		levelOut = append(levelOut, gin.H{"Lv": l.ID, "Exp": l.Point, "Blue": l.IconBlue, "QQ": l.IconQQ})
	}
	blueSpeed, qqSpeed := u.BlueSpeed, u.QqSpeed
	if blueSpeed <= 0 {
		blueSpeed = h.userSpeed(u.ID, "blue")
	}
	if qqSpeed <= 0 {
		qqSpeed = h.userSpeed(u.ID, "qq")
	}

	resp.OK(c, gin.H{
		"coins": u.Coins, "nickname": u.Nickname,
		"blue": gin.H{"lv": nobleLv(levels, u.BlueExp), "exp": u.BlueExp, "active": u.BlueEnd != nil && u.BlueEnd.After(time.Now()),
			"icon": model.NobleIconOf(levels, nobleLv(levels, u.BlueExp), "blue"), "start": u.BlueStart, "end": u.BlueEnd,
			"speed": blueSpeed, "days_left": daysLeft(u.BlueEnd)},
		"qq": gin.H{"lv": nobleLv(levels, u.QqExp), "exp": u.QqExp, "active": u.QqEnd != nil && u.QqEnd.After(time.Now()),
			"icon": model.NobleIconOf(levels, nobleLv(levels, u.QqExp), "qq"), "start": u.QqStart, "end": u.QqEnd,
			"speed": qqSpeed, "days_left": daysLeft(u.QqEnd)},
		"levels": levelOut, "plans": plans, "blue_rank": blueRank, "qq_rank": qqRank,
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
	Num    int  `json:"num"` // 购买数量（月/份），默认1
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
	levels := h.nobleLevels()
	if plan.Type == "blue" {
		start := now
		end := now.AddDate(0, 0, days)
		ptime := now
		if u.BlueEnd != nil && u.BlueEnd.After(now) {
			start = *u.BlueStart
			end = u.BlueEnd.AddDate(0, 0, days)
		}
		h.DB.Model(&u).Updates(map[string]interface{}{
			"coins":      gorm.Expr("coins - ?", totalCost),
			"blue_exp":   gorm.Expr("blue_exp + ?", totalGain),
			"blue_lv":    nobleLv(levels, u.BlueExp+totalGain),
			"blue_speed": plan.Speed,
			"blue_start": start, "blue_end": end, "blue_ptime": ptime,
		})
	} else {
		start := now
		end := now.AddDate(0, 0, days)
		ptime := now
		if u.QqEnd != nil && u.QqEnd.After(now) {
			start = *u.QqStart
			end = u.QqEnd.AddDate(0, 0, days)
		}
		h.DB.Model(&u).Updates(map[string]interface{}{
			"coins":     gorm.Expr("coins - ?", totalCost),
			"qq_exp":    gorm.Expr("qq_exp + ?", totalGain),
			"qq_lv":     nobleLv(levels, u.QqExp+totalGain),
			"qq_speed":  plan.Speed,
			"qq_start":  start, "qq_end": end, "qq_ptime": ptime,
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
	levels := h.nobleLevels()
	if plan.Type == "blue" {
		start, end := now, now.AddDate(0, 0, days)
		if to.BlueEnd != nil && to.BlueEnd.After(now) {
			start = *to.BlueStart
			end = to.BlueEnd.AddDate(0, 0, days)
		}
		h.DB.Model(&to).Updates(map[string]interface{}{
			"blue_exp": gorm.Expr("blue_exp + ?", totalGain), "blue_lv": nobleLv(levels, to.BlueExp+totalGain),
			"blue_speed": plan.Speed, "blue_start": start, "blue_end": end, "blue_ptime": now,
		})
	} else {
		start, end := now, now.AddDate(0, 0, days)
		if to.QqEnd != nil && to.QqEnd.After(now) {
			start = *to.QqStart
			end = to.QqEnd.AddDate(0, 0, days)
		}
		h.DB.Model(&to).Updates(map[string]interface{}{
			"qq_exp": gorm.Expr("qq_exp + ?", totalGain), "qq_lv": nobleLv(levels, to.QqExp+totalGain),
			"qq_speed": plan.Speed, "qq_start": start, "qq_end": end, "qq_ptime": now,
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

// 后台：特权统计（概览用）
func (h *NobleHandler) AdminStats(c *gin.Context) {
	now := time.Now()
	var blueTotal, blueActive, qqTotal, qqActive, planCount, expired int64
	h.DB.Model(&model.User{}).Where("blue_exp > 0").Count(&blueTotal)
	h.DB.Model(&model.User{}).Where("blue_exp > 0 AND blue_end > ?", now).Count(&blueActive)
	h.DB.Model(&model.User{}).Where("qq_exp > 0").Count(&qqTotal)
	h.DB.Model(&model.User{}).Where("qq_exp > 0 AND qq_end > ?", now).Count(&qqActive)
	h.DB.Model(&model.NoblePlan{}).Where("status = 1").Count(&planCount)
	h.DB.Model(&model.User{}).Where("(blue_exp > 0 AND blue_end IS NOT NULL AND blue_end < ?) OR (qq_exp > 0 AND qq_end IS NOT NULL AND qq_end < ?)", now, now).Count(&expired)
	resp.OK(c, gin.H{
		"blue_total": blueTotal, "blue_active": blueActive,
		"qq_total": qqTotal, "qq_active": qqActive,
		"plan_count": planCount, "expired": expired,
	})
}

// ============ 后台：贵宾等级（复刻诺哈 wap_vip_config：升级经验 + 图标） ============

func (h *NobleHandler) AdminLevels(c *gin.Context) {
	resp.OK(c, h.nobleLevels())
}

func (h *NobleHandler) AdminLevelCreate(c *gin.Context) {
	var req struct {
		Point    int    `json:"point"`
		IconBlue string `json:"icon_blue"`
		IconQQ   string `json:"icon_qq"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数有误")
		return
	}
	var id uint
	h.DB.Raw("SELECT COALESCE(MAX(id),0)+1 FROM noble_levels").Scan(&id)
	h.DB.Create(&model.NobleLevel{ID: id, Point: req.Point, IconBlue: req.IconBlue, IconQQ: req.IconQQ})
	resp.OK(c, gin.H{"id": id})
}

func (h *NobleHandler) AdminLevelUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Point    int    `json:"point"`
		IconBlue string `json:"icon_blue"`
		IconQQ   string `json:"icon_qq"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数有误")
		return
	}
	h.DB.Model(&model.NobleLevel{}).Where("id = ?", id).Updates(map[string]interface{}{
		"point": req.Point, "icon_blue": req.IconBlue, "icon_qq": req.IconQQ,
	})
	resp.OK(c, nil)
}

func (h *NobleHandler) AdminLevelDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 1 {
		resp.ParamError(c, "1级贵宾等级不可删除")
		return
	}
	h.DB.Delete(&model.NobleLevel{}, id)
	resp.OK(c, nil)
}

// ============ 后台：贵宾销售/开通方案（复刻诺哈 wap_vip_shop） ============

// 后台：方案管理
func (h *NobleHandler) AdminPlans(c *gin.Context) {
	var list []model.NoblePlan
	h.DB.Order("sort ASC, id ASC").Find(&list)
	resp.OK(c, list)
}

type planReq struct {
	Type   string `json:"type" binding:"required,oneof=blue qq"`
	Name   string `json:"name" binding:"required"`
	Money  int    `json:"money"`  // 币种 1=G币
	Cost   int    `json:"cost"`   // 销售价格（G币/月）
	Gain   int    `json:"gain"`   // 赠送经验
	Speed  int    `json:"speed"`  // 成长速度（点/天）
	Days   int    `json:"days"`   // 周期天数（30=包月，365=年费）
	Limit  int    `json:"limit"`  // 每号限购
	Stock  int    `json:"stock"`  // 库存数量
	Sales  int    `json:"sales"`  // 销售数量
	Stime  string `json:"stime"`  // 出售时间
	Etime  string `json:"etime"`  // 结束时间
	Status int    `json:"status"` // 销售状态 1上架 0下架
	Sort   int    `json:"sort"`
}

func (h *NobleHandler) AdminPlanCreate(c *gin.Context) {
	var req planReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请填写方案名称与类型")
		return
	}
	stime, etime := parseDate(req.Stime), parseDate(req.Etime)
	h.DB.Create(&model.NoblePlan{Type: req.Type, Name: req.Name, Money: req.Money, Cost: req.Cost, Gain: req.Gain, Speed: req.Speed,
		Days: req.Days, Limit: req.Limit, Stock: req.Stock, Sales: req.Sales, Stime: stime, Etime: etime, Status: req.Status, Sort: req.Sort})
	resp.OK(c, nil)
}

func (h *NobleHandler) AdminPlanUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req planReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数有误")
		return
	}
	stime, etime := parseDate(req.Stime), parseDate(req.Etime)
	h.DB.Model(&model.NoblePlan{}).Where("id = ?", id).Updates(map[string]interface{}{
		"type": req.Type, "name": req.Name, "money": req.Money, "cost": req.Cost, "gain": req.Gain,
		"speed": req.Speed, "days": req.Days, "limit": req.Limit, "stock": req.Stock, "sales": req.Sales,
		"stime": stime, "etime": etime, "status": req.Status, "sort": req.Sort,
	})
	resp.OK(c, nil)
}

func (h *NobleHandler) AdminPlanDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.NoblePlan{}, id)
	resp.OK(c, nil)
}

// parseDate 解析 "2006-01-02 15:04:05" / RFC3339 时间字符串，空返回 nil
func parseDate(s string) *time.Time {
	s = trimQuote(s)
	if s == "" {
		return nil
	}
	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02T15:04:05", time.RFC3339, "2006-01-02 15:04:05.999999", "2006-01-02"} {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return &t
		}
	}
	return nil
}

func trimQuote(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// ============ 后台：贵宾会员（复刻诺哈 wap_vip 用户 VIP） ============

// 后台：贵宾会员列表（复刻诺哈 user_list：仅列出开通会员的用户，含等级/成长/速度/开通时间/结束时间）
func (h *NobleHandler) AdminUsers(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	kw := trimQuote(c.Query("kw"))
	q := h.DB.Model(&model.User{})
	if kw != "" {
		q = q.Where("username LIKE ? OR nickname LIKE ?", "%"+kw+"%", "%"+kw+"%")
	}
	q = q.Where("blue_exp > 0 OR qq_exp > 0")
	var total int64
	q.Count(&total)
	var users []model.User
	q.Order("GREATEST(blue_exp, qq_exp) DESC, id ASC").Offset(offset).Limit(size).Find(&users)
	levels := h.nobleLevels()
	now := time.Now()
	out := []gin.H{}
	for _, u := range users {
		blueLv, qqLv := nobleLv(levels, u.BlueExp), nobleLv(levels, u.QqExp)
		out = append(out, gin.H{
			"id": u.ID, "nickname": u.Nickname, "color": u.Color,
			"blue_lv": blueLv, "blue_exp": u.BlueExp, "blue_speed": u.BlueSpeed,
			"blue_start": u.BlueStart, "blue_end": u.BlueEnd, "blue_ptime": u.BluePtime,
			"blue_icon": model.NobleIconOf(levels, blueLv, "blue"),
			"blue_active": u.BlueEnd != nil && u.BlueEnd.After(now),
			"qq_lv": qqLv, "qq_exp": u.QqExp, "qq_speed": u.QqSpeed,
			"qq_start": u.QqStart, "qq_end": u.QqEnd, "qq_ptime": u.QqPtime,
			"qq_icon": model.NobleIconOf(levels, qqLv, "qq"),
			"qq_active": u.QqEnd != nil && u.QqEnd.After(now),
		})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

type privItem struct {
	Lv    int     `json:"lv"` // 等级（0/留空=按成长值算；负数或0且exp=0=取消）
	Exp   int     `json:"exp"`
	Speed int     `json:"speed"`
	Ptime *string `json:"ptime"` // 成长时间
	Start *string `json:"start"` // 开通时间
	End   *string `json:"end"`   // 结束时间
}

type privUserReq struct {
	Blue *privItem `json:"blue"`
	Qq   *privItem `json:"qq"`
}

// 后台：编辑用户贵宾（复刻诺哈 user_edit_ok：等级/成长值/成长速度/成长时间/速度时间/开通时间/结束时间）
func (h *NobleHandler) AdminUserUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req privUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数有误")
		return
	}
	levels := h.nobleLevels()
	updates := map[string]interface{}{}
	if req.Blue != nil {
		updates["blue_exp"] = req.Blue.Exp
		updates["blue_ptime"] = parseDate(derefStr(req.Blue.Ptime))
		updates["blue_start"] = parseDate(derefStr(req.Blue.Start))
		updates["blue_end"] = parseDate(derefStr(req.Blue.End))
		updates["blue_speed"] = req.Blue.Speed
		if req.Blue.Exp <= 0 && req.Blue.Lv <= 0 {
			updates["blue_lv"] = 0
			updates["blue_speed"] = 0
			updates["blue_start"] = nil
			updates["blue_end"] = nil
			updates["blue_ptime"] = nil
		} else if req.Blue.Lv > 0 {
			updates["blue_lv"] = req.Blue.Lv
		} else {
			updates["blue_lv"] = nobleLv(levels, req.Blue.Exp)
		}
	}
	if req.Qq != nil {
		updates["qq_exp"] = req.Qq.Exp
		updates["qq_ptime"] = parseDate(derefStr(req.Qq.Ptime))
		updates["qq_start"] = parseDate(derefStr(req.Qq.Start))
		updates["qq_end"] = parseDate(derefStr(req.Qq.End))
		updates["qq_speed"] = req.Qq.Speed
		if req.Qq.Exp <= 0 && req.Qq.Lv <= 0 {
			updates["qq_lv"] = 0
			updates["qq_speed"] = 0
			updates["qq_start"] = nil
			updates["qq_end"] = nil
			updates["qq_ptime"] = nil
		} else if req.Qq.Lv > 0 {
			updates["qq_lv"] = req.Qq.Lv
		} else {
			updates["qq_lv"] = nobleLv(levels, req.Qq.Exp)
		}
	}
	if len(updates) == 0 {
		resp.ParamError(c, "参数有误")
		return
	}
	h.DB.Model(&model.User{}).Where("id = ?", id).Updates(updates)
	resp.OK(c, nil)
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// 一键给所有用户开通蓝钻/超Q（复刻诺哈批量开通）
func (h *NobleHandler) AdminBatch(c *gin.Context) {
	var req struct {
		Type string `json:"type" binding:"required,oneof=blue qq"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请选择开通类型")
		return
	}
	if req.Type == "blue" {
		h.DB.Exec("UPDATE users SET blue_exp = GREATEST(blue_exp,100), blue_lv = 1, blue_start = NOW(), blue_end = DATE_ADD(NOW(), INTERVAL 30 DAY), blue_ptime = NOW() WHERE blue_exp < 100")
	} else {
		h.DB.Exec("UPDATE users SET qq_exp = GREATEST(qq_exp,100), qq_lv = 1, qq_start = NOW(), qq_end = DATE_ADD(NOW(), INTERVAL 30 DAY), qq_ptime = NOW() WHERE qq_exp < 100")
	}
	resp.OK(c, nil)
}

// 后台：给单个用户开通蓝钻/超Q（复刻诺哈 user_add：等级1/成长值100/一个月）
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

// 后台：取消用户贵宾（复刻诺哈 user_del_ok：删除会员并清零）
func (h *NobleHandler) AdminUserClose(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Type string `json:"type" binding:"required,oneof=blue qq"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请选择取消类型")
		return
	}
	if req.Type == "blue" {
		h.DB.Model(&model.User{}).Where("id = ?", id).Updates(map[string]interface{}{
			"blue_exp": 0, "blue_lv": 0, "blue_speed": 0, "blue_start": nil, "blue_end": nil, "blue_ptime": nil,
		})
	} else {
		h.DB.Model(&model.User{}).Where("id = ?", id).Updates(map[string]interface{}{
			"qq_exp": 0, "qq_lv": 0, "qq_speed": 0, "qq_start": nil, "qq_end": nil, "qq_ptime": nil,
		})
	}
	resp.OK(c, nil)
}

// 后台：清除到期贵宾会员（复刻诺哈 user_bdel_ok：到期即清空）
func (h *NobleHandler) AdminExpiredClean(c *gin.Context) {
	h.DB.Exec("UPDATE users SET blue_exp=0, blue_lv=0, blue_speed=0, blue_start=NULL, blue_end=NULL, blue_ptime=NULL WHERE blue_end IS NOT NULL AND blue_end < NOW()")
	h.DB.Exec("UPDATE users SET qq_exp=0, qq_lv=0, qq_speed=0, qq_start=NULL, qq_end=NULL, qq_ptime=NULL WHERE qq_end IS NOT NULL AND qq_end < NOW()")
	resp.OK(c, nil)
}
