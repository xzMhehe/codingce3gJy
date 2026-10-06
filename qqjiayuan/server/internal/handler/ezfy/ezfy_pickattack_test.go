package ezfy

import "testing"

// ★ 2026-10-06 用户反馈（点8）：「指挥模块战斗伤害计算不对 —— 对空应该是对空军的基础伤害、
// 对地是对陆军的基础伤害、对海是对海军的基础伤害，现在好像都用的对地」。
// ezfyPickAttack 按**守方兵种类型**选攻方的对应攻击属性：1海军→AtkSea、2陆军→AtkGround、
// 3空军→AtkAir、4城防→取对地/对海较大值。这组测试把全组合锁死，防止以后再回归成「都用对地」。

func TestPickAttackByDefType(t *testing.T) {
	atk := &ezfyTroopStats{AtkSea: 10, AtkGround: 20, AtkAir: 30, Defence: 0, Health: 0, Speed: 0, AttackRange: 0}
	cases := []struct {
		defType int
		want    int
	}{
		{1, 10}, // 守方海军 → 用对海 AtkSea
		{2, 20}, // 守方陆军 → 用对地 AtkGround
		{3, 30}, // 守方空军 → 用对空 AtkAir
		{4, 20}, // 城防 → 对地/对海较大值 max(20,10)=20
	}
	for _, c := range cases {
		def := &ezfyTroopStats{Type: c.defType}
		if got := ezfyPickAttack(atk, def); got != c.want {
			t.Fatalf("def.Type=%d 应选 %d，实际 %d（全用对地会得到 20，空军/海军场景会错）", c.defType, c.want, got)
		}
	}
}

// 城防取「对地/对海较大值」：对海高于对地时取对海
func TestPickAttackCityWallPrefersHigherSea(t *testing.T) {
	atk := &ezfyTroopStats{AtkSea: 99, AtkGround: 5, AtkAir: 1}
	def := &ezfyTroopStats{Type: 4}
	if got := ezfyPickAttack(atk, def); got != 99 {
		t.Fatalf("城防应取对海 99，实际 %d", got)
	}
}

// ★ 2026-10-06 用户反馈（点1）：「科技加成、技能加成没有算入伤害当中」。
// 伤害公式里 atkBonus/defBonus 确实参与运算（ezfyCalcDamage），这里锁定：
// 攻方加成越高伤害越高、守方防御加成越高伤害越低、最小伤害 = 数量×5。
func TestCalcDamageUsesBonuses(t *testing.T) {
	count := int64(100)
	base := 200

	// 攻方加成 +50% 比无加成伤害高
	d0 := ezfyCalcDamage(base, 10, count, 0, 0)
	d50 := ezfyCalcDamage(base, 10, count, 50, 0)
	if d50 <= d0 {
		t.Fatalf("攻方加成 50%% 应增加伤害：无加成 %d，+50%% %d", d0, d50)
	}
	// 守方防御加成 +50% 比无加成伤害低
	dDef50 := ezfyCalcDamage(base, 10, count, 0, 50)
	if dDef50 >= d0 {
		t.Fatalf("守方防御加成 50%% 应降低伤害：无加成 %d，守方+50%% %d", d0, dDef50)
	}
	// 最小伤害 = 数量×5
	floor := ezfyCalcDamage(1, 9999, count, 0, 9999)
	if floor != count*5 {
		t.Fatalf("最小伤害应为 数量×5=%d，实际 %d", count*5, floor)
	}
}
