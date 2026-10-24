package ezfy

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 二战风云管理端 —— 战斗公式套维护
//
// ★ 2026-10-23 用户需求「战斗公式搞灵活一点，按套来，然后为当前服选择用哪套」：
//
//	一套公式 = 回合战斗引擎的全部可调参数（最大回合/初始距离/暴击加成/伤害系数…）。
//	管理端可维护多套（如「原版」「快节奏」「测试」），当前服在 ezfy_cfg_limit.battle_formula_id
//	里选一套生效；选 0 = 使用「战斗与经济」页里的内置公式。
//	读取端见 ezfy_battle.go 的 ezfyBattleFormulaActive()（整套覆盖，字段 0 回落硬编码默认）。
var ezfyBattleFormulaRules = []struct {
	label string
	ptr   func(*model.EzfyCfgBattleFormula) *int
	def   int
	min   int
	max   int
}{
	{"战斗最大回合数", func(f *model.EzfyCfgBattleFormula) *int { return &f.BattleMaxRounds }, ezfyBattleMaxRoundsDef, 1, 200},
	{"战场初始距离", func(f *model.EzfyCfgBattleFormula) *int { return &f.BattleStartDist }, ezfyBattleStartDistDef, 1000, 50000},
	{"战斗最大后退距离", func(f *model.EzfyCfgBattleFormula) *int { return &f.BattleRetreatMax }, ezfyBattleRetreatMaxDef, 1000, 100000},
	{"暴击基础伤害加成%", func(f *model.EzfyCfgBattleFormula) *int { return &f.CritBaseBonusPct }, ezfyCritBaseBonusPctDef, 1, 500},
	{"盟军驻军溃败阈值%", func(f *model.EzfyCfgBattleFormula) *int { return &f.GarrisonBreakPct }, ezfyGarrisonBreakPctDef, 1, 100},
	{"战斗每回合时长(毫秒)", func(f *model.EzfyCfgBattleFormula) *int { return &f.BattleRoundMs }, ezfyBattleRoundMsDef, 25000, 120000},
	{"伤害公式攻击系数", func(f *model.EzfyCfgBattleFormula) *int { return &f.AtkCoeff }, ezfyAtkCoeffDef, 1, 100},
	{"伤害公式攻击加值", func(f *model.EzfyCfgBattleFormula) *int { return &f.AtkBase }, ezfyAtkBaseDef, 1, 100000},
	{"伤害公式防御系数", func(f *model.EzfyCfgBattleFormula) *int { return &f.DefCoeff }, ezfyDefCoeffDef, 1, 100},
	{"伤害公式防御加值", func(f *model.EzfyCfgBattleFormula) *int { return &f.DefBase }, ezfyDefBaseDef, 1, 100000},
	{"最小伤害系数", func(f *model.EzfyCfgBattleFormula) *int { return &f.MinDmgCoeff }, ezfyMinDmgCoeffDef, 1, 1000},
}

type ezfyBattleFormulaIn struct {
	ID                    int    `json:"id"`
	Name                  string `json:"name"`
	Remark                string `json:"remark"`
	DamageFormula         *int   `json:"damage_formula"`
	ShowBattleBonusDetail *int   `json:"show_battle_bonus_detail"`
	BattleMaxRounds       *int   `json:"battle_max_rounds"`
	BattleStartDist       *int   `json:"battle_start_dist"`
	BattleRetreatMax      *int   `json:"battle_retreat_max"`
	CritBaseBonusPct      *int   `json:"crit_base_bonus_pct"`
	GarrisonBreakPct      *int   `json:"garrison_break_pct"`
	BattleRoundMs         *int   `json:"battle_round_ms"`
	AtkCoeff              *int   `json:"atk_coeff"`
	AtkBase               *int   `json:"atk_base"`
	DefCoeff              *int   `json:"def_coeff"`
	DefBase               *int   `json:"def_base"`
	MinDmgCoeff           *int   `json:"min_dmg_coeff"`
}

