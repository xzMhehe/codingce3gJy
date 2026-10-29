package ezfy

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestBotDailyAtkCapConfigurable 人机「每人每日出征」必须**可配置且真正生效**。
//
// 背景（2026-10-29 用户反馈）：「人机管理 → 每人每日出征 这个配置是不是有问题，超过 100 没法配置」。
// 实测两处 bug：
//  1. 管理端输入框 :max=100 + 后端 `if v > 100 { v = 100 }` → 想配 200 会被悄悄截成 100；
//  2. 这个配置以前**只是展示**（botAtkQuotaOK 真正卡的是难度阶梯 40/50/60），改它等于没改。
//
// 断言：
//  1. 后端保存上限放宽到 100000（不再写死 100）；
//  2. botAtkQuotaOK 先读配置，0 才回落难度阶梯；
//  3. 老库停在旧默认 3 的行由一次性修正归零（否则一启用就把人机砍到 3 次/天）。
func TestBotDailyAtkCapConfigurable(t *testing.T) {
	bot := rawFile(t, "ezfy_bot.go")
	admin := rawFile(t, "ezfy_admin_bot.go")

	// ① 保存上限放宽（只看 AdminEzfyBotConfigSet 的 body）
	setBody := funcBody(admin, "func (h *EzfyAdmin) AdminEzfyBotConfigSet(")
	if setBody == "" {
		t.Fatal("找不到 AdminEzfyBotConfigSet 函数体")
	}
	if !strings.Contains(setBody, "100000") {
		t.Fatal("每日出征上限没放宽到 100000（用户配 200 会被截断）")
	}
	if strings.Contains(setBody, "v > 100 {") {
		t.Fatal("AdminEzfyBotConfigSet 仍把每日出征上限截断到 100")
	}

	// ② 真正生效：配置优先，0 回落难度阶梯
	quota := funcBody(bot, "func (h *EzfyHandler) botAtkQuotaOK(")
	if quota == "" {
		t.Fatal("找不到 botAtkQuotaOK 函数体")
	}
	if !strings.Contains(quota, "ezfyBotDailyAtkCap()") {
		t.Fatal("botAtkQuotaOK 没读配置 ezfyBotDailyAtkCap() —— 管理端改了「每人每日出征」等于没改")
	}
	if !strings.Contains(quota, "ezfyBotAtkDailyOf(b.Level)") {
		t.Fatal("botAtkQuotaOK 丢了难度阶梯兜底（0 = 按难度 40/50/60）")
	}
	// 读取函数不能再把「没配」写成 3（0 = 按难度，是合法值）
	capFn := funcBody(bot, "func ezfyBotDailyAtkCap()")
	if capFn == "" {
		t.Fatal("找不到 ezfyBotDailyAtkCap 函数体")
	}
	if strings.Contains(capFn, "return 3") {
		t.Fatal("ezfyBotDailyAtkCap 仍把未配置兜底成 3 —— 会让「0 = 按难度」失效")
	}

	// ③ 一次性把旧默认 3 归零（settings 打标，只跑一次）
	seedSrc := rawFile(t, filepath.Join("..", "..", "seed", "seed.go"))
	fix := funcBody(seedSrc, "func ezfyBotAtkDailyDefault0(")
	if fix == "" {
		t.Fatal("缺 seed.ezfyBotAtkDailyDefault0（老库 bot_atk_daily=3 会让新逻辑把人机砍到 3 次/天）")
	}
	if !strings.Contains(fix, "ezfyOneTimeLimitFix") {
		t.Fatal("ezfyBotAtkDailyDefault0 没有一次性标记位 —— 每次重启都会把管理员填的值改回 0")
	}
	if !strings.Contains(seedSrc, "ezfyBotAtkDailyDefault0(db)") {
		t.Fatal("seed 没有调用 ezfyBotAtkDailyDefault0")
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
