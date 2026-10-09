package ezfy

import (
	"strings"
	"testing"
)

// TestWildlandSingleCollectorOnOrderEntry 一块附属野地「同时只能一支队伍采集」的卡控
// 必须做在**下单入口**（createOrder），不能只做在 StartCollect 那一步。
//
// ★★ 2026-10-09 用户反馈「自己的附属野地只能一支队伍采集卡控 bug，现在有人有几百支队伍采集」：
//
//	根因：原来的「一野地一采集队」只写在 StartCollect（开始采集）里，
//	查询口径是 `order_type = 7 AND status = 1 AND arrive_time > 0`。
//	但玩家可以绕过它 —— 在「附属野地 → [采集]」直接下发新的采集订单（order_type=4），
//	这条路径**从不经过 StartCollect**，且 order_type 与那条查询口径不一致 → 拦不到。
//	采集队到达后 processArrive 才把它改写成 order_type=7，此时新部队早已在途。
//	→ 同一坐标堆出几十上百支采集队（线上实测一块野地 29 支）。
func TestWildlandSingleCollectorOnOrderEntry(t *testing.T) {
	src := strings.ReplaceAll(rawFile(t, "ezfy_order.go"), "\r\n", "\n")

	// 取出 createOrder 里 orderType == 4（采集）那一整段
	i := strings.Index(src, "if orderType == 4 {")
	if i < 0 {
		t.Fatal("找不到 createOrder 里的采集(orderType == 4)分支")
	}
	j := strings.Index(src[i:], "if orderType == 7 {")
	if j < 0 {
		t.Fatal("找不到采集分支后面的 orderType == 7 分支（区间定位失败）")
	}
	collectBranch := src[i : i+j]

	if !strings.Contains(collectBranch, "ezfyWildlandOccupiedCount") {
		t.Fatalf("采集下单入口没做「一野地一队伍」卡控（缺 ezfyWildlandOccupiedCount）：\n%s", collectBranch)
	}
	if !strings.Contains(collectBranch, "已有部队") {
		t.Fatalf("采集下单入口的卡控缺少玩家可读的拦截文案：\n%s", collectBranch)
	}

	// 统一口径函数必须覆盖「在途 order_type=4」与「驻守 order_type=7」两种情况
	col := strings.ReplaceAll(rawFile(t, "ezfy_collect.go"), "\r\n", "\n")
	k := strings.Index(col, "func (h *EzfyHandler) ezfyWildlandOccupiedCount(")
	if k < 0 {
		t.Fatal("缺少 ezfyWildlandOccupiedCount（下单与开始采集共用的统一卡控口径）")
	}
	fn := col[k:]
	for _, want := range []string{"order_type = 4", "order_type = 7"} {
		if !strings.Contains(fn, want) {
			t.Fatalf("ezfyWildlandOccupiedCount 口径不完整（缺 %s）：\n%s", want, fn)
		}
	}

	// StartCollect（开始采集）也要用同一套口径，不能各写一份
	sc := col[strings.Index(col, "func (h *EzfyHandler) StartCollect("):]
	sc = sc[:strings.Index(sc, "func (h *EzfyHandler) HarvestAll(")]
	if !strings.Contains(sc, "ezfyWildlandOccupiedCount") {
		t.Fatalf("StartCollect 没与下单入口统一口径（应改用 ezfyWildlandOccupiedCount）：\n%s", sc)
	}
}

// TestMarchQueueCapPerCityNotPerUser 出征队列数必须按**当前城市**司令部等级卡控，
// 且统计口径要覆盖「驻守/采集/返航/战斗/等待」全部未结束队伍。
//
// ★★ 2026-10-09 用户反馈「出征队列数量和当前城市的司令部等级有关，没卡住」：
//
//	原口径 `user_id = ? AND status = 0`（按 user、只数在途）：
//	① 不分城市 → 多城玩家可绕过；
//	② 采集队到达后 status 由 0 变 1，名额立刻释放 → 「采集→到达→再采集」无限刷
//	   （线上单玩家 115 支采集队）。
//	现在必须按 city_id + status IN (0,1,2,5,6) 统计。
func TestMarchQueueCapPerCityNotPerUser(t *testing.T) {
	src := strings.ReplaceAll(rawFile(t, "ezfy_order.go"), "\r\n", "\n")

	// 司令部上限卡控（`int(marchingCnt) >= hq`）之前必须是按 city_id 统计
	anchor := "int(marchingCnt) >= hq"
	i := strings.Index(src, anchor)
	if i < 0 {
		t.Fatal("找不到出征队列上限卡控 `int(marchingCnt) >= hq`")
	}
	// 往回取统计那段（从 marchingCnt 的 Count 处截到卡控点）
	start := strings.Index(src, "Count(&marchingCnt)")
	if start < 0 {
		t.Fatal("找不到 marchingCnt 的 Count 调用")
	}
	// 统计那段可能出现在第 i 处之前；取最后一次出现在卡控之前的
	segEnd := i
	lastCnt := strings.LastIndex(src[:segEnd], "Count(&marchingCnt)")
	if lastCnt < 0 {
		t.Fatal("出征队列统计段里没有 Count(&marchingCnt)")
	}
	segStart := lastCnt - 400
	if segStart < 0 {
		segStart = 0
	}
	stat := src[segStart:lastCnt]

	if strings.Contains(stat, `Where("user_id = ? AND status = 0", uid)`) {
		t.Fatalf("出征队列仍按 user + 只数 status=0 统计（旧 bug 口径）：\n%s", stat)
	}
	if !strings.Contains(stat, "city_id = ?") {
		t.Fatalf("出征队列上限没有按当前城市(city_id)统计：\n%s", stat)
	}
	if !strings.Contains(stat, "status IN (0,1,2,5,6)") {
		t.Fatalf("出征队列统计口径没覆盖驻守/采集/返航/战斗/等待（应 status IN (0,1,2,5,6)）：\n%s", stat)
	}
}
