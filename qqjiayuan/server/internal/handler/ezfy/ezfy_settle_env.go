package ezfy

import (
	"strconv"

	"qqjiayuan/server/internal/model"
)

// 二战风云 —— 批量采集结算的「请求级缓存」。
//
// ★★ 2026-10-13 性能（用户反馈「一键采集 / 一键收获 卡顿」）：
//
//	「一键收获」原来对**每一支**采集部队都独立走一遍完整结算，每支要打 10+ 次跨库往返：
//	  查野地(1) + 原子认领(1) + 查城市(1) + 数宝物库存(1) + 插装备(N) + 写订单(1)
//	  + 插战报(1) + 战报补 city_id 回查(1) + 入城前读城资源(1) + 写城资源(1) + 读城资源(1)
//	  + 清空负重写订单(1) …
//	玩家带 20 支采集队 = 200+ 次往返；线上是 1核1G + 跨 WAN 的 MySQL，每次往返几十毫秒，
//	体感就是「点了按钮一直转圈」。
//
//	本结构把「整批共享、只读」的数据一次性预取进内存，把「可攒批」的写操作
//	（战报 / 装备 / 城市资源入账 / 负重清空）攒到最后一次性提交，
//	每支的往返降到 1~2 次（只剩必须逐支的「原子认领」与「订单结果写回」）。
//
// ★ 语义保证（与原来「逐支独立处理」完全等价）：
//   - 只读缓存：野地、城市一次 `WHERE id IN (...)` 取回，取不到就回落单条查询；
//   - 宝物剩余量：首次向 DB 取一次，之后本地递减（原来每支都 COUNT 一次，结果相同）；
//   - 城市资源入账：先按城合并；**若合并后会被资源最大值截断**，则该城退回逐支入库
//     （逐支「前后差值」口径与原来一字不差），避免「资源已满」时把本该留在负重里的量清掉。
type ezfySettleEnv struct {
	wilds  map[int64]*model.EzfyWildland
	cities map[int64]*model.EzfyCity
	// ★ 2026-10-21 采集新公式：按出发城市预取「该城名下所有野地 → 各资源总等级」，
	//   供 R（野地加成%）使用。批量结算/军队动态一次 IN 查询，避免逐支扫野地表。
	cityWildLevels map[int64]map[string]int

	reports []model.EzfyReport
	equips  []model.EzfyEquipment

	// 待入城资源。按支记录（不是直接按城合并），
	// 以便在「该城会被资源上限截断」时回退到逐支口径。
	carryIns  []ezfySettleIn // 负重取回入城（计入「本次入账」播报）
	directIns []ezfySettleIn // 负重装不下、直接入城的部分（不计入播报，与原来一致）

	// techsByCity 本批缓存的「**该城可享受的**科技等级表」（key = 出发城ID）。
	// ★ 原每支要查 2 条 SQL 才能拿到（城的 user_id + 科技等级）；
	// ★★ 2026-10-29 用户规则「没有科研中心建筑的城市不享受科技加成」→ 缓存必须**按城**
	//   （同一批里可能有多个出发城，各自的科研中心情况不同），用 techMap(cityId) 取一次。
	techsByCity map[uint]map[int]int

	// officers 本批缓存的军官（key = 城ID|军官名）。
	// ★ 原 `officerByName` 每支结算都要查一次（算采集后勤加成），批量时纯属重复。
	officers map[string]*model.EzfyOfficer

	treasureLeft     int
	treasureLeftInit bool
}

// ezfySettleIn 一笔待入城记录。
type ezfySettleIn struct {
	orderID uint
	cityID  int64
	c       ezfyCarry
}

// batchWildlands 批量取野地（id → 野地）。
//
// ★ 2026-10-13 性能：一键采集用它替代「每支一条 SELECT」（20 支 = 20 次跨库往返）。
func (h *EzfyHandler) batchWildlands(ids []int64) map[int64]*model.EzfyWildland {
	out := map[int64]*model.EzfyWildland{}
	if len(ids) == 0 {
		return out
	}
	var list []model.EzfyWildland
	h.DB.Where("id IN ?", ids).Find(&list)
	for i := range list {
		w := list[i]
		out[int64(w.ID)] = &w
	}
	return out
}

