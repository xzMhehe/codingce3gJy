package ezfy

import (
	"strings"
	"testing"
)

// ============ 军衔维护「查询卡顿」护栏（★ 2026-10-30 用户反馈）============
//
// 用户反馈：管理系统 → 军衔维护 → 玩家军衔维护，查询卡顿。
//
// 根因是**逐行 N+1**：每行 1 次 ezfy_city COUNT + 1 次 ezfy_profile 查询 +
// 1 次**跨库**家园 users 查询 → 每页 20 行 60 条 SQL，100 条/页 300 条。
//
// 本文件守住：列表接口只允许**批量**查询（城市数一次 GROUP BY、家园号码一次 IN），
// 不允许再出现「循环里按单个 uid 查库」的写法。

// TestAdminRankPlayersBatched 玩家军衔列表：必须批量取城市数与家园号码。
func TestAdminRankPlayersBatched(t *testing.T) {
	body := ezfyStripCommentLines(strings.ReplaceAll(
		ezfyFuncBody(t, "ezfy_admin_rank.go", "func (h *EzfyAdmin) AdminEzfyRankPlayers("), "\r\n", "\n"))

	for _, want := range []string{
		// ① 城市数：一次 GROUP BY（按页内 uid 集合）
		`Select("user_id, COUNT(*) AS cnt")`,
		`Where("user_id IN ?", uids).Group("user_id")`,
		// ② 家园号码：一次 IN 批量
		`h.home().Select("id, username").Where("id IN ?", uids)`,
		// ③ 排序补 id 兜底（同分翻页稳定）；id 也 DESC 才能吃 idx_profile_prestige 反向扫描
		`q.Order("prestige DESC, id DESC")`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("AdminEzfyRankPlayers 缺少批量查询 %q：\n%s", want, body)
		}
	}

	// 禁止回退成逐行查询
	for _, bad := range []string{
		`.Where("user_id = ?", p.UserID).Count(&cityCount)`,
		`h.ezfyAdminName(`,
		`h.ezfyAdminNameEx(`,
	} {
		if strings.Contains(body, bad) {
			t.Fatalf("AdminEzfyRankPlayers 里又出现逐行查询 %q（N+1 回归，查询会卡）", bad)
		}
	}
}

// TestAdminPrestigeRankCityCountBatched 军衔声望榜：实际城数也必须批量（同一类 N+1）。
func TestAdminPrestigeRankCityCountBatched(t *testing.T) {
	body := ezfyStripCommentLines(strings.ReplaceAll(
		ezfyFuncBody(t, "ezfy_admin_rank.go", "func (h *EzfyAdmin) AdminEzfyPrestigeRank("), "\r\n", "\n"))
	if !strings.Contains(body, `Where("user_id IN ?", pageUIDs).Group("user_id")`) {
		t.Fatalf("AdminEzfyPrestigeRank 的城市数应批量 GROUP BY：\n%s", body)
	}
	if strings.Contains(body, `.Where("user_id = ?", p.UserID).Count(&cityCount)`) {
		t.Fatal("AdminEzfyPrestigeRank 又出现逐行 COUNT（N+1 回归）")
	}
}

// TestAdminRanksPlayerCount 军衔配置页：玩家数用 SQL 聚合，且等级下标不能错位。
func TestAdminRanksPlayerCount(t *testing.T) {
	body := ezfyStripCommentLines(strings.ReplaceAll(
		ezfyFuncBody(t, "ezfy_admin_rank.go", "func (h *EzfyAdmin) AdminEzfyRanks("), "\r\n", "\n"))
	// ⚠️ `rank` 是 MySQL 8 保留字 → 手写 SQL 必须加反引号（裸写会 1064，被静默忽略后玩家数全 0）
	if !strings.Contains(body, "Where(\"`rank` > 0\").Group(\"`rank`\")") {
		t.Fatalf("AdminEzfyRanks 的玩家数应直接在库里按 rank 分组计数（且 `rank` 要加反引号）：\n%s", body)
	}
	if strings.Contains(body, `Where("rank > 0")`) || strings.Contains(body, `Group("rank")`) {
		t.Fatal("`rank` 是 MySQL 8 保留字，裸写会语法错误 → 必须加反引号")
	}
	// ★ 修 off-by-one：cnt 键是等级（1 起），取值必须 cnt[i+1]
	if strings.Contains(body, `"player_count": cnt[i],`) {
		t.Fatal("player_count 又用 0 起的下标取值（每个军衔会显示上一级的玩家数）")
	}
	if !strings.Contains(body, `"player_count": cnt[i+1],`) {
		t.Fatalf("player_count 必须按军衔等级（1 起）取值：\n%s", body)
	}
}

// TestProfilePrestigeIndex 排序索引：ORDER BY prestige DESC 分页必须有索引。
func TestProfilePrestigeIndex(t *testing.T) {
	seedSrc := readFrontFile(t, "../../seed/seed.go")
	if !strings.Contains(seedSrc, "CREATE INDEX idx_profile_prestige ON ezfy_profile(prestige)") {
		t.Fatal("seed/seed.go 缺少 ezfy_profile(prestige) 索引（军衔列表/声望榜排序会全表 filesort）")
	}
}
