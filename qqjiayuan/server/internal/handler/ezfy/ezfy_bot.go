// 人机（AI 玩家）模块 —— 身份层 + 调度 + 决策 + 执行（★ 2026-10-17）
//
// 设计见项目根 `人机模块方案.md`。一句话：人机 = **一个有状态的玩家代理**，
// 走和真人完全相同的业务接口（内部 HTTP 自调用 http://127.0.0.1:{port}/games/ezfy/*），
// 天然继承全部校验 / 懒结算 / 战报 / 免战 / 宣战 / 扣费原子性 —— 人机**不可能**做出玩家做不到的事。
//
// 三层：
//  1. 身份层：ezfyBotProvision()（家园 users + ezfy_account + ezfy_profile + ezfy_city，单事务写入 is_bot）
//  2. 调度层：BgTickBots() 每 60s 抢**租约锁**（多进程互斥）→ 取到点的 bot 分批处理
//  3. 决策层：botDecide() 优先级队列（保命 → 人口 → 建筑 → 兵 → 军事），**一轮只做一件事**（最像真人）
//
// ⚠️ 红线：
//   - 不走全图枚举找目标（坐标区间走索引）；
//   - 每轮最多 10 个 bot、每个 bot 每轮最多 1 个 HTTP 动作（1 核 1G 扛不住并发）；
//   - 人机不吃钻石、不充值、不参与充值榜；登录接口一律拒绝（见 EzfyLogin）。
package ezfy

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	mathrand "math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/authutil"
)

// ============ 配置常量 ============

const (
	// ezfyBotTickInterval tick 间隔：60s 一轮
	ezfyBotTickInterval = 60 * time.Second
	// ezfyBotBatchSize 每轮最多处理多少个（1 核 1G 的硬约束）
	ezfyBotBatchSize = 10
	// ezfyBotLeaseMS 租约有效期：90s（比 tick 长，进程崩了最迟 90s 后被接管）
	ezfyBotLeaseMS = int64(90 * 1000)
	// ezfyBotLeaseName 租约锁名
	ezfyBotLeaseName = "ezfy_bot_tick"
	// ezfyBotMaxCount 人机数量硬上限（防误配 1000 个把服务器打死）
	ezfyBotMaxCount = 50
	// ezfyBotHTTPTimeout 内部 HTTP 超时（懒结算可能稍慢）
	ezfyBotHTTPTimeout = 15 * time.Second
)

// ezfyBotBaseURL 内部自调用地址（router.Setup 里按实际端口设置）。
//
// ★ 走 127.0.0.1 不绕公网；端口取配置里的 server.port。
var ezfyBotBaseURL = "http://127.0.0.1:8080"

// SetEzfyBotBaseURL 由 router.Setup 调用，绑定当前实例端口。
func SetEzfyBotBaseURL(port int) {
	if port > 0 {
		ezfyBotBaseURL = fmt.Sprintf("http://127.0.0.1:%d", port)
	}
}

// ezfyBotChatPool 人机世界发言词库（口语化、像人话，★ 2026-10-17）。
//
//	[0] 交流玩法（12句） [1] 被打喊冤（8句） [2] 撩架挑衅（6句）。
//	每周每台人机各类句不重复；不同人机按 UserID 取不同起始偏移 → 发言内容互不相同。
var ezfyBotChatPool = [][]string{
	{
		"新手求教，前期是不是先堆农田比较稳？",
		"都说先升司令部出兵量大，我怎么觉得军工厂也不差啊",
		"火箭配掠夺是真的爽，一队过去对面就空了",
		"问下大佬们，集结令是等活动再烧还是平时就用了？",
		"研究先把指挥点上，出征上限高一截，亲测有效",
		"刚试了三骑兵带一火箭，掠夺效率比想象中高多了",
		"感觉六级矿不好打，还是先屯个十万兵再试",
		"建议新手先把稀矿厂升起来，后面科技全都要稀矿",
		"有人一起打野地刷声望吗，天天被高战蹲点",
		"资源还是得双开号攒，单开太慢了",
		"征兵真费油，火箭一口下去油仓就见底了",
		"昨天抢了个高战的城，油都搬回来一车一车的",
	},
	{
		"谁打我城了…刚出去采集回来就看见城在冒烟",
		"我就一默默发育的小号，大哥别盯着我呀",
		"守城兵全没了，谁这么狠啊",
		"我又被打飞了，才搬完家没两天",
		"无缘无故打我，我连谁打的都不知道",
		"还我兵啊，攒了一个星期的兵一晚上就没了",
		"能不能打之前说一声，我好把城里的兵撤走",
		"又是那个红色名字的，专挑半夜偷袭",
	},
	{
		"听说附近来了个高战？我的火箭已经饥渴难耐了",
		"谁再说这服没活人，出来干一架",
		"城里的兵都训完了，找个目标练练手",
		"有没有人一起组队，把对面那个盟打趴下",
		"天天打我盟友，这事咱们得算算",
		"晚上别睡太死，我随时可能上门",
	},
}

// ezfyBotWeekKey 本周序号（ISO 周，如 2026-W42）：跨周重置发言去重。
func ezfyBotWeekKey() string {
	y, w := time.Now().ISOWeek()
	return fmt.Sprintf("%d-W%02d", y, w)
}

// botChat 人机世界发言（★ 2026-10-17）：被打（最近 60 分钟有人打本 bot 城）→ 优先喊冤；
//
//	平时 → 8% 概率主动聊玩法/撩架（"时不时"说一句，别刷屏）。
//	每天句子不重复（chat_date/chat_used 记录），跨天清空重来。
func (h *EzfyHandler) botChat(b *model.EzfyBot, city *model.EzfyCity) (string, string, bool) {
	// ★★ 2026-10-19 用户规则「是否自动发言可在管理端编辑人机-基本信息里调」：
	//   ChatOn=0 → 完全闭嘴（被打喊冤也不说）。
	if b.ChatOn == 0 {
		return "", "", false
	}
	week := ezfyBotWeekKey()
	// ① 被打了？查最近 30 分钟内对「本 bot 任一城」的敌方侦察/掠夺/征服单（行军/驻守中）
	var ids []int64
	h.DB.Model(&model.EzfyCity{}).Where("user_id = ?", b.UserID).Pluck("id", &ids)
	victim := false
	if len(ids) > 0 {
		var hit int64
		h.DB.Model(&model.EzfyOrder{}).
			Where("order_type IN (1,2,3) AND target_type = 3 AND target_id IN ? AND status IN (0,1) AND user_id != ?",
				ids, b.UserID).Count(&hit)
		victim = hit > 0
	}
	// ② 主动聊天是低概率动作；被打喊冤优先（必说）
	if !victim && mathrand.Intn(100) >= 8 {
		return "", "", false
	}
	poolIdx := 0
	if victim {
		poolIdx = 1
	} else if mathrand.Intn(2) == 0 {
		poolIdx = 2
	}
	pool := ezfyBotChatPool[poolIdx]
	// ③ ★ 2026-10-17 一周不重复 + 每人机不同：
	//   每台人机取「自己的起始偏移」= UserID % 池长；发言按 (偏移 + 本周次数×步长) 逐个取，
	//   本周用过的相对序号记在 chat_used，跨周（ISO 周变化）清空重来。
	//   不同人机偏移不同 → 同一时刻说的句子不一样。
	offset := int(b.UserID) % len(pool)
	stride := 5 // 与池长互质，保证遍历完之前不撞
	if b.ChatDate != week {
		b.ChatUsed = "" // 跨周重置
	}
	usedSet := map[int]bool{}
	for _, s := range strings.Split(b.ChatUsed, ",") {
		if n, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
			usedSet[n] = true
		}
	}
	// 从自己的偏移开始按步长找本周第一个没说过、且属于所选语境的句
	content := ""
	take := -1
	for i := 0; i < len(pool); i++ {
		rel := (offset + i*stride) % len(pool)
		if usedSet[rel] {
			continue
		}
		take = rel
		content = pool[rel]
		break
	}
	if content == "" {
		return "", "", false // 本语境的句子这周说完了（被打时也会暂时没得说）
	}
	code, msg := h.botCall(b.UserID, "", "POST", "/games/ezfy/chat",
		map[string]interface{}{"channel": 1, "content": content})
	if code != 0 {
		return "世界发言", "发言失败: " + msg, false
	}
	seg := strconv.Itoa(take)
	newUsed := seg
	if b.ChatUsed != "" {
		newUsed = b.ChatUsed + "," + seg
	}
	h.DB.Model(&model.EzfyBot{}).Where("id = ?", b.ID).
		Updates(map[string]interface{}{"chat_date": week, "chat_used": newUsed})
	h.ezfyBotLog(b.UserID, "世界发言", content, 1)
	return "世界发言", content, true
}

// botHeal 立即恢复伤兵/逃兵（★ 2026-10-17 用户规则「人机会立即恢复伤兵、逃兵」）。
//
// 走真人 `/games/ezfy/troops/recover` 接口：免费版立即回城、只扣黄金（不花钻石，符合人机红线）。
// 每轮只处理「有伤兵/逃兵的城」，逐城 all=1（伤兵 type=0、逃兵 type=1），
// 黄金不足/伤兵营空时接口会拒 → 正常跳过（这轮做别的）。恢复不占用人口窗口。
func (h *EzfyHandler) botHeal(b *model.EzfyBot) (string, string, bool) {
	var cities []model.EzfyCity
	h.DB.Where("user_id = ?", b.UserID).Find(&cities)
	if len(cities) == 0 {
		return "", "", false
	}
	// 找出有伤兵/逃兵的城（聚一次库，避免逐城 HTTP 白打）
	var ids []int64
	for _, ct := range cities {
		ids = append(ids, int64(ct.ID))
	}
	if len(ids) == 0 {
		return "", "", false
	}
	var wTypes []int
	h.DB.Model(&model.EzfyWounded{}).Where("city_id IN ? AND count > 0", ids).
		Distinct("type").Pluck("type", &wTypes)
	if len(wTypes) == 0 {
		return "", "", false
	}
	// 逐城恢复（伤兵 0 / 逃兵 1）；同一城两种都恢复
	for _, ct := range cities {
		for _, wt := range wTypes {
			code, msg := h.botCall(b.UserID, "", "POST", "/games/ezfy/troops/recover",
				map[string]interface{}{"city_id": int64(ct.ID), "type": wt, "all": true})
			if code == 0 {
				label := "伤兵"
				if wt == 1 {
					label = "逃兵"
				}
				return "恢复" + label, fmt.Sprintf("%s 全部恢复", label), true
			}
			_ = msg // 黄金不足/空营 → 跳过，本轮做别的
		}
	}
	return "", "", false
}

// ezfyBotNickPool 昵称词库（军事风格，像真人 ID 而不是 NPC 编号）。
var ezfyBotNickPool = []string{
	"钢铁洪流", "沙漠之狐", "北境守望", "铁血孤鹰", "雪原狼", "暗夜尖刀",
	"远东之盾", "雷霆突击", "静默猎手", "赤色黎明", "黑潮", "孤城落日",
	"要塞之心", "银色弹壳", "冷锋", "焦土", "破晓", "远征军",
	"战地钟声", "铁幕", "长空猎鹰", "深蓝", "荒原旅人", "无名高地",
	"雪线之上", "燃烧海岸", "钢与火", "沉默的炮群", "夜航", "零号阵地",
}

// 难度档参数：发育节奏 / 兵力目标 / 攻击性
//
//	grow：两次行动间隔（分钟，含抖动）
//	troopTarget：主力兵力目标（决定它打谁）
//	aggression：0~100，越高越主动宣战、越敢打强目标
type ezfyBotLevelCfg struct {
	GrowMin     int
	GrowMax     int
	TroopTarget int64
	Aggression  int
}

func ezfyBotLevelOf(level int) ezfyBotLevelCfg {
	switch level {
	case 1: // 新手陪练：慢、防守型
		return ezfyBotLevelCfg{GrowMin: 8, GrowMax: 16, TroopTarget: 800, Aggression: 15}
	case 3: // 精英：快、侵略性高
		return ezfyBotLevelCfg{GrowMin: 3, GrowMax: 7, TroopTarget: 20000, Aggression: 80}
	default: // 普通
		return ezfyBotLevelCfg{GrowMin: 5, GrowMax: 11, TroopTarget: 5000, Aggression: 45}
	}
}

// ezfyBotBuildOrder 发育建筑优先级（按顺序找第一个「能建/能升」的）
//
//	农田 → 炼钢 → 石油 → 稀矿 → 市政厅 → 民居 → 军工厂 → 仓库 → 科研 → 军校 → 参谋部 → 司令部
//
// ★ 2026-10-17 实测修正：原来把市政厅/军工厂排在最前，人机升完这两样资源就见底、
//
//	而资源田一个没建（钢/油/稀产量为 0）→ 永远卡在「休整」。先建资源田才有产出。
var ezfyBotBuildOrder = []int{3, 4, 5, 6, 1, 2, 14, 12, 8, 9, 10, 13}

// ============ 身份层 ============

