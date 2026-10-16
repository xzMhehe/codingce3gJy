package ezfy

import (
	"os"
	"strings"
	"testing"
)

// TestOccupyWildlandClearsOldRowBeforeCreate 「征服野地/寇城 → 占领」必须先清掉**同坐标旧记录**
// 再插入自己的那条，否则会留下同坐标多条野地记录（一个坐标只允许一条）。
//
// ★★ 2026-10-16 用户反馈「征服其他人附属野地胜利后纳入自己附属野地，现在有问题」：
//
//	原实现无条件 `h.DB.Create(&wl)`。若该坐标早已被别人占领（表里已有 city_id=他人城的行），
//	就会同时存在两条记录。线上实测：153 个坐标重复，其中 81 个坐标归属冲突。后果：
//	  · 野地详情 WildlandView / 建新城 CreateCity 用 `First()` → 取到**旧行**（原主人），
//	    于是「地图显示归我、详情却写归属别人、mine=false」，我还能对自己的地反复下单；
//	  · 原主人的「附属野地」列表仍含这块地 → 白占他的野地上限、还能在上面采集；
//	  · len(wildlandList) 虚高 → 有空位却被判「附属野地已达上限」。
func TestOccupyWildlandClearsOldRowBeforeCreate(t *testing.T) {
	src := strings.ReplaceAll(rawFile(t, "ezfy_order.go"), "\r\n", "\n")

	// 定位「征服野地/寇城: 占领」这一段（到「征服玩家城市」为止）
	i := strings.Index(src, "// 征服野地/寇城: 占领")
	if i < 0 {
		t.Fatal("找不到「征服野地/寇城: 占领」注释块")
	}
	j := strings.Index(src[i:], "// 征服玩家城市")
	if j < 0 {
		t.Fatal("找不到占领段后面的「征服玩家城市」分界")
	}
	seg := src[i : i+j]

	clearIdx := strings.Index(seg, "h.ezfyClearWildlandAt(")
	createIdx := strings.Index(seg, "h.DB.Create(&wl)")
	if clearIdx < 0 {
		t.Fatalf("占领野地前没有清理同坐标旧记录（缺 ezfyClearWildlandAt）——\n"+
			"会导致同一坐标出现多条野地记录、归属冲突：\n%s", seg)
	}
	if createIdx < 0 {
		t.Fatalf("占领段里找不到 h.DB.Create(&wl)：\n%s", seg)
	}
	if clearIdx > createIdx {
		t.Fatalf("必须先 ezfyClearWildlandAt 再 Create 新野地行（当前顺序反了）")
	}
}

// TestClearWildlandAtReturnsOrdersBeforeDelete 清理野地行之前，必须先把该野地上**未结束**的
// 采集(4)/驻守(7)部队转返航 —— 直接删行会让部队的 target_id 悬空（凭空消失 / 结算异常）。
func TestClearWildlandAtReturnsOrdersBeforeDelete(t *testing.T) {
	col := strings.ReplaceAll(rawFile(t, "ezfy_collect.go"), "\r\n", "\n")
	k := strings.Index(col, "func (h *EzfyHandler) ezfyClearWildlandAt(")
	if k < 0 {
		t.Fatal("缺少 ezfyClearWildlandAt（占领/放弃野地共用的清理口径）")
	}
	rest := col[k:]
	end := strings.Index(rest, "\n}\n")
	if end < 0 {
		t.Fatal("ezfyClearWildlandAt 函数体定位失败")
	}
	fn := rest[:end]

	// 必须覆盖采集(4) 与 驻守(7) 两种订单
	for _, want := range []string{"order_type = 4", "order_type = 7", "status IN (0,1,5,6,98)"} {
		if !strings.Contains(fn, want) {
			t.Fatalf("ezfyClearWildlandAt 部队口径不完整（缺 %s）：\n%s", want, fn)
		}
	}
	// 转返航（status=2 + return_time）必须出现在删行之前
	retIdx := strings.Index(fn, `"status": 2`)
	delIdx := strings.Index(fn, "Delete(&model.EzfyWildland{})")
	if retIdx < 0 {
		t.Fatalf("ezfyClearWildlandAt 没有把采集/驻守部队转返航（缺 status=2）：\n%s", fn)
	}
	if delIdx < 0 {
		t.Fatalf("ezfyClearWildlandAt 没有删除野地行：\n%s", fn)
	}
	if retIdx > delIdx {
		t.Fatal("ezfyClearWildlandAt 顺序错误：必须先转返航再删野地行")
	}

	// AbandonWildland（放弃野地）也要走同一套口径，不能各写一份
	full := strings.ReplaceAll(rawFile(t, "ezfy.go"), "\r\n", "\n")
	ai := strings.Index(full, "func (h *EzfyHandler) AbandonWildland(")
	if ai < 0 {
		t.Fatal("找不到 AbandonWildland")
	}
	ab := full[ai:]
	if ae := strings.Index(ab, "\n}\n"); ae > 0 {
		ab = ab[:ae]
	}
	if !strings.Contains(ab, "ezfyClearWildlandAt") {
		t.Fatalf("AbandonWildland 没复用 ezfyClearWildlandAt（放弃野地要与占领野地同一口径）：\n%s", ab)
	}
}

