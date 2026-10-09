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
//
// ★ 2026-10-09 同日追加：用户要求在「掠夺比例」后面**再加一行「回收比例」**，
// 这一行专指伤兵回收占比、由 ezfyHealPctLine 生成（不写在 battleStatsTail 里）。
// 所以本测试改成锁两件事：① battleStatsTail 里的战利品字段叫「掠夺比例」；
// ② 伤兵那一行在 ezfyHealPctLine 里、字面量是「回收比例」。
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
	// ★ 伤兵回收那一行不在 battleStatsTail 里（它拿不到伤兵数据）→ 由 ezfyHealPctLine 生成
	if !strings.Contains(src, `回收比例: %d%%`) {
		t.Fatal("缺 ezfyHealPctLine 的「回收比例」行（伤兵回收占比）")
	}
}

// TestHealPctLine 「回收比例」行 = **看战报这一方自己**的伤兵入营占比。
//
// ★ 2026-10-09 用户口径（两轮）：①「回收比例就是伤兵回收比例 总的 基础+科技+军官技能（入伤兵营的）」；
// ②「回收比例 谁看展示谁的」→ 攻方看的战报显示攻方的比例、守方看的显示守方的比例。
func TestHealPctLine(t *testing.T) {
	// 用户实测战报：攻方战损 20458 → 伤兵 6546（32%）；守方战损 12881 → 伤兵 5152（40%）
	atk := []ezfyUnitGroup{{TroopId: 16, Count: 20458}}
	def := []ezfyUnitGroup{{TroopId: 16, Count: 12881}}
	if got, want := ezfyHealPctLine(6546, atk), "\n回收比例: 32%"; got != want {
		t.Fatalf("攻方视角 = %q，期望 %q", got, want)
	}
	if got, want := ezfyHealPctLine(5152, def), "\n回收比例: 40%"; got != want {
		t.Fatalf("守方视角 = %q，期望 %q", got, want)
	}
	// 该方零战损（没死人）→ 不输出这一行
	if got := ezfyHealPctLine(0, nil); got != "" {
		t.Fatalf("零战损应为空串，实际 %q", got)
	}
	// 多兵种战损合计
	multi := []ezfyUnitGroup{{TroopId: 16, Count: 1000}, {TroopId: 1, Count: 1000}}
	if got, want := ezfyHealPctLine(400, multi), "\n回收比例: 20%"; got != want {
		t.Fatalf("多兵种合计 = %q，期望 %q", got, want)
	}
}

// TestDefSrcTxt 防御加成的来源括号（2026-10-09 用户要求「防御也整个括号() 能看出来哪来的」）。
func TestDefSrcTxt(t *testing.T) {
	// 满科技、无军官 → 只有科技段
	got := ezfyDefSrcTxt([]ezfyBonusItem{
		{Name: "科技·装甲科技", Value: 30},
		{Name: "科技·掩体防御", Value: 20},
		{Name: "科技·重工技术", Value: 20},
	})
	if want := "(科技+70)"; got != want {
		t.Fatalf("科技段 = %q，期望 %q", got, want)
	}
	// 含城墙/军官/装备 → 按来源归类，顺序固定
	got = ezfyDefSrcTxt([]ezfyBonusItem{
		{Name: "城墙", Value: 45},
		{Name: "科技·装甲科技", Value: 30},
		{Name: "军官·冥王", Value: 20},
		{Name: "军官技能·弧形防御", Value: 60},
		{Name: "装备套装", Value: 15},
	})
	if want := "(城墙+45 军官+80 科技+30 装备套装+15)"; got != want {
		t.Fatalf("全来源 = %q，期望 %q", got, want)
	}
	// 全零 / 空 → 空串（老战场快照没有明细时不破坏格式）
	if got := ezfyDefSrcTxt([]ezfyBonusItem{{Name: "科技·装甲科技", Value: 0}}); got != "" {
		t.Fatalf("全零应为空串，实际 %q", got)
	}
	if got := ezfyDefSrcTxt(nil); got != "" {
		t.Fatalf("nil 应为空串，实际 %q", got)
	}
}

// TestBonusLineShowsDefSource 「战斗加成」行里防御也要带来源括号。
//
// ★★ 2026-10-09 用户要求「防御也整个括号() 能看出来哪来的」：
//
//	原来攻击有 `(军官+0 科技+70)`、防御只有裸数字。
//	满科技、无军官时两边防御都应是「防御+70%(科技+70)」（守方不再叠加城墙）。
func TestBonusLineShowsDefSource(t *testing.T) {
	defer stubCritTroops(t)()
	techBreak := []ezfyBonusItem{
		{Name: "科技·装甲科技", Value: 30},
		{Name: "科技·掩体防御", Value: 20},
		{Name: "科技·重工技术", Value: 20},
	}
	st := ezfyNewBattleState(
		[]ezfyUnitGroup{{TroopId: 11, Count: 100}},
		[]ezfyUnitGroup{{TroopId: 11, Count: 100}},
		70, 70, 70, 0, 0,
		30, 30,
		ezfyBattleBonus{}, ezfyBattleBonus{},
		ezfyBattleBonus{}, ezfyBattleBonus{}, "", "",
		"", "", 0, 0, 0, 0,
		nil, nil, nil, nil, techBreak,
		70, techBreak,
		map[int]int{}, map[int]int{}, map[int]int{}, map[int]int{},
		0, 0, 1, 2,
		ezfyTypeBonus{Speed: map[int]int{}}, ezfyTypeBonus{Speed: map[int]int{}})

	lines := bonusLinesOf(st.Head)
	if len(lines) != 2 {
		t.Fatalf("「战斗加成」应为攻/守两行，实际 %d 行：%v", len(lines), lines)
	}
	for _, ln := range lines {
		if !strings.Contains(ln, "防御+70%(科技+70)") {
			t.Fatalf("防御加成应带来源括号「防御+70%%(科技+70)」：\n%s", ln)
		}
	}
}
