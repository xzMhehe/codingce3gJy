package ezfy

import (
	"strings"
	"testing"
)

// ============ 人口上限口径守卫（2026-10-20 修正版）============
//
// 用户最终口径：
//  1. 民居容量是**软上限**：超过后仍然可以继续召集；
//  2. 召集人口 + 对应空闲人口同步增加；
//  3. 超出民居上限的人口会**慢慢回落**到民居上限，不是立刻夹平；
//  4. 人口达到/超过民居上限时，召集冷却为 5 分钟（普通情况仍是 3 秒）；
//  5. 「召集人口上限」是硬顶（自然增长/召集都不能超过），0 = 不限。

func TestEzfyPopCapOf(t *testing.T) {
	cases := []struct {
		name       string
		houseOn    bool
		flexOn     bool
		popMax     int64
		conveneMax int64
		want       int64
	}{
		{"都不限", false, false, 0, 0, 0},
		{"只开召集上限", false, false, 45000, 100000, 100000},
		{"只开民居软上限", true, false, 45000, 0, 45000},
		{"两个都开取小（民居更小）", true, false, 45000, 100000, 45000},
		{"两个都开取小（召集更小）", true, false, 45000, 30000, 30000},
		{"允许长期超过民居 → 只看召集上限", true, true, 45000, 100000, 100000},
		{"允许长期超过 + 无召集上限 → 不限", true, true, 45000, 0, 0},
		{"民居关掉 → 只看召集上限", false, false, 45000, 100000, 100000},
		{"pop_max=0（城里没民居）不参与", true, false, 0, 100000, 100000},
	}
	for _, c := range cases {
		if got := ezfyPopCapOf(c.houseOn, c.flexOn, c.popMax, c.conveneMax); got != c.want {
			t.Errorf("%s: ezfyPopCapOf(house=%v, flex=%v, popMax=%d, convene=%d) = %d, want %d",
				c.name, c.houseOn, c.flexOn, c.popMax, c.conveneMax, got, c.want)
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

// TestCalcResourceSoftDecay 懒结算必须走「软上限 + 缓慢回落」，而不是立刻夹平。
func TestCalcResourceSoftDecay(t *testing.T) {
	body := ezfyStripCommentLines(ezfyFuncBody(t, "ezfy.go", "func (h *EzfyHandler) calcResourceD("))
	for _, want := range []string{
		"ezfyConvenePopMaxCfg()", // 召集人口上限是硬顶，自然增长也不能超
		"ezfyHouseSoftCap(city.PopMax)",
		"ezfySettlePopOverCap(city.Pop, soft, city.PopOverCapAt, now, ezfyPopOverCapDecayPct())",
		`"pop_over_cap_at": city.PopOverCapAt`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("calcResourceD 缺少软上限回落逻辑（缺 %s）：\n%s", want, body)
		}
	}
	// 防回退：旧的「立刻夹到 effective cap」写法必须删除
	if strings.Contains(body, "ezfyEffPopCap(city.PopMax)") || strings.Contains(body, "city.Pop = cap") {
		t.Fatalf("calcResourceD 还在立刻夹平超限人口（用户要求缓慢回落）：\n%s", body)
	}
}

// TestConveneAllowsOverHouseCap 召集不能被民居软上限挡住，且到软上限后冷却 5 分钟。
func TestConveneAllowsOverHouseCap(t *testing.T) {
	body := ezfyStripCommentLines(ezfyFuncBody(t, "ezfy.go", "func (h *EzfyHandler) Convene("))
	// 只按召集人口上限硬顶截断
	if !strings.Contains(body, "popCap := ezfyConvenePopMaxCfg()") {
		t.Fatalf("Convene 未使用召集人口上限作为硬顶：\n%s", body)
	}
	if !strings.Contains(body, "popGain = popCap - city.Pop") {
		t.Fatalf("Convene 未按召集硬顶截断 popGain：\n%s", body)
	}
	// 不能再被民居软上限截断
	if strings.Contains(body, "ezfyEffPopCap") || strings.Contains(body, "city.PopMax - city.Pop") {
		t.Fatalf("Convene 仍被民居上限截断（超过上限应仍可召集）：\n%s", body)
	}
	// 人口达到/超过民居软上限 → 5 分钟冷却
	for _, want := range []string{
		"ezfyHouseSoftCap(city.PopMax)",
		"ezfyConveneOverCapCooldownMs",
		"5 分钟后再试",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("Convene 缺少「民居软上限时 5 分钟冷却」逻辑（缺 %s）：\n%s", want, body)
		}
	}
}

// TestPopGatesUseConveneHardCap 人口召集 / 懒结算 / 移民法案 三处都只把召集上限当硬顶。
func TestPopGatesUseConveneHardCap(t *testing.T) {
	src := ezfyStripCommentLines(rawFile(t, "ezfy.go"))
	if strings.Contains(src, "ezfyEffPopCap(") {
		t.Fatal("ezfy.go 仍引用已删除的 ezfyEffPopCap")
	}
	// calcResourceD 的硬顶夹取 + Convene + 移民法案 ItemType31
	if n := strings.Count(src, "ezfyConvenePopMaxCfg()"); n < 3 {
		t.Fatalf("ezfyConvenePopMaxCfg() 只有 %d 处，至少应有 3 处"+
			"（懒结算硬顶 / 人口召集 / 移民法案 ItemType31）", n)
	}
}

// TestFrontendPopCapSameFormula 前端展示/按钮/冷却必须和后端同口径。
func TestFrontendPopCapSameFormula(t *testing.T) {
	vue := strings.ReplaceAll(readFrontFile(t, "../../../../web/src/views/ezfy/Ezfy.vue"), "\r\n", "\n")
	vue = ezfyStripCommentLines(vue)
	// popCap 仍用于展示：min(召集上限, 民居软上限)
	if !strings.Contains(vue, "this.housePopLimitOn && !this.conveneFlexibleOn") {
		t.Fatal("前端 popCap 丢了「民居限制开 + 允许长期超过关」条件")
	}
	if !strings.Contains(vue, "Math.min(cap, pm)") {
		t.Fatal("前端 popCap 未与召集上限取小")
	}
	// 按钮只受召集硬顶限制（民居软上限不挡召集）
	if !strings.Contains(vue, "return this.convenePopMax > 0 && pop >= this.convenePopMax") {
		t.Fatal("前端 conveneBlocked 未只按召集人口硬顶禁用（民居软上限不该挡召集）")
	}
	// 达到/超过民居软上限 → 5 分钟冷却
	if !strings.Contains(vue, "5 * 60 * 1000") {
		t.Fatal("前端 conveneCooldownMs 未实现 5 分钟冷却")
	}

	cityVue := ezfyStripCommentLines(readFrontFile(t, "../../../../web/src/views/ezfy/modules/EzfyCity.vue"))
	// 召集页必须显示空闲人口（用户反馈「以前有现在没了」）
	if !strings.Contains(cityVue, "空闲 {{ ezfy.freePop }}") {
		t.Fatal("召集页未显示空闲人口")
	}
	// 不再显示「民居容量为硬上限…」这种与新规则矛盾的冗余提示
	if strings.Contains(cityVue, "民居容量为硬上限") {
		t.Fatal("召集页仍在提示「民居容量为硬上限」（现在已改为软上限）")
	}
	// 超过民居上限时要有「回落」提示
	if !strings.Contains(cityVue, "popOverHouseCap") {
		t.Fatal("前端未展示「超出民居上限会缓慢回落」提示")
	}
}
