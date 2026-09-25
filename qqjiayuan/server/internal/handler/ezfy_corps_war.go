package handler

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 二战风云 —— 军团外交 / 军团宣战 / 军团积分 / 军团商城（★ 2026-09-25 用户要求）
//
// 用户规则（已确认）：
//  1. 军团之间可标记外交关系：友好/敌对（军团长操作，**双向各记一条**）。友好、敌对都可以宣战。
//  2. 军团长可对另一个军团宣战：宣战后 **12 小时生效**，宣战后 **48 小时整场结束**
//     （即生效窗口 = 第 12~48 小时，共 36 小时）。
//  3. 生效期间双方军团**成员之间**可互相掠夺(订单2)/征服(订单3)，**不需要个人宣战**；
//     掠夺/征服获胜可获得军团战绩积分：**军团总积分 + 成员个人积分**（掠夺胜 +10，征服胜 +20）。
//  4. 军团商城货币 = 成员**个人军团积分**；商品分两类：资源包（food/steel/oil/rare/gold）、
//     道具（复用现有游戏道具配置 EzfyCfgItem，"道具池" = 现有道具表，发放走现有 addItem）。
//
// ★ 状态约定（与 model.EzfyCorpsWar 严格一致，别自创）：
//
//	1 = 待生效   2 = 交战中   3 = 已结束
const (
	ezfyCorpsWarDelayHours    = 12 // 宣战后多少小时生效
	ezfyCorpsWarDurationHours = 36 // 生效窗口（第 12~48 小时，共 36 小时）
	ezfyCorpsWarTotalHours    = 48 // 宣战后多少小时整场结束
	ezfyCorpsWarPointOrder2   = 10 // 掠夺获胜：个人/军团积分 +10
	ezfyCorpsWarPointOrder3   = 20 // 征服获胜：个人/军团积分 +20
)

// ============ 公共 helper ============

// corpsOfUser 取用户所属军团ID（无军团返回 0）
func (h *EzfyHandler) corpsOfUser(uid uint) uint {
	var mb model.EzfyCorpsMember
	if err := h.DB.Where("user_id = ?", uid).First(&mb).Error; err != nil {
		return 0
	}
	return mb.CorpsId
}

// corpsLeaderOf 取「当前用户是军团长」的军团ID（不是军团长返回 ok=false）
func (h *EzfyHandler) corpsLeaderOf(uid uint) (uint, bool) {
	var mb model.EzfyCorpsMember
	if err := h.DB.Where("user_id = ?", uid).First(&mb).Error; err != nil {
		return 0, false
	}
	if mb.IsLeader != 1 {
		return 0, false
	}
	return mb.CorpsId, true
}

// corpsNameById 取军团名（不存在返回空串）
func (h *EzfyHandler) corpsNameById(id uint) string {
	var cp model.EzfyCorps
	if err := h.DB.First(&cp, id).Error; err != nil {
		return ""
	}
	return cp.Name
}

// corpsLeaderNameOf 取军团长展示名
func (h *EzfyHandler) corpsLeaderNameOf(cp *model.EzfyCorps) string {
	p := h.ensureProfile(cp.LeaderUserId)
	if p.UserID == cp.LeaderUserId && p.Nickname != "" {
		return p.Nickname
	}
	return "未知"
}

// corpsWarTick 把军团宣战的状态按时间推进（供各接口调用前刷新）
//
// ★ 之所以要「惰性推进」：库里 status 可能是 1，但 EffectTime 早就过了 ——
// 没人访问就永远不刷新；且服务器停机期间到期的行也要归档。
//
//	① 待生效(1) 到生效时间 且 未过整场结束 → 交战中(2)
//	② 待生效/交战中 已过整场结束时间 → 已结束(3) + EndTime（含停机期间到期的）
func (h *EzfyHandler) corpsWarTick() {
	now := time.Now().UnixMilli()
	h.DB.Model(&model.EzfyCorpsWar{}).
		Where("status = 1 AND effect_time <= ? AND expire_time > ?", now, now).
		Update("status", 2)
	h.DB.Model(&model.EzfyCorpsWar{}).
		Where("status IN (1,2) AND expire_time <= ?", now).
		Updates(map[string]interface{}{"status": 3, "end_time": now})
}

