package handler

import (
	"testing"
	"time"
)

// TestCoastalIndexFasterThanFullEnum 证明「沿海平原缓存索引」比原来的
// 「每次请求全图枚举」快若干数量级 —— 这是 2026-09-21 线上事故的第 3 个热点。
//
// 原实现（已删除）：每次调用都
//
//	for x := 1; x < ezfyWorldSize; x++ { for y := 1; y < ezfyWorldSize; y++ { ok(x,y) } }
//
// 而 ok() 内部跑 ezfyContinentOf + ezfyTerrainEx(含 8 邻域判海洋)，
// 单次调用上百万次形状判定，1 核机器要吃满 CPU 几百毫秒~数秒。
func TestCoastalIndexFasterThanFullEnum(t *testing.T) {
	fullEnum := func(needCoastal bool) int {
		n := 0
		for x := 1; x < ezfyWorldSize; x++ {
			for y := 1; y < ezfyWorldSize; y++ {
				if x < 1 || y < 1 || x >= ezfyWorldSize || y >= ezfyWorldSize {
					continue
				}
				if ezfyContinentOf(x, y) != 1 {
					continue
				}
				ti := ezfyTerrainEx(x, y)
				if needCoastal {
					if ti != ezfyTerrainCoastalPlain {
						continue
					}
				} else if ti != 1 && ti != ezfyTerrainCoastalPlain {
					continue
				}
				n++
			}
		}
		return n
	}

	// 预热（避免把首次调用的 map/几何初始化算进去）
	ezfyContinentOf(1, 1)

	t0 := time.Now()
	oldCount := fullEnum(true)
	oldDur := time.Since(t0)
	t.Logf("旧实现：全图枚举沿海平原 = %d 格，耗时 %v", oldCount, oldDur)

	// 第一次会建索引（本来就是一次全图枚举，可以接受：进程内只跑一次）
	t1 := time.Now()
	cands := coastalPlainCandidates(1)
	buildDur := time.Since(t1)
	t.Logf("首次建索引：%d 格，耗时 %v", len(cands), buildDur)

	if len(cands) != oldCount {
		t.Fatalf("索引结果与全图枚举不一致：index=%d enum=%d", len(cands), oldCount)
	}

	// 再取 1000 次，应该几乎零成本（这是每座城实际走的路径）
	t2 := time.Now()
	for i := 0; i < 1000; i++ {
		_ = coastalPlainCandidates(1)
	}
	cachedDur := time.Since(t2)
	t.Logf("缓存后取 1000 次：%v（平均 %v/次）", cachedDur, cachedDur/1000)

	if cachedDur > oldDur {
		t.Fatalf("缓存没有起作用：1000 次缓存取值 %v 竟比 1 次全图枚举 %v 还慢",
			cachedDur, oldDur)
	}
}

// TestPickCoastalPosRespectsExcluded 验证取点时确实跳过「已占用」与「已排除」坐标。
func TestPickCoastalPosRespectsExcluded(t *testing.T) {
	cands := coastalPlainCandidates(1)
	if len(cands) < 3 {
		t.Skipf("洲 1 沿海平原太少(%d)，跳过", len(cands))
	}
	// 把除最后 3 格以外的全部排除，剩下的必须落在那 3 格里
	keep := map[[2]int]bool{}
	for _, p := range cands[len(cands)-3:] {
		keep[p] = true
	}
	excluded := map[[2]int]bool{}
	for _, p := range cands {
		if !keep[p] {
			excluded[p] = true
		}
	}
	for i := 0; i < 20; i++ {
		x, y, ok := pickCoastalPos(1, nil, excluded)
		if !ok {
			t.Fatalf("还有 3 格可用却返回 ok=false")
		}
		if !keep[[2]int{x, y}] {
			t.Fatalf("取到了被排除的坐标 (%d,%d)", x, y)
		}
	}

	// 全部占用 → 必须 ok=false
	occupied := map[[2]int]bool{}
	for _, p := range cands {
		occupied[p] = true
	}
	if _, _, ok := pickCoastalPos(1, occupied, nil); ok {
		t.Fatal("全部被占用时仍返回了可用坐标")
	}
}

// TestCoastalIndexInvalidate 验证地图配置变更后索引会重建。
func TestCoastalIndexInvalidate(t *testing.T) {
	_ = coastalPlainCandidates(1)
	coastalIdx.mu.RLock()
	ready := coastalIdx.ready
	coastalIdx.mu.RUnlock()
	if !ready {
		t.Fatal("首次调用后索引应已建立")
	}
	ezfyInvalidateCoastalIndex()
	coastalIdx.mu.RLock()
	ready = coastalIdx.ready
	coastalIdx.mu.RUnlock()
	if ready {
		t.Fatal("ezfyInvalidateCoastalIndex 后索引应失效")
	}
}
