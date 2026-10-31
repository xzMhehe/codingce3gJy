package ezfy

import (
	"os"
	"strings"
	"testing"

	"qqjiayuan/server/internal/model"
)

// ============ 军官技能随等级自动升级（2026-10-06） ============
//
// 规则(v2)：军官每满 30 级技能等级 +1（30级=1级…150级=5级、180级=6级）；
// 加成类统乘倍率；绝地逆袭 N 级 = 前 N 回合。普通军官最高 5 级、名将最高 6 级。
// 等级动态推导、不落库，学得晚自然按当前等级定级。

// TestEzfySkillLevelOf 技能等级边界（2026-10-06 v2：每 30 级 +1 级，普通军官 cap5 / 名将 cap6）。
func TestEzfySkillLevelOf(t *testing.T) {
	cases := []struct {
		level, cap, want int
	}{
		{0, 5, 1}, {1, 5, 1}, {29, 5, 1}, {30, 5, 1}, {59, 5, 1},
		{60, 5, 2}, {89, 5, 2}, {90, 5, 3}, {119, 5, 3},
		{120, 5, 4}, {149, 5, 4}, {150, 5, 5}, {179, 5, 5},
		// 普通军官最高 5 级：180 级仍封顶 5
		{180, 5, 5}, {999, 5, 5},
		// 名将最高 6 级：180 级正好 6 级（「前提达到 180 级」）
		{150, 6, 5}, {179, 6, 5}, {180, 6, 6}, {209, 6, 6}, {350, 6, 6},
	}
	for _, c := range cases {
		if got := ezfySkillLevelOf(c.level, c.cap); got != c.want {
			t.Fatalf("ezfySkillLevelOf(%d, %d) = %d, 期望 %d", c.level, c.cap, got, c.want)
		}
	}
	// nil 军官兜底 1 级
	h := &EzfyHandler{}
	if got := h.officerSkillLevel(nil); got != 1 {
		t.Fatalf("officerSkillLevel(nil) = %d, 期望 1", got)
	}
	if got := generalSkillLevel(nil); got != 1 {
		t.Fatalf("generalSkillLevel(nil) = %d, 期望 1", got)
	}
	// 守将（军官池）同口径：普通军官 kind=1 最高 5 级 / 名将 kind=2 最高 6 级
	g180 := &model.EzfyCfgGeneral{Level: 180, Kind: 2}
	if got := generalSkillLevel(g180); got != 6 {
		t.Fatalf("generalSkillLevel(名将180级) = %d, 期望 6", got)
	}
	g180p := &model.EzfyCfgGeneral{Level: 180, Kind: 1}
	if got := generalSkillLevel(g180p); got != 5 {
		t.Fatalf("generalSkillLevel(普通守将180级) = %d, 期望 5", got)
	}
	g50 := &model.EzfyCfgGeneral{Level: 50, Kind: 2}
	if got := generalSkillLevel(g50); got != 1 {
		t.Fatalf("generalSkillLevel(名将50级) = %d, 期望 1", got)
	}
}

// TestSkillBattleBonusScaled 攻击类技能加成必须乘倍率。
//
// ★ 2026-10-08 起：通用段只剩「尖兵猛攻 30*scale」；火炮压制(10)/四指阵列·狼群围猎(15)
// 已移到 officerTypeBonus 按兵种/按目标下发（倍率同样要乘，见 TestTypeSpecificBonusWired）。
func TestSkillBattleBonusScaled(t *testing.T) {
	src := rawFile(t, "ezfy_officer.go")
	for _, want := range []string{"30 * scale", "10 * scale", "15*scale", "h.officerSkillScale(o)"} {
		if !strings.Contains(src, want) {
			t.Fatalf("技能加成未按等级乘倍率（缺 %s）：\n%s", want, src)
		}
	}
}

