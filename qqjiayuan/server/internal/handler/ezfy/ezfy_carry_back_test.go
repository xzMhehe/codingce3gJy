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
	if !strings.Contains(body, `"gold"`) {
		t.Fatalf("finishReturn 入城时没有累加黄金（携带的黄金会丢）：\n%s", body)
	}
}
