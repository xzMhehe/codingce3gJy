package seed

import (
	"log"
	"strings"

	"gorm.io/gorm"

	"qqjiayuan/server/internal/model"
)

// ★★ 2026-10-10 二战风云独立库（多数据源）
//
// 二战征途的**全部数据**（ezfy_* + 自己的家信/系统设置）搬进独立库（config 的
// `ezfy_mysql`，如 qq_ezzt），与家园库（qq_jiayuan）分开：
//
//	qq_ezzt   ：ezfy_*（70 张） + private_messages（二战自己的消息） + settings
//	qq_jiayuan：账号/角色/权限 + 私信 + 设置 + 其它小游戏（users 仍在这里）
//
// 二战与家园**只靠「家园号」(users.id) 关联**：账号、昵称仍由家园库唯一持有，
// 二战经 `EzfyHandler.HomeDB` 只读（不复制、不需要同步）。
// ★ 消息（家信）**用二战自己的**：二战写的宣战/系统通知落在二战库的 private_messages，
//   不会再出现在玩家的家园家信里（用户明确要求）。

// EzfyUsesOwnDB 二战风云是否使用独立库（main 启动时按配置设置）。
//
// 为 true 时：`seed.Run`（家园）不再建/灌二战表与种子，改由 `RunEzfy` 在二战库上做。
var EzfyUsesOwnDB bool

// ezfyMigrateModels 二战库要建的模型：全部 ezfy_* + 二战自己要用的两张通用表。
//
// ⚠️ 不含 `model.User`：账号仍在家园库（二战经 HomeDB 只读）。
// ⚠️ 新增二战表请加到这里；新增「二战自己用」的通用表同理。
var ezfyMigrateModels = []interface{}{
	// —— 配置表 ——
	&model.EzfyCfgBuilding{}, &model.EzfyCfgBuildingLevel{}, &model.EzfyCfgTroop{},
	&model.EzfyCfgTech{}, &model.EzfyCfgTechLevel{}, &model.EzfyCfgWildland{},
	&model.EzfyCfgItem{}, &model.EzfyCfgTaskType{}, &model.EzfyCfgTask{},
	&model.EzfyCfgGeneral{}, &model.EzfyCfgSkill{}, &model.EzfyCfgEquipment{},
	&model.EzfyCfgEquipSet{}, &model.EzfyCfgChest{}, &model.EzfyCfgChestItem{},
	&model.EzfyCfgScheme{}, &model.EzfyCfgResource{}, &model.EzfyCfgRank{},
	&model.EzfyCfgLimit{}, &model.EzfyWordFilter{},
	// —— 玩家档案 / 城池 ——
	&model.EzfyProfile{}, &model.EzfyCity{}, &model.EzfyCityBuilding{},
	&model.EzfyCityTroop{}, &model.EzfyCityTech{}, &model.EzfyUserTech{}, &model.EzfyTrainQueue{},
	&model.EzfyMapArea{}, &model.EzfyMapTile{}, &model.EzfyMapStar{}, &model.EzfyActWild{},
	// —— 出征 / 战斗 / 战报 ——
	&model.EzfyOrder{}, &model.EzfyBattle{}, &model.EzfyReport{},
	&model.EzfyWildland{}, &model.EzfyOccupy{}, &model.EzfyWounded{},
	&model.EzfyRansom{}, // ★ 2026-10-07 赎城请求表
	// —— 军团 ——
	&model.EzfyWar{}, &model.EzfyCorps{}, &model.EzfyCorpsMember{}, &model.EzfyCorpsChat{},
	&model.EzfyCorpsRelation{}, &model.EzfyCorpsWar{}, &model.EzfyCorpsMall{}, &model.EzfyCorpsMallLog{},
	&model.EzfyCorpsApply{},
	// —— 道具 / 背包 / 装备 / 军官 ——
	&model.EzfyItem{}, &model.EzfySign{}, &model.EzfyGift{}, &model.EzfyTreasureSign{},
	&model.EzfyCityEffect{}, &model.EzfyCityTarget{}, &model.EzfyTask{}, &model.EzfyNotice{},
	&model.EzfyChat{}, &model.EzfyExchange{}, &model.EzfyExchangeTemplate{},
	&model.EzfyDiamondLog{}, &model.EzfyItemUseLog{},
	&model.EzfyOfficer{}, &model.EzfyEquipment{}, &model.EzfyRecruit{},
	&model.EzfyPreset{}, &model.EzfyActivity{}, &model.EzfyFriend{}, &model.EzfyFriendApply{},
	&model.EzfyLoveCard{},
	// —— 二战自己用的通用表（不共享家园）——
	&model.PrivateMessage{}, // 二战的家信/系统通知（不再写家园家信）
	&model.Setting{},        // 二战自己的系统开关（ezfy_maintenance 等）
}