func (in *ezfyBattleFormulaIn) applyTo(f *model.EzfyCfgBattleFormula) {
	if in.DamageFormula != nil {
		f.DamageFormula = *in.DamageFormula
	}
	if in.ShowBattleBonusDetail != nil {
		f.ShowBattleBonusDetail = *in.ShowBattleBonusDetail
	}
	if in.BattleMaxRounds != nil {
		f.BattleMaxRounds = *in.BattleMaxRounds
	}
	if in.BattleStartDist != nil {
		f.BattleStartDist = *in.BattleStartDist
	}
	if in.BattleRetreatMax != nil {
		f.BattleRetreatMax = *in.BattleRetreatMax
	}
	if in.CritBaseBonusPct != nil {
		f.CritBaseBonusPct = *in.CritBaseBonusPct
	}
	if in.GarrisonBreakPct != nil {
		f.GarrisonBreakPct = *in.GarrisonBreakPct
	}
	if in.BattleRoundMs != nil {
		f.BattleRoundMs = *in.BattleRoundMs
	}
	if in.AtkCoeff != nil {
		f.AtkCoeff = *in.AtkCoeff
	}
	if in.AtkBase != nil {
		f.AtkBase = *in.AtkBase
	}
	if in.DefCoeff != nil {
		f.DefCoeff = *in.DefCoeff
	}
	if in.DefBase != nil {
		f.DefBase = *in.DefBase
	}
	if in.MinDmgCoeff != nil {
		f.MinDmgCoeff = *in.MinDmgCoeff
	}
}

func ezfyBattleFormulaJSON(f model.EzfyCfgBattleFormula) gin.H {
	return gin.H{
		"id": f.ID, "name": f.Name, "remark": f.Remark,
		"damage_formula":           f.DamageFormula,
		"show_battle_bonus_detail": f.ShowBattleBonusDetail,
		"battle_max_rounds":        f.BattleMaxRounds, "battle_start_dist": f.BattleStartDist,
		"battle_retreat_max": f.BattleRetreatMax, "crit_base_bonus_pct": f.CritBaseBonusPct,
		"garrison_break_pct": f.GarrisonBreakPct, "battle_round_ms": f.BattleRoundMs,
		"atk_coeff": f.AtkCoeff, "atk_base": f.AtkBase,
		"def_coeff": f.DefCoeff, "def_base": f.DefBase, "min_dmg_coeff": f.MinDmgCoeff,
	}
}

// checkEzfyBattleFormula 数值 0 视为未填（回落默认），其余按区间校验。
func checkEzfyBattleFormula(f *model.EzfyCfgBattleFormula) string {
	for _, r := range ezfyBattleFormulaRules {
		p := r.ptr(f)
		if *p == 0 {
			*p = r.def
		}
		if *p < r.min || *p > r.max {
			return fmt.Sprintf("%s需在 %d~%d 之间", r.label, r.min, r.max)
		}
	}
	f.Name = strings.TrimSpace(f.Name)
	if f.Name == "" {
		return "请填写公式套名称"
	}
	if utf8.RuneCountInString(f.Name) > 50 {
		return "公式套名称最多 50 个字"
	}
	if utf8.RuneCountInString(f.Remark) > 500 {
		return "说明最多 500 个字"
	}
	if f.DamageFormula != 0 && f.DamageFormula != 1 {
		return "伤害算法只能选「原版」或「家园版」"
	}
	return ""
}

// AdminEzfyBattleFormulaList GET /admin/ezfy-battle-formulas —— 公式套列表 + 当前生效 ID
func (h *EzfyAdmin) AdminEzfyBattleFormulaList(c *gin.Context) {
	h.cfgs()
	var rows []model.EzfyCfgBattleFormula
	h.DB.Order("id").Find(&rows)
	out := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		out = append(out, ezfyBattleFormulaJSON(r))
	}
	var lim model.EzfyCfgLimit
	if err := h.DB.First(&lim, 1).Error; err != nil {
		lim.BattleFormulaID = 0
	}
	resp.OK(c, gin.H{"list": out, "active_id": lim.BattleFormulaID})
}

// AdminEzfyBattleFormulaCreate POST /admin/ezfy-battle-formulas —— 新建公式套
func (h *EzfyAdmin) AdminEzfyBattleFormulaCreate(c *gin.Context) {
	h.cfgs()
	var in ezfyBattleFormulaIn
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	f := model.EzfyCfgBattleFormula{Name: in.Name, Remark: strings.TrimSpace(in.Remark)}
	in.applyTo(&f)
	if msg := checkEzfyBattleFormula(&f); msg != "" {
		resp.ParamError(c, msg)
		return
	}
	var dup int64
	h.DB.Model(&model.EzfyCfgBattleFormula{}).Where("name = ?", f.Name).Count(&dup)
	if dup > 0 {
		resp.ParamError(c, "已存在同名公式套，请换个名称")
		return
	}
	if err := h.DB.Create(&f).Error; err != nil {
		resp.ParamError(c, "保存失败："+err.Error())
		return
	}
	h.ezfyReload()
	resp.OK(c, gin.H{"msg": "公式套「" + f.Name + "」已创建", "id": f.ID})
}

