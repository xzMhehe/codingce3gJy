package ezfy

import (
	"encoding/json"

	"qqjiayuan/server/internal/model"
)

// 二战风云 —— 采集「待带回资源」的存取
//
// ★ 规则（用户明确要求）：
//   - 采集到的资源先记在**部队身上**，只有**召回并返航到达**才入城；
//   - 容量上限 = 部队各兵种的 carry 之和（所以采集要带卡车/运输兵）；
//   - 「一键收获」只结算一次产出（收进待带回池），**不召回**；
//   - 宝物不受负重限制，直接进背包。

// ezfyCarry 待带回资源
type ezfyCarry struct {
	Food  int64 `json:"food"`
	Steel int64 `json:"steel"`
	Oil   int64 `json:"oil"`
	Rare  int64 `json:"rare"`
	Gold  int64 `json:"gold"`
}

func (c ezfyCarry) total() int64 { return c.Food + c.Steel + c.Oil + c.Rare + c.Gold }

// ezfyCarryFit 按「资源最大值」（二战系统配置，不写死）计算一次入城能装多少、超出多少。
//
// ★★ 2026-10-17 用户口径（派遣/运输/返航带回的拉资源）：
//
//	目标城/出发城按配置上限封顶累加，装不下的部分原路带回或系统回收。
//	返回 (可入库量, 超出量)。
func (h *EzfyHandler) ezfyCarryFit(cityID uint, c ezfyCarry) (fit, excess ezfyCarry) {
	if cityID <= 0 {
		return c, ezfyCarry{}
	}
	var city model.EzfyCity
	if err := h.DB.Select("food", "steel", "oil", "rare", "gold").
		First(&city, cityID).Error; err != nil {
		return c, ezfyCarry{}
	}
	fitRes := func(cur, inc int64, res string) (int64, int64) {
		if inc <= 0 {
			return 0, 0
		}
		max := ezfyResMaxOf(res) // 二战系统配置的「资源最大值」，不写死
		room := max - cur
		if room < 0 {
			room = 0
		}
		if inc <= room {
			return inc, 0
		}
		return room, inc - room
	}
	fit.Food, excess.Food = fitRes(city.Food, c.Food, "food")
	fit.Steel, excess.Steel = fitRes(city.Steel, c.Steel, "steel")
	fit.Oil, excess.Oil = fitRes(city.Oil, c.Oil, "oil")
	fit.Rare, excess.Rare = fitRes(city.Rare, c.Rare, "rare")
	fit.Gold, excess.Gold = fitRes(city.Gold, c.Gold, "gold")
	return fit, excess
}

// ezfyCarryFitTo 计算「携带资源入目标城」：按配置上限封顶，返回可入库量与超出量。
// 超出量由调用方原路带回出发城（记进 order.Carry / Resources）。
func (h *EzfyHandler) ezfyCarryFitTo(city *model.EzfyCity, c ezfyCarry) (fit, excess ezfyCarry) {
	fitRes := func(cur, inc int64, res string) (int64, int64) {
		if inc <= 0 {
			return 0, 0
		}
		max := ezfyResMaxOf(res) // 二战系统配置的「资源最大值」，不写死
		room := max - cur
		if room < 0 {
			room = 0
		}
		if inc <= room {
			return inc, 0
		}
		return room, inc - room
	}
	fit.Food, excess.Food = fitRes(city.Food, c.Food, "food")
	fit.Steel, excess.Steel = fitRes(city.Steel, c.Steel, "steel")
	fit.Oil, excess.Oil = fitRes(city.Oil, c.Oil, "oil")
	fit.Rare, excess.Rare = fitRes(city.Rare, c.Rare, "rare")
	fit.Gold, excess.Gold = fitRes(city.Gold, c.Gold, "gold")
	return fit, excess
}

// parseCarry 读订单上的待带回资源
func parseCarry(raw string) ezfyCarry {
	var c ezfyCarry
	if raw == "" {
		return c
	}
	_ = json.Unmarshal([]byte(raw), &c)
	return c
}

