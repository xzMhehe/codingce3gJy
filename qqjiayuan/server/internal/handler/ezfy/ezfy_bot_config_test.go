package ezfy

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestBotDailyAtkCapConfigurable 人机「每日出征次数」必须**按难度可配**且默认 60/100/160。
//
// 背景（2026-10-29 用户反馈）：
//
//	① 「每人每日出征 超过 100 没法配置」→ 原配置只展示不生效、且前后端都写死 100；
//	② 「按难度阶梯 40/50/60 也要能配置，默认 60/100/160」→ 改成按难度三个值，全局一刀切的旧列废弃。
//
// 断言：
//  1. botAtkQuotaOK 走 ezfyBotAtkDailyOf（唯一实现），阶梯读配置 lv1/2/3；
//  2. 默认值常量是 60/100/160，且 0/负数回落默认（不能出现「0 = 不打人」）；
//  3. 旧的全局 bot_atk_daily 不再参与判断；
//  4. 管理端 GET 下发三项、SET 校验并写三项；seed 补三列。
func TestBotDailyAtkCapConfigurable(t *testing.T) {
	bot := rawFile(t, "ezfy_bot.go")
	admin := rawFile(t, "ezfy_admin_bot.go")

	// ① 唯一实现 + 读配置
	quota := funcBody(bot, "func (h *EzfyHandler) botAtkQuotaOK(")
	if quota == "" {
		t.Fatal("找不到 botAtkQuotaOK 函数体")
	}
	if !strings.Contains(quota, "ezfyBotAtkDailyOf(b.Level)") {
		t.Fatal("botAtkQuotaOK 没走 ezfyBotAtkDailyOf(b.Level)（每日出征上限的唯一实现）")
	}
	ofBody := funcBody(bot, "func ezfyBotAtkDailyOf(")
	if ofBody == "" {
		t.Fatal("找不到 ezfyBotAtkDailyOf 函数体")
	}
	for _, col := range []string{"BotAtkDailyLv1", "BotAtkDailyLv2", "BotAtkDailyLv3"} {
		if !strings.Contains(ofBody, "ezfyCfg.limit."+col) {
			t.Fatalf("ezfyBotAtkDailyOf 没读 %s —— 后台改了不生效", col)
		}
	}
	if strings.Contains(ofBody, "BotAtkDaily ") || strings.Contains(ofBody, "limit.BotAtkDaily;") {
		t.Fatal("ezfyBotAtkDailyOf 仍在读已废弃的全局 bot_atk_daily")
	}

	// ② 默认值 60/100/160 + 非法值回落
	for _, s := range []string{"ezfyBotAtkDailyLv1Def = 60", "ezfyBotAtkDailyLv2Def = 100", "ezfyBotAtkDailyLv3Def = 160"} {
		if !strings.Contains(strings.Join(strings.Fields(bot), " "), s) {
			t.Fatalf("默认值不是 60/100/160（缺 %q）", s)
		}
	}
	if !strings.Contains(ofBody, "ezfyBotAtkDailyDefOf(level)") {
		t.Fatal("ezfyBotAtkDailyOf 缺「0/负数回落默认」的兜底")
	}

	// ③ 管理端 GET 下发三项
	getBody := funcBody(admin, "func (h *EzfyAdmin) AdminEzfyBotConfig(")
	if getBody == "" {
		t.Fatal("找不到 AdminEzfyBotConfig 函数体")
	}
	for _, k := range []string{"atk_daily_lv1", "atk_daily_lv2", "atk_daily_lv3"} {
		if !strings.Contains(getBody, `"`+k+`"`) {
			t.Fatalf("AdminEzfyBotConfig 没下发 %s", k)
		}
	}
	// ④ 管理端 SET 校验 + 写三项
	setBody := funcBody(admin, "func (h *EzfyAdmin) AdminEzfyBotConfigSet(")
	if setBody == "" {
		t.Fatal("找不到 AdminEzfyBotConfigSet 函数体")
	}
	for _, col := range []string{"bot_atk_daily_lv1", "bot_atk_daily_lv2", "bot_atk_daily_lv3"} {
		if !strings.Contains(setBody, col) {
			t.Fatalf("AdminEzfyBotConfigSet 没写 %s", col)
		}
	}
	if !strings.Contains(setBody, "100000") {
		t.Fatal("每日出征次数上限没放宽（应允许 1~100000）")
	}

	// ⑤ seed 补三列（默认 60/100/160）
	seedSrc := rawFile(t, filepath.Join("..", "..", "seed", "seed.go"))
	for _, col := range []string{"bot_atk_daily_lv1", "bot_atk_daily_lv2", "bot_atk_daily_lv3"} {
		if !strings.Contains(seedSrc, `"`+col+`"`) {
			t.Fatalf("seed 没补列 %s（老库会缺列）", col)
		}
	}
}

// TestBotMultiTargetPlans 人机必须**多目标并行**，不能盯着一个玩家打到死。
//
// 背景（2026-10-29 用户反馈）：「人机优先攻击附近打自己的玩家；人机可以出征多个队列到附近玩家，
// 现在是盯着一个人打，玩家反馈他也在附近却没有打他」。
//
// 断言：
//  1. 计划查询按 updated_at 升序（最久没动过的先推进）→ 出击后自然轮到下一个目标；
//  2. 计划未满（< ezfyBotMaxPlans）时还会继续锁定**新目标**（不是「没有计划才找」）；
//  3. botPickTarget 排除已有活跃计划的玩家（否则每轮锁的都是同一个人）；
//  4. 反击窗口不再是 10 分钟（原来打完就忘，反击几乎不触发），且按最近被打排序。
func TestBotMultiTargetPlans(t *testing.T) {
	bot := rawFile(t, "ezfy_bot.go")

	// ① 轮换
	if !strings.Contains(bot, `Order("updated_at ASC, id ASC").Limit(ezfyBotMaxPlans)`) {
		t.Fatal("计划查询没按 updated_at 升序轮换 —— 人机会一直打同一个目标")
	}
	// ② 多计划
	if !strings.Contains(bot, "if len(plans) < ezfyBotMaxPlans {") {
		t.Fatal("botMilitary 只在「没有计划」时才找新目标 —— 无法多队列打附近多个玩家")
	}
	if !strings.Contains(bot, "ezfyBotMaxPlans = 3") {
		t.Fatal("缺 ezfyBotMaxPlans 常量（多目标并行上限）")
	}
	// ③ 排除已计划玩家
	if !strings.Contains(bot, "bt2.status IN (0,1,2,3)") {
		t.Fatal("botPickTarget 没排除「已有计划」的玩家 —— 每轮锁定的还是同一个人")
	}
	// ④ 反击口径
	if !strings.Contains(bot, "ezfyBotRevengeWindowMs") {
		t.Fatal("反击窗口没参数化（ezfyBotRevengeWindowMs）")
	}
	if !strings.Contains(bot, "ORDER BY last_ms DESC") {
		t.Fatal("反击没按「最近被打」排序 —— 刚打过我的玩家应该优先回打")
	}
	if strings.Contains(bot, "o.created_at_ms >= ? - 600000") {
		t.Fatal("反击窗口还是 10 分钟（打完就忘，反击几乎不触发）")
	}
}
