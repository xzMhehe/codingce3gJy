package ezfy

import (
	"strings"
	"testing"
)

// TestWildlandPvp 附属野地的一批 bug 修复（用户 2026-10-19），源级接线断言：
//
//	① 附属野地增援部队到了要展示在「军情 → 驻军」；
//	② 自己军团（同盟）成员的附属野地不能出征/掠夺/征服；
//	③ 被攻击方附属野地增援带的军官，攻方打的时候要能显示；
//	④ 附属野地被打的守方战报要带逐回合详情（[展开/收起逐回合详情]）；
//	⑤ 有驻军的附属野地不能直接结算，要像打玩家城市一样进指挥室（def_user_id + 守方 [指挥]）。
func TestWildlandPvp(t *testing.T) {
	order := rawFile(t, "ezfy_order.go")
	api := rawFile(t, "ezfy_api.go")
	room := rawFile(t, "ezfy_battle_room.go")

	// ① 驻军 tab：status=3 增援单要覆盖附属野地（target_type 1/2）
	if !strings.Contains(api, `"((target_type = 3 AND target_id IN (?)) OR target_type IN (1,2))))"`) {
		t.Fatal("ReportDynamics 未把「附属野地驻军(target_type 1/2)」纳入 status=3 分支")
	}

	// ② 军团成员的附属野地不能打：同一野地主人城走 isAllyCity 判定
	if !strings.Contains(order, "if h.isAllyCity(uid, int64(oc.ID)) {") {
		t.Fatal("createOrder 未拦截「同盟(军团)成员的附属野地」")
	}
	if !strings.Contains(order, `return "不能" + ezfyOrderTypeName(orderType) +`) {
		t.Fatal("缺少「不能攻击同盟成员附属野地」的提示")
	}

	// ③ 守军带队军官要显示：聚合 + 战报行 + 指挥室 head
	if !strings.Contains(order, "func (h *EzfyHandler) ezfyWildDefOfficersDesc(") {
		t.Fatal("缺少 ezfyWildDefOfficersDesc（汇总附属野地守军带队军官）")
	}
	if !strings.Contains(order, "wildDefOfficerDesc = h.ezfyWildDefOfficersDesc(wildDefOrders)") {
		t.Fatal("附属野地守军分支未汇总带队军官")
	}
	// ★ 2026-10-19 军官行改挂在 [攻方]/[守方] 标题下（用户：「看不出哪一方的」）
	if !strings.Contains(order, `report += "军官:" + officerReportDesc(leadOfficer) + "\n"`) {
		t.Fatal("攻方战报未在 [攻方] 块展示带队军官")
	}
	if !strings.Contains(order, `report += "军官:" + wildDefOfficerDesc + "\n"`) {
		t.Fatal("攻方战报未在 [守方] 块展示野地增援带的军官")
	}
	if !strings.Contains(order, "defOfficerDesc = wildDefOfficerDesc") {
		t.Fatal("指挥室「准备回合」未把野地守军军官写进 defOfficerDesc")
	}

	// ④ 守方战报带逐回合详情 + **守方视角**：addWildlandOwnerReport 现搭守方正文并透传 detail
	if !strings.Contains(order, "atkCity *model.EzfyCity, targetName string, win bool, detail string, now int64,") {
		t.Fatal("addWildlandOwnerReport 未接收 detail/now（守方战报没有逐回合详情 / 模板参数）")
	}
	if !strings.Contains(order, `h.addReport(ownerUID, reportType, title, head+"\n\n"+body, detail, order.ID, ownerCityID)`) {
		t.Fatal("守方战报未把 detail 写进战报（前端不显示 [展开/收起逐回合详情]）")
	}
	if !strings.Contains(order, "敌方一支部队对 %s[ %d，%d ]进行了%s") {
		t.Fatal("守方战报仍是攻方视角（应为「敌方一支部队对…」）")
	}
	if strings.Count(order, ", targetName, true, detail, now,") < 1 ||
		strings.Count(order, ", targetName, win, detail, now,") < 1 {
		t.Fatal("addWildlandOwnerReport 的两处调用未传 detail/now")
	}
	// ④b 守方也有军官经验/军功声望（用户 2026-10-19：「军官经验、声望补下吧应该有的」）
	if !strings.Contains(order, "h.addWildGarrisonOfficerExp(wildDefOrders, wlOwnerCityID, wildDefExp)") {
		t.Fatal("野地守军带队军官未发战斗经验")
	}
	if !strings.Contains(order, "func (h *EzfyHandler) ezfyDefenderReportBody(s ezfyDefReportSpec) string {") {
		t.Fatal("缺少城市/野地共用的守方视角正文生成器 ezfyDefenderReportBody")
	}

	// ⑤ 有玩家守军的附属野地：像玩家城一样进指挥室，守方也能指挥
	//   · ezfyBattleStart / AutoFinish 记 def_user_id
	if strings.Count(room, "defUID, _ = h.ezfyWildlandOwnerOf(order.TargetX, order.TargetY, uid)") < 2 {
		t.Fatal("开战场/自动结算未给附属野地记 def_user_id（守方无法指挥）")
	}
	//   · 守方身份判定：DefUserID 优先、不再限定 target_type=3
	if !strings.Contains(room, "if b.DefUserID > 0 {\n\t\treturn b.DefUserID\n\t}") {
		t.Fatal("ezfyBattleDefenderUid 未优先返回 DefUserID（附属野地守方被判为无关人员）")
	}
	//   · 有真人守方即禁用一键自动战斗（与玩家城同口径）
	if strings.Count(room, "h.ezfyBattleDefenderUid(b) > 0") < 2 {
		t.Fatal("pvp/自动战斗未覆盖「有驻军的附属野地」")
	}
	//   · 守方的「军情 → 军队动态」要能看到 [指挥]（按归属城映射野地战场）
	// ★★ 2026-10-20 攻打野地的订单 target_id=0，必须按坐标映射；只按 target_id
	//   会导致守方永远看不到这场战斗。
	if !strings.Contains(api, "target_type IN (1,2) AND EXISTS (SELECT 1 FROM ezfy_wildland w ") ||
		!strings.Contains(api, "WHERE w.x = ezfy_battle.target_x AND w.y = ezfy_battle.target_y AND w.city_id = ?") {
		t.Fatal("ReportDynamics 防守战场未按坐标把附属野地战场映射到归属城（守方看不到 [指挥]）")
	}
	//   · 旧野地战场 def_user_id=0 时，也要按坐标反查野地主人，否则守方进指挥室被判无关人员
	if !strings.Contains(room, "if b.TargetType == 1 || b.TargetType == 2 {\n\t\tuid, _ := h.ezfyWildlandOwnerOf(b.TargetX, b.TargetY, b.UserID)") {
		t.Fatal("ezfyBattleDefenderUid 未按坐标兼容旧附属野地战场（守方无法指挥）")
	}
	//   · 守方轮询要能触发开战场（fetchIncoming 纳入打我方野地的到达订单）
	// ★★ 2026-10-20 同样不能用 target_id=0 去 JOIN，改按坐标 EXISTS。
	if !strings.Contains(order, "(ezfy_order.target_type IN (1,2) AND EXISTS (") ||
		!strings.Contains(order, "WHERE w.x = ezfy_order.target_x ") ||
		!strings.Contains(order, "AND w.y = ezfy_order.target_y AND w.city_id IN ?") {
		t.Fatal("fetchIncoming 未按坐标纳入「正在打我方附属野地」的到达订单（守方轮询推不动战场）")
	}
	if !strings.Contains(order, "军情警报: 敌军已抵达 来自 %s(%d,%d)") ||
		!strings.Contains(order, "if hasPlayerDefender && (order.TargetType == 1 || order.TargetType == 2) {") {
		t.Fatal("附属野地有守军时未给守方发「敌军已抵达」军情警报")
	}

	// ⑥ 附属野地战斗中的驻军/采集队不能召回（单支 + 一键召回都要卡）
	if !strings.Contains(order, "func (h *EzfyHandler) ezfyWildlandOrderInBattle(uid uint, order *model.EzfyOrder) bool {") {
		t.Fatal("缺少 ezfyWildlandOrderInBattle（无法判断野地守军是否在战斗中）")
	}
	if !strings.Contains(order, "if h.ezfyWildlandOrderInBattle(uid, &order) {") {
		t.Fatal("单支召回未拦截「附属野地战斗中的部队」")
	}
	col := rawFile(t, "ezfy_collect.go")
	if !strings.Contains(col, "activeWildBattles := h.ezfyWildlandBattleCoords(uid)") ||
		!strings.Contains(col, "activeWildBattles[[2]int{order.TargetX, order.TargetY}]") {
		t.Fatal("一键召回未拦截「附属野地战斗中的部队」")
	}
	if !strings.Contains(col, "支部队因正在战斗未能召回") {
		t.Fatal("一键召回未回报被跳过的战斗中部队")
	}
}
