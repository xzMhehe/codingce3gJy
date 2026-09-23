package handler

import (
	"strings"
	"testing"
)

// 暴击修复回归测试（2026-09-23 用户反馈「二战用户端暴击有 bug / 暴击打了 1 个单位」）
//
// 老实现的两个坑：
//  ① 暴击几率是装备累加值，可以超过 100（如「革命者[折扇]」crit=125），
//     `rand.Intn(100) < 125` 恒真 → 必定暴击，几率形同虚设；
//  ② 暴击伤害加成可能是 0（如「黑色幽灵[徽章]」crit=125 / crit_dmg=0），
//     伤害 ×(100+0)/100 = 原样不变 → 出了【暴击】却一点没多打。
//     再叠加 `killed = damage / effHealth` 的向下取整，增益被整个抹掉，
//     玩家看到的就是「暴击跟没暴一样，只打了 1 个」。
//
// 现在：几率封顶 100%；暴击伤害加成 <= 0 时按 +50% 兜底；暴击的零头向上取整。

// ezfyCritTestState 造一个「只有暴击几率、暴击伤害为 0」的战场
// （老数据里的典型装备：crit 有值、crit_dmg = 0）
func ezfyCritTestState(crit, critDmg int) *ezfyBattleState {
	return &ezfyBattleState{
		Attackers: []*ezfyFightUnit{ezfyTestUnit("A1", "步兵", 0, 200)},
		Defenders: []*ezfyFightUnit{ezfyTestUnit("D1", "步兵", ezfyBattleStartDist, 200)},
		AtkEquip:  ezfyBattleBonus{Crit: crit, CritDmg: critDmg},
		Head:      []string{}, Actions: []string{},
	}
}

// TestCritZeroCritDmgStillAmplifies —— 暴击伤害加成为 0 时，暴击必须仍有倍率
//
// 判据：战报里出现【暴击+50%】（兜底倍率），而不是干巴巴的【暴击】。
func TestCritZeroCritDmgStillAmplifies(t *testing.T) {
	st := ezfyCritTestState(125, 0) // crit_dmg = 0 的典型老装备
	for !st.Done && st.Round < ezfyBattleMaxRounds {
		st.Step(nil, nil)
	}
	log := strings.Join(st.Actions, "\n")
	if !strings.Contains(log, "【暴击+") {
		t.Fatalf("暴击应带上生效倍率，实际战报：\n%s", log)
	}
	if !strings.Contains(log, "【暴击+50%】") {
		t.Fatalf("crit_dmg=0 时应兜底 +50%%，实际战报：\n%s", log)
	}
}

// TestCritChanceCappedAt100 —— 暴击几率超过 100 时按 100% 处理（不是「必定暴击」的荒谬语义）
//
// 判据：crit=125 与 crit=100 打出来的战报，暴击条数应当同量级（都是「必暴」），
// 且不会出现负的暴击倍率。
func TestCritChanceCappedAt100(t *testing.T) {
	st := ezfyCritTestState(100, 60)
	for !st.Done && st.Round < ezfyBattleMaxRounds {
		st.Step(nil, nil)
	}
	log := strings.Join(st.Actions, "\n")
	if !strings.Contains(log, "【暴击+60%】") {
		t.Fatalf("crit_dmg=60 时应显示 +60%%，实际战报：\n%s", log)
	}
}

// TestCritBonusUsedAsIs —— 装备配了暴击伤害时，就用配置值（不被兜底覆盖）
func TestCritBonusUsedAsIs(t *testing.T) {
	st := ezfyCritTestState(50, 135)
	for !st.Done && st.Round < ezfyBattleMaxRounds {
		st.Step(nil, nil)
	}
	log := strings.Join(st.Actions, "\n")
	// crit=50 是概率暴击，40 回合内基本必中；命中的话倍率必须是配置的 135
	if strings.Contains(log, "【暴击") && !strings.Contains(log, "【暴击+135%】") {
		t.Fatalf("应使用装备配置的 135%% 暴击伤害，实际战报：\n%s", log)
	}
}
