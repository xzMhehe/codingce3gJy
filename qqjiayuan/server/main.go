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
	// 连接的是「已被别的实例灌好数据的共享库」时(seed.skip: true)，跳过全量初始化，
	// 否则每台节点启动都会对已填充的表重跑配置 INSERT(重复主键/跨 WAN 挂起)。
	if !cfg.Seed.Skip {
		seed.Run(db, cfg.Server.WebDir+"/static")
	} else {
		// ★ 2026-10-05 多机共享库跳过全量 seed 时，新配置列仍要幂等补上
		//   （gold_prod_mult 等），否则管理端保存报 Unknown column / 黄金产量归零。
		seed.EnsureEzfyLimitColumns(db)
	}

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