// ezfyBotProvision 创建一个人机账号（**唯一写入点**：三处 is_bot 一起落）。
//
// 事务：家园库 users + 二战库 ezfy_account/ezfy_profile/ezfy_city/ezfy_bot。
// ⚠️ 跨库（家园 qq_jiayuan / 二战 qq_ezzt）**不能同一个事务** → 分两段，失败时尽力回滚。
//
//	家园行先建（拿到自增 ID 才能当 user_id），二战段失败就把家园行删掉（物理删）。
func (h *EzfyHandler) ezfyBotProvision(level int, boost bool) (uint, error) {
	if level < 1 || level > 3 {
		level = 2
	}
	// ★ 2026-10-17 ensureBotBuilding 里要用 ezfyCfg.building() 查上限，先确保配置已加载
	h.cfgs()
	// 数量上限
	var cnt int64
	h.DB.Model(&model.EzfyBot{}).Count(&cnt)
	if cnt >= ezfyBotMaxCount {
		return 0, fmt.Errorf("人机数量已达上限 %d", ezfyBotMaxCount)
	}

	nick := h.ezfyBotPickNick()
	// 随机不可登录密码（is_bot 拦截在登录入口；这里再加一道「密码没人知道」）
	rawPwd := make([]byte, 16)
	if _, err := rand.Read(rawPwd); err != nil {
		return 0, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(hex.EncodeToString(rawPwd)), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	// ① 家园库账号（is_bot=1 = 账号级权威源）
	u := model.User{
		Nickname: nick,
		Password: string(hash),
		Gender:   1,
		Level:    1,
		Color:    "#004299",
		Config:   "10,1200,1500,1200,0",
		IsBot:    1,
	}
	if err := h.home().Create(&u).Error; err != nil {
		return 0, fmt.Errorf("建家园账号失败: %w", err)
	}
	h.home().Model(&model.User{}).Where("id = ?", u.ID).
		Update("username", strconv.FormatUint(uint64(u.ID), 10))
	uid := u.ID

	// ② 二战侧账号/档案/城（任一步失败 → 删掉家园行，避免留下半个人）
	rollback := func(cause error) (uint, error) {
		h.home().Exec("DELETE FROM users WHERE id = ?", uid)
		h.DB.Exec("DELETE FROM ezfy_account WHERE game_id = ?", uid)
		h.DB.Exec("DELETE FROM ezfy_profile WHERE user_id = ?", uid)
		h.DB.Exec("DELETE FROM ezfy_city WHERE user_id = ?", uid)
		h.DB.Exec("DELETE FROM ezfy_bot WHERE user_id = ?", uid)
		return 0, cause
	}
	if err := h.DB.Create(&model.EzfyAccount{
		GameID: uid, Password: string(hash), Nickname: nick, IsBot: 1,
	}).Error; err != nil {
		return rollback(fmt.Errorf("建二战账号失败: %w", err))
	}
	// 阵营随机（和真人一样有阵营，战报/地图配色才有区别）
	camp := 1
	if mathrand.Intn(2) == 0 {
		camp = 2
	}
	// ★ 2026-10-18 军衔默认中尉（rank 9，用户规则）+ **声望同步为该军衔门槛值**：
	//   此前 Prestige 固定 0 → 管理端显示「军衔中尉、声望 30」不一致；现在军衔与声望对应。
	botRank := ezfyRankAt(9)
	if err := h.DB.Create(&model.EzfyProfile{
		UserID: uid, GameUID: int64(uid), Nickname: nick,
		Prestige: botRank.NeedPrestige, Camp: camp, Rank: botRank.ID, IsBot: 1,
	}).Error; err != nil {
		return rollback(fmt.Errorf("建二战档案失败: %w", err))
	}
	// 建主城（复用玩家首次进游戏的同一套初始态：1 级市政厅/民居/农田 + 5000 资源）
	city := h.createMainCity(uid)
	if city.ID == 0 {
		return rollback(fmt.Errorf("建城失败"))
	}
	h.DB.Model(&model.EzfyCity{}).Where("id = ?", city.ID).Update("is_bot", 1)
	// ★ 2026-10-17 人机默认发育底盘：市政厅/农田/炼钢厂/石油基地/稀矿厂/军工厂/参谋部/司令部
	//   初始直接 9 级（司令部上限 12，其余 10），避免新人生机从 1 级慢慢爬、长期停在「休整/资源不足」。
	//   在 createMainCity 给的 1 级基础上补齐：已建的直接提到 9，未建的补建到 9。
	for _, bid := range ezfyBotDefaultBuildings {
		h.ensureBotBuilding(city.ID, bid, 9)
	}
	// ★ 2026-10-17 人机默认科技全 9 级（用户规则：新增人机默认科技都是 9 级）
	//   走 ezfy_user_tech（账号级科技，唯一键 uk_user_tech），与真人科技升级同一存储。
	for tid := range ezfyCfg.techs {
		h.ensureBotTech(uid, tid, 9)
	}
	// ★ 2026-10-17 资源默认 2000 万各项目（用户规则：新增人机 4 项资源各 2000 万），
	//   无条件生效（此前只有勾选「加速开局」才给 50 万，太寒酸；现在默认就是富开局）。
	h.DB.Model(&model.EzfyCity{}).Where("id = ?", city.ID).Updates(map[string]interface{}{
		"food": 20000000, "steel": 20000000, "oil": 20000000, "rare": 20000000,
	})
	// ★ 2026-10-17 人机默认人口 50 万（用户规则「默认人口 50 万吧，够用了」）：
	//   侦察显示当前人口 50 万；征兵线、出征人口承载都以它为基准，开局就能撑起大军团出征。
	//   （createMainCity 初始 100 上限太寒酸，人机懒得慢慢召集人口。）
	h.DB.Model(&model.EzfyCity{}).Where("id = ?", city.ID).Updates(map[string]interface{}{
		"pop": 500000, "pop_max": 500000,
	})
	// ★ 2026-10-17 人机创建默认送 2 个五星普通军官（军事/学识高、随机）：
	//   出生就有军官可用（出征带官、城守有人当），后续军官靠人机自己招。
	h.seedBotOfficers(uid, city.ID)
	// ★ 2026-10-17 新增人机按难度给初始兵力：侦察机 10/20/30 万、火箭 1/2/3 亿（有兵才敢宣战）
	scoutN, rocketN := int64(100000), int64(100000000)
	switch level {
	case 1:
		scoutN, rocketN = 100000, 100000000
	case 2:
		scoutN, rocketN = 200000, 200000000
	case 3:
		scoutN, rocketN = 300000, 300000000
	}
	h.ezfyBotSeedTroops(city.ID, scoutN, rocketN)
	// ★ 2026-10-17 人机默认持有 5000 个集结令（每个 +10 万出征上限，大兵力一样能派出去）
	h.addItem(uid, ezfyGatherItemID(), 5000)

	// ③ 人机档案：立刻可行动（首轮 tick 就会开始发育）
	now := time.Now().UnixMilli()
	if err := h.DB.Create(&model.EzfyBot{
		UserID: uid, Level: level, Enabled: 1,
		Persona:   ezfyBotPersona(level),
		NextActAt: now + int64(mathrand.Intn(120))*1000,
	}).Error; err != nil {
		return rollback(fmt.Errorf("建人机档案失败: %w", err))
	}
	h.ezfyBotLog(uid, "provision", fmt.Sprintf("生成人机 难度%d 昵称=%s", level, nick), 1)
	return uid, nil
}

// ezfyBotPickNick 从词库随机取一个未被占用的昵称（冲突加数字后缀）。
func (h *EzfyHandler) ezfyBotPickNick() string {
	pool := ezfyBotNickPool
	for i := 0; i < 60; i++ {
		name := pool[mathrand.Intn(len(pool))]
		if i >= 20 {
			name = name + strconv.Itoa(mathrand.Intn(90)+10)
		}
		var n int64
		h.home().Model(&model.User{}).Where("nickname = ?", name).Count(&n)
		if n == 0 {
			return name
		}
	}
	return "钢铁洪流" + strconv.FormatInt(time.Now().UnixNano()%100000, 10)
}

// ezfyBotEnsuredUniqueNick 返回一个未被占用的昵称（冲突自动补数字后缀）——生成时自定义马甲用。
func (h *EzfyHandler) ezfyBotEnsuredUniqueNick(name string) string {
	for i := 0; i < 60; i++ {
		cand := name
		if i > 0 {
			cand = name + strconv.Itoa(i+1)
		}
		var n int64
		h.home().Model(&model.User{}).Where("nickname = ?", cand).Count(&n)
		if n == 0 {
			return cand
		}
	}
	return name + "-" + strconv.FormatInt(time.Now().UnixMilli()%100000, 10)
}

// ezfyBotPersona 性格（目前只用于日志/展示，后续可影响节奏）。
func ezfyBotPersona(level int) string {
	switch level {
	case 1:
		return "保守"
	case 3:
		return "好战"
	}
	return "均衡"
}

// ezfyBotLog 写一条行为日志（管理端「行为日志」数据源）。
func (h *EzfyHandler) ezfyBotLog(uid uint, action, detail string, ok int) {
	h.DB.Create(&model.EzfyBotLog{BotUid: uid, Action: action, Detail: trimStr(detail, 240), Result: ok})
}

// ezfyBotSeedTroops 给人机城市塞初始兵力（★ 2026-10-17）。
//
// 直接建行（新起/新补的城市必然无兵，不会撞 uk_city_troop 唯一键）：
// 侦察机 troop_id=9、火箭 troop_id=8（种子兵种 ID）。
func (h *EzfyHandler) ezfyBotSeedTroops(cityID uint, scoutN, rocketN int64) {
	if scoutN > 0 {
		h.DB.Create(&model.EzfyCityTroop{CityId: int64(cityID), TroopId: 9, Count: scoutN})
	}
	if rocketN > 0 {
		h.DB.Create(&model.EzfyCityTroop{CityId: int64(cityID), TroopId: 8, Count: rocketN})
	}
}

// ============ 存量人机补一批集结令 ============

// ezfyBotGatherMark 存量人机「补 5000 集结令」的数据库标记（借用 ezfy_bot_lock 的 name 主键）。
const ezfyBotGatherMark = "ezfy_bot_gather_v1"

// ezfyBotGatherOnce 进程内只补一次（防同进程并发/重复跑）。
var ezfyBotGatherOnce sync.Once

// ezfyBotGatherBackfill 存量人机一次性补足 5000 集结令。
//
// ★ 2026-10-17 ezfyBotProvision 只负责**新造的人机**；线上已存在的人机走这里补齐，
//
//	否则它们背包里没有集结令、出征永远不烧（列表「集结令」列会一直是 0）。
//
// ⚠️ 为什么还要 DB 标记而不用 sync.Once 就够：sync.Once 每次重启都会重跑。
//
//	若每次重启都把「人机烧掉/管理端扣掉」的集结令补回 5000，管理端手动扣减会被悄悄吞掉。
//	→ 补完落一条标记行，之后任何一次重启都不再重复补。
func (h *EzfyHandler) ezfyBotGatherBackfill() {
	cid := ezfyGatherItemID()
	if cid <= 0 {
		return // 配置没就绪/没有集结令道具，跳过（下次 cfgs 再试也会被 Once 挡住——配置一般不会倒着变，可接受）
	}
	var marked int64
	h.DB.Model(&model.EzfyBotLock{}).Where("name = ?", ezfyBotGatherMark).Count(&marked)
	if marked > 0 {
		return
	}
	var uids []uint
	h.DB.Model(&model.EzfyBot{}).Select("user_id").Find(&uids)
	for _, uid := range uids {
		var cur int
		h.DB.Model(&model.EzfyItem{}).Where("user_id = ? AND cfg_id = ?", uid, cid).
			Select("count").Scan(&cur)
		if cur >= 5000 {
			continue // 已有 ≥5000（管理端发过了）→ 不动
		}
		if cur == 0 {
			h.addItem(uid, cid, 5000) // 背包里没有 → 直接建行
		} else {
			// 有但 < 5000（调号/发少了）→ 补到 5000
			h.DB.Exec("UPDATE ezfy_item SET count = 5000 WHERE user_id = ? AND cfg_id = ?", uid, cid)
		}
	}
	h.DB.Exec("INSERT IGNORE INTO ezfy_bot_lock (name, owner, expire_at, updated_at) VALUES (?, 'gather-backfill', 1, NOW())",
		ezfyBotGatherMark)
}

// ============ 调度层：租约锁 + tick ============

// ezfyBotOwner 当前进程标识（host:pid），用于租约锁归属判断。
func ezfyBotOwner() string {
	host, _ := os.Hostname()
	return fmt.Sprintf("%s:%d", host, os.Getpid())
}

// botAcquireLease 抢租约：带条件的原子 UPDATE（只有已过期/空持有者才抢得到）。
func (h *EzfyHandler) botAcquireLease(name, owner string, now, ttlMS int64) bool {
	// 锁行不存在时先补一行（幂等）
	h.DB.Exec("INSERT IGNORE INTO ezfy_bot_lock (name, owner, expire_at, updated_at) VALUES (?, '', 0, NOW())", name)
	res := h.DB.Exec(
		"UPDATE ezfy_bot_lock SET owner = ?, expire_at = ?, updated_at = NOW() "+
			"WHERE name = ? AND expire_at < ?", owner, now+ttlMS, name, now)
	return res.Error == nil && res.RowsAffected > 0
}

// botReleaseLease 释放租约（把到期时间归零，别的进程可立即接管）。
func (h *EzfyHandler) botReleaseLease(name, owner string) {
	h.DB.Exec("UPDATE ezfy_bot_lock SET expire_at = 0 WHERE name = ? AND owner = ?", name, owner)
}

// BgTickBots 人机主循环（常驻 goroutine，由 router.Setup 启动）。
//
// 启动先等 45 秒：让全量 seed / 建表 / 配置缓存先跑完（人机 tick 依赖这些）。
func (h *EzfyHandler) BgTickBots() {
	time.Sleep(45 * time.Second)
	for range time.Tick(ezfyBotTickInterval) {
		h.botTickOnce()
	}
}

// botTickOnce 一轮：抢锁 → 取到点的 bot（分批）→ 逐个决策 → 释放锁。
func (h *EzfyHandler) botTickOnce() {
	now := time.Now().UnixMilli()
	owner := ezfyBotOwner()
	if !h.botAcquireLease(ezfyBotLeaseName, owner, now, ezfyBotLeaseMS) {
		return // 别的进程在跑
	}
	defer h.botReleaseLease(ezfyBotLeaseName, owner)

	var bots []model.EzfyBot
	h.DB.Where("enabled = 1 AND next_act_at <= ?", now).
		Order("next_act_at ASC").Limit(ezfyBotBatchSize).Find(&bots)
	for _, b := range bots {
		h.botTickOne(b)
	}
}

// botTickOne 单个 bot 的一次行动：懒结算 → 决策 → 执行一个动作 → 写回下次时间。
func (h *EzfyHandler) botTickOne(b model.EzfyBot) {
	h.cfgs()
	uid := b.UserID
	cfg := ezfyBotLevelOf(b.Level)

	// ① 懒结算（等价于「玩家上线」）：结算在途订单 + 逐城产出/建筑完工
	h.processOrders(uid)
	city := h.getOrCreateCity(uid)
	if city.ID == 0 {
		h.botReschedule(b, cfg, "无城", "无法获取城市", false)
		return
	}
	h.refreshCity(uid, &city)
	// refreshCity 之后重新读一次，保证资源/人口是最新的
	h.DB.First(&city, city.ID)

	// ② 决策（一轮只做一件事）
	action, detail, ok := h.botDecide(&b, &city, cfg)

	// ③ 写日志 + 排下次行动（失败也排，避免卡死；但间隔更短，方便重试）
	h.ezfyBotLog(uid, action, detail, boolInt(ok))
	h.botReschedule(b, cfg, action, detail, ok)
}

// botReschedule 写回 next_act_at（随机抖动，模拟真人节奏）。
//
// ★ 2026-10-18 攻击频率修复：有活跃攻击计划（待宣战/待生效/可出击/冷却）时，
//
//	按「1 小时 / 频率」的节奏排下次行动 —— 让宣战 → 生效 → 出击链路按配置的
//	atk_per_hour 推进，而不是被 GrowMin~GrowMax（5~11 分钟）的发育节奏压住，
//	导致实际出征次数远低于配置（线上表现：频率配置 8 次/h，实际一天只有几次）。
func (h *EzfyHandler) botReschedule(b model.EzfyBot, cfg ezfyBotLevelCfg, action, detail string, ok bool) {
	gap := cfg.GrowMin + mathrand.Intn(cfg.GrowMax-cfg.GrowMin+1)
	if !ok {
		gap = 2 + mathrand.Intn(3) // 失败/跳过：2~4 分钟后重试
	} else if rate := ezfyBotAtkPerHourOf(b.Level, b.AtkPerHour); rate > 0 {
		// 成功行动后：若手头还有活跃攻击计划 → 下次行动按攻击间隔来
		var plan int64
		h.DB.Model(&model.EzfyBotTarget{}).
			Where("bot_uid = ? AND status IN (0,1,2,3)", b.UserID).Count(&plan)
		if plan > 0 {
			if g := 3600000 / int64(rate) / 60000; g > 2 {
				gap = int(g)
			} else {
				gap = 2 // 至少 2 分钟，避免宣战未生效时空转
			}
		}
	}
	now := time.Now().UnixMilli()
	h.DB.Model(&model.EzfyBot{}).Where("id = ?", b.ID).Updates(map[string]interface{}{
		"next_act_at": now + int64(gap)*60000,
		"last_act_at": now,
		"last_action": trimStr(action, 28),
	})
}

// ============ 执行器：内部 HTTP 自调用 ============

// botCall 以人机身份调用自己的业务接口。
//
// ★ 为什么走 HTTP 而不是直接调 handler：复用 100% 的校验/结算/战报/防抖逻辑，
//
//	人机不可能做出玩家做不到的事（方案 4.3）。
//
// ⚠️ path 传的是**业务路径**（如 `/games/ezfy/build`），内部统一补 `/api` 前缀 ——
// 全部路由都挂在 `api := r.Group("/api")` 下，漏了它就是 404「接口不存在」
// （第一次联调踩过：人机所有动作都静默失败、日志只剩「休整」）。
//
// 返回 (业务 code, msg)；网络错误时 code = -1。
func (h *EzfyHandler) botCall(uid uint, nick, method, path string, body interface{}) (int, string) {
	tok, err := authutil.GenerateEzfyToken(uid, nick, h.Secret, 1)
	if err != nil {
		return -1, "签发 token 失败: " + err.Error()
	}
	var rdr io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return -1, "序列化失败"
		}
		rdr = bytes.NewReader(buf)
	}
	req, err := http.NewRequest(method, ezfyBotBaseURL+"/api"+path, rdr)
	if err != nil {
		return -1, "构造请求失败"
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: ezfyBotHTTPTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return -1, "请求失败: " + err.Error()
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var out struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return -1, "响应解析失败"
	}
	return out.Code, out.Msg
}

