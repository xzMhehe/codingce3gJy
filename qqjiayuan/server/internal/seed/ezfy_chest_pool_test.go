package seed

import "testing"

// TestChestPoolCoversAllSets —— 用户规则「套装军官装备只能通过宝箱开启」的**兜底守卫**。
//
// ★ 2026-09-22 变更：宝箱开出来是**整套**（Kind=3，RefId = 套装 id），
// 所以守卫从「每个件都在池里」变成「**每个套装都在某个宝箱里**」——
// 漏一套，那套就永远拿不到。
func TestChestPoolCoversAllSets(t *testing.T) {
	inPool := map[int]bool{}
	seenID := map[int]bool{}
	for _, it := range buildEzfyChestItems() {
		if seenID[it.ID] {
			t.Fatalf("宝箱奖池 ID 重复: %d（同一宝箱内 ID 必须唯一）", it.ID)
		}
		seenID[it.ID] = true
		if it.Kind == 3 {
			inPool[it.RefId] = true
		}
	}

	// 第一批：套装 1~17
	for _, s := range ezfyEquipSetSeeds {
		if !inPool[s.ID] {
			t.Fatalf("套装 %d(%s) 不在任何宝箱奖池里 → 永远无法获得", s.ID, s.Name)
		}
	}
	// 第二批：军官装备系列 21~26
	for _, s := range ezfyOfficerSeriesSeeds {
		if !inPool[s.ID] {
			t.Fatalf("系列 %d(%s) 不在任何宝箱奖池里 → 永远无法获得", s.ID, s.SetName)
		}
	}
}

// TestChestSetPlanSetsExist —— 奖池里引用的套装 id 必须真实存在（防止写错 id 开出空箱）
func TestChestSetPlanSetsExist(t *testing.T) {
	known := map[int]bool{}
	for _, s := range ezfyEquipSetSeeds {
		known[s.ID] = true
	}
	for _, s := range ezfyOfficerSeriesSeeds {
		known[s.ID] = true
	}
	for _, plan := range ezfyChestSetPlan {
		for _, sid := range plan.SetIDs {
			if !known[sid] {
				t.Fatalf("宝箱 %d 的奖池引用了不存在的套装 %d", plan.ChestID, sid)
			}
		}
	}
}

// TestLooseEquipPriceInRange —— 散件定价必须落在合理区间
//
// ★ 2026-09-26 「商城装备页卖的太低，现钻石 × 10」→ 原 10~50 整体涨到 100~500。
// ★ 2026-10-07 散件改成「显式价优先」（按线上现值，9 件 1000 / 4 件 100）→ 上限放宽到 1000。
// ★★ 2026-10-14 用户「散件 钻石 按 50-100 合理梯度调整价格」→ 全档降到 50~100。
//
//	这里校验的是**最终写入的价**（ezfyLooseEquipPrice），字面量写错档会被测出来。
//	★ 套装件 / 系列件不在本用例范围（它们走 ezfySetDiamondPrice 阶梯 / 种子字面量，见 TestSetDiamondPriceLadder）。
func TestLooseEquipPriceInRange(t *testing.T) {
	check := func(name string, p int64) {
		if p < 50 || p > 100 {
			t.Fatalf("%s 的定价 %d 超出 50~100 钻区间", name, p)
		}
	}
	for _, l := range ezfyOfficerEquipLooseSeeds {
		check(l.Name, ezfyLooseEquipPrice(l.Diamond, l.Level, l.Dmg, l.Def, l.Hp, l.Move, l.Crit, l.CritDmg))
	}
}

// TestChestPoolWeightPositive —— 权重必须为正，否则该奖品抽不到（等于没进池）。
func TestChestPoolWeightPositive(t *testing.T) {
	for _, it := range buildEzfyChestItems() {
		if it.Weight <= 0 {
			t.Fatalf("宝箱 %d 的奖品(chest_id=%d kind=%d ref_id=%d) 权重为 %d，永远抽不到",
				it.ID, it.ChestId, it.Kind, it.RefId, it.Weight)
		}
	}
}

// TestChestPoolEveryChestNonEmpty —— 每个宝箱都要有奖池，否则开箱只会返回「空箱」。
func TestChestPoolEveryChestNonEmpty(t *testing.T) {
	cnt := map[int]int{}
	for _, it := range buildEzfyChestItems() {
		cnt[it.ChestId]++
	}
	for _, c := range ezfyChestSeeds {
		if cnt[c.ID] == 0 {
			t.Fatalf("宝箱 %d(%s) 的奖池为空", c.ID, c.Name)
		}
	}
}

// TestChestSetWeightOrdered —— 低档套装应比高档更容易出（权重单调不增）。
func TestChestSetWeightOrdered(t *testing.T) {
	prev := -1
	for tier := 1; tier <= 4; tier++ {
		w := ezfyChestSetWeight(tier)
		if w <= 0 {
			t.Fatalf("Tier %d 的权重必须为正，实际 %d", tier, w)
		}
		if prev >= 0 && w > prev {
			t.Fatalf("Tier %d 权重(%d) 高于低档 Tier %d(%d)，高档不该更容易出",
				tier, w, tier-1, prev)
		}
		prev = w
	}
}

