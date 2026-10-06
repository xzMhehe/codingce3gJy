package ezfy

import (
	"os"
	"strings"
	"testing"

	"qqjiayuan/server/internal/model"
)

// ============ 军官技能随等级自动升级（2026-10-06） ============
//
// 规则(v2)：军官每满 30 级技能等级 +1（30级=1级…150级=5级、180级=6级）；
// 加成类统乘倍率；绝地反击 N 级 = 前 N 回合。普通军官最高 5 级、名将最高 6 级。
// 等级动态推导、不落库，学得晚自然按当前等级定级。

// TestEzfySkillLevelOf 技能等级边界（2026-10-06 v2：每 30 级 +1 级，普通军官 cap5 / 名将 cap6）。
func TestEzfySkillLevelOf(t *testing.T) {
	cases := []struct {
		level, cap, want int
	}{
		{0, 5, 1}, {1, 5, 1}, {29, 5, 1}, {30, 5, 1}, {59, 5, 1},
		{60, 5, 2}, {89, 5, 2}, {90, 5, 3}, {119, 5, 3},
		{120, 5, 4}, {149, 5, 4}, {150, 5, 5}, {179, 5, 5},
		// 普通军官最高 5 级：180 级仍封顶 5
		{180, 5, 5}, {999, 5, 5},
		// 名将最高 6 级：180 级正好 6 级（「前提达到 180 级」）
		{150, 6, 5}, {179, 6, 5}, {180, 6, 6}, {209, 6, 6}, {350, 6, 6},
	}
	for _, c := range cases {
		if got := ezfySkillLevelOf(c.level, c.cap); got != c.want {
			t.Fatalf("ezfySkillLevelOf(%d, %d) = %d, 期望 %d", c.level, c.cap, got, c.want)
		}
	}
	// nil 军官兜底 1 级
	h := &EzfyHandler{}
	if got := h.officerSkillLevel(nil); got != 1 {
		t.Fatalf("officerSkillLevel(nil) = %d, 期望 1", got)
	}
	if got := generalSkillLevel(nil); got != 1 {
		t.Fatalf("generalSkillLevel(nil) = %d, 期望 1", got)
	}
	// 守将（军官池）同口径：普通军官 kind=1 最高 5 级 / 名将 kind=2 最高 6 级
	g180 := &model.EzfyCfgGeneral{Level: 180, Kind: 2}
	if got := generalSkillLevel(g180); got != 6 {
		t.Fatalf("generalSkillLevel(名将180级) = %d, 期望 6", got)
	}
	g180p := &model.EzfyCfgGeneral{Level: 180, Kind: 1}
	if got := generalSkillLevel(g180p); got != 5 {
		t.Fatalf("generalSkillLevel(普通守将180级) = %d, 期望 5", got)
	}
	g50 := &model.EzfyCfgGeneral{Level: 50, Kind: 2}
	if got := generalSkillLevel(g50); got != 1 {
		t.Fatalf("generalSkillLevel(名将50级) = %d, 期望 1", got)
	}
}

// TestSkillBattleBonusScaled 攻击类技能加成必须乘倍率（尖兵突击/火炮控制/四指编队·狼群战术）。
func TestSkillBattleBonusScaled(t *testing.T) {
	src := rawFile(t, "ezfy_officer.go")
	for _, want := range []string{"30 * scale", "15 * scale", "h.officerSkillScale(o)"} {
		if !strings.Contains(src, want) {
			t.Fatalf("officerSkillBattleBonus 未按等级乘倍率（缺 %s）：\n%s", want, src)
		}
	}
}

