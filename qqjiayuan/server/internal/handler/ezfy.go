package handler

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 二战风云 核心玩法：进入游戏/城池/建筑/资源懒结算/造兵/伤兵/科技

const (
	ezfyFactoryBuildingID = 14                     // 军工厂
	ezfyMaxFactoryCount   = 5                      // 军工厂最多建造数
	ezfyMaxBuildings      = 33                     // 军事区+资源区建筑总数上限
	ezfyMaxHouseCount     = 10                     // 民居最多建造数
	ezfyConveneGoldCost   = 100000                 // 召集人口消耗黄金
	ezfyConvenePopGain    = 100000                 // 召集获得人口
	ezfyNewCityGoldCost   = 100000                 // 平原起新城消耗黄金
	ezfyOilDivGrid        = 300                    // 出征耗油: 每格耗油 = 总兵力/300
	ezfyDispatchPeriod    = int64(8 * 3600 * 1000) // 派遣采集结算周期 8小时
	ezfyDispatchTreasure  = 10                     // 派遣结算宝物概率 1/10
	ezfyCommandCarryPct   = 10                     // 指挥艺术: 出征携带上限+%/级
	ezfyMaxUpgradeSeconds = 10                     // 一键满级: 每级升级时间(秒)
	ezfyDeserterRate      = 30                     // 守军战败溃逃比例%
	ezfyWarDelayHours     = 24                     // 宣战生效延迟(小时)
	ezfyWarDurationHours  = 48                     // 宣战有效期(小时)
)

var ezfyRequirePattern = regexp.MustCompile(`([^()（）]+)[（(]\s*(\d+)\s*级?\s*[）)]`)

// 科技研究所需科研中心等级
var ezfyTechAcademy = map[int]int{
	1: 1, 2: 1, 3: 1, 4: 1, 5: 1, 6: 2, 9: 3, 7: 3, 12: 3,
	13: 4, 10: 4, 8: 4, 11: 5, 19: 5, 18: 6, 14: 6, 21: 6,
	15: 7, 16: 8, 20: 9, 17: 10,
}

type EzfyHandler struct{ DB *gorm.DB }

func (h *EzfyHandler) cfgs() {
	ezfyCfg.load(h.DB)
}

// ============ 玩家/城市基础 ============

// ensureProfile 懒创建玩家档案（声望/阵营）
func (h *EzfyHandler) ensureProfile(uid uint) model.EzfyProfile {
	var p model.EzfyProfile
	if err := h.DB.Where("user_id = ?", uid).First(&p).Error; err == nil {
		return p
	}
	var u model.User
	nickname := ""
	if err := h.DB.Select("nickname").First(&u, uid).Error; err == nil {
		nickname = u.Nickname
	}
	p = model.EzfyProfile{UserID: uid, Nickname: nickname, Prestige: 0, Camp: 1}
	h.DB.Create(&p)
	return p
}

func (h *EzfyHandler) addPrestige(uid uint, amount int) {
	if amount <= 0 {
		return
	}
	// 节日活动·声望加成(福利.txt #11)
	if pct := h.actPct(ezfyActPrestige); pct > 0 {
		amount = amount * (100 + pct) / 100
	}
	p := h.ensureProfile(uid)
	h.DB.Model(&model.EzfyProfile{}).Where("id = ?", p.ID).
		Update("prestige", p.Prestige+amount)
}

// getOrCreateCity 懒创建主城（随机平原空位，初始建筑 市政厅/民居/农田 各1级）
func (h *EzfyHandler) getOrCreateCity(uid uint) model.EzfyCity {
	var city model.EzfyCity
	if err := h.DB.Where("user_id = ?", uid).Order("id ASC").First(&city).Error; err == nil {
		return city
	}
	pos := h.findFreePos()
	city = model.EzfyCity{
		UserID: uid, Name: "新城市",
		Feelings: 80, Grievance: 0, TaxRate: 20,
		Pop: 0, PopMax: 100,
		Gold: 20000, Food: 5000, Steel: 5000, Oil: 5000, Rare: 5000,
		GoldCap: 1000000, FoodCap: 100000, SteelCap: 100000, OilCap: 100000, RareCap: 100000,
		CityLevel: 1, LastTime: time.Now().UnixMilli(),
		WareFood: 25, WareSteel: 25, WareOil: 25, WareRare: 25,
		X: pos[0], Y: pos[1],
	}
	h.DB.Create(&city)
	h.initBuilding(city.ID, 1, 1)
	h.initBuilding(city.ID, 2, 1)
	h.initBuilding(city.ID, 3, 1)
	return city
}

func (h *EzfyHandler) initBuilding(cityId uint, buildingId, level int) {
	b := model.EzfyCityBuilding{CityId: int64(cityId), BuildingId: buildingId, Level: level, Status: 0}
	h.DB.Create(&b)
}

func (h *EzfyHandler) findFreePos() [2]int {
	for i := 0; i < 100; i++ {
		x := 50 + rand.Intn(400)
		y := 50 + rand.Intn(400)
		if ezfyTerrain(x, y) == 8 {
			continue
		}
		var n int64
		h.DB.Model(&model.EzfyCity{}).Where("x = ? AND y = ?", x, y).Count(&n)
		if n > 0 {
			continue
		}
		return [2]int{x, y}
	}
	return [2]int{200, 200}
}

// cityOf 按归属取城市（含被占领不可进入校验）
func (h *EzfyHandler) cityOf(uid uint, cityId int64) *model.EzfyCity {
	var city model.EzfyCity
	if err := h.DB.Where("user_id = ? AND id = ?", uid, cityId).First(&city).Error; err != nil {
		return nil
	}
	var oc int64
	h.DB.Model(&model.EzfyOccupy{}).Where("city_id = ? AND status = 1", city.ID).Count(&oc)
	if oc > 0 {
		return nil
	}
	return &city
}

func (h *EzfyHandler) buildingList(cityId uint) []model.EzfyCityBuilding {
	var list []model.EzfyCityBuilding
	h.DB.Where("city_id = ?", cityId).Order("building_id ASC").Find(&list)
	return list
}

func (h *EzfyHandler) buildingLevel(cityId uint, buildingId int) int {
	max := 0
	for _, b := range h.buildingList(cityId) {
		if b.BuildingId == buildingId && b.Level > max {
			max = b.Level
		}
	}
	return max
}

func (h *EzfyHandler) buildingTotalLevel(cityId uint, buildingId int) int {
	total := 0
	for _, b := range h.buildingList(cityId) {
		if b.BuildingId == buildingId {
			total += b.Level
		}
	}
	return total
}

func (h *EzfyHandler) areaBuildingCount(cityId uint) int {
	n := 0
	for _, b := range h.buildingList(cityId) {
		if c := ezfyCfg.building(b.BuildingId); c != nil && (c.Type == 1 || c.Type == 2 || c.Type == 3) {
			n++
		}
	}
	return n
}

func (h *EzfyHandler) techMap(cityId uint) map[int]int {
	var list []model.EzfyCityTech
	h.DB.Where("city_id = ?", cityId).Find(&list)
	m := map[int]int{}
	for _, t := range list {
		m[t.TechId] = t.Level
	}
	return m
}

func (h *EzfyHandler) troopList(cityId uint) []model.EzfyCityTroop {
	var list []model.EzfyCityTroop
	h.DB.Where("city_id = ?", cityId).Find(&list)
	return list
}

func (h *EzfyHandler) troopMap(cityId uint) map[int]int64 {
	m := map[int]int64{}
	for _, t := range h.troopList(cityId) {
		m[t.TroopId] = t.Count
	}
	return m
}

func (h *EzfyHandler) addTroop(cityId uint, troopId int, count int64) {
	var t model.EzfyCityTroop
	if err := h.DB.Where("city_id = ? AND troop_id = ?", cityId, troopId).First(&t).Error; err != nil {
		t = model.EzfyCityTroop{CityId: int64(cityId), TroopId: troopId, Count: count}
		h.DB.Create(&t)
		return
	}
	h.DB.Model(&model.EzfyCityTroop{}).Where("id = ?", t.ID).Update("count", t.Count+count)
}

func (h *EzfyHandler) addWounded(cityId uint, troopId, wtype int, count int64) {
	if count <= 0 {
		return
	}
	var w model.EzfyWounded
	if err := h.DB.Where("city_id = ? AND troop_id = ? AND type = ?", cityId, troopId, wtype).First(&w).Error; err != nil {
		w = model.EzfyWounded{CityId: int64(cityId), TroopId: troopId, Type: wtype, Count: count}
		h.DB.Create(&w)
		return
	}
	h.DB.Model(&model.EzfyWounded{}).Where("id = ?", w.ID).Update("count", w.Count+count)
}

func (h *EzfyHandler) wildlandList(cityId uint) []model.EzfyWildland {
	var list []model.EzfyWildland
	h.DB.Where("city_id = ?", cityId).Find(&list)
	return list
}

// isSeaCity 是否海城(城市本身建在海洋地形上)
// 复刻用户规则: 海城才能训练海军; 陆地城市不能训练海军
func (h *EzfyHandler) isSeaCity(city *model.EzfyCity) bool {
	return ezfyTerrain(city.X, city.Y) == 8
}

