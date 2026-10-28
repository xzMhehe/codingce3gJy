package ezfy

import (
	"strings"
	"testing"
)

// ============ 活动目标「可抓次数」按玩家累计（★ 2026-10-28 用户规则）============
//
// 用户原话：「地图管理 → 活动野地 → 可抓次数如果配置了 1 次，是**针对玩家**的 ——
// 他抓住过一次了，那么他就不能再抓了；即使他抓的被其他玩家抢走了，这个抓住过的玩家
// 也不能再抢了。配置其他次数也一样逻辑。」
//
// 旧实现的问题：判定数的是「该玩家当前拥有的该守将军官实例数」——
//
//	军官被抢走（PvP 俘虏/策反）后实例数归零 → 抓过的人又能再抓一次，与配置语义不符。
//
// 现在的口径（唯一判定 ezfyActCaptureFull + 流水表 ezfy_act_capture_log）：
//   - 每成功抓到一次就插一条流水，**只增不删**（没有回退/清零的写法）；
//   - 已抓次数 = 该玩家该守将的流水条数；
//   - 上限 = 0 → 不限；>0 → 条数 >= 上限 就概率归 0；
//   - 老数据兜底：一条流水都没有时按「当前拥有的军官实例数」补记 kind=2 流水。
//
// 本文件守住五条：
//  1. 纯函数判定行为正确；
//  2. 结算确实走「流水条数」而不是「数当前实例数」；
//  3. 抓到成功后必须真的插一条流水（否则上限永远不生效）；
//  4. 流水只增：写入口只有 Create，没有 UPDATE/DELETE；
//  5. 新表在 skip 分支也会被建出来（否则流水写不进去 → 能无限抓）。

// TestActCaptureFull 上限判定：0 = 不限；N = 第 N 次抓满。
func TestActCaptureFull(t *testing.T) {
	if ezfyActCaptureFull(0, 0) || ezfyActCaptureFull(0, 99) {
		t.Fatal("可抓次数 0 必须是「不限」，不能判满")
	}
	if ezfyActCaptureFull(-1, 5) {
		t.Fatal("可抓次数为负应按 0（不限）处理")
	}
	if ezfyActCaptureFull(1, 0) {
		t.Fatal("上限 1 次、还没抓过 → 不该判满")
	}
	if !ezfyActCaptureFull(1, 1) {
		t.Fatal("上限 1 次、已抓 1 次 → 必须判满（★ 本次修复的核心：抓过就不能再抓）")
	}
	if !ezfyActCaptureFull(2, 3) {
		t.Fatal("已抓次数超过上限时必须判满")
	}
	if ezfyActCaptureFull(3, 2) || !ezfyActCaptureFull(3, 3) {
		t.Fatal("上限 3 次：第 2 次不该满，第 3 次必须满")
	}
}

// TestActCaptureUsesAccumulatedCount 结算必须用「累计流水条数」，不能数当前军官实例数。
func TestActCaptureUsesAccumulatedCount(t *testing.T) {
	body := ezfyStripCommentLines(strings.ReplaceAll(
		ezfyFuncBody(t, "ezfy_activity_target.go", "func (h *EzfyHandler) processActivityBattle("), "\r\n", "\n"))

	if !strings.Contains(body, "h.ezfyActCaptureUsed(uid, g.ID)") {
		t.Fatalf("结算没有用累计已抓次数（ezfyActCaptureUsed）：\n%s", body)
	}
	if !strings.Contains(body, "ezfyActCaptureFull(defCfg.MaxCapture, captures)") {
		t.Fatalf("上限判定没有走唯一实现 ezfyActCaptureFull")
	}
	// ★ 关键：不许再回到「在结算里数军官实例数」的老口径
	if strings.Contains(body, "Count(&had)") {
		t.Fatalf("结算里又出现了「数当前拥有的军官实例数」——被抢走会归零，抓过的人能再抓：\n%s", body)
	}
	// 抓到成功 → 必须记一条流水（带坐标/城市，便于追溯）
	if !strings.Contains(body, "h.ezfyActCaptureLogAdd(uid, g.ID, order.TargetX, order.TargetY,") {
		t.Fatal("抓到成功后没有记抓取流水（ezfyActCaptureLogAdd）")
	}
	iCreate := strings.Index(body, "createCaptiveOfficer(")
	iAdd := strings.Index(body, "ezfyActCaptureLogAdd(")
	if iCreate < 0 || iAdd < 0 || iAdd < iCreate {
		t.Fatal("流水必须在 createCaptiveOfficer 返回成功后写（否则没抓到也占次数）")
	}
}