// batchCities 按城市 ID 批量取城（一次 IN 查询），供批量结算 / 多城资源满判定复用。
//
// ★ 2026-10-22 一键采集切到「全部城市」后，不能再像当前城那样只查一次资源是否满；
//
//	这里按订单涉及的城市一次取回，避免每支部队 / 每座城各打一条 SQL。
func (h *EzfyHandler) batchCities(ids []int64) map[int64]*model.EzfyCity {
	out := map[int64]*model.EzfyCity{}
	if len(ids) == 0 {
		return out
	}
	uniq := make([]int64, 0, len(ids))
	seen := map[int64]bool{}
	for _, id := range ids {
		if id > 0 && !seen[id] {
			seen[id] = true
			uniq = append(uniq, id)
		}
	}
	if len(uniq) == 0 {
		return out
	}
	var list []model.EzfyCity
	h.DB.Where("id IN ?", uniq).Find(&list)
	for i := range list {
		c := list[i]
		out[int64(c.ID)] = &c
	}
	return out
}

// ezfyCityAtResMax 该城五项资源是否已全部到达「资源最大值」（= 满了，再采集也入不了库）。
//
// ★ 与 `ezfyAtResMax` 同口径，但吃**已取好的城市对象**（批量结算已预取），省一条查询。
func (h *EzfyHandler) ezfyCityAtResMax(c *model.EzfyCity) bool {
	if c == nil {
		return false
	}
	// 五项全满才算满 —— 采集产出五项都有，只要还有一项没满，收获就仍有意义。
	return c.Food >= ezfyResMaxOf("food") &&
		c.Steel >= ezfyResMaxOf("steel") &&
		c.Oil >= ezfyResMaxOf("oil") &&
		c.Rare >= ezfyResMaxOf("rare") &&
		c.Gold >= ezfyResMaxOf("gold")
}

// newSettleEnv 为一批订单预取共享数据（野地 + 城市），各一次 IN 查询。
func (h *EzfyHandler) newSettleEnv(orders []model.EzfyOrder) *ezfySettleEnv {
	e := &ezfySettleEnv{
		wilds:          map[int64]*model.EzfyWildland{},
		cities:         map[int64]*model.EzfyCity{},
		cityWildLevels: map[int64]map[string]int{},
		officers:       map[string]*model.EzfyOfficer{},
	}
	if len(orders) == 0 {
		return e
	}
	wildIDs := make([]int64, 0, len(orders))
	cityIDs := make([]int64, 0, len(orders))
	seenW, seenC := map[int64]bool{}, map[int64]bool{}
	for i := range orders {
		if id := orders[i].TargetId; id > 0 && !seenW[id] {
			seenW[id] = true
			wildIDs = append(wildIDs, id)
		}
		if id := int64(orders[i].CityId); id > 0 && !seenC[id] {
			seenC[id] = true
			cityIDs = append(cityIDs, id)
		}
	}
	if len(wildIDs) > 0 {
		var wls []model.EzfyWildland
		h.DB.Where("id IN ?", wildIDs).Find(&wls)
		for i := range wls {
			w := wls[i]
			e.wilds[int64(w.ID)] = &w
		}
	}
	if len(cityIDs) > 0 {
		e.cities = h.batchCities(cityIDs)
		// ★ 2026-10-21 采集新公式的野地加成：一次取回这批城市名下所有野地，
		//   按「城市 → 资源 → 总等级」预先汇总（同城多支采集队共享，避免 N+1）。
		var cityWilds []model.EzfyWildland
		h.DB.Where("city_id IN ?", cityIDs).Find(&cityWilds)
		byCity := map[int64][]model.EzfyWildland{}
		for i := range cityWilds {
			w := cityWilds[i]
			byCity[w.CityId] = append(byCity[w.CityId], w)
		}
		for cid, list := range byCity {
			e.cityWildLevels[cid] = ezfyWildResLevels(list)
		}
	}
	return e
}

