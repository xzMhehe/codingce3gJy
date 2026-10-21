package ezfy

import (
	"testing"

	"qqjiayuan/server/internal/model"
)

// ============ 2026-10-21 采集新公式 F = M × N × R × A ÷ 2 ============
//
// 线上原版口径：
//
//	M = 卡车数量（默认兵种 3）
//	N = 整点小时（16:59 按 16 小时算，向下取整）
//	R = 出发城市同资源野地总等级 × 每级加成%（默认 3%/级）
//	A = 带队军官后勤
//
// 截图示例：180000 × 16 × 42% × 324 ÷ 2 = 195955200。

func TestEzfyGatherAmountScreenshotExample(t *testing.T) {
	got := ezfyGatherAmount(180000, 16, 42, 324, 2, 1)
	if got != 195955200 {
		t.Fatalf("采集公式示例计算错误：got %d, want 195955200", got)
	}
}

// ★ 2026-10-21 用户规则「海野需要比陆地野地高 3 倍」：
//	海野（海底森林/岛屿/渔场/矿礁）产出 = 公式结果 × 海野倍率（默认 3，管理端可配）。
func TestEzfyGatherAmountSeaBonusMult(t *testing.T) {
	cases := []struct {
		name    string
		seaMult float64
		want    int64
	}{
		{"陆地(倍率1)", 1, 1500},
		{"海野默认3倍", 3, 4500},
		{"海野温和1.5倍", 1.5, 2250},
		{"倍率0视为无加成", 0, 1500},
		{"倍率负数视为无加成", -3, 1500},
	}
	for _, c := range cases {
		if got := ezfyGatherAmount(1000, 1, 3, 100, 2, c.seaMult); got != c.want {
			t.Fatalf("%s：ezfyGatherAmount(seaMult=%g) = %d, want %d", c.name, c.seaMult, got, c.want)
		}
	}
	// 海野倍率过大时封顶 int64 最大值，不允许溢出成负数。
	const huge = int64(9223372036854775807) // math.MaxInt64
	if got := ezfyGatherAmount(huge, 1, 100, 100, 0.01, 10); got != huge {
		t.Fatalf("海野倍率封顶失败：got %d, want %d", got, huge)
	}
}

func TestEzfyGatherAmountGuards(t *testing.T) {
	cases := []struct {
		name          string
		trucks, hours int64
		wildPct       float64
		logistics     int
		divisor       float64
		want          int64
	}{
		{"没有卡车", 0, 16, 42, 324, 2, 0},
		{"不足整点", 180000, 0, 42, 324, 2, 0},
		{"没有野地加成", 180000, 16, 0, 324, 2, 0},
		{"没有后勤", 180000, 16, 42, 0, 2, 0},
		{"除数为0", 180000, 16, 42, 324, 0, 0},
		{"默认每级3%", 1000, 1, 3, 100, 2, 1500},
	}
	for _, c := range cases {
		if got := ezfyGatherAmount(c.trucks, c.hours, c.wildPct, c.logistics, c.divisor, 1); got != c.want {
			t.Fatalf("%s: got %d, want %d", c.name, got, c.want)
		}
	}
}

func TestEzfyGatherHoursFloor(t *testing.T) {
	const hour = int64(3600 * 1000)
	cases := []struct {
		ms   int64
		want int64
	}{
		{0, 0},
		{hour - 1, 0},
		{hour, 1},
		{16*hour + 59*60*1000, 16}, // 16:59 → 16
		{17*hour - 1, 16},
		{17 * hour, 17},
	}
	for _, c := range cases {
		if got := ezfyGatherHours(c.ms); got != c.want {
			t.Fatalf("ezfyGatherHours(%d) = %d, want %d", c.ms, got, c.want)
		}
	}
}

func TestEzfyGatherTruckCount(t *testing.T) {
	raw := `[{"troopId":1,"count":999},{"troopId":3,"count":1200},{"troopId":3,"count":300}]`
	if got := ezfyGatherTruckCount(raw); got != 1500 {
		t.Fatalf("卡车数量统计错误：got %d, want 1500", got)
	}
	// 老格式兼容 `兵种:数量,兵种:数量`
	if got := ezfyGatherTruckCount("3:50,1:999"); got != 50 {
		t.Fatalf("老格式卡车数量统计错误：got %d, want 50", got)
	}
}

func TestEzfyWildResLevelsAggregate(t *testing.T) {
	// 同资源的两块野地等级必须相加（R 看的是全城同资源总等级，不是单块野地）。
	x, y := 12, 34
	res := ezfyGatherResName(ezfyTerrainEx(x, y))
	wls := []model.EzfyWildland{
		{X: x, Y: y, Level: 10, CityId: 1},
		{X: x, Y: y, Level: 4, CityId: 1},
		{X: x, Y: y, Level: 0, CityId: 1}, // 0 级不计
	}
	if got := ezfyWildResLevels(wls)[res]; got != 14 {
		t.Fatalf("同资源野地总等级 = %d, want 14", got)
	}
}