// corpsActiveWarBetween 两人所属军团之间是否存在「生效中」的军团宣战
//
// 生效中 = Status 2 且 now ∈ [EffectTime, ExpireTime]。调用前先 corpsWarTick
// 把到期未结束的行归档，避免拿到陈旧状态。
func (h *EzfyHandler) corpsActiveWarBetween(a, b uint) *model.EzfyCorpsWar {
	ca, cb := h.corpsOfUser(a), h.corpsOfUser(b)
	if ca == 0 || cb == 0 || ca == cb {
		return nil
	}
	return h.corpsActiveWarBetweenCorps(ca, cb)
}

// corpsActiveWarBetweenCorps 两个军团之间的生效中宣战（无则 nil）
func (h *EzfyHandler) corpsActiveWarBetweenCorps(ca, cb uint) *model.EzfyCorpsWar {
	if ca == 0 || cb == 0 || ca == cb {
		return nil
	}
	h.corpsWarTick()
	now := time.Now().UnixMilli()
	var w model.EzfyCorpsWar
	err := h.DB.Where("status = 2 AND effect_time <= ? AND expire_time > ? AND "+
		"((atk_corps_id = ? AND def_corps_id = ?) OR (atk_corps_id = ? AND def_corps_id = ?))",
		now, now, ca, cb, cb, ca).Order("id DESC").First(&w).Error
	if err != nil {
		return nil
	}
	return &w
}

// ezfyCorpsWarLeftHours 剩余小时数（状态 1 = 距生效；状态 2 = 距整场结束），向上取整，负数归 0
func ezfyCorpsWarLeftHours(w *model.EzfyCorpsWar, now int64) int64 {
	if w.Status == 3 {
		return 0
	}
	target := w.ExpireTime
	if w.Status == 1 {
		target = w.EffectTime
	}
	d := target - now
	if d <= 0 {
		return 0
	}
	return (d + 3599999) / 3600000
}

// ezfyCorpsWarStatusName 状态中文名（含剩余小时）
func ezfyCorpsWarStatusName(w *model.EzfyCorpsWar, now int64) string {
	switch w.Status {
	case 1:
		return fmt.Sprintf("待生效(约%d小时)", ezfyCorpsWarLeftHours(w, now))
	case 2:
		return fmt.Sprintf("交战中(剩余约%d小时)", ezfyCorpsWarLeftHours(w, now))
	case 3:
		return "已结束"
	}
	return "未知"
}

// ezfyCorpsRelationName 外交关系类型中文名
func ezfyCorpsRelationName(t int) string {
	switch t {
	case 1:
		return "友好"
	case 2:
		return "敌对"
	}
	return "未标记"
}

// ezfyCorpsMallKindName 商城商品类型中文名
func ezfyCorpsMallKindName(k int) string {
	if k == 1 {
		return "资源包"
	}
	if k == 2 {
		return "道具"
	}
	return "未知"
}

// ezfyCorpsMallResText 资源包内容文案（只列非 0 的项）
func ezfyCorpsMallResText(it *model.EzfyCorpsMall) string {
	if it.Kind != 1 {
		return ""
	}
	parts := []string{}
	if it.Food != 0 {
		parts = append(parts, fmt.Sprintf("粮%d", it.Food))
	}
	if it.Steel != 0 {
		parts = append(parts, fmt.Sprintf("钢%d", it.Steel))
	}
	if it.Oil != 0 {
		parts = append(parts, fmt.Sprintf("油%d", it.Oil))
	}
	if it.Rare != 0 {
		parts = append(parts, fmt.Sprintf("稀矿%d", it.Rare))
	}
	if it.Gold != 0 {
		parts = append(parts, fmt.Sprintf("金%d", it.Gold))
	}
	return strings.Join(parts, " ")
}

// ezfyCorpsWarNotify 给宣战双方军团**全体成员**各发一条系统通知
func (h *EzfyHandler) ezfyCorpsWarNotify(w *model.EzfyCorpsWar) {
	body := fmt.Sprintf("【军团宣战】%s 军团 与 %s 军团 已互相宣战：%d 小时后生效，%d 小时后整场结束。\n"+
		"生效期间双方军团成员之间可互相掠夺/征服（无需个人宣战），获胜可获得军团战绩积分（掠夺胜+%d，征服胜+%d）。",
		w.AtkCorpsName, w.DefCorpsName, ezfyCorpsWarDelayHours, ezfyCorpsWarTotalHours,
		ezfyCorpsWarPointOrder2, ezfyCorpsWarPointOrder3)
	var members []model.EzfyCorpsMember
	h.DB.Where("corps_id IN ?", []uint{w.AtkCorpsId, w.DefCorpsId}).Find(&members)
	seen := map[uint]bool{}
	for _, m := range members {
		if m.UserId == 0 || seen[m.UserId] {
			continue
		}
		seen[m.UserId] = true
		h.DB.Create(&model.EzfyNotice{UserId: m.UserId, Title: "军团宣战", Content: body})
	}
}

