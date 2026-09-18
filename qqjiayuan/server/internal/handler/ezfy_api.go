package handler

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 二战风云 HTTP 接口层：建筑/军队/科技/军团/排行/商城/背包/任务/福利/战报/宣战/公告/司令部配置

func (h *EzfyHandler) bodyCity(uid uint, cityId int64) *model.EzfyCity {
	if cityId > 0 {
		if c := h.cityOf(uid, cityId); c != nil {
			return c
		}
	}
	main := h.getOrCreateCity(uid)
	return &main
}

func (h *EzfyHandler) readCityReq(c *gin.Context) (*model.EzfyCity, bool) {
	uid := middleware.GetUID(c)
	var req struct {
		CityId int64 `json:"city_id"`
	}
	_ = c.ShouldBindJSON(&req)
	city := h.bodyCity(uid, req.CityId)
	if city == nil {
		resp.ParamError(c, "城市不存在")
		return nil, false
	}
	return city, true
}

func (h *EzfyHandler) fail(c *gin.Context, msg string) {
	if msg == "" {
		resp.OK(c, gin.H{"msg": "ok"})
		return
	}
	resp.ParamError(c, msg)
}

// ============ 建筑 ============

func (h *EzfyHandler) Buildings(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	city := h.getOrCreateCity(uid)
	h.refreshCity(uid, &city)
	list := h.buildingList(city.ID)
	views := []gin.H{}
	for _, b := range list {
		cfg := ezfyCfg.building(b.BuildingId)
		lv := ezfyCfg.buildingLevel(b.BuildingId, b.Level)
		next := ezfyCfg.buildingLevel(b.BuildingId, b.Level+1)
		view := gin.H{"id": b.ID, "building_id": b.BuildingId, "level": b.Level,
			"status": b.Status, "end_time": b.EndTime, "start_time": b.StartTime}
		if cfg != nil {
			view["name"] = cfg.Name
			view["type"] = cfg.Type
			view["max_level"] = cfg.MaxLevel
			view["des"] = cfg.Des
			view["can_delete"] = cfg.CanDelete
		}
		if lv != nil {
			view["effect"] = lv.Effect
		}
		if next != nil {
			view["next_cost"] = gin.H{"food": next.Food, "steel": next.Steel, "oil": next.Oil, "rare": next.Rare, "gold": next.Gold}
			view["next_time"] = next.BuildTime
			view["next_effect"] = next.Effect
		}
		views = append(views, view)
	}
	resp.OK(c, gin.H{"buildings": views, "area_count": h.areaBuildingCount(city.ID), "area_cap": ezfyMaxBuildings})
}

func (h *EzfyHandler) Build(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		CityId     int64 `json:"city_id"`
		BuildingId int   `json:"building_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	h.cfgs()
	city := h.bodyCity(uid, req.CityId)
	h.fail(c, h.buildBuilding(city, req.BuildingId))
}

func (h *EzfyHandler) Upgrade(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		CityId   int64 `json:"city_id"`
		RecordId int64 `json:"record_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	h.cfgs()
	city := h.bodyCity(uid, req.CityId)
	h.fail(c, h.upgradeBuilding(city, req.RecordId))
}

func (h *EzfyHandler) MaxLevel(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		CityId   int64 `json:"city_id"`
		RecordId int64 `json:"record_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	h.cfgs()
	city := h.bodyCity(uid, req.CityId)
	h.fail(c, h.maxLevelBuilding(city, req.RecordId))
}

func (h *EzfyHandler) DeleteBuilding(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		CityId   int64 `json:"city_id"`
		RecordId int64 `json:"record_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	h.cfgs()
	city := h.bodyCity(uid, req.CityId)
	h.fail(c, h.deleteBuilding(city, req.RecordId))
}

