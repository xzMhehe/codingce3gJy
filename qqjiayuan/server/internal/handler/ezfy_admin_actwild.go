package handler

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm/clause"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 二战风云管理端 —— 活动野地配置（地图管理「活动野地」tab）
//
// ★ 2026-09-29 用户要求：活动野地配置不友好，优化成「按坐标列表管理」，
//   独立成 tab，且要能区别于普通野地（启用开关）+ 可配置活动（等级/守军/奖励）。
//
// 数据表 ezfy_act_wild：每个坐标一条。enabled=1 → 该格按活动野地玩法
//   （守军/奖励/等级用配置，缺省回退代码默认）；enabled=0 → 该格不是活动目标（普通野地）。
// 运行时判定走 ezfy_act_wild 优先 → mark 覆盖 → 坐标哈希（见 ezfyActTargetType）。

var ezfyActWildFields = map[string]string{
	"x": "int", "y": "int", "enabled": "int",
	"level": "int", "troops": "string", "res": "int64",
	"gold": "int64", "prestige": "int", "jewel": "string", "des": "string",
	"officer_id": "int", "treasures": "string", "capture_rate": "int", "max_capture": "int",
}

// AdminEzfyActWildList GET /admin/ezfy-act-wilds —— 活动野地配置列表（分页）
func (h *AdminHandler) AdminEzfyActWildList(c *gin.Context) {
	page, offset, size := pageOf(c, 15)
	word := strings.TrimSpace(c.Query("word"))
	enabled := atoiOr(c.Query("enabled"), -1)
	q := h.DB.Model(&model.EzfyActWild{})
	if word != "" {
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
	if enabled >= 0 {
		q = q.Where("enabled = ?", enabled)
	}
	var total int64
	q.Count(&total)
	var rows []model.EzfyActWild
	q.Order("id DESC").Offset(offset).Limit(size).Find(&rows)

	out := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		oName := ""
		oStar := 0
		if g := ezfyCfg.general(r.OfficerId); g != nil {
			oName = g.Name
			oStar = g.Star
		}
		out = append(out, gin.H{
			"id": r.ID, "x": r.X, "y": r.Y, "enabled": r.Enabled,
			"level": r.Level, "troops": r.Troops, "res": r.Res,
			"gold": r.Gold, "prestige": r.Prestige, "jewel": r.Jewel, "des": r.Des,
			"officer_id": r.OfficerId, "officer_name": oName, "officer_star": oStar,
			"treasures": r.Treasures, "capture_rate": r.CaptureRate, "max_capture": r.MaxCapture,
			// 当前生效判定（enabled=1 才算活动目标）
			"eff_act": (r.Enabled == 1),
		})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// AdminEzfyActWildSave POST /admin/ezfy-act-wilds —— 新增/更新某坐标的活动野地配置（按 x,y 幂等）
func (h *AdminHandler) AdminEzfyActWildSave(c *gin.Context) {
	var in map[string]interface{}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	vals := xyPickVals(in, ezfyActWildFields)
	x, _ := vals["x"].(int)
	y, _ := vals["y"].(int)
	if x <= 0 || y <= 0 {
		resp.ParamError(c, "请填写坐标 x / y（世界范围 ~50~450）")
		return
	}
	if msg := checkActWildVals(vals); msg != "" {
		resp.ParamError(c, msg)
		return
	}
	// 未填字段补默认
	if _, ok := vals["enabled"]; !ok {
		vals["enabled"] = 0
	}
	if _, ok := vals["level"]; !ok {
		vals["level"] = 0
	}
	if _, ok := vals["res"]; !ok {
		vals["res"] = 0
	}
	if _, ok := vals["gold"]; !ok {
		vals["gold"] = 0
	}
	if _, ok := vals["prestige"]; !ok {
		vals["prestige"] = 0
	}
	if s, _ := vals["troops"].(string); s == "" {
		vals["troops"] = ""
	}
	if s, _ := vals["jewel"].(string); s == "" {
		vals["jewel"] = ""
	}
	if s, _ := vals["des"].(string); s == "" {
		vals["des"] = ""
	}
	if _, ok := vals["officer_id"]; !ok {
		vals["officer_id"] = 0
	}
	if s, _ := vals["treasures"].(string); s == "" {
		vals["treasures"] = ""
	}
	if _, ok := vals["capture_rate"]; !ok {
		vals["capture_rate"] = 0
	}
	if _, ok := vals["max_capture"]; !ok {
		vals["max_capture"] = 1 // 默认同一玩家可抓 1 次
	}
	if err := h.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "x"}, {Name: "y"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"enabled", "level", "troops", "res", "gold", "prestige", "jewel", "des", "officer_id",
			"treasures", "capture_rate", "max_capture",
		}),
	}).Model(&model.EzfyActWild{}).Create(vals).Error; err != nil {
		resp.ParamError(c, "保存失败："+err.Error())
		return
	}
	h.ezfyReload()
	ok := "关闭（普通野地）"
	if v, _ := vals["enabled"].(int); v == 1 {
		ok = "启用（活动野地）"
	}
	resp.OK(c, gin.H{"msg": fmt.Sprintf("坐标 (%d,%d) 已保存并立即生效：%s", x, y, ok)})
}