// TestActCaptureLogOnlyGrows 流水只增：写入口只有 Create，且必须查 error。
func TestActCaptureLogOnlyGrows(t *testing.T) {
	src := strings.ReplaceAll(readFrontFile(t, "ezfy_activity_target.go"), "\r\n", "\n")

	if !strings.Contains(src, "func (h *EzfyHandler) ezfyActCaptureLogAdd(") {
		t.Fatal("缺少 ezfyActCaptureLogAdd（唯一写入口）")
	}
	i := strings.Index(src, "func (h *EzfyHandler) ezfyActCaptureLogAdd(")
	body := src[i:]
	if j := strings.Index(body[1:], "\nfunc "); j >= 0 {
		body = body[:j+1]
	}
	if !strings.Contains(body, "h.DB.Create(&model.EzfyActCaptureLog{") {
		t.Fatalf("流水写入必须走 Create（只插不改）：\n%s", body)
	}
	if strings.Contains(body, "Delete(") || strings.Contains(body, "Update(") ||
		strings.Contains(body, "Updates(") {
		t.Fatalf("流水写入里出现了 UPDATE/DELETE —— 流水必须只增不减：\n%s", body)
	}
	// 写入必须查 error：表没建出来时不能静默（静默 = 可抓次数恒 0 = 能无限抓）
	if !strings.Contains(body, ".Error; err != nil") {
		t.Fatal("流水 Create 没检查 error（表缺失时会静默失效 → 可无限抓）")
	}

	// 老数据兜底必须补记，否则「抓过又被抢走」的老玩家会回退
	if !strings.Contains(src, "model.EzfyActCaptureKindOfficer") {
		t.Fatal("缺少老数据兜底流水（KindOfficer）")
	}
	// 必须先走「已有流水」快路径，再兜底（兜底只在一条流水都没有时生效）
	if !strings.Contains(src, "if used > 0 {\n\t\treturn used\n\t}") {
		t.Fatal("ezfyActCaptureUsed 必须先走「已有流水条数」快路径，再兜底")
	}
}