func (h *EzfyHandler) SpeedBuilding(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		CityId   int64 `json:"city_id"`
		RecordId int64 `json:"record_id"`
		Minutes  int64 `json:"minutes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	h.cfgs()
	city := h.bodyCity(uid, req.CityId)
	if req.Minutes <= 0 {
		req.Minutes = 10
	}
	h.fail(c, h.speedUpBuilding(city, req.RecordId, req.Minutes))
}

// ============ 军队 ============

func (h *EzfyHandler) Troops(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	profile := h.ensureProfile(uid)
	city := h.getOrCreateCity(uid)
	h.refreshCity(uid, &city)
	camp := profile.Camp

	troopViews := []gin.H{}
	for tid, count := range h.troopMap(city.ID) {
		cfg := ezfyCfg.troop(tid)
		if cfg == nil {
			continue
		}
		troopViews = append(troopViews, gin.H{"troop_id": tid, "name": ezfyCfg.troopName(tid, camp),
			"count": count, "type": cfg.Type})
	}
	queues := []gin.H{}
	var qs []model.EzfyTrainQueue
	h.DB.Where("city_id = ? AND status = 0", city.ID).Order("start_time ASC").Find(&qs)
	for _, q := range qs {
		queues = append(queues, gin.H{"id": q.ID, "troop_id": q.TroopId,
			"name": ezfyCfg.troopName(q.TroopId, camp), "count": q.Count, "end_time": q.EndTime})
	}
	var wounded, deserters []model.EzfyWounded
	h.DB.Where("city_id = ? AND type = 0", city.ID).Order("troop_id ASC").Find(&wounded)
	h.DB.Where("city_id = ? AND type = 1", city.ID).Order("troop_id ASC").Find(&deserters)
	woundViews := []gin.H{}
	for _, w := range append(wounded, deserters...) {
		woundViews = append(woundViews, gin.H{"id": w.ID, "troop_id": w.TroopId,
			"name": ezfyCfg.troopName(w.TroopId, camp), "type": w.Type, "count": w.Count})
	}
	var popUsed int64
	for tid, count := range h.troopMap(city.ID) {
		if cfg := ezfyCfg.troop(tid); cfg != nil && cfg.Type != 4 {
			popUsed += int64(cfg.Pop) * count
		}
	}
	// 兵种配置一览
	cfgViews := []gin.H{}
	var allTroops []model.EzfyCfgTroop
	h.DB.Order("id ASC").Find(&allTroops)
	for _, t := range allTroops {
		cfgViews = append(cfgViews, gin.H{
			"id": t.ID, "name": ezfyCfg.troopName(t.ID, camp), "type": t.Type,
			"health": t.Health, "defence": t.Defence, "speed": t.Speed, "attack_range": t.AttackRange,
			"carry": t.Carry, "pop": t.Pop, "require": t.Require,
			"cost":       gin.H{"food": t.Food, "steel": t.Steel, "oil": t.Oil, "rare": t.Rare},
			"train_time": t.TrainTime,
		})
	}
	resp.OK(c, gin.H{
		"city": city, "troops": troopViews, "queues": queues, "wounded": woundViews,
		"pop": city.Pop, "pop_used": popUsed, "cfgs": cfgViews,
		"wall_level": h.buildingLevel(city.ID, 7),
	})
}

func (h *EzfyHandler) Train(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		CityId  int64 `json:"city_id"`
		TroopId int   `json:"troop_id"`
		Count   int   `json:"count"`
		Split   bool  `json:"split"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	h.cfgs()
	city := h.bodyCity(uid, req.CityId)
	h.fail(c, h.trainTroop(city, req.TroopId, req.Count, req.Split))
}

func (h *EzfyHandler) RecoverWounded(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		CityId  int64 `json:"city_id"`
		TroopId int   `json:"troop_id"`
		Type    int   `json:"type"`
		All     bool  `json:"all"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	h.cfgs()
	city := h.bodyCity(uid, req.CityId)
	if req.All {
		h.fail(c, h.recoverAllWounded(city, req.Type))
		return
	}
	h.fail(c, h.recoverWounded(city, req.TroopId, req.Type))
}

// ============ 科技 ============

func (h *EzfyHandler) Techs(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	city := h.getOrCreateCity(uid)
	h.refreshCity(uid, &city)
	techMap := h.techMap(city.ID)
	academy := h.buildingLevel(city.ID, 8)
	var all []model.EzfyCfgTech
	h.DB.Order("id ASC").Find(&all)
	views := []gin.H{}
	for _, t := range all {
		level := techMap[t.ID]
		var rec model.EzfyCityTech
		if err := h.DB.Where("city_id = ? AND tech_id = ? AND status = 1", city.ID, t.ID).First(&rec).Error; err == nil {
			views = append(views, gin.H{"tech_id": t.ID, "name": t.Name, "type": t.Type,
				"level": level, "max_level": t.MaxLevel, "des": t.Des, "effect": t.Effect,
				"academy_need": ezfyTechAcademy[t.ID], "academy": academy, "researching": true,
				"end_time": rec.EndTime})
			continue
		}
		next := ezfyCfg.techLevel(t.ID, level+1)
		view := gin.H{"tech_id": t.ID, "name": t.Name, "type": t.Type,
			"level": level, "max_level": t.MaxLevel, "des": t.Des, "effect": t.Effect,
			"academy_need": ezfyTechAcademy[t.ID], "academy": academy, "researching": false}
		if next != nil {
			view["next_cost"] = gin.H{"food": next.Food, "steel": next.Steel, "oil": next.Oil, "rare": next.Rare, "gold": next.Gold}
			view["next_time"] = next.ResearchTime
			view["next_effect"] = next.Effect
		}
		views = append(views, view)
	}
	resp.OK(c, gin.H{"techs": views, "academy": academy})
}

func (h *EzfyHandler) Research(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		CityId int64 `json:"city_id"`
		TechId int   `json:"tech_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	h.cfgs()
	city := h.bodyCity(uid, req.CityId)
	h.fail(c, h.researchTech(city, req.TechId))
}

