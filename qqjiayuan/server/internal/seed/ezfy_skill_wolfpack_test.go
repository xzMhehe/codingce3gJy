package seed

import (
	"strings"
	"testing"
)

// TestWolfPackSkillTextNavyOnly 狼群围猎(8) 的文案必须写明**海军打海军**（目标也必须是海军兵种）。
//
// ★★ 2026-10-31 用户两轮反馈：
//
//	① 「海军集群突击，攻击力提升且不限攻击目标」→ 读起来像全军生效；
//	② 随后明确「目标也是对海军兵种」→ 定稿口径 = 攻击方海军 + 目标海军。
//
// 实现见 officerTypeBonus / generalTypeBonus 的 `addVsType(海军, 海军, 15*scale)`。
// 这里锁住文案：效果列必须带「对海」，描述里不能再出现「不限攻击目标」。
func TestWolfPackSkillTextNavyOnly(t *testing.T) {
	var effect, des string
	found := false
	for i := range ezfyEzfyCfgSkill {
		if ezfyEzfyCfgSkill[i].ID == 8 {
			effect, des = ezfyEzfyCfgSkill[i].Effect, ezfyEzfyCfgSkill[i].Des
			found = true
			break
		}
	}
	if !found {
		t.Fatal("种子技能表里找不到 id=8（狼群围猎）")
	}
	if !strings.Contains(effect, "对海") {
		t.Fatalf("狼群围猎效果列应写明「海军对海攻击」，实际 %q", effect)
	}
	if strings.Contains(des, "不限攻击目标") {
		t.Fatalf("狼群围猎描述不能再出现「不限攻击目标」（会让人以为全军生效），实际 %q", des)
	}
	if !strings.Contains(des, "海军") {
		t.Fatalf("狼群围猎描述必须写明只对海军兵种目标生效，实际 %q", des)
	}
}
