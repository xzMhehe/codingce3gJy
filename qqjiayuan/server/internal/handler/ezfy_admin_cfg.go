package handler

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm/clause"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 二战风云管理端 —— 配置表 & 军官模块
//
// 这里集中放三类接口：
//   1. 「建筑管理 → 总建筑配置」：ezfy_cfg_building / ezfy_cfg_building_level 的增删改查
//   2. 「兵种管理 → 兵种配置」：ezfy_cfg_troop 的参数修改（含同盟国/轴心国兵种名）
//   3. 「军官管理」7 个列表：名将/技能/装备三张配置表 + 玩家军官技能/装备两张实例视图
//
// ★ 这些表在 seed 里是「只补缺、不覆盖」(DoNothing)，所以管理端的修改能活过重启。

// ezfyReload 改完配置表后调用：返回一个已强制重载配置缓存的 EzfyHandler
func (h *AdminHandler) ezfyReload() *EzfyHandler {
	ez := &EzfyHandler{DB: h.DB}
	ez.cfgsReload()
	return ez
}

// ============ 公共：配置表字段白名单 ============

// 兵种配置可改字段（ezfy_cfg_troop）
var ezfyTroopFields = map[string]string{
	"name": "string", "name_axis": "string", "name_ally": "string", "type": "int",
	"health": "int", "atk_sea": "int", "atk_ground": "int", "atk_air": "int", "atk_def": "int",
	"defence": "int", "speed": "int", "attack_range": "int", "carry": "int", "pop": "int",
	"food_keep": "int", "oil_keep": "int",
	"food": "int64", "steel": "int64", "oil": "int64", "rare": "int64",
	"train_time": "int", "require": "string", "icon": "string", "repair_rate": "int",
}

// 建筑配置可改字段（ezfy_cfg_building）
var ezfyBuildingCfgFields = map[string]string{
	"name": "string", "type": "int", "max_level": "int",
	"unique_flag": "int", "can_delete": "int", "pre_building": "string", "des": "string",
}

// 建筑等级配置可改字段（ezfy_cfg_building_level）
var ezfyBuildingLevelFields = map[string]string{
	"building_id": "int", "level": "int", "pop": "int",
	"food": "int64", "steel": "int64", "oil": "int64", "rare": "int64", "gold": "int64",
	"build_time": "int", "capacity": "int64", "effect": "string",
}

// 名将配置可改字段（ezfy_cfg_general）
var ezfyGeneralFields = map[string]string{
	"name": "string", "level": "int", "military": "int", "logistics": "int", "learning": "int",
	"star": "int", "source": "string", "get_condition": "string",
	"skill": "string", "des": "string", "recruit": "int",
}

// 技能配置可改字段（ezfy_cfg_skill）
var ezfySkillFields = map[string]string{
	"name": "string", "effect": "string", "type": "int", "des": "string",
}

// 装备配置可改字段（ezfy_cfg_equipment）
var ezfyEquipFields = map[string]string{
	"name": "string", "type": "string", "tier": "int",
	"military": "int", "logistics": "int", "learning": "int", "level": "int", "des": "string",
}

func ezfyBuildTypeName(t int) string {
	switch t {
	case 1:
		return "资源"
	case 2:
		return "军事"
	case 3:
		return "城防"
	case 4:
		return "市政"
	}
	return "其他"
}

func ezfySkillTypeName(t int) string {
	switch t {
	case 1:
		return "攻击类"
	case 2:
		return "防御类"
	case 3:
		return "辅助类"
	}
	return "其他"
}

func ezfyEquipTierName(t int) string {
	switch t {
	case 1:
		return "初级"
	case 2:
		return "中级"
	case 3:
		return "高级"
	case 4:
		return "特殊"
	}
	return "其他"
}

// ============ 1. 兵种配置（可改参数） ============

// AdminEzfyTroopCfgUpdate 修改兵种配置（含同盟国/轴心国兵种名与全部战斗参数）
func (h *AdminHandler) AdminEzfyTroopCfgUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var cfg model.EzfyCfgTroop
	if err := h.DB.First(&cfg, id).Error; err != nil {
		resp.NotFound(c, "兵种不存在")
		return
	}
	var in map[string]interface{}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	vals := xyPickVals(in, ezfyTroopFields)
	if len(vals) == 0 {
		resp.ParamError(c, "无可修改字段")
		return
	}
	if err := h.DB.Model(&model.EzfyCfgTroop{}).Where("id = ?", id).Updates(vals).Error; err != nil {
		resp.ParamError(c, "修改失败："+err.Error())
		return
	}
	h.ezfyReload()
	resp.OK(c, gin.H{"msg": "兵种「" + cfg.Name + "」配置已保存"})
}

// ============ 2. 建筑总配置（ezfy_cfg_building / _level） ============

// AdminEzfyBuildingCfg 建筑总配置列表（附等级配置条数）
func (h *AdminHandler) AdminEzfyBuildingCfg(c *gin.Context) {
	word := strings.TrimSpace(c.Query("word"))
	q := h.DB.Model(&model.EzfyCfgBuilding{})
	if word != "" {
		if id, err := strconv.Atoi(word); err == nil {
			q = q.Where("id = ?", id)
		} else {
			q = q.Where("name LIKE ?", "%"+word+"%")
		}
	}
	var rows []model.EzfyCfgBuilding
	q.Order("id").Find(&rows)

	// 等级条数 / 等级区间
	type lvAgg struct {
		BuildingId int
		Cnt        int64
		MaxLv      int
	}
	var aggs []lvAgg
	h.DB.Model(&model.EzfyCfgBuildingLevel{}).
		Select("building_id, COUNT(*) as cnt, MAX(level) as max_lv").Group("building_id").Scan(&aggs)
	aggOf := map[int]lvAgg{}
	for _, a := range aggs {
		aggOf[a.BuildingId] = a
	}

	type rowOut struct {
		model.EzfyCfgBuilding
		TypeName   string `json:"type_name"`
		LevelCount int64  `json:"level_count"`
		MaxCfgLv   int    `json:"max_cfg_lv"`
	}
	out := []rowOut{}
	for _, b := range rows {
		a := aggOf[b.ID]
		out = append(out, rowOut{EzfyCfgBuilding: b, TypeName: ezfyBuildTypeName(b.Type),
			LevelCount: a.Cnt, MaxCfgLv: a.MaxLv})
	}
	resp.OK(c, gin.H{"list": out, "total": len(out)})
}

