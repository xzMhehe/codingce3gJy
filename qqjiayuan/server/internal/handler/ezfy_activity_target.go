package handler

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/model"
)

// 二战风云 活动目标：活动野地 / 活动寇城 / 特殊城市
//
// 复刻 GameServiceImpl：
//   isActivityWildland / isActivityKouCity / isSpecialCity / getActivityLevel
//   buildActivityDefender / processActivityBattle
// 以及 MapController 里给格子打的 actWild / actKou / actCity 标记。
//
// 与普通野地/寇城的区别：
//   1. 位置由坐标哈希固定生成（不是随机刷、不落库），全服一致；
//   2. 守军按「活动等级 1~3」成倍增长，最强是特殊城市（100万/200万/500万）；
//   3. 打赢只结算奖励（资源/黄金/必掉宝物/大量声望），**不占领**，也不占用附属野地上限；
//   4. 战报统一是「战斗报告」，标题形如「特殊城市3级战斗报告: 特殊城市」。

// 活动目标类型
const (
	ezfyActNone = 0 // 非活动目标
	ezfyActWild = 1 // 活动野地
	ezfyActKou  = 2 // 活动寇城
	ezfyActCity = 3 // 特殊城市
)

// ============ 坐标哈希判定（复刻 GameServiceImpl） ============

// ezfyActivityWildland 活动野地：哈希命中 499 且距世界中心 > 100
func ezfyActivityWildland(x, y int) bool {
	if ezfyAbs(x*1729^y*3803)%499 != 0 {
		return false
	}
	return ezfyAbs(x-250)+ezfyAbs(y-250) > 100
}

// ezfyActivityKouCity 活动寇城：哈希命中 1499 且距世界中心 > 120
func ezfyActivityKouCity(x, y int) bool {
	if ezfyAbs(x*9157^y*2657)%1499 != 0 {
		return false
	}
	return ezfyAbs(x-250)+ezfyAbs(y-250) > 120
}

// ezfySpecialCity 特殊城市：哈希命中 2999 且距世界中心 > 150
func ezfySpecialCity(x, y int) bool {
	if ezfyAbs(x*6689^y*5347)%2999 != 0 {
		return false
	}
	return ezfyAbs(x-250)+ezfyAbs(y-250) > 150
}

// ezfyActivityLevel 活动目标等级 1~3（复刻 getActivityLevel）
func ezfyActivityLevel(x, y int) int {
	return ezfyAbs(x*817^y*1013)%3 + 1
}

// ezfyActTargetType 判定某格的「活动目标」类型（0=无 1=活动野地 2=活动寇城 3=特殊城市）
// 复刻 MapController 的三条互斥规则：
//
//	actKou  = isKou      && isActivityKouCity
//	actWild = !isKou                 && isActivityWildland
//	actCity = !isKou && !actWild     && isSpecialCity
func (h *EzfyHandler) ezfyActTargetType(x, y int) int {
	// ★ 管理端「地图格子覆盖」优先：某格被显式标成活动目标就用它，
	//   否则照旧按坐标哈希推导。
	switch ezfyMarkKindAt(x, y) {
	case model.EzfyMarkActKou:
		return ezfyActKou
	case model.EzfyMarkActWild:
		return ezfyActWild
	case model.EzfyMarkActCity:
		return ezfyActCity
	case model.EzfyMarkKou:
		return ezfyActNone // 普通寇城不算活动目标
	}
	return ezfyActTypeFor(x, y, h.ezfyIsKouCity(x, y))
}

// ezfyActTypeFor 纯函数版本（寇城判定由调用方传入，避免地图逐格重复查库）
func ezfyActTypeFor(x, y int, kou bool) int {
	if kou {
		if ezfyActivityKouCity(x, y) {
			return ezfyActKou
		}
		return ezfyActNone
	}
	if ezfyActivityWildland(x, y) {
		return ezfyActWild
	}
	if ezfySpecialCity(x, y) {
		return ezfyActCity
	}
	return ezfyActNone
}

// ezfyActTargetName 活动目标中文名
func ezfyActTargetName(actType int) string {
	switch actType {
	case ezfyActWild:
		return "活动野地"
	case ezfyActKou:
		return "活动寇城"
	case ezfyActCity:
		return "特殊城市"
	}
	return ""
}

