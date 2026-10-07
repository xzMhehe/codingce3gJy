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
		// ★★ 2026-10-07 线上事故：ezfy_ransom 赎城请求表只注册在 seed.Run 的 AutoMigrate 里，
		//   skip 分支不补 → 查询/发起赎城直接报 Error 1146 表不存在。
		seed.EnsureEzfyRansomTable(db)
		// ★★ 2026-10-06 线上事故：ezfy_officer.source 同样只由 AutoMigrate 建列，
		//   skip 分支不补 → 所有军官 INSERT 报 1054、战俘/招募全部静默失败。
		seed.EnsureEzfyOfficerColumns(db)
		// ★★ 2026-10-06 线上事故：ezfy_battle.atk_lock/def_lock（指挥室锁定列）同样只靠
		//   AutoMigrate 建列，skip 分支不补 → 新二进制开战场 INSERT 报 Unknown column、
		//   指挥室全失效（战斗秒出结果）。必须在这里幂等补列。
		seed.EnsureEzfyBattleLockColumns(db)
		// ★ 2026-10-05 索引同样要补：多机下只有一台跑全量 seed，另一台走这条 skip 路径，
		//   否则慢接口的复合索引在这台机器的库上永远建不出来（helper 幂等，先到先建）。
		seed.EnsureEzfyIndexes(db)
	}
	// ★ 2026-10-06 计谋「恫疑虚喝/隐真示假」kind 幂等补配：存量库已有计谋行且 kind=0，
	//   会卡「暂未实现」；全量 seed（seedEzfySchemes 表非空即跳过）与 skip 分支都要跑。
	seed.EnsureEzfySchemeKinds(db)

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