// TestGuardBonusScaled 防御类技能加成必须乘倍率（弧形壁垒），属性部分 officerGuardAttrBonus 不加倍。
//
// ★ 2026-10-31 弹幕覆盖移出：它按配置文案是「城防攻击范围+10%」，属**射程**技能，
// 不再算进防御加成（改由 officerTypeBonus 挂到城防射程，见 TestTypeSpecificBonusWired）。
func TestGuardBonusScaled(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_officer.go", "func (h *EzfyHandler) officerGuardBonus(")
	if !strings.Contains(body, "30 * scale") {
		t.Fatalf("officerGuardBonus 的技能部分未乘倍率：\n%s", body)
	}
	if !strings.Contains(body, "officerGuardAttrBonus") {
		t.Fatalf("officerGuardBonus 未保留属性部分（officerGuardAttrBonus）：\n%s", body)
	}
	if strings.Contains(body, `case "弹幕覆盖"`) {
		t.Fatalf("弹幕覆盖是射程技能，不该再进防御加成：\n%s", body)
	}
}

// TestSpeedSkillBonusScaled 速度类技能加成必须乘倍率，且**按兵种**下发。
//
// ★ 2026-10-09 行军速度也改为按兵种生效（`ezfy_order.go` 的 `ezfyMarchSpeed`）：
// 原来「不分兵种一律 +10%/级」的 `officerSpeedSkillBonus` 已删除（越岛登陆带队纯陆军不再
// 凭空加速），速度技能统一由 `officerTypeBonus` 按兵种下发（陆/空/海，10% × 技能等级）。
//
// ★ 2026-10-31 坦克冲锋再收窄：**只**轻型坦克(5)/重型坦克(6) → 走 SpeedTroop（兵种 ID）。
//
// ★★ 2026-10-31 二次修订：坦克冲锋/闪电疾攻/越岛登陆 由「速度」改「**攻速**」——
// 走 AtkSpeed/AtkSpeedTroop（只影响出手先后），**不再**写 Speed/SpeedTroop（移动/行军速度）。
func TestSpeedSkillBonusScaled(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_officer.go", "func (h *EzfyHandler) officerTypeBonus(")
	for _, want := range []string{
		"b.AtkSpeedTroop[id] += 10 * scale",           // 坦克冲锋 → 只轻坦/重坦（ezfyTankRushTroopIds）
		"b.AtkSpeed[ezfyTroopTypeAir] += 10 * scale",  // 闪电疾攻 → 空军攻速
		"b.AtkSpeed[ezfyTroopTypeNavy] += 10 * scale", // 越岛登陆 → 海军攻速
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("攻速类技能未按兵种 × 倍率下发（缺 %s）：\n%s", want, body)
		}
	}
	// 这三项技能不能再回到「速度」档（否则又变成加移动/行军速度）
	for _, bad := range []string{
		"b.Speed[ezfyTroopTypeAir] += 10 * scale",
		"b.Speed[ezfyTroopTypeNavy] += 10 * scale",
		"b.Speed[ezfyTroopTypeArmy] += 10 * scale",
		"b.SpeedTroop[id] += 10 * scale",
	} {
		if strings.Contains(body, bad) {
			t.Fatalf("速度技能已改攻速，不该再写 %s：\n%s", bad, body)
		}
	}
	// 行军速度必须走按兵种的 ezfyMarchSpeed（机械增压通用 + 喷气推进/军官技能按兵种）
	march := ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) ezfyMarchSpeed(")
	for _, want := range []string{
		"ezfyTechEffectValueAt(10, techMap[10])", // 机械增压：通用（全体兵种），★ 2026-10-31 起读配置
		"tb.Speed[ezfyTroopTypeAir] += v",        // 喷气推进：只空军
		"h.officerTypeBonus(lead, tb)",           // 军官速度技能：按兵种
		"tb.speedOf(cfg.Type)",                   // 按兵种取专属加成
		"tb.speedTroopOf(cfg.ID)",                // 坦克冲锋：按兵种 ID（只轻坦/重坦）
	} {
		if !strings.Contains(march, want) {
			t.Fatalf("行军速度未按兵种接入科技/军官速度加成（缺 %s）：\n%s", want, march)
		}
	}
}

// TestLootHealScaled 机械改装（回收率 5%/级、出征油耗 -10%/级）必须乘技能倍率。
//
// ★★ 2026-10-31 黄金洞察（原黄金眼）已由「战利品/掠夺率」改为「侦察/情报等级」，
// 不再有 `officerSkillScale` 的掠夺倍率点 —— 这里只盯机械改装（ezfy_order.go ≥2 处、
// activity_target.go ≥1 处，另加油耗减免 10%/级）。
func TestLootHealScaled(t *testing.T) {
	order := rawFile(t, "ezfy_order.go")
	// ★ 2026-10-15 机械改装回收率由 10%/级 → **5%/级**（名将满级最高 30%）：攻方战后恢复 + 守方伤兵回收都锁 5 系数
	if got := strings.Count(order, "+= 5 * h.officerSkillScale("); got < 2 {
		t.Fatalf("ezfy_order.go 里机械改装回收率 5%%/级 点只有 %d 处，期望 ≥2", got)
	}
	// 出征油耗减免 10%/级（同一个技能的另一半效果）
	if !strings.Contains(order, "return 10 * h.officerSkillScale(o)") {
		t.Fatalf("ezfy_order.go 里机械改装出征油耗减免没按技能等级放大")
	}
	act := rawFile(t, "ezfy_activity_target.go")
	if got := strings.Count(act, "+= 5 * h.officerSkillScale("); got < 1 {
		t.Fatalf("ezfy_activity_target.go 里机械改装回收率 5%%/级 点只有 %d 处，期望 ≥1", got)
	}
}

// TestActWildDefScaled 活动野地守将加成乘倍率（弧形壁垒），属性部分不加倍。
//
// ★ 2026-10-31 口径修正：弹幕覆盖（射程）/速度技能都不再走这个函数 ——
// 它们由 generalTypeBonus 按兵种下发（坦克冲锋只轻坦/重坦），所以这里只剩「防御」。
func TestActWildDefScaled(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_activity_target.go", "func ezfyActWildDefBonus(")
	if !strings.Contains(body, "generalSkillScale(g)") {
		t.Fatalf("ezfyActWildDefBonus 未按守将等级取倍率：\n%s", body)
	}
	if !strings.Contains(body, "30 * scale") {
		t.Fatalf("ezfyActWildDefBonus 的弧形壁垒未乘倍率：\n%s", body)
	}
	if !strings.Contains(body, "ezfyAttrToBonus(g.Learning)") {
		t.Fatalf("ezfyActWildDefBonus 未保留守将属性部分（ezfyAttrToBonus）：\n%s", body)
	}
	if strings.Contains(body, `case "弹幕覆盖"`) {
		t.Fatalf("弹幕覆盖是射程技能，不该再进守军防御加成：\n%s", body)
	}
	// 守将的兵种专属技能（含速度/射程）必须走 generalTypeBonus 且乘倍率
	gt := ezfyFuncBody(t, "ezfy_activity_target.go", "func generalTypeBonus(")
	for _, want := range []string{
		"generalSkillScale(g)",
		"b.AtkSpeedTroop[id] += 10 * scale",        // 坦克冲锋 → 只轻坦/重坦（攻速）
		"b.Range[ezfyTroopTypeCity] += 10 * scale", // 弹幕覆盖 → 城防射程
	} {
		if !strings.Contains(gt, want) {
			t.Fatalf("generalTypeBonus 未按兵种下发守将技能（缺 %s）：\n%s", want, gt)
		}
	}
}

// TestCounterTriggerRounds 绝地逆袭按「前N回合」触发（st.Round <= counterRounds），不再是第1回合。
func TestCounterTriggerRounds(t *testing.T) {
	src := rawFile(t, "ezfy_battle.go")
	if !strings.Contains(src, "st.Round <= counterRounds") {
		t.Fatalf("绝地逆袭触发未按回合数判断（缺 st.Round <= counterRounds）")
	}
	if strings.Contains(src, "st.Round == 1") {
		t.Fatalf("仍有「第1回合」写法 st.Round == 1，未升级为前N回合")
	}
}

// TestCounterSnapshotRounds 快照需保存回合数字段，且老快照(bool)要能回退成前1回合。
func TestCounterSnapshotRounds(t *testing.T) {
	src := rawFile(t, "ezfy_battle.go")
	if !strings.Contains(src, `json:"atk_counter_rounds"`) {
		t.Fatalf("快照缺 atk_counter_rounds 字段")
	}
	if !strings.Contains(src, "snap.DefCounter") {
		t.Fatalf("快照重建缺老 bool 回退逻辑（snap.DefCounter）")
	}
	if !strings.Contains(src, "atkRounds = 1") {
		t.Fatalf("快照重建缺老快照回退成 1 回合（atkRounds = 1）")
	}
}

// TestSkillEffectTextAtShowsFinalValue 技能效果文案直接显示放大后的最终数值。
//
// 用户反馈（2026-10-06）：「尖兵猛攻(Lv.3 攻击力+30% (效果×3)) 别这么展示，
// 效果×3 太 low」—— 要求直接按等级算出最终数值展示（如 攻击力+90%）。
func TestSkillEffectTextAtShowsFinalValue(t *testing.T) {
	cases := []struct {
		skill string
		lv    int
		want  string
	}{
		{"尖兵猛攻", 1, "攻击力+30%"},
		{"尖兵猛攻", 3, "攻击力+90%"},
		{"弧形壁垒", 2, "防御力+60%"},
		{"四指阵列", 3, "空军对空攻击+45%"},
		{"弹幕覆盖", 3, "城防攻击范围+30%"},
		{"机械改装", 2, "回收率+10%, 出征油耗-20%"},
		{"黄金洞察", 3, "侦察/情报等级+3"},
		{"绝地逆袭", 2, "前2回合反击"},
	}
	for _, c := range cases {
		if got := ezfySkillEffectTextAt(c.skill, c.lv); got != c.want {
			t.Fatalf("ezfySkillEffectTextAt(%q, %d) = %q, 期望 %q", c.skill, c.lv, got, c.want)
		}
	}
	// 文案里不再出现「(效果×N)」后缀（只查函数体，避免注释干扰）
	body := ezfyFuncBody(t, "ezfy_officer.go", "func ezfySkillEffectTextAt(")
	if strings.Contains(body, "效果×") {
		t.Fatal("技能效果文案仍拼「(效果×N)」后缀，应直接展示最终数值")
	}
}

// TestBonusBreakdownDetail 战报加成拆解：军官技能 / 科技都**逐项展开**（多个分开展示）；
// 明细为 nil（老战场快照）回退旧格式合并展示。
func TestBonusBreakdownDetail(t *testing.T) {
	cases := []struct {
		name      string
		officer   int
		skill     int
		techBase  int
		equip     int
		officerNm string
		skills    []ezfyBonusItem
		techs     []ezfyBonusItem
		want      string
	}{
		{
			// 逐项展开：技能/科技各拆成具体名字
			"逐项", 252, 150, 352, 0, "冥王",
			[]ezfyBonusItem{{Name: "尖兵猛攻", Value: 90}, {Name: "火炮压制", Value: 60}},
			[]ezfyBonusItem{{Name: "枪械改良", Value: 40}, {Name: "增程弹药", Value: 54}},
			"(军官·冥王+102% 军官技能·尖兵猛攻+90% 军官技能·火炮压制+60% 科技·枪械改良+40% 科技·增程弹药+54%)",
		},
		{
			// 科技有值但全 0（明细里 value=0 不展示）
			"技能明细零项", 252, 150, 352, 0, "冥王",
			[]ezfyBonusItem{{Name: "尖兵猛攻", Value: 150}},
			[]ezfyBonusItem{{Name: "增程弹药", Value: 0}},
			"(军官·冥王+102% 军官技能·尖兵猛攻+150%)",
		},
		{
			// 老快照回退：nil 明细 → 合并展示「军官技能+N% / 科技+N%」
			"老快照回退", 252, 150, 352, 0, "冥王", nil, nil,
			"(军官·冥王+102% 军官技能+150% 科技+100%)",
		},
		{
			// ★ 2026-10-09 「装备」→「装备套装」（散件+套装合计，去掉歧义）
			"装备套装", 252, 150, 352, 120, "冥王",
			nil, nil,
			"(军官·冥王+102% 军官技能+150% 科技+100% 装备套装+120%)",
		},
		{"全零", 0, 0, 0, 0, "", nil, nil, ""},
	}
	for _, c := range cases {
		if got := ezfyBonusBreakdown(c.officer, c.skill, c.techBase, c.equip, c.officerNm, c.skills, c.techs); got != c.want {
			t.Fatalf("%s: ezfyBonusBreakdown = %q, 期望 %q", c.name, got, c.want)
		}
	}
}

// TestDefBonusTxt 被攻击方的「防御加成」被打行展示：逐项明细（城墙/科技/军官属性/军官技能/装备，
// Name 带前缀直接拼「守方防御加成+N%(...)」/「攻方防御加成+N%(...)」——带归属方，避免误读为给对方加成）；
// 全零/空明细 → 不展示防御段（空串）。
func TestDefBonusTxt(t *testing.T) {
	cases := []struct {
		name  string
		label string
		items []ezfyBonusItem
		want  string
	}{
		{
			"玩家城完整-守方",
			"守方防御加成",
			[]ezfyBonusItem{
				{Name: "城墙", Value: 50},
				{Name: "科技·均质装甲", Value: 30},
				{Name: "军官·冥王", Value: 20},
				{Name: "军官技能·弧形壁垒", Value: 60},
				{Name: "装备套装", Value: 20},
			},
			"守方防御加成+180%(城墙+50% 科技·均质装甲+30% 军官·冥王+20% 军官技能·弧形壁垒+60% 装备套装+20%)",
		},
		{
			"攻方军官防御",
			"攻方防御加成",
			[]ezfyBonusItem{
				{Name: "军官·Stalin（斯大林）", Value: 105},
				{Name: "军官技能·弧形壁垒", Value: 60},
			},
			"攻方防御加成+165%(军官·Stalin（斯大林）+105% 军官技能·弧形壁垒+60%)",
		},
		{
			"明细0项跳过",
			"守方防御加成",
			[]ezfyBonusItem{{Name: "城墙", Value: 50}, {Name: "科技·重型机床", Value: 0}},
			"守方防御加成+50%(城墙+50%)",
		},
		{"空明细", "守方防御加成", nil, ""},
		{"全零", "守方防御加成", []ezfyBonusItem{{Name: "城墙", Value: 0}}, ""},
	}
	for _, c := range cases {
		if got := ezfyDefBonusTxt(c.label, c.items); got != c.want {
			t.Fatalf("%s: ezfyDefBonusTxt = %q, 期望 %q", c.name, got, c.want)
		}
	}
}

// TestDefBonusInjected 被打行注入「防御加成」段（双向）：
//
//	· 攻方打守方 → 展示守方防御加成(defBonusTxt)
//	· 守方打攻方（含守方绝地逆袭还击）→ 展示攻方防御加成(atkDefBonusTxt)
//
// 2026-10-07 起攻方军官防御技能(弧形壁垒/弹幕覆盖)+装备 Def 生效：unitDefBonus 不再恒 0。
//
// ★ 2026-10-09 修「装备防御被算两遍」：`st.AtkDefBonus` / `defBonus` 里**已经含**装备 Def，
// 原来又各加了一次 → 战报显示 425% 实际按 486% 减伤。现在直接取这两个值（口径与展示一致）。
func TestDefBonusInjected(t *testing.T) {
	// ★ 2026-10-08 规范化行尾：仓库在 Windows 下是 CRLF（core.autocrlf=true），
	//   而本测试用 `\n` 拼多行片段 → 原来在 CRLF 工作区**永远匹配不上**（假红）。
	src := strings.ReplaceAll(rawFile(t, "ezfy_battle.go"), "\r\n", "\n")
	for _, want := range []string{
		// ★ 2026-10-31 明细按回合过滤：弧形壁垒/火炮压制未发动回合不展示（否则「显示加了却没生效」）
		`defBonusTxt := ezfyDefBonusTxt("守方防御加成", ezfyTimedBreakFilter(st.DefDefBreak, st.DefTimed, st.Round))`,
		`atkDefBonusTxt := ezfyDefBonusTxt("攻方防御加成", ezfyTimedBreakFilter(st.AtkDefBreak, st.AtkTimed, st.Round))`,
		`return fmt.Sprintf("%s+%d%%(%s)", label, total, strings.Join(parts, " "))`,
		// 守方打攻方 → 攻方防御减伤（已含装备 Def）；★ 2026-10-31 弧形壁垒只发动回合生效
		"unitDefBonus = atkDefBonusAll + st.AtkType.defOf(target.cfg.Type)\n",
		// 攻方打守方 → 守方防御减伤（已含装备 Def）；★ 2026-10-24 城墙只对城防兵种生效
		"unitDefBonus = defDefBonusAll + st.DefType.defOf(target.cfg.Type) + st.wallBonusFor(target)\n",
		// 两处「按回合扣掉弧形壁垒」的入口
		"atkDefBonusAll := st.AtkDefBonus - atkTimedDefOff",
		"defDefBonusAll := defBonus - defTimedDefOff",
		`json:"atk_def_bonus"`,
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("ezfy_battle.go 缺防御加成逻辑：%s", want)
		}
	}
	// 反击行双向：「攻方在打 → 展示攻方防御」「守方在打 → 展示守方防御」
	if !strings.Contains(src, "if isAtk {\n\t\t\t\t\tif atkDefBonusTxt != \"\"") ||
		!strings.Contains(src, "} else if defBonusTxt != \"\" {\n\t\t\t\t\tline += \", \" + defBonusTxt") {
		t.Fatal("ezfy_battle.go 反击行防御段注入不完整")
	}
}

// TestDefenderAtkTechMatchesAttacker 守方**攻击**加成的科技口径必须与攻方一致。
//
// ★★ 2026-10-08 用户反馈「两个号都是满科技，为什么科技加成不一样」：
//
//	守方原来用「均质装甲(7)+重型机床(9)+永备工事(16)」算攻击加成 ——
//	均质装甲的配置描述是「部队防御」，只该进防御加成；而「枪械改良(6)」
//	（描述是「部队攻击」）完全没算。结果满科技的两个号：【守方科技】行比【攻方科技】
//	少一项，攻击加成还偏低。
//
// ★★ 2026-10-09 用户再次要求「两个号都满科技，加成就该一样」：守方**不再额外吃永备工事(16)**。
//
//	那是守城方专属的一项，会让满科技的两个号在战报里显示成「攻方 攻击+70% / 守方 攻击+90%」。
//
// ★★ 2026-10-21 动员训练(5) 改为「造兵速度」科技：攻守双方的攻击计算/战报都不再包含它。
//
//	现在两边攻击口径完全一致：枪械改良(6) + 重型机床(9) + 军官加成。
func TestDefenderAtkTechMatchesAttacker(t *testing.T) {
	src := strings.ReplaceAll(rawFile(t, "ezfy_order.go"), "\r\n", "\n")
	if !strings.Contains(src, "defAtkBonus = defTech[6]*3 + defTech[9]*2 + h.officerBattleBonus(cityGuard)") {
		t.Fatal("守方攻击加成必须与攻方同口径（枪械改良6 + 重型机床9 + 军官）")
	}
	if strings.Contains(src, "defTech[5]") {
		t.Fatal("动员训练(5) 已改为造兵速度科技，不该再进守方攻击加成")
	}
	if strings.Contains(src, "defAtkBonus = defTech[6]*3 + defTech[9]*2 + defTech[16]*2") {
		t.Fatal("守方攻击加成不该再额外吃永备工事(16)：守城方专属项会让满科技的攻守两号加成不一致")
	}
	if strings.Contains(src, "defAtkBonus = defTech[7]*3") {
		t.Fatal("守方攻击加成不该用均质装甲(7)（描述是「部队防御」，只进防御加成）")
	}
	if !strings.Contains(src, `ezfyTechItem("枪械改良", defTech[6]*3)`) {
		t.Fatal("【守方科技】行缺枪械改良（应与【攻方科技】同集合）")
	}
	if strings.Contains(src, `ezfyTechItem("动员训练"`) {
		t.Fatal("【守方科技】行不该再列动员训练（已改为造兵速度科技）")
	}
}

// TestFourTechCategoriesWired 四类科技加成必须在**攻守双方**都真实参与计算。
//
// ★★ 2026-10-08 ：「攻击加成的科技、防御加成科技、速度加成科技、射程加成科技
// 这里都要展示，而且回合中也要实际有」。
//
//	展示侧由【攻方科技】/【守方科技】两行负责（同一集合，见 TestDefenderAtkTechMatchesAttacker）；
//	这里锁**计算侧** —— 每类科技都要在 ezfy_order.go 的加成表达式里出现，
//	否则就会出现「列表里列了、回合里不生效」的假展示。
//
// 科技 ID 对照：5 动员训练(造兵速度，不参与战斗) · 6 枪械改良(攻) · 7 均质装甲(防) ·
// 8 增程弹药(射程) · 9 重型机床(重装备攻防) · 10 机械增压(速度) · 16 永备工事(城防攻防) · 19 喷气推进(空军速度)
func TestFourTechCategoriesWired(t *testing.T) {
	src := strings.ReplaceAll(rawFile(t, "ezfy_order.go"), "\r\n", "\n")
	// ★ 速度只列**通用**那部分（机械增压 10「部队速度」）；喷气推进(19)是「空军速度」，
	//   属兵种专属，见下面的 TestTypeSpecificBonusWired。
	atkCases := []struct{ name, want string }{
		{"攻击", "atkBonus := officerBonus + atkTech[6]*3 + atkTech[9]*2"},
		// ★★ 2026-10-09 永备工事(16) 移出通用防御 → 只对城防单位生效（atkType.Def[4]）
		{"防御", "atkDefBonus += atkTech[7]*3 + atkTech[9]*2"},
		// ★ 2026-10-31 机械增压满级改 50% → 速度改读配置（ezfyTechEffectValueAt）
		{"速度(通用)", "atkSpeedBonus := ezfyTechEffectValueAt(10, atkTech[10])"},
		{"射程", "atkRangeBonus := atkTech[8] * 3"},
	}
	defCases := []struct{ name, want string }{
		// ★ 2026-10-09 「攻守加成一致」：守方攻击不再额外吃永备工事(16)
		{"攻击", "defAtkBonus = defTech[6]*3 + defTech[9]*2 + h.officerBattleBonus(cityGuard)"},
		// ★ 2026-10-09 同上：守方防御不再叠加城墙(建筑7)，永备工事(16) 也只对城防生效
		{"防御", "defBonus = defTech[7]*3 + defTech[9]*2"},
		{"速度(通用)", "defSpeedBonus = ezfyTechEffectValueAt(10, defTech[10])"},
		// ★ 射程只吃增程弹药(8)（2026-10-08 修掉虚高）
		{"射程", "defRangeBonus = defTech[8] * 3"},
	}
	for _, c := range atkCases {
		if !strings.Contains(src, c.want) {
			t.Fatalf("攻方缺「%s」类科技加成（列了却不生效）：%s", c.name, c.want)
		}
	}
	for _, c := range defCases {
		if !strings.Contains(src, c.want) {
			t.Fatalf("守方缺「%s」类科技加成（列了却不生效）：%s", c.name, c.want)
		}
	}
	// ★ 2026-10-21 动员训练(5) 已改为造兵速度科技，战斗计算/战报都不该再出现。
	for _, bad := range []string{"atkTech[5]*2", "defTech[5]*2"} {
		if strings.Contains(src, bad) {
			t.Fatalf("动员训练(5) 不该再参与战斗攻击加成：%s", bad)
		}
	}
}

// TestCoverDefenseOnlyForCityTroops 永备工事(16) 必须**只对城防单位(4) 生效**。
//
// ★★ 2026-10-09 用户口径：「永备工事 只对城防单位生效，也是属于兵种单位加成只不过是城防单位类型」。
//
//	所以它不能进全军通用的 defBonus / atkDefBonus，而要挂到 ezfyTypeBonus.Def[城防] 上，
//	由战斗引擎按**被攻击方的兵种 type** 取 —— 普通部队(陆/海/空)不吃，城防单位才吃。
func TestCoverDefenseOnlyForCityTroops(t *testing.T) {
	src := strings.ReplaceAll(rawFile(t, "ezfy_order.go"), "\r\n", "\n")
	// ① 通用防御里不许再出现永备工事（否则「列了却对全军生效」）
	for _, bad := range []string{
		"defBonus = defTech[7]*3 + defTech[16]*2",
		"atkDefBonus += atkTech[7]*3 + atkTech[16]*2",
	} {
		if strings.Contains(src, bad) {
			t.Fatalf("永备工事不该再进通用防御：%s", bad)
		}
	}
	// ② 必须挂到城防兵种的专属防御上（攻守双方各一处）
	for _, want := range []string{
		"atkType.Def[ezfyTroopTypeCity] = v",
		"defType.Def[ezfyTroopTypeCity] = v",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("永备工事必须挂到城防(4) 的兵种专属防御上：%s", want)
		}
	}
	// ③ 战斗引擎按**被攻击方**兵种 type 取这一截（普攻 + 反击共 4 处）
	bat := strings.ReplaceAll(rawFile(t, "ezfy_battle.go"), "\r\n", "\n")
	for _, want := range []string{
		"st.DefType.defOf(target.cfg.Type)",
		"st.AtkType.defOf(target.cfg.Type)",
		"st.AtkType.defOf(unit.cfg.Type)",
		"st.DefType.defOf(unit.cfg.Type)",
	} {
		if !strings.Contains(bat, want) {
			t.Fatalf("战斗引擎必须按被攻击方兵种取专属防御：%s", want)
		}
	}
}

