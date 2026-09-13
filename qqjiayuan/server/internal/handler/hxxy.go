package handler

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 幻想西游（复刻 PHP WAP 版"幻想西游"，全玩法）
// 公式照抄原版 ztt.php / ini_pz04.php / ltpk03.php / cwztt.php：
// - 属性: maxhp=(lv+15)²×K(门派系数) maxmp=⌈(lv+30)(lv+20)/4⌉ atk=(lv+1)(lv+2)×K+300 def=(lv+1)²×K+200 mg=(lv+1)(lv+2)×K+300
// - 升级: 所需经验 (lv+1)³(lv+2)+200；修炼经验上限 (lv+1)⁴(lv+2)+100
// - 伤害: 元素差优势×1.3 / 劣势×1.1 / 均势×1.2，10%暴击，最终 rand(s/2,s)；法术门派以魔攻作攻击
// 独立货币：银两 money / 金豆 beans；玩家 user_id 关联 users 表
type HxxyHandler struct{ DB *gorm.DB }

var hxSectNames = map[int]string{1: "将军府", 2: "龙宫", 3: "月宫", 4: "方寸山", 5: "普陀山"}
var hxSlotNames = map[int]string{1: "法宝", 2: "坐骑", 3: "武器", 4: "护甲", 5: "头盔", 6: "靴子", 7: "项链", 8: "手镯"}

// hxSectK 门派成长系数 K（照抄 ztt.php）
func hxSectK(sect int) (hpK, atkK, defK, mgK float64) {
	switch sect {
	case 1: // 将军府
		return 2, 4, 2, 3
	case 2: // 龙宫
		return 2, 3, 3, 3
	case 3: // 月宫
		return 4, 3, 2, 3
	case 4: // 方寸山
		return 2, 3, 2, 4
	default: // 普陀山
		return 3, 3, 2, 4
	}
}

// hxMagicSect 法术门派（以魔攻作攻击）
func hxMagicSect(sect int) bool { return sect == 3 || sect == 4 || sect == 5 }

// hxCombatAttr 战斗属性（基础公式+装备+头衔+住宅家具）
type hxCombatAttr struct {
	Name    string `json:"name"`
	Level   int    `json:"level"`
	MaxHP   int    `json:"max_hp"`
	MaxMP   int    `json:"max_mp"`
	Atk     int    `json:"atk"`  // 攻击（法术门派=魔攻）
	Mg      int    `json:"mg"`   // 魔攻
	Def     int    `json:"def"`
	Mf      int    `json:"mf"`
	Bg      int    `json:"bg"` // 冰攻
	Hg      int    `json:"hg"`
	Lg      int    `json:"lg"`
	Bf      int    `json:"bf"`
	Hf      int    `json:"hf"`
	Lf      int    `json:"lf"`
}

