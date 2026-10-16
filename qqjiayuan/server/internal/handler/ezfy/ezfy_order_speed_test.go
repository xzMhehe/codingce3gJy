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
//  2. 运输站改成「双向叠加」：运输(5)/增援(6) 出发城 + 目的地城运输站等级**相加**（每级 3%）。
//     原来是「两边都建了才生效、只按出发城算」的门槛规则 —— 玩家反馈「一个运输站加速太少了」。
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

// TestStationBonusStacksBothWays —— 运输/增援的运输站加成必须是「出发城 + 目的地城」相加
func TestStationBonusStacksBothWays(t *testing.T) {
	preview := ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) OrderPreview(")
	if !strings.Contains(preview, "stationLv += dstStationLv") {
		t.Fatal("OrderPreview 的运输站加成要改成双向叠加（stationLv += dstStationLv）")
	}
	if strings.Contains(preview, "stationLv < 1 || dstStationLv < 1") {
		t.Fatal("OrderPreview 里还留着「双向驿站」门槛规则（两边都建才生效），应改为等级相加")
	}
	create := ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) createOrder(")
	if !strings.Contains(create, "station += dstStation") {
		t.Fatal("createOrder 的运输站加成要改成双向叠加（station += dstStation）")
	}
	if strings.Contains(create, "station < 1 || dstStation < 1") {
		t.Fatal("createOrder 里还留着「双向驿站」门槛规则（两边都建才生效），应改为等级相加")
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
}
