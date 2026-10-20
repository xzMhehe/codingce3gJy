package ezfy

import (
	"strings"
	"testing"
)

// TestEzfyScoutInterceptDefenderText 防空拦截战报的守方视角：
// 攻方空战获胜时，守方侦察机被打光，不能再显示「击退敌军侦察」。
func TestEzfyScoutInterceptDefenderText(t *testing.T) {
	title, tail := ezfyScoutInterceptDefenderText(true)
	if title != "战斗报告: 防空拦截失败" {
		t.Fatalf("攻方胜时守方战报标题错误: %q", title)
	}
	if !strings.Contains(tail, "我方侦察机全部被击破") || !strings.Contains(tail, "突破拦截") {
		t.Fatalf("攻方胜时守方战报结尾错误: %q", tail)
	}

	title, tail = ezfyScoutInterceptDefenderText(false)
	if title != "战斗报告: 击退敌军侦察" {
		t.Fatalf("守方守住时战报标题错误: %q", title)
	}
	if !strings.Contains(tail, "敌军侦察机已被击破或击退") {
		t.Fatalf("守方守住时战报结尾错误: %q", tail)
	}
}
