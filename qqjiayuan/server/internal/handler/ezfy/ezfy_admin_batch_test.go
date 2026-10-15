package ezfy

import (
	"strings"
	"testing"
)

// TestAdminBattleQueueActiveOnly 管理端「战斗队列」= 所有**进行中**的军队行动。
//
// ★★ 2026-10-09 ：
//
//	① 把**出征队列**也并进来（主表从 ezfy_battle 换成 ezfy_order）；
//	② **已结束的不要**（战报模块能查）→ 只查 出征(0)/驻守(1)/返回(2)/战斗中(5)/等待(6)；
//	③ 修卡顿：原来每行 3 条 SQL（攻方昵称 + 守方昵称 + 订单状态）→ 改成批量取。
func TestAdminBattleQueueActiveOnly(t *testing.T) {
	src := strings.ReplaceAll(rawFile(t, "ezfy_admin_battle.go"), "\r\n", "\n")
	for _, want := range []string{
		"var ezfyAdminActiveOrderStatus = []int{0, 1, 2, ezfyOrderStatusBattle, ezfyOrderStatusWaiting}",
		`h.DB.Model(&model.EzfyOrder{}).Where("status IN ?", ezfyAdminActiveOrderStatus)`,
		"h.ezfyAdminNamesBatch(uids)",                                // 批量昵称（修 N+1）
		"battleByOrder[bs[i].OrderId] = bs[i]",                       // 批量战场
		`h.DB.Select("id, user_id, name").Where("id IN ?", cityIds)`, // 批量城池（守方）
		"ezfyAdminOrderStatusName(o.Status)",                         // 状态中文（出征/返回/等待）
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("管理端战斗队列改造缺失（缺 %s）", want)
		}
	}
	// 列表函数里不能再有「逐行查昵称」的 N+1
	i := strings.Index(src, "func (h *EzfyAdmin) AdminEzfyBattles(")
	j := strings.Index(src, "// ezfyBattleTargetTypeName")
	if i < 0 || j <= i {
		t.Fatal("找不到 AdminEzfyBattles 函数区间")
	}
	body := src[i:j]
	for _, bad := range []string{"h.ezfyAdminName(b.UserID)", "h.ezfyAdminName(b.DefUserID)"} {
		if strings.Contains(body, bad) {
			t.Fatalf("列表里还有 N+1（逐行查昵称）：%s", bad)
		}
	}
}

// TestAdminEzfyPlayersNoNPlusOne 管理端「玩家信息管理」列表禁止逐行查库。
//
// ★★ 2026-10-15 用户反馈：「玩家信息管理 分页查询严重卡顿，这才 200 多个人，切分页大小时更严重」。
//
//	根因不是数据量大，而是**查询条数被行数放大**：原来每一行都要打两条库 ——
//	  ① `h.home().First(&u, p.UserID)` 取家园昵称/家园号（跨库）
//	  ② `COUNT(*) FROM ezfy_city WHERE user_id = ?` 数城池
//	→ size=100 就是 **200 条往返**；线上 1核1G + RDS 跨 WAN 必然卡到几十秒。
//	实测（MySQL general_log，size=100）：旧版 users=100 + ezfy_city=100 条；
//	  新版固定 4 条业务查询（分页主查询 / 最后登陆 / 家园账号 / 城池数 GROUP BY），与 size 无关。
//
// 这里静态锁死：① 函数体里不得出现逐行 First/Count；② 三样附属数据必须走 `IN ?` 批量。
func TestAdminEzfyPlayersNoNPlusOne(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_admin.go", "func (h *EzfyAdmin) AdminEzfyPlayers(")
	// ★ 先剥掉注释行再断言：函数里的说明注释会引用「旧写法」的原文，
	//   不剥会把注释当成真代码误报（本条注释自己也写了 old 写法）。
	code := stripGoComments(body)
	for _, bad := range []string{
		"h.home().First(&u, p.UserID)",
		`h.DB.Model(&model.EzfyCity{}).Where("user_id = ?", p.UserID).Count(&cities)`,
	} {
		if strings.Contains(code, bad) {
			t.Fatalf("玩家列表里还有 N+1（逐行查库）：%s\n分页越大越慢，必须改成 IN 批量取", bad)
		}
	}
	body = code
	for _, want := range []string{
		`h.home().Select("id, username, nickname").Where("id IN ?", uids)`, // 家园账号：一次 IN
		`Where("game_id IN ?", uids)`,                                      // 最后登陆：一次 IN
		`Select("user_id, COUNT(*) AS cnt")`,                               // 城池数：一次 GROUP BY
		`Group("user_id")`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("玩家列表的批量取数缺失（缺 %s）", want)
		}
	}
}

// TestAdminEzfyItemGrantPlayersNoNPlusOne 「发放道具」目标玩家搜索同样禁止逐行查家园号
// （最多 20 行 → 原来 20 条跨库往返）。
func TestAdminEzfyItemGrantPlayersNoNPlusOne(t *testing.T) {
	body := stripGoComments(ezfyFuncBody(t, "ezfy_admin.go", "func (h *EzfyAdmin) AdminEzfyItemGrantPlayers("))
	if strings.Contains(body, "h.home().First(&u, p.UserID)") {
		t.Fatal("发放道具目标搜索里还有 N+1（逐行查家园号）：应改成 `Where(\"id IN ?\", gids)` 一次批量")
	}
	if !strings.Contains(body, `Where("id IN ?", gids)`) {
		t.Fatal("发放道具目标搜索必须一次 IN 批量取家园号")
	}
}

// stripGoComments 去掉 Go 源码里的整行注释（`//` 开头，允许前置空白），
// 供「禁止出现某种写法」的静态断言使用 —— 免得解释性注释里引用的旧写法被误判成真代码。
func stripGoComments(src string) string {
	var b strings.Builder
	for _, ln := range strings.Split(src, "\n") {
		if strings.HasPrefix(strings.TrimSpace(ln), "//") {
			continue
		}
		b.WriteString(ln)
		b.WriteString("\n")
	}
	return b.String()
}

// TestGiveResourcesSettlesFirst 管理端「发放资源」前必须先懒结算，并返回实际到账量。
//
// ★★ 2026-10-09 用户反馈「发放资源做的不够好」：原来直接读库改 + saveCityRes，
// 会把「上次落库到现在」的产量覆盖掉（与交易所下架同一个坑）；且只报「已发放」，
// 被资源最大值封顶时管理员看不出来。
func TestGiveResourcesSettlesFirst(t *testing.T) {
	api := strings.ReplaceAll(rawFile(t, "ezfy_api.go"), "\r\n", "\n")
	i := strings.Index(api, "func (h *EzfyHandler) giveResourcesNoCap(")
	if i < 0 {
		t.Fatal("找不到 giveResourcesNoCap")
	}
	body := api[i:]
	if k := strings.Index(body, "\n}\n"); k > 0 {
		body = body[:k]
	}
	ci := strings.Index(body, "h.calcResource(&city)")
	si := strings.Index(body, "h.saveCityRes(&city)")
	if ci < 0 {
		t.Fatalf("发放资源前没有先懒结算（会吃掉产量）：\n%s", body)
	}
	if si < 0 || ci > si {
		t.Fatalf("calcResource 必须在 saveCityRes 之前：\n%s", body)
	}
	if !strings.Contains(body, "[5]int64") {
		t.Fatalf("giveResourcesNoCap 应返回实际到账量（5 项）：\n%s", body)
	}
}