// ezfyActTargetDesc 活动目标详情页说明文案（照 activityIndex.html）
func ezfyActTargetDesc(actType int) string {
	switch actType {
	case ezfyActWild:
		return "陆/海随机刷新, 10万~30万守军, 胜利获得大量资源+黄金(元宝)+必定掉落宝物+大量声望"
	case ezfyActKou:
		return "20万~60万守军, 胜利获得巨大资源+黄金+宝物+声望"
	case ezfyActCity:
		return "100万~500万守军, 全服最强活动目标, 需要强力的部队! 胜利必定获得高级/特殊宝物, 巨额黄金与资源"
	}
	return ""
}

// ezfyActTargetLabel 「活动野地2级」这样的完整标签
func ezfyActTargetLabel(actType, level int) string {
	return ezfyActTargetName(actType) + strconv.Itoa(level) + "级"
}

// ============ 活动守军（复刻 buildActivityDefender） ============

// ezfyActivityDefender 活动目标守军
// 陆地：装甲车 50% / 轻型坦克 30% / 重型坦克 20%
// 海洋：驱逐舰 40% / 潜艇 30% / 战列舰 20% / 航母 10%
func ezfyActivityDefender(actType, level, terrain int) []ezfyUnitGroup {
	var base int64
	switch actType {
	case ezfyActKou:
		base = 200000 // 活动寇 20万/40万/60万
	case ezfyActCity:
		base = 1000000 // 特殊城市 100万/200万/500万
	default:
		base = 100000 // 活动野地 10万/20万/30万
	}
	total := base * int64(level)
	if actType == ezfyActCity && level == 3 {
		total = base * 5
	}
	ratio := [][2]int{{4, 50}, {5, 30}, {6, 20}} // 装甲车 / 轻型坦克 / 重型坦克
	if terrain == 8 {
		ratio = [][2]int{{13, 40}, {14, 30}, {15, 20}, {16, 10}} // 驱逐舰 / 潜艇 / 战列舰 / 航母
	}
	out := []ezfyUnitGroup{}
	for _, r := range ratio {
		out = append(out, ezfyUnitGroup{TroopId: r[0], Count: total * int64(r[1]) / 100})
	}
	return out
}

// ============ 活动战斗结算（复刻 processActivityBattle） ============

