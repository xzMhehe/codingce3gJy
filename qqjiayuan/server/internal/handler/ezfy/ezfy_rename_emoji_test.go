package ezfy

import "testing"

// TestEzfyHasEmoji 昵称禁表情（2026-10-11 用户要求「[修改昵称] 不能用表情」）。
//
// 规则：表情区块一律拦（含国旗 / 组合 emoji / 变体选择符 / 键帽）；
// ★☆ 与 ← → ↑ ↓ 这类常用符号放行，避免误伤。
func TestEzfyHasEmoji(t *testing.T) {
	blocked := []string{
		"😀", "🎉", "🔥", "🚀", "🧡", "🀄", // 1F000–1FAFF
		"❤", "✅", "⚡", "✂", "✈", // 2600–27BF
		"⭐", "⬛", // 2B00–2BFF
		"🇨🇳",    // 国旗（区域指示符，在 1F1E6–1F1FF）
		"1️⃣",     // 组合键帽（含 FE0F + 20E3）
		"👨‍👩‍👦",   // 组合 emoji（含零宽连接符 200D）
		"名字😀",   // 混在正常文字里也要拦
	}
	for _, s := range blocked {
		if !ezfyHasEmoji(s) {
			t.Fatalf("%q 应判定为含表情", s)
		}
	}
	allowed := []string{
		"张三", "Player01", "龙傲天", "名字(1)", "①②③",
		"★彡", "☆", // 常用装饰符号，不算表情
		"A→B", "↑", // 常用箭头
		"·", "～", "-_", " ", "【】", "！？",
	}
	for _, s := range allowed {
		if ezfyHasEmoji(s) {
			t.Fatalf("%q 被误判为表情（不该拦）", s)
		}
	}
}