func (h *EzfyHandler) isCoastalCity(city *model.EzfyCity) bool {
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			if dx == 0 && dy == 0 {
				continue
			}
			if ezfyTerrain(city.X+dx, city.Y+dy) == 8 {
				return true
			}
		}
	}
	return false
}

// ============ 懒结算五连（复刻 GameServiceImpl checkBuildingDone/collectTrainQueue/calcResource/processOrders） ============

// refreshCity 每次进入游戏接口前统一懒结算
func (h *EzfyHandler) refreshCity(uid uint, city *model.EzfyCity) {
	h.checkBuildingDone(city)
	h.checkTechDone(city)
	h.collectTrainQueue(city)
	h.calcResource(city)
	h.processOrders(uid)
}

func (h *EzfyHandler) checkBuildingDone(city *model.EzfyCity) {
	now := time.Now().UnixMilli()
	for _, b := range h.buildingList(city.ID) {
		if b.Status != 0 && now >= b.EndTime {
			b.Level++
			h.addPrestige(city.UserID, b.Level*10)
			cfg := ezfyCfg.building(b.BuildingId)
			if b.StartTime == 0 && cfg != nil && b.Level < cfg.MaxLevel {
				b.Status = 2
				b.EndTime = now + ezfyMaxUpgradeSeconds*1000
			} else {
				b.Status = 0
				if b.BuildingId == 1 {
					city.CityLevel = b.Level
				}
				h.taskProgress(city.UserID, "build_upgrade", 1)
			}
			h.DB.Model(&model.EzfyCityBuilding{}).Where("id = ?", b.ID).
				Updates(map[string]interface{}{"level": b.Level, "status": b.Status, "end_time": b.EndTime})
		}
	}
	var hall model.EzfyCityBuilding
	if err := h.DB.Where("city_id = ? AND building_id = 1", city.ID).First(&hall).Error; err == nil && hall.Level != city.CityLevel {
		city.CityLevel = hall.Level
		h.DB.Model(&model.EzfyCity{}).Where("id = ?", city.ID).Update("city_level", hall.Level)
	}
}

// calcResource 资源按小时懒结算（民心/民怨/科技/道具增产/野地产出/军队耗粮）
func (h *EzfyHandler) calcResource(city *model.EzfyCity) {
	now := time.Now().UnixMilli()
	last := city.LastTime
	if last <= 0 {
		last = now
	}
	if now <= last {
		return
	}
	hours := float64(now-last) / 3600000.0

	tech := h.techMap(city.ID)
	techFood := tech[1]
	techSteel := tech[2]
	techOil := tech[3]
	techRare := tech[4]
	techStore := tech[14]
	techSupply := tech[18]

	feelings := city.Feelings
	grievance := city.Grievance
	var feelingsDelta int
	if grievance >= 50 {
		feelingsDelta = -1
	} else if city.TaxRate <= 10 {
		feelingsDelta = 2
	} else if city.TaxRate <= 20 {
		feelingsDelta = 1
	} else if city.TaxRate <= 40 {
		feelingsDelta = 0
	} else if city.TaxRate <= 60 {
		feelingsDelta = -1
	} else {
		feelingsDelta = -2
	}
	feelings += int(float64(feelingsDelta)*hours + 0.5)
	if feelings < 0 {
		feelings = 0
	}
	if feelings > 100 {
		feelings = 100
	}
	grievance -= int(hours + 0.5)
	if grievance < 0 {
		grievance = 0
	}
	city.Feelings = feelings
	city.Grievance = grievance

	morale := float64(feelings) / 100.0
	if grievance >= 50 {
		morale *= 0.5
	}

	var foodProd, steelProd, oilProd, rareProd, goldProd int64
	var popMax int64
	var goldCap, resCap int64
	for _, b := range h.buildingList(city.ID) {
		lv := ezfyCfg.buildingLevel(b.BuildingId, b.Level)
		if lv == nil {
			continue
		}
		cfg := ezfyCfg.building(b.BuildingId)
		if cfg == nil {
			continue
		}
		if cfg.Type == 1 || b.BuildingId == 2 || b.BuildingId == 3 || b.BuildingId == 4 ||
			b.BuildingId == 5 || b.BuildingId == 6 || b.BuildingId == 11 || b.BuildingId == 12 {
			prod := lv.Capacity / 100
			switch b.BuildingId {
			case 2:
				popMax += lv.Capacity
			case 3:
				foodProd += prod
			case 4:
				steelProd += prod
			case 5:
				oilProd += prod
			case 6:
				rareProd += prod
			case 12:
				resCap = lv.Capacity
			}
		} else if b.BuildingId == 1 {
			goldCap = lv.Capacity
		}
	}
	city.PopMax = popMax
	if resCap <= 0 {
		resCap = 100000
	}
	city.FoodCap = resCap
	city.SteelCap = resCap
	city.OilCap = resCap
	city.RareCap = resCap
	city.GoldCap = goldCap

	foodProd = foodProd * int64(100+techFood*10) / 100
	steelProd = steelProd * int64(100+techSteel*10) / 100
	oilProd = oilProd * int64(100+techOil*10) / 100
	rareProd = rareProd * int64(100+techRare*10) / 100
	// 调整生产·开工率(0~100)，复刻原版 city/sourceSet.html
	foodProd = foodProd * int64(ezfyRate(city.RateFood)) / 100
	steelProd = steelProd * int64(ezfyRate(city.RateSteel)) / 100
	oilProd = oilProd * int64(ezfyRate(city.RateOil)) / 100
	rareProd = rareProd * int64(ezfyRate(city.RateRare)) / 100
	// 市长加成：产量 +10% + 后勤属性/20（复刻原版 mayorBonus）
	if mayorBonus := h.mayorBonusPct(city.ID); mayorBonus > 0 {
		foodProd = foodProd * int64(100+mayorBonus) / 100
		steelProd = steelProd * int64(100+mayorBonus) / 100
		oilProd = oilProd * int64(100+mayorBonus) / 100
		rareProd = rareProd * int64(100+mayorBonus) / 100
	}
	foodProd = int64(float64(foodProd) * morale)
	steelProd = int64(float64(steelProd) * morale)
	oilProd = int64(float64(oilProd) * morale)
	rareProd = int64(float64(rareProd) * morale)
	goldProd = int64(float64(city.Pop) * float64(city.TaxRate) / 100.0 * morale)

	var wildFood, wildSteel, wildOil, wildRare, wildGold int64
	wildlands := h.wildlandList(city.ID)
	for i := range wildlands {
		w := &wildlands[i]
		base := int64(w.Level) * 100
		if w.WildType == 2 {
			wildOil += base
			wildRare += base
			wildGold += base
		} else {
			wildFood += base
			wildSteel += base
			wildOil += base
			wildRare += base
		}
		h.degradeWildland(w, now)
	}

	var boost model.EzfyCityEffect
	if err := h.DB.Where("city_id = ? AND effect_type = 1", city.ID).First(&boost).Error; err == nil {
		if boost.UntilTime > now {
			mult := int64(100 + boost.Param1)
			foodProd = foodProd * mult / 100
			steelProd = steelProd * mult / 100
			oilProd = oilProd * mult / 100
			rareProd = rareProd * mult / 100
			goldProd = goldProd * mult / 100
			wildFood = wildFood * mult / 100
			wildSteel = wildSteel * mult / 100
			wildOil = wildOil * mult / 100
			wildRare = wildRare * mult / 100
			wildGold = wildGold * mult / 100
		} else {
			h.DB.Delete(&boost)
		}
	}

	// 节日活动·资源增产(福利.txt #3)
	if pct := h.actPct(ezfyActProduce); pct > 0 {
		mult := int64(100 + pct)
		foodProd = foodProd * mult / 100
		steelProd = steelProd * mult / 100
		oilProd = oilProd * mult / 100
		rareProd = rareProd * mult / 100
		wildFood = wildFood * mult / 100
		wildSteel = wildSteel * mult / 100
		wildOil = wildOil * mult / 100
		wildRare = wildRare * mult / 100
	}

	if city.Pop < city.PopMax {
		grow := int64(float64(city.PopMax) * 0.02 * hours)
		if grow < 1 {
			grow = 1
		}
		city.Pop += grow
		if city.Pop > city.PopMax {
			city.Pop = city.PopMax
		}
	}
	var troopFoodCost int64
	for tid, count := range h.troopMap(city.ID) {
		if cfg := ezfyCfg.troop(tid); cfg != nil {
			troopFoodCost += int64(cfg.FoodKeep) * count
		}
	}
	troopFoodCost = troopFoodCost * int64(100-techSupply*2) / 100
	troopFoodCost = int64(float64(troopFoodCost) * hours)

	food := city.Food - troopFoodCost
	prod := int64(float64(foodProd)*hours) + int64(float64(wildFood)*hours)
	food += min64(prod, max64(0, city.FoodCap-food))
	if food < 0 {
		food = 0
	}
	city.Food = food
	city.Steel += min64(int64(float64(steelProd)*hours)+int64(float64(wildSteel)*hours),
		max64(0, city.SteelCap-city.Steel))
	city.Oil += min64(int64(float64(oilProd)*hours)+int64(float64(wildOil)*hours),
		max64(0, city.OilCap-city.Oil))
	city.Rare += min64(int64(float64(rareProd)*hours)+int64(float64(wildRare)*hours),
		max64(0, city.RareCap-city.Rare))
	gold := city.Gold
	gold += min64(int64(float64(goldProd)*hours)+int64(float64(wildGold)*hours),
		max64(0, city.GoldCap-gold))
	if gold < 0 {
		gold = 0
	}
	city.Gold = gold

	if techStore > 0 {
		capBonus := int64(100 + techStore*2)
		city.FoodCap *= capBonus / 100
		city.SteelCap *= capBonus / 100
		city.OilCap *= capBonus / 100
		city.RareCap *= capBonus / 100
		city.GoldCap *= capBonus / 100
	}
	city.LastTime = now
	h.DB.Model(&model.EzfyCity{}).Where("id = ?", city.ID).Updates(map[string]interface{}{
		"feelings": city.Feelings, "grievance": city.Grievance,
		"pop": city.Pop, "pop_max": city.PopMax,
		"gold": city.Gold, "food": city.Food, "steel": city.Steel,
		"oil": city.Oil, "rare": city.Rare,
		"gold_cap": city.GoldCap, "food_cap": city.FoodCap,
		"steel_cap": city.SteelCap, "oil_cap": city.OilCap, "rare_cap": city.RareCap,
		"last_time": city.LastTime,
	})
}

