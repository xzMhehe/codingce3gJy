package handler

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 背包/仓库/装备/商店/银行

// hxBagView 背包行视图
func (h *HxxyHandler) hxBagView(b *model.HxxyBag) gin.H {
	v := gin.H{"id": b.ID, "kind": b.Kind, "ref_id": b.RefID, "count": b.Count, "bind": b.Bind, "store": b.Store, "used": b.Used}
	if b.Kind == "item" {
		var it model.HxxyItem
		if err := h.DB.First(&it, b.RefID).Error; err == nil {
			v["name"] = it.Name
			v["desc"] = it.Desc
			v["category"] = it.Category
			v["level"] = it.Level
			v["price"] = it.Price
			v["bean_price"] = it.BeanPrice
			v["effect"] = it.Effect
		}
	} else {
		var e model.HxxyEquip
		if err := h.DB.First(&e, b.RefID).Error; err == nil {
			v["name"] = e.Name
			v["desc"] = e.Desc
			v["category"] = e.Category
			v["slot_name"] = hxSlotNames[e.Category]
			v["level"] = e.Level
			v["price"] = e.Price
			v["bean_price"] = e.BeanPrice
			v["sect"] = e.Sect
			v["attrs"] = gin.H{"hp": e.HP, "atk": e.Atk, "mg": e.Mg, "def": e.Def,
				"bg": e.Bg, "hg": e.Hg, "lg": e.Lg, "bf": e.Bf, "hf": e.Hf, "lf": e.Lf}
			v["extra"] = hxJSONEx(b.Extra)
		}
	}
	return v
}

// hxJSONEx 装备附加信息
func hxJSONEx(extra string) gin.H {
	out := gin.H{"star": 0, "holes": 0, "gems": []uint{}}
	if extra != "" {
		var ex struct {
			Star  int    `json:"star"`
			Holes int    `json:"holes"`
			Gems  []uint `json:"gems"`
		}
		if json.Unmarshal([]byte(extra), &ex) == nil {
			out["star"] = ex.Star
			out["holes"] = ex.Holes
			if ex.Gems == nil {
				ex.Gems = []uint{}
			}
			out["gems"] = ex.Gems
		}
	}
	return out
}

// Bag 背包/仓库列表
func (h *HxxyHandler) Bag(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	store := 0
	if c.Query("store") == "1" {
		store = 1
	}
	var rows []model.HxxyBag
	h.DB.Where("player_id = ? AND store = ?", p.ID, store).Order("kind, ref_id").Find(&rows)
	items := []gin.H{}
	for i := range rows {
		items = append(items, h.hxBagView(&rows[i]))
	}
	resp.OK(c, gin.H{"player": h.hxPlayerBrief(p), "store": store, "items": items,
		"cap": p.BagCap, "used": len(items)})
}

