package handler

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// ---- 精武堂管理（游戏管理/精武堂：玩家/道具/技能/帮派/聊天） ----

// AdminJwtPlayers 精武堂玩家列表（word：家园号精确 / 昵称模糊）
func (h *AdminHandler) AdminJwtPlayers(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	word := strings.TrimSpace(c.Query("word"))
	q := h.DB.Model(&model.JwtPlayer{})
	if word != "" {
		if uid, err := strconv.Atoi(word); err == nil {
			q = q.Where("user_id = ?", uid)
		} else {
			q = q.Where("nick LIKE ?", "%"+word+"%")
		}
	}
	var total int64
	q.Count(&total)
	var list []model.JwtPlayer
	q.Order("id DESC").Offset(offset).Limit(size).Find(&list)
	// 批量取货币与创建时间
	uids := map[uint]bool{}
	for _, p := range list {
		uids[p.UserID] = true
	}
	coins := map[uint]int{}
	yb := map[uint]int{}
	created := map[uint]string{}
	if len(uids) > 0 {
		keys := make([]uint, 0, len(uids))
		for k := range uids {
			keys = append(keys, k)
		}
		var us []model.User
		h.DB.Select("id,coins,yuanbao,created_at").Where("id IN ?", keys).Find(&us)
		for _, u := range us {
			coins[u.ID] = u.Coins
			yb[u.ID] = u.YuanBao
			created[u.ID] = u.CreatedAt.Format("2006-01-02 15:04:05")
		}
	}
	out := make([]gin.H, 0, len(list))
	for _, p := range list {
		out = append(out, gin.H{
			"id": p.ID, "user_id": p.UserID, "nick": p.Nick, "sex": p.Sex,
			"level": p.Level, "exp": p.Exp, "next_exp": p.Level*64,
			"title": p.Title, "title_name": jwtTitleName(p.Title),
			"energy": p.Energy, "honor": p.Honor, "gang_id": p.GangID,
			"e_hp": p.EHp, "e_mp": p.EMp, "e_spd": p.ESpd, "e_atk": p.EAtk, "e_def": p.EDef,
			"cur_hp": p.CurHp, "cur_mp": p.CurMp,
			"practicing": p.Practicing, "starter": p.Starter,
			"coins": coins[p.UserID], "yuanbao": yb[p.UserID], "created": created[p.UserID],
		})
	}
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": out})
}

// AdminJwtPlayerDetail 精武堂玩家详情（档案+货币+装备八槽+背包）
func (h *AdminHandler) AdminJwtPlayerDetail(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("uid"))
	var p model.JwtPlayer
	if err := h.DB.Where("user_id = ?", uid).First(&p).Error; err != nil {
		resp.NotFound(c, "玩家不存在")
		return
	}
	var u model.User
	h.DB.Select("nickname,coins,yuanbao").First(&u, p.UserID)
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
			var it model.JwtItem
			if h.DB.First(&it, s.id).Error == nil {
				name = it.Name
			}
		}
		slots = append(slots, gin.H{"slot": s.name, "item_id": s.id, "item_name": name})
	}
	var bags []model.JwtBag
	h.DB.Where("user_id = ? AND amount > 0", uid).Order("id ASC").Find(&bags)
	if bags == nil {
		bags = []model.JwtBag{}
	}
	resp.OK(c, gin.H{
		"player": p, "nickname": u.Nickname, "coins": u.Coins, "yuanbao": u.YuanBao,
		"title_name": jwtTitleName(p.Title), "slots": slots, "bag": bags,
	})
}

