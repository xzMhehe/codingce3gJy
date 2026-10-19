package ezfy

import (
	"os"
	"strings"
	"testing"
)

// TestWildGarrison 附属野地「增援驻军」的形态与操作（用户 2026-10-19 澄清版）：
//
//	「驻守部队本身就是驻守用的，不然野地没部队别人征服下就拿了」
//
// 所以增援到附属野地的部队保持 **纯驻军**（order_type=6 + status=3）：
//   - 敌袭时作为守军参战；可单支/一键召回；**不提供采集/收获**（那是采集队 7 的事）。
//
// 同时覆盖同批修复的三个真 bug：
//  1. 战后剩余兵力没写回订单 → 下次继续打兵力又变满编；
//  2. 一键召回漏掉野地驻军（只认 target_type=3）→ 永远召不回；
//  3. 歼灭对方采集部队时 carry 被清空 = 资源蒸发，攻方「打附属野地拿不到资源」。
func TestWildGarrison(t *testing.T) {
	order := rawFile(t, "ezfy_order.go")

	// ① 增援到附属野地：保持 status=3（纯驻军），不能再转成 7（采集）
	if !strings.Contains(order, `"增援部队已抵达%d级野地(%d,%d)驻防, 敌袭时协同防守。"`) {
		t.Fatal("增援到附属野地的战报文案不对（应为纯驻防）")
	}
	if i := strings.Index(order, `"增援报告: 附属野地"`); i > 0 {
		seg := order[max(0, i-700):i]
		if strings.Contains(seg, `"order_type": 7`) {
			t.Fatal("增援到附属野地又被转成采集部队(7)了 —— 驻军不该能采集")
		}
		if !strings.Contains(seg, `"status": 3`) {
			t.Fatal("增援到附属野地未保持出站驻军形态(status=3)")
		}
	} else {
		t.Fatal("找不到「增援报告: 附属野地」分支")
	}

	// ② 战后剩余兵力写回
	if !strings.Contains(order, "h.ezfyWriteBackWildGarrisonLeft(wildDefOrders, br.DefenderLosses)") {
		t.Fatal("野地驻军战损未写回订单（下次继续打时守军数量还是满编）")
	}

	// ③ 缴获：歼灭对方采集部队时，carry 里的资源要交给攻方，不能凭空清空
	if !strings.Contains(order, "func (h *EzfyHandler) ezfyDestroyWildlandDefenders(") ||
		!strings.Contains(order, ") (int, ezfyCarry)") {
		t.Fatal("ezfyDestroyWildlandDefenders 未返回缴获资源")
	}
	if !strings.Contains(order, "lootFood += captured.Food") {
		t.Fatal("缴获的采集物资未并入战利品（攻方打附属野地拿不到资源）")
	}

	// ④ 军官：驻军订单(status=3 且未返航)也要算「军官在外」
	off := rawFile(t, "ezfy_officer.go")
	if !strings.Contains(off, "status IN (0,1,2,5,6) OR (status = 3 AND IFNULL(result,'') = '')") {
		t.Fatal("军官 busy 判定漏了 status=3（驻军军官会被自愈成空闲/可重复派出）")
	}

	// ⑤ 一键召回必须覆盖野地驻军（原来只认 target_type=3）
	col := rawFile(t, "ezfy_collect.go")
	if !strings.Contains(col, "target_type IN (1,2) OR (target_type = 3 AND target_id IN (?))") {
		t.Fatal("一键召回未覆盖附属野地驻军（target_type=1/2）")
	}
	// ⑥ 采集入口仍只认驻守采集部队(7)：驻军点不到采集
	if strings.Contains(col, "order.OrderType == 6 && order.TargetType != 3 && order.Status == 3") {
		t.Fatal("StartCollect 又把驻军就地转成采集部队了")
	}

	// ⑦ 前端：驻军/采集部队都要有 [召回]；驻军不再显示 [采集]
	web, err := os.ReadFile("../../../../web/src/views/ezfy/modules/EzfyReport.vue")
	if err != nil {
		t.Fatalf("读前端 EzfyReport.vue 失败: %v", err)
	}
	ws := string(web)
	if !strings.Contains(ws, `v-if="o.status === 1 || o.status === 3"`) {
		t.Fatal("驻军列表里 status=1/3 的部队没有 [召回] 入口")
	}
	if strings.Contains(ws, `o.status === 3 && o.target_type !== 3`) {
		t.Fatal("驻军列表又给 status=3 的野地驻军加了 [采集] 入口")
	}
}
