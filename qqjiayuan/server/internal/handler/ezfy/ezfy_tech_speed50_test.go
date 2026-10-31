package ezfy

import (
	"strings"
	"testing"
)

// ★ 2026-10-31 用户需求：机械增压(10)、喷气推进(19) 效果阶梯满级 50%。
//
// 这两项原来每级 2%/3%（满级 20%/30%），现在统一 5%/级 → 满级 50%。
// 数值以 ezfyTechEffectPer 为兜底基准（库中 effect_value 为空时生效），
// 管理端在「科技管理 → 等级配置 → 效果数值」里仍可单独覆盖。
func TestSpeedTechMax50(t *testing.T) {
	for _, id := range []int{10, 19} {
		if per := ezfyTechEffectPer[id]; per != 5 {
			t.Fatalf("科技 %d 每级应为 +5%%（满级 50%%），实际 +%d%%", id, per)
		}
		if got := ezfyTechEffectValueAt(id, 10); got != 50 {
			t.Fatalf("科技 %d 满级(10 级)应为 +50%%，实际 +%d%%", id, got)
		}
		// 阶梯必须单调递增到满级，不能中途跳变
		prev := 0
		for lv := 1; lv <= 10; lv++ {
			v := ezfyTechEffectValueAt(id, lv)
			if v <= prev {
				t.Fatalf("科技 %d 的 Lv%d 效果 +%d%% 未高于上一级 +%d%%", id, lv, v, prev)
			}
			prev = v
		}
	}
}

// 战斗/行军/战报明细/出征预览四处都必须**读配置**，不能再写死每级 2%/3%。
//
// 写死的后果：管理端把效果数值改成别的值，界面文案变了、实际结算没变（或反过来）。
func TestSpeedTechNoHardcodedRate(t *testing.T) {
	for _, f := range []string{"ezfy_order.go", "ezfy_activity_target.go"} {
		src := rawFile(t, f)
		for _, bad := range []string{"[10] * 2", "[19] * 3", "[10]*2", "[19]*3"} {
			if strings.Contains(src, bad) {
				t.Fatalf("%s 里还有写死的速度科技系数 %q —— 应改为 ezfyTechEffectValueAt(id, 等级)", f, bad)
			}
		}
	}
}
