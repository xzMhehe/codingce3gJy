package ezfy

import (
	"strings"
	"testing"

	"qqjiayuan/server/internal/model"
)

// ============ 2026-10-09 用户反馈批次 ============
//
// ① 科技的移动速度加成有兵种限制 → 战斗加成行要按兵种把专属速度列出来；
// ② 有些装备爆击不是 100%，发动爆击也要有提示（像军官发动技能那样）；
// ③ 军官反击的技能不会触发暴击；
// ④ 狼群战术只对「海军打海军」生效；
// ⑤ 机械改造：出征油耗降低 + 伤兵回收。

// stubCritTroops 给测试用的兵种配置（高射程保证开局即可交火；OilKeep 供油耗用例）。
func stubCritTroops(t *testing.T) func() {
	t.Helper()
	saved := ezfyCfg.troops
	ezfyCfg.troops = map[int]model.EzfyCfgTroop{
		11: {ID: 11, Name: "轰炸机", Type: 3, Health: 135,
			AtkSea: 60, AtkGround: 90, AtkAir: 70, Defence: 40, Speed: 1200,
			AttackRange: 99999, OilKeep: 100},
	}
	return func() { ezfyCfg.troops = saved }
}

// TestBonusLineShowsTypeSpeed 战斗加成行要拆解「移动速度」来源，并按兵种列专属加成。
//
// ★★ 2026-10-09 用户口径：「速度」与「移动距离加成」都是加成**移动速度**的
// （装备的移动距离% = 移动速度，不是攻速；攻速决定先手，是另一回事）。
// 汇总行要能看出 速度 = 科技(燃烧引擎) + 装备(移动距离) + 兵种专属(喷气引擎/军官技能)。
func TestBonusLineShowsTypeSpeed(t *testing.T) {
	defer stubCritTroops(t)()
	st := ezfyNewBattleState(
		[]ezfyUnitGroup{{TroopId: 11, Count: 100}}, // 兵种 11 = 空军
		[]ezfyUnitGroup{{TroopId: 11, Count: 100}},
		0, 0, 0, 20, 0, // 攻方通用速度 20%（燃烧引擎）
		0, 0,
		ezfyBattleBonus{}, ezfyBattleBonus{},
		ezfyBattleBonus{}, ezfyBattleBonus{}, "", "",
		"", "", 0, 0, 0, 0,
		nil, nil, nil, nil, nil,
		0, nil,
		map[int]int{}, map[int]int{}, map[int]int{}, map[int]int{},
		0, 0, 1, 2,
		ezfyTypeBonus{Speed: map[int]int{ezfyTroopTypeAir: 30}},
		ezfyTypeBonus{Speed: map[int]int{ezfyTroopTypeNavy: 10, ezfyTroopTypeAir: 30}})

	lines := bonusLinesOf(st.Head)
	if len(lines) != 2 {
		t.Fatalf("「战斗加成」应拆成两行（攻方/守方各一行），实际 %d 行：%v", len(lines), lines)
	}
	if !strings.HasPrefix(lines[0], "战斗加成: 攻方 ") || !strings.HasPrefix(lines[1], "战斗加成: 守方 ") {
		t.Fatalf("两行的顺序/前缀不对（应先攻方后守方）：%v", lines)
	}
	line := strings.Join(lines, "\n")
	// 攻方：通用速度 20（燃烧引擎）+ 兵种专属 30（本方就是空军）
	if !strings.Contains(line, "速度+20%(科技+20) 兵种专属速度(空军+30%)") {
		t.Fatalf("攻方移动速度拆解不对：\n%s", line)
	}
	// 守方：通用 0（没燃烧引擎）→ 不写空拆解；兵种专属只列**本方带了的**空军
	if !strings.Contains(line, "速度+0% 兵种专属速度(空军+30%)") {
		t.Fatalf("守方移动速度拆解不对：\n%s", line)
	}
	if strings.Contains(line, "海军+10%") {
		t.Fatalf("守方没带海军，不该列海军速度加成：\n%s", line)
	}
}