// ============ 决策层 ============

// botDecide 优先级队列：第一个能做的就做，返回后本轮结束。
//
//  0. 保命：城内部队为 0 且有军工厂 → 造兵（被打光后重建，最优先）
//  1. 人口：空闲人口不足 → 召集
//  2. 建筑：市政厅 → 军工厂 → 民居 → 资源田 → 仓库 → 科研 → 军校 → 参谋部 → 司令部
//  3. 兵：主力兵力 < 目标 → 训练（步兵）
//  4. 军事：宣战计划 / 出击（受开关与节流约束）
//  5. 闲置：什么都不做（等资源）
func (h *EzfyHandler) botDecide(b *model.EzfyBot, city *model.EzfyCity, cfg ezfyBotLevelCfg) (string, string, bool) {
	var bs []model.EzfyCityBuilding
	h.DB.Where("city_id = ?", city.ID).Find(&bs)
	buildingLv := map[int]int{}
	busy := false
	for _, x := range bs {
		if x.Level > 0 && buildingLv[x.BuildingId] < x.Level {
			buildingLv[x.BuildingId] = x.Level
		}
		if x.Status != 0 {
			busy = true
		}
	}
	var troopTotal int64
	h.DB.Model(&model.EzfyCityTroop{}).Where("city_id = ?", city.ID).
		Select("COALESCE(SUM(count),0)").Scan(&troopTotal)

	// 0) 保命（★ 2026-10-17 加强）：被打光（城里一个兵没有）→ **立即征兵，无冷却**，
	//    每轮 tick 都先补兵，直到能出门反击/发育为止。
	// 0') 伤兵/逃兵优先恢复（★ 2026-10-17 用户规则「人机会立即恢复伤兵、逃兵」）：
	//    有伤兵就先用黄金恢复（走真人接口，立即回城，免费版只扣黄金）。
	if a, d, ok := h.botHeal(b); ok {
		return a, d, ok
	}
	if troopTotal == 0 && buildingLv[14] > 0 {
		if a, d, ok := h.botTrain(city, troopTotal, cfg, 1); ok {
			return a, d, ok
		}
	}

	// 0½) 攻击优先（★ 2026-10-18 攻击频率修复）：
	//   有「可出击」的战争计划且攻频额度可用 → **先打仗**。
	//   原来军事排在第 4 步（召集/建筑/征兵/军官之后），人机一旦在发育就把轮次占掉，
	//   实际出征次数永远到不了配置的 atk_per_hour —— 这里是「可出击」的硬前置，
	//   只在真正能打的时候插队；额度用光/无计划时照常走下面的发育步骤。
	if h.botReadyPlan(b.UserID) && h.botAtkQuotaOK(b) {
		if a, d, ok := h.botMilitary(b, city, troopTotal, cfg); ok {
			return a, d, ok
		} else if a != "" {
			return a, d, false
		}
	}

	// 0½) 世界聊天：**放在最后（闲置前）** —— 只在什么都做不了时才说话，
	//    绝不抢占征兵/军官/军事轮次（★ 2026-10-17 回归修复：原来放最前面导致人机每天光说话不打人）。
	//    这里保留已经说过的句去重；优先级交给 step 5。

	// 1) 人口：空闲人口低于人口上限的 40% → 召集
	//
	// ★ 2026-10-17 实测修正：召集会被**民居人口上限**影响（单次量现在 = 民居上限 × 10%，
	//   默认 10%，见 ezfyConvenePopGainOf）。民居等级太低时收益很小，所以要求民居先升到
	//   2 级（pop_max ≥ 300）才召集；否则跳过，让建筑步骤继续升民居。
	if city.PopMax >= 300 && int64(city.Pop) < int64(city.PopMax)*2/5 {
		cost := ezfyConveneFoodCostCfg()
		if city.Food >= cost {
			code, msg := h.botCall(b.UserID, "", "POST", "/games/ezfy/city/convene",
				map[string]interface{}{"city_id": int64(city.ID)})
			if code == 0 {
				return "召集人口", fmt.Sprintf("人口 %d/%d，召集成功", city.Pop, city.PopMax), true
			}
			return "召集人口", "召集失败: " + msg, false
		}
	}

	// 2) 建筑（有施工中的就等，一次只升一个）
	buildWhy := ""
	if !busy {
		if a, d, ok, why := h.botBuild(city, bs, buildingLv, b.Level); ok {
			return a, d, ok
		} else {
			buildWhy = why
		}
	}

	// 3) 兵（★ 2026-10-17 征兵线 = 50 万，低于就征；兵种：陆地火箭 / 海城航母）
	if a, d, ok := h.botTrain(city, troopTotal, cfg, 0); ok {
		return a, d, ok
	}

	// 3½) 军官养成（★ 2026-10-17 人机军官编辑修复）：自己招募军官 + 给可加点军官点军事，
	//   这样「出场带高军事军官」才有得带，而不是永远只有管理端手填的几个人。
	//   ⚠️ 只走业务接口（严格按真人规则扣费/占容量），绝不绕过校验凭空加属性。
	if a, d, ok := h.botOfficer(city, b.Level); ok {
		return a, d, ok
	}

	// 4) 军事（总开关 / 允许攻击玩家 / 攻频节流）
	if a, d, ok := h.botMilitary(b, city, troopTotal, cfg); ok {
		return a, d, ok
	} else if a != "" {
		// ★★ 2026-10-17 修复：botMilitary 失败也带原因（出击失败/宣战失败/冷却中…），
		//   此前 ok=false 直接丢弃落到「休整」，线上经常只看到休整、看不到「为什么不动手」。
		//   失败信息完整透传 → tick 主循环写进行为日志 + 2~4 分钟短间隔重试。
		return a, d, false
	}

	// 5) 世界聊天（★ 2026-10-17 移到闲置位：军事/征兵/军官都做完才说话，不抢轮次）
	if a, d, ok := h.botChat(b, city); ok {
		return a, d, ok
	}

	// 5) 闲置（把「建筑为什么没动」带出来，管理端行为日志能直接看到卡点）
	why := buildWhy
	if why == "" {
		why = "资源/人口暂不足"
	}
	return "休整", fmt.Sprintf("%s（粮%d 钢%d 油%d 稀%d）", why, city.Food, city.Steel, city.Oil, city.Rare), true
}

