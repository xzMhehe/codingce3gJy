package seed

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"qqjiayuan/server/internal/model"
)

// 幻想西游种子：go:embed 提取产物 JSON（tools/hxxy_extract/extract.php 产出）
// 全部幂等：表为空才灌入。

//go:embed hxxy_data/maps.json
var hxxyMapsJSON []byte

//go:embed hxxy_data/mapgrids.json
var hxxyMapGridsJSON []byte

//go:embed hxxy_data/npcs.json
var hxxyNpcsJSON []byte

//go:embed hxxy_data/spawns.json
var hxxySpawnsJSON []byte

//go:embed hxxy_data/map_npcs.json
var hxxyMapNpcsJSON []byte

//go:embed hxxy_data/mapnpcs_extract.json
var hxxyMapNpcsExtractJSON []byte

//go:embed hxxy_data/items.json
var hxxyItemsJSON []byte

//go:embed hxxy_data/equips.json
var hxxyEquipsJSON []byte

//go:embed hxxy_data/skills.json
var hxxySkillsJSON []byte

//go:embed hxxy_data/pets.json
var hxxyPetsJSON []byte

//go:embed hxxy_data/bosses.json
var hxxyBossesJSON []byte

//go:embed hxxy_data/titles.json
var hxxyTitlesJSON []byte

type hxxyMapRaw struct {
	NodeID    uint   `json:"node_id"`
	Dtx       int    `json:"dtx"`
	Dty       int    `json:"dty"`
	Name      string `json:"name"`
	Desc      string `json:"desc"`
	Up        string `json:"up"`
	Down      string `json:"down"`
	Left      string `json:"left"`
	Right     string `json:"right"`
	UpJump    string `json:"up_jump"`
	DownJump  string `json:"down_jump"`
	LeftJump  string `json:"left_jump"`
	RightJump string `json:"right_jump"`
}

type hxxyNpcRaw struct {
	NpcID uint   `json:"npc_id"`
	Name  string `json:"name"`
	Level int    `json:"level"`
	HP    int    `json:"hp"`
	MaxHP int    `json:"max_hp"`
	MP    int    `json:"mp"`
	MaxMP int    `json:"max_mp"`
	Atk   int    `json:"atk"`
	Mg    int    `json:"mg"`
	Def   int    `json:"def"`
	Mf    int    `json:"mf"`
	Bg    int    `json:"bg"`
	Hg    int    `json:"hg"`
	Lg    int    `json:"lg"`
	Bf    int    `json:"bf"`
	Hf    int    `json:"hf"`
	Lf    int    `json:"lf"`
	Take  string `json:"take"`
}

type hxxySpawnRaw struct {
	Dtx        int    `json:"dtx"`
	Dty        int    `json:"dty"`
	NpcID      uint   `json:"npc_id"`
	Name       string `json:"name"`
	Difficulty string `json:"difficulty"`
}

type hxxyMapNpcRaw struct {
	Dtx      int           `json:"dtx"`
	Dty      int           `json:"dty"`
	NpcID    uint          `json:"npc_id"`
	Name     string        `json:"name"`
	Img      string        `json:"img"`
	Dialogue string        `json:"dialogue"`
	Shop     string        `json:"shop"`
	Teles    []hxxyTeleRaw `json:"teles"`
}

type hxxyTeleRaw struct {
	Name string `json:"name"`
	Dtx  int    `json:"dtx"`
	Dty  int    `json:"dty"`
}

type hxxyItemRaw struct {
	ItemID    uint   `json:"item_id"`
	Name      string `json:"name"`
	Desc      string `json:"desc"`
	Category  int    `json:"category"`
	BeanPrice int    `json:"bean_price"`
	Price     int    `json:"price"`
	Level     int    `json:"level"`
	Weight    int    `json:"weight"`
	Bind      int    `json:"bind"`
}