func (h *EzfyHandler) SpeedTech(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		CityId  int64 `json:"city_id"`
		Minutes int64 `json:"minutes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	h.cfgs()
	city := h.bodyCity(uid, req.CityId)
	if req.Minutes <= 0 {
		req.Minutes = 10
	}
	h.fail(c, h.speedUpTech(city, req.Minutes))
}

// ============ 司令部兵种战斗配置 ============

func (h *EzfyHandler) Targets(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	city := h.getOrCreateCity(uid)
	var list []model.EzfyCityTarget
	h.DB.Where("city_id = ?", city.ID).Find(&list)
	resp.OK(c, gin.H{"targets": list})
}

func (h *EzfyHandler) SaveTarget(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		CityId         int64 `json:"city_id"`
		TroopId        int   `json:"troop_id"`
		AtkTargetTroop int   `json:"atk_target_troop"`
		AtkMove        int   `json:"atk_move"`
		DefTargetTroop int   `json:"def_target_troop"`
		DefMove        int   `json:"def_move"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	h.cfgs()
	city := h.bodyCity(uid, req.CityId)
	if ezfyCfg.troop(req.TroopId) == nil {
		resp.ParamError(c, "兵种不存在")
		return
	}
	var t model.EzfyCityTarget
	if err := h.DB.Where("city_id = ? AND troop_id = ?", city.ID, req.TroopId).First(&t).Error; err != nil {
		t = model.EzfyCityTarget{CityId: int64(city.ID), TroopId: req.TroopId,
			AtkTargetTroop: req.AtkTargetTroop, AtkMove: req.AtkMove,
			DefTargetTroop: req.DefTargetTroop, DefMove: req.DefMove}
		h.DB.Create(&t)
	} else {
		h.DB.Model(&model.EzfyCityTarget{}).Where("id = ?", t.ID).Updates(map[string]interface{}{
			"atk_target_troop": req.AtkTargetTroop, "atk_move": req.AtkMove,
			"def_target_troop": req.DefTargetTroop, "def_move": req.DefMove})
	}
	resp.OK(c, gin.H{"msg": "战斗配置已保存"})
}

// ============ 军团 ============

func (h *EzfyHandler) CorpsList(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	profile := h.ensureProfile(uid)
	var corps []model.EzfyCorps
	h.DB.Order("id DESC").Find(&corps)
	views := []gin.H{}
	for _, cp := range corps {
		score := 0
		var members []model.EzfyCorpsMember
		h.DB.Where("corps_id = ?", cp.ID).Find(&members)
		for _, m := range members {
			p := h.ensureProfile(m.UserId)
			score += p.Prestige
		}
		leaderName := "未知"
		lp := h.ensureProfile(cp.LeaderUserId)
		if lp.UserID == cp.LeaderUserId {
			leaderName = lp.Nickname
		}
		views = append(views, gin.H{"id": cp.ID, "name": cp.Name, "notice": cp.Notice,
			"member_count": cp.MemberCount, "leader": leaderName, "battle_score": score,
			"camp": profile.Camp})
	}
	var myCorps *model.EzfyCorps
	var myMember model.EzfyCorpsMember
	if err := h.DB.Where("user_id = ?", uid).First(&myMember).Error; err == nil {
		var cp model.EzfyCorps
		if err := h.DB.First(&cp, myMember.CorpsId).Error; err == nil {
			myCorps = &cp
		}
	}
	resp.OK(c, gin.H{"corps": views, "my_corps": myCorps})
}

func (h *EzfyHandler) CorpsCreate(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	name := trimSpace(req.Name)
	if name == "" {
		resp.ParamError(c, "请输入军团名")
		return
	}
	if len([]rune(name)) > 10 {
		resp.ParamError(c, "军团名过长(限10字)")
		return
	}
	var exist model.EzfyCorpsMember
	if err := h.DB.Where("user_id = ?", uid).First(&exist).Error; err == nil {
		resp.ParamError(c, "你已在军团中")
		return
	}
	var count int64
	h.DB.Model(&model.EzfyCorps{}).Where("name = ?", name).Count(&count)
	if count > 0 {
		resp.ParamError(c, "军团名已存在")
		return
	}
	// 联络中心: 2 级才能创建联盟, 并消耗黄金
	city := h.getOrCreateCity(uid)
	h.refreshCity(uid, &city)
	liaison := h.buildingLevel(city.ID, ezfyBuildingLiaison)
	if liaison < 2 {
		resp.ParamError(c, "需要 2 级联络中心才能创建联盟(当前"+strconv.Itoa(liaison)+"级)")
		return
	}
	if city.Gold < ezfyCorpsCreateGold {
		resp.ParamError(c, "创建联盟需要"+strconv.Itoa(ezfyCorpsCreateGold)+"黄金")
		return
	}
	city.Gold -= ezfyCorpsCreateGold
	h.saveCityRes(&city)
	cp := model.EzfyCorps{Name: name, LeaderUserId: uid, Notice: "", MemberCount: 1}
	h.DB.Create(&cp)
	h.DB.Create(&model.EzfyCorpsMember{CorpsId: cp.ID, UserId: uid, IsLeader: 1, Title: "军团长"})
	resp.OK(c, gin.H{"msg": "军团创建成功"})
}

