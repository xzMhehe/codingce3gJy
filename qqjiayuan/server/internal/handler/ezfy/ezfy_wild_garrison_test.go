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
//  3. 攻击有归属(玩家附属)的野地不产出战利品（战果全 0，防刷资源）。
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

	// ③ 用户规则（2026-10-19 澄清）：攻击**有归属(玩家附属)**的野地**不产出资源/黄金**
	//   —— 「应该都是 0 ... 不然有人靠这个刷资源」。所以野地配置产出 + 缴获守军采集物资
	//   一律不计（战果全 0），采集队身上已采的资源随守军一起损失。
	if !strings.Contains(order, "playerOwnedWild := (order.TargetType == 1 || order.TargetType == 2) && wlOwnerCityID > 0") {
		t.Fatal("缺少「有归属野地战利品清零」判定（否则可反复打有主野地刷资源）")
	}
	if !strings.Contains(order, "lootFood, lootSteel, lootOil, lootRare, lootGold = 0, 0, 0, 0, 0") {
		t.Fatal("有归属野地未把战利品清零（战果应全 0）")
	}
	if strings.Contains(order, "lootFood += captured.Food") {
		t.Fatal("有归属野地仍把缴获的采集物资并入战利品（违反「战果全 0」）")
	}

	// ④ 军官：驻军订单(status=3 且未返航)也要算「军官在外」；但必须限定 order_type=6，
	//   否则「返航归队后 status=3、result 为空」的普通战斗订单会把军官永久判忙（线上斯大林事故）
	off := rawFile(t, "ezfy_officer.go")
	if !strings.Contains(off, "status IN (0,1,2,5,6) OR (status = 3 AND order_type = 6 AND IFNULL(result,'') = '')") {
		t.Fatal("军官 busy 判定的 status=3 分支必须限定 order_type=6（否则打完仗返航的订单会锁死军官）")
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

	// ⑧ 攻击**有玩家守军（驻军6 / 采集7）**的附属野地 → 必须进指挥室，不能自动战斗
	//   后端：守军订单非空就强制 autoBattle=false
	if !strings.Contains(order, "hasPlayerDefender := len(wildDefOrders) > 0") {
		t.Fatal("processArrive 未按「有玩家守军」强制手动指挥")
	}
	if !strings.Contains(order, "if order.TargetType == 3 || hasPlayerDefender {") {
		t.Fatal("有玩家守军的附属野地仍可能自动战斗")
	}
	//   详情接口下发 has_defender（口径与守军构建一致：6/7 + status 1/3）
	if !strings.Contains(order, `"has_defender": hasDefender`) {
		t.Fatal("野地详情未下发 has_defender（前端无法禁用自动战斗）")
	}
	if !strings.Contains(order, `Where("target_id = ? AND order_type IN (6,7) AND status IN (1,3)", w.ID)`) {
		t.Fatal("has_defender 口径不对（必须含采集部队 7 与驻军 6）")
	}
	//   前端：出征页「自动战斗」对这类目标强制「否」
	main, err := os.ReadFile("../../../../web/src/views/ezfy/Ezfy.vue")
	if err != nil {
		t.Fatalf("读前端 Ezfy.vue 失败: %v", err)
	}
	if !strings.Contains(string(main), "this.selDetail && this.selDetail.has_defender") {
		t.Fatal("前端 orderIsPlayerTarget 未包含「有玩家驻军的野地」")
	}
}
