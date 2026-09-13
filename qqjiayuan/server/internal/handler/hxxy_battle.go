package handler

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 战斗引擎（复刻 ltpk03.php 回合制：我方出手→敌方出手）
// 快照存 hxxy_battles，行动 attack/skill/catch/flee/item

// hxEnemy 敌方快照
type hxEnemy struct {
	ID    uint   `json:"id"`
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
	Kind  int    `json:"kind"` // 1npc 2boss
	Type  string `json:"type"` // npc/boss/dungeon
	ExpReward   int   `json:"exp_reward"`
	MoneyReward int64 `json:"money_reward"`
	Drops       string `json:"drops"`
	Difficulty  string `json:"difficulty"`
	// 副本信息
	DungeonID uint `json:"dungeon_id"`
	Floor     int  `json:"floor"`
}

// hxSelf 我方快照
type hxSelf struct {
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
	Sect  int    `json:"sect"`
	Pet   *hxPetSnap `json:"pet"`
}

type hxPetSnap struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Level int    `json:"level"`
	HP    int    `json:"hp"`
	MaxHP int    `json:"max_hp"`
	Atk   int    `json:"atk"`
	Def   int    `json:"def"`
	Mg    int    `json:"mg"`
	Mf    int    `json:"mf"`
}

// hxPetCombat 宠物战斗属性（照抄 cwztt.php 成长公式）
func (h *HxxyHandler) hxPetCombat(pet *model.HxxyPet) hxPetSnap {
	var sp model.HxxyPetSpecies
	if err := h.DB.First(&sp, pet.SpeciesID).Error; err != nil {
		return hxPetSnap{ID: pet.ID, Name: pet.Name, Level: pet.Level}
	}
	// 成长系数：星级每星 +15%，变异 ×1.5，品质 1-4 ×(1+0.2q)
	coef := (1.0 + 0.15*float64(pet.Star-1)) * (1.0 + 0.2*float64(pet.Quality-1))
	if pet.Mutate == 1 {
		coef *= 1.5
	}
	lv := float64(pet.Level)
	hp := ((lv+15)*(lv+15)*2 + float64(sp.MaxHP)) * coef
	atk := ((lv+1)*(lv+2)*3 + float64(sp.Atk)) * coef
	mg := ((lv+1)*(lv+2)*3 + float64(sp.Mg)) * coef
	def := ((lv+1)*(lv+1)*2 + float64(sp.Def)) * coef
	// 参战宠物资质取玩家属性，宠物与主人差距过大时压制
	snap := hxPetSnap{
		ID: pet.ID, Name: pet.Name, Level: pet.Level,
		HP: int(hp), MaxHP: int(hp),
		Atk: int(atk), Def: int(def), Mg: int(mg), Mf: int(def),
	}
	if pet.CurHP > 0 && pet.CurHP < snap.MaxHP {
		snap.HP = pet.CurHP
	}
	return snap
}

// hxCalcDmg 伤害结算（照抄 ltpk03.php）
// atk 攻击方攻击值 def 受方防御 gg 攻方元素攻和-受方元素防和 mult 技能倍率(百分比)
func hxCalcDmg(atk, def, gg, multPct int) (dmg int, crit bool) {
	mult := float64(multPct) / 100.0
	var s float64
	switch {
	case gg > 0: // 优势
		s = (float64(atk+100)*(1+float64(gg)/300) - float64(def)) * mult * 1.3
	case gg < 0: // 劣势
		s = (float64(atk) - (float64(def)+100)*(1+float64(-gg)/300)) * mult * 1.1
	default: // 均势
		s = (float64(atk+100) - float64(def)) * mult * 1.2
	}
	if s < 1 {
		s = 1
	}
	// 10% 暴击 ×(1+rand(1,20)/10)
	if rand.Intn(100) < 10 {
		crit = true
		s = s * (1 + float64(rand.Intn(20)+1)/10)
	}
	lo := int(s/2 + 0.5)
	hi := int(s)
	if hi < lo {
		hi = lo
	}
	dmg = lo + rand.Intn(hi-lo+1)
	if dmg < 1 {
		dmg = 1
	}
	return
}

