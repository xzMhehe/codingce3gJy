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

// 二战风云 世界聊天 + 交易所 + 被占城市管理

// ============ 世界聊天 ============

func (h *EzfyHandler) ChatList(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	var chats []model.EzfyChat
	h.DB.Order("id DESC").Limit(50).Find(&chats)
	views := make([]gin.H, 0, len(chats))
	for i := len(chats) - 1; i >= 0; i-- {
		ch := chats[i]
		views = append(views, gin.H{"id": ch.ID, "user_name": ch.UserName,
			"content": ch.Content, "created_at": ch.CreatedAt, "mine": ch.UserId == uid})
	}
	var online int64
	h.DB.Model(&model.EzfyProfile{}).Count(&online)
	resp.OK(c, gin.H{"chats": views, "players": online})
}

func (h *EzfyHandler) ChatSend(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
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
	h.DB.Create(&model.EzfyChat{UserId: uid, UserName: profile.Nickname, Content: content})
	resp.OK(c, gin.H{"msg": "发送成功"})
}

// ============ 交易所 ============

var ezfyResNames = map[int]string{1: "粮食", 2: "钢铁", 3: "石油", 4: "稀矿"}

func (h *EzfyHandler) ExchangeList(c *gin.Context) {
	uid := middleware.GetUID(c)
	var list []model.EzfyExchange
	h.DB.Where("status = 0").Order("id DESC").Limit(100).Find(&list)
	views := []gin.H{}
	for _, e := range list {
		if e.SellerId == uid && e.IsSystem != 1 {
			continue
		}
		views = append(views, gin.H{"id": e.ID, "seller_name": e.SellerName,
			"type": e.EsType, "type_name": ezfyResNames[e.EsType],
			"count": e.EsCount, "total_price": e.TotalPrice,
			"unit_price": e.TotalPrice / maxInt64(1, e.EsCount), "mine": e.SellerId == uid})
	}
	var mine []model.EzfyExchange
	h.DB.Where("seller_id = ? AND status = 0", uid).Order("id DESC").Find(&mine)
	mineViews := []gin.H{}
	for _, e := range mine {
		mineViews = append(mineViews, gin.H{"id": e.ID, "type": e.EsType,
			"type_name": ezfyResNames[e.EsType], "count": e.EsCount, "total_price": e.TotalPrice})
	}
	city := h.getOrCreateCity(uid)
	resp.OK(c, gin.H{"orders": views, "mine": mineViews, "gold": city.Gold})
}

func (h *EzfyHandler) ExchangeSell(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		EsType     int   `json:"es_type"`
		EsCount    int64 `json:"es_count"`
		TotalPrice int64 `json:"total_price"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if _, ok := ezfyResNames[req.EsType]; !ok {
		resp.ParamError(c, "资源类型错误")
		return
	}
	if req.EsCount <= 0 || req.TotalPrice <= 0 {
		resp.ParamError(c, "数量或价格错误")
		return
	}
	city := h.getOrCreateCity(uid)
	h.calcResource(&city)
	var stock int64
	switch req.EsType {
	case 1:
		stock = city.Food
	case 2:
		stock = city.Steel
	case 3:
		stock = city.Oil
	case 4:
		stock = city.Rare
	}
	if stock < req.EsCount {
		resp.ParamError(c, fmt.Sprintf("%s不足(现有%d)", ezfyResNames[req.EsType], stock))
		return
	}
	switch req.EsType {
	case 1:
		city.Food -= req.EsCount
	case 2:
		city.Steel -= req.EsCount
	case 3:
		city.Oil -= req.EsCount
	case 4:
		city.Rare -= req.EsCount
	}
	h.saveCityRes(&city)
	profile := h.ensureProfile(uid)
	h.DB.Create(&model.EzfyExchange{SellerId: uid, SellerName: profile.Nickname,
		EsType: req.EsType, EsCount: req.EsCount, TotalPrice: req.TotalPrice, Status: 0})
	resp.OK(c, gin.H{"msg": fmt.Sprintf("挂单成功: %s×%d 售%d黄金", ezfyResNames[req.EsType], req.EsCount, req.TotalPrice)})
}

func (h *EzfyHandler) ExchangeBuy(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Id uint `json:"id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	var e model.EzfyExchange
	if err := h.DB.Where("id = ? AND status = 0", req.Id).First(&e).Error; err != nil {
		resp.ParamError(c, "订单不存在或已成交")
		return
	}
	if e.SellerId == uid {
		resp.ParamError(c, "不能购买自己的挂单")
		return
	}
	city := h.getOrCreateCity(uid)
	h.calcResource(&city)
	if city.Gold < e.TotalPrice {
		resp.ParamError(c, fmt.Sprintf("黄金不足(需%d)", e.TotalPrice))
		return
	}
	city.Gold -= e.TotalPrice
	switch e.EsType {
	case 1:
		city.Food += e.EsCount
	case 2:
		city.Steel += e.EsCount
	case 3:
		city.Oil += e.EsCount
	case 4:
		city.Rare += e.EsCount
	}
	h.saveCityRes(&city)
	// 黄金转给卖家(受其黄金容量上限)
	var sellerCity model.EzfyCity
	if err := h.DB.Where("user_id = ?", e.SellerId).Order("id ASC").First(&sellerCity).Error; err == nil {
		sellerCity.Gold += e.TotalPrice
		if sellerCity.Gold > sellerCity.GoldCap {
			sellerCity.Gold = sellerCity.GoldCap
		}
		h.saveCityRes(&sellerCity)
	}
	h.DB.Model(&model.EzfyExchange{}).Where("id = ?", e.ID).
		Updates(map[string]interface{}{"status": 1, "buyer_id": uid})
	h.addReport(e.SellerId, 6, "交易成交",
		fmt.Sprintf("你挂单出售的%s×%d已被%s以%d黄金购得。", ezfyResNames[e.EsType], e.EsCount, h.ensureProfile(uid).Nickname, e.TotalPrice))
	resp.OK(c, gin.H{"msg": fmt.Sprintf("购买成功: %s×%d", ezfyResNames[e.EsType], e.EsCount)})
}

