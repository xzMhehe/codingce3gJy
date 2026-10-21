package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"

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

func main() {
	cfgPath := flag.String("config", "config.yaml", "配置文件路径")
	mode := flag.String("mode", "", "home|wwii，默认 home（家园服）；wwii=二战独立服")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("读取配置失败: %v", err)
	}
	// ★★ 2026-10-17 二战独立部署：-mode 启动参数优先，覆盖 config.yaml 的 server.mode。
	if *mode != "" {
		cfg.Server.Mode = *mode
	}

	switch cfg.Mode() {
	case "wwii":
		runWWIIMode(cfg)
	default:
		runHomeMode(cfg)
	}
}

// runHomeMode 家园服（默认，如 8080）：
//
//	· 只注册家园路由 + 管理端非二战路由；二战用户路由不注册（二战用户直连二战服）；
//	· 二战管理端路径 /api/admin/ezfy* 由通用反向代理转发到二战服（admin-web 零改动，
//	  RBAC 仍在家园侧 perm() 校验）；NoRoute 遇 /games/ezfy* 302 到二战默认区服 host；
//	· **不连 ezfyDB**：authH/adminH 的 EzfyDB 传 nil（syncEzfyPassword 已有 nil guard），
//	  二战库由二战服自己初始化，home 只在 homeDB 上幂等维护残留二战表（可回滚）。
func runHomeMode(cfg *config.Config) {
	// ★★ 2026-10-17 二战独立部署：home 模式不再自动识别/连接二战独立库，
	//   也**不初始化** ezfyDB —— 二战数据/二战服务全部由二战独立服承担。
	homeDB := database.Init(&cfg.Mysql)
	seed.EzfyUsesOwnDB = false
	log.Printf("★ 运行模式：家园（home），只连家园库 %s（二战独立部署：home 不连二战库）", cfg.Mysql.DBName)

	// ★★ 2026-10-10 道具 cfg_id 全体迁到 1001+（与装备配置 1~35 彻底不重叠，根除「同 ID 混淆」）。
	//   必须跑在 seed **之前**：种子里的道具已按新号 1001+ 灌，若老号行还在，
	//   库里会同时存在老号行和新号行（商城列表出现重复道具）。
	//   ★ 失败必须中止启动：带着老号继续跑，种子会再灌一套新号配置 → 商城道具重复。
	//   home 模式不连二战库，只在 homeDB 上维护残留二战表（幂等，供回滚期使用）。
	//   ★★ 2026-10-17 二战独立部署：家园库可能已无二战表（二战数据早已独立到 qq_ezzt，
	//     没有「残留表」可维护）→ 缺表直接跳过（无事可迁），不再报 1146 拒启。
	if !seed.EzfyUsesOwnDB {
		if homeDB.Migrator().HasTable("ezfy_cfg_item") {
			timedStep("EnsureEzfyItemIDsHighRange", func() {
				if err := seed.EnsureEzfyItemIDsHighRange(homeDB); err != nil {
					log.Fatalf("道具 cfg_id 改号失败，拒绝启动（避免道具配置出现老号/新号两套）：%v", err)
				}
			})
		} else {
			log.Printf("家园库无 ezfy_cfg_item（二战数据已独立到二战库），跳过道具改号")
		}
	}

	// 连接的是「已被别的实例灌好数据的共享库」时(seed.skip: true)，跳过全量初始化，
	// 否则每台节点启动都会对已填充的表重跑配置 INSERT(重复主键/跨 WAN 挂起)。
	// home 模式 EzfyUsesOwnDB 恒为 false，EnsureEzfy* 链照常执行
	// （homeDB 上残留的二战表继续幂等维护，可回滚）。
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
		// ★★ 2026-10-12 二战独立登录账号表：skip 分支不补 → 二战登录直接报表不存在。
		timedStep("EnsureEzfyAccountTable", func() { seed.EnsureEzfyAccountTable(homeDB) })
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
		// ★ 2026-10-14 待开宝箱表：同 hazard —— skip 分支不建表 → 共享库发放/开箱报表不存在。
		timedStep("EnsureEzfyUserChestTable", func() { seed.EnsureEzfyUserChestTable(homeDB) })
		// ★ 2026-10-14 宝箱累计抽取次数表（奖池 min_draws 解锁门槛）：同 hazard，skip 分支幂等补建。
		timedStep("EnsureEzfyUserChestPityTable", func() { seed.EnsureEzfyUserChestPityTable(homeDB) })
		// ★ 2026-10-16 开箱流水表（管理端「数据管理→宝箱开启」数据源）：同 hazard，skip 分支幂等补建。
		timedStep("EnsureEzfyChestLogTable", func() { seed.EnsureEzfyChestLogTable(homeDB) })
		// ★ 2026-10-21 钻石流水 extra（购买宝箱开出的道具）同步扩成 text，避免 30/100 连被截断。
		timedStep("EnsureEzfyDiamondLogColumns", func() { seed.EnsureEzfyDiamondLogColumns(homeDB) })
		// ★ 2026-10-14 风云榜周榜表：同 hazard，skip 分支幂等补建（否则战斗结算写榜报表不存在）。
		timedStep("EnsureEzfyPvpWeekTable", func() { seed.EnsureEzfyPvpWeekTable(homeDB) })
		// ★ 2026-10-14 风云榜同对手积分封顶表（防小号互刷）：同 hazard，幂等补建。
		timedStep("EnsureEzfyPvpFoeTable", func() { seed.EnsureEzfyPvpFoeTable(homeDB) })
		// ★ 2026-10-16 风云榜「我的战绩」场次明细表：同 hazard，幂等补建（战报会删，明细独立留存）。
		timedStep("EnsureEzfyPvpDetailTable", func() { seed.EnsureEzfyPvpDetailTable(homeDB) })
		// ★ 2026-10-14 风云榜「不参与排名」名单表：同 hazard，幂等补建。
		timedStep("EnsureEzfyPvpExcludeTable", func() { seed.EnsureEzfyPvpExcludeTable(homeDB) })
		// ★ 2026-10-17 统帅称号三张表（定义/持有/日结算）：同 hazard，skip 分支幂等补建。
		timedStep("EnsureEzfyTitleTables", func() { seed.EnsureEzfyTitleTables(homeDB) })
		// ★ 2026-10-17 人机模块四张表（档案/攻击计划/日志/租约锁）：同 hazard，幂等补建。
		timedStep("EnsureEzfyBotTables", func() { seed.EnsureEzfyBotTables(homeDB) })
		// ★ 2026-10-14 奖池 min_draws 列：存量库 AutoMigrate 可能漏加，skip 分支 HasColumn 幂等补。
		timedStep("EnsureEzfyChestItemMinDraws", func() { seed.EnsureEzfyChestItemMinDraws(homeDB) })
		// ★ 2026-10-10 线上事故：管理端发装备漏拷六项战斗属性 → 战报【守方装备】显示「无」。
		//   全量 seed 里的 repairEquipSnapshots 本来会自愈，但 skip 分支不跑 → 共享库永远修不了。
		//   挂到 skip 路径，任何一台重启都会把存量装备行六项对齐配置池并重建军官 JSON（幂等）。
		timedStep("EnsureEzfyEquipSnapshots", func() { seed.EnsureEzfyEquipSnapshots(homeDB) })
		// ★★ 2026-10-16 野地表存量脏数据去重（同坐标多条 / 城市格残留 → 一个坐标只允许一条记录）。
		//   必须排在 EnsureEzfyIndexes 之前：存量重复会让唯一索引 (x,y) 建不出来。
		//   （独立库模式走 seed.RunEzfy → seedEzfy，那里也调了同一个 helper。）
		timedStep("EnsureEzfyWildlandUnique", func() { seed.EnsureEzfyWildlandUnique(homeDB) })
		// ★ 2026-10-05 索引同样要补：多机下只有一台跑全量 seed，另一台走这条 skip 路径，
		//   否则慢接口的复合索引在这台机器的库上永远建不出来（helper 幂等，先到先建）。
		timedStep("EnsureEzfyIndexes", func() { seed.EnsureEzfyIndexes(homeDB) })
		// ★ 2026-10-06 计谋「恫疑虚喝/隐真示假」kind 幂等补配：存量库已有计谋行且 kind=0，
		//   会卡「暂未实现」；全量 seed（seedEzfySchemes 表非空即跳过）与 skip 分支都要跑。
		timedStep("EnsureEzfySchemeKinds", func() { seed.EnsureEzfySchemeKinds(homeDB) })
		// ★ 2026-10-10 首页布局回填：home_layout NULL→0（默认新布局）；玩家 10000 保留老布局。
		timedStep("EnsureEzfyHomeLayout", func() { seed.EnsureEzfyHomeLayout(homeDB) })
		// ★★ 2026-10-12 线上事故：skip 模式（多机共享库）不跑 seed.Run → seedRBAC 被跳过
		//   → 新加的菜单权限点永远不入 permissions 表。表现极具误导性：
		//   超管菜单是全量展示的（前端 my-perms 返回 super=true），点进去却被
		//   RequirePerm 拦成 403「你没有权限进入！」（免战管理 module:ezfyPeace）。
		//   幂等补权限点 + 给「全量角色」追加，自定义角色不动，见 seed/rbac_ensure.go。
		//   ★★ 2026-10-17 移出：见下面 if cfg.Seed.Skip 单独一步（原来挂在
		//   `else if !seed.EzfyUsesOwnDB` 里，独立库模式整段不跑 = 线上从来没补过）。
	}

	// ★★ 2026-10-17 线上事故修复：「管理端能登录进去，但里面全是没数据/403」。
	//   根因一：RBAC 补权限点原来挂在 `else if !seed.EzfyUsesOwnDB`（拆库 65000b4 引入）
	//   的分支里，而 EnsureRBACPerms 是 10-12 才加进去的 —— 线上是**独立库模式**
	//   （EzfyUsesOwnDB=true），这个 else if 恒为 false → 补权限点在线上**一次都没跑过**，
	//   10-12 之后新加的菜单权限码（module:ezfyPeace 等）在线上 permissions 表里根本不存在：
	//   超管菜单全量展示 → 点进去 RequirePerm 403「你没有权限进入！」→ 页面没数据。
	//   根因二：超管角色存量权限被改少时无法自愈（角色管理保存是 Replace 全量替换），
	//   EnsureRBACPerms 现在会每次启动把内置全量角色（super_admin/admin）对齐全部权限点
	//   （只增不删，见 rbac_ensure.go）。
	//   RBAC 表（permissions/roles/role_permissions）都在**家园库**，与单库/独立库无关，
	//   所以 skip 模式下无论哪种库形态都要补。非 skip 模式由 seed.Run → seedRBAC 全量灌。
	if cfg.Seed.Skip {
		timedStep("EnsureRBACPerms", func() { seed.EnsureRBACPerms(homeDB) })
	}

	// 家园论坛索引（幂等）：原来挂在 EnsureEzfyIndexes 里，二战拆库后必须在家园库上执行。
	timedStep("EnsureHomeForumIndexes", func() { seed.EnsureHomeForumIndexes(homeDB) })

	// ★★ 2026-10-16 线上事故：`users.session_at`（2026-10-15 token 服务端作废）只在
	//   seed.Run 的 AutoMigrate 里建列，skip 分支只补 ezfy 侧 → 多机共享库上 users 缺列，
	//   新用户注册全字段 INSERT 报 `Unknown column 'session_at'`（ERROR 1054）。
	//   这里在家园库上幂等补**家园全部缺失列**（只 ADD COLUMN，不删不改），
	//   与 EnsureEzfySchema 同构，任何一台节点重启都会自动补上。
	timedStep("EnsureHomeSchema", func() { seed.EnsureHomeSchema(homeDB) })

	// 不执行 RunEzfy：二战独立部署下，二战库的建表/搬库/种子由二战服（-mode=wwii）负责。

	gin.SetMode(gin.ReleaseMode)
	r := router.Setup(homeDB, nil, cfg)
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	fmt.Println("==========================================")
	fmt.Println("  家园社区 服务端已启动  http://127.0.0.1" + addr)
	fmt.Println("  默认管理员：账号 10000 / 密码 admin123")
	fmt.Println("  ★ 二战征途已独立部署：二战服务由二战服提供（/games/ezfy* 302 跳转）")
	fmt.Println("==========================================")
	if err := r.Run(addr); err != nil {
		log.Fatalf("启动失败: %v", err)
	}
}