// TestActCaptureLogTable 新表：model 定义 + 三处注册（AutoMigrate / 补列清单 / 幂等建表）。
func TestActCaptureLogTable(t *testing.T) {
	// ① model 有这张表、有玩家×守将联合索引
	mdl := strings.ReplaceAll(readFrontFile(t, "../../model/ezfy.go"), "\r\n", "\n")
	i := strings.Index(mdl, "type EzfyActCaptureLog struct {")
	if i < 0 {
		t.Fatal("缺少 EzfyActCaptureLog（抓取流水表）")
	}
	body := mdl[i:]
	if j := strings.Index(body, "\n}"); j > 0 {
		body = body[:j]
	}
	for _, f := range []string{"UserId", "GeneralId", "idx_act_cap_ug", "Kind"} {
		if !strings.Contains(body, f) {
			t.Fatalf("EzfyActCaptureLog 缺少 %s", f)
		}
	}
	if !strings.Contains(mdl, `return "ezfy_act_capture_log"`) {
		t.Fatal("EzfyActCaptureLog 的 TableName 不是 ezfy_act_capture_log")
	}

	// ② 进 AutoMigrate 列表（全新库 / 二战独立库靠它建表）
	if !strings.Contains(readFrontFile(t, "../../seed/ezfy_run.go"), "&model.EzfyActCaptureLog{}") {
		t.Fatal("ezfy_act_capture_log 不在 ezfyMigrateModels 里，全新库不会建表")
	}
	// ③ 进补列清单（半旧库补列）
	if !strings.Contains(readFrontFile(t, "../../seed/ezfy_schema.go"), "&model.EzfyActCaptureLog{}") {
		t.Fatal("ezfy_act_capture_log 不在 ezfySchemaModels 里，半旧库补不出列")
	}
	// ④ skip 分支幂等建表（EnsureEzfySchema 只补列不建表，新表必须单独建）
	if !strings.Contains(readFrontFile(t, "../../seed/ezfy_schema.go"),
		"func EnsureEzfyActCaptureLogTable(") {
		t.Fatal("缺少 EnsureEzfyActCaptureLogTable")
	}
	if !strings.Contains(readFrontFile(t, "../../seed/ezfy_run.go"), "EnsureEzfyActCaptureLogTable(db)") {
		t.Fatal("RunEzfy 没调用 EnsureEzfyActCaptureLogTable")
	}
	if !strings.Contains(readFrontFile(t, "../../../main.go"), "seed.EnsureEzfyActCaptureLogTable(homeDB)") {
		t.Fatal("main.go 的 seed.skip 分支没调用 EnsureEzfyActCaptureLogTable（单库模式会缺表）")
	}
	// ⑤ 存量抓取回填（否则「已开活动的区」管理端抓取记录一片空白 + 老玩家能多抓一次）
	sch := readFrontFile(t, "../../seed/ezfy_schema.go")
	if !strings.Contains(sch, "func EnsureEzfyActCaptureLogBackfill(") {
		t.Fatal("缺少 EnsureEzfyActCaptureLogBackfill（存量抓取回填）")
	}
	if !strings.Contains(readFrontFile(t, "../../seed/ezfy_run.go"), "EnsureEzfyActCaptureLogBackfill(db)") {
		t.Fatal("RunEzfy 没调用 EnsureEzfyActCaptureLogBackfill")
	}
	if !strings.Contains(readFrontFile(t, "../../../main.go"), "seed.EnsureEzfyActCaptureLogBackfill(homeDB)") {
		t.Fatal("main.go 的 seed.skip 分支没调用 EnsureEzfyActCaptureLogBackfill")
	}
	// 回填必须带一次性标记位，否则每次启动都重算
	if !strings.Contains(sch, "ezfy_act_capture_log_backfill_v2") {
		t.Fatal("回填没有一次性标记位（每次启动都会重跑）")
	}
	// ★ 2026-10-28 用户反馈「抓取信息不全」→ 回填必须**战报优先**（真实时间/坐标），
	//   且只认野地俘虏来源，PvP 抢来的/发放的/招募的都不算「抓过」。
	if !strings.Contains(sch, "俘虏敌将:") {
		t.Fatal("回填没有从战报还原真实抓取（信息会不全：时间同一秒、坐标 0）")
	}
	if !strings.Contains(sch, "source IN (?, ?)") ||
		!strings.Contains(sch, "model.EzfyOfficerSourceWildland") {
		t.Fatal("军官兜底没有只认「野地俘虏/历史未标注」来源（抢来/发放/招募的会被误算成抓过）")
	}
	if !strings.Contains(sch, "model.EzfyActCaptureKindReport") {
		t.Fatal("缺少「战报补记」来源标记")
	}
}

