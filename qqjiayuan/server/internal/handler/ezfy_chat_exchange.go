package handler

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 二战风云 世界聊天 + 交易所 + 被占城市管理

// ============ 聊天频道（复刻原版 chatB?type=：1公共 2军团 4系统）============

const (
	ezfyChanPublic  = 1 // 公共频道
	ezfyChanCorps   = 2 // 军团频道
	ezfyChanSystem  = 4 // 系统频道(只读)
	ezfyChatMaxRune = 25
)

// ChatList GET /games/ezfy/chat?channel=1|2|4
func (h *EzfyHandler) ChatList(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	channel := ezfyChanPublic
	if v, err := strconv.Atoi(c.DefaultQuery("channel", "1")); err == nil && v > 0 {
		channel = v
	}
	// 我的军团(军团频道前提)
	myCorps := h.myCorpsOf(uid)
	// 没有军团时军团频道降级为公共频道
	viewChannel := channel
	if channel == ezfyChanCorps && myCorps == nil {
		viewChannel = ezfyChanPublic
	}

	out := gin.H{
		"channel": viewChannel, "request_channel": channel,
		"has_corps": myCorps != nil, "corps_name": "",
		"corps_players": 0, // ★ 军团人数（没军团就是 0）
		"can_send":      viewChannel == ezfyChanPublic || viewChannel == ezfyChanCorps,
	}
	if myCorps != nil {
		out["corps_name"] = myCorps.Name
		out["corps_players"] = h.corpsMemberCount(int64(myCorps.ID))
	}

	switch viewChannel {
	case ezfyChanCorps:
		var list []model.EzfyCorpsChat
		h.DB.Where("corps_id = ?", myCorps.ID).Order("id DESC").Limit(50).Find(&list)
		ids := []uint{}
		for _, ch := range list {
			if ch.UserId > 0 {
				ids = append(ids, ch.UserId)
			}
		}
		nick := h.liveNicknames(ids)
		views := make([]gin.H, 0, len(list))
		for i := len(list) - 1; i >= 0; i-- {
			ch := list[i]
			name, color := ch.UserName, ""
			if v, ok := nick[ch.UserId]; ok {
				if v[0] != "" {
					name = v[0]
				}
				color = v[1]
			}
			views = append(views, gin.H{"id": ch.ID, "user_id": ch.UserId, "user_name": name, "color": color,
				"content": ch.Content, "created_at": ch.CreatedAt, "talk_type": 1, "mine": ch.UserId == uid})
		}
		out["chats"] = views
	case ezfyChanSystem:
		// 系统频道: 系统公告(全员+个人) + 系统消息(talk_type=0)
		var notices []model.EzfyNotice
		h.DB.Where("user_id = 0 OR user_id = ?", uid).
			Order("is_top DESC, id DESC").Limit(20).Find(&notices)
		nviews := make([]gin.H, 0, len(notices))
		for _, n := range notices {
			nviews = append(nviews, gin.H{"id": n.ID, "title": n.Title, "content": n.Content,
				"is_top": n.IsTop, "created_at": n.CreatedAt})
		}
		var msgs []model.EzfyChat
		h.DB.Where("channel = ? AND talk_type = 0", ezfyChanSystem).Order("id DESC").Limit(50).Find(&msgs)
		mviews := make([]gin.H, 0, len(msgs))
		for i := len(msgs) - 1; i >= 0; i-- {
			m := msgs[i]
			mviews = append(mviews, gin.H{"id": m.ID, "user_name": m.UserName, "content": m.Content,
				"created_at": m.CreatedAt, "talk_type": 0})
		}
		out["notices"] = nviews
		out["chats"] = mviews
	default:
		var chats []model.EzfyChat
		h.DB.Where("channel = ?", ezfyChanPublic).Order("id DESC").Limit(50).Find(&chats)
		// 实时昵称/颜色(玩家改了个性昵称, 历史消息也跟着变)
		ids := []uint{}
		for _, ch := range chats {
			if ch.UserId > 0 {
				ids = append(ids, ch.UserId)
			}
		}
		nick := h.liveNicknames(ids)
		views := make([]gin.H, 0, len(chats))
		for i := len(chats) - 1; i >= 0; i-- {
			ch := chats[i]
			name, color := ch.UserName, ""
			if v, ok := nick[ch.UserId]; ok {
				if v[0] != "" {
					name = v[0]
				}
				color = v[1]
			}
			views = append(views, gin.H{"id": ch.ID, "user_id": ch.UserId, "user_name": name, "color": color,
				"content": ch.Content, "created_at": ch.CreatedAt, "talk_type": ch.TalkType,
				"mine": ch.UserId == uid})
		}
		out["chats"] = views
	}

	// ★ 人数要按频道给：
	//   - 军团频道 → 军团人数（没军团 = 0）
	//   - 世界/系统频道 → 全服玩家数
	//   原来无论哪个频道都返回全服人数，军团频道会显示成「全服人数」，看着就是错的。
	var online int64
	h.DB.Model(&model.EzfyProfile{}).Count(&online)
	if viewChannel == ezfyChanCorps {
		out["players"] = out["corps_players"]
	} else {
		out["players"] = online
	}
	resp.OK(c, out)
}

