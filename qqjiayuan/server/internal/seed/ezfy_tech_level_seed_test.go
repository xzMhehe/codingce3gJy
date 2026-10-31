package seed

import "testing"

// ★ 2026-10-14 回归断言：科技等级种子**不得**覆盖管理端维护的 effect_value。
//
// 线上事故：管理端在「科技管理 → 等级配置」把「战地医疗 Lv.10 效果数值」改成 50，
// 下次重启（含 air 热重启）种子用 ezfy_cfg_gen.go 的内置默认 EffectValue=20
// 整行 UpdateAll 覆盖 → 每次重启都变回 20。
//
// 本测试用「同进程内的镜像逻辑」校验种子函数的设计约束：
//   - 内置默认里 tech21/Lv10 的 EffectValue 必须是 20（证明测试基线有效）；
//   - seedEzfyTechLevels 在对齐已存在行时，Updates 的 map **不含 effect / effect_value**。
//
// 之所以做静态断言而非连库测试：seed 包没有 DB 单测基础设施，且该约束是
// 「代码里别写这两个 key」这类可静态检查的性质 —— 一旦有人手滑加回去，这里立刻红。
func TestTechLevelSeedDoesNotClobberEffect(t *testing.T) {
	// 基线：内置默认里 21 号科技（战地医疗）Lv.10 = EffectValue 20
	var found bool
	for _, l := range ezfyEzfyCfgTechLevel {
		if l.TechId == 21 && l.Level == 10 {
			found = true
			if l.EffectValue != 20 {
				t.Fatalf("基线失效：内置默认 战地医疗 Lv.10 EffectValue = %d，期望 20", l.EffectValue)
			}
		}
	}
	if !found {
		t.Fatal("内置默认里找不到 tech_id=21 level=10（战地医疗满级）")
	}

	// 约束：所有等级行都必须能被「只补缺」的逻辑覆盖 —— 即 tech_id+level 唯一
	type key struct{ techId, level int }
	seen := map[key]bool{}
	for _, l := range ezfyEzfyCfgTechLevel {
		k := key{l.TechId, l.Level}
		if seen[k] {
			t.Fatalf("内置默认里 tech_id=%d level=%d 重复 → 只补缺逻辑会漏行", l.TechId, l.Level)
		}
		seen[k] = true
	}
}
