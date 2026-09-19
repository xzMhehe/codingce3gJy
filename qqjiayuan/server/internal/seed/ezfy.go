package seed

import (
	"log"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"qqjiayuan/server/internal/model"
)

// seedEzfy 二战风云：配置表幂等种子（数据源自 stzb-fk inithebing.sql 转换，见 ezfy_cfg_gen.go）
//
// 两种写入策略，按「该表是否允许管理端改」区分：
//
//   - batch（upsert，UpdateAll）：纯代码维护的表（科技/野地/道具/任务）。
//     每次启动用代码里的值对齐库，避免「代码对、库里错」的配置漂移。
//   - batchKeep（insert-only，DoNothing）：**管理端可编辑**的表。
//     只补缺失行，已存在的行原样保留 —— 否则管理端在「建筑总配置 / 兵种配置 /
//     名将列表 / 技能列表 / 装备列表」里改的参数，会在下次重启时被种子悄悄改回去。
//     代价：改了 ezfy_cfg_gen.go 里的默认值不会自动落到老库；
//     需要强制对齐时，删掉该行重启即可。
func seedEzfy(db *gorm.DB) {
	batch := func(items interface{}, table string) {
		if err := db.Clauses(clause.OnConflict{UpdateAll: true}).
			CreateInBatches(items, 200).Error; err != nil {
			log.Printf("ezfy 种子失败 %s: %v", table, err)
		}
	}
	// 只补缺，不覆盖：管理端可编辑的配置表
	batchKeep := func(items interface{}, table string) {
		if err := db.Clauses(clause.OnConflict{DoNothing: true}).
			CreateInBatches(items, 200).Error; err != nil {
			log.Printf("ezfy 种子失败 %s: %v", table, err)
		}
	}

	// —— 管理端可编辑：只补缺 ——
	batchKeep(ezfyEzfyCfgBuilding, "ezfy_cfg_building")
	batchKeep(ezfyEzfyCfgBuildingLevel, "ezfy_cfg_building_level")
	batchKeep(ezfyEzfyCfgTroop, "ezfy_cfg_troop")
	batchKeep(ezfyEzfyCfgGeneral, "ezfy_cfg_general")
	batchKeep(ezfyEzfyCfgSkill, "ezfy_cfg_skill")
	batchKeep(ezfyEzfyCfgEquipment, "ezfy_cfg_equipment")

	// 阵营兵种名: inithebing.sql 里是用 UPDATE 补的(不在 INSERT 列里),
	// 早期生成脚本只解析了 INSERT 导致全丢, 这里对「还没填过」的行补一次。
	// 加 name_axis='' 条件是为了不覆盖管理端改过的兵种名。
	for _, t := range ezfyEzfyCfgTroop {
		if t.NameAxis == "" && t.NameAlly == "" {
			continue
		}
		if err := db.Model(&model.EzfyCfgTroop{}).
			Where("id = ? AND (name_axis = '' OR name_axis IS NULL)", t.ID).
			Updates(map[string]interface{}{"name_axis": t.NameAxis, "name_ally": t.NameAlly}).Error; err != nil {
			log.Printf("ezfy 阵营兵种名写入失败 id=%d: %v", t.ID, err)
		}
	}

	// —— 纯代码维护：每次对齐 ——
	batch(ezfyEzfyCfgTech, "ezfy_cfg_tech")
	batch(ezfyEzfyCfgTechLevel, "ezfy_cfg_tech_level")
	batch(ezfyEzfyCfgWildland, "ezfy_cfg_wildland")
	batch(ezfyEzfyCfgItem, "ezfy_cfg_item")
	batch(ezfyEzfyCfgTaskType, "ezfy_cfg_task_type")
	batch(ezfyEzfyCfgTask, "ezfy_cfg_task")

	seedEzfyNotices(db)
	seedEzfyOfficerItems(db)
	seedEzfyActivities(db)
	seedEzfyResources(db)
}

// seedEzfyResources 资源显示名配置（管理端可改名，全站跟随）
//
// ★ 「只补缺不覆盖」：管理端改过的名字不能被启动时打回。
func seedEzfyResources(db *gorm.DB) {
	rows := []model.EzfyCfgResource{
		{ID: 1, Key: "gold", Name: "黄金", Short: "金", Sort: 1},
		{ID: 2, Key: "food", Name: "粮食", Short: "粮", Sort: 2},
		{ID: 3, Key: "steel", Name: "钢铁", Short: "钢", Sort: 3},
		{ID: 4, Key: "oil", Name: "石油", Short: "油", Sort: 4},
		{ID: 5, Key: "rare", Name: "稀矿", Short: "稀", Sort: 5},
	}
	if err := db.Clauses(clause.OnConflict{DoNothing: true}).
		CreateInBatches(rows, 50).Error; err != nil {
		log.Printf("ezfy 资源名种子失败: %v", err)
	}
}

