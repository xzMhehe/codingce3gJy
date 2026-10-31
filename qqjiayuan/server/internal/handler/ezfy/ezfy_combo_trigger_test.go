package ezfy

import (
	"strings"
	"testing"
)

// ============ 2026-10-31 四类战法（参考率土之滨）+ 连击/追击区分 ============
//
// 用户需求：按品质各新增 6 条组合技，并区分率土的四种战法类型：
//
//	被动(1)：必定发动，整场常驻
//	指挥(2)：战前（第 1 回合）掷一次骰，命中后整场/持续 N 回合
//	主动(3)：每回合开始掷骰（原行为）
//	追击(4)：每次普攻命中后掷骰，对被普攻目标追加战法伤害
//
// ★ 用户纠正「追击、连击不是一个意思」：
//   - 连击 = 本回合**多打一次普攻**（可换目标/可溢出，打完还能带出追击）
//   - 追击 = 普攻命中后的**战法伤害**，只打被普攻的那个目标，目标死了不再追

// comboTriggerState 造一个只有攻方带组合技的战场（双方同兵种，射程拉满，开局即可交火）
func comboTriggerState(t *testing.T, combo *ezfyComboSkill) *ezfyBattleState {
	t.Helper()
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
	st.AtkCombos = comboRuntimes([]*ezfyComboSkill{combo})
	// ★ 2026-10-31 默认初始距离改为「最大射程+余量」，开局不再贴脸 → 拉到接触距离再打
	ezfyCloseIn(st)
	return st
}

// TestComboTriggerPassive 被动：必定发动、整场常驻，只播报一次。
func TestComboTriggerPassive(t *testing.T) {
	defer stubCritTroops(t)()
	st := comboTriggerState(t, &ezfyComboSkill{
		ID: 11, Name: "战备动员", Kind: ezfyComboAtk, Level: 10, Value: 40,
		ProcPct: 0, Rounds: 0, TriggerType: ezfyComboTriggerPassive,
	})
	st.Step(nil, nil)
	if left := st.AtkCombos[0].Left; left <= 0 {
		t.Fatalf("被动技能第1回合就应生效（Left>0），实际 %d", left)
	}
	st.Step(nil, nil)
	if left := st.AtkCombos[0].Left; left <= 0 {
		t.Fatalf("被动技能应整场常驻（第2回合仍 Left>0），实际 %d", left)
	}
	// 只播报一次（第1回合），第2回合不再刷屏
	n := 0
	round := 0
	for _, ln := range st.Actions {
		if strings.HasPrefix(ln, "第") && strings.HasSuffix(ln, "回合:") {
			round++
			continue
		}
		if strings.Contains(ln, "发动组合技【战备动员】") {
			if round != 1 {
				t.Fatalf("被动技能只应在第1回合播报一次，实际出现在第%d回合：%s", round, ln)
			}
			n++
			if !strings.Contains(ln, "（被动·常驻）") {
				t.Fatalf("被动播报应标注「被动·常驻」：%s", ln)
			}
		}
	}
	if n != 1 {
		t.Fatalf("被动技能应只播报 1 次，实际 %d 次", n)
	}
	// 效果真的进了本回合加成（activeCombos 里能看到）
	if len(activeCombos(st.AtkCombos)) != 1 {
		t.Fatalf("被动技能应始终在生效列表里")
	}
}

