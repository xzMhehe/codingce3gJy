// 二战风云 玩家持有的待开宝箱（ezfy_user_chest）
//
// ★ 2026-10-14 用户需求「宝箱也能发放给玩家加个功能」：
//   商城买的宝箱可选即开（ChestOpen）或只买不开进背包（ChestBuy）；
//   管理端/活动发放的宝箱进 ezfy_user_chest，玩家在「背包→宝箱」里看数量、点开箱
//   —— 开箱不扣钱、不扣全库库存，走 ezfy_cfg_chest 同一套奖池（权重抽奖，奖品直发装备/道具背包）。
package ezfy

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// ChestBuy POST /games/ezfy/chest/buy  {chest_id, count, currency: gold|diamond}
//
// ★ 2026-10-14 用户需求「宝箱也能购买不用直接开」：只扣钱、扣库存，宝箱进 ezfy_user_chest
// （玩家背包 → 宝箱 tab），不开箱、不发奖励；玩家可去背包开，也可以后在商城宝箱页直接开。
// 与 ChestOpen 的区别：ChestOpen 买了即开（抽奖发奖），ChestBuy 只买不抽。
func (h *EzfyHandler) ChestBuy(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	city := h.getOrCreateCity(uid)
	h.calcResource(&city)
	var req struct {
		ChestId  int    `json:"chest_id"`
		Count    int    `json:"count"`
		Currency string `json:"currency"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.ChestId <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	if req.Count <= 0 {
		req.Count = 1
	}
	var chest model.EzfyCfgChest
	if err := h.DB.First(&chest, req.ChestId).Error; err != nil {
		h.fail(c, "宝箱不存在或已下架")
		return
	}
	maxBuy := chest.OpenMax
	if maxBuy <= 0 {
		maxBuy = 1
	}
	if req.Count > maxBuy {
		h.fail(c, fmt.Sprintf("单次最多购买%d个", maxBuy))
		return
	}
	if chest.Stock == 0 {
		h.fail(c, "该宝箱已售罄")
		return
	}
	if chest.Stock > 0 && req.Count > chest.Stock {
		h.fail(c, fmt.Sprintf("库存不足(剩余%d个)", chest.Stock))
		return
	}
	useDiamond := req.Currency == "diamond"
	price := chest.PriceGold
	unit := "黄金"
	if useDiamond {
		price = chest.PriceDiamond
		unit = "钻石"
	}
	if price <= 0 {
		h.fail(c, "该宝箱不支持用"+unit+"购买")
		return
	}
	total := price * int64(req.Count)
	var prof model.EzfyProfile
	if useDiamond {
		prof = h.ensureProfile(uid)
		// 并发卡控：条件原子扣（连点不会「扣一次钱买多个箱」）
		if !h.ezfySpendDiamond(prof.ID, total) {
			h.fail(c, fmt.Sprintf("钻石不足(需要%d钻石, 当前%d)", total, prof.Diamond))
			return
		}
		h.logDiamond(uid, -total, "购买宝箱(入背包): "+chest.Name)
	} else {
		if !h.ezfySpendGold(city.ID, total) {
			h.fail(c, fmt.Sprintf("黄金不足(需要%d黄金, 当前%d)", total, city.Gold))
			return
		}
		city.Gold -= total
	}
	// 库存条件原子扣；失败退回刚扣的钱
	if chest.Stock > 0 && !h.ezfySpendStock(&model.EzfyCfgChest{}, chest.ID, req.Count) {
		if useDiamond {
			h.ezfyRefundDiamond(prof.ID, total)
			h.logDiamond(uid, total, "购买宝箱回滚(库存不足): "+chest.Name)
		} else {
			h.ezfyRefundGold(city.ID, total)
			city.Gold += total
		}
		h.fail(c, fmt.Sprintf("库存不足(剩余%d个)", chest.Stock))
		return
	}
	// 宝箱入背包（ezfy_user_chest，按 user_id+chest_id 累加；不进道具背包，是独立「待开宝箱」）
	h.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "chest_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{"count": gorm.Expr("count + ?", req.Count), "updated_at": time.Now()}),
	}).Create(&model.EzfyUserChest{UserID: uid, ChestId: chest.ID, Count: req.Count})
	if chest.Stock > 0 {
		h.cfgsReload()
	}
	ezfyChestCacheDel(uid)
	resp.OK(c, gin.H{
		"msg":     fmt.Sprintf("购买成功: %d个%s已放入背包, 可去背包开启", req.Count, chest.Name),
		"gold":    city.Gold,
		"diamond": h.ensureProfile(uid).Diamond,
	})
}

// ChestBag GET /games/ezfy/chest/bag —— 我的待开宝箱列表（按宝箱汇总数量）
func (h *EzfyHandler) ChestBag(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	var rows []model.EzfyUserChest
	h.DB.Where("user_id = ? AND count > 0", uid).Find(&rows)
	var chestsMap = map[int]model.EzfyCfgChest{}
	var all []model.EzfyCfgChest
	h.DB.Find(&all)
	for _, ch := range all {
		chestsMap[ch.ID] = ch
	}
	list := make([]gin.H, 0, len(rows))
	total := 0
	for _, r := range rows {
		total += r.Count
		chest, ok := chestsMap[r.ChestId]
		if !ok {
			continue // 宝箱配置已删：持有行不展示，但不清数据（等运营处理）
		}
		// ★ 2026-10-14 带完整奖池（供背包内 [说明] 展开看能开出啥）+ 价格说明
		pool := []gin.H{}
		for _, p := range ezfyCfg.chestPool(chest.ID) {
			name, quality := "", p.Quality
			switch p.Kind {
			case 1:
				if cfg := ezfyCfg.equipment(p.RefId); cfg != nil {
					name = cfg.Name
					if quality == "" {
						quality = ezfyTierName(cfg.Tier)
					}
				}
			case 2:
				if cfg := ezfyCfg.item(p.RefId); cfg != nil {
					name = cfg.Name
				}
			case 3: // 整套（RefId = 套装 id）
				if s := ezfyCfg.equipSet(p.RefId); s != nil {
					name = s.Name + " 整套"
				}
			}
			if name == "" {
				continue
			}
			pool = append(pool, gin.H{
				"kind": p.Kind, "ref_id": p.RefId, "name": name,
				"count": p.Count, "weight": p.Weight, "quality": quality,
				"detail": h.ezfyChestPrizeDetail(&p), "min_draws": p.MinDraws,
			})
		}
		list = append(list, gin.H{
			"chest_id": r.ChestId, "name": chest.Name, "count": r.Count, "effect": chest.Effect,
			"open_max": chest.OpenMax,
			"price_gold": chest.PriceGold, "price_diamond": chest.PriceDiamond,
			"des": chest.Des, "pool": pool,
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
	// ★ 2026-10-14 去掉 OpenMax 单次上限：开的是自己背包里的待开宝箱，支持一次开「全部」（
	//   OpenMax 只约束商城购买/即开），仅校验持有数量，防超开。
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
	// ★ 2026-10-14 卡顿优化：批量抽奖（一次查总次数+一次累计写库，不再逐开逐查逐写）
	prizes := h.ezfyDrawChestsUnlocked(uid, chest.ID, pool, req.Count)
	results := []gin.H{}
	names := []string{}
	for _, prize := range prizes {
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
	// ★★ 2026-10-16 开箱流水：背包待开宝箱开箱也记一条（管理端「数据管理→宝箱开启」），
	//   与商城购买即开(ChestOpen)同口径，方便核实「发放的宝箱开出了什么」。
	if len(names) > 0 {
		result := "获得：" + strings.Join(agg, "、")
		if rs := []rune(result); len(rs) > 255 {
			result = string(rs[:255])
		}
		h.DB.Create(&model.EzfyChestLog{
			UserID: uid, ChestId: chest.ID, ChestName: chest.Name,
			Count: req.Count, Result: result,
		})
	}
	resp.OK(c, gin.H{
		"msg":     fmt.Sprintf("开箱成功: 获得 %s", strings.Join(agg, "、")),
		"results": results,
	})
}