func (h *EzfyHandler) CorpsJoin(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		CorpsId uint `json:"corps_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	var exist model.EzfyCorpsMember
	if err := h.DB.Where("user_id = ?", uid).First(&exist).Error; err == nil {
		resp.ParamError(c, "你已在军团中")
		return
	}
	var cp model.EzfyCorps
	if err := h.DB.First(&cp, req.CorpsId).Error; err != nil {
		resp.ParamError(c, "军团不存在")
		return
	}
	// 联络中心: 1 级才能加入联盟, 且受人数上限限制
	if h.liaisonLevel(uid) < 1 {
		resp.ParamError(c, "需要 1 级联络中心才能加入联盟")
		return
	}
	var memberCount int64
	h.DB.Model(&model.EzfyCorpsMember{}).Where("corps_id = ?", cp.ID).Count(&memberCount)
	if cap := h.corpsMemberCap(cp.ID); int(memberCount) >= cap {
		resp.ParamError(c, "该联盟人数已满("+strconv.Itoa(int(memberCount))+"/"+strconv.Itoa(cap)+")")
		return
	}
	h.DB.Create(&model.EzfyCorpsMember{CorpsId: cp.ID, UserId: uid, IsLeader: 0, Title: "成员"})
	h.DB.Model(&model.EzfyCorps{}).Where("id = ?", cp.ID).Update("member_count", cp.MemberCount+1)
	resp.OK(c, gin.H{"msg": "加入军团成功"})
}

func (h *EzfyHandler) CorpsLeave(c *gin.Context) {
	uid := middleware.GetUID(c)
	var mb model.EzfyCorpsMember
	if err := h.DB.Where("user_id = ?", uid).First(&mb).Error; err != nil {
		resp.ParamError(c, "你不在任何军团中")
		return
	}
	var cp model.EzfyCorps
	_ = h.DB.First(&cp, mb.CorpsId).Error
	h.DB.Delete(&mb)
	if mb.IsLeader == 1 {
		// 军团长退出 = 解散
		h.DB.Where("corps_id = ?", mb.CorpsId).Delete(&model.EzfyCorpsMember{})
		h.DB.Where("corps_id = ?", mb.CorpsId).Delete(&model.EzfyCorpsChat{})
		h.DB.Delete(&cp)
		resp.OK(c, gin.H{"msg": "军团长退出, 军团已解散"})
		return
	}
	h.DB.Model(&model.EzfyCorps{}).Where("id = ?", cp.ID).
		Update("member_count", maxInt(0, cp.MemberCount-1))
	resp.OK(c, gin.H{"msg": "已退出军团"})
}

func (h *EzfyHandler) CorpsKick(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		UserId uint `json:"user_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	var leader model.EzfyCorpsMember
	if err := h.DB.Where("user_id = ?", uid).First(&leader).Error; err != nil || leader.IsLeader != 1 {
		resp.ParamError(c, "只有军团长能踢人")
		return
	}
	if uid == req.UserId {
		resp.ParamError(c, "不能踢自己(军团长退出即解散)")
		return
	}
	var mb model.EzfyCorpsMember
	if err := h.DB.Where("user_id = ? AND corps_id = ?", req.UserId, leader.CorpsId).First(&mb).Error; err != nil {
		resp.ParamError(c, "该成员不在你的军团")
		return
	}
	h.DB.Delete(&mb)
	var cp model.EzfyCorps
	_ = h.DB.First(&cp, leader.CorpsId).Error
	h.DB.Model(&model.EzfyCorps{}).Where("id = ?", cp.ID).
		Update("member_count", maxInt(0, cp.MemberCount-1))
	resp.OK(c, gin.H{"msg": "已踢出成员"})
}