// botBuild 建造/升级建筑：按优先级找第一个「资源够 + 不在施工」的。
//
// 返回 (动作, 详情, 是否已执行, 没执行的原因)。
// 原因会带进「休整」日志，方便运营判断「为什么不动手」（资源不足 / 接口拒绝）。
func (h *EzfyHandler) botBuild(city *model.EzfyCity, bs []model.EzfyCityBuilding, lvOf map[int]int, botLevel int) (string, string, bool, string) {
	why := ""
	recOf := map[int]model.EzfyCityBuilding{}
	for _, x := range bs {
		if _, ok := recOf[x.BuildingId]; !ok || x.Level > recOf[x.BuildingId].Level {
			recOf[x.BuildingId] = x
		}
	}
	for _, bid := range ezfyBotBuildOrder {
		cfg := ezfyCfg.building(bid)
		if cfg == nil {
			continue
		}
		cur, has := recOf[bid]
		if !has || cur.Level <= 0 {
			// 未建造 → 建到 1 级
			cost := ezfyCfg.buildingLevel(bid, 1)
			if cost == nil || !ezfyBotCanPay(city, cost.Food, cost.Steel, cost.Oil, cost.Rare, cost.Gold) {
				if why == "" {
					why = "资源不足（" + cfg.Name + "）"
				}
				continue
			}
			code, msg := h.botCall(city.UserID, "", "POST", "/games/ezfy/build",
				map[string]interface{}{"city_id": int64(city.ID), "building_id": bid})
			if code == 0 {
				return "建造", fmt.Sprintf("建造 %s（1级）", cfg.Name), true, ""
			}
			// 建不了（数量上限/前置不满足）→ 继续看下一个，别死磕
			why = fmt.Sprintf("%s 建造被拒: %s", cfg.Name, msg)
			continue
		}
		// 已建造 → 升级（★ 9→10 需要建筑图纸，人机没有图纸 → 最高升到 9 级）
		target := cur.Level + 1
		if target > ezfyBotTargetLevel(bid, botLevel) || target > cfg.MaxLevel {
			continue
		}
		cost := ezfyCfg.buildingLevel(bid, target)
		if cost == nil || !ezfyBotCanPay(city, cost.Food, cost.Steel, cost.Oil, cost.Rare, cost.Gold) {
			if why == "" {
				why = "资源不足（" + cfg.Name + "）"
			}
			continue
		}
		code, msg := h.botCall(city.UserID, "", "POST", "/games/ezfy/building/upgrade",
			map[string]interface{}{"city_id": int64(city.ID), "record_id": int64(cur.ID)})
		if code == 0 {
			return "升级建筑", fmt.Sprintf("%s %d→%d 级", cfg.Name, cur.Level, target), true, ""
		}
		// ★ 2026-10-17 失败要**继续试下一个**，不能直接 return：
		//   「民居最多比市政厅高1级」这类规则会把民居卡住，return 就永远轮不到市政厅 → AI 整个卡死。
		why = fmt.Sprintf("%s 升级被拒: %s", cfg.Name, msg)
		continue
	}
	return "", "", false, why
}

// botTrain 造兵（★ 2026-10-17 改造）：
//
//	征兵线 50 万，低于就征。兵种按**当前局势动态选择**（用户规则「根据被打、出征别人优化造兵目标」）：
//	  - 近期被真人攻打（10 分钟内有敌方单）→ 造**守军步兵**(1)：Pop 1 便宜、军工厂 1 级就能训，
//	    被打光时能快速堆回来扛住下一波；
//	  - 有活跃攻击计划（status 0/1/2）→ 造**攻击兵种**：陆地火箭(8) / 海城航母(16)；
//	  - 无计划也无敌袭 → 默认攻击兵种（火箭/航母，随时准备出门打）。
//	force=1 保命（被打光立即补，无冷却）；普通调用 force=0（常态征兵，征不满说明被人口/资源卡住）。
func (h *EzfyHandler) botTrain(city *model.EzfyCity, troopTotal int64, cfg ezfyBotLevelCfg, force int) (string, string, bool) {
	// ★ 2026-10-18 征兵线改为管理端可配置（ezfy_cfg_limit.bot_train_threshold，默认 50 万）：
	//   低于该值就一直补；配 0/负值无意义时回落默认 50 万。
	const keepTroopsDef = 500000
	keepTroops := int64(keepTroopsDef)
	if ezfyCfg.ready() && ezfyCfg.limit.BotTrainThreshold > 0 {
		keepTroops = ezfyCfg.limit.BotTrainThreshold
	}
	if troopTotal >= keepTroops {
		return "", "", false
	}
	// 局势判定（★ 2026-10-17 动态兵种）
	tid := 8 // 默认攻击：火箭
	// ① 近期被打 → 守军步兵
	var hit int64
	now := time.Now().UnixMilli()
	h.DB.Model(&model.EzfyOrder{}).
		Where("order_type IN (1,2,3) AND target_type = 3 AND target_id = ? AND user_id != ? AND status IN (0,1) AND created_at_ms >= ?-600000",
			int64(city.ID), city.UserID, now).Count(&hit)
	if hit > 0 {
		tid = 1
	} else {
		// ② 无敌袭：有活跃攻击计划 → 攻击兵种（默认火箭）；海城可训则航母
		var active int64
		h.DB.Model(&model.EzfyBotTarget{}).
			Where("bot_uid = ? AND status IN (0,1,2)", city.UserID).Count(&active)
		if active > 0 && h.isSeaCity(city) {
			tid = 16
		}
	}
	t := ezfyCfg.troop(tid)
	// 不可训（航母要求 军工厂10/航海协会10）→ 退火箭
	if t == nil || tid == 16 {
		if tt := ezfyCfg.troop(8); tt != nil {
			t, tid = tt, 8
		}
	}
	if t == nil {
		return "", "", false
	}
	// 军工厂必须存在（兵工厂房等级）
	var fac model.EzfyCityBuilding
	facOK := h.DB.Where("city_id = ? AND building_id = 14 AND level > 0", city.ID).First(&fac).Error == nil
	// ★ 2026-10-18 修复「被打光后不知道征兵」：军工厂没有 / 等级不够训当前兵种
	//   （火箭要军工厂 6 级）时，原来直接 return false → 保命征兵被卡死，
	//   人机被打没了也永远不征兵。退到守军步兵（军工厂 1 级就能训）保底。
	if !facOK || (t.ID == 8 && fac.Level < 6) {
		if tt := ezfyCfg.troop(1); tt != nil {
			t, tid = tt, 1
		} else {
			return "", "", false
		}
	}
	gap := keepTroops - troopTotal
	if gap <= 0 {
		return "", "", false
	}
	// ★★ 2026-10-17 人机训练**不占人口**（用户规则：出征数量 30~40 万，火箭 Pop12 训 40 万
	//   要 480 万人口，50 万人口根本训不出来 → 出征只能带 4 万）。
	//   人机（IsBot=1）跳过人口门槛；真人照旧（训练校验 ezfy.go 也同步对 is_bot 跳过）。
	//   资源约束保留（人机也要留资源建设）。
	// 可负担数量（留 20% 资源做后续建设）
	maxByRes := int64(1 << 30)
	for _, pair := range [][2]int64{{t.Food, city.Food * 8 / 10}, {t.Steel, city.Steel * 8 / 10},
		{t.Oil, city.Oil * 8 / 10}, {t.Rare, city.Rare * 8 / 10}} {
		if pair[0] > 0 {
			if n := pair[1] / pair[0]; n < maxByRes {
				maxByRes = n
			}
		}
	}
	count := gap
	if count > maxByRes {
		count = maxByRes
	}
	if city.IsBot == 0 {
		if int64(city.Pop) < int64(t.Pop) {
			return "", "", false
		}
		// 人口上限（每个兵占 t.Pop 人口）
		if t.Pop > 0 {
			if n := int64(city.Pop) / int64(t.Pop); count > n {
				count = n
			}
		}
	}
	// 单批不超过 2000（训练队列有上限，且大批量排队太久）
	if count > 2000 {
		count = 2000
	}
	if count <= 0 {
		return "", "", false
	}
	code, msg := h.botCall(city.UserID, "", "POST", "/games/ezfy/troops/train",
		map[string]interface{}{"city_id": int64(city.ID), "troop_id": tid, "count": int(count)})
	if code == 0 {
		return "造兵", fmt.Sprintf("训练%s ×%d（当前 %d/%d）", t.Name, count, troopTotal, keepTroops), true
	}
	return "造兵", "训练失败: " + msg, false
}

// ezfyBotCanPay 资源是否够（本地判断，避免每次都发请求吃失败日志）。
func ezfyBotCanPay(city *model.EzfyCity, food, steel, oil, rare, gold int64) bool {
	return city.Food >= food && city.Steel >= steel && city.Oil >= oil &&
		city.Rare >= rare && city.Gold >= gold
}

// ============ 军事 AI ============

// botRevenge 被打反击（★ 2026-10-17）：近 10 分钟有真人侦察/掠夺/征服单命中本 bot 任一城 →
//
//	把攻方写成攻击计划（status=0 待宣战，下轮自动宣战 → 生效 → 征服）。
//	已有计划（任意状态，含冷却/放弃）→ 不再重复写（正常流程本来就会打它）。
//	返回 true = 已写入反击计划；false = 无可反击目标或已在计划中。
func (h *EzfyHandler) botRevenge(b *model.EzfyBot, city *model.EzfyCity) bool {
	var ids []int64
	h.DB.Model(&model.EzfyCity{}).Where("user_id = ?", b.UserID).Pluck("id", &ids)
	if len(ids) == 0 {
		return false
	}
	now := time.Now().UnixMilli()
	var hit struct {
		UID uint
	}
	// 近 10 分钟：侦察(1)/掠夺(2)/征服(3) 打本 bot 城、且不是 bot 自己（排除自己劫自己）
	h.DB.Raw(`
		SELECT o.user_id AS uid
		FROM ezfy_order o
		WHERE o.order_type IN (1,2,3) AND o.target_type = 3
		  AND o.target_id IN ? AND o.user_id != ?
		  AND o.status IN (0,1) AND o.created_at_ms >= ? - 600000
		ORDER BY o.id ASC LIMIT 1`, ids, b.UserID, now).Scan(&hit)
	if hit.UID == 0 {
		return false
	}
	// ★★ 2026-10-17 声望门槛修复：反击 = 主动把攻方纳入攻击目标，同样要过最低声望线。
	//   此前 botRevenge 直接写入计划不查声望 → 低于最低声望的玩家一打就被人机反打，
	//   等于把人机当免费复仇器（用户反馈「人机还是会把低于最低声望的纳入目标」根因之一）。
	if min := ezfyBotMinPrestige(); min > 0 {
		var ap model.EzfyProfile
		h.DB.Select("prestige").Where("user_id = ?", hit.UID).First(&ap)
		if ap.Prestige < min {
			h.ezfyBotLog(b.UserID, "反击", fmt.Sprintf("玩家 %d 攻击了我，但声望 %d 低于门槛 %d，不予反击", hit.UID, ap.Prestige, min), 0)
			return false
		}
	}
	// 已在计划中（任意状态）→ 不重复写（正常流程会打它；唯一键 (bot_uid,target_uid)）
	var dup int64
	h.DB.Model(&model.EzfyBotTarget{}).Where("bot_uid = ? AND target_uid = ?", b.UserID, hit.UID).Count(&dup)
	if dup > 0 {
		return false
	}
	var tc model.EzfyCity
	if err := h.DB.Where("user_id = ?", hit.UID).Order("id ASC").First(&tc).Error; err != nil {
		return false
	}
	h.DB.Create(&model.EzfyBotTarget{
		BotUid: b.UserID, TargetUid: hit.UID, TargetCityId: int64(tc.ID),
		Status: 0, Note: fmt.Sprintf("反击 %s(%d,%d)", tc.Name, tc.X, tc.Y),
	})
	h.DB.Model(&model.EzfyBot{}).Where("id = ?", b.ID).Update("next_act_at", now)
	h.ezfyBotLog(b.UserID, "反击", fmt.Sprintf("玩家 %d 攻击了我，已列入反击计划（下轮宣战征服）", hit.UID), 1)
	return true
}