func (h *EzfyHandler) degradeWildland(w *model.EzfyWildland, now int64) {
	base := w.UpdatedAt.UnixMilli()
	period := int64(2 * 24 * 3600000)
	if now-base < period {
		return
	}
	steps := (now - base) / period
	lv := w.Level - int(steps)
	if lv < 0 {
		h.DB.Delete(&model.EzfyWildland{}, w.ID)
		h.DB.Where("x = ? AND y = ?", w.X, w.Y).Delete(&model.EzfyMapArea{})
		return
	}
	h.DB.Model(&model.EzfyWildland{}).Where("id = ?", w.ID).
		Updates(map[string]interface{}{"level": lv, "updated_at": time.UnixMilli(base + steps*period)})
}

func (h *EzfyHandler) collectTrainQueue(city *model.EzfyCity) {
	now := time.Now().UnixMilli()
	var list []model.EzfyTrainQueue
	h.DB.Where("city_id = ? AND status = 0", city.ID).Find(&list)
	for _, q := range list {
		if q.EndTime <= now {
			h.addTroop(city.ID, q.TroopId, q.Count)
			h.DB.Model(&model.EzfyTrainQueue{}).Where("id = ?", q.ID).Update("status", 2)
		}
	}
}

// getResourceCalc 资源详情页数据（与 calcResource 同一套公式）
func (h *EzfyHandler) getResourceCalc(city *model.EzfyCity) gin.H {
	tech := h.techMap(city.ID)
	techFood, techSteel, techOil, techRare := tech[1], tech[2], tech[3], tech[4]
	techSupply, techStore := tech[18], tech[14]
	morale := float64(city.Feelings) / 100.0
	if city.Grievance >= 50 {
		morale *= 0.5
	}
	var foodBase, steelBase, oilBase, rareBase int64
	for _, b := range h.buildingList(city.ID) {
		lv := ezfyCfg.buildingLevel(b.BuildingId, b.Level)
		if lv == nil {
			continue
		}
		prod := lv.Capacity / 100
		switch b.BuildingId {
		case 3:
			foodBase += prod
		case 4:
			steelBase += prod
		case 5:
			oilBase += prod
		case 6:
			rareBase += prod
		}
	}
	goldBase := int64(float64(city.Pop) * float64(city.TaxRate) / 100.0)
	foodProd := foodBase * int64(100+techFood*10) / 100
	steelProd := steelBase * int64(100+techSteel*10) / 100
	oilProd := oilBase * int64(100+techOil*10) / 100
	rareProd := rareBase * int64(100+techRare*10) / 100
	foodProd = int64(float64(foodProd) * morale)
	steelProd = int64(float64(steelProd) * morale)
	oilProd = int64(float64(oilProd) * morale)
	rareProd = int64(float64(rareProd) * morale)
	goldProd := int64(float64(city.Pop) * float64(city.TaxRate) / 100.0 * morale)

	var wildFood, wildSteel, wildOil, wildRare, wildGold int64
	for _, w := range h.wildlandList(city.ID) {
		base := int64(w.Level) * 100
		if w.WildType == 2 {
			wildOil += base
			wildRare += base
			wildGold += base
		} else {
			wildFood += base
			wildSteel += base
			wildOil += base
			wildRare += base
		}
	}
	var boost model.EzfyCityEffect
	if err := h.DB.Where("city_id = ? AND effect_type = 1", city.ID).First(&boost).Error; err == nil && boost.UntilTime > time.Now().UnixMilli() {
		mult := int64(100 + boost.Param1)
		foodProd *= mult / 100
		steelProd *= mult / 100
		oilProd *= mult / 100
		rareProd *= mult / 100
		goldProd *= mult / 100
		wildFood *= mult / 100
		wildSteel *= mult / 100
		wildOil *= mult / 100
		wildRare *= mult / 100
		wildGold *= mult / 100
	}
	// 节日活动·资源增产(福利.txt #3) —— 与 calcResource 保持一致
	if pct := h.actPct(ezfyActProduce); pct > 0 {
		mult := int64(100 + pct)
		foodProd = foodProd * mult / 100
		steelProd = steelProd * mult / 100
		oilProd = oilProd * mult / 100
		rareProd = rareProd * mult / 100
		wildFood = wildFood * mult / 100
		wildSteel = wildSteel * mult / 100
		wildOil = wildOil * mult / 100
		wildRare = wildRare * mult / 100
	}
	var troopFood int64
	for tid, count := range h.troopMap(city.ID) {
		if cfg := ezfyCfg.troop(tid); cfg != nil {
			troopFood += int64(cfg.FoodKeep) * count
		}
	}
	troopFood = troopFood * int64(100-techSupply*2) / 100

	item := func(stock, cap, base, bonus, consume, total int64, extra gin.H) gin.H {
		m := gin.H{"stock": stock, "cap": cap, "base": base, "bonus": bonus, "consume": consume, "total": total, "store_tech": techStore}
		for k, v := range extra {
			m[k] = v
		}
		return m
	}
	return gin.H{
		"food":  item(city.Food, city.FoodCap, foodBase, foodProd-foodBase+wildFood, troopFood, foodProd+wildFood-troopFood, gin.H{"tech_prod": techFood, "troop_consume": troopFood, "supply_tech": techSupply}),
		"steel": item(city.Steel, city.SteelCap, steelBase, steelProd-steelBase+wildSteel, 0, steelProd+wildSteel, gin.H{"tech_prod": techSteel}),
		"oil":   item(city.Oil, city.OilCap, oilBase, oilProd-oilBase+wildOil, 0, oilProd+wildOil, gin.H{"tech_prod": techOil}),
		"rare":  item(city.Rare, city.RareCap, rareBase, rareProd-rareBase+wildRare, 0, rareProd+wildRare, gin.H{"tech_prod": techRare}),
		"gold":  item(city.Gold, city.GoldCap, goldBase, goldProd-goldBase+wildGold, 0, goldProd+wildGold, gin.H{"tech_prod": 0}),
	}
}

// ============ 建筑操作 ============

func (h *EzfyHandler) pay(city *model.EzfyCity, lv *model.EzfyCfgBuildingLevel) bool {
	if city.Food < lv.Food || city.Steel < lv.Steel || city.Oil < lv.Oil ||
		city.Rare < lv.Rare || city.Gold < lv.Gold {
		return false
	}
	city.Food -= lv.Food
	city.Steel -= lv.Steel
	city.Oil -= lv.Oil
	city.Rare -= lv.Rare
	city.Gold -= lv.Gold
	h.saveCityRes(city)
	return true
}

func (h *EzfyHandler) saveCityRes(city *model.EzfyCity) {
	h.DB.Model(&model.EzfyCity{}).Where("id = ?", city.ID).Updates(map[string]interface{}{
		"gold": city.Gold, "food": city.Food, "steel": city.Steel,
		"oil": city.Oil, "rare": city.Rare,
	})
}

