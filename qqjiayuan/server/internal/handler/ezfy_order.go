package handler

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
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
	// ★ 管理端「地图格子覆盖」优先：标记成寇城/活动寇城就一定是寇城，
	//   标记成活动野地/特殊城市就一定不是。
	if mk := ezfyMarkKindAt(x, y); mk > 0 {
		return mk == model.EzfyMarkKou || mk == model.EzfyMarkActKou
	}
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
	// 默认半径 2 → 5×5(复刻 map/index.html)
	r := 2
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

	// ★ 同盟成员集合：只有同盟(同一军团)玩家的城市才允许「运输 / 增援」。
	//   前端据此决定这两个按钮显不显示（宣战中一律不显示）。
	//   ★ 用户规则：「同盟玩家不能宣战」→ 前端也用它把 [宣战] 按钮藏掉。
	allyUsers := map[uint]bool{}
	var myMb model.EzfyCorpsMember
	if err := h.DB.Where("user_id = ?", uid).First(&myMb).Error; err == nil && myMb.CorpsId > 0 {
		var mbs []model.EzfyCorpsMember
		h.DB.Where("corps_id = ?", myMb.CorpsId).Find(&mbs)
		for _, m := range mbs {
			if m.UserId != uid {
				allyUsers[m.UserId] = true
			}
		}
	}

	// ★ 用户要求「点击地图的出征 → 看到玩家城市 → 点进去 → 展示玩家同盟名字」。
	//   一次性把所有涉及玩家的军团名载入（避免逐格查库），格子上带 corps_name。
	corpsNames := map[uint]string{}
	if len(userIDs) > 0 {
		var allMb []model.EzfyCorpsMember
		h.DB.Where("user_id IN ?", userIDs).Find(&allMb)
		cids := []uint{}
		seenCid := map[uint]bool{}
		for _, m := range allMb {
			if m.CorpsId > 0 && !seenCid[m.CorpsId] {
				seenCid[m.CorpsId] = true
				cids = append(cids, m.CorpsId)
			}
		}
		if len(cids) > 0 {
			var cs []model.EzfyCorps
			h.DB.Select("id, name").Where("id IN ?", cids).Find(&cs)
			corpsName := map[uint]string{}
			for _, cp := range cs {
				corpsName[cp.ID] = cp.Name
			}
			for _, m := range allMb {
				if m.CorpsId > 0 {
					corpsNames[m.UserId] = corpsName[m.CorpsId]
				}
			}
		}
	}

	// 已占领野地(一次性载入, 避免逐格查库)
	var allWilds []model.EzfyWildland
	h.DB.Select("x, y, city_id").Find(&allWilds)
	wildAt := map[string]int64{}
	for i := range allWilds {
		if allWilds[i].CityId > 0 {
			wildAt[fmt.Sprintf("%d,%d", allWilds[i].X, allWilds[i].Y)] = allWilds[i].CityId
		}
	}

	cells := []gin.H{}
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			// ★ 用 Ex 地形：平原且靠海显示为「沿海平原」(9)；海洋仍是 8
			terrain := ezfyTerrainEx(x, y)
			// ★ 每一格都带上所属大洲 / 大洋，前端才能标注「这个城/野地在哪个州」
			cell := gin.H{"x": x, "y": y, "terrain": terrain,
				"terrain_name": ezfyTerrainName(terrain), "continent": ezfyRegionName(x, y)}
			if c, ok := cityAt[fmt.Sprintf("%d,%d", x, y)]; ok {
				cell["area_type"] = 3
				cell["city_id"] = c.ID
				cell["user_id"] = c.UserID
				cell["name"] = c.Name
				cell["city_level"] = c.CityLevel
				cell["owner"] = userNames[c.UserID]
				cell["mine"] = c.UserID == uid
				cell["ally"] = allyUsers[c.UserID]
				// ★ 该城主的同盟（军团）名；没加入军团时为空串，前端显示「无」
				cell["corps_name"] = corpsNames[c.UserID]
			} else {
				kou := h.ezfyIsKouCity(x, y)
				switch {
				case terrain == ezfyTerrainSea:
					cell["area_type"] = 1
					cell["name"] = ezfyTerrainName(terrain) // 海洋
					cell["level"] = ezfyWildlandLevel(x, y)
				case kou:
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
				default:
					// 陆地野地: 名称取地形名(平原/草原/森林/盆地/丘陵/沼泽/山地), 不再一律叫「野地」
					cell["area_type"] = 1
					cell["name"] = ezfyTerrainName(terrain)
					cell["level"] = ezfyWildlandLevel(x, y)
				}
				// 活动目标标记: 复刻 mapView.html 的 actWild/actKou/actCity
				// (活动野地橙、活动寇城品红、特殊城市红, 三种都带活动等级 1~3)
				if act := h.ezfyActTargetType(x, y); act > 0 {
					actLevel := ezfyMarkLevelAt(x, y)
					cell["act_type"] = act
					cell["act_level"] = actLevel
					cell["act_name"] = ezfyActTargetName(act)
					// 格子名直接用活动标签, 目标详情页标题即「活动野地2级 / 特殊城市3级」
					cell["name"] = ezfyActTargetLabel(act, actLevel)
				}
			}
			// 该格是否已被某城占领(详情页据此决定能不能采集)
			if _, ok := wildAt[fmt.Sprintf("%d,%d", x, y)]; ok {
				cell["occupied"] = true
			}
			cells = append(cells, cell)
		}
	}
	// 发现精英中立城市(复刻地图页的「发现精英中立城市：[寇(x,y)]」)
	// 以当前视野中心为原点由近及远扫一圈寇城, 取最近的一个
	elite := gin.H{}
	for r := 1; r <= 30 && len(elite) == 0; r++ {
		for dx := -r; dx <= r && len(elite) == 0; dx++ {
			for dy := -r; dy <= r && len(elite) == 0; dy++ {
				if ezfyAbs(dx) != r && ezfyAbs(dy) != r {
					continue // 只看这一圈的边框
				}
				ex, ey := cx+dx, cy+dy
				if ezfyTerrain(ex, ey) == 8 || !h.ezfyIsKouCity(ex, ey) {
					continue
				}
				elite = gin.H{"x": ex, "y": ey, "level": ezfyKouLevel(ex, ey)}
			}
		}
	}
	resp.OK(c, gin.H{"cells": cells, "cx": cx, "cy": cy, "elite": elite})
}

