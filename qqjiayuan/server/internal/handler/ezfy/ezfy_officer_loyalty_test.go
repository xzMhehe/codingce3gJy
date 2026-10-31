package ezfy

import "testing"

// 败仗扣忠心：基础 1 点 + 按损兵占比追加，总计封顶 6 点。
func TestEzfyLoyaltyDropOnDefeat(t *testing.T) {
	cases := []struct {
		dead, total int64
		want        int
	}{
		{0, 10104900, 1},      // 没战损 → 只扣基础 1
		{10104900, 10104900, 6}, // 全灭（用户战报场景）→ 封顶 6，不再是 9
		{5052450, 10104900, 3},  // 损兵 50% → 1 + 2
		{2020980, 10104900, 2},  // 损兵 20% → 1 + 1
		{999999999, 10104900, 6}, // 战损超过总兵力 → 仍封顶 6
		{100, 0, 1},              // 总兵力数据缺失 → 只扣基础 1
	}
	for _, c := range cases {
		if got := ezfyLoyaltyDropOnDefeat(c.dead, c.total); got != c.want {
			t.Fatalf("战损 %d / 总兵力 %d 应扣 %d, 实际 %d", c.dead, c.total, c.want, got)
		}
	}
}
