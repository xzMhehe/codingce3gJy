package ezfy

import (
	"strings"
	"testing"
)

// TestTechLevel10RequiresAcademy10 科技 9 级 → 10 级（满级）必须科研中心 10 级。
//
// ★★ 2026-10-13 用户反馈「线上发现玩家不升级科研中心 10 级」：
//
//	原实现只按 techId 查 ezfyTechAcademy（「这条科技能不能研究」的**固有门槛**），
//	与等级无关 → 门槛低的科技（如科技 1 只要科研中心 1 级）可以一路升到 10 级，
//	科研中心 10 级变成没人升的摆设。
//	现在科研中心等级要求按**目标等级**算：目标 ≥ 10 时门槛抬到 10。
//	历史已到 10 级的玩家不动（只在开始研究时校验）。
func TestTechLevel10RequiresAcademy10(t *testing.T) {
	// ① 规则必须只有一处实现，且含「目标等级 ≥ 10 → 10 级」的判定
	src := strings.ReplaceAll(rawFile(t, "ezfy.go"), "\r\n", "\n")
	i := strings.Index(src, "func ezfyTechAcademyNeed(")
	if i < 0 {
		t.Fatal("缺少 ezfyTechAcademyNeed（科研中心等级要求的唯一实现）")
	}
	fn := src[i:]
	if j := strings.Index(fn, "\nfunc "); j > 0 {
		fn = fn[:j]
	}
	if !strings.Contains(fn, "ezfyTechAcademyTopLevel") {
		t.Fatalf("ezfyTechAcademyNeed 没有把「目标等级≥10」纳入门槛：\n%s", fn)
	}
	if !strings.Contains(fn, "ezfyTechAcademy[techId]") {
		t.Fatalf("ezfyTechAcademyNeed 丢了该科技的固有门槛：\n%s", fn)
	}
	if ezfyTechAcademyTopLevel != 10 {
		t.Fatalf("科技满级门槛应为 10 级，当前 %d", ezfyTechAcademyTopLevel)
	}

	// ② 行为校验：门槛低的科技升到 10 级必须要求科研中心 10 级；9 级仍按固有门槛
	if got := ezfyTechAcademyNeed(1, 10); got != 10 {
		t.Fatalf("科技1 升 10 级应需科研中心 10 级，实际 %d", got)
	}
	if got := ezfyTechAcademyNeed(1, 9); got != ezfyTechAcademy[1] {
		t.Fatalf("科技1 升 9 级不应被满级门槛影响，实际 %d", got)
	}
	if got := ezfyTechAcademyNeed(17, 10); got != 10 {
		t.Fatalf("固有门槛已是 10 的科技不应被改小，实际 %d", got)
	}

	// ③ 卡控点（researchTech）必须走这个函数、且按**下一级**算，不能退回 ezfyTechAcademy[techId]
	rt := src[strings.Index(src, "func (h *EzfyHandler) researchTech("):]
	rt = rt[:strings.Index(rt, "func (h *EzfyHandler) cancelTech(")]
	if !strings.Contains(rt, "ezfyTechAcademyNeed(techId, curLevel+1)") {
		t.Fatalf("researchTech 没有按目标等级校验科研中心（缺 ezfyTechAcademyNeed(techId, curLevel+1)）：\n%s", rt)
	}
	if strings.Contains(rt, "ezfyTechAcademy[techId]") {
		t.Fatalf("researchTech 仍在用只按 techId 的旧门槛：\n%s", rt)
	}
	// 卡控必须在「开始研究」入口（researchTech），不能在结算时补
	if !strings.Contains(rt, "blv[8] < academyNeed") {
		t.Fatalf("researchTech 缺少科研中心等级卡控：\n%s", rt)
	}

	// ④ 界面「前提：科研中心N级」必须与后端同口径（否则前端显示 1 级、点了报错）
	api := strings.ReplaceAll(rawFile(t, "ezfy_api.go"), "\r\n", "\n")
	techs := api[strings.Index(api, "func (h *EzfyHandler) Techs("):]
	techs = techs[:strings.Index(techs, "func (h *EzfyHandler) Research(")]
	if strings.Contains(techs, "ezfyTechAcademy[t.ID]") {
		t.Fatalf("Techs 接口仍返回旧口径的 academy_need（应为 ezfyTechAcademyNeed(t.ID, level+1)）：\n%s", techs)
	}
	if strings.Count(techs, "ezfyTechAcademyNeed(t.ID, level+1)") != 2 {
		t.Fatalf("Techs 的两处（研究中/未研究）都应用新口径：\n%s", techs)
	}
}