// AdminJwtPlayerUpdate 编辑精武堂玩家（等级/经验/能量/分配/气血/头衔 + 货币设置）
func (h *AdminHandler) AdminJwtPlayerUpdate(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("uid"))
	var p model.JwtPlayer
	if err := h.DB.Where("user_id = ?", uid).First(&p).Error; err != nil {
		resp.NotFound(c, "玩家不存在")
		return
	}
	var req struct {
		Level   int `json:"level"`
		Exp     int `json:"exp"`
		Energy  int `json:"energy"`
		EHp     int `json:"e_hp"`
		EMp     int `json:"e_mp"`
		ESpd    int `json:"e_spd"`
		EAtk    int `json:"e_atk"`
		EDef    int `json:"e_def"`
		CurHp   int `json:"cur_hp"`
		CurMp   int `json:"cur_mp"`
		Title   int `json:"title"`
		Coins   int `json:"coins"`
		Yuanbao int `json:"yuanbao"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if req.Level < 1 {
		req.Level = 1
	}
	if req.Exp < 0 {
		req.Exp = 0
	}
	if req.Energy < 0 {
		req.Energy = 0
	}
	if req.Title < 0 {
		req.Title = 0
	}
	if req.Title > 10 {
		req.Title = 10
	}
	h.DB.Model(&model.JwtPlayer{}).Where("id = ?", p.ID).Updates(map[string]interface{}{
		"level": req.Level, "exp": req.Exp, "energy": req.Energy,
		"e_hp": req.EHp, "e_mp": req.EMp, "e_spd": req.ESpd, "e_atk": req.EAtk, "e_def": req.EDef,
		"cur_hp": req.CurHp, "cur_mp": req.CurMp, "title": req.Title,
	})
	// 货币差异记账
	var u model.User
	h.DB.Select("coins,yuanbao").First(&u, uid)
	dc := req.Coins - u.Coins
	dy := req.Yuanbao - u.YuanBao
	up := map[string]interface{}{}
	if dc != 0 {
		up["coins"] = req.Coins
	}
	if dy != 0 {
		up["yuanbao"] = req.Yuanbao
	}
	if len(up) > 0 {
		h.DB.Model(&model.User{}).Where("id = ?", uid).Updates(up)
		if dc != 0 {
			addWalletLog(h.DB, uint(uid), "jwt", "管理端调整G币", "jwt_coins", dc)
		}
		if dy != 0 {
			addWalletLog(h.DB, uint(uid), "jwt", "管理端调整元宝", "jwt_yuanbao", dy)
		}
	}
	resp.OK(c, gin.H{"msg": "玩家数据已保存"})
}

// ---- 道具管理 ----

// AdminJwtItems 道具列表（word：名称模糊；cat：分类；src：来源）
func (h *AdminHandler) AdminJwtItems(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	word := c.Query("word")
	cat := c.Query("cat")
	src := c.Query("src")
	q := h.DB.Model(&model.JwtItem{})
	if word != "" {
		q = q.Where("name LIKE ?", "%"+word+"%")
	}
	if cat != "" {
		q = q.Where("cat = ?", cat)
	}
	if src != "" {
		q = q.Where("src = ?", src)
	}
	var total int64
	q.Count(&total)
	var list []model.JwtItem
	q.Order("id ASC").Offset(offset).Limit(size).Find(&list)
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": list})
}

// AdminJwtItemCreate 新增道具
func (h *AdminHandler) AdminJwtItemCreate(c *gin.Context) {
	var it model.JwtItem
	if err := c.ShouldBindJSON(&it); err != nil || strings.TrimSpace(it.Name) == "" {
		resp.ParamError(c, "道具名必填")
		return
	}
	it.ID = 0
	it.Name = strings.TrimSpace(it.Name)
	if it.Cat == "" {
		it.Cat = "other"
	}
	if it.Src == "" {
		it.Src = "shop"
	}
	if it.Currency == "" {
		it.Currency = "coins"
	}
	h.DB.Create(&it)
	resp.OK(c, gin.H{"msg": "道具「" + it.Name + "」已创建", "id": it.ID})
}

// AdminJwtItemUpdate 更新道具
func (h *AdminHandler) AdminJwtItemUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var old model.JwtItem
	if err := h.DB.First(&old, id).Error; err != nil {
		resp.NotFound(c, "道具不存在")
		return
	}
	var req model.JwtItem
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		resp.ParamError(c, "道具名必填")
		return
	}
	h.DB.Model(&old).Updates(map[string]interface{}{
		"name": strings.TrimSpace(req.Name), "cat": req.Cat, "src": req.Src, "price": req.Price,
		"currency": req.Currency, "level": req.Level, "atk": req.Atk, "def": req.Def,
		"hp": req.Hp, "mp": req.Mp, "spd": req.Spd, "hit": req.Hit, "crit": req.Crit,
		"dodge": req.Dodge, "recover_hp": req.RecoverHp, "recover_mp": req.RecoverMp,
		"desc": req.Desc, "status": req.Status, "mats": req.Mats, "fee": req.Fee,
	})
	resp.OK(c, gin.H{"msg": "道具已保存"})
}

// AdminJwtItemDelete 删除道具（清理背包与已装备槽位）
func (h *AdminHandler) AdminJwtItemDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var it model.JwtItem
	if err := h.DB.First(&it, id).Error; err != nil {
		resp.NotFound(c, "道具不存在")
		return
	}
	h.DB.Where("item_id = ?", id).Delete(&model.JwtBag{})
	for _, col := range []string{"weapon_id", "helmet_id", "armor_id", "shoes_id",
		"necklace_id", "bracelet_id", "ring_id", "medal_id"} {
		h.DB.Model(&model.JwtPlayer{}).Where(col+" = ?", id).Update(col, 0)
	}
	h.DB.Delete(&model.JwtItem{}, id)
	resp.OK(c, gin.H{"msg": "道具「" + it.Name + "」已删除"})
}

// ---- 技能管理 ----

// AdminJwtSkills 技能列表（word：名称模糊；act：1主动 0被动）
func (h *AdminHandler) AdminJwtSkills(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	word := c.Query("word")
	q := h.DB.Model(&model.JwtSkill{})
	if word != "" {
		q = q.Where("name LIKE ?", "%"+word+"%")
	}
	if act, err := strconv.Atoi(c.Query("act")); err == nil && (act == 0 || act == 1) {
		q = q.Where("act = ?", act)
	}
	var total int64
	q.Count(&total)
	var list []model.JwtSkill
	q.Order("id ASC").Offset(offset).Limit(size).Find(&list)
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": list})
}

// AdminJwtSkillCreate 新增技能
func (h *AdminHandler) AdminJwtSkillCreate(c *gin.Context) {
	var s model.JwtSkill
	if err := c.ShouldBindJSON(&s); err != nil || strings.TrimSpace(s.Name) == "" {
		resp.ParamError(c, "技能名必填")
		return
	}
	s.ID = 0
	s.Name = strings.TrimSpace(s.Name)
	if s.Currency == "" {
		s.Currency = "coins"
	}
	if s.WeaponReq == "" {
		s.WeaponReq = "无限制"
	}
	h.DB.Create(&s)
	resp.OK(c, gin.H{"msg": "技能「" + s.Name + "」已创建", "id": s.ID})
}

// AdminJwtSkillUpdate 更新技能
func (h *AdminHandler) AdminJwtSkillUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var old model.JwtSkill
	if err := h.DB.First(&old, id).Error; err != nil {
		resp.NotFound(c, "技能不存在")
		return
	}
	var req model.JwtSkill
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		resp.ParamError(c, "技能名必填")
		return
	}
	h.DB.Model(&old).Updates(map[string]interface{}{
		"name": strings.TrimSpace(req.Name), "act": req.Act, "level": req.Level, "price": req.Price,
		"currency": req.Currency, "weapon_req": req.WeaponReq, "coef": req.Coef, "hit": req.Hit,
		"crit": req.Crit, "crit_mul": req.CritMul, "dodge_add": req.DodgeAdd,
		"p_atk": req.PAtk, "p_def": req.PDef, "p_hp": req.PHp, "p_mp": req.PMp,
		"desc": req.Desc, "status": req.Status,
	})
	resp.OK(c, gin.H{"msg": "技能已保存"})
}

// AdminJwtSkillDelete 删除技能（清理已学技能）
func (h *AdminHandler) AdminJwtSkillDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var s model.JwtSkill
	if err := h.DB.First(&s, id).Error; err != nil {
		resp.NotFound(c, "技能不存在")
		return
	}
	h.DB.Where("skill_id = ?", id).Delete(&model.JwtLearnedSkill{})
	h.DB.Delete(&model.JwtSkill{}, id)
	resp.OK(c, gin.H{"msg": "技能「" + s.Name + "」已删除"})
}

// ---- 帮派管理 ----

// AdminJwtGangs 帮派列表（word：帮派名/帮主模糊）
func (h *AdminHandler) AdminJwtGangs(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	word := c.Query("word")
	q := h.DB.Model(&model.JwtGang{})
	if word != "" {
		q = q.Where("name LIKE ? OR master LIKE ?", "%"+word+"%", "%"+word+"%")
	}
	var total int64
	q.Count(&total)
	var list []model.JwtGang
	q.Order("level DESC, id ASC").Offset(offset).Limit(size).Find(&list)
	if list == nil {
		list = []model.JwtGang{}
	}
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": list})
}

// AdminJwtGangDelete 解散帮派（清理成员/申请，玩家帮派归属置空）
func (h *AdminHandler) AdminJwtGangDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var g model.JwtGang
	if err := h.DB.First(&g, id).Error; err != nil {
		resp.NotFound(c, "帮派不存在")
		return
	}
	h.DB.Delete(&model.JwtGang{}, id)
	h.DB.Where("gang_id = ?", id).Delete(&model.JwtGangMember{})
	h.DB.Where("gang_id = ?", id).Delete(&model.JwtGangApply{})
	h.DB.Model(&model.JwtPlayer{}).Where("gang_id = ?", id).Update("gang_id", 0)
	resp.OK(c, gin.H{"msg": "帮派「" + g.Name + "」已解散"})
}

// ---- 聊天记录 ----

// AdminJwtChats 精武堂聊天记录（word：家园号精确 / 昵称或内容模糊）
func (h *AdminHandler) AdminJwtChats(c *gin.Context) {
	page, offset, size := pageOf(c, 15)
	word := c.Query("word")
	q := h.DB.Model(&model.JwtChat{})
	if word != "" {
		if uid, err := strconv.Atoi(word); err == nil {
			q = q.Where("user_id = ?", uid)
		} else {
			q = q.Where("nick LIKE ? OR content LIKE ?", "%"+word+"%", "%"+word+"%")
		}
	}
	var total int64
	q.Count(&total)
	var list []model.JwtChat
	q.Order("id DESC").Offset(offset).Limit(size).Find(&list)
	if list == nil {
		list = []model.JwtChat{}
	}
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": list})
}

// AdminJwtChatDelete 删除聊天记录
func (h *AdminHandler) AdminJwtChatDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.JwtChat{}, id)
	resp.OK(c, gin.H{"msg": "聊天记录已删除"})
}

// AdminJwtRecords 比武记录列表（word：我的昵称/对手昵称模糊，可按 home 号过滤）
func (h *AdminHandler) AdminJwtRecords(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	word := strings.TrimSpace(c.Query("word"))
	q := h.DB.Model(&model.JwtArenaRecord{})
	if word != "" {
		if uid, err := strconv.Atoi(word); err == nil {
			q = q.Where("my_uid = ? OR opp_uid = ?", uid, uid)
		} else {
			q = q.Where("my_nick LIKE ? OR opp_nick LIKE ?", "%"+word+"%", "%"+word+"%")
		}
	}
	var total int64
	q.Count(&total)
	var list []model.JwtArenaRecord
	q.Order("id DESC").Offset(offset).Limit(size).Find(&list)
	out := make([]gin.H, 0, len(list))
	for _, r := range list {
		out = append(out, gin.H{
			"id": r.ID, "user_id": r.UserID, "my_uid": r.MyUID, "my_nick": r.MyNick,
			"opp_uid": r.OppUID, "opp_nick": r.OppNick, "result": r.Result,
			"exp": r.Exp, "coin": r.Coin, "my_level": r.MyLevel, "opp_level": r.OppLevel,
			"created_at": r.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": out})
}

// AdminJwtRecordDetail 比武记录详情（含整场战报）
func (h *AdminHandler) AdminJwtRecordDetail(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var r model.JwtArenaRecord
	if err := h.DB.First(&r, id).Error; err != nil {
		resp.NotFound(c, "记录不存在")
		return
	}
	resp.OK(c, gin.H{"detail": gin.H{
		"id": r.ID, "user_id": r.UserID, "my_uid": r.MyUID, "my_nick": r.MyNick,
		"opp_uid": r.OppUID, "opp_nick": r.OppNick, "result": r.Result,
		"exp": r.Exp, "coin": r.Coin,
		"my_level": r.MyLevel, "my_max_hp": r.MyMaxHp, "my_cur_hp": r.MyCurHp,
		"opp_level": r.OppLevel, "opp_max_hp": r.OppMaxHp, "opp_cur_hp": r.OppCurHp,
		"logs": r.LogsText, "created_at": r.CreatedAt.Format("2006-01-02 15:04:05"),
	}})
}

// AdminJwtRecordDelete 删除比武记录
func (h *AdminHandler) AdminJwtRecordDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.JwtArenaRecord{}, id)
	resp.OK(c, gin.H{"msg": "比武记录已删除"})
}
