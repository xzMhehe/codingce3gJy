package ezfy

import (
	"strings"
	"testing"

	"qqjiayuan/server/internal/model"
)

// ============ 2026-10-31 军官技能「速度 → 攻速」 ============
//
// 用户需求：
//
//	闪电疾攻 空军速度+% → **攻速**加成%；越岛登陆 海军速度+% → **攻速**；坦克冲锋 坦克速度+% → **攻速**。
//
// 口径（用户确认）：攻速 = **先手 / 出手顺序**，只决定同回合内的行动先后；
// 不加移动距离（每回合移动仍只看 speedBonus），也不进行军速度（ezfyMarchSpeed 只吃科技）。

// stubAtkSpeedTroops 攻速用例的兵种配置（速度/类型按真实 ID 取值，便于断言）。
func stubAtkSpeedTroops(t *testing.T) func() {
	t.Helper()
	saved := ezfyCfg.troops
	ezfyCfg.troops = map[int]model.EzfyCfgTroop{
		1: {ID: 1, Name: "步兵", Type: 2, Health: 100, Speed: 300, AttackRange: 99999},
		// ★ 2026-10-31 独立攻速用例：移动速度 1400，基础攻速 700 → 出手只看 700。
		2:  {ID: 2, Name: "摩托化骑兵", Type: 2, Health: 65, Speed: 1400, AtkSpeed: 700, AttackRange: 99999},
		5:  {ID: 5, Name: "轻型坦克", Type: 2, Health: 200, Speed: 500, AttackRange: 99999},
		6:  {ID: 6, Name: "重型坦克", Type: 2, Health: 300, Speed: 400, AttackRange: 99999},
		11: {ID: 11, Name: "轰炸机", Type: 3, Health: 135, Speed: 1000, AttackRange: 99999},
		13: {ID: 13, Name: "驱逐舰", Type: 1, Health: 300, Speed: 800, AttackRange: 99999},
	}
	return func() { ezfyCfg.troops = saved }
}

// TestEffAtkSpeed 有效攻速 = 兵种速度 × (1 + 攻速加成)，按兵种 type / 兵种 ID 分别生效。
func TestEffAtkSpeed(t *testing.T) {
	defer stubAtkSpeedTroops(t)()
	st := &ezfyBattleState{
		AtkType: ezfyTypeBonus{
			AtkSpeed:      map[int]int{ezfyTroopTypeAir: 50}, // 闪电疾攻 → 空军 +50%
			AtkSpeedTroop: map[int]int{5: 100, 6: 100},       // 坦克冲锋 → 轻坦/重坦 +100%
		},
		DefType: ezfyTypeBonus{AtkSpeed: map[int]int{ezfyTroopTypeNavy: 50}}, // 越岛登陆 → 海军 +50%
	}
	unit := func(id int) *ezfyFightUnit { return &ezfyFightUnit{cfg: ezfyStatsOf(id)} }

	cases := []struct {
		id    int
		isAtk bool
		want  int
	}{
		{11, true, 1500},  // 空军 1000 × 1.5（闪电疾攻）
		{5, true, 1000},   // 轻坦 500 × 2（坦克冲锋）
		{6, true, 800},    // 重坦 400 × 2（坦克冲锋）
		{1, true, 300},    // 步兵不吃任何攻速
		{2, true, 700},    // 独立攻速 700（不是移动速度 1400）→ 证明 atk_speed 真的生效
		{11, false, 1000}, // 守方没有空军攻速
		{13, false, 1200}, // 守方海军 800 × 1.5（越岛登陆）
	}
	for _, c := range cases {
		if got := st.effAtkSpeed(unit(c.id), c.isAtk); got != c.want {
			t.Fatalf("兵种%d(攻方=%v) 有效攻速 = %d，期望 %d", c.id, c.isAtk, got, c.want)
		}
	}
	// 老战场快照没有 AtkSpeed 字段 → 有效攻速 = 基础速度（行为与改动前一致）
	old := &ezfyBattleState{}
	if got := old.effAtkSpeed(unit(11), true); got != 1000 {
		t.Fatalf("无攻速加成的老快照应等于基础速度 1000，实际 %d", got)
	}
}

