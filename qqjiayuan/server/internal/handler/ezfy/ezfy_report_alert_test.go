package ezfy

import (
	"strings"
	"testing"
)

// TestReportAlertCondPassiveOnly 顶部导航「军情」感叹号角标的统计口径（2026-10-13 用户规则）：
//
//	「有未读的军情警讯 和 未读的**被**侦查/被掠夺/被征服战报 才提示」——
//	只算**被动挨打**类；自己主动出击的战果（侦查/掠夺/战斗/征服报告）不计入，
//	否则「掠夺报告: 寇城N级」这类刷野地战报动辄几千条未读，角标会永远闪、提示失去意义。
func TestReportAlertCondPassiveOnly(t *testing.T) {
	cond := ezfyReportAlertCond()
	// 必须包含：军情警讯(cat1) 全部 + 战斗报告里的被袭类
	for _, s := range []string{
		"军情警报", "被侦查报告", "城市归还", "将领叛离", "野地丢失",
		"被掠夺报告", "被征服报告", "城破报告",
	} {
		if !strings.Contains(cond, s) {
			t.Fatalf("角标口径漏了「%s」：\n%s", s, cond)
		}
	}
	// 主动战果不得混入（写成带引号的前缀，避免「被掠夺报告」误命中「掠夺报告」子串）
	for _, s := range []string{"'侦查报告%'", "'掠夺报告%'", "'战斗报告%'", "'征服报告%'"} {
		if strings.Contains(cond, s) {
			t.Fatalf("角标口径混入了主动战报 %s（用户要求只提示被动挨打类）：\n%s", s, cond)
		}
	}
}

// TestUnreadAlertSingleSource 角标口径只有一处实现：
//
//	/view（ezfy.go）与 /reports/counts（ezfy_api.go）都必须走 `ezfyUnreadAlertCount`，
//	不能再各写一份标题条件 —— 否则两处口径漂移（「页面有角标、进页却清零」这类怪象）。
//	★ 同时禁止 /view 回退到 ezfyReportCatCondAny（那是列表/徽标的全量口径）。
func TestUnreadAlertSingleSource(t *testing.T) {
	for _, f := range []string{"ezfy.go", "ezfy_api.go"} {
		src := strings.ReplaceAll(rawFile(t, f), "\r\n", "\n")
		if !strings.Contains(src, "ezfyUnreadAlertCount(") {
			t.Fatalf("%s 没走 ezfyUnreadAlertCount（角标口径必须单一实现）", f)
		}
	}
	view := strings.ReplaceAll(rawFile(t, "ezfy.go"), "\r\n", "\n")
	if strings.Contains(view, "is_read = 0 AND \"+ezfyReportCatCondAny()") {
		t.Fatal("/view 的角标仍在用 ezfyReportCatCondAny（全量口径）→ 主动战报会让角标永远闪")
	}
}