// AdminEzfyActWildToggle POST /admin/ezfy-act-wilds/:id/toggle —— 切换启用开关
func (h *AdminHandler) AdminEzfyActWildToggle(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var a model.EzfyActWild
	if err := h.DB.First(&a, id).Error; err != nil {
		resp.NotFound(c, "该活动野地配置不存在")
		return
	}
	a.Enabled = 1 - a.Enabled
	h.DB.Model(&model.EzfyActWild{}).Where("id = ?", id).Update("enabled", a.Enabled)
	h.ezfyReload()
	resp.OK(c, gin.H{"msg": fmt.Sprintf("坐标 (%d,%d) 已%s", a.X, a.Y, map[bool]string{true: "启用为活动野地", false: "关闭（按普通野地）"}[a.Enabled == 1])})
}

// AdminEzfyActWildDelete DELETE /admin/ezfy-act-wilds/:id —— 删除配置，恢复按哈希/mark 判定
func (h *AdminHandler) AdminEzfyActWildDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var a model.EzfyActWild
	if err := h.DB.First(&a, id).Error; err != nil {
		resp.NotFound(c, "该活动野地配置不存在")
		return
	}
	h.DB.Delete(&model.EzfyActWild{}, id)
	h.ezfyReload()
	resp.OK(c, gin.H{"msg": fmt.Sprintf("坐标 (%d,%d) 的活动野地配置已删除，恢复默认判定", a.X, a.Y)})
}

// checkActWildVals 校验活动野地配置值
func checkActWildVals(vals map[string]interface{}) string {
	if v, ok := vals["enabled"].(int); ok && v != 0 && v != 1 {
		return "启用开关只能是 0（关闭）/ 1（启用）"
	}
	if v, ok := vals["level"].(int); ok && v != 0 && (v < 1 || v > 3) {
		return "活动等级只能是 1~3（0 = 用默认）"
	}
	if v, ok := vals["res"].(int64); ok && v < 0 {
		return "资源奖励不能为负"
	}
	if v, ok := vals["gold"].(int64); ok && v < 0 {
		return "黄金奖励不能为负"
	}
	if v, ok := vals["prestige"].(int); ok && v < 0 {
		return "声望奖励不能为负"
	}
	// 守军 JSON 校验
	if s, ok := vals["troops"].(string); ok && strings.TrimSpace(s) != "" {
		if parseActWildTroops(s) == nil {
			return "守军配置格式不对，应为 [[兵种id,数量],...]"
		}
	}
	// 必掉宝物 JSON 校验 [[cfg_id,count],...]
	if s, ok := vals["treasures"].(string); ok && strings.TrimSpace(s) != "" {
		if parseActWildTreasures(s) == nil {
			return "宝物配置格式不对，应为 [[宝物id,数量],...]"
		}
	}
	// 守将被俘虏概率 0~100
	if v, ok := vals["capture_rate"].(int); ok && (v < 0 || v > 100) {
		return "被俘虏概率只能是 0~100（0 = 不俘虏，100 = 必俘虏）"
	}
	// 同一玩家可抓次数：0 = 不限；否则 >= 1
	if v, ok := vals["max_capture"].(int); ok && v < 0 {
		return "可抓次数不能为负（0 = 不限，>=1 为上限）"
	}
	// 守将军官必须来自军官池（普通军官/名将都可选）
	if v, ok := vals["officer_id"].(int); ok && v > 0 {
		if ezfyCfg.general(v) == nil {
			return "守将军官不在军官池中，请重新选择"
		}
	}
	return ""
}