// TestBonusLineNoIrrelevantTypeSpeed 纯海军（航母）部队不该出现「空军+30%」，且移动距离要进拆解。
//
// ★★ 2026-10-09 用户反馈（原战报）：双方都是航母（海军），战斗加成却写「速度+30%(空军+30%)」——
//
//	① 喷气引擎「空军速度」对海军根本不生效，列出来会误导；
//	② 攻方 30% 里那 10% 是套装「移动距离」，拆解里看不出来（「移动加成也不对」）。
func TestBonusLineNoIrrelevantTypeSpeed(t *testing.T) {
	saved := ezfyCfg.troops
	defer func() { ezfyCfg.troops = saved }()
	ezfyCfg.troops = map[int]model.EzfyCfgTroop{
		16: {ID: 16, Name: "航母", Type: 1, Health: 2200, AtkSea: 100, Defence: 100, Speed: 850, AttackRange: 3100},
	}
	st := ezfyNewBattleState(
		[]ezfyUnitGroup{{TroopId: 16, Count: 100}},
		[]ezfyUnitGroup{{TroopId: 16, Count: 100}},
		0, 0, 0, 20, 20, // 燃烧引擎 +20%（双方通用）
		30, 30,
		ezfyBattleBonus{Move: 10}, ezfyBattleBonus{}, // 攻方套装「移动距离+10%」
		ezfyBattleBonus{}, ezfyBattleBonus{}, "", "",
		"", "", 0, 0, 0, 0,
		nil, nil, nil, nil, nil,
		0, nil,
		map[int]int{}, map[int]int{}, map[int]int{}, map[int]int{},
		0, 0, 1, 2,
		ezfyTypeBonus{Speed: map[int]int{ezfyTroopTypeAir: 30}},
		ezfyTypeBonus{Speed: map[int]int{ezfyTroopTypeAir: 30}})

	line := strings.Join(bonusLinesOf(st.Head), "\n")
	t.Logf("战斗加成行：\n%s", line)
	// 攻方 移动速度 = 燃烧 20 + 套装移动 10 = 30，拆解出「装备套装+10」（移动距离）
	if !strings.Contains(line, "速度+30%(科技+20 装备套装+10)") {
		t.Fatalf("攻方移动速度拆解不对（移动距离没体现）：\n%s", line)
	}
	// 守方 移动速度 = 燃烧 20（无装备）
	if !strings.Contains(line, "速度+20%(科技+20)") {
		t.Fatalf("守方移动速度拆解不对：\n%s", line)
	}
	// 双方都是航母（海军）→ 不该出现「空军」
	if strings.Contains(line, "空军") {
		t.Fatalf("纯海军部队不该在汇总行出现空军速度加成：\n%s", line)
	}
}

// bonusLinesOf 取准备回合里的「战斗加成」行（2026-10-09 起拆成攻方/守方两行）。
func bonusLinesOf(head []string) []string {
	out := []string{}
	for _, h := range head {
		if strings.Contains(h, "战斗加成") {
			out = append(out, h)
		}
	}
	return out
}

// TestCritAnnounceAndCounterCrit 暴击要播报；反击同样能触发暴击。
//
// ★★ 2026-10-09 用户反馈：
//
//	② 「有些装备爆击不是 100%，发动爆击也要有提示，就和军官发动技能似的描述下」→
//	   命中暴击时单独播报一行「发动【暴击】！」。
//	③ 「军官反击的技能不会触发暴击注意下」→ 绝地反击原来固定不暴击，同一支部队
//	   「普通攻击会暴击、反击永远不暴击」，反击伤害系统性偏低。现在按反击方自己的
//	   装备 roll 暴击（本用例：守方装备 100% 暴击，攻守双方都必暴）。
func TestCritAnnounceAndCounterCrit(t *testing.T) {
	defer stubCritTroops(t)()
	// 攻方无暴击；守方装备 100% 暴击 / +100% 暴击伤害。
	// DefCounterRounds=5 → 攻方行动时守方反击（反击方=守方 → 用守方装备）。
	st := ezfyNewBattleState(
		[]ezfyUnitGroup{{TroopId: 11, Count: 200000}},
		[]ezfyUnitGroup{{TroopId: 11, Count: 200000}},
		0, 0, 0, 0, 0,
		0, 0,
		ezfyBattleBonus{}, ezfyBattleBonus{Crit: 100, CritDmg: 100},
		ezfyBattleBonus{}, ezfyBattleBonus{}, "", "",
		"", "", 0, 0, 0, 0,
		nil, nil, nil, nil, nil,
		0, nil,
		map[int]int{}, map[int]int{}, map[int]int{}, map[int]int{},
		0, 5, 1, 2,
		ezfyTypeBonus{}, ezfyTypeBonus{})
	for i := 0; i < 4 && !st.Done; i++ {
		st.Step(nil, nil)
	}

	anno, counterCrit, counterLines := 0, 0, 0
	for _, ln := range st.Actions {
		if strings.Contains(ln, "发动【暴击】") {
			anno++
		}
		if strings.Contains(ln, "绝地反击") {
			counterLines++
			if strings.Contains(ln, "【暴击") {
				counterCrit++
			}
		}
	}
	if anno == 0 {
		t.Fatalf("暴击没有发动播报（应有「发动【暴击】！」行）：\n%s", strings.Join(st.Actions, "\n"))
	}
	if counterLines == 0 {
		t.Fatalf("本场没有触发「绝地反击」，用例未覆盖反击路径")
	}
	if counterCrit == 0 {
		t.Fatalf("反击没有触发暴击（守方装备 100%% 暴击，%d 条反击行全部无暴击）：\n%s",
			counterLines, strings.Join(st.Actions, "\n"))
	}
	// 反击暴击也要有播报（「本次反击伤害+N%」）
	if !strings.Contains(strings.Join(st.Actions, "\n"), "本次反击伤害+") {
		t.Fatalf("反击暴击没有播报：\n%s", strings.Join(st.Actions, "\n"))
	}
}

