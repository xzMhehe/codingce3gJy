package ezfy

import (
	"strings"
	"testing"
)

// TestBatchCollectScopedToCurrentCity 一键采集 / 一键收获 必须只作用于**当前城市**。
//
// ★★ 2026-10-09 用户反馈「一键采集也是当前城市的采集空闲部队生效」：
//
//	原来只按 `user_id` 过滤 → 多城玩家会被连坐（别的城市的空闲驻军也被下达采集），
//	而军情→驻军 tab 是**按当前城**展示的（ReportDynamics 传 city_id），
//	玩家看到的列表里根本没有那几支部队，却提示「已对 N 支下达」；
//	「一键召回」早已按 `city_id` 收口，三件套口径应当一致。
func TestBatchCollectScopedToCurrentCity(t *testing.T) {
	src := strings.ReplaceAll(rawFile(t, "ezfy_collect.go"), "\r\n", "\n")

	seg := func(from, to string) string {
		i := strings.Index(src, from)
		j := strings.Index(src, to)
		if i < 0 || j <= i {
			t.Fatalf("找不到函数区间：%s … %s", from, to)
		}
		return src[i:j]
	}

	collectAll := seg("func (h *EzfyHandler) CollectAll(", "func (h *EzfyHandler) StartCollect(")
	for _, want := range []string{"h.currentCity(uid)", "city_id = ?"} {
		if !strings.Contains(collectAll, want) {
			t.Fatalf("一键采集没按当前城市收口（缺 %s）：\n%s", want, collectAll)
		}
	}

	harvestAll := seg("func (h *EzfyHandler) HarvestAll(", "func (h *EzfyHandler) RecallAll(")
	for _, want := range []string{"h.currentCity(uid)", "city_id = ?"} {
		if !strings.Contains(harvestAll, want) {
			t.Fatalf("一键收获没按当前城市收口（缺 %s）：\n%s", want, harvestAll)
		}
	}
}
