package handler

import (
	"fmt"
	"log"
	"sort"
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
// 1平原 2草原 3森林 4盆地 5丘陵 6沼泽 7岛屿 8海洋 9沿海平原(本项目扩展)
var ezfyTerrainNames = []string{"", "平原", "草原", "森林", "盆地", "丘陵", "沼泽", "岛屿", "海洋", "沿海平原"}

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

// ============ 野地采集: 按地形的产出资源 + 宝物池（用户规范, 2026-09-23） ============
//
// 平原(1)/沿海平原(9)没有珠宝; 岛屿按原版「海岛」宝物列表。
// 宝物只在采集中掉落, 战斗中不掉宝物。

// ezfyTerrainTreasureNames 地形 id → 可采集宝物名称列表（展示按此顺序）
var ezfyTerrainTreasureNames = map[int][]string{
	2: {"玛瑙项坠", "翡翠项链", "祖母绿"},    // 草原 → 粮食
	3: {"黑曜石戒指", "黄金手镯", "红宝石戒指"}, // 森林 → 粮食
	4: {"黑曜石戒指", "琥珀项链", "铂金戒指"},  // 盆地 → 稀矿
	5: {"黄金手镯", "玛瑙项坠", "红宝石戒指"},  // 丘陵 → 钢铁
	6: {"琥珀项链", "黄金手镯", "蓝宝石戒指"},  // 沼泽 → 石油
	7: {"琥珀项链", "红宝石戒指", "祖母绿"},    // 岛屿 → 钢铁
	8: {"翡翠项链", "蓝宝石戒指", "祖母绿"},   // 海底森林 → 石油
}

// ezfyGatherResName 地形 → 采集产出的资源名（粮食/钢铁/石油/稀矿）
func ezfyGatherResName(t int) string {
	switch t {
	case 4:
		return "稀矿"
	case 5, 7:
		return "钢铁"
	case 6, 8:
		return "石油"
	default:
		return "粮食" // 平原/草原/森林/沿海平原
	}
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
	equipSetMap    map[int]model.EzfyCfgEquipSet
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

// ready 配置缓存是否已加载过。
//
// ★ 为什么需要它：开关类字段「0 = 关」是有意义的值，如果某个 handler 忘了先调 h.cfgs()
// 就直接读 ezfyCfg.limit，读到的是零值结构体 → 开关被误判成「关」
// （实测踩过：/war/status 不调 cfgs()，于是宣战开关默认开却显示成关）。
// 所以开关类读取函数一律先问 ready()，没加载过就回落「默认值」。
func (c *ezfyConfigCache) ready() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.loaded
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

// ezfyDispatchPeriod 常驻采集结算一期时长（毫秒）。
// ★ 2026-09-24 用户要求「采集 12 小时才有宝物 → 4 小时且可配置」：
//   读管理端「建筑上限/系统配置」ezfy_cfg_limit.dispatch_period_h（小时），默认 4。
func ezfyDispatchPeriod() int64 {
	if h := ezfyCfg.limit.DispatchPeriodH; h > 0 {
		return int64(h) * 3600 * 1000
	}
	return 4 * 3600 * 1000
}

// ezfyMarchSpeedBonus 出征速度加成（百分比，0 = 无加成）。
// ★ 2026-09-24 用户要求「节假日让玩家队伍走快点」：管理端可配。
//   实际行军时间 = 原时间 × 100/(100+加成)；默认 0。
//   加成 > 0 才生效，负值/未配置一律按 0（无加成）处理。
func ezfyMarchSpeedBonus() float64 {
	if b := ezfyCfg.limit.MarchSpeedBonus; b > 0 {
		return b
	}
	return 0
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
	// ★ 单城兵力上限：默认 50 亿。2026-09-23 线上「负数兵力」事故后新增 ——
	//   训练 / 伤兵恢复 / addTroop 三处共用，防止兵力累加溢出成负数。
	//   2026-09-23 用户要求「10 亿太少，上调到 50 亿」。
	ezfyTroopMaxDef = int64(5000000000)
	// ★ 伤兵在营存活天数：默认 5 天，超时未恢复自动消失。
	ezfyWoundExpireDaysDef = 5
	// ★ 资源数值安全上限：任何路径写入资源都不得超过它（约 1 万亿）。
	//   远小于 int64 上限，仅用于兜底防溢出；游戏内实际生效的仍是各城「仓储上限」。
	ezfyResSafeMax = int64(1000000000000)
)

// ezfyTroopMaxCfg 单城兵力上限（读 ezfy_cfg_limit.troop_max，默认 50 亿）
//
// ★ 2026-09-23 线上事故：玩家总兵力显示 -8843547888967622000 —— int64 正向溢出翻负。
// 根因是训练/伤兵恢复累加无上限。本值是唯一的「兵力上限」收敛点：
// trainTroop（训练前校验）、recoverWounded/recoverAllWounded（恢复前校验）、
// addTroop（落库前夹取）三处都读它。
func ezfyTroopMaxCfg() int64 {
	if v := ezfyCfg.limit.TroopMax; v > 0 {
		return v
	}
	return ezfyTroopMaxDef
}

// ezfyWoundExpireDaysCfg 伤兵在营存活天数（默认 5 天）
//
// 0 / 未配置无意义（等于伤兵永不过期）→ 回落默认 5 天。
func ezfyWoundExpireDaysCfg() int {
	return ezfyLimitOr(ezfyCfg.limit.WoundExpireDays, ezfyWoundExpireDaysDef)
}

// ezfyClampRes 资源数值夹取：负数归 0，超过安全上限则截断。
//
// ★ 只做「防溢出」兜底，**不**替代各城仓储上限（cap）逻辑 ——
// 正常产出的截断仍在 calcResource 里按 FoodCap/SteelCap/... 处理。
func ezfyClampRes(v int64) int64 {
	if v < 0 {
		return 0
	}
	if v > ezfyResSafeMax {
		return ezfyResSafeMax
	}
	return v
}

// ezfyAddRes 资源加法（安全版）：先夹取当前值，再做不会溢出的加法，结果恒在 [0, ezfyResSafeMax]。
func ezfyAddRes(cur, delta int64) int64 {
	return ezfySafeAdd(ezfyClampRes(cur), delta, ezfyResSafeMax)
}

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

// ezfyWoundHealRate 伤兵恢复黄金折扣率（百分比口径：配置 100 = 100% = 原价）。
//
// ★ 2026-09-23 用户要求：伤兵恢复黄金也有「折扣率数」，放管理端「二战系统配置」配，
//   节假日调低 = 恢复便宜。★ 2026-09-24 修正：默认 100 = 现在的正常值，0/负数 → 回落 100。
func ezfyWoundHealRate() float64 {
	v := ezfyCfg.limit.WoundHealRate
	if v <= 0 {
		v = 100
	}
	return v / 100
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
	ezfyWarRequireDef  = 1 // 宣战功能：默认开（掠夺/征服需先宣战且生效）
	ezfyMarchCapDef    = 1 // 出征兵力上限：默认开（按司令部等级算）
	ezfyWildMultDef    = 1 // 野地兵力倍数：默认 1
	// ★ 2026-09-25 用户反馈「野地打完获得的资源太少」→ 野地战利品资源倍率，默认 1
	ezfyWildResMultDef = 1
	// ★ 2026-09-25 用户要求「采集资源倍率也加到系统管理里」→ 常驻采集产出资源倍率，默认 1
	ezfyGatherResMultDef = 1
)

// ezfyMarchCapOn 出征是否受「兵力上限」限制（关 = 不限兵力）
//
// ★ 用户要求「再加个出征上限开关，默认开，关闭出征没有上限」。
//
//	关掉后 ezfyOrderTroopCap() 会返回 unlimited=true，出征校验与出征页提示一起放开。
func ezfyMarchCapOn() bool {
	if !ezfyCfg.ready() {
		return ezfyMarchCapDef != 0
	}
	return ezfyCfg.limit.MarchCapOn != 0
}

// ============ 军官升星配置（2026-09-22 用户要求，2026-09-23 按用户要求简化）============

const (
	// ★ 2026-09-23 用户要求「军官升星做得太复杂，优化简约点」：
	//   去掉概率开关 / 每高 1 星递减 / 成功率下限 / 失败保留徽章四个配置，
	//   只留「功能开关 + 固定成功率 + 每星加成 + 星级上限」。
	//   规则：每次升星消耗 1 枚星级徽章，按固定概率判定，失败星级不变（徽章照扣）。
	ezfyStarChanceDef   = 20 // 升星成功率%（固定值）
	ezfyStarAttrGainDef = 10 // 每升 1 星三维各 +N
	// ★ 用户规则「军官最多 5 星」→ 星级上限默认 5
	//   （军官池里的星级本来就只发 1~5 星，升星也不该超过 5）
	ezfyStarMaxDef = 5
	// 升星功能开关默认值（1 = 开 / 0 = 关）
	ezfyStarUpDef = 1
)

// ezfyStarUpOn 升星功能是否开启（关 = 升星卡不能用）
func ezfyStarUpOn() bool {
	if !ezfyCfg.ready() {
		return true
	}
	return ezfyCfg.limit.OfficerStarUpOn != 0
}

func ezfyStarChanceBase() int { return ezfyLimitOr(ezfyCfg.limit.OfficerStarChance, ezfyStarChanceDef) }
func ezfyStarAttrGain() int {
	return ezfyLimitOr(ezfyCfg.limit.OfficerStarAttrGain, ezfyStarAttrGainDef)
}
func ezfyStarMax() int { return ezfyLimitOr(ezfyCfg.limit.OfficerStarMax, ezfyStarMaxDef) }

// ezfyStarSuccessRate 升星成功率（%）：固定值，不再有按星级递减/下限那一套。
func ezfyStarSuccessRate() int {
	rate := ezfyStarChanceBase()
	if rate < 1 {
		rate = 1
	}
	if rate > 100 {
		rate = 100
	}
	return rate
}

// ============ 训练一键加速黄金倍率 ============

// ezfySpeedTrainRate 训练一键加速黄金倍率（百分比口径：配置 100 = 100% = 原价）。
//
// ★ 2026-09-23 用户要求：黄金消耗太多，价格倍率放管理端「二战系统配置」配，
//   节假日想便宜点就把倍率调低（如 50 = 半价、10 = 一折）。
//   ★ 2026-09-24 用户修正：默认值 100 才是正常值（而不是 1），设置 0.01 时仍觉得贵、
//   说明要按「百分比」理解 —— 100 = 现在的正常消耗。0/负数无意义 → 回落 100。
func ezfySpeedTrainRate() float64 {
	v := ezfyCfg.limit.SpeedTrainRate
	if v <= 0 {
		v = 100
	}
	return v / 100
}

// ezfyWarRequireOn 是否要求「先宣战才能掠夺/征服别人城市」
//
// ★ 用户要求「加一个宣战功能开关，默认开启：开启 = 玩家之间需要宣战；
//
//	关闭 = 不需要宣战也能掠夺/征服」。
//
// 关掉后 isAtWar() 恒为 true（视为随时可交战），所以出征校验、战斗结算、
//
//	玩家端按钮状态三处会一起放开，不需要各自打补丁。
//
// 实现见下面那一份（带 ready() 兜底）—— 这里只留注释，别再写一份同名函数。
func ezfyRecruitCostOn() bool {
	if !ezfyCfg.ready() {
		return ezfyRecruitCostDef != 0
	}
	return ezfyCfg.limit.RecruitCostOn != 0
}

// ezfyFoodUpkeepOn 城内军队是否每小时耗粮
func ezfyFoodUpkeepOn() bool {
	if !ezfyCfg.ready() {
		return ezfyFoodUpkeepDef != 0
	}
	return ezfyCfg.limit.FoodUpkeepOn != 0
}

// ezfyMarchOilOn 出征是否消耗石油
func ezfyMarchOilOn() bool {
	if !ezfyCfg.ready() {
		return ezfyMarchOilDef != 0
	}
	return ezfyCfg.limit.MarchOilOn != 0
}

// ezfyWarRequireOn 是否要求「先宣战才能掠夺/征服别人城市」
func ezfyWarRequireOn() bool {
	if !ezfyCfg.ready() {
		return ezfyWarRequireDef != 0
	}
	return ezfyCfg.limit.WarRequireOn != 0
}

// ezfyWildTroopMult 野地/海野/寇城守军兵力倍数（默认 1；0 或负数无意义 → 回落 1）
func ezfyWildTroopMult() float64 {
	if !ezfyCfg.ready() {
		return ezfyWildMultDef
	}
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

// ezfyWildResMult 野地/海野/寇城**战斗胜利后的战利品**资源倍率（默认 1；0 或负数无意义 → 回落 1）
//
// ★ 2026-09-25 用户反馈「野地打完获得的资源太少」→ 管理端「二战系统配置」可调。
//
//	作用点只有一处：ezfy_order.go 里算野地战利品 `rnd` 的那一步。
//	**不含**驻守采集（采集产出是「野地等级 × 800 × 后勤加成」另一套公式）。
func ezfyWildResMult() float64 {
	if !ezfyCfg.ready() {
		return ezfyWildResMultDef
	}
	if m := ezfyCfg.limit.WildResMult; m > 0 {
		return m
	}
	return ezfyWildResMultDef
}

// ezfyScaleByWildResMult 把战利品按倍率放大（保留至少 1，避免倍率 < 1 时把战利品抹成 0）
func ezfyScaleByWildResMult(n int64) int64 {
	m := ezfyWildResMult()
	if m == 1 {
		return n
	}
	v := int64(float64(n) * m)
	if v < 1 {
		v = 1
	}
	return v
}

// ezfyGatherResMult 常驻采集产出资源倍率（默认 1；0 或负数无意义 → 回落 1）
//
// ★ 2026-09-25 用户要求「采集资源倍率也加到系统管理里」→ 管理端「二战系统配置」可调。
//
//	作用点只有一处：ezfy_order.go 的 dispatchGatherYield（采集产出 = 等级 × 800 × 后勤加成 × 陆海系数）。
//	**不含**战斗战利品（那是 ezfyWildResMult）。
func ezfyGatherResMult() float64 {
	if !ezfyCfg.ready() {
		return ezfyGatherResMultDef
	}
	if m := ezfyCfg.limit.GatherResMult; m > 0 {
		return m
	}
	return ezfyGatherResMultDef
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
	c.equipSetMap = map[int]model.EzfyCfgEquipSet{}
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
	var esets []model.EzfyCfgEquipSet
	db.Find(&esets)
	for _, s := range esets {
		c.equipSetMap[s.ID] = s
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
		WildResMult:   ezfyWildResMultDef,
		GatherResMult: ezfyGatherResMultDef,
		RecruitCostOn: ezfyRecruitCostDef, FoodUpkeepOn: ezfyFoodUpkeepDef, MarchOilOn: ezfyMarchOilDef,
		WarRequireOn: ezfyWarRequireDef, MarchCapOn: ezfyMarchCapDef,
		// ★ 训练加速黄金倍率 / 伤兵恢复黄金折扣率：百分比口径，默认 100 = 100% = 原价
		SpeedTrainRate: 100, WoundHealRate: 100,
		// ★ 2026-09-23：兵力上限 / 伤兵存活天数的缺行兜底（0 无意义 → 默认 10 亿 / 5 天）
		TroopMax: ezfyTroopMaxDef, WoundExpireDays: ezfyWoundExpireDaysDef}
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
// ★ 2026-09-24 用户要求「军衔需要声望太少，统一在原来基础上 ×10」。
func ezfyDefaultRanks() []model.EzfyCfgRank {
	return []model.EzfyCfgRank{
		{ID: 1, Name: "列兵", Post: "士兵", NeedPrestige: 0, CityMax: 1},
		{ID: 2, Name: "上等兵", Post: "班长", NeedPrestige: 1000, CityMax: 2},
		{ID: 3, Name: "下士", Post: "排长", NeedPrestige: 3000, CityMax: 3},
		{ID: 4, Name: "中士", Post: "排长", NeedPrestige: 6000, CityMax: 4},
		{ID: 5, Name: "上士", Post: "连长", NeedPrestige: 10000, CityMax: 5},
		{ID: 6, Name: "军士长", Post: "连长", NeedPrestige: 15000, CityMax: 6},
		{ID: 7, Name: "准尉", Post: "营长", NeedPrestige: 22000, CityMax: 7},
		{ID: 8, Name: "少尉", Post: "营长", NeedPrestige: 30000, CityMax: 8},
		{ID: 9, Name: "中尉", Post: "营长", NeedPrestige: 40000, CityMax: 9},
		{ID: 10, Name: "上尉", Post: "团长", NeedPrestige: 52000, CityMax: 10},
		{ID: 11, Name: "大尉", Post: "团长", NeedPrestige: 66000, CityMax: 11},
		{ID: 12, Name: "少校", Post: "旅长", NeedPrestige: 82000, CityMax: 12},
		{ID: 13, Name: "中校", Post: "旅长", NeedPrestige: 100000, CityMax: 13},
		{ID: 14, Name: "上校", Post: "旅长", NeedPrestige: 120000, CityMax: 14},
		{ID: 15, Name: "大校", Post: "师长", NeedPrestige: 145000, CityMax: 15},
		{ID: 16, Name: "少将", Post: "师长", NeedPrestige: 175000, CityMax: 16},
		{ID: 17, Name: "中将", Post: "军长", NeedPrestige: 210000, CityMax: 17},
		{ID: 18, Name: "上将", Post: "军长", NeedPrestige: 250000, CityMax: 18},
		{ID: 19, Name: "大将", Post: "军长", NeedPrestige: 300000, CityMax: 19},
		{ID: 20, Name: "五星上将", Post: "司令", NeedPrestige: 400000, CityMax: 20},
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

// equipSet 取套装配置（nil = 该 id 不是套装）
func (c *ezfyConfigCache) equipSet(id int) *model.EzfyCfgEquipSet {
	if id <= 0 {
		return nil
	}
	if s, ok := c.equipSetMap[id]; ok {
		return &s
	}
	return nil
}

// poolOfficers 军官池里的**普通军官**（kind=1 且 recruit=1），按 id 升序。
//
// ★ 2026-09-22 用户要求：军校招募/刷新**从池子里抽**，不再纯随机生成。
func (c *ezfyConfigCache) poolOfficers() []model.EzfyCfgGeneral {
	out := []model.EzfyCfgGeneral{}
	for _, g := range c.generals {
		if g.Kind == 1 && g.Recruit != 0 {
			out = append(out, g)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// equipSets 全部套装配置（按 id 升序）
func (c *ezfyConfigCache) equipSets() []model.EzfyCfgEquipSet {
	out := []model.EzfyCfgEquipSet{}
	for _, s := range c.equipSetMap {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
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
