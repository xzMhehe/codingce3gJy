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

// ============ 军团职位（第九轮：军团长可任命副团长 / 参谋长） ============

const (
	ezfyCorpsTitleVice  = "副团长" // 可发军团邮件
	ezfyCorpsTitleChief = "参谋长" // 荣誉职位
)

// ezfyValidCorpsTitle 合法职位（空字符串 = 普通成员）
func ezfyValidCorpsTitle(t string) bool {
	return t == "" || t == ezfyCorpsTitleVice || t == ezfyCorpsTitleChief
}

// ezfyCanMailCorps 是否有发军团邮件的权限（军团长 / 副团长）
func ezfyCanMailCorps(cp *model.EzfyCorps, mb *model.EzfyCorpsMember, uid uint) bool {
	if cp == nil {
		return false
	}
	if cp.LeaderUserId == uid {
		return true
	}
	return mb != nil && mb.Title == ezfyCorpsTitleVice
}

// CorpsSetTitle POST /games/ezfy/corps/member/title  { user_id, title }
//
// 用户规则：军团长可以给军团成员任职（副团长、参谋长）。
// title 传空字符串表示撤销职位。只有军团长能操作。
func (h *EzfyHandler) CorpsSetTitle(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		UserId uint   `json:"user_id"`
		Title  string `json:"title"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.UserId == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	title := trimSpace(req.Title)
	if !ezfyValidCorpsTitle(title) {
		resp.ParamError(c, "职位只能是「副团长」「参谋长」或留空撤销")
		return
	}
	cp := h.myCorpsOf(uid)
	if cp == nil {
		resp.ParamError(c, "你还不在任何军团中")
		return
	}
	if cp.LeaderUserId != uid {
		resp.Forbidden(c, "只有军团长可以任命军团职位")
		return
	}
	if req.UserId == uid {
		resp.ParamError(c, "军团长不需要任命自己")
		return
	}
	var mb model.EzfyCorpsMember
	if err := h.DB.Where("user_id = ? AND corps_id = ?", req.UserId, cp.ID).First(&mb).Error; err != nil {
		resp.ParamError(c, "该玩家不在你的军团中")
		return
	}
	if err := h.DB.Model(&model.EzfyCorpsMember{}).Where("id = ?", mb.ID).
		Update("title", title).Error; err != nil {
		resp.ParamError(c, "任命失败：" + err.Error())
		return
	}
	p := h.ensureProfile(req.UserId)
	var u model.User
	h.DB.First(&u, req.UserId)
	name := ezfyNickOf(p, &u)
	if name == "" {
		name = strconv.Itoa(int(req.UserId))
	}
	msg := "已任命「" + name + "」为" + title
	if title == "" {
		msg = "已撤销「" + name + "」的军团职位"
	} else {
		h.DB.Create(&model.EzfyNotice{UserId: req.UserId, Title: "军团任命", Content: msg})
	}
	resp.OK(c, gin.H{"msg": msg, "title": title})
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

// CorpsMail POST /games/ezfy/corps/mail —— 军团长给全体成员群发邮件
// 复刻 CorpsController.mail + CorpsServiceImpl.sendCorpsMail：
// 只有军团长能发、内容 500 字以内、给军团每个成员各写一封私信(Letter)。
func (h *EzfyHandler) CorpsMail(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	var req struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	content := trimSpace(req.Content)
	if content == "" {
		resp.ParamError(c, "邮件内容为空")
		return
	}
	if len([]rune(content)) > 500 {
		content = string([]rune(content)[:500])
	}
	cp := h.myCorpsOf(uid)
	if cp == nil {
		resp.ParamError(c, "你还不在任何军团中")
		return
	}
	// ★ 第九轮：军团长与**副团长**都能发军团邮件
	var mb model.EzfyCorpsMember
	h.DB.Where("user_id = ?", uid).First(&mb)
	if !ezfyCanMailCorps(cp, &mb, uid) {
		resp.Forbidden(c, "只有军团长或副团长可以发军团邮件")
		return
	}
	var members []model.EzfyCorpsMember
	h.DB.Where("corps_id = ?", cp.ID).Find(&members)
	sent := 0
	for _, m := range members {
		if m.UserId == 0 || m.UserId == uid {
			continue // 不给自己发
		}
		var u model.User
		if err := h.DB.First(&u, m.UserId).Error; err != nil {
			continue
		}
		h.DB.Create(&model.PrivateMessage{SenderID: uid, ReceiverID: m.UserId, Content: content})
		sent++
	}
	if sent == 0 {
		resp.ParamError(c, "军团没有可通知的成员")
		return
	}
	resp.OK(c, gin.H{"msg": "军团邮件已发送给 " + strconv.Itoa(sent) + " 名成员", "sent": sent})
}
