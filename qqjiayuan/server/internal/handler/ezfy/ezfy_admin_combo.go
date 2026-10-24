package ezfy

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// ============ 管理端：名将组合技能池（2026-10-23） ============
//
// 「组合技能」是系统专属战法：普通军官学不到，只能由管理端把池子里的技能
// 配给名将（ezfy_cfg_general.combo_skill_ids，JSON 数组，一个名将可配多个）。这里提供池子的增删改查，
// 名称/说明/数值/概率全部可改，改完调 h.ezfyReload() 让配置缓存立即生效。

// ezfyGeneralSkillFields 池子可改字段白名单（xyPickVals 用）
var ezfyGeneralSkillFields = map[string]string{
	"name": "string", "des": "string", "kind": "int",
	"base_value": "int", "grow_per_level": "int", "max_level": "int", "level_per_step": "int",
	"proc_pct": "int", "rounds": "int", "quality": "int",
	"enabled": "int", "sort_no": "int",
}

// AdminEzfyGeneralSkills GET /admin/ezfy-general-skills —— 组合技能池列表
func (h *EzfyAdmin) AdminEzfyGeneralSkills(c *gin.Context) {
	word := trimStr(c.Query("word"), 50)
	q := h.DB.Model(&model.EzfyCfgGeneralSkill{})
	if word != "" {
		if id, err := strconv.Atoi(word); err == nil {
			q = q.Where("id = ?", id)
		} else {
			q = q.Where("name LIKE ?", "%"+word+"%")
		}
	}
	var rows []model.EzfyCfgGeneralSkill
	q.Order("sort_no, id").Find(&rows)
	// ★ 2026-10-23 一并下发「哪些名将在用这个技能」——管理端一眼能看出搭配情况，
	//   也能从这里直接改配（见 AdminEzfyGeneralSkillAssign）。
	//   ★ 2026-10-24 多技能：名将可配多个，combo_skill_ids 是 JSON 数组 → 逐个累计
	var gs []model.EzfyCfgGeneral
	h.DB.Select("id, name, combo_skill_ids, combo_skill_id").Where("kind = 2").Find(&gs)
	users := map[int][]gin.H{}
	for _, g := range gs {
		for _, sid := range comboSkillIdsOfCfg(&g) {
			users[sid] = append(users[sid], gin.H{"id": g.ID, "name": g.Name})
		}
	}
	list := make([]gin.H, 0, len(rows))
	for _, s := range rows {
		gu := users[s.ID]
		if gu == nil {
			gu = []gin.H{}
		}
		list = append(list, gin.H{
			"generals":      gu,
			"general_count": len(gu),
			"id":            s.ID, "name": s.Name, "des": s.Des, "kind": s.Kind,
			"kind_name":      ezfyComboKindName(s.Kind),
			"base_value":     s.BaseValue,
			"grow":           s.GrowPerLevel,
			"max_level":      s.MaxLevel,
			"level_per_step": s.LevelPerStep,
			"proc_pct":       s.ProcPct,
			"rounds":         s.Rounds,
			"quality":        s.Quality,
			"enabled":        s.Enabled,
			"sort_no":        s.SortNo,
			"value_at_max":   ezfyComboValue(s.BaseValue, s.GrowPerLevel, s.MaxLevel, s.MaxLevel),
		})
	}
	resp.OK(c, gin.H{"list": list, "total": len(list)})
}

// AdminEzfyGeneralSkillCreate POST /admin/ezfy-general-skills
func (h *EzfyAdmin) AdminEzfyGeneralSkillCreate(c *gin.Context) {
	var in map[string]interface{}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	vals := xyPickVals(in, ezfyGeneralSkillFields)
	if vals["name"] == nil || vals["name"] == "" {
		resp.ParamError(c, "请填写技能名称")
		return
	}
	if _, ok := vals["kind"]; !ok {
		vals["kind"] = 1
	}
	if _, ok := vals["enabled"]; !ok {
		vals["enabled"] = 1
	}
	if _, ok := vals["max_level"]; !ok {
		vals["max_level"] = 10
	}
	if _, ok := vals["level_per_step"]; !ok {
		vals["level_per_step"] = 35 // 名将满级 350 → 10 档吃满
	}
	if err := h.DB.Model(&model.EzfyCfgGeneralSkill{}).Create(vals).Error; err != nil {
		resp.ParamError(c, "新增失败："+err.Error())
		return
	}
	h.ezfyReload()
	resp.OK(c, gin.H{"msg": "组合技能已新增"})
}

