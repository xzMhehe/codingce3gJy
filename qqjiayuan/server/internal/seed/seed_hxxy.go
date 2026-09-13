package seed

import (
	_ "embed"
	"encoding/json"
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

//go:embed hxxy_data/npcs.json
var hxxyNpcsJSON []byte

//go:embed hxxy_data/spawns.json
var hxxySpawnsJSON []byte

//go:embed hxxy_data/map_npcs.json
var hxxyMapNpcsJSON []byte

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
	NodeID uint   `json:"node_id"`
	Dtx    int    `json:"dtx"`
	Dty    int    `json:"dty"`
	Name   string `json:"name"`
	Desc   string `json:"desc"`
	Up     string `json:"up"`
	Down   string `json:"down"`
	Left   string `json:"left"`
	Right  string `json:"right"`
	UpJump string `json:"up_jump"`
	DownJump string `json:"down_jump"`
	LeftJump string `json:"left_jump"`
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
	Dtx      int    `json:"dtx"`
	Dty      int    `json:"dty"`
	NpcID    uint   `json:"npc_id"`
	Name     string `json:"name"`
	Img      string `json:"img"`
	Dialogue string `json:"dialogue"`
	Shop     string `json:"shop"`
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

	// ---------- 副本定义 ----------
	if hxxyCount(db, &model.HxxyDungeon{}) == 0 {
		rows := []model.HxxyDungeon{
			{ID: 1, Name: "大雁塔", Desc: "长安城内的大雁塔，塔中妖气缭绕，逐层挑战！", Floors: 10, Daily: 2, MinLevel: 10},
			{ID: 2, Name: "小雁塔", Desc: "小雁塔中藏有佛门至宝，层层把关。", Floors: 10, Daily: 2, MinLevel: 20},
			{ID: 3, Name: "兵马俑", Desc: "秦始皇陵兵马俑，千年佣兵苏醒，杀气冲天！", Floors: 15, Daily: 2, MinLevel: 30},
			{ID: 4, Name: "碑林", Desc: "碑林深处藏有上古碑灵，以文入武。", Floors: 15, Daily: 2, MinLevel: 40},
			{ID: 5, Name: "冰风谷", Desc: "终年寒风的冰封山谷，传说谷底有上古凶兽！", Floors: 20, Daily: 2, MinLevel: 50},
		}
		db.Create(&rows)
		log.Printf("[hxxy] 副本定义灌入 %d 条", len(rows))
	}

	// ---------- 新手任务（按 NPC/物品名动态查找，找不到就跳过） ----------
	if hxxyCount(db, &model.HxxyQuest{}) == 0 {
		hxxySeedQuests(db)
	}
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

// hxxySeedQuests 新手任务链（按名字查 NPC/物品，缺失即跳过该条）
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
	}
	specs := []qd{
		{"初入江湖", "击败新手村外的山贼，证明你的实力！", "hunt", npcID("山贼"), 5, 1, 200, 500, 2},
		{"小试身手", "再去击败5个强盗，声名渐起。", "hunt", npcID("强盗"), 5, 1, 300, 800, 3},
		{"拜见村长", "回去拜见村长，领取赏赐。", "talk", npcID("村长"), 1, 1, 100, 300, 4},
		{"药材收购", "药店缺药材，收集5株小幸运草。", "collect", itemID("小幸运草"), 5, 1, 250, 600, 0},
	}
	rows := []model.HxxyQuest{}
	for i, s := range specs {
		if s.target == 0 {
			continue
		}
		q := model.HxxyQuest{ID: uint(i + 1), Name: s.name, Desc: s.desc, Type: s.typ, TargetID: s.target, Count: s.count, MinLevel: s.minLv, ExpReward: s.exp, MoneyReward: s.money}
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
