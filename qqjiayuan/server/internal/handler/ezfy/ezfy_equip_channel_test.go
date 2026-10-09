package ezfy

import (
	"os"
	"strings"
	"testing"
)

// ============ 一、军官攻防换算（复刻《战斗机制》§6） ============

// TestEzfyAttrToBonus —— 用户反馈（2026-09-22）：「军官防御加成不对，防御是和学识相关」。
//
// 参考材料《战斗机制（家园玩家必看）》§6：
//
//	攻击加成百分点 = floor(有效军事 + 1) ÷ 2
//	防御加成百分点 = floor(有效学识 + 1) ÷ 2
//
// 即每 2 点属性 = 1 个百分点。用例里的 655 / 376 就是用户报的那名军官的真实属性。
func TestEzfyAttrToBonus(t *testing.T) {
	cases := []struct{ attr, want int }{
		{0, 0},
		{-5, 0},    // 异常入参不得产生负加成
		{1, 1},     // (1+1)/2
		{2, 1},     // (2+1)/2 = 1（向下取整）
		{3, 2},     // (3+1)/2
		{200, 100}, // 文档示例：有效学识 200 → 约 100% 防御加成
		{300, 150}, // 文档示例：有效军事 300 → 约 150% 攻击加成
		{376, 188}, // 用户报的军官：学识 376 → 防御+188
		{655, 328}, // 用户报的军官：军事 655 → 攻击+328
	}
	for _, c := range cases {
		if got := ezfyAttrToBonus(c.attr); got != c.want {
			t.Fatalf("ezfyAttrToBonus(%d) = %d, 期望 %d", c.attr, got, c.want)
		}
	}
}

// TestOfficerBonusNoLongerOneToOne —— 防回归：军事/学识**不能**再 1:1 当百分点。
//
// 老实现是 `o.Military + 装备`（军事 655 → 攻击+655%）和 `10 + 学识/20`（学识 376 → 防御+58），
// 前者是文档值的 2 倍、后者只有文档值的 1/3，两个方向都错。
func TestOfficerBonusNoLongerOneToOne(t *testing.T) {
	if ezfyAttrToBonus(655) == 655 {
		t.Fatal("攻击加成仍是 1:1（军事直接当百分点），未按《战斗机制》§6 除以 2")
	}
	if ezfyAttrToBonus(376) == 58 {
		t.Fatal("防御加成仍是老的 10 + 学识/20 口径，未按《战斗机制》§6 重写")
	}
}

