package ezfy

// ★★ 2026-10-23 玩家反馈「渔场、岛屿、沿海平原 是海野，采集和普通野地一样？」→ 用户拍板
// **沿海平原完全按海野**。本测试锁死两件事：
//
//	① 地形 → 采集资源（海野与陆地资源同名，这是玩家困惑的根源，不是 bug）：
//	   海底森林→石油、岛屿→钢铁、渔场→粮食、矿礁→稀矿、沿海平原→粮食；
//	② 两个谓词的分工不能混：
//	   · ezfyIsSeaWildTerrain = 展示/地图格（沿海平原**不是**，展示名仍是「海平」）；
//	   · ezfyIsSeaWildRules   = 玩法（采集倍率/守军/占领类型，沿海平原**是**）。
//	③ 海野倍率是配置项（gather_sea_bonus_mult，默认 3），代码里不得写死 —— 只允许
//	   通过 ezfyGatherSeaBonusMult() 读取。

import (
	"strings"
	"testing"
)

func TestSeaWildGatherMapping(t *testing.T) {
	// ① 采集资源按地形
	for _, c := range []struct {
		terrain int
		want    string
		name    string
	}{
		{ezfyTerrainSea, "石油", "海森"},
		{ezfyTerrainIsland, "钢铁", "岛屿"},
		{ezfyTerrainFishing, "粮食", "渔场"},
		{ezfyTerrainOreReef, "稀矿", "矿礁"},
		{ezfyTerrainCoastalPlain, "粮食", "海平"},
	} {
		if got := ezfyGatherResName(c.terrain); got != c.want {
			t.Fatalf("%s 采集资源 = %s, want %s", c.name, got, c.want)
		}
	}
	// ② 两个谓词分工
	for _, c := range []struct {
		terrain  int
		display  bool // ezfyIsSeaWildTerrain（展示/地图格）
		playable bool // ezfyIsSeaWildRules（玩法）
		name     string
	}{
		{ezfyTerrainSea, true, true, "海森"},
		{ezfyTerrainIsland, true, true, "岛屿"},
		{ezfyTerrainFishing, true, true, "渔场"},
		{ezfyTerrainOreReef, true, true, "矿礁"},
		// ★ 沿海平原：展示上仍是陆地地形（名字「海平」、地图格不画海格），
		//   但**玩法完全按海野**（2026-10-23 用户规则）
		{ezfyTerrainCoastalPlain, false, true, "海平"},
		{1, false, false, "平原"},
	} {
		if got := ezfyIsSeaWildTerrain(c.terrain); got != c.display {
			t.Fatalf("%s(%d) 展示口径 = %v, want %v", c.name, c.terrain, got, c.display)
		}
		if got := ezfyIsSeaWildRules(c.terrain); got != c.playable {
			t.Fatalf("%s(%d) 玩法口径 = %v, want %v", c.name, c.terrain, got, c.playable)
		}
	}
}

// TestSeaWildGatherMultFromCfg 海野倍率必须读配置（默认 3），不能写死。
func TestSeaWildGatherMultFromCfg(t *testing.T) {
	if ezfyGatherSeaBonusMultDef != 3 {
		t.Fatalf("海野倍率默认值应为 3, 实际 %g", ezfyGatherSeaBonusMultDef)
	}
	src := ezfyStripCommentLines(rawFile(t, "ezfy_order.go"))
	if !strings.Contains(src, "ezfyGatherSeaBonusMult()") {
		t.Fatal("采集计算没有走 ezfyGatherSeaBonusMult()（倍率必须可配置，别写死 3）")
	}
}