// wildland 取缓存的野地（未预取到返回 nil）。
func (e *ezfySettleEnv) wildland(id int64) *model.EzfyWildland {
	if e == nil || e.wilds == nil {
		return nil
	}
	return e.wilds[id]
}

// city 取缓存的城市（未预取到返回 nil）。
func (e *ezfySettleEnv) city(id int64) *model.EzfyCity {
	if e == nil || e.cities == nil {
		return nil
	}
	return e.cities[id]
}

// wildLevels 取该城预汇总的「资源 → 野地总等级」；未预取到返回 nil（调用方回落单次查询）。
func (e *ezfySettleEnv) wildLevels(cityID int64) map[string]int {
	if e == nil || e.cityWildLevels == nil {
		return nil
	}
	return e.cityWildLevels[cityID]
}

// treasureLeft 本批还剩几件宝物可放。首次向 DB 取一次，之后本地递减。
//
// ★ 与原 `ezfyTreasureLeft` 等价：本批新增的宝物只增在**本请求**里，
// 逐支 COUNT 与「首查一次 + 本地递减」结果相同。
func (h *EzfyHandler) envTreasureLeft(e *ezfySettleEnv, uid uint) int {
	if e == nil {
		return h.ezfyTreasureLeft(uid)
	}
	if !e.treasureLeftInit {
		e.treasureLeft = h.ezfyTreasureLeft(uid)
		e.treasureLeftInit = true
	}
	return e.treasureLeft
}

// envTechs 本批缓存的「该出发城可享受的科技等级表」（按 cityId 只查一次）。
//
// ★★ 2026-10-13 性能：原 `ezfyCarryCap(order)` → `ezfyCarryCapOf` → `techMap(cityId)`
// 每次都要打 2 条跨库查询（查城的 user_id + 查该玩家科技），而 settleDispatch 里
// 一支采集部队就要调 3 次负重上限（算负重余量 / 判负重是否满 / 拼战报文案）→ 6 条往返/支。
//
// ★★ 2026-10-29 用户规则：**没有「科研中心」建筑的城市不享受科技加成** → 缓存按**城**分，
// 直接复用 techMap(cityId)（它已把「本城有无科研中心」并进同一条 SQL，条数不变）。
func (h *EzfyHandler) envTechs(e *ezfySettleEnv, cityId uint) map[int]int {
	if e == nil {
		return nil
	}
	if e.techsByCity == nil {
		e.techsByCity = map[uint]map[int]int{}
	}
	if m, ok := e.techsByCity[cityId]; ok {
		return m
	}
	m := h.techMap(cityId)
	e.techsByCity[cityId] = m
	return m
}

// envCarryCap 本批缓存的负重上限（env == nil 时回落原实现）。
func (h *EzfyHandler) envCarryCap(e *ezfySettleEnv, order *model.EzfyOrder, uid uint) int64 {
	if e == nil {
		return h.ezfyCarryCap(order)
	}
	return h.ezfyCarryCapOf(parseGroups(order.Troops), uint(order.CityId), h.envTechs(e, uint(order.CityId)))
}

// envCarryFull 该订单部队的「待带回」是否已装满负重（env == nil 时回落原实现）。
func (h *EzfyHandler) envCarryFull(e *ezfySettleEnv, order *model.EzfyOrder, uid uint) bool {
	capTotal := h.envCarryCap(e, order, uid)
	if capTotal <= 0 {
		return false
	}
	return parseCarry(order.Carry).total() >= capTotal
}

// envOfficer 按 (城, 军官名) 取军官，批量结算时缓存。
//
// ★ 原 `officerByName` 每支结算都要查一次（算采集后勤加成），批量收获时纯属重复往返。
// 命中缓存返回同一指针；查不到也缓存（负缓存），避免同名反复查。
func (h *EzfyHandler) envOfficer(e *ezfySettleEnv, cityId uint, name string) *model.EzfyOfficer {
	if e == nil {
		return h.officerByName(cityId, name)
	}
	if name == "" {
		return nil
	}
	key := strconv.FormatUint(uint64(cityId), 10) + "|" + name
	if o, ok := e.officers[key]; ok {
		return o
	}
	o := h.officerByName(cityId, name)
	e.officers[key] = o
	return o
}

