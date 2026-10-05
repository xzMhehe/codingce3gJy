package ezfy

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"sync"
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

// fail 把「空字符串 = 成功」的动作结果转成响应。
//
// ⚠️ 成功时只说「操作成功」——玩家看不出刚才干了什么。
// 新代码请优先用 done() 传具体文案（用户要求：建造就说建造成功）。
func (h *EzfyHandler) fail(c *gin.Context, msg string) {
	if msg == "" {
		resp.OKMsg(c, "操作成功", nil)
		return
	}
	resp.ParamError(c, msg)
}

// done 动作结果 → 响应：成功用调用方给的具体文案，失败用动作返回的错误文案。
//
// ★ 用户反馈「用户端消息提示还都是 ok」：这些动作原来走 fail()，
// 成功时统一显示「ok」。现在每处都传具体文案，例如：
//
//	h.done(c, h.buildBuilding(city, id), "建造成功")
func (h *EzfyHandler) done(c *gin.Context, msg, successMsg string) {
	if msg == "" {
		resp.OKMsg(c, successMsg, nil)
		return
	}
	resp.ParamError(c, msg)
}

// ============ 建筑 ============

// ★ 2026-10-04 性能（用户反馈「/buildings 线上 4s」）：
//   原来每栋建筑调 buildingMaxLevel（内部再查一次完整建筑列表）= 几十条 RDS 往返，
//   加上 areaCounts ×2、refreshCity 的订单结算 —— 单次请求 40+ 条查询。
//   现在：建筑列表只查一次；市政厅等级内存取值；军事/资源区数量用 areaCountsOf 纯内存；
//   懒结算改走 refreshCityRead（跳过订单结算）；再加 3s 玩家级缓存兜底。
func (h *EzfyHandler) Buildings(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	if it, ok := ezfyPageCacheGet(uid, "buildings"); ok {
		resp.OK(c, it)
		return
	}
	_, city, cities := h.ezfyPageCity(uid)
	// ★★ 2026-10-05 性能：并行块一次性取齐「展示 + 懒结算」所需的全部只读数据，
	//   下面的懒结算（calcResourceD）因此零额外查询 —— 改造前 calcResource 会把
	//   科技/野地/部队/增产令/市长加成**再串行查一遍**（跨 WAN 白打 6 个往返）。
	var (
		list     []model.EzfyCityBuilding
		qs       []model.EzfyTrainQueue
		techRows []model.EzfyCityTech
		techs    map[int]int
		wilds    []model.EzfyWildland
		troops   map[int]int64
		boost    *model.EzfyCityEffect
		mayor    int
	)
	cids := cityIdsOf(cities)
	var wg sync.WaitGroup
	wg.Add(8)
	go func() { defer wg.Done(); list = h.buildingList(city.ID) }()
	go func() { defer wg.Done(); h.DB.Where("city_id = ? AND status = 0", city.ID).Order("start_time ASC").Find(&qs) }()
	go func() { defer wg.Done(); h.DB.Where("city_id IN ? AND status = 1", cids).Find(&techRows) }()
	go func() { defer wg.Done(); techs = h.techMapOf(uid) }()
	go func() { defer wg.Done(); wilds = h.wildlandList(city.ID) }()
	go func() { defer wg.Done(); troops = h.troopMap(city.ID) }()
	go func() { defer wg.Done(); boost = h.ezfyLoadActiveBoost(city.ID) }()
	go func() { defer wg.Done(); mayor = h.mayorBonusPct(city.ID) }()
	wg.Wait()
	snap := &resCalcData{buildings: list, techs: techs, wilds: wilds, troops: troops,
		boost: boost, boostDone: true, mayor: mayor}
	// 懒结算复用已取数据（零额外查询）
	h.checkBuildingDone(&city, list)
	h.checkTechDoneRows(&city, techRows)
	h.collectTrainQueue(&city, qs)
	h.calcResourceD(&city, snap)
	// ★ 市政厅等级一次内存取值（原每栋建筑各查一遍建筑列表）
	hallLevel := 0
	for _, b := range list {
		if b.BuildingId == 1 && b.Level > hallLevel {
			hallLevel = b.Level
		}
	}
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
			// ★ 等级上限按用户规则（市政厅10 / 参谋部·司令部·民居12 / 其他10，民居受市政厅约束）
			view["max_level"] = ezfyBuildingMaxLevel(b.BuildingId, hallLevel)
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
	// 可建造池: 复刻原版 BuildingController.buildList（内部直接用传入的 list，不再重复查库）
	pool := h.buildPool(&city, list)
	// ★ 第九轮：军事区 / 资源区数量上限分开（线上现值各 36，管理端可维护）
	mil, res := areaCountsOf(list)
	lim := ezfyLimit()
	data := gin.H{
		"buildings": views, "pool": pool,
		"area_count": mil + res, "area_cap": lim.MilitaryMax + lim.ResourceMax,
		"military_count": mil, "military_cap": lim.MilitaryMax,
		"resource_count": res, "resource_cap": lim.ResourceMax,
	}
	ezfyPageCacheSet(uid, "buildings", data)
	resp.OK(c, data)
}