// TestWolfPackOnlyNavyVsNavy 狼群战术只对「海军打海军」生效（四指编队=空军打空军同口径）。
//
// ★★ 2026-10-09 用户反馈「军官技能 狼群战术(Lv.6 海军对海攻击+90%)：这个技能只对海军打海军加成生效」。
func TestWolfPackOnlyNavyVsNavy(t *testing.T) {
	var tb ezfyTypeBonus
	tb.addVsType(ezfyTroopTypeNavy, ezfyTroopTypeNavy, 90) // 狼群战术 Lv.6 = 15% × 6

	navy := &ezfyTroopStats{Type: ezfyTroopTypeNavy}
	army := &ezfyTroopStats{Type: ezfyTroopTypeArmy}
	air := &ezfyTroopStats{Type: ezfyTroopTypeAir}

	// 海军打海军 → 生效
	if got := tb.attackBonusFor(ezfyTroopTypeNavy, navy); got != 90 {
		t.Fatalf("海军打海军应 +90%%，实际 +%d%%", got)
	}
	// 海军打陆军 / 空军 → 不生效
	if got := tb.attackBonusFor(ezfyTroopTypeNavy, army); got != 0 {
		t.Fatalf("海军打陆军不该有狼群战术加成，实际 +%d%%", got)
	}
	if got := tb.attackBonusFor(ezfyTroopTypeNavy, air); got != 0 {
		t.Fatalf("海军打空军不该有狼群战术加成，实际 +%d%%", got)
	}
	// 陆军 / 空军打海军 → 不生效（加成按**攻击方**兵种取）
	if got := tb.attackBonusFor(ezfyTroopTypeArmy, navy); got != 0 {
		t.Fatalf("陆军打海军不该有狼群战术加成，实际 +%d%%", got)
	}
	if got := tb.attackBonusFor(ezfyTroopTypeAir, navy); got != 0 {
		t.Fatalf("空军打海军不该有狼群战术加成，实际 +%d%%", got)
	}
	// def=nil 兜底：只返回通用型（这里没配 → 0）
	if got := tb.attackBonusFor(ezfyTroopTypeNavy, nil); got != 0 {
		t.Fatalf("def=nil 时应只返回通用型加成（0），实际 +%d%%", got)
	}
}

// TestOilReduceByMechanicalSkill 机械改造「出征油耗-10%/级」要真的减油耗。
//
// ★★ 2026-10-09 用户反馈「携带机械改造技能军官：出征油耗降低、伤兵回收加成」。
// 伤兵回收（回收率+10%/级）此前已生效；这里盯的是**油耗减免**，并保证：
//
//	减免在算完原价之后统一打折、最低 1 油、预览与实际下单同口径（同一个函数）。
func TestOilReduceByMechanicalSkill(t *testing.T) {
	defer stubCritTroops(t)()
	h := &EzfyHandler{}
	troops := []ezfyUnitGroup{{TroopId: 11, Count: 30000}} // 油耗 100/个 × 3 万 = 300 万
	const dist = 100

	base := h.ezfyOilCost(nil, 2, dist, troops, nil, 0)
	if base != 3000000*int64(dist)/ezfyOilDivGrid {
		t.Fatalf("基线油耗计算异常：%d", base)
	}
	// 机械改造 Lv.3 → 减免 30%
	got30 := h.ezfyOilCost(nil, 2, dist, troops, nil, 30)
	if want := base * 70 / 100; got30 != want {
		t.Fatalf("机械改造 -30%% 油耗不对：期望 %d，实际 %d", want, got30)
	}
	// 机械改造 Lv.6（名将满级）→ 减免 60%
	got60 := h.ezfyOilCost(nil, 2, dist, troops, nil, 60)
	if want := base * 40 / 100; got60 != want {
		t.Fatalf("机械改造 -60%% 油耗不对：期望 %d，实际 %d", want, got60)
	}
	// 异常输入：减免 > 90% 钳到 90%，且最低保底 1 油
	if got := h.ezfyOilCost(nil, 2, dist, troops, nil, 999); got != base/10 {
		t.Fatalf("超范围减免应钳到 90%%（期望 %d，实际 %d）", base/10, got)
	}
	// 运输(5) 走另一条公式，同样享受减免
	res := map[string]int64{"food": 10000000}
	baseT := h.ezfyOilCost(nil, 5, dist, nil, res, 0)
	if gotT := h.ezfyOilCost(nil, 5, dist, nil, res, 50); gotT != baseT*50/100 {
		t.Fatalf("运输油耗未享受减免：基线 %d，-50%% 后 %d", baseT, gotT)
	}
}

