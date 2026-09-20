package handler

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
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

// TestWorldGeography 回归测试：证明「世界坐标与大陆不匹配」已不存在。
//
// 背景：大陆几何定义在 150×150 坐标系里，而 ezfyInShape 会先把世界坐标乘
// ezfyLandScale(=150/500=0.3) 缩回去，所以 0~499 的世界坐标是被完整覆盖的。
// `待完成功能.md` 2.5 节的 Y1 是「改成大陆几何之前」的旧结论，本测试用数字把它钉死。
//
// 城市侧输入文件 /tmp/ezfy_cities.txt（每行 "id\tx\ty"），由
//
//	go run ./cmd/dbq -sql "SELECT id, x, y FROM ezfy_city" -n 500
//
// 导出后手工转成 TSV；文件不存在则跳过城市检查。
//
// 运行：go test ./internal/handler/ -run TestWorldGeography -v
func TestWorldGeography(t *testing.T) {
	const n = ezfyWorldSize // 500

	// ---- 1) 全世界 0~499 的陆地/海洋占比 ----
	total, ocean := 0, 0
	byContinent := map[int]int{}
	for x := 0; x < n; x++ {
		for y := 0; y < n; y++ {
			total++
			c := ezfyContinentOf(x, y)
			if c == ezfyOcean {
				ocean++
			}
			byContinent[c]++
		}
	}
	t.Logf("全世界 0~%d：总格 %d，陆地 %d (%.1f%%)，海洋 %d (%.1f%%)",
		n-1, total, total-ocean, float64(total-ocean)*100/float64(total),
		ocean, float64(ocean)*100/float64(total))

	// 陆地必须成规模：如果几何只覆盖了局部，这里会掉到很低
	if float64(total-ocean)/float64(total) < 0.30 {
		t.Errorf("陆地占比只有 %.1f%%，大陆几何可能没有铺满全世界", float64(total-ocean)*100/float64(total))
	}
	for _, id := range ezfyContinentIDs {
		t.Logf("  大陆%d %-4s: %6d 格 (%.1f%%)", id, ezfyContinentNames[id],
			byContinent[id], float64(byContinent[id])*100/float64(total))
		if byContinent[id] == 0 {
			t.Errorf("大陆%d %s 一格都没有，几何配置有问题", id, ezfyContinentNames[id])
		}
	}

	// ---- 2) 城市落点范围 50~449（findFreePos: 50+rand(400)）的陆地占比 ----
	lo, hi := 50, 449
	t2, o2 := 0, 0
	for x := lo; x <= hi; x++ {
		for y := lo; y <= hi; y++ {
			t2++
			if ezfyTerrain(x, y) == ezfyTerrainSea {
				o2++
			}
		}
	}
	t.Logf("城市落点区 %d~%d：总格 %d，陆地 %d (%.1f%%)，海洋 %d (%.1f%%)",
		lo, hi, t2, t2-o2, float64(t2-o2)*100/float64(t2), o2, float64(o2)*100/float64(t2))

	// ---- 3) 各大陆在世界坐标里的包围盒（看是否铺满整张图） ----
	type bbox struct{ x0, y0, x1, y1 int }
	bb := map[int]bbox{}
	for x := 0; x < n; x++ {
		for y := 0; y < n; y++ {
			c := ezfyContinentOf(x, y)
			if c == ezfyOcean {
				continue
			}
			b, ok := bb[c]
			if !ok {
				bb[c] = bbox{x, y, x, y}
				continue
			}
			if x < b.x0 {
				b.x0 = x
			}
			if y < b.y0 {
				b.y0 = y
			}
			if x > b.x1 {
				b.x1 = x
			}
			if y > b.y1 {
				b.y1 = y
			}
			bb[c] = b
		}
	}
	for _, id := range ezfyContinentIDs {
		if b, ok := bb[id]; ok {
			t.Logf("  大陆%d %-4s 包围盒 x[%d,%d] y[%d,%d]", id, ezfyContinentNames[id],
				b.x0, b.x1, b.y0, b.y1)
		}
	}

	// ---- 4) 沿海平原（海城唯一可建地）数量 ----
	coastal := 0
	for x := 0; x < n; x++ {
		for y := 0; y < n; y++ {
			if ezfyIsCoastalPlainAt(x, y) {
				coastal++
			}
		}
	}
	t.Logf("沿海平原格：%d (%.2f%% of world)", coastal, float64(coastal)*100/float64(total))
	if coastal == 0 {
		t.Error("全图没有沿海平原，海城将无处可建")
	}

	// ---- 5) findFreePos 的兜底坐标必须是陆地（否则会凭空把城扔进海里） ----
	if ezfyTerrain(200, 200) == ezfyTerrainSea {
		t.Errorf("findFreePos 兜底坐标 (200,200) 落在海洋上，会把城市扔进海里")
	} else {
		t.Logf("findFreePos 兜底坐标 (200,200) 地形=%s 区域=%s ✓",
			ezfyTerrainNameEx(200, 200), ezfyRegionName(200, 200))
	}

	// ---- 6) 库里真实城市落在什么地形上 ----
	f, err := os.Open("/tmp/ezfy_cities.txt")
	if err != nil {
		t.Logf("没有 /tmp/ezfy_cities.txt，跳过城市检查")
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	onOcean, onCoastal, onLand := 0, 0, 0
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 3 {
			continue
		}
		id, _ := strconv.Atoi(parts[0])
		x, _ := strconv.Atoi(parts[1])
		y, _ := strconv.Atoi(parts[2])
		tt := ezfyTerrain(x, y)
		te := ezfyTerrainEx(x, y)
		switch {
		case tt == ezfyTerrainSea:
			onOcean++
			t.Logf("  ★ 城%d (%d,%d) 在【海洋】上  terrain=%d(%s) 区域=%s",
				id, x, y, tt, ezfyTerrainName(tt), ezfyRegionName(x, y))
		case te == ezfyTerrainCoastalPlain:
			onCoastal++
			t.Logf("  城%d (%d,%d) 沿海平原 ✓ 区域=%s", id, x, y, ezfyRegionName(x, y))
		default:
			onLand++
			t.Logf("  城%d (%d,%d) 陆地 %s 区域=%s", id, x, y, ezfyTerrainName(te), ezfyRegionName(x, y))
		}
	}
	t.Logf("城市地形统计：海洋 %d / 沿海平原 %d / 内陆 %d", onOcean, onCoastal, onLand)
	if onOcean > 0 {
		t.Errorf("有 %d 座城市落在海洋地形上", onOcean)
	}
}
