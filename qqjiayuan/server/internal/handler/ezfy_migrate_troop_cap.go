package handler

import (
	"log"
	"sync"

	"gorm.io/gorm"

	"qqjiayuan/server/internal/model"
)

// 一次性数据迁移：把「超出城墙容量的城防兵」按比例缩回容量上限
//
// 背景：
//   - 早期管理端「发放部队 / 修改部队」没有数量上限（现已加 1 亿上限校验），
//     导致生产库里出现离谱的驻军：例如某城 1300 万反坦克炮 + 500 万榴弹炮，
//     而围墙 10 级的城防空间只有 10 万。
//   - 用户规则：「兵没有上限，城防超过的改成最高值即可，不能超过城墙等级空间。」
//     → 即**只有城防兵种(type 4)**受城防空间约束，超出的按原兵种比例等比缩容，
//       总占用刚好压回 capacity；陆军/海军/空军(type 1/2/3)不受此限制，保持原样。
//
// 幂等：缩容后各城 type4 总量 = capacity，后续启动不会再触发。
var ezfyTroopCapOnce sync.Once

// ezfyTroopTypeDefence 城防兵种（碉堡/榴弹炮/反坦克炮/防空炮）
const ezfyTroopTypeDefence = 4

// ezfyMigrateTroopCap 城防兵超容量 → 按比例缩回容量上限
func ezfyMigrateTroopCap(db *gorm.DB) {
	// 所有城的围墙容量
	type wallRow struct {
		CityID   int64
		Capacity int64
	}
	var walls []wallRow
	db.Raw(`SELECT b.city_id AS city_id, l.capacity AS capacity
	        FROM ezfy_city_building b
	        JOIN ezfy_cfg_building_level l ON l.building_id = b.building_id AND l.level = b.level
	        WHERE b.building_id = 7`).Scan(&walls)
	capOf := map[int64]int64{}
	for _, w := range walls {
		capOf[w.CityID] = w.Capacity
	}

	// 各城城防兵(type 4)明细
	type troopRow struct {
		ID      int64
		CityID  int64
		TroopID int
		Count   int64
	}
	var rows []troopRow
	db.Raw(`SELECT ct.id AS id, ct.city_id AS city_id, ct.troop_id AS troop_id, ct.count AS count
	        FROM ezfy_city_troop ct
	        JOIN ezfy_cfg_troop t ON t.id = ct.troop_id
	        WHERE t.type = ?`, ezfyTroopTypeDefence).Scan(&rows)

	byCity := map[int64][]troopRow{}
	for _, r := range rows {
		byCity[r.CityID] = append(byCity[r.CityID], r)
	}

	fixed := 0
	for cityID, list := range byCity {
		var used int64
		for _, r := range list {
			used += r.Count
		}
		capacity := capOf[cityID] // 没围墙 → 0，全部清掉
		if used <= capacity {
			continue
		}
		if capacity <= 0 {
			// 无围墙：城防空间为 0，直接清零
			for _, r := range list {
				db.Model(&model.EzfyCityTroop{}).Where("id = ?", r.ID).Update("count", 0)
				fixed++
			}
			continue
		}
		// 按原兵种比例等比缩容，最后一个兵种吃掉取整误差，保证总量恰好 = capacity
		var assigned int64
		for i, r := range list {
			var newCount int64
			if i == len(list)-1 {
				newCount = capacity - assigned
				if newCount < 0 {
					newCount = 0
				}
			} else {
				newCount = r.Count * capacity / used
			}
			assigned += newCount
			if newCount == r.Count {
				continue
			}
			db.Model(&model.EzfyCityTroop{}).Where("id = ?", r.ID).Update("count", newCount)
			fixed++
		}
	}
	if fixed > 0 {
		log.Printf("ezfy 城防超容迁移: %d 条城防兵已按城墙容量等比缩容", fixed)
	}
}
