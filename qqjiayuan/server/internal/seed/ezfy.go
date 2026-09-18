package seed

import (
	"log"

	"gorm.io/gorm"

	"qqjiayuan/server/internal/model"
)

// seedEzfy 二战风云：配置表幂等种子（数据源自 stzb-fk inithebing.sql 转换，见 ezfy_cfg_gen.go）
func seedEzfy(db *gorm.DB) {
	batch := func(items interface{}, table string) {
		var count int64
		if err := db.Table(table).Count(&count).Error; err != nil {
			log.Printf("ezfy 查表失败 %s: %v", table, err)
			return
		}
		if count > 0 {
			return
		}
		if err := db.Create(items).Error; err != nil {
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
