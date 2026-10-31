package ezfy

// ★★ 2026-10-23 名将组合技能（系统专属战法）测试：
//
//	① 等级成长：效果值 = base + (组合技等级-1)×grow，等级由军官等级分档（步长 35、上限 10）；
//	② 只有名将能拿到（普通军官 / 非名将池 / 已下架 → nil）；
//	③ 概率发动：ProcPct=100 必发、=0 永不发，发动后按 Rounds 持续；
//	④ 效果真的进战斗：攻击加成让伤害更高、破防让伤害更高、回合治疗能补兵。

import (
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/model"
)

// stubComboPool 临时往配置缓存里塞一条组合技（测试结束还原）
func stubComboPool(t *testing.T, s model.EzfyCfgGeneralSkill) func() {
	t.Helper()
	old := ezfyCfg.generalSkills
	m := map[int]model.EzfyCfgGeneralSkill{}
	for k, v := range old {
		m[k] = v
	}
	m[s.ID] = s
	ezfyCfg.generalSkills = m
	return func() { ezfyCfg.generalSkills = old }
}

// stubGeneralCfg 临时往名将池里塞一条名将配置
func stubGeneralCfg(t *testing.T, g model.EzfyCfgGeneral) func() {
	t.Helper()
	old := ezfyCfg.generals
	m := map[int]model.EzfyCfgGeneral{}
	for k, v := range old {
		m[k] = v
	}
	m[g.ID] = g
	ezfyCfg.generals = m
	return func() { ezfyCfg.generals = old }
}

func TestComboValueGrow(t *testing.T) {
	cases := []struct{ base, grow, lv, max, want int }{
		{7, 1, 1, 10, 7},
		{7, 1, 5, 10, 11},
		{7, 1, 10, 10, 16},
		{7, 1, 99, 10, 16}, // 超过上限 → 按满级
		{7, 1, 0, 10, 7},   // 非法等级 → 按 1 级
		{0, 0, 1, 10, 1},   // 至少 1
		{1, 1, 5, 5, 5},    // 反击回合这类「次数」型（上限 5 级）
		{52, 2, 10, 10, 70},
	}
	for _, c := range cases {
		if got := ezfyComboValue(c.base, c.grow, c.lv, c.max); got != c.want {
			t.Fatalf("base=%d grow=%d lv=%d → %d, want %d", c.base, c.grow, c.lv, got, c.want)
		}
	}
}

// TestComboLevelByOfficerLevel 组合技等级按**军官等级**分档（名将满级 350、人人 5 星，
// 所以不能拿星级当等级 —— 2026-10-23 用户纠正）。
// 默认步长 35、上限 10 → 350 级正好 10 级吃满。
func TestComboLevelByOfficerLevel(t *testing.T) {
	cases := []struct{ lv, max, step, want int }{
		{1, 10, 35, 1},
		{35, 10, 35, 1},
		{36, 10, 35, 2},
		{70, 10, 35, 2},
		{105, 10, 35, 3},
		{176, 10, 35, 6},
		{281, 10, 35, 9},
		{316, 10, 35, 10},
		{350, 10, 35, 10}, // 名将满级 → 正好 10 级
		{999, 10, 35, 10}, // 超过上限仍 10 级
		{350, 5, 35, 5},   // 管理端把上限调 5 → 5 级封顶（反击回合就这么配）
		{350, 5, 70, 5},   // 步长调回 70 → 350 级正好 5 级
		{0, 0, 0, 1},      // 非法输入回落（默认 10 级上限 / 步长 35）
	}
	for _, c := range cases {
		if got := ezfyComboLevelOf(c.lv, c.max, c.step); got != c.want {
			t.Fatalf("等级 %d / 上限 %d / 步长 %d → %d 级, want %d", c.lv, c.max, c.step, got, c.want)
		}
	}
}