// RunEzfy 二战库：建表 + **首次自动搬库** + 幂等补列 + 灌种子。
//
// 在 main.go 里当 `config.EzfySplit()` 为真时调用（单库模式由 seed.Run 一并承担）。
// 幂等：可重复启动；配置类种子是 upsert，玩家数据不动。
//
// homeDB = 家园库（搬库时的数据来源；单库模式下与 db 相同）。
func RunEzfy(db *gorm.DB, homeDB *gorm.DB) {
	if err := db.AutoMigrate(ezfyMigrateModels...); err != nil {
		log.Fatalf("二战库建表失败: %v", err)
	}
	// ★★ 首次启动：二战库还是空的 → 把家园库里现成的二战数据整体搬过来。
	//   必须在灌种子之前（种子只补配置，搬的是玩家数据）。
	ezfyCopyFromHomeIfEmpty(db, homeDB)
	// 跨版本补列（幂等；新库 AutoMigrate 已建全，这里主要照顾存量/半旧库）
	EnsureEzfySchema(db)
	EnsureEzfyLimitColumns(db)
	EnsureEzfyRansomTable(db)
	EnsureEzfyOfficerColumns(db)
	EnsureEzfyBattleLockColumns(db)
	EnsureEzfyExchangeColumns(db)
	EnsureEzfyOrderColumns(db)
	// ★ 道具 cfg_id 改号必须跑在 seed 之前：种子里的道具已按新号 1001+ 写
	if err := EnsureEzfyItemIDsHighRange(db); err != nil {
		log.Fatalf("二战库道具改号失败: %v", err)
	}
	// 配置种子（内部含 EnsureEzfyIndexes / 宝箱奖池补缺 / 装备快照修复 / 计谋 kind 等）
	seedEzfy(db)
	EnsureEzfySchemeKinds(db)
	EnsureEzfyHomeLayout(db)
	log.Printf("二战库初始化完成（独立库模式）")
}

// ezfyCopyFromHomeIfEmpty 首次启用独立库时，把家园库里的二战数据整体搬进二战库。
//
// 为什么必须搬：AutoMigrate 只建**空表** —— 不搬的话玩家的城/军官/道具/订单全部消失。
// 为什么必须由二进制搬（而不是运维手工搬）：老进程在切库前一直往家园库写，
//
//	手工搬完就开始漂移；放在启动时（老进程已停、新进程还没接客）搬才是**一致快照**。
//
// 安全性（吸取 2026-10-10「启动迁移反复重跑删道具」的教训）：
//
//	· 幂等：只在「二战库 ezfy_city 为空」且「家园库 ezfy_city 非空」时才搬；搬完再启动即 no-op。
//	· 不删不覆盖：逐表 `INSERT IGNORE ... SELECT`（主键冲突跳过），绝不 UPDATE/DELETE 任何行。
//	· 可核验：每张表打印搬了多少行，搬完对照家园库行数即可。
func ezfyCopyFromHomeIfEmpty(dst, src *gorm.DB) {
	if dst == nil || src == nil {
		return
	}
	// 同一个连接（单库模式）→ 不需要搬
	var dstName, srcName string
	dst.Raw("SELECT DATABASE()").Scan(&dstName)
	src.Raw("SELECT DATABASE()").Scan(&srcName)
	if dstName == "" || srcName == "" || dstName == srcName {
		return
	}
	var dstCities int64
	if err := dst.Model(&model.EzfyCity{}).Count(&dstCities).Error; err != nil {
		log.Printf("二战搬库: 读二战库失败，跳过：%v", err)
		return
	}
	if dstCities > 0 {
		return // 已经有数据（搬过 / 本来就在用）→ no-op
	}
	var srcCities int64
	if err := src.Model(&model.EzfyCity{}).Count(&srcCities).Error; err != nil {
		log.Printf("二战搬库: 读家园库失败，跳过：%v", err)
		return
	}
	if srcCities == 0 {
		log.Printf("二战搬库: 家园库也没有二战数据（全新部署），交给种子初始化")
		return
	}

	log.Printf("二战搬库: 二战库 %s 为空、家园库 %s 有 %d 座城 → 开始整体搬迁", dstName, srcName, srcCities)
	// 二战相关表 = 目标库里所有 ezfy_* + 二战自己用的两张通用表
	tables := []string{}
	dst.Raw(`SELECT TABLE_NAME FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME LIKE 'ezfy%'`).Scan(&tables)
	tables = append(tables, "private_messages", "settings")
	copied, skipped, rows := 0, 0, int64(0)
	for _, t := range tables {
		var n int64
		src.Raw(`SELECT COUNT(*) FROM information_schema.TABLES
			WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?`, srcName, t).Scan(&n)
		if n == 0 {
			skipped++
			continue // 源库里没这张表（如家园没有的新表）
		}
		// ★ 列必须取「两边都有」的交集再显式列出：源库是老库，常有历史多余列，
		//   直接 `SELECT *` 会 Column count doesn't match（实测 ezfy_cfg_limit 就踩了）。
		var cols []string
		dst.Raw(`SELECT COLUMN_NAME FROM information_schema.COLUMNS
			WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?`, t).Scan(&cols)
		if len(cols) == 0 {
			skipped++
			continue
		}
		keep := make([]string, 0, len(cols))
		for _, col := range cols {
			var hit int64
			src.Raw(`SELECT COUNT(*) FROM information_schema.COLUMNS
				WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? AND COLUMN_NAME = ?`,
				srcName, t, col).Scan(&hit)
			if hit > 0 {
				keep = append(keep, "`"+col+"`")
			}
		}
		if len(keep) == 0 {
			skipped++
			continue
		}
		list := strings.Join(keep, ",")
		res := dst.Exec("INSERT IGNORE INTO `" + t + "` (" + list + ") SELECT " + list +
			" FROM `" + srcName + "`.`" + t + "`")
		if res.Error != nil {
			log.Printf("二战搬库: 表 %s 搬迁失败: %v", t, res.Error)
			continue
		}
		if res.RowsAffected > 0 {
			rows += res.RowsAffected
		}
		copied++
	}
	log.Printf("二战搬库完成: %d 张表、共 %d 行（跳过 %d 张源库没有/无公共列的表）", copied, rows, skipped)
}
