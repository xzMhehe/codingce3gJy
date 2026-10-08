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
// 加成类统乘倍率；绝地反击 N 级 = 前 N 回合。普通军官最高 5 级、名将最高 6 级。
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

// TestSkillBattleBonusScaled 攻击类技能加成必须乘倍率（尖兵突击/火炮控制/四指编队·狼群战术）。
func TestSkillBattleBonusScaled(t *testing.T) {
	src := rawFile(t, "ezfy_officer.go")
	for _, want := range []string{"30 * scale", "15 * scale", "h.officerSkillScale(o)"} {
		if !strings.Contains(src, want) {
			t.Fatalf("officerSkillBattleBonus 未按等级乘倍率（缺 %s）：\n%s", want, src)
		}
	}
}

// TestGuardBonusScaled 防御类技能加成必须乘倍率（弧形防御/弹幕支援），属性部分 officerGuardAttrBonus 不加倍。
func TestGuardBonusScaled(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_officer.go", "func (h *EzfyHandler) officerGuardBonus(")
	if !strings.Contains(body, "30 * scale") || !strings.Contains(body, "10 * scale") {
		t.Fatalf("officerGuardBonus 的技能部分未乘倍率：\n%s", body)
	}
	if !strings.Contains(body, "officerGuardAttrBonus") {
		t.Fatalf("officerGuardBonus 未保留属性部分（officerGuardAttrBonus）：\n%s", body)
	}
}

// TestSpeedSkillBonusScaled 速度类技能加成必须乘倍率（坦克突袭/闪电袭击/越岛战术）。
func TestSpeedSkillBonusScaled(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_officer.go", "func (h *EzfyHandler) officerSpeedSkillBonus(")
	if !strings.Contains(body, "10 * h.officerSkillScale(o)") {
		t.Fatalf("officerSpeedSkillBonus 未按等级乘倍率：\n%s", body)
	}
}

// TestLootHealScaled 黄金眼掠夺 / 机械改造恢复 全部乘倍率（ezfy_order.go ≥3 处、activity_target.go ≥1 处）。
func TestLootHealScaled(t *testing.T) {
	order := rawFile(t, "ezfy_order.go")
	if got := strings.Count(order, "10 * h.officerSkillScale(leadOfficer)"); got < 3 {
		t.Fatalf("ezfy_order.go 里黄金眼/机械改造乘倍率点只有 %d 处，期望 ≥3", got)
	}
	act := rawFile(t, "ezfy_activity_target.go")
	if got := strings.Count(act, "10 * h.officerSkillScale(leadOfficer)"); got < 1 {
		t.Fatalf("ezfy_activity_target.go 里机械改造乘倍率点只有 %d 处，期望 ≥1", got)
	}
}

// TestActWildDefScaled 活动野地守将加成乘倍率（弧形防御/弹幕支援/速度），属性部分不加倍。
func TestActWildDefScaled(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_activity_target.go", "func ezfyActWildDefBonus(")
	if !strings.Contains(body, "generalSkillScale(g)") {
		t.Fatalf("ezfyActWildDefBonus 未按守将等级取倍率：\n%s", body)
	}
	if !strings.Contains(body, "10 * scale") {
		t.Fatalf("ezfyActWildDefBonus 的速度/弹幕支援未乘倍率：\n%s", body)
	}
	if !strings.Contains(body, "ezfyAttrToBonus(g.Learning)") {
		t.Fatalf("ezfyActWildDefBonus 未保留守将属性部分（ezfyAttrToBonus）：\n%s", body)
	}
}

// TestCounterTriggerRounds 绝地反击按「前N回合」触发（st.Round <= counterRounds），不再是第1回合。
func TestCounterTriggerRounds(t *testing.T) {
	src := rawFile(t, "ezfy_battle.go")
	if !strings.Contains(src, "st.Round <= counterRounds") {
		t.Fatalf("绝地反击触发未按回合数判断（缺 st.Round <= counterRounds）")
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
// 用户反馈（2026-10-06）：「尖兵突击(Lv.3 攻击力+30% (效果×3)) 别这么展示，
// 效果×3 太 low」—— 要求直接按等级算出最终数值展示（如 攻击力+90%）。
func TestSkillEffectTextAtShowsFinalValue(t *testing.T) {
	cases := []struct {
		skill string
		lv    int
		want  string
	}{
		{"尖兵突击", 1, "攻击力+30%"},
		{"尖兵突击", 3, "攻击力+90%"},
		{"弧形防御", 2, "防御力+60%"},
		{"四指编队", 3, "空军对空攻击+45%"},
		{"弹幕支援", 3, "城防攻击范围+30%"},
		{"机械改造", 2, "回收率+20%, 出征油耗-20%"},
		{"黄金眼", 3, "侦查等级+3"},
		{"绝地反击", 2, "前2回合反击"},
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
			[]ezfyBonusItem{{Name: "尖兵突击", Value: 90}, {Name: "火炮控制", Value: 60}},
			[]ezfyBonusItem{{Name: "军训艺术", Value: 40}, {Name: "弹道学", Value: 54}},
			"(军官·冥王+102% 军官技能·尖兵突击+90% 军官技能·火炮控制+60% 科技·军训艺术+40% 科技·弹道学+54%)",
		},
		{
			// 科技有值但全 0（明细里 value=0 不展示）
			"技能明细零项", 252, 150, 352, 0, "冥王",
			[]ezfyBonusItem{{Name: "尖兵突击", Value: 150}},
			[]ezfyBonusItem{{Name: "弹道学", Value: 0}},
			"(军官·冥王+102% 军官技能·尖兵突击+150%)",
		},
		{
			// 老快照回退：nil 明细 → 合并展示「军官技能+N% / 科技+N%」
			"老快照回退", 252, 150, 352, 0, "冥王", nil, nil,
			"(军官·冥王+102% 军官技能+150% 科技+100%)",
		},
		{
			"装备", 252, 150, 352, 120, "冥王",
			nil, nil,
			"(军官·冥王+102% 军官技能+150% 科技+100% 装备+120%)",
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
				{Name: "科技·装甲科技", Value: 30},
				{Name: "军官·冥王", Value: 20},
				{Name: "军官技能·弧形防御", Value: 60},
				{Name: "装备", Value: 20},
			},
			"守方防御加成+180%(城墙+50% 科技·装甲科技+30% 军官·冥王+20% 军官技能·弧形防御+60% 装备+20%)",
		},
		{
			"攻方军官防御",
			"攻方防御加成",
			[]ezfyBonusItem{
				{Name: "军官·Stalin（斯大林）", Value: 105},
				{Name: "军官技能·弧形防御", Value: 60},
			},
			"攻方防御加成+165%(军官·Stalin（斯大林）+105% 军官技能·弧形防御+60%)",
		},
		{
			"明细0项跳过",
			"守方防御加成",
			[]ezfyBonusItem{{Name: "城墙", Value: 50}, {Name: "科技·重工技术", Value: 0}},
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
//	· 守方打攻方（含守方绝地反击还击）→ 展示攻方防御加成(atkDefBonusTxt)
//
// 2026-10-07 起攻方军官防御技能(弧形防御/弹幕支援)+装备 Def 生效：unitDefBonus 不再恒 0。
func TestDefBonusInjected(t *testing.T) {
	// ★ 2026-10-08 规范化行尾：仓库在 Windows 下是 CRLF（core.autocrlf=true），
	//   而本测试用 `\n` 拼多行片段 → 原来在 CRLF 工作区**永远匹配不上**（假红）。
	src := strings.ReplaceAll(rawFile(t, "ezfy_battle.go"), "\r\n", "\n")
	for _, want := range []string{
		`defBonusTxt := ezfyDefBonusTxt("守方防御加成", st.DefDefBreak)`,
		`atkDefBonusTxt := ezfyDefBonusTxt("攻方防御加成", st.AtkDefBreak)`,
		`return fmt.Sprintf("%s+%d%%(%s)", label, total, strings.Join(parts, " "))`,
		"unitDefBonus = st.AtkDefBonus + st.AtkEquip.Def", // 守方打攻方 → 攻方防御减伤
		"unitDefBonus = defBonus + st.DefEquip.Def",       // 攻方打守方 → 守方防御(含装备)减伤
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
//	守方原来用「装甲科技(7)+重工技术(9)+掩体防御(16)」算攻击加成 ——
//	装甲科技的配置描述是「部队防御」，只该进防御加成；而「军训艺术(5)/武器科技(6)」
//	（描述都是「部队攻击」）完全没算。结果满科技的两个号：【守方科技】行比【攻方科技】
//	少两项，攻击加成还偏低。
func TestDefenderAtkTechMatchesAttacker(t *testing.T) {
	src := strings.ReplaceAll(rawFile(t, "ezfy_order.go"), "\r\n", "\n")
	if !strings.Contains(src, "defAtkBonus = defTech[5]*2 + defTech[6]*3 + defTech[9]*2 + defTech[16]*2") {
		t.Fatal("守方攻击加成必须含军训艺术(5)/武器科技(6)，与攻方同口径")
	}
	if strings.Contains(src, "defAtkBonus = defTech[7]*3") {
		t.Fatal("守方攻击加成不该用装甲科技(7)（描述是「部队防御」，只进防御加成）")
	}
	if !strings.Contains(src, `ezfyTechItem("军训艺术", defTech[5]*2)`) ||
		!strings.Contains(src, `ezfyTechItem("武器科技", defTech[6]*3)`) {
		t.Fatal("【守方科技】行缺军训艺术/武器科技（应与【攻方科技】同集合）")
	}
}

// TestFourTechCategoriesWired 四类科技加成必须在**攻守双方**都真实参与计算。
//
// ★★ 2026-10-08 用户要求：「攻击加成的科技、防御加成科技、速度加成科技、射程加成科技
// 这里都要展示，而且回合中也要实际有」。
//
//	展示侧由【攻方科技】/【守方科技】两行负责（同一集合，见 TestDefenderAtkTechMatchesAttacker）；
//	这里锁**计算侧** —— 每类科技都要在 ezfy_order.go 的加成表达式里出现，
//	否则就会出现「列表里列了、回合里不生效」的假展示。
//
// 科技 ID 对照：5 军训艺术(攻) · 6 武器科技(攻) · 7 装甲科技(防) · 8 弹道学(射程) ·
// 9 重工技术(重装备攻防) · 10 燃烧引擎(速度) · 16 掩体防御(城防攻防) · 19 喷气引擎(空军速度)
func TestFourTechCategoriesWired(t *testing.T) {
	src := strings.ReplaceAll(rawFile(t, "ezfy_order.go"), "\r\n", "\n")
	// ★ 速度只列**通用**那部分（燃烧引擎 10「部队速度」）；喷气引擎(19)是「空军速度」，
	//   属兵种专属，见下面的 TestTypeSpecificBonusWired。
	atkCases := []struct{ name, want string }{
		{"攻击", "atkBonus := officerBonus + atkTech[5]*2 + atkTech[6]*3 + atkTech[9]*2"},
		{"防御", "atkDefBonus += atkTech[7]*3 + atkTech[16]*2 + atkTech[9]*2"},
		{"速度(通用)", "atkSpeedBonus := atkTech[10] * 2"},
		{"射程", "atkRangeBonus := atkTech[8] * 3"},
	}
	defCases := []struct{ name, want string }{
		{"攻击", "defAtkBonus = defTech[5]*2 + defTech[6]*3 + defTech[9]*2 + defTech[16]*2"},
		{"防御", "defBonus = h.buildingLevel(target.ID, 7)*5 + defTech[7]*3 + defTech[16]*2 + defTech[9]*2"},
		{"速度(通用)", "defSpeedBonus = defTech[10] * 2"},
		{"射程", "defRangeBonus = defTech[8]*3 + defTech[16]*2"},
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
}

// TestTypeSpecificBonusWired 兵种专属加成必须**按兵种**下发，不能混进通用加成。
//
// ★★ 2026-10-08 用户确认：「玩家科技、军官技能有的是全部兵种，有的是指定兵种，不能一起加」。
//
//	带兵种的科技：喷气引擎(19)「空军速度+3%/级」；
//	带兵种的军官技能：火炮控制(陆军装甲攻击) / 四指编队(空军对空攻击) / 狼群战术(海军对海攻击) /
//	坦克突袭(陆军速度) / 闪电袭击(空军速度) / 越岛战术(海军速度)。
//	它们必须走 ezfyTypeBonus（Atk/Speed 按 troop type 索引），且**不得**出现在通用加成里。
func TestTypeSpecificBonusWired(t *testing.T) {
	ord := strings.ReplaceAll(rawFile(t, "ezfy_order.go"), "\r\n", "\n")
	off := strings.ReplaceAll(rawFile(t, "ezfy_officer.go"), "\r\n", "\n")
	bat := strings.ReplaceAll(rawFile(t, "ezfy_battle.go"), "\r\n", "\n")

	// ① 喷气引擎（空军速度）→ 只进空军
	for _, want := range []string{
		"atkType.Speed[ezfyTroopTypeAir] += v", // 攻方
		"defType.Speed[ezfyTroopTypeAir] += v", // 守方
	} {
		if !strings.Contains(ord, want) {
			t.Fatalf("喷气引擎（空军速度）未按兵种下发：%s", want)
		}
	}
	// ② 通用速度不得再包含喷气引擎（那是空军专属）
	for _, bad := range []string{
		"atkSpeedBonus := atkTech[10]*2 + atkTech[19]*3",
		"defSpeedBonus = defTech[10]*2 + defTech[19]*3",
	} {
		if strings.Contains(ord, bad) {
			t.Fatalf("通用速度不该含喷气引擎(19)（那是空军专属）：%s", bad)
		}
	}
	// ③ 军官的六个兵种技能必须按兵种下发
	for _, want := range []string{
		"atk[ezfyTroopTypeArmy] += 10 * scale",   // 火炮控制 → 陆军
		"atk[ezfyTroopTypeAir] += 15 * scale",    // 四指编队 → 空军
		"atk[ezfyTroopTypeNavy] += 15 * scale",   // 狼群战术 → 海军
		"speed[ezfyTroopTypeArmy] += 10 * scale", // 坦克突袭 → 陆军
		"speed[ezfyTroopTypeAir] += 10 * scale",  // 闪电袭击 → 空军
		"speed[ezfyTroopTypeNavy] += 10 * scale", // 越岛战术 → 海军
	} {
		if !strings.Contains(off, want) {
			t.Fatalf("军官兵种技能未按兵种下发：%s", want)
		}
	}
	// ④ 通用技能加成里不得再出现兵种技能（尖兵突击是全体兵种，保留）
	if !strings.Contains(off, `if s == "尖兵突击" {`) {
		t.Fatal("通用技能加成应只保留「尖兵突击」（全体兵种）")
	}
	if i := strings.Index(off, "func (h *EzfyHandler) officerSkillBattleBonus"); i >= 0 {
		seg := off[i:]
		if j := strings.Index(seg, "// officerSkillsBreak"); j > 0 {
			seg = seg[:j]
		}
		for _, bad := range []string{"火炮控制", "四指编队", "狼群战术"} {
			if strings.Contains(seg, bad) {
				t.Fatalf("通用技能加成里不该再出现兵种技能：%s", bad)
			}
		}
	}
	// ⑤ 战斗引擎必须按兵种取值叠加（攻击 + 速度，攻守双方）
	for _, want := range []string{
		"atkBonus + st.AtkType.atkOf(unit.cfg.Type)",
		"st.DefAtkBonus + st.DefType.atkOf(unit.cfg.Type)",
		"atkSpeedBonus + st.AtkType.speedOf(unit.cfg.Type)",
		"defSpeedBonus + st.DefType.speedOf(unit.cfg.Type)",
	} {
		if !strings.Contains(bat, want) {
			t.Fatalf("战斗引擎未按兵种叠加加成：%s", want)
		}
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

// TestWildDefSkillBonus 野地/寇城守将防御类技能（弧形防御/弹幕支援）计入守军防御加成并逐项拆解。
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
	return string(b)
}