// botMilitary 军事：计划表驱动的「宣战 → 等 24h → 出击」。
//
// 硬约束：打玩家城必须先宣战，宣战 24h 生效、48h 有效期 → 不能看到谁就打谁。
// 开关：ezfy_cfg_limit 的 bot_atk_on（管理端「人机管理 → 允许攻击玩家」）。
func (h *EzfyHandler) botMilitary(b *model.EzfyBot, city *model.EzfyCity, troopTotal int64, cfg ezfyBotLevelCfg) (string, string, bool) {
	if !ezfyBotAttackOn() {
		return "", "", false
	}
	// ★ 2026-10-17 被打反击：近 10 分钟有敌方侦察/掠夺/征服单打本 bot 的城 → 写反击计划（征服），
	//   下轮自动走「宣战 → 生效 → 征服」，反击不受攻频节流限制（计划表驱动，宣战前不打）。
	//   ⚠️ 真人规则：征服玩家城必须先宣战，反击也走同一链路（人机不可能绕过）。
	if h.botRevenge(b, city) {
		return "", "", false // 反击计划已写入，本轮让宣战流程接管
	}
	// 兵力不足目标的一半 → 不打仗，专心发育
	if troopTotal < cfg.TroopTarget/2 || troopTotal < 200 {
		return "", "", false
	}
	now := time.Now().UnixMilli()

	// ① 已有计划：推进状态
	var plans []model.EzfyBotTarget
	h.DB.Where("bot_uid = ? AND status IN (0,1,2,3)", b.UserID).Order("id ASC").Limit(3).Find(&plans)
	for _, p := range plans {
		switch p.Status {
		case 3: // 冷却重试：到点转回待宣战
			// ★ 2026-10-17 修复：冷却截止此前被写成 cool_until（botAttack 失败）与 effect_time（宣战失败）两处混用，
			//   case 只查 EffectTime → botAttack 失败写的 2h 冷却形同虚设，每轮都重试刷屏。
			//   统一起点判断：CoolUntil 未到 → 等；否则 EffectTime 未到 → 等；都到点 → 恢复宣战。
			if p.CoolUntil > 0 && now < p.CoolUntil {
				continue
			}
			if p.EffectTime > 0 && now < p.EffectTime {
				continue
			}
			h.DB.Model(&model.EzfyBotTarget{}).Where("id = ?", p.ID).Update("status", 0)
			return "重试宣战", fmt.Sprintf("目标 %d 冷却结束，重新宣战", p.TargetUid), true
		case 1: // 已宣战，等生效
			if now >= p.EffectTime {
				h.DB.Model(&model.EzfyBotTarget{}).Where("id = ?", p.ID).Update("status", 2)
				return "宣战生效", fmt.Sprintf("对目标 %d 的宣战已生效，准备出击", p.TargetUid), true
			}
			// ★★ 2026-10-19 宣战等待期：打玩家城要等宣战生效，但**附属野地不需要宣战** →
			//   等待期先征服该目标玩家的附属野地（用户规则「宣战有等待期，等待期可以打玩家附属野地」）。
			//   目标城太远 → 先自迁城贴近（与 case 2 出击前同一口径，迁城不受冷却）。
			var wtc model.EzfyCity
			if err := h.DB.First(&wtc, p.TargetCityId).Error; err == nil &&
				ezfyAbs(city.X-wtc.X)+ezfyAbs(city.Y-wtc.Y) > 10 {
				if a, d, ok := h.botRelocateNear(b, city, wtc.X, wtc.Y); ok {
					return a, d, ok
				}
			}
			if wt, ok := h.botPickWildTargetOf(p.TargetUid, city); ok {
				if !h.botAtkQuotaOK(b) {
					return "", "", false
				}
				return h.botAttackWild(b, city, wt)
			}
			continue
		case 2: // 可出击
			// ★★ 2026-10-18 攻击频率修复：出战前先判目标是否还有效 ——
			//   目标城已不存在 / 已被占走（ezfy_occupy.status=1）/ 战争已结束 → 计划作废换新目标，
			//   避免「宣战 24h 后目标早没了」还在反复对空城宣战、狂刷失败。
			var tcity model.EzfyCity
			if err := h.DB.First(&tcity, p.TargetCityId).Error; err != nil {
				h.DB.Model(&model.EzfyBotTarget{}).Where("id = ?", p.ID).Update("status", 4)
				return "放弃目标", fmt.Sprintf("目标 %d 城市已不存在", p.TargetUid), true
			}
			var occ int64
			h.DB.Model(&model.EzfyOccupy{}).Where("city_id = ? AND status = 1", p.TargetCityId).Count(&occ)
			if occ > 0 {
				h.DB.Model(&model.EzfyBotTarget{}).Where("id = ?", p.ID).Update("status", 4)
				return "放弃目标", fmt.Sprintf("目标 %s(%d,%d) 已被占领，换新目标", tcity.Name, tcity.X, tcity.Y), true
			}
			if !h.isAtWar(b.UserID, p.TargetUid) {
				h.DB.Model(&model.EzfyBotTarget{}).Where("id = ?", p.ID).Update("status", 4)
				return "放弃目标", fmt.Sprintf("与目标 %d 的战争已结束", p.TargetUid), true
			}
			// ★ 2026-10-17 出征时间 > 10 分钟 → 人机先自迁城贴近目标，再出击
			//   （默认行军速度换算：每格约 60 秒 → 10 分钟 ≈ 10 格；用曼哈顿距离估算，
			//    实际受兵种/科技速度影响，够用来做「太远就搬家」的判断。迁城不受冷却。）
			if dist := ezfyAbs(city.X-tcity.X) + ezfyAbs(city.Y-tcity.Y); dist > 10 {
				if a, d, ok := h.botRelocateNear(b, city, tcity.X, tcity.Y); ok {
					return a, d, ok
				}
			}
			// ★ 2026-10-17 攻频从「每日上限」改成「按小时频率」：宣战/生效照常推进，
			//   只在真正出击这里按「次/小时」节流（默认按难度 6/8/10，管理端可覆盖）。
			if !h.botAtkQuotaOK(b) {
				return "", "", false
			}
			return h.botAttack(b, city, p, troopTotal)
		case 0: // 待宣战 → 立即宣战
			code, msg := h.botCall(b.UserID, "", "POST", "/games/ezfy/war/declare",
				map[string]interface{}{"target_user_id": p.TargetUid, "city_id": int64(city.ID)})
			if code == 0 {
				eff := now + int64(ezfyWarDelayHours)*3600000
				h.DB.Model(&model.EzfyBotTarget{}).Where("id = ?", p.ID).
					Updates(map[string]interface{}{"status": 1, "effect_time": eff})
				return "宣战", fmt.Sprintf("向玩家 %d 宣战，%d 小时后生效", p.TargetUid, ezfyWarDelayHours), true
			}
			// ★ 2026-10-17 宣战失败分类（此前一律置 status=4 放弃，目标会莫名从列表消失）
			//   ① 目标已不需要宣战（可直打）→ 直接可出击
			if strings.Contains(msg, "不需要宣战") {
				h.DB.Model(&model.EzfyBotTarget{}).Where("id = ?", p.ID).Update("status", 2)
				return "宣战跳过", fmt.Sprintf("目标 %d 不需要宣战，直接出击", p.TargetUid), true
			}
			//   ② 已宣战/已处于战争 → 沿用现有战争生效时间，状态挂起等待
			if strings.Contains(msg, "已宣战") || strings.Contains(msg, "已处于战争") || strings.Contains(msg, "战争") {
				w := h.getWar(b.UserID, uint(p.TargetUid))
				if w != nil && w.Status == 2 {
					// 战争已生效 → 直接出击
					h.DB.Model(&model.EzfyBotTarget{}).Where("id = ?", p.ID).Update("status", 2)
					return "宣战已生效", fmt.Sprintf("与目标 %d 战争已生效，直接出击", p.TargetUid), true
				}
				if w != nil && w.EffectTime > 0 {
					h.DB.Model(&model.EzfyBotTarget{}).Where("id = ?", p.ID).
						Updates(map[string]interface{}{"status": 1, "effect_time": w.EffectTime})
					return "宣战挂起", fmt.Sprintf("与目标 %d 已有宣战，等待生效", p.TargetUid), true
				}
				// 没查到战争记录 → 先挂起 2 小时再试（冷却写在 cool_until，见 case 3）
				h.DB.Model(&model.EzfyBotTarget{}).Where("id = ?", p.ID).
					Updates(map[string]interface{}{"status": 3, "cool_until": now + 2*3600000})
				return "宣战失败", msg, false
			}
			//   ③ 同盟成员 → 真的打不了，放弃并留提示
			if strings.Contains(msg, "同盟") {
				h.DB.Model(&model.EzfyBotTarget{}).Where("id = ?", p.ID).Update("status", 4)
				return "宣战失败", fmt.Sprintf("目标 %d 为同盟成员，放弃该计划", p.TargetUid), false
			}
			//   ④ 其他未知失败 → 冷却 2 小时重试，不消失（冷却写在 cool_until，见 case 3）
			h.DB.Model(&model.EzfyBotTarget{}).Where("id = ?", p.ID).
				Updates(map[string]interface{}{"status": 3, "cool_until": now + 2*3600000})
			return "宣战失败", msg, false
		}
	}

	// ② 没有计划 → 先找**玩家城市**（坐标区间走索引，绝不遍历全图）
	if tgt, ok := h.botPickTarget(city, troopTotal, cfg); ok {
		if err := h.DB.Create(&model.EzfyBotTarget{
			BotUid: b.UserID, TargetUid: tgt.UserID, TargetCityId: int64(tgt.CityID),
			Status: 0, Note: fmt.Sprintf("%s(%d,%d) 兵力%d", tgt.Name, tgt.X, tgt.Y, tgt.Troops),
		}).Error; err != nil {
			return "", "", false // 唯一键冲突（已有计划）→ 下轮再说
		}
		return "锁定目标", fmt.Sprintf("锁定 %s(%d,%d) 兵力%d", tgt.Name, tgt.X, tgt.Y, tgt.Troops), true
	}
	// ③ 攻击不了玩家（周围没有可打的城市 / 都打不过 / 没有符合条件的目标）
	//   → 改打**玩家占领的附属野地**（用户规则 2026-10-19）：
	//     · 必须是**征服(3)**，不是掠夺 —— 直接把对方的野地归属打掉；
	//     · 胜利后立刻**放弃**（人机只摧毁，不占领，见 processArrive 的 bot 分支）；
	//     · 只打有主的野地（中立野地不打，抢不到任何玩家的东西）。
	//   原来是「25% 概率优先掠野」，会随机打中立野地，不符合用户口径。
	if wt, ok := h.botPickWildTarget(city); ok {
		if !h.botAtkQuotaOK(b) {
			return "", "", false
		}
		return h.botAttackWild(b, city, wt)
	}
	return "", "", false
}

// botReadyPlan 该人机是否有「可出击」的战争计划（★ 2026-10-18 攻击频率修复用）。
//
//	只数 status=2（宣战已生效、可以打）；有 → 本轮优先推进攻击。
func (h *EzfyHandler) botReadyPlan(botUID uint) bool {
	var n int64
	h.DB.Model(&model.EzfyBotTarget{}).Where("bot_uid = ? AND status = 2", botUID).Count(&n)
	return n > 0
}

// ============ 打玩家附属野地（★ 2026-10-18）============

// ezfyBotWildTarget 一块可打的玩家附属野地快照
type ezfyBotWildTarget struct {
	ID       int64
	X, Y     int
	Level    int
	WildType int
	Owner    string
}

// botPickWildTarget 在自己周围挑一块**真人玩家占领**的附属野地（征服后放弃，不写占领）。
//
// ★★ 2026-10-19 用户规则「人机不该攻击没有归属的附属野地」：
//
//	`JOIN ezfy_city` 保证 city_id 指向的城真实存在，再加 `c.user_id > 0` 排除
//	「野地行还在、归属城已无主」的历史脏数据 —— 这种行以前会被误判成「玩家野地」。
//
// 过滤（全部下推 SQL）：
//   - 野地已被占领（city_id > 0）且主人是真人（is_bot=0）、有 user_id、不是自己
//   - 坐标在自己 ±25 格内
//   - 未被采集（w.status != 1，且没有在途采集/驻守部队）
//   - 新手村周围的低级野地（level <= 2）不打，留给人机自己发育
func (h *EzfyHandler) botPickWildTarget(city *model.EzfyCity) (ezfyBotWildTarget, bool) {
	var r ezfyBotWildTarget
	err := h.DB.Raw(`
SELECT w.id AS id, w.x AS x, w.y AS y, w.level AS level, w.wild_type AS wild_type,
       COALESCE(p.nickname, '') AS owner
FROM ezfy_wildland w
JOIN ezfy_city c ON c.id = w.city_id
LEFT JOIN ezfy_profile p ON p.user_id = c.user_id
WHERE w.city_id > 0 AND c.is_bot = 0 AND c.user_id > 0 AND c.user_id <> ?
  AND c.id NOT IN (SELECT city_id FROM ezfy_occupy WHERE status = 1)
  AND w.x BETWEEN ? AND ? AND w.y BETWEEN ? AND ?
  AND w.status = 0 AND w.level > 2
  AND NOT EXISTS (SELECT 1 FROM ezfy_order o
                  WHERE o.target_id = w.id AND o.target_type = 1
                    AND o.status IN (0,1,2,5,6))
ORDER BY RAND() LIMIT 1`,
		city.UserID, city.X-25, city.X+25, city.Y-25, city.Y+25).Scan(&r).Error
	if err != nil || r.ID == 0 {
		return r, false
	}
	return r, true
}