func (h *EzfyHandler) buildBuilding(city *model.EzfyCity, buildingId int) string {
	h.refreshCity(city.UserID, city)
	cfg := ezfyCfg.building(buildingId)
	if cfg == nil {
		return "建筑不存在"
	}
	lv := ezfyCfg.buildingLevel(buildingId, 1)
	if lv == nil {
		return "建筑配置缺失"
	}
	countOfType := func() int64 {
		var n int64
		h.DB.Model(&model.EzfyCityBuilding{}).Where("city_id = ? AND building_id = ?", city.ID, buildingId).Count(&n)
		return n
	}
	if cfg.UniqueFlag == 1 {
		if countOfType() > 0 {
			return "该建筑已存在"
		}
	}
	if buildingId == ezfyFactoryBuildingID {
		if countOfType() >= ezfyMaxFactoryCount {
			return fmt.Sprintf("军工厂最多建造%d个", ezfyMaxFactoryCount)
		}
	} else if buildingId == 2 {
		if countOfType() >= ezfyMaxHouseCount {
			return fmt.Sprintf("民居最多建造%d个", ezfyMaxHouseCount)
		}
	} else if cfg.Type == 2 || cfg.Type == 3 {
		if countOfType() > 0 {
			return "该建筑已存在"
		}
	}
	if cfg.Type == 1 || cfg.Type == 2 || cfg.Type == 3 {
		cnt := h.areaBuildingCount(city.ID)
		if cnt >= ezfyMaxBuildings {
			return fmt.Sprintf("建筑数量已达上限(%d/%d)", cnt, ezfyMaxBuildings)
		}
	}
	if buildingId == 19 && !h.isCoastalCity(city) && !h.isSeaCity(city) {
		return "航海协会只能建在沿海城市或海城"
	}
	if !h.pay(city, lv) {
		return "资源不足"
	}
	buildTech := h.techMap(city.ID)[11]
	now := time.Now().UnixMilli()
	buildMs := int64(lv.BuildTime) * 1000 * int64(100-buildTech*2) / 100
	// 节日活动·建造加速(福利.txt #19/#26)
	if pct := h.actPct(ezfyActBuild); pct > 0 {
		buildMs = buildMs * int64(100-pct) / 100
	}
	if buildMs < 1000 {
		buildMs = 1000
	}
	b := model.EzfyCityBuilding{CityId: int64(city.ID), BuildingId: buildingId, Level: 0, Status: 1,
		StartTime: now, EndTime: now + buildMs}
	h.DB.Create(&b)
	return ""
}

func (h *EzfyHandler) upgradeBuilding(city *model.EzfyCity, recordId int64) string {
	h.refreshCity(city.UserID, city)
	var b model.EzfyCityBuilding
	if err := h.DB.Where("id = ? AND city_id = ?", recordId, city.ID).First(&b).Error; err != nil {
		return "建筑不存在"
	}
	if b.Status != 0 {
		return "建筑正在施工中"
	}
	target := b.Level + 1
	cfg := ezfyCfg.building(b.BuildingId)
	if cfg == nil || target > cfg.MaxLevel {
		return "已达到最高等级"
	}
	lv := ezfyCfg.buildingLevel(b.BuildingId, target)
	if lv == nil {
		return "配置缺失"
	}
	if target >= 10 && h.itemCount(city.UserID, 6) <= 0 {
		return fmt.Sprintf("升级到%d级需要建筑图纸", target)
	}
	if !h.pay(city, lv) {
		return "资源不足"
	}
	if target >= 10 {
		h.consumeItem(city.UserID, 6)
	}
	buildTech := h.techMap(city.ID)[11]
	now := time.Now().UnixMilli()
	buildMs := int64(lv.BuildTime) * 1000 * int64(100-buildTech*2) / 100
	// 节日活动·建造加速(福利.txt #19/#26)
	if pct := h.actPct(ezfyActBuild); pct > 0 {
		buildMs = buildMs * int64(100-pct) / 100
	}
	if buildMs < 1000 {
		buildMs = 1000
	}
	h.DB.Model(&model.EzfyCityBuilding{}).Where("id = ?", b.ID).
		Updates(map[string]interface{}{"status": 2, "start_time": now, "end_time": now + buildMs})
	return ""
}

func (h *EzfyHandler) maxLevelBuilding(city *model.EzfyCity, recordId int64) string {
	h.refreshCity(city.UserID, city)
	var b model.EzfyCityBuilding
	if err := h.DB.Where("id = ? AND city_id = ?", recordId, city.ID).First(&b).Error; err != nil {
		return "建筑不存在"
	}
	if b.Status != 0 {
		return "建筑正在施工中"
	}
	cfg := ezfyCfg.building(b.BuildingId)
	if cfg == nil {
		return "建筑配置缺失"
	}
	if b.Level >= cfg.MaxLevel {
		return "该建筑已满级"
	}
	var needFood, needSteel, needOil, needRare, needGold int64
	for lv := b.Level + 1; lv <= cfg.MaxLevel; lv++ {
		if l := ezfyCfg.buildingLevel(b.BuildingId, lv); l != nil {
			needFood += l.Food
			needSteel += l.Steel
			needOil += l.Oil
			needRare += l.Rare
			needGold += l.Gold
		}
	}
	if city.Food < needFood || city.Steel < needSteel || city.Oil < needOil ||
		city.Rare < needRare || city.Gold < needGold {
		return fmt.Sprintf("资源不足: 满级需 粮%d 钢%d 油%d 稀矿%d 金%d", needFood, needSteel, needOil, needRare, needGold)
	}
	city.Food -= needFood
	city.Steel -= needSteel
	city.Oil -= needOil
	city.Rare -= needRare
	city.Gold -= needGold
	h.saveCityRes(city)
	now := time.Now().UnixMilli()
	h.DB.Model(&model.EzfyCityBuilding{}).Where("id = ?", b.ID).
		Updates(map[string]interface{}{"status": 2, "start_time": 0, "end_time": now + ezfyMaxUpgradeSeconds*1000})
	return ""
}

func (h *EzfyHandler) deleteBuilding(city *model.EzfyCity, recordId int64) string {
	var b model.EzfyCityBuilding
	if err := h.DB.Where("id = ? AND city_id = ?", recordId, city.ID).First(&b).Error; err != nil {
		return "建筑不存在"
	}
	cfg := ezfyCfg.building(b.BuildingId)
	if cfg == nil || cfg.CanDelete == 0 {
		return "该建筑不可拆除"
	}
	if b.Status != 0 {
		return "施工中不可拆除"
	}
	h.DB.Delete(&b)
	return ""
}

func (h *EzfyHandler) speedUpBuilding(city *model.EzfyCity, recordId int64, minutes int64) string {
	var b model.EzfyCityBuilding
	var err error
	if recordId > 0 {
		err = h.DB.Where("id = ? AND city_id = ?", recordId, city.ID).First(&b).Error
	} else {
		err = h.DB.Where("city_id = ? AND status != 0", city.ID).Order("end_time ASC").First(&b).Error
	}
	if err != nil {
		return "没有正在施工的建筑"
	}
	if b.Status == 0 {
		return "建筑没有在施工"
	}
	end := time.Now().UnixMilli()
	if remain := b.EndTime - minutes*60000; remain > end {
		end = remain
	}
	h.DB.Model(&model.EzfyCityBuilding{}).Where("id = ?", b.ID).Update("end_time", end)
	return ""
}

// ============ 造兵/伤兵/逃兵 ============

func (h *EzfyHandler) trainTroop(city *model.EzfyCity, troopId, count int, split bool) string {
	h.refreshCity(city.UserID, city)
	if count <= 0 {
		return "数量错误"
	}
	cfg := ezfyCfg.troop(troopId)
	if cfg == nil {
		return "兵种不存在"
	}
	// 海军(type 1)只能在海城训练(用户规则: 陆地城市不能训练海军)
	if cfg.Type == 1 && !h.isSeaCity(city) {
		return "海军只能在海城(建在海洋上的城市)训练, 陆地城市无法训练海军"
	}
	if cfg.Require != "" {
		matches := ezfyRequirePattern.FindAllStringSubmatch(cfg.Require, -1)
		if len(matches) > 0 {
			for _, m := range matches {
				name := m[1]
				need, _ := strconv.Atoi(m[2])
				if bid, ok := ezfyCfg.buildingByName[name]; ok {
					if h.buildingLevel(city.ID, bid) < need {
						return fmt.Sprintf("需要%s %d级", name, need)
					}
					continue
				}
				if tid, ok := ezfyCfg.techByName[name]; ok {
					if h.techMap(city.ID)[tid] < need {
						return fmt.Sprintf("需要科技%s %d级", name, need)
					}
				}
			}
		} else {
			if bid, ok := ezfyCfg.buildingByName[cfg.Require]; ok && h.buildingLevel(city.ID, bid) < 1 {
				return fmt.Sprintf("需要%s 1级", cfg.Require)
			}
		}
	}
	popUsed := h.troopPop(city.ID)
	popAvailable := city.Pop - popUsed
	if cfg.Type != 4 && int64(cfg.Pop)*int64(count) > popAvailable {
		return fmt.Sprintf("人口不足(当前居民%d, 军队占用%d, 可用%d); 可召集人口突破民居上限",
			city.Pop, popUsed, popAvailable)
	}
	food := cfg.Food * int64(count)
	steel := cfg.Steel * int64(count)
	oil := cfg.Oil * int64(count)
	rare := cfg.Rare * int64(count)
	// 节日活动·造兵打折(福利.txt #4)
	food, steel, oil, rare = h.trainCostWithActivity(food, steel, oil, rare)
	if city.Food < food || city.Steel < steel || city.Oil < oil || city.Rare < rare {
		return "资源不足"
	}
	if cfg.Type == 4 {
		wallLevel := h.buildingLevel(city.ID, 7)
		space := int64(0)
		if wall := ezfyCfg.buildingLevel(7, wallLevel); wall != nil {
			space = wall.Capacity
		}
		used := int64(0)
		for tid, cnt := range h.troopMap(city.ID) {
			if c := ezfyCfg.troop(tid); c != nil && c.Type == 4 {
				used += cnt
			}
		}
		if used+int64(count) > space {
			return fmt.Sprintf("城防空间不足(围墙%d级, 上限%d)", wallLevel, space)
		}
	}
	factoryTotal := h.buildingTotalLevel(city.ID, ezfyFactoryBuildingID)
	var activeCount int64
	h.DB.Model(&model.EzfyTrainQueue{}).Where("city_id = ? AND status = 0", city.ID).Count(&activeCount)
	var queueLimit int
	if cfg.Type == 4 {
		queueLimit = maxInt(1, factoryTotal)
	} else {
		queueLimit = factoryTotal
	}
	if queueLimit <= 0 {
		return "请先建造军工厂"
	}
	if int(activeCount) >= queueLimit {
		return fmt.Sprintf("训练队列已满(军工厂等级合计%d个队列)", queueLimit)
	}
	n := 1
	if split && cfg.Type != 4 {
		var factoryCount int64
		h.DB.Model(&model.EzfyCityBuilding{}).
			Where("city_id = ? AND building_id = ? AND status = 0", city.ID, ezfyFactoryBuildingID).Count(&factoryCount)
		free := queueLimit - int(activeCount)
		n = maxInt(1, minInt(int(factoryCount), free))
	}
	city.Food -= food
	city.Steel -= steel
	city.Oil -= oil
	city.Rare -= rare
	h.saveCityRes(city)
	now := time.Now().UnixMilli()
	per := count / n
	rem := count % n
	for i := 0; i < n; i++ {
		part := int64(per)
		if i < rem {
			part++
		}
		if part <= 0 {
			continue
		}
		q := model.EzfyTrainQueue{CityId: int64(city.ID), TroopId: troopId, Count: part, Status: 0,
			StartTime: now, EndTime: now + int64(cfg.TrainTime)*1000*part}
		h.DB.Create(&q)
	}
	h.taskProgress(city.UserID, "train_troop", count)
	return ""
}