// TestAtkSpeedDecidesTurnOrder 攻速高的先出手（先手），与「兵种基础速度」不同。
//
// 构造：双方都是同一兵种（基础速度相同 → 老逻辑下攻方恒定先手），
// 守方带 +100% 攻速 → 守方应抢在攻方之前行动。
func TestAtkSpeedDecidesTurnOrder(t *testing.T) {
	defer stubAtkSpeedTroops(t)()
	build := func(defType ezfyTypeBonus) *ezfyBattleState {
		return ezfyNewBattleState(
			[]ezfyUnitGroup{{TroopId: 11, Count: 1000}},
			[]ezfyUnitGroup{{TroopId: 11, Count: 1000}},
			0, 0, 0, 0, 0,
			0, 0,
			ezfyBattleBonus{}, ezfyBattleBonus{},
			ezfyBattleBonus{}, ezfyBattleBonus{}, "", "",
			"", "", 0, 0, 0, 0,
			nil, nil, nil, nil, nil,
			0, nil,
			map[int]int{}, map[int]int{}, map[int]int{}, map[int]int{},
			0, 0, 1, 2,
			ezfyTypeBonus{}, defType)
	}
	firstHitSide := func(st *ezfyBattleState) string {
		st.Step(nil, nil)
		for _, ln := range st.Actions {
			if strings.HasPrefix(ln, "【攻方】") || strings.HasPrefix(ln, "【守方】") {
				return ln
			}
		}
		return ""
	}

	// 基线：双方无攻速 → 稳定排序保留「攻方在前」
	base := firstHitSide(build(ezfyTypeBonus{}))
	if !strings.HasPrefix(base, "【攻方】") {
		t.Fatalf("无攻速加成时应攻方先手，实际：%s", base)
	}
	// 守方 +100% 攻速 → 守方先手
	fast := firstHitSide(build(ezfyTypeBonus{AtkSpeed: map[int]int{ezfyTroopTypeAir: 100}}))
	if !strings.HasPrefix(fast, "【守方】") {
		t.Fatalf("守方攻速 +100%% 应抢到先手，实际：%s", fast)
	}
}

// TestBaseAtkSpeedDecidesOrder 兵种基础攻速（atk_speed）本身就要决定先手，不靠军官加成。
//
// 构造：攻方海军轰炸机 Speed=1500 / AtkSpeed=880，守方轰炸机 Speed=1000 / AtkSpeed=2000。
// 按「速度」攻方 1500 > 1000 应先手；按「攻速」守方 2000 > 880 应先手 —— 断言守方先手。
func TestBaseAtkSpeedDecidesOrder(t *testing.T) {
	saved := ezfyCfg.troops
	defer func() { ezfyCfg.troops = saved }()
	ezfyCfg.troops = map[int]model.EzfyCfgTroop{
		21: {ID: 21, Name: "海军轰炸机", Type: 3, Health: 130, Speed: 1500, AtkSpeed: 880, AttackRange: 99999},
		11: {ID: 11, Name: "轰炸机", Type: 3, Health: 135, Speed: 1000, AtkSpeed: 2000, AttackRange: 99999},
	}
	st := ezfyNewBattleState(
		[]ezfyUnitGroup{{TroopId: 21, Count: 100}},
		[]ezfyUnitGroup{{TroopId: 11, Count: 100}},
		0, 0, 0, 0, 0,
		0, 0,
		ezfyBattleBonus{}, ezfyBattleBonus{},
		ezfyBattleBonus{}, ezfyBattleBonus{}, "", "",
		"", "", 0, 0, 0, 0,
		nil, nil, nil, nil, nil,
		0, nil,
		map[int]int{}, map[int]int{}, map[int]int{}, map[int]int{},
		0, 0, 1, 2,
		ezfyTypeBonus{}, ezfyTypeBonus{})
	st.Step(nil, nil)
	first := ""
	for _, ln := range st.Actions {
		if strings.HasPrefix(ln, "【攻方】") || strings.HasPrefix(ln, "【守方】") {
			first = ln
			break
		}
	}
	if !strings.HasPrefix(first, "【守方】") {
		t.Fatalf("守方轰炸机攻速 2000 > 海轰 880，应守方先手；实际：%s\n%s",
			first, strings.Join(st.Actions, "\n"))
	}
}

