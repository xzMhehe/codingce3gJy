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
	var cap int64
	for _, g := range parseGroups(order.Troops) {
		if g.Count <= 0 {
			continue
		}
		if cfg := ezfyCfg.troop(g.TroopId); cfg != nil && cfg.Carry > 0 {
			cap += int64(cfg.Carry) * int64(g.Count)
		}
	}
	return cap
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
