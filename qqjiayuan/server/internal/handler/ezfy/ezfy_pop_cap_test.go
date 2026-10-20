package ezfy

import (
	"strings"
	"testing"
)

// ============ 人口上限口径守卫（2026-10-20）============
//
// 用户报：「2 区关了『召集人口灵活配置』，民居 45000，之前开着时召集到 10 万，
// 关了不应该掉到 45000 吗？召集人口上限 100000 是不是把民居上限盖掉了？」
//
// 根因：老实现里「民居硬上限」只在**自然增长**分支里夹（且只在人口从下方长过 pop_max 时），
// 已经超过 pop_max 的人口（召集上来的）永远不会降；召集门也只在新召集时卡。
//
// 现在统一成一把尺子 ezfyPopCapOf：
//
//	有效上限 = min(召集人口上限, 民居容量（民居限制开 且 允许突破关 且 pop_max>0 时）)
//
// 并让 calcResource（每次懒结算）/ Convene / 移民法案 都走同一口径。

func TestEzfyPopCapOf(t *testing.T) {
	cases := []struct {
		name       string
		houseOn    bool
		flexOn     bool
		popMax     int64
		conveneMax int64
		want       int64
	}{
		{"都不限", false, false, 0, 0, 0},
		{"只开召集上限", false, false, 45000, 100000, 100000},
		{"只开民居硬上限", true, false, 45000, 0, 45000},
		{"两个都开取小（民居更小）", true, false, 45000, 100000, 45000},
		{"两个都开取小（召集更小）", true, false, 45000, 30000, 30000},
		{"允许突破民居 → 只看召集上限", true, true, 45000, 100000, 100000},
		{"允许突破 + 无召集上限 → 不限", true, true, 45000, 0, 0},
		{"民居关掉 → 只看召集上限", false, false, 45000, 100000, 100000},
		{"pop_max=0（城里没民居）不夹人口", true, false, 0, 100000, 100000},
	}
	for _, c := range cases {
		if got := ezfyPopCapOf(c.houseOn, c.flexOn, c.popMax, c.conveneMax); got != c.want {
			t.Errorf("%s: ezfyPopCapOf(house=%v, flex=%v, popMax=%d, convene=%d) = %d, want %d",
				c.name, c.houseOn, c.flexOn, c.popMax, c.conveneMax, got, c.want)
		}
	}
}

// TestCalcResourceClampsExistingPop 懒结算必须把「已经超限」的人口夹下来（不只管自然增长）。
func TestCalcResourceClampsExistingPop(t *testing.T) {
	body := ezfyStripCommentLines(ezfyFuncBody(t, "ezfy.go", "func (h *EzfyHandler) calcResourceD("))
	if !strings.Contains(body, "ezfyEffPopCap(city.PopMax)") {
		t.Fatalf("calcResourceD 未按「有效人口上限」夹现有超限人口（关掉允许突破民居上限后人口不会降）：\n%s", body)
	}
	if !strings.Contains(body, "city.Pop = cap") {
		t.Fatalf("calcResourceD 取到了上限但没有把 city.Pop 赋值为 cap：\n%s", body)
	}
	// 防回退：旧的「只夹召集硬上限」写法必须已删除，否则民居硬上限永远不生效。
	if strings.Contains(body, "hardCap := ezfyConvenePopMaxCfg()") {
		t.Fatalf("calcResourceD 还在用旧的单个召集上限写法（民居硬上限会被绕过）：\n%s", body)
	}
}

// TestConveneUsesEffPopCap 召集与懒结算必须同一把尺子。
func TestConveneUsesEffPopCap(t *testing.T) {
	body := ezfyStripCommentLines(ezfyFuncBody(t, "ezfy.go", "func (h *EzfyHandler) Convene("))
	if !strings.Contains(body, "popCap := ezfyEffPopCap(city.PopMax)") {
		t.Fatalf("Convene 未使用 ezfyEffPopCap（召集上限与民居上限仍各卡各的）：\n%s", body)
	}
	if !strings.Contains(body, "city.Pop+popGain > popCap") {
		t.Fatalf("Convene 未按有效上限截断 popGain：\n%s", body)
	}
}

// TestAllPopGatesUseEffPopCap 人口召集 / 懒结算 / 移民法案 三处必须全部走同一口径。
func TestAllPopGatesUseEffPopCap(t *testing.T) {
	src := ezfyStripCommentLines(rawFile(t, "ezfy.go"))
	if n := strings.Count(src, "ezfyEffPopCap(city.PopMax)"); n < 3 {
		t.Fatalf("ezfy.go 里 ezfyEffPopCap(city.PopMax) 只有 %d 处，至少应有 3 处"+
			"（懒结算 calcResourceD / 人口召集 Convene / 移民法案 ItemType31）", n)
	}
}

// TestFrontendPopCapSameFormula 前端分母与按钮禁用必须和后端同口径。
func TestFrontendPopCapSameFormula(t *testing.T) {
	vue := strings.ReplaceAll(readFrontFile(t, "../../../../web/src/views/ezfy/Ezfy.vue"), "\r\n", "\n")
	vue = ezfyStripCommentLines(vue)
	if !strings.Contains(vue, "this.housePopLimitOn && !this.conveneFlexibleOn") {
		t.Fatal("前端 popCap 丢了「民居限制开 + 允许突破关」条件")
	}
	if !strings.Contains(vue, "Math.min(cap, pm)") {
		t.Fatal("前端 popCap 未与召集上限取小")
	}
	if !strings.Contains(vue, "return this.popCap > 0 && pop >= this.popCap") {
		t.Fatal("前端 conveneBlocked 未直接用 popCap（会和后端上限口径漂移）")
	}
	// 召集页必须显示「有效上限」，别再只显示民居容纳导致「人口 10 万 / 上限 4.5 万」看不懂
	cityVue := ezfyStripCommentLines(readFrontFile(t, "../../../../web/src/views/ezfy/modules/EzfyCity.vue"))
	if !strings.Contains(cityVue, "有效上限: {{ ezfy.popCap > 0 ? ezfy.fmtBig(ezfy.popCap) : '不限' }}") {
		t.Fatal("召集页未显示有效人口上限")
	}
}
