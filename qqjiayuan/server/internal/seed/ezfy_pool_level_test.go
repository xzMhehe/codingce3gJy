package seed

import "testing"

// TestPoolLevelSpread —— 普通军官池的等级必须有差距（不再全是 150），且落在 1~150
//
// ★ 2026-09-26 用户要求：「军官池子普通军官等级不对，全变 150 了，要有等级差距」，
// 属性按等级同比缩放（150 级基准 × level/150）。
func TestPoolLevelSpread(t *testing.T) {
	byStar := map[int][]int{}
	for _, o := range buildEzfyPoolOfficers() {
		if o.Level < 1 || o.Level > 150 {
			t.Fatalf("%s 等级 %d 越界(应在 1~150)", o.Name, o.Level)
		}
		if o.Military < 1 || o.Logistics < 1 || o.Learning < 1 {
			t.Fatalf("%s 属性出现 0（%d/%d/%d）", o.Name, o.Military, o.Logistics, o.Learning)
		}
		byStar[o.Star] = append(byStar[o.Star], o.Level)
	}
	for st := 1; st <= 5; st++ {
		lv := byStar[st]
		if len(lv) == 0 {
			continue
		}
		min, max := lv[0], lv[0]
		for _, v := range lv {
			if v < min {
				min = v
			}
			if v > max {
				max = v
			}
		}
		if min == max {
			t.Fatalf("%d星 等级没有差距（全是 %d）", st, min)
		}
		t.Logf("%d星 %d 人，等级 %d~%d", st, len(lv), min, max)
	}
}