type hxxyEquipRaw struct {
	EquipID   uint   `json:"equip_id"`
	Name      string `json:"name"`
	Desc      string `json:"desc"`
	HP        int    `json:"hp"`
	Atk       int    `json:"atk"`
	Mg        int    `json:"mg"`
	Def       int    `json:"def"`
	Bg        int    `json:"bg"`
	Hg        int    `json:"hg"`
	Lg        int    `json:"lg"`
	Bf        int    `json:"bf"`
	Hf        int    `json:"hf"`
	Lf        int    `json:"lf"`
	Level     int    `json:"level"`
	Weight    int    `json:"weight"`
	Bind      int    `json:"bind"`
	BeanPrice int    `json:"bean_price"`
	Price     int    `json:"price"`
	Slot      int    `json:"slot"`
	Sect      int    `json:"sect"`
	Category  int    `json:"category"`
}

type hxxySkillRaw struct {
	SkillID    uint    `json:"skill_id"`
	Category   int     `json:"category"`
	Name       string  `json:"name"`
	Desc       string  `json:"desc"`
	Multiplier float64 `json:"multiplier"`
}

type hxxyPetRaw struct {
	SpeciesID uint   `json:"species_id"`
	Name      string `json:"name"`
	Level     int    `json:"level"`
	HP        int    `json:"hp"`
	MaxHP     int    `json:"max_hp"`
	MP        int    `json:"mp"`
	MaxMP     int    `json:"max_mp"`
	Atk       int    `json:"atk"`
	Mg        int    `json:"mg"`
	Def       int    `json:"def"`
	Mf        int    `json:"mf"`
	Bg        int    `json:"bg"`
	Hg        int    `json:"hg"`
	Lg        int    `json:"lg"`
	Bf        int    `json:"bf"`
	Hf        int    `json:"hf"`
	Lf        int    `json:"lf"`
}

type hxxyBossRaw struct {
	BossID uint   `json:"boss_id"`
	Name   string `json:"name"`
	Level  int    `json:"level"`
	HP     int    `json:"hp"`
	MaxHP  int    `json:"max_hp"`
	MP     int    `json:"mp"`
	MaxMP  int    `json:"max_mp"`
	Atk    int    `json:"atk"`
	Mg     int    `json:"mg"`
	Def    int    `json:"def"`
	Mf     int    `json:"mf"`
	Bg     int    `json:"bg"`
	Hg     int    `json:"hg"`
	Lg     int    `json:"lg"`
	Bf     int    `json:"bf"`
	Hf     int    `json:"hf"`
	Lf     int    `json:"lf"`
	Take   string `json:"take"`
}

type hxxyTitleRaw struct {
	TitleID uint   `json:"title_id"`
	Name    string `json:"name"`
	Desc    string `json:"desc"`
	HP      int    `json:"hp"`
	Atk     int    `json:"atk"`
	Def     int    `json:"def"`
	Mg      int    `json:"mg"`
}

type hxxyDrop struct {
	Type string `json:"type"` // item / equip
	ID   uint   `json:"id"`
	Rate int    `json:"rate"` // 万分比
}

func hxxyCount(db *gorm.DB, m interface{}) int64 {
	var c int64
	db.Model(m).Count(&c)
	return c
}

