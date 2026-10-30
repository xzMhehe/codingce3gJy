// 管理端 · 人机（AI 玩家）模块（★ 2026-10-17）
//
// 用户要求（人机模块方案 7.1）：人机管理要和玩家管理**彻底分开** ——
// 前端菜单/组件、后端路由/handler、权限点、数据（is_bot）五层全部独立，
// 日后维护人机模块不会碰到任何玩家管理代码，出问题能整块回滚。
//
// 路由前缀：/api/admin/ezfy-bot/*      权限点：module:ezfyBot
// 页面：admin-web/src/components/admin/AdminEzfyBot.vue
package ezfy

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// ezfyBotRow 人机列表一行（一次性批量补齐，不逐行查）
type ezfyBotRow struct {
	model.EzfyBot
	Nickname   string `json:"nickname"`
	Prestige   int    `json:"prestige"`
	Camp       int    `json:"camp"`
	GameUID    int64  `json:"game_uid"`
	CityID     uint   `json:"city_id"`
	CityName   string `json:"city_name"`
	X          int    `json:"x"`
	Y          int    `json:"y"`
	CityLevel  int    `json:"city_level"`
	TroopTotal int64  `json:"troop_total"`
}

// AdminEzfyBotList GET /admin/ezfy-bot/list —— 人机列表（一次全量，≤50 条）
//
// 与「玩家信息管理」完全独立：这里只出 is_bot=1 的人，那边只出 is_bot=0 的人。
func (h *EzfyAdmin) AdminEzfyBotList(c *gin.Context) {
	// ★ 2026-10-18 修复「发放资源里集结令显示 0」：集结令按道具名解析（ezfyGatherItemID），
	//   依赖 ezfyCfg 配置缓存；不先加载的话 gatherCID=0 → 全列表集结令都查成 0。
	h.ezfyH().cfgs()
	word := strings.TrimSpace(c.Query("word"))
	q := h.DB.Model(&model.EzfyBot{})
	if word != "" {
		// 昵称/用户ID 过滤（昵称在 profile，先拿 uid 再 IN，避免逐行子查询）
		if uid, err := strconv.Atoi(word); err == nil {
			q = q.Where("user_id = ?", uid)
		} else {
			var uids []uint
			h.DB.Model(&model.EzfyProfile{}).Where("nickname LIKE ? AND is_bot = 1", "%"+word+"%").
				Limit(200).Pluck("user_id", &uids)
			if len(uids) == 0 {
				resp.OK(c, gin.H{"list": []gin.H{}, "total": 0, "max": ezfyBotMaxCount})
				return
			}
			q = q.Where("user_id IN ?", uids)
		}
	}
	var bots []model.EzfyBot
	q.Order("id ASC").Limit(ezfyBotMaxCount + 10).Find(&bots)

	uids := make([]uint, 0, len(bots))
	for _, b := range bots {
		uids = append(uids, b.UserID)
	}
	profOf := map[uint]model.EzfyProfile{}
	cityOf := map[uint]model.EzfyCity{}
	cityNum := map[uint]int{}
	cityNames := map[uint][]string{}
	nextActOf := map[uint]int64{}
	troopOf := map[uint]int64{}
	// ★ 2026-10-17 人机集结令持有数（列表一眼可见，发/扣集结令有据可依）
	gatherOf := map[uint]int{}
	if len(uids) > 0 {
		var profs []model.EzfyProfile
		h.DB.Select("user_id, nickname, prestige, camp, game_uid, current_city_id").
			Where("user_id IN ?", uids).Find(&profs)
		for _, p := range profs {
			profOf[p.UserID] = p
		}
		var cities []model.EzfyCity
		h.DB.Select("id, user_id, name, x, y, city_level").Where("user_id IN ?", uids).Find(&cities)
		for _, ct := range cities {
			// 优先 current_city_id，否则取第一座
			if p, ok := profOf[ct.UserID]; ok && int64(ct.ID) == p.CurrentCityId {
				cityOf[ct.UserID] = ct
			} else if _, exists := cityOf[ct.UserID]; !exists {
				cityOf[ct.UserID] = ct
			}
			// ★ 2026-10-28 多城市：人机可能有多座城（被打光建新城 + 旧城归还/赎回）
			cityNum[ct.UserID]++
			cityNames[ct.UserID] = append(cityNames[ct.UserID], ct.Name)
		}
		// ★ 2026-10-28 多城市：兵力 = 该人机**全部城**的兵力合计（原来只算当前城，多城会漏）
		ids := make([]int64, 0, len(cities))
		cityIDOf := map[uint]uint{}
		for _, ct := range cities {
			ids = append(ids, int64(ct.ID))
			cityIDOf[ct.ID] = ct.UserID
		}
		if len(ids) > 0 {
			var sums []struct {
				CityID uint
				Total  int64
			}
			h.DB.Model(&model.EzfyCityTroop{}).Select("city_id, SUM(count) AS total").
				Where("city_id IN ?", ids).Group("city_id").Scan(&sums)
			for _, s := range sums {
				if uid, ok := cityIDOf[s.CityID]; ok {
					troopOf[uid] += s.Total
				}
			}
		}
		// 每座城的下次行动时间（状态列取最早的一座；★ 2026-10-28 多城市后
		// ezfy_bot.next_act_at 只作「管理端踢一下」的标记，不再是真实排期）
		var nexts []struct {
			BotUid uint
			Next   int64
		}
		h.DB.Model(&model.EzfyBotCity{}).Select("bot_uid, MIN(next_act_at) AS next").
			Where("bot_uid IN ? AND enabled = 1", uids).Group("bot_uid").Scan(&nexts)
		for _, n := range nexts {
			nextActOf[n.BotUid] = n.Next
		}
		// ★ 2026-10-17 人机集结令持有数
		gatherCID := ezfyGatherItemID()
		if gatherCID > 0 {
			var items []model.EzfyItem
			h.DB.Where("user_id IN ? AND cfg_id = ?", uids, gatherCID).Find(&items)
			for _, it := range items {
				gatherOf[it.UserId] = it.Count
			}
		}
	}

	now := time.Now().UnixMilli()
	out := make([]gin.H, 0, len(bots))
	for _, b := range bots {
		p := profOf[b.UserID]
		ct := cityOf[b.UserID]
		out = append(out, gin.H{
			"user_id": b.UserID, "nickname": p.Nickname, "game_uid": p.GameUID,
			"level": b.Level, "level_text": ezfyBotLevelText(b.Level),
			"persona": b.Persona, "enabled": b.Enabled,
			"prestige": p.Prestige, "camp": p.Camp, "camp_name": ezfyCampName(p.Camp),
			"city_id": ct.ID, "city_name": ct.Name, "x": ct.X, "y": ct.Y, "city_level": ct.CityLevel,
			// ★ 2026-10-28 多城市：城数 + 全部城名（列表「城市」列展示「N 座」并 tooltip 列出）
			"city_count": cityNum[b.UserID], "city_names": cityNames[b.UserID],
			"troop_total":  troopOf[b.UserID],
			"gather_count": gatherOf[b.UserID],
			// ★ 2026-10-17 出征设置：原值（0=按难度默认）+ 生效值（默认按难度阶梯）
			"atk_per_hour": b.AtkPerHour, "atk_per_hour_eff": ezfyBotAtkPerHourOf(b.Level, b.AtkPerHour),
			"atk_troop": b.AtkTroop, "atk_troop_eff": ezfyBotAtkTroopOf(b.Level, b.AtkTroop),
			"atk_type": b.AtkType, "atk_type_eff": ezfyBotOrderTypeOf(b.AtkType), // 0=默认征服，1侦查/2掠夺/3征服
			// ★ 2026-10-19 是否自动发言（编辑人机-基本信息可调）：0=关闭 1=开启
			"chat_on": b.ChatOn,
			// ★ 2026-10-25 是否打玩家附属野地（编辑人机-基本信息可调）：0=不打（默认）1=打
			"attack_wild": b.AttackWild,
			// ★ 2026-10-26 目标最小兵力（按人机分，编辑人机-基本信息可调，默认 100000，0=不限）
			"target_min_troops": b.TargetMinTroops,
			// ★ 2026-10-30 出征前自动补集结令（默认开）+ 连续出征未下城次数（阶梯增兵）
			"gather_top_up":   b.GatherTopUp,
			"atk_fail_streak": b.AtkFailStreak,
			"last_act_at": b.LastActAt, "last_action": b.LastAction,
			"next_act_at": nextActOf[b.UserID],
			"status_text": ezfyBotCityStatusText(b, nextActOf[b.UserID], now),
			"atk_count_today": b.AtkCountToday, "atk_date": b.AtkDate,
		})
	}
	resp.OK(c, gin.H{
		"list": out, "total": len(out), "max": ezfyBotMaxCount,
		"attack_on": ezfyBotAttackOn(),
		// ★ 2026-10-29 每日出征次数按难度下发（默认 60/100/160），顶栏三个输入框直接展示
		"atk_daily_lv1": ezfyBotAtkDailyOf(1),
		"atk_daily_lv2": ezfyBotAtkDailyOf(2),
		"atk_daily_lv3": ezfyBotAtkDailyOf(3),
		// ★ 2026-10-17 免耗粮开关一并下发，前端 list 页首帧即显示真实状态（此前每次进页面自动变「关」）
		"food_free": ezfyBotFoodFree(1),
		// ★ 2026-10-17 自动攻击目标最低声望门槛（0=不限），供顶栏配置展示
		"min_prestige": ezfyBotMinPrestige(),
		// ★ 2026-10-26 顶栏还缺这两项（前端从 /list 首帧取配置，原来只从 /config 返回 →
		//   进页面时「征兵线 / 出征时间段」永远显示默认值）。一并下发。
		"train_threshold": ezfyBotTrainThreshold(),
		"atk_windows":     ezfyBotAtkWindows(),
		// ★ 2026-10-26 自动挑目标时玩家城最小兵力（默认 10000，0 = 不限）
		"target_min_troops": ezfyBotTargetMinTroops(),
	})
}

