package seed

import (
	"log"
	"time"

	"gorm.io/gorm"

	"qqjiayuan/server/internal/model"
)

// EnsureEzfyWildlandUnique 幂等清理 ezfy_wildland 的存量脏数据，保证「一个坐标只有一条野地记录」。
//
// ★★ 2026-10-16 背景（线上实测 153 个坐标重复、其中 81 个坐标归属冲突）：
//
//	`processArrive` 的「征服野地/寇城 → 占领」原来**无条件 Create** 新的 ezfy_wildland 行；
//	若该坐标早已被别人占领（表里已有 city_id = 他人城的行），就会留下同坐标多条记录。后果：
//	  - 野地详情 `WildlandView`、建新城 `CreateCity` 都用 `First()` → 取到**旧行**（原主人），
//	    于是「地图显示归我、详情却写归属别人、mine=false」，我还能对自己的地反复下单刷资源；
//	  - 原主人的「附属野地」列表仍含这块地 → 白占他的野地上限、还能在上面采集；
//	  - `len(wildlandList(cityID))` 虚高 → 明明有空位却被判「附属野地已达上限」。
//
//	新代码已在「占领 / 放弃野地」时清同坐标旧行（`EzfyHandler.ezfyClearWildlandAt`），
//	这里负责把**存量**收口：每个坐标只保留 **id 最大（= 最后一次占领）** 的那条，其余删除。
//
// 同时清理第二类残留：**坐标上已经有城市**的野地行（建城时本该被删，历史版本漏删）——
//
//	这类行在地图上被城市覆盖看不见，却照样占玩家的附属野地上限。
//
// 幂等 / 安全：
//   - 没有重复、也没有「城市格野地」时纯读，no-op（绝不整表重写）；
//   - 保留 id 最大的一条 → 归属与「最后占领者」一致，不会误判归属；
//   - 删行前把被删行上**未结束**的采集(4)/驻守(7)部队转返航（口径同
//     `EzfyHandler.ezfyWildlandOccupiedCount`：order_type=4 的 status(0,1,5,6,98)
//     + order_type=7 的 status=1），否则部队 target_id 悬空 → 凭空消失 / 结算异常。
func EnsureEzfyWildlandUnique(db *gorm.DB) {
	if db == nil || !db.Migrator().HasTable("ezfy_wildland") {
		return
	}
	type xyRow struct {
		X int
		Y int
		N int64
	}
	// ① 同坐标重复
	var dups []xyRow
	if err := db.Model(&model.EzfyWildland{}).
		Select("x, y, COUNT(*) AS n").
		Group("x, y").Having("COUNT(*) > 1").Find(&dups).Error; err != nil {
		log.Printf("ezfy 清理重复野地失败(分组查询): %v", err)
		return
	}
	// ② 坐标已被城市占用（城市格上不该再有野地行）
	var onCity []xyRow
	if db.Migrator().HasTable("ezfy_city") {
		if err := db.Table("ezfy_wildland AS w").
			Select("w.x AS x, w.y AS y, COUNT(*) AS n").
			Joins("JOIN ezfy_city c ON c.x = w.x AND c.y = w.y").
			Group("w.x, w.y").Find(&onCity).Error; err != nil {
			log.Printf("ezfy 清理城市格野地失败(查询): %v", err)
			onCity = nil
		}
	}
	if len(dups) == 0 && len(onCity) == 0 {
		return
	}
	now := time.Now().UnixMilli()
	removedRows, returnedOrders, removedOnCity := 0, 0, 0

	// 转返航 + 删行（被删 id 集合）
	clear := func(ids []int64) {
		if len(ids) == 0 {
			return
		}
		var orders []model.EzfyOrder
		db.Where("target_id IN ? AND ((order_type = 4 AND status IN (0,1,5,6,98)) OR (order_type = 7 AND status = 1))", ids).
			Find(&orders)
		for i := range orders {
			o := &orders[i]
			travel := int64(60000)
			if t := (o.ReturnTime - o.StartTime) / 2; t > 0 {
				travel = t
			} else if t := o.ArriveTime - o.StartTime; t > 0 {
				travel = t
			}
			if err := db.Model(&model.EzfyOrder{}).Where("id = ?", o.ID).
				Updates(map[string]interface{}{"status": 2, "result": o.Troops,
					"return_time": now + travel, "carry": o.Carry}).Error; err != nil {
				log.Printf("ezfy 重复野地部队返航失败(order=%d): %v", o.ID, err)
				continue
			}
			returnedOrders++
		}
		if err := db.Where("id IN ?", ids).Delete(&model.EzfyWildland{}).Error; err != nil {
			log.Printf("ezfy 删除重复野地失败: %v", err)
			return
		}
		removedRows += len(ids)
	}

	// ① 重复坐标：保留 id 最大的一条
	for _, d := range dups {
		var rows []model.EzfyWildland
		if err := db.Select("id").Where("x = ? AND y = ?", d.X, d.Y).Order("id ASC").Find(&rows).Error; err != nil {
			continue
		}
		if len(rows) < 2 {
			continue
		}
		keepID := rows[len(rows)-1].ID
		ids := make([]int64, 0, len(rows)-1)
		for i := range rows {
			if rows[i].ID != keepID {
				ids = append(ids, int64(rows[i].ID))
			}
		}
		clear(ids)
	}
	// ② 城市格上的野地行：整组删掉
	for _, d := range onCity {
		var rows []model.EzfyWildland
		if err := db.Select("id").Where("x = ? AND y = ?", d.X, d.Y).Find(&rows).Error; err != nil {
			continue
		}
		ids := make([]int64, 0, len(rows))
		for i := range rows {
			ids = append(ids, int64(rows[i].ID))
		}
		before := removedRows
		clear(ids)
		removedOnCity += removedRows - before
	}
	if removedRows > 0 {
		log.Printf("ezfy 已清理 %d 条野地脏数据（重复坐标 %d 个 / 城市格残留 %d 条；%d 支采集/驻守部队已返航）",
			removedRows, len(dups), removedOnCity, returnedOrders)
	}
}
