package handler

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
)

// ---- 旧实现（改地图之前）的地形/海城判定，仅用于对比 ----

func oldTerrain(x, y int) int {
	h := ezfyAbs(x*73856093 ^ y*19349663)
	return h%8 + 1
}

func oldHasSeaNeighbor(x, y int) bool {
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			if dx == 0 && dy == 0 {
				continue
			}
			if oldTerrain(x+dx, y+dy) == 8 {
				return true
			}
		}
	}
	return false
}

func oldIsSeaCity(x, y int) bool {
	if oldTerrain(x, y) != 1 || !oldHasSeaNeighbor(x, y) {
		return false
	}
	return ezfyAbs(x*40503^y*2654435761)%3 == 0
}

// TestOnlineSeaCities 拿线上库导出的城市坐标，对比「旧规则 / 新规则」下的海城判定。
// 输入文件: /tmp/online_cities.txt，每行 "id\tuser_id\tx\ty"。
func TestOnlineSeaCities(t *testing.T) {
	f, err := os.Open("/tmp/online_cities.txt")
	if err != nil {
		t.Skip("没有 /tmp/online_cities.txt，跳过")
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	type row struct {
		id, uid, x, y int
		oldSea, newSea bool
		oldT, newT     int
	}
	var rows []row
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		p := strings.Fields(line)
		if len(p) < 4 {
			continue
		}
		id, _ := strconv.Atoi(p[0])
		uid, _ := strconv.Atoi(p[1])
		x, _ := strconv.Atoi(p[2])
		y, _ := strconv.Atoi(p[3])
		rows = append(rows, row{
			id: id, uid: uid, x: x, y: y,
			oldT: oldTerrain(x, y), newT: ezfyTerrainEx(x, y),
			oldSea: oldIsSeaCity(x, y), newSea: ezfyTerrainEx(x, y) == ezfyTerrainCoastalPlain,
		})
	}

	oldSeaN, newSeaN, lost, gained, onOceanOld, onOceanNew := 0, 0, 0, 0, 0, 0
	for _, r := range rows {
		if r.oldSea {
			oldSeaN++
		}
		if r.newSea {
			newSeaN++
		}
		if r.oldSea && !r.newSea {
			lost++
		}
		if !r.oldSea && r.newSea {
			gained++
		}
		if r.oldT == 8 {
			onOceanOld++
		}
		if r.newT == 8 {
			onOceanNew++
		}
	}

	fmt.Printf("线上城市总数: %d\n", len(rows))
	fmt.Printf("旧规则海城: %d    新规则海城: %d\n", oldSeaN, newSeaN)
	fmt.Printf("旧是海城、新不是(掉海城): %d    旧不是、新是(变海城): %d\n", lost, gained)
	fmt.Printf("旧规则落在海洋(地形8)上的城: %d    新规则落在海洋上的城: %d\n", onOceanOld, onOceanNew)

	fmt.Println("\n--- 旧是海城但新规则不再是（这些城的玩家会「突然不能造海军」）---")
	for _, r := range rows {
		if r.oldSea && !r.newSea {
			fmt.Printf("  城%-4d uid=%-9d (%3d,%3d) 旧地形=%-4s 新地形=%s\n",
				r.id, r.uid, r.x, r.y, ezfyTerrainName(r.oldT), ezfyTerrainName(r.newT))
		}
	}
	fmt.Println("\n--- 新规则下是海城的城 ---")
	for _, r := range rows {
		if r.newSea {
			fmt.Printf("  城%-4d uid=%-9d (%3d,%3d) %s\n", r.id, r.uid, r.x, r.y, ezfyRegionName(r.x, r.y))
		}
	}
	fmt.Println("\n--- 新规则下落在海洋里的城（会被迁移）---")
	for _, r := range rows {
		if r.newT == ezfyTerrainSea {
			fmt.Printf("  城%-4d uid=%-9d (%3d,%3d) 旧地形=%s\n", r.id, r.uid, r.x, r.y, ezfyTerrainName(r.oldT))
		}
	}

	// 机器可读输出，方便跟库里的「有航海协会的城」做差集
	oldIDs := []string{}
	newIDs := []string{}
	for _, r := range rows {
		if r.oldSea {
			oldIDs = append(oldIDs, strconv.Itoa(r.id))
		}
		if r.newSea {
			newIDs = append(newIDs, strconv.Itoa(r.id))
		}
	}
	fmt.Printf("OLDSEA=%s\n", strings.Join(oldIDs, ","))
	fmt.Printf("NEWSEA=%s\n", strings.Join(newIDs, ","))
}
