package ezfy

import (
	"strings"
	"testing"
)

// TestBotWildAttack 人机打「玩家附属野地」的口径（用户 2026-10-19）：
//
//	「人机若攻击不了玩家，那么应该攻击玩家的附属野地，而且是征服，
//	 胜利后就放弃拿到的附属野地；现在人机在攻击没有归属的附属野地不对」
//
// 断言四件事：
//  1. 目标顺序：先找玩家城（botPickTarget），失败才打野地 —— 不再是「25% 概率随机掠野」；
//  2. 只打**有主**野地（JOIN ezfy_city + user_id > 0 + is_bot=0）；
//  3. 打野地必须用**征服(3)**，不是掠夺(2)；
//  4. 征服成功后，攻方是人机 → 立刻放弃（只摧毁归属，不占领）。
func TestBotWildAttack(t *testing.T) {
	bot := rawFile(t, "ezfy_bot.go")

	// ① 顺序：botPickTarget 必须在 botPickWildTarget 之前
	iCity := strings.Index(bot, "if tgt, ok := h.botPickTarget(city, troopTotal, cfg); ok {")
	iWild := strings.Index(bot, "if wt, ok := h.botPickWildTarget(city); ok {")
	if iCity < 0 || iWild < 0 {
		t.Fatal("botMilitary 里找不到 botPickTarget / botPickWildTarget")
	}
	if iCity > iWild {
		t.Fatal("人机仍然优先打野地 —— 应先找玩家城，打不了才打野地")
	}
	if strings.Contains(bot, "mathrand.Intn(100) < 25") {
		t.Fatal("「25% 概率随机掠野」还在（会随机打中立野地）")
	}

	// ② 只打有主野地
	if !strings.Contains(bot, "c.is_bot = 0 AND c.user_id > 0 AND c.user_id <> ?") {
		t.Fatal("botPickWildTarget 未排除「无主/脏数据」野地")
	}

	// ③ 征服(3) 而不是掠夺(2)
	if !strings.Contains(bot, `"order_type": 3, "target_type": 1`) {
		t.Fatal("人机打野地仍是掠夺(2)，应为征服(3)")
	}

	// ④ 征服成功后，人机立刻放弃
	order := rawFile(t, "ezfy_order.go")
	if !strings.Contains(order, "if city.IsBot == 1 {\n\t\t\t\th.ezfyClearWildlandAt(order.TargetX, order.TargetY, now)") {
		t.Fatal("人机征服野地后没有放弃（应只摧毁归属、不占领）")
	}
	// 人机打野地要自动结算（没有指挥能力，不能卡在指挥室）
	if !strings.Contains(order, "if city.IsBot == 1 && order.TargetType != 3 {") {
		t.Fatal("人机打野地仍会被强制进指挥室（人机无法指挥，会干等）")
	}
}