func (h *EzfyHandler) troopPop(cityId uint) int64 {
	pop := int64(0)
	for tid, count := range h.troopMap(cityId) {
		if cfg := ezfyCfg.troop(tid); cfg != nil && cfg.Type != 4 {
			pop += int64(cfg.Pop) * count
		}
	}
	return pop
}

func (h *EzfyHandler) recoverWounded(city *model.EzfyCity, troopId, wtype int) string {
	var w model.EzfyWounded
	if err := h.DB.Where("city_id = ? AND troop_id = ? AND type = ?", city.ID, troopId, wtype).First(&w).Error; err != nil {
		return "兵营中没有该兵种"
	}
	if w.Count <= 0 {
		return "兵营中没有该兵种"
	}
	h.addTroop(city.ID, w.TroopId, w.Count)
	h.DB.Delete(&w)
	return ""
}

func (h *EzfyHandler) recoverAllWounded(city *model.EzfyCity, wtype int) string {
	var list []model.EzfyWounded
	h.DB.Where("city_id = ? AND type = ?", city.ID, wtype).Find(&list)
	if len(list) == 0 {
		return "兵营中空空如也"
	}
	for _, w := range list {
		if w.Count <= 0 {
			continue
		}
		h.addTroop(city.ID, w.TroopId, w.Count)
		h.DB.Delete(&w)
	}
	return ""
}

// ============ 科技 ============

func (h *EzfyHandler) researchTech(city *model.EzfyCity, techId int) string {
	h.refreshCity(city.UserID, city)
	cfg := ezfyCfg.tech(techId)
	if cfg == nil {
		return "科技不存在"
	}
	academyNeed := 1
	if v, ok := ezfyTechAcademy[techId]; ok {
		academyNeed = v
	}
	if h.buildingLevel(city.ID, 8) < academyNeed {
		return fmt.Sprintf("需要科研中心%d级才能研究%s", academyNeed, cfg.Name)
	}
	curLevel := h.techMap(city.ID)[techId]
	if curLevel >= cfg.MaxLevel {
		return "已达到最高等级"
	}
	lv := ezfyCfg.techLevel(techId, curLevel+1)
	if lv == nil {
		return "配置缺失"
	}
	if cfg.PreTech > 0 {
		if h.techMap(city.ID)[cfg.PreTech] < cfg.PreTechLevel {
			pre := ezfyCfg.tech(cfg.PreTech)
			preName := ""
			if pre != nil {
				preName = pre.Name
			}
			return fmt.Sprintf("需要先研究%s %d级", preName, cfg.PreTechLevel)
		}
	}
	var researching int64
	h.DB.Model(&model.EzfyCityTech{}).Where("city_id = ? AND status = 1", city.ID).Count(&researching)
	if researching > 0 {
		return "已有科技研究中"
	}
	if city.Food < lv.Food || city.Steel < lv.Steel || city.Oil < lv.Oil ||
		city.Rare < lv.Rare || city.Gold < lv.Gold {
		return "资源不足"
	}
	city.Food -= lv.Food
	city.Steel -= lv.Steel
	city.Oil -= lv.Oil
	city.Rare -= lv.Rare
	city.Gold -= lv.Gold
	h.saveCityRes(city)
	now := time.Now().UnixMilli()
	var t model.EzfyCityTech
	if err := h.DB.Where("city_id = ? AND tech_id = ?", city.ID, techId).First(&t).Error; err != nil {
		t = model.EzfyCityTech{CityId: int64(city.ID), TechId: techId, Level: 0, Status: 1, EndTime: now + h.techResearchMs(lv.ResearchTime)}
		h.DB.Create(&t)
		return ""
	}
	h.DB.Model(&model.EzfyCityTech{}).Where("id = ?", t.ID).
		Updates(map[string]interface{}{"status": 1, "end_time": now + h.techResearchMs(lv.ResearchTime)})
	return ""
}

// cancelTech 取消研究(复刻原版 techIndex 的 [取消]): 全额退还本次研究消耗, 等级不变
func (h *EzfyHandler) cancelTech(city *model.EzfyCity, techId int) string {
	h.refreshCity(city.UserID, city)
	var t model.EzfyCityTech
	if err := h.DB.Where("city_id = ? AND tech_id = ? AND status = 1", city.ID, techId).First(&t).Error; err != nil {
		return "该科技没有在研究中"
	}
	if lv := ezfyCfg.techLevel(techId, t.Level+1); lv != nil {
		city.Food += lv.Food
		city.Steel += lv.Steel
		city.Oil += lv.Oil
		city.Rare += lv.Rare
		city.Gold += lv.Gold
		h.saveCityRes(city)
	}
	h.DB.Model(&model.EzfyCityTech{}).Where("id = ?", t.ID).
		Updates(map[string]interface{}{"status": 0, "end_time": 0})
	return ""
}

func (h *EzfyHandler) checkTechDone(city *model.EzfyCity) {
	now := time.Now().UnixMilli()
	var list []model.EzfyCityTech
	h.DB.Where("city_id = ? AND status = 1", city.ID).Find(&list)
	for _, t := range list {
		if now >= t.EndTime {
			h.DB.Model(&model.EzfyCityTech{}).Where("id = ?", t.ID).
				Updates(map[string]interface{}{"level": t.Level + 1, "status": 0})
			h.taskProgress(city.UserID, "tech_research", 1)
		}
	}
}

func (h *EzfyHandler) speedUpTech(city *model.EzfyCity, minutes int64) string {
	var t model.EzfyCityTech
	if err := h.DB.Where("city_id = ? AND status = 1", city.ID).First(&t).Error; err != nil {
		return "没有研究中的科技"
	}
	end := time.Now().UnixMilli()
	if remain := t.EndTime - minutes*60000; remain > end {
		end = remain
	}
	h.DB.Model(&model.EzfyCityTech{}).Where("id = ?", t.ID).Update("end_time", end)
	return ""
}

func (h *EzfyHandler) speedUpTrain(city *model.EzfyCity, queueId int64, minutes int64) string {
	var q model.EzfyTrainQueue
	var err error
	if queueId > 0 {
		err = h.DB.Where("id = ? AND city_id = ?", queueId, city.ID).First(&q).Error
	} else {
		err = h.DB.Where("city_id = ? AND status = 0", city.ID).Order("start_time ASC").First(&q).Error
	}
	if err != nil {
		return "没有训练中的队列"
	}
	end := time.Now().UnixMilli()
	if remain := q.EndTime - minutes*60000; remain > end {
		end = remain
	}
	h.DB.Model(&model.EzfyTrainQueue{}).Where("id = ?", q.ID).Update("end_time", end)
	return ""
}

// ============ 道具(背包/商城/加速/免战/增产) ============

func (h *EzfyHandler) itemCount(uid uint, cfgId int) int {
	var it model.EzfyItem
	if err := h.DB.Where("user_id = ? AND cfg_id = ?", uid, cfgId).First(&it).Error; err != nil {
		return 0
	}
	return it.Count
}

