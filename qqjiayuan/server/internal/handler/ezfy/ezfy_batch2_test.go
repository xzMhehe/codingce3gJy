package ezfy

import (
	"strings"
	"testing"
)

// TestPvPDefenderWounded PvP 守方战损也要进伤兵营，且回收率吃「守方科技 + 城守军官技能」。
//
// ★★ 2026-10-09 用户要求「pvp 伤兵也是入伤兵营、回收比例也要有科技加成、军官技能」：
//
//	原来守方战损只扣兵（仅攻方获胜时进逃兵营），打一场 PvP 守方掉几万兵一个都回不来。
func TestPvPDefenderWounded(t *testing.T) {
	src := strings.ReplaceAll(rawFile(t, "ezfy_order.go"), "\r\n", "\n")
	for _, want := range []string{
		"defHealTech = defTech[21] * 2",                          // 守方科技·治愈伤兵
		`h.officerHasSkill(cityGuard, "机械改造")`,                   // 城守军官技能
		"h.addWounded(target.ID, g.TroopId, 0, wounded)",         // 守方伤兵入营（type=0 伤兵）
		`"守方", defRepairedTotal, br.DefenderLosses, defHealTech`, // 战报里也写一行
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("PvP 守方伤兵入营未接入（缺 %s）", want)
		}
	}
}

// TestNewCityRules 起新城：坐标必须是**自己的附属野地**（平原/沿海平原）+ 五项资源各 10 万。
//
// ★★ 2026-10-09 用户要求「坐标必须是自己的附属野地（平原、沿海平原），而且资源是各 10 万」。
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

// TestConveneCooldown3s 人口召集卡控 3 秒（原来 5 秒，前后端要一致）。
func TestConveneCooldown3s(t *testing.T) {
	src := strings.ReplaceAll(rawFile(t, "ezfy.go"), "\r\n", "\n")
	if strings.Contains(src, "nowCd-last < 5000") {
		t.Fatal("召集卡控还是 5 秒")
	}
	if !strings.Contains(src, "nowCd-last < 3000") {
		t.Fatal("召集卡控不是 3 秒")
	}
	if !strings.Contains(src, "请 3 秒后再试") {
		t.Fatal("召集卡控提示文案没跟着改成 3 秒")
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
