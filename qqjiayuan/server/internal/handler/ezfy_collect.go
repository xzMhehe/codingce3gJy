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
			[]ezfyUnitGroup{{TroopId: troopId, Count: 1}}, nil, "", 0)
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
// 对所有「驻守中(status=1)」的派遣部队立即结算一个采集周期并召回。
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
	for i := range orders {
		order := &orders[i]
		// 单程时长必须在结算前取: settleDispatch 会把 ArriveTime 推到下一个周期
		travel := ezfyOneWayTravel(order)
		// 先结算一个周期(发放资源, 可能获得宝物)
		h.settleDispatch(uid, order, now)
		// 再召回: 恢复为返航状态
		order.Status = 2
		order.Result = order.Troops
		order.ReturnTime = now + travel
		h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
			Updates(map[string]interface{}{"status": 2, "result": order.Troops, "return_time": order.ReturnTime})
		settled++
	}
	resp.OK(c, gin.H{"msg": fmt.Sprintf("已收获并召回 %d 支采集部队", settled)})
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
