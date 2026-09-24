package handler

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 二战风云 一键采集 / 一键收获
// 复刻 `二战风云/templates/report/index.html` 军队动态区的两个按钮
// （原版 /home/city_troop/begincollectall.html 与 stopcollectall.html）

// CollectAll POST /games/ezfy/wild/collect-all —— 一键采集
//
// 对本城所有空闲的已占领野地下达「派遣驻守采集」命令(命令类型 7)：
// 每块野地派 1 支采集队 + 1 名空闲军官，部队常驻野地每 8 小时结算一次产出，
// 可随时用「一键收获 / 一键召回」处理。
//
// ★ 修复记录：
//   - 原来把命令类型写成 4(一次性采集)、且 officer 传空串，
//     而 createOrder 对 4/7 都要求「必须携带军官」→ 一键采集**永远失败**；
//   - 现在自动挑选空闲军官带队（一名军官同时只能带一支部队），
//     并覆盖 4/7 两类命令，与「一键收获 / 一键召回」口径一致。
func (h *EzfyHandler) CollectAll(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	var req struct {
		CityId int64 `json:"city_id"`
	}
	_ = c.ShouldBindJSON(&req)
	city := h.bodyCity(uid, req.CityId)
	h.refreshCity(uid, city)

	var wls []model.EzfyWildland
	h.DB.Where("city_id = ? AND status = 0", city.ID).Order("id ASC").Find(&wls)
	if len(wls) == 0 {
		resp.ParamError(c, "没有可采集的野地(先在「附属野地」占领野地)")
		return
	}

	// 采集队: 城内最弱的一个非城防兵种（负重决定能装多少，所以按野地产量配数量）
	troopId, weakest, carryPer := 0, 0, 0
	tm := h.troopMap(city.ID)
	for id, cnt := range tm {
		if cnt <= 0 {
			continue
		}
		cfg := ezfyCfg.troop(id)
		if cfg == nil || cfg.Type == 4 {
			continue
		}
		if troopId == 0 || cfg.AtkGround < weakest {
			troopId, weakest, carryPer = id, cfg.AtkGround, cfg.Carry
		}
	}
	if troopId == 0 {
		resp.ParamError(c, "城内没有可派出的部队")
		return
	}
	if carryPer <= 0 {
		carryPer = 1
	}

	// 空闲军官池（不在出征中、不是俘虏、没有带未结束的命令）
	idleOfficers := []model.EzfyOfficer{}
	for _, o := range h.officerList(city.ID) {
		if o.Status == 1 || o.IsCaptive == 1 {
			continue
		}
		if o.Position != 0 {
			continue // 市长/城守有城务在身，不能带队出征采集
		}
		if h.officerBusyOrder(city.ID, o.Name) {
			continue
		}
		idleOfficers = append(idleOfficers, o)
	}
	if len(idleOfficers) == 0 {
		resp.ParamError(c, "采集必须由军官带队，城内没有空闲军官(可在军校招募或先召回出征部队)")
		return
	}

	ok, fail := 0, []string{}
	oi := 0
	for _, wl := range wls {
		if oi >= len(idleOfficers) {
			fail = append(fail, "空闲军官不足")
			break
		}
		// 一块野地一次产出 ≈ 等级 × 800 × 4(陆地) / ×3(海野)，按此配够负重的兵
		need := int64(wl.Level) * 800 * 4
		if wl.WildType == 2 {
			need = int64(wl.Level) * 800 * 3
		}
		want := (need + int64(carryPer) - 1) / int64(carryPer)
		if want < 1 {
			want = 1
		}
		avail := h.troopMap(city.ID)[troopId]
		if avail <= 0 {
			fail = append(fail, "部队不足")
			break
		}
		if want > avail {
			want = avail
		}
		msg := h.createOrder(uid, city, 7, wl.X, wl.Y, 1, int64(wl.ID),
			[]ezfyUnitGroup{{TroopId: troopId, Count: want}}, nil, idleOfficers[oi].Name, 0, 0)
		if msg != "" {
			fail = append(fail, msg)
			continue
		}
		oi++
		ok++
	}
	if ok == 0 {
		resp.ParamError(c, "一键采集失败: "+strings.Join(dedupStrings(fail), "; "))
		return
	}
	msg := fmt.Sprintf("已对 %d 块野地下达驻守采集命令(每块 1 名军官带队)", ok)
	if len(fail) > 0 {
		msg += fmt.Sprintf("(%d 块跳过: %s)", len(fail), strings.Join(dedupStrings(fail), "; "))
	}
	resp.OK(c, gin.H{"msg": msg})
}

// HarvestAll POST /games/ezfy/wild/harvest-all —— 一键收获
//
// ★ 用户规则：「收获就是收获」—— 只把采集产出结算进部队的「待带回」池，
//   **不召回**。资源只有「召回并返航到达」才会入城（见 finishReturn）。
func (h *EzfyHandler) HarvestAll(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	h.processOrders(uid)
	now := time.Now().UnixMilli()

	var orders []model.EzfyOrder
	h.DB.Where("user_id = ? AND order_type = 7 AND status = 1", uid).Order("id ASC").Find(&orders)
	if len(orders) == 0 {
		resp.ParamError(c, "没有正在采集的部队")
		return
	}

	settled := 0
	skipped := 0
	var loadedTotal int64
	for i := range orders {
		order := &orders[i]
		before := parseCarry(order.Carry).total()
		if periods, ok := h.settleDispatch(uid, order, now); ok {
			loadedTotal += parseCarry(order.Carry).total() - before
			settled += periods
		} else {
			skipped++ // 未满一个采集周期或野地已丢失(已自动返航)
		}
	}
	msg := fmt.Sprintf("已收获 %d 支采集部队，共装入待带回资源 %d（资源需「召回」才会运回城里）",
		settled, loadedTotal)
	if skipped > 0 {
		msg += fmt.Sprintf("；%d 支驻守尚不满一个采集周期（可[一键召回]按驻守时长折算资源，无宝物）", skipped)
	}
	resp.OK(c, gin.H{"msg": msg})
}

// RecallAll POST /games/ezfy/wild/recall-all —— 一键召回
//
// 把所有驻守中的采集部队改成返航；**返航到达时**才会把「待带回资源」运回城里。
func (h *EzfyHandler) RecallAll(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	h.processOrders(uid)
	now := time.Now().UnixMilli()

	var orders []model.EzfyOrder
	h.DB.Where("user_id = ? AND order_type = 7 AND status = 1", uid).Order("id ASC").Find(&orders)
	if len(orders) == 0 {
		resp.ParamError(c, "没有正在采集的部队")
		return
	}
	n := 0
	var back int64
	for i := range orders {
		order := &orders[i]
		// 召回前先结算产出: 满一个采集周期给资源+宝物, 提前召回只有按比例的资源(无宝物)
		h.settleDispatchOnRecall(uid, order, now)
		if order.Status != 1 {
			continue // 野地已丢失, settleDispatch 已把部队自动改成返航
		}
		travel := ezfyOneWayTravel(order)
		back += parseCarry(order.Carry).total()
		h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
			Updates(map[string]interface{}{"status": 2, "result": order.Troops,
				"return_time": now + travel})
		n++
	}
	resp.OK(c, gin.H{"msg": fmt.Sprintf("已召回 %d 支采集部队，共带回资源 %d（到达后入库）", n, back)})
}

// dedupStrings 去重(保持顺序), 用于把重复的失败原因合并
func dedupStrings(list []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, s := range list {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}