// TestTypeDefBonusOf 兵种专属防御的取值与展示（永备工事只对城防生效）。
func TestTypeDefBonusOf(t *testing.T) {
	tb := ezfyTypeBonus{Def: map[int]int{ezfyTroopTypeCity: 20}}
	if got := tb.defOf(ezfyTroopTypeCity); got != 20 {
		t.Fatalf("城防单位应吃 20%% 专属防御，实际 %d", got)
	}
	if got := tb.defOf(ezfyTroopTypeArmy); got != 0 {
		t.Fatalf("普通陆军不该吃永备工事，实际 %d", got)
	}
	// 老战场快照没有 Def 字段 → 返回 0（不能崩）
	if got := (ezfyTypeBonus{}).defOf(ezfyTroopTypeCity); got != 0 {
		t.Fatalf("nil 表应返回 0，实际 %d", got)
	}
	// 展示：只有该方真的带了城防单位才显示
	if got, want := ezfyTypeDefTxt(tb, map[int]bool{ezfyTroopTypeCity: true}), " 兵种专属防御(城防+20%)"; got != want {
		t.Fatalf("展示文案 = %q，期望 %q", got, want)
	}
	if got := ezfyTypeDefTxt(tb, map[int]bool{ezfyTroopTypeArmy: true}); got != "" {
		t.Fatalf("没带城防单位时不该展示，实际 %q", got)
	}
}

