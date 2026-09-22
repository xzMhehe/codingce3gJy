package handler

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"gorm.io/gorm"

	"qqjiayuan/server/internal/model"
)

// 二战风云 地理/配置辅助：WorldGeo 大陆几何 + 坐标哈希地形 + 军衔 + 配置缓存

// ============ 大陆几何（复刻 WorldGeo.java，与 world_gen.py 同一套坐标系） ============

const (
	ezfyOcean = 0
)

var ezfyContinentNames = []string{"大海", "欧洲", "亚洲", "非洲", "北美洲", "南美洲", "大洋洲", "南极洲"}

// 形状: type=0 矩形(x0,y0,x1,y1); type=1 椭圆(cx,cy,a,b)
var ezfyLands = [][][]int{
	{{0, 3, 3, 40, 46}, {0, 3, 3, 13, 10}, {0, 30, 5, 44, 16}, {0, 36, 12, 46, 30}},
	{{0, 33, 46, 44, 60}, {1, 28, 74, 12, 14}, {1, 38, 98, 24, 24}, {0, 33, 60, 45, 72}},
	{{0, 50, 4, 86, 34}, {0, 55, 2, 65, 9}, {0, 46, 8, 50, 19}},
	{{0, 53, 32, 81, 84}, {0, 62, 76, 79, 92}, {0, 48, 76, 54, 80},
		{0, 26, 34, 34, 46}, {1, 30, 60, 16, 12}},
	{{0, 78, 2, 150, 36}, {0, 86, 36, 116, 58}, {0, 112, 50, 128, 68},
		{0, 122, 34, 148, 60}, {0, 140, 18, 150, 52}, {0, 102, 20, 116, 30}},
	{{0, 116, 96, 141, 124}, {0, 142, 124, 148, 127}, {0, 118, 70, 138, 82},
		{0, 140, 78, 148, 88}},
	{{0, 12, 133, 138, 148}},
}

var ezfyHoles = [][][]int{
	{{0, 13, 27, 25, 45}, {0, 23, 21, 38, 32}},
	{},
	{{0, 36, 24, 72, 30}, {0, 48, 30, 68, 38}, {0, 80, 24, 84, 30}},
	{{0, 60, 40, 70, 52}, {0, 72, 34, 78, 42}},
	{{0, 86, 22, 100, 34}, {0, 92, 36, 108, 46}},
	{},
	{},
}

var ezfyContinentIDs = []int{4, 5, 1, 3, 2, 6, 7}

// ezfyWorldSize 世界坐标范围（0 ~ ezfyWorldSize-1）
const ezfyWorldSize = 500

// ezfyLandScale 世界坐标 → 大陆几何坐标系(150×150) 的缩放系数
const ezfyLandScale = 150.0 / float64(ezfyWorldSize)

// ezfyTerrainSea 海洋地形 id（与 ezfyTerrainNames 下标一致）
const ezfyTerrainSea = 8

func ezfyInShape(x, y int, s []int) bool {
	// 大陆几何定义在 150×150 的坐标里，先把世界坐标缩回去再判形状
	fx := float64(x) * ezfyLandScale
	fy := float64(y) * ezfyLandScale
	if s[0] == 0 {
		return float64(s[1]) <= fx && fx <= float64(s[3]) && float64(s[2]) <= fy && fy <= float64(s[4])
	}
	a := float64(s[3])
	b := float64(s[4])
	dx := (fx - float64(s[1])) / a
	dy := (fy - float64(s[2])) / b
	return dx*dx+dy*dy <= 1
}

// EzfyContinentOf 是 ezfyContinentOf 的导出包装，供 CLI 运维工具（ezfymigrate）
// 在日志里显示落点所属大洲用；游戏逻辑仍走内部实现，两者行为完全一致。
func EzfyContinentOf(x, y int) int { return ezfyContinentOf(x, y) }

func ezfyContinentOf(x, y int) int {
	for i := 0; i < len(ezfyContinentIDs); i++ {
		for _, s := range ezfyLands[i] {
			if ezfyInShape(x, y, s) {
				for _, h := range ezfyHoles[i] {
					if ezfyInShape(x, y, h) {
						return ezfyOcean
					}
				}
				return ezfyContinentIDs[i]
			}
		}
	}
	return ezfyOcean
}

func ezfyContinentName(x, y int) string {
	c := ezfyContinentOf(x, y)
	if c < 0 || c >= len(ezfyContinentNames) {
		return "大海"
	}
	return ezfyContinentNames[c]
}

