package ezfy

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// ============ 管理端 · 战斗队列（战场）============
//
// ★ 2026-10-08 新增（用户要求「二战管理端加个战斗队列管理，看具体战斗，能手动/自动都这种」）：
//
//	列表 + 详情 + 手动推进一回合 + 一键自动打完 + 强制结算/清理卡死战场。
//
// 战斗推进**完全复用玩家端那套**（ezfyBattleTick / ezfyBattleFinishToOrder / processArrive），
// 管理端只负责「触发」与「展示」，不另写第二套战斗逻辑（避免两边口径漂移）。
//
// ⚠️ 并发说明：EzfyAdmin 上没有 EzfyHandler 的方法，这里临时构造 `&EzfyHandler{DB: h.DB, HomeDB: h.HomeDB}` 调用。
//
//	它与玩家端请求是两个实例（processing 重入标记不共享），理论上可能与玩家端同时推进同一场战斗。
//	缓解：① 每次操作前**重新读一次战场行**拿最新 round_start；
//	② ezfyBattleTick 是「按 round_start 补算」，重复调用不会把回合算多（只会在同一回合上多写一次快照）；
//	③ 管理端操作是低频手动行为。真要严格互斥需要 DB 乐观锁，当前不做（收益 < 复杂度）。

// ezfyAdminActiveOrderStatus 管理端「战斗队列」纳入的订单状态（**只在进行中**）：
// 0 出征 / 1 驻守 / 2 返回 / 5 战斗中 / 6 等待。
//
// ★ 2026-10-09 用户要求：已结束的不要（战报模块能查），出征队列也要并进来。
var ezfyAdminActiveOrderStatus = []int{0, 1, 2, ezfyOrderStatusBattle, ezfyOrderStatusWaiting}

// ezfyAdminOrderStatusName 管理端口径的订单状态名（与玩家端「出征 / 返回 / 等待」一致）
func ezfyAdminOrderStatusName(s int) string {
	switch s {
	case 0:
		return "出征"
	case 1:
		return "驻守"
	case 2:
		return "返回"
	case ezfyOrderStatusBattle:
		return "战斗中"
	case ezfyOrderStatusWaiting:
		return "等待"
	case ezfyOrderStatusProcessing:
		return "结算中"
	case 3:
		return "已完成"
	case 4:
		return "已终止"
	}
	return "未知"
}

// ezfyAdminTroopsText 订单兵力 → 紧凑文案（如「航母×204900, 轰炸机×100」）。
func ezfyAdminTroopsText(troopsJSON string) string {
	groups := parseGroups(troopsJSON)
	if len(groups) == 0 {
		return ""
	}
	parts := []string{}
	for _, g := range groups {
		if g.Count <= 0 {
			continue
		}
		name := fmt.Sprintf("兵种%d", g.TroopId)
		if cfg := ezfyCfg.troop(g.TroopId); cfg != nil && cfg.Name != "" {
			name = cfg.Name
		}
		parts = append(parts, fmt.Sprintf("%s×%d", name, g.Count))
	}
	return strings.Join(parts, ", ")
}

