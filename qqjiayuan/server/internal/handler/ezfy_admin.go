package handler

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 二战风云管理端（玩家/数据/流水/系统），复刻 stzb-fk GM 能力，表均带 ezfy_ 前缀

// AdminEzfyPlayers 玩家列表（word=昵称/家园号/用户ID 模糊）
func (h *AdminHandler) AdminEzfyPlayers(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	word := strings.TrimSpace(c.Query("word"))
	q := h.DB.Model(&model.EzfyProfile{})
	if word != "" {
		var wu model.User
		h.DB.Select("id").Where("username = ? OR nickname = ?", word, word).First(&wu)
		if uid, err := strconv.Atoi(word); err == nil {
			q = q.Where("user_id = ?", uid)
			if wu.ID > 0 {
				q = q.Or("user_id = ?", wu.ID)
			}
		} else {
			q = q.Where("nickname LIKE ?", "%"+word+"%")
			if wu.ID > 0 {
				q = q.Or("user_id = ?", wu.ID)
			}
		}
	}
	var total int64
	q.Count(&total)
	var rows []model.EzfyProfile
	q.Order("prestige DESC, id ASC").Offset(offset).Limit(size).Find(&rows)
	type rowOut struct {
		model.EzfyProfile
		HomeNick  string `json:"home_nick"`
		HomeNum   string `json:"home_num"`
		CityCount int64  `json:"city_count"`
		CampName  string `json:"camp_name"`
		RankName  string `json:"rank_name"`
	}
	out := []rowOut{}
	for _, p := range rows {
		var u model.User
		h.DB.First(&u, p.UserID)
		var cities int64
		h.DB.Model(&model.EzfyCity{}).Where("user_id = ?", p.UserID).Count(&cities)
		out = append(out, rowOut{EzfyProfile: p, HomeNick: u.Nickname, HomeNum: u.Username,
			CityCount: cities, CampName: ezfyCampName(p.Camp), RankName: ezfyRankName(p.Prestige)})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

func ezfyCampName(camp int) string {
	if camp == 2 {
		return "轴心国"
	}
	return "同盟国"
}

// AdminEzfyPlayerDetail 玩家详情（档案+城池+背包+军团+最近出征）
func (h *AdminHandler) AdminEzfyPlayerDetail(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var p model.EzfyProfile
	if err := h.DB.Where("user_id = ?", id).First(&p).Error; err != nil {
		resp.NotFound(c, "玩家不存在")
		return
	}
	var cities []model.EzfyCity
	h.DB.Where("user_id = ?", p.UserID).Find(&cities)
	var bag []model.EzfyItem
	h.DB.Where("user_id = ?", p.UserID).Find(&bag)
	// 补道具名称
	type bagOut struct {
		model.EzfyItem
		ItemName string `json:"item_name"`
	}
	bagViews := []bagOut{}
	for _, it := range bag {
		var cfg model.EzfyCfgItem
		name := ""
		if err := h.DB.First(&cfg, it.CfgId).Error; err == nil {
			name = cfg.Name
		}
		bagViews = append(bagViews, bagOut{EzfyItem: it, ItemName: name})
	}
	var members []model.EzfyCorpsMember
	h.DB.Where("user_id = ?", p.UserID).Find(&members)
	type corpsOut struct {
		model.EzfyCorpsMember
		CorpsName string `json:"corps_name"`
	}
	corpsViews := []corpsOut{}
	for _, m := range members {
		var cp model.EzfyCorps
		name := ""
		if err := h.DB.First(&cp, m.CorpsId).Error; err == nil {
			name = cp.Name
		}
		corpsViews = append(corpsViews, corpsOut{EzfyCorpsMember: m, CorpsName: name})
	}
	var orders []model.EzfyOrder
	h.DB.Where("user_id = ?", p.UserID).Order("id DESC").Limit(20).Find(&orders)
	var u model.User
	homeNick, homeNum := "", ""
	if err := h.DB.First(&u, p.UserID).Error; err == nil {
		homeNick, homeNum = u.Nickname, u.Username
	}
	resp.OK(c, gin.H{"player": p, "home_nick": homeNick, "home_num": homeNum,
		"cities": cities, "bag": bagViews, "corps": corpsViews, "orders": orders,
		"camp_name": ezfyCampName(p.Camp), "rank_name": ezfyRankName(p.Prestige)})
}

// AdminEzfyPlayerUpdate 修改玩家（昵称/声望/阵营）
func (h *AdminHandler) AdminEzfyPlayerUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var p model.EzfyProfile
	if err := h.DB.Where("user_id = ?", id).First(&p).Error; err != nil {
		resp.NotFound(c, "玩家不存在")
		return
	}
	var in struct {
		Nickname *string `json:"nickname"`
		Prestige *int    `json:"prestige"`
		Camp     *int    `json:"camp"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	updates := map[string]interface{}{}
	if in.Nickname != nil && strings.TrimSpace(*in.Nickname) != "" {
		updates["nickname"] = trimStr(strings.TrimSpace(*in.Nickname), 20)
	}
	if in.Prestige != nil && *in.Prestige >= 0 {
		updates["prestige"] = *in.Prestige
	}
	if in.Camp != nil && (*in.Camp == 1 || *in.Camp == 2) {
		updates["camp"] = *in.Camp
	}
	if len(updates) > 0 {
		h.DB.Model(&model.EzfyProfile{}).Where("id = ?", p.ID).Updates(updates)
	}
	resp.OK(c, gin.H{"msg": "修改成功"})
}

// AdminEzfyGrant 发放资源/道具（资源入主城并按仓储上限截断，道具入背包）
// AdminEzfyGrantOfficer POST /admin/ezfy-players/:id/grant-officer  {general_id}
// 名将只能由管理端发放(用户要求): 直接把 cfg_general 里的名将变成该玩家的军官
func (h *AdminHandler) AdminEzfyGrantOfficer(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var in struct {
		GeneralID int `json:"general_id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.GeneralID <= 0 {
		resp.ParamError(c, "请选择名将")
		return
	}
	msg, errMsg := h.ezfyGrantGeneral(uint(id), in.GeneralID)
	if errMsg != "" {
		resp.ParamError(c, errMsg)
		return
	}
	resp.OK(c, gin.H{"msg": msg})
}

func (h *AdminHandler) AdminEzfyGrant(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var p model.EzfyProfile
	if err := h.DB.Where("user_id = ?", id).First(&p).Error; err != nil {
		resp.NotFound(c, "玩家不存在")
		return
	}
	var in struct {
		Gold  int64 `json:"gold"`
		Food  int64 `json:"food"`
		Steel int64 `json:"steel"`
		Oil   int64 `json:"oil"`
		Rare  int64 `json:"rare"`
		Items []struct {
			CfgID int `json:"cfg_id"`
			Count int `json:"count"`
		} `json:"items"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	ez := &EzfyHandler{DB: h.DB}
	msg := "已发放"
	if in.Gold != 0 || in.Food != 0 || in.Steel != 0 || in.Oil != 0 || in.Rare != 0 {
		// GM 发放不按仓储上限截断（玩家要多少给多少，可以超上限堆着）
		ez.giveResourcesNoCap(p.UserID, in.Food, in.Steel, in.Oil, in.Rare, in.Gold)
		if in.Gold != 0 {
			msg += fmt.Sprintf(" 黄金%+d", in.Gold)
		}
		if in.Food != 0 {
			msg += fmt.Sprintf(" 粮食%+d", in.Food)
		}
		if in.Steel != 0 {
			msg += fmt.Sprintf(" 钢铁%+d", in.Steel)
		}
		if in.Oil != 0 {
			msg += fmt.Sprintf(" 石油%+d", in.Oil)
		}
		if in.Rare != 0 {
			msg += fmt.Sprintf(" 稀矿%+d", in.Rare)
		}
	}
	for _, it := range in.Items {
		if it.CfgID <= 0 || it.Count <= 0 {
			continue
		}
		var cfg model.EzfyCfgItem
		if err := h.DB.First(&cfg, it.CfgID).Error; err != nil {
			resp.ParamError(c, "道具不存在："+strconv.Itoa(it.CfgID))
			return
		}
		ez.addItem(p.UserID, it.CfgID, it.Count)
		msg += fmt.Sprintf(" 【%s】×%d", cfg.Name, it.Count)
	}
	// 站内通知
	h.DB.Create(&model.EzfyNotice{UserId: p.UserID, Title: "管理员发放",
		Content: strings.TrimSpace(msg) + "，请查收。"})
	resp.OK(c, gin.H{"msg": msg})
}

// AdminEzfyPlayerDelete 删除玩家（档案+城池+全部游戏数据）
func (h *AdminHandler) AdminEzfyPlayerDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var p model.EzfyProfile
	if err := h.DB.Where("user_id = ?", id).First(&p).Error; err != nil {
		resp.NotFound(c, "玩家不存在")
		return
	}
	uid := p.UserID
	var cityIds []int64
	h.DB.Model(&model.EzfyCity{}).Where("user_id = ?", uid).Pluck("id", &cityIds)
	if len(cityIds) > 0 {
		h.DB.Where("city_id IN ?", cityIds).Delete(&model.EzfyCityBuilding{})
		h.DB.Where("city_id IN ?", cityIds).Delete(&model.EzfyCityTroop{})
		h.DB.Where("city_id IN ?", cityIds).Delete(&model.EzfyCityTech{})
		h.DB.Where("city_id IN ?", cityIds).Delete(&model.EzfyTrainQueue{})
		h.DB.Where("city_id IN ?", cityIds).Delete(&model.EzfyWildland{})
		h.DB.Where("city_id IN ?", cityIds).Delete(&model.EzfyWounded{})
		h.DB.Where("city_id IN ?", cityIds).Delete(&model.EzfyCityEffect{})
		h.DB.Where("city_id IN ?", cityIds).Delete(&model.EzfyCityTarget{})
	}
	h.DB.Where("user_id = ?", uid).Delete(&model.EzfyCity{})
	h.DB.Where("user_id = ?", uid).Delete(&model.EzfyOrder{})
	h.DB.Where("user_id = ?", uid).Delete(&model.EzfyReport{})
	h.DB.Where("user_id = ?", uid).Delete(&model.EzfyItem{})
	h.DB.Where("user_id = ?", uid).Delete(&model.EzfySign{})
	h.DB.Where("user_id = ?", uid).Delete(&model.EzfyGift{})
	h.DB.Where("user_id = ?", uid).Delete(&model.EzfyTask{})
	h.DB.Where("user_id = ?", uid).Delete(&model.EzfyNotice{})
	h.DB.Where("user_id = ?", uid).Delete(&model.EzfyCorpsMember{})
	h.DB.Where("user_id = ?", uid).Delete(&model.EzfyProfile{})
	resp.OK(c, gin.H{"msg": "玩家及全部游戏数据已删除"})
}

// ============ 数据管理（配置表 + 城池，复用 xy 数据管理白名单模式） ============

// ezfyTableDef 数据表定义（Fields 为可编辑字段白名单：字段名(json) -> 类型）
type ezfyTableDef struct {
	Model  interface{}
	Fields map[string]string
}

var ezfyTableDefs = map[string]ezfyTableDef{
	"activities": {&model.EzfyActivity{}, map[string]string{
		"name": "string", "type": "int", "param": "int",
		"start_time": "int64", "end_time": "int64", "status": "int", "des": "string",
	}},
	"items": {&model.EzfyCfgItem{}, map[string]string{
		"name": "string", "item_type": "int", "param1": "int64",
		"price_gold": "int64", "icon": "string", "description": "string",
	}},
	"taskTypes": {&model.EzfyCfgTaskType{}, map[string]string{
		"name": "string", "code": "string", "reset_type": "int", "sort_no": "int", "status": "int",
	}},
	"tasks": {&model.EzfyCfgTask{}, map[string]string{
		"name": "string", "task_type": "string", "target": "int", "reward_gold": "int64",
		"reward_food": "int64", "reward_steel": "int64", "reward_oil": "int64",
		"reward_rare": "int64", "reward_prestige": "int", "sort_no": "int", "type_id": "int", "status": "int",
	}},
}

// ezfyDataMoved 已经从「数据管理」迁到专属模块的表 → 提示去哪改
//
// ★ 之前「数据管理」把建筑/兵种/科技/野地/城池也放进来了，与
//   建筑管理 / 兵种管理 / 科技管理 / 地图管理 / 城市管理 完全重复，
//   同一个字段两处能改、种子策略还不一样，容易改出不一致。
//   现在数据管理只保留「没有专属模块」的零散配置表。
var ezfyDataMoved = map[string]string{
	"buildings":      "「建筑管理 → 总建筑配置」",
	"buildingLevels": "「建筑管理 → 总建筑配置 → 等级配置」",
	"troops":         "「兵种管理 → 兵种配置」",
	"techs":          "「科技管理 → 科技配置」",
	"techLevels":     "「科技管理 → 科技配置 → 等级配置」",
	"wildlands":      "「地图管理 → 野地类型」",
	"cities":         "「城市管理」",
}

func (h *AdminHandler) ezfyTableOf(c *gin.Context) (ezfyTableDef, bool) {
	name := c.Param("table")
	def, ok := ezfyTableDefs[name]
	if !ok {
		if where, moved := ezfyDataMoved[name]; moved {
			resp.ParamError(c, "该表已迁到 "+where+" 维护，请到对应模块操作（避免两处重复配置）")
			return def, false
		}
		resp.ParamError(c, "未知数据表")
	}
	return def, ok
}

// AdminEzfyData 数据分页查询（table=buildings/buildingLevels/troops/techs/techLevels/wildlands/items/taskTypes/tasks/cities）
func (h *AdminHandler) AdminEzfyData(c *gin.Context) {
	def, ok := h.ezfyTableOf(c)
	if !ok {
		return
	}
	page, offset, size := pageOf(c, 10)
	word := strings.TrimSpace(c.Query("word"))
	var total int64
	q := h.DB.Model(def.Model)
	lq := h.DB.Model(def.Model)
	if word != "" {
		if id, err := strconv.Atoi(word); err == nil {
			q = q.Where("id = ?", id)
			lq = lq.Where("id = ?", id)
		} else {
			q = q.Where("name LIKE ?", "%"+word+"%")
			lq = lq.Where("name LIKE ?", "%"+word+"%")
		}
	}
	q.Count(&total)
	var rows []map[string]interface{}
	lq.Order("id").Offset(offset).Limit(size).Find(&rows)
	resp.OK(c, gin.H{"list": rows, "total": total, "page": page, "size": size})
}

// AdminEzfyDataCreate 新增数据
func (h *AdminHandler) AdminEzfyDataCreate(c *gin.Context) {
	def, ok := h.ezfyTableOf(c)
	if !ok {
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

// AdminEzfyDataUpdate 修改数据
func (h *AdminHandler) AdminEzfyDataUpdate(c *gin.Context) {
	def, ok := h.ezfyTableOf(c)
	if !ok {
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

// AdminEzfyDataDelete 删除数据
func (h *AdminHandler) AdminEzfyDataDelete(c *gin.Context) {
	def, ok := h.ezfyTableOf(c)
	if !ok {
		return
	}
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.DB.Delete(def.Model, id).Error; err != nil {
		resp.ParamError(c, "删除失败："+err.Error())
		return
	}
	resp.OK(c, gin.H{"msg": "删除成功"})
}

// ============ 流水管理（出征/世界聊天/交易所） ============

// AdminEzfyOrders 出征订单列表（word=用户ID/玩家昵称，type=出征类型，status=状态）
func (h *AdminHandler) AdminEzfyOrders(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	word := strings.TrimSpace(c.Query("word"))
	orderType := atoiOr(c.Query("type"), 0)
	status := atoiOr(c.Query("status"), -1)
	q := h.DB.Model(&model.EzfyOrder{})
	if word != "" {
		if uid, err := strconv.Atoi(word); err == nil {
			q = q.Where("user_id = ?", uid)
		} else {
			var ids []uint
			h.DB.Model(&model.EzfyProfile{}).Select("user_id").
				Where("nickname LIKE ?", "%"+word+"%").Scan(&ids)
			if len(ids) > 0 {
				q = q.Where("user_id IN ?", ids)
			} else {
				q = q.Where("1 = 0")
			}
		}
	}
	if orderType > 0 {
		q = q.Where("order_type = ?", orderType)
	}
	if status >= 0 {
		q = q.Where("status = ?", status)
	}
	var total int64
	q.Count(&total)
	var rows []model.EzfyOrder
	q.Order("id DESC").Offset(offset).Limit(size).Find(&rows)
	type rowOut struct {
		model.EzfyOrder
		PlayerName string `json:"player_name"`
		HomeNum    string `json:"home_num"`
		TypeName   string `json:"type_name"`
	}
	out := []rowOut{}
	for _, o := range rows {
		pn, hn := h.ezfyAdminName(o.UserID)
		out = append(out, rowOut{EzfyOrder: o, PlayerName: pn, HomeNum: hn,
			TypeName: ezfyOrderTypeName(o.OrderType)})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// AdminEzfyOrderDelete 删除出征订单
func (h *AdminHandler) AdminEzfyOrderDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.EzfyOrder{}, id)
	resp.OK(c, gin.H{"msg": "已删除"})
}

// AdminEzfyChats 世界聊天列表
func (h *AdminHandler) AdminEzfyChats(c *gin.Context) {
	page, offset, size := pageOf(c, 20)
	word := strings.TrimSpace(c.Query("word"))
	q := h.DB.Model(&model.EzfyChat{})
	if word != "" {
		q = q.Where("user_name LIKE ? OR content LIKE ?", "%"+word+"%", "%"+word+"%")
	}
	var total int64
	q.Count(&total)
	var rows []model.EzfyChat
	q.Order("id DESC").Offset(offset).Limit(size).Find(&rows)
	resp.OK(c, gin.H{"list": rows, "total": total, "page": page, "size": size})
}

// AdminEzfyChatDelete 删除世界聊天
func (h *AdminHandler) AdminEzfyChatDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.EzfyChat{}, id)
	resp.OK(c, gin.H{"msg": "已删除"})
}

// AdminEzfyExchanges 交易所挂单列表
func (h *AdminHandler) AdminEzfyExchanges(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	word := strings.TrimSpace(c.Query("word"))
	status := atoiOr(c.Query("status"), -1)
	q := h.DB.Model(&model.EzfyExchange{})
	if word != "" {
		q = q.Where("seller_name LIKE ?", "%"+word+"%")
	}
	if status >= 0 {
		q = q.Where("status = ?", status)
	}
	var total int64
	q.Count(&total)
	var rows []model.EzfyExchange
	q.Order("id DESC").Offset(offset).Limit(size).Find(&rows)
	type rowOut struct {
		model.EzfyExchange
		TypeName   string `json:"type_name"`
		StatusName string `json:"status_name"`
	}
	out := []rowOut{}
	statusNames := map[int]string{0: "在售", 1: "成交", 2: "下架"}
	for _, e := range rows {
		out = append(out, rowOut{EzfyExchange: e, TypeName: ezfyResNames[e.EsType],
			StatusName: statusNames[e.Status]})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// AdminEzfyExchangeDelete 删除交易所挂单
func (h *AdminHandler) AdminEzfyExchangeDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.EzfyExchange{}, id)
	resp.OK(c, gin.H{"msg": "已删除"})
}

func (h *AdminHandler) ezfyAdminName(uid uint) (string, string) {
	var p model.EzfyProfile
	nick := ""
	if err := h.DB.Where("user_id = ?", uid).First(&p).Error; err == nil {
		nick = p.Nickname
	}
	var u model.User
	num := ""
	if err := h.DB.First(&u, uid).Error; err == nil {
		num = u.Username
	}
	return nick, num
}

// atoiOr 解析整数，空串/非法时返回默认值
func atoiOr(s string, def int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

// ============ 系统管理（军团/公告/统计/维护） ============

// AdminEzfyCorps 军团列表
func (h *AdminHandler) AdminEzfyCorps(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	word := strings.TrimSpace(c.Query("word"))
	q := h.DB.Model(&model.EzfyCorps{})
	if word != "" {
		q = q.Where("name LIKE ?", "%"+word+"%")
	}
	var total int64
	q.Count(&total)
	var rows []model.EzfyCorps
	q.Order("member_count DESC, id ASC").Offset(offset).Limit(size).Find(&rows)
	type rowOut struct {
		model.EzfyCorps
		LeaderName string `json:"leader_name"`
	}
	out := []rowOut{}
	for _, r := range rows {
		ln, _ := h.ezfyAdminName(r.LeaderUserId)
		out = append(out, rowOut{EzfyCorps: r, LeaderName: ln})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// AdminEzfyCorpsDelete 解散军团（清除成员与军团聊天）
func (h *AdminHandler) AdminEzfyCorpsDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var cp model.EzfyCorps
	if err := h.DB.First(&cp, id).Error; err != nil {
		resp.NotFound(c, "军团不存在")
		return
	}
	h.DB.Where("corps_id = ?", id).Delete(&model.EzfyCorpsMember{})
	h.DB.Where("corps_id = ?", id).Delete(&model.EzfyCorpsChat{})
	h.DB.Delete(&model.EzfyCorps{}, id)
	resp.OK(c, gin.H{"msg": "已解散军团「" + cp.Name + "」"})
}

// AdminEzfyNotices 公告列表（user_id=0 全员公告）
func (h *AdminHandler) AdminEzfyNotices(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	q := h.DB.Model(&model.EzfyNotice{}).Where("user_id = 0")
	var total int64
	q.Count(&total)
	var rows []model.EzfyNotice
	q.Order("is_top DESC, id DESC").Offset(offset).Limit(size).Find(&rows)
	resp.OK(c, gin.H{"list": rows, "total": total, "page": page, "size": size})
}

// AdminEzfyAnnounce 发布游戏公告（全员可见，玩家游戏内公告栏展示）
func (h *AdminHandler) AdminEzfyAnnounce(c *gin.Context) {
	var in struct {
		Title   string `json:"title"`
		Content string `json:"content"`
		IsTop   int    `json:"is_top"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || strings.TrimSpace(in.Content) == "" {
		resp.ParamError(c, "请输入公告内容")
		return
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = "系统公告"
	}
	h.DB.Create(&model.EzfyNotice{UserId: 0, Title: trimStr(title, 100),
		Content: trimStr(strings.TrimSpace(in.Content), 2000), IsTop: in.IsTop})
	// 同步到世界聊天频道
	h.DB.Create(&model.EzfyChat{UserId: 0, UserName: "系统", Content: trimStr("【公告】"+in.Content, 200)})
	resp.OK(c, gin.H{"msg": "公告已发布"})
}

// AdminEzfyNoticeDelete 删除公告
func (h *AdminHandler) AdminEzfyNoticeDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Where("user_id = 0").Delete(&model.EzfyNotice{}, id)
	resp.OK(c, gin.H{"msg": "已删除"})
}

// AdminEzfyStats 游戏统计
func (h *AdminHandler) AdminEzfyStats(c *gin.Context) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	var players, todayPlayers, cities, orders, todayOrders, corps, exchanges int64
	h.DB.Model(&model.EzfyProfile{}).Count(&players)
	// EzfyProfile 无 created_at，用当日新建城池数代替「今日新增」
	h.DB.Model(&model.EzfyCity{}).Where("created_at >= ?", today).Count(&todayPlayers)
	h.DB.Model(&model.EzfyCity{}).Count(&cities)
	h.DB.Model(&model.EzfyOrder{}).Count(&orders)
	h.DB.Model(&model.EzfyOrder{}).Where("created_at >= ?", today).Count(&todayOrders)
	h.DB.Model(&model.EzfyCorps{}).Count(&corps)
	h.DB.Model(&model.EzfyExchange{}).Where("status = 0").Count(&exchanges)
	type agg struct{ Sum int64 }
	var gold, food, steel, oil, rare, prestige agg
	h.DB.Model(&model.EzfyCity{}).Select("COALESCE(SUM(gold),0) as sum").Scan(&gold)
	h.DB.Model(&model.EzfyCity{}).Select("COALESCE(SUM(food),0) as sum").Scan(&food)
	h.DB.Model(&model.EzfyCity{}).Select("COALESCE(SUM(steel),0) as sum").Scan(&steel)
	h.DB.Model(&model.EzfyCity{}).Select("COALESCE(SUM(oil),0) as sum").Scan(&oil)
	h.DB.Model(&model.EzfyCity{}).Select("COALESCE(SUM(rare),0) as sum").Scan(&rare)
	h.DB.Model(&model.EzfyProfile{}).Select("COALESCE(SUM(prestige),0) as sum").Scan(&prestige)
	// 阵营分布
	type campRow struct {
		Camp int   `json:"camp"`
		Cnt  int64 `json:"cnt"`
	}
	camps := []campRow{}
	h.DB.Model(&model.EzfyProfile{}).Select("camp, COUNT(*) as cnt").Group("camp").Scan(&camps)
	// 声望排行 TOP10
	type rankRow struct {
		Nickname string `json:"nickname"`
		Prestige int    `json:"prestige"`
		Camp     int    `json:"camp"`
	}
	tops := []rankRow{}
	h.DB.Model(&model.EzfyProfile{}).Select("nickname, prestige, camp").
		Order("prestige DESC").Limit(10).Scan(&tops)
	resp.OK(c, gin.H{
		"players": players, "today_players": todayPlayers,
		"cities": cities, "orders": orders, "today_orders": todayOrders,
		"corps": corps, "exchanges": exchanges,
		"gold": gold.Sum, "food": food.Sum, "steel": steel.Sum, "oil": oil.Sum,
		"rare": rare.Sum, "prestige": prestige.Sum,
		"camps": camps, "tops": tops,
	})
}

