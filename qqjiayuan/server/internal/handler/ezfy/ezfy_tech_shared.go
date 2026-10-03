package ezfy

import (
	"log"
	"sync"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"qqjiayuan/server/internal/model"
)

// 二战风云 —— 科技等级「用户级共用」（2026-09-28 用户规则）
//
// 用户原话：「多个城市可以分别研究,不同城市不能研究同一个科技,但是等级共用,
//   研究限制来自当前城市科研中心等级」「没有主城概念啊，所有城市都是一样的」。
//
// 现实现：
//   ① 科技**等级**存用户级 `ezfy_user_tech`(user_id+tech_id+level)，全城共用、无主城概念；
//      `h.techMap(cityId)` 内部按城市找 uid 再读等级，所有调用点无需改动；
//   ② 研究**队列**存 `ezfy_city_tech`(status=1)，按**发起城市**各自一条 → 多城可并行研究；
//      同一科技同一时刻只能在一个城市研究（researchTech 里全局互斥）；
//   ③ 科研中心等级要求取**当前城市**的科研中心等级（不再取全城最高）；
//   ④ 启动时把历史 `ezfy_city_tech` 的 status=0 等级行按 (用户, 科技) 取最高合并进
//      ezfy_user_tech 并删除原行；status=1 研究队列不动（见 ezfyMigrateSharedTech）。

// ezfyCityIds 玩家所有城市 id 列表
func (h *EzfyHandler) ezfyCityIds(uid uint) []uint {
	var ids []uint
	h.DB.Model(&model.EzfyCity{}).Where("user_id = ?", uid).Pluck("id", &ids)
	return ids
}

var ezfySharedTechOnce sync.Once

// ezfyMigrateSharedTech 历史数据合并：把 ezfy_city_tech 的 status=0 等级行
// 按 (用户, 科技) 取最高等级合并进用户级 ezfy_user_tech，并删除原行。
// status=1 的研究队列不动（是新版多城研究正常数据）。
// 幂等：等级行合并后即被删除，后续启动是空操作。
func ezfyMigrateSharedTech(db *gorm.DB) {
	if !db.Migrator().HasTable("ezfy_city_tech") || !db.Migrator().HasTable("ezfy_user_tech") {
		return
	}
	// 每个城市 -> 归属玩家
	var cities []model.EzfyCity
	db.Find(&cities)
	if len(cities) == 0 {
		return
	}
	cityOwner := map[uint]uint{}
	for _, c := range cities {
		cityOwner[c.ID] = c.UserID
	}

	var rows []model.EzfyCityTech
	db.Where("status = 0").Find(&rows)
	moved := 0
	for _, r := range rows {
		uid, ok := cityOwner[uint(r.CityId)]
		if !ok {
			continue
		}
		// 按 (用户, 科技) 取最高等级合并进用户级
		var ut model.EzfyUserTech
		err := db.Where("user_id = ? AND tech_id = ?", uid, r.TechId).First(&ut).Error
		if err != nil {
			db.Create(&model.EzfyUserTech{UserId: uid, TechId: r.TechId, Level: r.Level})
		} else if r.Level > ut.Level {
			db.Model(&model.EzfyUserTech{}).Where("id = ?", ut.ID).Update("level", r.Level)
		}
		db.Delete(&model.EzfyCityTech{}, r.ID)
		moved++
	}
	if moved > 0 {
		log.Printf("ezfy 科技用户级迁移: 合并 %d 条历史等级记录到 ezfy_user_tech", moved)
	}
}

// ezfyUpsertTechLevel 给指定玩家写入某科技等级（管理端一键满级用）。
// 城市 id 仅用于反查玩家；等级记录落在用户级。
func (h *EzfyHandler) ezfyUpsertTechLevel(cityId uint, techId, level int) {
	var uid uint
	h.DB.Model(&model.EzfyCity{}).Where("id = ?", cityId).Select("user_id").Scan(&uid)
	if uid == 0 {
		return
	}
	h.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "tech_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"level"}),
	}).Create(&model.EzfyUserTech{UserId: uid, TechId: techId, Level: level})
}
