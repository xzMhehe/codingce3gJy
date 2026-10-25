package ezfy

import (
	"strings"
	"testing"
)

// ============ 洗点口径守卫（2026-10-25 反复确认后的最终版）============
//
// 用户原话：
//
//	「洗点就是洗点成原来军官池子武将的属性，等级不变；
//	 待分配的属性 = 现在属性之和 − 军官池属性之和，玩家可以重新分配加点」
//	「你这洗点不对啊 没把升星的属性洗下来……刚才还能洗下来的」
//	「提示不要这么多 简约提示」
//
// ⇒ **洗点把三维全部洗回军官池属性，玩家加的点 + 升星加的点一起退回成待分配点**；
// 升星明细（star_points / star_military / star_logistics / star_learning）一并归零。
//
// ⚠️ 中途试过「只洗玩家加的点、保留升星加成」（洗点目标 = 池属性 + 升星明细），
// 结果玩家洗点退回 0 点，被否 —— 别再改回去。

// TestRespecMessageIsShort —— 洗点提示必须是一句话（不要再罗列前后属性/升星加成）
func TestRespecMessageIsShort(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy.go", "func (h *EzfyHandler) useItemOnce(")
	if body == "" {
		t.Skip("useItemOnce 定位失败，跳过")
	}
	if !strings.Contains(body, `"使用成功: 洗点完成，待分配属性点 +%d"`) {
		t.Fatal("洗点提示应简约为「使用成功: 洗点完成，待分配属性点 +N」")
	}
	// ⚠️ 「使用成功」前缀不能丢：useItem 靠它区分成功/失败，少了会返回 400
	if !strings.Contains(body, "使用成功") {
		t.Fatal("洗点提示必须带「使用成功」前缀（否则 useItem 会当失败返回 400）")
	}
	for _, bad := range []string{"其中升星加点", "升星加成 军事+", "属性已重置为军官池初始属性"} {
		if strings.Contains(body, bad) {
			t.Fatalf("洗点提示不该再包含 %q（用户要求简约提示）", bad)
		}
	}
}

// TestRespecResetsToPool —— 洗点把三维洗回池属性，升星明细归零（静态断言）
func TestRespecResetsToPool(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy.go", "func (h *EzfyHandler) useItemOnce(")
	if body == "" {
		t.Skip("useItemOnce 定位失败，跳过")
	}
	if strings.Contains(body, "officerStarSplit") {
		t.Fatal("洗点不该再按升星明细保留加成（会被玩家反馈「退 0 点」）")
	}
	for _, want := range []string{
		`"military": bm, "logistics": bl, "learning": be`,
		`"star_points": 0`,
		`"star_military": 0`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("洗点分支缺少 %q", want)
		}
	}
}