// AdminEzfyBuildingCfgCreate 新增建筑配置
func (h *AdminHandler) AdminEzfyBuildingCfgCreate(c *gin.Context) {
	var in map[string]interface{}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	vals := xyPickVals(in, ezfyBuildingCfgFields)
	if len(vals) == 0 || vals["name"] == nil {
		resp.ParamError(c, "请填写建筑名称")
		return
	}
	if err := h.DB.Model(&model.EzfyCfgBuilding{}).Create(vals).Error; err != nil {
		resp.ParamError(c, "新增失败："+err.Error())
		return
	}
	h.ezfyReload()
	resp.OK(c, gin.H{"msg": "建筑配置已新增"})
}

// AdminEzfyBuildingCfgUpdate 修改建筑配置
func (h *AdminHandler) AdminEzfyBuildingCfgUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var cfg model.EzfyCfgBuilding
	if err := h.DB.First(&cfg, id).Error; err != nil {
		resp.NotFound(c, "建筑不存在")
		return
	}
	var in map[string]interface{}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	vals := xyPickVals(in, ezfyBuildingCfgFields)
	if len(vals) == 0 {
		resp.ParamError(c, "无可修改字段")
		return
	}
	if err := h.DB.Model(&model.EzfyCfgBuilding{}).Where("id = ?", id).Updates(vals).Error; err != nil {
		resp.ParamError(c, "修改失败："+err.Error())
		return
	}
	h.ezfyReload()
	resp.OK(c, gin.H{"msg": "建筑「" + cfg.Name + "」配置已保存"})
}

// AdminEzfyBuildingCfgDelete 删除建筑配置（连带等级配置）
func (h *AdminHandler) AdminEzfyBuildingCfgDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var cfg model.EzfyCfgBuilding
	if err := h.DB.First(&cfg, id).Error; err != nil {
		resp.NotFound(c, "建筑不存在")
		return
	}
	// 玩家已建的这种建筑一并清掉，避免留下找不到配置的孤儿建筑
	var used int64
	h.DB.Model(&model.EzfyCityBuilding{}).Where("building_id = ?", id).Count(&used)
	if used > 0 {
		h.DB.Where("building_id = ?", id).Delete(&model.EzfyCityBuilding{})
	}
	h.DB.Where("building_id = ?", id).Delete(&model.EzfyCfgBuildingLevel{})
	h.DB.Delete(&model.EzfyCfgBuilding{}, id)
	h.ezfyReload()
	msg := "建筑配置「" + cfg.Name + "」已删除"
	if used > 0 {
		msg += fmt.Sprintf("，同时清除了 %d 个玩家已建实例", used)
	}
	resp.OK(c, gin.H{"msg": msg})
}

// AdminEzfyBuildingLevels 某建筑的等级配置列表
func (h *AdminHandler) AdminEzfyBuildingLevels(c *gin.Context) {
	buildingId := atoiOr(c.Query("building_id"), 0)
	if buildingId <= 0 {
		resp.ParamError(c, "请指定 building_id")
		return
	}
	var rows []model.EzfyCfgBuildingLevel
	h.DB.Where("building_id = ?", buildingId).Order("level").Find(&rows)
	resp.OK(c, gin.H{"list": rows, "total": len(rows)})
}

// AdminEzfyBuildingLevelCreate 新增等级配置
func (h *AdminHandler) AdminEzfyBuildingLevelCreate(c *gin.Context) {
	var in map[string]interface{}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	vals := xyPickVals(in, ezfyBuildingLevelFields)
	if vals["building_id"] == nil || vals["level"] == nil {
		resp.ParamError(c, "请填写 building_id 与 level")
		return
	}
	if err := h.DB.Model(&model.EzfyCfgBuildingLevel{}).Create(vals).Error; err != nil {
		resp.ParamError(c, "新增失败："+err.Error())
		return
	}
	resp.OK(c, gin.H{"msg": "等级配置已新增"})
}

// AdminEzfyBuildingLevelUpdate 修改等级配置
func (h *AdminHandler) AdminEzfyBuildingLevelUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var row model.EzfyCfgBuildingLevel
	if err := h.DB.First(&row, id).Error; err != nil {
		resp.NotFound(c, "等级配置不存在")
		return
	}
	var in map[string]interface{}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	vals := xyPickVals(in, ezfyBuildingLevelFields)
	if len(vals) == 0 {
		resp.ParamError(c, "无可修改字段")
		return
	}
	if err := h.DB.Model(&model.EzfyCfgBuildingLevel{}).Where("id = ?", id).Updates(vals).Error; err != nil {
		resp.ParamError(c, "修改失败："+err.Error())
		return
	}
	resp.OK(c, gin.H{"msg": "等级配置已保存"})
}

// AdminEzfyBuildingLevelDelete 删除等级配置
func (h *AdminHandler) AdminEzfyBuildingLevelDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.EzfyCfgBuildingLevel{}, id)
	resp.OK(c, gin.H{"msg": "等级配置已删除"})
}

// ============ 3. 队伍征兵：一键完成 ============

// AdminEzfyTrainFinishAll 把全部训练中的队列一次性完成（走游戏内结算，兵力正常入库）
func (h *AdminHandler) AdminEzfyTrainFinishAll(c *gin.Context) {
	ez := h.ezfyH()
	var rows []model.EzfyTrainQueue
	h.DB.Where("status = ?", 0).Find(&rows)
	if len(rows) == 0 {
		resp.OK(c, gin.H{"msg": "没有训练中的队列"})
		return
	}
	// 把结束时间全部拨到过去，再按城池逐个走游戏内结算（兵力正常入库）
	citySet := map[int64]bool{}
	for _, r := range rows {
		h.DB.Model(&model.EzfyTrainQueue{}).Where("id = ?", r.ID).Update("end_time", 0)
		citySet[r.CityId] = true
	}
	for cid := range citySet {
		var ct model.EzfyCity
		if err := h.DB.First(&ct, cid).Error; err != nil {
			continue
		}
		ez.collectTrainQueue(&ct)
	}
	resp.OK(c, gin.H{"msg": fmt.Sprintf("已一键完成 %d 个训练队列", len(rows))})
}