// TestChestSetProbabilityBaseline 各宝箱「出套装概率」的**现状基线**
// （★★ 2026-10-18 用户定案：按线上 1 区现状，不再强制 1%）
//
// 历史：2026-10-09 曾要求「战神/荣耀/统帅宝箱 出套装概率 1%」。但 1 区在管理端被反复调过奖池，
// 现状已明显偏离该规则 —— 战神 0.34%、荣耀 **6.65%**、统帅 3.00%（其中荣耀/统帅各多了一行
// 「套装 1 × 权重 200」，疑似误加，但用户确认按现状定案）。
//
// 所以这里不再校验 1%，而是**钉住当前基线**：谁改奖池把概率带偏，这个用例就会红，
// 逼改动者显式确认「是有意调整，还是误加了一行」。基线来源 = 线上 1 区 2026-10-18 快照
// （`ezfyChestPoolSeedRows`，与 qq_ezzt.ezfy_cfg_chest_item 逐行一致）。
func TestChestSetProbabilityBaseline(t *testing.T) {
	want := map[int][2]int{ // chestID -> {套装权重, 总权重}
		1: {0, 5300},
		2: {2, 112},
		3: {3, 133},
		4: {4, 1194},
		5: {208, 3128},
		6: {436, 14536},
	}
	got := map[int][2]int{}
	for _, it := range buildEzfyChestItems() {
		v := got[it.ChestId]
		v[1] += it.Weight
		if it.Kind == 3 {
			v[0] += it.Weight
		}
		got[it.ChestId] = v
	}
	for _, chest := range []int{1, 2, 3, 4, 5, 6} {
		w, g := want[chest], got[chest]
		if g != w {
			pct := 0.0
			if g[1] > 0 {
				pct = float64(g[0]) * 100 / float64(g[1])
			}
			t.Fatalf("宝箱 %d 奖池基线变了：套装权重/总权重 = %d/%d（出套装 %.2f%%），基线 %d/%d。\n"+
				"若是有意调整奖池，请同步更新本用例的基线；若是误加（例如多了一行套装），请把它修掉。",
				chest, g[0], g[1], pct, w[0], w[1])
		}
	}
}

// TestSetDiamondPriceLadder —— 套装件 1~17 的商城钻石单价阶梯（2026-10-14 用户「都是钻石价」）。
//
// ★★ 2026-10-15 用户「套装玩家反馈太贵了 按品质 100-200 合理梯度调整钻石价格」：
//   - 1~7 阶梯 100/100/120/120/150/160/140；
//   - 8~17 用种子 s.Diamond（T3 150/160，T4 顶套 200 封顶）；
//   - 系列件 21~26 的钻价 = 种子 s.Diamond（130/140/160/180，见 TestSeriesDiamondPriceInRange）。
// 锁住这张价目表：防止日后有人改 ezfySetDiamondPrice 或把 8~17 的 Diamond 字面量改坏。
func TestSetDiamondPriceLadder(t *testing.T) {
	want := map[int]int64{
		1: 100, 2: 100, 3: 120, 4: 120, 5: 150, 6: 160, 7: 140,
		8: 180, 9: 180, 10: 180, 11: 150, 12: 160,
		13: 180, 14: 180, 15: 180, 16: 200, 17: 200,
	}
	// 1~7 由 ezfySetDiamondPrice 决定（fallback 传 0，确保走阶梯）
	for id := 1; id <= 7; id++ {
		if got := ezfySetDiamondPrice(id, 0); got != want[id] {
			t.Fatalf("套装 %d 的钻石单价 = %d，期望 %d", id, got, want[id])
		}
	}
	// 8~17 必须与种子 s.Diamond 一致（用 fallback 走字面量）
	for _, s := range ezfyEquipSetSeeds {
		if s.ID < 8 {
			continue
		}
		if got := ezfySetDiamondPrice(s.ID, s.Diamond); got != want[s.ID] {
			t.Fatalf("套装 %d(%s) 的钻石单价 = %d，期望 %d", s.ID, s.Name, got, want[s.ID])
		}
		if s.Diamond != want[s.ID] {
			t.Fatalf("套装 %d(%s) 种子里 Diamond=%d，期望 %d（8~17 的值必须写在种子字面量里）",
				s.ID, s.Name, s.Diamond, want[s.ID])
		}
	}
	// 每个套装都必须在表里（漏项会让该套装件价 0 = 上不了架）
	for _, s := range ezfyEquipSetSeeds {
		if _, ok := want[s.ID]; !ok {
			t.Fatalf("套装 %d(%s) 没有钻石单价（价 0 会上不了架）", s.ID, s.Name)
		}
	}
	// 上架依赖：全部 17 套的最终价必须 > 0
	for _, s := range ezfyEquipSetSeeds {
		if p := ezfySetDiamondPrice(s.ID, s.Diamond); p <= 0 {
			t.Fatalf("套装 %d(%s) 钻石价 %d，上架会因「无价格」被过滤掉", s.ID, s.Name, p)
		}
	}
}

// TestSeriesDiamondPriceInRange —— 系列件 21~26（商城「套装」页在售的 11 件套单件）钻价必须在 100~200。
//
// ★★ 2026-10-15 用户「套装玩家反馈太贵了 按品质 100-200 合理梯度调整钻石价格」：
//
//	系列件与套装件同属商城「套装」页，单件价从 250~400 再降到 130~180。
func TestSeriesDiamondPriceInRange(t *testing.T) {
	for _, s := range ezfyOfficerSeriesSeeds {
		if s.Diamond < 100 || s.Diamond > 200 {
			t.Fatalf("系列 %d(%s) 的单件钻价 %d 超出 100~200 区间", s.ID, s.SetName, s.Diamond)
		}
	}
}
