package ezfy

import (
	"strings"
	"testing"
)

// TestStaffMaxLevel15 参谋部(建筑10)等级上限 = 15。
//
// ★ 2026-10-14 用户：「参谋部等级上调到 15 级」——由 12 上调到 15。
//
//	本断言防止被后续改动悄悄改回 12（上限同时影响升级校验 `/building/upgrade`
//	与前端 `max_level` 下发）。
func TestStaffMaxLevel15(t *testing.T) {
	if got := ezfyBuildingMaxLevel(10, 10); got != 15 {
		t.Fatalf("参谋部(10) 等级上限应为 15，实际 %d", got)
	}
	// 回归：其它建筑上限不能被误伤
	cases := map[int]int{1: 10, 13: 12}
	for bid, want := range cases {
		if got := ezfyBuildingMaxLevel(bid, 10); got != want {
			t.Fatalf("建筑 %d 上限应为 %d，实际 %d", bid, want, got)
		}
	}
	// 民居受市政厅约束（市政厅<10 时 = 市政厅+1）
	if got := ezfyBuildingMaxLevel(2, 5); got != 6 {
		t.Fatalf("民居(2) 市政厅5级时上限应为 6，实际 %d", got)
	}
}

// TestStaffBlueprintRule 参谋部 10 级及以后每级都要建筑图纸（与司令部同款口径）。
//
// ★ 2026-10-14 用户：「升级到 15 级也是需要建筑图纸」。
//
//	规则：所有建筑 9→10 需图纸；司令部(13)/参谋部(10) 10级及以后每级都需图纸。
//	两处实现必须同口径：① `upgradeBuilding`（单次升级）；② `upgradeBuildingMax`（一键升级）。
//	本断言静态扫描两处，确认参谋部（10）都出现在「target/lv >= 11 需图纸」的条件里。
func TestStaffBlueprintRule(t *testing.T) {
	src := strings.ReplaceAll(rawFile(t, "ezfy.go"), "\r\n", "\n")
	// 单次升级：needBlueprint 条件里必须含 b.BuildingId == 10
	if !strings.Contains(src, "b.BuildingId == 2 || b.BuildingId == 13 || b.BuildingId == 10) && target >= 11") {
		t.Fatal("upgradeBuilding 的图纸条件漏了参谋部(10) —— 参谋部 10级后升级不扣图纸")
	}
	// 一键升级：同一口径
	if !strings.Contains(src, "b.BuildingId == 2 || b.BuildingId == 13 || b.BuildingId == 10) && lv >= 11") {
		t.Fatal("upgradeBuildingMax 的图纸条件漏了参谋部(10) —— 一键/单次升级口径不一致")
	}
}

// TestStaffLevelSeedHasUpTo15 种子配置里参谋部必须有 13/14/15 三级。
//
// ★ 关键：升级校验通过（上限15）但配置缺失时，`ezfyCfg.buildingLevel(10, 13)` 返回 nil，
//
//	会走到「配置缺失」分支直接拒绝 —— 玩家永远升不到 13 级。
//	本断言静态扫描种子文件，确认 13/14/15 都在。
func TestStaffLevelSeedHasUpTo15(t *testing.T) {
	src := strings.ReplaceAll(rawFile(t, "../../seed/ezfy_cfg_gen.go"), "\r\n", "\n")
	for _, want := range []string{
		"BuildingId: 10, Level: 13",
		"BuildingId: 10, Level: 14",
		"BuildingId: 10, Level: 15",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("参谋部种子配置缺少「%s」—— 上限开到15却无该级配置，升级会报「配置缺失」", want)
		}
	}
}