// TestAtkSpeedNotInSpeedBuckets 攻速不得混进「速度」档（否则又变成加移动/行军速度）。
func TestAtkSpeedNotInSpeedBuckets(t *testing.T) {
	h := &EzfyHandler{}
	o := &model.EzfyOfficer{Name: "T", Level: 150, Skill: `["坦克冲锋","闪电疾攻","越岛登陆"]`}
	tb := h.officerTypeBonus(o, ezfyTypeBonus{})

	// 攻速：轻坦/重坦（兵种 ID）+ 空军/海军（兵种 type）
	if tb.atkSpeedTroopOf(5) != 50 || tb.atkSpeedTroopOf(6) != 50 {
		t.Fatalf("坦克冲锋应给轻坦/重坦 +50%% 攻速，实际 %d/%d",
			tb.atkSpeedTroopOf(5), tb.atkSpeedTroopOf(6))
	}
	if tb.atkSpeedOf(ezfyTroopTypeAir) != 50 || tb.atkSpeedOf(ezfyTroopTypeNavy) != 50 {
		t.Fatalf("闪电疾攻/越岛登陆应给空/海 +50%% 攻速，实际 %d/%d",
			tb.atkSpeedOf(ezfyTroopTypeAir), tb.atkSpeedOf(ezfyTroopTypeNavy))
	}
	// 速度档必须为空（这三项技能已不再加移动/行军速度）
	if len(tb.Speed) != 0 || len(tb.SpeedTroop) != 0 {
		t.Fatalf("速度档应为空（已改攻速），实际 Speed=%v SpeedTroop=%v", tb.Speed, tb.SpeedTroop)
	}
}

// TestTypeAtkSpeedTxt 战报「战斗加成」行要单列兵种专属攻速，且只列本方真正带了的兵种。
func TestTypeAtkSpeedTxt(t *testing.T) {
	defer stubAtkSpeedTroops(t)()
	tb := ezfyTypeBonus{
		AtkSpeed:      map[int]int{ezfyTroopTypeAir: 50}, // 闪电疾攻
		AtkSpeedTroop: map[int]int{5: 50, 6: 50},         // 坦克冲锋（只列带了的那支）
	}
	units := []ezfyUnitGroup{{TroopId: 11, Count: 10}, {TroopId: 5, Count: 10}}
	if got, want := ezfyTypeAtkSpeedTxt(tb, units), " 兵种专属攻速(空军+50% 轻型坦克+50%)"; got != want {
		t.Fatalf("展示文案 = %q，期望 %q", got, want)
	}
	// 没带空军/坦克 → 不展示（避免「凭空多了攻速」的歧义）
	if got := ezfyTypeAtkSpeedTxt(tb, []ezfyUnitGroup{{TroopId: 1, Count: 10}}); got != "" {
		t.Fatalf("没带对应兵种时不该展示攻速加成，实际 %q", got)
	}
	// 老快照（无攻速）→ 空串，老战报格式不变
	if got := ezfyTypeAtkSpeedTxt(ezfyTypeBonus{}, units); got != "" {
		t.Fatalf("无攻速来源时应为空串，实际 %q", got)
	}
}