// AdminEzfyGeneralSkillUpdate PUT /admin/ezfy-general-skills/:id
func (h *EzfyAdmin) AdminEzfyGeneralSkillUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var sc model.EzfyCfgGeneralSkill
	if err := h.DB.First(&sc, id).Error; err != nil {
		resp.NotFound(c, "组合技能不存在")
		return
	}
	var in map[string]interface{}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	vals := xyPickVals(in, ezfyGeneralSkillFields)
	if len(vals) == 0 {
		resp.ParamError(c, "无可修改字段")
		return
	}
	if err := h.DB.Model(&model.EzfyCfgGeneralSkill{}).Where("id = ?", id).Updates(vals).Error; err != nil {
		resp.ParamError(c, "修改失败："+err.Error())
		return
	}
	h.ezfyReload()
	resp.OK(c, gin.H{"msg": "组合技能「" + sc.Name + "」已保存"})
}

// AdminEzfyGeneralSkillDelete DELETE /admin/ezfy-general-skills/:id
//
// ★ 有名称在用它就拒绝删除（避免名将身上挂着已不存在的技能 → 战斗里静默失效），
// 想停用请把「启用」改成 0（下架后战斗不再发动，但配置保留）。
func (h *EzfyAdmin) AdminEzfyGeneralSkillDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var sc model.EzfyCfgGeneralSkill
	if err := h.DB.First(&sc, id).Error; err != nil {
		resp.NotFound(c, "组合技能不存在")
		return
	}
	var used int64
	// ★ 2026-10-24 多技能：combo_skill_ids 是 JSON 数组字符串，不能用 WHERE 等值，
	//   逐个名将解析后统计「在用的个数」。
	var gs []model.EzfyCfgGeneral
	h.DB.Select("id, combo_skill_ids, combo_skill_id").Where("kind = 2").Find(&gs)
	for _, g := range gs {
		for _, sid := range comboSkillIdsOfCfg(&g) {
			if sid == id {
				used++
				break
			}
		}
	}
	if used > 0 {
		resp.ParamError(c, "还有 "+strconv.FormatInt(used, 10)+" 名名将在用这个组合技能，请先改配或直接下架（启用=0）")
		return
	}
	h.DB.Delete(&model.EzfyCfgGeneralSkill{}, id)
	h.ezfyReload()
	resp.OK(c, gin.H{"msg": "组合技能「" + sc.Name + "」已删除"})
}

