package handler

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"regexp"
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

// 二战风云 核心玩法：进入游戏/城池/建筑/资源懒结算/造兵/伤兵/科技

const (
	ezfyFactoryBuildingID = 14 // 军工厂（★ 不限数量，只受军事区建筑上限约束）
	// ★ 建筑数量上限（军事区/资源区各 33、民居 10）已迁到 ezfy_cfg_limit 表，
	//   管理端「二战风云 → 建筑上限配置」可维护，见 ezfyLimit()。
	ezfyConveneFoodCost   = 100000                 // 召集人口消耗粮食
	ezfyConvenePopGain    = 100000                 // 召集获得人口
	ezfyNewCityGoldCost   = 100000                 // 平原起新城消耗黄金
	ezfyOilDivGrid        = 300                    // 出征耗油: 每格耗油 = 总兵力/300
	ezfyDispatchPeriod    = int64(12 * 3600 * 1000) // 常驻采集结算一期 12 小时
	ezfyTreasureExtraPct  = 20                     // 每期在保底 1 件宝物的基础上, 额外 1 件概率%
	ezfyCommandCarryPct   = 10                     // 指挥艺术: 出征携带上限+%/级
	ezfyMaxUpgradeSeconds = 10                     // 一键满级: 每级升级时间(秒)
	ezfyDeserterRate      = 30                     // 守军战败溃逃比例%
	ezfyWarDelayHours     = 24                     // 宣战生效延迟(小时)
	ezfyWarDurationHours  = 48                     // 宣战有效期(小时)
	// ★ 第九轮：取消训练手续费（%），按常见游戏取 10%
	ezfyCancelTrainFeePct = 10
	// ★ 第九轮：军官忠诚 —— 派遣不再扣，只有打败仗才扣（见 ezfy_battle.go）
	ezfyLoyaltyOnDefeat = 3 // 败仗基础扣忠心
	// ★ 建筑图纸道具 cfg_id（ezfy_cfg_item 表 ID=10，ItemType=6），升级到 10 级及以上必需
	ezfyBlueprintItemID = 10
	// ★ 司令部兵种战斗配置「防守」第三种状态：不参与防御（被攻击时防御战斗兵种不含它）
	ezfyDefMoveNone = -1
)

var ezfyRequirePattern = regexp.MustCompile(`([^()（）]+)[（(]\s*(\d+)\s*级?\s*[）)]`)

// 科技研究所需科研中心等级
var ezfyTechAcademy = map[int]int{
	1: 1, 2: 1, 3: 1, 4: 1, 5: 1, 6: 2, 9: 3, 7: 3, 12: 3,
	13: 4, 10: 4, 8: 4, 11: 5, 19: 5, 18: 6, 14: 6, 21: 6,
	15: 7, 16: 8, 20: 9, 17: 10,
}

// EzfyHandler 二战风云处理器。
//
// ★★ processing 是「骚结算重入守卫」（2026-09-21 线上性能事故沉淀）：
// refreshCity → processOrders → processArrive → refreshCity 是一条天然环：
// 战斗结算时若懒结算防守方，而防守方也有到期订单、且该订单又指向我方，
// 就会互相自喂 —— CPU 打满、订单无限写库。原来的守卫只判断
// `target.UserID == uid`，覆盖不了「A↔B 互相打」和「同一轮订单未落库被重入」。
// processOrders / refreshCity 进函数前抢锁，已在处理中的 uid 直接跳过。
type EzfyHandler struct {
	DB *gorm.DB
	// processing 记录「正在被本 goroutine 懒结算的 uid」，防止订单结算递归重入
	processing sync.Map
}

// enterProcess 标记 uid 进入结算；返回 false 表示已在结算中（调用方应立即返回）。
func (h *EzfyHandler) enterProcess(uid uint) bool {
	_, loaded := h.processing.LoadOrStore(uid, struct{}{})
	return !loaded
}

// exitProcess 结算结束，释放标记。
func (h *EzfyHandler) exitProcess(uid uint) { h.processing.Delete(uint(uid)) }

func (h *EzfyHandler) cfgs() {
	ezfyCfg.load(h.DB)
	// 一次性迁移：旧版「建在海洋上」的海城 → 沿海平原（幂等，进程内只跑一次）
	ezfySeaMigrateOnce.Do(func() { ezfyMigrateSeaCities(h.DB) })
	// 一次性迁移：科技从「按城各存」合并为「所有城池公用」（幂等，见 ezfy_tech_shared.go）
	ezfySharedTechOnce.Do(func() { ezfyMigrateSharedTech(h.DB) })
	// 一次性迁移：存量战报「野地N级」→ 具体地形名（幂等，见 ezfy_migrate_report.go）
	ezfyReportMigrateOnce.Do(func() { ezfyMigrateReportTitles(h.DB) })
	// 一次性迁移：城防兵超城墙容量 → 按比例缩回（幂等，见 ezfy_migrate_troop_cap.go）
	ezfyTroopCapOnce.Do(func() { ezfyMigrateTroopCap(h.DB) })
}

// cfgsReload 强制重载配置缓存。管理端改过 ezfy_cfg_* 后必须调它，
// 否则进程内的缓存还是旧值，玩家端要重启才看得到改动。
func (h *EzfyHandler) cfgsReload() {
	ezfyCfg.reload(h.DB)
}

// ============ 玩家/城市基础 ============

// ensureProfile 懒创建玩家档案（声望/阵营）
func (h *EzfyHandler) ensureProfile(uid uint) model.EzfyProfile {
	var p model.EzfyProfile
	if err := h.DB.Where("user_id = ?", uid).First(&p).Error; err == nil {
		// 老数据补游戏ID（首次 = 家园ID）
		if p.GameUID == 0 {
			p.GameUID = int64(uid)
			h.DB.Model(&model.EzfyProfile{}).Where("id = ?", p.ID).Update("game_uid", p.GameUID)
		}
		return p
	}
	var u model.User
	nickname := ""
	if err := h.DB.Select("nickname").First(&u, uid).Error; err == nil {
		nickname = u.Nickname
	}
	// ★ 游戏ID 首次 = 家园ID，之后永不随家园ID变化
	p = model.EzfyProfile{UserID: uid, GameUID: int64(uid), Nickname: nickname, Prestige: 0, Camp: 1}
	h.DB.Create(&p)
	return p
}

// currentCity 当前操作的城市：优先 profile.current_city_id，无效时回落 id 最小的主城
func (h *EzfyHandler) currentCity(uid uint) model.EzfyCity {
	var p model.EzfyProfile
	if err := h.DB.Where("user_id = ?", uid).First(&p).Error; err == nil && p.CurrentCityId > 0 {
		var city model.EzfyCity
		if err := h.DB.Where("id = ? AND user_id = ?", p.CurrentCityId, uid).First(&city).Error; err == nil {
			return city
		}
	}
	var city model.EzfyCity
	if err := h.DB.Where("user_id = ?", uid).Order("id ASC").First(&city).Error; err == nil {
		return city
	}
	return h.createMainCity(uid)
}

// mainCity 主城（id 最小，建城扣费/城市列表基准用，不受切换影响）
func (h *EzfyHandler) mainCity(uid uint) model.EzfyCity {
	var city model.EzfyCity
	if err := h.DB.Where("user_id = ?", uid).Order("id ASC").First(&city).Error; err == nil {
		return city
	}
	return h.createMainCity(uid)
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
	before := ezfyRankName(p.Prestige)
	after := ezfyRankName(p.Prestige + amount)
	h.DB.Model(&model.EzfyProfile{}).Where("id = ?", p.ID).
		Update("prestige", p.Prestige+amount)
	// ★ 军衔晋升写一条系统消息（用户要求：首页世界聊天要能看到「恭喜玩家晋升XX」）
	if after != before {
		h.ezfySysChat("恭喜玩家 %s 军衔晋升至 %s！", h.ezfyProfileName(uid), after)
	}
}

