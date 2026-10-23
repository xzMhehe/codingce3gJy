package ezfy

import (
	"testing"

	"qqjiayuan/server/internal/model"
)

// ★ 2026-10-24 伤害算法随公式套切换：内置 = 原版，选套 = 套内 damage_formula。
// 玩家端无感知，管理端切套后由 cfgsReload / 30s 刷新即时生效。
func TestEzfyDamageFormulaSwitch(t *testing.T) {
	oldID, oldAtk, oldAtkBase, oldDef, oldDefBase, oldMin, oldForms :=
		ezfyCfg.limit.BattleFormulaID, ezfyCfg.limit.AtkCoeff, ezfyCfg.limit.AtkBase,
		ezfyCfg.limit.DefCoeff, ezfyCfg.limit.DefBase, ezfyCfg.limit.MinDmgCoeff, ezfyCfg.formulas
	defer func() {
		ezfyCfg.limit.BattleFormulaID = oldID
		ezfyCfg.limit.AtkCoeff = oldAtk
		ezfyCfg.limit.AtkBase = oldAtkBase
		ezfyCfg.limit.DefCoeff = oldDef
		ezfyCfg.limit.DefBase = oldDefBase
		ezfyCfg.limit.MinDmgCoeff = oldMin
		ezfyCfg.formulas = oldForms
	}()

	ezfyCfg.limit.BattleFormulaID = 0
	ezfyCfg.limit.AtkCoeff = 5
	ezfyCfg.limit.AtkBase = 1000
	ezfyCfg.limit.DefCoeff = 3
	ezfyCfg.limit.DefBase = 10
	ezfyCfg.limit.MinDmgCoeff = 5
	ezfyCfg.formulas = map[int]model.EzfyCfgBattleFormula{
		9: {ID: 9, DamageFormula: 1},
	}

	if got := ezfyDamageFormulaCfg(); got != 0 {
		t.Fatalf("内置应为原版算法，得到 %d", got)
	}
	// 原版：(1000*5+1000)/(100*(1+20%)*3+10)*10*1.5 = 240
	if got := ezfyCalcDamage(1000, 100, 10, 50, 20); got != 240 {
		t.Fatalf("原版伤害应为 240，得到 %d", got)
	}

	ezfyCfg.limit.BattleFormulaID = 9
	if got := ezfyDamageFormulaCfg(); got != 1 {
		t.Fatalf("选套后应为图片算法，得到 %d", got)
	}
	// 图片：10*1000²*(1+50%)/(100*(1+20%)) = 125000
	if got := ezfyCalcDamage(1000, 100, 10, 50, 20); got != 125000 {
		t.Fatalf("图片算法伤害应为 125000，得到 %d", got)
	}

	// 套失效：立刻回落到内置原版，不报错、不影响玩家
	ezfyCfg.limit.BattleFormulaID = 99
	if got := ezfyDamageFormulaCfg(); got != 0 {
		t.Fatalf("套不存在应回落原版，得到 %d", got)
	}
	if got := ezfyCalcDamage(1000, 100, 10, 50, 20); got != 240 {
		t.Fatalf("套不存在应回落原版伤害 240，得到 %d", got)
	}
}

func TestEzfyWallDurabilityAndDamage(t *testing.T) {
	want := []int64{0, 20000, 80000, 180000, 320000, 500000, 720000, 980000, 1280000, 1620000, 2000000}
	for lv, hp := range want {
		if got := ezfyWallDurabilityMax(lv); got != hp {
			t.Fatalf("城墙 %d 级耐久应为 %d，得到 %d", lv, hp, got)
		}
		if lv > 0 {
			if got := ezfyWallDefBonusMax(lv); got != lv*5 {
				t.Fatalf("城墙 %d 级防御上限应为 %d，得到 %d", lv, lv*5, got)
			}
		}
	}

	st := &ezfyBattleState{WallEnabled: true, WallHp: 50000, WallHpMax: 200000, WallDefBonusMax: 40}
	if got := st.wallDefBonus(); got != 10 {
		t.Fatalf("耐久 1/4 时防御加成应为 10%%，得到 %d", got)
	}
	st.WallHp = 0
	if got := st.wallDefBonus(); got != 0 {
		t.Fatalf("耐久为 0 时防御加成应为 0，得到 %d", got)
	}
	st.WallEnabled = false
	st.WallHp = 200000
	if got := st.wallDefBonus(); got != 0 {
		t.Fatalf("掠夺战不应有城墙防御，得到 %d", got)
	}

	// 10 个兵、对防 1000、攻击加成 50%：10*1000²*150%/20000 = 75000
	if got := ezfyWallDamage(1000, 10, 50); got != 75000 {
		t.Fatalf("城墙伤害应为 75000，得到 %d", got)
	}
	if got := ezfyWallDamage(1000, 10, -100); got != 0 {
		t.Fatalf("攻击加成 -100%% 时城墙伤害应为 0，得到 %d", got)
	}
	if got := ezfyWallDamage(0, 10, 50); got != 0 {
		t.Fatalf("基础攻击 0 时城墙伤害应为 0，得到 %d", got)
	}
}