// ezfyRegionName 该坐标属于哪个「大洲 / 大洋」，用于地图格子与城市、野地的所属标注。
func ezfyRegionName(x, y int) string {
	if ezfyContinentOf(x, y) != ezfyOcean {
		return ezfyContinentName(x, y)
	}
	return ezfyOceanName(x, y)
}

// ezfyOceanName 大洋名称（按世界地图的方位粗略划分）。
//
// 说明：原版只给了「七大洲」，没有给海洋分区名。这里按常见的大洋方位补上，
// 让玩家在地图上看到的不只是「海洋」两个字。
func ezfyOceanName(x, y int) string {
	switch {
	case y < 20:
		return "北冰洋"
	case y > 440:
		return "南冰洋"
	case x < 60 || x > 420:
		return "太平洋"
	case x < 160:
		return "大西洋"
	default:
		return "印度洋"
	}
}

// ============ 坐标哈希（复刻 GameServiceImpl getTerrain/getWildlandLevel/getKouLevel） ============

func ezfyAbs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// ezfyLandTerrain 陆地地形（1~7，不含海洋）
//
// ★ 用一个**非周期**的混合散列，避免旧实现那种 8×8 重复条纹。
func ezfyLandTerrain(x, y int) int {
	h := ezfyAbs(x*374761393 + y*668265263)
	h = (h ^ (h >> 13)) * 1274126177
	h = ezfyAbs(h ^ (h >> 16))
	return h%7 + 1
}

// ezfyTerrain 地形：**先判大陆**，海洋连成片，陆地上再取 1~7 的地形。
//
// ★ 用户反馈「海洋是小水坑、地图不对」的根因：
//
//	旧实现直接写 `abs(x*73856093 ^ y*19349663) % 8 + 1`，而这个哈希**周期是 8**
//	（(x+8) 与 x 同余），于是整张地图是 8×8 的重复图案、8 种地形均匀铺满，
//	海洋只占 1/8 且互不相连 —— 看起来就是满地小水坑，也谈不上「世界地图」。
//
//	现在改成先按七大洲几何判陆地/海洋，海洋自然连成大片，
//	陆地内部再用散列取 1~7 的地形（1 平原 2 草原 3 森林 4 盆地 5 丘陵 6 沼泽 7 山地）。
func ezfyTerrain(x, y int) int {
	if ezfyContinentOf(x, y) == ezfyOcean {
		return ezfyTerrainSea
	}
	return ezfyLandTerrain(x, y)
}

// ezfyTerrainCoastalPlain 沿海平原（地形 id 9）—— ★ 海城只能建在这里
//
// 用户规则：海城不是建在「海洋」上，而是建在「沿海平原」上。
// 沿海平原 = 平原(1) 且 8 邻域内存在海洋(8)，是派生地形、不占哈希桶，
// 这样原版 8 种地形（含珠宝按地形 1-8 的映射）完全不受影响。
const ezfyTerrainCoastalPlain = 9

// ezfyHasSeaNeighbor 8 邻域内是否有海洋
func ezfyHasSeaNeighbor(x, y int) bool {
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			if dx == 0 && dy == 0 {
				continue
			}
			if ezfyTerrain(x+dx, y+dy) == ezfyTerrainSea {
				return true
			}
		}
	}
	return false
}

// ezfyIsCoastalPlainAt 该坐标是否沿海平原
//
// ★ 改成大陆地图后，这里**不再需要**旧实现那个「1/3 抽样」的补丁：
//
//	旧地图是周期 8 的图案，每一个平原格都恰好有一个海洋邻居，
//	只能用独立散列从「靠海的平原」里挑出 1/3，否则平原会 100% 变成沿海平原。
//	现在大陆是成片的，内陆平原本来就没有海洋邻居，天然区分开了。
func ezfyIsCoastalPlainAt(x, y int) bool {
	return ezfyTerrain(x, y) == 1 && ezfyHasSeaNeighbor(x, y)
}

// ezfyTerrainEx 实际地形：平原且靠海 → 沿海平原(9)，其余同 ezfyTerrain
//
// ★ 只用于「展示 + 建城选址 + 海城判定」；
//
//	依赖原版 8 种地形的玩法逻辑（珠宝按地形、野地产出）继续用 ezfyTerrain。
func ezfyTerrainEx(x, y int) int {
	// ★ 管理端「地图格子覆盖」优先：配了地形就强制用配置值
	if t := ezfyTileAt(x, y); t != nil && t.Terrain > 0 {
		return t.Terrain
	}
	if ezfyIsCoastalPlainAt(x, y) {
		return ezfyTerrainCoastalPlain
	}
	return ezfyTerrain(x, y)
}