// envAddEquip 攒一件待落库的装备，并把本批宝物余量减一。
func (h *EzfyHandler) envAddEquip(e *ezfySettleEnv, city *model.EzfyCity, cfg *model.EzfyCfgEquipment) {
	eq := h.buildEquipment(city, cfg)
	if e == nil {
		h.DB.Create(&eq)
		return
	}
	e.equips = append(e.equips, eq)
	e.treasureLeft--
}

// envReport 攒一条待落库的战报。
func (h *EzfyHandler) envReport(e *ezfySettleEnv, uid uint, reportType int, title, content string, detailAndOrder ...interface{}) {
	r := h.newReport(uid, reportType, title, content, detailAndOrder...)
	if e == nil {
		h.DB.Create(&r)
		return
	}
	e.reports = append(e.reports, r)
}

// envHarvest 采集产出入城。
//
// env == nil → 直接入库（单支接口的原行为）；
// env != nil → 只记待入账，由 envFlush 统一提交（返回名义入账量）。
func (h *EzfyHandler) envHarvest(e *ezfySettleEnv, orderID uint, cityID int64, food, steel, oil, rare, gold int64, direct bool) int64 {
	amount := food + steel + oil + rare + gold
	if amount <= 0 || cityID <= 0 {
		return 0
	}
	if e == nil {
		return h.harvestToCity(cityID, food, steel, oil, rare, gold)
	}
	in := ezfySettleIn{orderID: orderID, cityID: cityID,
		c: ezfyCarry{Food: food, Steel: steel, Oil: oil, Rare: rare, Gold: gold}}
	if direct {
		e.directIns = append(e.directIns, in)
	} else {
		e.carryIns = append(e.carryIns, in)
	}
	return amount
}

// envHarvestCarry 把部队「待带回负重」记为待入城，并在内存里清空负重。
//
// ★ 与原 `harvestCarryToCity` 的差别只有一点：落库（carry=” 的 UPDATE）推迟到
// envFlush 之后由调用方**批量**执行 —— 原实现每支都要读两次城资源 + 写一次城 + 写一次订单。
func (h *EzfyHandler) envHarvestCarry(e *ezfySettleEnv, order *model.EzfyOrder) {
	c := parseCarry(order.Carry)
	if c.total() <= 0 {
		return
	}
	if e == nil {
		h.harvestCarryToCity(order)
		return
	}
	e.carryIns = append(e.carryIns, ezfySettleIn{orderID: order.ID, cityID: int64(order.CityId), c: c})
	// 内存里清空；真正落库由调用方的批量 UPDATE 负责。
	order.Carry = carryJSON(ezfyCarry{})
}

// ezfyHarvestWouldCap 这批资源入城后是否会被「资源最大值」截断。
//
// 与 `ezfyResAddExpr` 严格同口径：结果 = GREATEST(现值, LEAST(上限, 现值+增量))，
// 只要**任一项**「现值 + 增量 > 上限」就会截断（含老数据已超上限、增量被吞的情况）。
// 截断时不能走合并入库的快路径 —— 那会把「本该留在负重里」的资源一起清掉。
func (h *EzfyHandler) ezfyHarvestWouldCap(city *model.EzfyCity, c ezfyCarry) bool {
	if city == nil {
		return true
	}
	return city.Food+c.Food > ezfyResMaxOf("food") ||
		city.Steel+c.Steel > ezfyResMaxOf("steel") ||
		city.Oil+c.Oil > ezfyResMaxOf("oil") ||
		city.Rare+c.Rare > ezfyResMaxOf("rare") ||
		city.Gold+c.Gold > ezfyResMaxOf("gold")
}

