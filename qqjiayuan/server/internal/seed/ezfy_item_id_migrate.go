package seed

import (
	"encoding/json"
	"log"

	"gorm.io/gorm"

	"qqjiayuan/server/internal/model"
)

// ★★ 2026-10-10 道具 cfg_id 全体迁到 1001+（与装备配置 1~35 彻底不重叠）
//
// 背景（线上事故）：`ezfy_cfg_item`（道具）与 `ezfy_cfg_equipment`（装备）是**两张互不相关、
// 各自从 1 开始编号**的配置表 → 数值 ID 大面积重叠（30 = 建筑加速80% 也 = 黑曜石戒指）。
// 只要有一处代码「按 ID 区间猜身份」，就会误伤玩家数据：
// `ezfyMigrateTreasureBag` 按 `cfg_id 27~35` 把玩家真实加速道具当「误发宝物」删掉换成珠宝装备，
// 而且它挂在启动路径上（sync.Once 只对进程内有效）→ **每次重新部署都清空一次**，
// 玩家反馈「买的道具、打野地掉的道具丢了」。
//
// 修复分两层：
//  1. 危险迁移永久摘除（见 handler/ezfy/ezfy.go 的 cfgs()）+ 静态断言；
//  2. 本迁移把道具 ID 整体搬出重叠区间（1~41 → 1001~1041），从根上消除「同 ID」。
const (
	// ezfyItemIDShift 改号偏移：旧号 + 1000 = 新号
	ezfyItemIDShift = 1000
	// ezfyItemIDLegacyMax 旧号上限（道具旧号 1~41，含已废弃的 26/27 发电卡）
	ezfyItemIDLegacyMax = 41
	// ezfyItemIDLegacyProbe 判定「还是老号」的探测上限（老号最大 41，取 100 留余量）
	ezfyItemIDLegacyProbe = 100
)

// EnsureEzfyItemIDsHighRange 幂等 + 事务：把道具 cfg_id 从 1~41 迁到 1001~1041。
//
// 幂等：只在「ezfy_cfg_item 里仍存在 id <= 100 的行」时执行；执行后老号消失 → 再跑即 no-op。
// 事务：全部改写在一个事务里，失败整体回滚，不会留下「半迁移」状态。
//
// ⚠️ 调用时机：必须跑在 `seed.Run`（道具种子已按新号 1001+ 灌数据）**之前**，
// 否则会同时出现老号行和新号行（商城列表出现重复道具）。见 main.go。
// 返回 error 时**必须中止启动**：带着老号继续跑，种子会再灌一套新号配置 → 道具重复。
func EnsureEzfyItemIDsHighRange(db *gorm.DB) error {
	var legacy int64
	if err := db.Model(&model.EzfyCfgItem{}).Where("id <= ?", ezfyItemIDLegacyProbe).Count(&legacy).Error; err != nil {
		return err
	}
	if legacy == 0 {
		return nil // 已迁移过（或全新库直接由种子按新号灌）→ no-op
	}
	log.Printf("ezfy 道具改号: 检测到 %d 行老号道具(id<=%d)，迁到 +%d …",
		legacy, ezfyItemIDLegacyProbe, ezfyItemIDShift)

	err := db.Transaction(func(tx *gorm.DB) error {
		// ① 道具配置本体
		if err := tx.Exec("UPDATE ezfy_cfg_item SET id = id + ? WHERE id BETWEEN 1 AND ?",
			ezfyItemIDShift, ezfyItemIDLegacyMax).Error; err != nil {
			return err
		}
		// ② 玩家背包
		if err := tx.Exec("UPDATE ezfy_item SET cfg_id = cfg_id + ? WHERE cfg_id BETWEEN 1 AND ?",
			ezfyItemIDShift, ezfyItemIDLegacyMax).Error; err != nil {
			return err
		}
		// ③ 道具使用流水
		if err := tx.Exec("UPDATE ezfy_item_use_logs SET cfg_id = cfg_id + ? WHERE cfg_id BETWEEN 1 AND ?",
			ezfyItemIDShift, ezfyItemIDLegacyMax).Error; err != nil {
			return err
		}
		// ④ 宝箱奖池：kind=2 才是道具（kind=1 装备 / kind=3 套装 别动）
		if tx.Migrator().HasTable("ezfy_cfg_chest_item") {
			if err := tx.Exec("UPDATE ezfy_cfg_chest_item SET ref_id = ref_id + ? WHERE kind = 2 AND ref_id BETWEEN 1 AND ?",
				ezfyItemIDShift, ezfyItemIDLegacyMax).Error; err != nil {
				return err
			}
		}
		// ⑤ 军团商城：kind=2 才是道具
		if tx.Migrator().HasTable("ezfy_corps_mall") {
			if err := tx.Exec("UPDATE ezfy_corps_mall SET item_id = item_id + ? WHERE kind = 2 AND item_id BETWEEN 1 AND ?",
				ezfyItemIDShift, ezfyItemIDLegacyMax).Error; err != nil {
				return err
			}
		}
		// ⑥ 野地类型「商城道具掉落」JSON：[[道具id, 数量, 概率%], ...]
		return ezfyShiftWildlandDropItems(tx)
	})
	if err != nil {
		log.Printf("ezfy 道具改号失败（已回滚，仍是老号）：%v", err)
		return err
	}
	log.Printf("ezfy 道具改号完成：1~%d → %d~%d", ezfyItemIDLegacyMax,
		1+ezfyItemIDShift, ezfyItemIDLegacyMax+ezfyItemIDShift)
	return nil
}

// ezfyShiftWildlandDropItems 把 ezfy_cfg_wildland.drop_items 里 1~41 的道具 id 全部 +1000。
// 非合法 JSON（历史文本值）一律跳过不动；表/列不存在直接跳过。
func ezfyShiftWildlandDropItems(tx *gorm.DB) error {
	if !tx.Migrator().HasTable("ezfy_cfg_wildland") || !tx.Migrator().HasColumn("ezfy_cfg_wildland", "drop_items") {
		return nil
	}
	var rows []model.EzfyCfgWildland
	if err := tx.Where("drop_items IS NOT NULL AND drop_items <> '' AND drop_items <> '[]'").
		Find(&rows).Error; err != nil {
		return err
	}
	for i := range rows {
		out, changed, ok := ezfyShiftDropItemsJSON(rows[i].DropItems)
		if !ok || !changed {
			continue
		}
		if err := tx.Model(&model.EzfyCfgWildland{}).Where("id = ?", rows[i].ID).
			Update("drop_items", out).Error; err != nil {
			return err
		}
	}
	return nil
}

// ezfyShiftDropItemsJSON 把野地「商城道具掉落」JSON（[[道具id, 数量, 概率%], ...]）里
// 1~41 的道具 id 全部 +1000。
//
// 返回 (新 JSON, 是否有改动, 是否为合法 JSON)。不是合法 JSON（历史文本值如「珠宝(平原)」）
// 时 ok=false —— 调用方跳过不动，保持兼容。
func ezfyShiftDropItemsJSON(raw string) (string, bool, bool) {
	var arr [][]int
	if err := json.Unmarshal([]byte(raw), &arr); err != nil {
		return raw, false, false
	}
	changed := false
	for j := range arr {
		if len(arr[j]) > 0 && arr[j][0] >= 1 && arr[j][0] <= ezfyItemIDLegacyMax {
			arr[j][0] += ezfyItemIDShift
			changed = true
		}
	}
	if !changed {
		return raw, false, true
	}
	out, err := json.Marshal(arr)
	if err != nil {
		return raw, false, false
	}
	return string(out), true, true
}
