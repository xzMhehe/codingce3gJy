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

// TestAlertReportTitleHasCoord 所有「被动挨打」类军情警讯的**标题**必须带城市坐标。
//
// ★ 2026-10-14 用户反馈：「被侦查报告: 新城市」这种只有名字，别的玩家有重名分不清 → 要加坐标。
//
//	军情警讯列表只显示标题（不点进去），所以重名城必须在标题里就能区分。
//	体例统一为 `xxx报告: 城市名(X,Y)`，与「被掠夺报告: X(286,294)」一致。
//
//	本测试用静态扫描锁定：
//	  ① 被侦查报告(ezfy_order.go) 标题必须含 %d,%d 坐标；
//	  ② 被掠夺/被征服报告(ezfy_order.go) 标题必须含 TargetX/TargetY；
//	  ③ 军情警报(来袭/已抵达) 标题必须含坐标。
func TestAlertReportTitleHasCoord(t *testing.T) {
	src := strings.ReplaceAll(rawFile(t, "ezfy_order.go"), "\r\n", "\n")
	for _, want := range []string{
		// 被侦查报告：侦查方城市坐标
		`被侦查报告: %s(%d,%d)`,
		// 被掠夺报告：被攻击城市坐标
		`"被掠夺报告: "+targetName+"("+strconv.Itoa(order.TargetX)+","+strconv.Itoa(order.TargetY)+")"`,
		// 被征服报告：被攻击城市坐标
		`"被征服报告: "+targetName+"("+strconv.Itoa(order.TargetX)+","+strconv.Itoa(order.TargetY)+")"`,
		// 军情警报「敌军来袭」：来源城市坐标
		`title += fmt.Sprintf(" 来自 %s(%d,%d)", city.Name, city.X, city.Y)`,
		// 军情警报「敌军已抵达」：来源城市坐标
		`"军情警报: 敌军已抵达 来自 %s(%d,%d)"`,
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("军情警讯标题缺坐标（重名城无法区分）：期望片段\n  %s\n未在 ezfy_order.go 中找到", want)
		}
	}
}
