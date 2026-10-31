package seed

import (
	"strconv"
	"testing"
)

// ★ 2026-10-31 机械增压(10)/喷气推进(19) 满级 50%：
// 主表 Effect 与等级表阶梯文案都必须与代码里的每级 5% 对齐，
// 否则管理端/玩家看到的是旧数值（配置文案与结算不一致）。
func TestSpeedTechSeedText(t *testing.T) {
	wantMain := map[int]string{10: "部队速度+5%", 19: "空军速度+5%"}
	for id, want := range wantMain {
		var found bool
		for _, tech := range ezfyEzfyCfgTech {
			if tech.ID != id {
				continue
			}
			found = true
			if tech.Effect != want {
				t.Fatalf("科技 %d 主表 Effect = %q，期望 %q", id, tech.Effect, want)
			}
		}
		if !found {
			t.Fatalf("主表里找不到科技 id=%d", id)
		}
	}

	wantPrefix := map[int]string{10: "部队速度", 19: "空军速度"}
	for id, prefix := range wantPrefix {
		levels := 0
		for _, l := range ezfyEzfyCfgTechLevel {
			if l.TechId != id {
				continue
			}
			levels++
			want := prefix + "+" + strconv.Itoa(l.Level*5) + "%"
			if l.Effect != want {
				t.Fatalf("科技 %d Lv%d 的 Effect = %q，期望 %q（每级 +5%%）", id, l.Level, l.Effect, want)
			}
		}
		if levels != 10 {
			t.Fatalf("科技 %d 等级配置应有 10 级，实际 %d 级", id, levels)
		}
	}
}
