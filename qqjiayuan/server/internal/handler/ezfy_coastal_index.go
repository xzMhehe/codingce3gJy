package handler

import (
	"math/rand"
	"sync"
)

// 沿海平原候选点缓存（★ 2026-09-21 线上性能事故修复）
//
// 背景：迁城（沿海迁城计划）要在「目标洲内找一个空闲的沿海平原」。
// 沿海平原 = 平原(1) 且 8 邻域内有海洋(8)，是**派生地形**，在地图上占比极小
// （线上欧洲沿海平原总共只有 55 格 / 一万多格陆地）。纯随机采样几乎撞不上，
// 于是迁城逻辑改成「全图枚举 500×500 的所有格子」。
//
// 问题：单次全图枚举 = 250000 次 ok()，而每次 ok() 内部：
//   - ezfyContinentOf：遍历 7 大洲、每洲多个多边形做点在形状内判定
//   - ezfyTerrainEx → ezfyHasSeaNeighbor → 8 邻域各再算一次 ezfyTerrain
//     （即 ezfyContinentOf 又要跑 9 遍）
//   - ezfyTileAt：每次取 RLock
//
// 合计每格要跑十几次「点在多边形内」判定 → 单次全图枚举上百万次形状运算，
// 1 核机器直接吃满 CPU 几秒。批量迁城逐城调用 → CPU 尖峰 + 全表读 cities 的 IO 尖峰。
//
// 修法：**把「全图枚举」的结果缓存起来**。
//   - 索引按洲分组，只存「天生就是沿海平原」的坐标（与城市占用无关）；
//   - 是否被占用在取用时用 occupied 集合过滤（占用状态每次实时读库，保证正确）；
//   - 地图格子覆盖配置（ezfy_map_tile）改动时失效重建。
//
// 这样全图枚举**整个进程只跑一次**，之后每座城都是 O(1) 随机取点。

// coastalIndex 按大洲分组的「沿海平原」坐标索引
type coastalIndex struct {
	mu sync.RWMutex
	// byContinent[洲] = 该洲所有沿海平原坐标
	byContinent map[int][][2]int
	ready       bool
}

var coastalIdx coastalIndex

// ezfyInvalidateCoastalIndex 地图覆盖配置变更后调用，强制下次重建索引。
func ezfyInvalidateCoastalIndex() {
	coastalIdx.mu.Lock()
	coastalIdx.ready = false
	coastalIdx.byContinent = nil
	coastalIdx.mu.Unlock()
}

// coastalPlainCandidates 取「某洲所有沿海平原坐标」。
//
// ★ 只算一次并缓存：全图枚举 500×500 在进程生命周期内只跑一次。
// 返回的切片是只读的，调用方**不要**修改（需要过滤请基于它自建集合）。
func coastalPlainCandidates(continent int) [][2]int {
	coastalIdx.mu.RLock()
	if coastalIdx.ready {
		byC := coastalIdx.byContinent
		coastalIdx.mu.RUnlock()
		// 快路径：索引已建好，直接取该洲
		if byC == nil {
			return nil
		}
		return byC[continent]
	}
	coastalIdx.mu.RUnlock()

	coastalIdx.mu.Lock()
	defer coastalIdx.mu.Unlock()
	// 双检：可能已被其它 goroutine 建好
	if coastalIdx.ready {
		return coastalIdx.byContinent[continent]
	}

	byC := map[int][][2]int{}
	// ★ 枚举整个世界的坐标，按地形分类。
	//   ezfyTerrainEx 内部要判洲 + 判 8 邻域海洋，是整个流程最贵的部分 ——
	//   这里只跑一次，之后永久复用。
	for x := 1; x < ezfyWorldSize; x++ {
		for y := 1; y < ezfyWorldSize; y++ {
			if ezfyTerrainEx(x, y) != ezfyTerrainCoastalPlain {
				continue
			}
			c := ezfyContinentOf(x, y)
			byC[c] = append(byC[c], [2]int{x, y})
		}
	}
	coastalIdx.byContinent = byC
	coastalIdx.ready = true
	return byC[continent]
}

// pickCoastalPos 在指定洲内随机取一个「未被占用、未排除」的沿海平原坐标。
//
// occupied / excluded 为 nil 时表示不做该项过滤。
// 返回 ok=false 表示该洲内没有可用位置。
//
// ★ 复杂度：候选集是缓存的，本函数只做一次线性过滤 + 随机取点，
//
//	与「全图枚举」相比省掉了上百万次形状判定。
func pickCoastalPos(continent int, occupied map[[2]int]bool, excluded map[[2]int]bool) (int, int, bool) {
	cands := coastalPlainCandidates(continent)
	if len(cands) == 0 {
		return 0, 0, false
	}
	// 先随机起点扫一遍，避免总是「按 x,y 顺序」拿到同一个角落的位置
	start := 0
	if len(cands) > 1 {
		start = rand.Intn(len(cands))
	}
	for i := 0; i < len(cands); i++ {
		p := cands[(start+i)%len(cands)]
		if excluded != nil && excluded[p] {
			continue
		}
		if occupied != nil && occupied[p] {
			continue
		}
		return p[0], p[1], true
	}
	return 0, 0, false
}