// AdminEzfyBotCreate POST /admin/ezfy-bot/create {count, level, boost, nickname} —— 批量生成人机
//
// count 单次最多 10（避免一次建太多把 1 核打满）；level 1新手/2普通/3精英。
// nickname 可选马甲（只支持 count=1 时给这台人机起名；留空走自动词库随机昵称）。
func (h *EzfyAdmin) AdminEzfyBotCreate(c *gin.Context) {
	var in struct {
		Count    int    `json:"count"`
		Level    int    `json:"level"`
		Boost    bool   `json:"boost"`
		Nickname string `json:"nickname"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if in.Count < 1 || in.Count > 10 {
		resp.ParamError(c, "单次生成 1~10 个")
		return
	}
	if in.Level < 1 || in.Level > 3 {
		in.Level = 2
	}
	in.Nickname = strings.TrimSpace(in.Nickname)
	if in.Nickname != "" && in.Count > 1 {
		resp.ParamError(c, "自定义昵称（马甲）仅支持单次生成 1 个")
		return
	}
	// 生成走 EzfyHandler（建城/建档案的懒结算都在那边）
	ez := &EzfyHandler{DB: h.DB, HomeDB: h.HomeDB}
	ez.cfgs()
	created, failed := 0, 0
	var names []string
	for i := 0; i < in.Count; i++ {
		uid, err := ez.ezfyBotProvision(in.Level, in.Boost)
		if err != nil {
			failed++
			continue
		}
		created++
		// 马甲：覆盖自动词库昵称（家园 users + 二战 account/profile 三处同步改）
		if in.Nickname != "" && created == 1 {
			final := ez.ezfyBotEnsuredUniqueNick(in.Nickname)
			h.DB.Model(&model.User{}).Where("id = ?", uid).Update("nickname", final)
			h.DB.Model(&model.EzfyAccount{}).Where("game_id = ?", uid).Update("nickname", final)
			h.DB.Model(&model.EzfyProfile{}).Where("user_id = ?", uid).Update("nickname", final)
			names = append(names, final)
			continue
		}
		names = append(names, ez.ezfyBotNickOf(uid))
	}
	msg := "已生成 " + strconv.Itoa(created) + " 个人机"
	if len(names) > 0 {
		msg += "：" + strings.Join(names, "、")
	}
	if failed > 0 {
		msg += "（" + strconv.Itoa(failed) + " 个失败，可能已达数量上限）"
	}
	resp.OK(c, gin.H{"msg": msg, "created": created, "failed": failed})
}

// AdminEzfyBotToggle POST /admin/ezfy-bot/:uid/toggle {enabled} —— 单个启停
func (h *EzfyAdmin) AdminEzfyBotToggle(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("uid"))
	var in struct {
		Enabled int `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || uid <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	if in.Enabled != 0 {
		in.Enabled = 1
	}
	res := h.DB.Model(&model.EzfyBot{}).Where("user_id = ?", uid).Update("enabled", in.Enabled)
	if res.RowsAffected == 0 {
		resp.NotFound(c, "人机不存在")
		return
	}
	if in.Enabled == 1 {
		// 重新启用时立刻可行动，不用等下一轮
		h.DB.Model(&model.EzfyBot{}).Where("user_id = ?", uid).Update("next_act_at", time.Now().UnixMilli())
		resp.OK(c, gin.H{"msg": "已启用"})
		return
	}
	resp.OK(c, gin.H{"msg": "已暂停"})
}

// AdminEzfyBotLevel POST /admin/ezfy-bot/:uid/level {level} —— 调整难度档
func (h *EzfyAdmin) AdminEzfyBotLevel(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("uid"))
	var in struct {
		Level int `json:"level"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || uid <= 0 || in.Level < 1 || in.Level > 3 {
		resp.ParamError(c, "参数错误（难度 1~3）")
		return
	}
	res := h.DB.Model(&model.EzfyBot{}).Where("user_id = ?", uid).Updates(map[string]interface{}{
		"level": in.Level, "persona": ezfyBotPersona(in.Level),
	})
	if res.RowsAffected == 0 {
		resp.NotFound(c, "人机不存在")
		return
	}
	resp.OK(c, gin.H{"msg": "难度已调整为「" + ezfyBotLevelText(in.Level) + "」"})
}

// AdminEzfyBotAct POST /admin/ezfy-bot/:uid/act —— 立即行动一次（排障：看它这轮做了什么）
func (h *EzfyAdmin) AdminEzfyBotAct(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("uid"))
	if uid <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var b model.EzfyBot
	if err := h.DB.Where("user_id = ?", uid).First(&b).Error; err != nil {
		resp.NotFound(c, "人机不存在")
		return
	}
	// ★★ 2026-10-17 卡顿修复：决策 = 懒结算 + 内部 HTTP，同步等要 400ms+。
	//   改成后台 goroutine，管理端立刻返回；1~2 秒后刷新列表/日志即可看到动作。
	ez := &EzfyHandler{DB: h.DB, HomeDB: h.HomeDB, Secret: ezfyBotAdminSecret, ExpH: 1}
	bot := b
	go func() {
		ez.cfgs()
		ez.botTickOne(bot)
	}()
	resp.OK(c, gin.H{"msg": "已提交行动，稍后刷新查看它的动作"})
}

// ezfyBotAdminSecret 管理端「立即行动」用的 JWT 密钥。
//
// ★ 由 router.Setup 注入（与 cfg.Jwt.Secret 一致）—— 管理端 handler 本身不持有 Secret，
//
//	但「立即行动」要替人机签 token 打内部接口，必须能拿到。
var ezfyBotAdminSecret = ""

// SetEzfyBotAdminSecret 由 router.Setup 调用。
func SetEzfyBotAdminSecret(secret string) { ezfyBotAdminSecret = secret }

// AdminEzfyBotActAll POST /admin/ezfy-bot/act-all —— 一键立即行动：全部启用人机各连跑 N 轮
//
// ★ 2026-10-17 用户要求「加速人机变强」：手动批量加速（如活动前/测试时）。
//
//	每轮 = 一次 botTickOne（懒结算 → 决策 → 执行一个动作 → 写日志/排下次），
//	与「立即行动一轮」完全同链路，只是循环跑。轮数上限 50，防止误填刷爆日志。
func (h *EzfyAdmin) AdminEzfyBotActAll(c *gin.Context) {
	var in struct {
		Rounds int `json:"rounds"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if in.Rounds < 1 || in.Rounds > 50 {
		resp.ParamError(c, "轮数需在 1~50 之间")
		return
	}
	var bots []model.EzfyBot
	h.DB.Where("enabled = 1").Find(&bots)
	if len(bots) == 0 {
		resp.ParamError(c, "没有运行中的人机")
		return
	}
	// ★★ 2026-10-17 卡顿修复：原来 N 人机 × M 轮**全部串行**（50 个 × 10 轮 = 500 次懒结算+HTTP，
	//   每轮 400ms+ → 请求挂几分钟）。改成：后台 goroutine + 8 并发 worker pool，
	//   管理端立刻返回；带防重入标志（上一批没跑完再点直接拒绝）。
	if !atomic.CompareAndSwapInt32(&ezfyBotActAllRunning, 0, 1) {
		resp.ParamError(c, "上一批批量行动还在执行中，请稍后再试")
		return
	}
	ez := &EzfyHandler{DB: h.DB, HomeDB: h.HomeDB, Secret: ezfyBotAdminSecret, ExpH: 1}
	rounds, botsCopy := in.Rounds, bots
	go func() {
		defer atomic.StoreInt32(&ezfyBotActAllRunning, 0)
		ez.cfgs()
		sem := make(chan struct{}, 8)
		var wg sync.WaitGroup
		for i := 0; i < rounds; i++ {
			for _, b := range botsCopy {
				wg.Add(1)
				sem <- struct{}{}
				go func(bot model.EzfyBot) {
					defer wg.Done()
					defer func() { <-sem }()
					ez.botTickOne(bot)
				}(b)
			}
			wg.Wait()
		}
	}()
	resp.OK(c, gin.H{"msg": fmt.Sprintf("已在后台对 %d 个人机各执行 %d 轮，稍后刷新列表/日志查看", len(bots), in.Rounds)})
}

// ezfyBotActAllRunning 一键批量行动的防重入标志（1 = 正在跑）。
var ezfyBotActAllRunning int32

// AdminEzfyBotDelete DELETE /admin/ezfy-bot/:uid —— 删除单个人机（连根拔掉）
func (h *EzfyAdmin) AdminEzfyBotDelete(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("uid"))
	if uid <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var b model.EzfyBot
	if err := h.DB.Where("user_id = ?", uid).First(&b).Error; err != nil {
		resp.NotFound(c, "人机不存在")
		return
	}
	nick := ""
	{
		var p model.EzfyProfile
		h.DB.Select("nickname").Where("user_id = ?", uid).First(&p)
		nick = p.Nickname
	}
	if err := h.ezfyBotPurgeOne(uint(uid)); err != nil {
		resp.ServerError(c, err)
		return
	}
	resp.OK(c, gin.H{"msg": "已删除人机「" + nick + "」及其全部数据"})
}

// AdminEzfyBotClear POST /admin/ezfy-bot/clear —— 一键清空全部人机
//
// 走删号框架（ezfyPurgePlayerData）：二战库全清 + 家园账号物理删除。
func (h *EzfyAdmin) AdminEzfyBotClear(c *gin.Context) {
	var bots []model.EzfyBot
	h.DB.Select("user_id").Find(&bots)
	okCnt, failCnt := 0, 0
	for _, b := range bots {
		if err := h.ezfyBotPurgeOne(b.UserID); err != nil {
			failCnt++
			continue
		}
		okCnt++
	}
	// 计划表/日志表里可能残留孤儿行（人机已删但计划还在）
	h.DB.Exec("DELETE FROM ezfy_bot_target")
	h.DB.Exec("DELETE FROM ezfy_bot_log")
	// ★ 2026-10-28 多城市：每座城的行动槽同样要清
	h.DB.Exec("DELETE FROM ezfy_bot_city")
	resp.OK(c, gin.H{
		"msg":     "已清空 " + strconv.Itoa(okCnt) + " 个人机" + map[bool]string{true: "（" + strconv.Itoa(failCnt) + " 个失败）"}[failCnt > 0],
		"removed": okCnt, "failed": failCnt,
	})
}

// ezfyBotPurgeOne 删一个人机的全部数据：二战库（purge 框架）+ 家园 users 行。
func (h *EzfyAdmin) ezfyBotPurgeOne(uid uint) error {
	var cities []model.EzfyCity
	h.DB.Select("id").Where("user_id = ?", uid).Find(&cities)
	ids := make([]int64, 0, len(cities))
	for _, ct := range cities {
		ids = append(ids, int64(ct.ID))
	}
	if _, err := h.ezfyPurgePlayerData(uid, ids); err != nil {
		return err
	}
	// 家园账号（只删带 is_bot=1 的，防误删真人）
	h.home().Exec("DELETE FROM users WHERE id = ? AND is_bot = 1", uid)
	return nil
}

