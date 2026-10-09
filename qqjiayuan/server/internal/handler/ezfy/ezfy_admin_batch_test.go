package ezfy

import (
	"strings"
	"testing"
)

// TestAdminBattleQueueActiveOnly 管理端「战斗队列」= 所有**进行中**的军队行动。
//
// ★★ 2026-10-09 用户要求：
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
