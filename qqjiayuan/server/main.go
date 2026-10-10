package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/config"
	"qqjiayuan/server/internal/router"
	"qqjiayuan/server/internal/seed"
	"qqjiayuan/server/pkg/database"
)

// timedStep ★ 2026-10-10 启动链耗时打点：用户反馈「部署完还要等 1~3 分钟服务才起」，
// 之前只能猜是哪步慢。现在每步打一行日志，部署后看 server.log 就能定位耗时大头
// （预期：EnsureEzfySchema 已从近两千次跨 WAN 查询压到 2 次，应在秒级）。
func timedStep(name string, fn func()) {
	t0 := time.Now()
	fn()
	log.Printf("[启动链] %s 耗时 %v", name, time.Since(t0).Round(time.Millisecond))
}

// ezfySplitDBName 自动识别用的二战独立库名（固定就这一个）。
const ezfySplitDBName = "qq_ezzt"

// detectEzfySplitDB 同实例上是否已有「二战独立库」且里面已经有二战数据。
//
// ★★ 2026-10-10 加这个是为了让拆库能**只靠部署**完成：tools/deploy.bat 解压时
// `--exclude='Linuxbushu/server/config.yaml'` 且把服务器上已有 config.yaml 备份还原，
// 所以仓库里加的 `ezfy_mysql` 段根本带不上去。这里改成启动时探测：
// 同实例存在 `qq_ezzt` 且 `ezfy_city` 有数据 → 自动启用独立库。
//
// ★ 安全：**只读探测，绝不创建**（`database.Init` 才会建库，这里只查 information_schema）。
//
//	库不存在 / 是空的 → 返回空串，保持单库模式，行为与以前完全一致。
//	显式配置了 `ezfy_mysql` 时根本不会走到这里（调用方先判 EzfySplit）。
func detectEzfySplitDB(homeDB *gorm.DB) string {
	var hasTable int64
	if err := homeDB.Raw(`SELECT COUNT(*) FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = 'ezfy_city'`, ezfySplitDBName).Scan(&hasTable).Error; err != nil {
		return ""
	}
	if hasTable == 0 {
		return ""
	}
	var rows int64
	if err := homeDB.Raw("SELECT COUNT(*) FROM `" + ezfySplitDBName + "`.ezfy_city").Scan(&rows).Error; err != nil {
		return ""
	}
	if rows == 0 {
		return ""
	}
	return ezfySplitDBName
}