// TestGuardBonusScaled 防御类技能加成必须乘倍率（弧形防御/弹幕支援），属性部分 officerGuardAttrBonus 不加倍。
func TestGuardBonusScaled(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_officer.go", "func (h *EzfyHandler) officerGuardBonus(")
	if !strings.Contains(body, "30 * scale") || !strings.Contains(body, "10 * scale") {
		t.Fatalf("officerGuardBonus 的技能部分未乘倍率：\n%s", body)
	}
	if !strings.Contains(body, "officerGuardAttrBonus") {
		t.Fatalf("officerGuardBonus 未保留属性部分（officerGuardAttrBonus）：\n%s", body)
	}
}

// TestSpeedSkillBonusScaled 速度类技能加成必须乘倍率（坦克突袭/闪电袭击/越岛战术）。
func TestSpeedSkillBonusScaled(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_officer.go", "func (h *EzfyHandler) officerSpeedSkillBonus(")
	if !strings.Contains(body, "10 * h.officerSkillScale(o)") {
		t.Fatalf("officerSpeedSkillBonus 未按等级乘倍率：\n%s", body)
	}
}

// TestLootHealScaled 黄金眼掠夺 / 机械改造恢复 全部乘倍率（ezfy_order.go ≥3 处、activity_target.go ≥1 处）。
func TestLootHealScaled(t *testing.T) {
	order := rawFile(t, "ezfy_order.go")
	if got := strings.Count(order, "10 * h.officerSkillScale(leadOfficer)"); got < 3 {
		t.Fatalf("ezfy_order.go 里黄金眼/机械改造乘倍率点只有 %d 处，期望 ≥3", got)
	}
	act := rawFile(t, "ezfy_activity_target.go")
	if got := strings.Count(act, "10 * h.officerSkillScale(leadOfficer)"); got < 1 {
		t.Fatalf("ezfy_activity_target.go 里机械改造乘倍率点只有 %d 处，期望 ≥1", got)
	}
}

// TestActWildDefScaled 活动野地守将加成乘倍率（弧形防御/弹幕支援/速度），属性部分不加倍。
func TestActWildDefScaled(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_activity_target.go", "func ezfyActWildDefBonus(")
	if !strings.Contains(body, "generalSkillScale(g)") {
		t.Fatalf("ezfyActWildDefBonus 未按守将等级取倍率：\n%s", body)
	}
	if !strings.Contains(body, "10 * scale") {
		t.Fatalf("ezfyActWildDefBonus 的速度/弹幕支援未乘倍率：\n%s", body)
	}
	if !strings.Contains(body, "ezfyAttrToBonus(g.Learning)") {
		t.Fatalf("ezfyActWildDefBonus 未保留守将属性部分（ezfyAttrToBonus）：\n%s", body)
	}
}

// TestCounterTriggerRounds 绝地反击按「前N回合」触发（st.Round <= counterRounds），不再是第1回合。
func TestCounterTriggerRounds(t *testing.T) {
	src := rawFile(t, "ezfy_battle.go")
	if !strings.Contains(src, "st.Round <= counterRounds") {
		t.Fatalf("绝地反击触发未按回合数判断（缺 st.Round <= counterRounds）")
	}
	if strings.Contains(src, "st.Round == 1") {
		t.Fatalf("仍有「第1回合」写法 st.Round == 1，未升级为前N回合")
	}
}

// TestCounterSnapshotRounds 快照需保存回合数字段，且老快照(bool)要能回退成前1回合。
func TestCounterSnapshotRounds(t *testing.T) {
	src := rawFile(t, "ezfy_battle.go")
	if !strings.Contains(src, `json:"atk_counter_rounds"`) {
		t.Fatalf("快照缺 atk_counter_rounds 字段")
	}
	if !strings.Contains(src, "snap.DefCounter") {
		t.Fatalf("快照重建缺老 bool 回退逻辑（snap.DefCounter）")
	}
	if !strings.Contains(src, "atkRounds = 1") {
		t.Fatalf("快照重建缺老快照回退成 1 回合（atkRounds = 1）")
	}
}

// TestSkillEffectTextAtShowsFinalValue 技能效果文案直接显示放大后的最终数值。
//
// 用户反馈（2026-10-06）：「尖兵突击(Lv.3 攻击力+30% (效果×3)) 别这么展示，
// 效果×3 太 low」—— 要求直接按等级算出最终数值展示（如 攻击力+90%）。
func TestSkillEffectTextAtShowsFinalValue(t *testing.T) {
	cases := []struct{ skill string; lv int; want string }{
		{"尖兵突击", 1, "攻击力+30%"},
		{"尖兵突击", 3, "攻击力+90%"},
		{"弧形防御", 2, "防御力+60%"},
		{"四指编队", 3, "空军对空攻击+45%"},
		{"弹幕支援", 3, "城防攻击范围+30%"},
		{"机械改造", 2, "回收率+20%, 出征油耗-20%"},
		{"黄金眼", 3, "侦查等级+3"},
		{"绝地反击", 2, "前2回合反击"},
	}
	for _, c := range cases {
		if got := ezfySkillEffectTextAt(c.skill, c.lv); got != c.want {
			t.Fatalf("ezfySkillEffectTextAt(%q, %d) = %q, 期望 %q", c.skill, c.lv, got, c.want)
		}
	}
	// 文案里不再出现「(效果×N)」后缀（只查函数体，避免注释干扰）
	body := ezfyFuncBody(t, "ezfy_officer.go", "func ezfySkillEffectTextAt(")
	if strings.Contains(body, "效果×") {
		t.Fatal("技能效果文案仍拼「(效果×N)」后缀，应直接展示最终数值")
	}
}

// ============ 辅助 ============

// rawFile 读取指定源码文件的全部内容（静态断言用）。
func rawFile(t *testing.T, file string) string {
	t.Helper()
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("读源码失败 %s: %v", file, err)
	}
	return string(b)
}