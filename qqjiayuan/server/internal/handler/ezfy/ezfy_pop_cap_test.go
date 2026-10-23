package ezfy

import (
	"strings"
	"testing"

	"qqjiayuan/server/internal/model"
)

// ============ 人口上限口径守卫（2026-10-23 最终版）============
//
// 用户最终口径：
//  1. 人口上限只由民居容量（pop_max）决定；
//  2. 人口未满 → 召集加总人口（空闲同步增加），最多加到 pop_max；
//  3. 人口已满 → 召集/移民法案仍可继续：总人口不增加，本次召集量转为空闲人口；
//     先扣待恢复人口，剩余进入 pop_extra（允许空闲人口暂时超过总人口）；
//  4. 人口达到/超过民居上限时，召集冷却默认 300 秒（普通情况仍是 3 秒），两档均可配置；
//  5. 总人口永远不能超过民居上限：超限数据在下一次结算时**立即夹平**；
//     额外空闲人口（pop_extra）超过正常空闲的部分，按 pop_extra_decay_seconds
//     （默认 300 秒 = 5 分钟）线性回落。

func TestEzfyPopGatherPlan(t *testing.T) {
	cases := []struct {
		name    string
		pop     int64
		popMax  int64
		want    int64
		recover int64
		gain    int64
		idle    bool
	}{
		{"没有民居 → 无法召集", 0, 0, 100, 50, 0, false},
		{"人口未满 → 加人口", 100, 1000, 100, 0, 100, false},
		{"人口贴近上限 → 截断到上限", 950, 1000, 100, 0, 50, false},
		{"人口差 1 人 → 只加 1 人", 999, 1000, 100, 0, 1, false},
		{"人口已满 + 有待恢复 → 固定获得召集量", 1000, 1000, 100, 60, 100, true},
		{"人口已满 + 待恢复不足 → 固定获得召集量", 1000, 1000, 100, 30, 100, true},
		{"人口已满 + 无待恢复 → 仍可召集", 1000, 1000, 100, 0, 100, true},
		{"人口超限（老数据）+ 待恢复 → 固定获得召集量", 1200, 1000, 100, 30, 100, true},
		{"待恢复为负 → 不影响召集", 1000, 1000, 100, -5, 100, true},
		{"want <= 0 → 不召集", 100, 1000, 0, 0, 0, false},
	}
	for _, c := range cases {
		gain, idle := ezfyPopGatherPlan(c.pop, c.popMax, c.want, c.recover)
		if gain != c.gain || idle != c.idle {
			t.Errorf("%s: ezfyPopGatherPlan(pop=%d, max=%d, want=%d, recover=%d) = (%d,%v), want (%d,%v)",
				c.name, c.pop, c.popMax, c.want, c.recover, gain, idle, c.gain, c.idle)
		}
	}
}

// TestEzfyPopFreeIncludesExtra 空闲人口 = 人口 − 建筑占用 − 待恢复 + 额外空闲（允许超过总人口）。
func TestEzfyPopFreeIncludesExtra(t *testing.T) {
	for _, c := range []struct {
		name        string
		pop         int64
		buildingPop int64
		troopPop    int64
		recover     int64
		extra       int64
		want        int64
	}{
		{"普通口径", 1000, 100, 0, 200, 0, 700},
		{"额外空闲让空闲超过总人口", 1000, 100, 0, 0, 500, 1400},
		{"扣到负数夹 0", 100, 200, 0, 0, 0, 0},
		{"负额外空闲按原口径", 1000, 100, 0, 100, -50, 750},
	} {
		if got := ezfyPopFree(c.pop, c.buildingPop, c.troopPop, c.recover, c.extra); got != c.want {
			t.Errorf("%s: ezfyPopFree = %d, want %d", c.name, got, c.want)
		}
	}
}

// TestSettlePopExtraDecay 额外空闲人口按截止时间线性回落：首次起算、半程减半、到点归零。
func TestSettlePopExtraDecay(t *testing.T) {
	const durationMs = int64(5 * 60 * 1000)
	const start = int64(1700000000000)
	city := &model.EzfyCity{}
	ezfyAddPopExtraAt(city, 1000, start, durationMs)
	if city.PopExtra != 1000 || city.PopExtraAt != start || city.PopExtraEndAt != start+durationMs {
		t.Fatalf("增加额外空闲人口后状态不对: %+v", city)
	}
	// 首次结算（已经带了 at/end，不应重置）——半程后应剩 500
	ezfySettlePopExtraAt(city, start+durationMs/2, durationMs)
	if city.PopExtra != 500 {
		t.Fatalf("半程应剩 500，got %d", city.PopExtra)
	}
	// 到点必须归零
	ezfySettlePopExtraAt(city, start+durationMs, durationMs)
	if city.PopExtra != 0 || city.PopExtraEndAt != 0 {
		t.Fatalf("到点应清零，got %+v", city)
	}
}