func main() {
	cfgPath := flag.String("config", "config.yaml", "配置文件路径")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("读取配置失败: %v", err)
	}

	// ★★ 2026-10-10 双数据源（多数据源）：家园库 + 二战独立库。
	//   homeDB = 家园库（qq_jiayuan）：账号/角色/权限/私信/设置 + 其它小游戏
	//   ezfyDB = 二战库（qq_ezzt）  ：全部 ezfy_* + 二战自己的家信(private_messages)与设置
	//   未配 ezfy_mysql（或与家园同库）时退化为单库模式，行为与以前完全一致。
	homeDB := database.Init(&cfg.Mysql)
	// ★★ 2026-10-10 未显式配置 ezfy_mysql 时，**自动识别**同实例上是否已有二战独立库：
	//   有（且里面已有二战数据）→ 自动启用独立库模式。
	//   目的：让「只部署、不改服务器 config.yaml」也能完成拆库
	//   （tools/deploy.bat 解压时排除了 config.yaml 并原样还原，配置改不动）。
	//   ★ 只认「已经存在且有数据」的库，**不会凭空创建**；显式配置的 ezfy_mysql 优先级更高。
	if !cfg.EzfySplit() {
		if name := detectEzfySplitDB(homeDB); name != "" {
			cfg.EzfyMysql = config.MysqlConfig{DBName: name} // 其余字段留空 → EzfyDBConfig 自动沿用 mysql
			log.Printf("自动识别到二战独立库 %s（同实例、已有数据）→ 启用独立库模式", name)
		}
	}
	seed.EzfyUsesOwnDB = cfg.EzfySplit()
	ezfyDB := homeDB
	if seed.EzfyUsesOwnDB {
		ezfyCfg := cfg.EzfyDBConfig()
		ezfyDB = database.Init(&ezfyCfg)
		log.Printf("★ 二战数据库模式：独立库 %s（家园库 %s）", cfg.EzfyMysql.DBName, cfg.Mysql.DBName)
	} else {
		log.Printf("★ 二战数据库模式：单库（与家园同库 %s）—— 如需拆库，"+
			"在 config.yaml 加 ezfy_mysql.dbname，或先建好名为 qq_ezzt 的库并搬入二战数据", cfg.Mysql.DBName)
	}

	// ★★ 2026-10-10 道具 cfg_id 全体迁到 1001+（与装备配置 1~35 彻底不重叠，根除「同 ID 混淆」）。
	//   必须跑在 seed **之前**：种子里的道具已按新号 1001+ 灌，若老号行还在，
	//   库里会同时存在老号行和新号行（商城列表出现重复道具）。
	//   ★ 失败必须中止启动：带着老号继续跑，种子会再灌一套新号配置 → 商城道具重复。
	//   独立库模式下由 RunEzfy 在二战库上做（这里跳过，免得又去改家园库里那份没人用的副本）。
	if !seed.EzfyUsesOwnDB {
		timedStep("EnsureEzfyItemIDsHighRange", func() {
			if err := seed.EnsureEzfyItemIDsHighRange(homeDB); err != nil {
				log.Fatalf("道具 cfg_id 改号失败，拒绝启动（避免道具配置出现老号/新号两套）：%v", err)
			}
		})
	}

	// 连接的是「已被别的实例灌好数据的共享库」时(seed.skip: true)，跳过全量初始化，
	// 否则每台节点启动都会对已填充的表重跑配置 INSERT(重复主键/跨 WAN 挂起)。
	if !cfg.Seed.Skip {
		seed.Run(homeDB, cfg.Server.WebDir+"/static")
	} else if !seed.EzfyUsesOwnDB {
		// 单库模式（二战表在家园库）下的「幂等补列」链。独立库模式下这些改由 RunEzfy 承担。
		// ★★ 2026-10-09 通用补列（放在最前）：把「model 里有、库里没有」的 ezfy 列**全部**
		//   幂等补上。这是第 5 次踩「skip 库缺列 → 写入静默失败」之后加的兜底 ——
		//   之前每次加字段都要手写一个 EnsureXxxColumns，漏一次就出线上事故
		//   （officer.source / battle.atk_lock / order.auto_battle / report.type_name）。
		//   只 ADD COLUMN，不删列不改类型，见 seed/ezfy_schema.go。
		timedStep("EnsureEzfySchema", func() { seed.EnsureEzfySchema(homeDB) })
		// ★ 2026-10-05 多机共享库跳过全量 seed 时，新配置列仍要幂等补上
		//   （gold_prod_mult 等），否则管理端保存报 Unknown column / 黄金产量归零。
		timedStep("EnsureEzfyLimitColumns", func() { seed.EnsureEzfyLimitColumns(homeDB) })
		// ★★ 2026-10-07 线上事故：ezfy_ransom 赎城请求表只注册在 seed.Run 的 AutoMigrate 里，
		//   skip 分支不补 → 查询/发起赎城直接报 Error 1146 表不存在。
		timedStep("EnsureEzfyRansomTable", func() { seed.EnsureEzfyRansomTable(homeDB) })
		// ★★ 2026-10-06 线上事故：ezfy_officer.source 同样只由 AutoMigrate 建列，
		//   skip 分支不补 → 所有军官 INSERT 报 1054、战俘/招募全部静默失败。
		timedStep("EnsureEzfyOfficerColumns", func() { seed.EnsureEzfyOfficerColumns(homeDB) })
		// ★★ 2026-10-06 线上事故：ezfy_battle.atk_lock/def_lock（指挥室锁定列）同样只靠
		//   AutoMigrate 建列，skip 分支不补 → 新二进制开战场 INSERT 报 Unknown column、
		//   指挥室全失效（战斗秒出结果）。必须在这里幂等补列。
		timedStep("EnsureEzfyBattleLockColumns", func() { seed.EnsureEzfyBattleLockColumns(homeDB) })
		// ★ 2026-10-07 交易行 ezfy_exchange.city_id（挂单所在城市）同样只靠 AutoMigrate 建列，
		//   skip 分支不补 → 新二进制挂单 INSERT 报 Unknown column 'city_id'，交易行直接挂不了单。
		timedStep("EnsureEzfyExchangeColumns", func() { seed.EnsureEzfyExchangeColumns(homeDB) })
		// ★★ 2026-10-09 线上隐患：ezfy_order.auto_battle（2026-10-07 自动战斗）同样只靠
		//   AutoMigrate 建列，skip 分支不补 → 共享库上所有出征下单报 Unknown column，
		//   而 createOrder 的 Create 在 goroutine 里且忽略 error → 订单静默丢失、兵/油照扣。
		timedStep("EnsureEzfyOrderColumns", func() { seed.EnsureEzfyOrderColumns(homeDB) })
		// ★ 2026-10-09 宝箱奖池：新加的道具奖品行也只由 seed.Run 补缺（backfillEzfyChestPool），
		//   skip 分支不跑 → 共享库上永远开不出新道具。幂等（只补缺、不动已有行）。
		timedStep("EnsureEzfyChestPool", func() { seed.EnsureEzfyChestPool(homeDB) })
		// ★ 2026-10-10 线上事故：管理端发装备漏拷六项战斗属性 → 战报【守方装备】显示「无」。
		//   全量 seed 里的 repairEquipSnapshots 本来会自愈，但 skip 分支不跑 → 共享库永远修不了。
		//   挂到 skip 路径，任何一台重启都会把存量装备行六项对齐配置池并重建军官 JSON（幂等）。
		timedStep("EnsureEzfyEquipSnapshots", func() { seed.EnsureEzfyEquipSnapshots(homeDB) })
		// ★ 2026-10-05 索引同样要补：多机下只有一台跑全量 seed，另一台走这条 skip 路径，
		//   否则慢接口的复合索引在这台机器的库上永远建不出来（helper 幂等，先到先建）。
		timedStep("EnsureEzfyIndexes", func() { seed.EnsureEzfyIndexes(homeDB) })
		// ★ 2026-10-06 计谋「恫疑虚喝/隐真示假」kind 幂等补配：存量库已有计谋行且 kind=0，
		//   会卡「暂未实现」；全量 seed（seedEzfySchemes 表非空即跳过）与 skip 分支都要跑。
		timedStep("EnsureEzfySchemeKinds", func() { seed.EnsureEzfySchemeKinds(homeDB) })
		// ★ 2026-10-10 首页布局回填：home_layout NULL→0（默认新布局）；玩家 10000 保留老布局。
		timedStep("EnsureEzfyHomeLayout", func() { seed.EnsureEzfyHomeLayout(homeDB) })
	}

	// 家园论坛索引（幂等）：原来挂在 EnsureEzfyIndexes 里，二战拆库后必须在家园库上执行。
	timedStep("EnsureHomeForumIndexes", func() { seed.EnsureHomeForumIndexes(homeDB) })

	// 二战独立库：建表 +（首次）自动把家园库里的二战数据整体搬过来 + 灌种子。
	if seed.EzfyUsesOwnDB {
		timedStep("RunEzfy", func() { seed.RunEzfy(ezfyDB, homeDB) })
	}

	gin.SetMode(gin.ReleaseMode)
	r := router.Setup(homeDB, ezfyDB, cfg)
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	fmt.Println("==========================================")
	fmt.Println("  家园社区 服务端已启动  http://127.0.0.1" + addr)
	fmt.Println("  默认管理员：账号 10000 / 密码 admin123")
	fmt.Println("==========================================")
	if err := r.Run(addr); err != nil {
		log.Fatalf("启动失败: %v", err)
	}
}
