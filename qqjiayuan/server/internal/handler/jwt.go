package handler

import (
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

// 精武堂（复刻 3GQQ wap 精武堂，全玩法）
// 成长闭环：初始 5 能量分配属性 -> 商店买药/装备 -> 修炼升级+能量 -> 技能书店学主动技能
// -> 排行选对手比武赚经验/荣誉/比武材料 -> 图纸+材料锻造高级装备 -> 头衔/帮派
type JwtHandler struct{ DB *gorm.DB }

//
// ---------- 内部工具 ----------
//

// jwtBrief 取用户全局货币(G币/元宝)+昵称
func (h *JwtHandler) jwtBrief(uid uint) *model.User {
	var u model.User
	h.DB.Select("id,username,nickname,coins,yuanbao").First(&u, uid)
	return &u
}

// jwtPlayer 懒创建玩家档案
func (h *JwtHandler) jwtPlayer(uid uint) *model.JwtPlayer {
	var p model.JwtPlayer
	if err := h.DB.Where("user_id = ?", uid).First(&p).Error; err != nil {
		u := h.jwtBrief(uid)
		p = model.JwtPlayer{UserID: uid, Level: 1, Energy: 5, Nick: u.Nickname,
			TaskDate: time.Now().Format("2006-01-02"), SignDate: time.Now().Format("2006-01-02")}
		h.DB.Create(&p)
	}
	return &p
}

// jwtUpdateTaskDate 每日归零任务计数
func (h *JwtHandler) jwtRoll(p *model.JwtPlayer) {
	today := time.Now().Format("2006-01-02")
	if p.TaskDate != today {
		h.DB.Model(&model.JwtPlayer{}).Where("id = ?", p.ID).Updates(map[string]interface{}{
			"task_date": today, "arena_cnt": 0, "arena_opps": "", "pill_cnt": 0,
			"skill_cnt": 0, "train_cnt": 0, "chat_cnt": 0, "reward_claim": 0})
		p.TaskDate, p.ArenaCnt, p.ArenaOpps, p.PillCnt, p.SkillCnt, p.TrainCnt, p.ChatCnt, p.RewardClaim =
			today, 0, "", 0, 0, 0, 0, 0
	}
	// 周签到换周清空
	y, wk := time.Now().ISOWeek()
	if p.SignDate != today {
		cur := strconv.Itoa(y) + "-w" + strconv.Itoa(wk)
		if !strings.Contains(p.SignDate, cur) {
			h.DB.Model(&model.JwtPlayer{}).Where("id = ?", p.ID).Update("sign_week", "")
			p.SignWeek = ""
			p.SignDate = cur
		}
		h.DB.Model(&model.JwtPlayer{}).Where("id = ?", p.ID).Update("sign_date", today)
	}
}

// jwtItemByID 查询物品
func (h *JwtHandler) jwtItemByID(id uint) *model.JwtItem {
	var it model.JwtItem
	if err := h.DB.First(&it, id).Error; err != nil {
		return nil
	}
	return &it
}

// jwtBagCount 背包持有数
func (h *JwtHandler) jwtBagCount(uid uint, itemID uint) int {
	var b model.JwtBag
	if err := h.DB.Where("user_id = ? AND item_id = ?", uid, itemID).First(&b).Error; err != nil {
		return 0
	}
	return b.Amount
}

// jwtBagAdd 背包加数量（存在则累加）
func (h *JwtHandler) jwtBagAdd(uid uint, it *model.JwtItem, amount int) {
	var b model.JwtBag
	if err := h.DB.Where("user_id = ? AND item_id = ?", uid, it.ID).First(&b).Error; err == nil {
		h.DB.Model(&model.JwtBag{}).Where("id = ?", b.ID).Update("amount", gorm.Expr("amount + ?", amount))
	} else {
		h.DB.Create(&model.JwtBag{UserID: uid, ItemID: it.ID, Name: it.Name, Cat: it.Cat, Amount: amount})
	}
}

// jwtBagSub 背包减数量；减到 0 删除
func (h *JwtHandler) jwtBagSub(uid uint, itemID uint, amount int) bool {
	var b model.JwtBag
	if err := h.DB.Where("user_id = ? AND item_id = ?", uid, itemID).First(&b).Error; err != nil {
		return false
	}
	newAmt := b.Amount - amount
	if newAmt < 0 {
		return false
	}
	if newAmt == 0 {
		h.DB.Delete(&model.JwtBag{}, b.ID)
	} else {
		h.DB.Model(&model.JwtBag{}).Where("id = ?", b.ID).Update("amount", newAmt)
	}
	return true
}

// jwtTitleName 头衔显示
func jwtTitleName(tier int) string {
	titles := []string{"无名小卒", "少侠", "侠士", "大侠", "豪杰", "宗师", "霸者", "圣者", "武尊", "武王", "至尊战神"}
	if tier >= 0 && tier < len(titles) {
		return titles[tier]
	}
	return "无名小卒"
}

// jwtArenaOppsHas 判断今日已比武对手列表是否包含某玩家（逗号分隔）
func jwtArenaOppsHas(list string, uid uint) bool {
	for _, s := range strings.Split(list, ",") {
		if strings.TrimSpace(s) == "" {
			continue
		}
		if v, err := strconv.Atoi(strings.TrimSpace(s)); err == nil && uint(v) == uid {
			return true
		}
	}
	return false
}

// jwtSkillEffect 技能战报效果描述（复刻 比武.xhtml 叙事：造成XX效果）
func jwtSkillEffect(name string) string {
	effects := map[string]string{
		"排山倒海":  "造成中毒效果，每回合损失10点气血",
		"降龙十八掌": "掌力惊天，势不可挡",
		"潇湘剑雨":  "剑光如雨，连绵不绝",
	}
	if e, ok := effects[name]; ok {
		return e
	}
	return "威力惊人"
}

// jwtPassiveBonus 被动技能加成
func (h *JwtHandler) jwtPassiveBonus(uid uint) model.JwtSkill {
	var out model.JwtSkill
	var learned []model.JwtLearnedSkill
	h.DB.Where("user_id = ?", uid).Find(&learned)
	for _, l := range learned {
		var s model.JwtSkill
		if err := h.DB.First(&s, l.SkillID); err == nil && s.Act == 0 {
			out.PAtk += s.PAtk
			out.PDef += s.PDef
			out.PHp += s.PHp
			out.PMp += s.PMp
			out.Crit += s.Crit
		}
	}
	return out
}

// jwtCombat 合成战斗属性（基础+能量分配+装备+被动）
func (h *JwtHandler) jwtCombat(p *model.JwtPlayer) model.JwtCombat {
	ps := h.jwtPassiveBonus(p.UserID)
	eq := h.jwtEquipBonus(p)
	c := model.JwtCombat{}
	c.MaxHp = 20 + p.EHp*24 + p.Level*4 + eq.Hp + ps.PHp
	c.MaxMp = 10 + p.EMp*9 + p.Level*4 + eq.Mp + ps.PMp
	c.Speed = p.ESpd + p.Level/2 + eq.Spd
	c.Atk = p.EAtk + p.Level/2 + eq.Atk + ps.PAtk
	c.Def = p.EDef + p.Level/3 + eq.Def + ps.PDef
	c.Hit = 90 + eq.Hit
	c.Crit = 5 + eq.Crit + ps.Crit
	c.CritMul = 150
	c.Dodge = eq.Dodge
	return c
}

// jwtEquipBonus 装备加成（替换已穿上的装备按同名 id）
func (h *JwtHandler) jwtEquipBonus(p *model.JwtPlayer) model.JwtItem {
	var sum model.JwtItem
	slots := []uint{p.WeaponID, p.HelmetID, p.ArmorID, p.ShoesID, p.NecklaceID, p.BraceletID, p.RingID, p.MedalID}
	for _, id := range slots {
		if id == 0 {
			continue
		}
		if it := h.jwtItemByID(id); it != nil {
			sum.Atk += it.Atk
			sum.Def += it.Def
			sum.Hp += it.Hp
			sum.Mp += it.Mp
			sum.Spd += it.Spd
			sum.Hit += it.Hit
			sum.Crit += it.Crit
			sum.Dodge += it.Dodge
		}
	}
	return sum
}

// jwtGainExp 加经验并处理升级（每级+1能量）
func (h *JwtHandler) jwtGainExp(p *model.JwtPlayer, exp int) (levels int) {
	if exp <= 0 {
		return 0
	}
	p.Exp += exp
	levels = 0
	for p.Exp >= p.Level*64 {
		p.Exp -= p.Level * 64
		p.Level++
		p.Energy++
		levels++
	}
	h.DB.Model(&model.JwtPlayer{}).Where("id = ?", p.ID).Updates(map[string]interface{}{
		"level": p.Level, "exp": p.Exp, "energy": p.Energy})
	return
}

// jwtSlotOf 物品类目对应装备槽字段
func hrefJwtSlot(cat string) string {
	switch cat {
	case "weapon":
		return "weapon_id"
	case "helmet":
		return "helmet_id"
	case "armor":
		return "armor_id"
	case "shoes":
		return "shoes_id"
	case "necklace":
		return "necklace_id"
	case "bracelet":
		return "bracelet_id"
	case "ring":
		return "ring_id"
	case "medal":
		return "medal_id"
	}
	return ""
}

//
// ---------- 玩家属性 ----------
//

// View 我的属性：资料 + 战斗属性 + 装备栏 + 任务计数
func (h *JwtHandler) View(c *gin.Context) {
	uid := middleware.GetUID(c)
	p := h.jwtPlayer(uid)
	h.jwtRoll(p)
	// 新手一次性元宝：进入游戏即可在道具商店购买
	if p.Starter == 0 {
		h.DB.Model(&model.JwtPlayer{}).Where("id = ?", p.ID).Update("starter", 1)
		h.DB.Model(&model.User{}).Where("id = ?", uid).Update("yuanbao", gorm.Expr("yuanbao + 500"))
		addWalletLog(h.DB, uid, "jwt", "精武堂新手元宝", "jwt_yuanbao", 500)
		p.Starter = 1
	}
	u := h.jwtBrief(uid)
	cbt := h.jwtCombat(p)
	signed := jwtSignStatus(p.SignWeek)
	// 装备栏：name=槽位名，item_name=实际已装备的道具名（未装备 无）
	jwtSlots := []gin.H{}
	slotItemID := map[string]uint{"weapon_id": p.WeaponID, "helmet_id": p.HelmetID, "armor_id": p.ArmorID,
		"shoes_id": p.ShoesID, "necklace_id": p.NecklaceID, "bracelet_id": p.BraceletID,
		"ring_id": p.RingID, "medal_id": p.MedalID}
	for _, d := range []struct{ slot, label, idx string }{
		{"weapon", "武器", "weapon_id"}, {"helmet", "头盔", "helmet_id"}, {"armor", "盔甲", "armor_id"},
		{"shoes", "战鞋", "shoes_id"}, {"necklace", "项链", "necklace_id"}, {"bracelet", "手镯", "bracelet_id"},
		{"ring", "戒指", "ring_id"}, {"medal", "勋章", "medal_id"},
	} {
		itemID := slotItemID[d.idx]
		itemName := "无"
		if itemID != 0 {
			if it := h.jwtItemByID(itemID); it != nil {
				itemName = it.Name
			}
		}
		jwtSlots = append(jwtSlots, gin.H{"slot": d.slot, "name": d.label, "item_name": itemName, "item_id": itemID})
	}
	slotNames := map[string]string{"weapon_id": "武器", "helmet_id": "头盔", "armor_id": "盔甲", "shoes_id": "战鞋",
		"necklace_id": "项链", "bracelet_id": "手镯", "ring_id": "戒指", "medal_id": "勋章"}
	resp.OK(c, gin.H{
		"uid": uid, "nick": p.Nick, "coins": u.Coins, "yuanbao": u.YuanBao,
		"level": p.Level, "exp": p.Exp, "next_exp": p.Level * 64, "title": p.Title, "sex": p.Sex,
		"title_name": jwtTitleName(p.Title), "energy": p.Energy,
		"gang_id": p.GangID, "created": u.CreatedAt.Format("2006-01-02 15:04:05"),
		"combat": cbt,
		"cur_hp": func() int {
			if p.CurHp > 0 && p.CurHp <= cbt.MaxHp {
				return p.CurHp
			}
			return cbt.MaxHp
		}(),
		"cur_mp": func() int {
			if p.CurMp > 0 && p.CurMp <= cbt.MaxMp {
				return p.CurMp
			}
			return cbt.MaxMp
		}(),
		"alloc": gin.H{"hp": p.EHp, "mp": p.EMp, "spd": p.ESpd, "atk": p.EAtk, "def": p.EDef},
		"slots": jwtSlots, "slot_names": slotNames,
		"task": gin.H{"date": p.TaskDate, "arena": p.ArenaCnt, "pill": p.PillCnt,
			"skill": p.SkillCnt, "train": p.TrainCnt, "chat": p.ChatCnt, "reward_claim": p.RewardClaim},
		"sign": signed, "sign_week": p.SignWeek,
	})
}

// EnergyAlloc 能量分配
func (h *JwtHandler) EnergyAlloc(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Field string `json:"field" binding:"required"` // hp/mp/spd/atk/def
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	p := h.jwtPlayer(uid)
	if p.Energy <= 0 {
		resp.ParamError(c, "没有可用能量点，升级可获得能量")
		return
	}
	col := ""
	switch req.Field {
	case "hp":
		col = "e_hp"
	case "mp":
		col = "e_mp"
	case "spd":
		col = "e_spd"
	case "atk":
		col = "e_atk"
	case "def":
		col = "e_def"
	default:
		resp.ParamError(c, "属性字段错误")
		return
	}
	h.DB.Model(&model.JwtPlayer{}).Where("id = ?", p.ID).Update(col, gorm.Expr(col+" + 1"))
	h.DB.Model(&model.JwtPlayer{}).Where("id = ?", p.ID).Update("energy", gorm.Expr("energy - 1"))
	resp.OK(c, gin.H{"msg": "分配成功"})
}

//
// ---------- 商店 ----------
//

// Shop 道具商店（src=shop）
func (h *JwtHandler) Shop(c *gin.Context) {
	var items []model.JwtItem
	h.DB.Where("status = 1 AND src = 'shop'").Order("id ASC").Find(&items)
	out := []gin.H{}
	for _, it := range items {
		out = append(out, gin.H{"id": it.ID, "name": it.Name, "cat": it.Cat, "price": it.Price,
			"currency": it.Currency, "level": it.Level, "atk": it.Atk, "def": it.Def,
			"hp": it.Hp, "mp": it.Mp, "spd": it.Spd, "recover_hp": it.RecoverHp, "recover_mp": it.RecoverMp,
			"desc": it.Desc})
	}
	resp.OK(c, out)
}

// Buy 购买（G币/元宝）
func (h *JwtHandler) Buy(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		ID     uint `json:"id" binding:"required"`
		Amount int  `json:"amount" binding:"min=1,max=99"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if req.Amount == 0 {
		req.Amount = 1
	}
	it := h.jwtItemByID(req.ID)
	if it == nil || it.Src != "shop" || it.Status != 1 {
		resp.NotFound(c, "无此物品")
		return
	}
	p := h.jwtPlayer(uid)
	if it.Level > p.Level {
		resp.ParamError(c, "需要"+strconv.Itoa(it.Level)+"级才能购买")
		return
	}
	u := h.jwtBrief(uid)
	cost := it.Price * req.Amount
	if it.Currency == "yuanbao" {
		if u.YuanBao < cost {
			resp.ParamError(c, "元宝不足，需要"+strconv.Itoa(cost)+" 元宝")
			return
		}
		h.DB.Model(&model.User{}).Where("id = ?", uid).Update("yuanbao", gorm.Expr("yuanbao - ?", cost))
		addWalletLog(h.DB, uid, "jwt", "精武堂购买["+it.Name+"]", "jwt_yuanbao", -cost)
	} else {
		if u.Coins < cost {
			resp.ParamError(c, "G币不足，需要"+strconv.Itoa(cost)+" G币")
			return
		}
		h.DB.Model(&model.User{}).Where("id = ?", uid).Update("coins", gorm.Expr("coins - ?", cost))
		addWalletLog(h.DB, uid, "jwt", "精武堂购买["+it.Name+"]", "jwt_coins", -cost)
	}
	h.jwtBagAdd(uid, it, req.Amount)
	cur := "G币"
	if it.Currency == "yuanbao" {
		cur = "元宝"
	}
	resp.OK(c, gin.H{"msg": "购买成功", "name": it.Name, "amount": req.Amount, "total": cost, "currency": cur})
}

//
// ---------- 技能 ----------
//

// SkillShop 技能书店（复刻原站 技能书店.xhtml；已学技能附带当前等级/熟练度供 技能领悟 展示）
func (h *JwtHandler) SkillShop(c *gin.Context) {
	uid := middleware.GetUID(c)
	var skills []model.JwtSkill
	h.DB.Where("status = 1").Order("id ASC").Find(&skills)
	var learned []model.JwtLearnedSkill
	h.DB.Where("user_id = ?", uid).Find(&learned)
	learnedMap := map[uint]bool{}
	learnLv := map[uint]int{}
	learnPractice := map[uint]int{}
	for _, l := range learned {
		learnedMap[l.SkillID] = true
		learnLv[l.SkillID] = l.Level
		learnPractice[l.SkillID] = l.Practice
	}
	out := []gin.H{}
	for _, s := range skills {
		row := gin.H{"id": s.ID, "name": s.Name, "act": s.Act,
			"level": s.Level, "price": s.Price, "currency": s.Currency, "coef": s.Coef,
			"hit": s.Hit, "crit": s.Crit, "crit_mul": s.CritMul, "dodge_add": s.DodgeAdd,
			"p_atk": s.PAtk, "p_def": s.PDef, "p_hp": s.PHp, "p_mp": s.PMp,
			"desc": s.Desc, "learned": learnedMap[s.ID]}
		if learnedMap[s.ID] {
			row["lv"] = learnLv[s.ID]
			row["practice"] = learnPractice[s.ID]
		}
		out = append(out, row)
	}
	resp.OK(c, out)
}

// LearnSkill 学习技能
func (h *JwtHandler) LearnSkill(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		ID uint `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	p := h.jwtPlayer(uid)
	var s model.JwtSkill
	if err := h.DB.First(&s, req.ID).Error; err != nil {
		resp.NotFound(c, "无此技能")
		return
	}
	if s.Level > p.Level {
		resp.ParamError(c, "需要"+strconv.Itoa(s.Level)+"级才能学习")
		return
	}
	var n int64
	h.DB.Model(&model.JwtLearnedSkill{}).Where("user_id = ? AND skill_id = ?", uid, s.ID).Count(&n)
	if n > 0 {
		resp.ParamError(c, "已学会该技能")
		return
	}
	u := h.jwtBrief(uid)
	if s.Currency == "yuanbao" {
		if u.YuanBao < s.Price {
			resp.ParamError(c, "元宝不足")
			return
		}
		h.DB.Model(&model.User{}).Where("id = ?", uid).Update("yuanbao", gorm.Expr("yuanbao - ?", s.Price))
		addWalletLog(h.DB, uid, "jwt", "精武堂学习技能["+s.Name+"]", "jwt_yuanbao", -s.Price)
	} else {
		if u.Coins < s.Price {
			resp.ParamError(c, "G币不足，需要"+strconv.Itoa(s.Price)+" G币")
			return
		}
		h.DB.Model(&model.User{}).Where("id = ?", uid).Update("coins", gorm.Expr("coins - ?", s.Price))
		addWalletLog(h.DB, uid, "jwt", "精武堂学习技能["+s.Name+"]", "jwt_coins", -s.Price)
	}
	h.DB.Create(&model.JwtLearnedSkill{UserID: uid, SkillID: s.ID})
	h.DB.Model(&model.JwtPlayer{}).Where("id = ?", p.ID).Update("skill_cnt", gorm.Expr("skill_cnt + 1"))
	resp.OK(c, gin.H{"msg": "学习成功！"})
}

//
// ---------- 修炼 ----------
//

// Practice 修炼(开始/查询)。支持普通/加长8/加长24小时。
func (h *JwtHandler) Practice(c *gin.Context) {
	uid := middleware.GetUID(c)
	p := h.jwtPlayer(uid)
	now := time.Now()
	var req struct {
		Start bool   `json:"start"`
		Stop  bool   `json:"stop"`
		Type  string `json:"type"` // normal(4h)/long8(8h)/long24(24h)
	}
	_ = c.ShouldBindJSON(&req)

	// 满时修炼收益（按修炼模式）
	gainFull := func(totalH float64) int {
		switch {
		case totalH >= 23:
			return (20 + p.Level*2) * 6 // 24小时 = 6倍
		case totalH >= 7:
			return (20 + p.Level*2) * 2 // 8小时 = 2倍
		default:
			return 20 + p.Level*2 // 4小时 普通
		}
	}
	// settle 结算修炼：按已修炼时长比例给经验，技能点=经验（修炼5分钟内不获得经验）
	settle := func() (exp int, skill int) {
		if p.PracticeAt != nil && p.PracticeEnd != nil {
			totalH := p.PracticeEnd.Sub(*p.PracticeAt).Hours()
			elapsedH := now.Sub(*p.PracticeAt).Hours()
			full := gainFull(totalH)
			if elapsedH < 5.0/60.0 {
				exp = 0 // 5分钟内取消不获得任何经验
			} else {
				exp = int(float64(full) * elapsedH / totalH)
				if exp < 1 {
					exp = 1
				}
			}
			skill = exp
		}
		h.jwtGainExp(p, exp)
		h.DB.Model(&model.JwtPlayer{}).Where("id = ?", p.ID).Updates(map[string]interface{}{
			"practicing": 0, "practice_at": nil, "practice_end": nil,
			"skill_point": gorm.Expr("skill_point + ?", skill)})
		p.Practicing = 0
		return
	}

	// 到期自动结算（含技能点）
	if p.Practicing == 1 && p.PracticeEnd != nil && now.After(*p.PracticeEnd) {
		exp, skill := settle()
		resp.OK(c, gin.H{"msg": "修炼完成！经验+" + strconv.Itoa(exp) + "，技能点+" + strconv.Itoa(skill),
			"stop": true, "exp": exp, "skill": skill, "practicing": 0})
		return
	}
	// 手动停止修炼（复刻 停止修炼.xhtml：修炼结束！经验+X 技能点+X）
	if req.Stop {
		if p.Practicing != 1 {
			resp.ParamError(c, "当前未在修炼")
			return
		}
		exp, skill := settle()
		resp.OK(c, gin.H{"msg": "修炼结束！", "stop": true, "exp": exp, "skill": skill, "practicing": 0})
		return
	}
	// 开始修炼
	if req.Start {
		if p.Practicing == 1 {
			resp.ParamError(c, "已在修炼中，请先停止修炼")
			return
		}
		u := h.jwtBrief(uid)
		var end time.Time
		var cost int
		var typeName string
		switch req.Type {
		case "long8":
			end = now.Add(8 * time.Hour)
			cost = 50
			typeName = "加长修炼(8小时)"
		case "long24":
			end = now.Add(24 * time.Hour)
			cost = 120
			typeName = "加长修炼(24小时)"
		default: // normal
			end = now.Add(4 * time.Hour)
			cost = 0
			typeName = "普通修炼(4小时)"
		}
		if cost > 0 {
			if u.YuanBao < cost {
				resp.ParamError(c, "元宝不足，需要"+strconv.Itoa(cost)+"元宝")
				return
			}
			h.DB.Model(&model.User{}).Where("id = ?", uid).Update("yuanbao", gorm.Expr("yuanbao - ?", cost))
			addWalletLog(h.DB, uid, "jwt", "精武堂修炼加长时长", "jwt_yuanbao", -cost)
		}
		h.DB.Model(&model.JwtPlayer{}).Where("id = ?", p.ID).Updates(map[string]interface{}{
			"practicing": 1, "practice_at": now, "practice_end": end, "train_cnt": gorm.Expr("train_cnt + 1")})
		resp.OK(c, gin.H{"msg": "开始" + typeName + "成功！消耗体力20",
			"start": true, "type_name": typeName, "practicing": 1})
		return
	}

	// 查询修炼状态（练功房展示）
	remaining := 0
	if p.Practicing == 1 && p.PracticeEnd != nil {
		d := p.PracticeEnd.Sub(now)
		if d > 0 {
			remaining = int(d.Minutes())
		}
	}
	var practiceType string
	if p.Practicing == 1 && p.PracticeEnd != nil {
		dur := p.PracticeEnd.Sub(*p.PracticeAt).Hours()
		switch {
		case dur >= 23:
			practiceType = "加长修炼(24小时)"
		case dur >= 7:
			practiceType = "加长修炼(8小时)"
		default:
			practiceType = "普通修炼(4小时)"
		}
	} else {
		practiceType = "普通(4小时)"
	}
	resp.OK(c, gin.H{"practicing": p.Practicing, "remaining_min": remaining,
		"practice_type": practiceType, "practice_skill": p.SkillPoint})
}

//
// ---------- 比武 ----------
//

// ArenaList 比武排行榜/对手列表（真实家园玩家，不足则懒创建精武堂档案，不使用假机器人）
func (h *JwtHandler) ArenaList(c *gin.Context) {
	uid := middleware.GetUID(c)
	var players []model.JwtPlayer
	h.DB.Where("user_id <> ?", uid).Order("level DESC, honor DESC").Limit(30).Find(&players)
	// 真实玩家不足 8 人时，从家园已有用户里懒创建精武堂档案补齐（数据真实可点开查看）
	const want = 8
	if len(players) < want {
		short := want - len(players)
		var cand []uint
		h.DB.Raw(`SELECT u.id FROM users u
			LEFT JOIN jwt_players p ON p.user_id = u.id
			WHERE u.id <> ? AND p.id IS NULL
			ORDER BY u.level DESC, u.id ASC LIMIT ?`, uid, short).Scan(&cand)
		for _, cid := range cand {
			players = append(players, *h.jwtPlayer(cid))
		}
	}
	out := []gin.H{}
	for i, p := range players {
		cbt := h.jwtCombat(&p)
		curHp := p.CurHp
		if curHp < 1 || curHp > cbt.MaxHp {
			curHp = cbt.MaxHp
		}
		curMp := p.CurMp
		if curMp < 1 || curMp > cbt.MaxMp {
			curMp = cbt.MaxMp
		}
		out = append(out, gin.H{"uid": p.UserID, "nick": p.Nick, "level": p.Level, "honor": p.Honor,
			"title": jwtTitleName(p.Title), "max_hp": cbt.MaxHp, "max_mp": cbt.MaxMp, "atk": cbt.Atk, "def": cbt.Def,
			"cur_hp": curHp, "cur_mp": curMp, "bot": false, "idx": i})
	}
	resp.OK(c, out)
}

// Arena 比武
func (h *JwtHandler) Arena(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Uid   uint   `json:"uid" binding:"required"`
		Level int    `json:"level"` // 机器人对手等级（前端排行返回）
		Nick  string `json:"nick"`  // 机器人对手昵称
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if req.Uid == uid {
		resp.ParamError(c, "不能与自己比武")
		return
	}
	p := h.jwtPlayer(uid)
	// 重复比武：今日已与该玩家比武过则拒绝（复刻 比武重复.xhtml）
	if jwtArenaOppsHas(p.ArenaOpps, req.Uid) {
		resp.Fail(c, 200, 450, "今日已与该玩家比武过了，请换一个人比武")
		return
	}
	my := h.jwtCombat(p)
	// 主动技能：取已学主动技能中伤害系数最高者
	var mySkill model.JwtSkill
	h.DB.Raw(`SELECT s.id, s.name, s.act, s.level, s.coef, s.hit, s.crit, s.crit_mul, s.dodge_add
		FROM jwt_learned_skills ls
		JOIN jwt_skills s ON s.id = ls.skill_id
		WHERE ls.user_id = ? AND s.act = 1
		ORDER BY s.coef DESC LIMIT 1`, uid).Scan(&mySkill)
	if mySkill.ID == 0 {
		resp.ParamError(c, "您还没有学会任何主动技能，无法进行比武！请先学习一些主动技能。")
		return
	}
	// 对手属性（非机器人对手按玩家真实属性；机器人采用前端排行里的等级/昵称）
	var oppStats = model.JwtCombat{MaxHp: 20 + 5*2, MaxMp: 10 + 5, Atk: 3, Def: 1, Hit: 90, Crit: 5, CritMul: 150}
	oppNick := ""
	oppReal := false
	var opp model.JwtPlayer
	if err := h.DB.Where("user_id = ?", req.Uid).First(&opp).Error; err != nil {
		lv := req.Level
		if lv <= 0 {
			lv = 5
		}
		oppNick = req.Nick
		if oppNick == "" {
			oppNick = "神秘侠客"
		}
		oppStats.MaxHp = 20 + lv*2
		oppStats.MaxMp = 10 + lv
		oppStats.Atk = lv/2 + 1
		oppStats.Def = lv/3 + 1
	} else {
		oppNick = opp.Nick
		oppReal = true
		oppStats = h.jwtCombat(&opp)
	}
	if oppStats.MaxHp <= 0 {
		oppStats.MaxHp = 20
	}
	// 回合制战斗日志（复刻 比武.xhtml 叙事格式："第N回合，X施展出【技能】，效果描述，攻击Y，气血-D【-D/剩余】"）
	logs := []gin.H{}
	myHp, oppHp := my.MaxHp, oppStats.MaxHp
	winner := -1 // -1进行中 0我胜 1敌胜 2平
	round := 0
	// 伤害保底：按对方气血上限比例出伤，保证战斗在六~八回合内结束，避免长战报刷屏
	myFloor := oppStats.MaxHp/6 + 1
	oppFloor := my.MaxHp/8 + 1
	for round < 30 && myHp > 0 && oppHp > 0 {
		round++
		// 我方主动技能攻击
		baseDmg := float64(my.Atk) * float64(100+mySkill.Coef) / 100.0 * float64(mySkill.Coef) / 100.0
		dmg := int(baseDmg * (100.0 / (100.0 + float64(oppStats.Def))))
		if dmg < myFloor {
			dmg = myFloor
		}
		effect := jwtSkillEffect(mySkill.Name)
		attacker := p.Nick
		if attacker == "" {
			attacker = "你"
		}
		if rand.Intn(100) < mySkill.Hit {
			isCrit := rand.Intn(100) < mySkill.Crit
			if isCrit {
				dmg = dmg * my.CritMul / 100
			}
			oldOppHp := oppHp
			oppHp -= dmg
			// 原版叙事：第N回合，X施展出【技能】，效果，攻击对手，气血-D【-D/剩余】
			text := "第" + strconv.Itoa(round) + "回合，" + attacker + "施展出【" + mySkill.Name + "】，" + effect + "，攻击" + oppNick + "，气血-" + strconv.Itoa(dmg) + "【-" + strconv.Itoa(dmg) + "/" + strconv.Itoa(oldOppHp) + "】"
			logs = append(logs, gin.H{"round": round, "text": text})
		} else {
			text := "第" + strconv.Itoa(round) + "回合，" + attacker + "施展出【" + mySkill.Name + "】，被 [" + oppNick + "] 躲开了！"
			logs = append(logs, gin.H{"round": round, "text": text})
		}
		if oppHp <= 0 {
			winner = 0
			break
		}
		// 对方平砍反击（对手无主动技能，只平砍）
		dmg2 := int(float64(oppStats.Atk) * (100.0 / (100.0 + float64(my.Def))))
		if dmg2 < oppFloor {
			dmg2 = oppFloor
		}
		if rand.Intn(100) < oppStats.Hit {
			oldMyHp := myHp
			myHp -= dmg2
			text := "第" + strconv.Itoa(round) + "回合，" + oppNick + "反击，" + attacker + "气血-" + strconv.Itoa(dmg2) + "【-" + strconv.Itoa(dmg2) + "/" + strconv.Itoa(oldMyHp) + "】"
			logs = append(logs, gin.H{"round": round, "text": text})
		} else {
			text := "第" + strconv.Itoa(round) + "回合，" + oppNick + "反击被你躲开了！"
			logs = append(logs, gin.H{"round": round, "text": text})
		}
		if myHp <= 0 {
			winner = 1
			break
		}
	}
	if winner == -1 {
		winner = 2
	}
	// 结算
	msg := "比武结束，战成平手"
	gainExp := 10 + oppStats.MaxHp/4
	gainCoin := 50 + oppStats.MaxHp*2
	h.DB.Model(&model.JwtPlayer{}).Where("id = ?", p.ID).Update("arena_cnt", gorm.Expr("arena_cnt + 1"))
	if winner == 0 {
		levels := h.jwtGainExp(p, gainExp)
		h.DB.Model(&model.JwtPlayer{}).Where("id = ?", p.ID).Update("honor", gorm.Expr("honor + 1"))
		drop := h.jwtRandomBattleMat()
		msg = "你战胜了 [" + oppNick + "]！经验+" + strconv.Itoa(gainExp)
		if levels > 0 {
			msg += "，" + strconv.Itoa(levels) + " 新等级"
		}
		h.DB.Model(&model.User{}).Where("id = ?", uid).Update("coins", gorm.Expr("coins + ?", gainCoin))
		addWalletLog(h.DB, uid, "jwt", "精武堂比武胜利", "jwt_coins", gainCoin)
		msg += " G币+" + strconv.Itoa(gainCoin)
		if drop != nil {
			h.jwtBagAdd(uid, drop, 1)
			msg += "，掉落[" + drop.Name + "]"
		}
	} else if winner == 1 {
		levels := h.jwtGainExp(p, gainExp/3)
		msg = "你败于 [" + oppNick + "]！经验+" + strconv.Itoa(gainExp/3)
		if levels > 0 {
			msg += "，" + strconv.Itoa(levels) + " 新等级"
		}
	}
	// 比武受创，持久化当前气血/气力（供药品恢复），下次比武重置
	if myHp < 1 {
		myHp = 1
	}
	h.DB.Model(&model.JwtPlayer{}).Where("id = ?", p.ID).Updates(map[string]interface{}{
		"cur_hp": myHp, "cur_mp": my.MaxMp / 2})
	// 动态落库（首页「动态」feed，复刻原站：向[xxx]发起比武 / 被[xxx]攻击）
	wText := "失败"
	gainedExp, gainedCoin := 0, 0
	if winner == 0 {
		wText = "胜利"
		gainedExp = gainExp
		gainedCoin = gainCoin
	} else if winner == 1 {
		wText = "失败"
		gainedExp = gainExp / 3
	} else {
		wText = "平手"
	}
	h.DB.Create(&model.JwtLog{UserID: uid, Nick: p.Nick, Msg: "向[" + oppNick + "]发起比武 " + wText})
	if oppReal {
		h.DB.Create(&model.JwtLog{UserID: req.Uid, Nick: oppNick, Msg: "被[" + p.Nick + "]攻击 " + wText})
	}
	// 持久化整场比武记录（复刻 比武记录.xhtml：双方都可查看详情）
	oppLevel := 0
	if oppReal {
		oppLevel = opp.Level
	} else if req.Level > 0 {
		oppLevel = req.Level
	}
	oppCur := oppHp
	if oppCur < 1 {
		oppCur = 1
	}
	logLines := []string{}
	for _, lg := range logs {
		logLines = append(logLines, lg["text"].(string))
	}
	rec := model.JwtArenaRecord{
		UserID: uid, MyUID: uid, MyNick: p.Nick,
		OppUID: req.Uid, OppNick: oppNick,
		Result: wText, Exp: gainedExp, Coin: gainedCoin,
		MyLevel: p.Level, MyMaxHp: my.MaxHp, MyCurHp: myHp,
		OppLevel: oppLevel, OppMaxHp: oppStats.MaxHp, OppCurHp: oppCur,
		LogsText: strings.Join(logLines, "\n"),
	}
	h.DB.Create(&rec)
	if oppReal {
		// 被攻击方也存一条（视角对调：对方为「我」，胜负反过来）
		oppRes := "平手"
		if wText == "胜利" {
			oppRes = "失败"
		} else if wText == "失败" {
			oppRes = "胜利"
		}
		h.DB.Create(&model.JwtArenaRecord{
			UserID: req.Uid, MyUID: req.Uid, MyNick: oppNick,
			OppUID: uid, OppNick: p.Nick,
			Result: oppRes, Exp: gainedExp, Coin: gainedCoin,
			MyLevel: oppLevel, MyMaxHp: oppStats.MaxHp, MyCurHp: oppCur,
			OppLevel: p.Level, OppMaxHp: my.MaxHp, OppCurHp: myHp,
			LogsText: strings.Join(logLines, "\n"),
		})
	}
	// 记录今日已比武的对手（防止同日重复比武）
	if !jwtArenaOppsHas(p.ArenaOpps, req.Uid) {
		newOpps := p.ArenaOpps
		if newOpps != "" {
			newOpps += ","
		}
		newOpps += strconv.Itoa(int(req.Uid))
		h.DB.Model(&model.JwtPlayer{}).Where("id = ?", p.ID).Update("arena_opps", newOpps)
		p.ArenaOpps = newOpps
	}
	resp.OK(c, gin.H{"msg": msg, "winner": winner, "round": round, "logs": logs,
		"exp": gainedExp, "coin": gainedCoin, "result": wText})
}

// jwtPlayerSkillName 取用户最强主动技能名
func (h *JwtHandler) jwtPlayerSkillName(uid uint) string {
	var s model.JwtSkill
	h.DB.Raw(`SELECT s.name FROM jwt_learned_skills ls JOIN jwt_skills s ON s.id=ls.skill_id
		WHERE ls.user_id = ? AND s.act = 1 ORDER BY s.coef DESC LIMIT 1`, uid).Scan(&s)
	if s.Name == "" {
		return "普通攻击"
	}
	return s.Name
}

// ArenaLive 直播：最近一场比武（复刻 比武.xhtml 的【直播】，两名对手对战，展示经验/G币奖励，两人可点开属性页）
func (h *JwtHandler) ArenaLive(c *gin.Context) {
	uid := middleware.GetUID(c)
	var players []model.JwtPlayer
	h.DB.Where("user_id <> ?", uid).Order("level DESC, honor DESC").Limit(2).Find(&players)
	if len(players) < 2 {
		resp.OK(c, gin.H{"live": nil})
		return
	}
	a, b := &players[0], &players[1]
	ca, cb := h.jwtCombat(a), h.jwtCombat(b)
	aSkill := h.jwtPlayerSkillName(a.UserID)
	bSkill := h.jwtPlayerSkillName(b.UserID)
	logs := []gin.H{}
	ha, hb := ca.MaxHp, cb.MaxHp
	if ha < 1 {
		ha = 1
	}
	if hb < 1 {
		hb = 1
	}
	aFloor := cb.MaxHp/6 + 1
	bFloor := ca.MaxHp/8 + 1
	winner := -1
	for round := 1; round <= 12 && ha > 0 && hb > 0; round++ {
		// A 出手
		admg := int(float64(ca.Atk) * 2.0 * (100.0 / (100.0 + float64(cb.Def)*0.4)))
		if admg < aFloor {
			admg = aFloor
		}
		if rand.Intn(100) < 90 {
			oldB := hb
			hb -= admg
			if hb < 0 {
				hb = 0
			}
			logs = append(logs, gin.H{"round": round, "text": "第" + strconv.Itoa(round) + "回合，" + a.Nick + "施展出【" + aSkill + "】，威力惊人，攻击" + b.Nick + "，气血-" + strconv.Itoa(admg) + "【-" + strconv.Itoa(admg) + "/" + strconv.Itoa(oldB) + "】"})
		}
		if hb <= 0 {
			winner = 0
			break
		}
		// B 出手
		bdmg := int(float64(cb.Atk) * 2.0 * (100.0 / (100.0 + float64(ca.Def)*0.4)))
		if bdmg < bFloor {
			bdmg = bFloor
		}
		if rand.Intn(100) < 90 {
			oldA := ha
			ha -= bdmg
			if ha < 0 {
				ha = 0
			}
			logs = append(logs, gin.H{"round": round, "text": "第" + strconv.Itoa(round) + "回合，" + b.Nick + "施展出【" + bSkill + "】，威力惊人，攻击" + a.Nick + "，气血-" + strconv.Itoa(bdmg) + "【-" + strconv.Itoa(bdmg) + "/" + strconv.Itoa(oldA) + "】"})
		}
		if ha <= 0 {
			winner = 1
			break
		}
	}
	if winner == -1 {
		winner = 0
	}
	win, lose := a, b
	if winner == 1 {
		win, lose = b, a
	}
	exp := 6 + lose.Level*2
	coin := 20 + lose.Level*8
	resp.OK(c, gin.H{"live": gin.H{
		"win_uid": win.UserID, "win_nick": win.Nick,
		"lose_uid": lose.UserID, "lose_nick": lose.Nick,
		"exp": exp, "coin": coin, "result": "胜利", "logs": logs}})
}

// jwtRandomBattleMat 随机掉一个比武材料
func (h *JwtHandler) jwtRandomBattleMat() *model.JwtItem {
	var mats []model.JwtItem
	h.DB.Where("src = 'battle'").Find(&mats)
	if len(mats) == 0 {
		return nil
	}
	return &mats[rand.Intn(len(mats))]
}

// ArenaRecords 比武记录（复刻 比武记录.xhtml：我的昵称 VS 对方昵称 + 时间）
func (h *JwtHandler) ArenaRecords(c *gin.Context) {
	uid := middleware.GetUID(c)
	var recs []model.JwtArenaRecord
	h.DB.Where("user_id = ?", uid).Order("id desc").Limit(50).Find(&recs)
	list := []gin.H{}
	for _, r := range recs {
		list = append(list, gin.H{
			"id": r.ID, "my_nick": r.MyNick, "opp_nick": r.OppNick,
			"result": r.Result, "created_at": r.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	resp.OK(c, gin.H{"list": list, "total": len(list)})
}

// RecordDetail 比武记录详情（复刻 点击记录后的比武详情页）
func (h *JwtHandler) RecordDetail(c *gin.Context) {
	uid := middleware.GetUID(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	var r model.JwtArenaRecord
	if err := h.DB.Where("id = ? AND user_id = ?", id, uid).First(&r).Error; err != nil {
		resp.ParamError(c, "记录不存在")
		return
	}
	logs := []gin.H{}
	for i, line := range strings.Split(r.LogsText, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		logs = append(logs, gin.H{"round": i + 1, "text": line})
	}
	resp.OK(c, gin.H{"detail": gin.H{
		"my_nick": r.MyNick, "my_level": r.MyLevel, "my_cur_hp": r.MyCurHp, "my_max_hp": r.MyMaxHp,
		"opp_nick": r.OppNick, "opp_level": r.OppLevel, "opp_cur_hp": r.OppCurHp, "opp_max_hp": r.OppMaxHp,
		"result": r.Result, "exp": r.Exp, "coin": r.Coin,
		"logs": logs, "count": len(logs),
	}})
}

//
// ---------- 背包/装备 ----------
//

// Bag 背包列表
func (h *JwtHandler) Bag(c *gin.Context) {
	uid := middleware.GetUID(c)
	var bags []model.JwtBag
	h.DB.Where("user_id = ? AND amount > 0", uid).Order("id ASC").Find(&bags)
	out := []gin.H{}
	for _, b := range bags {
		var it model.JwtItem
		h.DB.First(&it, b.ItemID)
		out = append(out, gin.H{"item_id": b.ItemID, "name": b.Name, "cat": b.Cat, "amount": b.Amount,
			"level": it.Level, "desc": it.Desc,
			"atk": it.Atk, "def": it.Def, "hp": it.Hp, "mp": it.Mp, "spd": it.Spd,
			"recover_hp": it.RecoverHp, "recover_mp": it.RecoverMp})
	}
	resp.OK(c, out)
}

// UseItem 使用道具（复刻 使用道具.xhtml：恢复气血/气力，支持数量与「一键使用至满」）
func (h *JwtHandler) UseItem(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		ID    uint   `json:"id" binding:"required"`
		Count int    `json:"count"`
		Mode  string `json:"mode"` // "" 或 "use" 普通 / "full" 一键至满
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	it := h.jwtItemByID(req.ID)
	if it == nil || (it.RecoverHp <= 0 && it.RecoverMp <= 0) {
		resp.ParamError(c, "该道具不能直接使用")
		return
	}
	have := h.jwtBagCount(uid, it.ID)
	if have <= 0 {
		resp.ParamError(c, "背包中没有该道具")
		return
	}
	p := h.jwtPlayer(uid)
	cbt := h.jwtCombat(p)
	curHp, curMp := p.CurHp, p.CurMp
	if curHp < 1 || curHp > cbt.MaxHp {
		curHp = cbt.MaxHp
	}
	if curMp < 1 || curMp > cbt.MaxMp {
		curMp = cbt.MaxMp
	}
	needHp := cbt.MaxHp - curHp
	needMp := cbt.MaxMp - curMp
	if needHp <= 0 && needMp <= 0 {
		resp.ParamError(c, "气血和气力都已满，无需使用！")
		return
	}
	// 确定使用数量
	want := req.Count
	if req.Mode == "full" {
		// 计算恰好回满需要的数量
		fullNeed := 0
		if it.RecoverHp > 0 {
			fullNeed = (needHp + it.RecoverHp - 1) / it.RecoverHp
		}
		if it.RecoverMp > 0 {
			mpNeed := (needMp + it.RecoverMp - 1) / it.RecoverMp
			if mpNeed > fullNeed {
				fullNeed = mpNeed
			}
		}
		if fullNeed < 1 {
			fullNeed = 1
		}
		want = fullNeed
	}
	if want < 1 {
		want = 1
	}
	if want > have {
		want = have
	}
	// 逐份恢复
	usedHp, usedMp, usedItems := 0, 0, 0
	for n := 0; n < want; n++ {
		dh := it.RecoverHp
		dm := it.RecoverMp
		takeHp := dh
		if takeHp > needHp {
			takeHp = needHp
		}
		takeMp := dm
		if takeMp > needMp {
			takeMp = needMp
		}
		if takeHp <= 0 && takeMp <= 0 {
			break
		}
		needHp -= takeHp
		needMp -= takeMp
		usedHp += takeHp
		usedMp += takeMp
		h.jwtBagSub(uid, it.ID, 1)
		if want == 1 {
			h.DB.Model(&model.JwtPlayer{}).Where("id = ?", uid).Update("pill_cnt", gorm.Expr("pill_cnt + 1"))
		}
	}
	curHp += usedHp
	curMp += usedMp
	h.DB.Model(&model.JwtPlayer{}).Where("user_id = ?", uid).Updates(map[string]interface{}{
		"cur_hp": curHp, "cur_mp": curMp})
	msg := "使用[" + it.Name + "]成功"
	if usedHp > 0 {
		msg += "，恢复气血" + strconv.Itoa(usedHp)
	}
	if usedMp > 0 {
		msg += "，恢复气力" + strconv.Itoa(usedMp)
	}
	resp.OK(c, gin.H{"msg": msg, "cur_hp": curHp, "cur_mp": curMp,
		"max_hp": cbt.MaxHp, "max_mp": cbt.MaxMp, "used": usedItems})
}

// ItemDrop 丢弃道具（放入背包后丢出，减少数量）
func (h *JwtHandler) ItemDrop(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		ID uint `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if h.jwtBagCount(uid, req.ID) <= 0 {
		resp.ParamError(c, "背包中没有该道具")
		return
	}
	h.jwtBagSub(uid, req.ID, 1)
	resp.OK(c, gin.H{"msg": "丢弃1个成功"})
}

// Equip 穿装备（武器/防具等）
func (h *JwtHandler) Equip(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		ID uint `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	it := h.jwtItemByID(req.ID)
	if it == nil {
		resp.NotFound(c, "无此物品")
		return
	}
	slot := hrefJwtSlot(it.Cat)
	if slot == "" {
		resp.ParamError(c, "该物品不能装备")
		return
	}
	p := h.jwtPlayer(uid)
	if it.Level > p.Level {
		resp.ParamError(c, "需要"+strconv.Itoa(it.Level)+"级才能装备")
		return
	}
	if h.jwtBagCount(uid, it.ID) <= 0 {
		resp.ParamError(c, "背包中没有该物品")
		return
	}
	// 若当前穿有同类装备，先卸下（回背包）
	hjwtSlots := map[string]uint{
		"weapon_id": p.WeaponID, "helmet_id": p.HelmetID, "armor_id": p.ArmorID, "shoes_id": p.ShoesID,
		"necklace_id": p.NecklaceID, "bracelet_id": p.BraceletID, "ring_id": p.RingID, "medal_id": p.MedalID}
	if oldID := hjwtSlots[slot]; oldID > 0 && oldID != it.ID {
		h.jwtBagAdd(uid, h.jwtItemByID(oldID), 1)
		h.DB.Model(&model.JwtPlayer{}).Where("id = ?", p.ID).Update(slot, 0)
	}
	h.jwtBagSub(uid, it.ID, 1)
	h.DB.Model(&model.JwtPlayer{}).Where("id = ?", p.ID).Update(slot, it.ID)
	resp.OK(c, gin.H{"msg": "装备[" + it.Name + "]成功"})
}

// Unequip 卸下装备
func (h *JwtHandler) Unequip(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Slot string `json:"slot" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	col := ""
	switch req.Slot {
	case "weapon":
		col = "weapon_id"
	case "helmet":
		col = "helmet_id"
	case "armor":
		col = "armor_id"
	case "shoes":
		col = "shoes_id"
	case "necklace":
		col = "necklace_id"
	case "bracelet":
		col = "bracelet_id"
	case "ring":
		col = "ring_id"
	case "medal":
		col = "medal_id"
	default:
		resp.ParamError(c, "槽位错误")
		return
	}
	p := h.jwtPlayer(uid)
	hjwtSlots := map[string]uint{
		"weapon_id": p.WeaponID, "helmet_id": p.HelmetID, "armor_id": p.ArmorID, "shoes_id": p.ShoesID,
		"necklace_id": p.NecklaceID, "bracelet_id": p.BraceletID, "ring_id": p.RingID, "medal_id": p.MedalID}
	id := hjwtSlots[col]
	if id == 0 {
		resp.ParamError(c, "该槽位没有装备")
		return
	}
	if it := h.jwtItemByID(id); it != nil {
		h.jwtBagAdd(uid, it, 1)
	}
	h.DB.Model(&model.JwtPlayer{}).Where("id = ?", p.ID).Update(col, 0)
	resp.OK(c, gin.H{"msg": "已卸下装备"})
}

// ProfileEdit 修改资料（复刻 修改资料.xhtml：性别 + 昵称）
func (h *JwtHandler) ProfileEdit(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Sex  int    `json:"sex"` // 0保密 1男 2女
		Name string `json:"name"`
	}
	_ = c.ShouldBindJSON(&req)
	if req.Sex < 0 || req.Sex > 2 {
		req.Sex = 0
	}
	p := h.jwtPlayer(uid)
	updates := map[string]interface{}{"sex": req.Sex}
	if strings.TrimSpace(req.Name) != "" {
		updates["nick"] = strings.TrimSpace(req.Name)
		h.DB.Model(&model.User{}).Where("id = ?", uid).Update("nickname", strings.TrimSpace(req.Name))
	}
	h.DB.Model(&model.JwtPlayer{}).Where("id = ?", p.ID).Updates(updates)
	resp.OK(c, gin.H{"msg": "修改成功！"})
}

//
// ---------- 锻造 ----------
//

// Forge 装备锻造（图纸+材料+元宝）
func (h *JwtHandler) Forge(c *gin.Context) {
	uid := middleware.GetUID(c)
	p := h.jwtPlayer(uid)
	var items []model.JwtItem
	h.DB.Where("status = 1 AND src = 'forge'").Order("level ASC, id ASC").Find(&items)
	out := []gin.H{}
	for _, it := range items {
		status := "ok"
		if it.Level > p.Level {
			status = "等级不足"
		}
		results := jwtParseMats(it.Mats, uid, h)
		out = append(out, gin.H{"id": it.ID, "name": it.Name, "cat": it.Cat, "level": it.Level,
			"fee": it.Fee, "atk": it.Atk, "def": it.Def, "hp": it.Hp, "mp": it.Mp, "spd": it.Spd,
			"mats": results, "status": status})
	}
	resp.OK(c, out)
}

// jwtParseMats 解析材料需求，返回 [{name, need, have}]
func jwtParseMats(mats string, uid uint, h *JwtHandler) []gin.H {
	if mats == "" {
		return []gin.H{}
	}
	out := []gin.H{}
	for _, seg := range strings.Split(mats, ",") {
		parts := strings.Split(strings.TrimSpace(seg), "x")
		if len(parts) != 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		need, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
		have := 0
		var it model.JwtItem
		if err := h.DB.Where("name = ?", name).First(&it).Error; err == nil {
			have = h.jwtBagCount(uid, it.ID)
		}
		out = append(out, gin.H{"name": name, "need": need, "have": have})
	}
	return out
}

// ForgeItem 执行锻造
func (h *JwtHandler) ForgeItem(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		ID uint `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	p := h.jwtPlayer(uid)
	it := h.jwtItemByID(req.ID)
	if it == nil || it.Src != "forge" {
		resp.NotFound(c, "无此锻造配方")
		return
	}
	if it.Level > p.Level {
		resp.ParamError(c, "需要"+strconv.Itoa(it.Level)+"级才能锻造")
		return
	}
	u := h.jwtBrief(uid)
	if u.YuanBao < it.Fee {
		resp.ParamError(c, "元宝不足，需要"+strconv.Itoa(it.Fee)+" 元宝")
		return
	}
	// 校验材料
	for _, mat := range jwtParseMats(it.Mats, uid, h) {
		mn, _ := mat["name"].(string)
		needArr := strings.Split(strconv.Itoa(mat["need"].(int))+ "_"+mn, "_")
		needStr := needArr[0]
		need, _ := strconv.Atoi(needStr)
		have := mat["have"].(int)
		if have < need {
			resp.ParamError(c, "缺少材料："+mn+(needStr)+"x"+strconv.Itoa(need-have))
			return
		}
	}
	// 扣材料
	for _, seg := range strings.Split(it.Mats, ",") {
		parts := strings.Split(strings.TrimSpace(seg), "x")
		if len(parts) != 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		need, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
		var mit model.JwtItem
		if err := h.DB.Where("name = ?", name).First(&mit).Error; err == nil {
			h.jwtBagSub(uid, mit.ID, need)
		}
	}
	h.DB.Model(&model.User{}).Where("id = ?", uid).Update("yuanbao", gorm.Expr("yuanbao - ?", it.Fee))
	addWalletLog(h.DB, uid, "jwt", "精武堂锻造["+it.Name+"]", "jwt_yuanbao", -it.Fee)
	h.jwtBagAdd(uid, it, 1)
	resp.OK(c, gin.H{"msg": "锻造成功！获得[" + it.Name + "]"})
}

//
// ---------- 每日任务/礼包/签到 ----------
//

// jwtSignStatus 返回本周各天是否已签
func jwtSignStatus(week string) map[string]bool {
	m := map[string]bool{}
	for i := 1; i <= 7; i++ {
		m[strconv.Itoa(i)] = false
	}
	if week != "" {
		for _, d := range strings.Split(week, ",") {
			m[d] = true
		}
	}
	return m
}

// Tasks 每日任务进度
func (h *JwtHandler) Tasks(c *gin.Context) {
	uid := middleware.GetUID(c)
	p := h.jwtPlayer(uid)
	h.jwtRoll(p)
	tasks := []gin.H{
		{"key": "train", "name": "完成一次修炼", "cur": p.TrainCnt, "need": 1},
		{"key": "arena", "name": "完成10次比武", "cur": p.ArenaCnt, "need": 10},
		{"key": "chat", "name": "公共世界聊天发言", "cur": p.ChatCnt, "need": 1},
		{"key": "pill", "name": "使用气力丸一次", "cur": p.PillCnt, "need": 1},
		{"key": "skill", "name": "学习或修炼技能一次", "cur": p.SkillCnt, "need": 1},
	}
	done := 0
	for _, t := range tasks {
		if t["cur"].(int) >= t["need"].(int) {
			done++
		}
	}
	resp.OK(c, gin.H{"tasks": tasks, "done": done, "need_done": 2, "reward_claim": p.RewardClaim})
}

// Sign 每周签到
func (h *JwtHandler) Sign(c *gin.Context) {
	uid := middleware.GetUID(c)
	p := h.jwtPlayer(uid)
	h.jwtRoll(p)
	weekday := int(time.Now().Weekday()) // 0=周日
	day := (weekday + 6) % 7 + 1         // 1-7 (周一开始)
	if jwtSignStatus(p.SignWeek)[strconv.Itoa(day)] {
		resp.ParamError(c, "今天已签到")
		return
	}
	// 签到奖励 G币/元宝
	coin := 1000 + p.Level*100
	h.DB.Model(&model.User{}).Where("id = ?", uid).Update("coins", gorm.Expr("coins + ?", coin))
	addWalletLog(h.DB, uid, "jwt", "精武堂每日签到", "jwt_coins", coin)
	newWeek := p.SignWeek
	if newWeek == "" {
		newWeek = strconv.Itoa(day)
	} else {
		newWeek += "," + strconv.Itoa(day)
	}
	h.DB.Model(&model.JwtPlayer{}).Where("id = ?", p.ID).Update("sign_week", newWeek)
	resp.OK(c, gin.H{"msg": "签到成功！获得" + strconv.Itoa(coin) + " G币"})
}

// Reward 领取每日礼包（需完成>=2个任务）
func (h *JwtHandler) Reward(c *gin.Context) {
	uid := middleware.GetUID(c)
	p := h.jwtPlayer(uid)
	h.jwtRoll(p)
	if p.RewardClaim == 1 {
		resp.ParamError(c, "今日礼包已领取")
		return
	}
	done := 0
	if p.TrainCnt >= 1 {
		done++
	}
	if p.ArenaCnt >= 10 {
		done++
	}
	if p.ChatCnt >= 1 {
		done++
	}
	if p.PillCnt >= 1 {
		done++
	}
	if p.SkillCnt >= 1 {
		done++
	}
	if done < 2 {
		resp.ParamError(c, "需完成至少2个任务才能领取礼包")
		return
	}
	coin := 2000 + p.Level*200
	yb := 0
	// 周末礼包加倍
	if int(time.Now().Weekday()) == 6 || int(time.Now().Weekday()) == 0 {
		yb = 2
	}
	updates := map[string]interface{}{"reward_claim": 1}
	h.DB.Model(&model.JwtPlayer{}).Where("id = ?", p.ID).Updates(updates)
	h.DB.Model(&model.User{}).Where("id = ?", uid).Update("coins", gorm.Expr("coins + ?", coin))
	addWalletLog(h.DB, uid, "jwt", "精武堂每日礼包", "jwt_coins", coin)
	msg := "领取成功！获得" + strconv.Itoa(coin) + " G币"
	if yb > 0 {
		h.DB.Model(&model.User{}).Where("id = ?", uid).Update("yuanbao", gorm.Expr("yuanbao + ?", yb))
		addWalletLog(h.DB, uid, "jwt", "精武堂周末礼包", "jwt_yuanbao", yb)
		msg += "，" + strconv.Itoa(yb) + " 元宝"
	}
	resp.OK(c, gin.H{"msg": msg})
}

// Chat 发布聊天（复刻原站 chat_add.aspx，type=0公共/1个人/2世界，发布成功后显示「发布成功！」）
func (h *JwtHandler) Chat(c *gin.Context) {
	uid := middleware.GetUID(c)
	p := h.jwtPlayer(uid)
	var req struct {
		Content string `json:"content"`
		Type    int    `json:"type"`
	}
	_ = c.ShouldBindJSON(&req)
	content := strings.TrimSpace(req.Content)
	if content != "" {
		runes := []rune(content)
		if len(runes) > 100 {
			content = string(runes[:100])
		}
		ctype := req.Type
		if ctype < 0 || ctype > 2 {
			ctype = 0
		}
		h.DB.Create(&model.JwtChat{UserID: uid, Nick: p.Nick, Content: content, Type: ctype})
	}
	h.DB.Model(&model.JwtPlayer{}).Where("id = ?", p.ID).Update("chat_cnt", gorm.Expr("chat_cnt + 1"))
	resp.OK(c, gin.H{"msg": "发布成功！"})
}

// ChatList 聊天记录。type 缺省/-1=全部(首页 feed 合并显示)；0公共/1个人/2世界=按频道过滤
func (h *JwtHandler) ChatList(c *gin.Context) {
	ctype := -1
	if v := c.Query("type"); v != "" {
		ctype, _ = strconv.Atoi(v)
	}
	if ctype < -1 || ctype > 2 {
		ctype = -1
	}
	q := h.DB
	if ctype >= 0 {
		q = q.Where("type = ?", ctype)
	}
	var msgs []model.JwtChat
	q.Order("id DESC").Limit(20).Find(&msgs)
	out := []gin.H{}
	for i := len(msgs) - 1; i >= 0; i-- {
		m := msgs[i]
		out = append(out, gin.H{"nick": m.Nick, "content": m.Content, "type": m.Type,
			"time": m.CreatedAt.Format("2006-01-02 15:04:05")})
	}
	resp.OK(c, out)
}

// Logs 我的动态（首页「动态」，复刻原站：比武等事件流）
func (h *JwtHandler) Logs(c *gin.Context) {
	uid := middleware.GetUID(c)
	var logs []model.JwtLog
	h.DB.Where("user_id = ?", uid).Order("id DESC").Limit(10).Find(&logs)
	out := []gin.H{}
	for _, l := range logs {
		out = append(out, gin.H{"msg": l.Msg,
			"time": l.CreatedAt.Format("2006-01-02 15:04:05")})
	}
	resp.OK(c, out)
}

//
// ---------- 头衔 ----------
//

// Titles 头衔列表/激活
func (h *JwtHandler) Titles(c *gin.Context) {
	uid := middleware.GetUID(c)
	p := h.jwtPlayer(uid)
	titleRows := []struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Level int    `json:"level"`
		Price int    `json:"price"`
	}{}
	titles := []string{"少侠", "侠士", "大侠", "豪杰", "宗师", "霸者", "圣者", "武尊", "武王", "至尊战神"}
	for i, name := range titles {
		titleRows = append(titleRows, struct {
			ID    int    `json:"id"`
			Name  string `json:"name"`
			Level int    `json:"level"`
			Price int    `json:"price"`
		}{ID: i + 1, Name: name, Level: (i + 1) * 100, Price: (i + 1) * 100})
	}
	activated := false
	if p.Title > 0 {
		activated = true
	}
	var req struct {
		ID int `json:"id"`
	}
	_ = c.ShouldBindJSON(&req)
	if req.ID > 0 {
		idx := req.ID - 1
		if idx < 0 || idx >= len(titleRows) {
			resp.ParamError(c, "头衔不存在")
			return
		}
		t := titleRows[idx]
		if p.Level < t.Level {
			resp.ParamError(c, "等级不足，需要"+strconv.Itoa(t.Level)+"级")
			return
		}
		u := h.jwtBrief(uid)
		if u.YuanBao < t.Price {
			resp.ParamError(c, "元宝不足，需要"+strconv.Itoa(t.Price)+" 元宝")
			return
		}
		h.DB.Model(&model.User{}).Where("id = ?", uid).Update("yuanbao", gorm.Expr("yuanbao - ?", t.Price))
		addWalletLog(h.DB, uid, "jwt", "精武堂激活头衔["+t.Name+"]", "jwt_yuanbao", -t.Price)
		h.DB.Model(&model.JwtPlayer{}).Where("id = ?", p.ID).Update("title", t.ID)
		resp.OK(c, gin.H{"msg": "激活头衔[" + t.Name + "]成功！"})
		return
	}
	resp.OK(c, gin.H{"titles": titleRows, "current": p.Title, "level": p.Level, "activated": activated})
}

//
// ---------- 帮派 ----------
//

// Gangs 帮派列表
func (h *JwtHandler) Gangs(c *gin.Context) {
	uid := middleware.GetUID(c)
	var gangs []model.JwtGang
	h.DB.Order("level DESC, id ASC").Find(&gangs)
	p := h.jwtPlayer(uid)
	out := []gin.H{}
	for _, g := range gangs {
		out = append(out, gin.H{"id": g.ID, "name": g.Name, "level": g.Level,
			"master": g.Master, "members": g.Members})
	}
	resp.OK(c, gin.H{"list": out, "my_gang": p.GangID})
}

// GangCreate 创建帮派（需帮派令旗）
func (h *JwtHandler) GangCreate(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" || len([]rune(name)) > 12 {
		resp.ParamError(c, "帮派名需1-12字")
		return
	}
	p := h.jwtPlayer(uid)
	if p.GangID > 0 {
		resp.ParamError(c, "您已有帮派")
		return
	}
	if p.Level < 10 {
		resp.ParamError(c, "帮主等级需达10级才能创建帮派(当前"+strconv.Itoa(p.Level)+"级)")
		return
	}
	var flag model.JwtItem
	if err := h.DB.Where("name = ?", "帮派令旗").First(&flag).Error; err != nil {
		resp.NotFound(c, "帮派令旗不存在")
		return
	}
	if h.jwtBagCount(uid, flag.ID) <= 0 {
		resp.ParamError(c, "需要1个帮派令旗(商店购买)才能创建帮派")
		return
	}
	var n int64
	h.DB.Model(&model.JwtGang{}).Where("name = ?", name).Count(&n)
	if n > 0 {
		resp.ParamError(c, "帮派名已存在")
		return
	}
	u := h.jwtBrief(uid)
	g := model.JwtGang{Name: name, MasterID: uid, Master: u.Nickname, Members: 1, Notice: "新帮派成立", CreatedAt: time.Now()}
	h.DB.Create(&g)
	h.jwtBagSub(uid, flag.ID, 1)
	h.DB.Create(&model.JwtGangMember{GangID: g.ID, UserID: uid, IsMaster: 1, JoinedAt: time.Now()})
	h.DB.Model(&model.JwtPlayer{}).Where("id = ?", p.ID).Update("gang_id", g.ID)
	resp.OK(c, gin.H{"msg": "创建帮派[" + name + "]成功！"})
}

// GangJoin 加入帮派
func (h *JwtHandler) GangJoin(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		ID uint `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	p := h.jwtPlayer(uid)
	if p.GangID > 0 {
		resp.ParamError(c, "您已有帮派")
		return
	}
	var g model.JwtGang
	if err := h.DB.First(&g, req.ID).Error; err != nil {
		resp.NotFound(c, "帮派不存在")
		return
	}
	if g.Members >= 20 {
		resp.ParamError(c, "帮派已满员")
		return
	}
	h.DB.Model(&model.JwtGang{}).Where("id = ?", g.ID).Update("members", gorm.Expr("members + 1"))
	h.DB.Create(&model.JwtGangMember{GangID: g.ID, UserID: uid, JoinedAt: time.Now()})
	h.DB.Model(&model.JwtPlayer{}).Where("id = ?", p.ID).Update("gang_id", g.ID)
	resp.OK(c, gin.H{"msg": "已加入帮派[" + g.Name + "]！"})
}

// GangLeave 退出帮派
func (h *JwtHandler) GangLeave(c *gin.Context) {
	uid := middleware.GetUID(c)
	p := h.jwtPlayer(uid)
	if p.GangID == 0 {
		resp.ParamError(c, "您没有帮派")
		return
	}
	var g model.JwtGang
	h.DB.First(&g, p.GangID)
	if g.ID == 0 {
		resp.ParamError(c, "帮派不存在")
		return
	}
	if g.MasterID == uid {
		// 解散
		h.DB.Delete(&model.JwtGang{}, g.ID)
		h.DB.Where("gang_id = ?", g.ID).Delete(&model.JwtGangMember{})
		h.DB.Model(&model.JwtPlayer{}).Where("gang_id = ?", g.ID).Update("gang_id", 0)
		resp.OK(c, gin.H{"msg": "帮派已解散"})
		return
	}
	h.DB.Where("user_id = ?", uid).Delete(&model.JwtGangMember{})
	h.DB.Model(&model.JwtGang{}).Where("id = ?", g.ID).Update("members", gorm.Expr("members - 1"))
	h.DB.Model(&model.JwtPlayer{}).Where("id = ?", p.ID).Update("gang_id", 0)
	resp.OK(c, gin.H{"msg": "已退出帮派"})
}

// LevelUp 手动升级（复刻 升级.xhtml：消耗经验升级）
func (h *JwtHandler) LevelUp(c *gin.Context) {
	uid := middleware.GetUID(c)
	p := h.jwtPlayer(uid)
	if p.Exp >= p.Level*64 {
		p.Exp -= p.Level * 64
		p.Level++
		p.Energy++
		h.DB.Model(&model.JwtPlayer{}).Where("id = ?", p.ID).Updates(map[string]interface{}{
			"level": p.Level, "exp": p.Exp, "energy": p.Energy})
		resp.OK(c, gin.H{"msg": "升级成功！升到" + strconv.Itoa(p.Level) + "级",
			"level": p.Level, "exp": p.Exp})
		return
	}
	need := p.Level*64 - p.Exp
	resp.ParamError(c, "升级失败！需要"+strconv.Itoa(need)+"点经验才能升到"+strconv.Itoa(p.Level+1)+"级！")
}

// jwtApplied 是否已申请加入某帮派（待审批）
func (h *JwtHandler) jwtApplied(gangID, uid uint) int {
	var n int64
	h.DB.Model(&model.JwtGangApply{}).Where("gang_id = ? AND user_id = ? AND status = ?", gangID, uid, 0).Count(&n)
	if n > 0 {
		return 1
	}
	return 0
}

// GangView 帮派详情（复刻 帮派详情.xhtml）
func (h *JwtHandler) GangView(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		ID uint `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	var g model.JwtGang
	if err := h.DB.First(&g, req.ID).Error; err != nil {
		resp.NotFound(c, "帮派不存在")
		return
	}
	const maxLevel = 3
	levelText := strconv.Itoa(g.Level) + "级"
	if g.Level >= maxLevel {
		levelText += "(已满级)"
	}
	// 帮派经验：当前/所需 (升级进度 N%)，对齐 帮派详情.xhtml
	required := g.Level * 1000
	if required < 1000 {
		required = 1000
	}
	expText := strconv.Itoa(g.Exp) + "/" + strconv.Itoa(required) + " (升级进度 " + strconv.Itoa(g.Exp*100/required) + "%)"
	p := h.jwtPlayer(uid)
	resp.OK(c, gin.H{
		"id": g.ID, "name": g.Name, "level_text": levelText, "exp_text": expText,
		"master": g.Master, "members": g.Members, "notice": g.Notice,
		"created_at": g.CreatedAt.Format("2006-01-02 15:04:05"),
		"exp_desc":   "经验产出渠道=成员加入(等级×10)/捐献银币/捐矿石/挖矿/温泉/战旗/比武胜利",
		"my_gang":    p.GangID, "in_gang": p.GangID == g.ID, "already_apply": h.jwtApplied(g.ID, uid),
	})
}

// GangApply 申请加入帮派（复刻 申请加入帮派.xhtml：提交申请，等帮主审批）
func (h *JwtHandler) GangApply(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		ID  uint   `json:"id" binding:"required"`
		Msg string `json:"msg"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	p := h.jwtPlayer(uid)
	if p.GangID > 0 {
		resp.ParamError(c, "您已有帮派")
		return
	}
	var g model.JwtGang
	if err := h.DB.First(&g, req.ID).Error; err != nil {
		resp.NotFound(c, "帮派不存在")
		return
	}
	if h.jwtApplied(g.ID, uid) == 1 {
		resp.ParamError(c, "您已申请过该帮派，请等待审批")
		return
	}
	h.DB.Create(&model.JwtGangApply{GangID: g.ID, UserID: uid, Msg: strings.TrimSpace(req.Msg), CreatedAt: time.Now()})
	resp.OK(c, gin.H{"msg": "申请提交成功！请等待帮主审批"})
}

//
// ---------- 排行 ----------
//

// Ranking 等级排行
func (h *JwtHandler) Ranking(c *gin.Context) {
	var players []model.JwtPlayer
	h.DB.Order("level DESC, exp DESC, id ASC").Limit(50).Find(&players)
	out := []gin.H{}
	for i, p := range players {
		out = append(out, gin.H{"rank": i + 1, "uid": p.UserID, "nick": p.Nick, "level": p.Level,
			"honor": p.Honor, "title": jwtTitleName(p.Title)})
	}
	resp.OK(c, out)
}

// TitleRanking 头衔榜（复刻 头衔榜.xhtml：按等级列玩家，显示头衔[]+昵称(等级)+比武）
func (h *JwtHandler) TitleRanking(c *gin.Context) {
	var players []model.JwtPlayer
	h.DB.Order("level DESC, exp DESC, id ASC").Limit(30).Find(&players)
	out := []gin.H{}
	for i, p := range players {
		out = append(out, gin.H{"rank": i + 1, "uid": p.UserID, "nick": p.Nick, "level": p.Level,
			"title_name": jwtTitleName(p.Title)})
	}
	resp.OK(c, out)
}

// MySkill 我的技能（已学技能列表，复刻原站 我的技能.xhtml：主动/被动分组 + (Lv.X)(熟练度) + 装备态）
func (h *JwtHandler) MySkill(c *gin.Context) {
	uid := middleware.GetUID(c)
	type row struct {
		ID       uint   `json:"id"`
		Name     string `json:"name"`
		Act      int    `json:"act"`
		Level    int    `json:"level"`    // 已学技能当前等级
		Practice int    `json:"practice"` // 熟练度
		Equip    int    `json:"equip"`
		PHp      int    `json:"p_hp"`
		PMp      int    `json:"p_mp"`
		PAtk     int    `json:"p_atk"`
		PDef     int    `json:"p_def"`
	}
	var rows []row
	h.DB.Raw(`SELECT s.id, s.name, s.act, ls.level, ls.practice, ls.equip, s.p_hp, s.p_mp, s.p_atk, s.p_def
		FROM jwt_learned_skills ls JOIN jwt_skills s ON s.id = ls.skill_id
		WHERE ls.user_id = ? ORDER BY s.id ASC`, uid).Scan(&rows)
	out := []gin.H{}
	for _, r := range rows {
		out = append(out, gin.H{"id": r.ID, "name": r.Name, "act": r.Act, "level": r.Level,
			"practice": r.Practice, "equip": r.Equip,
			"p_hp": r.PHp, "p_mp": r.PMp, "p_atk": r.PAtk, "p_def": r.PDef})
	}
	resp.OK(c, out)
}

// SkillAct 技能装备/卸下（复刻原站 my_skills.aspx?act=equip/unequip&id=XX）
func (h *JwtHandler) SkillAct(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Act string `json:"act" binding:"required"` // equip / unequip
		ID  uint   `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	var l model.JwtLearnedSkill
	if err := h.DB.Where("user_id = ? AND skill_id = ?", uid, req.ID).First(&l).Error; err != nil {
		resp.NotFound(c, "尚未学习该技能")
		return
	}
	switch req.Act {
	case "equip":
		h.DB.Model(&model.JwtLearnedSkill{}).Where("user_id = ?", uid).Update("equip", 0)
		h.DB.Model(&model.JwtLearnedSkill{}).Where("id = ?", l.ID).Update("equip", 1)
		resp.OK(c, gin.H{"msg": "装备成功！"})
	case "unequip":
		h.DB.Model(&model.JwtLearnedSkill{}).Where("id = ?", l.ID).Update("equip", 0)
		resp.OK(c, gin.H{"msg": "已卸下！"})
	default:
		resp.ParamError(c, "操作错误")
	}
}

// Friends 我的好友（复刻原站 我的好友 列表，来自社区好友关系）
func (h *JwtHandler) Friends(c *gin.Context) {
	uid := middleware.GetUID(c)
	var nicks []string
	h.DB.Raw(`SELECT u.nickname FROM friendships f JOIN users u ON u.id = f.friend_id
		WHERE f.user_id = ? AND f.status = 1 ORDER BY f.id ASC`, uid).Scan(&nicks)
	if nicks == nil {
		nicks = []string{}
	}
	resp.OK(c, nicks)
}

// PlayerProfile 查看他人档案（复刻原站 点开用户详情：昵称/性别/等级/称号/装备/我要比武/好友修炼状态）
func (h *JwtHandler) PlayerProfile(c *gin.Context) {
	me := middleware.GetUID(c)
	var req struct {
		UID uint `json:"uid"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.UID == 0 {
		resp.ParamError(c, "缺少目标用户")
		return
	}
	target := req.UID

	p := model.JwtPlayer{}
	if err := h.DB.Where("user_id = ?", target).First(&p).Error; err != nil {
		// 目标尚无精武堂档案：返回 8 槽"无"，装备区照常展示
		slotDefs := []string{"武器", "头盔", "盔甲", "战鞋", "项链", "手镯", "戒指", "勋章"}
		slots := make([]gin.H, 0, 8)
		for _, name := range slotDefs {
			slots = append(slots, gin.H{"slot": name, "item_name": "无"})
		}
		resp.OK(c, gin.H{"uid": target, "nick": "", "gender": "保密",
			"level": 1, "title_name": "无名小卒", "slots": slots, "practicing": 0,
			"practice_min": 0, "is_friend": 0})
		return
	}

	u := model.User{}
	h.DB.Select("nickname,gender").First(&u, target)
	gender := "保密"
	if u.Gender == 1 {
		gender = "男"
	} else if u.Gender == 2 {
		gender = "女"
	}

	// 装备八槽
	slots := []gin.H{}
	slotDefs := []struct {
		name string
		id   uint
	}{
		{"武器", p.WeaponID}, {"头盔", p.HelmetID}, {"盔甲", p.ArmorID}, {"战鞋", p.ShoesID},
		{"项链", p.NecklaceID}, {"手镯", p.BraceletID}, {"戒指", p.RingID}, {"勋章", p.MedalID},
	}
	for _, s := range slotDefs {
		name := "无"
		if s.id > 0 {
			if it := h.jwtItemByID(s.id); it != nil {
				name = it.Name
			}
		}
		slots = append(slots, gin.H{"slot": s.name, "item_name": name})
	}

	// 是否是我的好友 + 修炼状态
	var isFriend int
	h.DB.Raw(`SELECT COUNT(*) FROM friendships WHERE user_id = ? AND friend_id = ? AND status = 1`,
		me, target).Scan(&isFriend)
	practiceMin := 0
	if p.Practicing == 1 && p.PracticeEnd != nil {
		d := p.PracticeEnd.Sub(time.Now()).Minutes()
		if d > 0 {
			practiceMin = int(d)
		}
	}

	resp.OK(c, gin.H{"uid": target, "nick": p.Nick, "gender": gender, "level": p.Level,
		"title_name": jwtTitleName(p.Title), "slots": slots,
		"practicing": p.Practicing, "practice_min": practiceMin, "is_friend": isFriend})
}