func (h *EzfyHandler) addItem(uid uint, cfgId, count int) {
	var it model.EzfyItem
	if err := h.DB.Where("user_id = ? AND cfg_id = ?", uid, cfgId).First(&it).Error; err != nil {
		h.DB.Create(&model.EzfyItem{UserId: uid, CfgId: cfgId, Count: count})
		return
	}
	h.DB.Model(&model.EzfyItem{}).Where("id = ?", it.ID).Update("count", it.Count+count)
}

func (h *EzfyHandler) consumeItem(uid uint, cfgId int) {
	var it model.EzfyItem
	if err := h.DB.Where("user_id = ? AND cfg_id = ?", uid, cfgId).First(&it).Error; err != nil {
		return
	}
	if it.Count <= 1 {
		h.DB.Delete(&it)
	} else {
		h.DB.Model(&model.EzfyItem{}).Where("id = ?", it.ID).Update("count", it.Count-1)
	}
}

func (h *EzfyHandler) addCityEffect(cityId uint, effectType, param1 int, hours int64) {
	var exist model.EzfyCityEffect
	now := time.Now().UnixMilli()
	until := now + hours*3600000
	if err := h.DB.Where("city_id = ? AND effect_type = ?", cityId, effectType).First(&exist).Error; err != nil {
		h.DB.Create(&model.EzfyCityEffect{CityId: int64(cityId), EffectType: effectType, Param1: param1, UntilTime: until})
		return
	}
	remain := exist.UntilTime - now
	if remain < 0 {
		remain = 0
	}
	h.DB.Model(&model.EzfyCityEffect{}).Where("id = ?", exist.ID).
		Updates(map[string]interface{}{"param1": param1, "until_time": now + remain + hours*3600000})
}

func (h *EzfyHandler) hasCityEffect(cityId uint, effectType int) bool {
	var e model.EzfyCityEffect
	if err := h.DB.Where("city_id = ? AND effect_type = ?", cityId, effectType).First(&e).Error; err != nil {
		return false
	}
	if e.UntilTime <= time.Now().UnixMilli() {
		h.DB.Delete(&e)
		return false
	}
	return true
}

// useItem 使用道具（type 3/4/5 加速优先选最早结束的目标）
// useItem 使用道具（支持批量：count 个；军官类道具需指定 officerId/skillId）
// 复刻设计文档《QQ家园二战风云.txt》道具 #7 招生简章 / #8 经验书 / #9 军官技能书·重修书
func (h *EzfyHandler) useItem(uid uint, city *model.EzfyCity, cfgId, count int, officerId int64, skillId int) string {
	cfg := ezfyCfg.item(cfgId)
	if cfg == nil {
		return "道具不存在"
	}
	if count <= 0 {
		count = 1
	}
	have := h.itemCount(uid, cfgId)
	if have <= 0 {
		return "道具数量不足"
	}
	if count > have {
		return fmt.Sprintf("道具数量不足(现有%d个)", have)
	}
	if count > 99 {
		return "单次最多使用99个"
	}
	// 单次生效类道具不能批量
	if (cfg.ItemType == 6 || cfg.ItemType == 11 || cfg.ItemType == 12) && count > 1 {
		return cfg.Name + "每次只能使用1个"
	}
	// 需要指定军官的道具
	needOfficer := cfg.ItemType == 10 || cfg.ItemType == 11 || cfg.ItemType == 12
	if needOfficer && officerId <= 0 {
		return "请先选择要使用的军官"
	}
	if cfg.ItemType == 11 && skillId <= 0 {
		return "请选择要学习的技能"
	}

	var lastMsg string
	for i := 0; i < count; i++ {
		msg := h.useItemOnce(uid, city, cfg, officerId, skillId)
		if !strings.HasPrefix(msg, "使用成功") {
			if i == 0 {
				return msg
			}
			// 批量中途失败: 已生效的部分保留, 返回说明
			return fmt.Sprintf("已使用%d个后中断: %s", i, msg)
		}
		lastMsg = msg
	}
	if count > 1 {
		return fmt.Sprintf("使用成功: %s×%d", cfg.Name, count)
	}
	return lastMsg
}

// useItemOnce 单个道具生效（内部函数, 由 useItem 调用）
func (h *EzfyHandler) useItemOnce(uid uint, city *model.EzfyCity, cfg *model.EzfyCfgItem, officerId int64, skillId int) string {
	cfgId := cfg.ID
	param := cfg.Param1
	var err string
	switch cfg.ItemType {
	case 1:
		city.Food += param
		city.Steel += param
		city.Oil += param
		city.Rare += param
		h.saveCityRes(city)
		h.consumeItem(uid, cfgId)
		return fmt.Sprintf("使用成功: 粮食/钢铁/石油/稀矿各+%d", param)
	case 2:
		city.Gold = min64(city.GoldCap, city.Gold+param)
		h.saveCityRes(city)
		h.consumeItem(uid, cfgId)
		return fmt.Sprintf("使用成功: 黄金+%d", param)
	case 3:
		err = h.speedUpBuilding(city, 0, param)
		if err != "" {
			return err
		}
		h.consumeItem(uid, cfgId)
		return fmt.Sprintf("使用成功: 当前建筑升级-%d分钟", param)
	case 4:
		err = h.speedUpTrain(city, 0, param)
		if err != "" {
			return err
		}
		h.consumeItem(uid, cfgId)
		return fmt.Sprintf("使用成功: 当前训练队列-%d分钟", param)
	case 5:
		err = h.speedUpTech(city, param)
		if err != "" {
			return err
		}
		h.consumeItem(uid, cfgId)
		return fmt.Sprintf("使用成功: 当前科技研究-%d分钟", param)
	case 6:
		return "建筑图纸将在建筑升级到10级时自动消耗"
	case 7:
		h.addCityEffect(city.ID, 1, int(param), 24)
		h.consumeItem(uid, cfgId)
		return fmt.Sprintf("使用成功: 资源产量+%d%%, 持续24小时", param)
	case 8:
		h.addCityEffect(city.ID, 2, 0, param)
		h.consumeItem(uid, cfgId)
		return fmt.Sprintf("使用成功: 城市免战保护%d小时", param)
	case 9: // 招生简章: 立即刷新军校候选(不占每日次数)
		if h.buildingLevel(city.ID, ezfyBuildingAcademy) < 1 {
			return "需要先建造军校"
		}
		if err := h.refreshRecruitFree(uid); err != "" {
			return err
		}
		h.consumeItem(uid, cfgId)
		return "使用成功: 军校候选名将已刷新"
	case 10: // 经验书
		o := h.officerOf(city.ID, officerId)
		if o == nil {
			return "军官不存在"
		}
		h.addOfficerExp(city, o.ID, param)
		h.consumeItem(uid, cfgId)
		return fmt.Sprintf("使用成功: %s 获得%d经验", o.Name, param)
	case 11: // 军官技能书: 免费学一个技能
		o := h.officerOf(city.ID, officerId)
		if o == nil {
			return "军官不存在"
		}
		if o.Status == 1 {
			return "军官出征中, 无法学习技能"
		}
		sk := ezfyCfg.skill(skillId)
		if sk == nil {
			return "技能不存在"
		}
		skills := officerSkills(o)
		if len(skills) >= ezfyOfficerMaxSkill {
			return "技能已满(最多3个)"
		}
		for _, s := range skills {
			if s == sk.Name {
				return "已学习该技能"
			}
		}
		skills = append(skills, sk.Name)
		h.saveOfficerSkills(o, skills)
		h.consumeItem(uid, cfgId)
		return fmt.Sprintf("使用成功: %s 学会了「%s」", o.Name, sk.Name)
	case 12: // 重修书: 属性回到名将初始值, 技能清空(等级/经验保留)
		o := h.officerOf(city.ID, officerId)
		if o == nil {
			return "军官不存在"
		}
		if o.Status == 1 {
			return "军官出征中, 无法重修"
		}
		initMil, initLog, initLea := o.Military, o.Logistics, o.Learning
		if g := ezfyCfg.general(o.GeneralId); g != nil {
			initMil, initLog, initLea = g.Military, g.Logistics, g.Learning
		}
		h.DB.Model(&model.EzfyOfficer{}).Where("id = ?", o.ID).Updates(map[string]interface{}{
			"military": initMil, "logistics": initLog, "learning": initLea,
			"skill": "", "update_time": time.Now(),
		})
		h.consumeItem(uid, cfgId)
		return fmt.Sprintf("使用成功: %s 已重修(属性回到初始值, 技能清空)", o.Name)
	default:
		return "道具类型错误"
	}
}

// ============ 任务 ============

var ezfyStateTaskTypes = map[string]bool{"city_level": true, "army_count": true, "wild_count": true}

func (h *EzfyHandler) initTasks(uid uint) {
	var cfgs []model.EzfyCfgTask
	h.DB.Where("status = 1").Order("sort_no ASC").Find(&cfgs)
	today := time.Now().Format("2006-01-02")
	for _, cfg := range cfgs {
		var count int64
		h.DB.Model(&model.EzfyTask{}).Where("user_id = ? AND cfg_id = ?", uid, cfg.ID).Count(&count)
		if count > 0 {
			continue
		}
		h.DB.Create(&model.EzfyTask{UserId: uid, CfgId: cfg.ID, Current: 0, Status: 0, TaskDate: today})
	}
}

