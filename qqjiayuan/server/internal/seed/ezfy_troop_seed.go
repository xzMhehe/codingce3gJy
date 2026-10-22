package seed

import (
	"log"

	"gorm.io/gorm"

	"qqjiayuan/server/internal/model"
)

// 二战兵种配置「补缺」种子
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

// EnsureEzfyCfgTroops 单库 skip 模式下幂等补新增兵种配置（1区/2区各自执行一次）。
func EnsureEzfyCfgTroops(db *gorm.DB) {
	if !db.Migrator().HasTable(&model.EzfyCfgTroop{}) {
		return
	}
	ensureEzfyTroopByID(db, ezfyNavalBomberTroopID)
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
