package handler

import (
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
	"treasure": "string", "des": "string",
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
func (h *AdminHandler) AdminEzfyMapOptions(c *gin.Context) {
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
		gviews = append(gviews, gin.H{"id": g.ID, "name": g.Name, "star": g.Star,
			"military": g.Military, "logistics": g.Logistics, "learning": g.Learning})
	}
	resp.OK(c, gin.H{"troops": tviews, "generals": gviews})
}

// AdminEzfyWildCfgList 野地类型配置列表
func (h *AdminHandler) AdminEzfyWildCfgList(c *gin.Context) {
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
			"officer_id": r.OfficerId, "officer_name": h.ezfyGeneralName(r.OfficerId),
			"treasure": r.Treasure, "des": r.Des,
		})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// ezfyGeneralName 军官池（ezfy_cfg_general）里的军官名
func (h *AdminHandler) ezfyGeneralName(id int) string {
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
func (h *AdminHandler) ezfyCheckWildOfficer(vals map[string]interface{}) string {
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
func (h *AdminHandler) AdminEzfyWildCfgCreate(c *gin.Context) {
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
func (h *AdminHandler) AdminEzfyWildCfgUpdate(c *gin.Context) {
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
func (h *AdminHandler) AdminEzfyWildCfgDelete(c *gin.Context) {
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
func (h *AdminHandler) AdminEzfyWildlandList(c *gin.Context) {
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
	out := make([]gin.H, 0, len(rows))
	for _, w := range rows {
		out = append(out, h.ezfyWildlandRow(w))
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// ezfyWildlandRow 组装一行野地展示数据（城池/玩家/地形/类型名/状态名 + 该等级配置）
func (h *AdminHandler) ezfyWildlandRow(w model.EzfyWildland) gin.H {
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
		"terrain":      ezfyTerrain(w.X, w.Y),
		"terrain_name": ezfyTerrainNameEx(w.X, w.Y),
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
func (h *AdminHandler) AdminEzfyWildlandCreate(c *gin.Context) {
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
func (h *AdminHandler) AdminEzfyWildlandUpdate(c *gin.Context) {
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
func (h *AdminHandler) AdminEzfyWildlandFinish(c *gin.Context) {
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
func (h *AdminHandler) AdminEzfyWildlandDelete(c *gin.Context) {
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
func (h *AdminHandler) AdminEzfyTechCfgList(c *gin.Context) {
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
	h.DB.Model(&model.EzfyCityTech{}).Select("tech_id, COUNT(*) as cnt").
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
func (h *AdminHandler) AdminEzfyTechCfgCreate(c *gin.Context) {
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
func (h *AdminHandler) AdminEzfyTechCfgUpdate(c *gin.Context) {
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
func (h *AdminHandler) AdminEzfyTechCfgDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var t model.EzfyCfgTech
	if err := h.DB.First(&t, id).Error; err != nil {
		resp.NotFound(c, "科技不存在")
		return
	}
	var owned int64
	h.DB.Model(&model.EzfyCityTech{}).Where("tech_id = ?", id).Count(&owned)
	if owned > 0 {
		resp.ParamError(c, "该科技已被 "+strconv.FormatInt(owned, 10)+" 个玩家城池研究，不能删除")
		return
	}
	h.DB.Delete(&model.EzfyCfgTechLevel{}, "tech_id = ?", id)
	h.DB.Delete(&model.EzfyCfgTech{}, id)
	h.ezfyReload()
	resp.OK(c, gin.H{"msg": "科技已删除"})
}

// AdminEzfyTechLevelList 某科技的等级配置
func (h *AdminHandler) AdminEzfyTechLevelList(c *gin.Context) {
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
func (h *AdminHandler) AdminEzfyTechLevelCreate(c *gin.Context) {
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
func (h *AdminHandler) AdminEzfyTechLevelUpdate(c *gin.Context) {
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
func (h *AdminHandler) AdminEzfyTechLevelDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.EzfyCfgTechLevel{}, id)
	h.ezfyReload()
	resp.OK(c, gin.H{"msg": "等级配置已删除"})
}

// ============ 4. 资源名称维护（ezfy_cfg_resource） ============

// AdminEzfyResCfgList 资源名称列表
func (h *AdminHandler) AdminEzfyResCfgList(c *gin.Context) {
	var rows []model.EzfyCfgResource
	h.DB.Order("sort, id").Find(&rows)
	resp.OK(c, gin.H{"list": rows, "total": len(rows),
		"usage": "改名后游戏端与管理端展示全部跟随；Key 是程序内部标识，不要改"})
}

// AdminEzfyResCfgUpdate 修改资源名称（改名后全站生效）
func (h *AdminHandler) AdminEzfyResCfgUpdate(c *gin.Context) {
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
	resp.OK(c, gin.H{"msg": "「" + r.Name + "」已改名为「" + newName + "」"})
}

// AdminEzfyResCfgReset 一键恢复默认资源名
func (h *AdminHandler) AdminEzfyResCfgReset(c *gin.Context) {
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
	resp.OK(c, gin.H{"msg": "资源名称已恢复默认"})
}