// BagDiscard 丢弃
func (h *HxxyHandler) BagDiscard(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		BagID uint `json:"bag_id"`
		Count int  `json:"count"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.BagID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	if in.Count <= 0 {
		in.Count = 1
	}
	// 已穿装备不可丢
	for _, bid := range h.hxEqBagIDs(p) {
		if bid == in.BagID {
			resp.ParamError(c, "已穿戴的装备请先卸下")
			return
		}
	}
	if h.hxBagSub(p.ID, in.BagID, in.Count) {
		resp.OK(c, gin.H{"msg": "丢弃成功"})
		return
	}
	resp.ParamError(c, "物品不存在或数量不足")
}

// BagUse 使用物品（药品/丹药/卷轴/礼包）
func (h *HxxyHandler) BagUse(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		BagID uint `json:"bag_id"`
		Dtx   int  `json:"dtx"` // 腾云符目标
		Dty   int  `json:"dty"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.BagID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var b model.HxxyBag
	if err := h.DB.Where("id = ? AND player_id = ? AND store = 0 AND kind = 'item'", in.BagID, p.ID).First(&b).Error; err != nil {
		resp.ParamError(c, "物品不存在")
		return
	}
	var it model.HxxyItem
	if err := h.DB.First(&it, b.RefID).Error; err != nil {
		resp.ParamError(c, "物品数据异常")
		return
	}
	eff := map[string]interface{}{}
	if it.Effect != "" {
		json.Unmarshal([]byte(it.Effect), &eff)
	}
	if len(eff) == 0 {
		resp.ParamError(c, "该物品无法直接使用")
		return
	}
	getInt := func(k string) int { v, _ := eff[k].(float64); return int(v) }
	msg := ""

	// 丹药每日限用
	daily := getInt("daily")
	if daily > 0 && b.Used >= daily {
		resp.ParamError(c, fmt.Sprintf("【%s】今日已用 %d 个，达到每日限用", it.Name, daily))
		return
	}

	consume := true
	switch {
	case getInt("hp") > 0 || getInt("mp") > 0:
		a := h.hxAttrs(p)
		p.HP += getInt("hp")
		if p.HP > a.MaxHP {
			p.HP = a.MaxHP
		}
		p.MP += getInt("mp")
		if p.MP > a.MaxMP {
			p.MP = a.MaxMP
		}
		h.DB.Model(&model.HxxyPlayer{}).Where("id = ?", p.ID).Updates(map[string]interface{}{"hp": p.HP, "mp": p.MP})
		msg = fmt.Sprintf("使用了【%s】", it.Name)
	case getInt("maxhp") > 0 || getInt("maxmp") > 0 || getInt("atk") > 0 || getInt("def") > 0 || getInt("mg") > 0:
		// 丹药永久加成 → 存头衔式加成：并入住宅家具表简化实现（每人独立记录）
		msg = h.hxPillBonus(p, &it, eff)
	case getInt("full") == 1:
		a := h.hxAttrs(p)
		h.DB.Model(&model.HxxyPlayer{}).Where("id = ?", p.ID).Updates(map[string]interface{}{"hp": a.MaxHP, "mp": a.MaxMP})
		p.HP, p.MP = a.MaxHP, a.MaxMP
		msg = "【万能果】下肚，气血法力全部恢复！"
	case getInt("vip") > 0:
		h.DB.Model(&model.HxxyPlayer{}).Where("id = ?", p.ID).Update("vip", p.Vip+getInt("vip"))
		p.Vip += getInt("vip")
		msg = fmt.Sprintf("获得 %d 分钟VIP练级祝福！", getInt("vip"))
	case getInt("beans") > 0:
		h.hxWallet(p, "beans", int64(getInt("beans")), "使用金豆物品")
		msg = fmt.Sprintf("获得 %d 金豆！", getInt("beans"))
	case getInt("box") == 1:
		msg = h.hxOpenBox(p)
	case eff["goto"] != nil:
		target, _ := eff["goto"].(string)
		parts := strings.SplitN(target, "_", 2)
		if len(parts) == 2 {
			x := atoiH(parts[0])
			y := atoiH(parts[1])
			h.DB.Model(&model.HxxyPlayer{}).Where("id = ?", p.ID).Updates(map[string]interface{}{"map_x": x, "map_y": y})
			p.MapX, p.MapY = x, y
			msg = "一阵白光，你回到了" + h.hxNodeName(x, y)
		}
	case getInt("skill") > 0:
		var cnt int64
		h.DB.Model(&model.HxxyPlayerSkill{}).Where("player_id = ? AND skill_id = ?", p.ID, getInt("skill")).Count(&cnt)
		if cnt > 0 {
			consume = false
			msg = "你已经学会了该技能"
		} else {
			h.DB.Create(&model.HxxyPlayerSkill{PlayerID: p.ID, SkillID: uint(getInt("skill")), Level: 1})
			var sk model.HxxySkill
			h.DB.First(&sk, getInt("skill"))
			msg = "你学会了【" + sk.Name + "】！"
		}
	case getInt("skill_sect") == 1:
		msg = h.hxLearnSectSkills(p)
	case getInt("teleport") == 1:
		if h.hxNode(in.Dtx, in.Dty) != nil {
			h.DB.Model(&model.HxxyPlayer{}).Where("id = ?", p.ID).Updates(map[string]interface{}{"map_x": in.Dtx, "map_y": in.Dty})
			p.MapX, p.MapY = in.Dtx, in.Dty
			msg = "腾云驾雾，你来到了" + h.hxNodeName(in.Dtx, in.Dty)
		} else {
			resp.ParamError(c, "请选择传送目的地")
			return
		}
	default:
		resp.ParamError(c, "该物品暂时无法使用")
		return
	}

	if consume {
		if b.Count == 1 {
			h.DB.Delete(&model.HxxyBag{}, b.ID)
		} else {
			h.DB.Model(&model.HxxyBag{}).Where("id = ?", b.ID).Updates(map[string]interface{}{"count": b.Count - 1, "used": b.Used + 1})
		}
	}
	resp.OK(c, gin.H{"msg": msg, "player": h.hxPlayerBrief(p)})
}