// ezfyTileKey 覆盖表的键（世界坐标上限 1000，够用）
func ezfyTileKey(x, y int) int64 { return int64(x)*100000 + int64(y) }

// ezfyTileAt 取某格的覆盖配置（没有则 nil）
func ezfyTileAt(x, y int) *model.EzfyMapTile {
	c := &ezfyCfg
	c.mu.RLock()
	defer c.mu.RUnlock()
	if t, ok := c.tiles[ezfyTileKey(x, y)]; ok {
		return &t
	}
	return nil
}

// ezfyMarkKindAt 取某格的「标记类型」（0=无 1=寇城 2=活动寇城 3=活动野地 4=特殊城市）
func ezfyMarkKindAt(x, y int) int {
	if t := ezfyTileAt(x, y); t != nil {
		return t.MarkKind
	}
	return 0
}

// ezfyMarkLevelAt 取某格的活动等级（覆盖优先，没覆盖就按哈希算）
func ezfyMarkLevelAt(x, y int) int {
	if t := ezfyTileAt(x, y); t != nil && t.MarkLevel > 0 {
		return t.MarkLevel
	}
	return ezfyActivityLevel(x, y)
}

// ezfyTerrainNames 地形名（复刻原版 MapController.TERRAIN_NAMES，索引即地形 id）
// 1平原 2草原 3森林 4盆地 5丘陵 6沼泽 7山地 8海洋 9沿海平原(本项目扩展)
var ezfyTerrainNames = []string{"", "平原", "草原", "森林", "盆地", "丘陵", "沼泽", "山地", "海洋", "沿海平原"}

// ezfyTerrainName 地形 id → 中文名
func ezfyTerrainName(t int) string {
	if t < 1 || t >= len(ezfyTerrainNames) {
		return "未知"
	}
	return ezfyTerrainNames[t]
}

// ezfyTerrainNameEx 按坐标取实际地形名（含沿海平原）
func ezfyTerrainNameEx(x, y int) string {
	return ezfyTerrainName(ezfyTerrainEx(x, y))
}

// ============ 一次性数据迁移：旧版海城 → 沿海平原 ============

var ezfySeaMigrateOnce sync.Once

// ezfyMigrateSeaCities 把建在「海洋」上的旧海城搬到最近的「沿海平原」格。
//
// 背景：旧版把海城定义成「建在海洋地形上」，现在按用户规则改成「只能建在沿海平原上」。
// 幂等：迁完后不再有城市落在海洋地形，后续启动是空操作。
func ezfyMigrateSeaCities(db *gorm.DB) {
	var cities []model.EzfyCity
	db.Find(&cities)
	moved := 0
	for _, ct := range cities {
		if ezfyTerrain(ct.X, ct.Y) != 8 {
			continue
		}
		pos, ok := ezfyNearestCoastalPlain(db, ct.X, ct.Y)
		if !ok {
			log.Printf("ezfy 海城迁移: 城%d (%d,%d) 附近找不到沿海平原，跳过", ct.ID, ct.X, ct.Y)
			continue
		}
		oldX, oldY := ct.X, ct.Y
		updates := map[string]interface{}{"x": pos[0], "y": pos[1]}
		// 自动命名的「新城X,Y」跟着新坐标走（玩家自己改过的名字不动）
		if ct.Name == fmt.Sprintf("新城%d,%d", oldX, oldY) {
			updates["name"] = fmt.Sprintf("新城%d,%d", pos[0], pos[1])
		}
		if err := db.Model(&model.EzfyCity{}).Where("id = ?", ct.ID).
			Updates(updates).Error; err != nil {
			log.Printf("ezfy 海城迁移失败 城%d: %v", ct.ID, err)
			continue
		}
		// 清掉旧坐标上的「玩家城」地图区域记录（新坐标由后续逻辑重建）
		db.Where("x = ? AND y = ? AND area_type = ?", oldX, oldY, 3).Delete(&model.EzfyMapArea{})
		log.Printf("ezfy 海城迁移: 城%d「%s」(%d,%d) → (%d,%d) 沿海平原", ct.ID, ct.Name, oldX, oldY, pos[0], pos[1])
		moved++
	}
	if moved > 0 {
		log.Printf("ezfy 海城迁移完成，共 %d 座", moved)
	}
}

