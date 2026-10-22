package ezfy

import "testing"

// TestEzfyTroopSortKey 兵种展示顺序：海军轰炸机(21) 要排在 轰炸机(11) 与 特种兵(12) 之间。
//
// ★ 2026-10-22 新增兵种只能用新 id（1~20 是老兵种固定 id），
// 若直接按 id 排序，海军轰炸机会掉到列表最末尾（防空炮之后），
// 所以用 ezfyTroopSortKey 单独给它一个插入位；其余兵种相对顺序必须保持不变。
func TestEzfyTroopSortKey(t *testing.T) {
	if !(ezfyTroopSortKey(11) < ezfyTroopSortKey(ezfyTroopNavalBomber)) {
		t.Fatalf("海军轰炸机没有排在轰炸机(11)之后")
	}
	if !(ezfyTroopSortKey(ezfyTroopNavalBomber) < ezfyTroopSortKey(12)) {
		t.Fatalf("海军轰炸机没有排在特种兵(12)之前")
	}
	// 老兵种之间相对顺序不变（严格单调递增）
	prev := -1
	for id := 1; id <= 20; id++ {
		k := ezfyTroopSortKey(id)
		if k <= prev {
			t.Fatalf("老兵种 id=%d 的排序键 %d 没有递增（前一个 %d）", id, k, prev)
		}
		prev = k
	}
}