func (h *EzfyHandler) resetDailyTasks(uid uint) {
	today := time.Now().Format("2006-01-02")
	var mine []model.EzfyTask
	h.DB.Where("user_id = ?", uid).Find(&mine)
	for _, t := range mine {
		if t.TaskDate == today {
			continue
		}
		var c model.EzfyCfgTask
		if err := h.DB.First(&c, t.CfgId).Error; err != nil || c.TypeId <= 0 {
			continue
		}
		var tp model.EzfyCfgTaskType
		if err := h.DB.First(&tp, c.TypeId).Error; err != nil || tp.ResetType != 1 {
			continue
		}
		updates := map[string]interface{}{"task_date": today}
		if t.Current > 0 {
			updates["current"] = 0
			updates["status"] = 0
		}
		h.DB.Model(&model.EzfyTask{}).Where("id = ?", t.ID).Updates(updates)
	}
}

func (h *EzfyHandler) calcStateValue(uid uint, taskType string) int {
	city := h.getOrCreateCity(uid)
	switch taskType {
	case "city_level":
		return h.buildingLevel(city.ID, 1)
	case "army_count":
		sum := int64(0)
		for _, c := range h.troopMap(city.ID) {
			sum += c
		}
		return int(sum)
	case "wild_count":
		return len(h.wildlandList(city.ID))
	default:
		return 0
	}
}

func (h *EzfyHandler) taskProgress(uid uint, taskType string, delta int) {
	if delta <= 0 {
		return
	}
	var mine []model.EzfyTask
	h.DB.Where("user_id = ? AND status = 0", uid).Find(&mine)
	if len(mine) == 0 {
		return
	}
	for _, t := range mine {
		var cfg model.EzfyCfgTask
		if err := h.DB.First(&cfg, t.CfgId).Error; err != nil || cfg.TaskType != taskType {
			continue
		}
		if cfg.Status == 0 {
			continue
		}
		cur := t.Current + delta
		if cur > cfg.Target {
			cur = cfg.Target
		}
		status := 0
		if cur >= cfg.Target {
			status = 1
		}
		h.DB.Model(&model.EzfyTask{}).Where("id = ?", t.ID).
			Updates(map[string]interface{}{"current": cur, "status": status})
	}
}

func (h *EzfyHandler) taskAward(uid uint, taskId int64) string {
	var t model.EzfyTask
	if err := h.DB.Where("id = ? AND user_id = ?", taskId, uid).First(&t).Error; err != nil {
		return "任务不存在"
	}
	if t.Status != 1 {
		return "任务未完成"
	}
	var cfg model.EzfyCfgTask
	if err := h.DB.First(&cfg, t.CfgId).Error; err != nil {
		return "任务配置缺失"
	}
	if cfg.Status == 0 {
		return "任务已停用"
	}
	city := h.getOrCreateCity(uid)
	city.Food = min64(city.FoodCap, city.Food+cfg.RewardFood)
	city.Steel = min64(city.SteelCap, city.Steel+cfg.RewardSteel)
	city.Oil = min64(city.OilCap, city.Oil+cfg.RewardOil)
	city.Rare = min64(city.RareCap, city.Rare+cfg.RewardRare)
	city.Gold = min64(city.GoldCap, city.Gold+cfg.RewardGold)
	h.saveCityRes(&city)
	if cfg.RewardPrestige > 0 {
		h.addPrestige(uid, cfg.RewardPrestige)
	}
	now := time.Now()
	h.DB.Model(&model.EzfyTask{}).Where("id = ?", t.ID).
		Updates(map[string]interface{}{"status": 2, "task_date": now.Format("2006-01-02"), "finish_time": now})
	return ""
}

// ============ HTTP Handlers ============

// View 游戏主页面数据: 档案+城市列表+当前城(懒结算)+建筑/军队/科技/队列/野地/命令概览
func (h *EzfyHandler) View(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	profile := h.ensureProfile(uid)
	city := h.getOrCreateCity(uid)
	h.refreshCity(uid, &city)

	var cities []model.EzfyCity
	h.DB.Where("user_id = ?", uid).Order("id ASC").Find(&cities)

	buildings := h.buildingList(city.ID)
	buildingViews := make([]gin.H, 0, len(buildings))
	for _, b := range buildings {
		cfg := ezfyCfg.building(b.BuildingId)
		lv := ezfyCfg.buildingLevel(b.BuildingId, b.Level)
		next := ezfyCfg.buildingLevel(b.BuildingId, b.Level+1)
		view := gin.H{
			"id": b.ID, "building_id": b.BuildingId, "level": b.Level,
			"status": b.Status, "end_time": b.EndTime, "start_time": b.StartTime,
		}
		if cfg != nil {
			view["name"] = cfg.Name
			view["type"] = cfg.Type
			view["max_level"] = cfg.MaxLevel
			view["des"] = cfg.Des
			view["can_delete"] = cfg.CanDelete
		}
		if lv != nil {
			view["effect"] = lv.Effect
			view["capacity"] = lv.Capacity
		}
		if next != nil {
			view["next_cost"] = gin.H{"food": next.Food, "steel": next.Steel, "oil": next.Oil, "rare": next.Rare, "gold": next.Gold}
			view["next_time"] = next.BuildTime
			view["next_effect"] = next.Effect
		}
		buildingViews = append(buildingViews, view)
	}
	// 可建造池(军事区 type2/3 + 资源区 type1), 供建筑页直接渲染, 前端不再硬编码
	buildingPool := h.buildPool(&city, buildings)

	camp := profile.Camp
	troopViews := []gin.H{}
	for tid, count := range h.troopMap(city.ID) {
		cfg := ezfyCfg.troop(tid)
		if cfg == nil {
			continue
		}
		troopViews = append(troopViews, gin.H{
			"troop_id": tid, "name": ezfyCfg.troopName(tid, camp), "count": count, "type": cfg.Type,
			"pop": cfg.Pop, "food_keep": cfg.FoodKeep,
		})
	}
	var wounded []model.EzfyWounded
	h.DB.Where("city_id = ?", city.ID).Order("type ASC, troop_id ASC").Find(&wounded)

	queues := []gin.H{}
	var qs []model.EzfyTrainQueue
	h.DB.Where("city_id = ? AND status = 0", city.ID).Order("start_time ASC").Find(&qs)
	for _, q := range qs {
		queues = append(queues, gin.H{"id": q.ID, "troop_id": q.TroopId,
			"name": ezfyCfg.troopName(q.TroopId, camp), "count": q.Count, "end_time": q.EndTime})
	}

	techViews := []gin.H{}
	for _, t := range h.techMap(city.ID) {
		if cfg := ezfyCfg.tech(t); cfg != nil {
			techViews = append(techViews, gin.H{"tech_id": t, "name": cfg.Name, "level": h.techMap(city.ID)[t]})
		}
	}

	var wildlands []model.EzfyWildland
	h.DB.Where("city_id = ?", city.ID).Find(&wildlands)
	wildViews := []gin.H{}
	for _, w := range wildlands {
		wildViews = append(wildViews, gin.H{"id": w.ID, "x": w.X, "y": w.Y, "level": w.Level,
			"wild_type": w.WildType, "terrain_name": ezfyTerrainName(ezfyTerrain(w.X, w.Y))})
	}

	var marching, occupying int64
	h.DB.Model(&model.EzfyOrder{}).Where("user_id = ? AND status = 0", uid).Count(&marching)
	h.DB.Model(&model.EzfyOrder{}).Where("user_id = ? AND status = 1", uid).Count(&occupying)
	var unreadReports int64
	h.DB.Model(&model.EzfyReport{}).Where("user_id = ? AND is_read = 0", uid).Count(&unreadReports)

	acct, ulv, uexp := h.ezfyUserBrief(uid)
	resp.OK(c, gin.H{
		"profile":       profile,
		"account":       acct,
		"user_level":    ulv,
		"user_exp":      uexp,
		"officer_count": h.officerCount(city.ID),
		"rank_name":     ezfyRankName(profile.Prestige),
		"rank_post":     ezfyRankPost(profile.Prestige),
		"cities":        cities,
		"city":          city,
		"continent":     ezfyContinentName(city.X, city.Y),
		// 海城/陆地城市(海城可建航海协会、训练海军)
		"is_sea":         h.isSeaCity(&city),
		"city_kind":      map[bool]string{true: "海城", false: "陆地城市"}[h.isSeaCity(&city)],
		"protected":      h.hasCityEffect(city.ID, 2),
		"boost":          h.hasCityEffect(city.ID, 1),
		"buildings":      buildingViews,
		"building_pool":  buildingPool,
		"troops":         troopViews,
		"wounded":        wounded,
		"queues":         queues,
		"techs":          techViews,
		"wildlands":      wildViews,
		"marching":       marching,
		"occupying":      occupying,
		"unread_reports": unreadReports,
	})
}