// AdminEzfyServer 服务器维护状态
func (h *AdminHandler) AdminEzfyServer(c *gin.Context) {
	var on, notice string
	h.DB.Model(&model.Setting{}).Select("`value`").Where("`key` = 'ezfy_maintenance'").Scan(&on)
	h.DB.Model(&model.Setting{}).Select("`value`").Where("`key` = 'ezfy_maintenance_notice'").Scan(&notice)
	var players, cities int64
	h.DB.Model(&model.EzfyProfile{}).Count(&players)
	h.DB.Model(&model.EzfyCity{}).Count(&cities)
	resp.OK(c, gin.H{"maintenance": on == "1", "notice": notice, "players": players, "cities": cities})
}

// AdminEzfyServerSet 设置维护模式（on=true 维护中，游戏接口统一拦截）
func (h *AdminHandler) AdminEzfyServerSet(c *gin.Context) {
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
	xySettingSet(h.DB, "ezfy_maintenance", val)
	xySettingSet(h.DB, "ezfy_maintenance_notice", strings.TrimSpace(in.Notice))
	if in.On {
		resp.OK(c, gin.H{"msg": "二战风云已进入维护模式，玩家将无法进行游戏操作"})
	} else {
		resp.OK(c, gin.H{"msg": "二战风云已开放，玩家可正常游戏"})
	}
}

// EzfyMaintGate 服务器维护拦截（维护中所有游戏接口统一返回维护公告）
func (h *EzfyHandler) EzfyMaintGate() gin.HandlerFunc {
	return func(c *gin.Context) {
		var on, notice string
		h.DB.Model(&model.Setting{}).Select("`value`").Where("`key` = 'ezfy_maintenance'").Scan(&on)
		if on == "1" {
			h.DB.Model(&model.Setting{}).Select("`value`").Where("`key` = 'ezfy_maintenance_notice'").Scan(&notice)
			if strings.TrimSpace(notice) == "" {
				notice = "服务器维护中，请稍后再来"
			}
			resp.ParamError(c, "【服务器维护中】"+notice)
			c.Abort()
			return
		}
		c.Next()
	}
}