// atoiH 简单转换
func atoiH(s string) int {
	n := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0
		}
		n = n*10 + int(ch-'0')
	}
	return n
}

// hxPillBonus 丹药永久加成（并入家具 JSON，与住宅加成同源）
func (h *HxxyHandler) hxPillBonus(p *model.HxxyPlayer, it *model.HxxyItem, eff map[string]interface{}) string {
	field := ""
	val := 0
	for k, target := range map[string]string{"maxhp": "hp", "maxmp": "mp", "atk": "atk", "def": "def", "mg": "mg"} {
		if v, ok := eff[k].(float64); ok && v > 0 {
			field = target
			val = int(v)
			break
		}
	}
	if field == "" {
		return "该丹药暂时无法使用"
	}
	var house model.HxxyHouse
	if err := h.DB.Where("player_id = ?", p.ID).First(&house).Error; err != nil {
		house = model.HxxyHouse{PlayerID: p.ID, Furniture: "[]"}
		h.DB.Create(&house)
	}
	var fs []map[string]interface{}
	json.Unmarshal([]byte(house.Furniture), &fs)
	fs = append(fs, map[string]interface{}{"id": 0, "name": it.Name, "bonus": field, "val": val})
	nb, _ := json.Marshal(fs)
	h.DB.Model(&model.HxxyHouse{}).Where("id = ?", house.ID).Update("furniture", string(nb))
	return fmt.Sprintf("服下【%s】，%s永久 +%d！", it.Name, map[string]string{"hp": "气血", "mp": "法力", "atk": "攻击", "def": "防御", "mg": "魔攻"}[field], val)
}

// hxOpenBox 开宝箱（随机奖励）
func (h *HxxyHandler) hxOpenBox(p *model.HxxyPlayer) string {
	r := 1 + rand.Intn(100)
	switch {
	case r <= 40:
		m := int64(100 + rand.Intn(900))
		h.hxWallet(p, "money", m, "开宝箱")
		return fmt.Sprintf("宝箱哗啦一响，获得 %d 银两！", m)
	case r <= 60:
		n := 1 + rand.Intn(5)
		h.hxWallet(p, "beans", int64(n), "开宝箱")
		return fmt.Sprintf("宝箱里金光一闪，获得 %d 金豆！", n)
	case r <= 80:
		var it model.HxxyItem
		if err := h.DB.Where("category = 5").Order("RAND()").First(&it).Error; err == nil {
			h.hxBagAdd(p, "item", it.ID, 1, it.Bind)
			return fmt.Sprintf("获得【%s】×1！", it.Name)
		}
	default:
		var eq model.HxxyEquip
		if err := h.DB.Where("category BETWEEN 3 AND 8").Order("RAND()").First(&eq).Error; err == nil {
			h.hxBagAdd(p, "equip", eq.ID, 1, eq.Bind)
			return fmt.Sprintf("宝箱底部竟藏着【%s】！", eq.Name)
		}
	}
	return "宝箱空空如也……"
}

// hxLearnSectSkills 门派秘籍：学习本门派可用技能
func (h *HxxyHandler) hxLearnSectSkills(p *model.HxxyPlayer) string {
	var skills []model.HxxySkill
	h.DB.Where("category = 1 AND (sect = 0 OR sect = ?)", p.Sect).Find(&skills)
	learned := 0
	for _, s := range skills {
		var cnt int64
		h.DB.Model(&model.HxxyPlayerSkill{}).Where("player_id = ? AND skill_id = ?", p.ID, s.ID).Count(&cnt)
		if cnt == 0 {
			h.DB.Create(&model.HxxyPlayerSkill{PlayerID: p.ID, SkillID: s.ID, Level: 1})
			learned++
		}
	}
	return fmt.Sprintf("研读秘籍，你掌握了 %d 个新技能！", learned)
}

// ---------- 仓库 ----------

// Warehouse 仓库
func (h *HxxyHandler) Warehouse(c *gin.Context) {
	h.Bag(c)
}

// WhDeposit 存入仓库
func (h *HxxyHandler) WhDeposit(c *gin.Context) {
	h.hxStoreMove(c, 1, "存入")
}

// WhTakeout 取出
func (h *HxxyHandler) WhTakeout(c *gin.Context) {
	h.hxStoreMove(c, 0, "取出")
}