// TestTroopAtkSpeedSnapshot 基础攻速要随战场快照往返；老快照没有 atk_speed → 回退 speed。
func TestTroopAtkSpeedSnapshot(t *testing.T) {
	u := ezfyTestUnit("A1", "步兵", 0, 100)
	u.cfg.Speed = 1400
	u.cfg.AtkSpeed = 700
	st := &ezfyBattleState{
		Attackers: []*ezfyFightUnit{u},
		Defenders: []*ezfyFightUnit{ezfyTestUnit("D1", "步兵", 100, 100)},
		Head:      []string{}, Actions: []string{},
	}
	snap := st.Snapshot()
	if snap.Attackers[0].AtkSpeed != 700 {
		t.Fatalf("快照应保存基础攻速 700，实际 %d", snap.Attackers[0].AtkSpeed)
	}
	rebuilt := ezfyBattleStateFromSnapshot(snap)
	if got := rebuilt.effAtkSpeed(rebuilt.Attackers[0], true); got != 700 {
		t.Fatalf("快照重建后攻速应为 700，实际 %d", got)
	}

	// 老快照（加 atk_speed 之前创建，没有该字段）→ 优先读当前配置的 atk_speed，
	// 这样在途老战场也能吃到管理端新调的攻速。
	savedTroops := ezfyCfg.troops
	defer func() { ezfyCfg.troops = savedTroops }()
	ezfyCfg.troops = map[int]model.EzfyCfgTroop{
		1: {ID: 1, Name: "步兵", Type: 2, Speed: 800, AtkSpeed: 880},
	}
	snap.Attackers[0].AtkSpeed = 0
	snap.Attackers[0].Speed = 800
	old := ezfyBattleStateFromSnapshot(snap)
	if got := old.effAtkSpeed(old.Attackers[0], true); got != 880 {
		t.Fatalf("老快照应优先用当前配置 atk_speed=880，实际 %d", got)
	}

	// 当前配置 atk_speed 也是 0（跟随速度）→ 回退快照里的 speed。
	ezfyCfg.troops[1] = model.EzfyCfgTroop{ID: 1, Name: "步兵", Type: 2, Speed: 800, AtkSpeed: 0}
	old2 := ezfyBattleStateFromSnapshot(snap)
	if got := old2.effAtkSpeed(old2.Attackers[0], true); got != 800 {
		t.Fatalf("当前配置 atk_speed=0 时应回退快照 speed=800，实际 %d", got)
	}
}

// TestTroopAtkSpeedFieldRegistered 管理端「编辑兵种配置」必须能保存 atk_speed。
func TestTroopAtkSpeedFieldRegistered(t *testing.T) {
	if got := ezfyTroopFields["atk_speed"]; got != "int" {
		t.Fatalf("ezfyTroopFields 缺少 atk_speed（管理端保存会丢字段），实际 %q", got)
	}
}

// TestCarryTechMax50 科技 13「标准化装卸」阶梯：每级 +5%、满级 +50%，且负重上限按配置结算。
func TestCarryTechMax50(t *testing.T) {
	if per := ezfyTechEffectPer[13]; per != 5 {
		t.Fatalf("标准化装卸每级应为 +5%%（满级 50%%），实际 +%d%%", per)
	}
	prev := 0
	for lv := 1; lv <= 10; lv++ {
		v := ezfyTechEffectValueAt(13, lv)
		if v <= prev {
			t.Fatalf("标准化装卸 Lv%d 效果 +%d%% 未高于上一级 +%d%%", lv, v, prev)
		}
		prev = v
	}
	if got := ezfyTechEffectValueAt(13, 10); got != 50 {
		t.Fatalf("标准化装卸满级应为 +50%%，实际 +%d%%", got)
	}
	// 实际负重：兵种负重 50 × 10 个 = 500，满级科技 → 750
	saved := ezfyCfg.troops
	defer func() { ezfyCfg.troops = saved }()
	ezfyCfg.troops = map[int]model.EzfyCfgTroop{1: {ID: 1, Name: "步兵", Type: 2, Carry: 50}}
	h := &EzfyHandler{}
	groups := []ezfyUnitGroup{{TroopId: 1, Count: 10}}
	if got := h.ezfyCarryCapOf(groups, 1, map[int]int{13: 10}); got != 750 {
		t.Fatalf("满级标准化装卸负重应为 750，实际 %d", got)
	}
	if got := h.ezfyCarryCapOf(groups, 1, map[int]int{13: 0}); got != 500 {
		t.Fatalf("没研究标准化装卸负重应为 500，实际 %d", got)
	}
}
