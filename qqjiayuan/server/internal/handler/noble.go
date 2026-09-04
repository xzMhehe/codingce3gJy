package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

type NobleHandler struct{ DB *gorm.DB }

// 开通方案（对齐参考站特权开通）
type noblePlan struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Level int    `json:"level"` // 1蓝钻 2超Q
	Cost  int    `json:"cost"`
	Gain  int    `json:"gain"` // 成长值
}

var noblePlans = []noblePlan{
	{1, "包月蓝钻等级加速", 1, 500, 100},
	{2, "包月超Q等级加速", 2, 1000, 200},
	{3, "年费蓝钻等级飞速", 1, 5000, 1200},
	{4, "年费超Q等级飞速", 2, 8000, 2400},
}

// 用户端：超Q/特权中心
func (h *NobleHandler) View(c *gin.Context) {
	uid := middleware.GetUID(c)
	var u model.User
	h.DB.First(&u, uid)

	type row struct {
		UserID   uint   `json:"user_id"`
		Nickname string `json:"nickname"`
		Color    string `json:"color"`
		Exp      int    `json:"noble_exp"`
	}
	var rank []row
	h.DB.Raw(`SELECT id AS user_id, nickname, color, noble_exp AS exp FROM users WHERE noble > 0 ORDER BY noble_exp DESC LIMIT 10`).Scan(&rank)

	resp.OK(c, gin.H{
		"noble": u.Noble, "noble_exp": u.NobleExp, "coins": u.Coins,
		"priv": u.Priv, "plans": noblePlans, "rank": rank,
	})
}

type nobleActReq struct {
	PlanID int `json:"plan_id" binding:"required"`
}

// 开通/续费超Q
func (h *NobleHandler) Activate(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req nobleActReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请选择开通方案")
		return
	}
	var plan *noblePlan
	for i := range noblePlans {
		if noblePlans[i].ID == req.PlanID {
			plan = &noblePlans[i]
			break
		}
	}
	if plan == nil {
		resp.ParamError(c, "请选择开通方案")
		return
	}
	var u model.User
	h.DB.First(&u, uid)
	if u.Coins < plan.Cost {
		resp.ParamError(c, "金币不足，开通需要 " + strconv.Itoa(plan.Cost) + " 金币")
		return
	}
	newLevel := u.Noble
	if plan.Level > newLevel {
		newLevel = plan.Level
	}
	h.DB.Model(&u).Updates(map[string]interface{}{
		"coins":      gorm.Expr("coins - ?", plan.Cost),
		"noble":      newLevel,
		"noble_exp":  gorm.Expr("noble_exp + ?", plan.Gain),
	})
	resp.OK(c, gin.H{"noble": newLevel, "noble_exp": u.NobleExp + plan.Gain, "coins": u.Coins - plan.Cost})
}

// 后台：超Q用户列表（带超Q状态）
func (h *NobleHandler) AdminList(c *gin.Context) {
	page, offset := pageOf(c, 20)
	var total int64
	h.DB.Model(&model.User{}).Count(&total)
	var users []model.User
	h.DB.Where("noble > 0 OR noble_exp > 0").Order("noble_exp DESC").Offset(offset).Limit(20).Find(&users)
	out := []gin.H{}
	for _, u := range users {
		out = append(out, gin.H{"id": u.ID, "nickname": u.Nickname, "color": u.Color, "noble": u.Noble, "noble_exp": u.NobleExp})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page})
}

type nobleAdminReq struct {
	Noble    int `json:"noble"`
	NobleExp int `json:"noble_exp"`
}

// 后台：设置用户超Q等级/成长
func (h *NobleHandler) AdminUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req nobleAdminReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数有误")
		return
	}
	if req.Noble < 0 {
		req.Noble = 0
	}
	if req.Noble > 2 {
		req.Noble = 2
	}
	if req.NobleExp < 0 {
		req.NobleExp = 0
	}
	h.DB.Model(&model.User{}).Where("id = ?", id).Updates(map[string]interface{}{"noble": req.Noble, "noble_exp": req.NobleExp})
	resp.OK(c, nil)
}