func (h *EzfyHandler) ExchangeCancel(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Id uint `json:"id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	var e model.EzfyExchange
	if err := h.DB.Where("id = ? AND seller_id = ? AND status = 0", req.Id, uid).First(&e).Error; err != nil {
		resp.ParamError(c, "订单不存在")
		return
	}
	city := h.getOrCreateCity(uid)
	switch e.EsType {
	case 1:
		city.Food += e.EsCount
	case 2:
		city.Steel += e.EsCount
	case 3:
		city.Oil += e.EsCount
	case 4:
		city.Rare += e.EsCount
	}
	h.saveCityRes(&city)
	h.DB.Model(&model.EzfyExchange{}).Where("id = ?", e.ID).Update("status", 2)
	resp.OK(c, gin.H{"msg": "已下架, 资源退回"})
}

// CorpsMembers 军团成员列表
func (h *EzfyHandler) CorpsMembers(c *gin.Context) {
	uid := middleware.GetUID(c)
	var mb model.EzfyCorpsMember
	if err := h.DB.Where("user_id = ?", uid).First(&mb).Error; err != nil {
		resp.OK(c, gin.H{"members": []gin.H{}, "in_corps": false})
		return
	}
	var members []model.EzfyCorpsMember
	h.DB.Where("corps_id = ?", mb.CorpsId).Order("is_leader DESC, id ASC").Find(&members)
	views := []gin.H{}
	for _, m := range members {
		p := h.ensureProfile(m.UserId)
		views = append(views, gin.H{"user_id": m.UserId, "name": p.Nickname,
			"is_leader": m.IsLeader, "title": m.Title,
			"prestige": p.Prestige, "rank_name": ezfyRankName(p.Prestige)})
	}
	resp.OK(c, gin.H{"members": views, "in_corps": true})
}

// ============ 被占城市管理 ============

func (h *EzfyHandler) OccupyList(c *gin.Context) {
	uid := middleware.GetUID(c)
	var cities []model.EzfyCity
	h.DB.Where("user_id = ?", uid).Find(&cities)
	ids := make([]int64, 0, len(cities))
	for _, ct := range cities {
		ids = append(ids, int64(ct.ID))
	}
	views := []gin.H{}
	if len(ids) > 0 {
		var list []model.EzfyOccupy
		h.DB.Where("atk_city_id IN ? AND status = 1", ids).Find(&list)
		for _, o := range list {
			p := h.ensureProfile(o.DefUserId)
			views = append(views, gin.H{"id": o.ID, "x": o.X, "y": o.Y,
				"city_name": o.CityName, "def_user": p.Nickname})
		}
	}
	resp.OK(c, gin.H{"occupies": views})
}

func (h *EzfyHandler) OccupyOp(c *gin.Context) {
	uid := middleware.GetUID(c)
	op := c.Param("op")
	var req struct {
		OccupyId uint `json:"occupy_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	var cities []model.EzfyCity
	h.DB.Where("user_id = ?", uid).Find(&cities)
	ids := make([]int64, 0, len(cities))
	for _, ct := range cities {
		ids = append(ids, int64(ct.ID))
	}
	var o model.EzfyOccupy
	if err := h.DB.Where("id = ? AND atk_city_id IN ? AND status = 1", req.OccupyId, ids).First(&o).Error; err != nil {
		resp.ParamError(c, "占领记录不存在")
		return
	}
	var city model.EzfyCity
	if err := h.DB.First(&city, o.CityId).Error; err != nil {
		resp.ParamError(c, "城市数据不存在")
		return
	}
	switch op {
	case "build":
		h.DB.Model(&model.EzfyOccupy{}).Where("id = ?", o.ID).Update("status", 4)
		h.DB.Model(&model.EzfyCity{}).Where("id = ?", city.ID).
			Updates(map[string]interface{}{"feelings": 50, "grievance": 0, "last_time": time.Now().UnixMilli()})
		h.addReport(uid, 5, "建城成功",
			fmt.Sprintf("你将被占领的城市[%s]正式建立为自己的城市, 可在城市列表切换操作。", city.Name))
		resp.OK(c, gin.H{"msg": "建城成功"})
	case "destroy":
		name := o.CityName
		h.deleteCityData(int64(city.ID))
		h.DB.Model(&model.EzfyOccupy{}).Where("id = ?", o.ID).Update("status", 3)
		h.addReport(uid, 5, "摧毁城市", fmt.Sprintf("你摧毁了占领的城市[%s]。", name))
		resp.OK(c, gin.H{"msg": "摧毁成功"})
	case "return":
		h.DB.Model(&model.EzfyCity{}).Where("id = ?", city.ID).
			Updates(map[string]interface{}{"user_id": o.DefUserId})
		if city.Feelings < 20 {
			h.DB.Model(&model.EzfyCity{}).Where("id = ?", city.ID).Update("feelings", 20)
		}
		h.DB.Model(&model.EzfyOccupy{}).Where("id = ?", o.ID).Update("status", 2)
		h.addReport(o.DefUserId, 5, "城市归还",
			fmt.Sprintf("你被占领的城市[%s]已由敌方归还!\n民心已恢复。", city.Name))
		resp.OK(c, gin.H{"msg": "已归还"})
	default:
		resp.ParamError(c, "未知操作")
	}
}