// hxAttrs 计算玩家战斗属性（基础公式 + 装备 + 头衔 + 家具）
func (h *HxxyHandler) hxAttrs(p *model.HxxyPlayer) hxCombatAttr {
	hpK, atkK, defK, mgK := hxSectK(p.Sect)
	lv := float64(p.Level)
	a := hxCombatAttr{Name: p.Name, Level: p.Level}
	a.MaxHP = int(math.Pow(lv+15, 2) * hpK)
	a.MaxMP = int(math.Ceil((lv + 30) * (lv + 20) / 4))
	a.Atk = int((lv+1)*(lv+2)*atkK) + 300
	a.Def = int(math.Pow(lv+1, 2)*defK) + 200
	a.Mg = int((lv+1)*(lv+2)*mgK) + 300

	// 装备加成（8 部位，含星级加成：每星 +10% 基础属性）
	for _, bid := range h.hxEqBagIDs(p) {
		if bid == 0 {
			continue
		}
		var b model.HxxyBag
		if err := h.DB.First(&b, bid).Error; err != nil || b.Kind != "equip" {
			continue
		}
		var e model.HxxyEquip
		if err := h.DB.First(&e, b.RefID).Error; err != nil {
			continue
		}
		star := 0
		var ex struct {
			Star int     `json:"star"`
			Gems []uint  `json:"gems"`
		}
		if b.Extra != "" {
			json.Unmarshal([]byte(b.Extra), &ex)
			star = ex.Star
		}
		mul := 1.0 + 0.1*float64(star)
		a.MaxHP += int(float64(e.HP)*mul)
		a.Atk += int(float64(e.Atk)*mul)
		a.Mg += int(float64(e.Mg)*mul)
		a.Def += int(float64(e.Def)*mul)
		a.Bg += e.Bg
		a.Hg += e.Hg
		a.Lg += e.Lg
		a.Bf += e.Bf
		a.Hf += e.Hf
		a.Lf += e.Lf
		// 镶嵌宝石加成
		for _, gid := range ex.Gems {
			a.hxGemBonus(h.DB, gid)
		}
	}
	// 头衔加成
	if p.TitleID > 0 {
		var t model.HxxyTitle
		if err := h.DB.First(&t, p.TitleID).Error; err == nil {
			a.MaxHP += t.HP
			a.Atk += t.Atk
			a.Def += t.Def
			a.Mg += t.Mg
		}
	}
	// 住宅家具加成
	var house model.HxxyHouse
	if err := h.DB.Where("player_id = ?", p.ID).First(&house).Error; err == nil && house.Furniture != "" {
		var fs []struct {
			ID   uint   `json:"id"`
			Name string `json:"name"`
			Bonus string `json:"bonus"`
			Val  int    `json:"val"`
		}
		if json.Unmarshal([]byte(house.Furniture), &fs) == nil {
			for _, f := range fs {
				switch f.Bonus {
				case "hp":
					a.MaxHP += f.Val
				case "atk":
					a.Atk += f.Val
				case "def":
					a.Def += f.Val
				case "mg":
					a.Mg += f.Val
				}
			}
		}
	}
	// 元素攻防基础 = 等级（照抄原版：元素攻防=等级+装备加成）
	a.Bg += p.Level
	a.Hg += p.Level
	a.Lg += p.Level
	a.Bf += p.Level
	a.Hf += p.Level
	a.Lf += p.Level
	// 法术门派：攻击值取魔攻（照抄 ltpk03.php）
	if hxMagicSect(p.Sect) {
		a.Atk = a.Mg
	}
	a.Mf = a.Def
	return a
}

// hxGemBonus 宝石加成（红=hp 蓝=mp 黄=atk 绿=def 白=mg，名字里的一~五为档位）
func (a *hxCombatAttr) hxGemBonus(db *gorm.DB, gid uint) {
	var it model.HxxyItem
	if err := db.First(&it, gid).Error; err != nil || it.Category != 2 {
		return
	}
	tier := 1
	for i, c := range []string{"一", "二", "三", "四", "五"} {
		if strings.Contains(it.Name, c) {
			tier = i + 1
		}
	}
	switch {
	case strings.Contains(it.Name, "红"):
		a.MaxHP += 100 * tier
	case strings.Contains(it.Name, "蓝"):
		a.MaxMP += 80 * tier
	case strings.Contains(it.Name, "黄"):
		a.Atk += 50 * tier
		a.Mg += 50 * tier
	case strings.Contains(it.Name, "绿"):
		a.Def += 50 * tier
	case strings.Contains(it.Name, "白"):
		a.Mg += 60 * tier
	}
}

// hxxySettingGet 读服务器配置（settings 键值表）
func (h *HxxyHandler) hxxySettingGet(key string) string {
	var val string
	h.DB.Model(&model.Setting{}).Select("`value`").Where("`key` = ?", key).Scan(&val)
	return val
}