// seedEzfyActivities 节日活动模板（E1，设计依据 `参考材料/开发文档/福利.txt`）
//
// 原版 Java 只做了静态说明页，这里做成可配置、真实生效的活动。
// 默认全部「未开启」(status=0)，避免上线即改数值；管理端「数据管理 → 节日活动」
// 把 status 改成 1 并填好起止时间即可生效。时间戳单位毫秒。
//
//	type: 1资源增产 2造兵打折 3建造加速 4研究加速 5声望加成
func seedEzfyActivities(db *gorm.DB) {
	var count int64
	if err := db.Table("ezfy_activity").Count(&count).Error; err != nil || count > 0 {
		return
	}
	rows := []model.EzfyActivity{
		{ID: 1, Name: "丰收节", Type: 1, Param: 20, Status: 0,
			Des: "活动期间全城资源产量 +20%，野地产出同样生效"},
		{ID: 2, Name: "军工动员", Type: 2, Param: 25, Status: 0,
			Des: "活动期间训练部队的资源消耗 -25%"},
		{ID: 3, Name: "建设狂潮", Type: 3, Param: 30, Status: 0,
			Des: "活动期间建造/升级建筑耗时 -30%"},
		{ID: 4, Name: "科技峰会", Type: 4, Param: 20, Status: 0,
			Des: "活动期间科技研究耗时 -20%"},
		{ID: 5, Name: "荣耀之战", Type: 5, Param: 10, Status: 0,
			Des: "活动期间战斗获得的军功声望 +10%"},
	}
	if err := db.Create(&rows).Error; err != nil {
		log.Printf("ezfy 节日活动种子失败: %v", err)
	}
}

// seedEzfyOfficerItems 军官类道具（复刻设计文档《QQ家园二战风云.txt》道具 #7/#8/#9）
//
// 原工程这批道具标为「未实现」，这里补齐；按 ID 幂等 upsert，老库也能补上。
//
//	13 招生简章   ItemType 9  立即刷新军校候选名将(不占每日 5 次)
//	14 经验书     ItemType 10 指定军官获得经验
//	15 军官技能书 ItemType 11 指定军官免费学习 1 个技能
//	16 重修书     ItemType 12 重置军官属性成长并清空技能(等级/经验保留)
//	17 改名卡     ItemType 13 统帅页改昵称(首次免费, 之后每次消耗 1 张)
//	18 阵营转换道具 ItemType 14 统帅页改阵营(首次免费, 之后每次消耗 1 个)
func seedEzfyOfficerItems(db *gorm.DB) {
	rows := []model.EzfyCfgItem{
		{ID: 13, Name: "招生简章", ItemType: 9, Param1: 1, PriceGold: 500,
			Description: "立即刷新军校候选名将, 不占用每日刷新次数"},
		{ID: 14, Name: "经验书", ItemType: 10, Param1: 1000, PriceGold: 300,
			Description: "指定军官获得1000点经验"},
		{ID: 15, Name: "军官技能书", ItemType: 11, Param1: 1, PriceGold: 1000,
			Description: "指定军官免费学习1个技能(不消耗黄金)"},
		{ID: 16, Name: "重修书", ItemType: 12, Param1: 0, PriceGold: 800,
			Description: "重置军官属性成长并清空已学技能(等级与经验保留)"},
		{ID: 17, Name: "改名卡", ItemType: 13, Param1: 1, PriceGold: 500,
			Description: "在统帅页修改玩家昵称(首次改名免费, 之后每次消耗1张)"},
		{ID: 18, Name: "阵营转换道具", ItemType: 14, Param1: 1, PriceGold: 800,
			Description: "在统帅页转换阵营(首次转换免费, 之后每次消耗1个)"},
	}
	for _, it := range rows {
		var count int64
		db.Model(&model.EzfyCfgItem{}).Where("id = ?", it.ID).Count(&count)
		if count > 0 {
			// 已存在则只同步名称/说明, 不动价格(避免覆盖后台调价)
			db.Model(&model.EzfyCfgItem{}).Where("id = ?", it.ID).
				Updates(map[string]interface{}{"name": it.Name, "item_type": it.ItemType,
					"param1": it.Param1, "description": it.Description})
			continue
		}
		db.Create(&it)
	}
}

// seedEzfyNotices 游戏内置公告（幂等：标题存在即跳过）
func seedEzfyNotices(db *gorm.DB) {
	rows := []model.EzfyNotice{
		{UserId: 0, IsTop: 1, Title: "二战风云开服公告",
			Content: "各位司令官，欢迎来到二战风云！建造城池、发展资源、训练部队，出征野地掠夺资源。攻占寇城可以获得丰厚战利品。掠夺/征服其他玩家城池需先宣战，宣战24小时后生效。祝各位武运昌隆！"},
		{UserId: 0, IsTop: 0, Title: "新手提示",
			Content: "进入游戏自动获得主城(市政厅/民居/农田各1级)。先用黄金召集人口，再建资源建筑。造兵需要军工厂，研究科技需要科研中心。市政厅等级决定可占领野地数量上限。"},
	}
	for _, n := range rows {
		var count int64
		db.Model(&model.EzfyNotice{}).Where("title = ? AND user_id = 0", n.Title).Count(&count)
		if count == 0 {
			db.Create(&n)
		}
	}
}
