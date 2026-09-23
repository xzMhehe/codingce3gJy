package handler

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
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

func (h *EzfyHandler) Buildings(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	city := h.getOrCreateCity(uid)
	h.refreshCity(uid, &city)
	list := h.buildingList(city.ID)
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
			view["max_level"] = h.buildingMaxLevel(city.ID, b.BuildingId)
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
	// 可建造池: 复刻原版 BuildingController.buildList
	pool := h.buildPool(&city, list)
	// ★ 第九轮：军事区 / 资源区数量上限分开（各 33，管理端可维护）
	mil, res := h.areaCounts(city.ID)
	lim := ezfyLimit()
	resp.OK(c, gin.H{
		"buildings": views, "pool": pool,
		"area_count": mil + res, "area_cap": lim.MilitaryMax + lim.ResourceMax,
		"military_count": mil, "military_cap": lim.MilitaryMax,
		"resource_count": res, "resource_cap": lim.ResourceMax,
	})
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
	// ★ 军事区/资源区各自有**硬上限**（默认各 33，管理端可维护）：
	//   即使军工厂/民居设了「不限数量」，也不能超过所属区域的总数上限（用户规则）。
	//   pool 这里必须和 buildBuilding 一致地按区域卡，否则会出现「队列里能点、一建就报已达上限」。
	mil, res := h.areaCounts(city.ID)
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

func (h *EzfyHandler) MaxLevel(c *gin.Context) {
	uid := middleware.GetUID(c)
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
	h.done(c, h.maxLevelBuilding(city, req.RecordId), "建筑已升到最高级")
}

func (h *EzfyHandler) DeleteBuilding(c *gin.Context) {
	uid := middleware.GetUID(c)
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

func (h *EzfyHandler) Troops(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	profile := h.ensureProfile(uid)
	city := h.getOrCreateCity(uid)
	h.refreshCity(uid, &city)
	camp := profile.Camp

	troopViews := []gin.H{}
	for tid, count := range h.troopMap(city.ID) {
		cfg := ezfyCfg.troop(tid)
		if cfg == nil {
			continue
		}
		troopViews = append(troopViews, gin.H{"troop_id": tid, "name": ezfyCfg.troopName(tid, camp),
			"count": count, "type": cfg.Type})
	}
	queues := []gin.H{}
	var qs []model.EzfyTrainQueue
	h.DB.Where("city_id = ? AND status = 0", city.ID).Order("start_time ASC").Find(&qs)
	for _, q := range qs {
		queues = append(queues, gin.H{"id": q.ID, "troop_id": q.TroopId,
			"name": ezfyCfg.troopName(q.TroopId, camp), "count": q.Count, "end_time": q.EndTime})
	}
	var wounded, deserters []model.EzfyWounded
	h.DB.Where("city_id = ? AND type = 0", city.ID).Order("troop_id ASC").Find(&wounded)
	h.DB.Where("city_id = ? AND type = 1", city.ID).Order("troop_id ASC").Find(&deserters)
	// ★ 2026-09-23：超过「伤兵存活天数」还没救治的伤兵直接消失（用户要求 5 天）
	wounded = h.filterExpiredWounded(wounded)
	deserters = h.filterExpiredWounded(deserters)
	woundViews := []gin.H{}
	for _, w := range append(wounded, deserters...) {
		woundViews = append(woundViews, gin.H{"id": w.ID, "troop_id": w.TroopId,
			"name": ezfyCfg.troopName(w.TroopId, camp), "type": w.Type, "count": w.Count,
			// ★ 用户要求「恢复伤兵需要黄金」：把单价一起下发，前端在[恢复]旁边显示要花多少钱
			"heal_gold": ezfyWoundHealGoldPer(w.TroopId)})
	}
	// ★ 占用人口 = 只有「训练队列里还没出厂」的新兵占（用户规则：部队不占人口位置）。
	//   统一走 troopPop，别再在这里手写一份，否则又会出现两处口径不一致的 bug。
	popUsed := h.troopPop(city.ID)
	// 兵种配置一览
	cfgViews := []gin.H{}
	var allTroops []model.EzfyCfgTroop
	h.DB.Order("id ASC").Find(&allTroops)
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
	wallLevel := h.buildingLevel(city.ID, 7)
	defSpace := int64(0)
	if wall := ezfyCfg.buildingLevel(7, wallLevel); wall != nil {
		defSpace = wall.Capacity
	}
	defUsed := h.defenceSpaceUsed(city.ID)
	// 军工厂座数(复刻 createTroop.html 的「全部工厂 / 仅此工厂」)
	factoryCount := 0
	for _, b := range h.buildingList(city.ID) {
		if b.BuildingId == ezfyFactoryBuildingID {
			factoryCount++
		}
	}
	resp.OK(c, gin.H{
		"city": city, "troops": troopViews, "queues": queues, "wounded": woundViews,
		"pop": city.Pop, "pop_used": popUsed, "cfgs": cfgViews,
		"wall_level":         wallLevel,
		"train_discount":     discount,
		"defence_space":      defSpace,
		"defence_space_used": defUsed,
		"factory_total":      h.buildingTotalLevel(city.ID, ezfyFactoryBuildingID),
		"factory_count":      factoryCount,
	})
}

// DismissDefence POST /games/ezfy/troops/dismiss —— 拆除城防设施
// 复刻 troopDefence.html 每行的 [拆除]（原版 Java 无对应接口, 模板里是失效的旧链接）
func (h *EzfyHandler) DismissDefence(c *gin.Context) {
	uid := middleware.GetUID(c)
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

// ezfySpeedGoldPerSec 训练一键加速收费: 每剩余 1 秒 10 黄金
const ezfySpeedGoldPerSec = 10

// SpeedTrainAll POST /games/ezfy/troops/speed-all —— 训练一键加速
// all_city=true 时对所有城市生效(复刻 militaryIndex.html 底部的两个按钮)
func (h *EzfyHandler) SpeedTrainAll(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		CityId  int64 `json:"city_id"`
		AllCity bool  `json:"all_city"`
	}
	_ = c.ShouldBindJSON(&req)
	h.cfgs()
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
	cost := totalSec * ezfySpeedGoldPerSec
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
	city := h.getOrCreateCity(uid)
	h.refreshCity(uid, &city)
	// ★ 第九轮：科技所有城池公用 —— 等级取科技城，科研中心取玩家所有城的最高等级
	techCity := h.techCityId(city.ID)
	techMap := h.techMap(city.ID)
	academy := h.maxAcademyLevel(uid)
	var all []model.EzfyCfgTech
	h.DB.Order("id ASC").Find(&all)
	views := []gin.H{}
	for _, t := range all {
		level := techMap[t.ID]
		var rec model.EzfyCityTech
		if err := h.DB.Where("city_id = ? AND tech_id = ? AND status = 1", techCity, t.ID).First(&rec).Error; err == nil {
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
	resp.OK(c, gin.H{"techs": views, "academy": academy})
}

func (h *EzfyHandler) Research(c *gin.Context) {
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
	h.done(c, h.researchTech(city, req.TechId), "科技研究已开始")
}

func (h *EzfyHandler) SpeedTech(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		CityId  int64 `json:"city_id"`
		Minutes int64 `json:"minutes"`
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
	h.done(c, h.speedUpTech(city, req.Minutes), "科技研究已加速完成")
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
	h.done(c, h.cancelTech(city, req.TechId), "已取消研究, 消耗已全额退还")
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
	views := []gin.H{}
	for _, cp := range corps {
		score := 0
		var members []model.EzfyCorpsMember
		h.DB.Where("corps_id = ?", cp.ID).Find(&members)
		for _, m := range members {
			p := h.ensureProfile(m.UserId)
			score += p.Prestige
		}
		leaderName := "未知"
		lp := h.ensureProfile(cp.LeaderUserId)
		if lp.UserID == cp.LeaderUserId {
			leaderName = lp.Nickname
		}
		views = append(views, gin.H{"id": cp.ID, "name": cp.Name, "notice": cp.Notice,
			"member_count": counts[int64(cp.ID)], "leader": leaderName, "battle_score": score,
			"camp": profile.Camp})
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
	var exist model.EzfyCorpsMember
	if err := h.DB.Where("user_id = ?", uid).First(&exist).Error; err == nil {
		resp.ParamError(c, "你已在军团中")
		return
	}
	var cp model.EzfyCorps
	if err := h.DB.First(&cp, req.CorpsId).Error; err != nil {
		resp.ParamError(c, "军团不存在")
		return
	}
	// 联络中心: 1 级才能加入联盟, 且受人数上限限制
	if h.liaisonLevel(uid) < 1 {
		resp.ParamError(c, "需要 1 级联络中心才能加入联盟")
		return
	}
	var memberCount int64
	h.DB.Model(&model.EzfyCorpsMember{}).Where("corps_id = ?", cp.ID).Count(&memberCount)
	if cap := h.corpsMemberCap(cp.ID); int(memberCount) >= cap {
		resp.ParamError(c, "该联盟人数已满("+strconv.Itoa(int(memberCount))+"/"+strconv.Itoa(cap)+")")
		return
	}
	h.DB.Create(&model.EzfyCorpsMember{CorpsId: cp.ID, UserId: uid, IsLeader: 0, Title: "成员"})
	h.DB.Model(&model.EzfyCorps{}).Where("id = ?", cp.ID).Update("member_count", cp.MemberCount+1)
	resp.OK(c, gin.H{"msg": "加入军团成功"})
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
	resp.OK(c, gin.H{"status": status, "text": text, "war_require": ezfyWarRequireOn()})
}

// ============ 排行榜 ============

func (h *EzfyHandler) Rank(c *gin.Context) {
	h.cfgs()
	// 声望榜
	var profiles []model.EzfyProfile
	h.DB.Order("prestige DESC").Limit(20).Find(&profiles)
	prestigeRank := []gin.H{}
	for i, p := range profiles {
		prestigeRank = append(prestigeRank, gin.H{"rank": i + 1, "name": p.Nickname, "user_id": p.UserID,
			"prestige": p.Prestige, "rank_name": ezfyRankName(p.Prestige)})
	}
	// 兵力榜(不含城防)
	var troops []model.EzfyCityTroop
	h.DB.Where("troop_id < 17").Find(&troops)
	sumByCity := map[int64]int64{}
	for _, t := range troops {
		sumByCity[t.CityId] += t.Count
	}
	type kv struct {
		k int64
		v int64
	}
	arr := []kv{}
	for k, v := range sumByCity {
		arr = append(arr, kv{k, v})
	}
	for i := 0; i < len(arr); i++ {
		for j := i + 1; j < len(arr); j++ {
			if arr[j].v > arr[i].v {
				arr[i], arr[j] = arr[j], arr[i]
			}
		}
	}
	troopRank := []gin.H{}
	for i, e := range arr {
		if i >= 20 {
			break
		}
		var city model.EzfyCity
		if err := h.DB.First(&city, e.k).Error; err != nil {
			continue
		}
		p := h.ensureProfile(city.UserID)
		troopRank = append(troopRank, gin.H{"rank": i + 1, "city_name": city.Name,
			"role_name": p.Nickname, "user_id": city.UserID, "count": e.v})
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
	corpsRank := []gin.H{}
	for i, cp := range corps {
		score := 0
		var members []model.EzfyCorpsMember
		h.DB.Where("corps_id = ?", cp.ID).Find(&members)
		for _, m := range members {
			score += h.ensureProfile(m.UserId).Prestige
		}
		corpsRank = append(corpsRank, gin.H{"rank": i + 1, "name": cp.Name,
			"member_count": rankCounts[int64(cp.ID)], "battle_score": score})
	}
	// 军衔表（★ 含「可建城数」一列，与 model.EzfyCfgRank 一致）
	ranks := []gin.H{}
	for _, r := range ezfyCfg.rankList() {
		ranks = append(ranks, gin.H{
			"id": r.ID, "name": r.Name, "post": r.Post,
			"need": r.NeedPrestige, "city_max": r.CityMax,
		})
	}
	// 当前玩家的军衔与建城额度（军衔限制分城数量）
	uid := middleware.GetUID(c)
	me := h.ensureProfile(uid)
	var myCities int64
	h.DB.Model(&model.EzfyCity{}).Where("user_id = ?", uid).Count(&myCities)
	mine := gin.H{
		"rank_name": ezfyRankName(me.Prestige), "rank_post": ezfyRankPost(me.Prestige),
		"prestige": me.Prestige, "city_max": ezfyRankCityMax(me.Prestige), "city_count": myCities,
	}
	resp.OK(c, gin.H{"prestige": prestigeRank, "troops": troopRank, "corps": corpsRank,
		"ranks": ranks, "mine": mine})
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
	case 1, 2:
		return "资源道具"
	case 3, 4, 5:
		return "加速道具"
	case 6:
		return "建筑图纸"
	case 7, 8:
		return "增益道具"
	case 9, 10, 11, 12:
		return "军官道具"
	// ★ 19 = 军官升星卡（用户要求放到「军官道具」分类下）
	case 19:
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
		// ★ 单次购买数量上限（管理端「建筑上限配置」页维护，默认 9999）
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
	// ★ 单次购买数量上限（用户要求「原来卡控 1-99，改成可配置的，默认 1-9999」）
	//   上限读 ezfy_cfg_limit.mall_buy_max（管理端「建筑上限配置」页维护），默认 9999。
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
	for _, it := range items {
		if it.Count <= 0 {
			continue
		}
		cfg := ezfyCfg.item(it.CfgId)
		if cfg == nil {
			continue
		}
		views = append(views, gin.H{"cfg_id": it.CfgId, "count": it.Count,
			"name": cfg.Name, "item_type": cfg.ItemType, "description": cfg.Description, "param1": cfg.Param1,
			// ★ 背包也按分类展示（与商城同一套归类口径，见 ezfyItemCategory）
			"category": ezfyItemCategory(cfg)})
	}
	// 军官类道具的目标选择需要军官列表与技能列表
	city := h.getOrCreateCity(uid)
	officers := []gin.H{}
	for _, o := range h.officerList(city.ID) {
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
	resp.OK(c, gin.H{"items": views, "officers": officers, "skills": skills})
}

func (h *EzfyHandler) UseItem(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		CityId    int64 `json:"city_id"`
		CfgId     int   `json:"cfg_id"`
		Count     int   `json:"count"`
		OfficerId int64 `json:"officer_id"`
		SkillId   int   `json:"skill_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	h.cfgs()
	city := h.bodyCity(uid, req.CityId)
	msg := h.useItem(uid, city, req.CfgId, req.Count, req.OfficerId, req.SkillId)
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

func (h *EzfyHandler) Tasks(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	h.initTasks(uid)
	h.resetPeriodTasks(uid)
	today := time.Now().Format("2006-01-02")
	var cfgs []model.EzfyCfgTask
	h.DB.Where("status = 1").Order("sort_no ASC").Find(&cfgs)
	var mine []model.EzfyTask
	h.DB.Where("user_id = ?", uid).Find(&mine)
	myMap := map[int]model.EzfyTask{}
	for _, t := range mine {
		myMap[t.CfgId] = t
	}
	// 状态型任务同步
	for _, cfg := range cfgs {
		if !ezfyStateTaskTypes[cfg.TaskType] {
			continue
		}
		t, ok := myMap[cfg.ID]
		if !ok || t.Status == 2 {
			continue
		}
		cur := h.calcStateValue(uid, cfg.TaskType)
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
	var types []model.EzfyCfgTaskType
	h.DB.Order("sort_no ASC").Find(&types)
	typeMap := map[int]model.EzfyCfgTaskType{}
	for _, tp := range types {
		typeMap[tp.ID] = tp
	}
	groups := []gin.H{}
	byType := map[int][]gin.H{}
	order := []int{}
	for _, cfg := range cfgs {
		t, ok := myMap[cfg.ID]
		if !ok {
			continue
		}
		row := gin.H{"id": t.ID, "cfg_id": cfg.ID, "name": cfg.Name, "target": cfg.Target,
			"current": t.Current, "status": t.Status,
			"reward": gin.H{"gold": cfg.RewardGold, "food": cfg.RewardFood, "steel": cfg.RewardSteel,
				"oil": cfg.RewardOil, "rare": cfg.RewardRare, "prestige": cfg.RewardPrestige}}
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
	_ = today
	resp.OK(c, gin.H{"groups": groups})
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

func (h *EzfyHandler) giveResources(uid uint, food, steel, oil, rare, gold int64) {
	city := h.getOrCreateCity(uid)
	// ★ 2026-09-23：先做不会溢出的加法，再按仓储上限截断（原来的 city.Food+food 在极端值下会溢出翻负）
	city.Food = min64(ezfyClampRes(city.FoodCap), ezfyAddRes(city.Food, food))
	city.Steel = min64(ezfyClampRes(city.SteelCap), ezfyAddRes(city.Steel, steel))
	city.Oil = min64(ezfyClampRes(city.OilCap), ezfyAddRes(city.Oil, oil))
	city.Rare = min64(ezfyClampRes(city.RareCap), ezfyAddRes(city.Rare, rare))
	city.Gold = min64(ezfyClampRes(city.GoldCap), ezfyAddRes(city.Gold, gold))
	h.saveCityRes(&city)
}

// giveResNoCap 给「指定城市」加资源，**不按仓储上限截断**。
//
// 用于退还类操作（取消训练/取消研究），避免玩家觉得「退少了」。
// 负数是合法的，结果不会低于 0。
func (h *EzfyHandler) giveResNoCap(city *model.EzfyCity, food, steel, oil, rare, gold int64) {
	// ★ 2026-09-23：改用安全加法，结果恒在 [0, ezfyResSafeMax]，不会溢出翻负
	city.Food = ezfyAddRes(city.Food, food)
	city.Steel = ezfyAddRes(city.Steel, steel)
	city.Oil = ezfyAddRes(city.Oil, oil)
	city.Rare = ezfyAddRes(city.Rare, rare)
	city.Gold = ezfyAddRes(city.Gold, gold)
	h.saveCityRes(city)
}

// giveResourcesNoCap 管理端专用发放：**不按仓储上限截断**。
//
// 游戏内正常产出走 giveResources（超上限就丢掉溢出部分），但 GM 发资源如果也被
// 上限吃掉，就会出现「明明发了 100 万，玩家只收到 3 万」的困惑 —— 所以管理端
// 的「送资源 / 批量发放」一律走这里，允许资源超上限堆着。
// 负数是合法的（可用来扣减），但结果不会低于 0。
func (h *EzfyHandler) giveResourcesNoCap(uid uint, food, steel, oil, rare, gold int64) {
	city := h.getOrCreateCity(uid)
	// ★ 2026-09-23：安全加法（不按仓储上限截断，但仍受数值安全上限保护，不会溢出翻负）
	city.Food = ezfyAddRes(city.Food, food)
	city.Steel = ezfyAddRes(city.Steel, steel)
	city.Oil = ezfyAddRes(city.Oil, oil)
	city.Rare = ezfyAddRes(city.Rare, rare)
	city.Gold = ezfyAddRes(city.Gold, gold)
	h.saveCityRes(&city)
}

func (h *EzfyHandler) Welfare(c *gin.Context) {
	uid := middleware.GetUID(c)
	today := time.Now().Format("2006-01-02")
	yest := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	var signedToday, signedYest int64
	h.DB.Model(&model.EzfySign{}).Where("user_id = ? AND sign_date = ?", uid, today).Count(&signedToday)
	h.DB.Model(&model.EzfySign{}).Where("user_id = ? AND sign_date = ?", uid, yest).Count(&signedYest)
	signCount := 1
	if signedYest > 0 {
		var s model.EzfySign
		if err := h.DB.Where("user_id = ? AND sign_date = ?", uid, yest).First(&s).Error; err == nil {
			signCount = s.SignCount + 1
		}
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
	// 礼包状态
	// ★ 用户要求删掉「市政厅20/30/40级礼包」→ 这里只保留 新手/每周/市政厅10级
	gifts := gin.H{}
	for _, t := range []string{"newbie", "weekly", "level10"} {
		var n int64
		h.DB.Model(&model.EzfyGift{}).Where("user_id = ? AND gift_type = ?", uid, t).Count(&n)
		gifts[t] = n > 0
	}
	city := h.getOrCreateCity(uid)
	profile := h.ensureProfile(uid)
	resp.OK(c, gin.H{
		"signed_today": signedToday > 0, "sign_count": signCount,
		"rewards": rewards, "gifts": gifts, "city_level": city.CityLevel,
		"prestige": profile.Prestige, "rank_name": ezfyRankName(profile.Prestige),
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
	h.DB.Create(&model.EzfySign{UserId: uid, SignDate: today, SignCount: count})
	r := ezfySignRewards[(count-1)%7]
	// ★ 签到奖励不受仓储上限截断（用户要求：签到/任务/礼包领到的资源不能被上限吃掉）
	h.giveResourcesNoCap(uid, r[1], r[2], r[3], r[4], r[0])
	if r[5] > 0 {
		h.addPrestige(uid, int(r[5]))
	}
	resp.OK(c, gin.H{"msg": fmt.Sprintf("签到成功(连续%d天), 奖励已发放", count)})
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
		strings.HasPrefix(title, "征服报告"):
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
		return "守卫"
	case strings.HasPrefix(title, "军情警报"):
		return "警报"
	case strings.HasPrefix(title, "侦查报告"):
		return "侦察"
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
	}
	// 兜底：按 report_type
	switch reportType {
	case 1:
		return "侦察"
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

// Reports GET /games/ezfy/reports?category=1|2|3&word=xxx
func (h *EzfyHandler) Reports(c *gin.Context) {
	uid := middleware.GetUID(c)
	category, _ := strconv.Atoi(c.DefaultQuery("category", "0"))
	word := strings.TrimSpace(c.Query("word"))

	q := h.DB.Where("user_id = ?", uid)
	if word != "" {
		q = q.Where("title LIKE ?", "%"+word+"%")
	}
	var reports []model.EzfyReport
	q.Order("id DESC").Limit(200).Find(&reports)

	views := []gin.H{}
	counts := map[int]int{}
	for _, r := range reports {
		cat := ezfyReportCategory(r.Title)
		counts[cat]++
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
	city := h.getOrCreateCity(uid)
	resp.OK(c, gin.H{"reports": views, "counts": counts, "radar": h.buildingLevel(city.ID, 21)})
}

// ReportDynamics GET /games/ezfy/reports/dynamics
// 军队动态: 所有在外的部队(出征/采集/派遣/侦查/掠夺/运输/增援)
// 复刻 `二战风云/templates/report/index.html` 的「军队动态」区
func (h *EzfyHandler) ReportDynamics(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	// ★ 走统一懒结算：原来这里只处理了「抵达(0)」和「返航(2)」，
	//   漏掉了「驻守采集(1) 到点结算」，导致军队动态页看到的采集进度/待带回资源是旧的。
	h.processOrders(uid)
	now := time.Now().UnixMilli()
	var orders []model.EzfyOrder
	h.DB.Where("user_id = ? AND status IN (0,1,2,?,?)", uid, ezfyOrderStatusBattle, ezfyOrderStatusWaiting).
		Order("id DESC").Limit(100).Find(&orders)
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
	for i := range orders {
		o := &orders[i]
		timeLabel, timeText := "", ""
		statusName := ""
		switch o.Status {
		case 0:
			statusName = "出征"
			timeLabel = "抵达时间"
			timeText = ezfyDurationText((o.ArriveTime - now) / 1000)
		case 1:
			statusName = "采集"
			timeLabel = "已驻守"
			timeText = ezfyDurationText((now - o.ArriveTime) / 1000)
		case 2:
			statusName = "返回"
			timeLabel = "返回时间"
			timeText = ezfyDurationText((o.ReturnTime - now) / 1000)
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
		views = append(views, gin.H{
			"id": o.ID, "order_type": o.OrderType, "type_name": ezfyOrderTypeName(o.OrderType),
			"target_type": o.TargetType, "target_name": h.ezfyTargetName(o),
			"target_x": o.TargetX, "target_y": o.TargetY,
			"status": o.Status, "status_name": statusName,
			"officer": o.Officer, "time_label": timeLabel, "time_text": timeText,
			"arrive_time": o.ArriveTime, "return_time": o.ReturnTime,
			"carry": c, "carry_total": c.total(), "carry_cap": h.ezfyCarryCap(o),
			// 指挥室：可指挥时前端显示 [指挥]
			"can_command":    o.Status == ezfyOrderStatusBattle,
			"battle_round":   battleRound,
			"battle_max":     ezfyBattleMaxRounds,
			"battle_left_ms": battleLeft,
		})
	}

	// ★ 防守方视角（2026-09-23 用户要求「敌人打自己，自己也能指挥」）：
	//   战场/订单属于**攻方**，上面的军队动态按 user_id 查不到守方要防守的这场战斗。
	//   这里单独把「正在被攻打(def_user_id = 我方)」的战场拼进列表，让守方也有 [指挥] 入口。
	var defBattles []model.EzfyBattle
	h.DB.Where("def_user_id = ? AND status = 1", uid).Find(&defBattles)
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
