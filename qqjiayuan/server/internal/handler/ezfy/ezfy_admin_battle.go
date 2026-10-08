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
// ⚠️ 并发说明：EzfyAdmin 上没有 EzfyHandler 的方法，这里临时构造 `&EzfyHandler{DB: h.DB}` 调用。
//
//	它与玩家端请求是两个实例（processing 重入标记不共享），理论上可能与玩家端同时推进同一场战斗。
//	缓解：① 每次操作前**重新读一次战场行**拿最新 round_start；
//	② ezfyBattleTick 是「按 round_start 补算」，重复调用不会把回合算多（只会在同一回合上多写一次快照）；
//	③ 管理端操作是低频手动行为。真要严格互斥需要 DB 乐观锁，当前不做（收益 < 复杂度）。

// AdminEzfyBattles GET /admin/ezfy-battles —— 战场列表
//
// 查询参数：page / size / word（战场ID·订单ID·用户ID·游戏ID·昵称·目标名）/
//
//	status（-1 全部 / 1 进行中 / 2 已结束）/ target_type（0 全部 / 1 野地 / 2 寇城 / 3 玩家城）
func (h *EzfyAdmin) AdminEzfyBattles(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	word := strings.TrimSpace(c.Query("word"))
	status := strings.TrimSpace(c.DefaultQuery("status", "-1"))
	ttStr := strings.TrimSpace(c.DefaultQuery("target_type", "0"))

	q := h.DB.Model(&model.EzfyBattle{})
	if status == "1" || status == "2" {
		q = q.Where("status = ?", status)
	}
	if tt, err := strconv.Atoi(ttStr); err == nil && tt > 0 {
		q = q.Where("target_type = ?", tt)
	}
	if word != "" {
		if n, err := strconv.Atoi(word); err == nil {
			// 数字：战场ID / 订单ID / 攻守双方 user_id / 游戏ID(game_uid)
			q = q.Where(`id = ? OR order_id = ? OR user_id = ? OR def_user_id = ?
				OR user_id IN (SELECT user_id FROM ezfy_profile WHERE game_uid = ?)
				OR def_user_id IN (SELECT user_id FROM ezfy_profile WHERE game_uid = ?)`,
				n, n, n, n, n, n)
		} else {
			var ids []uint
			h.DB.Model(&model.EzfyProfile{}).Select("user_id").
				Where("nickname LIKE ?", "%"+word+"%").Scan(&ids)
			var uids []uint
			h.DB.Model(&model.User{}).Select("id").
				Where("username LIKE ?", "%"+word+"%").Scan(&uids)
			ids = append(ids, uids...)
			if len(ids) > 0 {
				q = q.Where("(user_id IN ? OR def_user_id IN ? OR target_name LIKE ?)",
					ids, ids, "%"+word+"%")
			} else {
				q = q.Where("target_name LIKE ?", "%"+word+"%")
			}
		}
	}

	var rows []model.EzfyBattle
	var total int64
	q.Count(&total)
	q.Order("id DESC").Offset(offset).Limit(size).Find(&rows)

	type rowOut struct {
		model.EzfyBattle
		AtkName        string `json:"atk_name"`
		AtkHome        string `json:"atk_home"`
		DefName        string `json:"def_name"`
		DefHome        string `json:"def_home"`
		OrderStatus    int    `json:"order_status"`
		TargetTypeName string `json:"target_type_name"`
		WinName        string `json:"win_name"`
	}
	out := []rowOut{}
	for _, b := range rows {
		an, ah := h.ezfyAdminName(b.UserID)
		dn, dh := h.ezfyAdminName(b.DefUserID)
		os := -1
		var o model.EzfyOrder
		if err := h.DB.Select("id, status").First(&o, b.OrderId).Error; err == nil {
			os = o.Status
		}
		out = append(out, rowOut{EzfyBattle: b, AtkName: an, AtkHome: ah,
			DefName: dn, DefHome: dh, OrderStatus: os,
			TargetTypeName: ezfyBattleTargetTypeName(b.TargetType),
			WinName:        ezfyBattleWinName(b.Status, b.Win)})
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
	eh := &EzfyHandler{DB: h.DB}
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
	eh := &EzfyHandler{DB: h.DB}
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
	eh := &EzfyHandler{DB: h.DB}
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
