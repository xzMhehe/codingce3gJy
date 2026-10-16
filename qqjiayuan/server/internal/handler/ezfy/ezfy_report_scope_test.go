package ezfy

import (
	"strings"
	"testing"
)

// ============ 战报「全部城市」口径守卫（2026-10-16）============
//
// 用户要求：「二战用户端 战斗报告查询改成查全部，别限制当前城市」。
//
// 规则：军情警讯(category=1) 与 战斗报告(category=2) 的**列表 / 徽标 / 一键删除**都按
// 「全部城市」口径 —— 玩家名下所有城的战报一并展示；city_id 只留给页面级
// 「雷达/侦察/情报等级」兜底（军情警讯另有 ezfyFillReportIntel 的逐条权威口径）。
//
// 顺带守住两条历史事故红线（别再被改回去）：
//   - 列表**不得**再出现「按城市过滤 + 老行(city_id=0/NULL) 反查订单」的 EXISTS 路径
//     —— 2026-10-11 线上 33s 超时的源头（逐行跑 2 个 EXISTS → 超过前端 15s 超时 → 两个 tab 全空）；
//   - 分类条件必须**下推 SQL** —— 2026-10-11「战斗报告(N) 却显示 (暂无战斗报告)」，
//     当时是「先取最新 200 条再在 Go 里过滤」，采集报告刷屏就把战斗报告挤出窗口。

// TestReportListAllCitiesScope 列表与徽标都必须是「全账号」口径，且不含城市过滤 / EXISTS 补查。
func TestReportListAllCitiesScope(t *testing.T) {
	body := ezfyStripCommentLines(strings.ReplaceAll(
		ezfyFuncBody(t, "ezfy_api.go", "func (h *EzfyHandler) Reports("), "\r\n", "\n"))

	// ① 列表不再按城市过滤
	if strings.Contains(body, `Where("city_id = ?"`) || strings.Contains(body, "queryCityId") {
		t.Fatalf("Reports 仍在按当前城过滤战报（用户要求：改成查全部）：\n%s", body)
	}
	// ② 老行 EXISTS 补查必须已删除（33s 超时源头）
	if strings.Contains(body, "EXISTS") {
		t.Fatalf("Reports 又出现了 EXISTS 补查（历史 33s 超时源头，全城口径下不需要）：\n%s", body)
	}
	// ③ 分类条件仍要下推 SQL（否则「徽标有数字、列表恒空」会复现）
	if !strings.Contains(body, "catCond := ezfyReportCategoryCond(category)") ||
		!strings.Contains(body, "q = q.Where(catCond)") {
		t.Fatalf("Reports 的分类条件没有下推到 SQL（LIMIT 必须作用在筛完分类之后）：\n%s", body)
	}

	// ④ 徽标统计同口径：全账号**单条**聚合，不再有按城过滤 / 老行补查
	cnt := ezfyStripCommentLines(strings.ReplaceAll(
		ezfyFuncBody(t, "ezfy_api.go", "func (h *EzfyHandler) ezfyReportCounts("), "\r\n", "\n"))
	if strings.Contains(cnt, "EXISTS") || strings.Contains(cnt, "city_id = ?") {
		t.Fatalf("ezfyReportCounts 又按城市过滤/补查了（徽标必须与「全城」列表同口径）：\n%s", cnt)
	}
	if !strings.Contains(cnt, "h.DB.Raw(sumSQL, uid)") {
		t.Fatalf("ezfyReportCounts 应只跑一条单表聚合 SQL：\n%s", cnt)
	}
}

// TestReportClearAllCitiesFromUI 一键删除必须与「全城列表」一致 —— 前端两个分区都传 city_id=0。
//
// 否则列表是全城的、删除只删当前城 → 「删完还剩一堆」。
func TestReportClearAllCitiesFromUI(t *testing.T) {
	vue := strings.ReplaceAll(readFrontFile(t, "../../../../web/src/views/ezfy/Ezfy.vue"), "\r\n", "\n")
	i := strings.Index(vue, "async doClearReports ()")
	if i < 0 {
		t.Fatal("找不到 doClearReports（Ezfy.vue 结构变了？）")
	}
	seg := vue[i:]
	if j := strings.Index(seg, "\n    delReport ("); j > 0 {
		seg = seg[:j]
	}
	seg = ezfyStripCommentLines(seg)
	if !strings.Contains(seg, "city_id: 0") {
		t.Fatalf("doClearReports 必须传 city_id: 0（全部城市），否则「删完列表还剩一堆」：\n%s", seg)
	}
	if strings.Contains(seg, "this.city ? this.city.id : 0") {
		t.Fatalf("doClearReports 还在按当前城删除：\n%s", seg)
	}
}

// ezfyStripCommentLines 去掉整行 `//` 注释（守卫断言只看代码，不看注释里的历史说明）。
func ezfyStripCommentLines(src string) string {
	var b strings.Builder
	for _, l := range strings.Split(src, "\n") {
		if strings.HasPrefix(strings.TrimSpace(l), "//") {
			continue
		}
		b.WriteString(l)
		b.WriteString("\n")
	}
	return b.String()
}
