package ezfy

import (
	"strings"
	"testing"
)

// TestBattleHeadOrder 准备回合（Head）的展示顺序回归：
//
//	军官 → 科技 → 装备 → 套装 → **战斗加成(总计)** → 战场初始相距
//
// ★ 2026-10-08 用户要求「战斗加成放到最下面，这相当于总加成」。
// 这个测试同时把 head 逐行打出来，方便肉眼核对战报/指挥室的准备回合内容。
func TestBattleHeadOrder(t *testing.T) {
	st := ezfyNewBattleState(nil, nil,
		100, 50, 30, 20, 10, // atkBonus, defBonus, defAtkBonus, atkSpeedBonus, defSpeedBonus
		15, 5, // atkRangeBonus, defRangeBonus
		ezfyBattleBonus{Dmg: 49, Def: 61, Hp: 73, Move: 10, Crit: 45, CritDmg: 64}, // atkEquip（含套装的全量）
		ezfyBattleBonus{}, // defEquip
		ezfyBattleBonus{Dmg: 49, Def: 61, Hp: 73, Move: 10, Crit: 45, CritDmg: 64}, // atkSet
		ezfyBattleBonus{}, // defSet
		"赤色锤镰[裁决]", "", // atkSetDesc, defSetDesc
		"Michael·Adams Lv.150 攻击加成+365% 防御加成+144%",
		"斯大林 Lv.350 攻击加成+325% 守军防御+100%",
		365, 325, // atkOfficerBonus, defOfficerBonus
		150, 180, // atkOfficerSkill, defOfficerSkill
		[]ezfyBonusItem{{Name: "尖兵突击", Value: 150}},
		[]ezfyBonusItem{{Name: "尖兵突击", Value: 180}},
		[]ezfyBonusItem{{Name: "军训艺术", Value: 20}, {Name: "武器科技", Value: 30}, {Name: "重工技术", Value: 20}},
		[]ezfyBonusItem{{Name: "装甲科技", Value: 30}, {Name: "掩体防御", Value: 20}},
		[]ezfyBonusItem{{Name: "城墙", Value: 45}},
		355, []ezfyBonusItem{{Name: "军官·Michael·Adams", Value: 144}}, // atkDefBonus, atkDefBreak
		map[int]int{}, map[int]int{}, map[int]int{}, map[int]int{},
		5, 6, 1, 2,
		ezfyTypeBonus{}, ezfyTypeBonus{})

	t.Logf("准备回合共 %d 行：", len(st.Head))
	for i, ln := range st.Head {
		t.Logf("  %d| %s", i, ln)
	}

	idxBonus, idxSet, idxTech := -1, -1, -1
	for i, ln := range st.Head {
		switch {
		case strings.HasPrefix(ln, "战斗加成:"):
			idxBonus = i
		case strings.HasPrefix(ln, "【攻方套装】"):
			idxSet = i
		case strings.HasPrefix(ln, "【攻方科技】"):
			idxTech = i
		}
	}
	if idxBonus < 0 {
		t.Fatal("准备回合缺少「战斗加成」行")
	}
	if idxTech < 0 {
		t.Fatal("准备回合缺少【攻方科技】行（科技明细必须展示）")
	}
	if idxSet < 0 {
		t.Fatal("准备回合缺少【攻方套装】行")
	}
	if idxBonus < idxSet {
		t.Fatalf("「战斗加成」必须排在明细之后：bonus=%d < 套装=%d", idxBonus, idxSet)
	}
	if idxBonus < idxTech {
		t.Fatalf("「战斗加成」必须排在【攻方科技】之后：bonus=%d < 科技=%d", idxBonus, idxTech)
	}
	// 最后一行是场景描述，战斗加成应紧挨在它前面
	if last := st.Head[len(st.Head)-1]; !strings.HasPrefix(last, "战场初始相距") {
		t.Fatalf("最后一行应是场景描述，实际：%s", last)
	}
}
