package ezfy

import (
	"strings"
	"testing"
)

// TestEzfyCarryTakeDiscardsOverflow 负重装满后，多出来的资源必须直接丢弃。
func TestEzfyCarryTakeDiscardsOverflow(t *testing.T) {
	c := ezfyCarry{Food: 150}
	if fit, dropped := ezfyCarryTake(c, 100); fit.Food != 100 || dropped != 50 {
		t.Fatalf("超出负重未截断: fit=%+v dropped=%d, want food=100 dropped=50", fit, dropped)
	}
	if fit, dropped := ezfyCarryTake(c, 0); fit.total() != 0 || dropped != 150 {
		t.Fatalf("负重为0时应全部丢弃: fit=%+v dropped=%d", fit, dropped)
	}
	if fit, dropped := ezfyCarryTake(c, -1); fit.Food != 150 || dropped != 0 {
		t.Fatalf("不限负重时不应截断: fit=%+v dropped=%d", fit, dropped)
	}
	if fit, dropped := ezfyCarryTake(c, 200); fit.Food != 150 || dropped != 0 {
		t.Fatalf("负重足够时不应丢弃: fit=%+v dropped=%d", fit, dropped)
	}
}

func TestEzfyGatherSetAmount(t *testing.T) {
	g := ezfyGatherCalc{ResName: "钢铁", Steel: 100, Amount: 100}
	ezfyGatherSetAmount(&g, 40)
	if g.Steel != 40 || g.Amount != 40 || g.Food != 0 {
		t.Fatalf("截断采集产出错误: %+v", g)
	}
}

// TestSettleCarryFullKeepsTreasure 负重满后只停资源，宝物仍按完整期数结算；
// 订单不能因负重满自动停止，只有宝物满才停止。
func TestSettleCarryFullKeepsTreasure(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) settleDispatch(")
	for _, want := range []string{
		"h.ezfyFitGatherPeriods(",
		"for i := int64(0); i < periods; i++",
		"if treasureFull {",
		`upd["collect_start"] = 0`,
		"宝物仍按周期产出",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("settleDispatch 缺少「负重满停资源、宝物继续」逻辑 %q：\n%s", want, body)
		}
	}
	for _, bad := range []string{"periods = fitPeriods", "if treasureFull || carryFull"} {
		if strings.Contains(body, bad) {
			t.Fatalf("settleDispatch 仍在负重满时停止/截断宝物期数 %q：\n%s", bad, body)
		}
	}
}

// TestHarvestAllIncludesStoppedCarry 兼容旧数据/异常状态下 arrive_time=0 但 carry 还在；
// 一键收获必须能把它们收掉，否则玩家无法重新采集。
func TestHarvestAllIncludesStoppedCarry(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_collect.go", "func (h *EzfyHandler) HarvestAll(")
	if !strings.Contains(body, "carry <> ''") {
		t.Fatalf("一键收获没有包含「已停止但负重未清」的部队：\n%s", body)
	}
}
