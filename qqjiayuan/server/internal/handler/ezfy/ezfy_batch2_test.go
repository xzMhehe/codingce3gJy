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
		`h.officerHasSkill(cityGuard, "机械改造")`,           // 城守军官技能
		"h.addWounded(target.ID, g.TroopId, 0, wounded)", // 守方伤兵入营（type=0 伤兵）
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
}

// TestConveneClampToCap 人口召集「贴近上限」时必须还能召集（截断到上限），不能整单拒绝。
//
// ★★ 2026-10-09 用户报「人口 46 万时再召集却不行了，应该仍可以召集、只不过是召集到上限为止」：
//
//	线上 `convene_pop_max` = 50 万、`convene_pop_gain` = 5 万 → 原判定
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
func TestExchangeCancelSettlesFirst(t *testing.T) {
	src := strings.ReplaceAll(rawFile(t, "ezfy_chat_exchange.go"), "\r\n", "\n")
	i := strings.Index(src, "func (h *EzfyHandler) ExchangeCancel(")
	j := strings.Index(src, "// ezfySysSellFeePct")
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