// AdminEzfyGeneralSkillAssign POST /admin/ezfy-general-skills/:id/assign
//
// 把组合技能**配给名将**（等价于在名将编辑里改 combo_skill_ids，但可以一次配多个）：
//
//	{ "general_ids": [1,18,9], "unbind": false }   → 这些名将**学会**该技能（追加到已有组合技，去重）
//	{ "general_ids": [], "unbind": true }          → 把所有会该技能的名将**解除**（从各自组合技列表移除）
//
// 只接受 kind=2 的名将；普通军官（kind=1）一律忽略（系统专属，军官学不到）。
// ★ 2026-10-24 多技能：学会 = 追加+去重（不再覆盖原组合技）；解除 = 只移除该技能，保留其余。
func (h *EzfyAdmin) AdminEzfyGeneralSkillAssign(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var sc model.EzfyCfgGeneralSkill
	if err := h.DB.First(&sc, id).Error; err != nil {
		resp.NotFound(c, "组合技能不存在")
		return
	}
	var in struct {
		GeneralIds []int `json:"general_ids"`
		Unbind     bool  `json:"unbind"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if in.Unbind {
		var gs []model.EzfyCfgGeneral
		h.DB.Select("id, combo_skill_ids, combo_skill_id").Where("kind = 2").Find(&gs)
		removed := 0
		for _, g := range gs {
			ids := comboSkillIdsOfCfg(&g)
			nid := comboSkillIdsRemove(ids, id)
			if len(nid) == len(ids) {
				continue // 本来就没配，跳过
			}
			removed++
			h.DB.Model(&model.EzfyCfgGeneral{}).Where("id = ?", g.ID).
				Updates(map[string]interface{}{"combo_skill_ids": comboSkillIdsTxt(nid), "combo_skill_id": 0})
		}
		h.ezfyReload()
		resp.OK(c, gin.H{"msg": "已解除 " + strconv.Itoa(removed) + " 名名将的「" + sc.Name + "」"})
		return
	}
	if len(in.GeneralIds) == 0 {
		resp.ParamError(c, "请选择要学习该组合技能的名将")
		return
	}
	// 只保留 kind=2 的名将，防止误把普通军官配进去
	var okIDs []int
	h.DB.Model(&model.EzfyCfgGeneral{}).
		Where("kind = 2 AND id IN ?", in.GeneralIds).Pluck("id", &okIDs)
	if len(okIDs) == 0 {
		resp.ParamError(c, "选中的都不是名将（普通军官学不到组合技能）")
		return
	}
	// ★ 2026-10-24 多技能：逐名将「追加 + 去重」写入 combo_skill_ids（JSON 数组），
	//   旧单技能字段 combo_skill_id 一并清 0（已迁移到新字段，不再当回退源）。
	var gs []model.EzfyCfgGeneral
	h.DB.Where("kind = 2 AND id IN ?", okIDs).Find(&gs)
	for _, g := range gs {
		nid := comboSkillIdsAdd(comboSkillIdsOfCfg(&g), id)
		if err := h.DB.Model(&model.EzfyCfgGeneral{}).Where("id = ?", g.ID).
			Updates(map[string]interface{}{"combo_skill_ids": comboSkillIdsTxt(nid), "combo_skill_id": 0}).Error; err != nil {
			resp.ParamError(c, "配给失败："+err.Error())
			return
		}
	}
	h.ezfyReload()
	msg := "已让 " + strconv.Itoa(len(gs)) + " 名名将学会「" + sc.Name + "」"
	if len(gs) < len(in.GeneralIds) {
		msg += "（另有 " + strconv.Itoa(len(in.GeneralIds)-len(gs)) + " 个不是名将，已忽略）"
	}
	resp.OK(c, gin.H{"msg": msg})
}

// AdminEzfyGeneralSkillPickers GET /admin/ezfy-general-skills/pickers
//
// 给「配给名将」弹窗用的名将清单（kind=2，含当前已配的组合技 ID 数组，前端可据此回显）。
func (h *EzfyAdmin) AdminEzfyGeneralSkillPickers(c *gin.Context) {
	var rows []model.EzfyCfgGeneral
	h.DB.Select("id, name, star, combo_skill_ids, combo_skill_id").Where("kind = 2").Order("id").Find(&rows)
	list := make([]gin.H, 0, len(rows))
	for _, g := range rows {
		list = append(list, gin.H{"id": g.ID, "name": g.Name, "star": g.Star,
			"combo_skills": comboSkillIdsOfCfg(&g)})
	}
	var skills []model.EzfyCfgGeneralSkill
	h.DB.Select("id, name, kind").Order("sort_no, id").Find(&skills)
	sl := make([]gin.H, 0, len(skills))
	for _, s := range skills {
		sl = append(sl, gin.H{"id": s.ID, "name": s.Name, "kind_name": ezfyComboKindName(s.Kind)})
	}
	resp.OK(c, gin.H{"generals": list, "skills": sl})
}
