package ezfy

import "testing"

// TestWildTerrainDisplayName 守住用户 2026-10-05 纠正的两条口径。
//
// 用户原话（第一次）：「海岛占领以后显示海底森林」——岛屿不该叫海底森林；
//
//	「就是展示还是岛屿 其实也是海野 就是采集资源跟海底森林不一样」。
//
// 用户原话（第二次纠正，把「海洋」和「海底森林」分开了）：
//
//	「只有 海底森林 是海底森林啊，海洋就是海洋啊」。
//
// 所以三个概念必须分清：
//
//	① 海洋(地形8) + **无野地** = 纯海洋 → 展示「海洋」
//	② 海洋(地形8) + **有野地** = 海底森林 → 展示「海底森林」
//	③ 岛屿(地形7) = 海野玩法，但展示名是「岛屿」
//
// 这三条以前被改错过（「岛屿也是海野」那轮把岛屿也写成了海底森林；随后我又一度把
// 所有地形 8 都当成海底森林），所以逐条断言钉住。
func TestWildTerrainDisplayName(t *testing.T) {
	// ① 玩法判定：海洋/岛屿/渔场/矿礁都算海野；沿海平原不算
	if !ezfyIsSeaWildTerrain(ezfyTerrainIsland) {
		t.Fatal("岛屿(7) 应当按海野处理（ezfyIsSeaWildTerrain(7) 必须为 true）")
	}
	if !ezfyIsSeaWildTerrain(ezfyTerrainSea) {
		t.Fatal("海洋(8) 应当按海野处理")
	}
	if !ezfyIsSeaWildTerrain(ezfyTerrainFishing) {
		t.Fatal("渔场(10) 应当按海野处理")
	}
	if !ezfyIsSeaWildTerrain(ezfyTerrainOreReef) {
		t.Fatal("矿礁(11) 应当按海野处理")
	}
	if ezfyIsSeaWildTerrain(ezfyTerrainCoastalPlain) {
		t.Fatal("沿海平原(9) 不是海野")
	}

	// ② 在世界地图里分别找一个「岛屿格」「纯海洋格（无野地）」「海洋野地格（有野地）」
	islandX, islandY, foundIsland := -1, -1, false
	pureX, pureY, foundPure := -1, -1, false
	seaWildX, seaWildY, foundSeaWild := -1, -1, false
	fishingX, fishingY, foundFishing := -1, -1, false
	oreX, oreY, foundOre := -1, -1, false
	for x := 1; x < ezfyWorldSize; x++ {
		for y := 1; y < ezfyWorldSize; y++ {
			switch ezfyTerrainEx(x, y) {
			case ezfyTerrainIsland:
				if !foundIsland {
					islandX, islandY, foundIsland = x, y, true
				}
			case ezfyTerrainSea:
				if ezfyWildlandLevel(x, y) == 0 {
					if !foundPure {
						pureX, pureY, foundPure = x, y, true
					}
				} else if !foundSeaWild {
					seaWildX, seaWildY, foundSeaWild = x, y, true
				}
			case ezfyTerrainFishing:
				if !foundFishing {
					fishingX, fishingY, foundFishing = x, y, true
				}
			case ezfyTerrainOreReef:
				if !foundOre {
					oreX, oreY, foundOre = x, y, true
				}
			}
		}
		if foundIsland && foundPure && foundSeaWild && foundFishing && foundOre {
			break
		}
	}
	if !foundIsland || !foundPure || !foundSeaWild || !foundFishing || !foundOre {
		t.Skip("世界地图里找不到齐全的格子，跳过展示名断言")
	}

	// ③ 岛屿：海野玩法，但展示名是「岛屿」
	if got := ezfyWildTerrainDisplayName(islandX, islandY, true); got != "岛屿" {
		t.Fatalf("岛屿(%d,%d) 展示名应为「岛屿」，实际 %q —— 岛屿是海野玩法，但不能叫「海底森林」",
			islandX, islandY, got)
	}

	// ④ 纯海洋：就是「海洋」，不是海底森林（knownWild=false 走 level 判定）
	if got := ezfyWildTerrainDisplayName(pureX, pureY, false); got != "海洋" {
		t.Fatalf("纯海洋(%d,%d)（无野地）展示名应为「海洋」，实际 %q —— "+
			"用户明确纠正过「海洋就是海洋」，别把所有地形 8 都当成海底森林", pureX, pureY, got)
	}
	// 即使调用方误传 knownWild=true，只要地形是海且有野地才该叫海底森林 —— 这里断言的是
	// 「knownWild 只影响『确定有野地』的场景」，纯海洋场景必须传 false。
	if got := ezfyWildTerrainDisplayName(seaWildX, seaWildY, false); got != "海底森林" {
		t.Fatalf("海洋野地(%d,%d) 展示名应为「海底森林」，实际 %q", seaWildX, seaWildY, got)
	}

	// ★ 2026-10-20 新增海野：渔场 / 矿礁的展示名与采集资源
	if got := ezfyWildTerrainDisplayName(fishingX, fishingY, false); got != "渔场" {
		t.Fatalf("渔场(%d,%d) 展示名应为「渔场」，实际 %q", fishingX, fishingY, got)
	}
	if got := ezfyWildTerrainDisplayName(oreX, oreY, false); got != "矿礁" {
		t.Fatalf("矿礁(%d,%d) 展示名应为「矿礁」，实际 %q", oreX, oreY, got)
	}

	// ⑤ 采集资源必须按**地形**区分（用户：「采集资源跟海底森林不一样」）
	if r := ezfyGatherResName(ezfyTerrainIsland); r != "钢铁" {
		t.Fatalf("岛屿采集资源应为「钢铁」，实际 %q", r)
	}
	if r := ezfyGatherResName(ezfyTerrainSea); r != "石油" {
		t.Fatalf("海底森林采集资源应为「石油」，实际 %q", r)
	}
	if r := ezfyGatherResName(ezfyTerrainFishing); r != "粮食" {
		t.Fatalf("渔场采集资源应为「粮食」，实际 %q", r)
	}
	if r := ezfyGatherResName(ezfyTerrainOreReef); r != "稀矿" {
		t.Fatalf("矿礁采集资源应为「稀矿」，实际 %q", r)
	}
	if ezfyTerrainTreasureNames[ezfyTerrainIsland][0] == ezfyTerrainTreasureNames[ezfyTerrainSea][0] {
		t.Fatal("岛屿与海底森林的可采集宝物池不应完全相同（按地形区分）")
	}
	if len(ezfyTerrainTreasureNames[ezfyTerrainFishing]) == 0 ||
		len(ezfyTerrainTreasureNames[ezfyTerrainOreReef]) == 0 {
		t.Fatal("渔场/矿礁必须配置可采集宝物池")
	}
}

// TestNewSeaWildAdminTerrain 管理端格子覆盖必须允许新地形 10/11，同时挡住越界值。
func TestNewSeaWildAdminTerrain(t *testing.T) {
	for _, id := range []int{ezfyTerrainFishing, ezfyTerrainOreReef} {
		if msg := ezfyCheckTileVals(map[string]interface{}{"terrain": id}); msg != "" {
			t.Fatalf("管理端地形 %d 应被允许，实际报错: %s", id, msg)
		}
	}
	if msg := ezfyCheckTileVals(map[string]interface{}{"terrain": 12}); msg == "" {
		t.Fatal("管理端地形 12 应被拒绝")
	}
}
