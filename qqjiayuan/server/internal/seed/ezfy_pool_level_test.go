package seed

import "testing"

// TestPoolLevelSpread —— 普通军官池的等级/属性规则断言。
//
// ★ 规则演进（改断言前先看这里）：
//   - 2026-09-26 用户要求「军官池子普通军官等级不对，全变 150 了，要有等级差距」→
//     当时改成按星级分层随机（1星 1~60 … 5星 120~150），本测试就是那时加的。
//   - ★ 2026-09-29 用户**改口径**：「所有池子军官统一 1 级」，属性改为
//     「三属性之和 ≤200 且按星级递减」（`ezfyPoolAttrSumOf`），按等级缩放（`ezfyScaleByLevel`）废弃。
//     → 于是「等级必须有差距」这条断言**与现行规则相反**，测试一直红着（存量红测）。
//     本文件按现行规则重写断言：等级恒 1；把「等级差距」改成「三属性之和必须随星级递增」，
//     保住原测试的意图（防止池子数值被一刀切）。
func TestPoolLevelSpread(t *testing.T) {
	byStar := map[int][]int{} // star → 三属性之和
	for _, o := range buildEzfyPoolOfficers() {
		// 现行规则：池子里所有普通军官都是 1 级（属性已按星级定档，不再随等级缩放）
		if o.Level != 1 {
			t.Fatalf("%s 等级应为 1（2026-09-29 用户规则「所有池子军官统一 1 级」），实际 %d",
				o.Name, o.Level)
		}
		if o.Military < 1 || o.Logistics < 1 || o.Learning < 1 {
			t.Fatalf("%s 属性出现 0（%d/%d/%d）", o.Name, o.Military, o.Logistics, o.Learning)
		}
		sum := o.Military + o.Logistics + o.Learning
		if sum > 200 {
			t.Fatalf("%s 三属性之和 %d 超过上限 200", o.Name, sum)
		}
		if want := ezfyPoolAttrSumOf(o.Star); sum != want {
			t.Fatalf("%s(%d星) 三属性之和应为 %d，实际 %d（%d/%d/%d）",
				o.Name, o.Star, want, sum, o.Military, o.Logistics, o.Learning)
		}
		byStar[o.Star] = append(byStar[o.Star], sum)
	}
	for st := 1; st <= 5; st++ {
		if len(byStar[st]) == 0 {
			t.Fatalf("%d 星没有任何池子军官", st)
		}
		t.Logf("%d星 %d 人，属性之和 %d", st, len(byStar[st]), byStar[st][0])
	}
	// 「差距」现在体现在属性之和上：必须随星级严格递增
	for st := 2; st <= 5; st++ {
		if ezfyPoolAttrSumOf(st) <= ezfyPoolAttrSumOf(st-1) {
			t.Fatalf("%d星 属性之和没有高于 %d星（%d vs %d）—— 星级差距丢失",
				st, st-1, ezfyPoolAttrSumOf(st), ezfyPoolAttrSumOf(st-1))
		}
	}
}
