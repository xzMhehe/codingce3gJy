package ezfy

import (
	"strings"
	"testing"
)

// ============ 出征耗时「提速来源」口径守卫 ============
//
// 背景（2026-10-16 玩家反馈「有运输站/科技加成反而变久了」）：
//
//  1. **侦察技巧(科技12) 只加军情情报等级，不加速行军**。耗时公式里那级 `tech[12]*2`
//     是原版公式的残留（原版 `ezfy_cfg_tech` 12 号写的是「行军速度+2%」，本项目后来把它的
//     效果改成了「情报+1级」，两套口径就打架了）。
//     真正加速行军的是 燃烧引擎(科技10，通用) + 喷气引擎(科技19，空军)，已折进 ezfyMarchSpeed。
//
//  2. 运输站(建筑20) 是**倍数**口径（对齐 cfg 文案「部队行动速度提高1.5~6.0倍(己方/盟友基地间)」）：
//     只按**出发城**等级 → 1级1.5倍 … 10级6.0倍；只对「己方/盟友基地间」的调动生效
//     （运输5 / 增援·驻守盟友6 / 派遣8）；**不**把目的地城等级加进来（6倍×6倍太夸张）。
//
// 预览(/order/preview) 与下单(createOrder) 必须**同口径**，否则「看到的耗时」≠「实际耗时」。

// TestMarchSpeedIgnoresReconTech —— 耗时公式里不得再出现「侦察技巧(科技12)」那一级
func TestMarchSpeedIgnoresReconTech(t *testing.T) {
	bad := []string{"techMap[12]*2", "techMap[12] * 2", "tech[12]*2", "tech[12] * 2"}
	for _, fn := range []string{
		"func (h *EzfyHandler) OrderPreview(",
		"func (h *EzfyHandler) createOrder(",
	} {
		body := ezfyFuncBody(t, "ezfy_order.go", fn)
		for _, b := range bad {
			if strings.Contains(body, b) {
				t.Fatalf("%s 里又出现 `%s` —— 侦察技巧(科技12)只加军情情报等级，不加速行军。\n"+
					"行军速度请走 ezfyMarchSpeed（燃烧引擎10 通用 / 喷气引擎19 空军）。", fn, b)
			}
		}
	}
}

// TestStationBonusIsMultiplierNotStack —— 运输站是「出发城等级 → 速度倍数」，不再双向叠加
func TestStationBonusIsMultiplierNotStack(t *testing.T) {
	for _, fn := range []string{
		"func (h *EzfyHandler) OrderPreview(",
		"func (h *EzfyHandler) createOrder(",
	} {
		body := ezfyFuncBody(t, "ezfy_order.go", fn)
		for _, bad := range []string{"dstStation", "stationLv += ", "station += ", "stationLv*3", "station*3"} {
			if strings.Contains(body, bad) {
				t.Fatalf("%s 里又出现 `%s` —— 运输站现在是「出发城等级→速度倍数」的单边口径，\n"+
					"不要再按等级相加/按百分比除（规则只在 ezfyStationSpeedMult / ezfyStationOrderType）。", fn, bad)
			}
		}
		if !strings.Contains(body, "ezfyStationSpeedMult(") {
			t.Fatalf("%s 没有走 ezfyStationSpeedMult 算运输站倍数（规则只在那一个函数里实现）", fn)
		}
		if !strings.Contains(body, "ezfyStationOrderType(") {
			t.Fatalf("%s 没有走 ezfyStationOrderType 判断是否吃运输站加成", fn)
		}
	}
}

// TestStationMultiplierTable —— 倍数表与原版文案一致：1级1.5倍 … 10级6.0倍；未建/越界回落 1.0
func TestStationMultiplierTable(t *testing.T) {
	cases := map[int]float64{0: 1, 1: 1.5, 2: 2.0, 5: 3.5, 10: 6.0, 11: 6.0, 99: 6.0, -3: 1}
	for lv, want := range cases {
		if got := ezfyStationSpeedMult(lv); got != want {
			t.Fatalf("运输站 %d 级应为 %v 倍，实际 %v", lv, want, got)
		}
	}
}

// TestStationOrderTypeScope —— 运输站只作用于「己方/盟友基地间」的订单
func TestStationOrderTypeScope(t *testing.T) {
	for _, ot := range []int{5, 6, 8} { // 运输 / 增援·驻守盟友城 / 派遣
		if !ezfyStationOrderType(ot) {
			t.Fatalf("订单类型 %d 应吃运输站加成（己方/盟友基地间）", ot)
		}
	}
	for _, ot := range []int{0, 1, 2, 3, 4, 7, 9} { // 侦查/掠夺/征服/采集/驻守采集…
		if ezfyStationOrderType(ot) {
			t.Fatalf("订单类型 %d 不该吃运输站加成（战斗/野外行动）", ot)
		}
	}
}

// TestSpdBonusNamesRealSources —— 「提速」清单要写清来源，且不能把侦察技巧算成提速
func TestSpdBonusNamesRealSources(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) OrderPreview(")
	if strings.Contains(body, `"科技+%d%%"`) {
		t.Fatal("「提速」清单里又出现含糊的「科技+%」，请写明 燃烧引擎 / 喷气引擎")
	}
	if !strings.Contains(body, `"燃烧引擎+%d%%"`) {
		t.Fatal("「提速」清单应列出 燃烧引擎（科技10，通用行军速度）")
	}
	if !strings.Contains(body, `"喷气引擎+%d%%"`) {
		t.Fatal("「提速」清单应列出 喷气引擎（科技19，仅部队含空军时）")
	}
	if !strings.Contains(body, `"运输站×%.1f倍"`) {
		t.Fatal("「提速」清单里运输站要按倍数展示（运输站×N.N倍）")
	}
}
