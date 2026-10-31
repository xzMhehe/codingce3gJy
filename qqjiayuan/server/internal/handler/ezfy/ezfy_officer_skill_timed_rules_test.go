package ezfy

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"qqjiayuan/server/internal/model"
)

// ============ 普通军官技能口径护栏（2026-10-31 用户 5 条修正） ============
//
// 用户口径：
//  1. 弧形防御：每 3 回合提升防御力，持续 1 回合（不是常驻）
//  2. 火炮控制：每 3 回合提升**陆军装甲部队**（装甲车4/轻坦5/重坦6/突击炮7/火箭8）攻击，
//     持续 1 回合；「攻击力」= 本回合 对地/对空/对海/对防 各 +等级数值
//  3. 坦克突袭：只生效 轻型坦克(5)、重型坦克(6)
//  4. 狼群战术：海军**攻击力**+15%/级（百分比，不是「对海攻击数值」；不分目标）
//  5. 弹幕支援：只对**城防单位**的**攻击范围**加成（不是防御、不是全兵种）

// testSkillOfficer 构造一个带指定技能的军官（Level 90 → 技能 3 级 = 倍率 3）
func testSkillOfficer(skills ...string) *model.EzfyOfficer {
	raw := "["
	for i, s := range skills {
		if i > 0 {
			raw += ","
		}
		raw += `"` + s + `"`
	}
	raw += "]"
	return &model.EzfyOfficer{Name: "测试军官", Level: 90, Skill: raw}
}

// TestOfficerSkillTimedProc 第 1、4、7 … 回合发动，其余回合不发动。
func TestOfficerSkillTimedProc(t *testing.T) {
	h := &EzfyHandler{}
	o := testSkillOfficer("弧形防御", "火炮控制")
	list := h.officerTimedSkills(o)
	if len(list) != 2 {
		t.Fatalf("应返回 2 个间隔技能，得到 %d", len(list))
	}
	for _, s := range list {
		if s.Interval != 3 || s.Duration != 1 {
			t.Fatalf("%s 间隔/持续应为 3/1，得到 %d/%d", s.Name, s.Interval, s.Duration)
		}
		for _, r := range []int{1, 4, 7, 10} {
			if !s.procOn(r) {
				t.Fatalf("%s 第%d回合应发动", s.Name, r)
			}
		}
		for _, r := range []int{2, 3, 5, 6, 8, 9} {
			if s.procOn(r) {
				t.Fatalf("%s 第%d回合不应发动", s.Name, r)
			}
		}
	}
}