// carryJSON 序列化
func carryJSON(c ezfyCarry) string {
	b, _ := json.Marshal(c)
	return string(b)
}

// ezfyCarryCap 该订单部队的总负重上限
func (h *EzfyHandler) ezfyCarryCap(order *model.EzfyOrder) int64 {
	return h.ezfyCarryCapOf(parseGroups(order.Troops), uint(order.CityId))
}

// ezfyCarryCapOf 一组部队的总负重上限（= Σ 兵种 carry × 数量，再乘装载技术加成）
//
// ★★ 2026-09-28 修复「装载技术没实际作用」：
//
//	科技 13「装载技术 部队负重+2%」原来只写在 ezfy_cfg_tech.effect 里，
//	**后端从来没有读过这个 tech_id** → 玩家把它研到 10 级，负重一点不涨。
//	现在在这里统一加成（这是「负重上限」的唯一收敛点，采集/运输/出征全走它）：
//	    负重上限 = Σ(carry × 数量) × (100 + 装载技术等级 × 2) / 100
//
// ⚠️ cityId 用来反查玩家（科技等级存用户级 ezfy_user_tech，见 techMap）。
//
//	cityId 传 0 时按「无加成」处理，方便调用方在没有城市上下文时降级。
//
// ★ 2026-10-05 techs 可选：调用方已取好的科技等级 map（如 /order/preview 并行块）时传入，
//
//	避免这里再查一次 techMap（2 条跨 WAN 查询）。
func (h *EzfyHandler) ezfyCarryCapOf(groups []ezfyUnitGroup, cityId uint, techs ...map[int]int) int64 {
	var base int64
	for _, g := range groups {
		if g.Count <= 0 {
			continue
		}
		if cfg := ezfyCfg.troop(g.TroopId); cfg != nil && cfg.Carry > 0 {
			base += int64(cfg.Carry) * int64(g.Count)
		}
	}
	if base > 0 && cityId > 0 {
		lv := 0
		if len(techs) > 0 && techs[0] != nil {
			lv = techs[0][ezfyLoadTechID]
		} else {
			lv = h.techMap(cityId)[ezfyLoadTechID]
		}
		if lv > 0 {
			base = base * int64(100+lv*ezfyLoadTechPct) / 100
		}
	}
	return base
}

// addCarryToOrder 把一次采集产出记进部队「待带回(carry)」。
//
// ★★ 2026-10-21 用户新规则（用户原话「装不下需要用户点击 停止/一键收获 后才能进入原城市」）：
//
//	产出（**含超出负重的部分**）不再即时入城、也**不再丢弃** —— 全部存进 carry
//	（可超出负重上限，上限只用于「满/未满」展示，不截断存储）。等玩家点
//	[停止采集]/[一键收获]/[召回] 时，才由 harvestCarryToCity 一次性进入
//	部队**原出发城市**（旧「封顶丢弃」「超重直接入城」两条逻辑一并废除）。
//
// 返回存入后的 carry 总量；本次产出为 0 时不写库（避免无谓 UPDATE）。
func (h *EzfyHandler) addCarryToOrder(order *model.EzfyOrder, food, steel, oil, rare, gold int64) int64 {
	cur := parseCarry(order.Carry)
	if food+steel+oil+rare+gold <= 0 {
		return cur.total()
	}
	cur.Food += food
	cur.Steel += steel
	cur.Oil += oil
	cur.Rare += rare
	cur.Gold += gold
	order.Carry = carryJSON(cur)
	return cur.total()
}

// ezfyCarryFull 该订单部队的「待带回」是否已达负重上限。
//
// ★ 2026-09-28 用户规则：采集产出累积进 carry，部队**不自动停止**、持续采集。
// ★★ 2026-10-21 口径修正：满了之后产出**照常累加**（不丢弃、不即时入城），
//
//	超出上限的部分等玩家点 [停止采集]/[一键收获] 后一并进入原出发城市。
//	上限只作为「满/未满」展示与 [采集] 发起点的一个提示，不做截断。
func (h *EzfyHandler) ezfyCarryFull(order *model.EzfyOrder) bool {
	capTotal := h.ezfyCarryCap(order)
	if capTotal <= 0 {
		return false
	}
	return parseCarry(order.Carry).total() >= capTotal
}