// ChatSend POST /games/ezfy/chat {channel, content}
func (h *EzfyHandler) ChatSend(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Content string `json:"content"`
		Channel int    `json:"channel"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	channel := req.Channel
	if channel == 0 {
		channel = ezfyChanPublic
	}
	content := trimSpace(req.Content)
	if content == "" {
		resp.ParamError(c, "消息为空")
		return
	}
	if r := []rune(content); len(r) > ezfyChatMaxRune {
		content = string(r[:ezfyChatMaxRune])
	}
	// ★ 第九轮：二战聊天敏感词（独立维护页 ezfy_word_filter），拦截类直接拒绝
	if filtered, blocked := ezfyFilterChat(content); blocked {
		resp.Forbidden(c, "你的发言包含敏感词，请修改后再发")
		return
	} else {
		content = filtered
	}
	profile := h.ensureProfile(uid)
	// 复刻原版聊天: 每次发言 30 秒冷却(前端有倒计时, 服务端同样兜底)
	if left := h.chatCooldownLeft(uid); left > 0 {
		resp.ParamError(c, fmt.Sprintf("发言冷却中, 还需 %d 秒", left))
		return
	}
	switch channel {
	case ezfyChanCorps:
		myCorps := h.myCorpsOf(uid)
		if myCorps == nil {
			resp.ParamError(c, "你还没有加入军团")
			return
		}
		h.DB.Create(&model.EzfyCorpsChat{CorpsId: myCorps.ID, UserId: uid, UserName: profile.Nickname, Content: content})
		resp.OK(c, gin.H{"msg": "军团频道发送成功"})
		return
	case ezfyChanSystem:
		resp.Forbidden(c, "系统频道仅系统可发言")
		return
	}
	h.DB.Create(&model.EzfyChat{UserId: uid, UserName: profile.Nickname,
		Content: content, Channel: ezfyChanPublic, TalkType: 1})
	resp.OK(c, gin.H{"msg": "发送成功"})
}

// ezfyChatCooldownSec 发言冷却秒数(复刻原版聊天 30 秒)
const ezfyChatCooldownSec = 30

// chatCooldownLeft 距下次可发言还剩多少秒(0 表示可以发言)
func (h *EzfyHandler) chatCooldownLeft(uid uint) int64 {
	var last model.EzfyChat
	if err := h.DB.Where("user_id = ? AND talk_type = 1", uid).Order("id DESC").First(&last).Error; err != nil {
		return 0
	}
	elapsed := time.Now().Unix() - last.CreatedAt.Unix()
	if elapsed >= ezfyChatCooldownSec {
		return 0
	}
	return ezfyChatCooldownSec - elapsed
}

// liveNicknames 批量取玩家**当前**的昵称与昵称颜色
// (聊天表里存的是发消息时的快照, 玩家改了个性昵称后要能实时反映)
func (h *EzfyHandler) liveNicknames(ids []uint) map[uint][2]string {
	out := map[uint][2]string{}
	if len(ids) == 0 {
		return out
	}
	var us []model.User
	h.DB.Select("id, nickname, color").Where("id IN ?", ids).Find(&us)
	for _, u := range us {
		out[u.ID] = [2]string{u.Nickname, u.Color}
	}
	return out
}

// myCorpsOf 我所在的军团
func (h *EzfyHandler) myCorpsOf(uid uint) *model.EzfyCorps {
	var mb model.EzfyCorpsMember
	if err := h.DB.Where("user_id = ?", uid).First(&mb).Error; err != nil {
		return nil
	}
	var cp model.EzfyCorps
	if err := h.DB.First(&cp, mb.CorpsId).Error; err != nil {
		return nil
	}
	return &cp
}

// HomeChat GET /games/ezfy/chat/home —— 首页「世界聊天」预览
//
// 汇总四个来源并按时间倒序, 每条带频道标识(用户要求):
//
//	[世界] —— 公共频道的玩家发言
//	[军团] —— 我所在军团的聊天
//	[私聊] —— 发给我的私信
//	[系统] —— talk_type = 0 的系统消息
//
// ★ 昵称/颜色一律**实时**从 users 表取(不是发消息时存的快照),
//
//	这样玩家改了个性昵称、换了昵称颜色, 聊天里也会跟着变。
func (h *EzfyHandler) HomeChat(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	type row struct {
		key     string
		tag     string
		user    string
		color   string
		content string
		at      time.Time
	}
	rows := []row{}

	// 实时昵称/颜色(用户 id -> {昵称, 颜色})
	nickOf := func(ids []uint) map[uint][2]string {
		out := map[uint][2]string{}
		if len(ids) == 0 {
			return out
		}
		var us []model.User
		h.DB.Select("id, nickname, color").Where("id IN ?", ids).Find(&us)
		for _, u := range us {
			out[u.ID] = [2]string{u.Nickname, u.Color}
		}
		return out
	}
	ids := []uint{}
	push := func(id uint) { ids = append(ids, id) }

	// 系统
	var sys []model.EzfyChat
	h.DB.Where("talk_type = 0").Order("id DESC").Limit(10).Find(&sys)
	for _, m := range sys {
		rows = append(rows, row{"sys-" + strconv.Itoa(int(m.ID)), "系统", m.UserName, "", m.Content, m.CreatedAt})
	}
	// 军团(我的军团)
	if cp := h.myCorpsOf(uid); cp != nil {
		var cs []model.EzfyCorpsChat
		h.DB.Where("corps_id = ?", cp.ID).Order("id DESC").Limit(10).Find(&cs)
		for _, m := range cs {
			push(m.UserId)
			rows = append(rows, row{"corps-" + strconv.Itoa(int(m.ID)), "军团", m.UserName, "", m.Content, m.CreatedAt})
		}
	}
	// 私聊(发给我的)
	var pms []model.PrivateMessage
	h.DB.Where("receiver_id = ?", uid).Order("id DESC").Limit(10).Find(&pms)
	for _, m := range pms {
		push(m.SenderID)
		rows = append(rows, row{"pm-" + strconv.Itoa(int(m.ID)), "私聊", "", "", m.Content, m.CreatedAt})
	}
	// 世界(公共频道)
	var pub []model.EzfyChat
	h.DB.Where("channel = ? AND talk_type = 1", ezfyChanPublic).Order("id DESC").Limit(10).Find(&pub)
	for _, m := range pub {
		push(m.UserId)
		rows = append(rows, row{"pub-" + strconv.Itoa(int(m.ID)), "世界", m.UserName, "", m.Content, m.CreatedAt})
	}

	// 用实时昵称/颜色覆盖快照
	nick := nickOf(ids)
	fix := func(r *row, uid uint) {
		if uid == 0 {
			return
		}
		if v, ok := nick[uid]; ok {
			if v[0] != "" {
				r.user = v[0]
			}
			r.color = v[1]
		}
	}
	// 回填 uid 到 row(简单起见按 key 前缀再查一次)
	uidOf := map[string]uint{}
	for _, m := range sys {
		uidOf["sys-"+strconv.Itoa(int(m.ID))] = 0
	}
	if cp := h.myCorpsOf(uid); cp != nil {
		var cs []model.EzfyCorpsChat
		h.DB.Where("corps_id = ?", cp.ID).Order("id DESC").Limit(10).Find(&cs)
		for _, m := range cs {
			uidOf["corps-"+strconv.Itoa(int(m.ID))] = m.UserId
		}
	}
	for _, m := range pms {
		uidOf["pm-"+strconv.Itoa(int(m.ID))] = m.SenderID
	}
	for _, m := range pub {
		uidOf["pub-"+strconv.Itoa(int(m.ID))] = m.UserId
	}
	for i := range rows {
		fix(&rows[i], uidOf[rows[i].key])
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].at.After(rows[j].at) })
	if len(rows) > 8 {
		rows = rows[:8]
	}
	views := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		// user_id 供前端点玩家名 → 游戏内「统帅信息」页(不能跳去家园个人主页)
		views = append(views, gin.H{"key": r.key, "tag": r.tag, "user_id": uidOf[r.key],
			"user_name": r.user, "color": r.color, "content": r.content, "created_at": r.at})
	}
	var online int64
	h.DB.Model(&model.EzfyProfile{}).Count(&online)
	resp.OK(c, gin.H{"chats": views, "players": online})
}

// ============ 交易所 ============

var ezfyResNames = map[int]string{1: "粮食", 2: "钢铁", 3: "石油", 4: "稀矿"}

func (h *EzfyHandler) ExchangeList(c *gin.Context) {
	uid := middleware.GetUID(c)
	var list []model.EzfyExchange
	h.DB.Where("status = 0").Order("id DESC").Limit(100).Find(&list)
	views := []gin.H{}
	for _, e := range list {
		if e.SellerId == uid && e.IsSystem != 1 {
			continue
		}
		views = append(views, gin.H{"id": e.ID, "seller_name": e.SellerName,
			"type": e.EsType, "type_name": ezfyResNames[e.EsType],
			"count": e.EsCount, "total_price": e.TotalPrice,
			"unit_price": e.TotalPrice / maxInt64(1, e.EsCount), "mine": e.SellerId == uid})
	}
	var mine []model.EzfyExchange
	h.DB.Where("seller_id = ? AND status = 0", uid).Order("id DESC").Find(&mine)
	mineViews := []gin.H{}
	for _, e := range mine {
		mineViews = append(mineViews, gin.H{"id": e.ID, "type": e.EsType,
			"type_name": ezfyResNames[e.EsType], "count": e.EsCount, "total_price": e.TotalPrice})
	}
	city := h.getOrCreateCity(uid)
	resp.OK(c, gin.H{"orders": views, "mine": mineViews, "gold": city.Gold})
}

func (h *EzfyHandler) ExchangeSell(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		EsType     int   `json:"es_type"`
		EsCount    int64 `json:"es_count"`
		TotalPrice int64 `json:"total_price"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if _, ok := ezfyResNames[req.EsType]; !ok {
		resp.ParamError(c, "资源类型错误")
		return
	}
	if req.EsCount <= 0 || req.TotalPrice <= 0 {
		resp.ParamError(c, "数量或价格错误")
		return
	}
	city := h.getOrCreateCity(uid)
	h.calcResource(&city)
	var stock int64
	switch req.EsType {
	case 1:
		stock = city.Food
	case 2:
		stock = city.Steel
	case 3:
		stock = city.Oil
	case 4:
		stock = city.Rare
	}
	if stock < req.EsCount {
		resp.ParamError(c, fmt.Sprintf("%s不足(现有%d)", ezfyResNames[req.EsType], stock))
		return
	}
	switch req.EsType {
	case 1:
		city.Food -= req.EsCount
	case 2:
		city.Steel -= req.EsCount
	case 3:
		city.Oil -= req.EsCount
	case 4:
		city.Rare -= req.EsCount
	}
	h.saveCityRes(&city)
	profile := h.ensureProfile(uid)
	h.DB.Create(&model.EzfyExchange{SellerId: uid, SellerName: profile.Nickname,
		EsType: req.EsType, EsCount: req.EsCount, TotalPrice: req.TotalPrice, Status: 0})
	resp.OK(c, gin.H{"msg": fmt.Sprintf("挂单成功: %s×%d 售%d黄金", ezfyResNames[req.EsType], req.EsCount, req.TotalPrice)})
}

