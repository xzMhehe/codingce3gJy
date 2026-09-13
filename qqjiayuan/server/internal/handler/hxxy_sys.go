package handler

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 技能/宠物/任务/副本/BOSS/修炼/头衔/签到/排行

// Skills 技能列表（已学+书店）
func (h *HxxyHandler) Skills(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var learned []model.HxxyPlayerSkill
	h.DB.Where("player_id = ?", p.ID).Find(&learned)
	// 兜底：老角色若一个技能都没有，补发初始技能（普攻 + 本门派技能），
	// 与原版一致——每个玩家至少拥有基础技能，避免技能页/战斗快捷键可选项为空
	if len(learned) == 0 {
		var ids []uint
		h.DB.Model(&model.HxxySkill{}).
			Where("category = 1 AND (sect = 0 OR sect = ?) AND id <> 1", p.Sect).
			Pluck("id", &ids)
		ids = append(ids, 1) // 普攻
		for _, sid := range ids {
			var cnt int64
			h.DB.Model(&model.HxxyPlayerSkill{}).Where("player_id = ? AND skill_id = ?", p.ID, sid).Count(&cnt)
			if cnt == 0 {
				h.DB.Create(&model.HxxyPlayerSkill{PlayerID: p.ID, SkillID: sid, Level: 1})
			}
		}
		h.DB.Where("player_id = ?", p.ID).Find(&learned)
	}
	owned := map[uint]bool{}
	mine := []gin.H{}
	for _, l := range learned {
		var s model.HxxySkill
		if err := h.DB.First(&s, l.SkillID).Error; err == nil {
			owned[s.ID] = true
			mine = append(mine, gin.H{"skill_id": s.ID, "name": s.Name, "desc": s.Desc,
				"category": s.Category, "mp_cost": s.MpCost, "multiplier": s.Multiplier})
		}
	}
	// 技能书店：本门派可学未学
	var shop []model.HxxySkill
	h.DB.Where("category = 1 AND (sect = 0 OR sect = ?)", p.Sect).Order("id").Find(&shop)
	store := []gin.H{}
	for _, s := range shop {
		if owned[s.ID] {
			continue
		}
		store = append(store, gin.H{"skill_id": s.ID, "name": s.Name, "desc": s.Desc,
			"mp_cost": s.MpCost, "multiplier": s.Multiplier, "learn_level": s.LearnLevel,
			"price": int64(s.Multiplier*20 + 500)})
	}
	resp.OK(c, gin.H{"mine": mine, "store": store})
}

