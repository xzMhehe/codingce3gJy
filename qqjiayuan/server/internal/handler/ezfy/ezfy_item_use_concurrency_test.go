package ezfy

import (
	"strings"
	"testing"
)

// ============ 背包道具使用并发卡控守卫（2026-10-25）============
//
// 用户反馈：「背包道具使用 前端防抖、后端多点卡控，现在玩家反馈 部分道具 瞬间多点
// 能用好多次 bug，还有开箱都加下防抖，单次使用 1s 卡控」。
//
// 根因（两条，缺一条都堵不住）：
//
//	① 前端 doUse 没有防抖 → 连点一次打出 N 个并发请求；
//	② 后端 consumeItemN 是「SELECT 取快照 → Update(count = it.Count - n)」的**读-改-写**，
//	   而且所有 case 都是「先生效、再扣」→ N 个并发请求都能通过 useItem 的 `count > have`
//	   校验并各自生效，最后各自把数量写回 0 → **1 个道具生效 N 次**
//	   （资源包刷资源 / 加速道具白加速 / 经验书刷经验 / 升星卡连升）。
//
// 现在的两道防线（改代码时别退回任何一条）：
//
//	A. 入口卡控：UseItem 全部道具 1 秒、加速类 3 秒；ChestOpen / ChestBagOpen / ChestBuy 1 秒；
//	B. 先扣后生效：useItemOnce 每个消耗型 case 先调 ezfyTakeItemUse 原子扣 1 个，
//	   扣不到就直接返回、**绝不生效**；consumeItemN 本身也是条件原子扣。

// TestConsumeItemIsAtomic —— 扣道具必须是条件原子 UPDATE，禁止读-改-写
func TestConsumeItemIsAtomic(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy.go", "func (h *EzfyHandler) consumeItemN(")
	if !strings.Contains(body, "count >= ?") {
		t.Fatal("consumeItemN 必须用条件原子扣（WHERE count >= n）：" +
			"否则并发连点会把同一件道具用多次（玩家反馈的「瞬间多点能用好多次」）")
	}
	if !strings.Contains(body, `gorm.Expr("count - ?", n)`) {
		t.Fatal("consumeItemN 必须在 SQL 侧做 count - n 原子自减")
	}
	for _, bad := range []string{"it.Count - n", "it.Count-n"} {
		if strings.Contains(body, bad) {
			t.Fatal("consumeItemN 不能再「读快照再写回」(it.Count - n)：" +
				"并发下 N 个请求读到同一个旧值、各自写回，等于同一道具被用 N 次")
		}
	}
	if !strings.Contains(body, "RowsAffected == 0") {
		t.Fatal("consumeItemN 扣不到（RowsAffected==0）必须返回 false，让调用方不生效")
	}
}

// TestUseItemOnceTakeBeforeEffect —— 使用道具必须「先扣后生效」
func TestUseItemOnceTakeBeforeEffect(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy.go", "func (h *EzfyHandler) useItemOnce(")
	if strings.Contains(body, `h.consumeItem(uid, cfgId, "使用道具")`) {
		t.Fatal("useItemOnce 里不能再出现「生效之后再扣道具」的 h.consumeItem(...)：" +
			"必须先调 h.ezfyTakeItemUse，扣不到就 return（否则连点并发时同一道具生效多次）")
	}
	// 消耗型 case：1/2/27/28/29/30/31/3/4/5/24/25/26/7/8/9/10/11/12/19/33
	if n := strings.Count(body, "h.ezfyTakeItemUse(uid, cfgId)"); n < 20 {
		t.Fatalf("消耗型 case 必须全部先扣后生效，目前只有 %d 处 ezfyTakeItemUse（应 >= 20）", n)
	}
}

// TestUseItemThrottled —— 背包道具使用：普通道具 1 秒卡控、加速类 3 秒
func TestUseItemThrottled(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_api.go", "func (h *EzfyHandler) UseItem(")
	if !strings.Contains(body, `ezfyActionThrottle(uid, "bag_use"`) {
		t.Fatal(`UseItem 必须走 ezfyActionThrottle(uid, "bag_use", ...) 卡控`)
	}
	if !strings.Contains(body, "useGap := int64(1000)") {
		t.Fatal("普通道具单次使用必须是 1 秒卡控（用户要求「单次使用 1s 卡控」）")
	}
	if !strings.Contains(body, "useGap = 3000") {
		t.Fatal("加速类道具（3/4/5/24/25/26）应保持 3 秒卡控")
	}
}

// TestChestOpenThrottled —— 开箱 / 买箱接口都要有 1 秒防抖
func TestChestOpenThrottled(t *testing.T) {
	for _, tc := range []struct{ file, sig, action string }{
		{"ezfy_officer.go", "func (h *EzfyHandler) ChestOpen(", `"chest_open"`},
		{"ezfy_chest_bag.go", "func (h *EzfyHandler) ChestBagOpen(", `"chest_open"`},
		{"ezfy_chest_bag.go", "func (h *EzfyHandler) ChestBuy(", `"chest_buy"`},
	} {
		body := ezfyFuncBody(t, tc.file, tc.sig)
		if !strings.Contains(body, "ezfyActionThrottle(uid, "+tc.action) {
			t.Fatalf("%s 缺少防抖：应调用 ezfyActionThrottle(uid, %s, ...)", tc.sig, tc.action)
		}
		if !strings.Contains(body, ", 1000)") {
			t.Fatalf("%s 的防抖时长应为 1000ms", tc.sig)
		}
	}
}
