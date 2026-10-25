package ezfy

import (
	"os"
	"strings"
	"testing"
)

// ============ 活动目标「兜底被俘次数」守卫（★ 2026-10-25 用户需求）============
//
// 用户要求：「二战 地图管理 活动野地 和活动目标配置，都加个兜底被俘次数，
// 就是至少打多少次才有可能被俘虏，默认 0，就是直接概率被俘虏；**注意不要展示到用户端**」。
//
// 口径（唯一实现 ezfyActCaptureAllowed）：
//   - min_capture_wins = 0 → 第一次打赢就按 CaptureRate 概率判（原行为，默认值）；
//   - min_capture_wins = N → 前 N-1 次打赢必定俘不到；第 N 次起才按概率判。
//   - 次数 = **打赢次数**（战败/平局不算）—— 俘虏只在胜利时判定，按胜场计不会被 1 兵碰瓷刷次数。
//
// 本文件守住三条：
//  1. 判定函数行为正确（0 = 无门槛；N = 第 N 次才放开）；
//  2. 结算里**确实**用了它，且未达标时静默不俘（不许往战报里写门槛）；
//  3. 字段只走管理端：两个配置接口都要收发，**用户端一个都不许出现**。

// TestActCaptureAllowed 兜底次数判定：0 = 直接按概率；N = 第 N 次才放开。
func TestActCaptureAllowed(t *testing.T) {
	if !ezfyActCaptureAllowed(0, 0) {
		t.Fatal("兜底次数 0（默认）必须是「第一次打赢就按概率」，不能被拦")
	}
	for wins := int64(0); wins < 3; wins++ {
		if ezfyActCaptureAllowed(3, wins) {
			t.Fatalf("兜底 3 次时，第 %d 次打赢不该放开俘虏", wins+1)
		}
	}
	if !ezfyActCaptureAllowed(3, 3) {
		t.Fatal("兜底 3 次时，第 3 次打赢应当放开俘虏")
	}
	if !ezfyActCaptureAllowed(3, 99) {
		t.Fatal("已远超兜底次数时应一直放开")
	}
	// 负数按 0 处理（不设门槛），与「默认 0」语义一致
	if !ezfyActCaptureAllowed(-1, 0) {
		t.Fatal("兜底次数为负应按 0（无门槛）处理")
	}
}

// TestActCaptureGateInBattle 结算必须用统一判定，且未达标时**静默**（不写战报）。
func TestActCaptureGateInBattle(t *testing.T) {
	body := ezfyStripCommentLines(strings.ReplaceAll(
		ezfyFuncBody(t, "ezfy_activity_target.go", "func (h *EzfyHandler) processActivityBattle("), "\r\n", "\n"))

	if !strings.Contains(body, "ezfyActCaptureAllowed(defCfg.MinCaptureWins, wins)") {
		t.Fatalf("俘虏判定没有走 ezfyActCaptureAllowed（口径必须唯一实现）：\n%s", body)
	}
	// 未达标 → rate 归 0，且必须在 `if rate > 0 {` 之前
	iGate := strings.Index(body, "ezfyActCaptureAllowed(")
	iCreate := strings.Index(body, "createCaptiveOfficer(")
	if iGate < 0 || iCreate < 0 || iGate > iCreate {
		t.Fatal("兜底判定必须在 createCaptiveOfficer 之前（否则未达标也会发俘虏）")
	}
	// ★ 用户要求「不要展示到用户端」：门槛不许写进战报
	for _, line := range strings.Split(body, "\n") {
		if !strings.Contains(line, "report") {
			continue
		}
		if strings.Contains(line, "兜底") || strings.Contains(line, "MinCaptureWins") ||
			strings.Contains(line, "次才") {
			t.Fatalf("兜底被俘次数不许写进玩家战报：%s", strings.TrimSpace(line))
		}
	}
	// 胜场必须真的累加（否则门槛永远停在 0）
	if !strings.Contains(body, "ezfyActCaptureWinAdd(uid, g.ID)") {
		t.Fatal("结算里没有累加「打赢该守将」次数（ezfyActCaptureWinAdd）")
	}
}

// TestActCaptureFieldAdminOnly 字段只走管理端：两个配置页收发，用户端一律不出现。
func TestActCaptureFieldAdminOnly(t *testing.T) {
	// ① 管理端两个接口都要能存、能读
	for _, f := range []string{"ezfy_admin_actwild.go", "ezfy_admin_actlevel.go"} {
		src := strings.ReplaceAll(readFrontFile(t, f), "\r\n", "\n")
		if !strings.Contains(src, `"min_capture_wins": "int"`) {
			t.Fatalf("%s 没有把 min_capture_wins 加进可写字段白名单（保存会丢值）", f)
		}
		if !strings.Contains(src, `"min_capture_wins"`) {
			t.Fatalf("%s 缺少 min_capture_wins", f)
		}
		if !strings.Contains(src, `vals["min_capture_wins"] = 0`) {
			t.Fatalf("%s 缺少默认值 0（未填时必须回落到「直接按概率」）", f)
		}
	}
	// OnConflict 的 DoUpdates 必须带上，否则「已存在的坐标/等级」保存后不更新
	wild := strings.ReplaceAll(readFrontFile(t, "ezfy_admin_actwild.go"), "\r\n", "\n")
	if !strings.Contains(wild, `"def_level", "captive_level", "def_move_dir", "min_capture_wins",`) {
		t.Fatal("活动野地 upsert 的 DoUpdates 列表漏了 min_capture_wins")
	}
	lvl := strings.ReplaceAll(readFrontFile(t, "ezfy_admin_actlevel.go"), "\r\n", "\n")
	if !strings.Contains(lvl, `"def_level", "captive_level", "def_move_dir", "min_capture_wins",`) {
		t.Fatal("活动目标配置 upsert 的 DoUpdates 列表漏了 min_capture_wins")
	}

	// ② ★★ 用户端一个都不许出现：只允许这三处使用（两个管理端配置接口 + 战斗结算内部）
	allowed := map[string]bool{
		"ezfy_admin_actwild.go":   true,
		"ezfy_admin_actlevel.go":  true,
		"ezfy_activity_target.go": true, // 配置结构体 + 结算判定（服务端内部，不下发）
	}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("读目录失败: %v", err)
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") || allowed[name] {
			continue
		}
		b, err := os.ReadFile(name)
		if err != nil {
			continue
		}
		if strings.Contains(string(b), "min_capture_wins") || strings.Contains(string(b), "MinCaptureWins") {
			t.Fatalf("%s 里出现了 min_capture_wins —— 这个门槛只允许管理端收发（用户要求不要展示到用户端）", name)
		}
	}
}
