package ezfy

// ★★ 2026-10-31 用户口径：连击 = **两次攻击**，每一次都要**单独计算暴击**，
// 不能共用同一次判定（主攻击那次在 Step 里掷过，连击追加的那次必须再掷一次）。
// 本文件盯三件事：
//  ① 连击追打真的会暴击，且播报/行内标记与主攻击同口径；
//  ② 连击的暴击与主攻击互不共用（Step 里满暴击时应有 2 条独立播报）；
//  ③ 暴击确实放大连击伤害（同样兵力下损失更多）。

import (
	"strings"
	"testing"
)

// TestComboDoubleCritSeparateFromMainAttack 连击的第二次普攻在 Step 里独立 roll 暴击：
// 满暴击时战报应出现「主攻击 1 条 + 连击 1 条」两条暴击播报，连击行还带自己的【暴击+N%】。
func TestComboDoubleCritSeparateFromMainAttack(t *testing.T) {
	build := func(crit int) *ezfyBattleState {
		st := &ezfyBattleState{
			Attackers: []*ezfyFightUnit{ezfyTestUnit("A1", "步兵", 0, 100)},
			Defenders: []*ezfyFightUnit{ezfyTestUnit("D1", "步兵", 100, 200)},
			AtkEquip:  ezfyBattleBonus{Crit: crit, CritDmg: 50},
			Head:      []string{},
			Actions:   []string{},
		}
		st.AtkCombos = comboRuntimes([]*ezfyComboSkill{{
			ID: 8, Name: "波次突击", Kind: ezfyComboDouble, Level: 10, Value: 100,
			ProcPct: 100, Rounds: 1,
		}})
		return st
	}

	// ① 满暴击：主攻击与连击各掷一次 → 2 条暴击播报
	st := build(100)
	st.Step(nil, nil)
	log := strings.Join(st.Actions, "\n")
	if got := strings.Count(log, "发动【暴击】"); got < 2 {
		t.Fatalf("主攻击与连击应各自独立 roll 暴击（至少 2 条播报），实际 %d 条：\n%s", got, log)
	}
	if !strings.Contains(log, "本次连击伤害+50%") {
		t.Fatalf("连击的暴击播报应写明「本次连击伤害」：\n%s", log)
	}
	if !strings.Contains(log, "发动组合技【波次突击】连击【暴击+50%】") {
		t.Fatalf("连击行没带自己的【暴击+N%%】标记：\n%s", log)
	}

	// ② 0% 暴击：主攻击与连击都不该出现暴击，但连击本身照常结算
	st0 := build(0)
	st0.Step(nil, nil)
	log0 := strings.Join(st0.Actions, "\n")
	if strings.Contains(log0, "暴击") {
		t.Fatalf("暴击几率 0 时不该出现暴击：\n%s", log0)
	}
	if !strings.Contains(log0, "发动组合技【波次突击】连击，") {
		t.Fatalf("连击本身仍应正常结算：\n%s", log0)
	}
}

// TestComboFollowUpOwnCritAmplifiesDamage 直接验连击追打：自己掷暴击、伤害真的变高。
func TestComboFollowUpOwnCritAmplifiesDamage(t *testing.T) {
	run := func(critChance int) (string, int64) {
		st := &ezfyBattleState{Actions: []string{}}
		unit := ezfyTestUnit("A1", "步兵", 0, 100)
		target := ezfyTestUnit("D1", "步兵", 100, 200)
		st.comboFollowUp(unit, target, []*ezfyFightUnit{target}, 500, 0, 0, 0, 0,
			"【攻方】", "【守方】", func(u *ezfyFightUnit) string { return u.cfg.Name },
			"波次突击", 100, "连击", false, critChance, 50)
		return strings.Join(st.Actions, "\n"), target.initialCount - target.count
	}

	logCrit, lossCrit := run(100)
	if !strings.Contains(logCrit, "发动【暴击】") {
		t.Fatalf("连击追打应能自己掷出暴击：\n%s", logCrit)
	}
	if !strings.Contains(logCrit, "连击【暴击+50%】") {
		t.Fatalf("连击追打行应带自己的暴击标记：\n%s", logCrit)
	}
	logNo, lossNo := run(0)
	if strings.Contains(logNo, "暴击") {
		t.Fatalf("暴击几率 0 时连击追打不该暴击：\n%s", logNo)
	}
	if lossCrit <= lossNo {
		t.Fatalf("连击暴击后损失应更多：暴击 %d vs 不暴击 %d", lossCrit, lossNo)
	}
}

// TestComboCritUsesSameFallback 连击暴击伤害为 0 时按基础倍率兜底（与主攻击同口径）。
func TestComboCritUsesSameFallback(t *testing.T) {
	st := &ezfyBattleState{Actions: []string{}}
	unit := ezfyTestUnit("A1", "步兵", 0, 100)
	target := ezfyTestUnit("D1", "步兵", 100, 200)
	st.comboFollowUp(unit, target, []*ezfyFightUnit{target}, 500, 0, 0, 0, 0,
		"【攻方】", "【守方】", func(u *ezfyFightUnit) string { return u.cfg.Name },
		"波次突击", 100, "连击", false, 100, 0)
	log := strings.Join(st.Actions, "\n")
	if !strings.Contains(log, "本次连击伤害+50%") {
		t.Fatalf("暴击伤害为 0 时应按基础 +50%% 兜底：\n%s", log)
	}
}
