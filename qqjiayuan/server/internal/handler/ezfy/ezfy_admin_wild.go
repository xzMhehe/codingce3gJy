package ezfy

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 二战风云管理端 —— 地图野地 / 科技配置 / 资源名称
//
// 四块内容：
//   1. 野地类型维护（ezfy_cfg_wildland）：野地/海野/寇城的等级、守军、产出区间、宝物
//   2. 玩家野地维护（ezfy_wildland）：全量列表 + 新增 + 编辑 + 删除（原先只有只读列表）
//   3. 科技配置维护（ezfy_cfg_tech / ezfy_cfg_tech_level）：原先只能只读参考
//   4. 资源名称维护（ezfy_cfg_resource）：金/粮/钢/油/稀 可改名，全站展示跟随
//
// ★ 这几张表在 seed 里都是「只补缺、不覆盖」(DoNothing)，管理端的修改能活过重启。

// ============ 公共：字段白名单 ============

// 野地类型配置可改字段（ezfy_cfg_wildland）
var ezfyWildCfgFields = map[string]string{
	"type": "int", "level": "int", "troops": "string",
	"res_min": "int64", "res_max": "int64",
	"officer_min": "int", "officer_max": "int", "officer_id": "int",
	"treasure": "string", "drop_items": "string", "des": "string",
}

// 玩家野地可改字段（ezfy_wildland）
var ezfyWildFields = map[string]string{
	"city_id": "int64", "x": "int", "y": "int", "wild_type": "int",
	"level": "int", "gain": "string", "status": "int",
	"start_time": "int64", "end_time": "int64",
}

// 科技配置可改字段（ezfy_cfg_tech）
var ezfyTechCfgFields = map[string]string{
	"name": "string", "type": "int", "max_level": "int",
	"pre_building": "int", "pre_tech": "int", "pre_tech_level": "int",
	"effect": "string", "des": "string",
}

// 科技等级配置可改字段（ezfy_cfg_tech_level）
var ezfyTechLevelFields = map[string]string{
	"tech_id": "int", "level": "int",
	"food": "int64", "steel": "int64", "oil": "int64", "rare": "int64", "gold": "int64",
	"research_time": "int", "effect": "string",
}

// 资源名称配置可改字段（ezfy_cfg_resource）
var ezfyResCfgFields = map[string]string{
	"name": "string", "short": "string", "sort": "int",
}

func ezfyWildTypeName(t int) string {
	switch t {
	case 1:
		return "陆地野地"
	case 2:
		return "海野"
	case 3:
		return "寇城"
	}
	return "其他"
}

func ezfyWildStatusName(s int) string {
	switch s {
	case 1:
		return "采集中"
	case 2:
		return "已占领"
	}
	return "空闲"
}

// ============ 1. 野地类型维护（ezfy_cfg_wildland） ============

// AdminEzfyMapOptions GET /admin/ezfy-map/options
//
// 给「野地类型」弹窗用的下拉数据：兵种列表（守军搭配）+ 军官池（守军军官，最多选 1 个）。
// 单独开这个接口是为了不让地图管理依赖兵种管理/军官管理的权限。
func (h *EzfyAdmin) AdminEzfyMapOptions(c *gin.Context) {
	var troops []model.EzfyCfgTroop
	h.DB.Order("id").Find(&troops)
	tviews := make([]gin.H, 0, len(troops))
	for _, t := range troops {
		tviews = append(tviews, gin.H{"id": t.ID, "name": t.Name, "type": t.Type,
			"type_name": ezfyTroopTypeName(t.Type)})
	}
	var gens []model.EzfyCfgGeneral
	h.DB.Order("id").Find(&gens)
	gviews := make([]gin.H, 0, len(gens))
	for _, g := range gens {
		gviews = append(gviews, gin.H{"id": g.ID, "name": g.Name, "star": g.Star, "kind": g.Kind,
			"military": g.Military, "logistics": g.Logistics, "learning": g.Learning})
	}
	// ★ 2026-09-29 活动野地「必掉宝物」多行编辑器下拉：珠宝类装备（cfg_id 落到 ezfy_cfg_equipment）
	var equips []model.EzfyCfgEquipment
	h.DB.Where("type = ?", "珠宝").Order("id").Find(&equips)
	jviews := make([]gin.H, 0, len(equips))
	for _, e := range equips {
		jviews = append(jviews, gin.H{"id": e.ID, "name": e.Name})
	}
	// ★ 2026-10-05 商城道具下拉：野地类型「商城道具掉落」行编辑器用（运营下拉选择，不用手写 JSON）
	var items []model.EzfyCfgItem
	h.DB.Order("id").Find(&items)
	iviews := make([]gin.H, 0, len(items))
	for _, it := range items {
		iviews = append(iviews, gin.H{"id": it.ID, "name": it.Name})
	}
	resp.OK(c, gin.H{"troops": tviews, "generals": gviews, "jewels": jviews, "items": iviews})
}

