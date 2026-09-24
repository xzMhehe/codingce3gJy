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
	// 聊天分页默认每页条数（用户反馈「聊天页太长了」）
	ezfyChatPageSize = 15
	ezfyChatPageMax  = 50
	// 系统消息比玩家发言长一点（「恭喜 xxx 晋升上校」这类），但也别长到刷屏
	ezfySysChatMaxRune = 120
	// 首页「世界聊天」预览条数（用户要求：默认展示 6 条）
	ezfyHomeChatLimit = 6
)

// ezfySysChat 往**系统频道**写一条消息（talk_type=0，只读）。
//
// 用途：把「军衔晋升 / 采集到宝物 / 战斗掉落装备 / 招募到五星军官」这类值得全服看到的事件
// 推到世界聊天与首页预览里（用户要求：这些事原来没有任何交互反馈）。
//
// 内容同样过一遍敏感词（玩家昵称可能被起成敏感词）并按长度截断。
func (h *EzfyHandler) ezfySysChat(format string, args ...interface{}) {
	content := fmt.Sprintf(format, args...)
	if filtered, blocked := ezfyFilterChat(content); blocked {
		return
	} else {
		content = filtered
	}
	if r := []rune(content); len(r) > ezfySysChatMaxRune {
		content = string(r[:ezfySysChatMaxRune])
	}
	if trimSpace(content) == "" {
		return
	}
	// ⚠️ 这里**必须用 map 建**，不能写成 &model.EzfyChat{...TalkType: 0}。
	//    model.EzfyChat.TalkType 带 `gorm:"default:1"` 标签，GORM 对「带 default 标签且当前是零值」
	//    的字段会**从 INSERT 里剔除**，让数据库默认值 1 生效 —— 结果就是系统消息被存成玩家消息
	//    (talk_type=1)，而系统频道按 `talk_type = 0` 查，永远查不到（首页预览同样漏掉）。
	//    走 map 时 GORM 只插入显式给出的列，零值不会被吞。
	h.DB.Model(&model.EzfyChat{}).Create(map[string]interface{}{
		"user_id":    0,
		"user_name":  "系统",
		"content":    content,
		"channel":    ezfyChanSystem,
		"talk_type":  0,
		"created_at": time.Now(),
	})
}

// ezfyChatPager 解析聊天分页参数（?page=1&size=15）
func ezfyChatPager(c *gin.Context) (page, size int) {
	page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	size, _ = strconv.Atoi(c.DefaultQuery("size", strconv.Itoa(ezfyChatPageSize)))
	if size < 1 {
		size = ezfyChatPageSize
	}
	if size > ezfyChatPageMax {
		size = ezfyChatPageMax
	}
	return page, size
}

