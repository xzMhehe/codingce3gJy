package seed

import (
	"encoding/json"
	"log"
	"strings"

	"gorm.io/gorm"

	"qqjiayuan/server/internal/model"
)

// ezfyLegacyJewelNames 「老珠宝」名单（非采集宝物的历史珠宝）。
//
// ★ 2026-10-11 「老珠宝全部删除，种子里面也都去掉」：这 8 种已从 ezfy_cfg_equipment
// 删除（配置 ID 原为 19~26，号段作废），玩家实例/军官 JSON 里的残留由
// 仓库根 `删除-老珠宝-2026-10-11.sql` 清理。这里只处理**掉落配置**里的残留（见下）。
var ezfyLegacyJewelNames = []string{
	"红宝石", "蓝宝石", "绿宝石", "翡翠", "紫水晶", "猫眼石", "玛瑙", "珍珠",
}

// ensureEzfyNoLegacyJewelDrop 清「地图管理 → 野地类型 → 宝物掉落」配置里残留的老珠宝条目。
//
// ★ 2026-10-11 「二战地图管理 → 宝物掉落 那里，老宝物也要不能配置掉落」：
//
//	老珠宝已从 ezfy_cfg_equipment 删除，但**野地类型的掉落配置**（ezfy_cfg_wildland.treasure）
//	里还留着它们（线上/本地各 20+ 行，见 id 2~29 那批）。后果：
//	  ① 玩家打赢照配也不掉（`equipmentByName` 解析不到 → 静默跳过）→「配了却永远不掉」；
//	  ② 管理端打开还看得到这些老珠宝，一保存就被 `ezfyCheckWildTreasure` 拒绝。
//	所以启动时幂等清一次：按**名字**剔除老珠宝条目，剔完为空则落空串。
//
// 幂等/安全：
//   - **只在确实含老珠宝时才写库**（无残留 = 纯读，no-op；绝不整表重写）；
//   - 只动 `treasure` 列，按行 UPDATE，不碰管理端配置的其它字段；
//   - 不是合法 JSON（如历史文本值「中级/高级」）的行直接跳过，不误伤。
func ensureEzfyNoLegacyJewelDrop(db *gorm.DB) {
	if db == nil || !db.Migrator().HasTable("ezfy_cfg_wildland") {
		return
	}
	legacy := make(map[string]bool, len(ezfyLegacyJewelNames))
	for _, n := range ezfyLegacyJewelNames {
		legacy[n] = true
	}
	var rows []model.EzfyCfgWildland
	if err := db.Select("id", "treasure").Find(&rows).Error; err != nil {
		log.Printf("ezfy 清理老珠宝掉落配置失败(读取): %v", err)
		return
	}
	cleaned := 0
	for _, r := range rows {
		raw := strings.TrimSpace(r.Treasure)
		if raw == "" || !strings.Contains(raw, "name") {
			continue
		}
		var items []struct {
			Name  string `json:"name"`
			Count int    `json:"count"`
			Pct   int    `json:"pct"`
		}
		if err := json.Unmarshal([]byte(raw), &items); err != nil {
			continue // 老文本值 / 非法 JSON → 不碰
		}
		kept := make([]map[string]interface{}, 0, len(items))
		hit := false
		for _, it := range items {
			if legacy[strings.TrimSpace(it.Name)] {
				hit = true
				continue
			}
			kept = append(kept, map[string]interface{}{
				"name": it.Name, "count": it.Count, "pct": it.Pct,
			})
		}
		if !hit {
			continue
		}
		out := ""
		if len(kept) > 0 {
			b, err := json.Marshal(kept)
			if err != nil {
				log.Printf("ezfy 清理老珠宝掉落配置失败(id=%d 序列化): %v", r.ID, err)
				continue
			}
			out = string(b)
		}
		if err := db.Model(&model.EzfyCfgWildland{}).Where("id = ?", r.ID).
			Update("treasure", out).Error; err != nil {
			log.Printf("ezfy 清理老珠宝掉落配置失败(id=%d): %v", r.ID, err)
			continue
		}
		cleaned++
	}
	if cleaned > 0 {
		log.Printf("ezfy 已清理 %d 条野地类型掉落配置里的老珠宝（老宝物不再可配置掉落）", cleaned)
	}
}
