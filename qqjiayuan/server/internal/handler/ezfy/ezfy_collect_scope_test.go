package ezfy

import (
	"strings"
	"testing"
)

// TestGarrisonBatchScopeDefaultsToCurrentCity 一键采集 / 一键收获 / 一键召回 的默认范围。
//
// ★★ 2026-10-09 用户反馈「一键采集只对当前城市的采集空闲部队生效」：
//
//	原来只按 `user_id` 过滤 → 多城玩家会被连坐（别的城市的空闲驻军也被下达采集），
//	而军情→驻军 tab 是**按当前城**展示的（ReportDynamics 传 city_id），
//	玩家看到的列表里根本没有那几支部队，却提示「已对 N 支下达」；
//	「一键召回」早已按 `city_id` 收口，三件套口径应当一致。
//
// ★★ 2026-10-22 用户规则：默认仍是当前城市，但二战系统配置可改成「全部城市」。
//
//	这里把两个分支都钉住：配置开关必须走 ezfyGarrisonBatchAllCities()；
//	currentCity / city_id 过滤必须放在 !allCities 分支里，不能无条件执行。
func TestGarrisonBatchScopeDefaultsToCurrentCity(t *testing.T) {
	src := strings.ReplaceAll(rawFile(t, "ezfy_collect.go"), "\r\n", "\n")

	seg := func(from, to string) string {
		i := strings.Index(src, from)
		j := strings.Index(src, to)
		if i < 0 || j <= i {
			t.Fatalf("找不到函数区间：%s … %s", from, to)
		}
		return src[i:j]
	}

	cases := []struct {
		name string
		fn   string
		want []string
	}{
		{
			name: "一键采集",
			fn:   seg("func (h *EzfyHandler) CollectAll(", "func (h *EzfyHandler) StartCollect("),
			want: []string{
				"allCities := ezfyGarrisonBatchAllCities()",
				"if !allCities",
				"current = h.currentCity(uid)",
				`q = q.Where("city_id = ?", current.ID)`,
				"h.batchCities(cityIDs)",            // 全部城市：一次 IN 取城判资源满
				"h.ezfyCityAtResMax(city)",          // 按城判定，不逐支查
				"h.ezfyAtResMax(int64(current.ID))", // 当前城市：仍只判一次
			},
		},
		{
			name: "一键收获",
			fn:   seg("func (h *EzfyHandler) HarvestAll(", "func (h *EzfyHandler) RecallAll("),
			want: []string{
				"allCities := ezfyGarrisonBatchAllCities()",
				"if !allCities",
				"current = h.currentCity(uid)",
				`q = q.Where("city_id = ?", current.ID)`,
				"h.newSettleEnv(orders)", // 多城也走批量预取
			},
		},
		{
			name: "一键召回",
			fn:   seg("func (h *EzfyHandler) RecallAll(", "func (h *EzfyHandler) StopCollect("),
			want: []string{
				"allCities := ezfyGarrisonBatchAllCities()",
				"if !allCities",
				"current = h.currentCity(uid)",
				`q = q.Where("city_id = ?", current.ID)`,
				"h.newSettleEnv(settleOrders)", // 全部城市：结算数据一次预取
				"h.envFlush(env)",              // 战报/宝物批量落库
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, want := range tc.want {
				if !strings.Contains(tc.fn, want) {
					t.Fatalf("%s缺少 %q：\n%s", tc.name, want, tc.fn)
				}
			}
		})
	}
}

// TestGarrisonBatchScopeConfigFallsBackToCurrentCity 老库 0 / 配置未加载时必须按当前城市。
func TestGarrisonBatchScopeConfigFallsBackToCurrentCity(t *testing.T) {
	helper := ezfyFuncBody(t, "ezfy_geo.go", "func ezfyGarrisonBatchAllCities(")
	for _, want := range []string{"!ezfyCfg.ready()", "return false", "== ezfyGarrisonBatchScopeAll"} {
		if !strings.Contains(helper, want) {
			t.Fatalf("驻军批量操作范围读取缺少 %q（老库 0 / 未加载必须回落当前城市）：\n%s", want, helper)
		}
	}
}