func TestComboOnlyForGenerals(t *testing.T) {
	restorePool := stubComboPool(t, model.EzfyCfgGeneralSkill{
		ID: 901, Name: "测试组合技", Kind: ezfyComboAtk, BaseValue: 10, GrowPerLevel: 2,
		MaxLevel: 10, LevelPerStep: 35, ProcPct: 50, Rounds: 1, Enabled: 1,
	})
	defer restorePool()
	restoreG := stubGeneralCfg(t, model.EzfyCfgGeneral{ID: 9901, Name: "测试名将", Kind: 2, ComboSkillIds: "[901]", Star: 5})
	defer restoreG()
	// 普通军官池（kind=1）即便误配了 combo_skill_ids 也不生效
	restoreG2 := stubGeneralCfg(t, model.EzfyCfgGeneral{ID: 9902, Name: "测试普通", Kind: 1, ComboSkillIds: "[901]"})
	defer restoreG2()

	// 210 级 → 1+(210-1)/35 = 6 级：10 + 5×2 = 20
	cs := ezfyComboOfGeneralCfg(ezfyCfg.general(9901), 210)
	if len(cs) != 1 {
		t.Fatalf("名将应拿到 1 个组合技, 实际 %d 个", len(cs))
	}
	c := cs[0]
	if c.Level != 6 || c.Value != 20 {
		t.Fatalf("210 级名将应为 6 级/20 点, 实际 Lv.%d/%d", c.Level, c.Value)
	}
	// 350 级（名将满级）→ 10 级吃满：10 + 9×2 = 28
	max := ezfyComboOfGeneralCfg(ezfyCfg.general(9901), 350)
	if len(max) != 1 || max[0].Level != 10 || max[0].Value != 28 {
		t.Fatalf("350 级名将应为 10 级/28 点, 实际 %+v", max)
	}
	if len(ezfyComboOfGeneralCfg(ezfyCfg.general(9902), 5)) != 0 {
		t.Fatal("普通军官（kind=1）不该有组合技")
	}
	if len(ezfyComboOfGeneralCfg(nil, 5)) != 0 {
		t.Fatal("nil 名将配置应返回空")
	}
	// ★ 2026-10-24 多技能：一个名将配多个技能 → 全部下发、顺序按配置
	restorePool2 := stubComboPool(t, model.EzfyCfgGeneralSkill{
		ID: 902, Name: "测试组合技2", Kind: ezfyComboDef, BaseValue: 5, GrowPerLevel: 1,
		MaxLevel: 10, LevelPerStep: 35, ProcPct: 30, Rounds: 2, Enabled: 1,
	})
	defer restorePool2()
	restoreG3 := stubGeneralCfg(t, model.EzfyCfgGeneral{ID: 9903, Name: "测试名将2", Kind: 2, ComboSkillIds: "[901,902]"})
	defer restoreG3()
	multi := ezfyComboOfGeneralCfg(ezfyCfg.general(9903), 100)
	if len(multi) != 2 {
		t.Fatalf("配 2 个技能应下发 2 个, 实际 %d", len(multi))
	}
	if multi[0].ID != 901 || multi[1].ID != 902 {
		t.Fatalf("多技能顺序/ID 不对：%+v", multi)
	}
	// 多技能里混了不存在的 ID → 只跳过不存在那个
	restoreG4 := stubGeneralCfg(t, model.EzfyCfgGeneral{ID: 9904, Name: "测试名将3", Kind: 2, ComboSkillIds: "[901,999]"})
	defer restoreG4()
	skip := ezfyComboOfGeneralCfg(ezfyCfg.general(9904), 100)
	if len(skip) != 1 || skip[0].ID != 901 {
		t.Fatalf("不存在/下架的 ID 应跳过：%+v", skip)
	}
	// ★ 2026-10-24 旧字段回退：combo_skill_ids 为空、只有 combo_skill_id → 按旧字段读一个
	restoreG5 := stubGeneralCfg(t, model.EzfyCfgGeneral{ID: 9905, Name: "测试名将-旧字段", Kind: 2, ComboSkillId: 901})
	defer restoreG5()
	legacy := ezfyComboOfGeneralCfg(ezfyCfg.general(9905), 350)
	if len(legacy) != 1 || legacy[0].ID != 901 || legacy[0].Level != 10 {
		t.Fatalf("旧字段 combo_skill_id 回退失败：%+v", legacy)
	}
	// 下架后不再生效
	s := ezfyCfg.generalSkill(901)
	s.Enabled = 0
	restorePool3 := stubComboPool(t, *s)
	defer restorePool3()
	if len(ezfyComboOfGeneralCfg(ezfyCfg.general(9901), 5)) != 0 {
		t.Fatal("下架（enabled=0）的组合技不该生效")
	}
}