func seedHxxy(db *gorm.DB) {
	// ---------- 地图 ----------
	if hxxyCount(db, &model.HxxyMapNode{}) == 0 {
		var raws []hxxyMapRaw
		mustJSON(hxxyMapsJSON, &raws)
		rows := make([]model.HxxyMapNode, 0, len(raws))
		for _, r := range raws {
			rows = append(rows, model.HxxyMapNode{
				ID: r.NodeID, Dtx: r.Dtx, Dty: r.Dty, Name: r.Name, Desc: r.Desc,
				Up: r.Up, Down: r.Down, Left: r.Left, Right: r.Right,
				UpJump: r.UpJump, DownJump: r.DownJump, LeftJump: r.LeftJump, RightJump: r.RightJump,
			})
		}
		db.CreateInBatches(rows, 200)
		log.Printf("[hxxy] 地图节点灌入 %d 条", len(rows))
	}

	// ---------- 地图网格（查看地图，复刻 xdt） ----------
	if hxxyCount(db, &model.HxxyMapGrid{}) == 0 {
		var raws map[string]json.RawMessage
		mustJSON(hxxyMapGridsJSON, &raws)
		rows := make([]model.HxxyMapGrid, 0, len(raws))
		for k, v := range raws {
			dtx, err := strconv.Atoi(k)
			if err != nil {
				continue
			}
			rows = append(rows, model.HxxyMapGrid{Dtx: dtx, Grid: string(v)})
		}
		db.CreateInBatches(rows, 100)
		log.Printf("[hxxy] 地图网格灌入 %d 张", len(rows))
	}

	// ---------- 物品 ----------
	if hxxyCount(db, &model.HxxyItem{}) == 0 {
		var raws []hxxyItemRaw
		mustJSON(hxxyItemsJSON, &raws)
		rows := make([]model.HxxyItem, 0, len(raws))
		for _, r := range raws {
			rows = append(rows, model.HxxyItem{
				ID: r.ItemID, Name: r.Name, Desc: r.Desc, Category: r.Category,
				BeanPrice: r.BeanPrice, Price: r.Price, Level: r.Level, Weight: r.Weight, Bind: r.Bind,
				Effect: hxxyItemEffect(r),
			})
		}
		db.CreateInBatches(rows, 200)
		log.Printf("[hxxy] 物品灌入 %d 条", len(rows))
	}

	// ---------- 装备 ----------
	if hxxyCount(db, &model.HxxyEquip{}) == 0 {
		var raws []hxxyEquipRaw
		mustJSON(hxxyEquipsJSON, &raws)
		rows := make([]model.HxxyEquip, 0, len(raws))
		for _, r := range raws {
			rows = append(rows, model.HxxyEquip{
				ID: r.EquipID, Name: r.Name, Desc: r.Desc,
				HP: r.HP, Atk: r.Atk, Mg: r.Mg, Def: r.Def,
				Bg: r.Bg, Hg: r.Hg, Lg: r.Lg, Bf: r.Bf, Hf: r.Hf, Lf: r.Lf,
				Level: r.Level, Weight: r.Weight, Bind: r.Bind,
				BeanPrice: r.BeanPrice, Price: r.Price,
				Slot: r.Slot, Sect: r.Sect, Category: r.Category,
			})
		}
		db.CreateInBatches(rows, 200)
		log.Printf("[hxxy] 装备灌入 %d 条", len(rows))
	}

	// ---------- 技能 ----------
	if hxxyCount(db, &model.HxxySkill{}) == 0 {
		var raws []hxxySkillRaw
		mustJSON(hxxySkillsJSON, &raws)
		rows := make([]model.HxxySkill, 0, len(raws))
		for _, r := range raws {
			sk := model.HxxySkill{
				ID: r.SkillID, Category: r.Category, Name: r.Name, Desc: r.Desc,
				Multiplier: int(r.Multiplier * 100), // 倍率存整数百分比（100=1.0倍）
				MpCost:     int(r.Multiplier * 20),
				LearnLevel: 1,
			}
			if r.SkillID == 1 {
				sk.MpCost = 0
			}
			switch {
			case strings.Contains(r.Desc, "将军府"):
				sk.Sect = 1
			case strings.Contains(r.Desc, "龙宫"):
				sk.Sect = 2
			case strings.Contains(r.Desc, "月宫"):
				sk.Sect = 3
			case strings.Contains(r.Desc, "方寸山"):
				sk.Sect = 4
			case strings.Contains(r.Desc, "普陀山"):
				sk.Sect = 5
			}
			if sk.Sect > 0 {
				sk.LearnLevel = 10
			}
			rows = append(rows, sk)
		}
		db.CreateInBatches(rows, 200)
		log.Printf("[hxxy] 技能灌入 %d 条", len(rows))
	}

	// ---------- 宠物种族 ----------
	if hxxyCount(db, &model.HxxyPetSpecies{}) == 0 {
		var raws []hxxyPetRaw
		mustJSON(hxxyPetsJSON, &raws)
		rows := make([]model.HxxyPetSpecies, 0, len(raws))
		for _, r := range raws {
			rows = append(rows, model.HxxyPetSpecies{
				ID: r.SpeciesID, Name: r.Name, Level: r.Level,
				HP: r.HP, MaxHP: r.MaxHP, MP: r.MP, MaxMP: r.MaxMP,
				Atk: r.Atk, Mg: r.Mg, Def: r.Def, Mf: r.Mf,
				Bg: r.Bg, Hg: r.Hg, Lg: r.Lg, Bf: r.Bf, Hf: r.Hf, Lf: r.Lf,
			})
		}
		db.CreateInBatches(rows, 200)
		log.Printf("[hxxy] 宠物种族灌入 %d 条", len(rows))
	}

	// ---------- BOSS ----------
	if hxxyCount(db, &model.HxxyBoss{}) == 0 {
		var raws []hxxyBossRaw
		mustJSON(hxxyBossesJSON, &raws)
		rows := make([]model.HxxyBoss, 0, len(raws))
		for _, r := range raws {
			rows = append(rows, model.HxxyBoss{
				ID: r.BossID, Name: r.Name, Level: r.Level,
				HP: r.HP, MaxHP: r.MaxHP, MP: r.MP, MaxMP: r.MaxMP,
				Atk: r.Atk, Mg: r.Mg, Def: r.Def, Mf: r.Mf,
				Bg: r.Bg, Hg: r.Hg, Lg: r.Lg, Bf: r.Bf, Hf: r.Hf, Lf: r.Lf, Take: r.Take,
			})
		}
		db.CreateInBatches(rows, 200)
		log.Printf("[hxxy] BOSS灌入 %d 条", len(rows))
	}

	// ---------- 头衔 ----------
	if hxxyCount(db, &model.HxxyTitle{}) == 0 {
		var raws []hxxyTitleRaw
		mustJSON(hxxyTitlesJSON, &raws)
		rows := make([]model.HxxyTitle, 0, len(raws))
		for _, r := range raws {
			rows = append(rows, model.HxxyTitle{
				ID: r.TitleID, Name: r.Name, Desc: r.Desc,
				HP: r.HP, Atk: r.Atk, Def: r.Def, Mg: r.Mg,
			})
		}
		db.CreateInBatches(rows, 200)
		log.Printf("[hxxy] 头衔灌入 %d 条", len(rows))
	}

	// ---------- NPC + 刷怪点（刷怪表决定战斗 NPC 与掉落/奖励） ----------
	if hxxyCount(db, &model.HxxyNpc{}) == 0 {
		var raws []hxxyNpcRaw
		mustJSON(hxxyNpcsJSON, &raws)
		rows := make([]model.HxxyNpc, 0, len(raws))
		for _, r := range raws {
			rows = append(rows, model.HxxyNpc{
				ID: r.NpcID, Name: r.Name, Level: r.Level,
				HP: r.HP, MaxHP: r.MaxHP, MP: r.MP, MaxMP: r.MaxMP,
				Atk: r.Atk, Mg: r.Mg, Def: r.Def, Mf: r.Mf,
				Bg: r.Bg, Hg: r.Hg, Lg: r.Lg, Bf: r.Bf, Hf: r.Hf, Lf: r.Lf, Take: r.Take,
			})
		}
		db.CreateInBatches(rows, 200)
		log.Printf("[hxxy] NPC灌入 %d 条", len(rows))
	}

	if hxxyCount(db, &model.HxxySpawn{}) == 0 {
		var raws []hxxySpawnRaw
		mustJSON(hxxySpawnsJSON, &raws)
		rows := make([]model.HxxySpawn, 0, len(raws))
		for _, r := range raws {
			rows = append(rows, model.HxxySpawn{
				Dtx: r.Dtx, Dty: r.Dty, NpcID: r.NpcID, Name: r.Name, Difficulty: r.Difficulty,
			})
		}
		db.CreateInBatches(rows, 200)
		log.Printf("[hxxy] 刷怪点灌入 %d 条", len(rows))

		// 标记战斗 NPC + 生成掉落/奖励（复刻原版打怪：经验/银两随等级与难度成长）
		hxxyMarkBattleNpcs(db, raws)
	}

	// ---------- 功能NPC放置（传送/商店/对话，提取自原版 xy020.php + npc.php） ----------
	if hxxyCount(db, &model.HxxyMapNpc{}) == 0 {
		var raws []hxxyMapNpcRaw
		mustJSON(hxxyMapNpcsJSON, &raws)
		rows := make([]model.HxxyMapNpc, 0, len(raws))
		for _, r := range raws {
			teles := ""
			if len(r.Teles) > 0 {
				if b, err := json.Marshal(r.Teles); err == nil {
					teles = string(b)
				}
			}
			rows = append(rows, model.HxxyMapNpc{
				Dtx: r.Dtx, Dty: r.Dty, NpcID: r.NpcID, Name: r.Name,
				Img: r.Img, Dialogue: r.Dialogue, Shop: r.Shop, Teles: teles,
			})
		}
		db.CreateInBatches(rows, 200)
		log.Printf("[hxxy] 功能NPC灌入 %d 条", len(rows))
	}

	// ---------- 节点NPC放置补全（提取自原版 map/*.php 各节点 clj=7 的 NPC 链接，388条）----------
	// 幂等：只补缺不覆盖（管理端已在 hxxy_map_npcs 配置的商店/传送保持不变）
	{
		var ext []hxxyMapNpcRaw
		mustJSON(hxxyMapNpcsExtractJSON, &ext)
		have := map[string]bool{}
		var cur []model.HxxyMapNpc
		db.Find(&cur)
		for _, r := range cur {
			have[fmt.Sprintf("%d_%d_%d", r.Dtx, r.Dty, r.NpcID)] = true
		}
		var npcIDs []uint
		db.Model(&model.HxxyNpc{}).Pluck("id", &npcIDs)
		npcSet := map[uint]bool{}
		for _, id := range npcIDs {
			npcSet[id] = true
		}
		rows := []model.HxxyMapNpc{}
		for _, r := range ext {
			if r.NpcID == 0 || !npcSet[r.NpcID] { // 原版活动NPC不在库内则跳过
				continue
			}
			key := fmt.Sprintf("%d_%d_%d", r.Dtx, r.Dty, r.NpcID)
			if have[key] {
				continue
			}
			have[key] = true
			rows = append(rows, model.HxxyMapNpc{Dtx: r.Dtx, Dty: r.Dty, NpcID: r.NpcID, Name: r.Name})
		}
		if len(rows) > 0 {
			db.CreateInBatches(rows, 200)
			log.Printf("[hxxy] 节点NPC补全 %d 条", len(rows))
		}
	}

	// ---------- 副本定义（复刻原版 fb/*：斩妖台/北俱芦洲/变异竹林/水帘洞天/老君洞 × 普通/困难/梦魇/地狱） ----------
	if hxxyCount(db, &model.HxxyDungeon{}) == 0 {
		type fbDef struct {
			Base string
			Desc string
			Lv   [4]int // 普通/困难/梦魇/地狱 进入等级
		}
		defs := []fbDef{
			{"北俱芦洲", "北俱芦洲妖气冲天，击杀副本内5大守护BOSS！", [4]int{15, 30, 45, 60}},
			{"水帘洞天", "水帘洞天藏于花果山，击杀副本内5大守护BOSS！", [4]int{20, 35, 50, 65}},
			{"变异竹林", "变异竹林妖魔横行，击杀副本内5大守护BOSS！", [4]int{25, 40, 55, 70}},
			{"斩妖台", "斩妖台乃天庭斩妖除魔之地，击杀副本内5大守护BOSS！", [4]int{30, 45, 60, 75}},
			{"老君洞", "老君洞中丹炉异变，击杀副本内5大守护BOSS！", [4]int{35, 50, 65, 80}},
		}
		diffs := [4]string{"普通", "困难", "梦魇", "地狱"}
		rows := []model.HxxyDungeon{}
		id := uint(1)
		for _, d := range defs {
			for i, diff := range diffs {
				rows = append(rows, model.HxxyDungeon{
					ID: id, Name: fmt.Sprintf("%s副本【%s】", d.Base, diff), Desc: d.Desc,
					Floors: 5, Daily: 1, MinLevel: d.Lv[i],
				})
				id++
			}
		}
		db.Create(&rows)
		log.Printf("[hxxy] 副本定义灌入 %d 条", len(rows))
	}

	// ---------- 任务库（内部按数量判断是否重建） ----------
	hxxySeedQuests(db)
}