// HxxyMaintGate 服务器维护拦截（复刻原版关服维护：维护中所有游戏接口统一返回维护公告）
func (h *HxxyHandler) HxxyMaintGate() gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.hxxySettingGet("hxxy_maintenance") == "1" {
			notice := strings.TrimSpace(h.hxxySettingGet("hxxy_maintenance_notice"))
			if notice == "" {
				notice = "服务器维护中，请稍后再来"
			}
			resp.ParamError(c, "【服务器维护中】"+notice)
			c.Abort()
			return
		}
		c.Next()
	}
}

// hxPlayer 取当前玩家（未建角返回 nil）
func (h *HxxyHandler) hxPlayer(c *gin.Context) *model.HxxyPlayer {
	uid := middleware.GetUID(c)
	var p model.HxxyPlayer
	if err := h.DB.Where("user_id = ?", uid).First(&p).Error; err != nil {
		return nil
	}
	// 每日计数归零
	today := time.Now().Format("2006-01-02")
	if p.DayDate != today {
		h.DB.Model(&model.HxxyPlayer{}).Where("id = ?", p.ID).Updates(map[string]interface{}{
			"day_date": today, "day_signin": 0, "day_battle": 0, "day_dungeon": 0, "day_arena": 0})
		p.DayDate, p.DaySignin, p.DayBattle, p.DayDungeon, p.DayArena = today, 0, 0, 0, 0
	}
	return &p
}

// hxExpNeed 升级所需经验（照抄 ini_pz04.php：lv 升 lv+1 需 (lv+1)³(lv+2)+200）
func hxExpNeed(lv int) int {
	l := float64(lv)
	return int(math.Pow(l+1, 3)*(l+2)) + 200
}

// hxXiulianCap 修炼经验上限（(lv+1)⁴(lv+2)+100）
func hxXiulianCap(lv int) int {
	l := float64(lv)
	return int(math.Pow(l+1, 4)*(l+2)) + 100
}

// hxGainExp 加经验+升级判定（升级回满血蓝）
func (h *HxxyHandler) hxGainExp(p *model.HxxyPlayer, exp int) (levels int, msg string) {
	if exp <= 0 {
		return 0, ""
	}
	p.Exp += exp
	for p.Exp >= hxExpNeed(p.Level) {
		p.Exp -= hxExpNeed(p.Level)
		p.Level++
		levels++
	}
	if levels > 0 {
		a := h.hxAttrs(p)
		p.HP = a.MaxHP
		p.MP = a.MaxMP
		msg = fmt.Sprintf("恭喜升级！当前等级 %d 级，气血法力已回满。", p.Level)
	}
	h.DB.Model(&model.HxxyPlayer{}).Where("id = ?", p.ID).Updates(map[string]interface{}{
		"level": p.Level, "exp": p.Exp, "hp": p.HP, "mp": p.MP})
	return
}

// hxWallet 货币变动+流水
func (h *HxxyHandler) hxWallet(p *model.HxxyPlayer, currency string, amount int64, reason string) {
	var balance int64
	switch currency {
	case "money":
		p.Money += amount
		balance = p.Money
		h.DB.Model(&model.HxxyPlayer{}).Where("id = ?", p.ID).Update("money", p.Money)
	case "beans":
		p.Beans += int(amount)
		balance = int64(p.Beans)
		h.DB.Model(&model.HxxyPlayer{}).Where("id = ?", p.ID).Update("beans", p.Beans)
	case "bank":
		p.Bank += amount
		balance = p.Bank
		h.DB.Model(&model.HxxyPlayer{}).Where("id = ?", p.ID).Update("bank", p.Bank)
	default:
		return
	}
	if amount != 0 {
		h.DB.Create(&model.HxxyWalletLog{PlayerID: p.ID, Currency: currency, Amount: amount, Balance: balance, Reason: reason})
	}
}

