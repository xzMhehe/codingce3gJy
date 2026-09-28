package handler

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
//	⚠️ cityId 用来反查玩家（科技等级存用户级 ezfy_user_tech，见 techMap）。
//	   cityId 传 0 时按「无加成」处理，方便调用方在没有城市上下文时降级。
func (h *EzfyHandler) ezfyCarryCapOf(groups []ezfyUnitGroup, cityId uint) int64 {
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
		if lv := h.techMap(cityId)[ezfyLoadTechID]; lv > 0 {
			base = base * int64(100+lv*ezfyLoadTechPct) / 100
		}
	}
	return base
}

// addCarryToOrder 把一次采集产出记进「待带回」，超出负重的部分会被丢弃
//
// 返回 (实际装入, 因负重丢弃)
func (h *EzfyHandler) addCarryToOrder(order *model.EzfyOrder, food, steel, oil, rare, gold int64) (int64, int64) {
	cur := parseCarry(order.Carry)
	capTotal := h.ezfyCarryCap(order)
	room := capTotal - cur.total()
	if room < 0 {
		room = 0
	}
	add := food + steel + oil + rare + gold
	if add <= 0 {
		return 0, 0
	}
	dropped := int64(0)
	if add > room {
		dropped = add - room
		// 按比例折算实际装入（从后往前扣，保证不超）
		scale := func(v int64) int64 {
			if add == 0 {
				return 0
			}
			return v * room / add
		}
		food, steel, oil, rare, gold = scale(food), scale(steel), scale(oil), scale(rare), scale(gold)
		add = room
	}
	cur.Food += food
	cur.Steel += steel
	cur.Oil += oil
	cur.Rare += rare
	cur.Gold += gold
	order.Carry = carryJSON(cur)
	return add, dropped
}

// ezfyCarryFull 该订单部队的「待带回」是否已装满负重。
//
// ★ 2026-09-28 用户规则：「超过负重继续采集那么就不会再采集」——
//   采到装满之后自动停下(arrive_time=0 原地待命)，不再空转累积、也不再报「资源丢弃」。
func (h *EzfyHandler) ezfyCarryFull(order *model.EzfyOrder) bool {
	capTotal := h.ezfyCarryCap(order)
	if capTotal <= 0 {
		return false
	}
	return parseCarry(order.Carry).total() >= capTotal
}

// harvestToCity 把一次采集产出**直接累加进「起点城市」**(order.CityId)。
//
// ★ 2026-09-28 用户规则（本轮最大改动）：
//
//	原来产出是「装进部队 carry 待带回，召回返航到达才入城」，造成一串用户可见的 bug：
//	  ① [一键收获] 后资源不进任何地方，玩家以为丢了；
//	  ② [一键召回] 后要等返航，中途资源既不在城也不在产出里，看着像丢了；
//	  ③ carry 超负重部分被静默丢弃（addCarryToOrder 的 dropped），到城确实少了。
//	现在改为：**收获即入城**（入的是部队出发的那座城），carry 不再参与采集结算。
//
// 入库走 ezfyResAddExpr（DB 侧原子累加 + ezfyResSafeMax 溢出兜底）。
//
// ★★ 2026-09-28 二次修复「资源没有入城市」：
//
//	ezfyResAddExpr 原带 LEAST(resMax, ...) 封顶，城市已超 21 亿时会把本次增量**吞掉**，
//	而这里仍返回 amount → 前端显示「已入库(N)」但库里不变。现已改为无条件累加；
//	「城市资源是否已满」由调用方在**发起采集前**用 ezfyAtResMax 判定并拦截。
func (h *EzfyHandler) harvestToCity(cityID int64, food, steel, oil, rare, gold int64) int64 {
	amount := food + steel + oil + rare + gold
	if amount <= 0 || cityID <= 0 {
		return 0
	}
	h.DB.Model(&model.EzfyCity{}).Where("id = ?", cityID).Updates(map[string]interface{}{
		"food":  ezfyResAddExpr("food", food),
		"steel": ezfyResAddExpr("steel", steel),
		"oil":   ezfyResAddExpr("oil", oil),
		"rare":  ezfyResAddExpr("rare", rare),
		"gold":  ezfyResAddExpr("gold", gold),
	})
	return amount
}

// ezfyCityResTotal 读一座城的资源总量(五项之和)。
//
// ★ 用途：采集改成「收获即入城」后，各接口用它做**前后差值**来报「本次入账多少」，
//   比在内存里累加更准（入库会被配置的资源最大值封顶，内存累加会虚报）。
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
