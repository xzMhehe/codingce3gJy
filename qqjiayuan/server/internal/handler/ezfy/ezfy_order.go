package ezfy

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"sync"
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
	// ★ 2026-10-03 性能：地图渲染只需要视野内的城市，别把整张地图所有城（含全部资源/兵力）
	//   跨 WAN 拉回来。按视野(±r)裁剪，数据量从全表降到几十格。
	h.DB.Select("id, x, y, user_id, name, city_level").
		Where("x >= ? AND x <= ? AND y >= ? AND y <= ?", cx-r, cx+r, cy-r, cy+r).
		Find(&myCities)
	cityAt := map[string]*model.EzfyCity{}
	for i := range myCities {
		c := &myCities[i]
		cityAt[fmt.Sprintf("%d,%d", c.X, c.Y)] = c
	}
	// 玩家昵称（城主显示）
	// ★ 2026-10-06 修复「点击地图城市，城主显示的是家园名字」：游戏昵称存
	//   ezfy_profile.nickname（改名/转阵营都写它），家园昵称是 users.nickname。
	//   这里 profile.nickname 优先、空才回落 users.nickname —— 与 ezfyNickOf 同口径，
	//   否则玩家在游戏里改过名后，地图/详情上看到的是旧家园名。
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
		// 游戏昵称覆盖家园昵称：ezfy_profile.nickname 有值的玩家用游戏内的名字
		var ps []model.EzfyProfile
		h.DB.Select("user_id, nickname").Where("user_id IN ? AND nickname <> ''", userIDs).Find(&ps)
		for _, p := range ps {
			userNames[p.UserID] = p.Nickname
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

	// ★ 「点击地图的出征 → 看到玩家城市 → 点进去 → 展示玩家同盟名字」。
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

	// 已占领野地(一次载入视野内, 避免逐格查库; ★ 2026-10-03 按视野裁剪, 不再全表拉回)
	var allWilds []model.EzfyWildland
	h.DB.Select("x, y, city_id").
		Where("x >= ? AND x <= ? AND y >= ? AND y <= ?", cx-r, cx+r, cy-r, cy+r).
		Find(&allWilds)
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
			// ★ 2026-10-05 用户口径：**海洋=没有野地的海格；海底森林=海里有野地的那块**。
			//   所以地形名也要走 ezfyWildTerrainDisplayName —— 纯海洋格给「海洋」，
			//   有野地的海格给「海底森林」，与 name / 详情页 terrain_name 三处完全同口径。
			cell := gin.H{"x": x, "y": y, "terrain": terrain,
				"terrain_name": ezfyWildTerrainDisplayName(x, y, false), "continent": ezfyRegionName(x, y)}
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
					// 纯海洋 → 「海洋」(详情页不显示守军/军官/出征按钮)；带野地 → 「海底森林」。
					// ★ 2026-10-05 统一走 ezfyWildTerrainDisplayName（口径只有一处），这里 knownWild=false。
					cell["name"] = ezfyWildTerrainDisplayName(x, y, false)
					if lvl == 0 {
						cell["is_ocean"] = true
					} else {
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
					// 陆地野地: 名称取地形名(平原/草原/森林/盆地/丘陵/沼泽/山地/岛屿), 不再一律叫「野地」
					// ★ 2026-10-05 用户纠正：岛屿虽然按海野玩法处理，但**展示名仍是「岛屿」**，
					//   不能显示成「海底森林」（原实现在这里把岛屿改成了海底森林，属 bug，已删）。
					cell["area_type"] = 1
					cell["name"] = ezfyTerrainName(terrain)
					cell["level"] = ezfyWildlandLevel(x, y)
				}
				// 活动目标标记: 复刻 mapView.html 的 actWild/actKou/actCity
				// (活动野地橙、活动寇城品红、特殊城市红, 三种都带活动等级 1~3)
				// ★ 2026-10-05 名将野地按玩家判定：该玩家已抓到守将 → 该格对其是普通野地，
				//   不再标活动标记/守将标识（没抓到的玩家照常看到名将野地）。
				if act := h.ezfyActTargetType(x, y); act > 0 && !h.playerOwnsActWildGeneral(uid, x, y) {
					actLevel := ezfyMarkLevelAt(x, y)
					cell["act_type"] = act
					cell["act_level"] = actLevel
					cell["act_name"] = ezfyActTargetName(act)
					// 格子名直接用活动标签, 目标详情页标题即「活动野地2级 / 特殊城市3级」
					cell["name"] = ezfyActTargetLabel(act, actLevel)
					// ★ 2026-09-30 带守将军官的活动野地加特殊标识（玩家区分普通活动野地与带名将守将的）
					if aw := ezfyActWildAt(x, y); aw != nil && aw.Enabled == 1 && aw.OfficerId > 0 {
						cell["act_officer"] = true
						if g := ezfyCfg.general(aw.OfficerId); g != nil {
							cell["act_officer_name"] = g.Name
							cell["act_officer_kind"] = g.Kind // 1普通军官 2名将
							cell["act_officer_star"] = g.Star
						}
					}
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
	// ★ 2026-10-05 名将野地按玩家判定：已抓到守将的玩家，该坐标对其是普通野地 → 走下面普通野地详情
	if act := h.ezfyActTargetType(x, y); act > 0 && !h.playerOwnsActWildGeneral(uid, x, y) {
		resp.OK(c, h.ezfyActWildlandView(uid, profile.Camp, x, y, act))
		return
	}
	// ★★ 2026-10-05 用户口径 + 服务端权威：**走哪套野地配置一律按地形推断，不再信任客户端传的 type**。
	//
	//	原来只在「客户端没传合法 type」时才推断，而前端 openCell 一定会传：
	//	  `const ttype = cell.areaType === 2 ? 3 : (cell.terrain === 8 ? 2 : 1)`
	//	→ 岛屿(地形 7) 被推成 1（**陆地野地**），于是详情页预览的是陆军守军；
	//	  而真正打起来时 processArrive 按 `ezfyIsSeaWildTerrain` 判成海野 → 用的是海军守军。
	//	  实测 (10,15) 岛屿：type=1 预览「摩托化掷弹兵/虎式重型坦克」，type=2 才是「驱逐舰/潜艇」，
	//	  与实际战斗口径（海野）不符 —— 玩家会按错误的情报备兵。
	//
	//	现在改成与 processArrive / MapView 完全同一套判定（顺序也一致：海洋分支在地图里优先于寇城）：
	//	  海野(海洋 8 / 岛屿 7) → 2 ；寇城 → 3 ；其余 → 1（陆地野地）。
	//	⚠️ 客户端的 `type` 查询参数已不再参与判定（传了也不影响结果），保留只是为了兼容旧前端。
	ttype := 1
	switch {
	case ezfyIsSeaWildTerrain(ezfyTerrainEx(x, y)):
		ttype = 2
	case h.ezfyIsKouCity(x, y):
		ttype = 3
	}
	// ★ 纯海洋（地形 8 且**该格没有野地**）：只显示「地形：海洋」，无守军/军官/出征按钮。
	//   用户口径「海洋上不会有野地」→ 纯海洋绝不能走进野地分支
	//   （否则会显示「采集可获得石油」+ 野地宝物池，等于凭空造了一块野地）。
	//   ⚠️ 判定必须看**地形**，不能看 ttype：早先写的是 `ttype == 2 && level == 0`，
	//   一旦客户端传了 type=1 就绕过守卫、直接按陆地野地渲染纯海洋。
	if ttype != 3 && ezfyTerrainEx(x, y) == ezfyTerrainSea && ezfyWildlandLevel(x, y) == 0 {
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
	// 地形显示名: 海洋里的野地→海底森林, 岛屿→岛屿, 寇城→平原(用户规范)
	// ★ 2026-10-05 用户纠正：① 纯海洋是「海洋」、海底森林是海洋里带野地的那块；② 岛屿(7) 也是海野玩法，
	//   但展示名保留「岛屿」。走 ezfyWildTerrainDisplayName。
	//   ttype==2 时上面已确认 `ezfyWildlandLevel(x,y) > 0`（纯海洋在更早的分支直接返回 is_ocean），
	//   所以这里 knownWild=true。
	terrainName := ezfyTerrainNameEx(x, y)
	if ttype == 2 {
		terrainName = ezfyWildTerrainDisplayName(x, y, true)
	} else if ttype == 3 {
		terrainName = "平原"
	}
	// 归属: 已占领该野地的玩家(复刻 mapView 的【归属: xxx】)
	// ★ 2026-09-28 用户规则「自己的附属野地不能侦查/掠夺/征服，除非先放弃」→
	//   这里顺带下发布尔 mine（口径与 owner 完全同源），前端据此灰掉那三个命令；
	//   后端 createOrder 里也有同样校验，防止绕过前端直接下单。
	owner, isMine := "", false
	var w model.EzfyWildland
	if err := h.DB.Where("x = ? AND y = ?", x, y).First(&w).Error; err == nil && w.CityId > 0 {
		var oc model.EzfyCity
		if err := h.DB.First(&oc, w.CityId).Error; err == nil {
			if oc.UserID == uid {
				owner = "我"
				isMine = true
			} else {
				owner = h.ensureProfile(oc.UserID).Nickname
			}
		}
	}
	// ★ 2026-09-30 详情直接下发收藏态，前端收藏按钮不用依赖异步 loadStars
	var starCnt int64
	h.DB.Model(&model.EzfyMapStar{}).Where("user_id = ? AND x = ? AND y = ?", uid, x, y).Count(&starCnt)
	resp.OK(c, gin.H{
		"x": x, "y": y, "type": ttype, "level": level,
		"name": cfg.Des, "troops": previews,
		// ★ 2026-09-25：这里下发的「胜利奖励」也要带上「野地获取资源倍率」——
		//   前端会显示成「胜利奖励：粮/钢/油/稀矿 各N」，只在实际结算时放大就会
		//   「预览 1 万、真打给 5 万」，玩家反而以为算错了。两边必须同一口径。
		"res_min": ezfyScaleByWildResMult(cfg.ResMin), "res_max": ezfyScaleByWildResMult(cfg.ResMax),
		"res_mult": ezfyWildResMult(), "terrain": ezfyTerrainEx(x, y),
		"terrain_name": terrainName,
		"continent":    ezfyRegionName(x, y),
		"treasures":    treasures,
		"gather_res":   gatherRes,
		"treasure":     cfg.Treasure, // 寇城宝物档次(初级/中级/高级)
		"owner":        owner,
		"mine":         isMine,      // ★ 自己的附属野地（不能侦查/掠夺/征服，要先放弃）
		"is_starred":   starCnt > 0, // ★ 2026-09-30 收藏态
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

// addResToCityDB 把一笔资源原子累加进某座城（无条件累加，收敛到配置的「资源最大值」）。
//
// ★ 2026-09-25 「攻击野地获得的资源也要累加」「各项资源有最大的配置」。
//   - 之前战利品是在内存里 city.Food += n 再 saveCityRes(city)：整行写回，
//     并发下会把别的请求刚写入的增量覆盖掉（同类事故在运输/采集那几处已经修过），
//     野地/寇城战利品却漏了 —— 这就是「打野地的资源没累加」的根因。
//   - 现在统一走 ezfyResAddExpr：DB 侧原子累加，且累加到「资源最大值」就不再增加。
//   - 非正值直接跳过，避免拼无用 SQL。
func (h *EzfyHandler) addResToCityDB(cityId uint, food, steel, oil, rare, gold int64) {
	if cityId == 0 || food+steel+oil+rare+gold == 0 {
		return
	}
	upd := map[string]interface{}{}
	if food > 0 {
		upd["food"] = ezfyResAddExpr("food", food)
	}
	if steel > 0 {
		upd["steel"] = ezfyResAddExpr("steel", steel)
	}
	if oil > 0 {
		upd["oil"] = ezfyResAddExpr("oil", oil)
	}
	if rare > 0 {
		upd["rare"] = ezfyResAddExpr("rare", rare)
	}
	if gold > 0 {
		upd["gold"] = ezfyResAddExpr("gold", gold)
	}
	if len(upd) == 0 {
		return
	}
	h.DB.Model(&model.EzfyCity{}).Where("id = ?", cityId).Updates(upd)
}

// isAtWar 是否可以对该玩家发起掠夺/征服
//
// ★ 「加一个宣战功能开关，关闭后不需要宣战也能掠夺/征服」→
// 开关关掉时恒为 true（视为随时可交战）。这样出征校验、战斗结算两处一起放开，
// 不会出现「出征放行了、到达时又被判没宣战而返航」的不一致。
func (h *EzfyHandler) isAtWar(a, b uint) bool {
	if !ezfyWarRequireOn() {
		return true
	}
	// ★ 2026-09-25 「军团宣战生效期间，双方军团成员之间可直接掠夺/征服，无需个人宣战」：
	//   在原有个人宣战判断之外，新增「两人所属军团之间存在生效中的军团宣战 → 返回 true」。
	//   这样出征校验与战斗结算两处口径一致（原注释就是要求两处一起放开）。
	if h.corpsActiveWarBetween(a, b) != nil {
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
	// ★ 2026-10-05 性能：选城（含被占校验）与「玩家城市列表」**并行**取，
	//   后者直接传给 createOrder 的懒结算复用 —— 省掉懒结算里那次串行的城市列表查询（跨 WAN ~120ms）。
	var city *model.EzfyCity
	var cities []model.EzfyCity
	var cwg sync.WaitGroup
	cwg.Add(2)
	go func() { defer cwg.Done(); city = h.cityOf(uid, req.CityId) }()
	go func() { defer cwg.Done(); h.DB.Where("user_id = ?", uid).Order("id ASC").Find(&cities) }()
	cwg.Wait()
	if city == nil {
		city2 := h.getOrCreateCity(uid)
		city = &city2
		cities = nil // 新建的城不在上面那份列表里，交给懒结算自己查
	}
	waitMin := req.WaitMin
	if waitMin < 0 {
		waitMin = 0
	}
	if waitMin > 1440 {
		waitMin = 1440
	}
	if msg := h.createOrder(uid, city, req.OrderType, req.TargetX, req.TargetY, req.TargetType, req.TargetId, req.Troops, req.Resources, req.Officer, waitMin, req.Gather, cities); msg != "" {
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
	// ★ 2026-10-05 性能（用户反馈「/order/preview 还是 2s」）：原 refreshCity 串行跑
	//   建筑/科技/队列/资源懒结算 + processOrders（5+ 条跨 WAN RDS），是 2s 的根源。
	//   预览是**纯只读计算**（不下达命令、不扣资源），订单/资源结算交给 /view 轮询照常
	//   推进（资源数值最多滞后一轮轮询，可接受）。这里一个并行波把预览需要的
	//   科技等级 / 指挥室 / 司令部 / 带队军官 / 集结令持有量 全部取完（1 RTT）。
	var userTechs []model.EzfyUserTech
	var stationLv, hqLv int
	var lead *model.EzfyOfficer
	var gatherHave int
	var wg sync.WaitGroup
	wg.Add(5)
	go func() { defer wg.Done(); h.DB.Where("user_id = ?", uid).Find(&userTechs) }()
	go func() { defer wg.Done(); stationLv = h.buildingLevel(city.ID, 20) }()
	go func() { defer wg.Done(); hqLv = h.buildingLevel(city.ID, 13) }()
	go func() {
		defer wg.Done()
		if req.Officer != "" {
			lead = h.officerByName(city.ID, req.Officer)
		}
	}()
	go func() { defer wg.Done(); gatherHave = h.itemCount(uid, ezfyGatherItemID) }()
	wg.Wait()
	techMap := map[int]int{}
	for _, t := range userTechs {
		techMap[t.TechId] = t.Level
	}

	valid := []ezfyUnitGroup{}
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
		if slowest == 0 || cfg.Speed < slowest {
			slowest = cfg.Speed
		}
	}
	// ★ 2026-09-28 负重统一走 ezfyCarryCapOf（含「装载技术」加成）。
	//   原来这里手写 `carry += cfg.Carry * count`，与 ezfyCarryCapOf 是**两套实现** ——
	//   加了科技加成后如果只改一处，出征页预览的负重就会和实际出征时校验的负重对不上。
	carry := h.ezfyCarryCapOf(valid, city.ID, techMap)
	distance := ezfyAbs(city.X-req.TargetX) + ezfyAbs(city.Y-req.TargetY)
	oilCost := h.ezfyOilCost(city, req.OrderType, distance, valid, req.Resources)

	var travelSec int64
	if distance > 0 && slowest > 0 {
		// ★ 2026-10-05 科技/指挥室/带队军官来自上方并行块（零额外查询）；带队军官只查一次
		travelSec = int64(distance) * 60 * 300 / int64(slowest)
		travelSec = travelSec * 100 / int64(100+techMap[12]*2)
		travelSec = travelSec * 100 / int64(100+stationLv*3)
		if s := h.officerSpeedSkillBonus(lead); s > 0 {
			// ★ 2026-10-06 移速技能随军官等级自动升级：-N% 行军时间按当前加成算
			travelSec = travelSec * 100 / int64(100+s)
		}
		// ★ 2026-09-28 军官军事加成出征速度：每点军事 +0.1%（可配，与 createOrder 同口径）
		if lead != nil && lead.Military > 0 {
			travelSec = int64(float64(travelSec) * 100 / (100 + float64(lead.Military)*ezfyOfficerSpeedPerMil()))
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
	capNow, capUnlimited := h.ezfyOrderTroopCap(city.ID, gather, req.Officer,
		&ezfyTroopCapReuse{techs: techMap, hqLv: hqLv, lead: lead})
	// ★ 2026-10-02 自城派遣(8)在非战斗状态/免战期间无上限（预览与下单同口径）
	if req.OrderType == 8 && h.dispatchNoCap(uid, city) {
		capUnlimited = true
	}
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
		"hq_level":       hqLv,
		"gather":         gather,
		"gather_per":     ezfyGatherBonusPer(),
		"gather_max":     gatherMax,
		"gather_have":    gatherHave,
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
	// ★ 「出征集结令上限后台管理系统可维护，最大默认 99」→ 默认 99（线上现值）。
	//   真正的上限由 ezfyGatherMax() 从 ezfy_cfg_limit.gather_max_per_order 读取，
	//   管理端「建筑上限配置」页可改，改完 cfgsReload() 即时生效。
	//   这个常量只在配置行缺失/为 0 时兜底。
	ezfyGatherMaxDefault = 99
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
//	+ 出征军官军事 × ezfyOfficerCapPerMil()            ← 2026-09-28 用户规则：军官军事累加上限（可配置）
//
// ★ 「再加个出征上限开关，默认开；关闭后出征没有上限」→
//
//	开关关掉时返回 (0, true)，调用方一律用 unlimited 判断，**不要**拿 0 去比大小。
//
// ezfyTroopCapReuse 出征上限计算的预取数据（/order/preview 并行块已取好时传入，避免重复查库）
type ezfyTroopCapReuse struct {
	techs map[int]int
	hqLv  int
	lead  *model.EzfyOfficer
}

// ezfyOrderTroopCap 出征兵力上限：司令部等级 × 1万 × 指挥艺术科技 + 集结令加成 + 军官军事加成。
// ★ 2026-10-05 reuse 可选：调用方已并行取好的 科技map/司令部等级/带队军官 时传入（预览接口），
//
//	其余调用方（createOrder 等）不传，函数内部照旧自查。
func (h *EzfyHandler) ezfyOrderTroopCap(cityId uint, gather int, officer string,
	reuse ...*ezfyTroopCapReuse) (cap int64, unlimited bool) {
	if !ezfyMarchCapOn() {
		return 0, true
	}
	var tmap map[int]int
	hqLv := 0
	var lead *model.EzfyOfficer
	if len(reuse) > 0 && reuse[0] != nil {
		r := reuse[0]
		tmap, hqLv, lead = r.techs, r.hqLv, r.lead
	}
	if tmap == nil {
		tmap = h.techMap(cityId)
	}
	if hqLv <= 0 {
		hqLv = h.buildingLevel(cityId, 13)
	}
	cap = int64(10000*hqLv) * int64(100+tmap[15]*ezfyCommandCarryPct) / 100
	if gather > 0 {
		cap += int64(gather) * ezfyGatherBonusPer()
	}
	if lead == nil && officer != "" {
		lead = h.officerByName(cityId, officer)
	}
	if lead != nil && lead.Military > 0 {
		cap += int64(lead.Military) * int64(ezfyOfficerCapPerMil())
	}
	return cap, false
}

// playerAtWar 玩家是否处于「战斗状态」（个人宣战交战中 或 所属军团宣战生效中）。
// ★ 2026-10-02 用户规则：自城派遣在非战斗状态下不设携带上限（守城兵可以自由调动），
//
//	处于战斗状态则恢复正常上限（避免被调走兵力守不住城）。
//
// ★ 2026-10-02 修复：只认「交战中」(status=2 且生效中)，**待生效宣战(status=1)不算战斗状态**，
//
//	否则刚宣战还没开打时派遣也会被卡上限（线上反馈「AI大本营派遣还是 144,000 上限」）。
func (h *EzfyHandler) playerAtWar(uid uint) bool {
	now := time.Now().UnixMilli()
	var cnt int64
	// 个人宣战交战中：status=2 且未过期（status 由 warStatus/tick 在到 EffectTime 时推进为 2）
	h.DB.Model(&model.EzfyWar{}).
		Where("status = 2 AND expire_time > ? AND (atk_user_id = ? OR def_user_id = ?)", now, uid, uid).
		Count(&cnt)
	if cnt > 0 {
		return true
	}
	// 军团宣战生效中：status=2 且 now ∈ [effect_time, expire_time)（与 corpsActiveWarBetweenCorps 同口径）
	if cid := h.corpsOfUser(uid); cid > 0 {
		h.corpsWarTick()
		var wc int64
		h.DB.Model(&model.EzfyCorpsWar{}).
			Where("status = 2 AND effect_time <= ? AND expire_time > ? AND (atk_corps_id = ? OR def_corps_id = ?)", now, now, cid, cid).
			Count(&wc)
		if wc > 0 {
			return true
		}
	}
	return false
}

// dispatchNoCap 自城派遣(8)是否放开携带上限：玩家名下任一城有生效中的免战保护令
// （★ 2026-10-02 保护令**全账号生效**）或 不处于战斗状态 → 无上限（油照常消耗）；
// 处于战斗状态 → 正常上限。
func (h *EzfyHandler) dispatchNoCap(uid uint, city *model.EzfyCity) bool {
	if h.hasAnyPeaceEffect(uid) {
		return true
	}
	return !h.playerAtWar(uid)
}

// createOrder 出征下单。
//
// ★★ 2026-10-05 性能（用户反馈「/order 3s 多」）：本函数原来在懒结算之后还有一堆
// **各自独立、串行**的查询与写入（线上单价：读 ~120ms / 写 ~240ms）：
//
//	officerByName ×4（军官校验 / 移速技能 / 军事加成 / officerGoOut 各查一次）
//	buildingLevel ×2 + techMap ×1 + troopMap ×1 + ezfyOrderTroopCap 内部又各来一遍
//	扣兵：每个兵种 SELECT + UPDATE/DELETE（2N 条）
//	集结令：每个道具一次 consumeItem（N 条写）
//	收尾 6 个写（资源 / 订单 / 扣兵 / 军官状态 / 道具 / 战报）全部串行
//
// 现在：① 懒结算改为返回**快照**，建筑等级/科技/部队全部复用（0 额外读）；
//
//	② 军官只查一次；③ 扣兵合并成 1 条 CASE UPDATE + 1 条 DELETE；④ 集结令一次扣完；
//	⑤ 收尾的独立写**并行**发出（不同表，互不依赖）。
func (h *EzfyHandler) createOrder(uid uint, city *model.EzfyCity, orderType, targetX, targetY, targetType int,
	targetId int64, troops []ezfyUnitGroup, resources map[string]int64, officer string, waitMin, gather int,
	cities ...[]model.EzfyCity) string {

	// ★ 懒结算返回快照：下面的建筑等级 / 科技等级 / 城内部队全部走内存，零额外读
	// cities 可选：调用方已查好玩家城市列表时传入，懒结算直接复用（省一条串行 RTT）。
	var cityList []model.EzfyCity
	if len(cities) > 0 {
		cityList = cities[0]
	}
	// ★★ 2026-10-05 性能：三条**互不依赖**的校验查询（防重 / 司令部在途数 / 目标是不是自己的野地）
	//   原来分散在函数三处、**串行**发出 = 3 个跨 WAN 往返（线上 ~360ms）。
	//   现在直接塞进懒结算的那个并行波里（ezfyLoadCityData 的 extra 参数），
	//   与建筑/科技/部队等 8 条查询**同时**发出 → 整段只花 1 个 RTT。
	var dupCnt, marchingCnt int64
	var ownWild *model.EzfyWildland
	d := h.ezfyLoadCityData(uid, city, cityList,
		func() {
			h.DB.Model(&model.EzfyOrder{}).
				Where("user_id = ? AND city_id = ? AND order_type = ? AND target_id = ? AND target_x = ? AND target_y = ? AND start_time >= ?",
					uid, city.ID, orderType, targetId, targetX, targetY, time.Now().UnixMilli()-3000).
				Count(&dupCnt)
		},
		func() {
			h.DB.Model(&model.EzfyOrder{}).Where("user_id = ? AND status = 0", uid).Count(&marchingCnt)
		},
		func() {
			var w model.EzfyWildland
			if err := h.DB.Where("x = ? AND y = ?", targetX, targetY).First(&w).Error; err == nil {
				ownWild = &w
			}
		})
	h.ezfySettleCity(uid, city, d, true)
	blv := d.buildingLevelsOf(h, city.ID)
	tech := d.techsOf(h, city.ID)
	cityTroops := d.troopsOf(h, city.ID)
	// ★ 防抖幂等(2026-09-24 用户反馈「出征了显示多条」)：
	//   网络超时/连点/客户端重发会让同一次出征重复下单。
	//   3 秒内同「城市+类型+目标」的订单视为重复提交，直接拒绝。
	//   （正常情况下一次作战结束后 3 秒内对同一目标重复出征几乎不可能；
	//   侦查→掠夺等不同 order_type 不受影响）
	// ★ 结果来自上面那个并行波（dupCnt），不再单独查一次
	if dupCnt > 0 {
		return "命令已下达, 请勿重复出征"
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
	// ★ 2026-09-30 用户规则：侦察只有侦察机才能侦察。
	//   侦察(orderType 1)必须携带侦察机(troopId 9)，且只能带侦察机；
	//   其它部队不能混编侦察，防止用普通部队当炮灰探路。
	const ezfyReconPlaneTroopID = 9
	if orderType == 1 {
		if len(validTroops) == 0 {
			return "侦察必须携带侦察机(侦察机)"
		}
		for _, t := range validTroops {
			if t.TroopId != ezfyReconPlaneTroopID {
				c := ezfyCfg.troop(t.TroopId)
				name := "该部队"
				if c != nil {
					name = c.Name
				}
				return "侦察只能派侦察机, 不能混编其它部队(如「" + name + "」)"
			}
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
	// ★ 2026-10-02 用户规则：海军兵种(驱逐舰/潜艇/战列舰/航母)只能出征
	//   岛屿/海底森林(海洋)/沿海平原（含建在沿海平原上的城市）目标，
	//   出征到其它地形时在出征前卡控提示（按目标坐标地形判定，覆盖野地/寇城/城市）。
	if ezfyHasNavalTroops(validTroops) && !ezfyNavalTargetAllowed(targetX, targetY) {
		return "海军部队只能出征岛屿/海底森林/沿海平原"
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
	// ★ 2026-09-28 用户规则：自己的附属野地不能侦查/掠夺/征服 ——
	//   要先到「附属野地」页把这块地[放弃]（放弃后该坐标恢复为中立野地，才能再打）。
	//   ⚠️ 野地类目标的 targetType 是 1/2，**3 才是玩家城市**（城市那边由下面
	//   「掠夺/征服玩家城需先宣战」那段管，别在这里重复拦）。
	//   归属判定与 WildlandView 下发的 owner/mine 同源：野地记录 → 城市 → UserID。
	if (orderType == 1 || orderType == 2 || orderType == 3) && targetType != 3 {
		// ★ 野地记录来自上面那个并行波（ownWild），不再单独查一次
		if ownWild != nil && ownWild.CityId > 0 {
			var oc model.EzfyCity
			if err := h.DB.First(&oc, ownWild.CityId).Error; err == nil && oc.UserID == uid {
				return "这是你自己的附属野地, 不能" + ezfyOrderTypeName(orderType) +
					"; 如要攻打请先在「附属野地」里[放弃]该野地"
			}
		}
	}
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
	// ★ 2026-10-05 性能：军官只查一次并全程复用（原来「校验/移速技能/军事加成/置出征态」各查一次 = 4 条）
	var lead *model.EzfyOfficer
	if officer != "" {
		lead = h.officerByName(city.ID, officer)
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
		// ★ 2026-10-02 用户规则澄清：**海城的陆军可以攻击陆地**（陆城/陆野均可），
		//   不再按「海城/陆城」卡控城池交战；海军兵种的目标地形限制已由上方统一校验
		//   （ezfyNavalTargetAllowed：岛屿/海底森林/沿海平原，含建在其上的城市）。
		//   即：只有海军兵种受地形限制，陆军/空军不受海城出身影响。
		// ★ 2026-09-27 免战保护令**绝对生效**（宣战也不能打）。
		//   目标城市处于免战保护期时直接拦截出征，避免部队白跑一趟。
		if h.hasCityEffect(uint(targetId), 2) {
			return "该玩家使用了免战保护, 无法出征"
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
	// 随军资源校验（★ 2026-09-30 扣减移到全部校验通过后，避免「已扣资源但后续
	// 司令部上限/目标太近/石油不足 等失败」导致资源凭空消失 —— 用户反馈运输丢资源）
	// ★ 2026-10-07 所有出征类型都能携带随军资源（不止运输/派遣）：
	//   玩家资源多了可随身带出腾仓库/防被抢，出发城照常扣减；负重上限 = 部队负重。
	if hasRes {
		f, s, o, r, g := resources["food"], resources["steel"], resources["oil"], resources["rare"], resources["gold"]
		if f < 0 || s < 0 || o < 0 || r < 0 || g < 0 {
			return "资源数量错误"
		}
		// ★ 第九轮：携带资源必须有部队来装（负重决定能带多少），军官可以不带队。
		if len(validTroops) == 0 {
			return "携带资源需要部队来装载(卡车负重最高)"
		}
		// ★ 2026-09-28 传 city.ID：负重上限含「装载技术」加成（与出征/采集同一口径）
		cap := h.ezfyCarryCapOf(validTroops, city.ID)
		if total := f + s + o + r + g; total > cap {
			return fmt.Sprintf("负重不足: 本次要携带%d, 部队负重只有%d(多带卡车可提高)", total, cap)
		}
		if city.Food < f || city.Steel < s || city.Oil < o || city.Rare < r || city.Gold < g {
			return "资源不足,无法携带"
		}
	}
	// 司令部限制（★ 复用快照里的建筑等级）
	hq := blv[13]
	if hq < 1 {
		return "需要先建造司令部"
	}
	// ★ 在途队伍数来自上面那个并行波（marchingCnt），不再单独查一次
	if int(marchingCnt) >= hq {
		return fmt.Sprintf("司令部%d级, 同时只能出征%d支队伍", hq, hq)
	}
	// ★ 2026-10-02 用户反馈「运输无上限是bug」：运输(5)也纳入携带上限校验（和其他出征一致）。
	//   仅派遣(8)在非战斗状态/免战期间无上限（油照常消耗）。
	// ★ 复用快照（科技等级 / 司令部等级 / 带队军官）—— 原来这里内部又会各查一遍
	carryCap, capUnlimited := h.ezfyOrderTroopCap(city.ID, gather, officer,
		&ezfyTroopCapReuse{techs: tech, hqLv: hq, lead: lead})
	// ★ 2026-10-02 自城派遣(8)在非战斗状态/免战期间无上限（油照常消耗）
	if orderType == 8 && h.dispatchNoCap(uid, city) {
		capUnlimited = true
	}
	if !capUnlimited && total > carryCap {
		msg := fmt.Sprintf("司令部%d级, 携带上限%d万部队", hq, carryCap/10000)
		if gm := ezfyGatherMax(); gather < gm {
			msg += fmt.Sprintf("。可使用集结令提高上限: 每个+%d, 单次最多%d个", ezfyGatherBonusPer(), gm)
		}
		return msg
	}
	distance := ezfyAbs(city.X-targetX) + ezfyAbs(city.Y-targetY)
	if distance == 0 {
		return "目标太近了"
	}
	// 耗油
	// ★ 2026-10-05 性能：这里**只改内存、不立刻写库** —— 与下面的随军资源合并成**一次**整行写
	//   （原来油一次 saveCityRes、随军资源再一次 = 2 条 ~240ms 的写往返）。
	oilCost := h.ezfyOilCost(city, orderType, distance, validTroops, resources)
	if city.Oil < oilCost {
		return fmt.Sprintf("石油不足: 本次出征需耗油%d, 当前油库仅%d", oilCost, city.Oil)
	}
	city.Oil -= oilCost

	// ★ 2026-09-30 随军资源在**全部校验通过**后才扣（司令部上限/距离/耗油都过了，
	//   不会再出现「失败但资源已扣」的丢资源 bug）
	// ★ 2026-10-07 所有出征类型都扣（不止运输/派遣）——随身带出的资源从出发城扣减
	if hasRes {
		city.Food -= resources["food"]
		city.Steel -= resources["steel"]
		city.Oil -= resources["oil"]
		city.Rare -= resources["rare"]
		city.Gold -= resources["gold"]
	}

	// ★ 2026-10-05 性能：科技/驿站等级走快照（原来 techMap 2 条 + buildingLevel 1 条）
	station := blv[20]
	travelSec := int64(distance) * 60 * 300 / int64(slowest)
	travelSec = travelSec * 100 / int64(100+tech[12]*2)
	travelSec = travelSec * 100 / int64(100+station*3)
	// 带队军官「移速」技能: 行军 +10%
	// ★ 2026-10-05 性能：复用上面已经查好的 lead（原来这里又各查一次，共 4 次军官查询）
	if s := h.officerSpeedSkillBonus(lead); s > 0 {
		// ★ 2026-10-06 移速技能随军官等级自动升级：-N% 行军时间按当前加成算
		travelSec = travelSec * 100 / int64(100+s)
	}
	// ★ 2026-09-28 军官军事加成出征速度：每点军事 +0.1%（可配）
	if lead != nil && lead.Military > 0 {
		travelSec = int64(float64(travelSec) * 100 / (100 + float64(lead.Military)*ezfyOfficerSpeedPerMil()))
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

	// ★★ 2026-10-05 性能：所有校验都过了，从这里开始**写库**。
	//   这些写落在**不同的表**上、互不依赖 → **并行**发出。
	//   原来它们是 6 条串行写（城市资源 / 集结令 / 订单 / 扣兵 / 军官状态 / 战报），
	//   线上一次写 ~240ms → 光收尾就 ~1.4s。并行后只花「最慢的那一条」≈ 250ms。
	var wwg sync.WaitGroup
	// ① 城市资源（油 + 随军资源，已合并成一次整行写）
	wwg.Add(1)
	go func() { defer wwg.Done(); h.saveCityRes(city) }()
	// ② 订单行
	wwg.Add(1)
	go func() { defer wwg.Done(); h.DB.Create(&order) }()
	// ③ 从城市扣兵：一条 CASE UPDATE + 一条清零 DELETE（原来每个兵种 SELECT+UPDATE = 2N 条）
	wwg.Add(1)
	go func() { defer wwg.Done(); h.deductCityTroops(city.ID, validTroops) }()
	// ④ 集结令：一次扣完（原来是每个道具一条 consumeItem）
	if gather > 0 {
		wwg.Add(1)
		go func() { defer wwg.Done(); h.consumeItemN(uid, ezfyGatherItemID, gather, "出征集结令") }()
	}
	// ⑤ 带队军官置出征态（复用已查好的 lead，不再按名字查一次）
	if lead != nil {
		wwg.Add(1)
		go func() { defer wwg.Done(); h.officerGoOutByID(lead.ID, true) }()
	}
	// ⑥ 雷达站事前预警（自己的读 + 一条战报写，也一起并行）
	wwg.Add(1)
	go func() {
		defer wwg.Done()
		h.ezfyOrderRadarWarn(city, &order, orderType, targetType, targetId, officer, validTroops)
	}()
	wwg.Wait()
	return ""
}

// ezfyOrderRadarWarn 出征时给**被攻击方**发事前预警（雷达站 + 侦察技巧决定能看到多少）。
//
// ★ 2026-10-05 从 createOrder 里抽出来：① 让收尾的 6 个写能并行；
// ② 本段自己的读（目标城 + 情报等级）也不再串在写前面。
//
//	侦查(1)      → 「被侦查报告」（军情警讯）
//	掠夺(2)/征服(3) → 「军情警报: 敌军来袭!」，细节随情报等级递增
//
// ★ 2026-09-25 「军情警讯里面展示下对面城市名字以及地址，雷达站以及科技满足的情况下展示，
// 不然我只知道有人打我，不知道哪来的」：情报等级 = 雷达站等级 + **侦察技巧科技等级**（合计封顶 10）。
// 「出发城市（名称+坐标）」门槛定在 **2 级**，并且够等级时会**写进战报标题**
// —— 军情警讯列表只显示标题，不点进去也要看得见。
//
// 注意：这里只发**事前预警**；被掠夺/城破这类**事后结果**报告在 processArrive 里发，
// 不受雷达站限制 —— 否则玩家资源被抢光了却毫不知情。
func (h *EzfyHandler) ezfyOrderRadarWarn(city *model.EzfyCity, order *model.EzfyOrder,
	orderType, targetType int, targetId int64, officer string, validTroops []ezfyUnitGroup) {
	// 雷达站预警：**能不能提前看见，取决于被攻击方自己城市的雷达站等级**。
	//
	//	侦查(1)      → 「被侦查报告」（军情警讯）
	//	掠夺(2)/征服(3) → 「军情警报: 敌军来袭!」，细节随情报等级递增
	//
	// ★ 2026-09-25 「军情警讯里面展示下对面城市名字以及地址，雷达站以及科技满足的情况下展示，
	//
	//	不然我只知道有人打我，不知道哪来的」：
	//	情报等级 = 雷达站等级 + **侦察技巧科技等级**（合计封顶 10），见 h.ezfyIntelLevel。
	//	「出发城市（名称+坐标）」门槛定在 **2 级**（雷达站1级 + 侦察技巧1级就能看到），
	//	并且够等级时会**写进战报标题** —— 军情警讯列表只显示标题，不点进去也要看得见。
	//
	// 注意：这里只发**事前预警**；被掠夺/城破这类**事后结果**报告在 processArrive 里发，
	// 不受雷达站限制 —— 否则玩家资源被抢光了却毫不知情。
	if targetType == 3 && targetId > 0 && (orderType == 1 || orderType == 2 || orderType == 3) {
		var target model.EzfyCity
		if err := h.DB.First(&target, targetId).Error; err == nil && target.UserID > 0 {
			radar := h.ezfyIntelLevel(target.ID)
			if radar >= 1 && orderType == 1 {
				body := "有敌军对我方城市进行了侦查!\n"
				if radar >= 3 {
					body += fmt.Sprintf("侦查方城市: %s(%d,%d)\n", city.Name, city.X, city.Y)
				}
				if radar >= 5 {
					body += fmt.Sprintf("侦查时间: %s\n", time.UnixMilli(order.StartTime).Format("01-02 15:04"))
				}
				body += fmt.Sprintf("(情报等级%d: 雷达站等级越高、侦察技巧越高, 情报越详细)", radar)
				h.addReport(target.UserID, 6, "被侦查报告: "+city.Name, body, "", 0, target.ID)
			}
			if radar >= 1 && orderType != 1 {
				// ★ 出发城市（名称+坐标）放正文**最前面** —— 玩家最想知道的就是「谁、从哪来」
				originLine := ""
				if radar >= 2 {
					originLine = fmt.Sprintf("出发城市: %s(%d,%d)\n", city.Name, city.X, city.Y)
				}
				warn := "军情警报: 敌方部队正向我方城市进发!\n" + originLine
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
				// ★ 把「还差什么才能看到出发城市」直接告诉玩家，否则他永远不知道该怎么解锁
				if radar < 2 {
					warn += fmt.Sprintf("(情报等级%d 不足: 雷达站+侦察技巧 合计到 2 级才能看到来袭城市与坐标)\n", radar)
				}
				warn += fmt.Sprintf("(情报等级%d = 雷达站%d级 + 侦察技巧%d级)", radar, h.buildingLevel(target.ID, 21), h.techMap(target.ID)[ezfyReconTechID])
				title := "军情警报: 敌军来袭!"
				if originLine != "" {
					// 列表只显示标题 → 把来源也带上，不点进去就能看到
					title += fmt.Sprintf(" 来自 %s(%d,%d)", city.Name, city.X, city.Y)
				}
				h.addReport(target.UserID, 6, title, warn, "", 0, target.ID)
			}
		}
	}
}

// deductCityTroops 出征后从城市扣兵。
//
// ★ 2026-10-05 性能：原来每个兵种「SELECT 行 + UPDATE/DELETE」= 2N 条跨 WAN 往返；
// 现在合并成**一条 CASE UPDATE**（一次改完所有兵种）+ **一条 DELETE**（清掉归零/负数行）。
// 数量用 GREATEST(count-?,0) 夹取，绝不会写出负数兵力（与全站防负数防线一致）。
func (h *EzfyHandler) deductCityTroops(cityId uint, troops []ezfyUnitGroup) {
	if len(troops) == 0 {
		return
	}
	ids := make([]int, 0, len(troops))
	var expr strings.Builder
	expr.WriteString("CASE troop_id")
	args := make([]interface{}, 0, len(troops)*2)
	for _, t := range troops {
		if t.Count <= 0 {
			continue
		}
		ids = append(ids, t.TroopId)
		expr.WriteString(" WHEN ? THEN GREATEST(count - ?, 0)")
		args = append(args, t.TroopId, t.Count)
	}
	if len(ids) == 0 {
		return
	}
	expr.WriteString(" END")
	h.DB.Model(&model.EzfyCityTroop{}).Where("city_id = ? AND troop_id IN ?", cityId, ids).
		Update("count", gorm.Expr(expr.String(), args...))
	h.DB.Where("city_id = ? AND troop_id IN ? AND count <= 0", cityId, ids).
		Delete(&model.EzfyCityTroop{})
}

// 雷达站预警相关常量
const (
	ezfyRadarBuildingID = 21 // 建筑：雷达站（「对敌军入侵进行预警」）
	ezfyReconTechID     = 12 // 科技：侦察技巧（原来只加行军速度，现在同时加成军情情报）
	ezfyIntelMaxLevel   = 10 // 情报等级封顶
)

// ezfyIntelLevel 被攻击方城市的「军情情报等级」= 雷达站等级 + 侦察技巧等级（合计封顶 10）。
//
// ★ 2026-09-25 「军情警讯里面展示下对面城市名字以及地址，雷达站以及科技满足的情况下
//
//	展示，不然我只知道有人打我，不知道哪来的」。
//
// 规则：
//   - **雷达站（建筑 21）= 主渠道**：0 级收不到任何事前预警（保持原规则不变）
//   - **侦察技巧（科技 12）= 加成**：每 1 级让情报等级 +1 —— 名字本来就叫「侦察」，
//     原来只加行军速度，现在顺带加成情报，科技终于有用了
//   - **「出发城市（名称+坐标）」门槛 = 2 级** → 雷达站 1 级 + 侦察技巧 1 级 就能看到；
//     只有雷达站 1 级且没升科技时，正文会明确提示「合计到 2 级才能看到」
//
// 逐级解锁：2=出发城市+意图 / 3=预计到达 / 4=统帅 / 5=出发时间 / 7=兵力(模糊) / 9=兵力(精确)
func (h *EzfyHandler) ezfyIntelLevel(cityID uint) int {
	radar := h.buildingLevel(cityID, ezfyRadarBuildingID)
	if radar <= 0 {
		return 0
	}
	lv := radar + h.techMap(cityID)[ezfyReconTechID]
	if lv > ezfyIntelMaxLevel {
		lv = ezfyIntelMaxLevel
	}
	return lv
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
			// ★ 2026-09-30 行军计谋使用标记：scheme_fast=神兵天降已用  scheme_back=战略转移已用
			"scheme_fast": o.SchemeUsed & 1,
			"scheme_back": (o.SchemeUsed >> 1) & 1,
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
// ★ 「出征队列可以取消」：原来只允许 type=7(驻守采集) 召回，
// 其余命令一律返回「该命令不支持召回」。现在所有**还在外面**的命令
// （status 0 行军中 / 1 驻守中）都能取消，部队原路返回出发城市。
//
// 返程时间：
//   - 行军中(0)：已经走了多久就花多久回去，最少 10 秒
//   - 驻守中(1)：按单程行军时长返航
//   - 驻军(3, 增援到友军城)：按单程行军时长返航（★ 2026-10-02 允许单独召回驻军）
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
	// ★ 2026-10-02 「自己也能单独召回驻守的军队」：
	//   出站驻军(增援到友军城, status=3)允许单独召回, 按单程返航回出发城市。
	//   仅允许「活跃驻军」(result 为空, 未返航过) 且目标城属于他人
	//   —— 已归队的驻军(Result=兵力)与增援自己城市的订单召回会重复入兵, 一律拦截。
	if order.Status == 3 && order.OrderType == 6 {
		if order.Result != "" {
			resp.ParamError(c, "该驻军已返航归队, 无法召回")
			return
		}
		var tcity model.EzfyCity
		if err := h.DB.First(&tcity, order.TargetId).Error; err != nil || tcity.UserID == uid {
			resp.ParamError(c, "该部队不在盟友城市, 无法召回")
			return
		}
	} else if order.Status != 0 && order.Status != 1 {
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
	// ★ 2026-10-07 所有出征类型的随军资源都原样带回（不止运输/派遣）：
	//   出发时已从城里扣掉，召回/返航时必须带回，否则随身资源凭空消失。
	//   随身资源并入 Carry：采集部队(驻守采集)的采集产出也在 Carry 里，不能覆盖丢产出。
	carry := order.Carry
	if strings.TrimSpace(order.Resources) != "" {
		c := parseCarry(order.Carry)
		r := parseCarry(order.Resources)
		c.Food += r.Food
		c.Steel += r.Steel
		c.Oil += r.Oil
		c.Rare += r.Rare
		c.Gold += r.Gold
		carry = carryJSON(c)
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
// processOrders 懒结算订单。★ 2026-10-04 cities 为可选参数：调用方（如 /view）已把
// 玩家城市列表查好时传入，processIncoming 直接复用，省掉一次「重查城市 id」的 RTT。
func (h *EzfyHandler) processOrders(uid uint, cities ...[]int64) {
	if !h.enterProcess(uid) {
		// 已在结算中（递归回调）→ 跳过，交回上层继续处理，避免无限自喂
		return
	}
	defer h.exitProcess(uid)

	now := time.Now().UnixMilli()
	// ★ 2026-10-04 性能（用户反馈「军官/展示接口线上 2s~3s 卡」）：
	//   快速路径 —— 先拉订单（含 status IN (0,1,2,5,6,98)）。列表为空说明该玩家
	//   **没有任何未结算订单**（自然也没有 status=98 死单），此时死单自愈 UPDATE 和
	//   战场推进都可以跳过，只查一次「敌军攻打我方」的订单后直接返回。
	//   原实现先发死单自愈 UPDATE 再 SELECT，玩家空闲时那条 UPDATE 没有匹配行也要
	//   白白跨 WAN 往返一次 —— 高频展示接口每次都被它拖慢。
	// ★ 2026-10-04 性能（用户反馈「军官接口线上 3s 卡」）：原查询不带 status 过滤，
	//   把玩家**全部历史订单**（含 3完成/4阵亡，行内还有 8KB+ 的 troops/result/battle_result
	//   大字段）每次都从 RDS 全量拉回来 —— 活跃玩家几百上千行、一次几 MB，跨 WAN 必卡。
	//   结算循环实际只处理 0/1/2/5/6/98 六种状态，历史行读回来也不参与，纯浪费。
	//   idx_user 单列索引可快速定位该玩家，再按 status IN 过滤后行数骤降。
	// ★ 2026-10-05 性能：「我的订单」与「来袭订单」两条查询**互不依赖**，改成并行取 ——
	//   原来串行 2 个跨 WAN 往返（线上 ~230ms），而它们是所有展示/操作接口懒结算的必经两步。
	var orders, incoming []model.EzfyOrder
	incReady := false
	var pw sync.WaitGroup
	pw.Add(2)
	go func() {
		defer pw.Done()
		h.DB.Where("user_id = ? AND status IN (0,1,2,5,6,98)", uid).Order("id ASC").Find(&orders)
	}()
	go func() { defer pw.Done(); incoming, incReady = h.fetchIncoming(uid, now, cities...) }()
	pw.Wait()

	// 处理完自己的订单后，用**已取好**的来袭订单列表结算（原来的 processIncoming 串行版）
	settleIncoming := func() {
		if !incReady {
			return
		}
		for i := range incoming {
			o := &incoming[i]
			// processArrive 用 uid 参数定位**攻方**城市（cityOfOrder 拿 order.CityId），
			// 所以这里必须传 o.UserID（攻方），不是当前轮询的 uid（守方）。
			h.processArrive(o.UserID, o, now)
		}
	}
	if len(orders) == 0 {
		settleIncoming()
		return
	}
	// ★ 死单自愈：status=98(结算中) 超过 60 秒没被写回正常状态的订单，
	//   说明结算过程异常退出（老部署强杀进程等），重置回「行进」，下次到达再结算。
	//   （orders 非空时才可能有 98 死单，空列表跳过此 UPDATE）
	//
	// ★ 2026-10-05 性能：再加一层「列表里根本没有 98 单就不发这条 UPDATE」——
	//   活跃玩家的 orders 恒非空，于是每个请求都要白打一次跨 WAN 的写往返
	//   （实测 rows:0，纯浪费）。98 是极罕见的异常态，绝大多数请求都能跳过。
	if hasProcessingOrder(orders) {
		h.DB.Model(&model.EzfyOrder{}).
			Where("user_id = ? AND status = ? AND updated_at < ?", uid, ezfyOrderStatusProcessing,
				time.Now().Add(-60*time.Second)).
			Updates(map[string]interface{}{"status": 0})
	}
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
			// ★ 2026-09-23 目标已被抢占 → 部队「等待」。
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
	// ★ 2026-10-04 修复「打完了还一直显示」：攻方下线后没人 tick 战场，
	//   守方自己的轮询也把「正在打我方城市」的战场懒推进：打完了立刻收尾
	//   （finishToOrder 把订单重置回行进 status=0，随后 processIncoming 结算它）。
	//   只处理订单仍处于「战斗中(5)」的战场 —— 已结算订单留下的僵尸行由 ezfyBattleTick
	//   自愈 + 军队动态查询加状态过滤兜底，避免把已完成的订单重新拉回结算。
	//   ★ 2026-10-04 性能：上面过滤后的 orders 里没有战斗中订单就不查 defPending（省 1 次子查询）。
	var defPending []model.EzfyBattle
	if hasBattleOrder(orders) {
		h.DB.Where("def_user_id = ? AND status = 1 AND order_id IN (SELECT id FROM ezfy_order WHERE status = ?)",
			uid, ezfyOrderStatusBattle).Find(&defPending)
	}
	for i := range defPending {
		b := &defPending[i]
		if _, done := h.ezfyBattleTick(b, now); done {
			h.ezfyBattleFinishToOrder(b, now)
		}
	}

	// ★ 2026-09-23 「敌人来了没提示 / 军情警讯不及时」：
	//   防守方自己的轮询也能触发「打到我家城市的敌军到达 + 开战场」——
	//   否则进攻方下线时，敌军会一直卡在「行进中」，防守方连「敌军已抵达」都收不到。
	// ★ 2026-10-05 性能：来袭订单已在本函数开头**并行**取好，这里直接结算（不再查库）。
	settleIncoming()
}

// hasProcessingOrder 给定订单列表里是否存在「结算中(98)」的异常残留订单。
// 用于跳过「死单自愈 UPDATE」——没有 98 单时那条 UPDATE 恒 rows:0，纯浪费一次写往返。
func hasProcessingOrder(orders []model.EzfyOrder) bool {
	for i := range orders {
		if orders[i].Status == ezfyOrderStatusProcessing {
			return true
		}
	}
	return false
}

// hasBattleOrder 给定订单列表里是否存在「战斗中」订单
func hasBattleOrder(orders []model.EzfyOrder) bool {
	for i := range orders {
		if orders[i].Status == ezfyOrderStatusBattle {
			return true
		}
	}
	return false
}

// processIncoming 把「正在攻打 uid 名下城市、已到点」的敌方订单结算掉（开战场 / 发军情警讯）。
// 只处理 target_type=3（玩家城）且 status=0（行进中，到点）的订单，交给 processArrive 走统一流程。
// ★ 2026-10-04 cities 可选：调用方（如 /view）已查好城市 id 时传入，省一次 Pluck 的 RTT。
//
// ★ 2026-10-04 性能（用户反馈「/view 2s+」）：3 秒内刚确认过「无来袭订单」就直接跳过，
//
//	省一条串行 RDS（/view 每 3 秒一次 cache miss，常 idle 玩家这条 SELECT 每次空转）。
//	敌军新出征的到达最多滞后 3 秒被发现（与 /view 缓存 TTL 同级，可接受）。
var (
	ezfyIncomingMemoMu sync.Mutex
	ezfyIncomingMemo   = map[uint]int64{} // uid → 最近一次「确认无敌军来袭」的毫秒时间戳
)

// fetchIncoming 只做「取来袭订单」这一步（不发军情/不开战场），返回 (列表, 是否需要结算)。
//
// ★ 2026-10-05 性能：从 processIncoming 拆出来，让调用方（processOrders）能把这条查询
//
//	与「我的订单」查询**并行**发出（原来两条串行 = 2 个跨 WAN 往返）。
//	第二返回值 ready=false 表示「3 秒内刚确认过无敌军来袭」→ 调用方直接跳过结算。
func (h *EzfyHandler) fetchIncoming(uid uint, now int64, cities ...[]int64) ([]model.EzfyOrder, bool) {
	ezfyIncomingMemoMu.Lock()
	last, ok := ezfyIncomingMemo[uid]
	ezfyIncomingMemoMu.Unlock()
	if ok && now-last < ezfyViewCacheTTLMs {
		return nil, false
	}
	var ids []int64
	if len(cities) > 0 && cities[0] != nil {
		ids = cities[0]
	} else {
		h.DB.Model(&model.EzfyCity{}).Where("user_id = ?", uid).Pluck("id", &ids)
	}
	if len(ids) == 0 {
		return nil, false
	}
	var orders []model.EzfyOrder
	h.DB.Where("status = 0 AND target_type = 3 AND target_id IN ? AND arrive_time <= ?", ids, now).
		Order("id ASC").Find(&orders)
	if len(orders) == 0 {
		// 确认无来袭 → 记 memo（下次 3 秒内跳过）
		ezfyIncomingMemoMu.Lock()
		if len(ezfyIncomingMemo) > 16384 {
			ezfyIncomingMemo = map[uint]int64{}
		}
		ezfyIncomingMemo[uid] = now
		ezfyIncomingMemoMu.Unlock()
		return nil, false
	}
	return orders, true
}

func (h *EzfyHandler) processIncoming(uid uint, now int64, cities ...[]int64) {
	orders, ready := h.fetchIncoming(uid, now, cities...)
	if !ready {
		return
	}
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
		// ★ 2026-09-25 「各项资源有最大的配置」→ 入库统一走 ezfyResAddExpr：
		//   无条件累加，累加到配置的「资源最大值」（默认 100 亿）为止，且不拉低已有更大值。
		h.DB.Model(&model.EzfyCity{}).Where("id = ?", city.ID).Updates(map[string]interface{}{
			"food":  ezfyResAddExpr("food", c.Food),
			"steel": ezfyResAddExpr("steel", c.Steel),
			"oil":   ezfyResAddExpr("oil", c.Oil),
			"rare":  ezfyResAddExpr("rare", c.Rare),
			"gold":  ezfyResAddExpr("gold", c.Gold),
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
	// ★ 2026-10-02 驻军(增援到盟友城)返航归队后订单已完结：兵力/军官已入城,
	//   直接删除订单 —— 否则 status=3 常驻残留会被军情误判为「驻守中」、
	//   且一直占用盟友城驻军槽位, 导致新驻军无法增援。
	if order.OrderType == 6 && order.TargetType == 3 {
		h.DB.Delete(&model.EzfyOrder{}, order.ID)
	}
}

// beginReturn 异常返航: 兵力无损带回
// ezfyOilCost 出征耗油(运输按携带资源量计, 其余按兵种油耗×数量×距离计)
//
// ★ 「加个出征油耗开关，默认开；关了出征消耗油 0」→ 关掉时直接返回 0。
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
//   - 宝物每期至少 1 件直接进背包(不受负重限制), 每期另有 20% 概率多 1 件,
//     采集时间越长期数越多宝物越多; 平原/沿海平原无珠宝则该期没有宝物。
//
// ★ 2026-09-28 用户规则修正（本轮核心）:
//   - 资源**累积进部队负重(carry)**（负重封顶、多采部分丢弃），不直接入城；
//     「收获」/「停止采集」/「召回」时取回负重入城（见各入口）。
//   - **负重满了不再自动停止**：封顶后部队继续驻守采集，由玩家手动[停止采集]/[一键收获]/[召回]取回。
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
			"所采集的野地已不属于我方, 采集部队已返航。", "", order.ID, order.CityId)
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
	// ★ 2026-09-28 规则修正：产出**累积进部队负重(carry)**（负重封顶、多采部分丢弃），
	//   不直接入城；待玩家[停止采集]/[一键收获]/[召回]时取回负重入城（见各入口）。
	//   从负重取回仍走「前后差值」口径，保证反馈数=实际进账数。
	//
	// ★★ 2026-09-30 修复「达到负重还丢资源」：
	//   原来负重满了仍照常推进 arrive_time，每次结算都把**整期产出丢弃**（addCarryToOrder
	//   按比例折算只装进 room，多出的部分永久消失），玩家收益持续损失。
	//   现在改成**装不下负重的部分直接入起点城市**（与「收获即入城」同一口径，走 harvestToCity）：
	//   负重能装的进负重、超出的部分直接入库 —— 资源永远不会凭空消失，也不影响部队继续驻守采集。
	//   做法：先按负重剩余空间算「能装多少」，剩下的拆出来直接入库。
	city := h.cityOfOrder(order, uid)
	cur := parseCarry(order.Carry)
	room := h.ezfyCarryCap(order) - cur.total()
	if room < 0 {
		room = 0
	}
	direct := int64(0) // 超出负重、直接入城的资源量
	if amt > room && room > 0 {
		direct = amt - room
		// 负重部分按比例装（食物/钢铁/石油/稀矿）
		scale := func(v int64) int64 { return v * room / amt }
		sf, ss, so, sr := scale(food), scale(steel), scale(oil), scale(rare)
		h.addCarryToOrder(order, sf, ss, so, sr, 0)
		// 超出部分按比例直接入起点城市（不丢）
		dF := food - sf
		dS := steel - ss
		dO := oil - so
		dR := rare - sr
		if city != nil {
			h.harvestToCity(int64(city.ID), dF, dS, dO, dR, 0)
		}
	} else {
		_, _ = h.addCarryToOrder(order, food, steel, oil, rare, 0)
	}
	cur = parseCarry(order.Carry)
	desc := fmt.Sprintf("采集部队在野地%d级(%d,%d)驻守满%d期\n产出: %s%d",
		level, wl.X, wl.Y, periods, resName, amt)
	if gainPct > 100 {
		desc += fmt.Sprintf("(等级×%d期, 军官后勤加成 +%d%%)", periods, gainPct-100)
	} else {
		desc += fmt.Sprintf("(等级×%d期)", periods)
	}
	desc += fmt.Sprintf("\n已入负重 %d/%d", cur.total(), h.ezfyCarryCap(order))
	if direct > 0 {
		desc += fmt.Sprintf("\n负重已满, 超出部分 %d 已直接入起点城市(不丢弃)", direct)
	}
	// 宝物: 每满一个采集周期(每期)至少 1 件, 直接进背包; 每期另有 20% 概率多 1 件
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
	// ★ 2026-09-28 用户规则「负重封顶+无自动停止」：负重满了就封顶（产出累积到 carry 上限，
	//   多采部分丢弃），但部队**不自动停止**、持续采集；由玩家手动[停止采集]/[一键收获]/[召回]取回负重。
	nextArrive := now + ezfyDispatchPeriod()
	if h.ezfyCarryFull(order) {
		desc += "负重已满, 部队仍继续驻守采集(多采部分丢弃)。可点[停止采集]或[一键收获]取回负重。\n"
	} else {
		desc += fmt.Sprintf("部队继续驻守采集, 可随时[召回]或[停止采集]。\n负重 %d/%d\n",
			cur.total(), h.ezfyCarryCap(order))
	}
	order.ArriveTime = nextArrive
	order.Result = order.Troops
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
	// ★ 2026-09-28 「采集后勤加成率可调」→ 后勤点数先乘 ezfyOfficerGatherMult 再折算百分比。
	gainPct := 100
	if officer := h.officerByName(uint(order.CityId), order.Officer); officer != nil {
		gainPct += int(float64(officer.Logistics) * ezfyOfficerGatherMult())
		if gainPct > 200 {
			gainPct = 200
		}
	}
	// ★ 2026-09-28 「采集和野地等级有关，越高级采越多」：
	//   每期基础 = 800 × (等级 ^ gatherLevelPow)，让高等级加速增长（默认幂次 1.3）；再乘后勤加成%。
	per := int64(800 * math.Pow(float64(wl.Level), ezfyGatherLevelPow()) * float64(gainPct) / 100)
	mult := int64(4)
	if wl.WildType == 2 {
		mult = 3
		// ★ 2026-09-28 「海野采集更高些，给海野加个系数 1~2」：
		//   海野基础陆海系数低(3 vs 陆地4)，乘上本系数拉高海野采集收益（默认 1.5 → 4.5，比陆地更高）。
		if sm := ezfyGatherSeaMult(); sm != 1 {
			mult = int64(float64(mult) * sm)
		}
	}
	amt := per * mult
	if ms > 0 && ms < ezfyDispatchPeriod() {
		amt = amt * ms / ezfyDispatchPeriod()
	}
	// ★ 2026-09-25 「采集资源倍率也加到系统管理里」→ 产出 × 倍率（默认 1 = 原样）
	if gm := ezfyGatherResMult(); gm != 1 {
		amt = int64(float64(amt) * gm)
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
//
//	至少给 1 点资源, 不再设 1 分钟硬门槛。
func (h *EzfyHandler) settleDispatchOnRecall(uid uint, order *model.EzfyOrder, now int64) {
	if order.ArriveTime <= 0 {
		return // 空闲驻守: 还没开始采集, 没有产出
	}
	if now >= order.ArriveTime {
		h.settleDispatch(uid, order, now)
		return
	}
	// 未满一期: 提前召回, 资源按已驻守时长比例结算, 不做宝物
	h.settlePartialCollect(uid, order, now, "提前召回")
}

// settlePartialCollect 未满一个采集周期的部分结算(「停止采集」/「提前召回」共用):
//   - 资源按已进入本期的时长比例折算，**直接入起点城市**（★ 2026-09-28 起不再进部队 carry）;
//   - **不做宝物**(不满一个采集周期无宝物);
//   - 只把结果写进 result, **不接回** —— 是否返航由调用方决定。
//
// label: 描述文案(「停止采集」/「提前召回」), 用于战报正文与标题。
func (h *EzfyHandler) settlePartialCollect(uid uint, order *model.EzfyOrder, now int64, label string) {
	if order.TargetId <= 0 {
		return
	}
	var wl model.EzfyWildland
	if err := h.DB.First(&wl, order.TargetId).Error; err != nil || wl.CityId != order.CityId {
		return
	}
	started := order.ArriveTime - ezfyDispatchPeriod() // 本期起算点(=上次结算或开始采集时间)
	elapsed := max64(1, now-started)
	if elapsed > ezfyDispatchPeriod() {
		elapsed = ezfyDispatchPeriod()
	}
	food, steel, oil, rare, gainPct, resName := h.dispatchGatherYield(order, &wl, elapsed)
	// ★ 2026-09-28 规则修正：产出**累积进部队负重(carry)**（负重封顶、多采部分丢弃），
	//   不直接入城；取回负重入城由各入口（StopCollect/HarvestAll/RecallAll）负责。
	// ★★ 2026-09-30 同 settleDispatch 的修复：负重满了的部分直接入起点城市，不丢弃。
	//   这里入口（停止/收获/召回）随后都会 harvestCarryToCity，负重装不下就按比例拆出直接入库。
	city := h.cityOfOrder(order, uid)
	cur := parseCarry(order.Carry)
	room := h.ezfyCarryCap(order) - cur.total()
	if room < 0 {
		room = 0
	}
	amt := food + steel + oil + rare
	direct := int64(0)
	if amt > room && room > 0 {
		direct = amt - room
		scale := func(v int64) int64 { return v * room / amt }
		sf, ss, so, sr := scale(food), scale(steel), scale(oil), scale(rare)
		h.addCarryToOrder(order, sf, ss, so, sr, 0)
		dF, dS, dO, dR := food-sf, steel-ss, oil-so, rare-sr
		if city != nil {
			h.harvestToCity(int64(city.ID), dF, dS, dO, dR, 0)
		}
	} else {
		_, _ = h.addCarryToOrder(order, food, steel, oil, rare, 0)
	}
	cur = parseCarry(order.Carry)
	desc := fmt.Sprintf("采集部队在野地%d级(%d,%d)%s, 按驻守时长折算资源\n产出: %s%d",
		wl.Level, wl.X, wl.Y, label, resName, amt)
	if gainPct > 100 {
		desc += fmt.Sprintf("(等级×%d分钟, 军官后勤加成 +%d%%)", max64(elapsed/60000, 1), gainPct-100)
	} else {
		desc += fmt.Sprintf("(等级×%d分钟)", max64(elapsed/60000, 1))
	}
	desc += fmt.Sprintf("\n已入负重 %d/%d", cur.total(), h.ezfyCarryCap(order))
	if direct > 0 {
		desc += fmt.Sprintf("\n负重已满, 超出部分 %d 已直接入起点城市(不丢弃)", direct)
	}
	desc += fmt.Sprintf("\n（%s不满一个采集周期, 本期没有宝物, 取回负重需[停止采集]/[召回]入城）", label)
	order.Result = order.Troops
	h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
		Updates(map[string]interface{}{"result": order.Result, "carry": order.Carry})
	h.addReport(uid, 5, "采集报告: "+label+"结算", desc, "", order.ID)
}

// ezfyOrderTargetBusy 目标是否已被别的玩家「抢先指挥」。
//
// ★ 2026-09-23 A、B 出征同一个目标，A 已经在指挥(战斗中)的话，
//
//	B 应当「等待」，不能再同时开一个指挥室。
//	判断口径：同目标(target_type + 坐标)下存在**其他**订单处于「战斗中」(status=5)，
//	或存在**更早**的「等待」(status=6)订单（按 id 排队，防止多个等待者互相死锁）。
//	这里的 status=5 即「有进行中的战场在等玩家指挥」，把它当成目标被占用。
//
// ★ 2026-09-24 「玩家城市被征服/被掠夺中时，后到的攻击队伍进等待队列」：
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
	// ★ 2026-10-05 名将野地按玩家判定：已抓到守将的玩家 → 该坐标对其是普通野地，
	//   跳过活动结算，走下面通用战斗分支（普通野地守军/可占领）。
	if order.OrderType == 2 || order.OrderType == 3 {
		if act := h.ezfyActTargetType(order.TargetX, order.TargetY); act > 0 &&
			!h.playerOwnsActWildGeneral(uid, order.TargetX, order.TargetY) {
			h.processActivityBattle(uid, city, order, now, act)
			return
		}
	}

	// 采集(4)/派遣(7): 到达已占领野地 → 转为常驻驻军(空闲待命)
	// ★ 2026-09-24 用户规则: 到达后驻守**空闲**, 必须手工点[采集](或驻军区的[一键采集])
	//   才开始采集 —— 不再自动进采集。开始采集(StartCollect)后每满一个采集周期结算一期:
	//   资源**直接入起点城市**(等级越高越多, 军官后勤加成), 宝物直接进背包(每期至少 1 件);
	//   ★ 2026-09-28：负重装满后自动停止采集(原地待命)。空闲驻军用 arrive_time=0 表示(不参与结算)。
	if order.OrderType == 4 || order.OrderType == 7 {
		var wl model.EzfyWildland
		if err := h.DB.First(&wl, order.TargetId).Error; err != nil || wl.CityId != order.CityId {
			h.beginReturn(order, now, 0)
			h.addReport(uid, 5, "采集报告: 野地丢失",
				"采集目标野地已不属于我方, 采集部队已返航。", "", order.ID, order.CityId)
			return
		}
		order.Status = 1
		order.OrderType = 7 // 统一口径为驻守采集, 后续结算/一键收获/一键召回都按 7 处理
		order.Result = order.Troops
		order.ArriveTime = 0 // 0 = 空闲驻守, 待玩家手工[采集]才开始采集
		h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
			Updates(map[string]interface{}{"status": 1, "order_type": 7, "result": order.Troops, "arrive_time": 0})
		h.addReport(uid, 5, "采集报告: 部队已抵达",
			fmt.Sprintf("采集部队已抵达野地(%d,%d)驻守, 当前空闲待命。\n请到「军情 → 驻军」点[采集](或[一键采集])开始采集: 每满一个采集周期结算一期, 资源**直接入库到出发城市**(等级越高越多, 军官后勤每点+1%%), 宝物直接进背包(每期至少1件)。负重装满会自动停止采集。可随时[召回]撤兵。",
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
			"food":  ezfyResAddExpr("food", f),
			"steel": ezfyResAddExpr("steel", s),
			"oil":   ezfyResAddExpr("oil", o),
			"rare":  ezfyResAddExpr("rare", r),
			"gold":  ezfyResAddExpr("gold", g),
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
	// ★ 2026-10-02 盟军驻军改造():
	//   · 增援**自己的城市** → 兵力并入目标城(部队调动, 行为不变);
	//   · 增援**盟友城市** → **不送兵**, 兵力保留在订单 = 一个「驻军队列」;
	//     敌军进攻该城时, 守方部队 = 驻军队列(按到达先后) + 友军自身部队(最后),
	//     先打先驻守的队列, 全部驻军被消灭后才轮到友军自己的部队(见战斗结算守方部队构建)。
	if order.OrderType == 6 {
		var target model.EzfyCity
		if err := h.DB.First(&target, order.TargetId).Error; err != nil {
			h.beginReturn(order, now, 0)
			h.addReport(uid, 5, "增援报告: 目标城市不存在",
				"增援目标城市已不存在, 增援部队已返航。", "", order.ID)
			return
		}
		troops := parseGroups(order.Troops)
		own := h.isOwnCity(uid, int64(target.ID))
		desc := fmt.Sprintf("增援部队已抵达%s, 协助防守。\n", target.Name)
		if own {
			for _, g := range troops {
				if g.Count <= 0 {
					continue
				}
				h.addTroop(target.ID, g.TroopId, g.Count)
				desc += ezfyCfg.troopName(g.TroopId, 0) + "×" + strconv.FormatInt(g.Count, 10) + " "
			}
		} else {
			for _, g := range troops {
				if g.Count <= 0 {
					continue
				}
				desc += ezfyCfg.troopName(g.TroopId, 0) + "×" + strconv.FormatInt(g.Count, 10) + " "
			}
			desc += "\n部队已编入驻防队列(不并入该城兵力): 敌军进攻时先攻击先到达的驻军, 驻军全部被消灭后才攻击我方部队。"
		}
		if own {
			// ★ 2026-10-02 增援自己城市: 兵力/军官已并入目标城, 订单已完结 → 删除,
			//   避免 status=3 僵尸订单被军情误判为驻军、占用驻军槽位。
			h.DB.Delete(&model.EzfyOrder{}, order.ID)
		} else {
			order.Status = 3
			h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).Update("status", 3)
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
				"food":  ezfyResAddExpr("food", res["food"]),
				"steel": ezfyResAddExpr("steel", res["steel"]),
				"oil":   ezfyResAddExpr("oil", res["oil"]),
				"rare":  ezfyResAddExpr("rare", res["rare"]),
				"gold":  ezfyResAddExpr("gold", res["gold"]),
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
	// 攻击加成：军训艺术(5)+2%/级 · 武器科技(6)+3%/级 · 重工技术(9)+2%/级
	// ★ 2026-10-06 弹道学(8) 改为**射程加成**（用户要求：弹道学=射程，不参与攻击加成）
	atkBonus := officerBonus + atkTech[5]*2 + atkTech[6]*3 + atkTech[9]*2
	// 速度加成：燃烧引擎(10)+2%/级 · 喷气引擎(19)+3%/级
	atkSpeedBonus := atkTech[10]*2 + atkTech[19]*3
	// ★ 2026-10-06 攻方射程加成：弹道学(8)+3%/级（射程 = 基础射程 × (1+科技加成)，用户要求）
	atkRangeBonus := atkTech[8] * 3
	// ★★ 2026-09-28 修复：这里原来是**两段一模一样的 if**，军官「移速」技能被加了两次 +10
	//   （活动目标那条路径 ezfy_activity_target.go 只加一次）。
	//   后果：带移速技能的军官出征，速度加成虚高 10%，与活动战、与界面描述都不一致。
	// ★ 2026-10-06 技能随军官等级自动升级：速度技能加成也随等级 ×N
	atkSpeedBonus += h.officerSpeedSkillBonus(leadOfficer)
	atkOfficerDesc := h.officerBattleDesc(leadOfficer, h.officerBaseBonus(leadOfficer), "攻击加成")
	// ★ 2026-10-06 战报拆解逐项明细：攻方科技/技能逐项（科技名见 ezfy_cfg 种子表）
	atkTechs := ezfyBonusItems(
		ezfyTechItem("军训艺术", atkTech[5]*2),
		ezfyTechItem("武器科技", atkTech[6]*3),
		ezfyTechItem("重工技术", atkTech[9]*2),
	)
	atkSkillBreak := h.officerSkillsBreak(leadOfficer)
	// ★ 装备六项战斗加成（伤害/防御/生命/移动距离/暴击几率/暴击伤害）
	atkEquip := h.officerBattleEquipBonus(leadOfficer)
	// ★ 2026-10-07 攻方「防御加成」：出征军官属性(学识)+防御技能(弧形防御/弹幕支援)+装备 Def。
	//   原引擎攻方被打时防御恒 0 —— 军官带弧形防御 Lv.5「防御力+150%」既不生效也不展示（用户反馈）。
	//   与守方城守口径完全对称；无军官 → 0/nil → 攻方无防御加成（回退老行为）。
	atkDefBonus := 0
	var atkDefBreak []ezfyBonusItem
	if leadOfficer != nil {
		attr := h.officerGuardAttrBonus(leadOfficer)
		atkDefBonus += attr
		atkDefBreak = append(atkDefBreak, ezfyBonusItem{Name: "军官·" + leadOfficer.Name, Value: attr})
		for _, s := range h.officerGuardSkillsBreak(leadOfficer) {
			atkDefBonus += s.Value
			atkDefBreak = append(atkDefBreak, ezfyBonusItem{Name: "军官技能·" + s.Name, Value: s.Value})
		}
	}
	atkDefBonus += atkEquip.Def
	atkDefBreak = append(atkDefBreak, ezfyTechItem("装备", atkEquip.Def)...)
	// 城守(仅玩家城市防守方)
	var cityGuard *model.EzfyOfficer
	defEquip := ezfyBattleBonus{}
	defOfficerDesc := ""

	defender := []ezfyUnitGroup{}
	var defExclude map[int]bool // 城市「不参与防御」兵种集合（case 3 里填充，驻军战块后构建友军守军时用）
	targetName := ""
	var lootFood, lootSteel, lootOil, lootRare, lootGold int64
	win := false
	defBonus := 0
	defSpeedBonus := 0
	// ★ 2026-10-06 守方攻击/射程加成（城防/守城部队行动时用，默认 0；玩家城分支里填）
	defAtkBonus := 0
	// ★ 2026-10-06 守方攻击加成中「军官」占的百分点（野地守将/城守），战报日志拆解展示用
	defOfficerAtkBonus := 0
	// ★ 2026-10-06 守方拆解逐项明细（城守技能/守方科技；野地无科技 → nil，case 3 里填）
	var defSkillBreak []ezfyBonusItem
	var defTechBreak []ezfyBonusItem
	// ★ 2026-10-06 守方「防御加成」逐项明细（城墙/科技/军官属性/军官技能/装备，Name 带前缀；被攻击行展示用）
	var defDefBreak []ezfyBonusItem
	defRangeBonus := 0
	wildLevel := 0
	wildDefCamp := 0 // 野地守军阵营: 1盟军(野地) 2轴心国(寇城), 0无
	var target *model.EzfyCity
	// ★ 2026-10-06 野地/寇城守将（军官池 EzfyCfgGeneral）提升到外层作用域，
	//   战斗引擎的「守方绝地反击」用它传参（原来局部在 cfg.OfficerId 块内、从未生效）
	var defGeneral *model.EzfyCfgGeneral

	// ★ case 0：老数据/异常请求可能没带 target_type，按「野地」处理，
	//   否则会落进 default，导致战报标题变成「侦查报告: 」（目标名为空）。
	switch order.TargetType {
	case 0, 1, 2:
		// 活动目标: 侦查时按活动守军回报情报(不走普通野地配置表)
		// ★ 2026-10-05 名将野地按玩家判定：已抓到守将的玩家 → 该坐标对其是普通野地，
		//   这里与 processArrive 活动结算路由同口径，跳过活动守军走普通野地配置。
		if act := h.ezfyActTargetType(order.TargetX, order.TargetY); act > 0 &&
			!h.playerOwnsActWildGeneral(uid, order.TargetX, order.TargetY) {
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
		} else if ezfyIsSeaWildTerrain(ezfyTerrainEx(order.TargetX, order.TargetY)) {
			// ★ 2026-10-04 与地图同口径（Ex 含覆盖表/沿海平原），避免覆盖成岛屿/沿海平原的
			//   海洋格被误当「海野」配置
			// ★ 2026-10-05 岛屿也属于海野 → 岛屿守军走海野配置
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
		// ★ 野地守将（2026-09-24 ）：军官必须来自军官池(ezfy_cfg_general)、
		//   每块野地最多 1 名，配在野地类型的 officer_id 上；守将的学识给守军提供防御加成。
		wildDefCamp = 1 // 野地守军按盟军兵种名展示
		if cfgType == 3 {
			wildDefCamp = 2 // 寇城守军按轴心国兵种名展示
		}
		if cfg.OfficerId > 0 {
			defGeneral = ezfyCfg.general(cfg.OfficerId)
			if defGeneral != nil {
				guardAttr := ezfyAttrToBonus(defGeneral.Learning)
				defBonus += guardAttr
				// ★ 2026-10-06 守将防御类技能（弧形防御/弹幕支援）也计入守军防御加成
				//   （与活动守军 ezfyActWildDefBonus 同口径；原来野地守将只有属性加成，
				//   带防御技能的守将加成完全看不出来 → 明细也不会拆出「军官技能·」段）
				defBonus += generalSkillDefBonus(defGeneral)
				// ★ 2026-10-06 守将加成同时作用于守军攻击（统一加成口径）
				defAtkBonus += guardAttr
				defOfficerAtkBonus = guardAttr
				// ★ 2026-10-06 守方「防御加成」逐项明细（被打时展示：属性+技能，与 defBonus 构成同口径）
				defDefBreak = append(defDefBreak, ezfyBonusItem{Name: "军官·" + defGeneral.Name, Value: guardAttr})
				for _, s := range generalSkillDefBreak(defGeneral) {
					defDefBreak = append(defDefBreak, ezfyBonusItem{Name: "军官技能·" + s.Name, Value: s.Value})
				}
				defOfficerDesc = defGeneral.Name + " Lv." + strconv.Itoa(defGeneral.Level) + " 守军防御+" + strconv.Itoa(guardAttr) + "%"
			}
		}
		// ★ 战报里的野地要标出**具体地形类型**（丘陵/沼泽/平原…），
		//   原来一律写「野地N级」，看不出打的是什么地形。
		// ★ 2026-10-04 与地图同口径改用 Ex（覆盖表/沿海平原），修复「地图沿海平原、战报平原」不一致
		// ★ 2026-10-05 统一走 ezfyWildTerrainDisplayName：海洋野地→海底森林、岛屿→岛屿、纯海洋→海洋。
		//   能走到这里说明该等级匹配到了野地配置（cfg==nil 早已 return），所以 knownWild=true。
		name := ezfyWildTerrainDisplayName(order.TargetX, order.TargetY, true)
		if order.TargetType == 2 {
			name = "寇城"
		}
		targetName = name + strconv.Itoa(level) + "级"
		rnd := cfg.ResMin + rand.Int63n(cfg.ResMax-cfg.ResMin+1)
		lootTech := atkTech[17] * 2
		if h.officerHasSkill(leadOfficer, "黄金眼") {
			// ★ 2026-10-06 技能随军官等级自动升级：掠夺加成也随等级 ×N
			lootTech += 10 * h.officerSkillScale(leadOfficer)
		}
		rnd = rnd * int64(100+lootTech) / 100
		// ★ 2026-09-25 用户反馈「野地打完获得的资源太少」→ 管理端「二战系统配置 → 野地获取资源倍率」。
		//   放在**所有既有加成之后**做最后一道放大：掠夺技巧 / 黄金眼照旧生效，倍率再乘上去。
		//   只影响这一处（野地/海野/寇城的战斗战利品），不含驻守采集。
		rnd = ezfyScaleByWildResMult(rnd)
		lootFood, lootSteel, lootOil, lootRare, lootGold = rnd, rnd, rnd, rnd, rnd
		// ★ 2026-10-04 用户规则：掠夺不拿黄金，只有征服才能获得黄金（野地/寇城同样适用）
		if order.OrderType == 2 {
			lootGold = 0
		}
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
		//   （友军自身守军在这里构建；盟友驻军队列在下方「驻军串行战」先打完）
		defExclude = h.defExcludeSet(target.ID)
		defTech := h.techMap(target.ID)
		// 守方防御加成：城墙(建筑7) + 装甲科技(7)+3%/级 + 掩体防御(16)+2%/级
		// ★ 2026-09-28 补上重工技术(9)+2%/级：该科技描述是「重装备**攻防**+2%」，
		//   攻方那条路径已加(atkBonus)，守方这条原来漏了 → 被攻击时这 2%/级 完全不生效。
		defBonus = h.buildingLevel(target.ID, 7)*5 + defTech[7]*3 + defTech[16]*2 + defTech[9]*2
		defSpeedBonus = defTech[10]*2 + defTech[19]*3
		// 城守: 守城防御 +10% 及 防御/掩体/生命/鼓舞技能
		cityGuard = h.positionOfficer(target.ID, ezfyPositionGuard)
		defBonus += h.officerGuardBonus(cityGuard)
		// ★ 2026-10-06 守方攻击/射程加成（用户要求「科技加成、技能加成算入伤害」+「射程=基础×科技」）：
		//   守方攻击加成与守方防御同科技口径（装甲科技7/重工技术9/掩体防御16）+ 城守军官攻击技能；
		//   守方射程加成 = 弹道学(8)*3 + 掩体防御(16)*2
		defAtkBonus = defTech[7]*3 + defTech[9]*2 + defTech[16]*2 + h.officerBattleBonus(cityGuard)
		// ★ 2026-10-06 城守军官占守方攻击加成的百分点（战报日志拆解用）
		defOfficerAtkBonus = h.officerBattleBonus(cityGuard)
		// ★ 2026-10-06 守方拆解逐项明细：城守技能 + 守方科技（与 defAtkBonus 同口径）
		defSkillBreak = h.officerSkillsBreak(cityGuard)
		defTechBreak = ezfyBonusItems(
			ezfyTechItem("装甲科技", defTech[7]*3),
			ezfyTechItem("重工技术", defTech[9]*2),
			ezfyTechItem("掩体防御", defTech[16]*2),
		)
		defRangeBonus = defTech[8]*3 + defTech[16]*2
		defEquip = h.officerBattleEquipBonus(cityGuard)
		// ★ 2026-10-06 守方「防御加成」逐项明细（城墙/科技/城守属性+技能/装备，
		//   被打行展示「防御加成+N%(城墙+50% 科技·装甲科技+30% …)」，与 defBonus 构成同口径）
		defDefBreak = ezfyBonusItems(
			ezfyTechItem("城墙", h.buildingLevel(target.ID, 7)*5),
			ezfyTechItem("科技·装甲科技", defTech[7]*3),
			ezfyTechItem("科技·掩体防御", defTech[16]*2),
			ezfyTechItem("科技·重工技术", defTech[9]*2),
		)
		if cityGuard != nil {
			defDefBreak = append(defDefBreak, ezfyBonusItem{Name: "军官·" + cityGuard.Name, Value: h.officerGuardAttrBonus(cityGuard)})
			for _, s := range h.officerGuardSkillsBreak(cityGuard) {
				defDefBreak = append(defDefBreak, ezfyBonusItem{Name: "军官技能·" + s.Name, Value: s.Value})
			}
		}
		defDefBreak = append(defDefBreak, ezfyTechItem("装备", defEquip.Def)...)
		// ★ 传「属性部分」的防御加成（有效学识÷2），技能由 officerBattleDesc 自己列，
		//   否则技能会被算两遍。原来这里硬编码 10，与实际生效值不符。
		defOfficerDesc = h.officerBattleDesc(cityGuard, h.officerGuardAttrBonus(cityGuard), "守军防御")
	}

	// 侦查: 不战斗只报告情报, 部队随即返航
	// 报告细节按**侦查方侦察技巧(科技12)等级**分级（见 scoutReportBody 注释）：
	//   玩家城市 → 资源/人口民心/建筑/军队/将领/科技/最后在线时间（逐级解锁）
	//   野地寇城 → 守军情况
	if order.OrderType == 1 {
		// ★ 侦查报告：返程时长同样只认 ezfyOneWayTravel
		travel := ezfyOneWayTravel(order)
		order.Status = 2
		order.Result = order.Troops
		order.ReturnTime = now + travel
		h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
			Updates(map[string]interface{}{"status": 2, "result": order.Troops, "return_time": order.ReturnTime})
		// ★ 2026-10-02 用户规则：侦查成功率按**侦察机数量**概率（一架必成不对），
		//   失败只发失败战报、不获取任何情报（部队照常返航，不损兵）。
		if !ezfyReconSucceed(order) {
			h.addReport(uid, 1, "侦查失败: "+targetName,
				fmt.Sprintf("公文报告:侦查失败\n我方一支部队对%s[%d，%d]的侦查未能成功，未获取到任何情报，部队已返航。\n",
					targetName, order.TargetX, order.TargetY), "", order.ID)
			return
		}
		h.addReport(uid, 1, "侦查报告: "+targetName,
			h.scoutReportBody(uid, order, targetName, target, defender), "", order.ID)
		return
	}

	// ★★ 指挥室（2026-09-22 ）：战斗类订单到达后**不立即结算**，
	//   先开一场战场，玩家在「军情 → 军队动态 → [指挥]」里下达前进/暂停/后退；
	//   每回合 30 秒、最多 40 回合，不下指令则按「前进」自动推进。
	//   BattleResult 非空 = 这场仗已经在指挥室里打完了 → 直接用结果走下面的常规结算，
	//   所以战报/掠夺/经验/征服这些战后逻辑全部复用，没有第二套实现。
	atkTargets := h.buildTargetMap(city.ID, true)
	defTargets := h.buildTargetMap(cityIdOf(target), false)
	atkMoves := h.buildMoveMap(city.ID, true)
	defMoves := h.buildMoveMap(cityIdOf(target), false)

	// ★★ 2026-10-02 盟军驻军串行战斗（）：
	//   目标为玩家城时，守方 = 盟友驻军队列（按到达先后，先到先被打）+ 友军自身部队（最后）。
	//   **每个驻军队列是一场独立战斗、独立战报**：打驻守A → 自动打驻守B → 全部驻军被打完
	//   才与友军自身部队进行主城战（主城战仍走指挥室）。
	//   驻军战报（攻方 + 各驻军方）都是 PvP 报告（order_id>0 且 order target_type=3）
	//   → 自动进入「军团战报」，友军、驻军方、全军团都能看到。
	//   驻军战自动结算（纯驻守无人指挥，不走指挥室）。
	garrisonStopped := false // 攻方被驻军挡住（平局或全灭）→ 主城战不进行
	if target != nil && target.UserID != uid {
		var garOrders []model.EzfyOrder
		h.DB.Where("target_id = ? AND target_type = 3 AND order_type = 6 AND status = 3", target.ID).
			Order("arrive_time ASC, id ASC").Find(&garOrders)
		for gi := range garOrders {
			go_ := &garOrders[gi]
			garTroops := parseGroups(go_.Troops)
			garTotal := int64(0)
			for _, g := range garTroops {
				garTotal += g.Count
			}
			if garTotal <= 0 {
				// 空壳驻军订单（历史送兵遗留）直接清掉
				h.DB.Delete(&model.EzfyOrder{}, go_.ID)
				continue
			}
			atkTotal := int64(0)
			for _, g := range attacker {
				atkTotal += g.Count
			}
			if atkTotal <= 0 {
				break
			}
			// ★ 2026-10-02 驻军战用「溃败撤退」：守方剩余兵力跌破阈值即判定战败、战斗提前结束，
			//   剩余部队自动返航回出发城市 —— 这样「驻军战败 → 回到自己城市」才有兵可回。
			gbr := ezfySimulateBreak(attacker, garTroops,
				atkBonus, 0, atkSpeedBonus, 0,
				atkRangeBonus, 0, 0,
				atkEquip, ezfyBattleBonus{},
				atkOfficerDesc, "",
				officerBonus, 0, // 驻军战：守方无军官；攻方军官加成照常拆解展示
				atkTargets, defTargets, atkMoves, defMoves,
				ezfyGarrisonBreakPct)
			var gcity model.EzfyCity
			gcityName := "友军"
			if err := h.DB.First(&gcity, go_.CityId).Error; err == nil {
				gcityName = gcity.Name
			}
			gDetail := ""
			for _, a := range gbr.Actions {
				gDetail += a + "\n"
			}
			// 驻军剩余兵力（gbr.DefenderLosses 全量保序，与 garTroops 对齐）
			left := map[int]int64{}
			for _, g := range garTroops {
				left[g.TroopId] += g.Count
			}
			for i, lg := range gbr.DefenderLosses {
				if lg.Count <= 0 {
					continue
				}
				if i < len(garTroops) {
					left[garTroops[i].TroopId] -= lg.Count
					if left[garTroops[i].TroopId] < 0 {
						left[garTroops[i].TroopId] = 0
					}
				}
			}
			gLeftStr := ""
			for tid, cnt := range left {
				if cnt > 0 {
					gLeftStr += strconv.Itoa(tid) + ":" + strconv.FormatInt(cnt, 10) + ","
				}
			}
			if len(gLeftStr) > 0 {
				gLeftStr = gLeftStr[:len(gLeftStr)-1]
			}
			// ★ 2026-10-02 用户规则：打平/打赢继续留守；只有战败 → 剩余部队自动返航回出发城市。
			endNote := ""
			if gbr.AttackerWin {
				if gLeftStr == "" {
					endNote = "\n驻军全军覆没, 未留下剩余部队。"
				} else {
					endNote = "\n驻军战败, 剩余部队已自动返航回出发城市。"
				}
			}
			// 驻军方战报（report_type=2 PvP → 军团战报可见）
			h.addReport(go_.UserID, 2, "驻防战报: "+targetName,
				fmt.Sprintf("你的驻军(来自%s)在%s(%d,%d)的驻防战斗已结束!\n%s\n%s%s",
					gcityName, targetName, order.TargetX, order.TargetY,
					battleOutcomeText(!gbr.AttackerWin, gbr.Draw), lossText(gbr.DefenderLosses), endNote),
				gDetail, int64(order.ID), target.ID)
			// 攻方战报（对这支驻军）
			h.addReport(uid, 2, "战斗报告: 击溃"+gcityName+"的驻军",
				fmt.Sprintf("我方部队在%s(%d,%d)击溃了来自%s的盟军驻军!\n%s",
					targetName, order.TargetX, order.TargetY, gcityName,
					lossText(gbr.DefenderLosses)),
				gDetail, int64(order.ID), target.ID)
			if gLeftStr == "" {
				// 驻军全灭（未触发溃败撤退）→ 删除队列
				h.DB.Delete(&model.EzfyOrder{}, go_.ID)
			} else if gbr.AttackerWin {
				// ★ 2026-10-02 驻军战败 → 剩余部队自动返航回出发城市
				//   （返航到达后兵力按 order.CityId 回到自己城市，finishReturn 入城）
				back := now + ezfyOneWayTravel(go_)
				h.DB.Model(&model.EzfyOrder{}).Where("id = ?", go_.ID).
					Updates(map[string]interface{}{"status": 2, "result": gLeftStr, "return_time": back})
			} else {
				// 打平 / 打赢 → 剩余部队继续留守驻守
				h.DB.Model(&model.EzfyOrder{}).Where("id = ?", go_.ID).Update("troops", gLeftStr)
			}
			// 攻方剩余兵力进入下一支驻军/主城战
			attacker = gbr.AttackerLeft
			if gbr.Draw {
				// 平局 = 这支驻军守住，攻方未能突破，不再打后面的驻军
				garrisonStopped = true
				break
			}
		}
		if !garrisonStopped {
			// 驻军全部被打完后，才轮到友军自身部队
			for tid, count := range h.troopMap(target.ID) {
				if count > 0 && !defExclude[tid] {
					defender = append(defender, ezfyUnitGroup{TroopId: tid, Count: count})
				}
			}
		}
	}
	// 攻方被驻军挡住或已无兵 → 返航，不进行主城战
	if garrisonStopped {
		resultStr := ""
		for _, g := range attacker {
			if g.Count > 0 {
				resultStr += strconv.Itoa(g.TroopId) + ":" + strconv.FormatInt(g.Count, 10) + ","
			}
		}
		if len(resultStr) > 0 {
			resultStr = resultStr[:len(resultStr)-1]
		}
		order.Result = resultStr
		order.Status = 2
		order.ReturnTime = now + ezfyOneWayTravel(order)
		h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
			Updates(map[string]interface{}{"status": 2, "result": resultStr, "return_time": order.ReturnTime})
		h.addReport(uid, 2, "战斗报告: 进攻受阻(驻军拦截)",
			fmt.Sprintf("我方部队进攻%s(%d,%d)时被盟军驻军拦截, 未能攻入城市, 部队已返航。",
				targetName, order.TargetX, order.TargetY), "", order.ID, target.ID)
		return
	}

	var br ezfyBattleResult
	if done, ok := ezfyBattleResultDecode(order.BattleResult); ok {
		br = done
	} else {
		// ★ 阵营兵种名：守方是玩家城时用守方阵营；野地=盟军、寇城=轴心国，其余 0(通用名)
		defCamp := wildDefCamp
		if target != nil {
			defCamp = h.ensureProfile(target.UserID).Camp
		}
		// ★ 守方「绝地反击」生效回合数：玩家城 = 城守军官；野地/寇城 = 野地守将（无守将 → 0）
		defCounterRounds := 0
		if target != nil {
			defCounterRounds = h.officerCounterRounds(cityGuard)
		} else {
			defCounterRounds = generalCounterRounds(defGeneral)
		}
		st := ezfyNewBattleState(attacker, defender,
			atkBonus, defBonus, defAtkBonus, atkSpeedBonus, defSpeedBonus,
			atkRangeBonus, defRangeBonus,
			atkEquip, defEquip, atkOfficerDesc, defOfficerDesc,
			officerBonus, defOfficerAtkBonus, // 军官占的「攻击加成」百分点（战报日志拆解用）
			// ★ 2026-10-06 军官加成里「技能」占的百分点（拆解单独展示「军官技能+N%」）
			h.officerSkillBattleBonus(leadOfficer), h.officerSkillBattleBonus(cityGuard),
			// ★ 2026-10-06 技能/科技逐项明细（战报展示「军官技能·尖兵突击+N%」「科技·弹道学+N%」）
			atkSkillBreak, defSkillBreak, atkTechs, defTechBreak, defDefBreak,
			// ★ 2026-10-07 攻方「防御加成」（军官属性+防御技能+装备 Def）
			atkDefBonus, atkDefBreak,
			atkTargets, defTargets, atkMoves, defMoves,
			// ★ 军官技能「绝地反击」随等级升级：生效前N回合（攻方带队/守方城守或野地守将各自判定）
			h.officerCounterRounds(leadOfficer), defCounterRounds,
			h.ensureProfile(uid).Camp, defCamp)
		// ★ 2026-09-23 目标被别的玩家抢先指挥时，本部队改为「等待」，
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
			// ★ 2026-09-23 「敌人来了没提示 / 军情警讯不及时」：
			//   敌军**到达**我方城市开战时，立即给守方发一条「军情警讯」。
			//   （「敌军来袭」预警已在 createOrder 时发；这里补「已抵达」的实时消息，
			//   不依赖雷达站 —— 结果类消息不受雷达限制。）
			// ★ 2026-09-25 「军情警讯里要看到对面城市名字和地址」→ 这里带上**坐标**，
			//   并且写进标题（列表只显示标题，不点进去也要看得见）。这条是「人已经到了」的事后
			//   消息，不受雷达站限制，所以来源一定给全 —— 保证玩家至少在这一步知道谁打了他。
			if order.TargetType == 3 && target != nil && target.UserID > 0 && target.UserID != uid {
				h.addReport(target.UserID, 6,
					fmt.Sprintf("军情警报: 敌军已抵达 来自 %s(%d,%d)", city.Name, city.X, city.Y),
					fmt.Sprintf("敌方部队已抵达我方城市「%s」(%d,%d) 附近，双方即将交战！\n来袭方城市：%s(%d,%d)\n请到「军情 → 军队动态」进入[指挥]部署守军。",
						target.Name, order.TargetX, order.TargetY, city.Name, city.X, city.Y),
					"", 0, target.ID)
			}
			// ★ 「等待指挥」不要放进战斗报告列表 —— 战斗还没结束，战报应当是**结果**。
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

	// ★ 2026-10-06 被掠夺/被征服报告正文要等战斗结算完（用与攻方同款的完整正文）才写，
	//   相关临时值提前到函数级作用域，各分支里只赋值。
	lootFeel := 0     // 掠夺民心扣减值
	defConqBody := "" // 被征服报告顶部「守方结论」段
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
	// ★ 2026-10-05 战报里的兵种名**统一用基础兵种名**（不带阵营前缀），
	//   攻方/守方都不再按阵营取 name_ally / name_axis。
	// 带队军官 / 城守军官
	if leadOfficer != nil {
		report += "军官:" + officerReportDesc(leadOfficer) + "\n"
	}
	if cityGuard != nil {
		report += "守军军官:" + officerReportDesc(cityGuard) + "\n"
	}

	atkBefore := groupCounts(attacker)
	atkAfter := groupCounts(br.AttackerLeft)
	// ★ 2026-09-24 40 回合平局时双方标签都显示 [平] 而不是胜/败
	atkTag, defTag := winText(win), winText(!win)
	if draw {
		atkTag, defTag = "平", "平"
	}
	report += fmt.Sprintf("[%s]攻方:%s\n", atkTag, city.Name)
	report += troopChangeText(atkBefore, atkAfter)
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
	report += troopChangeText(defBefore, defAfter)

	detail := ""
	for _, a := range br.Actions {
		detail += a + "\n"
	}
	detail += "\n[双方兵力]\n"
	detail += fmt.Sprintf("[%s]攻方:%s\n", atkTag, city.Name)
	detail += troopChangeText(atkBefore, atkAfter)
	detail += fmt.Sprintf("--------------------\n[%s]守方:%s\n", defTag, targetName)
	detail += troopChangeText(defBefore, defAfter)
	detail += "[双方兵力]"

	// 攻方战损: 按修复率入伤兵营
	losses := br.AttackerLosses
	var deadCount int64
	for _, g := range losses {
		deadCount += g.Count
	}
	healTech := atkTech[21] * 2
	// 带队军官「机械改造」技能: 战后伤兵恢复 +10%
	if h.officerHasSkill(leadOfficer, "机械改造") {
		// ★ 2026-10-06 技能随军官等级自动升级：恢复加成也随等级 ×N
		healTech += 10 * h.officerSkillScale(leadOfficer)
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
	// ★ 2026-09-28 同样统一走 ezfyCarryCapOf（含「装载技术」加成），
	//   否则掠夺时「能搬走多少」会比部队真实负重少，玩家看到战利品被无理由截断。
	carry := h.ezfyCarryCapOf(left, city.ID)

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
		// ★★ 2026-09-27 免战保护令**绝对生效**（宣战也不能打）。
		//   目标城市处于免战保护期时本次战斗不结算：无战利品、不扣民心、武将不被俘，
		//   部队到达后直接返航，战报提示「该玩家使用免战道具, 无法结算」。
		if targetProtected && order.TargetType == 3 {
			report += "\n该玩家使用了免战道具, 无法结算!"
			travel := ezfyOneWayTravel(order)
			order.Status = 2
			order.ReturnTime = now + travel
			report += h.battleStatsTail(uid, 0, 0)
			repTitle := "出征报告: " + targetName
			if order.OrderType == 2 {
				repTitle = "掠夺报告: " + targetName
			} else if order.OrderType == 3 {
				repTitle = "征服报告: " + targetName
			}
			h.addReport(uid, 3, repTitle, report, detail, order.ID)
			h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
				Updates(map[string]interface{}{"status": order.Status, "result": order.Result, "return_time": order.ReturnTime})
			return
		}
		// ★ 2026-09-25 「军团交战期掠夺/征服获胜可获得军团战绩积分（军团总积分 + 成员个人积分）」：
		//   只在「掠夺(2)/征服(3) 攻打玩家城市 且 攻击方获胜」这一处发放，**只加一次**。
		//   helper 内部自己判断是否处于生效中的军团交战期（不处于则什么都不做），
		//   所以这里不需要再做军团判断。
		if (order.OrderType == 2 || order.OrderType == 3) && order.TargetType == 3 &&
			target != nil && target.UserID != 0 && target.UserID != uid {
			h.ezfyCorpsWarAward(uid, target.UserID, order.OrderType)
		}
		if targetProtected && order.TargetType == 3 {
			report += "\n目标城市处于免战保护期, 无法掠夺资源!"
		}
		if order.TargetType == 3 && target != nil {
			// 玩家城市: 掠夺比例10%+掠夺技巧, 上限50%
			lootRate := 10 + atkTech[17]*2
			if h.officerHasSkill(leadOfficer, "黄金眼") {
				// ★ 2026-10-06 技能随军官等级自动升级：掠夺率也随等级 ×N
				lootRate += 10 * h.officerSkillScale(leadOfficer)
			}
			// 免战保护（含宣战）期间掠夺量为 0；早退分支已统一 return，此处为防御保留
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
				// ★ 2026-10-04 用户规则：掠夺不拿黄金（只有征服才拿）。
				//   黄金份额置 0 后再做负重缩放，腾出的负重让给其它四资源。
				if order.OrderType == 2 && i == 4 {
					loot[i] = 0
				}
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
			// ★ 2026-10-04 负重上限加固：部队全灭(负重 0)时战利品一根也带不回去。
			//   原来 `totalLoot > carry && carry > 0` 在 carry==0 时直接跳过缩放 → 全灭白拿全仓。
			if carry <= 0 {
				loot[0], loot[1], loot[2], loot[3], loot[4] = 0, 0, 0, 0, 0
			} else if totalLoot > carry {
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
				// ★ 2026-09-25 「攻击野地获得的资源也要累加」→ 战利品入账统一走
				//   ezfyResAddExpr（DB 原子累加 + 资源最大值），不再「内存加完整行写回」：
				//   原写法在并发下会被别的请求覆盖掉，且会连带写回其它陈旧字段。
				h.addResToCityDB(city.ID, lootFood, lootSteel, lootOil, lootRare, lootGold)
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
			// ★ 2026-10-04 与地图同口径（Ex 含覆盖表/沿海平原）
			// ★ 2026-10-05 岛屿也属于海野 → 占领后记成海野（采集走海野系数）
			if order.TargetType == 2 || ezfyIsSeaWildTerrain(ezfyTerrainEx(order.TargetX, order.TargetY)) {
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
				// ★ 2026-10-05 战报兵种名统一用基础兵种名（不带阵营前缀）
				report += fmt.Sprintf("\n俘获: %s×%d", ezfyCfg.troopName(capturedTroopId, 0), capturedCount)
			}
			// ★ 2026-10-05 野地类型「商城道具掉落」（管理端在野地类型里配，默认空=不掉）：
			//   打赢该类型野地/海野/寇城后按 [[cfg_id,数量,概率%],...] 掉落商城道具到背包，
			//   每条独立按概率判定（概率缺省 = 100%）。
			//   ⚠️ 这里在 switch 之外，case 里的 cfg 不可见 → 按同一口径重新取配置。
			wcType := 1
			if order.TargetType == 2 {
				wcType = 3
			} else if ezfyIsSeaWildTerrain(ezfyTerrainEx(order.TargetX, order.TargetY)) {
				wcType = 2
			}
			if wcfg := ezfyCfg.wildland(wcType, wildLevel); wcfg != nil && strings.TrimSpace(wcfg.DropItems) != "" {
				for _, d := range parseWildlandItemDrops(wcfg.DropItems) {
					if rand.Intn(100) >= d[2] {
						continue // 未命中概率，不掉
					}
					if it := ezfyCfg.item(d[0]); it != nil {
						h.addItem(uid, d[0], d[1])
						report += fmt.Sprintf("\n掉落道具: %s×%d", it.Name, d[1])
					}
				}
			}
			// ★ 2026-10-05 宝物掉落（下拉配置 + 概率）：[{"name","count","pct"}]，老文本值解析失败则不掉
			if wcfg := ezfyCfg.wildland(wcType, wildLevel); wcfg != nil && strings.TrimSpace(wcfg.Treasure) != "" {
				var drops []wildTreasureDrop
				if err := json.Unmarshal([]byte(wcfg.Treasure), &drops); err == nil {
					for _, d := range drops {
						if d.Count <= 0 || strings.TrimSpace(d.Name) == "" {
							continue
						}
						pct := d.Pct
						if pct <= 0 {
							pct = 100
						}
						if pct > 100 {
							pct = 100
						}
						if rand.Intn(100) >= pct {
							continue // 未命中概率，不掉
						}
						if eq := ezfyCfg.equipmentByName(d.Name); eq != nil {
							for k := 0; k < d.Count; k++ {
								h.addEquipment(city, eq)
							}
							report += fmt.Sprintf("\n掉落宝物: %s×%d", eq.Name, d.Count)
						}
					}
				}
			}
		}

		// 征服玩家城市
		// ★ 2026-10-06 是否**新建成**占领记录（城池没了 → 全城军官都掉忠诚）；
		//   声明在征服块外，结束区（掠夺/征服共用的军官忠诚结算）也要读它
		newOccupy := false
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
				// ★ 2026-09-25：战利品入账改为 DB 原子累加 + 资源最大值（同下）
				h.addResToCityDB(city.ID, lootFood, lootSteel, lootOil, lootRare, lootGold)
				report += h.battleStatsTail(uid, 0, recyclePct)
				// ★ 2026-10-06 单次征服攻打（城没占下来，城池还在）→ 只扣城守忠诚
				if frag := h.defectDefenderOfficers(city, target, uid, false); frag != "" {
					report += frag
				}
				h.addReport(uid, 3, "征服报告: "+targetName, report, detail, order.ID)
				h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
					Updates(map[string]interface{}{"status": order.Status, "result": order.Result, "return_time": order.ReturnTime})
				return
			}
			// 免战保护绝对生效（含宣战）：民心已失也无法征服；早退分支已统一 return，此处为防御保留
			if targetProtected {
				report += "\n民心已失，但目标处于免战保护期，无法征服!"
				target.Feelings = cur
				h.DB.Model(&model.EzfyCity{}).Where("id = ?", target.ID).Update("feelings", target.Feelings)
				travel := ezfyOneWayTravel(order)
				order.Status = 2
				order.ReturnTime = now + travel
				report += h.battleStatsTail(uid, 0, recyclePct)
				// ★ 2026-10-06 免战拦截但战斗已打（城池仍在）→ 只扣城守忠诚
				if frag := h.defectDefenderOfficers(city, target, uid, false); frag != "" {
					report += frag
				}
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
			// ★★ 2026-09-27 「仅剩一城不可被占领」→ 后又改口：**允许占光**，
			//   守方自由城被占光后由系统补给一座随机新城市（保证玩家永远有城）。
			//   征服结算既能由攻方轮询(processOrders)触发、也能由守方轮询(processIncoming)触发，
			//   多个进攻方可能同时对同一守方做「统计现城数→建占领记录」的读-改-写；
			//   不加锁会集体判定通过、重复建占领记录。这里按守方 uid 分片加锁，
			//   锁内重查「未被占领的自由城数」（排除已有 status=1 占领记录的城），
			//   并单独防「重复攻打同一座已占城」。
			lock := ezfyOccupyLock(target.UserID)
			lock.Lock()
			// 该城是否已被占走（防重复攻打同一座城）
			var dupOccupy int64
			h.DB.Model(&model.EzfyOccupy{}).Where("city_id = ? AND status = 1", target.ID).Count(&dupOccupy)
			// 守方「未被占领的自由城数」：user_id 是他的城 且 没有 status=1(占走未处理) 的占领记录
			var freeCount int64
			h.DB.Model(&model.EzfyCity{}).
				Where("user_id = ? AND id NOT IN (SELECT city_id FROM ezfy_occupy WHERE status = 1)", target.UserID).
				Count(&freeCount)
			// 补给标记：占掉这座后守方自由城清零 → 解锁后补一座新城
			needReplenish := false
			lastCity := false
			// ★ 2026-10-06 本场是否**新建成**占领记录（城池没了 → 全城军官都掉忠诚）。
			//   newOccupy 声明在征服块外（见上），此处只赋值，避免遮蔽外层变量。
			switch {
			case dupOccupy > 0:
				// 这座城已被先到的队伍占走，本次不能重复占
				lastCity = true
				report += "\n该城市已被其他部队占领, 无法重复占领!"
			default:
				occ := model.EzfyOccupy{CityId: int64(target.ID), CityName: targetName,
					AtkUserId: uid, AtkCityId: int64(city.ID), DefUserId: target.UserID,
					X: target.X, Y: target.Y, Status: 1}
				h.DB.Create(&occ)
				newOccupy = true
				report += "\n占领成功! 城市已归入你的附属, 可在[附属野地]中摧毁/归还"
				// ★ 2026-10-02 盟军驻军：城市被占领 → 该城盟军驻军全部失效（残余一并清除），
				//   并给各驻军方发战报（report_type=4 PvP → 军团战报可见）
				var gars []model.EzfyOrder
				h.DB.Where("target_id = ? AND target_type = 3 AND order_type = 6 AND status = 3", target.ID).Find(&gars)
				for _, go_ := range gars {
					var gc model.EzfyCity
					gcName := "友军城市"
					if err := h.DB.First(&gc, go_.CityId).Error; err == nil {
						gcName = gc.Name
					}
					h.addReport(go_.UserID, 4, "驻防战报: 驻防城市失守",
						fmt.Sprintf("你驻守的%s(%d,%d)已被敌军占领!\n你的驻军(来自%s)已全部损失。",
							targetName, order.TargetX, order.TargetY, gcName), "", int64(order.ID), target.ID)
				}
				if len(gars) > 0 {
					h.DB.Where("target_id = ? AND target_type = 3 AND order_type = 6 AND status = 3", target.ID).
						Delete(&model.EzfyOrder{})
				}
				if freeCount <= 1 {
					// 占掉这座后守方已无自由城 → 系统补给
					needReplenish = true
				}
			}
			lock.Unlock()
			// 补给在解锁后执行（不占用守方锁；建城自带 findFreePos 找空位）
			if needReplenish {
				newCity := h.replenishCity(target.UserID)
				// ★ 2026-10-07 用户反馈「被打飞后战报不该告诉攻击者新坐标」：
				//   攻方战报不再暴露补给新城市的坐标，避免被追打；守方自己的补偿报告(L3475)保留坐标。
				report += fmt.Sprintf("\n守方城市已全部被占, 系统已补给新城市[%s]", newCity.Name)
				h.addReport(target.UserID, 5, "系统补偿新城市",
					fmt.Sprintf("你的全部城市已被敌方占领!\n系统已补偿一座新城市[%s](%d,%d), 请重新发展。", newCity.Name, newCity.X, newCity.Y), "", 0, newCity.ID)
			}
			h.saveCityRes(target)
			h.DB.Model(&model.EzfyCity{}).Where("id = ?", target.ID).
				Updates(map[string]interface{}{"feelings": 0, "grievance": target.Grievance})
			// ★ 2026-10-06 「被征服报告」：顶部保留守方结论，完整战斗正文（与攻方
			//   「征服报告」同款：主题/时间/公文报告/战斗经过/双方兵力/战果/逐回合详情）
			//   在结算完统一拼写（见函数末尾），标题带守方城名+坐标
			//   （原误用攻方 city.Name 是 bug）。
			if !lastCity {
				defConqBody = fmt.Sprintf("你的城市%s已被敌方部队占领!\n民心清零!\n被掠夺资源: 粮%d 钢%d 油%d 稀矿%d 金%d",
					targetName, lootFood, lootSteel, lootOil, lootRare, lootGold)
			} else {
				// 该城已被先到的队伍占走（本次重复攻打）
				defConqBody = fmt.Sprintf("敌方部队再次攻打你的城市%s!\n民心清零, 但该城已被其他部队占领, 无法重复占领!\n被掠夺资源: 粮%d 钢%d 油%d 稀矿%d 金%d",
					targetName, lootFood, lootSteel, lootOil, lootRare, lootGold)
			}
		}
		// 普通掠夺(含成功掠夺玩家城市): 民心-N 民怨+N
		// ★ 用户反馈「民心每次 -5 现在太多」→ 扣多少改为管理端可配
		//   （ezfy_cfg_limit.loot_feelings，默认 2）。
		if order.OrderType == 2 && order.TargetType == 3 && target != nil {
			lootFeel = ezfyLootFeelingsCfg()
			target.Feelings = maxInt(0, target.Feelings-lootFeel)
			target.Grievance = minInt(100, target.Grievance+lootFeel)
			h.saveCityRes(target)
			h.DB.Model(&model.EzfyCity{}).Where("id = ?", target.ID).
				Updates(map[string]interface{}{"feelings": target.Feelings, "grievance": target.Grievance})
			// ★ 2026-10-06 「被掠夺报告」正文延后到战斗结算完统一写（见函数末尾），
			//   与攻方「掠夺报告」同款完整格式；这里只扣民心/民怨。
		}
		// 掠夺资源入账
		if order.OrderType == 2 || order.OrderType == 3 {
			// ★ 2026-09-25 「攻击野地获得的资源也要累加」→ 走 DB 原子累加 + 资源最大值：
			//   原来在内存里 += 再 saveCityRes（整行写回），既可能覆盖并发写入的增量，
			//   也把「无上限累加」这个规则散落在多处；现在统一到 addResToCityDB。
			//   野地/海野/寇城的战利品就是从这里入账的（order 2/3 打 target_type 1/2）。
			h.addResToCityDB(city.ID, lootFood, lootSteel, lootOil, lootRare, lootGold)
		}
		// 攻打玩家城市: 目标城军官忠诚下降, 归零者弃城成为我方战俘
		// (复刻用户说明的 PvP 战俘来源: 把对方军官忠诚打成 0)
		// ★ 2026-10-06 用户规则修正：
		//   掠夺成功(城还在)          → 只扣城守忠诚
		//   征服成功且**新建占领记录**（城池没了）→ 全城军官都扣忠诚，归零者成俘
		//   重复攻打已被人占走的城（lastCity）→ 该城军官在首次被占时已处理过，不再重复扣
		if order.TargetType == 3 && target != nil {
			if order.OrderType == 2 {
				if frag := h.defectDefenderOfficers(city, target, uid, false); frag != "" {
					report += frag
				}
			} else if order.OrderType == 3 && newOccupy {
				if frag := h.defectDefenderOfficers(city, target, uid, true); frag != "" {
					report += frag
				}
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
			} else if ezfyIsSeaWildTerrain(ezfyTerrainEx(order.TargetX, order.TargetY)) {
				// ★ 2026-10-05 岛屿也属于海野 → 岛屿守将俘虏按海野配置
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
		// ★ 2026-10-06 守方被动战报（被掠夺/被征服）：**守方视角**标题 = 守方城名+守方坐标
		//   （targetName = target.Name 即被掠夺/被征服的那座守城），
		//   如 [掠夺] 被掠夺报告: 我的城名(X,Y)；与攻方报告(L3490)保持对称口径。
		if order.TargetType == 3 && target != nil {
			if order.OrderType == 2 {
				h.addReport(target.UserID, 2, "被掠夺报告: "+targetName+"("+strconv.Itoa(order.TargetX)+","+strconv.Itoa(order.TargetY)+")",
					fmt.Sprintf("你的城市%s被敌方部队掠夺!\n民心-%d 民怨+%d\n\n%s", targetName, lootFeel, lootFeel, report),
					detail, 0, target.ID)
			} else if order.OrderType == 3 && defConqBody != "" {
				h.addReport(target.UserID, 4, "被征服报告: "+targetName+"("+strconv.Itoa(order.TargetX)+","+strconv.Itoa(order.TargetY)+")",
					defConqBody+"\n\n"+report, detail, 0, target.ID)
			}
		}
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
			// ★ 2026-10-06 守方成功守住也按进攻意图归「战斗报告」：
			//   掠夺→被掠夺报告 / 征服→被征服报告，军情警讯不再出现防守报告（只留预警）。
			// ★ 2026-10-06 正文升级为与攻方报告同款的完整公文模板：
			//   主题/出发地/目的地/时间/公文报告/一段式描述/军衔声望/军官/攻守兵力块/
			//   军功声望/军官经验/战果/个人荣誉等（守方视角：敌方进攻、我方守住）。
			defTitle, defType := "被掠夺报告", 2
			ocTitle := "掠夺"
			if order.OrderType == 3 {
				defTitle, defType = "被征服报告", 4
				ocTitle = "征服"
			}
			defProfile := h.ensureProfile(target.UserID)
			defReport := fmt.Sprintf("主题:%s\n出发地:%s(%d,%d)\n目的地:%s(%d,%d)\n时间:%s\n公文报告:%s\n敌方一支部队对 %s[ %d，%d ]进行了%s。战斗共持续 %d 回合，我方战斗胜利！\n",
				defTitle, city.Name, city.X, city.Y, targetName, order.TargetX, order.TargetY,
				time.UnixMilli(now).Format("2006-01-02 15:04"), defTitle,
				targetName, order.TargetX, order.TargetY, ocTitle, br.Rounds)
			defReport += fmt.Sprintf("军衔声望:%d\n", defProfile.Prestige)
			if cityGuard != nil {
				defReport += "军官:" + officerReportDesc(cityGuard) + "\n"
			}
			// 攻守兵力块复用上面的标签/兵力口径（win=false：攻方落败、守方获胜）
			defReport += fmt.Sprintf("[%s]攻方:%s\n", atkTag, city.Name)
			defReport += troopChangeText(atkBefore, atkAfter)
			defReport += fmt.Sprintf("--------------------\n[%s]守方:%s\n", defTag, targetName)
			defReport += troopChangeText(defBefore, defAfter)
			defReport += "\n军功声望+100"
			if defExp > 0 {
				defReport += fmt.Sprintf("\n军官经验+%d", defExp)
			}
			defReport += "\n战果\n黄金:0\n粮食:0\n钢铁:0\n石油:0\n稀矿:0"
			defReport += h.battleStatsTail(target.UserID, 100, 0)
			h.addReport(target.UserID, defType, defTitle+": "+targetName+
				"("+strconv.Itoa(order.TargetX)+","+strconv.Itoa(order.TargetY)+")",
				defReport, detail, 0, target.ID)
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
// ★ （2026-09-23）：被攻击时，防御战斗的兵种列表**不包含**标记了「不参与防御」的兵种。
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

// wildTreasureDrop 野地类型「宝物掉落」单条配置（管理端下拉编辑器生成的结构化 JSON）
type wildTreasureDrop struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
	Pct   int    `json:"pct"` // 掉落概率 %（缺省/<=0 = 100）
}

// parseWildlandItemDrops 解析野地类型「商城道具掉落」配置：[[道具cfg_id,数量,概率%],...]
//
// ★ 2026-10-05 新增概率：第 3 位 = 单次掉落概率%（1~100）；缺省（只有 2 位）= 100% 必掉。
func parseWildlandItemDrops(raw string) [][3]int {
	var rows [][]int
	if err := json.Unmarshal([]byte(raw), &rows); err != nil {
		return nil
	}
	out := [][3]int{}
	for _, r := range rows {
		if len(r) < 2 || r[0] <= 0 || r[1] <= 0 {
			continue
		}
		pct := 100
		if len(r) >= 3 && r[2] > 0 {
			pct = r[2]
			if pct > 100 {
				pct = 100
			}
		}
		out = append(out, [3]int{r[0], r[1], pct})
	}
	return out
}

// parseWildlandTroops 解析 [[兵种id,最小,最大],...] 生成守军(随机数量)
//
// ★ 「加个野地兵力倍数配置，默认 1，可以调整倍数」→ 随机出来的数量再乘倍数。
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

// troopChangeText 兵力变化文本。
//
// ★ 2026-10-05 战报里兵种名**统一展示基础兵种名**（不再按阵营显示
//
//	阵营兵种名 name_ally / name_axis），损失展示成 `兵种名 战前->战后(-损失)`，
//	例如 `驱逐舰 16833->0(-16833)`。
func troopChangeText(before, after map[int]int64) string {
	ids := []int{}
	for tid := range before {
		ids = append(ids, tid)
	}
	// 按兵种 id 排序, 避免 Go map 随机遍历导致战报里兵种顺序每次都变
	sort.Ints(ids)
	text := ""
	for _, tid := range ids {
		// camp 传 0 → troopName 返回基础兵种名（不带阵营前缀）
		name := ezfyCfg.troopName(tid, 0)
		if name == "" {
			name = "兵种" + strconv.Itoa(tid)
		}
		b := before[tid]
		a := after[tid]
		if a > b {
			a = b
		}
		if b-a > 0 || b > 0 {
			text += fmt.Sprintf("%s %d->%d(-%d)\n", name, b, a, b-a)
		}
	}
	return text
}

// lossText 守军损失文本。
//
// ★ 2026-10-05 兵种名**统一展示基础兵种名**（不按阵营）。
func lossText(groups []ezfyUnitGroup) string {
	if len(groups) == 0 {
		return "守军无损失"
	}
	text := "守军损失: "
	for _, g := range groups {
		if g.Count <= 0 {
			continue // ★ 全量保序的守方损失里含 count=0 的占位项，不能打印「×0」
		}
		// camp 传 0 → troopName 返回基础兵种名（不带阵营前缀）
		name := ezfyCfg.troopName(g.TroopId, 0)
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

// battleOutcomeText 战报结局文案（★ 2026-09-24 40 回合未分胜负显示平局而非失败）
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

// ezfyReconPlaneCount 本次侦查携带的侦察机数量（决定侦查成功率）。
func ezfyReconPlaneCount(order *model.EzfyOrder) int64 {
	var n int64
	for _, g := range parseGroups(order.Troops) {
		if g.TroopId == 9 { // 侦察机（createOrder 已校验侦查只能带侦察机）
			n += g.Count
		}
	}
	return n
}

// ezfyReconSucceed 侦查是否成功：按侦察机数量概率判定。
//
// ★ 2026-10-02 用户规则：侦查成功率按**侦察机数量**平滑上升，梯度要陡，按数量级拉开
//
//	（原 1 架 20%、10 架 89% 的累乘曲线太浅，改成数量级曲线）。曲线为 logistic：
//	  成功率 = cap × 1/(1 + e^-k·(log10(n) - x0))，n = 携带侦察机数。
//	  n=10≈11%  n=100≈26%  n=1千≈48%  n=1万≈69%  n=5万≈80%  n=10万≈84%
//	  cap = ezfy_cfg_limit.recon_success_pct（默认 95 = 封顶 95%），可在二战系统配置调整。
func ezfyReconSucceed(order *model.EzfyOrder) bool {
	n := ezfyReconPlaneCount(order)
	if n <= 0 {
		return false
	}
	// 封顶百分比（默认 95，0 无意义 → 回落默认）
	capPct := ezfyCfg.limit.ReconSuccessPct
	if capPct <= 0 || capPct > 100 {
		capPct = ezfyReconSuccessPctDef
	}
	// 按数量级平滑上升的 logistic 曲线：x = log10(n)，天然 ≤ 封顶
	x := math.Log10(float64(n))
	rate := (capPct / 100) / (1 + math.Exp(-ezfyReconK*(x-ezfyReconX0)))
	return rand.Float64() < rate
}

// addReport 战报写入
// scoutReportBody 侦查报告正文
//
// ★★ 2026-10-02 用户规则重做：报告细节受**侦查方侦察技巧(科技12)等级**卡控，
// 不再让低等级侦察把别人「看个精光」：
//
//	<6 级：建筑/科技只能模糊看到有哪些（等级不清晰）；兵种只能模糊看到类别
//	      （陆军/海军/空军/城防），数量无法查清
//	 6 级：建筑、科技等级精确
//	 7 级：城防 + 陆军兵种名与精确数量
//	 8 级：海军、空军兵种名与精确数量
//	 9 级：该城所有军官、谁是城守、等级
//	10 级：玩家最后在线时间
//
// 野地与寇城（无城市建筑）保持原样只给守军情况。
// 城市报告格式取自 `参考材料/开发文档/侦察报告1.txt`。
func (h *EzfyHandler) scoutReportBody(uid uint, order *model.EzfyOrder, targetName string,
	target *model.EzfyCity, defender []ezfyUnitGroup) string {
	// 侦查方侦察技巧等级（科技12，0-10），决定报告能看清多少细节
	scoutLv := h.techMap(uint(order.CityId))[ezfyReconTechID]
	var b strings.Builder
	fmt.Fprintf(&b, "公文报告:侦查报告\n我方一支部队对%s[%d，%d]进行了侦查。侦查过程中未受到任何阻拦。\n",
		targetName, order.TargetX, order.TargetY)

	// 野地 / 寇城: 只有守军
	if target == nil {
		b.WriteString("守军情况: ")
		if len(defender) == 0 {
			b.WriteString("无敌军驻守")
		}
		// ★ 2026-10-05 侦查报告兵种名**统一用基础兵种名**（不带阵营前缀），与战斗报告口径一致
		cfgType := 1
		level := ezfyWildlandLevel(order.TargetX, order.TargetY)
		if order.TargetType == 2 {
			cfgType = 3
			level = ezfyKouLevel(order.TargetX, order.TargetY)
		} else if ezfyIsSeaWildTerrain(ezfyTerrainEx(order.TargetX, order.TargetY)) {
			// ★ 2026-10-04 与地图同口径（Ex 含覆盖表/沿海平原）
			// ★ 2026-10-05 岛屿也属于海野 → 侦查守军/守将按海野配置
			cfgType = 2
		}
		for _, g := range defender {
			if cfg := ezfyCfg.troop(g.TroopId); cfg != nil {
				// camp 传 0 → 基础兵种名
				name := ezfyCfg.troopName(g.TroopId, 0)
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

	// 建筑: 按建筑 id 排序, 同类多座依次列出
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
	// ★ 侦察技巧<6级: 只能模糊看到已有的建筑, 建筑等级不清晰
	if scoutLv < 6 {
		b.WriteString("建筑(等级不详): ")
		for _, id := range ids {
			b.WriteString(names[id])
			b.WriteString(" ")
		}
	} else {
		b.WriteString("建筑等级 ")
		for _, id := range ids {
			b.WriteString(names[id])
			for _, lv := range levels[id] {
				fmt.Fprintf(&b, "%d,", lv)
			}
		}
	}
	b.WriteString("\n")

	// 军队/城防分列（★ 2026-10-05 兵种名统一用基础兵种名，与战斗报告口径一致）
	// ensureProfile 仅为兜底补建目标档案（老数据可能没档案），这里不再取阵营
	_ = h.ensureProfile(target.UserID)
	// ★ 2026-10-06 计谋伪装：目标城主人有生效中的「恫疑虚喝/隐真示假」时，
	//   展示的兵种与数量换成随机假数据（实际兵种数量不变），仅对能看清数量的敌人生效。
	fake := h.ezfySchemeFake(target)
	var defTxt, armyTxt, navyTxt, airTxt []string
	var hasDef, hasArmy, hasNavy, hasAir bool
	for tid, cnt := range h.troopMap(target.ID) {
		cfg := ezfyCfg.troop(tid)
		if cfg == nil || cnt <= 0 {
			continue
		}
		// camp 传 0 → 基础兵种名
		name := ezfyCfg.troopName(tid, 0)
		if name == "" {
			name = cfg.Name
		}
		if fake != 0 {
			// 随机兵种（同类别内随机）+ 假数量：恫疑虚喝=1亿 / 隐真示假=1000内
			if ft := ezfySchemeFakeTroop(cfg.Type); ft != nil {
				name = ft.Name
				if name == "" {
					name = ft.NameAxis
				}
			}
			cnt = ezfySchemeFakeCount(fake)
		}
		switch cfg.Type {
		case 1: // 海军
			hasNavy = true
			navyTxt = append(navyTxt, fmt.Sprintf("%s%d ", name, cnt))
		case 2: // 陆军
			hasArmy = true
			armyTxt = append(armyTxt, fmt.Sprintf("%s%d ", name, cnt))
		case 3: // 空军
			hasAir = true
			airTxt = append(airTxt, fmt.Sprintf("%s%d ", name, cnt))
		case 4: // 城防
			hasDef = true
			defTxt = append(defTxt, fmt.Sprintf("%s×%d ", name, cnt))
		}
	}
	if scoutLv < 7 {
		// ★ 侦察技巧<7级: 兵种类型只能模糊查看, 数量无法查清
		var cats []string
		if hasDef {
			cats = append(cats, "城防")
		}
		if hasArmy {
			cats = append(cats, "陆军")
		}
		if hasNavy {
			cats = append(cats, "海军")
		}
		if hasAir {
			cats = append(cats, "空军")
		}
		if len(cats) > 0 {
			b.WriteString("兵力(数量不明): " + strings.Join(cats, "、") + "\n")
		} else {
			b.WriteString("兵力: 无驻军\n")
		}
	} else {
		// 7级: 城防+陆军精确; 8级: 海军+空军精确
		if len(defTxt) > 0 {
			b.WriteString("城防数量：" + strings.Join(defTxt, "") + "\n")
		}
		if len(armyTxt) > 0 {
			b.WriteString("军队数量(陆军)：" + strings.Join(armyTxt, "") + "\n")
		}
		if scoutLv >= 8 {
			if len(navyTxt) > 0 {
				b.WriteString("军队数量(海军)：" + strings.Join(navyTxt, "") + "\n")
			}
			if len(airTxt) > 0 {
				b.WriteString("军队数量(空军)：" + strings.Join(airTxt, "") + "\n")
			}
		} else {
			// 7级: 海/空军仍模糊, 只提示类别存在
			var cats []string
			if hasNavy {
				cats = append(cats, "海军")
			}
			if hasAir {
				cats = append(cats, "空军")
			}
			if len(cats) > 0 {
				b.WriteString("兵力(数量不明)：" + strings.Join(cats, "、") + "\n")
			}
		}
	}

	// ★ 侦察技巧≥9级: 才能看到该城所有军官、谁是城守、等级
	if scoutLv >= 9 {
		officers := h.officerList(target.ID)
		offText := ""
		for _, o := range officers {
			tag := ""
			if o.Position == ezfyPositionGuard {
				tag = "城守"
			}
			offText += fmt.Sprintf("%s(%d级)%s、", o.Name, o.Level, tag)
		}
		if offText == "" {
			offText = "无"
		}
		b.WriteString("将领等级：" + offText + "\n")
	}

	// 科技: <6级 只能模糊查看已有科技, ≥6级 精确等级
	techIDs := []int{}
	tm := h.techMap(target.ID)
	for id := range tm {
		techIDs = append(techIDs, id)
	}
	sort.Ints(techIDs)
	techText := ""
	if scoutLv < 6 {
		for _, id := range techIDs {
			if cfg := ezfyCfg.tech(id); cfg != nil {
				techText += cfg.Name + " "
			}
		}
		b.WriteString("科技(等级不详)：" + techText + "\n")
	} else {
		for _, id := range techIDs {
			if cfg := ezfyCfg.tech(id); cfg != nil {
				techText += fmt.Sprintf("%s%d ", cfg.Name, tm[id])
			}
		}
		b.WriteString("科技等级：" + techText + "\n")
	}

	// ★ 侦察技巧≥10级: 才能看到玩家最后在线时间（取账号最近活跃时间）
	if scoutLv >= 10 {
		var owner model.User
		if err := h.DB.First(&owner, target.UserID).Error; err == nil && owner.LastActiveAt != nil {
			b.WriteString("最后在线时间：" + owner.LastActiveAt.Format("2006-01-02 15:04:05") + "\n")
		}
	}

	if scoutLv < 10 {
		fmt.Fprintf(&b, "（侦察技巧%d级, 部分情报无法查清）\n", scoutLv)
	}
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
			case uint:
				// ★ 2026-10-01 修复：EzfyOrder.ID 是 uint，缺此分支导致战报 order_id 全部落 0
				r.OrderId = int64(n)
			}
		case 2:
			// ★ 2026-10-01 军情按当前城过滤：战报创建处把所属城市 ID 带进来
			switch n := v.(type) {
			case int64:
				r.CityId = n
			case uint:
				r.CityId = int64(n)
			case int:
				r.CityId = int64(n)
			}
		}
	}
	// ★ 2026-10-01 军情按当前城过滤：很多调用点没传 city_id（城市落 NULL/0），
	//   集中兜底——有订单号就按订单回查出发点城市，让新战报都正确归属城市。
	if r.CityId == 0 && r.OrderId > 0 {
		var ocid int64
		h.DB.Model(&model.EzfyOrder{}).Where("id = ?", r.OrderId).Pluck("city_id", &ocid)
		if ocid > 0 {
			r.CityId = ocid
		}
	}
	h.DB.Create(&r)
}