// processActivityBattle 活动目标战斗结算（独立于普通野地/寇城流程，打赢不占领）
func (h *EzfyHandler) processActivityBattle(uid uint, city *model.EzfyCity, order *model.EzfyOrder,
	now int64, actType int) {

	level := ezfyActivityLevel(order.TargetX, order.TargetY)
	terrain := ezfyTerrain(order.TargetX, order.TargetY)
	label := ezfyActTargetLabel(actType, level)

	defender := ezfyActivityDefender(actType, level, terrain)
	attacker := parseGroups(order.Troops)

	// 攻方加成与普通出征完全一致（军官 + 科技 + 技能）
	atkTech := h.techMap(city.ID)
	leadOfficer := h.officerByName(city.ID, order.Officer)
	atkBonus := h.officerBattleBonus(leadOfficer) +
		atkTech[5]*2 + atkTech[6]*3 + atkTech[8]*3 + atkTech[9]*2
	atkSpeedBonus := atkTech[10]*2 + atkTech[19]*3
	if h.officerSpeedSkill(leadOfficer) {
		atkSpeedBonus += 10
	}
	atkOfficerDesc := h.officerBattleDesc(leadOfficer, h.officerBaseBonus(leadOfficer), "攻击加成")

	// ★★ 指挥室（2026-09-22 用户要求）：活动目标也是战斗，同样先开战场等玩家指挥，
	//   与普通野地/寇城/玩家城保持一致（否则打活动城不能指挥，玩家会困惑）。
	//   BattleResult 非空 = 已在指挥室里打完，直接用结果结算。
	// ★ 守方阵营: 活动野地=盟军、活动寇/特殊城市=轴心国(战报兵种名按阵营显示)
	defCamp := 1
	if actType == ezfyActKou || actType == ezfyActCity {
		defCamp = 2
	}
	var br ezfyBattleResult
	if done, ok := ezfyBattleResultDecode(order.BattleResult); ok {
		br = done
	} else {
		// 活动守军无城墙/无科技/无城守 → 防守方加成为 0（复刻原版传 0 与空 map）
		// ★ 攻方装备六项加成照常生效；守方阵营: 活动野地=盟军、活动寇/特殊城市=轴心国
		st := ezfyNewBattleState(attacker, defender, atkBonus, 0, atkSpeedBonus, 0,
			h.officerBattleEquipBonus(leadOfficer), ezfyBattleBonus{},
			atkOfficerDesc, "", h.buildTargetMap(city.ID, true), map[int]int{},
			h.buildMoveMap(city.ID, true), map[int]int{},
			h.officerHasSkill(leadOfficer, "绝地反击"), false,
			h.ensureProfile(uid).Camp, defCamp)
		// ★★ 2026-09-27 用户反馈「活动野地不能指挥/没有战报/资源不累加」：
		//   根因是活动流程缺少普通野地的「目标被抢先指挥 → 等待」机制（见 ezfyOrderTargetBusy）。
		//   多支部队打同一活动目标时，后到部队复用先到部队的战场（b.OrderId 是别人的订单），
		//   自己查不到战场 → 点[指挥]报「没有战斗记录」；
		//   且每次结算 battle_result 解码为空 → 重新开新战场 → 永远打不完（无战报、资源不加）。
		//   这里与普通野地流程(L2043)同款：目标正忙 → 进入「等待(6)」，上一场打完后
		//   processOrders 自动放行重进指挥。
		if ezfyOrderTargetBusy(h, order, int64(order.ID)) {
			order.Status = ezfyOrderStatusWaiting
			h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
				Update("status", ezfyOrderStatusWaiting)
			return
		}
		if b := h.ezfyBattleStart(uid, order, st, label, now); b != nil {
			order.Status = ezfyOrderStatusBattle
			h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).
				Update("status", ezfyOrderStatusBattle)
			// ★ 同普通战斗：「等待指挥」不发战报，避免污染战斗报告列表
			return
		}
		// 开战场失败（极端情况）→ 兜底直接模拟，绝不让部队卡住
		for !st.Done {
			st.Step(nil, nil)
		}
		br = st.Result()
	}
	win := br.AttackerWin
	draw := br.Draw

	// ★ 带队军官战功经验（2026-09-27 用户反馈「战败军官经验是 0」）：
	//   活动流程原来完全没给军官经验（打赢打输都是 0）。统一走普通战斗同款口径
	//   ezfyOfficerBattleExp —— 基础经验 = 击杀数/10 + 30，**即便战败、0 击杀也有 30 点**，
	//   胜利再多拿 50%。写在函数开头只算数值，战报正文与写库放在结果判定后统一处理。
	atkExp := int64(0)
	if leadOfficer != nil {
		atkExp = ezfyOfficerBattleExp(enemyDeadOf(br.DefenderLosses), win)
	}

	profile := h.ensureProfile(uid)
	report := fmt.Sprintf("主题:战斗报告\n出发地:%s(%d,%d)\n目的地:%s(%d,%d)\n时间:%s\n公文报告:战斗报告\n我方一支部队对%s[ %d，%d ]发起了进攻。战斗共持续 %d 回合，我方战斗%s\n",
		city.Name, city.X, city.Y, label, order.TargetX, order.TargetY,
		time.UnixMilli(now).Format("2006-01-02 15:04"), label,
		order.TargetX, order.TargetY, br.Rounds, battleOutcomeText(win, draw))
	report += fmt.Sprintf("统帅声望:%d\n", profile.Prestige)
	if leadOfficer != nil {
		report += "军官:" + officerReportDesc(leadOfficer) + "\n"
	}

	atkBefore := groupCounts(attacker)
	atkAfter := groupCounts(br.AttackerLeft)
	// ★ 2026-09-24 用户要求：40 回合平局时双方标签都显示 [平]
	atkTag, defTag := winText(win), winText(!win)
	if draw {
		atkTag, defTag = "平", "平"
	}
	report += fmt.Sprintf("[%s]攻方:%s\n", atkTag, city.Name)
	report += troopChangeText(atkBefore, atkAfter, profile.Camp)
	report += fmt.Sprintf("--------------------\n[%s]守方:%s\n", defTag, label)
	defBefore := groupCounts(defender)
	defAfter := map[int]int64{}
	for _, g := range br.DefenderLosses {
		defAfter[g.TroopId] = maxInt64(0, defBefore[g.TroopId]-g.Count)
	}
	for tid, cnt := range defBefore {
		if _, ok := defAfter[tid]; !ok {
			defAfter[tid] = cnt
		}
	}
	report += troopChangeText(defBefore, defAfter, defCamp)

	// 详细战报：逐回合过程 + 双方兵力变化
	detail := ""
	for _, a := range br.Actions {
		detail += a + "\n"
	}
	detail += "\n[双方兵力]\n"
	detail += fmt.Sprintf("[%s]攻方:%s\n", atkTag, city.Name)
	if leadOfficer != nil {
		detail += "军官:" + officerReportDesc(leadOfficer) + "\n"
	}
	detail += troopChangeText(atkBefore, atkAfter, profile.Camp)
	detail += fmt.Sprintf("--------------------\n[%s]守方:%s\n", defTag, label)
	detail += troopChangeText(defBefore, defAfter, defCamp)
	detail += "[双方兵力]"

	// 攻方战损入伤兵营：兵种修复率% + 治愈伤兵科技 2%/级 + 机械改造 10%
	healTech := atkTech[21] * 2
	if h.officerHasSkill(leadOfficer, "机械改造") {
		healTech += 10
	}
	var repairedTotal int64
	for _, g := range br.AttackerLosses {
		if g.Count <= 0 {
			continue
		}
		rate := 10
		if cfg := ezfyCfg.troop(g.TroopId); cfg != nil {
			rate = cfg.RepairRate
		}
		rate += healTech
		wounded := g.Count * int64(rate) / 100
		if wounded > g.Count {
			wounded = g.Count
		}
		if wounded > 0 {
			h.addWounded(city.ID, g.TroopId, 0, wounded)
			repairedTotal += wounded
		}
	}

	// 剩余部队（返航后归还）
	resultStr := ""
	for _, g := range br.AttackerLeft {
		if g.Count > 0 {
			resultStr += strconv.Itoa(g.TroopId) + ":" + strconv.FormatInt(g.Count, 10) + ","
		}
	}
	if len(resultStr) > 0 {
		resultStr = resultStr[:len(resultStr)-1]
	}
	order.Result = resultStr

	prestigeGain := 0
	if win {
		// 奖励：资源（档位基础值 × 类型倍数）+ 黄金（元宝）+ 必定掉宝 + 大量声望
		res := ezfyActRewardBase(actType, level)
		// ★ 2026-09-27 活动城奖励必须「累加」到「资源最大值」，不能按仓储上限 clamp：
		//   旧写法 min64(city.XxxCap*3, old+res) 会把「已超过仓储 3 倍」的存量直接拉低，
		//   导致打一次活动城资源反而变少（用户反馈「资源会掉」）。
		//   统一走 ezfyAddResMax（结果 = max(现值, min(资源最大值, 现值+增量))），与生产/入库同口径：
		//   无条件累加、只受资源最大值约束、老值超出也不被拉低。
		city.Food = ezfyAddResMax("food", city.Food, res)
		city.Steel = ezfyAddResMax("steel", city.Steel, res)
		city.Oil = ezfyAddResMax("oil", city.Oil, res)
		city.Rare = ezfyAddResMax("rare", city.Rare, res)
		gold := ezfyActGoldReward(actType, level)
		city.Gold = ezfyAddResMax("gold", city.Gold, gold)
		h.saveCityRes(city)

		report += fmt.Sprintf("\n战利品: 粮%d 钢%d 油%d 稀矿%d 黄金%d", res, res, res, res, gold)
		if loot := h.wildlandLoot(city, level*3, terrain, true); loot != "" {
			report += "\n" + loot
		}
		prestigeGain = ezfyActPrestigeReward(actType, level)
		h.addPrestige(uid, prestigeGain)
		report += fmt.Sprintf("\n军功声望+%d", prestigeGain)
		order.Status = 2
		report = "我军胜利!\n" + report
	} else if draw {
		// ★ 2026-09-24 用户要求：40 回合平局——残部返航（不是阵亡，不能 status=4 吃掉幸存部队）
		order.Status = 2
		report = "我军与敌军打成平局!\n" + report
	} else {
		order.Status = 4 // 全队阵亡
		report = "我军战败!\n" + report
	}
	// ★ 带队军官经验：胜负平局**都给**（用户要求「战败也有经验」），与普通战斗同款口径。
	//   即便战败/0 击杀也有基数 30 点；写库放这里统一处理（函数开头只算了数值）。
	if leadOfficer != nil && atkExp > 0 {
		h.addOfficerExp(city, leadOfficer.ID, atkExp)
		report += fmt.Sprintf("\n军官经验+%d", atkExp)
	}
	if repairedTotal > 0 {
		report += fmt.Sprintf("\n伤兵入营: %d", repairedTotal)
	}
	report += h.battleStatsTail(uid, prestigeGain, 50)

	// 返航时长与去程一致
	order.ReturnTime = now + ezfyAbs64(order.ArriveTime-order.StartTime)
	h.DB.Model(&model.EzfyOrder{}).Where("id = ?", order.ID).Updates(map[string]interface{}{
		"status": order.Status, "result": order.Result, "return_time": order.ReturnTime,
	})
	// ★ 用户反馈：标题「活动野地3级战斗报告: 活动野地」冒号后面没带等级，看着像缺了目标。
	//   统一成「活动野地3级战斗报告: 活动野地3级(323,69)」，与普通战报「地形N级(坐标)」一致。
	h.addReport(uid, 3, label+"战斗报告: "+label+"("+strconv.Itoa(order.TargetX)+","+strconv.Itoa(order.TargetY)+")",
		report, detail, order.ID)
}