// hxxyMarkBattleNpcs 刷怪表中的 NPC 标记为战斗型，并生成 经验/银两奖励 与 掉落
func hxxyMarkBattleNpcs(db *gorm.DB, spawns []hxxySpawnRaw) {
	// npc_id -> 最高难度
	diff := map[uint]string{}
	for _, s := range spawns {
		if s.Difficulty == "困难" || diff[s.NpcID] == "" {
			if s.Difficulty == "困难" {
				diff[s.NpcID] = "困难"
			} else if diff[s.NpcID] == "" {
				diff[s.NpcID] = "普通"
			}
		}
	}
	// 掉落候选：低级药品（cat5）+ 各部位装备（cat3-8）
	var heals []model.HxxyItem
	db.Where("category = 5 AND level <= 3").Order("id").Limit(10).Find(&heals)
	type eqRow struct {
		ID  uint
		Lvl int
	}
	var eqs []eqRow
	db.Model(&model.HxxyEquip{}).Select("id, level as lvl").Where("category BETWEEN 3 AND 8 AND level <= 10").Order("id").Scan(&eqs)

	for npcID, d := range diff {
		npc := &model.HxxyNpc{}
		if err := db.First(npc, npcID).Error; err != nil {
			continue
		}
		mult := 1
		if d == "困难" {
			mult = 2
		}
		exp := npc.Level*npc.Level*3 + 50
		money := npc.Level*20 + 10
		updates := map[string]interface{}{
			"kind":         1,
			"exp_reward":   exp * mult,
			"money_reward": int64(money) * int64(mult),
		}
		// 掉落：药品 20% + 装备 5%（确定性选取避免随机漂移）
		drops := []hxxyDrop{}
		if len(heals) > 0 {
			drops = append(drops, hxxyDrop{Type: "item", ID: heals[int(npcID)%len(heals)].ID, Rate: 2000})
		}
		if len(eqs) > 0 {
			drops = append(drops, hxxyDrop{Type: "equip", ID: eqs[int(npcID)%len(eqs)].ID, Rate: 500})
		}
		if b, err := json.Marshal(drops); err == nil {
			updates["drops"] = string(b)
		}
		db.Model(&model.HxxyNpc{}).Where("id = ?", npcID).Updates(updates)
	}
}

