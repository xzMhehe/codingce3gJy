package handler

import (
	"fmt"
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
// ★ 2026-09-24 用户规则: 采集部队到达野地后驻守**空闲**, 需手工点[采集]才开始。
//   「一键采集」= 对本城所有**空闲驻军**(status=1, arrive_time=0)批量下达采集命令;
//   派新部队到未驻守的野地走「附属野地 → [采集]」。
func (h *EzfyHandler) CollectAll(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	h.processOrders(uid)
	now := time.Now().UnixMilli()

	var orders []model.EzfyOrder
	h.DB.Where("user_id = ? AND status = 1 AND order_type = 7 AND arrive_time = 0", uid).
		Order("id ASC").Find(&orders)
	if len(orders) == 0 {
		resp.ParamError(c, "没有空闲的驻军部队(派采集队请到「附属野地 → [采集]」; 已开始采集的部队等待结算即可)")
		return
	}
	ok, fail := 0, 0
	for i := range orders {
		order := &orders[i]
		var wl model.EzfyWildland
		if err := h.DB.First(&wl, order.TargetId).Error; err != nil || wl.CityId != order.CityId {
			h.beginReturn(order, now, 0)
			fail++
			continue
		}
		order.ArriveTime = now + ezfyDispatchPeriod()
		h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
			Update("arrive_time", order.ArriveTime)
		ok++
	}
	msg := fmt.Sprintf("已对 %d 支空闲驻军下达采集命令(每满一个采集周期结算一期)", ok)
	if fail > 0 {
		msg += fmt.Sprintf("(%d 支野地已丢失, 部队自动返航)", fail)
	}
	resp.OK(c, gin.H{"msg": msg})
}

// StartCollect POST /games/ezfy/wild/start-collect —— 单支空闲驻军开始采集
//
// ★ 2026-09-24 用户规则: 驻军没采集就是「空闲」状态, 手工点[采集]才进入采集状态。
//   驻军趋(情报→驻军)/野地列表/出征队列上的 [采集] 都走本接口。
func (h *EzfyHandler) StartCollect(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	var req struct {
		OrderId int64 `json:"order_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.OrderId <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	h.processOrders(uid)
	now := time.Now().UnixMilli()
	var order model.EzfyOrder
	if err := h.DB.Where("id = ? AND user_id = ?", req.OrderId, uid).First(&order).Error; err != nil {
		resp.ParamError(c, "命令不存在")
		return
	}
	if order.Status != 1 || order.OrderType != 7 {
		resp.ParamError(c, "该部队不是驻守采集部队")
		return
	}
	if order.ArriveTime > 0 {
		resp.ParamError(c, "该部队已在采集中")
		return
	}
	var wl model.EzfyWildland
	if err := h.DB.First(&wl, order.TargetId).Error; err != nil || wl.CityId != order.CityId {
		resp.ParamError(c, "采集野地已丢失")
		return
	}
	next := now + ezfyDispatchPeriod()
	h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
		Updates(map[string]interface{}{"arrive_time": next})
	h.addReport(uid, 5, "采集报告: 开始采集",
		fmt.Sprintf("驻守在野地%d级(%d,%d)的部队开始采集, 每满一个采集周期结算一期: 资源装进部队待召回(军官后勤每点+1%%), 宝物直接进背包(每期至少1件)。",
			wl.Level, wl.X, wl.Y), "", order.ID)
	resp.OK(c, gin.H{"msg": fmt.Sprintf("采集开始, 一个采集周期后首次结算(野地%d级 %d,%d)",
		wl.Level, wl.X, wl.Y)})
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