// WildlandView 野地/寇城详情（守军配置预览）
func (h *EzfyHandler) WildlandView(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	x, _ := strconv.Atoi(c.Query("x"))
	y, _ := strconv.Atoi(c.Query("y"))
	profile := h.ensureProfile(uid)
	// 活动目标(活动野地/活动寇城/特殊城市): 守军/奖励/说明走活动配置, 不走普通野地配置表
	if act := h.ezfyActTargetType(x, y); act > 0 {
		resp.OK(c, h.ezfyActWildlandView(uid, profile.Camp, x, y, act))
		return
	}
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
				// ★ 守军预览同样乘「野地兵力倍数」，否则玩家看到的和实际打到的不一致
				previews = append(previews, troopRange{
					TroopId: int(rg[0]), Name: ezfyCfg.troopName(int(rg[0]), profile.Camp),
					Min: ezfyScaleByWildMult(rg[1]), Max: ezfyScaleByWildMult(rg[2]),
				})
			}
		}
	}
	// 采集可获得的珠宝(按地形固定, 复刻 cfg_equipment 的「珠宝(地形)」)
	jewelName := ""
	if j := h.randomJewel(ezfyTerrain(x, y)); j != nil {
		jewelName = j.Name
	}
	// 归属: 已占领该野地的玩家(复刻 mapView 的【归属: xxx】)
	owner := ""
	var w model.EzfyWildland
	if err := h.DB.Where("x = ? AND y = ?", x, y).First(&w).Error; err == nil && w.CityId > 0 {
		var oc model.EzfyCity
		if err := h.DB.First(&oc, w.CityId).Error; err == nil {
			if oc.UserID == uid {
				owner = "我"
			} else {
				owner = h.ensureProfile(oc.UserID).Nickname
			}
		}
	}
	resp.OK(c, gin.H{
		"x": x, "y": y, "type": ttype, "level": level,
		"name": cfg.Des, "troops": previews,
		"res_min": cfg.ResMin, "res_max": cfg.ResMax, "terrain": ezfyTerrain(x, y),
		"terrain_name": ezfyTerrainNameEx(x, y),
		"continent":    ezfyContinentName(x, y),
		"jewel":        jewelName,
		"owner":        owner,
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
		// ★ 7 = 驻守野地长期采集（一键采集用），8 = 城际调兵（城市列表的[派遣]）
		return "驻守采集"
	case 8:
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

// isAtWar 是否可以对该玩家发起掠夺/征服
//
// ★ 用户要求「加一个宣战功能开关，关闭后不需要宣战也能掠夺/征服」→
// 开关关掉时恒为 true（视为随时可交战）。这样出征校验、战斗结算两处一起放开，
// 不会出现「出征放行了、到达时又被判没宣战而返航」的不一致。
func (h *EzfyHandler) isAtWar(a, b uint) bool {
	if !ezfyWarRequireOn() {
		return true
	}
	return h.warStatus(a, b) == 2
}

// CreateOrder 出征下单（侦查/掠夺/征服/采集/运输/增援/派遣）
func (h *EzfyHandler) CreateOrder(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	var req struct {
		CityId     int64            `json:"city_id"`
		OrderType  int              `json:"order_type"`
		TargetX    int              `json:"target_x"`
		TargetY    int              `json:"target_y"`
		TargetType int              `json:"target_type"`
		TargetId   int64            `json:"target_id"`
		Troops     []ezfyUnitGroup  `json:"troops"`
		Resources  map[string]int64 `json:"resources"`
		Officer    string           `json:"officer"`
		WaitMin    int              `json:"wait_min"` // 宿营分钟数(≤1440)
		Gather     int              `json:"gather"`   // ★ 集结令个数(0~10)，提高本次出征兵力上限
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
	waitMin := req.WaitMin
	if waitMin < 0 {
		waitMin = 0
	}
	if waitMin > 1440 {
		waitMin = 1440
	}
	if msg := h.createOrder(uid, city, req.OrderType, req.TargetX, req.TargetY, req.TargetType, req.TargetId, req.Troops, req.Resources, req.Officer, waitMin, req.Gather); msg != "" {
		resp.ParamError(c, msg)
		return
	}
	resp.OK(c, gin.H{"msg": ezfyOrderTypeName(req.OrderType) + "命令已下达, 部队出发"})
}

// OrderPreview POST /games/ezfy/order/preview
// 复刻原版出征页的 [计算] 按钮：出征前预览 油耗/负重/耗时, 不下达命令、不扣资源。
func (h *EzfyHandler) OrderPreview(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	var req struct {
		CityId    int64            `json:"city_id"`
		OrderType int              `json:"order_type"`
		TargetX   int              `json:"target_x"`
		TargetY   int              `json:"target_y"`
		Troops    []ezfyUnitGroup  `json:"troops"`
		Resources map[string]int64 `json:"resources"`
		Officer   string           `json:"officer"`
		WaitMin   int              `json:"wait_min"`
		Gather    int              `json:"gather"` // ★ 集结令个数
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	city := h.bodyCity(uid, req.CityId)
	h.refreshCity(uid, city)

	valid := []ezfyUnitGroup{}
	var carry int64
	slowest := 0
	for _, t := range req.Troops {
		if t.Count <= 0 {
			continue
		}
		cfg := ezfyCfg.troop(t.TroopId)
		if cfg == nil {
			continue
		}
		// ★ 防御兵种(type=4 城防) 固定阵地，不能出征 —— 预览时直接忽略
		if cfg.Type == 4 {
			continue
		}
		valid = append(valid, t)
		carry += int64(cfg.Carry) * t.Count
		if slowest == 0 || cfg.Speed < slowest {
			slowest = cfg.Speed
		}
	}
	distance := ezfyAbs(city.X-req.TargetX) + ezfyAbs(city.Y-req.TargetY)
	oilCost := h.ezfyOilCost(city, req.OrderType, distance, valid, req.Resources)

	var travelSec int64
	if distance > 0 && slowest > 0 {
		tech := h.techMap(city.ID)
		station := h.buildingLevel(city.ID, 20)
		travelSec = int64(distance) * 60 * 300 / int64(slowest)
		travelSec = travelSec * 100 / int64(100+tech[12]*2)
		travelSec = travelSec * 100 / int64(100+station*3)
		if lead := h.officerByName(city.ID, req.Officer); h.officerSpeedSkill(lead) {
			travelSec = travelSec * 100 / 110
		}
		if travelSec < 10 {
			travelSec = 10
		}
	}
	waitMin := req.WaitMin
	if waitMin < 0 {
		waitMin = 0
	}
	if waitMin > 1440 {
		waitMin = 1440
	}
	needSec := travelSec*2 + int64(waitMin)*60
	// ★ 出征兵力上限（含集结令加成）—— 前端 [计算] 时直接显示「本次出兵 N / 上限 M」
	gather := req.Gather
	if gather < 0 {
		gather = 0
	}
	gatherMax := ezfyGatherMax()
	if gather > gatherMax {
		gather = gatherMax
	}
	totalPreview := int64(0)
	for _, t := range valid {
		totalPreview += t.Count
	}
	// ★ 管理端「出征上限」开关关掉时 capUnlimited=true（前端显示「不限」）
	capNow, capUnlimited := h.ezfyOrderTroopCap(city.ID, gather)
	resp.OK(c, gin.H{
		"oil_used":    oilCost,
		"oil_enough":  city.Oil >= oilCost,
		"oil_have":    city.Oil,
		"carry":       carry,
		"distance":    distance,
		"travel_sec":  travelSec,
		"wait_min":    waitMin,
		"need_sec":    needSec,
		"need_time":   ezfyDurationText(needSec),
		"travel_time": ezfyDurationText(travelSec),
		"return_time": ezfyDurationText(travelSec),
		// 出征上限相关
		// ★ cap_unlimited = 管理端把「出征上限」开关关了 → 前端显示「不限」而不是一串巨大数字
		"troop_total":    totalPreview,
		"troop_cap":      capNow,
		"cap_unlimited":  capUnlimited,
		"hq_level":       h.buildingLevel(city.ID, 13),
		"gather":         gather,
		"gather_per":     ezfyGatherBonusPer(),
		"gather_max":     gatherMax,
		"gather_have":    h.itemCount(uid, ezfyGatherItemID),
		"troop_over_cap": !capUnlimited && totalPreview > capNow,
	})
}

// ezfyDurationText 秒 → 「1小时23分45秒」文本
func ezfyDurationText(sec int64) string {
	if sec <= 0 {
		return "0秒"
	}
	d := sec / 86400
	sec %= 86400
	hr := sec / 3600
	sec %= 3600
	mi := sec / 60
	sec %= 60
	out := ""
	if d > 0 {
		out += fmt.Sprintf("%d天", d)
	}
	if hr > 0 {
		out += fmt.Sprintf("%d小时", hr)
	}
	if mi > 0 {
		out += fmt.Sprintf("%d分", mi)
	}
	if sec > 0 || out == "" {
		out += fmt.Sprintf("%d秒", sec)
	}
	return out
}

// ============ 集结令 / 出征兵力上限 ============

const (
	// ezfyGatherItemID 集结令道具 id（ezfy_cfg_item）
	ezfyGatherItemID = 19
	// ezfyGatherDefaultPer 每个集结令提升的出征上限（配置表 param1 优先）
	ezfyGatherDefaultPer = 100000
	// ezfyGatherMaxDefault 单次出征最多使用多少个集结令的**默认值**。
	// ★ 用户要求「出征集结令上限后台管理系统可维护，最大默认 50」→ 默认 50。
	//   真正的上限由 ezfyGatherMax() 从 ezfy_cfg_limit.gather_max_per_order 读取，
	//   管理端「建筑上限配置」页可改，改完 cfgsReload() 即时生效。
	//   这个常量只在配置行缺失/为 0 时兜底。
	ezfyGatherMaxDefault = 50
)

// ezfyGatherBonusPer 每个集结令提升的出征上限（读配置 param1，缺省 10 万）
func ezfyGatherBonusPer() int64 {
	if it := ezfyCfg.item(ezfyGatherItemID); it != nil && it.Param1 > 0 {
		return it.Param1
	}
	return ezfyGatherDefaultPer
}

// ezfyOrderTroopCap 本次出征的兵力上限
//
//	= 司令部等级 × 1万 × (1 + 指挥艺术科技等级 × 10%)   ← 原版规则（司令部「每次出征上限N人」）
//	+ 集结令个数 × ezfyGatherBonusPer()                ← 用户规则：每个集结令 +10 万
//
// ★ 用户要求「再加个出征上限开关，默认开；关闭后出征没有上限」→
//
//	开关关掉时返回 (0, true)，调用方一律用 unlimited 判断，**不要**拿 0 去比大小。
func (h *EzfyHandler) ezfyOrderTroopCap(cityId uint, gather int) (cap int64, unlimited bool) {
	if !ezfyMarchCapOn() {
		return 0, true
	}
	hq := h.buildingLevel(cityId, 13)
	cap = int64(10000*hq) * int64(100+h.techMap(cityId)[15]*ezfyCommandCarryPct) / 100
	if gather > 0 {
		cap += int64(gather) * ezfyGatherBonusPer()
	}
	return cap, false
}

func (h *EzfyHandler) createOrder(uid uint, city *model.EzfyCity, orderType, targetX, targetY, targetType int,
	targetId int64, troops []ezfyUnitGroup, resources map[string]int64, officer string, waitMin, gather int) string {

	h.refreshCity(uid, city)
	// 过滤数量为0的部队
	validTroops := []ezfyUnitGroup{}
	for _, t := range troops {
		if t.Count > 0 {
			// ★ 防御兵种(type=4 城防：碉堡/榴弹炮/反坦克炮/防空炮…) 固定阵地，不能出征
			if c := ezfyCfg.troop(t.TroopId); c != nil && c.Type == 4 {
				return "防御兵种「" + c.Name + "」固定阵地，不能出征"
			}
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
				// 盟友驻军(复刻联络中心): 只能增援同一联盟成员的城市,
				// 且目标城需有联络中心, 驻军队伍数受其等级限制
				if !ally {
					return "增援(部队调动)仅限自己的城市或同一联盟成员的城市"
				}
				var tc model.EzfyCity
				if err := h.DB.First(&tc, targetId).Error; err != nil {
					return "目标城市不存在"
				}
				cap := h.allyGarrisonCap(tc.ID)
				if cap < 1 {
					return "目标城市未建造联络中心, 无法接收盟友驻军"
				}
				if h.allyGarrisonCount(tc.ID) >= cap {
					return fmt.Sprintf("目标城市联络中心%d级, 最多接收%d支盟友驻军", cap, cap)
				}
			}
		} else {
			// ★ 第九轮用户规则：自己城市之间能运输，同盟(军团)成员之间也能运输；
			//   运量看负重（所以要用卡车等部队装），**可以不带队军官**。
			if !own && !ally {
				return "运输目标必须是自己或同盟成员的城市"
			}
			if !hasRes {
				return "运输必须携带资源"
			}
		}
	}
	// 派遣(8): 城际调兵 —— 只能派往自己的城市，必须带部队
	if orderType == 8 {
		if targetType != 3 || targetId <= 0 {
			return "派遣目标必须为自己的城市"
		}
		if !h.isOwnCity(uid, targetId) {
			return "派遣只能派往自己的城市"
		}
		if targetId == int64(city.ID) {
			return "不能派遣到当前所在城市"
		}
		if len(validTroops) == 0 {
			return "派遣必须携带部队"
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
	// ★ 集结令：先校验参数（不超过管理端配置的单次上限、背包要够），再校验兵力与上限
	if gather < 0 {
		gather = 0
	}
	if gm := ezfyGatherMax(); gather > gm {
		return fmt.Sprintf("集结令单次最多使用%d个", gm)
	}
	if gather > 0 {
		if have := h.itemCount(uid, ezfyGatherItemID); have < gather {
			return fmt.Sprintf("集结令不足: 需要%d个, 当前只有%d个", gather, have)
		}
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
	// ★ 出征兵力上限见下面的司令部限制（含集结令加成），这里不再重复校验
	if orderType == 4 {
		var wl model.EzfyWildland
		if err := h.DB.Where("id = ? AND city_id = ?", targetId, city.ID).First(&wl).Error; err != nil {
			return "只能采集已占领的野地"
		}
		// ★ 用户规则：采集/派遣都要带一个军官（带队）
		if officer == "" {
			return "采集部队必须携带一名军官"
		}
	}
	if orderType == 7 {
		var wl model.EzfyWildland
		if err := h.DB.Where("id = ? AND city_id = ?", targetId, city.ID).First(&wl).Error; err != nil {
			return "只能派遣到已占领的野地"
		}
		if officer == "" {
			return "派遣部队必须携带军官"
		}
	}
	// 带队军官校验: 必须存在且在职(未出征/非俘虏)
	if officer != "" {
		lead := h.officerByName(city.ID, officer)
		if lead == nil {
			return "军官不存在"
		}
		// ★ 用户规则：同一座城市里，一个军官同时只能带一支队伍出征。
		//   只要他还有未结束的命令（行军中/驻守中/返航中），就不能再接新命令。
		//   这里查命令表而不是只看 officer.Status —— 后者可能因历史数据漂移不准。
		if lead.Status == 1 || h.officerBusyOrder(city.ID, officer) {
			return "军官" + lead.Name + "正在出征中, 未归队前不能再次出征"
		}
		if lead.IsCaptive == 1 {
			return "俘虏不能带队出征, 请先在军校收编"
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
	// 运输/派遣: 扣减随军资源
	if (orderType == 5 || orderType == 8) && hasRes {
		f, s, o, r, g := resources["food"], resources["steel"], resources["oil"], resources["rare"], resources["gold"]
		if f < 0 || s < 0 || o < 0 || r < 0 || g < 0 {
			return "资源数量错误"
		}
		// ★ 第九轮：运输必须有部队来装（负重决定能运多少），军官可以不带队。
		if len(validTroops) == 0 {
			return "运输需要携带部队来装载资源(卡车负重最高)"
		}
		cap := h.ezfyCarryCapOf(validTroops)
		if total := f + s + o + r + g; total > cap {
			return fmt.Sprintf("负重不足: 本次要携带%d, 部队负重只有%d(多带卡车可提高)", total, cap)
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
	if orderType != 5 && orderType != 8 {
		// ★ 携带上限 = 司令部等级 × 1万 × 指挥艺术加成 + 集结令加成（每个集结令 +10 万）
		//   ★ 管理端「出征上限」开关关掉时 capUnlimited=true → 完全不做这个校验
		carryCap, capUnlimited := h.ezfyOrderTroopCap(city.ID, gather)
		if !capUnlimited && total > carryCap {
			msg := fmt.Sprintf("司令部%d级, 携带上限%d万部队", hq, carryCap/10000)
			if gm := ezfyGatherMax(); gather < gm {
				msg += fmt.Sprintf("。可使用集结令提高上限: 每个+%d, 单次最多%d个", ezfyGatherBonusPer(), gm)
			}
			return msg
		}
	}
	distance := ezfyAbs(city.X-targetX) + ezfyAbs(city.Y-targetY)
	if distance == 0 {
		return "目标太近了"
	}
	// 耗油
	oilCost := h.ezfyOilCost(city, orderType, distance, validTroops, resources)
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
	// 带队军官「移速」技能: 行军 +10%
	if lead := h.officerByName(city.ID, officer); h.officerSpeedSkill(lead) {
		travelSec = travelSec * 100 / 110
	}
	if travelSec < 10 {
		travelSec = 10
	}
	now := time.Now().UnixMilli()
	// 宿营: 到达后停留 waitMin 分钟再返航(复刻原版出征页的「宿营」, 上限 24 小时)
	if waitMin < 0 {
		waitMin = 0
	}
	if waitMin > 1440 {
		waitMin = 1440
	}
	// ★ 走到这里所有校验都过了，才真正扣掉集结令（失败路径不能白扣玩家道具）
	if gather > 0 {
		for i := 0; i < gather; i++ {
			h.consumeItem(uid, ezfyGatherItemID)
		}
	}
	order := model.EzfyOrder{
		UserID: uid, CityId: int64(city.ID),
		OrderType: orderType, TargetType: targetType,
		TargetX: targetX, TargetY: targetY, TargetId: targetId,
		Troops:    groupsJSON(validTroops),
		Officer:   officer,
		StartTime: now, ArriveTime: now + travelSec*1000,
		ReturnTime: now + travelSec*1000*2, Status: 0,
		OilUsed: oilCost, WaitMin: waitMin,
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
	// 带队军官: 置为出征中, 忠诚 -5(归零自动离职)
	if officer != "" {
		h.officerGoOut(city, officer, true)
	}
	// 雷达站预警：**能不能提前看见，取决于被攻击方自己城市的雷达站等级**。
	//
	//	侦查(1)      → 「被侦查报告」（军情警讯）
	//	掠夺(2)/征服(3) → 「军情警报: 敌军来袭!」，细节随雷达等级递增
	//
	// 注意：这里只发**事前预警**；被掠夺/城破这类**事后结果**报告在 processArrive 里发，
	// 不受雷达站限制 —— 否则玩家资源被抢光了却毫不知情。
	if targetType == 3 && targetId > 0 && (orderType == 1 || orderType == 2 || orderType == 3) {
		var target model.EzfyCity
		if err := h.DB.First(&target, targetId).Error; err == nil && target.UserID > 0 {
			radar := h.buildingLevel(target.ID, 21)
			if radar >= 1 && orderType == 1 {
				body := "有敌军对我方城市进行了侦查!\n"
				if radar >= 3 {
					body += "侦查方城市: " + city.Name + "\n"
				}
				if radar >= 5 {
					body += fmt.Sprintf("侦查时间: %s\n", time.UnixMilli(order.StartTime).Format("01-02 15:04"))
				}
				body += fmt.Sprintf("(雷达站%d级: 等级越高, 情报越详细)", radar)
				h.addReport(target.UserID, 6, "被侦查报告: "+city.Name, body)
			}
			if radar >= 1 && orderType != 1 {
				warn := "军情警报: 敌方部队正向我方城市进发!\n"
				if radar >= 2 {
					warn += "进攻意图: " + ezfyOrderTypeName(orderType) + "\n"
				}
				if radar >= 3 {
					warn += fmt.Sprintf("预计到达时间: %s\n", time.UnixMilli(order.ArriveTime).Format("01-02 15:04"))
				}
				if radar >= 4 {
					if officer == "" {
						warn += "统帅: 无(未带军官)\n"
					} else {
						warn += "统帅: " + officer + "\n"
					}
				}
				if radar >= 5 {
					warn += "出发城市: " + city.Name + "\n"
				}
				if radar >= 6 {
					warn += fmt.Sprintf("出发时间: %s\n", time.UnixMilli(order.StartTime).Format("01-02 15:04"))
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
	// ★ 用户规则：出征队列只列**还在外面**的部队（行军中/驻守中/返航中）。
	//   已结束(3已完成/4已终止)的命令不再常驻队列，战报里还能查到。
	h.DB.Where("user_id = ? AND status IN (0,1,2)", uid).Order("id DESC").Limit(50).Find(&orders)
	views := []gin.H{}
	for _, o := range orders {
		views = append(views, gin.H{
			"id": o.ID, "order_type": o.OrderType, "type_name": ezfyOrderTypeName(o.OrderType),
			"target_type": o.TargetType, "target_x": o.TargetX, "target_y": o.TargetY,
			"start_time": o.StartTime, "arrive_time": o.ArriveTime, "return_time": o.ReturnTime,
			"start_text": ezfyFmtTime(o.StartTime), "arrive_text": ezfyFmtTime(o.ArriveTime),
			"return_text": ezfyFmtTime(o.ReturnTime),
			"wait_min":    o.WaitMin,
			"status":      o.Status, "status_name": ezfyOrderStatusName(o.Status),
			"troops": parseGroups(o.Troops), "resources": o.Resources,
			"result": o.Result, "oil_used": o.OilUsed, "officer": o.Officer,
		})
	}
	resp.OK(c, gin.H{"orders": views})
}

// RecallOrder 召回派遣
// ezfyOneWayTravel 命令的单程行军时长(毫秒)
//
// 创建命令时: ArriveTime = start + travel, ReturnTime = start + 2*travel,
// 所以 (ReturnTime - StartTime)/2 恒为单程时长。
// ⚠️ 不能用 ArriveTime - StartTime: 派遣(7) 抵达后会把 ArriveTime 推到「下一个结算周期」,
// 那样算出来会凭空多出 8 小时。
func ezfyOneWayTravel(order *model.EzfyOrder) int64 {
	if t := (order.ReturnTime - order.StartTime) / 2; t > 0 {
		return t
	}
	if t := order.ArriveTime - order.StartTime; t > 0 {
		return t
	}
	return 60000
}

// RecallOrder 取消出征命令（原「召回」，已放开到所有命令类型）
//
// ★ 用户要求「出征队列可以取消」：原来只允许 type=7(驻守采集) 召回，
// 其余命令一律返回「该命令不支持召回」。现在所有**还在外面**的命令
// （status 0 行军中 / 1 驻守中）都能取消，部队原路返回出发城市。
//
// 返程时间：
//   - 行军中(0)：已经走了多久就花多久回去，最少 10 秒
//   - 驻守中(1)：按单程行军时长返航
//
// 随军资源：运输(5)/派遣(8) 出发时已从城里扣掉，取消时写进 Carry，
// 由 finishReturn 原样带回出发城市（受仓储上限截断，不会凭空多出资源）。
// 宿营：取消时 WaitMin 清零，不再原地等待。
func (h *EzfyHandler) RecallOrder(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	var req struct {
		OrderId int64 `json:"order_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	var order model.EzfyOrder
	if err := h.DB.Where("id = ? AND user_id = ?", req.OrderId, uid).First(&order).Error; err != nil {
		resp.ParamError(c, "命令不存在")
		return
	}
	if order.Status != 0 && order.Status != 1 {
		resp.ParamError(c, "该命令已在返航中或已结束, 无法取消")
		return
	}
	now := time.Now().UnixMilli()
	oneWay := ezfyOneWayTravel(&order)
	var back int64
	if order.Status == 0 {
		back = now - order.StartTime // 已走时长 ≈ 返程时长
		if back > oneWay {
			back = oneWay
		}
	} else {
		back = oneWay
	}
	if back < 10000 {
		back = 10000
	}
	// 运输(5)/派遣(8)：随军资源出发时就扣了，取消必须原样带回
	carry := order.Carry
	if (order.OrderType == 5 || order.OrderType == 8) && strings.TrimSpace(order.Resources) != "" {
		carry = order.Resources
	}
	order.Status = 2
	order.Result = order.Troops
	order.Carry = carry
	order.WaitMin = 0
	order.ReturnTime = now + back
	h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).Updates(map[string]interface{}{
		"status": 2, "result": order.Result, "carry": carry,
		"wait_min": 0, "return_time": order.ReturnTime,
	})
	name := ezfyOrderTypeName(order.OrderType)
	h.addReport(uid, 5, name+"报告: 已取消",
		fmt.Sprintf("%s命令已取消, 部队原路返回, 预计%s后抵达出发城市。", name, ezfyDurationText(back/1000)),
		"", order.ID)
	resp.OK(c, gin.H{"msg": name + "已取消, 部队正在返回"})
}

// ============ 订单结算 ============

// processOrders 结算该 uid 名下所有到期的行军命令。
//
// ★★ 重入守卫（2026-09-21 线上性能事故）：本函数与 refreshCity 构成闭环 ——
//
//	refreshCity → processOrders → processArrive → refreshCity(防守方) → processOrders …
//
// 战斗时懒结算防守方是必要的（否则用陈旧民心算征服），但必须防自喂。
// 原来只在两处写了 `target.UserID == uid` 的判断，覆盖不了 A↔B 互打、
// 以及「同一轮内订单对象还是 status=0、未落库改状态」被重复进 processArrive 的情况。
// 现在统一在这里加 per-uid 守卫：同一个 uid 已经在结算中，第二次调用直接返回。
//
// 注意：守卫只在**订单结算**这一层加，cityViews / calcResource 等纯展示逻辑不受影响。
func (h *EzfyHandler) processOrders(uid uint) {
	if !h.enterProcess(uid) {
		// 已在结算中（递归回调）→ 跳过，交回上层继续处理，避免无限自喂
		return
	}
	defer h.exitProcess(uid)

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
	// ★ 部队带回的采集资源在这里入城（受仓储上限截断）
	c := parseCarry(order.Carry)
	if c.total() > 0 {
		city.Food = min64(city.FoodCap, city.Food+c.Food)
		city.Steel = min64(city.SteelCap, city.Steel+c.Steel)
		city.Oil = min64(city.OilCap, city.Oil+c.Oil)
		city.Rare = min64(city.RareCap, city.Rare+c.Rare)
		city.Gold = min64(city.GoldCap, city.Gold+c.Gold)
		h.saveCityRes(city)
		h.addReport(uid, 5, "部队返航: 采集资源已入库",
			fmt.Sprintf("采集部队返回%s\n带回: 粮%d 钢%d 油%d 稀矿%d 金%d",
				city.Name, c.Food, c.Steel, c.Oil, c.Rare, c.Gold), "", order.ID)
	}
	// 带队军官归来, 恢复在职
	if order.Officer != "" {
		h.officerGoOut(city, order.Officer, false)
	}
	order.Status = 3
	order.Carry = ""
	h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
		Updates(map[string]interface{}{"status": 3, "carry": ""})
}

// beginReturn 异常返航: 兵力无损带回
// ezfyOilCost 出征耗油(运输按携带资源量计, 其余按兵种油耗×数量×距离计)
//
// ★ 用户要求「加个出征油耗开关，默认开；关了出征消耗油 0」→ 关掉时直接返回 0。
// 出征预览(/order/preview)与真正下单(createOrder)都走这里，所以「看到的 0」就是「实扣的 0」。
func (h *EzfyHandler) ezfyOilCost(city *model.EzfyCity, orderType, distance int,
	troops []ezfyUnitGroup, resources map[string]int64) int64 {
	if !ezfyMarchOilOn() {
		return 0
	}
	if orderType == 5 {
		f, s, o, r, g := resources["food"], resources["steel"], resources["oil"], resources["rare"], resources["gold"]
		return maxInt64(1, (f+s+o+r+g)/10000+int64(distance)/50)
	}
	var oilUnitTotal int64
	for _, t := range troops {
		if cfg := ezfyCfg.troop(t.TroopId); cfg != nil {
			oilUnitTotal += int64(cfg.OilKeep) * t.Count
		}
	}
	return maxInt64(1, oilUnitTotal*int64(distance)/ezfyOilDivGrid)
}

func (h *EzfyHandler) beginReturn(order *model.EzfyOrder, now int64, travelSec int64) {
	travel := travelSec * 1000
	if travel <= 0 {
		travel = order.ArriveTime - order.StartTime
	}
	if travel <= 0 {
		travel = 60000
	}
	// 宿营: 到达后停留 wait_min 分钟再返航
	wait := int64(order.WaitMin) * 60000
	order.Status = 2
	order.Result = order.Troops
	order.ReturnTime = now + wait + travel
	// ★ Carry 必须一起落库，否则「随部队带回的资源」下次读库就丢了
	h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
		Updates(map[string]interface{}{"status": 2, "result": order.Troops,
			"return_time": order.ReturnTime, "carry": order.Carry})
}

// settleDispatch 派遣驻守采集结算(每8小时, 1/10 概率宝物)
func (h *EzfyHandler) settleDispatch(uid uint, order *model.EzfyOrder, now int64) {
	var wl model.EzfyWildland
	err := h.DB.First(&wl, order.TargetId).Error
	// 注意：这里不再需要 city（采集产出改为记在部队身上，返航到达才入城）
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
	// ★ 采集产出**先记在部队身上**（待带回），不直接入城；超出负重的部分丢弃。
	//   只有「召回并返航到达」才会入城（见 finishReturn）。
	loaded, dropped := h.addCarryToOrder(order, food, steel, oil, rare, gold)
	cur := parseCarry(order.Carry)
	desc := fmt.Sprintf("派遣部队在野地%d级(%d,%d)完成一次采集结算\n产出: 粮%d 钢%d 油%d 稀矿%d 金%d\n",
		level, wl.X, wl.Y, food, steel, oil, rare, gold)
	desc += fmt.Sprintf("本次装入部队: %d（负重 %d/%d）\n", loaded, cur.total(), h.ezfyCarryCap(order))
	if dropped > 0 {
		desc += fmt.Sprintf("⚠ 负重已满, %d 资源没能装上（多带运输兵/卡车可提高负重）\n", dropped)
	}
	desc += "资源要**召回部队**才能带回城里。\n"
	if rand.Intn(ezfyDispatchTreasure) == 0 {
		pool := []int{1, 4, 5, 6, 7, 8, 9}
		cfgId := pool[rand.Intn(len(pool))]
		if cfg := ezfyCfg.item(cfgId); cfg != nil {
			h.addItem(uid, cfgId, 1)
			// ★ 宝物不受负重限制，直接进背包
			desc += "运气爆棚! 获得宝物(已直接放入背包): " + cfg.Name + "\n"
			// ★ 系统消息（用户要求：采集出宝物要能看到）
			h.ezfySysChat("恭喜玩家 %s 在野地%d级(%d,%d)采集到宝物：%s",
				h.ezfyProfileName(uid), level, wl.X, wl.Y, cfg.Name)
		}
	}
	desc += "部队继续驻守采集, 可随时召回。"
	order.ArriveTime = now + ezfyDispatchPeriod
	order.Result = order.Troops
	// ★ 必须把 carry 一起落库 —— 否则「待带回资源」只存在于内存里，
	//   下一次请求重新读库就丢了（测试就是靠这条断言抓出来的）
	h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
		Updates(map[string]interface{}{"arrive_time": order.ArriveTime,
			"result": order.Result, "carry": order.Carry})
	h.addReport(uid, 5, "派遣报告: 采集结算", desc, "", order.ID)
}

func (h *EzfyHandler) processArrive(uid uint, order *model.EzfyOrder, now int64) {
	city := h.cityOfOrder(order, uid)

	// 活动目标(活动野地/活动寇城/特殊城市): 掠夺/征服走独立的活动战斗结算
	// (打赢只结算资源/黄金/宝物/声望, 不占领、不占附属野地上限)
	if order.OrderType == 2 || order.OrderType == 3 {
		if act := h.ezfyActTargetType(order.TargetX, order.TargetY); act > 0 {
			h.processActivityBattle(uid, city, order, now, act)
			return
		}
	}

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
		// ★ 用户规则：采集产出与带队军官属性挂钩 —— 后勤每 1 点 +1%，上限 +100%。
		gainPct := 100
		if o := h.officerByName(city.ID, order.Officer); o != nil {
			gainPct += o.Logistics
			if gainPct > 200 {
				gainPct = 200
			}
		}
		base := int64(level) * 800 * int64(gainPct) / 100
		var food, steel, oil, rare, gold int64
		if wl.WildType == 2 {
			oil, rare, gold = base, base, base
		} else {
			food, steel, oil, rare = base, base, base, base
		}
		// ★ 采到的资源装在部队身上，返航到达才入城；超负重丢弃
		loaded, dropped := h.addCarryToOrder(order, food, steel, oil, rare, gold)
		cur := parseCarry(order.Carry)
		travel := ezfyAbs64(order.ArriveTime - order.StartTime)
		order.Status = 2
		order.Result = order.Troops
		order.ReturnTime = now + travel
		h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
			Updates(map[string]interface{}{"status": 2, "result": order.Result,
				"return_time": order.ReturnTime, "carry": order.Carry})
		gdesc := fmt.Sprintf("我军在占领的野地采集了8小时\n产出: 粮%d 钢%d 油%d 稀矿%d 金%d\n", food, steel, oil, rare, gold)
		if gainPct > 100 {
			gdesc += fmt.Sprintf("军官后勤加成: +%d%%\n", gainPct-100)
		}
		gdesc += fmt.Sprintf("装入部队: %d（负重 %d/%d）\n", loaded, cur.total(), h.ezfyCarryCap(order))
		if dropped > 0 {
			gdesc += fmt.Sprintf("⚠ 负重已满, %d 资源没能装上\n", dropped)
		}
		// ★ 用户规则：宝物只能从「采集」获得 —— 每次采集有 1/10 概率捡到宝物(直接进背包, 不受负重限制)
		if rand.Intn(ezfyDispatchTreasure) == 0 {
			pool := []int{1, 4, 5, 6, 7, 8, 9}
			cfgId := pool[rand.Intn(len(pool))]
			if cfg := ezfyCfg.item(cfgId); cfg != nil {
				h.addItem(uid, cfgId, 1)
				gdesc += "运气爆棚! 获得宝物(已直接放入背包): " + cfg.Name + "\n"
				h.ezfySysChat("恭喜玩家 %s 在野地%d级(%d,%d)采集到宝物：%s",
					h.ezfyProfileName(uid), level, wl.X, wl.Y, cfg.Name)
			}
		}
		gdesc += "部队正在返回, 到达后资源入库。"
		h.addReport(uid, 5, fmt.Sprintf("采集报告: 野地%d级(%d,%d)", wl.Level, wl.X, wl.Y), gdesc, "", order.ID)
		return
	}

	// 运输: 向目标城市运送资源
	//
	// ★ 第九轮修复：
	//   1) 目标城仓储装不下的部分**原路带回**（记进 Carry，随部队返航入库），不再凭空蒸发；
	//   2) 目标城已消失时，整批资源原样带回；
	//   3) 部队（含同盟运输）一律返航回出发城市 —— beginReturn 会落库 Carry。
	if order.OrderType == 5 {
		res := h.parseResMap(order.Resources)
		f, s, o, r, g := res["food"], res["steel"], res["oil"], res["rare"], res["gold"]
		var target model.EzfyCity
		if err := h.DB.First(&target, order.TargetId).Error; err != nil {
			order.Carry = carryJSON(ezfyCarry{Food: f, Steel: s, Oil: o, Rare: r, Gold: g})
			h.beginReturn(order, now, 0)
			h.addReport(uid, 5, "运输报告: 目标城市不存在",
				fmt.Sprintf("运输目标城市已不存在, 运输部队已返航, 资源将随部队带回%s。", city.Name), "", order.ID)
			return
		}
		// 超出目标城仓储上限的部分原路带回
		overF := max64(0, target.Food+f-target.FoodCap)
		overS := max64(0, target.Steel+s-target.SteelCap)
		overO := max64(0, target.Oil+o-target.OilCap)
		overR := max64(0, target.Rare+r-target.RareCap)
		overG := max64(0, target.Gold+g-target.GoldCap)
		target.Food = min64(target.FoodCap, target.Food+f)
		target.Steel = min64(target.SteelCap, target.Steel+s)
		target.Oil = min64(target.OilCap, target.Oil+o)
		target.Rare = min64(target.RareCap, target.Rare+r)
		target.Gold = min64(target.GoldCap, target.Gold+g)
		h.saveCityRes(&target)
		order.Carry = carryJSON(ezfyCarry{Food: overF, Steel: overS, Oil: overO, Rare: overR, Gold: overG})
		desc := fmt.Sprintf("运输部队已到达%s\n", target.Name)
		if f+s+o+r+g > 0 {
			desc += fmt.Sprintf("送达: 粮%d 钢%d 油%d 稀矿%d 金%d\n", f-overF, s-overS, o-overO, r-overR, g-overG)
		}
		if overF+overS+overO+overR+overG > 0 {
			desc += fmt.Sprintf("⚠ 目标城仓储已满, 粮%d 钢%d 油%d 稀矿%d 金%d 将随部队带回\n",
				overF, overS, overO, overR, overG)
		}
		desc += "护送部队正在返航, 到达后回到出发城市。"
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
		// 随军军官调任到目标城市(职位清空); 盟友驻军时军官留在本城
		if order.Officer != "" {
			if h.isOwnCity(uid, int64(target.ID)) {
				h.moveOfficerTo(city, order.Officer, target.ID)
				desc += "\n军官 " + order.Officer + " 随军抵达"
			} else {
				h.officerGoOut(city, order.Officer, false)
				desc += "\n军官 " + order.Officer + " 护送完成后返回本城"
			}
		}
		h.addReport(uid, 5, "增援报告: "+target.Name, desc)
		if target.UserID > 0 && target.UserID != uid {
			h.addReport(target.UserID, 5, "增援到达: "+city.Name,
				city.Name+"的增援部队已抵达并驻防!\n"+desc)
		}
		return
	}

	// 派遣(8): 把部队 / 军官 / 随军资源送到自己的另一座城市（城际调兵）
	//
	// ★ 用户规则：「派遣也是出征」—— 目标必须是自己名下的城市。
	if order.OrderType == 8 {
		var target model.EzfyCity
		if err := h.DB.First(&target, order.TargetId).Error; err != nil || target.UserID != uid {
			h.beginReturn(order, now, 0)
			h.addReport(uid, 5, "派遣报告: 目标城市无效",
				"派遣目标城市不存在或已不属于我方, 派遣部队与资源已原路带回。", "", order.ID)
			return
		}
		troops := parseGroups(order.Troops)
		desc := fmt.Sprintf("派遣部队已抵达%s(%d,%d)\n", target.Name, target.X, target.Y)
		for _, g := range troops {
			if g.Count <= 0 {
				continue
			}
			h.addTroop(target.ID, g.TroopId, g.Count)
			desc += ezfyCfg.troopName(g.TroopId, 0) + "×" + strconv.FormatInt(g.Count, 10) + " "
		}
		// 随军资源入目标城（受仓储上限截断，装不下的部分随部队原路带回）
		res := h.parseResMap(order.Resources)
		back := ezfyCarry{}
		total := res["food"] + res["steel"] + res["oil"] + res["rare"] + res["gold"]
		if total > 0 {
			// ★ 只做「资源懒结算」，**不能**调 refreshCity ——
			//   refreshCity 会 processOrders(uid)，而本订单正是该 uid 下
			//   status=0 且已到期的订单 → 会再次进到这里，无限递归把服务打死。
			//   （processOrders 现在另有 per-uid 重入守卫，这里是第二道防线。）
			//   军官列表显式传入，工资才扣得对（calcResource 自己不查库）。
			h.calcResource(&target, h.officerList(target.ID))
			room := func(cur, cap, v int64) (int64, int64) {
				if cap <= 0 {
					cap = cur
				}
				r := cap - cur
				if r < 0 {
					r = 0
				}
				if v > r {
					return r, v - r
				}
				return v, 0
			}
			var put int64
			put, back.Food = room(target.Food, target.FoodCap, res["food"])
			target.Food += put
			put, back.Steel = room(target.Steel, target.SteelCap, res["steel"])
			target.Steel += put
			put, back.Oil = room(target.Oil, target.OilCap, res["oil"])
			target.Oil += put
			put, back.Rare = room(target.Rare, target.RareCap, res["rare"])
			target.Rare += put
			put, back.Gold = room(target.Gold, target.GoldCap, res["gold"])
			target.Gold += put
			h.saveCityRes(&target)
			desc += fmt.Sprintf("\n随军资源已入库: 粮%d 钢%d 油%d 稀矿%d 金%d",
				res["food"]-back.Food, res["steel"]-back.Steel, res["oil"]-back.Oil,
				res["rare"]-back.Rare, res["gold"]-back.Gold)
			if back.total() > 0 {
				desc += fmt.Sprintf("\n⚠ 目标城仓储已满, %d 资源随部队原路带回", back.total())
			}
		}
		// 随军军官调任目标城市（清空职位）
		if order.Officer != "" {
			h.moveOfficerTo(city, order.Officer, target.ID)
			desc += "\n军官 " + order.Officer + " 随军调往" + target.Name
		}
		if back.total() > 0 {
			// 兵力已经进城，回程只带「装不下的资源」：Result 置空避免兵力重复入账
			travel := ezfyAbs64(order.ArriveTime - order.StartTime)
			if travel <= 0 {
				travel = 60000
			}
			order.Status = 2
			order.Result = ""
			order.Carry = carryJSON(back)
			order.ReturnTime = now + travel
			h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
				Updates(map[string]interface{}{"status": 2, "result": "", "carry": order.Carry,
					"return_time": order.ReturnTime})
		} else {
			order.Status = 3
			h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
				Updates(map[string]interface{}{"status": 3, "carry": ""})
		}
		h.addReport(uid, 5, "派遣报告: "+target.Name, desc, "", order.ID)
		return
	}

	// ============ 战斗类: 侦查/掠夺/征服 ============
	attacker := parseGroups(order.Troops)
	atkTech := h.techMap(city.ID)
	// 带队军官(军事属性 + 装备 + 技能)与科技加成
	leadOfficer := h.officerByName(city.ID, order.Officer)
	officerBonus := h.officerBattleBonus(leadOfficer)
	atkBonus := officerBonus + atkTech[5]*2 + atkTech[6]*3 + atkTech[8]*3 + atkTech[9]*2
	atkSpeedBonus := atkTech[10]*2 + atkTech[19]*3
	if h.officerSpeedSkill(leadOfficer) {
		atkSpeedBonus += 10
	}
	if h.officerSpeedSkill(leadOfficer) {
		atkSpeedBonus += 10
	}
	atkOfficerDesc := h.officerBattleDesc(leadOfficer, h.officerBaseBonus(leadOfficer), "攻击加成")
	// ★ 装备六项战斗加成（伤害/防御/生命/移动距离/暴击几率/暴击伤害）
	atkEquip := h.officerBattleEquipBonus(leadOfficer)
	// 城守(仅玩家城市防守方)
	var cityGuard *model.EzfyOfficer
	defEquip := ezfyBattleBonus{}
	defOfficerDesc := ""

	defender := []ezfyUnitGroup{}
	targetName := ""
	var lootFood, lootSteel, lootOil, lootRare, lootGold int64
	win := false
	defBonus := 0
	defSpeedBonus := 0
	wildLevel := 0
	var target *model.EzfyCity

	// ★ case 0：老数据/异常请求可能没带 target_type，按「野地」处理，
	//   否则会落进 default，导致战报标题变成「侦查报告: 」（目标名为空）。
	switch order.TargetType {
	case 0, 1, 2:
		// 活动目标: 侦查时按活动守军回报情报(不走普通野地配置表)
		if act := h.ezfyActTargetType(order.TargetX, order.TargetY); act > 0 {
			wildLevel = ezfyActivityLevel(order.TargetX, order.TargetY)
			defender = ezfyActivityDefender(act, wildLevel, ezfyTerrain(order.TargetX, order.TargetY))
			targetName = ezfyActTargetLabel(act, wildLevel)
			break
		}
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
		// ★ 用户要求：战报里的野地要标出**具体地形类型**（丘陵/沼泽/平原…），
		//   原来一律写「野地N级」，看不出打的是什么地形。
		name := ezfyTerrainName(ezfyTerrain(order.TargetX, order.TargetY))
		if order.TargetType == 2 {
			name = "寇城"
		} else if ezfyTerrain(order.TargetX, order.TargetY) == 8 {
			name = "海野"
		}
		targetName = name + strconv.Itoa(level) + "级"
		rnd := cfg.ResMin + rand.Int63n(cfg.ResMax-cfg.ResMin+1)
		lootTech := atkTech[17] * 2
		if h.officerHasSkill(leadOfficer, "黄金眼") {
			lootTech += 10
		}
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
		// ★ 结算前先把防守方城市懒结算到当前（建筑完工/资源产出/民心回复/训练完成），
		//   否则用的是「防守方上次登录时」的陈旧民心与资源 —— 民心偏低会让征服异常容易。
		//   ★ 防递归：refreshCity → processOrders 现在有 per-uid 重入守卫
		//   （见 processOrders 函数头），防守方正在结算中会直接返回，不会自喂。
		//   防守方 == 自己时仍然只做资源懒结算，少绕一圈。
		if target.UserID != uid {
			h.refreshCity(target.UserID, target)
		} else {
			h.calcResource(target, h.officerList(target.ID))
		}
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
		// 城守: 守城防御 +10% 及 防御/掩体/生命/鼓舞技能
		cityGuard = h.positionOfficer(target.ID, ezfyPositionGuard)
		defBonus += h.officerGuardBonus(cityGuard)
		defEquip = h.officerBattleEquipBonus(cityGuard)
		defOfficerDesc = h.officerBattleDesc(cityGuard, 10, "守军防御")
	}

	// 侦查: 不战斗只报告情报, 部队随即返航
	// 报告格式复刻 `参考材料/开发文档/侦察报告1.txt`:
	//   玩家城市 → 资源数量/人口民心/建筑等级/城防数量/军队数量/将领等级/科技等级/最后活动时间
	//   野地寇城 → 守军情况
	if order.OrderType == 1 {
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
			h.scoutReportBody(uid, order, targetName, target, defender), "", order.ID)
		return
	}

	br := ezfySimulate(attacker, defender, atkBonus, defBonus, atkSpeedBonus, defSpeedBonus,
		atkEquip, defEquip,
		atkOfficerDesc, defOfficerDesc, h.buildTargetMap(city.ID, true), h.buildTargetMap(cityIdOf(target), false),
		h.buildMoveMap(city.ID, true), h.buildMoveMap(cityIdOf(target), false))
	win = br.AttackerWin

	// ★★ 军官经验结算（2026-09-21 重做）
	//
	// 用户规则：
	//   ① 出征的**攻方**与**守方**军官都要拿到经验（原来攻方只在打赢时给）；
	//   ② **胜利一方拿得更多**（胜负加成拉开差距，鼓励打胜仗）；
	//   ③ 战损也算贡献（打得多、损耗大，经验相应多）。
	//
	// 统一口径（攻守共用同一个函数，避免两边公式再漂移）：
	//
	//	基础经验 = 击杀敌军数 / 10 + 参战基数 30
	//	胜利加成 = +50%（赢的一方额外多拿一半）
	//
	// 注：攻方原来写死 myDead/10 + 50（按自己的战损算），语义是「越惨越有经验」，
	// 与「胜利拿更多」相悖，这里一并改成按**击杀**计算（把对方打死才有战功）。
	defExp := int64(0)
	// 守方：拿到城守军官的城才有（野地/寇城无军官，自然跳过）
	if cityGuard != nil && target != nil {
		defExp = ezfyOfficerBattleExp(enemyDeadOf(br.DefenderLosses), !win)
		h.addOfficerExp(target, cityGuard.ID, defExp)
	}
	// 攻方：带队军官无论胜负都给经验（原来只在 if win 分支里给，
	// 打输的部队回来军官一点经验都没有，与用户规则①不符）。
	// ★ 这里只算数值、不写库 —— 写库放在下面战报正文里做：
	//   胜/败两个分支各自往战报追加「军官经验+N」后再调用 addOfficerExp。
	atkExp := int64(0)
	if leadOfficer != nil {
		atkExp = ezfyOfficerBattleExp(enemyDeadOf(br.DefenderLosses), win)
	}

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
	// ★ 用户要求：战报里的兵种名用「阵营兵种名」(同盟国/轴心国各自的叫法)，
	//   不再是笼统的大类名。攻方用攻方阵营，守方用守方阵营。
	atkCamp := profile.Camp
	defCamp := 0
	if target != nil {
		defCamp = h.ensureProfile(target.UserID).Camp
	}
	// 带队军官 / 城守军官
	if leadOfficer != nil {
		report += "军官:" + officerReportDesc(leadOfficer) + "\n"
	}
	if cityGuard != nil {
		report += "守军军官:" + officerReportDesc(cityGuard) + "\n"
	}

	atkBefore := groupCounts(attacker)
	atkAfter := groupCounts(br.AttackerLeft)
	report += fmt.Sprintf("[%s]攻方:%s\n", winText(win), city.Name)
	report += troopChangeText(atkBefore, atkAfter, atkCamp)
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
	report += troopChangeText(defBefore, defAfter, defCamp)

	detail := ""
	for _, a := range br.Actions {
		detail += a + "\n"
	}
	detail += "\n[双方兵力]\n"
	detail += fmt.Sprintf("[%s]攻方:%s\n", winText(win), city.Name)
	detail += troopChangeText(atkBefore, atkAfter, atkCamp)
	detail += fmt.Sprintf("--------------------\n[%s]守方:%s\n", winText(!win), targetName)
	detail += troopChangeText(defBefore, defAfter, defCamp)
	detail += "[双方兵力]"

	// 攻方战损: 按修复率入伤兵营
	losses := br.AttackerLosses
	var deadCount int64
	for _, g := range losses {
		deadCount += g.Count
	}
	healTech := atkTech[21] * 2
	// 带队军官「修养」技能: 战后伤兵恢复 +10%
	if h.officerHasSkill(leadOfficer, "机械改造") {
		healTech += 10
	}
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
	wareNote := ""
	recyclePct := 0 // 战报里的「回收比例」(玩家城按掠夺比例)
	// ★ 用户反馈：野地/寇城战报里「回收比例:0%」看着像 bug。
	//   野地没有仓库保护额度，战利品是整份资源 → 回收比例就是 100%。
	if win && (order.TargetType == 1 || order.TargetType == 2) {
		recyclePct = 100
	}
	prestigeGain := 0

	if win {
		if targetProtected && order.TargetType == 3 {
			report += "\n目标城市处于免战保护期, 无法掠夺资源!"
		}
		if order.TargetType == 3 && target != nil {
			// 玩家城市: 掠夺比例10%+掠夺技巧, 上限50%
			lootRate := 10 + atkTech[17]*2
			if h.officerHasSkill(leadOfficer, "黄金眼") {
				lootRate += 10
			}
			if targetProtected || order.OrderType != 2 && order.OrderType != 3 {
				lootRate = 0
			}
			if lootRate > 50 {
				lootRate = 50
			}
			recyclePct = lootRate
			defRes := []int64{target.Food, target.Steel, target.Oil, target.Rare, target.Gold}
			loot := make([]int64, 5)
			var totalLoot int64
			for i := 0; i < 5; i++ {
				loot[i] = defRes[i] * int64(lootRate) / 100
				totalLoot += loot[i]
			}
			// 仓库保护: 目标仓库等级决定各项资源保护额度, 保护额度内的资源不可掠夺
			if order.OrderType == 2 || order.OrderType == 3 {
				prot, note := h.lootAfterWareProtect(target,
					[4]int64{target.Food, target.Steel, target.Oil, target.Rare},
					[4]int64{loot[0], loot[1], loot[2], loot[3]})
				loot[0], loot[1], loot[2], loot[3] = prot[0], prot[1], prot[2], prot[3]
				wareNote = note
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
				report += h.battleStatsTail(uid, pg, 0)
				h.addReport(uid, 2, reportType+": "+targetName, report, detail, order.ID)
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
			// ★ 用户反馈「征服民心每次 -5 现在太多」→ 单次扣多少改为管理端可配
			//   （ezfy_cfg_limit.conquer_feelings_max，默认 2）。
			//   原来封顶写死 20，且按「幸存兵力/2000」动态算，大兵团一次就能清零民心。
			//   现在既保留动态计算（小部队扣得少），又用配置值封顶。
			if cm := ezfyConquerFeelingsCfg(); feelingDrop > cm {
				feelingDrop = cm
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
				report += fmt.Sprintf("\n征服战果\n黄金:%d\n粮食:%d\n钢铁:%d\n石油:%d\n稀矿:%d", lootGold, lootFood, lootSteel, lootOil, lootRare)
				city.Food += lootFood
				city.Steel += lootSteel
				city.Oil += lootOil
				city.Rare += lootRare
				city.Gold += lootGold
				h.saveCityRes(city)
				report += h.battleStatsTail(uid, 0, recyclePct)
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
				report += h.battleStatsTail(uid, 0, recyclePct)
				h.addReport(uid, 3, "征服报告: "+targetName, report, detail, order.ID)
				h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
					Updates(map[string]interface{}{"status": order.Status, "result": order.Result, "return_time": order.ReturnTime})
				return
			}
			report += "\n民心已失，征服成功!"
			target.Feelings = 0
			target.Grievance = minInt(100, target.Grievance+50)
			// ★ 城破后按剩余资源的 50% 再掠夺一次，回收比例与之一致（原来显示的是普通掠夺比例）
			recyclePct = 50
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
					targetName, lootFood, lootSteel, lootOil, lootRare, lootGold, lossText(br.DefenderLosses, defCamp))
			} else {
				defReportBody = fmt.Sprintf("敌方部队攻破了你的城市%s!\n民心清零, 但该城市是你的最后一座城, 无法被占领!\n被掠夺资源: 粮%d 钢%d 油%d 稀矿%d 金%d\n%s",
					targetName, lootFood, lootSteel, lootOil, lootRare, lootGold, lossText(br.DefenderLosses, defCamp))
			}
			h.addReport(target.UserID, 4, "城破报告: "+city.Name, defReportBody, detail)
		}
		// 普通掠夺(含成功掠夺玩家城市): 民心-N 民怨+N
		// ★ 用户反馈「民心每次 -5 现在太多」→ 扣多少改为管理端可配
		//   （ezfy_cfg_limit.loot_feelings，默认 2）。
		if order.OrderType == 2 && order.TargetType == 3 && target != nil {
			lootFeel := ezfyLootFeelingsCfg()
			target.Feelings = maxInt(0, target.Feelings-lootFeel)
			target.Grievance = minInt(100, target.Grievance+lootFeel)
			h.saveCityRes(target)
			h.DB.Model(&model.EzfyCity{}).Where("id = ?", target.ID).
				Updates(map[string]interface{}{"feelings": target.Feelings, "grievance": target.Grievance})
			h.addReport(target.UserID, 2, "被掠夺报告: "+city.Name,
				fmt.Sprintf("你的城市%s被敌方部队掠夺!\n被掠夺资源: 粮%d 钢%d 油%d 稀矿%d 金%d\n民心-%d 民怨+%d\n%s\n%s",
					targetName, lootFood, lootSteel, lootOil, lootRare, lootGold,
					lootFeel, lootFeel, lossText(br.DefenderLosses, defCamp), wareNote),
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
		// 攻打玩家城市: 目标城军官忠诚下降, 归零者弃城成为我方战俘
		// (复刻用户说明的 PvP 战俘来源: 把对方军官忠诚打成 0)
		if (order.OrderType == 2 || order.OrderType == 3) && order.TargetType == 3 && target != nil {
			if frag := h.defectDefenderOfficers(city, target, uid); frag != "" {
				report += frag
			}
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
		// ★ 用户规则：**宝物只能通过「采集」获得** —— 打野地/寇城不再掉落装备与珠宝。
		//   （原来这里调 wildlandLoot 掉宝，属于 bug；现只保留「俘虏守将」）
		if (order.TargetType == 1 || order.TargetType == 2) && wildLevel >= 1 {
			// 该野地/寇城配置里有军官才可能俘到(没军官就什么都没有)
			wt := 1
			if order.TargetType == 2 {
				wt = 3
			} else if ezfyTerrain(order.TargetX, order.TargetY) == 8 {
				wt = 2
			}
			if cap := h.captureWildlandOfficer(city, wt, wildLevel, false); cap != "" {
				report += "\n" + cap
			}
		}
		var enemyDead int64
		for _, g := range br.DefenderLosses {
			enemyDead += g.Count
		}
		if enemyDead > 0 {
			h.taskProgress(uid, "kill_enemy", int(enemyDead))
		}
		// 带队军官战功经验: 见函数开头的统一口径（攻守共用 ezfyOfficerBattleExp）。
		// ★ 原来是 myDead/10 + 50（按自己战损算），语义是「越惨越有经验」，
		//   与用户规则「胜利方拿更多」相悖，已改为按击杀数算并叠加胜利加成。
		if leadOfficer != nil {
			h.addOfficerExp(city, leadOfficer.ID, atkExp)
			report += fmt.Sprintf("\n军官经验+%d", atkExp)
		}
		travel := ezfyAbs64(order.ArriveTime - order.StartTime)
		order.Status = 2
		order.ReturnTime = now + travel
		if wareNote != "" {
			report += "\n" + wareNote
		}
		report += fmt.Sprintf("\n战果\n黄金:%d\n粮食:%d\n钢铁:%d\n石油:%d\n稀矿:%d", lootGold, lootFood, lootSteel, lootOil, lootRare)
		if repairedTotal > 0 {
			report += fmt.Sprintf("\n伤兵入营: %d(可前往司令部伤兵营恢复)", repairedTotal)
		}
		report += h.battleStatsTail(uid, prestigeGain, recyclePct)
		h.addReport(uid, 2, reportType+": "+targetName, report, detail, order.ID)
	} else {
		// ★ 第九轮：打败仗 → 幸存部队撤退返航（原来 status=4 是终止态，
		//   幸存兵力凭空消失、带队军官永远卡在「出征中」，属于 bug）。
		travel := ezfyAbs64(order.ArriveTime - order.StartTime)
		order.Status = 2
		order.ReturnTime = now + travel
		if repairedTotal > 0 {
			report += fmt.Sprintf("\n伤兵入营: %d(可前往司令部伤兵营恢复)", repairedTotal)
		}
		// ★ 军官忠诚：只有打败仗才掉，且按战损比例合理计算（基础 3 点，全灭 10 点）
		if leadOfficer != nil {
			var myDead, myTotal int64
			for _, g := range br.AttackerLosses {
				myDead += g.Count
			}
			for _, g := range parseGroups(order.Troops) {
				myTotal += g.Count
			}
			delta := ezfyLoyaltyOnDefeat
			if myTotal > 0 {
				delta += int(float64(ezfyLoyaltyOnDefeat*2) * float64(myDead) / float64(myTotal))
			}
			if delta > 10 {
				delta = 10
			}
			h.officerLoseLoyalty(uid, city.ID, leadOfficer.Name, delta, "")
			report += fmt.Sprintf("\n带队军官 %s 因战败忠诚度-%d", leadOfficer.Name, delta)
		}
		// ★ 用户规则①：**打输也要给经验**（原来只在 if win 分支给，
		//   败仗回来军官经验一点不动）。数值已在函数开头按统一口径算好，
		//   败方拿的是「无胜利加成」的基础经验，天然少于胜方。
		if leadOfficer != nil && atkExp > 0 {
			h.addOfficerExp(city, leadOfficer.ID, atkExp)
			report += fmt.Sprintf("\n军官经验+%d", atkExp)
		}
		report += "\n残部正在撤退返航。"
		report += h.battleStatsTail(uid, prestigeGain, recyclePct)
		h.addReport(uid, 2, reportType+": "+targetName, report, detail, order.ID)
		if order.TargetType == 3 && target != nil {
			h.addReport(target.UserID, 4, "守卫报告: "+city.Name,
				fmt.Sprintf("你的城市%s成功抵挡了敌方部队的进攻!\n%s", targetName, lossText(br.DefenderLosses, defCamp)), detail)
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
//
// ★ 用户要求「加个野地兵力倍数配置，默认 1，可以调整倍数」→ 随机出来的数量再乘倍数。
// 野地详情里的守军预览走同一个倍数（见 MapWildland），保证「看到的」=「打到的」。
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
		groups = append(groups, ezfyUnitGroup{TroopId: tid, Count: ezfyScaleByWildMult(count)})
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

// troopChangeText 兵力变化文本；camp 为阵营(1 同盟国 / 2 轴心国)，
// 用于取「阵营兵种名」(ezfy_cfg_troop.name_ally / name_axis)。
func troopChangeText(before, after map[int]int64, camp int) string {
	ids := []int{}
	for tid := range before {
		ids = append(ids, tid)
	}
	// 按兵种 id 排序, 避免 Go map 随机遍历导致战报里兵种顺序每次都变
	sort.Ints(ids)
	text := ""
	for _, tid := range ids {
		name := ezfyCfg.troopName(tid, camp)
		if name == "" {
			name = "兵种" + strconv.Itoa(tid)
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

// lossText 守军损失文本；camp 为守方阵营(1 同盟国 / 2 轴心国)
func lossText(groups []ezfyUnitGroup, camp int) string {
	if len(groups) == 0 {
		return "守军无损失"
	}
	text := "守军损失: "
	for _, g := range groups {
		name := ezfyCfg.troopName(g.TroopId, camp)
		if name == "" {
			name = "兵种" + strconv.Itoa(g.TroopId)
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
// scoutReportBody 侦查报告正文
// 玩家城市给出完整情报(资源/人口民心/建筑等级/城防/军队/将领/科技/最后活动时间),
// 野地与寇城只给守军情况。格式取自 `参考材料/开发文档/侦察报告1.txt`。
func (h *EzfyHandler) scoutReportBody(uid uint, order *model.EzfyOrder, targetName string,
	target *model.EzfyCity, defender []ezfyUnitGroup) string {
	var b strings.Builder
	fmt.Fprintf(&b, "公文报告:侦查报告\n我方一支部队对%s[%d，%d]进行了侦查。侦查过程中未受到任何阻拦。\n",
		targetName, order.TargetX, order.TargetY)

	// 野地 / 寇城: 只有守军
	if target == nil {
		b.WriteString("守军情况: ")
		if len(defender) == 0 {
			b.WriteString("无敌军驻守")
		}
		for _, g := range defender {
			if cfg := ezfyCfg.troop(g.TroopId); cfg != nil {
				fmt.Fprintf(&b, "%s×%d ", cfg.Name, g.Count)
			}
		}
		b.WriteString("\n侦查完成, 部队已返航。")
		return b.String()
	}

	// 玩家城市: 完整情报
	fmt.Fprintf(&b, "资源数量 粮食%d 钢铁%d 石油%d 稀矿%d 黄金%d\n",
		target.Food, target.Steel, target.Oil, target.Rare, target.Gold)
	fmt.Fprintf(&b, "人口%d 民心%d\n", target.Pop, target.Feelings)

	// 建筑等级: 按建筑 id 排序, 同类多座依次列出
	names := map[int]string{}
	levels := map[int][]int{}
	ids := []int{}
	for _, cb := range h.buildingList(target.ID) {
		if _, ok := levels[cb.BuildingId]; !ok {
			ids = append(ids, cb.BuildingId)
			if cfg := ezfyCfg.building(cb.BuildingId); cfg != nil {
				names[cb.BuildingId] = cfg.Name
			}
		}
		levels[cb.BuildingId] = append(levels[cb.BuildingId], cb.Level)
	}
	sort.Ints(ids)
	b.WriteString("建筑等级 ")
	for _, id := range ids {
		b.WriteString(names[id])
		for _, lv := range levels[id] {
			fmt.Fprintf(&b, "%d,", lv)
		}
	}
	b.WriteString("\n")

	// 城防 / 军队分列
	defText, armyText := "", ""
	for tid, cnt := range h.troopMap(target.ID) {
		cfg := ezfyCfg.troop(tid)
		if cfg == nil || cnt <= 0 {
			continue
		}
		if cfg.Type == 4 {
			defText += fmt.Sprintf("%s×%d ", cfg.Name, cnt)
		} else {
			armyText += fmt.Sprintf("%s%d ", cfg.Name, cnt)
		}
	}
	b.WriteString("城防数量：" + defText + "\n")
	b.WriteString("军队数量：" + armyText + "\n")

	// 将领等级
	officers := h.officerList(target.ID)
	offText := ""
	for _, o := range officers {
		offText += fmt.Sprintf("%s(%d级)、", o.Name, o.Level)
	}
	b.WriteString("将领等级：" + offText + "\n")

	// 科技等级
	techText := ""
	techIDs := []int{}
	tm := h.techMap(target.ID)
	for id := range tm {
		techIDs = append(techIDs, id)
	}
	sort.Ints(techIDs)
	for _, id := range techIDs {
		if cfg := ezfyCfg.tech(id); cfg != nil {
			techText += fmt.Sprintf("%s%d ", cfg.Name, tm[id])
		}
	}
	b.WriteString("科技等级：" + techText + "\n")

	lastActive := ""
	if target.UpdatedAt.Unix() > 0 {
		lastActive = target.UpdatedAt.Format("2006-01-02 15:04:05")
	}
	b.WriteString("最后活动时间：" + lastActive + "\n")
	b.WriteString("侦查完成。")
	return b.String()
}

// enemyDeadOf 汇总一组战损里的兵力总数（击杀数）。
func enemyDeadOf(losses []ezfyUnitGroup) int64 {
	var n int64
	for _, g := range losses {
		n += g.Count
	}
	return n
}

// ezfyOfficerBattleExp 出征军官战斗经验（攻守双方**共用**同一个口径）。
//
// ★★ 2026-09-21 用户规则重做：
//
//	① 攻方与守方军官都要拿到经验（原来攻方只在打赢时给）；
//	② 胜利的一方拿得更多；
//	③ 战功按「击杀敌军数」衡量（把对方打死才有战功）。
//
// 公式：
//
//	基础 = 击杀数 / 10 + 参战基数 30
//	胜方 = 基础 × 1.5（向下取整）
//
// 效果对比（击杀 100 → 基础 40）：胜方 60，败方 40。差距明显但不夸张，
// 败方也确有收获，与「打赢更有价值」的直觉一致。
//
// 另：原实现是攻方按**自己战损**算（myDead/10 + 50），会造成
// 「被全歼的经验反而最高」这种反直觉结果，故一并改掉。
func ezfyOfficerBattleExp(enemyDead int64, won bool) int64 {
	if enemyDead < 0 {
		enemyDead = 0
	}
	exp := enemyDead/10 + 30
	if won {
		exp = exp * 3 / 2
	}
	return exp
}

// battleStatsTail 战报尾部的战果统计段
// (复刻 `参考材料/开发文档/掠夺报告1.txt` 的 个人荣誉/个人战绩/军团战绩/回收比例 + [双方兵力];
// 军功声望与军官经验已在上文正文里输出, 这里不重复)
func (h *EzfyHandler) battleStatsTail(uid uint, prestigeGain, recyclePct int) string {
	profile := h.ensureProfile(uid)
	return fmt.Sprintf("\n个人荣誉:%d\n个人战绩:%d\n军团战绩:%d\n回收比例:%d%%\n[双方兵力]",
		profile.Prestige/6, prestigeGain, 0, recyclePct)
}

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
