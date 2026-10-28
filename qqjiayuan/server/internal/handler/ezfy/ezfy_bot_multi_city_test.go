package ezfy

import (
	"strings"
	"testing"
)

// TestBotMultiCity 人机多城市护栏（用户 2026-10-28）：
//
//	「线上人机被打了以后城市被人占领，然后又被归还，那么人机是可以玩两个城市的，
//	 现在人机只能玩一个城市 —— bug 修复；人机能玩多个城市，而且不用有主城市概念，
//	 所有平等的，都可以玩、都可以出兵攻击。」
//
// 断言（缺一条就回退成「只玩一座城」）：
//  1. botTickOnce 不再用 getOrCreateCity 取「唯一那座城」，而是取**城市行动槽**；
//  2. 每座城各自决策（botActCity 被 tick 与管理端手动触发共用）；
//  3. 槽表 ezfy_bot_city 已登记建表 + 删号清单（漏登记 = 线上建不出表 / 删号残留）；
//  4. 出征节流仍是**账号级**（决策前重读 ezfy_bot，多城不会各打一次把攻频打穿）；
//  5. 被占领中的城不能出兵/发育。
func TestBotMultiCity(t *testing.T) {
	bot := rawFile(t, "ezfy_bot.go")

	// ① tick 走「城市行动槽」
	tick := ezfyFuncBody(t, "ezfy_bot.go", "func (h *EzfyHandler) botTickOnce(")
	if strings.Contains(tick, "getOrCreateCity") {
		t.Fatal("botTickOnce 仍在取「当前城」（getOrCreateCity）→ 只会玩一座城")
	}
	if !strings.Contains(tick, "[]model.EzfyBotCity") {
		t.Fatal("botTickOnce 没有从 ezfy_bot_city 取行动槽")
	}
	if !strings.Contains(tick, "h.botTickCitySlot(") {
		t.Fatal("botTickOnce 没有逐城执行（botTickCitySlot）")
	}

	// ② 每座城各自决策，且与「管理端立即行动」共用同一套
	slot := ezfyFuncBody(t, "ezfy_bot.go", "func (h *EzfyHandler) botTickCitySlot(")
	if !strings.Contains(slot, "h.botActCity(") {
		t.Fatal("botTickCitySlot 没有调用 botActCity（逐城决策）")
	}
	one := ezfyFuncBody(t, "ezfy_bot.go", "func (h *EzfyHandler) botTickOne(")
	if !strings.Contains(one, "h.botCities(") {
		t.Fatal("botTickOne（管理端立即行动）没有遍历全部城市")
	}
	if !strings.Contains(one, "h.botActCity(") {
		t.Fatal("botTickOne 没有逐城决策")
	}

	// ③ 槽表登记：建表 + 删号
	run := rawFile(t, "../../seed/ezfy_run.go")
	if !strings.Contains(run, "&model.EzfyBotCity{}") {
		t.Fatal("ezfy_bot_city 没登记到 ezfyMigrateModels（线上不会建表）")
	}
	purge := rawFile(t, "ezfy_admin_purge.go")
	if !strings.Contains(purge, `{"ezfy_bot_city", []string{"bot_uid"}}`) {
		t.Fatal("ezfy_bot_city 没登记到删号清单（删人机后行动槽残留）")
	}

	// ④ 账号级出征节流：决策前重读 ezfy_bot
	act := ezfyFuncBody(t, "ezfy_bot.go", "func (h *EzfyHandler) botActCity(")
	if !strings.Contains(act, "h.DB.First(&fresh, b.ID)") {
		t.Fatal("botActCity 决策前没有重读 ezfy_bot → 同轮多城各用旧快照，攻频会被打穿")
	}

	// ⑤ 被占领的城不出兵
	if !strings.Contains(slot, "h.cityOccupied(") {
		t.Fatal("botTickCitySlot 没有跳过「正被玩家占领」的城（会从被占城派兵）")
	}
	cities := ezfyFuncBody(t, "ezfy_bot.go", "func (h *EzfyHandler) botCities(")
	if !strings.Contains(cities, "ezfy_occupy") {
		t.Fatal("botCities 没有排除被占领中的城")
	}

	// ⑥ 自动发言：新建人机默认关（用户规则）
	model := rawFile(t, "../../model/ezfy_bot.go")
	if !strings.Contains(model, `ChatOn int `+"`"+`gorm:"default:0;`) {
		t.Fatal("ezfy_bot.chat_on 默认值不是 0（新建人机应默认不自动发言）")
	}
	if !strings.Contains(bot, "ChatOn: 0,") {
		t.Fatal("ezfyBotProvision 没有显式把新建人机的 chat_on 落成 0")
	}
}