// TestEzfyClampPopToHouseCap 超限人口必须立即夹到民居上限，不允许任何缓慢回落。
func TestEzfyClampPopToHouseCap(t *testing.T) {
	for _, c := range []struct {
		name   string
		pop    int64
		popMax int64
		want   int64
	}{
		{"未超限 → 原样", 45000, 45000, 45000},
		{"低于上限 → 原样", 10000, 45000, 10000},
		{"超过上限 → 立即夹平", 100000, 45000, 45000},
		{"超过 1 人 → 立即夹平", 45001, 45000, 45000},
		{"没有民居/上限未知 → 不动人口", 100000, 0, 100000},
		{"上限为负 → 不动人口", 100000, -1, 100000},
	} {
		if got := ezfyClampPopToHouseCap(c.pop, c.popMax); got != c.want {
			t.Errorf("%s: ezfyClampPopToHouseCap(%d, %d) = %d, want %d", c.name, c.pop, c.popMax, got, c.want)
		}
	}
}

// TestTrainPopRecoverStartsAtConscription 征兵占用的人口必须**从征兵当下**开始慢恢复，
// 不能等训练队列出厂（造完）才计入待恢复。
func TestTrainPopRecoverStartsAtConscription(t *testing.T) {
	train := ezfyStripCommentLines(ezfyFuncBody(t, "ezfy.go", "func (h *EzfyHandler) trainTroopLocked("))
	for _, want := range []string{
		"h.ezfySettlePopRecover(city)",
		"city.PopRecover += int64(cfg.Pop) * int64(count)",
		"city.PopRecoverEndAt = now + ezfyPopRecoverSeconds()*1000",
	} {
		if !strings.Contains(train, want) {
			t.Fatalf("trainTroopLocked 未在征兵当下转入待恢复人口（缺 %s）：\n%s", want, train)
		}
	}
	// 出厂时不能再累加一次，否则同一笔人口被算两次、恢复量翻倍。
	collect := ezfyStripCommentLines(ezfyFuncBody(t, "ezfy.go", "func (h *EzfyHandler) collectTrainQueue("))
	if strings.Contains(collect, "PopRecover +=") {
		t.Fatalf("collectTrainQueue 出厂时仍在累加待恢复人口（应在征兵当下算）：\n%s", collect)
	}
}

// TestSettlePopRecoverByConfiguredSeconds 「多少秒内空闲人口恢复满」：
//
//	默认 1200 秒；按截止时间线性恢复，到点必须归零，不能按旧的固定每小时恢复量慢慢磨。
func TestSettlePopRecoverByConfiguredSeconds(t *testing.T) {
	if got := ezfyPopRecoverSeconds(); got != 1200 {
		t.Fatalf("空闲人口恢复满默认应为 1200 秒，got %d", got)
	}
	const durationMs = int64(1200 * 1000)
	const start = int64(1700000000000)
	city := &model.EzfyCity{PopRecover: 1000}
	// 首次结算：起算一整轮，不瞬间恢复
	ezfySettlePopRecoverAt(city, start, durationMs)
	if city.PopRecover != 1000 || city.PopRecoverAt != start || city.PopRecoverEndAt != start+durationMs {
		t.Fatalf("首次结算应只起算恢复窗口，got %+v", city)
	}
	// 半程：恢复一半
	ezfySettlePopRecoverAt(city, start+durationMs/2, durationMs)
	if city.PopRecover != 500 {
		t.Fatalf("半程应剩 500，got %d", city.PopRecover)
	}
	// 到点：恢复满
	ezfySettlePopRecoverAt(city, start+durationMs, durationMs)
	if city.PopRecover != 0 || city.PopRecoverEndAt != 0 {
		t.Fatalf("到点应恢复满，got %+v", city)
	}
}

