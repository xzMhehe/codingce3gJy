package handler

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// ============ 战场指挥室（实时指挥） ============
//
// ★ 2026-09-22 用户要求：「实现指挥功能」，入口在 军情 → 军队动态 → [指挥]。
//
// 玩法（复刻《战斗机制（家园玩家必看）》§1）：
//   - 部队到达目标后**不立即结算**，而是开一场战场；
//   - 每回合 30 秒，前 25 秒可下达「前进 / 暂停 / 后退」，最后 5 秒锁定并由服务器结算；
//   - 最多 40 回合，一方全灭或回合耗尽即结束；
//   - 结束后把结果回写订单，走原有的战后结算（战报/掠夺/经验/征服全部复用）。
//
// 实现要点：**懒结算**。不常驻定时器，每次请求按「已经过去多少个 30 秒」补算回合。
// 好处是玩家离线也不会卡死（默认指令前进，战斗自然推进），代价是战斗结束的
// 时刻由「下一次请求」决定 —— 这与本游戏其它系统（资源产出、建筑完工）口径一致。

const (
	ezfyBattleRoundMs = int64(30000) // 每回合 30 秒
	ezfyBattleCmdMs   = int64(25000) // 前 25 秒可下达指令，最后 5 秒锁定
	// ezfyBattleLogTail 下发给前端的日志条数（日志会一直累积，全量下发会拖慢轮询）
	ezfyBattleLogTail = 40
	// ezfyBattleLogMax 战场快照里保留的日志条数上限
	//
	// ★ 性能（服务器只有 1 核）：快照是整段 JSON 存库的，每次推进都要「反序列化 → 序列化」，
	// 日志无限增长会让这个开销线性变大。正常 40 回合远达不到这个上限，
	// 这里只防异常情况下快照无限膨胀。
	ezfyBattleLogMax = 1000
	// ezfyOrderStatusBattle 订单状态：战斗中（指挥室进行中，等玩家指挥）
	// 0 行进 / 1 驻守中 / 2 返回 / 3 完成 / 4 阵亡 / 5 战斗中 / 6 等待
	ezfyOrderStatusBattle = 5
	// ezfyOrderStatusWaiting 订单状态：等待（目标已被别的玩家抢先开始指挥，排队等上一场打完）
	ezfyOrderStatusWaiting = 6
)

// ============ 攻方逐兵种指令表 ============
//
// ★ 用户要求（2026-09-22）：「指挥不是指挥全部，自己带的兵种都能指挥，就是单独指挥」。
// 存法：JSON `{"1":"advance","3":"hold"}`（troopId → 指令）。
// 没给的兵种 → 回落司令部「兵种战斗配置」；键 0 = 旧格式遗留的「全军统一指令」。

// ezfyBattleCmdName 指令中文名（空串 = 沿用司令部配置）
func ezfyBattleCmdName(cmd string) string {
	switch cmd {
	case ezfyCmdAdvance:
		return "前进"
	case ezfyCmdHold:
		return "停止"
	case ezfyCmdRetreat:
		return "后退"
	}
	return "默认"
}

// ezfyAtkCmdsParse 解析攻方逐兵种指令表（兼容旧格式：整串 = 全军统一指令）
func ezfyAtkCmdsParse(raw string) map[int]string {
	out := map[int]string{}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return out
	}
	if raw == ezfyCmdAdvance || raw == ezfyCmdHold || raw == ezfyCmdRetreat {
		out[0] = raw // 旧格式：全军统一
		return out
	}
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

