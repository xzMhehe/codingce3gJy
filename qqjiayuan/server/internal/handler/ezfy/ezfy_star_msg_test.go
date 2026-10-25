package ezfy

import (
	"strings"
	"testing"
)

// ============ 军官升星：玩家端不露概率 / 每星随机加成 守卫（2026-10-25）============
//
// 用户规则（原话）：
//   - 「提示不要给玩家展示具体概率，就是去掉」「成功率 20%，成功后三维各 +10 都去掉，运维自己知道就行」
//   - 「升星失败(成功率20%，星级不变) 提示改为 升星失败」
//   - 「每星三维加成 改成随机 1-配置的数，比如配置了 10 那么随机 1-10 给 后、军、学 加，都随机加」
//
// 概率与加成都是**运维参数**（管理端 ezfy_cfg_limit 可调），玩家端一句话都不该出现。

// TestStarUpMsgHidesRate —— 升星失败文案里不能带成功率
func TestStarUpMsgHidesRate(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_officer.go", "func (h *EzfyHandler) officerStarUp(")
	if strings.Contains(body, "升星失败(成功率") || strings.Contains(body, "成功率%d%%") {
		t.Fatal("升星失败提示不能再带「成功率」——那是运维参数，玩家端只显示「升星失败」")
	}
	if !strings.Contains(body, `return "升星失败", false`) {
		t.Fatal("升星失败应返回纯文案「升星失败」")
	}
}

// TestStarUpRandomGain —— 每星三维**各自独立**随机 +1~配置值（不是固定加配置值）
func TestStarUpRandomGain(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_officer.go", "func (h *EzfyHandler) officerStarUp(")
	for _, want := range []string{
		"ezfyStarAttrGain()",
		"o.Military + 1 + rand.Intn(gainMax)",
		"o.Logistics + 1 + rand.Intn(gainMax)",
		"o.Learning + 1 + rand.Intn(gainMax)",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("每星三维应各自随机 +1~配置值，缺少 %q", want)
		}
	}
}
