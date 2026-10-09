package ezfy

import (
	"strings"
	"testing"
)

// ============ 「自己被处于指挥时，指挥里面的兵种不能再出征」守卫 ============
//
// 用户规则（2026-10-09）：
//   - 本城正在被敌军攻击、进入防御指挥室（指挥中）时，**参战兵种**不能再出征 ——
//     战斗结算的守方战损是「按城池当前兵力扣减」的（见 processArrive 守方战损块），
//     若允许把参战守军出征带走，战损就扣不到兵，等于把守军从「出征」这个口子悄悄转移走，
//     让这场防御白打。
//   - 例外（用户明确）：司令部标了「不参与防御」(def_move=-1) 的兵种**不参与当次防御**，
//     带走它们不影响战斗结算 → 照常放行。
//
// 这三条守卫分别盯：判定口径 / 出征拦截 / 前端数据源。规则再改时记得同步这里。

// TestCityDefendLockedExcludesNonDefenders —— 判定集合必须排除「不参与防御」的兵种
func TestCityDefendLockedExcludesNonDefenders(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) cityDefendLocked(")
	if !strings.Contains(body, "defExcludeSet(cityId)") {
		t.Fatal("cityDefendLocked 必须用 defExcludeSet 排除「不参与防御」的兵种 —— 它们不参与当次防御，应当放行")
	}
	if !strings.Contains(body, "ezfyOrderStatusBattle") {
		t.Fatal("cityDefendLocked 必须以「订单 status=5(战斗中)」判定指挥室是否仍在进行")
	}
	if !strings.Contains(body, "target_id = ? AND status = 1") {
		t.Fatal("cityDefendLocked 必须查「本城为守方、进行中」的战场（target_id + status=1）")
	}
}

// TestCreateOrderBlocksDefendersWhileUnderAttack —— 出征入口必须逐兵种拦参战守军
func TestCreateOrderBlocksDefendersWhileUnderAttack(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) createOrder(")
	if !strings.Contains(body, "cityDefendLocked(city.ID)") {
		t.Fatal("createOrder 必须取本城「防御指挥中」的参战兵种锁定集合（cityDefendLocked）")
	}
	if !strings.Contains(body, "defLocked[t.TroopId]") {
		t.Fatal("createOrder 必须对出征兵力逐兵种检查是否被锁定（defLocked[t.TroopId]）")
	}
	if !strings.Contains(body, "参战部队") {
		t.Fatal("createOrder 的拦截文案应当说明「参战部队不能出征」")
	}
}

// TestOrderPreviewSendsLockedTroops —— 预览接口下发锁定列表（前端据此禁用数量框）
func TestOrderPreviewSendsLockedTroops(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) OrderPreview(")
	for _, want := range []string{"def_locked", "locked_troops", "cityDefendLocked(city.ID)"} {
		if !strings.Contains(body, want) {
			t.Fatalf("OrderPreview 应当下发 `%s`（前端据此禁用参战兵种的数量框）", want)
		}
	}
}

// TestIntSetKeysSorted —— 下发用的小工具：nil 也要给空数组（前端 Array 操作不炸）
func TestIntSetKeysSorted(t *testing.T) {
	if got := ezfyIntSetKeys(nil); len(got) != 0 {
		t.Fatalf("nil 集合应当返回空切片，实际 %v", got)
	}
	got := ezfyIntSetKeys(map[int]bool{3: true, 1: true, 2: false})
	want := []int{1, 3}
	if len(got) != len(want) {
		t.Fatalf("集合键数量不对: %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("集合键应当升序且只含真值: got %v want %v", got, want)
		}
	}
}