// hxElemDiff 元素差（冰火雷三支求和）
func hxElemDiff(aBg, aHg, aLg, bBf, bHf, bLf int) int {
	return (aBg - bBf) + (aHg - bHf) + (aLg - bLf)
}

// hxNpcEnemy NPC → 敌方快照
func hxNpcEnemy(npc *model.HxxyNpc, diff, typ string, dungeonID uint, floor int) hxEnemy {
	return hxEnemy{
		ID: npc.ID, Name: npc.Name, Level: npc.Level,
		HP: npc.MaxHP, MaxHP: npc.MaxHP, MP: npc.MaxMP, MaxMP: npc.MaxMP,
		Atk: npc.Atk, Mg: npc.Mg, Def: npc.Def, Mf: npc.Mf,
		Bg: npc.Bg, Hg: npc.Hg, Lg: npc.Lg, Bf: npc.Bf, Hf: npc.Hf, Lf: npc.Lf,
		Kind: 1, Type: typ, ExpReward: npc.ExpReward, MoneyReward: int64(npc.MoneyReward),
		Drops: npc.Drops, Difficulty: diff, DungeonID: dungeonID, Floor: floor,
	}
}

// BattleStart 开始战斗（打怪）
func (h *HxxyHandler) BattleStart(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	// 气血归零不可开战（防止开出无法行动的战斗）
	if p.HP <= 0 {
		resp.ParamError(c, "你已身受重伤，先恢复气血再战斗！")
		return
	}
	// 有未完成战斗则直接进入
	if b := h.hxOpenBattle(p.ID); b != nil {
		resp.OK(c, h.hxBattleView(p, b))
		return
	}
	var in struct {
		NpcID uint `json:"npc_id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.NpcID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var npc model.HxxyNpc
	if err := h.DB.First(&npc, in.NpcID).Error; err != nil {
		resp.ParamError(c, "这里没有这个对手")
		return
	}
	// 功能NPC（村长/船夫等）也可被攻击：无预设奖励时按等级推算（与刷怪奖励公式一致）
	if npc.ExpReward == 0 {
		npc.ExpReward = npc.Level*npc.Level*3 + 50
		npc.MoneyReward = npc.Level*20 + 10
	}
	// 难度取刷怪表
	diff := "普通"
	var sp model.HxxySpawn
	if err := h.DB.Where("npc_id = ?", npc.ID).First(&sp).Error; err == nil {
		diff = sp.Difficulty
	}
	enemy := hxNpcEnemy(&npc, diff, "npc", 0, 0)
	self := h.hxSelfSnap(p)
	logs := []string{fmt.Sprintf("遭遇【%s】（%d级·%s），战斗开始！", enemy.Name, enemy.Level, enemy.Difficulty)}
	if npc.Take != "" {
		logs = append(logs, fmt.Sprintf("【%s】：%s", enemy.Name, npc.Take))
	}
	b := model.HxxyBattle{
		PlayerID: p.ID, Type: "npc", EnemyID: npc.ID, EnemyName: npc.Name,
		Round: 0, Status: 1,
	}
	b.Enemy = hxJSON(enemy)
	b.Self = hxJSON(self)
	b.Log = hxJSON(logs)
	h.DB.Create(&b)
	resp.OK(c, h.hxBattleView(p, &b))
}

// BossChallenge 挑战世界 BOSS
func (h *HxxyHandler) BossChallenge(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	if b := h.hxOpenBattle(p.ID); b != nil {
		resp.ParamError(c, "你正在战斗中")
		return
	}
	var in struct {
		BossID uint `json:"boss_id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.BossID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var boss model.HxxyBoss
	if err := h.DB.First(&boss, in.BossID).Error; err != nil {
		resp.ParamError(c, "没有这个BOSS")
		return
	}
	if boss.RespawnAt != nil && boss.RespawnAt.After(time.Now()) {
		resp.ParamError(c, fmt.Sprintf("【%s】刚被击败，%s 刷新", boss.Name, boss.RespawnAt.Format("15:04")))
		return
	}
	enemy := hxEnemy{
		ID: boss.ID, Name: boss.Name, Level: boss.Level,
		HP: boss.MaxHP, MaxHP: boss.MaxHP, MP: boss.MaxMP, MaxMP: boss.MaxMP,
		Atk: boss.Atk, Mg: boss.Mg, Def: boss.Def, Mf: boss.Mf,
		Bg: boss.Bg, Hg: boss.Hg, Lg: boss.Lg, Bf: boss.Bf, Hf: boss.Hf, Lf: boss.Lf,
		Kind: 2, Type: "boss",
		ExpReward: boss.Level * boss.Level * 100, MoneyReward: int64(boss.Level) * 500,
	}
	self := h.hxSelfSnap(p)
	logs := []string{fmt.Sprintf("你向世界BOSS【%s】发起了挑战！", boss.Name)}
	if boss.Take != "" {
		logs = append(logs, fmt.Sprintf("【%s】：%s", boss.Name, boss.Take))
	}
	b := model.HxxyBattle{
		PlayerID: p.ID, Type: "boss", EnemyID: boss.ID, EnemyName: boss.Name,
		Round: 0, Status: 1, Enemy: hxJSON(enemy), Self: hxJSON(self), Log: hxJSON(logs),
	}
	h.DB.Create(&b)
	resp.OK(c, h.hxBattleView(p, &b))
}

// DungeonEnter 进入副本下一层（逐层战斗）
func (h *HxxyHandler) DungeonEnter(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	if b := h.hxOpenBattle(p.ID); b != nil {
		resp.ParamError(c, "你正在战斗中")
		return
	}
	var in struct {
		DungeonID uint `json:"dungeon_id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.DungeonID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var dg model.HxxyDungeon
	if err := h.DB.First(&dg, in.DungeonID).Error; err != nil {
		resp.ParamError(c, "没有这个副本")
		return
	}
	if p.Level < dg.MinLevel {
		resp.ParamError(c, fmt.Sprintf("【%s】需要 %d 级才能进入", dg.Name, dg.MinLevel))
		return
	}
	// 每日次数
	var run model.HxxyDungeonRun
	h.DB.Where("player_id = ? AND dungeon_id = ?", p.ID, dg.ID).First(&run)
	today := time.Now().Format("2006-01-02")
	if run.DayDate != today {
		run.DayDate, run.CountToday = today, 0
	}
	if run.CountToday >= dg.Daily {
		resp.ParamError(c, fmt.Sprintf("今日【%s】次数已用完（每日 %d 次）", dg.Name, dg.Daily))
		return
	}
	floor := run.Floor + 1
	if floor > dg.Floors {
		resp.ParamError(c, fmt.Sprintf("【%s】已全部通关，明日再来！", dg.Name))
		return
	}
	// 选取与目标等级接近的困难怪
	targetLv := dg.MinLevel + (floor-1)*dg.Floors/dg.Floors*2 + floor/2
	var npc model.HxxyNpc
	err := h.DB.Where("kind = 1 AND level BETWEEN ? AND ?", targetLv, targetLv+8).Order("RAND()").First(&npc).Error
	if err != nil {
		h.DB.Where("kind = 1 AND level <= ?", targetLv+5).Order("level DESC").First(&npc)
	}
	if npc.ID == 0 {
		resp.ParamError(c, "副本怪物数据缺失")
		return
	}
	enemy := hxNpcEnemy(&npc, "困难", "dungeon", dg.ID, floor)
	enemy.ExpReward *= 2
	enemy.MoneyReward *= 2
	self := h.hxSelfSnap(p)
	logs := []string{fmt.Sprintf("【%s】第 %d/%d 层：守护者【%s】现身！", dg.Name, floor, dg.Floors, npc.Name)}
	b := model.HxxyBattle{
		PlayerID: p.ID, Type: "dungeon", EnemyID: npc.ID, EnemyName: npc.Name,
		Round: 0, Status: 1, Enemy: hxJSON(enemy), Self: hxJSON(self), Log: hxJSON(logs),
	}
	h.DB.Create(&b)
	if run.ID > 0 {
		h.DB.Model(&model.HxxyDungeonRun{}).Where("id = ?", run.ID).Updates(map[string]interface{}{"day_date": today, "count_today": run.CountToday + 1})
	} else {
		h.DB.Create(&model.HxxyDungeonRun{PlayerID: p.ID, DungeonID: dg.ID, Floor: 0, DayDate: today, CountToday: 1})
	}
	resp.OK(c, h.hxBattleView(p, &b))
}

// hxSelfSnap 我方快照（含参战宠物）
func (h *HxxyHandler) hxSelfSnap(p *model.HxxyPlayer) hxSelf {
	a := h.hxAttrs(p)
	s := hxSelf{
		Name: p.Name, Level: p.Level,
		HP: p.HP, MaxHP: a.MaxHP, MP: p.MP, MaxMP: a.MaxMP,
		Atk: a.Atk, Mg: a.Mg, Def: a.Def, Mf: a.Mf,
		Bg: a.Bg, Hg: a.Hg, Lg: a.Lg, Bf: a.Bf, Hf: a.Hf, Lf: a.Lf, Sect: p.Sect,
	}
	if pet := h.hxFightingPet(p.ID); pet != nil {
		ps := h.hxPetCombat(pet)
		s.Pet = &ps
	}
	return s
}

// hxOpenBattle 取进行中战斗
func (h *HxxyHandler) hxOpenBattle(playerID uint) *model.HxxyBattle {
	var b model.HxxyBattle
	if err := h.DB.Where("player_id = ? AND status = 1", playerID).First(&b).Error; err != nil {
		return nil
	}
	return &b
}

// BattleState 查询战斗
func (h *HxxyHandler) BattleState(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	b := h.hxOpenBattle(p.ID)
	if b == nil {
		resp.OK(c, gin.H{"in_battle": false})
		return
	}
	resp.OK(c, h.hxBattleView(p, b))
}

// hxBattleView 战斗视图
func (h *HxxyHandler) hxBattleView(p *model.HxxyPlayer, b *model.HxxyBattle) gin.H {
	var enemy hxEnemy
	var self hxSelf
	var logs []string
	json.Unmarshal([]byte(b.Enemy), &enemy)
	json.Unmarshal([]byte(b.Self), &self)
	json.Unmarshal([]byte(b.Log), &logs)
	// 已学技能（供出招）
	var skills []gin.H
	var ps []model.HxxyPlayerSkill
	h.DB.Where("player_id = ?", p.ID).Find(&ps)
	for _, l := range ps {
		var s model.HxxySkill
		if err := h.DB.First(&s, l.SkillID).Error; err == nil && s.Category == 1 && s.ID != 1 {
			skills = append(skills, gin.H{"skill_id": s.ID, "name": s.Name, "mp_cost": s.MpCost, "multiplier": s.Multiplier})
		}
	}
	return gin.H{
		"in_battle": b.Status == 1, "battle_id": b.ID, "type": b.Type, "round": b.Round,
		"status": b.Status, "enemy": enemy, "self": self, "log": logs, "skills": skills,
	}
}

// BattleAction 战斗行动
func (h *HxxyHandler) BattleAction(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	b := h.hxOpenBattle(p.ID)
	if b == nil {
		resp.ParamError(c, "当前没有进行中的战斗")
		return
	}
	var in struct {
		Act     string `json:"act"` // attack/skill/catch/flee/item
		SkillID uint   `json:"skill_id"`
		BagID   uint   `json:"bag_id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	var enemy hxEnemy
	var self hxSelf
	var logs []string
	json.Unmarshal([]byte(b.Enemy), &enemy)
	json.Unmarshal([]byte(b.Self), &self)
	json.Unmarshal([]byte(b.Log), &logs)
	// 快照异常（气血归零的遗留战斗）直接按战败结算，避免软锁
	if self.HP <= 0 || enemy.HP <= 0 {
		if self.HP <= 0 {
			self.HP = 0
			logs = append(logs, "你不敌败下阵来……")
			h.hxFinishBattle(p, b, &enemy, &self, logs, 3)
		} else {
			enemy.HP = 0
			logs = append(logs, fmt.Sprintf("【%s】被你击败了！", enemy.Name))
			h.hxFinishBattle(p, b, &enemy, &self, logs, 2)
		}
		resp.OK(c, h.hxBattleView(p, b))
		return
	}

	end := 0 // 2胜 3败 4逃
	switch in.Act {
	case "attack":
		hxRound(&enemy, &self, &logs, 77, "普攻") // 普攻攻击值=⌈max/1.3⌉ → 等效倍率 ≈77
	case "skill":
		var sk model.HxxySkill
		if err := h.DB.First(&sk, in.SkillID).Error; err != nil || sk.Category != 1 {
			resp.ParamError(c, "没有这个技能")
			return
		}
		var owned int64
		h.DB.Model(&model.HxxyPlayerSkill{}).Where("player_id = ? AND skill_id = ?", p.ID, sk.ID).Count(&owned)
		if owned == 0 {
			resp.ParamError(c, "你还没有学会【"+sk.Name+"】")
			return
		}
		if self.MP < sk.MpCost {
			resp.ParamError(c, "法力不足")
			return
		}
		self.MP -= sk.MpCost
		hxRound(&enemy, &self, &logs, sk.Multiplier, sk.Name)
	case "catch": // 捕捉（对玩家无效→此处仅 NPC）
		var owned int64
		h.DB.Model(&model.HxxyPlayerSkill{}).Where("player_id = ? AND skill_id = 3", p.ID).Count(&owned)
		if owned == 0 {
			resp.ParamError(c, "你没有学会【捕捉】，去杂货铺购买《宠物指南》学习吧")
			return
		}
		if enemy.Kind == 2 {
			logs = append(logs, "BOSS无法被捕捉！")
		} else if enemy.Level > p.Level {
			logs = append(logs, fmt.Sprintf("【%s】等级高于你，无法捕捉！", enemy.Name))
		} else {
			rate := int(0.1 + (1-float64(enemy.HP)/float64(enemy.MaxHP))*0.6*100)
			if rand.Intn(100) < rate {
				hxCreatePetFromNpc(h, p, &enemy)
				logs = append(logs, fmt.Sprintf("捕捉成功！【%s】成为了你的宠物，快去宠物页面看看吧！", enemy.Name))
				end = 2
				h.hxFinishBattle(p, b, &enemy, &self, logs, end)
				resp.OK(c, h.hxBattleView(p, b))
				return
			}
			logs = append(logs, fmt.Sprintf("捕捉失败！【%s】挣脱了（剩余气血越少成功率越高）", enemy.Name))
		}
	case "flee":
		if rand.Intn(100) < 60 {
			logs = append(logs, "你转身逃跑了！")
			end = 4
			h.hxFinishBattle(p, b, &enemy, &self, logs, end)
			resp.OK(c, h.hxBattleView(p, b))
			return
		}
		logs = append(logs, "逃跑失败！")
	case "item":
		if !h.hxBattleUseItem(p, &self, in.BagID, &logs) {
			resp.ParamError(c, "使用失败")
			return
		}
	default:
		resp.ParamError(c, "未知行动")
		return
	}

	// 敌方出手（若我方未胜未逃）
	if end == 0 && enemy.HP > 0 {
		atk := enemy.Atk
		if enemy.Mg > atk && (enemy.Mg > 0) {
			atk = enemy.Mg // 法系怪取魔攻
		}
		gg := hxElemDiff(enemy.Bg, enemy.Hg, enemy.Lg, self.Bf, self.Hf, self.Lf)
		dmg, crit := hxCalcDmg(atk, self.Def, gg, 100)
		self.HP -= dmg
		line := fmt.Sprintf("【%s】出手，对你造成 %d 点伤害", enemy.Name, dmg)
		if crit {
			line += "（暴击！）"
		}
		logs = append(logs, line)
		// 宠物替主反击一击
		if self.Pet != nil && self.Pet.HP > 0 {
			pdmg, pcrit := hxCalcDmg(self.Pet.Atk, enemy.Def, 0, 100)
			enemy.HP -= pdmg
			pl := fmt.Sprintf("宠物【%s】扑击，对【%s】造成 %d 点伤害", self.Pet.Name, enemy.Name, pdmg)
			if pcrit {
				pl += "（暴击！）"
			}
			logs = append(logs, pl)
		}
		if self.HP <= 0 {
			self.HP = 0
			logs = append(logs, "你不敌败下阵来……")
			end = 3
		} else if enemy.HP <= 0 {
			enemy.HP = 0
			logs = append(logs, fmt.Sprintf("【%s】被你击败了！", enemy.Name))
			end = 2
		}
	}

	// 更新快照
	b.Round++
	b.Enemy = hxJSON(enemy)
	b.Self = hxJSON(self)
	b.Log = hxJSON(logs)
	if end != 0 {
		h.hxFinishBattle(p, b, &enemy, &self, logs, end)
		resp.OK(c, h.hxBattleView(p, b))
		return
	}
	h.DB.Model(&model.HxxyBattle{}).Where("id = ?", b.ID).Updates(map[string]interface{}{
		"round": b.Round, "enemy": b.Enemy, "self": b.Self, "log": b.Log})
	// 同步玩家血蓝
	h.DB.Model(&model.HxxyPlayer{}).Where("id = ?", p.ID).Updates(map[string]interface{}{"hp": self.HP, "mp": self.MP})
	resp.OK(c, h.hxBattleView(p, b))
}

// hxRound 我方一击（act 侧）
func hxRound(enemy *hxEnemy, self *hxSelf, logs *[]string, multPct int, skillName string) {
	gg := hxElemDiff(self.Bg, self.Hg, self.Lg, enemy.Bf, enemy.Hf, enemy.Lf)
	dmg, crit := hxCalcDmg(self.Atk, enemy.Def, gg, multPct)
	enemy.HP -= dmg
	line := fmt.Sprintf("你使出【%s】，对【%s】造成 %d 点伤害", skillName, enemy.Name, dmg)
	if crit {
		line += "（暴击！）"
	}
	if enemy.HP <= 0 {
		enemy.HP = 0
		line += fmt.Sprintf("，【%s】被击败了！", enemy.Name)
	}
	*logs = append(*logs, line)
}

// hxCreatePetFromNpc 捕捉成功 → 生成宠物实例（种族快照=NPC 属性）
func hxCreatePetFromNpc(h *HxxyHandler, p *model.HxxyPlayer, e *hxEnemy) {
	sp := model.HxxyPetSpecies{
		ID: e.ID, Name: e.Name, Level: e.Level,
		HP: e.MaxHP / 2, MaxHP: e.MaxHP / 2, MP: e.MaxMP, MaxMP: e.MaxMP,
		Atk: e.Atk, Mg: e.Mg, Def: e.Def, Mf: e.Mf,
		Bg: e.Bg, Hg: e.Hg, Lg: e.Lg, Bf: e.Bf, Hf: e.Hf, Lf: e.Lf,
	}
	// 种族表按 npc id 补录（幂等：已存在跳过）
	var cnt int64
	h.DB.Model(&model.HxxyPetSpecies{}).Where("id = ?", sp.ID).Count(&cnt)
	if cnt == 0 {
		h.DB.Create(&sp)
	}
	mutate := 0
	if rand.Intn(100) < 5 {
		mutate = 1 // 5% 变异
	}
	pet := model.HxxyPet{
		PlayerID: p.ID, SpeciesID: sp.ID, Name: e.Name, Level: e.Level,
		Star: 1 + rand.Intn(3), Mutate: mutate, Quality: 1 + rand.Intn(4),
	}
	h.DB.Create(&pet)
}

// hxBattleUseItem 战斗中用药
func (h *HxxyHandler) hxBattleUseItem(p *model.HxxyPlayer, self *hxSelf, bagID uint, logs *[]string) bool {
	var b model.HxxyBag
	if err := h.DB.Where("id = ? AND player_id = ? AND store = 0 AND kind = 'item'", bagID, p.ID).First(&b).Error; err != nil {
		return false
	}
	var it model.HxxyItem
	if err := h.DB.First(&it, b.RefID).Error; err != nil {
		return false
	}
	var eff struct {
		HP  int `json:"hp"`
		MP  int `json:"mp"`
	}
	if it.Effect != "" {
		json.Unmarshal([]byte(it.Effect), &eff)
	}
	if eff.HP == 0 && eff.MP == 0 {
		return false
	}
	self.HP += eff.HP
	if self.HP > self.MaxHP {
		self.HP = self.MaxHP
	}
	self.MP += eff.MP
	if self.MP > self.MaxMP {
		self.MP = self.MaxMP
	}
	if b.Count == 1 {
		h.DB.Delete(&model.HxxyBag{}, b.ID)
	} else {
		h.DB.Model(&model.HxxyBag{}).Where("id = ?", b.ID).Update("count", b.Count-1)
	}
	*logs = append(*logs, fmt.Sprintf("你使用了【%s】", it.Name))
	return true
}

// hxFinishBattle 结算：奖励/掉落/副本进度/BOSS刷新/历史
func (h *HxxyHandler) hxFinishBattle(p *model.HxxyPlayer, b *model.HxxyBattle, enemy *hxEnemy, self *hxSelf, logs []string, result int) {
	exp, money := 0, int64(0)
	loot := []string{}
	if result == 2 {
		exp = enemy.ExpReward
		money = enemy.MoneyReward
		// VIP 练级祝福 +50%
		if p.Vip > 0 {
			exp = exp * 3 / 2
			h.DB.Model(&model.HxxyPlayer{}).Where("id = ?", p.ID).Update("vip", p.Vip-1)
			p.Vip--
		}
		// 掉落
		if enemy.Drops != "" {
			var drops []struct {
				Type string `json:"type"`
				ID   uint   `json:"id"`
				Rate int    `json:"rate"`
			}
			if json.Unmarshal([]byte(enemy.Drops), &drops) == nil {
				for _, d := range drops {
					if rand.Intn(10000) < d.Rate {
						bind := 0
						name := ""
						if d.Type == "item" {
							var it model.HxxyItem
							if h.DB.First(&it, d.ID).Error == nil {
								name = it.Name
								bind = it.Bind
							}
						} else {
							var eq model.HxxyEquip
							if h.DB.First(&eq, d.ID).Error == nil {
								name = eq.Name
								bind = eq.Bind
							}
						}
						if name != "" {
							h.hxBagAdd(p, d.Type, d.ID, 1, bind)
							loot = append(loot, name)
						}
					}
				}
			}
		}
		if len(loot) > 0 {
			logs = append(logs, "获得掉落："+joinCN(loot))
		}
		// 任务打怪计数
		h.hxQuestHuntProgress(p.ID, enemy.ID)
		// 每日战斗计数
		h.DB.Model(&model.HxxyPlayer{}).Where("id = ?", p.ID).UpdateColumn("day_battle", p.DayBattle+1)
		// 经验/银两
		if exp > 0 {
			_, lvMsg := h.hxGainExp(p, exp)
			logs = append(logs, fmt.Sprintf("获得经验 %d 点。%s", exp, lvMsg))
		}
		if money > 0 {
			h.hxWallet(p, "money", money, "战斗奖励")
			logs = append(logs, fmt.Sprintf("获得银两 %d。", money))
		}
		// 宠物经验
		if pet := h.hxFightingPet(p.ID); pet != nil {
			petExp := exp / 3
			if petExp > 0 {
				h.hxPetGainExp(pet, petExp)
			}
			h.DB.Model(&model.HxxyPet{}).Where("id = ?", pet.ID).Update("cur_hp", self.Pet.HP)
		}
		// 副本进度
		if b.Type == "dungeon" && enemy.DungeonID > 0 {
			h.DB.Model(&model.HxxyDungeonRun{}).
				Where("player_id = ? AND dungeon_id = ? AND floor < ?", p.ID, enemy.DungeonID, enemy.Floor).
				Update("floor", enemy.Floor)
			if enemy.Floor >= 10 {
				logs = append(logs, "副本已全部通关！")
			}
		}
		// BOSS 刷新
		if b.Type == "boss" {
			at := time.Now().Add(2 * time.Hour)
			h.DB.Model(&model.HxxyBoss{}).Where("id = ?", enemy.ID).Update("respawn_at", at)
		}
	} else if result == 3 {
		if b.Type == "pvp" || b.Type == "tower" {
			// 比武/通天塔战败：原地重伤（银两转移在 hxPvpSettle 处理，层数重置在 hxTowerSettle 处理）
			h.DB.Model(&model.HxxyPlayer{}).Where("id = ?", p.ID).Update("hp", 1)
			logs = append(logs, "你已身受重伤，气血只剩 1 点。")
		} else {
			// 战败：回城复活，扣 10% 银两
			penalty := p.Money / 10
			if penalty > 0 {
				h.hxWallet(p, "money", -penalty, "战败抚恤")
			}
			h.DB.Model(&model.HxxyPlayer{}).Where("id = ?", p.ID).Updates(map[string]interface{}{"map_x": 0, "map_y": 0, "hp": 1})
			logs = append(logs, fmt.Sprintf("你被送回了新手村休养，损失 %d 银两。", penalty))
		}
	}

	// 同步玩家血蓝
	h.DB.Model(&model.HxxyPlayer{}).Where("id = ?", p.ID).Updates(map[string]interface{}{"hp": self.HP, "mp": self.MP})
	// 归档
	b.Status = result
	b.Enemy = hxJSON(*enemy)
	b.Self = hxJSON(*self)
	b.Log = hxJSON(logs)
	h.DB.Model(&model.HxxyBattle{}).Where("id = ?", b.ID).Updates(map[string]interface{}{
		"status": b.Status, "round": b.Round, "enemy": b.Enemy, "self": b.Self, "log": b.Log})
	// 历史记录
	h.DB.Create(&model.HxxyBattleLog{
		PlayerID: p.ID, Type: b.Type, EnemyName: enemy.Name, Result: result,
		Round: b.Round, Exp: exp, Money: int(money), Loot: hxJSON(loot),
	})
	// 首页消息区通知（复刻原版系统动态）
	switch result {
	case 2:
		msg := fmt.Sprintf("你成功击败了【%s】，获得经验 %d、银两 %d。", enemy.Name, exp, money)
		if len(loot) > 0 {
			msg += "掉落：" + joinCN(loot)
		}
		h.hxNotify(p.ID, msg)
	case 3:
		if b.Type != "pvp" && b.Type != "tower" { // 比武/通天塔的败讯由各自结算函数发送
			h.hxNotify(p.ID, fmt.Sprintf("你败给了【%s】，被送回新手村休养。", enemy.Name))
		}
	case 4:
		h.hxNotify(p.ID, fmt.Sprintf("你从【%s】手中成功逃脱。", enemy.Name))
	}
	// 玩法专属结算（通天塔进度 / 比武银两转移与恶名 / 国战权杖与内奸）
	if b.Type == "tower" {
		h.hxTowerSettle(p, result)
	} else if b.Type == "pvp" && (result == 2 || result == 3) {
		h.hxPvpSettle(p, b, enemy, result)
	} else if b.Type == "gz" {
		h.hxGzSettle(p, b, enemy, result)
	}
}

// hxPetGainExp 宠物升级（经验=主人经验/3，升级属性随 cwztt 公式自动成长）
func (h *HxxyHandler) hxPetGainExp(pet *model.HxxyPet, exp int) {
	pet.Exp += exp
	for pet.Exp >= pet.Level*80 {
		pet.Exp -= pet.Level * 80
		pet.Level++
	}
	h.DB.Model(&model.HxxyPet{}).Where("id = ?", pet.ID).Updates(map[string]interface{}{"level": pet.Level, "exp": pet.Exp})
}

func hxJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func joinCN(items []string) string {
	out := ""
	for i, s := range items {
		if i > 0 {
			out += "、"
		}
		out += s
	}
	return out
}
