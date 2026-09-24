package handler

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 二战风云 管理端（第九轮新增）
//
//	1. 建筑数量上限配置（军事区/资源区各 33，管理端可维护）
//	2. 钻石发放（钻石只能管理端发放，玩家端只读余额）
//	3. 二战聊天敏感词（独立维护页，与社区「黑名单榜」分开）

// ============ 1. 建筑数量上限配置 ============

// AdminEzfyBuildLimitGet GET /admin/ezfy-build-limit
//
// 管理端「系统配置」页：建筑上限 + 各项数值 + 玩法开关，一次全量返回。
func (h *AdminHandler) AdminEzfyBuildLimitGet(c *gin.Context) {
	lim := model.EzfyCfgLimit{ID: 1, MilitaryMax: 33, ResourceMax: 33, HouseMax: 10, FactoryMax: 0,
		GatherMaxPerOrder: ezfyGatherMaxDefault, MallBuyMax: ezfyMallBuyMaxDef,
		ConquerFeelingsMax: ezfyConquerFeelingsDef, LootFeelings: ezfyLootFeelingsDef,
		OfficerSalaryPerLevel: ezfyOfficerSalaryDef, WoundHealDivisor: ezfyWoundHealDivisorDef,
		WildTroopMult: ezfyWildMultDef,
		// ★ 三个开关的默认值都写进初始值：新建行时 GORM 会显式写 1（列上没有 gorm default 标签）
		RecruitCostOn: ezfyRecruitCostDef, FoodUpkeepOn: ezfyFoodUpkeepDef, MarchOilOn: ezfyMarchOilDef,
		WarRequireOn: ezfyWarRequireDef, MarchCapOn: ezfyMarchCapDef,
		// ★ 军官升星（简化后：功能开关 + 固定成功率 + 每星加成 + 星级上限）
		OfficerStarUpOn:   ezfyStarUpDef,
		OfficerStarChance: ezfyStarChanceDef, OfficerStarAttrGain: ezfyStarAttrGainDef,
		OfficerStarMax: ezfyStarMaxDef,
		// ★ 训练加速黄金倍率 / 伤兵恢复黄金折扣率（百分比口径：100 = 100% = 原价）+ 伤兵恢复黄金折扣率
		SpeedTrainRate: 100, WoundHealRate: 100,
		// ★ 2026-09-23 线上「负数兵力」事故：单城兵力上限 + 伤兵存活天数
		TroopMax: ezfyTroopMaxDef, WoundExpireDays: ezfyWoundExpireDaysDef}
	if err := h.DB.First(&lim, 1).Error; err != nil {
		h.DB.Create(&lim)
	}
	// ★ 商城单次购买上限兜底（0 无意义 = 禁止购买），默认 9999
	if lim.MallBuyMax <= 0 {
		lim.MallBuyMax = ezfyMallBuyMaxDef
	}
	// ★ 集结令上限兜底：老行没这列时可能是 0，回落到默认 50（0 无意义 = 禁用道具）
	if lim.GatherMaxPerOrder <= 0 {
		lim.GatherMaxPerOrder = ezfyGatherMaxDefault
	}
	// ★ 战斗/经济数值兜底：这几个 0 同样无意义（0 = 不扣民心 / 军官免费 / 恢复免费）
	if lim.ConquerFeelingsMax <= 0 {
		lim.ConquerFeelingsMax = ezfyConquerFeelingsDef
	}
	if lim.LootFeelings <= 0 {
		lim.LootFeelings = ezfyLootFeelingsDef
	}
	if lim.OfficerSalaryPerLevel <= 0 {
		lim.OfficerSalaryPerLevel = ezfyOfficerSalaryDef
	}
	if lim.WoundHealDivisor <= 0 {
		lim.WoundHealDivisor = ezfyWoundHealDivisorDef
	}
	// ★ 野地兵力倍数：0 / 负数无意义 → 回落 1
	if lim.WildTroopMult <= 0 {
		lim.WildTroopMult = ezfyWildMultDef
	}
	// ★ 军官升星的数值项：0 无意义 → 回落默认值（开关项不兜底，0 = 关）
	if lim.OfficerStarChance <= 0 {
		lim.OfficerStarChance = ezfyStarChanceDef
	}
	if lim.OfficerStarAttrGain <= 0 {
		lim.OfficerStarAttrGain = ezfyStarAttrGainDef
	}
	if lim.OfficerStarMax <= 0 {
		lim.OfficerStarMax = ezfyStarMaxDef
	}
	// ★ 训练加速黄金倍率：0 / 负数无意义 → 回落 100（100 = 100% = 原价）
	if lim.SpeedTrainRate <= 0 {
		lim.SpeedTrainRate = 100
	}
	// ★ 伤兵恢复黄金折扣率：0 / 负数无意义 → 回落 100
	if lim.WoundHealRate <= 0 {
		lim.WoundHealRate = 100
	}
	// ★ 2026-09-23：单城兵力上限 / 伤兵存活天数（0 无意义 → 回落默认值）
	if lim.TroopMax <= 0 {
		lim.TroopMax = ezfyTroopMaxDef
	}
	if lim.WoundExpireDays <= 0 {
		lim.WoundExpireDays = ezfyWoundExpireDaysDef
	}
	// ★ 2026-09-24：采集周期小时数（0 无意义 → 回落默认 4 小时）
	if lim.DispatchPeriodH <= 0 {
		lim.DispatchPeriodH = 4
	}
	// ★ 三个开关**不做** <= 0 兜底：0 就是「关」，是合法值。
	//   只有 NULL 才是没配过（列是后来补的），seed 启动时已回填 1。
	resp.OK(c, lim)
}

