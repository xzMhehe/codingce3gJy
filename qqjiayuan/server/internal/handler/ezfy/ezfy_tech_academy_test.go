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