// ezfyCorpsWarAward 军团战绩发放：攻击方在「军团交战期」内攻打另一军团成员获胜后调用
//
// ★ 调用点只在掠夺(2)/征服(3) 攻打**玩家城市**且**攻击方获胜**的分支里（只加一次）。
// 内部自己判断是否处于军团交战期（含军团校验），不处于则什么都不做。
//
//	① 军团战绩：攻方是我方军团 → AtkPoint 累加，否则 → DefPoint 累加（本次宣战双方各自累计）
//	② 获胜军团「总积分」+ 攻击者「个人积分」各 +N（N：掠夺 10 / 征服 20），全部走原子表达式防并发。
func (h *EzfyHandler) ezfyCorpsWarAward(atkUid, defUid uint, orderType int) {
	if orderType != 2 && orderType != 3 {
		return
	}
	ca, cb := h.corpsOfUser(atkUid), h.corpsOfUser(defUid)
	if ca == 0 || cb == 0 || ca == cb {
		return
	}
	w := h.corpsActiveWarBetweenCorps(ca, cb)
	if w == nil {
		return
	}
	point := int64(ezfyCorpsWarPointOrder2)
	if orderType == 3 {
		point = int64(ezfyCorpsWarPointOrder3)
	}
	// ① 本次宣战的双方各自累计战绩（攻方获胜 → 记到攻击者所属军团那一列）
	col := "def_point"
	if w.AtkCorpsId == ca {
		col = "atk_point"
	}
	h.DB.Model(&model.EzfyCorpsWar{}).Where("id = ?", w.ID).
		Update(col, gorm.Expr(col+" + ?", point))
	// ② 获胜军团总积分 + 攻击者个人军团积分
	h.DB.Model(&model.EzfyCorps{}).Where("id = ?", ca).
		Update("points", gorm.Expr("points + ?", point))
	h.DB.Model(&model.EzfyCorpsMember{}).Where("user_id = ?", atkUid).
		Update("points", gorm.Expr("points + ?", point))
}

// ============ 1. 军团外交 ============