// TestTypeSpecificBonusWired 兵种专属加成必须**按兵种**下发，不能混进通用加成。
//
// ★★ 2026-10-08 用户确认：「玩家科技、军官技能有的是全部兵种，有的是指定兵种，不能一起加」。
//
//	带兵种的科技：喷气推进(19)「空军速度+3%/级」；
//	带兵种的军官技能：火炮压制(陆军装甲攻击) / 四指阵列(空军对空攻击) / 狼群围猎(海军对海攻击) /
//	坦克冲锋(陆军速度) / 闪电疾攻(空军速度) / 越岛登陆(海军速度)。
//	它们必须走 ezfyTypeBonus（Atk/Speed 按 troop type 索引），且**不得**出现在通用加成里。
func TestTypeSpecificBonusWired(t *testing.T) {
	ord := strings.ReplaceAll(rawFile(t, "ezfy_order.go"), "\r\n", "\n")
	off := strings.ReplaceAll(rawFile(t, "ezfy_officer.go"), "\r\n", "\n")
	bat := strings.ReplaceAll(rawFile(t, "ezfy_battle.go"), "\r\n", "\n")

	// ① 喷气推进（空军速度）→ 只进空军
	for _, want := range []string{
		"atkType.Speed[ezfyTroopTypeAir] += v", // 攻方
		"defType.Speed[ezfyTroopTypeAir] += v", // 守方
	} {
		if !strings.Contains(ord, want) {
			t.Fatalf("喷气推进（空军速度）未按兵种下发：%s", want)
		}
	}
	// ② 通用速度不得再包含喷气推进（那是空军专属）
	for _, bad := range []string{
		"atkSpeedBonus := atkTech[10]*2 + atkTech[19]*3",
		"defSpeedBonus = defTech[10]*2 + defTech[19]*3",
	} {
		if strings.Contains(ord, bad) {
			t.Fatalf("通用速度不该含喷气推进(19)（那是空军专属）：%s", bad)
		}
	}
	// ③ 军官的六个兵种技能必须按兵种下发（2026-10-31 用户确认的口径）：
	//   火炮压制 → **陆军装甲部队**（装甲车4/轻坦5/重坦6/突击炮7/火箭8）攻击属性 +10/级；
	//   四指阵列 → 空军打空军；狼群围猎 → 海军**攻击力%**（不分目标）；
	//   坦克冲锋 → **只**轻坦5/重坦6 **攻速**；闪电疾攻/越岛登陆 → 空/海**攻速**；
	//   弹幕覆盖 → **只**城防单位的射程。
	for _, want := range []string{
		"b.AtkFlatTroop[id] += 10 * scale",                          // 火炮压制 → 装甲部队（按兵种 ID）
		"b.addVsType(ezfyTroopTypeAir, ezfyTroopTypeAir, 15*scale)", // 四指阵列 → 空军打空军
		"b.Atk[ezfyTroopTypeNavy] += 15 * scale",                    // 狼群围猎 → 海军攻击力%
		"b.AtkSpeedTroop[id] += 10 * scale",                         // 坦克冲锋 → 只轻坦/重坦攻速
		"b.AtkSpeed[ezfyTroopTypeAir] += 10 * scale",                // 闪电疾攻 → 空军攻速
		"b.AtkSpeed[ezfyTroopTypeNavy] += 10 * scale",               // 越岛登陆 → 海军攻速
		"b.Range[ezfyTroopTypeCity] += 10 * scale",                  // 弹幕覆盖 → 只城防射程
	} {
		if !strings.Contains(off, want) {
			t.Fatalf("军官兵种技能未按兵种下发：%s", want)
		}
	}
	// 旧口径必须消失：火炮压制不再整档给陆军、狼群围猎不再只对海、坦克冲锋不再整档给陆军，
	// 且三个速度技能已改攻速 → 不得再写 Speed/SpeedTroop。
	for _, bad := range []string{
		"b.AtkFlat[ezfyTroopTypeArmy] += 10 * scale",
		"b.addVsType(ezfyTroopTypeNavy, ezfyTroopTypeNavy, 15*scale)",
		"b.Speed[ezfyTroopTypeArmy] += 10 * scale",
		"b.Speed[ezfyTroopTypeAir] += 10 * scale",
		"b.Speed[ezfyTroopTypeNavy] += 10 * scale",
		"b.SpeedTroop[id] += 10 * scale",
	} {
		if strings.Contains(off, bad) {
			t.Fatalf("军官兵种技能仍是旧口径：%s", bad)
		}
	}
	// ④ 通用技能加成里不得再出现兵种技能（尖兵猛攻是全体兵种，保留）
	if !strings.Contains(off, `if s == "尖兵猛攻" {`) {
		t.Fatal("通用技能加成应只保留「尖兵猛攻」（全体兵种）")
	}
	if i := strings.Index(off, "func (h *EzfyHandler) officerSkillBattleBonus"); i >= 0 {
		seg := off[i:]
		if j := strings.Index(seg, "// officerSkillsBreak"); j > 0 {
			seg = seg[:j]
		}
		for _, bad := range []string{"火炮压制", "四指阵列", "狼群围猎"} {
			if strings.Contains(seg, bad) {
				t.Fatalf("通用技能加成里不该再出现兵种技能：%s", bad)
			}
		}
	}
	// ⑤ 战斗引擎必须按兵种取值叠加（攻击 + 速度 + 射程，攻守双方）
	for _, want := range []string{
		"tb := st.AtkType.attackBonusFor(unit.cfg.Type, target.cfg)", // 攻方行动（含"对目标"加成）
		"tb := st.DefType.attackBonusFor(unit.cfg.Type, target.cfg)", // 守方行动
		"unitAtkBonus = atkBonus + tb",
		"unitAtkBonus = st.DefAtkBonus + tb",
		"atkSpeedBonus + st.AtkType.speedOf(unit.cfg.Type)",
		"defSpeedBonus + st.DefType.speedOf(unit.cfg.Type)",
		"st.AtkType.speedTroopOf(unit.cfg.ID)",   // 坦克冲锋按兵种 ID
		"st.DefType.speedTroopOf(unit.cfg.ID)",   // 同上（守方）
		"st.AtkType.rangeOf(unit.cfg.Type)",      // 弹幕覆盖只给城防射程
		"st.DefType.rangeOf(unit.cfg.Type)",      // 同上（守方）
		"st.AtkType.atkFlatTroopOf(unit.cfg.ID)", // 火炮压制按兵种 ID
		"st.DefType.atkFlatTroopOf(unit.cfg.ID)", // 同上（守方）
	} {
		if !strings.Contains(bat, want) {
			t.Fatalf("战斗引擎未按兵种叠加加成：%s", want)
		}
	}
}

