package ezfy

import (
	"path/filepath"
	"strings"
	"testing"
)

// ============ 二战管理端「搜索条件」口径护栏（2026-10-15）============
//
// 用户规则（原话）：「二战管理系统里面 所有家园号条件查询 全部换成游戏号 查询」。
//
// 背景：二战有两个「号」——
//
//	家园号 = 家园库 `users.username`（会随「转靓号」变化）；
//	游戏号 = `ezfy_profile.game_uid`（建档时 = 家园ID，之后与 user_id 解耦，玩家在游戏里看到/报客服的就是它）。
//
// 管理端各页原来的 word 搜索会拿输入去**家园库** `users.username` 反查 uid → 用家园号才搜得到，
// 玩家报游戏号反而搜不到。现在统一：
//
//	数字 → 用户ID 或 游戏号（user_id / game_uid）
//	非数字 → 玩家昵称（+ 家园昵称，那本来就不是「家园号条件」）
//
// 本测试静态锁死「管理端不得再按 users.username 过滤」——
// 玩家端（ezfy_friend / ezfy_mail / ezfy_player）**不在此范围**，仍可按家园号找好友/收件人。

// TestAdminSearchNoHomeAccountNumber 管理端 handler 里不得再出现按家园号（users.username）过滤的条件。
func TestAdminSearchNoHomeAccountNumber(t *testing.T) {
	files, err := filepath.Glob("ezfy_admin*.go")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) < 10 {
		t.Fatalf("只匹配到 %d 个 ezfy_admin*.go（文件被改名/移动了？）", len(files))
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src := stripGoComments(strings.ReplaceAll(rawFile(t, f), "\r\n", "\n"))
		for _, bad := range []string{"username = ?", "username LIKE"} {
			if strings.Contains(src, bad) {
				t.Fatalf("%s 里还有按家园号过滤的条件（%q）—— 用户规则：管理端所有家园号条件查询"+
					"都要换成游戏号（ezfy_profile.game_uid）", f, bad)
			}
		}
	}
}

// TestAdminPlayerSearchUsesGameUID 关键页的 word 搜索必须真的按游戏号匹配。
func TestAdminPlayerSearchUsesGameUID(t *testing.T) {
	cases := []struct {
		file string
		sig  string
		want []string
	}{
		{"ezfy_admin.go", "func (h *EzfyAdmin) AdminEzfyPlayers(", []string{"user_id = ? OR game_uid = ?"}},
		{"ezfy_admin.go", "func (h *EzfyAdmin) AdminEzfyItemGrantPlayers(", []string{"user_id = ? OR game_uid = ?"}},
		{"ezfy_admin_peace.go", "func (h *EzfyAdmin) AdminEzfyPeace(", []string{"user_id = ? OR game_uid = ?"}},
		{"ezfy_admin_monitor.go", "func monitorWordUids(", []string{"user_id = ? OR game_uid = ?"}},
		{"ezfy_admin_war.go", "func (h *EzfyAdmin) AdminEzfyWars(", []string{"user_id = ? OR game_uid = ?"}},
	}
	for _, c := range cases {
		body := stripGoComments(ezfyFuncBody(t, c.file, c.sig))
		for _, w := range c.want {
			if !strings.Contains(body, w) {
				t.Fatalf("%s::%s 的搜索没有按游戏号匹配（缺 %q）", c.file, c.sig, w)
			}
		}
	}
}

// TestAdminPeaceWordFilterGrouped 免战页的 word 条件必须与 only 条件**用括号分组**。
//
// ★ 原来写成 `q.Where(...)` 后面多次 `q.Or(...)`，而 only=1 时已有一个
// `user_id IN (...)` 条件 → SQL 的 AND 优先于 OR，拼出来是
//
//	`(only条件 AND user_id=?) OR user_id=? OR game_uid=?`
//
// 于是「只看在免战的玩家 + 搜索」会把不在免战的玩家也带出来。改后必须是一个 Where 内的括号条件。
func TestAdminPeaceWordFilterGrouped(t *testing.T) {
	body := stripGoComments(ezfyFuncBody(t, "ezfy_admin_peace.go", "func (h *EzfyAdmin) AdminEzfyPeace("))
	i := strings.Index(body, `if word != "" {`)
	if i < 0 {
		t.Fatal("找不到免战页的 word 过滤块")
	}
	seg := body[i:]
	if j := strings.Index(seg, "\n\t}"); j > 0 {
		seg = seg[:j]
	}
	if strings.Contains(seg, "q.Or(") {
		t.Fatalf("免战页 word 过滤又用 q.Or 拼条件了（会与 only 条件串味）：\n%s", seg)
	}
	if !strings.Contains(seg, `"user_id = ? OR game_uid = ?"`) {
		t.Fatalf("免战页 word 过滤必须在一个 Where 内按 用户ID/游戏号 匹配：\n%s", seg)
	}
}