// botPickWildTargetOf 打**指定玩家**的附属野地（★ 2026-10-19 宣战等待期用）。
//
// 与 botPickWildTarget 同一套过滤口径，只是归属限定为该玩家本人（`c.user_id = ?`）：
// 打玩家城要等宣战生效，但附属野地不需要宣战 —— 宣战等待期就先征服目标玩家的野地。
func (h *EzfyHandler) botPickWildTargetOf(userID uint, city *model.EzfyCity) (ezfyBotWildTarget, bool) {
	var r ezfyBotWildTarget
	err := h.DB.Raw(`
SELECT w.id AS id, w.x AS x, w.y AS y, w.level AS level, w.wild_type AS wild_type,
       COALESCE(p.nickname, '') AS owner
FROM ezfy_wildland w
JOIN ezfy_city c ON c.id = w.city_id
LEFT JOIN ezfy_profile p ON p.user_id = c.user_id
WHERE w.city_id > 0 AND c.is_bot = 0 AND c.user_id = ?
  AND c.id NOT IN (SELECT city_id FROM ezfy_occupy WHERE status = 1)
  AND w.x BETWEEN ? AND ? AND w.y BETWEEN ? AND ?
  AND w.status = 0 AND w.level > 2
  AND NOT EXISTS (SELECT 1 FROM ezfy_order o
                  WHERE o.target_id = w.id AND o.target_type = 1
                    AND o.status IN (0,1,2,5,6))
ORDER BY RAND() LIMIT 1`,
		userID, city.X-25, city.X+25, city.Y-25, city.Y+25).Scan(&r).Error
	if err != nil || r.ID == 0 {
		return r, false
	}
	return r, true
}

// botAttackWild 人机**征服**一块真人玩家的附属野地，胜利后立刻放弃（只摧毁归属，自己不占）。
//
// ★★ 2026-10-19 用户规则：
//   - 「人机若攻击不了玩家，那么应该攻击玩家的附属野地」→ 由 botMilitary 在
//     botPickTarget 失败后调用（不再 25% 随机掠野）；
//   - 「而且是征服，胜利后就放弃拿到的附属野地」→ order_type=3；
//     征服成功后由 processArrive 的 bot 分支调 ezfyClearWildlandAt 放弃；
//   - 「现在人机在攻击没有归属的附属野地不对」→ botPickWildTarget 只挑有主野地。
//
// 打野不走战争链路（野地目标不需要宣战）。
func (h *EzfyHandler) botAttackWild(b *model.EzfyBot, city *model.EzfyCity, wt ezfyBotWildTarget) (string, string, bool) {
	var troops []model.EzfyCityTroop
	h.DB.Where("city_id = ? AND count > 0", city.ID).Find(&troops)
	var total int64
	for _, t := range troops {
		total += int64(t.Count)
	}
	if total == 0 {
		return "", "", false
	}
	// 掠野出半兵（留一半守城/继续发育）
	want := ezfyBotAtkTroopOf(b.Level, b.AtkTroop) / 2
	if want > total {
		want = total
	}
	groups := make([]ezfyUnitGroup, 0, len(troops))
	for _, t := range troops {
		n := int64(t.Count) * want / total
		if n <= 0 {
			continue
		}
		groups = append(groups, ezfyUnitGroup{TroopId: int(t.TroopId), Count: n})
	}
	if len(groups) == 0 {
		return "", "", false
	}
	// 出征上限（与 botAttack 同一口径：集结令扩容 + 等比缩放）
	gather := 0
	if ezfyMarchCapOn() {
		gather = h.itemCount(b.UserID, ezfyGatherItemID())
		if maxG := ezfyGatherMax(); gather > maxG {
			gather = maxG
		}
	}
	cap, unlimited := h.ezfyOrderTroopCap(city.ID, gather, "")
	if !unlimited && cap > 0 {
		var gTotal int64
		for _, g := range groups {
			gTotal += g.Count
		}
		if gTotal > cap {
			for i := range groups {
				groups[i].Count = groups[i].Count * cap / gTotal
			}
			trim := groups[:0]
			for _, g := range groups {
				if g.Count > 0 {
					trim = append(trim, g)
				}
			}
			groups = trim
		}
	}
	if len(groups) == 0 {
		return "", "", false
	}
	code, msg := h.botCall(b.UserID, "", "POST", "/games/ezfy/order", map[string]interface{}{
		"city_id": int64(city.ID), "order_type": 3, "target_type": 1,
		"target_id": wt.ID, "target_x": wt.X, "target_y": wt.Y,
		"troops": groups, "auto_battle": 1, "gather": gather,
	})
	if code == 0 {
		h.botAtkCountInc(b)
		return "征服玩家野地", fmt.Sprintf("征服玩家野地 Lv%d(%d,%d) 派兵 %d 支(胜利后放弃)", wt.Level, wt.X, wt.Y, len(groups)), true
	}
	return "征服野地失败", msg, false
}

// ezfyBotTargetRow 目标筛选结果
type ezfyBotTargetRow struct {
	CityID uint
	UserID uint
	X      int
	Y      int
	Name   string
	Troops int64
}

// botPickTarget 在**自己周围**找一个可打的真实玩家城。
//
// 过滤（全部下推 SQL，走索引）：
//   - 非人机（is_bot = 0）、不是自己
//   - 坐标在自己 ±25 格内（走 idx_xy / is_bot 索引）
//   - ★ 2026-10-17 声望 ≥ 配置门槛 bot_min_prestige（管理端可配，默认 500000，0 = 不限）
//   - 目标兵力 < 自己兵力 × 0.6（打得过才打）
//   - 无生效中的免战保护（ezfy_city_effect effect_type=2）
//   - 城建时间 > 3 天（新手保护）
//   - 不在已放弃的冷却名单里
//
// 排序：声望高优先（人机专挑高声望的大号打），同声望挑兵力少的软柿子。
func (h *EzfyHandler) botPickTarget(city *model.EzfyCity, myTroops int64, cfg ezfyBotLevelCfg) (ezfyBotTargetRow, bool) {
	var r ezfyBotTargetRow
	limit := myTroops * 60 / 100
	if limit < 50 {
		return r, false
	}
	now := time.Now().UnixMilli()
	newbieBefore := time.Now().AddDate(0, 0, -3)
	err := h.DB.Raw(`
SELECT c.id AS city_id, c.user_id AS user_id, c.x AS x, c.y AS y,
       COALESCE(p.nickname, '') AS name, COALESCE(t.troops, 0) AS troops
FROM ezfy_city c
LEFT JOIN ezfy_profile p ON p.user_id = c.user_id
LEFT JOIN (SELECT city_id, SUM(count) AS troops FROM ezfy_city_troop GROUP BY city_id) t ON t.city_id = c.id
WHERE c.is_bot = 0
  AND c.user_id <> ?
  AND c.x BETWEEN ? AND ? AND c.y BETWEEN ? AND ?
  AND c.created_at < ?
  AND p.prestige >= ?
  AND COALESCE(t.troops, 0) BETWEEN 1 AND ?
  AND NOT EXISTS (SELECT 1 FROM ezfy_city_effect e
                  WHERE e.city_id = c.id AND e.effect_type = 2 AND e.until_time > ?)
  AND NOT EXISTS (SELECT 1 FROM ezfy_bot_target bt
                  WHERE bt.bot_uid = ? AND bt.target_uid = c.user_id AND bt.status = 3 AND bt.cool_until > ?)
ORDER BY p.prestige DESC, COALESCE(t.troops, 0) ASC
LIMIT 1`,
		city.UserID, city.X-25, city.X+25, city.Y-25, city.Y+25,
		newbieBefore, ezfyBotMinPrestige(), limit, now, city.UserID, now).Scan(&r).Error
	if err != nil || r.CityID == 0 {
		return r, false
	}
	return r, true
}

// botOfficer 人机军官养成（★ 2026-10-17 人机军官编辑修复）。
//
// ②③ 两条都走**和真人完全相同的业务接口**（/officers/:id/attr、/acade/recruit/hire），
//
//	扣黄金/占参谋部容量/同名限制/招募上限全部照常 —— 绝不绕过校验凭空加属性。
func (h *EzfyHandler) botOfficer(city *model.EzfyCity, botLevel int) (string, string, bool) {
	uid := city.UserID
	// ① 先验军校/参谋部等级（没建这两个建筑就不用想了）
	var academy, staff int
	h.DB.Model(&model.EzfyCityBuilding{}).
		Where("city_id = ? AND building_id = ?", city.ID, ezfyBuildingAcademy).
		Select("COALESCE(MAX(level),0)").Scan(&academy)
	h.DB.Model(&model.EzfyCityBuilding{}).
		Where("city_id = ? AND building_id = ?", city.ID, ezfyBuildingStaff).
		Select("COALESCE(MAX(level),0)").Scan(&staff)
	if academy < 1 || staff < 1 {
		return "", "", false
	}
	// ①½ 任命城守（★ 2026-10-17 用户规则「人机自己设置城守」）：当前城没有城守 →
	//   把军事最高的在职军官任命为城守（position=2，走真人任命接口）。
	//   城守给守军加防御还领被动经验，人机被打飞后新城市也立刻有守将。
	//   ★★ 2026-10-19 用户规则「设置城守优先设置有技能的军官」：技能优先于军事。
	if h.positionOfficer(city.ID, ezfyPositionGuard) == nil {
		var best model.EzfyOfficer
		if err := h.DB.Where("city_id = ? AND status = 0", city.ID).
			Order("CASE WHEN skill IS NULL OR skill = '' OR skill = '[]' THEN 1 ELSE 0 END ASC, military DESC, level DESC").First(&best).Error; err == nil && best.ID > 0 &&
			best.Position != ezfyPositionGuard {
			code, errMsg := h.botCall(uid, "", "POST",
				fmt.Sprintf("/games/ezfy/officers/%d/position", best.ID),
				map[string]interface{}{"position": 2})
			if code == 0 {
				return "任命城守", fmt.Sprintf("%s 就任城守", best.Name), true
			}
			// 任命失败（无参谋部等）不拦截，继续其余军官动作
			_ = errMsg
		}
	}
	// ② 优先「点属性」：本城在职军官里有待分配点的 → 一次全点当前**最高**属性
	//   （★ 2026-10-18 用户规则「哪个属性高就给哪个属性加点」：专精人机将领的长板，
	//   比固定点军事更贴合将领定位）。
	var o model.EzfyOfficer
	if err := h.DB.Where("city_id = ? AND status = 0 AND free_points > 0", city.ID).
		Order("free_points DESC").First(&o).Error; err == nil {
		// 找出三维里当前最高的那一项；并列时军事 > 后勤 > 学识 优先
		attrName, attrKey := "military", "军事"
		max := o.Military
		if o.Logistics > max {
			attrName, attrKey, max = "logistics", "后勤", o.Logistics
		}
		if o.Learning > max {
			attrName, attrKey = "learning", "学识"
		}
		code, msg := h.botCall(uid, "", "POST",
			fmt.Sprintf("/games/ezfy/officers/%d/attr", o.ID),
			map[string]interface{}{"attr": attrName, "count": o.FreePoints})
		if code == 0 {
			return "军官加点", fmt.Sprintf("%s %s+%d", o.Name, attrKey, o.FreePoints), true
		}
		return "军官加点", fmt.Sprintf("%s 加点失败: %s", o.Name, msg), false
	}
	// ②½ 无技能军官自动配技能（★ 2026-10-18 用户规则「人机会给没有技能的军官配置
	//   技能 随机的三个技能，不消耗技能书」）：直接写 skill JSON（不走道具接口，
	//   不打技能书消耗 —— 人机没有道具生产回路，不该卡在这）。
	//   ★★ 2026-10-19 用户规则「人机带技能的军官只能有三个」：已有技能军官 ≥ 3 就不再配。
	var skillCnt int64
	h.DB.Model(&model.EzfyOfficer{}).
		Where("city_id = ? AND status = 0 AND skill IS NOT NULL AND skill <> '' AND skill <> '[]'", city.ID).
		Count(&skillCnt)
	if skillCnt < 3 {
		var noSkill model.EzfyOfficer
		if err := h.DB.Where("city_id = ? AND status = 0 AND (skill IS NULL OR skill = '' OR skill = '[]')",
			city.ID).Order("military DESC, level DESC").First(&noSkill).Error; err == nil {
			names := h.ezfyRandomSkills(3)
			if len(names) > 0 {
				h.saveOfficerSkills(&noSkill, names)
				return "配置技能", fmt.Sprintf("%s 学会 %d 个技能（%s）", noSkill.Name, len(names),
					strings.Join(names, "/")), true
			}
		}
	}
	// ③ 待分配点都点完了 → 「招募」：军官没满员就招。
	//   ★ 2026-10-18 用户规则「人机军官不够用就自己招募5星普通军官」：优先从军官池
	//   取一条**5 星普通军官**（黄金够才招，5 星 3% 概率随机摇等不起）；
	//   池子里没有招得起的 5 星 → 回落原逻辑（recruitInfo 随机候选招第一个够黄金的）。
	var cnt int64
	h.DB.Model(&model.EzfyOfficer{}).Where("city_id = ?", city.ID).Count(&cnt)
	if int(cnt) >= staff {
		return "", "", false
	}
	if draft, ok := h.ezfyBotFiveStarDraft(city.Gold); ok {
		date := recruitCycleKey()
		var rec model.EzfyRecruit
		cands := joinDrafts([]ezfyOfficerDraft{*draft})
		if err := h.DB.Where("user_id = ? AND recruit_date = ?", uid, date).First(&rec).Error; err != nil {
			h.DB.Create(&model.EzfyRecruit{UserId: uid, RecruitDate: date, RefreshCount: 0, Candidates: cands})
		} else {
			h.DB.Model(&model.EzfyRecruit{}).Where("id = ?", rec.ID).
				Updates(map[string]interface{}{"candidates": cands})
		}
		code, msg := h.botCall(uid, "", "POST", "/games/ezfy/acade/recruit/hire",
			map[string]interface{}{"key": draft.Key})
		if code == 0 {
			return "招募军官", fmt.Sprintf("%s（5星普通军官 军事%d）", draft.Name, draft.Military), true
		}
		return "招募军官", "招募失败: " + msg, false
	}
	drafts, _, _ := h.recruitInfo(uid, academy)
	for _, d := range drafts {
		if d.Key == "" {
			continue
		}
		if city.Gold < d.Cost {
			continue // 挑黄金够的第一个招
		}
		code, msg := h.botCall(uid, "", "POST", "/games/ezfy/acade/recruit/hire",
			map[string]interface{}{"key": d.Key})
		if code == 0 {
			return "招募军官", fmt.Sprintf("%s（%d星 军事%d）", d.Name, d.Star, d.Military), true
		}
		return "招募军官", "招募失败: " + msg, false
	}
	return "", "", false
}

