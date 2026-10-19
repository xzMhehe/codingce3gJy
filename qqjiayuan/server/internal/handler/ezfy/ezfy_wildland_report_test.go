package ezfy

import (
	"strings"
	"testing"
)

// TestWildlandOwnerReport 附属野地守方战报（用户 2026-10-19：
// 「自己附属野地被征服、被掠夺、被侦查 战斗报告也要能看」）。
//
// 静态检查关键接线（本包测试都是这个风格：纯函数 + 源码断言，不依赖 DB）：
//  1. 结算前捕获野地主人 —— 征服成功会走 ezfyClearWildlandAt 清记录，之后再按坐标查就查不到原主人；
//  2. 守方战报在「正常结算末尾」和「野地上限提前 return」两处都要写（后者会跳过末尾）；
//  3. 标题必须是「被掠夺/被征服报告」前缀 —— ezfyReportCategory 只认这两个前缀归「战斗报告(2)」，
//     「防守报告」会被归到「其他(3)」→ 玩家在战斗报告页看不到；
//  4. 侦查：野地没有雷达站，出征即给主人发「被侦查报告」；
//  5. 无主野地 / 自己打自己 → 不写。
func TestWildlandOwnerReport(t *testing.T) {
	src := rawFile(t, "ezfy_order.go")

	if !strings.Contains(src, "wlOwnerUID, wlOwnerCityID = h.ezfyWildlandOwnerOf(order.TargetX, order.TargetY, uid)") {
		t.Fatal("ezfy_order.go 结算前未捕获附属野地主人（征服清记录后会丢失原主人）")
	}
	if n := strings.Count(src, "h.addWildlandOwnerReport("); n < 2 {
		t.Fatalf("守方战报调用点不足（末尾统一 + 野地上限 return 各一处），实际 %d", n)
	}
	for _, want := range []string{`"被掠夺报告: "`, `"被征服报告: "`} {
		if !strings.Contains(src, want) {
			t.Fatalf("缺少守方战报标题 %s（必须能被 ezfyReportCategory 归到战斗报告）", want)
		}
	}
	if !strings.Contains(src, "有敌军对我方附属野地进行了侦查") {
		t.Fatal("附属野地被侦查时未给主人写报告")
	}
	if !strings.Contains(src, "owner.UserID == atkUID") {
		t.Fatal("ezfyWildlandOwnerOf 未排除「自己打自己的野地」")
	}
	// 守军覆没报告合并成一条（原来按驻军/采集订单逐条写，野地主人一次被打收 N 条重复）
	if !strings.Contains(src, "if len(orders) > 0 {\n\t\th.addReport(owner.UserID, 2, \"战斗报告: 附属野地失守\"") {
		t.Fatal("「附属野地失守」未收敛成一条（仍按订单逐条写）")
	}
}
