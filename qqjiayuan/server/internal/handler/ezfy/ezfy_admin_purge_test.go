package ezfy

import (
	"regexp"
	"strings"
	"testing"
)

// ============ 删号「漏表护栏」============
//
// 用户规则（2026-10-15）：「玩家信息管理 → 操作里面的删除 是物理操作，把该玩家的所有数据都删除掉」。
//
// 老实现只清了 16 张表 → 登录账号 / 军官 / 装备 / 交易行 / 邮件 / 好友 / 风云榜 / 宣战 /
// 占领 / 军团 等一大批全留着。改法见 `ezfy_admin_purge.go`（声明式清单 + 单事务）。
//
// 本测试是那两张清单的**护栏**：从 `internal/model/ezfy.go` 抽出**全部** `TableName()`
// 定义的表名，逐个要求「在删号清单里 或 在全局白名单里」——
// 以后新增二战表忘了加删号清单，这里直接失败，不会再出现「删号删不干净」。

// ezfyTableNameRe 匹配 model 里单行写法的 TableName 定义。
var ezfyTableNameRe = regexp.MustCompile(`TableName\(\) string \{ return "([a-z_]+)" \}`)

// TestEzfyPlayerPurgeCoversAllTables 删号清单必须覆盖二战库的每一张表。
func TestEzfyPlayerPurgeCoversAllTables(t *testing.T) {
	src := strings.ReplaceAll(rawFile(t, "../../model/ezfy.go"), "\r\n", "\n")
	matches := ezfyTableNameRe.FindAllStringSubmatch(src, -1)
	if len(matches) < 60 {
		t.Fatalf("从 internal/model/ezfy.go 只解析出 %d 张表（解析规则失效？表名写法变了？）", len(matches))
	}
	// 两张「二战自己用的通用表」：Setting 在 setting.go 里声明 settings；
	// PrivateMessage 没有 TableName()，走 GORM 默认命名 → private_messages。
	names := []string{"settings", "private_messages"}
	for _, m := range matches {
		names = append(names, m[1])
	}

	covered := map[string]string{} // 表名 → 归属说明
	for _, r := range ezfyPurgeByUid {
		covered[r.Table] = "ezfyPurgeByUid"
	}
	for _, r := range ezfyPurgeByCity {
		covered[r.Table] = "ezfyPurgeByCity"
	}
	for _, tb := range ezfyPurgeSpecialTables {
		covered[tb] = "ezfyPurgeSpecialTables（专门逻辑处理）"
	}
	for _, tb := range ezfyPurgeGlobalTables {
		covered[tb] = "ezfyPurgeGlobalTables（与玩家无关，不删）"
	}

	for _, n := range names {
		if covered[n] == "" {
			t.Fatalf("新增的二战表 %q 没有出现在删号清单里 —— 请在 ezfy_admin_purge.go 的 "+
				"ezfyPurgeByUid / ezfyPurgeByCity 里补上，或在 ezfyPurgeGlobalTables 里说明为什么与玩家无关", n)
		}
	}
}

// TestEzfyPlayerPurgeListsSane 两张清单自身的自检：
// ① 同一张表在同一张清单里不得重复；② 不能既在删号清单又在全局白名单（自相矛盾）。
func TestEzfyPlayerPurgeListsSane(t *testing.T) {
	global := map[string]bool{}
	for _, tb := range ezfyPurgeGlobalTables {
		global[tb] = true
	}
	for name, list := range map[string][]ezfyPurgeRule{
		"ezfyPurgeByUid":  ezfyPurgeByUid,
		"ezfyPurgeByCity": ezfyPurgeByCity,
	} {
		seen := map[string]bool{}
		for _, r := range list {
			if seen[r.Table] {
				t.Fatalf("%s 里表 %q 重复了", name, r.Table)
			}
			seen[r.Table] = true
			if global[r.Table] {
				t.Fatalf("表 %q 既在 %s 里（要删）又在 ezfyPurgeGlobalTables 里（不删），自相矛盾", r.Table, name)
			}
			if len(r.Cols) == 0 {
				t.Fatalf("%s 里表 %q 没有归属列", name, r.Table)
			}
		}
	}
}

// TestEzfyPlayerDeleteUsesPurge 删号入口必须走清单化的事务函数，不能退回「一堆手写 Delete」。
func TestEzfyPlayerDeleteUsesPurge(t *testing.T) {
	body := stripGoComments(ezfyFuncBody(t, "ezfy_admin.go", "func (h *EzfyAdmin) AdminEzfyPlayerDelete("))
	if !strings.Contains(body, "h.ezfyPurgePlayerData(uid, cityIds)") {
		t.Fatal("AdminEzfyPlayerDelete 必须调用 ezfyPurgePlayerData（声明式清单 + 单事务）")
	}
	if strings.Contains(body, ".Delete(&model.Ezfy") {
		t.Fatal("AdminEzfyPlayerDelete 又出现手写 GORM Delete 了 —— 逐表手写必然漏表，请加进 ezfy_admin_purge.go 的清单")
	}
}

// TestEzfyPurgeConditionHelpers 条件拼装小工具：占位符个数必须与参数个数一致（否则删错范围）。
func TestEzfyPurgeConditionHelpers(t *testing.T) {
	if got := ezfyPurgeEqCond([]string{"user_id"}); got != "user_id = ?" {
		t.Fatalf("单列 EQ 条件不对: %s", got)
	}
	if got := ezfyPurgeEqCond([]string{"user_id", "friend_id"}); got != "user_id = ? OR friend_id = ?" {
		t.Fatalf("双列 EQ 条件不对: %s", got)
	}
	if got := ezfyPurgeInCond([]string{"city_id", "atk_city_id"}); got != "city_id IN ? OR atk_city_id IN ?" {
		t.Fatalf("双列 IN 条件不对: %s", got)
	}
	if args := ezfyPurgeArgs(uint(7), 2); len(args) != 2 || args[0] != uint(7) || args[1] != uint(7) {
		t.Fatalf("参数复制不对: %v", args)
	}
}