// TestTypeBonusInAttackAndCounter 兵种专属加成必须在**普通攻击与反击**里都生效，且日志要标出「兵种专属+N%」。
//
// ★★ 2026-10-08 用户反馈「狼群围猎(Lv.6 海军对海攻击+90%) 没有对海军/对海攻击加成」：
//
//	① 反击的 cbAtk 原来只取通用加成 → 按兵种的技能在反击时**完全不生效**（数值就错）；
//	② 普通攻击里数值算对了，但拆解（bonusBreak 只拆通用段）看不到 → 玩家相加对不上总数，以为没生效。
func TestTypeBonusInAttackAndCounter(t *testing.T) {
	bat := strings.ReplaceAll(rawFile(t, "ezfy_battle.go"), "\r\n", "\n")
	for _, want := range []string{
		// 普通攻击：数值含专属 + 行尾标注
		"unitAtkBonus = atkBonus + tb",
		"unitAtkBonus = st.DefAtkBonus + tb",
		`typeAtkTxt = fmt.Sprintf(" 兵种专属+%d%%", tb)`,
		// 反击：数值含反击方专属（含"对目标"加成）+ 行尾标注
		"cbType = st.DefType.attackBonusFor(target.cfg.Type, unit.cfg)",
		"cbType = st.AtkType.attackBonusFor(target.cfg.Type, unit.cfg)",
		"cbAtk = st.DefAtkBonus + cbType",
		"cbAtk = atkBonus + cbType",
		// 兵种攻击属性加成（绝对值）在攻击/反击里都要加上
		"ezfyCalcDamage(baseAtk+atkFlat,",
		"ezfyCalcDamage(ezfyPickAttack(target.cfg, unit.cfg)+cbFlat,",
		`cbTypeTxt = fmt.Sprintf(" 兵种专属+%d%%", cbType)`,
	} {
		if !strings.Contains(bat, want) {
			t.Fatalf("兵种专属加成未接入（攻击/反击/日志标注）：%s", want)
		}
	}
}

