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

// 每天成长速度
const nobleSpeed = 10

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

// 用户端：特权中心聚合（首页/成长体系/开通/个人中心所需数据）
func (h *NobleHandler) View(c *gin.Context) {
	uid := middleware.GetUID(c)
	var u model.User
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
		"blue": gin.H{"lv": lvOf(u.BlueExp), "exp": u.BlueExp, "active": u.BlueExp > 0,
			"icon": blueIcon(lvOf(u.BlueExp)), "start": u.BlueStart, "end": u.BlueEnd, "speed": nobleSpeed},
		"qq": gin.H{"lv": lvOf(u.QqExp), "exp": u.QqExp, "active": u.QqExp > 0,
			"icon": qqIcon(lvOf(u.QqExp)), "start": u.QqStart, "end": u.QqEnd, "speed": nobleSpeed},
		"levels": nobleLevels, "plans": plans, "blue_rank": blueRank, "qq_rank": qqRank,
	})
}

type nobleActReq struct {
	PlanID uint `json:"plan_id" binding:"required"`
}

// 开通/续费（蓝钻或超Q）
func (h *NobleHandler) Activate(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req nobleActReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请选择开通方案")
		return
	}
	var plan model.NoblePlan
	if err := h.DB.First(&plan, req.PlanID).Error; err != nil {
		resp.NotFound(c, "开通方案不存在")
		return
	}
	var u model.User
	h.DB.First(&u, uid)
	if u.Coins < plan.Cost {
		resp.ParamError(c, "G币不足，开通需要 "+strconv.Itoa(plan.Cost)+" G币")
		return
	}
	now := time.Now()
	days := plan.Days
	if days <= 0 {
		days = 30
	}
	if plan.Type == "blue" {
		start := now
		end := now.AddDate(0, 0, days)
		if u.BlueEnd != nil && u.BlueEnd.After(now) {
			start = *u.BlueStart
			end = u.BlueEnd.AddDate(0, 0, days)
		}
		h.DB.Model(&u).Updates(map[string]interface{}{
			"coins": gorm.Expr("coins - ?", plan.Cost),
			"blue_exp": gorm.Expr("blue_exp + ?", plan.Gain),
			"blue_lv": lvOf(u.BlueExp + plan.Gain),
			"blue_start": start, "blue_end": end,
		})
	} else {
		start := now
		end := now.AddDate(0, 0, days)
		if u.QqEnd != nil && u.QqEnd.After(now) {
			start = *u.QqStart
			end = u.QqEnd.AddDate(0, 0, days)
		}
		h.DB.Model(&u).Updates(map[string]interface{}{
			"coins": gorm.Expr("coins - ?", plan.Cost),
			"qq_exp": gorm.Expr("qq_exp + ?", plan.Gain),
			"qq_lv": lvOf(u.QqExp + plan.Gain),
			"qq_start": start, "qq_end": end,
		})
	}
	resp.OK(c, gin.H{"type": plan.Type, "cost": plan.Cost, "gain": plan.Gain})
}

// 后台：方案管理
func (h *NobleHandler) AdminPlans(c *gin.Context) {
	var list []model.NoblePlan
	h.DB.Order("sort ASC").Find(&list)
	resp.OK(c, list)
}

type planReq struct {
	Type string `json:"type" binding:"required,oneof=blue qq"`
	Name string `json:"name" binding:"required"`
	Cost int    `json:"cost"`
	Gain int    `json:"gain"`
	Days int    `json:"days"`
	Sort int    `json:"sort"`
}

func (h *NobleHandler) AdminPlanCreate(c *gin.Context) {
	var req planReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请填写方案名称与类型")
		return
	}
	h.DB.Create(&model.NoblePlan{Type: req.Type, Name: req.Name, Cost: req.Cost, Gain: req.Gain, Days: req.Days, Sort: req.Sort})
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
		"type": req.Type, "name": req.Name, "cost": req.Cost, "gain": req.Gain, "days": req.Days, "sort": req.Sort,
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
		h.DB.Exec("UPDATE users SET blue_exp = GREATEST(blue_exp,100), blue_lv = 1, blue_start = NOW(), blue_end = DATE_ADD(NOW(), INTERVAL 30 DAY)")
	} else {
		h.DB.Exec("UPDATE users SET qq_exp = GREATEST(qq_exp,100), qq_lv = 1, qq_start = NOW(), qq_end = DATE_ADD(NOW(), INTERVAL 30 DAY)")
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
		h.DB.Exec("UPDATE users SET blue_exp = GREATEST(blue_exp,100), blue_lv = 1, blue_start = NOW(), blue_end = DATE_ADD(NOW(), INTERVAL 30 DAY) WHERE id = ?", id)
	} else {
		h.DB.Exec("UPDATE users SET qq_exp = GREATEST(qq_exp,100), qq_lv = 1, qq_start = NOW(), qq_end = DATE_ADD(NOW(), INTERVAL 30 DAY) WHERE id = ?", id)
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
