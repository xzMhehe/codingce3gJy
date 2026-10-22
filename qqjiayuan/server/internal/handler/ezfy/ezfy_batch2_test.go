package ezfy

import (
	"strings"
	"testing"
)

// TestPvPDefenderWounded PvP 守方战损也要进伤兵营，且回收率吃「守方科技 + 城守军官技能」。
//
// ★★ 2026-10-09 「pvp 伤兵也是入伤兵营、回收比例也要有科技加成、军官技能」：
//
//	原来守方战损只扣兵（仅攻方获胜时进逃兵营），打一场 PvP 守方掉几万兵一个都回不来。
//
// ★ 2026-10-09 同日追加：战报里**不再单独列**「攻方/守方伤兵入营」两行（，
// 与尾部「回收比例」重复）→ 这里改成断言「回收比例」那行带了守方数据。
func TestPvPDefenderWounded(t *testing.T) {
	src := strings.ReplaceAll(rawFile(t, "ezfy_order.go"), "\r\n", "\n")
	for _, want := range []string{
		"defHealTech = ezfyTechEffectValueAt(21, defTech[21])", // 守方科技·治愈伤兵（管理端效果数值）
		`h.officerHasSkill(cityGuard, "机械改造")`,                 // 城守军官技能
		"h.addWounded(target.ID, g.TroopId, 0, wounded)",       // 守方伤兵入营（type=0 伤兵）
		// 战报里两方的回收比例各算一份（「谁看展示谁的」）
		"healLineAtk := ezfyHealPctLine(repairedTotal, br.AttackerLosses)",
		"healLineDef := ezfyHealPctLine(defRepairedTotal, br.DefenderLosses)",
		// ★ 2026-10-19 守方正文**不再复用攻方正文**（那正是「守方战报是攻方视角」的根因）：
		//   改由 ezfyDefenderReportBody 生成，守方「回收比例」直接吃 healLineDef。
		"HealLineDef: healLineDef",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("PvP 守方伤兵入营未接入（缺 %s）", want)
		}
	}
}

// TestDefenderReportIsDefenderPerspective ★ 2026-10-19 用户两点：
//
//	① 玩家城市守方战报原来**直接复用攻方正文**（defBody := report）→ 守方看到的是
//	   「我方一支部队对 XXX 进行了征服」+ 攻方的军衔声望/军官经验/战果，完全是攻方视角。
//	   现在玩家城市与附属野地统一走 ezfyDefenderReportBody()（守方视角）。
//	② 守方也要有军功声望与军官经验（原来野地守军军官白打一场、守方零声望）。
func TestDefenderReportIsDefenderPerspective(t *testing.T) {
	src := strings.ReplaceAll(rawFile(t, "ezfy_order.go"), "\r\n", "\n")
	// ① 攻方视角正文的复用源必须消失（先剥掉注释行：注释里引用旧写法是正常的）
	code := stripGoCommentLines(src)
	if strings.Contains(code, "defBody := report") {
		t.Fatal("玩家城市守方战报仍在复用攻方正文（守方看到攻方视角）")
	}
	if strings.Contains(code, "defBody = strings.Replace(report, healLineAtk, healLineDef, 1)") {
		t.Fatal("玩家城市守方战报仍靠字符串替换攻方正文的「回收比例」")
	}
	for _, want := range []string{
		// 城市/野地共用的守方视角正文生成器
		"func (h *EzfyHandler) ezfyDefenderReportBody(s ezfyDefReportSpec) string {",
		// 守方视角描述句（敌方进攻、我方战败/守住）
		"敌方一支部队对 %s[ %d，%d ]进行了%s。战斗共持续 %d 回合，我方战斗%s",
		// ② 守方军功声望（城市守住/被攻破、野地被掠夺/被征服 都给）
		"PrestigeGain: 100, DefExp: defExp, HealLineDef: healLineDef",
		"defPrestige := 100",
		"h.addPrestige(target.UserID, 100)",
		// ② 野地守军带队军官经验
		"h.addWildGarrisonOfficerExp(wildDefOrders, wlOwnerCityID, wildDefExp)",
		// 守方「击杀数」= 攻方战损（原来误传守方自己人的损失）
		"defExp = ezfyOfficerBattleExp(enemyDeadOf(br.AttackerLosses), !win)",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("守方视角 / 守方奖励接入缺失: %s", want)
		}
	}
}