// TestTechBonusRequiresAcademy 没有「科研中心」建筑的城市**不享受任何科技加成**。
//
// ★★ 2026-10-29 用户规则（当面明确）：科技等级属于玩家（`ezfy_user_tech` 按 user_id），
// 但**生效按城市判** —— 本城建筑 8（科研中心）等级为 0 时，该城所有科技加成按 0 级算
// （产量/训练/采集/载重/侦察/建造速度…）。科技与城市只有两处关系：
// ① 研究由某城发起（队列 ezfy_city_tech）② 本测试这条「没科研中心不享受加成」。
//
// ★ 用户端**不加任何提示文案**，规则静默生效（别在页面上加「本城无科研中心…」之类冗余提示）。
func TestTechBonusRequiresAcademy(t *testing.T) {
	src := strings.ReplaceAll(rawFile(t, "ezfy.go"), "\r\n", "\n")
	// ① 常量：科研中心 = 建筑 8
	if !strings.Contains(src, "ezfyAcademyBuildingID = 8") {
		t.Fatal("缺少常量 ezfyAcademyBuildingID = 8（科研中心建筑 ID）")
	}
	// ② techMap 必须按「本城科研中心等级 > 0」判定，否则返回空表
	tm := stripGoComments(ezfyFuncBody(t, "ezfy.go", "func (h *EzfyHandler) techMap(cityId uint) map[int]int {"))
	for _, want := range []string{"ezfy_city_building", "ezfyAcademyBuildingID", "return map[int]int{}"} {
		if !strings.Contains(tm, want) {
			t.Fatalf("techMap 没把「本城有无科研中心」纳入判定（缺 %s）：\n%s", want, tm)
		}
	}
	// ③ 纯内存版：用已取到的建筑列表判定（产量快照等零额外 SQL 场景）
	hm := stripGoComments(ezfyFuncBody(t, "ezfy.go", "func ezfyBuildingsHaveAcademy(buildings []model.EzfyCityBuilding) bool {"))
	if !strings.Contains(hm, "ezfyAcademyBuildingID") || !strings.Contains(hm, "Level > 0") {
		t.Fatalf("ezfyBuildingsHaveAcademy 判定口径不对：\n%s", hm)
	}
	// ④ 四个产量/结算快照都必须经 ezfyTechsPick 按城过滤（不能直接把账号级 techs 喂进去）
	for _, f := range []string{"ezfy.go", "ezfy_api.go", "ezfy_chat_exchange.go"} {
		if c := strings.Count(strings.ReplaceAll(rawFile(t, f), "\r\n", "\n"), "ezfyTechsPick("); c == 0 {
			t.Fatalf("%s 里没有用 ezfyTechsPick 按城过滤科技加成", f)
		}
	}
	// ⑤ 批量结算的负重科技缓存必须按**城**（同一批可能有多个出发城，科研中心情况不同）
	env := stripGoComments(ezfyFuncBody(t, "ezfy_settle_env.go", "func (h *EzfyHandler) envTechs(e *ezfySettleEnv, cityId uint) map[int]int {"))
	if !strings.Contains(env, "techsByCity[cityId]") {
		t.Fatalf("envTechs 没按城市缓存（应 key = 出发城ID）：\n%s", env)
	}
	if strings.Contains(env, "techMapOf(") {
		t.Fatalf("envTechs 仍直接用账号级 techMapOf（会绕过科研中心判定）：\n%s", env)
	}
	// ⑥ 采集的负重判定必须按订单的出发城取科技
	col := stripGoComments(rawFile(t, "ezfy_collect.go"))
	if !strings.Contains(col, "academyOf[uint(order.CityId)]") {
		t.Fatal("一键采集的负重判定没按出发城的科研中心过滤科技加成")
	}
	// ⑦ 公共取数入口 ezfyLoadCityData（~30 处 refreshCity* 都走它）必须按城过滤
	lcd := stripGoComments(ezfyFuncBody(t, "ezfy.go", "func (h *EzfyHandler) ezfyLoadCityData(uid uint, city *model.EzfyCity, cities []model.EzfyCity, extra ...func()) *resCalcData {"))
	if !strings.Contains(lcd, "ezfyTechsPick(") {
		t.Fatalf("ezfyLoadCityData 没按本城科研中心过滤科技表（所有懒结算都会吃错加成）：\n%s", lcd)
	}
	// ⑧ 出征预览的负重（标准化装卸）也必须按出发城判定
	ord := stripGoComments(rawFile(t, "ezfy_order.go"))
	if !strings.Contains(ord, "h.buildingLevel(city.ID, ezfyAcademyBuildingID)") {
		t.Fatal("出征预览的负重科技没按出发城的科研中心过滤")
	}
}
