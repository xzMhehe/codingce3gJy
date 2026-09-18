package handler

import (
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

func ezfyWildlandLevel(x, y int) int {
	h := ezfyAbs(x*83492791 ^ y*6291469)
	return h % 11
}

func ezfyKouLevel(x, y int) int {
	h := ezfyAbs(x*83492791 ^ y*6291469)
	return h % 11
}

// ============ 军衔（复刻 ConfigServer.RANKS，20 档） ============

var ezfyRanks = [][3]string{
	{"列兵", "士兵", "0"}, {"上等兵", "班长", "100"}, {"下士", "排长", "300"}, {"中士", "排长", "600"},
	{"上士", "连长", "1000"}, {"军士长", "连长", "1500"}, {"准尉", "营长", "2200"}, {"少尉", "营长", "3000"},
	{"中尉", "营长", "4000"}, {"上尉", "团长", "5200"}, {"大尉", "团长", "6600"}, {"少校", "旅长", "8200"},
	{"中校", "旅长", "10000"}, {"上校", "旅长", "12000"}, {"大校", "师长", "14500"}, {"少将", "师长", "17500"},
	{"中将", "军长", "21000"}, {"上将", "军长", "25000"}, {"大将", "军长", "30000"}, {"五星上将", "司令", "40000"},
}

func ezfyRankIndex(prestige int) int {
	idx := 0
	for i := 0; i < len(ezfyRanks); i++ {
		need := 0
		for _, ch := range ezfyRanks[i][2] {
			need = need*10 + int(ch-'0')
		}
		if prestige >= need {
			idx = i
		}
	}
	return idx
}

func ezfyRankName(prestige int) string { return ezfyRanks[ezfyRankIndex(prestige)][0] }
func ezfyRankPost(prestige int) string { return ezfyRanks[ezfyRankIndex(prestige)][1] }

// ============ 配置缓存（进程内加载，seed 完成后首用时加载） ============

type ezfyConfigCache struct {
	once           sync.Once
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
}

var ezfyCfg ezfyConfigCache

func (c *ezfyConfigCache) load(db *gorm.DB) {
	c.once.Do(func() {
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
	})
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