// TestNewCityRules 起新城：坐标必须是**自己的附属野地**（平原/沿海平原）+ 五项资源各 10 万。
//
// ★★ 2026-10-09 「坐标必须是自己的附属野地（平原、沿海平原），而且资源是各 10 万」。
func TestNewCityRules(t *testing.T) {
	src := strings.ReplaceAll(rawFile(t, "ezfy.go"), "\r\n", "\n")
	for _, want := range []string{
		"ezfyNewCityResCost = 100000", // 各 10 万（原来 5 万）
		"model.EzfyWildland{}",        // 查附属野地
		"只能在自己占领的野地上建新城",
		"该野地不属于你",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("起新城规则缺失（缺 %s）", want)
		}
	}
}

// TestConveneCooldown 人口召集卡控：普通 3 秒；人口达到/超过民居软上限时 5 分钟。
func TestConveneCooldown(t *testing.T) {
	src := strings.ReplaceAll(rawFile(t, "ezfy.go"), "\r\n", "\n")
	if strings.Contains(src, "nowCd-last < 5000") {
		t.Fatal("召集卡控还是旧的 5 秒")
	}
	if !strings.Contains(src, "ezfyConveneCooldownMs        = 3000") {
		t.Fatal("普通召集卡控不是 3 秒")
	}
	if !strings.Contains(src, "ezfyConveneOverCapCooldownMs = 5 * 60 * 1000") {
		t.Fatal("超过民居软上限后的召集卡控不是 5 分钟")
	}
	if !strings.Contains(src, "nowCd-last < cdMs") {
		t.Fatal("召集卡控没有按 cdMs（3 秒 / 5 分钟）判断")
	}
	for _, want := range []string{"请 3 秒后再试", "请 5 分钟后再试"} {
		if !strings.Contains(src, want) {
			t.Fatalf("召集卡控提示文案缺「%s」", want)
		}
	}
	// ★ 2026-10-21 用户报「召集按钮卡所有城市」：冷却必须按城市记，不能按玩家 uid。
	if !strings.Contains(src, "ezfyConveneMemo[city.ID] = nowCd") {
		t.Fatal("召集卡控未按 city.ID 记录冷却（会卡住同账号其它城市）")
	}
	if strings.Contains(src, "ezfyConveneMemo[uid]") {
		t.Fatal("召集卡控仍按玩家 uid 记录冷却（应改为 city.ID）")
	}
	vue := ezfyStripCommentLines(readFrontFile(t, "../../../../web/src/views/ezfy/Ezfy.vue"))
	for _, want := range []string{"atByCity[cid]", "city_id: cid"} {
		if !strings.Contains(vue, want) {
			t.Fatalf("前端 doConvene 未按城市独立冷却/未带上 city_id（缺 %s）", want)
		}
	}
	if strings.Contains(vue, "this.lastConveneAt") {
		t.Fatal("前端 doConvene 仍在用全账号共享的 lastConveneAt（应改为按城市）")
	}
}

// TestMallCatAndPagerLayout 商城分类「去数字 + 每行 4 个 + 两字名」与分页条固定位置。
func TestMallCatAndPagerLayout(t *testing.T) {
	shop := ezfyStripCommentLines(readFrontFile(t, "../../../../web/src/views/ezfy/modules/EzfyShop.vue"))
	for _, want := range []string{"mall-cat-grid", "mallCatShort(c)"} {
		if !strings.Contains(shop, want) {
			t.Fatalf("商城分类未按要求改造（缺 %s）", want)
		}
	}
	for _, bad := range []string{"全部·{{ ezfy.mallItems.length }}", "mallCatCounts[c]"} {
		if strings.Contains(shop, bad) {
			t.Fatalf("商城分类仍显示数字（发现 %s）", bad)
		}
	}
	vue := ezfyStripCommentLines(readFrontFile(t, "../../../../web/src/views/ezfy/Ezfy.vue"))
	for _, want := range []string{
		"mallCatShort (c)",
		"grid-template-columns: repeat(4, max-content)",
		"justify-content: start",
		"grid-template-columns: 4.5em 10em 4.5em",
		"margin: 8px 0 4px",
	} {
		if !strings.Contains(vue, want) {
			t.Fatalf("商城分类/分页样式缺失（缺 %s）", want)
		}
	}
	// 种子：建筑图纸库存无限（-1）
	seed := rawFile(t, "../../seed/ezfy_cfg_gen.go")
	found := false
	for _, line := range strings.Split(seed, "\n") {
		if !strings.Contains(line, `Name: "建筑图纸"`) {
			continue
		}
		found = true
		if !strings.Contains(line, "Stock: -1,") {
			t.Fatalf("商城种子「建筑图纸」库存未改成无限：%s", line)
		}
	}
	if !found {
		t.Fatal("商城种子里找不到「建筑图纸」")
	}
}