// CorpsRelations GET /games/ezfy/corps/relations
//
// 返回：in_corps / can_manage / my_corps / relations（本军团已标记的关系）/
// corps_list（所有军团 + 与本军团的关系与进行中宣战状态）
func (h *EzfyHandler) CorpsRelations(c *gin.Context) {
	uid := middleware.GetUID(c)
	myCorpsId := h.corpsOfUser(uid)
	canManage := false
	myView := gin.H{}
	var myCorps model.EzfyCorps
	if myCorpsId != 0 {
		h.DB.First(&myCorps, myCorpsId)
		var mb model.EzfyCorpsMember
		if err := h.DB.Where("user_id = ?", uid).First(&mb).Error; err == nil {
			canManage = mb.IsLeader == 1
		}
		myView = gin.H{"id": myCorps.ID, "name": myCorps.Name, "points": myCorps.Points}
	}

	// 本军团已标记的外交关系（单向读一份即可，双向是两条对称记录）
	relType := map[uint]int{}
	relations := []gin.H{}
	if myCorpsId != 0 {
		var rels []model.EzfyCorpsRelation
		h.DB.Where("corps_id = ?", myCorpsId).Find(&rels)
		for _, r := range rels {
			name := h.corpsNameById(r.TargetCorpsId)
			if name == "" {
				continue
			}
			relType[r.TargetCorpsId] = r.Type
			relations = append(relations, gin.H{"corps_id": r.TargetCorpsId, "name": name,
				"type": r.Type, "type_name": ezfyCorpsRelationName(r.Type)})
		}
	}

	// 进行中的军团宣战（我方军团维度：对方军团ID → 状态/剩余小时）
	h.corpsWarTick()
	now := time.Now().UnixMilli()
	warStatus := map[uint]int{}
	warLeft := map[uint]int64{}
	if myCorpsId != 0 {
		var wars []model.EzfyCorpsWar
		h.DB.Where("(atk_corps_id = ? OR def_corps_id = ?) AND status IN (1,2)", myCorpsId, myCorpsId).Find(&wars)
		for i := range wars {
			w := wars[i]
			other := w.DefCorpsId
			if w.DefCorpsId == myCorpsId {
				other = w.AtkCorpsId
			}
			warStatus[other] = w.Status
			warLeft[other] = ezfyCorpsWarLeftHours(&w, now)
		}
	}

	var corps []model.EzfyCorps
	h.DB.Order("id DESC").Find(&corps)
	ids := make([]int64, 0, len(corps))
	for _, cp := range corps {
		ids = append(ids, int64(cp.ID))
	}
	counts := h.corpsMemberCountMap(ids)
	corpsList := make([]gin.H, 0, len(corps))
	for i := range corps {
		cp := corps[i]
		// ★ 前端在同一张表上渲染 [标记友好]/[标记敌对]/[宣战] 按钮，
		//   自己军团出现在列表里只会产生必然失败的无效按钮，故排除（后端也校验了不能对自己操作）。
		if myCorpsId != 0 && cp.ID == myCorpsId {
			continue
		}
		corpsList = append(corpsList, gin.H{
			"id": cp.ID, "name": cp.Name,
			"member_count":    counts[int64(cp.ID)],
			"leader_name":     h.corpsLeaderNameOf(&cp),
			"points":          cp.Points,
			"relation_type":   relType[cp.ID],
			"war_status":      warStatus[cp.ID],
			"war_remaining_h": warLeft[cp.ID],
		})
	}
	resp.OK(c, gin.H{"in_corps": myCorpsId != 0, "can_manage": canManage,
		"my_corps": myView, "relations": relations, "corps_list": corpsList})
}

// CorpsRelationSet POST /games/ezfy/corps/relation {corps_id, type(0取消 1友好 2敌对)}
//
// ★ 仅军团长可操作；type≠0 时双向各写一条，type=0 时删两条。
func (h *EzfyHandler) CorpsRelationSet(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		CorpsId uint `json:"corps_id"`
		Type    int  `json:"type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	myCorpsId, ok := h.corpsLeaderOf(uid)
	if !ok {
		resp.ParamError(c, "只有军团长可以设置军团外交关系")
		return
	}
	if req.CorpsId == 0 || req.CorpsId == myCorpsId {
		resp.ParamError(c, "目标军团不正确")
		return
	}
	if req.Type < 0 || req.Type > 2 {
		resp.ParamError(c, "外交关系类型错误")
		return
	}
	var target model.EzfyCorps
	if err := h.DB.First(&target, req.CorpsId).Error; err != nil {
		resp.ParamError(c, "目标军团不存在")
		return
	}
	if req.Type == 0 {
		h.DB.Where("(corps_id = ? AND target_corps_id = ?) OR (corps_id = ? AND target_corps_id = ?)",
			myCorpsId, req.CorpsId, req.CorpsId, myCorpsId).Delete(&model.EzfyCorpsRelation{})
		resp.OK(c, gin.H{"msg": "已取消与「" + target.Name + "」的外交标记"})
		return
	}
	// 双向写入（幂等 upsert：uk_pair = corps_id + target_corps_id）
	for _, pair := range [][2]uint{{myCorpsId, req.CorpsId}, {req.CorpsId, myCorpsId}} {
		h.DB.Exec("INSERT INTO ezfy_corps_relation(corps_id, target_corps_id, type, created_at, updated_at) "+
			"VALUES(?, ?, ?, NOW(), NOW()) ON DUPLICATE KEY UPDATE type = VALUES(type), updated_at = NOW()",
			pair[0], pair[1], req.Type)
	}
	resp.OK(c, gin.H{"msg": "已标记与「" + target.Name + "」为" + ezfyCorpsRelationName(req.Type)})
}

// ============ 2. 军团宣战 ============

// CorpsWarDeclare POST /games/ezfy/corps/war/declare {corps_id}
//
// ★ 仅军团长；双方之间不能有进行中(1/2)的宣战；宣战后 12 小时生效、48 小时整场结束。
func (h *EzfyHandler) CorpsWarDeclare(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		CorpsId uint `json:"corps_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	myCorpsId, ok := h.corpsLeaderOf(uid)
	if !ok {
		resp.ParamError(c, "只有军团长可以宣战")
		return
	}
	if req.CorpsId == 0 || req.CorpsId == myCorpsId {
		resp.ParamError(c, "不能对自己军团宣战")
		return
	}
	var myCorps, target model.EzfyCorps
	if err := h.DB.First(&myCorps, myCorpsId).Error; err != nil {
		resp.ParamError(c, "军团不存在")
		return
	}
	if err := h.DB.First(&target, req.CorpsId).Error; err != nil {
		resp.ParamError(c, "目标军团不存在")
		return
	}
	var n int64
	h.DB.Model(&model.EzfyCorpsWar{}).
		Where("status IN (1,2) AND ((atk_corps_id = ? AND def_corps_id = ?) OR (atk_corps_id = ? AND def_corps_id = ?))",
			myCorpsId, req.CorpsId, req.CorpsId, myCorpsId).Count(&n)
	if n > 0 {
		resp.ParamError(c, "双方已存在进行中的军团宣战，无需重复宣战")
		return
	}
	now := time.Now().UnixMilli()
	w := model.EzfyCorpsWar{
		AtkCorpsId: myCorpsId, DefCorpsId: req.CorpsId,
		AtkCorpsName: myCorps.Name, DefCorpsName: target.Name,
		AtkUserId: uid, Status: 1,
		DeclareTime: now,
		EffectTime:  now + int64(ezfyCorpsWarDelayHours)*3600000,
		ExpireTime:  now + int64(ezfyCorpsWarTotalHours)*3600000,
	}
	if err := h.DB.Create(&w).Error; err != nil {
		resp.ParamError(c, "宣战失败："+err.Error())
		return
	}
	h.ezfyCorpsWarNotify(&w)
	h.ezfySysChat("【军团宣战】%s 军团向 %s 军团宣战了，%d 小时后生效，%d 小时后整场结束！",
		myCorps.Name, target.Name, ezfyCorpsWarDelayHours, ezfyCorpsWarTotalHours)
	resp.OK(c, gin.H{
		"msg": fmt.Sprintf("已向「%s」军团宣战，%d 小时后生效，%d 小时后整场结束",
			target.Name, ezfyCorpsWarDelayHours, ezfyCorpsWarTotalHours),
		"id": w.ID,
	})
}

