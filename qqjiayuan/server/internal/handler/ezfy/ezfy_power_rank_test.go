package ezfy

import (
	"strings"
	"testing"
)

// ============ 战力榜兵种分守卫（★ 2026-10-25 用户反馈「侦察机多 排名第一」）============
//
// 用户要求：「战力榜兵种那个不合理 —— 有人侦察机多 排名第一，应该按兵种占用人口、数量来算；
// 注意不要太大，现在都有 20 亿兵的，这种兵种分算到 5 万左右合理」。
//
// 事故根因：旧口径的兵种「质量」= (生命+防御+三系攻击+**速度**+射程)/100，
// 侦察机速度 5000（全兵种最高）→ 质量 50.95，比战列舰(45.88)/航母(65.45 除外)都高，
// 再乘 count^0.8 → 侦察机堆量直接第一。
//
// 本文件守住三条：
//  1. 兵种分只由「数量 × 占用人口」决定，封顶 5 万，且**严格单调**（兵多的人必须排前面）；
//  2. 侦察机(1 人口) 堆 20 亿 必须打不过 航母(55 人口) 4 亿；
//  3. 玩家端 / 管理端战力榜必须是**同一套实现**，不许再各自算一套兵种质量。

// TestTroopPowerPopulationBased 兵种分只看人口占用，且侦察机堆量不再赢。
func TestTroopPowerPopulationBased(t *testing.T) {
	// 侦察机 1 人口 × 20 亿 = 20 亿人口；航母 55 人口 × 4 亿 = 220 亿人口
	recon := ezfyTroopPower(20e8 * 1)
	carrier := ezfyTroopPower(4e8 * 55)
	if !(carrier > recon) {
		t.Fatalf("20 亿侦察机(%0.f) 不应高于 4 亿航母(%0.f)：兵种分必须按人口占用算", recon, carrier)
	}
	// 没兵 = 0 分
	if got := ezfyTroopPower(0); got != 0 {
		t.Fatalf("没兵时兵种分必须为 0，实际 %v", got)
	}
	// 严格单调：人口越多分越高（同名次不翻转）
	prev := -1.0
	for _, pop := range []float64{1, 1e4, 1e6, 1e8, 1e9, 1e10, 1e11, 1e12} {
		v := ezfyTroopPower(pop)
		if v <= prev {
			t.Fatalf("兵种分必须严格单调递增：pop=%g 得 %v，上一档 %v", pop, v, prev)
		}
		prev = v
	}
}

// TestTroopPowerCapped 20 亿兵也不能把兵种分顶爆（用户口径：5 万左右）。
func TestTroopPowerCapped(t *testing.T) {
	if ezfyPowerTroopCap != 50000 {
		t.Fatalf("兵种分上限应为 5 万（用户口径），实际 %v", ezfyPowerTroopCap)
	}
	// 极端：1 万亿人口（远超现在 20 亿兵的量级）
	if got := ezfyTroopPower(1e12); got >= ezfyPowerTroopCap {
		t.Fatalf("兵种分必须恒小于上限 %v，实际 %v", ezfyPowerTroopCap, got)
	}
	// 当前全服第一量级（约 700 亿人口）应落在 4~5 万区间
	if got := ezfyTroopPower(7e10); got < 40000 || got > ezfyPowerTroopCap {
		t.Fatalf("700 亿人口的兵种分应落在 4~5 万，实际 %v", got)
	}
}

// TestPowerPartsSum 三项明细之和必须等于总战力（前端「战力明细」三列相加要对得上）。
func TestPowerPartsSum(t *testing.T) {
	power, tech, build, troop := ezfyPowerParts(27300, 73040, 7e10, 0.5)
	if power != tech+build+troop {
		t.Fatalf("总战力必须等于三项之和：power=%v tech=%v build=%v troop=%v", power, tech, build, troop)
	}
	if troop <= 0 || tech <= 0 || build <= 0 {
		t.Fatalf("三项都应有值：tech=%v build=%v troop=%v", tech, build, troop)
	}
}

// TestPowerRankSingleImplementation 玩家端与管理端必须共用 ezfyPowerParts，不许再各算一套兵种质量。
func TestPowerRankSingleImplementation(t *testing.T) {
	player := ezfyStripCommentLines(strings.ReplaceAll(
		ezfyFuncBody(t, "ezfy_api.go", "func (h *EzfyHandler) Rank("), "\r\n", "\n"))
	admin := ezfyStripCommentLines(strings.ReplaceAll(
		ezfyFuncBody(t, "ezfy_admin_rank.go", "func (h *EzfyAdmin) AdminEzfyPowerRank("), "\r\n", "\n"))

	for name, body := range map[string]string{"玩家端 Rank": player, "管理端 AdminEzfyPowerRank": admin} {
		if !strings.Contains(body, "ezfyPowerParts(") {
			t.Fatalf("%s 没有走 ezfyPowerParts（战力口径必须唯一实现）：\n%s", name, body)
		}
		// 旧口径指纹：兵种质量（含速度）与 count^pow 加权 —— 出现即回退
		if strings.Contains(body, "troopQuality") {
			t.Fatalf("%s 又出现了 troopQuality（侦察机速度 5000 会把它算成最高质量 → 侦察机堆量第一）：\n%s", name, body)
		}
		if strings.Contains(body, "math.Pow(float64(t.Count)") {
			t.Fatalf("%s 又按 count^pow 算兵种分了（应改人口口径 Σ数量×人口）：\n%s", name, body)
		}
	}
	// 兵种人口只能有一处实现
	src := ezfyStripCommentLines(strings.ReplaceAll(readFrontFile(t, "ezfy_power.go"), "\r\n", "\n"))
	if !strings.Contains(src, "func ezfyTroopPop(") {
		t.Fatal("ezfy_power.go 里找不到 ezfyTroopPop（兵种人口口径唯一实现）")
	}
}
