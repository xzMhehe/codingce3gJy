package handler

import (
	"strings"
	"testing"
)

// ============ 「返航时长」口径守卫 ============
//
// 背景（2026-09-22 线上反馈）：「我有个军队 20717 天 9 小时 32 分才能回来」。
//
// 根因：`processArrive` 等处用 `|ArriveTime - StartTime|` 现算「单程行军时长」，
// 而 `ArriveTime` 是一个**会被外部改动**的字段：
//   - 测试/运维为了免等待会把 arrive_time 拨到 1000（= 1970 年）；
//   - 指挥室流程一度把它拨到 now。
//
// 只要它被动过，算出来的 travel 就会变成几十年，返航时间直接爆表。
//
// 正确口径（`ezfyOneWayTravel`）：创建订单时 `ReturnTime = start + 2*travel`，
// 所以 `(ReturnTime - StartTime) / 2` 恒为单程时长，且这两个字段结算时才会写。

// TestProcessArriveUsesOneWayTravel —— processArrive 不得再用 ArriveTime-StartTime
func TestProcessArriveUsesOneWayTravel(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) processArrive(")
	for _, bad := range []string{"ArriveTime - order.StartTime", "ezfyAbs64(order.ArriveTime"} {
		if strings.Contains(body, bad) {
			t.Fatalf("processArrive 里又出现 `%s` —— 返航时长会算出天文数字（线上「20717 天」就是这个）。\n"+
				"请改用 ezfyOneWayTravel(order)。", bad)
		}
	}
	if !strings.Contains(body, "ezfyOneWayTravel(order)") {
		t.Fatal("processArrive 里应当有 ezfyOneWayTravel(order) 作为返程时长口径")
	}
}

// TestBeginReturnUsesOneWayTravel —— beginReturn 的兜底同样不能碰 ArriveTime-StartTime
func TestBeginReturnUsesOneWayTravel(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) beginReturn(")
	if strings.Contains(body, "ArriveTime - order.StartTime") {
		t.Fatal("beginReturn 的兜底又用了 ArriveTime-StartTime，请改用 ezfyOneWayTravel(order)")
	}
}

// TestOneWayTravelPrefersReturnTime —— 口径本身要优先用 ReturnTime-StartTime
func TestOneWayTravelPrefersReturnTime(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_order.go", "func ezfyOneWayTravel(")
	if !strings.Contains(body, "order.ReturnTime - order.StartTime") {
		t.Fatal("ezfyOneWayTravel 必须以 (ReturnTime-StartTime)/2 为首选口径")
	}
}

// TestBattleFinishDoesNotTouchArriveTime —— 指挥室结束战斗时不得改 arrive_time
//
// 指挥室只需要写 battle_result + 把 status 拨回 0；
// 结算由 processOrders 直接调 processArrive 触发，**不依赖 arrive_time**。
func TestBattleFinishDoesNotTouchArriveTime(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_battle_room.go", "func (h *EzfyHandler) ezfyBattleFinishToOrder(")
	if strings.Contains(body, `"arrive_time"`) {
		t.Fatal("ezfyBattleFinishToOrder 又在改 arrive_time —— 它是单程时长的计算基准，会让返航时间爆表")
	}
}

// TestBattleTickDoesNotTouchOrder —— 战场推进不应写订单表
func TestBattleTickDoesNotTouchOrder(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_battle_room.go", "func (h *EzfyHandler) ezfyBattleTick(")
	if strings.Contains(body, "model.EzfyOrder{}") {
		t.Fatal("ezfyBattleTick 不应写订单表（推进回合只该动 ezfy_battle）")
	}
}
