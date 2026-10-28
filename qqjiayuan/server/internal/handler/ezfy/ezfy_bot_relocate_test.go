package ezfy

import (
	"strings"
	"testing"
)

// TestBotRelocateToTarget 人机「迁城打目标城市」（用户 2026-10-28：「人机AI 不会迁城打目标城市了」）。
//
// 两个根因：
//  1. 迁城只写在 botMilitary 里，而 botMilitary 排在召集/建筑/征兵/军官之后 →
//     人机只要还有东西可建可造就永远轮不到迁城（宣战等待的 24 小时里也不挪窝）；
//  2. 落点算法 radius 3~6 每圈**随机只试 20 个点**、每点一条 SQL →
//     线上以真实城坐标为目标实测 **12.5% 找不到落点**，人机静默放弃迁城、几十格外远程出征。
//
// 断言（缺一条就会回退成「不迁城」）：
//  1. botDecide 里「先迁城贴近」排在建筑/征兵之前（0¾ 步）；
//  2. 落点算法：一次 SQL 取占用 + 整圈全扫（不再随机抽 20 个）+ 半径到 12 + 海城优先沿海平原；
//  3. 迁城失败要写行为日志（运营能看出为什么没迁）；
//  4. 多城市：只让离目标最近的那座城去迁，不把每座城都搬到同一战区。
func TestBotRelocateToTarget(t *testing.T) {
	// ① 0¾ 站位：必须在建筑（botBuild）与征兵（botTrain 常态调用）之前
	decide := ezfyFuncBody(t, "ezfy_bot.go", "func (h *EzfyHandler) botDecide(")
	if decide == "" {
		t.Fatal("未找到 botDecide")
	}
	iReloc := strings.Index(decide, "h.botRelocateForPlan(")
	if iReloc < 0 {
		t.Fatal("botDecide 里没有「先迁城贴近目标」这一步")
	}
	iBuild := strings.Index(decide, "h.botBuild(")
	if iBuild < 0 || iReloc > iBuild {
		t.Fatal("「先迁城贴近目标」必须排在建筑/征兵之前（否则永远轮不到迁城）")
	}

	// ② 落点算法：整圈全扫 + 一次 SQL + 半径 12 + 海城
	pick := ezfyFuncBody(t, "ezfy_bot.go", "func (h *EzfyHandler) ezfyBotPickRelocateSpot(")
	if pick == "" {
		t.Fatal("未找到 ezfyBotPickRelocateSpot")
	}
	if strings.Contains(pick, "checked > 20") {
		t.Fatal("落点算法还在「每圈随机只试 20 个点」（线上 12.5% 找不到落点）")
	}
	if !strings.Contains(pick, "x BETWEEN ? AND ? AND y BETWEEN ? AND ?") {
		t.Fatal("落点算法没有一次性取附近占用（会逐点打库）")
	}
	if !strings.Contains(pick, "const maxR = ezfyBotRelocateDist") {
		t.Fatal("落点搜索半径没有跟 ezfyBotRelocateDist 对齐")
	}
	if !strings.Contains(pick, "ezfyAbs(dx)+ezfyAbs(dy) > ezfyBotRelocateDist") {
		t.Fatal("落点没有限制在「迁完就不用再迁」的范围内（会无限迁城）")
	}
	if !strings.Contains(pick, "ezfyTerrainCoastalPlain") {
		t.Fatal("落点算法没有处理海城（海城应优先沿海平原，否则会丢海城身份）")
	}

	// ③ 失败留痕 + ④ 多城市只迁最近的一座
	reloc := ezfyFuncBody(t, "ezfy_bot.go", "func (h *EzfyHandler) botRelocateNear(")
	if reloc == "" {
		t.Fatal("未找到 botRelocateNear")
	}
	if !strings.Contains(reloc, "迁城失败") {
		t.Fatal("迁城找不到落点时没有写行为日志（运营看不出为什么没迁）")
	}
	if !strings.Contains(reloc, "h.botCityNearestTo(") {
		t.Fatal("多城市下没有「只让离目标最近的那座城去迁」的约束")
	}
	if !strings.Contains(reloc, "h.isSeaCity(city)") {
		t.Fatal("迁城没有按城市类型（海城/陆城）选落点")
	}
}