// deleteCityData 删除城市及其所有数据
func (h *EzfyHandler) deleteCityData(cityId int64) {
	h.DB.Where("city_id = ?", cityId).Delete(&model.EzfyCityBuilding{})
	h.DB.Where("city_id = ?", cityId).Delete(&model.EzfyCityTroop{})
	h.DB.Where("city_id = ?", cityId).Delete(&model.EzfyCityTech{})
	h.DB.Where("city_id = ?", cityId).Delete(&model.EzfyTrainQueue{})
	h.DB.Where("city_id = ?", cityId).Delete(&model.EzfyWildland{})
	h.DB.Where("city_id = ?", cityId).Delete(&model.EzfyWounded{})
	h.DB.Where("city_id = ?", cityId).Delete(&model.EzfyCityTarget{})
	h.DB.Where("city_id = ?", cityId).Delete(&model.EzfyCityEffect{})
	h.DB.Where("city_id = ?", cityId).Delete(&model.EzfyOrder{})
	h.DB.Delete(&model.EzfyCity{}, cityId)
}

// WildlandFull 附属野地+被占城市(cityWild 页数据)
func (h *EzfyHandler) WildlandFull(c *gin.Context) {
	uid := middleware.GetUID(c)
	city := h.getOrCreateCity(uid)
	h.refreshCity(uid, &city)
	var wildlands []model.EzfyWildland
	h.DB.Where("city_id = ?", city.ID).Find(&wildlands)
	wildViews := []gin.H{}
	for _, w := range wildlands {
		wildViews = append(wildViews, gin.H{"id": w.ID, "x": w.X, "y": w.Y,
			"wild_type": w.WildType, "level": w.Level, "status": w.Status})
	}
	var occupies []model.EzfyOccupy
	h.DB.Where("atk_city_id = ? AND status = 1", city.ID).Find(&occupies)
	occViews := []gin.H{}
	for _, o := range occupies {
		p := h.ensureProfile(o.DefUserId)
		occViews = append(occViews, gin.H{"id": o.ID, "x": o.X, "y": o.Y,
			"city_name": o.CityName, "def_user": p.Nickname})
	}
	resp.OK(c, gin.H{"city": city, "wildlands": wildViews, "occupies": occViews,
		"hall_level": h.buildingLevel(city.ID, 1)})
}

