package ezfy

import (
	"strings"
	"testing"

	"qqjiayuan/server/internal/model"
)

// TestEzfyLootPeriodSuffix 采集宝物播报的期数后缀：1 期不写，多期注明「共结算N期」。
//
// ★ 2026-10-22 用户规则：世界频道只播「最近一期」的宝物，补算多期必须说明白，
// 否则玩家会以为少给了（原来是把 24 期的明细全堆在一条播报里）。
func TestEzfyLootPeriodSuffix(t *testing.T) {
	cases := []struct {
		periods int64
		want    string
	}{
		{1, ""},
		{0, ""},
		{-1, ""},
		{2, "（共结算2期）"},
		{24, "（共结算24期）"},
	}
	for _, c := range cases {
		if got := ezfyLootPeriodSuffix(c.periods); got != c.want {
			t.Errorf("ezfyLootPeriodSuffix(%d) = %q, want %q", c.periods, got, c.want)
		}
	}
}

// TestGatherLootBroadcastPerPeriod 采集宝物世界播报只播「最近一期」。
//
// 防回归（2026-10-22 用户反馈「播报的是总的，改成播报按期，不然太多了」）：
//   - settleDispatch 不能再把 treasureNames（全部期数）塞进播报/批次；
//   - 世界播报与批次都只认 lastLoot（最近一期），并带上结算期数；
//   - 战报里仍然保留全部宝物（玩家要能核对）。
func TestGatherLootBroadcastPerPeriod(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) settleDispatch(")
	for _, want := range []string{
		"lastLoot := []string{}",     // 最近一期
		"lastLoot = cur",             // 只在有掉落的期更新
		"lootBatch[0].periods += int(periods)",
		"for _, tn := range lastLoot", // 批次只收最近一期
		"len(lastLoot), strings.Join(lastLoot", // 单条播报只列最近一期
		"ezfyLootPeriodSuffix(periods)",
		"strings.Join(treasureNames, \", \")", // 战报仍列全部
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("settleDispatch 采集宝物播报没有「只播最近一期」：缺少 %q\n%s", want, body)
		}
	}
	// 批次/播报绝不能再用全量 treasureNames
	for _, bad := range []string{
		"for _, tn := range treasureNames",
		"strings.Join(treasureNames, \"、\")",
	} {
		if strings.Contains(body, bad) {
			t.Fatalf("settleDispatch 仍在把全部期数的宝物塞进播报：%q\n%s", bad, body)
		}
	}

	// flush：合并播报同样要带期数后缀
	flush := ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) ezfyLootBatchFlush(")
	if !strings.Contains(flush, "ezfyLootPeriodSuffix(int64(b.periods))") {
		t.Fatalf("ezfyLootBatchFlush 播报缺少结算期数后缀：\n%s", flush)
	}
	if !strings.Contains(flush, "periods int") && !strings.Contains(flush, "b.periods") {
		t.Fatalf("ezfyLootBatch 没有累计结算期数：\n%s", flush)
	}
}

// TestGatherLootBroadcastSwitch 采集宝物播报开关（二战系统配置，**默认关**）。
//
// ★ 2026-10-26 用户规则：「采集珠宝播报在二战系统配置加个开关吧，默认关，
// 采集不播报也不入播报的库」→ 两个播报出口（单条结算 / 批量合并）都必须先判开关；
// 关掉时既不播报，也不写 ezfy_chat（不调 ezfySysChat 就不会落库）。
func TestGatherLootBroadcastSwitch(t *testing.T) {
	// ① 单条结算：批次/播报整段都在开关里
	body := ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) settleDispatch(")
	if !strings.Contains(body, "if ezfyLootBroadcastOn() {") {
		t.Fatalf("settleDispatch 采集宝物播报没有判开关（关掉仍会播报/入播报库）：\n%s", body)
	}
	// ② 批量合并播报：入口先判开关并直接 return（连昵称查询都省掉）
	flush := ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) ezfyLootBatchFlush(")
	iSw := strings.Index(flush, "if !ezfyLootBroadcastOn() {")
	iChat := strings.Index(flush, "ezfySysChat(")
	if iSw < 0 {
		t.Fatalf("ezfyLootBatchFlush 入口没有判开关：\n%s", flush)
	}
	if iChat < 0 || iSw > iChat {
		t.Fatalf("ezfyLootBatchFlush 的开关判断必须在播报之前：\n%s", flush)
	}
	// ③ 开关默认关：配置缓存没加载时也按「关」处理
	acc := ezfyFuncBody(t, "ezfy_geo.go", "func ezfyLootBroadcastOn() bool {")
	if !strings.Contains(acc, "ezfyLootBroadcastDef == 1") {
		t.Fatalf("ezfyLootBroadcastOn 未加载配置时没有按默认值（关）处理：\n%s", acc)
	}
	// ④ 默认值必须是 0（关）——用户要求「默认关」
	if !strings.Contains(rawFile(t, "ezfy_geo.go"), "ezfyLootBroadcastDef = 0") {
		t.Fatal("采集宝物播报开关默认值不是 0（用户要求默认关）")
	}
	// ⑤ 管理端必须能改（入参 + 落库显式写），否则开关等于没有
	admin := rawFile(t, "ezfy_admin_round9.go")
	for _, want := range []string{
		"`json:\"loot_broadcast_on\"`",
		"lim.LootBroadcastOn = *in.LootBroadcastOn",
		"\"loot_broadcast_on\": lim.LootBroadcastOn",
	} {
		if !strings.Contains(admin, want) {
			t.Fatalf("管理端系统配置没有接通采集宝物播报开关：缺少 %q", want)
		}
	}
	// ⑥ 运行时行为：0 = 关（不播报）、1 = 开；配置未加载也按「关」
	oldLoaded, oldLimit := ezfyCfg.loaded, ezfyCfg.limit
	defer func() { ezfyCfg.loaded, ezfyCfg.limit = oldLoaded, oldLimit }()
	ezfyCfg.loaded = true
	ezfyCfg.limit = model.EzfyCfgLimit{LootBroadcastOn: 0}
	if ezfyLootBroadcastOn() {
		t.Fatal("开关配 0（关）时 ezfyLootBroadcastOn 仍返回 true")
	}
	ezfyCfg.limit = model.EzfyCfgLimit{LootBroadcastOn: 1}
	if !ezfyLootBroadcastOn() {
		t.Fatal("开关配 1（开）时 ezfyLootBroadcastOn 返回 false")
	}
	ezfyCfg.loaded = false
	if ezfyLootBroadcastOn() {
		t.Fatal("配置未加载时采集宝物播报应默认关闭")
	}
}
