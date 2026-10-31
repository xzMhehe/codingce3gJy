package seed

import (
	"log"

	"gorm.io/gorm"

	"qqjiayuan/server/internal/model"
)

// ezfyTechSpeed50Fix 一次性把「机械增压(10)」「喷气推进(19)」改成每级 +5%、满级 +50%。
//
// ★ 2026-10-31 用户需求：这两项速度科技满级效果统一到 50%
// （机械增压原每级 2% 满级 20%；喷气推进原每级 3% 满级 30%）。
//
// 为什么必须单独迁移：`ezfy_cfg_tech_level` 的 `effect` / `effect_value` 被
// seedEzfyTechLevels **刻意保护**（2026-10-14 事故：整行 UpdateAll 把管理端改的
// 「战地医疗 Lv.10 = 50」每次重启冲回 20）→ 改 ezfy_cfg_gen.go 里的文案**不会落到老库**。
//
// 用 settings 打标只跑一次：之后管理端在「科技管理 → 等级配置 → 效果数值」里再调，
// 重启不会被改回去。新库不需要它（种子直接插新文案），打标也照写，避免重复执行。
func ezfyTechSpeed50Fix(db *gorm.DB) {
	ezfyTechLadder50(db, "ezfy_tech_speed50_v1",
		[]ezfyTech50Item{{10, "部队速度"}, {19, "空军速度"}},
		"ezfy 科技速度满级50%：机械增压/喷气推进 已改为每级+5%（满级+50%）")
}

// ezfyTechLoad50Fix 一次性把「标准化装卸(13) 部队负重」改成每级 +5%、满级 +50%。
//
// ★★ 2026-10-31 用户需求：「标准化装卸 最高等级加成调整为 50%，其他阶梯调整」
// → 与机械增压/喷气推进同口径（线性 5%/级：5,10,15,…,50）。
//
// 同样的坑：`ezfy_cfg_tech_level.effect` / `effect_value` 被 seedEzfyTechLevels 刻意保护
// （管理端独家维护），改种子文案落不到老库 → 单独一次性迁移，新 key 不与上面的 10/19 共用。
func ezfyTechLoad50Fix(db *gorm.DB) {
	ezfyTechLadder50(db, "ezfy_tech_load50_v1",
		[]ezfyTech50Item{{13, "部队负重"}},
		"ezfy 科技负重满级50%：标准化装卸 已改为每级+5%（满级+50%）")
}

// ezfyTech50Item 一条「满级 50%」阶梯科技：techId + 效果文案前缀
type ezfyTech50Item struct {
	id     int
	prefix string
}

// ezfyTechLadder50 通用实现：把指定科技的 1~10 级阶梯写成「level*5」并打标只跑一次。
func ezfyTechLadder50(db *gorm.DB, key string, items []ezfyTech50Item, doneMsg string) {
	if !db.Migrator().HasTable(&model.EzfyCfgTechLevel{}) || !db.Migrator().HasTable("settings") {
		return
	}
	var marked int64
	db.Raw("SELECT COUNT(*) FROM settings WHERE `key` = ?", key).Scan(&marked)
	if marked > 0 {
		return
	}
	for _, it := range items {
		// 阶梯值直接由 level 生成：Lv1..10 → 5,10,15,…,50
		res := db.Exec("UPDATE ezfy_cfg_tech_level SET effect_value = level * 5, "+
			"effect = CONCAT(?, '+', level * 5, '%') WHERE tech_id = ?", it.prefix, it.id)
		if res.Error != nil {
			log.Printf("ezfy 科技满级50%%：tech=%d 对齐失败（下次启动重试）: %v", it.id, res.Error)
			return
		}
	}
	db.Exec("INSERT INTO settings (`key`, `value`) VALUES (?, ?)", key, "1")
	log.Printf("%s", doneMsg)
}