// TestOfficerGuardBonusSourceUsesAttrBonus —— 静态源码断言：
// officerGuardBonus 必须走 officerGuardAttrBonus（属性换算），不能再出现 `lea/20` 或固定 +10。
func TestOfficerGuardBonusSourceUsesAttrBonus(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_officer.go", "func (h *EzfyHandler) officerGuardBonus(")
	if !strings.Contains(body, "officerGuardAttrBonus") {
		t.Fatalf("officerGuardBonus 未使用属性换算函数，实际实现：\n%s", body)
	}
	if strings.Contains(body, "lea/20") {
		t.Fatalf("officerGuardBonus 里仍有老的 `lea/20` 口径：\n%s", body)
	}
	if strings.Contains(body, "10 + lea") {
		t.Fatalf("officerGuardBonus 里仍有老的固定 +10 基础值：\n%s", body)
	}
}

// ============ 二、套装装备渠道（只能开宝箱） ============

// TestBattleLootExcludesSetPieces —— 用户规则：「套装军官装备不能通过战斗掉落获得」。
//
// ★ 2026-10-09 掉落改走 `wildlandConfigLoot`（「地图管理 → 野地类型」手动配置的宝物/道具，
// 掠夺与征服都会掉）后，这里改成盯它：**配置里填了套装名也必须被拦掉** ——
// 否则管理端填个套装名就能绕过「套装只能开宝箱」这条规则刷套装。
func TestBattleLootExcludesSetPieces(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) wildlandConfigLoot(")
	if !strings.Contains(body, "eq.SetId > 0") {
		t.Fatalf("wildlandConfigLoot 未排除套装件，野地掉落仍会掉套装：\n%s", body)
	}
}

// TestWildlandLootAnnouncesDrops —— 用户要求（2026-10-09）：
// 「打野地 掉落宝物 也要播报 系统消息」+「商城道具掉落 也播报」。
//
// 原来「掉落宝物 / 掉落道具」只写进战报（玩家自己看得到），系统频道没有任何播报；
// 采集到宝物却是有播报的（见 ezfy_order.go 采集分支）—— 这里把野地/寇城战斗掉落补齐。
//
// 静态断言四件事：
//  1. 掉到东西后确实往系统频道写（ezfySysChat）；
//  2. **两类掉落都进同一条**播报（lootNames 被 append 两次：道具 + 宝物），
//     一场战斗只刷一条，且不会漏掉商城道具；
//  3. 接收 targetName（调用方传入「海底森林2级」/「寇城3级」），播报文案与战报标题同源；
//  4. 两个掉落解析函数都还在（防止为了加播报把掉落逻辑改坏）。
func TestWildlandLootAnnouncesDrops(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) wildlandConfigLoot(")
	if !strings.Contains(body, "ezfySysChat") {
		t.Fatalf("wildlandConfigLoot 掉落物品后没有系统频道播报：\n%s", body)
	}
	if n := strings.Count(body, "lootNames = append("); n != 2 {
		t.Fatalf("wildlandConfigLoot 应把「商城道具 + 宝物」都攒进同一条播报（期望 2 处 append，实际 %d）：\n%s", n, body)
	}
	if !strings.Contains(body, "targetName string") {
		t.Fatalf("wildlandConfigLoot 未接收 targetName，播报文案无法与战报标题同源：\n%s", body)
	}
	for _, fn := range []string{"parseWildlandItemDrops(", "parseWildTreasureDrops("} {
		if !strings.Contains(body, fn) {
			t.Fatalf("wildlandConfigLoot 里找不到掉落解析函数 %s（掉落逻辑被改坏了）：\n%s", fn, body)
		}
	}
}

// TestEquipShopOnlySellsLoosePieces —— 商城只卖**散件**。
//
// ★ 2026-09-22 用户定稿：「散件也上吧，价格按加成 10 钻石到 50 钻石不等」
// —— 六大系列的单件（可单穿）与纯散件都能买；
// ★ 2026-10-06 用户定稿：套装装备**全部下架**（set_id > 0 一律不售，含系列套装），
// 套装件只能开宝箱，「套装装备只能通过宝箱开启」这条规则对所有套装生效。
//
// 列表侧与购买侧都要拦（防前端被绕过直接 POST cfg_id）。
func TestEquipShopOnlySellsLoosePieces(t *testing.T) {
	list := ezfyFuncBody(t, "ezfy_officer.go", "func (h *EzfyHandler) equipShopList(")
	if !strings.Contains(list, `e.SetId > 0`) {
		t.Fatalf("equipShopList 未拦住套装件（set_id > 0 全部下架）：\n%s", list)
	}
	if !strings.Contains(list, `"slot"`) {
		t.Fatalf("equipShopList 应按**部位**分组下发：\n%s", list)
	}

	buy := ezfyFuncBody(t, "ezfy_officer.go", "func (h *EzfyHandler) EquipShopBuy(")
	if !strings.Contains(buy, `cfg.SetId > 0`) {
		t.Fatalf("EquipShopBuy 未拦截套装件购买：\n%s", buy)
	}
}

// ============ 三、辅助 ============

// ezfyFuncBody 取某个函数从签名到首个「行首 }」之间的源码（用于静态断言）。
func ezfyFuncBody(t *testing.T, file, sig string) string {
	t.Helper()
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("读源码失败 %s: %v", file, err)
	}
	src := string(b)
	i := strings.Index(src, sig)
	if i < 0 {
		t.Fatalf("源码 %s 里找不到函数签名 %q", file, sig)
	}
	rest := src[i:]
	if j := strings.Index(rest, "\n}"); j >= 0 {
		return rest[:j]
	}
	return rest
}
