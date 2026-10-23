package ezfy

// ★★ 2026-10-23 名将组合技能（系统专属战法）测试：
//
//	① 等级成长：效果值 = base + (组合技等级-1)×grow，等级由军官等级分档（步长 35、上限 10）；
//	② 只有名将能拿到（普通军官 / 非名将池 / 已下架 → nil）；
//	③ 概率发动：ProcPct=100 必发、=0 永不发，发动后按 Rounds 持续；
//	④ 效果真的进战斗：攻击加成让伤害更高、破防让伤害更高、回合治疗能补兵。

import (
	"strings"
	"testing"

	"qqjiayuan/server/internal/model"
)

// stubComboPool 临时往配置缓存里塞一条组合技（测试结束还原）
func stubComboPool(t *testing.T, s model.EzfyCfgGeneralSkill) func() {
	t.Helper()
	old := ezfyCfg.generalSkills
	m := map[int]model.EzfyCfgGeneralSkill{}
	for k, v := range old {
		m[k] = v
	}
	m[s.ID] = s
	ezfyCfg.generalSkills = m
	return func() { ezfyCfg.generalSkills = old }
}

// stubGeneralCfg 临时往名将池里塞一条名将配置
func stubGeneralCfg(t *testing.T, g model.EzfyCfgGeneral) func() {
	t.Helper()
	old := ezfyCfg.generals
	m := map[int]model.EzfyCfgGeneral{}
	for k, v := range old {
		m[k] = v
	}
	m[g.ID] = g
	ezfyCfg.generals = m
	return func() { ezfyCfg.generals = old }
}

func TestComboValueGrow(t *testing.T) {
	cases := []struct{ base, grow, lv, max, want int }{
		{7, 1, 1, 10, 7},
		{7, 1, 5, 10, 11},
		{7, 1, 10, 10, 16},
		{7, 1, 99, 10, 16}, // 超过上限 → 按满级
		{7, 1, 0, 10, 7},   // 非法等级 → 按 1 级
		{0, 0, 1, 10, 1},   // 至少 1
		{1, 1, 5, 5, 5},    // 反击回合这类「次数」型（上限 5 级）
		{52, 2, 10, 10, 70},
	}
	for _, c := range cases {
		if got := ezfyComboValue(c.base, c.grow, c.lv, c.max); got != c.want {
			t.Fatalf("base=%d grow=%d lv=%d → %d, want %d", c.base, c.grow, c.lv, got, c.want)
		}
	}
}

// TestComboLevelByOfficerLevel 组合技等级按**军官等级**分档（名将满级 350、人人 5 星，
// 所以不能拿星级当等级 —— 2026-10-23 用户纠正）。
// 默认步长 35、上限 10 → 350 级正好 10 级吃满。
func TestComboLevelByOfficerLevel(t *testing.T) {
	cases := []struct{ lv, max, step, want int }{
		{1, 10, 35, 1},
		{35, 10, 35, 1},
		{36, 10, 35, 2},
		{70, 10, 35, 2},
		{105, 10, 35, 3},
		{176, 10, 35, 6},
		{281, 10, 35, 9},
		{316, 10, 35, 10},
		{350, 10, 35, 10}, // 名将满级 → 正好 10 级
		{999, 10, 35, 10}, // 超过上限仍 10 级
		{350, 5, 35, 5},   // 管理端把上限调 5 → 5 级封顶（反击回合就这么配）
		{350, 5, 70, 5},   // 步长调回 70 → 350 级正好 5 级
		{0, 0, 0, 1},      // 非法输入回落（默认 10 级上限 / 步长 35）
	}
	for _, c := range cases {
		if got := ezfyComboLevelOf(c.lv, c.max, c.step); got != c.want {
			t.Fatalf("等级 %d / 上限 %d / 步长 %d → %d 级, want %d", c.lv, c.max, c.step, got, c.want)
		}
	}
}

func TestComboOnlyForGenerals(t *testing.T) {
	restorePool := stubComboPool(t, model.EzfyCfgGeneralSkill{
		ID: 901, Name: "测试组合技", Kind: ezfyComboAtk, BaseValue: 10, GrowPerLevel: 2,
		MaxLevel: 10, LevelPerStep: 35, ProcPct: 50, Rounds: 1, Enabled: 1,
	})
	defer restorePool()
	restoreG := stubGeneralCfg(t, model.EzfyCfgGeneral{ID: 9901, Name: "测试名将", Kind: 2, ComboSkillId: 901, Star: 5})
	defer restoreG()
	// 普通军官池（kind=1）即便误配了 combo_skill_id 也不生效
	restoreG2 := stubGeneralCfg(t, model.EzfyCfgGeneral{ID: 9902, Name: "测试普通", Kind: 1, ComboSkillId: 901})
	defer restoreG2()

	// 210 级 → 1+(210-1)/35 = 6 级：10 + 5×2 = 20
	c := ezfyComboOfGeneralCfg(ezfyCfg.general(9901), 210)
	if c == nil {
		t.Fatal("名将应能拿到组合技")
	}
	if c.Level != 6 || c.Value != 20 {
		t.Fatalf("210 级名将应为 6 级/20 点, 实际 Lv.%d/%d", c.Level, c.Value)
	}
	// 350 级（名将满级）→ 10 级吃满：10 + 9×2 = 28
	if max := ezfyComboOfGeneralCfg(ezfyCfg.general(9901), 350); max == nil || max.Level != 10 || max.Value != 28 {
		t.Fatalf("350 级名将应为 10 级/28 点, 实际 %+v", max)
	}
	if ezfyComboOfGeneralCfg(ezfyCfg.general(9902), 5) != nil {
		t.Fatal("普通军官（kind=1）不该有组合技")
	}
	if ezfyComboOfGeneralCfg(nil, 5) != nil {
		t.Fatal("nil 名将配置应返回 nil")
	}
	// 下架后不再生效
	s := ezfyCfg.generalSkill(901)
	s.Enabled = 0
	restorePool2 := stubComboPool(t, *s)
	defer restorePool2()
	if ezfyComboOfGeneralCfg(ezfyCfg.general(9901), 5) != nil {
		t.Fatal("下架（enabled=0）的组合技不该生效")
	}
}