// bumpCachedCityRes 把一次入库同步进缓存的城市对象（与 ezfyResAddExpr 严格同口径）。
//
// ★ 为什么需要：同一座城可能既有「负重取回入城」又有「负重装不下直入城」两批，
// 前一批入库后城市已接近上限，后一批的「是否会被截断」判断必须用**更新后**的值，
// 否则会误判成「不会截断」而走合并快路径，把本该留在负重里的资源一起清掉。
func bumpCachedCityRes(c *model.EzfyCity, food, steel, oil, rare, gold int64) {
	if c == nil {
		return
	}
	c.Food = ezfyAddResMax("food", c.Food, food)
	c.Steel = ezfyAddResMax("steel", c.Steel, steel)
	c.Oil = ezfyAddResMax("oil", c.Oil, oil)
	c.Rare = ezfyAddResMax("rare", c.Rare, rare)
	c.Gold = ezfyAddResMax("gold", c.Gold, gold)
}

// envFlush 提交本批攒下的写入：战报 / 装备批量落库，采集资源按城入库。
//
// 返回 (实际入账总量, 需要清空负重(carry)的订单 ID 列表)。
//
// ★ 入城分两条路径，保证与「逐支 harvestCarryToCity」完全等价：
//   - 快路径：该城这批入账**不会被资源最大值截断** → 合并成一次 UPDATE
//     （原 N 次「读城资源 + 写城资源 + 读城资源」）；
//   - 慢路径：会被截断 → 退回逐支入库 + 逐支按差值判断是否清空负重（与原来一字不差）。
func (h *EzfyHandler) envFlush(e *ezfySettleEnv) (int64, []uint) {
	if e == nil {
		return 0, nil
	}
	if len(e.reports) > 0 {
		h.DB.CreateInBatches(e.reports, 100)
		e.reports = nil
	}
	if len(e.equips) > 0 {
		h.DB.CreateInBatches(e.equips, 100)
		e.equips = nil
	}
	if len(e.carryIns) == 0 && len(e.directIns) == 0 {
		return 0, nil
	}

	var gained int64
	cleared := []uint{}
	clearedSet := map[uint]bool{}
	markCleared := func(id uint) {
		if id > 0 && !clearedSet[id] {
			clearedSet[id] = true
			cleared = append(cleared, id)
		}
	}

	apply := func(list []ezfySettleIn, countGain bool) {
		// 按城汇总
		byCity := map[int64]ezfyCarry{}
		order := []int64{}
		perCity := map[int64][]ezfySettleIn{}
		for _, in := range list {
			if in.cityID <= 0 {
				continue
			}
			if _, ok := byCity[in.cityID]; !ok {
				order = append(order, in.cityID)
			}
			c := byCity[in.cityID]
			c.Food += in.c.Food
			c.Steel += in.c.Steel
			c.Oil += in.c.Oil
			c.Rare += in.c.Rare
			c.Gold += in.c.Gold
			byCity[in.cityID] = c
			perCity[in.cityID] = append(perCity[in.cityID], in)
		}
		for _, cityID := range order {
			agg := byCity[cityID]
			if agg.total() <= 0 {
				continue
			}
			if !h.ezfyHarvestWouldCap(e.cities[cityID], agg) {
				// 快路径：整城一次入库
				got := h.harvestToCity(cityID, agg.Food, agg.Steel, agg.Oil, agg.Rare, agg.Gold)
				// 同步缓存，供同一座城的后续批次（负重直入城）继续准确判断
				bumpCachedCityRes(e.cities[cityID], agg.Food, agg.Steel, agg.Oil, agg.Rare, agg.Gold)
				if countGain {
					gained += got
				}
				if got > 0 {
					for _, in := range perCity[cityID] {
						markCleared(in.orderID)
					}
				}
				continue
			}
			// 慢路径：逐支入库（与原实现同口径）
			for _, in := range perCity[cityID] {
				got := h.harvestToCity(cityID, in.c.Food, in.c.Steel, in.c.Oil, in.c.Rare, in.c.Gold)
				bumpCachedCityRes(e.cities[cityID], in.c.Food, in.c.Steel, in.c.Oil, in.c.Rare, in.c.Gold)
				if countGain {
					gained += got
				}
				if got > 0 {
					markCleared(in.orderID)
				}
			}
		}
	}
	apply(e.carryIns, true)   // 负重取回：计入播报
	apply(e.directIns, false) // 负重装不下直入城：不计入播报（与原来一致）

	e.carryIns = nil
	e.directIns = nil
	return gained, cleared
}