func (h *HxxyHandler) hxStoreMove(c *gin.Context, target int, verb string) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		BagID uint `json:"bag_id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.BagID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	// 已穿装备不可存
	if target == 1 {
		for _, bid := range h.hxEqBagIDs(p) {
			if bid == in.BagID {
				resp.ParamError(c, "已穿戴的装备请先卸下")
				return
			}
		}
	}
	var cnt int64
	h.DB.Model(&model.HxxyBag{}).Where("id = ? AND player_id = ?", in.BagID, p.ID).Count(&cnt)
	if cnt == 0 {
		resp.ParamError(c, "物品不存在")
		return
	}
	h.DB.Model(&model.HxxyBag{}).Where("id = ?", in.BagID).Update("store", target)
	resp.OK(c, gin.H{"msg": verb + "成功"})
}

// ---------- 装备 ----------

// EquipWear 穿戴装备（校验等级/门派，同部位自动替换）
func (h *HxxyHandler) EquipWear(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		BagID uint `json:"bag_id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.BagID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var b model.HxxyBag
	if err := h.DB.Where("id = ? AND player_id = ? AND kind = 'equip' AND store = 0", in.BagID, p.ID).First(&b).Error; err != nil {
		resp.ParamError(c, "装备不存在")
		return
	}
	var e model.HxxyEquip
	if err := h.DB.First(&e, b.RefID).Error; err != nil {
		resp.ParamError(c, "装备数据异常")
		return
	}
	if e.Category < 1 || e.Category > 8 {
		resp.ParamError(c, "该装备不可穿戴")
		return
	}
	if p.Level < e.Level {
		resp.ParamError(c, fmt.Sprintf("需要 %d 级才能穿戴【%s】", e.Level, e.Name))
		return
	}
	// 门派限制：0/6/7 视为无限制
	if e.Sect >= 1 && e.Sect <= 5 && e.Sect != p.Sect {
		resp.ParamError(c, fmt.Sprintf("【%s】为%s专属", e.Name, hxSectNames[e.Sect]))
		return
	}
	// 卸下同部位旧装备
	old := map[int]*uint{1: &p.EqSlot1, 2: &p.EqSlot2, 3: &p.EqSlot3, 4: &p.EqSlot4,
		5: &p.EqSlot5, 6: &p.EqSlot6, 7: &p.EqSlot7, 8: &p.EqSlot8}[e.Category]
	if *old != 0 {
		h.DB.Model(&model.HxxyPlayer{}).Where("id = ?", p.ID).Update(fmt.Sprintf("eq_slot%d", e.Category), 0)
	}
	h.DB.Model(&model.HxxyPlayer{}).Where("id = ?", p.ID).Update(fmt.Sprintf("eq_slot%d", e.Category), b.ID)
	*old = b.ID
	resp.OK(c, gin.H{"msg": fmt.Sprintf("穿上了【%s】", e.Name), "player": h.hxPlayerBrief(p)})
}

// EquipTakeoff 卸下
func (h *HxxyHandler) EquipTakeoff(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		Slot int `json:"slot"` // 1-8
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.Slot < 1 || in.Slot > 8 {
		resp.ParamError(c, "参数错误")
		return
	}
	cur := map[int]*uint{1: &p.EqSlot1, 2: &p.EqSlot2, 3: &p.EqSlot3, 4: &p.EqSlot4,
		5: &p.EqSlot5, 6: &p.EqSlot6, 7: &p.EqSlot7, 8: &p.EqSlot8}[in.Slot]
	if *cur == 0 {
		resp.ParamError(c, "该部位没有装备")
		return
	}
	h.DB.Model(&model.HxxyPlayer{}).Where("id = ?", p.ID).Update(fmt.Sprintf("eq_slot%d", in.Slot), 0)
	*cur = 0
	resp.OK(c, gin.H{"msg": "卸下了" + hxSlotNames[in.Slot], "player": h.hxPlayerBrief(p)})
}