// TestOfficerTypeBonusScope 火炮控制只给装甲部队攻击属性；坦克突袭只给轻坦/重坦速度；
// 狼群战术给海军攻击力%（不分目标）；弹幕支援只给城防射程。
func TestOfficerTypeBonusScope(t *testing.T) {
	h := &EzfyHandler{}
	o := testSkillOfficer("火炮控制", "坦克突袭", "狼群战术", "弹幕支援", "四指编队")
	tb := h.officerTypeBonus(o, ezfyTypeBonus{})
	const scale = 3

	// ① 火炮控制：装甲车4 / 轻坦5 / 重坦6 / 突击炮7 / 火箭8 = 10×3
	for _, id := range []int{4, 5, 6, 7, 8} {
		if got := tb.atkFlatTroopOf(id); got != 10*scale {
			t.Fatalf("火炮控制 兵种%d 攻击属性应为 %d，得到 %d", id, 10*scale, got)
		}
	}
	// 非装甲陆军（步兵1/摩托化骑兵2/卡车3/特种兵12）不吃
	for _, id := range []int{1, 2, 3, 12} {
		if got := tb.atkFlatTroopOf(id); got != 0 {
			t.Fatalf("火炮控制 不应作用于兵种%d，得到 +%d", id, got)
		}
	}
	// 也不再按「陆军 type」整档下发（原来 AtkFlat[2] 全军陆军都吃）
	if got := tb.atkFlatOf(ezfyTroopTypeArmy); got != 0 {
		t.Fatalf("火炮控制 不应再按陆军 type 下发，得到 %d", got)
	}

	// ② 坦克突袭：只轻坦5 / 重坦6
	if tb.speedTroopOf(5) != 10*scale || tb.speedTroopOf(6) != 10*scale {
		t.Fatalf("坦克突袭应只作用于轻坦/重坦，得到 %d/%d", tb.speedTroopOf(5), tb.speedTroopOf(6))
	}
	for _, id := range []int{1, 2, 3, 4, 7, 8} {
		if got := tb.speedTroopOf(id); got != 0 {
			t.Fatalf("坦克突袭 不应作用于兵种%d，得到 +%d", id, got)
		}
	}
	if got := tb.speedOf(ezfyTroopTypeArmy); got != 0 {
		t.Fatalf("坦克突袭 不应再按陆军 type 下发，得到 %d", got)
	}

	// ③ 狼群战术：海军攻击力% +15×3（不分目标）
	if got := tb.atkOf(ezfyTroopTypeNavy); got != 15*scale {
		t.Fatalf("狼群战术 海军攻击力应为 +%d%%，得到 +%d%%", 15*scale, got)
	}
	if got := tb.atkVsType(ezfyTroopTypeNavy, ezfyTroopTypeNavy); got != 0 {
		t.Fatalf("狼群战术 不应再走「对海」目标档，得到 +%d%%", got)
	}

	// ④ 弹幕支援：城防射程 10×3，其它兵种 0
	if got := tb.rangeOf(ezfyTroopTypeCity); got != 10*scale {
		t.Fatalf("弹幕支援 城防射程应为 +%d%%，得到 +%d%%", 10*scale, got)
	}
	for _, ty := range []int{ezfyTroopTypeNavy, ezfyTroopTypeArmy, ezfyTroopTypeAir} {
		if got := tb.rangeOf(ty); got != 0 {
			t.Fatalf("弹幕支援 不应作用于兵种type%d 射程，得到 +%d%%", ty, got)
		}
	}
}

// TestOfficerGuardBonusNoBarrage 弹幕支援不再算进「防御加成」（它是射程技能）。
func TestOfficerGuardBonusNoBarrage(t *testing.T) {
	h := &EzfyHandler{}
	// 用技能明细段判定（officerGuardBonus 会读库算学识，单测不连库）
	for _, it := range h.officerGuardSkillsBreak(testSkillOfficer("弹幕支援")) {
		if it.Name == "弹幕支援" {
			t.Fatalf("弹幕支援 不应再进防御加成明细")
		}
	}
	got := 0
	for _, it := range h.officerGuardSkillsBreak(testSkillOfficer("弧形防御", "弹幕支援")) {
		if it.Name == "弹幕支援" {
			t.Fatalf("弹幕支援 不应再进防御加成明细")
		}
		got += it.Value
	}
	if got != 30*3 {
		t.Fatalf("防御技能段应只有弧形防御 +%d%%，得到 +%d%%", 30*3, got)
	}
}

// ---------- 战斗引擎：按回合生效 ----------

var ezfyTestRoundRe = regexp.MustCompile(`^第(\d+)回合:`)
var ezfyTestDmgRe = regexp.MustCompile(`造成(\d+)伤害`)

var ezfyTestAtkLineRe = regexp.MustCompile(`^【攻方】`)

// ezfyTestAtkDamageByRound 取「攻方」每个回合第一笔攻击伤害
func ezfyTestAtkDamageByRound(actions []string) map[int]int64 {
	out := map[int]int64{}
	round := 0
	for _, ln := range actions {
		if m := ezfyTestRoundRe.FindStringSubmatch(ln); m != nil {
			round, _ = strconv.Atoi(m[1])
			continue
		}
		if round == 0 || !ezfyTestAtkLineRe.MatchString(ln) {
			continue
		}
		if _, ok := out[round]; ok {
			continue
		}
		if m := ezfyTestDmgRe.FindStringSubmatch(ln); m != nil {
			v, _ := strconv.ParseInt(m[1], 10, 64)
			out[round] = v
		}
	}
	return out
}