// TestMallBuyMaxDefault999 商城单次购买上限默认值必须是 999（集结令原 15）。
//
// ★ 2026-10-22 用户规则：「集结令购买上限 15 改成 999；其他消耗道具单次购买也都 999」。
func TestMallBuyMaxDefault999(t *testing.T) {
	if ezfyMallBuyMaxDef != 999 {
		t.Fatalf("商城单次购买默认上限 = %d，应为 999", ezfyMallBuyMaxDef)
	}
	vue := ezfyStripCommentLines(readFrontFile(t, "../../../../web/src/views/ezfy/Ezfy.vue"))
	for _, want := range []string{
		"mallBuyMax: 999",
		"parseInt(r.data.buy_max) > 0 ? parseInt(r.data.buy_max) : 999",
	} {
		if !strings.Contains(vue, want) {
			t.Fatalf("商城购买上限前端兜底未同步为 999（缺 %s）", want)
		}
	}
}

// TestConveneClampToCap 人口召集「贴近上限」时必须还能召集（截断到上限），不能整单拒绝。
//
// ★★ 2026-10-09 用户报「人口 46 万时再召集却不行了，应该仍可以召集、只不过是召集到上限为止」：
//
//	线上 `convene_pop_max` = 50 万、单次召集约 5 万 → 原判定
//	`city.Pop + popGain > popCap` 会把「加完会超上限」整单拒绝，
//	于是人口 45 万以上的 400 座城**一次也召集不了**，明明还差几十万才满。
//
// 现在改成：能加多少加多少（刚好加到上限），粮食按比例收；
// 只有「人口已经达到/超过上限」才拒绝。
func TestConveneClampToCap(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy.go", "func (h *EzfyHandler) Convene(")

	// 1) ★★ 2026-10-20 只按「召集人口上限」硬顶截断，不能整单拒绝；
	//    民居容量是软上限，不参与截断（超过后仍可召集，超出部分缓慢回落）。
	for _, want := range []string{
		"popCap := ezfyConvenePopMaxCfg()",
		"if popCap > 0 && city.Pop+popGain > popCap",
		"popGain = popCap - city.Pop",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("Convene 未按召集人口硬顶截断（缺 %s）", want)
		}
	}
	// 2) 截断后粮食要按比例收，否则会出现「花 20 万粮只加 2 点人口」
	if !strings.Contains(body, "foodCost = foodCost * popGain / baseGain") {
		t.Fatal("Convene 截断后粮食没按比例折算")
	}
	// 3) 已经到硬顶时仍要拒绝（不能返回 0 人口却扣粮）
	if !strings.Contains(body, "city.Pop >= popCap") {
		t.Fatal("Convene 未处理「人口已达召集上限」的拒绝分支")
	}
	// 4) 防回退：不能再引用已删除的「有效上限」硬夹逻辑
	if strings.Contains(body, "ezfyEffPopCap") {
		t.Fatal("Convene 仍引用 ezfyEffPopCap（民居软上限会错误地挡住召集）")
	}
}

// TestConvenePopGainByHouseCap 召集人口按「当前民居上限 × 配置百分比」计算（默认 10%）。
//
// ★ 2026-10-22 用户规则：「[召集] 改成当前民居上限的 10%，默认；二战系统配置可配默认 10%，
// 原来是写固定的（默认 1 万）这个配置去掉」。
func TestConvenePopGainByHouseCap(t *testing.T) {
	for _, c := range []struct {
		popMax int64
		pct    int64
		want   int64
	}{
		{0, 10, 1},        // 异常/无民居：至少 1，避免召集 0 收益
		{100, 10, 10},     // 1 级民居
		{47200, 10, 4720}, // 线上示例：4.72 万 × 10%
		{180000, 10, 18000},
		{47200, 150, 70800}, // 允许超过 100%：150% = 1.5 倍民居上限
		{10000, 250, 25000}, // 250% = 2.5 倍民居上限
		{333, 10, 33},       // 向下取整
	} {
		if got := ezfyConvenePopGainWithPct(c.popMax, c.pct); got != c.want {
			t.Fatalf("pop_max=%d pct=%d%% 召集人口=%d，期望 %d", c.popMax, c.pct, got, c.want)
		}
	}
	// 默认配置（10%）走 ezfyConvenePopGainOf 的完整路径
	if got := ezfyConvenePopGainOf(47200); got != 4720 {
		t.Fatalf("默认配置下 pop_max=47200 召集人口=%d，期望 4720", got)
	}
	body := ezfyFuncBody(t, "ezfy.go", "func (h *EzfyHandler) Convene(")
	if !strings.Contains(body, "popGain := ezfyConvenePopGainOf(city.PopMax)") {
		t.Fatal("Convene 未按当前民居上限计算召集人口")
	}
	if strings.Contains(body, "ezfyConvenePopGainCfg") {
		t.Fatal("Convene 仍在引用已删除的固定召集人数配置")
	}
}