// AdminEzfyBattles GET /admin/ezfy-battles —— 战斗队列 = 所有**进行中**的军队行动
//
// ★★ 2026-10-09 用户要求（原实现只列战场、含已结束、且很卡）：
//
//	① **把出征队列也并进来** → 主表从 `ezfy_battle` 换成 `ezfy_order`
//	   （每个战场都能对上它的订单），列表里既有「战斗中(5)」也有「出征(0)/驻守(1)/返回(2)/等待(6)」；
//	② **不要已结束的**（status=3/4 的订单、status=2 的战场）→ 战报模块能查历史；
//	③ **修 N+1**：原来每行 3 条 SQL（攻方昵称 + 守方昵称 + 订单状态），
//	   50 行就是 150 条跨 WAN 查询 → 现在**批量**取昵称 / 战场 / 城池（各 1 条）。
//
// 查询参数：page / size / word（订单ID·用户ID·游戏ID·昵称）/ kind（订单状态，空=全部）/
//
//	target_type（0 全部 / 1 野地 / 2 寇城 / 3 玩家城）
func (h *EzfyAdmin) AdminEzfyBattles(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	word := strings.TrimSpace(c.Query("word"))
	ttStr := strings.TrimSpace(c.DefaultQuery("target_type", "0"))
	kindStr := strings.TrimSpace(c.DefaultQuery("kind", ""))

	q := h.DB.Model(&model.EzfyOrder{}).Where("status IN ?", ezfyAdminActiveOrderStatus)
	if k, err := strconv.Atoi(kindStr); err == nil {
		for _, s := range ezfyAdminActiveOrderStatus {
			if s == k {
				q = q.Where("status = ?", k)
				break
			}
		}
	}
	if tt, err := strconv.Atoi(ttStr); err == nil && tt > 0 {
		q = q.Where("target_type = ?", tt)
	}
	if word != "" {
		if n, err := strconv.Atoi(word); err == nil {
			// 数字：订单ID / 发起人 user_id / 目标ID / 家园号(game_uid)
			q = q.Where(`id = ? OR user_id = ? OR target_id = ?
				OR user_id IN (SELECT user_id FROM ezfy_profile WHERE game_uid = ?)`, n, n, n, n)
		} else {
			var ids []uint
			h.DB.Model(&model.EzfyProfile{}).Select("user_id").
				Where("nickname LIKE ?", "%"+word+"%").Scan(&ids)
			var uids []uint
			h.home().Model(&model.User{}).Select("id").
				Where("username LIKE ?", "%"+word+"%").Scan(&uids)
			ids = append(ids, uids...)
			if len(ids) > 0 {
				q = q.Where("user_id IN ?", ids)
			} else {
				q = q.Where("1 = 0") // 昵称/账号都没命中 → 不返回任何行
			}
		}
	}

	var total int64
	q.Count(&total)
	var orders []model.EzfyOrder
	q.Order("id DESC").Offset(offset).Limit(size).Find(&orders)

	// —— 批量取关联数据（各 1 条 SQL，替代原来的 N+1）——
	var oids, cityIds []int64
	uids := make([]uint, 0, len(orders)*2)
	for i := range orders {
		o := &orders[i]
		if o.Status == ezfyOrderStatusBattle {
			oids = append(oids, int64(o.ID))
		}
		if o.TargetType == 3 && o.TargetId > 0 {
			cityIds = append(cityIds, o.TargetId)
		}
		uids = append(uids, o.UserID)
	}
	battleByOrder := map[int64]model.EzfyBattle{}
	if len(oids) > 0 {
		var bs []model.EzfyBattle
		h.DB.Where("order_id IN ?", oids).Find(&bs)
		for i := range bs {
			battleByOrder[bs[i].OrderId] = bs[i]
			uids = append(uids, bs[i].DefUserID)
		}
	}
	cityOwner := map[int64]uint{}
	cityName := map[int64]string{}
	if len(cityIds) > 0 {
		var cs []model.EzfyCity
		h.DB.Select("id, user_id, name").Where("id IN ?", cityIds).Find(&cs)
		for i := range cs {
			cityOwner[int64(cs[i].ID)] = cs[i].UserID
			cityName[int64(cs[i].ID)] = cs[i].Name
			uids = append(uids, cs[i].UserID)
		}
	}
	nick, num := h.ezfyAdminNamesBatch(uids)

	out := []gin.H{}
	for i := range orders {
		o := &orders[i]
		row := gin.H{
			"id": o.ID, "user_id": o.UserID,
			"atk_name": nick[o.UserID], "atk_home": num[o.UserID],
			"order_type": o.OrderType, "order_type_name": ezfyOrderTypeName(o.OrderType),
			"target_type": o.TargetType, "target_type_name": ezfyBattleTargetTypeName(o.TargetType),
			"target_x": o.TargetX, "target_y": o.TargetY,
			"officer":     o.Officer,
			"troops_text": ezfyAdminTroopsText(o.Troops),
			"status":      o.Status, "status_name": ezfyAdminOrderStatusName(o.Status),
			"start_time": o.StartTime, "arrive_time": o.ArriveTime, "return_time": o.ReturnTime,
			"created_at": o.CreatedAt,
			"battle_id":  0, "round": 0, "win": 0, "win_name": "",
		}
		if o.TargetType == 3 {
			row["target_name"] = cityName[o.TargetId]
			if du := cityOwner[o.TargetId]; du > 0 {
				row["def_user_id"] = du
				row["def_name"] = nick[du]
				row["def_home"] = num[du]
			}
		} else {
			lv := ezfyWildlandLevel(o.TargetX, o.TargetY)
			if o.TargetType == 2 {
				lv = ezfyKouLevel(o.TargetX, o.TargetY)
			}
			row["target_name"] = fmt.Sprintf("%s%d级", ezfyWildTerrainDisplayName(o.TargetX, o.TargetY, true), lv)
		}
		if b, ok := battleByOrder[int64(o.ID)]; ok {
			row["battle_id"] = b.ID
			row["round"] = b.Round
			row["win"] = b.Win
			row["win_name"] = ezfyBattleWinName(b.Status, b.Win)
			if b.TargetName != "" {
				row["target_name"] = b.TargetName
			}
		}
		out = append(out, row)
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// ezfyBattleTargetTypeName 战场目标类型 → 中文
func ezfyBattleTargetTypeName(t int) string {
	switch t {
	case 1:
		return "野地"
	case 2:
		return "寇城"
	case 3:
		return "玩家城"
	}
	return "未知"
}

// ezfyBattleWinName 战场结果 → 中文（管理端列表/详情展示，不裸露 0/1/2/3）
func ezfyBattleWinName(status, win int) string {
	if status != 2 {
		return "进行中"
	}
	switch win {
	case 1:
		return "攻方胜"
	case 2:
		return "守方胜"
	case 3:
		return "平局"
	}
	return "未判定"
}

// AdminEzfyBattleDetail GET /admin/ezfy-battles/:id —— 战场详情
//
// 下发：战场行 + 订单摘要 + 双方玩家 + 快照（双方部队/位置/准备回合/逐回合日志/加成）。
func (h *EzfyAdmin) AdminEzfyBattleDetail(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var b model.EzfyBattle
	if err := h.DB.First(&b, id).Error; err != nil {
		resp.NotFound(c, "战场不存在")
		return
	}
	snap, _ := ezfyBattleSnapshotDecode(b.State)
	var order model.EzfyOrder
	hasOrder := h.DB.First(&order, b.OrderId).Error == nil
	an, ah := h.ezfyAdminName(b.UserID)
	dn, dh := h.ezfyAdminName(b.DefUserID)

	orderOut := gin.H{}
	if hasOrder {
		orderOut = gin.H{
			"id": order.ID, "status": order.Status,
			"order_type": order.OrderType, "type_name": ezfyOrderTypeName(order.OrderType),
			"target_type": order.TargetType, "target_x": order.TargetX, "target_y": order.TargetY,
			"troops": order.Troops, "officer": order.Officer,
			"auto_battle": order.AutoBattle,
		}
	}
	resp.OK(c, gin.H{
		"battle": b, "order": orderOut, "has_order": hasOrder,
		"atk_name": an, "atk_home": ah, "def_name": dn, "def_home": dh,
		"target_type_name": ezfyBattleTargetTypeName(b.TargetType),
		"win_name":         ezfyBattleWinName(b.Status, b.Win),
		"snapshot": gin.H{
			"round": snap.Round, "done": snap.Done,
			"attacker_win": snap.AttackerWin, "draw": snap.Draw,
			"attackers": snap.Attackers, "defenders": snap.Defenders,
			"head": snap.Head, "actions": snap.Actions,
			"atk_officer_desc": snap.AtkOfficerDesc, "def_officer_desc": snap.DefOfficerDesc,
			"atk_bonus": snap.AtkBonus, "def_bonus": snap.DefBonus,
			"def_atk_bonus":   snap.DefAtkBonus,
			"atk_speed_bonus": snap.AtkSpeedBonus, "def_speed_bonus": snap.DefSpeedBonus,
			"atk_range_bonus": snap.AtkRangeBonus, "def_range_bonus": snap.DefRangeBonus,
			"atk_def_bonus": snap.AtkDefBonus,
		},
	})
}

// AdminEzfyBattleTick POST /admin/ezfy-battles/:id/tick —— 手动推进一个回合
//
// 做法：把本回合开始时间拨到「刚好过点」，再走一次玩家端那套懒结算 tick（推进恰好 1 回合）。
func (h *EzfyAdmin) AdminEzfyBattleTick(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	// 重新读一次，拿最新 round_start（减少与玩家端并发的竞争窗口）
	var b model.EzfyBattle
	if err := h.DB.First(&b, id).Error; err != nil {
		resp.NotFound(c, "战场不存在")
		return
	}
	if b.Status != 1 {
		resp.ParamError(c, "该战斗已结束，无需推进")
		return
	}
	eh := &EzfyHandler{DB: h.DB, HomeDB: h.HomeDB}
	eh.cfgs()
	now := time.Now().UnixMilli()
	b.RoundStart = now - ezfyBattleRoundMs // 拨到「刚好到点」
	_, done := eh.ezfyBattleTick(&b, now)

	msg := fmt.Sprintf("已推进到第 %d 回合", b.Round)
	if done {
		msg = fmt.Sprintf("第 %d 回合分出胜负，战斗结束（点[强制结算]回写订单）", b.Round)
	}
	resp.OK(c, gin.H{"msg": msg, "round": b.Round, "done": done,
		"win_name": ezfyBattleWinName(b.Status, b.Win)})
}

// AdminEzfyBattleAuto POST /admin/ezfy-battles/:id/auto —— 一键自动打完
//
// 反复推进直到分出胜负（最多 ezfyBattleMaxRounds 回合），结束后回写订单并结算，
// 相当于管理员替玩家按了「自动战斗」。
func (h *EzfyAdmin) AdminEzfyBattleAuto(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var b model.EzfyBattle
	if err := h.DB.First(&b, id).Error; err != nil {
		resp.NotFound(c, "战场不存在")
		return
	}
	if b.Status != 1 {
		resp.ParamError(c, "该战斗已结束")
		return
	}
	eh := &EzfyHandler{DB: h.DB, HomeDB: h.HomeDB}
	eh.cfgs()
	now := time.Now().UnixMilli()
	rounds := 0
	for i := 0; i < ezfyBattleMaxRounds+2; i++ {
		b.RoundStart = now - ezfyBattleRoundMs
		_, done := eh.ezfyBattleTick(&b, now)
		rounds++
		if done {
			break
		}
	}
	settled := false
	if b.Status == 2 {
		settled = h.ezfyAdminSettleBattle(eh, &b, now)
	}
	resp.OK(c, gin.H{
		"msg": fmt.Sprintf("已自动打完：共推进 %d 回合，结果「%s」%s",
			rounds, ezfyBattleWinName(b.Status, b.Win), map[bool]string{true: "，订单已结算", false: ""}[settled]),
		"round": b.Round, "done": b.Status == 2,
		"win_name": ezfyBattleWinName(b.Status, b.Win), "settled": settled,
	})
}

// AdminEzfyBattleForce POST /admin/ezfy-battles/:id/force —— 强制结算 / 清理卡死战场
//
// 三种情况分别处理：
//
//	① 战场已结束(status=2) 但订单还卡在「战斗中(5)」→ 回写结果并结算（发战报/掠夺/返航）；
//	② 战场还在进行(status=1) 但订单已不在战斗中（僵尸行：订单被删/已结算）→ 直接把战场标记结束；
//	③ 订单根本不存在 → 直接把战场标记结束。
func (h *EzfyAdmin) AdminEzfyBattleForce(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var b model.EzfyBattle
	if err := h.DB.First(&b, id).Error; err != nil {
		resp.NotFound(c, "战场不存在")
		return
	}
	eh := &EzfyHandler{DB: h.DB, HomeDB: h.HomeDB}
	eh.cfgs()
	now := time.Now().UnixMilli()

	var order model.EzfyOrder
	hasOrder := h.DB.First(&order, b.OrderId).Error == nil

	if b.Status == 2 {
		if !hasOrder {
			resp.OK(c, gin.H{"msg": "战场已结束，且订单不存在（无需结算）", "cleaned": false})
			return
		}
		if order.Status != ezfyOrderStatusBattle {
			resp.OK(c, gin.H{"msg": "战场已结束，订单也不在「战斗中」（无需结算）", "cleaned": false})
			return
		}
		settled := h.ezfyAdminSettleBattle(eh, &b, now)
		resp.OK(c, gin.H{"msg": map[bool]string{true: "已强制结算：结果已回写订单并完成结算", false: "结算未生效（订单状态已变化）"}[settled],
			"settled": settled})
		return
	}

	// 战场还在进行：订单不在战斗中 / 不存在 → 僵尸行，直接结束
	if !hasOrder || order.Status != ezfyOrderStatusBattle {
		h.DB.Model(&model.EzfyBattle{}).Where("id = ?", b.ID).
			Updates(map[string]interface{}{"status": 2, "win": 0})
		resp.OK(c, gin.H{"msg": "该战场是僵尸数据（订单已不在战斗中），已标记结束", "cleaned": true})
		return
	}
	resp.ParamError(c, "该战场仍在进行中，订单也还在战斗中 —— 请用[推进一回合]或[自动打完]")
}

// ezfyAdminSettleBattle 战场已结束 → 把结果回写订单并触发结算。
//
// 复用玩家端两步：ezfyBattleFinishToOrder（订单 5 → 0 + 写 battle_result）
// → processArrive（发战报/掠夺/返航/经验等常规结算）。
// 返回是否真的结算成功（订单状态已变化 → false）。
func (h *EzfyAdmin) ezfyAdminSettleBattle(eh *EzfyHandler, b *model.EzfyBattle, now int64) bool {
	var order model.EzfyOrder
	if err := h.DB.First(&order, b.OrderId).Error; err != nil {
		return false
	}
	if order.Status != ezfyOrderStatusBattle {
		return false
	}
	eh.ezfyBattleFinishToOrder(b, now)
	// 重新读一次：finishToOrder 只在 status=5 时才回写
	var after model.EzfyOrder
	if err := h.DB.First(&after, b.OrderId).Error; err != nil {
		return false
	}
	if after.Status == ezfyOrderStatusBattle {
		return false // 没被改（说明 CAS 没命中）
	}
	eh.processArrive(after.UserID, &after, now)
	return true
}
