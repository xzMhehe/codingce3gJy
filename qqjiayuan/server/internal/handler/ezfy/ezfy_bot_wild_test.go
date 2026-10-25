package ezfy

import (
	"strings"
	"testing"
)

// TestBotWildAttack 人机打「玩家附属野地」的口径（用户 2026-10-19）：
//
//	「打玩家城需要宣战、宣战有等待期；等待期里打不了城，但可以征服玩家的附属野地，
//	 而且是征服，胜利后就放弃拿到的附属野地；人机不该打没有归属的野地」
//
// 断言：
//  1. 宣战等待期（status=1）→ 先征服**该目标玩家**的附属野地（botPickWildTargetOf）；
//  2. 找玩家城失败 → 兜底打野地（botPickWildTarget），不再是「25% 概率随机掠野」；
//  3. 只打**有主**野地（JOIN ezfy_city + user_id > 0 + is_bot=0）；
//  4. 打野地必须用**征服(3)**，不是掠夺(2)；
//  5. 征服成功后，攻方是人机 → 立刻放弃（只摧毁归属，不占领）。
func TestBotWildAttack(t *testing.T) {
	bot := rawFile(t, "ezfy_bot.go")

	// ① 打野地时机（★ 2026-10-19 用户澄清结构）：
	//   · **宣战等待期（status=1）**→ 打该目标玩家的附属野地（botPickWildTargetOf，
	//     野地不需要宣战；在 botPickTarget 之前，计划推进优先）；
	//   · 找玩家城失败 → 兜底打野地（botPickWildTarget，在 botPickTarget 之后）。
	iCity := strings.Index(bot, "if tgt, ok := h.botPickTarget(city, troopTotal, cfg); ok {")
	if iCity < 0 {
		t.Fatal("botMilitary 里找不到 botPickTarget")
	}
	// 宣战等待期打目标玩家野地：必须在 botPickTarget 之前（计划推进在前）
	iWildWait := strings.Index(bot, "if wt, ok := h.botPickWildTargetOf(p.TargetUid, city); ok {")
	if iWildWait < 0 {
		t.Fatal("宣战等待期（status=1）没有打目标玩家附属野地的逻辑")
	}
	if iWildWait > iCity {
		t.Fatal("「宣战等待期打野地」应放在 botPickTarget 之前（计划推进优先）")
	}
	// 找玩家城失败后的兜底打野地：必须在 botPickTarget 之后
	iWildFall := strings.LastIndex(bot, "if wt, ok := h.botPickWildTarget(city); ok {")
	if iWildFall < iCity {
		t.Fatal("「找玩家城失败后打野地」的兜底应放在 botPickTarget 之后")
	}
	if strings.Contains(bot, "mathrand.Intn(100) < 25") {
		t.Fatal("「25% 概率随机掠野」还在（会随机打中立野地）")
	}

	// ② 只打有主野地（通用挑法排除无主/脏数据；指定玩家的挑法限定归属本人）
	if !strings.Contains(bot, "c.is_bot = 0 AND c.user_id > 0 AND c.user_id <> ?") {
		t.Fatal("botPickWildTarget 未排除「无主/脏数据」野地")
	}
	if !strings.Contains(bot, "c.is_bot = 0 AND c.user_id = ?") {
		t.Fatal("botPickWildTargetOf 未限定为指定玩家的野地")
	}

	// ③ 征服(3) 而不是掠夺(2)
	if !strings.Contains(bot, `"order_type": 3, "target_type": 1`) {
		t.Fatal("人机打野地仍是掠夺(2)，应为征服(3)")
	}

	// ④ 征服成功后，人机立刻放弃（只摧毁归属、不占领）—— 去空白后匹配，避免缩进变化误报
	order := rawFile(t, "ezfy_order.go")
	norm := strings.Map(func(r rune) rune {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			return -1
		}
		return r
	}, order)
	if !strings.Contains(norm, "ifcity.IsBot==1{h.ezfyClearWildlandAt(order.TargetX,order.TargetY,now)") {
		t.Fatal("人机征服野地后没有放弃（应只摧毁归属、不占领）")
	}
	// 人机打野地要自动结算（没有指挥能力，不能卡在指挥室）
	if !strings.Contains(order, "if city.IsBot == 1 && order.TargetType != 3 {") {
		t.Fatal("人机打野地仍会被强制进指挥室（人机无法指挥，会干等）")
	}

	// ⑤ 战报不再追加机制说明「(人机只摧毁野地归属, 不占领)」—— 用户要求去掉（冗余提示）
	if strings.Contains(order, `report += "\n(人机只摧毁野地归属`) {
		t.Fatal("战报里仍有冗余提示「人机只摧毁野地归属, 不占领」，应删除")
	}

	// ⑥ ★ 2026-10-25 打野地开关（用户：管理端可编辑，**默认不打**玩家野地）
	//   两处打野地分支都必须被 `b.AttackWild == 1` 守卫 —— 漏一处就等于「默认也在打野地」。
	//   去空白后匹配，避免缩进变化误报。
	botNorm := strings.Map(func(r rune) rune {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			return -1
		}
		return r
	}, bot)
	for _, frag := range []string{
		"ifb.AttackWild==1{ifwt,ok:=h.botPickWildTargetOf(p.TargetUid,city);ok{",
		"ifb.AttackWild==1{ifwt,ok:=h.botPickWildTarget(city);ok{",
	} {
		if !strings.Contains(botNorm, frag) {
			t.Fatalf("打野地分支缺少 b.AttackWild == 1 守卫（默认不打玩家野地）：%s", frag)
		}
	}
	// 管理端编辑接口必须能改这个开关（否则运营无处可调）
	adm := rawFile(t, "ezfy_admin_bot.go")
	if !strings.Contains(adm, "AttackWild *int") || !strings.Contains(adm, `ezUpd["attack_wild"]`) {
		t.Fatal("人机编辑接口没有 attack_wild 开关（打野地不可编辑）")
	}
	// 列表要下发原值，编辑弹窗才能回填真实状态
	if !strings.Contains(adm, `"attack_wild": b.AttackWild`) {
		t.Fatal("人机列表没有下发 attack_wild（编辑弹窗回填恒为关）")
	}
}
