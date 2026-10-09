package ezfy

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
//
// ★ 2026-10-09 守方起始距离改为 300（在双方射程 500 内）：原来放 ezfyBattleStartDist(6000)，
//   要 30 个回合才够得着 —— 用「固定回合数」比较损失时前几回合根本没交火（损失恒 0）。
func ezfyDefBonusState(defDefBonus int) *ezfyBattleState {
	return &ezfyBattleState{
		Attackers: []*ezfyFightUnit{ezfyTestUnit("A1", "步兵", 0, 200)},
		Defenders: []*ezfyFightUnit{ezfyTestUnit("D1", "步兵", 300, 200)},
		DefEquip:  ezfyBattleBonus{Def: defDefBonus},
		Head:      []string{}, Actions: []string{},
	}
}

// TestDefenderDefenceBonusAppliesWhenAttacked —— 守方防御加成必须在「被攻击」时生效
//
// ★ 2026-10-09 修正测试本身（原来长期假红）：旧写法把两场都跑到**战斗结束**再比总损失，
// 但双方兵力相同 —— 加不加防御最终都会被全歼（总损失都是 200）→ 断言恒失败。
// 改为比较**固定回合数内**的损失：+100% 防御时同样的回合里损失必须更少。
func TestDefenderDefenceBonusAppliesWhenAttacked(t *testing.T) {
	const rounds = 2
	base := ezfyDefBonusState(0)
	withDef := ezfyDefBonusState(100)
	for i := 0; i < rounds && !base.Done; i++ {
		base.Step(nil, nil)
	}
	for i := 0; i < rounds && !withDef.Done; i++ {
		withDef.Step(nil, nil)
	}
	baseLoss := base.Defenders[0].initialCount - base.Defenders[0].count
	withLoss := withDef.Defenders[0].initialCount - withDef.Defenders[0].count
	t.Logf("前 %d 回合守方损失: 防御+0%% → %d ; 防御+100%% → %d", rounds, baseLoss, withLoss)

	if baseLoss <= 0 {
		t.Fatalf("基线异常：守方一点没挨打（损失 %d），测试前提不成立", baseLoss)
	}
	if withLoss >= baseLoss {
		t.Fatalf("守方防御加成没生效：+100%% 防御时损失 %d，未加防御时损失 %d（应该明显更少）",
			withLoss, baseLoss)
	}
}
