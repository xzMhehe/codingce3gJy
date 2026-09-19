package handler

import (
	"log"
	"sync"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"qqjiayuan/server/internal/model"
)

// 二战风云 —— 科技「所有城池公用」（第九轮用户规则）
//
// 用户原话：「玩家科技所有城池公用改下这个」。
//
// 原实现：`ezfy_city_tech` 按 city_id 存，分城各有各的科技；
//         而且 `checkTechDone` 只结算**当前所在城**，切到别的城时
//         主城的研究永远不完成 → 玩家看到「科技加成没生效」。
//
// 现实现：科技的读写统一落到**玩家的科技城**（= 主城，id 最小的城）。
//   ① 所有 `h.techMap(cityId)` 调用点无需改动 —— techMap 内部自动换算成科技城；
//   ② `checkTechDone` 也按科技城结算，所以在任何城都能看到研究完成；
//   ③ 科研中心等级要求取「玩家所有城的最高科研中心等级」，避免「在主城能研究、
//      切到新分城就不能研究」的困惑；
//   ④ 启动时把历史数据按 (用户, 科技) 取最高等级合并进科技城，并删除分城的行
//      （幂等，见 ezfyMigrateSharedTech）。

// techCityId 该城对应的「科技归属城」= 玩家主城（id 最小的城）。
//
// 城不存在时原样返回，保证不会因为查不到而把科技写到 0 号城。
func (h *EzfyHandler) techCityId(cityId uint) uint {
	if cityId == 0 {
		return 0
	}
	var uid uint
	row := h.DB.Model(&model.EzfyCity{}).Where("id = ?", cityId).Select("user_id").Row()
	if err := row.Scan(&uid); err != nil || uid == 0 {
		return cityId
	}
	var mainID uint
	row2 := h.DB.Model(&model.EzfyCity{}).Where("user_id = ?", uid).Select("MIN(id)").Row()
	if err := row2.Scan(&mainID); err != nil || mainID == 0 {
		return cityId
	}
	return mainID
}

// maxAcademyLevel 玩家所有城市里最高的科研中心等级
func (h *EzfyHandler) maxAcademyLevel(uid uint) int {
	var ids []uint
	h.DB.Model(&model.EzfyCity{}).Where("user_id = ?", uid).Pluck("id", &ids)
	best := 0
	for _, id := range ids {
		if lv := h.buildingLevel(id, 8); lv > best {
			best = lv
		}
	}
	return best
}

var ezfySharedTechOnce sync.Once

// ezfyMigrateSharedTech 历史数据合并：把分城的科技等级并入主城。
//
// 规则：同一玩家同一科技取**最高等级**；「研究中」的状态也一并搬到主城。
// 幂等：合并后分城不再有行，后续启动是空操作。
func ezfyMigrateSharedTech(db *gorm.DB) {
	if !db.Migrator().HasTable("ezfy_city_tech") {
		return
	}
	// 每个玩家的主城
	type ownerRow struct {
		CityId uint
		UserID uint
		MainId uint
	}
	var cities []model.EzfyCity
	db.Find(&cities)
	if len(cities) == 0 {
		return
	}
	mainOf := map[uint]uint{} // uid -> main city id
	for _, c := range cities {
		if m, ok := mainOf[c.UserID]; !ok || c.ID < m {
			mainOf[c.UserID] = c.ID
		}
	}
	cityOwner := map[uint]uint{}
	for _, c := range cities {
		cityOwner[c.ID] = c.UserID
	}

	var rows []model.EzfyCityTech
	db.Find(&rows)
	moved := 0
	for _, r := range rows {
		uid, ok := cityOwner[uint(r.CityId)]
		if !ok {
			continue
		}
		mainID := mainOf[uid]
		if uint(r.CityId) == mainID {
			continue
		}
		// 分城的行 → 合并进主城
		var main model.EzfyCityTech
		err := db.Where("city_id = ? AND tech_id = ?", mainID, r.TechId).First(&main).Error
		if err != nil {
			// 主城没有 → 直接把这条搬到主城
			db.Model(&model.EzfyCityTech{}).Where("id = ?", r.ID).Update("city_id", int64(mainID))
			moved++
			continue
		}
		upd := map[string]interface{}{}
		if r.Level > main.Level {
			upd["level"] = r.Level
		}
		// 研究中的状态搬到主城（主城空闲时）
		if r.Status == 1 && main.Status != 1 {
			upd["status"] = 1
			upd["end_time"] = r.EndTime
		}
		if len(upd) > 0 {
			db.Model(&model.EzfyCityTech{}).Where("id = ?", main.ID).Updates(upd)
		}
		db.Delete(&model.EzfyCityTech{}, r.ID)
		moved++
	}
	if moved > 0 {
		log.Printf("ezfy 科技共用迁移: 合并/删除 %d 条分城科技记录", moved)
	}
}

// ezfyMoveTechTo 把科技城的科技数据搬到另一座城（摧毁主城时用，避免科技凭空消失）
func (h *EzfyHandler) ezfyMoveTechTo(fromCityId, toCityId uint) {
	if fromCityId == 0 || toCityId == 0 || fromCityId == toCityId {
		return
	}
	var rows []model.EzfyCityTech
	h.DB.Where("city_id = ?", fromCityId).Find(&rows)
	for _, r := range rows {
		var exist model.EzfyCityTech
		if err := h.DB.Where("city_id = ? AND tech_id = ?", toCityId, r.TechId).First(&exist).Error; err != nil {
			h.DB.Model(&model.EzfyCityTech{}).Where("id = ?", r.ID).Update("city_id", int64(toCityId))
			continue
		}
		if r.Level > exist.Level {
			h.DB.Model(&model.EzfyCityTech{}).Where("id = ?", exist.ID).Update("level", r.Level)
		}
		h.DB.Delete(&model.EzfyCityTech{}, r.ID)
	}
}

// ezfyUpsertTechLevel 给指定城写入某科技等级（管理端一键满级用）
func (h *EzfyHandler) ezfyUpsertTechLevel(cityId uint, techId, level int) {
	h.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "city_id"}, {Name: "tech_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"level", "status", "end_time"}),
	}).Create(&model.EzfyCityTech{
		CityId: int64(cityId), TechId: techId, Level: level, Status: 0, EndTime: 0,
	})
}
