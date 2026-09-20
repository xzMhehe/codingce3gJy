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
// 对本城所有空闲的已占领野地下达采集命令；每块野地派 1 个城内最弱的非城防兵种作采集队。
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

	// 采集队: 城内最弱的一个非城防兵种
	troopId, weakest := 0, 0
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
			troopId, weakest = id, cfg.AtkGround
		}
	}
	if troopId == 0 {
		resp.ParamError(c, "城内没有可派出的部队")
		return
	}

	ok, fail := 0, []string{}
	for _, wl := range wls {
		// 每个采集队只需 1 个兵, 兵不够就停
		if h.troopMap(city.ID)[troopId] <= 0 {
			fail = append(fail, "部队不足")
			break
		}
		msg := h.createOrder(uid, city, 4, wl.X, wl.Y, 1, int64(wl.ID),
			[]ezfyUnitGroup{{TroopId: troopId, Count: 1}}, nil, "", 0, 0)
		if msg != "" {
			fail = append(fail, msg)
			continue
		}
		ok++
	}
	if ok == 0 {
		resp.ParamError(c, "一键采集失败: "+strings.Join(dedupStrings(fail), "; "))
		return
	}
	msg := fmt.Sprintf("已对 %d 块野地下达采集命令", ok)
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
	var loadedTotal int64
	for i := range orders {
		order := &orders[i]
		before := parseCarry(order.Carry).total()
		// 结算一个周期：产出记进「待带回」，宝物直接进背包（不召回）
		h.settleDispatch(uid, order, now)
		// settleDispatch 只改了内存里的 order.Carry，这里落库
		h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
			Updates(map[string]interface{}{"carry": order.Carry, "arrive_time": order.ArriveTime,
				"result": order.Result})
		loadedTotal += parseCarry(order.Carry).total() - before
		settled++
	}
	resp.OK(c, gin.H{"msg": fmt.Sprintf("已收获 %d 支采集部队，共装入待带回资源 %d（资源需「召回」才会运回城里）",
		settled, loadedTotal)})
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
