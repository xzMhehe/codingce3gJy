package ezfy

import (
	"strings"
	"testing"

	"qqjiayuan/server/internal/model"
)

// ============ 2026-10-21 采集新公式 F = M × N × R × A ÷ 2 ============
//
// 线上原版口径：
//
//	M = 部队总兵力（出征部队**所有兵种数量之和**；原说明以运输卡车举例，
//	    不同兵种的差异只体现在负重）
//	N = 小时数：满期结算按整点（16:59 按 16 小时算，向下取整）；
//	    [停止采集]/[一键收获]/[提前召回] 未满一期按真实毫秒**连续小时**折算
//	    （2026-10-21 用户规则「采集多少就是收获多少」）
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
//
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
		name      string
		trucks    int64
		hours     float64
		wildPct   float64
		logistics int
		divisor   float64
		want      int64
	}{
		{"没有兵力", 0, 16, 42, 324, 2, 0},
		{"0小时", 180000, 0, 42, 324, 2, 0},
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

// ★★ 2026-10-21 用户规则「不要按整点卡，采集多少就是收获多少」：
//
//	未满一期的停止/收获/召回折算按**真实毫秒连续小时**进公式 ——
//	例：1000 兵 × 37 分钟 × 3% × 100 后勤 ÷ 2 = 925（此前整点取整 hours=0 只拿保底 1 点）。
//	保底（驻守过至少 1 点）在 dispatchGatherYieldPartial / dispatchGatherYield 层生效。
func TestEzfyGatherAmountPartialContinuous(t *testing.T) {
	const minuteMs = int64(60 * 1000)
	got := ezfyGatherAmount(1000, float64(37*minuteMs)/(3600*1000), 3, 100, 2, 1)
	if got != 925 {
		t.Fatalf("连续折算错误：1000兵×37分钟 = %d, want 925", got)
	}
	// 不足 1 分钟的极短驻守：产出按比例趋近 0（保底由调用方处理）。
	if got := ezfyGatherAmount(1000, float64(30*1000)/(3600*1000), 3, 100, 2, 1); got != 12 {
		t.Fatalf("连续折算(30秒)错误：got %d, want 12", got)
	}
}

// ★ 2026-10-21 部分结算文案：<1 小时显示分钟，否则两位小数小时。
func TestEzfyGatherHoursText(t *testing.T) {
	cases := []struct {
		hoursF float64
		want   string
	}{
		{37.0 / 60, "37分钟"},
		{0.02, "1分钟"}, // 72 秒 → 约 1 分
		{1, "1小时"},
		{16, "16小时"},
		{23.75, "23.75小时"},
	}
	for _, c := range cases {
		if got := ezfyGatherHoursText(c.hoursF); got != c.want {
			t.Fatalf("ezfyGatherHoursText(%g) = %q, want %q", c.hoursF, got, c.want)
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

func TestEzfyGatherTroopCount(t *testing.T) {
	// ★ 2026-10-21 口径修正：M = **所有兵种数量之和**（不再限定卡车兵种）。
	raw := `[{"troopId":1,"count":999},{"troopId":3,"count":1200},{"troopId":3,"count":300}]`
	if got := ezfyGatherTroopCount(raw); got != 2499 {
		t.Fatalf("部队总兵力统计错误：got %d, want 2499", got)
	}
	// 老格式兼容 `兵种:数量,兵种:数量`
	if got := ezfyGatherTroopCount("3:50,1:999"); got != 1049 {
		t.Fatalf("老格式部队总兵力统计错误：got %d, want 1049", got)
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

// ★★ 2026-10-31 用户规则：采集公式的后勤项（A）改吃**最终后勤**（基础 + 装备 + 套装），
// 为后续出「后勤装备」做准备；不能再直接用军官基础后勤。
func TestEzfyGatherLogisticsCountsEquipment(t *testing.T) {
	o := &model.EzfyOfficer{
		Logistics: 100,
		Equipment: `[{"slot":"ring","logistics":10},{"slot":"necklace","logistics":5}]`,
	}
	h := &EzfyHandler{}
	if _, log, _ := h.officerEquipBonus(o); log != 15 {
		t.Fatalf("装备后勤汇总 = %d, want 15", log)
	}
	body := ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) gatherLogistics(")
	if !strings.Contains(body, "officerEffective(") {
		t.Fatalf("gatherLogistics 没有取最终后勤（基础+装备+套装）：\n%s", body)
	}
	if !strings.Contains(body, "env.gatherLogs") {
		t.Fatalf("gatherLogistics 没有复用批次缓存 env.gatherLogs（批量结算会重复解析装备 JSON）：\n%s", body)
	}
	yield := ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) gatherYieldAt(")
	if !strings.Contains(yield, "h.gatherLogistics(env, officer)") {
		t.Fatalf("gatherYieldAt 没有使用最终后勤：\n%s", yield)
	}
	if strings.Contains(yield, "logistics = officer.Logistics") {
		t.Fatalf("gatherYieldAt 仍在用军官基础后勤：\n%s", yield)
	}
}
