package ezfy

// 战报入库开关（★ 2026-10-31 用户需求）
//
// 用户要求：「二战系统配置加个功能，就是各种类型战报是否入库，分开，默认都入库」。
// 场景：脚本/刷野地会产出海量「采集/运输/派遣」类战报，把 ezfy_report 撑大、未读角标永远闪；
// 运营可以按类型关掉入库，玩家军情里就不再出现这一类（不影响任何结算逻辑）。
//
// ★★ 唯一判定 `ezfyReportSaveEnabled(reportType)`，两个写入入口共用：
//   - `addReport`（玩家端单条写入）
//   - `envReport`（懒结算攒批写入）
//   ⚠️ 新加写战报的入口也必须过这里，否则开关对它无效。
//
// ★ 0 是**合法值**（= 不入库）→ 读取端不能用「<=0 回落默认」那套；
//   配置缓存没加载（启动早期）时一律**放行**，宁可多写也不能误吞战报。

// 战报大类（与 `EzfyReport.ReportType` 取值一致）
const (
	ezfyReportScout   = 1 // 侦查：侦查报告 / 侦查失败
	ezfyReportBattle  = 2 // 战斗：战斗报告 / 被掠夺报告 / 驻防战报 / 进攻受阻
	ezfyReportConquer = 3 // 征服：征服报告
	ezfyReportBeaten  = 4 // 被征服：被征服报告 / 驻防城市失守
	ezfyReportLogi    = 5 // 后勤：采集 / 运输 / 派遣 / 增援 / 建城 / 赎城 / 返航 / 城市状态
	ezfyReportSystem  = 6 // 系统：交易 / 军官升星 / 将领升级 / 叛离 / 计谋
)

// ezfyReportSaveName 战报大类的管理端名称（配置项 tooltip / 日志用）
func ezfyReportSaveName(kind int) string {
	switch kind {
	case ezfyReportScout:
		return "侦查"
	case ezfyReportBattle:
		return "战斗"
	case ezfyReportConquer:
		return "征服"
	case ezfyReportBeaten:
		return "被征服"
	case ezfyReportLogi:
		return "后勤"
	case ezfyReportSystem:
		return "系统"
	}
	return "未知"
}

// ezfyReportSaveEnabled 该类型战报是否入库（唯一判定）。
func ezfyReportSaveEnabled(reportType int) bool {
	if !ezfyCfg.ready() {
		return true // 配置未加载 → 默认入库，避免启动早期误吞战报
	}
	l := ezfyCfg.limit
	switch reportType {
	case ezfyReportScout:
		return l.ReportSaveScout != 0
	case ezfyReportBattle:
		return l.ReportSaveBattle != 0
	case ezfyReportConquer:
		return l.ReportSaveConquer != 0
	case ezfyReportBeaten:
		return l.ReportSaveBeaten != 0
	case ezfyReportLogi:
		return l.ReportSaveLogi != 0
	case ezfyReportSystem:
		return l.ReportSaveSystem != 0
	}
	return true // 未知类型一律入库（不认识的类型不该被静默丢掉）
}