// TestOfficerOilReducePct 只有带「机械改造」的军官才有油耗减免，且按技能等级 ×10%。
func TestOfficerOilReducePct(t *testing.T) {
	h := &EzfyHandler{}
	if got := h.officerOilReducePct(nil); got != 0 {
		t.Fatalf("无军官应无减免，实际 %d", got)
	}
	// 150 级普通军官 → 技能 5 级 → 减免 50%
	o := &model.EzfyOfficer{Name: "Test", Level: 150, Skill: `["机械改造"]`}
	if got := h.officerOilReducePct(o); got != 50 {
		t.Fatalf("150 级机械改造应为 -50%%，实际 %d", got)
	}
	// 不带该技能 → 0
	o2 := &model.EzfyOfficer{Name: "Test2", Level: 150, Skill: `["尖兵突击"]`}
	if got := h.officerOilReducePct(o2); got != 0 {
		t.Fatalf("不带机械改造应无减免，实际 %d", got)
	}
}

// TestMarchSpeedPerTroopType 行军速度按兵种接入科技/军官速度加成。
//
// ★★ 2026-10-09 用户确认「行军也按兵种接入」：燃烧引擎(部队速度)=通用；
// 喷气引擎(空军速度)/军官速度技能(陆·空·海)=只对对应兵种生效；混编取最慢。
func TestMarchSpeedPerTroopType(t *testing.T) {
	saved := ezfyCfg.troops
	defer func() { ezfyCfg.troops = saved }()
	ezfyCfg.troops = map[int]model.EzfyCfgTroop{
		1:  {ID: 1, Name: "步兵", Type: 2, Speed: 1000},   // 陆军
		11: {ID: 11, Name: "轰炸机", Type: 3, Speed: 1000}, // 空军
		13: {ID: 13, Name: "驱逐舰", Type: 1, Speed: 1000}, // 海军
	}
	h := &EzfyHandler{}
	tech := map[int]int{10: 5, 19: 10} // 燃烧引擎 5 级(+10%) / 喷气引擎 10 级(+30%)

	// 纯陆军：只吃燃烧引擎 → 1000×1.10 = 1100
	if got := h.ezfyMarchSpeed([]ezfyUnitGroup{{TroopId: 1, Count: 10}}, tech, nil); got != 1100 {
		t.Fatalf("纯陆军行军速度应为 1100（只吃燃烧引擎），实际 %d", got)
	}
	// 纯空军：燃烧 +10% + 喷气 +30% → 1400
	if got := h.ezfyMarchSpeed([]ezfyUnitGroup{{TroopId: 11, Count: 10}}, tech, nil); got != 1400 {
		t.Fatalf("纯空军行军速度应为 1400（燃烧+喷气），实际 %d", got)
	}
	// 混编（陆军 1100 + 空军 1400）→ 取最慢 1100（慢的拖后腿）
	if got := h.ezfyMarchSpeed([]ezfyUnitGroup{{TroopId: 1, Count: 10}, {TroopId: 11, Count: 10}}, tech, nil); got != 1100 {
		t.Fatalf("混编行军速度应取最慢 1100，实际 %d", got)
	}
	// 军官「越岛战术」（海军速度）带纯陆军 → 不生效（兵种不符）
	navyOfficer := &model.EzfyOfficer{Name: "N", Level: 150, Skill: `["越岛战术"]`}
	if got := h.ezfyMarchSpeed([]ezfyUnitGroup{{TroopId: 1, Count: 10}}, tech, navyOfficer); got != 1100 {
		t.Fatalf("越岛战术带陆军不该加速，实际 %d", got)
	}
	// 同一军官带纯海军 → 生效（通用 +10% + 越岛 50%）→ 1600
	if got := h.ezfyMarchSpeed([]ezfyUnitGroup{{TroopId: 13, Count: 10}}, tech, navyOfficer); got != 1600 {
		t.Fatalf("越岛战术带海军应加速到 1600，实际 %d", got)
	}
	// 城防部队不参与（无部队 → 0，调用方回落 300）
	if got := h.ezfyMarchSpeed(nil, tech, nil); got != 0 {
		t.Fatalf("无部队应返回 0，实际 %d", got)
	}
}