// ★ 2026-10-24 征服战：守军全灭后必须继续拆城墙，耐久归零才判攻方胜；
// 拆墙伤害取「对防」AtkDef（城墙属于防御设施），并吃对城防(type=4)的兵种专属加成。
func TestEzfyWallSiegeAfterDefendersDead(t *testing.T) {
	mk := func() *ezfyBattleState {
		a := ezfyTestUnit("A1", "步兵", 0, 10)
		a.cfg.AtkGround = 1
		a.cfg.AtkDef = 1000
		return &ezfyBattleState{
			Attackers:   []*ezfyFightUnit{a},
			Defenders:   []*ezfyFightUnit{},
			WallEnabled: true, WallHp: 100000, WallHpMax: 100000,
			AtkType: ezfyTypeBonus{AtkVsType: map[int]map[int]int{
				ezfyTroopTypeArmy: {ezfyTroopTypeCity: 50},
			}},
			Head: []string{}, Actions: []string{},
		}
	}

	st := mk()
	st.Step(nil, nil)
	// 10*1000²*150%/20000 = 75000；守军已灭但墙未破 → 继续战斗，不能判胜
	if st.WallHp != 25000 {
		t.Fatalf("围城伤害应按对防+对城防加成计算，城墙剩余应为 25000，得到 %d", st.WallHp)
	}
	if st.Done || st.AttackerWin || st.Draw {
		t.Fatalf("城墙未破不能结束战斗，done=%v win=%v draw=%v", st.Done, st.AttackerWin, st.Draw)
	}

	st.Step(nil, nil)
	if !st.Done || !st.AttackerWin || st.Draw {
		t.Fatalf("城墙耐久归零后应判攻方胜，done=%v win=%v draw=%v", st.Done, st.AttackerWin, st.Draw)
	}
	if st.WallHp != 0 {
		t.Fatalf("城墙耐久应归零，得到 %d", st.WallHp)
	}

	// 回合耗尽仍未破墙 → 平局（守方算守住）
	st = mk()
	st.WallHp = 1 << 60
	st.WallHpMax = st.WallHp
	st.Round = ezfyBattleMaxRoundsCfg() - 1
	st.Step(nil, nil)
	if !st.Done || !st.Draw || st.AttackerWin {
		t.Fatalf("回合耗尽且城墙未破应判平局，done=%v win=%v draw=%v", st.Done, st.AttackerWin, st.Draw)
	}
}

// 城墙只属于「建了围墙的玩家城市」：野地/寇城/掠夺/没建墙的城 WallEnabled=false，
// 守军全灭应立即判攻方胜，不能进入围城阶段。
func TestEzfyNoWallDefendersDeadImmediateWin(t *testing.T) {
	a := ezfyTestUnit("A1", "步兵", 0, 10)
	st := &ezfyBattleState{
		Attackers:   []*ezfyFightUnit{a},
		Defenders:   []*ezfyFightUnit{},
		WallEnabled: false,
		Head:        []string{}, Actions: []string{},
	}
	st.Step(nil, nil)
	if !st.Done || !st.AttackerWin || st.Draw {
		t.Fatalf("无城墙时守军全灭应立即判攻方胜，done=%v win=%v draw=%v", st.Done, st.AttackerWin, st.Draw)
	}
}