func (h *EzfyHandler) ExchangeBuy(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Id uint `json:"id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	var e model.EzfyExchange
	if err := h.DB.Where("id = ? AND status = 0", req.Id).First(&e).Error; err != nil {
		resp.ParamError(c, "订单不存在或已成交")
		return
	}
	if e.SellerId == uid {
		resp.ParamError(c, "不能购买自己的挂单")
		return
	}
	city := h.getOrCreateCity(uid)
	h.calcResource(&city)
	if city.Gold < e.TotalPrice {
		resp.ParamError(c, fmt.Sprintf("黄金不足(需%d)", e.TotalPrice))
		return
	}
	city.Gold -= e.TotalPrice
	switch e.EsType {
	case 1:
		city.Food += e.EsCount
	case 2:
		city.Steel += e.EsCount
	case 3:
		city.Oil += e.EsCount
	case 4:
		city.Rare += e.EsCount
	}
	h.saveCityRes(&city)
	// 黄金转给卖家(受其黄金容量上限)
	var sellerCity model.EzfyCity
	if err := h.DB.Where("user_id = ?", e.SellerId).Order("id ASC").First(&sellerCity).Error; err == nil {
		sellerCity.Gold += e.TotalPrice
		if sellerCity.Gold > sellerCity.GoldCap {
			sellerCity.Gold = sellerCity.GoldCap
		}
		h.saveCityRes(&sellerCity)
	}
	h.DB.Model(&model.EzfyExchange{}).Where("id = ?", e.ID).
		Updates(map[string]interface{}{"status": 1, "buyer_id": uid})
	h.addReport(e.SellerId, 6, "交易成交",
		fmt.Sprintf("你挂单出售的%s×%d已被%s以%d黄金购得。", ezfyResNames[e.EsType], e.EsCount, h.ensureProfile(uid).Nickname, e.TotalPrice))
	resp.OK(c, gin.H{"msg": fmt.Sprintf("购买成功: %s×%d", ezfyResNames[e.EsType], e.EsCount)})
}

func (h *EzfyHandler) ExchangeCancel(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Id uint `json:"id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	var e model.EzfyExchange
	if err := h.DB.Where("id = ? AND seller_id = ? AND status = 0", req.Id, uid).First(&e).Error; err != nil {
		resp.ParamError(c, "订单不存在")
		return
	}
	city := h.getOrCreateCity(uid)
	switch e.EsType {
	case 1:
		city.Food += e.EsCount
	case 2:
		city.Steel += e.EsCount
	case 3:
		city.Oil += e.EsCount
	case 4:
		city.Rare += e.EsCount
	}
	h.saveCityRes(&city)
	h.DB.Model(&model.EzfyExchange{}).Where("id = ?", e.ID).Update("status", 2)
	resp.OK(c, gin.H{"msg": "已下架, 资源退回"})
}

