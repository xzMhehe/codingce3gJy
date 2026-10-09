package ezfy

import (
	"strings"
	"testing"

	"qqjiayuan/server/internal/model"
)

// ============ 2026-10-09 排查「回合中移动距离还是兵种初始速度」 ============
//
// 用用户那份征服报告的真实参数（航母 vs 航母、燃烧引擎+20%、攻方套装移动距离+10%）复现：
//
//	攻方航母 前进 1105 = 850 ×(1 + 20%燃烧引擎 + 10%套装移动距离)
//	守方航母 前进 1020 = 850 ×(1 + 20%燃烧引擎)
//
// 结论：引擎**是**把速度加成算进移动的（喷气引擎是「空军速度」，航母是海军 → 不吃，
// 这是玩家以为「速度加成没生效」的真正原因，已改展示口径，见 TestBonusLine* ）。

// ezfyMoveSpeedState 造一个「航母 vs 航母」的战场（参数与用户征服报告一致）
func ezfyMoveSpeedState() *ezfyBattleState {
	return ezfyNewBattleState(
		[]ezfyUnitGroup{{TroopId: 16, Count: 204900}},
		[]ezfyUnitGroup{{TroopId: 16, Count: 200000}},
		634, 395, 595, 20, 20, // atkBonus, defBonus, defAtkBonus, atkSpeed(燃烧引擎), defSpeed
		30, 30,
		// 攻方套装全量：移动距离+10%
		ezfyBattleBonus{Dmg: 49, Def: 61, Hp: 73, Move: 10, Crit: 45, CritDmg: 64},
		ezfyBattleBonus{},
		ezfyBattleBonus{}, ezfyBattleBonus{}, "", "",
		"Michael·Adams Lv.150 攻击加成+365%", "斯大林 Lv.350 攻击加成+325%",
		515, 505, 150, 180,
		nil, nil, nil, nil, nil,
		425, nil,
		map[int]int{}, map[int]int{}, map[int]int{}, map[int]int{},
		5, 6, 1, 2,
		ezfyTypeBonus{Speed: map[int]int{ezfyTroopTypeAir: 30}},
		ezfyTypeBonus{Speed: map[int]int{ezfyTroopTypeAir: 30}})
}

func ezfyStubCarrier(t *testing.T) func() {
	t.Helper()
	saved := ezfyCfg.troops
	ezfyCfg.troops = map[int]model.EzfyCfgTroop{
		16: {ID: 16, Name: "航母", Type: 1, Health: 2200, AtkSea: 100, Defence: 100, Speed: 850, AttackRange: 3100},
	}
	return func() { ezfyCfg.troops = saved }
}

// TestMovementUsesSpeedBonus 回合里的「前进」距离 = 兵种速度 ×(1 + 移动速度加成)。
//
// 移动速度加成 = 科技(燃烧引擎) + 装备(移动距离) + 兵种专属(喷气引擎/军官速度技能)。
func TestMovementUsesSpeedBonus(t *testing.T) {
	defer ezfyStubCarrier(t)()
	st := ezfyMoveSpeedState()
	for _, h := range st.Head {
		if strings.Contains(h, "战斗加成") {
			t.Logf("战斗加成行：%s", h)
		}
	}
	st.Step(nil, nil)
	joined := strings.Join(st.Actions, "\n")
	t.Logf("第1回合：\n%s", joined)
	// 攻方航母：850 ×(1+20%燃烧+10%套装移动) = 1105
	if !strings.Contains(joined, "航母前进1105") {
		t.Fatalf("攻方航母前进距离未吃速度加成（期望 1105 = 850×1.30）：\n%s", joined)
	}
	// 守方航母：850 ×(1+20%燃烧) = 1020（守方没装备）
	if !strings.Contains(joined, "航母前进1020") {
		t.Fatalf("守方航母前进距离未吃速度加成（期望 1020 = 850×1.20）：\n%s", joined)
	}
}

// TestMovementAfterSnapshotRoundTrip 指挥室（打玩家城市走的就是这条）每回合「存快照 → 重建」，
// 重建后速度加成不能丢：前进距离必须仍是 1105。
func TestMovementAfterSnapshotRoundTrip(t *testing.T) {
	defer ezfyStubCarrier(t)()
	st := ezfyMoveSpeedState()

	snap := st.Snapshot()
	enc := ezfyBattleSnapshotEncode(snap)
	dec, ok := ezfyBattleSnapshotDecode(enc)
	if !ok {
		t.Fatal("快照编解码失败")
	}
	rebuilt := ezfyBattleStateFromSnapshot(dec)
	if rebuilt.AtkSpeedBonus != 20 || rebuilt.AtkEquip.Move != 10 {
		t.Fatalf("快照重建丢了速度加成：AtkSpeedBonus=%d AtkEquip.Move=%d",
			rebuilt.AtkSpeedBonus, rebuilt.AtkEquip.Move)
	}
	before := len(rebuilt.Actions)
	rebuilt.Step(nil, nil)
	joined := strings.Join(rebuilt.Actions[before:], "\n")
	if !strings.Contains(joined, "航母前进1105") {
		t.Fatalf("指挥室重建后前进距离未吃速度加成（期望 1105）：\n%s", joined)
	}
}