// OrderView 命令详情
func (h *EzfyHandler) OrderView(c *gin.Context) {
	uid := middleware.GetUID(c)
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var o model.EzfyOrder
	if err := h.DB.Where("id = ? AND user_id = ?", id, uid).First(&o).Error; err != nil {
		resp.NotFound(c, "命令不存在")
		return
	}
	h.cfgs()
	troops := parseGroups(o.Troops)
	troopViews := []gin.H{}
	for _, g := range troops {
		name := "兵种" + strconv.Itoa(g.TroopId)
		if cfg := ezfyCfg.troop(g.TroopId); cfg != nil {
			name = cfg.Name
		}
		troopViews = append(troopViews, gin.H{"name": name, "count": g.Count})
	}
	// 目的地名称(复刻 report/viewCityTroopOut 的「目的地」)
	toName := h.ezfyTargetName(&o)
	// 出发地
	fromName := ""
	if city := h.cityOfOrder(&o, uid); city != nil {
		fromName = city.Name + "(" + strconv.Itoa(city.X) + "," + strconv.Itoa(city.Y) + ")"
	}
	// 携带资源
	resViews := []gin.H{}
	if res := h.parseResMap(o.Resources); len(res) > 0 {
		for _, k := range []string{"gold", "food", "steel", "oil", "rare"} {
			if v, ok := res[k]; ok && v > 0 {
				resViews = append(resViews, gin.H{"key": k, "name": ezfyResLabel(k), "count": v})
			}
		}
	}
	// 关联战报
	var rep model.EzfyReport
	hasReport := h.DB.Where("user_id = ? AND order_id = ?", uid, o.ID).Order("id DESC").First(&rep).Error == nil
	resp.OK(c, gin.H{
		"id": o.ID, "order_type": o.OrderType, "type_name": ezfyOrderTypeName(o.OrderType),
		"target_type": o.TargetType, "target_name": toName, "from_name": fromName,
		"target_x": o.TargetX, "target_y": o.TargetY, "status": o.Status,
		"status_name": ezfyOrderStatusName(o.Status),
		"start_time":  o.StartTime, "arrive_time": o.ArriveTime, "return_time": o.ReturnTime,
		"start_text": ezfyFmtTime(o.StartTime), "arrive_text": ezfyFmtTime(o.ArriveTime),
		"return_text": ezfyFmtTime(o.ReturnTime),
		"troops":      troopViews, "oil_used": o.OilUsed, "officer": o.Officer,
		"resources": resViews,
		"report_id": func() int64 {
			if hasReport {
				return int64(rep.ID)
			}
			return 0
		}(),
	})
}

// ezfyResLabel 资源 key → 中文名
func ezfyResLabel(k string) string {
	switch k {
	case "gold":
		return "黄金"
	case "food":
		return "粮食"
	case "steel":
		return "钢铁"
	case "oil":
		return "石油"
	default:
		return "稀矿"
	}
}

// ezfyFmtTime 毫秒时间戳 → yyyy-MM-dd HH:mm:ss
func ezfyFmtTime(ms int64) string {
	if ms <= 0 {
		return ""
	}
	return time.UnixMilli(ms).Format("2006-01-02 15:04:05")
}

// ezfyOrderStatusName 命令状态中文
func ezfyOrderStatusName(s int) string {
	switch s {
	case 0:
		return "行军中"
	case 1:
		return "驻守中"
	case 2:
		return "返航中"
	case 3:
		return "已完成"
	case 4:
		return "已终止"
	default:
		return "未知"
	}
}

// ezfyTargetName 命令目的地名称(城市名 / 野地N级 / 寇城N级 / 海野N级)
func (h *EzfyHandler) ezfyTargetName(o *model.EzfyOrder) string {
	switch o.TargetType {
	case 1, 2:
		level := ezfyWildlandLevel(o.TargetX, o.TargetY)
		name := "野地"
		if o.TargetType == 2 {
			level = ezfyKouLevel(o.TargetX, o.TargetY)
			name = "寇城"
		} else if ezfyTerrain(o.TargetX, o.TargetY) == 8 {
			name = "海野"
		}
		return name + strconv.Itoa(level) + "级"
	case 3:
		var c model.EzfyCity
		if err := h.DB.First(&c, o.TargetId).Error; err == nil {
			return c.Name
		}
		return "城市"
	default:
		return "未知"
	}
}