// ezfyRandomSkills 随机取 n 个技能名（不重复）。
func (h *EzfyHandler) ezfyRandomSkills(n int) []string {
	h.cfgs()
	names := make([]string, 0, len(ezfyCfg.skills))
	for _, s := range ezfyCfg.skills {
		names = append(names, s.Name)
	}
	if len(names) == 0 {
		return nil
	}
	// 洗牌后取前 n 个
	for i := len(names) - 1; i > 0; i-- {
		j := mathrand.Intn(i + 1)
		names[i], names[j] = names[j], names[i]
	}
	if len(names) > n {
		names = names[:n]
	}
	return names
}

// ezfyBotFiveStarDraft 从军官池取一条**招得起的 5 星普通军官**候选（黄金不够的跳过）。
//
// ★ 2026-10-18 用户规则「人机军官不够用就自己招募5星普通军官」。返回的候选只负责
// 塞进今日候选表，雇佣仍走真人同款 /acade/recruit/hire 业务接口（黄金/容量/同名照常校验）。
func (h *EzfyHandler) ezfyBotFiveStarDraft(gold int64) (*ezfyOfficerDraft, bool) {
	h.cfgs()
	pool := ezfyCfg.poolOfficers()
	if len(pool) == 0 {
		return nil, false
	}
	best := -1
	for i, g := range pool {
		if g.Star != 5 {
			continue
		}
		lv := g.Level
		if lv <= 0 || lv > ezfyOfficerMaxLevel {
			lv = ezfyOfficerMaxLevel
		}
		if int64(lv)*ezfyRecruitCostPerLevel > gold {
			continue // 招不起
		}
		if best < 0 || g.Military > pool[best].Military {
			best = i
		}
	}
	if best < 0 {
		return nil, false
	}
	g := pool[best]
	lv := g.Level
	if lv <= 0 || lv > ezfyOfficerMaxLevel {
		lv = ezfyOfficerMaxLevel
	}
	return &ezfyOfficerDraft{
		Key:    g.Name + "-" + strconv.Itoa(g.ID) + "-" + strconv.FormatInt(time.Now().UnixNano(), 10),
		PoolId: g.ID, Name: g.Name, Level: lv, Star: g.Star,
		Logistics: g.Logistics, Military: g.Military, Learning: g.Learning,
		Cost: int64(lv) * ezfyRecruitCostPerLevel,
	}, true
}

// botRelocateNear 人机自迁城（★ 2026-10-17 用户规则「人机迁城不受冷却限制」）。
//
// 走管理端同款路径：checkMoveTarget 校验后直接落库坐标 —— **不写 move_cool_until（不受冷却）、
// 不消耗迁城道具**。但副作用必须和真人迁城一样完整，否则产生脏数据：
//
//	① 清旧坐标的地图「玩家城」区域记录（ezfy_map_area area_type=3），否则地图还显示旧位置有城；
//	② 清落点坐标残留野地（城市脚下不能叠野地图标）；
//	③ 旧坐标若残留附属野地 → 在途采集(4)/驻守(7)部队先转返航，再删野地行（直接删会丢部队，
//	  见放弃野地的历史教训）。
//
// 频率很低（只有目标距主城 > 10 格、要远征时触发），不担心性能。
func (h *EzfyHandler) botRelocateNear(b *model.EzfyBot, city *model.EzfyCity, nearX, nearY int) (string, string, bool) {
	oldX, oldY := city.X, city.Y
	tx, ty, ok := h.ezfyBotPickRelocateSpot(nearX, nearY)
	if !ok {
		return "", "", false // 附近没空地 → 本轮不迁（下轮再试）
	}
	// ① 落库新坐标（不动 move_cool_until → 人机不受迁城冷却）
	if err := h.DB.Model(&model.EzfyCity{}).Where("id = ?", city.ID).
		Updates(map[string]interface{}{"x": tx, "y": ty}).Error; err != nil {
		return "迁城", "迁城落库失败: " + err.Error(), false
	}
	city.X, city.Y = tx, ty
	// ② 清旧坐标地图区域记录
	h.DB.Where("x = ? AND y = ? AND area_type = ?", oldX, oldY, 3).Delete(&model.EzfyMapArea{})
	// ③ 清新坐标脚下野地残留
	h.DB.Where("x = ? AND y = ?", tx, ty).Delete(&model.EzfyWildland{})
	// ④ 旧坐标残留附属野地：在途部队转返航 → 删野地行
	var oldWls []model.EzfyWildland
	h.DB.Where("x = ? AND y = ?", oldX, oldY).Find(&oldWls)
	if len(oldWls) > 0 {
		nowRel := time.Now().UnixMilli()
		for i := range oldWls {
			wid := oldWls[i].ID
			var wOrders []model.EzfyOrder
			h.DB.Where("user_id = ? AND target_id = ? AND order_type IN (4,7) AND status IN (0,1)", b.UserID, wid).Find(&wOrders)
			for j := range wOrders {
				o := &wOrders[j]
				o.Status = 2
				o.Result = o.Troops
				o.ReturnTime = nowRel + ezfyOneWayTravel(o)
				h.DB.Model(&model.EzfyOrder{}).Where("id = ?", o.ID).
					Updates(map[string]interface{}{"status": 2, "result": o.Result,
						"return_time": o.ReturnTime, "carry": o.Carry})
			}
		}
		h.DB.Where("x = ? AND y = ?", oldX, oldY).Delete(&model.EzfyWildland{})
	}
	h.ezfyBotLog(b.UserID, "迁城", fmt.Sprintf("%s(%d,%d) → (%d,%d)，靠近目标附近", city.Name, oldX, oldY, tx, ty), 1)
	return "迁城", fmt.Sprintf("%s(%d,%d) → (%d,%d)", city.Name, oldX, oldY, tx, ty), true
}

// ezfyBotPickRelocateSpot 在目标坐标附近 3~6 格环形内找一个「合法空位」。
//
// 从小圈到大圈（迁太近会被对方贴脸堵门，太远失去迁城意义）；每圈随机打散尝试顺序，
// 命中 checkMoveTarget（坐标 1~499 / 平原 / 无人占用）即返回；每圈最多试 20 个点防卡。
func (h *EzfyHandler) ezfyBotPickRelocateSpot(nearX, nearY int) (int, int, bool) {
	for radius := 3; radius <= 6; radius++ {
		cand := make([][2]int, 0, radius*8)
		for dx := -radius; dx <= radius; dx++ {
			for dy := -radius; dy <= radius; dy++ {
				if ezfyAbs(dx) < radius && ezfyAbs(dy) < radius {
					continue // 只取环上坐标
				}
				cand = append(cand, [2]int{nearX + dx, nearY + dy})
			}
		}
		// Fisher-Yates 随机打散，避免每轮都从同一角开始
		for i := len(cand) - 1; i > 0; i-- {
			j := mathrand.Intn(i + 1)
			cand[i], cand[j] = cand[j], cand[i]
		}
		checked := 0
		for _, p := range cand {
			checked++
			if checked > 20 {
				break
			}
			if h.checkMoveTarget(p[0], p[1], false) == "" {
				return p[0], p[1], true
			}
		}
	}
	return 0, 0, false
}

// botAttack 出击：按难度默认出兵 30~40 万、带本城军事最高的军官，掠夺目标城。
func (h *EzfyHandler) botAttack(b *model.EzfyBot, city *model.EzfyCity, p model.EzfyBotTarget, troopTotal int64) (string, string, bool) {
	var troops []model.EzfyCityTroop
	h.DB.Where("city_id = ? AND count > 0", city.ID).Find(&troops)
	var total int64
	for _, t := range troops {
		total += int64(t.Count)
	}
	if total == 0 {
		return "", "", false
	}
	// ★ 2026-10-17 出征数量按难度默认 30~40 万（管理端可单独覆盖），全兵种按占比分配；
	//   兵力不足目标数 → 全派（新手期没得选，派多少是多少）。
	//   出征类型：默认**征服**(3)，管理端调整记录在 b.AtkType（1侦察/2掠夺/3征服）。
	//   侦察只派 1 兵探路（大部队去送没意义）。
	leadName := ""
	var lead model.EzfyOfficer
	// ★★ 2026-10-18 城守/市长有城务在身不能带队（createOrder 会拒「「城守」xx有城务在身」）：
	//   先找**无官职**在职军官（military 0 也带，出征吃经验/升级）；若本城全是官员，
	//   则把军事最高的那位**临时卸任**（position=0，走真人卸任接口）再带 —— 人机出征不再被卡。
	//   ★★ 2026-10-19 用户规则「人机出征优先带有技能的军官」：技能优先于军事。
	h.DB.Where("city_id = ? AND status = 0 AND position = 0", city.ID).
		Order("CASE WHEN skill IS NULL OR skill = '' OR skill = '[]' THEN 1 ELSE 0 END ASC, military DESC, level DESC").First(&lead)
	if lead.ID == 0 {
		h.DB.Where("city_id = ? AND status = 0", city.ID).
			Order("CASE WHEN skill IS NULL OR skill = '' OR skill = '[]' THEN 1 ELSE 0 END ASC, military DESC, level DESC").First(&lead)
		if lead.ID > 0 && lead.Position != 0 {
			h.botCall(b.UserID, "", "POST",
				fmt.Sprintf("/games/ezfy/officers/%d/position", lead.ID),
				map[string]interface{}{"position": 0})
		}
	}
	if lead.ID > 0 {
		leadName = lead.Name
	}
	atkType := ezfyBotOrderTypeOf(b.AtkType)
	want := ezfyBotAtkTroopOf(b.Level, b.AtkTroop)
	if atkType == 1 {
		want = 1 // 侦察兵
	}
	if want > total {
		want = total
	}
	groups := make([]ezfyUnitGroup, 0, len(troops))
	for _, t := range troops {
		n := int64(t.Count) * want / total
		if n <= 0 {
			continue
		}
		groups = append(groups, ezfyUnitGroup{TroopId: int(t.TroopId), Count: n})
	}
	if len(groups) == 0 {
		return "", "", false
	}
	// ★★ 2026-10-17 修复：出征兵力必须 ≤ 出征上限（司令部等级×1万×指挥科技 + 集结令 + 军官），
	//   否则线上 march_cap_on=1 → createOrder 拒绝 → 人机「有 1 亿兵也打不出去」（只看见休整）。
	//   超了 → 全兵种等比缩放到上限内（派够上限也够拆目标城）。
	//   集结令不够 → 烧背包集结令扩额（每个 +10 万，上限 ezfyGatherMax()），createOrder 里真实扣减。
	gather := 0
	if ezfyMarchCapOn() {
		gather = h.itemCount(b.UserID, ezfyGatherItemID())
		if maxG := ezfyGatherMax(); gather > maxG {
			gather = maxG
		}
	}
	cap, unlimited := h.ezfyOrderTroopCap(city.ID, gather, leadName)
	if !unlimited && cap > 0 {
		var gTotal int64
		for _, g := range groups {
			gTotal += g.Count
		}
		if gTotal > cap {
			for i := range groups {
				groups[i].Count = groups[i].Count * cap / gTotal
			}
			trim := groups[:0]
			for _, g := range groups {
				if g.Count > 0 {
					trim = append(trim, g)
				}
			}
			groups = trim
		}
	}
	if len(groups) == 0 {
		return "出击失败", "出征兵力缩放后为空（上限过小）", false
	}
	// ★★ 2026-10-17 出征数量 bug 修复：此前这里加了一段「人口承载等比缩放」——以为
	//   createOrder 出征会校验人口。实际**出征不占人口**（部队一出征就从城内部队表扣掉、
	//   人口瞬间归还，见 ezfy.go 出征注释）→ 那段裁剪把 30~40 万火箭缩到 4 万（50 万人口
	//   / Pop12），人机出征数量永远不对。已删除；真正卡人口的是**训练**（botTrain 已对
	//   is_bot 跳过人口门槛，训练校验 ezfy.go 也跳过），训得出就能带得出。
	var tc model.EzfyCity
	if err := h.DB.First(&tc, p.TargetCityId).Error; err != nil {
		h.DB.Model(&model.EzfyBotTarget{}).Where("id = ?", p.ID).Update("status", 4)
		return "出击放弃", "目标城不存在", false
	}
	body := map[string]interface{}{
		"city_id": int64(city.ID), "order_type": atkType,
		"target_type": 3, "target_id": int64(tc.ID),
		"target_x": tc.X, "target_y": tc.Y,
		"troops": groups, "auto_battle": 1, "gather": gather,
	}
	if leadName != "" {
		body["officer"] = leadName // 带高军事军官出战
	}
	code, msg := h.botCall(b.UserID, "", "POST", "/games/ezfy/order", body)
	if code == 0 {
		// ★★ 2026-10-18 攻击频率修复：出击成功**不再冷却 12 小时**，计划保持「可出击」——
		//   同一场宣战生效期内按 atk_per_hour 频率反复出击（每次出击后由 botReschedule
		//   按攻击间隔排下一轮），直到目标被征服（ezfy_occupy）或战争结束，由
		//   botMilitary case2 出战前检查拦截并换新目标。原来打一次冷却 12h，频率再高也白搭。
		h.DB.Model(&model.EzfyBotTarget{}).Where("id = ?", p.ID).Update("status", 2)
		h.botAtkCountInc(b)
		return "出击", fmt.Sprintf("%s %s(%d,%d) 派兵 %d 支", ezfyOrderTypeName(atkType), p.Note, tc.X, tc.Y, len(groups)), true
	}
	// 失败（免战/兵力不足/队列满）→ 冷却 2 小时后重试
	h.DB.Model(&model.EzfyBotTarget{}).Where("id = ?", p.ID).
		Updates(map[string]interface{}{"status": 3, "cool_until": time.Now().Add(2 * time.Hour).UnixMilli()})
	return "出击失败", msg, false
}