// TestBattleRoomHeadKeepsBonusLine 指挥室下发的准备回合（head）不能把「战斗加成」汇总行截掉。
//
// ★★ 2026-10-08 用户反馈「指挥模块里汇总的没加」：`ezfy_battle_room.go` 原来 `head[:6]`，
// 而准备回合顺序是 军官×2 → 科技×2 → 装备×2 → 套装×0~2 → **战斗加成** → 场景描述，
// 6 行正好卡在【守方装备】→ 末尾的汇总行被切掉（战报不受影响，战报用 st.Head 全量）。
// 上限必须 ≥ 12。
func TestBattleRoomHeadKeepsBonusLine(t *testing.T) {
	src := strings.ReplaceAll(rawFile(t, "ezfy_battle_room.go"), "\r\n", "\n")
	if strings.Contains(src, "head = head[:6]") || strings.Contains(src, "if len(head) > 6 {") {
		t.Fatal("指挥室 head 截断上限仍是 6 —— 会把末尾的「战斗加成」汇总行切掉")
	}
	if !strings.Contains(src, "if len(head) > 12 {") {
		t.Fatal("指挥室 head 截断上限应为 12（要覆盖 军官/科技/装备/套装/战斗加成/场景 全部行）")
	}
}

// TestAtkDefBonusWired 出征军官的防御加成已接入战斗引擎（order.go 与 activity_target.go 都构造并传参）。
func TestAtkDefBonusWired(t *testing.T) {
	for _, f := range []string{"ezfy_order.go", "ezfy_activity_target.go"} {
		src := rawFile(t, f)
		if !strings.Contains(src, "atkDefBonus += attr") || !strings.Contains(src, "atkDefBonus += atkEquip.Def") {
			t.Fatalf("%s 缺攻方防御加成构造（属性+装备）：%s", f, src)
		}
		if !strings.Contains(src, "atkDefBonus, atkDefBreak,") {
			t.Fatalf("%s 缺攻方防御传参（atkDefBonus, atkDefBreak）", f)
		}
	}
}