// ezfyAtkCmdsEncode 序列化逐兵种指令表
func ezfyAtkCmdsEncode(m map[int]string) string {
	b, err := json.Marshal(m)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// ezfyAtkCmdOf 取某兵种的指令（0 号键 = 旧格式的全军统一指令）
func ezfyAtkCmdOf(m map[int]string, troopId int) string {
	if v, ok := m[troopId]; ok {
		return v
	}
	if v, ok := m[0]; ok {
		return v
	}
	return ""
}

// ezfyBattleByOrder 取订单对应的战场（同一订单只保留最新一场）
func (h *EzfyHandler) ezfyBattleByOrder(orderId int64) *model.EzfyBattle {
	var b model.EzfyBattle
	if err := h.DB.Where("order_id = ?", orderId).Order("id DESC").First(&b).Error; err != nil {
		return nil
	}
	return &b
}

// ezfyBattleStart 为订单开一场战场（幂等：已有进行中的直接返回）
func (h *EzfyHandler) ezfyBattleStart(uid uint, order *model.EzfyOrder,
	st *ezfyBattleState, targetName string, now int64) *model.EzfyBattle {

	if b := h.ezfyBattleByOrder(int64(order.ID)); b != nil && b.Status == 1 {
		return b
	}
	b := &model.EzfyBattle{
		OrderId: int64(order.ID), UserID: uid, CityId: order.CityId,
		TargetType: order.TargetType, TargetId: order.TargetId,
		TargetX: order.TargetX, TargetY: order.TargetY, TargetName: targetName,
		// AtkCmd 留空 = 各兵种沿用司令部「兵种战斗配置」（玩家在司令部设过「停止」的兵种不该被强制前进）
		Status: 1, Round: 0, Win: 0, AtkCmd: "",
		State:      ezfyBattleSnapshotEncode(st.Snapshot()),
		RoundStart: now,
	}
	if err := h.DB.Create(b).Error; err != nil {
		return nil
	}
	return b
}

// ezfyBattleTick 懒推进：把「已经到点」的回合逐回合结算掉。
//
// 返回最新快照 + 是否已结束。守方不下指令（按司令部兵种配置行动），
// 攻方用 b.AtkCmd（玩家在指令窗口内下达的）。
func (h *EzfyHandler) ezfyBattleTick(b *model.EzfyBattle, now int64) (ezfyBattleSnapshot, bool) {
	snap, ok := ezfyBattleSnapshotDecode(b.State)
	if !ok {
		return ezfyBattleSnapshot{}, true
	}
	if snap.Done || b.Status != 1 {
		return snap, true
	}
	st := ezfyBattleStateFromSnapshot(snap)

	// 逐兵种指令表解析一次，整个补算过程复用
	cmds := ezfyAtkCmdsParse(b.AtkCmd)

	steps := 0
	for !st.Done && now-b.RoundStart >= ezfyBattleRoundMs {
		st.Step(cmds, "")
		b.RoundStart += ezfyBattleRoundMs
		steps++
		// 防御性上限：即使时间戳异常（比如系统时钟跳变）也绝不死循环
		if steps > ezfyBattleMaxRounds+1 {
			st.Done = true
			break
		}
	}
	if st.Done {
		b.Status = 2
		if st.AttackerWin {
			b.Win = 1
		} else {
			b.Win = 2
		}
	}
	if steps > 0 || st.Done {
		b.Round = st.Round
		b.State = ezfyBattleSnapshotEncode(st.Snapshot())
		h.DB.Model(&model.EzfyBattle{}).Where("id = ?", b.ID).Updates(map[string]interface{}{
			"round": b.Round, "state": b.State, "status": b.Status,
			"win": b.Win, "round_start": b.RoundStart,
		})
	}
	out, _ := ezfyBattleSnapshotDecode(b.State)
	return out, st.Done
}

// ezfyBattleFinishToOrder 把战场结果回写订单，让 processArrive 走常规结算。
//
// ★ 这是「不重复实现一套战后逻辑」的关键：战场只负责**打出结果**，
// 战报/掠夺资源/军官经验/征服占领仍然由 processArrive 里那一大段统一处理。
// 返回写入订单的 battle_result（调用方可以直接塞回内存里的订单，省一次查库）。
func (h *EzfyHandler) ezfyBattleFinishToOrder(b *model.EzfyBattle, now int64) string {
	snap, ok := ezfyBattleSnapshotDecode(b.State)
	if !ok {
		return ""
	}
	st := ezfyBattleStateFromSnapshot(snap)
	payload, err := json.Marshal(st.Result())
	if err != nil {
		return ""
	}
	// ★★ 只写 battle_result 与 status，**绝不能动 arrive_time**：
	//   `arrive_time - start_time` 是「单程行军时长」的计算基准（返航时长也用它），
	//   拨到 now 会让返航时间算出天文数字（线上出现过「20717 天才能回来」）。
	//   结算改由 processOrders 直接调 processArrive 触发（见那里的 status==5 分支）。
	h.DB.Model(&model.EzfyOrder{}).Where("id = ?", b.OrderId).Updates(map[string]interface{}{
		"battle_result": string(payload),
		"status":        0,
	})
	return string(payload)
}

// ezfyBattleResultDecode 解析订单上回写的战场结果
func ezfyBattleResultDecode(s string) (ezfyBattleResult, bool) {
	res := ezfyBattleResult{}
	if s == "" {
		return res, false
	}
	if err := json.Unmarshal([]byte(s), &res); err != nil {
		return res, false
	}
	return res, true
}

// ezfyBattleView 下发给前端的战场视图
func (h *EzfyHandler) ezfyBattleView(b *model.EzfyBattle, snap ezfyBattleSnapshot, now int64, atkCamp int) gin.H {
	// 本回合剩余时间：过了就是 0（等待下一次请求推进）
	left := b.RoundStart + ezfyBattleRoundMs - now
	if left < 0 {
		left = 0
	}
	phase := "lock" // 锁定结算期
	if left > ezfyBattleRoundMs-ezfyBattleCmdMs {
		phase = "cmd" // 指令期（前 25 秒）
	}

	// ★ 逐兵种指令：攻方每个兵种各自带自己的指令，前端每行单独显示/下达
	cmds := ezfyAtkCmdsParse(b.AtkCmd)

	// ★ 逐兵种「优先攻击目标」（2026-09-23 用户要求）：默认来自司令部兵种战斗配置
	//   （开战时抄进快照的 AtkTargets），玩家在指挥室可逐兵种改。
	tgtOf := func(troopId int) int {
		if snap.AtkTargets == nil {
			return 0
		}
		return snap.AtkTargets[troopId]
	}
	troopName := func(id int) string {
		if id == 0 {
			return "最近目标"
		}
		if c := ezfyCfg.troop(id); c != nil {
			return c.Name
		}
		return "兵种" + strconv.Itoa(id)
	}

	// ★ 守方**还活着**的兵种(剩余>0)：目标已打光(剩余0)的兵种不算，页面自动改显「最近目标」。
	aliveDef := map[int]bool{}
	for _, du := range snap.Defenders {
		if du.Count > 0 {
			aliveDef[du.TroopId] = true
		}
	}

	units := func(list []ezfyBattleUnitSnap, isAtk bool) []gin.H {
		out := []gin.H{}
		for _, u := range list {
			cmd := ""
			tgt := 0
			if isAtk {
				cmd = ezfyAtkCmdOf(cmds, u.TroopId)
				tgt = tgtOf(u.TroopId)
				// ★ 2026-09-23 修复「目标剩余0 不自动切换」：
				//   引擎里优先目标没了会自动打最近（ezfyPickTarget ②），但页面把存的目标原样展示，
				//   玩家就会看到目标已 0 却还指着一个残兵。这里把已经打光(剩余0)的目标
				//   改成显示「最近目标」，和引擎的实战行为一致。
				if tgt != 0 && !aliveDef[tgt] {
					tgt = 0
				}
			}
			// ★ 攻方兵种名展示玩家**自己阵营**的叫法（2026-09-23 用户要求），
			//   同盟国/轴心国各自的兵种名统一在这里按攻方阵营解析。
			//   守方(AI/野地)没有阵营概念，继续用快照里的通用名。
			name := u.Name
			if isAtk && atkCamp != 0 {
				if cn := ezfyCfg.troopName(u.TroopId, atkCamp); cn != "" {
					name = cn
				}
			}
			out = append(out, gin.H{
				"troop_id": u.TroopId, "name": name,
				"count": u.Count, "initial": u.InitialCount, "pos": u.Pos,
				"cmd": cmd, "cmd_name": ezfyBattleCmdName(cmd),
				"target_troop": tgt, "target_name": troopName(tgt),
			})
		}
		return out
	}

	// 目标下拉框的可选项 = 最近目标 + 守方兵种（去重）；
	// 再把攻方当前已配的目标补进去 —— 守方没有该兵种时（已全灭/司令部配的别的兵种）
	// 下拉框也要能显示当前值，否则会显示成空白。
	optSeen := map[int]bool{0: true}
	opts := []gin.H{{"id": 0, "name": "最近目标"}}
	for _, u := range snap.Defenders {
		if optSeen[u.TroopId] {
			continue
		}
		optSeen[u.TroopId] = true
		opts = append(opts, gin.H{"id": u.TroopId, "name": u.Name})
	}
	for _, u := range snap.Attackers {
		t := tgtOf(u.TroopId)
		if t != 0 && !optSeen[t] {
			optSeen[t] = true
			opts = append(opts, gin.H{"id": t, "name": troopName(t)})
		}
	}
	atkTargets := snap.AtkTargets
	if atkTargets == nil {
		atkTargets = map[int]int{}
	}

	// 日志只下发尾部若干条（含每回合标题行，够玩家看战况）
	logs := snap.Actions
	if len(logs) > ezfyBattleLogTail {
		logs = logs[len(logs)-ezfyBattleLogTail:]
	}
	head := snap.Head
	if len(head) > 6 {
		head = head[:6]
	}

	atkTotal, defTotal := int64(0), int64(0)
	for _, u := range snap.Attackers {
		atkTotal += u.Count
	}
	for _, u := range snap.Defenders {
		defTotal += u.Count
	}

	return gin.H{
		"order_id": b.OrderId, "target_name": b.TargetName,
		"target_x": b.TargetX, "target_y": b.TargetY, "target_type": b.TargetType,
		"round": maxInt(b.Round, 1), "max_round": ezfyBattleMaxRounds,
		"status": b.Status, "win": b.Win,
		"atk_cmd":  b.AtkCmd, // 原始 JSON（前端不用，调试用）
		"atk_cmds": cmds,     // troopId -> 指令，前端每行按钮高亮用
		// troopId -> 优先攻击目标（0=最近），前端每行下拉框的当前值
		"atk_targets": atkTargets,
		// 目标下拉框可选：最近目标 + 守方兵种（含当前值兜底）
		"target_options": opts,
		"phase":          phase, "round_left_ms": left,
		"round_ms": ezfyBattleRoundMs, "cmd_window_ms": ezfyBattleCmdMs,
		"attackers": units(snap.Attackers, true), "defenders": units(snap.Defenders, false),
		"atk_total": atkTotal, "def_total": defTotal,
		"head": head, "actions": logs, "done": snap.Done,
	}
}

// BattleState GET /games/ezfy/battle?order_id=N —— 战场状态（含懒推进）
func (h *EzfyHandler) BattleState(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	orderId, _ := strconv.ParseInt(c.Query("order_id"), 10, 64)
	if orderId <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var order model.EzfyOrder
	if err := h.DB.First(&order, orderId).Error; err != nil || order.UserID != uid {
		resp.NotFound(c, "出征部队不存在")
		return
	}
	b := h.ezfyBattleByOrder(orderId)
	if b == nil {
		resp.NotFound(c, "该部队没有战斗记录")
		return
	}
	now := time.Now().UnixMilli()
	snap, done := h.ezfyBattleTick(b, now)
	if done && b.Status == 2 {
		// 战斗刚结束 → 结果回写订单（下一次 processOrders 就会出战报）
		h.ezfyBattleFinishToOrder(b, now)
	}
	resp.OK(c, h.ezfyBattleView(b, snap, now, h.ensureProfile(uid).Camp))
}

// BattleCmd POST /games/ezfy/battle/cmd  {order_id, troop_id, cmd: advance|hold|retreat}
//
// ★ 用户要求：指挥是**逐兵种**的（「自己带的兵种都能指挥，就是单独指挥」）。
// troop_id 省略或传 0 = 给全部参战兵种下同一条指令（快捷）。
func (h *EzfyHandler) BattleCmd(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	var req struct {
		OrderId int64  `json:"order_id"`
		TroopId int    `json:"troop_id"`
		Cmd     string `json:"cmd"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	switch req.Cmd {
	case ezfyCmdAdvance, ezfyCmdHold, ezfyCmdRetreat:
	default:
		resp.ParamError(c, "指令只能是 前进/停止/后退")
		return
	}
	var order model.EzfyOrder
	if err := h.DB.First(&order, req.OrderId).Error; err != nil || order.UserID != uid {
		resp.NotFound(c, "出征部队不存在")
		return
	}
	b := h.ezfyBattleByOrder(req.OrderId)
	if b == nil {
		resp.NotFound(c, "该部队没有战斗记录")
		return
	}
	now := time.Now().UnixMilli()
	// ★ 先把「已经到点」的回合结算掉：这样玩家在锁定后才下的指令
	//   只会影响**下一回合**，而不是篡改已经打完的回合。
	snap, done := h.ezfyBattleTick(b, now)
	if done && b.Status == 2 {
		h.ezfyBattleFinishToOrder(b, now)
		resp.OK(c, gin.H{"done": true, "msg": "战斗已结束", "state": h.ezfyBattleView(b, snap, now, h.ensureProfile(uid).Camp)})
		return
	}
	// 逐兵种指令表
	cmds := ezfyAtkCmdsParse(b.AtkCmd)
	st := ezfyBattleStateFromSnapshot(snap)
	// 旧格式的「全军统一」指令先摊到各兵种，再删掉 0 号键（一次性平滑迁移）
	if v, ok := cmds[0]; ok {
		for _, u := range st.Attackers {
			if _, has := cmds[u.cfg.ID]; !has {
				cmds[u.cfg.ID] = v
			}
		}
		delete(cmds, 0)
	}
	if req.TroopId > 0 {
		// 指定兵种：必须真的在这支部队里
		found := false
		for _, u := range st.Attackers {
			if u.cfg.ID == req.TroopId {
				found = true
				break
			}
		}
		if !found {
			resp.ParamError(c, "该兵种不在这支部队里")
			return
		}
		cmds[req.TroopId] = req.Cmd
	} else {
		for _, u := range st.Attackers {
			cmds[u.cfg.ID] = req.Cmd
		}
	}
	b.AtkCmd = ezfyAtkCmdsEncode(cmds)
	h.DB.Model(&model.EzfyBattle{}).Where("id = ?", b.ID).Update("atk_cmd", b.AtkCmd)
	resp.OK(c, gin.H{"done": false, "state": h.ezfyBattleView(b, snap, now, h.ensureProfile(uid).Camp)})
}

// BattleTarget POST /games/ezfy/battle/target  {order_id, troop_id, target_troop}
//
// ★ 2026-09-23 用户要求：「指挥战场的时候兵种目标带过来，指挥的时候玩家也能配置，
// 默认是司令部配置的，敌对没有目标则默认攻击距离最近的」。
//
// 语义：
//   - 指挥室每行兵种的**默认目标** = 开战时从司令部「兵种战斗配置」抄进快照的 AtkTargets；
//   - 玩家在指挥室可以逐兵种改（target_troop = 0 表示「最近目标」）；
//   - 改的是**本场战斗**，不回写司令部配置（司令部的改动只影响之后新开的战场）。
//
// 存法：直接改战场快照 ezfy_battle.state 里的 AtkTargets —— 与 AtkCmd 不同，
// 这里不额外加列：快照本来就带 AtkTargets（ezfyBattleSnapshot.AtkTargets），
// 每回合重建时 `ezfyBattleStateFromSnapshot` 会原样读回来，所以改快照即持久化，
// 且「tick 没推进回合时不写库」也不会丢（改完当场就写了一次）。
func (h *EzfyHandler) BattleTarget(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	var req struct {
		OrderId     int64 `json:"order_id"`
		TroopId     int   `json:"troop_id"`
		TargetTroop int   `json:"target_troop"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	var order model.EzfyOrder
	if err := h.DB.First(&order, req.OrderId).Error; err != nil || order.UserID != uid {
		resp.NotFound(c, "出征部队不存在")
		return
	}
	b := h.ezfyBattleByOrder(req.OrderId)
	if b == nil {
		resp.NotFound(c, "该部队没有战斗记录")
		return
	}
	now := time.Now().UnixMilli()
	// 先把「已经到点」的回合按**旧目标**结算掉，再改目标 ——
	// 否则玩家能在锁定后才改目标去篡改已经打完的回合。
	snap, done := h.ezfyBattleTick(b, now)
	if done && b.Status == 2 {
		h.ezfyBattleFinishToOrder(b, now)
		resp.OK(c, gin.H{"done": true, "msg": "战斗已结束", "state": h.ezfyBattleView(b, snap, now, h.ensureProfile(uid).Camp)})
		return
	}
	// 只能指挥**自己带了的**兵种（与逐兵种指令同一口径）
	found := false
	for _, u := range snap.Attackers {
		if u.TroopId == req.TroopId {
			found = true
			break
		}
	}
	if !found {
		resp.ParamError(c, "该兵种不在这支部队里")
		return
	}
	// 0 = 最近目标；其余必须是合法兵种
	if req.TargetTroop != 0 && ezfyCfg.troop(req.TargetTroop) == nil {
		resp.ParamError(c, "目标兵种不存在")
		return
	}
	if snap.AtkTargets == nil {
		snap.AtkTargets = map[int]int{}
	}
	snap.AtkTargets[req.TroopId] = req.TargetTroop
	b.State = ezfyBattleSnapshotEncode(snap)
	h.DB.Model(&model.EzfyBattle{}).Where("id = ?", b.ID).Update("state", b.State)
	resp.OK(c, gin.H{"done": false, "state": h.ezfyBattleView(b, snap, now, h.ensureProfile(uid).Camp)})
}

// BattleAuto POST /games/ezfy/battle/auto  {order_id} —— 自动战斗：按当前指令一口气打完
//
// ★ 这是 30 秒/回合节奏的**减压阀**：小规模战斗几回合就结束，不必真等几分钟。
func (h *EzfyHandler) BattleAuto(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	var req struct {
		OrderId int64 `json:"order_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	var order model.EzfyOrder
	if err := h.DB.First(&order, req.OrderId).Error; err != nil || order.UserID != uid {
		resp.NotFound(c, "出征部队不存在")
		return
	}
	b := h.ezfyBattleByOrder(req.OrderId)
	if b == nil {
		resp.NotFound(c, "该部队没有战斗记录")
		return
	}
	now := time.Now().UnixMilli()
	// 先把到点的回合补算掉，再一路跑到结束
	h.ezfyBattleTick(b, now)
	snap, ok := ezfyBattleSnapshotDecode(b.State)
	if !ok {
		h.fail(c, "战场数据损坏")
		return
	}
	st := ezfyBattleStateFromSnapshot(snap)
	// ★ 用户要求：「玩家点击自动战斗后 默认自己的军队全部前进」
	//   —— 忽略当前指令，剩下的回合一律全军前进（否则玩家按了暂停再点自动，会一直站着挨打）。
	advance := map[int]string{}
	for _, u := range st.Attackers {
		advance[u.cfg.ID] = ezfyCmdAdvance
	}
	for !st.Done {
		st.Step(advance, "")
	}
	b.Status = 2
	if st.AttackerWin {
		b.Win = 1
	} else {
		b.Win = 2
	}
	b.Round = st.Round
	b.State = ezfyBattleSnapshotEncode(st.Snapshot())
	h.DB.Model(&model.EzfyBattle{}).Where("id = ?", b.ID).Updates(map[string]interface{}{
		"round": b.Round, "state": b.State, "status": b.Status, "win": b.Win,
	})
	h.ezfyBattleFinishToOrder(b, now)
	out, _ := ezfyBattleSnapshotDecode(b.State)
	resp.OK(c, gin.H{"done": true, "msg": "战斗已结束", "state": h.ezfyBattleView(b, out, now, h.ensureProfile(uid).Camp)})
}
