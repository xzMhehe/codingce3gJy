package ezfy

import (
	"strings"
	"testing"
)

// ============ 2026-10-13 一键采集 / 一键收获 卡顿 ============
//
// 原实现对**每一支**采集部队都独立走一遍完整结算，每支要打 10+ 次跨库往返：
//
//	查野地(1) + 原子认领(1) + 查城市(1) + 数宝物库存(1) + 插装备(N) + 写订单(1)
//	+ 插战报(1) + 战报补 city_id 回查(1) + 入城前读城资源(1) + 写城资源(1) + 读城资源(1)
//	+ 清空负重写订单(1) …；另外 `ezfyCarryCap` 每调一次还要查 2 条 SQL（城 user_id + 科技表），
//	而一支部队就要调 3 次。
//
// 玩家带 20 支采集队 = 200+ 次往返；线上是 1核1G + 跨 WAN 的 MySQL，体感就是「点了按钮一直转圈」。
//
// 现在改为「批量收口」：整批共享的只读数据一次预取，可攒批的写攒到最后一次提交。
// 下面用源码断言把这两个不变量钉住，防止以后改回「逐支打库」。

// TestBatchCollectNoPerUnitDBRoundTrip 一键采集：整批只允许常数条 SQL。
func TestBatchCollectNoPerUnitDBRoundTrip(t *testing.T) {
	fn := ezfyFuncBody(t, "ezfy_collect.go", "func (h *EzfyHandler) CollectAll(")

	// 野地必须批量取（一次 WHERE id IN），不得逐支 h.DB.First
	if !strings.Contains(fn, "h.batchWildlands(") {
		t.Fatalf("一键采集没有批量预取野地：\n%s", fn)
	}
	if strings.Contains(fn, "h.DB.First(") {
		t.Fatalf("一键采集里还有逐支的单条查询（h.DB.First）：\n%s", fn)
	}
	// 整批 arrive_time / collect_start 完全相同 → 必须合并成一条 UPDATE
	if !strings.Contains(fn, `Where("id IN ?", okIDs)`) {
		t.Fatalf("一键采集没有把整批「下达采集」合并成一条 UPDATE：\n%s", fn)
	}
	// 资源是否已满只判一次（本接口只处理当前城市）
	if !strings.Contains(fn, "h.ezfyAtResMax(int64(current.ID))") {
		t.Fatalf("一键采集没有把「资源是否已满」提到循环外只判一次：\n%s", fn)
	}
	// 整函数内的 DB 往返次数必须是常数（≤2：查空闲驻军 + 批量下达）
	if n := strings.Count(fn, "h.DB."); n > 2 {
		t.Fatalf("一键采集的跨库往返随部队支数增长了（函数内 h.DB. 出现 %d 次，应 ≤2）：\n%s", n, fn)
	}
}

// TestBatchHarvestNoPerUnitDBRoundTrip 一键收获：整批只允许常数条 SQL + 每支 2 条必需写。
func TestBatchHarvestNoPerUnitDBRoundTrip(t *testing.T) {
	fn := ezfyFuncBody(t, "ezfy_collect.go", "func (h *EzfyHandler) HarvestAll(")

	for _, want := range []string{
		"h.newSettleEnv(orders)",           // 野地 / 城市一次预取
		"h.settleDispatch(uid, order, now, env, lootBatch)", // 批量缓存传进结算
		"h.envHarvestCarry(env, order)",    // 负重入城攒批
		"h.envFlush(env)",                  // 战报 / 装备 / 资源入账一次提交
		`Where("id IN ?", okIDs)`,          // 整批「停止采集」一条 UPDATE
		`Where("id IN ?", cleared)`,        // 整批「清空负重」一条 UPDATE
	} {
		if !strings.Contains(fn, want) {
			t.Fatalf("一键收获缺少批量收口调用 %q：\n%s", want, fn)
		}
	}
	if strings.Contains(fn, "h.DB.First(") {
		t.Fatalf("一键收获里还有逐支的单条查询（h.DB.First）：\n%s", fn)
	}
	// 常数 ≤3：查采集部队 + 批量停止采集 + 批量清空负重
	if n := strings.Count(fn, "h.DB."); n > 3 {
		t.Fatalf("一键收获的跨库往返随部队支数增长了（函数内 h.DB. 出现 %d 次，应 ≤3）：\n%s", n, fn)
	}
}