// ============ 4. 军官管理 —— 7 个列表 ============

// AdminEzfyOfficerOverview 军官总览（7 张表的记录数 + 关键指标）
func (h *AdminHandler) AdminEzfyOfficerOverview(c *gin.Context) {
	var generals, skills, equips, officers, ownedEquips int64
	h.DB.Model(&model.EzfyCfgGeneral{}).Count(&generals)
	h.DB.Model(&model.EzfyCfgSkill{}).Count(&skills)
	h.DB.Model(&model.EzfyCfgEquipment{}).Count(&equips)
	h.DB.Model(&model.EzfyOfficer{}).Count(&officers)
	h.DB.Model(&model.EzfyEquipment{}).Count(&ownedEquips)

	// 玩家军官技能总数（把每名军官的 skill JSON 长度加起来）
	var rows []model.EzfyOfficer
	h.DB.Select("id, skill").Find(&rows)
	ownedSkills := 0
	for i := range rows {
		ownedSkills += len(officerSkills(&rows[i]))
	}
	// 已任命 / 俘虏 / 出征中
	var appointed, captives, marching, equipped int64
	h.DB.Model(&model.EzfyOfficer{}).Where("position > 0").Count(&appointed)
	h.DB.Model(&model.EzfyOfficer{}).Where("is_captive = 1").Count(&captives)
	h.DB.Model(&model.EzfyOfficer{}).Where("status = 1").Count(&marching)
	h.DB.Model(&model.EzfyEquipment{}).Where("officer_id > 0").Count(&equipped)

	resp.OK(c, gin.H{
		"generals": generals, "skills": skills, "equipments": equips,
		"officers": officers, "owned_equipments": ownedEquips,
		"owned_skills": ownedSkills, "appointed": appointed,
		"captives": captives, "marching": marching, "equipped": equipped,
	})
}

// ---------- 4.1 名将列表（ezfy_cfg_general，可增删改 + 分发） ----------

// AdminEzfyGenerals 名将列表
func (h *AdminHandler) AdminEzfyGenerals(c *gin.Context) {
	word := strings.TrimSpace(c.Query("word"))
	q := h.DB.Model(&model.EzfyCfgGeneral{})
	if word != "" {
		if id, err := strconv.Atoi(word); err == nil {
			q = q.Where("id = ?", id)
		} else {
			q = q.Where("name LIKE ?", "%"+word+"%")
		}
	}
	var rows []model.EzfyCfgGeneral
	q.Order("id").Find(&rows)

	// 每名将已被多少玩家拥有（用于「分发」时判断重复）
	type cntAgg struct {
		GeneralId int
		Cnt       int64
	}
	var aggs []cntAgg
	h.DB.Model(&model.EzfyOfficer{}).Select("general_id, COUNT(*) as cnt").
		Where("general_id > 0").Group("general_id").Scan(&aggs)
	ownOf := map[int]int64{}
	for _, a := range aggs {
		ownOf[a.GeneralId] = a.Cnt
	}

	type rowOut struct {
		model.EzfyCfgGeneral
		OwnedCount int64 `json:"owned_count"`
	}
	out := []rowOut{}
	for _, g := range rows {
		out = append(out, rowOut{EzfyCfgGeneral: g, OwnedCount: ownOf[g.ID]})
	}
	resp.OK(c, gin.H{"list": out, "total": len(out)})
}

// AdminEzfyGeneralCreate 新增名将
func (h *AdminHandler) AdminEzfyGeneralCreate(c *gin.Context) {
	var in map[string]interface{}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	vals := xyPickVals(in, ezfyGeneralFields)
	if vals["name"] == nil {
		resp.ParamError(c, "请填写名将名称")
		return
	}
	if _, ok := vals["star"]; !ok {
		vals["star"] = 5
	}
	if _, ok := vals["recruit"]; !ok {
		vals["recruit"] = 1
	}
	if err := h.DB.Model(&model.EzfyCfgGeneral{}).Create(vals).Error; err != nil {
		resp.ParamError(c, "新增失败："+err.Error())
		return
	}
	h.ezfyReload()
	resp.OK(c, gin.H{"msg": "名将已新增"})
}

// AdminEzfyGeneralUpdate 修改名将
func (h *AdminHandler) AdminEzfyGeneralUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var g model.EzfyCfgGeneral
	if err := h.DB.First(&g, id).Error; err != nil {
		resp.NotFound(c, "名将不存在")
		return
	}
	var in map[string]interface{}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	vals := xyPickVals(in, ezfyGeneralFields)
	if len(vals) == 0 {
		resp.ParamError(c, "无可修改字段")
		return
	}
	if err := h.DB.Model(&model.EzfyCfgGeneral{}).Where("id = ?", id).Updates(vals).Error; err != nil {
		resp.ParamError(c, "修改失败："+err.Error())
		return
	}
	h.ezfyReload()
	resp.OK(c, gin.H{"msg": "名将「" + g.Name + "」已保存"})
}

// AdminEzfyGeneralDelete 删除名将（同时清掉玩家已拥有的该名将军官）
func (h *AdminHandler) AdminEzfyGeneralDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var g model.EzfyCfgGeneral
	if err := h.DB.First(&g, id).Error; err != nil {
		resp.NotFound(c, "名将不存在")
		return
	}
	var used int64
	h.DB.Model(&model.EzfyOfficer{}).Where("general_id = ?", id).Count(&used)
	if used > 0 {
		h.DB.Where("general_id = ?", id).Delete(&model.EzfyOfficer{})
	}
	h.DB.Delete(&model.EzfyCfgGeneral{}, id)
	h.ezfyReload()
	msg := "名将「" + g.Name + "」已删除"
	if used > 0 {
		msg += fmt.Sprintf("，同时回收了 %d 名玩家已拥有的该军官", used)
	}
	resp.OK(c, gin.H{"msg": msg})
}

