package ezfy

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 二战风云 一键采集 / 一键收获
// 复刻 `二战风云/templates/report/index.html` 军队动态区的两个按钮
// （原版 /home/city_troop/begincollectall.html 与 stopcollectall.html）

// CollectAll POST /games/ezfy/wild/collect-all —— 一键采集
//
// ★ 2026-09-24 用户规则: 采集部队到达野地后驻守**空闲**, 需手工点[采集]才开始。
//   「一键采集」= 对本城所有**空闲驻军**(status=1, arrive_time=0)批量下达采集命令;
//   派新部队到未驻守的野地走「附属野地 → [采集]」。
//
// ★★ 2026-10-09 用户反馈「一键采集只对**当前城市**的采集空闲部队生效」：
//
//	原来只按 `user_id` 过滤 —— 玩家有多座城时，会把**别的城市**的空闲驻军也一起下达采集，
//	而军情→驻军 tab 是按当前城展示的（ReportDynamics 传 city_id）→ 玩家看到的列表里
//	根本没有那几支部队，却提示「已对 N 支下达」，与「一键召回」（已按 city_id 收口）也不一致。
//	现在与 一键召回 同口径：只处理 `city_id = 当前城` 的空闲驻军。
func (h *EzfyHandler) CollectAll(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	h.processOrders(uid)
	now := time.Now().UnixMilli()
	current := h.currentCity(uid)

	var orders []model.EzfyOrder
	h.DB.Where("user_id = ? AND city_id = ? AND status = 1 AND order_type = 7 AND arrive_time = 0",
		uid, current.ID).
		Order("id ASC").Find(&orders)
	if len(orders) == 0 {
		resp.ParamError(c, "当前城市没有空闲的驻军部队(派采集队请到「附属野地 → [采集]」; 已开始采集的部队等待结算即可)")
		return
	}
	// ★★ 2026-09-28 资源已满的守卫：一键采集前逐城判定，避免「采完了收获却是 0」。
	//
	// ★★ 2026-10-13 性能（用户反馈「一键采集 卡顿」）：
	//	原实现每支都要打 3 次跨库往返（查野地 + 查城判资源是否满 + 写订单），20 支 = 60 次，
	//	线上（1核1G + 跨 WAN）就是几秒的转圈。现在：
	//	  · 野地一次 `WHERE id IN (...)` 批量取回；
	//	  · 本接口只处理**当前城市**的空闲驻军 → 资源是否已满只需判一次（原来每支都判同一座城）；
	//	  · 整批的 arrive_time / collect_start **完全相同** → 一条 `WHERE id IN (...)` 批量下达。
	//	整批固定 3 次往返，与部队支数无关。
	atMax := h.ezfyAtResMax(int64(current.ID))
	wildIDs := make([]int64, 0, len(orders))
	for i := range orders {
		wildIDs = append(wildIDs, orders[i].TargetId)
	}
	wilds := h.batchWildlands(wildIDs)

	fullMsg := ""
	okIDs := make([]int64, 0, len(orders))
	fail, skip := 0, 0
	for i := range orders {
		order := &orders[i]
		wl := wilds[order.TargetId]
		if wl == nil || wl.CityId != order.CityId {
			h.beginReturn(order, now, 0)
			fail++
			continue
		}
		if atMax {
			// 起点城市五项资源全满 → 采集无意义，跳过（不报错，其它城照常采）
			fullMsg = "出发城市资源已达上限, 采集产出无法入库; 请先消耗资源或提升资源最大值配置"
			skip++
			continue
		}
		okIDs = append(okIDs, int64(order.ID))
	}
	// 全部因为「资源满」被跳过 → 直接以业务错误回，前端才会提示原因
	if len(okIDs) == 0 && skip > 0 && fail == 0 {
		resp.ParamError(c, fullMsg)
		return
	}
	// ★ 2026-10-13 性能：同一批 arrive_time / collect_start 完全相同 → 一条 UPDATE 批量下达
	//   （原每支一条 UPDATE，20 支就是 20 次跨库往返）。
	//   collect_start 记录采集起始，用于「累计采集时长」（★ 2026-09-28）。
	if len(okIDs) > 0 {
		h.DB.Model(&model.EzfyOrder{}).Where("id IN ?", okIDs).
			Updates(map[string]interface{}{
				"arrive_time":   now + ezfyDispatchPeriod(),
				"collect_start": now,
			})
	}
	msg := fmt.Sprintf("已对 %d 支空闲驻军下达采集命令(每满一个采集周期结算一期)", len(okIDs))
	if fail > 0 {
		msg += fmt.Sprintf("(%d 支野地已丢失, 部队自动返航)", fail)
	}
	if skip > 0 {
		msg += fmt.Sprintf("(%d 支跳过: 出发城市资源已达上限)", skip)
	}
	resp.OK(c, gin.H{"msg": msg})
}

