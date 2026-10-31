package ezfy

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"qqjiayuan/server/internal/model"
)

// readWebFile 读前端源码（路径相对 web/src），静态断言用。
func readWebFile(rel string) (string, error) {
	b, err := os.ReadFile("../../../../web/src/" + rel)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ============ 军官技能槽位（2026-10-31 用户反馈「遗忘技能2，技能3窜到第2行」） ============
//
// 口径：Skill 数组按**槽位固定长度**存，空槽写空串。
//   · 遗忘只清空该格（不整段前移）；
//   · 学习填**第一个空槽**（不无脑 append）；
//   · officerSkills 过滤空槽（战斗/属性逻辑只认真的学会的技能）。

// TestSkillSlotsForgetKeepsPosition 遗忘中间那一格后，第 3 个技能仍在第 3 行。
func TestSkillSlotsForgetKeepsPosition(t *testing.T) {
	o := &model.EzfyOfficer{Skill: `["尖兵突击","弧形防御","绝地反击"]`}
	raw := officerSkillSlotsRaw(o)
	if len(raw) != 3 {
		t.Fatalf("原始槽位应为 3 个，得到 %d", len(raw))
	}
	// 模拟 forgetSkill：把第 2 格清空（而不是删掉）
	raw[1] = ""
	b, _ := json.Marshal(raw)
	o.Skill = string(b)

	// ① 逻辑侧：只剩两个技能，且顺序保持
	got := officerSkills(o)
	if len(got) != 2 || got[0] != "尖兵突击" || got[1] != "绝地反击" {
		t.Fatalf("officerSkills 应为 [尖兵突击 绝地反击]，得到 %v", got)
	}
	// ② 展示侧：槽位不窜位 —— 第 2 格空、第 3 格仍是绝地反击
	slots := officerSkillSlotsRaw(o)
	if len(slots) != 3 || slots[0] != "尖兵突击" || slots[1] != "" || slots[2] != "绝地反击" {
		t.Fatalf("遗忘后槽位应保持 [尖兵突击, 空, 绝地反击]，得到 %v", slots)
	}
}

// TestSkillSlotsAppendFillsFirstEmpty 学习填第一个空槽，不把技能挤到末尾。
func TestSkillSlotsAppendFillsFirstEmpty(t *testing.T) {
	slots := []string{"尖兵突击", "", "绝地反击"}
	got := ezfySkillSlotsAppend(slots, "弧形防御")
	if len(got) != 3 || got[1] != "弧形防御" {
		t.Fatalf("新技能应填进第 2 个空槽，得到 %v", got)
	}
	// 已满 3 个 → 不覆盖任何已学技能
	full := []string{"A", "B", "C"}
	if out := ezfySkillSlotsAppend(full, "D"); len(out) != 3 || out[2] != "C" {
		t.Fatalf("技能已满时不该写入，得到 %v", out)
	}
	// 老数据紧凑数组（没有空槽）→ 追加到末尾
	if out := ezfySkillSlotsAppend([]string{"A"}, "B"); len(out) != 2 || out[1] != "B" {
		t.Fatalf("紧凑数组应追加，得到 %v", out)
	}
}

// TestOfficerSkillsSkipsEmptySlots 空槽不参与任何战斗/属性计算。
func TestOfficerSkillsSkipsEmptySlots(t *testing.T) {
	o := &model.EzfyOfficer{Skill: `["","弧形防御",""]`}
	if got := officerSkills(o); len(got) != 1 || got[0] != "弧形防御" {
		t.Fatalf("只应返回已学会的技能，得到 %v", got)
	}
	if got := officerSkills(&model.EzfyOfficer{Skill: ""}); len(got) != 0 {
		t.Fatalf("没有技能时应返回空，得到 %v", got)
	}
	if got := officerSkillSlotsRaw(nil); len(got) != 0 {
		t.Fatalf("nil 军官应返回空，得到 %v", got)
	}
}

// TestOfficerSkillViewBySlot 详情接口按**槽位**下发（前端据此渲染技能1/2/3）。
func TestOfficerSkillViewBySlot(t *testing.T) {
	// 后端拼装逻辑：slot = 下标+1，空槽 name=""，只给已学技能补 effect/des
	o := &model.EzfyOfficer{Skill: `["尖兵突击","","绝地反击"]`}
	raw := officerSkillSlotsRaw(o)
	n := ezfyOfficerMaxSkill
	if len(raw) > n {
		n = len(raw)
	}
	names := make([]string, 0, n)
	for i := 0; i < n; i++ {
		name := ""
		if i < len(raw) {
			name = strings.TrimSpace(raw[i])
		}
		names = append(names, name)
	}
	if len(names) != 3 || names[0] != "尖兵突击" || names[1] != "" || names[2] != "绝地反击" {
		t.Fatalf("详情槽位应为 [尖兵突击, 空, 绝地反击]，得到 %v", names)
	}
	// 效果文案随等级（3 级 = ×3）
	if got := ezfySkillEffectTextAt("尖兵突击", 3); got != "攻击力+90%" {
		t.Fatalf("尖兵突击 Lv.3 = %q", got)
	}
	// ★ 2026-10-31 口径修正后的效果文案
	cases := map[string]string{
		"火炮控制": "装甲部队攻击+30",
		"坦克突袭": "坦克速度+30%",
		"狼群战术": "海军攻击力+45%",
		"弹幕支援": "城防攻击范围+30%",
	}
	for skill, want := range cases {
		if got := ezfySkillEffectTextAt(skill, 3); got != want {
			t.Fatalf("%s Lv.3 效果文案应为 %q，得到 %q", skill, want, got)
		}
	}
}

// TestOfficerDetailFrontendWiring 前端详情页的交互/文案（静态断言，防止被改回去）。
//
//	· 已穿戴装备表收窄（ezfy-eq-worn-tbl）+ 操作文案 [卸] / [穿]；
//	· 技能/特殊技能点击展开、离开详情收起。
func TestOfficerDetailFrontendWiring(t *testing.T) {
	acade, err := readWebFile("views/ezfy/modules/EzfyAcademy.vue")
	if err != nil {
		t.Fatalf("读 EzfyAcademy.vue 失败: %v", err)
	}
	for _, want := range []string{
		"ezfy-eq-worn-tbl",                // 已穿戴装备表单独收窄
		">[卸]<",                           // 操作：[卸下] → [卸]
		">[穿]<",                           // 背包：[穿戴] → [穿]
		"ezfy.toggleOfficerSkill(",        // 技能点击展开/收起
		"ezfy.toggleOfficerCombo(",        // 特殊技能点击展开/收起
		"ezfy.officerSkillOpen === i + 1", // 只展开被点的那一格
		"ezfy.officerComboOpen === cs.id", // 只展开被点的那个组合技
	} {
		if !strings.Contains(acade, want) {
			t.Fatalf("EzfyAcademy.vue 缺详情页交互/文案：%s", want)
		}
	}
	page, err := readWebFile("views/ezfy/Ezfy.vue")
	if err != nil {
		t.Fatalf("读 Ezfy.vue 失败: %v", err)
	}
	for _, want := range []string{
		"resetOfficerExpand", // 离开详情收起
		"if (o === 'officerdetail' && n !== 'officerdetail')", // watch.cur 里复位
		"const slot = Number(s.slot) || (i + 1)",              // 按槽位渲染（不按下标）
	} {
		if !strings.Contains(page, want) {
			t.Fatalf("Ezfy.vue 缺技能槽位/展开复位逻辑：%s", want)
		}
	}
}