// hxBagAdd 背包加入（物品同 ref 合并；装备独立行）
func (h *HxxyHandler) hxBagAdd(p *model.HxxyPlayer, kind string, refID uint, count int, bind int) *model.HxxyBag {
	if kind == "item" {
		var b model.HxxyBag
		if err := h.DB.Where("player_id = ? AND kind = 'item' AND ref_id = ? AND store = 0", p.ID, refID).First(&b).Error; err == nil {
			h.DB.Model(&model.HxxyBag{}).Where("id = ?", b.ID).Update("count", gorm.Expr("count + ?", count))
			b.Count += count
			return &b
		}
	}
	b := model.HxxyBag{PlayerID: p.ID, Kind: kind, RefID: refID, Count: count, Bind: bind}
	if kind == "equip" {
		b.Extra = `{"star":0,"holes":0,"gems":[]}`
	}
	h.DB.Create(&b)
	return &b
}

// hxBagSub 背包扣除（where=0 背包内），减到 0 删除
func (h *HxxyHandler) hxBagSub(playerID, bagID uint, count int) bool {
	var b model.HxxyBag
	if err := h.DB.Where("id = ? AND player_id = ? AND store = 0", bagID, playerID).First(&b).Error; err != nil {
		return false
	}
	if b.Count < count {
		return false
	}
	if b.Count == count {
		h.DB.Delete(&model.HxxyBag{}, b.ID)
	} else {
		h.DB.Model(&model.HxxyBag{}).Where("id = ?", b.ID).Update("count", b.Count-count)
	}
	return true
}

// hxEqBagIDs 已穿装备 bag id 列表（按部位 1-8）
func (h *HxxyHandler) hxEqBagIDs(p *model.HxxyPlayer) []uint {
	return []uint{p.EqSlot1, p.EqSlot2, p.EqSlot3, p.EqSlot4, p.EqSlot5, p.EqSlot6, p.EqSlot7, p.EqSlot8}
}

// hxNode 查询地图节点
func (h *HxxyHandler) hxNode(dtx, dty int) *model.HxxyMapNode {
	var n model.HxxyMapNode
	if err := h.DB.Where("dtx = ? AND dty = ?", dtx, dty).First(&n).Error; err != nil {
		return nil
	}
	return &n
}

// hxDirText 方向值转显示文本（"1_33" → 出口目标名）
func (h *HxxyHandler) hxDirText(v string) gin.H {
	if v == "" {
		return nil
	}
	parts := strings.SplitN(v, "_", 2)
	if len(parts) != 2 {
		return nil
	}
	x, _ := strconv.Atoi(parts[0])
	y, _ := strconv.Atoi(parts[1])
	n := h.hxNode(x, y)
	name := v
	if n != nil {
		name = n.Name
	}
	return gin.H{"dir": v, "name": name, "jump": false}
}

//
// ---------- 入口 ----------
//

// Status 游戏入口状态（是否已建角）
func (h *HxxyHandler) Status(c *gin.Context) {
	uid := middleware.GetUID(c)
	var p model.HxxyPlayer
	if err := h.DB.Where("user_id = ?", uid).First(&p).Error; err != nil {
		resp.OK(c, gin.H{"has_player": false, "sects": h.hxSectList()})
		return
	}
	// 管理员封号检查
	if p.BanUntil > time.Now().Unix() {
		resp.ParamError(c, "你的账号已被封禁" + h.hxBanTip(p.BanUntil) + "，如有疑问请联系管理员")
		return
	}
	resp.OK(c, gin.H{"has_player": true, "player": h.hxPlayerBrief(&p)})
}

// hxBanTip 封禁剩余时长文案
func (h *HxxyHandler) hxBanTip(until int64) string {
	if until >= 4102444800 {
		return "（永久）"
	}
	mins := (until - time.Now().Unix()) / 60
	if mins >= 60*24 {
		return fmt.Sprintf("（剩余 %d 天）", mins/60/24)
	}
	if mins >= 60 {
		return fmt.Sprintf("（剩余 %d 小时）", mins/60)
	}
	return fmt.Sprintf("（剩余 %d 分钟）", mins)
}

