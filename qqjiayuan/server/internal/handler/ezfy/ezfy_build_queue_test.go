package ezfy

import "testing"

// TestEzfyBuildQueueCalc 建造队列上限计算：基础 5 + 未过期的「征募告示」层数。
//
// ★ 2026-10-22 用户规则：默认每城 5 个建造队列；「征募告示」+1 队列 / 24 小时，
// 可叠加最多 10 个；过期后整体失效。
func TestEzfyBuildQueueCalc(t *testing.T) {
	const now = int64(1_700_000_000_000)
	future := now + 3600*1000
	past := now - 1

	cases := []struct {
		name    string
		base    int
		stacks  int
		until   int64
		wantMax int
		wantBon int
	}{
		{"无加成=基础上限", 5, 0, 0, 5, 0},
		{"基础上限 0 回落 5", 0, 0, 0, 5, 0},
		{"用 1 个征募告示", 5, 1, future, 6, 1},
		{"叠满 10 个", 5, 10, future, 15, 10},
		{"层数超上限夹到 10", 5, 12, future, 15, 10},
		{"已过期整体失效", 5, 10, past, 5, 0},
		{"管理员把基础改成 8 再叠 2", 8, 2, future, 10, 2},
	}
	for _, c := range cases {
		st := ezfyBuildQueueCalc(c.base, c.stacks, c.until, now)
		if st.Max != c.wantMax || st.Bonus != c.wantBon {
			t.Errorf("%s: Max=%d Bonus=%d, want Max=%d Bonus=%d",
				c.name, st.Max, st.Bonus, c.wantMax, c.wantBon)
		}
	}
}