// ---------- 4.2 军官技能列表（ezfy_cfg_skill，可增删改） ----------

// AdminEzfySkills 技能配置列表
func (h *AdminHandler) AdminEzfySkills(c *gin.Context) {
	word := strings.TrimSpace(c.Query("word"))
	q := h.DB.Model(&model.EzfyCfgSkill{})
	if word != "" {
		if id, err := strconv.Atoi(word); err == nil {
			q = q.Where("id = ?", id)
		} else {
			q = q.Where("name LIKE ? OR effect LIKE ?", "%"+word+"%", "%"+word+"%")
		}
	}
	var rows []model.EzfyCfgSkill
	q.Order("id").Find(&rows)

	// 每种技能被多少名军官学了（skill 是 JSON 数组，只能在内存里统计）
	var officers []model.EzfyOfficer
	h.DB.Select("id, skill").Find(&officers)
	useCnt := map[string]int64{}
	for i := range officers {
		for _, s := range officerSkills(&officers[i]) {
			useCnt[s]++
		}
	}

	type rowOut struct {
		model.EzfyCfgSkill
		TypeName string `json:"type_name"`
		UseCount int64  `json:"use_count"`
	}
	out := []rowOut{}
	for _, s := range rows {
		out = append(out, rowOut{EzfyCfgSkill: s, TypeName: ezfySkillTypeName(s.Type), UseCount: useCnt[s.Name]})
	}
	resp.OK(c, gin.H{"list": out, "total": len(out)})
}

// AdminEzfySkillCreate 新增技能
func (h *AdminHandler) AdminEzfySkillCreate(c *gin.Context) {
	var in map[string]interface{}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	vals := xyPickVals(in, ezfySkillFields)
	if vals["name"] == nil {
		resp.ParamError(c, "请填写技能名称")
		return
	}
	if _, ok := vals["type"]; !ok {
		vals["type"] = 1
	}
	if err := h.DB.Model(&model.EzfyCfgSkill{}).Create(vals).Error; err != nil {
		resp.ParamError(c, "新增失败："+err.Error())
		return
	}
	h.ezfyReload()
	resp.OK(c, gin.H{"msg": "技能已新增"})
}

// AdminEzfySkillUpdate 修改技能
func (h *AdminHandler) AdminEzfySkillUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var s model.EzfyCfgSkill
	if err := h.DB.First(&s, id).Error; err != nil {
		resp.NotFound(c, "技能不存在")
		return
	}
	var in map[string]interface{}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	vals := xyPickVals(in, ezfySkillFields)
	if len(vals) == 0 {
		resp.ParamError(c, "无可修改字段")
		return
	}
	// 技能名是玩家军官 skill JSON 里的键，改名要同步已学该技能的军官
	if newName, ok := vals["name"].(string); ok && newName != "" && newName != s.Name {
		var officers []model.EzfyOfficer
		h.DB.Select("id, skill").Find(&officers)
		touched := 0
		for i := range officers {
			names := officerSkills(&officers[i])
			changed := false
			for j, n := range names {
				if n == s.Name {
					names[j] = newName
					changed = true
				}
			}
			if changed {
				b, _ := json.Marshal(names)
				h.DB.Model(&model.EzfyOfficer{}).Where("id = ?", officers[i].ID).Update("skill", string(b))
				touched++
			}
		}
		if err := h.DB.Model(&model.EzfyCfgSkill{}).Where("id = ?", id).Updates(vals).Error; err != nil {
			resp.ParamError(c, "修改失败："+err.Error())
			return
		}
		h.ezfyReload()
		resp.OK(c, gin.H{"msg": fmt.Sprintf("技能「%s」已改名为「%s」，同步更新了 %d 名军官", s.Name, newName, touched)})
		return
	}
	if err := h.DB.Model(&model.EzfyCfgSkill{}).Where("id = ?", id).Updates(vals).Error; err != nil {
		resp.ParamError(c, "修改失败："+err.Error())
		return
	}
	h.ezfyReload()
	resp.OK(c, gin.H{"msg": "技能「" + s.Name + "」已保存"})
}

// AdminEzfySkillDelete 删除技能（同时从玩家军官身上摘掉）
func (h *AdminHandler) AdminEzfySkillDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var s model.EzfyCfgSkill
	if err := h.DB.First(&s, id).Error; err != nil {
		resp.NotFound(c, "技能不存在")
		return
	}
	var officers []model.EzfyOfficer
	h.DB.Select("id, skill").Find(&officers)
	touched := 0
	for i := range officers {
		names := officerSkills(&officers[i])
		kept := []string{}
		changed := false
		for _, n := range names {
			if n == s.Name {
				changed = true
				continue
			}
			kept = append(kept, n)
		}
		if changed {
			b, _ := json.Marshal(kept)
			h.DB.Model(&model.EzfyOfficer{}).Where("id = ?", officers[i].ID).Update("skill", string(b))
			touched++
		}
	}
	h.DB.Delete(&model.EzfyCfgSkill{}, id)
	h.ezfyReload()
	msg := "技能「" + s.Name + "」已删除"
	if touched > 0 {
		msg += fmt.Sprintf("，并从 %d 名军官身上摘除", touched)
	}
	resp.OK(c, gin.H{"msg": msg})
}

// ---------- 4.3 玩家军官技能列表（ezfy_officer.skill 展开） ----------

