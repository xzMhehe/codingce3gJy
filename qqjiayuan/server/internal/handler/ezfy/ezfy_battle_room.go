package ezfy

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

// ezfyUserOnline 玩家是否「在线」：auth/jwt 中间件每次请求都会刷新 users.last_active_at，
// 游戏内任何操作（切页/点按钮/拉数据）都算活跃。近 ezfyOnlineWindowSec 秒内有活跃 →
// 视为在线（战斗到点照常进指挥室）；超过阈值 → 视为离线。
//
// ★ 2026-10-07 用户需求：「在线就可以指挥，只有离线的才会自动结算（打野地/非玩家）」。
// 战斗到达结算时用它分流：在线 → 开战场等玩家指挥；离线 → 抵达即自动打完，避免部队
// 占着目标在「等待(6)」排队、把同格其他人全堵住（线上活动野地一直排队就是这个场景）。
const ezfyOnlineWindowSec = 180

func (h *EzfyHandler) ezfyUserOnline(uid uint) bool {
	if uid == 0 {
		return false
	}
	var u model.User
	if err := h.home().Select("last_active_at").First(&u, uid).Error; err != nil || u.LastActiveAt == nil {
		return false
	}
	return time.Since(*u.LastActiveAt) <= ezfyOnlineWindowSec*time.Second
}

//
// ★ 2026-09-22 「实现指挥功能」，入口在 军情 → 军队动态 → [指挥]。
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
	// ezfyOrderStatusProcessing 结算中（processArrive 的 CAS 占位：同一订单只允许一个入口结算，
	// 防攻方 processOrders 与守方 processIncoming 双入口重复结算 → 用户反馈「征服报告出现两封」）
	ezfyOrderStatusProcessing = 98
)

// ============ 攻方逐兵种指令表 ============
//
// ★ （2026-09-22）：「指挥不是指挥全部，自己带的兵种都能指挥，就是单独指挥」。
// 存法：JSON `{"1":"advance","3":"hold"}`（troopId → 指令）。
// 没给的兵种 → 回落司令部「兵种战斗配置」；键 0 = 旧格式遗留的「全军统一指令」。

