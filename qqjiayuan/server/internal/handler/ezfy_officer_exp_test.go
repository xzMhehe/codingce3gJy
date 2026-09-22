package handler

import (
	"os"
	"strings"
	"testing"
)

// TestOfficerBattleExpWinnerGetsMore —— 用户规则②「胜利的一方获取更多经验」。
func TestOfficerBattleExpWinnerGetsMore(t *testing.T) {
	cases := []int64{0, 1, 100, 999, 10000}
	for _, dead := range cases {
		win := ezfyOfficerBattleExp(dead, true)
		lose := ezfyOfficerBattleExp(dead, false)
		if win <= lose {
			t.Fatalf("击杀 %d 时，胜方经验(%d) 应严格大于败方(%d)", dead, win, lose)
		}
	}
}

// TestOfficerBattleExpBothSidesPositive —— 用户规则①「攻防都要能获取经验」。
//
// 即使一个敌人都没打死（击杀 0），败方也应拿到参战基数 30 —— 保证
// 「打输也有经验」而不是 0。
func TestOfficerBattleExpBothSidesPositive(t *testing.T) {
	if got := ezfyOfficerBattleExp(0, false); got <= 0 {
		t.Fatalf("败方击杀 0 也应拿到正经验，实际 %d", got)
	}
	if got := ezfyOfficerBattleExp(0, true); got <= 0 {
		t.Fatalf("胜方击杀 0 也应拿到正经验，实际 %d", got)
	}
	// 参战基数就是 30
	if got := ezfyOfficerBattleExp(0, false); got != 30 {
		t.Fatalf("击杀 0 的败方经验应为基数 30，实际 %d", got)
	}
	if got := ezfyOfficerBattleExp(0, true); got != 45 {
		t.Fatalf("击杀 0 的胜方经验应为 30*1.5=45，实际 %d", got)
	}
}

// TestOfficerBattleExpMonotonic —— 杀得越多经验越多（战功口径）。
func TestOfficerBattleExpMonotonic(t *testing.T) {
	prev := int64(-1)
	for _, dead := range []int64{0, 10, 100, 1000, 100000} {
		got := ezfyOfficerBattleExp(dead, false)
		if got <= prev {
			t.Fatalf("击杀 %d 时经验(%d) 未随击杀数递增(上一个 %d)", dead, got, prev)
		}
		prev = got
	}
}

// TestOfficerBattleExpNegativeDeadSafe —— 异常入参不得产生负数经验。
func TestOfficerBattleExpNegativeDeadSafe(t *testing.T) {
	if got := ezfyOfficerBattleExp(-999, false); got < 0 {
		t.Fatalf("负击杀数应被夹到 0，经验不得为负，实际 %d", got)
	}
}

// TestOfficerExpAwardedInBothBranches —— 静态源码断言：
// 攻方军官的经验写入**必须同时出现在胜、败两个分支里**。
//
// 这是 2026-09-21 修的 bug：原来 addOfficerExp(city, leadOfficer.ID, ...)
// 只写在 `if win {` 分支内，导致打败仗的部队回来军官经验一点不动。
// 这里用「抠函数体 + 统计出现次数」的方式守住这个回归。
func TestOfficerExpAwardedInBothBranches(t *testing.T) {
	body := readGoSourceFile(t, "ezfy_order.go")
	fn := extractGoFuncBody(t, body, "func (h *EzfyHandler) processArrive(")
	if fn == "" {
		t.Fatal("未找到 processArrive 函数体")
	}
	// 攻方军官经验写入（带 leadOfficer.ID 的那一次）应出现 2 次：胜 1 次 + 败 1 次
	n := strings.Count(fn, "h.addOfficerExp(city, leadOfficer.ID, atkExp)")
	if n < 2 {
		t.Fatalf("攻方军官经验写入应同时存在于胜/败两个分支，实际出现 %d 次", n)
	}
	// 守方军官经验也要写
	if !strings.Contains(fn, "h.addOfficerExp(target, cityGuard.ID, defExp)") {
		t.Fatal("守方城守军官经验写入缺失")
	}
	// 旧的「按自己战损算」写法不得作为**代码**复活
	// （注释里提到旧公式是正常的，所以要先剥掉注释行再判断）。
	if strings.Contains(stripGoCommentLines(fn), "myDead/10 + 50") {
		t.Fatal("检测到旧的攻方经验公式 myDead/10 + 50（应按击杀数 + 胜利加成）")
	}
}

// stripGoCommentLines 去掉整行的 // 注释，便于对「真实代码」做断言
// （注释里引用旧实现是正常且有益的，不该被判为回归）。
func stripGoCommentLines(src string) string {
	var b strings.Builder
	for _, line := range strings.Split(src, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "//") {
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

// readGoSourceFile 读同目录下的 .go 源文件内容。
func readGoSourceFile(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("读取 %s 失败: %v", name, err)
	}
	return string(b)
}

// extractGoFuncBody 从 src 中抠出以 sig 开头的那个函数的**函数体**（不含签名）。
// 用花括号配对，避免把函数后面的代码也算进来。
func extractGoFuncBody(t *testing.T, src, sig string) string {
	t.Helper()
	i := strings.Index(src, sig)
	if i < 0 {
		return ""
	}
	// 找到第一个 '{'
	open := strings.IndexByte(src[i:], '{')
	if open < 0 {
		return ""
	}
	open += i
	depth := 0
	for j := open; j < len(src); j++ {
		switch src[j] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return src[open : j+1]
			}
		}
	}
	return ""
}
