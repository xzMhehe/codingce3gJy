package ezfy

import (
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

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
	// 首页「世界聊天」预览条数（默认展示 2 条）
	ezfyHomeChatLimit = 2
)

// ezfySysChat 往**系统频道**写一条消息（talk_type=0，只读）。
//
// 用途：把「军衔晋升 / 采集到宝物 / 战斗掉落装备 / 招募到五星军官」这类值得全服看到的事件
// 推到世界聊天与首页预览里（这些事原来没有任何交互反馈）。
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
// ★ 聊天页太长 → 分页；排序改**时间降序（最新的在最上面）**。
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
	h.home().Select("id, nickname, color").Where("id IN ?", ids).Find(&us)
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
// 汇总四个来源并按时间倒序, 每条带频道标识():
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
// ★ 展示规则()：取**最新 N 条**（ezfyHomeChatLimit），按时间**升序**排列（最早的在上、最新的在下）。
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
		h.home().Select("id, nickname, color").Where("id IN ?", ids).Find(&us)
		for _, u := range us {
			out[u.ID] = [2]string{u.Nickname, u.Color}
		}
		return out
	}
	ids := []uint{}
	push := func(id uint) { ids = append(ids, id) }

	// ★ 2026-10-03 性能：四个来源 + 在线人数彼此独立，并行打 RDS，
	//   把首页 30s 轮询 chat/home 的串行查询压成一次往返（顺带清掉了重复的军团查询）。
	var (
		sys    []model.EzfyChat
		cs     []model.EzfyCorpsChat
		pms    []model.PrivateMessage
		pub    []model.EzfyChat
		online int64
	)
	var wg sync.WaitGroup
	wg.Add(5)
	go func() { // 系统
		defer wg.Done()
		var m []model.EzfyChat
		h.DB.Where("talk_type = 0").Order("id DESC").Limit(10).Find(&m)
		sys = m
	}()
	go func() { // 我的军团 + 军团聊天
		defer wg.Done()
		mc := h.myCorpsOf(uid)
		if mc == nil {
			return
		}
		var m []model.EzfyCorpsChat
		h.DB.Where("corps_id = ?", mc.ID).Order("id DESC").Limit(10).Find(&m)
		cs = m
	}()
	go func() { // 私聊(发给我的)
		defer wg.Done()
		var m []model.PrivateMessage
		h.DB.Where("receiver_id = ?", uid).Order("id DESC").Limit(10).Find(&m)
		pms = m
	}()
	go func() { // 世界(公共频道)
		defer wg.Done()
		var m []model.EzfyChat
		h.DB.Where("channel = ? AND talk_type = 1", ezfyChanPublic).Order("id DESC").Limit(10).Find(&m)
		pub = m
	}()
	go func() { // 在线人数
		defer wg.Done()
		var n int64
		h.DB.Model(&model.EzfyProfile{}).Count(&n)
		online = n
	}()
	wg.Wait()

	// 系统
	for _, m := range sys {
		rows = append(rows, row{"sys-" + strconv.Itoa(int(m.ID)), "系统", m.UserName, "", m.Content, m.CreatedAt})
	}
	// 军团(我的军团)
	for _, m := range cs {
		push(m.UserId)
		rows = append(rows, row{"corps-" + strconv.Itoa(int(m.ID)), "军团", m.UserName, "", m.Content, m.CreatedAt})
	}
	// 私聊(发给我的)
	for _, m := range pms {
		push(m.SenderID)
		rows = append(rows, row{"pm-" + strconv.Itoa(int(m.ID)), "私聊", "", "", m.Content, m.CreatedAt})
	}
	// 世界(公共频道)
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
	for _, m := range cs {
		uidOf["corps-"+strconv.Itoa(int(m.ID))] = m.UserId
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

	// ★ 默认展示 ezfyHomeChatLimit 条，**升序**（最早的在上、最新的在下）。
	//   所以先按时间降序取「最新的 N 条」，再翻转成升序输出。
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
	resp.OK(c, gin.H{"chats": views, "players": online})
}

// ============ 交易所 ============

var ezfyResNames = map[int]string{1: "粮食", 2: "钢铁", 3: "石油", 4: "稀矿"}

// 交易所计价货币
const (
	ezfyMoneyGold    = 1 // 黄金（玩家挂单只能用它）
	ezfyMoneyDiamond = 2 // 钻石（只有系统挂单能用）
	// ★ 2026-10-04 交易所建筑 ID（seed: {ID:11, Name:"交易所", MaxLevel:10}）。
	//   玩家挂单上限 = 交易所等级 × 2（见 ExchangeSell / ExchangeList）。
	ezfyBuildingExchange = 11
)

func ezfyMoneyName(cur int) string {
	if cur == ezfyMoneyDiamond {
		return "钻石"
	}
	return "黄金"
}

// ezfyExchangeLocks 交易所挂单出售/购买/下架的并发锁（按玩家 id 分片）。
//
// ★ 2026-10-06 修复「交易所连续点击重复执行」：
//
//	挂单出售/购买/下架都是「读库存 → 扣资源/收款 → 建单/改状态」的读-改-写，
//	连点会并发进入同一段结算：出售连点会重复扣资源、重复建挂单；
//	下架连点会重复退回资源（挂单重复返款）；购买连点会重复扣钱。
//	按玩家串行化后，后到的请求看到资源已扣/订单已成交，直接按正常校验拒绝。
var ezfyExchangeLocks [64]sync.Mutex

