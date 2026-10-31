package ezfy

import (
	"testing"

	"qqjiayuan/server/internal/model"
)

// ★ 2026-10-31 用户规则：战场默认距离不再是固定 6000，而是
// `max(攻方最大单兵种最终射程, 守方最大单兵种最终射程) + 余量`，
// 余量在「二战系统配置」里可调（0 合法）。
//
// 「最终射程」= 兵种基础射程 × (1 + 通用射程% + 该兵种专属射程%)，与引擎 rangeD 同口径。
func TestBattleStartDistAutoFromMaxRange(t *testing.T) {
	savedTroops, savedLimit := ezfyCfg.troops, ezfyCfg.limit
	defer func() { ezfyCfg.troops, ezfyCfg.limit = savedTroops, savedLimit }()

	ezfyCfg.troops = map[int]model.EzfyCfgTroop{
		1:  {ID: 1, Name: "步兵", Type: 2, AttackRange: 20},
		18: {ID: 18, Name: "榴弹炮", Type: 4, AttackRange: 5300},
	}
	ezfyCfg.limit = model.EzfyCfgLimit{
		BattleMaxRounds: 40, BattleStartDist: 0, BattleStartDistMargin: 2000,
	}
	atk := []ezfyUnitGroup{{TroopId: 1, Count: 10}}
	def := []ezfyUnitGroup{{TroopId: 18, Count: 10}}

	// 守方榴弹炮 5300 × 1.10 = 5830（通用射程 +10%），+2000 余量 = 7830
	if got := ezfyBattleStartDistFor(atk, def, 10, 10, ezfyTypeBonus{}, ezfyTypeBonus{}); got != 7830 {
		t.Fatalf("默认距离应为 5830+2000=7830，实际 %d", got)
	}
	// 余量配 0（合法）→ 贴着最远射程开局
	ezfyCfg.limit.BattleStartDistMargin = 0
	if got := ezfyBattleStartDistFor(atk, def, 10, 10, ezfyTypeBonus{}, ezfyTypeBonus{}); got != 5830 {
		t.Fatalf("余量 0 时距离应为 5830，实际 %d", got)
	}
	// 余量 500
	ezfyCfg.limit.BattleStartDistMargin = 500
	if got := ezfyBattleStartDistFor(atk, def, 10, 10, ezfyTypeBonus{}, ezfyTypeBonus{}); got != 6330 {
		t.Fatalf("余量 500 时距离应为 6330，实际 %d", got)
	}
	// 手动固定值优先（管理员想固定就固定）
	ezfyCfg.limit.BattleStartDist = 5000
	if got := ezfyBattleStartDistFor(atk, def, 10, 10, ezfyTypeBonus{}, ezfyTypeBonus{}); got != 5000 {
		t.Fatalf("手动固定 5000 应优先，实际 %d", got)
	}
}

// 双方都没有射程（纯运输/空部队）→ 回落历史默认，不能返回 0（0 会让城墙贴着攻方）。
func TestBattleStartDistNoRangeFallback(t *testing.T) {
	savedLimit := ezfyCfg.limit
	defer func() { ezfyCfg.limit = savedLimit }()
	ezfyCfg.limit = model.EzfyCfgLimit{BattleMaxRounds: 40, BattleStartDistMargin: 2000}
	if got := ezfyBattleStartDistFor(nil, nil, 0, 0, ezfyTypeBonus{}, ezfyTypeBonus{}); got != ezfyBattleStartDistDef {
		t.Fatalf("无部队时应回落 %d，实际 %d", ezfyBattleStartDistDef, got)
	}
}

// 新战场必须把算出来的距离固化进快照字段（否则暂停重进会退回默认值）。
func TestBattleStartDistFixedInState(t *testing.T) {
	saved := ezfyCfg.troops
	defer func() { ezfyCfg.troops = saved }()
	ezfyCfg.troops = map[int]model.EzfyCfgTroop{
		1: {ID: 1, Name: "步兵", Type: 2, AttackRange: 20},
	}
	st := ezfyNewBattleState([]ezfyUnitGroup{{TroopId: 1, Count: 10}},
		[]ezfyUnitGroup{{TroopId: 1, Count: 10}},
		0, 0, 0, 0, 0, 0, 0,
		ezfyBattleBonus{}, ezfyBattleBonus{},
		ezfyBattleBonus{}, ezfyBattleBonus{}, "", "", "", "",
		0, 0, 0, 0,
		nil, nil, nil, nil, nil, 0, nil,
		map[int]int{}, map[int]int{}, map[int]int{}, map[int]int{},
		0, 0, 0, 0, ezfyTypeBonus{}, ezfyTypeBonus{})
	if st.StartDist <= 0 {
		t.Fatalf("新战场必须固化 StartDist，实际 %d", st.StartDist)
	}
	if st.Defenders[0].pos != st.StartDist {
		t.Fatalf("守方初始位置应等于 StartDist：pos=%d startDist=%d", st.Defenders[0].pos, st.StartDist)
	}
}
