package handler

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm/clause"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 二战风云管理端 —— 地图格子覆盖
//
// ★ 用户要求：「所有野地管理端也要能维护呀，而且能够改变土地类型，也能设置寇城、活动寇城」。
//
// 本项目的地图（地形 / 野地等级 / 寇城 / 活动目标）**全部是坐标哈希推导的、不落库**，
// 所以这里做一张覆盖表 ezfy_map_tile：
//   - Terrain   > 0 时强制该格地形（1平原…8海洋, 9沿海平原）
//   - MarkKind  > 0 时强制该格标记（1寇城 2活动寇城 3活动野地 4特殊城市）
//
// 覆盖优先于哈希，改完调 ezfyReload() 立即生效（不用重启）。

var ezfyTileFields = map[string]string{
	"x": "int", "y": "int", "terrain": "int",
	"mark_kind": "int", "mark_level": "int", "des": "string",
}

// ezfyMarkKindName 标记类型中文名
func ezfyMarkKindName(k int) string {
	switch k {
	case model.EzfyMarkKou:
		return "寇城"
	case model.EzfyMarkActKou:
		return "活动寇城"
	case model.EzfyMarkActWild:
		return "活动野地"
	case model.EzfyMarkActCity:
		return "特殊城市"
	}
	return "无"
}

// AdminEzfyMapTiles GET /admin/ezfy-map-tiles —— 已配置的格子覆盖列表
func (h *AdminHandler) AdminEzfyMapTiles(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	word := strings.TrimSpace(c.Query("word"))
	mark := atoiOr(c.Query("mark_kind"), -1)
	q := h.DB.Model(&model.EzfyMapTile{})
	if word != "" {
		// 支持「x,y」精确坐标，也支持备注模糊
		if parts := strings.SplitN(word, ",", 2); len(parts) == 2 {
			if x, e1 := strconv.Atoi(strings.TrimSpace(parts[0])); e1 == nil {
				if y, e2 := strconv.Atoi(strings.TrimSpace(parts[1])); e2 == nil {
					q = q.Where("x = ? AND y = ?", x, y)
				}
			}
		} else {
			q = q.Where("des LIKE ?", "%"+word+"%")
		}
	}
	if mark >= 0 {
		q = q.Where("mark_kind = ?", mark)
	}
	var total int64
	q.Count(&total)
	var rows []model.EzfyMapTile
	q.Order("id DESC").Offset(offset).Limit(size).Find(&rows)

	out := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		out = append(out, gin.H{
			"id": r.ID, "x": r.X, "y": r.Y,
			"terrain": r.Terrain, "terrain_name": ezfyTerrainName(r.Terrain),
			// 该格「实际生效」的地形/标记（覆盖 or 哈希）
			"eff_terrain": r.Terrain, "eff_terrain_name": ezfyTerrainName(r.Terrain),
			"mark_kind": r.MarkKind, "mark_name": ezfyMarkKindName(r.MarkKind),
			"mark_level": r.MarkLevel, "des": r.Des,
			"updated_at": r.UpdatedAt,
		})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// AdminEzfyMapTileCell GET /admin/ezfy-map-tile?x=&y= —— 查某格（含哈希结果与覆盖结果）
//
// 供管理端「按坐标设置」用：先看这格现在是什么（哈希/覆盖），再决定怎么改。
func (h *AdminHandler) AdminEzfyMapTileCell(c *gin.Context) {
	x := atoiOr(c.Query("x"), -1)
	y := atoiOr(c.Query("y"), -1)
	if x < 0 || y < 0 {
		resp.ParamError(c, "请填写坐标 x / y")
		return
	}
	ez := h.ezfyH()
	hashTerrain := ezfyTerrainEx(x, y)
	hashMark := 0
	if ez.ezfyIsKouCity(x, y) {
		hashMark = model.EzfyMarkKou
	}
	if act := ezfyActTypeFor(x, y, ez.ezfyIsKouCity(x, y)); act > 0 {
		switch act {
		case ezfyActKou:
			hashMark = model.EzfyMarkActKou
		case ezfyActWild:
			hashMark = model.EzfyMarkActWild
		case ezfyActCity:
			hashMark = model.EzfyMarkActCity
		}
	}

	var ov model.EzfyMapTile
	hasOv := h.DB.Where("x = ? AND y = ?", x, y).First(&ov).Error == nil

	effTerrain, effMark, effLevel := hashTerrain, hashMark, ezfyActivityLevel(x, y)
	if hasOv {
		if ov.Terrain > 0 {
			effTerrain = ov.Terrain
		}
		if ov.MarkKind > 0 {
			effMark = ov.MarkKind
		}
		if ov.MarkLevel > 0 {
			effLevel = ov.MarkLevel
		}
	}

	// 该格有没有玩家城 / 玩家野地
	var cityName, wildOwner string
	var ct model.EzfyCity
	if h.DB.Where("x = ? AND y = ?", x, y).First(&ct).Error == nil {
		cityName = ct.Name
	}
	var wl model.EzfyWildland
	if h.DB.Where("x = ? AND y = ?", x, y).First(&wl).Error == nil {
		var c2 model.EzfyCity
		if h.DB.First(&c2, wl.CityId).Error == nil {
			wildOwner = c2.Name
		}
	}

	resp.OK(c, gin.H{
		"x": x, "y": y,
		"hash_terrain": hashTerrain, "hash_terrain_name": ezfyTerrainName(hashTerrain),
		"hash_mark": hashMark, "hash_mark_name": ezfyMarkKindName(hashMark),
		"eff_terrain": effTerrain, "eff_terrain_name": ezfyTerrainName(effTerrain),
		"eff_mark": effMark, "eff_mark_name": ezfyMarkKindName(effMark),
		"eff_level":      effLevel,
		"has_override":   hasOv,
		"override":       ov,
		"city_name":      cityName,
		"wild_owner":     wildOwner,
		"wildland_level": ezfyWildlandLevel(x, y),
	})
}

// AdminEzfyMapTileSave POST /admin/ezfy-map-tiles —— 新增或更新某格的覆盖（按 x,y 幂等）
func (h *AdminHandler) AdminEzfyMapTileSave(c *gin.Context) {
	var in map[string]interface{}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	vals := xyPickVals(in, ezfyTileFields)
	x, _ := vals["x"].(int)
	y, _ := vals["y"].(int)
	if x < 0 || y < 0 {
		resp.ParamError(c, "请填写坐标 x / y")
		return
	}
	if msg := ezfyCheckTileVals(vals); msg != "" {
		resp.ParamError(c, msg)
		return
	}
	vals["updated_at"] = time.Now()
	// ★ 按 (x,y) 唯一索引 upsert —— 同一格重复保存不会产生多行
	if err := h.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "x"}, {Name: "y"}},
		DoUpdates: clause.AssignmentColumns([]string{"terrain", "mark_kind", "mark_level", "des", "updated_at"}),
	}).Model(&model.EzfyMapTile{}).Create(vals).Error; err != nil {
		resp.ParamError(c, "保存失败："+err.Error())
		return
	}
	h.ezfyReload()
	resp.OK(c, gin.H{"msg": fmt.Sprintf("坐标 (%d,%d) 已保存并立即生效", x, y)})
}