// TestEzfyHousePopCapOf 人口上限必须按「当前建筑列表里的民居等级」实时求和。
func TestEzfyHousePopCapOf(t *testing.T) {
	saved := ezfyCfg.buildingLvls
	defer func() { ezfyCfg.buildingLvls = saved }()
	ezfyCfg.buildingLvls = map[int]map[int]model.EzfyCfgBuildingLevel{
		2: {
			8:  {Capacity: 3600},
			10: {Capacity: 5500},
		},
		3: {10: {Capacity: 9999}},
	}
	buildings := []model.EzfyCityBuilding{
		{BuildingId: 2, Level: 8},
		{BuildingId: 2, Level: 10},
		{BuildingId: 3, Level: 10}, // 非民居，不计入
		{BuildingId: 2, Level: 99}, // 配置缺失，不计入
	}
	if got := ezfyHousePopCapOf(buildings); got != 9100 {
		t.Fatalf("ezfyHousePopCapOf = %d，期望 3600+5500=9100", got)
	}
}

// TestCalcResourceHouseCap 懒结算必须按实时民居 pop_max 封顶；超限立即夹平。
func TestCalcResourceHouseCap(t *testing.T) {
	body := ezfyStripCommentLines(ezfyFuncBody(t, "ezfy.go", "func (h *EzfyHandler) calcResourceD("))
	for _, want := range []string{
		"if city.PopMax > 0 && city.Pop < city.PopMax {",
		"city.Pop = ezfyClampPopToHouseCap(city.Pop, city.PopMax)",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("calcResourceD 缺少「民居上限 + 立即夹平」逻辑（缺 %s）：\n%s", want, body)
		}
	}
	// 防回退：旧的开关/硬上限/缓慢回落逻辑必须删除
	for _, banned := range []string{
		"ezfyHousePopLimitOn", "ezfyConvenePopMaxCfg", "ezfyHouseSoftCap", "ezfyEffPopCap",
		"ezfySettlePopOverCap", "ezfyPopOverCapDecayPct", "pop_over_cap_at",
	} {
		if strings.Contains(body, banned) {
			t.Fatalf("calcResourceD 仍引用已删除的 %s：\n%s", banned, body)
		}
	}
}

// TestCalcResourceWriteGuardedByUpdatedAt 合并写必须带 updated_at 乐观锁，
// 防止请求开始时读到的旧快照（pop_extra=0）覆盖刚召集出来的临时空闲人口。
func TestCalcResourceWriteGuardedByUpdatedAt(t *testing.T) {
	body := ezfyStripCommentLines(ezfyFuncBody(t, "ezfy.go", "func (h *EzfyHandler) calcResourceD("))
	for _, want := range []string{
		"origUpdatedAt := city.UpdatedAt",
		`q.Where("updated_at = ?", origUpdatedAt)`,
		`"pop_extra": city.PopExtra`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("calcResourceD 合并写缺少并发保护（缺 %s）：\n%s", want, body)
		}
	}
}

// TestTrainPopAvailableIncludesExtra 训练人口校验必须把临时空闲人口算进可用人口。
func TestTrainPopAvailableIncludesExtra(t *testing.T) {
	train := ezfyStripCommentLines(ezfyFuncBody(t, "ezfy.go", "func (h *EzfyHandler) trainTroopLocked("))
	for _, want := range []string{
		"city.Pop - popUsed - city.PopRecover + city.PopExtra",
		"临时空闲%d",
	} {
		if !strings.Contains(train, want) {
			t.Fatalf("trainTroopLocked 人口校验未计入临时空闲人口（缺 %s）：\n%s", want, train)
		}
	}
}