// hxSectList 门派介绍（文案复刻原版 xy295.php，过滤广告）
func (h *HxxyHandler) hxSectList() []gin.H {
	return []gin.H{
		{"id": 1, "name": "将军府", "sex": 0, "desc": "大唐开国元勋程咬金所创，门下弟子骁勇善战，以刚猛的物理攻击闻名三界。"},
		{"id": 2, "name": "龙宫", "sex": 0, "desc": "四海龙王坐镇的水下宫殿，弟子防御出众，坚如磐石。"},
		{"id": 3, "name": "月宫", "sex": 2, "desc": "嫦娥仙子居住的广寒宫，只收女弟子，法术飘逸，气血充沛。"},
		{"id": 4, "name": "方寸山", "sex": 0, "desc": "菩提祖师道场，斜月三星洞中修得一身玄妙法术，魔攻冠绝三界。"},
		{"id": 5, "name": "普陀山", "sex": 1, "desc": "观音大士的道场，只收男弟子，法力精深，普度众生。"},
	}
}

// hxPlayerBrief 玩家简要信息
func (h *HxxyHandler) hxPlayerBrief(p *model.HxxyPlayer) gin.H {
	a := h.hxAttrs(p)
	node := h.hxNode(p.MapX, p.MapY)
	nodeName := "未知之地"
	if node != nil {
		nodeName = node.Name
	}
	return gin.H{
		"id": p.ID, "name": p.Name, "sex": p.Sex, "sect": p.Sect, "sect_name": hxSectNames[p.Sect],
		"level": p.Level, "exp": p.Exp, "exp_need": hxExpNeed(p.Level),
		"hp": p.HP, "max_hp": a.MaxHP, "mp": p.MP, "max_mp": a.MaxMP,
		"money": p.Money, "bank": p.Bank, "beans": p.Beans,
		"vip": p.Vip, "title_id": p.TitleID,
		"bag_cap": p.BagCap, "wh_cap": p.WhCap, "emz": p.Emz,
		"map_x": p.MapX, "map_y": p.MapY, "node_name": nodeName,
		"xiulian_switch": p.XiulianSwitch, "xiulian_exp": p.XiulianExp, "xiulian_cap": hxXiulianCap(p.Level),
		"fighting_pet": h.hxFightingPet(p.ID),
	}
}

// hxFightingPet 参战宠物
func (h *HxxyHandler) hxFightingPet(playerID uint) *model.HxxyPet {
	var pet model.HxxyPet
	if err := h.DB.Where("player_id = ? AND fighting = 1", playerID).First(&pet).Error; err != nil {
		return nil
	}
	return &pet
}

