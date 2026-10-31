package ezfy

import "testing"

// ★★ 2026-10-31 用户反馈（线上战报）：「个人荣誉:1333633 获取也太变态了，算法改下，少些」。
//
// 个人荣誉 = 军衔声望的开方压缩（纯展示，不参与任何结算）。
// 老算法是 `prestige / 6`：声望是历史累计，老玩家几百万声望 → 个人荣誉 7 位数，
// 与旁边的「个人战绩 100」完全不是一个量级。改为开方后 1 亿声望也才 1 万。
func TestPersonalHonorCompressed(t *testing.T) {
	cases := []struct {
		prestige int
		want     int
	}{
		{0, 0},
		{-100, 0}, // 异常负数按 0
		{100, 10},
		{10000, 100},
		{1000000, 1000},
		{8001800, 2828}, // 用户截图里那场：老算法 1333633
		{100000000, 10000},
	}
	for _, c := range cases {
		if got := ezfyPersonalHonor(c.prestige); got != c.want {
			t.Fatalf("个人荣誉(声望 %d) = %d，期望 %d", c.prestige, got, c.want)
		}
	}
	// 单调递增：声望涨 → 荣誉不降
	prev := -1
	for p := 1; p <= 100000000; p *= 10 {
		got := ezfyPersonalHonor(p)
		if got < prev {
			t.Fatalf("个人荣誉不单调：声望 %d → %d，上一档 %d", p, got, prev)
		}
		prev = got
	}
	// 与老口径对比：8 百万声望从 1333633 降到 4 位数以内
	if got := ezfyPersonalHonor(8001800); got >= 10000 {
		t.Fatalf("8 百万声望的个人荣誉应压到 1 万以内，实际 %d", got)
	}
}
