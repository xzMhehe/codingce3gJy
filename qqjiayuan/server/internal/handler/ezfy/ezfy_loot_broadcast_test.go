package ezfy

import (
	"strings"
	"testing"
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
