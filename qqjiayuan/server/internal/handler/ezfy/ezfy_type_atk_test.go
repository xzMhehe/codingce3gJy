package ezfy

import (
	"strings"
	"testing"

	"qqjiayuan/server/internal/model"
)

// TestAttackTechBreakFiltersNonAttackTechs 行动行的攻击拆解只能列攻击科技。
//
// ★★ 2026-10-31 用户反馈「狼群围猎加成对不上」：原先把【攻方科技】的全量科技
// （含均质装甲/增程弹药/机械增压等）塞进攻击拆解，括号内合计远大于攻击总数。
func TestAttackTechBreakFiltersNonAttackTechs(t *testing.T) {
	items := []ezfyBonusItem{
		{Name: "枪械改良", Value: 30},
		{Name: "均质装甲", Value: 30},
		{Name: "增程弹药", Value: 30},
		{Name: "重型机床", Value: 20},
		{Name: "机械增压", Value: 50},
		{Name: "永备工事", Value: 20},
		{Name: "喷气推进", Value: 50},
	}
	got := ezfyAttackTechItems(items)
	if len(got) != 2 || got[0].Name != "枪械改良" || got[1].Name != "重型机床" {
		t.Fatalf("攻击拆解应只保留枪械改良+重型机床，实际 %+v", got)
	}
}

// TestWolfPackBonusPerTargetAndLog 狼群围猎 = 海军打海军才生效：
//
//	① 汇总行明确写「兵种专属攻击(海军对海+75%)」，方便和行动行对账；
//	② 攻击行把 75% 并进总加成括号，不再写成括号外的独立一段；
//	③ 溢出打到陆军时，必须去掉这 75%（不能沿用原目标的海军对海加成）。
func TestWolfPackBonusPerTargetAndLog(t *testing.T) {
	saved := ezfyCfg.troops
	defer func() { ezfyCfg.troops = saved }()
	ezfyCfg.troops = map[int]model.EzfyCfgTroop{
		// 攻方海军，攻击高、能一击打穿第一目标
		31: {ID: 31, Name: "测试舰", Type: ezfyTroopTypeNavy, Health: 1000,
			AtkSea: 100, AtkGround: 100, AtkAir: 100, Defence: 100, Speed: 100, AttackRange: 99999},
		// 守方第一目标：海军，1 个、1 血，必定被秒
		1: {ID: 1, Name: "敌舰", Type: ezfyTroopTypeNavy, Health: 1,
			AtkSea: 0, AtkGround: 0, AtkAir: 0, Defence: 1, Speed: 0, AttackRange: 1},
		// 守方第二目标：陆军，血厚，只吃溢出的无专属加成伤害
		2: {ID: 2, Name: "敌坦克", Type: ezfyTroopTypeArmy, Health: 1000,
			AtkSea: 0, AtkGround: 0, AtkAir: 0, Defence: 1000, Speed: 0, AttackRange: 1},
	}
	atkType := ezfyTypeBonus{
		AtkVsType: map[int]map[int]int{
			ezfyTroopTypeNavy: {ezfyTroopTypeNavy: 75}, // 狼群围猎 V：海军打海军 +75%
		},
	}
	st := ezfyCloseIn(ezfyNewBattleState(
		[]ezfyUnitGroup{{TroopId: 31, Count: 1000}},
		[]ezfyUnitGroup{{TroopId: 1, Count: 1}, {TroopId: 2, Count: 1000000}},
		0, 0, 0, 0, 0,
		0, 0,
		ezfyBattleBonus{}, ezfyBattleBonus{},
		ezfyBattleBonus{}, ezfyBattleBonus{}, "", "",
		"", "", 0, 0, 0, 0,
		nil, nil, nil, nil, nil,
		0, nil,
		map[int]int{31: 1}, map[int]int{}, map[int]int{}, map[int]int{}, // 优先打海军敌舰
		0, 0, 1, 2,
		atkType, ezfyTypeBonus{}))
	st.Step(nil, nil)

	head := strings.Join(st.Head, "\n")
	if !strings.Contains(head, "兵种专属攻击(海军对海+75%)") {
		t.Fatalf("汇总行应明确列出狼群围猎的兵种专属攻击，得到：\n%s", head)
	}

	var firstLine, overflowLine string
	for _, ln := range st.Actions {
		if strings.Contains(ln, "测试舰攻击") && !strings.Contains(ln, "势不可挡") {
			firstLine = ln
		}
		if strings.Contains(ln, "势不可挡") {
			overflowLine = ln
		}
	}
	if firstLine == "" || overflowLine == "" {
		t.Fatalf("应产生主攻击与溢出攻击行，实际：\n%s", strings.Join(st.Actions, "\n"))
	}
	// 主攻击：75% 必须在总加成括号内（总数 75% = 兵种专属 75%），不能写成「75%(…) + 别处再来 75%」
	if !strings.Contains(firstLine, "攻击加成+75%(兵种专属+75%)") {
		t.Fatalf("主攻击行应把兵种专属并入总加成括号，实际：%s", firstLine)
	}
	// 溢出目标变成陆军：狼群围猎不再生效
	if strings.Contains(overflowLine, "兵种专属+75%") {
		t.Fatalf("溢出打到陆军时不应继续带海军对海 +75%%，实际：%s", overflowLine)
	}
	if !strings.Contains(overflowLine, "敌坦克") {
		t.Fatalf("溢出目标应为陆军敌坦克，实际：%s", overflowLine)
	}
}