// ChatList GET /games/ezfy/chat?channel=1|2|4&page=1&size=15
//
// ★ 用户要求：聊天页太长 → 分页；排序改**时间降序（最新的在最上面）**。
func (h *EzfyHandler) ChatList(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	channel := ezfyChanPublic
	if v, err := strconv.Atoi(c.DefaultQuery("channel", "1")); err == nil && v > 0 {
		channel = v
	}
	page, size := ezfyChatPager(c)
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
		var total int64
		h.DB.Model(&model.EzfyCorpsChat{}).Where("corps_id = ?", myCorps.ID).Count(&total)
		var list []model.EzfyCorpsChat
		h.DB.Where("corps_id = ?", myCorps.ID).Order("id DESC").
			Offset((page - 1) * size).Limit(size).Find(&list)
		ids := []uint{}
		for _, ch := range list {
			if ch.UserId > 0 {
				ids = append(ids, ch.UserId)
			}
		}
		nick := h.liveNicknames(ids)
		views := make([]gin.H, 0, len(list))
		// ★ 时间降序：最新的在最上面（原实现是取最新 N 条再反转成升序）
		for _, ch := range list {
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
		out["total"] = total
		out["page"] = page
		out["size"] = size
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
		var total int64
		h.DB.Model(&model.EzfyChat{}).Where("channel = ? AND talk_type = 0", ezfyChanSystem).Count(&total)
		var msgs []model.EzfyChat
		h.DB.Where("channel = ? AND talk_type = 0", ezfyChanSystem).Order("id DESC").
			Offset((page - 1) * size).Limit(size).Find(&msgs)
		mviews := make([]gin.H, 0, len(msgs))
		for _, m := range msgs {
			mviews = append(mviews, gin.H{"id": m.ID, "user_name": m.UserName, "content": m.Content,
				"created_at": m.CreatedAt, "talk_type": 0})
		}
		out["notices"] = nviews
		out["chats"] = mviews
		out["total"] = total
		out["page"] = page
		out["size"] = size
	default:
		var total int64
		h.DB.Model(&model.EzfyChat{}).Where("channel = ? AND talk_type = 1", ezfyChanPublic).Count(&total)
		var chats []model.EzfyChat
		h.DB.Where("channel = ? AND talk_type = 1", ezfyChanPublic).Order("id DESC").
			Offset((page - 1) * size).Limit(size).Find(&chats)
		// 实时昵称/颜色(玩家改了个性昵称, 历史消息也跟着变)
		ids := []uint{}
		for _, ch := range chats {
			if ch.UserId > 0 {
				ids = append(ids, ch.UserId)
			}
		}
		nick := h.liveNicknames(ids)
		views := make([]gin.H, 0, len(chats))
		for _, ch := range chats {
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
		out["total"] = total
		out["page"] = page
		out["size"] = size
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
//
// ★ 展示规则(用户要求)：取**最新 6 条**，按时间**升序**排列（最早的在上、最新的在下）。
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

	// ★ 用户要求：默认展示 6 条，**升序**（最早的在上、最新的在下）。
	//   所以先按时间降序取「最新的 6 条」，再翻转成升序输出。
	sort.Slice(rows, func(i, j int) bool { return rows[i].at.After(rows[j].at) })
	if len(rows) > ezfyHomeChatLimit {
		rows = rows[:ezfyHomeChatLimit]
	}
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
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

// 交易所计价货币
const (
	ezfyMoneyGold    = 1 // 黄金（玩家挂单只能用它）
	ezfyMoneyDiamond = 2 // 钻石（只有系统挂单能用）
)

func ezfyMoneyName(cur int) string {
	if cur == ezfyMoneyDiamond {
		return "钻石"
	}
	return "黄金"
}

func (h *EzfyHandler) ExchangeList(c *gin.Context) {
	uid := middleware.GetUID(c)
	// ★ 2026-09-24 用户要求：卖家挂单/我的挂单都做分页（默认每页 10 条）
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	if size < 1 {
		size = 10
	}
	if size > 50 {
		size = 50
	}
	// 卖家挂单(在售、非自己的)
	// ★ 2026-09-24 用户要求: 卖家挂单加「资源类别」检索(单多了得一页页翻)
	esType, _ := strconv.Atoi(c.DefaultQuery("es_type", "0"))
	base := h.DB.Model(&model.EzfyExchange{}).
		Where("status = 0 AND NOT (seller_id = ? AND is_system != 1)", uid)
	if esType >= 1 && esType <= 4 {
		if _, ok := ezfyResNames[esType]; ok {
			base = base.Where("es_type = ?", esType)
		}
	}
	var total int64
	base.Count(&total)
	var list []model.EzfyExchange
	base.Order("is_system DESC, id DESC").Offset((page - 1) * size).Limit(size).Find(&list)
	views := []gin.H{}
	for _, e := range list {
		seller := e.SellerName
		if e.IsSystem == 1 {
			seller = "系统"
		}
		views = append(views, gin.H{"id": e.ID, "seller_name": seller,
			"type": e.EsType, "type_name": ezfyResNames[e.EsType],
			"count": e.EsCount, "total_price": e.TotalPrice,
			"currency": e.Currency, "currency_name": ezfyMoneyName(e.Currency),
			"unit_price": e.TotalPrice / maxInt64(1, e.EsCount), "mine": e.SellerId == uid})
	}
	// 我的挂单(分页)
	mpage, _ := strconv.Atoi(c.DefaultQuery("mpage", "1"))
	if mpage < 1 {
		mpage = 1
	}
	msize, _ := strconv.Atoi(c.DefaultQuery("msize", "10"))
	if msize < 1 {
		msize = 10
	}
	if msize > 50 {
		msize = 50
	}
	var mtotal int64
	h.DB.Model(&model.EzfyExchange{}).Where("seller_id = ? AND status = 0", uid).Count(&mtotal)
	var mine []model.EzfyExchange
	h.DB.Where("seller_id = ? AND status = 0", uid).Order("id DESC").
		Offset((mpage - 1) * msize).Limit(msize).Find(&mine)
	mineViews := []gin.H{}
	for _, e := range mine {
		mineViews = append(mineViews, gin.H{"id": e.ID, "type": e.EsType,
			"type_name": ezfyResNames[e.EsType], "count": e.EsCount, "total_price": e.TotalPrice,
			"currency": e.Currency, "currency_name": ezfyMoneyName(e.Currency)})
	}
	city := h.getOrCreateCity(uid)
	resp.OK(c, gin.H{"orders": views, "total": total, "page": page, "size": size,
		"mine": mineViews, "mtotal": mtotal, "mpage": mpage, "msize": msize,
		"gold":    city.Gold,
		"diamond": h.ensureProfile(uid).Diamond})
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
	// ★ 玩家挂单一律**黄金计价**（用户规则：玩家卖只能按黄金买卖）。
	//   钻石定价是系统挂单专属能力，由管理端「交易行维护」新增。
	h.DB.Create(&model.EzfyExchange{SellerId: uid, SellerName: profile.Nickname,
		EsType: req.EsType, EsCount: req.EsCount, TotalPrice: req.TotalPrice,
		Status: 0, IsSystem: 0, Currency: ezfyMoneyGold})
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
	money := ezfyMoneyName(e.Currency)
	if e.Currency == ezfyMoneyDiamond {
		// 钻石计价（只有系统挂单会出现）—— 扣档案上的钻石余额
		p := h.ensureProfile(uid)
		if p.Diamond < e.TotalPrice {
			resp.ParamError(c, fmt.Sprintf("钻石不足(需%d, 现有%d)", e.TotalPrice, p.Diamond))
			return
		}
		h.DB.Model(&model.EzfyProfile{}).Where("id = ?", p.ID).
			Update("diamond", p.Diamond-e.TotalPrice)
	} else {
		if city.Gold < e.TotalPrice {
			resp.ParamError(c, fmt.Sprintf("黄金不足(需%d)", e.TotalPrice))
			return
		}
		city.Gold -= e.TotalPrice
	}
	// ★ 买的资源**不受仓储上限截断**（用户确认：交易所买的资源超上限也能买到、不会凭空少）
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
	// 黄金/钻石转给卖家；系统挂单不回款给任何玩家
	// ★ 2026-09-24 规则修正：卖家收款同样不受仓储上限截断（只有数据库字段最大值才溢出）
	if e.IsSystem != 1 {
		var sellerCity model.EzfyCity
		if err := h.DB.Where("user_id = ?", e.SellerId).Order("id ASC").First(&sellerCity).Error; err == nil {
			sellerCity.Gold = ezfyAddRes(sellerCity.Gold, e.TotalPrice)
			h.saveCityRes(&sellerCity)
		}
	}
	if e.IsSystem != 1 {
		h.DB.Model(&model.EzfyExchange{}).Where("id = ?", e.ID).
			Updates(map[string]interface{}{"status": 1, "buyer_id": uid})
		h.addReport(e.SellerId, 6, "交易成交",
			fmt.Sprintf("你挂单出售的%s×%d已被%s以%d%s购得。", ezfyResNames[e.EsType], e.EsCount, h.ensureProfile(uid).Nickname, e.TotalPrice, money))
	}
	// ★ 系统挂单不写成交状态、不通知卖家：保持 status=0 恒在售，
	//   玩家可以反复购买（用户要求：资源大/中/小包是「买不完」的无限库存）。
	resp.OK(c, gin.H{"msg": fmt.Sprintf("购买成功: %s×%d（花费%d%s）", ezfyResNames[e.EsType], e.EsCount, e.TotalPrice, money)})
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
		// ★ 2026-09-24 军衔 ×10 后卡控：占城「建城」同样受军衔可建城数限制，
		//   与 CreateCity 同一口径，防绕过军衔上限白嫖一座城。
		prof := h.ensureProfile(uid)
		var owned int64
		h.DB.Model(&model.EzfyCity{}).Where("user_id = ?", uid).Count(&owned)
		maxCity := ezfyRankCityMax(prof.Prestige)
		if int(owned) >= maxCity {
			resp.ParamError(c, fmt.Sprintf("当前军衔「%s」最多只能拥有 %d 座城市（已有 %d 座），提升声望可解锁更多",
				ezfyRankName(prof.Prestige), maxCity, owned))
			return
		}
		h.DB.Model(&model.EzfyOccupy{}).Where("id = ?", o.ID).Update("status", 4)
		// ★ 修复：「建立城市」原来只改民心、**没有把城市归属改成自己**，
		//   导致玩家点了建城、战报也说「已建立为自己的城市」，实际城市还是别人的。
		//   这里把 user_id 真正过户给占领方，民心重置为 50。
		h.DB.Model(&model.EzfyCity{}).Where("id = ?", city.ID).
			Updates(map[string]interface{}{
				"user_id": uid, "feelings": 50, "grievance": 0,
				"last_time": time.Now().UnixMilli(),
			})
		h.addReport(uid, 5, "建城成功",
			fmt.Sprintf("你将被占领的城市[%s]正式建立为自己的城市, 可在城市列表切换操作。", city.Name))
		h.addReport(o.DefUserId, 5, "城市被占领",
			fmt.Sprintf("你的城市[%s](%d,%d)已被敌方建立为自己的城市, 不再属于你。", o.CityName, o.X, o.Y))
		resp.OK(c, gin.H{"msg": "建城成功"})
	case "destroy":
		name := o.CityName
		// ★ 2026-09-24 用户规则：占城摧毁 = 坐标变回原来的平原土地，建筑啥的都没了。
		//   除了清掉城市数据，还要删掉该坐标的「玩家城」地图区域记录（与 ezfyDestroyCity 同口径），
		//   否则地图上永远残留一块「已摧毁城市」的格子。
		h.deleteCityData(int64(city.ID))
		h.DB.Where("x = ? AND y = ?", city.X, city.Y).Delete(&model.EzfyMapArea{})
		h.DB.Model(&model.EzfyOccupy{}).Where("id = ?", o.ID).Update("status", 3)
		h.addReport(uid, 5, "摧毁城市",
			fmt.Sprintf("你摧毁了占领的城市[%s](%d,%d), 该坐标恢复为普通平原。", name, city.X, city.Y))
		if o.DefUserId != 0 {
			h.addReport(o.DefUserId, 5, "城市被摧毁",
				fmt.Sprintf("你被占领的城市[%s](%d,%d)已被敌方摧毁, 该坐标恢复为普通平原。", name, city.X, city.Y))
		}
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
	// ★ 2026-09-24：占城摧毁后军官/装备一并清理（对齐 ezfyDestroyCity，不留孤儿数据）
	h.DB.Where("city_id = ?", cityId).Delete(&model.EzfyOfficer{})
	h.DB.Where("city_id = ?", cityId).Delete(&model.EzfyEquipment{})
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
	case ezfyOrderStatusBattle:
		return "战斗中"
	case ezfyOrderStatusWaiting:
		return "等待"
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
	case 1: // 野地: 地形名 + 等级(海上的野地用「海底森林」)
		tn := ezfyTerrainNameEx(o.TargetX, o.TargetY)
		if ezfyTerrain(o.TargetX, o.TargetY) == 8 {
			tn = "海底森林"
		}
		return tn + "(" + strconv.Itoa(ezfyWildlandLevel(o.TargetX, o.TargetY)) + ")"
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
