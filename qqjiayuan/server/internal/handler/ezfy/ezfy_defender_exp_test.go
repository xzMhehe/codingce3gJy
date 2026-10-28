package ezfy

import (
	"strings"
	"testing"
)

// TestDefenderOfficerExpOnCityAttack 守方军官经验（用户 2026-10-28）：
//
//	「城防守城的时候也有经验，现在玩家反馈没有，就是被征服、被掠夺」
//
// 线上实况（本地 122 条守方战报统计）：94 条**完全没有军官行** —— 玩家没任命城守就没有经验
// （用户口径：**没城守就不给经验**，不做兜底）；另外「征服失败（民心尚存）」分支直接 return，
// 守方连「被征服报告」都没有，经验在库里发了也看不到。
//
// 断言（缺一条就回退成「守方没经验」）：
//  1. 守方军官**只认城守**（position=2），不做「守军主将」兜底；
//  2. 守方经验写**实际到账**值（满级为 0），战报不再写假数字；
//  3. 「征服失败（民心尚存）」也要给守方写被征服报告 + 参战声望；
//  4. 战报只在到账 > 0 时写「军官经验+N」（满级到账 0 → 不写，用户要求简约、不要冗余文案）。
func TestDefenderOfficerExpOnCityAttack(t *testing.T) {
	order := readGoSourceFile(t, "ezfy_order.go")
	officer := readGoSourceFile(t, "ezfy_officer.go")

	// ① 守方军官 = 城守（用户明确：没城守就不给经验，不要兜底）
	if !strings.Contains(order, "defExpOfficer := cityGuard") {
		t.Fatal("守方军官没有直接取城守（defExpOfficer := cityGuard）")
	}
	if strings.Contains(order, "defenseLeadOfficer") {
		t.Fatal("守方经验又出现了「守军主将」兜底（用户口径：没城守就不给经验）")
	}
	if !strings.Contains(order, "h.addOfficerExp(target, defExpOfficer.ID, raw)") {
		t.Fatal("守方军官经验没有写到 defExpOfficer（城守）")
	}

	// ② 经验写实际到账值：addOfficerExp 必须返回 int64，满级返回 0
	if !strings.Contains(officer, "func (h *EzfyHandler) addOfficerExp(city *model.EzfyCity, officerId uint, exp int64) int64 {") {
		t.Fatal("addOfficerExp 没有返回实际到账经验（满级会写假数字）")
	}
	if !strings.Contains(order, "defExp = h.addOfficerExp(target, defExpOfficer.ID, raw)") {
		t.Fatal("守方经验没有取 addOfficerExp 的实际到账值")
	}

	// ③ 征服失败（民心尚存）也要给守方写被征服报告
	fn := extractGoFuncBody(t, order, "func (h *EzfyHandler) processArrive(")
	if fn == "" {
		t.Fatal("未找到 processArrive")
	}
	iFail := strings.Index(fn, "民心尚存, 城市未被征服")
	if iFail < 0 {
		t.Fatal("征服失败（民心尚存）分支没有给守方写「被征服报告」")
	}
	// 抬头句之后紧跟「写守方战报 + 参战声望」（窗口取 1600 字，足够覆盖该分支）
	win := fn[iFail:]
	if len(win) > 1600 {
		win = win[:1600]
	}
	if !strings.Contains(win, "h.addReport(target.UserID, 4,") {
		t.Fatal("征服失败分支里没有写守方战报（addReport(target.UserID, 4, ...)）")
	}
	if !strings.Contains(win, "h.addPrestige(target.UserID, 100)") {
		t.Fatal("征服失败分支没有给守方参战声望 +100")
	}

	// ④ 经验行只写**实际到账**值；满级（到账 0）干脆不写这一行 —— 用户要求「简约、别写冗余」
	if !strings.Contains(order, `b += fmt.Sprintf("\n军官经验+%d", s.DefExp)`) {
		t.Fatal("守方战报没有按实际到账经验输出「军官经验+N」")
	}
	if strings.Contains(order, "已满级") {
		t.Fatal("战报里出现了「已满级」这类冗余文案（用户要求简约，满级就不写经验行）")
	}
	// 攻方也不能再写公式值（满级会显示 +N 但经验不涨）
	if strings.Contains(fn, `fmt.Sprintf("\n军官经验+%d", atkExp)`) {
		t.Fatal("攻方战报还在用公式值写经验（满级会显示 +N 但经验不涨）")
	}
	if !strings.Contains(fn, "if eff := h.addOfficerExp(city, leadOfficer.ID, atkExp); eff > 0 {") {
		t.Fatal("攻方战报没有按实际到账经验输出")
	}
}