// AdminEzfyWildCfgList 野地类型配置列表
func (h *EzfyAdmin) AdminEzfyWildCfgList(c *gin.Context) {
	page, offset, size := pageOf(c, 20)
	wType := atoiOr(c.Query("type"), -1)
	level := atoiOr(c.Query("level"), 0)
	q := h.DB.Model(&model.EzfyCfgWildland{})
	if wType >= 0 {
		q = q.Where("type = ?", wType)
	}
	if level > 0 {
		q = q.Where("level = ?", level)
	}
	var total int64
	q.Count(&total)
	var rows []model.EzfyCfgWildland
	q.Order("type, level, id").Offset(offset).Limit(size).Find(&rows)
	out := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		out = append(out, gin.H{
			"id": r.ID, "type": r.Type, "type_name": ezfyWildTypeName(r.Type),
			"level": r.Level, "troops": r.Troops,
			"res_min": r.ResMin, "res_max": r.ResMax,
			"officer_min": r.OfficerMin, "officer_max": r.OfficerMax,
			// ★ 2026-10-05 性能：守将名走配置缓存（原 ezfyGeneralName 每行一条 SQL → 一页 20 条）
			"officer_id": r.OfficerId, "officer_name": h.ezfyGeneralNameCached(r.OfficerId),
			"treasure":   r.Treasure, "drop_items": r.DropItems, "des": r.Des,
		})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// ezfyGeneralName 军官池（ezfy_cfg_general）里的军官名
func (h *EzfyAdmin) ezfyGeneralName(id int) string {
	if id <= 0 {
		return ""
	}
	var g model.EzfyCfgGeneral
	if err := h.DB.First(&g, id).Error; err != nil {
		return ""
	}
	return g.Name
}

// ezfyCheckWildOfficer 校验野地的守军军官
//
// ★ 用户规则：野地最多只能配置**一个**军官，而且必须来自「军官池」（ezfy_cfg_general）。
func (h *EzfyAdmin) ezfyCheckWildOfficer(vals map[string]interface{}) string {
	raw, ok := vals["officer_id"]
	if !ok {
		return ""
	}
	id, _ := raw.(int)
	if id <= 0 {
		return "" // 0 = 不设守将
	}
	var g model.EzfyCfgGeneral
	if err := h.DB.First(&g, id).Error; err != nil {
		return "军官池里没有这个军官（ID " + strconv.Itoa(id) + "）"
	}
	return ""
}

// AdminEzfyWildCfgCreate 新增野地类型配置
func (h *EzfyAdmin) AdminEzfyWildCfgCreate(c *gin.Context) {
	var in map[string]interface{}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	vals := xyPickVals(in, ezfyWildCfgFields)
	if _, ok := vals["type"]; !ok {
		resp.ParamError(c, "请选择野地类型")
		return
	}
	if _, ok := vals["level"]; !ok {
		resp.ParamError(c, "请填写野地等级")
		return
	}
	if msg := h.ezfyCheckWildOfficer(vals); msg != "" {
		resp.ParamError(c, msg)
		return
	}
	if err := h.DB.Model(&model.EzfyCfgWildland{}).Create(vals).Error; err != nil {
		resp.ParamError(c, "新增失败："+err.Error())
		return
	}
	h.ezfyReload()
	resp.OK(c, gin.H{"msg": "野地类型已新增"})
}

// AdminEzfyWildCfgUpdate 修改野地类型配置
func (h *EzfyAdmin) AdminEzfyWildCfgUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var cfg model.EzfyCfgWildland
	if err := h.DB.First(&cfg, id).Error; err != nil {
		resp.NotFound(c, "野地类型不存在")
		return
	}
	var in map[string]interface{}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	vals := xyPickVals(in, ezfyWildCfgFields)
	if len(vals) == 0 {
		resp.ParamError(c, "无可修改字段")
		return
	}
	if msg := h.ezfyCheckWildOfficer(vals); msg != "" {
		resp.ParamError(c, msg)
		return
	}
	if err := h.DB.Model(&model.EzfyCfgWildland{}).Where("id = ?", id).Updates(vals).Error; err != nil {
		resp.ParamError(c, "修改失败："+err.Error())
		return
	}
	h.ezfyReload()
	resp.OK(c, gin.H{"msg": "野地类型已保存"})
}