// CorpsWarList GET /games/ezfy/corps/war
//
// 本军团相关宣战列表（进行中在前，最近结束的保留若干条）
func (h *EzfyHandler) CorpsWarList(c *gin.Context) {
	uid := middleware.GetUID(c)
	myCorpsId := h.corpsOfUser(uid)
	canManage := false
	myView := gin.H{}
	if myCorpsId == 0 {
		resp.OK(c, gin.H{"in_corps": false, "can_manage": false, "my_corps": myView, "wars": []gin.H{}})
		return
	}
	var myCorps model.EzfyCorps
	h.DB.First(&myCorps, myCorpsId)
	var mb model.EzfyCorpsMember
	if err := h.DB.Where("user_id = ?", uid).First(&mb).Error; err == nil {
		canManage = mb.IsLeader == 1
	}
	myView = gin.H{"id": myCorps.ID, "name": myCorps.Name, "points": myCorps.Points}

	h.corpsWarTick()
	now := time.Now().UnixMilli()
	var active, ended []model.EzfyCorpsWar
	h.DB.Where("(atk_corps_id = ? OR def_corps_id = ?) AND status IN (1,2)", myCorpsId, myCorpsId).
		Order("id DESC").Find(&active)
	h.DB.Where("(atk_corps_id = ? OR def_corps_id = ?) AND status = 3", myCorpsId, myCorpsId).
		Order("id DESC").Limit(10).Find(&ended)

	wars := make([]gin.H, 0, len(active)+len(ended))
	appendWar := func(w model.EzfyCorpsWar) {
		mineIsAtk := w.AtkCorpsId == myCorpsId
		oppId, oppName := w.DefCorpsId, w.DefCorpsName
		myPoint, oppPoint := w.AtkPoint, w.DefPoint
		if !mineIsAtk {
			oppId, oppName = w.AtkCorpsId, w.AtkCorpsName
			myPoint, oppPoint = w.DefPoint, w.AtkPoint
		}
		wars = append(wars, gin.H{
			"id": w.ID, "opp_corps_id": oppId, "opp_corps_name": oppName,
			"mine_is_atk": mineIsAtk, "status": w.Status,
			"status_name":  ezfyCorpsWarStatusName(&w, now),
			"declare_time": w.DeclareTime, "effect_time": w.EffectTime, "expire_time": w.ExpireTime,
			"end_time": w.EndTime, "my_point": myPoint, "opp_point": oppPoint,
			"remaining_h": ezfyCorpsWarLeftHours(&w, now),
		})
	}
	for _, w := range active {
		appendWar(w)
	}
	for _, w := range ended {
		appendWar(w)
	}
	resp.OK(c, gin.H{"in_corps": true, "can_manage": canManage, "my_corps": myView, "wars": wars})
}