// AdminEzfyBattleFormulaUpdate PUT /admin/ezfy-battle-formulas/:id —— 编辑公式套
func (h *EzfyAdmin) AdminEzfyBattleFormulaUpdate(c *gin.Context) {
	h.cfgs()
	id, _ := strconv.Atoi(c.Param("id"))
	var f model.EzfyCfgBattleFormula
	if err := h.DB.First(&f, id).Error; err != nil {
		resp.NotFound(c, "该公式套不存在（可能已被删除），请刷新后重试")
		return
	}
	var in ezfyBattleFormulaIn
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	f.Name, f.Remark = in.Name, strings.TrimSpace(in.Remark)
	in.applyTo(&f)
	if msg := checkEzfyBattleFormula(&f); msg != "" {
		resp.ParamError(c, msg)
		return
	}
	var dup int64
	h.DB.Model(&model.EzfyCfgBattleFormula{}).Where("name = ? AND id <> ?", f.Name, f.ID).Count(&dup)
	if dup > 0 {
		resp.ParamError(c, "已存在同名公式套，请换个名称")
		return
	}
	if err := h.DB.Save(&f).Error; err != nil {
		resp.ParamError(c, "保存失败："+err.Error())
		return
	}
	h.ezfyReload()
	resp.OK(c, gin.H{"msg": "公式套「" + f.Name + "」已保存"})
}

// AdminEzfyBattleFormulaDelete DELETE /admin/ezfy-battle-formulas/:id —— 删除公式套
func (h *EzfyAdmin) AdminEzfyBattleFormulaDelete(c *gin.Context) {
	h.cfgs()
	id, _ := strconv.Atoi(c.Param("id"))
	var f model.EzfyCfgBattleFormula
	if err := h.DB.First(&f, id).Error; err != nil {
		resp.NotFound(c, "该公式套不存在")
		return
	}
	var lim model.EzfyCfgLimit
	h.DB.First(&lim, 1)
	if lim.BattleFormulaID == f.ID {
		resp.ParamError(c, "「"+f.Name+"」正在本服生效，不能删除；请先切换到其他公式套")
		return
	}
	if err := h.DB.Delete(&model.EzfyCfgBattleFormula{}, f.ID).Error; err != nil {
		resp.ParamError(c, "删除失败："+err.Error())
		return
	}
	h.ezfyReload()
	resp.OK(c, gin.H{"msg": "公式套「" + f.Name + "」已删除"})
}

// AdminEzfyBattleFormulaApply POST /admin/ezfy-battle-formulas/:id/apply —— 设为当前服生效公式
//
// id = 0 表示切回「内置配置」（使用「战斗与经济」页里的公式数值）。
func (h *EzfyAdmin) AdminEzfyBattleFormulaApply(c *gin.Context) {
	h.cfgs()
	id, _ := strconv.Atoi(c.Param("id"))
	var name string
	if id < 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	if id > 0 {
		var f model.EzfyCfgBattleFormula
		if err := h.DB.First(&f, id).Error; err != nil {
			resp.NotFound(c, "该公式套不存在（可能已被删除），请刷新后重试")
			return
		}
		name = f.Name
	}
	// 防御老库缺列（skip 节点可能没跑过补列）：HasColumn 秒回。
	if !h.DB.Migrator().HasColumn("ezfy_cfg_limit", "battle_formula_id") {
		h.DB.Exec("ALTER TABLE ezfy_cfg_limit ADD COLUMN battle_formula_id int DEFAULT 0")
		h.DB.Exec("UPDATE ezfy_cfg_limit SET battle_formula_id = 0 WHERE battle_formula_id IS NULL")
	}
	if err := h.DB.Model(&model.EzfyCfgLimit{}).Where("id = 1").
		Update("battle_formula_id", id).Error; err != nil {
		resp.ParamError(c, "保存失败："+err.Error())
		return
	}
	var cnt int64
	h.DB.Model(&model.EzfyCfgLimit{}).Where("id = 1").Count(&cnt)
	if cnt == 0 {
		if err := h.DB.Create(&model.EzfyCfgLimit{ID: 1, BattleFormulaID: id}).Error; err != nil {
			resp.ParamError(c, "保存失败："+err.Error())
			return
		}
	}
	h.ezfyReload()
	if id == 0 {
		resp.OK(c, gin.H{"msg": "已切换为内置战斗公式（立即生效）"})
		return
	}
	resp.OK(c, gin.H{"msg": "本服已切换为战斗公式套「" + name + "」（立即生效）"})
}
