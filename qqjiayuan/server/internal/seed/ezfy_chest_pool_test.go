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
//
//	这里校验的是**最终写入的价**（ezfyLooseEquipPrice），字面量写错档会被测出来。
//	★ 套装件 / 系列件**不单卖**（价恒为 0，见 seedEzfyEquipSets），不参与本用例。
func TestLooseEquipPriceInRange(t *testing.T) {
	check := func(name string, p int64) {
		if p < 100 || p > 1000 {
			t.Fatalf("%s 的定价 %d 超出 100~1000 钻区间", name, p)
		}
	}
	for _, l := range ezfyOfficerEquipLooseSeeds {
		check(l.Name, ezfyLooseEquipPrice(l.Diamond, l.Dmg, l.Def, l.Hp, l.Move, l.Crit, l.CritDmg))
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

// TestChestSetProbabilityOnePercent 4/5/6 号箱的「出套装概率」必须 ≈ 1%。
//
// ★★ 2026-10-09 用户要求：「战神宝箱、荣耀宝箱、统帅宝箱 出套装概率应该是 1%」。
//
//	口径 = 套装总权重 ÷ 全部奖品总权重。这三个箱子各自加了一批道具（11 个 × 权重 100），
//	道具权重（约 1190~1220）远大于套装 → 套装**总权重**必须压到 12 才能维持 1%
//	（战神 2 套各 6 / 荣耀 4 套各 3 / 统帅 12 套各 1，见 ezfyChestOverride）。
//
// ⚠️ **以后往这三个箱子加奖品或调权重时，记得同步套装权重**，否则概率会被带偏；
// 老库的行由 `migrateEzfyChestSetWeight` 幂等迁移（只改「还是旧值」的行）。
func TestChestSetProbabilityOnePercent(t *testing.T) {
	items := buildEzfyChestItems()
	for _, chestID := range []int{4, 5, 6} {
		setW, total := 0, 0
		for _, it := range items {
			if it.ChestId != chestID {
				continue
			}
			total += it.Weight
			if it.Kind == 3 {
				setW += it.Weight
			}
		}
		if total == 0 {
			t.Fatalf("宝箱 %d 奖池为空", chestID)
		}
		pct := float64(setW) * 100 / float64(total)
		if pct < 0.9 || pct > 1.1 {
			t.Fatalf("宝箱 %d 出套装概率 = %.2f%%（套装权重 %d / 总权重 %d），应在 1%% 附近",
				chestID, pct, setW, total)
		}
	}
}
