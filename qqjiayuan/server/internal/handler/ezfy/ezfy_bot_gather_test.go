package ezfy

import (
	"strings"
	"testing"
)

// TestBotAttackTopsUpGather 人机出征必须**自动补集结令**（★ 2026-10-30 用户规则）：
// 人机没有任何获取道具的途径，背包集结令常年 0 个 → 出征被司令部上限卡到几万兵，
// 「单次出征 30~40 万」形同虚设 → 经常带少量兵去送败仗。
// botAttack 出征前要把背包补满到「二战系统配置」的集结令单次上限（ezfyGatherMax），
// 并按需烧（need = 把出征兵力抬到 want 所需个数），而不是永远 0 个；
// 开关在「人机管理-编辑人机」的「自动补集结令」（默认开，b.GatherTopUp != 0 才补）。
func TestBotAttackTopsUpGather(t *testing.T) {
	src := strings.ReplaceAll(rawFile(t, "ezfy_bot.go"), "\r\n", "\n")
	i := strings.Index(src, "func (h *EzfyHandler) botAttack(")
	j := strings.Index(src, "// ============ 风控开关 ============")
	if i < 0 || j <= i {
		t.Fatal("找不到 botAttack 函数区间")
	}
	body := src[i:j]
	for _, want := range []string{
		// 开关：默认开，管理端可关
		"if ezfyMarchCapOn() && b.GatherTopUp != 0 {",
		// 补满到配置上限（读 ezfyGatherMax，管理端可调）
		"if have := h.itemCount(b.UserID, ezfyGatherItemID()); have < maxG {",
		"h.addItem(b.UserID, ezfyGatherItemID(), maxG-have)",
		// 按需使用：只烧把出征兵力抬到 want 所需的个数
		"if baseCap, _ := h.ezfyOrderTroopCap(city.ID, 0, leadName); want > baseCap {",
		"need = int((want - baseCap + per - 1) / per)",
		// 阶梯增兵：连续出征没打下来 → 兵力目标阶梯加码
		"want = ezfyBotSteppedWant(want, int64(b.AtkFailStreak))",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("botAttack 缺少集结令补发/按需使用/阶梯增兵逻辑（缺 %s）", want)
		}
	}
}

// TestBotSteppedWant 阶梯增兵口径：每多打一次没占下 → +50%，封顶 12 次（7 倍）。
func TestBotSteppedWant(t *testing.T) {
	base := int64(350000)
	cases := []struct {
		streak, want int64
	}{
		{0, 350000},   // 没失败过 → 原值
		{-3, 350000},  // 异常负值 → 原值
		{1, 525000},   // +50%
		{2, 700000},   // +100%
		{4, 1050000},  // +200%
		{12, 2450000}, // 封顶 7 倍
		{99, 2450000}, // 超过封顶仍按 12 次算
	}
	for _, c := range cases {
		if got := ezfyBotSteppedWant(base, c.streak); got != c.want {
			t.Fatalf("streak=%d: want=%d，期望 %d", c.streak, got, c.want)
		}
	}
}

// TestBotTrainFollowsSteppedWant 「不够就征兵」：训练目标要跟着阶梯后的出征兵力走，
// 否则打不动就永远停在固定征兵线（默认 50 万），出征数量也上不去。
func TestBotTrainFollowsSteppedWant(t *testing.T) {
	src := strings.ReplaceAll(rawFile(t, "ezfy_bot.go"), "\r\n", "\n")
	i := strings.Index(src, "func (h *EzfyHandler) botTrain(")
	j := strings.Index(src[i:], "// ezfyBotCanPay")
	if i < 0 || j <= 0 {
		t.Fatal("找不到 botTrain 函数区间")
	}
	j += i
	body := src[i:j]
	for _, want := range []string{
		"if br := h.ezfyBotOf(city.UserID); br != nil && br.AtkFailStreak > 0 {",
		"ezfyBotSteppedWant(ezfyBotAtkTroopOf(br.Level, br.AtkTroop), int64(br.AtkFailStreak))",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("botTrain 征兵线没有跟随阶梯出征兵力（缺 %s）", want)
		}
	}
}
