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
	if err := h.DB.Create(&model.EzfyProfile{
		UserID: uid, GameUID: int64(uid), Nickname: nick,
		Prestige: 0, Camp: camp, Rank: 1, IsBot: 1,
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
	// 运营初始化（方案 4.7「开服调参」）：可选「加速开局」——给一笔启动资源，
	// 否则人机要花很久才能攒够「召集人口」的 20 万粮（从零开始是默认档）。
	if boost {
		h.DB.Model(&model.EzfyCity{}).Where("id = ?", city.ID).Updates(map[string]interface{}{
			"food": 500000, "steel": 200000, "oil": 200000, "rare": 200000,
		})
	}

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
func (h *EzfyHandler) botReschedule(b model.EzfyBot, cfg ezfyBotLevelCfg, action, detail string, ok bool) {
	gap := cfg.GrowMin + mathrand.Intn(cfg.GrowMax-cfg.GrowMin+1)
	if !ok {
		gap = 2 + mathrand.Intn(3) // 失败/跳过：2~4 分钟后重试
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

	// 0) 保命：部队打光了，先造兵
	if troopTotal == 0 && buildingLv[14] > 0 {
		if a, d, ok := h.botTrain(city, 0, cfg); ok {
			return a, d, ok
		}
	}

	// 1) 人口：空闲人口低于人口上限的 40% → 召集
	//
	// ★ 2026-10-17 实测修正：召集一次固定 20 万粮、给 5 万人口，但会被**民居人口上限**截断。
	//   1 级民居上限只有 100 → 花 20 万粮只换 100 人口，极亏。
	//   所以要求民居先升到 2 级（pop_max ≥ 300）才召集；否则跳过，让建筑步骤继续升民居。
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

	// 3) 兵
	if a, d, ok := h.botTrain(city, troopTotal, cfg); ok {
		return a, d, ok
	}

	// 4) 军事（总开关 / 允许攻击玩家 / 每日节流）
	if a, d, ok := h.botMilitary(b, city, troopTotal, cfg); ok {
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

// botTrain 造兵：兵力低于目标就造步兵（Require 只有「军工厂1级」，最稳）。
//
// troopTotal = 当前总兵力（保命调用时传 0 表示「刚被打光」）。
func (h *EzfyHandler) botTrain(city *model.EzfyCity, troopTotal int64, cfg ezfyBotLevelCfg) (string, string, bool) {
	if troopTotal >= cfg.TroopTarget {
		return "", "", false
	}
	t := ezfyCfg.troop(1) // 步兵
	if t == nil {
		return "", "", false
	}
	// 军工厂等级（步兵 Require = 军工厂(1级)）
	var fac model.EzfyCityBuilding
	if err := h.DB.Where("city_id = ? AND building_id = 14 AND level > 0", city.ID).First(&fac).Error; err != nil {
		return "", "", false
	}
	gap := cfg.TroopTarget - troopTotal
	if gap <= 0 {
		return "", "", false
	}
	// ★ 2026-10-17 人口门槛：没人口直接跳过（不然每次发请求都吃「人口不足」失败，白烧往返）。
	if int64(city.Pop) < int64(t.Pop) {
		return "", "", false
	}
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
	// 人口上限（每个兵占 t.Pop 人口）
	if t.Pop > 0 {
		if n := int64(city.Pop) / int64(t.Pop); count > n {
			count = n
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
		map[string]interface{}{"city_id": int64(city.ID), "troop_id": 1, "count": int(count)})
	if code == 0 {
		return "造兵", fmt.Sprintf("训练步兵 ×%d（当前 %d/%d）", count, troopTotal, cfg.TroopTarget), true
	}
	return "造兵", "训练失败: " + msg, false
}

// ezfyBotCanPay 资源是否够（本地判断，避免每次都发请求吃失败日志）。
func ezfyBotCanPay(city *model.EzfyCity, food, steel, oil, rare, gold int64) bool {
	return city.Food >= food && city.Steel >= steel && city.Oil >= oil &&
		city.Rare >= rare && city.Gold >= gold
}

// ============ 军事 AI ============

// botMilitary 军事：计划表驱动的「宣战 → 等 24h → 出击」。
//
// 硬约束：打玩家城必须先宣战，宣战 24h 生效、48h 有效期 → 不能看到谁就打谁。
// 开关：ezfy_cfg_limit 的 bot_atk_on（管理端「人机管理 → 允许攻击玩家」）。
func (h *EzfyHandler) botMilitary(b *model.EzfyBot, city *model.EzfyCity, troopTotal int64, cfg ezfyBotLevelCfg) (string, string, bool) {
	if !ezfyBotAttackOn() {
		return "", "", false
	}
	// 兵力不足目标的一半 → 不打仗，专心发育
	if troopTotal < cfg.TroopTarget/2 || troopTotal < 200 {
		return "", "", false
	}
	// 每日节流
	if !h.botAtkQuotaOK(b) {
		return "", "", false
	}
	now := time.Now().UnixMilli()

	// ① 已有计划：推进状态
	var plans []model.EzfyBotTarget
	h.DB.Where("bot_uid = ? AND status IN (0,1,2)", b.UserID).Order("id ASC").Limit(3).Find(&plans)
	for _, p := range plans {
		switch p.Status {
		case 1: // 已宣战，等生效
			if now >= p.EffectTime {
				h.DB.Model(&model.EzfyBotTarget{}).Where("id = ?", p.ID).Update("status", 2)
				return "宣战生效", fmt.Sprintf("对目标 %d 的宣战已生效，准备出击", p.TargetUid), true
			}
			continue
		case 2: // 可出击
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
			// 宣战失败（不需要宣战/同盟/已有宣战）→ 放弃该计划
			h.DB.Model(&model.EzfyBotTarget{}).Where("id = ?", p.ID).Update("status", 4)
			return "宣战失败", msg, false
		}
	}

	// ② 没有计划 → 找目标（坐标区间走索引，绝不遍历全图）
	if tgt, ok := h.botPickTarget(city, troopTotal, cfg); ok {
		if err := h.DB.Create(&model.EzfyBotTarget{
			BotUid: b.UserID, TargetUid: tgt.UserID, TargetCityId: int64(tgt.CityID),
			Status: 0, Note: fmt.Sprintf("%s(%d,%d) 兵力%d", tgt.Name, tgt.X, tgt.Y, tgt.Troops),
		}).Error; err != nil {
			return "", "", false // 唯一键冲突（已有计划）→ 下轮再说
		}
		return "锁定目标", fmt.Sprintf("锁定 %s(%d,%d) 兵力%d", tgt.Name, tgt.X, tgt.Y, tgt.Troops), true
	}
	return "", "", false
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
//   - 目标兵力 < 自己兵力 × 0.6（打得过才打）
//   - 无生效中的免战保护（ezfy_city_effect effect_type=2）
//   - 城建时间 > 3 天（新手保护）
//   - 不在已放弃的冷却名单里
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
  AND COALESCE(t.troops, 0) BETWEEN 1 AND ?
  AND NOT EXISTS (SELECT 1 FROM ezfy_city_effect e
                  WHERE e.city_id = c.id AND e.effect_type = 2 AND e.until_time > ?)
  AND NOT EXISTS (SELECT 1 FROM ezfy_bot_target bt
                  WHERE bt.bot_uid = ? AND bt.target_uid = c.user_id AND bt.status = 3 AND bt.cool_until > ?)
ORDER BY COALESCE(t.troops, 0) ASC
LIMIT 1`,
		city.UserID, city.X-25, city.X+25, city.Y-25, city.Y+25,
		newbieBefore, limit, now, city.UserID, now).Scan(&r).Error
	if err != nil || r.CityID == 0 {
		return r, false
	}
	return r, true
}

// botAttack 出击：派 60% 兵力掠夺目标城。
func (h *EzfyHandler) botAttack(b *model.EzfyBot, city *model.EzfyCity, p model.EzfyBotTarget, troopTotal int64) (string, string, bool) {
	var troops []model.EzfyCityTroop
	h.DB.Where("city_id = ? AND count > 0", city.ID).Find(&troops)
	groups := make([]ezfyUnitGroup, 0, len(troops))
	for _, t := range troops {
		n := t.Count * 6 / 10
		if n <= 0 {
			continue
		}
		groups = append(groups, ezfyUnitGroup{TroopId: int(t.TroopId), Count: n})
	}
	if len(groups) == 0 {
		return "", "", false
	}
	var tc model.EzfyCity
	if err := h.DB.First(&tc, p.TargetCityId).Error; err != nil {
		h.DB.Model(&model.EzfyBotTarget{}).Where("id = ?", p.ID).Update("status", 4)
		return "出击放弃", "目标城不存在", false
	}
	code, msg := h.botCall(b.UserID, "", "POST", "/games/ezfy/order", map[string]interface{}{
		"city_id": int64(city.ID), "order_type": 2, // 掠夺
		"target_type": 3, "target_id": int64(tc.ID),
		"target_x": tc.X, "target_y": tc.Y,
		"troops": groups, "auto_battle": 1,
	})
	if code == 0 {
		// 冷却 12 小时 + 记一次出征
		h.DB.Model(&model.EzfyBotTarget{}).Where("id = ?", p.ID).
			Updates(map[string]interface{}{"status": 3, "cool_until": time.Now().Add(12 * time.Hour).UnixMilli()})
		h.botAtkCountInc(b)
		return "出击", fmt.Sprintf("掠夺 %s(%d,%d) 派兵 %d 支", p.Note, tc.X, tc.Y, len(groups)), true
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

// ezfyBotDailyAtkCap 每个人机每日出征次数上限（ezfy_cfg_limit.bot_atk_daily，默认 3）。
func ezfyBotDailyAtkCap() int {
	if n := ezfyCfg.limit.BotAtkDaily; n > 0 {
		return n
	}
	return 3
}

// ezfyBotFoodFree 人机军队免耗粮开关（ezfy_cfg_limit.bot_food_free）。
//
// ★ 只对 city.IsBot=1 的城生效（调用方保证），真人**永远照常扣粮**；
//   判断只读内存 city 字段 + 配置缓存 —— calcResource 红线路径零查询。
//   0 是合法值（默认正常耗粮），不能用 ezfyLimitOr。
func ezfyBotFoodFree(isBot int) bool {
	if isBot == 0 {
		return false
	}
	if !ezfyCfg.ready() {
		return false
	}
	return ezfyCfg.limit.BotFoodFree != 0
}

// botAtkQuotaOK 今日出征次数是否还没到上限（跨天自动清零）。
func (h *EzfyHandler) botAtkQuotaOK(b *model.EzfyBot) bool {
	today := time.Now().Format("20060102")
	cnt := b.AtkCountToday
	if b.AtkDate != today {
		cnt = 0
	}
	return cnt < ezfyBotDailyAtkCap()
}

// botAtkCountInc 今日出征次数 +1（跨天重置）。
func (h *EzfyHandler) botAtkCountInc(b *model.EzfyBot) {
	today := time.Now().Format("20060102")
	cnt := 1
	if b.AtkDate == today {
		cnt = b.AtkCountToday + 1
	}
	h.DB.Model(&model.EzfyBot{}).Where("id = ?", b.ID).
		Updates(map[string]interface{}{"atk_count_today": cnt, "atk_date": today})
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