// EquipUpgrade 装备强化（星级，费用随星级与等级增长）
func (h *HxxyHandler) EquipUpgrade(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		BagID uint `json:"bag_id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.BagID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var b model.HxxyBag
	if err := h.DB.Where("id = ? AND player_id = ? AND kind = 'equip'", in.BagID, p.ID).First(&b).Error; err != nil {
		resp.ParamError(c, "装备不存在")
		return
	}
	var e model.HxxyEquip
	h.DB.First(&e, b.RefID)
	ex := map[string]interface{}{}
	if b.Extra != "" {
		json.Unmarshal([]byte(b.Extra), &ex)
	}
	star := 0
	if v, ok := ex["star"].(float64); ok {
		star = int(v)
	}
	if star >= 10 {
		resp.ParamError(c, "已强化到最高星级")
		return
	}
	cost := int64((star+1)*(star+1)*500 + e.Level*100)
	if p.Money < cost {
		resp.ParamError(c, fmt.Sprintf("强化需要 %d 银两，银两不足", cost))
		return
	}
	h.hxWallet(p, "money", -cost, "装备强化")
	ex["star"] = star + 1
	nb, _ := json.Marshal(ex)
	h.DB.Model(&model.HxxyBag{}).Where("id = ?", b.ID).Update("extra", string(nb))
	resp.OK(c, gin.H{"msg": fmt.Sprintf("【%s】强化成功！当前 %d 星（每星属性+10%%）", e.Name, star+1)})
}

// EquipHole 装备打孔（最多3孔）
func (h *HxxyHandler) EquipHole(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		BagID uint `json:"bag_id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.BagID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var b model.HxxyBag
	if err := h.DB.Where("id = ? AND player_id = ? AND kind = 'equip'", in.BagID, p.ID).First(&b).Error; err != nil {
		resp.ParamError(c, "装备不存在")
		return
	}
	ex := map[string]interface{}{}
	if b.Extra != "" {
		json.Unmarshal([]byte(b.Extra), &ex)
	}
	holes := 0
	if v, ok := ex["holes"].(float64); ok {
		holes = int(v)
	}
	if holes >= 3 {
		resp.ParamError(c, "已经打满 3 个孔了")
		return
	}
	cost := int64((holes + 1) * 2000)
	if p.Money < cost {
		resp.ParamError(c, fmt.Sprintf("打孔需要 %d 银两，银两不足", cost))
		return
	}
	h.hxWallet(p, "money", -cost, "装备打孔")
	ex["holes"] = holes + 1
	nb, _ := json.Marshal(ex)
	h.DB.Model(&model.HxxyBag{}).Where("id = ?", b.ID).Update("extra", string(nb))
	resp.OK(c, gin.H{"msg": fmt.Sprintf("打孔成功！当前 %d/3 孔", holes+1)})
}

// EquipGem 镶嵌宝石（消耗背包中分类2物品）
func (h *HxxyHandler) EquipGem(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		BagID  uint `json:"bag_id"`  // 装备
		GemBag uint `json:"gem_bag"` // 宝石背包行
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.BagID == 0 || in.GemBag == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var b model.HxxyBag
	if err := h.DB.Where("id = ? AND player_id = ? AND kind = 'equip'", in.BagID, p.ID).First(&b).Error; err != nil {
		resp.ParamError(c, "装备不存在")
		return
	}
	var g model.HxxyBag
	if err := h.DB.Where("id = ? AND player_id = ? AND kind = 'item' AND store = 0", in.GemBag, p.ID).First(&g).Error; err != nil {
		resp.ParamError(c, "宝石不存在")
		return
	}
	var it model.HxxyItem
	h.DB.First(&it, g.RefID)
	if it.Category != 2 {
		resp.ParamError(c, "该物品不是宝石")
		return
	}
	ex := map[string]interface{}{}
	if b.Extra != "" {
		json.Unmarshal([]byte(b.Extra), &ex)
	}
	holes := 0
	if v, ok := ex["holes"].(float64); ok {
		holes = int(v)
	}
	gems := []uint{}
	if v, ok := ex["gems"].([]interface{}); ok {
		for _, gv := range v {
			if f, ok := gv.(float64); ok {
				gems = append(gems, uint(f))
			}
		}
	}
	if len(gems) >= holes {
		resp.ParamError(c, "先打孔才能镶嵌（孔数不足）")
		return
	}
	gems = append(gems, it.ID)
	ex["gems"] = gems
	nb, _ := json.Marshal(ex)
	h.DB.Model(&model.HxxyBag{}).Where("id = ?", b.ID).Update("extra", string(nb))
	h.hxBagSub(p.ID, g.ID, 1)
	resp.OK(c, gin.H{"msg": fmt.Sprintf("镶嵌【%s】成功！", it.Name)})
}

// ---------- 商店 ----------

