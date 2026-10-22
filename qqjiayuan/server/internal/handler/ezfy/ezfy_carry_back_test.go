package ezfy

import (
	"strings"
	"testing"
)

// TestReturnBringsBackCarriedResources —— 防回归：**返航必须把「随军资源」带回来**。
//
// ★★ 2026-10-11 线上反馈「派遣携带黄金，取消/返航后黄金不见了」：
//
//	随军资源(order.Resources)在出发时已从出发城扣掉，此后只有两种归宿：
//	  ① 送达目标城（运输/派遣/增援**到达**时入目标城 + 清空 resources）；
//	  ② 随部队返航入城（finishReturn 并入 Carry）。
//	原来只有「召回」一条路径做了 ②，其余返航（异常返航/侦查/战斗/放弃野地/遣返驻军…）
//	都只带 Carry → Resources 凭空消失。现在统一收敛到 finishReturn 一处。
//
// 本断言锁死这个不变式，避免以后有人在 finishReturn 里把这段删掉。
func TestReturnBringsBackCarriedResources(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) finishReturn(")
	if !strings.Contains(body, "parseCarry(order.Resources)") {
		t.Fatalf("finishReturn 未把「随军资源」(order.Resources) 并入 Carry —— "+
			"取消/异常返航时玩家携带的资源会凭空消失：\n%s", body)
	}
	// ★ 2026-10-22 入城逻辑收敛到 ezfyDeliverCarryToCity（先结算、再算 fit、事务入库）。
	//   这里断言 finishReturn 仍走统一入口，并到 helper 里确认黄金确实会被累加。
	if !strings.Contains(body, "h.ezfyDeliverCarryToCity(city, c)") {
		t.Fatalf("finishReturn 入城没有走 ezfyDeliverCarryToCity（携带的黄金可能丢）：\n%s", body)
	}
	helper := ezfyFuncBody(t, "ezfy_carry.go", "func (h *EzfyHandler) ezfyDeliverCarryToCity(")
	if !strings.Contains(helper, `ezfyResAddExpr("gold", fit.Gold)`) {
		t.Fatalf("ezfyDeliverCarryToCity 入库时没有累加黄金（携带的黄金会丢）：\n%s", helper)
	}
}

// TestTransportAndDispatchUseCarryDelivery —— 运输(5)/派遣(8)到达都必须走统一入库入口。
//
// 防回归：如果以后有人在到达分支里直接 `UPDATE ... ezfyResAddExpr`，就会绕开
// 「先懒结算目标城，再按结算后的容量算 fit」这一步，待落库产量又会把运来的资源吞掉。
func TestTransportAndDispatchUseCarryDelivery(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) processArrive(")
	if n := strings.Count(body, "h.ezfyDeliverCarryToCity("); n < 2 {
		t.Fatalf("运输/派遣到达没有都走 ezfyDeliverCarryToCity（找到 %d 处）—— "+
			"绕开统一入口会导致待结算产量占满容量后资源被 SQL 封顶吞掉：\n%s", n, body)
	}
}

// TestCarryDeliverySettlesTargetBeforeFit —— 结算顺序的根因回归测试。
//
// ★★ 2026-10-22 线上「A 城黄金拉到 B 城后消失」：
//
//	B 城除黄金外都满了，黄金还剩一点容量；运输到达时，B 城在途产量/工资等
//	待懒结算。若先按旧快照算 fit、再结算，待落库的黄金产量会把剩余容量填满，
//	最后 SQL 封顶就会吞掉运来的黄金，而旧的 fit 又没把超出量记进 Carry。
//
// 因此 helper 里 `calcResource` 必须出现在 `ezfyCarryFitAt` 之前。
func TestCarryDeliverySettlesTargetBeforeFit(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_carry.go", "func (h *EzfyHandler) ezfyDeliverCarryToCity(")
	ci := strings.Index(body, "h.calcResource(city")
	fi := strings.Index(body, "ezfyCarryFitAt")
	if ci < 0 || fi < 0 {
		t.Fatalf("ezfyDeliverCarryToCity 缺少懒结算或 fit：\n%s", body)
	}
	if ci > fi {
		t.Fatalf("ezfyDeliverCarryToCity 先算 fit 再结算 —— "+
			"待落库产量会在结算后占满容量，运输/派遣资源会被 SQL 封顶吞掉：\n%s", body)
	}
	for _, want := range []string{
		"h.DB.Transaction",
		`clause.Locking{Strength: "UPDATE"}`,
		`ezfyResAddExpr("gold", fit.Gold)`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("ezfyDeliverCarryToCity 缺少 %q：\n%s", want, body)
		}
	}
}

// TestCarryFitGoldOnlyWhenOtherResourcesFull —— 纯 fit 逻辑回归：
// 其它四项已到上限时，黄金仍应按自己的剩余容量正常入库，不能被整包丢掉。
func TestCarryFitGoldOnlyWhenOtherResourcesFull(t *testing.T) {
	cur := ezfyCarry{
		Food:  ezfyResMaxOf("food"),
		Steel: ezfyResMaxOf("steel"),
		Oil:   ezfyResMaxOf("oil"),
		Rare:  ezfyResMaxOf("rare"),
		Gold:  0,
	}
	inc := ezfyCarry{Gold: 12345}
	fit, excess := ezfyCarryFitAt(cur, inc)
	if fit.Gold != inc.Gold || excess.Gold != 0 {
		t.Fatalf("其它资源满时黄金应正常入库: fit=%+v excess=%+v", fit, excess)
	}
	for _, v := range []int64{fit.Food, fit.Steel, fit.Oil, fit.Rare} {
		if v != 0 {
			t.Fatalf("已满资源不应再入库: fit=%+v", fit)
		}
	}
}
