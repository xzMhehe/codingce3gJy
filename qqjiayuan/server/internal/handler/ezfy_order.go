package handler

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 二战风云 地图/出征/订单结算/占领/宣战/战报

// ============ 地图 ============

// ezfyIsKouCity 哈希生成寇城位置(距城心区较远), 被摧毁后复活期内不显示
func (h *EzfyHandler) ezfyIsKouCity(x, y int) bool {
	hh := ezfyAbs(x*5381 ^ y*33)
	if hh%97 != 0 {
		return false
	}
	dist := ezfyAbs(x-250) + ezfyAbs(y-250)
	if dist <= 150 {
		return false
	}
	var area model.EzfyMapArea
	if err := h.DB.Where("x = ? AND y = ? AND area_type = 2", x, y).First(&area).Error; err == nil {
		if area.StartTime > time.Now().UnixMilli() {
			return false
		}
	}
	return true
}

// MapView 以中心坐标返回区域地图（默认以当前城为中心）
func (h *EzfyHandler) MapView(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	cx, _ := strconv.Atoi(c.Query("x"))
	cy, _ := strconv.Atoi(c.Query("y"))
	if cx == 0 && cy == 0 {
		city := h.getOrCreateCity(uid)
		cx, cy = city.X, city.Y
	}
	r := 7
	if v, err := strconv.Atoi(c.Query("r")); err == nil && v > 0 && v <= 15 {
		r = v
	}
	var myCities []model.EzfyCity
	h.DB.Find(&myCities)
	cityAt := map[string]*model.EzfyCity{}
	for i := range myCities {
		c := &myCities[i]
		cityAt[fmt.Sprintf("%d,%d", c.X, c.Y)] = c
	}
	// 玩家昵称（城主显示）
	userNames := map[uint]string{}
	userIDs := []uint{}
	for _, c := range myCities {
		userIDs = append(userIDs, c.UserID)
	}
	if len(userIDs) > 0 {
		var users []model.User
		h.DB.Select("id, nickname").Where("id IN ?", userIDs).Find(&users)
		for _, u := range users {
			userNames[u.ID] = u.Nickname
		}
	}

	cells := []gin.H{}
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			terrain := ezfyTerrain(x, y)
			cell := gin.H{"x": x, "y": y, "terrain": terrain, "continent": ""}
			if x == cx && y == cy {
				cell["continent"] = ezfyContinentName(x, y)
			}
			if c, ok := cityAt[fmt.Sprintf("%d,%d", x, y)]; ok {
				cell["area_type"] = 3
				cell["city_id"] = c.ID
				cell["user_id"] = c.UserID
				cell["name"] = c.Name
				cell["city_level"] = c.CityLevel
				cell["owner"] = userNames[c.UserID]
				cell["mine"] = c.UserID == uid
			} else if terrain == 8 {
				cell["area_type"] = 1
				cell["name"] = "海野"
				cell["level"] = ezfyWildlandLevel(x, y)
			} else if h.ezfyIsKouCity(x, y) {
				cell["area_type"] = 2
				cell["name"] = "寇城"
				cell["level"] = ezfyKouLevel(x, y)
				var area model.EzfyMapArea
				if err := h.DB.Where("x = ? AND y = ?", x, y).First(&area).Error; err == nil {
					if area.AreaType == 2 && area.StartTime > time.Now().UnixMilli() {
						cell["revive_at"] = area.StartTime
						cell["name"] = "寇城(废墟)"
					}
				}
			} else {
				cell["area_type"] = 1
				cell["name"] = "野地"
				cell["level"] = ezfyWildlandLevel(x, y)
			}
			cells = append(cells, cell)
		}
	}
	resp.OK(c, gin.H{"cells": cells, "cx": cx, "cy": cy})
}

// WildlandView 野地/寇城详情（守军配置预览）
func (h *EzfyHandler) WildlandView(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	x, _ := strconv.Atoi(c.Query("x"))
	y, _ := strconv.Atoi(c.Query("y"))
	profile := h.ensureProfile(uid)
	ttype, _ := strconv.Atoi(c.Query("type"))
	if ttype != 1 && ttype != 2 && ttype != 3 {
		if ezfyTerrain(x, y) == 8 {
			ttype = 2
		} else if h.ezfyIsKouCity(x, y) {
			ttype = 3
		} else {
			ttype = 1
		}
	}
	level := ezfyWildlandLevel(x, y)
	if ttype == 3 {
		level = ezfyKouLevel(x, y)
	}
	cfg := ezfyCfg.wildland(ttype, level)
	if cfg == nil {
		resp.ParamError(c, "目标配置缺失")
		return
	}
	// 守军预览: [[兵种id,最小,最大],...]
	type troopRange struct {
		TroopId int    `json:"troop_id"`
		Name    string `json:"name"`
		Min     int64  `json:"min"`
		Max     int64  `json:"max"`
	}
	previews := []troopRange{}
	var ranges [][]int64
	if err := json.Unmarshal([]byte(cfg.Troops), &ranges); err == nil {
		for _, rg := range ranges {
			if len(rg) >= 3 {
				previews = append(previews, troopRange{
					TroopId: int(rg[0]), Name: ezfyCfg.troopName(int(rg[0]), profile.Camp),
					Min: rg[1], Max: rg[2],
				})
			}
		}
	}
	resp.OK(c, gin.H{
		"x": x, "y": y, "type": ttype, "level": level,
		"name": cfg.Des, "troops": previews,
		"res_min": cfg.ResMin, "res_max": cfg.ResMax, "terrain": ezfyTerrain(x, y),
	})
}

// ============ 出征 ============

func ezfyOrderTypeName(orderType int) string {
	switch orderType {
	case 1:
		return "侦查"
	case 2:
		return "掠夺"
	case 3:
		return "征服"
	case 4:
		return "采集"
	case 5:
		return "运输"
	case 6:
		return "增援"
	case 7:
		return "派遣"
	default:
		return "未知"
	}
}

func (h *EzfyHandler) isOwnCity(uid uint, cityId int64) bool {
	var n int64
	h.DB.Model(&model.EzfyCity{}).Where("id = ? AND user_id = ?", cityId, uid).Count(&n)
	return n > 0
}

func (h *EzfyHandler) isAllyCity(uid uint, cityId int64) bool {
	var city model.EzfyCity
	if err := h.DB.First(&city, cityId).Error; err != nil || city.UserID == uid {
		return false
	}
	var mine, theirs model.EzfyCorpsMember
	hasMine := h.DB.Where("user_id = ?", uid).First(&mine).Error == nil
	hasTheirs := h.DB.Where("user_id = ?", city.UserID).First(&theirs).Error == nil
	return hasMine && hasTheirs && mine.CorpsId == theirs.CorpsId
}

func (h *EzfyHandler) getWar(a, b uint) *model.EzfyWar {
	var w model.EzfyWar
	err := h.DB.Where("((atk_user_id = ? AND def_user_id = ?) OR (atk_user_id = ? AND def_user_id = ?)) AND status IN (1,2)",
		a, b, b, a).Order("id DESC").First(&w).Error
	if err != nil {
		return nil
	}
	return &w
}