// Shop 商店（虚拟商店按 kind 分类；复刻原版买药/买装备）
// kind: medicine 药店 weapon 武器店 armor 防具店 jewel 首饰店 grocery 杂货铺 pet 宠物店
func (h *HxxyHandler) Shop(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	kind := c.Param("kind")
	goods := []gin.H{}
	switch kind {
	case "medicine":
		var its []model.HxxyItem
		h.DB.Where("category = 5").Order("id").Limit(60).Find(&its)
		for _, it := range its {
			goods = append(goods, gin.H{"kind": "item", "ref_id": it.ID, "name": it.Name, "level": it.Level, "price": it.Price, "bean_price": it.BeanPrice, "desc": it.Desc})
		}
	case "weapon", "armor", "jewel":
		cats := map[string][]int{"weapon": {3}, "armor": {4, 5, 6}, "jewel": {7, 8}}[kind]
		var eqs []model.HxxyEquip
		h.DB.Where("category IN ? AND (sect IN (0,6,7) OR sect = ?)", cats, p.Sect).Order("level, id").Limit(120).Find(&eqs)
		for _, e := range eqs {
			goods = append(goods, gin.H{"kind": "equip", "ref_id": e.ID, "name": e.Name, "level": e.Level, "price": e.Price, "bean_price": e.BeanPrice, "desc": e.Desc, "sect": e.Sect, "category": e.Category})
		}
	case "grocery":
		var its []model.HxxyItem
		h.DB.Where("category IN (1,4,8)").Order("id").Limit(60).Find(&its)
		for _, it := range its {
			goods = append(goods, gin.H{"kind": "item", "ref_id": it.ID, "name": it.Name, "level": it.Level, "price": it.Price, "bean_price": it.BeanPrice, "desc": it.Desc})
		}
	case "pet":
		var its []model.HxxyItem
		h.DB.Where("id IN (3,4)").Find(&its) // 宠物指南/门派秘籍
		for _, it := range its {
			goods = append(goods, gin.H{"kind": "item", "ref_id": it.ID, "name": it.Name, "level": it.Level, "price": it.Price, "bean_price": it.BeanPrice, "desc": it.Desc})
		}
		var sps []model.HxxyPetSpecies
		h.DB.Where("level <= ?", p.Level+5).Order("level").Limit(15).Find(&sps)
		pets := []gin.H{}
		for _, sp := range sps {
			pets = append(pets, gin.H{"species_id": sp.ID, "name": sp.Name, "level": sp.Level, "price": sp.Level * 5000, "bean_price": sp.Level*2 + 10})
		}
		resp.OK(c, gin.H{"kind": kind, "goods": goods, "pets": pets})
		return
	default:
		resp.ParamError(c, "没有这个商店")
		return
	}
	resp.OK(c, gin.H{"kind": kind, "goods": goods})
}

// ShopBuy 购买（银两/金豆）
func (h *HxxyHandler) ShopBuy(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		Kind     string `json:"kind"` // item/equip/pet
		RefID    uint   `json:"ref_id"`
		Count    int    `json:"count"`
		Currency string `json:"currency"` // money/beans
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.RefID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	if in.Count <= 0 {
		in.Count = 1
	}
	if in.Currency == "" {
		in.Currency = "money"
	}

	switch in.Kind {
	case "pet":
		var sp model.HxxyPetSpecies
		if err := h.DB.First(&sp, in.RefID).Error; err != nil {
			resp.ParamError(c, "没有这个宠物")
			return
		}
		if p.Beans < sp.Level*2+10 {
			resp.ParamError(c, "金豆不足")
			return
		}
		h.hxWallet(p, "beans", int64(-(sp.Level*2 + 10)), "购买宠物【"+sp.Name+"】")
		h.DB.Create(&model.HxxyPet{PlayerID: p.ID, SpeciesID: sp.ID, Name: sp.Name, Level: sp.Level, Star: 1, Quality: 1})
		resp.OK(c, gin.H{"msg": fmt.Sprintf("成功购买宠物【%s】！快去宠物页面查看吧。", sp.Name)})
		return
	default:
		var name string
		var price, beanPrice, lvl, bind int
		if in.Kind == "equip" {
			var e model.HxxyEquip
			if err := h.DB.First(&e, in.RefID).Error; err != nil {
				resp.ParamError(c, "没有这个装备")
				return
			}
			name, price, beanPrice, lvl, bind = e.Name, e.Price, e.BeanPrice, e.Level, e.Bind
			if p.Level < lvl {
				resp.ParamError(c, fmt.Sprintf("需要 %d 级才能装备", lvl))
				return
			}
		} else {
			var it model.HxxyItem
			if err := h.DB.First(&it, in.RefID).Error; err != nil {
				resp.ParamError(c, "没有这个物品")
				return
			}
			name, price, beanPrice, lvl, bind = it.Name, it.Price, it.BeanPrice, it.Level, it.Bind
		}
		total := int64(0)
		if in.Currency == "beans" {
			if beanPrice <= 0 {
				resp.ParamError(c, "该商品不支持金豆购买")
				return
			}
			total = int64(beanPrice * in.Count)
			if p.Beans < int(total) {
				resp.ParamError(c, "金豆不足")
				return
			}
		} else {
			if price <= 0 {
				resp.ParamError(c, "该商品不支持银两购买")
				return
			}
			total = int64(price * in.Count)
			if p.Money < total {
				resp.ParamError(c, fmt.Sprintf("需要 %d 银两，银两不足", total))
				return
			}
		}
		h.hxWallet(p, in.Currency, -total, "购买【"+name+"】×"+fmt.Sprint(in.Count))
		h.hxBagAdd(p, in.Kind, in.RefID, in.Count, bind)
		cur := "银两"
		if in.Currency == "beans" {
			cur = "金豆"
		}
		resp.OK(c, gin.H{"msg": fmt.Sprintf("购买【%s】×%d 成功！花费 %d %s。", name, in.Count, total, cur)})
	}
}