// ============ 3. 军团商城 ============

// CorpsMallList GET /games/ezfy/corps/mall
func (h *EzfyHandler) CorpsMallList(c *gin.Context) {
	uid := middleware.GetUID(c)
	myCorpsId := h.corpsOfUser(uid)
	if myCorpsId == 0 {
		resp.OK(c, gin.H{"in_corps": false, "my_points": 0, "corps_points": 0,
			"items": []gin.H{}, "my_bought": gin.H{}})
		return
	}
	var mb model.EzfyCorpsMember
	h.DB.Where("user_id = ?", uid).First(&mb)
	var cp model.EzfyCorps
	h.DB.First(&cp, myCorpsId)

	var rows []model.EzfyCorpsMall
	h.DB.Where("enabled = 1").Order("sort ASC, id ASC").Find(&rows)
	// 道具名（前端「内容」列用 item_name，缺失时回退到商品名）
	itemNames := map[int]string{}
	cfgIds := []int{}
	for i := range rows {
		if rows[i].Kind == 2 && rows[i].ItemId > 0 {
			cfgIds = append(cfgIds, rows[i].ItemId)
		}
	}
	if len(cfgIds) > 0 {
		var cfgs []model.EzfyCfgItem
		h.DB.Where("id IN ?", cfgIds).Find(&cfgs)
		for _, cf := range cfgs {
			itemNames[cf.ID] = cf.Name
		}
	}
	items := make([]gin.H, 0, len(rows))
	for i := range rows {
		it := rows[i]
		items = append(items, gin.H{
			"id": it.ID, "kind": it.Kind, "kind_name": ezfyCorpsMallKindName(it.Kind), "name": it.Name,
			"food": it.Food, "steel": it.Steel, "oil": it.Oil, "rare": it.Rare, "gold": it.Gold,
			"res_text": ezfyCorpsMallResText(&it),
			"item_id":  it.ItemId, "item_count": it.ItemCount, "item_name": itemNames[it.ItemId],
			"price": it.Price, "limit": it.LimitCount, "stock": it.Stock, "sold": it.Sold,
			"enabled": it.Enabled, "sort": it.Sort,
		})
	}
	// 我的已购数量（按 user + mall 聚合，供前端显示「已购 x / 限购 y」）
	type boughtRow struct {
		MallId uint
		N      int64
	}
	var brs []boughtRow
	h.DB.Model(&model.EzfyCorpsMallLog{}).Select("mall_id, SUM(`count`) AS n").
		Where("user_id = ?", uid).Group("mall_id").Scan(&brs)
	myBought := gin.H{}
	for _, b := range brs {
		myBought[strconv.FormatUint(uint64(b.MallId), 10)] = b.N
	}
	resp.OK(c, gin.H{"in_corps": true, "my_points": mb.Points, "corps_points": cp.Points,
		"items": items, "my_bought": myBought})
}