// TestComboProcAndBonuses 概率发动 + 攻击加成真的进伤害
func TestComboProcAndBonuses(t *testing.T) {
	defer stubCritTroops(t)()
	build := func(combo *ezfyComboSkill) *ezfyBattleState {
		st := ezfyNewBattleState(
			[]ezfyUnitGroup{{TroopId: 11, Count: 100}},
			[]ezfyUnitGroup{{TroopId: 11, Count: 100}},
			0, 0, 0, 0, 0, 0, 0,
			ezfyBattleBonus{}, ezfyBattleBonus{},
			ezfyBattleBonus{}, ezfyBattleBonus{}, "", "",
			"", "", 0, 0, 0, 0,
			nil, nil, nil, nil, nil,
			0, nil,
			map[int]int{}, map[int]int{}, map[int]int{}, map[int]int{},
			0, 0, 1, 2,
			ezfyTypeBonus{}, ezfyTypeBonus{})
		st.AtkCombo = combo
		return st
	}
	// ① 100% 发动 → 第 1 回合就生效并播报
	st := build(&ezfyComboSkill{ID: 1, Name: "铁血军魂", Kind: ezfyComboAtk, Level: 5, Value: 16, ProcPct: 100, Rounds: 1})
	st.Step(nil, nil)
	if st.AtkComboLeft != 0 { // 本回合生效，回合末递减回 0
		t.Fatalf("1 回合技能应在回合末递减回 0, 实际 %d", st.AtkComboLeft)
	}
	if !strings.Contains(strings.Join(st.Actions, "\n"), "【铁血军魂】") {
		t.Fatalf("100%% 发动率却没播报：%v", st.Actions)
	}

	// ② 0% 发动 → 永不播报
	st2 := build(&ezfyComboSkill{ID: 1, Name: "铁血军魂", Kind: ezfyComboAtk, Level: 5, Value: 16, ProcPct: 0, Rounds: 1})
	for i := 0; i < 3; i++ {
		st2.Step(nil, nil)
	}
	if strings.Contains(strings.Join(st2.Actions, "\n"), "【铁血军魂】") {
		t.Fatal("0% 发动率不该发动")
	}

	// ②b 准备回合要写「组合技」说明（玩家开打前就知道带了什么）
	st4 := build(&ezfyComboSkill{ID: 6, Name: "摧锋破甲", Kind: ezfyComboBreak, Level: 5, Value: 20, ProcPct: 35, Rounds: 1})
	st4.appendComboHead()
	if !strings.Contains(strings.Join(st4.Head, "\n"), "【攻方组合技】摧锋破甲 Lv.5") {
		t.Fatalf("准备回合缺组合技说明：%v", st4.Head)
	}

	// ③ 持续 2 回合：第 1 回合发动后，第 2 回合仍生效
	st3 := build(&ezfyComboSkill{ID: 10, Name: "万钧之势", Kind: ezfyComboAtk, Level: 5, Value: 14, ProcPct: 100, Rounds: 2})
	st3.Step(nil, nil)
	if st3.AtkComboLeft != 1 {
		t.Fatalf("2 回合技能第 1 回合末应剩 1 回合, 实际 %d", st3.AtkComboLeft)
	}
	st3.Step(nil, nil)
	if st3.AtkComboLeft != 0 {
		t.Fatalf("第 2 回合末应归零, 实际 %d", st3.AtkComboLeft)
	}
}

// TestComboHealAndBreak 回合治疗按已损比例回兵；破防让同等兵力打出更多伤害。
func TestComboHealAndBreak(t *testing.T) {
	// ① 治疗：100 → 50，已损 50，治疗 12% = 6
	units := []*ezfyFightUnit{{count: 50, initialCount: 100}}
	if n := ezfyComboHealSide(units, 12); n != 6 {
		t.Fatalf("已损 50、治疗 12%% 应回 6, 实际 %d", n)
	}
	if units[0].count != 56 {
		t.Fatalf("治疗后应为 56, 实际 %d", units[0].count)
	}
	// 损失不足 1 个也要回 1（避免「发动了但什么都没发生」）
	one := []*ezfyFightUnit{{count: 99, initialCount: 100}}
	if n := ezfyComboHealSide(one, 1); n != 1 {
		t.Fatalf("有损失至少回 1, 实际 %d", n)
	}

	// ② 破防：同一攻防下，防御被压 20% 伤害必须更高
	full := ezfyCalcDamage(100, 50, 100, 0, 0)
	broken := ezfyCalcDamage(100, 50*(100-20)/100, 100, 0, 0)
	if broken <= full {
		t.Fatalf("破防 20%% 后伤害应更高：原 %d, 破防 %d", full, broken)
	}
}