// AdminEzfyBotGrant POST /admin/ezfy-bot/:uid/grant —— 给人机城发资源（加速发育）
//
// ★ 与「玩家信息管理」的发放走同一个内部实现（giveResourcesNoCap，资源入当前所在城），
//
//	但入口独立挂 module:ezfyBot —— 人机已从玩家管理列表排除，运营只能从这里发。
//
// ★ 只发五类资源，**不发钻石**：人机不参与钻石/充值体系（方案 4.1）。
// ★ 发完把 next_act_at 提前到当前时刻 —— 资源到位立刻行动，不用等下一轮 tick。
func (h *EzfyAdmin) AdminEzfyBotGrant(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("uid"))
	if uid <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var in struct {
		// ★ 2026-10-28 多城市：资源落到哪座城（0/不传 = 当前城）
		CityID int64 `json:"city_id"`
		Gold   int64 `json:"gold"`
		Food   int64 `json:"food"`
		Steel  int64 `json:"steel"`
		Oil    int64 `json:"oil"`
		Rare   int64 `json:"rare"`
		// ★ 2026-10-17 集结令（正数发放进背包，负数从背包扣除，最低扣到 0）
		Gather int64 `json:"gather"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if in.Gold == 0 && in.Food == 0 && in.Steel == 0 && in.Oil == 0 && in.Rare == 0 && in.Gather == 0 {
		resp.ParamError(c, "请先填写要发放的数量")
		return
	}
	// 必须是人机（防误发真人 / 拿这个接口绕过玩家管理）
	var b model.EzfyBot
	if err := h.DB.Where("user_id = ?", uid).First(&b).Error; err != nil {
		resp.NotFound(c, "人机不存在")
		return
	}
	nick := ""
	ez := &EzfyHandler{DB: h.DB, HomeDB: h.HomeDB}
	nick = ez.ezfyBotNickOf(uint(uid))
	city, got := ez.giveResourcesNoCap(uint(uid), in.Food, in.Steel, in.Oil, in.Rare, in.Gold, in.CityID)
	msg := "已发放入「" + city.Name + "」"
	add := func(label string, want, real int64) {
		if want == 0 {
			return
		}
		msg += fmt.Sprintf(" %s%+d", label, real)
		if real != want {
			msg += fmt.Sprintf("(填%+d, 受资源上限/下限截断)", want)
		}
	}
	add("粮食", in.Food, got[0])
	add("钢铁", in.Steel, got[1])
	add("石油", in.Oil, got[2])
	add("稀矿", in.Rare, got[3])
	add("黄金", in.Gold, got[4])
	// ★ 2026-10-17 集结令走道具背包（ezfy_item）：正数发、负数扣（最低扣到 0），
	//   人机出征时自动烧集结令扩出征上限。
	if in.Gather != 0 {
		ez.cfgs()
		if gatherCID := ezfyGatherItemID(); gatherCID > 0 {
			if in.Gather > 0 {
				ez.addItem(uint(uid), gatherCID, int(in.Gather))
			} else {
				ez.DB.Exec("UPDATE ezfy_item SET count = GREATEST(count + ?, 0) WHERE user_id = ? AND cfg_id = ?",
					in.Gather, uid, gatherCID)
			}
			msg += fmt.Sprintf(" 集结令%+d", in.Gather)
		}
	}
	// 让它立刻消化这批资源（下一轮 tick 不用等）
	h.DB.Model(&model.EzfyBot{}).Where("user_id = ?", uid).Update("next_act_at", time.Now().UnixMilli())
	ez.ezfyBotLog(uint(uid), "管理端发资源", trimStr(msg, 200), 1)
	// ★ 2026-10-17 返回发放后的最新集结令数量：前端弹窗「当前」直接用返回值刷新，
	//   否则发放完显示的还是打开弹窗时的旧值（集结令数量 bug）。
	gatherNow := 0
	if gatherCID := ezfyGatherItemID(); gatherCID > 0 {
		gatherNow = ez.itemCount(uint(uid), gatherCID)
	}
	resp.OK(c, gin.H{"msg": fmt.Sprintf("已给人机「%s」发放：%s（它会立刻行动消化资源）", nick, msg), "gather_count": gatherNow})
}

// AdminEzfyBotBuildings GET /admin/ezfy-bot/:uid/buildings —— 编辑弹窗的数据源
//
// ★ 2026-10-17 用户要求「加速人机变强」：编辑数据从「建筑」扩展到整城——
//
//	返回人机当前城的：人口(总/上限/待恢复)、建筑行（含未建造坑位 level=0）、
//	兵种数量（ezfy_city_troop 全兵种）、科技等级（ezfy_user_tech 用户级）、
//	军官列表（ezfy_officer 按城），外加昵称/声望/城市名，一次回填表单。
func (h *EzfyAdmin) AdminEzfyBotBuildings(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("uid"))
	if uid <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var b model.EzfyBot
	if err := h.DB.Where("user_id = ?", uid).First(&b).Error; err != nil {
		resp.NotFound(c, "人机不存在")
		return
	}
	var p model.EzfyProfile
	h.DB.Where("user_id = ?", uid).First(&p)
	// ★ 2026-10-28 多城市：city_id 指定看哪座城（不传 = 当前城/第一座）
	wantCity, _ := strconv.Atoi(c.Query("city_id"))
	city := h.ezfyBotPickCity(uint(uid), int64(wantCity))
	out := gin.H{
		"nickname":    p.Nickname,
		"prestige":    p.Prestige,
		"city_id":     city.ID,
		"city_name":   city.Name,
		"cities":      h.ezfyBotCityList(uint(uid)),
		"pop":         city.Pop,              // 总人口
		"pop_max":     city.PopMax,           // 人口上限
		"pop_recover": city.PopRecover,       // 待恢复人口
		"pop_extra":   city.PopExtra,         // 额外空闲人口（人口满后召集产生，会回落）
		"free_pop":    ezBotPopFree(h, city), // 空闲人口（可直接编辑，保存时反推总人口/额外空闲）
		"buildings":   []gin.H{},
		"troops":      []gin.H{},
		"techs":       []gin.H{},
		"officers":    []gin.H{},
	}
	h.cfgs()
	if city.ID > 0 {
		var rows []model.EzfyCityBuilding
		h.DB.Where("city_id = ?", city.ID).Order("building_id ASC, level DESC").Find(&rows)
		list := make([]gin.H, 0, len(rows))
		for _, r := range rows {
			name := ""
			if cfg := ezfyCfg.building(r.BuildingId); cfg != nil {
				name = cfg.Name
			}
			list = append(list, gin.H{
				"id": r.ID, "building_id": r.BuildingId, "name": name,
				"level": r.Level, "max_level": func() int {
					if cfg := ezfyCfg.building(r.BuildingId); cfg != nil {
						return cfg.MaxLevel
					}
					return 10
				}(), "status": r.Status,
			})
		}
		out["buildings"] = list

		// 兵种数量：ezfy_city_troop 按城查，全兵种配置对齐（未造的 count=0）
		var troops []model.EzfyCityTroop
		h.DB.Where("city_id = ?", city.ID).Find(&troops)
		troopCount := map[int]int64{}
		for _, t := range troops {
			troopCount[t.TroopId] = t.Count
		}
		troopList := []gin.H{}
		for _, tc := range ezfyCfg.sortedTroops() {
			troopList = append(troopList, gin.H{
				"troop_id": tc.ID, "name": tc.Name, "count": troopCount[tc.ID],
			})
		}
		out["troops"] = troopList

		// 科技等级：ezfy_user_tech 用户级（无主城概念，全城共用），按用户查
		var uts []model.EzfyUserTech
		h.DB.Where("user_id = ?", uint(uid)).Find(&uts)
		techLevel := map[int]int{}
		for _, t := range uts {
			techLevel[t.TechId] = t.Level
		}
		techList := []gin.H{}
		techIDs := make([]int, 0, len(ezfyCfg.techs))
		for id := range ezfyCfg.techs {
			techIDs = append(techIDs, id)
		}
		sort.Ints(techIDs)
		for _, id := range techIDs {
			if cfg := ezfyCfg.tech(id); cfg != nil {
				techList = append(techList, gin.H{
					"tech_id": id, "name": cfg.Name, "max_level": cfg.MaxLevel,
					"level": techLevel[id],
				})
			}
		}
		out["techs"] = techList

		// 军官：ezfy_officer 按城查（逻辑删除自动过滤）
		var officers []model.EzfyOfficer
		h.DB.Where("city_id = ?", city.ID).Order("position DESC, id ASC").Find(&officers)
		offList := make([]gin.H, 0, len(officers))
		for _, o := range officers {
			offList = append(offList, gin.H{
				"id": o.ID, "name": o.Name, "star": o.Star, "level": o.Level,
				"exp": o.Exp, "military": o.Military, "logistics": o.Logistics,
				"learning": o.Learning, "skills": officerSkills(&o), "position": o.Position,
				// ★ 2026-10-17 管理端加点：带出待分配点，前端可改三维最终值（差额自动从这扣）
				"free_points": o.FreePoints,
			})
		}
		out["officers"] = offList
	}
	resp.OK(c, out)
}

// AdminEzfyBotEdit POST /admin/ezfy-bot/:uid/edit —— 编辑人机（昵称/声望/城市名/建筑/人口/科技/兵种/军官）
//
// body（全部可选，传了才改）：
//
//	{nickname, prestige, city_name,
//	 pop, pop_max, pop_recover,            // 城市人口（空闲人口 = 人口 − 建筑占用 − 训练占用 − 待恢复）
//	 buildings:[{id, level}],              // 建筑等级（改等级会顺带清施工状态）
//	 troops:[{troop_id, count}],           // 兵种数量（按 (city_id,troop_id) upsert，count=0 删行）
//	 techs:[{tech_id, level}],             // 科技等级（ezfy_user_tech 用户级 upsert）
//	 officers:[{id, level, exp, skills[]}] // 军官等级/经验/技能（技能为名字数组）
//	}
//
// ★ 昵称**三处同步**：家园 users（展示权威源）/ ezfy_profile / ezfy_account ——
//
//	漏一处就会出现「地图上叫 A、聊天里叫 B」的分裂。
//
// ★ 建筑直接改库（管理端权限），顺手清掉施工状态（改等级时正在建的队列没有意义）。
func (h *EzfyAdmin) AdminEzfyBotEdit(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("uid"))
	if uid <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var b model.EzfyBot
	if err := h.DB.Where("user_id = ?", uid).First(&b).Error; err != nil {
		resp.NotFound(c, "人机不存在")
		return
	}
	var in struct {
		// ★ 2026-10-28 多城市：作用于哪座城（0/不传 = 当前城）。城市名/人口/建筑/兵种/军官都按城存。
		CityID     int64   `json:"city_id"`
		Nickname   *string `json:"nickname"`
		Prestige   *int    `json:"prestige"`
		CityName   *string `json:"city_name"`
		Pop        *int64  `json:"pop"`
		PopMax     *int64  `json:"pop_max"`
		PopRecover *int64  `json:"pop_recover"`
		FreePop    *int64  `json:"free_pop"`
		Buildings  []struct {
			ID    int64 `json:"id"`
			Level int   `json:"level"`
		} `json:"buildings"`
		Troops []struct {
			TroopID int   `json:"troop_id"`
			Count   int64 `json:"count"`
		} `json:"troops"`
		Techs []struct {
			TechID int `json:"tech_id"`
			Level  int `json:"level"`
		} `json:"techs"`
		Officers []struct {
			ID         int      `json:"id"`
			Level      int      `json:"level"`
			Exp        int64    `json:"exp"`
			FreePoints *int     `json:"free_points"` // ★ 2026-10-17 直接设待分配点（0 合法）
			Military   *int     `json:"military"`    // ★ 三维直接设最终值，差额自动从待分配点扣
			Logistics  *int     `json:"logistics"`
			Learning   *int     `json:"learning"`
			Skills     []string `json:"skills"`
		} `json:"officers"`
		// ★ 2026-10-17 出征设置（0=按难度默认：攻频 新手6/普通8/精英10 次·小时，出兵 30/35/40万）
		AtkPerHour *int   `json:"atk_per_hour"`
		AtkTroop   *int64 `json:"atk_troop"`
		// ★ 2026-10-17 出征类型：1侦察 2掠夺 3征服（不传/0=默认征服）
		AtkType *int `json:"atk_type"`
		// ★ 2026-10-19 是否自动发言（基本信息）：0=关闭 1=开启（默认开）
		ChatOn *int `json:"chat_on"`
		// ★ 2026-10-25 是否打玩家附属野地（基本信息）：0=不打（默认）1=打
		AttackWild *int `json:"attack_wild"`
		// ★ 2026-10-26 目标最小兵力（基本信息，按人机分）：自动挑目标时玩家城的最小兵力，
		//   默认 100000；**0 合法 = 不限**，所以用指针接收（nil=不修改）
		TargetMinTroops *int64 `json:"target_min_troops"`
		// ★ 2026-10-30 出征前自动补集结令（基本信息）：1=开（默认）0=关
		GatherTopUp *int `json:"gather_top_up"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	changes := []string{}

	// ① 昵称（三处同步）
	if in.Nickname != nil {
		nick := strings.TrimSpace(*in.Nickname)
		if nick != "" {
			r := []rune(nick)
			if len(r) < 2 || len(r) > 20 {
				resp.ParamError(c, "昵称 2~20 个字")
				return
			}
			var cnt int64
			h.home().Model(&model.User{}).Where("nickname = ? AND id <> ?", nick, uid).Count(&cnt)
			if cnt > 0 {
				resp.ParamError(c, "昵称「"+nick+"」已被占用")
				return
			}
			h.home().Model(&model.User{}).Where("id = ?", uid).Update("nickname", nick)
			h.DB.Model(&model.EzfyProfile{}).Where("user_id = ?", uid).Update("nickname", nick)
			h.DB.Model(&model.EzfyAccount{}).Where("game_id = ?", uid).Update("nickname", nick)
			changes = append(changes, "昵称→"+nick)
		}
	}

	// ② 声望
	if in.Prestige != nil {
		if *in.Prestige < 0 {
			resp.ParamError(c, "声望不能为负")
			return
		}
		h.DB.Model(&model.EzfyProfile{}).Where("user_id = ?", uid).Update("prestige", *in.Prestige)
		changes = append(changes, fmt.Sprintf("声望→%d", *in.Prestige))
	}

	// ③ 城市名 / ④ 建筑等级（都挂在**指定城**上；不传 city_id 就是当前城）
	city := h.ezfyBotPickCity(uint(uid), in.CityID)
	if in.CityName != nil && city.ID > 0 {
		cn := strings.TrimSpace(*in.CityName)
		if cn != "" {
			h.DB.Model(&model.EzfyCity{}).Where("id = ?", city.ID).Update("name", trimStr(cn, 20))
			changes = append(changes, "城市名→"+trimStr(cn, 20))
		}
	}
	if len(in.Buildings) > 0 && city.ID > 0 {
		n := 0
		for _, bl := range in.Buildings {
			if bl.Level < 0 || bl.Level > 15 {
				continue
			}
			// 归属校验：建筑行必须真属于这个人机的城（AND city_id），防乱传 id 改别人的城
			res := h.DB.Model(&model.EzfyCityBuilding{}).
				Where("id = ? AND city_id = ?", bl.ID, city.ID).
				Updates(map[string]interface{}{
					"level": bl.Level, "status": 0, "start_time": 0, "end_time": 0, "target_level": 0,
				})
			n += int(res.RowsAffected)
		}
		if n > 0 {
			changes = append(changes, fmt.Sprintf("建筑等级 %d 项", n))
		}
	}

	// ⑤ 人口：总人口 / 上限 / 待恢复人口（空闲人口由此推导，无需单独字段）
	// ★ 2026-10-17 侦察读的是「当前人口 pop」：配置 pop_max 时同步 pop = pop_max，
	//   否则别人侦察永远看到初始的 5000（用户反馈：配置了 50 万但侦察显示默认 5000）
	if city.ID > 0 && (in.Pop != nil || in.PopMax != nil || in.PopRecover != nil) {
		upd := map[string]interface{}{}
		if in.Pop != nil && *in.Pop >= 0 {
			upd["pop"] = *in.Pop
		}
		if in.PopMax != nil && *in.PopMax >= 0 {
			upd["pop_max"] = *in.PopMax
			if _, ok := upd["pop"]; !ok {
				upd["pop"] = *in.PopMax // 只配上限时把当前人口一起提到上限（要的就是显示 50 万）
			}
		}
		if in.PopRecover != nil && *in.PopRecover >= 0 {
			now := time.Now().UnixMilli()
			upd["pop_recover"] = *in.PopRecover
			upd["pop_recover_at"] = now
			if *in.PopRecover > 0 {
				upd["pop_recover_end_at"] = now + ezfyPopRecoverSeconds()*1000
			} else {
				upd["pop_recover_end_at"] = 0
			}
		}
		if len(upd) > 0 {
			h.DB.Model(&model.EzfyCity{}).Where("id = ?", city.ID).Updates(upd)
			// 待恢复人口改小/清零 → 空闲人口立刻恢复（与移民法案道具同口径）
			changes = append(changes, "人口(总/上限/待恢复)")
		}
	}

	// ⑤½ 空闲人口（★ 2026-10-17 用户要求「空闲人口也能编辑」）
	//
	// 空闲人口 = 人口 − 建筑占用 − 训练占用 − 待恢复 + 额外空闲（ezfyPopFree）。
	// 其中「额外空闲」是人口达到民居上限后召集产生、会回落的部分；编辑时：
	//   ① 先尽量用总人口满足目标空闲（最多到 pop_max）；
	//   ② 总人口到上限后仍不够的部分写入 pop_extra，并按配置重新起算回落窗口。
	// ★ 必须放在 ⑤ 之后执行：同时传 pop 和 free_pop 时以 free_pop 为准（它基于最终 pop 反推）。
	if city.ID > 0 && in.FreePop != nil {
		if *in.FreePop < 0 {
			resp.ParamError(c, "空闲人口不能为负")
			return
		}
		if *in.FreePop > ezfyResSafeMax {
			resp.ParamError(c, "空闲人口超过安全上限")
			return
		}
		ez := &EzfyHandler{DB: h.DB, HomeDB: h.HomeDB}
		// 重新读一次城（⑤ 可能刚改过 pop/pop_max/pop_recover）
		var c2 model.EzfyCity
		h.DB.First(&c2, city.ID)
		bPop := ez.buildingPop(&c2)
		tPop := ez.troopPop(c2.ID)
		normalPop := bPop + tPop + c2.PopRecover
		if normalPop > ezfyResSafeMax {
			normalPop = ezfyResSafeMax
		}
		newPop := *in.FreePop + normalPop
		if newPop < 0 || newPop > ezfyResSafeMax {
			newPop = ezfyResSafeMax
		}
		if c2.PopMax > 0 && newPop > c2.PopMax {
			newPop = c2.PopMax
		}
		extra := *in.FreePop - (newPop - normalPop)
		if extra < 0 {
			extra = 0
		}
		now := time.Now().UnixMilli()
		upd := map[string]interface{}{"pop": newPop}
		if extra > 0 {
			upd["pop_extra"] = extra
			upd["pop_extra_at"] = now
			upd["pop_extra_end_at"] = now + ezfyPopExtraDecayMs()
		} else {
			upd["pop_extra"] = 0
			upd["pop_extra_at"] = now
			upd["pop_extra_end_at"] = 0
		}
		h.DB.Model(&model.EzfyCity{}).Where("id = ?", c2.ID).Updates(upd)
		msg := fmt.Sprintf("空闲人口→%d", *in.FreePop)
		if extra > 0 {
			msg += fmt.Sprintf("(总人口受上限限制, %d 计入临时空闲)", extra)
		}
		changes = append(changes, msg)
	}

	// ⑥ 兵种数量：按 (city_id, troop_id) upsert；count=0 删除该行
	if len(in.Troops) > 0 && city.ID > 0 {
		n := 0
		for _, tp := range in.Troops {
			if tp.TroopID <= 0 || tp.Count < 0 {
				continue
			}
			if tp.Count == 0 {
				res := h.DB.Where("city_id = ? AND troop_id = ?", city.ID, tp.TroopID).Delete(&model.EzfyCityTroop{})
				if res.RowsAffected > 0 {
					n++
				}
				continue
			}
			var exist model.EzfyCityTroop
			err := h.DB.Where("city_id = ? AND troop_id = ?", city.ID, tp.TroopID).First(&exist).Error
			if err != nil { // 无行 → 新建
				if h.DB.Create(&model.EzfyCityTroop{CityId: int64(city.ID), TroopId: tp.TroopID, Count: tp.Count}).Error == nil {
					n++
				}
			} else { // 有行 → 更新数量
				if h.DB.Model(&model.EzfyCityTroop{}).Where("id = ?", exist.ID).Update("count", tp.Count).Error == nil {
					n++
				}
			}
		}
		if n > 0 {
			changes = append(changes, fmt.Sprintf("兵种 %d 种", n))
		}
	}

	// ⑦ 科技等级：ezfy_user_tech 用户级 upsert（全城共用）
	if len(in.Techs) > 0 {
		n := 0
		for _, tc := range in.Techs {
			if tc.TechID <= 0 || tc.Level < 0 {
				continue
			}
			if tc.Level > 100 {
				tc.Level = 100
			}
			var exist model.EzfyUserTech
			err := h.DB.Where("user_id = ? AND tech_id = ?", uint(uid), tc.TechID).First(&exist).Error
			if err != nil {
				if h.DB.Create(&model.EzfyUserTech{UserId: uint(uid), TechId: tc.TechID, Level: tc.Level}).Error == nil {
					n++
				}
			} else {
				if h.DB.Model(&model.EzfyUserTech{}).Where("id = ?", exist.ID).Update("level", tc.Level).Error == nil {
					n++
				}
			}
		}
		if n > 0 {
			changes = append(changes, fmt.Sprintf("科技 %d 项", n))
		}
	}

	// ⑧ 军官：等级 / 经验 / 技能 / 属性加点
	//   加点规则（★ 2026-10-17 用户要求「管理端也要能加点」）：
	//   三维(military/logistics/learning)直接设**最终值**，差额自动从待分配点扣减——
	//   三维合计调高 → 待分配点减少；调低 → 多出的点退回待分配点。
	//   同时显式传 free_points → 以传值为准（管理端可直接给点）。
	//   若调高但待分配点不够 → 拒绝并提示（防凭空加属性）。
	if len(in.Officers) > 0 {
		n := 0
		for _, of := range in.Officers {
			if of.ID <= 0 {
				continue
			}
			upd := map[string]interface{}{}
			if of.Level >= 1 {
				lv := of.Level
				if lv > 350 {
					lv = 350
				}
				upd["level"] = lv
			}
			if of.Exp >= 0 {
				upd["exp"] = of.Exp
			}
			if of.Skills != nil {
				// 技能去重、每名军官最多 3 个（与游戏规则一致）
				seen := map[string]bool{}
				names := make([]string, 0, len(of.Skills))
				for _, s := range of.Skills {
					s = strings.TrimSpace(s)
					if s == "" || seen[s] {
						continue
					}
					seen[s] = true
					names = append(names, s)
					if len(names) >= 3 {
						break
					}
				}
				buf, _ := json.Marshal(names)
				upd["skill"] = string(buf)
			}
			// ★ 加点：三维最终值 + 待分配点守恒
			if of.Military != nil || of.Logistics != nil || of.Learning != nil || of.FreePoints != nil {
				var o model.EzfyOfficer
				if err := h.DB.Where("id = ? AND city_id = ?", of.ID, city.ID).First(&o).Error; err == nil {
					oldSum := o.Military + o.Logistics + o.Learning
					newM, newLg, newLn := o.Military, o.Logistics, o.Learning
					if of.Military != nil {
						newM = *of.Military
					}
					if of.Logistics != nil {
						newLg = *of.Logistics
					}
					if of.Learning != nil {
						newLn = *of.Learning
					}
					if newM < 0 || newLg < 0 || newLn < 0 {
						resp.ParamError(c, "军官属性值不能为负")
						return
					}
					delta := (newM + newLg + newLn) - oldSum
					newFree := o.FreePoints - delta
					if of.FreePoints != nil {
						newFree = *of.FreePoints // 显式给点，管理端说了算
					}
					if newFree < 0 {
						resp.ParamError(c, fmt.Sprintf("「%s」三维合计超出待分配点 %d 点，请先调高待分配点或降低属性", o.Name, -newFree))
						return
					}
					if of.Military != nil {
						upd["military"] = newM
					}
					if of.Logistics != nil {
						upd["logistics"] = newLg
					}
					if of.Learning != nil {
						upd["learning"] = newLn
					}
					if of.FreePoints != nil || delta != 0 {
						upd["free_points"] = newFree
					}
				}
			}
			if len(upd) > 0 {
				// 归属校验：军官行必须属于这个人机的城
				res := h.DB.Model(&model.EzfyOfficer{}).
					Where("id = ? AND city_id = ?", of.ID, city.ID).Updates(upd)
				n += int(res.RowsAffected)
			}
		}
		if n > 0 {
			changes = append(changes, fmt.Sprintf("军官 %d 名", n))
		}
	}

	// ⑨ 出征设置：攻击频率（次/小时）、单次出征兵力；0 = 恢复按难度默认
	ezUpd := map[string]interface{}{}
	if in.AtkPerHour != nil {
		if *in.AtkPerHour < 0 || *in.AtkPerHour > 60 {
			resp.ParamError(c, "攻击频率需在 0~60 次/小时（0=按难度默认）")
			return
		}
		ezUpd["atk_per_hour"] = *in.AtkPerHour
	}
	if in.AtkTroop != nil {
		if *in.AtkTroop < 0 || *in.AtkTroop > 500000000 {
			resp.ParamError(c, "出征兵力需在 0~5 亿（0=按难度默认）")
			return
		}
		ezUpd["atk_troop"] = *in.AtkTroop
	}
	if in.AtkType != nil {
		if *in.AtkType < 0 || *in.AtkType > 3 {
			resp.ParamError(c, "出征类型需为 1侦察/2掠夺/3征服（0=默认征服）")
			return
		}
		ezUpd["atk_type"] = *in.AtkType
	}
	// ⑩ 是否自动发言（★ 2026-10-19 用户规则「编辑人机-基本信息可调」）
	if in.ChatOn != nil {
		if *in.ChatOn < 0 || *in.ChatOn > 1 {
			resp.ParamError(c, "自动发言需为 0/1（0=关闭 1=开启）")
			return
		}
		ezUpd["chat_on"] = *in.ChatOn
		changes = append(changes, fmt.Sprintf("自动发言→%s", map[bool]string{true: "开", false: "关"}[*in.ChatOn == 1]))
	}
	// ⑪ 是否打玩家附属野地（★ 2026-10-25 用户要求「可编辑，默认不打玩家野地」）
	if in.AttackWild != nil {
		if *in.AttackWild < 0 || *in.AttackWild > 1 {
			resp.ParamError(c, "打野地需为 0/1（0=不打 1=打）")
			return
		}
		ezUpd["attack_wild"] = *in.AttackWild
		changes = append(changes, fmt.Sprintf("打野地→%s", map[bool]string{true: "开", false: "关"}[*in.AttackWild == 1]))
	}
	// ⑫ 目标最小兵力（★ 2026-10-26 用户要求「配置按人机分，默认最少 100000 兵力」）
	//   **0 合法 = 不限**；负值无意义直接拒。
	if in.TargetMinTroops != nil {
		if *in.TargetMinTroops < 0 || *in.TargetMinTroops > 1000000000000 {
			resp.ParamError(c, "目标最小兵力不能为负数（0 = 不限）")
			return
		}
		ezUpd["target_min_troops"] = *in.TargetMinTroops
		changes = append(changes, fmt.Sprintf("目标最小兵力→%d", *in.TargetMinTroops))
	}
	// ⑬ 出征前自动补集结令（★ 2026-10-30：默认开，关掉后人机背包没有集结令 → 被司令部出征上限卡死）
	if in.GatherTopUp != nil {
		if *in.GatherTopUp < 0 || *in.GatherTopUp > 1 {
			resp.ParamError(c, "自动补集结令需为 0/1（0=关闭 1=开启）")
			return
		}
		ezUpd["gather_top_up"] = *in.GatherTopUp
		changes = append(changes, fmt.Sprintf("自动补集结令→%s", map[bool]string{true: "开", false: "关"}[*in.GatherTopUp == 1]))
	}
	if len(ezUpd) > 0 {
		h.DB.Model(&model.EzfyBot{}).Where("user_id = ?", uid).Updates(ezUpd)
		changes = append(changes, "出征设置")
	}

	if len(changes) == 0 {
		resp.ParamError(c, "没有要修改的内容")
		return
	}
	ez := &EzfyHandler{DB: h.DB, HomeDB: h.HomeDB}
	ez.ezfyBotLog(uint(uid), "管理端编辑", trimStr(strings.Join(changes, "；"), 200), 1)
	// 发完立刻行动（改建筑等级后它下一轮就能用上）
	h.DB.Model(&model.EzfyBot{}).Where("user_id = ?", uid).Update("next_act_at", time.Now().UnixMilli())
	resp.OK(c, gin.H{"msg": "已修改：" + strings.Join(changes, "；")})
}