// CorpsMembers 军团成员列表
func (h *EzfyHandler) CorpsMembers(c *gin.Context) {
	uid := middleware.GetUID(c)
	var mb model.EzfyCorpsMember
	if err := h.DB.Where("user_id = ?", uid).First(&mb).Error; err != nil {
		resp.OK(c, gin.H{"members": []gin.H{}, "in_corps": false})
		return
	}
	var members []model.EzfyCorpsMember
	h.DB.Where("corps_id = ?", mb.CorpsId).Order("is_leader DESC, id ASC").Find(&members)
	// ★ 第九轮：军团长可任命副团长/参谋长；军团长与副团长都能发军团邮件
	canManage := mb.IsLeader == 1
	canMail := canManage || mb.Title == ezfyCorpsTitleVice
	views := []gin.H{}
	for _, m := range members {
		p := h.ensureProfile(m.UserId)
		var u model.User
		h.DB.First(&u, m.UserId)
		views = append(views, gin.H{"user_id": m.UserId, "name": ezfyNickOf(p, &u),
			"is_leader": m.IsLeader, "title": m.Title,
			"prestige": p.Prestige, "rank_name": ezfyRankName(p.Prestige)})
	}
	resp.OK(c, gin.H{"members": views, "in_corps": true,
		"can_manage": canManage, "can_mail": canMail,
		"my_title": mb.Title, "is_leader": mb.IsLeader})
}

