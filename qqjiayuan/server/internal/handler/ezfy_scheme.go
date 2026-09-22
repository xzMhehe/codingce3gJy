package handler

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 二战风云 · 计谋（2026-09-22 用户要求）
//
// 「信号弹也是道具，可以黄金、钻石购买，加上，用于计谋消耗。」
//
// 设计：
//   - 信号弹 = 普通道具（cfg_id=24, ItemType 20），黄金 / 钻石双渠道，库存无限；
//   - 计谋配置放 ezfy_cfg_scheme（管理端可维护：名称/说明/消耗数量/上下架），
//     不再写死在前端；
//   - 发动一次计谋 = 扣对应数量的信号弹 + 写一条战报；
//     Kind=1（先发制人）额外让双方立即进入「可战争」状态。
const ezfySchemeItemID = 24 // 信号弹

// ezfySchemeBulletName 信号弹的显示名（管理端可改名，别写死）
func (h *EzfyHandler) ezfySchemeBulletName() string {
	if it := ezfyCfg.item(ezfySchemeItemID); it != nil {
		return it.Name
	}
	return "信号弹"
}

// Schemes GET /games/ezfy/schemes —— 计谋列表（含持有信号弹数量）
func (h *EzfyHandler) Schemes(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	var rows []model.EzfyCfgScheme
	h.DB.Where("enabled <> 0").Order("sort_no, id").Find(&rows)
	have := h.itemCount(uid, ezfySchemeItemID)
	list := []gin.H{}
	for _, s := range rows {
		list = append(list, gin.H{
			"id": s.ID, "name": s.Name, "des": s.Des, "bullet": s.Bullet,
			"kind": s.Kind, "war_minutes": s.WarMinutes, "war_max_minutes": s.WarMaxMinutes,
			"enough": have >= s.Bullet,
		})
	}
	resp.OK(c, gin.H{
		"schemes":        list,
		"bullet_item_id": ezfySchemeItemID, "bullet_name": h.ezfySchemeBulletName(),
		"bullet_have": have,
	})
}

// ezfySchemeOfficerLearning 取「发动计谋的军官」的学识
//
// 原版写「可战争时间为军官学识×1分钟」。这里按优先级取：
// 市长 → 城守 → 本城学识最高的军官；一个都没有就按 0 算（只拿保底时长）。
func (h *EzfyHandler) ezfySchemeOfficerLearning(city *model.EzfyCity) (string, int) {
	for _, pos := range []int{ezfyPositionMayor, ezfyPositionGuard} {
		if o := h.positionOfficer(city.ID, pos); o != nil {
			_, _, lea := h.officerEffective(o)
			return o.Name, lea
		}
	}
	var best *model.EzfyOfficer
	var bestLea int
	for _, o := range h.officerList(city.ID) {
		if o.IsCaptive == 1 {
			continue
		}
		oo := o
		_, _, lea := h.officerEffective(&oo)
		if best == nil || lea > bestLea {
			best = &oo
			bestLea = lea
		}
	}
	if best == nil {
		return "", 0
	}
	return best.Name, bestLea
}

