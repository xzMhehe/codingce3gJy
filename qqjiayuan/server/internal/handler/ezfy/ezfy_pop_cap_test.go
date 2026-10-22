package ezfy

import (
	"strings"
	"testing"

	"qqjiayuan/server/internal/model"
)

// ============ 人口上限口径守卫（2026-10-22 最终版）============
//
// 用户最终口径：
//  1. 人口上限只由民居容量（pop_max）决定；
//  2. 人口未满 → 召集加总人口（空闲同步增加），最多加到 pop_max；
//  3. 人口已满 → 召集/移民法案只扣待恢复人口，恢复空闲人口；
//  4. 人口达到/超过民居上限时，召集冷却为 5 分钟（普通情况仍是 3 秒）；
//  5. 空闲人口超过「人口上限 − 占用」的部分自动回落（老数据走缓慢回落）。

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
		{"人口已满 + 有待恢复 → 只恢复空闲", 1000, 1000, 100, 60, 60, true},
		{"人口已满 + 待恢复不足 → 恢复完", 1000, 1000, 100, 30, 30, true},
		{"人口已满 + 无待恢复 → 拒绝", 1000, 1000, 100, 0, 0, true},
		{"人口超限（老数据）+ 待恢复 → 恢复空闲", 1200, 1000, 100, 30, 30, true},
		{"待恢复为负 → 按 0 处理", 1000, 1000, 100, -5, 0, true},
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

// TestEzfySettlePopOverCap 超限人口必须「慢慢回落」，且时间余量不能丢。
//
// ★ 2026-10-21 用户要求「100% 也要 5 分钟内回落」→ 速度基准从「每小时」改成「每 5 分钟」。
func TestEzfySettlePopOverCap(t *testing.T) {
	const now = int64(1_700_000_000_000)
	const cap = int64(45000)
	const fiveMin = int64(300_000)
	const hour = 12 * fiveMin

	// 未超限 → 时间戳清零
	if pop, at := ezfySettlePopOverCap(45000, cap, now, now, 10); pop != 45000 || at != 0 {
		t.Fatalf("未超限应原样返回且 at=0，got pop=%d at=%d", pop, at)
	}
	// 首次进入超限（at=0）：只记时间，本次不回落
	if pop, at := ezfySettlePopOverCap(100000, cap, 0, now, 10); pop != 100000 || at != now {
		t.Fatalf("首次超限只应记时间，got pop=%d at=%d", pop, at)
	}
	// 5 分钟：超出 55000 × 10%/5 分钟 = 5500，人口 100000 → 94500
	pop, at := ezfySettlePopOverCap(100000, cap, now, now+fiveMin, 10)
	if pop != 94500 || at != now+fiveMin {
		t.Fatalf("10%%/5 分钟应回落 5500 人，got pop=%d at=%d", pop, at)
	}
	// 速度可配置：50%/5 分钟 → 超出 55000 × 50% = 27500，人口 100000 → 72500
	if pop, at := ezfySettlePopOverCap(100000, cap, now, now+fiveMin, 50); pop != 72500 || at != now+fiveMin {
		t.Fatalf("50%%/5 分钟应回落 27500 人，got pop=%d at=%d", pop, at)
	}
	// 100%/5 分钟：正好 5 分钟回落到民居上限，时间戳清零
	if pop, at := ezfySettlePopOverCap(100000, cap, now, now+fiveMin, 100); pop != cap || at != 0 {
		t.Fatalf("100%%/5 分钟应 5 分钟内回落到 %d 且 at=0，got pop=%d at=%d", cap, pop, at)
	}
	// 100%/5 分钟：2 分钟按比例回落 40%（55000 × 40% = 22000）
	if pop, at := ezfySettlePopOverCap(100000, cap, now, now+2*60_000, 100); pop != 78000 || at != now+2*60_000 {
		t.Fatalf("100%%/5 分钟 2 分钟应回落 22000 人，got pop=%d at=%d", pop, at)
	}
	// 非法速度 → 回落默认 100%/5 分钟
	if pop, at := ezfySettlePopOverCap(100000, cap, now, now+fiveMin, 0); pop != cap || at != 0 {
		t.Fatalf("非法速度应回落默认 100%%，got pop=%d at=%d", pop, at)
	}
	// 1 小时：足够回落到民居上限，时间戳清零
	if pop, at := ezfySettlePopOverCap(100000, cap, now, now+hour, 10); pop != cap || at != 0 {
		t.Fatalf("1 小时应回落到 %d 且 at=0，got pop=%d at=%d", cap, pop, at)
	}
	// 超出量很小、时间很短：不足 1 人时保留时间余量（at 不推进）
	if pop, at := ezfySettlePopOverCap(45010, cap, now, now+60_000, 10); pop != 45010 || at != now {
		t.Fatalf("不足 1 人时应保留时间余量，got pop=%d at=%d", pop, at)
	}
	// 超出量很小、时间够长：刚好回落 1 人
	pop, at = ezfySettlePopOverCap(45010, cap, now, now+fiveMin, 10)
	if pop != 45009 || at != now+fiveMin {
		t.Fatalf("超出 10 人 5 分钟应回落 1 人，got pop=%d at=%d", pop, at)
	}
	// 时间戳异常（未来）→ 重置为 now，不回落
	if pop, at := ezfySettlePopOverCap(100000, cap, now+hour, now, 10); pop != 100000 || at != now {
		t.Fatalf("时间戳在未来应重置，got pop=%d at=%d", pop, at)
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

// TestPopOverCapDecayConfig 回落速度必须来自二战系统配置，而不是写死常量。
func TestPopOverCapDecayConfig(t *testing.T) {
	body := ezfyStripCommentLines(ezfyFuncBody(t, "ezfy.go", "func ezfyPopOverCapDecayPct() int {"))
	for _, want := range []string{
		"ezfyCfg.limit.PopOverCapDecayPct",
		"ezfyPopOverCapDecayPctDef",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("超限人口回落速度未读取系统配置（缺 %s）：\n%s", want, body)
		}
	}
}

// TestCalcResourceHouseCap 懒结算必须以民居 pop_max 封顶；老数据超限走缓慢回落。
func TestCalcResourceHouseCap(t *testing.T) {
	body := ezfyStripCommentLines(ezfyFuncBody(t, "ezfy.go", "func (h *EzfyHandler) calcResourceD("))
	for _, want := range []string{
		"if city.PopMax > 0 && city.Pop < city.PopMax {",
		"if city.Pop > city.PopMax {",
		"ezfySettlePopOverCap(city.Pop, city.PopMax, city.PopOverCapAt, now, ezfyPopOverCapDecayPct())",
		`"pop_over_cap_at": city.PopOverCapAt`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("calcResourceD 缺少「民居上限 + 超限回落」逻辑（缺 %s）：\n%s", want, body)
		}
	}
	// 防回退：旧的开关/硬上限函数必须删除
	for _, banned := range []string{"ezfyHousePopLimitOn", "ezfyConvenePopMaxCfg", "ezfyHouseSoftCap", "ezfyEffPopCap"} {
		if strings.Contains(body, banned) {
			t.Fatalf("calcResourceD 仍引用已删除的 %s：\n%s", banned, body)
		}
	}
}

