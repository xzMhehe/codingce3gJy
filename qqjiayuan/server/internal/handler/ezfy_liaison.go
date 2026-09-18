package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 二战风云 联络中心（复刻原版 liaison/liaisonIndex.html 的设计描述）
//
// 原版页面文案：
//   "联络中心是盟友间互相联络的建筑
//    1级联络中心可以 加入联盟，2级联络中心可以 创建联盟
//    创建联盟需消耗50钻石
//    每级联络中心可以多一支盟友驻军、多10人联盟人数上限"
//
// 注：原 Java 工程此页是**纯静态文案**，后端无任何逻辑（任务.md 也标为未完成）。
// 这里按文案把规则真正实现；钻石体系本项目未做，按项目惯例用黄金代替（同「城市迁移」）。

const (
	ezfyBuildingLiaison  = 15    // 联络中心
	ezfyCorpsMemberPerLv = 10    // 每级联络中心 +10 人联盟人数上限
	ezfyCorpsCreateGold  = 50000 // 创建联盟消耗(原版 50 钻石, 以 5 万黄金代替)
)

// liaisonLevel 某玩家主城的联络中心等级
func (h *EzfyHandler) liaisonLevel(uid uint) int {
	city := h.getOrCreateCity(uid)
	return h.buildingLevel(city.ID, ezfyBuildingLiaison)
}

// corpsMemberCap 联盟人数上限 = 军团长所在城市联络中心等级 × 10（至少 10）
func (h *EzfyHandler) corpsMemberCap(corpsId uint) int {
	var cp model.EzfyCorps
	if err := h.DB.First(&cp, corpsId).Error; err != nil {
		return 0
	}
	level := h.liaisonLevel(cp.LeaderUserId)
	if level < 1 {
		level = 1
	}
	return level * ezfyCorpsMemberPerLv
}

// allyGarrisonCap 盟友驻军上限 = 本城联络中心等级
func (h *EzfyHandler) allyGarrisonCap(cityId uint) int {
	return h.buildingLevel(cityId, ezfyBuildingLiaison)
}

// allyGarrisonCount 该城市当前已接收的盟友驻军队伍数（增援命令常驻中）
func (h *EzfyHandler) allyGarrisonCount(cityId uint) int {
	var n int64
	h.DB.Model(&model.EzfyOrder{}).
		Where("target_id = ? AND target_type = 3 AND order_type = 6 AND status = 3", cityId).
		Count(&n)
	return int(n)
}

// sameCorps 两人是否在同一军团
func (h *EzfyHandler) sameCorps(a, b uint) bool {
	var ma, mb model.EzfyCorpsMember
	if err := h.DB.Where("user_id = ?", a).First(&ma).Error; err != nil {
		return false
	}
	if err := h.DB.Where("user_id = ?", b).First(&mb).Error; err != nil {
		return false
	}
	return ma.CorpsId == mb.CorpsId
}

// Liaison GET /games/ezfy/liaison —— 联络中心信息
func (h *EzfyHandler) Liaison(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	city := h.getOrCreateCity(uid)
	h.refreshCity(uid, &city)

	level := h.buildingLevel(city.ID, ezfyBuildingLiaison)
	out := gin.H{
		"level": level, "can_join": level >= 1, "can_create": level >= 2,
		"create_cost":      ezfyCorpsCreateGold,
		"member_per_level": ezfyCorpsMemberPerLv,
		"my_corps":         nil, "member_count": 0, "member_cap": 0,
		"garrison_cap": level, "garrison_used": h.allyGarrisonCount(city.ID),
		"garrisons": []gin.H{},
	}

	// 我的联盟
	if cp := h.myCorpsOf(uid); cp != nil {
		var count int64
		h.DB.Model(&model.EzfyCorpsMember{}).Where("corps_id = ?", cp.ID).Count(&count)
		out["my_corps"] = gin.H{"id": cp.ID, "name": cp.Name, "notice": cp.Notice,
			"leader_user_id": cp.LeaderUserId}
		out["member_count"] = count
		out["member_cap"] = h.corpsMemberCap(cp.ID)
	}

	// 盟军驻军: 别人增援到我城的常驻部队
	var orders []model.EzfyOrder
	h.DB.Where("target_id = ? AND target_type = 3 AND order_type = 6 AND status = 3", city.ID).
		Order("id DESC").Limit(50).Find(&orders)
	views := make([]gin.H, 0, len(orders))
	for _, o := range orders {
		var from model.EzfyCity
		_ = h.DB.First(&from, o.CityId).Error
		troops := []gin.H{}
		for _, g := range parseGroups(o.Troops) {
			name := "兵种" + strconv.Itoa(g.TroopId)
			if cfg := ezfyCfg.troop(g.TroopId); cfg != nil {
				name = cfg.Name
			}
			troops = append(troops, gin.H{"name": name, "count": g.Count})
		}
		views = append(views, gin.H{"id": o.ID, "from_city": from.Name, "from_user_id": from.UserID,
			"officer": o.Officer, "troops": troops})
	}
	out["garrisons"] = views
	resp.OK(c, out)
}