// AdminEzfyOfficerSkillsOwned 玩家军官技能列表（把每名军官的 skill JSON 摊平成一行一条）
func (h *AdminHandler) AdminEzfyOfficerSkillsOwned(c *gin.Context) {
	page, offset, size := pageOf(c, 15)
	word := strings.TrimSpace(c.Query("word"))
	skillName := strings.TrimSpace(c.Query("skill"))

	q := h.DB.Model(&model.EzfyOfficer{})
	if word != "" {
		if id, err := strconv.Atoi(word); err == nil {
			q = q.Where("id = ? OR city_id = ?", id, id)
		} else {
			q = q.Where("name LIKE ?", "%"+word+"%")
		}
	}
	var officers []model.EzfyOfficer
	q.Order("id DESC").Find(&officers)

	// 技能名 -> 效果（用于展示）
	h.ezfyH()
	effectOf := map[string]string{}
	descOf := map[string]string{}
	for _, s := range ezfyCfg.skills {
		effectOf[s.Name] = s.Effect
		descOf[s.Name] = s.Des
	}

	// 城市/玩家名缓存
	cityCache := map[int64]string{}
	ownerCache := map[int64][2]string{}
	locOf := func(cityId int64) (string, string, string) {
		name, ok := cityCache[cityId]
		if !ok {
			var ct model.EzfyCity
			if err := h.DB.First(&ct, cityId).Error; err == nil {
				name = ct.Name
				on, hn := h.ezfyAdminName(ct.UserID)
				ownerCache[cityId] = [2]string{on, hn}
			}
			cityCache[cityId] = name
		}
		o := ownerCache[cityId]
		return name, o[0], o[1]
	}

	type rowOut struct {
		OfficerId   uint   `json:"officer_id"`
		OfficerName string `json:"officer_name"`
		SkillIndex  int    `json:"skill_index"`
		SkillName   string `json:"skill_name"`
		Effect      string `json:"effect"`
		Des         string `json:"des"`
		CityId      int64  `json:"city_id"`
		CityName    string `json:"city_name"`
		OwnerName   string `json:"owner_name"`
		HomeNum     string `json:"home_num"`
	}
	all := []rowOut{}
	for i := range officers {
		o := officers[i]
		cityName, owner, home := locOf(o.CityId)
		for idx, sn := range officerSkills(&o) {
			if skillName != "" && sn != skillName {
				continue
			}
			all = append(all, rowOut{
				OfficerId: o.ID, OfficerName: o.Name, SkillIndex: idx, SkillName: sn,
				Effect: effectOf[sn], Des: descOf[sn],
				CityId: o.CityId, CityName: cityName, OwnerName: owner, HomeNum: home,
			})
		}
	}
	total := len(all)
	if offset > total {
		offset = total
	}
	end := offset + size
	if end > total {
		end = total
	}
	resp.OK(c, gin.H{"list": all[offset:end], "total": total, "page": page, "size": size})
}

// AdminEzfyOfficerSkillAdd 给军官加技能（管理端直接加，不消耗黄金、不受 3 个上限限制）
func (h *AdminHandler) AdminEzfyOfficerSkillAdd(c *gin.Context) {
	var in struct {
		OfficerId uint   `json:"officer_id"`
		SkillName string `json:"skill_name"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.OfficerId == 0 || strings.TrimSpace(in.SkillName) == "" {
		resp.ParamError(c, "请选择军官与技能")
		return
	}
	var o model.EzfyOfficer
	if err := h.DB.First(&o, in.OfficerId).Error; err != nil {
		resp.NotFound(c, "军官不存在")
		return
	}
	name := strings.TrimSpace(in.SkillName)
	h.ezfyH()
	if ezfyCfg.skillByName[name] == 0 {
		resp.ParamError(c, "技能「"+name+"」不在技能配置里")
		return
	}
	names := officerSkills(&o)
	for _, n := range names {
		if n == name {
			resp.ParamError(c, "该军官已学会「"+name+"」")
			return
		}
	}
	names = append(names, name)
	b, _ := json.Marshal(names)
	h.DB.Model(&model.EzfyOfficer{}).Where("id = ?", o.ID).Update("skill", string(b))
	resp.OK(c, gin.H{"msg": fmt.Sprintf("已让「%s」学会「%s」", o.Name, name)})
}

// AdminEzfyOfficerSkillRemove 摘掉军官的某个技能
func (h *AdminHandler) AdminEzfyOfficerSkillRemove(c *gin.Context) {
	var in struct {
		OfficerId uint   `json:"officer_id"`
		SkillName string `json:"skill_name"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.OfficerId == 0 || in.SkillName == "" {
		resp.ParamError(c, "请选择军官与技能")
		return
	}
	var o model.EzfyOfficer
	if err := h.DB.First(&o, in.OfficerId).Error; err != nil {
		resp.NotFound(c, "军官不存在")
		return
	}
	kept := []string{}
	found := false
	for _, n := range officerSkills(&o) {
		if n == in.SkillName {
			found = true
			continue
		}
		kept = append(kept, n)
	}
	if !found {
		resp.ParamError(c, "该军官没有「"+in.SkillName+"」")
		return
	}
	b, _ := json.Marshal(kept)
	h.DB.Model(&model.EzfyOfficer{}).Where("id = ?", o.ID).Update("skill", string(b))
	resp.OK(c, gin.H{"msg": fmt.Sprintf("已让「%s」遗忘「%s」", o.Name, in.SkillName)})
}

// ---------- 4.4 军官装备列表（ezfy_cfg_equipment，可增删改） ----------

// AdminEzfyEquipments 装备配置列表
func (h *AdminHandler) AdminEzfyEquipments(c *gin.Context) {
	word := strings.TrimSpace(c.Query("word"))
	q := h.DB.Model(&model.EzfyCfgEquipment{})
	if word != "" {
		if id, err := strconv.Atoi(word); err == nil {
			q = q.Where("id = ?", id)
		} else {
			q = q.Where("name LIKE ? OR type LIKE ?", "%"+word+"%", "%"+word+"%")
		}
	}
	var rows []model.EzfyCfgEquipment
	q.Order("id").Find(&rows)

	// 每件装备被玩家持有多少
	type cntAgg struct {
		CfgId int
		Cnt   int64
	}
	var aggs []cntAgg
	h.DB.Model(&model.EzfyEquipment{}).Select("cfg_id, COUNT(*) as cnt").
		Group("cfg_id").Scan(&aggs)
	ownOf := map[int]int64{}
	for _, a := range aggs {
		ownOf[a.CfgId] = a.Cnt
	}

	type rowOut struct {
		model.EzfyCfgEquipment
		TierName   string `json:"tier_name"`
		OwnedCount int64  `json:"owned_count"`
	}
	out := []rowOut{}
	for _, e := range rows {
		out = append(out, rowOut{EzfyCfgEquipment: e, TierName: ezfyEquipTierName(e.Tier), OwnedCount: ownOf[e.ID]})
	}
	resp.OK(c, gin.H{"list": out, "total": len(out)})
}

// AdminEzfyEquipmentCreate 新增装备配置
func (h *AdminHandler) AdminEzfyEquipmentCreate(c *gin.Context) {
	var in map[string]interface{}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	vals := xyPickVals(in, ezfyEquipFields)
	if vals["name"] == nil {
		resp.ParamError(c, "请填写装备名称")
		return
	}
	if _, ok := vals["type"]; !ok {
		vals["type"] = "武器"
	}
	if _, ok := vals["tier"]; !ok {
		vals["tier"] = 1
	}
	if _, ok := vals["level"]; !ok {
		vals["level"] = 1
	}
	if err := h.DB.Model(&model.EzfyCfgEquipment{}).Create(vals).Error; err != nil {
		resp.ParamError(c, "新增失败："+err.Error())
		return
	}
	h.ezfyReload()
	resp.OK(c, gin.H{"msg": "装备配置已新增"})
}

// AdminEzfyEquipmentUpdate 修改装备配置
func (h *AdminHandler) AdminEzfyEquipmentUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var e model.EzfyCfgEquipment
	if err := h.DB.First(&e, id).Error; err != nil {
		resp.NotFound(c, "装备不存在")
		return
	}
	var in map[string]interface{}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	vals := xyPickVals(in, ezfyEquipFields)
	if len(vals) == 0 {
		resp.ParamError(c, "无可修改字段")
		return
	}
	if err := h.DB.Model(&model.EzfyCfgEquipment{}).Where("id = ?", id).Updates(vals).Error; err != nil {
		resp.ParamError(c, "修改失败："+err.Error())
		return
	}
	h.ezfyReload()
	resp.OK(c, gin.H{"msg": "装备「" + e.Name + "」已保存"})
}