func (h *EzfyHandler) CorpsNotice(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Notice string `json:"notice"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	var leader model.EzfyCorpsMember
	if err := h.DB.Where("user_id = ?", uid).First(&leader).Error; err != nil || leader.IsLeader != 1 {
		resp.ParamError(c, "只有军团长能修改公告")
		return
	}
	notice := trimSpace(req.Notice)
	if len([]rune(notice)) > 200 {
		r := []rune(notice)
		notice = string(r[:200])
	}
	h.DB.Model(&model.EzfyCorps{}).Where("id = ?", leader.CorpsId).Update("notice", notice)
	resp.OK(c, gin.H{"msg": "公告已更新"})
}

func (h *EzfyHandler) CorpsChats(c *gin.Context) {
	uid := middleware.GetUID(c)
	var mb model.EzfyCorpsMember
	if err := h.DB.Where("user_id = ?", uid).First(&mb).Error; err != nil {
		resp.OK(c, gin.H{"chats": []gin.H{}, "in_corps": false})
		return
	}
	var chats []model.EzfyCorpsChat
	h.DB.Where("corps_id = ?", mb.CorpsId).Order("id DESC").Limit(50).Find(&chats)
	views := []gin.H{}
	for i := len(chats) - 1; i >= 0; i-- {
		ch := chats[i]
		views = append(views, gin.H{"id": ch.ID, "user_name": ch.UserName,
			"content": ch.Content, "created_at": ch.CreatedAt, "mine": ch.UserId == uid})
	}
	resp.OK(c, gin.H{"chats": views, "in_corps": true})
}

func (h *EzfyHandler) CorpsChat(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	var mb model.EzfyCorpsMember
	if err := h.DB.Where("user_id = ?", uid).First(&mb).Error; err != nil {
		resp.ParamError(c, "你不在任何军团中")
		return
	}
	content := trimSpace(req.Content)
	if content == "" {
		resp.ParamError(c, "消息为空")
		return
	}
	if len([]rune(content)) > 200 {
		r := []rune(content)
		content = string(r[:200])
	}
	profile := h.ensureProfile(uid)
	h.DB.Create(&model.EzfyCorpsChat{CorpsId: mb.CorpsId, UserId: uid,
		UserName: profile.Nickname, Content: content})
	resp.OK(c, gin.H{"msg": "发送成功"})
}

// ============ 宣战 ============

func (h *EzfyHandler) DeclareWar(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		TargetUserId uint  `json:"target_user_id"`
		CityId       int64 `json:"city_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if req.TargetUserId == 0 && req.CityId > 0 {
		var tc model.EzfyCity
		if err := h.DB.First(&tc, req.CityId).Error; err == nil {
			req.TargetUserId = tc.UserID
		}
	}
	if req.TargetUserId == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	if req.TargetUserId == uid {
		resp.ParamError(c, "不能对自己宣战")
		return
	}
	if h.getWar(uid, req.TargetUserId) != nil {
		resp.ParamError(c, "已与该玩家宣战(待生效或交战中)")
		return
	}
	now := time.Now().UnixMilli()
	w := model.EzfyWar{AtkUserId: uid, DefUserId: req.TargetUserId, Status: 1,
		DeclareTime: now, EffectTime: now + ezfyWarDelayHours*3600000,
		ExpireTime: now + (ezfyWarDelayHours+ezfyWarDurationHours)*3600000}
	h.DB.Create(&w)
	resp.OK(c, gin.H{"msg": fmt.Sprintf("宣战成功, %d小时后生效, 生效后%d小时内可互相掠夺/征服", ezfyWarDelayHours, ezfyWarDurationHours)})
}

func (h *EzfyHandler) WarStatus(c *gin.Context) {
	uid := middleware.GetUID(c)
	targetUserId, _ := strconv.ParseUint(c.Query("target_user_id"), 10, 64)
	tid := uint(targetUserId)
	status := h.warStatus(uid, tid)
	text := "未宣战"
	if w := h.getWar(uid, tid); w != nil {
		now := time.Now().UnixMilli()
		if status == 1 {
			hs := (w.EffectTime - now + 3599999) / 3600000
			text = fmt.Sprintf("宣战中(约%d小时后开战)", maxInt64(1, hs))
		} else if status == 2 {
			hs := (w.ExpireTime - now + 3599999) / 3600000
			text = fmt.Sprintf("交战中(剩余约%d小时)", maxInt64(0, hs))
		}
	}
	resp.OK(c, gin.H{"status": status, "text": text})
}

// ============ 排行榜 ============

func (h *EzfyHandler) Rank(c *gin.Context) {
	h.cfgs()
	// 声望榜
	var profiles []model.EzfyProfile
	h.DB.Order("prestige DESC").Limit(20).Find(&profiles)
	prestigeRank := []gin.H{}
	for i, p := range profiles {
		prestigeRank = append(prestigeRank, gin.H{"rank": i + 1, "name": p.Nickname,
			"prestige": p.Prestige, "rank_name": ezfyRankName(p.Prestige)})
	}
	// 兵力榜(不含城防)
	var troops []model.EzfyCityTroop
	h.DB.Where("troop_id < 17").Find(&troops)
	sumByCity := map[int64]int64{}
	for _, t := range troops {
		sumByCity[t.CityId] += t.Count
	}
	type kv struct {
		k int64
		v int64
	}
	arr := []kv{}
	for k, v := range sumByCity {
		arr = append(arr, kv{k, v})
	}
	for i := 0; i < len(arr); i++ {
		for j := i + 1; j < len(arr); j++ {
			if arr[j].v > arr[i].v {
				arr[i], arr[j] = arr[j], arr[i]
			}
		}
	}
	troopRank := []gin.H{}
	for i, e := range arr {
		if i >= 20 {
			break
		}
		var city model.EzfyCity
		if err := h.DB.First(&city, e.k).Error; err != nil {
			continue
		}
		p := h.ensureProfile(city.UserID)
		troopRank = append(troopRank, gin.H{"rank": i + 1, "city_name": city.Name,
			"role_name": p.Nickname, "count": e.v})
	}
	// 军团榜
	var corps []model.EzfyCorps
	h.DB.Order("member_count DESC").Limit(20).Find(&corps)
	corpsRank := []gin.H{}
	for i, cp := range corps {
		score := 0
		var members []model.EzfyCorpsMember
		h.DB.Where("corps_id = ?", cp.ID).Find(&members)
		for _, m := range members {
			score += h.ensureProfile(m.UserId).Prestige
		}
		corpsRank = append(corpsRank, gin.H{"rank": i + 1, "name": cp.Name, "member_count": cp.MemberCount, "battle_score": score})
	}
	// 军衔表
	ranks := []gin.H{}
	for _, r := range ezfyRanks {
		need, _ := strconv.Atoi(r[2])
		ranks = append(ranks, gin.H{"name": r[0], "post": r[1], "need": need})
	}
	resp.OK(c, gin.H{"prestige": prestigeRank, "troops": troopRank, "corps": corpsRank, "ranks": ranks})
}