// TestSettleUsesEnvCaches 结算函数必须走请求级缓存，不得每支重新查库。
func TestSettleUsesEnvCaches(t *testing.T) {
	dispatch := ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) settleDispatch(")
	if !strings.Contains(dispatch, "env *ezfySettleEnv") {
		t.Fatalf("settleDispatch 没有接收批量缓存 env：\n%s", dispatch)
	}
	// ★ 最隐蔽的大头：ezfyCarryCap 每调一次要查 2 条 SQL，而一支部队要调 3 次。
	for _, banned := range []string{"h.ezfyCarryCap(order)", "h.ezfyCarryFull(order)"} {
		if strings.Contains(dispatch, banned) {
			t.Fatalf("settleDispatch 还在逐支查负重（%s，每次 2 条 SQL）：\n%s", banned, dispatch)
		}
	}
	for _, want := range []string{"h.envCarryCap(env, order, uid)", "h.envCarryFull(env, order, uid)",
		"h.envTreasureLeft(env, uid)", "h.envAddEquip(env, city, eq)", "h.envReport(env, uid, 5,"} {
		if !strings.Contains(dispatch, want) {
			t.Fatalf("settleDispatch 没有走请求级缓存（缺 %s）：\n%s", want, dispatch)
		}
	}
	// 战报必须带上 city_id，否则 addReport 会按订单回查一次城市
	if !strings.Contains(dispatch, `"", order.ID, order.CityId)`) {
		t.Fatalf("settleDispatch 的采集战报没有传 city_id（会多一次回查）：\n%s", dispatch)
	}

	partial := ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) settlePartialCollect(")
	if !strings.Contains(partial, "env *ezfySettleEnv") {
		t.Fatalf("settlePartialCollect 没有接收批量缓存 env：\n%s", partial)
	}
	if strings.Contains(partial, "h.ezfyCarryCap(order)") {
		t.Fatalf("settlePartialCollect 还在逐支查负重：\n%s", partial)
	}
}

// TestEnvFlushFallsBackWhenCapped 资源会被上限截断时，合并入库必须回退到逐支口径。
//
// ★ 关键正确性保证：原来「入城前后读两次城资源取差值」，只有实际入账 > 0 才清空部队负重。
// 若把「会被资源最大值截断」的批次也合并入库，就会把本该留在负重里的资源一起清掉。
func TestEnvFlushFallsBackWhenCapped(t *testing.T) {
	env := rawFile(t, "ezfy_settle_env.go")
	if !strings.Contains(env, "h.ezfyHarvestWouldCap(") {
		t.Fatalf("envFlush 没有「会被资源上限截断」的判定：\n%s", env)
	}
	flush := ezfyFuncBody(t, "ezfy_settle_env.go", "func (h *EzfyHandler) envFlush(")
	// 快路径（整城一次入库）与慢路径（逐支）必须都存在
	if !strings.Contains(flush, "快路径") || !strings.Contains(flush, "慢路径") {
		t.Fatalf("envFlush 缺少快/慢两条入城路径：\n%s", flush)
	}
	if !strings.Contains(flush, "h.ezfyHarvestWouldCap(e.cities[cityID], agg)") {
		t.Fatalf("envFlush 没有按城判定是否会被截断：\n%s", flush)
	}
	// 装备 / 战报必须批量落库
	if !strings.Contains(flush, "CreateInBatches") {
		t.Fatalf("envFlush 没有把战报/装备攒批落库：\n%s", flush)
	}
}