func ezfyExchangeLock(uid uint) *sync.Mutex { return &ezfyExchangeLocks[uid%64] }

// ezfyExchangeCityOf 取挂单的「归属城市」——成交收款 / 下架退款都打回这座城。
//
// ★ 2026-10-07 修复「成交后黄金进错城」：
//
//	挂单时资源是从**当前操作城市**（currentCity，玩家可在城市列表切换）扣的，
//	但成交打款与下架退款原来一律 `WHERE user_id = ? ORDER BY id ASC`（恒取主城），
//	玩家切城后挂单 → 黄金/退回的资源全跑进主城，与卖资源的城市对不上。
//
// 口径：优先用挂单时记录的城市（且必须仍属于该卖家，防止城市被摧毁/过户后串号）；
//
//	cityId<=0（老数据）或该城已不属于卖家时，回落卖家主城（id 最小）保持兼容。
func (h *EzfyHandler) ezfyExchangeCityOf(sellerId uint, cityId int64) (model.EzfyCity, bool) {
	if cityId > 0 {
		var c model.EzfyCity
		if err := h.DB.Where("id = ? AND user_id = ?", cityId, sellerId).First(&c).Error; err == nil {
			return c, true
		}
	}
	var c model.EzfyCity
	if err := h.DB.Where("user_id = ?", sellerId).Order("id ASC").First(&c).Error; err == nil {
		return c, true
	}
	return c, false
}

func (h *EzfyHandler) ExchangeList(c *gin.Context) {
	uid := middleware.GetUID(c)
	// ★ 2026-09-24 卖家挂单/我的挂单都做分页（默认每页 10 条）
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
	// ★ 2026-09-24  卖家挂单加「资源类别」检索(单多了得一页页翻)
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
	// ★ 2026-10-04 玩家挂单上限 = 交易所等级×2（与 ExchangeSell 卡控同口径，前端展示用）
	sellMax := h.buildingLevel(city.ID, ezfyBuildingExchange) * 2
	resp.OK(c, gin.H{"orders": views, "total": total, "page": page, "size": size,
		"mine": mineViews, "mtotal": mtotal, "mpage": mpage, "msize": msize,
		"sell_max": sellMax,
		"gold":     city.Gold,
		"diamond":  h.ensureProfile(uid).Diamond,
		// ★ 2026-09-30 向系统出售资源：下发回收比例 / 手续费 / 黄金上限，供前端展示与判断
		"sys_sell_ratio": ezfySysSellRatioMap(),
		"sys_sell_fee":   ezfySysSellFeePct,
		"gold_max":       ezfyResMaxOf("gold")})
}

func (h *EzfyHandler) ExchangeSell(c *gin.Context) {
	uid := middleware.GetUID(c)
	// ★ 2026-10-06 挂单出售串行化，防连点重复扣资源/重复建单
	mu := ezfyExchangeLock(uid)
	mu.Lock()
	defer mu.Unlock()
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
	// ★ 2026-09-28 玩家挂单出售的黄金价格上限卡控到 10 亿，防止标天价
	const sellGoldMax = int64(1000000000) // 10 亿
	if req.TotalPrice > sellGoldMax {
		resp.ParamError(c, fmt.Sprintf("出售价格不能超过%d黄金", sellGoldMax))
		return
	}
	// ★ 2026-09-28 挂单出售单价按 1:100 卡控（卖 1 粮食单价不能超过 100 黄金），
	//   数量随意（1/2/50/60 都行），比例在「二战系统配置」页可灵活配置。
	unitPriceMax := ezfySellPriceMax()
	if req.TotalPrice/req.EsCount > int64(unitPriceMax) {
		resp.ParamError(c, fmt.Sprintf("挂单单价不能超过%d黄金/单位(当前%d)",
			unitPriceMax, req.TotalPrice/req.EsCount))
		return
	}
	city := h.getOrCreateCity(uid)
	h.calcResource(&city)
	// ★ 2026-10-04 玩家挂单上限 = 交易所等级 × 2（存量挂单不动，新挂单卡控）。
	//   必须在扣资源**之前**校验，否则被拒的挂单会白扣一次资源。
	exchangeLv := h.buildingLevel(city.ID, ezfyBuildingExchange)
	orderLimit := exchangeLv * 2
	if orderLimit <= 0 {
		resp.ParamError(c, "需要先建造「交易所」才能挂单出售")
		return
	}
	var active int64
	h.DB.Model(&model.EzfyExchange{}).Where("seller_id = ? AND status = 0", uid).Count(&active)
	if active >= int64(orderLimit) {
		resp.ParamError(c, fmt.Sprintf("挂单已达上限：交易所%d级 → 最多%d单（当前%d单）。请升级交易所，或等现有挂单成交/下架后再挂",
			exchangeLv, orderLimit, active))
		return
	}
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
	// ★ 2026-10-07 记下「挂单所在城市」：资源是从这座城扣的，
	//   成交收款 / 下架退款都要回到同一座城（否则切城挂单会让黄金进错城）。
	h.DB.Create(&model.EzfyExchange{SellerId: uid, SellerName: profile.Nickname,
		CityId: int64(city.ID),
		EsType: req.EsType, EsCount: req.EsCount, TotalPrice: req.TotalPrice,
		Status: 0, IsSystem: 0, Currency: ezfyMoneyGold})
	resp.OK(c, gin.H{"msg": fmt.Sprintf("挂单成功: %s×%d 售%d黄金", ezfyResNames[req.EsType], req.EsCount, req.TotalPrice)})
}