// TestComboFollowUpRecomputesTypeBonusForNewTarget 连击换目标时按**新目标**重取兵种专属加成。
//
// ★★ 2026-10-31 与狼群围猎同源：连击追加的第二次攻击原来沿用主目标算好的加成，
// 原目标（海军）被打死后改打陆军时仍会带着「海军对海 +75%」，伤害与日志都对不上。
func TestComboFollowUpRecomputesTypeBonusForNewTarget(t *testing.T) {
	st := &ezfyBattleState{Actions: []string{}}
	unit := &ezfyFightUnit{id: "A1", count: 1000, initialCount: 1000, pos: 0, hp: 100,
		cfg: &ezfyTroopStats{Name: "测试舰", Type: ezfyTroopTypeNavy,
			AtkSea: 200, AtkGround: 200, AtkAir: 200, Defence: 10, Speed: 100, AttackRange: 500}}
	// 原目标：海军，已被主攻击打光
	deadNavy := &ezfyFightUnit{id: "D1", count: 0, initialCount: 1, pos: 50, hp: 10,
		cfg: &ezfyTroopStats{Name: "敌舰", Type: ezfyTroopTypeNavy, Health: 10, Defence: 10, AttackRange: 1}}
	// 换打目标：陆军，血厚
	army := &ezfyFightUnit{id: "D2", count: 1000000, initialCount: 1000000, pos: 100, hp: 100,
		cfg: &ezfyTroopStats{Name: "敌坦克", Type: ezfyTroopTypeArmy, Health: 100, Defence: 10, AttackRange: 1}}

	seen := []int{}
	atkBonusAt := func(t *ezfyFightUnit) int {
		seen = append(seen, t.cfg.Type)
		if t.cfg.Type == ezfyTroopTypeNavy {
			return 75 // 狼群围猎：只有打海军才生效
		}
		return 0
	}
	st.comboFollowUp(unit, deadNavy, []*ezfyFightUnit{deadNavy, army}, 500, 0,
		atkBonusAt, func(*ezfyFightUnit) int { return 0 }, 0,
		"【攻方】", "【守方】", func(u *ezfyFightUnit) string { return u.cfg.Name },
		"波次突击", 100, "连击", false, 0, 0)

	if len(seen) != 1 || seen[0] != ezfyTroopTypeArmy {
		t.Fatalf("连击换目标后应按陆军目标重取加成，实际取到兵种 %v", seen)
	}
	want := ezfyCalcDamage(ezfyPickAttack(unit.cfg, army.cfg), army.cfg.Defence, unit.count, 0, 0) / int64(army.hp)
	if want < 1 {
		want = 1
	}
	if got := army.initialCount - army.count; got != want {
		t.Fatalf("连击打陆军不应带海军对海+75%%，消灭 %d，按无专属加成应为 %d", got, want)
	}
}

// TestCounterLogPutsWolfPackInsideBracket 反击行同样把兵种专属并入总加成括号。
//
// ★★ 2026-10-31 用户反馈「狼群围猎加成对不上」：反击原来把「兵种专属+75%」写在括号外，
// 和普攻行「攻击加成+250%(…)」的读数口径不一致，看起来像再加一次。
func TestCounterLogPutsWolfPackInsideBracket(t *testing.T) {
	saved := ezfyCfg.troops
	defer func() { ezfyCfg.troops = saved }()
	ezfyCfg.troops = map[int]model.EzfyCfgTroop{
		31: {ID: 31, Name: "测试舰", Type: ezfyTroopTypeNavy, Health: 100000,
			AtkSea: 50, AtkGround: 50, AtkAir: 50, Defence: 50, Speed: 100, AttackRange: 99999},
	}
	st := ezfyCloseIn(ezfyNewBattleState(
		[]ezfyUnitGroup{{TroopId: 31, Count: 1000}},
		[]ezfyUnitGroup{{TroopId: 31, Count: 100000}},
		0, 0, 0, 0, 0,
		0, 0,
		ezfyBattleBonus{}, ezfyBattleBonus{},
		ezfyBattleBonus{}, ezfyBattleBonus{}, "", "",
		"", "", 0, 0, 0, 0,
		nil, nil, nil, nil, nil,
		0, nil,
		map[int]int{}, map[int]int{}, map[int]int{}, map[int]int{},
		0, 5, 1, 2, // 守方（海军）前 5 回合发动绝地逆袭
		ezfyTypeBonus{},
		ezfyTypeBonus{AtkVsType: map[int]map[int]int{
			ezfyTroopTypeNavy: {ezfyTroopTypeNavy: 75}, // 守方狼群围猎
		}}))
	st.Step(nil, nil)

	counterLine := ""
	for _, ln := range st.Actions {
		if strings.Contains(ln, "绝地逆袭") {
			counterLine = ln
		}
	}
	if counterLine == "" {
		t.Fatalf("本场应触发守方反击，实际：\n%s", strings.Join(st.Actions, "\n"))
	}
	if !strings.Contains(counterLine, "攻击加成+75%(兵种专属+75%)") {
		t.Fatalf("反击行应把兵种专属并入总加成括号，实际：%s", counterLine)
	}
}