// ezfyBotCurrentCity 人机的当前城（current_city_id 优先，无效回落第一座）。
func (h *EzfyAdmin) ezfyBotCurrentCity(uid uint) model.EzfyCity {
	return h.ezfyBotPickCity(uid, 0)
}

// ezfyBotPickCity 取人机名下的一座城：cityID > 0 且归属该人机 → 用它；
// 否则 profile.current_city_id；再否则 id 最小的城。
//
// ★ 2026-10-28 多城市：管理端所有「作用于一座城」的入口都走这里，city_id 显式传就绝不猜。
func (h *EzfyAdmin) ezfyBotPickCity(uid uint, cityID int64) model.EzfyCity {
	var city model.EzfyCity
	if cityID > 0 {
		if err := h.DB.Where("id = ? AND user_id = ?", cityID, uid).First(&city).Error; err == nil {
			return city
		}
	}
	var p model.EzfyProfile
	if err := h.DB.Where("user_id = ?", uid).First(&p).Error; err == nil && p.CurrentCityId > 0 {
		h.DB.Where("id = ? AND user_id = ?", p.CurrentCityId, uid).First(&city)
		if city.ID > 0 {
			return city
		}
	}
	h.DB.Where("user_id = ?", uid).Order("id ASC").First(&city)
	return city
}

