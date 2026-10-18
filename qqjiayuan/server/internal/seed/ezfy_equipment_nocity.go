package seed

import (
	"log"

	"gorm.io/gorm"

	"qqjiayuan/server/internal/model"
)

// EnsureEzfyEquipmentNoCity 把 `ezfy_equipment.city_id` 全部清零（一次性，幂等）。
//
// ★★ 2026-10-18 线上事故（用户 35806363 丢 3 件青天白日）：
//
//	装备是**玩家背包资产** —— 只认 `user_id`（归属账号）+ `officer_id`（穿在谁身上，0=背包），
//	和城市没有任何关系。`city_id` 只是历史「获得时的出生城」，穿戴 / 俘虏转移 / 派遣到别城
//	之后都**不会更新**（备份里就有 `city_id=59` 却穿在城 46 军官身上的行）。
//
//	只要代码里还有任何一处 `WHERE city_id = ?` 去删/迁装备，就等于把「出生地」当「存放地」：
//	  · 2026-10-13 弃城：一次丢了 305 件（35806411）
//	  · 2026-10-18 占城摧毁：丢 3 件青天白日（35806363，城 59 曾是这三件的出生城）
//
// 处理：**把存量 city_id 全部清零**，让任何按 city_id 的 SQL 天然命中 0 行；
// 新装备在 `buildEquipment` 里也不再写 city_id（恒 0）。
//
// 安全性（吸取 2026-10-10「启动迁移反复重跑删道具」的教训）：
//
//	· 只 UPDATE 一个**没有任何业务读取**的字段，不增删任何行；
//	· 带 `city_id <> 0` 条件 → 第二次启动就是 no-op；
//	· 不依赖 `sync.Once`（重启即归零），天然可重复执行。
func EnsureEzfyEquipmentNoCity(db *gorm.DB) {
	if db == nil || !db.Migrator().HasTable(&model.EzfyEquipment{}) {
		return
	}
	res := db.Model(&model.EzfyEquipment{}).Where("city_id <> 0").Update("city_id", 0)
	if res.Error != nil {
		log.Printf("二战装备 city_id 清零失败: %v", res.Error)
		return
	}
	if res.RowsAffected > 0 {
		log.Printf("二战装备 city_id 清零：%d 行（装备归属只看 user_id/officer_id）", res.RowsAffected)
	}
}