// ============ 商城/背包 ============

func (h *EzfyHandler) Mall(c *gin.Context) {
	h.cfgs()
	var items []model.EzfyCfgItem
	h.DB.Order("id ASC").Find(&items)
	resp.OK(c, gin.H{"items": items})
}

func (h *EzfyHandler) Buy(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		CityId int64 `json:"city_id"`
		CfgId  int   `json:"cfg_id"`
		Count  int   `json:"count"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	h.cfgs()
	if req.Count <= 0 {
		resp.ParamError(c, "数量错误")
		return
	}
	cfg := ezfyCfg.item(req.CfgId)
	if cfg == nil {
		resp.ParamError(c, "道具不存在")
		return
	}
	cost := cfg.PriceGold * int64(req.Count)
	city := h.bodyCity(uid, req.CityId)
	if city.Gold < cost {
		resp.ParamError(c, "黄金不足")
		return
	}
	city.Gold -= cost
	h.saveCityRes(city)
	h.addItem(uid, req.CfgId, req.Count)
	resp.OK(c, gin.H{"msg": fmt.Sprintf("购买成功: %s×%d", cfg.Name, req.Count)})
}

func (h *EzfyHandler) Bag(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	var items []model.EzfyItem
	h.DB.Where("user_id = ?", uid).Find(&items)
	views := []gin.H{}
	for _, it := range items {
		if it.Count <= 0 {
			continue
		}
		cfg := ezfyCfg.item(it.CfgId)
		if cfg == nil {
			continue
		}
		views = append(views, gin.H{"cfg_id": it.CfgId, "count": it.Count,
			"name": cfg.Name, "item_type": cfg.ItemType, "description": cfg.Description, "param1": cfg.Param1})
	}
	// 军官类道具的目标选择需要军官列表与技能列表
	city := h.getOrCreateCity(uid)
	officers := []gin.H{}
	for _, o := range h.officerList(city.ID) {
		officers = append(officers, gin.H{"id": o.ID, "name": o.Name, "level": o.Level,
			"military": o.Military, "logistics": o.Logistics, "learning": o.Learning,
			"status": o.Status, "status_name": ezfyOfficerStatusName(&o),
			"is_captive": o.IsCaptive, "skills": officerSkills(&o)})
	}
	skills := []gin.H{}
	for _, s := range ezfyCfg.skills {
		skills = append(skills, gin.H{"id": s.ID, "name": s.Name, "effect": s.Effect})
	}
	sort.Slice(skills, func(i, j int) bool { return skills[i]["id"].(int) < skills[j]["id"].(int) })
	resp.OK(c, gin.H{"items": views, "officers": officers, "skills": skills})
}

func (h *EzfyHandler) UseItem(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		CityId    int64 `json:"city_id"`
		CfgId     int   `json:"cfg_id"`
		Count     int   `json:"count"`
		OfficerId int64 `json:"officer_id"`
		SkillId   int   `json:"skill_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	h.cfgs()
	city := h.bodyCity(uid, req.CityId)
	msg := h.useItem(uid, city, req.CfgId, req.Count, req.OfficerId, req.SkillId)
	if msg == "" {
		resp.OK(c, gin.H{"msg": "ok"})
		return
	}
	if strings.HasPrefix(msg, "使用成功") {
		resp.OK(c, gin.H{"msg": msg})
		return
	}
	resp.ParamError(c, msg)
}

// ============ 任务 ============