// ezfyNearestCoastalPlain 从 (x,y) 向外螺旋找最近的、无城市的沿海平原格
func ezfyNearestCoastalPlain(db *gorm.DB, x, y int) ([2]int, bool) {
	for r := 1; r <= 150; r++ {
		for dx := -r; dx <= r; dx++ {
			for dy := -r; dy <= r; dy++ {
				// 只走外圈
				if ezfyAbs(dx) != r && ezfyAbs(dy) != r {
					continue
				}
				nx, ny := x+dx, y+dy
				if nx < 0 || ny < 0 || nx > 499 || ny > 499 {
					continue
				}
				if !ezfyIsCoastalPlainAt(nx, ny) {
					continue
				}
				var n int64
				db.Model(&model.EzfyCity{}).Where("x = ? AND y = ?", nx, ny).Count(&n)
				if n > 0 {
					continue
				}
				return [2]int{nx, ny}, true
			}
		}
	}
	// ★ 兜底：实在找不到沿海平原（世界地图改版后，落在远洋上的城市可能离海岸很远），
	//   退一步找一块「无城市的平原」，保证城市不会一直卡在海洋里。
	//   宁可牺牲「海城」属性，也不能让玩家的城留在水里。
	for r := 1; r <= 150; r++ {
		for dx := -r; dx <= r; dx++ {
			for dy := -r; dy <= r; dy++ {
				if ezfyAbs(dx) != r && ezfyAbs(dy) != r {
					continue
				}
				nx, ny := x+dx, y+dy
				if nx < 0 || ny < 0 || nx > 499 || ny > 499 {
					continue
				}
				if ezfyTerrain(nx, ny) != 1 {
					continue
				}
				var n int64
				db.Model(&model.EzfyCity{}).Where("x = ? AND y = ?", nx, ny).Count(&n)
				if n > 0 {
					continue
				}
				return [2]int{nx, ny}, true
			}
		}
	}
	return [2]int{}, false
}

func ezfyWildlandLevel(x, y int) int {
	h := ezfyAbs(x*83492791 ^ y*6291469)
	return h % 11
}

func ezfyKouLevel(x, y int) int {
	h := ezfyAbs(x*83492791 ^ y*6291469)
	return h % 11
}

// ============ 军衔（复刻 ConfigServer.RANKS，20 档） ============

// 军衔数据已迁到 ezfy_cfg_rank 表（见 model.EzfyCfgRank / ezfyDefaultRanks），
// 这里不再保留第二份硬编码，避免两处不一致。

// ezfyRankIndex 声望 → 军衔下标（读配置缓存，管理端改过军衔表也生效）
func ezfyRankIndex(prestige int) int {
	ranks := ezfyCfg.rankList()
	idx := 0
	for i := range ranks {
		if prestige >= ranks[i].NeedPrestige {
			idx = i
		}
	}
	return idx
}

// rankList 军衔列表（缓存为空时回落内置默认）
func (c *ezfyConfigCache) rankList() []model.EzfyCfgRank {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.ranks) == 0 {
		return ezfyDefaultRanks()
	}
	return c.ranks
}

// ezfyRankOf 声望对应的军衔配置
func ezfyRankOf(prestige int) model.EzfyCfgRank {
	ranks := ezfyCfg.rankList()
	idx := ezfyRankIndex(prestige)
	if idx < 0 || idx >= len(ranks) {
		idx = 0
	}
	return ranks[idx]
}

func ezfyRankName(prestige int) string { return ezfyRankOf(prestige).Name }
func ezfyRankPost(prestige int) string { return ezfyRankOf(prestige).Post }

// ezfyRankCityMax 该声望下最多能拥有几座城（★ 军衔限制分城数量）
func ezfyRankCityMax(prestige int) int {
	m := ezfyRankOf(prestige).CityMax
	if m <= 0 {
		m = 1
	}
	return m
}

// ============ 配置缓存（进程内加载，seed 完成后首用时加载） ============

