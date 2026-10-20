package ezfy

import (
	"strings"
	"testing"
)

// TestEzfyWildDefCombatBonusTech 附属野地守军的科技加成必须与玩家城守方同口径。
//
// 用户反馈（2026-10-20）：自己是守方，军官带着航母在采集，但战报/指挥室显示
// 【守方军官】只有名字、攻击/防御 0%，【守方科技】无。这里用 nil 军官验证
// 科技部分（不依赖数据库/军官实例），确保归属城的科技能算出非零加成。
func TestEzfyWildDefCombatBonusTech(t *testing.T) {
	h := &EzfyHandler{}
	tech := map[int]int{
		5: 10, 6: 10, 7: 10, 8: 10, 9: 10, 10: 10, 16: 10, 19: 10, 21: 10,
	}
	b := h.ezfyWildDefCombatBonus(tech, nil)

	if b.DefBonus != 50 { // 装甲科技 30 + 重工技术 20
		t.Fatalf("守方防御加成 = %d, 期望 50", b.DefBonus)
	}
	if b.DefAtkBonus != 70 { // 军训 20 + 武器 30 + 重工 20
		t.Fatalf("守方攻击加成 = %d, 期望 70", b.DefAtkBonus)
	}
	if b.DefSpeedBonus != 20 { // 燃烧引擎 20
		t.Fatalf("守方速度加成 = %d, 期望 20", b.DefSpeedBonus)
	}
	if b.DefRangeBonus != 30 { // 弹道学 30
		t.Fatalf("守方射程加成 = %d, 期望 30", b.DefRangeBonus)
	}
	if b.DefHealTech != 20 { // 治愈伤兵 20
		t.Fatalf("守方伤兵回收科技 = %d, 期望 20", b.DefHealTech)
	}
	if b.Type.Speed[ezfyTroopTypeAir] != 30 { // 喷气引擎只加空军
		t.Fatalf("守方空军速度 = %d, 期望 30", b.Type.Speed[ezfyTroopTypeAir])
	}
	if b.Type.Def[ezfyTroopTypeCity] != 20 { // 掩体防御只加城防单位
		t.Fatalf("守方城防防御 = %d, 期望 20", b.Type.Def[ezfyTroopTypeCity])
	}
	if len(b.TechBreak) != 8 {
		t.Fatalf("守方科技明细 %d 项, 期望 8 项（军训/武器/装甲/弹道/重工/燃烧/掩体/喷气）", len(b.TechBreak))
	}
	if len(b.DefBreak) != 2 {
		t.Fatalf("守方防御明细 %d 项, 期望 2 项（装甲科技/重工技术）", len(b.DefBreak))
	}
	if b.OfficerDesc != "" {
		t.Fatalf("无军官时不应有军官描述, 实际 %q", b.OfficerDesc)
	}
}

// TestEzfyWildDefBonusWiring 附属野地守军必须把「主将军官 + 归属城科技」接进战斗引擎。
//
// 该测试是源码接线断言：processArrive 的野地分支必须：
//  1. 用归属城 techMap 与主将计算 ezfyWildDefCombatBonus；
//  2. 把 defBonus/defAtkBonus/defTechBreak/defDefBreak/defEquip/defType 等全部赋值；
//  3. 指挥室/战报里的【守方军官】使用主将的完整加成描述；
//  4. 绝地反击/套装/技能也使用主将，而不是继续传 nil 或只传城守。
func TestEzfyWildDefBonusWiring(t *testing.T) {
	order := rawFile(t, "ezfy_order.go")
	for _, want := range []string{
		"wildDefCityID = uint(wildDef.CityId)",
		"wildDefCommander = h.ezfyWildDefCommander(wildDefOrders, wildDefCityID)",
		"wildDefTech := h.techMap(wildDefCityID)",
		"wildDefBonus := h.ezfyWildDefCombatBonus(wildDefTech, wildDefCommander)",
		"defBonus = wildDefBonus.DefBonus",
		"defAtkBonus = wildDefBonus.DefAtkBonus",
		"defSpeedBonus = wildDefBonus.DefSpeedBonus",
		"defRangeBonus = wildDefBonus.DefRangeBonus",
		"defHealTech = wildDefBonus.DefHealTech",
		"defOfficerAtkBonus = wildDefBonus.OfficerAtkBonus",
		"defEquip = wildDefBonus.Equip",
		"defSkillBreak = wildDefBonus.SkillBreak",
		"defTechBreak = wildDefBonus.TechBreak",
		"defDefBreak = wildDefBonus.DefBreak",
		"defType = wildDefBonus.Type",
		"defOfficerDesc = wildDefBonus.OfficerDesc",
		"defCounterRounds = h.officerCounterRounds(wildDefCommander)",
		"defSetOfficer = wildDefCommander",
		"h.officerSkillBattleBonus(leadOfficer), h.officerSkillBattleBonus(defSetOfficer)",
	} {
		if !strings.Contains(order, want) {
			t.Fatalf("附属野地守军加成未接线（缺 %q）", want)
		}
	}

	// 主将选择：等级最高、同等级取 id 小者，避免多支守军军官加成无序叠加。
	for _, want := range []string{
		`Order("level DESC, id ASC").Limit(1).Find(&os)`,
		"func (h *EzfyHandler) ezfyWildDefCommander(",
	} {
		if !strings.Contains(order, want) {
			t.Fatalf("附属野地守军主将选择缺失（缺 %q）", want)
		}
	}

	// 守军订单必须按**坐标**聚合（与 ReportDynamics / fetchIncoming 同口径）：
	// 野地行的 id 在征服/重建后可能变化，只按 target_id 会漏掉守军 → 加成整段归零。
	for _, want := range []string{
		`target_type <> 3 AND order_type IN (6,7) AND status IN (1,3) `,
		`AND target_x = ? AND target_y = ?", order.TargetX, order.TargetY`,
		"wildDefTech := h.techMap(wildDefCityID)",
		"if t := h.techMap(cid); len(t) > 0 {",
	} {
		if !strings.Contains(order, want) {
			t.Fatalf("附属野地守军坐标聚合/科技回退缺失（缺 %q）", want)
		}
	}

	// 公式必须包含与玩家城守方一致的科技/军官/装备项。
	body := ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) ezfyWildDefCombatBonus(")
	for _, want := range []string{
		"tech[7]*3 + tech[9]*2",
		"tech[5]*2 + tech[6]*3 + tech[9]*2",
		"tech[10] * 2",
		"tech[8] * 3",
		"h.officerGuardBonus(officer)",
		"h.officerBattleBonus(officer)",
		"h.officerBattleEquipBonus(officer)",
		`ezfyTechItem("军训艺术"`,
		`ezfyTechItem("武器科技"`,
		`ezfyTechItem("装甲科技"`,
		`ezfyTechItem("弹道学"`,
		`ezfyTechItem("重工技术"`,
		`ezfyTechItem("燃烧引擎"`,
		`ezfyTechItem("掩体防御"`,
		`ezfyTechItem("喷气引擎"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("附属野地守军加成公式缺少 %q", want)
		}
	}
}
