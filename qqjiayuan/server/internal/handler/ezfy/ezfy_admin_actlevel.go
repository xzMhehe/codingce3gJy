package ezfy

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm/clause"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 二战风云管理端 —— 活动目标「等级配置」
//
// ★ 2026-10-22 用户需求：
//
//	活动野地 / 活动寇城 / 特殊城市 三类活动目标，位置可以先不维护，
//	但每个等级的「兵力配置、资源奖励、必掉宝物、掉落道具」必须可维护。
//
// ★ 2026-10-24 与活动野地同款：每个等级还可配 守将 / 守军等级 / 守军行为 / 被俘，
//   坐标级「活动野地」配置（ezfy_act_wild）仍然优先于本表。
//
// 数据表 ezfy_act_level：(ActType, Level) 唯一。
// 运行时由 ezfy_act_level_cfg 读取（配置缓存），战斗结算/详情预览都走它；
// 坐标级「活动野地」配置（ezfy_act_wild）仍然优先于本表。
var ezfyActLevelFields = map[string]string{
	"act_type": "int", "level": "int", "troops": "string",
	"res": "int64", "gold": "int64", "prestige": "int",
	"treasures": "string", "drop_items": "string", "des": "string",
	// ★ 2026-10-24 与活动野地同款：守将 / 守军微调 / 被俘
	"officer_id": "int", "def_level": "int", "captive_level": "int",
	"def_move_dir": "string", "capture_rate": "int", "max_capture": "int",
}

// AdminEzfyActLevelList GET /admin/ezfy-act-levels —— 活动目标等级配置列表
func (h *EzfyAdmin) AdminEzfyActLevelList(c *gin.Context) {
	h.cfgs()
	page, offset, size := pageOf(c, 50)
	actType := atoiOr(c.Query("act_type"), 0)
	level := atoiOr(c.Query("level"), 0)
	q := h.DB.Model(&model.EzfyActLevel{})
	if actType > 0 {
		q = q.Where("act_type = ?", actType)
	}
	if level > 0 {
		q = q.Where("level = ?", level)
	}
	var total int64
	q.Count(&total)
	var rows []model.EzfyActLevel
	q.Order("act_type, level, id").Offset(offset).Limit(size).Find(&rows)

	out := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		oName := ""
		oStar := 0
		if g := ezfyCfg.general(r.OfficerId); g != nil {
			oName = g.Name
			oStar = g.Star
		}
		out = append(out, gin.H{
			"id": r.ID, "act_type": r.ActType, "act_type_name": ezfyActTargetName(r.ActType),
			"level": r.Level, "troops": r.Troops,
			"res": r.Res, "gold": r.Gold, "prestige": r.Prestige,
			"treasures": r.Treasures, "drop_items": r.DropItems, "des": r.Des,
			// ★ 2026-10-24 守将 / 守军微调 / 被俘（编辑弹窗回填 + 列表展示）
			"officer_id": r.OfficerId, "officer_name": oName, "officer_star": oStar,
			"capture_rate": r.CaptureRate, "max_capture": r.MaxCapture,
			"def_level": r.DefLevel, "captive_level": r.CaptiveLevel, "def_move_dir": r.DefMoveDir,
			"updated_at": r.UpdatedAt,
		})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// AdminEzfyActLevelSave POST /admin/ezfy-act-levels —— 新增/编辑等级配置
func (h *EzfyAdmin) AdminEzfyActLevelSave(c *gin.Context) {
	h.cfgs()
	var in map[string]interface{}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	vals := xyPickVals(in, ezfyActLevelFields)
	actType, _ := vals["act_type"].(int)
	level, _ := vals["level"].(int)
	if actType < ezfyActWild || actType > ezfyActCity {
		resp.ParamError(c, "请选择活动类型（活动野地/活动寇城/特殊城市）")
		return
	}
	if level < 1 || level > 3 {
		resp.ParamError(c, "活动等级只能是 1~3")
		return
	}
	if msg := checkActLevelVals(vals); msg != "" {
		resp.ParamError(c, msg)
		return
	}
	// 数值字段缺省补 0 / 空串，避免 NULL / Unknown column
	for _, k := range []string{"res", "gold", "prestige"} {
		if _, ok := vals[k]; !ok {
			vals[k] = 0
		}
	}
	for _, k := range []string{"troops", "treasures", "drop_items", "des"} {
		if s, _ := vals[k].(string); s == "" {
			vals[k] = ""
		}
	}
	// ★ 2026-10-24 守将 / 被俘默认值（与活动野地同款）：概率 0、次数 1、等级 0、行为空
	if _, ok := vals["officer_id"]; !ok {
		vals["officer_id"] = 0
	}
	if _, ok := vals["capture_rate"]; !ok {
		vals["capture_rate"] = 0
	}
	if _, ok := vals["max_capture"]; !ok {
		vals["max_capture"] = 1
	}
	for _, k := range []string{"def_level", "captive_level"} {
		if _, ok := vals[k]; !ok {
			vals[k] = 0
		}
	}
	if _, ok := vals["def_move_dir"]; !ok {
		vals["def_move_dir"] = ""
	}

	id := ezfyAnyInt(in["id"])
	if id > 0 {
		var cur model.EzfyActLevel
		if err := h.DB.First(&cur, id).Error; err != nil {
			resp.NotFound(c, "该活动目标配置不存在（可能已被删除），请刷新后重试")
			return
		}
		var dup int64
		h.DB.Model(&model.EzfyActLevel{}).
			Where("act_type = ? AND level = ? AND id <> ?", actType, level, id).Count(&dup)
		if dup > 0 {
			resp.ParamError(c, fmt.Sprintf("「%s %d级」已存在另一条配置，请直接编辑那一条",
				ezfyActTargetName(actType), level))
			return
		}
		if err := h.DB.Model(&model.EzfyActLevel{}).Where("id = ?", id).Updates(vals).Error; err != nil {
			resp.ParamError(c, "保存失败："+err.Error())
			return
		}
		h.ezfyReload()
		resp.OK(c, gin.H{"msg": fmt.Sprintf("「%s %d级」已保存并立即生效",
			ezfyActTargetName(actType), level)})
		return
	}

	if err := h.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "act_type"}, {Name: "level"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"troops", "res", "gold", "prestige", "treasures", "drop_items", "des",
			// ★ 2026-10-24 守将 / 守军微调 / 被俘
			"officer_id", "capture_rate", "max_capture",
			"def_level", "captive_level", "def_move_dir",
		}),
	}).Model(&model.EzfyActLevel{}).Create(vals).Error; err != nil {
		resp.ParamError(c, "保存失败："+err.Error())
		return
	}
	h.ezfyReload()
	resp.OK(c, gin.H{"msg": fmt.Sprintf("「%s %d级」已保存并立即生效",
		ezfyActTargetName(actType), level)})
}

