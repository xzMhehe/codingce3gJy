package handler

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 二战风云 预设编队（司令部保存的出征模板：军官 + 集结令 + 兵力，不含目标/随军资源/宿营）
// 数据存 ezfy_preset 表，按 user_id 隔离；管理端零改动。

// 单账号最多保存的预设数
const ezfyPresetMax = 10

// Presets GET /games/ezfy/presets
func (h *EzfyHandler) Presets(c *gin.Context) {
	uid := middleware.GetUID(c)
	list := []model.EzfyPreset{}
	h.DB.Where("user_id = ?", uid).Order("id DESC").Find(&list)
	out := make([]gin.H, 0, len(list))
	for _, p := range list {
		troops := parseGroups(p.Troops)
		var total int64
		for _, t := range troops {
			total += t.Count
		}
		out = append(out, gin.H{
			"id":         p.ID,
			"name":       p.Name,
			"officer":    p.Officer,
			"gather":     p.Gather,
			"troops":     troops,
			"troop_total": total,
		})
	}
	resp.OK(c, gin.H{"presets": out})
}

// PresetAdd POST /games/ezfy/presets  {name, officer, gather, troops:[{troopId,count}]}
func (h *EzfyHandler) PresetAdd(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Name    string          `json:"name"`
		Officer string          `json:"officer"`
		Gather  int             `json:"gather"`
		Troops  []ezfyUnitGroup `json:"troops"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		resp.ParamError(c, "请填写预设名称")
		return
	}
	if len([]rune(name)) > 20 {
		resp.ParamError(c, "预设名称最多20字")
		return
	}
	// 兵力校验：兵种必须存在且非城防(type!=4)，数量合理，条数/长度防撑爆 varchar(2000)
	h.cfgs()
	if len(req.Troops) > 20 {
		resp.ParamError(c, "预设兵力最多20种兵种")
		return
	}
	valid := []ezfyUnitGroup{}
	for _, t := range req.Troops {
		if t.Count <= 0 {
			continue
		}
		cfg := ezfyCfg.troop(t.TroopId)
		if cfg == nil || cfg.Type == 4 {
			resp.ParamError(c, "包含不可出征的兵种")
			return
		}
		if t.Count > 1000000000 {
			resp.ParamError(c, "单兵种数量过大")
			return
		}
		valid = append(valid, t)
	}
	troopsJson, _ := json.Marshal(valid)
	if len(troopsJson) > 2000 {
		resp.ParamError(c, "预设兵力数据过长")
		return
	}
	// 集结令个数夹到管理端上限（与 OrderPreview 同口径）
	gather := req.Gather
	if gather < 0 {
		gather = 0
	}
	if mx := ezfyGatherMax(); gather > mx {
		gather = mx
	}
	// 数量上限
	var cnt int64
	h.DB.Model(&model.EzfyPreset{}).Where("user_id = ?", uid).Count(&cnt)
	if cnt >= ezfyPresetMax {
		resp.ParamError(c, "最多保存"+strconv.Itoa(ezfyPresetMax)+"个预设")
		return
	}
	officer := strings.TrimSpace(req.Officer)
	h.DB.Create(&model.EzfyPreset{UserID: uid, Name: name, Officer: officer, Gather: gather, Troops: string(troopsJson)})
	resp.OK(c, gin.H{"msg": "预设「" + name + "」已保存"})
}

// PresetDelete POST /games/ezfy/presets/delete  {id}
func (h *EzfyHandler) PresetDelete(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Id int64 `json:"id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	h.DB.Where("id = ? AND user_id = ?", req.Id, uid).Delete(&model.EzfyPreset{})
	resp.OK(c, gin.H{"msg": "已删除预设"})
}
