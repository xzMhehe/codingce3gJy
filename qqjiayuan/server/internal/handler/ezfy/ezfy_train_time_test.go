package ezfy

import "testing"

// TestEzfyTrainUnitSeconds 造兵时间公式：floor(P * 250/(250+M) * 10/(10+L))。
func TestEzfyTrainUnitSeconds(t *testing.T) {
	cases := []struct {
		name     string
		panel    int
		military int
		trainArt int
		want     int64
	}{
		{"无加成等于初始值", 60, 0, 0, 60},
		{"军事100", 60, 100, 0, 42},
		{"军事200", 60, 200, 0, 33},
		{"军事30向下取整", 100, 30, 0, 89},
		{"面板195军事30", 195, 30, 0, 174},
		{"动员训练10级", 60, 0, 10, 30},
		{"军事100加军训10级", 60, 100, 10, 21},
		{"图片航母示例", 6650, 384, 10, 1311},
		{"面板0", 0, 100, 10, 0},
		{"负数面板", -10, 100, 10, 0},
		{"负数军事按0处理", 60, -5, 0, 60},
		{"负数科技按0处理", 60, 0, -5, 60},
		{"军事极高时取整为0", 1, 1000, 10, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ezfyTrainUnitSeconds(c.panel, c.military, c.trainArt); got != c.want {
				t.Fatalf("ezfyTrainUnitSeconds(%d, %d, %d) = %d, want %d",
					c.panel, c.military, c.trainArt, got, c.want)
			}
		})
	}
}

// TestEzfyTrainArtTechEffect 动员训练(5) 只加造兵速度，每级展示值仍保持原来的 2%。
func TestEzfyTrainArtTechEffect(t *testing.T) {
	if got := ezfyTechEffectPrefix[5]; got != "造兵速度" {
		t.Fatalf("动员训练效果前缀 = %q, 期望 造兵速度", got)
	}
	if got := ezfyTechEffectPer[5]; got != 2 {
		t.Fatalf("动员训练每级加成 = %d, 期望 2", got)
	}
}
