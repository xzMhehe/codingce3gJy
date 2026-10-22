package seed

import (
	"log"

	"gorm.io/gorm"

	"qqjiayuan/server/internal/model"
)

// 二战兵种配置「补缺 / 一次性对齐」种子
//
// ★ 2026-10-22 新增「海军轰炸机」（id 21，空军·对海特化）。
//
// 兵种本体登记在 ezfy_cfg_gen.go：全量种子 / 二战独立库（RunEzfy→seedEzfy）走
// `batchKeep`（OnConflict DoNothing）自动插入，1区、2区各自的库都会补上。
//
// 但「多机共享库 + seed.skip:true」的部署**不跑全量种子** → 新兵种永远不存在
// （兵种列表里看不到、训练不了）。所以这里再挂一个按 ID 判存在的幂等补种，
// 由 main.go 的 skip 分支调用（与 EnsureEzfyBuildQueueItem 同一套思路）。
//
// 只补不覆盖：已存在就跳过，管理端在「兵种配置」里改过的数值不会被启动冲掉。
const ezfyNavalBomberTroopID = 21

// ezfyNavalBomberAlignKey 一次性对齐的 settings 标记位。
//
// ★ 2026-10-22 用户二次调整（对海 75→110 / 训练时间 90→100 秒 / 阵营名补全为「型号+海军轰炸机」，
// 轴心国从日本换成德国 Fw 200 秃鹰）。已经部署过的库（id=21 已在表里）靠 batchKeep 补不进去，
// 所以用 settings 打标**只对齐一次** —— 之后管理员在「兵种配置」里再怎么调，重启都不会被顶回来。
const ezfyNavalBomberAlignKey = "ezfy_naval_bomber_align_20261022"

// EnsureEzfyCfgTroops 单库 skip 模式下幂等补新增兵种配置（1区/2区各自执行一次）。
func EnsureEzfyCfgTroops(db *gorm.DB) {
	if !db.Migrator().HasTable(&model.EzfyCfgTroop{}) {
		return
	}
	ensureEzfyTroopByID(db, ezfyNavalBomberTroopID)
	alignEzfyNavalBomber(db)
}

// ensureEzfyTroopByID 按 ID 从内置兵种清单里补一行（存在则跳过）。
func ensureEzfyTroopByID(db *gorm.DB, id int) {
	var n int64
	if err := db.Model(&model.EzfyCfgTroop{}).Where("id = ?", id).Count(&n).Error; err != nil {
		log.Printf("ezfy 兵种配置 id=%d 存在性检查失败: %v", id, err)
		return
	}
	if n > 0 {
		return
	}
	for i := range ezfyEzfyCfgTroop {
		t := ezfyEzfyCfgTroop[i]
		if t.ID != id {
			continue
		}
		if err := db.Create(&t).Error; err != nil {
			log.Printf("ezfy 补兵种配置 id=%d 失败: %v", id, err)
			return
		}
		log.Printf("ezfy 补兵种配置: %s (id=%d)", t.Name, t.ID)
		return
	}
	log.Printf("ezfy 补兵种配置 id=%d 跳过：内置清单里没有这一项", id)
}

// alignEzfyNavalBomber 一次性把「海军轰炸机」的 4 列对齐到最新口径（settings 打标，只跑一次）。
//
// 覆盖的列：对海 / 训练时间 / 同盟国名 / 轴心国名。
// 幂等：settings 有标记就跳过；settings 表不存在时也跳过（新库由 batchKeep 直接插新值）。
func alignEzfyNavalBomber(db *gorm.DB) {
	if !db.Migrator().HasTable(&model.EzfyCfgTroop{}) || !db.Migrator().HasTable("settings") {
		return
	}
	var marked int64
	db.Raw("SELECT COUNT(*) FROM settings WHERE `key` = ?", ezfyNavalBomberAlignKey).Scan(&marked)
	if marked > 0 {
		return
	}
	var want *model.EzfyCfgTroop
	for i := range ezfyEzfyCfgTroop {
		if ezfyEzfyCfgTroop[i].ID == ezfyNavalBomberTroopID {
			want = &ezfyEzfyCfgTroop[i]
			break
		}
	}
	if want == nil {
		return
	}
	res := db.Model(&model.EzfyCfgTroop{}).Where("id = ?", ezfyNavalBomberTroopID).
		Updates(map[string]interface{}{
			"atk_sea":    want.AtkSea,
			"train_time": want.TrainTime,
			"name_ally":  want.NameAlly,
			"name_axis":  want.NameAxis,
		})
	if res.Error != nil {
		log.Printf("ezfy 海军轰炸机一次性对齐失败（下次启动会重试）: %v", res.Error)
		return
	}
	db.Exec("INSERT INTO settings (`key`, `value`) VALUES (?, ?)", ezfyNavalBomberAlignKey, "1")
	if res.RowsAffected > 0 {
		log.Printf("ezfy 海军轰炸机已对齐: 对海%d 训练%d秒 %s / %s",
			want.AtkSea, want.TrainTime, want.NameAlly, want.NameAxis)
	}
}
