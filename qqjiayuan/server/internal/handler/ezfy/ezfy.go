package ezfy

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 二战风云 核心玩法：进入游戏/城池/建筑/资源懒结算/造兵/伤兵/科技

const (
	ezfyFactoryBuildingID = 14 // 军工厂（★ 不限数量，只受军事区建筑上限约束）
	// ★ 建筑数量上限（军事区/资源区各 36、民居 33）已迁到 ezfy_cfg_limit 表，
	//   管理端「二战风云 → 建筑上限配置」可维护，见 ezfyLimit()。
	// ★ 2026-09-26 「花费 10万粮食 召集 10万人口也要能配置」：
	//   召集消耗粮食 / 获得人口已迁到 ezfy_cfg_limit（convene_food_cost / convene_pop_gain），
	//   管理端「二战系统配置 → 玩法开关」可维护，见 ezfyConveneFoodCostCfg / ezfyConvenePopGainCfg。
	// ★ 2026-10-09 用户要求「起新城资源各 10 万」（前端文案原来还写着「消耗10万黄金」）→ 5 万改回 10 万。
	ezfyNewCityResCost = 100000 // 起新城消耗: 粮食/钢铁/石油/稀矿/黄金 各 10 万（★ 2026-09-28 曾为 10 万黄金；2026-10-09 由各 5 万改回各 10 万）
	ezfyOilDivGrid     = 300   // 出征耗油: 每格耗油 = 总兵力/300
	// ★ 2026-09-24 「采集 12 小时才有宝物 → 4 小时且可配置」：
	//   采集结算周期不再写死，读取管理端配置 ezfy_cfg_limit.dispatch_period_h（小时，默认 4），
	//   见 ezfyDispatchPeriod()。
	ezfyTreasureExtraPct  = 20 // 每期在保底 1 件宝物的基础上, 额外 1 件概率%
	ezfyCommandCarryPct   = 10 // 指挥艺术: 出征携带上限+%/级
	ezfyMaxUpgradeSeconds = 10 // 一键满级: 每级升级时间(秒)
	ezfyDeserterRate      = 30 // 守军战败溃逃比例%
	ezfyWarDelayHours     = 24 // 宣战生效延迟(小时)
	ezfyWarDurationHours  = 48 // 宣战有效期(小时)
	// ★ 第九轮：取消训练手续费（%），按常见游戏取 10%
	ezfyCancelTrainFeePct = 10
	// ★ 第九轮：军官忠诚 —— 派遣不再扣，只有打败仗才扣（见 ezfy_battle.go）
	ezfyLoyaltyOnDefeat = 3 // 败仗基础扣忠心
	// ★ 建筑图纸道具 cfg_id（ezfy_cfg_item 表 ID=10，ItemType=6），升级到 10 级及以上必需
	ezfyBlueprintItemID = 10
	// ★ 司令部兵种战斗配置「防守」第三种状态：不参与防御（被攻击时防御战斗兵种不含它）
	ezfyDefMoveNone = -1
	// ★ 2026-09-28 科技 13「装载技术」：部队负重 +2%/级。
	//   原来这条科技只在 ezfy_cfg_tech.effect 里写着，**代码里从没读过** →
	//   玩家研到满级负重也不涨，就是反馈的「科技没实际作用」。见 ezfyCarryCapOf。
	ezfyLoadTechID  = 13
	ezfyLoadTechPct = 2
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
	// ★ 2026-10-03 双机共享一个 RDS：周期刷新让两台进程内配置缓存收敛（30s）。
	ezfyStartConfigReloader.Do(func() { go ezfyPeriodicReload(h.DB) })
	ezfyCfg.load(h.DB)
	// ★ 2026-10-05 永久停用「开机自动搬城」（用户明确要求）。
	//
	//   原来这里会调 ezfyMigrateSeaCities，在每次启动时把「地形判定=海洋」的城搬到最近的
	//   沿海平原。问题：它依赖 ezfyTerrainEx 的输入（地图格子覆盖 ezfy_map_tile）。一旦
	//   启动瞬间瓦片读取失败/抖动，地形整体翻转，就会把一批本不该动的城判成「海城」并整体
	//   位移 —— 玩家看到的就是「重新部署后城市坐标又变了」。这是同一 bug 反复复发的根源。
	//
	//   现在：城市坐标**只由**玩家操作（迁城）或**显式运维命令**改动，启动流程绝不自动搬城。
	//   需要一次性修数据时，跑运维命令：
	//     ./ezfymigrate --coastal --apply --yes   （把建有航海协会的城迁回沿海平原）
	// → 该自动迁移函数已整体删除（见 ezfy_geo.go 头部说明），不再随进程启动执行。
	// 一次性迁移：科技从「按城各存」合并为「所有城池公用」（幂等，见 ezfy_tech_shared.go）
	ezfySharedTechOnce.Do(func() { ezfyMigrateSharedTech(h.DB) })
	// 一次性迁移：存量战报「野地N级」→ 具体地形名（幂等，见 ezfy_migrate_report.go）
	ezfyReportMigrateOnce.Do(func() { ezfyMigrateReportTitles(h.DB) })
	// 一次性迁移：城防兵超城墙容量 → 按比例缩回（幂等，见 ezfy_migrate_troop_cap.go）
	ezfyTroopCapOnce.Do(func() { ezfyMigrateTroopCap(h.DB) })
	// 一次性迁移：老玩家已晋升军衔落位（不用补宝物）+ 军衔宝物需求落表（幂等，见 ezfy_rank_treasure.go）
	ezfyRankInitOnce.Do(func() { ezfyMigrateRankInit(h.DB) })
	// 一次性迁移：上等兵/下士 晋升无需珠宝（把 2/3 级宝物需求清成 []，幂等）
	ezfyRankNoJewelOnce.Do(func() { ezfyMigrateRankNoJewel(h.DB) })
	// 一次性迁移：历史「宝物签到」误发到道具表的宝物 → 装备表（幂等，见 ezfy_rank_treasure.go）
	ezfyTreasureBagOnce.Do(func() { ezfyMigrateTreasureBag(h.DB) })
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
	// ★ 2026-09-28 新玩家军衔从 1 级（列兵）起步，晋升需声望+宝物（见 ezfy_rank_treasure.go）
	p = model.EzfyProfile{UserID: uid, GameUID: int64(uid), Nickname: nickname, Prestige: 0, Camp: 1, Rank: 1}
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

// anyCity 取玩家已有的任意一座城（id 最小 = 主城），不会建城。
func (h *EzfyHandler) anyCity(uid uint) (model.EzfyCity, bool) {
	var city model.EzfyCity
	if err := h.DB.Where("user_id = ?", uid).Order("id ASC").First(&city).Error; err != nil {
		return city, false
	}
	return city, true
}

// mainCity 主城（id 最小，建城扣费/城市列表基准用，不受切换影响）
func (h *EzfyHandler) mainCity(uid uint) model.EzfyCity {
	if city, ok := h.anyCity(uid); ok {
		return city
	}
	return h.createMainCity(uid)
}

// ezfyCityInitLocks 懒建城串行锁（按 uid 分片成固定 64 把，不用 map 以免无限增长）。
//
// ★ 用户反馈「玩家一进游戏就有两座城」的根因：
//
//	前端 mounted 会**并发**打好几个都要走 getOrCreateCity 的接口
//	（/view、/troops、/techs、/res-cfg…），每个请求都是「查不到城 → 建一座」，
//	于是同一毫秒各建一座。线上实测两座城的 created_at 只差 4~8ms，
//	且 3 个「声望 0（列兵，可建城数=1）」的玩家都有 2 座城 ——
//	说明不是军衔/升衔带来的，就是并发重复建城。
//
// 修法：建城前先抢「本玩家的锁」，进锁后二次确认，真的没有城才建；
// 抢到锁时发现别人已建好，直接复用它，绝不多建。
var ezfyCityInitLocks [64]sync.Mutex

func ezfyCityInitLock(uid uint) *sync.Mutex { return &ezfyCityInitLocks[uid%64] }

// ezfyPrestigeLocks 军衔晋升播报的并发锁（见 addPrestige 里的说明）
var ezfyPrestigeLocks [64]sync.Mutex

// ezfyOccupyLocks 征服「最终一城」判定的并发锁：按**守方** uid 分片，
// 串行化同一守方的多路并发征服。征服结算既能由攻方轮询(processOrders)触发、
// 也能由守方轮询(processIncoming)触发，多个进攻方可能同时在 processArrive 里
// 对同一守方做「统计现城数→建占领记录」的读-改-写；不加锁会集体判定通过、把守方打到 0 城。
// 锁内先按「未被占领的自由城」计数，只允许守方在占掉这座后仍至少剩 1 城时才建占领记录。
var ezfyOccupyLocks [64]sync.Mutex

func ezfyOccupyLock(uid uint) *sync.Mutex { return &ezfyOccupyLocks[uid%64] }

// ezfyTrainLocks 训练/建造的并发锁：按**城市** id 分片。
// ★ 2026-09-28 修复「城防数量超过城防空间」并发漏洞：
//   原来 trainTroop 里「读已占用量(defenceSpaceUsed) → 校验空间 → 建队列」不是原子的，
//   两个并发请求同时读到同一个已占用量、双双通过校验，城防总量最终超过围墙容量。
var ezfyTrainLocks [64]sync.Mutex

func ezfyTrainLock(cityID uint) *sync.Mutex { return &ezfyTrainLocks[cityID%64] }

// ezfySpeedTrainLocks 训练一键加速的并发锁（按 **玩家** id 分片）。
// ★ 2026-10-03 修复「频繁点击零消耗黄金」：SpeedTrainAll 里
//   「读队列剩余秒数 → 算钱 → 校验黄金 → 扣黄金 → 清空队列剩余时间」不是原子的，
//   连点会并发进入同一段结算：第二路看到已被清零的剩余时间（算成 0 秒），
//   于是算出 0 黄金并返回「消耗0黄金」。串行化后一次只能有一个请求在结算扣费。
var ezfySpeedTrainLocks [64]sync.Mutex

func ezfySpeedTrainLock(uid uint) *sync.Mutex { return &ezfySpeedTrainLocks[uid%64] }

func (h *EzfyHandler) addPrestige(uid uint, amount int) {
	if amount <= 0 {
		return
	}
	// 节日活动·声望加成(福利.txt #11)
	if pct := h.actPct(ezfyActPrestige); pct > 0 {
		amount = amount * (100 + pct) / 100
	}
	// ★★ 2026-09-26 修复「军衔晋升播报重复 N 次」：
	//
	//	前端 mounted 会**并发**打好几个接口（/view、/troops、/techs…），每个请求都会触发
	//	懒结算 → addPrestige 并发进入，各自读到**同一个旧 prestige**，于是 `after != before`
	//	同时成立、同一个军衔被播报多次。
	//	线上实测：id 104/105/106 三条一模一样的「恭喜玩家 小哥哥 军衔晋升至 军士长！」，
	//	时间戳 14:58:25.508 / .544 / .552 —— 只差 8~36ms，典型并发重复写入。
	//	顺带：`p.Prestige + amount` 是读-改-写，并发时还会**丢更新**（几次只加到 1 次）。
	//
	//	修法：按玩家加锁把同一玩家的 addPrestige 串行化，并在锁内重新读库拿最新值 ——
	//	第 2、3 个请求读到的已是更新后的 prestige，`after == before`，自然不再播报。
	mu := &ezfyPrestigeLocks[uid%64]
	mu.Lock()
	defer mu.Unlock()

	p := h.ensureProfile(uid)
	// ★ 2026-09-28 军衔不再自动跟随声望：声望只涨数值，不直接晋升。
	//   新玩家（Rank>0）的军衔固定，这里不会触发播报；晋升播报在 /promote 提交宝物时发。
	//   老玩家（Rank=0）仍回落声望推导，跨过门槛继续按旧规则播报。
	beforeLv := ezfyProfileRank(&p)
	afterLv := beforeLv
	if p.Rank <= 0 {
		afterLv = ezfyRankIndex(p.Prestige+amount) + 1
	}
	h.DB.Model(&model.EzfyProfile{}).Where("id = ?", p.ID).
		Update("prestige", p.Prestige+amount)
	// ★ 军衔晋升写一条系统消息（首页世界聊天要能看到「恭喜玩家晋升XX」）
	if afterLv != beforeLv {
		h.ezfySysChat("恭喜玩家 %s 军衔晋升至 %s！", h.ezfyProfileName(uid), ezfyRankNameAt(afterLv))
	}
}

// createMainCity 建主城（首次进游戏，随机平原空位，初始建筑 市政厅/民居/农田 各1级）
//
// ★ 并发调用下只会建出一座：抢到本玩家的建城锁后二次确认，
// 别人已经建好了就直接复用（首个请求建城，其余请求拿到同一座城）。
func (h *EzfyHandler) createMainCity(uid uint) model.EzfyCity {
	mu := ezfyCityInitLock(uid)
	mu.Lock()
	defer mu.Unlock()
	if exist, ok := h.anyCity(uid); ok {
		return exist
	}
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

// replenishCity 守方被征服占光后，系统补给一座随机新城市（保底，保证玩家永远有城）。
//
// ★ 为什么不能复用 createMainCity：createMainCity 开头有 anyCity 复用检查——
// 被占的 EzfyCity 行并没有删除/过户（只建了占领记录），anyCity 会误判「还有城」而拒绝补新城。
// 补给场景由征服结算判定「自由城已清零」后才调用，直接建一座新城即可。
// 落点 / 初始资源 / 基础建筑 / 当前城市指向 均与 createMainCity 一致。
func (h *EzfyHandler) replenishCity(uid uint) model.EzfyCity {
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
	// 当前城市指向新城（守方原当前城已被占，切到新城避免操作报错）
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
	dc := ezfyDefaultContinent()
	order := []int{dc}
	for _, a := range ezfyMoveAreas {
		if a.ID != dc {
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
	// ★ 2026-10-05 性能：两条查询互不依赖（被占判定用的就是入参 cityId），改**并行** ——
	//   原来串行 2 个跨 WAN 往返（线上 ~220ms），而 bodyCity 是所有操作接口的第一步。
	var city model.EzfyCity
	var oc int64
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); h.DB.Where("user_id = ? AND id = ?", uid, cityId).First(&city) }()
	go func() {
		defer wg.Done()
		h.DB.Model(&model.EzfyOccupy{}).Where("city_id = ? AND status = 1", cityId).Count(&oc)
	}()
	wg.Wait()
	if city.ID == 0 || oc > 0 {
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

// buildingLevelsOf 一次性建筑列表 → 「building_id → 最高等级」内存 map。
// ★ 2026-10-04 性能：展示接口多次调 buildingLevel（每次 = 一次完整 buildingList 查询）
//   时改为一次查全、纯内存取值，省掉重复 RDS 往返。
func buildingLevelsOf(list []model.EzfyCityBuilding) map[int]int {
	m := map[int]int{}
	for _, b := range list {
		if b.Level > m[b.BuildingId] {
			m[b.BuildingId] = b.Level
		}
	}
	return m
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
// ★ 第九轮用户规则：军事区与资源区数量上限**分开**，各 36（线上现值，管理端可维护，见 ezfy_cfg_limit）。
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

// areaCountsOf 按已取到的建筑列表纯内存统计军事/资源区数量（免重复全表查询）。
// ★ 2026-10-04 性能：/buildings 原来 areaCounts ×2 + 每栋建筑 buildingMaxLevel ×1
//   都是「再查一次完整建筑列表」，几十栋楼 = 几十条 RDS 往返 → 4s。
func areaCountsOf(list []model.EzfyCityBuilding) (military, resource int) {
	for _, b := range list {
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

// techMap 玩家科技等级表（用户级）。
//
// ★ 2026-09-28 「没有主城概念，所有城市都是一样的」：等级存 ezfy_user_tech
// （按 user_id），所有调用点（结算/展示/加成）自动变成全账号共用，不用逐个改。
func (h *EzfyHandler) techMap(cityId uint) map[int]int {
	var uid uint
	h.DB.Model(&model.EzfyCity{}).Where("id = ?", cityId).Select("user_id").Scan(&uid)
	return h.techMapOf(uid)
}

// techMapOf 按 user_id 取科技等级表。
//
// ★ 2026-10-05 性能：调用方**已经知道 uid** 时（/view、/resources、/buildings…）直接用它，
// 省掉 techMap 里那条 `SELECT user_id FROM ezfy_city WHERE id = ?`（跨 WAN 就是一次白跑往返）。
func (h *EzfyHandler) techMapOf(uid uint) map[int]int {
	m := map[int]int{}
	if uid == 0 {
		return m
	}
	var list []model.EzfyUserTech
	h.DB.Where("user_id = ?", uid).Find(&list)
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
// ★ 2026-09-23 「超过限制不能训练，提示超过限额」。
//
//	口径：当前兵力(城内 + 训练队列) + 本次要加的量 > troop_max → 拒绝。
//	返回空串表示通过，否则返回可直接展示给玩家的提示文案。
// cityTroopTotalD 同 cityTroopTotal，但复用已查好的部队表 + 训练队列（两者都非 nil 时零 SQL）。
func (h *EzfyHandler) cityTroopTotalD(troops map[int]int64, trainQ []model.EzfyTrainQueue) int64 {
	max := ezfyTroopMaxCfg()
	var total int64
	for _, cnt := range troops {
		if cnt > 0 {
			total = ezfySafeAdd(total, cnt, max)
		}
	}
	for _, q := range trainQ {
		if q.Count > 0 {
			total = ezfySafeAdd(total, q.Count, max)
		}
	}
	return total
}

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

// checkTroopCapD 同 checkTroopCap，但用快照里的部队表/训练队列纯内存算（省 2 条跨 WAN 往返）。
func (h *EzfyHandler) checkTroopCapD(add int64, troops map[int]int64, trainQ []model.EzfyTrainQueue) string {
	if add <= 0 {
		return "数量错误"
	}
	max := ezfyTroopMaxCfg()
	cur := h.cityTroopTotalD(troops, trainQ)
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
// ★ 2026-09-23 「伤兵 5 天不救治直接消失」。
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
// ★ 统一口径：建在沿海平原上的叫「沿海城市」，其余叫「内陆城市」。
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
	h.refreshCityD(uid, city)
}

// refreshCityD 同 refreshCity，但**把快照返回给调用方**。
//
// ★★ 2026-10-05 性能：需要「结算完还要继续用建筑等级/科技/部队」的接口（如出征下单）
// 用它，可以省掉后面 N 次重复查询（buildingLevel/techMap/troopMap 每次都是一整条跨 WAN 往返）。
func (h *EzfyHandler) refreshCityD(uid uint, city *model.EzfyCity) *resCalcData {
	return h.refreshCityDWithCities(uid, city, nil)
}

// refreshCityDWithCities 同 refreshCityD，但调用方已经查好「玩家城市列表」时传入，
// 省掉懒结算里那次串行的城市列表查询（跨 WAN ~120ms）。cities 为 nil 时自己查。
func (h *EzfyHandler) refreshCityDWithCities(uid uint, city *model.EzfyCity, cities []model.EzfyCity) *resCalcData {
	// 原来是 5 步各自查库（~15 条串行跨 WAN = 1s+）。
	// 现在先一次并行取齐（1 个 RTT），再全部走快照结算 —— 读 0 条，只留资源落库 1 条写。
	d := h.ezfyLoadCityData(uid, city, cities)
	h.ezfySettleCity(uid, city, d, true)
	return d
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
	// ★ 2026-10-05 性能：同 refreshCity，一次并行取数 + 快照结算（军官列表由调用方传入复用）。
	d := h.ezfyLoadCityData(uid, city, nil)
	h.checkBuildingDone(city, d.buildingsOf(h, city.ID))
	h.checkTechDoneRows(city, d.techRows)
	h.collectTrainQueue(city, d.trainQOf(h, city.ID))
	h.calcResourceD(city, d, officers)
	h.processOrders(uid, d.cityIDIntsOf(h, uid))
}

// refreshCityRead 只读展示页的轻量懒结算：建筑/科技/训练队列/资源照跑，
// 但**跳过订单结算**（processOrders 每次至少 3 条 RDS 往返，且串行在结算链尾）。
//
// 纯展示接口（军官/军校/技能/装备/任务）调用它，配合 3s 玩家级缓存把单次响应压到 <1s；
// 订单事件（返航/战斗/敌军到达）仍由 /view 轮询与操作接口的完整 refreshCity 照常推进，
// 展示页滞后最长 3 秒（缓存 TTL），可接受。
func (h *EzfyHandler) refreshCityRead(uid uint, city *model.EzfyCity) {
	// ★★ 2026-10-05 性能（用户反馈「军校招募/军官技能 2~3s」）：
	//   原来 4 步各自查库（建筑/科技/队列/资源共 ~15 条**串行**跨 WAN 往返）→ 2~3s。
	//   现在一次并行取齐（1 个 RTT）+ 快照结算（读 0 条）。展示页的 5 个调用点
	//   （军官列表 / 军校招募 / 装备图鉴 / 军官技能 / 招募详情）全部受益。
	//   ⚠️ 需要**复用这份快照**的接口请改用 `ezfyPageSettle`（它把快照返回给调用方），
	//      否则调用方还会再查一次建筑列表（那又是一条跨 WAN 往返）。
	h.ezfySettleCity(uid, city, h.ezfyLoadCityData(uid, city, nil), false)
}

// checkBuildingDone 建筑完成懒结算。reuse 传本请求已查好的建筑列表时可省一次 buildingList 查询
// （/view 30s 轮询已把 buildings 并入并行块，这里零重复 SQL）。
func (h *EzfyHandler) checkBuildingDone(city *model.EzfyCity, reuse ...[]model.EzfyCityBuilding) {
	now := time.Now().UnixMilli()
	var buildings []model.EzfyCityBuilding
	if len(reuse) > 0 && reuse[0] != nil {
		buildings = reuse[0]
	} else {
		buildings = h.buildingList(city.ID)
	}
	for i := range buildings {
		b := &buildings[i]
		if b.Status != 0 && now >= b.EndTime {
			// ★★ 2026-09-26 修复「建筑完成被并发重复结算」：
			//
			//	前端 mounted 会**并发**打好几个接口（/view、/troops、/techs…），每个请求都跑
			//	懒结算 → 同一栋楼被多个请求同时判定为「已完成」，于是
			//	  · `addPrestige` 被重复调用（实测 5 个并发 = 声望 +300 而不是 +60）
			//	  · `taskProgress("build_upgrade")` 被重复 +1
			//	  · 军衔播报也跟着重复（用户报的「播报 3 次」就是这个的冰山一角）
			//	（level 本身是幂等的 —— 各请求都写同一个 `b.Level+1`，所以看不出来。）
			//
			//	修法：**先算好完成后的状态，再用「status <> 0」做条件更新抢占**，
			//	只有 RowsAffected=1 的那个请求才算真正完成结算，其余直接跳过。
			newLevel := b.Level + 1
			cfg := ezfyCfg.building(b.BuildingId)
			// ★ 2026-09-25 用户纠正「一键9级 = 一键升级到 9 级，而不是升级满」：
			//   连锁模式(StartTime=0) 每级自动接续，但**升到目标等级就停** ——
			//   TargetLevel=0 的老数据仍按「升到该建筑上限」处理（旧行为）。
			chainTarget := b.TargetLevel
			if chainTarget <= 0 {
				chainTarget = h.buildingMaxLevel(city.ID, b.BuildingId)
			}
			newStatus, newEnd, newTarget := 0, b.EndTime, 0
			if b.StartTime == 0 && cfg != nil && newLevel < chainTarget {
				// 连锁：自动接下一级（target_level 保持，别清）
				newStatus = 2
				newEnd = now + ezfyMaxUpgradeSeconds*1000
				newTarget = b.TargetLevel
			}
			res := h.DB.Model(&model.EzfyCityBuilding{}).
				Where("id = ? AND status <> 0", b.ID).
				Updates(map[string]interface{}{"level": newLevel, "status": newStatus,
					"end_time": newEnd, "target_level": newTarget})
			if res.Error != nil || res.RowsAffected == 0 {
				continue // 已被其他并发请求结算过，别再重复加声望/任务进度
			}
			// ★ 2026-10-05 结算成功 → 同步内存里的行（建筑页/首页复用这份已取数据时
			//   立刻拿到新等级，不再显示旧等级 —— 修「建筑升级等级不动」）
			b.Level = newLevel
			b.Status = newStatus
			b.EndTime = newEnd
			b.TargetLevel = newTarget
			// ↓ 以下副作用只在「真正抢到结算权」时执行
			h.addPrestige(city.UserID, newLevel*10)
			if newStatus == 0 {
				if b.BuildingId == 1 {
					city.CityLevel = newLevel
				}
				h.taskProgress(city.UserID, "build_upgrade", 1)
			}
		}
	}
	var hall model.EzfyCityBuilding
	found := false
	if len(reuse) > 0 && reuse[0] != nil { // 复用并行块已查好的建筑，找市政厅（零 SQL）
		for _, b := range reuse[0] {
			if b.BuildingId == 1 {
				hall, found = b, true
				break
			}
		}
	}
	if !found {
		if err := h.DB.Where("city_id = ? AND building_id = 1", city.ID).First(&hall).Error; err != nil {
			return
		}
	}
	if hall.Level != city.CityLevel {
		city.CityLevel = hall.Level
		h.DB.Model(&model.EzfyCity{}).Where("id = ?", city.ID).Update("city_level", hall.Level)
	}
}

// calcResource 资源按小时懒结算（科技/开工率/市长后勤加成/道具增产/野地产出/军队耗粮/军官工资）
//
// ★ 2026-09-26 **民心/民怨不再影响产量**（原来 5 种产量都 × 民心系数，
// 民心不满时全体减产，还会把资源详情页的「加成产量」算成负数）。
// 民心/民怨本身仍然有效：决定能否被征服、被掠夺时扣减、安抚花费。
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
	h.calcResourceD(city, nil, officers...)
}

// calcResourceD 与 calcResource 完全同口径，但可传入**本请求已查好的城市只读数据快照** d：
// 科技/建筑/部队/野地/增产令/活动/市长全部走内存，函数内只剩最后一次资源写回（1 条 SQL）。
//
// ★ 2026-10-05 性能（用户反馈「/view /resources /troops /buildings 还是 2s+」）：
//
//	改造前 calcResource 自己串行查 8~9 条（techMap 还额外多一条 SELECT user_id），
//	而调用方（/view 等）的并行块里**刚刚查过同一批数据** —— 等于每个请求白打 8~9 个跨 WAN 往返。
//	d 为 nil 时行为与改造前逐字一致（各自自查），所以只有显式传快照的调用点才变快。
func (h *EzfyHandler) calcResourceD(city *model.EzfyCity, d *resCalcData, officers ...[]model.EzfyOfficer) {
	now := time.Now().UnixMilli()
	last := city.LastTime
	if last <= 0 {
		last = now
	}
	if now <= last {
		return
	}
	hours := float64(now-last) / 3600000.0

	tech := d.techsOf(h, city.ID)
	techFood := tech[1]
	techSteel := tech[2]
	techOil := tech[3]
	techRare := tech[4]
	techStore := tech[14]
	techSupply := tech[18]

	// ★★ 2026-09-28 用户规则大改：**民心与税率联动，「民心 + 税率 = 100」**。
	//
	//	- 基准民心 = 100 − 税率（税率 20% → 民心 80）。设税率时民心**立即**设为该值（见 SetTax）。
	//	- 被征服/掠夺导致民心下降、民怨升高，**但税率不动**；民心之后**自动回归**，
	//	  每小时朝 (100 − 税率) 靠拢，直到重新满足 民心 + 税率 = 100。
	//	- 回归速度：民心低于基准时每分钟 +1（即每小时 +60 上限），温和且能在几小时内恢复。
	//	- 民怨是**独立**的一条线：不再像原来那样「每小时自动 −1」（那会让民怨永远自动消失），
	//	  只能靠安抚降低；民怨 > 0 会持续掉人口（见下方人口段）。
	//
	//	旧实现的问题（已废弃）：民心按「税率档位」每小时 ±1~2，与税率只是**间接**相关；
	//	税 20% 时民心会停在 100 而不是 80，与用户的「民心+税率=100」口径不符。
	feelings := city.Feelings
	grievance := city.Grievance
	// 税率兜底到 [0,100]，避免脏数据把基准民心算成负数
	tax := city.TaxRate
	if tax < 0 {
		tax = 0
	}
	if tax > 100 {
		tax = 100
	}
	baseFeelings := 100 - tax // 民心基准 = 100 − 税率
	if feelings < baseFeelings {
		// 民心低于基准（被征服/掠夺打下来）→ 自动回归，每分钟 +1
		feelings += int(hours*60 + 0.5)
		if feelings > baseFeelings {
			feelings = baseFeelings
		}
	} else if feelings > baseFeelings {
		// 民心高于基准（刚降过税率 / 安抚加过头）→ 同样回归，每分钟 −1
		feelings -= int(hours*60 + 0.5)
		if feelings < baseFeelings {
			feelings = baseFeelings
		}
	}
	if feelings < 0 {
		feelings = 0
	}
	if feelings > 100 {
		feelings = 100
	}
	if grievance < 0 {
		grievance = 0
	}
	if grievance > 100 {
		grievance = 100
	}
	city.Feelings = feelings
	city.Grievance = grievance

	// ★★ 2026-09-26 「民心不该影响产量」：
	//
	//	原来这里有一段 `morale := feelings/100`（民怨 ≥50 再折半），
	//	然后 5 种产量全部 `× morale` —— 民心不满时所有资源一起减产，
	//	而资源详情页的「加成产量」是按「总产出 − 基础」算的，于是被算成负数
	//	（用户报「加成产量都是负的」）。
	//	**现在产量不再受民心/民怨影响**：产量只由「建筑 × 科技 × 开工率 × 市长后勤加成」
	//	决定。民心/民怨仍然保留原有作用（决定能否被征服、掠夺扣减、安抚），只是不再扣产量。

	var foodProd, steelProd, oilProd, rareProd, goldProd int64
	var popMax int64
	var goldCap, resCap int64
	// ★ 资源建筑(农田3/炼钢4/石油5/稀矿6)自带「增加容量」，需累加到对应资源上限
	//   （此前只取仓库容量 resCap，农田/炼钢/石油/稀矿的容量没算进去 → 上限 bug）
	var foodCap, steelCap, oilCap, rareCap int64
	for _, b := range d.buildingsOf(h, city.ID) {
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
	// 市长加成：产量 +(10 + 后勤/20)*3 %（★ 2026-09-27 翻三倍；复刻原版 mayorBonus）
	// ★ 2026-10-05 性能：原来这里调了**两次** mayorBonusPct（两次查库），改为只取一次。
	mayorBonus := d.mayorOf(h, city.ID)
	if mayorBonus > 0 {
		foodProd = foodProd * int64(100+mayorBonus) / 100
		steelProd = steelProd * int64(100+mayorBonus) / 100
		oilProd = oilProd * int64(100+mayorBonus) / 100
		rareProd = rareProd * int64(100+mayorBonus) / 100
	}
	// ★ 2026-09-26：不再 × morale（民心/民怨不扣产量，见上方说明）
	goldProd = int64(float64(city.Pop) * float64(city.TaxRate) / 100.0)
	// ★ 2026-09-24 修复「黄金 加成产量0 但显示[市长加成+23%]」: 市长加成同样作用于黄金
	if mayorBonus > 0 {
		goldProd = goldProd * int64(100+mayorBonus) / 100
	}

	var wildFood, wildSteel, wildOil, wildRare, wildGold int64
	wildlands := d.wildsOf(h, city.ID)
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
		// ★ 2026-10-05 用户反馈「附属野地每次更新都会没/等级还会变」：
		//   原来是 degradeWildland 按 UpdatedAt 每 2 天扣 1 级、扣到 0 直接删野地，
		//   老野地/低级野地因此频繁消失、等级跳动。按移除该降级机制，
		//   野地保持征服时的等级，不再随时间消失。
	}

	// ★ 2026-10-05 性能：增产令走快照（d.boostDone 时零 SQL），过期清理仍在 boostOf 里做。
	if boost := d.boostOf(h, city.ID, now); boost != nil {
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

	// ★ 2026-09-26 「民居容量限制」开关关掉时，民居不再限制人口 —— 自然增长
	//   不再按 pop_max 封顶（人口可无限增长）；开着时保持原行为（增长到 pop_max 就停）。
	housePopLimited := ezfyHousePopLimitOn()
	if !housePopLimited || city.Pop < city.PopMax {
		grow := int64(float64(city.PopMax) * 0.02 * hours)
		if grow < 1 {
			grow = 1
		}
		city.Pop += grow
		if housePopLimited && city.Pop > city.PopMax {
			city.Pop = city.PopMax
		}
	}
	// ★★ 2026-09-28 用户规则：**民怨 > 0 就掉人口，民怨 = 0 不掉**。
	//
	//	速率设计（「合理掉下」，别太狠也别没感觉）：
	//	  每小时流失 = pop_max × 民怨% × 2%
	//	  → 民怨 10 时 0.2%/时（一天约 4.8%）；民怨 50 时 1%/时（一天约 24%）；
	//	    民怨 100 时 2%/时（一天约 48%，很痛但不至于一夜清零）。
	//	与自然增长（2%/时）放在一起看：民怨 100 时人口净流失趋近于 0（增长被流失吃掉），
	//	民怨越高越明显 —— 既不会「毫无感觉」，也不会瞬间掏空。
	//
	//	★ 注意：流失基于 pop_max 而不是当前 pop，否则人口越低流失越慢、永远掉不干净。
	if grievance > 0 {
		lost := int64(float64(city.PopMax) * float64(grievance) / 100.0 * 0.02 * hours)
		// 有民怨时至少按小时掉 1 人，否则短时间进入页面看不出变化
		if lost < 1 && hours > 0 {
			lost = 1
		}
		city.Pop -= lost
		if city.Pop < 0 {
			city.Pop = 0
		}
	}
	// ★ 2026-09-26 「玩家城市人口不能超过配置的人口上限」：
	//   全局硬性上限（管理端可配，0 = 不限）同样封顶自然增长，与召集门同一口径。
	if hardCap := ezfyConvenePopMaxCfg(); hardCap > 0 && city.Pop > hardCap {
		city.Pop = hardCap
	}
	// ★ 「耗粮开关也做个吧，默认开」→ 关掉时城内军队每小时不扣粮。
	var troopFoodCost int64
	if ezfyFoodUpkeepOn() {
		for tid, count := range d.troopsOf(h, city.ID) {
			if cfg := ezfyCfg.troop(tid); cfg != nil {
				troopFoodCost += int64(cfg.FoodKeep) * count
			}
		}
		troopFoodCost = troopFoodCost * int64(100-techSupply*2) / 100
		troopFoodCost = int64(float64(troopFoodCost) * hours)
	}

	// ★ 2026-09-27 「资源产量也做成累加」：**唯一上限 = 资源最大值**。
	//   产量不再被仓储上限（city.FoodCap 等）卡住，与其它获取方式一样无条件累加到「资源最大值」为止。
	//   ⚠️ city.XxxCap（仓储）仅保留展示，不再作为产量收敛点 —— 永不参与计算。
	//
	// ★★ 2026-09-28 「[一键收获]/停止 资源没有入城市」事故同步修复：
	//   原来这里用 min64(prod, max64(0, prodCap-cur)) 做产出封顶 —— 与 ezfyResAddExpr
	//   的封顶**不是同一套语义**（一个「超出部分丢掉」，一个「原值不低于上限就整笔吞掉」）。
	//   现在统一成「无条件累加、只防溢出」，资源最大值不再在这里做截断
	//   —— 否则「产量正在涨、玩家又收获一笔」会互相打架。
	//   注意：老数据已超上限的城市，产量照常累加（不再出现「停在 21 亿永远不动」）。
	//
	//   ⚡ 资源最大值仍旧下发到资源详情页（getResourceCalc 的 cap 字段）供展示，
	//   玩家侧的「已满」判定统一走 ezfyAtResMax。
	// ★ 2026-09-26 城市产量倍率（管理端「二战系统配置」可调，默认 1，**0 = 产量归零**）。
	// ★ 2026-10-05 拆成两个：粮/钢/油/稀矿 用 res_prod_mult，黄金用 gold_prod_mult。
	//   乘在「城市产量 + 野地驻守产出」的**合计**上，即最终入库的那份产出。
	//   ⚠️ 必须与 `getResourceCalc`（资源详情页展示）同口径，否则「详情页显示 1 万、实际入库 100」。
	foodProd = ezfyScaleResByProdMult(foodProd)
	steelProd = ezfyScaleResByProdMult(steelProd)
	oilProd = ezfyScaleResByProdMult(oilProd)
	rareProd = ezfyScaleResByProdMult(rareProd)
	// ★ 2026-10-05 产量倍率拆开：黄金走独立的 gold_prod_mult，不再跟资源共用 res_prod_mult
	goldProd = ezfyScaleGoldByProdMult(goldProd)
	wildFood = ezfyScaleResByProdMult(wildFood)
	wildSteel = ezfyScaleResByProdMult(wildSteel)
	wildOil = ezfyScaleResByProdMult(wildOil)
	wildRare = ezfyScaleResByProdMult(wildRare)
	wildGold = ezfyScaleGoldByProdMult(wildGold)
	prod := int64(float64(foodProd)*hours) + int64(float64(wildFood)*hours)
	// ★ 2026-09-30 恢复「资源最大值唯一硬上限」：产量累加同样不得超过 21 亿。
	//
	// ★★ 修复「军队耗粮亿级时粮食卡在产量附近」：
	//   原来先算 `food := city.Food - troopFoodCost`（军队耗粮亿级 → 变成很大的负数），
	//   再 `ezfyAddResMax("food", food, prod)` —— 其内部第一步 `ezfyClampRes(cur)`
	//   就把这个负的 cur 直接夹成 0，于是那笔亿级耗粮被吞掉，只在 0 上叠了当期产量 prod。
	//   结果：耗粮超过库存+产量时，粮食不会真的扣到 0，而是停在 prod 附近
	//   （玩家体感「库存没了却一直卡在某个数」）。
	//   正确做法：把「产量 − 耗粮」作为**净增量** delta 传进去，
	//   负 delta 会让 ezfyAddResMax 正常把库存扣到 0（最低 0）。
	city.Food = ezfyAddResMax("food", city.Food, prod-troopFoodCost)
	city.Steel = ezfyAddResMax("steel", city.Steel,
		int64(float64(steelProd)*hours)+int64(float64(wildSteel)*hours))
	city.Oil = ezfyAddResMax("oil", city.Oil,
		int64(float64(oilProd)*hours)+int64(float64(wildOil)*hours))
	city.Rare = ezfyAddResMax("rare", city.Rare,
		int64(float64(rareProd)*hours)+int64(float64(wildRare)*hours))
	gold := city.Gold
	gold += int64(float64(goldProd)*hours) + int64(float64(wildGold)*hours)
	// ★ 军官工资：每名军官每小时消耗「等级 × ezfy_cfg_limit.officer_salary_per_level」黄金。
	//   与「军队耗粮」同一套懒结算口径 —— 按小时累计，离线期间照样扣。
	//   用户反馈「军官是消耗黄金的，黄金现在消耗 0」，这就是那笔消耗。
	//
	//   ⚡ 性能：officers 由调用方传入时（View / Officers / 战斗结算）用纯内存计算，
	//   零额外 SQL；未传时**兜底查一次**，保证工资一定扣得到。
	//   两种情形下本函数最多都只产生 1 次军官查询 —— 绝不会像事故版本那样重复触发。
	//   注意兜底只在真正需要结算（hours 有意义）时才走，避免空转。
	// ★ 2026-10-05 性能：军官列表在本函数里**只查一次**（工资与在职经验共用）。
	//   原来工资那一支查一次 officerList、下面 accrueDutyExp 又自己查一次 ——
	//   officerList 内部是 2 条 SQL（军官表 + 出征态自愈的命令表），跨 WAN 一次 ~230ms。
	var offList []model.EzfyOfficer
	if len(officers) > 0 {
		offList = officers[0]
	}
	per := int64(ezfyOfficerSalaryPerLvCfg())
	if per > 0 {
		if offList == nil {
			offList = h.officerList(city.ID)
		}
		gold -= int64(float64(officerSalaryOf(offList)) * hours)
	}
	if gold < 0 {
		gold = 0
	}
	city.Gold = ezfyAddResMax("gold", gold, 0) // 只做上限夹取（gold 已含工资扣减）

	// ★ 2026-09-29 市长/城守在任被动经验：随懒结算一起按时间结算
	// ★ 2026-10-05 性能：直接复用上面**同一个** offList（原来是再查一次 officerList）
	h.accrueDutyExp(city, offList)
	// 把这份（已被结算同步过的）军官列表挂进快照，展示接口可直接复用
	if d != nil && offList != nil {
		d.officers = offList
	}

	if techStore > 0 {
		// ★★ 2026-09-26 同类修复（与下面 getResourceCalc 的增产令是同一个坑）：
		//   `x *= capBonus / 100` 会**先算整数除法** `110/100 = 1` → 容量纹丝不动。
		//   改成先乘后除。
		capBonus := int64(100 + techStore*2)
		city.FoodCap = city.FoodCap * capBonus / 100
		city.SteelCap = city.SteelCap * capBonus / 100
		city.OilCap = city.OilCap * capBonus / 100
		city.RareCap = city.RareCap * capBonus / 100
		city.GoldCap = city.GoldCap * capBonus / 100
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
	// ★★ 2026-10-05 性能：**资源结算的「写合并」**。
	//
	//	线上一次 `UPDATE ezfy_city ...` 实测 ~250ms（Aurora 提交 + 跨 WAN），而前端每 30 秒就轮询一次
	//	/view —— 等于每个请求都白付一次写。但资源结算本身是**按小时**的增量，
	//	30 秒内那点产出（几万分之一小时）根本不需要立刻落库。
	//
	//	做法：距上次落库不足 `ezfyResWriteCoalesceMs` 时**只更新内存 city、不写库**。
	//	 · 本次响应仍是最新值（前端看到的资源、人口、民心都是结算后的）✔
	//	 · 下次结算从**更早的 last_time** 把这段时间一并算上 → **不会丢产出** ✔
	//	 · 进程崩溃/重启也只回退到「上次落库」那一刻，之后照常补算 ✔
	//	实测把 /view、/buildings、/officers/skills 等每请求省掉一个 ~250ms 的写往返。
	if now-last >= ezfyResWriteCoalesceMs {
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
}

// ezfyResWriteCoalesceMs 资源结算落库的最小间隔（见 calcResourceD 的写合并说明）。
//
//	60 秒：与前端 30 秒轮询错开一档，既不丢产出、又把写往返从「每请求一次」降到「每分钟一次」。
const ezfyResWriteCoalesceMs = 60 * 1000

// collectTrainQueue 训练完成懒结算。reuse 传本请求已查好的队列时可省一次查询
// （/view 30s 轮询已把 trainQueues 并入并行块，这里零重复 SQL）。
func (h *EzfyHandler) collectTrainQueue(city *model.EzfyCity, reuse ...[]model.EzfyTrainQueue) {
	now := time.Now().UnixMilli()
	var list []model.EzfyTrainQueue
	if len(reuse) > 0 && reuse[0] != nil {
		list = reuse[0]
	} else {
		h.DB.Where("city_id = ? AND status = 0", city.ID).Find(&list)
	}
	for _, q := range list {
		if q.EndTime > now {
			continue
		}
		// ★★ 2026-09-26 修复「训练完成被并发重复入库」：
		//
		//	`addTroop` 是**累加**、而 `status` 写回是**幂等**的（都写 2），
		//	所以并发请求各自跑懒结算时，同一个队列的兵会被入库 N 次 ——
		//	表面看不出异常（status 还是 2），实际兵凭空翻了 N 倍。
		//	改成条件更新抢占：只有把 status 从 0 改成 2 的那个请求才入库。
		res := h.DB.Model(&model.EzfyTrainQueue{}).
			Where("id = ? AND status = 0", q.ID).
			Update("status", 2)
		if res.Error != nil || res.RowsAffected == 0 {
			continue // 已被其他并发请求收走
		}
		h.addTroop(city.ID, q.TroopId, q.Count)
	}
}

// resCalcData 一次请求内**已经查好**的「城市只读数据快照」。
//
// ★★ 2026-10-05 性能（用户反馈「这几个接口还是 2s+」）：
//
//	本结构现在同时服务两个消费方 ——
//	  · `getResourceCalcWith`（资源详情展示）
//	  · `calcResourceD`（资源懒结算，真正入库的那一份）
//	把「建筑 / 科技 / 部队 / 野地 / 训练队列 / 增产令 / 市长加成 / 节日增产」全部预取，
//	懒结算就从「~9 条串行 RDS」变成「0 条读 + 1 条写」。
//	跨 WAN 每次往返 13~37ms，去掉 9 次就是 ~200~350ms。
//
//	任一字段为 nil / 未加载时，消费方各自退化为「自己查一次」——与改造前行为完全一致，
//	所以只有显式构造快照的调用点才会变快，其它调用点不受影响。
type resCalcData struct {
	buildings []model.EzfyCityBuilding
	techs     map[int]int
	wilds     []model.EzfyWildland
	troops    map[int]int64

	// ★ 2026-10-03 性能：/view 并行块已查好的增产令与市长加成，传入后
	//   getResourceCalcWith 不再重复打库（原来 /view 每次轮询多 2 次 RDS 往返）。
	boost *model.EzfyCityEffect // 生效中的增产令（effect_type=1）
	// ★ 2026-10-05：boostDone 区分「没查」与「查了确实没有」——
	//   原来用 `boost == nil` 表示「没查」，于是「确认没有增产令」的请求每次都要再查一遍。
	boostDone bool
	mayor     int // 市长后勤加成 %；<0 表示未传入，内部自查

	// ★★ 2026-10-05：懒结算的另外三份输入（之前每步都各自重查一遍，跨 WAN 白打往返）
	//   · cities   —— 玩家城市列表：checkTechDoneRows（多城研究）+ processOrders 都要用
	//   · trainQ   —— 训练队列：collectTrainQueue / 人口占用 / 兵力上限 / 城防空间
	//   · techRows —— 进行中科技行：checkTechDoneRows 复用
	cities   []model.EzfyCity
	trainQ   []model.EzfyTrainQueue
	techRows []model.EzfyCityTech

	// ★ 2026-10-05：懒结算里查到的军官列表（工资 + 在职经验共用），
	//   结算时会把新的 exp/level 同步回内存 → 展示接口可直接复用，不用再查一次军官表。
	officers []model.EzfyOfficer

	// ★ 2026-10-05：本请求已取到的玩家档案（ezfyPageSettle 填）。
	//   军校刷新次数上限要读 profile.recruit_free_limit，原来为此又查了一次 profile。
	profile *model.EzfyProfile
}

// ---- 统一取数入口：有快照用快照，没有就自查一次（旧行为）----

func (d *resCalcData) buildingsOf(h *EzfyHandler, cityID uint) []model.EzfyCityBuilding {
	if d != nil && d.buildings != nil {
		return d.buildings
	}
	return h.buildingList(cityID)
}

func (d *resCalcData) techsOf(h *EzfyHandler, cityID uint) map[int]int {
	if d != nil && d.techs != nil {
		return d.techs
	}
	return h.techMap(cityID)
}

func (d *resCalcData) wildsOf(h *EzfyHandler, cityID uint) []model.EzfyWildland {
	if d != nil && d.wilds != nil {
		return d.wilds
	}
	return h.wildlandList(cityID)
}

func (d *resCalcData) troopsOf(h *EzfyHandler, cityID uint) map[int]int64 {
	if d != nil && d.troops != nil {
		return d.troops
	}
	return h.troopMap(cityID)
}

func (d *resCalcData) mayorOf(h *EzfyHandler, cityID uint) int {
	if d != nil && d.mayor >= 0 {
		return d.mayor
	}
	return h.mayorBonusPct(cityID)
}

func (d *resCalcData) trainQOf(h *EzfyHandler, cityID uint) []model.EzfyTrainQueue {
	if d != nil && d.trainQ != nil {
		return d.trainQ
	}
	var qs []model.EzfyTrainQueue
	h.DB.Where("city_id = ? AND status = 0", cityID).Order("start_time ASC").Find(&qs)
	return qs
}

// officersOf 该城军官列表：懒结算已经查过就复用（且 exp/level 已同步为结算后的值），
// 否则自查一次。展示接口用它替代 `h.officerList(cityID)`，省一次跨 WAN 往返（officerList 内部 2 条 SQL）。
func (d *resCalcData) officersOf(h *EzfyHandler, cityID uint) []model.EzfyOfficer {
	if d != nil && d.officers != nil {
		return d.officers
	}
	return h.officerList(cityID)
}

// buildingLevelsOf 该城「building_id → 最高等级」内存 map（有快照用快照，没有就查一次）。
// 用于替代 `h.buildingLevel(cityID, id)` 的逐次查库（每次都是一条完整 buildingList 查询）。
func (d *resCalcData) buildingLevelsOf(h *EzfyHandler, cityID uint) map[int]int {
	return buildingLevelsOf(d.buildingsOf(h, cityID))
}

// cityIDsOf 快照里的玩家城市 id 列表（懒结算 processOrders / checkTechDoneRows 用）。
// 快照没带城市列表时回退查一次，保证行为与改造前一致。
func (d *resCalcData) cityIDsOf(h *EzfyHandler, uid uint) []uint {
	if d != nil && d.cities != nil {
		return cityIdsOf(d.cities)
	}
	return h.ezfyCityIds(uid)
}

// cityIDIntsOf 同上，但转成 processOrders 要的 []int64（避免每个调用点各写一遍循环）。
func (d *resCalcData) cityIDIntsOf(h *EzfyHandler, uid uint) []int64 {
	ids := d.cityIDsOf(h, uid)
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		out = append(out, int64(id))
	}
	return out
}

// ★★ 2026-10-05 性能：展示/操作接口统一的「取数 + 懒结算」两步走。
//
// 背景：线上库跨 WAN 每次往返 40~55ms（实测 20 条串行 SELECT = 1.08s），而这些接口原来是
// 「getOrCreateCity 串行 2 条 → refreshCityRead/refreshCity 内部再串行 ~15 条」，
// 一个页面就是 1.5~3s。现在改成：
//
//	第一波并行（1 个 RTT）：城市列表 / 建筑 / 科技 / 野地 / 部队 / 训练队列 / 进行中科技 / 增产令 / 市长
//	第二波懒结算：全部走快照 → **零额外读**，只留资源落库那 1 条写
//
// 所以 `refreshCity*` 三个函数现在都是「并行取数 + 快照结算」的薄封装 ——
// 它们被 ~30 处调用，改这里等于全部提速。
// extra 可选：调用方自己的查询（如出征下单的「防重 / 在途数 / 目标归属」三条校验），
// 会被塞进**同一个并行波**里一起发出去 —— 省掉它们在主流程里各占一个串行往返。
// ⚠️ extra 在 city 与 cities 都已就绪之后才会跑（它们通常要用 city.ID）。
func (h *EzfyHandler) ezfyLoadCityData(uid uint, city *model.EzfyCity, cities []model.EzfyCity, extra ...func()) *resCalcData {
	d := &resCalcData{}
	// ⚠️ 城市列表必须**先**拿到：下面的 techRows 要用它拼 `city_id IN (...)`。
	//   曾经把这条查询塞进并行块 → 与 techRows 的 goroutine 竞争读 `cities`，
	//   结果拼出 `IN (NULL)`、techRows 恒为空 → **科技完成结算与「本城是否已在研究」全失效**。
	if cities == nil {
		h.DB.Where("user_id = ?", uid).Order("id ASC").Find(&cities)
	}
	var wg sync.WaitGroup
	wg.Add(8 + len(extra))
	for _, fn := range extra {
		go func(f func()) { defer wg.Done(); f() }(fn)
	}
	go func() { defer wg.Done(); d.buildings = h.buildingList(city.ID) }()
	go func() { defer wg.Done(); d.techs = h.techMapOf(uid) }()
	go func() { defer wg.Done(); d.wilds = h.wildlandList(city.ID) }()
	go func() { defer wg.Done(); d.troops = h.troopMap(city.ID) }()
	go func() {
		defer wg.Done()
		var qs []model.EzfyTrainQueue
		h.DB.Where("city_id = ? AND status = 0", city.ID).Order("start_time ASC").Find(&qs)
		d.trainQ = qs
	}()
	go func() {
		defer wg.Done()
		// 进行中科技行：全玩家城市一次 IN 查完（多城研究）
		var rows []model.EzfyCityTech
		h.DB.Where("city_id IN ? AND status = 1", cityIdsOf(cities)).Find(&rows)
		d.techRows = rows
	}()
	go func() { defer wg.Done(); d.boost = h.ezfyLoadActiveBoost(city.ID) }()
	go func() { defer wg.Done(); d.mayor = h.mayorBonusPct(city.ID) }()
	wg.Wait()
	d.boostDone = true // 已确认过（含「确实没有增产令」），消费方零查询
	d.cities = cities
	return d
}

// ezfySettleCity 用快照跑懒结算。withOrders=true 时额外跑订单结算（操作接口用）；
// 展示页传 false（processOrders 每次至少 3 条串行 RDS，订单事件交给 /view 轮询推进）。
func (h *EzfyHandler) ezfySettleCity(uid uint, city *model.EzfyCity, d *resCalcData, withOrders bool) {
	h.checkBuildingDone(city, d.buildingsOf(h, city.ID))
	h.checkTechDoneRows(city, d.techRows)
	h.collectTrainQueue(city, d.trainQOf(h, city.ID))
	h.calcResourceD(city, d)
	if withOrders {
		h.processOrders(uid, d.cityIDIntsOf(h, uid))
	}
}

// ezfyPageSettle 展示页标准流程：档案 + 城市列表（1 RTT）→ 并行取数（1 RTT）→ 只读懒结算。
// 返回当前城与快照 —— 调用方**直接复用快照**（建筑等级/科技/人口/兵力上限…都别再查库）。
func (h *EzfyHandler) ezfyPageSettle(uid uint) (model.EzfyCity, *resCalcData) {
	h.cfgs()
	var profile model.EzfyProfile
	var cities []model.EzfyCity
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); profile = h.ensureProfile(uid) }()
	go func() { defer wg.Done(); h.DB.Where("user_id = ?", uid).Order("id ASC").Find(&cities) }()
	wg.Wait()
	city := h.ezfyViewCurrentCity(profile, cities)
	if len(cities) == 0 {
		cities = []model.EzfyCity{city} // 首次进游戏刚建的主城，补进列表供后续复用
	}
	d := h.ezfyLoadCityData(uid, &city, cities)
	d.profile = &profile // 供「军校刷新次数上限」等复用，省一次 profile 查询
	h.ezfySettleCity(uid, &city, d, false)
	return city, d
}

// ezfyActionSettle 操作接口标准流程：同 ezfyPageSettle，但**跑完整懒结算（含订单）**。
func (h *EzfyHandler) ezfyActionSettle(uid uint, city *model.EzfyCity) *resCalcData {
	d := h.ezfyLoadCityData(uid, city, nil)
	h.ezfySettleCity(uid, city, d, true)
	return d
}

// ezfyLoadActiveBoost 查「生效中的增产令」(effect_type=1)，已过期的顺手删除。
//
// 与 calcResource 旧行为一致（过期即清），供并行块预取用：
//
//	boost := h.ezfyLoadActiveBoost(city.ID)   // 1 条 SQL
//	snap := &resCalcData{..., boost: boost, boostDone: true}
//
// 返回 nil = 确认没有生效中的增产令（配合 boostDone=true，后续消费方零查询）。
func (h *EzfyHandler) ezfyLoadActiveBoost(cityID uint) *model.EzfyCityEffect {
	var e model.EzfyCityEffect
	if err := h.DB.Where("city_id = ? AND effect_type = 1", cityID).First(&e).Error; err != nil {
		return nil
	}
	if e.UntilTime > time.Now().UnixMilli() {
		return &e
	}
	h.DB.Delete(&e)
	return nil
}

// boostOf 取生效中的增产令。boostDone=true 表示调用方已确认过（含「确实没有」）→ 不再查库。
func (d *resCalcData) boostOf(h *EzfyHandler, cityID uint, now int64) *model.EzfyCityEffect {
	if d != nil && d.boostDone {
		if d.boost != nil && d.boost.UntilTime > now {
			return d.boost
		}
		return nil
	}
	var e model.EzfyCityEffect
	if err := h.DB.Where("city_id = ? AND effect_type = 1", cityID).First(&e).Error; err != nil {
		return nil
	}
	if e.UntilTime > now {
		return &e
	}
	// 已过期：顺手清掉（与旧 calcResource 行为一致）
	h.DB.Delete(&e)
	return nil
}

// getResourceCalc 资源详情结算（纯内存公式）。d 为 nil 时内部自查四张表；
// 传 d（如 View 已载入同一批数据）则全部复用，零额外查询。
func (h *EzfyHandler) getResourceCalc(city *model.EzfyCity) gin.H {
	return h.getResourceCalcWith(city, nil)
}

func (h *EzfyHandler) getResourceCalcWith(city *model.EzfyCity, d *resCalcData) gin.H {
	buildings := d.buildingsOf(h, city.ID)
	techs := d.techsOf(h, city.ID)
	wilds := d.wildsOf(h, city.ID)
	troops := d.troopsOf(h, city.ID)
	tech := techs
	techFood, techSteel, techOil, techRare := tech[1], tech[2], tech[3], tech[4]
	techSupply, techStore := tech[18], tech[14]
	// ★ 2026-09-26 「民心不该影响产量」→ 这里不再算 morale，产量与民心/民怨无关
	var foodBase, steelBase, oilBase, rareBase int64
	for _, b := range buildings {
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

	// ★ 与 calcResource 对齐：开工率 → 市长加成（2026-09-26 起不再乘民心）
	//   （原来详情页漏了开工率与市长加成，导致「详情页的数字」和「实际每小时产量」对不上）
	rateFood := int64(ezfyRate(city.RateFood))
	rateSteel := int64(ezfyRate(city.RateSteel))
	rateOil := int64(ezfyRate(city.RateOil))
	rateRare := int64(ezfyRate(city.RateRare))
	// ★ 2026-10-05 性能：原来无条件先查一次 mayorBonusPct（哪怕调用方已经把 mayor 传进来），
	//   等于每个 /view 请求白打一条 SQL。现在由 mayorOf 决定「用快照还是自查」。
	mayor := int64(d.mayorOf(h, city.ID))
	applyProd := func(base, rate int64) int64 {
		v := base * rate / 100
		if mayor > 0 {
			v = v * (100 + mayor) / 100
		}
		return v
	}
	foodProd := applyProd(foodBaseTech, rateFood)
	steelProd := applyProd(steelBaseTech, rateSteel)
	oilProd := applyProd(oilBaseTech, rateOil)
	rareProd := applyProd(rareBaseTech, rateRare)
	goldProd := int64(float64(city.Pop) * float64(city.TaxRate) / 100.0)
	// ★ 2026-09-24 与 calcResource 对齐: 市长加成同样作用于黄金(否则加成产量恒 0)
	if mayor > 0 {
		goldProd = goldProd * (100 + mayor) / 100
	}

	// ★★ 2026-09-26 修复「加成产量(每小时) 全是负数」：
	//
	//	原来 base 只到「建筑 × 科技」，而 bonus 用「总产出 − 建筑×科技」算 ——
	//	`foodProd` 里已经乘过**开工率**和**民心**，`foodBaseTech` 没有，
	//	于是「开工率不满 / 民心不满造成的减产」被算进了「加成」：
	//	  民心 80% → bonus = baseTech×0.8 − baseTech = **−20% × baseTech**（负的）
	//	所有资源都受同一套系数影响，玩家看到的就是「所有资源加成产量都是负的」。
	//
	//	现在的口径（界面 4 行仍自洽：基础 + 加成 − 耗量 = 总产量）：
	//	  · base  = 建筑 × 科技 × 开工率     ← 实际基础产出（不含市长/道具/活动）
	//	  · bonus = 总产出 − base + 野地      ← 只剩真正的**正向加成**，恒 ≥ 0
	//	★ 另按用户 2026-09-26 要求：**民心/民怨不再影响产量**（原来 base 里还有 × 民心），
	//	  产量只由「建筑 × 科技 × 开工率 × 市长后勤加成」决定。
	//	  · total 与 calcResource 同口径（本次只改拆分，不改实际产量公式）
	realBase := func(base, rate int64) int64 {
		return base * rate / 100
	}
	foodBaseReal := realBase(foodBaseTech, rateFood)
	steelBaseReal := realBase(steelBaseTech, rateSteel)
	oilBaseReal := realBase(oilBaseTech, rateOil)
	rareBaseReal := realBase(rareBaseTech, rateRare)
	// 黄金没有开工率
	goldBaseReal := goldBase

	var wildFood, wildSteel, wildOil, wildRare, wildGold int64
	for _, w := range wilds {
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
	boostPct := 0
	boostUntil := int64(0)
	// ★ 2026-10-05 性能：增产令统一走 boostOf —— 快照命中时零 SQL；
	//   boostDone=true 让「已确认没有增产令」的请求也不再重复查库。
	boost := d.boostOf(h, city.ID, time.Now().UnixMilli())
	if boost != nil {
		// ★★ 2026-09-26 修复「增产令用了没加成（详情页不显示）」：
		//
		//	原来这里写的是 `foodProd *= mult / 100` —— Go 里这等价于
		//	`foodProd = foodProd * (mult / 100)`，**先算 `mult/100`**，
		//	而 `150/100` 是**整数除法 = 1** → 乘了个 1，产量纹丝不动。
		//	（`calcResource` 里写的是 `foodProd = foodProd * mult / 100`，先乘后除 = 2400 ✓，
		//	 所以「实际入库有 +50%、详情页显示没有」——用户看到界面没变，报「没加成」。）
		//	⚠️ 凡是 `x * pct / 100` 一律**先乘后除**，别写 `x *= pct / 100`。
		mult := int64(100 + boost.Param1)
		boostPct = boost.Param1
		boostUntil = boost.UntilTime
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
	// ★ 2026-09-26 城市产量倍率（管理端「二战系统配置」可调，默认 1，**0 = 产量归零**）。
	// ★ 2026-10-05 拆成两个：粮/钢/油/稀矿 用 res_prod_mult，黄金用 gold_prod_mult。
	//   放在所有加成（市长/道具增产/节日活动）**之后**，与 `calcResource` 同口径。
	//   ⚠️ **`base` 也要一起乘**：倍率是「产量系数」而不是「加成」——
	//   只乘总产出的话，倍率 < 1 时 `bonus = 总产出 − base` 会变成**负数**
	//   （实测倍率 0 时 base=1600 / bonus=-1600），违反「加成产量恒 ≥ 0」的既定口径。
	//   一起乘之后：倍率 2 → base 翻倍、bonus 不变；倍率 0 → base 和 bonus 都归 0。
	foodBaseReal = ezfyScaleResByProdMult(foodBaseReal)
	steelBaseReal = ezfyScaleResByProdMult(steelBaseReal)
	oilBaseReal = ezfyScaleResByProdMult(oilBaseReal)
	rareBaseReal = ezfyScaleResByProdMult(rareBaseReal)
	goldBaseReal = ezfyScaleGoldByProdMult(goldBaseReal)
	foodProd = ezfyScaleResByProdMult(foodProd)
	steelProd = ezfyScaleResByProdMult(steelProd)
	oilProd = ezfyScaleResByProdMult(oilProd)
	rareProd = ezfyScaleResByProdMult(rareProd)
	// ★ 2026-10-05 产量倍率拆开：黄金走独立的 gold_prod_mult，不再跟资源共用 res_prod_mult
	goldProd = ezfyScaleGoldByProdMult(goldProd)
	wildFood = ezfyScaleResByProdMult(wildFood)
	wildSteel = ezfyScaleResByProdMult(wildSteel)
	wildOil = ezfyScaleResByProdMult(wildOil)
	wildRare = ezfyScaleResByProdMult(wildRare)
	wildGold = ezfyScaleGoldByProdMult(wildGold)
	var troopFood int64
	// ★ 耗粮开关关掉时这里也要显示 0，否则界面写着「每小时耗粮 N」，实际却不扣
	if ezfyFoodUpkeepOn() {
		for tid, count := range troops {
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
	// ★ 2026-09-26 起不再下发 morale_pct：民心/民怨已不参与产量，前端 base 行不再显示民心
	// ★ 2026-09-27 唯一上限 = 资源最大值：cap 下发 ezfyResMaxOf，不再用仓储 city.XxxCap。
	// ★ 2026-09-30 「增产令使用了要在资源详情简约体现」：
	//   把增产幅度/剩余时长透出到每种资源，前端在加成产量行显示 [增产令+N%]。
	// ★ 2026-10-06 黄金「耗量(每小时)」显示军官工资：原来下发 consume=0，
	//   而 calcResource 懒结算里每小时都实扣军官工资 —— 页面显示与实际扣费对不上
	//   （用户：「耗量(每小时): 0 里现在不对」）。资源详情页非热路径，允许查库算工资。
	goldConsume := h.officerSalaryPerHour(city.ID)
	return gin.H{
		"food": item(city.Food, ezfyResMaxOf("food"), foodBaseReal, foodProd-foodBaseReal+wildFood, troopFood, foodProd+wildFood-troopFood,
			gin.H{"tech_prod": techFood, "troop_consume": troopFood, "troop_consume_raw": troopFoodRaw,
				"supply_tech": techSupply, "base_building": foodBase, "rate": rateFood, "mayor_bonus": mayor, "boost_pct": boostPct, "boost_until": boostUntil}),
		"steel": item(city.Steel, ezfyResMaxOf("steel"), steelBaseReal, steelProd-steelBaseReal+wildSteel, 0, steelProd+wildSteel,
			gin.H{"tech_prod": techSteel, "base_building": steelBase, "rate": rateSteel, "mayor_bonus": mayor, "boost_pct": boostPct, "boost_until": boostUntil}),
		"oil": item(city.Oil, ezfyResMaxOf("oil"), oilBaseReal, oilProd-oilBaseReal+wildOil, 0, oilProd+wildOil,
			gin.H{"tech_prod": techOil, "base_building": oilBase, "rate": rateOil, "mayor_bonus": mayor, "boost_pct": boostPct, "boost_until": boostUntil}),
		"rare": item(city.Rare, ezfyResMaxOf("rare"), rareBaseReal, rareProd-rareBaseReal+wildRare, 0, rareProd+wildRare,
			gin.H{"tech_prod": techRare, "base_building": rareBase, "rate": rateRare, "mayor_bonus": mayor, "boost_pct": boostPct, "boost_until": boostUntil}),
		"gold": item(city.Gold, ezfyResMaxOf("gold"), goldBaseReal, goldProd-goldBaseReal+wildGold, goldConsume, goldProd+wildGold-goldConsume,
			gin.H{"tech_prod": 0, "base_building": goldBase, "rate": 100, "mayor_bonus": mayor, "boost_pct": boostPct, "boost_until": boostUntil}),
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
	// ★ 2026-10-05 必须连 `last_time` 一起写：calcResourceD 做了「写合并」（不足 1 分钟不落库），
	//   这里若只写资源不写 last_time，下次结算会从**旧的 last_time** 再把这段时间的产出算一遍
	//   → 资源凭空翻倍。带上 last_time 表示「截止该时刻的产出已落库」。
	//   （city.LastTime 未经过懒结算时就是库里的原值，写回等于空操作，安全。）
	h.DB.Model(&model.EzfyCity{}).Where("id = ?", city.ID).Updates(map[string]interface{}{
		"gold": city.Gold, "food": city.Food, "steel": city.Steel,
		"oil": city.Oil, "rare": city.Rare,
		"last_time": city.LastTime,
	})
}

// cancelBuildingUpgrade 取消施工中的建筑升级，并**全额退还**已扣的资源与图纸
//
// ★ 2026-09-26 「已升级的建筑用户端去掉（多余的升级按钮），加个升级状态时 [取消] 功能」。
//
//	退还口径与扣费口径**严格对称**（否则会变成刷资源漏洞）：
//	  · 普通升级（`start_time != 0`）：只扣过 `b.Level+1` 这一级 → 只退这一级
//	  · 连锁升级（`start_time == 0`，一键 N 级）：`maxLevelBuilding` 是**一次性扣了
//	    `b.Level+1 ~ target_level` 的全部费用与图纸** → 要全退
//	  · 旧数据连锁（`target_level == 0`，一键升满）：退未建成部分 `当前Level+1 ~ 建筑上限`
//	  · 建筑还没建成（`level == 0`）：取消 = 撤销建造（删记录 + 退 1 级费用）
//
// ★★ 2026-09-26 修复「取消不退换资源」：
//   - **先读建筑再 refreshCity**。原来先 `refreshCity()`（其内部 `checkBuildingDone`
//     会把已到结束时间的建筑**直接结算成完工**，status 置 0、level+1），随后才读建筑，
//     于是读到 status==0 就返回「该建筑没有在施工」→ 一分不退，而玩家 UI 还显示施工中，
//     看起来就像「取消失败了」。
//   - 建筑单次升级耗时很短（BuildTime 秒级，下限 1 秒），玩家点取消时往往**已完工**。
//     对「本次取消请求开始时仍在施工（原 status != 0）、恰在此步被 `checkBuildingDone`
//     结算成完工」的情况：已建成，按「该建筑已完成建造」处理，保留新等级、不退任何资源。
// ★★ 2026-09-27 严重漏洞修复：**取消升级零退还，建筑保留当前等级**。
//    升级/一键升满开局就一次性扣光资源+图纸（见 upgradeBuilding / maxLevelBuilding），
//    而旧逻辑取消时又 giveResNoCap + addItem 全额退还 → 玩家可「一键升满 → 立即取消」反复刷，
//    还能连 9→10 / 民居 11+ 的建筑图纸一起刷。现在无论一键还是单级升级：
//    **取消后不退还任何资源/图纸，且建筑保留当前等级**（不回退、不推进），只停止施工；
//    防并发/重复取消仍用 CAS 抢占领位。
func (h *EzfyHandler) cancelBuildingUpgrade(city *model.EzfyCity, recordId int64) string {
	// 1) 先读建筑；如果本来就空闲，直接返回（不提前刷资源）
	var b0 model.EzfyCityBuilding
	if err := h.DB.Where("id = ? AND city_id = ?", recordId, city.ID).First(&b0).Error; err != nil {
		return "建筑不存在"
	}
	if b0.Status == 0 {
		return "该建筑没有在施工"
	}
	// 2) refreshCity：把「已到结束时间」的建筑懒结算掉，避免卡死在施工中；也用于识别「恰在此刻完工」
	h.refreshCity(city.UserID, city)
	var b model.EzfyCityBuilding
	if err := h.DB.Where("id = ? AND city_id = ?", recordId, city.ID).First(&b).Error; err != nil {
		return "建筑不存在"
	}
	// 3) 升级已在此次取消前完工 → 已建成，无「取消」可言；资源/图纸是该次**真实升级**的投入，不退。
	//    （完工时 checkBuildingDone 已加过对应声望与新等级，这里保留现状，不扣回。）
	if b.Status == 0 {
		return "该建筑已完成建造"
	}
	// 4) 仍在施工中 → 取消：只停止施工（status=0、清掉暂存的目标/时间），**保留当前等级**，
	//    **不退任何资源或图纸**。CAS 抢占（Where status<>0）只允许一方成功，避免并发/重复取消重复处理。
	res := h.DB.Model(&model.EzfyCityBuilding{}).
		Where("id = ? AND status <> 0", b.ID).
		Updates(map[string]interface{}{
			"status": 0, "target_level": 0, "start_time": 0, "end_time": 0,
		})
	if res.Error != nil {
		return "取消失败，请重试"
	}
	if res.RowsAffected == 0 {
		// 没抢到：说明已被别的取消/结算抢先，避免重复处理
		return "该建筑没有在施工"
	}
	return "已取消升级（资源与建筑图纸不作退还，建筑保留当前等级）"
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
	// ★ 军事区 / 资源区数量上限**分开**（线上现值各 36，管理端可维护）
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
	//   规则（2026-09-23 用户修正）：**所有建筑 9→10 级**都需图纸；
	//   民居(2) 10→11、11→12 也要。
	//   ★ 2026-09-30 用户追加：**司令部(13) 10级及以后每升一级都需要图纸**（9→10、10→11、11→12）。
	needBlueprint := target == 10 ||
		((b.BuildingId == 2 || b.BuildingId == 13) && target >= 11)
	if needBlueprint && h.itemCount(city.UserID, ezfyBlueprintItemID) <= 0 {
		return fmt.Sprintf("升级到%d级需要建筑图纸", target)
	}
	if !h.pay(city, lv) {
		return "资源不足"
	}
	if needBlueprint {
		h.consumeItem(city.UserID, ezfyBlueprintItemID, "建筑升级")
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

// maxLevelBuilding 一键升级建筑：升到 targetLevel（0 = 升到该建筑上限）
//
// ★ 2026-09-25 用户纠正「一键9级 不对，是一键升级到 9 级，而不是升级满」：
//   前端按钮文案是「一键{{max_level-1}}级」（市政厅 10 级 → 一键9级），
//   语义就是「一键升到那一级为止」——**不能**越过它去升满级：
//   9→10 是要建筑图纸的坎，玩家点「一键9级」本意就是只到 9 级。
//   现在按 targetLevel 结算（资源/图纸都只算到目标级），并且施工一次到位。
//   返回 (实际目标等级, 错误信息)；错误信息为空表示成功。
func (h *EzfyHandler) maxLevelBuilding(city *model.EzfyCity, recordId int64, targetLevel int) (int, string) {
	h.refreshCity(city.UserID, city)
	var b model.EzfyCityBuilding
	if err := h.DB.Where("id = ? AND city_id = ?", recordId, city.ID).First(&b).Error; err != nil {
		return 0, "建筑不存在"
	}
	if b.Status != 0 {
		return 0, "建筑正在施工中"
	}
	cfg := ezfyCfg.building(b.BuildingId)
	if cfg == nil {
		return 0, "建筑配置缺失"
	}
	// ★ 第九轮等级规则（市政厅10 / 参谋部·司令部·民居12 / 其他10，民居 ≤ 市政厅+1）
	maxLv := h.buildingMaxLevel(city.ID, b.BuildingId)
	// ★ 2026-09-25：目标等级 = 前端下发的 target_level（缺省/越界时按建筑上限兜底）
	target := maxLv
	if targetLevel > 0 && targetLevel < maxLv {
		target = targetLevel
	}
	if target > maxLv {
		target = maxLv
	}
	if b.Level >= target {
		if targetLevel > 0 && targetLevel < maxLv {
			return 0, fmt.Sprintf("该建筑已是%d级", b.Level)
		}
		return 0, "该建筑已满级"
	}
	var needFood, needSteel, needOil, needRare, needGold int64
	needBlueprint := 0
	for lv := b.Level + 1; lv <= target; lv++ {
		// ★ 与 upgradeBuilding 同口径：所有建筑 9→10、民居(2) 10→11 / 11→12 需图纸；
		//   ★ 2026-09-30 司令部(13) 10级及以后每级都需图纸
		if lv == 10 || ((b.BuildingId == 2 || b.BuildingId == 13) && lv >= 11) {
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
		return 0, fmt.Sprintf("一键升到%d级需要%d张建筑图纸(当前不足)", target, needBlueprint)
	}
	if city.Food < needFood || city.Steel < needSteel || city.Oil < needOil ||
		city.Rare < needRare || city.Gold < needGold {
		return 0, fmt.Sprintf("资源不足: 升到%d级需 粮%d 钢%d 油%d 稀矿%d 金%d",
			target, needFood, needSteel, needOil, needRare, needGold)
	}
	city.Food -= needFood
	city.Steel -= needSteel
	city.Oil -= needOil
	city.Rare -= needRare
	city.Gold -= needGold
	h.saveCityRes(city)
	if needBlueprint > 0 {
		h.consumeItemN(city.UserID, ezfyBlueprintItemID, needBlueprint, "建筑升级")
	}
	now := time.Now().UnixMilli()
	h.DB.Model(&model.EzfyCityBuilding{}).Where("id = ?", b.ID).
		Updates(map[string]interface{}{"status": 2, "start_time": 0,
			"end_time": now + ezfyMaxUpgradeSeconds*1000, "target_level": target})
	return target, ""
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
	// ★ 拆除是一级一级拆，而不是直接整栋拆没；降到 0 级才彻底移除
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
	return h.defenceSpaceUsedD(h.troopMap(cityId), h.trainQOfRaw(cityId))
}

// defenceSpaceUsedD 同 defenceSpaceUsed，但复用已查好的部队表 + 训练队列（零 SQL）。
func (h *EzfyHandler) defenceSpaceUsedD(troops map[int]int64, trainQ []model.EzfyTrainQueue) int64 {
	var used int64
	for tid, cnt := range troops {
		if c := ezfyCfg.troop(tid); c != nil && c.Type == 4 {
			used += cnt
		}
	}
	for _, q := range trainQ {
		if c := ezfyCfg.troop(q.TroopId); c != nil && c.Type == 4 {
			used += q.Count
		}
	}
	return used
}

// trainQOfRaw 该城进行中的训练队列（无快照时的兜底查询）。
func (h *EzfyHandler) trainQOfRaw(cityId uint) []model.EzfyTrainQueue {
	var qs []model.EzfyTrainQueue
	h.DB.Where("city_id = ? AND status = 0", cityId).Order("start_time ASC").Find(&qs)
	return qs
}

// trainTroop 训练/建造入口：先按城市分片加锁，再执行真正的训练逻辑。
// 锁保证「读已占用 → 校验城防空间 → 建队列」原子化，杜绝并发超容。
func (h *EzfyHandler) trainTroop(city *model.EzfyCity, troopId, count int, split bool) string {
	lock := ezfyTrainLock(uint(city.ID))
	lock.Lock()
	defer lock.Unlock()
	return h.trainTroopLocked(city, troopId, count, split)
}

// trainTroopLocked 训练/建造城防的真正逻辑。
//
// ★★ 2026-10-05 性能（用户反馈「/troops/train 3s」）：改造前本函数 = refreshCity（~15 条串行）
//
//	+ 需求里的 buildingLevel / techMap（每个需求各查一次）+ cityPopUsed（2 条）
//	+ checkTroopCap（2 条）+ 围墙等级（1 条）+ defenceSpaceUsed（2 条）≈ **30 条几乎全串行**。
//	现在先一次并行取数（ezfyActionSettle 里 1 个 RTT），下面全部走内存 map/切片：
//	  · 建筑等级 → blv（一次 buildingList 的产物）    · 科技等级 → tech
//	  · 人口占用 → cityPopUsedD（建筑 + 训练队列）    · 兵力上限 → checkTroopCapD
//	  · 城防空间 → defenceSpaceUsedD
//	d 为空时兜底自己建一份（行为不变，只是慢）。
func (h *EzfyHandler) trainTroopLocked(city *model.EzfyCity, troopId, count int, split bool, d ...*resCalcData) string {
	var snap *resCalcData
	if len(d) > 0 && d[0] != nil {
		snap = d[0]
	} else {
		snap = h.ezfyActionSettle(city.UserID, city)
	}
	blv := snap.buildingLevelsOf(h, city.ID)
	tech := snap.techsOf(h, city.ID)
	troops := snap.troopsOf(h, city.ID)
	trainQ := snap.trainQOf(h, city.ID)

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
					if blv[bid] < need {
						return fmt.Sprintf("需要%s %d级", name, need)
					}
					continue
				}
				if tid, ok := ezfyCfg.techByName[name]; ok {
					if tech[tid] < need {
						return fmt.Sprintf("需要科技%s %d级", name, need)
					}
				}
			}
		} else {
			if bid, ok := ezfyCfg.buildingByName[cfg.Require]; ok && blv[bid] < 1 {
				return fmt.Sprintf("需要%s 1级", cfg.Require)
			}
		}
	}
	// 人口校验：只跟「正在训练、还没出厂」的兵比 —— 已训练完成的部队不占人口（用户规则）
	// ★ 「征兵资源消耗开关关了的话，征兵不消耗资源，也无需空闲人口」→
	//   开关关掉时整段跳过（不校验人口、不扣资源）。
	recruitCost := ezfyRecruitCostOn()
	if recruitCost {
		popUsed := h.cityPopUsedD(city.ID, snap.buildingsOf(h, city.ID), trainQ)
		popAvailable := city.Pop - popUsed
		if cfg.Type != 4 && int64(cfg.Pop)*int64(count) > popAvailable {
			return fmt.Sprintf("人口不足(当前居民%d, 建筑及训练已占用%d, 可用%d); 可召集人口突破民居上限",
				city.Pop, popUsed, popAvailable)
		}
	}
	// ★ 2026-09-23 「超过限制不能训练，提示超过限额」：
	//   兵力累加没有任何上限，单兵种 count 撑爆 int64 就翻成负数
	//   （线上事故：玩家总兵力 -8843547888967622000）。
	//   这里按 ezfy_cfg_limit.troop_max 统一卡控（城内现有 + 训练队列 + 本次）。
	//   城防(type 4)同样存在 ezfy_city_troop 里、同样会溢出，所以一并卡。
	if msg := h.checkTroopCapD(int64(count), troops, trainQ); msg != "" {
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
		wallLevel := blv[7]
		space := int64(0)
		if wall := ezfyCfg.buildingLevel(7, wallLevel); wall != nil {
			space = wall.Capacity
		}
		used := h.defenceSpaceUsedD(troops, trainQ)
		if used+int64(count) > space {
			return fmt.Sprintf("城防空间不足(围墙%d级, 上限%d, 已占用%d)", wallLevel, space, used)
		}
	}
	factoryTotal := h.buildingTotalLevel(city.ID, ezfyFactoryBuildingID)
	var activeCount int64
	h.DB.Model(&model.EzfyTrainQueue{}).Where("city_id = ? AND status = 0", city.ID).Count(&activeCount)
	// ★ 2026-10-05 军工厂机制（用户规则）：
	//   每个军工厂最大队列数 = 自己等级；
	//   训练某兵种时只有「等级 ≥ 需求(need_factory)」的军工厂可用，
	//   队列上限 = 可用军工厂等级合计（原实现把不满足需求的厂也算进上限，虚高）；
	//   [全部工厂] 模式 = 把数量平分给全部可用军工厂（各开一队列）→ 预计耗时 = 单个耗时×数量/可用厂数。
	needFactory := 0
	if cfg.Type != 4 {
		needFactory = ezfyNeedFactoryLevel(cfg.Require)
	}
	queueLimit := 0
	eligible := 0
	if cfg.Type == 4 {
		queueLimit = maxInt(1, factoryTotal)
	} else {
		var factories []model.EzfyCityBuilding
		h.DB.Where("city_id = ? AND building_id = ? AND status = 0", city.ID, ezfyFactoryBuildingID).Find(&factories)
		for _, f := range factories {
			if needFactory <= 0 || f.Level >= needFactory {
				queueLimit += f.Level
				eligible++
			}
		}
	}
	if queueLimit <= 0 {
		return "请先建造军工厂"
	}
	if int(activeCount) >= queueLimit {
		return fmt.Sprintf("训练队列已满(可用军工厂等级合计%d个队列)", queueLimit)
	}
	n := 1
	if split && cfg.Type != 4 {
		free := queueLimit - int(activeCount)
		n = maxInt(1, minInt(eligible, free))
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
	// ★ 「征兵资源消耗开关关了，征兵无需空闲人口」→ 关掉时人口占用恒为 0，
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

// cityPopUsedD 与 cityPopUsed 同口径，但复用本请求已查好的建筑列表 / 训练队列：
// 两者都非 nil 时零 SQL（原来各自再查一次 buildingList / train_queue）。
func (h *EzfyHandler) cityPopUsedD(cityId uint, buildings []model.EzfyCityBuilding, trainQ []model.EzfyTrainQueue) int64 {
	pop := int64(0)
	if buildings != nil {
		for _, b := range buildings {
			if lv := ezfyCfg.buildingLevel(b.BuildingId, b.Level); lv != nil {
				pop += int64(lv.Pop)
			}
		}
	} else {
		pop += h.buildingPop(cityId)
	}
	if !ezfyRecruitCostOn() {
		return pop
	}
	if trainQ == nil {
		return pop + h.troopPop(cityId)
	}
	for _, q := range trainQ {
		if cfg := ezfyCfg.troop(q.TroopId); cfg != nil && cfg.Type != 4 {
			pop += int64(cfg.Pop) * q.Count
		}
	}
	return pop
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

func (h *EzfyHandler) recoverWounded(city *model.EzfyCity, troopId, wtype int, count int64) string {
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
	// ★ 2026-10-06 「加个恢复数量，不然全部整不起」：count>0 且小于在营数量 =
	//   只恢复指定数量（其余留在营里下次再恢复）；count<=0 或大于等于在营数量 = 全部恢复（兼容旧行为）。
	if count <= 0 || count >= w.Count {
		count = w.Count
	}
	// ★ 2026-09-23 「恢复的数量导致负数的情况也卡控，不能恢复」。
	//   恢复 = 往城里加兵，所以和训练共用同一个兵力上限校验。
	if msg := h.checkTroopCap(city.ID, count); msg != "" {
		return msg
	}
	h.calcResource(city)
	cost := ezfyWoundHealGoldPer(w.TroopId) * count
	if city.Gold < cost {
		return fmt.Sprintf("黄金不足: 恢复%d个需要%d黄金, 当前只有%d", count, cost, city.Gold)
	}
	city.Gold -= cost
	h.saveCityRes(city)
	h.addTroop(city.ID, w.TroopId, count)
	if count >= w.Count {
		h.DB.Delete(&w)
	} else {
		w.Count -= count
		h.DB.Model(&model.EzfyWounded{}).Where("id = ?", w.ID).Update("count", w.Count)
	}
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

// researchTech 开始研究。
//
// ★★ 2026-10-05 性能（用户反馈「/techs/research 3s」）：改造前本函数 = refreshCity（~15 条串行）
//
//	+ buildingLevel ×2（每次一条完整 buildingList）+ techMap ×2（每次 2 条）+ ezfyCityIds
//	+ dup/busy 两个 Count ≈ **30 条几乎全串行的跨 WAN 往返** → 1.5~3s。
//	现在：可选传入调用方已建好的快照 d（ezfyActionSettle），全部改走内存：
//	  · 建筑等级 → d.buildingLevelsOf（一次 buildingList 的产物）
//	  · 科技等级 → d.techsOf
//	  · 「本城/同科技是否已在研究」→ 直接用 d.techRows（checkTechDoneRows 已把完成的行置 0）
//	d 为空时兜底自己建一份（行为不变，只是慢）。
func (h *EzfyHandler) researchTech(city *model.EzfyCity, techId int, d ...*resCalcData) string {
	var snap *resCalcData
	if len(d) > 0 && d[0] != nil {
		snap = d[0]
	} else {
		snap = h.ezfyActionSettle(city.UserID, city)
	}
	blv := snap.buildingLevelsOf(h, city.ID)
	tech := snap.techsOf(h, city.ID)

	cfg := ezfyCfg.tech(techId)
	if cfg == nil {
		return "科技不存在"
	}
	academyNeed := 1
	if v, ok := ezfyTechAcademy[techId]; ok {
		academyNeed = v
	}
	// ★ 2026-09-28 研究限制来自**当前城市**的科研中心等级（不再取全城最高）
	if blv[8] < academyNeed {
		return fmt.Sprintf("本城需要科研中心%d级才能研究%s（当前%d级）", academyNeed, cfg.Name, blv[8])
	}
	curLevel := tech[techId]
	if curLevel >= cfg.MaxLevel {
		return "已达到最高等级"
	}
	lv := ezfyCfg.techLevel(techId, curLevel+1)
	if lv == nil {
		return "配置缺失"
	}
	if cfg.PreTech > 0 {
		if tech[cfg.PreTech] < cfg.PreTechLevel {
			pre := ezfyCfg.tech(cfg.PreTech)
			preName := ""
			if pre != nil {
				preName = pre.Name
			}
			return fmt.Sprintf("需要先研究%s %d级", preName, cfg.PreTechLevel)
		}
	}
	// ★ 2026-09-28 多城研究互斥：**同一科技**同一时刻只能在一个城市研究。
	//   不同城市可以各研究各的（互不抢槽），但同一科技撞了就拦下。
	// ★ 2026-10-05：直接用快照里的「进行中科技行」判定，省掉两条 Count 跨 WAN 往返。
	//   注意跳过 Status != 1 的行 —— checkTechDoneRows 会把**本请求刚结算完**的行在内存里置 0。
	for i := range snap.techRows {
		if snap.techRows[i].Status != 1 {
			continue
		}
		if snap.techRows[i].TechId == techId {
			return fmt.Sprintf("%s 已在其他城市研究中, 不能重复研究", cfg.Name)
		}
	}
	// ★ 2026-09-28 fix：每个城市同时只能有**一条**研究队列（无论什么科技），避免本城开多条队列。
	for i := range snap.techRows {
		if snap.techRows[i].Status != 1 {
			continue
		}
		if snap.techRows[i].CityId == int64(city.ID) {
			return "本城已有科技在研究, 请先完成或取消后再研究"
		}
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
	// ★ 2026-09-28 等级记录在用户级（ezfy_user_tech），研究队列记录落在**发起城市**。
	//   Level 存开始时的全局等级，结算时按它 +1 写回用户级等级。
	// ★ 2026-09-28 fix：ezfy_city_tech 上 (city_id, tech_id) 是唯一索引，升级/取消后再研究
	//   会走到同一行 —— 必须 upsert（存在则复用激活），否则 Create 因唯一冲突静默失败且资源已扣。
	h.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "city_id"}, {Name: "tech_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"level", "status", "end_time"}),
	}).Create(&model.EzfyCityTech{CityId: int64(city.ID), TechId: techId, Level: curLevel,
		Status: 1, EndTime: now + h.techResearchMs(lv.ResearchTime)})
	return ""
}

// cancelTech 取消研究: 清除研究队列, 研究等级不变。
// ★ 2026-09-28 用户口径：取消研究**不退还**已消耗资源（与建筑取消不同, 仅停止该研究）。
func (h *EzfyHandler) cancelTech(city *model.EzfyCity, techId int) string {
	var t model.EzfyCityTech
	// ★ 2026-09-28 研究队列记录在「发起城市」；取消按当前城 + 该科技查进行中记录
	if err := h.DB.Where("city_id = ? AND tech_id = ? AND status = 1", city.ID, techId).First(&t).Error; err != nil {
		return "该科技没有在研究中"
	}
	h.DB.Delete(&model.EzfyCityTech{}, t.ID)
	return ""
}

// checkTechDone 科技完成懒结算。reuse 传本请求已查好的「玩家城市 id 列表」时
// 可省一次 ezfyCityIds 查询（/view 30s 轮询已把 cities 并入并行块，这里零重复 SQL）。
func (h *EzfyHandler) checkTechDone(city *model.EzfyCity, reuse ...[]uint) {
	// ★ 2026-09-28 多城研究：进行中的队列记录分布在玩家各城，全部都要结算；
	//   等级 +1 写到用户级 ezfy_user_tech（全城共用、无主城概念）。
	var list []model.EzfyCityTech
	cityIds := h.ezfyCityIds(city.UserID)
	if len(reuse) > 0 && len(reuse[0]) > 0 {
		cityIds = reuse[0]
	}
	h.DB.Where("city_id IN ? AND status = 1", cityIds).Find(&list)
	h.checkTechDoneRows(city, list)
}

// checkTechDoneRows 对已查好的「进行中科技」做完成结算（写部分；list 由调用方预取可省 1 条串行 RTT）。
// ★ 2026-10-05 结算成功的行把内存里的 Status 置 0，调用方（techs 页）据此直接从这批行
//   构建「研究中」集合，不再多查一次 status=1 的记录。
func (h *EzfyHandler) checkTechDoneRows(city *model.EzfyCity, list []model.EzfyCityTech) {
	now := time.Now().UnixMilli()
	for i := range list {
		t := &list[i]
		if now < t.EndTime {
			continue
		}
		// ★ 2026-09-26 同 checkBuildingDone：条件更新抢占，避免并发重复结算
		res := h.DB.Model(&model.EzfyCityTech{}).
			Where("id = ? AND status = 1", t.ID).
			Updates(map[string]interface{}{"status": 0})
		if res.Error != nil || res.RowsAffected == 0 {
			continue
		}
		t.Status = 0 // 已结算完，内存同步标记（供调用方复用）
		// 等级 +1 写用户级（无记录则建）
		h.DB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "tech_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"level"}),
		}).Create(&model.EzfyUserTech{UserId: city.UserID, TechId: t.TechId, Level: t.Level + 1})
		h.taskProgress(city.UserID, "tech_research", 1)
	}
}

func (h *EzfyHandler) speedUpTech(city *model.EzfyCity, minutes int64) string {
	var t model.EzfyCityTech
	// ★ 2026-09-28 研究队列记录在「发起城市」，加速按当前城查
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

// itemCounts 一次查询取该玩家多件道具的持有数（map[cfgId]count，缺省 0）。
// ★ 2026-10-04 军官详情原来对升星卡/改名卡/技能书各查一次 itemCount（3 条 SQL），
//   合并成一条「user_id + cfg_id IN」查询。
func (h *EzfyHandler) itemCounts(uid uint, cfgIds ...int) map[int]int {
	out := map[int]int{}
	if len(cfgIds) == 0 {
		return out
	}
	var items []model.EzfyItem
	h.DB.Where("user_id = ? AND cfg_id IN ?", uid, cfgIds).Find(&items)
	for _, it := range items {
		out[it.CfgId] = it.Count
	}
	return out
}

func (h *EzfyHandler) addItem(uid uint, cfgId, count int) {
	if count <= 0 {
		return
	}
	var it model.EzfyItem
	if err := h.DB.Where("user_id = ? AND cfg_id = ?", uid, cfgId).First(&it).Error; err != nil {
		h.DB.Create(&model.EzfyItem{UserId: uid, CfgId: cfgId, Count: count})
		return
	}
	// ★ 2026-10-08 原子自增（原 `Update("count", it.Count+count)` 是读-改-写，
	//   并发发放/连点购买时同一行的自增会互相覆盖 → 少发道具）。
	h.DB.Model(&model.EzfyItem{}).Where("id = ?", it.ID).
		Update("count", gorm.Expr("count + ?", count))
}

func (h *EzfyHandler) consumeItem(uid uint, cfgId int, reason ...string) {
	h.consumeItemN(uid, cfgId, 1, reason...)
}

// ezfyBestSpeedItem 背包里某类加速道具（ezfyItemTypeBuildSpeed / TrainSpeed / TechSpeed）中
// **加速时长最短**那一个的 cfg_id（返回 0 = 一个都没有）。
//
// ★ 2026-09-26 加：给「服务端自己挑道具」的场景用（如 /techs/speed），
// 口径必须和前端 accItems()[0] 一致 —— 优先消耗最短的，免得误吃玩家花钱买的 2 小时道具。
func (h *EzfyHandler) ezfyBestSpeedItem(uid uint, itemType int) int {
	var items []model.EzfyItem
	h.DB.Where("user_id = ? AND count > 0", uid).Find(&items)
	best, bestMin := 0, int64(0)
	for _, it := range items {
		cfg := ezfyCfg.item(it.CfgId)
		if cfg == nil || cfg.ItemType != itemType {
			continue
		}
		if best == 0 || cfg.Param1 < bestMin {
			best, bestMin = cfg.ID, cfg.Param1
		}
	}
	return best
}

// consumeItemN 一次扣掉 n 个道具（n <= 0 时什么都不做）。
//
// ★ 批量道具（如经验书一次用几千本）必须走这个 —— 原来循环里逐本调 consumeItem，
// 一次请求就是几千条 SELECT + UPDATE。
// reason：消耗原因（写进「道具使用」流水，管理端可查），不传时默认「道具消耗」。
func (h *EzfyHandler) consumeItemN(uid uint, cfgId, n int, reason ...string) {
	if n <= 0 {
		return
	}
	var it model.EzfyItem
	if err := h.DB.Where("user_id = ? AND cfg_id = ?", uid, cfgId).First(&it).Error; err != nil {
		return
	}
	if n >= it.Count {
		h.DB.Delete(&it)
	} else {
		h.DB.Model(&model.EzfyItem{}).Where("id = ?", it.ID).Update("count", it.Count-n)
	}
	rs := "道具消耗"
	if len(reason) > 0 && reason[0] != "" {
		rs = reason[0]
	}
	h.logItemUse(uid, cfgId, n, rs)
}

// logItemUse 记录道具消耗流水（管理端「数据管理 → 道具使用」查看，用于核实「丢道具」反馈）
func (h *EzfyHandler) logItemUse(uid uint, cfgId, count int, reason string) {
	if count <= 0 {
		return
	}
	itemName, itemType := "", 0
	if cfg := ezfyCfg.item(cfgId); cfg != nil {
		itemName, itemType = cfg.Name, cfg.ItemType
	}
	h.DB.Create(&model.EzfyItemUseLog{
		UserId: uid, CfgId: cfgId, ItemName: itemName, ItemType: itemType,
		Count: count, Reason: reason,
	})
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

// hasAnyPeaceEffect 玩家名下**任意**城市是否有生效中的免战保护令（effect_type=2）。
// ★ 2026-10-02 免战保护令改为**全账号生效**——任一城用了保护令，
//   该玩家所有城的自城派遣都放开携带上限（与单城保护令的「出征拦截」语义区分开）。
// 走索引：ezfy_city_effect 联合唯一索引 (city_id,effect_type) + ezfy_city idx_user。
func (h *EzfyHandler) hasAnyPeaceEffect(uid uint) bool {
	var one int
	err := h.DB.Raw(
		"SELECT EXISTS(SELECT 1 FROM ezfy_city_effect e JOIN ezfy_city c ON c.id = e.city_id "+
			"WHERE c.user_id = ? AND e.effect_type = 2 AND e.until_time > ?) AS x",
		uid, time.Now().UnixMilli()).Scan(&one).Error
	return err == nil && one == 1
}

// useItem 使用道具（支持批量：count 个；军官类道具需指定 officerId/skillId；
// recordId：加速类道具指定目标（建筑升级记录 id / 训练队列 id），0 = 由后端自动挑最早的一条）
// 复刻设计文档《QQ家园二战风云.txt》道具 #7 招生简章 / #8 经验书 / #9 军官技能书·重修书
func (h *EzfyHandler) useItem(uid uint, city *model.EzfyCity, cfgId, count int, officerId int64, skillId int, recordId int64) string {
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
	// ★ 「军官经验道具最大只能用 100 不对，没有上限卡控」→ 经验书取消单次数量上限
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
		maxLv := h.officerMaxLevelOf(o)
		per := cfg.Param1
		if per <= 0 {
			return "道具配置有误(经验为0)"
		}
		// 升到满级还差多少经验（升级需要 等级×200，与 addOfficerExp 同一口径）
		var need int64
		for lv, exp := o.Level, o.Exp; lv < maxLv; lv++ {
			need += int64(lv)*200 - exp
			exp = 0
		}
		if need <= 0 {
			return fmt.Sprintf("%s 已达最高等级%d级, 经验书不消耗", o.Name, maxLv)
		}
		used := int64(count)
		if maxBooks := (need + per - 1) / per; maxBooks < used {
			used = maxBooks
		}
		h.addOfficerExp(city, o.ID, per*used)
		h.consumeItemN(uid, cfgId, int(used), "使用道具")
		msg := fmt.Sprintf("使用成功: %s 获得%d经验", o.Name, per*used)
		if used < int64(count) {
			msg += fmt.Sprintf("（已达%d级上限，本次只消耗%d本，其余%d本留在背包）",
				maxLv, used, int64(count)-used)
		}
		return msg
	}

	var lastMsg string
	for i := 0; i < count; i++ {
		msg := h.useItemOnce(uid, city, cfg, officerId, skillId, recordId)
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

// pctSpeedEnd 百分比加速后的新结束时间：剩余时长直接减 pct%（保留毫秒级精度）。
// remain <= 0（已到点/超时未结算）时不改，原样返回。
func pctSpeedEnd(now, endTime, pct int64) int64 {
	remain := endTime - now
	if remain <= 0 {
		return endTime
	}
	return now + remain*(100-pct)/100
}

// useItemOnce 单个道具生效（内部函数, 由 useItem 调用；
// recordId：加速类道具指定目标（建筑升级记录 id / 训练队列 id），0 = 自动挑最早的一条）
func (h *EzfyHandler) useItemOnce(uid uint, city *model.EzfyCity, cfg *model.EzfyCfgItem, officerId int64, skillId int, recordId int64) string {
	cfgId := cfg.ID
	param := cfg.Param1
	var err string
	switch cfg.ItemType {
	case 1:
		// ★ 2026-09-24 用户反馈「资源包用了资源变少」：原来把 city 内存里的旧值 +param
		//   整行写回( saveCityRes )，会覆盖掉这期间懒结算/运输/掠夺等并发写入的增量。
		//   改成 DB 原子累加 col + param，只基于库里最新值加，不丢任何存量。
		//   ★ 2026-09-24 再修：不能套 LEAST(cap, ...)——道具是凭空发资源，截在上限会
		//   把多出的部分丢掉（玩家反馈「用了资源又变成上限了」）。道具加资源一律无条件累加。
		if err := h.DB.Model(&model.EzfyCity{}).Where("id = ?", city.ID).Updates(map[string]interface{}{
			"food":  ezfyResAddExpr("food", param),
			"steel": ezfyResAddExpr("steel", param),
			"oil":   ezfyResAddExpr("oil", param),
			"rare":  ezfyResAddExpr("rare", param),
		}).Error; err != nil {
			return "资源累加失败: " + err.Error()
		}
		h.consumeItem(uid, cfgId, "使用道具")
		return fmt.Sprintf("使用成功: 粮食/钢铁/石油/稀矿各+%d", param)
	case 2:
		if err := h.DB.Model(&model.EzfyCity{}).Where("id = ?", city.ID).
			Update("gold", ezfyResAddExpr("gold", param)).Error; err != nil {
			return "黄金累加失败: " + err.Error()
		}
		h.consumeItem(uid, cfgId, "使用道具")
		return fmt.Sprintf("使用成功: 黄金+%d", param)
	// ★ 2026-09-28 单资源礼包（2钻礼包2~5，ItemType 27~30）：
	//   与黄金包(ItemType 2)同款实现，各自只加一种资源，无条件累加不截上限。
	case 27: // 粮食包
		if err := h.DB.Model(&model.EzfyCity{}).Where("id = ?", city.ID).
			Update("food", ezfyResAddExpr("food", param)).Error; err != nil {
			return "粮食累加失败: " + err.Error()
		}
		h.consumeItem(uid, cfgId, "使用道具")
		return fmt.Sprintf("使用成功: 粮食+%d", param)
	case 28: // 钢铁包
		if err := h.DB.Model(&model.EzfyCity{}).Where("id = ?", city.ID).
			Update("steel", ezfyResAddExpr("steel", param)).Error; err != nil {
			return "钢铁累加失败: " + err.Error()
		}
		h.consumeItem(uid, cfgId, "使用道具")
		return fmt.Sprintf("使用成功: 钢铁+%d", param)
	case 29: // 石油包
		if err := h.DB.Model(&model.EzfyCity{}).Where("id = ?", city.ID).
			Update("oil", ezfyResAddExpr("oil", param)).Error; err != nil {
			return "石油累加失败: " + err.Error()
		}
		h.consumeItem(uid, cfgId, "使用道具")
		return fmt.Sprintf("使用成功: 石油+%d", param)
	case 30: // 稀矿包
		if err := h.DB.Model(&model.EzfyCity{}).Where("id = ?", city.ID).
			Update("rare", ezfyResAddExpr("rare", param)).Error; err != nil {
			return "稀矿累加失败: " + err.Error()
		}
		h.consumeItem(uid, cfgId, "使用道具")
		return fmt.Sprintf("使用成功: 稀矿+%d", param)
	case 3:
		err = h.speedUpBuilding(city, recordId, param)
		if err != "" {
			return err
		}
		h.consumeItem(uid, cfgId, "使用道具")
		return fmt.Sprintf("使用成功: 当前建筑升级-%d分钟", param)
	case 4:
		err = h.speedUpTrain(city, recordId, param)
		if err != "" {
			return err
		}
		h.consumeItem(uid, cfgId, "使用道具")
		return fmt.Sprintf("使用成功: 当前训练队列-%d分钟", param)
	case 5:
		err = h.speedUpTech(city, param)
		if err != "" {
			return err
		}
		h.consumeItem(uid, cfgId, "使用道具")
		return fmt.Sprintf("使用成功: 当前科技研究-%d分钟", param)
	// ★ 2026-09-27 百分比加速道具（ItemType 24/25/26，Param1 = 30/60/80）：
	//   按「剩余时间」直接减 param%，每次使用都基于最新剩余时长（可叠加）。
	case 24: // 建筑加速%
		var b model.EzfyCityBuilding
		if err := h.DB.Where("city_id = ? AND status != 0", city.ID).
			Order("end_time ASC").First(&b).Error; err != nil {
			return "没有正在施工的建筑"
		}
		if recordId > 0 {
			// ★ 2026-09-27 修复「没有按指定目标扣减」：点哪条建筑就减哪条
			var target model.EzfyCityBuilding
			if err := h.DB.Where("id = ? AND city_id = ?", recordId, city.ID).First(&target).Error; err != nil {
				return "没有正在施工的建筑"
			}
			if target.Status == 0 {
				return "建筑没有在施工"
			}
			b = target
		}
		h.DB.Model(&model.EzfyCityBuilding{}).Where("id = ?", b.ID).
			Update("end_time", pctSpeedEnd(time.Now().UnixMilli(), b.EndTime, param))
		h.consumeItem(uid, cfgId, "使用道具")
		return fmt.Sprintf("使用成功: 当前建筑升级剩余时间减少%d%%", param)
	case 25: // 训练加速%
		var q model.EzfyTrainQueue
		if err := h.DB.Where("city_id = ? AND status = 0", city.ID).
			Order("start_time ASC").First(&q).Error; err != nil {
			return "没有训练中的队列"
		}
		if recordId > 0 {
			// ★ 2026-09-27 修复「没有按指定目标扣减」：点哪条队列就减哪条
			var target model.EzfyTrainQueue
			if err := h.DB.Where("id = ? AND city_id = ?", recordId, city.ID).First(&target).Error; err != nil {
				return "没有训练中的队列"
			}
			q = target
		}
		h.DB.Model(&model.EzfyTrainQueue{}).Where("id = ?", q.ID).
			Update("end_time", pctSpeedEnd(time.Now().UnixMilli(), q.EndTime, param))
		h.consumeItem(uid, cfgId, "使用道具")
		return fmt.Sprintf("使用成功: 当前训练队列剩余时间减少%d%%", param)
	case 26: // 科技加速%
		var t model.EzfyCityTech
		// ★ 2026-09-28 研究队列记录在「发起城市」
		if err := h.DB.Where("city_id = ? AND status = 1", city.ID).
			First(&t).Error; err != nil {
			return "没有研究中的科技"
		}
		h.DB.Model(&model.EzfyCityTech{}).Where("id = ?", t.ID).
			Update("end_time", pctSpeedEnd(time.Now().UnixMilli(), t.EndTime, param))
		h.consumeItem(uid, cfgId, "使用道具")
		return fmt.Sprintf("使用成功: 当前科技研究剩余时间减少%d%%", param)
	case 6:
		return "建筑图纸将在建筑升级到10级时自动消耗"
	case 7:
		h.addCityEffect(city.ID, 1, int(param), 24)
		h.consumeItem(uid, cfgId, "使用道具")
		return fmt.Sprintf("使用成功: 资源产量+%d%%, 持续24小时", param)
	case 8:
		// ★ 2026-10-02 免战保护令(24小时) 也要有 24 小时冷却
		now := time.Now().UnixMilli()
		var prof model.EzfyProfile
		if err := h.DB.Select("peace_cool_until").First(&prof, uid).Error; err == nil && prof.PeaceCoolUntil > now {
			remainH := (prof.PeaceCoolUntil - now + 3599999) / 3600000
			return fmt.Sprintf("免战保护令冷却中, 剩余%d小时", remainH)
		}
		h.addCityEffect(city.ID, 2, 0, param)
		h.consumeItem(uid, cfgId, "使用道具")
		h.DB.Model(&model.EzfyProfile{}).Where("id = ?", uid).Update("peace_cool_until", now+24*3600000)
		return fmt.Sprintf("使用成功: 城市免战保护%d小时(冷却24小时)", param)
	case 9: // 招生简章: 立即刷新军校候选(不占每日次数)
		if h.buildingLevel(city.ID, ezfyBuildingAcademy) < 1 {
			return "需要先建造军校"
		}
		if err := h.refreshRecruitFree(uid); err != "" {
			return err
		}
		h.consumeItem(uid, cfgId, "使用道具")
		return "使用成功: 军校候选名将已刷新"
	case 10: // 经验书
		o := h.officerOf(city.ID, officerId)
		if o == nil {
			return "军官不存在"
		}
		h.addOfficerExp(city, o.ID, param)
		h.consumeItem(uid, cfgId, "使用道具")
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
		h.consumeItem(uid, cfgId, "使用道具")
		return fmt.Sprintf("使用成功: %s 学会了「%s」", o.Name, sk.Name)
	case 12: // 军官洗点卡（重修书）
		// ★ 用户规则（原话）：「洗点就是洗点成原来军官池子武将的属性，等级不变；
		//   待分配的属性 = 现在属性之和 − 军官池属性之和，玩家可以重新分配加点」。
		//
		//   重置目标走 officerBaseAttr（原始属性 = 军官池武将属性，升星不改它），
		//   于是已分配的点 + 升星加成一起退回成可用属性点。
		//   旧实现是「随机重新分配」，余数 `newLea = total - 军事 - 后勤` **全给学识**，
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
		// ★ 用户规则：「洗点只是属性，跟其他没关系」——
		//   只重置三维属性 + 退回待分配点，**技能/等级/经验/忠诚/装备一概不动**。
		h.DB.Model(&model.EzfyOfficer{}).Where("id = ?", o.ID).Updates(map[string]interface{}{
			"military": bm, "logistics": bl, "learning": be,
			// base_* 一起回到洗点目标，之后「已分配点数」的显示才对得上
			"base_military": bm, "base_logistics": bl, "base_learning": be,
			"free_points": free, "update_time": time.Now(),
		})
		h.consumeItem(uid, cfgId, "使用道具")
		starPts := 0
		if o.StarPoints > 0 {
			starPts = o.StarPoints
		}
		return fmt.Sprintf("使用成功: %s 洗点完成\n军事 %d→%d  后勤 %d→%d  学识 %d→%d\n"+
			"待分配属性点 +%d（共 %d 点，去军官详情分配）\n（属性已重置为军官池初始属性；等级/经验/技能保留）"+
			"\n其中升星加点 %d 点（含在待分配点内）",
			o.Name, o.Military, bm, o.Logistics, bl, o.Learning, be, refund, free, starPts)
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
		h.consumeItem(uid, cfgId, "使用道具")
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

var ezfyStateTaskTypes = map[string]bool{"city_level": true, "army_count": true, "wild_count": true, "has_city": true}

// ezfyTaskRewardRes 返回某任务结算用的「资源」奖励（粮/钢/油/稀）。
//
// ★ 2026-09-29 资源奖励按任务类型加成：
//     新手任务(type_id=1)：四种生产资源 ×1000；
//     日常任务(type_id=2)/ 每周任务(type_id=4)：四种生产资源 ×10。
//   黄金/声望不改。任务列表展示与发奖都用同一口径，保证玩家看到多少、领到多少一致。
//   注意这里只做展示/发奖加成，不改表里存的原始数值（管理端「数据管理」仍存基数）。
func ezfyTaskRewardRes(cfg *model.EzfyCfgTask) (int64, int64, int64, int64) {
	food, steel, oil, rare := cfg.RewardFood, cfg.RewardSteel, cfg.RewardOil, cfg.RewardRare
	switch cfg.TypeId {
	case 1: // 新手任务
		food *= 1000
		steel *= 1000
		oil *= 1000
		rare *= 1000
	case 2, 4: // 日常任务、每周任务
		food *= 10
		steel *= 10
		oil *= 10
		rare *= 10
	}
	return food, steel, oil, rare
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
	case "has_city":
		// ★ 2026-09-30 新手任务「首个城池」：只要拥有首城即满足（目标 1）。
		//   getOrCreateCity 保证玩家必有城池，故恒为 1。
		return 1
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
	// ★ 2026-09-30 「首个城池助力」补领时发到**最早(主)城池**：
	//   老玩家当前城可能早已不是首城，奖励必须进 id 最小的主城，而不是当前操作城。
	//   仅对 has_city（首个城池）任务生效；其它任务维持原逻辑（发到当前城）。
	if cfg.TaskType == "has_city" {
		city = h.mainCity(uid)
	}
	// ★ 任务奖励**不受仓储上限截断**（）。
	//   原来走 min64(cap, ...)，仓储满了领奖就等于白发；只有「城市自身产量」才该被上限卡住。
	//   ★ 新手任务资源 ×1000（见 ezfyTaskRewardRes）
	rf, rs, ro, rr := ezfyTaskRewardRes(&cfg)
	h.giveResNoCap(&city, rf, rs, ro, rr, cfg.RewardGold)
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

	// ★ 2026-10-03 性能（第三批）：/view 是首页 30s 轮询 + 进入游戏即时加载的目标接口。
	//   本批目标：把单次 RDS 往返数从 ~15+ 压到 ~4 次。
	//   1) 城市列表提前同步取（原 getOrCreateCity 内部会重复查 profile + 单查 city）；
	//   2) 懒结算改为复用本请求已查好的 buildings/城市ID/trainQueues，不再重复打库；
	//   3) itemCount/mayor/增产令并入并行块，getResourceCalcWith 变纯内存；
	//   4) 3 秒短 TTL 玩家级缓存兜底 —— 轮询/加载的绝大多数请求直接命中，不进结算。
	//      （cache 未命中才跑懒结算；游戏 tick 仍由其它接口照常推进。）
	if it, ok := ezfyViewCacheGet(uid); ok {
		resp.OK(c, it)
		return
	}
	// ★ 2026-10-04 数据构建抽成 viewPayload：切城等「需要即时刷新新城数据」的接口直接复用，
	//   不再「切完再 GET /view」打第二遍（那会再跑一次完整懒结算 + 十几条查询）。
	data := h.viewPayload(uid)
	ezfyViewCacheSet(uid, data)
	resp.OK(c, data)
}

// viewPayload 构建 /view 完整数据（纯构建不做缓存；调用方自行决定缓存/下发）。
func (h *EzfyHandler) viewPayload(uid uint) gin.H {
	h.cfgs()
	// ★ 2026-10-04 临时耗时日志：排查线上 /view 卡顿（确认后端计算 vs 传输层瓶颈），定位后移除
	_vpStart := time.Now()
	defer func() { log.Printf("ezfy viewPayload %dms uid=%d", time.Since(_vpStart).Milliseconds(), uid) }()
	// ★ 2026-10-04 性能（用户反馈「/view 线上 3s」）：档案 + 城市列表是两条独立查询，
	//   原来串行（2 个 RTT），拿到当前城后又串行查军官（2 个 RTT）——
	//   跨 WAN 慢 RDS 下 /view 缓存未命中时偏慢。改为两波并行：
	//   第一波 profile + cities（1 个 RTT）→ 定出当前城；
	//   第二波 officerList + 其余 16 条只读查询（1 个 RTT）。
	//   串行 RTT 从 4 降到 2，加上进程内缓存兜底，绝大多数请求零 SQL。
	var profile model.EzfyProfile
	var cities []model.EzfyCity
	var wg0 sync.WaitGroup
	wg0.Add(2)
	go func() { defer wg0.Done(); profile = h.ensureProfile(uid) }()
	go func() { defer wg0.Done(); h.DB.Where("user_id = ?", uid).Order("id ASC").Find(&cities) }()
	wg0.Wait()
	// 城市列表提前同步取：供城市挑选 + 懒结算 checkTechDone(多城研究) 复用，
	// 省掉 currentCity 里「重复查 profile + 单查 city」的两条串行 SQL。
	city := h.ezfyViewCurrentCity(profile, cities)

	// ★ 军官列表在本请求内只读一次，供懒结算扣工资 + 下面的展示复用。
	//   本接口是首页 30s 轮询的目标，重复查军官表曾是线上 IO 飙升的主因。
	//   已并入下方第二波并行块（wg.Add(17) 里第一条）。

	// ★ 2026-10-03 性能：以下只读查询互相独立、且只依赖 uid/city.ID，用 sync.WaitGroup
	//   并行打 RDS，把 30s 轮询 /view 的串行往返(~1s+) 压到接近一次往返量。
	//   gorm v2 链式调用并发安全；wg.Wait() 提供 happens-before，无数据竞争。
	var (
		buildings []model.EzfyCityBuilding
		troops    map[int]int64
		tmap      map[int]int
		wildlands []model.EzfyWildland
		officers  []model.EzfyOfficer

		trainQueues   []model.EzfyTrainQueue // 训练队列原始行：懒结算 collectTrainQueue 复用
		techRows      []model.EzfyCityTech   // 进行中科技：懒结算 checkTechDoneRows 复用（★ 2026-10-04 并入第二波省 1 条串行 RTT）
		marching      int64
		occupying     int64
		unreadReports int64
		popUsed       int64
		protected     bool
		boost         bool
		boostEff      *model.EzfyCityEffect
		mayorPct      int
		gatherHave    int
		acct          string
		ulv, uexp     int
	)

	// ★ 2026-10-03 性能：以下只读查询互相独立、且只依赖 uid/city.ID，用 sync.WaitGroup
	//   并行打 RDS，把 30s 轮询 /view 的串行往返(~1s+) 压到接近一次往返量。
	//   gorm v2 链式调用并发安全；wg.Wait() 提供 happens-before，无数据竞争。
	// ★ 2026-10-04 瘦身：建筑/军队/科技/野地/队列/伤兵**不再随 /view 下发**（移到对应页面
	//   进入时各自接口加载），但原数据仍要查回来供懒结算与首页产量(res_prod)计算复用。
	var wg sync.WaitGroup
	wg.Add(14)
	go func() { // 军官列表（含出征态自愈 + 一次 orderListByCity 自愈判定）
		defer wg.Done()
		officers = h.officerList(city.ID)
	}()
	go func() { // 进行中科技（懒结算 checkTechDoneRows 复用；写部分在并行块后串行跑）
		defer wg.Done()
		h.DB.Where("city_id IN ? AND status = 1", cityIdsOf(cities)).Find(&techRows)
	}()
	go func() { // 建筑列表（供 getResourceCalcWith + checkBuildingDone 复用）
		defer wg.Done()
		buildings = h.buildingList(city.ID)
	}()
	go func() { // 军队表（getResourceCalcWith 复用）
		defer wg.Done()
		troops = h.troopMap(city.ID)
	}()
	go func() { // 科技表（★ 2026-10-05 用 techMapOf(uid)，省掉 techMap 内部的 SELECT user_id 往返）
		defer wg.Done()
		tmap = h.techMapOf(uid)
	}()
	go func() { // 野地列表
		defer wg.Done()
		var w []model.EzfyWildland
		h.DB.Where("city_id = ?", city.ID).Find(&w)
		wildlands = w
	}()
	go func() { // 训练队列原始行（懒结算 collectTrainQueue 复用；列表展示由 /troops 提供）
		defer wg.Done()
		var qs []model.EzfyTrainQueue
		h.DB.Where("city_id = ? AND status = 0", city.ID).Order("start_time ASC").Find(&qs)
		trainQueues = qs
	}()
	go func() { // 出征中 / 占领中数量
		defer wg.Done()
		var m, o int64
		h.DB.Model(&model.EzfyOrder{}).Where("user_id = ? AND status = 0", uid).Count(&m)
		h.DB.Model(&model.EzfyOrder{}).Where("user_id = ? AND status = 1", uid).Count(&o)
		marching, occupying = m, o
	}()
	go func() { // 未读战报
		defer wg.Done()
		var c int64
		h.DB.Model(&model.EzfyReport{}).Where("user_id = ? AND is_read = 0", uid).Count(&c)
		unreadReports = c
	}()
	// ★ 2026-10-05 性能：已占用人口不再单独查库 —— wg.Wait() 之后用本请求已取到的
	//   buildings + trainQueues 纯内存算（原来 buildingPop/troopPop 各自再查一次）。
	go func() { // 免战保护令（effect_type=2）
		defer wg.Done()
		protected = h.hasCityEffect(city.ID, 2)
	}()
	go func() { // 加速效果（effect_type=1）：展示布尔 + 资源结算复用（过期顺手删）
		defer wg.Done()
		var e model.EzfyCityEffect
		if err := h.DB.Where("city_id = ? AND effect_type = 1", city.ID).First(&e).Error; err == nil {
			if e.UntilTime > time.Now().UnixMilli() {
				boostEff = &e
				boost = true
			} else {
				h.DB.Delete(&e)
			}
		}
	}()
	go func() { // 市长后勤加成 %（getResourceCalcWith 复用，不再重复查库）
		defer wg.Done()
		mayorPct = h.mayorBonusPct(city.ID)
	}()
	go func() { // 集结令背包持有量
		defer wg.Done()
		gatherHave = h.itemCount(uid, ezfyGatherItemID)
	}()
	go func() { // 家园账号 / 等级 / 经验
		defer wg.Done()
		a, l, e := h.ezfyUserBrief(uid)
		acct, ulv, uexp = a, l, e
	}()
	wg.Wait()

	// ★ 2026-10-03 性能：懒结算改为「复用并行块已查好的数据」——
	//   原 refreshCityWithOfficers 在并行块之前跑，内部对 buildings/城市ID/trainQueues
	//   各自再查一遍（每请求白白多打 ~4 条 RDS）。这里传入已查数据，零重复查询。
	//   顺序保持原样：calcResource 在 processOrders 之前（processOrders 可能向当前城市写资源，
	//   必须最后写，避免覆盖返航/战斗入库的资源）。
	//   calcResource 未到结算点(每小时)时为 0 查询，processOrders 保持串行链尾。
	// ★ 2026-10-05 性能：已占用人口用已取数据纯内存算（零 SQL）。
	popUsed = h.cityPopUsedD(city.ID, buildings, trainQueues)

	// ★★ 2026-10-05 性能（用户反馈「这几个接口还是 2s+」）：
	//   把本请求并行块已取到的「城市只读数据」打包成快照，交给懒结算与资源详情复用 ——
	//   calcResource 由「8~9 条串行 RDS」变成「0 读 + 1 写」，getResourceCalcWith 也零额外查询。
	//   这两个消费方原来会把 buildingList / techMap / wildlandList / troopMap / 增产令 /
	//   市长加成 / 节日活动**全部再查一遍**（跨 WAN 每个请求白打 ~8 个往返）。
	snap := &resCalcData{
		buildings: buildings, techs: tmap, wilds: wildlands, troops: troops,
		boost: boostEff, boostDone: true, mayor: mayorPct,
	}
	h.checkBuildingDone(&city, buildings)
	// ★ 2026-10-04 科技行已在第二波并行取好，只跑写部分（省 1 条串行 RTT）
	h.checkTechDoneRows(&city, techRows)
	h.collectTrainQueue(&city, trainQueues)
	h.calcResourceD(&city, snap, officers)
	// ★ 2026-10-04 传入已查好的城市列表：processIncoming 直接复用，省一次 Pluck 的 RTT
	cids := make([]int64, 0, len(cities))
	for _, c := range cities {
		cids = append(cids, int64(c.ID))
	}
	h.processOrders(uid, cids)

	// ★ 2026-10-04 瘦身：建筑/可建造池/军队/科技/野地/训练队列/伤兵列表**不再由 /view 下发**，
	//   对应页面进入时各自调 /buildings、/troops、/techs、/city/wildfull 懒加载（前端 go() 已接线）。
	//   上面并行块仍把原始数据查回来，供懒结算与 res_prod 计算使用（零重复查询）。

	// ★ 2026-09-28 首页头部资源栏「/」右侧展示**每小时产量**（与资源详情页同一口径）。
	//   复用 getResourceCalc 的 total（净产量：产出 − 军队耗粮），保证两边数字永远一致。
	resProd := gin.H{}
	// ★ 2026-10-03 性能：getResourceCalc 内部会查 buildingList/techMap/wildlandList/boost/troopMap，
	//   在 30s 轮询的 /view 上之前每次循环都重算 5 遍（5×6 条 SQL 跨 WAN 往返）。
	//   改成只算一次，再从这里按资源键取值。
	//   ★ 2026-10-03 第二批：把这请求已查好的 buildings/tmap/wildlands/troops 传入，
	//   让 getResourceCalc 变成纯内存，只剩 mayor/boost 两条小查询。
	//   ★ 2026-10-03 第三批：mayor/增产令也并入并行块查好传入，getResourceCalcWith 完全零 SQL。
	resCalc := h.getResourceCalcWith(&city, snap)
	for _, k := range []string{"gold", "food", "steel", "oil", "rare"} {
		if it, ok := resCalc[k].(gin.H); ok {
			resProd[k] = it["total"]
		}
	}
	// ★ 军事区/资源区上限（线上现值各 36，管理端可维护）：随 /view 下发，前端不再硬编码
	lim := ezfyLimit()
	data := gin.H{
		"profile":    profile,
		"account":    acct,
		"user_level": ulv,
		"user_exp":   uexp,
		// ★ 在职军官数：直接用本请求已取到的 officers 统计，
		//   不要再调 h.officerCount（它内部又查一次 officerList）。
		"officer_count": officerCountOf(officers),
		"rank_name":     ezfyRankNameAt(ezfyProfileRank(&profile)),
		"rank_post":     ezfyRankPostAt(ezfyProfileRank(&profile)),
		"cities":        h.cityViews(cities),
		"diamond":       profile.Diamond,
		// ★ 2026-10-07 赎城金额（管理端可配）：前端城市列表 [赎城] 确认弹窗展示
		"ransom_cost": ezfyRansomCost(),
		"city":          city,
		"res_prod":      resProd, // ★ 2026-09-28 各资源每小时净产量(与资源详情页同口径), 头部资源栏「/」右侧展示
		// ★★ 2026-10-05 修复「玩家反馈离线资源不涨」的**显示口径**问题：
		//   产量真正的收敛点是全局「资源最大值」ezfy_cfg_limit.res_max_*（线上 61 亿），
		//   而 city.xxx_cap 只是**仓储展示值**，按设计「永不参与计算」（见 calcResourceD 注释）。
		//   前端原来只拿到仓储上限，显示成「42亿/55亿」→ 玩家以为还能涨，
		//   实际早被 61 亿的全局上限卡住不动 → 报「资源不涨」。
		//   这里把真上限下发给前端，前端据此显示「已满」。
		"res_max": gin.H{
			"gold":  ezfyResMaxOf("gold"),
			"food":  ezfyResMaxOf("food"),
			"steel": ezfyResMaxOf("steel"),
			"oil":   ezfyResMaxOf("oil"),
			"rare":  ezfyResMaxOf("rare"),
		},
		// ★ 2026-09-28 安抚参数(管理端可配)：前端安抚页直接展示，不再硬编码「民怨×100」。
		//   cd_left = 距离下次可安抚的剩余毫秒(0 = 现在就能安抚)。
		"placate": gin.H{
			"gold":          ezfyPlacateGoldCost(),
			"grievance":     ezfyPlacateGrievanceDown(),
			"feelings":      ezfyPlacateFeelingsUp(),
			"cooldown_min":  ezfyPlacateCooldownMinutes(),
			"cd_left":       max64(0, city.PlacateTime+ezfyPlacateCooldownMs()-time.Now().UnixMilli()),
		},
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
		"protected": protected,
		"boost":     boost,
		// ★ 军事区/资源区各自上限（分开下发；建筑列表/可建造池已移到 /buildings）
		"military_cap": lim.MilitaryMax,
		"resource_cap": lim.ResourceMax,
		"marching":     marching,
		"occupying":    occupying,
		// ★ 占用人口 = 建筑占用人口 + 训练中未出厂的新兵占用（部队不占人口位置）。
		//   否则没进过「军队」页时 troopsData 还是空的 → 空闲人口会显示成满人口（用户反馈的 bug）
		"pop_used":       popUsed,
		"unread_reports": unreadReports,
		// ★ 资源显示名（管理端可改名，前端一律读这里，不要再写死「粮食/钢铁/…」）
		"res_names": ezfyResCfgOf(h.DB),
		// ★ 集结令配置：跟着 /view 一起下发，前端一进页面就是准确值。
		//   原来只有 /order/preview 才返回 gather_max，前端在「还没点[计算]」时
		//   兜底写死 50 → 管理端配了 999 也只能填 50（用户反馈的 bug）。
		"gather_max":  ezfyGatherMax(),
		"gather_per":  ezfyGatherBonusPer(),
		"gather_have": gatherHave,
		// ★ 军官工资（黄金/小时）：军官页直接展示，让玩家看得见钱花在哪
		//   用上面已取到的 officers 做纯内存计算（勿改回 officerSalaryPerHour）
		"officer_salary": officerSalaryOf(officers),
		// ★ 2026-09-26：民居容量限制 / 召集人口灵活配置两个开关，前端「召集人口」页按它提示与禁用按钮
		"house_pop_limit_on":  ezfyHousePopLimitOn(),
		"convene_flexible_on": ezfyConveneFlexOn(),
		// ★ 2026-09-26：召集消耗粮食 / 获得人口（前端文案与按钮禁用都要用，勿再写死 10 万）
		"convene_food_cost": ezfyConveneFoodCostCfg(),
		"convene_pop_gain":  ezfyConvenePopGainCfg(),
		// ★ 2026-09-26：召集硬性人口上限（0 = 不限），前端提示与按钮禁用都要用
		"convene_pop_max": ezfyConvenePopMaxCfg(),
	}
	// ★ 2026-10-03 第三批：3 秒短 TTL 玩家级缓存。写操作后的刷新最多滞后 3 秒（可接受）。
	//   这里只负责「构建」，缓存/下发由 View（或切城等接口）决定。
	return data
}

// ★ 2026-10-03 /view 玩家级短 TTL 缓存。
//
// ★★ 2026-10-05 目前两台机器、以后可能多台，进程内缓存会互相 miss、且每台机器的
//   写操作只清了自己那台的缓存 → 跨机数据不一致。**三套缓存全部改为空操作**（Get 永远 miss、
//   Set/Del 直通，保留签名让所有调用点照常编译），展示接口直查 DB，靠 SQL/索引/两波并行/
//   页面懒加载把延迟压在目标内。TTL 常量仅供其它进程内 memo（行军结算快速路径）复用。
const ezfyViewCacheTTLMs = 3000

func ezfyViewCacheGet(uid uint) (gin.H, bool) { return nil, false }

func ezfyViewCacheSet(uid uint, data gin.H) {}

// ezfyViewCacheDel 缓存已禁用（空操作）。
func ezfyViewCacheDel(uid uint) {}

// ★ 2026-10-04 二战 5 个展示接口（军官 /officers、军校 /acade/recruit、技能 /officers/skills、
//   装备 /officers/equipments、任务 /tasks）的 3s TTL 玩家级缓存 —— ★ 2026-10-05 已禁用（空操作，
//   见上注释：多机缓存不一致），调用点保留以兼容编译，写操作后的 Del 也无副作用。

// ★ 2026-10-05 训练一键加速 5 秒卡控（前后端都卡，防连点/脚本反复刷黄金结算）
var (
	ezfySpeedTrainMu   sync.Mutex
	ezfySpeedTrainMemo = map[uint]int64{}
)

// ★ 2026-10-05 人口召集卡控（与训练一键加速一致）
// ★ 2026-10-09 用户要求 5 秒 → **3 秒**（前端 doConvene 同步改）
var (
	ezfyConveneMu   sync.Mutex
	ezfyConveneMemo = map[uint]int64{}
)

func ezfyPageCacheGet(uid uint, name string) (gin.H, bool) { return nil, false }

func ezfyPageCacheSet(uid uint, name string, data gin.H) {}

// ezfyPageCacheDel 缓存已禁用（空操作）。
func ezfyPageCacheDel(uid uint) {}

// ★ 2026-10-04 /chest 3s TTL 玩家级缓存 —— ★ 2026-10-05 已禁用（空操作，见上方多机不一致注释）。
func ezfyChestCacheGet(uid uint) (gin.H, bool) { return nil, false }

func ezfyChestCacheSet(uid uint, data gin.H) {}

func ezfyChestCacheDel(uid uint) {}

// ezfyViewCurrentCity 从本请求已取到的玩家城市列表里挑当前城市（与 currentCity 同口径）：
// 优先 profile.current_city_id（须属于该玩家），其次最小 id 城市，都没有则建主城（首登一次性）。
func (h *EzfyHandler) ezfyViewCurrentCity(p model.EzfyProfile, cities []model.EzfyCity) model.EzfyCity {
	if p.CurrentCityId > 0 {
		for _, c := range cities {
			if int64(c.ID) == p.CurrentCityId {
				return c
			}
		}
	}
	if len(cities) > 0 {
		return cities[0]
	}
	return h.createMainCity(p.UserID)
}

// ezfyPageCity 展示接口统一取「档案 + 当前城 + 城市列表」：两波并行（第一波 1 个 RTT）。
//
// ★ 2026-10-04 性能（用户反馈军官/装备/野地等接口 cache 未命中仍 3s+）：
//   原来 getOrCreateCity 串行查 profile → city（2 RTT），再跑 refreshCityRead 的
//   建筑/科技/队列（3~4 RTT）…… miss 路径十几条串行 RTT。这里第一波把
//   profile + 城市列表**并行**取出（1 RTT），懒结算数据由调用方第二波并行补齐。
func (h *EzfyHandler) ezfyPageCity(uid uint) (model.EzfyProfile, model.EzfyCity, []model.EzfyCity) {
	var profile model.EzfyProfile
	var cities []model.EzfyCity
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); profile = h.ensureProfile(uid) }()
	go func() { defer wg.Done(); h.DB.Where("user_id = ?", uid).Order("id ASC").Find(&cities) }()
	wg.Wait()
	return profile, h.ezfyViewCurrentCity(profile, cities), cities
}

// cityIdsOf 取城市 id 列表（懒结算 checkTechDone 复用，避免重复 ezfyCityIds 查询）。
func cityIdsOf(cities []model.EzfyCity) []uint {
	ids := make([]uint, 0, len(cities))
	for _, c := range cities {
		ids = append(ids, c.ID)
	}
	return ids
}

// ezfyResCfgOf 读取资源显示名配置（管理端可在「资源管理 → 资源名称维护」改）
// 返回 {"gold":"黄金","food":"粮食",...,"_short":{"gold":"金",...}}
// 表为空时回落到内置默认值，保证前端永远拿得到名字。
func ezfyResCfgOf(db *gorm.DB) gin.H {
	// ★ 从内存配置缓存读（30s 周期收敛 / 管理端改完 cfgsReload 即时），请求零 SQL。
	//   表为空时 buildResCfgView 已回落内置默认，前端永远拿得到名字。
	c := &ezfyCfg
	c.load(db) // 幂等：已加载直接返回，未加载先读库兜底
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.resCfg == nil {
		return buildResCfgView(nil)
	}
	return c.resCfg
}

// buildResCfgView 把 ezfy_cfg_resource 行组装成 {key:name, "_short":{key:short}}。
// rows 为空时回落到内置默认名字（黄金/粮食/…），保证前端永远拿得到名字。
func buildResCfgView(rows []model.EzfyCfgResource) gin.H {
	def := []model.EzfyCfgResource{
		{Key: "gold", Name: "黄金", Short: "金"},
		{Key: "food", Name: "粮食", Short: "粮"},
		{Key: "steel", Name: "钢铁", Short: "钢"},
		{Key: "oil", Name: "石油", Short: "油"},
		{Key: "rare", Name: "稀矿", Short: "稀"},
	}
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
	// ★ 2026-10-07 赎城：被占领(occupy status=1) / 有待处理赎城请求 —— 前端据此切换按钮
	Occupied  bool `json:"occupied"`
	Ransoming bool `json:"ransoming"`
}

func (h *EzfyHandler) cityViews(list []model.EzfyCity) []ezfyCityView {
	// ★ 2026-10-07 赎城：一次 IN 查询整批城市的被占/待赎状态，避免每城一条 SQL
	occupied := map[int64]bool{}
	ransoming := map[int64]bool{}
	if len(list) > 0 {
		ids := make([]int64, 0, len(list))
		for _, ct := range list {
			ids = append(ids, int64(ct.ID))
		}
		var occIds []int64
		h.DB.Model(&model.EzfyOccupy{}).Where("status = 1 AND city_id IN ?", ids).Pluck("city_id", &occIds)
		for _, id := range occIds {
			occupied[id] = true
		}
		var ranIds []int64
		h.DB.Model(&model.EzfyRansom{}).Where("status = 0 AND city_id IN ?", ids).Pluck("city_id", &ranIds)
		for _, id := range ranIds {
			ransoming[id] = true
		}
	}
	out := make([]ezfyCityView, 0, len(list))
	for i := range list {
		out = append(out, ezfyCityView{
			EzfyCity:  list[i],
			IsSea:     h.isSeaCity(&list[i]),
			Kind:      h.cityKind(&list[i]),
			Continent: ezfyRegionName(list[i].X, list[i].Y),
			Occupied:  occupied[int64(list[i].ID)],
			Ransoming: ransoming[int64(list[i].ID)],
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
	// ★ 2026-10-04 切城性能（用户反馈「切城卡 + 资源栏延迟 3s」）：
	//   ① 3s TTL 的 /view 缓存里是旧城数据，必须清掉，否则切完 3 秒内 /view 仍返回旧城；
	//   ② 直接返回新城完整 view 数据 —— 前端一次请求完成「切城 + 全量刷新」，
	//      不再 POST 后再 GET /view（省一次 round-trip 和一次重复懒结算）。
	ezfyViewCacheDel(uid)
	// ★ 2026-10-04 切城后军队/附属野地/装备商城等按城缓存也一并清掉（避免 3 秒内显示旧城数据）
	ezfyPageCacheDel(uid)
	data := h.viewPayload(uid)
	data["msg"] = "已切换到「" + ct.Name + "」"
	data["city_id"] = ct.ID
	resp.OK(c, data)
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
	// ★★ 2026-10-09 用户规则：「坐标必须是自己的附属野地（平原、沿海平原）」——
	//   起新城要落在自己**已占领**的野地上（`ezfy_wildland.city_id ∈ 我的城`），
	//   不能随便找块空地就建（否则整张地图的平原都能被抢建）。
	var wl model.EzfyWildland
	if err := h.DB.Where("x = ? AND y = ?", req.X, req.Y).First(&wl).Error; err != nil {
		resp.ParamError(c, "只能在自己占领的野地上建新城（请先征服该野地）")
		return
	}
	if !h.isOwnCity(uid, int64(wl.CityId)) {
		resp.ParamError(c, "该野地不属于你，只能在自己占领的野地上建新城")
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
	// ★ 同时防「连点两下[建新城]」：配额判断 + 建城整段拿本玩家的建城锁，
	//   否则两次请求会同时看到「还没到上限」各建一座（与首次进游戏重复建城同一根因）。
	mu := ezfyCityInitLock(uid)
	mu.Lock()
	defer mu.Unlock()
	prof := h.ensureProfile(uid)
	var owned int64
	h.DB.Model(&model.EzfyCity{}).Where("user_id = ?", uid).Count(&owned)
	profLv := ezfyProfileRank(&prof)
	maxCity := ezfyRankCityMaxAt(profLv)
	if int(owned) >= maxCity {
		resp.ParamError(c, fmt.Sprintf("当前军衔「%s」最多只能拥有 %d 座城市（已有 %d 座），提升声望可解锁更多",
			ezfyRankNameAt(profLv), maxCity, owned))
		return
	}

	// 扣费走「当前操作的城市」（★ 用户规则 2026-09-28：起新城不再消耗黄金，
	// 改为所有资源（粮食/钢铁/石油/稀矿/黄金）各 5 万，扣的是当前城市资源，任一不足都无法起新城）
	cur := h.getOrCreateCity(uid)
	cost := int64(ezfyNewCityResCost)
	if cur.Gold < cost || cur.Food < cost || cur.Steel < cost || cur.Oil < cost || cur.Rare < cost {
		resp.ParamError(c, fmt.Sprintf("建造新城需要粮食/钢铁/石油/稀矿/黄金各%d", cost))
		return
	}
	cur.Gold -= cost
	cur.Food -= cost
	cur.Steel -= cost
	cur.Oil -= cost
	cur.Rare -= cost
	h.saveCityRes(&cur)
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
		fmt.Sprintf("花费粮食/钢铁/石油/稀矿/黄金各%d在%s(%d,%d)建造了新城[%s]\n新城自带基础建筑: %s(1级), 可到[城市列表]切换操作。%s",
			cost, kind, req.X, req.Y, city.Name, ezfyBaseBuildingNames(h.DB), extra))
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
	// ★ 2026-10-04 与切城一致：清 /view 缓存，避免 3s TTL 内 cities 列表还带着已摧毁的城
	ezfyViewCacheDel(uid)
	resp.OK(c, gin.H{"msg": "城市「" + ct.Name + "」已摧毁，该坐标恢复为普通平原"})
}

// ezfyDestroyCity 真正拆除一座城：清掉它的建筑/部队/科技/军官/野地/队列等，
// 并抹掉该坐标的「玩家城」地图区域记录（于是变回普通平原，不属于任何玩家）。
func (h *EzfyHandler) ezfyDestroyCity(uid uint, ct *model.EzfyCity) string {
	cid := ct.ID
	// ★ 2026-09-28 科技等级存用户级(ezfy_user_tech)，拆城无需搬家、等级不随城消失。
	// 还在外面的部队/采集队：一并撤掉（否则会留下指向已删城市的孤儿订单）
	h.DB.Where("city_id = ?", cid).Delete(&model.EzfyOrder{})
	// ★ 2026-10-02 盟军驻军：驻守到这座城的盟友驻军订单 city_id 是**出发城市**（删不到），
	//   必须按 target_id 清掉，并给各驻军方发战报
	var gars []model.EzfyOrder
	h.DB.Where("target_id = ? AND target_type = 3 AND order_type = 6 AND status = 3", cid).Find(&gars)
	for _, go_ := range gars {
		var gc model.EzfyCity
		gcName := "友军城市"
		if err := h.DB.First(&gc, go_.CityId).Error; err == nil {
			gcName = gc.Name
		}
		h.addReport(go_.UserID, 5, "驻防战报: 驻防城市被摧毁",
			fmt.Sprintf("你驻守的「%s」已被摧毁!\n你的驻军(来自%s)已全部损失。", ct.Name, gcName), "", 0, cid)
	}
	if len(gars) > 0 {
		h.DB.Where("target_id = ? AND target_type = 3 AND order_type = 6 AND status = 3", cid).
			Delete(&model.EzfyOrder{})
	}
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
// SetTax 调整税率
//
// ★ 2026-09-28 用户规则：**民心 + 税率 = 100**，设税率时民心**立即**联动到 (100 − 税率)。
// 例如税率设成 20% → 民心立即变 80。之后被征服/掠夺把民心打下去时税率不动，
// 民心由 calcResource 每小时自动回归到基准值（见那里的说明）。
func (h *EzfyHandler) SetTax(c *gin.Context) {
	uid := middleware.GetUID(c)
	ezfyPageCacheDel(uid) // 改税率 → 资源详情缓存失效
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
	// ★ 民心立即联动：民心 = 100 − 税率
	feelings := 100 - req.TaxRate
	h.DB.Model(&model.EzfyCity{}).Where("id = ?", city.ID).
		Updates(map[string]interface{}{"tax_rate": req.TaxRate, "feelings": feelings})
	resp.OK(c, gin.H{"msg": fmt.Sprintf("税率已调整为%d%%, 民心联动为%d", req.TaxRate, feelings)})
}

// Convene 召集人口（只消耗粮食，夜间只 +人口）
func (h *EzfyHandler) Convene(c *gin.Context) {
	uid := middleware.GetUID(c)
	ezfyPageCacheDel(uid) // 召集人口 → 资源详情缓存失效
	// ★ 2026-10-05 卡控（防连点刷人口）；★ 2026-10-09 用户要求 5 秒 → 3 秒
	{
		nowCd := time.Now().UnixMilli()
		ezfyConveneMu.Lock()
		if last, ok := ezfyConveneMemo[uid]; ok && nowCd-last < 3000 {
			ezfyConveneMu.Unlock()
			resp.ParamError(c, "操作过于频繁, 请 3 秒后再试")
			return
		}
		if len(ezfyConveneMemo) > 16384 {
			ezfyConveneMemo = map[uint]int64{}
		}
		ezfyConveneMemo[uid] = nowCd
		ezfyConveneMu.Unlock()
	}
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
	// ★★ 2026-09-26 修复「召集只该耗粮食，结果其他/更多粮食也被一起扣了」：
	//   注意这里**不再调 h.calcResource(city)** —— calcResource 是懒结算，会顺带扣
	//   「军队耗粮 + 军官工资」并回写全部资源，导致玩家一点召集就看到粮食(及其它)一起掉。
	//   召集是即时操作，只用当前已落库的粮食/人口判断与扣减，只写 food 和 pop 两列即可。
	foodCost := ezfyConveneFoodCostCfg()
	popGain := ezfyConvenePopGainCfg()
	baseGain := popGain // 未截断前的「单次召集人口」，用于按比例折算粮食
	// ★★ 2026-10-09 用户报「人口 46 万时再召集被整体拒绝，应该仍能召集、只是加到上限为止」：
	//   原来两个上限都是「本次召集后会超上限 → **整单拒绝**」→ 人口已经贴近上限的玩家
	//   一次也召集不了（线上 45 万~50 万之间有 400 座城被误拒），明明还差几十万才满。
	//   现在改成 **按上限截断**：能加多少加多少，刚好加到上限为止；
	//   只有「人口已经达到/超过上限」时才拒绝（那时确实一点也加不进去）。
	//   ⚠️ 截断后粮食**按比例收**（见下），否则会出现「花 20 万粮只加 2 点人口」这种坑。
	//
	//   全局硬性上限（管理端「二战系统配置」可配，0 = 不限），对召集**永远**生效。
	if popCap := ezfyConvenePopMaxCfg(); popCap > 0 {
		if city.Pop >= popCap {
			resp.ParamError(c, fmt.Sprintf("人口已达上限(%d), 无法继续召集", popCap))
			return
		}
		if city.Pop+popGain > popCap {
			popGain = popCap - city.Pop
		}
	}
	// ★ 2026-09-26 加「民居容量限制 / 召集人口灵活配置」两个开关：
	//   只有「民居容量限制」开着（民居上限才存在）且「召集人口灵活配置」关着时，
	//   召集才受民居容量上限约束；任一条件不满足都维持原有的「可突破上限」行为。
	// ★ 2026-10-09 与硬性上限同口径：改成按上限截断，而不是整单拒绝。
	if ezfyHousePopLimitOn() && !ezfyConveneFlexOn() {
		if city.Pop >= city.PopMax {
			resp.ParamError(c, fmt.Sprintf("人口已达民居容纳上限(%d), 无法继续召集", city.PopMax))
			return
		}
		if city.Pop+popGain > city.PopMax {
			popGain = city.PopMax - city.Pop
		}
	}
	// ★ 2026-10-09 截断后粮食按比例折算：原价「foodCost 粮 = baseGain 人口」，
	//   本次只加 popGain 人口 → 只收 foodCost × popGain / baseGain（向下取整，至少 1）。
	if popGain < baseGain && baseGain > 0 {
		foodCost = foodCost * popGain / baseGain
		if foodCost < 1 {
			foodCost = 1
		}
	}
	if city.Food < foodCost {
		resp.ParamError(c, fmt.Sprintf("粮食不足, 召集%d人口需要%d粮食", popGain, foodCost))
		return
	}
	city.Food -= foodCost
	city.Pop += popGain
	// ★ 2026-10-09 被上限截断时把「为什么只加了这么多」说清楚，
	//   否则玩家看到「人口+2」会以为是 bug（其实是已顶到上限）。
	conveneMsg := fmt.Sprintf("召集成功, 人口+%d", popGain)
	if popGain < baseGain {
		conveneMsg = fmt.Sprintf("召集成功, 人口+%d(已到人口上限, 本次只补到上限)", popGain)
	}
	// 只写 food / pop 两列，绝不回写其它资源（这就是上面那句「只耗粮食」的落点）。
	h.DB.Model(&model.EzfyCity{}).Where("id = ?", city.ID).Updates(map[string]interface{}{
		"food": city.Food,
		"pop":  city.Pop,
	})
	resp.OK(c, gin.H{"msg": conveneMsg, "pop": city.Pop, "pop_max": city.PopMax, "gain": popGain})
}

// Placate 安抚民心
//
// ★ 2026-09-28 用户规则（本次改动）：
//   - 花费固定 **5 万黄金**；
//   - 效果：**民怨 −2、民心 +1**（原来是一次性花「民怨×100」黄金把民怨清零）；
//   - **15 分钟只能安抚一次**（用 city.PlacateTime 记录上次安抚时间）。
//
// 注意：安抚把民心 +1 会让它暂时高于「100 − 税率」的基准，
// 之后由 calcResource 的回归逻辑每分钟 −1 慢慢落回基准 —— 符合用户「安抚只是临时顶一下」的预期。
func (h *EzfyHandler) Placate(c *gin.Context) {
	uid := middleware.GetUID(c)
	ezfyPageCacheDel(uid) // 安抚 → 资源详情缓存失效
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
	// 15 分钟冷却
	now := time.Now().UnixMilli()
	if city.PlacateTime > 0 {
		left := city.PlacateTime + ezfyPlacateCooldownMs() - now
		if left > 0 {
			resp.ParamError(c, fmt.Sprintf("安抚冷却中, 还需 %d 秒", (left+999)/1000))
			return
		}
	}
	cost := ezfyPlacateGoldCost()
	if city.Gold < cost {
		resp.ParamError(c, fmt.Sprintf("黄金不足, 安抚需要%d黄金", cost))
		return
	}
	// 效果：民怨 −2、民心 +1（各自夹取到 [0,100]）
	grievance := city.Grievance - ezfyPlacateGrievanceDown()
	if grievance < 0 {
		grievance = 0
	}
	feelings := city.Feelings + ezfyPlacateFeelingsUp()
	if feelings > 100 {
		feelings = 100
	}
	city.Gold -= cost
	city.Feelings = feelings
	city.Grievance = grievance
	city.PlacateTime = now
	h.saveCityRes(city)
	h.DB.Model(&model.EzfyCity{}).Where("id = ?", city.ID).
		Updates(map[string]interface{}{"feelings": feelings, "grievance": grievance,
			"placate_time": now})
	resp.OK(c, gin.H{"msg": fmt.Sprintf("安抚成功: 民怨 -%d → %d, 民心 +%d → %d",
		ezfyPlacateGrievanceDown(), grievance, ezfyPlacateFeelingsUp(), feelings)})
}

// AbandonWildland 放弃野地
func (h *EzfyHandler) AbandonWildland(c *gin.Context) {
	uid := middleware.GetUID(c)
	ezfyPageCacheDel(uid) // 放弃野地 → 附属野地缓存失效
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
//
// ★ 2026-10-04 性能（用户反馈「/resources 2s+」）：第一波 档案+城市列表（1 RTT）定当前城，
//   第二波 6 条计算数据并行（1 RTT）；加 3s 玩家级缓存（税率/增产/安抚等写操作会失效）。
func (h *EzfyHandler) Resources(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		CityId int64 `json:"city_id"`
	}
	_ = c.ShouldBindJSON(&req)
	if it, ok := ezfyPageCacheGet(uid, "resources"); ok {
		resp.OK(c, it)
		return
	}
	// ★ 2026-09-26 补：本接口原来没调 h.cfgs()，而 getResourceCalc / calcResource 都依赖
	//   ezfyCfg（建筑等级表、兵种表）。若它是本次进程里第一个「要用配置」的请求，
	//   ezfyCfg 还是零值 → 建筑产量全算 0（表现为「基础产量 0」）。
	//   正常流程下前端会先打 /view（那里有 cfgs），所以平时不暴露，但不能靠别人兜底。
	h.cfgs()
	_, city, _ := h.ezfyPageCity(uid)
	if req.CityId > 0 {
		if cc := h.cityOf(uid, req.CityId); cc != nil {
			city = *cc
		}
	}
	// ★ 2026-10-04 性能（用户反馈「/resources 卡 2s」）：原来 getResourceCalc(nil)
	//   内部**串行**查 techMap/buildingList/wildlandList/troopMap + mayor/boost 共 6 条 SQL，
	//   跨 WAN 慢 RDS 下单请求 1-2s。改为 6 条独立查询**并行**预载，再走
	//   getResourceCalcWith 纯内存版（零 SQL）。
	// ★ 2026-10-05：同一份快照**同时**喂给懒结算 calcResourceD —— 原来 calcResource 自己又把这
	//   6 张表串行查了一遍（每请求白打 6 个跨 WAN 往返），现在整条链只剩 1 次写回。
	var (
		buildings []model.EzfyCityBuilding
		techs     map[int]int
		wilds     []model.EzfyWildland
		troops    map[int]int64
		boost     *model.EzfyCityEffect
		mayor     int
	)
	var wg sync.WaitGroup
	wg.Add(6)
	go func() { defer wg.Done(); buildings = h.buildingList(city.ID) }()
	go func() { defer wg.Done(); techs = h.techMapOf(uid) }()
	go func() { defer wg.Done(); wilds = h.wildlandList(city.ID) }()
	go func() { defer wg.Done(); troops = h.troopMap(city.ID) }()
	go func() { defer wg.Done(); boost = h.ezfyLoadActiveBoost(city.ID) }()
	go func() { defer wg.Done(); mayor = h.mayorBonusPct(city.ID) }()
	wg.Wait()
	snap := &resCalcData{buildings: buildings, techs: techs, wilds: wilds, troops: troops,
		boost: boost, boostDone: true, mayor: mayor}
	// 先结算（把资源推进到当前时刻），再按同一份快照出详情 —— 顺序与改造前一致。
	h.calcResourceD(&city, snap)
	data := gin.H{"city": city, "calc": h.getResourceCalcWith(&city, snap)}
	ezfyPageCacheSet(uid, "resources", data)
	resp.OK(c, data)
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
