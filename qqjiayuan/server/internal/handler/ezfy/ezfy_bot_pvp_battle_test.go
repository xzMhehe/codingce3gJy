package ezfy

import (
	"strings"
	"testing"
)

// TestBotPvpBattle 人机与玩家城市战斗的指挥口径（用户 2026-10-20）：
//
//	「现在打人机城市玩家不能指挥了 bug 修复；人机打玩家城市也要进入走指挥模块逻辑；
//	 人机也是真人啊，按真人算的，只不过是机器操控人的帐号。」
//
// 统一口径：玩家城市（含人机城）之间的战斗一律进指挥室 ——
//   - 人机由后台 botBattleCmds 每 15 秒逐回合指挥，等价于一直在线，
//     不能再按 users.last_active_at 判离线触发「双方都离线 → 自动结算」；
//   - 人机与玩家各用各的逐兵种指令，不再镜像玩家指令。
func TestBotPvpBattle(t *testing.T) {
	order := rawFile(t, "ezfy_order.go")
	room := rawFile(t, "ezfy_battle_room.go")

	// ① 人机视为在线（否则玩家打人机城到点时会被自动打完，进不了指挥室）
	if !strings.Contains(room, "func (h *EzfyHandler) ezfyBattleSideOnline(uid uint) bool {") {
		t.Fatal("缺少 ezfyBattleSideOnline（人机战斗不能按 last_active_at 判离线）")
	}
	if !strings.Contains(room, "if h.ezfyBotIsBot(uid) {\n\t\treturn true\n\t}") {
		t.Fatal("ezfyBattleSideOnline 未把人机视为在线")
	}

	// ② 到达结算用新口径判攻守双方在线：
	//    玩家打人机城（守方人机在线） / 人机打玩家城（攻方人机在线）都进指挥室。
	if !strings.Contains(order, "if !autoBattle && !h.ezfyBattleSideOnline(uid) {") {
		t.Fatal("processArrive 仍在用 ezfyUserOnline 判攻方离线（打人机城会被自动结算）")
	}
	if !strings.Contains(order, "defOnline = target != nil && target.UserID > 0 && h.ezfyBattleSideOnline(target.UserID)") {
		t.Fatal("processArrive 未用 ezfyBattleSideOnline 判守方（人机守城会被判离线）")
	}

	// ③ 玩家城市仍强制进指挥室（不受 auto_battle 字段影响）
	if !strings.Contains(order, "if order.TargetType == 3 || hasPlayerDefender {\n\t\t\tautoBattle = false\n\t\t}") {
		t.Fatal("玩家城/有守军野地未强制进指挥室")
	}

	// ④ 人机侧由自己的 botBattleCmds 指挥，不再镜像玩家指令
	if strings.Contains(room, "cmds = defCmds") || strings.Contains(room, "defCmds = cmds") {
		t.Fatal("指挥引擎仍在镜像玩家指令（人机应按自己的 botBattleCmds 指挥）")
	}
	if !strings.Contains(room, "h.botBattleCmds()") {
		t.Fatal("后台未给人机下达战场指令（人机侧无法指挥）")
	}
	if !strings.Contains(room, "field = \"def_cmd\"") {
		t.Fatal("botBattleCmds 未覆盖人机守城（def_cmd）")
	}
}
