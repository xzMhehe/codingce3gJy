package ezfy

import (
	"testing"

	"qqjiayuan/server/internal/model"
)

// ★ 2026-10-23 战斗公式套：选了套整套覆盖，未选/套不存在回落 limit 内置值，
// 套内字段为 0 时回落硬编码默认值（不是 limit 值）。
func TestEzfyBattleFormulaActiveOverridesLimit(t *testing.T) {
	oldID, oldRounds, oldAtk, oldAtkBase, oldForms := ezfyCfg.limit.BattleFormulaID,
		ezfyCfg.limit.BattleMaxRounds, ezfyCfg.limit.AtkCoeff, ezfyCfg.limit.AtkBase, ezfyCfg.formulas
	defer func() {
		ezfyCfg.limit.BattleFormulaID = oldID
		ezfyCfg.limit.BattleMaxRounds = oldRounds
		ezfyCfg.limit.AtkCoeff = oldAtk
		ezfyCfg.limit.AtkBase = oldAtkBase
		ezfyCfg.formulas = oldForms
	}()

	ezfyCfg.limit.BattleFormulaID = 0
	ezfyCfg.limit.BattleMaxRounds = 40
	ezfyCfg.limit.AtkCoeff = 5
	ezfyCfg.limit.AtkBase = 1000
	ezfyCfg.formulas = map[int]model.EzfyCfgBattleFormula{
		7: {ID: 7, BattleMaxRounds: 12, AtkCoeff: 9, AtkBase: 0},
	}

	if got := ezfyBattleMaxRoundsCfg(); got != 40 {
		t.Fatalf("未选公式套应读内置值 40，得到 %d", got)
	}

	ezfyCfg.limit.BattleFormulaID = 7
	if got := ezfyBattleMaxRoundsCfg(); got != 12 {
		t.Fatalf("选套后应读套内 12，得到 %d", got)
	}
	if got := ezfyAtkCoeffCfg(); got != 9 {
		t.Fatalf("选套后应读套内攻击系数 9，得到 %d", got)
	}
	if got := ezfyAtkBaseCfg(); got != ezfyAtkBaseDef {
		t.Fatalf("套内 0 应回落硬编码默认 %d，得到 %d", ezfyAtkBaseDef, got)
	}

	// 套被删 / ID 失效：回落 limit 内置值（与旧行为一致）
	ezfyCfg.limit.BattleFormulaID = 99
	if got := ezfyBattleMaxRoundsCfg(); got != 40 {
		t.Fatalf("公式套不存在应回落内置 40，得到 %d", got)
	}
}
