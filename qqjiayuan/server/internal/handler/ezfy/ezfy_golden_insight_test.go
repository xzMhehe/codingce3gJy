package ezfy

import (
	"strings"
	"testing"

	"qqjiayuan/server/internal/model"
)

// ============ 2026-10-31 黄金洞察（原黄金眼/黄金瞳） ============
//
// 用户口径：
//
//	「他应该就是相当于提高侦察等级的，比如我侦察 1 级，带个有该技能的军官，侦察等级就叠加，
//	  超了就叠加到最高级；相反当市长的时候可以提高情报等级，预警更清晰」。
//
// 也就是同一个技能的两面：
//   - 带队**出征侦查** → 加「侦察等级」（决定侦查报告能看清多少细节，scoutReportBody）
//   - 在城里**当市长** → 加「情报等级」（军情预警更详细，ezfyIntelLevel）
//
// 原来的「野地/寇城战利品 +10%/级」「掠夺率 +10%/级」已按用户口径**移除**。

// TestReconSkillLevel 技能等级即加成等级；没学 / 无军官 = 0。
func TestReconSkillLevel(t *testing.T) {
	h := &EzfyHandler{}
	// 150 级普通军官 → 技能 5 级
	o := &model.EzfyOfficer{Name: "T", Level: 150, Skill: `["黄金洞察"]`}
	if got := h.ezfyReconSkillLevel(o); got != 5 {
		t.Fatalf("150 级黄金洞察应为 +5 级，实际 %d", got)
	}
	// 没学该技能 → 0
	o2 := &model.EzfyOfficer{Name: "T2", Level: 150, Skill: `["尖兵猛攻"]`}
	if got := h.ezfyReconSkillLevel(o2); got != 0 {
		t.Fatalf("没学黄金洞察应为 0，实际 %d", got)
	}
	// nil 军官 → 0
	if got := h.ezfyReconSkillLevel(nil); got != 0 {
		t.Fatalf("无军官应为 0，实际 %d", got)
	}
}

// TestGoldenInsightWired 侦察等级 / 情报等级两条链路都必须接上技能，且封顶 10。
func TestGoldenInsightWired(t *testing.T) {
	ord := strings.ReplaceAll(rawFile(t, "ezfy_order.go"), "\r\n", "\n")
	api := strings.ReplaceAll(rawFile(t, "ezfy_api.go"), "\r\n", "\n")

	// ① 出征侦查：报告细节等级 = 无线电侦听 + 带队军官技能等级（封顶 10）
	scout := ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) scoutReportBody(")
	for _, want := range []string{
		"h.techMap(uint(order.CityId))[ezfyReconTechID]",
		"h.ezfyReconSkillLevel(h.officerByName(uint(order.CityId), order.Officer))",
		"if scoutLv > ezfyIntelMaxLevel {",
	} {
		if !strings.Contains(scout, want) {
			t.Fatalf("侦查报告没接上黄金洞察（缺 %s）：\n%s", want, scout)
		}
	}

	// ② 市长：情报等级 = 雷达站 + 无线电侦听 + 市长技能（封顶 10）
	intel := ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) ezfyIntelLevel(")
	for _, want := range []string{
		"h.ezfyMayorReconLevel(cityID)",
		"if lv > ezfyIntelMaxLevel {",
	} {
		if !strings.Contains(intel, want) {
			t.Fatalf("情报等级没接上市长技能（缺 %s）：\n%s", want, intel)
		}
	}
	// 市长加成必须按「本城市长」取（position = 市长、非俘虏）
	mayor := ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) ezfyMayorReconLevel(")
	for _, want := range []string{
		"position = ?", "ezfyPositionMayor", "h.ezfyReconSkillLevel(&o)",
	} {
		if !strings.Contains(mayor, want) {
			t.Fatalf("市长加成取错了军官（缺 %s）：\n%s", want, mayor)
		}
	}

	// ③ 军情逐条下发（批量）也要同口径 —— 不能只有战报正文加了、接口没加
	if !strings.Contains(api, "mayorRecon[int64(mayors[i].CityId)] = lv") {
		t.Fatalf("军情逐条情报等级没接市长技能：\n%s", api)
	}
	if !strings.Contains(api, "intel = radar + recon + mayorRecon[cid]") {
		t.Fatalf("军情逐条情报等级没把市长加成算进去：\n%s", api)
	}

	// ④ 旧效果（战利品 / 掠夺率）必须彻底移除
	if strings.Contains(ord, `officerHasSkill(leadOfficer, "黄金洞察")`) {
		t.Fatalf("黄金洞察不该再加战利品/掠夺率（已改为加侦察/情报等级）：\n%s", ord)
	}
}
