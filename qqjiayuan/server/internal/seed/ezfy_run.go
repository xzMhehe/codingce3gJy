package seed

import (
	"log"

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

// RunEzfy 二战库：建表 + 幂等补列 + 灌种子。
//
// 在 main.go 里当 `config.EzfySplit()` 为真时调用（单库模式由 seed.Run 一并承担）。
// 幂等：可重复启动；配置类种子是 upsert，玩家数据不动。
func RunEzfy(db *gorm.DB) {
	if err := db.AutoMigrate(ezfyMigrateModels...); err != nil {
		log.Fatalf("二战库建表失败: %v", err)
	}
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