// ============ 被占城市管理 ============

func (h *EzfyHandler) OccupyList(c *gin.Context) {
	uid := middleware.GetUID(c)
	var cities []model.EzfyCity
	h.DB.Where("user_id = ?", uid).Find(&cities)
	ids := make([]int64, 0, len(cities))
	for _, ct := range cities {
		ids = append(ids, int64(ct.ID))
	}
	views := []gin.H{}
	if len(ids) > 0 {
		var list []model.EzfyOccupy
		h.DB.Where("atk_city_id IN ? AND status = 1", ids).Find(&list)
		for _, o := range list {
			p := h.ensureProfile(o.DefUserId)
			views = append(views, gin.H{"id": o.ID, "x": o.X, "y": o.Y,
				"city_name": o.CityName, "def_user": p.Nickname})
		}
	}
	resp.OK(c, gin.H{"occupies": views})
}

func (h *EzfyHandler) OccupyOp(c *gin.Context) {
	uid := middleware.GetUID(c)
	op := c.Param("op")
	var req struct {
		OccupyId uint `json:"occupy_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	var cities []model.EzfyCity
	h.DB.Where("user_id = ?", uid).Find(&cities)
	ids := make([]int64, 0, len(cities))
	for _, ct := range cities {
		ids = append(ids, int64(ct.ID))
	}
	var o model.EzfyOccupy
	if err := h.DB.Where("id = ? AND atk_city_id IN ? AND status = 1", req.OccupyId, ids).First(&o).Error; err != nil {
		resp.ParamError(c, "占领记录不存在")
		return
	}
	var city model.EzfyCity
	if err := h.DB.First(&city, o.CityId).Error; err != nil {
		resp.ParamError(c, "城市数据不存在")
		return
	}
	switch op {
	case "build":
		h.DB.Model(&model.EzfyOccupy{}).Where("id = ?", o.ID).Update("status", 4)
		h.DB.Model(&model.EzfyCity{}).Where("id = ?", city.ID).
			Updates(map[string]interface{}{"feelings": 50, "grievance": 0, "last_time": time.Now().UnixMilli()})
		h.addReport(uid, 5, "建城成功",
			fmt.Sprintf("你将被占领的城市[%s]正式建立为自己的城市, 可在城市列表切换操作。", city.Name))
		resp.OK(c, gin.H{"msg": "建城成功"})
	case "destroy":
		name := o.CityName
		h.deleteCityData(int64(city.ID))
		h.DB.Model(&model.EzfyOccupy{}).Where("id = ?", o.ID).Update("status", 3)
		h.addReport(uid, 5, "摧毁城市", fmt.Sprintf("你摧毁了占领的城市[%s]。", name))
		resp.OK(c, gin.H{"msg": "摧毁成功"})
	case "return":
		h.DB.Model(&model.EzfyCity{}).Where("id = ?", city.ID).
			Updates(map[string]interface{}{"user_id": o.DefUserId})
		if city.Feelings < 20 {
			h.DB.Model(&model.EzfyCity{}).Where("id = ?", city.ID).Update("feelings", 20)
		}
		h.DB.Model(&model.EzfyOccupy{}).Where("id = ?", o.ID).Update("status", 2)
		h.addReport(o.DefUserId, 5, "城市归还",
			fmt.Sprintf("你被占领的城市[%s]已由敌方归还!\n民心已恢复。", city.Name))
		resp.OK(c, gin.H{"msg": "已归还"})
	default:
		resp.ParamError(c, "未知操作")
	}
}

// deleteCityData 删除城市及其所有数据
func (h *EzfyHandler) deleteCityData(cityId int64) {
	h.DB.Where("city_id = ?", cityId).Delete(&model.EzfyCityBuilding{})
	h.DB.Where("city_id = ?", cityId).Delete(&model.EzfyCityTroop{})
	h.DB.Where("city_id = ?", cityId).Delete(&model.EzfyCityTech{})
	h.DB.Where("city_id = ?", cityId).Delete(&model.EzfyTrainQueue{})
	h.DB.Where("city_id = ?", cityId).Delete(&model.EzfyWildland{})
	h.DB.Where("city_id = ?", cityId).Delete(&model.EzfyWounded{})
	h.DB.Where("city_id = ?", cityId).Delete(&model.EzfyCityTarget{})
	h.DB.Where("city_id = ?", cityId).Delete(&model.EzfyCityEffect{})
	h.DB.Where("city_id = ?", cityId).Delete(&model.EzfyOrder{})
	h.DB.Delete(&model.EzfyCity{}, cityId)
}

// WildlandFull 附属野地+被占城市(cityWild 页数据)
func (h *EzfyHandler) WildlandFull(c *gin.Context) {
	uid := middleware.GetUID(c)
	city := h.getOrCreateCity(uid)
	h.refreshCity(uid, &city)
	var wildlands []model.EzfyWildland
	h.DB.Where("city_id = ?", city.ID).Find(&wildlands)
	wildViews := []gin.H{}
	for _, w := range wildlands {
		// ★ 必须带 terrain_name：前端 loadWilds() 会用这里的返回**整体覆盖** wildlands，
		//   之前漏了这个字段，导致「附属野地」页面的【地形】列永远是空的。
		wildViews = append(wildViews, gin.H{"id": w.ID, "x": w.X, "y": w.Y,
			"wild_type": w.WildType, "level": w.Level, "status": w.Status,
			"terrain": ezfyTerrainEx(w.X, w.Y), "terrain_name": ezfyTerrainNameEx(w.X, w.Y),
			"continent": ezfyRegionName(w.X, w.Y)})
	}
	var occupies []model.EzfyOccupy
	h.DB.Where("atk_city_id = ? AND status = 1", city.ID).Find(&occupies)
	occViews := []gin.H{}
	for _, o := range occupies {
		p := h.ensureProfile(o.DefUserId)
		occViews = append(occViews, gin.H{"id": o.ID, "x": o.X, "y": o.Y,
			"city_name": o.CityName, "def_user": p.Nickname})
	}
	resp.OK(c, gin.H{"city": city, "wildlands": wildViews, "occupies": occViews,
		"hall_level": h.buildingLevel(city.ID, 1)})
}

// OrderView 命令详情
func (h *EzfyHandler) OrderView(c *gin.Context) {
	uid := middleware.GetUID(c)
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var o model.EzfyOrder
	if err := h.DB.Where("id = ? AND user_id = ?", id, uid).First(&o).Error; err != nil {
		resp.NotFound(c, "命令不存在")
		return
	}
	h.cfgs()
	troops := parseGroups(o.Troops)
	troopViews := []gin.H{}
	for _, g := range troops {
		name := "兵种" + strconv.Itoa(g.TroopId)
		if cfg := ezfyCfg.troop(g.TroopId); cfg != nil {
			name = cfg.Name
		}
		troopViews = append(troopViews, gin.H{"name": name, "count": g.Count})
	}
	// 目的地名称(复刻 report/viewCityTroopOut 的「目的地」)
	toName := h.ezfyTargetName(&o)
	// 出发地
	fromName := ""
	if city := h.cityOfOrder(&o, uid); city != nil {
		fromName = city.Name + "(" + strconv.Itoa(city.X) + "," + strconv.Itoa(city.Y) + ")"
	}
	// 携带资源
	resViews := []gin.H{}
	if res := h.parseResMap(o.Resources); len(res) > 0 {
		for _, k := range []string{"gold", "food", "steel", "oil", "rare"} {
			if v, ok := res[k]; ok && v > 0 {
				resViews = append(resViews, gin.H{"key": k, "name": ezfyResLabel(k), "count": v})
			}
		}
	}
	// 关联战报
	var rep model.EzfyReport
	hasReport := h.DB.Where("user_id = ? AND order_id = ?", uid, o.ID).Order("id DESC").First(&rep).Error == nil
	resp.OK(c, gin.H{
		"id": o.ID, "order_type": o.OrderType, "type_name": ezfyOrderTypeName(o.OrderType),
		"target_type": o.TargetType, "target_name": toName, "from_name": fromName,
		"target_x": o.TargetX, "target_y": o.TargetY, "status": o.Status,
		"status_name": ezfyOrderStatusName(o.Status),
		"start_time":  o.StartTime, "arrive_time": o.ArriveTime, "return_time": o.ReturnTime,
		"start_text": ezfyFmtTime(o.StartTime), "arrive_text": ezfyFmtTime(o.ArriveTime),
		"return_text": ezfyFmtTime(o.ReturnTime),
		"troops":      troopViews, "oil_used": o.OilUsed, "officer": o.Officer,
		"resources": resViews,
		"report_id": func() int64 {
			if hasReport {
				return int64(rep.ID)
			}
			return 0
		}(),
	})
}

// ezfyResLabel 资源 key → 中文名
func ezfyResLabel(k string) string {
	switch k {
	case "gold":
		return "黄金"
	case "food":
		return "粮食"
	case "steel":
		return "钢铁"
	case "oil":
		return "石油"
	default:
		return "稀矿"
	}
}

// ezfyFmtTime 毫秒时间戳 → yyyy-MM-dd HH:mm:ss
func ezfyFmtTime(ms int64) string {
	if ms <= 0 {
		return ""
	}
	return time.UnixMilli(ms).Format("2006-01-02 15:04:05")
}

// ezfyOrderStatusName 命令状态中文
func ezfyOrderStatusName(s int) string {
	switch s {
	case 0:
		return "行军中"
	case 1:
		return "驻守中"
	case 2:
		return "返航中"
	case 3:
		return "已完成"
	case 4:
		return "已终止"
	default:
		return "未知"
	}
}

// ezfyTargetName 命令目的地名称
// 复刻 report/index.html 的「目标：盆地(5)(347,2)」—— 野地/海野用「地形名(等级)」,
// 寇城/特殊目标用「类型(等级)」, 玩家城市用城市名。
func (h *EzfyHandler) ezfyTargetName(o *model.EzfyOrder) string {
	tt := o.TargetType
	if tt == 0 {
		// ★ 兜底：老数据/异常请求可能没带 target_type（前端正常都会带），
		//   这里按地图实际情况推断一下，避免军情列表里显示成「未知」。
		var n int64
		h.DB.Model(&model.EzfyCity{}).Where("x = ? AND y = ?", o.TargetX, o.TargetY).Count(&n)
		switch {
		case n > 0:
			tt = 3
		case h.ezfyIsKouCity(o.TargetX, o.TargetY):
			tt = 2
		default:
			tt = 1
		}
	}
	switch tt {
	case 1: // 野地(含海野): 地形名 + 等级
		return ezfyTerrainNameEx(o.TargetX, o.TargetY) +
			"(" + strconv.Itoa(ezfyWildlandLevel(o.TargetX, o.TargetY)) + ")"
	case 2: // 寇城
		return "寇城(" + strconv.Itoa(ezfyKouLevel(o.TargetX, o.TargetY)) + ")"
	case 4: // 特殊目标(活动野地/特殊城市)
		return "特殊(" + strconv.Itoa(ezfyWildlandLevel(o.TargetX, o.TargetY)) + ")"
	case 3:
		var c model.EzfyCity
		if err := h.DB.First(&c, o.TargetId).Error; err == nil {
			return c.Name
		}
		return "城市"
	default:
		return "未知"
	}
}