// ezfyActRewardBase 活动目标基础奖励（复刻 processActivityBattle 的 resBase 与类型倍数）
func ezfyActRewardBase(actType, level int) int64 {
	resBase := []int64{100000, 250000, 600000}
	res := resBase[minInt(2, level-1)]
	switch actType {
	case ezfyActKou:
		res *= 2
	case ezfyActCity:
		res *= 5
	}
	return res
}

// ezfyActGoldReward 活动目标黄金奖励（复刻 5000/10000/20000 × 等级）
func ezfyActGoldReward(actType, level int) int64 {
	base := int64(5000)
	switch actType {
	case ezfyActKou:
		base = 10000
	case ezfyActCity:
		base = 20000
	}
	return base * int64(level)
}

// ezfyActPrestigeReward 活动目标声望奖励（复刻 200 × 等级 × 类型倍数）
func ezfyActPrestigeReward(actType, level int) int {
	p := 200 * level
	switch actType {
	case ezfyActKou:
		p *= 3
	case ezfyActCity:
		p *= 5
	}
	return p
}

// ezfyActWildlandView 活动目标详情（守军预览 + 奖励预览 + 说明）
func (h *EzfyHandler) ezfyActWildlandView(uid uint, camp, x, y, actType int) gin.H {
	level := ezfyActivityLevel(x, y)
	terrain := ezfyTerrain(x, y)
	defs := ezfyActivityDefender(actType, level, terrain)
	troops := []gin.H{}
	var total int64
	for _, g := range defs {
		total += g.Count
		troops = append(troops, gin.H{
			"troop_id": g.TroopId, "name": ezfyCfg.troopName(g.TroopId, camp),
			"min": g.Count, "max": g.Count,
		})
	}
	jewelName := ""
	if j := h.randomJewel(terrain); j != nil {
		jewelName = j.Name
	}
	return gin.H{
		"x": x, "y": y, "type": 1, "level": level,
		"name":     ezfyActTargetLabel(actType, level),
		"act_type": actType, "act_level": level,
		"act_name":     ezfyActTargetName(actType),
		"act_desc":     ezfyActTargetDesc(actType),
		"act_total":    total,
		"troops":       troops,
		"res_min":      ezfyActRewardBase(actType, level),
		"res_max":      ezfyActRewardBase(actType, level),
		"gold":         ezfyActGoldReward(actType, level),
		"prestige":     ezfyActPrestigeReward(actType, level),
		"terrain":      terrain,
		"terrain_name": ezfyTerrainName(ezfyTerrainEx(x, y)),
		"continent":    ezfyContinentName(x, y),
		"jewel":        jewelName,
		"owner":        "",
	}
}
