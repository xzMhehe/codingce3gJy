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
	"gorm.io/gorm"

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
					lvl := ezfyWildlandLevel(x, y)
					if lvl == 0 {
						// 纯海洋: 无野地, 详情页不显示守军/军官/出征按钮
						cell["name"] = ezfyTerrainName(terrain) // 海洋
						cell["is_ocean"] = true
					} else {
						cell["name"] = "海底森林"
						cell["level"] = lvl
					}
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
	// ★ 纯海洋(海野等级0): 只显示「地形：海洋」, 无守军/军官/出征按钮
	if ttype == 2 && ezfyWildlandLevel(x, y) == 0 {
		resp.OK(c, gin.H{"x": x, "y": y, "type": 0, "is_ocean": true,
			"terrain": 8, "terrain_name": "海洋", "continent": ezfyRegionName(x, y)})
		return
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
	// ★ 采集可获得(按地形固定): 资源名 + 宝物池（平原/沿海平原无珠宝）
	var treasures []string
	gatherRes := ""
	if ttype == 1 || ttype == 2 {
		te := ezfyTerrainEx(x, y)
		treasures = ezfyTerrainTreasureNames[te]
		gatherRes = ezfyGatherResName(te)
	}
	if treasures == nil {
		treasures = []string{}
	}
	// 地形显示名: 海野→海底森林, 寇城→平原(用户规范)
	terrainName := ezfyTerrainNameEx(x, y)
	if ttype == 2 {
		terrainName = "海底森林"
	} else if ttype == 3 {
		terrainName = "平原"
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
		"res_min": cfg.ResMin, "res_max": cfg.ResMax, "terrain": ezfyTerrainEx(x, y),
		"terrain_name": terrainName,
		"continent":    ezfyRegionName(x, y),
		"treasures":    treasures,
		"gather_res":   gatherRes,
		"treasure":     cfg.Treasure, // 寇城宝物档次(初级/中级/高级)
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
		// ★ 出征速度加成（与 createOrder 同口径，保证预览与实际一致）
		if b := ezfyMarchSpeedBonus(); b > 0 {
			travelSec = int64(float64(travelSec) * 100 / (100 + b))
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
	// ★ 防抖幂等(2026-09-24 用户反馈「出征了显示多条」)：
	//   网络超时/连点/客户端重发会让同一次出征重复下单。
	//   3 秒内同「城市+类型+目标」的订单视为重复提交，直接拒绝。
	//   （正常情况下一次作战结束后 3 秒内对同一目标重复出征几乎不可能；
	//   侦查→掠夺等不同 order_type 不受影响）
	{
		var recent int64
		h.DB.Model(&model.EzfyOrder{}).
			Where("user_id = ? AND city_id = ? AND order_type = ? AND target_id = ? AND target_x = ? AND target_y = ? AND start_time >= ?",
				uid, city.ID, orderType, targetId, targetX, targetY, time.Now().UnixMilli()-3000).
			Count(&recent)
		if recent > 0 {
			return "命令已下达, 请勿重复出征"
		}
	}
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
		// ★ 2026-09-24 俘虏出征 bug 加固：只有 Status=0(在职) 的军官才能带队，
		//   Status=1(出征中)/2(被俘) 一律拒绝（历史数据里被俘军官可能 IsCaptive=0, 单看字段会漏拦）。
		if lead.Status != 0 || h.officerBusyOrder(city.ID, officer) {
			return "军官" + lead.Name + "正在出征中, 未归队前不能再次出征"
		}
		if lead.IsCaptive == 1 {
			return "俘虏不能带队出征, 请先在军校收编"
		}
		// ★ 市长/城守不得带队出征（城务在身），需先卸任
		if lead.Position != 0 {
			return "「" + ezfyPositionName(lead.Position) + "」" + lead.Name + "有城务在身, 请先卸任再出征"
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
	// ★ 出征速度加成（管理端「二战系统配置」可配）：节假日调高让队伍走快点
	if b := ezfyMarchSpeedBonus(); b > 0 {
		travelSec = int64(float64(travelSec) * 100 / (100 + b))
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
	// ★ 用户规则：出征队列只列**还在外面**的部队（行军中/驻守中/返航中/战斗中/等待）。
	//   已结束(3已完成/4已终止)的命令不再常驻队列，战报里还能查到。
	h.DB.Where("user_id = ? AND status IN (0,1,2,?,?)", uid, ezfyOrderStatusBattle, ezfyOrderStatusWaiting).
		Order("id DESC").Limit(50).Find(&orders)
	// ★ 战斗中的订单要带上回合进度：**一次查出全部战场**再按 order_id 取，
	//   别在循环里逐条查（性能红线：1核1G 机器上 N+1 会直接打满）。
	//   而且**只在真的有战斗中订单时才查** —— 绝大多数请求没有战斗，不该多打一次 DB。
	battleRounds := map[int64]int{}
	hasBattle := false
	for i := range orders {
		if orders[i].Status == ezfyOrderStatusBattle {
			hasBattle = true
			break
		}
	}
	if hasBattle {
		var battles []model.EzfyBattle
		h.DB.Where("user_id = ? AND status = 1", uid).Find(&battles)
		for _, b := range battles {
			// ★ 回合从 1 开始展示（第 1 回合不得显示 0），与指挥室口径一致
			battleRounds[b.OrderId] = maxInt(b.Round, 1)
		}
	}
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
			// 指挥室：可指挥时前端显示 [指挥]
			"can_command":  o.Status == ezfyOrderStatusBattle,
			"battle_round": battleRounds[int64(o.ID)],
			"battle_max":   ezfyBattleMaxRounds,
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
	// ★ 指挥室：战斗中的部队不能召回 —— 让它先打完（或点「自动战斗」一键打完）
	if order.Status == ezfyOrderStatusBattle {
		resp.ParamError(c, "部队正在战斗中, 不能取消；请到「军情 → 军队动态 → [指挥]」里打完或点[自动战斗]")
		return
	}
	if order.Status != 0 && order.Status != 1 {
		resp.ParamError(c, "该命令已在返航中或已结束, 无法取消")
		return
	}
	now := time.Now().UnixMilli()
	// ★ 驻守采集召回: 先结算产出 —— 满一个采集周期结算资源+宝物, 提前召回只有按比例的资源(无宝物)
	if order.Status == 1 && order.OrderType == 7 {
		h.settleDispatchOnRecall(uid, &order, now)
		if order.Status != 1 {
			resp.OK(c, gin.H{"msg": "采集野地已丢失, 部队已自动返航"})
			return
		}
	}
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
	// ★ 死单自愈：status=98(结算中) 超过 60 秒没被写回正常状态的订单，
	//   说明结算过程异常退出（老部署强杀进程等），重置回「行进」，下次到达再结算。
	h.DB.Model(&model.EzfyOrder{}).
		Where("user_id = ? AND status = ? AND updated_at < ?", uid, ezfyOrderStatusProcessing,
			time.Now().Add(-60*time.Second)).
		Updates(map[string]interface{}{"status": 0})
	var orders []model.EzfyOrder
	h.DB.Where("user_id = ?", uid).Order("id ASC").Find(&orders)
	for i := range orders {
		order := &orders[i]
		// ★ 指挥室：战斗中的订单先推进战场（懒结算）。
		//   刚打完 → 结果并回订单、状态回到「行进中」，紧接着走常规结算；
		//   还在打 → 跳过，等玩家指挥或下一回合自动推进。
		if order.Status == ezfyOrderStatusBattle {
			b := h.ezfyBattleByOrder(int64(order.ID))
			if b == nil {
				// 战场记录缺失（异常）→ 兜底按老流程直接结算，绝不让部队卡住
				order.Status = 0
				h.processArrive(uid, order, now)
				continue
			}
			if _, done := h.ezfyBattleTick(b, now); !done {
				continue // 还在打，等玩家指挥或下一回合
			}
			order.BattleResult = h.ezfyBattleFinishToOrder(b, now)
			order.Status = 0
			// ★ 直接结算，**不依赖 arrive_time** —— 那个字段是「单程时长」的计算基准，
			//   动它会让返航时间变成天文数字（见 ezfyBattleFinishToOrder 的注释）。
			h.processArrive(uid, order, now)
			continue
		}
		if order.Status == 0 && now >= order.ArriveTime {
			h.processArrive(uid, order, now)
		} else if order.Status == ezfyOrderStatusWaiting {
			// ★ 2026-09-23 用户要求：目标已被抢占 → 部队「等待」。
			//   目标不再忙碌(上一场打完、订单不再是战斗中)时，放行重新进指挥。
			if !ezfyOrderTargetBusy(h, order, int64(order.ID)) {
				order.Status = 0
				h.processArrive(uid, order, now)
			}
		} else if order.Status == 1 && order.OrderType == 7 && order.ArriveTime > 0 && now >= order.ArriveTime {
			h.settleDispatch(uid, order, now)
		} else if order.Status == 2 && now >= order.ReturnTime {
			h.finishReturn(uid, order)
		}
	}
	// ★ 2026-09-23 用户要求「敌人来了没提示 / 军情警讯不及时」：
	//   防守方自己的轮询也能触发「打到我家城市的敌军到达 + 开战场」——
	//   否则进攻方下线时，敌军会一直卡在「行进中」，防守方连「敌军已抵达」都收不到。
	h.processIncoming(uid, now)
}

// processIncoming 把「正在攻打 uid 名下城市、已到点」的敌方订单结算掉（开战场 / 发军情警讯）。
// 只处理 target_type=3（玩家城）且 status=0（行进中，到点）的订单，交给 processArrive 走统一流程。
func (h *EzfyHandler) processIncoming(uid uint, now int64) {
	var cities []int64
	h.DB.Model(&model.EzfyCity{}).Where("user_id = ?", uid).Pluck("id", &cities)
	if len(cities) == 0 {
		return
	}
	var orders []model.EzfyOrder
	h.DB.Where("status = 0 AND target_type = 3 AND target_id IN ? AND arrive_time <= ?", cities, now).
		Order("id ASC").Find(&orders)
	for i := range orders {
		o := &orders[i]
		// processArrive 用 uid 参数定位**攻方**城市（cityOfOrder 拿 order.CityId），
		// 所以这里必须传 o.UserID（攻方），不是当前轮询的 uid（守方）。
		h.processArrive(o.UserID, o, now)
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
	// ★ 部队带回的采集资源在这里入城
	// ★ 2026-09-24 用户反馈「运输/采集资源变少」：同上，把整行写回改成 DB 原子累加，
	//   不覆盖这期间其它写入的增量。
	//   ★ 2026-09-24 规则修正（用户确认原版口径）：入城资源不受仓储上限截断，
	//   只有超过数据库字段最大值才会溢出——去掉 LEAST(cap, ...)，改为无条件累加。
	c := parseCarry(order.Carry)
	if c.total() > 0 {
		h.DB.Model(&model.EzfyCity{}).Where("id = ?", city.ID).Updates(map[string]interface{}{
			"food":  gorm.Expr("food + ?", c.Food),
			"steel": gorm.Expr("steel + ?", c.Steel),
			"oil":   gorm.Expr("oil + ?", c.Oil),
			"rare":  gorm.Expr("rare + ?", c.Rare),
			"gold":  gorm.Expr("gold + ?", c.Gold),
		})
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
		// ★ 同 processArrive：单程时长只认 ezfyOneWayTravel，
		//   别用 ArriveTime-StartTime（被改过就是天文数字）
		travel = ezfyOneWayTravel(order)
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

// settleDispatch 常驻采集结算(每满一个采集周期一期)
//
// ★ 2026-09-23 用户规则:
//   - 采集 = 常驻: 满一个采集周期结算一期; 玩家离线错过也会一次性补算(上限 24 期);
//   - 资源按地形单一资源产出: 野地等级越高越多、带队军官后勤每 1 点 +1%(上限 +100%);
//   - 资源先装进部队「待带回」(超负重丢弃), 只有召回并返航到达才入城(见 finishReturn);
//   - 宝物每期至少 1 件直接进背包(不受负重限制), 每期另有 20% 概率多 1 件,
//     采集时间越长期数越多宝物越多; 平原/沿海平原无珠宝则该期没有宝物。
//
// 返回 (结算期数, 是否结算); 未满一期或野地已丢(部队自动返航)时 settled=false。
func (h *EzfyHandler) settleDispatch(uid uint, order *model.EzfyOrder, now int64) (int, bool) {
	// ★ 空闲驻守(arrive_time=0)或未满一期: 没有可结算的期数
	if order.ArriveTime <= 0 || now < order.ArriveTime {
		return 0, false
	}
	var wl model.EzfyWildland
	if err := h.DB.First(&wl, order.TargetId).Error; err != nil || wl.CityId != order.CityId {
		h.beginReturn(order, now, 0)
		h.addReport(uid, 5, "采集报告: 野地丢失",
			"所采集的野地已不属于我方, 采集部队已返航。", "", order.ID)
		return 0, false
	}
	// 期数: 到点的那一期 + 玩家离线漏掉的整期; 上限 24 期防止长时间积压
	periods := 1 + (now-order.ArriveTime)/ezfyDispatchPeriod()
	if periods > 24 {
		periods = 24
	}
	level := wl.Level
	terrain := ezfyTerrainEx(wl.X, wl.Y)
	food, steel, oil, rare, gainPct, resName := h.dispatchGatherYield(order, &wl, int64(periods)*ezfyDispatchPeriod())
	amt := food + steel + oil + rare
	// ★ 产出先记在部队身上(待带回), 不直接入城; 超负重部分丢弃
	loaded, dropped := h.addCarryToOrder(order, food, steel, oil, rare, 0)
	cur := parseCarry(order.Carry)
	desc := fmt.Sprintf("采集部队在野地%d级(%d,%d)驻守满%d期\n产出: %s%d",
		level, wl.X, wl.Y, periods, resName, amt)
	if gainPct > 100 {
		desc += fmt.Sprintf("(等级×%d期, 军官后勤加成 +%d%%)", periods, gainPct-100)
	} else {
		desc += fmt.Sprintf("(等级×%d期)", periods)
	}
	desc += fmt.Sprintf("\n本次装入部队: %d（负重 %d/%d）\n", loaded, cur.total(), h.ezfyCarryCap(order))
	if dropped > 0 {
		desc += fmt.Sprintf("⚠ 负重已满, %d 资源没能装上（多带运输兵/卡车可提高负重）\n", dropped)
	}
	// 宝物: 每满一个采集周期(每期)至少 1 件, 直接进背包; 每期另有 20% 概率多 1 件
	city := h.cityOfOrder(order, uid)
	treasureNames := []string{}
	for i := int64(0); i < periods; i++ {
		n := 1
		if rand.Intn(100) < ezfyTreasureExtraPct {
			n = 2
		}
		for j := 0; j < n; j++ {
			eq := h.randomTerrainTreasure(terrain)
			if eq == nil || city == nil {
				continue // 平原/沿海平原无珠宝
			}
			h.addEquipment(city, eq)
			treasureNames = append(treasureNames, eq.Name)
		}
	}
	if len(treasureNames) > 0 {
		desc += fmt.Sprintf("获得宝物(已直接放入背包): %s\n", strings.Join(treasureNames, ", "))
		h.ezfySysChat("恭喜玩家 %s 在野地%d级(%d,%d)采集到宝物: %s",
			h.ezfyProfileName(uid), level, wl.X, wl.Y, strings.Join(treasureNames, ", "))
	} else {
		desc += "本期无宝物(该地形不出珠宝)。\n"
	}
	desc += "资源需召回部队返航到达后才会入城。\n部队继续驻守采集, 可随时召回。"
	order.ArriveTime = now + ezfyDispatchPeriod()
	order.Result = order.Troops
	// ★ 必须把 carry 一起落库 —— 否则「待带回资源」只存在于内存里
	h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
		Updates(map[string]interface{}{"arrive_time": order.ArriveTime,
			"result": order.Result, "carry": order.Carry})
	h.addReport(uid, 5, "采集报告: 驻守结算", desc, "", order.ID)
	return int(periods), true
}

// dispatchGatherYield 按驻守时长(毫秒)计算采集资源产出。
//
// 每期(一个采集周期)产出 = 野地等级 × 800 × 后勤加成; 陆地 ×4, 海野(wild_type=2) ×3;
// 时长不足一期时按比例折算。返回 (粮食, 钢铁, 石油, 稀矿, 加成%, 资源名)。
func (h *EzfyHandler) dispatchGatherYield(order *model.EzfyOrder, wl *model.EzfyWildland, ms int64) (int64, int64, int64, int64, int, string) {
	// 军官后勤加成: 每 1 点 +1%, 上限 +100%
	gainPct := 100
	if officer := h.officerByName(uint(order.CityId), order.Officer); officer != nil {
		gainPct += officer.Logistics
		if gainPct > 200 {
			gainPct = 200
		}
	}
	per := int64(wl.Level) * 800 * int64(gainPct) / 100
	mult := int64(4)
	if wl.WildType == 2 {
		mult = 3
	}
	amt := per * mult
	if ms > 0 && ms < ezfyDispatchPeriod() {
		amt = amt * ms / ezfyDispatchPeriod()
	}
	amt = max64(0, amt)
	// ★ 2026-09-24 修复「提前采集资源0」: 驻守过就至少给 1 点资源
	if ms > 0 && amt < 1 {
		amt = 1
	}
	resName := ezfyGatherResName(ezfyTerrainEx(wl.X, wl.Y))
	var food, steel, oil, rare int64
	switch resName {
	case "钢铁":
		steel = amt
	case "石油":
		oil = amt
	case "稀矿":
		rare = amt
	default:
		food = amt
	}
	return food, steel, oil, rare, gainPct, resName
}

// settleDispatchOnRecall 召回驻守采集部队前的结算:
//   - 空闲驻守(arrive_time=0, 还没开始采集) → 无产出, 直接原样召回;
//   - 已满一期 → 走 settleDispatch 完整结算(资源 + 宝物);
//   - 未满一期(提前召回) → 只有按驻守时长比例折算的资源, **没有宝物**;
//
// ★ 2026-09-23 用户规则: 提前结束采集只有资源没有宝物, 满足一个采集周期才能有宝物。
// ★ 2026-09-24 修复「提前结束采集提示采集资源0」: 只要驻守过(哪怕几秒)就按比例折算,
//   至少给 1 点资源, 不再设 1 分钟硬门槛。
func (h *EzfyHandler) settleDispatchOnRecall(uid uint, order *model.EzfyOrder, now int64) {
	if order.ArriveTime <= 0 {
		return // 空闲驻守: 还没开始采集, 没有产出
	}
	if now >= order.ArriveTime {
		h.settleDispatch(uid, order, now)
		return
	}
	// 未满一期: 提前召回, 资源按已驻守时长比例结算, 宝物不给
	if order.TargetId > 0 {
		var wl model.EzfyWildland
		if err := h.DB.First(&wl, order.TargetId).Error; err == nil && wl.CityId == order.CityId {
			started := order.ArriveTime - ezfyDispatchPeriod() // 本期起算点(=上次结算或开始采集时间)
			elapsed := max64(1, now-started)
			if elapsed > ezfyDispatchPeriod() {
				elapsed = ezfyDispatchPeriod()
			}
			food, steel, oil, rare, gainPct, resName := h.dispatchGatherYield(order, &wl, elapsed)
			amt := food + steel + oil + rare
			loaded, dropped := h.addCarryToOrder(order, food, steel, oil, rare, 0)
			cur := parseCarry(order.Carry)
			desc := fmt.Sprintf("采集部队在野地%d级(%d,%d)提前召回, 按驻守时长折算资源\n产出: %s%d",
				wl.Level, wl.X, wl.Y, resName, amt)
			if gainPct > 100 {
				desc += fmt.Sprintf("(等级×%d分钟, 军官后勤加成 +%d%%)", max64(elapsed/60000, 1), gainPct-100)
			} else {
				desc += fmt.Sprintf("(等级×%d分钟)", max64(elapsed/60000, 1))
			}
			desc += fmt.Sprintf("\n本次装入部队: %d（负重 %d/%d）", loaded, cur.total(), h.ezfyCarryCap(order))
			if dropped > 0 {
				desc += fmt.Sprintf("\n⚠ 负重已满, %d 资源没能装上", dropped)
			}
			desc += "\n（提前召回不满一个采集周期, 本期没有宝物）"
			order.Result = order.Troops
			h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
				Updates(map[string]interface{}{"result": order.Result, "carry": order.Carry})
			h.addReport(uid, 5, "采集报告: 提前召回结算", desc, "", order.ID)
		}
	}
}

// ezfyOrderTargetBusy 目标是否已被别的玩家「抢先指挥」。
//
// ★ 2026-09-23 用户要求：A、B 出征同一个目标，A 已经在指挥(战斗中)的话，
//
//	B 应当「等待」，不能再同时开一个指挥室。
//	判断口径：同目标(target_type + 坐标)下存在**其他**订单处于「战斗中」(status=5)，
//	或存在**更早**的「等待」(status=6)订单（按 id 排队，防止多个等待者互相死锁）。
//	这里的 status=5 即「有进行中的战场在等玩家指挥」，把它当成目标被占用。
//
// ★ 2026-09-24 用户要求「玩家城市被征服/被掠夺中时，后到的攻击队伍进等待队列」：
//
//	等待(6)也占位 —— 新到达者只认现存战斗(5)或等待(6)就排队；放行时只让**最早**的
//	等待者先走（id 更小的优先），后面的继续等，形成 FIFO 队列。
//	战斗(5)无条件阻塞（战场的 id 可能晚于排队者，不能用 id 比较），等待(6)按 id 排先来后到。
func ezfyOrderTargetBusy(h *EzfyHandler, o *model.EzfyOrder, exceptID int64) bool {
	var n int64
	h.DB.Model(&model.EzfyOrder{}).
		Where("target_type = ? AND target_x = ? AND target_y = ? AND id <> ? AND "+
			"(status = ? OR (status = ? AND id < ?))",
			o.TargetType, o.TargetX, o.TargetY, exceptID,
			ezfyOrderStatusBattle, ezfyOrderStatusWaiting, o.ID).
		Count(&n)
	return n > 0
}

func (h *EzfyHandler) processArrive(uid uint, order *model.EzfyOrder, now int64) {
	// ★ 2026-09-24 用户反馈「征服报告出现两封」：
	//   同一订单有两个结算入口 —— 攻方 processOrders 的战斗分支、守方 processIncoming。
	//   并发/先后到达时会重复调用 processArrive，战报写两遍、掠夺结算两遍。
	//   入口用 CAS 抢占「结算权」：status 0(行进)/5(战斗中) → 98(结算中)，
	//   抢不到说明已有对方入口在结算，直接返回。
	res := h.DB.Model(&model.EzfyOrder{}).
		Where("id = ? AND user_id = ? AND status IN (0, ?, ?)", order.ID, uid,
			ezfyOrderStatusBattle, ezfyOrderStatusWaiting).
		Updates(map[string]interface{}{"status": ezfyOrderStatusProcessing})
	if res.RowsAffected == 0 {
		return
	}
	city := h.cityOfOrder(order, uid)

	// 活动目标(活动野地/活动寇城/特殊城市): 掠夺/征服走独立的活动战斗结算
	// (打赢只结算资源/黄金/宝物/声望, 不占领、不占附属野地上限)
	if order.OrderType == 2 || order.OrderType == 3 {
		if act := h.ezfyActTargetType(order.TargetX, order.TargetY); act > 0 {
			h.processActivityBattle(uid, city, order, now, act)
			return
		}
	}

	// 采集(4)/派遣(7): 到达已占领野地 → 转为常驻驻军(空闲待命)
	// ★ 2026-09-24 用户规则: 到达后驻守**空闲**, 必须手工点[采集](或驻军区的[一键采集])
	//   才开始采集 —— 不再自动进采集。开始采集(StartCollect)后每满一个采集周期结算一期:
	//   资源装进部队待带回(等级越高越多, 军官后勤加成), 宝物直接进背包(每期至少 1 件);
	//   只有召回并返航到达才把资源运回城中。空闲驻军用 arrive_time=0 表示(不参与结算)。
	if order.OrderType == 4 || order.OrderType == 7 {
		var wl model.EzfyWildland
		if err := h.DB.First(&wl, order.TargetId).Error; err != nil || wl.CityId != order.CityId {
			h.beginReturn(order, now, 0)
			h.addReport(uid, 5, "采集报告: 野地丢失",
				"采集目标野地已不属于我方, 采集部队已返航。", "", order.ID)
			return
		}
		order.Status = 1
		order.OrderType = 7 // 统一口径为驻守采集, 后续结算/一键收获/一键召回都按 7 处理
		order.Result = order.Troops
		order.ArriveTime = 0 // 0 = 空闲驻守, 待玩家手工[采集]才开始采集
		h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
			Updates(map[string]interface{}{"status": 1, "order_type": 7, "result": order.Troops, "arrive_time": 0})
		h.addReport(uid, 5, "采集报告: 部队已抵达",
			fmt.Sprintf("采集部队已抵达野地(%d,%d)驻守, 当前空闲待命。\n请到「军情 → 驻军」点[采集](或[一键采集])开始采集: 每满一个采集周期结算一期, 资源装进部队待召回(等级越高越多, 军官后勤每点+1%%), 宝物直接进背包(每期至少1件)。可随时召回。",
				wl.X, wl.Y), "", order.ID)
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
		// ★ 2026-09-24 规则修正（用户确认原版口径）：运输到达入城**不受仓储上限截断**
		//   （只有超过数据库字段最大值才溢出），全部入库、不再把超出部分原路带回。
		h.DB.Model(&model.EzfyCity{}).Where("id = ?", target.ID).Updates(map[string]interface{}{
			"food":  gorm.Expr("food + ?", f),
			"steel": gorm.Expr("steel + ?", s),
			"oil":   gorm.Expr("oil + ?", o),
			"rare":  gorm.Expr("rare + ?", r),
			"gold":  gorm.Expr("gold + ?", g),
		})
		order.Carry = ""
		desc := fmt.Sprintf("运输部队已到达%s\n", target.Name)
		if f+s+o+r+g > 0 {
			desc += fmt.Sprintf("送达: 粮%d 钢%d 油%d 稀矿%d 金%d\n", f, s, o, r, g)
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
		// 随军资源入目标城
		// ★ 2026-09-24 规则修正（用户确认原版口径）：不受仓储上限截断（只有超过
		//   数据库字段最大值才溢出），装不下的不再原路带回；仍用 DB 原子累加，
		//   不覆盖这期间其它写入的增量。
		res := h.parseResMap(order.Resources)
		back := ezfyCarry{}
		total := res["food"] + res["steel"] + res["oil"] + res["rare"] + res["gold"]
		if total > 0 {
			h.calcResource(&target, h.officerList(target.ID))
			h.DB.Model(&model.EzfyCity{}).Where("id = ?", target.ID).Updates(map[string]interface{}{
				"food":  gorm.Expr("food + ?", res["food"]),
				"steel": gorm.Expr("steel + ?", res["steel"]),
				"oil":   gorm.Expr("oil + ?", res["oil"]),
				"rare":  gorm.Expr("rare + ?", res["rare"]),
				"gold":  gorm.Expr("gold + ?", res["gold"]),
			})
			desc += fmt.Sprintf("\n随军资源已入库: 粮%d 钢%d 油%d 稀矿%d 金%d",
				res["food"], res["steel"], res["oil"], res["rare"], res["gold"])
		}
		// 随军军官调任目标城市（清空职位）
		if order.Officer != "" {
			h.moveOfficerTo(city, order.Officer, target.ID)
			desc += "\n军官 " + order.Officer + " 随军调往" + target.Name
		}
		if back.total() > 0 {
			// 兵力已经进城，回程只带「装不下的资源」：Result 置空避免兵力重复入账
			// ★ 单程时长必须用 ezfyOneWayTravel（读 ReturnTime-StartTime）：
			//   `ArriveTime - StartTime` 一旦被外部改过（测试、指挥室流程）就会算出天文数字，
			//   玩家看到「20717 天才能回来」就是这么来的。
			travel := ezfyOneWayTravel(order)
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
	wildDefCamp := 0 // 野地守军阵营: 1盟军(野地) 2轴心国(寇城), 0无
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
		// ★ 野地守将（2026-09-24 用户要求）：军官必须来自军官池(ezfy_cfg_general)、
		//   每块野地最多 1 名，配在野地类型的 officer_id 上；守将的学识给守军提供防御加成。
		wildDefCamp = 1 // 野地守军按盟军兵种名展示
		if cfgType == 3 {
			wildDefCamp = 2 // 寇城守军按轴心国兵种名展示
		}
		if cfg.OfficerId > 0 {
			if g := ezfyCfg.general(cfg.OfficerId); g != nil {
				guardAttr := ezfyAttrToBonus(g.Learning)
				defBonus += guardAttr
				defOfficerDesc = g.Name + " Lv." + strconv.Itoa(g.Level) + " 守军防御+" + strconv.Itoa(guardAttr) + "%"
			}
		}
		// ★ 用户要求：战报里的野地要标出**具体地形类型**（丘陵/沼泽/平原…），
		//   原来一律写「野地N级」，看不出打的是什么地形。
		name := ezfyTerrainName(ezfyTerrain(order.TargetX, order.TargetY))
		if order.TargetType == 2 {
			name = "寇城"
		} else if ezfyTerrain(order.TargetX, order.TargetY) == 8 {
			name = "海底森林"
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
		// ★ 排除「不参与防御」的兵种：被攻击时防御战斗兵种列表不含它们
		defExclude := h.defExcludeSet(target.ID)
		for tid, count := range h.troopMap(target.ID) {
			if count > 0 && !defExclude[tid] {
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
		// ★ 传「属性部分」的防御加成（有效学识÷2），技能由 officerBattleDesc 自己列，
		//   否则技能会被算两遍。原来这里硬编码 10，与实际生效值不符。
		defOfficerDesc = h.officerBattleDesc(cityGuard, h.officerGuardAttrBonus(cityGuard), "守军防御")
	}

	// 侦查: 不战斗只报告情报, 部队随即返航
	// 报告格式复刻 `参考材料/开发文档/侦察报告1.txt`:
	//   玩家城市 → 资源数量/人口民心/建筑等级/城防数量/军队数量/将领等级/科技等级/最后活动时间
	//   野地寇城 → 守军情况
	if order.OrderType == 1 {
		// ★ 侦查报告：返程时长同样只认 ezfyOneWayTravel
		travel := ezfyOneWayTravel(order)
		order.Status = 2
		order.Result = order.Troops
		order.ReturnTime = now + travel
		h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
			Updates(map[string]interface{}{"status": 2, "result": order.Troops, "return_time": order.ReturnTime})
		h.addReport(uid, 1, "侦查报告: "+targetName,
			h.scoutReportBody(uid, order, targetName, target, defender), "", order.ID)
		return
	}

	// ★★ 指挥室（2026-09-22 用户要求）：战斗类订单到达后**不立即结算**，
	//   先开一场战场，玩家在「军情 → 军队动态 → [指挥]」里下达前进/暂停/后退；
	//   每回合 30 秒、最多 40 回合，不下指令则按「前进」自动推进。
	//   BattleResult 非空 = 这场仗已经在指挥室里打完了 → 直接用结果走下面的常规结算，
	//   所以战报/掠夺/经验/征服这些战后逻辑全部复用，没有第二套实现。
	atkTargets := h.buildTargetMap(city.ID, true)
	defTargets := h.buildTargetMap(cityIdOf(target), false)
	atkMoves := h.buildMoveMap(city.ID, true)
	defMoves := h.buildMoveMap(cityIdOf(target), false)

	var br ezfyBattleResult
	if done, ok := ezfyBattleResultDecode(order.BattleResult); ok {
		br = done
	} else {
		// ★ 阵营兵种名：守方是玩家城时用守方阵营；野地=盟军、寇城=轴心国，其余 0(通用名)
		defCamp := wildDefCamp
		if target != nil {
			defCamp = h.ensureProfile(target.UserID).Camp
		}
		st := ezfyNewBattleState(attacker, defender,
			atkBonus, defBonus, atkSpeedBonus, defSpeedBonus,
			atkEquip, defEquip, atkOfficerDesc, defOfficerDesc,
			atkTargets, defTargets, atkMoves, defMoves,
			// ★ 军官技能「绝地反击」：第1回合被打可反击（攻方带队/守方城守各自判定）
			h.officerHasSkill(leadOfficer, "绝地反击"), h.officerHasSkill(cityGuard, "绝地反击"),
			h.ensureProfile(uid).Camp, defCamp)
		// ★ 2026-09-23 用户要求：目标被别的玩家抢先指挥时，本部队改为「等待」，
		//   不重复开指挥室。上一场打完(那个订单不再处于战斗中)后，processOrders 会自动放行重进。
		if ezfyOrderTargetBusy(h, order, int64(order.ID)) {
			order.Status = ezfyOrderStatusWaiting
			h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
				Update("status", ezfyOrderStatusWaiting)
			return
		}
		if b := h.ezfyBattleStart(uid, order, st, targetName, now); b != nil {
			order.Status = ezfyOrderStatusBattle
			h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
				Update("status", ezfyOrderStatusBattle)
			// ★ 2026-09-23 用户要求「敌人来了没提示 / 军情警讯不及时」：
			//   敌军**到达**我方城市开战时，立即给守方发一条「军情警讯」。
			//   （「敌军来袭」预警已在 createOrder 时发；这里补「已抵达」的实时消息，
			//   不依赖雷达站 —— 结果类消息不受雷达限制。）
			if order.TargetType == 3 && target != nil && target.UserID > 0 && target.UserID != uid {
				h.addReport(target.UserID, 6, "军情警报: 敌军已抵达",
					fmt.Sprintf("敌方部队已抵达我方城市「%s」(%d,%d) 附近，双方即将交战！\n来袭方城市：%s\n请到「军情 → 军队动态」进入[指挥]部署守军。",
						target.Name, order.TargetX, order.TargetY, city.Name))
			}
			// ★ 用户要求：「等待指挥」不要放进战斗报告列表 —— 战斗还没结束，战报应当是**结果**。
			//   部队状态在「军情 → 军队动态 / 出征队列」里已显示「战斗中 + [指挥]」，
			//   再发一条战报只会把战斗报告列表搅乱。
			return
		}
		// 开战场失败（极端情况：写库异常）→ 兜底走老流程直接模拟，绝不让部队卡住
		for !st.Done {
			st.Step(nil, nil)
		}
		br = st.Result()
	}
	win = br.AttackerWin
	draw := br.Draw

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
		ezfyOrderTypeName(order.OrderType), br.Rounds, battleOutcomeText(win, draw))

	profile := h.ensureProfile(uid)
	report += fmt.Sprintf("军衔声望:%d\n", profile.Prestige)
	// ★ 用户要求：战报里的兵种名用「阵营兵种名」(同盟国/轴心国各自的叫法)，
	//   不再是笼统的大类名。攻方用攻方阵营，守方用守方阵营。
	atkCamp := profile.Camp
	// ★ 2026-09-24 修复: 野地=盟军兵种名、寇城=轴心国兵种名(之前这里重置成 0 → 战报里兵种全是通用名)
	defCamp := wildDefCamp
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
	// ★ 2026-09-24 用户要求：40 回合平局时双方标签都显示 [平] 而不是胜/败
	atkTag, defTag := winText(win), winText(!win)
	if draw {
		atkTag, defTag = "平", "平"
	}
	report += fmt.Sprintf("[%s]攻方:%s\n", atkTag, city.Name)
	report += troopChangeText(atkBefore, atkAfter, atkCamp)
	report += fmt.Sprintf("--------------------\n[%s]守方:%s\n", defTag, targetName)
	// ★★ 守方兵力(2026-09-23 修复报错)：原来 defBefore 直接取**野地配置满编兵力**，
	//   再用它减去战斗损失得出「剩余」。但战斗实际打到的是**已经损耗过的守军**
	//   （同一野地/城市被先前战斗打过、或指挥官中途换过），满编数 ≠ 战斗初始数，
	//   一减就冒出「打了 701400，还剩 182100」这种从没存在过的假剩余，
	//   让玩家误以为「战斗没打完就结束了」。
	//   现在守方 before/after 都取战斗结果本身的真实兵力（防御损失 + 战前剩余 → before，
	//   战后剩余 → after），与攻方口径一致，永远对得上。
	defAfter := groupCounts(br.DefenderLeft)
	defBefore := map[int]int64{}
	for _, g := range br.DefenderLosses {
		defBefore[g.TroopId] += g.Count
	}
	for tid, cnt := range defAfter {
		defBefore[tid] += cnt
	}
	report += troopChangeText(defBefore, defAfter, defCamp)

	detail := ""
	for _, a := range br.Actions {
		detail += a + "\n"
	}
	detail += "\n[双方兵力]\n"
	detail += fmt.Sprintf("[%s]攻方:%s\n", atkTag, city.Name)
	detail += troopChangeText(atkBefore, atkAfter, atkCamp)
	detail += fmt.Sprintf("--------------------\n[%s]守方:%s\n", defTag, targetName)
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
				travel := ezfyOneWayTravel(order)
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
				h.addReport(uid, 2, reportType+": "+targetName+
					"("+strconv.Itoa(order.TargetX)+","+strconv.Itoa(order.TargetY)+")", report, detail, order.ID)
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
				// ★ 俘获的守军用其原阵营兵种名: 野地=盟军、寇城=轴心国
				report += fmt.Sprintf("\n俘获: %s×%d", ezfyCfg.troopName(capturedTroopId, wildDefCamp), capturedCount)
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
				travel := ezfyOneWayTravel(order)
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
				travel := ezfyOneWayTravel(order)
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
		travel := ezfyOneWayTravel(order)
		order.Status = 2
		order.ReturnTime = now + travel
		if wareNote != "" {
			report += "\n" + wareNote
		}
		report += fmt.Sprintf("\n战果\n黄金:%d\n粮食:%d\n钢铁:%d\n石油:%d\n稀矿:%d", lootGold, lootFood, lootSteel, lootOil, lootRare)
		if repairedTotal > 0 {
			report += fmt.Sprintf("\n伤兵入营: %d", repairedTotal)
		}
		report += h.battleStatsTail(uid, prestigeGain, recyclePct)
		h.addReport(uid, 2, reportType+": "+targetName+
			"("+strconv.Itoa(order.TargetX)+","+strconv.Itoa(order.TargetY)+")", report, detail, order.ID)
	} else {
		// ★ 第九轮：打败仗 → 幸存部队撤退返航（原来 status=4 是终止态，
		//   幸存兵力凭空消失、带队军官永远卡在「出征中」，属于 bug）。
		travel := ezfyOneWayTravel(order)
		order.Status = 2
		order.ReturnTime = now + travel
		if repairedTotal > 0 {
			report += fmt.Sprintf("\n伤兵入营: %d", repairedTotal)
		}
		// ★ 军官忠诚：只有**打败仗**才掉，且按战损比例合理计算（基础 3 点，全灭 10 点）
		//   平局不算败仗，不掉忠诚。
		if leadOfficer != nil && !draw {
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
		h.addReport(uid, 2, reportType+": "+targetName+
			"("+strconv.Itoa(order.TargetX)+","+strconv.Itoa(order.TargetY)+")", report, detail, order.ID)
		if order.TargetType == 3 && target != nil {
			// ★ 2026-09-24 用户要求：军情列表展示 [防守报告] 城市名(坐标)，标题需携带守方城名+坐标
			h.addReport(target.UserID, 4, "守卫报告: "+targetName+
				"("+strconv.Itoa(order.TargetX)+","+strconv.Itoa(order.TargetY)+")",
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

// defExcludeSet 城市「不参与防御」的兵种集合（司令部「防守」= 不参与防御，def_move = -1）。
//
// ★ 用户要求（2026-09-23）：被攻击时，防御战斗的兵种列表**不包含**标记了「不参与防御」的兵种。
func (h *EzfyHandler) defExcludeSet(cityId uint) map[int]bool {
	m := map[int]bool{}
	if cityId <= 0 {
		return m
	}
	var list []model.EzfyCityTarget
	h.DB.Where("city_id = ? AND def_move = ?", cityId, ezfyDefMoveNone).Find(&list)
	for _, t := range list {
		m[t.TroopId] = true
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

// battleOutcomeText 战报结局文案（★ 2026-09-24 用户要求：40 回合未分胜负显示平局而非失败）
func battleOutcomeText(win, draw bool) string {
	if draw {
		return "与敌方打成平局！"
	}
	return winResultText(win)
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
		// ★ 兵种名按阵营: 野地=盟军, 寇城=轴心国
		camp := 1
		cfgType := 1
		level := ezfyWildlandLevel(order.TargetX, order.TargetY)
		if order.TargetType == 2 {
			camp = 2
			cfgType = 3
			level = ezfyKouLevel(order.TargetX, order.TargetY)
		} else if ezfyTerrain(order.TargetX, order.TargetY) == 8 {
			cfgType = 2
		}
		for _, g := range defender {
			if cfg := ezfyCfg.troop(g.TroopId); cfg != nil {
				name := ezfyCfg.troopName(g.TroopId, camp)
				if name == "" {
					name = cfg.Name
				}
				fmt.Fprintf(&b, "%s×%d ", name, g.Count)
			}
		}
		// ★ 守将（军官池配置，每块野地最多 1 名）
		if cfg := ezfyCfg.wildland(cfgType, level); cfg != nil && cfg.OfficerId > 0 {
			if g := ezfyCfg.general(cfg.OfficerId); g != nil {
				fmt.Fprintf(&b, "\n守将: %s Lv.%d", g.Name, g.Level)
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

	// 城防 / 军队分列（★ 兵种名用被侦查方的阵营兵种名，与战报口径一致）
	defText, armyText := "", ""
	defCamp := h.ensureProfile(target.UserID).Camp
	for tid, cnt := range h.troopMap(target.ID) {
		cfg := ezfyCfg.troop(tid)
		if cfg == nil || cnt <= 0 {
			continue
		}
		name := ezfyCfg.troopName(tid, defCamp)
		if name == "" {
			name = cfg.Name
		}
		if cfg.Type == 4 {
			defText += fmt.Sprintf("%s×%d ", name, cnt)
		} else {
			armyText += fmt.Sprintf("%s%d ", name, cnt)
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