// AdminEzfyBuildLimitUpdate PUT /admin/ezfy-build-limit
//
// 军事区(type 2/3/4) 与 资源区(type 1) 的数量上限**分开**维护，默认各 33。
// factory_max = 0 表示军工厂不限数量（默认，符合用户规则）。
func (h *AdminHandler) AdminEzfyBuildLimitUpdate(c *gin.Context) {
	var in struct {
		MilitaryMax           *int `json:"military_max"`
		ResourceMax           *int `json:"resource_max"`
		HouseMax              *int `json:"house_max"`
		FactoryMax            *int `json:"factory_max"`
		NoticeHomeCount       *int `json:"notice_home_count"`
		GatherMaxPerOrder     *int `json:"gather_max_per_order"`
		MallBuyMax            *int `json:"mall_buy_max"`
		ConquerFeelingsMax    *int `json:"conquer_feelings_max"`
		LootFeelings          *int `json:"loot_feelings"`
		OfficerSalaryPerLevel *int `json:"officer_salary_per_level"`
		WoundHealDivisor      *int `json:"wound_heal_divisor"`
		// ★ 系统配置新增：野地兵力倍数 + 四个玩法开关
		WildTroopMult *float64 `json:"wild_troop_mult"`
		RecruitCostOn *int     `json:"recruit_cost_on"`
		FoodUpkeepOn  *int     `json:"food_upkeep_on"`
		MarchOilOn    *int     `json:"march_oil_on"`
		WarRequireOn  *int     `json:"war_require_on"`
		MarchCapOn    *int     `json:"march_cap_on"`
		// ★ 军官升星（简化后：功能开关 + 数值）
		OfficerStarUpOn    *int `json:"officer_star_up_on"`
		OfficerStarChance  *int `json:"officer_star_chance"`
		OfficerStarAttrGain *int `json:"officer_star_attr_gain"`
		OfficerStarMax     *int `json:"officer_star_max"`
		// ★ 训练加速黄金倍率 / 伤兵恢复黄金折扣率（百分比口径，100 = 100% = 原价，节假日调低 = 便宜）
		SpeedTrainRate *float64 `json:"speed_train_rate"`
		WoundHealRate  *float64 `json:"wound_heal_rate"`
		// ★ 2026-09-23 线上「负数兵力」事故：单城兵力上限 + 伤兵存活天数
		TroopMax        *int64 `json:"troop_max"`
		WoundExpireDays *int   `json:"wound_expire_days"`
		// ★ 2026-09-24：采集结算一期小时数（默认 4）
		DispatchPeriodH *int `json:"dispatch_period_h"`
		// ★ 2026-09-24：出征速度加成（百分比，0 = 无加成，节假日调高让队伍走快点）
		MarchSpeedBonus *float64 `json:"march_speed_bonus"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	lim := model.EzfyCfgLimit{ID: 1, MilitaryMax: 33, ResourceMax: 33, HouseMax: 10, FactoryMax: 0,
		GatherMaxPerOrder: ezfyGatherMaxDefault, MallBuyMax: ezfyMallBuyMaxDef,
		ConquerFeelingsMax: ezfyConquerFeelingsDef, LootFeelings: ezfyLootFeelingsDef,
		OfficerSalaryPerLevel: ezfyOfficerSalaryDef, WoundHealDivisor: ezfyWoundHealDivisorDef,
		WildTroopMult: ezfyWildMultDef,
		RecruitCostOn: ezfyRecruitCostDef, FoodUpkeepOn: ezfyFoodUpkeepDef, MarchOilOn: ezfyMarchOilDef,
		WarRequireOn: ezfyWarRequireDef, MarchCapOn: ezfyMarchCapDef,
		OfficerStarUpOn:   ezfyStarUpDef,
		OfficerStarChance: ezfyStarChanceDef, OfficerStarAttrGain: ezfyStarAttrGainDef,
		OfficerStarMax: ezfyStarMaxDef,
		SpeedTrainRate: 100, WoundHealRate: 100,
		// ★ 2026-09-23 线上「负数兵力」事故：单城兵力上限 + 伤兵存活天数
		TroopMax: ezfyTroopMaxDef, WoundExpireDays: ezfyWoundExpireDaysDef}
	h.DB.First(&lim, 1)
	check := func(v *int, name string) (int, bool) {
		if v == nil {
			return 0, true
		}
		if *v < 0 || *v > 999 {
			resp.ParamError(c, name+" 需要在 0~999 之间")
			return 0, false
		}
		return *v, true
	}
	if v, ok := check(in.MilitaryMax, "军事区上限"); !ok {
		return
	} else if in.MilitaryMax != nil {
		lim.MilitaryMax = v
	}
	if v, ok := check(in.ResourceMax, "资源区上限"); !ok {
		return
	} else if in.ResourceMax != nil {
		lim.ResourceMax = v
	}
	if v, ok := check(in.HouseMax, "民居上限"); !ok {
		return
	} else if in.HouseMax != nil {
		lim.HouseMax = v
	}
	if v, ok := check(in.FactoryMax, "军工厂上限"); !ok {
		return
	} else if in.FactoryMax != nil {
		lim.FactoryMax = v
	}
	if v, ok := check(in.NoticeHomeCount, "首页公告条数"); !ok {
		return
	} else if in.NoticeHomeCount != nil {
		lim.NoticeHomeCount = v
	}
	// ★ 集结令单次使用上限：用户要求「设置的时候不要加上限，我设置多少都可以，默认 50」。
	//   只校验 > 0（0 等于把道具禁用，真要禁用请把道具下架），不再限制上界。
	if in.GatherMaxPerOrder != nil {
		if *in.GatherMaxPerOrder < 1 {
			resp.ParamError(c, "集结令单次上限至少为 1")
			return
		}
		lim.GatherMaxPerOrder = *in.GatherMaxPerOrder
	}
	// ★ 商城单次购买上限：用户要求「原来卡控 1-99，改成可配置的，默认 1-9999」。
	//   下限恒为 1（0 = 谁都买不了，无意义），上限给个防呆值 999999，避免误填天文数字。
	//   注意**不能**用上面的 check()——那个把上界卡在 999，装不下 9999 这个默认值。
	if in.MallBuyMax != nil {
		if *in.MallBuyMax < 1 || *in.MallBuyMax > 999999 {
			resp.ParamError(c, "商城单次购买上限需要在 1~999999 之间")
			return
		}
		lim.MallBuyMax = *in.MallBuyMax
	}
	if lim.MilitaryMax <= 0 {
		lim.MilitaryMax = 33
	}
	if lim.ResourceMax <= 0 {
		lim.ResourceMax = 33
	}
	if lim.HouseMax <= 0 {
		lim.HouseMax = 10
	}
	// ★ 首页公告条数允许 0（= 首页不展示公告），但不允许负数；未配过时默认 1
	if lim.NoticeHomeCount < 0 {
		lim.NoticeHomeCount = 1
	}
	// ★ 集结令上限兜底：老数据可能是 0（该列刚加），保存时归一化到默认 50
	if lim.GatherMaxPerOrder <= 0 {
		lim.GatherMaxPerOrder = ezfyGatherMaxDefault
	}
	// ★ 商城单次购买上限兜底：老数据可能是 0（该列刚加），归一化到默认 9999
	if lim.MallBuyMax <= 0 {
		lim.MallBuyMax = ezfyMallBuyMaxDef
	}
	// ★ 战斗/经济数值（用户要求「民心扣除后台可配置，默认 2」+「军官工资合理消耗」）
	//   这几个值 0 无意义，所以只接受 >= 1。
	setPos := func(v *int, dst *int, name string) bool {
		if v == nil {
			return true
		}
		if *v < 1 {
			resp.ParamError(c, name+"至少为 1")
			return false
		}
		*dst = *v
		return true
	}
	if !setPos(in.ConquerFeelingsMax, &lim.ConquerFeelingsMax, "征服单次扣民心") {
		return
	}
	if !setPos(in.LootFeelings, &lim.LootFeelings, "掠夺单次扣民心") {
		return
	}
	if !setPos(in.OfficerSalaryPerLevel, &lim.OfficerSalaryPerLevel, "军官工资系数") {
		return
	}
	if !setPos(in.WoundHealDivisor, &lim.WoundHealDivisor, "伤兵恢复系数") {
		return
	}
	// ★ 野地兵力倍数：允许小数（0.5 = 减半 / 2 = 翻倍），0 及负数无意义。
	//   ★ 用户要求「野地兵力倍数没有上限，现在是 100」→ **去掉上界**，填多少就是多少
	//   （与「出征集结令单次上限」同一套处理：只挡 <= 0）。
	if in.WildTroopMult != nil {
		m := *in.WildTroopMult
		if m <= 0 {
			resp.ParamError(c, "野地兵力倍数必须大于 0")
			return
		}
		lim.WildTroopMult = m
	}
	// ★ 三个玩法开关：0 = 关 / 1 = 开，两个值都合法，**不做** <=0 兜底（0 就是关）。
	setSwitch := func(v *int, dst *int, name string) bool {
		if v == nil {
			return true
		}
		if *v != 0 && *v != 1 {
			resp.ParamError(c, name+"只能是 0(关) 或 1(开)")
			return false
		}
		*dst = *v
		return true
	}
	if !setSwitch(in.RecruitCostOn, &lim.RecruitCostOn, "征兵资源消耗") {
		return
	}
	if !setSwitch(in.FoodUpkeepOn, &lim.FoodUpkeepOn, "军队耗粮") {
		return
	}
	if !setSwitch(in.MarchOilOn, &lim.MarchOilOn, "出征油耗") {
		return
	}
	if !setSwitch(in.WarRequireOn, &lim.WarRequireOn, "宣战功能") {
		return
	}
	if !setSwitch(in.MarchCapOn, &lim.MarchCapOn, "出征上限") {
		return
	}
	// ★ 军官升星的功能开关（0/1 都合法）
	if !setSwitch(in.OfficerStarUpOn, &lim.OfficerStarUpOn, "军官升星功能") {
		return
	}
	// ★ 军官升星的数值项：0 无意义，只接受 >= 1；成功率与上限不超过 100
	if v, ok := check(in.OfficerStarChance, "升星成功率"); !ok {
		return
	} else if in.OfficerStarChance != nil {
		if v < 1 || v > 100 {
			resp.ParamError(c, "升星成功率需要在 1~100 之间")
			return
		}
		lim.OfficerStarChance = v
	}
	if v, ok := check(in.OfficerStarAttrGain, "升星每星加点"); !ok {
		return
	} else if in.OfficerStarAttrGain != nil {
		if v < 1 {
			resp.ParamError(c, "升星每星加点至少为 1")
			return
		}
		lim.OfficerStarAttrGain = v
	}
	if v, ok := check(in.OfficerStarMax, "军官星级上限"); !ok {
		return
	} else if in.OfficerStarMax != nil {
		if v < 1 || v > 100 {
			resp.ParamError(c, "军官星级上限需要在 1~100 之间")
			return
		}
		lim.OfficerStarMax = v
	}
	// ★ 训练加速黄金倍率：允许小数（0.5 = 半价），0 及负数无意义；上界 100 防呆
	if in.SpeedTrainRate != nil {
		m := *in.SpeedTrainRate
		if m <= 0 || m > 100 {
			resp.ParamError(c, "训练加速黄金倍率需要在 0~100 之间（100 = 原价）")
			return
		}
		lim.SpeedTrainRate = m
	}
	// ★ 伤兵恢复黄金折扣率：与训练加速倍率同规则
	if in.WoundHealRate != nil {
		m := *in.WoundHealRate
		if m <= 0 || m > 100 {
			resp.ParamError(c, "伤兵恢复黄金折扣率需要在 0~100 之间（100 = 原价）")
			return
		}
		lim.WoundHealRate = m
	}
	// ★ 2026-09-23 线上「负数兵力」事故：
	//   单城兵力上限（至少 1，0 等于把训练全禁了，无意义）+ 伤兵存活天数（1~3650 天）。
	//   ⚠️ 不能用上面的 check() —— 那个把上界卡在 999，装不下「10 亿」这个默认值。
	if in.TroopMax != nil {
		if *in.TroopMax < 1 {
			resp.ParamError(c, "单城兵力上限至少为 1")
			return
		}
		lim.TroopMax = *in.TroopMax
	}
	if in.WoundExpireDays != nil {
		if *in.WoundExpireDays < 1 || *in.WoundExpireDays > 3650 {
			resp.ParamError(c, "伤兵存活天数需要在 1~3650 之间")
			return
		}
		lim.WoundExpireDays = *in.WoundExpireDays
	}
	// ★ 2026-09-24：采集周期小时数（至少 1 小时；上限 720 防呆 = 30 天）
	if in.DispatchPeriodH != nil {
		if *in.DispatchPeriodH < 1 || *in.DispatchPeriodH > 720 {
			resp.ParamError(c, "采集周期需要在 1~720 小时之间")
			return
		}
		lim.DispatchPeriodH = *in.DispatchPeriodH
	}
	// ★ 2026-09-24：出征速度加成（0 = 无加成，是合法值；上限 1000% 防呆）
	if in.MarchSpeedBonus != nil {
		m := *in.MarchSpeedBonus
		if m < 0 || m > 1000 {
			resp.ParamError(c, "出征速度加成需要在 0~1000 之间（0 = 无加成）")
			return
		}
		lim.MarchSpeedBonus = m
	}
	if lim.ConquerFeelingsMax <= 0 {
		lim.ConquerFeelingsMax = ezfyConquerFeelingsDef
	}
	if lim.LootFeelings <= 0 {
		lim.LootFeelings = ezfyLootFeelingsDef
	}
	if lim.OfficerSalaryPerLevel <= 0 {
		lim.OfficerSalaryPerLevel = ezfyOfficerSalaryDef
	}
	if lim.WoundHealDivisor <= 0 {
		lim.WoundHealDivisor = ezfyWoundHealDivisorDef
	}
	// ★ 野地兵力倍数兜底（老行可能是 0 / NULL）
	if lim.WildTroopMult <= 0 {
		lim.WildTroopMult = ezfyWildMultDef
	}
	// ★ 军官升星数值兜底（老行可能是 0 / NULL）；开关不兜底
	if lim.OfficerStarChance <= 0 {
		lim.OfficerStarChance = ezfyStarChanceDef
	}
	if lim.OfficerStarAttrGain <= 0 {
		lim.OfficerStarAttrGain = ezfyStarAttrGainDef
	}
	if lim.OfficerStarMax <= 0 {
		lim.OfficerStarMax = ezfyStarMaxDef
	}
	// ★ 训练加速黄金倍率兜底（老行可能是 0 / NULL）
	if lim.SpeedTrainRate <= 0 {
		lim.SpeedTrainRate = 100
	}
	// ★ 伤兵恢复黄金折扣率兜底（老行可能是 0 / NULL）
	if lim.WoundHealRate <= 0 {
		lim.WoundHealRate = 100
	}
	// ★ 2026-09-23：单城兵力上限 / 伤兵存活天数（0 无意义 → 回落默认值）
	if lim.TroopMax <= 0 {
		lim.TroopMax = ezfyTroopMaxDef
	}
	if lim.WoundExpireDays <= 0 {
		lim.WoundExpireDays = ezfyWoundExpireDaysDef
	}
	// ★ 2026-09-24：采集周期兜底（老行可能是 0 / NULL）
	if lim.DispatchPeriodH <= 0 {
		lim.DispatchPeriodH = 4
	}
	// ★ 三个开关**不兜底**：0 = 关，是合法值，兜底会把它改回开。
	//   （GORM 的 Save 走 UPDATE 全字段，零值会被写进去；下面 Save 后还会再核一遍。）
	lim.ID = 1
	if err := h.DB.Save(&lim).Error; err != nil {
		resp.ParamError(c, "保存失败："+err.Error())
		return
	}
	// ★ 开关再显式写一次：GORM 对「带 default 标签的零值字段」在部分路径会跳过写入，
	//   用 map 形式的 Updates 兜底，保证「关」一定能落库（这是本页最容易出的坑）。
	h.DB.Model(&model.EzfyCfgLimit{}).Where("id = 1").Updates(map[string]interface{}{
		"recruit_cost_on": lim.RecruitCostOn,
		"food_upkeep_on":  lim.FoodUpkeepOn,
		"march_oil_on":    lim.MarchOilOn,
		"war_require_on":  lim.WarRequireOn,
		"march_cap_on":    lim.MarchCapOn,
		"wild_troop_mult": lim.WildTroopMult,
		// ★ 军官升星功能开关同样要显式写（0 = 关 必须落库）
		"officer_star_up_on": lim.OfficerStarUpOn,
		// ★ 训练加速黄金倍率 / 伤兵恢复黄金折扣率同样用 map 显式写
		"speed_train_rate": lim.SpeedTrainRate,
		"wound_heal_rate":  lim.WoundHealRate,
		// ★ 2026-09-23：兵力上限（bigint）/ 伤兵存活天数，同样用 map 显式写，避开零值被吞的坑
		"troop_max":         lim.TroopMax,
		"wound_expire_days": lim.WoundExpireDays,
		// ★ 2026-09-24：采集周期小时数，同样用 map 显式写
		"dispatch_period_h": lim.DispatchPeriodH,
	})
	// ★ 写完必须重载配置缓存，否则玩家端要重启才生效
	h.ezfyH().cfgsReload()
	resp.OK(c, gin.H{"msg": "系统配置已保存并立即生效", "limit": lim})
}

// ============ 2. 钻石发放 ============

// AdminEzfyDiamondGrant POST /admin/ezfy-players/:id/diamond
//
// ★ 2026-09-23 用户要求：钻石字段在管理端「发放资源」处应叫**发放**而非「充值」。
// mode = add(默认，可负数扣减) | set(直接设为某值)
func (h *AdminHandler) AdminEzfyDiamondGrant(c *gin.Context) {
	uid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || uid == 0 {
		resp.ParamError(c, "玩家ID错误")
		return
	}
	var in struct {
		Amount int64  `json:"amount"`
		Mode   string `json:"mode"`
		Remark string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	ez := h.ezfyH()
	prof := ez.ensureProfile(uint(uid))
	if in.Mode == "set" {
		if in.Amount < 0 {
			resp.ParamError(c, "钻石不能为负数")
			return
		}
		prof.Diamond = in.Amount
	} else {
		if in.Amount == 0 {
			resp.ParamError(c, "请填写发放数量")
			return
		}
		prof.Diamond = prof.Diamond + in.Amount
		if prof.Diamond < 0 {
			prof.Diamond = 0
		}
	}
	if err := h.DB.Model(&model.EzfyProfile{}).Where("id = ?", prof.ID).
		Update("diamond", prof.Diamond).Error; err != nil {
		resp.ParamError(c, "发放失败："+err.Error())
		return
	}
	note := fmt.Sprintf("管理员为你发放钻石 %+d，当前余额 %d", in.Amount, prof.Diamond)
	if in.Mode == "set" {
		note = fmt.Sprintf("管理员将你的钻石余额设为 %d", prof.Diamond)
	}
	if strings.TrimSpace(in.Remark) != "" {
		note += "（" + strings.TrimSpace(in.Remark) + "）"
	}
	h.DB.Create(&model.EzfyNotice{UserId: uint(uid), Title: "钻石发放", Content: note})
	resp.OK(c, gin.H{"msg": note, "diamond": prof.Diamond})
}

// ============ 3. 二战聊天敏感词（独立维护页） ============

// AdminEzfyWordFilters GET /admin/ezfy-word-filters
func (h *AdminHandler) AdminEzfyWordFilters(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	word := strings.TrimSpace(c.Query("word"))
	q := h.DB.Model(&model.EzfyWordFilter{})
	if word != "" {
		q = q.Where("word LIKE ?", "%"+word+"%")
	}
	var total int64
	q.Count(&total)
	var rows []model.EzfyWordFilter
	lq := h.DB.Model(&model.EzfyWordFilter{})
	if word != "" {
		lq = lq.Where("word LIKE ?", "%"+word+"%")
	}
	lq.Order("id DESC").Offset(offset).Limit(size).Find(&rows)
	resp.OK(c, gin.H{"list": rows, "total": total, "page": page, "size": size})
}

// AdminEzfyWordFilterCreate POST /admin/ezfy-word-filters
func (h *AdminHandler) AdminEzfyWordFilterCreate(c *gin.Context) {
	var in struct {
		Word    string `json:"word"`
		Replace string `json:"replace"`
		Type    int    `json:"type"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	w := strings.TrimSpace(in.Word)
	if w == "" {
		resp.ParamError(c, "敏感词不能为空")
		return
	}
	if len([]rune(w)) > 50 {
		resp.ParamError(c, "敏感词最长50字")
		return
	}
	if in.Type != 2 {
		in.Type = 1
	}
	row := model.EzfyWordFilter{Word: w, Replace: strings.TrimSpace(in.Replace), Type: in.Type}
	if err := h.DB.Create(&row).Error; err != nil {
		resp.ParamError(c, "新增失败（该词可能已存在）："+err.Error())
		return
	}
	h.ezfyH().cfgsReload()
	resp.OK(c, gin.H{"msg": "已添加敏感词「" + w + "」", "id": row.ID})
}

// AdminEzfyWordFilterUpdate PUT /admin/ezfy-word-filters/:id
func (h *AdminHandler) AdminEzfyWordFilterUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var row model.EzfyWordFilter
	if err := h.DB.First(&row, id).Error; err != nil {
		resp.NotFound(c, "敏感词不存在")
		return
	}
	var in struct {
		Word    *string `json:"word"`
		Replace *string `json:"replace"`
		Type    *int    `json:"type"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if in.Word != nil {
		w := strings.TrimSpace(*in.Word)
		if w == "" {
			resp.ParamError(c, "敏感词不能为空")
			return
		}
		row.Word = w
	}
	if in.Replace != nil {
		row.Replace = strings.TrimSpace(*in.Replace)
	}
	if in.Type != nil {
		if *in.Type == 2 {
			row.Type = 2
		} else {
			row.Type = 1
		}
	}
	if err := h.DB.Save(&row).Error; err != nil {
		resp.ParamError(c, "保存失败："+err.Error())
		return
	}
	h.ezfyH().cfgsReload()
	resp.OK(c, gin.H{"msg": "已保存"})
}

// AdminEzfyWordFilterDelete DELETE /admin/ezfy-word-filters/:id
func (h *AdminHandler) AdminEzfyWordFilterDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.DB.Delete(&model.EzfyWordFilter{}, id).Error; err != nil {
		resp.ParamError(c, "删除失败："+err.Error())
		return
	}
	h.ezfyH().cfgsReload()
	resp.OK(c, gin.H{"msg": "已删除"})
}

// AdminEzfyWordFilterBulk POST /admin/ezfy-word-filters/bulk
//
// 批量导入：每行一个，格式 `词[,替换词[,类型]]`，类型 1=替换(默认) 2=拦截，# 开头忽略。
func (h *AdminHandler) AdminEzfyWordFilterBulk(c *gin.Context) {
	var in struct {
		Text string `json:"text"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	added, skipped := 0, 0
	for _, line := range strings.Split(in.Text, "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, ",")
		word := strings.TrimSpace(parts[0])
		if word == "" {
			continue
		}
		rep := ""
		typ := 1
		if len(parts) > 1 {
			rep = strings.TrimSpace(parts[1])
		}
		if len(parts) > 2 {
			if v, err := strconv.Atoi(strings.TrimSpace(parts[2])); err == nil && v == 2 {
				typ = 2
			}
		}
		row := model.EzfyWordFilter{Word: word, Replace: rep, Type: typ}
		if err := h.DB.Create(&row).Error; err != nil {
			skipped++
			continue
		}
		added++
	}
	h.ezfyH().cfgsReload()
	resp.OK(c, gin.H{"msg": fmt.Sprintf("导入完成：新增 %d 条，跳过（已存在/无效）%d 条", added, skipped),
		"added": added, "skipped": skipped})
}