// TestComboProcAndBonuses 概率发动 + 攻击加成真的进伤害
func TestComboProcAndBonuses(t *testing.T) {
	defer stubCritTroops(t)()
	build := func(combo *ezfyComboSkill) *ezfyBattleState {
		st := ezfyNewBattleState(
			[]ezfyUnitGroup{{TroopId: 11, Count: 100}},
			[]ezfyUnitGroup{{TroopId: 11, Count: 100}},
			0, 0, 0, 0, 0, 0, 0,
			ezfyBattleBonus{}, ezfyBattleBonus{},
			ezfyBattleBonus{}, ezfyBattleBonus{}, "", "",
			"", "", 0, 0, 0, 0,
			nil, nil, nil, nil, nil,
			0, nil,
			map[int]int{}, map[int]int{}, map[int]int{}, map[int]int{},
			0, 0, 1, 2,
			ezfyTypeBonus{}, ezfyTypeBonus{})
		st.AtkCombos = comboRuntimes([]*ezfyComboSkill{combo})
		return st
	}
	// ① 100% 发动 → 第 1 回合就生效并播报
	st := build(&ezfyComboSkill{ID: 1, Name: "装甲突击", Kind: ezfyComboAtk, Level: 10, Value: 180, ProcPct: 100, Rounds: 1})
	st.Step(nil, nil)
	if st.AtkCombos[0].Left != 0 { // 本回合生效，回合末递减回 0
		t.Fatalf("1 回合技能应在回合末递减回 0, 实际 %d", st.AtkCombos[0].Left)
	}
	if !strings.Contains(strings.Join(st.Actions, "\n"), "【装甲突击】") {
		t.Fatalf("100%% 发动率却没播报：%v", st.Actions)
	}
	// ★ 2026-10-23 用户反馈：发动播报必须排在「第N回合:」之后、部队行动之前，
	//   否则玩家以为技能没发动（原来被算到上一回合末尾）。
	if len(st.Actions) < 2 || st.Actions[0] != "第1回合:" {
		t.Fatalf("回合标题必须是本回合第一行：%v", st.Actions)
	}
	if !strings.Contains(st.Actions[1], "发动组合技【装甲突击】") {
		t.Fatalf("组合技播报应紧跟回合标题：%v", st.Actions[:2])
	}
	if !strings.Contains(st.Actions[1], "全军攻击+180%") {
		t.Fatalf("播报要写清效果（不能只写 +180%%）：%s", st.Actions[1])
	}

	// ② 0% 发动 → 永不播报
	st2 := build(&ezfyComboSkill{ID: 1, Name: "装甲突击", Kind: ezfyComboAtk, Level: 5, Value: 16, ProcPct: 0, Rounds: 1})
	for i := 0; i < 3; i++ {
		st2.Step(nil, nil)
	}
	if strings.Contains(strings.Join(st2.Actions, "\n"), "【装甲突击】") {
		t.Fatal("0% 发动率不该发动")
	}

	// ②b 准备回合要写「组合技」说明（玩家开打前就知道带了什么）
	//    ★ 2026-10-23 改成「并进军官行」；没有军官行时退回单独一行、插在准备回合: 之后
	st4 := build(&ezfyComboSkill{ID: 6, Name: "穿甲火力", Kind: ezfyComboBreak, Level: 10, Value: 80, ProcPct: 35, Rounds: 1})
	st4.appendComboHead()
	head4 := strings.Join(st4.Head, "\n")
	if !strings.Contains(head4, "穿甲火力(Lv.10 破防80%，发动35%)") {
		t.Fatalf("准备回合缺组合技说明：%v", st4.Head)
	}
	if strings.HasPrefix(head4, "【攻方组合技】") { // 必须落在「准备回合:」之后
		t.Fatalf("组合技行不该排在准备回合: 之前：%v", st4.Head)
	}

	// ③ 持续 2 回合：第 1 回合发动后，第 2 回合仍生效
	st3 := build(&ezfyComboSkill{ID: 10, Name: "全线总攻", Kind: ezfyComboAtk, Level: 5, Value: 14, ProcPct: 100, Rounds: 2})
	st3.Step(nil, nil)
	if st3.AtkCombos[0].Left != 1 {
		t.Fatalf("2 回合技能第 1 回合末应剩 1 回合, 实际 %d", st3.AtkCombos[0].Left)
	}
	st3.Step(nil, nil)
	if st3.AtkCombos[0].Left != 0 {
		t.Fatalf("第 2 回合末应归零, 实际 %d", st3.AtkCombos[0].Left)
	}
}

