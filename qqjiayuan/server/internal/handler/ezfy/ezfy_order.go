package ezfy

import (
	"encoding/json"
	"fmt"
	"log"
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

// ezfyKouRuinSet 一次性取回 [x0,x1]×[y0,y1] 范围内**已变废墟**的寇城坐标集合
// （ezfy_map_area.area_type=2 且 start_time > now 的那些格）。
//
// ★★ 2026-10-13 性能（用户反馈「点地图卡顿 / 点方向键也卡顿」）：
//
//	MapView 末尾的「发现精英中立城市」要由近及远扫到 ±30（约 3700 格），原来每格都调
//	`ezfyIsKouCity` —— 它内部 1/97 概率要查一次 ezfy_map_area，实测整张地图一次请求
//	要打 **~38 条跨 WAN SQL**（线上 1 核 1G + 远程 RDS，每次拖动地图都白挨这一下）。
//	现在改成**先把整个扫描范围的废墟一次性取回（1 条 SQL）**，扫描时纯内存比对。
func (h *EzfyHandler) ezfyKouRuinSet(x0, y0, x1, y1 int) map[string]bool {
	var rows []model.EzfyMapArea
	h.DB.Select("x, y, start_time").
		Where("area_type = 2 AND x BETWEEN ? AND ? AND y BETWEEN ? AND ?", x0, x1, y0, y1).
		Find(&rows)
	now := time.Now().UnixMilli()
	out := map[string]bool{}
	for _, a := range rows {
		if a.StartTime > now {
			out[fmt.Sprintf("%d,%d", a.X, a.Y)] = true
		}
	}
	return out
}

// MapView 以中心坐标返回区域地图（默认以当前城为中心）
func (h *EzfyHandler) MapView(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	cx, _ := strconv.Atoi(c.Query("x"))
	cy, _ := strconv.Atoi(c.Query("y"))
	// ★ 2026-10-30 活动目标每日上限按「当前城市」判定：次数用完后地图上按普通野地展示。
	//   城市/配额都走**懒加载**：视野里没有活动目标时连一条 COUNT 都不发（避免拖动地图多一次查库）。
	var actCity model.EzfyCity
	actCityLoaded := false
	currentCity := func() *model.EzfyCity {
		if !actCityLoaded {
			actCity = h.getOrCreateCity(uid)
			actCityLoaded = true
		}
		return &actCity
	}
	if cx == 0 && cy == 0 {
		city := currentCity()
		cx, cy = city.X, city.Y
	}
	actDailyChecked, actDailyReached := false, false
	actDailyBlocked := func() bool {
		if !actDailyChecked {
			actDailyReached = h.ezfyActDailyLimitReached(int64(currentCity().ID))
			actDailyChecked = true
		}
		return actDailyReached
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
	// ★ 2026-10-31 地图格显示「番号」：城主 → 番号（各区自己的 ezfy_profile.banner）。
	userBanners := map[uint]string{}
	userIDs := []uint{}
	for _, c := range myCities {
		userIDs = append(userIDs, c.UserID)
	}
	if len(userIDs) > 0 {
		var users []model.User
		h.home().Select("id, nickname").Where("id IN ?", userIDs).Find(&users)
		for _, u := range users {
			userNames[u.ID] = u.Nickname
		}
		// 游戏昵称覆盖家园昵称：ezfy_profile.nickname 有值的玩家用游戏内的名字。
		// ★ 同一查顺带取 banner（番号），地图格第一行显示番号而不是统一「城市」。
		var ps []model.EzfyProfile
		h.DB.Select("user_id, nickname, banner").Where("user_id IN ?", userIDs).Find(&ps)
		for _, p := range ps {
			if p.Nickname != "" {
				userNames[p.UserID] = p.Nickname
			}
			userBanners[p.UserID] = ezfyBannerOf(&p)
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

	// ★ ★ 2026-10-08 「敌人城市标红」：
	//   个人宣战/被宣战、军团敌对、军团之间战争 的对象城市 → 前端格子标红，方便找敌人。
	//   镜像 allyUsers 模式，但**惰性**：视野里没有玩家城市格就不做任何战争检索
	//   （拖动地图到海/野地时零开销），首次遇到城市格才计算，并带 30s 短缓存避免拖动时重复查询。
	var enemyUsers map[uint]bool

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

	// ★★ 2026-10-13 性能（用户反馈「点地图/点方向键卡顿」）：本请求里所有「寇城判定」都走这个
	//   内存闭包 —— 寇城废墟集合已一次性预取（见 ezfyKouRuinSet），不再逐格查库。
	//   主循环(±r，r≤15) 与下面精英城扫描(±30) 都在预取范围(±30)内。
	kouRuins := h.ezfyKouRuinSet(cx-30, cy-30, cx+30, cy+30)
	isKouFast := func(x, y int) bool {
		if mk := ezfyMarkKindAt(x, y); mk > 0 {
			return mk == model.EzfyMarkKou || mk == model.EzfyMarkActKou
		}
		if ezfyAbs(x*5381^y*33)%97 != 0 {
			return false
		}
		if ezfyAbs(x-250)+ezfyAbs(y-250) <= 150 {
			return false
		}
		return !kouRuins[fmt.Sprintf("%d,%d", x, y)]
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
				// ★ 2026-10-31 地图格第一行用城主「番号」（默认 新），第二行仍是坐标。
				banner := userBanners[c.UserID]
				if banner == "" {
					banner = ezfyDefaultBanner
				}
				cell["banner"] = banner
				cell["city_level"] = c.CityLevel
				cell["owner"] = userNames[c.UserID]
				cell["mine"] = c.UserID == uid
				cell["ally"] = allyUsers[c.UserID]
				// ★ 2026-10-08 敌人城市标红：首次遇到城市格才惰性检索（无城市视野零开销）
				if enemyUsers == nil {
					enemyUsers = h.ezfyMapEnemyUsersCached(uid, myMb.CorpsId)
				}
				cell["enemy"] = enemyUsers[c.UserID]
				// ★ 该城主的同盟（军团）名；没加入军团时为空串，前端显示「无」
				cell["corps_name"] = corpsNames[c.UserID]
			} else {
				kou := isKouFast(x, y)
				switch {
				case ezfyIsSeaWildTerrain(terrain) && terrain != ezfyTerrainIsland:
					// 海洋 8 / 渔场 10 / 矿礁 11：
					// 纯海洋 → 「海洋」(详情页不显示守军/军官/出征按钮)；带野地 → 海底森林/渔场/矿礁。
					// ★ 2026-10-05 统一走 ezfyWildTerrainDisplayName（口径只有一处），这里 knownWild=false。
					// ★ 2026-10-20 新增渔场/矿礁，和海洋同走海野分支，避免被寇城哈希覆盖。
					cell["area_type"] = 1
					lvl := ezfyWildlandLevel(x, y)
					cell["name"] = ezfyWildTerrainDisplayName(x, y, false)
					if terrain == ezfyTerrainSea && lvl == 0 {
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
				// ★★ 2026-10-30 名将活动目标不卡每日上限 → 本城次数用完后，名将格**照常**显示为活动目标。
				if act := h.ezfyActTargetType(x, y); act > 0 && !h.playerOwnsActWildGeneral(uid, x, y) {
					actLevel := ezfyMarkLevelAt(x, y)
					aw := ezfyActWildAt(x, y)
					// ★ 2026-10-24 守将来源与战斗/详情同口径：aw 启用优先，否则回退活动目标配置
					dc := ezfyActDefConfigOf(aw, act, actLevel)
					if h.ezfyActNamedTargetAt(x, y) || !actDailyBlocked() {
						cell["act_type"] = act
						cell["act_level"] = actLevel
						cell["act_name"] = ezfyActTargetName(act)
						// 格子名直接用活动标签, 目标详情页标题即「活动野地2级 / 特殊城市3级」
						cell["name"] = ezfyActTargetLabel(act, actLevel)
						// ★ 2026-09-30 带守将军官的活动野地加特殊标识（玩家区分普通活动野地与带名将守将的）
						if dc.OfficerId > 0 {
							cell["act_officer"] = true
							if g := ezfyCfg.general(dc.OfficerId); g != nil {
								cell["act_officer_name"] = g.Name
								cell["act_officer_kind"] = g.Kind // 1普通军官 2名将
								cell["act_officer_star"] = g.Star
							}
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
				te := ezfyTerrainEx(ex, ey)
				if te == ezfyTerrainSea || te == ezfyTerrainFishing || te == ezfyTerrainOreReef ||
					!isKouFast(ex, ey) {
					continue // 海里不刷精英寇城；岛屿(7)保持原逻辑
				}
				elite = gin.H{"x": ex, "y": ey, "level": ezfyKouLevel(ex, ey)}
			}
		}
	}
	resp.OK(c, gin.H{"cells": cells, "cx": cx, "cy": cy, "elite": elite})
}

// ezfyMapEnemyUsers 当前玩家在「地图上标红的敌人」用户集合：
//
//	① 个人战争：我 宣战 / 被宣战 的玩家（待生效 status=1、交战中 status=2 都算，含未过期的）。
//	② 军团敌对：我所在军团标记为「敌对」的军团 —— 全体成员。
//	③ 军团战争：我所在军团正处于「宣战/交战」的敌方军团 —— 全体成员。
//	未加入军团的玩家只算 ①。镜像 allyUsers 的构建方式，一处计算、每格直接比对。
//
// ★ 2026-10-08 「敌人城市标红」缓存：同一玩家 30s 内拖动地图不重复检索战争关系，
//
//	避免移动地图卡顿。键=当前玩家 uid（myCorps 由其推导，缓存期内加入/退军团最多延迟 30s 生效）。
type ezfyEnemyCacheEntry struct {
	set map[uint]bool
	exp int64
}

var (
	ezfyEnemyCacheMu sync.Mutex
	ezfyEnemyCache   = map[uint]ezfyEnemyCacheEntry{}
)

const ezfyEnemyCacheTTLMs = 30 * 1000 // 30 秒

func (h *EzfyHandler) ezfyMapEnemyUsersCached(uid, myCorps uint) map[uint]bool {
	now := time.Now().UnixMilli()
	ezfyEnemyCacheMu.Lock()
	if e, ok := ezfyEnemyCache[uid]; ok && e.exp > now {
		ezfyEnemyCacheMu.Unlock()
		return e.set
	}
	ezfyEnemyCacheMu.Unlock()

	set := h.ezfyMapEnemyUsers(uid, myCorps)

	ezfyEnemyCacheMu.Lock()
	ezfyEnemyCache[uid] = ezfyEnemyCacheEntry{set: set, exp: now + ezfyEnemyCacheTTLMs}
	ezfyEnemyCacheMu.Unlock()
	return set
}

func (h *EzfyHandler) ezfyMapEnemyUsers(uid, myCorps uint) map[uint]bool {
	enemy := map[uint]bool{}

	// ① 个人宣战 / 被宣战（EzfyWar.status 只有 1/2，无"已结束"态，须按 expire_time 过滤过期）
	now := time.Now().UnixMilli()
	var wars []model.EzfyWar
	h.DB.Where("status IN (1,2) AND expire_time > ? AND (atk_user_id = ? OR def_user_id = ?)",
		now, uid, uid).Find(&wars)
	for _, w := range wars {
		if w.AtkUserId == uid {
			enemy[w.DefUserId] = true
		} else {
			enemy[w.AtkUserId] = true
		}
	}

	if myCorps == 0 {
		return enemy
	}

	// ② 军团敌对：我方标记过 target 为「敌对」
	enemyCorps := map[uint]bool{}
	var rels []model.EzfyCorpsRelation
	h.DB.Where("corps_id = ? AND type = 2", myCorps).Find(&rels)
	for _, r := range rels {
		enemyCorps[r.TargetCorpsId] = true
	}

	// ③ 军团战争：我方与对方的宣战 / 交战（corpsWarTick 会把过期的清成 status=3，这里只认 1/2）
	var cws []model.EzfyCorpsWar
	h.DB.Where("status IN (1,2) AND (atk_corps_id = ? OR def_corps_id = ?)", myCorps, myCorps).Find(&cws)
	for _, cw := range cws {
		if cw.AtkCorpsId == myCorps {
			enemyCorps[cw.DefCorpsId] = true
		} else {
			enemyCorps[cw.AtkCorpsId] = true
		}
	}

	// 敌方军团的全体成员都算敌人
	if len(enemyCorps) > 0 {
		cids := make([]uint, 0, len(enemyCorps))
		for cid := range enemyCorps {
			cids = append(cids, cid)
		}
		var mbs []model.EzfyCorpsMember
		h.DB.Select("user_id").Where("corps_id IN ?", cids).Find(&mbs)
		for _, m := range mbs {
			enemy[m.UserId] = true
		}
	}
	return enemy
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
	// ★ 2026-10-30 当前城市当日活动攻击次数用完后，详情也按普通野地展示（与地图/战斗同口径）
	//   ★★ 名将活动目标不卡每日上限 → 次数用完后名将格仍展示活动详情。
	//   先判是不是活动目标，普通野地详情不再多一次配额/城市查询。
	if act := h.ezfyActTargetType(x, y); act > 0 && !h.playerOwnsActWildGeneral(uid, x, y) {
		if h.ezfyActNamedTargetAt(x, y) {
			resp.OK(c, h.ezfyActWildlandView(uid, profile.Camp, x, y, act))
			return
		}
		curCity := h.getOrCreateCity(uid)
		if !h.ezfyActDailyLimitReached(int64(curCity.ID)) {
			resp.OK(c, h.ezfyActWildlandView(uid, profile.Camp, x, y, act))
			return
		}
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
	//	  海野(海洋 8 / 岛屿 7 / 渔场 10 / 矿礁 11) → 2 ；寇城 → 3 ；其余 → 1（陆地野地）。
	//	  ★ 2026-10-23 沿海平原（带野地）也按海野 → 2（寇城仍优先）。
	//	⚠️ 客户端的 `type` 查询参数已不再参与判定（传了也不影响结果），保留只是为了兼容旧前端。
	ttype := 1
	switch {
	case ezfyIsSeaWildTerrain(ezfyTerrainEx(x, y)):
		ttype = 2
	case h.ezfyIsKouCity(x, y):
		ttype = 3
	case ezfyIsSeaWildRules(ezfyTerrainEx(x, y)) && ezfyWildlandLevel(x, y) > 0:
		// ★ 2026-10-23 沿海平原也按海野玩法（用户规则）→ 详情预览海野守军。
		//   ★ 寇城优先：沿海平原上可能压着寇城，不能因为「按海野」把寇城挡掉；
		//   ★ 只有真的带野地（lvl>0）才算，空沿海平原仍走陆地口径（避免改坏无野地的空地）。
		ttype = 2
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
	// 地形显示名: 海洋里的野地→海底森林, 岛屿→岛屿, 渔场/矿礁→各自地形名, 寇城→平原(用户规范)
	// ★ 2026-10-05 用户纠正：① 纯海洋是「海洋」、海底森林是海洋里带野地的那块；② 岛屿(7) 也是海野玩法，
	//   但展示名保留「岛屿」。走 ezfyWildTerrainDisplayName。
	// ★ 2026-10-20 新增渔场(10)/矿礁(11)，同走该函数，展示名就是「渔场」「矿礁」。
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
	// ★ 2026-10-22 野地归属玩家名可点：除昵称外一并下发玩家 user_id，
	//   前端据此走 openPlayer → games/ezfy?cur=playerinfo（与城主名同入口）。
	var ownerUserId uint
	var w model.EzfyWildland
	// ★ 2026-10-16 一个坐标只应有一条野地记录（唯一索引 uk_wildland_xy 保证）；
	//   Order("id DESC") 是兜底：万一存量重复还没被启动清理收口，也取**最新**那条（= 最后占领者），
	//   否则会取到旧行（原主人）→ 详情显示「归属别人」、mine=false（用户反馈的 bug 现象）。
	if err := h.DB.Where("x = ? AND y = ?", x, y).Order("id DESC").First(&w).Error; err == nil && w.CityId > 0 {
		var oc model.EzfyCity
		if err := h.DB.First(&oc, w.CityId).Error; err == nil {
			ownerUserId = oc.UserID
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
	// ★ 2026-10-11 采集中状态：该野地是否有部队正在采集/驻守（任意玩家）。
	//   采集中的野地不能被征服/掠夺（createOrder 同步拦截），前端据此置灰按钮并提示。
	collecting := false
	if w.ID > 0 {
		collecting = h.ezfyWildlandOccupiedCount(int64(w.ID), 0) > 0
	}
	// ★★ 2026-10-19 是否有**玩家守军**（驻军 6 / 驻守采集 7，未结束）：
	//   有 → 守方是真人部队，出征页「自动战斗」必须强制「否」（抵达后进指挥室），
	//   与打玩家城市同口径。口径与 processArrive 构建守军时完全一致（ezfyWildDefenderOrderCond）。
	hasDefender := false
	if w.ID > 0 {
		var dn int64
		h.DB.Model(&model.EzfyOrder{}).
			Where("target_id = ? AND "+ezfyWildDefenderOrderCond, w.ID).Count(&dn)
		hasDefender = dn > 0
	}
	resp.OK(c, gin.H{
		"x": x, "y": y, "type": ttype, "level": level,
		"name": cfg.Des, "troops": previews,
		// ★ 2026-09-25：这里下发的「胜利奖励」也要带上「野地获取资源倍率」——
		//   前端会显示成「胜利奖励：粮/钢/油/稀矿 各N」，只在实际结算时放大就会
		//   「预览 1 万、真打给 5 万」，玩家反而以为算错了。两边必须同一口径。
		"res_min": ezfyScaleByWildResMult(cfg.ResMin), "res_max": ezfyScaleByWildResMult(cfg.ResMax),
		"res_mult": ezfyWildResMult(), "terrain": ezfyTerrainEx(x, y),
		"terrain_name":  terrainName,
		"continent":     ezfyRegionName(x, y),
		"treasures":     treasures,
		"gather_res":    gatherRes,
		"treasure":      cfg.Treasure, // 寇城宝物档次(初级/中级/高级)
		"owner":         owner,
		"owner_user_id": ownerUserId, // ★ 归属玩家 user_id（前端点名字进统帅页 playerinfo；未占领=0）
		"mine":          isMine,      // ★ 自己的附属野地（不能侦查/掠夺/征服，要先放弃）
		"collecting":    collecting,  // ★ 2026-10-11 正在被采集（不能征服/掠夺）
		"has_defender":  hasDefender, // ★ 2026-10-19 有玩家守军 → 出征页强制手动指挥
		"is_starred":    starCnt > 0, // ★ 2026-09-30 收藏态
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

// ezfyMustLeadOfficer 该出征类型是否**必须**携带带队军官（★ 2026-10-30 用户规则）。
//
//	必须带：侦查(1) / 掠夺(2) / 征服(3) / 采集(4) / 运输(5) / 增援(6) / 驻守采集(7)
//	不用带：派遣(8，城际调兵) —— 唯一例外
//
// ★ 与「一个军官同时只能带一支队伍」的卡控配套：不带军官就能出兵，
// 等于绕开军官数量限制（有多少兵/多少资源就能派多少队），所以出征一律要求带队军官。
// ezfyMustLeadOfficer 该出征类型是否**必须**带一名带队军官（唯一判定）。
//
// ★ 2026-10-30 用户规则：除派遣(8)外，出征一律必须带军官。
// ★★ 2026-10-31 用户规则调整：**运输(5) 也改成「可带可不带」** —— 城际运兵/运资源是后勤任务，
// 不该因为「本城没有空闲军官」就发不出去（派遣(8) 同样可选）。
// ⚠️ 是**可选**不是**禁止**：传了 officer 仍走完整校验（存在/在职/未俘虏/未出征 + CAS 抢占），
// 到达后照样随军调往目标城（见 processArrive 的 5/6 分支 moveOfficerTo）。
// 现在必带军官的只剩：侦查(1)/掠夺(2)/征服(3)/采集(4)/增援(6)/驻守采集(7)。
func ezfyMustLeadOfficer(orderType int) bool {
	switch orderType {
	case 1, 2, 3, 4, 6, 7:
		return true
	}
	return false
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
	// ★★ 2026-10-20 修复「明明没宣战却提示已与该玩家宣战」：
	//   个人宣战没有定时归档任务，过期后 status 仍是 1/2（只有军团宣战有 corpsWarTick）。
	//   这里必须按 expire_time 过滤，否则只要双方历史上打过一次，过期记录就会一直命中：
	//   warStatus() 按过期时间正确返回 0（地图显示[宣战]），但 DeclareWar 用 getWar 判重
	//   → 点[宣战]永远回「已与该玩家宣战(待生效或交战中)」，该玩家再也无法重新宣战。
	now := time.Now().UnixMilli()
	var w model.EzfyWar
	err := h.DB.Where("((atk_user_id = ? AND def_user_id = ?) OR (atk_user_id = ? AND def_user_id = ?)) AND status IN (1,2) AND expire_time > ?",
		a, b, b, a, now).Order("id DESC").First(&w).Error
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
		// ★ 2026-10-07 自动战斗：1=到达即自动打完（无需指挥）；0=到达后进指挥室等玩家指挥。
		//   用**指针**区分「没传」与「显式传 0」。打玩家城市（target_type=3）后端**强制按 0 处理**
		//   （真人守方必须留指挥机会），前端那一项也是只读的 —— 这里再兜一道，防直接调接口绕过。
		AutoBattle *int `json:"auto_battle"`
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
		// ★ 2026-10-24 修复「被占城还能出征」：出征源城必须是自己的空闲城。
		//   显式传了 city_id 但不存在/不属于自己/已被占领 → 直接拒绝，不再静默回落主城。
		if req.CityId > 0 {
			resp.ParamError(c, "城市不存在或已被占领")
			return
		}
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
	// ★ 2026-10-07 自动战斗（出征页可配）：
	//   打野地 / AI 寇城 → 默认「是」(1)；打玩家城市(target_type=3) → **强制「否」(0)**，
	//   守方是真人、必须给指挥机会（前端那一项只读，这里再兜一道防直接调接口绕过）。
	autoBattle := 1
	if req.TargetType == 3 {
		autoBattle = 0
	} else if req.AutoBattle != nil {
		autoBattle = *req.AutoBattle
	}
	if msg := h.createOrder(uid, city, req.OrderType, req.TargetX, req.TargetY, req.TargetType, req.TargetId, req.Troops, req.Resources, req.Officer, waitMin, req.Gather, autoBattle, cities); msg != "" {
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
	if city == nil {
		resp.ParamError(c, "城市不存在或已被占领")
		return
	}
	// ★ 2026-10-05 性能（用户反馈「/order/preview 还是 2s」）：原 refreshCity 串行跑
	//   建筑/科技/队列/资源懒结算 + processOrders（5+ 条跨 WAN RDS），是 2s 的根源。
	//   预览是**纯只读计算**（不下达命令、不扣资源），订单/资源结算交给 /view 轮询照常
	//   推进（资源数值最多滞后一轮轮询，可接受）。这里一个并行波把预览需要的
	//   科技等级 / 指挥室 / 司令部 / 带队军官 / 集结令持有量 全部取完（1 RTT）。
	var userTechs []model.EzfyUserTech
	var stationLv, hqLv int
	var lead *model.EzfyOfficer
	var gatherHave int
	// ★ 2026-10-09 「自己被处于指挥时，指挥里面的兵种不能再出征」：
	//   本城正在被攻击时，参战兵种集合（供前端禁用数量框 + 提示），与其他查询并行取。
	var defLocked map[int]bool
	var wg sync.WaitGroup
	wg.Add(6)
	go func() { defer wg.Done(); h.DB.Where("user_id = ?", uid).Find(&userTechs) }()
	go func() { defer wg.Done(); stationLv = h.buildingLevel(city.ID, 20) }()
	go func() { defer wg.Done(); hqLv = h.buildingLevel(city.ID, 13) }()
	go func() {
		defer wg.Done()
		if req.Officer != "" {
			lead = h.officerByName(city.ID, req.Officer)
		}
	}()
	go func() { defer wg.Done(); gatherHave = h.itemCount(uid, ezfyGatherItemID()) }()
	go func() { defer wg.Done(); defLocked = h.cityDefendLocked(city.ID) }()
	wg.Wait()
	techMap := map[int]int{}
	for _, t := range userTechs {
		techMap[t.TechId] = t.Level
	}
	// ★ 2026-10-29 用户规则：没有「科研中心」建筑的城市不享受科技加成 →
	//   出征预览的负重（标准化装卸）与真实出征同口径，出发城没科研中心就按 0 级算。
	//   （与上面两个 buildingLevel 并行块合并成同一次建筑列表查询代价相同，仍是一次 buildingList）
	if h.buildingLevel(city.ID, ezfyAcademyBuildingID) <= 0 {
		techMap = map[int]int{}
	}

	valid := []ezfyUnitGroup{}
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
	}
	// ★ 2026-09-28 负重统一走 ezfyCarryCapOf（含「标准化装卸」加成）。
	//   原来这里手写 `carry += cfg.Carry * count`，与 ezfyCarryCapOf 是**两套实现** ——
	//   加了科技加成后如果只改一处，出征页预览的负重就会和实际出征时校验的负重对不上。
	carry := h.ezfyCarryCapOf(valid, city.ID, techMap)
	distance := ezfyAbs(city.X-req.TargetX) + ezfyAbs(city.Y-req.TargetY)
	oilCost := h.ezfyOilCost(city, req.OrderType, distance, valid, req.Resources, h.officerOilReducePct(lead))

	var travelSec int64
	// ★★ 2026-10-09 行军速度按兵种算（机械增压=通用 / 喷气推进=空军 / 军官速度技能=陆·空·海），
	//   与 createOrder 共用 ezfyMarchSpeed，保证预览与实际一致。
	//   ★ 无可出征部队（如纯运输资源）时回落 300 —— 与 createOrder 同口径（原来预览这里会算成 0）。
	marchSpeed := h.ezfyMarchSpeed(valid, techMap, lead)
	if marchSpeed < 1 {
		marchSpeed = 300
	}
	if distance > 0 {
		// ★ 2026-10-05 科技/指挥室/带队军官来自上方并行块（零额外查询）；带队军官只查一次
		//   ★★ 2026-10-16 删除「无线电侦听(科技12)×2%/级」这一级：无线电侦听只加军情情报等级，
		//   不加速行军。真正加速行军的是 机械增压(科技10，通用) + 喷气推进(科技19，空军)，
		//   它们已折进上面的 ezfyMarchSpeed，不需要在这里再除一次。
		travelSec = int64(distance) * 60 * 300 / int64(marchSpeed)
		// ★★ 2026-10-16 运输站改「速度倍数」（玩家口径，与 cfg 文案「部队行动速度提高1.5~6.0倍
		//   (己方/盟友基地间)」对齐）：只按**出发城**运输站等级，1级1.5倍 … 10级6.0倍。
		//   只对「己方/盟友基地间」的调动生效（运输5 / 增援·驻守盟友城6 / 派遣8）；
		//   战斗与野外行动（侦查1/掠夺2/征服3/采集4/驻守采集7）**不吃**运输站加成。
		//   不再双向叠加（原「出发城+目的地城等级相加」已废，6倍×6倍会太夸张）。
		if ezfyStationOrderType(req.OrderType) {
			travelSec = int64(float64(travelSec) / ezfyStationSpeedMult(stationLv))
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
	// ★ 2026-10-16 加成可视化（玩家反馈"看不出有没有耗时加成"）：耗时提速清单按**真正生效的
	//   来源**列 —— 运输站(只按出发城等级、只对己方/盟友基地间订单)、机械增压(科技10，通用)、
	//   喷气推进(科技19，只对空军，部队不含空军时不列)。
	//   ★ 不再列「科技12 无线电侦听」：它只加军情情报等级，不加速行军。
	spdBonus := []string{}
	if ezfyStationOrderType(req.OrderType) {
		if m := ezfyStationSpeedMult(stationLv); m > 1 {
			spdBonus = append(spdBonus, fmt.Sprintf("运输站×%.1f倍", m))
		}
	}
	// ★ 2026-10-31 满级 50%：读配置（与 ezfyMarchSpeed 同源，避免「清单写的」≠「实际算的」）
	if t := ezfyTechEffectValueAt(10, techMap[10]); t > 0 && len(valid) > 0 {
		spdBonus = append(spdBonus, fmt.Sprintf("机械增压+%d%%", t))
	}
	if t := ezfyTechEffectValueAt(19, techMap[19]); t > 0 && ezfyHasAirTroops(valid) {
		spdBonus = append(spdBonus, fmt.Sprintf("喷气推进+%d%%", t))
	}
	// ★★ 2026-10-31 标准化装卸满级 50%：与 ezfyCarryCapOf 同源读配置（原来写死 techMap[13]*2）
	loadPct := ezfyTechEffectValueAt(13, techMap[13])
	resp.OK(c, gin.H{
		"oil_used":    oilCost,
		"oil_enough":  city.Oil >= oilCost,
		"oil_have":    city.Oil,
		"carry":       carry,
		"distance":    distance,
		"spd_bonus":   spdBonus,
		"load_bonus":  loadPct,
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
		// ★ 2026-10-09 本城正在被敌军攻击（防御指挥室进行中）：
		//   locked_troops = 参战兵种 id（前端把它们的兵力数量框禁用并提示「参战部队不能出征」）；
		//   不参与防御(def_move=-1)的兵种不在这个列表里 —— 它们不参与当次防御，照常可出征。
		"def_locked":    len(defLocked) > 0,
		"locked_troops": ezfyIntSetKeys(defLocked),
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
	// ezfyGatherItemName 集结令道具名。
	// ★★ 2026-10-10 不再硬编码道具 id：道具已统一迁到 1001+（见 seed/ezfy_cfg_gen.go），
	//   按名字解析（ezfyGatherItemID()）才不会在调号段时静默失效。
	ezfyGatherItemName = "集结令"
	// ezfyGatherDefaultPer 每个集结令提升的出征上限（配置表 param1 优先）
	ezfyGatherDefaultPer = 100000
	// ezfyGatherMaxDefault 单次出征最多使用多少个集结令的**默认值**。
	// ★ 「出征集结令上限后台管理系统可维护，最大默认 99」→ 默认 99（线上现值）。
	//   真正的上限由 ezfyGatherMax() 从 ezfy_cfg_limit.gather_max_per_order 读取，
	//   管理端「建筑上限配置」页可改，改完 cfgsReload() 即时生效。
	//   这个常量只在配置行缺失/为 0 时兜底。
	ezfyGatherMaxDefault = 99
)

// ezfyGatherItemID 集结令的道具 cfg_id（0 = 配置里没有）。
func ezfyGatherItemID() int { return ezfyCfg.itemIDByName(ezfyGatherItemName) }

// ezfyGatherBonusPer 每个集结令提升的出征上限（读配置 param1，缺省 10 万）
func ezfyGatherBonusPer() int64 {
	if it := ezfyCfg.item(ezfyGatherItemID()); it != nil && it.Param1 > 0 {
		return it.Param1
	}
	return ezfyGatherDefaultPer
}

// ezfyOrderTroopCap 本次出征的兵力上限
//
//	= 司令部等级 × 1万 × (1 + 参谋作业科技等级 × 10%)   ← 原版规则（司令部「每次出征上限N人」）
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

// ezfyOrderTroopCap 出征兵力上限：司令部等级 × 1万 × 参谋作业科技 + 集结令加成 + 军官军事加成。
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
	targetId int64, troops []ezfyUnitGroup, resources map[string]int64, officer string, waitMin, gather, autoBattle int,
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
	// ★ 2026-10-09 「自己被处于指挥时，指挥里面的兵种不能再出征」：
	//   本城是否正在被攻击 + 参战兵种集合，与上面几条查询**并行**取（零额外 RTT）。
	//   未被攻击时内部只有 1 条 COUNT，命中才补 2 条（rare）。
	var defLocked map[int]bool
	d := h.ezfyLoadCityData(uid, city, cityList,
		func() {
			h.DB.Model(&model.EzfyOrder{}).
				Where("user_id = ? AND city_id = ? AND order_type = ? AND target_id = ? AND target_x = ? AND target_y = ? AND start_time >= ?",
					uid, city.ID, orderType, targetId, targetX, targetY, time.Now().UnixMilli()-3000).
				Count(&dupCnt)
		},
		func() {
			// ★★ 2026-10-09 修复「出征队列数没卡住」：
			//   原口径 `user_id = ? AND status = 0` 有两个漏洞：
			//   ① 按 **user** 统计（不分城市）→ 多城玩家在 A 城派满后，B 城仍能派，
			//      且官方口径是「**当前城市**的司令部等级」决定该城能同时出征几支；
			//   ② 只数 `status=0`（在途）→ 采集(order_type=4/7) 到达后 status 变 1，
			//      **立刻腾出一个名额**，玩家只要在野地上反复「采集→到达→再采集」就能
			//      无限刷队伍（线上实测单玩家 115 支采集队，远超司令部等级）。
			//   → 改为按 `city_id` 统计**全部未结束**的该城外出队伍：
			//      status IN (0 在途, 1 驻守/采集, 2 返航, 5 战斗, 6 等待) —— 只有真正
			//      结束（3 完成 / 4 阵亡）才释放名额，出发城市与司令部等级一一对应。
			h.DB.Model(&model.EzfyOrder{}).
				Where("city_id = ? AND status IN (0,1,2,5,6)", city.ID).
				Count(&marchingCnt)
		},
		func() {
			var w model.EzfyWildland
			// ★ 2026-10-16 取最新那条（同坐标只应有一条，见 WildlandView 的说明）
			if err := h.DB.Where("x = ? AND y = ?", targetX, targetY).Order("id DESC").First(&w).Error; err == nil {
				ownWild = &w
			}
		},
		func() { defLocked = h.cityDefendLocked(city.ID) })
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
	// ★★ 2026-10-15 免战保护令**双向生效**（用户规则：免战城不能被别人打，当前免战城也不能打别人，
	//   只能打野地）。
	//   目标侧拦截早已实现（见下面「掠夺/征服玩家城需先宣战」段里的 `hasCityEffect(targetId, 2)`），
	//   这里补**出击侧**：**当前城市**处于免战保护期时，不能对**玩家城市**(targetType=3)发起
	//   侦查(1)/掠夺(2)/征服(3) —— 即「只能打野地/寇城」。
	//   ★ 放在最前面（早于兵种/兵力校验）：免战是「这件事根本不允许做」的硬规则，
	//     先报「免战中不能打别人」比报「兵力不足/侦察机不能混编」更贴切。
	//   ★ 按**当前城**判定（免战保护令是给单座城打的，与目标侧按目标城判定对称）；
	//     野地(1)/寇城(2) 不受限，运输(5)/增援(6)/派遣(8) 是自家/盟友城，也不受影响。
	if (orderType == 1 || orderType == 2 || orderType == 3) && targetType == 3 && targetId > 0 {
		if h.hasCityEffect(city.ID, 2) {
			return "当前城市处于免战保护期, 不能" + ezfyOrderTypeName(orderType) +
				"其他玩家城市(只能攻打野地/寇城)"
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
	// ★★ 2026-10-17 用户口径：增援(6)=驻守，携带资源无效
	//   （盟友间资源运输靠运输(5)）。前端已禁用增援的资源输入，这里后端兜底：
	//   就算客户端伪造资源参数，也强制清空，不给目标城也不带回。
	if orderType == 6 && hasRes {
		resources = map[string]int64{}
		hasRes = false
	}
	// ★ 2026-09-30 用户规则：侦察只有侦察机才能侦察。
	//   侦察(orderType 1)必须携带侦察机(troopId 9)，且只能带侦察机；
	//   其它部队不能混编侦察，防止用普通部队当炮灰探路。
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
		// ★★ 2026-10-18 增援(6)也可派往**自己的附属野地**（驻守）：
		//   部队抵达后编为该野地「驻军队列」(status=3)，敌军进攻时与采集部队一同防守。
		//   增援驻军不占「一块野地一支采集队」的名额，可与采集部队并存（用户需求）。
		if orderType == 6 && targetType != 3 {
			var wl model.EzfyWildland
			if err := h.DB.First(&wl, targetId).Error; err != nil || wl.CityId != int64(city.ID) {
				return "增援(驻守)只能派往自己占领的附属野地"
			}
			if officer == "" {
				return "请选择带队军官"
			}
			if len(validTroops) == 0 {
				return "增援必须携带部队"
			}
		} else if targetType != 3 || targetId <= 0 {
			return "目标必须为城市"
		} else {
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
				//   运量看负重（所以要用卡车等部队装）。
				//   ★ 2026-10-30 用户规则更新：运输**也要带带队军官**（唯一例外是派遣8）。
				if !own && !ally {
					return "运输目标必须是自己或同盟成员的城市"
				}
				if !hasRes {
					return "运输必须携带资源"
				}
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
	// ★ 2026-10-20 新增渔场/矿礁，同属海野，海军也可出征。
	if ezfyHasNavalTroops(validTroops) && !ezfyNavalTargetAllowed(targetX, targetY) {
		return "海军部队只能出征岛屿/海森/渔场/矿礁/海平"
	}
	total := int64(0)
	for _, t := range validTroops {
		total += t.Count
	}
	if orderType != 5 && total <= 0 {
		return "请选择出征部队"
	}
	// ★ 集结令：先校验参数（不超过管理端配置的单次上限、背包要够），再校验兵力与上限
	if gather < 0 {
		gather = 0
	}
	if gm := ezfyGatherMax(); gather > gm {
		return fmt.Sprintf("集结令单次最多使用%d个", gm)
	}
	if gather > 0 {
		if have := h.itemCount(uid, ezfyGatherItemID()); have < gather {
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
	// ★★ 2026-10-09 用户规则：「自己被处于指挥时，指挥里面的兵种不能再出征」。
	//
	//	城市正在被敌军攻击（防御指挥室进行中）时，**参战守军**不能出征 ——
	//	否则战损按城池当前兵力扣减时扣不到兵（守军已随出征离开），等于把守军从
	//	「出征」这个口子悄悄转移走，让这场防御白打。
	//	★ 例外（用户明确）：司令部标了「不参与防御」(def_move=-1) 的兵种**不参与当次防御**，
	//	  带走它们不影响战斗结算 → 照常放行。
	//	参战集合来自上面那个并行波（defLocked），未被攻击时为 nil（零额外查询）。
	if len(defLocked) > 0 {
		for _, t := range validTroops {
			if defLocked[t.TroopId] {
				name := "兵种" + strconv.Itoa(t.TroopId)
				if cfg := ezfyCfg.troop(t.TroopId); cfg != nil {
					name = cfg.Name
				}
				return "城市正在被敌军攻击(指挥中), 参战部队「" + name +
					"」不能出征；如不想让它参战, 可在司令部设为「避战」"
			}
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
			if err := h.DB.First(&oc, ownWild.CityId).Error; err == nil {
				if oc.UserID == uid {
					return "这是你自己的附属野地, 不能" + ezfyOrderTypeName(orderType) +
						"; 如要攻打请先在「附属野地」里[放弃]该野地"
				}
				// ★★ 2026-10-19 用户规则：**自己军团（同盟）成员的附属野地也不能出征/掠夺/征服**。
				//   与「不能攻击同盟成员的城市」（上面掠夺/征服玩家城那段 isAllyCity 判定）同口径，
				//   原来只拦了自己名下的野地 —— 同团成员的野地照打不误（用户反馈的 bug）。
				if h.isAllyCity(uid, int64(oc.ID)) {
					return "不能" + ezfyOrderTypeName(orderType) +
						"同盟(军团)成员的附属野地"
				}
			}
		}
		// ★ 2026-10-18 用户规则更新：**玩家附属野地允许被攻打**（即使有人在驻守/采集）。
		//   守军 = 该野地的驻军 + 采集部队（无则为 0），其在野地的兵力与随军资源
		//   随战斗正常结算（全灭则资源损失）。原「采集中不能征服/掠夺」拦截已废弃。
		//   中立野地/寇城仍按配置守军（见 processArrive 守军构建），这里不再拦截。
	}
	// ★★ 2026-10-30 用户规则「除了派遣, 出征必须带军官」：
	//   征服(3) / 侦查(1) / 掠夺(2) / 采集(4) / 增援(6) / 驻守采集(7) 一律**强制**携带一名带队军官；
	// ★★ 2026-10-31 调整：**运输(5) 也不再强制**（与派遣(8) 同为后勤任务，没空闲军官也要能发）。
	//   口径统一在 ezfyMustLeadOfficer（唯一实现），这里只调用。
	//   ★ 放在兵力/资源校验**之前**：没带军官是最容易犯的错，先报这一条更贴切。
	//   ★ 人机(is_bot) 例外：人机可能还没建军校、或本城军官全在外面，
	//     强制会让它永远打不出去（botAttack 已经优先带军官，能带就带）。
	if officer == "" && ezfyMustLeadOfficer(orderType) && city.IsBot == 0 {
		return "请选择带队军官"
	}
	if orderType == 4 {
		var wl model.EzfyWildland
		if err := h.DB.Where("id = ? AND city_id = ?", targetId, city.ID).First(&wl).Error; err != nil {
			return "只能采集已占领的野地"
		}
		// ★ 用户规则：采集/派遣都要带一个军官（带队）
		//   ★ 2026-10-30 上面已统一强制（ezfyMustLeadOfficer），这里保留作防御。
		if officer == "" {
			return "请选择带队军官"
		}
		// ★★ 2026-10-09 修复「自己的附属野地只能一支队伍采集」卡控完全失效（线上有人几百支）：
		//   原实现把「一野地一采集队」的卡控**只做在 `StartCollect`（开始采集）那一步**，
		//   靠 `ezfyWildlandCollectingCount`（order_type=7 + status=1 + arrive_time>0）拦住。
		//   但玩家根本不用走那一步 —— 在「附属野地 → [采集]」直接**下发新的采集订单**
		//   （order_type=4，见前端 openWildGather）就能往同一块自己的野地反复派兵：
		//   · 每支新部队的 `orderType==4`，与那条卡控查询的 `order_type=7` **口径不一致**，拦不到；
		//   · 采集订单到达后 `processArrive` 才把它改写成 `order_type=7`，
		//     而那时新的采集队早已在路上了（只要军官够多就能无限派）。
		//   于是同一坐标最终堆出几十上百支 `order_type=7` 的采集部队 ——
		//   正是线上「一块野地 29 支队伍采集」的成因（且全是从 [采集] 入口来的，
		//   从未经过 StartCollect 的那个检查）。
		//   → 在这里按**同一套口径**补卡控：本野地只要已有**任何一支**未结束的部队
		//     （在途 order_type=4 / 驻守采集 order_type=7），就不再接收新的采集订单。
		//     与 StartCollect 的「已有部队采集」口径对齐，做到「一块附属野地同时只有一支队伍」。
		if n := h.ezfyWildlandOccupiedCount(targetId, 0); n > 0 {
			return "该野地已有部队在采集(一支队伍只能采一块野地), 请先[召回]再重新采集"
		}
	}
	if orderType == 7 {
		var wl model.EzfyWildland
		if err := h.DB.Where("id = ? AND city_id = ?", targetId, city.ID).First(&wl).Error; err != nil {
			return "只能派遣到已占领的野地"
		}
		if officer == "" {
			return "请选择带队军官"
		}
	}
	// 带队军官校验: 必须存在且在职(未出征/非俘虏)
	// ★ 2026-10-05 性能：军官只查一次并全程复用（原来「校验/移速技能/军事加成/置出征态」各查一次 = 4 条）
	// ★ 2026-10-24 同名军官不误伤：officerByNameUsable 在同名里优先挑「在职且未在外」的那位；
	//   在外校验按军官 **id**（存量在途订单已由 seed 迁移回填 officer_id）。
	var lead *model.EzfyOfficer
	if officer != "" {
		lead = h.officerByNameUsable(city.ID, officer)
		if lead == nil {
			return "军官不存在"
		}
		// ★ 用户规则：同一座城市里，一个军官同时只能带一支队伍出征。
		//   只要他还有未结束的命令（行军中/驻守中/返航中），就不能再接新命令。
		//   这里查命令表而不是只看 officer.Status —— 后者可能因历史数据漂移不准。
		// ★ 2026-09-24 俘虏出征 bug 加固：只有 Status=0(在职) 的军官才能带队，
		//   Status=1(出征中)/2(被俘) 一律拒绝（历史数据里被俘军官可能 IsCaptive=0, 单看字段会漏拦）。
		if lead.Status != 0 || h.officerBusyOrderByID(city.ID, officer, uint(lead.ID)) {
			return "军官" + lead.Name + "出征中"
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
		// ★★ 2026-10-07 修复「出征被占领的城市 → 出征队列空」：
		//   城市正在被占领（ezfy_occupy.status=1）时，它的归属与守军处于**中间态**
		//   （原守方还在、占领方已写入），此时再发起掠夺/征服会算出异常结果，
		//   线上表现就是「出征后跳转出征队列，队列却是 0」。
		//   → 与前端「只读」一致：占领结算完成前一律拒绝，避免脏数据。
		var occ int64
		h.DB.Model(&model.EzfyOccupy{}).Where("city_id = ? AND status = 1", targetId).Count(&occ)
		if occ > 0 {
			return "该城市正在被占领中, 暂时无法掠夺/征服"
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
		// ★ 第九轮：携带资源必须有部队来装（负重决定能带多少）。
		//   ★ 2026-10-30 用户规则更新：运输同样必须带带队军官（见上面的 ezfyMustLeadOfficer）。
		if len(validTroops) == 0 {
			return "携带资源需要部队来装载(卡车负重最高)"
		}
		// ★ 2026-09-28 传 city.ID：负重上限含「标准化装卸」加成（与出征/采集同一口径）
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
	oilCost := h.ezfyOilCost(city, orderType, distance, validTroops, resources, h.officerOilReducePct(lead))
	// ★ 2026-10-17 人机免耗粮开启时，人机出征同样免油耗（city.IsBot=1，真人照常扣油）
	if ezfyBotFoodFree(city.IsBot) {
		oilCost = 0
	}
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
	// ★★ 2026-10-16 运输站口径（与 /order/preview 同口径，规则见 ezfyStationSpeedMult / ezfyStationOrderType）：
	//   只按**出发城**运输站等级算「速度倍数」（1级1.5倍 … 10级6.0倍），只对己方/盟友基地间
	//   的调动生效（运输5/增援·驻守盟友6/派遣8），且**不双向叠加**。
	station := blv[20]
	// ★★ 2026-10-09 行军速度按兵种算（与 /order/preview 共用 ezfyMarchSpeed）：
	//   机械增压(10)=通用、喷气推进(19)=只空军、军官速度技能=陆/空/海 —— 原来行军完全不吃科技速度，
	//   且军官速度技能不分兵种一律生效（与「有兵种限制」的配置口径不符）。
	marchSpeed := h.ezfyMarchSpeed(validTroops, tech, lead)
	if marchSpeed < 1 {
		marchSpeed = 300 // 无可出征部队（如纯运输资源）→ 沿用旧默认速度
	}
	travelSec := int64(distance) * 60 * 300 / int64(marchSpeed)
	// ★★ 2026-10-16 删除「无线电侦听(科技12)×2%/级」这一级（与 /order/preview 同口径）：
	//   无线电侦听只加军情情报等级，不加速行军；加速行军的是 机械增压(10)/喷气推进(19)，
	//   已折进上面的 ezfyMarchSpeed。
	if ezfyStationOrderType(orderType) {
		travelSec = int64(float64(travelSec) / ezfyStationSpeedMult(station))
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
	// ★ 2026-10-30 活动目标每日攻打上限：下单时固化本单是否按活动目标结算。
	//   达到上限后 ActType 落 0 → 到达时走普通野地/寇城战斗，不再发活动奖励。
	//   侦查(1)不占次数，但次数用完后也按普通野地回报情报；并发下单由到达时的 id 排名二次卡控。
	//   ★★ 名将活动目标（配了守将）**不卡这个上限**：名将只受「可抓次数 max_capture」约束
	//      （抓满后该坐标对该玩家变普通地形，见 playerOwnsActWildGeneral）。
	actType := 0
	if targetType != 3 && (orderType == 1 || orderType == 2 || orderType == 3) {
		if act := h.ezfyActTargetType(targetX, targetY); act > 0 &&
			!h.playerOwnsActWildGeneral(uid, targetX, targetY) &&
			(h.ezfyActNamedTargetAt(targetX, targetY) || !h.ezfyActDailyLimitReached(int64(city.ID))) {
			actType = act
		}
	}
	order := model.EzfyOrder{
		UserID: uid, CityId: int64(city.ID),
		OrderType: orderType, TargetType: targetType,
		ActType: actType,
		TargetX: targetX, TargetY: targetY, TargetId: targetId,
		Troops:  groupsJSON(validTroops),
		Officer: officer,
		// ★ 2026-10-24 命令同时记带队军官 id：归航/自愈/出征校验按 id 精确关联（同名军官互不误伤）
		OfficerId: uint(0),
		StartTime: now, ArriveTime: now + travelSec*1000,
		ReturnTime: now + travelSec*1000*2, Status: 0,
		OilUsed: oilCost, WaitMin: waitMin, AutoBattle: autoBattle,
	}
	if lead != nil {
		order.OfficerId = uint(lead.ID)
	}
	if resources != nil {
		b, _ := json.Marshal(resources)
		order.Resources = string(b)
	}

	// ★★ 2026-10-30 用户规则「一个军官不能同时出征多次」—— **CAS 抢占**（最终裁决）。
	//
	//	上面那段「Status=0 + 没有未结束命令」是**读校验**：连点、双开、
	//	同时往两个不同目标出兵时，两个请求会**同时**通过读校验 → 同一名军官带两支队伍出去
	//	（线上真实出现过：一个军官出现在两条出征队列里）。
	//	dupCnt 那条防抖只认「同城市+同类型+同目标」，换个目标就绕过了。
	//	→ 这里用一条带条件的 UPDATE 把「置出征态」变成**原子抢占**：
	//	  只有成功把 status 从 0 改成 1 的那个请求才算抢到该军官；
	//	  另一个 RowsAffected=0 → 直接拒绝。此时资源/兵/集结令/订单**都还没写库**
	//	  （city.Oil 只改了内存副本，saveCityRes 在下面才发），所以直接 return 是干净的。
	//	★ 代价：多一次串行写往返（跨 WAN ~120ms），但这是「同一军官只能出一队」的硬保证。
	if lead != nil {
		res := h.DB.Model(&model.EzfyOfficer{}).
			Where("id = ? AND status = 0", lead.ID).
			Update("status", 1)
		if res.Error != nil || res.RowsAffected == 0 {
			return "军官" + lead.Name + "出征中"
		}
	}

	// ★★ 2026-10-30 扣兵提前到**写库之前**，且改成「条件扣减、扣不到就中止」。
	//
	//	原来它在下面的并行块里、用 GREATEST 静默夹取 —— 于是「校验用快照 → 扣兵用实时值」
	//	之间的窗口内，只要兵力被别的请求拿走（第二支队伍出征、军工厂完工前后的一批
	//	并发结算），第二个请求扣到 0 也照样成功出发，凭空多出一队兵（见 deductCityTroops 注释）。
	//	现在扣不到就回滚 + 报错，玩家只会看到「兵力不足，请刷新后重试」，不会白拿一队兵。
	//	⚠️ 此时订单/资源都还没写库（city.Oil 只改了内存副本），失败直接 return 是干净的。
	if msg := h.deductCityTroops(city.ID, validTroops); msg != "" {
		// 军官 CAS 已抢到出征态 → 还回去，否则这名军官会被一直判「出征中」
		if lead != nil {
			h.DB.Model(&model.EzfyOfficer{}).Where("id = ? AND status = 1", lead.ID).Update("status", 0)
		}
		return msg
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
	// ★ 2026-10-09 这条 Create 原来**丢掉了 error**：一旦缺列（如 skip 共享库缺
	//   auto_battle，见 seed.EnsureEzfyOrderColumns）或字段超长，就表现为
	//   「接口报成功、兵和油已扣、出征队列里什么都没有」。现在捕获并吼出来。
	var orderErr error
	go func() { defer wwg.Done(); orderErr = h.DB.Create(&order).Error }()
	// ③ 扣兵：★ 2026-10-30 已提前到上面（条件扣减 + 失败中止），这里不再重复。
	// ④ 集结令：一次扣完（原来是每个道具一条 consumeItem）
	if gather > 0 {
		wwg.Add(1)
		go func() { defer wwg.Done(); h.consumeItemN(uid, ezfyGatherItemID(), gather, "出征集结令") }()
	}
	// ⑤ 带队军官置出征态：★ 2026-10-30 已提前到上面的 **CAS 抢占**（并发安全），
	//   这里不再重复写一次（否则又多一次跨 WAN 写往返）。
	// ⑥ 雷达站事前预警（自己的读 + 一条战报写，也一起并行）
	wwg.Add(1)
	go func() {
		defer wwg.Done()
		h.ezfyOrderRadarWarn(city, &order, orderType, targetType, targetId, officer, validTroops)
	}()
	wwg.Wait()
	// ★ 2026-10-09 订单写入失败必须让玩家知道（原来静默：兵/油已扣、队列里却没有）。
	//   根因通常是「库上缺列」—— 多机共享库走 seed.skip 时不跑 AutoMigrate，
	//   新加的列要由 seed.EnsureEzfyOrderColumns 等幂等补列负责（见 main.go skip 分支）。
	if orderErr != nil {
		// ★ 2026-10-30 订单没写成功 → 把刚抢占到的出征态**还回去**，
		//   否则这名军官会被判「出征中」一直锁着（只能等下次军官列表自愈）。
		if lead != nil {
			h.DB.Model(&model.EzfyOfficer{}).Where("id = ? AND status = 1", lead.ID).Update("status", 0)
		}
		// ★ 2026-10-30 扣兵已经提前执行 → 订单没落库就必须把兵**原样退回**，
		//   否则玩家会「兵没了、出征队列里也没有」（原来的静默丢兵事故）。
		h.restoreTroops(city.ID, validTroops)
		log.Printf("【严重】出征订单写入失败(uid=%d city=%d orderType=%d): %v —— 检查该库是否缺列",
			uid, city.ID, orderType, orderErr)
		return "出征命令写入失败, 请稍后重试或联系管理员"
	}
	return ""
}

// ezfyOrderRadarWarn 出征时给**被攻击方**发事前预警（雷达站 + 无线电侦听决定能看到多少）。
//
// ★ 2026-10-05 从 createOrder 里抽出来：① 让收尾的 6 个写能并行；
// ② 本段自己的读（目标城 + 情报等级）也不再串在写前面。
//
//	侦查(1)      → 「被侦查报告」（军情警讯）
//	掠夺(2)/征服(3) → 「军情警报: 敌军来袭!」，细节随情报等级递增
//
// ★ 2026-09-25 「军情警讯里面展示下对面城市名字以及地址，雷达站以及科技满足的情况下展示，
// 不然我只知道有人打我，不知道哪来的」：情报等级 = 雷达站等级 + **无线电侦听科技等级**（合计封顶 10）。
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
	//	情报等级 = 雷达站等级 + **无线电侦听科技等级**（合计封顶 10），见 h.ezfyIntelLevel。
	//	「出发城市（名称+坐标）」门槛定在 **2 级**（雷达站1级 + 无线电侦听1级就能看到），
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
				// ★ 2026-10-14 被侦查报告：先亮出我方哪座城被侦查（自己的城无需情报等级）——
				//   玩家最想知道的就是「哪座城、从哪来」。
				body += fmt.Sprintf("我方城市: %s(%d,%d)\n", target.Name, target.X, target.Y)
				if radar >= 3 {
					body += fmt.Sprintf("侦查方城市: %s(%d,%d)\n", city.Name, city.X, city.Y)
				}
				if radar >= 5 {
					body += fmt.Sprintf("侦查时间: %s\n", time.UnixMilli(order.StartTime).Format("01-02 15:04"))
				}
				body += fmt.Sprintf("(情报等级%d: 雷达站等级越高、无线电侦听越高, 情报越详细)", radar)
				// ★ 2026-10-14 「只有名字看不出来，其他玩家有重名的，加坐标辨别」：
				//   标题里带上**侦查方城市坐标**，重名城一眼可分（体例与「被掠夺报告: X(286,294)」一致）。
				h.addReport(target.UserID, 6, fmt.Sprintf("被侦查报告: %s(%d,%d)", city.Name, city.X, city.Y), body, "", 0, target.ID)
			}
			if radar >= 1 && orderType != 1 {
				// ★ 出发城市（名称+坐标）放正文**最前面** —— 玩家最想知道的就是「谁、从哪来」
				originLine := ""
				if radar >= 2 {
					originLine = fmt.Sprintf("出发城市: %s(%d,%d)\n", city.Name, city.X, city.Y)
				}
				warn := "军情警报: 敌方部队正向我方城市进发!\n" + originLine
				// ★ 2026-10-18 「预警战报加个目标城市」：多城玩家一眼看清敌人要打哪座城。
				//   目标城市是**自己的城**，不依赖情报等级、无条件显示
				//   （体例与「被侦查报告」的「我方城市」一致）。
				warn += fmt.Sprintf("目标城市: %s(%d,%d)\n", target.Name, target.X, target.Y)
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
					warn += fmt.Sprintf("(情报等级%d 不足: 雷达站+无线电侦听 合计到 2 级才能看到来袭城市与坐标)\n", radar)
				}
				// ★★ 2026-10-31 情报等级来源拆解里带上「市长技能」那一截（有才写，没市长技能格式不变）
				intelSrc := fmt.Sprintf("雷达站%d级 + 无线电侦听%d级",
					h.buildingLevel(target.ID, 21), h.techMap(target.ID)[ezfyReconTechID])
				if mv := h.ezfyMayorReconLevel(target.ID); mv > 0 {
					intelSrc += fmt.Sprintf(" + 市长黄金洞察%d级", mv)
				}
				warn += fmt.Sprintf("(情报等级%d = %s)", radar, intelSrc)
				// ★ 2026-10-31 用户「军情警讯列表简化」：
				//   `军情警报: 敌军来袭! 来自 X(x,y)` → `军情: 敌军来袭 X(x,y)`
				//   （去掉「警报/!/来自」冗余，前缀统一成「军情: 」，列表更短更好扫）。
				//   ★ 前缀必须仍能被 `HasPrefix(title,"军情")` 命中（分类/标签/角标都靠它），
				//   老数据「军情警报: …」同样命中，不需要回填。
				title := "军情: 敌军来袭"
				if originLine != "" {
					// 列表只显示标题 → 把来源也带上，不点进去就能看到
					title += fmt.Sprintf(" %s(%d,%d)", city.Name, city.X, city.Y)
				}
				h.addReport(target.UserID, 6, title, warn, "", 0, target.ID)
			}
		}
	}

	// ★★ 2026-10-19 附属野地被侦查 → 野地主人收「被侦查报告」（军情警讯）。
	//   野地没有雷达站，也没有「城市」这一层过滤，直接给主人（自己的地被摸了必须知道）；
	//   标题带上侦查方城市名+坐标（重名城一眼可分，与城市被侦查同体例）。
	//   ★ 事后报告（侦查成功/失败）仍只给攻方，守方这条是**预警**（出征即发）。
	if (targetType == 1 || targetType == 2) && orderType == 1 {
		if ownerUID, ownerCityID, wl := h.ezfyWildlandOwnerInfo(order.TargetX, order.TargetY, city.UserID); ownerUID > 0 {
			// 「我方野地」写全**名字+等级+坐标**（与城市那条的「我方城市: 城名(x,y)」同体例，
			// 也与战斗中守方战报里的野地称谓一致），否则光看坐标分不清是哪块地。
			wlName := ezfyWildTerrainDisplayName(wl.X, wl.Y, true) + strconv.Itoa(wl.Level) + "级"
			body := fmt.Sprintf("有敌军对我方附属野地进行了侦查!\n我方野地: %s(%d,%d)\n侦查方城市: %s(%d,%d)\n侦查时间: %s\n",
				wlName, order.TargetX, order.TargetY, city.Name, city.X, city.Y,
				time.UnixMilli(order.StartTime).Format("01-02 15:04"))
			h.addReport(ownerUID, 6, fmt.Sprintf("被侦查报告: %s(%d,%d)", city.Name, city.X, city.Y),
				body, "", 0, ownerCityID)
		}
	}
}

// deductCityTroops 出征后从城市扣兵。返回非空字符串 = 扣兵失败（调用方必须中止并回滚）。
//
// ★ 2026-10-05 性能：原来每个兵种「SELECT 行 + UPDATE/DELETE」= 2N 条跨 WAN 往返；
// 现在每个兵种**一条**条件 UPDATE（相对扣减）+ 收尾一条 DELETE（清掉归零行）。
//
// ★★ 2026-10-30 线上事故（玩家反馈「派两队各 20w 火箭打活动寇，城里军工厂刚好完工，
// 结果凭空卡出 20w 火箭」）：
//
//	原实现用 `GREATEST(count - ?, 0)` **静默夹取** —— 扣不够就当扣满了。
//	而上面的兵力校验用的是**懒结算快照**（cityTroops），校验和扣兵之间隔着军官 CAS、
//	耗油计算等若干次跨 WAN 往返。两个出征请求（连点 / 双开 / 同时打两个不同目标，
//	dupCnt 防抖只认「同目标」所以绕得过去）会**同时**通过快照校验，
//	第二个请求扣兵时库里已经是 0 → GREATEST 夹成 0 → 不报错、部队照样出发
//	= 凭空多出一整队兵。
//
//	现在改成**条件扣减**：`WHERE count >= ?`，RowsAffected=0 就说明兵力已被
//	并发操作拿走 → 回滚已扣的兵种并返回错误，绝不允许「没扣到兵也出征」。
//	⚠️ 不要再退回 GREATEST 静默夹取：那是本 bug 的根因。
func (h *EzfyHandler) deductCityTroops(cityId uint, troops []ezfyUnitGroup) string {
	if len(troops) == 0 {
		return ""
	}
	done := make([]ezfyUnitGroup, 0, len(troops))
	ids := make([]int, 0, len(troops))
	for _, t := range troops {
		if t.Count <= 0 {
			continue
		}
		ids = append(ids, t.TroopId)
		res := h.DB.Model(&model.EzfyCityTroop{}).
			Where("city_id = ? AND troop_id = ? AND count >= ?", cityId, t.TroopId, t.Count).
			Update("count", gorm.Expr("ezfy_city_troop.count - ?", t.Count))
		if res.Error != nil {
			h.restoreTroops(cityId, done)
			log.Printf("[ezfy] 扣兵失败 city=%d troop=%d count=%d err=%v", cityId, t.TroopId, t.Count, res.Error)
			return "出征失败(扣兵异常), 请稍后重试"
		}
		if res.RowsAffected == 0 {
			// 兵力已被并发操作扣走（另一支队伍出征 / 战损结算）→ 整体回滚
			h.restoreTroops(cityId, done)
			name := "兵种" + strconv.Itoa(t.TroopId)
			if cfg := ezfyCfg.troop(t.TroopId); cfg != nil {
				name = cfg.Name
			}
			return fmt.Sprintf("兵力不足: 「%s」已被其他操作占用, 请刷新后重试", name)
		}
		done = append(done, t)
	}
	if len(ids) == 0 {
		return ""
	}
	h.DB.Where("city_id = ? AND troop_id IN ? AND count <= 0", cityId, ids).
		Delete(&model.EzfyCityTroop{})
	return ""
}

// restoreTroops 把已经扣掉的兵原样加回去（扣兵中途失败 / 订单写库失败时的回滚）。
func (h *EzfyHandler) restoreTroops(cityId uint, troops []ezfyUnitGroup) {
	for _, t := range troops {
		if t.Count > 0 {
			h.addTroop(cityId, t.TroopId, t.Count)
		}
	}
}

// 雷达站预警相关常量
const (
	ezfyRadarBuildingID   = 21 // 建筑：雷达站（「对敌军入侵进行预警」）
	ezfyReconTechID       = 12 // 科技：无线电侦听（原来只加行军速度，现在同时加成军情情报）
	ezfyIntelMaxLevel     = 10 // 情报等级封顶
	ezfyReconPlaneTroopID = 9  // ★ 2026-10-14 侦察机兵种 id（侦查只能带它；被侦查城有它且未设「不参与防御」时升空拦截空战）
)

// ezfyIntelLevel 被攻击方城市的「军情情报等级」= 雷达站等级 + 无线电侦听等级 + 市长技能（合计封顶 10）。
//
// ★ 2026-09-25 「军情警讯里面展示下对面城市名字以及地址，雷达站以及科技满足的情况下
//
//	展示，不然我只知道有人打我，不知道哪来的」。
//
// 规则：
//   - **雷达站（建筑 21）= 主渠道**：0 级收不到任何事前预警（保持原规则不变）
//   - **无线电侦听（科技 12）= 加成**：每 1 级让情报等级 +1 —— 名字本来就叫「侦察」，
//     原来只加行军速度，现在顺带加成情报，科技终于有用了
//   - **黄金洞察（军官技能）= 加成**：★ 2026-10-31 用户口径「当市长的时候可以提高情报等级，
//     预警更清晰」→ 本城**市长**带该技能时，按技能等级叠加（封顶 10），见 ezfyMayorReconLevel
//   - **「出发城市（名称+坐标）」门槛 = 2 级** → 雷达站 1 级 + 无线电侦听 1 级 就能看到；
//     只有雷达站 1 级且没升科技时，正文会明确提示「合计到 2 级才能看到」
//
// 逐级解锁：2=出发城市+意图 / 3=预计到达 / 4=统帅 / 5=出发时间 / 7=兵力(模糊) / 9=兵力(精确)
func (h *EzfyHandler) ezfyIntelLevel(cityID uint) int {
	radar := h.buildingLevel(cityID, ezfyRadarBuildingID)
	if radar <= 0 {
		return 0
	}
	lv := radar + h.techMap(cityID)[ezfyReconTechID] + h.ezfyMayorReconLevel(cityID)
	if lv > ezfyIntelMaxLevel {
		lv = ezfyIntelMaxLevel
	}
	return lv
}

// ezfyMayorReconLevel 本城**市长**带「黄金洞察」时给的情报等级加成（= 技能等级；无市长/没学 = 0）。
//
// ★ 2026-10-31 用户口径：「当市长的时候可以提高情报等级，预警更清晰」。
//
//	与「出征侦查时提高侦察等级」是同一个技能的两面：
//	  · 带队出征侦查 → 加**侦察等级**（看清侦查报告，见 scoutReportBody）
//	  · 在城里当市长   → 加**情报等级**（军情预警更详细，见 ezfyIntelLevel）
//
// 性能：只在「出征预警」和「军情页」各调一次（不在懒结算/循环里），1 条按主键索引的查询。
func (h *EzfyHandler) ezfyMayorReconLevel(cityID uint) int {
	if cityID == 0 {
		return 0
	}
	var o model.EzfyOfficer
	if err := h.DB.Where("city_id = ? AND position = ? AND is_captive = 0", cityID, ezfyPositionMayor).
		First(&o).Error; err != nil {
		return 0
	}
	return h.ezfyReconSkillLevel(&o)
}

// ezfyReconSkillLevel 「黄金洞察」给侦察/情报等级的加成（= 技能等级；没学 = 0）。
//
// ★ 2026-10-31 唯一实现：侦查报告（侦察等级）与市长情报等级都走它，别再各写一份。
func (h *EzfyHandler) ezfyReconSkillLevel(o *model.EzfyOfficer) int {
	if o == nil || !h.officerHasSkill(o, "黄金洞察") {
		return 0
	}
	return h.officerSkillLevel(o)
}

// ezfyOrderTimeLabel 出征队列 / 军队动态里「时间标签 + 文案」的统一口径。
//
// ★★ 2026-10-09 新增：出征队列（`/orders`）原来**完全不发** time_label/time_text，
// 而前端模板是 `{{ o.time_label }}：{{ ... }}` —— 遇到「等待中 / 驻守空闲」这类
// **没有倒计时**的状态，`_lt` 也拿不到值 → 整行就渲染成一个孤零零的冒号「：」
// （用户反馈「看都不知道军队在干啥」）。
//
// 返回 (label, text)；text 为空 = 交给前端按实时倒计时算（行军中/返航中/战斗中有倒计时）。
func (h *EzfyHandler) ezfyOrderTimeLabel(o *model.EzfyOrder, now int64) (string, string) {
	switch o.Status {
	case 0:
		return "到达剩余", ezfyDurationText((o.ArriveTime - now) / 1000)
	case 2:
		return "返航剩余", ezfyDurationText((o.ReturnTime - now) / 1000)
	case ezfyOrderStatusBattle:
		// 本回合倒计时由前端按 battle_left_ms 每秒重算（后端只给标签）
		return "本回合剩余", ""
	case ezfyOrderStatusWaiting:
		// ★ 等待中原来在军队动态里也是空（switch 没这个 case），这里统一口径
		return "已等待", ezfyDurationText((now - o.ArriveTime) / 1000)
	case 1:
		if o.OrderType == 7 {
			if o.ArriveTime > 0 {
				return "驻守采集", ""
			}
			return "驻守(空闲)", ""
		}
		return "驻守中", ""
	}
	return "", ""
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
	// ★ 2026-10-09 时间标签统一口径（见 ezfyOrderTimeLabel）
	nowMs := time.Now().UnixMilli()
	for _, o := range orders {
		tLabel, tText := h.ezfyOrderTimeLabel(&o, nowMs)
		// ★ 2026-10-30 出征队列同样按订单固化的 ActType 展示「活动」标识
		actType := 0
		if o.OrderType == 2 || o.OrderType == 3 {
			actType = h.ezfyOrderActType(uid, &o)
		}
		views = append(views, gin.H{
			"id": o.ID, "order_type": o.OrderType, "type_name": ezfyOrderTypeName(o.OrderType),
			"act_type":    actType,
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
			"battle_max":   ezfyBattleMaxRoundsCfg(),
			// ★ 2026-09-30 行军计谋使用标记：scheme_fast=神兵天降已用  scheme_back=战略转移已用
			"scheme_fast": o.SchemeUsed & 1,
			"scheme_back": (o.SchemeUsed >> 1) & 1,
			// ★ 2026-10-09 时间标签/文案（原来完全不发 → 等待中/驻守空闲渲染成孤零零的「：」）
			"time_label": tLabel, "time_text": tText,
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

// ezfyWildDefenderOrderCond 附属野地「在场守军」订单的 SQL 条件（★口径唯一实现，改这里就够）。
//
// ★★ 2026-10-25 用户反馈「全部召回的卡车实际没回来；野地有卡车、大号看不到、小号征服能发现」：
//
//	根因 = finishReturn 返航归队时**只删 order_type=6** 的订单，采集队(7) 残留 status=3；
//	而各处守军口径写成 `order_type IN (6,7) AND status IN (1,3)`，把这支**已经回城**的
//	采集队继续当成野地守军 → 野地永远挂着一支「打不死、召不回、驻军列表也看不到」的幽灵。
//
//	现在：采集队(7) 只有 status=1（真的在野地驻守/采集）才算守军；
//	      增援野地驻军(6) 按 status 1/3 都算（宁可多算，别让真守军漏掉被打）。
const ezfyWildDefenderOrderCond = "((order_type = 7 AND status = 1) OR (order_type = 6 AND status IN (1,3)))"

// ezfyWildlandOrderCanDefend 该订单是否是「附属野地上的驻军/采集队」——
// 口径必须与 ezfyWildDefenderOrderCond 完全一致：
//
//	6(增援驻军) 到达后常驻(status 3)，在途/异常态 1 也算；
//	7(驻守采集) 只有 status 1（真在野地）才算 —— 3 是「已返航归队」的僵尸单，不能再当守军。
func ezfyWildlandOrderCanDefend(order *model.EzfyOrder) bool {
	if order == nil || (order.TargetType != 1 && order.TargetType != 2) {
		return false
	}
	switch order.OrderType {
	case 7:
		return order.Status == 1
	case 6:
		return order.Status == 1 || order.Status == 3
	}
	return false
}

// ezfyWildlandBattleCoords 一次查出当前玩家可能参与防守的野地战场坐标。
//
// ★★ 2026-10-20 用户反馈：附属野地被征服/掠夺时，守方部队仍能在「驻军」里点[召回]，
// 战斗中的兵力被抽走会与战场快照/战后兵力回写打架（重复入兵/丢兵）。
// 战场订单属于攻方，且 target_id 可能为 0，所以按野地坐标匹配正在进行的战场。
// 一次查全（而不是每支部队一条 COUNT），避免 1 核服务器上的 N+1。
func (h *EzfyHandler) ezfyWildlandBattleCoords(uid uint) map[[2]int]bool {
	var battles []model.EzfyBattle
	h.DB.Select("target_x, target_y").
		Where("status = 1 AND target_type IN (1,2) AND user_id <> ? "+
			"AND order_id IN (SELECT id FROM ezfy_order WHERE status = ?)",
			uid, ezfyOrderStatusBattle).
		Find(&battles)
	m := make(map[[2]int]bool, len(battles))
	for _, b := range battles {
		m[[2]int{b.TargetX, b.TargetY}] = true
	}
	return m
}

// ezfyWildlandOrderInBattle 判断单支附属野地驻军/采集队是否正在被敌军攻打。
func (h *EzfyHandler) ezfyWildlandOrderInBattle(uid uint, order *model.EzfyOrder) bool {
	if !ezfyWildlandOrderCanDefend(order) {
		return false
	}
	return h.ezfyWildlandBattleCoords(uid)[[2]int{order.TargetX, order.TargetY}]
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
	// ★★ 2026-10-20 附属野地驻军/采集队作为守军参战时，订单本身仍是 status=1/3，
	//   不能只靠上面的 status==5 判断；否则战斗中点[召回]会把参战兵力抽走。
	if h.ezfyWildlandOrderInBattle(uid, &order) {
		resp.ParamError(c, "部队正在附属野地战斗中, 不能召回；请先到「军情 → 军队动态 → [指挥]」结束战斗")
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
		if order.TargetType != 3 {
			// ★★ 2026-10-18 附属野地驻军（增援野地）允许单独召回, 按单程返航回出发城市
			var wl model.EzfyWildland
			if err := h.DB.First(&wl, order.TargetId).Error; err != nil || wl.CityId != order.CityId {
				resp.ParamError(c, "该野地驻军不存在, 无法召回")
				return
			}
		} else {
			var tcity model.EzfyCity
			if err := h.DB.First(&tcity, order.TargetId).Error; err != nil || tcity.UserID == uid {
				resp.ParamError(c, "该部队不在盟友城市, 无法召回")
				return
			}
		}
	} else if order.Status != 0 && order.Status != 1 && order.Status != ezfyOrderStatusWaiting {
		// ★ 2026-10-09 「等待中的玩家可以自己取消」→ 等待(6) 也放行
		//   （部队已到达、只是在排队等交战，取消后按单程返航回出发城）。
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
	// ★★ 2026-10-11 随军资源(order.Resources)**不再在这里并入 Carry** ——
	//   统一由 finishReturn 在「返航到达入库」时并入（一处兜住取消/异常返航/战斗返航等所有路径）。
	//   这里若也并一次，会与 finishReturn 重复计入 → 资源翻倍。
	carry := order.Carry
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
	// ★ 2026-10-13 采集宝物播报防刷屏：本轮批量结算（多支采集队到点）掉落的宝物
	//   攒进 lootBatch，循环结束统一 flush 成一条合并播报（见 ezfyLootBatch）。
	// ★ 2026-10-22 每支队伍只攒「最近一期」的宝物，期数累加到 periods（播报注明共结算N期）。
	var lootBatch *ezfyLootBatch
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
			if lootBatch == nil {
				lootBatch = &ezfyLootBatch{}
			}
			h.settleDispatch(uid, order, now, nil, lootBatch)
		} else if order.Status == 2 && now >= order.ReturnTime {
			h.finishReturn(uid, order)
		}
	}
	// ★ 2026-10-13 采集宝物播报防刷屏：本轮批量结算的宝物统一合并成一条播报后发出。
	h.ezfyLootBatchFlush(uid, lootBatch)
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

// processIncoming 把「正在攻打 uid 名下城市/附属野地、已到点」的敌方订单结算掉（开战场 / 发军情警讯）。
// 处理 target_type=3（玩家城）与 target_type IN (1,2)（有玩家驻军的附属野地）且 status=0（行进中，到点）的订单，
// 交给 processArrive 走统一流程。
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
	// ★★ 2026-10-19 「有玩家驻军的附属野地」的守方也是真人：把「正在打我方附属野地」
	//   的到达订单也纳入来袭，让守方自己的轮询就能触发开战场/军情警报（原来只认 target_type=3 玩家城）。
	// ★★ 2026-10-20 攻打野地的订单（掠夺2/征服3）客户端只传目标类型+坐标，target_id=0，
	//   原来按 `w.id = ezfy_order.target_id` JOIN 永远匹配不到 → 守方轮询推不动战场。
	//   改成按坐标 EXISTS 映射野地归属城（同时兼容 target_id 有值的驻军/采集订单）。
	var orders []model.EzfyOrder
	h.DB.Table("ezfy_order").
		Select("ezfy_order.*"). // ★ 只取订单列，避免野地表同名列(id/x/y/city_id)在 SELECT * 时覆盖订单字段
		Where("ezfy_order.status = 0 AND ezfy_order.arrive_time <= ?", now).
		Where("((ezfy_order.target_type = 3 AND ezfy_order.target_id IN ?) OR "+
			"(ezfy_order.target_type IN (1,2) AND EXISTS ("+
			"SELECT 1 FROM ezfy_wildland w WHERE w.x = ezfy_order.target_x "+
			"AND w.y = ezfy_order.target_y AND w.city_id IN ?)))", ids, ids).
		Order("ezfy_order.id ASC").Find(&orders)
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
	// ★ 2026-10-07 修复「打完自己兵力会翻倍」：返航归队（战斗剩余兵力 addTroop 回城）原先
	//   没有并发守卫。两实例/重复触发的 processOrders 同时判定「status=2 且已到 return_time」
	//   时，各自执行一次 finishReturn → 剩余兵力被加进城市 **两次** = 玩家看到兵力翻倍。
	//   现在先原子抢占 status 2→3：谁抢到（RowsAffected=1）谁归还，抢不到的整段跳过。
	if res := h.DB.Model(&model.EzfyOrder{}).Where("id = ? AND status = 2", order.ID).
		Update("status", 3); res.RowsAffected != 1 {
		return
	}
	left := parseGroups(order.Result)
	city := h.cityOfOrder(order, uid)
	for _, g := range left {
		if g.Count > 0 {
			h.addTroop(city.ID, g.TroopId, g.Count)
		}
	} // 部队带回的采集资源在这里入城
	// ★ 2026-09-24 用户反馈「运输/采集资源变少」：同上，把整行写回改成 DB 原子累加，
	//   不覆盖这期间其它写入的增量。
	//   ★ 2026-09-24 规则修正（用户确认原版口径）：入城资源不受仓储上限截断，
	//   只有超过数据库字段最大值才会溢出——去掉 LEAST(cap, ...)，改为无条件累加。
	c := parseCarry(order.Carry)
	// ★★ 2026-10-11 修复「派遣携带黄金，取消/返航后黄金不见了」：
	//   随军资源(order.Resources)在**出发时就从出发城扣掉了**（见 createOrder），此后只有两种归宿：
	//     ① 送达目标城 —— 运输/派遣/增援**到达**时入目标城，并把 order.resources 清空；
	//     ② 随部队返航入城 —— 就是这里。
	//   原来只有「召回」那一条路径把 Resources 并进了 Carry，其余所有返航
	//   （取消派遣/异常返航/侦查返航/战斗结束返航/放弃野地/遣返驻军…）都只带 Carry，
	//   Resources 里的粮/钢/油/稀/金**凭空消失**。现在统一在这里并入，一处兜住所有返航。
	if strings.TrimSpace(order.Resources) != "" {
		r := parseCarry(order.Resources)
		c.Food += r.Food
		c.Steel += r.Steel
		c.Oil += r.Oil
		c.Rare += r.Rare
		c.Gold += r.Gold
	}
	if c.total() > 0 {
		// ★ 2026-09-25 「各项资源有最大的配置」→ 入库统一走 ezfyResAddExpr：
		//   无条件累加，累加到配置的「资源最大值」（默认 100 亿）为止，且不拉低已有更大值。
		// ★★ 2026-10-17 用户口径：返航带回出发城同样按「资源最大值」封顶累加
		//   （上限读二战系统配置，不写死），装不下的部分**系统回收**；
		//   战报如实显示实际入库量与回收量。
		// ★★ 2026-10-22 修复同款吞资源：必须先懒结算出发城，再算 fit。
		//   否则待落库产量会在返航入库之后把容量吃掉，而 fit 按旧快照算出的超出量不会退回。
		fit, rec := h.ezfyDeliverCarryToCity(city, c)
		// ★ 2026-10-11 文案改中性：本函数现在承接**所有**出征类型的返航资源入城
		//   （不止采集），别再说「采集部队」。标题前缀「部队返航」被战报分类依赖，保持不变。
		h.addReport(uid, 5, "部队返航: 资源已入库",
			fmt.Sprintf("部队返回%s\n带回: 粮%d 钢%d 油%d 稀矿%d 金%d", city.Name,
				fit.Food, fit.Steel, fit.Oil, fit.Rare, fit.Gold)+
				func() string {
					if rec.total() > 0 {
						return fmt.Sprintf("\n超出上限被系统回收: 粮%d 钢%d 油%d 稀矿%d 金%d",
							rec.Food, rec.Steel, rec.Oil, rec.Rare, rec.Gold)
					}
					return ""
				}(), "", order.ID)
	}
	// 带队军官归来, 恢复在职
	// ★ 2026-10-24 按命令里的军官 id 精确解锁（同名军官互不误伤）；老数据无 id 按名字兜底
	if order.Officer != "" {
		if order.OfficerId > 0 {
			h.officerGoOutByID(order.OfficerId, false)
		} else {
			h.officerGoOut(city, order.Officer, false)
		}
	}
	order.Status = 3
	order.Carry = ""
	h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
		Updates(map[string]interface{}{"status": 3, "carry": ""})
	// ★ 2026-10-02 驻军(增援到盟友城)返航归队后订单已完结：兵力/军官已入城,
	//   直接删除订单 —— 否则 status=3 常驻残留会被军情误判为「驻守中」、
	//   且一直占用盟友城驻军槽位, 导致新驻军无法增援。
	// ★★ 2026-10-19 扩到**所有增援(6)**：附属野地驻军召回后同样要删，
	//   否则残留的 status=3 订单会让 officerList 一直认为军官还在外（军官永远回不来）。
	// ★★ 2026-10-25 再扩到**驻守采集(7)**（用户反馈「召回的卡车实际没回来」）：
	//   采集队返航归队后残留 status=3，会被野地守军口径当成守军 → 野地上永远挂着
	//   一支「打不死、召不回、驻军列表也看不到」的幽灵部队，玩家以为召回失败。
	if order.OrderType == 6 || order.OrderType == 7 {
		h.DB.Delete(&model.EzfyOrder{}, order.ID)
	}
}

// beginReturn 异常返航: 兵力无损带回
// ezfyOilCost 出征耗油(运输按携带资源量计, 其余按兵种油耗×数量×距离计)
//
// ★ 「加个出征油耗开关，默认开；关了出征消耗油 0」→ 关掉时直接返回 0。
// 出征预览(/order/preview)与真正下单(createOrder)都走这里，所以「看到的 0」就是「实扣的 0」。
//
// ★★ 2026-10-09 带队军官「机械改装」技能（★ 2026-10-15 回收率降为 +5%/级，出征油耗维持 -10%/级）：
//
//	oilReducePct = 该军官的油耗减免百分点（0 = 无）。减免在**算出原价之后**统一打折，
//	预览与实际下单传同一个值 → 不会出现「预览便宜、实扣更贵」。最低保底 1 油。
func (h *EzfyHandler) ezfyOilCost(city *model.EzfyCity, orderType, distance int,
	troops []ezfyUnitGroup, resources map[string]int64, oilReducePct int) int64 {
	if !ezfyMarchOilOn() {
		return 0
	}
	var cost int64
	if orderType == 5 {
		f, s, o, r, g := resources["food"], resources["steel"], resources["oil"], resources["rare"], resources["gold"]
		cost = maxInt64(1, (f+s+o+r+g)/10000+int64(distance)/50)
	} else {
		var oilUnitTotal int64
		for _, t := range troops {
			if cfg := ezfyCfg.troop(t.TroopId); cfg != nil {
				oilUnitTotal += int64(cfg.OilKeep) * t.Count
			}
		}
		cost = maxInt64(1, oilUnitTotal*int64(distance)/ezfyOilDivGrid)
	}
	if oilReducePct > 0 {
		// 兜底：减免最多 90%（留 10% 保底，避免配置/数据异常把油耗抹成 0）
		if oilReducePct > 90 {
			oilReducePct = 90
		}
		cost = cost * int64(100-oilReducePct) / 100
		if cost < 1 {
			cost = 1
		}
	}
	return cost
}

// officerOilReducePct 带队军官「机械改装」的出征油耗减免%（10% × 技能等级；没学 = 0）。
//
// 与「回收率（伤兵恢复）+5%×等级」同源（★ 2026-10-15 回收率 10%/级 → 5%/级，见 ezfy_order.go 战后伤兵入营 / activity_target.go）。
func (h *EzfyHandler) officerOilReducePct(o *model.EzfyOfficer) int {
	if o != nil && h.officerHasSkill(o, "机械改装") {
		return 10 * h.officerSkillScale(o)
	}
	return 0
}

// ============ 运输站（建筑 20）：己方/盟友基地间的「行动速度倍数」============
//
// ★★ 2026-10-16 玩家口径重做（对齐 cfg 文案「部队行动速度提高1.5~6.0倍(己方/盟友基地间)」）：
//   - 口径是**倍数**，不是百分比：1 级 1.5 倍 … 10 级 6.0 倍（每级 +0.5 倍）；
//   - 只按**出发城**运输站等级算，**不**把目的地城的等级加进来（6倍×6倍会太夸张）；
//   - 只对「己方/盟友基地间」的调动生效（见 ezfyStationOrderType）。
//
// 行军时间 = 原时间 ÷ 倍数；等级 0（没建）→ 1.0 倍（不减时）。
// ★ /order/preview 与 createOrder 共用这两个函数 —— 规则只在这一处实现。

// ezfyStationSpeedMult 运输站等级 → 行动速度倍数（原版表：1级1.5倍 … 10级6.0倍）。
func ezfyStationSpeedMult(level int) float64 {
	if level <= 0 {
		return 1
	}
	if level > 10 {
		level = 10 // 运输站上限 10 级，防配置越界
	}
	return 1 + 0.5*float64(level)
}

// ezfyStationOrderType 运输站是否作用于该订单类型。
//
// 原版文案写的是「己方/盟友基地间」—— 所以只有这三类吃到：
//
//	运输(5) 己方/同盟城市互运、增援(6) 驻守盟友城市、派遣(8) 把自己的部队送去自己的另一座城。
//
// 侦查(1)/掠夺(2)/征服(3)/采集(4)/驻守采集(7) 等战斗与野外行动**不吃**运输站加成
// （否则 10 级 6 倍速会让打仗几乎瞬间到位）。
func ezfyStationOrderType(orderType int) bool {
	return orderType == 5 || orderType == 6 || orderType == 8
}

// ezfyMarchSpeed 全军**行军速度**（= 最慢那支部队的「有效速度」，慢的拖后腿）。
//
// 有效速度 = 兵种基础速度 × (1 + 通用速度加成 + **该兵种专属**速度加成)
//
//	· 通用：机械增压(10)「部队速度+5%/级，满级 50%」→ 全体兵种
//	· 兵种专属：喷气推进(19)「空军速度+5%/级，满级 50%」→ 只加空军
//
// ★★ 2026-10-31 军官技能「坦克冲锋/闪电疾攻/越岛登陆」已由**移动速度**改为**攻速**
// （只决定战斗内出手先后，见 ezfyTypeBonus.AtkSpeed）→ **不再**加速行军。
// 行军速度现在只吃科技（机械增压 / 喷气推进）。
//
// ★★ 2026-10-09 用户确认「行军也按兵种接入科技速度」（原来行军完全不吃科技速度，
// 且军官速度技能无论部队兵种一律 +10% —— 与「有兵种限制」的配置口径不符）：
//
//	现在按每支部队各自的兵种算有效速度再取最慢，喷气推进只在部队含空军时才会真正
//	缩短行军时间，越岛登陆带队纯陆军也不再凭空加速。
//
// 出征预览(/order/preview)与真正下单(createOrder)共用本函数，保证「看到的耗时」=「实际的耗时」。
// 没有任何可出征部队（如纯运输资源）→ 返回 0，调用方回落到旧默认速度 300。
func (h *EzfyHandler) ezfyMarchSpeed(troops []ezfyUnitGroup, techMap map[int]int, lead *model.EzfyOfficer) int {
	// ★ 2026-10-31 机械增压/喷气推进 满级改 50% → 不再写死每级 2%/3%，
	//   改为读配置（ezfy_cfg_tech_level.effect_value，缺省按 ezfyTechEffectPer 每级 5%）。
	//   这样管理端「科技管理 → 等级配置 → 效果数值」一改，行军与战斗速度立刻同源生效。
	gen := ezfyTechEffectValueAt(10, techMap[10]) // 机械增压：部队速度（通用）
	tb := ezfyTypeBonus{Speed: map[int]int{}}
	if v := ezfyTechEffectValueAt(19, techMap[19]); v > 0 { // 喷气推进：只加空军
		tb.Speed[ezfyTroopTypeAir] += v
	}
	tb = h.officerTypeBonus(lead, tb) // 军官技能：★ 2026-10-31 起三个速度技能改攻速 → 这里已无速度来源
	slowest := 0
	for _, t := range troops {
		if t.Count <= 0 {
			continue
		}
		cfg := ezfyCfg.troop(t.TroopId)
		if cfg == nil || cfg.Type == ezfyTroopTypeCity || cfg.Speed <= 0 {
			continue
		}
		// ★★ 2026-10-31 坦克冲锋/闪电疾攻/越岛登陆 已改为**攻速**（只影响战斗内出手先后），
		//   不再进 Speed/SpeedTroop → 这里对行军速度恒为 0（保留取值只为将来别的兵种专属速度来源）。
		sp := cfg.Speed * (100 + gen + tb.speedOf(cfg.Type) + tb.speedTroopOf(cfg.ID)) / 100
		if sp < 1 {
			sp = 1
		}
		if slowest == 0 || sp < slowest {
			slowest = sp
		}
	}
	return slowest
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
	// ★ 2026-10-11 resources 也一起落库：调用方若已把随军资源「送达目标城」，
	//   会先把 order.Resources 置空再调本函数（如运输到达），这里落库保证清空生效 ——
	//   否则 finishReturn 会把已送达的资源再带回一次。
	h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
		Updates(map[string]interface{}{"status": 2, "result": order.Troops,
			"return_time": order.ReturnTime, "carry": order.Carry,
			"resources": order.Resources})
}

// ezfyLootBatch 单次「批量结算」的宝物播报聚合器。
//
// ★ 2026-10-13 用户反馈「采集播报太多了」：玩家批量采集时，几十支采集队同一批结算，
//
//	每掉一件宝物就往世界频道播一条「恭喜玩家xx在野地x级采集到宝物」，全服频道刷屏。
//	批量入口（processOrders 轮询 / 一键收获）一次结算多支部队时，把掉落的宝物名攒进
//	本批次，循环结束统一 flush 成**一条**合并播报；单条结算（停止采集/召回）不传
//	批次，保持原样立即播报。
//
// ★ 2026-10-22 用户反馈「播报的是总的，改成播报按期，不然太多了」：
//
//	原来一次离线补算最多 24 期，一条播报就把 24 期的宝物全列出来（实测 12 件以上，
//	一条就占满频道）。现在 loot 里**只收「最近一期」掉的宝物**，periods 记录本次
//	实际结算了多少期，flush 时在末尾注明「共结算N期」——
//	条数不变（每人每轮结算仍 1 条），单条内容却只有 1~2 件。
type ezfyLootBatch struct {
	name    string   // 玩家昵称（首次掉落时取一次，避免每件宝物都查一次）
	loot    []string // 本批次「最近一期」掉落的宝物名
	periods int      // 本批次累计结算的采集期数（>1 时播报注明）
}

// add 记录一件掉落的宝物。
func (b *ezfyLootBatch) add(loot string) {
	if b == nil {
		return
	}
	b.loot = append(b.loot, loot)
}

// ezfyLootPeriodSuffix 采集宝物播报的「期数」后缀。
//
// ★ 2026-10-22 播报只列最近一期，补算多期时必须说明白，否则玩家会以为少给了：
// 只结算 1 期不写后缀，多期写成「（共结算N期）」。
func ezfyLootPeriodSuffix(periods int64) string {
	if periods <= 1 {
		return ""
	}
	return fmt.Sprintf("（共结算%d期）", periods)
}

// ezfyLootBatchFlush 结算批次结束：有掉落则发一条合并播报（无掉落直接返回）。
//
// ★ 2026-10-13 用户反馈「相同宝物按数量展示，不然播报太长」：
//
//	同种宝物聚合统计后拼接，形如「红宝石戒指×3」（全部宝物都带 ×N，顺序按本批首次掉落）。
//
// ★ 2026-10-22 只播最近一期 + 注明结算期数（见 ezfyLootBatch 注释）。
func (h *EzfyHandler) ezfyLootBatchFlush(uid uint, b *ezfyLootBatch) {
	if b == nil || len(b.loot) == 0 {
		return
	}
	// ★ 2026-10-26 二战系统配置「采集宝物播报」开关（默认关）：
	//   关 = 不播报、也不写播报库（ezfy_chat 系统频道），这里直接返回，连昵称查询都省掉。
	if !ezfyLootBroadcastOn() {
		return
	}
	if b.name == "" {
		b.name = h.ezfyProfileName(uid)
	}
	cnt := map[string]int{}
	order := []string{}
	for _, n := range b.loot {
		if cnt[n] == 0 {
			order = append(order, n)
		}
		cnt[n]++
	}
	parts := make([]string, 0, len(order))
	for _, n := range order {
		parts = append(parts, fmt.Sprintf("%s×%d", n, cnt[n]))
	}
	h.ezfySysChat("恭喜玩家 %s 采集收获宝物 %d 件: %s%s",
		b.name, len(b.loot), strings.Join(parts, "、"), ezfyLootPeriodSuffix(int64(b.periods)))
}

// settleDispatch 常驻采集结算(每满一个采集周期一期)
//
// ★ 2026-09-23 用户规则:
//   - 采集 = 常驻: 满一个采集周期结算一期; 玩家离线错过也会一次性补算(上限 24 期);
//   - 资源按地形单一资源产出: 野地等级越高越多、带队军官后勤每 1 点 +1%(上限 +100%);
//   - 宝物每期至少 1 件直接进背包(不受负重限制), 每期另有 20% 概率多 1 件,
//     采集时间越长期数越多宝物越多; 平原/沿海平原无珠宝则该期没有宝物。
//
// ★★ 2026-10-21 用户规则（本轮核心）:
//   - 资源先**累积进部队负重(carry)**，不直接入城；
//   - 负重装满后，本次超出的资源**直接丢弃**，之后资源不再增加；
//   - 但部队**不自动停止**，宝物仍按周期继续结算；
//   - 玩家点 [停止采集]/[一键收获] 清空负重后，才能重新采集资源；
//   - 宝物每期至少 1 件直接进背包（不受负重限制），只有宝物满才自动停止。
//
// 返回 (结算期数, 是否结算); 未满一期或野地已丢(部队自动返航)时 settled=false。
// lootBatch 可选：批量结算入口（processOrders 轮询 / 一键收获）传入聚合器，
// 把本批掉落的宝物合并成一条系统播报（见 ezfyLootBatch）；不传则逐件立即播报。
func (h *EzfyHandler) settleDispatch(uid uint, order *model.EzfyOrder, now int64, env *ezfySettleEnv, lootBatch ...*ezfyLootBatch) (int, bool) {
	// ★ 空闲驻守(arrive_time=0)或未满一期: 没有可结算的期数
	if order.ArriveTime <= 0 || now < order.ArriveTime {
		return 0, false
	}
	// ★ 2026-10-13 性能：批量结算时野地已预取（env），单支路径回落单条查询。
	var wl model.EzfyWildland
	if env != nil {
		if w := env.wildland(order.TargetId); w != nil {
			wl = *w
		}
	} else if err := h.DB.First(&wl, order.TargetId).Error; err != nil {
		wl.ID = 0 // 查询失败 → 按「野地已丢失」处理（与原来 err != nil 同口径）
	}
	if wl.ID == 0 || wl.CityId != order.CityId {
		h.beginReturn(order, now, 0)
		h.envReport(env, uid, 5, "采集报告: 野地丢失",
			"所采集的野地已不属于我方, 采集部队已返航。", "", order.ID, order.CityId)
		return 0, false
	}
	// 期数: 到点的那一期 + 玩家离线漏掉的整期; 上限 24 期防止长时间积压
	periods := 1 + (now-order.ArriveTime)/ezfyDispatchPeriod()
	if periods > 24 {
		periods = 24
	}
	// ★★ 2026-10-12 跨进程幂等认领（修复「一键收货随机多收资源 ~25 倍」）：
	//   enterProcess 只是 handler 的**进程内** map，双机共享库时两台 server 各自读到同一笔
	//   采集订单的旧 arrive_time，各结算一个完整周期（arrive_time 由最后一个写回者生效），
	//   同一周期被重复计产出 → 随机叠加多给。现在先把 arrive_time 原子推进
	//   （WHERE id=? AND arrive_time=旧值），只有没被另一台进程抢先结算的快照能认领成功；
	//   认领失败 = 已被他方结算 → 直接跳过，不重复给资源（幂等）。
	if !h.ezfyClaimSettle(order, order.ArriveTime, now+ezfyDispatchPeriod()) {
		return 0, false
	}
	level := wl.Level
	terrain := ezfyTerrainEx(wl.X, wl.Y)
	// ★★ 2026-10-21 用户规则：采集达到部队最大负重后，资源不再增加（本次超出的部分直接丢弃），
	//   但部队继续驻守，**宝物仍按周期结算**；想继续采集资源需点 [停止采集]/[一键收获]
	//   清空负重后再重新采集。离线补算多期时资源只结算负重装得下的期数，宝物仍按实际期数结算。
	capTotal := h.envCarryCap(env, order, uid)
	room := int64(-1) // <0 表示未配置负重上限，保持不限
	if capTotal > 0 {
		if before := parseCarry(order.Carry); before.total() >= capTotal {
			room = 0
		} else {
			room = capTotal - before.total()
		}
	}
	fitPeriods, g, dropped := h.ezfyFitGatherPeriods(order, &wl, periods, room, env)
	food, steel, oil, rare := g.Food, g.Steel, g.Oil, g.Rare
	resName, amt := g.ResName, g.Amount
	// ★ 2026-09-28 规则修正：产出**累积进部队负重(carry)**，不直接入城；
	//   取回负重入城由各入口（StopCollect/HarvestAll/RecallAll）负责。
	// ★ 2026-10-13 性能：批量结算时城市已预取（env），单支路径回落单条查询。
	var city *model.EzfyCity
	if env != nil {
		city = env.city(int64(order.CityId))
	}
	if city == nil {
		city = h.cityOfOrder(order, uid)
	}
	h.addCarryToOrder(order, food, steel, oil, rare, 0)
	cur := parseCarry(order.Carry)
	desc := fmt.Sprintf("采集部队在野地%d级(%d,%d)驻守结算%d期", level, wl.X, wl.Y, periods)
	if fitPeriods > 0 {
		desc += fmt.Sprintf("\n资源结算%d期\n产出: %s%d", fitPeriods, resName, amt)
		desc += fmt.Sprintf("(兵力%d × 整点%d小时 × 野地加成%g%% × 后勤%d ÷ %g)",
			g.Troops, g.Hours, g.WildBonusPct, g.Logistics, g.Divisor)
	} else {
		desc += "\n负重已满, 资源不再产出(宝物照常结算), 需[停止采集]/[一键收获]清空负重后才能重新采集资源。"
	}
	stored := cur.total()
	desc += fmt.Sprintf("\n已存资源 %d（负重上限 %d）", stored, capTotal)
	if dropped > 0 {
		desc += fmt.Sprintf("\n超出负重丢弃 %d", dropped)
	}
	carryFull := h.envCarryFull(env, order, uid)
	if carryFull && fitPeriods > 0 {
		desc += "\n负重已满, 资源不再增加; 宝物仍按周期产出, 需[停止采集]/[一键收获]清空负重后才能重新采集资源。"
	}
	// 宝物: 每满一个采集周期(每期)至少 1 件, 直接进「宝物」; 每期另有 20% 概率多 1 件
	//
	// ★ 2026-10-11 用户规则「宝物库存上限 5000（二战系统配置可配），超过 5000 采集的时候停止给提示」：
	//   装不下就不再给，并且**停止采集**（原地待命，与 [停止采集] 同语义），战报里给一句简约提示。
	treasureMax := ezfyTreasureMax()
	left := h.envTreasureLeft(env, uid)
	treasureNames := []string{} // 本次结算的**全部**宝物（只写战报，不用于世界播报）
	// ★ 2026-10-22 用户规则「播报的是总的，改成播报按期，不然太多了」：
	//   世界频道**只播最近一期**掉的宝物（1~2 件），补算多期时末尾注明「共结算N期」；
	//   原来一条播报会把 24 期的明细全堆出来（实测 12 件以上，一条占满频道）。
	//   战报里仍然列全部（玩家要能核对）。
	lastLoot := []string{}
	want := 0
	for i := int64(0); i < periods; i++ {
		n := 1
		if rand.Intn(100) < ezfyTreasureExtraPct {
			n++ // 每期另有 20% 概率多 1 件
		}
		want += n
		// 按期生成，顺便记下「最近一期」的掉落（宝物满导致后面几期没掉时，
		// lastLoot 停在最后一个真正有掉落的期，播报不会变成空）。
		cur := make([]string, 0, n)
		for k := 0; k < n && len(treasureNames) < left; k++ {
			eq := h.randomTerrainTreasure(terrain)
			if eq == nil || city == nil {
				continue // 平原/沿海平原无珠宝
			}
			h.envAddEquip(env, city, eq)
			treasureNames = append(treasureNames, eq.Name)
			cur = append(cur, eq.Name)
		}
		if len(cur) > 0 {
			lastLoot = cur
		}
	}
	treasureFull := left <= 0 || want > left
	if len(treasureNames) > 0 {
		desc += fmt.Sprintf("获得宝物(已直接放入宝物): %s\n", strings.Join(treasureNames, ", "))
		// ★ 2026-10-13 采集宝物播报防刷屏：批量结算（一键收获/轮询）传入 lootBatch 时，
		//   宝物并入批次由调用方合并成**一条**播报；单条结算（停止采集/召回）立即播报。
		// ★ 2026-10-22 并入批次的只有「最近一期」的 lastLoot，并累计结算期数。
		// ★ 2026-10-26 二战系统配置「采集宝物播报」开关（默认关）：
		//   关 = 不播报、也不写播报库（ezfy_chat），批次也不攒（省掉昵称查询）。
		//   战报里的「获得宝物」照常写 —— 只是世界频道不播。
		if ezfyLootBroadcastOn() {
			if len(lootBatch) > 0 && lootBatch[0] != nil {
				if lootBatch[0].name == "" {
					lootBatch[0].name = h.ezfyProfileName(uid)
				}
				lootBatch[0].periods += int(periods)
				for _, tn := range lastLoot {
					lootBatch[0].add(tn)
				}
			} else if len(lastLoot) > 0 {
				h.ezfySysChat("恭喜玩家 %s 采集收获宝物 %d 件: %s%s",
					h.ezfyProfileName(uid), len(lastLoot), strings.Join(lastLoot, "、"),
					ezfyLootPeriodSuffix(periods))
			}
		}
	} else if !treasureFull {
		desc += "本期无宝物(该地形不出珠宝)。\n"
	}
	if treasureFull {
		desc += fmt.Sprintf("宝物已达上限(%d), 采集已停止, 请先处理宝物后再点[采集]。\n", treasureMax)
	}
	// ★★ 2026-10-21 用户规则：负重满**不再自动停止**——资源停、宝物继续；
	//   只有宝物满才自动停止采集（原地待命，与 [停止采集] 同语义）。
	nextArrive := now + ezfyDispatchPeriod()
	if treasureFull {
		nextArrive = 0
	} else if carryFull {
		desc += fmt.Sprintf("部队继续驻守(只结算宝物), 可随时[召回]或[停止采集]。\n待带回 %d/%d\n",
			cur.total(), capTotal)
	} else {
		desc += fmt.Sprintf("部队继续驻守采集, 可随时[召回]或[停止采集]。\n待带回 %d/%d\n",
			cur.total(), capTotal)
	}
	order.ArriveTime = nextArrive
	order.Result = order.Troops
	upd := map[string]interface{}{"arrive_time": order.ArriveTime,
		"result": order.Result, "carry": order.Carry}
	if treasureFull {
		upd["collect_start"] = 0 // 宝物满停止采集（与 [停止采集]/[一键收获] 落库口径一致）
	}
	h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).Updates(upd)
	// ★ 2026-10-13 性能：传 order.CityId，省掉 addReport 里「按订单回查城市」的那次跨库查询。
	h.envReport(env, uid, 5, "采集报告: 驻守结算", desc, "", order.ID, order.CityId)
	return int(periods), true
}

// ezfyGatherSetAmount 把一次采集产出改成指定数量（采集一次只产一种资源）。
func ezfyGatherSetAmount(g *ezfyGatherCalc, amt int64) {
	if amt < 0 {
		amt = 0
	}
	g.Food, g.Steel, g.Oil, g.Rare = 0, 0, 0, 0
	switch g.ResName {
	case "钢铁":
		g.Steel = amt
	case "石油":
		g.Oil = amt
	case "稀矿":
		g.Rare = amt
	default:
		g.Food = amt
	}
	g.Amount = amt
}

// ezfyFitGatherPeriods 在剩余负重 room 内计算实际能结算资源的期数。
//
// ★★ 2026-10-21 用户规则：采集达到部队最大负重后，本次多出来的资源直接丢弃，
// 之后资源不再产出，但**宝物仍按周期继续结算**，直到玩家停止/收获清负重。
// 因此本函数只限制资源期数，不改变调用方按完整期数结算宝物。room < 0 表示不限负重。
//
// 返回 (实际资源期数, 对应产出, 最后一期超出负重被丢弃的量)。
func (h *EzfyHandler) ezfyFitGatherPeriods(order *model.EzfyOrder, wl *model.EzfyWildland,
	want, room int64, env *ezfySettleEnv) (int64, ezfyGatherCalc, int64) {
	period := ezfyDispatchPeriod()
	g := h.dispatchGatherYield(order, wl, want*period, env)
	if want <= 0 || room < 0 || g.Amount <= room {
		return want, g, 0
	}
	if room <= 0 {
		ezfyGatherSetAmount(&g, 0)
		return 0, g, 0
	}
	// 二分出「完整装得下」的最大期数，再补上装到负重上限的那一期。
	// ★ 性能：直接用首轮算出的公式参数本地算各期产量，避免在 processOrders 单支路径
	//   反复查军官/野地加成（原二分每轮一次 DB 查询，1 核线上会明显变慢）。
	amountAt := func(n int64) int64 {
		if n <= 0 {
			return 0
		}
		amt := ezfyGatherAmount(g.Troops, float64(ezfyGatherHours(n*period)),
			g.WildBonusPct, g.Logistics, g.Divisor, g.SeaMult)
		if amt < 1 {
			amt = 1
		}
		return amt
	}
	lo, hi := int64(0), want
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if amountAt(mid) <= room {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	n := lo + 1
	if n > want {
		n = want
	}
	if n != want {
		g = h.dispatchGatherYield(order, wl, n*period, env)
	}
	if g.Amount > room {
		dropped := g.Amount - room
		ezfyGatherSetAmount(&g, room)
		return n, g, dropped
	}
	return n, g, 0
}

// ezfyClaimSettle 原子认领一次采集结算窗口（跨进程幂等，防双机重复结算多给资源）。
//
// ★★ 2026-10-12 修复「一键收货随机多收资源」：enterProcess 只是进程内锁，双机共享库时
//
//	两台 server 会各自把同一笔采集订单的同一周期结算一次。这里用条件更新
//	`WHERE id=? AND arrive_time=?（旧快照）` 把 arrive_time 原子推进到 next：
//	只有没被另一台进程抢先结算的快照能认领成功（RowsAffected==1）；否则说明他方已结算 → 返回 false。
func (h *EzfyHandler) ezfyClaimSettle(order *model.EzfyOrder, oldArrive, next int64) bool {
	if oldArrive <= 0 {
		return false
	}
	res := h.DB.Model(&model.EzfyOrder{}).
		Where("id = ? AND arrive_time = ?", order.ID, oldArrive).
		Update("arrive_time", next)
	if res.Error != nil || res.RowsAffected == 0 {
		return false
	}
	return true
}

// ezfyGatherCalc 一次采集产量计算的全部明细（供入库/战报/实时预览共用）。
type ezfyGatherCalc struct {
	Food, Steel, Oil, Rare int64
	ResName                string
	Troops                 int64   // M：部队总兵力（所有兵种数量之和）
	Hours                  int64   // N：整点小时数（向下取整）；未满一期连续折算时 = floor(连续小时)
	HoursF                 float64 // N 的精确值：满期=整点整数；停止/收获折算=真实毫秒连续小时
	WildBonusPct           float64 // R：同资源野地总等级 × 每级加成%
	Logistics              int     // A：带队军官最终后勤（基础 + 装备 + 套装）
	Divisor                float64 // 公式除数（默认 2）
	IsSea                  bool    // 目标是否海野（海底森林/岛屿/渔场/矿礁）
	SeaMult                float64 // 海野采集倍率（仅海野生效，默认 3；陆地恒为 1）
	Amount                 int64   // F：本次产出合计（海野 = 公式结果 × SeaMult）
}

// ezfyGatherTroopCount 统计出征部队的**总兵力**（M）。
//
// ★ 2026-10-21 用户口径修正：M 不是「卡车数量」那么狭义 —— 是**所有兵种数量之和**
//
//	（原公式说明拿运输卡车举例而已）；不同兵种的差异只体现在**负重**不同，
//	对采集产量的贡献一律按兵力数量计。order.Troops 兼容 JSON
//	`[{"troopId":3,"count":100}]` 与老格式 `3:100,1:50`。
func ezfyGatherTroopCount(raw string) int64 {
	var total int64
	for _, g := range parseGroups(raw) {
		if g.Count > 0 {
			total += g.Count
		}
	}
	return total
}

// ezfyGatherAmount 采集公式核心：F = M × N × R × A ÷ divisor（海野再 × seaMult）。
//
//	M = troops（部队总兵力：所有兵种数量之和，兵种差异只体现在负重）
//	N = hours（小时数，float64）：
//	    - 满期结算路径传**整点小时**（向下取整后的整数）；
//	    - [停止采集]/[一键收获]/[提前召回] 未满一期的折算传**连续小时**
//	      （真实毫秒 ÷ 3600000；★★ 2026-10-21 用户规则「不要按整点卡，
//	      采集多少就是收获多少」——采 37 分钟就拿 37 分钟的产量）。
//	R = wildPct（百分比，例如 42 表示 42%）
//	A = logistics（带队军官**最终后勤**：基础 + 装备 + 套装，见 gatherLogistics）
//	seaMult = 海野采集倍率（default 3）：仅海野调用方传 >1 的值；陆地传 1；
//	          <=0 视为 1（不做海野加成），0 不会把产出清零。
//
// 用 +1e-6 再 floor，避免 42% 这类十进制在 float 里算成 195955199.99999997 被截成少 1。
func ezfyGatherAmount(troops int64, hours float64, wildPct float64, logistics int, divisor, seaMult float64) int64 {
	if troops <= 0 || hours <= 0 || wildPct <= 0 || logistics <= 0 || divisor <= 0 {
		return 0
	}
	v := float64(troops) * float64(hours) * wildPct * float64(logistics) / (100 * divisor)
	if seaMult > 0 && seaMult != 1 {
		v *= seaMult
	}
	if v <= 0 {
		return 0
	}
	if v >= float64(math.MaxInt64) {
		return math.MaxInt64
	}
	return int64(math.Floor(v + 1e-6))
}

// ezfyGatherWildBonusPctFor 计算 R：出发城市名下「同资源野地」的总等级 × 每级加成%。
//
// 野地加成不是看单块野地的等级，而是**全城同资源野地等级之和**：
//
//	例：城市名下 10 级森林 + 4 级森林，每级 3% → 森林（粮食）加成 42%。
//
// 批量结算用 env 预汇总（同城多支共享）；单支路径回落一次按城查询。
func (h *EzfyHandler) ezfyGatherWildBonusPctFor(order *model.EzfyOrder, wl *model.EzfyWildland, resName string, env *ezfySettleEnv) float64 {
	levels := 0
	if m := env.wildLevels(order.CityId); m != nil {
		levels = m[resName]
	} else {
		var wls []model.EzfyWildland
		h.DB.Select("city_id", "x", "y", "level").Where("city_id = ?", order.CityId).Find(&wls)
		levels = ezfyWildResLevels(wls)[resName]
	}
	// 防御：缓存里没算到目标野地（例如 env 是别的入口临时构造）时，至少把目标本身算进去。
	if levels <= 0 && wl != nil && wl.CityId == order.CityId &&
		ezfyGatherResName(ezfyTerrainEx(wl.X, wl.Y)) == resName {
		levels = wl.Level
	}
	if levels <= 0 {
		return 0
	}
	return float64(levels) * ezfyGatherWildBonusPct()
}

// dispatchGatherYield 按驻守时长(毫秒)计算采集资源产出。
//
// ★★ 2026-10-21 改为线上原版公式：F = M × N × R × A ÷ 2
//
//	M = 部队总兵力（出征部队**所有兵种数量之和**，原说明以运输卡车举例而已；
//	    不同兵种的差异只体现在负重，产量贡献一律按数量计）
//	N = 整点小时数（16:59 按 16 小时算，向下取整）
//	R = 野地加成%（城市名下同资源野地总等级 × 每级 3%）
//	A = 带队军官最终后勤（基础 + 装备 + 套装；2026-10-31 起吃装备）
//
// ★ 2026-10-21 用户规则「海野需要比陆地野地高 3 倍」：
//
//	海野（海底森林/岛屿/渔场/矿礁，ezfyIsSeaWildTerrain）产出 = 公式结果 × 海野倍率（默认 3，
//	管理端「二战系统配置」可调）；陆地野地不乘。
//
// 例：180000 卡车 × 16 小时 × 42% × 324 后勤 ÷ 2 = 195955200。
// 产出仍是单一资源（按地形判定）；ms 不足 1 整点时按原有兜底至少 1 点。
//
// ★★ 口径分工：
//   - 本函数 = **满期/整期口径**（settleDispatch 满期结算、ReportDynamics 每期预览）：
//     N 向下取整到整点小时；
//   - 未满一期的折算（[停止采集]/[一键收获]/[提前召回]）走 dispatchGatherYieldPartial：
//     N 用真实毫秒**连续小时**，不取整（2026-10-21 用户规则「采集多少就是收获多少」）。
func (h *EzfyHandler) dispatchGatherYield(order *model.EzfyOrder, wl *model.EzfyWildland, ms int64, envs ...*ezfySettleEnv) ezfyGatherCalc {
	// ★ 2026-10-13 性能：批量结算传入请求级缓存 env（军官/野地只查一次），单支路径不传。
	var env *ezfySettleEnv
	if len(envs) > 0 {
		env = envs[0]
	}
	// N = 整点小时数（16:59 按 16 小时算，向下取整）。
	return h.gatherYieldAt(order, wl, float64(ezfyGatherHours(ms)), ms, env)
}

// dispatchGatherYieldPartial 未满一期的部分结算口径（[停止采集]/[一键收获]/[提前召回] 共用）。
//
// ★★ 2026-10-21 用户规则「不要按整点卡，采集多少就是收获多少」：
//
//	N 按真实驻守毫秒折成**连续小时**（如 37 分钟 = 0.6167 小时），不再向下取整到整点 ——
//	此前采 37 分钟整点取整后 hours=0、只能拿保底 1 点（玩家反馈「负重资源 1 已入库」即此）。
//	驻守过的保底仍保留：ms>0 且产出不足 1 点时至少给 1（见 gatherYieldAt）。
func (h *EzfyHandler) dispatchGatherYieldPartial(order *model.EzfyOrder, wl *model.EzfyWildland, ms int64, env *ezfySettleEnv) ezfyGatherCalc {
	return h.gatherYieldAt(order, wl, float64(ms)/(3600*1000), ms, env)
}

// gatherLogistics 带队军官用于采集公式的「最终后勤」= 基础后勤 + 已穿戴装备 + 已穿齐套装。
//
// ★★ 2026-10-31 用户规则：采集公式的后勤项（A）改吃**最终后勤**，为后续上「后勤装备」做准备。
// 原来这里直接取 officer.Logistics（只有基础值），给采集军官穿 +后勤的装备/宝物产量完全不变，
// 与市长产量（走 officerEffective）的口径也不一致。
//
// 性能：批量结算 / 军队动态同一批里同一名军官只解析一次装备 JSON（env.gatherLogs 缓存）。
func (h *EzfyHandler) gatherLogistics(env *ezfySettleEnv, o *model.EzfyOfficer) int {
	if o == nil {
		return 0
	}
	if env != nil && env.gatherLogs == nil {
		env.gatherLogs = map[uint]int{} // 防御：手工构造的 env 没初始化也不 panic
	}
	if env != nil && o.ID > 0 {
		if log, ok := env.gatherLogs[o.ID]; ok {
			return log
		}
	}
	_, log, _ := h.officerEffective(o)
	if env != nil && o.ID > 0 {
		env.gatherLogs[o.ID] = log
	}
	return log
}

// gatherYieldAt 采集产出共用内部实现：hoursF 为计入公式的小时数（整点整数或连续小时，
// 由调用方决定口径）；fallbackMs>0 且产出不足 1 点时兜底 1 点（驻守过就不空手）。
func (h *EzfyHandler) gatherYieldAt(order *model.EzfyOrder, wl *model.EzfyWildland, hoursF float64, fallbackMs int64, env *ezfySettleEnv) ezfyGatherCalc {
	// 地形只算一次：资源名（粮/钢/油/稀）与「是否海野」共用同一地形判定，天然同源。
	terrain := ezfyTerrainEx(wl.X, wl.Y)
	resName := ezfyGatherResName(terrain)
	// ★ 2026-10-23 沿海平原也按海野（用户规则）→ 用 Rules 口径；倍率取配置（默认 3）
	isSea := ezfyIsSeaWildRules(terrain)
	troops := ezfyGatherTroopCount(order.Troops)
	logistics := 0
	// ★ 2026-10-24 优先按命令里的军官 id 取（同名军官互不误伤）；无 id（孤儿订单）才按名字，
	//   名字也关联不到 → 无后勤加成（与老逻辑一致）。
	var officer *model.EzfyOfficer
	if order.OfficerId > 0 {
		officer = h.officerOf(uint(order.CityId), int64(order.OfficerId))
	}
	if officer == nil {
		officer = h.envOfficer(env, uint(order.CityId), order.Officer)
	}
	if officer != nil {
		// ★ 2026-10-31 采集后勤 = 最终后勤（基础 + 装备 + 套装），不再是基础值。
		logistics = h.gatherLogistics(env, officer)
	}
	wildPct := h.ezfyGatherWildBonusPctFor(order, wl, resName, env)
	divisor := ezfyGatherFormulaDivisor()
	seaMult := 1.0
	if isSea {
		seaMult = ezfyGatherSeaBonusMult()
	}
	amt := ezfyGatherAmount(troops, hoursF, wildPct, logistics, divisor, seaMult)
	// ★ 2026-09-24 修复「提前采集资源0」: 驻守过就至少给 1 点资源。
	if fallbackMs > 0 && amt < 1 {
		amt = 1
	}
	out := ezfyGatherCalc{
		ResName: resName, Troops: troops, Hours: int64(hoursF), HoursF: hoursF,
		WildBonusPct: wildPct, Logistics: logistics, Divisor: divisor, Amount: amt,
		IsSea: isSea, SeaMult: seaMult,
	}
	switch resName {
	case "钢铁":
		out.Steel = amt
	case "石油":
		out.Oil = amt
	case "稀矿":
		out.Rare = amt
	default:
		out.Food = amt
	}
	return out
}

// ezfyGatherHoursText 折算时长文案：<1 小时显示「N分钟」，否则显示两位小数的「X.XX小时」。
//
// ★ 2026-10-21 部分结算改为连续小时折算后，战报里不再有「整点N小时/不足1整点」的表述。
func ezfyGatherHoursText(hoursF float64) string {
	if hoursF < 1 {
		return fmt.Sprintf("%d分钟", int64(math.Round(hoursF*60)))
	}
	return fmt.Sprintf("%g小时", math.Round(hoursF*100)/100)
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
func (h *EzfyHandler) settleDispatchOnRecall(uid uint, order *model.EzfyOrder, now int64, envs ...*ezfySettleEnv) {
	if order.ArriveTime <= 0 {
		return // 空闲驻守: 还没开始采集, 没有产出
	}
	var env *ezfySettleEnv
	if len(envs) > 0 {
		env = envs[0]
	}
	if now >= order.ArriveTime {
		h.settleDispatch(uid, order, now, env)
		return
	}
	// 未满一期: 提前召回, 资源按已驻守时长比例结算, 不做宝物
	h.settlePartialCollect(uid, order, now, "提前召回", env)
}

// settlePartialCollect 未满一个采集周期的部分结算(「停止采集」/「提前召回」共用):
//   - 资源按**真实驻守时长连续折算**（★★ 2026-10-21 用户规则「不要按整点卡，
//     采集多少就是收获多少」—— dispatchGatherYieldPartial，不再整点取整/保底1点卡死），
//     装得下的部分存部队「待带回(carry)」，超出剩余负重直接丢弃；入城由调用方入口
//     （[停止采集]/[一键收获]/[召回] 随后的 harvestCarryToCity）负责;
//   - **不做宝物**(不满一个采集周期无宝物);
//   - 只把结果写进 result, **不接回** —— 是否返航由调用方决定。
//
// label: 描述文案(「停止采集」/「提前召回」), 用于战报正文与标题。
func (h *EzfyHandler) settlePartialCollect(uid uint, order *model.EzfyOrder, now int64, label string, env *ezfySettleEnv) {
	if order.TargetId <= 0 {
		return
	}
	// ★ 2026-10-13 性能：批量结算时野地已预取（env），单支路径回落单条查询。
	var wl model.EzfyWildland
	if env != nil {
		if w := env.wildland(order.TargetId); w != nil {
			wl = *w
		}
	} else if err := h.DB.First(&wl, order.TargetId).Error; err != nil {
		wl.ID = 0
	}
	if wl.ID == 0 || wl.CityId != order.CityId {
		return
	}
	started := order.ArriveTime - ezfyDispatchPeriod() // 本期起算点(=上次结算或开始采集时间)
	elapsed := max64(1, now-started)
	if elapsed > ezfyDispatchPeriod() {
		elapsed = ezfyDispatchPeriod()
	}
	g := h.dispatchGatherYieldPartial(order, &wl, elapsed, env) // 未满一期：连续小时，不整点取整
	// ★★ 2026-10-21 用户规则：产出超出剩余负重的部分**直接丢弃**。
	//   本函数只负责把装得下的部分存进 carry；入城由调用方入口的点按钮动作触发。
	capTotal := h.envCarryCap(env, order, uid)
	room := int64(-1)
	if capTotal > 0 {
		if before := parseCarry(order.Carry); before.total() >= capTotal {
			room = 0
		} else {
			room = capTotal - before.total()
		}
	}
	fit, dropped := ezfyCarryTake(ezfyCarry{Food: g.Food, Steel: g.Steel, Oil: g.Oil, Rare: g.Rare}, room)
	food, steel, oil, rare := fit.Food, fit.Steel, fit.Oil, fit.Rare
	resName, amt := g.ResName, fit.total()
	h.addCarryToOrder(order, food, steel, oil, rare, 0)
	cur := parseCarry(order.Carry)
	desc := fmt.Sprintf("采集部队在野地%d级(%d,%d)%s, 按驻守时长折算资源",
		wl.Level, wl.X, wl.Y, label)
	if amt > 0 {
		desc += fmt.Sprintf("\n产出: %s%d", resName, amt)
		desc += fmt.Sprintf("(兵力%d × %s × 野地加成%g%% × 后勤%d ÷ %g)",
			g.Troops, ezfyGatherHoursText(g.HoursF), g.WildBonusPct, g.Logistics, g.Divisor)
	} else {
		desc += "\n负重已满, 本次没有新增资源"
	}
	stored := cur.total()
	desc += fmt.Sprintf("\n已存资源 %d（负重上限 %d）", stored, capTotal)
	if dropped > 0 {
		desc += fmt.Sprintf("\n超出负重丢弃 %d", dropped)
	}
	if capTotal > 0 && stored >= capTotal {
		desc += "\n负重已满, 资源不再增加; 请点[停止采集]/[一键收获]清空负重后重新采集资源。"
	}
	desc += fmt.Sprintf("\n（%s不满一个采集周期, 本期没有宝物, 取回负重需[停止采集]/[召回]入城）", label)
	order.Result = order.Troops
	h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
		Updates(map[string]interface{}{"result": order.Result, "carry": order.Carry})
	h.envReport(env, uid, 5, "采集报告: "+label+"结算", desc, "", order.ID, order.CityId)
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
		act := h.ezfyOrderActType(uid, order)
		// ★ 2026-10-30 到达时二次卡控活动目标每日上限：并发下单可能同时穿过创建时的读校验，
		//   这里按订单 id 排名只放行当日前 N 支；超出/已抓到守将 → 落回普通野地结算。
		if act > 0 && (h.playerOwnsActWildGeneral(uid, order.TargetX, order.TargetY) ||
			!h.ezfyActOrderWithinDailyLimit(order)) {
			act = 0
			order.ActType = 0
			h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).Update("act_type", 0)
		}
		if act > 0 {
			h.processActivityBattle(uid, city, order, now, act)
			return
		}
	}

	// 采集(4)/派遣(7): 到达已占领野地 → 转为常驻驻军(空闲待命)
	// ★ 2026-09-24 用户规则: 到达后驻守**空闲**, 必须手工点[采集](或驻军区的[一键采集])
	//   才开始采集 —— 不再自动进采集。开始采集(StartCollect)后每满一个采集周期结算一期:
	//   资源**直接入起点城市**(F = 卡车×整点小时×野地加成%×后勤÷2), 宝物直接进背包(每期至少 1 件);
	//   ★ 2026-10-21：负重装满后资源不再增加，但宝物继续按周期结算；空闲驻军用 arrive_time=0 表示(不参与结算)。
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
			fmt.Sprintf("采集部队已抵达野地(%d,%d)驻守, 当前空闲待命。\n请到「军情 → 驻军」点[采集](或[一键采集])开始采集: 每满一个采集周期结算一期, 资源**直接入库到出发城市**(等级越高越多, 军官后勤每点+1%%), 宝物直接进背包(每期至少1件)。负重装满后资源不再增加, 但宝物仍按周期结算; 想继续采资源请先[停止采集]/[一键收获]清空负重。可随时[召回]撤兵。",
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
		// ★★ 2026-10-17 用户口径（上限读二战系统配置 res_max_*，不写死）：
		//   运输到达目标城按「资源最大值」封顶累加，装不下的部分**原路带回出发城**
		//   （记进 Carry，随部队返航入出发城，同样封顶）。
		// ★★ 2026-10-22 修复「黄金被吞」：ezfyDeliverCarryToCity 会先把目标城
		//   懒结算到当前再算 fit，避免待落库产量占满剩余容量后 SQL 封顶把运来的资源吞掉。
		fit, exc := h.ezfyDeliverCarryToCity(&target, ezfyCarry{Food: f, Steel: s, Oil: o, Rare: r, Gold: g})
		if exc.total() > 0 {
			order.Carry = carryJSON(exc) // 装不下的原路带回出发城
		} else {
			order.Carry = ""
		}
		// ★ 2026-10-11 资源已送达目标城 → 清空 Resources，避免返航时又被 finishReturn 带回一次
		order.Resources = ""
		desc := fmt.Sprintf("运输部队已到达%s\n", target.Name)
		if fit.total() > 0 {
			desc += fmt.Sprintf("送达: 粮%d 钢%d 油%d 稀矿%d 金%d\n",
				fit.Food, fit.Steel, fit.Oil, fit.Rare, fit.Gold)
		}
		if exc.total() > 0 {
			desc += fmt.Sprintf("目标城已接近上限, 装不下随队带回: 粮%d 钢%d 油%d 稀矿%d 金%d\n",
				exc.Food, exc.Steel, exc.Oil, exc.Rare, exc.Gold)
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
		// ★★ 2026-10-18 增援到**自己的附属野地**(驻守野地)：
		//   抵达后部队编为该野地「驻军队列」(status=3, 与盟友城驻军同口径)，
		//   敌军进攻该野地时守军 = 驻军队列 + 采集部队(见战斗结算守军构建)；
		//   不占「一块野地一支采集队」名额，可与采集部队并存（用户需求）。
		if order.TargetType != 3 {
			var wl model.EzfyWildland
			if err := h.DB.First(&wl, order.TargetId).Error; err != nil || wl.CityId != order.CityId {
				h.beginReturn(order, now, 0)
				h.addReport(uid, 5, "增援报告: 野地已丢失",
					"增援的附属野地已不属于我方, 增援部队已返航。", "", order.ID)
				return
			}
			// ★★ 2026-10-19 用户澄清：**驻守部队就是驻守用的，不能采集**
			//   （「不然野地没部队，别人一征服就拿走了」）。
			//   所以这里保持「出站驻军」形态（order_type=6 + status=3）：
			//   · 敌军进攻该野地时作为守军参战（战斗结算按 ezfyWildDefenderOrderCond 构建守军）；
			//   · 玩家可单支/一键 [召回]；
			//   · **不提供** [采集]/[收获]（那是采集部队 order_type=7 的事）。
			order.Status = 3
			h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
				Updates(map[string]interface{}{"status": 3})
			h.addReport(uid, 5, "增援报告: 附属野地",
				fmt.Sprintf("增援部队已抵达%d级野地(%d,%d)驻防, 敌袭时协同防守。", wl.Level, wl.X, wl.Y), "", order.ID)
			return
		}
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
		// ★★ 2026-10-17 增援(6) = 驻守，不带资源（盟友间资源运输靠运输(5)）。
		//   创建订单时后端已强制清空 order.Resources（CreateOrder 兜底，防客户端伪造）。
		//   own 分支订单删除 → 残留退加回出发城；盟友城分支订单保留 → resources 留着，
		//   召回时由 finishReturn 并入带回出发城（见 own 分支的处理与下方 else 分支）。
		if own {
			// ★ 2026-10-02 增援自己城市: 兵力/军官已并入目标城, 订单已完结 → 删除,
			//   避免 status=3 僵尸订单被军情误判为驻军、占用驻军槽位。
			// ★★ 2026-10-17 订单删除无召回路径：残留 resources（旧数据）退加回出发城
			//   （按配置上限封顶），避免蒸发。
			if res := h.parseResMap(order.Resources); res["food"]+res["steel"]+res["oil"]+res["rare"]+res["gold"] > 0 {
				h.DB.Model(&model.EzfyCity{}).Where("id = ?", city.ID).Updates(map[string]interface{}{
					"food":  ezfyResAddExpr("food", res["food"]),
					"steel": ezfyResAddExpr("steel", res["steel"]),
					"oil":   ezfyResAddExpr("oil", res["oil"]),
					"rare":  ezfyResAddExpr("rare", res["rare"]),
					"gold":  ezfyResAddExpr("gold", res["gold"]),
				})
				desc += fmt.Sprintf("\n残留随军资源已退回%s: 粮%d 钢%d 油%d 稀矿%d 金%d",
					city.Name, res["food"], res["steel"], res["oil"], res["rare"], res["gold"])
			}
			h.DB.Delete(&model.EzfyOrder{}, order.ID)
		} else {
			// 盟友城驻军：订单保留，召回时 finishReturn 会把 order.Resources 并入带回出发城
			// （创单时已强制清空，正常为空；旧数据残留则随召回带回，不蒸发）。
			order.Status = 3
			h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
				Updates(map[string]interface{}{"status": 3})
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
		// ★★ 2026-10-17 用户口径（上限读二战系统配置 res_max_*，不写死）：
		//   派遣到达目标城按「资源最大值」封顶累加，装不下的部分**原路带回出发城**
		//   （记进 back → Carry，随部队返航入出发城，同样封顶）。
		// ★★ 2026-10-22 修复「黄金被吞」：原先先算 fit、再 calcResource 结算目标城，
		//   待落库产量会占掉容量，最后 SQL 封顶吞掉随军资源且不记 back。
		//   现在统一交给 ezfyDeliverCarryToCity（先结算目标城、再算 fit、事务内入库）。
		res := h.parseResMap(order.Resources)
		fit, exc := h.ezfyDeliverCarryToCity(&target, ezfyCarry{
			Food: res["food"], Steel: res["steel"], Oil: res["oil"], Rare: res["rare"], Gold: res["gold"],
		})
		back := exc
		if fit.total() > 0 {
			desc += fmt.Sprintf("\n随军资源已入库: 粮%d 钢%d 油%d 稀矿%d 金%d",
				fit.Food, fit.Steel, fit.Oil, fit.Rare, fit.Gold)
		}
		if exc.total() > 0 {
			desc += fmt.Sprintf("\n目标城已接近上限, 装不下随队带回: 粮%d 钢%d 油%d 稀矿%d 金%d",
				exc.Food, exc.Steel, exc.Oil, exc.Rare, exc.Gold)
		}
		// ★ 2026-10-11 随军资源已送达目标城（或本来就没有）→ 清空 Resources，
		//   否则返航时 finishReturn 会把它们再带回出发城（凭空多一份）。
		order.Resources = ""
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
					"return_time": order.ReturnTime, "resources": ""})
		} else {
			order.Status = 3
			h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
				Updates(map[string]interface{}{"status": 3, "carry": "", "resources": ""})
		}
		h.addReport(uid, 5, "派遣报告: "+target.Name, desc, "", order.ID)
		return
	}

	// ============ 战斗类: 侦查/掠夺/征服 ============
	attacker := parseGroups(order.Troops)
	atkTech := h.techMap(city.ID)
	// 带队军官(军事属性 + 装备 + 技能)与科技加成
	// ★ 2026-10-24 按命令里的军官 id 精确取（同名军官互不误伤）；老数据无 id 按名字兜底
	leadOfficer := h.officerByName(city.ID, order.Officer)
	if order.OfficerId > 0 {
		if o := h.officerOf(city.ID, int64(order.OfficerId)); o != nil {
			leadOfficer = o
		}
	}
	officerBonus := h.officerBattleBonus(leadOfficer)
	// 攻击加成：枪械改良(6)+3%/级 · 重型机床(9)+2%/级
	// ★ 2026-10-21 动员训练(5) 改为「造兵速度」科技，不再参与战斗攻击加成。
	// ★ 2026-10-06 增程弹药(8) 改为**射程加成**（：增程弹药=射程，不参与攻击加成）
	atkBonus := officerBonus + atkTech[6]*3 + atkTech[9]*2
	// 速度加成：只有机械增压(10)「部队速度」是**全体兵种**通用的；
	// ★ 2026-10-08 喷气推进(19)「空军速度」是**兵种专属** → 移到下方 atkType（原来错误地加了全军）
	// ★ 2026-10-31 满级 50%：读配置（不再写死每级 2%）
	atkSpeedBonus := ezfyTechEffectValueAt(10, atkTech[10])
	// ★ 2026-10-06 攻方射程加成：增程弹药(8)+3%/级（射程 = 基础射程 × (1+科技加成)，）
	atkRangeBonus := atkTech[8] * 3
	// ★★ 2026-10-08 兵种专属加成（配置里写明作用兵种的那些科技/军官技能）：
	//   · 喷气推进(19)「空军速度+3%/级」→ **只加空军**
	//   · 军官兵种技能：火炮压制=陆军装甲攻击 / 四指阵列=空军对空攻击 / 狼群围猎=海军对海攻击 /
	//     坦克冲锋=陆军速度 / 闪电疾攻=空军速度 / 越岛登陆=海军速度
	//   ★ 原来它们被无差别算进 atkBonus / atkSpeedBonus（全军受益）——
	//     用户反馈「空军速度只加空军，有些科技是特定兵种」。
	atkType := ezfyTypeBonus{Speed: map[int]int{}, Def: map[int]int{}}
	if v := ezfyTechEffectValueAt(19, atkTech[19]); v > 0 {
		atkType.Speed[ezfyTroopTypeAir] += v
	}
	// ★★ 2026-10-09 永备工事「只对城防单位生效，也是属于兵种单位加成只不过是城防单位类型」
	//   （用户口径）→ 不再进全军通用防御 `atkDefBonus`，改成挂在城防(4) 的**兵种专属防御**上。
	//   攻方出征**不能带城防兵**（createOrder 拦 type=4）→ 这一截对攻方实际恒不生效，
	//   保留是为了与守方口径对称（以后若放开城防出征也能直接生效）。
	if v := atkTech[16] * 2; v > 0 {
		atkType.Def[ezfyTroopTypeCity] = v
	}
	atkType = h.officerTypeBonus(leadOfficer, atkType)
	// ★ 2026-10-08 攻方军官行同时展示攻击加成与防御加成（都是**属性部分**，技能单列）：
	//   出征军官的属性/技能同样给部队提供防御（被打时减伤），原来军官行只写攻击加成（用户反馈）。
	atkOfficerDesc := h.officerBattleDesc(leadOfficer, h.officerBaseBonus(leadOfficer), "攻击加成",
		h.officerGuardAttrBonus(leadOfficer), "防御加成")
	// ★ 2026-10-06 战报拆解逐项明细：攻方科技/技能逐项（科技名见 ezfy_cfg 种子表）
	// ★ 2026-10-08 补全影响 攻击/防御/射程/速度 的全部攻方科技（原来只列攻击类 → 看头像显示不全）
	atkTechs := ezfyBonusItems(
		ezfyTechItem("枪械改良", atkTech[6]*3),
		ezfyTechItem("均质装甲", atkTech[7]*3),
		ezfyTechItem("增程弹药", atkTech[8]*3),
		ezfyTechItem("重型机床", atkTech[9]*2),
		ezfyTechItem("机械增压", ezfyTechEffectValueAt(10, atkTech[10])),
		ezfyTechItem("永备工事", atkTech[16]*2),
		ezfyTechItem("喷气推进", ezfyTechEffectValueAt(19, atkTech[19])),
	)
	atkSkillBreak := h.officerSkillsBreak(leadOfficer)
	// ★ 装备六项战斗加成（伤害/防御/生命/移动距离/暴击几率/暴击伤害）
	atkEquip := h.officerBattleEquipBonus(leadOfficer)
	// ★ 2026-10-07 攻方「防御加成」：出征军官属性(学识)+防御技能(弧形壁垒/弹幕覆盖)+装备 Def。
	//   原引擎攻方被打时防御恒 0 —— 军官带弧形壁垒 Lv.5「防御力+150%」既不生效也不展示（用户反馈）。
	//   ★ 2026-10-08 再加**防御科技**（均质装甲/重型机床）：原来只算军官+技能+装备，
	//     不带队军官时防御恒 0 → 玩家有防御科技也显示「防御+0%」（用户反馈）。口径与守方
	//     防御科技一致（守方 defBonus 用同 7/9 两档）。伤害减伤与该展示共用 atkDefBonus。
	//   ★★ 2026-10-09 永备工事(16) 移出通用防御 → 只对**城防单位**生效（见 atkType.Def[4]），
	//     所以这里不再 `+ atkTech[16]*2`，明细里也不列（否则「列了却不生效」）。
	atkDefBonus := 0
	var atkDefBreak []ezfyBonusItem
	atkDefBonus += atkTech[7]*3 + atkTech[9]*2
	atkDefBreak = ezfyBonusItems(atkDefBreak,
		ezfyTechItem("均质装甲", atkTech[7]*3), ezfyTechItem("重型机床", atkTech[9]*2))
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
	atkDefBreak = append(atkDefBreak, ezfyTechItem("装备套装", atkEquip.Def)...)
	// 城守(仅玩家城市防守方)
	var cityGuard *model.EzfyOfficer
	defEquip := ezfyBattleBonus{}
	defOfficerDesc := ""
	// ★★ 2026-10-19 玩家附属野地守军带队的军官（增援6/采集7 各带一名军官）：
	//   攻方打「有驻军的附属野地」时，战报/指挥室也要能看到守方带的军官
	//   （用户反馈「被攻击方附属野地增援带了军官，攻击方打的时候没显示」）。
	wildDefOfficerDesc := ""

	defender := []ezfyUnitGroup{}
	var defExclude map[int]bool // 城市「不参与」兵种集合（case 3 里填充，驻军战块后构建友军守军时用）
	// ★ 2026-10-28 城市「兵种防守数量」配置（case 3 里填充）：实际参战 = min(城内, 配置)。
	var defCaps map[int]int64
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
	// ★ 2026-10-09 PvP 守方「伤兵回收率」的额外加成%（守方战地医疗科技 + 城守军官技能·机械改装），
	//   case 3 里填；守方战损按「兵种修复率 + 这个值」进伤兵营（守方伤兵也入营 + 吃科技/技能）。
	defHealTech := 0
	// ★ 2026-10-09 守方战损入伤兵营的合计（战报里展示「守方伤兵入营」）
	defRepairedTotal := int64(0)
	// ★ 2026-10-08 守方**兵种专属**加成（喷气推进=空军速度、城守的兵种技能；野地/寇城无科技 → 保持空表）
	defType := ezfyTypeBonus{Atk: map[int]int{}, Speed: map[int]int{}}
	wildLevel := 0
	wildDefCamp := 0 // 野地守军阵营: 1盟军(野地) 2轴心国(寇城), 0无
	var target *model.EzfyCity
	// ★ 2026-10-06 野地/寇城守将（军官池 EzfyCfgGeneral）提升到外层作用域，
	//   战斗引擎的「守方绝地逆袭」用它传参（原来局部在 cfg.OfficerId 块内、从未生效）
	var defGeneral *model.EzfyCfgGeneral
	// ★★ 2026-10-19 玩家附属野地的守军订单（增援6/驻守采集7）提升到外层作用域：
	//   战斗结束后要把**剩余兵力写回这些订单** —— 原来只把损失记进战报，
	//   订单 Troops 仍是满编 → 下次继续打时守军数量不对（用户反馈）。
	var wildDefOrders []model.EzfyOrder
	// ★★ 2026-10-20 附属野地守军主将与归属城（科技/装备/技能加成的来源）。
	//   有玩家守军时，战斗引擎的守方加成必须按「归属城科技 + 主将军官」计算，
	//   不能继续沿用野地/寇城「无科技、无军官」的默认 0。
	var wildDefCommander *model.EzfyOfficer
	wildDefCityID := uint(0)

	// ★ case 0：老数据/异常请求可能没带 target_type，按「野地」处理，
	//   否则会落进 default，导致战报标题变成「侦查报告: 」（目标名为空）。
	switch order.TargetType {
	case 0, 1, 2:
		// 活动目标: 侦查时按活动守军回报情报(不走普通野地配置表)
		// ★ 2026-10-05 名将野地按玩家判定：已抓到守将的玩家 → 该坐标对其是普通野地，
		//   这里与 processArrive 活动结算路由同口径，跳过活动守军走普通野地配置。
		// ★ 2026-10-30 以订单固化/当日配额判定后的 ActType 为准：次数用完后侦查也按普通野地回报。
		if act := h.ezfyOrderActType(uid, order); act > 0 &&
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
		} else if ezfyIsSeaWildRules(ezfyTerrainEx(order.TargetX, order.TargetY)) {
			// ★ 2026-10-04 与地图同口径（Ex 含覆盖表/沿海平原），避免覆盖成岛屿/沿海平原的
			//   海洋格被误当「海野」配置
			// ★ 2026-10-05 岛屿也属于海野 → 岛屿守军走海野配置
			cfgType = 2
		}
		cfg := ezfyCfg.wildland(cfgType, level)
		if cfg == nil {
			h.beginReturn(order, now, 0)
			h.addReport(uid, 2, "战斗报告: 目标不存在",
				fmt.Sprintf("目标(%d,%d)不存在或已被摧毁, 部队已返航。", order.TargetX, order.TargetY), "", order.ID, city.ID)
			return
		}
		wildLevel = level
		// ★★ 2026-10-18 玩家附属野地守军（用户规则）：
		//   该坐标存在**玩家占领**的野地记录（ezfy_wildland.CityId>0）时：
		//   · 守军 = 该野地上的 驻军(增援6) + 采集/驻守采集(7) 部队，按订单实况；
		//   · 没有任何驻守/采集部队 → 守军为 **0**（不再用默认野地配置守军）；
		//   · 无守将/无配置加成（守军是真人部队，按基础值交战，与盟军驻军战同口径）。
		//   中立野地 / AI 寇城（无野地行）仍走原「配置守军」。
		var wildDef model.EzfyWildland
		if h.DB.Where("x = ? AND y = ?", order.TargetX, order.TargetY).Order("id DESC").First(&wildDef).Error == nil &&
			wildDef.CityId > 0 {
			// 自己的野地不会被攻打（createOrder 已拦），能到这里的主人必是他人
			// ★★ 2026-10-20 按**坐标**找守军订单，而不是 `target_id = 野地行 id`：
			//   攻打野地的订单 target_id 常为 0（客户端只传坐标），而守军订单虽然带
			//   野地 id，但野地行可能因征服/重建换过 id → 按 id 会漏掉全部守军，
			//   连带军官/科技加成一起变 0（与 ReportDynamics / fetchIncoming 同口径）。
			//   同时排除 target_type=3（城市增援），旧客户端可能把野地写成 0，
			//   所以用 `<> 3` 而不是 `IN (1,2)`，避免漏掉历史订单。
			// ★★ 2026-10-25 口径收进 ezfyWildDefenderOrderCond：采集队返航归队(status 3)
			//   不再算守军（否则野地永远挂着已回城的幽灵部队，见常量注释）。
			h.DB.Where("target_type <> 3 AND "+ezfyWildDefenderOrderCond+" "+
				"AND target_x = ? AND target_y = ?", order.TargetX, order.TargetY).
				Order("id ASC").Find(&wildDefOrders)
			cnt := map[int]int64{}
			sum := int64(0)
			for i := range wildDefOrders {
				for _, g := range parseGroups(wildDefOrders[i].Troops) {
					if g.Count > 0 {
						cnt[g.TroopId] += g.Count
						sum += g.Count
					}
				}
			}
			if sum <= 0 {
				wildDefOrders = nil // 全是空壳订单 → 视为无守军
			}
			defender = nil
			tids := make([]int, 0, len(cnt))
			for tid := range cnt {
				tids = append(tids, tid)
			}
			sort.Ints(tids)
			for _, tid := range tids {
				defender = append(defender, ezfyUnitGroup{TroopId: tid, Count: cnt[tid]})
			}
			wildDefCamp = 0 // 玩家部队按基础(通用)兵种名展示
			// ★★ 2026-10-19 汇总该野地守军各部队的带队军官（战报/指挥室展示）。
			wildDefOfficerDesc = h.ezfyWildDefOfficersDesc(wildDefOrders)
			// ★★ 2026-10-20 守方是真人部队时，按「野地归属城科技 + 守军主将」计算
			//   军官/科技/装备加成。原来这里只写名字，defBonus/defAtkBonus 等全为 0，
			//   玩家看到的战报就是「守方军官 0%、守方科技 无」（用户反馈）。
			wildDefCityID = uint(wildDef.CityId)
			wildDefCommander = h.ezfyWildDefCommander(wildDefOrders, wildDefCityID)
			// ★ 科技按账号共享（techMap 内部再查到 user_id）；野地归属城若已被放弃/
			//   记录失效导致查不到归属用户，则回退到守军订单的出发城，避免科技整段丢失。
			wildDefTech := h.techMap(wildDefCityID)
			if len(wildDefTech) == 0 {
				for i := range wildDefOrders {
					if cid := uint(wildDefOrders[i].CityId); cid > 0 && cid != wildDefCityID {
						if t := h.techMap(cid); len(t) > 0 {
							wildDefTech = t
							break
						}
					}
				}
			}
			wildDefBonus := h.ezfyWildDefCombatBonus(wildDefTech, wildDefCommander)
			defBonus = wildDefBonus.DefBonus
			defAtkBonus = wildDefBonus.DefAtkBonus
			defSpeedBonus = wildDefBonus.DefSpeedBonus
			defRangeBonus = wildDefBonus.DefRangeBonus
			defHealTech = wildDefBonus.DefHealTech
			defOfficerAtkBonus = wildDefBonus.OfficerAtkBonus
			defEquip = wildDefBonus.Equip
			defSkillBreak = wildDefBonus.SkillBreak
			defTechBreak = wildDefBonus.TechBreak
			defDefBreak = wildDefBonus.DefBreak
			defType = wildDefBonus.Type
			// 指挥室「准备回合」里的【守方军官】：有主将时展示完整加成说明；
			// 旧数据/军官已被删除时回退为原来的名字列表，至少让攻方看到是谁在守。
			defOfficerDesc = wildDefBonus.OfficerDesc
			if defOfficerDesc == "" {
				defOfficerDesc = wildDefOfficerDesc
			}
			targetName = ezfyWildTerrainDisplayName(order.TargetX, order.TargetY, true) +
				strconv.Itoa(wildDef.Level) + "级"
		} else {
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
					// ★ 2026-10-06 守将防御类技能（弧形壁垒/弹幕覆盖）也计入守军防御加成
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
					// ★ 2026-10-08 守将军官行与玩家城城守同口径：**同时**列出「攻击加成」与
					//   「守军防御」（属性部分，都是有效学识加成），技能逐项单列 —— 读者把属性项与
					//   技能项相加即得该守将对守军的全部贡献（原来只写守军防御，攻击那半看不到）。
					officerLv := generalSkillLevel(defGeneral)
					descParts := []string{}
					if guardAttr > 0 {
						descParts = append(descParts, "攻击加成+"+strconv.Itoa(guardAttr)+"%",
							"守军防御+"+strconv.Itoa(guardAttr)+"%")
					}
					for _, s := range generalSkillList(defGeneral) {
						if eff := ezfySkillEffectTextAt(s, officerLv); eff != "" {
							descParts = append(descParts, s+"(Lv."+strconv.Itoa(officerLv)+" "+eff+")")
						}
					}
					defOfficerDesc = defGeneral.Name + " Lv." + strconv.Itoa(defGeneral.Level)
					if len(descParts) > 0 {
						defOfficerDesc += " " + strings.Join(descParts, " ")
					}
				}
			}
		} // ★★ 2026-10-18 中立野地守军分支(else)结束；下面为 case 0/1/2 公共的战利品/名称计算
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
		// ★★ 2026-10-31 黄金洞察（原黄金眼）**不再加战利品**：用户口径「效果不对，它应该是
		//   提高侦察/情报等级」→ 改为 带队侦查加侦察等级、任市长加情报等级
		//   （见 scoutReportBody / ezfyIntelLevel）。这里只保留科技 17「战地征用」。
		rnd = rnd * int64(100+lootTech) / 100
		// ★ 2026-09-25 用户反馈「野地打完获得的资源太少」→ 管理端「二战系统配置 → 野地获取资源倍率」。
		//   放在**所有既有加成之后**做最后一道放大：战地征用照旧生效，倍率再乘上去。
		//   只影响这一处（野地/海野/寇城的战斗战利品），不含驻守采集。
		rnd = ezfyScaleByWildResMult(rnd)
		lootFood, lootSteel, lootOil, lootRare, lootGold = rnd, rnd, rnd, rnd, rnd
		// ★ 2026-10-08 用户规则修正：掠夺**野地/海野/寇城**产出里也包含黄金，
		//   只有掠夺**玩家城市**才不产出黄金（玩家城市的黄金清空在下面 case 3 分支单独处理）。
	case 3:
		var tc model.EzfyCity
		if err := h.DB.First(&tc, order.TargetId).Error; err != nil {
			h.beginReturn(order, now, 0)
			h.addReport(uid, 2, "战斗报告: 目标不存在",
				"目标城市已不存在, 部队已返航。", "", order.ID, city.ID)
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
				h.addReport(uid, 2, "战斗报告: 未宣战", msg, "", order.ID, city.ID)
				return
			}
		}
		targetName = target.Name
		// ★ 排除「不参与」的兵种：被攻击时防御战斗兵种列表不含它们
		//   （友军自身守军在这里构建；盟友驻军队列在下方「驻军串行战」先打完）
		defExclude = h.defExcludeSet(target.ID)
		// ★ 2026-10-28 兵种防守数量：城内有 1000 万、司令部只配 100 万 → 只有 100 万参战
		//   （剩下的照常留在城里，不参战也就不会按本场战损被扣）。
		defCaps = h.defCapMap(target.ID)
		defTech := h.techMap(target.ID)
		// ★★ 2026-10-09 用户规则（「两个号都是满科技，战斗加成理论上应该一样」）：
		//   守方防御**不再叠加城墙**（建筑7 ×5%/级）—— 原来守方比攻方凭空多一截城墙，
		//   满科技的两个号在战报里显示成「攻方 防御+70% / 守方 防御+115%」，玩家以为算错了。
		//   现在与攻方防御同口径（均质装甲 + 永备工事 + 重型机床 + 城守军官），
		//   攻守科技都满级时两边防御加成完全一致。
		//   ⚠️ 城墙建筑本身**保留**（它的「城防兵容量」作用不受影响），只是不再进防御加成。
		// ★ 2026-09-28 重型机床(9)+2%/级：该科技描述是「重装备**攻防**+2%」，攻方那条路径已加，
		//   守方这条原来漏了 → 被攻击时这 2%/级 完全不生效。
		// ★★ 2026-10-09 永备工事(16) 移出**通用**防御 → 改成只对**城防单位**生效
		//   （用户口径「永备工事只对城防单位生效，也是属于兵种单位加成只不过是城防单位类型」），
		//   见下方 defType.Def[ezfyTroopTypeCity]。
		defBonus = defTech[7]*3 + defTech[9]*2
		// 速度：只有机械增压(10)「部队速度」是全体兵种通用；喷气推进(19)「空军速度」是兵种专属
		// （★ 2026-10-08 移到下方 defType —— 原来错误地给全军加速）
		// ★ 2026-10-31 满级 50%：读配置（不再写死每级 2%）
		defSpeedBonus = ezfyTechEffectValueAt(10, defTech[10])
		// 城守: 守城防御 +10% 及 防御/掩体/生命/鼓舞技能
		cityGuard = h.positionOfficer(target.ID, ezfyPositionGuard)
		defBonus += h.officerGuardBonus(cityGuard)
		// ★ 2026-10-09 守方「伤兵回收率」的额外加成 = 守方科技·战地医疗(21) + 城守军官技能·机械改装。
		//   与攻方同口径（攻方见下方 healTech）：兵种修复率 + 科技 + 技能。
		// ★ 2026-10-14 战地医疗按管理端「效果数值」取（原为固定 2%/级），未配置走原公式。
		// ★ 2026-10-15 机械改装回收率由 10%/级 → **5%/级**（名将满级最高 30%）。
		defHealTech = ezfyTechEffectValueAt(21, defTech[21])
		if h.officerHasSkill(cityGuard, "机械改装") {
			defHealTech += 5 * h.officerSkillScale(cityGuard)
		}
		// ★ 2026-10-08 守方兵种专属加成：喷气推进(19)只加空军 + 城守军官的兵种技能
		//   （火炮压制=陆军 / 四指阵列=空军 / 狼群围猎=海军 / 坦克冲锋·闪电疾攻·越岛登陆=陆·空·海速度）
		defType = ezfyTypeBonus{Speed: map[int]int{}, Def: map[int]int{}}
		if v := ezfyTechEffectValueAt(19, defTech[19]); v > 0 {
			defType.Speed[ezfyTroopTypeAir] += v
		}
		// ★★ 2026-10-09 永备工事「只对城防单位生效」→ 挂在城防(4) 的**兵种专属防御**上，
		//   由战斗引擎按被攻击方的兵种 type 取（普通部队不吃，城防单位吃）。
		if v := defTech[16] * 2; v > 0 {
			defType.Def[ezfyTroopTypeCity] = v
		}
		defType = h.officerTypeBonus(cityGuard, defType)
		// ★★ 2026-10-08 修正守方**攻击**加成的科技口径（用户反馈「两个号都是满科技，为什么科技加成不一样」）：
		//
		//	原来用「均质装甲(7) + 重型机床(9) + 永备工事(16)」—— 但均质装甲的配置描述是
		//	**部队防御**，它只该进防御加成，却被当成攻击加成用了；而**动员训练(5) / 枪械改良(6)**
		//	（描述都是「部队攻击」）完全没算 → 守城部队白丢一截攻击加成，
		//	且【守方科技】行比【攻方科技】少这两项（满科技的两个号列出来却不一样）。
		//
		//	现在与攻方口径对齐：枪械改良+3%/级 · 重型机床+2%/级（重装备攻防）
		//	＋ 城守军官攻击加成。动员训练(5) 自 2026-10-21 起只加造兵速度，不再进战斗。
		//	★ 均质装甲(7) 只保留在**防御**加成里（defBonus / defDefBreak），不再重复进攻击。
		//	★ 守方射程加成 = 增程弹药(8)*3（与攻方同口径；永备工事只进城防攻防，不进射程 —— 见下方 defRangeBonus）
		// ★★ 2026-10-09 用户规则：**守方攻击不再额外吃「永备工事」(16)** ——
		//   那是守城方专属的一项，导致满科技的两个号在战报里显示成「攻方 攻击+70% / 守方 攻击+90%」。
		//   现在两边攻击口径完全一致（枪械改良 + 重型机床 + 带队/城守军官）。
		//   永备工事仍保留在**防御**里（defBonus / defDefBreak）—— 攻守双方都吃，不影响对称性。
		// ★ 2026-10-10 修复「装备一样加成不一样」：攻方攻击 = atkBonus + atkEquip.Dmg（装备伤害单独加），
		//   而守方攻击之前漏加了装备/套装伤害（defEquip.Dmg）。这里先算 defEquip（必须在 defAtkBonus 之前赋值），
		//   再补进去 —— 守城部队反击时身上装备/套装的「伤害」加成才真实生效。
		defEquip = h.officerBattleEquipBonus(cityGuard)
		defAtkBonus = defTech[6]*3 + defTech[9]*2 + h.officerBattleBonus(cityGuard) + defEquip.Dmg
		// ★ 2026-10-06 城守军官占守方攻击加成的百分点（战报日志拆解用）
		defOfficerAtkBonus = h.officerBattleBonus(cityGuard)
		// ★ 2026-10-06 守方拆解逐项明细：城守技能 + 守方科技（与 defAtkBonus 同口径）
		defSkillBreak = h.officerSkillsBreak(cityGuard)
		// ★ 2026-10-08 与【攻方科技】用**同一集合、同一顺序**：
		//   满科技的两个号现在列出的科技项完全一致，不会再出现「都是满科技、守方却少两项」的困惑。
		// ★ 2026-10-21 动员训练(5) 已改为造兵速度科技，战斗/战报不再列出。
		defTechBreak = ezfyBonusItems(
			ezfyTechItem("枪械改良", defTech[6]*3),
			ezfyTechItem("均质装甲", defTech[7]*3),
			ezfyTechItem("增程弹药", defTech[8]*3),
			ezfyTechItem("重型机床", defTech[9]*2),
			ezfyTechItem("机械增压", ezfyTechEffectValueAt(10, defTech[10])),
			ezfyTechItem("永备工事", defTech[16]*2),
			ezfyTechItem("喷气推进", ezfyTechEffectValueAt(19, defTech[19])),
		)
		// ★★ 2026-10-08 修复「守方射程加成虚高」（用户反馈「射程加成有问题 守方 50%？」）：
		//   原来是 `增程弹药(8)*3 + 永备工事(16)*2` —— 但永备工事的配置描述是
		//   「**城防攻防**+2% / 提高城防设施攻防」，只加城防的攻与防，**与射程无关**；
		//   攻方那条路径只有增程弹药(8)*3。两边口径不一致 → 满科技的守方凭空多 20% 射程。
		//   现在与攻方对齐：**守方射程 = 增程弹药(8)*3**。
		//   ★ 永备工事仍保留在守方**攻击**（defAtkBonus）与**防御**（defBonus/defDefBreak）里 —— 那是它的正确作用。
		defRangeBonus = defTech[8] * 3
		// ★ 2026-10-06 守方「防御加成」逐项明细（科技/城守属性+技能/装备，
		//   被打行展示「防御加成+N%(科技·均质装甲+30% …)」，与 defBonus 构成同口径）
		// ★ 2026-10-09 城墙项已移除（与 defBonus 同步 —— 守方不再叠加城墙防御）
		// ★★ 2026-10-09 永备工事也移除：它只对**城防单位**生效（见 defType.Def），
		//   不算进通用防御 → 不能列在通用明细里（否则「列了却不生效」）。
		defDefBreak = ezfyBonusItems(
			ezfyTechItem("科技·均质装甲", defTech[7]*3),
			ezfyTechItem("科技·重型机床", defTech[9]*2),
		)
		if cityGuard != nil {
			defDefBreak = append(defDefBreak, ezfyBonusItem{Name: "军官·" + cityGuard.Name, Value: h.officerGuardAttrBonus(cityGuard)})
			for _, s := range h.officerGuardSkillsBreak(cityGuard) {
				defDefBreak = append(defDefBreak, ezfyBonusItem{Name: "军官技能·" + s.Name, Value: s.Value})
			}
		}
		defDefBreak = append(defDefBreak, ezfyTechItem("装备套装", defEquip.Def)...)
		// ★ 传「属性部分」的攻击/防御加成（军事→攻击、有效学识÷2→守军防御），技能由
		//   officerBattleDesc 自己列，否则技能会被算两遍。原来这里硬编码 10，与实际生效值不符。
		// ★ 2026-10-08 城守军官行也补上「攻击加成」：城守的军事属性/攻击技能同样进 defAtkBonus
		//   （守城部队行动时吃），原来只显示守军防御，玩家看不到这部分攻击来源（用户反馈）。
		defOfficerDesc = h.officerBattleDesc(cityGuard, h.officerBaseBonus(cityGuard), "攻击加成",
			h.officerGuardAttrBonus(cityGuard), "守军防御")
	}

	// 侦查: 不战斗只报告情报, 部队随即返航
	// 报告细节按**侦查方无线电侦听(科技12)等级**分级（见 scoutReportBody 注释）：
	//   玩家城市 → 资源/人口民心/建筑/军队/将领/科技/最后在线时间（逐级解锁）
	//   野地寇城 → 守军情况
	if order.OrderType == 1 {
		// ★ 侦查报告：返程时长同样只认 ezfyOneWayTravel
		travel := ezfyOneWayTravel(order)
		order.Status = 2
		order.Result = order.Troops
		order.ReturnTime = now + travel
		// ★★ 2026-10-14 侦察机拦截空战（用户规则）：
		//   被侦查**玩家城市**若有侦察机、且其主人未在司令部把侦察机设为「不参与防御」，
		//   则双方侦察机在目标城市上空**纯侦察机空战**（不涉及其它兵种）。
		//   - 守方按战斗结果损兵（扣城内侦察机 + 按修复率进伤兵营），并收到拦截战报；
		//   - 攻方损兵体现在出战报 result（剩余侦察机返航），被击落也按修复率进伤兵营；
		//   - 攻方侦察机**全军覆没** → 侦查必失败、无情报；有幸存仍按数量概率判定成败。
		battleBlock := "" // 拦截空战战报正文（未拦截 → 空，保持原侦查报告样式）
		allShotDown := false
		if order.TargetType == 3 && target != nil && target.UserID > 0 && target.UserID != uid {
			defMap := h.troopMap(target.ID)
			// ★ 2026-10-28 侦察机也吃「防守数量」：司令部把侦察机防守数量配成 N 时，
			//   升空拦截的最多 N 架（与主战场口径一致，没配 = 全部）。
			defRecon := defCountOf(defMap[ezfyReconPlaneTroopID], defCaps, ezfyReconPlaneTroopID)
			if defRecon > 0 && !defExclude[ezfyReconPlaneTroopID] {
				var atkRecon []ezfyUnitGroup
				for _, g := range attacker {
					if g.TroopId == ezfyReconPlaneTroopID && g.Count > 0 {
						atkRecon = append(atkRecon, g)
					}
				}
				if len(atkRecon) > 0 {
					defReconG := []ezfyUnitGroup{{TroopId: ezfyReconPlaneTroopID, Count: defRecon}}
					ar := ezfySimulateBreak(atkRecon, defReconG, 0, 0, 0, 0, 0, 0, 0,
						ezfyBattleBonus{}, ezfyBattleBonus{}, "", "", 0, 0,
						nil, nil, nil, nil, ezfyBattleExtra{}, 0)
					// ---- 攻方侦察机战损（伤兵营 + 返航剩余） ----
					var dead int64
					for _, g := range ar.AttackerLosses {
						dead += g.Count
					}
					if dead > 0 {
						rate := 10
						if cfg := ezfyCfg.troop(ezfyReconPlaneTroopID); cfg != nil {
							rate = cfg.RepairRate
						}
						wounded := dead * int64(rate) / 100
						if wounded > 0 {
							h.addWounded(city.ID, ezfyReconPlaneTroopID, 0, wounded)
						}
					}
					leftStr := ""
					for _, g := range ar.AttackerLeft {
						if g.Count > 0 {
							leftStr += strconv.Itoa(g.TroopId) + ":" + strconv.FormatInt(g.Count, 10) + ","
						}
					}
					order.Result = strings.TrimSuffix(leftStr, ",")
					// ---- 守方侦察机战损（扣城内 + 伤兵营） ----
					// ★ 2026-10-30 改成原子条件扣减：原来「先查 exist 再写绝对值」，
					//   与同城的返航归队/军工厂完工并发时会互相覆盖（多兵或少兵）。
					for _, g := range ar.DefenderLosses {
						if g.Count <= 0 {
							continue
						}
						h.deductTroopCapped(int64(target.ID), g.TroopId, g.Count)
						rate := 10
						if cfg := ezfyCfg.troop(g.TroopId); cfg != nil {
							rate = cfg.RepairRate
						}
						wounded := g.Count * int64(rate) / 100
						if wounded > 0 {
							h.addWounded(target.ID, g.TroopId, 0, wounded)
						}
					}
					// ---- 空战战报正文（攻守双方各自的兵种变化行） ----
					// ★ 2026-10-21 空战战报的 [攻方]/[守方] 也显示玩家昵称（守方拦截只发生在玩家城）。
					airAtkLabel := h.ezfyReportSideName(uid, city.Name)
					airDefLabel := h.ezfyReportSideName(target.UserID, targetName)
					atkTagC, defTagC := winText(ar.AttackerWin), winText(!ar.AttackerWin)
					airLine := fmt.Sprintf("[%s]攻方:%s\n", atkTagC, airAtkLabel) +
						troopChangeText(groupCounts(atkRecon), groupCounts(ar.AttackerLeft)) +
						fmt.Sprintf("--------------------\n[%s]守方:%s\n", defTagC, airDefLabel) +
						troopChangeText(groupCounts(defReconG), groupCounts(ar.DefenderLeft))
					battleBlock = fmt.Sprintf("公文报告:侦查拦截\n我方侦察机抵达%s[%d，%d]上空时遭敌方侦察机拦截，双方侦察机展开空战，共持续 %d 回合！\n",
						targetName, order.TargetX, order.TargetY, ar.Rounds) + airLine + "\n------------------\n"
					// ---- 守方拦截战报 ----
					// ★ 2026-10-20 修正守方视角：空战是攻方胜时，守方侦察机已被打光，
					//   不能再写「击退敌军侦察 / 敌军侦察机已被击破或击退」——那是攻方胜的相反结果。
					defTitle, defTail := ezfyScoutInterceptDefenderText(ar.AttackerWin)
					defBlock := fmt.Sprintf("公文报告:防空拦截\n敌军侦察机企图侦查我方城市「%s」(%d,%d)，我军侦察机升空拦截，双方侦察机展开空战，共持续 %d 回合！\n",
						target.Name, order.TargetX, order.TargetY, ar.Rounds) + airLine + "\n" + defTail
					h.addReport(target.UserID, 2, defTitle, defBlock, "", order.ID, target.ID)
					if enemyDeadOf(ar.AttackerLeft) <= 0 {
						allShotDown = true
					}
				}
			}
		}
		h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
			Updates(map[string]interface{}{"status": 2, "result": order.Result, "return_time": order.ReturnTime})
		// ★ 2026-10-02 用户规则：侦查成功率按**侦察机数量**概率（一架必成不对），
		//   失败只发失败战报、不获取任何情报（部队照常返航，不损兵）。
		// ★ 2026-10-14 拦截空战：攻方侦察机被全部击落 → 必失败；有幸存仍按数量概率判定。
		if allShotDown {
			h.addReport(uid, 1, "侦查失败: "+targetName,
				battleBlock+fmt.Sprintf("我方侦察机全部被击落，未获取到任何情报。\n"), "", order.ID, city.ID)
			return
		}
		if !ezfyReconSucceed(order) {
			h.addReport(uid, 1, "侦查失败: "+targetName,
				battleBlock+fmt.Sprintf("公文报告:侦查失败\n我方一支部队对%s[%d，%d]的侦查未能成功，未获取到任何情报，仅存侦察机已返航。\n",
					targetName, order.TargetX, order.TargetY), "", order.ID, city.ID)
			return
		}
		// 成功：若有拦截，把空战正文接在情报之前
		wildOwned := false // ★★ 2026-10-18 玩家附属野地：报告不显示配置守将（守军=真人驻军/采集部队）
		if order.TargetType == 1 || order.TargetType == 2 {
			var wd model.EzfyWildland
			if h.DB.Where("x = ? AND y = ?", order.TargetX, order.TargetY).Order("id DESC").First(&wd).Error == nil && wd.CityId > 0 {
				wildOwned = true
			}
		}
		scoutBody := h.scoutReportBody(uid, order, targetName, target, defender, wildOwned)
		if battleBlock != "" {
			scoutBody = strings.Replace(scoutBody, "侦查过程中未受到任何阻拦。",
				"侦查过程中在目标城市上空与敌方侦察机交战，击破拦截后继续侦查。"+battleBlock, 1)
		}
		h.addReport(uid, 1, "侦查报告: "+targetName, scoutBody, "", order.ID, city.ID)
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
				// ★ 2026-10-08 驻军战也透传完整明细（原来包装函数一律传 nil）：
				//   否则准备回合没有【攻方科技】【攻方套装】行，套装效果还会被并进【攻方装备】行
				//   —— 与主城战格式不一致（用户反馈「战报、指挥模块展示不全」）。
				//   守方是盟友驻军（无科技/无城墙/无军官/无装备），守方明细一律留空。
				ezfyBattleExtra{
					AtkSkills: atkSkillBreak, AtkTechs: atkTechs,
					AtkDefBonus: atkDefBonus, AtkDefBreak: atkDefBreak,
					AtkSet:          h.officerSetEquipBonus(leadOfficer),
					AtkSetDesc:      h.officerSetsDesc(leadOfficer),
					AtkOfficerSkill: h.officerSkillBattleBonus(leadOfficer),
					// ★ 2026-10-08 兵种专属加成（喷气推进=空军速度等）
					AtkType: atkType,
					// ★ 2026-10-31 弧形壁垒/火炮压制「每3回合发动1回合」在驻军战里同样生效
					AtkTimed: h.officerTimedSkills(leadOfficer),
				},
				ezfyGarrisonBreakPctCfg())
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
			// ★ 2026-10-08 归属**攻方出发城**（原来传 target.ID = 被打的城，攻方按自己的城市
			//   过滤战报时匹配不上、标题里又没有坐标/自己城名 → 这条战报攻方根本看不到）。
			h.addReport(uid, 2, "战斗报告: 击溃"+gcityName+"的驻军",
				fmt.Sprintf("我方部队在%s(%d,%d)击溃了来自%s的盟军驻军!\n%s",
					targetName, order.TargetX, order.TargetY, gcityName,
					lossText(gbr.DefenderLosses)),
				gDetail, int64(order.ID), city.ID)
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
			// ★ 2026-10-28 每兵种按「防守数量」封顶：min(城内数量, 司令部配置)，未配 = 全部。
			for tid, count := range h.troopMap(target.ID) {
				if count > 0 && !defExclude[tid] {
					defender = append(defender, ezfyUnitGroup{TroopId: tid, Count: defCountOf(count, defCaps, tid)})
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
		// ★ 2026-10-08 同上：归属攻方出发城（原来传 target.ID，攻方在自己的城市视角看不到）
		h.addReport(uid, 2, "战斗报告: 进攻受阻(驻军拦截)",
			fmt.Sprintf("我方部队进攻%s(%d,%d)时被盟军驻军拦截, 未能攻入城市, 部队已返航。",
				targetName, order.TargetX, order.TargetY), "", order.ID, city.ID)
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
		// ★ 守方「绝地逆袭」生效回合数：玩家城 = 城守军官；有玩家守军的附属野地 =
		//   守军主将；中立野地/寇城 = 配置守将（无守将 → 0）。
		defCounterRounds := 0
		if target != nil {
			defCounterRounds = h.officerCounterRounds(cityGuard)
		} else if wildDefCommander != nil {
			defCounterRounds = h.officerCounterRounds(wildDefCommander)
		} else {
			defCounterRounds = generalCounterRounds(defGeneral)
		}
		// ★ 守方套装/技能展示军官：玩家城 = 城守军官；有玩家守军的附属野地 = 守军主将；
		//   中立野地/寇城守将是 AI 将军，无装备、无套装。
		defSetOfficer := cityGuard
		if target == nil {
			defSetOfficer = wildDefCommander
		}
		st := ezfyNewBattleState(attacker, defender,
			atkBonus, defBonus, defAtkBonus, atkSpeedBonus, defSpeedBonus,
			atkRangeBonus, defRangeBonus,
			atkEquip, defEquip,
			h.officerSetEquipBonus(leadOfficer), h.officerSetEquipBonus(defSetOfficer),
			h.officerSetsDesc(leadOfficer), h.officerSetsDesc(defSetOfficer),
			atkOfficerDesc, defOfficerDesc,
			officerBonus, defOfficerAtkBonus, // 军官占的「攻击加成」百分点（战报日志拆解用）
			// ★ 2026-10-06 军官加成里「技能」占的百分点（拆解单独展示「军官技能+N%」）
			// ★★ 2026-10-20 守方用 defSetOfficer（城守或附属野地守军主将），与套装展示同口径。
			h.officerSkillBattleBonus(leadOfficer), h.officerSkillBattleBonus(defSetOfficer),
			// ★ 2026-10-06 技能/科技逐项明细（战报展示「军官技能·尖兵猛攻+N%」「科技·增程弹药+N%」）
			atkSkillBreak, defSkillBreak, atkTechs, defTechBreak, defDefBreak,
			// ★ 2026-10-07 攻方「防御加成」（军官属性+防御技能+装备 Def）
			atkDefBonus, atkDefBreak,
			atkTargets, defTargets, atkMoves, defMoves,
			// ★ 军官技能「绝地逆袭」随等级升级：生效前N回合（攻方带队/守方城守或野地守将各自判定）
			h.officerCounterRounds(leadOfficer), defCounterRounds,
			h.ensureProfile(uid).Camp, defCamp,
			// ★ 2026-10-08 兵种专属加成（喷气推进=空军速度、军官兵种技能）
			atkType, defType)
		// ★★ 2026-10-23 名将组合技能（系统专属战法）：攻方带队名将 / 守方城守（或野地守将）。
		//   普通军官没有 GeneralId → comboSkillsOfOfficer 返回 nil，等于没这功能，不影响老战斗。
		//   ★ 2026-10-24 多技能：名将可配多个，comboRuntimes 包成运行时（每回合独立掷骰）。
		st.AtkCombos = comboRuntimes(h.comboSkillsOfOfficer(leadOfficer))
		if target != nil {
			st.DefCombos = comboRuntimes(h.comboSkillsOfOfficer(cityGuard))
		} else if wildDefCommander != nil {
			st.DefCombos = comboRuntimes(h.comboSkillsOfOfficer(wildDefCommander))
		} else if defGeneral != nil {
			st.DefCombos = comboRuntimes(ezfyComboOfGeneralCfg(defGeneral, defGeneral.Level))
		}
		st.appendComboHead()
		// ★★ 2026-10-31 普通军官「每3回合发动1回合」技能（弧形壁垒 / 火炮压制）：
		//   快照里的加成是发动回合的满额值，引擎按回合扣减（见 ezfy_skill_timed.go）。
		st.AtkTimed = h.officerTimedSkills(leadOfficer)
		if target != nil {
			st.DefTimed = h.officerTimedSkills(cityGuard)
		} else if wildDefCommander != nil {
			st.DefTimed = h.officerTimedSkills(wildDefCommander)
		} else {
			st.DefTimed = generalTimedSkills(defGeneral)
		}
		// ★ 2026-10-24 城墙耐久：只有「征服玩家城(3)」启用；「掠夺(2)」完全忽略城墙
		//   （不打耐久、也不吃城墙防御/射程加成）。上限 = 围墙等级²×20000。
		// ★ 2026-10-24 用户规则「每场都恢复满」：开战时城墙耐久**无条件补满**，
		//   不打跨场持久化损伤 —— 上一场打掉的耐久不带到下一场，城墙是一道每场重刷的屏障。
		if target != nil && order.OrderType == 3 {
			if lv := h.buildingLevel(target.ID, 7); lv > 0 {
				maxHp := ezfyWallDurabilityMax(lv)
				hp := maxHp
				target.WallHp = hp
				h.DB.Model(&model.EzfyCity{}).Where("id = ?", target.ID).Update("wall_hp", hp)
				st.WallEnabled = true
				st.WallHp = hp
				st.WallHpMax = maxHp
				st.WallDefBonusMax = ezfyWallDefBonusMax(lv)
				// ★★ 2026-10-31 用户规则：城墙**不再提供远程射程加成**（战报头部也去掉该项），
				//   城墙只剩「耐久 + 城防防御」两条，防御仍只对城防兵种生效（见 wallBonusFor）。
				// ★★ 2026-10-31 用户口径：「守城防御」有歧义（听起来像给全城加防）→ 战报改称「**城防防御**」。
				st.Head = append(st.Head, fmt.Sprintf("【城墙】耐久 %d/%d，城防防御+%d%%",
					hp, maxHp, ezfyWallDefBonusMax(lv)))
				// ★ 2026-10-24 守方无兵但有城墙：不能 0 回合直接判胜，仍需打完城墙耐久
				//   （前提：攻方有兵；否则没人拆墙）。墙破 → 攻方胜；回合耗尽未破 → 平局（守方视为守住）。
				//   城墙没破（hp=0）或没城墙（本分支不进来）仍按原 0 回合直接结算。
				if st.Done && len(st.Defenders) == 0 && len(st.Attackers) > 0 {
					st.Done, st.AttackerWin, st.Draw = false, false, false
				}
			}
		}
		// ★★ 2026-10-07 「自动战斗」配置（出征页可配，见 EzfyOrder.AutoBattle）：
		//   · 打野地 / AI 寇城 → 默认「是」：抵达即自动打完，无需指挥（守方不是真人）；
		//   · 打玩家城市 → 强制「否」：抵达后开战场进指挥室，等玩家部署守军。
		//   ★ 玩家城的例外（用户明确要求）：即使配置是「否」，只要**攻守双方当时都不在线**，
		//     就没人能指挥 → 直接自动结算。否则部队会一直占着目标，把后面排队的人堵死。
		autoBattle := order.AutoBattle == 1
		// ★★ 2026-10-07 安全加固（防「出征丢兵」）：**玩家城（target_type=3）一律不自动战斗**。
		//   原因：`auto_battle` 列的 default 是 1，部署时 AutoMigrate 会把**所有历史订单**填成 1；
		//   若直接信任该字段，打玩家城的**老订单**就会「到达即自动打完」→ 玩家没机会进指挥室
		//   → 部队按 AI 自动对局全灭 = 丢兵。
		//   所以这里**不信任字段值**，按 target_type 重新判定：
		//     玩家城 → 恒不自动（只有攻守双方都离线时才自动，见下）；
		//     野地 / AI 寇城 → 按字段（默认 1 = 自动）。
		// ★★ 2026-10-19 用户规则：**攻击有玩家驻守的附属野地**同样必须进指挥室。
		//   原来只按 target_type 判（野地恒按字段、默认自动）→ 打别人有驻军的野地
		//   一到就自动打完，守方（真人）根本没机会指挥。
		//   现在只要该野地上有活跃守军订单（增援6/驻守采集7，见上面的 wildDefOrders），
		//   就跟打玩家城市一个口径：强制不自动。
		hasPlayerDefender := len(wildDefOrders) > 0
		if order.TargetType == 3 || hasPlayerDefender {
			autoBattle = false
		}
		// ★★ 2026-10-19 人机没有指挥能力：打**野地**（含玩家驻军）仍自动结算，
		//   否则人机部队会进指挥室干等 20 分钟（40 回合 × 30 秒）才出结果，
		//   期间还占着攻击额度。打玩家城维持原逻辑（进指挥室，给真人守方机会）。
		if city.IsBot == 1 && order.TargetType != 3 {
			autoBattle = true
		}
		// ★★ 2026-10-08 离线自动结算**只对打玩家城（target_type=3）生效**。
		//   原因：180 秒在线窗口容易被放过（行军久、或结算那一刻不在操作），
		//   打野地 / AI 寇城时玩家在出征页选了「否」却仍被静默自动打完 →
		//   「选了否也不进指挥室」（用户反馈的 bug）。
		//   所以这里把「攻守双方都不在线 → 自动」的兜底**圈死在玩家城**：
		//     · 玩家城 → 攻守都不在线才自动（真人守方，避免占目标堵后面的人）；
		//     · 野地 / AI 寇城 → 尊重出征页所选，选了「否」就开战场进指挥室，
		//       绝不因在线窗口过期而偷偷自动打完。
		// ★★ 2026-10-19 有玩家驻军的附属野地同样适用「双方都不在线才自动」：
		//   否则攻方离线、守方也离线时，战场会一直挂着把目标堵死。
		// ★★ 2026-10-20 用户规则「人机也是真人，按真人算的，只不过是机器操控人的帐号」：
		//   人机由后台逐回合指挥，视为**一直在线**（见 ezfyBattleSideOnline）。
		//   这样玩家打人机城、人机打玩家城都会正常进指挥室，不会被
		//   「攻守双方都离线 → 自动结算」提前把仗打完。
		if !autoBattle && !h.ezfyBattleSideOnline(uid) {
			defOnline := false
			if order.TargetType == 3 {
				defOnline = target != nil && target.UserID > 0 && h.ezfyBattleSideOnline(target.UserID)
			} else if hasPlayerDefender {
				defOnline = h.ezfyBattleSideOnline(wildDefOrders[0].UserID)
			}
			if (order.TargetType == 3 || hasPlayerDefender) && !defOnline {
				autoBattle = true
			}
		}
		if autoBattle {
			br = h.ezfyBattleAutoFinish(uid, order, st, targetName, now)
		} else if ezfyOrderTargetBusy(h, order, int64(order.ID)) {
			order.Status = ezfyOrderStatusWaiting
			h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
				Update("status", ezfyOrderStatusWaiting)
			return
		} else if b := h.ezfyBattleStart(uid, order, st, targetName, now); b != nil {
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
					fmt.Sprintf("军情: 敌军抵达 %s(%d,%d)", city.Name, city.X, city.Y),
					fmt.Sprintf("敌方部队已抵达我方城市「%s」(%d,%d) 附近，双方即将交战！\n来袭方城市：%s(%d,%d)\n请到「军情 → 军队动态」进入[指挥]部署守军。",
						target.Name, order.TargetX, order.TargetY, city.Name, city.X, city.Y),
					"", 0, target.ID)
			}
			// ★★ 2026-10-19 附属野地有玩家守军（增援6/采集7）时，守方同样是真人：
			//   敌军抵达即给野地主人发「军情警报」，并提示到「军情 → 军队动态」[指挥]，
			//   与打玩家城市同口径（原来守方完全不知道，等打完才知道）。
			if hasPlayerDefender && (order.TargetType == 1 || order.TargetType == 2) {
				ownerUID := wildDefOrders[0].UserID
				if ownerUID > 0 && ownerUID != uid {
					h.addReport(ownerUID, 6,
						fmt.Sprintf("军情: 敌军抵达 %s(%d,%d)", city.Name, city.X, city.Y),
						fmt.Sprintf("敌方部队已抵达我方附属野地「%s」(%d,%d) 附近，双方即将交战！\n来袭方城市：%s(%d,%d)\n请到「军情 → 军队动态」进入[指挥]部署守军。",
							targetName, order.TargetX, order.TargetY, city.Name, city.X, city.Y),
						"", order.ID, wildDefOrders[0].CityId)
				}
			}
			// ★ 「等待指挥」不要放进战斗报告列表 —— 战斗还没结束，战报应当是**结果**。
			//   部队状态在「军情 → 军队动态 / 出征队列」里已显示「战斗中 + [指挥]」，
			//   再发一条战报只会把战斗报告列表搅乱。
			return
		} else {
			// 开战场失败（极端情况：写库异常）→ 兜底走老流程直接模拟，绝不让部队卡住
			for !st.Done {
				st.Step(nil, nil)
			}
			br = st.Result()
		}
	}
	win = br.AttackerWin
	draw := br.Draw
	// ★ 2026-10-24 城墙耐久**每场恢复满**（用户规则）：开战时已无条件补满（见上方初始化），
	//   这里不再把打掉的耐久写回 —— 城墙不跨场持久化损伤。

	// ★★ 风云榜（★ 2026-10-14）：按自然周统计**玩家 vs 玩家**战绩（灭敌/战损/场次/胜场）。
	//   只记打玩家城（target_type=3）的一场完整对局 —— 这里是攻守双方战果齐备、
	//   且 processArrive 顶部 CAS 保证只结算一次的汇聚点（自动战斗/指挥室两种路径都到此）。
	//   野地 / AI 寇城 / 活动目标（target=nil 或 target_type!=3）不算 PVP。
	// ★★ 2026-10-26 本场风云榜「积分明细」行 id + 双方战报 id：明细在下面先写，
	//   战报在结算末尾才写 → 拿到战报 id 后回填到明细（管理端「积分明细 → 查看战报」）。
	var pvpAtkDetailID, pvpDefDetailID, pvpAtkReportID, pvpDefReportID uint
	if order.TargetType == 3 && target != nil && target.UserID > 0 && target.UserID != uid {
		pvpAtkDetailID, pvpDefDetailID = h.accumPvpWeek(now, uid, target.UserID, br, win, draw)
	}

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
	// ★★ 2026-10-28 守方军官经验**只给城守**（用户明确口径：「没城守就不给经验」）：
	//
	//	曾经加过「没有城守时退到守军主将（城内军事最高的在职军官）」的兜底，用户否决 →
	//	守方经验/战报军官行一律只认 position=2 的城守；没任命城守 = 守城没有军官带队 = 不发经验。
	//	（战报/声望照常给，只是没有「军官经验」这一行。）
	defExpOfficer := cityGuard
	defExp := int64(0)
	// 守方：拿到**城守**的城才有（没城守不发经验；中立野地/寇城无玩家军官，自然跳过）
	if defExpOfficer != nil && target != nil {
		// ★★ 2026-10-19 口径修正：守方的「击杀数」= **攻方战损**(br.AttackerLosses)。
		//   原来传 br.DefenderLosses（守方自己人的损失）→ 守方杀敌越多、经验反而按自己死伤算，
		//   与上一条注释「按击杀敌军数衡量」自相矛盾（攻方那一处传的才是对的，见下面 atkExp）。
		raw := ezfyOfficerBattleExp(enemyDeadOf(br.AttackerLosses), !win)
		// ★ 2026-10-28 用**实际到账**经验（满级为 0）→ 战报不再写「+70 但经验不动」的假数字
		defExp = h.addOfficerExp(target, defExpOfficer.ID, raw)
	}
	// 攻方：带队军官无论胜负都给经验（原来只在 if win 分支里给，
	// 打输的部队回来军官一点经验都没有，与用户规则①不符）。
	// ★ 这里只算数值、不写库 —— 写库放在下面战报正文里做：
	//   胜/败两个分支各自往战报追加「军官经验+N」后再调用 addOfficerExp。
	atkExp := int64(0)
	if leadOfficer != nil {
		atkExp = ezfyOfficerBattleExp(enemyDeadOf(br.DefenderLosses), win)
	}

	// ★★ 2026-10-19 附属野地的「守方战报」（用户：「自己附属野地被征服、被掠夺、被侦查
	//   战斗报告也要能看」）：
	//   野地不是 EzfyCity，原来只有 target_type=3（玩家城市）才写守方报告 →
	//   野地主人被打完毫不知情（只有守军被歼灭时才有一条「附属野地失守」）。
	//   ★ 必须**在结算前**记下主人：征服成功会走 ezfyClearWildlandAt 清掉旧野地记录，
	//     那之后再按坐标查就查不到原主人了。
	var wlOwnerUID uint
	var wlOwnerCityID int64
	if order.TargetType == 1 || order.TargetType == 2 {
		wlOwnerUID, wlOwnerCityID = h.ezfyWildlandOwnerOf(order.TargetX, order.TargetY, uid)
	}

	// ★ 2026-10-21 战报 [攻方]/[守方] 行改为**玩家昵称**（用户：不要城市名）：
	//   攻方 = 玩家昵称；守方 = 玩家城城主 / 有主附属野地的城主；
	//   无主野地/寇城仍用「野地名+等级」（如 渔场10级）。
	profile := h.ensureProfile(uid)
	atkLabel := strings.TrimSpace(profile.Nickname)
	if atkLabel == "" {
		atkLabel = city.Name
	}
	defLabel := targetName
	if order.TargetType == 3 && target != nil {
		defLabel = h.ezfyReportSideName(target.UserID, targetName)
	} else if wlOwnerUID > 0 {
		defLabel = h.ezfyReportSideName(wlOwnerUID, targetName)
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
		defLabel, order.TargetX, order.TargetY,
		ezfyOrderTypeName(order.OrderType), br.Rounds, battleOutcomeText(win, draw))

	report += fmt.Sprintf("军衔声望:%d\n", profile.Prestige)
	// ★★ 2026-10-19 用户反馈「攻守军官堆在抬头，看不出哪一方的」：
	//   军官行不再集中写在「军衔声望」下方，改为**各自挂在 [攻方]/[守方] 标题下面**：
	//     [胜]攻方:再战辉煌基地
	//     军官:斯大林(350级)
	//     航母 244900->244880(-20)
	//     --------------------
	//     [败]守方:海底森林10级
	//     军官:Stalin(约瑟夫·斯大林)(350级)
	//     航母 10000->0(-10000)
	//   攻方军官 = 带队军官(leadOfficer)；守方军官 = 城守(cityGuard) 或
	//   野地驻军带队军官(wildDefOfficerDesc，增援6/采集7)。

	atkBefore := groupCounts(attacker)
	atkAfter := groupCounts(br.AttackerLeft)
	// ★ 2026-09-24 40 回合平局时双方标签都显示 [平] 而不是胜/败
	atkTag, defTag := winText(win), winText(!win)
	if draw {
		atkTag, defTag = "平", "平"
	}
	report += fmt.Sprintf("[%s]攻方:%s\n", atkTag, atkLabel)
	if leadOfficer != nil {
		report += "军官:" + officerReportDesc(leadOfficer) + "\n"
	}
	report += troopChangeText(atkBefore, atkAfter)
	report += fmt.Sprintf("--------------------\n[%s]守方:%s\n", defTag, defLabel)
	// ★★ 2026-10-19 守方军官跟在「[败]守方:」标题下（玩家城=城守；附属野地=驻军带队军官）。
	// ★ 2026-10-28 没有城守时显示「守军主将」（defExpOfficer，见上方经验归属），
	//   否则守方军官拿了经验、战报里却看不到人（玩家会以为没给）。
	if defExpOfficer != nil {
		report += "军官:" + officerReportDesc(defExpOfficer) + "\n"
	}
	if wildDefOfficerDesc != "" {
		report += "军官:" + wildDefOfficerDesc + "\n"
	}
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
	detail += fmt.Sprintf("[%s]攻方:%s\n", atkTag, atkLabel)
	if leadOfficer != nil {
		detail += "军官:" + officerReportDesc(leadOfficer) + "\n"
	}
	detail += troopChangeText(atkBefore, atkAfter)
	detail += fmt.Sprintf("--------------------\n[%s]守方:%s\n", defTag, defLabel)
	if defExpOfficer != nil {
		detail += "军官:" + officerReportDesc(defExpOfficer) + "\n"
	}
	if wildDefOfficerDesc != "" {
		detail += "军官:" + wildDefOfficerDesc + "\n"
	}
	detail += troopChangeText(defBefore, defAfter)
	detail += "[双方兵力]"

	// 攻方战损: 按修复率入伤兵营
	losses := br.AttackerLosses
	var deadCount int64
	for _, g := range losses {
		deadCount += g.Count
	}
	// ★ 2026-10-14 战地医疗按管理端「效果数值」取（原为固定 2%/级），未配置走原公式。
	healTech := ezfyTechEffectValueAt(21, atkTech[21])
	// 带队军官「机械改装」技能: 战后伤兵恢复 +5%/级（★ 2026-10-15 由 10%/级下调，最高 30%）
	if h.officerHasSkill(leadOfficer, "机械改装") {
		// ★ 2026-10-06 技能随军官等级自动升级：恢复加成也随等级 ×N
		healTech += 5 * h.officerSkillScale(leadOfficer)
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

	// 防守方(玩家城市)战损扣除 + 守方伤兵入营 + 逃兵
	//
	// ★★ 2026-10-09 「pvp 伤兵也是入伤兵营、回收比例也要有科技加成、军官技能」：
	//
	//	原来守方战损**只扣兵、不进伤兵营**（仅「攻方获胜」时按 ezfyDeserterRate 进逃兵营）——
	//	打一场 PvP，守方掉几万兵一个都回不来。现在与攻方同口径：守方战损按
	//	「兵种修复率 + 守方科技·战地医疗 + 城守军官技能·机械改装」进伤兵营（defHealTech）。
	if order.TargetType == 3 && target != nil {
		for _, g := range br.DefenderLosses {
			if g.Count <= 0 {
				continue
			}
			// ★ 2026-10-30 原子条件扣减（原来先读再写绝对值，并发会覆盖同城的加兵/扣兵）
			h.deductTroopCapped(int64(target.ID), g.TroopId, g.Count)
		}
		// 守方伤兵入营（按守方自己的修复率口径，与攻方对称）
		for _, g := range br.DefenderLosses {
			if g.Count <= 0 {
				continue
			}
			rate := 10
			if cfg := ezfyCfg.troop(g.TroopId); cfg != nil {
				rate = cfg.RepairRate
			}
			rate += defHealTech
			wounded := g.Count * int64(rate) / 100
			if wounded > g.Count {
				wounded = g.Count
			}
			if wounded > 0 {
				h.addWounded(target.ID, g.TroopId, 0, wounded)
				defRepairedTotal += wounded
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
	// ★ 2026-10-09 战报尾部的「回收比例」行（**伤兵回收占比**，见 ezfyHealPctLine）：
	//   「谁看展示谁的」→ 攻方视角 / 守方视角**各算一份**，各自的战报用各自那份。
	//   （攻方战损 = br.AttackerLosses；守方仅玩家城 PvP 才有伤兵数据。）
	healLineAtk := ezfyHealPctLine(repairedTotal, br.AttackerLosses)
	healLineDef := ezfyHealPctLine(defRepairedTotal, br.DefenderLosses)

	// 携带容量(剩余部队负重)
	// ★ 2026-09-28 同样统一走 ezfyCarryCapOf（含「标准化装卸」加成），
	//   否则掠夺时「能搬走多少」会比部队真实负重少，玩家看到战利品被无理由截断。
	carry := h.ezfyCarryCapOf(left, city.ID)

	targetProtected := false
	if order.TargetType == 3 && target != nil {
		targetProtected = h.hasCityEffect(target.ID, 2)
	}
	wareNote := ""
	recyclePct := 0 // 战报里的「回收比例」(玩家城按掠夺比例)
	// ★ 野地/寇城战报里「回收比例:0%」看着像 bug。
	//   野地没有仓库保护额度，战利品是整份资源 → 回收比例就是 100%。
	if win && (order.TargetType == 1 || order.TargetType == 2) {
		recyclePct = 100
	}
	prestigeGain := 0

	// ★★ 2026-10-19 用户规则：攻击**有归属(玩家附属)**的野地**不产出任何战利品** ——
	//   野地配置产出（前面按 cfg.ResMin/ResMax 算好的 rnd）+ 守军缴获一律不计，战果全 0。
	//   原因：「不然有人靠这个刷资源」—— 有主野地打完仍归主人，可被反复攻打刷战利品。
	//   中立野地 / AI 寇城（wlOwnerCityID=0）照旧产出，不受影响。
	//   ⚠️ 这里把 loot 清零后，下面「守方战报」仍会如实展示战果为 0。
	playerOwnedWild := (order.TargetType == 1 || order.TargetType == 2) && wlOwnerCityID > 0
	if playerOwnedWild {
		lootFood, lootSteel, lootOil, lootRare, lootGold = 0, 0, 0, 0, 0
		recyclePct = 0 // 无战利品 → 战报「掠夺比例」也应为 0（原野地口径写死 100%）
	}

	// ★★ 2026-10-19 玩家附属野地守军军官（增援6/采集7 各带一名）也拿战斗经验
	//   （用户：「军官经验…应该有的」）—— 原来只有玩家城城守有经验，野地驻军军官白打一场。
	//   守方口径与城守统一：击杀数 = **攻方战损**，胜负加成取守方(!win)；按归属城+军官名
	//   回查实例逐个发放（订单只存名字，与 ezfyWildDefOfficersDesc 同口径）。
	wildDefExp := int64(0)
	if len(wildDefOrders) > 0 && wlOwnerCityID > 0 {
		wildDefExp = ezfyOfficerBattleExp(enemyDeadOf(br.AttackerLosses), !win)
		h.addWildGarrisonOfficerExp(wildDefOrders, wlOwnerCityID, wildDefExp)
	}

	// ★★ 2026-10-19 野地驻军「被打掉一部分」→ 剩余兵力必须写回订单。
	//   原来只有攻方**全胜**时统一歼灭（status=4），攻方被打退/打平时守军的战损
	//   只出现在战报里，订单 Troops 仍是满编 → 下次继续打时守军数量又变回原来的满编。
	//   攻方胜时不在这里处理（守军由 ezfyDestroyWildlandDefenders 统一歼灭）。
	if len(wildDefOrders) > 0 && !win {
		h.ezfyWriteBackWildGarrisonLeft(wildDefOrders, br.DefenderLosses)
	}

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
			// ★ 2026-10-08 战报显式带**出发城 id**：原来只传 orderId → city_id=0，
			//   军情页按当前城过滤时只能靠「标题坐标反查订单」兜底（订单查不到就丢）。
			//   这里与守方战报（传 target.ID）口径一致，归属出发城 city.ID。
			h.addReport(uid, 3, repTitle, report, detail, order.ID, city.ID)
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
		// ★★ 2026-10-18 攻方战胜 **玩家附属野地** → 歼灭该野地守军(驻军+采集部队)：
		//   全部守军订单置阵亡(4) + 采集负重清零（采集资源随之损失, 不入野地主人城市）。
		//   掠夺/征服共用；在途采集队(4)不在此列（还没驻守到地, 由 ezfyClearWildlandAt 转返航）。
		if order.TargetType == 1 || order.TargetType == 2 {
			// ★★ 2026-10-19 用户规则：有归属(玩家)的野地不产出战利品 → 歼灭守军时，
			//   采集队身上已采的资源也**随之损失**，不并入攻方 loot（否则可反复刷资源）。
			//   中立野地/寇城不经过本函数，其战利品不受影响。
			h.ezfyDestroyWildlandDefenders(order.TargetX, order.TargetY, now)
		}
		if targetProtected && order.TargetType == 3 {
			report += "\n目标城市处于免战保护期, 无法掠夺资源!"
		}
		if order.TargetType == 3 && target != nil {
			// 玩家城市: 掠夺比例10%+战地征用, 上限50%
			// ★★ 2026-10-31 黄金洞察（原黄金眼）不再加掠夺率（它已改为加侦察/情报等级）
			lootRate := 10 + atkTech[17]*2
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

		// ★★ 2026-10-09 野地/寇城战斗**胜利掉落**：完全按「地图管理 → 野地类型」里手动配的
		//   宝物掉落(treasure) + 商城道具掉落(drop_items) 走。
		//   ⚠️ 原来这段只写在下面的「征服成功占领」分支里 → 玩家打野地/寇城通常是**掠夺(2)**，
		//      配置里写的宝物一个都不掉（用户反馈「都不掉这些」）。现在掠夺/征服都走这里。
		//   ★ 2026-10-09 掉到的宝物/商城道具同时往系统频道播报（「掉落宝物 / 商城道具掉落
		//     都要播报」），targetName（「海底森林2级」）一并传进去，播报文案与战报标题同源。
		if order.TargetType == 1 || order.TargetType == 2 {
			report += h.wildlandConfigLoot(uid, city, order, wildLevel, targetName)
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
				report += h.battleStatsTail(uid, pg, 0, healLineAtk)
				// ★ 2026-10-08 补出发城 id（原来 city_id=0，按当前城过滤时只能靠坐标反查兜底）
				h.addReport(uid, 2, reportType+": "+targetName+
					"("+strconv.Itoa(order.TargetX)+","+strconv.Itoa(order.TargetY)+")", report, detail, order.ID, city.ID)
				// ★ 2026-10-19 野地上限放弃占领 ≠ 没打：战利品已抢走、守军已歼灭，
				//   野地主人必须收到「被掠夺/被征服」报告（这个 return 会跳过函数末尾的统一写报）
				h.addWildlandOwnerReport(order, wlOwnerUID, wlOwnerCityID, city, targetName, true, detail, now,
					br, atkTag, defTag, atkBefore, atkAfter, defBefore, defAfter, leadOfficer, wildDefOfficerDesc, wildDefExp)
				h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
					Updates(map[string]interface{}{"status": order.Status, "result": order.Result, "return_time": order.ReturnTime})
				return
			}
			wildType := 1
			// ★ 2026-10-04 与地图同口径（Ex 含覆盖表/沿海平原）
			// ★ 2026-10-05 岛屿也属于海野 → 占领后记成海野（采集走海野系数）
			if order.TargetType == 2 || ezfyIsSeaWildRules(ezfyTerrainEx(order.TargetX, order.TargetY)) {
				wildType = 2
			}
			// ★★ 2026-10-16 修复「征服他人附属野地后原记录残留」（一个坐标只允许一条野地记录）：
			//   原来这里**无条件 Create 新行** —— 该坐标若早已被别人占领（表里已有 city_id=他人城的行），
			//   就会留下同坐标多条记录（线上实测 153 个坐标重复、81 个坐标归属冲突），后果：
			//     · 详情 WildlandView / 建新城 CreateCity 用 `First()` → 取到**旧行**（原主人）：
			//       「地图显示归我、详情却写归属别人、mine=false」，我还能对自己的地反复下单刷资源；
			//     · 原主人的「附属野地」列表仍含这块地 → 白占他的上限、还能在上面采集；
			//     · `len(wildlandList(city.ID))` 虚高 → 我这边明明有空位却被判「已达上限」。
			//   现在先把该坐标上的历史记录（含历史重复行）清掉，再建自己的那条。
			//   ⚠️ 清记录前必须把旧行上未结束的采集/驻守部队转返航（helper 内已处理）。
			h.ezfyClearWildlandAt(order.TargetX, order.TargetY, now)
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
			// ★★ 2026-10-19 人机「只摧毁、不占领」：征服成功后**立刻放弃**这块野地
			//   （用户规则：「而且是征服，胜利后就放弃拿到的附属野地」）。
			//   效果：原主人的野地归属被打掉、该坐标变回中立；人机不占上限、不留记录。
			//   与玩家点[放弃]同口径 —— ezfyClearWildlandAt 会先把该坐标上的部队转返航再删行。
			if city.IsBot == 1 {
				h.ezfyClearWildlandAt(order.TargetX, order.TargetY, now)
				h.DB.Model(&model.EzfyMapArea{}).Where("x = ? AND y = ?", order.TargetX, order.TargetY).
					Update("owner_id", 0)
				// ★ 2026-10-19 战报不再追加「(人机只摧毁野地归属, 不占领)」——
				//   这是机制说明（本应只写在注释里），玩家战报只需呈现战果。
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
			// ★ 2026-10-09 掉落逻辑已抽到 `wildlandConfigLoot`（在 `if win` 开头统一调用）——
			//   原来它只写在这里（征服成功占领），掠夺(2) 永远不掉 → 玩家反馈「配置里写的宝物都不掉」。
		}

		// 征服玩家城市
		// ★ 2026-10-06 是否**新建成**占领记录（城池没了 → 全城军官都掉忠诚）；
		//   声明在征服块外，结束区（掠夺/征服共用的军官忠诚结算）也要读它
		newOccupy := false
		// ★ 2026-10-24 本场实际扣掉的民心（= 占前民心），守方「被征服报告」头部展示用
		conqFeelDrop := 0
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
			//   （ezfy_cfg_limit.conquer_feelings_max，默认 5）。
			//   原来封顶写死 20，且按「幸存兵力/2000」动态算，大兵团一次就能清零民心。
			//   现在既保留动态计算（小部队扣得少），又用配置值封顶。
			//
			// ★★ 2026-10-29 用户规则「人机也是正常玩家，只是电脑控制」→ **取消人机单独上限**：
			//   2026-10-27 曾给人机单开一份 bot_conquer_feelings_max（默认 100 = 一次打光），
			//   结果玩家被 AI 征服一次就「民心 -90」，与真人单次最多 -5 完全不对等，被投诉。
			//   现在人机与真人**同一条规则、同一个配置**（conquer_feelings_max），
			//   人机要占城也只能像真人一样反复征服把民心磨到 0。
			if cm := ezfyConquerFeelingsCfg(); feelingDrop > cm {
				feelingDrop = cm
			}
			cur := target.Feelings - feelingDrop
			if cur < 0 {
				cur = 0
			}
			// ★ 2026-10-27 修正「攻方战报数字 ≠ 守方战报数字」：
			//   扣减量被民心实际值截断（民心只有 95，算出来扣 100）时，按**实际扣掉的**显示，
			//   否则攻方看到「民心 - 100」、守方看到「民心 - 95」，两边对不上。
			if cur == 0 && feelingDrop > target.Feelings {
				feelingDrop = target.Feelings
			}
			report += fmt.Sprintf("\n民心 - %d\n当前民心 %d", feelingDrop, cur)
			if cur > 0 {
				target.Feelings = cur
				h.saveCityRes(target)
				h.DB.Model(&model.EzfyCity{}).Where("id = ?", target.ID).
					Updates(map[string]interface{}{"feelings": target.Feelings, "feel_settle_at": time.Now().UnixMilli()})
				travel := ezfyOneWayTravel(order)
				order.Status = 2
				order.ReturnTime = now + travel
				report += "\n民心尚存，征服失败"
				report += fmt.Sprintf("\n征服战果\n黄金:%d\n粮食:%d\n钢铁:%d\n石油:%d\n稀矿:%d", lootGold, lootFood, lootSteel, lootOil, lootRare)
				// ★ 2026-09-25：战利品入账改为 DB 原子累加 + 资源最大值（同下）
				h.addResToCityDB(city.ID, lootFood, lootSteel, lootOil, lootRare, lootGold)
				report += h.battleStatsTail(uid, 0, recyclePct, healLineAtk)
				// ★ 2026-10-06 单次征服攻打（城没占下来，城池还在）→ 只扣城守忠诚
				if frag := h.defectDefenderOfficers(city, target, uid, false); frag != "" {
					report += frag
				}
				h.addReport(uid, 3, "征服报告: "+targetName, report, detail, order.ID)
				// ★★ 2026-10-28 守方战报补齐（用户：「城防守城的时候也有经验，玩家反馈被征服时没有」）：
				//   原来这个分支**直接 return** → 守方既没有「被征服报告」、也拿不到参战声望，
				//   军官经验虽然在库里发了却完全看不到反馈（玩家只能看到自己被打了）。
				//   现在与「攻方获胜/守方守住」两个分支同一套守方视角正文（含军官行 + 军官经验+N）。
				if target != nil && target.UserID != 0 && target.UserID != uid {
					defBody := h.ezfyDefenderReportBody(ezfyDefReportSpec{
						DefUID: target.UserID, DefName: targetName, DefWin: false, Draw: draw, Rounds: br.Rounds,
						Now: now, DefTitle: "被征服报告", OcTitle: "征服", AtkCity: city, Order: order,
						AtkTag: atkTag, DefTag: defTag,
						AtkBefore: atkBefore, AtkAfter: atkAfter, DefBefore: defBefore, DefAfter: defAfter,
						AtkOfficer: leadOfficer, DefOfficerDesc: officerReportDesc(defExpOfficer),
						PrestigeGain: 100, DefExp: defExp, HealLineDef: healLineDef,
						AtkLabel: atkLabel, DefLabel: defLabel,
						// 民心块与「被征服」同款：让守方看到民心扣了多少、还剩多少
						DefFeelShow: true, DefFeelDrop: feelingDrop, DefFeelCur: cur,
					})
					head := fmt.Sprintf("你的城市%s遭到「%s」的军队进攻, 民心尚存, 城市未被征服!",
						targetName, atkLabel)
					pvpDefReportID = h.addReport(target.UserID, 4, "被征服报告: "+targetName+
						"("+strconv.Itoa(order.TargetX)+","+strconv.Itoa(order.TargetY)+")",
						head+"\n\n"+defBody, detail, 0, target.ID)
					h.addPrestige(target.UserID, 100)
					h.ezfyPvpAttachReport(pvpDefDetailID, pvpDefReportID)
				}
				h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
					Updates(map[string]interface{}{"status": order.Status, "result": order.Result, "return_time": order.ReturnTime})
				return
			}
			// 免战保护绝对生效（含宣战）：民心已失也无法征服；早退分支已统一 return，此处为防御保留
			if targetProtected {
				report += "\n民心已失，但目标处于免战保护期，无法征服!"
				target.Feelings = cur
				h.DB.Model(&model.EzfyCity{}).Where("id = ?", target.ID).
					Updates(map[string]interface{}{"feelings": target.Feelings, "feel_settle_at": time.Now().UnixMilli()})
				travel := ezfyOneWayTravel(order)
				order.Status = 2
				order.ReturnTime = now + travel
				report += h.battleStatsTail(uid, 0, recyclePct, healLineAtk)
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
			// ★ 2026-10-24 占前记下实际扣掉的民心（守方「被征服报告」头部「民心 - N」用）
			conqFeelDrop = target.Feelings
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
				// ★ 2026-10-24 换主后城墙按新主的围墙等级重新补满（-1 = 下次读取时初始化）。
				h.DB.Model(&model.EzfyCity{}).Where("id = ?", target.ID).Update("wall_hp", -1)
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
			// ★ 2026-10-24 守方当前城被占走 → 立刻随机切到另一座自己的空闲城，
			//   避免守方还停在被占城里操作（bug：被征服后仍停留被占城）。
			//   无空闲城时不动 —— 下方补给流程（replenishCity）会建新城并切过去。
			if newOccupy || lastCity {
				h.switchAwayConqueredCity(target.UserID, int64(target.ID))
			}
			// ★★ 2026-10-19 用户规则「人机被即将打飞会给玩家发邮件让他别打了」：
			//   守方是人机、且该城被占走（新建占领记录）→ 人机给进攻方发邮件求饶。
			//   只剩最后一座自由城（freeCount <= 1，占掉就飞）→ 用「被打飞求放过」口径；
			//   还有余城 → 用「即将打飞，求停手」口径。邮箱防刷：同一对人机/玩家 24h 一封。
			if target.IsBot == 1 && newOccupy {
				h.ezfyBotBegMail(uid, target.UserID, freeCount <= 1)
			}
			// 补给在解锁后执行（不占用守方锁；建城自带 findFreePos 找空位）
			if needReplenish {
				newCity := h.replenishCity(target.UserID)
				// ★ 2026-10-08 「玩家被打飞系统播报」：守方所有自由城被占光→补给→系统频道播报全服可见
				h.ezfySysChat("%s 玩家被 %s 玩家打飞了！",
					h.ezfyProfileName(target.UserID), h.ezfyProfileName(uid))
				// ★ 2026-10-07 用户反馈「被打飞后战报不该告诉攻击者新坐标」：
				//   攻方战报不再暴露补给新城市的坐标，避免被追打；守方自己的补偿报告(L3475)保留坐标。
				report += fmt.Sprintf("\n守方城市已全部被占, 系统已补给新城市[%s]", newCity.Name)
				h.addReport(target.UserID, 5, "系统补偿新城市",
					fmt.Sprintf("你的全部城市已被敌方占领!\n系统已补偿一座新城市[%s](%d,%d), 请重新发展。", newCity.Name, newCity.X, newCity.Y), "", 0, newCity.ID)
			}
			h.saveCityRes(target)
			h.DB.Model(&model.EzfyCity{}).Where("id = ?", target.ID).
				Updates(map[string]interface{}{"feelings": 0, "grievance": target.Grievance,
					"feel_settle_at": time.Now().UnixMilli()})
			// ★ 2026-10-06 「被征服报告」：顶部保留守方结论，完整战斗正文（与攻方
			//   「征服报告」同款：主题/时间/公文报告/战斗经过/双方兵力/战果/逐回合详情）
			//   在结算完统一拼写（见函数末尾），标题带守方城名+坐标
			//   （原误用攻方 city.Name 是 bug）。
			if !lastCity {
				defConqBody = fmt.Sprintf("你的城市%s已被敌方部队占领!\n被掠夺资源: 粮%d 钢%d 油%d 稀矿%d 金%d",
					targetName, lootFood, lootSteel, lootOil, lootRare, lootGold)
			} else {
				// 该城已被先到的队伍占走（本次重复攻打）
				defConqBody = fmt.Sprintf("敌方部队再次攻打你的城市%s!\n但该城已被其他部队占领, 无法重复占领!\n被掠夺资源: 粮%d 钢%d 油%d 稀矿%d 金%d",
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
				Updates(map[string]interface{}{"feelings": target.Feelings, "grievance": target.Grievance,
					"feel_settle_at": time.Now().UnixMilli()})
			// ★ 2026-10-24 攻方「掠夺报告」补民心变化（与征服报告同款格式），
			//   玩家被掠夺后一眼能看出民心扣了多少、还剩多少。
			report += fmt.Sprintf("\n民心 - %d\n当前民心 %d", lootFeel, target.Feelings)
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
				// ★★ 2026-10-30 修复「掠夺资源刷声望」：原来 = 200 + 掠夺资源总量/10000，
				//   而单次掠夺的粮食+钢铁+石油+稀矿在后期是**几十亿**量级 →
				//   线上实测一场掠夺 +312679 声望（战报「军功声望+312679」），
				//   有人专门反复掠夺资源多的城刷军衔。声望是军衔晋升的唯一来源，
				//   这种量级直接让军衔体系失效。
				//   现在资源只按「开方压缩」给小额加成且**封顶 100** → 单场 200~300，
				//   与征服玩家城 500 / 寇城 80+20×等级 / 野地 30+10×等级 同一量级。
				prestigeGain = ezfyLootCityPrestige(lootFood + lootSteel + lootOil + lootRare)
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
			} else if ezfyIsSeaWildRules(ezfyTerrainEx(order.TargetX, order.TargetY)) {
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
		// ★ 2026-10-28 写**实际到账**经验（满级到账 0 → 干脆不写这一行，别写冗余文案）
		if leadOfficer != nil {
			if eff := h.addOfficerExp(city, leadOfficer.ID, atkExp); eff > 0 {
				report += fmt.Sprintf("\n军官经验+%d", eff)
			}
		}
		travel := ezfyOneWayTravel(order)
		order.Status = 2
		order.ReturnTime = now + travel
		if wareNote != "" {
			report += "\n" + wareNote
		}
		report += fmt.Sprintf("\n战果\n黄金:%d\n粮食:%d\n钢铁:%d\n石油:%d\n稀矿:%d", lootGold, lootFood, lootSteel, lootOil, lootRare)
		// ★ 2026-10-09 ：战报里**不再单独列**「攻方/守方伤兵入营」两行 ——
		//   尾部已经有「回收比例: 攻方X% 守方Y%」把两方占比汇总了，那两行的比例信息是重复的。
		//   （伤兵**数量**去伤兵营页面看；战报只留比例。）
		report += h.battleStatsTail(uid, prestigeGain, recyclePct, healLineAtk)
		// ★ 2026-10-06 守方被动战报（被掠夺/被征服）：**守方视角**标题 = 守方城名+守方坐标
		//   （targetName = target.Name 即被掠夺/被征服的那座守城），
		//   如 [掠夺] 被掠夺报告: 我的城名(X,Y)；与攻方报告(L3490)保持对称口径。
		if order.TargetType == 3 && target != nil {
			// ★★ 2026-10-19 用户反馈「防守方战报怎么是攻方视角」：
			//   原来这里 `defBody := report` **直接复用攻方正文** —— 守方看到的是
			//   「我方一支部队对 XXX 进行了征服」+ 攻方的军衔声望/军官经验/个人荣誉，
			//   完全是攻方视角。现在与「守方成功守住」分支共用 ezfyDefenderReportBody()，
			//   生成**守方视角**正文（敌方进攻、我方战败；军衔声望/战果/尾部都是守方自己的）。
			//   ★ 「回收比例」也不再需要字符串替换 —— 正文按守方口径直接生成（healLineDef）。
			defTitle, repType, ocTitle := "被掠夺报告", 2, "掠夺"
			if order.OrderType == 3 {
				defTitle, repType, ocTitle = "被征服报告", 4, "征服"
			}
			if order.OrderType == 2 || (order.OrderType == 3 && defConqBody != "") {
				defBody := h.ezfyDefenderReportBody(ezfyDefReportSpec{
					DefUID: target.UserID, DefName: targetName, DefWin: false, Draw: draw, Rounds: br.Rounds,
					Now: now, DefTitle: defTitle, OcTitle: ocTitle, AtkCity: city, Order: order,
					AtkTag: atkTag, DefTag: defTag,
					AtkBefore: atkBefore, AtkAfter: atkAfter, DefBefore: defBefore, DefAfter: defAfter,
					AtkOfficer: leadOfficer, DefOfficerDesc: officerReportDesc(defExpOfficer),
					PrestigeGain: 100, DefExp: defExp, HealLineDef: healLineDef,
					AtkLabel: atkLabel, DefLabel: defLabel,
					// ★ 2026-10-24 被征服报告：民心块放正文 [守方] 行下（用户指定格式，头部不再放）
					DefFeelShow: order.OrderType == 3, DefFeelDrop: conqFeelDrop, DefFeelCur: 0,
				})
				head := defConqBody
				if order.OrderType == 2 {
					// 掠夺：守方看不到战利品明细（原攻方正文里的「战果」）→ 抬头里补上「被掠夺资源」，
					// 不能因为改了视角就把「我被抢了多少」这个关键信息弄丢。
					// ★ 2026-10-24 被掠夺报告也展示「民心 - N / 当前民心 X」（与攻方掠夺报告/征服同款），
					//   让守方一眼看到民心扣了多少、还剩多少。
					head = fmt.Sprintf("你的城市%s被敌方部队掠夺!\n民心-%d 民怨+%d\n民心 - %d\n当前民心 %d\n被掠夺资源: 粮%d 钢%d 油%d 稀矿%d 金%d",
						targetName, lootFeel, lootFeel, lootFeel, target.Feelings, lootFood, lootSteel, lootOil, lootRare, lootGold)
				}
				if order.OrderType == 2 {
					pvpDefReportID = h.addReport(target.UserID, 2, "被掠夺报告: "+targetName+"("+strconv.Itoa(order.TargetX)+","+strconv.Itoa(order.TargetY)+")",
						head+"\n\n"+defBody, detail, 0, target.ID)
				} else {
					pvpDefReportID = h.addReport(target.UserID, repType, "被征服报告: "+targetName+"("+strconv.Itoa(order.TargetX)+","+strconv.Itoa(order.TargetY)+")",
						head+"\n\n"+defBody, detail, 0, target.ID)
				}
				// ★★ 2026-10-19 守方「军功声望」：**参战即得 100**（守住/被攻破都给，
				//   与守方军官经验同为「防守贡献」口径）；原来攻方获胜这一场守方一点声望都没有。
				h.addPrestige(target.UserID, 100)
			}
		}
		// ★ 2026-10-08 补出发城 id（与守方战报同口径；原来 city_id=0，按当前城过滤时靠坐标反查兜底）
		pvpAtkReportID = h.addReport(uid, 2, reportType+": "+targetName+
			"("+strconv.Itoa(order.TargetX)+","+strconv.Itoa(order.TargetY)+")", report, detail, order.ID, city.ID)
	} else {
		// ★ 第九轮：打败仗 → 幸存部队撤退返航（原来 status=4 是终止态，
		//   幸存兵力凭空消失、带队军官永远卡在「出征中」，属于 bug）。
		travel := ezfyOneWayTravel(order)
		order.Status = 2
		order.ReturnTime = now + travel
		// ★ 2026-10-09 ：战报里不再单独列「攻方/守方伤兵入营」两行
		//   （尾部「回收比例」已汇总两方占比）。伤兵数量去伤兵营看。
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
			if eff := h.addOfficerExp(city, leadOfficer.ID, atkExp); eff > 0 {
				report += fmt.Sprintf("\n军官经验+%d", eff)
			}
		}
		report += "\n残部正在撤退返航。"
		report += h.battleStatsTail(uid, prestigeGain, recyclePct, healLineAtk)
		// ★ 2026-10-08 补出发城 id（同上）
		pvpAtkReportID = h.addReport(uid, 2, reportType+": "+targetName+
			"("+strconv.Itoa(order.TargetX)+","+strconv.Itoa(order.TargetY)+")", report, detail, order.ID, city.ID)
		if order.TargetType == 3 && target != nil {
			// ★ 2026-10-06 守方成功守住也按进攻意图归「战斗报告」：
			//   掠夺→被掠夺报告 / 征服→被征服报告，军情警讯不再出现防守报告（只留预警）。
			// ★★ 2026-10-19 正文与「攻方获胜」分支统一走 ezfyDefenderReportBody()
			//   （守方视角：敌方进攻、我方守住；军衔声望/战果/尾部都是守方自己的）。
			//   平局不算守住（Win=false → 公文报告显示「与敌方打成平局！」）。
			defTitle, repType, ocTitle := "被掠夺报告", 2, "掠夺"
			if order.OrderType == 3 {
				defTitle, repType, ocTitle = "被征服报告", 4, "征服"
			}
			defReport := h.ezfyDefenderReportBody(ezfyDefReportSpec{
				DefUID: target.UserID, DefName: targetName, DefWin: !draw, Draw: draw, Rounds: br.Rounds,
				Now: now, DefTitle: defTitle, OcTitle: ocTitle, AtkCity: city, Order: order,
				AtkTag: atkTag, DefTag: defTag,
				AtkBefore: atkBefore, AtkAfter: atkAfter, DefBefore: defBefore, DefAfter: defAfter,
				AtkOfficer: leadOfficer, DefOfficerDesc: officerReportDesc(defExpOfficer),
				PrestigeGain: 100, DefExp: defExp, HealLineDef: healLineDef,
				AtkLabel: atkLabel, DefLabel: defLabel,
			})
			head := fmt.Sprintf("你的城市%s遭到「%s」的军队进攻, 我方守军击退了敌军!", targetName, atkLabel)
			pvpDefReportID = h.addReport(target.UserID, repType, defTitle+": "+targetName+
				"("+strconv.Itoa(order.TargetX)+","+strconv.Itoa(order.TargetY)+")",
				head+"\n\n"+defReport, detail, 0, target.ID)
			h.addPrestige(target.UserID, 100)
		}
	}

	// ★ 2026-10-19 附属野地守方战报：胜利 → 被掠夺/被征服；失败 → 防守成功。
	//   野地主人（可能是多城玩家）在这块地的归属城军情里就能看到这条。
	if order.TargetType == 1 || order.TargetType == 2 {
		h.addWildlandOwnerReport(order, wlOwnerUID, wlOwnerCityID, city, targetName, win, detail, now,
			br, atkTag, defTag, atkBefore, atkAfter, defBefore, defAfter, leadOfficer, wildDefOfficerDesc, wildDefExp)
	}
	// ★ 2026-10-26 把「本场战报」回填到风云榜积分明细（管理端「积分明细 → 查看本场战报」）：
	//   明细在战斗判定后先写、战报在结算末尾才写，所以这里补 UPDATE（攻守各 1 条，最多 2 条）。
	h.ezfyPvpAttachReport(pvpAtkDetailID, pvpAtkReportID)
	h.ezfyPvpAttachReport(pvpDefDetailID, pvpDefReportID)
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

// defExcludeSet 城市「避战」的兵种集合（司令部「防守」= 避战，def_move = -1）。
//
// ★ （2026-09-23）：被攻击时，防御战斗的兵种列表**不包含**标记了「避战」的兵种。
// ★ 2026-10-28：文案两易其名（不参与防御 → 不参与 → 避战），值始终是 -1，功能不变。
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

// defCapMap 城市「兵种防守数量」配置（司令部「兵种战斗配置」→ 每个兵种一行「防守数量」）。
//
// ★★ 2026-10-28 用户规则：
//
//	· 每个兵种可单独设置参与防御的兵力**上限**，默认「全部」；
//	· 实际参战 = min(城内该兵种数量, 配置值)：城内多于配置就按配置打，少于配置就按实际打；
//	· 上限 = 系统配置「单城兵力上限」（保存时已夹取，这里不再重复判）；
//	· **城市维度**（ezfy_city_target.city_id），每座城各一套，互不影响。
//
// 只返回「配置值 > 0」的兵种；map 里没有 = 全部（保持改造前的行为）。
func (h *EzfyHandler) defCapMap(cityId uint) map[int]int64 {
	m := map[int]int64{}
	if cityId <= 0 {
		return m
	}
	var list []model.EzfyCityTarget
	h.DB.Where("city_id = ? AND def_count > 0", cityId).Find(&list)
	for _, t := range list {
		if t.DefCount > 0 {
			m[t.TroopId] = t.DefCount
		}
	}
	return m
}

// defCountOf 该兵种实际参战兵力 = min(城内数量, 防守数量配置)。
//
// caps 里没有该兵种（或配置 <= 0）→ 视为「全部」，直接返回城内数量。
func defCountOf(count int64, caps map[int]int64, troopId int) int64 {
	if count <= 0 {
		return 0
	}
	if c := caps[troopId]; c > 0 && count > c {
		return c
	}
	return count
}

// cityDefendLocked 城市正在被敌军攻击（防御指挥室进行中）时，返回「参战兵种」集合。
//
// ★ 2026-10-09 用户规则：「自己被处于指挥时，指挥里面的兵种就不能再出征」。
//
//	理由：战斗结束的守方战损是**按城池当前兵力扣减**的（见 processArrive 的守方战损块），
//	如果允许把参战守军出征带走，战损就扣不到兵 —— 守军被「出征」这个通道悄悄转移走，
//	等于这场防御白打（攻方损失照算、守方零战损）。所以出征入口要把参战兵种锁死。
//
// ★ 例外（用户明确）：「不参与防御」(def_move=-1) 的兵种**不参与当次防御**
//
//	（defExcludeSet 已把它们从防御战斗里排除），带走它们不影响战斗结算 → 照常放行。
//
// 返回 nil = 城市当前没有进行中的防御指挥（不锁任何兵种）。
func (h *EzfyHandler) cityDefendLocked(cityId uint) map[int]bool {
	if cityId <= 0 {
		return nil
	}
	// 是否正在被攻击：存在「我方为守方、进行中」的战场，且其订单仍处于「战斗中(5)」。
	// 口径与「军队动态」里守方 [指挥] 入口完全一致（ezfy_api.go 的 defBattles 查询）。
	var cnt int64
	h.DB.Model(&model.EzfyBattle{}).
		Where("target_id = ? AND status = 1 AND order_id IN (SELECT id FROM ezfy_order WHERE status = ?)",
			cityId, ezfyOrderStatusBattle).Count(&cnt)
	if cnt == 0 {
		return nil
	}
	ex := h.defExcludeSet(cityId)
	locked := map[int]bool{}
	for tid, c := range h.troopMap(cityId) {
		if c > 0 && !ex[tid] {
			locked[tid] = true
		}
	}
	return locked
}

// ezfyIntSetKeys map[int]bool 的「真值键」升序切片（下发给前端用；nil → 空切片）。
func ezfyIntSetKeys(m map[int]bool) []int {
	out := []int{}
	for k, v := range m {
		if v {
			out = append(out, k)
		}
	}
	sort.Ints(out)
	return out
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

// parseWildTreasureDrops 解析野地类型「宝物掉落」配置：[{"name","count","pct"},...]
//
// pct 缺省/<=0 → 100（必掉），>100 钳到 100；count<=0 或 name 为空的条目丢弃。
// ★ 老文本值（如「中级/高级」）不是合法 JSON → 返回 nil（不掉落，兼容历史数据）。
func parseWildTreasureDrops(raw string) []wildTreasureDrop {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var rows []wildTreasureDrop
	if err := json.Unmarshal([]byte(raw), &rows); err != nil {
		return nil
	}
	out := []wildTreasureDrop{}
	for _, r := range rows {
		if r.Count <= 0 || strings.TrimSpace(r.Name) == "" {
			continue
		}
		if r.Pct <= 0 {
			r.Pct = 100
		}
		if r.Pct > 100 {
			r.Pct = 100
		}
		out = append(out, r)
	}
	return out
}

// ezfyWildDropReadyMs 打野地掉落冷却是否已过：距该玩家上次掉落 >= cooldownMs 毫秒（或从未掉过）→ true。
//
// ★ 冷却按**玩家**维度算（不分地图/野地等级）。上次掉落时间戳(ms)存在 ezfy_profile 的 col 列
// （宝物=wild_treasure_at / 商城道具=wild_item_at），0 = 从未掉过。
// cooldownMs <= 0 由调用方直接放行（= 关闭冷却），这里只处理 > 0 的判定。
func (h *EzfyHandler) ezfyWildDropReadyMs(uid uint, col string, cooldownMs int64) bool {
	var last []int64
	h.DB.Model(&model.EzfyProfile{}).Where("user_id = ?", uid).Pluck(col, &last)
	if len(last) == 0 || last[0] <= 0 {
		return true // 从未掉过 / 列尚未补出（老库）→ 放行，避免误伤
	}
	return time.Now().UnixMilli()-last[0] >= cooldownMs
}

// ezfyWildTreasureReady 打野地掉**宝物**冷却是否已过（读 wild_treasure_at）。
//
// ★ 2026-10-11 用户规则「打野地掉落宝物有点频繁了，即使我配置了 1%，如果玩家 2 小时内掉落了宝物那么就不会再掉」：
// 按**玩家**维度冷却（不分地图/野地等级）。上次掉宝时间戳(ms)存 ezfy_profile.wild_treasure_at，0 = 从未掉过。
// cooldownH <= 0 由调用方直接放行（= 关闭冷却），这里只处理 > 0 的判定。
func (h *EzfyHandler) ezfyWildTreasureReady(uid uint, cooldownH int) bool {
	return h.ezfyWildDropReadyMs(uid, "wild_treasure_at", int64(cooldownH)*3600000)
}

// ezfyWildItemReady 打野地掉**商城道具**冷却是否已过（读 wild_item_at）。
//
// ★ 2026-10-13 用户规则「野地商城道具掉落也加个限制，有人用脚本刷野地，掉落过了就不能再次掉落」：
// 与宝物冷却同款按**玩家**维度算，但时间戳字段独立（wild_item_at），两条掉落互不影响。
// ★ 2026-10-26 冷却单位由**小时**改成**秒**（默认 1800 秒），见 ezfyWildItemCooldownSec。
func (h *EzfyHandler) ezfyWildItemReady(uid uint, cooldownSec int) bool {
	return h.ezfyWildDropReadyMs(uid, "wild_item_at", int64(cooldownSec)*1000)
}

// wildlandConfigLoot 野地/寇城战斗**胜利**掉落：完全按「地图管理 → 野地类型」的手动配置走。
//
// ★★ 2026-10-09 ：「掉落 都走手动配置的」。两条配置各自逐条独立按概率判定：
//
//	· 宝物掉落 `treasure`    ：[{"name","count","pct"}]，按 name 精确匹配 ezfy_cfg_equipment
//	· 商城道具掉落 `drop_items`：[[cfg_id,数量,概率%]]
//
// ★ 修复的 bug：这段逻辑原来**只写在「征服成功占领」分支里**（order_type=3），
// 而玩家打野地/寇城绝大多数是**掠夺(2)** → 配置里写的宝物一个都不掉
// （用户反馈「玩家反馈 都不掉这些」）。现在抽成 helper，掠夺/征服都走这里。
//
// ★ 安全：**跳过套装件**（set_id > 0）—— 用户规则「套装军官装备只能通过宝箱开启」，
// 否则在野地类型里填个套装名就能绕过规则刷套装。
//
// ★★ 2026-10-09 「打野地 掉落宝物 / 商城道具掉落 都要播报 系统消息」：
// 掉落的**宝物 + 商城道具**除了写进战报，还往**系统频道**播一条全服可见的消息
// （与「采集到宝物」同一套口径，见 ezfy_chat_exchange.go 的 ezfySysChat）。
// 同一场战斗掉多件/两类时合并成**一条**，避免一次战斗刷屏。
// targetName 是战报标题用的目标名（如「海底森林2级」/「寇城3级」），由调用方传入，
// 保证「播报里看到的」=「战报里写的」。
//
// 返回战报文本（每行前置 \n）；什么都没掉 → 返回空串。
func (h *EzfyHandler) wildlandConfigLoot(uid uint, city *model.EzfyCity, order *model.EzfyOrder, wildLevel int, targetName string) string {
	// 野地类型口径与地图/守军一致：寇城=3、海野(含岛屿)=2、其余陆野=1
	wcType := 1
	if order.TargetType == 2 {
		wcType = 3
	} else if ezfyIsSeaWildRules(ezfyTerrainEx(order.TargetX, order.TargetY)) {
		wcType = 2
	}
	wcfg := ezfyCfg.wildland(wcType, wildLevel)
	if wcfg == nil {
		return ""
	}
	out := ""
	// ★★ 2026-10-09 「打野地 掉落宝物 / 商城道具掉落 都要播报 系统消息」：
	//   本场掉到的「商城道具 + 宝物」名字统一攒进 lootNames，循环结束后**合并成一条**
	//   系统频道播报（一场战斗只刷一条，避免刷屏；口径与「采集到宝物」一致）。
	lootNames := []string{}
	// ★ 2026-10-24 掉落宝物多条合并成一条战报（用户要求，与掉落道具同口径）：如「翡翠项链 ×1、蓝宝石戒指 ×1」
	treasureNames := []string{}
	// ★ 2026-10-11 掉落卡控（用户规则）：道具受「背包单件上限」(默认 1000)、
	//   宝物受「宝物库存上限」(默认 5000) 约束；装不下的**不入包**，战报里给一句简约提示。
	bagMax := ezfyBagItemMax()
	treasureMax := ezfyTreasureMax()
	// ① 商城道具掉落：[[cfg_id, 数量, 概率%],...]
	//
	// ★ 2026-10-13 用户规则「野地商城道具掉落也加个限制吧，有人用脚本刷野地暂时卡控，
	//   掉落过了就不能再次掉落，二战系统配置可配置间隔时间」→ 按**玩家**维度冷却：
	//   距上次掉商城道具不足 ezfy_cfg_limit.wild_item_cooldown_sec 秒（★ 2026-10-26 单位改秒，默认 1800）
	//   则本场**不掉道具**。冷却时长 0 = 关闭冷却（合法值）；**只卡商城道具，宝物掉落不受影响**（两条冷却各自独立）。
	//   掉到道具后才刷新冷却时间戳（没掉到不占冷却）。
	itemCooldownSec := ezfyWildItemCooldownSec()
	itemDropped := false
	if itemCooldownSec <= 0 || h.ezfyWildItemReady(uid, itemCooldownSec) {
		for _, d := range parseWildlandItemDrops(wcfg.DropItems) {
			if rand.Intn(100) >= d[2] {
				continue // 未命中概率，不掉
			}
			it := ezfyCfg.item(d[0])
			if it == nil {
				continue
			}
			if have := h.itemCount(uid, d[0]); have+d[1] > bagMax {
				out += fmt.Sprintf("\n背包已满, %s×%d 未能入包", it.Name, d[1])
				continue
			}
			h.addItem(uid, d[0], d[1])
			lootNames = append(lootNames, fmt.Sprintf("%s×%d", it.Name, d[1]))
			itemDropped = true
		}
	}
	// ★ 2026-10-24 掉落道具多条合并成一条战报（用户要求）：如「道具A×1、道具B×2」
	if len(lootNames) > 0 {
		out += fmt.Sprintf("\n掉落道具: %s", strings.Join(lootNames, "、"))
	}
	if itemDropped {
		// 本次掉到商城道具 → 记录掉道具时间，冷却从此刻起算（下次要等冷却过去才会再掉）
		h.DB.Model(&model.EzfyProfile{}).Where("user_id = ?", uid).
			Update("wild_item_at", time.Now().UnixMilli())
	}
	// ② 宝物掉落：[{"name","count","pct"},...]
	//
	// ★ 2026-10-11 用户规则「打野地掉落宝物有点频繁了，即使我配置了 1%，如果玩家 2 小时内
	//   掉落了宝物那么就不会再掉」→ 按**玩家**维度冷却：距上次掉宝不足
	//   ezfy_cfg_limit.wild_treasure_cooldown_h 小时（默认 2）则本场**不掉宝物**。
	//   冷却时长 0 = 关闭冷却（合法值）；**只卡宝物，商城道具掉落不受影响**。
	//   掉到宝物后才刷新冷却时间戳（没掉到不占冷却）。
	treasureLeft := h.ezfyTreasureLeft(uid)
	cooldownH := ezfyWildTreasureCooldownH()
	treasureDropped := false
	if cooldownH <= 0 || h.ezfyWildTreasureReady(uid, cooldownH) {
		for _, d := range parseWildTreasureDrops(wcfg.Treasure) {
			if rand.Intn(100) >= d.Pct {
				continue
			}
			eq := ezfyCfg.equipmentByName(d.Name)
			if eq == nil || eq.SetId > 0 {
				continue // 名字对不上 / 是套装件（套装只能开宝箱）
			}
			if d.Count > treasureLeft {
				out += fmt.Sprintf("\n宝物已满(%d), %s×%d 未能入包", treasureMax, eq.Name, d.Count)
				continue
			}
			for k := 0; k < d.Count; k++ {
				h.addEquipment(city, eq)
			}
			treasureLeft -= d.Count
			treasureNames = append(treasureNames, fmt.Sprintf("%s×%d", eq.Name, d.Count))
			lootNames = append(lootNames, fmt.Sprintf("%s×%d", eq.Name, d.Count))
			treasureDropped = true
		}
	}
	// ★ 2026-10-24 掉落宝物多条合并成一条战报（用户要求）：如「翡翠项链 ×1、蓝宝石戒指 ×1」
	if len(treasureNames) > 0 {
		out += fmt.Sprintf("\n掉落宝物: %s", strings.Join(treasureNames, "、"))
	}
	if treasureDropped {
		// 本次掉到宝物 → 记录掉宝时间，冷却从此刻起算（下次要等冷却过去才会再掉）
		h.DB.Model(&model.EzfyProfile{}).Where("user_id = ?", uid).
			Update("wild_treasure_at", time.Now().UnixMilli())
	}
	if len(lootNames) > 0 {
		// 播报目标名用调用方算好的 targetName（「海底森林2级」/「寇城3级」）；
		// 万一为空（理论上不会）回落「野地N级」，保证播报永远可读。
		label := strings.TrimSpace(targetName)
		if label == "" {
			label = "野地" + strconv.Itoa(wildLevel) + "级"
		}
		h.ezfySysChat("恭喜玩家 %s 在 %s(%d,%d) 缴获: %s",
			h.ezfyProfileName(uid), label, order.TargetX, order.TargetY,
			strings.Join(lootNames, "、"))
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

// ezfyScoutInterceptDefenderText 守方「防空拦截」战报的标题与结尾。
// 空战胜负按 ar.AttackerWin 判定：攻方胜 = 守方侦察机被打光、敌军突破拦截，
// 不能沿用「击退敌军侦察」；只有守方守住（含平局）才写击退。
func ezfyScoutInterceptDefenderText(attackerWin bool) (title, tail string) {
	if attackerWin {
		return "战斗报告: 防空拦截失败", "我方侦察机全部被击破，敌军侦察机突破拦截。\n"
	}
	return "战斗报告: 击退敌军侦察", "敌军侦察机已被击破或击退。\n"
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

// ezfyWildDefOfficersDesc 汇总「玩家附属野地」守军各部队（增援6/采集7）的带队军官，
// 生成战报正文与指挥室「准备回合」共用的展示文案（如「甲(5级), 乙(3级)」）。
//
// ★★ 2026-10-19 用户反馈「被攻击方附属野地增援带了军官，攻击方打的时候没显示」：
//
//	野地守军是真人部队（驻军/采集队各带一名军官），原来只有野地/寇城「配置守将」
//	那一分支会写 defOfficerDesc，玩家守军这分支完全没展示带队军官。
//	这里从守军订单聚合军官名（订单只存名字），再批量回查等级补上「(N级)」。
//	同野地守军都来自归属城（createOrder 限定 wl.CityId == 出发城），CityId 一致。
func (h *EzfyHandler) ezfyWildDefOfficersDesc(orders []model.EzfyOrder) string {
	if len(orders) == 0 {
		return ""
	}
	names := []string{}
	seen := map[string]bool{}
	cityIDs := map[uint]bool{}
	for i := range orders {
		n := strings.TrimSpace(orders[i].Officer)
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		names = append(names, n)
		if orders[i].CityId > 0 {
			cityIDs[uint(orders[i].CityId)] = true
		}
	}
	if len(names) == 0 {
		return ""
	}
	levels := map[string]int{}
	if len(cityIDs) > 0 {
		ids := make([]uint, 0, len(cityIDs))
		for cid := range cityIDs {
			ids = append(ids, cid)
		}
		var os []model.EzfyOfficer
		h.DB.Where("city_id IN ? AND name IN ?", ids, names).Find(&os)
		for i := range os {
			levels[os[i].Name] = os[i].Level
		}
	}
	parts := make([]string, 0, len(names))
	for _, n := range names {
		if lv := levels[n]; lv > 0 {
			parts = append(parts, n+"("+strconv.Itoa(lv)+"级)")
		} else {
			parts = append(parts, n)
		}
	}
	return strings.Join(parts, ", ")
}

// ezfyDefenderBonus 附属野地守军的战斗加成快照。
//
// ★★ 2026-10-20 用户反馈「我是守方，我的斯大林拿航母采集，战场里守方军官加成 0、
// 科技也没有」：附属野地有玩家守军时，之前只把军官名字写进战报，战斗引擎拿到的
// defBonus/defAtkBonus/defTechBreak 全是 0。现在与玩家城 case 3 的城守口径一致：
//
//	防御 = 均质装甲(7)×3 + 重型机床(9)×2 + 军官学识/防御技能
//	攻击 = 枪械改良(6)×3 + 重型机床(9)×2 + 军官攻击 + 装备伤害
//	速度/射程/兵种专属/装备/套装/伤兵回收同样按城守公式计算。
type ezfyDefenderBonus struct {
	DefBonus        int
	DefAtkBonus     int
	DefSpeedBonus   int
	DefRangeBonus   int
	DefHealTech     int
	OfficerAtkBonus int
	Equip           ezfyBattleBonus
	SkillBreak      []ezfyBonusItem
	TechBreak       []ezfyBonusItem
	DefBreak        []ezfyBonusItem
	Type            ezfyTypeBonus
	OfficerDesc     string
}

// ezfyWildDefCombatBonus 计算附属野地守军（增援6/驻守采集7）的战斗加成。
//
// tech 必须取**野地归属城**的科技表（玩家科技按账号共享，techMap(归属城) 即可），
// officer 为守军主将（见 ezfyWildDefCommander）；没有军官时传 nil，仍会保留科技加成。
func (h *EzfyHandler) ezfyWildDefCombatBonus(tech map[int]int, officer *model.EzfyOfficer) ezfyDefenderBonus {
	b := ezfyDefenderBonus{
		Type: ezfyTypeBonus{Speed: map[int]int{}, Def: map[int]int{}},
	}
	// 防御：与玩家城守军一致，不含城墙（城墙只守城，不守野地）。
	b.DefBonus = tech[7]*3 + tech[9]*2 + h.officerGuardBonus(officer)
	// 速度：机械增压(10) 全体 + 喷气推进(19) 只加空军 —— ★ 2026-10-31 满级 50%，读配置
	b.DefSpeedBonus = ezfyTechEffectValueAt(10, tech[10])
	// 伤兵回收：战地医疗(21) + 军官「机械改装」5%/级（野地守军损失写回订单，
	// 这里保留与城守同一口径，战报/后续扩展不会因为漏算而出现 0）。
	b.DefHealTech = ezfyTechEffectValueAt(21, tech[21])
	if h.officerHasSkill(officer, "机械改装") {
		b.DefHealTech += 5 * h.officerSkillScale(officer)
	}
	if v := ezfyTechEffectValueAt(19, tech[19]); v > 0 {
		b.Type.Speed[ezfyTroopTypeAir] += v
	}
	if v := tech[16] * 2; v > 0 {
		b.Type.Def[ezfyTroopTypeCity] = v
	}
	b.Type = h.officerTypeBonus(officer, b.Type)
	// 装备/套装六项（伤害→攻击、防御→防御、生命/暴击/移动按引擎口径生效）。
	b.Equip = h.officerBattleEquipBonus(officer)
	// 攻击：枪械改良/重型机床 + 军官攻击 + 装备伤害。
	b.OfficerAtkBonus = h.officerBattleBonus(officer)
	b.DefAtkBonus = tech[6]*3 + tech[9]*2 + b.OfficerAtkBonus + b.Equip.Dmg
	b.SkillBreak = h.officerSkillsBreak(officer)
	b.TechBreak = ezfyBonusItems(
		ezfyTechItem("枪械改良", tech[6]*3),
		ezfyTechItem("均质装甲", tech[7]*3),
		ezfyTechItem("增程弹药", tech[8]*3),
		ezfyTechItem("重型机床", tech[9]*2),
		ezfyTechItem("机械增压", ezfyTechEffectValueAt(10, tech[10])),
		ezfyTechItem("永备工事", tech[16]*2),
		ezfyTechItem("喷气推进", ezfyTechEffectValueAt(19, tech[19])),
	)
	b.DefRangeBonus = tech[8] * 3
	b.DefBreak = ezfyBonusItems(
		ezfyTechItem("科技·均质装甲", tech[7]*3),
		ezfyTechItem("科技·重型机床", tech[9]*2),
	)
	if officer != nil {
		b.DefBreak = append(b.DefBreak, ezfyBonusItem{Name: "军官·" + officer.Name, Value: h.officerGuardAttrBonus(officer)})
		for _, s := range h.officerGuardSkillsBreak(officer) {
			b.DefBreak = append(b.DefBreak, ezfyBonusItem{Name: "军官技能·" + s.Name, Value: s.Value})
		}
	}
	b.DefBreak = append(b.DefBreak, ezfyTechItem("装备套装", b.Equip.Def)...)
	b.OfficerDesc = h.officerBattleDesc(officer, h.officerBaseBonus(officer), "攻击加成",
		h.officerGuardAttrBonus(officer), "守军防御")
	return b
}

// ezfyWildDefCommander 选出附属野地守军的主将：等级最高的带队军官（同等级取 id 小者）。
//
// ★ 战斗引擎一方只有一个「军官加成」入口，多支守军（采集7 + 增援驻军6）无法各算各的；
// 取等级最高者作为主将，避免把多支守军的军官加成无序叠加给全体守军。
func (h *EzfyHandler) ezfyWildDefCommander(orders []model.EzfyOrder, ownerCityID uint) *model.EzfyOfficer {
	if len(orders) == 0 {
		return nil
	}
	cityIDs := map[uint]bool{}
	names := []string{}
	seen := map[string]bool{}
	for i := range orders {
		n := strings.TrimSpace(orders[i].Officer)
		if n == "" {
			continue
		}
		if !seen[n] {
			seen[n] = true
			names = append(names, n)
		}
		if orders[i].CityId > 0 {
			cityIDs[uint(orders[i].CityId)] = true
		}
	}
	if ownerCityID > 0 {
		cityIDs[ownerCityID] = true
	}
	if len(names) == 0 || len(cityIDs) == 0 {
		return nil
	}
	ids := make([]uint, 0, len(cityIDs))
	for cid := range cityIDs {
		ids = append(ids, cid)
	}
	var os []model.EzfyOfficer
	h.DB.Where("city_id IN ? AND name IN ?", ids, names).
		Order("level DESC, id ASC").Limit(1).Find(&os)
	if len(os) == 0 {
		return nil
	}
	return &os[0]
}

// ezfyWildlandOwnerOf 查某坐标「附属野地」的主人（0,0 = 无主/中立/寇城/自己打自己）。
//
// ★ 2026-10-19 附属野地战报：ezfy_wildland.CityId 指向占领者城市，
//
//	由此拿到 user_id 与归属城 —— 战报的 city_id 用**归属城**，
//	玩家切到那座城的「军情 → 战斗报告」就能看到（与城市守方战报同口径）。
//	atkUID = 攻方 uid：打自己的地不生成守方战报。
func (h *EzfyHandler) ezfyWildlandOwnerOf(x, y int, atkUID uint) (uid uint, cityID int64) {
	uid, cityID, _ = h.ezfyWildlandOwnerInfo(x, y, atkUID)
	return uid, cityID
}

// ezfyWildlandOwnerInfo 与 ezfyWildlandOwnerOf 完全同口径，额外把**野地行**一起返回 ——
// 供需要「我方野地名字(坐标)」的守方战报用（如「被侦查报告」）。这样同一份守方文案
// 不用为了拿 Level 再查一次同一行（服务器单核，少一次读就少一次）。
func (h *EzfyHandler) ezfyWildlandOwnerInfo(x, y int, atkUID uint) (uid uint, cityID int64, wl model.EzfyWildland) {
	if err := h.DB.Where("x = ? AND y = ?", x, y).Order("id DESC").First(&wl).Error; err != nil || wl.CityId <= 0 {
		return 0, 0, model.EzfyWildland{}
	}
	var owner model.EzfyCity
	if err := h.DB.First(&owner, wl.CityId).Error; err != nil || owner.UserID == 0 || owner.UserID == atkUID {
		return 0, 0, model.EzfyWildland{}
	}
	return owner.UserID, int64(owner.ID), wl
}

// ezfyReportSideName 战报 [攻方]/[守方] 行展示的名字：
// 真人玩家优先用游戏昵称（ezfy_profile.nickname），取不到时回退到城市名/野地名。
// ★ 2026-10-21 用户规则：战报不要显示城市名，要显示玩家名字。
func (h *EzfyHandler) ezfyReportSideName(uid uint, fallback string) string {
	if uid > 0 {
		if n := strings.TrimSpace(h.ensureProfile(uid).Nickname); n != "" {
			return n
		}
	}
	return fallback
}

// ezfyDefReportSpec 「守方视角」战报正文的输入（★ 2026-10-19）。
//
// ★★ 抽出来的原因：玩家城市守方战报原来在「攻方获胜」分支**直接复用攻方正文**
// （defBody := report）→ 守方看到的是「我方一支部队对 XXX 进行了征服」+ 攻方的
// 军衔声望/军官经验/个人荣誉，完全是攻方视角（用户：「防守方战报怎么是攻方视角」）。
// 「守方成功守住」分支本来就有正确模板，附属野地守方又是另一份手写模板 —— 三处很容易漂移。
// 现在统一由 ezfyDefenderReportBody 生成（抬头句另由调用方拼）：
//
//	主题/出发地(敌方城)/目的地(我方城或野地)/时间/公文报告(「敌方一支部队对…」)/
//	守方军衔声望(本人)/攻守兵力块(军官行挂各自标题下)/军功声望(守方)/军官经验(守方)/战果(守方)/尾部。
type ezfyDefReportSpec struct {
	DefUID   uint            // 守方玩家 uid（取本人军衔声望与尾部统计）
	DefName  string          // 守方目标名（城市名 / 野地名）
	DefWin   bool            // 守方是否获胜（顶住进攻）
	Draw     bool            // 40 回合未分胜负
	Rounds   int             // 回合数
	Now      int64           // 结算时间(ms)
	DefTitle string          // 主题/公文报告类型名："被掠夺报告"/"被征服报告"
	OcTitle  string          // 战斗描述用词："掠夺"/"征服"
	AtkCity  *model.EzfyCity // 攻方城（取名字与出发坐标）
	Order    *model.EzfyOrder

	AtkTag, DefTag                           string // 攻/守方 胜败平标签
	AtkBefore, AtkAfter, DefBefore, DefAfter map[int]int64

	AtkOfficer     *model.EzfyOfficer // 攻方带队军官（挂在 [攻方] 标题下）
	DefOfficerDesc string             // 守方军官展示文案（城守 / 野地驻军带队军官聚合）
	PrestigeGain   int                // 守方本场军功声望（>0 才输出该行）
	DefExp         int64              // 守方军官本场**实际到账**经验（>0 才输出该行）
	HealLineDef    string             // 守方「回收比例」行（空串=不显示）
	// ★ 2026-10-24 守方「民心」块（被征服报告正文 [守方] 行下展示，头部不再放）：扣减量 + 当前值
	DefFeelShow bool // 是否输出民心块
	DefFeelDrop int  // 民心扣减
	DefFeelCur  int  // 当前民心
	// ★ 2026-10-21 [攻方]/[守方] 行展示名（玩家昵称；空=沿用城市名/野地名）
	AtkLabel string
	DefLabel string
}

// ezfyDefenderReportBody 生成「守方视角」战报正文（**不含抬头句**，调用方自己拼），
// 玩家城市（被掠夺/被征服/守住）与附属野地（被掠夺/被征服/守住）共用，避免视角与口径漂移。
func (h *EzfyHandler) ezfyDefenderReportBody(s ezfyDefReportSpec) string {
	atkName := "敌军"
	atkX, atkY := 0, 0
	if s.AtkCity != nil {
		if s.AtkCity.Name != "" {
			atkName = s.AtkCity.Name
		}
		atkX, atkY = s.AtkCity.X, s.AtkCity.Y
	}
	tx, ty := 0, 0
	if s.Order != nil {
		tx, ty = s.Order.TargetX, s.Order.TargetY
	}
	// ★ 2026-10-21 [攻方]/[守方] 行显示玩家昵称；未传时回退城市名/野地名。
	atkLabel := strings.TrimSpace(s.AtkLabel)
	if atkLabel == "" {
		atkLabel = atkName
	}
	defLabel := strings.TrimSpace(s.DefLabel)
	if defLabel == "" {
		defLabel = s.DefName
	}
	defProfile := h.ensureProfile(s.DefUID)
	b := fmt.Sprintf("主题:%s\n出发地:%s(%d,%d)\n目的地:%s(%d,%d)\n时间:%s\n公文报告:%s\n敌方一支部队对 %s[ %d，%d ]进行了%s。战斗共持续 %d 回合，我方战斗%s\n",
		s.DefTitle, atkName, atkX, atkY, s.DefName, tx, ty,
		time.UnixMilli(s.Now).Format("2006-01-02 15:04"), s.DefTitle,
		defLabel, tx, ty, s.OcTitle, s.Rounds, battleOutcomeText(s.DefWin, s.Draw))
	b += fmt.Sprintf("军衔声望:%d\n", defProfile.Prestige)
	b += fmt.Sprintf("[%s]攻方:%s\n", s.AtkTag, atkLabel)
	if s.AtkOfficer != nil {
		b += "军官:" + officerReportDesc(s.AtkOfficer) + "\n"
	}
	b += troopChangeText(s.AtkBefore, s.AtkAfter)
	b += fmt.Sprintf("--------------------\n[%s]守方:%s\n", s.DefTag, defLabel)
	if s.DefFeelShow {
		// ★ 2026-10-26 末尾补 "\n"：否则「当前民心 0」会和下一行「军官:xxx」/兵种行黏在一起
		//   （用户反馈「当前民心 0军官:罗伯特·霍尔(150级)」需要换行）。
		b += fmt.Sprintf("\n民心 - %d\n当前民心 %d\n", s.DefFeelDrop, s.DefFeelCur)
	}
	if s.DefOfficerDesc != "" {
		b += "军官:" + s.DefOfficerDesc + "\n"
	}
	b += troopChangeText(s.DefBefore, s.DefAfter)
	if s.PrestigeGain > 0 {
		b += fmt.Sprintf("\n军功声望+%d", s.PrestigeGain)
	}
	if s.DefExp > 0 {
		b += fmt.Sprintf("\n军官经验+%d", s.DefExp)
	}
	// 守方战果恒为 0（战利品/失地都由抬头句与攻方战报承载）—— 与「回收比例/掠夺比例」口径一致。
	b += "\n战果\n黄金:0\n粮食:0\n钢铁:0\n石油:0\n稀矿:0"
	b += h.battleStatsTail(s.DefUID, s.PrestigeGain, 0, s.HealLineDef)
	return b
}

// addWildlandOwnerReport 给「附属野地」主人写守方战报（用户 2026-10-19：
// 「自己附属野地被征服、被掠夺、被侦查 战斗报告也要能看」）。
//
//	win=true  掠夺(2) → 「被掠夺报告」(类型 2)；征服(3) → 「被征服报告」(类型 4)
//	win=false         → 标题仍按**进攻意图**用「被掠夺/被征服报告」，正文写明守住了
//	                    （与玩家城市守方完全同口径：ezfyReportCategory 只有这两个前缀
//	                     才归「战斗报告(2)」，「防守报告」会被归到「其他(3)」看不到）
//
// detail = 逐回合战斗详情：原样写进守方战报，前端才会出现 [展开/收起逐回合详情]。
// 正文由 ezfyDefenderReportBody() 按**守方视角**生成（不再复用攻方正文），
// 守方军官经验(defExp)与军功声望(参战 +100)也在这里落地。
func (h *EzfyHandler) addWildlandOwnerReport(order *model.EzfyOrder, ownerUID uint, ownerCityID int64,
	atkCity *model.EzfyCity, targetName string, win bool, detail string, now int64,
	br ezfyBattleResult, atkTag, defTag string,
	atkBefore, atkAfter, defBefore, defAfter map[int]int64,
	leadOfficer *model.EzfyOfficer, defOfficerDesc string, defExp int64) {
	if ownerUID == 0 || ownerCityID == 0 {
		return
	}
	atkName := "敌军"
	if atkCity != nil && atkCity.Name != "" {
		atkName = atkCity.Name
	}
	// ★ 2026-10-21 守方战报里的攻方/守方也显示玩家昵称：
	//   攻方=玩家昵称；守方=附属野地城主（ownerUID）昵称。
	atkLabel := h.ezfyReportSideName(order.UserID, atkName)
	defLabel := h.ezfyReportSideName(ownerUID, targetName)
	pos := fmt.Sprintf("(%d,%d)", order.TargetX, order.TargetY)
	title := "被掠夺报告: " + targetName + pos
	defTitle := "被掠夺报告"
	reportType := 2
	ocName := "掠夺"
	head := fmt.Sprintf("你的附属野地 %s%s 遭到「%s」的军队进攻, 我方守军击退了敌军, 未被掠夺!", targetName, pos, atkLabel)
	if order.OrderType == 3 {
		title = "被征服报告: " + targetName + pos
		defTitle = "被征服报告"
		reportType = 4
		ocName = "征服"
		head = fmt.Sprintf("你的附属野地 %s%s 遭到「%s」的军队进攻, 我方守军击退了敌军, 野地仍在!", targetName, pos, atkLabel)
	}
	if win {
		if order.OrderType == 3 {
			head = fmt.Sprintf("你的附属野地 %s%s 被「%s」的军队征服, 该野地已被敌方占领!", targetName, pos, atkLabel)
		} else {
			head = fmt.Sprintf("你的附属野地 %s%s 被「%s」的军队掠夺!", targetName, pos, atkLabel)
		}
	}
	// ★★ 2026-10-19 守方「军功声望」：**参战即得 100**（守住/被攻破都给，与玩家城守方同口径，
	//   也与守方军官经验一样按「防守贡献」算）—— 用户：「军官经验、声望补下吧应该有的」。
	//   原来野地守方一点声望都没有（被攻破那份报告显示的其实是**攻方**的声望，见上）。
	defPrestige := 100
	body := h.ezfyDefenderReportBody(ezfyDefReportSpec{
		DefUID: ownerUID, DefName: targetName, DefWin: !win && !br.Draw, Draw: br.Draw, Rounds: br.Rounds,
		Now: now, DefTitle: defTitle, OcTitle: ocName, AtkCity: atkCity, Order: order,
		AtkTag: atkTag, DefTag: defTag,
		AtkBefore: atkBefore, AtkAfter: atkAfter, DefBefore: defBefore, DefAfter: defAfter,
		AtkOfficer: leadOfficer, DefOfficerDesc: defOfficerDesc,
		PrestigeGain: defPrestige, DefExp: defExp,
		AtkLabel: atkLabel, DefLabel: defLabel,
	})
	h.addReport(ownerUID, reportType, title, head+"\n\n"+body, detail, order.ID, ownerCityID)
	if defPrestige > 0 {
		h.addPrestige(ownerUID, defPrestige)
	}
}

// addWildGarrisonOfficerExp 给「玩家附属野地」守军各部队的带队军官发战斗经验
// （★ 2026-10-19 用户：「军官经验…应该有的」）。
//
// 野地守军是订单（增援6/采集7），每单只存带队军官**名字**；这里按归属城 + 名字
// 回查军官实例逐个加经验（与 ezfyWildDefOfficersDesc 同口径）。exp<=0 / 无名字则跳过。
func (h *EzfyHandler) addWildGarrisonOfficerExp(orders []model.EzfyOrder, ownerCityID int64, exp int64) {
	if exp <= 0 || ownerCityID <= 0 || len(orders) == 0 {
		return
	}
	names := make([]string, 0, len(orders))
	seen := map[string]bool{}
	for i := range orders {
		n := strings.TrimSpace(orders[i].Officer)
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		names = append(names, n)
	}
	if len(names) == 0 {
		return
	}
	var owner model.EzfyCity
	if err := h.DB.First(&owner, ownerCityID).Error; err != nil {
		return
	}
	var os []model.EzfyOfficer
	h.DB.Where("city_id = ? AND name IN ?", owner.ID, names).Find(&os)
	for i := range os {
		h.addOfficerExp(&owner, os[i].ID, exp)
	}
}

// ezfyWriteBackWildGarrisonLeft 野地驻军战损写回（**攻方未胜**时调用）。
//
// ★★ 2026-10-19 用户反馈「附属野地增援的驻军被打了一部分，下次继续打兵力还是原来的」：
//
//	野地守军 = 该野地上多支订单（驻守采集 7）的兵力汇总，战斗结果只给**总损失**，
//	所以按各订单兵力占比分摊剩余：
//	  剩余总量 = 总兵力 - 总损失（不为负）
//	  每支剩余 = 该支兵力 × 剩余总量 / 总兵力（整除，余数补给第一支有兵的订单）
//	  该支剩余 = 0 → status=4（阵亡，与攻方全胜时的处置一致）
//	兵种内部再按原比例缩放，避免「只剩卡车没有步兵」这类偏差。
//
// 攻方全胜时**不调用**：守军由 ezfyDestroyWildlandDefenders 统一歼灭（status=4 + carry 清零）。
func (h *EzfyHandler) ezfyWriteBackWildGarrisonLeft(orders []model.EzfyOrder, losses []ezfyUnitGroup) {
	var dead int64
	for _, g := range losses {
		dead += g.Count
	}
	if dead <= 0 {
		return
	}
	parsed := make([][]ezfyUnitGroup, len(orders))
	sub := make([]int64, len(orders))
	total := int64(0)
	for i := range orders {
		parsed[i] = parseGroups(orders[i].Troops)
		for _, g := range parsed[i] {
			sub[i] += g.Count
		}
		total += sub[i]
	}
	if total <= 0 {
		return
	}
	remain := total - dead
	if remain < 0 {
		remain = 0
	}
	left := make([]int64, len(orders))
	var assigned int64
	for i := range orders {
		left[i] = sub[i] * remain / total
		assigned += left[i]
	}
	// 整除的余数补给第一支有兵的订单，保证总和 = remain
	for i := 0; i < len(orders) && assigned < remain; i++ {
		if sub[i] > 0 {
			left[i] += remain - assigned
			assigned = remain
		}
	}
	for i := range orders {
		if left[i] <= 0 || sub[i] <= 0 {
			h.DB.Model(&model.EzfyOrder{}).Where("id = ?", orders[i].ID).Update("status", 4)
			continue
		}
		out := make([]ezfyUnitGroup, 0, len(parsed[i]))
		var sum int64
		for _, g := range parsed[i] {
			c := g.Count * left[i] / sub[i]
			if c > 0 {
				out = append(out, ezfyUnitGroup{TroopId: g.TroopId, Count: c})
				sum += c
			}
		}
		if len(out) == 0 {
			h.DB.Model(&model.EzfyOrder{}).Where("id = ?", orders[i].ID).Update("status", 4)
			continue
		}
		if sum < left[i] {
			out[0].Count += left[i] - sum
		}
		h.DB.Model(&model.EzfyOrder{}).Where("id = ?", orders[i].ID).Update("troops", groupsJSON(out))
	}
}

// ezfyDestroyWildlandDefenders 歼灭**玩家附属野地**的全部守军（用户规则 2026-10-18）。
//
// 玩家附属野地的守军 = 驻军(增援6, status=3) + 采集/驻守采集(7, status=1) 部队。
// 攻方战胜（掠夺/征服共用）→ 这些部队全部阵亡：
//
//	· 订单置 status=4(阵亡)，不再返航、不留战后兵力；
//	· 采集部队负重(carry) 一并清零 —— 「采集部队被打没, 采集到的资源也一起消失」,
//	  不会运回野地主人城市，也**不缴获给攻方**（与 HarvestAll 的「收获入城」相反, 这里直接损失）。
//
// ★★ 2026-10-19 用户规则：攻击**有归属(玩家附属)**的野地**不产出任何战利品**
//
//	（防「反复打同一块有主野地刷资源」）—— 所以守军采集队身上已采的资源也一并损失，
//	不并入攻方 loot。中立野地 / AI 寇城的战利品不受影响（不经过本函数）。
//
// 在途(4)采集队不在此列：它还没驻守到地，由 ezfyClearWildlandAt 转返航带回。
//
// ★★ 2026-10-19 用户规则（去重）：「附属野地失守」这条不再单独发 ——
// 野地主人已经会收到一条「被掠夺/被征服报告」（见 addWildlandOwnerReport），
// 「有第一个就够了」，再发一条纯属浪费服务器资源。本函数现在只负责歼灭守军订单。
func (h *EzfyHandler) ezfyDestroyWildlandDefenders(x, y int, now int64) {
	var wl model.EzfyWildland
	if err := h.DB.Where("x = ? AND y = ?", x, y).Order("id DESC").First(&wl).Error; err != nil || wl.CityId <= 0 {
		return // 中立野地/寇城: 没有玩家守军
	}
	var orders []model.EzfyOrder
	// ★★ 2026-10-25 改成**按坐标**找守军（与 processArrive 构建守军同口径）：
	//   野地行会因放弃/重建换 id，只按 target_id 会漏掉守军 → 打死了也清不掉；
	//   同时用 ezfyWildDefenderOrderCond 排除「已返航归队」的采集队僵尸单。
	h.DB.Where("target_type <> 3 AND "+ezfyWildDefenderOrderCond+
		" AND target_x = ? AND target_y = ?", x, y).Find(&orders)
	if len(orders) == 0 {
		return // 无驻军/采集部队 → 守军本来就是 0
	}
	ids := make([]int64, 0, len(orders))
	for i := range orders {
		ids = append(ids, int64(orders[i].ID))
	}
	// ★★ 2026-10-19 采集部队负重(carry) 直接清空：资源随守军一起损失，**不缴获给攻方**
	//   （用户：「攻击有归属的野地应该都是 0，不然有人靠这个刷资源」）。
	h.DB.Model(&model.EzfyOrder{}).Where("id IN ?", ids).
		Updates(map[string]interface{}{"status": 4, "carry": "", "arrive_time": 0, "collect_start": 0})
}

// addReport 战报写入
// scoutReportBody 侦查报告正文
//
// ★★ 2026-10-02 用户规则重做：报告细节受**侦查方无线电侦听(科技12)等级**卡控，
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
	target *model.EzfyCity, defender []ezfyUnitGroup, wildOwned bool) string {
	// 侦查方无线电侦听等级（科技12，0-10），决定报告能看清多少细节
	scoutLv := h.techMap(uint(order.CityId))[ezfyReconTechID]
	// ★★ 2026-10-31 黄金洞察（原黄金眼）：**带队军官**按技能等级叠加侦察等级（封顶 10）。
	//   用户口径：「我侦察 1 级，带个有该技能的军官，侦察等级就叠加，超了就叠加到最高级」。
	//   与「市长提高情报等级」同一个技能的两面（见 ezfyIntelLevel / ezfyMayorReconLevel）。
	if lv := h.ezfyReconSkillLevel(h.officerByName(uint(order.CityId), order.Officer)); lv > 0 {
		scoutLv += lv
		if scoutLv > ezfyIntelMaxLevel {
			scoutLv = ezfyIntelMaxLevel
		}
	}
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
		} else if ezfyIsSeaWildRules(ezfyTerrainEx(order.TargetX, order.TargetY)) {
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
		// ★★ 2026-10-18 玩家附属野地（用户规则）：守军是真人驻军/采集部队, 无配置守将 → 不显示
		if !wildOwned {
			if cfg := ezfyCfg.wildland(cfgType, level); cfg != nil && cfg.OfficerId > 0 {
				if g := ezfyCfg.general(cfg.OfficerId); g != nil {
					fmt.Fprintf(&b, "\n守将: %s Lv.%d", g.Name, g.Level)
				}
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
	// ★ 无线电侦听<6级: 只能模糊看到已有的建筑, 建筑等级不清晰
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
		// ★ 无线电侦听<7级: 兵种类型只能模糊查看, 数量无法查清
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

	// ★ 无线电侦听≥9级: 才能看到该城所有军官、谁是城守、等级
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

	// ★ 无线电侦听≥10级: 才能看到玩家最后在线时间（取账号最近活跃时间）
	if scoutLv >= 10 {
		var owner model.User
		if err := h.home().First(&owner, target.UserID).Error; err == nil && owner.LastActiveAt != nil {
			b.WriteString("最后在线时间：" + owner.LastActiveAt.Format("2006-01-02 15:04:05") + "\n")
		}
	}

	if scoutLv < 10 {
		fmt.Fprintf(&b, "（无线电侦听%d级, 部分情报无法查清）\n", scoutLv)
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

// ezfyLootCityPrestige 掠夺**玩家城市**（order_type=2, target_type=3）的军功声望。
//
// ★★ 2026-10-30 新增（修复「掠夺资源刷声望」）：
//
//	原实现 = 200 + 掠夺资源总量/10000。后期一座城能囤几十亿资源，单次掠夺 30% 就是
//	十亿级，除以 1 万仍然有 30 万 → 一场掠夺 +312679 声望（线上战报实锤），
//	有人专门找资源多的城反复掠夺刷军衔。声望是军衔晋升的唯一来源，必须收口。
//
// 现在的口径：固定基础 200（打赢掠夺该给的功劳），资源只作为**小额**加成，
// 且用开方压缩 + 封顶 100：
//
//	1 万资源  → +1      100 万资源 → +10
//	1 亿资源  → +100    更多      → +100（封顶）
//
// 单场上限 300，低于「征服玩家城 500」，与「寇城 80+20×等级 / 野地 30+10×等级」
// 处在同一量级 —— 掠夺仍是稳定收益，但不再能靠资源量刷出声望。
func ezfyLootCityPrestige(resTotal int64) int {
	const base = 200
	const maxBonus = 100
	if resTotal <= 0 {
		return base
	}
	bonus := int(math.Sqrt(float64(resTotal)) / 100)
	if bonus > maxBonus {
		bonus = maxBonus
	}
	return base + bonus
}

// ezfyPersonalHonor 战报尾部的「个人荣誉」= 军衔声望的开方压缩（**纯展示**，不参与任何结算）。
//
// ★★ 2026-10-31 用户反馈「个人荣誉:1333633 获取也太变态了，算法改下，少些」：
//
//	原来直接 `prestige / 6` —— 声望是**历史累计**，老玩家存量动辄几百万，
//	个人荣誉就跟着变成 7 位数，和旁边的「个人战绩 100」完全不是一个量级，看着像刷出来的。
//	现在改为开方（单调递增、增长极慢）：
//	  1 万声望 → 100 ／ 100 万 → 1,000 ／ 800 万 → 2,828 ／ 1 亿 → 10,000
//	即 1 亿声望的个人荣誉也才 1 万，不会再出现百万级。
//
// ⚠️ 这个数只出现在战报尾部的展示行，数据库里没有对应字段，改公式不需要迁移存量数据。
func ezfyPersonalHonor(prestige int) int {
	if prestige <= 0 {
		return 0
	}
	return int(math.Sqrt(float64(prestige)))
}

// battleStatsTail 战报尾部的战果统计段
// (复刻 `参考材料/开发文档/掠夺报告1.txt` 的 个人荣誉/个人战绩/军团战绩/掠夺比例 + [双方兵力];
// 军功声望与军官经验已在上文正文里输出, 这里不重复)
//
// ★ 2026-10-09 用户反馈「pvp 伤兵回收比例有问题」：这里的字段是**掠夺比例**（战利品占目标资源的比例），
// 与「伤兵回收率」完全无关 —— 但原字段名「回收比例」和游戏里「机械改装：回收率+5%」撞词，
// 玩家在战报里看到「回收比例:0%」（平局没掠夺到东西）就以为伤兵回收是 0%。改名为「掠夺比例」消除歧义。
//
// ★ 2026-10-09 同日追加：在「掠夺比例」后面**再加一行「回收比例」**，
// 这次专指**伤兵回收比例**（入伤兵营的占比，见 ezfyHealPctLine）——
// 两个字段含义不再重叠：掠夺比例 = 战利品，回收比例 = 伤兵。
// 由调用方按需传入（healPct 可变参数）：没有伤兵数据的早期返回分支不传 → 不输出这一行。
func (h *EzfyHandler) battleStatsTail(uid uint, prestigeGain, recyclePct int, healPct ...string) string {
	hp := ""
	if len(healPct) > 0 {
		hp = healPct[0]
	}
	profile := h.ensureProfile(uid)
	return fmt.Sprintf("\n个人荣誉:%d\n个人战绩:%d\n军团战绩:%d\n掠夺比例:%d%%%s\n[双方兵力]",
		ezfyPersonalHonor(profile.Prestige), prestigeGain, 0, recyclePct, hp)
}

// ezfyWoundedReportLine 战报里的「攻方伤兵入营」行。
//
// ★ 2026-10-09 用户反馈「pvp 伤兵回收比例应该有问题」——实测这一行的数值**是对的**
// （攻方战损 10697、航母修复率 20% + 满级战地医疗 +20% → 4278 = 40%），
// 但战报只写一个数字，玩家拿它跟「兵种详情页的修复率 20%」一对就以为算错了。
// 这里把**占比**和**修复率的额外加成**一并写出来，一眼能对账。
//
// ⚠️ 2026-10-09 同日：战报里**不再输出**这一行（尾部「回收比例」已经汇总了
// 攻守双方的占比，两行比例信息重复）。**函数保留** —— 伤兵结算逻辑（`rate += healTech`）
// 和 `repairedTotal` 仍在用；若以后要在战报里恢复伤兵数量，把调用加回 processArrive 即可。
func ezfyWoundedReportLine(repaired int64, losses []ezfyUnitGroup, healTech int) string {
	return ezfyWoundedReportLineSide("攻方", repaired, losses, healTech)
}

// ezfyWoundedReportLineSide 同 ezfyWoundedReportLine，但可指定归属方（「攻方」/「守方」）。
//
// ★ 2026-10-09 PvP 守方战损也进伤兵营后，战报里攻守各要一行，避免看错是谁的伤兵。
func ezfyWoundedReportLineSide(side string, repaired int64, losses []ezfyUnitGroup, healTech int) string {
	if repaired <= 0 {
		return ""
	}
	var dead int64
	for _, g := range losses {
		dead += g.Count
	}
	pct := int64(0)
	if dead > 0 {
		// ★ 四舍五入：伤兵数是向下取整来的（10697×40% = 4278.8 → 4278），
		//   直接整除会显示成 39%，玩家又该以为算错了。
		pct = (repaired*100 + dead/2) / dead
	}
	src := ""
	if healTech > 0 {
		src = fmt.Sprintf("，修复率额外加成+%d%%", healTech)
	}
	return fmt.Sprintf("\n%s伤兵入营: %d(占%s战损 %d 的 %d%%%s)", side, repaired, side, dead, pct, src)
}

// ezfyHealPctLine 战报「回收比例」行：**看战报这一方自己**的伤兵回收占比。
//
// ★ 2026-10-09 用户口径（两轮）：
//
//	①「回收比例就是伤兵回收比例 总的 基础+科技+军官技能（入伤兵营的）」；
//	②「回收比例 谁看展示谁的」→ 攻方看的战报显示攻方的比例、守方看的显示守方的比例，
//	  所以这里只接收**一方**的数据（调用方按视角各算一份）。
//
// 口径与「伤兵入营」行完全一致：占比 = 入营伤兵 ÷ 该方战损（四舍五入到整数百分点）。
// ★ 与「掠夺比例」是两个不同的东西：掠夺比例 = 战利品占目标资源的比例（平局为 0）。
// 该方零战损（没死人）→ 返回空串，不输出这一行。
func ezfyHealPctLine(repaired int64, losses []ezfyUnitGroup) string {
	var dead int64
	for _, g := range losses {
		dead += g.Count
	}
	if dead <= 0 {
		return ""
	}
	// 与伤兵行同样的四舍五入口径（伤兵数本身是向下取整来的）
	pct := (repaired*100 + dead/2) / dead
	return fmt.Sprintf("\n回收比例: %d%%", pct)
}

// ★ 2026-10-13 性能：拆成「构造(newReport) + 落库(addReport)」两半，
// 批量结算（一键收获）可以把整批战报攒起来一次 CreateInBatches，省 N 次跨库往返。
func (h *EzfyHandler) addReport(uid uint, reportType int, title, content string, detailAndOrder ...interface{}) uint {
	// ★★ 2026-10-31 战报入库开关（管理端「二战系统配置」按类型分开，默认都入库）：
	//   关掉的类型**直接不入库**并返回 0 —— 结算逻辑照常（资源/声望/经验都不受影响），
	//   只是玩家军情里看不到这类战报、也不占未读角标。调用方把返回值当「可空战报 id」用。
	if !ezfyReportSaveEnabled(reportType) {
		return 0
	}
	// ★ 2026-10-13 修复：原来写成 `h.DB.Create(h.newReport(...))` —— **传的是值不是指针**，
	//   GORM 在 ConvertToCreateValues 里对主键 SetInt 会 panic
	//   `reflect: reflect.Value.SetInt using unaddressable value`，
	//   任何走 addReport 的路径（将领升级战报 / 一键收获）都会打挂整个进程。
	//   必须先落到变量再取地址（与 ezfy_settle_env.go 的 flush 同款写法）。
	r := h.newReport(uid, reportType, title, content, detailAndOrder...)
	h.DB.Create(&r)
	// ★ 2026-10-26 返回新战报 id：风云榜积分明细要回填「本场战报」，
	//   供管理端「积分明细 → 查看战报」。老调用方忽略返回值即可。
	return r.ID
}

// newReport 构造一条战报（不落库）。
func (h *EzfyHandler) newReport(uid uint, reportType int, title, content string, detailAndOrder ...interface{}) model.EzfyReport {
	r := model.EzfyReport{UserID: uid, ReportType: reportType, Title: title,
		// ★ 2026-10-08 写战报时即落 type_name（真实类型名），供管理端 SQL 过滤，见 EzfyReportTypeName。
		TypeName: EzfyReportTypeName(reportType, title),
		Content:  content, IsRead: 0}
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
	return r
}