// CorpsMallBuy POST /games/ezfy/corps/mall/buy {id, count}
//
// ★ 货币 = 成员个人军团积分；扣分走 DB 原子表达式（带 points >= cost 条件）防并发超扣；
// 资源包「无条件累加」（不做上限截断，参考 ezfy.go 里资源包发放的现行写法）；道具走 addItem。
func (h *EzfyHandler) CorpsMallBuy(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Id    uint `json:"id"`
		Count int  `json:"count"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if req.Count < 1 {
		resp.ParamError(c, "数量错误")
		return
	}
	myCorpsId := h.corpsOfUser(uid)
	if myCorpsId == 0 {
		resp.ParamError(c, "请先加入军团")
		return
	}
	var mb model.EzfyCorpsMember
	if err := h.DB.Where("user_id = ?", uid).First(&mb).Error; err != nil {
		resp.ParamError(c, "请先加入军团")
		return
	}
	var item model.EzfyCorpsMall
	if err := h.DB.First(&item, req.Id).Error; err != nil {
		resp.ParamError(c, "商品不存在")
		return
	}
	if item.Enabled != 1 {
		resp.ParamError(c, "该商品已下架")
		return
	}
	if item.Kind != 1 && item.Kind != 2 {
		resp.ParamError(c, "商品配置错误")
		return
	}
	if item.Kind == 2 && (item.ItemId <= 0 || item.ItemCount <= 0) {
		resp.ParamError(c, "商品配置错误（道具参数缺失）")
		return
	}
	// 限购（LimitCount > 0 时：已购 + count ≤ LimitCount）
	if item.LimitCount > 0 {
		var bought int64
		h.DB.Model(&model.EzfyCorpsMallLog{}).Select("COALESCE(SUM(`count`),0)").
			Where("user_id = ? AND mall_id = ?", uid, item.ID).Scan(&bought)
		if bought+int64(req.Count) > int64(item.LimitCount) {
			resp.ParamError(c, fmt.Sprintf("每人限购 %d 个，你已购买 %d 个", item.LimitCount, bought))
			return
		}
	}
	// 库存（Stock >= 0 时：Sold + count ≤ Stock）
	if item.Stock >= 0 && item.Sold+req.Count > item.Stock {
		resp.ParamError(c, "库存不足")
		return
	}
	cost := item.Price * int64(req.Count)
	// ★ 扣个人军团积分：原子表达式 + points >= cost 条件，防并发超扣
	if cost > 0 {
		res := h.DB.Model(&model.EzfyCorpsMember{}).
			Where("user_id = ? AND points >= ?", uid, cost).
			Update("points", gorm.Expr("points - ?", cost))
		if res.Error != nil {
			resp.ParamError(c, "扣减失败："+res.Error.Error())
			return
		}
		if res.RowsAffected == 0 {
			resp.ParamError(c, fmt.Sprintf("个人军团积分不足（需要 %d，当前 %d）", cost, mb.Points))
			return
		}
	}
	// 扣库存（原子：剩余足够才扣）
	if item.Stock >= 0 {
		sres := h.DB.Model(&model.EzfyCorpsMall{}).
			Where("id = ? AND stock - sold >= ?", item.ID, req.Count).
			Update("sold", gorm.Expr("sold + ?", req.Count))
		if sres.Error != nil || sres.RowsAffected == 0 {
			if cost > 0 {
				h.DB.Model(&model.EzfyCorpsMember{}).Where("user_id = ?", uid).
					Update("points", gorm.Expr("points + ?", cost))
			}
			resp.ParamError(c, "库存不足")
			return
		}
	} else {
		h.DB.Model(&model.EzfyCorpsMall{}).Where("id = ?", item.ID).
			Update("sold", gorm.Expr("sold + ?", req.Count))
	}
	// 发放：资源包 → 无条件原子累加；道具 → 走现有 addItem
	if item.Kind == 1 {
		city := h.getOrCreateCity(uid)
		h.DB.Model(&model.EzfyCity{}).Where("id = ?", city.ID).Updates(map[string]interface{}{
			"food":  gorm.Expr("food + ?", item.Food*int64(req.Count)),
			"steel": gorm.Expr("steel + ?", item.Steel*int64(req.Count)),
			"oil":   gorm.Expr("oil + ?", item.Oil*int64(req.Count)),
			"rare":  gorm.Expr("rare + ?", item.Rare*int64(req.Count)),
			"gold":  gorm.Expr("gold + ?", item.Gold*int64(req.Count)),
		})
	} else {
		h.addItem(uid, item.ItemId, item.ItemCount*req.Count)
	}
	// 写购买记录
	h.DB.Create(&model.EzfyCorpsMallLog{CorpsId: myCorpsId, UserId: uid, MallId: item.ID,
		Kind: item.Kind, Count: req.Count, Cost: cost})
	var left model.EzfyCorpsMember
	h.DB.Where("user_id = ?", uid).First(&left)
	resp.OK(c, gin.H{"msg": fmt.Sprintf("购买成功：%s×%d", item.Name, req.Count),
		"cost": cost, "my_points": left.Points})
}
