package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 二战风云 仓库保护（复刻设计文档《二战风云建筑升级一览.txt》·仓库段）
//
// 设计原文：
//   "仓库是城市用来储存资源的地方，仓库等级越高，可存储的资源量越大，
//    玩家可在仓库中调配存储4项资源的比例，存储的资源不会被敌人抢夺走。"
//
// 本实现：仓库(建筑 12)等级 → 保护总量；玩家可调配 4 项资源的保护占比(合计≤100%)；
// 掠夺玩家城市时，每项资源先扣除各自的保护额度，只掠夺超出部分。
// 注：原 Java 工程此条在 任务.md 里标注为「未完成」（仓库建筑存在但掠夺未使用保护），
// 这里按设计文档补齐。

// 仓库 1-10 级保护总量（源自设计文档）
var ezfyWareProtect = [11]int64{0, 10000, 22000, 49500, 88000, 137500, 198000, 269500, 352000, 445500, 550000}

// ezfyWareRatios 取城市 4 项资源的保护占比（未配置时默认各 25%）
func ezfyWareRatios(city *model.EzfyCity) [4]int {
	r := [4]int{city.WareFood, city.WareSteel, city.WareOil, city.WareRare}
	if r[0]+r[1]+r[2]+r[3] == 0 {
		return [4]int{25, 25, 25, 25}
	}
	return r
}

// warehouseProtect 计算该城市每项资源的保护额度 [粮, 钢, 油, 稀]
func (h *EzfyHandler) warehouseProtect(city *model.EzfyCity) [4]int64 {
	level := h.buildingLevel(city.ID, 12)
	if level < 1 {
		return [4]int64{}
	}
	if level > 10 {
		level = 10
	}
	total := ezfyWareProtect[level]
	r := ezfyWareRatios(city)
	out := [4]int64{}
	for i := 0; i < 4; i++ {
		out[i] = total * int64(r[i]) / 100
	}
	return out
}

// Warehouse GET /games/ezfy/city/warehouse —— 仓库保护信息
func (h *EzfyHandler) Warehouse(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	city := h.getOrCreateCity(uid)
	h.refreshCity(uid, &city)
	level := h.buildingLevel(city.ID, 12)
	if level > 10 {
		level = 10
	}
	total := int64(0)
	if level >= 1 {
		total = ezfyWareProtect[level]
	}
	protect := h.warehouseProtect(&city)
	r := ezfyWareRatios(&city)
	res := []gin.H{
		{"key": "food", "name": "粮食", "ratio": r[0], "protect": protect[0], "have": city.Food},
		{"key": "steel", "name": "钢铁", "ratio": r[1], "protect": protect[1], "have": city.Steel},
		{"key": "oil", "name": "石油", "ratio": r[2], "protect": protect[2], "have": city.Oil},
		{"key": "rare", "name": "稀矿", "ratio": r[3], "protect": protect[3], "have": city.Rare},
	}
	nextTotal := int64(0)
	if level < 10 {
		nextTotal = ezfyWareProtect[level+1]
	}
	resp.OK(c, gin.H{
		"level": level, "total": total, "next_total": nextTotal,
		"capacity": city.FoodCap, "res": res,
		"ratio_sum": r[0] + r[1] + r[2] + r[3],
	})
}

// WarehouseSet POST /games/ezfy/city/warehouse —— 调配 4 项资源的保护比例
func (h *EzfyHandler) WarehouseSet(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	city := h.getOrCreateCity(uid)
	var req struct {
		Food  int `json:"food"`
		Steel int `json:"steel"`
		Oil   int `json:"oil"`
		Rare  int `json:"rare"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	for _, v := range []int{req.Food, req.Steel, req.Oil, req.Rare} {
		if v < 0 || v > 100 {
			resp.ParamError(c, "每项比例需在 0-100 之间")
			return
		}
	}
	if sum := req.Food + req.Steel + req.Oil + req.Rare; sum > 100 {
		resp.ParamError(c, "四项比例合计不能超过100%(当前"+strconv.Itoa(sum)+"%)")
		return
	}
	h.DB.Model(&model.EzfyCity{}).Where("id = ?", city.ID).Updates(map[string]interface{}{
		"ware_food": req.Food, "ware_steel": req.Steel, "ware_oil": req.Oil, "ware_rare": req.Rare,
	})
	resp.OK(c, gin.H{"msg": "仓库保护比例已保存"})
}

// lootAfterWareProtect 掠夺时扣除目标城市的仓库保护额度
// 入参 have 为目标当前资源 [粮,钢,油,稀], loot 为按比例算出的拟掠夺量;
// 返回实际可掠夺量, 并回写「被保护」的说明文本。
func (h *EzfyHandler) lootAfterWareProtect(target *model.EzfyCity, have [4]int64, loot [4]int64) ([4]int64, string) {
	protect := h.warehouseProtect(target)
	out := [4]int64{}
	var protectedTotal int64
	names := [4]string{"粮", "钢", "油", "稀"}
	detail := ""
	for i := 0; i < 4; i++ {
		if protect[i] <= 0 {
			out[i] = loot[i]
			continue
		}
		// 资源在保护额度内的部分不可掠夺
		avail := have[i] - protect[i]
		if avail < 0 {
			avail = 0
		}
		if loot[i] > avail {
			protectedTotal += loot[i] - avail
			out[i] = avail
		} else {
			out[i] = loot[i]
		}
	}
	if protectedTotal > 0 {
		detail = "目标仓库保护了部分资源:"
		for i := 0; i < 4; i++ {
			if protect[i] > 0 {
				detail += " " + names[i] + "护" + strconv.FormatInt(protect[i], 10)
			}
		}
		detail += " (合计少掠" + strconv.FormatInt(protectedTotal, 10) + ")"
	}
	return out, detail
}