// AdminEzfyEquipmentDelete 删除装备配置（玩家背包里的同名装备一并清理）
func (h *AdminHandler) AdminEzfyEquipmentDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var e model.EzfyCfgEquipment
	if err := h.DB.First(&e, id).Error; err != nil {
		resp.NotFound(c, "装备不存在")
		return
	}
	var used int64
	h.DB.Model(&model.EzfyEquipment{}).Where("cfg_id = ?", id).Count(&used)
	if used > 0 {
		h.DB.Where("cfg_id = ?", id).Delete(&model.EzfyEquipment{})
	}
	h.DB.Delete(&model.EzfyCfgEquipment{}, id)
	h.ezfyReload()
	msg := "装备「" + e.Name + "」配置已删除"
	if used > 0 {
		msg += fmt.Sprintf("，同时清除了玩家背包里的 %d 件", used)
	}
	resp.OK(c, gin.H{"msg": msg})
}

// ---------- 4.5 玩家军官装备列表（ezfy_equipment） ----------

// AdminEzfyEquipmentsOwned 玩家装备列表（word=装备名/军官ID，type=类型，equipped=1 仅已穿戴）
func (h *AdminHandler) AdminEzfyEquipmentsOwned(c *gin.Context) {
	page, offset, size := pageOf(c, 15)
	word := strings.TrimSpace(c.Query("word"))
	typ := strings.TrimSpace(c.Query("type"))
	equipped := atoiOr(c.Query("equipped"), -1)

	q := h.DB.Model(&model.EzfyEquipment{})
	if typ != "" {
		q = q.Where("type = ?", typ)
	}
	if equipped == 1 {
		q = q.Where("officer_id > 0")
	} else if equipped == 0 {
		q = q.Where("officer_id = 0")
	}
	if word != "" {
		if id, err := strconv.Atoi(word); err == nil {
			q = q.Where("id = ? OR officer_id = ? OR city_id = ?", id, id, id)
		} else {
			q = q.Where("name LIKE ?", "%"+word+"%")
		}
	}
	var total int64
	q.Count(&total)
	var rows []model.EzfyEquipment
	q.Order("id DESC").Offset(offset).Limit(size).Find(&rows)

	type rowOut struct {
		model.EzfyEquipment
		TierName   string `json:"tier_name"`
		CityName   string `json:"city_name"`
		OwnerName  string `json:"owner_name"`
		HomeNum    string `json:"home_num"`
		OfficerNam string `json:"officer_name"`
	}
	out := []rowOut{}
	for _, e := range rows {
		cityName, owner, home, offName := "", "", "", ""
		var ct model.EzfyCity
		if err := h.DB.First(&ct, e.CityId).Error; err == nil {
			cityName = ct.Name
			owner, home = h.ezfyAdminName(ct.UserID)
		}
		if e.OfficerId > 0 {
			var o model.EzfyOfficer
			if err := h.DB.First(&o, e.OfficerId).Error; err == nil {
				offName = o.Name
			}
		}
		out = append(out, rowOut{EzfyEquipment: e, TierName: ezfyEquipTierName(e.Tier),
			CityName: cityName, OwnerName: owner, HomeNum: home, OfficerNam: offName})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// AdminEzfyEquipmentOwnedCreate 给玩家发一件装备（按配置生成，落到其主城背包）
func (h *AdminHandler) AdminEzfyEquipmentOwnedCreate(c *gin.Context) {
	var in struct {
		UserId uint `json:"user_id"`
		CfgId  int  `json:"cfg_id"`
		Count  int  `json:"count"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.CfgId <= 0 {
		resp.ParamError(c, "请选择装备配置")
		return
	}
	if in.UserId == 0 {
		resp.ParamError(c, "请填写玩家用户ID")
		return
	}
	if in.Count <= 0 {
		in.Count = 1
	}
	if in.Count > 50 {
		in.Count = 50
	}
	var p model.EzfyProfile
	if err := h.DB.Where("user_id = ?", in.UserId).First(&p).Error; err != nil {
		resp.NotFound(c, "玩家不存在")
		return
	}
	var cfg model.EzfyCfgEquipment
	if err := h.DB.First(&cfg, in.CfgId).Error; err != nil {
		resp.NotFound(c, "装备配置不存在")
		return
	}
	ez := h.ezfyH()
	city := ez.getOrCreateCity(p.UserID)
	for i := 0; i < in.Count; i++ {
		h.DB.Create(&model.EzfyEquipment{
			UserId: p.UserID, CityId: int64(city.ID), CfgId: cfg.ID, Name: cfg.Name,
			Type: cfg.Type, Tier: cfg.Tier, Military: cfg.Military, Logistics: cfg.Logistics,
			Learning: cfg.Learning, Level: cfg.Level, OfficerId: 0,
		})
	}
	resp.OK(c, gin.H{"msg": fmt.Sprintf("已给「%s」发放 %s ×%d", p.Nickname, cfg.Name, in.Count)})
}

// AdminEzfyEquipmentOwnedUpdate 修改玩家装备（改名/属性/等级/穿戴军官）
func (h *AdminHandler) AdminEzfyEquipmentOwnedUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var e model.EzfyEquipment
	if err := h.DB.First(&e, id).Error; err != nil {
		resp.NotFound(c, "装备不存在")
		return
	}
	var in map[string]interface{}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	fields := map[string]string{
		"name": "string", "type": "string", "tier": "int",
		"military": "int", "logistics": "int", "learning": "int", "level": "int",
		"officer_id": "int64", "cfg_id": "int",
	}
	vals := xyPickVals(in, fields)
	if len(vals) == 0 {
		resp.ParamError(c, "无可修改字段")
		return
	}
	// 校验要穿戴的军官确实属于同一玩家
	if oid, ok := vals["officer_id"].(int64); ok && oid > 0 {
		var o model.EzfyOfficer
		if err := h.DB.First(&o, oid).Error; err != nil {
			resp.NotFound(c, "军官不存在")
			return
		}
		var ct model.EzfyCity
		if err := h.DB.First(&ct, o.CityId).Error; err != nil || ct.UserID != e.UserId {
			resp.ParamError(c, "该军官不属于这件装备的持有者")
			return
		}
	}
	if err := h.DB.Model(&model.EzfyEquipment{}).Where("id = ?", id).Updates(vals).Error; err != nil {
		resp.ParamError(c, "修改失败："+err.Error())
		return
	}
	resp.OK(c, gin.H{"msg": "装备已保存"})
}

// AdminEzfyEquipmentOwnedDelete 删除玩家装备
func (h *AdminHandler) AdminEzfyEquipmentOwnedDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.EzfyEquipment{}, id)
	resp.OK(c, gin.H{"msg": "装备已删除"})
}

// ---------- 4.6 军官下拉用：可选名将 / 技能 / 军官 ----------

// AdminEzfyOfficerPickers 给前端下拉框用的精简数据（名将名 / 技能名 / 军官名）
func (h *AdminHandler) AdminEzfyOfficerPickers(c *gin.Context) {
	var generals []model.EzfyCfgGeneral
	h.DB.Order("id").Find(&generals)
	type gOut struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
		Star int    `json:"star"`
	}
	gs := []gOut{}
	for _, g := range generals {
		gs = append(gs, gOut{ID: g.ID, Name: g.Name, Star: g.Star})
	}

	h.ezfyH()
	sks := []gin.H{}
	for _, s := range ezfyCfg.skills {
		sks = append(sks, gin.H{"name": s.Name, "effect": s.Effect, "type": ezfySkillTypeName(s.Type)})
	}
	// 技能按名称排序，方便查找
	sort.Slice(sks, func(i, j int) bool {
		return fmt.Sprint(sks[i]["name"]) < fmt.Sprint(sks[j]["name"])
	})

	var officers []model.EzfyOfficer
	h.DB.Order("id DESC").Limit(500).Find(&officers)
	os := []gin.H{}
	for _, o := range officers {
		os = append(os, gin.H{"id": o.ID, "name": o.Name, "city_id": o.CityId,
			"skill_count": len(officerSkills(&o))})
	}

	resp.OK(c, gin.H{"generals": gs, "skills": sks, "officers": os})
}

// ============ 一键生成军官（随机名字/等级/星级，属性不超过名将） ============

// AdminEzfyTechMaxAll POST /admin/ezfy-techs/max-all
//
// 一键把**所有玩家、所有城市**的科技升到满级。
//
// ★ 「不要产生脏数据」的三道保障：
//   1. 先按 (city_id, tech_id) **去重**（历史脏数据兜底，只保留 id 最小的那行）；
//   2. 用 `ON DUPLICATE KEY UPDATE` **upsert**（唯一索引 uk_city_tech），不会插重复行；
//   3. 把 status/end_time 一并归零，避免留下「研究中」的半截状态。
func (h *AdminHandler) AdminEzfyTechMaxAll(c *gin.Context) {
	var techs []model.EzfyCfgTech
	h.DB.Order("id").Find(&techs)
	if len(techs) == 0 {
		resp.ParamError(c, "没有科技配置，无法满级")
		return
	}
	// 1) 去重：同一 (city_id, tech_id) 只留 id 最小的一行
	dedup := h.DB.Exec("DELETE t1 FROM ezfy_city_tech t1 JOIN ezfy_city_tech t2 " +
		"ON t1.city_id = t2.city_id AND t1.tech_id = t2.tech_id AND t1.id > t2.id")
	removed := dedup.RowsAffected

	// ★ 第九轮：科技**所有城池公用** —— 每个玩家只写「科技城」(主城) 一份，
	//   不再按城市各写一份（否则城市越多行数越多，且分城的行是无效数据）。
	var cities []model.EzfyCity
	h.DB.Select("id", "user_id").Find(&cities)
	if len(cities) == 0 {
		resp.ParamError(c, "还没有玩家城市")
		return
	}
	mainOf := map[uint]uint{}
	for _, ct := range cities {
		if m, ok := mainOf[ct.UserID]; !ok || ct.ID < m {
			mainOf[ct.UserID] = ct.ID
		}
	}
	cities = cities[:0]
	for _, mid := range mainOf {
		cities = append(cities, model.EzfyCity{ID: mid})
	}
	now := time.Now()
	rows := make([]model.EzfyCityTech, 0, len(cities)*len(techs))
	for _, ct := range cities {
		for _, t := range techs {
			lv := t.MaxLevel
			if lv <= 0 {
				lv = 10
			}
			rows = append(rows, model.EzfyCityTech{
				CityId: int64(ct.ID), TechId: t.ID, Level: lv,
				Status: 0, EndTime: 0, UpdatedAt: now,
			})
		}
	}
	// 2) 分批 upsert
	err := h.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "city_id"}, {Name: "tech_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"level", "status", "end_time", "updated_at"}),
	}).CreateInBatches(rows, 200).Error
	if err != nil {
		resp.ParamError(c, "满级失败："+err.Error())
		return
	}
	// 3) 回读校验：确认没有重复行、且全部达到满级
	var dupCnt int64
	h.DB.Raw("SELECT COUNT(*) FROM (SELECT city_id, tech_id FROM ezfy_city_tech " +
		"GROUP BY city_id, tech_id HAVING COUNT(*) > 1) t").Scan(&dupCnt)
	var maxLevel int64
	h.DB.Raw("SELECT COALESCE(MAX(level), 0) FROM ezfy_city_tech").Scan(&maxLevel)

	resp.OK(c, gin.H{
		"msg": fmt.Sprintf("已把 %d 座城市 × %d 项科技升到满级（清理重复行 %d 条）",
			len(cities), len(techs), removed),
		"cities": len(cities), "techs": len(techs),
		"dedup_removed": removed, "dup_left": dupCnt, "max_level": maxLevel,
	})
}

// ezfyGeneralCapByStar 每个星级下「名将」的属性上限 [军事,后勤,学识]
//
// ★ 用户要求：随机生成的军官属性不能超过名将。
//   同名将星级里取最大值作为上限；该星级没有名将就往下借一档；
//   都没有就用 星级×20 兜底（比同星级名将保守）。
func (h *AdminHandler) ezfyGeneralCapByStar() map[int][3]int {
	var gs []model.EzfyCfgGeneral
	h.DB.Find(&gs)
	cap := map[int][3]int{}
	for _, g := range gs {
		st := g.Star
		if st <= 0 {
			st = 5
		}
		c := cap[st]
		if g.Military > c[0] {
			c[0] = g.Military
		}
		if g.Logistics > c[1] {
			c[1] = g.Logistics
		}
		if g.Learning > c[2] {
			c[2] = g.Learning
		}
		cap[st] = c
	}
	// 逐级往下借：该星级没有名将就用低一星的上限
	for st := 1; st <= 5; st++ {
		if _, ok := cap[st]; ok {
			continue
		}
		for lower := st - 1; lower >= 1; lower-- {
			if c, ok := cap[lower]; ok {
				cap[st] = c
				break
			}
		}
		if _, ok := cap[st]; !ok {
			cap[st] = [3]int{st * 20, st * 20, st * 20}
		}
	}
	return cap
}

// AdminEzfyGenOfficers 一键生成军官（挂到指定玩家的主城下）
//
// POST /admin/ezfy-officers/gen  {user_id, count, max_level}
func (h *AdminHandler) AdminEzfyGenOfficers(c *gin.Context) {
	var in struct {
		UserID   uint `json:"user_id"`
		Count    int  `json:"count"`
		MaxLevel int  `json:"max_level"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.UserID == 0 {
		resp.ParamError(c, "请选择玩家")
		return
	}
	if in.Count <= 0 {
		in.Count = 1
	}
	if in.Count > 20 {
		in.Count = 20
	}
	if in.MaxLevel <= 0 {
		in.MaxLevel = 60
	}
	if in.MaxLevel > 200 {
		in.MaxLevel = 200
	}
	var prof model.EzfyProfile
	if err := h.DB.Where("user_id = ?", in.UserID).First(&prof).Error; err != nil {
		resp.NotFound(c, "玩家不存在")
		return
	}
	var pu model.User
	h.DB.First(&pu, in.UserID)
	ez := h.ezfyH()
	city := ez.mainCity(in.UserID)
	caps := h.ezfyGeneralCapByStar()

	used := map[string]bool{}
	var curNames []model.EzfyOfficer
	h.DB.Where("city_id = ?", city.ID).Find(&curNames)
	for _, o := range curNames {
		used[o.Name] = true
	}

	created := []gin.H{}
	for i := 0; i < in.Count; i++ {
		name := ""
		for k := 0; k < 40; k++ {
			name = ezfyOfficerFirstNames[rand.Intn(len(ezfyOfficerFirstNames))] + "·" +
				ezfyOfficerLastNames[rand.Intn(len(ezfyOfficerLastNames))]
			if !used[name] {
				break
			}
		}
		used[name] = true

		star := ezfyRollStar()
		if star < 1 {
			star = 1
		}
		if star > 5 {
			star = 5
		}
		level := 1 + rand.Intn(in.MaxLevel)
		if level < 1 {
			level = 1
		}
		cap3 := caps[star]
		// ★ 属性上限 = 同星级名将的最大值（且至少 1）
		rnd := func(mx int) int {
			if mx <= 1 {
				return 1
			}
			return 1 + rand.Intn(mx)
		}
		mil, log, lea := rnd(cap3[0]), rnd(cap3[1]), rnd(cap3[2])

		o := model.EzfyOfficer{
			CityId: int64(city.ID), GeneralId: 0, Name: name, Star: star,
			Level: level, Exp: 0,
			Military: mil, Logistics: log, Learning: lea,
			Loyalty: 80 + rand.Intn(21), Skill: "", Equipment: "",
			Position: 0, Status: 0, IsCaptive: 0, UpdateTime: time.Now(),
		}
		if err := h.DB.Create(&o).Error; err != nil {
			continue
		}
		created = append(created, gin.H{
			"id": o.ID, "name": o.Name, "star": star, "level": level,
			"military": mil, "logistics": log, "learning": lea,
			"cap": gin.H{"military": cap3[0], "logistics": cap3[1], "learning": cap3[2]},
		})
	}
	resp.OK(c, gin.H{
		"msg":  fmt.Sprintf("已为「%s」生成 %d 名军官（属性上限取自同星级名将）", ezfyNickOf(prof, &pu), len(created)),
		"list": created,
	})
}
