package ezfy

import (
	"strings"
	"testing"
)

// ============ 兵力加减必须并发安全（★ 2026-10-30 线上事故）============
//
// 玩家反馈：「派了两队兵各 20w 火箭去刷 3 级活动寇，城里军工厂也在造火箭；
// 两队同时到、战报显示一个在结算中，然后军工厂的火箭刚好完工，就直接卡了 20w 火箭出来。」
//
// 根因（两条，互相叠加）：
//
//	① `addTroop` 是**读-改-写**：First 读出 count → Go 里算 count+n → UPDATE 写绝对值。
//	   军工厂完工（collectTrainQueue）与出征扣兵（deductCityTroops）并发时，
//	   加兵那条会把扣兵**整条覆盖**掉 → 部队出去了、城里的兵一点没少 = 凭空多一队。
//	② `deductCityTroops` 用 `GREATEST(count-?,0)` **静默夹取**：扣不够也算成功。
//	   出征校验用的是懒结算快照，两个出征请求（连点/双开/同时打两个不同目标）会
//	   同时通过校验，第二个请求扣到 0 也照样出发 → 又凭空多一队。
//
// 本文件守住四条不变量（都是「改了就会重新长 bug」的地方）：
//  1. addTroop 必须是**单条原子 upsert**（相对累加 + 上限夹取），不得退回读-改-写；
//  2. deductCityTroops 必须是**条件扣减**（`count >= ?`），扣不到要报错回滚；
//  3. createOrder 必须在**写库之前**扣兵，并检查返回值（失败即中止）；
//  4. 订单写库失败必须把兵退回去（否则「兵没了、队列里也没有」）。

// TestAddTroopIsAtomicUpsert 加兵必须是数据库侧相对累加，不能先读再写绝对值。
func TestAddTroopIsAtomicUpsert(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy.go", "func (h *EzfyHandler) addTroop(")
	if !strings.Contains(body, "clause.OnConflict") {
		t.Fatalf("addTroop 不是原子 upsert（缺 clause.OnConflict）——并发下会覆盖同期的扣兵，凭空多兵：\n%s", body)
	}
	if !strings.Contains(body, "LEAST(") {
		t.Fatalf("addTroop 没有在 SQL 里做上限夹取（LEAST）：\n%s", body)
	}
	// 回归哨兵：老的读-改-写写法（拿 First 读出来的 t.Count 算绝对值再写回）
	if strings.Contains(body, "ezfySafeAdd(t.Count") {
		t.Fatalf("addTroop 退回了「读-改-写绝对值」——这正是凭空多兵的根因：\n%s", body)
	}
	if strings.Contains(body, `Update("count", ezfySafeAdd(`) {
		t.Fatalf("addTroop 用绝对新值覆盖 count：并发扣兵会被整条覆盖：\n%s", body)
	}
}

// TestDeductCityTroopsRequiresEnough 扣兵必须条件扣减，扣不够要报错。
func TestDeductCityTroopsRequiresEnough(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) deductCityTroops(")
	if !strings.Contains(body, "count >= ?") {
		t.Fatalf("deductCityTroops 没有用「count >= ?」条件扣减：扣不到兵也会静默成功 → 部队凭空出发：\n%s", body)
	}
	if !strings.Contains(body, "RowsAffected == 0") {
		t.Fatalf("deductCityTroops 没有检查 RowsAffected（扣不到兵必须报错）：\n%s", body)
	}
	if !strings.Contains(body, "restoreTroops") {
		t.Fatalf("deductCityTroops 失败时没有回滚已扣的兵种：\n%s", body)
	}
	if strings.Contains(body, "GREATEST(count - ") {
		t.Fatalf("deductCityTroops 退回了 GREATEST 静默夹取（本 bug 的根因）：\n%s", body)
	}
}

// TestDeductTroopCappedIsRelative 战损扣减也必须用相对表达式，不能先读再写绝对值。
func TestDeductTroopCappedIsRelative(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy.go", "func (h *EzfyHandler) deductTroopCapped(")
	if !strings.Contains(body, "GREATEST(ezfy_city_troop.count - ?") {
		t.Fatalf("deductTroopCapped 不是原子相对扣减：\n%s", body)
	}
	if strings.Contains(body, "First(&exist)") {
		t.Fatalf("deductTroopCapped 退回了「先读再写绝对值」：\n%s", body)
	}
}

// TestCreateOrderDeductsBeforeWrite 扣兵必须在写库之前完成并检查返回值。
func TestCreateOrderDeductsBeforeWrite(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) createOrder(")
	deduct := strings.Index(body, "h.deductCityTroops(city.ID, validTroops)")
	if deduct < 0 {
		t.Fatalf("createOrder 没有扣兵调用：\n%s", body)
	}
	wait := strings.Index(body, "wwg.Wait()")
	if wait < 0 {
		t.Fatalf("createOrder 找不到写库并行块的 wwg.Wait()（结构变了，请同步本测试）：\n%s", body)
	}
	if deduct > wait {
		t.Fatalf("扣兵在写库之后执行 —— 扣不到兵也没法中止出征，必须提前到写库之前：\n%s", body)
	}
	if !strings.Contains(body, "if msg := h.deductCityTroops(") {
		t.Fatalf("createOrder 没有检查扣兵返回值（失败必须中止并回滚军官状态）：\n%s", body)
	}
	// 并行块里不能再有第二次扣兵（否则重复扣）
	if strings.Count(body, "h.deductCityTroops(") != 1 {
		t.Fatalf("createOrder 里出现了多次扣兵调用（重复扣兵）：\n%s", body)
	}
}

// TestCreateOrderRestoresTroopsOnOrderFail 订单写库失败必须把兵退回去。
func TestCreateOrderRestoresTroopsOnOrderFail(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) createOrder(")
	i := strings.Index(body, "if orderErr != nil {")
	if i < 0 {
		t.Fatalf("createOrder 找不到订单写入失败分支：\n%s", body)
	}
	tail := body[i:]
	if !strings.Contains(tail, "h.restoreTroops(") {
		t.Fatalf("订单写入失败没有退回已扣的兵（玩家会「兵没了、队列里也没有」）：\n%s", tail)
	}
}
