// 二战风云 玩家持有的待开宝箱（ezfy_user_chest）
//
// ★ 2026-10-14 用户需求「宝箱也能发放给玩家加个功能」：
//   购买的宝箱买了即开（ChestOpen，不进背包）；管理端/活动发放的宝箱进 ezfy_user_chest，
//   玩家在「我的宝箱」里看数量、点开箱 —— 不扣钱、不扣全库库存，
//   开箱走 ezfy_cfg_chest 同一套奖池（权重抽奖，奖品直发装备/道具背包）。
package ezfy

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// ChestBag GET /games/ezfy/chest/bag —— 我的待开宝箱列表（按宝箱汇总数量）
func (h *EzfyHandler) ChestBag(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	var rows []model.EzfyUserChest
	h.DB.Where("user_id = ? AND count > 0", uid).Find(&rows)
	list := make([]gin.H, 0, len(rows))
	total := 0
	for _, r := range rows {
		total += r.Count
		var chest model.EzfyCfgChest
		if err := h.DB.First(&chest, r.ChestId).Error; err != nil {
			continue // 宝箱配置已删：持有行不展示，但不清数据（等运营处理）
		}
		list = append(list, gin.H{
			"chest_id": r.ChestId, "name": chest.Name, "count": r.Count, "effect": chest.Effect,
			"open_max": chest.OpenMax,
		})
	}
	resp.OK(c, gin.H{"list": list, "total": total})
}

// ChestBagOpen POST /games/ezfy/chest/open-bag  {chest_id, count}
//
// 开管理端发放的待开宝箱：不扣钱/不扣库存，只扣持有数量，抽奖结果与购买宝箱同一套奖池。
// 单次开箱上限沿用宝箱配置的 OpenMax（防误点一次全开光）。
func (h *EzfyHandler) ChestBagOpen(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	city := h.getOrCreateCity(uid)
	h.calcResource(&city)
	var req struct {
		ChestId int `json:"chest_id"`
		Count   int `json:"count"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.ChestId <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	if req.Count <= 0 {
		req.Count = 1
	}
	var held model.EzfyUserChest
	if err := h.DB.Where("user_id = ? AND chest_id = ?", uid, req.ChestId).First(&held).Error; err != nil || held.Count <= 0 {
		h.fail(c, "没有待开的宝箱")
		return
	}
	var chest model.EzfyCfgChest
	if err := h.DB.First(&chest, req.ChestId).Error; err != nil {
		h.fail(c, "宝箱不存在或已下架")
		return
	}
	maxOpen := chest.OpenMax
	if maxOpen <= 0 {
		maxOpen = 1
	}
	if req.Count > maxOpen {
		h.fail(c, fmt.Sprintf("单次最多开%d个", maxOpen))
		return
	}
	if req.Count > held.Count {
		h.fail(c, fmt.Sprintf("待开宝箱数量不足(共%d个)", held.Count))
		return
	}
	pool := h.ezfyChestPool(chest.ID)
	if len(pool) == 0 {
		h.fail(c, "该宝箱还没配置奖池，请联系管理员")
		return
	}
	// 条件原子扣持有数量（并发/连点不会把同一个宝箱开两次）
	rs := h.DB.Model(&model.EzfyUserChest{}).
		Where("user_id = ? AND chest_id = ? AND count >= ?", uid, req.ChestId, req.Count).
		Update("count", gorm.Expr("count - ?", req.Count))
	if rs.Error != nil || rs.RowsAffected == 0 {
		h.fail(c, "待开宝箱数量不足, 请刷新后重试")
		return
	}
	results := []gin.H{}
	names := []string{}
	for i := 0; i < req.Count; i++ {
		prize := h.ezfyDrawChestUnlocked(uid, chest.ID, pool)
		if prize == nil {
			continue
		}
		desc := h.ezfyGrantChestPrize(&city, prize)
		if desc == "" {
			continue
		}
		names = append(names, desc)
		results = append(results, gin.H{"name": desc})
	}
	// ★ 2026-10-14 与 ChestOpen 同口径：同名奖品合并成「名×数量」，开一堆不再刷长串
	agg, seen := make([]string, 0, len(names)), map[string]int{}
	for _, nm := range names {
		seen[nm]++
	}
	for _, nm := range names {
		if seen[nm] > 1 {
			agg = append(agg, fmt.Sprintf("%s×%d", nm, seen[nm]))
		} else {
			agg = append(agg, nm)
		}
		seen[nm] = 0
	}
	if len(agg) == 0 {
		agg = []string{"空箱"}
	}
	resp.OK(c, gin.H{
		"msg":     fmt.Sprintf("开箱成功: 获得 %s", strings.Join(agg, "、")),
		"results": results,
	})
}