package ezfy

import (
	"strings"
	"testing"
)

// TestWoundedReportLine 战报「攻方伤兵入营」行要把**占比 + 修复率加成**写清楚。
//
// ★★ 2026-10-09 用户反馈「pvp 伤兵回收比例应该有问题」——用真实战报核对过，数值是对的：
//
//	攻方战损 10697；航母 `repair_rate = 20`（兵种基础修复率 20%）
//	+ 满级「治愈伤兵」科技 10 级 ×2%/级 = +20% → 实际按 40% 回收 → 10697×40% = 4278 ✓
//
// 玩家以为算错，是因为战报只给一个数字、而兵种详情页写的是 20%。这里把占比与加成写出来对账。
func TestWoundedReportLine(t *testing.T) {
	losses := []ezfyUnitGroup{{TroopId: 16, Count: 10697}}

	got := ezfyWoundedReportLine(4278, losses, 20)
	want := "\n攻方伤兵入营: 4278(占攻方战损 10697 的 40%，修复率额外加成+20%)"
	if got != want {
		t.Fatalf("伤兵行 = %q，期望 %q", got, want)
	}
	// 没有科技/技能加成 → 只有兵种修复率那部分
	if got := ezfyWoundedReportLine(2139, losses, 0); got != "\n攻方伤兵入营: 2139(占攻方战损 10697 的 20%)" {
		t.Fatalf("无加成时伤兵行 = %q", got)
	}
	// 没有伤兵 → 不输出这一行
	if got := ezfyWoundedReportLine(0, losses, 20); got != "" {
		t.Fatalf("无伤兵时应为空串，实际 %q", got)
	}
	// 多兵种战损合计
	multi := []ezfyUnitGroup{{TroopId: 16, Count: 1000}, {TroopId: 1, Count: 1000}}
	if got := ezfyWoundedReportLine(400, multi, 0); !strings.Contains(got, "占攻方战损 2000 的 20%") {
		t.Fatalf("多兵种战损合计不对：%q", got)
	}
}

// TestBattleStatsTailUsesLootPctLabel 战报尾部的字段必须叫「掠夺比例」而不是「回收比例」。
//
// ★ 2026-10-09：「回收率」在这个游戏里是**伤兵回收率**（机械改造：回收率+10%），
// 而战报尾部这个字段是**掠夺比例**（战利品占目标资源的比例，平局就是 0%）——
// 撞词会让玩家以为「伤兵回收 0%」。
func TestBattleStatsTailUsesLootPctLabel(t *testing.T) {
	src := strings.ReplaceAll(rawFile(t, "ezfy_order.go"), "\r\n", "\n")
	i := strings.Index(src, "func (h *EzfyHandler) battleStatsTail(")
	if i < 0 {
		t.Fatal("找不到 battleStatsTail")
	}
	body := src[i:]
	if j := strings.Index(body, "\n}\n"); j > 0 {
		body = body[:j]
	}
	if !strings.Contains(body, `掠夺比例:%d%%`) {
		t.Fatalf("战报尾部应写「掠夺比例」，实际：\n%s", body)
	}
	if strings.Contains(body, `回收比例`) {
		t.Fatalf("战报尾部不该再出现「回收比例」（与伤兵回收率撞词）：\n%s", body)
	}
}