// StartCollect POST /games/ezfy/wild/start-collect —— 单支空闲驻军开始采集
//
// ★ 2026-09-24 用户规则: 驻军没采集就是「空闲」状态, 手工点[采集]才进入采集状态。
//   驻军趋(情报→驻军)/野地列表/出征队列上的 [采集] 都走本接口。
func (h *EzfyHandler) StartCollect(c *gin.Context) {
	uid := middleware.GetUID(c)
	ezfyPageCacheDel(uid) // 开始采集 → 附属野地缓存失效
	h.cfgs()
	var req struct {
		OrderId int64 `json:"order_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.OrderId <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	h.processOrders(uid)
	now := time.Now().UnixMilli()
	var order model.EzfyOrder
	if err := h.DB.Where("id = ? AND user_id = ?", req.OrderId, uid).First(&order).Error; err != nil {
		resp.ParamError(c, "命令不存在")
		return
	}
	if order.Status != 1 || order.OrderType != 7 {
		resp.ParamError(c, "该部队不是驻守采集部队")
		return
	}
	if order.ArriveTime > 0 {
		resp.ParamError(c, "该部队已在采集中")
		return
	}
	// ★ 2026-10-08 一个野地同时只允许一个部队采集：
	//   本野地已另有部队在采集则直接拒绝（新采集卡控；历史已共存的部队不动，只看未来新发起的）。
	// ★ 2026-10-09 口径统一到 ezfyWildlandOccupiedCount（在途 order_type=4 也计入），
	//   与 createOrder 的下单入口同一套判定，避免「在途采集队」漏算。
	if n := h.ezfyWildlandOccupiedCount(order.TargetId, int64(order.ID)); n > 0 {
		resp.ParamError(c, "已有部队采集")
		return
	}
	var wl model.EzfyWildland
	if err := h.DB.First(&wl, order.TargetId).Error; err != nil || wl.CityId != order.CityId {
		resp.ParamError(c, "采集野地已丢失")
		return
	}
	// ★ 2026-09-28 用户规则「负重封顶+无自动停止」：负重满了多采部分丢弃、不再增加。
	//   已装满负重的部队不用补运输兵就点[采集]没意义，直接拦住。
	if h.ezfyCarryFull(&order) {
		resp.ParamError(c, "本部队负重已满, 可先[停止采集]取回负重或[召回]清空, 再继续采集")
		return
	}
	// ★★ 2026-09-28 采集资源已满的守卫：采了也入不了库，发起前就明确告知（别让玩家白等一轮）。
	if h.ezfyAtResMax(order.CityId) {
		resp.ParamError(c, "出发城市的资源已达上限, 采集产出无法入库; 请先消耗资源或提升资源最大值配置")
		return
	}
	order.ArriveTime = now + ezfyDispatchPeriod()
	h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
		Updates(map[string]interface{}{"arrive_time": order.ArriveTime, "collect_start": now})
	h.addReport(uid, 5, "采集报告: 开始采集",
		fmt.Sprintf("驻守在野地%d级(%d,%d)的部队开始采集, 每满一个采集周期结算一期。", wl.Level, wl.X, wl.Y), "", order.ID)
	resp.OK(c, gin.H{"msg": fmt.Sprintf("采集开始, 一个采集周期后首次结算(野地%d级 %d,%d)",
		wl.Level, wl.X, wl.Y)})
}

// HarvestAll POST /games/ezfy/wild/harvest-all —— 一键收获(批量)
//
// ★ 2026-09-28 用户规则澄清：**「一键收获」与「停止采集」是同一个功能**，
//   只是一个批量、一个单个 —— 都是「把已产出的资源收进起点城市」。
//   所以本接口 = 对每支在采集的部队走一遍「结算 → 入起点城 → 停止采集(原地待命)」。
//   （原实现只结算 carry 不落库、还要等召回返航，导致玩家以为资源丢了。）
//
// ★★ 2026-10-09 与「一键采集 / 一键召回」同口径：只处理**当前城市**出发的采集部队
//   （军情→驻军 tab 本身就是按当前城展示的，批量操作不该越界到别的城）。
func (h *EzfyHandler) HarvestAll(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	h.processOrders(uid)
	now := time.Now().UnixMilli()
	current := h.currentCity(uid)
	var req struct {
		Force bool `json:"force"` // ★ 2026-09-30 资源已满时的强制确认: 确认后 force=true 才真正结算
	}
	_ = c.ShouldBindJSON(&req)

	var orders []model.EzfyOrder
	h.DB.Where("user_id = ? AND city_id = ? AND order_type = 7 AND status = 1 AND arrive_time > 0",
		uid, current.ID).
		Order("id ASC").Find(&orders)
	if len(orders) == 0 {
		resp.ParamError(c, "当前城市没有正在采集的部队")
		return
	}

	// ★ 2026-09-30 任一采集部队的出发城市资源已达「资源最大值」时，
	//   一键收获产出入库会被资源上限封顶丢量，先整批确认一次。
	//
	// ★★ 2026-10-13 性能（用户反馈「一键收获 卡顿」）：
	//	原实现对**每支**采集部队独立走一遍完整结算，每支要打 10+ 次跨库往返
	//	（查野地 / 查城市 / 数宝物库存 / 插装备 / 写订单 / 插战报 / 战报补 city_id 回查 /
	//	 入城前后读两次城资源 + 写城资源 / 清空负重写订单 …）。
	//	玩家带 20 支采集队 = 200+ 次往返，线上（1核1G + 跨 WAN）就是「点了按钮一直转圈」。
	//	现在用 ezfySettleEnv 把「整批共享的只读数据」一次预取、「可攒批的写」攒到最后一次提交，
	//	每支只剩 1 次「原子认领」+ 1 次「订单结果写回」是必须逐支的。
	env := h.newSettleEnv(orders)
	if !req.Force {
		var cityID int64
		for i := range orders {
			// ★ 2026-10-13 性能：吃预取的城市对象判资源是否已满（原每支一条 SELECT）。
			if h.ezfyCityAtResMax(env.city(int64(orders[i].CityId))) {
				cityID = orders[i].CityId
				break
			}
		}
		if cityID > 0 {
			resp.OK(c, gin.H{
				"confirm": true,
				"msg": "有采集部队的出发城市资源已达配置的资源最大值, 本批一键收获的产出入库时会超出上限而被丢弃" +
					"(资源只会累加到资源最大值, 超出部分会消失)。确认仍要一键收获吗?",
			})
			return
		}
	}

	n := 0
	okIDs := make([]int64, 0, len(orders))
	// ★ 2026-10-13 采集宝物播报防刷屏：一键收获批量结算掉落的宝物合并成一条播报（见 ezfyLootBatch）。
	lootBatch := &ezfyLootBatch{}
	for i := range orders {
		order := &orders[i]
		// 满一个采集周期 → 完整结算(资源进负重+宝物); 不满 → 按采集时长折算资源(无宝物)
		if now >= order.ArriveTime {
			h.settleDispatch(uid, order, now, env, lootBatch)
		} else {
			h.settlePartialCollect(uid, order, now, "收获", env)
		}
		// ★ 2026-09-28 用户规则「负重封顶+无自动停止」：结算后把部队负重**入城**再原地待命。
		//   ★ 2026-10-13 性能：先记为待入账，由 envFlush 按城合并入库（原每支读两次城 + 写一次城）。
		h.envHarvestCarry(env, order)
		okIDs = append(okIDs, int64(order.ID))
		n++
	}
	// ★ 统一收口：收获后一律「停止采集」原地待命（与 [停止] 同一个语义）。
	//   ★ 2026-10-13 性能：整批一次批量落库（原每支一条 UPDATE）。
	if len(okIDs) > 0 {
		h.DB.Model(&model.EzfyOrder{}).Where("id IN ?", okIDs).
			Updates(map[string]interface{}{"arrive_time": 0, "collect_start": 0})
	}
	// ★ 2026-10-13 性能：战报 / 装备批量落库 + 采集资源按城合并入库，一次提交。
	gained, cleared := h.envFlush(env)
	// ★ 负重清空沿用「实际入账 > 0 才清」的口径（资源已满、被上限吞掉时保留负重，与原来一致）。
	if len(cleared) > 0 {
		h.DB.Model(&model.EzfyOrder{}).Where("id IN ?", cleared).
			Update("carry", "")
	}
	// ★ 2026-10-13 一键收获本批掉落的宝物合并成一条系统播报后发出。
	h.ezfyLootBatchFlush(uid, lootBatch)
	resp.OK(c, gin.H{"msg": fmt.Sprintf("已收获 %d 支采集部队, 资源已直接入库(共 %d), 部队原地待命", n, gained)})
}

// RecallAll POST /games/ezfy/wild/recall-all —— 一键召回
//
// ★ 2026-09-28 用户规则修正：资源在「收获/停止」时**已经直接入起点城市**，
//   所以召回**只是撤兵返航**，不再有「待带回资源」要等到达才入库
//   （原实现在这里既没落库 carry、又靠 finishReturn 清空 carry，造成「召回资源丢了」）。
//   为稳妥起见，召回前仍先做一次结算，保证「还在采但没收获过」的那部分资源不丢。
func (h *EzfyHandler) RecallAll(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	h.processOrders(uid)
	now := time.Now().UnixMilli()

	// ★ 2026-10-08 「一键召回只召回当前城市的」：以玩家**当前城市**(current_city_id)为起点，
	//   只召回**从这座城出发**的外出部队（采集/驻军），不再把玩家所有城的外出部队一起召回。
	current := h.currentCity(uid)

	var orders []model.EzfyOrder
	// ★ 2026-10-02 「一键召回」需覆盖出站驻军：
	//   驻守盟友城市的驻军(增援, status=3)也一并召回返航回出发城市。
	//   仅召回「活跃驻军」(result 为空 + 目标城属于他人)，避免旧僵尸/已归队订单重复入兵。
	allyCity := h.DB.Model(&model.EzfyCity{}).Select("id").Where("user_id <> ?", uid)
	h.DB.Where("user_id = ? AND city_id = ? AND ((order_type = 7 AND status = 1) OR (order_type = 6 AND status = 3 AND target_type = 3 AND result = '' AND target_id IN (?)))", uid, current.ID, allyCity).
		Order("id ASC").Find(&orders)
	if len(orders) == 0 {
		resp.ParamError(c, "当前城市没有可召回的外出部队(采集/驻军)")
		return
	}
	n := 0
	var gained int64
	for i := range orders {
		order := &orders[i]
		if order.OrderType == 7 {
			var before int64
			// 召回前先结算未入城的那部分产出（满一期给资源+宝物；不满一期只给按比例的资源），
			// 产出入负重(carry)，随返航到达时由 finishReturn 入城。
			if order.ArriveTime > 0 {
				before = parseCarry(order.Carry).total()
				h.settleDispatchOnRecall(uid, order, now)
				gained += parseCarry(order.Carry).total() - before
			}
			if order.Status != 1 {
				continue // 野地已丢失, settleDispatch 已把部队自动改成返航
			}
		}
		travel := ezfyOneWayTravel(order)
		h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
			Updates(map[string]interface{}{"status": 2, "result": order.Troops,
				"return_time": now + travel, "carry": order.Carry})
		n++
	}
	msg := fmt.Sprintf("已召回 %d 支部队返航", n)
	if gained > 0 {
		msg += fmt.Sprintf(", 召回前结算负重 %d(随返航入城)", gained)
	}
	resp.OK(c, gin.H{"msg": msg})
}

// StopCollect POST /games/ezfy/wild/stop-collect —— 单支采集部队「停止采集」
//
// ★ 2026-09-28 用户规则澄清：**「停止采集」与「一键收获」是同一个功能**（一个单个、一个批量）：
//   - 已满一个采集周期 → 完整结算(资源 + 宝物);
//   - 未满一个采集周期 → 只按驻守时长折算资源, **没有宝物**;
//   - 资源**直接入起点城市**（不再装进部队待带回）;
//   - 停止后部队原地待命(驻守空闲 arrive_time=0)，之后可再点[采集]继续，或[召回]撤兵。
func (h *EzfyHandler) StopCollect(c *gin.Context) {
	uid := middleware.GetUID(c)
	ezfyPageCacheDel(uid) // 停止采集 → 附属野地缓存失效
	h.cfgs()
	var req struct {
		OrderId int64 `json:"order_id"`
		Force   bool `json:"force"` // ★ 2026-09-30 资源已满时的强制确认: 玩家确认后带 force=true 才真正结算
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.OrderId <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	h.processOrders(uid)
	now := time.Now().UnixMilli()
	var order model.EzfyOrder
	if err := h.DB.Where("id = ? AND user_id = ?", req.OrderId, uid).First(&order).Error; err != nil {
		resp.ParamError(c, "命令不存在")
		return
	}
	if order.Status != 1 || order.OrderType != 7 {
		resp.ParamError(c, "该部队不是驻守采集部队")
		return
	}
	if order.ArriveTime <= 0 {
		resp.ParamError(c, "该部队已在待命(未在采集中)")
		return
	}
	// ★ 2026-09-30 出发城市资源已达到配置的「资源最大值」时，
	//   本次收获产出入库会被资源上限封顶丢量（其余资源只会累加到资源最大值、超出的会消失）。
	//   先向玩家确认：是否仍要停止/收获；确认后(force=true)才真正结算入库。
	if !req.Force && h.ezfyAtResMax(order.CityId) {
		resp.OK(c, gin.H{
			"confirm": true,
			"msg": "出发城市资源已达配置的资源最大值, 本次停止收获的产出入库时会超出上限而被丢弃" +
				"(资源只会累加到资源最大值, 超出部分会消失)。确认仍要停止采集并取回吗?",
		})
		return
	}
	if now >= order.ArriveTime {
		h.settleDispatch(uid, &order, now, nil) // 满一期: 完整结算(资源+宝物)
	} else {
		h.settlePartialCollect(uid, &order, now, "停止采集", nil) // 未满一期: 只有资源
	}
	// ★ 2026-09-28 用户规则「负重封顶+无自动停止」：停止采集 = 取回部队负重入城
	gained := h.harvestCarryToCity(&order)
	// ★★ 修复「停止没停」：结算/入城后必须把采集状态清零(原地待命)。
	//   否则 arrive_time 仍 >0（满一期时 settleDispatch 还会推进到下一期），
	//   部队会一直处于采集中、可被反复[停止]/[收获]刷资源。
	h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
		Updates(map[string]interface{}{"arrive_time": 0, "collect_start": 0})
	resp.OK(c, gin.H{"msg": fmt.Sprintf("已停止采集, 负重资源 %d 已入库, 部队原地待命(可再[采集]继续或[召回]撤兵)",
		gained)})
}

// dedupStrings 去重(保持顺序), 用于把重复的失败原因合并
func dedupStrings(list []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, s := range list {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

// ezfyWildlandCollectingCount 该野地上当前「正在采集」的部队数（排除自己 orderId）。
//
// ★ 2026-10-08 附属野地「一个野地同时只允许一个部队采集」的新卡控：
//
//	采集中 = order_type 7(驻守采集) + status 1 + arrive_time > 0。
//	只看最新发起的采集是否撞上正在采集的部队；历史已共存的采集部队不在此列（不动它们）。
func (h *EzfyHandler) ezfyWildlandCollectingCount(targetID, excludeOrderID int64) int64 {
	var n int64
	h.DB.Model(&model.EzfyOrder{}).
		Where("target_id = ? AND order_type = 7 AND status = 1 AND arrive_time > 0 AND id != ?",
			targetID, excludeOrderID).
		Count(&n)
	return n
}

// ezfyWildlandOccupiedCount 该野地上当前**所有未结束**的部队数（排除 excludeOrderID）。
//
// ★★ 2026-10-09 新增：把「一块附属野地同时只能有一支队伍」的卡控统一到**下单入口**。
//
//	上面 `ezfyWildlandCollectingCount` 只看「order_type=7 且已在采集(arrive_time>0)」，
//	是**旧口径**、只在 `StartCollect` 那一步用 —— 拦不住「附属野地 → [采集]」直接
//	下发新采集订单（order_type=4）这条真正的漏洞路径（见 createOrder 里的说明）。
//
//	本函数口径 = 覆盖该野地的**全部活跃队伍**：
//	  · 在途部队（order_type=4，采购中还没到）—— status IN (0, 1, 5, 6, 98)，即未结束；
//	  · 驻守/采集部队（order_type=7，status=1）—— 空闲待命或采集中都算「已占位」。
//	只算未结束态（3 完成 / 4 阵亡 等历史行不计），避免被旧数据永久挡住。
func (h *EzfyHandler) ezfyWildlandOccupiedCount(targetID, excludeOrderID int64) int64 {
	if targetID <= 0 {
		return 0
	}
	var n int64
	h.DB.Model(&model.EzfyOrder{}).
		Where("target_id = ? AND ((order_type = 4 AND status IN (0, 1, 5, 6, 98)) OR (order_type = 7 AND status = 1)) AND id != ?",
			targetID, excludeOrderID).
		Count(&n)
	return n
}

// ezfyClearWildlandAt 清掉某坐标上的**全部**野地记录（含历史重复行），返回 (被清掉的行数, 被转返航的部队数)。
//
// ★★ 2026-10-16 新增。**一个坐标只允许存在一条野地记录**（归属唯一）——
//
//	调用点：① 野地被他人征服占领（`ezfy_order.go` 占领段）；
//	        ② 主动放弃野地（`AbandonWildland`）。
//
// 为什么不能直接 DELETE：野地行上可能挂着**未结束**的采集(4)/驻守(7)部队
// （`order.target_id = 野地 id`），直接删行会让这些部队的 target_id 悬空
// → 部队凭空消失 / 结算异常（放弃野地的历史教训，见 AbandonWildland 注释）。
// 所以先把它们转成返航，再删行。
//
// 部队口径与 `ezfyWildlandOccupiedCount` 完全对齐（「算作占用该野地」= 「会被转返航」）：
// order_type=4 的 status(0,1,5,6,98) + order_type=7 的 status=1。
func (h *EzfyHandler) ezfyClearWildlandAt(x, y int, now int64) (rows int, orders int) {
	var olds []model.EzfyWildland
	h.DB.Where("x = ? AND y = ?", x, y).Find(&olds)
	if len(olds) == 0 {
		return 0, 0
	}
	ids := make([]int64, 0, len(olds))
	for i := range olds {
		ids = append(ids, int64(olds[i].ID))
	}
	var gos []model.EzfyOrder
	h.DB.Where("target_id IN ? AND ((order_type = 4 AND status IN (0,1,5,6,98)) OR (order_type = 7 AND status = 1))", ids).
		Find(&gos)
	for i := range gos {
		o := &gos[i]
		travel := ezfyOneWayTravel(o)
		o.Status = 2
		o.Result = o.Troops
		o.ReturnTime = now + travel
		h.DB.Model(&model.EzfyOrder{}).Where("id = ?", o.ID).
			Updates(map[string]interface{}{"status": 2, "result": o.Result,
				"return_time": o.ReturnTime, "carry": o.Carry})
	}
	h.DB.Where("id IN ?", ids).Delete(&model.EzfyWildland{})
	return len(olds), len(gos)
}