func (h *EzfyHandler) ExchangeBuy(c *gin.Context) {
	uid := middleware.GetUID(c)
	// ★ 2026-10-06 购买串行化，防连点重复扣钱/重复成交
	mu := ezfyExchangeLock(uid)
	mu.Lock()
	defer mu.Unlock()
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
		// ★ 2026-09-28 钻石流水
		h.logDiamond(uid, -e.TotalPrice, "交易所购买挂单")
	} else {
		if city.Gold < e.TotalPrice {
			resp.ParamError(c, fmt.Sprintf("黄金不足(需%d)", e.TotalPrice))
			return
		}
		city.Gold -= e.TotalPrice
	}
	// ★ 买的资源**不受仓储上限截断**（用户确认：交易所买的资源超上限也能买到、不会凭空少）
	// ★ 2026-09-30 仍受「资源最大值」硬上限（21 亿）约束
	switch e.EsType {
	case 1:
		city.Food = ezfyAddResMax("food", city.Food, e.EsCount)
	case 2:
		city.Steel = ezfyAddResMax("steel", city.Steel, e.EsCount)
	case 3:
		city.Oil = ezfyAddResMax("oil", city.Oil, e.EsCount)
	case 4:
		city.Rare = ezfyAddResMax("rare", city.Rare, e.EsCount)
	}
	h.saveCityRes(&city)
	// 黄金/钻石转给卖家；系统挂单不回款给任何玩家
	// ★ 2026-09-24 规则修正：卖家收款同样不受仓储上限截断（只有数据库字段最大值才溢出）
	// ★ 2026-09-30 仍受「资源最大值」硬上限（21 亿）约束
	if e.IsSystem != 1 {
		// ★ 2026-10-07 黄金打给「挂单所在城市」，而不是卖家主城（用户反馈 bug：
		//   在 B 城卖的资源，成交黄金却进了主城 A）。
		if sellerCity, ok := h.ezfyExchangeCityOf(e.SellerId, e.CityId); ok {
			sellerCity.Gold = ezfyAddResMax("gold", sellerCity.Gold, e.TotalPrice)
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
	//   玩家可以反复购买（资源大/中/小包是「买不完」的无限库存）。
	resp.OK(c, gin.H{"msg": fmt.Sprintf("购买成功: %s×%d（花费%d%s）", ezfyResNames[e.EsType], e.EsCount, e.TotalPrice, money)})
}

func (h *EzfyHandler) ExchangeCancel(c *gin.Context) {
	uid := middleware.GetUID(c)
	// ★ 2026-10-06 下架串行化，防连点重复退回资源
	mu := ezfyExchangeLock(uid)
	mu.Lock()
	defer mu.Unlock()
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
	// ★ 2026-10-07 退回「挂单所在城市」（资源就是从这座城扣的），不是当前所在城。
	//   老数据 city_id=0 / 城市已不存在时回落到当前城，保持兼容。
	city := h.getOrCreateCity(uid)
	if c, ok := h.ezfyExchangeCityOf(uid, e.CityId); ok {
		city = c
	}
	// ★★ 2026-10-09 用户反馈「交易所下架资源会被吃掉」：根因是这里**没先懒结算**就 saveCityRes ——
	//   city 是「上次落库那一刻」的旧值，saveCityRes 会把这几分钟/几小时的产量一起覆盖掉。
	//   与 ExchangeSell 同口径：先把产量结算进来，再退回挂单的资源。
	h.calcResource(&city)
	before := city.Food + city.Steel + city.Oil + city.Rare
	switch e.EsType {
	case 1:
		city.Food = ezfyAddResMax("food", city.Food, e.EsCount)
	case 2:
		city.Steel = ezfyAddResMax("steel", city.Steel, e.EsCount)
	case 3:
		city.Oil = ezfyAddResMax("oil", city.Oil, e.EsCount)
	case 4:
		city.Rare = ezfyAddResMax("rare", city.Rare, e.EsCount)
	}
	after := city.Food + city.Steel + city.Oil + city.Rare
	h.saveCityRes(&city)
	h.DB.Model(&model.EzfyExchange{}).Where("id = ?", e.ID).Update("status", 2)
	// ★ 退回量按「资源最大值」封顶，超出部分会被丢弃 → 明确告知（别让玩家以为又是丢资源的 bug）。
	back := after - before
	msg := fmt.Sprintf("已下架, 退回%s×%d", ezfyResNames[e.EsType], back)
	if lost := e.EsCount - back; lost > 0 {
		msg = fmt.Sprintf("已下架, 退回%s×%d（其中 %d 因资源已达最大值被丢弃, 可先消耗资源再下架）",
			ezfyResNames[e.EsType], back, lost)
	}
	resp.OK(c, gin.H{"msg": msg})
}

// ezfySysSellFeePct 向系统出售资源的手续费百分比（默认 10 = 10%）。
//
// ★ 2026-09-30 「玩家获得的黄金手续费：玩家获取的黄金价格 10% 扣除」。
const ezfySysSellFeePct = 10

// ezfySysSellRatioMap 向系统出售资源回收比例 map（es_type → 每100单位黄金），供前端展示。
func ezfySysSellRatioMap() map[int]int {
	return map[int]int{1: ezfySysSellRatio(1), 2: ezfySysSellRatio(2), 3: ezfySysSellRatio(3), 4: ezfySysSellRatio(4)}
}

// ExchangeSysSell POST /games/ezfy/exchange/sys-sell {es_type, es_count}
//
// ★ 2026-09-30 「玩家可向系统出售资源获得黄金」：
//   - 把资源**直接卖给系统**（不走挂单，不产生订单行）；
//   - 直接扣城市资源、加城市黄金；
//   - 每 100 单位 → 按配置比例换黄金（默认粮10/钢10/油20/稀25，交易行维护可配）；
//   - 玩家实得黄金再扣 10% 手续费；
//   - 黄金仍受「黄金资源最大值」（res_max_gold）硬上限约束；若本次到账会超上限，
//     超出部分会丢失，必须提醒玩家（gold_lost=true + lost_gold），没超过不提醒。
func (h *EzfyHandler) ExchangeSysSell(c *gin.Context) {
	uid := middleware.GetUID(c)
	// ★ 2026-10-06 向系统出售串行化，防连点重复扣资源/重复得黄金
	mu := ezfyExchangeLock(uid)
	mu.Lock()
	defer mu.Unlock()
	var req struct {
		EsType  int   `json:"es_type"`
		EsCount int64 `json:"es_count"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if _, ok := ezfyResNames[req.EsType]; !ok {
		resp.ParamError(c, "资源类型错误")
		return
	}
	if req.EsCount <= 0 {
		resp.ParamError(c, "数量必须大于 0")
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
	// 扣资源
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

	// 换算黄金：每 100 单位 → N 黄金；实得再扣 10% 手续费
	ratio := int64(ezfySysSellRatio(req.EsType))
	base := req.EsCount * ratio / 100
	received := base * (100 - ezfySysSellFeePct) / 100
	fee := base - received

	// 黄金封顶与丢量判断：受「黄金资源最大值」硬上限约束
	before := city.Gold
	goldMax := ezfyResMaxOf("gold")
	after := before + received
	if after > goldMax {
		after = goldMax
	}
	lost := (before + received) - after
	city.Gold = after
	h.saveCityRes(&city)

	msg := fmt.Sprintf("向系统出售 %s×%d 成功，获得 %d 黄金（手续费已扣 %d）",
		ezfyResNames[req.EsType], req.EsCount, received, fee)
	if lost > 0 {
		msg += fmt.Sprintf("；因超过黄金上限丢失 %d", lost)
	}
	resp.OK(c, gin.H{"msg": msg, "received_gold": received, "fee": fee,
		"gold_lost": lost > 0, "lost_gold": lost})
}

// ezfyTreasureSellPrice 宝物出售给系统的单价（黄金/件）。
//
// ★ 2026-10-08 「采集的宝物可以卖给系统，按品质 10 万~50W 不等，收 10% 手续费」
//
//	—— 用户确认：宝物当前没有品质差异（采集宝物全 Tier1），统一按 20 万黄金/件出售，收 10% 手续费。
const ezfyTreasureSellPrice = 200000

// ExchangeTreasureSell POST /games/ezfy/exchange/treasure-sell {cfg_id, count}
//
// ★ 2026-10-08 「宝物卖给系统」：
//   - 只卖**未穿戴**的采集宝物（ezfy_equipment.officer_id = 0，且 cfg_id 属采集宝物）;
//   - 统一 20 万黄金/件，玩家实得再扣 10% 手续费（ezfySysSellFeePct）;
//   - 删除对应宝物记录，黄金入账并受「黄金资源最大值」硬上限约束（超出丢量提示）。
func (h *EzfyHandler) ExchangeTreasureSell(c *gin.Context) {
	uid := middleware.GetUID(c)
	// 出售串行化，防连点重复扣宝物 / 重复得黄金
	mu := ezfyExchangeLock(uid)
	mu.Lock()
	defer mu.Unlock()
	var req struct {
		CfgId int `json:"cfg_id"`
		Count int `json:"count"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if req.CfgId <= 0 || req.Count <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	// 校验是采集宝物（黑铁幽灵等普通装备不算可售宝物）
	cfge, ok := ezfyCfg.equipments[req.CfgId]
	if !ok || !ezfyCollectibleTreasureNames()[cfge.Name] {
		resp.ParamError(c, "非可出售宝物")
		return
	}
	// 校验未穿戴的持有量
	var own int64
	h.DB.Model(&model.EzfyEquipment{}).
		Where("user_id = ? AND officer_id = 0 AND cfg_id = ?", uid, req.CfgId).
		Count(&own)
	if own < int64(req.Count) {
		resp.ParamError(c, fmt.Sprintf("宝物不足(现有%d件)", own))
		return
	}
	// 删除 count 件未穿戴宝物
	if entity := h.DB.Where("user_id = ? AND officer_id = 0 AND cfg_id = ?", uid, req.CfgId).
		Limit(req.Count).Delete(&model.EzfyEquipment{}); entity.Error != nil {
		resp.ServerError(c, entity.Error)
		return
	}
	// 结算黄金：单价×数量 → 实得扣 10% 手续费
	base := int64(req.Count) * ezfyTreasureSellPrice
	received := base * (100 - ezfySysSellFeePct) / 100
	fee := base - received

	city := h.getOrCreateCity(uid)
	h.calcResource(&city)
	before := city.Gold
	goldMax := ezfyResMaxOf("gold")
	after := before + received
	if after > goldMax {
		after = goldMax
	}
	lost := (before + received) - after
	city.Gold = after
	h.saveCityRes(&city)
	// 出售改变了背包/宝物，玩家级缓存全局失效
	ezfyPageCacheDel(uid)

	msg := fmt.Sprintf("出售宝物 %s×%d 成功，获得 %d 黄金（手续费已扣 %d）",
		cfge.Name, req.Count, received, fee)
	if lost > 0 {
		msg += fmt.Sprintf("；因超过黄金上限丢失 %d", lost)
	}
	resp.OK(c, gin.H{"msg": msg, "received_gold": received, "fee": fee,
		"gold_lost": lost > 0, "lost_gold": lost})
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
	// ★ 2026-10-03 优化 /corps/members 到 1s 内：原来每个成员都单独查 2 次
	//   （ensureProfile + First(user)），成员一多就是 2N 次 DB 查询。
	//   改成按人批量拉 profile 和 user 各一次，N 再大也只有 2 次查询。
	userIDs := make([]uint, 0, len(members))
	for _, m := range members {
		userIDs = append(userIDs, m.UserId)
	}
	profMap := make(map[uint]model.EzfyProfile, len(members))
	if len(userIDs) > 0 {
		var profs []model.EzfyProfile
		h.DB.Select("user_id", "nickname", "prestige", "rank").Where("user_id IN ?", userIDs).Find(&profs)
		for i := range profs {
			profMap[profs[i].UserID] = profs[i]
		}
	}
	userMap := make(map[uint]model.User, len(members))
	if len(userIDs) > 0 {
		var users []model.User
		// 只取昵称：model.User 里 AvatarBase64 是 longtext，全员拉全量会很慢。
		h.home().Select("id", "nickname").Where("id IN ?", userIDs).Find(&users)
		for i := range users {
			userMap[users[i].ID] = users[i]
		}
	}
	for _, m := range members {
		p := profMap[m.UserId]
		u := userMap[m.UserId]
		// ★ 2026-09-25 「军团页展示个人军团积分」：每项带 points
		views = append(views, gin.H{"user_id": m.UserId, "name": ezfyNickOf(p, &u),
			"is_leader": m.IsLeader, "title": m.Title,
			"prestige": p.Prestige, "rank_name": ezfyRankNameAt(ezfyProfileRank(&p)),
			"points": m.Points})
	}
	// ★ 军团总积分（军团商城/军团页展示）
	corpsPoints := int64(0)
	var cp model.EzfyCorps
	if err := h.DB.First(&cp, mb.CorpsId).Error; err == nil {
		corpsPoints = cp.Points
	}
	resp.OK(c, gin.H{"members": views, "in_corps": true,
		"can_manage": canManage, "can_mail": canMail,
		"my_title": mb.Title, "is_leader": mb.IsLeader,
		"corps_points": corpsPoints})
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
		profLv := ezfyProfileRank(&prof)
		maxCity := ezfyRankCityMaxAt(profLv)
		if int(owned) >= maxCity {
			resp.ParamError(c, fmt.Sprintf("当前军衔「%s」最多只能拥有 %d 座城市（已有 %d 座），提升声望可解锁更多",
				ezfyRankNameAt(profLv), maxCity, owned))
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
			fmt.Sprintf("你被占领的城市[%s]已由敌方归还!\n民心已恢复。", city.Name), "", 0, city.ID)
		resp.OK(c, gin.H{"msg": "已归还"})
	default:
		resp.ParamError(c, "未知操作")
	}
}

// RansomCreate POST /city/ransom {city_id} —— 被占城市原主人发起赎城
//
// ★ 2026-10-07 赎城功能：发起即扣「赎城金额」钻石（押金），攻击者同意后金额归攻击者、城市返还，
//
//	拒绝/撤销押金退回。仅「占领中」(occupy status=1) 可赎；同一城市同时最多 1 条待处理请求。
func (h *EzfyHandler) RansomCreate(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		CityId int64 `json:"city_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	var city model.EzfyCity
	if err := h.DB.Where("id = ? AND user_id = ?", req.CityId, uid).First(&city).Error; err != nil {
		resp.ParamError(c, "城市不存在")
		return
	}
	// ★ 2026-10-07 修复「赎城能一直扣」：原实现是「查 pending → 扣钻 → 插入」三段分离，
	//   快速双击/连点/并发会同时通过查重（都看到 0 笔）→ 各扣一次钻石、各插入一条请求。
	//   现在整段放进一个事务，并先对**城市行加排他锁(FOR UPDATE)**：
	//   同城赎城被强制串行，后到的请求等前者提交后必能看到新插入的 pending → 拒绝。
	//   （FOR UPDATE 是当前读、不建立快照，事务里第一次一致性读发生在锁拿到之后，
	//    所以后到事务能看到先行事务已提交的插入。）
	tx := h.DB.Begin()
	defer tx.Rollback()
	var occ model.EzfyOccupy
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", city.ID).First(&city).Error; err != nil {
		resp.ParamError(c, "城市不存在")
		return
	}
	if err := tx.Where("city_id = ? AND status = 1", city.ID).First(&occ).Error; err != nil {
		resp.ParamError(c, "该城市未被占领, 无需赎城")
		return
	}
	if occ.DefUserId != uid {
		resp.ParamError(c, "只有城市原主人才能发起赎城")
		return
	}
	var pending int64
	if err := tx.Model(&model.EzfyRansom{}).Where("city_id = ? AND status = 0", city.ID).Count(&pending).Error; err != nil {
		resp.ParamError(c, "查询赎城请求失败："+err.Error())
		return
	}
	if pending > 0 {
		resp.ParamError(c, "该城市已有一笔待处理的赎城请求")
		return
	}
	cost := ezfyRansomCost()
	prof := h.ensureProfile(uid)
	if prof.Diamond < cost {
		resp.ParamError(c, fmt.Sprintf("钻石不足: 赎城需要%d钻石, 当前余额%d", cost, prof.Diamond))
		return
	}
	// 原子扣减（WHERE diamond>=cost 兜底并发扣爆）
	if res := tx.Model(&model.EzfyProfile{}).Where("id = ? AND diamond >= ?", prof.ID, cost).
		Update("diamond", gorm.Expr("diamond - ?", cost)); res.RowsAffected != 1 {
		resp.ParamError(c, "扣钻石失败, 请重试")
		return
	}
	r := model.EzfyRansom{CityId: int64(city.ID), OccupyId: occ.ID, DefUserId: uid,
		AtkUserId: occ.AtkUserId, CityName: city.Name, X: city.X, Y: city.Y,
		Cost: cost, Status: 0}
	if err := tx.Create(&r).Error; err != nil {
		resp.ParamError(c, "创建赎城请求失败："+err.Error())
		return
	}
	if err := tx.Commit().Error; err != nil {
		resp.ParamError(c, "提交失败："+err.Error())
		return
	}
	h.logDiamond(uid, -cost, "赎城请求押金")
	h.addReport(occ.AtkUserId, 5, "赎城请求",
		fmt.Sprintf("原主人「%s」想花%d钻石赎回你占领的城市[%s](%d,%d)，可在[附属野地]的「赎回请求」中同意或拒绝。",
			prof.Nickname, cost, city.Name, city.X, city.Y))
	resp.OK(c, gin.H{"msg": fmt.Sprintf("已发起赎城请求, 已扣除%d钻石押金; 占领方同意后返还城市, 拒绝则自动退回。", cost)})
}

// RansomHandle POST /city/ransom/handle {ransom_id, op} —— 占领方处理赎城请求（op 1同意 2拒绝）
func (h *EzfyHandler) RansomHandle(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		RansomId uint `json:"ransom_id"`
		Op       int  `json:"op"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	var r model.EzfyRansom
	if err := h.DB.First(&r, req.RansomId).Error; err != nil {
		resp.ParamError(c, "赎城请求不存在")
		return
	}
	if r.AtkUserId != uid {
		resp.ParamError(c, "只有占领方才能处理该赎城请求")
		return
	}
	if r.Status != 0 {
		resp.ParamError(c, "该请求已处理")
		return
	}
	if req.Op == 1 {
		// 同意前重校验占领记录：已被建立/摧毁/归还(status≠1) → 押金退回、请求作废
		var occ model.EzfyOccupy
		if h.DB.Where("id = ? AND status = 1", r.OccupyId).First(&occ).Error != nil {
			h.refundRansom(&r, 3, "该城市已被占领方处理, 赎城请求作废, 押金退回")
			resp.ParamError(c, "该城市已被处理(建立/摧毁/归还), 赎城请求已作废并退回押金")
			return
		}
		// 同意：金额转给占领方 + 城市返还（同「放弃归还」口径：user_id 本仍属守方，补民心）
		atkProf := h.ensureProfile(uid)
		if err := h.DB.Model(&model.EzfyProfile{}).Where("id = ?", atkProf.ID).
			Update("diamond", atkProf.Diamond+r.Cost).Error; err != nil {
			resp.ParamError(c, "钻石发放失败："+err.Error())
			return
		}
		h.logDiamond(uid, r.Cost, "赎城收入")
		var ct model.EzfyCity
		if h.DB.First(&ct, r.CityId).Error == nil && ct.Feelings < 20 {
			h.DB.Model(&model.EzfyCity{}).Where("id = ?", r.CityId).Update("feelings", 20)
		}
		h.DB.Model(&model.EzfyOccupy{}).Where("id = ?", r.OccupyId).Update("status", 2)
		h.DB.Model(&model.EzfyRansom{}).Where("id = ?", r.ID).Update("status", 1)
		h.addReport(uid, 5, "赎城成功",
			fmt.Sprintf("你同意了赎城, 收到%d钻石, 城市[%s](%d,%d)已归还原主人。", r.Cost, r.CityName, r.X, r.Y))
		h.addReport(r.DefUserId, 5, "赎城成功",
			fmt.Sprintf("占领方已同意你的赎城, 你成功赎回城市[%s](%d,%d)! 民心已恢复。", r.CityName, r.X, r.Y))
		resp.OK(c, gin.H{"msg": fmt.Sprintf("已同意赎城, 收到%d钻石, 城市已归还原主人", r.Cost)})
		return
	}
	// 拒绝：押金退回原主人
	h.refundRansom(&r, 2, "占领方拒绝了赎城请求, 押金退回")
	h.addReport(uid, 5, "赎城拒绝",
		fmt.Sprintf("你拒绝了原主人赎回城市[%s](%d,%d)的请求。", r.CityName, r.X, r.Y))
	resp.OK(c, gin.H{"msg": "已拒绝该赎城请求, 原主人押金已退回"})
}

// RansomCancel POST /city/ransom/cancel {city_id} —— 原主人撤销待处理赎城请求（押金退回）
//
// ★ 撤销按 city_id 定位（守方视角只知道自己的城市，不知道请求 id；发起与撤销入参保持对称）。
// ★ 2026-10-07 撤销赎城也加事务+行锁：原实现「查 status=0 → 退押金 → 置 status」三段分离，
//
//	双击/并发会同时通过查询 → 押金双退。现在先锁请求行、再原子置终态、再退押金。
func (h *EzfyHandler) RansomCancel(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		CityId int64 `json:"city_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	tx := h.DB.Begin()
	defer tx.Rollback()
	var r model.EzfyRansom
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("city_id = ? AND def_user_id = ? AND status = 0", req.CityId, uid).
		First(&r).Error; err != nil {
		resp.ParamError(c, "该城市没有待处理的赎城请求")
		return
	}
	// 置终态（锁内原子）：并发的第二个撤销等锁后这里 RowsAffected=0 → 拒绝，押金不会双退
	if res := tx.Model(&model.EzfyRansom{}).Where("id = ? AND status = 0", r.ID).
		Update("status", 3); res.RowsAffected != 1 {
		resp.ParamError(c, "该请求已被处理")
		return
	}
	defProf := h.ensureProfile(r.DefUserId)
	if err := tx.Model(&model.EzfyProfile{}).Where("id = ?", defProf.ID).
		Update("diamond", gorm.Expr("diamond + ?", r.Cost)).Error; err != nil {
		resp.ParamError(c, "退回押金失败："+err.Error())
		return
	}
	if err := tx.Commit().Error; err != nil {
		resp.ParamError(c, "提交失败："+err.Error())
		return
	}
	h.logDiamond(r.DefUserId, r.Cost, "赎城请求撤销退回")
	h.addReport(r.AtkUserId, 5, "赎城请求撤销",
		fmt.Sprintf("原主人撤销了赎回城市[%s](%d,%d)的请求。", r.CityName, r.X, r.Y))
	resp.OK(c, gin.H{"msg": "已撤销赎城请求, 押金已退回"})
}

// refundRansom 退回赎城押金并通知原主人（请求置终态 newStatus；reason 用于钻石流水与通知文案）
func (h *EzfyHandler) refundRansom(r *model.EzfyRansom, newStatus int, reason string) {
	defProf := h.ensureProfile(r.DefUserId)
	h.DB.Model(&model.EzfyProfile{}).Where("id = ?", defProf.ID).
		Update("diamond", defProf.Diamond+r.Cost)
	h.logDiamond(r.DefUserId, r.Cost, reason)
	h.DB.Model(&model.EzfyRansom{}).Where("id = ?", r.ID).Update("status", newStatus)
	h.addReport(r.DefUserId, 5, "赎城押金退回",
		fmt.Sprintf("你为赎回城市[%s](%d,%d)支付的%d钻石押金已退回: %s", r.CityName, r.X, r.Y, r.Cost, reason))
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
//
// ★ 2026-10-04 性能（用户反馈「/city/wildfull 线上 2s+」）：懒结算改走 refreshCityRead
//
//	（跳过订单结算 3~4 条 RDS）；再加 3s 玩家级缓存，占领/放弃/采集等写操作统一失效。
func (h *EzfyHandler) WildlandFull(c *gin.Context) {
	uid := middleware.GetUID(c)
	if it, ok := ezfyPageCacheGet(uid, "wildfull"); ok {
		resp.OK(c, it)
		return
	}
	// ★ 2026-10-04 再优化（用户反馈「wildfull 3s+」）：第一波 档案+城市列表（1 RTT）定当前城，
	//   第二波 5 条查询 + 建筑列表并行（1 RTT），懒结算复用已取数据 → 串行 RTT 从 ~7 降到 ~2。
	_, city, cities := h.ezfyPageCity(uid)
	// ★ 2026-10-04 性能（用户反馈「wildfull 卡」）：独立查询并行（1 个 RTT）；
	//   被占城市归属玩家的游戏昵称改为一次 IN 批量查，消除原来「每行 ensureProfile」的 N+1。
	var (
		wildlands   []model.EzfyWildland
		gatherIds   []int64
		idleOrders  []model.EzfyOrder
		occupies    []model.EzfyOccupy
		ransoms     []model.EzfyRansom // ★ 2026-10-07 赎城：发给占领方的待处理赎回请求
		buildings   []model.EzfyCityBuilding
		trainQueues []model.EzfyTrainQueue
		techRows    []model.EzfyCityTech
		techs       map[int]int
		troops      map[int]int64
		boost       *model.EzfyCityEffect
		mayor       int
	)
	var wg sync.WaitGroup
	// ★ 2026-10-05 修复「wildfull 调用失败」：原 wg.Add(7) 但下面只有 6 个 goroutine
	//   （昵称查询早改成串行 IN 批量查），WaitGroup 永远等不到第 7 次 Done → 请求死锁挂死。
	// ★ 2026-10-05 性能：并行块再补 5 条（科技行/科技表/部队/增产令/市长加成），
	//   让下面的 calcResourceD 零额外查询（原来它自己又串行查了 6 遍）。
	wg.Add(12)
	go func() { defer wg.Done(); h.DB.Where("city_id = ?", city.ID).Find(&wildlands) }()
	go func() {
		defer wg.Done()
		h.DB.Where("atk_user_id = ? AND status = 0", uid).Order("id ASC").Find(&ransoms)
	}()
	go func() { defer wg.Done(); h.DB.Where("city_id IN ? AND status = 1", cityIdsOf(cities)).Find(&techRows) }()
	go func() { defer wg.Done(); techs = h.techMapOf(uid) }()
	go func() { defer wg.Done(); troops = h.troopMap(city.ID) }()
	go func() { defer wg.Done(); boost = h.ezfyLoadActiveBoost(city.ID) }()
	go func() { defer wg.Done(); mayor = h.mayorBonusPct(city.ID) }()
	go func() {
		defer wg.Done()
		h.DB.Model(&model.EzfyOrder{}).
			Where("user_id = ? AND status = 1 AND order_type = 7 AND arrive_time > 0", uid).
			Pluck("target_id", &gatherIds)
	}()
	go func() {
		defer wg.Done()
		h.DB.Where("user_id = ? AND status = 1 AND order_type = 7 AND arrive_time = 0", uid).Find(&idleOrders)
	}()
	go func() { defer wg.Done(); h.DB.Where("atk_city_id = ? AND status = 1", city.ID).Find(&occupies) }()
	go func() { defer wg.Done(); buildings = h.buildingList(city.ID) }()
	go func() {
		defer wg.Done()
		h.DB.Where("city_id = ? AND status = 0", city.ID).Order("start_time ASC").Find(&trainQueues)
	}()
	wg.Wait()
	// 懒结算复用已取数据（零额外查询）
	h.checkBuildingDone(&city, buildings)
	h.checkTechDoneRows(&city, techRows)
	h.collectTrainQueue(&city, trainQueues)
	h.calcResourceD(&city, &resCalcData{buildings: buildings, techs: techs, wilds: wildlands, troops: troops,
		boost: boost, boostDone: true, mayor: mayor})
	hallLevel := 0
	for _, b := range buildings {
		if b.BuildingId == 1 && b.Level > hallLevel {
			hallLevel = b.Level
		}
	}

	// ★ 2026-09-28 修复：附属野地页状态必须与 /view 同一套实时判定（常驻制下野地表 status 恒为 0），
	//   原来这里直接返回 w.Status + 不传 idle_order_id，导致「明明在采集却显示空闲」、
	//   「驻守空闲的野地不显示[开始采集]」（用户反馈 bug）。
	gathering := map[int64]bool{}
	for _, id := range gatherIds {
		gathering[id] = true
	}
	idleOrderByWild := map[int64]uint{}
	for _, o := range idleOrders {
		idleOrderByWild[o.TargetId] = o.ID
	}
	wildViews := []gin.H{}
	for _, w := range wildlands {
		sts := w.Status
		idleOrderId := uint(0)
		if gathering[int64(w.ID)] {
			sts = 1
		} else if oid, ok := idleOrderByWild[int64(w.ID)]; ok {
			sts = 0
			idleOrderId = oid
		}
		// ★ 必须带 terrain_name：前端 loadWilds() 会用这里的返回**整体覆盖** wildlands，
		//   之前漏了这个字段，导致「附属野地」页面的【地形】列永远是空的。
		wildViews = append(wildViews, gin.H{"id": w.ID, "x": w.X, "y": w.Y,
			"wild_type": w.WildType, "level": w.Level, "status": sts,
			"idle_order_id": idleOrderId,
			"terrain":       ezfyTerrainEx(w.X, w.Y),
			// ★ 2026-10-05 用户纠正：① 海洋野地→海底森林、岛屿→岛屿（岛屿仍是海野玩法，只是名字不同）；
			//   ② 表里有这一行 ⇒ 确定有野地 ⇒ knownWild=true（不看 level，避免 level=0 的海洋野地被显示成「海洋」）。
			"terrain_name": ezfyWildTerrainDisplayName(w.X, w.Y, true),
			"continent":    ezfyRegionName(w.X, w.Y)})
	}
	// 被占城市归属玩家 + 赎城请求发起人（原主人）的游戏昵称（原每行 ensureProfile 一次库 → 改一次 IN 查询）
	nick := map[uint]string{}
	ids := make([]uint, 0, len(occupies)+len(ransoms))
	addNickID := func(id uint) {
		for _, e := range ids {
			if e == id {
				return
			}
		}
		ids = append(ids, id)
	}
	for _, o := range occupies {
		addNickID(o.DefUserId)
	}
	for _, r := range ransoms {
		addNickID(r.DefUserId)
	}
	if len(ids) > 0 {
		var profs []model.EzfyProfile
		h.DB.Select("user_id", "nickname").Where("user_id IN ?", ids).Find(&profs)
		for _, p := range profs {
			nick[p.UserID] = p.Nickname
		}
	}
	occViews := []gin.H{}
	for _, o := range occupies {
		occViews = append(occViews, gin.H{"id": o.ID, "x": o.X, "y": o.Y,
			"city_id": o.CityId, "city_name": o.CityName, "def_user": nick[o.DefUserId]})
	}
	ransomViews := []gin.H{}
	for _, r := range ransoms {
		ransomViews = append(ransomViews, gin.H{"id": r.ID, "x": r.X, "y": r.Y,
			"city_id": r.CityId, "city_name": r.CityName, "def_user": nick[r.DefUserId], "cost": r.Cost})
	}
	data := gin.H{"city": city, "wildlands": wildViews, "occupies": occViews, "ransoms": ransomViews,
		"hall_level": hallLevel}
	ezfyPageCacheSet(uid, "wildfull", data)
	resp.OK(c, data)
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
		// ★ 2026-10-09 用户反馈「出征队列里等待中的看不出军队在干啥」→ 文案说清楚为什么在等
		return "等待中(排队等待交战)"
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
	case 1: // 野地: 展示名 + 等级(海洋野地→海底森林；岛屿→岛屿；纯海洋→海洋)
		// ★ 2026-10-05 统一走 ezfyWildTerrainDisplayName。target_type=1 就是野地 ⇒ knownWild=true。
		tn := ezfyWildTerrainDisplayName(o.TargetX, o.TargetY, true)
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