// TestPopExtraDecayCfg 临时空闲人口回落窗口必须从二战系统配置读取，默认 300 秒。
func TestPopExtraDecayCfg(t *testing.T) {
	old := ezfyCfg
	defer func() { ezfyCfg = old }()
	ezfyCfg.limit = model.EzfyCfgLimit{}
	if got := ezfyPopExtraDecayMs(); got != 300000 {
		t.Fatalf("临时空闲回落默认值 = %d ms，期望 300000（5 分钟）", got)
	}
	ezfyCfg.limit.PopExtraDecaySeconds = 42
	if got := ezfyPopExtraDecayMs(); got != 42000 {
		t.Fatalf("临时空闲回落配置 42 秒未生效，got %d ms", got)
	}
	ezfyCfg.limit.PopExtraDecaySeconds = 0
	if got := ezfyPopExtraDecayMs(); got != 300000 {
		t.Fatalf("临时空闲回落配置 0 应回落默认 300 秒，got %d ms", got)
	}
	// 实际新增临时空闲也要用配置窗口，而不是写死的 5 分钟。
	ezfyCfg.limit.PopExtraDecaySeconds = 600
	city := &model.EzfyCity{}
	ezfyAddPopExtra(city, 100)
	if city.PopExtra != 100 || city.PopExtraEndAt-city.PopExtraAt != 600000 {
		t.Fatalf("临时空闲新增未使用配置的 600 秒窗口: %+v", city)
	}

	// 后台保存/读取、seed 补列、管理端 UI 必须都接上这个配置。
	adminGo := ezfyStripCommentLines(rawFile(t, "ezfy_admin_round9.go"))
	for _, want := range []string{
		"PopExtraDecaySeconds",
		`"pop_extra_decay_seconds": lim.PopExtraDecaySeconds`,
		"临时空闲回落时间需要在",
	} {
		if !strings.Contains(adminGo, want) {
			t.Fatalf("管理端后端未接入临时空闲回落配置（缺 %s）", want)
		}
	}
	seedGo := ezfyStripCommentLines(rawFile(t, "../../seed/seed.go"))
	for _, want := range []string{
		`addLimitCol("pop_extra_decay_seconds", 300)`,
		`ADD COLUMN pop_extra_decay_seconds int DEFAULT 300`,
	} {
		if !strings.Contains(seedGo, want) {
			t.Fatalf("seed 未补临时空闲回落配置列（缺 %s）", want)
		}
	}
	adminVue := ezfyStripCommentLines(readFrontFile(t, "../../../../admin-web/src/components/admin/AdminEzfyBuildLimit.vue"))
	for _, want := range []string{"pop_extra_decay_seconds", "临时空闲回落时间", "默认 300 秒（5 分钟）"} {
		if !strings.Contains(adminVue, want) {
			t.Fatalf("管理端 UI 未提供临时空闲回落配置（缺 %s）", want)
		}
	}
}

func TestConveneCooldownCfg(t *testing.T) {
	old := ezfyCfg
	defer func() { ezfyCfg = old }()
	ezfyCfg.limit = model.EzfyCfgLimit{}
	if got := ezfyConveneCooldownMs(false); got != 3000 {
		t.Fatalf("普通召集间隔默认值 = %d ms，期望 3000", got)
	}
	if got := ezfyConveneCooldownMs(true); got != 300000 {
		t.Fatalf("人口满召集间隔默认值 = %d ms，期望 300000", got)
	}
	ezfyCfg.limit.ConveneCooldownSeconds = 5
	ezfyCfg.limit.ConveneCapCooldownSeconds = 7
	if got := ezfyConveneCooldownMs(false); got != 5000 {
		t.Fatalf("普通召集间隔配置 5 秒未生效，got %d ms", got)
	}
	if got := ezfyConveneCooldownMs(true); got != 7000 {
		t.Fatalf("人口满召集间隔配置 7 秒未生效，got %d ms", got)
	}
	for ms, want := range map[int64]string{300000: "5 分钟", 5000: "5 秒", 90000: "1 分 30 秒"} {
		if got := ezfyConveneCooldownTip(ms); got != want {
			t.Fatalf("冷却提示 %d ms = %q，期望 %q", ms, got, want)
		}
	}
}

// TestConveneByHouseCap 召集以实时民居上限为准：未满加人口；已满总人口不动，加空闲人口。
func TestConveneByHouseCap(t *testing.T) {
	body := ezfyStripCommentLines(ezfyFuncBody(t, "ezfy.go", "func (h *EzfyHandler) Convene("))
	for _, want := range []string{
		"city.PopMax = h.housePopCap(city.ID)",
		"city.Pop = ezfyClampPopToHouseCap(city.Pop, city.PopMax)",
		"ezfyPopGatherPlan(city.Pop, city.PopMax, popGain, city.PopRecover)",
		"recoverUse := gain",
		"city.PopRecover -= recoverUse",
		"ezfyAddPopExtra(city, extraGain)",
		"city.Pop += gain",
		"foodCost = foodCost * gain / baseGain",
		"ezfyConveneCooldownMs(city.Pop >= city.PopMax)",
		"ezfyConveneCooldownTip(cdMs)",
		`"pop_max":            city.PopMax`,
		`"pop_extra":`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("Convene 缺少「实时民居上限 + 两阶段召集」逻辑（缺 %s）：\n%s", want, body)
		}
	}
	for _, banned := range []string{"ezfyConvenePopMaxCfg", "ezfyHouseSoftCap", "ezfyHousePopLimitOn"} {
		if strings.Contains(body, banned) {
			t.Fatalf("Convene 仍引用已删除的 %s：\n%s", banned, body)
		}
	}
}