// ezfyBattleCmdName 指令中文名（空串 = 沿用司令部配置）
func ezfyBattleCmdName(cmd string) string {
	switch cmd {
	case ezfyCmdAdvance:
		return "前进"
	case ezfyCmdHold:
		return "待命"
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

// ezfyBattleDefenderUid 取战场守方玩家 uid（仅 target_type=3 攻击玩家城时有值，0 = AI/野地/寇城）。
// 优先读 DefUserID（新开战场已落库），旧战场没写时回落按 target_id 查城市归属。
func (h *EzfyHandler) ezfyBattleDefenderUid(b *model.EzfyBattle) uint {
	if b == nil || b.TargetType != 3 {
		return 0
	}
	if b.DefUserID > 0 {
		return b.DefUserID
	}
	var city model.EzfyCity
	if err := h.DB.First(&city, b.TargetId).Error; err == nil {
		return city.UserID
	}
	return 0
}

// ezfyBattleSide 返回 viewer 相对战场是 "atk"(攻方) / "def"(守方) / ""(无关人员)。
func (h *EzfyHandler) ezfyBattleSide(b *model.EzfyBattle, uid uint) string {
	if b == nil {
		return ""
	}
	if b.UserID == uid {
		return "atk"
	}
	if h.ezfyBattleDefenderUid(b) == uid {
		return "def"
	}
	return ""
}

// ezfyBattleStart 为订单开一场战场（幂等：已有进行中的直接返回）
func (h *EzfyHandler) ezfyBattleStart(uid uint, order *model.EzfyOrder,
	st *ezfyBattleState, targetName string, now int64) *model.EzfyBattle {

	if b := h.ezfyBattleByOrder(int64(order.ID)); b != nil && b.Status == 1 {
		return b
	}
	defUID := uint(0)
	if order.TargetType == 3 {
		var tc model.EzfyCity
		if err := h.DB.First(&tc, order.TargetId).Error; err == nil {
			defUID = tc.UserID
		}
	}
	b := &model.EzfyBattle{
		OrderId: int64(order.ID), UserID: uid, CityId: order.CityId,
		TargetType: order.TargetType, TargetId: order.TargetId,
		TargetX: order.TargetX, TargetY: order.TargetY, TargetName: targetName,
		// AtkCmd 留空 = 各兵种沿用司令部「兵种战斗配置」（玩家在司令部设过「停止」的兵种不该被强制前进）
		Status: 1, Round: 0, Win: 0, AtkCmd: "", DefCmd: "",
		DefUserID:  defUID,
		State:      ezfyBattleSnapshotEncode(st.Snapshot()),
		RoundStart: now,
	}
	if err := h.DB.Create(b).Error; err != nil {
		return nil
	}
	return b
}

// ezfyBattleAutoFinish 非玩家目标（活动野地/活动寇城/特殊城市/普通野地/AI 寇城）**抵达即自动结算**：
// 直接按默认指令把战场跑完（不建「等待指挥」的战场），返回结果 br。
//
// ★ 2026-10-07 用户反馈「线上活动野地一直等在队列」：原来这类战斗先 ezfyBattleStart 开战场
//
//	等玩家进指挥室点交战，玩家不开/不在线 → 战场冻结在“战斗中”、order 占着目标
//	(ezfyOrderTargetBusy) → 同格后续玩家部队全堵在「等待(6)」。
//	守方没有真人、没必要等对局，改成到达即自动打完；玩家城（真人守方）仍走 ezfyBattleStart。
//
// 攻打历史：结束后在 ezfy_battle 留一条**终态(status=2)**记录（管理端「攻打历史」要查这里），
// 幂等 —— 同订单已有战场行不重复建（避免 processOrders 每个 tick 重复结算时刷多行）。
func (h *EzfyHandler) ezfyBattleAutoFinish(uid uint, order *model.EzfyOrder,
	st *ezfyBattleState, targetName string, now int64) ezfyBattleResult {
	for !st.Done {
		st.Step(nil, nil)
	}
	br := st.Result()
	if h.ezfyBattleByOrder(int64(order.ID)) == nil {
		defUID := uint(0)
		if order.TargetType == 3 {
			var tc model.EzfyCity
			if err := h.DB.First(&tc, order.TargetId).Error; err == nil {
				defUID = tc.UserID
			}
		}
		win := 2
		if br.AttackerWin {
			win = 1
		} else if br.Draw {
			win = 3
		}
		b := &model.EzfyBattle{
			OrderId: int64(order.ID), UserID: uid, CityId: order.CityId,
			TargetType: order.TargetType, TargetId: order.TargetId,
			TargetX: order.TargetX, TargetY: order.TargetY, TargetName: targetName,
			Status: 2, Round: br.Rounds, Win: win,
			DefUserID: defUID, State: ezfyBattleSnapshotEncode(st.Snapshot()),
			RoundStart: now,
		}
		h.DB.Create(b)
	}
	return br
}

// ezfyBattleTick 懒推进：把「已经到点」的回合逐回合结算掉。
//
// 返回最新快照 + 是否已结束。守方无指令时按司令部兵种配置行动，
// 攻守双方各自的逐兵种指令（玩家能指挥）分别取 b.AtkCmd / b.DefCmd。
func (h *EzfyHandler) ezfyBattleTick(b *model.EzfyBattle, now int64) (ezfyBattleSnapshot, bool) {
	snap, ok := ezfyBattleSnapshotDecode(b.State)
	if !ok {
		return ezfyBattleSnapshot{}, true
	}
	// ★ 2026-09-24 修复「军队动态里被攻击的动态打完了还一直显示」：
	//   一方无兵时战场在 ezfyBattleStart 开局快照就是 done(round=0)、行状态却还是 1，
	//   老代码这里直接 return，行永远卡在 status=1 → 守方军队动态一直显示「战斗中」。
	//   发现「快照已结束但行状态没跟上」时顺手把行修好(win 从快照反推)。
	if snap.Done {
		if b.Status != 2 {
			b.Status = 2
			b.Win = 2 // 默认守方胜
			if snap.AttackerWin {
				b.Win = 1
			} else if snap.Draw {
				b.Win = 3
			}
			h.DB.Model(&model.EzfyBattle{}).Where("id = ?", b.ID).Updates(map[string]interface{}{
				"status": b.Status, "win": b.Win,
			})
		}
		return snap, true
	}
	if b.Status != 1 {
		return snap, true
	}
	st := ezfyBattleStateFromSnapshot(snap)

	// 逐兵种指令表解析一次，整个补算过程复用
	cmds := ezfyAtkCmdsParse(b.AtkCmd)
	defCmds := ezfyAtkCmdsParse(b.DefCmd)

	steps := 0
	for !st.Done && now-b.RoundStart >= ezfyBattleRoundMs {
		st.Step(cmds, defCmds)
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
		} else if st.Draw {
			b.Win = 3 // 平局（40 回合未分胜负，守方视为守住）
		} else {
			b.Win = 2
		}
	}
	if steps > 0 || st.Done {
		b.Round = st.Round
		b.State = ezfyBattleSnapshotEncode(st.Snapshot())
		// ★ 2026-10-06 新回合开始 → 清除双方锁定（本回合锁定的配置只对本回合生效）
		if steps > 0 {
			b.AtkLock = 0
			b.DefLock = 0
		}
		h.DB.Model(&model.EzfyBattle{}).Where("id = ?", b.ID).Updates(map[string]interface{}{
			"round": b.Round, "state": b.State, "status": b.Status,
			"win": b.Win, "round_start": b.RoundStart,
			"atk_lock": b.AtkLock, "def_lock": b.DefLock,
		})
	}
	out, _ := ezfyBattleSnapshotDecode(b.State)
	return out, st.Done
}

// BgTickBattles 后台兜底推进所有「战斗中」战场 + 自动放行「等待」订单 + **离线玩家的到期活**
// （服务启动时常驻 goroutine）。
//
// ★ 2026-10-01 线上 bug：活动野地/活动寇城/特殊城市的战场 def_user_id=0，
//
//	没有「守方轮询」兜底，只靠攻方本人轮询懒结算 —— 攻方中途下线，战场就
//	永久冻结在 status=5（差几回合打不完），同目标排队的「等待(6)」订单因
//	ezfyOrderTargetBusy 判定忙碌而永远放行不了，玩家看到「一直等待」。
//	这里周期性找出「有活跃战场」+「有等待订单」的 uid，统一复用 processOrders(uid)——
//	它与玩家在线轮询走同一套逻辑：
//	  1) 战斗中战场 → ezfyBattleTick 按 30 秒/回合自动推进（默认指令前进），
//	     打到 40 回合耗尽或一方全灭即结束，ezfyBattleFinishToOrder 回写 + processArrive 结算；
//	  2) 等待订单 → 目标不再忙时自动放行重新进指挥，**即使该玩家也离线**，
//	     整条排队队列按 id 先后逐场打下去，不再互相卡住。
//	processOrders 内部自带 enterProcess 防重入 + processArrive 的 CAS 抢占，
//	与玩家在线轮询天然互斥，不会重复结算。
//
// ★★ 2026-10-09 用户反馈「没在线 军队就一直在路上、建筑升级也一样」：
//
//	懒结算只在玩家**发请求**时推进 → 离线玩家（以及别人看他的城/军队）拿到的
//	是「停在原地」的旧状态。这里补上两类「已到点」的兜底（每 30 秒一次，隔一次 tick）：
//	  · 到期订单：status=0 且 arrive_time<=now（行军抵达）/ status=2 且 return_time<=now（返航到城）/
//	    status=1 且 order_type=7 且 arrive_time<=now（驻守采集到点）；
//	  · 到期建筑升级：ezfy_city_building.end_time<=now（建造/升级/一键连锁）。
//	命中 uid 后跑 processOrders（订单）+ 逐城 refreshCity（建筑/科技/训练/资源），
//	与在线轮询同一套结算逻辑，不新增第二套实现。
//
// 性能：到期订单走 idx_status（status 只有 0/1/2 三种活状态，行数很小），
// 建筑表按 end_time 扫（城市数×建筑数，量级 万级以内）；
// 每轮最多处理 ezfyBgTickMaxUids 个玩家，避免停服后积压造成瞬时雪崩。
func (h *EzfyHandler) BgTickBattles() {
	tick := 0
	for range time.Tick(15 * time.Second) {
		tick++
		now := time.Now().UnixMilli()
		// 只处理「有活跃战场」或「有等待订单」的 uid，避免全量扫描订单表
		seen := map[uint]bool{}
		// cityOnly：由「到期建筑/到期订单」发现的 uid —— 这些还要逐城跑一次完整结算
		cityOnly := map[uint]bool{}
		var battleUids, waitUids []uint
		h.DB.Model(&model.EzfyBattle{}).
			Where("status = ?", 1).
			Distinct("user_id").Pluck("user_id", &battleUids)
		h.DB.Model(&model.EzfyOrder{}).
			Where("status = ?", ezfyOrderStatusWaiting).
			Distinct("user_id").Pluck("user_id", &waitUids)
		for _, uid := range append(battleUids, waitUids...) {
			if uid != 0 {
				seen[uid] = true
			}
		}
		// ★ 2026-10-09 离线兜底：每 30 秒（隔一次 tick）补扫「已到点」的行军/返航/采集订单 + 到期建筑
		if tick%2 == 0 {
			var dueUids []uint
			h.DB.Model(&model.EzfyOrder{}).
				Where("(status = 0 AND arrive_time > 0 AND arrive_time <= ?)"+
					" OR (status = 2 AND return_time > 0 AND return_time <= ?)"+
					" OR (status = 1 AND order_type = 7 AND arrive_time > 0 AND arrive_time <= ?)",
					now, now, now).
				Distinct("user_id").Pluck("user_id", &dueUids)
			var bUids []uint
			h.DB.Raw(`SELECT DISTINCT c.user_id FROM ezfy_city_building b
				JOIN ezfy_city c ON c.id = b.city_id
				WHERE b.end_time > 0 AND b.end_time <= ?`, now).Scan(&bUids)
			for _, uid := range append(dueUids, bUids...) {
				if uid != 0 {
					seen[uid] = true
					cityOnly[uid] = true
				}
			}
		}
		n := 0
		for uid := range seen {
			if n >= ezfyBgTickMaxUids {
				break // 积压过多 → 本轮到此，下一轮继续（避免瞬时打满数据库）
			}
			n++
			h.processOrders(uid)
			if cityOnly[uid] {
				h.settleDueCities(uid)
			}
		}
	}
}

// ezfyBgTickMaxUids 后台兜底每轮最多处理的玩家数（防止长时间停服后一次性全量结算）。
const ezfyBgTickMaxUids = 40

// settleDueCities 对该玩家名下每座城跑一次完整懒结算（建筑完工 / 科技完成 / 训练出厂 / 资源产出）。
//
// ★ 2026-10-09 专供后台兜底：玩家离线时建筑升级也要「按时完成」，别人（侦查/排行/军团）
// 看到的才是最新状态。逐城调用与在线轮询同一个 refreshCity，不新增第二套逻辑。
func (h *EzfyHandler) settleDueCities(uid uint) {
	var cities []model.EzfyCity
	h.DB.Where("user_id = ?", uid).Find(&cities)
	for i := range cities {
		c := cities[i]
		h.refreshCity(uid, &c)
	}
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
	// ★★ 2026-10-06 修复「打一下生成十个战报」：这里**只允许把「战斗中(5)」的订单打回 0**。
	//   战场打完（b.Status=2）后 BattleState 每次被拉都会走到本函数 —— 老代码无条件写
	//   status=0，会把已结算(2/3)的订单重新打开，processArrive 的 CAS(0/5/6→98)再次命中
	//   → 同一条征服/被征服报告反复落库。加 `AND status=5` 后已结算订单不再被二次结算。
	h.DB.Model(&model.EzfyOrder{}).
		Where("id = ? AND status = ?", b.OrderId, ezfyOrderStatusBattle).
		Updates(map[string]interface{}{
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

// ezfyBattleView 下发给前端的战场视图。
// viewerCamp / viewerIsAtk：观察方自己的阵营与攻守身份 —— 观察方只能指挥「自己这一方」，
// 另一方（敌方）不下发指令。
//
// ★ 2026-10-05 用户要求：兵种名**统一用基础兵种名**（不带阵营前缀），viewerCamp 已不再参与取名。
func (h *EzfyHandler) ezfyBattleView(b *model.EzfyBattle, snap ezfyBattleSnapshot, now int64, viewerCamp int, viewerIsAtk bool) gin.H {
	// 本回合剩余时间：过了就是 0（等待下一次请求推进）
	left := b.RoundStart + ezfyBattleRoundMs - now
	if left < 0 {
		left = 0
	}
	phase := "lock" // 锁定结算期
	if left > ezfyBattleRoundMs-ezfyBattleCmdMs {
		phase = "cmd" // 指令期（前 25 秒）
	}

	// ★ 2026-10-06 攻守双方玩家昵称：PvP（target_type=3 打玩家城）时双方都是真人，
	//   指挥室敌方行的「AI」标签换成玩家昵称；打野地/寇城（无玩家守方）保持 AI 兜底。
	atkName, defName := "", ""
	if b.TargetType == 3 {
		atkName = h.ensureProfile(b.UserID).Nickname
		if defUID := h.ezfyBattleDefenderUid(b); defUID > 0 {
			defName = h.ensureProfile(defUID).Nickname
		}
	}

	// ★ 逐兵种指令：攻守双方各自带自己的指令，前端每行单独显示/下达
	atkCmds := ezfyAtkCmdsParse(b.AtkCmd)
	defCmds := ezfyAtkCmdsParse(b.DefCmd)

	// 攻守双方各自「优先攻击目标」快照
	atkTgtOf := func(troopId int) int {
		if snap.AtkTargets == nil {
			return 0
		}
		return snap.AtkTargets[troopId]
	}
	defTgtOf := func(troopId int) int {
		if snap.DefTargets == nil {
			return 0
		}
		return snap.DefTargets[troopId]
	}
	// ★ 2026-10-05 用户要求：指挥模块兵种名**统一展示基础兵种名**（不带阵营前缀）。
	troopName := func(id int) string {
		if id == 0 {
			return "最近"
		}
		// camp 传 0 → 基础兵种名
		if cn := ezfyCfg.troopName(id, 0); cn != "" {
			return cn
		}
		return "兵种" + strconv.Itoa(id)
	}

	// 观察方自己的指令/目标表，以及他要瞄准的「敌方」兵种表
	myCmds := atkCmds
	myTgtOf := atkTgtOf
	enemySnaps := snap.Defenders
	if !viewerIsAtk {
		myCmds = defCmds
		myTgtOf = defTgtOf
		enemySnaps = snap.Attackers
	}
	// 敌方**还活着**的兵种(剩余>0)：目标已打光(剩余0)的兵种不算，页面自动改显「最近目标」。
	aliveEnemy := map[int]bool{}
	for _, eu := range enemySnaps {
		if eu.Count > 0 {
			aliveEnemy[eu.TroopId] = true
		}
	}

	units := func(list []ezfyBattleUnitSnap, isAtk bool) []gin.H {
		out := []gin.H{}
		for _, u := range list {
			cmd := ""
			tgt := 0
			// 只给「自己这一方」的兵种下发指令/目标，另一方（敌方）显示为 AI/近况
			if isAtk == viewerIsAtk {
				cmd = ezfyAtkCmdOf(myCmds, u.TroopId)
				tgt = myTgtOf(u.TroopId)
				// ★ 2026-09-23 修复「目标剩余0 不自动切换」：
				//   引擎里优先目标没了会自动打最近（ezfyPickTarget ②），页面把已打光(剩余0)
				//   的目标改成显示「最近目标」，和引擎的实战行为一致。
				if tgt != 0 && !aliveEnemy[tgt] {
					tgt = 0
				}
			}
			// ★ 2026-10-05 用户要求：指挥模块兵种名统一用基础兵种名（不带阵营前缀）
			name := u.Name
			if cn := ezfyCfg.troopName(u.TroopId, 0); cn != "" {
				name = cn
			}
			// ★ 2026-10-10 战场位置动画：补 icon(兵种图标) 供前端位置条展示
			icon := ""
			if tc := ezfyCfg.troop(u.TroopId); tc != nil {
				icon = tc.Icon
			}
			out = append(out, gin.H{
				"troop_id": u.TroopId, "name": name,
				"count": u.Count, "initial": u.InitialCount, "pos": u.Pos,
				"icon": icon,
				"cmd": cmd, "cmd_name": ezfyBattleCmdName(cmd),
				"target_troop": tgt, "target_name": troopName(tgt),
			})
		}
		return out
	}

	// 目标下拉框 = 最近目标 + 敌方兵种（去重）；再把己方当前已配但敌方没有的目标补进去
	// （敌方全灭/司令部配了别的兵种时，下拉框也要能显示当前值，否则会显示空白）。
	optSeen := map[int]bool{0: true}
	opts := []gin.H{{"id": 0, "name": "最近"}}
	for _, u := range enemySnaps {
		if optSeen[u.TroopId] {
			continue
		}
		optSeen[u.TroopId] = true
		// ★ 2026-10-05 用户要求：目标下拉里的兵种名统一用基础兵种名
		name := u.Name
		if cn := ezfyCfg.troopName(u.TroopId, 0); cn != "" {
			name = cn
		}
		opts = append(opts, gin.H{"id": u.TroopId, "name": name})
	}
	myList := snap.Attackers
	if !viewerIsAtk {
		myList = snap.Defenders
	}
	for _, u := range myList {
		t := myTgtOf(u.TroopId)
		if t != 0 && !optSeen[t] {
			optSeen[t] = true
			opts = append(opts, gin.H{"id": t, "name": troopName(t)})
		}
	}
	myTargets := map[int]int{}
	if viewerIsAtk {
		if snap.AtkTargets != nil {
			myTargets = snap.AtkTargets
		}
	} else if snap.DefTargets != nil {
		myTargets = snap.DefTargets
	}

	// 日志只下发尾部若干条（含每回合标题行，够玩家看战况）
	logs := snap.Actions
	if len(logs) > ezfyBattleLogTail {
		logs = logs[len(logs)-ezfyBattleLogTail:]
	}
	head := snap.Head
	// ★★ 2026-10-08 修复「指挥室看不到『战斗加成』汇总行」（用户反馈「就是汇总的没加」）：
	//   原来只下发前 6 行，而准备回合的顺序是
	//     军官×2 → 科技×2 → 装备×2 → 套装×0~2 → **战斗加成** → 场景描述
	//   —— 6 行正好卡在【守方装备】，末尾的汇总行被切掉了（战报不受影响，因为战报直接用 st.Head 全量）。
	//   head 是固定的小数组（最多 ~10 行、每行百余字节），放宽到 12 行足够覆盖，
	//   同时仍保留上限，避免将来 head 意外膨胀时下发过大。
	if len(head) > 12 {
		head = head[:12]
	}

	atkTotal, defTotal := int64(0), int64(0)
	for _, u := range snap.Attackers {
		atkTotal += u.Count
	}
	for _, u := range snap.Defenders {
		defTotal += u.Count
	}

	// PvP（攻击玩家城）→ 双方都能指挥，一键[自动战斗]对另一方不公平，禁用
	pvp := b.TargetType == 3 && h.ezfyBattleDefenderUid(b) > 0

	// ★ 2026-10-06 保存/锁定配置：我方是否已锁定、对方是否已锁定、我方能否锁定。
	//   my_locked = 观察方自己这一边是否已保存配置（保存即锁定，双方都锁或到点立即结算）。
	myLocked := b.AtkLock
	if !viewerIsAtk {
		myLocked = b.DefLock
	}
	canLock := b.Status == 1 && !snap.Done && myLocked == 0

	return gin.H{
		"order_id": b.OrderId, "target_name": b.TargetName,
		"target_x": b.TargetX, "target_y": b.TargetY, "target_type": b.TargetType,
		"round": maxInt(b.Round, 1), "max_round": ezfyBattleMaxRounds,
		"status": b.Status, "win": b.Win, "draw": snap.Draw,
		"atk_cmd":  b.AtkCmd, // 原始 JSON（前端不用，调试用）
		"atk_cmds": atkCmds,  // troopId -> 指令，前端每行按钮高亮用
		"def_cmd":  b.DefCmd, "def_cmds": defCmds,
		// 观察方是不是攻方（守方视角时前端把指挥按钮渲染到守方行上）
		"is_atk": viewerIsAtk,
		// ★ 2026-10-06 敌方玩家昵称（PvP 才有值，野地/寇城为空 → 前端显示 AI）
		"atk_name": atkName, "def_name": defName,
		// PvP 真人对抗时禁用[自动战斗]
		"pvp":      pvp,
		"can_auto": !pvp,
		// ★ 2026-10-06 锁定状态：atk_locked/def_locked = 双方是否已保存配置；
		//   my_locked = 观察方自己是否已锁；can_lock = 现在还能不能保存
		"atk_locked": b.AtkLock, "def_locked": b.DefLock,
		"my_locked": myLocked, "can_lock": canLock,
		// 观察方自己的逐兵种优先攻击目标（0=最近）
		"my_targets": myTargets,
		// 目标下拉框可选：最近目标 + 敌方兵种（含当前值兜底）
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
	b := h.ezfyBattleByOrder(orderId)
	if b == nil {
		resp.NotFound(c, "该部队没有战斗记录")
		return
	}
	side := h.ezfyBattleSide(b, uid)
	if side == "" {
		resp.NotFound(c, "出征部队不存在")
		return
	}
	now := time.Now().UnixMilli()
	snap, done := h.ezfyBattleTick(b, now)
	if done && b.Status == 2 {
		// 战斗刚结束 → 结果回写订单（下一次 processOrders 就会出战报）
		// ★ 2026-10-05 修复「指挥结束了却迟迟看不到结果」：订单属于**攻方**，
		//   若此刻是**守方**在指挥室里收的尾（攻方已离线），攻方那条订单不会有人去结算，
		//   战报/掠夺要等攻方下次上线才出。这里直接按攻方身份把订单结算掉。
		//   ⚠️ processArrive 内部有 CAS 抢占（status 0/5 → 98），与攻方入口并发也不会重复结算。
		// ★★ 2026-10-06 修复「打一下生成十个战报」：战场打完(行 status=2)后本接口每次被拉
		//   都会走进这个分支 —— 只有「仍处于战斗中(5)」或「刚被别的入口打回行进(0)」的
		//   订单才允许回写+结算；已结算(2/3)的订单直接跳过（finishToOrder 内部还有
		//   `AND status=5` 守卫双保险），否则同一会场反复拉取会把订单重置回 0 重新结算。
		var atkOrder model.EzfyOrder
		if err := h.DB.First(&atkOrder, b.OrderId).Error; err != nil {
			resp.OK(c, h.ezfyBattleView(b, snap, now, h.ensureProfile(uid).Camp, side == "atk"))
			return
		}
		switch atkOrder.Status {
		case ezfyOrderStatusBattle:
			h.ezfyBattleFinishToOrder(b, now)
			// 重新读一次拿到写好的 battle_result，结算正文才完整
			var settled model.EzfyOrder
			if err := h.DB.First(&settled, b.OrderId).Error; err == nil {
				h.processArrive(b.UserID, &settled, now)
			}
		case 0:
			// 已被别的入口 finishToOrder 打回行进中，直接走一次 CAS 结算（抢不到即跳过）
			h.processArrive(b.UserID, &atkOrder, now)
		}
	}
	resp.OK(c, h.ezfyBattleView(b, snap, now, h.ensureProfile(uid).Camp, side == "atk"))
}

// BattleCmd POST /games/ezfy/battle/cmd  {order_id, troop_id, cmd: advance|hold|retreat}
//
// ★ 指挥是**逐兵种**的（「自己带的兵种都能指挥，就是单独指挥」）。
// troop_id 省略或传 0 = 给全部参战兵种下同一条指令（快捷）。
// cmd 传空串 = 清除指令，该兵种回落司令部「兵种战斗配置」（前端下拉可回「默认」）。
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
	case ezfyCmdAdvance, ezfyCmdHold, ezfyCmdRetreat, "":
	default:
		resp.ParamError(c, "指令只能是 前进/待命/后退")
		return
	}
	b := h.ezfyBattleByOrder(req.OrderId)
	if b == nil {
		resp.NotFound(c, "该部队没有战斗记录")
		return
	}
	side := h.ezfyBattleSide(b, uid)
	if side == "" {
		resp.NotFound(c, "出征部队不存在")
		return
	}
	// ★ 2026-10-06 本回合配置已锁定（保存过配置）：必须先[取消配置]才能再改指令
	if (side == "atk" && b.AtkLock == 1) || (side != "atk" && b.DefLock == 1) {
		resp.ParamError(c, "本回合配置已锁定，请先取消配置")
		return
	}
	now := time.Now().UnixMilli()
	// ★ 2026-09-24 用户反馈「之前选了后退、改成停止后部队还在后退」：
	//   老顺序是**先结算到期回合、再写指令** —— 懒结算会把积压的回合一次性按
	//   旧指令跑完，玩家点了停止却看到部队连续后撤好几步，像指令失灵。
	//   改成**先写指令、后结算**：只要回合还没被结算，新指令立即生效。
	prevSnap, _ := ezfyBattleSnapshotDecode(b.State)
	prevSt := ezfyBattleStateFromSnapshot(prevSnap)
	myUnits := prevSt.Attackers
	if side != "atk" {
		myUnits = prevSt.Defenders
	}
	// 攻方写 AtkCmd、守方写 DefCmd —— 各自指挥自己这一边的兵种
	cmds := ezfyAtkCmdsParse(b.AtkCmd)
	field := "atk_cmd"
	if side != "atk" {
		cmds = ezfyAtkCmdsParse(b.DefCmd)
		field = "def_cmd"
	}
	// 旧格式的「全军统一」指令先摊到各兵种，再删掉 0 号键（一次性平滑迁移）
	if v, ok := cmds[0]; ok {
		for _, u := range myUnits {
			if _, has := cmds[u.cfg.ID]; !has {
				cmds[u.cfg.ID] = v
			}
		}
		delete(cmds, 0)
	}
	// cmd 为空串 = 清除指令（回落司令部「兵种战斗配置」）
	if req.Cmd == "" {
		if req.TroopId > 0 {
			delete(cmds, req.TroopId)
		} else {
			for _, u := range myUnits {
				delete(cmds, u.cfg.ID)
			}
		}
		enc := ezfyAtkCmdsEncode(cmds)
		if side == "atk" {
			b.AtkCmd = enc
		} else {
			b.DefCmd = enc
		}
		h.DB.Model(&model.EzfyBattle{}).Where("id = ?", b.ID).Update(field, enc)
		snap, done := h.ezfyBattleTick(b, now)
		if done && b.Status == 2 {
			h.ezfyBattleFinishToOrder(b, now)
			resp.OK(c, gin.H{"done": true, "msg": "战斗已结束", "state": h.ezfyBattleView(b, snap, now, h.ensureProfile(uid).Camp, side == "atk")})
			return
		}
		resp.OK(c, gin.H{"done": false, "state": h.ezfyBattleView(b, snap, now, h.ensureProfile(uid).Camp, side == "atk")})
		return
	}
	if req.TroopId > 0 {
		// 指定兵种：必须真的在这支部队里
		found := false
		for _, u := range myUnits {
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
		for _, u := range myUnits {
			cmds[u.cfg.ID] = req.Cmd
		}
	}
	enc := ezfyAtkCmdsEncode(cmds)
	if side == "atk" {
		b.AtkCmd = enc
	} else {
		b.DefCmd = enc
	}
	h.DB.Model(&model.EzfyBattle{}).Where("id = ?", b.ID).Update(field, enc)

	snap, done := h.ezfyBattleTick(b, now)
	if done && b.Status == 2 {
		h.ezfyBattleFinishToOrder(b, now)
		resp.OK(c, gin.H{"done": true, "msg": "战斗已结束", "state": h.ezfyBattleView(b, snap, now, h.ensureProfile(uid).Camp, side == "atk")})
		return
	}
	resp.OK(c, gin.H{"done": false, "state": h.ezfyBattleView(b, snap, now, h.ensureProfile(uid).Camp, side == "atk")})
}

// BattleTarget POST /games/ezfy/battle/target  {order_id, troop_id, target_troop}
//
// ★ 2026-09-23 「指挥战场的时候兵种目标带过来，指挥的时候玩家也能配置，
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
	b := h.ezfyBattleByOrder(req.OrderId)
	if b == nil {
		resp.NotFound(c, "该部队没有战斗记录")
		return
	}
	side := h.ezfyBattleSide(b, uid)
	if side == "" {
		resp.NotFound(c, "出征部队不存在")
		return
	}
	now := time.Now().UnixMilli()
	// 先把「已经到点」的回合按**旧目标**结算掉，再改目标 ——
	// 否则玩家能在锁定后才改目标去篡改已经打完的回合。
	snap, done := h.ezfyBattleTick(b, now)
	if done && b.Status == 2 {
		h.ezfyBattleFinishToOrder(b, now)
		resp.OK(c, gin.H{"done": true, "msg": "战斗已结束", "state": h.ezfyBattleView(b, snap, now, h.ensureProfile(uid).Camp, side == "atk")})
		return
	}
	// ★ 2026-10-06 本回合配置已锁定（保存过配置）：必须先[取消配置]才能再改目标
	//   —— 注意放在 tick 之后：若 tick 推进了新回合已自动清锁，则不受影响
	if (side == "atk" && b.AtkLock == 1) || (side != "atk" && b.DefLock == 1) {
		resp.ParamError(c, "本回合配置已锁定，请先取消配置")
		return
	}
	// 只能指挥**自己带了**的兵种（攻方改 AtkTargets、守方改 DefTargets）
	mySnaps := snap.Attackers
	targets := snap.AtkTargets
	if side != "atk" {
		mySnaps = snap.Defenders
		targets = snap.DefTargets
	}
	found := false
	for _, u := range mySnaps {
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
	if targets == nil {
		targets = map[int]int{}
	}
	targets[req.TroopId] = req.TargetTroop
	if side == "atk" {
		snap.AtkTargets = targets
	} else {
		snap.DefTargets = targets
	}
	b.State = ezfyBattleSnapshotEncode(snap)
	h.DB.Model(&model.EzfyBattle{}).Where("id = ?", b.ID).Update("state", b.State)
	resp.OK(c, gin.H{"done": false, "state": h.ezfyBattleView(b, snap, now, h.ensureProfile(uid).Camp, side == "atk")})
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
	b := h.ezfyBattleByOrder(req.OrderId)
	if b == nil {
		resp.NotFound(c, "该部队没有战斗记录")
		return
	}
	side := h.ezfyBattleSide(b, uid)
	if side == "" {
		resp.NotFound(c, "出征部队不存在")
		return
	}
	// ★ 2026-09-23 「两个人都在指挥的话不能点击[自动战斗]」：
	//   攻击玩家城（PvP，双方都能指挥）时，一键打完对另一方不公平，禁用。
	if b.TargetType == 3 && h.ezfyBattleDefenderUid(b) > 0 {
		resp.OK(c, gin.H{"done": false, "msg": "真人对抗无法自动战斗，请逐回合指挥"})
		return
	}
	// ★ 2026-10-06 本回合配置已锁定：自动战斗等于无视锁定单方面结算，先取消锁定再用
	if (side == "atk" && b.AtkLock == 1) || (side != "atk" && b.DefLock == 1) {
		resp.ParamError(c, "本回合配置已锁定，请先取消配置")
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
	// ★ 「玩家点击自动战斗后 默认自己的军队全部前进」
	//   —— 忽略当前指令，剩下的回合一律全军前进（否则玩家按了暂停再点自动，会一直站着挨打）。
	advance := map[int]string{}
	for _, u := range st.Attackers {
		advance[u.cfg.ID] = ezfyCmdAdvance
	}
	for !st.Done {
		st.Step(advance, nil)
	}
	b.Status = 2
	if st.AttackerWin {
		b.Win = 1
	} else if st.Draw {
		b.Win = 3 // 平局（40 回合未分胜负，守方视为守住）
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
	resp.OK(c, gin.H{"done": true, "msg": "战斗已结束", "state": h.ezfyBattleView(b, out, now, h.ensureProfile(uid).Camp, side == "atk")})
}

// BattleLock POST /games/ezfy/battle/lock  {order_id, lock: bool} —— 保存 / 取消本回合配置
//
// ★ 2026-10-06 用户要求（点6）：「玩家设置好兵种状态后，点击保存，直接进入伤害锁定；
// 在锁定时间内，可以取消重新配置；如果时间剩余不多，哪怕没操作完成，就按照当前操作直接锁定伤害」。
//
// 语义：
//   - lock=true（保存配置）：把我方锁字段置 1，本回合禁改指令/目标（只能先取消再改）。
//     若双方都已锁 → 立即结算本回合（RoundStart 前移一回合时长 → tick 推进并自动清锁），
//     不会干等到 30 秒到点。
//   - lock=false（取消配置）：仅当本回合尚未结算（RoundStart + 回合时长 > now）时置 0；
//     已到点被 tick 结算掉的回合不能取消（那等于篡改历史）。
//   - 本回合最后 5 秒（锁定期）本来就不能再下指令，保存/取消照常可用，到点由 tick 兜底结算。
func (h *EzfyHandler) BattleLock(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	var req struct {
		OrderId int64 `json:"order_id"`
		Lock    bool  `json:"lock"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	b := h.ezfyBattleByOrder(req.OrderId)
	if b == nil {
		resp.NotFound(c, "该部队没有战斗记录")
		return
	}
	side := h.ezfyBattleSide(b, uid)
	if side == "" {
		resp.NotFound(c, "出征部队不存在")
		return
	}
	now := time.Now().UnixMilli()
	// 战场已结束则无需再锁
	snap, done := h.ezfyBattleTick(b, now)
	if done && b.Status == 2 {
		h.ezfyBattleFinishToOrder(b, now)
		resp.OK(c, gin.H{"done": true, "msg": "战斗已结束", "state": h.ezfyBattleView(b, snap, now, h.ensureProfile(uid).Camp, side == "atk")})
		return
	}

	field := "atk_lock"
	if side != "atk" {
		field = "def_lock"
	}
	if req.Lock {
		if (side == "atk" && b.AtkLock == 1) || (side != "atk" && b.DefLock == 1) {
			resp.ParamError(c, "本回合配置已锁定")
			return
		}
		if side == "atk" {
			b.AtkLock = 1
		} else {
			b.DefLock = 1
		}
		h.DB.Model(&model.EzfyBattle{}).Where("id = ?", b.ID).Update(field, 1)
		// 双方都已锁 → 立即结算本回合：RoundStart 前移一回合时长，
		// 下面的 tick 就会把这一回合推进掉并清锁（保存即锁定、双锁即结算）。
		if b.AtkLock == 1 && b.DefLock == 1 {
			b.RoundStart -= ezfyBattleRoundMs
			h.DB.Model(&model.EzfyBattle{}).Where("id = ?", b.ID).Update("round_start", b.RoundStart)
		}
	} else {
		// 取消配置：本回合已到点结算（被 tick 推进过、锁已清）时无需取消；
		// 若仍未到点但锁字段是 0（本来就没锁），也无所谓，幂等处理。
		if now-b.RoundStart >= ezfyBattleRoundMs {
			resp.ParamError(c, "本回合已结算，无法取消")
			return
		}
		if side == "atk" {
			b.AtkLock = 0
		} else {
			b.DefLock = 0
		}
		h.DB.Model(&model.EzfyBattle{}).Where("id = ?", b.ID).Update(field, 0)
	}
	out, d2 := h.ezfyBattleTick(b, now)
	if d2 && b.Status == 2 {
		h.ezfyBattleFinishToOrder(b, now)
	}
	resp.OK(c, gin.H{"done": b.Status == 2, "state": h.ezfyBattleView(b, out, now, h.ensureProfile(uid).Camp, side == "atk")})
}