// buildPool 返回该城当前可建造的建筑池(复刻原版 BuildingController.buildList)。
//   - 军事区 = type 2/3: 军工厂可建 5 个、民居可建 10 个, 其余每类限 1 个
//   - 资源区 = type 1: 农田/炼钢厂/石油基地/稀矿厂 **可重复建造**, 只受建筑总数上限约束
//   - 市政厅(type 4)自动存在, 不入池; 航海协会(19)仅沿海城市可建
func (h *EzfyHandler) buildPool(city *model.EzfyCity, list []model.EzfyCityBuilding) []gin.H {
	cntOf := map[int]int{}
	for _, b := range list {
		cntOf[b.BuildingId]++
	}
	ids := make([]int, 0, len(ezfyCfg.buildings))
	for id := range ezfyCfg.buildings {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	// ★ 军事区/资源区各自有**硬上限**（线上现值各 36，管理端可维护）：
	//   即使军工厂/民居设了「不限数量」，也不能超过所属区域的总数上限（用户规则）。
	//   pool 这里必须和 buildBuilding 一致地按区域卡，否则会出现「队列里能点、一建就报已达上限」。
	//   ★ 2026-10-04 性能：直接用传入的 list 纯内存统计（原来内部再查一次完整建筑列表）
	mil, res := areaCountsOf(list)
	pool := []gin.H{}
	for _, id := range ids {
		cfg := ezfyCfg.buildings[id]
		if cfg.Type != 1 && cfg.Type != 2 && cfg.Type != 3 {
			continue
		}
		lv := ezfyCfg.buildingLevel(id, 1)
		if lv == nil {
			continue
		}
		var can bool
		lim := ezfyLimit()
		switch {
		case cfg.Type == 1:
			// 资源建筑可重复建造(原版资源区 can = true)
			can = true
		case id == ezfyFactoryBuildingID:
			// ★ 第九轮用户规则：军工厂**不限数量**（但受军事区总数上限约束）
			can = true
		case id == 2:
			can = cntOf[id] < lim.HouseMax
		default:
			can = cntOf[id] == 0
		}
		// ★ 区域总数硬上限：满了就不再出现在可建队列里（与 buildBuilding 口径一致）
		if can {
			if cfg.Type == 1 {
				can = res < lim.ResourceMax
			} else {
				can = mil < lim.MilitaryMax
			}
		}
		// ★ 第九轮：航海协会(19) 只能建在【海城】(沿海平原)，陆城不得建造
		if id == 19 && !h.isSeaCity(city) {
			can = false
		}
		if !can {
			continue
		}
		pool = append(pool, gin.H{
			"building_id": id, "name": cfg.Name, "type": cfg.Type,
			"max_level": cfg.MaxLevel, "des": cfg.Des, "effect": lv.Effect,
			"cost": gin.H{"food": lv.Food, "steel": lv.Steel, "oil": lv.Oil, "rare": lv.Rare, "gold": lv.Gold},
			"time": lv.BuildTime, "built_count": cntOf[id],
		})
	}
	return pool
}

func (h *EzfyHandler) Build(c *gin.Context) {
	uid := middleware.GetUID(c)
	ezfyPageCacheDel(uid) // 建造 → 建筑列表/可建造池缓存失效
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
	h.done(c, h.buildBuilding(city, req.BuildingId), "建造命令已下达")
}

func (h *EzfyHandler) Upgrade(c *gin.Context) {
	uid := middleware.GetUID(c)
	ezfyPageCacheDel(uid) // 升级 → 建筑列表缓存失效
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
	h.done(c, h.upgradeBuilding(city, req.RecordId), "建筑已开始升级")
}

// MaxLevel 一键升级建筑
//
// ★ 2026-09-25 用户纠正「一键9级 不对，是一键升级到 9 级，而不是升级满」：
//   按钮文案是「一键{{max_level-1}}级」，那就必须**升到那一级为止**（停在 9 级，
//   不越过 9→10 这道要建筑图纸的坎、也不升到满级）。target_level 由前端下发，
//   后端按「目标等级」结算资源与图纸；没带目标等级时仍按「升到满级」兼容。
func (h *EzfyHandler) MaxLevel(c *gin.Context) {
	uid := middleware.GetUID(c)
	ezfyPageCacheDel(uid) // 一键升级 → 建筑列表缓存失效
	var req struct {
		CityId   int64 `json:"city_id"`
		RecordId int64 `json:"record_id"`
		// 目标等级（0 = 不带，按建筑上限升满）
		TargetLevel int `json:"target_level"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	h.cfgs()
	city := h.bodyCity(uid, req.CityId)
	target, msg := h.maxLevelBuilding(city, req.RecordId, req.TargetLevel)
	// ★ 2026-09-29 修复：不能走 h.done(c, msg, msg) —— done 把「非空 msg」一律当错误，
	//   会令**成功**的一键升级也返回 400（把成功文案显示成报错），前端因此不刷新。
	//   成功走 OK，失败才 ParamError。
	if msg == "" {
		if target > 0 {
			msg = fmt.Sprintf("建筑已一键升级到%d级", target)
		} else {
			msg = "建筑已一键升级"
		}
		resp.OKMsg(c, msg, nil)
		return
	}
	resp.ParamError(c, msg)
}

// CancelBuilding POST /games/ezfy/building/cancel
//
// ★ 2026-09-26 用户要求：「已升级的建筑（施工中）用户端去掉（多余的升级按钮），
// 加个升级状态时 [取消] 功能」——取消施工并**全额退还**已扣资源与图纸。
func (h *EzfyHandler) CancelBuilding(c *gin.Context) {
	uid := middleware.GetUID(c)
	ezfyPageCacheDel(uid) // 取消施工 → 建筑列表缓存失效
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
	// ⚠️ 不能直接用 h.done：它把「非空 msg」一律当错误 → 退还文案会被当成失败返回 400。
	msg := h.cancelBuildingUpgrade(city, req.RecordId)
	// 「已取消升级」「该建筑已完成…」都算成功路径（后者=建筑刚好完工、保留了新等级）
	if strings.HasPrefix(msg, "已取消升级") || strings.HasPrefix(msg, "该建筑已完成") {
		resp.OK(c, gin.H{"msg": msg})
		return
	}
	resp.ParamError(c, msg)
}

func (h *EzfyHandler) DeleteBuilding(c *gin.Context) {
	uid := middleware.GetUID(c)
	ezfyPageCacheDel(uid) // 拆除 → 建筑列表缓存失效
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
	h.done(c, h.deleteBuilding(city, req.RecordId), "建筑已拆除")
}

func (h *EzfyHandler) SpeedBuilding(c *gin.Context) {
	uid := middleware.GetUID(c)
	ezfyPageCacheDel(uid) // 加速 → 建筑列表缓存失效
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
	h.done(c, h.speedUpBuilding(city, req.RecordId, req.Minutes), "建筑已加速完成")
}

// ============ 军队 ============

// ★ 2026-10-04 性能（用户反馈「/troops 线上 2s+」）：
//   原来 20+ 条查询全部串行（refreshCity 订单结算 + 每处 buildingList/troopMap 重复查）。
//   现在：懒结算改走 refreshCityRead（跳过订单结算）；只读查询并入并行块（1 个 RTT）；
//   建筑相关（围墙等级/军工厂座数与总等级）与城防占用全部用已取数据纯内存算；
//   再加 3s 玩家级缓存，训练/拆除/解散/伤兵恢复等写操作统一失效。
func (h *EzfyHandler) Troops(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	if it, ok := ezfyPageCacheGet(uid, "troops"); ok {
		resp.OK(c, it)
		return
	}
	// ★ 2026-10-04 第一波 档案+城市列表（1 RTT）定当前城，第二波下方并行
	profile, city, _ := h.ezfyPageCity(uid)
	camp := profile.Camp

	var (
		troopMap  map[int]int64
		qs        []model.EzfyTrainQueue
		wounded   []model.EzfyWounded
		deserters []model.EzfyWounded
		popUsed   int64
		buildings []model.EzfyCityBuilding
	)
	var wg sync.WaitGroup
	// ★ 2026-10-05 性能：已占用人口不再单独查库（原来 buildingPop/troopPop 各查一次），
	//   wg.Wait() 之后用本请求已取到的 buildings + qs 纯内存算。
	wg.Add(5)
	go func() { defer wg.Done(); troopMap = h.troopMap(city.ID) }()
	go func() { defer wg.Done(); h.DB.Where("city_id = ? AND status = 0", city.ID).Order("start_time ASC").Find(&qs) }()
	go func() { defer wg.Done(); h.DB.Where("city_id = ? AND type = 0", city.ID).Order("troop_id ASC").Find(&wounded) }()
	go func() { defer wg.Done(); h.DB.Where("city_id = ? AND type = 1", city.ID).Order("troop_id ASC").Find(&deserters) }()
	go func() { defer wg.Done(); buildings = h.buildingList(city.ID) }()
	wg.Wait()
	popUsed = h.cityPopUsedD(city.ID, buildings, qs)
	// 懒结算复用已取数据（零额外查询）
	// ★ 2026-10-05 性能（用户反馈「/troops 还是 2s」）：原来这里还跑 checkTechDone +
	//   calcResource（各 2~4 条串行跨 WAN 查询），是本页 2s 的根源。军队页不展示科技/资源，
	//   科技/资源懒结算交给 /view 轮询照常推进（最多滞后一轮轮询），这里只保留
	//   建筑完工 + 训练队列出厂两个**纯内存、空闲零写**的结算。
	h.checkBuildingDone(&city, buildings)
	h.collectTrainQueue(&city, qs)
	// ★ 2026-09-23：超过「伤兵存活天数」还没救治的伤兵直接消失（用户要求 5 天）
	wounded = h.filterExpiredWounded(wounded)
	deserters = h.filterExpiredWounded(deserters)

	troopViews := []gin.H{}
	for tid, count := range troopMap {
		cfg := ezfyCfg.troop(tid)
		if cfg == nil {
			continue
		}
		troopViews = append(troopViews, gin.H{"troop_id": tid, "name": ezfyCfg.troopName(tid, camp),
			"count": count, "type": cfg.Type})
	}
	queues := []gin.H{}
	for _, q := range qs {
		queues = append(queues, gin.H{"id": q.ID, "troop_id": q.TroopId,
			"name": ezfyCfg.troopName(q.TroopId, camp), "count": q.Count, "end_time": q.EndTime})
	}
	woundViews := []gin.H{}
	for _, w := range append(wounded, deserters...) {
		woundViews = append(woundViews, gin.H{"id": w.ID, "troop_id": w.TroopId,
			"name": ezfyCfg.troopName(w.TroopId, camp), "type": w.Type, "count": w.Count,
			// ★ 用户要求「恢复伤兵需要黄金」：把单价一起下发，前端在[恢复]旁边显示要花多少钱
			"heal_gold": ezfyWoundHealGoldPer(w.TroopId)})
	}
	// 兵种配置一览
	cfgViews := []gin.H{}
	// ★ 2026-10-03 性能：兵种配置在 cfgs() 已整表载入进程内缓存，改用内存缓存，省一次跨 WAN 全表查询。
	allTroops := ezfyCfg.sortedTroops()
	// 节日活动·造兵打折: 列表展示的就是打折后的实际消耗
	discount := h.actPct(ezfyActTrain)
	// 已训练数量(城内部队), 供兵种详情页显示「现有:N」
	for _, t := range allTroops {
		f, s, o, r := h.trainCostWithActivity(t.Food, t.Steel, t.Oil, t.Rare)
		cfgViews = append(cfgViews, gin.H{
			"id": t.ID, "name": ezfyCfg.troopName(t.ID, camp), "type": t.Type,
			"health": t.Health, "defence": t.Defence, "speed": t.Speed, "attack_range": t.AttackRange,
			"carry": t.Carry, "pop": t.Pop, "require": t.Require,
			"atk_sea": t.AtkSea, "atk_ground": t.AtkGround, "atk_air": t.AtkAir, "atk_def": t.AtkDef,
			"food_keep": t.FoodKeep, "oil_keep": t.OilKeep,
			"icon": t.Icon, "repair_rate": t.RepairRate,
			// 「军工厂(N级)」从 require 里解析(复刻 createTroop.html 的「需要军工厂：N级」)
			"need_factory": ezfyNeedFactoryLevel(t.Require),
			"cost":         gin.H{"food": f, "steel": s, "oil": o, "rare": r},
			"raw_cost":     gin.H{"food": t.Food, "steel": t.Steel, "oil": t.Oil, "rare": t.Rare},
			"train_time":   t.TrainTime,
		})
	}
	// 城防空间(围墙容量)与已占用(复刻 troopDefence.html 的「围墙：N级 城防空间：(used/cap)」)
	// ★ 已占用含训练队列里还没出来的城防，与 trainTroop 的校验口径保持一致
	blv := buildingLevelsOf(buildings)
	wallLevel := blv[7]
	defSpace := int64(0)
	if wall := ezfyCfg.buildingLevel(7, wallLevel); wall != nil {
		defSpace = wall.Capacity
	}
	// ★ 2026-10-04 城防占用直接用上面已取到的军队表+训练队列纯内存算（原来再查 2 遍）
	defUsed := int64(0)
	for tid, cnt := range troopMap {
		if c := ezfyCfg.troop(tid); c != nil && c.Type == 4 {
			defUsed += cnt
		}
	}
	for _, q := range qs {
		if c := ezfyCfg.troop(q.TroopId); c != nil && c.Type == 4 {
			defUsed += q.Count
		}
	}
	// 军工厂座数与总等级（复刻 createTroop.html 的「全部工厂 / 仅此工厂」）——纯内存
	factoryTotal, factoryCount := 0, 0
	for _, b := range buildings {
		if b.BuildingId == ezfyFactoryBuildingID {
			factoryCount++
			factoryTotal += b.Level
		}
	}
	data := gin.H{
		"city": city, "troops": troopViews, "queues": queues, "wounded": woundViews,
		"pop": city.Pop, "pop_used": popUsed, "cfgs": cfgViews,
		"wall_level":         wallLevel,
		"train_discount":     discount,
		"defence_space":      defSpace,
		"defence_space_used": defUsed,
		"factory_total":      factoryTotal,
		"factory_count":      factoryCount,
	}
	ezfyPageCacheSet(uid, "troops", data)
	resp.OK(c, data)
}

// DismissDefence POST /games/ezfy/troops/dismiss —— 拆除城防设施
// 复刻 troopDefence.html 每行的 [拆除]（原版 Java 无对应接口, 模板里是失效的旧链接）
func (h *EzfyHandler) DismissDefence(c *gin.Context) {
	uid := middleware.GetUID(c)
	ezfyPageCacheDel(uid) // 拆城防 → 军队页缓存失效
	var req struct {
		CityId  int64 `json:"city_id"`
		TroopId int   `json:"troop_id"`
		Count   int64 `json:"count"` // 0 或省略 = 全部拆除
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	h.cfgs()
	cfg := ezfyCfg.troop(req.TroopId)
	if cfg == nil || cfg.Type != 4 {
		resp.ParamError(c, "该兵种不是城防设施")
		return
	}
	city := h.bodyCity(uid, req.CityId)
	var ct model.EzfyCityTroop
	if err := h.DB.Where("city_id = ? AND troop_id = ?", city.ID, req.TroopId).First(&ct).Error; err != nil || ct.Count <= 0 {
		resp.ParamError(c, "城内没有该城防设施")
		return
	}
	n := ct.Count
	if req.Count > 0 && req.Count < n {
		n = req.Count
	}
	if ct.Count-n <= 0 {
		h.DB.Delete(&model.EzfyCityTroop{}, ct.ID)
	} else {
		h.DB.Model(&model.EzfyCityTroop{}).Where("id = ?", ct.ID).Update("count", ct.Count-n)
	}
	name := ezfyCfg.troopName(req.TroopId, h.ensureProfile(uid).Camp)
	resp.OK(c, gin.H{"msg": fmt.Sprintf("已拆除 %s×%d, 城防空间已释放", name, n)})
}

// ezfyNeedFactoryLevel 从 require 文本里解析「军工厂(N级)」的 N, 找不到返回 0
// (复刻 createTroop.html 的「需要军工厂：N级」)
func ezfyNeedFactoryLevel(require string) int {
	const key = "军工厂("
	i := strings.Index(require, key)
	if i < 0 {
		return 0
	}
	rest := require[i+len(key):]
	j := strings.Index(rest, "级)")
	if j < 0 {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(rest[:j]))
	if err != nil {
		return 0
	}
	return n
}

func (h *EzfyHandler) Train(c *gin.Context) {
	uid := middleware.GetUID(c)
	ezfyPageCacheDel(uid) // 征兵 → 军队页缓存失效
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
	h.done(c, h.trainTroop(city, req.TroopId, req.Count, req.Split), "征兵已开始")
}

// CancelTrain POST /games/ezfy/troops/train/cancel {queue_id}
//
// 用户要求：征兵队列玩家可以自己取消。取消时把当初消耗的资源全额退还。
func (h *EzfyHandler) CancelTrain(c *gin.Context) {
	uid := middleware.GetUID(c)
	ezfyPageCacheDel(uid) // 取消训练 → 军队页缓存失效
	var req struct {
		QueueId int64 `json:"queue_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.QueueId <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	h.cfgs()
	var q model.EzfyTrainQueue
	if err := h.DB.First(&q, req.QueueId).Error; err != nil {
		resp.NotFound(c, "训练队列不存在")
		return
	}
	city := h.cityOf(uid, q.CityId)
	if city == nil {
		resp.ParamError(c, "该队列不属于你")
		return
	}
	// 先把到点的队列结算掉，避免「马上就要完成时取消」的歧义
	h.refreshCity(uid, city)
	var q2 model.EzfyTrainQueue
	if err := h.DB.First(&q2, req.QueueId).Error; err != nil || q2.Status != 0 {
		resp.ParamError(c, "该队列已完成，无法取消")
		return
	}
	cfg := ezfyCfg.troop(q.TroopId)
	if cfg == nil {
		resp.ParamError(c, "兵种配置不存在")
		return
	}
	// ★ 免费征兵（开关关着建的队列）没扣过资源 → 取消时**不退还**，
	//   否则「趁开关关着排队、等开关打开再取消」就能凭空换出资源。
	//   注意这里用队列上的 Free 标记（建队列时的状态），而不是当前开关状态。
	if q.FreeTrain != 0 {
		h.DB.Delete(&model.EzfyTrainQueue{}, q.ID)
		resp.OK(c, gin.H{"msg": fmt.Sprintf("已取消「%s×%d」的训练（免费征兵，不退还资源）", cfg.Name, q.Count)})
		return
	}
	food := cfg.Food * q.Count
	steel := cfg.Steel * q.Count
	oil := cfg.Oil * q.Count
	rare := cfg.Rare * q.Count
	// 与训练时同一套折扣算法，保证退还基数 = 当初扣的
	food, steel, oil, rare = h.trainCostWithActivity(food, steel, oil, rare)
	// ★ 第九轮用户规则：取消训练要收手续费（按常见游戏 10%），
	//   且退还**不受仓储上限影响**（原实现被 FoodCap 截断，玩家会觉得「退少了」）。
	fee := int64(ezfyCancelTrainFeePct)
	food -= food * fee / 100
	steel -= steel * fee / 100
	oil -= oil * fee / 100
	rare -= rare * fee / 100
	h.giveResNoCap(city, food, steel, oil, rare, 0)
	h.DB.Delete(&model.EzfyTrainQueue{}, q.ID)
	resp.OK(c, gin.H{"msg": fmt.Sprintf("已取消「%s×%d」的训练，扣除%d%%手续费后退还 粮%d 钢%d 油%d 稀矿%d",
		cfg.Name, q.Count, ezfyCancelTrainFeePct, food, steel, oil, rare)})
}

// DisbandTroops 解散部队（用户规则：军队页面要有解散按钮，数量由玩家自己输入）
//
// 解散直接销毁兵力（不退还任何资源），用于清理占地力的低级兵。
func (h *EzfyHandler) DisbandTroops(c *gin.Context) {
	uid := middleware.GetUID(c)
	ezfyPageCacheDel(uid) // 解散部队 → 军队页缓存失效
	var req struct {
		CityId  int64 `json:"city_id"`
		TroopId int   `json:"troop_id"`
		Count   int64 `json:"count"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.TroopId <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	h.cfgs()
	city := h.bodyCity(uid, req.CityId)
	h.refreshCity(uid, city)
	cfg := ezfyCfg.troop(req.TroopId)
	if cfg == nil {
		resp.ParamError(c, "兵种不存在")
		return
	}
	if req.Count <= 0 {
		resp.ParamError(c, "解散数量必须大于 0")
		return
	}
	owned := h.troopMap(city.ID)[req.TroopId]
	if owned <= 0 {
		resp.ParamError(c, fmt.Sprintf("本城没有「%s」", cfg.Name))
		return
	}
	if req.Count > owned {
		resp.ParamError(c, fmt.Sprintf("兵力不足: 「%s」只有%d", cfg.Name, owned))
		return
	}
	remain := owned - req.Count
	if remain == 0 {
		h.DB.Where("city_id = ? AND troop_id = ?", city.ID, req.TroopId).Delete(&model.EzfyCityTroop{})
	} else {
		h.DB.Model(&model.EzfyCityTroop{}).Where("city_id = ? AND troop_id = ?", city.ID, req.TroopId).
			Update("count", remain)
	}
	resp.OK(c, gin.H{"msg": fmt.Sprintf("已解散「%s」×%d，剩余%d", cfg.Name, req.Count, remain),
		"remain": remain})
}

// ezfySpeedGoldPerSec 训练一键加速原价: 每剩余 1 秒 10 黄金
// ★ 实际收费再乘管理端「二战系统配置 → 训练加速黄金倍率」(speed_train_rate)，
//
//	节假日想便宜点就把倍率调低（见 ezfySpeedTrainRate）。
const ezfySpeedGoldPerSec = 10

// SpeedTrainAll POST /games/ezfy/troops/speed-all —— 训练一键加速
// all_city=true 时对所有城市生效(复刻 militaryIndex.html 底部的两个按钮)
func (h *EzfyHandler) SpeedTrainAll(c *gin.Context) {
	uid := middleware.GetUID(c)
	ezfyPageCacheDel(uid) // 训练一键加速 → 军队页缓存失效
	var req struct {
		CityId  int64 `json:"city_id"`
		AllCity bool  `json:"all_city"`
	}
	_ = c.ShouldBindJSON(&req)
	h.cfgs()

	// ★ 2026-10-05 5 秒卡控（用户要求：前后端双保险，防连点/脚本反复刷黄金结算）
	{
		nowCd := time.Now().UnixMilli()
		ezfySpeedTrainMu.Lock()
		if last, ok := ezfySpeedTrainMemo[uid]; ok && nowCd-last < 5000 {
			ezfySpeedTrainMu.Unlock()
			resp.ParamError(c, "操作过于频繁, 请 5 秒后再试")
			return
		}
		if len(ezfySpeedTrainMemo) > 16384 {
			ezfySpeedTrainMemo = map[uint]int64{}
		}
		ezfySpeedTrainMemo[uid] = nowCd
		ezfySpeedTrainMu.Unlock()
	}

	// 串行化整个「读剩余秒数→算钱→扣费→清空队列」，杜绝连点并发结算（见 ezfySpeedTrainLocks）。
	lock := ezfySpeedTrainLock(uid)
	lock.Lock()
	defer lock.Unlock()

	now := time.Now().UnixMilli()

	targets := []model.EzfyCity{}
	if req.AllCity {
		h.DB.Where("user_id = ?", uid).Find(&targets)
	} else {
		targets = append(targets, *h.bodyCity(uid, req.CityId))
	}
	if len(targets) == 0 {
		resp.ParamError(c, "没有可加速的城市")
		return
	}
	ids := []int64{}
	for _, t := range targets {
		ids = append(ids, int64(t.ID))
	}
	var queues []model.EzfyTrainQueue
	h.DB.Where("city_id IN ? AND status = 0", ids).Find(&queues)
	if len(queues) == 0 {
		resp.ParamError(c, "没有训练中的队列")
		return
	}
	var totalSec int64
	for _, q := range queues {
		if s := (q.EndTime - now) / 1000; s > 0 {
			totalSec += s
		}
	}
	// 连点第二发进来时队列已被上一发清零，剩余 0 秒：直接拒绝，不要走「扣 0 黄金」的成功分支。
	if totalSec <= 0 {
		resp.ParamError(c, "训练队列已全部完成，没有可加速的队列")
		return
	}
	cost := int64(float64(totalSec)*float64(ezfySpeedGoldPerSec)*ezfySpeedTrainRate() + 0.5)
	if cost < 1 {
		cost = 1
	}
	main := h.getOrCreateCity(uid)
	if main.Gold < cost {
		resp.ParamError(c, fmt.Sprintf("黄金不足: 一键加速需%d黄金(剩余%d秒), 当前只有%d", cost, totalSec, main.Gold))
		return
	}
	main.Gold -= cost
	h.saveCityRes(&main)
	h.DB.Model(&model.EzfyTrainQueue{}).Where("city_id IN ? AND status = 0", ids).Update("end_time", now)
	scope := "本城"
	if req.AllCity {
		scope = "所有城市"
	}
	resp.OK(c, gin.H{"msg": fmt.Sprintf("%s训练一键加速完成: %d 个队列, 消耗%d黄金", scope, len(queues), cost)})
}

func (h *EzfyHandler) RecoverWounded(c *gin.Context) {
	uid := middleware.GetUID(c)
	ezfyPageCacheDel(uid) // 伤兵恢复 → 军队页缓存失效
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
		h.done(c, h.recoverAllWounded(city, req.Type), "伤兵已全部恢复")
		return
	}
	h.done(c, h.recoverWounded(city, req.TroopId, req.Type), "伤兵已恢复")
}

// ============ 科技 ============

func (h *EzfyHandler) Techs(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	// ★ 2026-09-28 多城研究：GET ?city_id= 指定查看的城市，0/缺省 = 主城
	// ★ 2026-10-05 修复「?city_id= 一直被忽略」：readCityReq 只绑 JSON body（GET 请求无效），
	//   直接读 query 参数选城，与前端 loadTechs 的调用（?city_id=X）对上。
	cityId, _ := strconv.ParseInt(c.Query("city_id"), 10, 64)
	city := h.bodyCity(uid, cityId)
	// ★ 2026-10-05 性能（用户反馈「/techs 还是 3s」）：原 refreshCityRead 串行跑
	//   checkBuildingDone / collectTrainQueue / calcResource（建筑、训练队列、军官各查一遍 +
	//   资源结算写库），跨 WAN RDS 就是 3s 的根源。科技页只需要「已完成科技」结算：
	//   第一波并行取 城市id列表 / 科研中心等级 / 科技配置表（1 RTT）；
	//   再一条 IN 查全部进行中科技行 → checkTechDoneRows（仅过期行写库，空闲零写）→
	//   用户级科技等级与研究态都由这批数据纯内存构建，去掉全部无关懒结算。
	var cityIds []uint
	var academy int
	var all []model.EzfyCfgTech
	var wg sync.WaitGroup
	wg.Add(3)
	go func() { defer wg.Done(); cityIds = h.ezfyCityIds(uid) }()
	go func() { defer wg.Done(); academy = h.buildingLevel(city.ID, 8) }() // 当前城科研中心等级
	go func() { defer wg.Done(); h.DB.Order("id ASC").Find(&all) }()
	wg.Wait()

	// 进行中的科技记录：全玩家城市一次 IN 查完；完成结算 + 研究态都从这批行构建（零多余查询）
	var techRows []model.EzfyCityTech
	h.DB.Where("city_id IN ? AND status = 1", cityIds).Find(&techRows)
	h.checkTechDoneRows(city, techRows)

	// 用户级科技等级（全城共用）：checkTechDoneRows 已把完成的 +1 写库，这里取到新等级
	var userTechs []model.EzfyUserTech
	h.DB.Where("user_id = ?", uid).Find(&userTechs)
	techMap := map[int]int{}
	for _, t := range userTechs {
		techMap[t.TechId] = t.Level
	}

	// 研究中的判断 = 该科技在玩家任一城市是否有进行中记录（不同城不能研究同一科技）
	// ★ 刚被 checkTechDoneRows 结算完的行 Status=0 → 自动排除，不会还显示「研究中」
	researchMap := map[int]model.EzfyCityTech{}
	for i := range techRows {
		r := &techRows[i]
		if r.Status != 1 {
			continue
		}
		if _, ok := researchMap[r.TechId]; !ok {
			researchMap[r.TechId] = *r
		}
	}
	views := []gin.H{}
	for _, t := range all {
		level := techMap[t.ID]
		if rec, ok := researchMap[t.ID]; ok {
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
	data := gin.H{"techs": views, "academy": academy}
	resp.OK(c, data)
}

func (h *EzfyHandler) Research(c *gin.Context) {
	uid := middleware.GetUID(c)
	ezfyPageCacheDel(uid) // 开始研究 → 科技页缓存失效
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
	// ★ 2026-10-05 性能（用户反馈「/techs/research 3s」）：先一次并行取数 + 完整懒结算，
	//   再把快照交给 researchTech 复用（建筑等级/科技等级/进行中研究全走内存，零额外读）。
	d := h.ezfyActionSettle(uid, city)
	h.done(c, h.researchTech(city, req.TechId, d), "科技研究已开始")
}

func (h *EzfyHandler) SpeedTech(c *gin.Context) {
	uid := middleware.GetUID(c)
	ezfyPageCacheDel(uid) // 科技加速 → 科技页缓存失效
	var req struct {
		CityId int64 `json:"city_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	h.cfgs()
	city := h.bodyCity(uid, req.CityId)
	// ★ 2026-09-26 修复「加速道具买完实际使用不生效」：
	//
	//	原来这里直接 `h.speedUpTech(city, req.Minutes)` —— **不校验道具、不扣任何东西**，
	//	而且 Minutes 由客户端传（传 999999 就能把研究瞬间刷完）：
	//	  ① 是个经济漏洞（免费无限加速）；
	//	  ② 商城卖的「科技加速30分钟/2小时」(item_type=5) 因此**根本没有被消耗的地方**，
	//	     玩家买完在科技页点 [加速] 看着减了 10 分钟、道具却一个没少 →「买完用了不生效」。
	//	现在统一走「消耗科技加速道具」，与建筑页 [加速] 同一口径。
	cfgID := h.ezfyBestSpeedItem(uid, ezfyItemTypeTechSpeed)
	if cfgID == 0 {
		resp.ParamError(c, "没有科技加速道具，请到商城购买")
		return
	}
	msg := h.useItem(uid, city, cfgID, 1, 0, 0, 0)
	if !strings.HasPrefix(msg, "使用成功") {
		resp.ParamError(c, msg)
		return
	}
	resp.OK(c, gin.H{"msg": msg})
}

// CancelTech POST /games/ezfy/techs/cancel  {tech_id} —— 取消研究并全额退还消耗
func (h *EzfyHandler) CancelTech(c *gin.Context) {
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
	h.done(c, h.cancelTech(city, req.TechId), "已取消研究, 已消耗资源不退还")
}

// ============ 调整生产(开工率) ============

// ProduceInfo GET /games/ezfy/city/produce —— 复刻原版 city/sourceSet.html
func (h *EzfyHandler) ProduceInfo(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	city := h.getOrCreateCity(uid)
	h.refreshCity(uid, &city)
	resp.OK(c, gin.H{
		"rate_food": ezfyRate(city.RateFood), "rate_steel": ezfyRate(city.RateSteel),
		"rate_oil": ezfyRate(city.RateOil), "rate_rare": ezfyRate(city.RateRare),
		"city": gin.H{"id": city.ID, "name": city.Name, "x": city.X, "y": city.Y},
	})
}

// ProduceSet POST /games/ezfy/city/produce  {rate_food,rate_steel,rate_oil,rate_rare}
func (h *EzfyHandler) ProduceSet(c *gin.Context) {
	uid := middleware.GetUID(c)
	ezfyPageCacheDel(uid) // 改生产比例 → 资源详情缓存失效
	var req struct {
		CityId    int64 `json:"city_id"`
		RateFood  int   `json:"rate_food"`
		RateSteel int   `json:"rate_steel"`
		RateOil   int   `json:"rate_oil"`
		RateRare  int   `json:"rate_rare"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	h.cfgs()
	city := h.bodyCity(uid, req.CityId)
	vals := []int{req.RateFood, req.RateSteel, req.RateOil, req.RateRare}
	for _, v := range vals {
		if v < 1 || v > 100 {
			resp.ParamError(c, "开工率需在 1~100 之间")
			return
		}
	}
	h.refreshCity(uid, city)
	h.DB.Model(&model.EzfyCity{}).Where("id = ?", city.ID).Updates(map[string]interface{}{
		"rate_food": req.RateFood, "rate_steel": req.RateSteel,
		"rate_oil": req.RateOil, "rate_rare": req.RateRare,
	})
	resp.OK(c, gin.H{"msg": "开工率已调整"})
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
	tc := ezfyCfg.troop(req.TroopId)
	if tc == nil {
		resp.ParamError(c, "兵种不存在")
		return
	}
	// ★ 防御兵种(type=4 城防：碉堡/榴弹炮/反坦克炮/防空炮…) 固定阵地，
	//   前进/停止一律按「停止」落库，前端也不给选。
	if tc.Type == 4 {
		req.AtkMove = 0
		req.DefMove = 0
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

// corpsMemberCount 军团**实时**成员数（以 ezfy_corps_member 为准）
//
// ★ 不要用 ezfy_corps.member_count 这个计数字段：它是「加入 +1 / 退出 -1」维护的，
//
//	任何一次异常中断（例如加入成功但计数更新失败）都会让它永久漂移。
//	实测出现过「表里 3 人、字段写 2 人」，玩家看到的人数就是错的。
func (h *EzfyHandler) corpsMemberCount(corpsId int64) int64 {
	var n int64
	h.DB.Model(&model.EzfyCorpsMember{}).Where("corps_id = ?", corpsId).Count(&n)
	return n
}

// corpsMemberCountMap 批量取多个军团的成员数（避免列表页 N+1 查询）
func (h *EzfyHandler) corpsMemberCountMap(ids []int64) map[int64]int64 {
	out := map[int64]int64{}
	if len(ids) == 0 {
		return out
	}
	type row struct {
		CorpsId int64
		N       int64
	}
	var rows []row
	h.DB.Model(&model.EzfyCorpsMember{}).
		Select("corps_id, COUNT(*) AS n").Where("corps_id IN ?", ids).
		Group("corps_id").Scan(&rows)
	for _, r := range rows {
		out[r.CorpsId] = r.N
	}
	return out
}

func (h *EzfyHandler) CorpsList(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	profile := h.ensureProfile(uid)
	var corps []model.EzfyCorps
	h.DB.Order("id DESC").Find(&corps)
	ids := make([]int64, 0, len(corps))
	for _, cp := range corps {
		ids = append(ids, int64(cp.ID))
	}
	counts := h.corpsMemberCountMap(ids)

	// ★ 2026-10-03 性能：原实现 for 循环内逐军团查成员、逐成员 ensureProfile（N+1，
	//   数百次 RDS 往返 → 4s）。改为批量查成员 + 批量查 profile，全程 2 次往返。
	memberByCorps := map[int64][]model.EzfyCorpsMember{}
	needUIDs := map[uint]bool{}
	if len(ids) > 0 {
		var members []model.EzfyCorpsMember
		h.DB.Where("corps_id IN ?", ids).Find(&members)
		for _, m := range members {
			memberByCorps[int64(m.CorpsId)] = append(memberByCorps[int64(m.CorpsId)], m)
			needUIDs[m.UserId] = true
		}
	}
	for _, cp := range corps {
		needUIDs[cp.LeaderUserId] = true
	}
	profileByUID := map[uint]model.EzfyProfile{}
	if len(needUIDs) > 0 {
		uids := make([]uint, 0, len(needUIDs))
		for u := range needUIDs {
			uids = append(uids, u)
		}
		var ps []model.EzfyProfile
		h.DB.Where("user_id IN ?", uids).Find(&ps)
		for _, p := range ps {
			profileByUID[p.UserID] = p
		}
	}

	views := []gin.H{}
	for _, cp := range corps {
		score := 0
		for _, m := range memberByCorps[int64(cp.ID)] {
			score += profileByUID[m.UserId].Prestige
		}
		leaderName := "未知"
		if lp, ok := profileByUID[cp.LeaderUserId]; ok && lp.UserID == cp.LeaderUserId {
			leaderName = lp.Nickname
		}
		views = append(views, gin.H{"id": cp.ID, "name": cp.Name, "notice": cp.Notice,
			"member_count": counts[int64(cp.ID)], "leader": leaderName, "battle_score": score,
			"camp": profile.Camp,
			// ★ 2026-09-25 用户要求「军团积分」：军团总积分（原有字段不动，只补这一个）
			"points": cp.Points,
			// ★ 2026-09-30 入团审核开关（前端列表展示 [申请]/[申请待审] 文案用）
			"need_review": cp.NeedReview})
	}
	// ★ 我的军团也带上实时人数（前端「我的军团(N人)」直接用它）
	var myCorpsView interface{}
	var myMember model.EzfyCorpsMember
	if err := h.DB.Where("user_id = ?", uid).First(&myMember).Error; err == nil {
		var cp model.EzfyCorps
		if err := h.DB.First(&cp, myMember.CorpsId).Error; err == nil {
			myCorpsView = gin.H{
				"id": cp.ID, "name": cp.Name, "notice": cp.Notice,
				"leader_user_id": cp.LeaderUserId,
				"member_count":   counts[int64(cp.ID)],
				"points":         cp.Points,
				"need_review":    cp.NeedReview,
			}
		}
	}
	resp.OK(c, gin.H{"corps": views, "my_corps": myCorpsView})
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
	// ★ 2026-09-30 入口统一走 CorpsApply（含审核开关分支），此处复用同一逻辑。
	msg, data, errMsg := h.applyCorpsMember(uid, req.CorpsId)
	if errMsg != "" {
		resp.ParamError(c, errMsg)
		return
	}
	d := gin.H{"msg": msg}
	for k, v := range data {
		d[k] = v
	}
	resp.OK(c, d)
}

// applyCorpsMember 处理「申请入团」：open 军团直接入团，需审核军团写申请待军团长审批。
//
// ★ 2026-09-30 用户要求「进军团需要审核」。返回：
//   - errMsg != ""：校验失败信息（调用方回 resp.ParamError）
//   - 否则 msg/data：成功信息 + 附带 need_review 标记
func (h *EzfyHandler) applyCorpsMember(uid uint, corpsId uint) (string, gin.H, string) {
	var exist model.EzfyCorpsMember
	if err := h.DB.Where("user_id = ?", uid).First(&exist).Error; err == nil {
		return "", nil, "你已在军团中"
	}
	var cp model.EzfyCorps
	if err := h.DB.First(&cp, corpsId).Error; err != nil {
		return "", nil, "军团不存在"
	}
	// 联络中心: 1 级才能加入联盟, 且受人数上限限制
	if h.liaisonLevel(uid) < 1 {
		return "", nil, "需要 1 级联络中心才能加入联盟"
	}
	// 需审核军团：写申请（幂等），不直接入团
	if cp.NeedReview == 1 {
		var pending model.EzfyCorpsApply
		if err := h.DB.Where("corps_id = ? AND user_id = ? AND status = 0", cp.ID, uid).First(&pending).Error; err == nil {
			return "", nil, "已提交申请, 等待军团长审核"
		}
		var memberCount int64
		h.DB.Model(&model.EzfyCorpsMember{}).Where("corps_id = ?", cp.ID).Count(&memberCount)
		if cap := h.corpsMemberCap(cp.ID); int(memberCount) >= cap {
			return "", nil, "该联盟人数已满(" + strconv.Itoa(int(memberCount)) + "/" + strconv.Itoa(cap) + ")"
		}
		h.DB.Create(&model.EzfyCorpsApply{CorpsId: cp.ID, UserId: uid, Status: 0})
		return "申请已提交, 等待军团长审核", gin.H{"need_review": 1}, ""
	}
	// 无需审核：直接入团（原逻辑）
	var memberCount int64
	h.DB.Model(&model.EzfyCorpsMember{}).Where("corps_id = ?", cp.ID).Count(&memberCount)
	if cap := h.corpsMemberCap(cp.ID); int(memberCount) >= cap {
		return "", nil, "该联盟人数已满(" + strconv.Itoa(int(memberCount)) + "/" + strconv.Itoa(cap) + ")"
	}
	h.DB.Create(&model.EzfyCorpsMember{CorpsId: cp.ID, UserId: uid, IsLeader: 0, Title: "成员"})
	h.DB.Model(&model.EzfyCorps{}).Where("id = ?", cp.ID).Update("member_count", cp.MemberCount+1)
	return "加入军团成功", gin.H{"need_review": 0}, ""
}

// CorpsApply POST /corps/apply {corps_id} —— 申请入团
//
// ★ 2026-09-30 用户要求「进军团需要审核」：open 直接入团；开启审核的军团落申请待团长审批。
func (h *EzfyHandler) CorpsApply(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		CorpsId uint `json:"corps_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	msg, data, errMsg := h.applyCorpsMember(uid, req.CorpsId)
	if errMsg != "" {
		resp.ParamError(c, errMsg)
		return
	}
	d := gin.H{"msg": msg}
	for k, v := range data {
		d[k] = v
	}
	resp.OK(c, d)
}

// CorpsApplyList GET /corps/apply —— 军团长查看本人军团的待审入团申请
func (h *EzfyHandler) CorpsApplyList(c *gin.Context) {
	uid := middleware.GetUID(c)
	var cp model.EzfyCorps
	if err := h.DB.Where("leader_user_id = ?", uid).First(&cp).Error; err != nil {
		resp.OK(c, gin.H{"applies": []gin.H{}})
		return
	}
	var list []model.EzfyCorpsApply
	h.DB.Where("corps_id = ? AND status = 0", cp.ID).Order("id ASC").Find(&list)
	views := []gin.H{}
	for _, a := range list {
		p := h.ensureProfile(a.UserId)
		views = append(views, gin.H{"apply_id": a.ID, "user_id": a.UserId,
			"name": p.Nickname, "created_at": a.CreatedAt})
	}
	resp.OK(c, gin.H{"applies": views, "need_review": cp.NeedReview})
}

// CorpsApplyHandle POST /corps/apply/handle {apply_id, op} —— 军团长通过/拒绝入团申请（op 1=通过 2=拒绝）
func (h *EzfyHandler) CorpsApplyHandle(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		ApplyId uint `json:"apply_id"`
		Op      int  `json:"op"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	var a model.EzfyCorpsApply
	if err := h.DB.First(&a, req.ApplyId).Error; err != nil {
		resp.ParamError(c, "申请不存在")
		return
	}
	var cp model.EzfyCorps
	if err := h.DB.Where("id = ? AND leader_user_id = ?", a.CorpsId, uid).First(&cp).Error; err != nil {
		resp.ParamError(c, "只有军团长才能处理入团申请")
		return
	}
	if a.Status != 0 {
		resp.ParamError(c, "该申请已处理")
		return
	}
	h.DB.Model(&model.EzfyCorpsApply{}).Where("id = ?", a.ID).Update("status", req.Op)
	if req.Op == 2 {
		resp.OK(c, gin.H{"msg": "已拒绝该申请"})
		return
	}
	// 通过（op != 2 视为 1）：幂等建成员 + 人数+1；可能已在其它申请通过后加入
	var mb model.EzfyCorpsMember
	if err := h.DB.Where("user_id = ?", a.UserId).First(&mb).Error; err != nil {
		var memberCount int64
		h.DB.Model(&model.EzfyCorpsMember{}).Where("corps_id = ?", cp.ID).Count(&memberCount)
		if cap := h.corpsMemberCap(cp.ID); int(memberCount) >= cap {
			resp.ParamError(c, "该联盟人数已满("+strconv.Itoa(int(memberCount))+"/"+strconv.Itoa(cap)+")")
			return
		}
		h.DB.Create(&model.EzfyCorpsMember{CorpsId: cp.ID, UserId: a.UserId, IsLeader: 0, Title: "成员"})
		h.DB.Model(&model.EzfyCorps{}).Where("id = ?", cp.ID).
			Update("member_count", cp.MemberCount+1)
	}
	resp.OK(c, gin.H{"msg": "已通过申请, 该玩家已入团"})
}

// CorpsNeedReview POST /corps/need-review {need_review} —— 军团长设置入团审核开关
func (h *EzfyHandler) CorpsNeedReview(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		NeedReview int `json:"need_review"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	var cp model.EzfyCorps
	if err := h.DB.Where("leader_user_id = ?", uid).First(&cp).Error; err != nil {
		resp.ParamError(c, "只有军团长才能设置入团审核")
		return
	}
	nr := 0
	if req.NeedReview == 1 {
		nr = 1
	}
	h.DB.Model(&model.EzfyCorps{}).Where("id = ?", cp.ID).Update("need_review", nr)
	msg := "已设为无需审核, 新玩家可直接加入"
	if nr == 1 {
		msg = "已开启入团审核, 新玩家申请后需你在[军团信息]审核"
	}
	resp.OK(c, gin.H{"msg": msg, "need_review": nr})
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
		views = append(views, gin.H{"id": ch.ID, "user_id": ch.UserId, "user_name": ch.UserName,
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
	// ★ 第九轮：二战聊天敏感词
	if filtered, blocked := ezfyFilterChat(content); blocked {
		resp.Forbidden(c, "你的发言包含敏感词，请修改后再发")
		return
	} else {
		content = filtered
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
	// ★ 管理端「宣战功能」关掉时不需要宣战（掠夺/征服直接可打）→ 直接拒绝宣战请求，
	//   免得老缓存的前端还能点出 [宣战]，在系统频道刷出没有意义的播报。
	if !ezfyWarRequireOn() {
		resp.ParamError(c, "当前不需要宣战，可直接掠夺/征服")
		return
	}
	// ★ 用户规则：「同盟玩家不能宣战」。
	//   同盟 = 同一个军团（与地图上「运输/增援」的判定口径完全一致，见 ezfy_order.go）。
	//   两边任一方没军团都不算同盟。
	if h.sameCorps(uid, req.TargetUserId) {
		resp.ParamError(c, "同盟成员之间不能宣战")
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

	// ★ 用户要求「宣战也要如系统消息」：宣战方与被宣战方各发一条系统消息（EzfyNotice），
	//   与战争管理后台的提示口径保持一致（ezfy_admin_war.go）。
	atkName, _ := h.ezfyPlayerName(uid)
	if atkName == "" {
		atkName = h.ezfyProfileName(uid)
	}
	if atkName == "" {
		atkName = fmt.Sprintf("玩家%d", uid)
	}
	defName, _ := h.ezfyPlayerName(req.TargetUserId)
	if defName == "" {
		defName = h.ezfyProfileName(req.TargetUserId)
	}
	if defName == "" {
		defName = fmt.Sprintf("玩家%d", req.TargetUserId)
	}
	defTip := fmt.Sprintf("【宣战】%s 向你宣战，%d 小时后生效，生效后 %d 小时内可互相掠夺/征服。",
		atkName, ezfyWarDelayHours, ezfyWarDurationHours)
	h.DB.Create(&model.EzfyNotice{UserId: req.TargetUserId, Title: "宣战", Content: defTip})
	// ★ 2026-10-05 用户要求：宣战后自动给被宣战方发一封站内信（邮件，来源 = 宣战方），
	//   与上面系统消息同口径；对方在「邮件」页能看到这条宣战消息。
	h.DB.Create(&model.PrivateMessage{SenderID: uid, ReceiverID: req.TargetUserId, Content: defTip})
	atkTip := fmt.Sprintf("【宣战】你已向 %s 宣战，%d 小时后生效，生效后 %d 小时内可互相掠夺/征服。",
		defName, ezfyWarDelayHours, ezfyWarDurationHours)
	h.DB.Create(&model.EzfyNotice{UserId: uid, Title: "宣战", Content: atkTip})

	// ★ 用户要求「首页世界聊天那块，谁向谁宣战也播报展示」→ 往**系统频道**写一条全服可见的播报。
	//   ezfySysChat 写的是 channel=4 / talk_type=0，首页「世界聊天」预览与聊天页的系统频道都会显示，
	//   所以这里不需要另开接口，玩家端不用改。
	//   ⚠️ 注意：宣战本身没有次数上限（只挡了「对同一个人重复宣战」），
	//   如果将来发现有人刷屏，可以在 ezfySysChat 外面加个节流/上限。
	h.ezfySysChat("【宣战】%s 向 %s 宣战了，%d 小时后生效！", atkName, defName, ezfyWarDelayHours)

	resp.OK(c, gin.H{"msg": fmt.Sprintf("宣战成功, %d小时后生效, 生效后%d小时内可互相掠夺/征服", ezfyWarDelayHours, ezfyWarDurationHours)})
}

// ezfyPlayerName 取玩家在二战里的展示名（优先 profile.nickname）
func (h *EzfyHandler) ezfyPlayerName(uid uint) (string, error) {
	var p model.EzfyProfile
	if err := h.DB.Where("user_id = ?", uid).First(&p).Error; err != nil {
		return "", err
	}
	return p.Nickname, nil
}

func (h *EzfyHandler) WarStatus(c *gin.Context) {
	// ★ 必须先 cfgs()：war_require 读的是配置缓存，不加载就只会拿到默认值
	//   （踩过：这里漏了 cfgs()，开关明明是开，接口却回 false）
	h.cfgs()
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
	// ★ war_require：管理端「宣战功能」开关的当前值。玩家端用它决定
	//   掠夺/征服按钮要不要「需先宣战」那套限制（false = 直接可点）。
	//   注意这里**不**把 status 伪造成 2 —— 那样会让「同盟城市」的 运输/增援 按钮
	//   被误判成交战中而消失，所以开关单独下发，由前端分别处理。
	//
	// ★ 2026-09-25 用户要求「军团宣战生效期间成员之间可直接打」：新增两个字段（原有字段不变，前端已依赖）。
	//   at_war    = 个人交战中 || 宣战开关关闭 || 军团交战生效；前端据此放开掠夺/征服按钮。
	//   corps_war = {active, corps_name(对方军团名), text(军团交战期文案)}。
	cw := h.corpsActiveWarBetween(uid, tid)
	atWar := status == 2 || !ezfyWarRequireOn() || cw != nil
	corpsWar := gin.H{"active": false, "corps_name": "", "text": ""}
	if cw != nil {
		now := time.Now().UnixMilli()
		hs := (cw.ExpireTime - now + 3599999) / 3600000
		if hs < 0 {
			hs = 0
		}
		oppName := cw.DefCorpsName
		if cw.DefCorpsId == h.corpsOfUser(uid) {
			oppName = cw.AtkCorpsName
		}
		corpsWar = gin.H{"active": true, "corps_name": oppName,
			"text": fmt.Sprintf("军团交战期(剩余约%d小时)", hs)}
	}
	resp.OK(c, gin.H{"status": status, "text": text, "war_require": ezfyWarRequireOn(),
		"at_war": atWar, "corps_war": corpsWar})
}

// ============ 排行榜 ============

// ★ 2026-10-02 1核1G 线上 CPU 100% 优化：战力榜每次请求都全表扫描 ezfy_city /
//   ezfy_city_troop / ezfy_city_building / ezfy_user_tech 四张表，而前端首页每 30 秒
//   轮询 /view 时又连带调一次 /rank → 单核 MySQL 被查询洪水打满，所有请求排队变慢
//   （同一个 DELETE 语句从 0.4ms 恶化到 30ms+）。这些底层数据（声望/战力/军团/军衔表）
//   变化缓慢，做 30 秒内存缓存；过期后第一个请求重算，其余并发请求复用。
//   玩家本人「军衔/晋升」依赖请求 uid，每次现算（rankMine），不缓存。
type ezfyRankHeavy struct {
	prestige []gin.H
	troops   []gin.H
	corps    []gin.H
	ranks    []gin.H
}

var (
	ezfyRankHeavyMu   sync.Mutex
	ezfyRankHeavyCond = sync.NewCond(&ezfyRankHeavyMu)
	ezfyRankHeavyAt   int64
	ezfyRankHeavyData *ezfyRankHeavy
	ezfyRankHeavyBusy bool
)

// ezfyRankHeavyGet 取排行缓存。返回 (data, true)=直接复用；返回 (nil, false)=
// 调用方成为**单飞 owner**，负责重算并调 ezfyRankHeavyFinish 交账。
// ★ 单飞：重启后所有玩家首页同时命中缓存过期 → 只有第一个真正全表扫描，
//   其余请求 Cond.Wait 等它算完复用，避免 N 个并发全表扫描把 1 核打爆。
func ezfyRankHeavyGet() (*ezfyRankHeavy, bool) {
	ezfyRankHeavyMu.Lock()
	defer ezfyRankHeavyMu.Unlock()
	for {
		now := time.Now().UnixMilli()
		if ezfyRankHeavyData != nil && now-ezfyRankHeavyAt < 30000 {
			return ezfyRankHeavyData, true
		}
		if !ezfyRankHeavyBusy {
			ezfyRankHeavyBusy = true
			return nil, false
		}
		ezfyRankHeavyCond.Wait()
	}
}

func ezfyRankHeavyFinish(pre, tr, cr, rk []gin.H) *ezfyRankHeavy {
	hv := &ezfyRankHeavy{prestige: pre, troops: tr, corps: cr, ranks: rk}
	ezfyRankHeavyMu.Lock()
	ezfyRankHeavyData = hv
	ezfyRankHeavyAt = time.Now().UnixMilli()
	ezfyRankHeavyBusy = false
	ezfyRankHeavyMu.Unlock()
	ezfyRankHeavyCond.Broadcast()
	return hv
}

// rankMine 玩家本人军衔/晋升信息（依赖请求 uid，每次现算，不参与 30s 缓存）
func (h *EzfyHandler) rankMine(uid uint) gin.H {
	me := h.ensureProfile(uid)
	var myCities int64
	h.DB.Model(&model.EzfyCity{}).Where("user_id = ?", uid).Count(&myCities)
	meLv := ezfyProfileRank(&me)
	mine := gin.H{
		"rank_id": ezfyRankAt(meLv).ID, "rank_name": ezfyRankNameAt(meLv), "rank_post": ezfyRankPostAt(meLv),
		"prestige": me.Prestige, "city_max": ezfyRankCityMaxAt(meLv), "city_count": myCities,
		"rank_level": meLv,
	}
	// ★ 下一级晋升信息（声望门槛 + 所需宝物 + 背包现有量），供前端[晋升]按钮展示与校验
	if meLv < len(ezfyCfg.rankList()) {
		nr := ezfyCfg.rankList()[meLv]
		reqs := ezfyRankTreasureReqs(nr.ID)
		// ★ 2026-10-03 性能：原实现每件宝物一条 COUNT（跨 WAN RDS），改成一次 GROUP BY 批量统计。
		cfgIDs := make([]int, 0, len(reqs))
		for _, r := range reqs {
			if cfg := ezfyEquipCfgByName(r.Name); cfg != nil {
				cfgIDs = append(cfgIDs, cfg.ID)
			}
		}
		owned := map[int]int64{}
		if len(cfgIDs) > 0 {
			var rows []struct {
				CfgID int
				N     int64
			}
			h.DB.Model(&model.EzfyEquipment{}).
				Select("cfg_id, COUNT(*) AS n").
				Where("user_id = ? AND officer_id = 0 AND cfg_id IN ?", uid, cfgIDs).
				Group("cfg_id").Scan(&rows)
			for _, r := range rows {
				owned[r.CfgID] = r.N
			}
		}
		tr := []gin.H{}
		for _, r := range reqs {
			cfg := ezfyEquipCfgByName(r.Name)
			have := int64(0)
			if cfg != nil {
				have = owned[cfg.ID]
			}
			tr = append(tr, gin.H{"name": r.Name, "count": r.Count, "have": have})
		}
		mine["next"] = gin.H{"id": nr.ID, "name": nr.Name, "post": nr.Post,
			"need": nr.NeedPrestige, "treasures": tr}
	}
	return mine
}

func (h *EzfyHandler) Rank(c *gin.Context) {
	h.cfgs()
	uid := middleware.GetUID(c)
	// ★ 30s 缓存命中：跳过 4 张全表扫描，直接复用底层数据（mine 每次现算）
	if hv, ok := ezfyRankHeavyGet(); ok {
		resp.OK(c, gin.H{"prestige": hv.prestige, "troops": hv.troops, "corps": hv.corps,
			"ranks": hv.ranks, "mine": h.rankMine(uid)})
		return
	}
	// 单飞 owner 路径：即使重算过程异常退出也要交账，否则并发等待的请求会永久阻塞
	defer func() {
		if r := recover(); r != nil {
			ezfyRankHeavyMu.Lock()
			ezfyRankHeavyBusy = false
			ezfyRankHeavyMu.Unlock()
			ezfyRankHeavyCond.Broadcast()
			panic(r)
		}
	}()
	// 声望榜
	var profiles []model.EzfyProfile
	h.DB.Order("prestige DESC").Limit(20).Find(&profiles)
	prestigeRank := []gin.H{}
	for i, p := range profiles {
		prestigeRank = append(prestigeRank, gin.H{"rank": i + 1, "name": p.Nickname, "user_id": p.UserID,
			"prestige": p.Prestige, "rank_name": ezfyRankNameAt(ezfyProfileRank(&p))})
	}
	// 战力榜（★ 2026-10-02 兵力榜 → 战力榜，用户要求柔和科技/建筑/兵种，避免纯兵力碾压吓到新人）
	//   战力 = 科技战力(用户级, 等级全城共用) + 建筑战力 + 兵种战力；
	//   建筑/兵种按玩家「最好城市」(综合得分最高的城)计算，展示城市名也用该城。
	//   权重读 ezfy_cfg_limit（管理端「二战系统配置」可调）：
	//   科技每级 power_tech_per_level、每项已研究 power_tech_per_tech、
	//   建筑每级 power_build_per_level、每兵种类型 power_troop_type、
	//   兵种数量按 count^power_troop_pow × 质量/100（软化新老差距）。
	lim := ezfyCfg.limit
	pow := lim.PowerTroopPow
	if pow <= 0 {
		pow = 0.8
	}
	// 总量压缩幂次：最终战力 = 原始总和^此值（默认 0.5）。幂次<1 时只缩大数、小数几乎不缩，
	// 且严格单调递增 → 排名顺序不变（不像除法那样把低战力压成小数）。
	cp := lim.PowerCompressPow
	if cp <= 0 {
		cp = 0.5
	}
	// 兵种质量 = 生命+防御+攻击(防/陆/空)+速度+射程, ÷100 → 每单位战力基数
	troopQuality := func(id int) float64 {
		if t := ezfyCfg.troop(id); t != nil {
			return float64(t.Health+t.Defence+t.AtkDef+t.AtkGround+t.AtkAir+t.Speed+t.AttackRange) / 100
		}
		return 10
	}
	// 城市 → 归属玩家 / 城市名
	var cities []model.EzfyCity
	h.DB.Find(&cities)
	cityUser := map[int64]int64{}
	cityName := map[int64]string{}
	for _, c := range cities {
		cityUser[int64(c.ID)] = int64(c.UserID)
		cityName[int64(c.ID)] = c.Name
	}
	// 兵种战力按「玩家所有城市」聚合（含城防 type=4）：Σ count^pow × 质量 + 兵种类型数 × 系数。
	// ★ 2026-10-02 用户反馈：只算最好城市会导致榜首建筑明细=2 的失真，量应取全城合计，
	//   「最好城市」只用来选展示主城名。
	var troops []model.EzfyCityTroop
	h.DB.Find(&troops)
	userTroopScore := map[int64]float64{}
	userTroopTypes := map[int64]map[int]bool{}
	for _, t := range troops {
		if t.Count <= 0 {
			continue
		}
		uid := cityUser[t.CityId]
		if uid == 0 {
			continue
		}
		if userTroopTypes[uid] == nil {
			userTroopTypes[uid] = map[int]bool{}
		}
		if !userTroopTypes[uid][t.TroopId] {
			userTroopTypes[uid][t.TroopId] = true
			userTroopScore[uid] += float64(lim.PowerTroopType)
		}
		userTroopScore[uid] += math.Pow(float64(t.Count), pow) * troopQuality(t.TroopId)
	}
	// 建筑战力按「玩家所有城市」聚合；cityBuildSum 用于选展示主城（建筑等级最高的城）
	var builds []model.EzfyCityBuilding
	h.DB.Find(&builds)
	userBuildScore := map[int64]float64{}
	cityBuildSum := map[int64]int{}
	for _, b := range builds {
		uid := cityUser[b.CityId]
		if uid == 0 {
			continue
		}
		userBuildScore[uid] += float64(b.Level * lim.PowerBuildPerLevel)
		cityBuildSum[b.CityId] += b.Level
	}
	// 用户级科技战力：Σ(等级×每级分) + 每项已研究 + 每项分
	var userTechs []model.EzfyUserTech
	h.DB.Find(&userTechs)
	userTechScore := map[int64]float64{}
	for _, ut := range userTechs {
		if ut.Level <= 0 {
			continue
		}
		userTechScore[int64(ut.UserId)] += float64(ut.Level*lim.PowerTechPerLevel) + float64(lim.PowerTechPerTech)
	}
	// 展示主城：建筑等级最高的城；没有建筑但有兵的玩家兜底取任一有兵城
	bestCity := map[int64]int64{}
	bestLv := map[int64]int{}
	for cid, lv := range cityBuildSum {
		uid := cityUser[cid]
		if lv > bestLv[uid] {
			bestLv[uid] = lv
			bestCity[uid] = cid
		}
	}
	for _, t := range troops {
		if t.Count <= 0 {
			continue
		}
		uid := cityUser[t.CityId]
		if uid == 0 {
			continue
		}
		if _, ok := bestCity[uid]; !ok {
			bestCity[uid] = t.CityId
		}
	}
	type pw struct {
		uid   int64
		power float64
		tech  float64
		build float64
		troop float64
	}
	all := map[int64]*pw{}
	uidSet := map[int64]bool{}
	for uid := range userTechScore {
		uidSet[uid] = true
	}
	for uid := range userBuildScore {
		uidSet[uid] = true
	}
	for uid := range userTroopScore {
		uidSet[uid] = true
	}
	for uid := range uidSet {
		techRaw, buildRaw, troopRaw := userTechScore[uid], userBuildScore[uid], userTroopScore[uid]
		raw := techRaw + buildRaw + troopRaw
		e := &pw{uid: uid, power: math.Pow(raw, cp)}
		if raw > 0 {
			// 明细各自独立幂压缩后按比例归一化：总和 = power，且小分量不会被线性占比压成 0。
			// （power 仍由 raw^cp 决定 → 排序与原始总和严格一致，不受明细拆分影响）
			t2 := math.Pow(techRaw, cp)
			b2 := math.Pow(buildRaw, cp)
			o2 := math.Pow(troopRaw, cp)
			sum2 := t2 + b2 + o2
			if sum2 > 0 {
				e.tech = e.power * t2 / sum2
				e.build = e.power * b2 / sum2
				e.troop = e.power * o2 / sum2
			}
		}
		all[uid] = e
	}
	arr := make([]*pw, 0, len(all))
	for _, v := range all {
		arr = append(arr, v)
	}
	sort.Slice(arr, func(i, j int) bool { return arr[i].power > arr[j].power })
	// ★ 2026-10-03 性能：原实现逐条 ensureProfile（top20 = 20 次 RDS 往返），
	//   改成一次 WHERE user_id IN 批量查档案（照搬 CorpsList 的批量模式）。
	topUIDs := make([]uint, 0, 20)
	for i, e := range arr {
		if i >= 20 {
			break
		}
		topUIDs = append(topUIDs, uint(e.uid))
	}
	profileByUID := map[uint]model.EzfyProfile{}
	if len(topUIDs) > 0 {
		var ps []model.EzfyProfile
		h.DB.Where("user_id IN ?", topUIDs).Find(&ps)
		for _, p := range ps {
			profileByUID[p.UserID] = p
		}
	}
	troopRank := []gin.H{}
	for i, e := range arr {
		if i >= 20 {
			break
		}
		p := profileByUID[uint(e.uid)]
		troopRank = append(troopRank, gin.H{"rank": i + 1, "city_name": cityName[bestCity[e.uid]],
			"role_name": p.Nickname, "user_id": e.uid,
			"power": int64(e.power), "tech_power": int64(e.tech),
			"build_power": int64(e.build), "troop_power": int64(e.troop)})
	}
	// 军团榜
	var corps []model.EzfyCorps
	// 排序仍按计数字段（只影响顺序），展示用实时人数
	h.DB.Order("member_count DESC").Limit(20).Find(&corps)
	rankIds := make([]int64, 0, len(corps))
	for _, cp := range corps {
		rankIds = append(rankIds, int64(cp.ID))
	}
	rankCounts := h.corpsMemberCountMap(rankIds)
	// ★ 2026-10-03 性能：原实现逐军团查成员、逐成员 ensureProfile（N+1，数十次 RDS 往返），
	//   照搬 CorpsList 的批量模式：一次查全部成员 + 一次批量查档案，固定 2 次往返。
	memberByCorps := map[int64][]model.EzfyCorpsMember{}
	needUIDs := map[uint]bool{}
	if len(rankIds) > 0 {
		var members []model.EzfyCorpsMember
		h.DB.Where("corps_id IN ?", rankIds).Find(&members)
		for _, m := range members {
			memberByCorps[int64(m.CorpsId)] = append(memberByCorps[int64(m.CorpsId)], m)
			needUIDs[m.UserId] = true
		}
	}
	for _, cp := range corps {
		needUIDs[cp.LeaderUserId] = true
	}
	corpsProfileByUID := map[uint]model.EzfyProfile{}
	if len(needUIDs) > 0 {
		uids := make([]uint, 0, len(needUIDs))
		for u := range needUIDs {
			uids = append(uids, u)
		}
		var ps []model.EzfyProfile
		h.DB.Where("user_id IN ?", uids).Find(&ps)
		for _, p := range ps {
			corpsProfileByUID[p.UserID] = p
		}
	}
	corpsRank := []gin.H{}
	for i, cp := range corps {
		score := 0
		for _, m := range memberByCorps[int64(cp.ID)] {
			score += corpsProfileByUID[m.UserId].Prestige
		}
		corpsRank = append(corpsRank, gin.H{"rank": i + 1, "name": cp.Name,
			"member_count": rankCounts[int64(cp.ID)], "battle_score": score})
	}
	// 军衔表（★ 含「可建城数」一列，与 model.EzfyCfgRank 一致；2026-09-28 新增「宝物」列）
	ranks := []gin.H{}
	for _, r := range ezfyCfg.rankList() {
		ranks = append(ranks, gin.H{
			"id": r.ID, "name": r.Name, "post": r.Post,
			"need": r.NeedPrestige, "city_max": r.CityMax,
			"treasures": ezfyTreasureListText(ezfyRankTreasureReqs(r.ID)),
		})
	}
	// ★ 重计算完成：写入 30s 缓存（下一个请求直接命中），响应组装与缓存命中路径完全一致
	hv := ezfyRankHeavyFinish(prestigeRank, troopRank, corpsRank, ranks)
	resp.OK(c, gin.H{"prestige": hv.prestige, "troops": hv.troops, "corps": hv.corps,
		"ranks": hv.ranks, "mine": h.rankMine(uid)})
}

// ============ 商城/背包 ============

// ezfyItemCategory 道具的商城分类（管理端没填 category 时按类型自动归类）
// ezfyIsDiamondItem 是否「钻石道具」。
//
// ★ 不能只看 price_diamond > 0：用户要求集结令走钻石渠道但**默认 0 钻石**（先免费放开），
// 这时价格是 0，靠价格判不出来。所以再加一条：管理端把 category 填成「钻石道具」也算。
func ezfyIsDiamondItem(it *model.EzfyCfgItem) bool {
	// ★ 用户规则：管理端把分类配成「钻石道具 / 黄金道具」时**锁定货币**，
	//   钻石道具只能用钻石买，黄金道具只能用黄金买。
	switch strings.TrimSpace(it.Category) {
	case "钻石道具":
		return true
	case "黄金道具":
		return false
	}
	return it.PriceDiamond > 0
}

// ezfyItemPayCurrency 该道具锁定的支付货币："" = 不锁（自动/双渠道），"gold" / "diamond" = 锁定
func ezfyItemPayCurrency(it *model.EzfyCfgItem) string {
	switch strings.TrimSpace(it.Category) {
	case "钻石道具":
		return "diamond"
	case "黄金道具":
		return "gold"
	}
	return ""
}

// ezfyUnlimitedStock 库存为负数表示**无限**（用户规则：库存 -1 = 可以任意购买）
func ezfyUnlimitedStock(stock int) bool { return stock < 0 }

func ezfyItemCategory(it *model.EzfyCfgItem) string {
	if c := strings.TrimSpace(it.Category); c != "" {
		return c
	}
	if it.PriceDiamond > 0 {
		return "钻石道具"
	}
	switch it.ItemType {
	case 1, 2, 27, 28, 29, 30:
		return "资源道具"
	case 3, 4, 5:
		return "加速道具"
	case 6:
		return "建筑图纸"
	case 7, 8:
		return "增益道具"
	case 9, 10, 11, 12:
		return "军官道具"
	// ★ 19 = 星级徽章（用户要求放到「军官道具」分类下）
	case 19:
		return "军官道具"
	// ★ 21 = 军官改名卡（军官道具）
	case 21:
		return "军官道具"
	// ★ 20 = 信号弹（计谋消耗品）
	case 20:
		return "计谋道具"
	case 13, 14:
		return "身份道具"
	case 15:
		return "出征道具"
	case 16, 17, 18:
		return "迁城道具"
	}
	return "其他"
}

// Mall GET /games/ezfy/mall —— 商城道具（★ 第九轮：带分类与钻石价，前端做分类页签 + 分页）
func (h *EzfyHandler) Mall(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	var items []model.EzfyCfgItem
	h.DB.Order("id ASC").Find(&items)
	views := make([]gin.H, 0, len(items))
	cats := []string{}
	seen := map[string]bool{}
	for i := range items {
		it := items[i]
		// ★ 2026-09-27 为爱发电卡：管理端发放专用，不进商城（避免裸奔成 0 价可购）
		if isLoveCardItem(it.ItemType) {
			continue
		}
		cat := ezfyItemCategory(&it)
		if !seen[cat] {
			seen[cat] = true
			cats = append(cats, cat)
		}
		payCur := ezfyItemPayCurrency(&it)
		views = append(views, gin.H{
			"id": it.ID, "name": it.Name, "item_type": it.ItemType, "param1": it.Param1,
			"price_gold": it.PriceGold, "price_diamond": it.PriceDiamond,
			"icon": it.Icon, "description": it.Description, "stock": it.Stock,
			"category": cat, "is_diamond": ezfyIsDiamondItem(&it),
			// ★ 锁定的支付货币（"gold"/"diamond"/""）；锁定后前端只出对应那一种价格
			"pay_currency": payCur,
			// ★ 双渠道：黄金价和钻石价都 > 0 且**没有锁定货币**时，玩家可以任选一种支付
			"dual_pay": payCur == "" && it.PriceGold > 0 && it.PriceDiamond > 0,
			// ★ 库存 -1 = 无限可购（前端显示「无限」）
			"unlimited": ezfyUnlimitedStock(it.Stock),
		})
	}
	prof := h.ensureProfile(uid)
	resp.OK(c, gin.H{
		"items": views, "categories": cats, "diamond": prof.Diamond,
		// ★ 单次购买数量上限（管理端「建筑上限配置」页维护，线上现值 99）
		//   前端输入框 max / 前端校验都用它，避免和写死的 99 打架。
		"buy_max": ezfyMallBuyMaxCfg(),
	})
}

func (h *EzfyHandler) Buy(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		CityId int64 `json:"city_id"`
		CfgId  int   `json:"cfg_id"`
		Count  int   `json:"count"`
		// ★ 双渠道道具的支付方式："gold" / "diamond"；留空按默认（有钻石价则钻石优先）
		PayWith string `json:"pay_with"`
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
	// ★ 单次购买数量上限（管理端可配置）
	//   上限读 ezfy_cfg_limit.mall_buy_max（管理端「建筑上限配置」页维护），线上现值 99。
	if mx := ezfyMallBuyMaxCfg(); req.Count > mx {
		resp.ParamError(c, fmt.Sprintf("单次最多购买 %d 个", mx))
		return
	}
	cfg := ezfyCfg.item(req.CfgId)
	if cfg == nil {
		resp.ParamError(c, "道具不存在")
		return
	}
	// ★ 库存校验（管理端在「数据管理 → 道具配置」维护，默认 100）
	//   从库里读最新值，不用配置缓存 —— 管理端改完立即生效，不用等缓存重载。
	//   ★ 用户规则：库存 **-1 = 无限**，可以任意购买（不做数量校验、也不扣库存）。
	var live model.EzfyCfgItem
	stock := 0
	unlimited := false
	if err := h.DB.First(&live, req.CfgId).Error; err == nil {
		stock = live.Stock
		unlimited = ezfyUnlimitedStock(stock)
		if !unlimited && stock < req.Count {
			if stock <= 0 {
				resp.ParamError(c, fmt.Sprintf("「%s」已售罄", cfg.Name))
			} else {
				resp.ParamError(c, fmt.Sprintf("「%s」库存不足，只剩 %d 个", cfg.Name, stock))
			}
			return
		}
	}
	// ★ 支付渠道判定（第十二轮：支持「黄金 / 钻石」双渠道道具）
	//
	//	用户规则：「迁城计划 是道具 可以用黄金 和 钻石 购买 单独的 但是功能是一样的」
	//
	//	三种情况：
	//	  ① pay_with = "gold"    → 强制走黄金（需 price_gold > 0）
	//	  ② pay_with = "diamond" → 强制走钻石（需 price_diamond > 0）
	//	  ③ pay_with 留空        → 老行为：有钻石价就走钻石（category=钻石道具 的免费道具也是这条）；
	//                          只有黄金价则走黄金
	//
	//	这样既保住了老道具（集结令等）的既有语义，又让迁城道具能两种钱都买。
	useDiamond := ezfyIsDiamondItem(cfg)
	// ★ 用户规则：分类配成「钻石道具 / 黄金道具」时锁定货币，两种钱不能混用
	payCur := ezfyItemPayCurrency(cfg)
	// ★ 用户反馈「标价 0 钻石的集结令买不了，提示『不支持用钻石购买』」：
	//   价格为 0 且**分类已经锁定该货币**时，0 是「免费发放」而不是「不支持」，
	//   不能再拿 price<=0 去拦（集结令就是 Category=钻石道具 + 0 钻石的免费道具）。
	//   只有「没锁定该货币、价格又 <= 0」才是真的不支持这种钱。
	freeByLock := func(cur string) bool { return payCur == cur }
	if req.PayWith == "gold" {
		if payCur == "diamond" {
			resp.ParamError(c, fmt.Sprintf("「%s」是钻石道具，只能用钻石购买", cfg.Name))
			return
		}
		if cfg.PriceGold <= 0 && !freeByLock("gold") {
			resp.ParamError(c, fmt.Sprintf("「%s」不支持用黄金购买", cfg.Name))
			return
		}
		useDiamond = false
	} else if req.PayWith == "diamond" {
		if payCur == "gold" {
			resp.ParamError(c, fmt.Sprintf("「%s」是黄金道具，只能用黄金购买", cfg.Name))
			return
		}
		if cfg.PriceDiamond <= 0 && !freeByLock("diamond") {
			resp.ParamError(c, fmt.Sprintf("「%s」不支持用钻石购买", cfg.Name))
			return
		}
		useDiamond = true
	}
	// 锁定货币时，忽略前端传来的相反渠道
	if payCur == "diamond" {
		useDiamond = true
	} else if payCur == "gold" {
		useDiamond = false
	}
	if useDiamond {
		cost := cfg.PriceDiamond * int64(req.Count)
		prof := h.ensureProfile(uid)
		if prof.Diamond < cost {
			resp.ParamError(c, fmt.Sprintf("钻石不足: 需要%d钻石, 当前余额%d（也可改用黄金购买）", cost, prof.Diamond))
			return
		}
		if cost > 0 {
			if err := h.DB.Model(&model.EzfyProfile{}).Where("id = ?", prof.ID).
				Update("diamond", prof.Diamond-cost).Error; err != nil {
				resp.ParamError(c, "扣钻石失败："+err.Error())
				return
			}
			// ★ 2026-09-28 钻石流水
			h.logDiamond(uid, -cost, "商城购买: "+cfg.Name)
		}
		if !unlimited {
			h.DB.Model(&model.EzfyCfgItem{}).Where("id = ?", req.CfgId).
				Updates(map[string]interface{}{"stock": stock - req.Count})
		}
		h.addItem(uid, req.CfgId, req.Count)
		resp.OK(c, gin.H{"msg": fmt.Sprintf("购买成功: %s×%d（消耗%d钻石）", cfg.Name, req.Count, cost),
			"stock_left": stockLeft(unlimited, stock, req.Count), "diamond": prof.Diamond - cost})
		return
	}
	cost := cfg.PriceGold * int64(req.Count)
	// 同上：分类锁定为「黄金道具」时，0 黄金 = 免费发放，不算「不支持黄金购买」。
	if cfg.PriceGold <= 0 && !freeByLock("gold") {
		resp.ParamError(c, fmt.Sprintf("「%s」不支持用黄金购买", cfg.Name))
		return
	}
	city := h.bodyCity(uid, req.CityId)
	if city.Gold < cost {
		resp.ParamError(c, "黄金不足")
		return
	}
	city.Gold -= cost
	h.saveCityRes(city)
	// 扣库存（用 map 更新，避免 GORM 的 default:100 把 0 当未设置）
	if !unlimited {
		h.DB.Model(&model.EzfyCfgItem{}).Where("id = ?", req.CfgId).
			Updates(map[string]interface{}{"stock": stock - req.Count})
	}
	h.addItem(uid, req.CfgId, req.Count)
	resp.OK(c, gin.H{"msg": fmt.Sprintf("购买成功: %s×%d", cfg.Name, req.Count),
		"stock_left": stockLeft(unlimited, stock, req.Count)})
}

// stockLeft 购买后剩余库存（无限库存返回 -1）
func stockLeft(unlimited bool, stock, count int) int {
	if unlimited {
		return -1
	}
	return stock - count
}

func (h *EzfyHandler) Bag(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	var items []model.EzfyItem
	h.DB.Where("user_id = ?", uid).Find(&items)
	views := []gin.H{}
	// ★ 2026-09-27 修复「背包相同道具分开显示太多」：
	//   同一道具(cfg_id)的 ezfy_item 可能存在多行(历史/分次获得)，原来逐行展示会铺满一页。
	//   这里按 cfg_id 合并求和，展示为统一的「道具名×总数」，保序取首次出现顺序。
	aggCount := map[int]int{}
	aggOrder := []int{}
	for _, it := range items {
		if it.Count <= 0 {
			continue
		}
		cfg := ezfyCfg.item(it.CfgId)
		if cfg == nil {
			continue
		}
		if _, seen := aggCount[it.CfgId]; !seen {
			aggOrder = append(aggOrder, it.CfgId)
		}
		aggCount[it.CfgId] += it.Count
	}
	for _, cid := range aggOrder {
		cfg := ezfyCfg.item(cid)
		views = append(views, gin.H{"cfg_id": cid, "count": aggCount[cid],
			"name": cfg.Name, "item_type": cfg.ItemType, "description": cfg.Description, "param1": cfg.Param1,
			// ★ 背包也按分类展示（与商城同一套归类口径，见 ezfyItemCategory）
			"category": ezfyItemCategory(cfg)})
	}
	// 军官类道具的目标选择需要军官列表与技能列表
	city := h.getOrCreateCity(uid)
	officers := []gin.H{}
	for _, o := range h.officerList(city.ID) {
		// ★ 2026-09-30 修复「野地军官/俘虏能被选为经验书目标」：俘虏不能作为军官类道具目标
		if o.IsCaptive == 1 {
			continue
		}
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
	// ★ 2026-09-28 背包展示宝物（用户要求「背包也要展示宝物, 相同宝物×数量」）：
	//   宝物 = 野地采集/宝物签到掉落的珠宝，存装备表(ezfy_equipment)且未穿戴(officer_id=0)，
	//   按 cfg_id 合并成「宝物名×数量」与道具一起下发。
	type trAgg struct {
		CfgId int
		Cnt   int64
	}
	var trAggs []trAgg
	h.DB.Model(&model.EzfyEquipment{}).Where("user_id = ? AND officer_id = 0", uid).
		Select("cfg_id, count(*) AS cnt").Group("cfg_id").Scan(&trAggs)
	trViews := []gin.H{}
	treasureSet := ezfyCollectibleTreasureNames() // ★ 只展示可采集的宝物（9 种珠宝），黑色幽灵[徽章]等普通装备不算宝物
	for _, a := range trAggs {
		cfg, ok := ezfyCfg.equipments[a.CfgId]
		if !ok || !treasureSet[cfg.Name] {
			continue
		}
		trViews = append(trViews, gin.H{"cfg_id": a.CfgId, "name": cfg.Name, "count": a.Cnt})
	}
	resp.OK(c, gin.H{"items": views, "treasures": trViews, "officers": officers, "skills": skills})
}

func (h *EzfyHandler) UseItem(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		CityId    int64 `json:"city_id"`
		CfgId     int   `json:"cfg_id"`
		Count     int   `json:"count"`
		OfficerId int64 `json:"officer_id"`
		SkillId   int   `json:"skill_id"`
		RecordId  int64 `json:"record_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	h.cfgs()
	city := h.bodyCity(uid, req.CityId)
	msg := h.useItem(uid, city, req.CfgId, req.Count, req.OfficerId, req.SkillId, req.RecordId)
	if msg == "" {
		// ★ 原来这里是写死的 "ok"（用户反馈的「提示还都是 ok」）
		resp.OKMsg(c, "道具使用成功", nil)
		return
	}
	if strings.HasPrefix(msg, "使用成功") {
		resp.OK(c, gin.H{"msg": msg})
		return
	}
	resp.ParamError(c, msg)
}

// ============ 任务 ============

// ★ 2026-10-04 性能（用户反馈「/tasks 线上 4s」）：整个 handler 重构为
//   「一次并行取数 → 纯内存处理」：
//   · 任务配置/我的任务/周期类型 三条独立查询并行打 RDS（原来串行 3 条）；
//   · 补建缺失任务由「每条配置一条 COUNT」改为「一次 Pluck + 内存判重」（N 条 → 0~1 条）；
//   · 周期重置由「每任务 2 条查询」改为「复用已取的配置/类型表」全内存判定；
//   · 状态型任务的值只按 task_type 算一次（原来同名任务重复算 N 遍）。
//   配合 3s 玩家级缓存（ezfyPageCacheGet/Set），命中时零 SQL。
func (h *EzfyHandler) Tasks(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	if it, ok := ezfyPageCacheGet(uid, "tasks"); ok {
		resp.OK(c, it)
		return
	}
	var cfgs []model.EzfyCfgTask
	var mine []model.EzfyTask
	var types []model.EzfyCfgTaskType
	var wg sync.WaitGroup
	wg.Add(3)
	go func() { defer wg.Done(); h.DB.Where("status = 1").Order("sort_no ASC").Find(&cfgs) }()
	go func() { defer wg.Done(); h.DB.Where("user_id = ?", uid).Find(&mine) }()
	go func() { defer wg.Done(); h.DB.Order("sort_no ASC").Find(&types) }()
	wg.Wait()
	today := time.Now().Format("2006-01-02")
	// 我的任务表（内存版）：补建缺失任务时同步写回，保证当次请求就能看到（与原 initTasks 行为一致）
	myMap := map[int]model.EzfyTask{}
	for _, t := range mine {
		myMap[t.CfgId] = t
	}
	// 周期类型表（周期重置用，复用上面已取的数据）
	typeMap := map[int]*model.EzfyCfgTaskType{}
	for i := range types {
		tp := &types[i]
		typeMap[tp.ID] = tp
	}
	cfgMap := map[int]*model.EzfyCfgTask{}
	for i := range cfgs {
		cfgMap[cfgs[i].ID] = &cfgs[i]
	}
	// 周期重置（原 resetPeriodTasks）：只对「周期已切换」的任务写库
	now := time.Now()
	for cfgID, t := range myMap {
		if t.ID == 0 { // 刚补建的，不可能跨周期，跳过重置
			continue
		}
		cfg, ok := cfgMap[cfgID]
		if !ok || cfg.TypeId <= 0 {
			continue
		}
		tp, ok := typeMap[cfg.TypeId]
		if !ok {
			continue
		}
		key := ezfyPeriodKey(tp.ResetType, now)
		if key == "" || t.TaskDate == key {
			continue
		}
		updates := map[string]interface{}{"task_date": key}
		if t.Current > 0 {
			updates["current"] = 0
			updates["status"] = 0
			t.Current, t.Status = 0, 0
		}
		t.TaskDate = key
		myMap[cfgID] = t
		h.DB.Model(&model.EzfyTask{}).Where("id = ?", t.ID).Updates(updates)
	}
	// 补建缺失任务（原 initTasks）——放在周期重置之后，与旧顺序一致
	for _, cfg := range cfgs {
		if _, ok := myMap[cfg.ID]; ok {
			continue
		}
		t := model.EzfyTask{UserId: uid, CfgId: cfg.ID, Current: 0, Status: 0, TaskDate: today}
		h.DB.Create(&t)
		myMap[cfg.ID] = t
	}
	// 状态型任务同步（原 calcStateValue 每任务查库，现按 task_type 去重只算一次）
	stateCache := map[string]int{}
	for _, cfg := range cfgs {
		if !ezfyStateTaskTypes[cfg.TaskType] {
			continue
		}
		t, ok := myMap[cfg.ID]
		if !ok || t.Status == 2 {
			continue
		}
		cur, ok := stateCache[cfg.TaskType]
		if !ok {
			cur = h.calcStateValue(uid, cfg.TaskType)
			stateCache[cfg.TaskType] = cur
		}
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
	groups := []gin.H{}
	byType := map[int][]gin.H{}
	order := []int{}
	for _, cfg := range cfgs {
		t, ok := myMap[cfg.ID]
		if !ok {
			continue
		}
		// ★ 新手任务资源 ×1000（见 ezfyTaskRewardRes），列表展示与发奖同口径
		df, ds, do, dr := ezfyTaskRewardRes(&cfg)
		row := gin.H{"id": t.ID, "cfg_id": cfg.ID, "name": cfg.Name, "target": cfg.Target,
			"current": t.Current, "status": t.Status,
			"reward": gin.H{"gold": cfg.RewardGold, "food": df, "steel": ds,
				"oil": do, "rare": dr, "prestige": cfg.RewardPrestige}}
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
	// ★ 为爱发电卡：未发放时 love_cards 为空数组（前端不显示该 tab）。
	//   一次查库复用（原 loveCardsView + loveCardTotalClaimable 各查一次）。
	loveCards := h.loveCards(uid)
	loveViews := make([]gin.H, 0, len(loveCards))
	loveTotal := 0
	nowMS := time.Now().UnixMilli()
	for i := range loveCards {
		loveViews = append(loveViews, loveCardView(&loveCards[i]))
		loveTotal += loveCardClaimable(&loveCards[i], nowMS)
	}
	data := gin.H{"groups": groups, "love_cards": loveViews, "love_total_claimable": loveTotal}
	ezfyPageCacheSet(uid, "tasks", data)
	resp.OK(c, data)
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
	ezfyPageCacheDel(uid) // 任务状态变了，3s 内不再返回旧列表
	h.done(c, h.taskAward(uid, req.TaskId), "奖励已领取")
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

// ★ 2026-09-28 宝物签到：7 天一轮；逢第 5/6/7 天多给（里程碑增量），方便不采集的懒人攒晋升宝物。
//   抽取范围 = 9 种采集宝物（装备配置 ID 27-35，见 ezfyTerrainTreasureNames）。
var ezfyTreasureSignRewards = [7]int{2, 2, 2, 2, 4, 6, 8} // position(1-7) → 当日宝物件数

// ezfyTreasureSignQty 连续宝物签到天数 → 当天应得宝物件数（7 天循环）
func ezfyTreasureSignQty(count int) int {
	pos := (count-1)%7 + 1
	return ezfyTreasureSignRewards[pos-1]
}

// ezfyTreasureSignNames 9 种采集宝物的配置名（用于展示与随机抽取）
func ezfyTreasureSignNames() []string {
	return []string{"黄金手镯", "玛瑙项坠", "红宝石戒指", "黑曜石戒指",
		"琥珀项链", "铂金戒指", "翡翠项链", "祖母绿", "蓝宝石戒指"}
}

func (h *EzfyHandler) giveResources(uid uint, food, steel, oil, rare, gold int64) {
	city := h.getOrCreateCity(uid)
	// ★ 2026-09-30 恢复「资源最大值唯一硬上限」：任何累加都不得超过 21 亿（见 ezfyAddResMax）。
	//   （2026-09-24 曾放宽为只夹 1 万亿，导致资源能累加超上限，用户已反馈为 bug。）
	city.Food = ezfyAddResMax("food", city.Food, food)
	city.Steel = ezfyAddResMax("steel", city.Steel, steel)
	city.Oil = ezfyAddResMax("oil", city.Oil, oil)
	city.Rare = ezfyAddResMax("rare", city.Rare, rare)
	city.Gold = ezfyAddResMax("gold", city.Gold, gold)
	h.saveCityRes(&city)
}

// giveResNoCap 给「指定城市」加资源，**不按仓储上限截断**。
//
// 用于退还类操作（取消训练，2026-09-28 取消研究改为不退款），避免玩家觉得「退少了」。
// 负数是合法的，结果不会低于 0。
// ★ 2026-09-30：仍受「资源最大值」硬上限约束（累计不得超过 21 亿），见 ezfyAddResMax。
func (h *EzfyHandler) giveResNoCap(city *model.EzfyCity, food, steel, oil, rare, gold int64) {
	city.Food = ezfyAddResMax("food", city.Food, food)
	city.Steel = ezfyAddResMax("steel", city.Steel, steel)
	city.Oil = ezfyAddResMax("oil", city.Oil, oil)
	city.Rare = ezfyAddResMax("rare", city.Rare, rare)
	city.Gold = ezfyAddResMax("gold", city.Gold, gold)
	h.saveCityRes(city)
}

// giveResourcesNoCap 管理端专用发放：**不按仓储上限截断**。
//
// ★ 2026-09-30 用户要求「资源不能累加超过资源最大值（每项资源唯一硬上限，默认 21 亿）」：
//   管理端发放同样封顶在 21 亿（老数据已超的不拉低、也不再增长）。
//   如确需超过当前上限，请先在「二战系统配置」把对应 资源最大值 调高再发。
//   负数是合法的（可用来扣减），但结果不会低于 0。
func (h *EzfyHandler) giveResourcesNoCap(uid uint, food, steel, oil, rare, gold int64) {
	city := h.getOrCreateCity(uid)
	city.Food = ezfyAddResMax("food", city.Food, food)
	city.Steel = ezfyAddResMax("steel", city.Steel, steel)
	city.Oil = ezfyAddResMax("oil", city.Oil, oil)
	city.Rare = ezfyAddResMax("rare", city.Rare, rare)
	city.Gold = ezfyAddResMax("gold", city.Gold, gold)
	h.saveCityRes(&city)
}

func (h *EzfyHandler) Welfare(c *gin.Context) {
	uid := middleware.GetUID(c)
	today := time.Now().Format("2006-01-02")
	yest := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	// ★ 2026-10-03 性能：以下 8 组只读查询互不依赖，并行打 RDS，
	//   把 welfare 页的串行查询压成一次往返。
	var (
		signedToday, signedYest             int64
		yestSign                            model.EzfySign
		gifts                               = gin.H{}
		city                                model.EzfyCity
		profile                             model.EzfyProfile
		trsSignedToday, trsSignedYest       int64
		yestTrs                             model.EzfyTreasureSign
		trsReward                           string
	)
	var wg sync.WaitGroup
	wg.Add(8)
	go func() { // 今日是否已签到
		defer wg.Done()
		var n int64
		h.DB.Model(&model.EzfySign{}).Where("user_id = ? AND sign_date = ?", uid, today).Count(&n)
		signedToday = n
	}()
	go func() { // 昨日签到记录（决定连续签到天数 +1）
		defer wg.Done()
		var n int64
		h.DB.Model(&model.EzfySign{}).Where("user_id = ? AND sign_date = ?", uid, yest).Count(&n)
		signedYest = n
		if n > 0 {
			h.DB.Where("user_id = ? AND sign_date = ?", uid, yest).First(&yestSign)
		}
	}()
	go func() { // 三个礼包是否已领取（新手/每周/市政厅10级）
		defer wg.Done()
		g := gin.H{}
		for _, t := range []string{"newbie", "weekly", "level10"} {
			var n int64
			h.DB.Model(&model.EzfyGift{}).Where("user_id = ? AND gift_type = ?", uid, t).Count(&n)
			g[t] = n > 0
		}
		gifts = g
	}()
	go func() { // 主城
		defer wg.Done()
		city = h.getOrCreateCity(uid)
	}()
	go func() { // 玩家档案
		defer wg.Done()
		profile = h.ensureProfile(uid)
	}()
	go func() { // 宝物签到：今日状态
		defer wg.Done()
		var n int64
		h.DB.Model(&model.EzfyTreasureSign{}).Where("user_id = ? AND sign_date = ?", uid, today).Count(&n)
		trsSignedToday = n
	}()
	go func() { // 宝物签到：昨日记录（决定宝物连续天数 +1）
		defer wg.Done()
		var n int64
		h.DB.Model(&model.EzfyTreasureSign{}).Where("user_id = ? AND sign_date = ?", uid, yest).Count(&n)
		trsSignedYest = n
		if n > 0 {
			h.DB.Where("user_id = ? AND sign_date = ?", uid, yest).First(&yestTrs)
		}
	}()
	go func() { // 宝物签到：今日已获得宝物名
		defer wg.Done()
		trsReward = h.ezfyTreasureRewardToday(uid, today)
	}()
	wg.Wait()

	signCount := 1
	if signedYest > 0 {
		signCount = yestSign.SignCount + 1
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
	// ★ 2026-10-03 性能：礼包/主城/档案/宝物签到状态已在上方 WaitGroup 并行取好，这里只剩纯计算。
	trsCount := 1
	if trsSignedYest > 0 {
		trsCount = yestTrs.SignCount + 1
	}
	resp.OK(c, gin.H{
		"signed_today": signedToday > 0, "sign_count": signCount,
		"rewards": rewards, "gifts": gifts, "city_level": city.CityLevel,
		"prestige": profile.Prestige, "rank_name": ezfyRankNameAt(ezfyProfileRank(&profile)),
		// ★ 2026-09-28 宝物签到
		"treasure_signed_today": trsSignedToday > 0, "treasure_count": trsCount,
		"treasure_qty":    ezfyTreasureSignQty(trsCount),
		"treasure_reward": trsReward,
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
	// ★ 2026-09-26 线上「无限签到刷资源」事故：必须**先落库、落库成功才发奖**。
	//   旧代码 `h.DB.Create(...)` 不看 error，而 ezfy_sign 早期又只有 sign_date 单列唯一索引
	//   （全服每天只放行一条），于是除第一个玩家外所有人的 INSERT 都静默失败、奖励却照发
	//   → 反复请求即可无限刷资源，且自己的记录从未入库，前端一直显示「未签到」。
	//   现在 (user_id, sign_date) 复合唯一索引 + 这里校验 error，双保险：
	//   并发重复请求也只会有一次插入成功，其余一律拒绝、不发奖。
	if err := h.DB.Create(&model.EzfySign{UserId: uid, SignDate: today, SignCount: count}).Error; err != nil {
		resp.ParamError(c, "今天已经签到过了")
		return
	}
	r := ezfySignRewards[(count-1)%7]
	// ★ 签到奖励不受仓储上限截断（用户要求：签到/任务/礼包领到的资源不能被上限吃掉）
	h.giveResourcesNoCap(uid, r[1], r[2], r[3], r[4], r[0])
	if r[5] > 0 {
		h.addPrestige(uid, int(r[5]))
	}
	resp.OK(c, gin.H{"msg": fmt.Sprintf("签到成功(连续%d天), 奖励已发放", count)})
}

// ezfyTreasureRewardToday 今日宝物签到已获得的宝物名（逗号分隔），未签则空串
func (h *EzfyHandler) ezfyTreasureRewardToday(uid uint, today string) string {
	var ts model.EzfyTreasureSign
	if err := h.DB.Where("user_id = ? AND sign_date = ?", uid, today).First(&ts).Error; err != nil {
		return ""
	}
	return ts.NReward
}

// TreasureSign POST /games/ezfy/welfare/treasure-sign —— 宝物签到（7 天一轮，逢 5/6/7 天多给）
func (h *EzfyHandler) TreasureSign(c *gin.Context) {
	uid := middleware.GetUID(c)
	today := time.Now().Format("2006-01-02")
	// 与每日签到同款防刷：先落库（(user_id, sign_date) 复合唯一索引），成功才发奖
	var exist model.EzfyTreasureSign
	if err := h.DB.Where("user_id = ? AND sign_date = ?", uid, today).First(&exist).Error; err == nil {
		resp.ParamError(c, "今天宝物已经签到过了")
		return
	}
	yest := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	count := 1
	var y model.EzfyTreasureSign
	if err := h.DB.Where("user_id = ? AND sign_date = ?", uid, yest).First(&y).Error; err == nil {
		count = y.SignCount + 1
	}
	if err := h.DB.Create(&model.EzfyTreasureSign{UserId: uid, SignDate: today, SignCount: count}).Error; err != nil {
		resp.ParamError(c, "今天宝物已经签到过了")
		return
	}
	qty := ezfyTreasureSignQty(count)
	names := ezfyTreasureSignNames()
	// ★ 2026-09-28 修复「签到宝物没到账」：原来 addItem 发进道具表(ezfy_item)，
	//   而宝物(装备配置 27-35)采集掉落是进装备表(ezfy_equipment)——两套库导致背包、军衔晋升都看不到。
	//   统一改为 addEquipment 进装备表，与采集掉宝同一口径。
	city, ok := h.anyCity(uid)
	if !ok {
		city = h.getOrCreateCity(uid) // 福利页必有游戏存档
	}
	var won []string
	for i := 0; i < qty; i++ {
		name := names[rand.Intn(len(names))]
		if cfg := ezfyEquipCfgByName(name); cfg != nil {
			h.addEquipment(&city, cfg)
			won = append(won, name)
		}
	}
	// ★ 2026-09-28 用户要求展示签到领到的具体宝物名 → 落库, 前端已签时展示
	h.DB.Model(&model.EzfyTreasureSign{}).Where("user_id = ? AND sign_date = ?", uid, today).
		Update("n_reward", strings.Join(won, ","))
	resp.OK(c, gin.H{"msg": fmt.Sprintf("宝物签到成功(连续%d天), 获得%d件宝物: %s", count, qty, strings.Join(won, ","))})
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
		h.giveResourcesNoCap(uid, 50000, 30000, 20000, 10000, 5000)
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
		h.giveResourcesNoCap(uid, 20000, 20000, 20000, 20000, 2000)
		recordGift("weekly")
		resp.OK(c, gin.H{"msg": "每周福利领取成功"})
	// ★ 用户要求删掉「市政厅20/30/40级礼包」→ 只剩 市政厅10级礼包。
	//   注意：老的 ezfy_gifts 里 level20/30/40 的历史领取记录不删（只是不再有入口）。
	case "level10":
		needLevel, gold, res := 10, int64(5000), int64(50000)
		if hasGift(giftType) {
			resp.ParamError(c, "该礼包已领取")
			return
		}
		city := h.getOrCreateCity(uid)
		if city.CityLevel < needLevel {
			resp.ParamError(c, fmt.Sprintf("%s需要达到%d级", h.buildingName(1), needLevel))
			return
		}
		h.giveResourcesNoCap(uid, res, res*3/5, res*2/5, res/5, gold)
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

	// ★ 用户要求「首页公告默认只能展示一条，管理端可以配置」：
	//   首页外露公告取前 N 条（N = ezfy_cfg_limit.notice_home_count，默认 1，0 = 不展示）。
	limit := h.ezfyNoticeHomeCount()
	home := []model.EzfyNotice{}
	if limit > 0 && len(notices) > 0 {
		n := limit
		if n > len(notices) {
			n = len(notices)
		}
		home = notices[:n]
	}
	resp.OK(c, gin.H{"notices": notices, "home_notices": home, "home_limit": limit})
}

// ezfyNoticeHomeCount 首页公告展示条数（读 ezfy_cfg_limit，缺省 1）
func (h *EzfyHandler) ezfyNoticeHomeCount() int {
	var lim model.EzfyCfgLimit
	if err := h.DB.First(&lim, 1).Error; err != nil {
		return 1
	}
	if lim.NoticeHomeCount < 0 {
		return 1
	}
	return lim.NoticeHomeCount
}

// ============ 战报 ============

// ezfyReportCategory 战报归类(复刻原版军情页的三块: 军队动态/军情警讯/战斗报告)
//
//	1 军情警讯 —— 别人打我(雷达预警 / 被掠夺 / 城破 / 守卫)
//	2 战斗报告 —— 我发起的战斗结果(侦查 / 掠夺 / 征服)
//	3 其他     —— 采集派遣 / 增援运输 / 系统消息
//
// ezfyReportCategory 战报归类（复刻 report/index.html 的三个分区）
//
//	1 军情警讯 —— 别人打我 / 我的地盘出事(雷达预警、被侦查、被掠夺、城破、守卫、被归还、野地丢失)
//	2 战斗报告 —— 我打别人(侦查 / 掠夺 / 征服), 供「战报查询」
//	3 其他     —— 后勤与系统(采集、运输、增援、派遣、建城、交易、将领变动)
func ezfyReportCategory(title string) int {
	switch {
	case strings.HasPrefix(title, "军情警报"),
		strings.HasPrefix(title, "被侦查报告"), // 雷达站≥1 才收得到
		strings.HasPrefix(title, "被掠夺报告"),
		strings.HasPrefix(title, "城破报告"),
		strings.HasPrefix(title, "守卫报告"),
		strings.HasPrefix(title, "城市归还"),
		strings.HasPrefix(title, "将领叛离"), // 被攻打后忠诚归零叛离, 属于军情警讯
		strings.Contains(title, "野地丢失"):
		return 1
	case strings.HasPrefix(title, "侦查报告"),
		strings.HasPrefix(title, "掠夺报告"),
		strings.HasPrefix(title, "战斗报告"),
		strings.HasPrefix(title, "征服报告"),
		// ★ 活动目标战报标题形如「活动野地3级战斗报告: 活动野地3级(323,69)」，
		//   不是以「战斗报告」开头，老逻辑把它误归到「其他(3)」→ 战斗报告页查不到
		//   （用户反馈「打活动城市没有战报」）。这里按标题**包含**「战斗报告」兜底归入。
		strings.Contains(title, "战斗报告"):
		return 2
	}
	return 3
}

// ezfyReportTypeName 战报标签(前端列表里的 [xxx] 前缀)
//
// ★ 不能只看 report_type：老代码把「掠夺/战斗/被掠夺」都写成 2，
//
//	导致战报列表里清一色显示 [掠夺]（用户反馈「都是掠夺」）。
//	这里优先按**标题前缀**判定，标题没有可识别前缀时才退回 report_type。
func ezfyReportTypeName(reportType int, title string) string {
	switch {
	case strings.HasPrefix(title, "被掠夺报告"):
		return "被掠夺"
	case strings.HasPrefix(title, "被侦查报告"):
		return "被侦查"
	case strings.HasPrefix(title, "城破报告"):
		return "城破"
	case strings.HasPrefix(title, "守卫报告"):
		return "防守报告"
	case strings.HasPrefix(title, "军情警报"):
		return "预警"
	case strings.HasPrefix(title, "侦查报告"):
		return "侦查"
	case strings.HasPrefix(title, "掠夺报告"):
		return "掠夺"
	case strings.HasPrefix(title, "征服报告"):
		return "征服"
	case strings.Contains(title, "战斗报告"):
		return "战斗"
	case strings.HasPrefix(title, "采集报告"):
		return "采集"
	case strings.HasPrefix(title, "运输报告"):
		return "运输"
	case strings.HasPrefix(title, "增援报告"):
		return "增援"
	case strings.HasPrefix(title, "派遣报告"):
		return "派遣"
	case strings.HasPrefix(title, "建城报告"):
		return "建城"
	case strings.HasPrefix(title, "交易报告"):
		return "交易"
	case strings.HasPrefix(title, "将领叛离"):
		return "叛离"
	case strings.HasPrefix(title, "城市归还"):
		return "归还"
	case strings.HasPrefix(title, "部队返航"):
		return "返航"
	case strings.HasPrefix(title, "运输到达"):
		return "运输"
	case strings.HasPrefix(title, "增援到达"):
		return "增援"
	case strings.HasPrefix(title, "新城建成"), strings.HasPrefix(title, "建城成功"):
		return "建城"
	case strings.HasPrefix(title, "城市已摧毁"), strings.HasPrefix(title, "摧毁城市"):
		return "摧毁"
	case strings.HasPrefix(title, "城市被占领"):
		return "占领"
	case strings.HasPrefix(title, "城市迁移完成"):
		return "迁移"
	case strings.HasPrefix(title, "计谋发动"), strings.HasPrefix(title, "计谋:"):
		return "计谋"
	case strings.HasPrefix(title, "军官调遣"):
		return "调遣"
	case strings.HasPrefix(title, "将领离职"):
		return "离职"
	case strings.HasPrefix(title, "将领升级"):
		return "升级"
	case strings.HasPrefix(title, "交易成交"):
		return "交易"
	}
	// 兜底：按 report_type
	switch reportType {
	case 1:
		return "侦查"
	case 2:
		return "战斗"
	case 3:
		return "征服"
	case 4:
		return "战斗"
	case 5:
		return "采集"
	case 6:
		return "系统"
	}
	return "战报"
}

func ezfyReportCategoryName(cat int) string {
	switch cat {
	case 1:
		return "军情警讯"
	case 2:
		return "战斗报告"
	case 3:
		return "其他"
	}
	return "全部"
}

// ezfyCityReportCond 军情按城市过滤的条件片段（cityId > 0 时拼到 WHERE 里）。
//
// ★ 2026-10-01 修复「按城市检索后战报看不见」：老战报因 addReport uint bug
//   order_id 全为 0，原来 `city_id = 0 AND order_id IN (该城订单)` 永远匹配不上，
//   且大量老战报 city_id 存的是 NULL（141 条）——`city_id = 0` 同样匹配不上，
//   导致历史战报从城市视角全部消失。改为**按标题坐标反查该城的出征订单**归属：
//   标题形如「战斗报告: 活动野地3级(258,100)」，与 ezfy_order.target_x/y 比对，
//   city_id 为 0 或 NULL 的老战报都走这条路。新战报（city_id>0）仍走第一分支；
//   无匹配订单的老防守/系统战报无法归属城市，仅在「全部」视图展示。
// ★ 2026-10-02 修复「本城战报看不到」：老防守战报（被侦查/被掠夺/城破/守卫等）
//   city_id=0/NULL 且标题**只有城市名、无坐标**（如「被掠夺报告: 无忧」），
//   上面两条路（city_id 直配、标题坐标反查出征订单）都匹配不上，从城市视角全部消失。
//   这里加第三条路：按标题包含的**城名**反查我的 ezfy_city 归属该城。
func ezfyCityReportCond(cityId int64) string {
	return fmt.Sprintf(`(city_id = %d OR ((city_id = 0 OR city_id IS NULL) AND (
		EXISTS (
			SELECT 1 FROM ezfy_order o
			WHERE o.user_id = ezfy_report.user_id AND o.city_id = %d
			  AND ezfy_report.title LIKE CONCAT('%%', CONCAT(CONCAT('(', o.target_x), CONCAT(',', CONCAT(o.target_y, ')'))), '%%')
		)
		OR EXISTS (
			SELECT 1 FROM ezfy_city ct
			WHERE ct.user_id = ezfy_report.user_id AND ct.id = %d
			  AND ezfy_report.title LIKE CONCAT('%%', ct.name, '%%')
		)
	)))`, cityId, cityId, cityId)
}

// ezfyReportCounts 统计军情警讯(1)/战斗报告(2)的**真实**数量（tab 徽标数字）。
//
// ★ 2026-10-01 修复「徽标数字时有时无/无故漂移」：原来在 Reports 里用
//   「最近 200 条的窗口计数」——新报告把旧报告挤出窗口后，数字在玩家什么都没
//   删的情况下自己变少甚至归零。这里改单条 SQL 按标题条件聚合全量，与
//   ezfyReportCategory 的判定规则保持同步（改判定时这里要一起改）。
// ★ 2026-10-01 军情按当前城过滤：cityId>0 时只统计该城的战报
//   （新战报带 city_id；老攻击战报 city_id=0 按标题坐标反查订单归属城市）。
func (h *EzfyHandler) ezfyReportCounts(uid uint, cityId int64) map[int]int {
	cityCond := ""
	if cityId > 0 {
		cityCond = " AND " + ezfyCityReportCond(cityId)
	}
	row := h.DB.Raw(`SELECT
		COALESCE(SUM(CASE WHEN title LIKE '军情警报%' OR title LIKE '被侦查报告%' OR title LIKE '被掠夺报告%'
			OR title LIKE '城破报告%' OR title LIKE '守卫报告%' OR title LIKE '城市归还%'
			OR title LIKE '将领叛离%' OR title LIKE '%野地丢失%' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN title LIKE '侦查报告%' OR title LIKE '掠夺报告%'
			OR title LIKE '战斗报告%' OR title LIKE '征服报告%' OR title LIKE '%战斗报告%' THEN 1 ELSE 0 END), 0)
		FROM ezfy_report WHERE user_id = ?`+cityCond, uid).Row()
	var c1, c2 int
	if row != nil {
		row.Scan(&c1, &c2)
	}
	return map[int]int{1: c1, 2: c2}
}

// ReportCounts GET /games/ezfy/reports/counts?city_id=xx —— 只取 tab 徽标数字，不标记已读。
// （军情页无论落在哪个分区都要刷新徽标，但不能因此把没看的战报标记成已读，
//  所以从 Reports 里拆出独立接口。）
func (h *EzfyHandler) ReportCounts(c *gin.Context) {
	uid := middleware.GetUID(c)
	cityId, _ := strconv.ParseInt(c.DefaultQuery("city_id", "0"), 10, 64)
	resp.OK(c, gin.H{"counts": h.ezfyReportCounts(uid, cityId)})
}

// Reports GET /games/ezfy/reports?category=1|2|3&word=xxx&city_id=xx
func (h *EzfyHandler) Reports(c *gin.Context) {
	uid := middleware.GetUID(c)
	category, _ := strconv.Atoi(c.DefaultQuery("category", "0"))
	word := strings.TrimSpace(c.Query("word"))
	cityId, _ := strconv.ParseInt(c.DefaultQuery("city_id", "0"), 10, 64)
	// ★ 2026-09-30 军团战报：展示本军团团员的 PvP 战报（不含 NPC/野地/系统）
	if c.Query("corps") == "1" {
		h.corpsReports(c, uid, word)
		return
	}

	q := h.DB.Where("user_id = ?", uid)
	if word != "" {
		q = q.Where("title LIKE ?", "%"+word+"%")
	}
	// ★ 2026-10-01 军情按当前城过滤：cityId>0 时只拉当前城的战报。
	//   新战报已写 city_id；老战报(city_id=0)按标题坐标反查该城出征订单归属。
	if cityId > 0 {
		q = q.Where(ezfyCityReportCond(cityId))
	}
	var reports []model.EzfyReport
	q.Order("id DESC").Limit(200).Find(&reports)

	views := []gin.H{}
	// ★ 徽标数字用全量真实统计（不再受「最近 200 条窗口」影响，见 ezfyReportCounts）
	counts := h.ezfyReportCounts(uid, cityId)
	for _, r := range reports {
		cat := ezfyReportCategory(r.Title)
		if category > 0 && cat != category {
			continue
		}
		if len(views) >= 50 {
			continue
		}
		views = append(views, gin.H{"id": r.ID, "title": r.Title, "report_type": r.ReportType,
			"type_name": ezfyReportTypeName(r.ReportType, r.Title), "is_read": r.IsRead, "order_id": r.OrderId,
			"category": cat, "category_name": ezfyReportCategoryName(cat),
			"created_at": r.CreatedAt})
		if r.IsRead == 0 {
			h.DB.Model(&model.EzfyReport{}).Where("id = ?", r.ID).Update("is_read", 1)
		}
	}
	// ★ 军情警讯的可见范围由**自己城市的雷达站**决定：没有雷达站收不到「来袭预警/被侦查」，
	//   但被掠夺/城破这类事后结果照样会有。这里把雷达等级一并下发，前端据此给提示。
	// ★ 2026-09-25：同时下发**侦察技巧等级**与**合计情报等级** —— 现在「出发城市+坐标」
	//   由「雷达站 + 侦察技巧」合计决定，前端要按这两个值告诉玩家还差多少才能看到来袭城市。
	// ★ 2026-10-01 军情按当前城过滤：雷达/情报等级也取**所查看的城市**，不是主城
	radarCity := h.getOrCreateCity(uid)
	if cityId > 0 {
		if ct := h.cityOf(uid, cityId); ct != nil {
			radarCity = *ct
		}
	}
	resp.OK(c, gin.H{"reports": views, "counts": counts,
		"radar": h.buildingLevel(radarCity.ID, ezfyRadarBuildingID),
		"recon": h.techMap(radarCity.ID)[ezfyReconTechID],
		"intel": h.ezfyIntelLevel(radarCity.ID)})
}

// corpsReports 军团战报 —— 展示本军团团员的 PvP 战斗战报（对战玩家城，
// 不含野地/寇城/活动目标/系统消息）。
func (h *EzfyHandler) corpsReports(c *gin.Context, uid uint, word string) {
	empty := func() { resp.OK(c, gin.H{"reports": []gin.H{}, "counts": map[int]int{}, "corps": true}) }
	myCorp := h.corpsOfUser(uid)
	if myCorp == 0 {
		empty()
		return
	}
	var members []model.EzfyCorpsMember
	h.DB.Where("corps_id = ?", myCorp).Find(&members)
	uids := make([]uint, 0, len(members))
	for _, m := range members {
		uids = append(uids, m.UserId)
	}
	if len(uids) == 0 {
		empty()
		return
	}
	q := h.DB.Where("user_id IN ? AND report_type IN (1,2,3,4)", uids)
	if word != "" {
		q = q.Where("title LIKE ?", "%"+word+"%")
	}
	var reports []model.EzfyReport
	q.Order("id DESC").Limit(500).Find(&reports)

	// 判定 PvP：报告的 order 必须 target_type==3（玩家城）；野地(1)/寇城(2) 排除。
	orderIds := []int64{}
	for _, r := range reports {
		if r.OrderId > 0 {
			orderIds = append(orderIds, r.OrderId)
		}
	}
	pvp := map[int64]bool{}
	if len(orderIds) > 0 {
		var orders []model.EzfyOrder
		h.DB.Select("id, target_type").Where("id IN ?", orderIds).Find(&orders)
		for _, o := range orders {
			if o.TargetType == 3 {
				pvp[int64(o.ID)] = true
			}
		}
	}
	// 归属团员昵称
	ownerName := map[uint]string{}
	ids := []uint{}
	for _, r := range reports {
		if _, ok := ownerName[r.UserID]; !ok {
			ownerName[r.UserID] = ""
			ids = append(ids, r.UserID)
		}
	}
	if len(ids) > 0 {
		var ps []model.EzfyProfile
		h.DB.Select("user_id, nickname").Where("user_id IN ?", ids).Find(&ps)
		for _, p := range ps {
			if p.Nickname != "" {
				ownerName[p.UserID] = p.Nickname
			}
		}
	}
	counts := map[int]int{}
	views := []gin.H{}
	for _, r := range reports {
		if r.ReportType == 6 || r.OrderId <= 0 || !pvp[r.OrderId] {
			continue
		}
		cat := ezfyReportCategory(r.Title)
		counts[cat]++
		if len(views) >= 50 {
			continue
		}
		views = append(views, gin.H{"id": r.ID, "title": r.Title, "report_type": r.ReportType,
			"type_name": ezfyReportTypeName(r.ReportType, r.Title), "is_read": 1, "order_id": r.OrderId,
			"owner_name": ownerName[r.UserID],
			"category": "corps", "category_name": "军团战报", "created_at": r.CreatedAt})
	}
	resp.OK(c, gin.H{"reports": views, "counts": counts, "corps": true})
}

// ReportDynamics GET /games/ezfy/reports/dynamics?city_id=xx
// 军队动态: 所有在外的部队(出征/采集/派遣/侦查/掠夺/运输/增援)
// 复刻 `二战风云/templates/report/index.html` 的「军队动态」区
// ★ 2026-10-01 军情按当前城过滤：city_id>0 时只展示当前城出发的部队，
//   防守战场也只展示「正在攻打当前城」的（守方视角）。
func (h *EzfyHandler) ReportDynamics(c *gin.Context) {
	uid := middleware.GetUID(c)
	cityId, _ := strconv.ParseInt(c.DefaultQuery("city_id", "0"), 10, 64)
	h.cfgs()
	// ★ 走统一懒结算：原来这里只处理了「抵达(0)」和「返航(2)」，
	//   漏掉了「驻守采集(1) 到点结算」，导致军队动态页看到的采集进度/待带回资源是旧的。
	h.processOrders(uid)
	now := time.Now().UnixMilli()
	var orders []model.EzfyOrder
	// ★ 2026-10-02 用户要求「军情 → 驻军要展示自己驻守盟友城市的驻军」：
	//   status=3(常驻) 的增援订单(到友军城)此前不在动态查询内, 出站驻军不可见。
	//   一并纳入；仅展示「活跃驻军」= result 为空(未返航过) 且目标城属于他人，
	//   排除增援自己城市的僵尸订单与已归队订单。
	allyCity := h.DB.Model(&model.EzfyCity{}).Select("id").Where("user_id <> ?", uid)
	oq := h.DB.Where("user_id = ? AND (status IN (0,1,2,?,?) OR (status = 3 AND order_type = 6 AND result = '' AND target_id IN (?)))",
		uid, ezfyOrderStatusBattle, ezfyOrderStatusWaiting, allyCity)
	if cityId > 0 {
		oq = oq.Where("city_id = ?", cityId)
	}
	oq.Order("id DESC").Limit(100).Find(&orders)
	// ★ 性能：战场进度**一次查完**再按 order_id 取。
	//   原来在下面的循环里逐条 ezfyBattleByOrder = N+1（服务器只有 1 核，这条红线不能踩）。
	battleRounds := map[int64]int{}
	battleLeftMs := map[int64]int64{}
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
			battleRounds[b.OrderId] = b.Round
			left := b.RoundStart + ezfyBattleRoundMs - now
			if left < 0 {
				left = 0
			}
			battleLeftMs[b.OrderId] = left
		}
	}
	views := []gin.H{}
	// ★ 2026-09-28 用户要求「累计采集/采集资源要实时变化、累加展示，不能只靠刷新」：
	//   先批量查出**采集中**部队的野地，后端据此下发「每期产出」，
	//   前端拿到 accept 之后用本地下发时间实时 extrapolate 累加，不必等满一期才结算。
	gatherWl := map[int64]*model.EzfyWildland{}
	var gatherWlIDs []int64
	for i := range orders {
		o := &orders[i]
		if o.Status == 1 && o.OrderType == 7 && o.ArriveTime > 0 {
			gatherWlIDs = append(gatherWlIDs, o.TargetId)
		}
	}
	if len(gatherWlIDs) > 0 {
		var wls []model.EzfyWildland
		h.DB.Where("id IN ?", gatherWlIDs).Find(&wls)
		for i := range wls {
			wp := &wls[i]
			gatherWl[int64(wp.ID)] = wp
		}
	}
	// ★ 2026-09-28 用户要求「军队动态/驻军要显示这支部队是从哪个城出来的」：
	//   订单表只存了 city_id（军队所在城 id），**没有下发城名/坐标**，前端看不出番号。
	//   这里一次批量查出来（禁止在下面循环里逐条查 = N+1，1 核服务器红线），
	//   下发 from_city / from_x / from_y 三个字段。
	fromCity := map[int64]*model.EzfyCity{}
	var fromCityIDs []int64
	seenFrom := map[int64]bool{}
	for i := range orders {
		cid := orders[i].CityId
		if cid > 0 && !seenFrom[cid] {
			seenFrom[cid] = true
			fromCityIDs = append(fromCityIDs, cid)
		}
	}
	if len(fromCityIDs) > 0 {
		var cs []model.EzfyCity
		h.DB.Where("id IN ?", fromCityIDs).Find(&cs)
		for i := range cs {
			cp := &cs[i]
			fromCity[int64(cp.ID)] = cp
		}
	}
	for i := range orders {
		o := &orders[i]
		timeLabel, timeText := "", ""
		statusName := ""
		// ★ 采集实况：采集中部队下发 per 每期产出 + 集 = 前置项，前端每秒本地累加展示
		var gather *gin.H
		if o.Status == 1 && o.OrderType == 7 && o.ArriveTime > 0 {
			periodMs := ezfyDispatchPeriod()
			var pf, ps, po, pr int64
			if wl := gatherWl[o.TargetId]; wl != nil && wl.CityId == o.CityId {
				// 单期产出（时长=整一个结算周期，抽掉「按比例折算」那部分）
				pf, ps, po, pr, _, _ = h.dispatchGatherYield(o, wl, periodMs)
			}
			gather = &gin.H{
				"start_ms":  o.CollectStart, // 本期从哪个时刻开始累计
				"period_ms": periodMs,       // 一个结算周期的毫秒数
				"per_food":  pf,
				"per_steel": ps,
				"per_oil":   po,
				"per_rare":  pr,
			}
		}
		switch o.Status {
		case 0:
			statusName = "出征"
			timeLabel = "抵达时间"
			timeText = ezfyDurationText((o.ArriveTime - now) / 1000)
		case 1:
			if o.OrderType == 7 && o.ArriveTime <= 0 {
				// ★ 2026-09-24 用户规则: 采集部队到达后驻守**空闲**, 手工点[采集]才进入采集
				statusName = "驻守(空闲)"
				timeLabel = "待机"
				timeText = "空闲待命, 点[采集]开始采集"
			} else {
				statusName = "驻守采集"
				// ★ 2026-09-28 用户要求: 采集不再显示「下次结算」倒计时,
				//   改为「累计采集了多长时间」(按小时/分钟累计, 从 collect_start 起算)。
				timeLabel = "累计采集"
				accum := now - o.CollectStart
				if o.CollectStart <= 0 || accum < 0 {
					timeText = "刚采集"
				} else {
					timeText = ezfyDurationText(accum / 1000)
				}
			}
		case 2:
			statusName = "返回"
			timeLabel = "返回时间"
			timeText = ezfyDurationText((o.ReturnTime - now) / 1000)
		case 3:
			// ★ 2026-10-02 出站驻军(增援到友军城, 常驻待命)：不采集, 可[召回]或等对方城主[遣返]
			statusName = "驻守中"
			timeLabel = "驻守"
			timeText = "增援友军城, 不可采集; 可[召回]撤兵或由对方城主[遣返]"
		case ezfyOrderStatusBattle:
			statusName = "战斗中"
			timeLabel = "本回合剩余"
		}
		// ★ 指挥室（2026-09-22 用户要求）：战斗中的部队带上回合进度与本回合倒计时，
		//   前端据此在这一行显示 [指挥] 入口（军情 → 军队动态 → 指挥）。
		battleRound, battleLeft := 0, int64(0)
		if o.Status == ezfyOrderStatusBattle {
			if left, ok := battleLeftMs[int64(o.ID)]; ok {
				battleRound = battleRounds[int64(o.ID)]
				battleLeft = left
				timeText = ezfyDurationText(left / 1000)
			} else {
				timeText = "等待指挥"
			}
		}
		// ★ 采集部队带上「待带回资源」与负重，前端可展示（资源要召回才入城）
		c := parseCarry(o.Carry)
		// ★ 活动目标标识：攻击/征服(掠夺)部队目的地在活动格时下发 act_type，
		//   前端据此在坐标旁加「活动」标示（用户要求「打活动坐标要有标识」）。
		// ★ 2026-10-05 名将野地按玩家判定：已抓到守将的玩家 → 该坐标对其是普通野地，不标活动
		actType := 0
		if o.OrderType == 2 || o.OrderType == 3 {
			if act := h.ezfyActTargetType(o.TargetX, o.TargetY); act > 0 &&
				!h.playerOwnsActWildGeneral(uid, o.TargetX, o.TargetY) {
				actType = act
			}
		}
		// ★ 2026-09-28 出发地(军队所属城)：军情 → 驻军/军队动态 顶部显示「起点：城名(城x,城y)」
		fromName, fromX, fromY := "", 0, 0
		if cp := fromCity[int64(o.CityId)]; cp != nil {
			fromName, fromX, fromY = cp.Name, cp.X, cp.Y
		}
		views = append(views, gin.H{
			"id": o.ID, "order_type": o.OrderType, "type_name": ezfyOrderTypeName(o.OrderType),
			"target_type": o.TargetType, "target_name": h.ezfyTargetName(o),
			"target_x": o.TargetX, "target_y": o.TargetY, "act_type": actType,
			"from_city": fromName, "from_x": fromX, "from_y": fromY,
			"status": o.Status, "status_name": statusName,
			"officer": o.Officer, "time_label": timeLabel, "time_text": timeText,
			"arrive_time": o.ArriveTime, "return_time": o.ReturnTime,
			"carry": c, "carry_total": c.total(), "carry_cap": h.ezfyCarryCap(o),
			// 采集实况（采集中部队才有）：每期产出 + 起算点，前端据此实时累加展示
			"gather": gather,
			// 指挥室：可指挥时前端显示 [指挥]
			"can_command":    o.Status == ezfyOrderStatusBattle,
			"battle_round":   battleRound,
			"battle_max":     ezfyBattleMaxRounds,
			"battle_left_ms": battleLeft,
			// ★ 2026-09-30 行军计谋使用标记（与出征队列 /orders 同口径）：
			//   神兵天降=去程(出征中)减80%  战略转移=回程(返回中)减360分钟，每支部队各限一次
			"scheme_fast": o.SchemeUsed & 1,
			"scheme_back": (o.SchemeUsed >> 1) & 1,
		})
	}

	// ★ 防守方视角（2026-09-23 用户要求「敌人打自己，自己也能指挥」）：
	//   战场/订单属于**攻方**，上面的军队动态按 user_id 查不到守方要防守的这场战斗。
	//   这里单独把「正在被攻打(def_user_id = 我方)」的战场拼进列表，让守方也有 [指挥] 入口。
	var defBattles []model.EzfyBattle
	// ★ 2026-09-24 修复「被攻击的动态打完了还一直显示」：只展示订单仍处于
	//   「战斗中(5)」的战场。订单已结算(征服/返航)但战场行没更新(历史 bug 留下的
	//   僵尸行)一律不再展示；这类行由 ezfyBattleTick 自愈 + 本次线上数据修复清理。
	dbq := h.DB.Where("def_user_id = ? AND status = 1 AND order_id IN (SELECT id FROM ezfy_order WHERE status = ?)",
		uid, ezfyOrderStatusBattle)
	// ★ 2026-10-01 防守战场按「被打的城市」过滤（battle.target_id = 守方城市 id）
	if cityId > 0 {
		dbq = dbq.Where("target_id = ?", cityId)
	}
	dbq.Find(&defBattles)
	// ★★ 2026-10-05 修复「守方指挥结束了还显示指挥 / 指挥结束有延迟」：
	//
	//	战场与订单都属于**攻方**，而 `processOrders(uid)` 只会推进「自己发起」的订单 ——
	//	守方无论怎么轮询都推不动这场战斗，只能干等攻方上线或后台 ticker，
	//	表现就是「守方点完指挥，指挥一直不结束」（攻方离线时能卡很久）。
	//	这里把「打我方城市」的战场也 tick 一遍：回合到点就推进，打完了就地收尾。
	//	⚠️ 安全：`processArrive` 内部用 CAS 抢占结算权（status 0/5 → 98），
	//	  与攻方入口并发到达也不会重复结算战报/掠夺。
	for i := range defBattles {
		b := &defBattles[i]
		if _, done := h.ezfyBattleTick(b, now); !done {
			continue
		}
		h.ezfyBattleFinishToOrder(b, now)
		// 顺手把攻方那条订单也结算掉，别让部队卡在「已打完但没结算」
		var atkOrder model.EzfyOrder
		if err := h.DB.First(&atkOrder, b.OrderId).Error; err == nil {
			h.processArrive(b.UserID, &atkOrder, now)
		}
	}
	// tick 后已经打完的战场（ezfyBattleTick 会把行状态置 2）不再展示
	if len(defBattles) > 0 {
		live := defBattles[:0]
		for i := range defBattles {
			if defBattles[i].Status == 1 {
				live = append(live, defBattles[i])
			}
		}
		defBattles = live
	}
	for _, b := range defBattles {
		// 来袭敌军来源：攻击方城市（查不到就兜底显示玩家 uID）
		atkName := "玩家" + strconv.FormatUint(uint64(b.UserID), 10)
		atkX, atkY := b.TargetX, b.TargetY
		var atkCity model.EzfyCity
		if err := h.DB.Where("user_id = ?", b.UserID).Order("id ASC").First(&atkCity).Error; err == nil {
			atkName = atkCity.Name
			atkX, atkY = atkCity.X, atkCity.Y
		}
		left := b.RoundStart + ezfyBattleRoundMs - now
		if left < 0 {
			left = 0
		}
		views = append(views, gin.H{
			"id": b.OrderId, "order_type": 0, "type_name": "防御",
			"target_type": b.TargetType, "target_name": atkName,
			"target_x": atkX, "target_y": atkY,
			"status": ezfyOrderStatusBattle, "status_name": "战斗中",
			"officer": "", "time_label": "本回合剩余", "time_text": ezfyDurationText(left / 1000),
			"arrive_time": 0, "return_time": 0,
			"carry": nil, "carry_total": 0, "carry_cap": 0,
			"can_command":    true,
			"battle_round":   b.Round,
			"battle_max":     ezfyBattleMaxRounds,
			"battle_left_ms": left,
			"is_defend":      true,
		})
	}
	resp.OK(c, gin.H{"dynamics": views, "count": len(views)})
}

// ReportView GET /games/ezfy/reports/:id
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

// ReportDelete POST /games/ezfy/reports/:id/delete
// ★ 用户要求「战斗报告展开后可删除，不需要二次确认」：只允许删除自己名下的一条战报。
func (h *EzfyHandler) ReportDelete(c *gin.Context) {
	uid := middleware.GetUID(c)
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	res := h.DB.Where("id = ? AND user_id = ?", id, uid).Delete(&model.EzfyReport{})
	if res.RowsAffected == 0 {
		resp.NotFound(c, "战报不存在")
		return
	}
	resp.OK(c, gin.H{"msg": "已删除"})
}

// ReportClear POST /games/ezfy/reports/clear
//
// ★ 2026-09-26 用户要求「战报查询 [查询] 右边加个 [一键删除]，物理删除吧节约服务器资源」：
//
//	只清**自己名下**的战报（`WHERE user_id = ?`），GORM 走 DELETE 真删行、不软删。
//	⚠️ 不带 user_id 的批量 Delete 会清全表 —— 这里必须带，且只认 middleware 里的 uid。
//	⚠️ 路由用 `/reports/clear` 而不是 `/reports/:id/delete` 的同级形式：
//	   路径段数不同（2 段 vs 3 段），Gin 不会和 `:id` 冲突。
//
// ★ 2026-10-01 军情按当前城过滤：city_id>0 时只删**当前城市**的战报（口径与 Reports 列表一致）。
func (h *EzfyHandler) ReportClear(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		CityId int64 `json:"city_id"`
	}
	c.ShouldBindJSON(&req)
	q := h.DB.Where("user_id = ?", uid)
	if req.CityId > 0 {
		// ★ 2026-10-01 修复：老战报 order_id 全为 0，不能靠 order_id 关联城市，
		//   与 Reports/ezfyReportCounts 同口径按标题坐标反查该城出征订单
		q = q.Where(ezfyCityReportCond(req.CityId))
	}
	res := q.Delete(&model.EzfyReport{})
	resp.OK(c, gin.H{"msg": fmt.Sprintf("已删除 %d 条战报", res.RowsAffected), "deleted": res.RowsAffected})
}