func (h *EzfyHandler) Tasks(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	h.initTasks(uid)
	h.resetDailyTasks(uid)
	today := time.Now().Format("2006-01-02")
	var cfgs []model.EzfyCfgTask
	h.DB.Where("status = 1").Order("sort_no ASC").Find(&cfgs)
	var mine []model.EzfyTask
	h.DB.Where("user_id = ?", uid).Find(&mine)
	myMap := map[int]model.EzfyTask{}
	for _, t := range mine {
		myMap[t.CfgId] = t
	}
	// 状态型任务同步
	for _, cfg := range cfgs {
		if !ezfyStateTaskTypes[cfg.TaskType] {
			continue
		}
		t, ok := myMap[cfg.ID]
		if !ok || t.Status == 2 {
			continue
		}
		cur := h.calcStateValue(uid, cfg.TaskType)
		if cur != t.Current {
			setCur := cur
			if setCur > cfg.Target {
				setCur = cfg.Target
			}
			status := 0
			if setCur >= cfg.Target {
				status = 1
			}
			h.DB.Model(&model.EzfyTask{}).Where("id = ?", t.ID).
				Updates(map[string]interface{}{"current": setCur, "status": status})
		}
	}
	// 分组
	var types []model.EzfyCfgTaskType
	h.DB.Order("sort_no ASC").Find(&types)
	typeMap := map[int]model.EzfyCfgTaskType{}
	for _, tp := range types {
		typeMap[tp.ID] = tp
	}
	groups := []gin.H{}
	byType := map[int][]gin.H{}
	order := []int{}
	for _, cfg := range cfgs {
		t, ok := myMap[cfg.ID]
		if !ok {
			continue
		}
		row := gin.H{"id": t.ID, "cfg_id": cfg.ID, "name": cfg.Name, "target": cfg.Target,
			"current": t.Current, "status": t.Status,
			"reward": gin.H{"gold": cfg.RewardGold, "food": cfg.RewardFood, "steel": cfg.RewardSteel,
				"oil": cfg.RewardOil, "rare": cfg.RewardRare, "prestige": cfg.RewardPrestige}}
		typeId := cfg.TypeId
		if _, exists := byType[typeId]; !exists {
			order = append(order, typeId)
		}
		byType[typeId] = append(byType[typeId], row)
	}
	for _, typeId := range order {
		tp, ok := typeMap[typeId]
		if !ok {
			continue
		}
		groups = append(groups, gin.H{"id": tp.ID, "name": tp.Name, "reset_type": tp.ResetType, "tasks": byType[typeId]})
	}
	_ = today
	resp.OK(c, gin.H{"groups": groups})
}

