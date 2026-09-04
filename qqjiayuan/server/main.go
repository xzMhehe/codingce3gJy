package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/config"
	"qqjiayuan/server/internal/router"
	"qqjiayuan/server/internal/seed"
	"qqjiayuan/server/pkg/database"
)

func main() {
	cfgPath := flag.String("config", "config.yaml", "配置文件路径")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("读取配置失败: %v", err)
	}

	db := database.Init(&cfg.Mysql)
	seed.Run(db, cfg.Server.WebDir+"/static")

	gin.SetMode(gin.ReleaseMode)
	r := router.Setup(db, cfg)
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	fmt.Println("==========================================")
	fmt.Println("  家园社区 服务端已启动  http://127.0.0.1" + addr)
	fmt.Println("  默认管理员：账号 10000 / 密码 admin123")
	fmt.Println("==========================================")
	if err := r.Run(addr); err != nil {
		log.Fatalf("启动失败: %v", err)
	}
}
