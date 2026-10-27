package main

// ★★ 2026-10-27 用户第二次反馈「家园库为啥还会创建二战的表」：
//
//	10-23 那次只是把「写死 false」改成「按配置判定」（配了 ezfy_mysql 才跳过），
//	但线上家园服的配置里根本没写 ezfy_mysql（家园服本来就不连二战库），
//	于是 EzfyUsesOwnDB=false → skip 分支的 EnsureEzfy* 链又把
//	ezfy_ransom / ezfy_pvp_* / ezfy_bot / ezfy_title 等 20 张表建回了家园库。
//
//	现在改为 **home 模式恒为 true**：家园服既不连 ezfyDB、也不注册二战路由，
//	二战表只由 wwii 模式的 seed.RunEzfy 在二战库上建。本测试锁死这个结论。
//
// ⚠️ 别再改回「按配置判定」或「写死 false」—— 那两条路都已经在线上翻过车。

import (
	"os"
	"strings"
	"testing"
)

func TestHomeModeDoesNotCreateEzfyTables(t *testing.T) {
	b, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("读 main.go 失败: %v", err)
	}
	src := string(b)

	if strings.Contains(src, "seed.EzfyUsesOwnDB = false") {
		t.Fatal("runHomeMode 又把 EzfyUsesOwnDB 写死 false —— 家园库会被继续建 ezfy_* 残留表")
	}
	// 2026-10-27：按配置判定也不行。线上家园服没配 ezfy_mysql，判定恒为 false，
	// 结果 20 张表又被建回家园库。home 模式必须无条件不碰二战表。
	if strings.Contains(src, "seed.EzfyUsesOwnDB = cfg.") {
		t.Fatal("runHomeMode 又按配置判定 EzfyUsesOwnDB —— 家园服配置里没有 ezfy_mysql 时" +
			"会退化成 false，二战表会被建进家园库")
	}
	if !strings.Contains(src, "seed.EzfyUsesOwnDB = true") {
		t.Fatal("runHomeMode 没有把 EzfyUsesOwnDB 置为 true（家园库会继续被建二战表）")
	}
	// 下面这段 else-if 分支保留着（恒不执行），别删：删了会让「单库模式」的补列逻辑
	// 变成编译期消失，排查历史问题时看不出痕迹。
	if !strings.Contains(src, "if !seed.EzfyUsesOwnDB {") {
		t.Fatal("EzfyUsesOwnDB 的单库分支不见了（历史上这条分支负责家园库补列）")
	}

	// 清理存量残留的运维工具要在（默认 dry-run，--apply 才删）。
	if _, err := os.Stat("cmd/ezfypurgehome/main.go"); err != nil {
		t.Fatal("缺少清理工具 cmd/ezfypurgehome（家园库二战残留表清理）")
	}
}