func (h *EzfyHandler) warStatus(a, b uint) int {
	w := h.getWar(a, b)
	if w == nil {
		return 0
	}
	now := time.Now().UnixMilli()
	if now >= w.ExpireTime {
		return 0
	}
	if now >= w.EffectTime {
		if w.Status != 2 {
			h.DB.Model(&model.EzfyWar{}).Where("id = ?", w.ID).Update("status", 2)
			w.Status = 2
		}
		return 2
	}
	return 1
}

func (h *EzfyHandler) isAtWar(a, b uint) bool {
	return h.warStatus(a, b) == 2
}

// CreateOrder 出征下单（侦查/掠夺/征服/采集/运输/增援/派遣）
func (h *EzfyHandler) CreateOrder(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	var req struct {
		CityId     int64           `json:"city_id"`
		OrderType  int             `json:"order_type"`
		TargetX    int             `json:"target_x"`
		TargetY    int             `json:"target_y"`
		TargetType int             `json:"target_type"`
		TargetId   int64           `json:"target_id"`
		Troops     []ezfyUnitGroup `json:"troops"`
		Resources  map[string]int64 `json:"resources"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	city := h.cityOf(uid, req.CityId)
	if city == nil {
		city2 := h.getOrCreateCity(uid)
		city = &city2
	}
	if msg := h.createOrder(uid, city, req.OrderType, req.TargetX, req.TargetY, req.TargetType, req.TargetId, req.Troops, req.Resources); msg != "" {
		resp.ParamError(c, msg)
		return
	}
	resp.OK(c, gin.H{"msg": ezfyOrderTypeName(req.OrderType) + "命令已下达, 部队出发"})
}

func (h *EzfyHandler) createOrder(uid uint, city *model.EzfyCity, orderType, targetX, targetY, targetType int,
	targetId int64, troops []ezfyUnitGroup, resources map[string]int64) string {

	h.refreshCity(uid, city)
	// 过滤数量为0的部队
	validTroops := []ezfyUnitGroup{}
	for _, t := range troops {
		if t.Count > 0 {
			validTroops = append(validTroops, t)
		}
	}
	hasRes := false
	for _, v := range resources {
		if v > 0 {
			hasRes = true
			break
		}
	}
	if orderType != 5 && len(validTroops) == 0 {
		return "请选择出征部队"
	}
	if orderType == 5 && !hasRes && len(validTroops) == 0 {
		return "请填写运输资源或选择运输部队"
	}
	if orderType == 5 || orderType == 6 {
		if targetType != 3 || targetId <= 0 {
			return "目标必须为城市"
		}
		own := h.isOwnCity(uid, targetId)
		ally := !own && h.isAllyCity(uid, targetId)
		if orderType == 6 {
			if !own {
				return "增援(部队调动)仅限自己的城市"
			}
		} else {
			if !own && !ally {
				return "运输目标必须是自己或同盟成员的城市"
			}
			if !own && len(validTroops) > 0 {
				return "同盟运输仅限资源, 不能携带部队"
			}
			if !own && !hasRes {
				return "同盟运输必须携带资源"
			}
		}
	}
	for _, t := range validTroops {
		if cfg := ezfyCfg.troop(t.TroopId); cfg != nil && cfg.Type == 4 {
			return "城防部队不能出征"
		}
	}
	total := int64(0)
	slowest := int(^uint(0) >> 1)
	for _, t := range validTroops {
		total += t.Count
		if cfg := ezfyCfg.troop(t.TroopId); cfg != nil && cfg.Speed > 0 && cfg.Speed < slowest {
			slowest = cfg.Speed
		}
	}
	if orderType != 5 && total <= 0 {
		return "请选择出征部队"
	}
	if slowest == int(^uint(0)>>1) {
		slowest = 300
	}
	cityTroops := h.troopMap(city.ID)
	for _, t := range validTroops {
		owned := cityTroops[t.TroopId]
		if t.Count > owned {
			name := "兵种" + strconv.Itoa(t.TroopId)
			if cfg := ezfyCfg.troop(t.TroopId); cfg != nil {
				name = cfg.Name
			}
			return fmt.Sprintf("兵力不足: %s 只有%d可用", name, owned)
		}
	}
	if orderType == 4 {
		var wl model.EzfyWildland
		if err := h.DB.Where("id = ? AND city_id = ?", targetId, city.ID).First(&wl).Error; err != nil {
			return "只能采集已占领的野地"
		}
	}
	if orderType == 7 {
		var wl model.EzfyWildland
		if err := h.DB.Where("id = ? AND city_id = ?", targetId, city.ID).First(&wl).Error; err != nil {
			return "只能派遣到已占领的野地"
		}
	}
	// 掠夺/征服玩家城: 需先宣战且已生效
	if (orderType == 2 || orderType == 3) && targetType == 3 && targetId > 0 {
		var tc model.EzfyCity
		if err := h.DB.First(&tc, targetId).Error; err != nil {
			return "目标城市不存在"
		}
		if tc.UserID == uid {
			return "不能攻击自己的城市"
		}
		if h.isAllyCity(uid, targetId) {
			return "不能攻击同盟成员的城市"
		}
		if !h.isAtWar(uid, tc.UserID) {
			if w := h.getWar(uid, tc.UserID); w != nil {
				waitH := (w.EffectTime - time.Now().UnixMilli() + 3599999) / 3600000
				if waitH < 1 {
					waitH = 1
				}
				return fmt.Sprintf("宣战尚未生效, 约%d小时后开战", waitH)
			}
			return "需先对对方宣战, 宣战生效后方可掠夺/征服"
		}
	}
	// 运输: 扣减运输资源
	if orderType == 5 && hasRes {
		f, s, o, r, g := resources["food"], resources["steel"], resources["oil"], resources["rare"], resources["gold"]
		if f < 0 || s < 0 || o < 0 || r < 0 || g < 0 {
			return "资源数量错误"
		}
		if city.Food < f || city.Steel < s || city.Oil < o || city.Rare < r || city.Gold < g {
			return "资源不足,无法运输"
		}
		city.Food -= f
		city.Steel -= s
		city.Oil -= o
		city.Rare -= r
		city.Gold -= g
		h.saveCityRes(city)
	}
	// 司令部限制
	hq := h.buildingLevel(city.ID, 13)
	if hq < 1 {
		return "需要先建造司令部"
	}
	var marching int64
	h.DB.Model(&model.EzfyOrder{}).Where("user_id = ? AND status = 0", uid).Count(&marching)
	if int(marching) >= hq {
		return fmt.Sprintf("司令部%d级, 同时只能出征%d支队伍", hq, hq)
	}
	if orderType != 5 {
		carryCap := int64(10000*hq) * int64(100+h.techMap(city.ID)[15]*ezfyCommandCarryPct) / 100
		if total > carryCap {
			return fmt.Sprintf("司令部%d级, 携带上限%d万部队", hq, carryCap/10000)
		}
	}
	distance := ezfyAbs(city.X-targetX) + ezfyAbs(city.Y-targetY)
	if distance == 0 {
		return "目标太近了"
	}
	// 耗油
	var oilCost int64
	if orderType == 5 {
		f, s, o, r, g := resources["food"], resources["steel"], resources["oil"], resources["rare"], resources["gold"]
		oilCost = maxInt64(1, (f+s+o+r+g)/10000+int64(distance)/50)
	} else {
		var oilUnitTotal int64
		for _, t := range validTroops {
			if cfg := ezfyCfg.troop(t.TroopId); cfg != nil {
				oilUnitTotal += int64(cfg.OilKeep) * t.Count
			}
		}
		oilCost = maxInt64(1, oilUnitTotal*int64(distance)/ezfyOilDivGrid)
	}
	if city.Oil < oilCost {
		return fmt.Sprintf("石油不足: 本次出征需耗油%d, 当前油库仅%d", oilCost, city.Oil)
	}
	city.Oil -= oilCost
	h.saveCityRes(city)

	tech := h.techMap(city.ID)
	station := h.buildingLevel(city.ID, 20)
	travelSec := int64(distance) * 60 * 300 / int64(slowest)
	travelSec = travelSec * 100 / int64(100+tech[12]*2)
	travelSec = travelSec * 100 / int64(100+station*3)
	if travelSec < 10 {
		travelSec = 10
	}
	now := time.Now().UnixMilli()
	order := model.EzfyOrder{
		UserID: uid, CityId: int64(city.ID),
		OrderType: orderType, TargetType: targetType,
		TargetX: targetX, TargetY: targetY, TargetId: targetId,
		Troops: groupsJSON(validTroops),
		StartTime: now, ArriveTime: now + travelSec*1000,
		ReturnTime: now + travelSec*1000*2, Status: 0,
		OilUsed: oilCost,
	}
	if resources != nil {
		b, _ := json.Marshal(resources)
		order.Resources = string(b)
	}
	h.DB.Create(&order)
	// 从城市扣兵
	for _, t := range validTroops {
		var exist model.EzfyCityTroop
		if err := h.DB.Where("city_id = ? AND troop_id = ?", city.ID, t.TroopId).First(&exist).Error; err == nil && exist.Count >= t.Count {
			remain := exist.Count - t.Count
			if remain == 0 {
				h.DB.Delete(&exist)
			} else {
				h.DB.Model(&model.EzfyCityTroop{}).Where("id = ?", exist.ID).Update("count", remain)
			}
		}
	}
	// 雷达站预警
	if targetType == 3 && targetId > 0 && (orderType == 1 || orderType == 2 || orderType == 3) {
		var target model.EzfyCity
		if err := h.DB.First(&target, targetId).Error; err == nil && target.UserID > 0 {
			radar := h.buildingLevel(target.ID, 21)
			if radar >= 1 {
				warn := "军情警报: 敌方部队正向我方城市进发!\n"
				if radar >= 2 {
					warn += "进攻意图: " + ezfyOrderTypeName(orderType) + "\n"
				}
				if radar >= 3 {
					warn += fmt.Sprintf("预计到达时间: %s\n", time.UnixMilli(order.ArriveTime).Format("01-02 15:04"))
				}
				if radar >= 5 {
					warn += "出发城市: " + city.Name + "\n"
				}
				if radar >= 7 && len(validTroops) > 0 {
					tinfo := ""
					for _, t := range validTroops {
						tinfo += ezfyCfg.troopName(t.TroopId, 0) + "×" + strconv.FormatInt(t.Count, 10) + " "
					}
					if radar >= 9 {
						warn += "兵力构成: " + tinfo + "\n"
					} else {
						warn += "兵力构成: " + tinfo + "(模糊数量)\n"
					}
				}
				h.addReport(target.UserID, 6, "军情警报: 敌军来袭!", warn)
			}
		}
	}
	return ""
}

// OrderList 我的命令列表
func (h *EzfyHandler) OrderList(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	var orders []model.EzfyOrder
	h.DB.Where("user_id = ?", uid).Order("id DESC").Limit(50).Find(&orders)
	views := []gin.H{}
	for _, o := range orders {
		views = append(views, gin.H{
			"id": o.ID, "order_type": o.OrderType, "type_name": ezfyOrderTypeName(o.OrderType),
			"target_type": o.TargetType, "target_x": o.TargetX, "target_y": o.TargetY,
			"start_time": o.StartTime, "arrive_time": o.ArriveTime, "return_time": o.ReturnTime,
			"status": o.Status, "troops": parseGroups(o.Troops), "resources": o.Resources,
			"result": o.Result, "oil_used": o.OilUsed,
		})
	}
	resp.OK(c, gin.H{"orders": views})
}

// RecallOrder 召回派遣
func (h *EzfyHandler) RecallOrder(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct{ OrderId int64 `json:"order_id"` }
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	var order model.EzfyOrder
	if err := h.DB.Where("id = ? AND user_id = ?", req.OrderId, uid).First(&order).Error; err != nil {
		resp.ParamError(c, "命令不存在")
		return
	}
	if order.OrderType != 7 {
		resp.ParamError(c, "该命令不支持召回")
		return
	}
	if order.Status != 0 && order.Status != 1 {
		resp.ParamError(c, "当前状态无法召回")
		return
	}
	travel := order.ArriveTime - order.StartTime
	if travel <= 0 {
		travel = 60000
	}
	order.Status = 2
	order.Result = order.Troops
	order.ReturnTime = time.Now().UnixMilli() + travel
	h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
		Updates(map[string]interface{}{"status": 2, "result": order.Troops, "return_time": order.ReturnTime})
	h.addReport(uid, 5, "派遣报告: 部队召回",
		fmt.Sprintf("派遣部队已奉命返航, 预计%d分钟后返回出发城市, 部队将归队。", travel/1000/60), "", order.ID)
	resp.OK(c, gin.H{"msg": "已召回"})
}

// ============ 订单结算 ============

func (h *EzfyHandler) processOrders(uid uint) {
	now := time.Now().UnixMilli()
	var orders []model.EzfyOrder
	h.DB.Where("user_id = ?", uid).Order("id ASC").Find(&orders)
	for i := range orders {
		order := &orders[i]
		if order.Status == 0 && now >= order.ArriveTime {
			h.processArrive(uid, order, now)
		} else if order.Status == 1 && order.OrderType == 7 && now >= order.ArriveTime {
			h.settleDispatch(uid, order, now)
		} else if order.Status == 2 && now >= order.ReturnTime {
			h.finishReturn(uid, order)
		}
	}
}

func (h *EzfyHandler) cityOfOrder(order *model.EzfyOrder, uid uint) *model.EzfyCity {
	var city model.EzfyCity
	if err := h.DB.First(&city, order.CityId).Error; err == nil {
		return &city
	}
	main := h.getOrCreateCity(uid)
	return &main
}

func (h *EzfyHandler) finishReturn(uid uint, order *model.EzfyOrder) {
	left := parseGroups(order.Result)
	city := h.cityOfOrder(order, uid)
	for _, g := range left {
		if g.Count > 0 {
			h.addTroop(city.ID, g.TroopId, g.Count)
		}
	}
	order.Status = 3
	h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).Update("status", 3)
}

// beginReturn 异常返航: 兵力无损带回
func (h *EzfyHandler) beginReturn(order *model.EzfyOrder, now int64, travelSec int64) {
	travel := travelSec * 1000
	if travel <= 0 {
		travel = order.ArriveTime - order.StartTime
	}
	if travel <= 0 {
		travel = 60000
	}
	order.Status = 2
	order.Result = order.Troops
	order.ReturnTime = now + travel
	h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
		Updates(map[string]interface{}{"status": 2, "result": order.Troops, "return_time": order.ReturnTime})
}

// settleDispatch 派遣驻守采集结算(每8小时, 1/10 概率宝物)
func (h *EzfyHandler) settleDispatch(uid uint, order *model.EzfyOrder, now int64) {
	var wl model.EzfyWildland
	err := h.DB.First(&wl, order.TargetId).Error
	city := h.cityOfOrder(order, uid)
	if err != nil || wl.CityId != order.CityId {
		h.beginReturn(order, now, 0)
		h.addReport(uid, 5, "派遣报告: 野地丢失",
			"所派遣的野地已不属于我方, 派遣部队已返航。", "", order.ID)
		return
	}
	level := wl.Level
	var food, steel, oil, rare, gold int64
	if wl.WildType == 2 {
		oil, rare, gold = int64(level)*800, int64(level)*800, int64(level)*800
	} else {
		food, steel, oil, rare = int64(level)*800, int64(level)*800, int64(level)*800, int64(level)*800
	}
	desc := fmt.Sprintf("派遣部队在野地%d级(%d,%d)完成一次采集结算\n获得: 粮%d 钢%d 油%d 稀矿%d 金%d\n",
		level, wl.X, wl.Y, food, steel, oil, rare, gold)
	city.Food = min64(city.FoodCap, city.Food+food)
	city.Steel = min64(city.SteelCap, city.Steel+steel)
	city.Oil = min64(city.OilCap, city.Oil+oil)
	city.Rare = min64(city.RareCap, city.Rare+rare)
	city.Gold = min64(city.GoldCap, city.Gold+gold)
	h.saveCityRes(city)
	if rand.Intn(ezfyDispatchTreasure) == 0 {
		pool := []int{1, 4, 5, 6, 7, 8, 9}
		cfgId := pool[rand.Intn(len(pool))]
		if cfg := ezfyCfg.item(cfgId); cfg != nil {
			h.addItem(uid, cfgId, 1)
			desc += "运气爆棚! 获得宝物: " + cfg.Name + "\n"
		}
	}
	desc += "部队继续驻守采集, 可随时召回。"
	order.ArriveTime = now + ezfyDispatchPeriod
	order.Result = order.Troops
	h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
		Updates(map[string]interface{}{"arrive_time": order.ArriveTime, "result": order.Result})
	h.addReport(uid, 5, "派遣报告: 采集结算", desc, "", order.ID)
}

func (h *EzfyHandler) processArrive(uid uint, order *model.EzfyOrder, now int64) {
	city := h.cityOfOrder(order, uid)

	// 派遣: 到达已占领野地, 驻守采集
	if order.OrderType == 7 {
		var wl model.EzfyWildland
		if err := h.DB.First(&wl, order.TargetId).Error; err != nil || wl.CityId != order.CityId {
			h.beginReturn(order, now, 0)
			h.addReport(uid, 5, "派遣报告: 野地丢失",
				"所派遣的野地已不属于我方, 派遣部队已返航。", "", order.ID)
			return
		}
		order.Status = 1
		order.Result = order.Troops
		order.ArriveTime = now + ezfyDispatchPeriod
		h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
			Updates(map[string]interface{}{"status": 1, "result": order.Troops, "arrive_time": order.ArriveTime})
		h.addReport(uid, 5, "派遣报告: 部队已抵达",
			fmt.Sprintf("派遣部队已抵达已占领野地(%d,%d)驻守采集\n每8小时结算一次资源, 有概率获得宝物, 可随时召回。", wl.X, wl.Y), "", order.ID)
		return
	}

	// 采集: 在我方占领的野地上采集8小时资源
	if order.OrderType == 4 {
		var wl model.EzfyWildland
		if err := h.DB.First(&wl, order.TargetId).Error; err != nil || wl.CityId != order.CityId {
			h.beginReturn(order, now, 0)
			h.addReport(uid, 5, "采集报告: 野地丢失",
				"采集目标野地已不属于我方, 采集部队已返航。", "", order.ID)
			return
		}
		level := wl.Level
		var food, steel, oil, rare, gold int64
		if wl.WildType == 2 {
			oil, rare, gold = int64(level)*800, int64(level)*800, int64(level)*800
		} else {
			food, steel, oil, rare = int64(level)*800, int64(level)*800, int64(level)*800, int64(level)*800
		}
		city.Food = min64(city.FoodCap, city.Food+food)
		city.Steel = min64(city.SteelCap, city.Steel+steel)
		city.Oil = min64(city.OilCap, city.Oil+oil)
		city.Rare = min64(city.RareCap, city.Rare+rare)
		city.Gold = min64(city.GoldCap, city.Gold+gold)
		h.saveCityRes(city)
		travel := ezfyAbs64(order.ArriveTime - order.StartTime)
		order.Status = 2
		order.Result = "gather:8h"
		order.ReturnTime = now + travel
		h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
			Updates(map[string]interface{}{"status": 2, "result": "gather:8h", "return_time": order.ReturnTime})
		h.addReport(uid, 5, fmt.Sprintf("采集报告: 野地%d级(%d,%d)", wl.Level, wl.X, wl.Y),
			fmt.Sprintf("我军在占领的野地采集了8小时\n获得: 粮%d 钢%d 油%d 稀矿%d 金%d\n部队正在返回。", food, steel, oil, rare, gold))
		return
	}

	// 运输: 向目标城市运送资源
	if order.OrderType == 5 {
		var target model.EzfyCity
		if err := h.DB.First(&target, order.TargetId).Error; err != nil {
			h.beginReturn(order, now, 0)
			res := h.parseResMap(order.Resources)
			city.Food = min64(city.FoodCap, city.Food+res["food"])
			city.Steel = min64(city.SteelCap, city.Steel+res["steel"])
			city.Oil = min64(city.OilCap, city.Oil+res["oil"])
			city.Rare = min64(city.RareCap, city.Rare+res["rare"])
			city.Gold = min64(city.GoldCap, city.Gold+res["gold"])
			h.saveCityRes(city)
			h.addReport(uid, 5, "运输报告: 目标城市不存在",
				"运输目标城市已不存在, 运输部队与资源已返航。", "", order.ID)
			return
		}
		res := h.parseResMap(order.Resources)
		f, s, o, r, g := res["food"], res["steel"], res["oil"], res["rare"], res["gold"]
		target.Food = min64(target.FoodCap, target.Food+f)
		target.Steel = min64(target.SteelCap, target.Steel+s)
		target.Oil = min64(target.OilCap, target.Oil+o)
		target.Rare = min64(target.RareCap, target.Rare+r)
		target.Gold = min64(target.GoldCap, target.Gold+g)
		h.saveCityRes(&target)
		desc := fmt.Sprintf("运输部队已到达%s\n", target.Name)
		if f+s+o+r+g > 0 {
			desc += fmt.Sprintf("送达: 粮%d 钢%d 油%d 稀矿%d 金%d\n", f, s, o, r, g)
		}
		desc += "资源已送达, 护送部队正在返航。"
		h.beginReturn(order, now, 0)
		h.addReport(uid, 5, "运输报告: "+target.Name, desc)
		if target.UserID > 0 && target.UserID != uid {
			h.addReport(target.UserID, 5, "运输到达: "+city.Name,
				"来自"+city.Name+"的运输部队已到达\n"+desc)
		}
		return
	}

	// 增援: 部队常驻目标城市协防
	if order.OrderType == 6 {
		var target model.EzfyCity
		if err := h.DB.First(&target, order.TargetId).Error; err != nil {
			h.beginReturn(order, now, 0)
			h.addReport(uid, 5, "增援报告: 目标城市不存在",
				"增援目标城市已不存在, 增援部队已返航。", "", order.ID)
			return
		}
		troops := parseGroups(order.Troops)
		desc := fmt.Sprintf("增援部队已抵达%s, 协助防守。\n", target.Name)
		for _, g := range troops {
			if g.Count <= 0 {
				continue
			}
			h.addTroop(target.ID, g.TroopId, g.Count)
			desc += ezfyCfg.troopName(g.TroopId, 0) + "×" + strconv.FormatInt(g.Count, 10) + " "
		}
		order.Status = 3
		h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).Update("status", 3)
		h.addReport(uid, 5, "增援报告: "+target.Name, desc)
		if target.UserID > 0 && target.UserID != uid {
			h.addReport(target.UserID, 5, "增援到达: "+city.Name,
				city.Name+"的增援部队已抵达并驻防!\n"+desc)
		}
		return
	}

	// ============ 战斗类: 侦查/掠夺/征服 ============
	attacker := parseGroups(order.Troops)
	atkTech := h.techMap(city.ID)
	atkBonus := atkTech[5]*2 + atkTech[6]*3
	atkSpeedBonus := atkTech[10]*2 + atkTech[19]*3

	defender := []ezfyUnitGroup{}
	targetName := ""
	var lootFood, lootSteel, lootOil, lootRare, lootGold int64
	win := false
	defBonus := 0
	defSpeedBonus := 0
	wildLevel := 0
	var target *model.EzfyCity

	switch order.TargetType {
	case 1, 2:
		level := ezfyWildlandLevel(order.TargetX, order.TargetY)
		if order.TargetType == 2 {
			level = ezfyKouLevel(order.TargetX, order.TargetY)
		}
		cfgType := 1
		if order.TargetType == 2 {
			cfgType = 3
		} else if ezfyTerrain(order.TargetX, order.TargetY) == 8 {
			cfgType = 2
		}
		cfg := ezfyCfg.wildland(cfgType, level)
		if cfg == nil {
			h.beginReturn(order, now, 0)
			h.addReport(uid, 2, "战斗报告: 目标不存在",
				fmt.Sprintf("目标(%d,%d)不存在或已被摧毁, 部队已返航。", order.TargetX, order.TargetY), "", order.ID)
			return
		}
		wildLevel = level
		defender = parseWildlandTroops(cfg.Troops)
		name := "野地"
		if order.TargetType == 2 {
			name = "寇城"
		} else if ezfyTerrain(order.TargetX, order.TargetY) == 8 {
			name = "海野"
		}
		targetName = name + strconv.Itoa(level) + "级"
		rnd := cfg.ResMin + rand.Int63n(cfg.ResMax-cfg.ResMin+1)
		lootTech := atkTech[17] * 2
		rnd = rnd * int64(100+lootTech) / 100
		lootFood, lootSteel, lootOil, lootRare, lootGold = rnd, rnd, rnd, rnd, rnd
	case 3:
		var tc model.EzfyCity
		if err := h.DB.First(&tc, order.TargetId).Error; err != nil {
			h.beginReturn(order, now, 0)
			h.addReport(uid, 2, "战斗报告: 目标不存在",
				"目标城市已不存在, 部队已返航。", "", order.ID)
			return
		}
		target = &tc
		if order.OrderType == 2 || order.OrderType == 3 {
			if target.UserID == uid || !h.isAtWar(uid, target.UserID) {
				h.beginReturn(order, now, 0)
				msg := "双方未处于交战状态, 部队未交战已返航。"
				if target.UserID == uid {
					msg = "目标城市已归属我方, 部队未交战已返航。"
				}
				h.addReport(uid, 2, "战斗报告: 未宣战", msg, "", order.ID)
				return
			}
		}
		targetName = target.Name
		for tid, count := range h.troopMap(target.ID) {
			if count > 0 {
				defender = append(defender, ezfyUnitGroup{TroopId: tid, Count: count})
			}
		}
		defTech := h.techMap(target.ID)
		defBonus = h.buildingLevel(target.ID, 7)*5 + defTech[7]*3 + defTech[16]*2
		defSpeedBonus = defTech[10]*2 + defTech[19]*3
	}

	// 侦查: 不战斗只报告守军, 部队随即返航
	if order.OrderType == 1 {
		sb := ""
		for _, g := range defender {
			cfg := ezfyCfg.troop(g.TroopId)
			if cfg != nil {
				sb += cfg.Name + "×" + strconv.FormatInt(g.Count, 10) + " "
			}
		}
		travel := order.ArriveTime - order.StartTime
		if travel <= 0 {
			travel = 60000
		}
		order.Status = 2
		order.Result = order.Troops
		order.ReturnTime = now + travel
		h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
			Updates(map[string]interface{}{"status": 2, "result": order.Troops, "return_time": order.ReturnTime})
		h.addReport(uid, 1, "侦查报告: "+targetName,
			fmt.Sprintf("公文报告:侦查报告\n我方一支部队对%s[%d，%d]进行了侦查。侦查过程中未受到任何阻拦。\n守军情况: %s\n侦查完成, 部队已返航。",
				targetName, order.TargetX, order.TargetY, sb), "", order.ID)
		return
	}

	br := ezfySimulate(attacker, defender, atkBonus, defBonus, atkSpeedBonus, defSpeedBonus,
		"", "", h.buildTargetMap(city.ID, true), h.buildTargetMap(cityIdOf(target), false),
		h.buildMoveMap(city.ID, true), h.buildMoveMap(cityIdOf(target), false))
	win = br.AttackerWin

	reportType := "掠夺报告"
	if order.OrderType == 3 {
		reportType = "征服报告"
	}
	report := fmt.Sprintf("主题:%s\n出发地:%s(%d,%d)\n目的地:%s(%d,%d)\n时间:%s\n公文报告:%s\n我方一支部队对 %s[ %d，%d ]进行了%s。战斗共持续 %d 回合，我方战斗%s\n",
		reportType, city.Name, city.X, city.Y, targetName, order.TargetX, order.TargetY,
		time.UnixMilli(now).Format("2006-01-02 15:04"), reportType,
		targetName, order.TargetX, order.TargetY,
		ezfyOrderTypeName(order.OrderType), br.Rounds, winResultText(win))

	profile := h.ensureProfile(uid)
	report += fmt.Sprintf("军衔声望:%d\n", profile.Prestige)

	atkBefore := groupCounts(attacker)
	atkAfter := groupCounts(br.AttackerLeft)
	report += fmt.Sprintf("[%s]攻方:%s\n", winText(win), city.Name)
	report += troopChangeText(atkBefore, atkAfter)
	report += fmt.Sprintf("--------------------\n[%s]守方:%s\n", winText(!win), targetName)
	defBefore := groupCounts(defender)
	defAfter := map[int]int64{}
	for _, g := range br.DefenderLosses {
		defAfter[g.TroopId] = maxInt64(0, defBefore[g.TroopId]-g.Count)
	}
	for tid, cnt := range defBefore {
		if _, ok := defAfter[tid]; !ok {
			defAfter[tid] = cnt
		}
	}
	report += troopChangeText(defBefore, defAfter)

	detail := ""
	for _, a := range br.Actions {
		detail += a + "\n"
	}
	detail += "\n[双方兵力]\n"
	detail += fmt.Sprintf("[%s]攻方:%s\n", winText(win), city.Name)
	detail += troopChangeText(atkBefore, atkAfter)
	detail += fmt.Sprintf("--------------------\n[%s]守方:%s\n", winText(!win), targetName)
	detail += troopChangeText(defBefore, defAfter)
	detail += "[双方兵力]"

	// 攻方战损: 按修复率入伤兵营
	losses := br.AttackerLosses
	var deadCount int64
	for _, g := range losses {
		deadCount += g.Count
	}
	healTech := atkTech[21] * 2
	var repairedTotal int64
	if deadCount > 0 {
		for _, g := range losses {
			if g.Count <= 0 {
				continue
			}
			rate := 10
			if cfg := ezfyCfg.troop(g.TroopId); cfg != nil {
				rate = cfg.RepairRate
			}
			rate += healTech
			wounded := g.Count * int64(rate) / 100
			if wounded > g.Count {
				wounded = g.Count
			}
			if wounded > 0 {
				h.addWounded(city.ID, g.TroopId, 0, wounded)
				repairedTotal += wounded
			}
		}
	}
	// 剩余部队
	left := br.AttackerLeft
	resultStr := ""
	for _, g := range left {
		if g.Count > 0 {
			resultStr += strconv.Itoa(g.TroopId) + ":" + strconv.FormatInt(g.Count, 10) + ","
		}
	}
	if len(resultStr) > 0 {
		resultStr = resultStr[:len(resultStr)-1]
	}
	order.Result = resultStr

	// 防守方(玩家城市)战损扣除与逃兵
	if order.TargetType == 3 && target != nil {
		for _, g := range br.DefenderLosses {
			if g.Count <= 0 {
				continue
			}
			var exist model.EzfyCityTroop
			if err := h.DB.Where("city_id = ? AND troop_id = ?", target.ID, g.TroopId).First(&exist).Error; err == nil {
				remain := exist.Count - g.Count
				if remain <= 0 {
					h.DB.Delete(&exist)
				} else {
					h.DB.Model(&model.EzfyCityTroop{}).Where("id = ?", exist.ID).Update("count", remain)
				}
			}
		}
		if win {
			for _, g := range br.DefenderLosses {
				deserters := g.Count * ezfyDeserterRate / 100
				if deserters > 0 {
					h.addWounded(target.ID, g.TroopId, 1, deserters)
				}
			}
		}
		h.saveCityRes(target)
	}

	// 携带容量(剩余部队负重)
	var carry int64
	for _, g := range left {
		if cfg := ezfyCfg.troop(g.TroopId); cfg != nil {
			carry += int64(cfg.Carry) * g.Count
		}
	}

	targetProtected := false
	if order.TargetType == 3 && target != nil {
		targetProtected = h.hasCityEffect(target.ID, 2)
	}

	if win {
		if targetProtected && order.TargetType == 3 {
			report += "\n目标城市处于免战保护期, 无法掠夺资源!"
		}
		if order.TargetType == 3 && target != nil {
			// 玩家城市: 掠夺比例10%+掠夺技巧, 上限50%
			lootRate := 10 + atkTech[17]*2
			if targetProtected || order.OrderType != 2 && order.OrderType != 3 {
				lootRate = 0
			}
			if lootRate > 50 {
				lootRate = 50
			}
			defRes := []int64{target.Food, target.Steel, target.Oil, target.Rare, target.Gold}
			loot := make([]int64, 5)
			var totalLoot int64
			for i := 0; i < 5; i++ {
				loot[i] = defRes[i] * int64(lootRate) / 100
				totalLoot += loot[i]
			}
			if totalLoot > carry && carry > 0 {
				scale := float64(carry) / float64(totalLoot)
				for i := 0; i < 5; i++ {
					loot[i] = int64(float64(loot[i]) * scale)
				}
			}
			lootFood, lootSteel, lootOil, lootRare, lootGold = loot[0], loot[1], loot[2], loot[3], loot[4]
			target.Food = maxInt64(0, target.Food-lootFood)
			target.Steel = maxInt64(0, target.Steel-lootSteel)
			target.Oil = maxInt64(0, target.Oil-lootOil)
			target.Rare = maxInt64(0, target.Rare-lootRare)
			target.Gold = maxInt64(0, target.Gold-lootGold)
		}

		// 征服野地/寇城: 占领
		if order.OrderType == 3 && (order.TargetType == 1 || order.TargetType == 2) {
			hallLevel := h.buildingLevel(city.ID, 1)
			owned := len(h.wildlandList(city.ID))
			if owned >= hallLevel {
				city.Food += lootFood
				city.Steel += lootSteel
				city.Oil += lootOil
				city.Rare += lootRare
				city.Gold += lootGold
				h.saveCityRes(city)
				travel := ezfyAbs64(order.ArriveTime - order.StartTime)
				order.Status = 2
				order.ReturnTime = now + travel
				report += fmt.Sprintf("\n我军胜利!但附属野地数量已达上限(%d块), 放弃占领.\n", hallLevel)
				report += fmt.Sprintf("\n掠夺资源: 粮%d 钢%d 油%d 稀矿%d 金%d", lootFood, lootSteel, lootOil, lootRare, lootGold)
				pg := 30 + wildLevel*10
				if order.TargetType == 2 {
					pg = 80 + wildLevel*20
				}
				h.addPrestige(uid, pg)
				report += fmt.Sprintf("\n军功声望+%d", pg)
				report += "\n[双方兵力]"
				h.addReport(uid, 2, "战斗报告: "+targetName, report, detail, order.ID)
				h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
					Updates(map[string]interface{}{"status": order.Status, "result": order.Result, "return_time": order.ReturnTime})
				return
			}
			wildType := 1
			if order.TargetType == 2 || ezfyTerrain(order.TargetX, order.TargetY) == 8 {
				wildType = 2
			}
			wl := model.EzfyWildland{CityId: int64(city.ID), X: order.TargetX, Y: order.TargetY,
				WildType: wildType, Level: wildLevel, Status: 0}
			h.DB.Create(&wl)
			h.taskProgress(uid, "occupy_wild", 1)
			var area model.EzfyMapArea
			if err := h.DB.Where("x = ? AND y = ?", order.TargetX, order.TargetY).First(&area).Error; err != nil {
				area = model.EzfyMapArea{X: order.TargetX, Y: order.TargetY, AreaType: 1,
					OwnerId: int64(city.ID), Level: wildLevel, StartTime: time.Now().UnixMilli()}
				h.DB.Create(&area)
			} else {
				h.DB.Model(&model.EzfyMapArea{}).Where("id = ?", area.ID).
					Updates(map[string]interface{}{"area_type": 1, "owner_id": int64(city.ID), "level": wildLevel})
			}
			// 寇城被摧毁, 24小时后复活
			if order.TargetType == 2 {
				reviveAt := now + 24*3600000
				var ka model.EzfyMapArea
				if err := h.DB.Where("x = ? AND y = ?", order.TargetX, order.TargetY).First(&ka).Error; err != nil {
					ka = model.EzfyMapArea{X: order.TargetX, Y: order.TargetY, AreaType: 2,
						Level: wildLevel, StartTime: reviveAt}
					h.DB.Create(&ka)
				} else {
					h.DB.Model(&model.EzfyMapArea{}).Where("id = ?", ka.ID).
						Updates(map[string]interface{}{"area_type": 2, "owner_id": 0, "level": wildLevel, "start_time": reviveAt})
				}
			}
			// 俘获: 小概率收服守军
			capturedCount := int64(0)
			capturedTroopId := 0
			if len(defender) > 0 && rand.Intn(100) < 15 {
				g := defender[rand.Intn(len(defender))]
				cap := g.Count / 100
				if cap < 1 {
					cap = 1
				}
				capturedTroopId = g.TroopId
				capturedCount = cap
				h.addTroop(city.ID, capturedTroopId, capturedCount)
			}
			if capturedCount > 0 {
				report += fmt.Sprintf("\n俘获: %s×%d", ezfyCfg.troopName(capturedTroopId, 0), capturedCount)
			}
		}

		// 征服玩家城市
		if order.OrderType == 3 && order.TargetType == 3 && target != nil {
			surv := int64(0)
			for _, g := range left {
				surv += g.Count
			}
			feelingDrop := int(surv / 2000)
			if feelingDrop < 1 {
				feelingDrop = 1
			}
			cur := target.Feelings - feelingDrop
			if cur < 0 {
				cur = 0
			}
			report += fmt.Sprintf("\n民心 - %d\n当前民心 %d", feelingDrop, cur)
			if cur > 0 {
				target.Feelings = cur
				h.saveCityRes(target)
				h.DB.Model(&model.EzfyCity{}).Where("id = ?", target.ID).Update("feelings", target.Feelings)
				travel := ezfyAbs64(order.ArriveTime - order.StartTime)
				order.Status = 2
				order.ReturnTime = now + travel
				report += "\n民心尚存，征服失败"
				report += fmt.Sprintf("\n征战果\n黄金:%d\n粮食:%d\n钢铁:%d\n石油:%d\n稀矿:%d", lootGold, lootFood, lootSteel, lootOil, lootRare)
				city.Food += lootFood
				city.Steel += lootSteel
				city.Oil += lootOil
				city.Rare += lootRare
				city.Gold += lootGold
				h.saveCityRes(city)
				report += "\n[双方兵力]"
				h.addReport(uid, 3, "征服报告: "+targetName, report, detail, order.ID)
				h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
					Updates(map[string]interface{}{"status": order.Status, "result": order.Result, "return_time": order.ReturnTime})
				return
			}
			if targetProtected {
				report += "\n民心已失，但目标处于免战保护期，无法征服!"
				target.Feelings = cur
				h.DB.Model(&model.EzfyCity{}).Where("id = ?", target.ID).Update("feelings", target.Feelings)
				travel := ezfyAbs64(order.ArriveTime - order.StartTime)
				order.Status = 2
				order.ReturnTime = now + travel
				report += "\n[双方兵力]"
				h.addReport(uid, 3, "征服报告: "+targetName, report, detail, order.ID)
				h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
					Updates(map[string]interface{}{"status": order.Status, "result": order.Result, "return_time": order.ReturnTime})
				return
			}
			report += "\n民心已失，征服成功!"
			target.Feelings = 0
			target.Grievance = minInt(100, target.Grievance+50)
			defRes := []int64{target.Food, target.Steel, target.Oil, target.Rare, target.Gold}
			loot := make([]int64, 5)
			for i := 0; i < 5; i++ {
				loot[i] = defRes[i] * 50 / 100
			}
			lootFood, lootSteel, lootOil, lootRare, lootGold = loot[0], loot[1], loot[2], loot[3], loot[4]
			target.Food = defRes[0] - lootFood
			target.Steel = defRes[1] - lootSteel
			target.Oil = defRes[2] - lootOil
			target.Rare = defRes[3] - lootRare
			target.Gold = defRes[4] - lootGold
			lastCity := false
			var cityCount int64
			h.DB.Model(&model.EzfyCity{}).Where("user_id = ?", target.UserID).Count(&cityCount)
			if cityCount <= 1 {
				lastCity = true
			}
			if !lastCity {
				occ := model.EzfyOccupy{CityId: int64(target.ID), CityName: targetName,
					AtkUserId: uid, AtkCityId: int64(city.ID), DefUserId: target.UserID,
					X: target.X, Y: target.Y, Status: 1}
				h.DB.Create(&occ)
				report += "\n占领成功! 城市已归入你的附属, 可在[附属野地]中摧毁/归还"
			} else {
				report += "\n该玩家仅剩最后一座城市, 无法占领!"
			}
			h.saveCityRes(target)
			h.DB.Model(&model.EzfyCity{}).Where("id = ?", target.ID).
				Updates(map[string]interface{}{"feelings": 0, "grievance": target.Grievance})
			defReportBody := ""
			if !lastCity {
				defReportBody = fmt.Sprintf("你的城市%s已被敌方部队占领!\n民心清零!\n被掠夺资源: 粮%d 钢%d 油%d 稀矿%d 金%d\n%s",
					targetName, lootFood, lootSteel, lootOil, lootRare, lootGold, lossText(br.DefenderLosses))
			} else {
				defReportBody = fmt.Sprintf("敌方部队攻破了你的城市%s!\n民心清零, 但该城市是你的最后一座城, 无法被占领!\n被掠夺资源: 粮%d 钢%d 油%d 稀矿%d 金%d\n%s",
					targetName, lootFood, lootSteel, lootOil, lootRare, lootGold, lossText(br.DefenderLosses))
			}
			h.addReport(target.UserID, 4, "城破报告: "+city.Name, defReportBody, detail)
		}
		// 普通掠夺(含成功掠夺玩家城市): 民心-5 民怨+5
		if order.OrderType == 2 && order.TargetType == 3 && target != nil {
			target.Feelings = maxInt(0, target.Feelings-5)
			target.Grievance = minInt(100, target.Grievance+5)
			h.saveCityRes(target)
			h.DB.Model(&model.EzfyCity{}).Where("id = ?", target.ID).
				Updates(map[string]interface{}{"feelings": target.Feelings, "grievance": target.Grievance})
			h.addReport(target.UserID, 2, "被掠夺报告: "+city.Name,
				fmt.Sprintf("你的城市%s被敌方部队掠夺!\n被掠夺资源: 粮%d 钢%d 油%d 稀矿%d 金%d\n民心-5 民怨+5\n%s",
					targetName, lootFood, lootSteel, lootOil, lootRare, lootGold, lossText(br.DefenderLosses)),
				detail)
		}
		// 掠夺资源入账
		if order.OrderType == 2 || order.OrderType == 3 {
			city.Food += lootFood
			city.Steel += lootSteel
			city.Oil += lootOil
			city.Rare += lootRare
			city.Gold += lootGold
			h.saveCityRes(city)
		}
		// 军功声望
		prestigeGain := 0
		switch order.TargetType {
		case 1:
			prestigeGain = 30 + wildLevel*10
		case 2:
			prestigeGain = 80 + wildLevel*20
		case 3:
			if order.OrderType == 3 {
				prestigeGain = 500
			} else {
				prestigeGain = 200 + int((lootFood+lootSteel+lootOil+lootRare)/10000)
			}
		}
		if prestigeGain > 0 {
			h.addPrestige(uid, prestigeGain)
			report += fmt.Sprintf("\n军功声望+%d", prestigeGain)
		}
		if order.TargetType == 1 {
			h.taskProgress(uid, "battle_wild", 1)
		} else if order.TargetType == 2 {
			h.taskProgress(uid, "battle_kou", 1)
		}
		var enemyDead int64
		for _, g := range br.DefenderLosses {
			enemyDead += g.Count
		}
		if enemyDead > 0 {
			h.taskProgress(uid, "kill_enemy", int(enemyDead))
		}
		travel := ezfyAbs64(order.ArriveTime - order.StartTime)
		order.Status = 2
		order.ReturnTime = now + travel
		report += fmt.Sprintf("\n战果\n黄金:%d\n粮食:%d\n钢铁:%d\n石油:%d\n稀矿:%d", lootGold, lootFood, lootSteel, lootOil, lootRare)
		if repairedTotal > 0 {
			report += fmt.Sprintf("\n伤兵入营: %d(可前往司令部伤兵营恢复)", repairedTotal)
		}
		report += "\n[双方兵力]"
		h.addReport(uid, 2, "战斗报告: "+targetName, report, detail, order.ID)
	} else {
		order.Status = 4
		if repairedTotal > 0 {
			report += fmt.Sprintf("\n伤兵入营: %d(可前往司令部伤兵营恢复)", repairedTotal)
		}
		report += "\n[双方兵力]"
		h.addReport(uid, 2, "战斗报告: "+targetName, report, detail, order.ID)
		if order.TargetType == 3 && target != nil {
			h.addReport(target.UserID, 4, "守卫报告: "+city.Name,
				fmt.Sprintf("你的城市%s成功抵挡了敌方部队的进攻!\n%s", targetName, lossText(br.DefenderLosses)), detail)
			h.addPrestige(target.UserID, 100)
		}
	}
	h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
		Updates(map[string]interface{}{"status": order.Status, "result": order.Result, "return_time": order.ReturnTime})
}

// ============ 辅助 ============

func (h *EzfyHandler) parseResMap(s string) map[string]int64 {
	m := map[string]int64{}
	if s == "" {
		return m
	}
	_ = json.Unmarshal([]byte(s), &m)
	return m
}

func (h *EzfyHandler) buildTargetMap(cityId uint, atk bool) map[int]int {
	if cityId <= 0 {
		return nil
	}
	var list []model.EzfyCityTarget
	h.DB.Where("city_id = ?", cityId).Find(&list)
	m := map[int]int{}
	for _, t := range list {
		if atk {
			m[t.TroopId] = t.AtkTargetTroop
		} else {
			m[t.TroopId] = t.DefTargetTroop
		}
	}
	return m
}

func (h *EzfyHandler) buildMoveMap(cityId uint, atk bool) map[int]int {
	if cityId <= 0 {
		return nil
	}
	var list []model.EzfyCityTarget
	h.DB.Where("city_id = ?", cityId).Find(&list)
	m := map[int]int{}
	for _, t := range list {
		if atk {
			m[t.TroopId] = t.AtkMove
		} else {
			m[t.TroopId] = t.DefMove
		}
	}
	return m
}

// parseWildlandTroops 解析 [[兵种id,最小,最大],...] 生成守军(随机数量)
func parseWildlandTroops(s string) []ezfyUnitGroup {
	groups := []ezfyUnitGroup{}
	var ranges [][]int64
	if err := json.Unmarshal([]byte(s), &ranges); err != nil {
		return groups
	}
	for _, rg := range ranges {
		if len(rg) < 3 {
			continue
		}
		lo, hi, tid := rg[1], rg[2], int(rg[0])
		if hi < lo {
			hi = lo
		}
		count := lo
		if hi > lo {
			count = lo + rand.Int63n(hi-lo+1)
		}
		groups = append(groups, ezfyUnitGroup{TroopId: tid, Count: count})
	}
	return groups
}

func groupCounts(groups []ezfyUnitGroup) map[int]int64 {
	m := map[int]int64{}
	for _, g := range groups {
		m[g.TroopId] += g.Count
	}
	return m
}

func troopChangeText(before, after map[int]int64) string {
	ids := []int{}
	for tid := range before {
		ids = append(ids, tid)
	}
	text := ""
	for _, tid := range ids {
		name := "兵种" + strconv.Itoa(tid)
		if cfg := ezfyCfg.troop(tid); cfg != nil {
			name = cfg.Name
		}
		b := before[tid]
		a := after[tid]
		if a > b {
			a = b
		}
		if b-a > 0 || b > 0 {
			text += fmt.Sprintf("%s %d->%d 损失%d\n", name, b, a, b-a)
		}
	}
	return text
}

func lossText(groups []ezfyUnitGroup) string {
	if len(groups) == 0 {
		return "守军无损失"
	}
	text := "守军损失: "
	for _, g := range groups {
		name := "兵种" + strconv.Itoa(g.TroopId)
		if cfg := ezfyCfg.troop(g.TroopId); cfg != nil {
			name = cfg.Name
		}
		text += name + "×" + strconv.FormatInt(g.Count, 10) + " "
	}
	return text
}

func winText(win bool) string {
	if win {
		return "胜"
	}
	return "败"
}

func winResultText(win bool) string {
	if win {
		return "胜利！"
	}
	return "失败！"
}

func cityIdOf(c *model.EzfyCity) uint {
	if c == nil {
		return 0
	}
	return c.ID
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func ezfyAbs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

// addReport 战报写入
func (h *EzfyHandler) addReport(uid uint, reportType int, title, content string, detailAndOrder ...interface{}) {
	r := model.EzfyReport{UserID: uid, ReportType: reportType, Title: title, Content: content, IsRead: 0}
	for i, v := range detailAndOrder {
		switch i {
		case 0:
			if s, ok := v.(string); ok {
				r.Detail = s
			}
		case 1:
			switch n := v.(type) {
			case int64:
				r.OrderId = n
			case int:
				r.OrderId = int64(n)
			}
		}
	}
	h.DB.Create(&r)
}
