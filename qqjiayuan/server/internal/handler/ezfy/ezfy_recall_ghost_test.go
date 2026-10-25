package ezfy

import (
	"os"
	"strings"
	"testing"

	"qqjiayuan/server/internal/model"
)

// 回归：2026-10-25 玩家反馈「不小心全部召回的卡车实际没回来；现在野地有卡车，
//
//	大号显示没卡车，用小号征服就能发现有卡车」。
//
// 根因（两个叠加）：
//  1. finishReturn 返航归队时**只删 order_type=6**，采集队(7) 残留 status=3；
//  2. 各处野地守军口径写成 `order_type IN (6,7) AND status IN (1,3)`，
//     把「已经回城」的采集队继续当成野地守军 → 野地上永远挂着一支
//     「打不死、召不回、驻军列表也看不到」的幽灵部队。
//
// 现场数据（本地 qq_ezzt 快照）：135 条 order_type=7 AND status=3，
// 全部 result 非空（兵已入城）、carry 已清空 —— 正是已归队的僵尸单。
func TestWildDefenderCondExcludesReturnedGather(t *testing.T) {
	// 口径文本：采集队(7) 只能 status=1；驻军(6) 才允许 1/3
	if !strings.Contains(ezfyWildDefenderOrderCond, "order_type = 7 AND status = 1") {
		t.Fatalf("守军口径必须限定「采集队(7) 只有 status=1 才算守军」：%s", ezfyWildDefenderOrderCond)
	}
	if !strings.Contains(ezfyWildDefenderOrderCond, "order_type = 6 AND status IN (1,3)") {
		t.Fatalf("守军口径必须保留「野地驻军(6) status 1/3 都算」：%s", ezfyWildDefenderOrderCond)
	}

	// 行为口径（Go 侧判断必须与 SQL 常量一致）
	cases := []struct {
		order model.EzfyOrder
		want  bool
		desc  string
	}{
		{model.EzfyOrder{OrderType: 7, TargetType: 1, Status: 1}, true, "在场采集队(7/1)算守军"},
		{model.EzfyOrder{OrderType: 7, TargetType: 1, Status: 3}, false, "已返航归队的采集队(7/3)不能再算守军"},
		{model.EzfyOrder{OrderType: 6, TargetType: 1, Status: 3}, true, "野地驻军(6/3)算守军"},
		{model.EzfyOrder{OrderType: 6, TargetType: 2, Status: 1}, true, "异常态驻军(6/1)保守算守军"},
		{model.EzfyOrder{OrderType: 7, TargetType: 3, Status: 1}, false, "城市增援不是野地守军"},
	}
	for _, c := range cases {
		o := c.order
		if got := ezfyWildlandOrderCanDefend(&o); got != c.want {
			t.Fatalf("%s：ezfyWildlandOrderCanDefend=%v, want %v", c.desc, got, c.want)
		}
	}
}

// 返航归队必须删掉采集队订单 —— 否则野地守军口径一放宽就复活成幽灵部队。
func TestFinishReturnDeletesGatherOrder(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) finishReturn(")
	if !strings.Contains(body, "if order.OrderType == 6 || order.OrderType == 7 {") {
		t.Fatal("finishReturn 未删除采集队(7)订单 → 归队后残留 status=3，野地会挂幽灵部队")
	}
	if !strings.Contains(body, "h.DB.Delete(&model.EzfyOrder{}, order.ID)") {
		t.Fatal("finishReturn 未删除订单")
	}
}

// 各处守军查询必须复用常量，禁止再写死 `order_type IN (6,7) AND status IN (1,3)`。
// （2026-10-25 之前就是这段字面量在 4 个地方各写一份，改漏一处就又出幽灵。）
func TestNoLegacyWildDefenderCond(t *testing.T) {
	for _, f := range []string{"ezfy_order.go", "ezfy_api.go"} {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("读源码失败 %s: %v", f, err)
		}
		for i, line := range strings.Split(string(b), "\n") {
			trim := strings.TrimSpace(line)
			if strings.HasPrefix(trim, "//") {
				continue // 注释里提到旧口径无所谓
			}
			if strings.Contains(line, "order_type IN (6,7)") {
				t.Fatalf("%s:%d 仍在写死旧守军口径，必须改用 ezfyWildDefenderOrderCond：%s", f, i+1, trim)
			}
		}
	}
}