// ezfyTestRunRounds 跑两回合，返回完整战斗日志
//
// 攻守都是「重型坦克 1000」（陆军，id=6）。攻击力给到 2000，保证伤害远高于
// 「最小伤害 = 数量×5」的兜底线 —— 否则两回合都会被兜底成同一个数，测不出加成差异。
func ezfyTestRunRounds(t *testing.T, atkTB, defTB ezfyTypeBonus,
	atkTimed, defTimed []*ezfyTimedSkill, defBonus int) []string {
	t.Helper()
	saved := ezfyCfg.troops
	defer func() { ezfyCfg.troops = saved }()
	ezfyCfg.troops = map[int]model.EzfyCfgTroop{
		6: {ID: 6, Name: "重型坦克", Type: 2, Health: 400,
			AtkSea: 100, AtkGround: 2000, AtkAir: 80, AtkDef: 100, Defence: 100, Speed: 700, AttackRange: 99999},
	}
	st := ezfyNewBattleState([]ezfyUnitGroup{{TroopId: 6, Count: 1000}},
		[]ezfyUnitGroup{{TroopId: 6, Count: 1000}},
		0, defBonus, 0, 0, 0, 0, 0,
		ezfyBattleBonus{}, ezfyBattleBonus{},
		ezfyBattleBonus{}, ezfyBattleBonus{}, "", "", "", "",
		0, 0, 0, 0,
		nil, nil, nil, nil, nil, 0, nil,
		map[int]int{}, map[int]int{}, map[int]int{}, map[int]int{},
		0, 0, 0, 0, atkTB, defTB)
	st.AtkTimed, st.DefTimed = atkTimed, defTimed
	st.Step(nil, nil)
	st.Step(nil, nil)
	return st.Actions
}

// ezfyTestAtkLineByRound 取「攻方」每个回合第一笔攻击行（含加成说明）
func ezfyTestAtkLineByRound(actions []string) map[int]string {
	out := map[int]string{}
	round := 0
	for _, ln := range actions {
		if m := ezfyTestRoundRe.FindStringSubmatch(ln); m != nil {
			round, _ = strconv.Atoi(m[1])
			continue
		}
		if round == 0 || !ezfyTestAtkLineRe.MatchString(ln) || !strings.Contains(ln, "造成") {
			continue
		}
		if _, ok := out[round]; ok {
			continue
		}
		out[round] = ln
	}
	return out
}

// TestArcDefenseOnlyOnProcRounds 弧形防御只在第 1 回合（发动回合）减伤。
func TestArcDefenseOnlyOnProcRounds(t *testing.T) {
	h := &EzfyHandler{}
	o := testSkillOfficer("弧形防御")
	defBonus := 30 * h.officerSkillScale(o) // 军官技能段（学识那截单测不连库，与口径无关）
	timed := h.officerTimedSkills(o)
	dmg := ezfyTestAtkDamageByRound(ezfyTestRunRounds(t, ezfyTypeBonus{}, ezfyTypeBonus{}, nil, timed, defBonus))
	if dmg[1] == 0 || dmg[2] == 0 {
		t.Fatalf("两回合都要有攻方伤害行，得到 %+v", dmg)
	}
	if dmg[2] <= dmg[1] {
		t.Fatalf("第2回合（未发动）应比第1回合（发动）打得更疼：第1回合=%d 第2回合=%d", dmg[1], dmg[2])
	}
	// 扣减函数口径：未发动回合整段扣掉，发动回合不扣
	if off := ezfyTimedDefOff(timed, 2); off != defBonus {
		t.Fatalf("第2回合应扣掉弧形防御 +%d%%，得到 +%d%%", defBonus, off)
	}
	if off := ezfyTimedDefOff(timed, 1); off != 0 {
		t.Fatalf("第1回合（发动）不应扣减，得到 +%d%%", off)
	}
}