// getOrCreateCity 懒创建主城（随机平原空位，初始建筑 市政厅/民居/农田 各1级）
// createMainCity 建主城（首次进游戏）
func (h *EzfyHandler) createMainCity(uid uint) model.EzfyCity {
	pos := h.findFreePos()
	city := model.EzfyCity{
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
	// 当前城市指向主城
	h.DB.Model(&model.EzfyProfile{}).Where("user_id = ?", uid).
		Update("current_city_id", int64(city.ID))
	return city
}

// getOrCreateCity 当前操作的城市（分城切换后即切到那座城）
func (h *EzfyHandler) getOrCreateCity(uid uint) model.EzfyCity {
	return h.currentCity(uid)
}

func (h *EzfyHandler) initBuilding(cityId uint, buildingId, level int) {
	b := model.EzfyCityBuilding{CityId: int64(cityId), BuildingId: buildingId, Level: level, Status: 0}
	h.DB.Create(&b)
}

// findFreePos 新玩家首次进游戏的落点
//
// ★ 第十二轮：默认落在**欧洲**（ezfyDefaultMoveContinent），
//
//	内测玩家互相离得近才打得起仗（用户规则：「新玩家 默认 建城市也是默认欧洲城市」）。
//	欧洲满员时按「亚洲 → 非洲 → 北美洲 → 南美洲 → 大洋洲 → 南极洲」依次兜底，
//	最后再退回全世界随机（保证永远建得出城，不会卡住新玩家）。
func (h *EzfyHandler) findFreePos() [2]int {
	order := []int{ezfyDefaultMoveContinent}
	for _, a := range ezfyMoveAreas {
		if a.ID != ezfyDefaultMoveContinent {
			order = append(order, a.ID)
		}
	}
	for _, continent := range order {
		if x, y, ok := h.findFreePosInContinent(continent, false); ok {
			return [2]int{x, y}
		}
	}
	// 全世界兜底（正常情况下走不到这里）
	for i := 0; i < 2000; i++ {
		x := rand.Intn(ezfyWorldSize)
		y := rand.Intn(ezfyWorldSize)
		t := ezfyTerrainEx(x, y)
		if t != 1 && t != ezfyTerrainCoastalPlain {
			continue
		}
		var n int64
		h.DB.Model(&model.EzfyCity{}).Where("x = ? AND y = ?", x, y).Count(&n)
		if n == 0 {
			return [2]int{x, y}
		}
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
	m, r := h.areaCounts(cityId)
	return m + r
}

// areaCounts 分别统计「军事区」与「资源区」的建筑数量。
//
// ★ 第九轮用户规则：军事区与资源区数量上限**分开**，各 33（管理端可维护，见 ezfy_cfg_limit）。
// 分区判据与前端 buildZone 一致：军事区 = type 2/3/4，资源区 = type 1。
func (h *EzfyHandler) areaCounts(cityId uint) (military, resource int) {
	for _, b := range h.buildingList(cityId) {
		c := ezfyCfg.building(b.BuildingId)
		if c == nil {
			continue
		}
		if c.Type == 1 {
			resource++
		} else if c.Type == 2 || c.Type == 3 || c.Type == 4 {
			military++
		}
	}
	return
}

// ezfyBuildingMaxLevel 建筑等级上限（用户规则，覆盖配置表 max_level）
//
//	市政厅(1)            → 10
//	民居(2)              → 最多比市政厅高 1 级；市政厅到 10 级时民居可升到 12
//	参谋部(10)/司令部(13) → 12
//	其他                 → 10
func ezfyBuildingMaxLevel(buildingId, hallLevel int) int {
	switch buildingId {
	case 1:
		return 10
	case 2:
		if hallLevel >= 10 {
			return 12
		}
		if hallLevel+1 >= 12 {
			return 12
		}
		return hallLevel + 1
	case 10, 13:
		return 12
	}
	return 10
}

// hallLevelOf 该城市市政厅当前等级
func (h *EzfyHandler) hallLevelOf(cityId uint) int {
	return h.buildingLevel(cityId, 1)
}

// buildingMaxLevel 该城该建筑的等级上限。
//
// ★ 第九轮用户规则**优先于配置表**：等级上限是玩法规则，不是数值配置。
//
//	配置表 ezfy_cfg_building.max_level 只作参考（历史上民居被写成 10，
//	会把「民居最多比市政厅高1级」这条规则整个压掉，所以这里不再取它的值）。
func (h *EzfyHandler) buildingMaxLevel(cityId uint, buildingId int) int {
	return ezfyBuildingMaxLevel(buildingId, h.hallLevelOf(cityId))
}

// techMap 某城的科技等级表。
//
// ★ 第九轮：科技**所有城池公用** —— 内部先换算成「科技城」(玩家主城)，
// 所以所有调用点（结算/展示/加成）自动变成全账号共用，不用逐个改。
func (h *EzfyHandler) techMap(cityId uint) map[int]int {
	var list []model.EzfyCityTech
	h.DB.Where("city_id = ?", h.techCityId(cityId)).Find(&list)
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

// ezfySafeAdd 安全加法：结果恒落在 [0, max]，绝不溢出成负数。
//
// ★ 2026-09-23 线上事故（玩家总兵力 -8843547888967622000）的最后一道保险：
//
//	即使上层某处漏了校验，兵力也绝不会被写到 int64 溢出。
func ezfySafeAdd(a, b, max int64) int64 {
	if b > 0 {
		if a > max-b { // max-b 不会溢出（b > 0 且 max 恒正）
			return max
		}
		return a + b
	}
	if c := a + b; c > 0 { // b <= 0 时 a+b 只会变小，不会上溢
		return c
	}
	return 0
}

// cityTroopTotal 城市当前兵力合计
//
// 口径 = 城内现有部队 + 训练队列里还没出厂的新兵（与前端「总兵力」展示口径一致）。
// 城防(type 4)也计入 —— 它同样存在 ezfy_city_troop 里，同样会溢出。
func (h *EzfyHandler) cityTroopTotal(cityId uint) int64 {
	var total int64
	for _, t := range h.troopList(cityId) {
		if t.Count > 0 {
			total = ezfySafeAdd(total, t.Count, ezfyTroopMaxCfg())
		}
	}
	var qs []model.EzfyTrainQueue
	h.DB.Where("city_id = ? AND status = 0", cityId).Find(&qs)
	for _, q := range qs {
		if q.Count > 0 {
			total = ezfySafeAdd(total, q.Count, ezfyTroopMaxCfg())
		}
	}
	return total
}

// checkTroopCap 训练 / 伤兵恢复前的「兵力上限」校验。
//
// ★ 2026-09-23 用户要求：「超过限制不能训练，提示超过限额」。
//
//	口径：当前兵力(城内 + 训练队列) + 本次要加的量 > troop_max → 拒绝。
//	返回空串表示通过，否则返回可直接展示给玩家的提示文案。
func (h *EzfyHandler) checkTroopCap(cityId uint, add int64) string {
	if add <= 0 {
		return "数量错误"
	}
	max := ezfyTroopMaxCfg()
	cur := h.cityTroopTotal(cityId)
	if cur >= max || add > max-cur {
		return fmt.Sprintf("超过限额(单城兵力上限%d, 当前%d)", max, cur)
	}
	return ""
}

func (h *EzfyHandler) addTroop(cityId uint, troopId int, count int64) {
	if count == 0 {
		return
	}
	max := ezfyTroopMaxCfg()
	var t model.EzfyCityTroop
	if err := h.DB.Where("city_id = ? AND troop_id = ?", cityId, troopId).First(&t).Error; err != nil {
		// ★ 新建行也夹取：负数直接不建行，超上限截断（防溢出兜底）
		n := ezfySafeAdd(0, count, max)
		if n <= 0 {
			return
		}
		t = model.EzfyCityTroop{CityId: int64(cityId), TroopId: troopId, Count: n}
		h.DB.Create(&t)
		return
	}
	h.DB.Model(&model.EzfyCityTroop{}).Where("id = ?", t.ID).
		Update("count", ezfySafeAdd(t.Count, count, max))
}

func (h *EzfyHandler) addWounded(cityId uint, troopId, wtype int, count int64) {
	if count <= 0 {
		return
	}
	max := ezfyTroopMaxCfg()
	var w model.EzfyWounded
	if err := h.DB.Where("city_id = ? AND troop_id = ? AND type = ?", cityId, troopId, wtype).First(&w).Error; err != nil {
		w = model.EzfyWounded{CityId: int64(cityId), TroopId: troopId, Type: wtype,
			Count: ezfySafeAdd(0, count, max)}
		h.DB.Create(&w)
		return
	}
	// ★ 2026-09-23：伤兵数量同样夹取，防止「恢复时一次性加回城里」把兵力撑溢出。
	h.DB.Model(&model.EzfyWounded{}).Where("id = ?", w.ID).
		Update("count", ezfySafeAdd(w.Count, count, max))
}

// filterExpiredWounded 从「已经查出来的伤兵列表」里剔除过期项，并把过期记录删库。
//
// ★ 2026-09-23 用户要求：「伤兵 5 天不救治直接消失」。
//
//	口径按「最后一次入营时间」(ezfy_wounded.updated_at) 算 ——
//	addWounded 累加时会自动刷新 updated_at，所以玩家持续有伤兵入营会顺延；
//	超过 wound_expire_days（默认 5 天）既没恢复也没新增的，直接消失。
//
// ⚠️ 刻意**复用调用方已查到的列表**做内存过滤，不额外发 SELECT；
// 只有确实存在过期项时才发一次按主键的 DELETE。这样即使被首页轮询接口
// （View，30s 一次）调用，稳态下也几乎零额外开销 —— 性能红线。
func (h *EzfyHandler) filterExpiredWounded(list []model.EzfyWounded) []model.EzfyWounded {
	days := ezfyWoundExpireDaysCfg()
	if days <= 0 || len(list) == 0 {
		return list
	}
	deadline := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	kept := make([]model.EzfyWounded, 0, len(list))
	var ids []uint
	for _, w := range list {
		if w.UpdatedAt.Before(deadline) {
			ids = append(ids, w.ID)
			continue
		}
		kept = append(kept, w)
	}
	if len(ids) > 0 {
		h.DB.Where("id IN ?", ids).Delete(&model.EzfyWounded{})
	}
	return kept
}

func (h *EzfyHandler) wildlandList(cityId uint) []model.EzfyWildland {
	var list []model.EzfyWildland
	h.DB.Where("city_id = ?", cityId).Find(&list)
	return list
}

// isSeaCity 是否海城
//
// ★ 用户规则：海城不是建在「海洋」上，而是建在「沿海平原」上（见 ezfy_geo.go）。
// 只有海城能训练海军、建航海协会。
func (h *EzfyHandler) isSeaCity(city *model.EzfyCity) bool {
	return ezfyTerrainEx(city.X, city.Y) == ezfyTerrainCoastalPlain
}

// isCoastalCity 是否沿海（自己或邻域靠海，航海协会用）
func (h *EzfyHandler) isCoastalCity(city *model.EzfyCity) bool {
	if ezfyHasSeaNeighbor(city.X, city.Y) {
		return true
	}
	return ezfyTerrain(city.X, city.Y) == 8
}

// cityKind 城市类型文案
//
// ★ 用户要求统一口径：建在沿海平原上的叫「沿海城市」，其余叫「内陆城市」。
//
//	（原来叫「海城 / 陆地城市」，与城市列表、城市状态页两处不一致）
func (h *EzfyHandler) cityKind(city *model.EzfyCity) string {
	if h.isSeaCity(city) {
		return "沿海城市"
	}
	return "内陆城市"
}

// ============ 懒结算五连（复刻 GameServiceImpl checkBuildingDone/collectTrainQueue/calcResource/processOrders） ============

// refreshCity 每次进入游戏接口前统一懒结算
//
// ★ 军官工资所需的数据由 calcResource 内部**按需惰性加载**（见 officerSalaryOfCity），
//
//	所以这里不必预先查军官表：只有「确实要按小时扣工资」的那一次才查一次库，
//	且同一请求内复用（request 级缓存），不会每个 callee 重复查。
func (h *EzfyHandler) refreshCity(uid uint, city *model.EzfyCity) {
	h.checkBuildingDone(city)
	h.checkTechDone(city)
	h.collectTrainQueue(city)
	h.calcResource(city)
	h.processOrders(uid)
}

// refreshCityWithOfficers 与 refreshCity 相同，但把「已经取到的军官列表」传给
// calcResource 算工资，从而**省掉一次军官查询**。
//
// ★ 高频接口（View 等，尤其是被前端轮询的首页）应当走这个版本：
//
//	officers := h.officerList(city.ID)
//	h.refreshCityWithOfficers(uid, &city, officers)
//
// 普通接口直接用 refreshCity 即可（calcResource 会兜底查一次，同样只查一次）。
func (h *EzfyHandler) refreshCityWithOfficers(uid uint, city *model.EzfyCity,
	officers []model.EzfyOfficer) {
	h.checkBuildingDone(city)
	h.checkTechDone(city)
	h.collectTrainQueue(city)
	h.calcResource(city, officers)
	h.processOrders(uid)
}

func (h *EzfyHandler) checkBuildingDone(city *model.EzfyCity) {
	now := time.Now().UnixMilli()
	for _, b := range h.buildingList(city.ID) {
		if b.Status != 0 && now >= b.EndTime {
			b.Level++
			h.addPrestige(city.UserID, b.Level*10)
			cfg := ezfyCfg.building(b.BuildingId)
			if b.StartTime == 0 && cfg != nil && b.Level < h.buildingMaxLevel(city.ID, b.BuildingId) {
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

// calcResource 资源按小时懒结算（民心/民怨/科技/道具增产/野地产出/军队耗粮/军官工资）
//
// ★★ 性能红线（2026-09-21 线上事故）：本函数是**所有接口的必经懒结算**，
// 军官工资那一项**绝不能**在这里直接查库。之前写的是
//
//	gold -= officerSalaryPerHour(city.ID)   // → officerList → 2 条 SQL
//
// 每请求凭空多打 2 条 SQL，配合前端 30s 轮询把线上 1核1G 的 IO/CPU 打满。
//
// 现在的口径：工资用 officerSalaryOf 做**纯内存**计算。
// 军官数据由调用方通过可选参数 officers 传入；不传则本函数**自己查一次**
// （只在这一处、且只在真正要结算时查），保证「工资一定扣得到、且最多查一次」。
// 高频路径（View / Officers / battle）请显式传 officers 以复用已有查询结果。
func (h *EzfyHandler) calcResource(city *model.EzfyCity, officers ...[]model.EzfyOfficer) {
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
	// ★ 资源建筑(农田3/炼钢4/石油5/稀矿6)自带「增加容量」，需累加到对应资源上限
	//   （此前只取仓库容量 resCap，农田/炼钢/石油/稀矿的容量没算进去 → 上限 bug）
	var foodCap, steelCap, oilCap, rareCap int64
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
				foodCap += lv.Capacity
			case 4:
				steelProd += prod
				steelCap += lv.Capacity
			case 5:
				oilProd += prod
				oilCap += lv.Capacity
			case 6:
				rareProd += prod
				rareCap += lv.Capacity
			case 12:
				resCap += lv.Capacity
			}
		} else if b.BuildingId == 1 {
			goldCap = lv.Capacity
		}
	}
	city.PopMax = popMax
	if resCap <= 0 {
		resCap = 100000
	}
	city.FoodCap = resCap + foodCap
	city.SteelCap = resCap + steelCap
	city.OilCap = resCap + oilCap
	city.RareCap = resCap + rareCap
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
	// ★ 用户要求「耗粮开关也做个吧，默认开」→ 关掉时城内军队每小时不扣粮。
	var troopFoodCost int64
	if ezfyFoodUpkeepOn() {
		for tid, count := range h.troopMap(city.ID) {
			if cfg := ezfyCfg.troop(tid); cfg != nil {
				troopFoodCost += int64(cfg.FoodKeep) * count
			}
		}
		troopFoodCost = troopFoodCost * int64(100-techSupply*2) / 100
		troopFoodCost = int64(float64(troopFoodCost) * hours)
	}

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
	// ★ 军官工资：每名军官每小时消耗「等级 × ezfy_cfg_limit.officer_salary_per_level」黄金。
	//   与「军队耗粮」同一套懒结算口径 —— 按小时累计，离线期间照样扣。
	//   用户反馈「军官是消耗黄金的，黄金现在消耗 0」，这就是那笔消耗。
	//
	//   ⚡ 性能：officers 由调用方传入时（View / Officers / 战斗结算）用纯内存计算，
	//   零额外 SQL；未传时**兜底查一次**，保证工资一定扣得到。
	//   两种情形下本函数最多都只产生 1 次军官查询 —— 绝不会像事故版本那样重复触发。
	//   注意兜底只在真正需要结算（hours 有意义）时才走，避免空转。
	per := int64(ezfyOfficerSalaryPerLvCfg())
	if per > 0 {
		var salary int64
		if len(officers) > 0 {
			salary = officerSalaryOf(officers[0])
		} else {
			salary = officerSalaryOf(h.officerList(city.ID))
		}
		gold -= int64(float64(salary) * hours)
	}
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
	// ★ 2026-09-23：资源 / 人口 / 仓储上限统一夹取到 [0, ezfyResSafeMax]。
	//   原有的「按仓储上限截断」逻辑在上面（资源累加处）已经生效，这里只是最后兜一层底，
	//   保证**任何**路径都不会把资源字段写溢出成负数（线上事故的同类风险）。
	city.Food = ezfyClampRes(city.Food)
	city.Steel = ezfyClampRes(city.Steel)
	city.Oil = ezfyClampRes(city.Oil)
	city.Rare = ezfyClampRes(city.Rare)
	city.Gold = ezfyClampRes(city.Gold)
	city.Pop = ezfyClampRes(city.Pop)
	city.PopMax = ezfyClampRes(city.PopMax)
	city.FoodCap = ezfyClampRes(city.FoodCap)
	city.SteelCap = ezfyClampRes(city.SteelCap)
	city.OilCap = ezfyClampRes(city.OilCap)
	city.RareCap = ezfyClampRes(city.RareCap)
	city.GoldCap = ezfyClampRes(city.GoldCap)
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

	// ★ 基础产量 = 建筑基础产量 × 科技加成。
	//   原来科技只体现在「加成产量」里，玩家看完科技页再回资源详情，看到基础产量纹丝不动，
	//   反馈就是「科技没有生效」。现在科技直接并进基础产量，一眼可见。
	foodBaseTech := foodBase * int64(100+techFood*10) / 100
	steelBaseTech := steelBase * int64(100+techSteel*10) / 100
	oilBaseTech := oilBase * int64(100+techOil*10) / 100
	rareBaseTech := rareBase * int64(100+techRare*10) / 100

	// ★ 与 calcResource 对齐：开工率 → 市长加成 → 民心
	//   （原来详情页漏了开工率与市长加成，导致「详情页的数字」和「实际每小时产量」对不上）
	rateFood := int64(ezfyRate(city.RateFood))
	rateSteel := int64(ezfyRate(city.RateSteel))
	rateOil := int64(ezfyRate(city.RateOil))
	rateRare := int64(ezfyRate(city.RateRare))
	mayor := int64(h.mayorBonusPct(city.ID))
	applyProd := func(base, rate int64) int64 {
		v := base * rate / 100
		if mayor > 0 {
			v = v * (100 + mayor) / 100
		}
		return int64(float64(v) * morale)
	}
	foodProd := applyProd(foodBaseTech, rateFood)
	steelProd := applyProd(steelBaseTech, rateSteel)
	oilProd := applyProd(oilBaseTech, rateOil)
	rareProd := applyProd(rareBaseTech, rateRare)
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
	// ★ 耗粮开关关掉时这里也要显示 0，否则界面写着「每小时耗粮 N」，实际却不扣
	if ezfyFoodUpkeepOn() {
		for tid, count := range h.troopMap(city.ID) {
			if cfg := ezfyCfg.troop(tid); cfg != nil {
				troopFood += int64(cfg.FoodKeep) * count
			}
		}
	}
	// ★ 原始耗粮（未扣补给技巧）与实扣耗粮都下发：
	//   用户反馈「补给技巧Lv10 -20% 没实现啊」——其实公式是对的（raw×80%），
	//   只是界面只显示「实扣值 + 一个 -20% 标签」，看起来像是没扣。两个值都给出才不歧义。
	troopFoodRaw := troopFood
	troopFood = troopFood * int64(100-techSupply*2) / 100

	item := func(stock, cap, base, bonus, consume, total int64, extra gin.H) gin.H {
		m := gin.H{"stock": stock, "cap": cap, "base": base, "bonus": bonus, "consume": consume, "total": total, "store_tech": techStore}
		for k, v := range extra {
			m[k] = v
		}
		return m
	}
	return gin.H{
		"food": item(city.Food, city.FoodCap, foodBaseTech, foodProd-foodBaseTech+wildFood, troopFood, foodProd+wildFood-troopFood,
			gin.H{"tech_prod": techFood, "troop_consume": troopFood, "troop_consume_raw": troopFoodRaw,
				"supply_tech": techSupply, "base_building": foodBase, "rate": rateFood, "mayor_bonus": mayor}),
		"steel": item(city.Steel, city.SteelCap, steelBaseTech, steelProd-steelBaseTech+wildSteel, 0, steelProd+wildSteel,
			gin.H{"tech_prod": techSteel, "base_building": steelBase, "rate": rateSteel, "mayor_bonus": mayor}),
		"oil": item(city.Oil, city.OilCap, oilBaseTech, oilProd-oilBaseTech+wildOil, 0, oilProd+wildOil,
			gin.H{"tech_prod": techOil, "base_building": oilBase, "rate": rateOil, "mayor_bonus": mayor}),
		"rare": item(city.Rare, city.RareCap, rareBaseTech, rareProd-rareBaseTech+wildRare, 0, rareProd+wildRare,
			gin.H{"tech_prod": techRare, "base_building": rareBase, "rate": rateRare, "mayor_bonus": mayor}),
		"gold": item(city.Gold, city.GoldCap, goldBase, goldProd-goldBase+wildGold, 0, goldProd+wildGold,
			gin.H{"tech_prod": 0, "base_building": goldBase, "rate": 100, "mayor_bonus": mayor}),
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
	// ★ 2026-09-23 线上「负数兵力」事故后加的统一防线：
	//   资源落库前一律夹取到 [0, ezfyResSafeMax]，任何调用方都不可能写出负数/溢出值。
	//   内存对象同步回写，保证「库里存的」和「本次请求后续用的」一致。
	city.Gold = ezfyClampRes(city.Gold)
	city.Food = ezfyClampRes(city.Food)
	city.Steel = ezfyClampRes(city.Steel)
	city.Oil = ezfyClampRes(city.Oil)
	city.Rare = ezfyClampRes(city.Rare)
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
	lim := ezfyLimit()
	if buildingId == ezfyFactoryBuildingID {
		// ★ 第九轮用户规则：军工厂**不限数量**，只要军事区建筑上限没到就能一直建。
		//   仅当管理端把 factory_max 配成正数时才限制（默认 0 = 不限）。
		if lim.FactoryMax > 0 && countOfType() >= int64(lim.FactoryMax) {
			return fmt.Sprintf("军工厂最多建造%d个", lim.FactoryMax)
		}
	} else if buildingId == 2 {
		if countOfType() >= int64(lim.HouseMax) {
			return fmt.Sprintf("民居最多建造%d个", lim.HouseMax)
		}
	} else if cfg.Type == 2 || cfg.Type == 3 {
		if countOfType() > 0 {
			return "该建筑已存在"
		}
	}
	// ★ 军事区 / 资源区数量上限**分开**（各 33，管理端可维护）
	mil, res := h.areaCounts(city.ID)
	if cfg.Type == 1 {
		if res >= lim.ResourceMax {
			return fmt.Sprintf("资源区建筑数量已达上限(%d/%d)", res, lim.ResourceMax)
		}
	} else if cfg.Type == 2 || cfg.Type == 3 || cfg.Type == 4 {
		if mil >= lim.MilitaryMax {
			return fmt.Sprintf("军事区建筑数量已达上限(%d/%d)", mil, lim.MilitaryMax)
		}
	}
	// ★ 航海协会：只有沿海城市（沿海平原）能建，内陆城市一律不行
	if buildingId == 19 && !h.isSeaCity(city) {
		return "航海协会只能建在沿海城市(沿海平原上的城市)"
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
	if cfg == nil {
		return "建筑配置缺失"
	}
	// ★ 第九轮等级规则：市政厅10 / 参谋部·司令部·民居12 / 其他10；民居最多比市政厅高1级
	if max := h.buildingMaxLevel(city.ID, b.BuildingId); target > max {
		if b.BuildingId == 2 && h.hallLevelOf(city.ID) < 10 && target == h.hallLevelOf(city.ID)+2 {
			return fmt.Sprintf("民居最多比市政厅高1级（市政厅%d级，民居最多%d级）",
				h.hallLevelOf(city.ID), h.hallLevelOf(city.ID)+1)
		}
		return fmt.Sprintf("该建筑最高%d级", max)
	}
	lv := ezfyCfg.buildingLevel(b.BuildingId, target)
	if lv == nil {
		return "配置缺失"
	}
	// ★ 建筑图纸道具 cfg_id = 10（ItemType=6），不是 6 —— 6 是「训练加速30分钟」。
	//   规则（2026-09-23 用户修正）：**所有建筑 9→10 级**都需图纸；民居(2) 10→11、11→12 也要。
	needBlueprint := target == 10 || (b.BuildingId == 2 && target >= 11)
	if needBlueprint && h.itemCount(city.UserID, ezfyBlueprintItemID) <= 0 {
		return fmt.Sprintf("升级到%d级需要建筑图纸", target)
	}
	if !h.pay(city, lv) {
		return "资源不足"
	}
	if needBlueprint {
		h.consumeItem(city.UserID, ezfyBlueprintItemID)
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
	// ★ 第九轮等级规则（市政厅10 / 参谋部·司令部·民居12 / 其他10，民居 ≤ 市政厅+1）
	maxLv := h.buildingMaxLevel(city.ID, b.BuildingId)
	if b.Level >= maxLv {
		return "该建筑已满级"
	}
	var needFood, needSteel, needOil, needRare, needGold int64
	needBlueprint := 0
	for lv := b.Level + 1; lv <= maxLv; lv++ {
		// ★ 与 upgradeBuilding 同口径：所有建筑 9→10、民居(2) 10→11 / 11→12 需图纸
		if lv == 10 || (b.BuildingId == 2 && lv >= 11) {
			needBlueprint++
		}
		if l := ezfyCfg.buildingLevel(b.BuildingId, lv); l != nil {
			needFood += l.Food
			needSteel += l.Steel
			needOil += l.Oil
			needRare += l.Rare
			needGold += l.Gold
		}
	}
	if needBlueprint > 0 && h.itemCount(city.UserID, ezfyBlueprintItemID) < needBlueprint {
		return fmt.Sprintf("一键满级需要%d张建筑图纸(当前不足)", needBlueprint)
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
	if needBlueprint > 0 {
		h.consumeItemN(city.UserID, ezfyBlueprintItemID, needBlueprint)
	}
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
	// ★ 用户要求：拆除是一级一级拆，而不是直接整栋拆没；降到 0 级才彻底移除
	if b.Level <= 1 {
		h.DB.Delete(&b)
	} else {
		h.DB.Model(&model.EzfyCityBuilding{}).Where("id = ?", b.ID).Update("level", b.Level-1)
	}
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

// defenceSpaceUsed 城防空间已占用 = 已建成的城防 + **训练队列里还没出来的城防**。
//
// ★ 用户反馈「城防可以随便造」的根因：原来只统计已建成的城防，
//
//	于是玩家可以连下多张训练单、每单都不超上限，最后总量突破围墙容量。
//	把队列里的量也算进来才是真正的「占用」。
func (h *EzfyHandler) defenceSpaceUsed(cityId uint) int64 {
	var used int64
	for tid, cnt := range h.troopMap(cityId) {
		if c := ezfyCfg.troop(tid); c != nil && c.Type == 4 {
			used += cnt
		}
	}
	var qs []model.EzfyTrainQueue
	h.DB.Where("city_id = ? AND status = 0", cityId).Find(&qs)
	for _, q := range qs {
		if c := ezfyCfg.troop(q.TroopId); c != nil && c.Type == 4 {
			used += q.Count
		}
	}
	return used
}

func (h *EzfyHandler) trainTroop(city *model.EzfyCity, troopId, count int, split bool) string {
	h.refreshCity(city.UserID, city)
	if count <= 0 {
		return "数量错误"
	}
	cfg := ezfyCfg.troop(troopId)
	if cfg == nil {
		return "兵种不存在"
	}
	// 海军(type 1)只能在沿海城市训练(用户规则: 内陆城市不能训练海军)
	if cfg.Type == 1 && !h.isSeaCity(city) {
		return "海军只能在沿海城市(建在沿海平原上的城市)训练, 内陆城市无法训练海军"
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
	// 人口校验：只跟「正在训练、还没出厂」的兵比 —— 已训练完成的部队不占人口（用户规则）
	// ★ 用户要求「征兵资源消耗开关关了的话，征兵不消耗资源，也无需空闲人口」→
	//   开关关掉时整段跳过（不校验人口、不扣资源）。
	recruitCost := ezfyRecruitCostOn()
	if recruitCost {
		popUsed := h.cityPopUsed(city.ID)
		popAvailable := city.Pop - popUsed
		if cfg.Type != 4 && int64(cfg.Pop)*int64(count) > popAvailable {
			return fmt.Sprintf("人口不足(当前居民%d, 建筑及训练已占用%d, 可用%d); 可召集人口突破民居上限",
				city.Pop, popUsed, popAvailable)
		}
	}
	// ★ 2026-09-23 用户要求「超过限制不能训练，提示超过限额」：
	//   兵力累加没有任何上限，单兵种 count 撑爆 int64 就翻成负数
	//   （线上事故：玩家总兵力 -8843547888967622000）。
	//   这里按 ezfy_cfg_limit.troop_max 统一卡控（城内现有 + 训练队列 + 本次）。
	//   城防(type 4)同样存在 ezfy_city_troop 里、同样会溢出，所以一并卡。
	if msg := h.checkTroopCap(city.ID, int64(count)); msg != "" {
		return msg
	}
	food := cfg.Food * int64(count)
	steel := cfg.Steel * int64(count)
	oil := cfg.Oil * int64(count)
	rare := cfg.Rare * int64(count)
	// 节日活动·造兵打折(福利.txt #4)；★ 征兵资源消耗开关关掉时这里会整体返回 0
	food, steel, oil, rare = h.trainCostWithActivity(food, steel, oil, rare)
	if recruitCost && (city.Food < food || city.Steel < steel || city.Oil < oil || city.Rare < rare) {
		return "资源不足"
	}
	if cfg.Type == 4 {
		wallLevel := h.buildingLevel(city.ID, 7)
		space := int64(0)
		if wall := ezfyCfg.buildingLevel(7, wallLevel); wall != nil {
			space = wall.Capacity
		}
		used := h.defenceSpaceUsed(city.ID)
		if used+int64(count) > space {
			return fmt.Sprintf("城防空间不足(围墙%d级, 上限%d, 已占用%d)", wallLevel, space, used)
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
	// ★ 免费征兵标记：开关关着建的队列，取消时不退还资源（见 EzfyTrainQueue.FreeTrain）
	freeFlag := 0
	if !recruitCost {
		freeFlag = 1
	}
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
			StartTime: now, EndTime: now + int64(cfg.TrainTime)*1000*part, FreeTrain: freeFlag}
		h.DB.Create(&q)
	}
	h.taskProgress(city.UserID, "train_troop", count)
	return ""
}

// troopPop 城市「已占用人口」= **训练队列里还没出厂的新兵**占用的人口。
//
// ★ 用户规则（2026-09-21 明确）：兵确实用人口训练，但**不占用人口位置**。
//
//	即：已训练完成的部队（城内驻军 / 出征在外 / 返航途中）一律不再占人口，
//	    只有「正在训练、还没出厂」的新兵临时占用，出厂进城的瞬间人口就归还。
//
// 背景（用户反馈的 bug）：原实现只统计 ezfy_city_troop（城内部队表），于是
//   - 部队一出征就从城内部队表扣掉 → 占用人口瞬间归零、空闲人口变回满值；
//   - 训练队列里的新兵出厂前完全不占人口。
//     两处口径都不对：前者让「出征=凭空多出人口」，后者让「训练不吃人口」。
//     统一成「只看训练队列」后，空闲人口 = 人口 - 训练中占用，语义唯一。
//
// 注意：城防设施(type 4)不占人口，它走「城防空间」那一套（见 defenceSpaceUsed），
//
//	这里必须排除，否则城防会被重复计一次。
func (h *EzfyHandler) troopPop(cityId uint) int64 {
	// ★ 用户要求「征兵资源消耗开关关了，征兵无需空闲人口」→ 关掉时人口占用恒为 0，
	//   空闲人口 = 人口（与「训练不占人口」的语义一致，界面不会显示「被占满」）。
	if !ezfyRecruitCostOn() {
		return 0
	}
	var qs []model.EzfyTrainQueue
	h.DB.Where("city_id = ? AND status = 0", cityId).Find(&qs)
	pop := int64(0)
	for _, q := range qs {
		if cfg := ezfyCfg.troop(q.TroopId); cfg != nil && cfg.Type != 4 {
			pop += int64(cfg.Pop) * q.Count
		}
	}
	return pop
}

// buildingPop 城市「已占用人口」中的**建筑占用**部分：每栋已建建筑的「占用人口」(lv.Pop) 累加。
//
// ★ 用户反馈（2026-09-23）：建筑的「占用人口」此前完全没计入，导致空闲人口虚高。
//
//	每家建筑（市政厅/资源建筑/军事建筑/城防等）升级后都有「占用人口」，统一在此累加。
func (h *EzfyHandler) buildingPop(cityId uint) int64 {
	pop := int64(0)
	for _, b := range h.buildingList(cityId) {
		if lv := ezfyCfg.buildingLevel(b.BuildingId, b.Level); lv != nil {
			pop += int64(lv.Pop)
		}
	}
	return pop
}

// cityPopUsed 城市「已占用人口」= 建筑占用 + 训练队列占用。
func (h *EzfyHandler) cityPopUsed(cityId uint) int64 {
	return h.buildingPop(cityId) + h.troopPop(cityId)
}

// ezfyWoundHealGoldPer 恢复 1 个该兵种伤兵需要的黄金
//
//	= ceil(兵种总造价 / ezfy_cfg_limit.wound_heal_divisor) × 折扣率，最低 1 黄金。
//
// ★ 用户规则：「伤兵不参与消耗粮食、恢复伤兵需要黄金」——
// 伤兵在营里不耗粮（它们不在 ezfy_city_troop 里，calcResource 的耗粮只算在编部队），
// 但恢复出厂要花钱。用「总造价 / 系数」而不是固定值，是为了让高级兵种恢复更贵。
// ★ 2026-09-23：再乘管理端「伤兵恢复黄金折扣率」(wound_heal_rate)，节假日调低 = 恢复便宜。
func ezfyWoundHealGoldPer(troopId int) int64 {
	t := ezfyCfg.troop(troopId)
	if t == nil {
		return 1
	}
	total := int64(t.Food) + int64(t.Steel) + int64(t.Oil) + int64(t.Rare)
	d := int64(ezfyWoundHealDivisorCfg())
	if d < 1 {
		d = 1
	}
	g := (total + d - 1) / d
	g = int64(float64(g)*ezfyWoundHealRate() + 0.5)
	if g < 1 {
		g = 1
	}
	return g
}

// officerSalaryPerHour 本城军官每小时工资合计（黄金）
//
// ★ 用户反馈「军官是消耗黄金的，黄金现在消耗 0」→ 军官工资。
// 俘虏(IsCaptive=1)不发工资（还没收编，不算自己人）。
//
// ⚠️ 性能红线（2026-09-21 线上事故）：
//
//	本函数会查库（officerList 内部 2 条 SQL），**绝不能**放进 calcResource 之类的
//	高频懒结算路径里 —— calcResource 是所有接口的必经之路，在里面查库 = 每个请求
//	都多打一次 DB，1核1G 的线上库会被打满（IO/CPU 双飙升）。
//	需要「随懒结算一起扣工资」时，请用 officerSalaryOf(list)，把已经取到的
//	军官列表传进去（calcResource 已改为接收 officers 参数）。
func (h *EzfyHandler) officerSalaryPerHour(cityId uint) int64 {
	return officerSalaryOf(h.officerList(cityId))
}

// officerSalaryOf 按给定的军官列表算每小时工资合计（纯内存，不查库）。
//
// ★ 这是「性能红线」要求的安全出口：调用方自己把军官列表取一次，
// 反复算多少次都不会再产生 SQL。calcResource / Officers / View 都走它。
func officerSalaryOf(list []model.EzfyOfficer) int64 {
	var total int64
	per := int64(ezfyOfficerSalaryPerLvCfg())
	for i := range list {
		o := &list[i]
		if o.IsCaptive == 1 {
			continue
		}
		lv := int64(o.Level)
		if lv < 1 {
			lv = 1
		}
		total += lv * per
	}
	return total
}

func (h *EzfyHandler) recoverWounded(city *model.EzfyCity, troopId, wtype int) string {
	var w model.EzfyWounded
	if err := h.DB.Where("city_id = ? AND troop_id = ? AND type = ?", city.ID, troopId, wtype).First(&w).Error; err != nil {
		return "兵营中没有该兵种"
	}
	// ★ 已过期的伤兵不可恢复（列表里早已消失，这里防直接调接口绕过）
	if len(h.filterExpiredWounded([]model.EzfyWounded{w})) == 0 {
		return "兵营中没有该兵种"
	}
	if w.Count <= 0 {
		return "兵营中没有该兵种"
	}
	// ★ 2026-09-23 用户要求：「恢复的数量导致负数的情况也卡控，不能恢复」。
	//   恢复 = 往城里加兵，所以和训练共用同一个兵力上限校验。
	if msg := h.checkTroopCap(city.ID, w.Count); msg != "" {
		return msg
	}
	h.calcResource(city)
	cost := ezfyWoundHealGoldPer(w.TroopId) * w.Count
	if city.Gold < cost {
		return fmt.Sprintf("黄金不足: 恢复%d个需要%d黄金, 当前只有%d", w.Count, cost, city.Gold)
	}
	city.Gold -= cost
	h.saveCityRes(city)
	h.addTroop(city.ID, w.TroopId, w.Count)
	h.DB.Delete(&w)
	return ""
}

func (h *EzfyHandler) recoverAllWounded(city *model.EzfyCity, wtype int) string {
	var list []model.EzfyWounded
	h.DB.Where("city_id = ? AND type = ?", city.ID, wtype).Find(&list)
	// ★ 过期的伤兵不可恢复（顺便把过期记录清掉）
	list = h.filterExpiredWounded(list)
	if len(list) == 0 {
		return "兵营中空空如也"
	}
	// ★ 2026-09-23：一键恢复 = 一次性把全部伤兵加回城里，所以按「总量」做一次上限校验
	//   （逐个校验会漏掉「每个都不超、加起来超了」的情况，且叠加后同样能溢出）。
	max := ezfyTroopMaxCfg()
	cur := h.cityTroopTotal(city.ID)
	var pending int64
	for _, w := range list {
		if w.Count > 0 {
			pending = ezfySafeAdd(pending, w.Count, max)
		}
	}
	if pending <= 0 {
		return "兵营中空空如也"
	}
	if cur >= max || pending > max-cur {
		return fmt.Sprintf("超过限额(单城兵力上限%d, 当前%d, 待恢复%d)", max, cur, pending)
	}
	h.calcResource(city)
	var cost int64
	for _, w := range list {
		if w.Count <= 0 {
			continue
		}
		cost += ezfyWoundHealGoldPer(w.TroopId) * w.Count
	}
	if city.Gold < cost {
		return fmt.Sprintf("黄金不足: 全部恢复需要%d黄金, 当前只有%d", cost, city.Gold)
	}
	city.Gold -= cost
	h.saveCityRes(city)
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
	// ★ 科技全城公用 → 科研中心等级取玩家所有城的最高值
	if h.maxAcademyLevel(city.UserID) < academyNeed {
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
	// ★ 科技全城公用：读写都落到「科技城」（玩家主城）
	techCity := h.techCityId(city.ID)
	var researching int64
	h.DB.Model(&model.EzfyCityTech{}).Where("city_id = ? AND status = 1", techCity).Count(&researching)
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
	// ★ 科技写入「科技城」（全城公用）
	var t model.EzfyCityTech
	if err := h.DB.Where("city_id = ? AND tech_id = ?", techCity, techId).First(&t).Error; err != nil {
		t = model.EzfyCityTech{CityId: int64(techCity), TechId: techId, Level: 0, Status: 1, EndTime: now + h.techResearchMs(lv.ResearchTime)}
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
	if err := h.DB.Where("city_id = ? AND tech_id = ? AND status = 1", h.techCityId(city.ID), techId).First(&t).Error; err != nil {
		return "该科技没有在研究中"
	}
	if lv := ezfyCfg.techLevel(techId, t.Level+1); lv != nil {
		// 退还也不受仓储上限截断（与取消训练一致）
		h.giveResNoCap(city, lv.Food, lv.Steel, lv.Oil, lv.Rare, lv.Gold)
	}
	h.DB.Model(&model.EzfyCityTech{}).Where("id = ?", t.ID).
		Updates(map[string]interface{}{"status": 0, "end_time": 0})
	return ""
}

func (h *EzfyHandler) checkTechDone(city *model.EzfyCity) {
	now := time.Now().UnixMilli()
	// ★ 第九轮：科技所有城池公用 —— 研究状态统一落在「科技城」(主城)，
	//   所以在任意城进入游戏都能结算完成，不会再出现「切城后研究卡住/加成没生效」。
	tid := h.techCityId(city.ID)
	var list []model.EzfyCityTech
	h.DB.Where("city_id = ? AND status = 1", tid).Find(&list)
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
	// ★ 科技所有城池公用 → 研究中的记录在「科技城」(主城)
	if err := h.DB.Where("city_id = ? AND status = 1", h.techCityId(city.ID)).First(&t).Error; err != nil {
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
	h.consumeItemN(uid, cfgId, 1)
}

// consumeItemN 一次扣掉 n 个道具（n <= 0 时什么都不做）。
//
// ★ 批量道具（如经验书一次用几千本）必须走这个 —— 原来循环里逐本调 consumeItem，
// 一次请求就是几千条 SELECT + UPDATE。
func (h *EzfyHandler) consumeItemN(uid uint, cfgId, n int) {
	if n <= 0 {
		return
	}
	var it model.EzfyItem
	if err := h.DB.Where("user_id = ? AND cfg_id = ?", uid, cfgId).First(&it).Error; err != nil {
		return
	}
	if n >= it.Count {
		h.DB.Delete(&it)
		return
	}
	h.DB.Model(&model.EzfyItem{}).Where("id = ?", it.ID).Update("count", it.Count-n)
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
	// ★ 用户要求「军官经验道具最大只能用 100 不对，没有上限卡控」→ 经验书取消单次数量上限
	//   （真正的上限只剩「背包里有多少」，上面那条已经挡了）。
	//   ⚠️ 只对经验书放开：其余道具是「一本一次」的循环实现（每本都要读写库），
	//   放开会让一次请求打上万条 SQL，所以仍保留 99 的防呆上限。
	if count > 99 && cfg.ItemType != 10 {
		return "单次最多使用99个"
	}
	// 单次生效类道具不能批量
	if (cfg.ItemType == 6 || cfg.ItemType == 11 || cfg.ItemType == 12) && count > 1 {
		return cfg.Name + "每次只能使用1个"
	}
	// 需要指定军官的道具（★ 19 = 军官升星卡，也要选军官）
	needOfficer := cfg.ItemType == 10 || cfg.ItemType == 11 || cfg.ItemType == 12 || cfg.ItemType == 19
	if needOfficer && officerId <= 0 {
		return "请先选择要使用的军官"
	}
	if cfg.ItemType == 11 && skillId <= 0 {
		return "请选择要学习的技能"
	}
	// ★ 经验书(ItemType 10)：整批一次结算，并且**只扣真正用得上**的本数。
	//
	//	用户两条要求：
	//	  ① 「军官经验道具最大只能用 100 不对，没有上限卡控」→ 上面已取消 99 上限；
	//	  ② 「150 级超了，退回没用的经验书就行」→ 满级后再用会白扔（addOfficerExp 会把
	//	     满级后的经验直接清零），所以这里先算「升到满级还需要多少经验」，
	//	     只消耗够用的本数，其余原样留在背包里。
	if cfg.ItemType == 10 {
		o := h.officerOf(city.ID, officerId)
		if o == nil {
			return "军官不存在"
		}
		per := cfg.Param1
		if per <= 0 {
			return "道具配置有误(经验为0)"
		}
		// 升到满级还差多少经验（升级需要 等级×200，与 addOfficerExp 同一口径）
		var need int64
		for lv, exp := o.Level, o.Exp; lv < ezfyOfficerMaxLevel; lv++ {
			need += int64(lv)*200 - exp
			exp = 0
		}
		if need <= 0 {
			return fmt.Sprintf("%s 已达最高等级%d级, 经验书不消耗", o.Name, ezfyOfficerMaxLevel)
		}
		used := int64(count)
		if maxBooks := (need + per - 1) / per; maxBooks < used {
			used = maxBooks
		}
		h.addOfficerExp(city, o.ID, per*used)
		h.consumeItemN(uid, cfgId, int(used))
		msg := fmt.Sprintf("使用成功: %s 获得%d经验", o.Name, per*used)
		if used < int64(count) {
			msg += fmt.Sprintf("（已达%d级上限，本次只消耗%d本，其余%d本留在背包）",
				ezfyOfficerMaxLevel, used, int64(count)-used)
		}
		return msg
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
	case 12: // 重修书（洗点）
		// ★ 2026-09-22 用户要求：**属性退回「军官池里的原始属性」，已分配的点全部退回为可用属性点**，
		//   由玩家自己重新分配。
		//
		//   旧实现是「随机重新分配」，而且余数 `newLea = total - 军事 - 后勤` **全给学识**，
		//   玩家投诉「洗点后全加到学识上了」—— 那个实现已废弃。
		o := h.officerOf(city.ID, officerId)
		if o == nil {
			return "军官不存在"
		}
		if o.Status == 1 {
			return "军官出征中, 无法重修"
		}
		bm, bl, be := officerBaseAttr(o)
		refund := (o.Military - bm) + (o.Logistics - bl) + (o.Learning - be)
		if refund < 0 {
			refund = 0
		}
		free := o.FreePoints + refund
		h.DB.Model(&model.EzfyOfficer{}).Where("id = ?", o.ID).Updates(map[string]interface{}{
			"military": bm, "logistics": bl, "learning": be,
			"free_points": free, "skill": "", "update_time": time.Now(),
		})
		h.consumeItem(uid, cfgId)
		return fmt.Sprintf("使用成功: %s 重修完成\n军事 %d→%d  后勤 %d→%d  学识 %d→%d\n"+
			"退回可用属性点 +%d（当前 %d 点，去军官详情分配）\n（技能已清空，等级与经验保留）",
			o.Name, o.Military, bm, o.Logistics, bl, o.Learning, be, refund, free)
	case 19: // 星级徽章：按固定概率升 1 星，三维各 +N（概率/加多少/上限都走管理端配置）
		if !ezfyStarUpOn() {
			return "升星功能已关闭"
		}
		o := h.officerOf(city.ID, officerId)
		if o == nil {
			return "军官不存在"
		}
		// ★ 简化后：无论成功失败都消耗 1 枚徽章；已达上限则提前拦住，不白白扣卡
		if o.Star >= ezfyStarMax() {
			return fmt.Sprintf("星级已达上限(%d星)", ezfyStarMax())
		}
		msg, ok := h.officerStarUp(city, officerId)
		h.consumeItem(uid, cfgId)
		if !ok {
			return msg
		}
		return "使用成功: " + msg
	case 16, 17, 18:
		// ★ 三种迁城道具：**不能在背包里直接点「使用」**。
		//
		//   迁城必须知道「迁到哪座城 / 迁到哪个洲 / 迁到哪个坐标」，
		//   这些参数只有市政厅→城市迁移页（/games/ezfy/city/move）才有。
		//   所以这里只做「引导」，道具的消耗在 MoveCity 里完成
		//   （否则玩家在背包里误点一下就把道具用掉了、城却没动 —— 会被当成 bug 报上来）。
		return fmt.Sprintf("【%s】请在「市政厅 → 城市迁移」页面使用", cfg.Name)
	case 21: // 军官改名卡：在军官详情页使用（不在背包直接点）
		return fmt.Sprintf("【%s】请在「军官 → 军官详情」页面使用", cfg.Name)
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

func (h *EzfyHandler) resetPeriodTasks(uid uint) {
	now := time.Now()
	var mine []model.EzfyTask
	h.DB.Where("user_id = ?", uid).Find(&mine)
	for _, t := range mine {
		var c model.EzfyCfgTask
		if err := h.DB.First(&c, t.CfgId).Error; err != nil || c.TypeId <= 0 {
			continue
		}
		var tp model.EzfyCfgTaskType
		if err := h.DB.First(&tp, c.TypeId).Error; err != nil {
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
		}
		h.DB.Model(&model.EzfyTask{}).Where("id = ?", t.ID).Updates(updates)
	}
}

// ezfyPeriodKey 返回任务当前所属周期的标识（用于跨周期重置判断）：
//   - 每日(reset_type=1)：当天日期
//   - 每周(reset_type=2)：本周周一所在日期
//   - 一次性(reset_type=0)：返回空串（永不跨期重置）
func ezfyPeriodKey(resetType int, now time.Time) string {
	switch resetType {
	case 1:
		return now.Format("2006-01-02")
	case 2:
		wd := int(now.Weekday())
		if wd == 0 {
			wd = 7
		}
		return now.AddDate(0, 0, -(wd - 1)).Format("2006-01-02")
	default:
		return ""
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
	// ★ 任务奖励**不受仓储上限截断**（用户要求）。
	//   原来走 min64(cap, ...)，仓储满了领奖就等于白发；只有「城市自身产量」才该被上限卡住。
	h.giveResNoCap(&city, cfg.RewardFood, cfg.RewardSteel, cfg.RewardOil, cfg.RewardRare, cfg.RewardGold)
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

	// ★ 军官列表在本请求内只读一次，供懒结算扣工资 + 下面的展示复用。
	//   本接口是首页 30s 轮询的目标，重复查军官表曾是线上 IO 飙升的主因。
	officers := h.officerList(city.ID)
	h.refreshCityWithOfficers(uid, &city, officers)

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
	// ★ 2026-09-23：超过「伤兵存活天数」还没救治的伤兵直接消失（用户要求 5 天）
	wounded = h.filterExpiredWounded(wounded)

	queues := []gin.H{}
	var qs []model.EzfyTrainQueue
	h.DB.Where("city_id = ? AND status = 0", city.ID).Order("start_time ASC").Find(&qs)
	for _, q := range qs {
		queues = append(queues, gin.H{"id": q.ID, "troop_id": q.TroopId,
			"name": ezfyCfg.troopName(q.TroopId, camp), "count": q.Count, "end_time": q.EndTime})
	}

	// ★ 已研究的科技列表（供首页/统帅页展示）
	//   原实现 `for _, t := range h.techMap(city.ID)` 把 **value(等级)** 当成了 tech_id 去查配置，
	//   于是「炼钢5级」被显示成「军训艺术 0级」，且 map 遍历顺序随机 → 同一条重复出现。
	//   现在按 tech_id 升序遍历，等级取 map 的 value。
	techViews := []gin.H{}
	tmap := h.techMap(city.ID)
	techIds := make([]int, 0, len(tmap))
	for id := range tmap {
		techIds = append(techIds, id)
	}
	sort.Ints(techIds)
	for _, id := range techIds {
		if cfg := ezfyCfg.tech(id); cfg != nil {
			techViews = append(techViews, gin.H{"tech_id": id, "name": cfg.Name, "level": tmap[id]})
		}
	}

	var wildlands []model.EzfyWildland
	h.DB.Where("city_id = ?", city.ID).Find(&wildlands)
	// ★ 采集中状态按该野地上是否有「驻守采集」订单实时判定(常驻制, 不再依赖野地表的 status 字段)
	var gatherIds []int64
	h.DB.Model(&model.EzfyOrder{}).
		Where("user_id = ? AND status = 1 AND order_type = 7", uid).
		Pluck("target_id", &gatherIds)
	gathering := map[int64]bool{}
	for _, id := range gatherIds {
		gathering[id] = true
	}
	wildViews := []gin.H{}
	for _, w := range wildlands {
		sts := w.Status
		if gathering[int64(w.ID)] {
			sts = 1
		}
		wildViews = append(wildViews, gin.H{"id": w.ID, "x": w.X, "y": w.Y, "level": w.Level,
			"wild_type": w.WildType, "terrain": ezfyTerrainEx(w.X, w.Y), "terrain_name": ezfyTerrainNameEx(w.X, w.Y),
			"status": sts, "continent": ezfyRegionName(w.X, w.Y)})
	}

	var marching, occupying int64
	h.DB.Model(&model.EzfyOrder{}).Where("user_id = ? AND status = 0", uid).Count(&marching)
	h.DB.Model(&model.EzfyOrder{}).Where("user_id = ? AND status = 1", uid).Count(&occupying)
	var unreadReports int64
	h.DB.Model(&model.EzfyReport{}).Where("user_id = ? AND is_read = 0", uid).Count(&unreadReports)

	acct, ulv, uexp := h.ezfyUserBrief(uid)
	// ★ 军事区/资源区上限（各 33，管理端可维护）：随 /view 下发，前端不再硬编码
	lim := ezfyLimit()
	resp.OK(c, gin.H{
		"profile":    profile,
		"account":    acct,
		"user_level": ulv,
		"user_exp":   uexp,
		// ★ 在职军官数：直接用本请求已取到的 officers 统计，
		//   不要再调 h.officerCount（它内部又查一次 officerList）。
		"officer_count": officerCountOf(officers),
		"rank_name":     ezfyRankName(profile.Prestige),
		"rank_post":     ezfyRankPost(profile.Prestige),
		"cities":        h.cityViews(cities),
		"diamond":       profile.Diamond,
		"city":          city,
		"continent":     ezfyContinentName(city.X, city.Y),
		// 海城/陆地城市(海城可建航海协会、训练海军)
		"is_sea":       h.isSeaCity(&city),
		"city_kind":    h.cityKind(&city),
		"terrain":      ezfyTerrainEx(city.X, city.Y),
		"terrain_name": ezfyTerrainNameEx(city.X, city.Y),
		"is_coastal":   h.isCoastalCity(&city),
		// ★ 游戏ID（不随家园ID变化）与家园号码（转靓号后跟着变）
		"game_uid":        profile.GameUID,
		"home_num":        acct,
		"current_city_id": city.ID,
		"protected":       h.hasCityEffect(city.ID, 2),
		"boost":           h.hasCityEffect(city.ID, 1),
		"buildings":       buildingViews,
		"building_pool":   buildingPool,
		// ★ 军事区/资源区各自上限（分开下发）
		"military_cap": lim.MilitaryMax,
		"resource_cap": lim.ResourceMax,
		"troops":       troopViews,
		"wounded":      wounded,
		"queues":       queues,
		"techs":        techViews,
		"wildlands":    wildViews,
		"marching":     marching,
		"occupying":    occupying,
		// ★ 占用人口 = 建筑占用人口 + 训练中未出厂的新兵占用（部队不占人口位置）。
		//   否则没进过「军队」页时 troopsData 还是空的 → 空闲人口会显示成满人口（用户反馈的 bug）
		"pop_used":       h.cityPopUsed(city.ID),
		"unread_reports": unreadReports,
		// ★ 资源显示名（管理端可改名，前端一律读这里，不要再写死「粮食/钢铁/…」）
		"res_names": ezfyResCfgOf(h.DB),
		// ★ 集结令配置：跟着 /view 一起下发，前端一进页面就是准确值。
		//   原来只有 /order/preview 才返回 gather_max，前端在「还没点[计算]」时
		//   兜底写死 50 → 管理端配了 999 也只能填 50（用户反馈的 bug）。
		"gather_max":  ezfyGatherMax(),
		"gather_per":  ezfyGatherBonusPer(),
		"gather_have": h.itemCount(uid, ezfyGatherItemID),
		// ★ 军官工资（黄金/小时）：军官页直接展示，让玩家看得见钱花在哪
		//   用上面已取到的 officers 做纯内存计算（勿改回 officerSalaryPerHour）
		"officer_salary": officerSalaryOf(officers),
	})
}

// ezfyResCfgOf 读取资源显示名配置（管理端可在「资源管理 → 资源名称维护」改）
// 返回 {"gold":"黄金","food":"粮食",...,"_short":{"gold":"金",...}}
// 表为空时回落到内置默认值，保证前端永远拿得到名字。
func ezfyResCfgOf(db *gorm.DB) gin.H {
	def := []model.EzfyCfgResource{
		{Key: "gold", Name: "黄金", Short: "金"},
		{Key: "food", Name: "粮食", Short: "粮"},
		{Key: "steel", Name: "钢铁", Short: "钢"},
		{Key: "oil", Name: "石油", Short: "油"},
		{Key: "rare", Name: "稀矿", Short: "稀"},
	}
	var rows []model.EzfyCfgResource
	db.Order("sort, id").Find(&rows)
	if len(rows) == 0 {
		rows = def
	}
	out := gin.H{}
	short := gin.H{}
	for _, r := range rows {
		if r.Key == "" {
			continue
		}
		out[r.Key] = r.Name
		short[r.Key] = r.Short
	}
	out["_short"] = short
	return out
}

// buildingName 取建筑显示名（从配置表读，管理端改名后文案跟随）
func (h *EzfyHandler) buildingName(id int) string {
	h.cfgs()
	if b := ezfyCfg.building(id); b != nil && b.Name != "" {
		return b.Name
	}
	return "建筑#" + strconv.Itoa(id)
}

// ezfyBaseBuildingNames 新城自带的 3 个基础建筑名字（从配置表读，管理端改名后文案跟随）
func ezfyBaseBuildingNames(db *gorm.DB) string {
	names := []string{}
	for _, id := range []int{1, 2, 3} { // 市政厅 / 民居 / 农田
		var b model.EzfyCfgBuilding
		if err := db.First(&b, id).Error; err == nil && b.Name != "" {
			names = append(names, b.Name)
		}
	}
	if len(names) == 0 {
		return "市政厅/民居/农田"
	}
	return strings.Join(names, "/")
}

// ResCfg 游戏端读取资源显示名（管理端改名后前端可立即跟随，无需重进游戏）
func (h *EzfyHandler) ResCfg(c *gin.Context) {
	resp.OK(c, ezfyResCfgOf(h.DB))
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
	h.cfgs()
	var cities []model.EzfyCity
	h.DB.Where("user_id = ?", uid).Order("id ASC").Find(&cities)
	resp.OK(c, gin.H{"cities": h.cityViews(cities)})
}

// ezfyCityView 城市列表项：在 EzfyCity 全部字段之上补 is_sea / kind。
//
// ★ 第九轮：海城判据只在后端算（沿海平原 = ezfyTerrainEx == 9），
//
//	前端**不要**再自己用坐标哈希算 —— 旧版前端用「地形==8(海洋)」判海城，
//	导致真正的海城在列表里显示成「陆地城市」（用户反馈的 bug）。
type ezfyCityView struct {
	model.EzfyCity
	IsSea bool   `json:"is_sea"`
	Kind  string `json:"city_kind"` // ★ 与 /view 的 city_kind 保持同名，前端不要出现两套
	// ★ 所属大洲 / 大洋（世界地图改版后，城市要标注在哪个州）
	Continent string `json:"continent"`
}

func (h *EzfyHandler) cityViews(list []model.EzfyCity) []ezfyCityView {
	out := make([]ezfyCityView, 0, len(list))
	for i := range list {
		out = append(out, ezfyCityView{
			EzfyCity:  list[i],
			IsSea:     h.isSeaCity(&list[i]),
			Kind:      h.cityKind(&list[i]),
			Continent: ezfyRegionName(list[i].X, list[i].Y),
		})
	}
	return out
}

// SwitchCity 切换城市
// SwitchCity 切换当前操作的城市（★ 必须落库，否则下次请求又回落到主城 —— 这就是「分城切不过去」的根因）
func (h *EzfyHandler) SwitchCity(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		CityId int64 `json:"city_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	ct := h.cityOf(uid, req.CityId)
	if ct == nil {
		resp.ParamError(c, "城市不存在或已被占领")
		return
	}
	h.ensureProfile(uid)
	if err := h.DB.Model(&model.EzfyProfile{}).Where("user_id = ?", uid).
		Update("current_city_id", int64(ct.ID)).Error; err != nil {
		resp.ParamError(c, "切换失败："+err.Error())
		return
	}
	resp.OK(c, gin.H{"msg": "已切换到「" + ct.Name + "」", "city_id": ct.ID})
}

// CreateCity 新建分城（平原 → 陆地城市；沿海平原 → 海城）
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
	// ★ 用户规则：只能建在「平原」或「沿海平原」上；海城只能建在沿海平原
	terr := ezfyTerrainEx(req.X, req.Y)
	if terr != 1 && terr != ezfyTerrainCoastalPlain {
		resp.ParamError(c, "只能在平原或沿海平原上建造新城（海城需要沿海平原）")
		return
	}
	isSea := terr == ezfyTerrainCoastalPlain
	var n int64
	h.DB.Model(&model.EzfyCity{}).Where("x = ? AND y = ?", req.X, req.Y).Count(&n)
	if n > 0 {
		resp.ParamError(c, "该位置已有城市, 无法建造")
		return
	}
	// ★ 军衔限制分城数量（可建城数见 ezfy_cfg_rank.city_max）
	prof := h.ensureProfile(uid)
	var owned int64
	h.DB.Model(&model.EzfyCity{}).Where("user_id = ?", uid).Count(&owned)
	maxCity := ezfyRankCityMax(prof.Prestige)
	if int(owned) >= maxCity {
		resp.ParamError(c, fmt.Sprintf("当前军衔「%s」最多只能拥有 %d 座城市（已有 %d 座），提升声望可解锁更多",
			ezfyRankName(prof.Prestige), maxCity, owned))
		return
	}

	// 扣费走主城（不受当前切换影响）
	main := h.mainCity(uid)
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
		kind = "沿海平原"
		extra = "\n该城为【沿海城市】: 可建造航海协会并训练海军。"
	}
	h.addReport(uid, 5, "新城建成",
		fmt.Sprintf("花费%d黄金在%s(%d,%d)建造了新城[%s]\n新城自带基础建筑: %s(1级), 可到[城市列表]切换操作。%s",
			ezfyNewCityGoldCost, kind, req.X, req.Y, city.Name, ezfyBaseBuildingNames(h.DB), extra))
	resp.OK(c, gin.H{"msg": "新城建成", "city": city})
}

// DestroyCity 摧毁自己的城市（★ 仅限「非当前所在」的城市）
//
// 用户规则：摧毁后该坐标变回普通平原（不再属于任何玩家）。
func (h *EzfyHandler) DestroyCity(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		CityId int64 `json:"city_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.CityId <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	ct := h.cityOf(uid, req.CityId)
	if ct == nil {
		resp.ParamError(c, "城市不存在或已被占领")
		return
	}
	// 至少保留一座城
	var owned int64
	h.DB.Model(&model.EzfyCity{}).Where("user_id = ?", uid).Count(&owned)
	if owned <= 1 {
		resp.ParamError(c, "至少要保留一座城市")
		return
	}
	// ★ 仅能摧毁**非当前所在**的城市（既有规则，勿改）：
	//   否则玩家会把自己正站着的城拆掉，操作无法撤销。
	cur := h.currentCity(uid)
	if cur.ID == ct.ID {
		resp.ParamError(c, "不能摧毁当前所在的城市，请先切换到别的城市")
		return
	}
	if msg := h.ezfyDestroyCity(uid, ct); msg != "" {
		resp.ParamError(c, msg)
		return
	}
	resp.OK(c, gin.H{"msg": "城市「" + ct.Name + "」已摧毁，该坐标恢复为普通平原"})
}

// ezfyDestroyCity 真正拆除一座城：清掉它的建筑/部队/科技/军官/野地/队列等，
// 并抹掉该坐标的「玩家城」地图区域记录（于是变回普通平原，不属于任何玩家）。
func (h *EzfyHandler) ezfyDestroyCity(uid uint, ct *model.EzfyCity) string {
	cid := ct.ID
	// ★ 科技是全城公用的：如果拆的正好是「科技城」(主城)，先把科技搬到剩下的第一座城，
	//   否则科技会跟着一起消失。
	if h.techCityId(cid) == cid {
		var next model.EzfyCity
		if err := h.DB.Where("user_id = ? AND id <> ?", uid, cid).Order("id ASC").First(&next).Error; err == nil {
			h.ezfyMoveTechTo(cid, next.ID)
		}
	}
	// 还在外面的部队/采集队：一并撤掉（否则会留下指向已删城市的孤儿订单）
	h.DB.Where("city_id = ?", cid).Delete(&model.EzfyOrder{})
	h.DB.Where("city_id = ?", cid).Delete(&model.EzfyCityBuilding{})
	h.DB.Where("city_id = ?", cid).Delete(&model.EzfyCityTroop{})
	h.DB.Where("city_id = ?", cid).Delete(&model.EzfyCityTech{})
	h.DB.Where("city_id = ?", cid).Delete(&model.EzfyTrainQueue{})
	h.DB.Where("city_id = ?", cid).Delete(&model.EzfyWildland{})
	h.DB.Where("city_id = ?", cid).Delete(&model.EzfyWounded{})
	h.DB.Where("city_id = ?", cid).Delete(&model.EzfyCityEffect{})
	h.DB.Where("city_id = ?", cid).Delete(&model.EzfyCityTarget{})
	h.DB.Where("city_id = ?", cid).Delete(&model.EzfyOfficer{})
	h.DB.Where("city_id = ?", cid).Delete(&model.EzfyEquipment{})
	// 被这座城占领的玩家城/野地记录也要释放
	h.DB.Where("city_id = ? AND status = 1", cid).Delete(&model.EzfyOccupy{})
	// ★ 地图区域：删掉「玩家城」记录 → 该格回到普通地形
	h.DB.Where("x = ? AND y = ?", ct.X, ct.Y).Delete(&model.EzfyMapArea{})
	if err := h.DB.Delete(&model.EzfyCity{}, cid).Error; err != nil {
		return "摧毁失败：" + err.Error()
	}
	h.addReport(uid, 5, "城市已摧毁",
		fmt.Sprintf("城市「%s」(%d,%d) 已被摧毁，该坐标恢复为普通平原。", ct.Name, ct.X, ct.Y), "", 0)
	return ""
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

// Convene 召集人口（粮食召集不受民居上限限制）
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
	if city.Food < ezfyConveneFoodCost {
		resp.ParamError(c, fmt.Sprintf("粮食不足, 召集10万人口需要%d粮食", ezfyConveneFoodCost))
		return
	}
	city.Food -= ezfyConveneFoodCost
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
	// ★ 用户反馈修复：原来直接删野地记录，**驻守/采集中的部队会凭空消失**。
	//   现在先把这个野地上的采集(4)/驻守(7)命令改成返航，兵力与已采资源随部队回城。
	now := time.Now().UnixMilli()
	var orders []model.EzfyOrder
	h.DB.Where("user_id = ? AND target_id = ? AND order_type IN (4,7) AND status IN (0,1)", uid, w.ID).Find(&orders)
	for i := range orders {
		o := &orders[i]
		travel := ezfyOneWayTravel(o)
		o.Status = 2
		o.Result = o.Troops
		o.ReturnTime = now + travel
		h.DB.Model(&model.EzfyOrder{}).Where("id = ?", o.ID).
			Updates(map[string]interface{}{"status": 2, "result": o.Result,
				"return_time": o.ReturnTime, "carry": o.Carry})
	}
	h.DB.Delete(&w)
	msg := "已放弃该野地"
	if len(orders) > 0 {
		msg += fmt.Sprintf("，%d 支采集/驻守部队已返航（到达后兵力与资源回城）", len(orders))
	}
	resp.OK(c, gin.H{"msg": msg})
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