// SkillLearn 学习技能（银两）
func (h *HxxyHandler) SkillLearn(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		SkillID uint `json:"skill_id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.SkillID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var s model.HxxySkill
	if err := h.DB.First(&s, in.SkillID).Error; err != nil || s.Category != 1 {
		resp.ParamError(c, "没有这个技能")
		return
	}
	if s.Sect >= 1 && s.Sect <= 5 && s.Sect != p.Sect {
		resp.ParamError(c, fmt.Sprintf("【%s】为%s专属技能", s.Name, hxSectNames[s.Sect]))
		return
	}
	if p.Level < s.LearnLevel {
		resp.ParamError(c, fmt.Sprintf("需要 %d 级才能学习【%s】", s.LearnLevel, s.Name))
		return
	}
	var cnt int64
	h.DB.Model(&model.HxxyPlayerSkill{}).Where("player_id = ? AND skill_id = ?", p.ID, s.ID).Count(&cnt)
	if cnt > 0 {
		resp.ParamError(c, "你已经学会了")
		return
	}
	price := int64(s.Multiplier*20 + 500)
	if p.Money < price {
		resp.ParamError(c, fmt.Sprintf("学费 %d 银两，银两不足", price))
		return
	}
	h.hxWallet(p, "money", -price, "学习技能【"+s.Name+"】")
	h.DB.Create(&model.HxxyPlayerSkill{PlayerID: p.ID, SkillID: s.ID, Level: 1})
	resp.OK(c, gin.H{"msg": fmt.Sprintf("学会【%s】！消耗 %d 银两。", s.Name, price)})
}

// Pets 宠物列表
func (h *HxxyHandler) Pets(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var pets []model.HxxyPet
	h.DB.Where("player_id = ?", p.ID).Order("id").Find(&pets)
	list := []gin.H{}
	for i := range pets {
		sp := h.hxPetCombat(&pets[i])
		var spName string
		h.DB.Model(&model.HxxyPetSpecies{}).Select("name").Where("id = ?", pets[i].SpeciesID).Scan(&spName)
		list = append(list, gin.H{
			"id": pets[i].ID, "name": pets[i].Name, "level": pets[i].Level, "exp": pets[i].Exp,
			"exp_need": pets[i].Level * 80, "star": pets[i].Star, "mutate": pets[i].Mutate,
			"quality": pets[i].Quality, "fighting": pets[i].Fighting,
			"hp": pets[i].CurHP, "max_hp": sp.MaxHP, "atk": sp.Atk, "def": sp.Def, "mg": sp.Mg,
		})
	}
	resp.OK(c, gin.H{"pets": list})
}

// PetAct 宠物操作：fight参战 rest休息 free放生 rename改名 heal治疗
func (h *HxxyHandler) PetAct(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		Act   string `json:"act"`
		PetID uint   `json:"pet_id"`
		Name  string `json:"name"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.PetID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var pet model.HxxyPet
	if err := h.DB.Where("id = ? AND player_id = ?", in.PetID, p.ID).First(&pet).Error; err != nil {
		resp.ParamError(c, "宠物不存在")
		return
	}
	switch in.Act {
	case "fight":
		h.DB.Model(&model.HxxyPet{}).Where("player_id = ?", p.ID).Update("fighting", 0)
		h.DB.Model(&model.HxxyPet{}).Where("id = ?", pet.ID).Update("fighting", 1)
		resp.OK(c, gin.H{"msg": fmt.Sprintf("【%s】已参战，战斗中会与你并肩作战！", pet.Name)})
	case "rest":
		h.DB.Model(&model.HxxyPet{}).Where("id = ?", pet.ID).Update("fighting", 0)
		resp.OK(c, gin.H{"msg": fmt.Sprintf("【%s】回去休息了。", pet.Name)})
	case "free":
		h.DB.Delete(&model.HxxyPet{}, pet.ID)
		resp.OK(c, gin.H{"msg": fmt.Sprintf("你含泪放生了【%s】……", pet.Name)})
	case "rename":
		in.Name = trimStr(in.Name, 10)
		if in.Name == "" {
			resp.ParamError(c, "请输入新名字")
			return
		}
		h.DB.Model(&model.HxxyPet{}).Where("id = ?", pet.ID).Update("name", in.Name)
		resp.OK(c, gin.H{"msg": "宠物改名成功！"})
	case "heal":
		cost := int64(pet.Level * 20)
		if p.Money < cost {
			resp.ParamError(c, fmt.Sprintf("治疗需要 %d 银两", cost))
			return
		}
		h.hxWallet(p, "money", -cost, "宠物治疗")
		h.DB.Model(&model.HxxyPet{}).Where("id = ?", pet.ID).Update("cur_hp", 0)
		resp.OK(c, gin.H{"msg": fmt.Sprintf("花了 %d 银两为【%s】恢复了气血！", cost, pet.Name)})
	default:
		resp.ParamError(c, "未知操作")
	}
}

// Quests 任务列表（可接/进行中/已完成）
func (h *HxxyHandler) Quests(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var all []model.HxxyQuest
	h.DB.Order("id").Find(&all)
	var mine []model.HxxyPlayerQuest
	h.DB.Where("player_id = ?", p.ID).Find(&mine)
	st := map[uint]model.HxxyPlayerQuest{}
	for _, m := range mine {
		st[m.QuestID] = m
	}
	available, active, done := []gin.H{}, []gin.H{}, []gin.H{}
	for _, q := range all {
		target := h.hxQuestTargetName(&q)
		from := ""
		if q.FromNpc > 0 {
			from = h.hxNpcName(q.FromNpc)
		}
		base := gin.H{"quest_id": q.ID, "name": q.Name, "desc": q.Desc, "type": q.Type, "category": q.Category,
			"count": q.Count, "exp": q.ExpReward, "money": q.MoneyReward, "bean": q.BeanReward, "target": target, "from": from}
		if q.ItemReward > 0 {
			if q.ItemEquip == 1 {
				var e model.HxxyEquip
				if h.DB.First(&e, q.ItemReward).Error == nil {
					base["item_name"] = e.Name
				}
			} else {
				var it model.HxxyItem
				if h.DB.First(&it, q.ItemReward).Error == nil {
					base["item_name"] = it.Name
				}
			}
		}
		if m, ok := st[q.ID]; ok {
			if m.Status == 3 {
				done = append(done, base)
			} else {
				b2 := gin.H(base)
				b2["progress"] = m.Progress
				b2["status"] = m.Status
				active = append(active, b2)
			}
			continue
		}
		if p.Level >= q.MinLevel {
			available = append(available, base)
		}
	}
	resp.OK(c, gin.H{"available": available, "active": active, "done": done})
}

