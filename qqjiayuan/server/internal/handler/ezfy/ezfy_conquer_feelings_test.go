package ezfy

import (
	"strings"
	"testing"
)

// TestConquerFeelingsSameForBot —— 防回归：人机征服扣民心必须与真人**同一口径**。
//
// 背景（2026-10-29 用户反馈）：「AI 征服别人 玩家反馈还是 -90 不对，人机征服玩家民心减多少
// 也按二战系统配置的正常玩家的减，人机也是正常玩家，只是电脑控制」。
//
// 历史：2026-10-27 曾给人机单开一份 `bot_conquer_feelings_max`（默认 100 = 一次打光），
// 玩家被 AI 打一次就「民心 -90」，与真人单次最多 -5 完全不对等。
//
// 断言：
//  1. 征服扣民心的封顶只走 ezfyConquerFeelingsCfg()，函数体内**不许出现 IsBot 分支**；
//  2. 人机专用配置 `ezfyBotConquerFeelingsCfg` / `BotConquerFeelingsMax` 已彻底删除；
//  3. 管理端配置项也一并去掉（`bot_conquer_feelings_max` 不再出现）。
func TestConquerFeelingsSameForBot(t *testing.T) {
	order := rawFile(t, "ezfy_order.go")

	// ① 找到「征服玩家城市」块里的民心封顶逻辑
	i := strings.Index(order, "feelingDrop := int(surv / 2000)")
	if i < 0 {
		t.Fatal("找不到征服民心扣减的计算（feelingDrop）")
	}
	tail := order[i:]
	end := strings.Index(tail, "cur := target.Feelings - feelingDrop")
	if end < 0 {
		t.Fatal("找不到「cur := target.Feelings - feelingDrop」")
	}
	seg := tail[:end]
	if !strings.Contains(seg, "ezfyConquerFeelingsCfg()") {
		t.Fatal("征服扣民心没走 ezfyConquerFeelingsCfg()（统一配置）")
	}
	if strings.Contains(seg, "IsBot") {
		t.Fatal("征服扣民心块里仍有 IsBot 分支 —— 人机与真人口径又分叉了（玩家会看到 AI 一次打掉几十民心）")
	}
	if strings.Contains(seg, "ezfyBotConquerFeelingsCfg") {
		t.Fatal("征服扣民心仍在读人机专用上限 ezfyBotConquerFeelingsCfg")
	}

	// ② 人机专用配置彻底删除
	geo := rawFile(t, "ezfy_geo.go")
	if strings.Contains(geo, "ezfyBotConquerFeelingsDef") || strings.Contains(geo, "func ezfyBotConquerFeelingsCfg(") {
		t.Fatal("ezfy_geo.go 里人机专用扣民心配置没删干净")
	}
	model := rawFile(t, "../../model/ezfy.go")
	if strings.Contains(model, "BotConquerFeelingsMax") {
		t.Fatal("model.EzfyCfgLimit 仍有 BotConquerFeelingsMax 字段（应已取消）")
	}
	round9 := rawFile(t, "ezfy_admin_round9.go")
	if strings.Contains(round9, "BotConquerFeelingsMax") || strings.Contains(round9, "bot_conquer_feelings_max") {
		t.Fatal("二战系统配置接口仍收/发 bot_conquer_feelings_max")
	}
}