// runWWIIMode 二战独立服（-mode=wwii，如 8081/8082）：
//
//	· 只注册二战路由（登录/SSO/zone/用户路由/管理端路由带内部 secret 校验）+ 托管二战前端；
//	· 连 homeDB（只读：账号/昵称/game_servers）+ ezfyDB（读写）；
//	· 启动跑 seed.RunEzfy（建表/搬库或 fresh 灌种子）+ 三个 BgTick goroutine；
//	· **跳过** seed.Run/EnsureEzfy* 链/EnsureRBACPerms/EnsureHomeForumIndexes/EnsureHomeSchema
//	  （这些是 home 模式职责，RBAC 表在 homeDB 上由家园服维护）。
func runWWIIMode(cfg *config.Config) {
	if cfg.EzfyMysql.DBName == "" {
		log.Fatal("二战模式必须配置 ezfy_mysql.dbname（二战独立库，如 qq_ezzt / qq_ezzt2）")
	}
	homeDB := database.Init(&cfg.Mysql)
	ezfyCfg := cfg.EzfyDBConfig()
	ezfyDB := database.Init(&ezfyCfg)
	seed.EzfyUsesOwnDB = true
	seed.EzfyBootstrapFresh = cfg.EzfyBootstrapFresh()
	log.Printf("★ 运行模式：二战独立服（wwii），家园库 %s（只读）+ 二战库 %s（读写）",
		cfg.Mysql.DBName, cfg.EzfyMysql.DBName)
	if seed.EzfyBootstrapFresh {
		log.Printf("★ ezfy_mysql.bootstrap=fresh：二战库 %s 跳过搬库（只建表+灌种子）", cfg.EzfyMysql.DBName)
	}

	// 二战库初始化：建表 +（非 fresh 时首次自动搬库）+ 幂等补列 + 灌种子。
	// 独立库模式下道具 cfg_id 1001+ 迁移由 RunEzfy 内部承担（见 ezfy_run.go）。
	timedStep("RunEzfy", func() { seed.RunEzfy(ezfyDB, homeDB) })

	gin.SetMode(gin.ReleaseMode)
	r := router.Setup(homeDB, ezfyDB, cfg)
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	fmt.Println("==========================================")
	fmt.Println("  二战征途 独立服已启动  " + cfg.ZoneTitle() + "  http://127.0.0.1" + addr)
	fmt.Println("  默认管理员：账号 10000 / 密码 admin123")
	fmt.Println("==========================================")
	if err := r.Run(addr); err != nil {
		log.Fatalf("启动失败: %v", err)
	}
}
