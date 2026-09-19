package seed

import (
	"log"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"qqjiayuan/server/internal/model"
)

// seedEzfy 二战风云：配置表幂等种子（数据源自 stzb-fk inithebing.sql 转换，见 ezfy_cfg_gen.go）
//
// 采用「按主键 upsert」而非「表为空才写入」：
// 早期版本用 count>0 就跳过，导致 ezfy_cfg_gen.go 修正过的配置（如市政厅 can_delete）
// 永远不会落到已有库上，出现「代码对、库里错」的配置漂移。
// 配置表以 inithebing.sql 转换结果为唯一事实来源，每次启动对齐一次。
func seedEzfy(db *gorm.DB) {
	batch := func(items interface{}, table string) {
		if err := db.Clauses(clause.OnConflict{UpdateAll: true}).
			CreateInBatches(items, 200).Error; err != nil {
			log.Printf("ezfy 种子失败 %s: %v", table, err)
		}
	}
	batch(ezfyEzfyCfgBuilding, "ezfy_cfg_building")
	batch(ezfyEzfyCfgBuildingLevel, "ezfy_cfg_building_level")
	batch(ezfyEzfyCfgTroop, "ezfy_cfg_troop")
	batch(ezfyEzfyCfgTech, "ezfy_cfg_tech")
	batch(ezfyEzfyCfgTechLevel, "ezfy_cfg_tech_level")
	batch(ezfyEzfyCfgWildland, "ezfy_cfg_wildland")
	batch(ezfyEzfyCfgItem, "ezfy_cfg_item")
	batch(ezfyEzfyCfgTaskType, "ezfy_cfg_task_type")
	batch(ezfyEzfyCfgTask, "ezfy_cfg_task")
	// 军官/学院：名将 31 / 技能 15 / 装备 26
	batch(ezfyEzfyCfgGeneral, "ezfy_cfg_general")
	batch(ezfyEzfyCfgSkill, "ezfy_cfg_skill")
	batch(ezfyEzfyCfgEquipment, "ezfy_cfg_equipment")

	seedEzfyNotices(db)
	seedEzfyOfficerItems(db)
}

// seedEzfyOfficerItems 军官类道具（复刻设计文档《QQ家园二战风云.txt》道具 #7/#8/#9）
//
// 原工程这批道具标为「未实现」，这里补齐；按 ID 幂等 upsert，老库也能补上。
//
//	13 招生简章   ItemType 9  立即刷新军校候选名将(不占每日 5 次)
//	14 经验书     ItemType 10 指定军官获得经验
//	15 军官技能书 ItemType 11 指定军官免费学习 1 个技能
//	16 重修书     ItemType 12 重置军官属性成长并清空技能(等级/经验保留)
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
