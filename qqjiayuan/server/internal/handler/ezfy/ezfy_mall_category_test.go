package ezfy

import (
	"testing"

	"qqjiayuan/server/internal/model"
)

func TestEzfyCategoryOfFunctionFirst(t *testing.T) {
	cases := []struct {
		name         string
		category     string
		itemType     int
		priceDiamond int64
		want         string
	}{
		{"钻石定价加速仍归加速", "", 3, 2, "加速道具"},
		{"显式功能分类优先", "资源道具", 3, 2, "资源道具"},
		{"钻石锁定不霸占展示分类", "钻石道具", 15, 0, "出征道具"},
		{"黄金锁定不霸占展示分类", "黄金道具", 7, 0, "增益道具"},
		{"未知类型保留显式货币分类", "钻石道具", 99, 10, "钻石道具"},
		{"未知类型按钻石价兜底", "", 99, 10, "钻石道具"},
		{"未知类型落其他", "", 99, 0, "其他"},
		{"移民法案归内政", "钻石道具", 31, 2, "内政道具"},
		{"军团联络令归军团", "钻石道具", 32, 10, "军团道具"},
	}
	for _, c := range cases {
		if got := ezfyCategoryOf(c.category, c.itemType, c.priceDiamond); got != c.want {
			t.Errorf("%s: ezfyCategoryOf(%q, %d, %d) = %q, want %q",
				c.name, c.category, c.itemType, c.priceDiamond, got, c.want)
		}
	}
}

func TestEzfySortItemsByCategory(t *testing.T) {
	items := []model.EzfyCfgItem{
		{ID: 3, ItemType: 15},
		{ID: 2, ItemType: 3, PriceDiamond: 2},
		{ID: 1, ItemType: 1},
		{ID: 4, ItemType: 3},
	}
	ezfySortItemsByCategory(items)
	wantIDs := []int{1, 2, 4, 3}
	for i, want := range wantIDs {
		if items[i].ID != want {
			t.Fatalf("排序后第 %d 个 ID = %d, want %d（完整顺序 %v）", i, items[i].ID, want, items)
		}
	}
	if got := ezfyItemCategory(&items[2]); got != "加速道具" {
		t.Fatalf("同类商品应聚在一起，第 3 个分类 = %q", got)
	}
}