// TestCarryResAllOrderTypes 随军资源**所有出征类型**都能携带（不止运输/派遣）：
//   - createOrder 扣出发城资源不看 orderType（if hasRes 就扣）；
//   - 召回/返航随身资源原样带回（并入 Carry，采集产出不丢）；
//   - 前端随军资源块对所有类型显示并带提示。
//
// 背景：玩家资源多了可随身带出腾仓库/防被抢；只有 5/8 能带会造成「订单成功但资源没扣」的体感 bug。
func TestCarryResAllOrderTypes(t *testing.T) {
	order := rawFile(t, "ezfy_order.go")
	if strings.Contains(order, "仅运输/派遣可携带随军资源") {
		t.Fatalf("ezfy_order.go 仍残留「非运输/派遣拒绝」逻辑, 应支持所有类型带资源")
	}
	if strings.Contains(order, "if hasRes && orderType != 5 && orderType != 8") {
		t.Fatalf("ezfy_order.go 仍残留 5/8 限定扣减条件")
	}
	if !strings.Contains(order, "if hasRes {") {
		t.Fatalf("ezfy_order.go 缺「所有类型 hasRes 都扣出发城资源」")
	}
	if !strings.Contains(order, "c := parseCarry(order.Carry)") || !strings.Contains(order, "r := parseCarry(order.Resources)") {
		t.Fatalf("ezfy_order.go 召回未把随身资源并入 Carry（采集产出会丢）")
	}
	// 前端：随军资源对所有类型显示（不含 5/8 限定）+ 提示文案
	web, err := os.ReadFile("../../../../web/src/views/ezfy/modules/EzfyMap.vue")
	if err != nil {
		t.Fatalf("读前端 Ezfymap.vue 失败: %v", err)
	}
	ws := string(web)
	if strings.Contains(ws, `v-if="ezfy.orderType === 5 || ezfy.orderType === 8"`) {
		t.Fatalf("Ezfymap.vue 仍把随军资源限定在 运输/派遣")
	}
	if !strings.Contains(ws, "采集产出入库时随身资源不重复入库") {
		t.Fatalf("Ezfymap.vue 随军资源缺简洁提示")
	}
}

