package handler

import "testing"

// 守方防御加成是否真的生效（2026-09-23 排查战斗公式时发现的可疑点）
//
// 背景：`ezfyCalcDamage(baseAtk, def, count, atkBonus, defBonus)` 的第 5 个参数
// 是**被攻击方（target）的防御加成**（它作用在 target.cfg.Defence 上）：
//
//	bonusDef := def * (100 + defBonus) / 100
//
// 而 Step 里选参数时写的是：
//
//	unitAtkBonus := 0
//	unitDefBonus := defBonus   // 守方行动时
//	if isAtk {
//	    unitAtkBonus = atkBonus
//	    unitDefBonus = 0       // 攻方行动时
//	}
//
// 若这个赋值是反的，后果是：
//   - 「攻方打守方」时守方防御加成 = 0 → **城墙 / 城守 / 防御装备在被打时完全不起作用**；
//   - 「守方反击攻方」时反而把守方自己的防御加成算到了攻方头上。
//
// 本测试用客观数值判定：给守方 +100% 防御加成后，守方挨打受到的损失**必须变少**。

// ezfyDefBonusState 造一个「守方带 N% 防御加成」的战场（其余完全相同）
func ezfyDefBonusState(defDefBonus int) *ezfyBattleState {
	return &ezfyBattleState{
		Attackers: []*ezfyFightUnit{ezfyTestUnit("A1", "步兵", 0, 200)},
		Defenders: []*ezfyFightUnit{ezfyTestUnit("D1", "步兵", ezfyBattleStartDist, 200)},
		DefEquip:  ezfyBattleBonus{Def: defDefBonus},
		Head:      []string{}, Actions: []string{},
	}
}

// TestDefenderDefenceBonusAppliesWhenAttacked —— 守方防御加成必须在「被攻击」时生效
func TestDefenderDefenceBonusAppliesWhenAttacked(t *testing.T) {
	base := ezfyDefBonusState(0)
	withDef := ezfyDefBonusState(100)
	for !base.Done && base.Round < ezfyBattleMaxRounds {
		base.Step(nil, "")
	}
	for !withDef.Done && withDef.Round < ezfyBattleMaxRounds {
		withDef.Step(nil, "")
	}
	baseLoss := base.Defenders[0].initialCount - base.Defenders[0].count
	withLoss := withDef.Defenders[0].initialCount - withDef.Defenders[0].count
	t.Logf("守方损失: 防御+0%% → %d ; 防御+100%% → %d", baseLoss, withLoss)

	if baseLoss <= 0 {
		t.Fatalf("基线异常：守方一点没挨打（损失 %d），测试前提不成立", baseLoss)
	}
	if withLoss >= baseLoss {
		t.Fatalf("守方防御加成没生效：+100%% 防御时损失 %d，未加防御时损失 %d（应该明显更少）",
			withLoss, baseLoss)
	}
}