type ezfyConfigCache struct {
	mu             sync.RWMutex
	loaded         bool
	buildings      map[int]model.EzfyCfgBuilding
	buildingLvls   map[int]map[int]model.EzfyCfgBuildingLevel
	troops         map[int]model.EzfyCfgTroop
	techs          map[int]model.EzfyCfgTech
	techLvls       map[int]map[int]model.EzfyCfgTechLevel
	wildlands      map[int]map[int]model.EzfyCfgWildland
	items          map[int]model.EzfyCfgItem
	generals       map[int]model.EzfyCfgGeneral
	skills         map[int]model.EzfyCfgSkill
	skillByName    map[string]int
	equipments     map[int]model.EzfyCfgEquipment
	buildingByName map[string]int
	techByName     map[string]int
	// 地图格子覆盖（key = x*100000+y），管理端改完走 cfgsReload 生效
	tiles map[int64]model.EzfyMapTile
	// 军衔配置（按等级 1..N 排序）
	ranks []model.EzfyCfgRank
	// 建筑数量上限（军事区/资源区分开，管理端可维护）
	limit model.EzfyCfgLimit
	// 二战聊天敏感词（独立维护页）
	words []model.EzfyWordFilter
}

// ezfyLimit 取建筑数量上限配置（带默认值兜底）
func ezfyLimit() model.EzfyCfgLimit {
	l := ezfyCfg.limit
	if l.MilitaryMax <= 0 {
		l.MilitaryMax = 33
	}
	if l.ResourceMax <= 0 {
		l.ResourceMax = 33
	}
	if l.HouseMax <= 0 {
		l.HouseMax = 10
	}
	return l
}

// ezfyGatherMax 单次出征最多使用几个集结令（读 ezfy_cfg_limit.gather_max_per_order）
//
// ★ 用户要求「出征集结令上限后台管理系统可维护，最大默认 50」，默认 50。
// 0 或未配置时回落默认值（集结令上限为 0 无意义 —— 等于禁用了这个道具）。
func ezfyGatherMax() int {
	if n := ezfyCfg.limit.GatherMaxPerOrder; n > 0 {
		return n
	}
	return ezfyGatherMaxDefault
}

// ============ 战斗 / 经济数值（ezfy_cfg_limit，管理端可维护）============
//
// ★ 这几个值都「0 无意义」：0 = 不扣民心 / 军官免费 / 恢复免费，
//
//	所以读到 <= 0 时一律回落默认值（与 ezfyGatherMax 同一套兜底思路）。
const (
	ezfyConquerFeelingsDef  = 2   // 征服单次最多扣民心（默认 2）
	ezfyLootFeelingsDef     = 2   // 掠夺每次扣民心（默认 2）
	ezfyOfficerSalaryDef    = 2   // 军官工资：每级每小时黄金（默认 2）
	ezfyWoundHealDivisorDef = 100 // 恢复伤兵黄金 = 兵种总造价 / 该值（默认 100）
	// ★ 商城单次购买数量上限（默认 9999；原来前端写死 99）
	ezfyMallBuyMaxDef = 9999
)

func ezfyLimitOr(v, def int) int {
	if v > 0 {
		return v
	}
	return def
}

// ezfyConquerFeelingsCfg 征服(3) 单次最多扣目标多少民心
func ezfyConquerFeelingsCfg() int {
	return ezfyLimitOr(ezfyCfg.limit.ConquerFeelingsMax, ezfyConquerFeelingsDef)
}

// ezfyLootFeelingsCfg 掠夺(2) 每次扣目标多少民心
func ezfyLootFeelingsCfg() int {
	return ezfyLimitOr(ezfyCfg.limit.LootFeelings, ezfyLootFeelingsDef)
}

// ezfyOfficerSalaryPerLvCfg 军官工资：每名军官每小时消耗「等级 × 该值」黄金
func ezfyOfficerSalaryPerLvCfg() int {
	return ezfyLimitOr(ezfyCfg.limit.OfficerSalaryPerLevel, ezfyOfficerSalaryDef)
}

// ezfyWoundHealDivisorCfg 恢复伤兵单价系数：单价 = ceil(兵种总造价 / 该值)，最低 1 黄金
func ezfyWoundHealDivisorCfg() int {
	return ezfyLimitOr(ezfyCfg.limit.WoundHealDivisor, ezfyWoundHealDivisorDef)
}

// ezfyMallBuyMaxCfg 商城单次购买数量上限（下限恒为 1，默认 9999）
//
// ★ 用户要求「商城购买现在卡控 1-99，改成可配置的，默认 1-9999」。
//
//	前端输入框 max、前端校验、后端校验**都**读这一个值，避免两边不一致。
func ezfyMallBuyMaxCfg() int {
	return ezfyLimitOr(ezfyCfg.limit.MallBuyMax, ezfyMallBuyMaxDef)
}