// TestArtilleryControlOnlyOnProcRounds 火炮控制只在发动回合给装甲部队加攻击属性。
//
// 攻击属性是**绝对值**（+10/级），在千级攻击力下不足以改变整数除法后的伤害，
// 所以直接断言战报里的「兵种攻击+N」标记：发动回合有、未发动回合没有。
func TestArtilleryControlOnlyOnProcRounds(t *testing.T) {
	h := &EzfyHandler{}
	o := testSkillOfficer("火炮控制")
	tb := h.officerTypeBonus(o, ezfyTypeBonus{})
	timed := h.officerTimedSkills(o)
	lines := ezfyTestAtkLineByRound(ezfyTestRunRounds(t, tb, ezfyTypeBonus{}, timed, nil, 0))
	if !strings.Contains(lines[1], "兵种攻击+30") {
		t.Fatalf("第1回合（发动）应带「兵种攻击+30」：%s", lines[1])
	}
	if strings.Contains(lines[2], "兵种攻击") {
		t.Fatalf("第2回合（未发动）不应再带兵种攻击加成：%s", lines[2])
	}
	// 扣减函数口径：非发动回合按兵种 ID 全额扣掉
	off := ezfyTimedAtkFlatOff(timed, 2)
	for _, id := range []int{4, 5, 6, 7, 8} {
		if off[id] != 30 {
			t.Fatalf("第2回合兵种%d 应扣掉 30，得到 %d", id, off[id])
		}
	}
	if len(ezfyTimedAtkFlatOff(timed, 1)) != 0 {
		t.Fatalf("第1回合（发动）不应扣减")
	}
}

// TestBarrageRangeOnlyCity 弹幕支援只加城防单位的射程：普通兵种射程不变。
func TestBarrageRangeOnlyCity(t *testing.T) {
	saved := ezfyCfg.troops
	defer func() { ezfyCfg.troops = saved }()
	ezfyCfg.troops = map[int]model.EzfyCfgTroop{
		1: {ID: 1, Name: "步兵", Type: 2, Health: 100,
			AtkSea: 10, AtkGround: 30, AtkAir: 5, AtkDef: 20, Defence: 30, Speed: 500, AttackRange: 100},
		20: {ID: 20, Name: "防空炮", Type: 4, Health: 200,
			AtkSea: 10, AtkGround: 20, AtkAir: 90, AtkDef: 60, Defence: 80, Speed: 300, AttackRange: 100},
	}
	tb := ezfyTypeBonus{}
	tb.Range = map[int]int{ezfyTroopTypeCity: 30}
	// 城防：射程 100 → 130；步兵：仍 100
	if got := 100 * (100 + tb.rangeOf(ezfyTroopTypeCity)) / 100; got != 130 {
		t.Fatalf("城防射程应为 130，得到 %d", got)
	}
	if got := 100 * (100 + tb.rangeOf(ezfyTroopTypeArmy)) / 100; got != 100 {
		t.Fatalf("步兵射程应保持 100，得到 %d", got)
	}
}

// TestWolfPackNavyAttackPct 狼群战术是「海军攻击力%」——打陆军目标也生效（不是只对海）。
func TestWolfPackNavyAttackPct(t *testing.T) {
	h := &EzfyHandler{}
	tb := h.officerTypeBonus(testSkillOfficer("狼群战术"), ezfyTypeBonus{})
	armyTarget := &ezfyTroopStats{Type: ezfyTroopTypeArmy}
	if got := tb.attackBonusFor(ezfyTroopTypeNavy, armyTarget); got != 15*3 {
		t.Fatalf("狼群战术 海军打陆军也应 +%d%%，得到 +%d%%", 15*3, got)
	}
	airTarget := &ezfyTroopStats{Type: ezfyTroopTypeAir}
	if got := tb.attackBonusFor(ezfyTroopTypeNavy, airTarget); got != 15*3 {
		t.Fatalf("狼群战术 海军打空军也应 +%d%%，得到 +%d%%", 15*3, got)
	}
	if got := tb.attackBonusFor(ezfyTroopTypeArmy, armyTarget); got != 0 {
		t.Fatalf("狼群战术 不应作用于陆军，得到 +%d%%", got)
	}
}
