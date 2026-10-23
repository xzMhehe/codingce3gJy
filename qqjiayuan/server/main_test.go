package main

// ★★ 2026-10-23 用户反馈「家园库为什么还在建二战的表」：
//
//	根因是 runHomeMode 里把 `seed.EzfyUsesOwnDB` **写死 false**，于是家园服每次启动都会跑
//	整条 EnsureEzfy* 链（补列/补表/灌种子），家园库里那套 90+ 张 ezfy_* 残留表不但清不掉，
//	还会被继续建出来。现在改成按配置判定：配了独立二战库（ezfy_mysql.dbname 且与家园库不同）
//	就完全不碰 ezfy_* 表。本测试锁死这个判定，防止以后又被改回去。

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
	if !strings.Contains(src, `seed.EzfyUsesOwnDB = cfg.EzfyMysql.DBName != "" && cfg.EzfyMysql.DBName != cfg.Mysql.DBName`) {
		t.Fatal("runHomeMode 没有按「ezfy_mysql.dbname 且不同于家园库」判定 EzfyUsesOwnDB")
	}
	// 单库部署（没配 ezfy_mysql）必须保持老行为：EzfyUsesOwnDB=false 时仍在家园库建表。
	if !strings.Contains(src, "if !seed.EzfyUsesOwnDB {") {
		t.Fatal("EzfyUsesOwnDB 的单库分支不见了（单库部署会不再建二战表）")
	}

	// 清理存量残留的运维工具要在（默认 dry-run，--apply 才删）。
	if _, err := os.Stat("cmd/ezfypurgehome/main.go"); err != nil {
		t.Fatal("缺少清理工具 cmd/ezfypurgehome（家园库二战残留表清理）")
	}
}