// ============ 风控开关 ============

// ezfyBotAttackOn 「允许攻击玩家」开关（ezfy_cfg_limit.bot_atk_on，默认 1）。
//
// ⚠️ 0 是合法值（= 只发育不打人）→ 不能用 ezfyLimitOr；配置未加载时按默认「开」。
func ezfyBotAttackOn() bool {
	if !ezfyCfg.ready() {
		return true
	}
	return ezfyCfg.limit.BotAtkOn != 0
}

// ============ 人机求饶邮件 ============

var ezfyBotBegLock sync.Mutex
var ezfyBotBegRecent = map[string]int64{} // "botUID-atkUID" → 上次求饶邮件时间戳

// ezfyBotBegMail 人机被「即将打飞」→ 给进攻玩家发求饶邮件（★ 2026-10-19 用户规则
// 「人机要是被即将打飞会给玩家发邮件让他别打了」）。
//
//	knockedOut：占掉这座后守方已无自由城（当场被打飞）→ 求放过口径；
//	还有余城 → 求停手口径。
//
// 防刷：同一对人机/玩家 24 小时内最多一封（纯内存，服务器单进程）。
func (h *EzfyHandler) ezfyBotBegMail(atkUID, botUID uint, knockedOut bool) {
	key := fmt.Sprintf("%d-%d", botUID, atkUID)
	now := time.Now().UnixMilli()
	ezfyBotBegLock.Lock()
	if last, ok := ezfyBotBegRecent[key]; ok && now-last < 24*3600000 {
		ezfyBotBegLock.Unlock()
		return
	}
	ezfyBotBegRecent[key] = now
	ezfyBotBegLock.Unlock()
	name := h.ezfyProfileName(botUID)
	if name == "" {
		name = "新城市"
	}
	if knockedOut {
		h.ezfyMailCreate(botUID, atkUID, name, "别打了！",
			"我都被你打飞了……新家刚搬好，求你放过我。我保证不再犯你的地盘。")
		return
	}
	h.ezfyMailCreate(botUID, atkUID, name, "求别打了！",
		"城都被你占了，再打我就真要飞了。咱们停手吧，我认输。")
}

// ezfyBotDailyAtkCap 每个人机每日出征次数上限（ezfy_cfg_limit.bot_atk_daily，默认 3）。
func ezfyBotDailyAtkCap() int {
	if n := ezfyCfg.limit.BotAtkDaily; n > 0 {
		return n
	}
	return 3
}

// ezfyBotMinPrestige 人机自动挑攻击目标的最低声望门槛（ezfy_cfg_limit.bot_min_prestige，默认 500000）。
//
// 0 = 不限（对全体真人有效）；只影响 botPickTarget 自动选目标，管理端手动
// 「出征最近敌对城」不受此限制。（0 是合法值，配置未加载时回默认）
func ezfyBotMinPrestige() int {
	if !ezfyCfg.ready() {
		return 500000
	}
	if n := ezfyCfg.limit.BotMinPrestige; n > 0 {
		return n
	}
	return 0
}

// ezfyBotFoodFree 人机军队免耗粮开关（ezfy_cfg_limit.bot_food_free）。
//
// ★ 只对 city.IsBot=1 的城生效（调用方保证），真人**永远照常扣粮**；
//
//	判断只读内存 city 字段 + 配置缓存 —— calcResource 红线路径零查询。
//	0 是合法值（默认正常耗粮），不能用 ezfyLimitOr。
func ezfyBotFoodFree(isBot int) bool {
	if isBot == 0 {
		return false
	}
	if !ezfyCfg.ready() {
		return false
	}
	return ezfyCfg.limit.BotFoodFree != 0
}

// botAtkQuotaOK 该人机「现在能否再出征」：按「次/小时」频率节流。
//
// ★ 2026-10-17 从「每日上限」改成「每小时频率」：默认按难度阶梯（新手6 普通8 精英10 次/小时），
//
//	管理端可在人机编辑里单独覆盖（atk_per_hour>0）。时间窗 = 1小时 / 频率；
//	距上次出征不足时间窗 → 本小时额度用光，等下一轮再试。
func (h *EzfyHandler) botAtkQuotaOK(b *model.EzfyBot) bool {
	// ★ 2026-10-18 出征时间段：配置了时间段（如 20:00-22:00）就只在段内出击，空 = 全天不限
	if !ezfyBotAtkWindowOK() {
		return false
	}
	rate := ezfyBotAtkPerHourOf(b.Level, b.AtkPerHour)
	if rate <= 0 {
		return true
	}
	window := 3600000 / int64(rate)
	now := time.Now().UnixMilli()
	if b.LastAtkAt > 0 && now-b.LastAtkAt < window {
		return false
	}
	// ★ 2026-10-17 每日出征次数上限（用户规则「征服次数按难度 40-60 阶梯」）：
	//   新手 40 / 普通 50 / 精英 60 次/天；今日次数达到 → 今天不再出征（跨天重置）。
	daily := ezfyBotAtkDailyOf(b.Level)
	if daily > 0 && b.AtkDate == time.Now().Format("20060102") && b.AtkCountToday >= daily {
		return false
	}
	return true
}

// botAtkCountInc 今日出征次数 +1（跨天重置），并记录本次出征时间供攻频节流。
func (h *EzfyHandler) botAtkCountInc(b *model.EzfyBot) {
	today := time.Now().Format("20060102")
	cnt := 1
	if b.AtkDate == today {
		cnt = b.AtkCountToday + 1
	}
	h.DB.Model(&model.EzfyBot{}).Where("id = ?", b.ID).
		Updates(map[string]interface{}{"atk_count_today": cnt, "atk_date": today, "last_atk_at": time.Now().UnixMilli()})
}

// ezfyBotAtkDailyOf 每日出征次数上限（★ 2026-10-17 用户规则「征服次数按难度 40-60 阶梯」）：
//
//	新手 40 / 普通 50 / 精英 60 次/天；0 或负 = 不限制。
func ezfyBotAtkDailyOf(level int) int {
	switch level {
	case 1:
		return 40
	case 3:
		return 60
	}
	return 50
}

// ezfyBotAtkWindowOK 出征时间段是否放行（ezfy_cfg_limit.bot_atk_windows，管理端可配）。
//
// 格式：逗号分隔多个时间段，如 "20:00-22:00,08:00-12:00"；空 = 全天不限。
// 支持跨天时段（如 22:00-02:00 = 晚上十点到凌晨两点）。0 是合法值，不能用 ezfyLimitOr。
func ezfyBotAtkWindowOK() bool {
	if !ezfyCfg.ready() {
		return true
	}
	raw := strings.TrimSpace(ezfyCfg.limit.BotAtkWindows)
	if raw == "" {
		return true
	}
	now := time.Now()
	cur := now.Hour()*60 + now.Minute()
	for _, seg := range strings.Split(raw, ",") {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		parts := strings.Split(seg, "-")
		if len(parts) != 2 {
			continue
		}
		s, e := ezfyBotParseHM(parts[0]), ezfyBotParseHM(parts[1])
		if s < 0 || e < 0 {
			continue
		}
		if s <= e { // 同天时段
			if cur >= s && cur <= e {
				return true
			}
		} else if cur >= s || cur <= e { // 跨天时段
			return true
		}
	}
	return false
}

// ezfyBotParseHM 解析 "HH:MM" → 当日分钟数；非法返回 -1。
func ezfyBotParseHM(s string) int {
	p := strings.Split(strings.TrimSpace(s), ":")
	if len(p) != 2 {
		return -1
	}
	h, e1 := strconv.Atoi(strings.TrimSpace(p[0]))
	m, e2 := strconv.Atoi(strings.TrimSpace(p[1]))
	if e1 != nil || e2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return -1
	}
	return h*60 + m
}

// ezfyBotAtkPerHourOf 攻击频率（次/小时）：0=按难度默认（新手6 普通8 精英10），管理端可单独覆盖。
func ezfyBotAtkPerHourOf(level, override int) int {
	if override > 0 {
		return override
	}
	switch level {
	case 1:
		return 6
	case 3:
		return 10
	}
	return 8
}

// ezfyBotAtkTroopOf 单次出征兵力：0=按难度默认（新手30万 普通35万 精英40万），管理端可单独覆盖。
func ezfyBotAtkTroopOf(level int, override int64) int64 {
	if override > 0 {
		return override
	}
	switch level {
	case 1:
		return 300000
	case 3:
		return 400000
	}
	return 350000
}

// ezfyBotOrderTypeOf 该人机出征类型（ezfy_order.order_type）：
//
//	0=默认**征服**(3)，管理端可调 1侦察 / 2掠夺 / 3征服。
func ezfyBotOrderTypeOf(atkType int) int {
	if atkType >= 1 && atkType <= 3 {
		return atkType
	}
	return 3 // 默认征服
}

// boolInt bool → 0/1（日志用）。
func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ============ 其它小工具 ============

// ezfyBotIsBot 某个 user_id 是不是人机（二战侧权威源）。
func (h *EzfyHandler) ezfyBotIsBot(uid uint) bool {
	var n int64
	h.DB.Model(&model.EzfyProfile{}).Where("user_id = ? AND is_bot = 1", uid).Count(&n)
	return n > 0
}

// ezfyBotLevelText 难度档中文名（管理端展示）。
func ezfyBotLevelText(l int) string {
	switch l {
	case 1:
		return "新手陪练"
	case 3:
		return "精英"
	}
	return "普通"
}

// ezfyBotStatusText 当前状态文案（管理端展示）。
func ezfyBotStatusText(b model.EzfyBot, now int64) string {
	if b.Enabled == 0 {
		return "已暂停"
	}
	if b.NextActAt > now {
		return "等待行动"
	}
	return "行动中"
}

// ezfyBotNickOf 取人机展示昵称（profile 优先）。
func (h *EzfyHandler) ezfyBotNickOf(uid uint) string {
	var p model.EzfyProfile
	if err := h.DB.Select("nickname").Where("user_id = ?", uid).First(&p).Error; err == nil && p.Nickname != "" {
		return p.Nickname
	}
	return "人机" + strconv.FormatUint(uint64(uid), 10)
}

// trimBotStr 截断字符串（按 rune）。
func trimBotStr(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}

// 防止 strings 未使用（配置解析留给后续扩展）
var _ = strings.TrimSpace

// ezfyBotTargetLevel 该建筑的「目标等级」（按难度），达到后 AI 就跳到下一个建筑。
//
// ★ 2026-10-17 实测修正：原来只卡「最高 9 级」→ 人机会把第一优先的农田一路升到 9 级，
//
//	其它建筑（军工厂/民居）一个不建，永远造不了兵。现在按建筑分目标，保证均衡发展。
func ezfyBotTargetLevel(bid, botLevel int) int {
	base := 2
	switch bid {
	case 3, 4, 5, 6: // 资源田：产出是命脉，优先级最高
		base = 4
	case 2: // 民居（人口上限）
		base = 4
	case 1: // 市政厅
		base = 3
	case 14: // 军工厂
		base = 3
	}
	lv := base + (botLevel - 1)
	if lv > 9 {
		lv = 9 // ★ 9→10 需要建筑图纸，人机没有图纸
	}
	return lv
}
