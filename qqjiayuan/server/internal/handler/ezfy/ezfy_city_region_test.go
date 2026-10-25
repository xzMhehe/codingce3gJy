package ezfy

import (
	"os"
	"strings"
	"testing"
)

// ============ 城市状态页「洲名 / 海城」口径守卫（2026-10-25）============
//
// 用户反馈（原话）：
//
//	「城市: 再战辉煌基地(250,11) 大海 [沿海城市]」→「城市: …(250,11) 北冰洋 [海城]」
//
// 两个问题都在**源头口径**，不是单页文案：
//  1. /view 下发的 `continent` 用的是 `ezfyContinentName` —— 它只认陆地大洲，
//     海洋坐标一律回落成「大海」；应该用 `ezfyRegionName`（陆地给洲名、海洋给大洋名）。
//  2. `cityKind()` 还返回长文案「沿海城市 / 内陆城市」，用户要短的「海城 / 陆城」
//     （与建城页图例 2026-10-24 的简化保持一致）。

// TestRegionNameForUserCity —— 用户报的那座城 (250,11) 必须显示「北冰洋」而不是「大海」
func TestRegionNameForUserCity(t *testing.T) {
	if got := ezfyRegionName(250, 11); got != "北冰洋" {
		t.Fatalf("坐标(250,11) 的所属区域应为「北冰洋」，实际 %q", got)
	}
	// 海洋坐标绝不能再出现「大海」（那是 ezfyContinentName 对海洋的兜底值）
	for _, p := range [][2]int{{250, 11}, {5, 500}, {470, 200}, {100, 300}} {
		if got := ezfyRegionName(p[0], p[1]); got == "大海" || got == "" {
			t.Errorf("坐标(%d,%d) 的所属区域不该是 %q", p[0], p[1], got)
		}
	}
	// 陆地坐标仍然给洲名（不是海洋名）
	if got := ezfyRegionName(120, 120); got == "" {
		t.Error("陆地坐标也该有洲名")
	}
}

// TestViewUsesRegionNameAndShortCityKind —— 静态断言：/view 用 ezfyRegionName，cityKind 用短文案
func TestViewUsesRegionNameAndShortCityKind(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy.go", "func (h *EzfyHandler) cityKind(")
	if !strings.Contains(body, `"海城"`) || !strings.Contains(body, `"陆城"`) {
		t.Fatalf("cityKind 应返回短文案「海城/陆城」，实际函数体：%s", body)
	}

	// 直接扫整份 ezfy.go：View() 太长，ezfyFuncBody 会在中间某个 `}` 处截断
	b, err := os.ReadFile("ezfy.go")
	if err != nil {
		t.Fatalf("读 ezfy.go 失败: %v", err)
	}
	src := string(b)
	if !strings.Contains(src, `"continent": ezfyRegionName(city.X, city.Y)`) {
		t.Fatal("/view 的 continent 必须用 ezfyRegionName（陆地洲名 + 海洋大洋名），不能用 ezfyContinentName")
	}
	if strings.Contains(src, `"continent": ezfyContinentName(city.X, city.Y)`) {
		t.Fatal("/view 不该再用 ezfyContinentName —— 它对海洋只返回「大海」")
	}
}