// hxNpcName NPC名
func (h *HxxyHandler) hxNpcName(id uint) string {
	var n string
	h.DB.Model(&model.HxxyNpc{}).Select("name").Where("id = ?", id).Scan(&n)
	return n
}

// QuestAbandon 放弃任务（未完成的可放弃，进度清零）
func (h *HxxyHandler) QuestAbandon(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		QuestID uint `json:"quest_id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.QuestID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	res := h.DB.Where("player_id = ? AND quest_id = ? AND status <> 3", p.ID, in.QuestID).Delete(&model.HxxyPlayerQuest{})
	if res.RowsAffected == 0 {
		resp.ParamError(c, "你没有进行中的这个任务")
		return
	}
	resp.OK(c, gin.H{"msg": "你放弃了该任务。"})
}

// QuestAccept 接任务
func (h *HxxyHandler) QuestAccept(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		QuestID uint `json:"quest_id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.QuestID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var q model.HxxyQuest
	if err := h.DB.First(&q, in.QuestID).Error; err != nil {
		resp.ParamError(c, "任务不存在")
		return
	}
	if p.Level < q.MinLevel {
		resp.ParamError(c, "等级不足")
		return
	}
	var cnt int64
	h.DB.Model(&model.HxxyPlayerQuest{}).Where("player_id = ? AND quest_id = ?", p.ID, q.ID).Count(&cnt)
	if cnt > 0 {
		resp.ParamError(c, "该任务已在进行或已完成")
		return
	}
	h.DB.Create(&model.HxxyPlayerQuest{PlayerID: p.ID, QuestID: q.ID, Status: 1})
	resp.OK(c, gin.H{"msg": "接受任务【" + q.Name + "】：" + q.Desc})
}