// AdminEzfyWildCfgDelete 删除野地类型配置
func (h *EzfyAdmin) AdminEzfyWildCfgDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var cfg model.EzfyCfgWildland
	if err := h.DB.First(&cfg, id).Error; err != nil {
		resp.NotFound(c, "野地类型不存在")
		return
	}
	h.DB.Delete(&model.EzfyCfgWildland{}, id)
	h.ezfyReload()
	resp.OK(c, gin.H{"msg": "野地类型已删除"})
}

// ============ 2. 玩家野地维护（ezfy_wildland） ============

// AdminEzfyWildlandList 玩家野地列表（支持按城池/类型/状态/坐标筛选）
func (h *EzfyAdmin) AdminEzfyWildlandList(c *gin.Context) {
	page, offset, size := pageOf(c, 20)
	word := strings.TrimSpace(c.Query("word"))
	wType := atoiOr(c.Query("type"), -1)
	status := atoiOr(c.Query("status"), -1)
	q := h.DB.Model(&model.EzfyWildland{})
	if wType >= 0 {
		q = q.Where("wild_type = ?", wType)
	}
	if status >= 0 {
		q = q.Where("status = ?", status)
	}
	if word != "" {
		if n, err := strconv.Atoi(word); err == nil {
			// 纯数字：城池ID / 坐标 x / 坐标 y 都能命中
			q = q.Where("city_id = ? OR x = ? OR y = ?", n, n, n)
		} else {
			var cids []uint
			h.DB.Model(&model.EzfyCity{}).Select("id").
				Where("name LIKE ?", "%"+word+"%").Scan(&cids)
			if len(cids) > 0 {
				q = q.Where("city_id IN ?", cids)
			} else {
				q = q.Where("1 = 0")
			}
		}
	}
	var total int64
	q.Count(&total)
	var rows []model.EzfyWildland
	q.Order("id DESC").Offset(offset).Limit(size).Find(&rows)

	// ★★ 2026-10-05 性能（用户反馈「地图管理 tab 查询超时」）：
	//   原来 ezfyWildlandRow **每行**单独查「城市 + 玩家档案 + 玩家账号 + 野地配置 + 守将名」
	//   = 4~5 条 SQL，一页 20 行就是 80~100 条跨 WAN 往返 → 必然超时。
	//   现在按本页数据**批量取一次**：城市 1 条 + 档案 1 条 + 账号 1 条 + 野地配置 1 条，
	//   守将名走配置缓存 —— 一页固定 4 条 SQL，与页大小无关。
	cityIDs := make([]uint, 0, len(rows))
	for _, w := range rows {
		if w.CityId > 0 {
			cityIDs = append(cityIDs, uint(w.CityId))
		}
	}
	cityMap := map[uint]model.EzfyCity{}
	if len(cityIDs) > 0 {
		var cts []model.EzfyCity
		h.DB.Where("id IN ?", cityIDs).Find(&cts)
		for _, ct := range cts {
			cityMap[ct.ID] = ct
		}
	}
	uids := make([]uint, 0, len(cityMap))
	for _, ct := range cityMap {
		uids = append(uids, ct.UserID)
	}
	nickMap, numMap := h.ezfyAdminNamesBatch(uids)
	cfgMap := h.ezfyWildCfgMap()
	out := make([]gin.H, 0, len(rows))
	for _, w := range rows {
		out = append(out, h.ezfyWildlandRowBatch(w, cityMap, nickMap, numMap, cfgMap))
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// AdminEzfyWildlandBatchDelete POST /admin/ezfy-wildlands/batch-delete  {ids:[...]}
//
// ★ 2026-10-05 用户要求「管理端删除做好批量删除、没用的历史数据要做物理删除」。
//   ⚠️ 全站 ezfy 模型**都没有 gorm.DeletedAt**，所以 `Delete` 本来就是**物理删除**（真 DELETE 行），
//   不会留软删标记 —— 这里保持一致，批量删除也是物理删。
func (h *EzfyAdmin) AdminEzfyWildlandBatchDelete(c *gin.Context) {
	ids := ezfyBatchIDs(c)
	if len(ids) == 0 {
		resp.ParamError(c, "请先勾选要删除的记录")
		return
	}
	res := h.DB.Where("id IN ?", ids).Delete(&model.EzfyWildland{})
	if res.Error != nil {
		resp.ParamError(c, "批量删除失败："+res.Error.Error())
		return
	}
	resp.OK(c, gin.H{"msg": fmt.Sprintf("已物理删除 %d 条野地记录（勾选 %d 条）", res.RowsAffected, len(ids)),
		"deleted": res.RowsAffected})
}

// ezfyBatchIDs 从请求体里取批量操作的 id 列表（兼容 {ids:[1,2]} 与 {ids:"1,2"} 两种写法）。
func ezfyBatchIDs(c *gin.Context) []int64 {
	var in struct {
		IDs interface{} `json:"ids"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.IDs == nil {
		return nil
	}
	out := []int64{}
	switch v := in.IDs.(type) {
	case []interface{}:
		for _, it := range v {
			switch n := it.(type) {
			case float64:
				if n > 0 {
					out = append(out, int64(n))
				}
			case string:
				if id, e := strconv.ParseInt(strings.TrimSpace(n), 10, 64); e == nil && id > 0 {
					out = append(out, id)
				}
			}
		}
	case string:
		for _, part := range strings.Split(v, ",") {
			if id, e := strconv.ParseInt(strings.TrimSpace(part), 10, 64); e == nil && id > 0 {
				out = append(out, id)
			}
		}
	}
	return out
}

// ezfyAdminNamesBatch 批量取「玩家昵称 + 账号名」（原 ezfyAdminName 是每行 2 条 SQL 的 N+1）。
func (h *EzfyAdmin) ezfyAdminNamesBatch(uids []uint) (map[uint]string, map[uint]string) {
	nick := map[uint]string{}
	num := map[uint]string{}
	if len(uids) == 0 {
		return nick, num
	}
	var ps []model.EzfyProfile
	h.DB.Select("user_id, nickname").Where("user_id IN ?", uids).Find(&ps)
	for _, p := range ps {
		nick[p.UserID] = p.Nickname
	}
	var us []model.User
	h.DB.Select("id, username").Where("id IN ?", uids).Find(&us)
	for _, u := range us {
		num[u.ID] = u.Username
	}
	return nick, num
}

// ezfyWildCfgMap 一次取全部野地配置，按 "type:level" 建索引（原实现每行查一次）。
func (h *EzfyAdmin) ezfyWildCfgMap() map[string]model.EzfyCfgWildland {
	var cfgs []model.EzfyCfgWildland
	h.DB.Order("id").Find(&cfgs)
	m := make(map[string]model.EzfyCfgWildland, len(cfgs))
	for _, c := range cfgs {
		k := fmt.Sprintf("%d:%d", c.Type, c.Level)
		if _, ok := m[k]; !ok { // 与原来 Order("id").First 同口径：取 id 最小的那条
			m[k] = c
		}
	}
	return m
}

// ezfyWildlandRowBatch 同 ezfyWildlandRow，但城市/玩家名/配置全部由调用方批量传入（零额外 SQL）。
// 守将名走 ezfyCfg 配置缓存（原 ezfyGeneralName 每行一条 SQL）。
func (h *EzfyAdmin) ezfyWildlandRowBatch(w model.EzfyWildland,
	cityMap map[uint]model.EzfyCity, nickMap, numMap map[uint]string,
	cfgMap map[string]model.EzfyCfgWildland) gin.H {
	cityName, owner, home := "", "", ""
	if ct, ok := cityMap[uint(w.CityId)]; ok {
		cityName = ct.Name
		owner = nickMap[ct.UserID]
		home = numMap[ct.UserID]
	}
	cfg, hasCfg := cfgMap[fmt.Sprintf("%d:%d", w.WildType, w.Level)]
	row := gin.H{
		"id": w.ID, "city_id": w.CityId, "x": w.X, "y": w.Y,
		"wild_type": w.WildType, "type_name": ezfyWildTypeName(w.WildType),
		"level": w.Level, "gain": w.Gain, "status": w.Status,
		"status_name": ezfyWildStatusName(w.Status),
		"start_time":  w.StartTime, "end_time": w.EndTime,
		"created_at": w.CreatedAt, "updated_at": w.UpdatedAt,
		"city_name": cityName, "owner_name": owner, "home_num": home,
		"terrain": ezfyTerrain(w.X, w.Y),
		// ★ 2026-10-05：野地记录行 ⇒ 确定有野地，海里那块叫「海底森林」（岛屿仍是「岛屿」）
		"terrain_name": ezfyWildTerrainDisplayName(w.X, w.Y, true),
		"has_cfg":      hasCfg,
	}
	if hasCfg {
		row["cfg_res_min"] = cfg.ResMin
		row["cfg_res_max"] = cfg.ResMax
		row["cfg_officer_min"] = cfg.OfficerMin
		row["cfg_officer_max"] = cfg.OfficerMax
		row["cfg_officer_id"] = cfg.OfficerId
		row["cfg_officer_name"] = h.ezfyGeneralNameCached(cfg.OfficerId)
		row["cfg_treasure"] = cfg.Treasure
	}
	return row
}

// ezfyGeneralNameCached 同 ezfyGeneralName，但走进程内配置缓存（0 条 SQL）。
func (h *EzfyAdmin) ezfyGeneralNameCached(id int) string {
	if id <= 0 {
		return ""
	}
	if g := ezfyCfg.general(id); g != nil {
		return g.Name
	}
	return ""
}

// ezfyWildlandRow 组装一行野地展示数据（城池/玩家/地形/类型名/状态名 + 该等级配置）
func (h *EzfyAdmin) ezfyWildlandRow(w model.EzfyWildland) gin.H {
	cityName, owner, home := "", "", ""
	if w.CityId > 0 {
		var ct model.EzfyCity
		if err := h.DB.First(&ct, w.CityId).Error; err == nil {
			cityName = ct.Name
			owner, home = h.ezfyAdminName(ct.UserID)
		}
	}
	// 该野地等级对应的配置（守军/产出区间），方便管理端判断合理性
	var cfg model.EzfyCfgWildland
	hasCfg := h.DB.Where("type = ? AND level = ?", w.WildType, w.Level).
		Order("id").First(&cfg).Error == nil
	row := gin.H{
		"id": w.ID, "city_id": w.CityId, "x": w.X, "y": w.Y,
		"wild_type": w.WildType, "type_name": ezfyWildTypeName(w.WildType),
		"level": w.Level, "gain": w.Gain, "status": w.Status,
		"status_name": ezfyWildStatusName(w.Status),
		"start_time":  w.StartTime, "end_time": w.EndTime,
		"created_at": w.CreatedAt, "updated_at": w.UpdatedAt,
		"city_name": cityName, "owner_name": owner, "home_num": home,
		"terrain": ezfyTerrain(w.X, w.Y),
		// ★ 2026-10-05：野地记录行 ⇒ 确定有野地，海里那块叫「海底森林」（岛屿仍是「岛屿」）
		"terrain_name": ezfyWildTerrainDisplayName(w.X, w.Y, true),
		"has_cfg":      hasCfg,
	}
	if hasCfg {
		row["cfg_res_min"] = cfg.ResMin
		row["cfg_res_max"] = cfg.ResMax
		row["cfg_officer_min"] = cfg.OfficerMin
		row["cfg_officer_max"] = cfg.OfficerMax
		row["cfg_officer_id"] = cfg.OfficerId
		row["cfg_officer_name"] = h.ezfyGeneralName(cfg.OfficerId)
		row["cfg_treasure"] = cfg.Treasure
	}
	return row
}

// AdminEzfyWildlandCreate 新增玩家野地
func (h *EzfyAdmin) AdminEzfyWildlandCreate(c *gin.Context) {
	var in map[string]interface{}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	vals := xyPickVals(in, ezfyWildFields)
	if _, ok := vals["city_id"]; !ok {
		resp.ParamError(c, "请填写归属城池")
		return
	}
	if _, ok := vals["wild_type"]; !ok {
		vals["wild_type"] = 1
	}
	if _, ok := vals["level"]; !ok {
		vals["level"] = 1
	}
	if err := h.DB.Model(&model.EzfyWildland{}).Create(vals).Error; err != nil {
		resp.ParamError(c, "新增失败："+err.Error())
		return
	}
	resp.OK(c, gin.H{"msg": "野地已新增"})
}

// AdminEzfyWildlandUpdate 编辑玩家野地（坐标/类型/等级/产出/状态）
func (h *EzfyAdmin) AdminEzfyWildlandUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var w model.EzfyWildland
	if err := h.DB.First(&w, id).Error; err != nil {
		resp.NotFound(c, "野地不存在")
		return
	}
	var in map[string]interface{}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	vals := xyPickVals(in, ezfyWildFields)
	if len(vals) == 0 {
		resp.ParamError(c, "无可修改字段")
		return
	}
	if err := h.DB.Model(&model.EzfyWildland{}).Where("id = ?", id).Updates(vals).Error; err != nil {
		resp.ParamError(c, "修改失败："+err.Error())
		return
	}
	resp.OK(c, gin.H{"msg": "野地已保存"})
}

// AdminEzfyWildlandFinish 立即完成采集（把 end_time 拨到过去）
func (h *EzfyAdmin) AdminEzfyWildlandFinish(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var w model.EzfyWildland
	if err := h.DB.First(&w, id).Error; err != nil {
		resp.NotFound(c, "野地不存在")
		return
	}
	h.DB.Model(&model.EzfyWildland{}).Where("id = ?", id).
		Updates(map[string]interface{}{"end_time": time.Now().UnixMilli() - 1})
	resp.OK(c, gin.H{"msg": "已标记为采集完成（下次进入游戏结算）"})
}

// AdminEzfyWildlandDelete 删除玩家野地
func (h *EzfyAdmin) AdminEzfyWildlandDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.EzfyWildland{}, id)
	resp.OK(c, gin.H{"msg": "野地已删除"})
}

// ============ 3. 科技配置维护（ezfy_cfg_tech / ezfy_cfg_tech_level） ============

func ezfyTechTypeName(t int) string {
	switch t {
	case 1:
		return "生产"
	case 2:
		return "军事"
	case 3:
		return "辅助"
	}
	return "其他"
}

// AdminEzfyTechCfgList 科技配置列表（含等级配置条数）
func (h *EzfyAdmin) AdminEzfyTechCfgList(c *gin.Context) {
	page, offset, size := pageOf(c, 20)
	word := strings.TrimSpace(c.Query("word"))
	q := h.DB.Model(&model.EzfyCfgTech{})
	if word != "" {
		if id, err := strconv.Atoi(word); err == nil {
			q = q.Where("id = ?", id)
		} else {
			q = q.Where("name LIKE ?", "%"+word+"%")
		}
	}
	var total int64
	q.Count(&total)
	var rows []model.EzfyCfgTech
	q.Order("id").Offset(offset).Limit(size).Find(&rows)

	// 每项科技的等级配置条数 + 已研究玩家数
	type agg struct {
		TechId int
		Cnt    int64
	}
	var lvAgg []agg
	h.DB.Model(&model.EzfyCfgTechLevel{}).Select("tech_id, COUNT(*) as cnt").
		Group("tech_id").Scan(&lvAgg)
	lvOf := map[int]int64{}
	for _, a := range lvAgg {
		lvOf[a.TechId] = a.Cnt
	}
	var ownAgg []agg
	// ★ 2026-09-28 科技等级用户级共用：按玩家数统计
	h.DB.Model(&model.EzfyUserTech{}).Select("tech_id, COUNT(*) as cnt").
		Group("tech_id").Scan(&ownAgg)
	ownOf := map[int]int64{}
	for _, a := range ownAgg {
		ownOf[a.TechId] = a.Cnt
	}

	out := make([]gin.H, 0, len(rows))
	for _, t := range rows {
		out = append(out, gin.H{
			"id": t.ID, "name": t.Name, "type": t.Type, "type_name": ezfyTechTypeName(t.Type),
			"max_level": t.MaxLevel, "pre_building": t.PreBuilding,
			"pre_tech": t.PreTech, "pre_tech_level": t.PreTechLevel,
			"effect": t.Effect, "des": t.Des,
			"level_count": lvOf[t.ID], "owned_count": ownOf[t.ID],
		})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// AdminEzfyTechCfgCreate 新增科技
func (h *EzfyAdmin) AdminEzfyTechCfgCreate(c *gin.Context) {
	var in map[string]interface{}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	vals := xyPickVals(in, ezfyTechCfgFields)
	if vals["name"] == nil {
		resp.ParamError(c, "请填写科技名称")
		return
	}
	if _, ok := vals["max_level"]; !ok {
		vals["max_level"] = 10
	}
	if err := h.DB.Model(&model.EzfyCfgTech{}).Create(vals).Error; err != nil {
		resp.ParamError(c, "新增失败："+err.Error())
		return
	}
	h.ezfyReload()
	resp.OK(c, gin.H{"msg": "科技已新增"})
}

// AdminEzfyTechCfgUpdate 修改科技
func (h *EzfyAdmin) AdminEzfyTechCfgUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var t model.EzfyCfgTech
	if err := h.DB.First(&t, id).Error; err != nil {
		resp.NotFound(c, "科技不存在")
		return
	}
	var in map[string]interface{}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	vals := xyPickVals(in, ezfyTechCfgFields)
	if len(vals) == 0 {
		resp.ParamError(c, "无可修改字段")
		return
	}
	if err := h.DB.Model(&model.EzfyCfgTech{}).Where("id = ?", id).Updates(vals).Error; err != nil {
		resp.ParamError(c, "修改失败："+err.Error())
		return
	}
	h.ezfyReload()
	resp.OK(c, gin.H{"msg": "科技「" + t.Name + "」已保存"})
}

// AdminEzfyTechCfgDelete 删除科技（连带其等级配置）
func (h *EzfyAdmin) AdminEzfyTechCfgDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var t model.EzfyCfgTech
	if err := h.DB.First(&t, id).Error; err != nil {
		resp.NotFound(c, "科技不存在")
		return
	}
	var owned int64
	// ★ 2026-09-28 科技等级用户级共用：按玩家数统计
	h.DB.Model(&model.EzfyUserTech{}).Where("tech_id = ?", id).Count(&owned)
	if owned > 0 {
		resp.ParamError(c, "该科技已被 "+strconv.FormatInt(owned, 10)+" 位玩家掌握，不能删除")
		return
	}
	h.DB.Delete(&model.EzfyCfgTechLevel{}, "tech_id = ?", id)
	h.DB.Delete(&model.EzfyCfgTech{}, id)
	h.ezfyReload()
	resp.OK(c, gin.H{"msg": "科技已删除"})
}

// AdminEzfyTechLevelList 某科技的等级配置
func (h *EzfyAdmin) AdminEzfyTechLevelList(c *gin.Context) {
	techId := atoiOr(c.Query("tech_id"), 0)
	q := h.DB.Model(&model.EzfyCfgTechLevel{})
	if techId > 0 {
		q = q.Where("tech_id = ?", techId)
	}
	var rows []model.EzfyCfgTechLevel
	q.Order("tech_id, level").Find(&rows)
	resp.OK(c, gin.H{"list": rows, "total": len(rows)})
}

// AdminEzfyTechLevelCreate 新增科技等级配置
func (h *EzfyAdmin) AdminEzfyTechLevelCreate(c *gin.Context) {
	var in map[string]interface{}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	vals := xyPickVals(in, ezfyTechLevelFields)
	if _, ok := vals["tech_id"]; !ok {
		resp.ParamError(c, "请选择科技")
		return
	}
	if _, ok := vals["level"]; !ok {
		resp.ParamError(c, "请填写等级")
		return
	}
	if err := h.DB.Model(&model.EzfyCfgTechLevel{}).Create(vals).Error; err != nil {
		resp.ParamError(c, "新增失败："+err.Error())
		return
	}
	h.ezfyReload()
	resp.OK(c, gin.H{"msg": "科技等级配置已新增"})
}

// AdminEzfyTechLevelUpdate 修改科技等级配置
func (h *EzfyAdmin) AdminEzfyTechLevelUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var lv model.EzfyCfgTechLevel
	if err := h.DB.First(&lv, id).Error; err != nil {
		resp.NotFound(c, "等级配置不存在")
		return
	}
	var in map[string]interface{}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	vals := xyPickVals(in, ezfyTechLevelFields)
	if len(vals) == 0 {
		resp.ParamError(c, "无可修改字段")
		return
	}
	if err := h.DB.Model(&model.EzfyCfgTechLevel{}).Where("id = ?", id).Updates(vals).Error; err != nil {
		resp.ParamError(c, "修改失败："+err.Error())
		return
	}
	h.ezfyReload()
	resp.OK(c, gin.H{"msg": "等级配置已保存"})
}

// AdminEzfyTechLevelDelete 删除科技等级配置
func (h *EzfyAdmin) AdminEzfyTechLevelDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.EzfyCfgTechLevel{}, id)
	h.ezfyReload()
	resp.OK(c, gin.H{"msg": "等级配置已删除"})
}

// ============ 4. 资源名称维护（ezfy_cfg_resource） ============

// AdminEzfyResCfgList 资源名称列表
func (h *EzfyAdmin) AdminEzfyResCfgList(c *gin.Context) {
	var rows []model.EzfyCfgResource
	h.DB.Order("sort, id").Find(&rows)
	resp.OK(c, gin.H{"list": rows, "total": len(rows),
		"usage": "改名后游戏端与管理端展示全部跟随；Key 是程序内部标识，不要改"})
}

// AdminEzfyResCfgUpdate 修改资源名称（改名后全站生效）
func (h *EzfyAdmin) AdminEzfyResCfgUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var r model.EzfyCfgResource
	if err := h.DB.First(&r, id).Error; err != nil {
		resp.NotFound(c, "资源配置不存在")
		return
	}
	var in map[string]interface{}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	vals := xyPickVals(in, ezfyResCfgFields)
	if len(vals) == 0 {
		resp.ParamError(c, "无可修改字段")
		return
	}
	if err := h.DB.Model(&model.EzfyCfgResource{}).Where("id = ?", id).Updates(vals).Error; err != nil {
		resp.ParamError(c, "修改失败："+err.Error())
		return
	}
	newName := r.Name
	if v, ok := vals["name"].(string); ok && v != "" {
		newName = v
	}
	h.ezfyH().cfgsReload() // 资源名进进程内缓存，改完即时生效
	resp.OK(c, gin.H{"msg": "「" + r.Name + "」已改名为「" + newName + "」"})
}

// AdminEzfyResCfgReset 一键恢复默认资源名
func (h *EzfyAdmin) AdminEzfyResCfgReset(c *gin.Context) {
	def := []model.EzfyCfgResource{
		{ID: 1, Key: "gold", Name: "黄金", Short: "金", Sort: 1},
		{ID: 2, Key: "food", Name: "粮食", Short: "粮", Sort: 2},
		{ID: 3, Key: "steel", Name: "钢铁", Short: "钢", Sort: 3},
		{ID: 4, Key: "oil", Name: "石油", Short: "油", Sort: 4},
		{ID: 5, Key: "rare", Name: "稀矿", Short: "稀", Sort: 5},
	}
	for _, d := range def {
		h.DB.Model(&model.EzfyCfgResource{}).Where("id = ?", d.ID).
			Updates(map[string]interface{}{"name": d.Name, "short": d.Short, "sort": d.Sort})
	}
	h.ezfyH().cfgsReload() // 恢复默认后刷新进程内缓存
	resp.OK(c, gin.H{"msg": "资源名称已恢复默认"})
}