// SchemeUse POST /games/ezfy/scheme/use  {scheme_id, target_x, target_y}
//
// 消耗信号弹发动计谋。Kind=1（先发制人）需要给目标城市坐标。
func (h *EzfyHandler) SchemeUse(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	city := h.getOrCreateCity(uid)
	h.calcResource(&city)
	var req struct {
		SchemeId int `json:"scheme_id"`
		TargetX  int `json:"target_x"`
		TargetY  int `json:"target_y"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	var sc model.EzfyCfgScheme
	if err := h.DB.First(&sc, req.SchemeId).Error; err != nil || sc.Enabled == 0 {
		h.fail(c, "计谋不存在或已下架")
		return
	}
	need := sc.Bullet
	if need <= 0 {
		need = 1
	}
	name := h.ezfySchemeBulletName()
	if have := h.itemCount(uid, ezfySchemeItemID); have < need {
		h.fail(c, fmt.Sprintf("%s不足: 发动「%s」需要%d个, 当前只有%d个（可在商城购买）",
			name, sc.Name, need, have))
		return
	}

	// ===== Kind=1 先发制人：需要目标城市，且 6 小时内不能重复中计 =====
	var target *model.EzfyCity
	if sc.Kind == 1 {
		if req.TargetX == 0 && req.TargetY == 0 {
			h.fail(c, "请选择要发动计谋的目标城市")
			return
		}
		var t model.EzfyCity
		if err := h.DB.Where("x = ? AND y = ?", req.TargetX, req.TargetY).First(&t).Error; err != nil {
			h.fail(c, "目标坐标上没有城市")
			return
		}
		if t.UserID == uid {
			h.fail(c, "不能对自己的城市发动计谋")
			return
		}
		target = &t
		nowMs := time.Now().UnixMilli()
		// 「中计城市 6 小时内不再中计」
		//
		// ★ 两点都要注意：
		//   ① 只看**已经生效过**的战争记录（EffectTime <= now）——
		//      一条「宣战待生效」(status=1) 的 EffectTime 在未来，
		//      拿它算 `now - EffectTime` 会得到负数 → 被误判成「刚中过计」（踩过）。
		//   ② 要扫**所有**方向的记录（A→B 和 B→A 都算），不能只取一条。
		var wars []model.EzfyWar
		h.DB.Where("(atk_user_id = ? AND def_user_id = ?) OR (atk_user_id = ? AND def_user_id = ?)",
			uid, t.UserID, t.UserID, uid).Find(&wars)
		for i := range wars {
			w := wars[i]
			if w.EffectTime <= nowMs && nowMs-w.EffectTime < 6*3600*1000 {
				h.fail(c, "该城市 6 小时内已经中过「先发制人」，请稍后再试")
				return
			}
		}
	}

	// ===== 扣信号弹 =====
	h.consumeItemN(uid, ezfySchemeItemID, need)

	// ===== 生效 + 写战报 =====
	msg := fmt.Sprintf("已发动计谋「%s」，消耗%s×%d", sc.Name, name, need)
	if sc.Kind == 1 && target != nil {
		officerName, lea := h.ezfySchemeOfficerLearning(&city)
		// 原版：可战争时间 = 军官学识 × 1 分钟（学识就是分钟数），最多持续 6 小时
		minutes := lea
		if minutes < 1 {
			minutes = 1
		}
		if sc.WarMaxMinutes > 0 && minutes > sc.WarMaxMinutes {
			minutes = sc.WarMaxMinutes
		}
		nowMs := time.Now().UnixMilli()
		h.DB.Create(&model.EzfyWar{
			AtkUserId: uid, DefUserId: target.UserID, Status: 2,
			DeclareTime: nowMs, EffectTime: nowMs, ExpireTime: nowMs + int64(minutes)*60000,
		})
		who := officerName
		if who == "" {
			who = "无军官"
		}
		msg = fmt.Sprintf("已发动计谋「先发制人」，消耗%s×%d：与「%s」进入可战争状态 %d 分钟（%s 学识 %d）",
			name, need, target.Name, minutes, who, lea)
		h.addReport(target.UserID, 3, "计谋: 先发制人",
			fmt.Sprintf("我方城市「%s」被敌军发动计谋「先发制人」，已进入可战争状态 %d 分钟。",
				target.Name, minutes), "")
	}
	h.addReport(uid, 6, "计谋发动: "+sc.Name, msg+"\n"+sc.Des, "")
	h.done(c, "", msg)
}

// ============ 管理端：计谋配置 CRUD ============

// AdminEzfySchemes 计谋配置列表
func (h *AdminHandler) AdminEzfySchemes(c *gin.Context) {
	word := trimStr(c.Query("word"), 50)
	q := h.DB.Model(&model.EzfyCfgScheme{})
	if word != "" {
		if id, err := strconv.Atoi(word); err == nil {
			q = q.Where("id = ?", id)
		} else {
			q = q.Where("name LIKE ?", "%"+word+"%")
		}
	}
	var rows []model.EzfyCfgScheme
	q.Order("sort_no, id").Find(&rows)
	resp.OK(c, gin.H{"list": rows, "total": len(rows)})
}

// AdminEzfySchemeCreate 新增计谋
func (h *AdminHandler) AdminEzfySchemeCreate(c *gin.Context) {
	var in map[string]interface{}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	vals := xyPickVals(in, ezfySchemeFields)
	if vals["name"] == nil {
		resp.ParamError(c, "请填写计谋名称")
		return
	}
	if _, ok := vals["bullet"]; !ok {
		vals["bullet"] = 4
	}
	if _, ok := vals["enabled"]; !ok {
		vals["enabled"] = 1
	}
	if err := h.DB.Model(&model.EzfyCfgScheme{}).Create(vals).Error; err != nil {
		resp.ParamError(c, "新增失败："+err.Error())
		return
	}
	resp.OK(c, gin.H{"msg": "计谋已新增"})
}

// AdminEzfySchemeUpdate 修改计谋
func (h *AdminHandler) AdminEzfySchemeUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var sc model.EzfyCfgScheme
	if err := h.DB.First(&sc, id).Error; err != nil {
		resp.NotFound(c, "计谋不存在")
		return
	}
	var in map[string]interface{}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	vals := xyPickVals(in, ezfySchemeFields)
	if len(vals) == 0 {
		resp.ParamError(c, "无可修改字段")
		return
	}
	if err := h.DB.Model(&model.EzfyCfgScheme{}).Where("id = ?", id).Updates(vals).Error; err != nil {
		resp.ParamError(c, "修改失败："+err.Error())
		return
	}
	resp.OK(c, gin.H{"msg": "计谋「" + sc.Name + "」已保存"})
}

// AdminEzfySchemeDelete 删除计谋
func (h *AdminHandler) AdminEzfySchemeDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var sc model.EzfyCfgScheme
	if err := h.DB.First(&sc, id).Error; err != nil {
		resp.NotFound(c, "计谋不存在")
		return
	}
	h.DB.Delete(&model.EzfyCfgScheme{}, id)
	resp.OK(c, gin.H{"msg": "计谋「" + sc.Name + "」已删除"})
}
