package ezfy

import (
	"strings"
	"testing"
)

// TestBotPeaceFiltersAllPaths 人机保护名单（★ 2026-10-30 用户规则「输入游戏ID → 人机不攻击这个玩家城市」）：
// 四条路径都要拦，少一条玩家就还是会被打：
//  1. 自动挑目标（botPickTarget）—— SQL 里排除名单
//  2. 自动挑野地（botPickWildTarget）—— SQL 里排除名单
//  3. 指定玩家野地（botPickWildTargetOf，宣战等待期）/ 手动指定目标 —— ezfyBotIsProtected 兜底
//  4. 被打反击（botRevenge）—— 名单里的玩家连反击计划都不写
func TestBotPeaceFiltersAllPaths(t *testing.T) {
	src := strings.ReplaceAll(rawFile(t, "ezfy_bot.go"), "\r\n", "\n")

	if pick := funcBody(src, "func (h *EzfyHandler) botPickTarget("); !strings.Contains(pick, "NOT IN (SELECT user_id FROM ezfy_bot_peace)") {
		t.Fatal("botPickTarget 没有排除人机保护名单（玩家城还会被自动选中）")
	}
	if wild := funcBody(src, "func (h *EzfyHandler) botPickWildTarget("); !strings.Contains(wild, "NOT IN (SELECT user_id FROM ezfy_bot_peace)") {
		t.Fatal("botPickWildTarget 没有排除人机保护名单（玩家野地还会被打）")
	}
	if body := funcBody(src, "func (h *EzfyHandler) botPickWildTargetOf("); !strings.Contains(body, "h.ezfyBotIsProtected(userID)") {
		t.Fatal("botPickWildTargetOf 没有校验保护名单（宣战等待期仍会打名单玩家的野地）")
	}
	if body := funcBody(src, "func (h *EzfyHandler) botAttack("); !strings.Contains(body, "h.ezfyBotIsProtected(p.TargetUid)") {
		t.Fatal("botAttack 没有校验保护名单（管理端手动指定的目标仍会打）")
	}
	if body := funcBody(src, "func (h *EzfyHandler) botRevenge("); !strings.Contains(body, "h.ezfyBotIsProtected(hit.UID)") {
		t.Fatal("botRevenge 没有校验保护名单（名单里的玩家打人机会被反击）")
	}
}
