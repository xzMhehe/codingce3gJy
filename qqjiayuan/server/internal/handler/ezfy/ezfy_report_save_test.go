package ezfy

import (
	"strings"
	"testing"

	"qqjiayuan/server/internal/model"
)

// ============ 战报入库开关护栏（★ 2026-10-31 用户需求）============
//
// 用户要求：「二战系统配置加个功能，就是各种类型战报是否入库，分开，默认都入库；
//
//	注意不要有冗余页面提示，有提示就放到 i 里面」。
//
// 口径：
//   - 按 report_type 分 6 类：1侦查 2战斗 3征服 4被征服 5后勤 6系统；
//   - 1 = 入库（默认），0 = 不入库（不写 ezfy_report，军情里看不到、不占未读角标）；
//   - 两个写入入口（addReport 单条 / envReport 攒批）共用 ezfyReportSaveEnabled；
//   - 配置缓存没加载时一律放行（宁可多写，不能误吞）。

// TestReportSaveEnabled 纯判定：默认全入库、关掉的类型单独生效、未知类型仍入库。
func TestReportSaveEnabled(t *testing.T) {
	oldLoaded := ezfyCfg.loaded
	oldLim := ezfyCfg.limit
	defer func() {
		ezfyCfg.loaded = oldLoaded
		ezfyCfg.limit = oldLim
	}()

	// ① 配置未加载 → 全部放行
	ezfyCfg.loaded = false
	for k := 1; k <= 6; k++ {
		if !ezfyReportSaveEnabled(k) {
			t.Fatalf("配置未加载时 %s 类战报应默认入库", ezfyReportSaveName(k))
		}
	}

	// ② 全部默认入库（loaded 但字段为 1）
	ezfyCfg.loaded = true
	ezfyCfg.limit = modelLimitAllOn()
	for k := 1; k <= 6; k++ {
		if !ezfyReportSaveEnabled(k) {
			t.Fatalf("%s 类战报默认应入库", ezfyReportSaveName(k))
		}
	}

	// ③ 只关「后勤」→ 只有 5 不入库，其它不受影响（「分开」的关键）
	ezfyCfg.limit.ReportSaveLogi = 0
	if ezfyReportSaveEnabled(5) {
		t.Fatal("后勤类关掉后仍入库")
	}
	for _, k := range []int{1, 2, 3, 4, 6} {
		if !ezfyReportSaveEnabled(k) {
			t.Fatalf("关掉后勤不该影响 %s 类", ezfyReportSaveName(k))
		}
	}

	// ④ 全部关掉
	ezfyCfg.limit.ReportSaveScout, ezfyCfg.limit.ReportSaveBattle = 0, 0
	ezfyCfg.limit.ReportSaveConquer, ezfyCfg.limit.ReportSaveBeaten = 0, 0
	ezfyCfg.limit.ReportSaveSystem = 0
	for k := 1; k <= 6; k++ {
		if ezfyReportSaveEnabled(k) {
			t.Fatalf("%s 类关掉后仍入库", ezfyReportSaveName(k))
		}
	}

	// ⑤ 未知类型（未来新增）一律入库，不能被静默丢掉
	if !ezfyReportSaveEnabled(99) {
		t.Fatal("未知 report_type 应默认入库")
	}
}

// modelLimitAllOn 6 个开关全开（默认状态）。
func modelLimitAllOn() (l model.EzfyCfgLimit) {
	l.ReportSaveScout, l.ReportSaveBattle, l.ReportSaveConquer = 1, 1, 1
	l.ReportSaveBeaten, l.ReportSaveLogi, l.ReportSaveSystem = 1, 1, 1
	return l
}

// TestReportSaveBothEntrypoints 两个写入口都必须过开关（漏一个 = 开关对那条路径无效）。
func TestReportSaveBothEntrypoints(t *testing.T) {
	add := ezfyStripCommentLines(strings.ReplaceAll(
		ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) addReport("), "\r\n", "\n"))
	if !strings.Contains(add, "if !ezfyReportSaveEnabled(reportType)") {
		t.Fatalf("addReport 没有过战报入库开关：\n%s", add)
	}
	env := ezfyStripCommentLines(strings.ReplaceAll(
		ezfyFuncBody(t, "ezfy_settle_env.go", "func (h *EzfyHandler) envReport("), "\r\n", "\n"))
	if !strings.Contains(env, "if !ezfyReportSaveEnabled(reportType)") {
		t.Fatalf("envReport（攒批写入）没有过战报入库开关：\n%s", env)
	}
}

// TestReportSaveAdminPlumbing 管理端：字段收发 / 0 合法 / 补列 / 显式落库 / 前端开关与 tips。
func TestReportSaveAdminPlumbing(t *testing.T) {
	admin := strings.ReplaceAll(readFrontFile(t, "ezfy_admin_round9.go"), "\r\n", "\n")
	cols := []string{"report_save_scout", "report_save_battle", "report_save_conquer",
		"report_save_beaten", "report_save_logi", "report_save_system"}
	for _, c := range cols {
		if !strings.Contains(admin, `"`+c+`"`) {
			t.Fatalf("ezfy_admin_round9.go 缺少配置项 %q 的收发", c)
		}
		// ★ 0 合法（= 不入库）→ 保存时必须显式写，否则 0 会被 GORM default 吞掉
		if !strings.Contains(admin, `"`+c+`":`) {
			t.Fatalf("%q 没有显式写入保存 map（0 会被 GORM default 吞掉）", c)
		}
		// 补列（老库）
		if !strings.Contains(admin, `"ALTER TABLE ezfy_cfg_limit ADD COLUMN " + col + " int DEFAULT 1"`) {
			t.Fatalf("缺少 %q 的补列语句", c)
		}
	}
	// 只接受 0/1
	if !strings.Contains(admin, `"战报入库开关只能是 0(不入库) 或 1(入库)"`) {
		t.Fatal("缺少 0/1 合法性校验")
	}

	vue := strings.ReplaceAll(readFrontFile(t,
		"../../../../admin-web/src/components/admin/AdminEzfyBuildLimit.vue"), "\r\n", "\n")
	for _, c := range cols {
		if !strings.Contains(vue, c+": 1") {
			t.Fatalf("前端默认值缺少 %q（默认全部入库）", c)
		}
		if !strings.Contains(vue, "'"+c+"'") {
			t.Fatalf("前端 fieldGroups.switch 缺少 %q（保存时不会提交）", c)
		}
		if !strings.Contains(vue, c+":") {
			t.Fatalf("前端 tips 缺少 %q 的 i 悬浮说明", c)
		}
	}
	// 用户要求：说明只放 i 里，页面上不写冗余提示
	if !strings.Contains(vue, `reportSaveFields`) || !strings.Contains(vue, `tips[r.key]`) {
		t.Fatal("战报入库开关应走 reportSaveFields + i 悬浮说明")
	}
	if strings.Contains(vue, "关闭后该类战报不会写入") {
		t.Fatal("页面上不应出现冗余说明文案（说明放 i 悬浮里）")
	}
}