// TestComboTriggerCommand 指挥：战前掷一次，命中整场；没命中就整场不生效。
func TestComboTriggerCommand(t *testing.T) {
	defer stubCritTroops(t)()
	// 100% → 整场生效
	st := comboTriggerState(t, &ezfyComboSkill{
		ID: 24, Name: "钳形攻势", Kind: ezfyComboAtk, Level: 10, Value: 180,
		ProcPct: 100, Rounds: 0, TriggerType: ezfyComboTriggerCommand,
	})
	st.Step(nil, nil)
	if left := st.AtkCombos[0].Left; left <= 0 {
		t.Fatalf("指挥 100%% 应在战前生效，实际 Left=%d", left)
	}
	st.Step(nil, nil)
	if left := st.AtkCombos[0].Left; left <= 0 {
		t.Fatalf("指挥 Rounds=0 应整场生效，实际 Left=%d", left)
	}
	if !strings.Contains(strings.Join(st.Actions, "\n"), "（指挥·战前生效）") {
		t.Fatalf("指挥发动应标注「指挥·战前生效」：%v", st.Actions)
	}
	// 0% → 整场不生效
	st2 := comboTriggerState(t, &ezfyComboSkill{
		ID: 24, Name: "钳形攻势", Kind: ezfyComboAtk, Level: 10, Value: 180,
		ProcPct: 0, Rounds: 0, TriggerType: ezfyComboTriggerCommand,
	})
	for i := 0; i < 2; i++ {
		st2.Step(nil, nil)
	}
	if left := st2.AtkCombos[0].Left; left != 0 {
		t.Fatalf("指挥没掷中就不该生效，实际 Left=%d", left)
	}
	if strings.Contains(strings.Join(st2.Actions, "\n"), "【天王星行动】") {
		t.Fatal("指挥没掷中不该播报")
	}
}

// TestComboTriggerPursuit 追击：每次普攻命中后掷骰，播报是「追击」不是「连击」。
func TestComboTriggerPursuit(t *testing.T) {
	defer stubCritTroops(t)()
	st := comboTriggerState(t, &ezfyComboSkill{
		ID: 27, Name: "乘胜追击", Kind: ezfyComboPursue, Level: 10, Value: 100,
		ProcPct: 100, Rounds: 1, TriggerType: ezfyComboTriggerPursuit,
	})
	st.Step(nil, nil)
	joined := strings.Join(st.Actions, "\n")
	if !strings.Contains(joined, "发动组合技【乘胜追击】追击") {
		t.Fatalf("追击技能应播报「追击」：%v", st.Actions)
	}
	if strings.Contains(joined, "【硫磺岛】连击") {
		t.Fatalf("追击不该被写成连击：%v", st.Actions)
	}
	// 追击是「每次攻击独立掷骰」，不占回合层 → 不进 activeCombos（不加成属性）
	if len(activeCombos(st.AtkCombos)) != 0 {
		t.Fatal("追击技能不该进回合生效列表（它不是常驻加成）")
	}
}

// TestComboDoubleVsPursuit 连击播报「连击」、追击播报「追击」，两者互不冒充。
func TestComboDoubleVsPursuit(t *testing.T) {
	defer stubCritTroops(t)()
	st := comboTriggerState(t, &ezfyComboSkill{
		ID: 22, Name: "梯次冲锋", Kind: ezfyComboDouble, Level: 10, Value: 180,
		ProcPct: 100, Rounds: 1, TriggerType: ezfyComboTriggerActive,
	})
	st.Step(nil, nil)
	joined := strings.Join(st.Actions, "\n")
	if !strings.Contains(joined, "发动组合技【梯次冲锋】连击") {
		t.Fatalf("连击技能应播报「连击」：%v", st.Actions)
	}
	if strings.Contains(joined, "【诺曼底登陆】追击") {
		t.Fatalf("连击不该被写成追击：%v", st.Actions)
	}
}

// TestComboPursuitOf 只挑「追击」类技能，并按效果值从高到低排（每次普攻只让最强的一条掷骰）。
func TestComboPursuitOf(t *testing.T) {
	rts := comboRuntimes([]*ezfyComboSkill{
		{ID: 1, Name: "装甲突击", Kind: ezfyComboAtk, Value: 180, TriggerType: ezfyComboTriggerActive},
		{ID: 16, Name: "侧翼伏击", Kind: ezfyComboPursue, Value: 70, TriggerType: ezfyComboTriggerPursuit},
		{ID: 27, Name: "乘胜追击", Kind: ezfyComboPursue, Value: 100, TriggerType: ezfyComboTriggerPursuit},
	})
	got := comboPursuitOf(rts)
	if len(got) != 2 {
		t.Fatalf("应只挑出 2 条追击技能，实际 %d", len(got))
	}
	if got[0].Name != "乘胜追击" || got[1].Name != "侧翼伏击" {
		t.Fatalf("追击技能应按效果值降序：%s / %s", got[0].Name, got[1].Name)
	}
	if len(comboPursuitOf(nil)) != 0 {
		t.Fatal("没有追击技能时应返回空")
	}
}
