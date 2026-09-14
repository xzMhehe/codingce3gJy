// 家园社区 - 数据库初始化工具
//
// 作用：建库 + 建表 + 写入种子数据。全部幂等，可以重复执行。
// 后端 server.exe 启动时也会自动建表灌种子，但不会建库；
// 这个工具用来在首次部署前把库准备好，顺便验证 MySQL 连接是否正常。
//
// 用法：
//
//	dbinit.exe                          # 用同目录的 config.yaml
//	dbinit.exe -config D:\x\config.yaml # 指定配置
//	dbinit.exe -static D:\x\web\dist\static  # 指定前端静态目录（可选）
package main

import (
	"flag"
	"fmt"
	"log"

	"qqjiayuan/server/internal/config"
	"qqjiayuan/server/internal/seed"
	"qqjiayuan/server/pkg/database"
)

func main() {
	cfgPath := flag.String("config", "config.yaml", "配置文件路径")
	staticDir := flag.String("static", "", "前端静态目录，用于登记图片素材；默认取 web_dir/static，目录不存在会自动跳过")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("读取配置失败: %v（当前目录下需要有 config.yaml）", err)
	}

	fmt.Println("==========================================")
	fmt.Println("  家园社区 - 数据库初始化工具")
	fmt.Println("==========================================")
	fmt.Printf("  配置文件 : %s\n", *cfgPath)
	fmt.Printf("  MySQL    : %s:%d\n", cfg.Mysql.Host, cfg.Mysql.Port)
	fmt.Printf("  账号     : %s\n", cfg.Mysql.User)
	fmt.Printf("  数据库   : %s\n", cfg.Mysql.DBName)
	fmt.Println("==========================================")
	fmt.Println()

	fmt.Println("[1/2] 建库（已存在则跳过）...")
	if err := database.EnsureDatabase(&cfg.Mysql); err != nil {
		fmt.Println()
		fmt.Println("建库失败。常见原因：")
		fmt.Println("  - MySQL 服务没启动，或 host/port 配错")
		fmt.Println("  - 账号或密码不对")
		fmt.Println("  - 账号没有 CREATE 权限（用 root 执行一次，或先手工建库）")
		fmt.Println()
		log.Fatalf("错误详情: %v", err)
	}
	fmt.Printf("      库 %s 已就绪\n\n", cfg.Mysql.DBName)

	fmt.Println("[2/2] 建表 + 写入种子数据（幂等，数据量大时会慢一些）...")
	db := database.Init(&cfg.Mysql)

	sd := *staticDir
	if sd == "" {
		sd = cfg.Server.WebDir + "/static"
	}
	seed.Run(db, sd)

	fmt.Println()
	fmt.Println("==========================================")
	fmt.Println("  初始化完成")
	fmt.Println("  默认管理员：账号 10000 / 密码 admin123")
	fmt.Println("  接下来可以直接启动 server.exe")
	fmt.Println("==========================================")
}
