package handler

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
		{-5, 0},   // 异常入参不得产生负加成
		{1, 1},    // (1+1)/2
		{2, 1},    // (2+1)/2 = 1（向下取整）
		{3, 2},    // (3+1)/2
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
// randomEquipment 是战斗掉落（活动目标/野地）取装备的唯一出口，
// 必须排除 set_id > 0 的套装件（珠宝本来就被排除）。
func TestBattleLootExcludesSetPieces(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_officer.go", "func (h *EzfyHandler) randomEquipment(")
	if !strings.Contains(body, "e.SetId > 0") {
		t.Fatalf("randomEquipment 未排除套装件，战斗掉落仍会掉套装：\n%s", body)
	}
}

// TestEquipShopOnlySellsLoosePieces —— 商城只卖**散件**。
//
// ★ 2026-09-22 用户定稿：「散件也上吧，价格按加成 10 钻石到 50 钻石不等」
// —— 六大系列的单件（可单穿）与纯散件都能买；
// 但**第一批套装**（set_id 1~17、Series 为空）仍然只能开宝箱，
// 否则「套装装备只能通过宝箱开启」这条规则就形同虚设。
//
// 列表侧与购买侧都要拦（防前端被绕过直接 POST cfg_id）。
func TestEquipShopOnlySellsLoosePieces(t *testing.T) {
	list := ezfyFuncBody(t, "ezfy_officer.go", "func (h *EzfyHandler) equipShopList(")
	if !strings.Contains(list, `e.SetId > 0 && e.Series == ""`) {
		t.Fatalf("equipShopList 未拦住第一批套装件（只该卖散件）：\n%s", list)
	}
	if !strings.Contains(list, `"slot"`) {
		t.Fatalf("equipShopList 应按**部位**分组下发：\n%s", list)
	}

	buy := ezfyFuncBody(t, "ezfy_officer.go", "func (h *EzfyHandler) EquipShopBuy(")
	if !strings.Contains(buy, `cfg.SetId > 0 && cfg.Series == ""`) {
		t.Fatalf("EquipShopBuy 未拦截第一批套装件购买：\n%s", buy)
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
