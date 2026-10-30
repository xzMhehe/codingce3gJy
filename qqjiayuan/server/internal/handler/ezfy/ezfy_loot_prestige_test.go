package ezfy

import (
	"strings"
	"testing"
)

// TestLootCityPrestigeReasonable 掠夺玩家城市的军功声望必须「封顶」。
//
// ★★ 2026-10-30 用户反馈：线上战报出现
//
//	「军功声望+312679 / 个人战绩:312679」—— 有人靠反复掠夺资源多的城刷声望/军衔。
//
// 根因：prestigeGain = 200 + (粮+钢+油+稀矿)/10000，后期单次掠夺 31 亿资源 →
// 312479 + 200 = 312679。声望是军衔晋升的唯一来源，必须收口。
func TestLootCityPrestigeReasonable(t *testing.T) {
	cases := []struct {
		res  int64
		want int
	}{
		{0, 200},                 // 没抢到东西：只有基础功劳
		{10_000, 201},            // 1 万资源 +1
		{1_000_000, 210},         // 100 万资源 +10
		{100_000_000, 300},       // 1 亿资源 → 加成吃满
		{3_124_799_997, 300},     // 线上那场（31 亿）→ 仍然是 300，不再是 31 万
		{9_000_000_000_000, 300}, // 极端值也不会溢出上限
	}
	for _, c := range cases {
		if got := ezfyLootCityPrestige(c.res); got != c.want {
			t.Fatalf("掠夺资源 %d → 声望 %d, 期望 %d", c.res, got, c.want)
		}
	}
	// 掠夺单场上限必须**低于**征服玩家城（500），否则「打下来」不如「抢一把」
	if ezfyLootCityPrestige(9_000_000_000) >= 500 {
		t.Fatal("掠夺声望上限不应达到/超过征服玩家城的 500")
	}
}

// TestLootPrestigeNotResourceLinear 静态护栏：掠夺声望不得再按资源线性换算。
//
// 旧的 `200 + int((lootFood+lootSteel+lootOil+lootRare)/10000)` 一旦被人加回来，
// 声望又会被资源量放大成天文数字。
func TestLootPrestigeNotResourceLinear(t *testing.T) {
	src := strings.ReplaceAll(rawFile(t, "ezfy_order.go"), "\r\n", "\n")
	code := stripGoCommentLines(src)
	if strings.Contains(code, "(lootFood+lootSteel+lootOil+lootRare)/10000") {
		t.Fatal("掠夺声望又变回「按资源量线性换算」（刷声望 bug 回归）")
	}
	if !strings.Contains(code, "prestigeGain = ezfyLootCityPrestige(lootFood + lootSteel + lootOil + lootRare)") {
		t.Fatal("掠夺玩家城的声望没有走 ezfyLootCityPrestige（唯一实现）")
	}
}
