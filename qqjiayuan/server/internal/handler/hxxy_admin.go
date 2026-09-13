package handler

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 幻想西游管理端（玩家/数据/发放/战斗流水/钱包流水）

// AdminXyPlayers 玩家列表（word=角色名/ID/家园用户ID 模糊）
func (h *AdminHandler) AdminXyPlayers(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	word := strings.TrimSpace(c.Query("word"))
	q := h.DB.Model(&model.HxxyPlayer{})
	if word != "" {
		if uid, err := strconv.Atoi(word); err == nil {
			q = q.Where("id = ? OR user_id = ?", uid, uid)
		} else {
			q = q.Where("name LIKE ?", "%"+word+"%")
		}
	}
	var total int64
	q.Count(&total)
	var rows []model.HxxyPlayer
	q.Order("id DESC").Offset(offset).Limit(size).Find(&rows)
	// 补充家园账号昵称
	type rowOut struct {
		model.HxxyPlayer
		HomeNick string `json:"home_nick"`
	}
	out := []rowOut{}
	for _, p := range rows {
		var nick string
		h.DB.Model(&model.User{}).Select("nickname").Where("id = ?", p.UserID).Scan(&nick)
		out = append(out, rowOut{HxxyPlayer: p, HomeNick: nick})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// AdminXyPlayerDetail 玩家详情（档案+背包+最近战斗）
func (h *AdminHandler) AdminXyPlayerDetail(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var p model.HxxyPlayer
	if err := h.DB.First(&p, id).Error; err != nil {
		resp.NotFound(c, "玩家不存在")
		return
	}
	var bag []model.HxxyBag
	h.DB.Where("player_id = ?", p.ID).Order("id DESC").Limit(100).Find(&bag)
	var battles []model.HxxyBattleLog
	h.DB.Where("player_id = ?", p.ID).Order("id DESC").Limit(20).Find(&battles)
	var pets []model.HxxyPet
	h.DB.Where("player_id = ?", p.ID).Find(&pets)
	var homeNick string
	h.DB.Model(&model.User{}).Select("nickname").Where("id = ?", p.UserID).Scan(&homeNick)
	resp.OK(c, gin.H{"player": p, "home_nick": homeNick, "bag": bag, "battles": battles, "pets": pets})
}

// AdminXyPlayerUpdate 修改玩家（等级/银两/金豆/血蓝/恶名/名字/位置）
func (h *AdminHandler) AdminXyPlayerUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var p model.HxxyPlayer
	if err := h.DB.First(&p, id).Error; err != nil {
		resp.NotFound(c, "玩家不存在")
		return
	}
	var in struct {
		Name  *string `json:"name"`
		Level *int    `json:"level"`
		Money *int64  `json:"money"`
		Bank  *int64  `json:"bank"`
		Beans *int    `json:"beans"`
		HP    *int    `json:"hp"`
		MP    *int    `json:"mp"`
		Emz   *int    `json:"emz"`
		Vip   *int    `json:"vip"`
		MapX  *int    `json:"map_x"`
		MapY  *int    `json:"map_y"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	updates := map[string]interface{}{}
	if in.Name != nil && strings.TrimSpace(*in.Name) != "" {
		updates["name"] = trimStr(strings.TrimSpace(*in.Name), 12)
	}
	if in.Level != nil && *in.Level >= 1 {
		updates["level"] = *in.Level
	}
	if in.Money != nil {
		updates["money"] = *in.Money
	}
	if in.Bank != nil {
		updates["bank"] = *in.Bank
	}
	if in.Beans != nil {
		updates["beans"] = *in.Beans
	}
	if in.HP != nil {
		updates["hp"] = *in.HP
	}
	if in.MP != nil {
		updates["mp"] = *in.MP
	}
	if in.Emz != nil {
		updates["emz"] = *in.Emz
	}
	if in.Vip != nil {
		updates["vip"] = *in.Vip
	}
	if in.MapX != nil {
		updates["map_x"] = *in.MapX
	}
	if in.MapY != nil {
		updates["map_y"] = *in.MapY
	}
	if len(updates) > 0 {
		h.DB.Model(&model.HxxyPlayer{}).Where("id = ?", p.ID).Updates(updates)
	}
	resp.OK(c, gin.H{"msg": "修改成功"})
}

// AdminXyGrant 发放道具/装备（入背包）
func (h *AdminHandler) AdminXyGrant(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var p model.HxxyPlayer
	if err := h.DB.First(&p, id).Error; err != nil {
		resp.NotFound(c, "玩家不存在")
		return
	}
	var in struct {
		Kind  string `json:"kind"` // item/equip
		RefID uint   `json:"ref_id"`
		Count int    `json:"count"`
		Beans int    `json:"beans"` // 可选：同时发金豆
		Money int64  `json:"money"` // 可选：同时发银两
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.RefID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	if in.Count <= 0 {
		in.Count = 1
	}
	hxg := &HxxyHandler{DB: h.DB}
	if in.Kind == "equip" {
		var e model.HxxyEquip
		if err := h.DB.First(&e, in.RefID).Error; err != nil {
			resp.ParamError(c, "装备不存在")
			return
		}
		hxg.hxBagAdd(&p, "equip", e.ID, in.Count, e.Bind)
		resp.OK(c, gin.H{"msg": "已发放装备【" + e.Name + "】×" + strconv.Itoa(in.Count)})
		return
	}
	var it model.HxxyItem
	if err := h.DB.First(&it, in.RefID).Error; err != nil {
		resp.ParamError(c, "物品不存在")
		return
	}
	hxg.hxBagAdd(&p, "item", it.ID, in.Count, it.Bind)
	msg := "已发放物品【" + it.Name + "】×" + strconv.Itoa(in.Count)
	if in.Money > 0 {
		hxg.hxWallet(&p, "money", in.Money, "管理员发放")
		msg += "，银两 +" + strconv.FormatInt(in.Money, 10)
	}
	if in.Beans > 0 {
		hxg.hxWallet(&p, "beans", int64(in.Beans), "管理员发放")
		msg += "，金豆 +" + strconv.Itoa(in.Beans)
	}
	resp.OK(c, gin.H{"msg": msg})
}

// AdminXyPlayerDelete 删除角色
func (h *AdminHandler) AdminXyPlayerDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var p model.HxxyPlayer
	if err := h.DB.First(&p, id).Error; err != nil {
		resp.NotFound(c, "玩家不存在")
		return
	}
	h.DB.Where("player_id = ?", p.ID).Delete(&model.HxxyBag{})
	h.DB.Where("player_id = ?", p.ID).Delete(&model.HxxyPet{})
	h.DB.Where("player_id = ?", p.ID).Delete(&model.HxxyPlayerSkill{})
	h.DB.Where("player_id = ?", p.ID).Delete(&model.HxxyPlayerQuest{})
	h.DB.Where("player_id = ?", p.ID).Delete(&model.HxxyBattleLog{})
	h.DB.Where("player_id = ?", p.ID).Delete(&model.HxxyWalletLog{})
	h.DB.Delete(&model.HxxyPlayer{}, p.ID)
	resp.OK(c, gin.H{"msg": "角色及数据已删除"})
}

// xyTableDef 游戏数据表定义（查询+增删改共用；Fields 为可编辑字段白名单）
type xyTableDef struct {
	Model  interface{}
	Fields map[string]string // 字段名(json) -> 类型 int/int64/string
}

// xyTableDefs 全部可管理数据表
var xyTableDefs = map[string]xyTableDef{
	"items": {&model.HxxyItem{}, map[string]string{
		"name": "string", "desc": "string", "category": "int", "price": "int", "bean_price": "int",
		"level": "int", "weight": "int", "bind": "int", "effect": "string",
	}},
	"equips": {&model.HxxyEquip{}, map[string]string{
		"name": "string", "desc": "string", "category": "int", "sect": "int", "slot": "int",
		"level": "int", "weight": "int", "bind": "int", "price": "int", "bean_price": "int",
		"hp": "int", "atk": "int", "mg": "int", "def": "int",
		"bg": "int", "hg": "int", "lg": "int", "bf": "int", "hf": "int", "lf": "int",
	}},
	"npcs": {&model.HxxyNpc{}, map[string]string{
		"name": "string", "take": "string", "drops": "string",
		"level": "int", "kind": "int", "hp": "int", "max_hp": "int", "mp": "int", "max_mp": "int",
		"atk": "int", "mg": "int", "def": "int", "mf": "int",
		"bg": "int", "hg": "int", "lg": "int", "bf": "int", "hf": "int", "lf": "int",
		"exp_reward": "int", "money_reward": "int",
	}},
	"skills": {&model.HxxySkill{}, map[string]string{
		"name": "string", "desc": "string", "category": "int", "sect": "int",
		"mp_cost": "int", "multiplier": "int", "learn_level": "int",
	}},
	"maps": {&model.HxxyMapNode{}, map[string]string{
		"name": "string", "desc": "string", "dtx": "int", "dty": "int",
		"up": "string", "down": "string", "left": "string", "right": "string",
		"up_jump": "string", "down_jump": "string", "left_jump": "string", "right_jump": "string",
	}},
	"bosses": {&model.HxxyBoss{}, map[string]string{
		"name": "string", "take": "string", "level": "int",
		"hp": "int", "max_hp": "int", "mp": "int", "max_mp": "int",
		"atk": "int", "mg": "int", "def": "int", "mf": "int",
		"bg": "int", "hg": "int", "lg": "int", "bf": "int", "hf": "int", "lf": "int",
	}},
	"pets": {&model.HxxyPetSpecies{}, map[string]string{
		"name": "string", "level": "int", "star": "int", "quality": "int",
		"hp": "int", "max_hp": "int", "mp": "int", "max_mp": "int",
		"atk": "int", "mg": "int", "def": "int", "mf": "int",
		"bg": "int", "hg": "int", "lg": "int", "bf": "int", "hf": "int", "lf": "int",
	}},
	"titles": {&model.HxxyTitle{}, map[string]string{
		"name": "string", "desc": "string", "hp": "int", "atk": "int", "def": "int", "mg": "int",
	}},
	"spawns": {&model.HxxySpawn{}, map[string]string{
		"name": "string", "difficulty": "string", "dtx": "int", "dty": "int", "npc_id": "int",
	}},
	"mapnpcs": {&model.HxxyMapNpc{}, map[string]string{
		"name": "string", "img": "string", "dialogue": "string", "shop": "string", "teles": "string",
		"dtx": "int", "dty": "int", "npc_id": "int",
	}},
}

// AdminXyData 游戏数据分页查询（table=items/equips/npcs/skills/spawns/maps/bosses/pets/titles/mapnpcs）
func (h *AdminHandler) AdminXyData(c *gin.Context) {
	table := c.Param("table")
	page, offset, size := pageOf(c, 10)
	word := strings.TrimSpace(c.Query("word"))
	def, ok := xyTableDefs[table]
	if !ok {
		resp.ParamError(c, "未知数据表")
		return
	}
	var total int64
	q := h.DB.Model(def.Model)
	if word != "" {
		if id, err := strconv.Atoi(word); err == nil {
			q = q.Where("id = ?", id)
		} else {
			q = q.Where("name LIKE ?", "%"+word+"%")
		}
	}
	q.Count(&total)
	var rows []map[string]interface{}
	lq := h.DB.Model(def.Model)
	if word != "" {
		if id, err := strconv.Atoi(word); err == nil {
			lq = lq.Where("id = ?", id)
		} else {
			lq = lq.Where("name LIKE ?", "%"+word+"%")
		}
	}
	lq.Order("id").Offset(offset).Limit(size).Find(&rows)
	resp.OK(c, gin.H{"list": rows, "total": total, "page": page, "size": size})
}

// xyConvVal 按白名单类型转换 JSON 值
func xyConvVal(v interface{}, typ string) (interface{}, bool) {
	switch typ {
	case "string":
		s, ok := v.(string)
		return s, ok
	default: // int / int64（JSON 数字均为 float64）
		f, ok := v.(float64)
		if !ok {
			return nil, false
		}
		if typ == "int64" {
			return int64(f), true
		}
		return int(f), true
	}
}

// xyPickVals 从请求体提取白名单字段
func xyPickVals(in map[string]interface{}, fields map[string]string) map[string]interface{} {
	vals := map[string]interface{}{}
	for k, typ := range fields {
		v, ok := in[k]
		if !ok {
			continue
		}
		if cv, ok2 := xyConvVal(v, typ); ok2 {
			vals[k] = cv
		}
	}
	return vals
}

// AdminXyDataCreate 新增游戏数据
func (h *AdminHandler) AdminXyDataCreate(c *gin.Context) {
	def, ok := xyTableDefs[c.Param("table")]
	if !ok {
		resp.ParamError(c, "未知数据表")
		return
	}
	var in map[string]interface{}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	vals := xyPickVals(in, def.Fields)
	if len(vals) == 0 {
		resp.ParamError(c, "无可写入字段")
		return
	}
	if err := h.DB.Model(def.Model).Create(vals).Error; err != nil {
		resp.ParamError(c, "新增失败："+err.Error())
		return
	}
	resp.OK(c, gin.H{"msg": "新增成功"})
}

// AdminXyDataUpdate 修改游戏数据
func (h *AdminHandler) AdminXyDataUpdate(c *gin.Context) {
	def, ok := xyTableDefs[c.Param("table")]
	if !ok {
		resp.ParamError(c, "未知数据表")
		return
	}
	id, _ := strconv.Atoi(c.Param("id"))
	var in map[string]interface{}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	vals := xyPickVals(in, def.Fields)
	if len(vals) == 0 {
		resp.ParamError(c, "无可修改字段")
		return
	}
	if err := h.DB.Model(def.Model).Where("id = ?", id).Updates(vals).Error; err != nil {
		resp.ParamError(c, "修改失败："+err.Error())
		return
	}
	resp.OK(c, gin.H{"msg": "修改成功"})
}

// AdminXyDataDelete 删除游戏数据
func (h *AdminHandler) AdminXyDataDelete(c *gin.Context) {
	def, ok := xyTableDefs[c.Param("table")]
	if !ok {
		resp.ParamError(c, "未知数据表")
		return
	}
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.DB.Delete(def.Model, id).Error; err != nil {
		resp.ParamError(c, "删除失败："+err.Error())
		return
	}
	resp.OK(c, gin.H{"msg": "删除成功"})
}

// AdminXyBattles 战斗流水
func (h *AdminHandler) AdminXyBattles(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	word := strings.TrimSpace(c.Query("word"))
	q := h.DB.Model(&model.HxxyBattleLog{})
	if word != "" {
		if pid, err := strconv.Atoi(word); err == nil {
			q = q.Where("player_id = ?", pid)
		} else {
			q = q.Where("enemy_name LIKE ?", "%"+word+"%")
		}
	}
	var total int64
	q.Count(&total)
	var rows []model.HxxyBattleLog
	q.Order("id DESC").Offset(offset).Limit(size).Find(&rows)
	// 补玩家名
	type rowOut struct {
		model.HxxyBattleLog
		PlayerName string `json:"player_name"`
	}
	out := []rowOut{}
	for _, r := range rows {
		out = append(out, rowOut{HxxyBattleLog: r, PlayerName: h.hxAdminXyName(r.PlayerID)})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// AdminXyBattleDelete 删除战斗流水
func (h *AdminHandler) AdminXyBattleDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.HxxyBattleLog{}, id)
	resp.OK(c, gin.H{"msg": "已删除"})
}

// AdminXyWallet 货币流水
func (h *AdminHandler) AdminXyWallet(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	word := strings.TrimSpace(c.Query("word"))
	q := h.DB.Model(&model.HxxyWalletLog{})
	if word != "" {
		if pid, err := strconv.Atoi(word); err == nil {
			q = q.Where("player_id = ?", pid)
		} else {
			q = q.Where("reason LIKE ?", "%"+word+"%")
		}
	}
	var total int64
	q.Count(&total)
	var rows []model.HxxyWalletLog
	q.Order("id DESC").Offset(offset).Limit(size).Find(&rows)
	type rowOut struct {
		model.HxxyWalletLog
		PlayerName string `json:"player_name"`
	}
	out := []rowOut{}
	for _, r := range rows {
		out = append(out, rowOut{HxxyWalletLog: r, PlayerName: h.hxAdminXyName(r.PlayerID)})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

func (h *AdminHandler) hxAdminXyName(id uint) string {
	var n string
	h.DB.Model(&model.HxxyPlayer{}).Select("name").Where("id = ?", id).Scan(&n)
	return n
}

// AdminXyStats 游戏统计（对齐原版 GM【查看游戏统计】）
func (h *AdminHandler) AdminXyStats(c *gin.Context) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	var players, todayPlayers, battles, todayBattles, gangs, pets, bags int64
	h.DB.Model(&model.HxxyPlayer{}).Count(&players)
	h.DB.Model(&model.HxxyPlayer{}).Where("created_at >= ?", today).Count(&todayPlayers)
	h.DB.Model(&model.HxxyBattleLog{}).Count(&battles)
	h.DB.Model(&model.HxxyBattleLog{}).Where("created_at >= ?", today).Count(&todayBattles)
	h.DB.Model(&model.HxxyGang{}).Count(&gangs)
	h.DB.Model(&model.HxxyPet{}).Count(&pets)
	h.DB.Model(&model.HxxyBag{}).Count(&bags)
	type agg struct{ Sum int64 }
	var money, bank, beans agg
	h.DB.Model(&model.HxxyPlayer{}).Select("COALESCE(SUM(money),0) as sum").Scan(&money)
	h.DB.Model(&model.HxxyPlayer{}).Select("COALESCE(SUM(bank),0) as sum").Scan(&bank)
	h.DB.Model(&model.HxxyPlayer{}).Select("COALESCE(SUM(beans),0) as sum").Scan(&beans)
	// 等级分布 TOP
	type lvRow struct {
		Level int   `json:"level"`
		Cnt   int64 `json:"cnt"`
	}
	lvs := []lvRow{}
	h.DB.Model(&model.HxxyPlayer{}).Select("level, COUNT(*) as cnt").Group("level").Order("level DESC").Limit(10).Scan(&lvs)
	resp.OK(c, gin.H{
		"players": players, "today_players": todayPlayers,
		"battles": battles, "today_battles": todayBattles,
		"gangs": gangs, "pets": pets, "bags": bags,
		"money": money.Sum, "bank": bank.Sum, "beans": beans.Sum,
		"levels": lvs,
	})
}

// AdminXyAnnounce 发布系统消息（游戏聊天频道以「系统」名义发言）
func (h *AdminHandler) AdminXyAnnounce(c *gin.Context) {
	var in struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || strings.TrimSpace(in.Content) == "" {
		resp.ParamError(c, "请输入公告内容")
		return
	}
	content := trimStr(strings.TrimSpace(in.Content), 200)
	h.DB.Create(&model.HxxyChat{PlayerID: 0, Name: "系统", Content: content})
	// 同时推送到每位玩家首页消息区
	var players []model.HxxyPlayer
	h.DB.Find(&players)
	for _, p := range players {
		h.DB.Create(&model.HxxyMsg{PlayerID: p.ID, FromName: "系统", Kind: "sys", Content: "【公告】" + content})
	}
	resp.OK(c, gin.H{"msg": "系统消息已发布"})
}

// AdminXyGrantAll 全服发放（银两/金豆/物品）
func (h *AdminHandler) AdminXyGrantAll(c *gin.Context) {
	var in struct {
		Kind    string `json:"kind"` // money/beans/item
		Amount  int64  `json:"amount"`
		ItemID  uint   `json:"item_id"`
		Count   int    `json:"count"`
		Reason  string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if in.Reason == "" {
		in.Reason = "全服奖励"
	}
	var players []model.HxxyPlayer
	h.DB.Find(&players)
	if len(players) == 0 {
		resp.ParamError(c, "暂无玩家")
		return
	}
	hxg := &HxxyHandler{DB: h.DB}
	n := 0
	notify := func(playerID uint, content string) {
		h.DB.Create(&model.HxxyMsg{PlayerID: playerID, FromName: "系统", Kind: "sys", Content: content})
	}
	switch in.Kind {
	case "money", "beans":
		if in.Amount <= 0 {
			resp.ParamError(c, "发放数量必须大于0")
			return
		}
		cur := in.Kind
		curName := "银两"
		if cur == "beans" {
			curName = "金豆"
		}
		for _, p := range players {
			hxg.hxWallet(&p, cur, in.Amount, in.Reason)
			notify(p.ID, fmt.Sprintf("【全服奖励】%s：%s +%d。", in.Reason, curName, in.Amount))
			n++
		}
	case "item":
		var it model.HxxyItem
		if err := h.DB.First(&it, in.ItemID).Error; err != nil {
			resp.ParamError(c, "物品不存在")
			return
		}
		if in.Count <= 0 {
			in.Count = 1
		}
		for _, p := range players {
			hxg.hxBagAdd(&p, "item", it.ID, in.Count, it.Bind)
			notify(p.ID, fmt.Sprintf("【全服奖励】%s：获得【%s】×%d。", in.Reason, it.Name, in.Count))
			n++
		}
	default:
		resp.ParamError(c, "未知发放类型")
		return
	}
	resp.OK(c, gin.H{"msg": fmt.Sprintf("已向全服 %d 名玩家发放", n)})
}

// xyBanDur 时长选项（分钟）→ 截止时间戳；-1=永久
func xyBanDur(mins int64) int64 {
	if mins < 0 {
		return 4102444800 // 2100-01-01 永久
	}
	return time.Now().Unix() + mins*60
}

// AdminXyBan 封号/解封（mins=-1永久，0解除）
func (h *AdminHandler) AdminXyBan(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var p model.HxxyPlayer
	if err := h.DB.First(&p, id).Error; err != nil {
		resp.NotFound(c, "玩家不存在")
		return
	}
	var in struct {
		Mins int64 `json:"mins"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	until := xyBanDur(in.Mins)
	if in.Mins == 0 {
		until = 0
	}
	h.DB.Model(&model.HxxyPlayer{}).Where("id = ?", p.ID).Update("ban_until", until)
	if in.Mins == 0 {
		resp.OK(c, gin.H{"msg": "已解除【" + p.Name + "】的封号"})
	} else {
		resp.OK(c, gin.H{"msg": "已封号【" + p.Name + "】"})
	}
}

// AdminXyMute 禁言/解除（mins=-1永久，0解除）
func (h *AdminHandler) AdminXyMute(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var p model.HxxyPlayer
	if err := h.DB.First(&p, id).Error; err != nil {
		resp.NotFound(c, "玩家不存在")
		return
	}
	var in struct {
		Mins int64 `json:"mins"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	until := xyBanDur(in.Mins)
	if in.Mins == 0 {
		until = 0
	}
	h.DB.Model(&model.HxxyPlayer{}).Where("id = ?", p.ID).Update("mute_until", until)
	if in.Mins == 0 {
		resp.OK(c, gin.H{"msg": "已解除【" + p.Name + "】的禁言"})
	} else {
		resp.OK(c, gin.H{"msg": "已禁言【" + p.Name + "】"})
	}
}

// xySettingSet 写 settings 键值（存在则更新，否则创建）
func xySettingSet(db *gorm.DB, key, val string) {
	var cnt int64
	db.Model(&model.Setting{}).Where("`key` = ?", key).Count(&cnt)
	if cnt > 0 {
		db.Model(&model.Setting{}).Where("`key` = ?", key).Update("value", val)
	} else {
		db.Create(&model.Setting{Key: key, Value: val})
	}
}

// AdminXyServer 服务器维护状态
func (h *AdminHandler) AdminXyServer(c *gin.Context) {
	var on, notice string
	h.DB.Model(&model.Setting{}).Select("`value`").Where("`key` = 'hxxy_maintenance'").Scan(&on)
	h.DB.Model(&model.Setting{}).Select("`value`").Where("`key` = 'hxxy_maintenance_notice'").Scan(&notice)
	var players int64
	h.DB.Model(&model.HxxyPlayer{}).Count(&players)
	resp.OK(c, gin.H{"maintenance": on == "1", "notice": notice, "players": players})
}

// AdminXyServerSet 设置维护模式（on=true 维护中，玩家进游戏看到公告；false 开放）
func (h *AdminHandler) AdminXyServerSet(c *gin.Context) {
	var in struct {
		On     bool   `json:"on"`
		Notice string `json:"notice"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	val := "0"
	if in.On {
		val = "1"
	}
	xySettingSet(h.DB, "hxxy_maintenance", val)
	xySettingSet(h.DB, "hxxy_maintenance_notice", strings.TrimSpace(in.Notice))
	if in.On {
		resp.OK(c, gin.H{"msg": "服务器已进入维护模式，玩家将无法进入游戏"})
	} else {
		resp.OK(c, gin.H{"msg": "服务器已开放，玩家可正常进入"})
	}
}