// ============ 系统配置：玩法开关 + 野地兵力倍数 ============
//
// ⚠️ 这三个开关是「0 有意义」的字段（0 = 关），所以**不能**用 ezfyLimitOr ——
// 那个把 0 当「没配置」回落默认值，会把管理员关掉的开关又打开。
// 直接用 `!= 0` 判断：只有显式存了 0 才算关。
// 默认值靠两处保证：① DB 列默认值 1（seed 补列时 ALTER ... DEFAULT 1）；
// ② 老行 NULL 由 seed 回填 1（只回填 NULL，不动 0）。
const (
	ezfyRecruitCostDef = 1 // 征兵消耗资源：默认开
	ezfyFoodUpkeepDef  = 1 // 军队耗粮：默认开
	ezfyMarchOilDef    = 1 // 出征油耗：默认开
	ezfyWildMultDef    = 1 // 野地兵力倍数：默认 1
)

// ezfyRecruitCostOn 征兵是否消耗资源（关 = 不消耗资源、也无需空闲人口）
func ezfyRecruitCostOn() bool {
	return ezfyCfg.limit.RecruitCostOn != 0
}

// ezfyFoodUpkeepOn 城内军队是否每小时耗粮
func ezfyFoodUpkeepOn() bool {
	return ezfyCfg.limit.FoodUpkeepOn != 0
}

// ezfyMarchOilOn 出征是否消耗石油
func ezfyMarchOilOn() bool {
	return ezfyCfg.limit.MarchOilOn != 0
}

// ezfyWildTroopMult 野地/海野/寇城守军兵力倍数（默认 1；0 或负数无意义 → 回落 1）
func ezfyWildTroopMult() float64 {
	if m := ezfyCfg.limit.WildTroopMult; m > 0 {
		return m
	}
	return ezfyWildMultDef
}

// ezfyScaleByWildMult 把守军兵力按倍数放大（最少 1 个，避免倍数 < 1 时把守军抹成 0）
func ezfyScaleByWildMult(n int64) int64 {
	m := ezfyWildTroopMult()
	if m == 1 {
		return n
	}
	v := int64(float64(n) * m)
	if v < 1 {
		v = 1
	}
	return v
}

// ezfyWords 取二战聊天敏感词
func ezfyWords() []model.EzfyWordFilter {
	return ezfyCfg.words
}

// ezfyFilterChat 过滤二战聊天内容。
// 返回 (过滤后的文本, 是否被拦截)。拦截类敏感词直接拒绝发言。
func ezfyFilterChat(text string) (string, bool) {
	out := text
	for _, w := range ezfyWords() {
		if w.Word == "" {
			continue
		}
		if !strings.Contains(out, w.Word) {
			continue
		}
		if w.Type == 2 {
			return out, true
		}
		rep := w.Replace
		if rep == "" {
			rep = strings.Repeat("*", len([]rune(w.Word)))
		}
		out = strings.ReplaceAll(out, w.Word, rep)
	}
	return out, false
}

var ezfyCfg ezfyConfigCache

// load 首次调用时从库里加载全部配置；已加载过就直接返回（等价于原来的 sync.Once）
func (c *ezfyConfigCache) load(db *gorm.DB) {
	c.mu.RLock()
	ok := c.loaded
	c.mu.RUnlock()
	if ok {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.loaded {
		return
	}
	c.loadLocked(db)
	c.loaded = true
}

// reload 强制重新读库。管理端在「建筑总配置 / 兵种配置 / 名将 / 技能 / 装备」里
// 改了参数后调用它，改动立刻生效，不用重启进程。
func (c *ezfyConfigCache) reload(db *gorm.DB) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.loadLocked(db)
	c.loaded = true
	// ★ 地图格子覆盖配置（ezfy_map_tile）可能改了地形 → 沿海平原索引必须重建，
	//   否则管理端新配的沿海/陆地不被迁城逻辑看到。
	ezfyInvalidateCoastalIndex()
}