// ShopSell 出售（半价银两）
func (h *HxxyHandler) ShopSell(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		BagID uint `json:"bag_id"`
		Count int  `json:"count"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.BagID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	if in.Count <= 0 {
		in.Count = 1
	}
	for _, bid := range h.hxEqBagIDs(p) {
		if bid == in.BagID {
			resp.ParamError(c, "已穿戴的装备请先卸下")
			return
		}
	}
	var b model.HxxyBag
	if err := h.DB.Where("id = ? AND player_id = ? AND store = 0", in.BagID, p.ID).First(&b).Error; err != nil {
		resp.ParamError(c, "物品不存在")
		return
	}
	if b.Count < in.Count {
		resp.ParamError(c, "数量不足")
		return
	}
	price := 0
	if b.Kind == "item" {
		var it model.HxxyItem
		h.DB.First(&it, b.RefID)
		price = it.Price
	} else {
		var e model.HxxyEquip
		h.DB.First(&e, b.RefID)
		price = e.Price
	}
	total := int64(price) / 2 * int64(in.Count)
	h.hxBagSub(p.ID, in.BagID, in.Count)
	h.hxWallet(p, "money", total, "出售物品")
	resp.OK(c, gin.H{"msg": fmt.Sprintf("出售成功，获得 %d 银两。", total)})
}

// ---------- 银行 ----------

// Bank 银行信息
func (h *HxxyHandler) Bank(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	resp.OK(c, gin.H{"money": p.Money, "bank": p.Bank})
}

// BankDeposit 存款
func (h *HxxyHandler) BankDeposit(c *gin.Context) {
	h.hxBankMove(c, 1)
}

// BankWithdraw 取款
func (h *HxxyHandler) BankWithdraw(c *gin.Context) {
	h.hxBankMove(c, -1)
}

func (h *HxxyHandler) hxBankMove(c *gin.Context, dir int) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		Amount int64 `json:"amount"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.Amount <= 0 {
		resp.ParamError(c, "请输入正确的金额")
		return
	}
	if dir == 1 {
		if p.Money < in.Amount {
			resp.ParamError(c, "身上银两不足")
			return
		}
		h.hxWallet(p, "money", -in.Amount, "银行存款")
		h.hxWallet(p, "bank", in.Amount, "银行存款")
		resp.OK(c, gin.H{"msg": fmt.Sprintf("存入 %d 银两。", in.Amount), "money": p.Money, "bank": p.Bank})
	} else {
		if p.Bank < in.Amount {
			resp.ParamError(c, "存款不足")
			return
		}
		h.hxWallet(p, "bank", -in.Amount, "银行取款")
		h.hxWallet(p, "money", in.Amount, "银行取款")
		resp.OK(c, gin.H{"msg": fmt.Sprintf("取出 %d 银两。", in.Amount), "money": p.Money, "bank": p.Bank})
	}
}