// TestBgTickOfflineSettle 后台兜底要覆盖「到期订单 + 到期建筑」——离线玩家世界也要往前走。
//
// ★★ 2026-10-09 用户反馈「没在线 军队就一直在路上、建筑升级也一样」。
func TestBgTickOfflineSettle(t *testing.T) {
	src := strings.ReplaceAll(rawFile(t, "ezfy_battle_room.go"), "\r\n", "\n")
	for _, want := range []string{
		"status = 0 AND arrive_time > 0 AND arrive_time <= ?", // 行军抵达
		"status = 2 AND return_time > 0 AND return_time <= ?", // 返航到城
		"b.end_time > 0 AND b.end_time <= ?",                  // 建筑升级到期
		"h.settleDueCities(uid)",                              // 逐城完整结算
		"ezfyBgTickMaxUids",                                   // 有每轮上限，防雪崩
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("后台兜底没覆盖离线到期活（缺 %s）", want)
		}
	}
}

// TestExchangeCancelSettlesFirst 交易所下架前必须**先懒结算**，否则 saveCityRes 会把这段时间的产量覆盖掉。
//
// ★★ 2026-10-09 用户反馈「交易所下架资源会被吃掉」：city 是上次落库的旧值，
// 直接退回 + 整行写回 → 产量凭空少一截。退回被封顶丢弃时也要明确提示。
// TestSysSellFeeBlocksMallPackArbitrage 商城「大资源包」买价 5 万，整套卖给系统毛收入 6.5 万。
//
// ★★ 2026-10-22 用户反馈套利漏洞：10% 手续费后实得 58,500 > 50,000，净赚 8,500。
// 默认手续费必须高到让这条套利亏钱，否则改配置时很容易把漏洞放回来。
func TestSysSellFeeBlocksMallPackArbitrage(t *testing.T) {
	const packPrice int64 = 50000 // 大资源包（ezfy_cfg_gen.go ID 1002）
	const packEach int64 = 100000 // 粮/钢/油/稀各 10 万
	gross := packEach * int64(10+10+20+25) / 100
	net := gross * int64(100-ezfySysSellFeeDefault) / 100
	if net >= packPrice {
		t.Fatalf("默认手续费 %d%% 仍有套利空间：卖得 %d >= 买价 %d", ezfySysSellFeeDefault, net, packPrice)
	}
	if ezfySysSellDailyLimitDef <= 0 {
		t.Fatalf("每日回收上限默认值必须为正数，当前 %d", ezfySysSellDailyLimitDef)
	}
}

func TestExchangeCancelSettlesFirst(t *testing.T) {
	src := strings.ReplaceAll(rawFile(t, "ezfy_chat_exchange.go"), "\r\n", "\n")
	i := strings.Index(src, "func (h *EzfyHandler) ExchangeCancel(")
	j := strings.Index(src, "func (h *EzfyHandler) ExchangeSysSell(")
	if i < 0 || j <= i {
		t.Fatal("找不到 ExchangeCancel 函数区间")
	}
	body := src[i:j]
	ci := strings.Index(body, "h.calcResource(&city)")
	si := strings.Index(body, "h.saveCityRes(&city)")
	if ci < 0 {
		t.Fatalf("下架前没有先懒结算（会吃掉这段时间的产量）：\n%s", body)
	}
	if si < 0 || ci > si {
		t.Fatalf("calcResource 必须在 saveCityRes 之前：\n%s", body)
	}
	if !strings.Contains(body, "因资源已达最大值被丢弃") {
		t.Fatalf("退回量被封顶丢弃时没有提示：\n%s", body)
	}
}