// QuestSubmit 提交任务（hunt 按进度 collect 收集物品 talk 直接完成）
func (h *HxxyHandler) QuestSubmit(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		QuestID uint `json:"quest_id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.QuestID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var q model.HxxyQuest
	if err := h.DB.First(&q, in.QuestID).Error; err != nil {
		resp.ParamError(c, "任务不存在")
		return
	}
	var m model.HxxyPlayerQuest
	if err := h.DB.Where("player_id = ? AND quest_id = ?", p.ID, q.ID).First(&m).Error; err != nil {
		resp.ParamError(c, "你还没有接这个任务")
		return
	}
	if m.Status == 3 {
		resp.ParamError(c, "该任务已完成")
		return
	}
	if q.Type == "collect" {
		// 检查并扣除收集物
		var b model.HxxyBag
		if err := h.DB.Where("player_id = ? AND kind = 'item' AND ref_id = ? AND store = 0", p.ID, q.TargetID).First(&b).Error; err != nil || b.Count < q.Count {
			resp.ParamError(c, "收集物不足")
			return
		}
		h.hxBagSub(p.ID, b.ID, q.Count)
	} else if q.Type == "hunt" && m.Progress < q.Count {
		resp.ParamError(c, fmt.Sprintf("还差 %d 个目标未击败", q.Count-m.Progress))
	}
	// 奖励
	h.hxWallet(p, "money", q.MoneyReward, "任务奖励【"+q.Name+"】")
	msg := fmt.Sprintf("任务【%s】完成！获得经验 %d、银两 %d。", q.Name, q.ExpReward, q.MoneyReward)
	if q.ExpReward > 0 {
		_, lvMsg := h.hxGainExp(p, q.ExpReward)
		if lvMsg != "" {
			msg += lvMsg
		}
	}
	if q.BeanReward > 0 {
		h.hxWallet(p, "beans", int64(q.BeanReward), "任务奖励")
		msg += fmt.Sprintf("金豆 +%d。", q.BeanReward)
	}
	if q.ItemReward > 0 {
		kind := "item"
		name := ""
		if q.ItemEquip == 1 {
			kind = "equip"
			var e model.HxxyEquip
			h.DB.First(&e, q.ItemReward)
			name = e.Name
		} else {
			var it model.HxxyItem
			h.DB.First(&it, q.ItemReward)
			name = it.Name
		}
		h.hxBagAdd(p, kind, q.ItemReward, 1, 0)
		msg += fmt.Sprintf("获得【%s】×1。", name)
	}
	h.DB.Model(&model.HxxyPlayerQuest{}).Where("id = ?", m.ID).Update("status", 3)
	// 日常任务奖励国家贡献/经验/声望（复刻 rcrw26-28：+10/+10/+10）
	if q.Category == 3 {
		h.hxGangReward(p, 10, 10, 10)
		msg += "获得国家贡献+10、国家经验+10、国家声望+10。"
	}
	// 后续任务提示
	if q.NextQuest > 0 {
		msg += "有新的任务可以接取了！"
	}
	resp.OK(c, gin.H{"msg": msg})
}

// hxQuestTargetName 任务目标名（hunt/talk→NPC名 collect→物品名）
func (h *HxxyHandler) hxQuestTargetName(q *model.HxxyQuest) string {
	if q.TargetID == 0 {
		return ""
	}
	if q.Type == "collect" {
		var it model.HxxyItem
		if err := h.DB.First(&it, q.TargetID).Error; err == nil {
			return it.Name
		}
		return ""
	}
	var n model.HxxyNpc
	if err := h.DB.First(&n, q.TargetID).Error; err == nil {
		return n.Name
	}
	return ""
}

// hxQuestHuntProgress 打怪计数（战斗胜利后调用）
func (h *HxxyHandler) hxQuestHuntProgress(playerID, npcID uint) {
	var qs []model.HxxyPlayerQuest
	h.DB.Where("player_id = ? AND status = 1", playerID).Find(&qs)
	for _, m := range qs {
		var q model.HxxyQuest
		if err := h.DB.First(&q, m.QuestID).Error; err != nil || q.Type != "hunt" || q.TargetID != npcID {
			continue
		}
		progress := m.Progress + 1
		st := 1
		if progress >= q.Count {
			st = 2
		}
		h.DB.Model(&model.HxxyPlayerQuest{}).Where("id = ?", m.ID).Updates(map[string]interface{}{"progress": progress, "status": st})
	}
}

// Dungeons 副本列表（复刻原版 fb/*：五副本×四难度，激活→杀5守护BOSS→完成）
func (h *HxxyHandler) Dungeons(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var dgs []model.HxxyDungeon
	h.DB.Order("id").Find(&dgs)
	today := time.Now().Format("2006-01-02")
	list := []gin.H{}
	for _, dg := range dgs {
		var run model.HxxyDungeonRun
		h.DB.Where("player_id = ? AND dungeon_id = ?", p.ID, dg.ID).First(&run)
		kills, done := 0, false
		if run.DayDate == today {
			kills = run.Floor
			done = run.Done == 1
		}
		list = append(list, gin.H{
			"dungeon_id": dg.ID, "name": dg.Name, "desc": dg.Desc,
			"floors": dg.Floors, "min_level": dg.MinLevel,
			"kills": kills, "done": done, "locked": p.Level < dg.MinLevel,
		})
	}
	resp.OK(c, gin.H{"dungeons": list})
}

// Bosses BOSS 列表
func (h *HxxyHandler) Bosses(c *gin.Context) {
	var bs []model.HxxyBoss
	h.DB.Order("id").Find(&bs)
	list := []gin.H{}
	for _, b := range bs {
		alive := b.RespawnAt == nil || b.RespawnAt.Before(time.Now())
		item := gin.H{"boss_id": b.ID, "name": b.Name, "level": b.Level,
			"max_hp": b.MaxHP, "atk": b.Atk, "def": b.Def, "alive": alive}
		if !alive {
			item["respawn_at"] = b.RespawnAt.Format("15:04")
		}
		list = append(list, item)
	}
	resp.OK(c, gin.H{"bosses": list})
}

// Titles 称号一览（复刻 xy477.php：分页20/页编号列表，已获得红字/未获得黑字；mine 为已激活头衔供佩戴管理）
func (h *HxxyHandler) Titles(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	page := 1
	if v := c.Query("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			page = n
		}
	}
	const size = 20
	var ownedIDs []uint
	h.DB.Model(&model.HxxyPlayerTitle{}).Where("player_id = ?", p.ID).Pluck("title_id", &ownedIDs)
	ownedSet := map[uint]bool{}
	for _, id := range ownedIDs {
		ownedSet[id] = true
	}
	var mine []gin.H
	for _, id := range ownedIDs {
		var t model.HxxyTitle
		if err := h.DB.First(&t, id).Error; err == nil {
			mine = append(mine, gin.H{"title_id": t.ID, "name": t.Name, "desc": t.Desc,
				"price": int64(t.HP+t.Atk+t.Def+t.Mg)/10 + 1000, "owned": true})
		}
	}
	var total int64
	h.DB.Model(&model.HxxyTitle{}).Count(&total)
	totalPages := int((total + size - 1) / size)
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	var all []model.HxxyTitle
	h.DB.Order("id").Offset((page - 1) * size).Limit(size).Find(&all)
	list := []gin.H{}
	for _, t := range all {
		list = append(list, gin.H{"title_id": t.ID, "name": t.Name, "desc": t.Desc,
			"price": int64(t.HP+t.Atk+t.Def+t.Mg)/10 + 1000, "owned": ownedSet[t.ID]})
	}
	resp.OK(c, gin.H{"list": list, "mine": mine, "worn": p.TitleID,
		"page": page, "total_pages": totalPages, "count": total})
}

// TitleActivate 激活头衔（银两）
func (h *HxxyHandler) TitleActivate(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		TitleID uint `json:"title_id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.TitleID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var t model.HxxyTitle
	if err := h.DB.First(&t, in.TitleID).Error; err != nil {
		resp.ParamError(c, "头衔不存在")
		return
	}
	var cnt int64
	h.DB.Model(&model.HxxyPlayerTitle{}).Where("player_id = ? AND title_id = ?", p.ID, t.ID).Count(&cnt)
	if cnt > 0 {
		resp.ParamError(c, "已激活该头衔")
		return
	}
	price := int64(t.HP+t.Atk+t.Def+t.Mg)/10 + 1000
	if p.Money < price {
		resp.ParamError(c, fmt.Sprintf("激活需要 %d 银两", price))
		return
	}
	h.hxWallet(p, "money", -price, "激活头衔【"+t.Name+"】")
	h.DB.Create(&model.HxxyPlayerTitle{PlayerID: p.ID, TitleID: t.ID})
	resp.OK(c, gin.H{"msg": fmt.Sprintf("恭喜你！获得称号%s，永久增加属性，快去佩戴吧！", t.Name)})
}

