package ezfy

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 二战风云 · 计谋（2026-09-22 ）
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

// SchemeUse POST /games/ezfy/scheme/use  {scheme_id, target_x, target_y, order_id}
//
// 消耗信号弹发动计谋。
//
//	Kind=1（先发制人）需要给目标城市坐标 target_x/target_y。
//	Kind=2（神兵天降）/ Kind=3（战略转移）作用于「自己的某支部队」，需要给 order_id；
//	   · 神兵天降：去程(行进中)剩余时间减 80%
//	   · 战略转移：回程减 360 分钟
//	  每种计谋每支部队各限一次（用 EzfyOrder.SchemeUsed 位标记）。
func (h *EzfyHandler) SchemeUse(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	city := h.getOrCreateCity(uid)
	h.calcResource(&city)
	var req struct {
		SchemeId int   `json:"scheme_id"`
		TargetX  int   `json:"target_x"`
		TargetY  int   `json:"target_y"`
		OrderId  int64 `json:"order_id"`
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
	// ★ 2026-10-02 未实现的计谋一律卡控，提示暂未实现，不允许发动。
	//   目前已实现：Kind=1 先发制人 / Kind=2 神兵天降 / Kind=3 战略转移 /
	//              Kind=4 恫疑虚喝 / Kind=5 隐真示假。
	if sc.Kind < 1 || sc.Kind > 5 {
		h.fail(c, "「"+sc.Name+"」暂未实现, 敬请期待")
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
		// ★ 2026-10-02 先发制人**不能破免战保护令** ——
		//   目标城市处于免战保护期时不能对其发动（与出征拦截同口径 hasCityEffect）。
		if h.hasCityEffect(uint(t.ID), 2) {
			h.fail(c, "该城市使用了免战保护, 无法对其发动先发制人")
			return
		}
	}

	// ===== Kind 2/3 行军计谋：神兵天降(去程减80%) / 战略转移(回程减360分钟) =====
	// 作用于「自己的某支部队」，需 order_id；每种计谋每支部队各限一次。
	if sc.Kind == 2 || sc.Kind == 3 {
		if req.OrderId == 0 {
			h.fail(c, "请选择要发动计谋的部队")
			return
		}
		var order model.EzfyOrder
		if err := h.DB.Where("id = ? AND user_id = ?", req.OrderId, uid).First(&order).Error; err != nil {
			h.fail(c, "部队不存在")
			return
		}
		now := time.Now().UnixMilli()
		var flag int
		var newVal int64
		switch sc.Kind {
		case 2: // 神兵天降：去程(行进中)剩余时间减 80%
			if order.Status != 0 {
				h.fail(c, "神兵天降只能对行进中的部队使用")
				return
			}
			flag = 1 // bit0
			newVal = pctSpeedEnd(now, order.ArriveTime, 80)
		case 3: // 战略转移：回程减 360 分钟
			if order.Status != 2 {
				h.fail(c, "战略转移只能对返航中的部队使用")
				return
			}
			flag = 2 // bit1
			newVal = now + (order.ReturnTime - now) - 360*60000
			if newVal < now {
				newVal = now
			}
		}
		if order.SchemeUsed&flag != 0 {
			h.fail(c, "这支部队已经用过「"+sc.Name+"」了")
			return
		}
		// ★ 2026-09-30 用户反馈「返程剩 <360 分钟战略转移用不了」：
		//   战略转移减 360 分钟，剩余不足 360 分钟时 newVal 已被夹到 now(立即返航)，
		//   这是**期望效果**，不能当「即将返回」拒绝；只有神兵天降(去程)剩余时间
		//   本来就耗尽(newVal==now)才提示无需使用。
		if newVal <= time.Now().UnixMilli() && sc.Kind != 3 {
			h.fail(c, "这支队伍已即将到达/返回，无需使用计谋")
			return
		}
		// 扣信号弹 + 写位标记 + 改时间（一个事务里完成）
		h.DB.Transaction(func(tx *gorm.DB) error {
			h.consumeItemN(uid, ezfySchemeItemID, need, "发动计谋")
			if sc.Kind == 2 {
				tx.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
					Updates(map[string]interface{}{"arrive_time": newVal, "scheme_used": order.SchemeUsed | flag})
			} else {
				tx.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
					Updates(map[string]interface{}{"return_time": newVal, "scheme_used": order.SchemeUsed | flag})
			}
			return nil
		})
		msg := fmt.Sprintf("已发动计谋「%s」，消耗%s×%d：%s", sc.Name, name, need, sc.Des)
		h.addReport(uid, 6, "计谋发动: "+sc.Name, msg, "")
		h.done(c, "", msg)
		return
	}

	// ===== Kind 4/5 守城伪装计谋：恫疑虚喝(展示1亿假兵) / 隐真示假(展示1000内假兵) =====
	// ★ 2026-10-06 发动后自己**所有城市**生效 1 小时（多次发动叠加总时长，复用
	//   addCityEffect 的「剩余时间累加」逻辑）；仅在被敌人侦查出兵力数量时对敌方生效，
	//   实际兵种与数量不变。
	if sc.Kind == 4 || sc.Kind == 5 {
		effectType := 3 // 恫疑虚喝
		effectDesc := "被敌人侦查时展示随机兵种1亿兵效果, 实际兵力不变"
		if sc.Kind == 5 {
			effectType = 4 // 隐真示假
			effectDesc = "被敌人侦查时展示随机兵种极少兵力(几乎都在1000内), 隐藏实力"
		}
		h.consumeItemN(uid, ezfySchemeItemID, need, "发动计谋")
		h.ezfySchemeEffectCities(uid, effectType)
		msg := fmt.Sprintf("已发动计谋「%s」，消耗%s×%d：自己所有城市生效1小时（多次发动叠加时长），%s。",
			sc.Name, name, need, effectDesc)
		h.addReport(uid, 6, "计谋发动: "+sc.Name, msg+"\n"+sc.Des, "")
		h.done(c, "", msg)
		return
	}

	// ===== 扣信号弹 =====
	h.consumeItemN(uid, ezfySchemeItemID, need, "发动计谋")

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

// ============ 守城伪装计谋（恫疑虚喝 / 隐真示假） ============
//
// ★ 2026-10-06 效果存 ezfy_city_effect：effect_type 3 = 恫疑虚喝（展示 1亿 假兵），
//   4 = 隐真示假（展示 1000 内假兵）。发动后对玩家**所有城市**写入效果行，
//   每次发动 +1 小时（addCityEffect 把剩余时间与新时长累加 → 多次发动叠加总时长）。

// ezfySchemeEffectCities 把计谋伪装效果写到玩家名下所有城市（各 +1 小时，时长叠加）。
func (h *EzfyHandler) ezfySchemeEffectCities(uid uint, effectType int) {
	var cities []model.EzfyCity
	h.DB.Select("id").Where("user_id = ?", uid).Find(&cities)
	for _, ct := range cities {
		h.addCityEffect(ct.ID, effectType, 1, 1)
	}
}

// ezfySchemeFake 侦查目标城市主人是否有生效中的计谋伪装效果，返回：
//   3 = 恫疑虚喝（展示 1亿 假兵）/ 4 = 隐真示假（展示 1000 内假兵）/ 0 = 无效果。
// 计谋是「自己所有城市都生效」，这里按**主人维度** JOIN 查询：发动后才新建的城市
// 没有效果行，但主人其它城有 → 同样命中（与 hasAnyPeaceEffect 同一思路）。
// 两个效果同时生效时，隐真示假（隐藏实力）优先。
func (h *EzfyHandler) ezfySchemeFake(target *model.EzfyCity) int {
	if target == nil {
		return 0
	}
	var typ []int
	h.DB.Raw(
		"SELECT e.effect_type FROM ezfy_city_effect e JOIN ezfy_city c ON c.id = e.city_id "+
			"WHERE c.user_id = ? AND e.effect_type IN (3, 4) AND e.until_time > ? "+
			"GROUP BY e.effect_type", target.UserID, time.Now().UnixMilli()).Scan(&typ)
	if len(typ) == 0 {
		return 0
	}
	for _, t := range typ {
		if t == 4 {
			return 4 // 隐真示假优先
		}
	}
	return 3
}

// ezfySchemeFakeTroop 计谋伪装用随机兵种（同类别内随机一个，用于「随机兵种」展示）。
func ezfySchemeFakeTroop(troopType int) *model.EzfyCfgTroop {
	var pool []model.EzfyCfgTroop
	for _, t := range ezfyCfg.sortedTroops() {
		if t.Type == troopType {
			pool = append(pool, t)
		}
	}
	if len(pool) == 0 {
		return nil
	}
	t := pool[rand.Intn(len(pool))]
	return &t
}

// ezfySchemeFakeCount 计谋伪装假数量：
//   fakeKind 3（恫疑虚喝）→ 固定 1亿（100000000）吓唬敌人；
//   fakeKind 4（隐真示假）→ 1000 内随机小数量，隐藏实力。
func ezfySchemeFakeCount(fakeKind int) int64 {
	if fakeKind == 3 {
		return 100000000
	}
	return int64(rand.Intn(999) + 1)
}

// ============ 管理端：计谋配置 CRUD ============

// AdminEzfySchemes 计谋配置列表
func (h *EzfyAdmin) AdminEzfySchemes(c *gin.Context) {
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
func (h *EzfyAdmin) AdminEzfySchemeCreate(c *gin.Context) {
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
func (h *EzfyAdmin) AdminEzfySchemeUpdate(c *gin.Context) {
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
func (h *EzfyAdmin) AdminEzfySchemeDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var sc model.EzfyCfgScheme
	if err := h.DB.First(&sc, id).Error; err != nil {
		resp.NotFound(c, "计谋不存在")
		return
	}
	h.DB.Delete(&model.EzfyCfgScheme{}, id)
	resp.OK(c, gin.H{"msg": "计谋「" + sc.Name + "」已删除"})
}