func (h *EzfyHandler) TaskAward(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		TaskId int64 `json:"task_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	h.cfgs()
	h.fail(c, h.taskAward(uid, req.TaskId))
}

// ============ 福利(签到/礼包) ============

var ezfySignRewards = [7][6]int64{
	{500, 5000, 0, 0, 0, 0},
	{500, 0, 5000, 0, 0, 0},
	{1000, 0, 0, 3000, 0, 0},
	{1000, 0, 0, 0, 2000, 0},
	{2000, 10000, 0, 0, 0, 0},
	{2000, 0, 8000, 0, 0, 0},
	{5000, 5000, 5000, 5000, 5000, 100},
}

func (h *EzfyHandler) giveResources(uid uint, food, steel, oil, rare, gold int64) {
	city := h.getOrCreateCity(uid)
	city.Food = min64(city.FoodCap, city.Food+food)
	city.Steel = min64(city.SteelCap, city.Steel+steel)
	city.Oil = min64(city.OilCap, city.Oil+oil)
	city.Rare = min64(city.RareCap, city.Rare+rare)
	city.Gold = min64(city.GoldCap, city.Gold+gold)
	h.saveCityRes(&city)
}

func (h *EzfyHandler) Welfare(c *gin.Context) {
	uid := middleware.GetUID(c)
	today := time.Now().Format("2006-01-02")
	yest := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	var signedToday, signedYest int64
	h.DB.Model(&model.EzfySign{}).Where("user_id = ? AND sign_date = ?", uid, today).Count(&signedToday)
	h.DB.Model(&model.EzfySign{}).Where("user_id = ? AND sign_date = ?", uid, yest).Count(&signedYest)
	signCount := 1
	if signedYest > 0 {
		var s model.EzfySign
		if err := h.DB.Where("user_id = ? AND sign_date = ?", uid, yest).First(&s).Error; err == nil {
			signCount = s.SignCount + 1
		}
	}
	rewards := []gin.H{}
	for i, r := range ezfySignRewards {
		text := ""
		if r[0] > 0 {
			text += fmt.Sprintf("黄金%d ", r[0])
		}
		if r[1] > 0 {
			text += fmt.Sprintf("粮食%d ", r[1])
		}
		if r[2] > 0 {
			text += fmt.Sprintf("钢铁%d ", r[2])
		}
		if r[3] > 0 {
			text += fmt.Sprintf("石油%d ", r[3])
		}
		if r[4] > 0 {
			text += fmt.Sprintf("稀矿%d ", r[4])
		}
		if r[5] > 0 {
			text += fmt.Sprintf("声望%d", r[5])
		}
		rewards = append(rewards, gin.H{"day": i + 1, "reward": text})
	}
	// 礼包状态
	gifts := gin.H{}
	for _, t := range []string{"newbie", "weekly", "level10", "level20", "level30", "level40"} {
		var n int64
		h.DB.Model(&model.EzfyGift{}).Where("user_id = ? AND gift_type = ?", uid, t).Count(&n)
		gifts[t] = n > 0
	}
	city := h.getOrCreateCity(uid)
	profile := h.ensureProfile(uid)
	resp.OK(c, gin.H{
		"signed_today": signedToday > 0, "sign_count": signCount,
		"rewards": rewards, "gifts": gifts, "city_level": city.CityLevel,
		"prestige": profile.Prestige, "rank_name": ezfyRankName(profile.Prestige),
	})
}

func (h *EzfyHandler) Sign(c *gin.Context) {
	uid := middleware.GetUID(c)
	today := time.Now().Format("2006-01-02")
	var exist model.EzfySign
	if err := h.DB.Where("user_id = ? AND sign_date = ?", uid, today).First(&exist).Error; err == nil {
		resp.ParamError(c, "今天已经签到过了")
		return
	}
	yest := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	count := 1
	var y model.EzfySign
	if err := h.DB.Where("user_id = ? AND sign_date = ?", uid, yest).First(&y).Error; err == nil {
		count = y.SignCount + 1
	}
	h.DB.Create(&model.EzfySign{UserId: uid, SignDate: today, SignCount: count})
	r := ezfySignRewards[(count-1)%7]
	h.giveResources(uid, r[1], r[2], r[3], r[4], r[0])
	if r[5] > 0 {
		h.addPrestige(uid, int(r[5]))
	}
	resp.OK(c, gin.H{"msg": fmt.Sprintf("签到成功(连续%d天), 奖励已发放", count)})
}

func (h *EzfyHandler) Gift(c *gin.Context) {
	uid := middleware.GetUID(c)
	giftType := c.Param("type")
	hasGift := func(t string) bool {
		var n int64
		h.DB.Model(&model.EzfyGift{}).Where("user_id = ? AND gift_type = ?", uid, t).Count(&n)
		return n > 0
	}
	recordGift := func(t string) {
		h.DB.Create(&model.EzfyGift{UserId: uid, GiftType: t})
	}
	switch giftType {
	case "newbie":
		if hasGift("newbie") {
			resp.ParamError(c, "新手礼包已领取")
			return
		}
		h.giveResources(uid, 50000, 30000, 20000, 10000, 5000)
		recordGift("newbie")
		resp.OK(c, gin.H{"msg": "新手礼包领取成功"})
	case "weekly":
		var last model.EzfyGift
		if err := h.DB.Where("user_id = ? AND gift_type = ?", uid, "weekly").Order("id DESC").First(&last).Error; err == nil {
			if last.CreatedAt.Format("2006-01") == time.Now().Format("2006-01") &&
				sameWeek(last.CreatedAt, time.Now()) {
				resp.ParamError(c, "本周福利已领取")
				return
			}
		}
		h.giveResources(uid, 20000, 20000, 20000, 20000, 2000)
		recordGift("weekly")
		resp.OK(c, gin.H{"msg": "每周福利领取成功"})
	case "level10", "level20", "level30", "level40":
		needLevel, gold, res := 0, int64(0), int64(0)
		switch giftType {
		case "level10":
			needLevel, gold, res = 10, 5000, 50000
		case "level20":
			needLevel, gold, res = 20, 10000, 100000
		case "level30":
			needLevel, gold, res = 30, 20000, 200000
		case "level40":
			needLevel, gold, res = 40, 50000, 500000
		}
		if hasGift(giftType) {
			resp.ParamError(c, "该礼包已领取")
			return
		}
		city := h.getOrCreateCity(uid)
		if city.CityLevel < needLevel {
			resp.ParamError(c, fmt.Sprintf("市政厅需要达到%d级", needLevel))
			return
		}
		h.giveResources(uid, res, res*3/5, res*2/5, res/5, gold)
		recordGift(giftType)
		resp.OK(c, gin.H{"msg": "礼包领取成功"})
	default:
		resp.ParamError(c, "礼包类型错误")
	}
}

func sameWeek(a, b time.Time) bool {
	ya, wa := a.ISOWeek()
	yb, wb := b.ISOWeek()
	return ya == yb && wa == wb
}

// ============ 公告 ============

func (h *EzfyHandler) Notices(c *gin.Context) {
	uid := middleware.GetUID(c)
	var notices []model.EzfyNotice
	h.DB.Where("user_id = 0 OR user_id = ?", uid).Order("is_top DESC, id DESC").Limit(30).Find(&notices)
	resp.OK(c, gin.H{"notices": notices})
}

// ============ 战报 ============

func (h *EzfyHandler) Reports(c *gin.Context) {
	uid := middleware.GetUID(c)
	var reports []model.EzfyReport
	h.DB.Where("user_id = ?", uid).Order("id DESC").Limit(50).Find(&reports)
	views := []gin.H{}
	typeName := map[int]string{1: "侦察", 2: "掠夺", 3: "征服", 4: "战斗", 5: "采集", 6: "系统"}
	for _, r := range reports {
		views = append(views, gin.H{"id": r.ID, "title": r.Title, "report_type": r.ReportType,
			"type_name": typeName[r.ReportType], "is_read": r.IsRead, "order_id": r.OrderId,
			"created_at": r.CreatedAt})
		if r.IsRead == 0 {
			h.DB.Model(&model.EzfyReport{}).Where("id = ?", r.ID).Update("is_read", 1)
		}
	}
	resp.OK(c, gin.H{"reports": views})
}

func (h *EzfyHandler) ReportView(c *gin.Context) {
	uid := middleware.GetUID(c)
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var r model.EzfyReport
	if err := h.DB.Where("id = ? AND user_id = ?", id, uid).First(&r).Error; err != nil {
		resp.NotFound(c, "战报不存在")
		return
	}
	h.DB.Model(&model.EzfyReport{}).Where("id = ?", r.ID).Update("is_read", 1)
	resp.OK(c, gin.H{"report": r})
}