// TitleWear 佩戴头衔
func (h *HxxyHandler) TitleWear(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		TitleID uint `json:"title_id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if in.TitleID == 0 {
		h.DB.Model(&model.HxxyPlayer{}).Where("id = ?", p.ID).Update("title_id", 0)
		resp.OK(c, gin.H{"msg": "已摘下头衔"})
		return
	}
	var cnt int64
	h.DB.Model(&model.HxxyPlayerTitle{}).Where("player_id = ? AND title_id = ?", p.ID, in.TitleID).Count(&cnt)
	if cnt == 0 {
		resp.ParamError(c, "请先激活该头衔")
		return
	}
	h.DB.Model(&model.HxxyPlayer{}).Where("id = ?", p.ID).Update("title_id", in.TitleID)
	var t model.HxxyTitle
	h.DB.First(&t, in.TitleID)
	resp.OK(c, gin.H{"msg": "佩戴头衔【" + t.Name + "】！"})
}

// Signin 签到（复刻原版：月累计签到，2/5/10/15/25 次阶梯奖励，每月1日清零）
func (h *HxxyHandler) Signin(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	today := time.Now().Format("2006-01-02")
	if p.DaySignin == 1 {
		resp.ParamError(c, "对不起！你今日已经签到过了")
		return
	}
	// 连签天数
	var last model.HxxySignin
	streak := 1
	if err := h.DB.Where("player_id = ?", p.ID).Order("id DESC").First(&last).Error; err == nil {
		yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
		if last.Day == yesterday {
			streak = last.Streak + 1
		}
	}
	if streak > 7 {
		streak = 7
	}
	money := int64(100 * streak)
	beans := 0
	if streak == 7 {
		beans = 10
	}
	h.hxWallet(p, "money", money, "签到奖励")
	if beans > 0 {
		h.hxWallet(p, "beans", int64(beans), "连签7天奖励")
	}
	h.DB.Create(&model.HxxySignin{PlayerID: p.ID, Day: today, Streak: streak,
		Reward: fmt.Sprintf("银两%d+金豆%d", money, beans)})
	h.DB.Model(&model.HxxyPlayer{}).Where("id = ?", p.ID).Update("day_signin", 1)
	// 全服公告（复刻原版 msgg02.php）
	h.hxWorldMsg("恭喜玩家" + p.Name + "完成了每日签到任务，累计签到有豪礼相送哦！！")
	msg := "恭喜你！签到成功！！"
	if beans > 0 {
		msg += fmt.Sprintf("连续签到 %d 天，获得 %d 银两 + %d 金豆！", streak, money, beans)
	} else {
		msg += fmt.Sprintf("连续签到 %d 天，获得 %d 银两！", streak, money)
	}
	resp.OK(c, gin.H{"msg": msg})
}

// hxHxTiers 阶梯奖励表
var hxSignTiers = []struct {
	Tier  int
	Money int64
	Beans int64
}{
	{2, 2000, 5}, {5, 5000, 10}, {10, 15000, 20}, {15, 40000, 30}, {25, 100000, 50},
}

// SigninInfo 签到页数据（复刻原版 xy409.php）
func (h *HxxyHandler) SigninInfo(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	month := time.Now().Format("2006-01")
	var cnt int64
	h.DB.Model(&model.HxxySignin{}).Where("player_id = ? AND day LIKE ?", p.ID, month+"%").Count(&cnt)
	var claims []model.HxxySigninClaim
	h.DB.Where("player_id = ? AND month = ?", p.ID, month).Find(&claims)
	claimed := map[int]bool{}
	for _, cl := range claims {
		claimed[cl.Tier] = true
	}
	tiers := []gin.H{}
	for _, t := range hxSignTiers {
		tiers = append(tiers, gin.H{"tier": t.Tier, "money": t.Money, "beans": t.Beans, "claimed": claimed[t.Tier]})
	}
	resp.OK(c, gin.H{
		"month":        int(time.Now().Month()),
		"month_cn":     fmt.Sprintf("%d月", int(time.Now().Month())),
		"count":        cnt,
		"today_signed": p.DaySignin == 1,
		"tiers":        tiers,
	})
}

// SigninClaim 领取累计签到奖励（复刻原版 xy411.php）
func (h *HxxyHandler) SigninClaim(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		Tier int `json:"tier"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.Tier <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var tier *struct {
		Tier  int
		Money int64
		Beans int64
	}
	for i := range hxSignTiers {
		if hxSignTiers[i].Tier == in.Tier {
			tier = &hxSignTiers[i]
			break
		}
	}
	if tier == nil {
		resp.ParamError(c, "没有这个签到奖励")
		return
	}
	month := time.Now().Format("2006-01")
	var cnt int64
	h.DB.Model(&model.HxxySignin{}).Where("player_id = ? AND day LIKE ?", p.ID, month+"%").Count(&cnt)
	if int(cnt) < tier.Tier {
		resp.ParamError(c, fmt.Sprintf("对不起！你的签到次数不足（差%d天）", tier.Tier-int(cnt)))
		return
	}
	var existed int64
	h.DB.Model(&model.HxxySigninClaim{}).Where("player_id = ? AND month = ? AND tier = ?", p.ID, month, tier.Tier).Count(&existed)
	if existed > 0 {
		resp.ParamError(c, fmt.Sprintf("对不起！你已领取过%d次签到奖励了", tier.Tier))
		return
	}
	h.DB.Create(&model.HxxySigninClaim{PlayerID: p.ID, Month: month, Tier: tier.Tier})
	if tier.Money > 0 {
		h.hxWallet(p, "money", tier.Money, fmt.Sprintf("%d次签到奖励", tier.Tier))
	}
	if tier.Beans > 0 {
		h.hxWallet(p, "beans", tier.Beans, fmt.Sprintf("%d次签到奖励", tier.Tier))
	}
	h.hxWorldMsg(fmt.Sprintf("恭喜玩家%s领取到了%d次签到奖励！！获得了大量奖励！！", p.Name, tier.Tier))
	resp.OK(c, gin.H{"msg": fmt.Sprintf("恭喜你！领取到了%d次签到奖励（银两%d+金豆%d）", tier.Tier, tier.Money, tier.Beans)})
}

