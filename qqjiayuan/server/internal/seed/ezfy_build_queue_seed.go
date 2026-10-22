package seed

import (
	"log"

	"gorm.io/gorm"

	"qqjiayuan/server/internal/model"
)

// 二战「内政道具」补缺种子（幂等：按名字判存在，已存在即 no-op）
//
// ★ 2026-10-22 征募告示（ItemType 33）：
//
//	用户规则 —— 使用后当前城市建造队列 +1，有效期 24 小时；可叠加使用，最多 10 个。
//
// 为什么单独写一份补缺：
//   - 道具本体已登记在 ezfy_cfg_gen.go（全量种子 / 二战独立库会 upsert 进库）；
//   - 但「多机共享库 + seed.skip:true」的部署不会跑全量种子，
//     新道具不补就永远买不到（商城看不到、背包也用不了）。
//   - 这里按**名字**判存在（不硬编码 cfg_id），已存在时不动价格/库存，
//     管理端调过价不会被启动覆盖。
func seedEzfyBuildQueueItems(db *gorm.DB) {
	ensureEzfyCfgItemByName(db, "征募告示")
}

// EnsureEzfyBuildQueueItem 单库 skip 模式下幂等补「征募告示」道具配置。
//
// 与 EnsureEzfyActLevelTable 同款原因：skip 分支不跑 seed.Run，
// 新增道具不会入库 → 商城/背包里根本没有这件东西。
func EnsureEzfyBuildQueueItem(db *gorm.DB) {
	if !db.Migrator().HasTable(&model.EzfyCfgItem{}) {
		return
	}
	seedEzfyBuildQueueItems(db)
}

// ensureEzfyCfgItemByName 按名字从内置道具清单里补一行（存在则跳过）。
func ensureEzfyCfgItemByName(db *gorm.DB, name string) {
	var n int64
	if err := db.Model(&model.EzfyCfgItem{}).Where("name = ?", name).Count(&n).Error; err != nil {
		log.Printf("ezfy 道具「%s」存在性检查失败: %v", name, err)
		return
	}
	if n > 0 {
		return
	}
	for i := range ezfyEzfyCfgItem {
		it := ezfyEzfyCfgItem[i]
		if it.Name != name {
			continue
		}
		if err := db.Create(&it).Error; err != nil {
			log.Printf("ezfy 补道具「%s」失败: %v", name, err)
			return
		}
		log.Printf("ezfy 补道具: %s (id=%d)", it.Name, it.ID)
		return
	}
	log.Printf("ezfy 补道具「%s」跳过：内置清单里没有这一项", name)
}