// loadLocked 真正干活的部分，调用方必须已持有写锁
func (c *ezfyConfigCache) loadLocked(db *gorm.DB) {
	c.buildings = map[int]model.EzfyCfgBuilding{}
	c.buildingLvls = map[int]map[int]model.EzfyCfgBuildingLevel{}
	c.troops = map[int]model.EzfyCfgTroop{}
	c.techs = map[int]model.EzfyCfgTech{}
	c.techLvls = map[int]map[int]model.EzfyCfgTechLevel{}
	c.wildlands = map[int]map[int]model.EzfyCfgWildland{}
	c.items = map[int]model.EzfyCfgItem{}
	c.generals = map[int]model.EzfyCfgGeneral{}
	c.skills = map[int]model.EzfyCfgSkill{}
	c.skillByName = map[string]int{}
	c.equipments = map[int]model.EzfyCfgEquipment{}
	c.buildingByName = map[string]int{}
	c.techByName = map[string]int{}

	var bs []model.EzfyCfgBuilding
	db.Find(&bs)
	for _, b := range bs {
		c.buildings[b.ID] = b
		c.buildingByName[b.Name] = b.ID
	}
	var bls []model.EzfyCfgBuildingLevel
	db.Find(&bls)
	for _, l := range bls {
		m, ok := c.buildingLvls[l.BuildingId]
		if !ok {
			m = map[int]model.EzfyCfgBuildingLevel{}
			c.buildingLvls[l.BuildingId] = m
		}
		m[l.Level] = l
	}
	var ts []model.EzfyCfgTroop
	db.Find(&ts)
	for _, t := range ts {
		c.troops[t.ID] = t
	}
	var tcs []model.EzfyCfgTech
	db.Find(&tcs)
	for _, t := range tcs {
		c.techs[t.ID] = t
		c.techByName[t.Name] = t.ID
	}
	var tls []model.EzfyCfgTechLevel
	db.Find(&tls)
	for _, l := range tls {
		m, ok := c.techLvls[l.TechId]
		if !ok {
			m = map[int]model.EzfyCfgTechLevel{}
			c.techLvls[l.TechId] = m
		}
		m[l.Level] = l
	}
	var ws []model.EzfyCfgWildland
	db.Find(&ws)
	for _, w := range ws {
		m, ok := c.wildlands[w.Type]
		if !ok {
			m = map[int]model.EzfyCfgWildland{}
			c.wildlands[w.Type] = m
		}
		m[w.Level] = w
	}
	var its []model.EzfyCfgItem
	db.Find(&its)
	for _, it := range its {
		c.items[it.ID] = it
	}
	var gens []model.EzfyCfgGeneral
	db.Find(&gens)
	for _, g := range gens {
		c.generals[g.ID] = g
	}
	var sks []model.EzfyCfgSkill
	db.Find(&sks)
	for _, s := range sks {
		c.skills[s.ID] = s
		c.skillByName[s.Name] = s.ID
	}
	var eqs []model.EzfyCfgEquipment
	db.Find(&eqs)
	for _, e := range eqs {
		c.equipments[e.ID] = e
	}

	// 地图格子覆盖（改地形 / 设寇城·活动寇城；管理端可维护）
	var tiles []model.EzfyMapTile
	db.Find(&tiles)
	tm := make(map[int64]model.EzfyMapTile, len(tiles))
	for _, t := range tiles {
		tm[ezfyTileKey(t.X, t.Y)] = t
	}
	c.tiles = tm

	// 军衔配置（管理端可维护；表为空时回落内置默认，保证排名逻辑永远可用）
	var rks []model.EzfyCfgRank
	db.Order("id").Find(&rks)
	if len(rks) == 0 {
		rks = ezfyDefaultRanks()
	}
	c.ranks = rks

	// 建筑数量上限（单行；缺行时用默认 33/33/10/0）
	// ★ 三个玩法开关的默认值也必须写在这里：缺行时如果留 0，会变成「全关」，
	//   与「默认开」的语义相反（见 ezfyRecruitCostOn / ezfyFoodUpkeepOn / ezfyMarchOilOn）。
	c.limit = model.EzfyCfgLimit{ID: 1, MilitaryMax: 33, ResourceMax: 33, HouseMax: 10, FactoryMax: 0,
		GatherMaxPerOrder: ezfyGatherMaxDefault, MallBuyMax: ezfyMallBuyMaxDef,
		WildTroopMult: ezfyWildMultDef,
		RecruitCostOn: ezfyRecruitCostDef, FoodUpkeepOn: ezfyFoodUpkeepDef, MarchOilOn: ezfyMarchOilDef}
	var lim model.EzfyCfgLimit
	if err := db.First(&lim, 1).Error; err == nil {
		c.limit = lim
	}

	// 二战聊天敏感词（独立维护页）
	var wds []model.EzfyWordFilter
	db.Order("id").Find(&wds)
	c.words = wds
}

