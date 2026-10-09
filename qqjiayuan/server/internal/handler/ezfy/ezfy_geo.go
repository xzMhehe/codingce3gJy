package ezfy

import (
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

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

// ezfyTerrainIsland 岛屿地形 id（1平原…7岛屿 8海洋 9沿海平原）
const ezfyTerrainIsland = 7

// ezfyIsSeaWildTerrain 该地形是否按「海野」处理：海底森林(8) + 岛屿(7)。
//
// ★ 2026-10-05 用户规则「岛屿也属于海野」——岛屿上的野地守军配置、采集系数、
//
//	战报命名、占领记录 wild_type 一律按海野口径（原来只有地形 8 算海野）。
func ezfyIsSeaWildTerrain(t int) bool {
	return t == ezfyTerrainSea || t == ezfyTerrainIsland
}

// ezfyWildTerrainDisplayName 野地/海野的**展示**名。
//
// ★★ 2026-10-05 用户纠正（三个概念必须分清，之前混了）：
//
//	① **海洋** = 海里**没有野地**的那种格子（地形 id 8）。「海洋就是海洋」。
//	② **海底森林** = 海里**有野地**的那一格 —— 它跟海洋不是一回事
//	   （用户原话：「只有 海底森林 是海底森林啊，海洋就是海洋啊」「海洋上不会有野地」）。
//	③ **岛屿** = 地形(7)。它**按海野玩法**处理（守军走海野配置、海军可打、占领后 wild_type=2），
//	   但**展示名仍是「岛屿」**，也不能叫海底森林
//	   （用户报的「海岛占领以后显示海底森林」就是这个 bug）。
//
// 规则：
//
//	地形 8 + 有野地 → "海底森林"
//	地形 8 + 无野地 → "海洋"      ← 纯海洋，绝不是海底森林
//	地形 7          → "岛屿"
//	其它地形        → 各自地形名（丘陵/沼泽/平原…）
//
// knownWild 由调用方声明「该坐标上是否**确定**有野地」：
//   - true  —— 已占领野地记录行（表里有行）/ 已判定为海野的详情 / 已匹配到野地配置的战斗目标；
//   - false —— 不确定（地图格、订单目标预览），此时函数按 `ezfyWildlandLevel` 判「纯海洋 vs 海底森林」
//     （与 `MapView` 里 `lvl == 0 → 海洋 / 否则海底森林` 完全同口径）。
//
// ⚠️ 采集产出另按**地形**区分（岛屿→钢铁、海底森林→石油），那是 `ezfyGatherResName` 的事，别在这里管。
// ⚠️ 新增任何「海野」相关的展示文案时一律走这个函数，
//
//	别自己写 `ezfyIsSeaWildTerrain(...) → "海底森林"`（那样纯海洋、以及岛屿都会变成海底森林）。
func ezfyWildTerrainDisplayName(x, y int, knownWild bool) string {
	if ezfyTerrainEx(x, y) != ezfyTerrainSea {
		return ezfyTerrainNameEx(x, y)
	}
	if knownWild || ezfyWildlandLevel(x, y) > 0 {
		return "海底森林"
	}
	return "海洋"
}

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

// ezfyActWildAt 取某格的活动野地配置（没有则 nil）
func ezfyActWildAt(x, y int) *model.EzfyActWild {
	c := &ezfyCfg
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.actWilds[ezfyTileKey(x, y)]
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

// ezfyEquipmentByName 按名称找装备配置（9 种珠宝也落在装备表里；野地类型「宝物掉落」按名配置用）
func (c *ezfyConfigCache) equipmentByName(name string) *model.EzfyCfgEquipment {
	for i := range c.equipments {
		e := c.equipments[i]
		if e.Name == name {
			return &e
		}
	}
	return nil
}

// ezfyHasNavalTroops 出征部队里是否含有海军兵种（兵种 type=1：驱逐舰/潜艇/战列舰/航母）。
//
// ★ 2026-10-02 用户规则：海军兵种只能用于海战，出征攻打陆城时需卡控提示。
func ezfyHasNavalTroops(units []ezfyUnitGroup) bool {
	for _, u := range units {
		if c := ezfyCfg.troop(u.TroopId); c != nil && c.Type == 1 {
			return true
		}
	}
	return false
}

// ezfyNavalTargetAllowed 海军兵种可出征的目标地形。
//
// ★ 2026-10-02 用户规则：海军只能用于「岛屿(7)/海底森林(海洋8)/沿海平原(9)」
// 以及建在沿海平原上的城市（沿海平原地形已含城市坐标）的出征，其它地形一律卡控。
// 用 ezfyTerrainEx 判定，管理端改地图地形后立即生效。
func ezfyNavalTargetAllowed(x, y int) bool {
	switch ezfyTerrainEx(x, y) {
	case 7, ezfyTerrainSea, ezfyTerrainCoastalPlain:
		return true
	}
	return false
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
	7: {"琥珀项链", "红宝石戒指", "祖母绿"},   // 岛屿 → 钢铁
	8: {"翡翠项链", "蓝宝石戒指", "祖母绿"},   // 海底森林 → 石油
}

// ezfyCollectibleTreasureNames 可采集/可用于军衔晋升的宝物名集合（9 种珠宝）。
//
// ★ 2026-09-28 只有「能采集的宝物」可以提交晋升军衔，
//
//	普通装备（黑色幽灵[徽章]、合金装甲等）虽然同属装备表，但不算宝物。
func ezfyCollectibleTreasureNames() map[string]bool {
	set := map[string]bool{}
	for _, names := range ezfyTerrainTreasureNames {
		for _, n := range names {
			set[n] = true
		}
	}
	return set
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

// ============ 城市自动迁移已停用（2026-10-05） ============
//
// 原「开机自动搬城」逻辑（ezfyMigrateSeaCities / ezfyNearestCoastalPlain，把落在海洋格
// 的城搬到最近的沿海平原）已**整体删除**，原因：
//
//	它依赖 ezfyTerrainEx 的输入（地图格子覆盖 ezfy_map_tile）。启动瞬间若瓦片读取失败/
//	抖动 → 地形整体翻转 → 一批本不该动的城被判成「海城」并整体位移，玩家看到的就是
//	「重新部署后城市坐标又变了」。这是同一 bug 反复复发的根源。
//
// 现在城市坐标**只由**玩家操作（迁城）或**显式运维命令**改动，启动流程绝不自动搬城。
// 需要一次性修数据时，跑运维命令：
//
//	./ezfymigrate --coastal --apply --yes   （把建有航海协会的城迁回沿海平原）
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

// ============ 军衔等级（2026-09-28 军衔不再自动跟随声望） ============
//
// 现在声望达标只是晋升前提，还要提交宝物（见 ezfy_rank_treasure.go）。
// 玩家实际军衔以 profile.Rank 为准；老玩家 Rank 还没写（=0）时回落声望推导，
// 保证旧账不缩水（已有军衔不被扣回去）。

// ezfyProfileRank 玩家当前军衔等级（1..N）
func ezfyProfileRank(p *model.EzfyProfile) int {
	if p.Rank > 0 {
		return p.Rank
	}
	return ezfyRankIndex(p.Prestige) + 1
}

// ezfyRankAt 军衔等级（1..N，等级 = 列表下标 + 1，列表按 id 升序）→ 配置
func ezfyRankAt(level int) model.EzfyCfgRank {
	ranks := ezfyCfg.rankList()
	if level < 1 {
		level = 1
	}
	if level > len(ranks) {
		level = len(ranks)
	}
	return ranks[level-1]
}

func ezfyRankNameAt(level int) string { return ezfyRankAt(level).Name }
func ezfyRankPostAt(level int) string { return ezfyRankAt(level).Post }

// ezfyRankCityMaxAt 该军衔等级下最多能拥有几座城（★ 军衔限制分城数量）
func ezfyRankCityMaxAt(level int) int {
	m := ezfyRankAt(level).CityMax
	if m <= 0 {
		m = 1
	}
	return m
}

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
	// 活动野地配置（key = x*100000+y），2026-09-29 地图管理「活动野地」tab 维护
	actWilds map[int64]*model.EzfyActWild
	// ★ 2026-10-05 地图瓦片表的「廉价指纹」（行数+最大id+最大updated_at）：
	//   周期刷新时先比它，只有变了才真去拉瓦片 —— 修「多机改地形不生效」。
	heavyFp uint64
	// 军衔配置（按等级 1..N 排序）
	ranks []model.EzfyCfgRank
	// 建筑数量上限（军事区/资源区分开，管理端可维护）
	limit model.EzfyCfgLimit
	// 二战聊天敏感词（独立维护页）
	words []model.EzfyWordFilter
	// 资源显示名配置（管理端可改名，前端读它替代写死「粮食/钢铁/…」）
	// 预组装成 ezfyResCfgOf 的返回结构，读请求零 SQL（热接口 /res-cfg、/view 都靠它）
	// 用 map[string]interface{} 而非 gin.H，避免本包混入 gin 依赖
	resCfg map[string]interface{}
	// 宝箱配置（上架宝箱 + 奖池；★ 2026-10-04 并入配置缓存，/chest 列表不再每次查这两张表
	// 也顺带消除「逐箱查奖池」的 N+1）
	chests     []model.EzfyCfgChest
	chestPools map[int][]model.EzfyCfgChestItem
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

// ezfyDefaultContinent 新玩家落地洲（读 ezfy_cfg_limit.default_continent，可在二战系统配置调整）
//
// ★ 2026-10-08 原来写死欧洲(1)，现在做成管理端可配。大洲 ID：1欧洲 2亚洲 3非洲 4北美洲 5南美洲 6大洋洲 7南极洲。
// 0 / 越界 / 未加载时回落默认欧洲（保证老玩家与新玩家行为不回退）。
func ezfyDefaultContinent() int {
	if n := ezfyCfg.limit.DefaultContinent; n >= 1 && n <= 7 {
		return n
	}
	return ezfyDefaultMoveContinent
}

// ezfyGatherMax 单次出征最多使用几个集结令（读 ezfy_cfg_limit.gather_max_per_order）
//
// ★ 「出征集结令上限后台管理系统可维护」，默认 99（线上现值）。
// 0 或未配置时回落默认值（集结令上限为 0 无意义 —— 等于禁用了这个道具）。
func ezfyGatherMax() int {
	if n := ezfyCfg.limit.GatherMaxPerOrder; n > 0 {
		return n
	}
	return ezfyGatherMaxDefault
}

// ezfySellPriceMax 挂单出售单价上限（黄金/单位）
//
// ★ 2026-09-28 「挂单出售按 1:100 卡控单价：卖 1 粮食价格不能超过 100，
//
//	数量随意（1/2/50/60），比例在二战系统配置可灵活配置」。
//
// 0 或未配置时回落默认 100。
func ezfySellPriceMax() int {
	if n := ezfyCfg.limit.SellPriceMax; n > 0 {
		return n
	}
	return 100
}

// ezfyRansomCost 赎城所需钻石（读 ezfy_cfg_limit.ransom_cost，管理端「建筑上限配置」可维护）
//
// ★ 2026-10-07 赎城功能。发起即按此金额扣原主人钻石（押金），同意后转给占领方，
//   拒绝/撤销退回。0 或未配置时回落默认 500（0 无意义 = 禁止赎城）。
func ezfyRansomCost() int64 {
	if n := ezfyCfg.limit.RansomCost; n > 0 {
		return int64(n)
	}
	return ezfyRansomCostDefault
}

// ezfySysSellRatio 向系统出售资源的回收比例（每 100 单位 → 黄金）。
//
// ★ 2026-09-30 「玩家可向系统出售资源获得黄金，比例可配置」：
//
//	返回 es_type(1粮/2钢/3油/4稀) 对应的「每100单位黄金」数，0 或未配置时回落各自默认。
//	玩家实得黄金还要在换算后再扣 10% 手续费（见 ExchangeSysSell）。
func ezfySysSellRatio(esType int) int {
	lim := ezfyCfg.limit
	switch esType {
	case 1:
		if lim.SysSellFood > 0 {
			return lim.SysSellFood
		}
		return 10
	case 2:
		if lim.SysSellSteel > 0 {
			return lim.SysSellSteel
		}
		return 10
	case 3:
		if lim.SysSellOil > 0 {
			return lim.SysSellOil
		}
		return 20
	case 4:
		if lim.SysSellRare > 0 {
			return lim.SysSellRare
		}
		return 25
	}
	return 0
}

// ezfyOfficerCapPerMil 出征军官每 1 点军事累加的出征上限（二战系统配置可调，默认 2000）
func ezfyOfficerCapPerMil() int {
	if n := ezfyCfg.limit.OfficerCapPerMilitary; n > 0 {
		return n
	}
	return 2000
}

// ezfyOfficerSpeedPerMil 出征军官每 1 点军事加成的出征速度（%，二战系统配置可调，默认 0.1）
func ezfyOfficerSpeedPerMil() float64 {
	if n := ezfyCfg.limit.OfficerSpeedPerMilitary; n > 0 {
		return n
	}
	return 0.1
}

// ezfyDispatchPeriod 常驻采集结算一期时长（毫秒）。
// ★ 2026-09-24 「采集 12 小时才有宝物 → 更短且可配置」：
//
//	读管理端「建筑上限/系统配置」ezfy_cfg_limit.dispatch_period_h（小时），默认 1（线上现值）。
func ezfyDispatchPeriod() int64 {
	if h := ezfyCfg.limit.DispatchPeriodH; h > 0 {
		return int64(h) * 3600 * 1000
	}
	return 1 * 3600 * 1000
}

// ezfyMarchSpeedBonus 出征速度加成（百分比，0 = 无加成）。
// ★ 2026-09-24 「节假日让玩家队伍走快点」：管理端可配。
//
//	实际行军时间 = 原时间 × 100/(100+加成)；默认 0（加成 > 0 才生效，负值/未配置按 0 处理）。
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
	ezfyConquerFeelingsDef  = 5  // 征服单次最多扣民心（默认 5）
	ezfyLootFeelingsDef     = 3  // 掠夺每次扣民心（默认 3）
	ezfyRansomCostDefault   = 500 // 赎城所需钻石（默认 500）
	ezfyOfficerSalaryDef    = 20 // 军官工资：每级每小时黄金（★ 2026-09-26 由 100 改成 20）
	ezfyWoundHealDivisorDef = 50 // 恢复伤兵黄金 = 兵种总造价 / 该值（默认 50）
	// ★ 商城单次购买数量上限（默认 99）
	ezfyMallBuyMaxDef = 99
	// ★ 单城兵力上限：默认 50 亿。2026-09-23 线上「负数兵力」事故后新增 ——
	//   训练 / 伤兵恢复 / addTroop 三处共用，防止兵力累加溢出成负数。
	//   2026-09-23 「10 亿太少，上调到 50 亿」。
	ezfyTroopMaxDef = int64(5000000000)
	// ★ 伤兵在营存活天数：默认 3 天，超时未恢复自动消失。
	ezfyWoundExpireDaysDef = 3
	// ★ 侦查成功率封顶（百分比，默认 95）：0 无意义 → 回落该值。
	//   配合下面两个曲线参数：成功率 = cap × 1/(1 + e^-k·(log10(n)-x0))，n = 侦察机数。
	//   k/x0 为曲线陡度/中点（10 架≈11%、1 千架≈48%、1 万≈69%、5 万≈80%、10 万≈84%）。
	ezfyReconSuccessPctDef = 95.0
	// ★ 侦查成功率的数量级曲线参数（梯度陡、按 10/1千/1万/5万/10万 拉开）。
	//   k 越大越陡（接近阶跃）；x0 越大曲线越右移（同数量级成功率越低）。
	ezfyReconK  = 1.0
	ezfyReconX0 = 3.0
	// ★ 资源数值安全上限：任何路径写入资源都不得超过它（约 1 万亿）。
	//   远小于 int64 上限，仅用于兜底防溢出；游戏内实际生效的仍是各城「仓储上限」。
	ezfyResSafeMax = int64(1000000000000)

	// ★ 2026-09-28 用户规则：安抚每次花 5 万黄金、民怨 −2、民心 +1、15 分钟一次。
	ezfyPlacateGoldDef      = int64(50000) // 安抚花费黄金（默认 5 万）
	ezfyPlacateGrievanceDef = 2            // 安抚降低民怨（默认 2）
	ezfyPlacateFeelingsDef  = 1            // 安抚提升民心（默认 1）
	ezfyPlacateCooldownDef  = 15           // 安抚冷却分钟数（默认 15）
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

// ============ 资源最大值（二战系统配置，默认 21 亿）============
//
// ★ 2026-09-27 「资源产量也做成累加」：**所有**资源统一只受「资源最大值」这一个硬上限，
//
//	（默认 21 亿）。产量 / 其它一切获取方式都无条件累加到该值为止，不再被仓储上限卡住。
//
// 规则：
//
//	① 资源**产量**（calcResource 自动产出）**同样无条件累加**，只在此处收敛 —— 不再看仓储上限；
//	② 其它所有获取方式（野地/寇城战利品、掠夺/征服入账、采集返航、运输、派遣、
//	   资源包、签到福利/任务奖励、交易行成交、军团商城…）一律累加，同样只在此处停；
//	③ 老数据已经超过该值的**不会被拉低**（表达式里用 GREATEST 保住较大值）。
const ezfyResMaxDef = int64(2100000000) // 默认 21 亿

// ezfyResMaxOf 取某项资源的「资源最大值」配置（res ∈ food/steel/oil/rare/gold）
//
// 0 / 未配置 → 回落 21 亿；再兜一层数据库安全上限，任何配置都写不出溢出值。
func ezfyResMaxOf(res string) int64 {
	var v int64
	if ezfyCfg.ready() {
		switch res {
		case "food":
			v = ezfyCfg.limit.ResMaxFood
		case "steel":
			v = ezfyCfg.limit.ResMaxSteel
		case "oil":
			v = ezfyCfg.limit.ResMaxOil
		case "rare":
			v = ezfyCfg.limit.ResMaxRare
		case "gold":
			v = ezfyCfg.limit.ResMaxGold
		}
	}
	if v <= 0 || v > ezfyResSafeMax {
		if v > ezfyResSafeMax {
			return ezfyResSafeMax
		}
		return ezfyResMaxDef
	}
	return v
}

// ezfyResAddExpr 生成「入库累加」SQL：原子累加，硬上限 = 该资源的「资源最大值」（默认 21 亿）。
//
// ★★ 2026-09-30 修复「资源能累加超过资源最大值」：
//
//	2026-09-28 为修「[一键收获]/停止 资源没有入城市」事故，把封顶从「资源最大值」改成了
//	「ezfyResSafeMax(1 万亿)」—— 于是玩家/管理端发资源都能一路累加到 1 万亿，超过 21 亿上限。
//	现在恢复「每项资源唯一硬上限 = 资源最大值」：
//	  - 正数累加 → LEAST(资源最大值, col + n)，封顶不超上限；
//	  - 老数据已超上限的城不拉低（GREATEST 保住原值），但也不再增长；
//	  - 负数扣减 → 正常减少，最低 0（不能因为 GREATEST 把扣减吞掉）。
func ezfyResAddExpr(res string, n int64) clause.Expr {
	max := ezfyResMaxOf(res)
	if n < 0 {
		return gorm.Expr("GREATEST(0, `"+res+"` + ?)", n)
	}
	return gorm.Expr("GREATEST(`"+res+"`, LEAST(?, `"+res+"` + ?))", max, n)
}

// ezfyAddResMax 内存版「入库累加」：硬上限 = 该资源的「资源最大值」（默认 21 亿）。
//
// ★ 与 ezfyResAddExpr 严格同口径（2026-09-30 起）：
//   - 正数累加 → min(资源最大值, 现值 + 增量)，封顶不超上限；
//   - delta=0（仅黄金产量结算使用）→ 现值超过资源最大值时拉回上限，老数据超限自动收敛；
//   - 负数扣减 → 正常减少，最低 0。
//
// 供不便走 SQL 表达式的发放路径使用。
func ezfyAddResMax(res string, cur, delta int64) int64 {
	max := ezfyResMaxOf(res)
	cur = ezfyClampRes(cur)
	if delta == 0 {
		// ★ 2026-10-01 修复「黄金产量超过资源最大值」：
		//   delta=0 的唯一调用点是 calcResource 的黄金产量结算（gold 先算好净增量再以 0 传入夹取）。
		//   原实现直接 return cur，**完全跳过资源最大值** → 黄金产量每小时累加从未被配置上限截断，
		//   一路涨到 77.67 亿（配置仅 61 亿）。这里 cur 超过资源最大值时拉回上限：
		//   老数据超限的城在下次懒结算自动收敛到配置值，与 food/steel/oil/rare 产量路径口径一致。
		if cur > max {
			return max
		}
		return cur
	}
	if delta < 0 {
		c := cur + delta
		if c < 0 {
			c = 0
		}
		return c
	}
	if cur >= max {
		return cur // 已到/超过资源最大值：不再增长（也不拉低）
	}
	return ezfySafeAdd(cur, delta, max)
}

// ezfyAtResMax 该城某项资源是否已到达「资源最大值」（= 满了，再采集也入不了库）。
//
// ★ 用途：采集 / 收获是玩家**主动发起、要等时间**的操作，到顶时必须在发起前就告知，
// 否则玩家等完一轮才发现资源没进账 —— 这正是「资源没有入城市」的体感来源之一。
func (h *EzfyHandler) ezfyAtResMax(cityID int64) bool {
	if cityID <= 0 {
		return false
	}
	var c model.EzfyCity
	if err := h.DB.Select("food", "steel", "oil", "rare", "gold").
		First(&c, cityID).Error; err != nil {
		return false
	}
	// 五项全满才算满 —— 采集产出五项都有，只要还有一项没满，收获就仍有意义。
	return c.Food >= ezfyResMaxOf("food") &&
		c.Steel >= ezfyResMaxOf("steel") &&
		c.Oil >= ezfyResMaxOf("oil") &&
		c.Rare >= ezfyResMaxOf("rare") &&
		c.Gold >= ezfyResMaxOf("gold")
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

// ezfyPlacateGoldCost 安抚花费的黄金（默认 5 万）
func ezfyPlacateGoldCost() int64 {
	if v := ezfyCfg.limit.PlacateGold; v > 0 {
		return v
	}
	return ezfyPlacateGoldDef
}

// ezfyPlacateGrievanceDown 安抚降低的民怨点数（默认 2）
func ezfyPlacateGrievanceDown() int {
	return ezfyLimitOr(ezfyCfg.limit.PlacateGrievance, ezfyPlacateGrievanceDef)
}

// ezfyPlacateFeelingsUp 安抚提升的民心点数（默认 1）
func ezfyPlacateFeelingsUp() int {
	return ezfyLimitOr(ezfyCfg.limit.PlacateFeelings, ezfyPlacateFeelingsDef)
}

// ezfyPlacateCooldownMinutes 安抚冷却分钟数（默认 15）
func ezfyPlacateCooldownMinutes() int {
	if v := ezfyCfg.limit.PlacateCooldownMin; v > 0 {
		return v
	}
	return ezfyPlacateCooldownDef
}

// ezfyPlacateCooldownMs 安抚冷却毫秒数
func ezfyPlacateCooldownMs() int64 {
	return int64(ezfyPlacateCooldownMinutes()) * 60000
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
// ★ 2026-09-23 伤兵恢复黄金也有「折扣率数」，放管理端「二战系统配置」配，
//
//	节假日调低 = 恢复便宜。★ 2026-09-24 修正：默认 100 = 现在的正常值，0/负数 → 回落 100。
func ezfyWoundHealRate() float64 {
	v := ezfyCfg.limit.WoundHealRate
	if v <= 0 {
		v = 100
	}
	return v / 100
}

// ezfyMallBuyMaxCfg 商城单次购买数量上限（下限恒为 1，默认 99）
//
// ★ 「商城购买卡控改成可配置的」（2026-09-26 线上现值 = 99）。
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
	// ★ 2026-09-26 「召集人口那里加两个开关」
	ezfyHousePopLimitDef = 1  // 民居容量限制：默认开（民居容量决定人口上限）
	ezfyConveneFlexDef   = 1  // 召集人口灵活配置：默认开（召集可突破民居上限）
	ezfyWildMultDef      = 10 // 野地兵力倍数：默认 10
	// ★ 2026-09-25 用户反馈「野地打完获得的资源太少」→ 野地战利品资源倍率，默认 10
	ezfyWildResMultDef = 10
	// ★ 2026-09-25 「采集资源倍率也加到系统管理里」→ 常驻采集产出资源倍率，默认 10
	ezfyGatherResMultDef = 10
	// ★ 2026-09-28 采集军官后勤属性加成率倍率 / 市长产量加成倍率，默认 1
	ezfyOfficerGatherMultDef = 1
	ezfyMayorGainMultDef     = 1
	// ★ 2026-09-28 采集等级成长幂次，默认 1.3（高等级野地产出加速型增长）
	ezfyGatherLevelPowDef = 1.3
	// ★ 2026-09-28 海野采集系数，默认 1.5（海野基础 3 × 1.5 = 4.5，比同级陆野 4 更高）
	ezfyGatherSeaMultDef = 1.5
	// ★ 2026-09-28 军校刷新周期模式，默认按小时（2）；1 = 按天
	ezfyRecruitCycleHourlyDef = 2
	// ★ 2026-10-09 原「战斗掉落·中级/高级/特殊宝物概率 + 活动野地掉宝概率」四个全局配置已**删除**
	//   （用户要求「掉落 都走手动配置的」）：野地/寇城掉落走「地图管理 → 野地类型」里的
	//   宝物/道具掉落配置（`wildlandConfigLoot`），活动野地走 `EzfyActWild.Treasures`。
	//   ⚠️ `ezfy_cfg_limit` 上的 drop_t2/drop_t3/drop_t4/drop_act_pct 四个**列保留**（不动历史数据），
	//   只是代码不再读写。
)

// ezfyMarchCapOn 出征是否受「兵力上限」限制（关 = 不限兵力）
//
// ★ 「再加个出征上限开关，默认开，关闭出征没有上限」。
//
//	关掉后 ezfyOrderTroopCap() 会返回 unlimited=true，出征校验与出征页提示一起放开。
func ezfyMarchCapOn() bool {
	if !ezfyCfg.ready() {
		return ezfyMarchCapDef != 0
	}
	return ezfyCfg.limit.MarchCapOn != 0
}

// ezfyHousePopLimitOn 民居容量是否限制人口上限（关 = 民居不限制人口，人口可无限增长）
//
// ★ 2026-09-26 「召集人口那里加个民居容量限制开关，默认开」。
//
//	关掉后 calcResource 里的人口自然增长不再按 pop_max 封顶（pop_max 仍照常计算/展示）。
func ezfyHousePopLimitOn() bool {
	if !ezfyCfg.ready() {
		return ezfyHousePopLimitDef != 0
	}
	return ezfyCfg.limit.HousePopLimitOn != 0
}

// ezfyConveneFlexOn 召集人口是否可突破民居上限（关 = 召集同样受民居容量约束）
//
// ★ 2026-09-26 「召集人口灵活配置，默认开（现有行为：可突破上限）」。
// 注意：只有在「民居容量限制」也开着时，民居上限才存在；两者都开时才需要在 Convene 里卡上限。
func ezfyConveneFlexOn() bool {
	if !ezfyCfg.ready() {
		return ezfyConveneFlexDef != 0
	}
	return ezfyCfg.limit.ConveneFlexibleOn != 0
}

// ============ 召集人口：消耗粮食 / 获得人口 ============
//
// ★ 2026-09-26 「花费 10万粮食 召集 10万人口也要能配置，现在是写死的」。
// 原来写死在 ezfy.go 的 const（ezfyConveneFoodCost / ezfyConvenePopGain），现迁到
// ezfy_cfg_limit（convene_food_cost / convene_pop_gain），管理端「二战系统配置」可维护。
// 0 / 未配置无意义 → 回落默认 10 万（seed 用 addLimitCol 只回填 NULL，不覆盖管理端的值）。
const (
	ezfyConveneFoodCostDef = 100000 // 召集一次消耗粮食，默认 10 万
	ezfyConvenePopGainDef  = 100000 // 召集一次获得人口，默认 10 万
	// ★ 2026-09-26 「玩家城市人口不能超过配置的人口上限，超过则禁止召集」：
	//   全局硬性人口上限，默认 0 = 不限。⚠️ 0 是有效值（不限），不能用 ezfyLimitOr 兜底。
	ezfyConvenePopMaxDef = 0
)

// ezfyConveneFoodCostCfg 召集一次消耗的粮食（默认 10 万）
func ezfyConveneFoodCostCfg() int64 {
	return int64(ezfyLimitOr(ezfyCfg.limit.ConveneFoodCost, ezfyConveneFoodCostDef))
}

// ezfyConvenePopGainCfg 召集一次获得的人口（默认 10 万）
func ezfyConvenePopGainCfg() int64 {
	return int64(ezfyLimitOr(ezfyCfg.limit.ConvenePopGain, ezfyConvenePopGainDef))
}

// ezfyConvenePopMaxCfg 召集硬性人口上限（0 = 不限）。
//
// ⚠️ 0 是有效值（= 不限上限），直接返回配置值，不能用 ezfyLimitOr（那会把它兜底成别的数）。
func ezfyConvenePopMaxCfg() int64 {
	if !ezfyCfg.ready() {
		return ezfyConvenePopMaxDef
	}
	return int64(ezfyCfg.limit.ConvenePopMax)
}

// ============ 军官升星配置（2026-09-22 ，2026-09-23 按简化）============

const (
	// ★ 2026-09-23 「军官升星做得太复杂，优化简约点」：
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
	// 招生简章出五星军官概率默认值（1 = 1%，默认 1 倍率）
	ezfyRecruitFiveStarDef = 1
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

// ezfyRecruitFiveStarRate 招生简章出五星军官概率（%）：默认 1%，范围 1~100。
// 100 = 100% 必刷出 5 星军官。
func ezfyRecruitFiveStarRate() int {
	return clampInt(ezfyLimitOr(ezfyCfg.limit.RecruitFiveStarRate, ezfyRecruitFiveStarDef), 1, 100)
}

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
// ★ 2026-09-23 黄金消耗太多，价格倍率放管理端「二战系统配置」配，
//
//	节假日想便宜点就把倍率调低（如 50 = 半价、10 = 一折）。
//	★ 2026-09-24 用户修正：默认值 100 才是正常值（而不是 1），设置 0.01 时仍觉得贵、
//	说明要按「百分比」理解 —— 100 = 现在的正常消耗。0/负数无意义 → 回落 100。
func ezfySpeedTrainRate() float64 {
	v := ezfyCfg.limit.SpeedTrainRate
	if v <= 0 {
		v = 100
	}
	return v / 100
}

// ezfyWarRequireOn 是否要求「先宣战才能掠夺/征服别人城市」
//
// ★ 「加一个宣战功能开关，默认开启：开启 = 玩家之间需要宣战；
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

// ezfyWildTroopMult 野地/海野/寇城守军兵力倍数（默认 10；0 或负数无意义 → 回落 10）
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

// ezfyWildResMult 野地/海野/寇城**战斗胜利后的战利品**资源倍率（默认 10；0 或负数无意义 → 回落 10）
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

// ezfyResProdMultDef 城市资源产量倍率默认值
const ezfyResProdMultDef = 1.0

// ezfyResProdMult 城市每小时「资源」（粮/钢/油/稀矿）产量的整体倍率
//
// ★ 2026-09-26 「二战加个产量加成倍率，默认 1，可以调整 >= 0 的任意数量」。
// ★ 2026-10-05 拆开：本倍率只作用于资源，黄金产量走 ezfyGoldProdMult。
//
//	⚠️ **0 是合法值**（= 产量归零），不是「未配置」——
//	所以这里**故意不做 `<= 0 就回落默认`**（那套是 `ezfyWildResMult` 的口径，不适用于倍率）。
//	「未配置」由 seed 补列时**只把 NULL 回填成 1** 来兜，不靠读取端猜。
//	⚠️ 两个收敛点必须同时改：`calcResource`（实际入库）与 `getResourceCalc`（详情页展示），
//	否则会出现「详情页显示 1 万、实际只入库 100」。
func ezfyResProdMult() float64 {
	if !ezfyCfg.ready() {
		return ezfyResProdMultDef
	}
	return ezfyCfg.limit.ResProdMult
}

// ezfyScaleResByProdMult 资源产量按倍率缩放（倍率 1 时原样返回，避免无谓的浮点误差）
func ezfyScaleResByProdMult(n int64) int64 {
	m := ezfyResProdMult()
	if m == 1 {
		return n
	}
	if m <= 0 {
		return 0
	}
	return int64(float64(n) * m)
}

// ezfyGoldProdMultDef 黄金产量倍率默认值
const ezfyGoldProdMultDef = 1.0

// ezfyGoldProdMult 城市每小时「黄金」产量的整体倍率（★ 2026-10-05 与资源倍率拆开）
//
//	口径与 ezfyResProdMult 一致：0 合法（黄金产量归零）；NULL 由 seed 回填 1。
func ezfyGoldProdMult() float64 {
	if !ezfyCfg.ready() {
		return ezfyGoldProdMultDef
	}
	return ezfyCfg.limit.GoldProdMult
}

// ezfyScaleGoldByProdMult 黄金产量按倍率缩放（倍率 1 时原样返回）
func ezfyScaleGoldByProdMult(n int64) int64 {
	m := ezfyGoldProdMult()
	if m == 1 {
		return n
	}
	if m <= 0 {
		return 0
	}
	return int64(float64(n) * m)
}

// ezfyGatherResMult 常驻采集产出资源倍率（默认 10；0 或负数无意义 → 回落 10）
//
// ★ 2026-09-25 「采集资源倍率也加到系统管理里」→ 管理端「二战系统配置」可调。
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

// ezfyOfficerGatherMult 采集「军官后勤属性」加成率倍率（默认 1；0 / 负 / NULL → 回落 1）
//
// ★ 2026-09-28 「采集后勤加成率可调」：dispatchGatherYield 里
// gainPct = 100 + floor(后勤 × 本倍率)，封顶 200。
func ezfyOfficerGatherMult() float64 {
	if !ezfyCfg.ready() {
		return ezfyOfficerGatherMultDef
	}
	if m := ezfyCfg.limit.OfficerGatherMult; m > 0 {
		return m
	}
	return ezfyOfficerGatherMultDef
}

// ezfyMayorGainMult 市长产量加成倍率（默认 1；NULL → 回落 1；0 合法 = 关闭市长加成）
//
// ★ 2026-09-28 「市长加成整体可调」：mayorBonusPct 结果 × 本倍率。
func ezfyMayorGainMult() float64 {
	if !ezfyCfg.ready() {
		return ezfyMayorGainMultDef
	}
	if m := ezfyCfg.limit.MayorGainMult; m < 0 {
		return ezfyMayorGainMultDef
	}
	return ezfyCfg.limit.MayorGainMult
}

// ezfyGatherLevelPow 采集等级成长幂次（默认 1.3；0 / 负 / NULL → 回落 1.3）
//
// ★ 2026-09-28 「越高级的野地采集越多」：dispatchGatherYield 里
// per = 800 × (野地等级 ^ 本幂次)。1.0 = 纯线性；>1 = 高等级加速增长。
func ezfyGatherLevelPow() float64 {
	if !ezfyCfg.ready() {
		return ezfyGatherLevelPowDef
	}
	if p := ezfyCfg.limit.GatherLevelPow; p > 0 {
		return p
	}
	return ezfyGatherLevelPowDef
}

// ezfyGatherSeaMult 海野采集系数（默认 1.5；0 / 负 / NULL → 回落 1.5）
func ezfyGatherSeaMult() float64 {
	if !ezfyCfg.ready() {
		return ezfyGatherSeaMultDef
	}
	if m := ezfyCfg.limit.GatherSeaMult; m > 0 {
		return m
	}
	return ezfyGatherSeaMultDef
}

// ezfyRecruitCycleHourly 军校刷新周期是否按小时（默认按小时；1=按天 / 2=按小时，非法回落按小时）
//
// ★ 2026-09-28 刷新周期可在二战系统配置切换按天/按小时，默认按小时。
func ezfyRecruitCycleHourly() bool {
	if !ezfyCfg.ready() {
		return true
	}
	return ezfyCfg.limit.RecruitCycleMode != 1
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
	// ★ 2026-10-04 性能：快速路径**无锁**读 loaded（与既有无锁访问器同一约定）。
	//   原来这里 RLock 查 loaded，而 reload（每 30s 周期 + 管理端保存）持**写锁**重读配置，
	//   期间所有请求的 cfgs() 都会阻塞 —— 线上表现为「每隔 30s 所有接口一起卡一下」。
	//   改无锁后 reload 期间请求直接跳过加载（读旧配置，行为不变）。
	if c.loaded {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.loaded {
		return
	}
	c.loadLocked(db, false)
	c.loaded = true
}

// reload 强制重新读库。管理端在「建筑总配置 / 兵种配置 / 名将 / 技能 / 装备」里
// 改了参数后调用它，改动立刻生效，不用重启进程。
//
// ★ 2026-10-03 双机共享 RDS 后，这台苹果的是「谁改了配置」只有命中的那台进程知道，
//
//	本机用 cfgsReload 显式刷新；另一台靠 ezfyPeriodicReload 周期刷新收敛。
//	海岸索引只依赖「地图格子覆盖」(ezfy_map_tile 的 Terrain)，其它配置表不改变外形，
//	所以只有当瓦片覆盖实际变化时才重建索引 —— 避免周期刷新把 500×500 全图扫描扛下来。
//
// ★ 2026-10-04 skipHeavy=true（周期刷新）：只刷小配置表，跳过 25 万行地图瓦片表与
//
//	活动野地表 —— 这两张表是管理端**低频改动**，仍由 cfgsReload 全量刷新收敛。
func (c *ezfyConfigCache) reload(db *gorm.DB, skipHeavy ...bool) {
	skip := len(skipHeavy) > 0 && skipHeavy[0]
	// ★★ 2026-10-05：周期刷新改成「先算廉价指纹，只有真变了才拉地图瓦片」。
	//   原实现是**无条件跳过**地图瓦片 —— 双机部署下，管理端改地形只对处理请求的那台生效，
	//   另一台永远看不到（与活动野地同一个 bug）。现在指纹一变就跟着刷新，多机自动收敛；
	//   没变时连读都不读（指纹是聚合查询，比拉 4.8 万行便宜几个数量级）。
	loadTiles := true
	if skip {
		fp := ezfyHeavyFingerprint(db)
		c.mu.RLock()
		changed := !c.loaded || c.heavyFp != fp
		c.mu.RUnlock()
		loadTiles = changed
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	oldTileFp := ezfyTilesFingerprint(c.tiles)
	c.loadLocked(db, loadTiles)
	c.loaded = true
	if !loadTiles {
		return // 瓦片未动，无需指纹比对/海岸索引重建
	}
	c.heavyFp = ezfyHeavyFingerprint(db)
	if ezfyTilesFingerprint(c.tiles) != oldTileFp {
		// ★ 地图格子覆盖配置可能改了地形 → 沿海平原索引必须重建，
		//   否则管理端新配的沿海/陆地不被迁城逻辑看到。
		ezfyInvalidateCoastalIndex()
	}
}

// ezfyHeavyFingerprint 给「可能被周期刷新跳过」的重表算一个**廉价指纹**：
//
//	行数 + 最大 id + 最大 updated_at。新增、删除、修改都会让它变化。
//	用途：多机部署时判断另一台机器有没有改过这些表，变了才真去拉（见 reload）。
//
// ⚠️ updated_at 精度是秒级 —— 同一秒内的两次改动可能漏检（管理端手工操作，可忽略）。
// ⚠️ 只放**行数不大、但全表拉取代价高**的表；活动野地只有几行，直接每次都读，不进指纹。
func ezfyHeavyFingerprint(db *gorm.DB) uint64 {
	var t struct {
		N  int64
		Mx int64
		Ts int64
	}
	// ★ 2026-10-07 修复：MySQL 的 UNIX_TIMESTAMP() 返回 **DECIMAL（带小数，如 1790552181.446）**，
	//   驱动以字符串/[]byte 返回 → Scan 进 int64 直接报
	//   `converting driver.Value type []uint8 ("...") to a int64`，于是每 30 秒刷一条错误日志。
	//   套一层 CAST(... AS SIGNED) 取整即可 —— 指纹只要求「表变了就不同」，秒级精度完全够用。
	db.Raw("SELECT COUNT(*) AS n, IFNULL(MAX(id),0) AS mx, IFNULL(CAST(UNIX_TIMESTAMP(MAX(updated_at)) AS SIGNED),0) AS ts FROM ezfy_map_tile").Scan(&t)
	return uint64(t.N)*1000003 + uint64(t.Mx)*7 + uint64(t.Ts)*31
}

// ezfyTilesFingerprint 对「地图格子覆盖」集合算一个指纹，用于判断这会刷新是否动了地形。
// 只取影响沿海平原判定的字段(坐标 + Terrain)，避免轻微配置漂移触发海岸索引全图重建。
func ezfyTilesFingerprint(m map[int64]model.EzfyMapTile) uint64 {
	h := uint64(len(m)) * 104729
	for k, t := range m {
		h = h*31 + uint64(k) + uint64(t.Terrain)*131
	}
	return h
}

// ezfyStartConfigReloader 保证周期刷新只启动一次（按请求懒启动，拿到 db 引用）。
var ezfyStartConfigReloader sync.Once

// ezfyPeriodicReload 每 30s 从共享 RDS 重读二战配置，让两台服务器的进程内缓存收敛。
// ★ 2026-10-04 skipHeavy=true：只刷小配置表（几十~几百行），跳过地图瓦片/活动野地
//
//	两张重表 —— 原来每 30s 全图扫描 + 持写锁，期间全站请求被拖住（用户反馈「很卡」）。
//	地图/活动野地是管理端低频改动，靠管理端保存时的 cfgsReload 全量收敛。
func ezfyPeriodicReload(db *gorm.DB) {
	for range time.Tick(30 * time.Second) {
		ezfyCfg.reload(db, true)
	}
}

// loadLocked 真正干活的部分，调用方必须已持有写锁。
// ★ 2026-10-04 新增 skipHeavy：周期刷新时跳过「地图瓦片 / 活动野地」两张重表
//
//	（25 万行全图扫描跨 WAN RDS 要几百毫秒~秒级，且期间持写锁拖住全站），
//	这两张表只由管理端低频改动，全量刷新的 cfgsReload 仍会读取。
func (c *ezfyConfigCache) loadLocked(db *gorm.DB, loadTiles bool) {
	buildings := map[int]model.EzfyCfgBuilding{}
	buildingLvls := map[int]map[int]model.EzfyCfgBuildingLevel{}
	troops := map[int]model.EzfyCfgTroop{}
	techs := map[int]model.EzfyCfgTech{}
	techLvls := map[int]map[int]model.EzfyCfgTechLevel{}
	wildlands := map[int]map[int]model.EzfyCfgWildland{}
	items := map[int]model.EzfyCfgItem{}
	generals := map[int]model.EzfyCfgGeneral{}
	skills := map[int]model.EzfyCfgSkill{}
	skillByName := map[string]int{}
	equipments := map[int]model.EzfyCfgEquipment{}
	equipSetMap := map[int]model.EzfyCfgEquipSet{}
	buildingByName := map[string]int{}
	techByName := map[string]int{}

	var bs []model.EzfyCfgBuilding
	db.Find(&bs)
	for _, b := range bs {
		buildings[b.ID] = b
		buildingByName[b.Name] = b.ID
	}
	var bls []model.EzfyCfgBuildingLevel
	db.Find(&bls)
	for _, l := range bls {
		m, ok := buildingLvls[l.BuildingId]
		if !ok {
			m = map[int]model.EzfyCfgBuildingLevel{}
			buildingLvls[l.BuildingId] = m
		}
		m[l.Level] = l
	}
	var ts []model.EzfyCfgTroop
	db.Find(&ts)
	for _, t := range ts {
		troops[t.ID] = t
	}
	var tcs []model.EzfyCfgTech
	db.Find(&tcs)
	for _, t := range tcs {
		techs[t.ID] = t
		techByName[t.Name] = t.ID
	}
	var tls []model.EzfyCfgTechLevel
	db.Find(&tls)
	for _, l := range tls {
		m, ok := techLvls[l.TechId]
		if !ok {
			m = map[int]model.EzfyCfgTechLevel{}
			techLvls[l.TechId] = m
		}
		m[l.Level] = l
	}
	var ws []model.EzfyCfgWildland
	db.Find(&ws)
	for _, w := range ws {
		m, ok := wildlands[w.Type]
		if !ok {
			m = map[int]model.EzfyCfgWildland{}
			wildlands[w.Type] = m
		}
		m[w.Level] = w
	}
	var its []model.EzfyCfgItem
	db.Find(&its)
	for _, it := range its {
		items[it.ID] = it
	}
	var gens []model.EzfyCfgGeneral
	db.Find(&gens)
	for _, g := range gens {
		generals[g.ID] = g
	}
	var sks []model.EzfyCfgSkill
	db.Find(&sks)
	for _, s := range sks {
		skills[s.ID] = s
		skillByName[s.Name] = s.ID
	}
	var eqs []model.EzfyCfgEquipment
	db.Find(&eqs)
	for _, e := range eqs {
		equipments[e.ID] = e
	}
	var esets []model.EzfyCfgEquipSet
	db.Find(&esets)
	for _, s := range esets {
		equipSetMap[s.ID] = s
	}

	// ★★ 2026-10-07 修复「活动野地战斗 0 回合」：
	//   原来这里是先把下面 14 个 map **清空**、再查库**逐条填充** —— 而读函数
	//   （troop/general/skill/equipment/tech/...）都没有加锁，reload 窗口内
	//   （查库跨 WAN RDS 几十~几百毫秒）并发请求会读到**空 map** →
	//   兵种配置查不到 → 战斗双方参战单位被过滤空 → 判 0 回合（线上 449 条战报：
	//   战报显示攻方有兵，实际参战兵力为 0，守方零损失）。
	//   改为：先在局部 map 上构建完整数据，最后**一次性替换字段** —— 读者要么看到
	//   旧值、要么看到新值，绝不会看到空 map；读取端无需加锁、也没有锁阻塞。
	//   ★ 以后新增「清空 + 逐条填充」式缓存，也必须走这个模式。
	c.buildings = buildings
	c.buildingLvls = buildingLvls
	c.troops = troops
	c.techs = techs
	c.techLvls = techLvls
	c.wildlands = wildlands
	c.items = items
	c.generals = generals
	c.skills = skills
	c.skillByName = skillByName
	c.equipments = equipments
	c.equipSetMap = equipSetMap
	c.buildingByName = buildingByName
	c.techByName = techByName

	// 宝箱配置 + 奖池（★ 2026-10-04 并入配置缓存：/chest 原每次请求查表 + 逐箱查奖池）
	var chs []model.EzfyCfgChest
	db.Where("enabled <> 0").Order("sort_no, id").Find(&chs)
	c.chests = chs
	var cps []model.EzfyCfgChestItem
	db.Order("id").Find(&cps)
	pm := make(map[int][]model.EzfyCfgChestItem, len(cps))
	for _, p := range cps {
		pm[p.ChestId] = append(pm[p.ChestId], p)
	}
	c.chestPools = pm

	if loadTiles {
		// 地图格子覆盖（改地形 / 设寇城·活动寇城；管理端可维护）
		// ★ 2026-10-04 周期刷新默认跳过（全表拉取太贵）；2026-10-05 起改为
		//   「先算廉价指纹，只有真变了才拉」→ 多机也能收敛（见 ezfyHeavyFingerprint）。
		var tiles []model.EzfyMapTile
		if err := db.Find(&tiles).Error; err != nil {
			// ★★ 2026-10-05 修复「重启后地形整体翻转、玩家城市被误搬」：
			//   读瓦片失败时**保留旧缓存**，绝不能用空表覆盖 ——
			//   空表 = 所有覆盖消失 = 算法海洋重新变回海洋 = 建在覆盖格上的城
			//   被判定成「海洋上的城」，触发（旧版）开机自动搬城 → 玩家坐标自己变。
			//   即便现在已停用自动搬城，也不能让一次 WAN 抖动把整张地图地形翻掉。
			log.Printf("ezfy 地图瓦片读取失败，保留旧缓存（%d 格）: %v", len(c.tiles), err)
		} else {
			tm := make(map[int64]model.EzfyMapTile, len(tiles))
			for _, t := range tiles {
				tm[ezfyTileKey(t.X, t.Y)] = t
			}
			c.tiles = tm
		}
	}

	// ★★ 2026-10-05 修复「活动野地配置了没生效」：这段原来和地图瓦片一起被放在
	//   `if !skipHeavy` 里，而**周期刷新（每 30s）带 skipHeavy=true** —— 双机部署时
	//   管理端保存只会打到其中一台，那台走 cfgsReload 全量刷新；**另一台只跑周期刷新，
	//   它的 c.actWilds 永远停在旧值（甚至是空的）**。玩家请求被负载均衡到这台时，
	//   看到的就是「配置了没生效」（表现为时灵时不灵）。
	//   活动野地表只有几行（线上 3 行），**每次都读**的成本可以忽略，必须无条件刷新。
	var aws []model.EzfyActWild
	db.Find(&aws)
	awm := make(map[int64]*model.EzfyActWild, len(aws))
	for _, a := range aws {
		cp := a
		awm[ezfyTileKey(a.X, a.Y)] = &cp
	}
	c.actWilds = awm

	// 军衔配置（管理端可维护；表为空时回落内置默认，保证排名逻辑永远可用）
	var rks []model.EzfyCfgRank
	db.Order("id").Find(&rks)
	if len(rks) == 0 {
		rks = ezfyDefaultRanks()
	}
	c.ranks = rks

	// 建筑数量上限（单行；缺行时用线上现值 36/36/33/20）
	// ★ 三个玩法开关的默认值也必须写在这里：缺行时如果留 0，会变成「全关」，
	//   与「默认开」的语义相反（见 ezfyRecruitCostOn / ezfyFoodUpkeepOn / ezfyMarchOilOn）。
	c.limit = model.EzfyCfgLimit{ID: 1, MilitaryMax: 36, ResourceMax: 36, HouseMax: 33, FactoryMax: 20,
		GatherMaxPerOrder: ezfyGatherMaxDefault, MallBuyMax: ezfyMallBuyMaxDef,
		// ★ 2026-09-26：召集消耗粮食 / 获得人口（缺行时给默认 10 万）
		ConveneFoodCost: ezfyConveneFoodCostDef, ConvenePopGain: ezfyConvenePopGainDef,
		// ★ 2026-09-26：召集硬性人口上限（缺行时默认 0 = 不限）
		ConvenePopMax: ezfyConvenePopMaxDef,
		WildTroopMult: ezfyWildMultDef,
		WildResMult:   ezfyWildResMultDef,
		GatherResMult: ezfyGatherResMultDef,
		RecruitCostOn: ezfyRecruitCostDef, FoodUpkeepOn: ezfyFoodUpkeepDef, MarchOilOn: ezfyMarchOilDef,
		WarRequireOn: ezfyWarRequireDef, MarchCapOn: ezfyMarchCapDef,
		// ★ 2026-09-26：民居容量限制 / 召集人口灵活配置（缺行时同样要显式给默认开）
		HousePopLimitOn: ezfyHousePopLimitDef, ConveneFlexibleOn: ezfyConveneFlexDef,
		// ★ 训练加速黄金倍率 / 伤兵恢复黄金折扣率：百分比口径（线上现值 0.1 = 训练近乎免费）
		SpeedTrainRate: 0.1, WoundHealRate: 100,
		// ★ 2026-10-08 新玩家落地洲缺行兜底（默认欧洲；0/越界行读取时回落见 ezfyDefaultContinent）
		DefaultContinent: ezfyDefaultMoveContinent,
		// ★ 2026-09-23：兵力上限 / 伤兵存活天数的缺行兜底（0 无意义 → 默认 50 亿 / 3 天）
		TroopMax: ezfyTroopMaxDef, WoundExpireDays: ezfyWoundExpireDaysDef,
		// ★ 2026-10-02：侦察机每架侦查成功率%（0 无意义 → 回落默认 20）
		ReconSuccessPct: ezfyReconSuccessPctDef,
		// ★ 2026-10-02 战力榜权重缺行兜底（兵力榜 → 战力榜）
		PowerTechPerLevel: 120, PowerTechPerTech: 100, PowerBuildPerLevel: 80,
		PowerTroopType: 300, PowerTroopPow: 0.8, PowerCompressPow: 0.5}
	var lim model.EzfyCfgLimit
	if err := db.First(&lim, 1).Error; err == nil {
		c.limit = lim
	}

	// 二战聊天敏感词（独立维护页）
	var wds []model.EzfyWordFilter
	db.Order("id").Find(&wds)
	c.words = wds

	// 资源显示名配置（管理端「资源名称维护」可改名；表为空回落内置默认）
	// 预组装成返回结构，读侧（/res-cfg、/view）零 SQL。
	var resRows []model.EzfyCfgResource
	db.Order("sort, id").Find(&resRows)
	c.resCfg = buildResCfgView(resRows)
}

// ezfyDefaultRanks 内置兜底军衔（与 seed 一致，复刻原版 rankIndex.html）
// ★ 2026-09-24 「军衔需要声望太少，统一在原来基础上 ×10」。
func ezfyDefaultRanks() []model.EzfyCfgRank {
	// ★ 2026-09-27 军衔门槛按**线上 ezfy_cfg_rank 现值**对齐（与 seed.seedEzfyRanks 口径一致）
	return []model.EzfyCfgRank{
		{ID: 1, Name: "列兵", Post: "士兵", NeedPrestige: 0, CityMax: 1},
		{ID: 2, Name: "上等兵", Post: "班长", NeedPrestige: 1000, CityMax: 2},
		{ID: 3, Name: "下士", Post: "排长", NeedPrestige: 3000, CityMax: 3},
		{ID: 4, Name: "中士", Post: "排长", NeedPrestige: 6000, CityMax: 4},
		{ID: 5, Name: "上士", Post: "连长", NeedPrestige: 10000, CityMax: 5},
		{ID: 6, Name: "军士长", Post: "连长", NeedPrestige: 30000, CityMax: 6},
		{ID: 7, Name: "准尉", Post: "营长", NeedPrestige: 50000, CityMax: 7},
		{ID: 8, Name: "少尉", Post: "营长", NeedPrestige: 80000, CityMax: 8},
		{ID: 9, Name: "中尉", Post: "营长", NeedPrestige: 120000, CityMax: 9},
		{ID: 10, Name: "上尉", Post: "团长", NeedPrestige: 200000, CityMax: 10},
		{ID: 11, Name: "大尉", Post: "团长", NeedPrestige: 300000, CityMax: 11},
		{ID: 12, Name: "少校", Post: "旅长", NeedPrestige: 500000, CityMax: 12},
		{ID: 13, Name: "中校", Post: "旅长", NeedPrestige: 1000000, CityMax: 13},
		{ID: 14, Name: "上校", Post: "旅长", NeedPrestige: 2000000, CityMax: 14},
		{ID: 15, Name: "大校", Post: "师长", NeedPrestige: 4000000, CityMax: 15},
		{ID: 16, Name: "少将", Post: "师长", NeedPrestige: 8000000, CityMax: 16},
		{ID: 17, Name: "中将", Post: "军长", NeedPrestige: 16000000, CityMax: 17},
		{ID: 18, Name: "上将", Post: "军长", NeedPrestige: 32000000, CityMax: 18},
		{ID: 19, Name: "大将", Post: "军长", NeedPrestige: 64000000, CityMax: 19},
		{ID: 20, Name: "五星上将", Post: "司令", NeedPrestige: 100000000, CityMax: 20},
	}
}

// chestPool 某宝箱的奖池（按 id 升序，与「逐箱查库」同序；未配置返回空）
func (c *ezfyConfigCache) chestPool(chestId int) []model.EzfyCfgChestItem {
	return c.chestPools[chestId]
}

func (c *ezfyConfigCache) general(id int) *model.EzfyCfgGeneral {
	if g, ok := c.generals[id]; ok {
		return &g
	}
	return nil
}

// isGeneral 该池子 ID 是否为名将（kind==2）。
// ★ 2026-09-29 玩家军官列表据此打「是否名将」标。普通军官池（kind=1）不算。
func (c *ezfyConfigCache) isGeneral(id int) bool {
	g := c.general(id)
	return g != nil && g.Kind == 2
}

// generalByName 按名字回查军官池
//
// ★ 2026-10-06 起军校招来的军官也写 general_id（= 军官池 id），所以名字回查只兜底历史老数据。
// 同名多条时优先 kind=1（军校池），其次取 id 最小的，保证同一名军官每次查到的都一样。
func (c *ezfyConfigCache) generalByName(name string) *model.EzfyCfgGeneral {
	if name == "" {
		return nil
	}
	var best *model.EzfyCfgGeneral
	for id, g := range c.generals {
		// 三维全 0 的池子条目没意义，跳过
		if g.Name != name || g.Military+g.Logistics+g.Learning <= 0 {
			continue
		}
		if best == nil {
			cp := g
			best = &cp
			continue
		}
		var better bool
		if (g.Kind == 1) != (best.Kind == 1) {
			better = g.Kind == 1
		} else {
			better = id < best.ID
		}
		if better {
			cp := g
			best = &cp
		}
	}
	return best
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
// ★ 2026-09-22 军校招募/刷新**从池子里抽**，不再纯随机生成。
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

// sortedTroops 返回按 id 升序的兵种配置切片（读进程内缓存，不发 SQL）。
//
// ★ 2026-10-03 性能：/troops 是 30s 轮询接口，之前每次全表 SELECT ezfy_cfg_troop
//
//	经跨 WAN 到 RDS（单次 13~37ms）；配置早已整表载入缓存，直接读内存即可。
func (c *ezfyConfigCache) sortedTroops() []model.EzfyCfgTroop {
	ids := make([]int, 0, len(c.troops))
	for id := range c.troops {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	out := make([]model.EzfyCfgTroop, 0, len(ids))
	for _, id := range ids {
		out = append(out, c.troops[id])
	}
	return out
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