// ezfyBotCityList 人机名下**全部城市**的概览（编辑弹窗的城市下拉 / 城市管理用）。
//
// ★ 2026-10-28 多城市：人机被打光后会被补给流程建新城，旧城被「归还/赎回」→ 一个人机可能有多座城，
// 每座城都要能单独查看/维护（人口、资源、建筑、兵种、军官都按城存）。
func (h *EzfyAdmin) ezfyBotCityList(uid uint) []gin.H {
	var cities []model.EzfyCity
	h.DB.Where("user_id = ?", uid).Order("id ASC").Find(&cities)
	out := make([]gin.H, 0, len(cities))
	if len(cities) == 0 {
		return out
	}
	ids := make([]int64, 0, len(cities))
	for _, ct := range cities {
		ids = append(ids, int64(ct.ID))
	}
	troopOf := map[int64]int64{}
	var sums []struct {
		CityID int64
		Total  int64
	}
	h.DB.Model(&model.EzfyCityTroop{}).Select("city_id, COALESCE(SUM(count),0) AS total").
		Where("city_id IN ?", ids).Group("city_id").Scan(&sums)
	for _, s := range sums {
		troopOf[s.CityID] = s.Total
	}
	offOf := map[int64]int64{}
	var offs []struct {
		CityID int64
		Total  int64
	}
	h.DB.Model(&model.EzfyOfficer{}).Select("city_id, COUNT(*) AS total").
		Where("city_id IN ?", ids).Group("city_id").Scan(&offs)
	for _, s := range offs {
		offOf[s.CityID] = s.Total
	}
	occOf := map[int64]bool{}
	var occs []model.EzfyOccupy
	h.DB.Select("city_id").Where("city_id IN ? AND status = 1", ids).Find(&occs)
	for _, o := range occs {
		occOf[o.CityId] = true
	}
	// 行动槽状态（该城是否在自动行动 / 上次做了什么 / 下次什么时候动）
	slotOf := map[int64]model.EzfyBotCity{}
	var slots []model.EzfyBotCity
	h.DB.Where("bot_uid = ?", uid).Find(&slots)
	for _, s := range slots {
		slotOf[int64(s.CityId)] = s
	}
	for _, ct := range cities {
		sl := slotOf[int64(ct.ID)]
		run := 1
		if sl.ID == 0 {
			run = 1 // 槽还没建（下一轮 tick 自动补）→ 视为会行动
		} else {
			run = sl.Enabled
		}
		out = append(out, gin.H{
			"id": ct.ID, "name": ct.Name, "x": ct.X, "y": ct.Y,
			"city_level": ct.CityLevel,
			"pop":        ct.Pop, "pop_max": ct.PopMax,
			"food": ct.Food, "steel": ct.Steel, "oil": ct.Oil, "rare": ct.Rare, "gold": ct.Gold,
			"troop_total": troopOf[int64(ct.ID)],
			"officer_num": offOf[int64(ct.ID)],
			"occupied":    occOf[int64(ct.ID)],
			// 行动槽
			"auto_on":      run,
			"next_act_at":  sl.NextActAt,
			"last_act_at":  sl.LastActAt,
			"last_action":  sl.LastAction,
		})
	}
	return out
}

// AdminEzfyBotCities GET /admin/ezfy-bot/:uid/cities —— 人机名下全部城市 + 每城行动状态
//
// ★ 2026-10-28 多城市：人机没有主城概念，每座城各自发育/出兵；这里一次给出全部城的
// 人口/资源/兵力/军官数/是否被占/是否在自动行动，管理端「城市管理」tab 直接用。
func (h *EzfyAdmin) AdminEzfyBotCities(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("uid"))
	if uid <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var b model.EzfyBot
	if err := h.DB.Where("user_id = ?", uid).First(&b).Error; err != nil {
		resp.NotFound(c, "人机不存在")
		return
	}
	list := h.ezfyBotCityList(uint(uid))
	resp.OK(c, gin.H{"list": list, "total": len(list)})
}

// AdminEzfyBotCityToggle POST /admin/ezfy-bot/:uid/city/toggle {city_id, enabled}
//
// 单独停/启某座城的自动行动（人机其它城照常跑）。城被玩家占领中时不能启。
func (h *EzfyAdmin) AdminEzfyBotCityToggle(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("uid"))
	var in struct {
		CityID  int64 `json:"city_id"`
		Enabled int   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || uid <= 0 || in.CityID <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	if in.Enabled != 0 {
		in.Enabled = 1
	}
	var city model.EzfyCity
	if err := h.DB.Where("id = ? AND user_id = ?", in.CityID, uid).First(&city).Error; err != nil {
		resp.NotFound(c, "城市不存在或不属于该人机")
		return
	}
	if in.Enabled == 1 {
		var occ int64
		h.DB.Model(&model.EzfyOccupy{}).Where("city_id = ? AND status = 1", in.CityID).Count(&occ)
		if occ > 0 {
			resp.ParamError(c, "该城正被玩家占领中，无法启用自动行动")
			return
		}
	}
	// 槽可能还没建（tick 才会补）→ 先补再改
	var slot model.EzfyBotCity
	if err := h.DB.Where("city_id = ?", in.CityID).First(&slot).Error; err != nil {
		slot = model.EzfyBotCity{BotUid: uint(uid), CityId: uint(in.CityID), Enabled: in.Enabled,
			NextActAt: time.Now().UnixMilli()}
		h.DB.Create(&slot)
	} else {
		h.DB.Model(&model.EzfyBotCity{}).Where("id = ?", slot.ID).
			Updates(map[string]interface{}{"enabled": in.Enabled, "next_act_at": time.Now().UnixMilli()})
	}
	ez := &EzfyHandler{DB: h.DB, HomeDB: h.HomeDB}
	ez.ezfyBotLogCity(uint(uid), in.CityID, "城市管理",
		map[bool]string{true: "启用自动行动", false: "暂停自动行动"}[in.Enabled == 1], 1)
	resp.OK(c, gin.H{"msg": map[bool]string{true: "已启用该城自动行动", false: "已暂停该城自动行动"}[in.Enabled == 1]})
}

