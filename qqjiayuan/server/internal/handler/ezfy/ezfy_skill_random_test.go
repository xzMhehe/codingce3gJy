package ezfy

import (
	"strings"
	"testing"

	"qqjiayuan/server/internal/model"
)

// ============ 军官技能「随机学习」守卫（2026-10-25）============
//
// 用户规则（原话）：「点学习就是随机学一个军官技能」「消耗军官技能书随机学一个技能，
// 而不是军官指定技能学习消耗」。
//
// 两条入口都必须走随机：
//  1. 军官详情 [学习] → POST /officers/:id/skill {op:"learn"}（OfficerSkill → learnRandomSkill）
//  2. 背包里直接用「军官技能书」→ useItem ItemType 11
//
// 护栏目的：防止以后有人为了「方便」把 skill_id 又接回去（指定学习 = 玩家必刷最强技能）。

// TestLearnSkillIsRandom —— 两条学习入口都不再按 skill_id 选技能
func TestLearnSkillIsRandom(t *testing.T) {
	// ① 军官详情入口
	body := ezfyFuncBody(t, "ezfy_officer.go", "func (h *EzfyHandler) learnRandomSkill(")
	if !strings.Contains(body, "ezfyRandomSkillToLearn(skills)") {
		t.Fatal("learnRandomSkill 必须调 ezfyRandomSkillToLearn 随机挑技能（不能按 skill_id 指定）")
	}
	if strings.Contains(body, "ezfyCfg.skill(") {
		t.Fatal("learnRandomSkill 里不该再出现 ezfyCfg.skill(...)：学技能已改为随机，不能再按 id 取指定技能")
	}
	if !strings.Contains(body, "ezfySkillBookItemID()") {
		t.Fatal("learnRandomSkill 必须消耗「军官技能书」（ezfySkillBookItemID）")
	}

	// ② 背包道具入口（ItemType 11）
	use := ezfyFuncBody(t, "ezfy.go", "func (h *EzfyHandler) useItemOnce(")
	if !strings.Contains(use, "ezfyRandomSkillToLearn(skills)") {
		t.Fatal("useItemOnce 的 ItemType 11（军官技能书）也必须随机学一个技能")
	}
	if strings.Contains(use, "ezfyCfg.skill(skillId)") {
		t.Fatal("useItemOnce 里不该再按 skill_id 指定技能（学技能已改为随机）")
	}

	// ③ 随机候选必须「排除已学」且「候选先排序再取」——否则可能重复学同一个技能
	pick := ezfyFuncBody(t, "ezfy_officer.go", "func ezfyRandomSkillToLearn(")
	if !strings.Contains(pick, "learned[s.Name]") {
		t.Fatal("ezfyRandomSkillToLearn 必须排除该军官已学的技能")
	}
	if !strings.Contains(pick, "sort.Slice(") {
		t.Fatal("ezfyRandomSkillToLearn 候选要先按 ID 排序再随机取（range map 顺序不定，不可复现）")
	}
}

// TestRandomSkillToLearnPicksUnlearned —— 行为测试：随机到的技能一定「没学过」，且真的随机
//
// 直接塞一个假的技能池（不动数据库），连续抽 200 次：
//   - 已学「尖兵猛攻」永远不能被抽到（否则会重复学、白扣技能书）
//   - 池里 4 个未学技能都要有机会被抽到（不是恒取第一个）
func TestRandomSkillToLearnPicksUnlearned(t *testing.T) {
	old := ezfyCfg
	defer func() { ezfyCfg = old }()
	ezfyCfg.skills = map[int]model.EzfyCfgSkill{
		1: {ID: 1, Name: "尖兵猛攻"},
		2: {ID: 2, Name: "弧形壁垒"},
		3: {ID: 3, Name: "绝地逆袭"},
		4: {ID: 4, Name: "钢铁洪流"},
		5: {ID: 5, Name: "夜战突袭"},
	}
	seen := map[string]int{}
	for i := 0; i < 200; i++ {
		sk := ezfyRandomSkillToLearn([]string{"尖兵猛攻"})
		if sk == nil {
			t.Fatal("还有未学技能时不该返回 nil")
		}
		if sk.Name == "尖兵猛攻" {
			t.Fatal("随机到了已学技能：会重复学、白扣 1 本技能书")
		}
		seen[sk.Name]++
	}
	if len(seen) != 4 {
		t.Fatalf("4 个未学技能都该有机会被抽到，实际只出现 %d 种：%v", len(seen), seen)
	}

	// 学满 3 个后仍能抽（上限由调用方卡），但把整个池学完必须返回 nil（防死循环/空指针）
	all := []string{"尖兵猛攻", "弧形壁垒", "绝地逆袭", "钢铁洪流", "夜战突袭"}
	if sk := ezfyRandomSkillToLearn(all); sk != nil {
		t.Fatalf("技能池学完时应返回 nil，实际返回 %q", sk.Name)
	}
}
