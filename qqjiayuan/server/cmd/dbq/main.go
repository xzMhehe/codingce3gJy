// 家园社区 - 开发期临时查库工具（不进生产，用完即删）
//
// 作用：读取同目录的 config.yaml 连库，执行一条只读 SQL 并把结果打成表格。
// 目的：开发/排查时不必把带密码的配置读出来，也不用记 mysql 客户端参数。
//
// 用法：
//
//	go run ./cmd/dbq -sql "SELECT * FROM ezfy_cfg_limit"
//	go run ./cmd/dbq -config ../config.yaml -sql "SHOW TABLES LIKE 'ezfy_%'"
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"gorm.io/gorm"

	"qqjiayuan/server/internal/config"
	"qqjiayuan/server/pkg/database"
)

func main() {
	cfgPath := flag.String("config", "config.yaml", "配置文件路径")
	sqlText := flag.String("sql", "", "要执行的 SQL")
	maxRows := flag.Int("n", 200, "最多打印多少行")
	flag.Parse()

	query := strings.TrimSpace(*sqlText)
	if query == "" {
		// 也支持从 stdin 读，方便多行 SQL
		b, _ := os.ReadFile("/dev/stdin")
		query = strings.TrimSpace(string(b))
	}
	if query == "" {
		log.Fatal("用法: go run ./cmd/dbq -sql \"SELECT ...\"")
	}

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("读取配置失败: %v", err)
	}
	db := database.Init(&cfg.Mysql)

	run(db, query, *maxRows)
}

func run(db *gorm.DB, query string, maxRows int) {
	rows, err := db.Raw(query).Rows()
	if err != nil {
		log.Fatalf("查询失败: %v", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		log.Fatalf("取列名失败: %v", err)
	}

	fmt.Println(strings.Join(cols, " | "))
	fmt.Println(strings.Repeat("-", 60))

	n := 0
	for rows.Next() {
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			log.Fatalf("扫描行失败: %v", err)
		}
		out := make([]string, len(cols))
		for i, v := range vals {
			switch t := v.(type) {
			case nil:
				out[i] = "NULL"
			case []byte:
				out[i] = string(t)
			default:
				out[i] = fmt.Sprint(t)
			}
		}
		fmt.Println(strings.Join(out, " | "))
		n++
		if n >= maxRows {
			fmt.Printf("... (超过 %d 行已截断)\n", maxRows)
			break
		}
	}
	fmt.Printf("(%d 行)\n", n)
}