// AdminEzfyBotLogs GET /admin/ezfy-bot/logs?uid=&page=&size= —— 行为日志
//
// 排障利器：能看到「它上一次做了什么、为什么没动手」。
func (h *EzfyAdmin) AdminEzfyBotLogs(c *gin.Context) {
	page, offset, size := pageOf(c, 20)
	uid, _ := strconv.Atoi(c.Query("uid"))
	q := h.DB.Model(&model.EzfyBotLog{})
	if uid > 0 {
		q = q.Where("bot_uid = ?", uid)
	}
	var total int64
	q.Count(&total)
	var rows []model.EzfyBotLog
	q.Order("id DESC").Offset(offset).Limit(size).Find(&rows)

	// 批量补昵称
	uids := make([]uint, 0, len(rows))
	for _, r := range rows {
		uids = append(uids, r.BotUid)
	}
	nickOf := map[uint]string{}
	if len(uids) > 0 {
		var profs []model.EzfyProfile
		h.DB.Select("user_id, nickname").Where("user_id IN ?", uids).Find(&profs)
		for _, p := range profs {
			nickOf[p.UserID] = p.Nickname
		}
	}
	out := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		out = append(out, gin.H{
			"id": r.ID, "bot_uid": r.BotUid, "nickname": nickOf[r.BotUid],
			"action": r.Action, "detail": r.Detail, "result": r.Result,
			"created_at": r.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// AdminEzfyBotLogDelete POST /admin/ezfy-bot/log/:id/delete —— 物理删除单条行为日志
//
// ★ 2026-10-18 用户需求「人机行为日志支持管理员物理删除」：日志刷太多想清掉旧的。
func (h *EzfyAdmin) AdminEzfyBotLogDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	h.DB.Unscoped().Delete(&model.EzfyBotLog{}, id)
	resp.OK(c, gin.H{"msg": "已删除该条日志"})
}

// AdminEzfyBotLogClear POST /admin/ezfy-bot/logs/clear {bot_uid} —— 清空某个人机的全部行为日志（物理删除）
func (h *EzfyAdmin) AdminEzfyBotLogClear(c *gin.Context) {
	var in struct {
		BotUID uint `json:"bot_uid"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.BotUID <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	h.DB.Unscoped().Where("bot_uid = ?", in.BotUID).Delete(&model.EzfyBotLog{})
	resp.OK(c, gin.H{"msg": "已清空该人机的全部行为日志"})
}

// AdminEzfyBotConfig GET /admin/ezfy-bot/config —— 风控配置
func (h *EzfyAdmin) AdminEzfyBotConfig(c *gin.Context) {
	resp.OK(c, gin.H{
		"attack_on":     ezfyBotAttackOn(),
		"atk_daily_lv1": ezfyBotAtkDailyOf(1),
		"atk_daily_lv2": ezfyBotAtkDailyOf(2),
		"atk_daily_lv3": ezfyBotAtkDailyOf(3),
		"food_free":     ezfyCfg.ready() && ezfyCfg.limit.BotFoodFree != 0,
		"min_prestige":  ezfyBotMinPrestige(),
		"max_count":     ezfyBotMaxCount,
		"tick_seconds":  int(ezfyBotTickInterval.Seconds()),
		"batch_size":    ezfyBotBatchSize,
		// ★ 2026-10-18 征兵线（低于此值人机执行征兵；0/负 = 未配置，回落默认 50 万）
		"train_threshold": ezfyBotTrainThreshold(),
		// ★ 2026-10-18 人机出征时间段（空 = 默认 08:00-23:00；如 "20:00-22:00,08:00-12:00"）
		"atk_windows": ezfyBotAtkWindows(),
		// ★ 2026-10-26 「目标最小兵力」已改为**按人机分**（EzfyBot.TargetMinTroops，编辑人机里改），
		//   这里只保留全局兜底值给老版管理端显示，**不再参与目标筛选**。
		"target_min_troops": ezfyBotTargetMinTroops(),
	})
}

// ezfyBotAtkWindowsDef 人机出征时间段默认值。
//
// ★ 2026-10-26 用户规则「出征时间段 默认早上 8 点到晚上 11 点」：
// 空值**不再等于全天**，一律按 08:00-23:00 生效（要全天请显式填 00:00-23:59）。
const ezfyBotAtkWindowsDef = "08:00-23:00"

// ezfyBotAtkWindows 人机出征时间段（空 = 默认 08:00-23:00）
func ezfyBotAtkWindows() string {
	if ezfyCfg.ready() {
		if v := strings.TrimSpace(ezfyCfg.limit.BotAtkWindows); v != "" {
			return v
		}
	}
	return ezfyBotAtkWindowsDef
}

// ezfyBotTrainThreshold 人机征兵线（0/负值回落默认 50 万）
func ezfyBotTrainThreshold() int64 {
	if ezfyCfg.ready() && ezfyCfg.limit.BotTrainThreshold > 0 {
		return ezfyCfg.limit.BotTrainThreshold
	}
	return 500000
}

// AdminEzfyBotConfigSet POST /admin/ezfy-bot/config
// {attack_on, atk_daily_lv1, atk_daily_lv2, atk_daily_lv3, food_free, min_prestige, train_threshold, atk_windows}
//
// ★ 改完必须 ezfyCfg.reload：开关读的是进程内配置缓存（不 reload 要等 30s 周期刷新）。
func (h *EzfyAdmin) AdminEzfyBotConfigSet(c *gin.Context) {
	var in struct {
		AttackOn *int `json:"attack_on"`
		// ★ 2026-10-29 每日出征次数按难度分（新手/普通/精英），默认 60/100/160
		AtkDailyLv1    *int    `json:"atk_daily_lv1"`
		AtkDailyLv2    *int    `json:"atk_daily_lv2"`
		AtkDailyLv3    *int    `json:"atk_daily_lv3"`
		FoodFree       *int    `json:"food_free"`
		MinPrestige    *int    `json:"min_prestige"`
		TrainThreshold *int64  `json:"train_threshold"`
		AtkWindows     *string `json:"atk_windows"`
		// ★ 2026-10-26 自动挑目标时玩家城的最小兵力（已按人机分，见 EzfyBot.TargetMinTroops；
		//   这里只为兼容老版管理端提交，新前端不再传）
		TargetMinTroops *int64 `json:"target_min_troops"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	up := map[string]interface{}{}
	if in.AttackOn != nil {
		v := 0
		if *in.AttackOn != 0 {
			v = 1
		}
		up["bot_atk_on"] = v
	}
	// ★ 2026-10-29 每日出征次数按难度分（新手/普通/精英）：范围 1 ~ 100000。
	//   旧的全局 bot_atk_daily 已废弃不再写（一刀切没有意义）。
	for _, it := range []struct {
		col string
		val *int
	}{
		{"bot_atk_daily_lv1", in.AtkDailyLv1},
		{"bot_atk_daily_lv2", in.AtkDailyLv2},
		{"bot_atk_daily_lv3", in.AtkDailyLv3},
	} {
		if it.val == nil {
			continue
		}
		v := *it.val
		if v < 1 || v > 100000 {
			resp.ParamError(c, "每日出征次数需要在 1 ~ 100000 之间")
			return
		}
		up[it.col] = v
	}
	if in.FoodFree != nil {
		v := 0
		if *in.FoodFree != 0 {
			v = 1
		}
		up["bot_food_free"] = v
	}
	if in.MinPrestige != nil {
		v := *in.MinPrestige
		if v < 0 {
			v = 0
		}
		if v > 100000000 {
			v = 100000000
		}
		up["bot_min_prestige"] = v
	}
	if in.TrainThreshold != nil {
		v := *in.TrainThreshold
		// 过小（<1 万）视为没配 → 回落默认 50 万
		if v < 10000 {
			v = 500000
		}
		up["bot_train_threshold"] = v
	}
	if in.AtkWindows != nil {
		up["bot_atk_windows"] = strings.TrimSpace(*in.AtkWindows)
	}
	// ★ 2026-10-26 自动挑目标时玩家城最小兵力：**0 合法 = 不限**（所以不做「过小回落默认」），
	//   只拦负数；上界给个防呆值（10 亿）。
	if in.TargetMinTroops != nil {
		v := *in.TargetMinTroops
		if v < 0 {
			v = 0
		}
		if v > 1000000000 {
			v = 1000000000
		}
		up["bot_target_min_troops"] = v
	}
	if len(up) == 0 {
		resp.ParamError(c, "没有要修改的配置")
		return
	}
	if err := h.DB.Model(&model.EzfyCfgLimit{}).Where("id = 1").Updates(up).Error; err != nil {
		resp.ServerError(c, err)
		return
	}
	ezfyCfg.reload(h.DB, true)
	resp.OK(c, gin.H{"msg": "已保存"})
}

// AdminEzfyBotTargets GET /admin/ezfy-bot/:uid/targets —— 攻击目标计划列表
//
// 人机可以**同时挂多个目标**（(bot_uid,target_uid) 唯一），按顺序逐个推进状态机。
func (h *EzfyAdmin) AdminEzfyBotTargets(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("uid"))
	if uid <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var rows []model.EzfyBotTarget
	h.DB.Where("bot_uid = ? AND status IN (0,1,2,3)", uid).Order("id ASC").Limit(20).Find(&rows)
	// 目标昵称批量
	tids := make([]uint, 0, len(rows))
	for _, r := range rows {
		tids = append(tids, r.TargetUid)
	}
	nickOf := map[uint]string{}
	if len(tids) > 0 {
		var profs []model.EzfyProfile
		h.DB.Select("user_id, nickname").Where("user_id IN ?", tids).Find(&profs)
		for _, p := range profs {
			nickOf[p.UserID] = p.Nickname
		}
	}
	statusText := map[int]string{0: "待宣战", 1: "宣战待生效", 2: "可出击", 3: "冷却中", 4: "已放弃"}
	out := make([]gin.H, 0, len(rows))
	now := time.Now().UnixMilli()
	for _, r := range rows {
		out = append(out, gin.H{
			"id": r.ID, "target_uid": r.TargetUid, "nickname": nickOf[r.TargetUid],
			"status": r.Status, "status_text": statusText[r.Status],
			"note":       r.Note,
			"cool_until": r.CoolUntil,
			"cooling":    r.Status == 3 && r.CoolUntil > now,
		})
	}
	resp.OK(c, gin.H{"list": out, "total": len(out)})
}

// AdminEzfyBotOccupy GET /admin/ezfy-bot/:uid/occupy —— 人机占领的城市列表
//
// ★ 2026-10-18 人机征服玩家城市后占领记录写 ezfy_occupy（atk_user_id = 人机 uid）。
func (h *EzfyAdmin) AdminEzfyBotOccupy(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("uid"))
	if uid <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	page, offset, size := pageOf(c, 10)
	q := h.DB.Model(&model.EzfyOccupy{}).Where("atk_user_id = ?", uid)
	var total int64
	q.Count(&total)
	var rows []model.EzfyOccupy
	q.Order("id DESC").Offset(offset).Limit(size).Find(&rows)
	// 守方昵称批量
	duids := make([]uint, 0, len(rows))
	for _, r := range rows {
		duids = append(duids, r.DefUserId)
	}
	nickOf := map[uint]string{}
	if len(duids) > 0 {
		var profs []model.EzfyProfile
		h.DB.Select("user_id, nickname").Where("user_id IN ?", duids).Find(&profs)
		for _, p := range profs {
			nickOf[p.UserID] = p.Nickname
		}
	}
	// ★ 2026-10-18 待处理赎城请求数（status=0 按被赎城市聚合，管理端据此显示「同意赎回」）
	ransomCnt := map[uint]int64{}
	{
		occIds := make([]uint, 0, len(rows))
		for _, r := range rows {
			occIds = append(occIds, r.ID)
		}
		if len(occIds) > 0 {
			var rnList []struct {
				OccupyId uint
				Cnt      int64
			}
			h.DB.Model(&model.EzfyRansom{}).Select("occupy_id, COUNT(*) as cnt").
				Where("occupy_id IN ? AND status = 0", occIds).Group("occupy_id").Find(&rnList)
			for _, rn := range rnList {
				ransomCnt[rn.OccupyId] = rn.Cnt
			}
		}
	}
	out := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		st := "占领中"
		if r.Status == 2 {
			st = "已归还"
		}
		if r.Status == 3 {
			st = "已摧毁"
		}
		if r.Status == 4 {
			st = "已建城"
		}
		out = append(out, gin.H{
			"id": r.ID, "city_name": r.CityName, "x": r.X, "y": r.Y,
			"def_uid": r.DefUserId, "def_name": nickOf[r.DefUserId],
			"status": r.Status, "status_txt": st, "ransom_count": ransomCnt[r.ID],
			"created_at": r.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// adminOccupyReport 管理端替代人机操作占用城市时，给相关玩家写一条战报（对齐 newReport 简化版）。
func adminOccupyReport(db *gorm.DB, uid uint, reportType int, title, content string) {
	r := model.EzfyReport{UserID: uid, ReportType: reportType, TypeName: EzfyReportTypeName(reportType, title),
		Title: title, Content: content, IsRead: 0}
	db.Create(&r)
}

// adminRansomApprove 管理员代占领方（人机）同意赎城请求 —— 与玩家端 RansomHandle「同意」同口径：
// 事务 + 置终态兜底防重复发放，城市归还守方、占领方实收 80% 押金（20% 系统收）、流水/战报照落。
// 返回成功消息与错误；已处理(status≠0)等业务失败也走 error 返回。
func (h *EzfyAdmin) adminRansomApprove(c *gin.Context, r *model.EzfyRansom) (string, error) {
	if r.Status != 0 {
		return "", fmt.Errorf("该赎回请求已处理")
	}
	tx := h.DB.Begin()
	defer tx.Rollback()
	// 同意前重校验占领记录：已被建立/摧毁/归还(status≠1) → 请求作废 + 押金退回
	var occ model.EzfyOccupy
	if err := tx.Where("id = ? AND status = 1", r.OccupyId).First(&occ).Error; err != nil {
		if res := tx.Model(&model.EzfyRansom{}).Where("id = ? AND status = 0", r.ID).
			Update("status", 3); res.RowsAffected != 1 {
			return "", fmt.Errorf("该赎回请求已处理")
		}
		if err := tx.Model(&model.EzfyProfile{}).Where("user_id = ?", r.DefUserId).
			Update("diamond", gorm.Expr("diamond + ?", r.Cost)).Error; err != nil {
			return "", fmt.Errorf("退回押金失败：%s", err.Error())
		}
		tx.Commit()
		adminOccupyReport(h.DB, r.DefUserId, 5, "赎城押金退回",
			fmt.Sprintf("你为赎回城市[%s](%d,%d)支付的%d钻石押金已退回: 该城市已被占领方处理, 请求作废。",
				r.CityName, r.X, r.Y, r.Cost))
		return "", fmt.Errorf("该城市已被建城/摧毁/归还, 赎回请求已作废并退回押金")
	}
	// 原子置终态（锁内 + WHERE status=0 兜底）：并发重复同意等锁后必被拒，杜绝重复发放
	if res := tx.Model(&model.EzfyRansom{}).Where("id = ? AND status = 0", r.ID).
		Update("status", 1); res.RowsAffected != 1 {
		return "", fmt.Errorf("该赎回请求已处理")
	}
	// 20% 手续费：占领方实收 80%（整数向下取整）
	receive := r.Cost - r.Cost/5
	if err := tx.Model(&model.EzfyProfile{}).Where("user_id = ?", r.AtkUserId).
		Update("diamond", gorm.Expr("diamond + ?", receive)).Error; err != nil {
		return "", fmt.Errorf("钻石发放失败：%s", err.Error())
	}
	// 同意：城市返还（user_id 本仍属守方，补民心）
	var ct model.EzfyCity
	if tx.First(&ct, r.CityId).Error == nil && ct.Feelings < 20 {
		tx.Model(&model.EzfyCity{}).Where("id = ?", r.CityId).
			Updates(map[string]interface{}{"feelings": 20, "feel_settle_at": time.Now().UnixMilli()})
	}
	tx.Model(&model.EzfyOccupy{}).Where("id = ?", r.OccupyId).Update("status", 2)
	if err := tx.Commit().Error; err != nil {
		return "", fmt.Errorf("提交失败：%s", err.Error())
	}
	// 占领方钻石流水 + 双方战报
	adminOccupyRansomLog(h, r, receive)
	return fmt.Sprintf("[%s](%d,%d) 已同意赎城, 人机收到%d钻石(已扣20%%手续费), 城市已归还原主人",
		r.CityName, r.X, r.Y, receive), nil
}

// adminOccupyRansomLog 赎城成交后的钻石流水与双方战报（事务已提交，普通 DB 写）。
func adminOccupyRansomLog(h *EzfyAdmin, r *model.EzfyRansom, receive int64) {
	var bal int64
	h.DB.Model(&model.EzfyProfile{}).Where("user_id = ?", r.AtkUserId).Pluck("diamond", &bal)
	h.DB.Create(&model.EzfyDiamondLog{UserId: r.AtkUserId, Change: receive, Balance: bal,
		Reason: "赎城收入(已扣20%手续费,管理员代处理)"})
	adminOccupyReport(h.DB, r.AtkUserId, 5, "赎城成功",
		fmt.Sprintf("你同意了赎城, 收到%d钻石(已扣20%%手续费), 城市[%s](%d,%d)已归还原主人。",
			receive, r.CityName, r.X, r.Y))
	adminOccupyReport(h.DB, r.DefUserId, 5, "赎城成功",
		fmt.Sprintf("占领方已同意你的赎城, 你成功赎回城市[%s](%d,%d)! 民心已恢复。", r.CityName, r.X, r.Y))
}

// AdminEzfyBotOccupyOp POST /admin/ezfy-bot/occupy/:id/op {op: return|destroy|redeem}
//
// ★ 2026-10-18 管理员对人机占领的城市进行操作（与玩家端「附属野地」的归还/摧毁/同意赎回同口径）：
//
//	return  → 城市归还守方（user_id 过户 + 民心 + 占领记录 status=2）
//	destroy → 摧毁城市（城市/建筑/部队/军官等全删 + 地图格子恢复平原 + 占领记录 status=3）
//	redeem  → 若有原主人的待处理赎城申请，代替人机同意赎回（城市归还 + 人机收 80% 押金）
func (h *EzfyAdmin) AdminEzfyBotOccupyOp(c *gin.Context) {
	var in struct {
		Op string `json:"op"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || (in.Op != "return" && in.Op != "destroy" && in.Op != "redeem") {
		resp.ParamError(c, "参数错误")
		return
	}
	id, _ := strconv.Atoi(c.Param("id"))
	var o model.EzfyOccupy
	if err := h.DB.First(&o, id).Error; err != nil {
		resp.NotFound(c, "占领记录不存在")
		return
	}
	if o.Status != 1 {
		resp.ParamError(c, "当前状态不可操作（仅占领中可归还/摧毁/同意赎回）")
		return
	}
	switch in.Op {
	case "return":
		// 与玩家端 return 同口径：城市过户归还守方 + 民心兜底 + 占领记录置已归还
		h.DB.Model(&model.EzfyCity{}).Where("id = ?", o.CityId).
			Updates(map[string]interface{}{"user_id": o.DefUserId})
		var city model.EzfyCity
		if h.DB.First(&city, o.CityId).Error == nil && city.Feelings < 20 {
			h.DB.Model(&model.EzfyCity{}).Where("id = ?", o.CityId).
				Updates(map[string]interface{}{"feelings": 20, "feel_settle_at": time.Now().UnixMilli()})
		}
		h.DB.Model(&model.EzfyOccupy{}).Where("id = ?", o.ID).Update("status", 2)
		if o.DefUserId != 0 {
			adminOccupyReport(h.DB, o.DefUserId, 5, "城市归还",
				fmt.Sprintf("你被占领的城市[%s](%d,%d)已由敌方归还!\n民心已恢复。", o.CityName, o.X, o.Y))
		}
		resp.OK(c, gin.H{"msg": "已归还「" + o.CityName + "」给原主人"})
	case "destroy":
		// 与玩家端 destroy 同口径：城市数据连根删除 + 地图格子恢复 + 占领记录置已摧毁。
		// 装备不得按 city_id 删（归属 user_id，仅历史出生城）；本城军官装备放回背包（officer_id=0）。
		h.DB.Where("city_id = ?", o.CityId).Delete(&model.EzfyCityBuilding{})
		h.DB.Where("city_id = ?", o.CityId).Delete(&model.EzfyCityTroop{})
		h.DB.Where("city_id = ?", o.CityId).Delete(&model.EzfyCityTech{})
		h.DB.Where("city_id = ?", o.CityId).Delete(&model.EzfyTrainQueue{})
		h.DB.Where("city_id = ?", o.CityId).Delete(&model.EzfyWildland{})
		h.DB.Where("city_id = ?", o.CityId).Delete(&model.EzfyWounded{})
		h.DB.Where("city_id = ?", o.CityId).Delete(&model.EzfyCityTarget{})
		h.DB.Where("city_id = ?", o.CityId).Delete(&model.EzfyCityEffect{})
		h.DB.Where("city_id = ?", o.CityId).Delete(&model.EzfyOrder{})
		h.DB.Model(&model.EzfyEquipment{}).
			Where("officer_id IN (?)", h.DB.Model(&model.EzfyOfficer{}).Select("id").Where("city_id = ?", o.CityId)).
			Update("officer_id", 0)
		h.DB.Where("city_id = ?", o.CityId).Delete(&model.EzfyOfficer{})
		h.DB.Where("x = ? AND y = ?", o.X, o.Y).Delete(&model.EzfyMapArea{})
		h.DB.Delete(&model.EzfyCity{}, o.CityId)
		h.DB.Model(&model.EzfyOccupy{}).Where("id = ?", o.ID).Update("status", 3)
		adminOccupyReport(h.DB, o.DefUserId, 5, "城市被摧毁",
			fmt.Sprintf("你被占领的城市[%s](%d,%d)已被敌方摧毁, 该坐标恢复为普通平原。", o.CityName, o.X, o.Y))
		resp.OK(c, gin.H{"msg": "已摧毁城市「" + o.CityName + "」，坐标恢复平原"})
	case "redeem":
		var r model.EzfyRansom
		if err := h.DB.Where("city_id = ? AND occupy_id = ? AND status = 0", o.CityId, o.ID).First(&r).Error; err != nil {
			resp.ParamError(c, "该城市暂无待处理的赎城申请")
			return
		}
		msg, err := h.adminRansomApprove(c, &r)
		if err != nil {
			resp.ParamError(c, err.Error())
			return
		}
		resp.OK(c, gin.H{"msg": msg})
	}
}

// AdminEzfyBotOccupyRedeemAll POST /admin/ezfy-bot/occupy/redeem/all {uid} —— 统一赎回
//
// ★ 2026-10-18 一次同意该人机名下**全部**待处理赎城申请（每单与玩家端同意同口径），
// 返回成功/跳过/失败统计。
func (h *EzfyAdmin) AdminEzfyBotOccupyRedeemAll(c *gin.Context) {
	var in struct {
		UID uint `json:"uid"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.UID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var ransoms []model.EzfyRansom
	h.DB.Where("atk_user_id = ? AND status = 0", in.UID).Order("id ASC").Find(&ransoms)
	if len(ransoms) == 0 {
		resp.OK(c, gin.H{"msg": "该人机暂无待处理的赎城申请", "ok": 0, "skip": 0, "fail": 0})
		return
	}
	ok, skip, fail := 0, 0, 0
	failMsgs := []string{}
	for i := range ransoms {
		r := &ransoms[i]
		msg, err := h.adminRansomApprove(c, r)
		if err != nil {
			if strings.Contains(err.Error(), "已处理") {
				skip++
			} else {
				fail++
				failMsgs = append(failMsgs, fmt.Sprintf("[%s](%d,%d)：%s", r.CityName, r.X, r.Y, err.Error()))
			}
			continue
		}
		_ = msg
		ok++
	}
	resp.OK(c, gin.H{"msg": fmt.Sprintf("统一赎回完成：成功 %d 单，跳过 %d，失败 %d", ok, skip, fail),
		"ok": ok, "skip": skip, "fail": fail, "fails": failMsgs})
}

// AdminEzfyBotTargetSet POST /admin/ezfy-bot/:uid/target {target_uid} —— 管理端指定攻击目标
//
// ★ 与「自己找目标」并存：指定的目标直接写入计划表（status=0 待宣战），
//
//	人机下一轮自动走「宣战 → 等 24h 生效 → 出击」；发完把 next_act_at 提前，立刻开始。
//
// ★ 目标必须是**真人**（is_bot=0）—— 人机不打人机。
func (h *EzfyAdmin) AdminEzfyBotTargetSet(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("uid"))
	if uid <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var in struct {
		TargetUID uint `json:"target_uid"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.TargetUID == 0 {
		resp.ParamError(c, "请填写目标玩家ID或游戏号")
		return
	}
	var b model.EzfyBot
	if err := h.DB.Where("user_id = ?", uid).First(&b).Error; err != nil {
		resp.NotFound(c, "人机不存在")
		return
	}
	if uint(in.TargetUID) == uint(uid) {
		resp.ParamError(c, "不能指定自己")
		return
	}
	// 目标：支持用户ID或游戏号，必须是真人
	var tp model.EzfyProfile
	if err := h.DB.Where("user_id = ? OR game_uid = ?", in.TargetUID, in.TargetUID).First(&tp).Error; err != nil {
		resp.NotFound(c, "目标玩家不存在")
		return
	}
	if tp.IsBot == 1 {
		resp.ParamError(c, "目标是人机，人机不打人机")
		return
	}
	// ★ 2026-10-17 管理端手动指定**不受最低声望门槛限制**（运营操作：管理员说打谁就打谁）
	// ★★ 2026-10-17 强制开打修复：不再有「已在计划中」拒绝 —— 无论旧计划是什么状态
	//    （活跃 0/1/2、冷却 3、放弃 4），一律**复活重置**（status=0、清宣战生效/冷却），
	//    管理员每次下达都会立刻生效：人机下一轮重新宣战出击。
	// 目标当前城
	var tc model.EzfyCity
	h.DB.Where("user_id = ?", tp.UserID).Order("id ASC").First(&tc)
	note := fmt.Sprintf("管理端指定 %s(%d,%d)", tp.Nickname, tc.X, tc.Y)
	// ★ 2026-10-17 唯一键是 (bot_uid,target_uid) 不分状态：存在旧行时若直接 Create 会撞唯一键。
	//   统一走**复活旧行**（status 拉回 0、更新目标城与备注、清生效/冷却），复用而非新建。
	var old = model.EzfyBotTarget{}
	err := h.DB.Where("bot_uid = ? AND target_uid = ?", uid, tp.UserID).First(&old).Error
	if err == nil {
		h.DB.Model(&model.EzfyBotTarget{}).Where("id = ?", old.ID).
			Updates(map[string]interface{}{"status": 0, "target_city_id": int64(tc.ID), "note": note, "effect_time": 0, "cool_until": 0})
	} else if err := h.DB.Create(&model.EzfyBotTarget{
		BotUid: uint(uid), TargetUid: tp.UserID, TargetCityId: int64(tc.ID),
		Status: 0, Note: note,
	}).Error; err != nil {
		resp.ParamError(c, "写入计划失败（可能已存在）")
		return
	}
	h.DB.Model(&model.EzfyBot{}).Where("user_id = ?", uid).Update("next_act_at", time.Now().UnixMilli())
	ez := &EzfyHandler{DB: h.DB, HomeDB: h.HomeDB}
	ez.ezfyBotLog(uint(uid), "管理端指定目标", fmt.Sprintf("目标 %s(ID:%d)，下一轮自动宣战", tp.Nickname, tp.UserID), 1)
	resp.OK(c, gin.H{"msg": fmt.Sprintf("已把「%s」加入「%s」的攻击计划，它下一轮会自动宣战（24h 后生效）", tp.Nickname, ez.ezfyBotNickOf(uint(uid)))})
}

// AdminEzfyBotTargetDel DELETE /admin/ezfy-bot/target/:tid —— 删除一条攻击计划
func (h *EzfyAdmin) AdminEzfyBotTargetDel(c *gin.Context) {
	tid, _ := strconv.Atoi(c.Param("tid"))
	if tid <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	res := h.DB.Delete(&model.EzfyBotTarget{}, tid)
	if res.RowsAffected == 0 {
		resp.NotFound(c, "计划不存在")
		return
	}
	resp.OK(c, gin.H{"msg": "已删除该攻击计划"})
}

// ezBotPopFree 管理端算人机城的空闲人口（口径与玩家端 ezfyPopFree 完全一致）。
func ezBotPopFree(h *EzfyAdmin, city model.EzfyCity) int64 {
	ez := &EzfyHandler{DB: h.DB, HomeDB: h.HomeDB}
	return ezfyPopFree(city.Pop, ez.buildingPop(&city), ez.troopPop(city.ID), city.PopRecover, city.PopExtra)
}

// AdminEzfyBotAttackNearest POST /admin/ezfy-bot/:uid/attack-nearest —— 出征最近的敌对真人城市
//
// ★ 2026-10-17 用户需求：管理端一键让人机打「最近的真人敌对城」（非自己、非同盟）。
//
//	写入攻击计划(status=0 待宣战) + 置 next_act_at=now，人机下一轮自动 宣战→生效→出击。
func (h *EzfyAdmin) AdminEzfyBotAttackNearest(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("uid"))
	if uid <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var b model.EzfyBot
	if err := h.DB.Where("user_id = ?", uid).First(&b).Error; err != nil {
		resp.NotFound(c, "人机不存在")
		return
	}
	// ★ 2026-10-28 多城市：从哪座城出发找最近目标（?city_id=，不传 = 当前城）
	wantCity, _ := strconv.Atoi(c.Query("city_id"))
	botCity := h.ezfyBotPickCity(uint(uid), int64(wantCity))
	if botCity.ID == 0 {
		resp.NotFound(c, "人机城市不存在")
		return
	}
	// ★ 2026-10-17 修复：①真人候选直接用 JOIN 过滤（ezfy_profile.is_bot=0），
	//   不能用 `user_id IN (?万uid)` —— 全服真人上万就超 MySQL 占位符上限(65535) 报错；
	//   ② 同盟判断改为「预读军团映射 + 内存比对」，避免对每个候选城各查 2 次库（几十万城 = 超时）。
	var members []model.EzfyCorpsMember
	h.DB.Select("user_id, corps_id").Find(&members)
	corpsMap := make(map[uint]uint, len(members))
	for _, m := range members {
		corpsMap[m.UserId] = m.CorpsId
	}
	myCorps := corpsMap[uint(uid)] // 无军团时为 0，不会与真实军团号(≥1)匹配 → 天然非同盟
	type cand struct {
		ID       uint
		UserID   uint
		Name     string
		X, Y     int
		Prestige int
	}
	var cities []cand
	if err := h.DB.Model(&model.EzfyCity{}).
		Select("ezfy_city.id, ezfy_city.user_id, ezfy_city.name, ezfy_city.x, ezfy_city.y, p.prestige").
		Joins("JOIN ezfy_profile p ON p.user_id = ezfy_city.user_id AND p.is_bot = 0").
		Find(&cities).Error; err != nil {
		resp.ServerError(c, err)
		return
	}
	if len(cities) == 0 {
		resp.NotFound(c, "没有可攻击的敌对城市")
		return
	}
	// ★ 2026-10-17 管理端手动「出征最近敌对城」不受最低声望门槛限制（运营操作：管理员说打谁就打谁）
	best := model.EzfyCity{}
	bestDist := int64(1 << 62)
	ez := &EzfyHandler{DB: h.DB, HomeDB: h.HomeDB}
	for _, cty := range cities {
		if cty.UserID == uint(uid) {
			continue
		}
		// 同盟 = 双方都有军团且军团号相同（与 isAllyCity 同语义）
		if cid, ok := corpsMap[cty.UserID]; ok && cid == myCorps {
			continue
		}
		dx, dy := int64(cty.X-botCity.X), int64(cty.Y-botCity.Y)
		if d := dx*dx + dy*dy; d < bestDist {
			bestDist = d
			best = model.EzfyCity{ID: cty.ID, UserID: cty.UserID, Name: cty.Name, X: cty.X, Y: cty.Y}
		}
	}
	if best.ID == 0 {
		resp.NotFound(c, "没有可攻击的敌对城市")
		return
	}
	// ★★ 2026-10-17 强制开打修复：不再有「已在计划中」拒绝 —— 无论旧计划什么状态一律复活重置
	note := fmt.Sprintf("最近敌对 %s(%d,%d)", best.Name, best.X, best.Y)
	// ★ 2026-10-17 同 AdminEzfyBotTargetSet：存在旧行时复活复用，避免撞 (bot_uid,target_uid) 唯一键
	var old = model.EzfyBotTarget{}
	err := h.DB.Where("bot_uid = ? AND target_uid = ?", uid, best.UserID).First(&old).Error
	if err == nil {
		h.DB.Model(&model.EzfyBotTarget{}).Where("id = ?", old.ID).
			Updates(map[string]interface{}{"status": 0, "target_city_id": int64(best.ID), "note": note, "effect_time": 0, "cool_until": 0})
	} else if err := h.DB.Create(&model.EzfyBotTarget{
		BotUid: uint(uid), TargetUid: best.UserID, TargetCityId: int64(best.ID),
		Status: 0, Note: note,
	}).Error; err != nil {
		resp.ParamError(c, "写入计划失败（可能已存在）")
		return
	}
	h.DB.Model(&model.EzfyBot{}).Where("user_id = ?", uid).Update("next_act_at", time.Now().UnixMilli())
	ez.ezfyBotLog(uint(uid), "锁定最近敌对", fmt.Sprintf("锁定最近敌对城 %s(%d,%d)，下一轮自动宣战", best.Name, best.X, best.Y), 1)
	dist := int(math.Sqrt(float64(bestDist)))
	resp.OK(c, gin.H{"msg": fmt.Sprintf("已锁定最近的敌对城「%s」(距%d格)，人机下一轮自动宣战出击", best.Name, dist)})
}

// AdminEzfyBotRelocate POST /admin/ezfy-bot/:uid/relocate {x, y} —— 管理端给人机迁城
//
// ★ 2026-10-17 用户需求：输入目标坐标直接迁，复用玩家迁城 checkMoveTarget 校验
//
//	（坐标 1~499、平原/沿海平原、该坐标无城市），省去等迁城计划。
func (h *EzfyAdmin) AdminEzfyBotRelocate(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("uid"))
	if uid <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var in struct {
		X int `json:"x"`
		Y int `json:"y"`
		// ★ 2026-10-28 多城市：迁哪座城（0/不传 = 当前城）
		CityID int64 `json:"city_id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	ez := &EzfyHandler{DB: h.DB, HomeDB: h.HomeDB}
	if msg := ez.checkMoveTarget(in.X, in.Y, false); msg != "" {
		resp.ParamError(c, msg)
		return
	}
	var b model.EzfyBot
	if err := h.DB.Where("user_id = ?", uid).First(&b).Error; err != nil {
		resp.NotFound(c, "人机不存在")
		return
	}
	ct := h.ezfyBotPickCity(uint(uid), in.CityID)
	if ct.ID == 0 {
		resp.NotFound(c, "人机城市不存在")
		return
	}
	old := fmt.Sprintf("%s(%d,%d)", ct.Name, ct.X, ct.Y)
	if err := h.DB.Model(&model.EzfyCity{}).Where("id = ?", ct.ID).
		Updates(map[string]interface{}{"x": in.X, "y": in.Y}).Error; err != nil {
		resp.ServerError(c, err)
		return
	}
	h.DB.Model(&model.EzfyBot{}).Where("user_id = ?", uid).Update("next_act_at", time.Now().UnixMilli())
	ez.ezfyBotLogCity(uint(uid), int64(ct.ID), "管理端迁城", fmt.Sprintf("迁城 %s → (%d,%d)", old, in.X, in.Y), 1)
	resp.OK(c, gin.H{"msg": fmt.Sprintf("已把「%s」迁至 (%d,%d)", ct.Name, in.X, in.Y)})
}
