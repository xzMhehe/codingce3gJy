package handler

import "testing"

// 兵种目标（2026-09-23 用户要求）：
// 「指挥战场的时候兵种目标带过来，指挥的时候玩家也能配置，默认是司令部配置的，
//   敌对没有目标则默认攻击距离最近的」。
//
// 这组测试盯的是 ezfyPickTarget 的语义。
//
// ⚠️ 老实现里「优先攻击目标」**完全不生效**：它先把「全局最近距离」算进 minDist，
// 再拿配置兵种的距离去比 `d < minDist` —— 配置兵种的距离永远不可能小于全局最小值
// （顶多相等，而相等也不满足 `<`），所以司令部配了优先目标也永远打最近的。
// 修复后必须满足下面三条，否则「默认是司令部配置的」这句话就是空的。

// ezfyTgtUnit 造一个只关心 troopId / 位置的测试单位
func ezfyTgtUnit(id string, troopId, pos int) *ezfyFightUnit {
	return &ezfyFightUnit{
		id: id, count: 100, initialCount: 100, pos: pos,
		cfg: &ezfyTroopStats{
			ID: troopId, Name: "兵种", Type: 2, Health: 100,
			AtkGround: 200, Defence: 10, Speed: 100, AttackRange: 500,
		},
	}
}

// TestPickTargetPrefersConfiguredTroop —— 配了优先兵种且敌方有该兵种 → 打它，哪怕更远
func TestPickTargetPrefersConfiguredTroop(t *testing.T) {
	me := ezfyTgtUnit("A1", 1, 0)
	near := ezfyTgtUnit("D1", 9, 1000) // 更近，但不是配置目标
	far := ezfyTgtUnit("D2", 7, 5000)  // 更远，是配置目标

	got := ezfyPickTarget(me, []*ezfyFightUnit{near, far}, map[int]int{1: 7})
	if got == nil || got.cfg.ID != 7 {
		t.Fatalf("应优先打配置的兵种 7（哪怕更远），实际打了 %v", got)
	}
}

// TestPickTargetFallsBackToNearest —— 配的兵种敌方没有/已全灭 → 回落打最近的
func TestPickTargetFallsBackToNearest(t *testing.T) {
	me := ezfyTgtUnit("A1", 1, 0)
	near := ezfyTgtUnit("D1", 9, 1000)
	far := ezfyTgtUnit("D2", 7, 5000)

	// 配了 5，但敌方只有 9 和 7 → 打最近的 9
	got := ezfyPickTarget(me, []*ezfyFightUnit{near, far}, map[int]int{1: 5})
	if got == nil || got.cfg.ID != 9 {
		t.Fatalf("敌方没有兵种 5 时应打最近的兵种 9，实际打了 %v", got)
	}
}

// TestPickTargetNoConfigPicksNearest —— 没配目标 → 打最近的
func TestPickTargetNoConfigPicksNearest(t *testing.T) {
	me := ezfyTgtUnit("A1", 1, 0)
	near := ezfyTgtUnit("D1", 9, 1000)
	far := ezfyTgtUnit("D2", 7, 5000)

	got := ezfyPickTarget(me, []*ezfyFightUnit{near, far}, nil)
	if got == nil || got.cfg.ID != 9 {
		t.Fatalf("没配目标时应打最近的兵种 9，实际打了 %v", got)
	}
}

// TestStepHitsConfiguredTarget —— 端到端：配了优先目标后，战损真的落在配置的兵种身上
func TestStepHitsConfiguredTarget(t *testing.T) {
	me := ezfyTgtUnit("A1", 1, 0)
	near := ezfyTgtUnit("D1", 9, 300) // 两个守方都在攻方射程(500)内
	far := ezfyTgtUnit("D2", 7, 400)

	st := &ezfyBattleState{
		Attackers:  []*ezfyFightUnit{me},
		Defenders:  []*ezfyFightUnit{near, far},
		AtkTargets: map[int]int{1: 7}, // 攻方兵种 1 优先打兵种 7
		Head:       []string{}, Actions: []string{},
	}
	st.Step(nil, nil)

	if far.count >= far.initialCount {
		t.Fatalf("配置的优先目标（兵种 7）应该挨打，实际剩余 %d/%d", far.count, far.initialCount)
	}
	if near.count != near.initialCount {
		t.Fatalf("非目标兵种 9 不该挨打，实际剩余 %d/%d", near.count, near.initialCount)
	}
}