// ezfyAccount 家园账号(号码) / 等级 / 经验 —— 统帅信息页需要展示家园侧资料
func (h *EzfyHandler) ezfyUserBrief(uid uint) (account string, level, exp int) {
	var u model.User
	if err := h.DB.First(&u, uid).Error; err != nil {
		return "", 0, 0
	}
	return u.Username, u.Level, u.Exp
}

// ezfyRate 开工率归一化：0 视为未设置(按 100% 处理), 否则限制在 0~100
// 说明: 0 与「未设置」在本模型里无法区分, 而把开工率设成 0 等于停产(没有实际意义),
// 因此把 0 当作 100% 处理, 避免老数据(字段为 0)一上线就停产。
func ezfyRate(v int) int {
	if v <= 0 {
		return 100
	}
	if v > 100 {
		return 100
	}
	return v
}

// CityList 城市列表
func (h *EzfyHandler) CityList(c *gin.Context) {
	uid := middleware.GetUID(c)
	var cities []model.EzfyCity
	h.DB.Where("user_id = ?", uid).Order("id ASC").Find(&cities)
	resp.OK(c, gin.H{"cities": cities})
}

// SwitchCity 切换城市
func (h *EzfyHandler) SwitchCity(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		CityId int64 `json:"city_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if h.cityOf(uid, req.CityId) == nil {
		resp.ParamError(c, "城市不存在或已被占领")
		return
	}
	resp.OK(c, gin.H{"msg": "ok"})
}

// CreateCity 平原新建分城
func (h *EzfyHandler) CreateCity(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		X int `json:"x"`
		Y int `json:"y"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	h.cfgs()
	// 平原 → 陆地城市; 海洋 → 海城(可建海城建筑、训练海军)
	terr := ezfyTerrain(req.X, req.Y)
	if terr != 1 && terr != 8 {
		resp.ParamError(c, "只能在平原或海洋上建造新城")
		return
	}
	isSea := terr == 8
	var n int64
	h.DB.Model(&model.EzfyCity{}).Where("x = ? AND y = ?", req.X, req.Y).Count(&n)
	if n > 0 {
		resp.ParamError(c, "该位置已有城市, 无法建造")
		return
	}
	main := h.getOrCreateCity(uid)
	if main.Gold < ezfyNewCityGoldCost {
		resp.ParamError(c, fmt.Sprintf("建造新城需要%d黄金", ezfyNewCityGoldCost))
		return
	}
	main.Gold -= ezfyNewCityGoldCost
	h.saveCityRes(&main)
	city := model.EzfyCity{
		UserID: uid, Name: fmt.Sprintf("新城%d,%d", req.X, req.Y),
		Feelings: 80, TaxRate: 20, Pop: 0, PopMax: 100,
		Gold: 20000, Food: 5000, Steel: 5000, Oil: 5000, Rare: 5000,
		GoldCap: 1000000, FoodCap: 100000, SteelCap: 100000, OilCap: 100000, RareCap: 100000,
		CityLevel: 1, LastTime: time.Now().UnixMilli(), X: req.X, Y: req.Y,
		WareFood: 25, WareSteel: 25, WareOil: 25, WareRare: 25,
	}
	h.DB.Create(&city)
	h.initBuilding(city.ID, 1, 1)
	h.initBuilding(city.ID, 2, 1)
	h.initBuilding(city.ID, 3, 1)
	kind := "平原"
	extra := ""
	if isSea {
		kind = "海洋"
		extra = "\n该城为【海城】: 可建造航海协会并训练海军。"
	}
	h.addReport(uid, 5, "新城建成",
		fmt.Sprintf("花费%d黄金在%s(%d,%d)建造了新城[%s]\n新城自带基础建筑: 市政厅/民居/农田(1级), 可到[城市列表]切换操作。%s",
			ezfyNewCityGoldCost, kind, req.X, req.Y, city.Name, extra))
	resp.OK(c, gin.H{"msg": "新城建成", "city": city})
}

// RenameCity 城市改名
func (h *EzfyHandler) RenameCity(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		CityId int64  `json:"city_id"`
		Name   string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	city := h.bodyCity(uid, req.CityId)
	if city == nil {
		resp.ParamError(c, "城市不存在")
		return
	}
	name := trimSpace(req.Name)
	if name == "" || len([]rune(name)) > 20 {
		resp.ParamError(c, "城市名长度1-20字")
		return
	}
	h.DB.Model(&model.EzfyCity{}).Where("id = ?", city.ID).Update("name", name)
	resp.OK(c, gin.H{"msg": "改名成功"})
}

// SetTax 设置税率
func (h *EzfyHandler) SetTax(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		CityId  int64 `json:"city_id"`
		TaxRate int   `json:"tax_rate"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	city := h.bodyCity(uid, req.CityId)
	if req.TaxRate < 0 || req.TaxRate > 100 {
		resp.ParamError(c, "税率范围0-100")
		return
	}
	h.DB.Model(&model.EzfyCity{}).Where("id = ?", city.ID).Update("tax_rate", req.TaxRate)
	resp.OK(c, gin.H{"msg": "税率已调整"})
}

// Convene 召集人口（黄金召集不受民居上限限制）
func (h *EzfyHandler) Convene(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		CityId int64 `json:"city_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	city := h.bodyCity(uid, req.CityId)
	if city == nil {
		resp.ParamError(c, "城市不存在")
		return
	}
	h.calcResource(city)
	if city.Gold < ezfyConveneGoldCost {
		resp.ParamError(c, fmt.Sprintf("黄金不足, 召集10万人口需要%d黄金", ezfyConveneGoldCost))
		return
	}
	city.Gold -= ezfyConveneGoldCost
	city.Pop += ezfyConvenePopGain
	h.saveCityRes(city)
	h.DB.Model(&model.EzfyCity{}).Where("id = ?", city.ID).Update("pop", city.Pop)
	resp.OK(c, gin.H{"msg": fmt.Sprintf("召集成功, 人口+%d", ezfyConvenePopGain)})
}

// Placate 安抚民心（花费黄金降低民怨）
func (h *EzfyHandler) Placate(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		CityId int64 `json:"city_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	city := h.bodyCity(uid, req.CityId)
	if city == nil {
		resp.ParamError(c, "城市不存在")
		return
	}
	h.calcResource(city)
	if city.Grievance <= 0 {
		resp.ParamError(c, "民怨为0, 无需安抚")
		return
	}
	cost := int64(city.Grievance) * 100
	if city.Gold < cost {
		resp.ParamError(c, fmt.Sprintf("黄金不足, 安抚需要%d黄金(民怨×100)", cost))
		return
	}
	city.Gold -= cost
	city.Feelings += city.Grievance
	if city.Feelings > 100 {
		city.Feelings = 100
	}
	city.Grievance = 0
	h.saveCityRes(city)
	h.DB.Model(&model.EzfyCity{}).Where("id = ?", city.ID).
		Updates(map[string]interface{}{"feelings": city.Feelings, "grievance": 0})
	resp.OK(c, gin.H{"msg": "安抚成功, 民怨清零, 民心回升"})
}

// AbandonWildland 放弃野地
func (h *EzfyHandler) AbandonWildland(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		WildlandId int64 `json:"wildland_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	var w model.EzfyWildland
	if err := h.DB.Where("id = ? AND city_id IN (SELECT id FROM ezfy_city WHERE user_id = ?)", req.WildlandId, uid).First(&w).Error; err != nil {
		resp.ParamError(c, "野地不存在")
		return
	}
	h.DB.Delete(&w)
	resp.OK(c, gin.H{"msg": "已放弃该野地"})
}

// Resources 资源详情
func (h *EzfyHandler) Resources(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		CityId int64 `json:"city_id"`
	}
	_ = c.ShouldBindJSON(&req)
	city := h.getOrCreateCity(uid)
	if req.CityId > 0 {
		if cc := h.cityOf(uid, req.CityId); cc != nil {
			city = *cc
		}
	}
	h.calcResource(&city)
	resp.OK(c, gin.H{"city": city, "calc": h.getResourceCalc(&city)})
}

// ============ 通用小工具 ============

func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

// parseGroups 解析部队JSON [{"troopId":1,"count":100},...]；兼容 "1:100,2:50" 战果格式
type ezfyUnitGroup struct {
	TroopId int   `json:"troopId"`
	Count   int64 `json:"count"`
}

func parseGroups(s string) []ezfyUnitGroup {
	groups := []ezfyUnitGroup{}
	if s == "" {
		return groups
	}
	if err := json.Unmarshal([]byte(s), &groups); err == nil {
		return groups
	}
	for _, part := range strings.Split(s, ",") {
		kv := strings.Split(part, ":")
		if len(kv) == 2 {
			tid, err1 := strconv.Atoi(trimSpace(kv[0]))
			cnt, err2 := strconv.ParseInt(trimSpace(kv[1]), 10, 64)
			if err1 == nil && err2 == nil && tid > 0 && cnt > 0 {
				groups = append(groups, ezfyUnitGroup{TroopId: tid, Count: cnt})
			}
		}
	}
	return groups
}

func groupsJSON(groups []ezfyUnitGroup) string {
	b, _ := json.Marshal(groups)
	return string(b)
}