// hxxyItemEffect 按名称/描述/分类生成物品效果 JSON
// 格式：{"hp":n} {"mp":n} {"maxhp":n,"daily":n} {"exp":n} {"money":n} {"beans":n}
// {"full":1} {"vip":分钟} {"box":1} {"goto":"dtx_dty"} {"skill":技能id} {"skill_sect":1} {"teleport":1}
func hxxyItemEffect(r hxxyItemRaw) string {
	numRe := regexp.MustCompile(`(?:恢复(?:HP|MP|体力|内力)?|增加)(HP|MP|体力|内力)?(\d+)`)
	add := func(field string, n int) string { return `{"` + field + `":` + strconv.Itoa(n) + `}` }

	switch r.Category {
	case 5: // 药品/食物：解析"恢复N"或"增加N HP/MP"
		if m := numRe.FindStringSubmatch(r.Desc); m != nil {
			n := m[2]
			unit := m[1]
			if strings.Contains(r.Desc, "限用") { // 丹药：永久加成，每日限用
				daily := 100
				if dm := regexp.MustCompile(`限用(\d+)个`).FindStringSubmatch(r.Desc); dm != nil {
					daily, _ = strconv.Atoi(dm[1])
				}
				field := "maxhp"
				switch {
				case strings.Contains(r.Desc, "MP") || strings.Contains(r.Desc, "内力"):
					field = "maxmp"
				case strings.Contains(r.Desc, "攻击"):
					field = "atk"
				case strings.Contains(r.Desc, "防御"):
					field = "def"
				case strings.Contains(r.Desc, "魔攻"):
					field = "mg"
				}
				return `{"` + field + `":` + n + `,"daily":` + strconv.Itoa(daily) + `}`
			}
			if unit == "MP" || unit == "内力" || strings.Contains(r.Desc, "MP") || strings.Contains(r.Desc, "内力") {
				return add("mp", atoi(n))
			}
			return add("hp", atoi(n))
		}
		return add("hp", 100)
	case 1: // 卷轴/秘籍
		switch {
		case strings.Contains(r.Name, "回城卷"):
			return `{"goto":"1_0"}`
		case strings.Contains(r.Name, "宠物指南"):
			return `{"skill":3}`
		case strings.Contains(r.Name, "门派秘籍"):
			return `{"skill_sect":1}`
		case strings.Contains(r.Name, "VIP"):
			return `{"vip":30}`
		case strings.Contains(r.Name, "腾云符"):
			return `{"teleport":1}`
		}
	case 4: // 礼包/特殊
		switch {
		case strings.Contains(r.Name, "金豆"):
			return `{"beans":1}`
		case strings.Contains(r.Name, "万能果"):
			return `{"full":1}`
		case strings.Contains(r.Name, "礼包") || strings.Contains(r.Name, "宝箱") || strings.Contains(r.Name, "袋"):
			return `{"box":1}`
		}
	case 8: // 宝箱
		return `{"box":1}`
	}
	return ""
}

