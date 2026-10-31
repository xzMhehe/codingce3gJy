package ezfy

// ★★ 2026-10-31 统帅信息「番号」：
//   - 显示 1 个字，默认「新」；
//   - 存在各区 ezfy_profile.banner，1/2 区各自独立；
//   - 修改方式同昵称（内联 [修改]），但免费、不消耗改名卡。

import (
	"os"
	"strings"
	"testing"

	"qqjiayuan/server/internal/model"
)

func TestBannerOfDefault(t *testing.T) {
	if got := ezfyBannerOf(nil); got != "新" {
		t.Fatalf("nil 档案番号应为 新，实际 %q", got)
	}
	if got := ezfyBannerOf(&model.EzfyProfile{}); got != "新" {
		t.Fatalf("空番号应回退 新，实际 %q", got)
	}
	if got := ezfyBannerOf(&model.EzfyProfile{Banner: "虎"}); got != "虎" {
		t.Fatalf("番号应原样返回 虎，实际 %q", got)
	}
}

func TestBannerValid(t *testing.T) {
	cases := []struct {
		b    string
		want bool
	}{
		{"新", true},
		{"虎", true},
		{"A", true},
		{"1", true},
		{"", false},
		{"AB", false},
		{"新番", false},
		{"😀", false},
	}
	for _, c := range cases {
		if got := ezfyBannerValid(c.b); got != c.want {
			t.Errorf("番号 %q 校验 = %v，期望 %v", c.b, got, c.want)
		}
	}
}

// TestBotBannerPool 人机随机番号：词库里每个候选都必须是合法的 1 个字。
func TestBotBannerPool(t *testing.T) {
	if len(ezfyBotBannerPool) == 0 {
		t.Fatal("人机番号词库不能为空")
	}
	for _, b := range ezfyBotBannerPool {
		if !ezfyBannerValid(b) {
			t.Errorf("人机番号 %q 不是合法的 1 个字", b)
		}
	}
	for i := 0; i < 200; i++ {
		if b := ezfyBotPickBanner(); !ezfyBannerValid(b) {
			t.Fatalf("随机取到的番号 %q 不合法", b)
		}
	}
}

// TestMapCellShowsBanner 地图格第一行要用城主的番号（默认 新），不是统一「城市」。
func TestMapCellShowsBanner(t *testing.T) {
	back, err := os.ReadFile("ezfy_order.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(back), `cell["banner"] = banner`) {
		t.Fatal("MapView 的城市格没有下发 banner（前端口只能显示默认新）")
	}
	vue := ezfyStripCommentLines(readFrontFile(t, "../../../../web/src/views/ezfy/Ezfy.vue"))
	if !strings.Contains(vue, "if (cell.area_type === 3) return cell.banner || '新'") {
		t.Fatal("地图格 cellText 没有改用番号（应 return cell.banner || '新'）")
	}
}