// TestActCaptureNoReportDependency 抓取记录**不许依赖战报**（★ 2026-10-28 用户要求）。
//
// 用户原话：「注意别用战报关联查询，战报只保留 3 天的我会定期删除」。
// 所以战报只能出现在**一次性历史回填**里，且必须限定时间窗口；下面三处一律不许出现：
//
//	① 运行时判定 ezfyActCaptureUsed / ezfyActCaptureLogAdd（玩家每打一次都会走）
//	② 管理端抓取记录接口 AdminEzfyActWildCaptures（列表页会反复查）
//	③ 结算 processActivityBattle（战斗高频路径）
func TestActCaptureNoReportDependency(t *testing.T) {
	for _, fn := range []string{
		"func (h *EzfyHandler) ezfyActCaptureUsed(",
		"func (h *EzfyHandler) ezfyActCaptureLogAdd(",
		"func (h *EzfyHandler) processActivityBattle(",
	} {
		body := ezfyStripCommentLines(strings.ReplaceAll(
			ezfyFuncBody(t, "ezfy_activity_target.go", fn), "\r\n", "\n"))
		if strings.Contains(body, "EzfyReport") || strings.Contains(body, "ezfy_report") {
			t.Fatalf("%s 里出现了战报查询 —— 战报只留 3 天会被删，抓取记录/可抓次数绝不能依赖它", fn)
		}
	}
	adm := ezfyStripCommentLines(strings.ReplaceAll(
		ezfyFuncBody(t, "ezfy_admin_actwild.go", "func (h *EzfyAdmin) AdminEzfyActWildCaptures("), "\r\n", "\n"))
	if strings.Contains(adm, "EzfyReport") || strings.Contains(adm, "ezfy_report") {
		t.Fatal("抓取记录接口里出现了战报查询 —— 列表必须只读 ezfy_act_capture_log")
	}
	// 回填（唯一允许读战报的地方）必须带时间窗口，否则 longtext 全表扫
	sch := readFrontFile(t, "../../seed/ezfy_schema.go")
	i := strings.Index(sch, "func EnsureEzfyActCaptureLogBackfill(")
	if i < 0 {
		t.Fatal("缺少 EnsureEzfyActCaptureLogBackfill")
	}
	body := sch[i:]
	if j := strings.Index(body[1:], "\nfunc "); j >= 0 {
		body = body[:j+1]
	}
	if !strings.Contains(body, "EzfyReport") {
		t.Fatal("回填没有读战报？历史抓取会信息不全（时间/坐标丢失）")
	}
	if !strings.Contains(body, "AddDate(0, 0, -7)") {
		t.Fatal("回填读战报没限定时间窗口（战报只留 3 天，7 天余量足够，避免全表扫）")
	}
}

func TestActCaptureAdminUI(t *testing.T) {
	// ① 路由
	if !strings.Contains(readFrontFile(t, "../../router/router.go"),
		`admin.GET("/ezfy-act-wilds/:id/captures"`) {
		t.Fatal("缺少管理端路由 GET /admin/ezfy-act-wilds/:id/captures")
	}
	// ② handler
	adm := readFrontFile(t, "ezfy_admin_actwild.go")
	if !strings.Contains(adm, "func (h *EzfyAdmin) AdminEzfyActWildCaptures(") {
		t.Fatal("缺少 AdminEzfyActWildCaptures")
	}
	// 按守将查（流水按玩家×守将存，与坐标无关）
	if !strings.Contains(adm, `Where("general_id = ?", a.OfficerId)`) {
		t.Fatal("抓取记录必须按 general_id 查（同一守将多坐标共用一份计数）")
	}
	// 未配守将 / 表未建 都要给友好提示，不能 500
	if !strings.Contains(adm, "该坐标没有配置守将") || !strings.Contains(adm, "尚未创建") {
		t.Fatal("缺少「未配守将 / 表未建」的兜底提示")
	}

	// ③ 前端：按钮 + 模态框 + 接口 + 分页
	ui := strings.ReplaceAll(readFrontFile(t,
		"../../../../admin-web/src/components/admin/AdminEzfyMap.vue"), "\r\n", "\n")
	if !strings.Contains(ui, `@click="openAwCaptures(row)"`) {
		t.Fatal("活动野地操作列没有「抓取记录」按钮")
	}
	if !strings.Contains(ui, `:visible.sync="awCapDlg"`) {
		t.Fatal("缺少抓取记录模态框（awCapDlg）")
	}
	if !strings.Contains(ui, "'/admin/ezfy-act-wilds/' + this.awCapRow.id + '/captures'") {
		t.Fatal("前端没有调用 /captures 接口")
	}
	if !strings.Contains(ui, "awCapTotal") || !strings.Contains(ui, "loadAwCaptures()") {
		t.Fatal("抓取记录模态框缺少分页（awCapTotal / loadAwCaptures）")
	}
}