// hxxySeedQuests 任务库（按名字查 NPC/物品，缺失即跳过该条）
// 分类：1主线（next 链式推进） 2支线 3日常；不足 15 条时整表重建（保留逻辑简单）
func hxxySeedQuests(db *gorm.DB) {
	npcID := func(name string) uint {
		var n model.HxxyNpc
		if err := db.Where("name = ?", name).First(&n).Error; err != nil {
			return 0
		}
		return n.ID
	}
	itemID := func(name string) uint {
		var it model.HxxyItem
		if err := db.Where("name = ?", name).First(&it).Error; err != nil {
			return 0
		}
		return it.ID
	}
	type qd struct {
		name, desc, typ string
		target          uint
		count, minLv    int
		exp             int
		money           int64
		next            int // 同列表下标（1起）
		cat             int // 1主线 2支线 3日常
		from            string
	}
	specs := []qd{
		// ---------- 主线（链式） ----------
		{"初出茅庐", "村外的田地里闹起了大老鼠，去击败5只大老鼠，证明你的实力！", "hunt", npcID("大老鼠"), 5, 1, 200, 500, 2, 1, "村长"},
		{"剿灭海贼", "海边海贼猖獗，再去击败5个海贼，为民除害。", "hunt", npcID("*海贼"), 5, 3, 300, 800, 3, 1, "村长"},
		{"拜见村长", "初战告捷！回去拜见村长，领取赏赐。", "talk", npcID("村长"), 1, 3, 150, 400, 4, 1, "村长"},
		{"长安来客", "长安城的李白想见见你这颗新星，去拜访他吧。", "talk", npcID("李白"), 1, 5, 300, 600, 5, 1, "李白"},
		{"狼患", "城外野狼成患，猎杀8只野狼还百姓安宁。", "hunt", npcID("野狼"), 8, 6, 500, 1200, 6, 1, "李白"},
		{"筹备药材", "军营缺药材，收集5份云南白药（小捆）以备不时之需。", "collect", itemID("云南白药（小捆）"), 5, 6, 400, 1000, 7, 1, "李白"},
		{"除暴安良", "黑衣大汉在城郊横行霸道，击败6个教训他们！", "hunt", npcID("黑衣大汉"), 6, 8, 700, 1600, 8, 1, "张果老"},
		{"拜见张果老", "张果老对你颇为赏识，去拜见他。", "talk", npcID("张果老"), 1, 8, 500, 1000, 9, 1, "张果老"},
		{"缉拿飞贼", "近来飞贼四起，缉拿8名飞贼领赏。", "hunt", npcID("飞贼"), 8, 10, 900, 2200, 10, 1, "李捕头"},
		{"傲来传令", "把军情传给傲来店的店小二，越快越好。", "talk", npcID("店小二"), 1, 10, 600, 1200, 11, 1, "店小二"},
		{"东海练兵", "东海虾兵操演不断，击败10名虾兵立威。", "hunt", npcID("虾兵"), 10, 12, 1200, 3000, 12, 1, "龙宫大弟子"},
		{"龙宫拜见", "龙宫大弟子有意引荐你入龙宫，去拜见他。", "talk", npcID("龙宫大弟子"), 1, 12, 800, 2000, 0, 1, "龙宫大弟子"},
		// ---------- 支线 ----------
		{"送外卖", "张二妈想吃热包子，收集3个猪肉包给她送去。", "collect", itemID("猪肉包"), 3, 2, 150, 400, 0, 2, "张二妈"},
		{"蟹将作乱", "蟹将在滩涂作乱，击败5名蟹将。", "hunt", npcID("蟹将"), 5, 13, 1500, 3500, 0, 2, "渔夫海生"},
		{"西瓜精之患", "瓜田里出了西瓜精，摘除6只！", "hunt", npcID("西瓜精"), 6, 18, 2000, 4500, 0, 2, "小兰"},
		{"喽罗清剿", "喽罗们集结作乱，清剿10个。", "hunt", npcID("喽罗"), 10, 18, 2200, 5000, 0, 2, "古董老板"},
		{"野味尝鲜", "众酒客想尝鲜，收集5串冰糖葫芦助兴。", "collect", itemID("冰糖葫芦"), 5, 15, 800, 1800, 0, 2, "众酒客"},
		{"野蛮丫头", "野蛮丫头又在撒野，去击败6个让她安分点。", "hunt", npcID("野蛮丫头"), 6, 12, 1300, 2800, 0, 2, "萧晓月"},
		// ---------- 日常 ----------
		{"每日巡逻", "【日常】城郊巡逻，击败10只野狼。", "hunt", npcID("野狼"), 10, 5, 400, 900, 0, 3, "村长"},
		{"每日运镖", "【日常】帮船夫押一趟货，去见他领任务。", "talk", npcID("船夫"), 1, 5, 300, 800, 0, 3, "船夫"},
		{"每日采买", "【日常】店小二缺人手，收集5个素菜包。", "collect", itemID("素菜包"), 5, 5, 350, 800, 0, 3, "店小二"},
		{"每日缉盗", "【日常】小流氓又出来偷摸，教训10个。", "hunt", npcID("小流氓"), 10, 10, 800, 1800, 0, 3, "李捕头"},
		{"每日海防", "【日常】海防吃紧，击败12名虾兵。", "hunt", npcID("虾兵"), 12, 25, 3000, 6000, 0, 3, "龙宫大弟子"},
	}
	// 重新灌入：任务不足 15 条视为旧版种子，整表重建（连带清空玩家进度）
	var cnt int64
	db.Model(&model.HxxyQuest{}).Count(&cnt)
	if cnt >= 15 {
		return
	}
	db.Exec("DELETE FROM hxxy_player_quests")
	db.Exec("DELETE FROM hxxy_quests")
	rows := []model.HxxyQuest{}
	for i, s := range specs {
		if s.target == 0 {
			continue
		}
		q := model.HxxyQuest{ID: uint(i + 1), Name: s.name, Desc: s.desc, Type: s.typ, Category: s.cat,
			TargetID: s.target, Count: s.count, MinLevel: s.minLv, ExpReward: s.exp, MoneyReward: s.money}
		if fn := npcID(s.from); fn > 0 {
			q.FromNpc = fn
		}
		if s.next > 0 && s.next-1 < len(specs) && specs[s.next-1].target != 0 {
			q.NextQuest = uint(s.next)
		}
		rows = append(rows, q)
	}
	if len(rows) > 0 {
		db.Create(&rows)
		log.Printf("[hxxy] 任务灌入 %d 条", len(rows))
	}
}

func mustJSON(b []byte, v interface{}) {
	if err := json.Unmarshal(b, v); err != nil {
		log.Fatalf("[hxxy] JSON解析失败: %v", err)
	}
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	if n < 0 {
		return 0
	}
	return n
}