// TestComboHealAndBreak 回合治疗按已损比例回兵；破防让同等兵力打出更多伤害。
func TestComboHealAndBreak(t *testing.T) {
	// ① 治疗：100 → 50，已损 50，治疗 12% = 6
	units := []*ezfyFightUnit{{count: 50, initialCount: 100}}
	if n := ezfyComboHealSide(units, 12); n != 6 {
		t.Fatalf("已损 50、治疗 12%% 应回 6, 实际 %d", n)
	}
	if units[0].count != 56 {
		t.Fatalf("治疗后应为 56, 实际 %d", units[0].count)
	}
	// 损失不足 1 个也要回 1（避免「发动了但什么都没发生」）
	one := []*ezfyFightUnit{{count: 99, initialCount: 100}}
	if n := ezfyComboHealSide(one, 1); n != 1 {
		t.Fatalf("有损失至少回 1, 实际 %d", n)
	}

	// ② 破防：同一攻防下，防御被压 20% 伤害必须更高
	full := ezfyCalcDamage(100, 50, 100, 0, 0)
	broken := ezfyCalcDamage(100, 50*(100-20)/100, 100, 0, 0)
	if broken <= full {
		t.Fatalf("破防 20%% 后伤害应更高：原 %d, 破防 %d", full, broken)
	}
}

// TestComboHeadMergedIntoOfficer 准备回合：组合技**并进军官行**（格式对齐军官技能），
// 不再单独占一行、也不该掉到「战场初始相距」之后（2026-10-23 用户反馈）。
func TestComboHeadMergedIntoOfficer(t *testing.T) {
	st := &ezfyBattleState{
		Head: []string{
			"准备回合:",
			"【攻方军官】斯大林 Lv.350 攻击加成+477% 尖兵猛攻(Lv.6 攻击力+180%)",
			"【守方军官】杰里米·贝克 Lv.1 攻击加成+18%",
			"战场初始相距6000, 攻守双方相向推进",
		},
		AtkCombos: comboRuntimes([]*ezfyComboSkill{{ID: 1, Name: "装甲突击", Kind: ezfyComboAtk, Level: 10, Value: 180, ProcPct: 40, Rounds: 1}}),
		DefCombos: comboRuntimes([]*ezfyComboSkill{{ID: 2, Name: "纵深防御", Kind: ezfyComboDef, Level: 3, Value: 54, ProcPct: 35, Rounds: 1}}),
	}
	st.appendComboHead()
	if len(st.Head) != 4 {
		t.Fatalf("应并进已有行、不新增行：%v", st.Head)
	}
	if !strings.HasSuffix(st.Head[1], "装甲突击(Lv.10 攻击力+180%，发动40%)") {
		t.Fatalf("攻方军官行没带上组合技：%s", st.Head[1])
	}
	if !strings.HasSuffix(st.Head[2], "纵深防御(Lv.3 防御力+54%，发动35%)") {
		t.Fatalf("守方军官行没带上组合技：%s", st.Head[2])
	}
	if !strings.HasPrefix(st.Head[3], "战场初始相距") {
		t.Fatalf("最后一行应仍是战场初始相距：%v", st.Head)
	}
}

// TestComboEffectText 详情页「特殊技能：」一行的文案格式（2026-10-25 用户要求）。
//
//	特殊技能：[装甲突击]：+攻击力180%
//
// 要点：加号在最前、属性名在数值前、百分号在最后；9 种效果类型都不能掉回裸 `+180%`
// （否则玩家看不出加的是什么），次数型的反击用「回合」不用 `%`。
func TestComboEffectText(t *testing.T) {
	cases := []struct {
		kind int
		val  int
		want string
	}{
		{ezfyComboAtk, 180, "+攻击力180%"},
		{ezfyComboDef, 180, "+防御力180%"},
		{ezfyComboRange, 180, "+射程180%"},
		{ezfyComboSpeed, 180, "+速度180%"},
		{ezfyComboReduce, 80, "+受伤减免80%"},
		{ezfyComboBreak, 80, "+破防80%"},
		{ezfyComboHeal, 60, "+治疗60%"},
		{ezfyComboDouble, 180, "+连击伤害180%"},
		{ezfyComboCounter, 5, "+反击5回合"},
	}
	for _, c := range cases {
		got := ezfyComboEffectTxt(&ezfyComboSkill{Kind: c.kind, Value: c.val})
		if got != c.want {
			t.Errorf("kind=%d value=%d 文案应为 %q，实际 %q", c.kind, c.val, c.want, got)
		}
	}
	// 详情接口必须下发 effect_text（前端用 cs.effect_text 渲染这一行）
	b := ezfyComboBrief(&ezfyComboSkill{ID: 1, Name: "装甲突击", Kind: ezfyComboAtk, Level: 10, Value: 180, ProcPct: 40, Rounds: 1})
	if b == nil {
		t.Fatal("ezfyComboBrief 返回 nil")
	}
	if got, _ := b.(gin.H)["effect_text"].(string); got != "+攻击力180%" {
		t.Fatalf("officerComboView 下发的 effect_text 应为 %q，实际 %q", "+攻击力180%", got)
	}
}