// Rank 排行（level/money/pets）
func (h *HxxyHandler) Rank(c *gin.Context) {
	typ := c.Param("type")
	type row struct {
		ID    uint
		Name  string
		Val   int64
		Level int
	}
	var rows []row
	switch typ {
	case "level":
		h.DB.Model(&model.HxxyPlayer{}).Select("id, name, level, level as val").Order("level DESC, exp DESC").Limit(20).Scan(&rows)
	case "money":
		h.DB.Model(&model.HxxyPlayer{}).Select("id, name, money as val, level").Order("money DESC").Limit(20).Scan(&rows)
	case "pets":
		h.DB.Raw(`SELECT p.id, p.name, pe.level, pe.level as val FROM hxxy_pets pe JOIN hxxy_players p ON p.id = pe.player_id ORDER BY pe.level DESC LIMIT 20`).Scan(&rows)
	default:
		resp.ParamError(c, "未知排行类型")
		return
	}
	list := []gin.H{}
	for i, r := range rows {
		list = append(list, gin.H{"rank": i + 1, "player_id": r.ID, "name": r.Name, "val": r.Val, "level": r.Level})
	}
	resp.OK(c, gin.H{"type": typ, "list": list})
}

// WalletLogs 货币流水
func (h *HxxyHandler) WalletLogs(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var logs []model.HxxyWalletLog
	h.DB.Where("player_id = ?", p.ID).Order("id DESC").Limit(50).Find(&logs)
	resp.OK(c, gin.H{"logs": logs})
}