// TestWildlandUniqueGuardWired 存量去重 + 唯一索引必须都挂上：
//   - seed.EnsureEzfyWildlandUnique 收口存量（同坐标重复 / 城市格残留）；
//   - 必须在 EnsureEzfyIndexes 之前跑，否则唯一索引建不出来；
//   - 唯一索引 uk_wildland_xy 声明在 EnsureEzfyIndexes 里。
func TestWildlandUniqueGuardWired(t *testing.T) {
	seedSrc := strings.ReplaceAll(rawFile(t, "../../seed/ezfy.go"), "\r\n", "\n")

	uniqIdx := strings.Index(seedSrc, "EnsureEzfyWildlandUnique(db)")
	idxCallIdx := strings.Index(seedSrc, "EnsureEzfyIndexes(db)")
	if uniqIdx < 0 {
		t.Fatal("seedEzfy 里没有调用 EnsureEzfyWildlandUnique（存量脏数据不会收口）")
	}
	if idxCallIdx < 0 {
		t.Fatal("seedEzfy 里没有调用 EnsureEzfyIndexes")
	}
	// 去重必须排在 EnsureEzfyIndexes **之前**（第一次出现即 seedEzfy 内的调用点）——
	// 否则存量重复行会让唯一索引 uk_wildland_xy 建不出来。
	if uniqIdx > idxCallIdx {
		t.Fatal("EnsureEzfyWildlandUnique 调用顺序不对：必须排在 EnsureEzfyIndexes 之前")
	}
	if !strings.Contains(seedSrc, `"uk_wildland_xy", "x,y", true`) {
		t.Fatal("EnsureEzfyIndexes 里没有声明唯一索引 uk_wildland_xy(x,y)")
	}

	// main.go 的 skip 分支（多机共享库）也要跑去重，否则线上永远修不了存量
	mainSrc := strings.ReplaceAll(rawFile(t, "../../../main.go"), "\r\n", "\n")
	if !strings.Contains(mainSrc, "seed.EnsureEzfyWildlandUnique(homeDB)") {
		t.Fatal("main.go 的 seed.skip 分支没有跑 EnsureEzfyWildlandUnique（共享库存量脏数据永远清不掉）")
	}

	// 去重 helper 自身存在且带幂等/安全说明
	b, err := os.ReadFile("../../seed/ezfy_wildland_dedup.go")
	if err != nil {
		t.Fatalf("读 seed/ezfy_wildland_dedup.go 失败: %v", err)
	}
	dedup := string(b)
	for _, want := range []string{"func EnsureEzfyWildlandUnique(", "ezfy_wildland", "order_type = 4", "order_type = 7"} {
		if !strings.Contains(dedup, want) {
			t.Fatalf("EnsureEzfyWildlandUnique 实现不完整（缺 %s）", want)
		}
	}
}