// harvestCarryToCity 把部队「待带回负重(carry)」累加进起点城市，并清空负重。
//
// ★ 2026-09-28 用户规则「负重封顶+无自动停止」：采集产出累积进负重，由玩家手动
//
//	[停止采集]/[一键收获]/[召回]取回。本函数负责把取回的负重入城（走 harvestToCity，
//	前后无差值——carry 就是本次真正到账的量），同时把订单负重清零。
//	返回实际入城总量。
func (h *EzfyHandler) harvestCarryToCity(order *model.EzfyOrder) int64 {
	c := parseCarry(order.Carry)
	if c.total() <= 0 {
		return 0
	}
	got := h.harvestToCity(int64(order.CityId), c.Food, c.Steel, c.Oil, c.Rare, c.Gold)
	if got > 0 {
		order.Carry = carryJSON(ezfyCarry{})
		h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
			Updates(map[string]interface{}{"carry": ""})
	}
	return got
}

// harvestToCity 把一批资源**直接累加进「起点城市」**(order.CityId)。
//
// ★★ 2026-10-21 当前模型（用户规则「装不下等点停止/收获才进原城市」）：
//
//	采集产出一律先累积进部队「待带回(carry)」（可超负重上限、不丢弃、不即时入城）；
//	玩家点 [停止采集]/[一键收获] 时由 harvestCarryToCity 把整包 carry 经本函数
//	一次性进入部队**原出发城市**；[召回] 则随返航到达时入城（finishReturn 同口径）。
//
// 入库走 ezfyResAddExpr（DB 侧原子累加 + 资源最大值 21 亿硬上限）。
//
// ★★ 2026-09-28 修复「资源没有入城市」：
//
//	ezfyResAddExpr 原带 LEAST(resMax, ...) 封顶，城市已超 21 亿时会把本次增量**吞掉**，
//	而这里仍返回 amount → 前端显示「已入库(N)」但库里不变。
//	现在恢复「资源最大值唯一硬上限」，并且本函数**返回实际入账数**（前后差值），
//	既封顶不超上限、又保证界面报的数字=库里真的加的量。
func (h *EzfyHandler) harvestToCity(cityID int64, food, steel, oil, rare, gold int64) int64 {
	amount := food + steel + oil + rare + gold
	if amount <= 0 || cityID <= 0 {
		return 0
	}
	before := h.ezfyCityResTotal(cityID)
	h.DB.Model(&model.EzfyCity{}).Where("id = ?", cityID).Updates(map[string]interface{}{
		"food":  ezfyResAddExpr("food", food),
		"steel": ezfyResAddExpr("steel", steel),
		"oil":   ezfyResAddExpr("oil", oil),
		"rare":  ezfyResAddExpr("rare", rare),
		"gold":  ezfyResAddExpr("gold", gold),
	})
	after := h.ezfyCityResTotal(cityID)
	got := after - before
	if got < 0 {
		got = 0
	}
	return got
}

// ezfyCityResTotal 读一座城的资源总量(五项之和)。
//
// ★ 用途：采集改成「收获即入城」后，各接口用它做**前后差值**来报「本次入账多少」，
//
//	比在内存里累加更准（入库会被配置的资源最大值封顶，内存累加会虚报）。
func (h *EzfyHandler) ezfyCityResTotal(cityID int64) int64 {
	if cityID <= 0 {
		return 0
	}
	var c model.EzfyCity
	if err := h.DB.Select("food", "steel", "oil", "rare", "gold").
		First(&c, cityID).Error; err != nil {
		return 0
	}
	return c.Food + c.Steel + c.Oil + c.Rare + c.Gold
}
