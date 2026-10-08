package ezfy

import (
	"testing"

	"qqjiayuan/server/internal/model"
)

// TestActWildBattleRepro 复现线上「活动野地战斗 0 回合、攻方全灭、守方零损失」。
//
// 数据全部取自线上库（2026-10-07 19:26 那场）：
//
//	订单 38732: user 35806436(阿正) troops=[{15:204900},{16:100000}]  officer=Stalin（斯大林）
//	活动野地配置 ezfy_act_wild id=42 (288,98): level=3,
//	  troops=[[1,20000],[2,20000],[13,20000],[15,20000],[16,20000]]  officer_id=5(戴高乐)
//	战报 battle_result: {"Rounds":0,"AttackerLeft":[],"DefenderLeft":[]}
//
// 目的：判断本地代码在该输入下是否也会「开局即 Done」（参战单位被过滤空）。
func TestActWildBattleRepro(t *testing.T) {
	saved := ezfyCfg.troops
	defer func() { ezfyCfg.troops = saved }()

	// 线上 ezfy_cfg_troop 的真实属性
	ezfyCfg.troops = map[int]model.EzfyCfgTroop{
		1:  {ID: 1, Name: "步兵", Type: 2, Health: 100, AtkSea: 10, AtkGround: 18, AtkAir: 9, Defence: 10, Speed: 300, AttackRange: 20},
		2:  {ID: 2, Name: "摩托化骑兵", Type: 2, Health: 65, AtkSea: 7, AtkGround: 12, AtkAir: 5, Defence: 8, Speed: 1400, AttackRange: 30},
		13: {ID: 13, Name: "驱逐舰", Type: 1, Health: 350, AtkSea: 45, AtkGround: 65, AtkAir: 42, Defence: 45, Speed: 1000, AttackRange: 200},
		15: {ID: 15, Name: "战列舰", Type: 1, Health: 1500, AtkSea: 120, AtkGround: 85, AtkAir: 73, Defence: 120, Speed: 900, AttackRange: 1800},
		16: {ID: 16, Name: "航母", Type: 1, Health: 2200, AtkSea: 100, AtkGround: 95, AtkAir: 110, Defence: 100, Speed: 850, AttackRange: 3100},
	}

	attacker := []ezfyUnitGroup{{TroopId: 15, Count: 204900}, {TroopId: 16, Count: 100000}}
	defender := []ezfyUnitGroup{
		{TroopId: 1, Count: 20000}, {TroopId: 2, Count: 20000}, {TroopId: 13, Count: 20000},
		{TroopId: 15, Count: 20000}, {TroopId: 16, Count: 20000},
	}

	st := ezfyNewBattleState(attacker, defender,
		346, 40, 40, 50, 0, // atkBonus, defBonus, defAtkBonus, atkSpeed, defSpeed
		30, 0, // atkRange, defRange
		ezfyBattleBonus{}, ezfyBattleBonus{}, ezfyBattleBonus{}, ezfyBattleBonus{}, "", "",
		"Stalin（斯大林）", "戴高乐（Charles）",
		346, 40, 0, 0, // officerBonus, officerSkill
		nil, nil, nil, nil, nil, // skills/techs/defBreak
		165, nil, // atkDefBonus, atkDefBreak
		nil, nil, nil, nil, // targets/moves
		0, 0, 1, 1, // counterRounds, camps
		ezfyTypeBonus{}, ezfyTypeBonus{})

	t.Logf("开局: Done=%v  Attackers=%d  Defenders=%d", st.Done, len(st.Attackers), len(st.Defenders))
	if len(st.Attackers) == 0 || len(st.Defenders) == 0 {
		t.Fatalf("参战单位被过滤空 → 必然 0 回合。Attackers=%d Defenders=%d",
			len(st.Attackers), len(st.Defenders))
	}

	rounds := 0
	for !st.Done && rounds < 200 {
		st.Step(nil, nil)
		rounds++
	}
	br := st.Result()
	t.Logf("结果: Rounds=%d  攻方胜=%v  平局=%v  攻方剩余=%v  守方损失=%v",
		br.Rounds, br.AttackerWin, br.Draw, br.AttackerLeft, br.DefenderLosses)
}
