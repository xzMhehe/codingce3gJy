package ezfy

import (
	"strings"
	"testing"
)

// TestCorpsReportViewAndTab 军团战报三件事（2026-10-09 用户反馈）。
//
//	① 「战斗报告(14) 数量问题」：军团战报接口的 counts 是**全团团员 PvP 战报总数**
//	   （不是自己的未读数），前端不得拿它更新徽标 —— 后端这里保证徽标接口独立可用。
//	② 「军团战报点具体战报进去会消失」：详情接口原来只认 `user_id = 自己`，
//	   而同团团员的战报点进去必然 404 → 前端降级成列表项（没有 content）→ 标题在、正文空白。
//	   现在放行同团团员的 PvP 战报。
//	③ 「没军团的不展示军团战报 tab」：需要在「进页/切分区必调」的 /reports/counts 里下发 has_corps。
func TestCorpsReportViewAndTab(t *testing.T) {
	api := strings.ReplaceAll(rawFile(t, "ezfy_api.go"), "\r\n", "\n")

	// ② 详情接口放行同团团员（PvP 口径与军团战报列表一致）
	for _, want := range []string{
		"func (h *EzfyHandler) ezfyReportViewableByCorps(",
		"h.ezfyReportViewableByCorps(uid, &r)",
		"myCorp := h.corpsOfUser(uid)",
		"o.TargetType != 3", // 只放行打玩家城的战报
	} {
		if !strings.Contains(api, want) {
			t.Fatalf("军团战报详情未放行同团团员（缺 %s）", want)
		}
	}

	// ③ has_corps 必须在 ReportCounts 里（loadReportCounts 是军情页必调接口）
	i := strings.Index(api, "func (h *EzfyHandler) ReportCounts(")
	j := strings.Index(api, "func (h *EzfyHandler) Reports(")
	if i < 0 || j <= i {
		t.Fatal("找不到 ReportCounts / Reports 函数区间")
	}
	if !strings.Contains(api[i:j], `"has_corps": h.corpsOfUser(uid) > 0`) {
		t.Fatalf("ReportCounts 没下发 has_corps（前端无法判断是否显示军团战报 tab）：\n%s", api[i:j])
	}

	// ① 军团战报列表的 counts 语义要有明确警示（避免又被当徽标用）
	k := strings.Index(api, "func (h *EzfyHandler) corpsReports(")
	if k < 0 {
		t.Fatal("找不到 corpsReports")
	}
	if !strings.Contains(api[k:], "**不是**自己的未读数") {
		t.Fatal("corpsReports 的 counts 语义没写清楚（它是全团总数、不是未读数）")
	}
}
