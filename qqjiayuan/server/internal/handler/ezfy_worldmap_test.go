package handler

import (
	"fmt"
	"testing"
)

// TestWorldMapDump 开发期临时用：把整张世界地图按 10 格取样打出来，检查陆地/海洋分布。
func TestWorldMapDump(t *testing.T) {
	step := 10
	land, sea, coastal := 0, 0, 0
	for y := 0; y < ezfyWorldSize; y += step {
		line := ""
		for x := 0; x < ezfyWorldSize; x += step {
			t := ezfyTerrainEx(x, y)
			switch {
			case t == ezfyTerrainSea:
				sea++
				line += "~"
			case t == ezfyTerrainCoastalPlain:
				coastal++
				land++
				line += "c"
			case t == 7: // 山地
				land++
				line += "^"
			default:
				land++
				line += "."
			}
		}
		fmt.Println(line)
	}
	total := land + sea
	fmt.Printf("陆地 %d/%d = %.1f%%  海洋 %d/%d = %.1f%%  其中沿海平原 %d\n",
		land, total, float64(land)*100/float64(total),
		sea, total, float64(sea)*100/float64(total), coastal)

	// 七大洲各自占多少格
	byCont := map[string]int{}
	for y := 0; y < ezfyWorldSize; y += 5 {
		for x := 0; x < ezfyWorldSize; x += 5 {
			byCont[ezfyRegionName(x, y)]++
		}
	}
	for k, v := range byCont {
		fmt.Printf("  %s: %d\n", k, v)
	}
}

// TestCoastalPlainCensus 全图统计：各地形数量 + 沿海平原分布（海城选址用）。
func TestCoastalPlainCensus(t *testing.T) {
	byTerrain := map[int]int{}
	var plains, coastalPlains []string
	for y := 0; y < ezfyWorldSize; y += 2 {
		for x := 0; x < ezfyWorldSize; x += 2 {
			tt := ezfyTerrainEx(x, y)
			byTerrain[tt]++
			if ezfyTerrain(x, y) == 1 {
				plains = append(plains, fmt.Sprintf("(%d,%d)", x, y))
				if tt == ezfyTerrainCoastalPlain {
					coastalPlains = append(coastalPlains, fmt.Sprintf("(%d,%d)", x, y))
				}
			}
		}
	}
	sampled := 0
	for _, n := range byTerrain {
		sampled += n
	}
	fmt.Println("采样点总数:", sampled)
	for id := 1; id <= 9; id++ {
		fmt.Printf("  %-6s(id=%d): %d\n", ezfyTerrainName(id), id, byTerrain[id])
	}
	fmt.Printf("平原 %d，其中沿海平原 %d (占平原 %.1f%%)\n",
		len(plains), len(coastalPlains), float64(len(coastalPlains))*100/float64(len(plains)+1))
	fmt.Println("沿海平原样例:", coastalPlains[:minInt(len(coastalPlains), 12)])

	// 中纬度区域(y 200~300)的沿海平原，方便挑一个做海城测试
	mid := []string{}
	for y := 200; y <= 300; y++ {
		for x := 0; x < ezfyWorldSize; x++ {
			if ezfyTerrainEx(x, y) == ezfyTerrainCoastalPlain {
				mid = append(mid, fmt.Sprintf("(%d,%d)", x, y))
			}
		}
	}
	fmt.Printf("y∈[200,300] 的沿海平原共 %d 个，样例: %v\n", len(mid), mid[:minInt(len(mid), 15)])
}
