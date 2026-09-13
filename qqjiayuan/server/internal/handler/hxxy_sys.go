package handler

import (
	"fmt"
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
		base := gin.H{"quest_id": q.ID, "name": q.Name, "desc": q.Desc, "type": q.Type,
			"count": q.Count, "exp": q.ExpReward, "money": q.MoneyReward, "target": target}
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

// Dungeons 副本列表
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
		cnt := 0
		if run.DayDate == today {
			cnt = run.CountToday
		}
		list = append(list, gin.H{
			"dungeon_id": dg.ID, "name": dg.Name, "desc": dg.Desc, "floors": dg.Floors,
			"min_level": dg.MinLevel, "daily": dg.Daily, "used_today": cnt,
			"floor": run.Floor, "locked": p.Level < dg.MinLevel,
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

// Cultivate 修炼状态
func (h *HxxyHandler) Cultivate(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	resp.OK(c, gin.H{"switch": p.XiulianSwitch, "exp": p.XiulianExp, "cap": hxXiulianCap(p.Level),
		"desc": "开启修炼后，使用经验类道具获得的经验将存入修炼池；修炼池满后自动转为等级经验。"})
}

// CultivateToggle 修炼开关
func (h *HxxyHandler) CultivateToggle(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	sw := 1
	msg := "修炼已开启！经验将存入修炼池。"
	if p.XiulianSwitch == 1 {
		sw = 0
		msg = "修炼已关闭。"
	}
	h.DB.Model(&model.HxxyPlayer{}).Where("id = ?", p.ID).Update("xiulian_switch", sw)
	p.XiulianSwitch = sw
	resp.OK(c, gin.H{"msg": msg, "switch": sw})
}

// Titles 头衔列表（已激活 + 可激活）
func (h *HxxyHandler) Titles(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var owned []model.HxxyPlayerTitle
	h.DB.Where("player_id = ?", p.ID).Find(&owned)
	ownedMap := map[uint]bool{}
	mine := []gin.H{}
	for _, o := range owned {
		var t model.HxxyTitle
		if err := h.DB.First(&t, o.TitleID).Error; err == nil {
			ownedMap[t.ID] = true
			mine = append(mine, gin.H{"title_id": t.ID, "name": t.Name, "desc": t.Desc,
				"worn": p.TitleID == t.ID})
		}
	}
	var all []model.HxxyTitle
	h.DB.Order("id").Limit(60).Find(&all)
	store := []gin.H{}
	for _, t := range all {
		if ownedMap[t.ID] {
			continue
		}
		store = append(store, gin.H{"title_id": t.ID, "name": t.Name, "desc": t.Desc,
			"price": int64(t.HP+t.Atk+t.Def+t.Mg)/10 + 1000})
	}
	resp.OK(c, gin.H{"mine": mine, "store": store, "worn": p.TitleID})
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
	resp.OK(c, gin.H{"msg": fmt.Sprintf("头衔【%s】激活成功！去佩戴吧。", t.Name)})
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

// Signin 签到（连签递增，第 7 天送金豆）
func (h *HxxyHandler) Signin(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	today := time.Now().Format("2006-01-02")
	if p.DaySignin == 1 {
		resp.ParamError(c, "今日已签到")
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
	msg := fmt.Sprintf("连续签到 %d 天，获得 %d 银两！", streak, money)
	if beans > 0 {
		msg += fmt.Sprintf("额外获得 %d 金豆！", beans)
	}
	resp.OK(c, gin.H{"msg": msg})
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
		resp.ParamError(c, "充值码无效（演示码：XY666 / VIP666）")
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
