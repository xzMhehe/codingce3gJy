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

// TestWildlandLootAnnouncesDrops —— （2026-10-09）：
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

// TestEquipShopSellsOfficerEquip —— 商城在售 = 有价格的「军官装备」（含套装件/系列件）。
//
// ★ 2026-09-22 用户定稿：「散件也上吧，价格按加成 10 钻石到 50 钻石不等」；
// ★ 2026-10-06 用户定稿：套装装备**全部下架**（set_id > 0 一律不售，含系列套装）；
// ★★ 2026-10-14 用户定稿：「之前下的装备现在上架吧」→ **套装件（1~17）与系列件（21~26）重新上架**，
// 商城不再按 set_id 拦。所以现在**不能**再断言「set_id > 0 被拦住」，
// 而要断住「仍然只上架军官装备 + 有价格」这两条底线（老版武器/防具/珠宝别混进来）。
//
// 另：套装件/系列件仍**不会**从野地战斗掉落（见 TestBattleLootExcludesSetPieces）。
func TestEquipShopSellsOfficerEquip(t *testing.T) {
	list := ezfyFuncBody(t, "ezfy_officer.go", "func (h *EzfyHandler) equipShopList(")
	if strings.Contains(list, "e.SetId > 0") || strings.Contains(list, "e.SetId>0") {
		t.Fatalf("equipShopList 仍在拦套装件（2026-10-14 已重新上架）：\n%s", list)
	}
	if !strings.Contains(list, `e.Type != "军官装备" && e.Type != "套装"`) {
		t.Fatalf("equipShopList 未限制「只上架军官装备/套装」（老版武器/防具/珠宝会混进来）：\n%s", list)
	}
	if !strings.Contains(list, "e.PriceGold <= 0 && e.PriceDiamond <= 0") {
		t.Fatalf("equipShopList 未要求「有价格」，未定价装备（价 0）会上架：\n%s", list)
	}
	if !strings.Contains(list, `"slot"`) {
		t.Fatalf("equipShopList 应按**部位**分组下发：\n%s", list)
	}

	buy := ezfyFuncBody(t, "ezfy_officer.go", "func (h *EzfyHandler) EquipShopBuy(")
	if strings.Contains(buy, "cfg.SetId > 0") || strings.Contains(buy, "cfg.SetId>0") {
		t.Fatalf("EquipShopBuy 仍在拦套装件购买（2026-10-14 已重新上架）：\n%s", buy)
	}
}

// ============ 二·补、打野地掉宝冷却 + 宝物掉落配置校验（2026-10-11） ============

// TestWildlandTreasureCooldown —— （2026-10-11）：
// 「打野地掉落宝物有点频繁了，即使我配置了 1%，如果玩家 2 小时内掉落了宝物那么就不会再掉」。
//
// 静态断言：
//  1. wildlandConfigLoot 读「二战系统配置」的冷却时长（ezfyWildTreasureCooldownH）并做冷却判定
//     （ezfyWildTreasureReady）；
//  2. 掉到宝物后写 ezfy_profile.wild_treasure_at（冷却从此刻起算）；
//  3. 冷却判定函数存在且读 wild_treasure_at。
func TestWildlandTreasureCooldown(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) wildlandConfigLoot(")
	if !strings.Contains(body, "ezfyWildTreasureCooldownH()") {
		t.Fatalf("wildlandConfigLoot 未读取打野地掉宝冷却配置（二战系统配置）：\n%s", body)
	}
	if !strings.Contains(body, "ezfyWildTreasureReady(") {
		t.Fatalf("wildlandConfigLoot 未做掉宝冷却判定，宝物仍会频繁掉：\n%s", body)
	}
	if !strings.Contains(body, `Update("wild_treasure_at"`) {
		t.Fatalf("wildlandConfigLoot 掉到宝物后未刷新冷却时间戳（wild_treasure_at）：\n%s", body)
	}
	helper := ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) ezfyWildTreasureReady(")
	if !strings.Contains(helper, "wild_treasure_at") {
		t.Fatalf("ezfyWildTreasureReady 未按 wild_treasure_at 判定冷却：\n%s", helper)
	}
}

// TestWildlandItemDropCooldown —— （2026-10-13）：
// 「野地商城道具掉落也加个限制，有人用脚本刷野地，掉落过了就不能再次掉落，
// 二战系统配置可配置间隔时间」。
//
// 静态断言：
//  1. wildlandConfigLoot 读「二战系统配置」的掉道具冷却时长（★ 2026-10-26 单位改秒：
//     ezfyWildItemCooldownSec）并做冷却判定（ezfyWildItemReady）；
//  2. 掉到商城道具后写 ezfy_profile.wild_item_at（冷却从此刻起算）；
//  3. 冷却判定函数存在且读 wild_item_at。
//
// ★ 与宝物冷却各自独立：道具写 wild_item_at、宝物写 wild_treasure_at，互不影响。
func TestWildlandItemDropCooldown(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) wildlandConfigLoot(")
	if !strings.Contains(body, "ezfyWildItemCooldownSec()") {
		t.Fatalf("wildlandConfigLoot 未读取打野地掉道具冷却配置（二战系统配置）：\n%s", body)
	}
	if !strings.Contains(body, "ezfyWildItemReady(") {
		t.Fatalf("wildlandConfigLoot 未做掉道具冷却判定，道具仍会被脚本刷：\n%s", body)
	}
	if !strings.Contains(body, `Update("wild_item_at"`) {
		t.Fatalf("wildlandConfigLoot 掉到商城道具后未刷新冷却时间戳（wild_item_at）：\n%s", body)
	}
	helper := ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) ezfyWildItemReady(")
	if !strings.Contains(helper, "wild_item_at") {
		t.Fatalf("ezfyWildItemReady 未按 wild_item_at 判定冷却：\n%s", helper)
	}
}

// TestWildTreasureCfgRejectsSetPieces —— 管理端「地图管理 → 野地类型 → 宝物掉落」保存校验：
//
// 除了「名字查不到（含已删的老珠宝）」要拒，**套装件**（只能开宝箱）也要拒 —— 与掉落逻辑
// （wildlandConfigLoot 里 `eq.SetId > 0` 跳过）保持一致，避免「配了却永远不掉」。
func TestWildTreasureCfgRejectsSetPieces(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_admin_wild.go", "func (h *EzfyAdmin) ezfyCheckWildTreasure(")
	if !strings.Contains(body, "SetId > 0") {
		t.Fatalf("ezfyCheckWildTreasure 未拒绝套装件，管理端可把套装配成野地掉落：\n%s", body)
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
