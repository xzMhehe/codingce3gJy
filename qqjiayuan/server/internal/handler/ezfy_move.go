package handler

import (
	"fmt"
	"math/rand"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 二战风云 城市迁移（复刻原版 city/cityHallMove.html）
//
// 原版页面提供三种迁城方式，但原 Java 工程只实现了「花 20 万黄金 + 随机空位」一种，
// 区域/指定坐标/沿海三个分支在模板里有、后端没有。这里按页面语义补全：
//   low  迁城计划     —— 选区域(西欧/东欧/西亚/东亚/南亚/北非/南非/澳洲/北美/南美)，落该区域内的随机空平原
//   high 高级迁城计划 —— 指定 x/y，必须是未被占领的平原（非沿海）
//   sea  沿海迁城计划 —— 指定 x/y，必须是未被占领的沿海平原
//
// 三种方式均消耗 20 万黄金（沿用原 Java 的迁城费用）。
const ezfyMoveCityGoldCost = 200000

// ezfyMoveArea 迁城区域（把城市可落点的 50~450 区域按 5×2 均分，
// 原版 10 个区域名沿用页面文案）
type ezfyMoveArea struct {
	ID   int
	Name string
	X0   int
	Y0   int
	X1   int
	Y1   int
}

var ezfyMoveAreas = []ezfyMoveArea{
	{1, "西欧", 50, 50, 130, 250},
	{2, "东欧", 131, 50, 210, 250},
	{3, "西亚", 211, 50, 290, 250},
	{4, "东亚", 291, 50, 370, 250},
	{5, "南亚", 371, 50, 450, 250},
	{6, "北非", 50, 251, 130, 450},
	{7, "南非", 131, 251, 210, 450},
	{8, "澳洲", 211, 251, 290, 450},
	{9, "北美", 291, 251, 370, 450},
	{10, "南美", 371, 251, 450, 450},
}

// MoveInfo GET /games/ezfy/city/move —— 迁城页数据(区域列表/费用/当前坐标)
func (h *EzfyHandler) MoveInfo(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	city := h.getOrCreateCity(uid)
	h.refreshCity(uid, &city)
	areas := make([]gin.H, 0, len(ezfyMoveAreas))
	for _, a := range ezfyMoveAreas {
		areas = append(areas, gin.H{"id": a.ID, "name": a.Name})
	}
	resp.OK(c, gin.H{
		"areas": areas, "gold_cost": ezfyMoveCityGoldCost, "gold": city.Gold,
		"city": gin.H{"id": city.ID, "name": city.Name, "x": city.X, "y": city.Y},
	})
}

// MoveCity POST /games/ezfy/city/move  {city_id, type: low|high|sea, area_id, x, y}
func (h *EzfyHandler) MoveCity(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		CityId int64  `json:"city_id"`
		Type   string `json:"type"`
		AreaId int    `json:"area_id"`
		X      int    `json:"x"`
		Y      int    `json:"y"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	h.cfgs()
	city := h.bodyCity(uid, req.CityId)
	h.refreshCity(uid, city)
	if city.Gold < ezfyMoveCityGoldCost {
		resp.ParamError(c, fmt.Sprintf("迁城需要%d黄金, 当前只有%d", ezfyMoveCityGoldCost, city.Gold))
		return
	}

	var tx, ty int
	switch req.Type {
	case "low":
		area := ezfyMoveAreaOf(req.AreaId)
		if area == nil {
			resp.ParamError(c, "请选择迁城区域")
			return
		}
		x, y, ok := h.findFreePlain(*area)
		if !ok {
			resp.ParamError(c, "该区域暂时没有可用空地, 请换个区域")
			return
		}
		tx, ty = x, y
	case "high", "sea":
		if msg := h.checkMoveTarget(req.X, req.Y, req.Type == "sea"); msg != "" {
			resp.ParamError(c, msg)
			return
		}
		tx, ty = req.X, req.Y
	default:
		resp.ParamError(c, "迁城方式不正确")
		return
	}

	oldX, oldY := city.X, city.Y
	city.Gold -= ezfyMoveCityGoldCost
	city.X, city.Y = tx, ty
	h.saveCityRes(city)
	h.DB.Model(&model.EzfyCity{}).Where("id = ?", city.ID).Updates(map[string]interface{}{"x": tx, "y": ty})

	h.addReport(uid, 6, "城市迁移完成",
		fmt.Sprintf("消耗%d黄金, 城市[%s]已从(%d,%d)迁移到(%d,%d)。\n附属野地不会随城迁移, 请重新占领。",
			ezfyMoveCityGoldCost, city.Name, oldX, oldY, tx, ty))
	resp.OK(c, gin.H{"msg": fmt.Sprintf("迁城成功, 新坐标(%d,%d)", tx, ty), "x": tx, "y": ty})
}

func ezfyMoveAreaOf(id int) *ezfyMoveArea {
	for i := range ezfyMoveAreas {
		if ezfyMoveAreas[i].ID == id {
			return &ezfyMoveAreas[i]
		}
	}
	return nil
}

// findFreePlain 在区域内随机找一个「未被占领的平原」
func (h *EzfyHandler) findFreePlain(a ezfyMoveArea) (int, int, bool) {
	w := a.X1 - a.X0 + 1
	hgt := a.Y1 - a.Y0 + 1
	for i := 0; i < 3000; i++ {
		x := a.X0 + rand.Intn(w)
		y := a.Y0 + rand.Intn(hgt)
		if ezfyTerrain(x, y) != 1 { // 1 = 平原
			continue
		}
		var n int64
		h.DB.Model(&model.EzfyCity{}).Where("x = ? AND y = ?", x, y).Count(&n)
		if n > 0 {
			continue
		}
		return x, y, true
	}
	return 0, 0, false
}

// checkMoveTarget 校验指定坐标能否迁城；needCoastal=true 时必须是沿海平原
func (h *EzfyHandler) checkMoveTarget(x, y int, needCoastal bool) string {
	if x < 1 || x > 500 || y < 1 || y > 500 {
		return "坐标需在 1~500 之间"
	}
	if ezfyTerrain(x, y) != 1 {
		return "只能迁移到平原"
	}
	var n int64
	h.DB.Model(&model.EzfyCity{}).Where("x = ? AND y = ?", x, y).Count(&n)
	if n > 0 {
		return "该坐标已有城市"
	}
	probe := model.EzfyCity{X: x, Y: y}
	coastal := h.isCoastalCity(&probe)
	if needCoastal && !coastal {
		return "沿海迁城只能迁移到沿海平原"
	}
	if !needCoastal && coastal {
		return "该坐标是沿海平原, 请使用沿海迁城计划"
	}
	return ""
}