// ezfyDefaultRanks 内置兜底军衔（与 seed 一致，复刻原版 rankIndex.html）
func ezfyDefaultRanks() []model.EzfyCfgRank {
	return []model.EzfyCfgRank{
		{ID: 1, Name: "列兵", Post: "士兵", NeedPrestige: 0, CityMax: 1},
		{ID: 2, Name: "上等兵", Post: "班长", NeedPrestige: 100, CityMax: 2},
		{ID: 3, Name: "下士", Post: "排长", NeedPrestige: 300, CityMax: 3},
		{ID: 4, Name: "中士", Post: "排长", NeedPrestige: 600, CityMax: 4},
		{ID: 5, Name: "上士", Post: "连长", NeedPrestige: 1000, CityMax: 5},
		{ID: 6, Name: "军士长", Post: "连长", NeedPrestige: 1500, CityMax: 6},
		{ID: 7, Name: "准尉", Post: "营长", NeedPrestige: 2200, CityMax: 7},
		{ID: 8, Name: "少尉", Post: "营长", NeedPrestige: 3000, CityMax: 8},
		{ID: 9, Name: "中尉", Post: "营长", NeedPrestige: 4000, CityMax: 9},
		{ID: 10, Name: "上尉", Post: "团长", NeedPrestige: 5200, CityMax: 10},
		{ID: 11, Name: "大尉", Post: "团长", NeedPrestige: 6600, CityMax: 11},
		{ID: 12, Name: "少校", Post: "旅长", NeedPrestige: 8200, CityMax: 12},
		{ID: 13, Name: "中校", Post: "旅长", NeedPrestige: 10000, CityMax: 13},
		{ID: 14, Name: "上校", Post: "旅长", NeedPrestige: 12000, CityMax: 14},
		{ID: 15, Name: "大校", Post: "师长", NeedPrestige: 14500, CityMax: 15},
		{ID: 16, Name: "少将", Post: "师长", NeedPrestige: 17500, CityMax: 16},
		{ID: 17, Name: "中将", Post: "军长", NeedPrestige: 21000, CityMax: 17},
		{ID: 18, Name: "上将", Post: "军长", NeedPrestige: 25000, CityMax: 18},
		{ID: 19, Name: "大将", Post: "军长", NeedPrestige: 30000, CityMax: 19},
		{ID: 20, Name: "五星上将", Post: "司令", NeedPrestige: 40000, CityMax: 20},
	}
}

func (c *ezfyConfigCache) general(id int) *model.EzfyCfgGeneral {
	if g, ok := c.generals[id]; ok {
		return &g
	}
	return nil
}

func (c *ezfyConfigCache) skill(id int) *model.EzfyCfgSkill {
	if s, ok := c.skills[id]; ok {
		return &s
	}
	return nil
}

func (c *ezfyConfigCache) equipment(id int) *model.EzfyCfgEquipment {
	if e, ok := c.equipments[id]; ok {
		return &e
	}
	return nil
}

func (c *ezfyConfigCache) building(id int) *model.EzfyCfgBuilding {
	if b, ok := c.buildings[id]; ok {
		return &b
	}
	return nil
}

func (c *ezfyConfigCache) buildingLevel(buildingId, level int) *model.EzfyCfgBuildingLevel {
	if m, ok := c.buildingLvls[buildingId]; ok {
		if l, ok := m[level]; ok {
			return &l
		}
	}
	return nil
}

func (c *ezfyConfigCache) troop(id int) *model.EzfyCfgTroop {
	if t, ok := c.troops[id]; ok {
		return &t
	}
	return nil
}

func (c *ezfyConfigCache) tech(id int) *model.EzfyCfgTech {
	if t, ok := c.techs[id]; ok {
		return &t
	}
	return nil
}

func (c *ezfyConfigCache) techLevel(techId, level int) *model.EzfyCfgTechLevel {
	if m, ok := c.techLvls[techId]; ok {
		if l, ok := m[level]; ok {
			return &l
		}
	}
	return nil
}

func (c *ezfyConfigCache) wildland(t, level int) *model.EzfyCfgWildland {
	if m, ok := c.wildlands[t]; ok {
		if w, ok := m[level]; ok {
			return &w
		}
	}
	return nil
}

func (c *ezfyConfigCache) item(id int) *model.EzfyCfgItem {
	if it, ok := c.items[id]; ok {
		return &it
	}
	return nil
}

func (c *ezfyConfigCache) troopName(troopId, camp int) string {
	t := c.troop(troopId)
	if t == nil {
		return ""
	}
	if camp == 2 && t.NameAxis != "" {
		return t.NameAxis
	}
	if camp == 1 && t.NameAlly != "" {
		return t.NameAlly
	}
	return t.Name
}
