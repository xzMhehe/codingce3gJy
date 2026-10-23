// 家园社区 · 二战风云 —— 清理家园库里的二战残留表（ezfypurgehome）
//
// 背景：二战征途 2026-10-10 起独立成库（config 的 `ezfy_mysql`，如 qq_ezzt），
// 家园库（qq_jiayuan）里那套 `ezfy_*` 表是**拆分前的残留快照**。历史上 home 模式启动时
// 会把 EzfyUsesOwnDB 写死 false，于是每次启动都在家园库上跑一遍 EnsureEzfy* 链路
// （补列/补表/灌种子），残留表不但清不掉，还会被继续建出来。
//
// 2026-10-23 起 home 模式已改为「配了独立二战库就完全不碰 ezfy 表」，本工具用来把
// **已经存在**的残留表清掉。
//
// ★ 运维工具，不参与游戏运行（不进 seed、不自动跑）。
// ★ 默认 dry-run 只打印清单，加 --apply 才真正 DROP。
// ★ 只删 `ezfy_` 前缀的表；`private_messages` / `settings` 这类家园自己也在用的表**不动**。
// ★ 家园库 == 二战库（单库部署）时直接拒绝运行 —— 那套表就是真数据。
//
// 用法：
//
//	# 1) 先看清单（不删任何东西）
//	./ezfypurgehome --config /opt/config.yaml
//
//	# 2) 先备份（强烈建议）
//	mysqldump -h127.0.0.1 -uroot -p --single-transaction qq_jiayuan \
//	  $(mysql -N -B -e "SELECT GROUP_CONCAT(table_name SEPARATOR ' ') FROM information_schema.tables \
//	     WHERE table_schema='qq_jiayuan' AND table_name LIKE 'ezfy\\_%'") > ezfy_home_backup.sql
//
//	# 3) 真删
//	./ezfypurgehome --config /opt/config.yaml --apply
package main

import (
	"flag"
	"fmt"
	"log"

	"qqjiayuan/server/internal/config"
	"qqjiayuan/server/pkg/database"
)

func main() {
	cfgPath := flag.String("config", "config.yaml", "配置文件路径")
	apply := flag.Bool("apply", false, "真正执行 DROP（不加则只打印清单，dry-run）")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("读取配置失败: %v（当前目录下需要有 config.yaml，或用 -config 指定）", err)
	}
	ezfyDBName := cfg.EzfyDBConfig().DBName
	if ezfyDBName == "" {
		ezfyDBName = cfg.Mysql.DBName
	}
	if ezfyDBName == cfg.Mysql.DBName {
		log.Fatalf("家园库与二战库是同一个库（%s）—— 这套 ezfy_* 表就是二战真数据，拒绝清理。\n"+
			"（要清的是「二战已独立库」的部署里家园库上的残留副本，见 config 的 ezfy_mysql.dbname）", cfg.Mysql.DBName)
	}

	db := database.Init(&cfg.Mysql)

	fmt.Println("============================================================")
	fmt.Println("  二战残留表清理（家园库）")
	fmt.Println("============================================================")
	fmt.Printf("  家园库 : %s:%d / %s\n", cfg.Mysql.Host, cfg.Mysql.Port, cfg.Mysql.DBName)
	fmt.Printf("  二战库 : %s（不连、不动）\n", ezfyDBName)
	fmt.Printf("  模式   : %s\n", map[bool]string{true: "★ APPLY（真删）", false: "dry-run（只列清单）"}[*apply])

	type row struct {
		Name string
		Rows int64
		MB   float64
	}
	var list []row
	var totalMB float64
	rows, err := db.Raw(`SELECT table_name, IFNULL(table_rows,0),
			ROUND((IFNULL(data_length,0)+IFNULL(index_length,0))/1024/1024, 2)
		FROM information_schema.tables
		WHERE table_schema = ? AND table_name LIKE 'ezfy\_%'
		ORDER BY table_name`, cfg.Mysql.DBName).Rows()
	if err != nil {
		log.Fatalf("查询表清单失败: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.Name, &r.Rows, &r.MB); err != nil {
			log.Fatalf("读取表清单失败: %v", err)
		}
		totalMB += r.MB
		list = append(list, r)
	}

	if len(list) == 0 {
		fmt.Println("\n家园库里没有 ezfy_* 残留表，无需清理。")
		return
	}
	fmt.Printf("\n发现 %d 张残留表，合计 %.1f MB：\n", len(list), totalMB)
	for _, r := range list {
		fmt.Printf("  %-38s 行数 %-10d %6.2f MB\n", r.Name, r.Rows, r.MB)
	}

	if !*apply {
		fmt.Println("\n[dry-run] 什么都没删。确认无误后加 --apply 执行。")
		fmt.Println("\n★ 建议先备份（把 ezfy_ 表整批 dump 出来，出事可回灌）：")
		fmt.Printf("  mysqldump -h%s -u%s -p --single-transaction --default-character-set=utf8mb4 %s \\\n"+
			"    $(mysql -h%s -u%s -p -N -B -e \"SET SESSION group_concat_max_len=1000000; "+
			"SELECT GROUP_CONCAT(table_name SEPARATOR ' ') FROM information_schema.tables "+
			"WHERE table_schema='%s' AND table_name LIKE 'ezfy\\\\_%%'\") > ezfy_home_backup.sql\n",
			cfg.Mysql.Host, cfg.Mysql.User, cfg.Mysql.DBName,
			cfg.Mysql.Host, cfg.Mysql.User, cfg.Mysql.DBName)
		return
	}

	// 关外键检查再逐张 DROP：残留表之间可能有历史外键，顺序删会互相卡住。
	db.Exec("SET FOREIGN_KEY_CHECKS = 0")
	dropped, failed := 0, 0
	for _, r := range list {
		if err := db.Exec("DROP TABLE IF EXISTS `" + r.Name + "`").Error; err != nil {
			failed++
			fmt.Printf("  [失败] %-38s %v\n", r.Name, err)
			continue
		}
		dropped++
	}
	db.Exec("SET FOREIGN_KEY_CHECKS = 1")

	fmt.Printf("\n完成：已删除 %d 张，失败 %d 张。\n", dropped, failed)
	if failed > 0 {
		log.Fatalf("有 %d 张表删除失败，请检查权限/锁（见上面 [失败] 行）", failed)
	}
	fmt.Println("提示：家园服重启后不会再在家园库建 ezfy_* 表（home 模式已按 ezfy_mysql 判定）。")
}