// TestPopGatesUseHouseCap 召集 / 移民法案共用同一套「实时民居上限两阶段」口径。
func TestPopGatesUseHouseCap(t *testing.T) {
	src := ezfyStripCommentLines(rawFile(t, "ezfy.go"))
	for _, banned := range []string{
		"ezfyConvenePopMaxCfg", "ezfyHouseSoftCap", "ezfyHousePopLimitOn", "ezfyConveneFlexOn",
		"ezfySettlePopOverCap", "ezfyPopOverCapDecayPct", "PopOverCapAt",
	} {
		if strings.Contains(src, banned) {
			t.Fatalf("ezfy.go 仍引用已删除的 %s", banned)
		}
	}
	// Convene + 移民法案 ItemType31 两处共用
	if n := strings.Count(src, "ezfyPopGatherPlan("); n < 2 {
		t.Fatalf("ezfyPopGatherPlan() 只有 %d 处，至少应有 2 处（人口召集 / 移民法案 ItemType31）", n)
	}
	// 两处都必须用当前建筑列表实时算上限，不能信任可能滞后的 ezfy_city.pop_max。
	if n := strings.Count(src, "city.PopMax = h.housePopCap(city.ID)"); n < 2 {
		t.Fatalf("实时民居上限只有 %d 处，至少应有 2 处（人口召集 / 移民法案 ItemType31）", n)
	}
}

// TestFrontendPopCapSameFormula 前端展示/按钮/冷却必须和后端同口径。
func TestFrontendPopCapSameFormula(t *testing.T) {
	vue := strings.ReplaceAll(readFrontFile(t, "../../../../web/src/views/ezfy/Ezfy.vue"), "\r\n", "\n")
	vue = ezfyStripCommentLines(vue)
	// popCap 只取民居上限
	if !strings.Contains(vue, "return (this.city && this.city.pop_max) || 0") {
		t.Fatal("前端 popCap 未只取民居上限")
	}
	// 人口满后召集仍可用：按钮只受「民居容量为 0」禁用（冷却由 doConvene/后端卡控）
	if !strings.Contains(vue, "return pm <= 0") {
		t.Fatal("前端 conveneBlocked 仍按人口已满禁用召集")
	}
	// 空闲人口必须计入额外空闲池
	if !strings.Contains(vue, "(this.popExtra || 0)") {
		t.Fatal("前端 freePop 未计入额外空闲人口 popExtra")
	}
	// 达到/超过民居上限 → 读取配置的召集间隔（默认 300 秒），不能写死 5 分钟
	if !strings.Contains(vue, "this.conveneCapCooldownSec") || !strings.Contains(vue, "this.conveneCooldownSec") {
		t.Fatal("前端 conveneCooldownMs 未读取召集间隔配置")
	}
	if strings.Contains(vue, "5 * 60 * 1000") {
		t.Fatal("前端 conveneCooldownMs 仍写死 5 分钟")
	}
	for _, banned := range []string{
		"convenePopMax", "housePopLimitOn", "conveneFlexibleOn", "houseSoftCapActive",
		"popOverHouseCap",
	} {
		if strings.Contains(vue, banned) {
			t.Fatalf("前端仍引用已删除的 %s", banned)
		}
	}

	cityVue := ezfyStripCommentLines(readFrontFile(t, "../../../../web/src/views/ezfy/modules/EzfyCity.vue"))
	// 召集页必须显示空闲人口（用户反馈「以前有现在没了」）
	if !strings.Contains(cityVue, "空闲 {{ ezfy.freePop }}") {
		t.Fatal("召集页未显示空闲人口")
	}
	// 人口已满时提示要短：只说明召集转为临时空闲，不再展开「总人口不变 / 超出部分回落」
	if !strings.Contains(cityVue, "人口已满，召集只加临时空闲") {
		t.Fatal("召集页未提示「人口已满，召集只加临时空闲」")
	}
	if strings.Contains(cityVue, "超出部分会回落") {
		t.Fatal("召集页仍在展示啰嗦的「超出部分会回落」说明")
	}
	// 超限总人口已改为立即夹平，前端不能再展示旧的「超限人口缓慢回落」提示
	if strings.Contains(cityVue, "popOverHouseCap") || strings.Contains(cityVue, "缓慢回落") {
		t.Fatal("前端仍展示已删除的「超限人口缓慢回落」提示")
	}
}