// VipRecharge 演示充值码（XY666→金豆 / VIP666→练级祝福）
func (h *HxxyHandler) VipRecharge(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		Code string `json:"code"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	switch in.Code {
	case "XY666":
		h.hxWallet(p, "beans", 10, "充值码XY666")
		resp.OK(c, gin.H{"msg": "充值成功！金豆 +10（演示充值码，每日可用）"})
	case "VIP666":
		h.DB.Model(&model.HxxyPlayer{}).Where("id = ?", p.ID).Update("vip", p.Vip+30)
		p.Vip += 30
		resp.OK(c, gin.H{"msg": "充值成功！VIP练级祝福 +30 分钟（打怪经验 1.5 倍）"})
	default:
		// 演示会员等级充值码：SVIP0~SVIP20 设置 VIP 等级
		if len(in.Code) >= 4 && in.Code[:4] == "SVIP" {
			if lv, err := strconv.Atoi(in.Code[4:]); err == nil && lv >= 0 && lv <= 20 {
				h.DB.Model(&model.HxxyPlayer{}).Where("id = ?", p.ID).Update("vip_lv", lv)
				resp.OK(c, gin.H{"msg": fmt.Sprintf("充值成功！会员等级已提升为 VIP%d 级（演示码：SVIP0~SVIP20）", lv)})
				return
			}
		}
		resp.ParamError(c, "充值码无效（演示码：XY666 / VIP666 / SVIP0~20）")
	}
}

// BattleLogs 战斗历史
func (h *HxxyHandler) BattleLogs(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var logs []model.HxxyBattleLog
	h.DB.Where("player_id = ?", p.ID).Order("id DESC").Limit(30).Find(&logs)
	resp.OK(c, gin.H{"logs": logs})
}

// trimStr 截断字符串
func trimStr(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}