// TestWildDefSkillBonus 野地/寇城守将防御类技能（弧形壁垒/弹幕覆盖）计入守军防御加成并逐项拆解。
func TestWildDefSkillBonus(t *testing.T) {
	order := rawFile(t, "ezfy_order.go")
	if !strings.Contains(order, "defBonus += generalSkillDefBonus(defGeneral)") {
		t.Fatal("ezfy_order.go 野地守将未把防御技能计入 defBonus（generalSkillDefBonus）")
	}
	if !strings.Contains(order, "defDefBreak = append(defDefBreak, ezfyBonusItem{Name: \"军官技能·\" + s.Name, Value: s.Value})") ||
		!strings.Contains(order, "for _, s := range generalSkillDefBreak(defGeneral)") {
		t.Fatal("ezfy_order.go 野地守将未把防御技能逐项拆进 defDefBreak")
	}
}

// ============ 辅助 ============

// rawFile 读取指定源码文件的全部内容（静态断言用）。
func rawFile(t *testing.T, file string) string {
	t.Helper()
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("读源码失败 %s: %v", file, err)
	}
	// 源码断言统一按 LF 比较：仓库在 Windows 检出后是 CRLF，
	// 带 \n 的多行断言会因此假失败（与代码对错无关）。
	return strings.ReplaceAll(string(b), "\r\n", "\n")
}