// TestConveneByHouseCap 召集以民居上限为准：未满加人口，已满只恢复空闲人口。
func TestConveneByHouseCap(t *testing.T) {
	body := ezfyStripCommentLines(ezfyFuncBody(t, "ezfy.go", "func (h *EzfyHandler) Convene("))
	for _, want := range []string{
		"ezfyPopGatherPlan(city.Pop, city.PopMax, popGain, city.PopRecover)",
		"if city.Pop >= city.PopMax {",
		"city.PopRecover -= gain",
		"city.Pop += gain",
		"foodCost = foodCost * gain / baseGain",
		"ezfyConveneOverCapCooldownMs",
		"5 分钟后再试",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("Convene 缺少「民居上限两阶段召集」逻辑（缺 %s）：\n%s", want, body)
		}
	}
	for _, banned := range []string{"ezfyConvenePopMaxCfg", "ezfyHouseSoftCap", "ezfyHousePopLimitOn"} {
		if strings.Contains(body, banned) {
			t.Fatalf("Convene 仍引用已删除的 %s：\n%s", banned, body)
		}
	}
}

// TestPopGatesUseHouseCap 召集 / 移民法案共用同一套「民居上限两阶段」口径。
func TestPopGatesUseHouseCap(t *testing.T) {
	src := ezfyStripCommentLines(rawFile(t, "ezfy.go"))
	for _, banned := range []string{"ezfyConvenePopMaxCfg", "ezfyHouseSoftCap", "ezfyHousePopLimitOn", "ezfyConveneFlexOn"} {
		if strings.Contains(src, banned) {
			t.Fatalf("ezfy.go 仍引用已删除的 %s", banned)
		}
	}
	// Convene + 移民法案 ItemType31 两处共用
	if n := strings.Count(src, "ezfyPopGatherPlan("); n < 2 {
		t.Fatalf("ezfyPopGatherPlan() 只有 %d 处，至少应有 2 处（人口召集 / 移民法案 ItemType31）", n)
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
	// 到民居上限且待恢复人口为 0 → 禁用召集
	if !strings.Contains(vue, "return pop >= pm && (this.popRecover || 0) <= 0") {
		t.Fatal("前端 conveneBlocked 未按「人口已满且无待恢复人口」禁用")
	}
	// 达到/超过民居上限 → 5 分钟冷却
	if !strings.Contains(vue, "5 * 60 * 1000") {
		t.Fatal("前端 conveneCooldownMs 未实现 5 分钟冷却")
	}
	for _, banned := range []string{"convenePopMax", "housePopLimitOn", "conveneFlexibleOn", "houseSoftCapActive"} {
		if strings.Contains(vue, banned) {
			t.Fatalf("前端仍引用已删除的 %s", banned)
		}
	}

	cityVue := ezfyStripCommentLines(readFrontFile(t, "../../../../web/src/views/ezfy/modules/EzfyCity.vue"))
	// 召集页必须显示空闲人口（用户反馈「以前有现在没了」）
	if !strings.Contains(cityVue, "空闲 {{ ezfy.freePop }}") {
		t.Fatal("召集页未显示空闲人口")
	}
	// 人口已满时提示召集只恢复空闲人口
	if !strings.Contains(cityVue, "人口已达民居上限，召集只恢复空闲人口") {
		t.Fatal("召集页未提示「人口已满时召集只恢复空闲人口」")
	}
	// 超过民居上限时要有「回落」提示
	if !strings.Contains(cityVue, "popOverHouseCap") {
		t.Fatal("前端未展示「超出民居上限会缓慢回落」提示")
	}
}