// AdminEzfyMapTileDelete DELETE /admin/ezfy-map-tiles/:id —— 删掉覆盖，恢复按哈希
func (h *AdminHandler) AdminEzfyMapTileDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var t model.EzfyMapTile
	if err := h.DB.First(&t, id).Error; err != nil {
		resp.NotFound(c, "该覆盖配置不存在")
		return
	}
	h.DB.Delete(&model.EzfyMapTile{}, id)
	h.ezfyReload()
	resp.OK(c, gin.H{"msg": fmt.Sprintf("坐标 (%d,%d) 的覆盖已删除，恢复按地图默认规则", t.X, t.Y)})
}

// ezfyCheckTileVals 校验覆盖值
func ezfyCheckTileVals(vals map[string]interface{}) string {
	if t, ok := vals["terrain"].(int); ok && t != 0 {
		if t < 1 || t > 9 {
			return "地形只能是 1~9（1平原…8海洋 9沿海平原），0 = 不覆盖"
		}
	}
	if k, ok := vals["mark_kind"].(int); ok && k != 0 {
		if k < 1 || k > 4 {
			return "标记只能是 0无 1寇城 2活动寇城 3活动野地 4特殊城市"
		}
	}
	if lv, ok := vals["mark_level"].(int); ok && lv != 0 {
		if lv < 1 || lv > 3 {
			return "活动等级只能是 1~3"
		}
	}
	return ""
}
