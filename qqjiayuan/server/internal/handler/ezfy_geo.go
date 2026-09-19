package handler

import (
	"fmt"
	"log"
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

func ezfyInShape(x, y int, s []int) bool {
	if s[0] == 0 {
		return s[1] <= x && x <= s[3] && s[2] <= y && y <= s[4]
	}
	a := float64(s[3])
	b := float64(s[4])
	dx := (float64(x) - float64(s[1])) / a
	dy := (float64(y) - float64(s[2])) / b
	return dx*dx+dy*dy <= 1
}

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

// ============ 坐标哈希（复刻 GameServiceImpl getTerrain/getWildlandLevel/getKouLevel） ============

func ezfyAbs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func ezfyTerrain(x, y int) int {
	h := ezfyAbs(x*73856093 ^ y*19349663)
	return h%8 + 1
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
			if ezfyTerrain(x+dx, y+dy) == 8 {
				return true
			}
		}
	}
	return false
}

// ezfyIsCoastalPlainAt 该坐标是否沿海平原
func ezfyIsCoastalPlainAt(x, y int) bool {
	return ezfyTerrain(x, y) == 1 && ezfyHasSeaNeighbor(x, y)
}

// ezfyTerrainEx 实际地形：平原且靠海 → 沿海平原(9)，其余同 ezfyTerrain
//
// ★ 只用于「展示 + 建城选址 + 海城判定」；
//
//	依赖原版 8 种地形的玩法逻辑（珠宝按地形、野地产出）继续用 ezfyTerrain。
func ezfyTerrainEx(x, y int) int {
	if ezfyIsCoastalPlainAt(x, y) {
		return ezfyTerrainCoastalPlain
	}
	return ezfyTerrain(x, y)
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
	for r := 1; r <= 40; r++ {
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
	// 军衔配置（按等级 1..N 排序）
	ranks []model.EzfyCfgRank
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

	// 军衔配置（管理端可维护；表为空时回落内置默认，保证排名逻辑永远可用）
	var rks []model.EzfyCfgRank
	db.Order("id").Find(&rks)
	if len(rks) == 0 {
		rks = ezfyDefaultRanks()
	}
	c.ranks = rks
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