// AdminEzfyActLevelDelete DELETE /admin/ezfy-act-levels/:id
func (h *EzfyAdmin) AdminEzfyActLevelDelete(c *gin.Context) {
	h.cfgs()
	id, _ := strconv.Atoi(c.Param("id"))
	var row model.EzfyActLevel
	if err := h.DB.First(&row, id).Error; err != nil {
		resp.NotFound(c, "该活动目标配置不存在")
		return
	}
	h.DB.Delete(&model.EzfyActLevel{}, id)
	h.ezfyReload()
	resp.OK(c, gin.H{"msg": fmt.Sprintf("「%s %d级」配置已删除，将回退内置默认",
		ezfyActTargetName(row.ActType), row.Level)})
}

// checkActLevelVals 校验活动目标等级配置
func checkActLevelVals(vals map[string]interface{}) string {
	if v, ok := vals["res"].(int64); ok && v < 0 {
		return "资源奖励不能为负"
	}
	if v, ok := vals["gold"].(int64); ok && v < 0 {
		return "黄金奖励不能为负"
	}
	if v, ok := vals["prestige"].(int); ok && v < 0 {
		return "声望奖励不能为负"
	}
	// 守军 JSON：【必须】校验兵种存在，避免配了不存在的兵种导致战斗直接空守军。
	if s, ok := vals["troops"].(string); ok && strings.TrimSpace(s) != "" {
		groups := parseActWildTroops(s)
		if groups == nil {
			return "守军配置格式不对，应为 [[兵种id,最小,最大],...]（最小/最大均为正整数）"
		}
		for _, g := range groups {
			if ezfyCfg.troop(g.TroopId) == nil {
				return fmt.Sprintf("守军兵种 ID %d 不存在，请重新选择", g.TroopId)
			}
		}
	}
	// 必掉宝物 JSON：[[装备cfg_id,数量],...]
	if s, ok := vals["treasures"].(string); ok && strings.TrimSpace(s) != "" {
		rows := parseActWildTreasures(s)
		if rows == nil {
			return "宝物配置格式不对，应为 [[宝物id,数量],...]"
		}
		for _, tr := range rows {
			e := ezfyCfg.equipment(tr[0])
			if e == nil {
				return fmt.Sprintf("宝物 ID %d 不存在，请重新选择", tr[0])
			}
			if e.SetId > 0 {
				return fmt.Sprintf("「%s」是套装件，只能开宝箱获得，不能配置为掉落", e.Name)
			}
		}
	}
	// 掉落道具 JSON：[[道具cfg_id,数量,概率%],...]
	if s, ok := vals["drop_items"].(string); ok && strings.TrimSpace(s) != "" {
		rows := parseWildlandItemDrops(s)
		if rows == nil {
			return "掉落道具格式不对，应为 [[道具id,数量,概率%],...]"
		}
		for _, d := range rows {
			if ezfyCfg.item(d[0]) == nil {
				return fmt.Sprintf("道具 ID %d 不存在，请重新选择", d[0])
			}
		}
	}
	// ★ 2026-10-24 与活动野地同款校验：守将 / 守军微调 / 被俘
	if v, ok := vals["capture_rate"].(int); ok && (v < 0 || v > 100) {
		return "被俘虏概率只能是 0~100（0 = 不俘虏，100 = 必俘虏）"
	}
	if v, ok := vals["max_capture"].(int); ok && v < 0 {
		return "可抓次数不能为负（0 = 不限，>=1 为上限）"
	}
	if s, ok := vals["def_move_dir"].(string); ok && strings.TrimSpace(s) != "" && strings.TrimSpace(s) != "{}" {
		m := map[int]int{}
		if err := json.Unmarshal([]byte(s), &m); err != nil {
			return "守军行为格式不对，应为 {\"兵种ID\":1前进/0停止/-1后退}"
		}
		for id, dir := range m {
			if dir != 0 && dir != 1 && dir != -1 {
				return fmt.Sprintf("兵种 %d 的守军行为只能是 1（前进）/ 0（停止）/ -1（后退）", id)
			}
		}
	}
	for _, k := range []string{"def_level", "captive_level"} {
		if v, ok := vals[k].(int); ok && v < 0 {
			return k + "不能为负（0 = 用军官池默认）"
		}
	}
	if v, ok := vals["officer_id"].(int); ok && v > 0 {
		if ezfyCfg.general(v) == nil {
			return "守将军官不在军官池中，请重新选择"
		}
	}
	return ""
}