// Create 建角（性别校验：月宫限女，普陀山限男）
func (h *HxxyHandler) Create(c *gin.Context) {
	uid := middleware.GetUID(c)
	var in struct {
		Name string `json:"name"`
		Sex  int    `json:"sex"`
		Sect int    `json:"sect"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.Name == "" {
		resp.ParamError(c, "请填写角色名")
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if len([]rune(in.Name)) > 12 {
		resp.ParamError(c, "角色名最多 12 个字")
		return
	}
	if in.Sex != 1 && in.Sex != 2 {
		in.Sex = 1
	}
	if _, ok := hxSectNames[in.Sect]; !ok {
		resp.ParamError(c, "请选择门派")
		return
	}
	if in.Sect == 3 && in.Sex != 2 {
		resp.ParamError(c, "月宫只收女弟子")
		return
	}
	if in.Sect == 5 && in.Sex != 1 {
		resp.ParamError(c, "普陀山只收男弟子")
		return
	}
	var exist int64
	h.DB.Model(&model.HxxyPlayer{}).Where("user_id = ?", uid).Count(&exist)
	if exist > 0 {
		resp.ParamError(c, "你已创建过角色")
		return
	}
	var nameCnt int64
	h.DB.Model(&model.HxxyPlayer{}).Where("name = ?", in.Name).Count(&nameCnt)
	if nameCnt > 0 {
		resp.ParamError(c, "角色名已被使用")
		return
	}
	p := model.HxxyPlayer{
		UserID: uid, Name: in.Name, Sex: in.Sex, Sect: in.Sect,
		Level: 1, HP: 100, MP: 100, Money: 1000, Beans: 10,
		MapX: 0, MapY: 0, // 新手村·村长家
		DayDate: time.Now().Format("2006-01-02"),
	}
	a := h.hxAttrs(&p)
	p.HP, p.MP = a.MaxHP, a.MaxMP
	if err := h.DB.Create(&p).Error; err != nil {
		resp.ServerError(c, err)
		return
	}
	// 初始技能：普攻 + 本门派技能
	h.DB.Exec("INSERT INTO hxxy_player_skills (player_id, skill_id, level) VALUES (?, 1, 1)", p.ID)
	h.DB.Exec("INSERT INTO hxxy_player_skills (player_id, skill_id, level) SELECT ?, id, 1 FROM hxxy_skills WHERE category = 1 AND (sect = 0 OR sect = ?) AND id <> 1", p.ID, p.Sect)
	// 初始装备：本门派 1 级武器（若存在）
	var eq model.HxxyEquip
	if err := h.DB.Where("category = 3 AND level <= 1 AND (sect IN (0,6,7) OR sect = ?)", p.Sect).Order("level DESC, id").First(&eq).Error; err == nil {
		b := h.hxBagAdd(&p, "equip", eq.ID, 1, 0)
		h.DB.Model(&model.HxxyPlayer{}).Where("id = ?", p.ID).Update("eq_slot3", b.ID)
	}
	resp.OK(c, gin.H{"msg": "创建角色成功！欢迎来到幻想西游的世界。", "player": h.hxPlayerBrief(&p)})
}

// State 当前地图状态（节点+出口+刷怪+回城）
func (h *HxxyHandler) State(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	node := h.hxNode(p.MapX, p.MapY)
	if node == nil {
		resp.ParamError(c, "地图数据异常，请联系管理员")
		return
	}
	// 本节点刷怪（精确节点优先，无则从区域池随机取 3）
	var spawns []model.HxxySpawn
	h.DB.Where("dtx = ? AND dty = ?", p.MapX, p.MapY).Order("id").Limit(6).Find(&spawns)
	if len(spawns) == 0 {
		var pool []model.HxxySpawn
		h.DB.Where("dtx = ? AND dty = 0", p.MapX).Find(&pool)
		rand.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
		if len(pool) > 3 {
			pool = pool[:3]
		}
		spawns = pool
	}
	enemies := []gin.H{}
	for _, s := range spawns {
		var npc model.HxxyNpc
		if err := h.DB.First(&npc, s.NpcID).Error; err != nil || npc.Kind != 1 {
			continue
		}
		enemies = append(enemies, gin.H{
			"npc_id": npc.ID, "name": npc.Name, "level": npc.Level,
			"difficulty": s.Difficulty, "take": npc.Take,
		})
	}
	// 本节点功能NPC（传送/商店/对话，复刻原版 fznpc）
	npcList := h.hxMapNpcList(p.MapX, p.MapY)
	resp.OK(c, gin.H{
		"player": h.hxPlayerBrief(p),
		"node": gin.H{
			"node_id": node.ID, "name": node.Name, "desc": node.Desc,
			"up": h.hxDirText(node.Up), "down": h.hxDirText(node.Down),
			"left": h.hxDirText(node.Left), "right": h.hxDirText(node.Right),
			"up_jump": h.hxDirJump(node.UpJump), "down_jump": h.hxDirJump(node.DownJump),
			"left_jump": h.hxDirJump(node.LeftJump), "right_jump": h.hxDirJump(node.RightJump),
		},
		"enemies": enemies,
		"npcs":    npcList,
	})
}

// hxMapNpcList 节点上的功能NPC列表
func (h *HxxyHandler) hxMapNpcList(dtx, dty int) []gin.H {
	var rows []model.HxxyMapNpc
	h.DB.Where("dtx = ? AND dty = ?", dtx, dty).Order("id").Find(&rows)
	list := []gin.H{}
	for _, r := range rows {
		var teles []hxTeleOpt
		if r.Teles != "" {
			json.Unmarshal([]byte(r.Teles), &teles)
		}
		list = append(list, gin.H{
			"id": r.ID, "npc_id": r.NpcID, "name": r.Name, "img": r.Img,
			"dialogue": r.Dialogue, "shop": r.Shop, "tele_count": len(teles),
		})
	}
	return list
}

// hxTeleOpt 传送目的地
type hxTeleOpt struct {
	Name string `json:"name"`
	Dtx  int    `json:"dtx"`
	Dty  int    `json:"dty"`
}

// NpcView 功能NPC交互视图（复刻原版 npc.php：图片+对话+动作链接）
func (h *HxxyHandler) NpcView(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	id, _ := strconv.Atoi(c.Param("id"))
	var row model.HxxyMapNpc
	if err := h.DB.First(&row, id).Error; err != nil || row.Dtx != p.MapX || row.Dty != p.MapY {
		resp.ParamError(c, "这里没有这个人")
		return
	}
	var teles []hxTeleOpt
	if row.Teles != "" {
		json.Unmarshal([]byte(row.Teles), &teles)
	}
	level := 0
	if row.NpcID > 0 {
		var npc model.HxxyNpc
		if err := h.DB.First(&npc, row.NpcID).Error; err == nil {
			level = npc.Level
		}
	}
	resp.OK(c, gin.H{
		"id": row.ID, "npc_id": row.NpcID, "name": row.Name, "img": row.Img,
		"dialogue": row.Dialogue, "shop": row.Shop, "level": level,
		"teles": teles,
	})
}

// NpcTeleport 功能NPC传送（复刻原版 xy020.php：校验该NPC在此节点提供此传送）
func (h *HxxyHandler) NpcTeleport(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		NpcID uint `json:"npc_id"`
		Dtx   int  `json:"dtx"`
		Dty   int  `json:"dty"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.NpcID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var row model.HxxyMapNpc
	if err := h.DB.First(&row, in.NpcID).Error; err != nil || row.Dtx != p.MapX || row.Dty != p.MapY {
		resp.ParamError(c, "这里没有这个人")
		return
	}
	var teles []hxTeleOpt
	if row.Teles != "" {
		json.Unmarshal([]byte(row.Teles), &teles)
	}
	ok := false
	for _, t := range teles {
		if t.Dtx == in.Dtx && t.Dty == in.Dty {
			ok = true
			break
		}
	}
	if !ok || h.hxNode(in.Dtx, in.Dty) == nil {
		resp.ParamError(c, "这个人不会带你去那里")
		return
	}
	h.DB.Model(&model.HxxyPlayer{}).Where("id = ?", p.ID).Updates(map[string]interface{}{"map_x": in.Dtx, "map_y": in.Dty})
	p.MapX, p.MapY = in.Dtx, in.Dty
	resp.OK(c, gin.H{"msg": row.Name + "把你送到了" + h.hxNodeName(in.Dtx, in.Dty), "player": h.hxPlayerBrief(p)})
}

// hxDirJump 跳转链接（带 jump 标记，前端显示"传送"）
func (h *HxxyHandler) hxDirJump(v string) gin.H {
	if v == "" {
		return nil
	}
	t := h.hxDirText(v)
	if t == nil {
		return nil
	}
	t["jump"] = true
	return t
}

// Move 走路（up/down/left/right）
func (h *HxxyHandler) Move(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		Dir string `json:"dir"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	node := h.hxNode(p.MapX, p.MapY)
	if node == nil {
		resp.ParamError(c, "地图数据异常")
		return
	}
	var target string
	switch in.Dir {
	case "up":
		target = node.Up
	case "down":
		target = node.Down
	case "left":
		target = node.Left
	case "right":
		target = node.Right
	default:
		resp.ParamError(c, "方向错误")
		return
	}
	if target == "" {
		resp.ParamError(c, "这个方向没有路")
		return
	}
	parts := strings.SplitN(target, "_", 2)
	if len(parts) != 2 {
		resp.ParamError(c, "道路不通")
		return
	}
	x, _ := strconv.Atoi(parts[0])
	y, _ := strconv.Atoi(parts[1])
	h.DB.Model(&model.HxxyPlayer{}).Where("id = ?", p.ID).Updates(map[string]interface{}{"map_x": x, "map_y": y})
	p.MapX, p.MapY = x, y
	resp.OK(c, gin.H{"msg": "你来到了" + h.hxNodeName(x, y), "player": h.hxPlayerBrief(p)})
}

// Jump 传送（*_jump 链接 / 腾云符）
func (h *HxxyHandler) Jump(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		Dtx int `json:"dtx"`
		Dty int `json:"dty"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.Dtx < 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	if h.hxNode(in.Dtx, in.Dty) == nil {
		resp.ParamError(c, "传送目的地不存在")
		return
	}
	h.DB.Model(&model.HxxyPlayer{}).Where("id = ?", p.ID).Updates(map[string]interface{}{"map_x": in.Dtx, "map_y": in.Dty})
	p.MapX, p.MapY = in.Dtx, in.Dty
	resp.OK(c, gin.H{"msg": "腾云驾雾，你来到了" + h.hxNodeName(in.Dtx, in.Dty), "player": h.hxPlayerBrief(p)})
}

func (h *HxxyHandler) hxNodeName(x, y int) string {
	n := h.hxNode(x, y)
	if n == nil {
		return "未知之地"
	}
	return n.Name
}

// Attrs 战斗属性明细（复刻原版状态页：基础+装备+头衔）
func (h *HxxyHandler) Attrs(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	a := h.hxAttrs(p)
	// 已穿装备明细
	equips := []gin.H{}
	for slot, bid := range h.hxEqBagIDs(p) {
		if bid == 0 {
			continue
		}
		var b model.HxxyBag
		if err := h.DB.First(&b, bid).Error; err != nil {
			continue
		}
		var e model.HxxyEquip
		if err := h.DB.First(&e, b.RefID).Error; err != nil {
			continue
		}
		equips = append(equips, gin.H{
			"slot": slot + 1, "slot_name": hxSlotNames[slot+1], "bag_id": b.ID,
			"name": e.Name, "level": e.Level, "star": hxExtraStar(b.Extra),
		})
	}
	resp.OK(c, gin.H{
		"player": h.hxPlayerBrief(p),
		"attrs":  a,
		"equips": equips,
		"sect_name": hxSectNames[p.Sect],
	})
}

// hxExtraStar 从装备 Extra JSON 取星级
func hxExtraStar(extra string) int {
	if extra == "" {
		return 0
	}
	var ex struct {
		Star int `json:"star"`
	}
	json.Unmarshal([]byte(extra), &ex)
	return ex.Star
}

// Rest 住宿恢复（银两 = 等级×10 回满血蓝）
func (h *HxxyHandler) Rest(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	cost := int64(p.Level * 10)
	if p.Money < cost {
		resp.ParamError(c, fmt.Sprintf("住宿需要 %d 银两，银两不足", cost))
		return
	}
	a := h.hxAttrs(p)
	h.hxWallet(p, "money", -cost, "客栈住宿")
	h.DB.Model(&model.HxxyPlayer{}).Where("id = ?", p.ID).Updates(map[string]interface{}{"hp": a.MaxHP, "mp": a.MaxMP})
	p.HP, p.MP = a.MaxHP, a.MaxMP
	resp.OK(c, gin.H{"msg": fmt.Sprintf("美美睡了一觉，花费 %d 银两，气血法力全部恢复！", cost), "player": h.hxPlayerBrief(p)})
}
