package ezfy

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
//	1. 建筑数量上限配置（军事区/资源区各 36，管理端可维护）
//	2. 钻石发放（钻石只能管理端发放，玩家端只读余额）
//	3. 二战聊天敏感词（独立维护页，与社区「黑名单榜」分开）

// ============ 1. 建筑数量上限配置 ============

// AdminEzfyBuildLimitGet GET /admin/ezfy-build-limit
//
// 管理端「系统配置」页：建筑上限 + 各项数值 + 玩法开关，一次全量返回。
func (h *EzfyAdmin) AdminEzfyBuildLimitGet(c *gin.Context) {
	lim := model.EzfyCfgLimit{ID: 1, MilitaryMax: 36, ResourceMax: 36, HouseMax: 33, FactoryMax: 20,
		GatherMaxPerOrder: ezfyGatherMaxDefault, MallBuyMax: ezfyMallBuyMaxDef,
		// ★ 2026-09-26：召集消耗粮食 / 获得人口（缺行时给默认 10 万）
		ConveneFoodCost: ezfyConveneFoodCostDef, ConvenePopGain: ezfyConvenePopGainDef,
		// ★ 2026-09-26：召集硬性人口上限（缺行时默认 0 = 不限）
		ConvenePopMax:      ezfyConvenePopMaxDef,
		ConquerFeelingsMax: ezfyConquerFeelingsDef, LootFeelings: ezfyLootFeelingsDef,
		// ★ 2026-09-28 安抚参数（默认 5万黄金 / 民怨-2 / 民心+1 / 15 分钟冷却）
		PlacateGold: ezfyPlacateGoldDef, PlacateGrievance: ezfyPlacateGrievanceDef,
		PlacateFeelings: ezfyPlacateFeelingsDef, PlacateCooldownMin: ezfyPlacateCooldownDef,
		OfficerSalaryPerLevel: ezfyOfficerSalaryDef, WoundHealDivisor: ezfyWoundHealDivisorDef,
		WildTroopMult: ezfyWildMultDef,
		WildResMult:   ezfyWildResMultDef,
		GatherResMult: ezfyGatherResMultDef,
		// ★ 2026-09-28 采集后勤加成倍率 / 市长加成倍率（默认 1）
		OfficerGatherMult: ezfyOfficerGatherMultDef,
		MayorGainMult:     ezfyMayorGainMultDef,
		GatherLevelPow:    ezfyGatherLevelPowDef,
		GatherSeaMult:     ezfyGatherSeaMultDef,
		RecruitCycleMode:  ezfyRecruitCycleHourlyDef,
		// ★ 2026-09-30 向系统出售资源回收比例（每100单位黄金，默认粮10/钢10/油20/稀25）
		SysSellFood: 10, SysSellSteel: 10, SysSellOil: 20, SysSellRare: 25,
		// ★ 2026-09-26 城市资源产量倍率（默认 1；**0 合法 = 产量归零，故不做 <=0 兜底**）
		ResProdMult: ezfyResProdMultDef,
		// ★ 2026-10-05 黄金产量倍率（与资源倍率拆开，默认 1；0 合法 = 黄金产量归零）
		GoldProdMult: ezfyGoldProdMultDef,
		// ★ 2026-10-05 战斗掉落宝物概率（中级/高级/特殊阈值 + 活动野地掉宝总概率）
		DropT2: ezfyDropT2Def, DropT3: ezfyDropT3Def, DropT4: ezfyDropT4Def,
		DropActPct: ezfyDropActPctDef,
		// ★ 三个开关的默认值都写进初始值：新建行时 GORM 会显式写 1（列上没有 gorm default 标签）
		RecruitCostOn: ezfyRecruitCostDef, FoodUpkeepOn: ezfyFoodUpkeepDef, MarchOilOn: ezfyMarchOilDef,
		WarRequireOn: ezfyWarRequireDef, MarchCapOn: ezfyMarchCapDef,
		// ★ 2026-09-26：民居容量限制 / 召集人口灵活配置（缺行时默认开，见 ezfyHousePopLimitOn）
		HousePopLimitOn: ezfyHousePopLimitDef, ConveneFlexibleOn: ezfyConveneFlexDef,
		// ★ 2026-09-30 招生简章出五星军官概率（默认 1 = 1%，100 = 必出）
		RecruitFiveStarRate: ezfyRecruitFiveStarDef,
		// ★ 军官升星（简化后：功能开关 + 固定成功率 + 每星加成 + 星级上限）
		OfficerStarUpOn:   ezfyStarUpDef,
		OfficerStarChance: ezfyStarChanceDef, OfficerStarAttrGain: ezfyStarAttrGainDef,
		OfficerStarMax: ezfyStarMaxDef,
		// ★ 训练加速黄金倍率 / 伤兵恢复黄金折扣率（百分比口径：100 = 100% = 原价）+ 伤兵恢复黄金折扣率
		SpeedTrainRate: 0.1, WoundHealRate: 100,
		// ★ 2026-09-23 线上「负数兵力」事故：单城兵力上限 + 伤兵存活天数
		TroopMax: ezfyTroopMaxDef, WoundExpireDays: ezfyWoundExpireDaysDef,
		// ★ 2026-10-02 侦察机每架侦查成功率%（默认 20）
		ReconSuccessPct: ezfyReconSuccessPctDef,
		// ★ 2026-09-25 「各项资源有最大的配置，默认 21 亿」
		ResMaxFood: ezfyResMaxDef, ResMaxSteel: ezfyResMaxDef, ResMaxOil: ezfyResMaxDef,
		ResMaxRare: ezfyResMaxDef, ResMaxGold: ezfyResMaxDef}
	if err := h.DB.First(&lim, 1).Error; err != nil {
		h.DB.Create(&lim)
	}
	// ★ 商城单次购买上限兜底（0 无意义 = 禁止购买），默认 99
	if lim.MallBuyMax <= 0 {
		lim.MallBuyMax = ezfyMallBuyMaxDef
	}
	// ★ 2026-09-26 召集消耗粮食 / 获得人口兜底（0 无意义），默认各 10 万
	if lim.ConveneFoodCost <= 0 {
		lim.ConveneFoodCost = ezfyConveneFoodCostDef
	}
	if lim.ConvenePopGain <= 0 {
		lim.ConvenePopGain = ezfyConvenePopGainDef
	}
	// ★ 集结令上限兜底：老行没这列时可能是 0，回落到默认 99（0 无意义 = 禁用道具）
	if lim.GatherMaxPerOrder <= 0 {
		lim.GatherMaxPerOrder = ezfyGatherMaxDefault
	}
	// ★ 2026-09-28 挂单出售单价上限兜底（0 回落默认 100 = 1:100 卡控）
	if lim.SellPriceMax <= 0 {
		lim.SellPriceMax = 100
	}
	// ★ 2026-09-28 军官军事加成兜底（0 回落默认：上限 +2000/点、速度 +0.1%/点）
	if lim.OfficerCapPerMilitary <= 0 {
		lim.OfficerCapPerMilitary = 2000
	}
	if lim.OfficerSpeedPerMilitary <= 0 {
		lim.OfficerSpeedPerMilitary = 0.1
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
	// ★ 野地兵力倍数：0 / 负数无意义 → 回落 10
	if lim.WildTroopMult <= 0 {
		lim.WildTroopMult = ezfyWildMultDef
	}
	// ★ 野地战利品资源倍率：0 / 负数无意义 → 回落 10
	if lim.WildResMult <= 0 {
		lim.WildResMult = ezfyWildResMultDef
	}
	// ★ 采集资源倍率：0 / 负数无意义 → 回落 10
	if lim.GatherResMult <= 0 {
		lim.GatherResMult = ezfyGatherResMultDef
	}
	// ★ 2026-09-28 采集后勤加成倍率：0 / 负数无意义 → 回落 1
	if lim.OfficerGatherMult <= 0 {
		lim.OfficerGatherMult = ezfyOfficerGatherMultDef
	}
	// ★ 2026-09-28 市长加成倍率：**0 合法**（= 关闭），只对负数兜底回落 1
	if lim.MayorGainMult < 0 {
		lim.MayorGainMult = ezfyMayorGainMultDef
	}
	// ★ 2026-09-28 采集等级成长幂次兜底（0 / NULL → 1.3）
	if lim.GatherLevelPow <= 0 {
		lim.GatherLevelPow = ezfyGatherLevelPowDef
	}
	// ★ 2026-09-28 海野采集系数兜底（0 / 负 / NULL → 1.5）
	if lim.GatherSeaMult <= 0 {
		lim.GatherSeaMult = ezfyGatherSeaMultDef
	}
	// ★ 2026-09-28 军校刷新周期兜底（只允许 1=按天 / 2=按小时，其余回落按小时）
	if lim.RecruitCycleMode != 1 && lim.RecruitCycleMode != 2 {
		lim.RecruitCycleMode = ezfyRecruitCycleHourlyDef
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
	// ★ 2026-09-30 招生简章出五星军官概率兜底（0 无意义 → 回落默认 1 = 1%）
	if lim.RecruitFiveStarRate <= 0 {
		lim.RecruitFiveStarRate = ezfyRecruitFiveStarDef
	}
	// ★ 训练加速黄金倍率：0 / 负数无意义 → 回落 0.1（线上现值；100 = 100% = 原价）
	if lim.SpeedTrainRate <= 0 {
		lim.SpeedTrainRate = 0.1
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
	// ★ 2026-09-24：采集周期小时数（0 无意义 → 回落默认 1 小时）
	if lim.DispatchPeriodH <= 0 {
		lim.DispatchPeriodH = 1
	}
	// ★ 2026-09-25：各项资源的「资源最大值」（0 无意义 → 回落默认 21 亿）
	if lim.ResMaxFood <= 0 {
		lim.ResMaxFood = ezfyResMaxDef
	}
	if lim.ResMaxSteel <= 0 {
		lim.ResMaxSteel = ezfyResMaxDef
	}
	if lim.ResMaxOil <= 0 {
		lim.ResMaxOil = ezfyResMaxDef
	}
	if lim.ResMaxRare <= 0 {
		lim.ResMaxRare = ezfyResMaxDef
	}
	if lim.ResMaxGold <= 0 {
		lim.ResMaxGold = ezfyResMaxDef
	}
	// ★ 2026-09-30：向系统出售资源回收比例（0 无意义 → 回落各自默认，粮10/钢10/油20/稀25）
	if lim.SysSellFood <= 0 {
		lim.SysSellFood = 10
	}
	if lim.SysSellSteel <= 0 {
		lim.SysSellSteel = 10
	}
	if lim.SysSellOil <= 0 {
		lim.SysSellOil = 20
	}
	if lim.SysSellRare <= 0 {
		lim.SysSellRare = 25
	}
	// ★ 2026-10-05 战斗掉落宝物概率兜底（0 / NULL → 回落默认 18/4/1 + 85）
	if lim.DropT2 <= 0 {
		lim.DropT2 = ezfyDropT2Def
	}
	if lim.DropT3 <= 0 {
		lim.DropT3 = ezfyDropT3Def
	}
	if lim.DropT4 <= 0 {
		lim.DropT4 = ezfyDropT4Def
	}
	if lim.DropActPct <= 0 {
		lim.DropActPct = ezfyDropActPctDef
	}
	// ★ 三个开关**不做** <= 0 兜底：0 就是「关」，是合法值。
	//   只有 NULL 才是没配过（列是后来补的），seed 启动时已回填 1。
	resp.OK(c, lim)
}

// AdminEzfyBuildLimitUpdate PUT /admin/ezfy-build-limit
//
// 军事区(type 2/3/4) 与 资源区(type 1) 的数量上限**分开**维护，默认各 36。
// factory_max = 0 表示军工厂不限数量（线上现值为 20）。
func (h *EzfyAdmin) AdminEzfyBuildLimitUpdate(c *gin.Context) {
	var in struct {
		MilitaryMax             *int     `json:"military_max"`
		ResourceMax             *int     `json:"resource_max"`
		HouseMax                *int     `json:"house_max"`
		FactoryMax              *int     `json:"factory_max"`
		NoticeHomeCount         *int     `json:"notice_home_count"`
		GatherMaxPerOrder       *int     `json:"gather_max_per_order"`
		SellPriceMax            *int     `json:"sell_price_max"`
		OfficerCapPerMilitary   *int     `json:"officer_cap_per_military"`
		OfficerSpeedPerMilitary *float64 `json:"officer_speed_per_military"`
		MallBuyMax              *int     `json:"mall_buy_max"`
		ConquerFeelingsMax      *int     `json:"conquer_feelings_max"`
		LootFeelings            *int     `json:"loot_feelings"`
		// ★ 2026-09-28 安抚参数（默认 5万黄金 / 民怨-2 / 民心+1 / 15 分钟冷却）
		PlacateGold           *int64 `json:"placate_gold"`
		PlacateGrievance      *int   `json:"placate_grievance"`
		PlacateFeelings       *int   `json:"placate_feelings"`
		PlacateCooldownMin    *int   `json:"placate_cooldown_min"`
		OfficerSalaryPerLevel *int   `json:"officer_salary_per_level"`
		WoundHealDivisor      *int   `json:"wound_heal_divisor"`
		// ★ 系统配置新增：野地兵力倍数 + 四个玩法开关
		WildTroopMult *float64 `json:"wild_troop_mult"`
		// ★ 2026-09-25：野地战利品资源倍率（默认 10，允许小数）
		WildResMult *float64 `json:"wild_res_mult"`
		// ★ 2026-09-25：采集资源倍率（默认 10，允许小数）
		GatherResMult *float64 `json:"gather_res_mult"`
		// ★ 2026-09-28：采集军官后勤属性加成倍率（默认 1）
		OfficerGatherMult *float64 `json:"officer_gather_mult"`
		// ★ 2026-09-28：市长产量加成倍率（默认 1；**0 合法 = 关闭市长加成**）
		MayorGainMult *float64 `json:"mayor_gain_mult"`
		// ★ 2026-09-28：采集等级成长幂次（默认 1.3，越高级采集越多）
		GatherLevelPow *float64 `json:"gather_level_pow"`
		// ★ 2026-09-28：海野采集系数（默认 1.5，1~2；越大海野采集收益越高）
		GatherSeaMult *float64 `json:"gather_sea_mult"`
		// ★ 2026-09-28：军校刷新周期（1=按天 2=按小时，默认按小时）
		RecruitCycleMode *int `json:"recruit_cycle_mode"`
		// ★ 2026-09-26 城市资源产量倍率（默认 1；**0 合法 = 产量归零**）
		ResProdMult *float64 `json:"res_prod_mult"`
		// ★ 2026-10-05 黄金产量倍率（与资源倍率拆开，默认 1；**0 合法 = 黄金产量归零**）
		GoldProdMult *float64 `json:"gold_prod_mult"`
		// ★ 2026-10-05 战斗掉落宝物概率（中级/高级/特殊 roll 阈值 + 活动野地掉宝总概率%）
		DropT2        *int `json:"drop_t2"`
		DropT3        *int `json:"drop_t3"`
		DropT4        *int `json:"drop_t4"`
		DropActPct    *int `json:"drop_act_pct"`
		RecruitCostOn *int `json:"recruit_cost_on"`
		FoodUpkeepOn  *int `json:"food_upkeep_on"`
		MarchOilOn    *int `json:"march_oil_on"`
		WarRequireOn  *int `json:"war_require_on"`
		MarchCapOn    *int `json:"march_cap_on"`
		// ★ 2026-09-26：民居容量限制 / 召集人口灵活配置（0/1 开关）
		HousePopLimitOn   *int `json:"house_pop_limit_on"`
		ConveneFlexibleOn *int `json:"convene_flexible_on"`
		// ★ 2026-09-26：召集消耗粮食 / 召集获得人口（原来写死 10 万）
		ConveneFoodCost *int `json:"convene_food_cost"`
		ConvenePopGain  *int `json:"convene_pop_gain"`
		// ★ 2026-09-26：召集硬性人口上限（0 = 不限，可配置为 0 关闭限制）
		ConvenePopMax *int `json:"convene_pop_max"`
		// ★ 军官升星（简化后：功能开关 + 数值）
		OfficerStarUpOn     *int `json:"officer_star_up_on"`
		OfficerStarChance   *int `json:"officer_star_chance"`
		OfficerStarAttrGain *int `json:"officer_star_attr_gain"`
		OfficerStarMax      *int `json:"officer_star_max"`
		// ★ 2026-09-30 招生简章出五星军官概率（1~100，默认 1 = 1%，100 = 必出）
		RecruitFiveStarRate *int `json:"recruit_five_star_rate"`
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
		// ★ 2026-09-25 「各项资源有最大的配置，默认 21 亿」
		ResMaxFood  *int64 `json:"res_max_food"`
		ResMaxSteel *int64 `json:"res_max_steel"`
		ResMaxOil   *int64 `json:"res_max_oil"`
		ResMaxRare  *int64 `json:"res_max_rare"`
		ResMaxGold  *int64 `json:"res_max_gold"`
		// ★ 2026-09-30：向系统出售资源回收比例（每100单位黄金，默认粮10/钢10/油20/稀25）
		SysSellFood  *int `json:"sys_sell_food"`
		SysSellSteel *int `json:"sys_sell_steel"`
		SysSellOil   *int `json:"sys_sell_oil"`
		SysSellRare  *int `json:"sys_sell_rare"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	lim := model.EzfyCfgLimit{ID: 1, MilitaryMax: 36, ResourceMax: 36, HouseMax: 33, FactoryMax: 20,
		GatherMaxPerOrder: ezfyGatherMaxDefault, MallBuyMax: ezfyMallBuyMaxDef,
		// ★ 2026-09-26：召集消耗粮食 / 获得人口（缺行时给默认 10 万）
		ConveneFoodCost: ezfyConveneFoodCostDef, ConvenePopGain: ezfyConvenePopGainDef,
		// ★ 2026-09-26：召集硬性人口上限（缺行时默认 0 = 不限）
		ConvenePopMax:      ezfyConvenePopMaxDef,
		ConquerFeelingsMax: ezfyConquerFeelingsDef, LootFeelings: ezfyLootFeelingsDef,
		// ★ 2026-09-28 安抚参数（默认 5万黄金 / 民怨-2 / 民心+1 / 15 分钟冷却）
		PlacateGold: ezfyPlacateGoldDef, PlacateGrievance: ezfyPlacateGrievanceDef,
		PlacateFeelings: ezfyPlacateFeelingsDef, PlacateCooldownMin: ezfyPlacateCooldownDef,
		OfficerSalaryPerLevel: ezfyOfficerSalaryDef, WoundHealDivisor: ezfyWoundHealDivisorDef,
		WildTroopMult: ezfyWildMultDef,
		WildResMult:   ezfyWildResMultDef,
		GatherResMult: ezfyGatherResMultDef,
		// ★ 2026-09-28 采集后勤加成倍率 / 市长加成倍率（默认 1）
		OfficerGatherMult: ezfyOfficerGatherMultDef,
		MayorGainMult:     ezfyMayorGainMultDef,
		GatherLevelPow:    ezfyGatherLevelPowDef,
		GatherSeaMult:     ezfyGatherSeaMultDef,
		RecruitCycleMode:  ezfyRecruitCycleHourlyDef,
		// ★ 2026-09-26 城市资源产量倍率（默认 1；**0 合法 = 产量归零，故不做 <=0 兜底**）
		ResProdMult:  ezfyResProdMultDef,
		GoldProdMult: ezfyGoldProdMultDef,
		// ★ 2026-10-05 战斗掉落宝物概率（中级/高级/特殊阈值 + 活动野地掉宝总概率）
		DropT2: ezfyDropT2Def, DropT3: ezfyDropT3Def, DropT4: ezfyDropT4Def,
		DropActPct:    ezfyDropActPctDef,
		RecruitCostOn: ezfyRecruitCostDef, FoodUpkeepOn: ezfyFoodUpkeepDef, MarchOilOn: ezfyMarchOilDef,
		WarRequireOn: ezfyWarRequireDef, MarchCapOn: ezfyMarchCapDef,
		// ★ 2026-09-26：民居容量限制 / 召集人口灵活配置
		HousePopLimitOn: ezfyHousePopLimitDef, ConveneFlexibleOn: ezfyConveneFlexDef,
		OfficerStarUpOn:   ezfyStarUpDef,
		OfficerStarChance: ezfyStarChanceDef, OfficerStarAttrGain: ezfyStarAttrGainDef,
		OfficerStarMax: ezfyStarMaxDef,
		// ★ 2026-09-30 招生简章出五星军官概率（默认 1 = 1%，100 = 必出）
		RecruitFiveStarRate: ezfyRecruitFiveStarDef,
		SpeedTrainRate:      0.1, WoundHealRate: 100,
		// ★ 2026-09-23 线上「负数兵力」事故：单城兵力上限 + 伤兵存活天数
		TroopMax: ezfyTroopMaxDef, WoundExpireDays: ezfyWoundExpireDaysDef,
		// ★ 2026-10-02 侦察机每架侦查成功率%（默认 20）
		ReconSuccessPct: ezfyReconSuccessPctDef,
		// ★ 2026-09-25 各项资源的「资源最大值」（默认 21 亿）
		ResMaxFood: ezfyResMaxDef, ResMaxSteel: ezfyResMaxDef, ResMaxOil: ezfyResMaxDef,
		ResMaxRare: ezfyResMaxDef, ResMaxGold: ezfyResMaxDef,
		// ★ 2026-09-30 向系统出售资源回收比例（每100单位黄金）
		SysSellFood: 10, SysSellSteel: 10, SysSellOil: 20, SysSellRare: 25}
	h.DB.First(&lim, 1)
	// ★ 2026-09-30 向系统出售资源回收比例兜底（0 无意义 → 回落各自默认）
	if lim.SysSellFood <= 0 {
		lim.SysSellFood = 10
	}
	if lim.SysSellSteel <= 0 {
		lim.SysSellSteel = 10
	}
	if lim.SysSellOil <= 0 {
		lim.SysSellOil = 20
	}
	if lim.SysSellRare <= 0 {
		lim.SysSellRare = 25
	}
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
	// ★ 集结令单次使用上限：「设置的时候不要加上限，我设置多少都可以」（线上现值 99）。
	//   只校验 > 0（0 等于把道具禁用，真要禁用请把道具下架），不再限制上界。
	if in.GatherMaxPerOrder != nil {
		if *in.GatherMaxPerOrder < 1 {
			resp.ParamError(c, "集结令单次上限至少为 1")
			return
		}
		lim.GatherMaxPerOrder = *in.GatherMaxPerOrder
	}
	// ★ 商城单次购买上限：可配置，线上现值为 99。
	//   下限恒为 1（0 = 谁都买不了，无意义），上限给个防呆值 999999，避免误填天文数字。
	//   注意**不能**用上面的 check()——那个把上界卡在 999，装不下 999999 这个防呆值。
	if in.MallBuyMax != nil {
		if *in.MallBuyMax < 1 || *in.MallBuyMax > 999999 {
			resp.ParamError(c, "商城单次购买上限需要在 1~999999 之间")
			return
		}
		lim.MallBuyMax = *in.MallBuyMax
	}
	// ★ 2026-09-26 「花费 10万粮食 召集 10万人口也要能配置」：
	//   两个值 0 都无意义（召集不要钱 / 召集不给人口），所以只接受 >= 1；
	//   上界卡在 10 亿（MySQL int 上限约 21 亿），避免误填天文数字把列写溢出。
	if in.ConveneFoodCost != nil {
		if *in.ConveneFoodCost < 1 || *in.ConveneFoodCost > 1000000000 {
			resp.ParamError(c, "召集消耗粮食需要在 1~1000000000 之间")
			return
		}
		lim.ConveneFoodCost = *in.ConveneFoodCost
	}
	if in.ConvenePopGain != nil {
		if *in.ConvenePopGain < 1 || *in.ConvenePopGain > 1000000000 {
			resp.ParamError(c, "召集获得人口需要在 1~1000000000 之间")
			return
		}
		lim.ConvenePopGain = *in.ConvenePopGain
	}
	// ★ 2026-09-26 「玩家城市人口不能超过配置的人口上限，超过则禁止召集」：
	//   全局硬性人口上限。0 = 不限（关闭限制），1~10 亿为有效封顶值。
	if in.ConvenePopMax != nil {
		if *in.ConvenePopMax < 0 || *in.ConvenePopMax > 1000000000 {
			resp.ParamError(c, "人口上限需要在 0~1000000000 之间（0 = 不限）")
			return
		}
		lim.ConvenePopMax = *in.ConvenePopMax
	}
	if lim.MilitaryMax <= 0 {
		lim.MilitaryMax = 36
	}
	if lim.ResourceMax <= 0 {
		lim.ResourceMax = 36
	}
	if lim.HouseMax <= 0 {
		lim.HouseMax = 33
	}
	// ★ 首页公告条数允许 0（= 首页不展示公告），但不允许负数；未配过时默认 1
	if lim.NoticeHomeCount < 0 {
		lim.NoticeHomeCount = 1
	}
	// ★ 集结令上限兜底：老数据可能是 0（该列刚加），保存时归一化到默认 99
	if lim.GatherMaxPerOrder <= 0 {
		lim.GatherMaxPerOrder = ezfyGatherMaxDefault
	}
	// ★ 商城单次购买上限兜底：老数据可能是 0（该列刚加），归一化到默认 99
	if lim.MallBuyMax <= 0 {
		lim.MallBuyMax = ezfyMallBuyMaxDef
	}
	// ★ 2026-09-26 召集消耗粮食 / 获得人口兜底：老数据/新列可能是 0，归一化到默认 10 万
	if lim.ConveneFoodCost <= 0 {
		lim.ConveneFoodCost = ezfyConveneFoodCostDef
	}
	if lim.ConvenePopGain <= 0 {
		lim.ConvenePopGain = ezfyConvenePopGainDef
	}
	// ★ 战斗/经济数值（「民心扣除后台可配置，默认 2」+「军官工资合理消耗」）
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
	// ★ 2026-09-28 安抚参数：四个都「0 无意义」→ 只接受 >= 1。
	//   placate_gold 是 int64，单独判（setPos 只处理 int）。
	if in.PlacateGold != nil {
		if *in.PlacateGold < 1 {
			resp.ParamError(c, "安抚花费黄金至少为 1")
			return
		}
		lim.PlacateGold = *in.PlacateGold
	}
	if !setPos(in.PlacateGrievance, &lim.PlacateGrievance, "安抚降低民怨") {
		return
	}
	if !setPos(in.PlacateFeelings, &lim.PlacateFeelings, "安抚提升民心") {
		return
	}
	if !setPos(in.PlacateCooldownMin, &lim.PlacateCooldownMin, "安抚冷却分钟") {
		return
	}
	// ★ 野地兵力倍数：允许小数（0.5 = 减半 / 2 = 翻倍），0 及负数无意义。
	//   ★ 「野地兵力倍数没有上限，现在是 100」→ **去掉上界**，填多少就是多少
	//   （与「出征集结令单次上限」同一套处理：只挡 <= 0）。
	if in.WildTroopMult != nil {
		m := *in.WildTroopMult
		if m <= 0 {
			resp.ParamError(c, "野地兵力倍数必须大于 0")
			return
		}
		lim.WildTroopMult = m
	}
	// ★ 2026-09-25 野地战利品资源倍率：同样允许小数（0.5 = 减半 / 2 = 翻倍），0 及负数无意义。
	//   「野地打完资源太少」→ 上不封顶，填多少就是多少（与野地兵力倍数同一套）。
	if in.WildResMult != nil {
		m := *in.WildResMult
		if m <= 0 {
			resp.ParamError(c, "野地获取资源倍率必须大于 0")
			return
		}
		lim.WildResMult = m
	}
	// ★ 2026-09-25 采集资源倍率：同样允许小数（0.5 = 减半 / 2 = 翻倍），0 及负数无意义。
	if in.GatherResMult != nil {
		m := *in.GatherResMult
		if m <= 0 {
			resp.ParamError(c, "采集资源倍率必须大于 0")
			return
		}
		lim.GatherResMult = m
	}
	// ★ 2026-09-28 采集后勤加成倍率：0 及负数无意义（= 无加成），仅接受正数。
	if in.OfficerGatherMult != nil {
		m := *in.OfficerGatherMult
		if m <= 0 {
			resp.ParamError(c, "采集后勤加成倍率必须大于 0")
			return
		}
		lim.OfficerGatherMult = m
	}
	// ★ 2026-09-28 市长加成倍率：**0 合法**（= 关闭市长加成），只拦负数。
	if in.MayorGainMult != nil {
		m := *in.MayorGainMult
		if m < 0 {
			resp.ParamError(c, "市长加成倍率不能为负数（0 表示关闭市长加成）")
			return
		}
		lim.MayorGainMult = m
	}
	// ★ 2026-09-28 采集等级成长幂次：0 及负数无意义（= 采集归零荒谬），仅接受正数。
	if in.GatherLevelPow != nil {
		m := *in.GatherLevelPow
		if m <= 0 {
			resp.ParamError(c, "采集等级成长幂次必须大于 0")
			return
		}
		lim.GatherLevelPow = m
	}
	// ★ 2026-09-28 海野采集系数：允许小数，1~2 建议区间；只拦非正数
	if in.GatherSeaMult != nil {
		m := *in.GatherSeaMult
		if m <= 0 {
			resp.ParamError(c, "海野采集系数必须大于 0（建议 1~2）")
			return
		}
		lim.GatherSeaMult = m
	}
	// ★ 2026-10-05 战斗掉落宝物概率：都是百分比 1~100（可填 100 = 必掉），0/负 拒绝
	for _, dp := range []struct {
		in   *int
		dst  *int
		name string
	}{
		{in.DropT2, &lim.DropT2, "中级宝物掉落概率"},
		{in.DropT3, &lim.DropT3, "高级宝物掉落概率"},
		{in.DropT4, &lim.DropT4, "特殊宝物掉落概率"},
		{in.DropActPct, &lim.DropActPct, "活动野地掉宝概率"},
	} {
		if dp.in == nil {
			continue
		}
		if *dp.in <= 0 || *dp.in > 100 {
			resp.ParamError(c, dp.name+"必须在 1~100 之间（% ）")
			return
		}
		*dp.dst = *dp.in
	}
	// ★ 2026-09-28 军校刷新周期：只允许 1=按天 2=按小时
	if in.RecruitCycleMode != nil {
		m := *in.RecruitCycleMode
		if m != 1 && m != 2 {
			resp.ParamError(c, "军校刷新周期只能为 1(按天) 或 2(按小时)")
			return
		}
		lim.RecruitCycleMode = m
	}
	// ★ 2026-09-26 城市资源产量倍率：允许小数，**且 0 合法**（= 产量归零）。
	//   用户原话：「默认 1，可以调整 >= 0 的任意数量」—— 所以只拦负数。
	if in.ResProdMult != nil {
		m := *in.ResProdMult
		if m < 0 {
			resp.ParamError(c, "产量加成倍率不能为负数（0 表示产量归零）")
			return
		}
		lim.ResProdMult = m
	}
	// ★ 2026-10-05 黄金产量倍率（与资源倍率拆开）：同样只拦负数，0 合法 = 黄金产量归零
	if in.GoldProdMult != nil {
		m := *in.GoldProdMult
		if m < 0 {
			resp.ParamError(c, "黄金产量加成倍率不能为负数（0 表示黄金产量归零）")
			return
		}
		lim.GoldProdMult = m
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
	// ★ 2026-09-26：民居容量限制 / 召集人口灵活配置（0/1 都合法）
	if !setSwitch(in.HousePopLimitOn, &lim.HousePopLimitOn, "民居容量限制") {
		return
	}
	if !setSwitch(in.ConveneFlexibleOn, &lim.ConveneFlexibleOn, "召集人口灵活配置") {
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
	// ★ 2026-09-30 招生简章出五星军官概率：1~100（0 无意义；100 = 必出 5 星）
	if in.RecruitFiveStarRate != nil {
		if *in.RecruitFiveStarRate < 1 || *in.RecruitFiveStarRate > 100 {
			resp.ParamError(c, "招生简章出五星军官概率需要在 1~100 之间（100 = 必出 5 星）")
			return
		}
		lim.RecruitFiveStarRate = *in.RecruitFiveStarRate
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
	// ★ 2026-09-28：挂单出售单价上限（黄金/单位，默认 100；0 回落默认，上限 100000 防呆）
	if in.SellPriceMax != nil {
		if *in.SellPriceMax < 0 || *in.SellPriceMax > 100000 {
			resp.ParamError(c, "挂单出售单价上限需要在 0~100000 之间")
			return
		}
		lim.SellPriceMax = *in.SellPriceMax
	}
	// ★ 2026-09-30：向系统出售资源回收比例（每100单位黄金，上限 100000 防呆）
	sysSellSet := func(v *int, dst *int, name string) bool {
		if v == nil {
			return true
		}
		if *v < 0 || *v > 100000 {
			resp.ParamError(c, name+"需要在 0~100000 之间")
			return false
		}
		*dst = *v
		return true
	}
	if !sysSellSet(in.SysSellFood, &lim.SysSellFood, "粮食回收比例") {
		return
	}
	if !sysSellSet(in.SysSellSteel, &lim.SysSellSteel, "钢铁回收比例") {
		return
	}
	if !sysSellSet(in.SysSellOil, &lim.SysSellOil, "石油回收比例") {
		return
	}
	if !sysSellSet(in.SysSellRare, &lim.SysSellRare, "稀矿回收比例") {
		return
	}
	// ★ 2026-09-28：军官军事每点累加的出征上限（默认 2000）/ 每点速度加成（默认 0.1，单位 %）
	if in.OfficerCapPerMilitary != nil {
		if *in.OfficerCapPerMilitary < 0 || *in.OfficerCapPerMilitary > 100000 {
			resp.ParamError(c, "军官军事每点出征上限需要在 0~100000 之间")
			return
		}
		lim.OfficerCapPerMilitary = *in.OfficerCapPerMilitary
	}
	if in.OfficerSpeedPerMilitary != nil {
		if *in.OfficerSpeedPerMilitary < 0 || *in.OfficerSpeedPerMilitary > 10 {
			resp.ParamError(c, "军官军事每点速度加成需要在 0~10 之间（0.1 = 每点 +0.1%）")
			return
		}
		lim.OfficerSpeedPerMilitary = *in.OfficerSpeedPerMilitary
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
	// ★ 2026-09-28 安抚参数兜底（老行可能是 0 / NULL）
	if lim.PlacateGold <= 0 {
		lim.PlacateGold = ezfyPlacateGoldDef
	}
	if lim.PlacateGrievance <= 0 {
		lim.PlacateGrievance = ezfyPlacateGrievanceDef
	}
	if lim.PlacateFeelings <= 0 {
		lim.PlacateFeelings = ezfyPlacateFeelingsDef
	}
	if lim.PlacateCooldownMin <= 0 {
		lim.PlacateCooldownMin = ezfyPlacateCooldownDef
	}
	// ★ 野地兵力倍数兜底（老行可能是 0 / NULL）
	if lim.WildTroopMult <= 0 {
		lim.WildTroopMult = ezfyWildMultDef
	}
	// ★ 野地战利品资源倍率兜底（老行可能是 0 / NULL）
	if lim.WildResMult <= 0 {
		lim.WildResMult = ezfyWildResMultDef
	}
	// ★ 采集资源倍率兜底（老行可能是 0 / NULL）
	if lim.GatherResMult <= 0 {
		lim.GatherResMult = ezfyGatherResMultDef
	}
	// ★ 2026-09-28 采集后勤加成倍率兜底（老行 0 / NULL → 1）
	if lim.OfficerGatherMult <= 0 {
		lim.OfficerGatherMult = ezfyOfficerGatherMultDef
	}
	// ★ 2026-09-28 市长加成倍率兜底（老行负数 → 1；**0 合法**不兜底）
	if lim.MayorGainMult < 0 {
		lim.MayorGainMult = ezfyMayorGainMultDef
	}
	// ★ 2026-09-28 采集等级成长幂次兜底（0 / NULL → 1.3）
	if lim.GatherLevelPow <= 0 {
		lim.GatherLevelPow = ezfyGatherLevelPowDef
	}
	// ★ 2026-09-28 海野采集系数兜底（0 / 负 / NULL → 1.5）
	if lim.GatherSeaMult <= 0 {
		lim.GatherSeaMult = ezfyGatherSeaMultDef
	}
	// ★ 2026-10-05 战斗掉落宝物概率兜底（老行 0 / NULL → 回落默认 18/4/1 + 85）
	if lim.DropT2 <= 0 {
		lim.DropT2 = ezfyDropT2Def
	}
	if lim.DropT3 <= 0 {
		lim.DropT3 = ezfyDropT3Def
	}
	if lim.DropT4 <= 0 {
		lim.DropT4 = ezfyDropT4Def
	}
	if lim.DropActPct <= 0 {
		lim.DropActPct = ezfyDropActPctDef
	}
	// ★ 2026-09-28 军校刷新周期兜底（只允许 1=按天 / 2=按小时，其余回落按小时）
	if lim.RecruitCycleMode != 1 && lim.RecruitCycleMode != 2 {
		lim.RecruitCycleMode = ezfyRecruitCycleHourlyDef
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
	// ★ 2026-09-30 招生简章出五星军官概率兜底（0 无意义 → 回落默认 1 = 1%）
	if lim.RecruitFiveStarRate <= 0 {
		lim.RecruitFiveStarRate = ezfyRecruitFiveStarDef
	}
	// ★ 训练加速黄金倍率兜底（老行可能是 0 / NULL）
	if lim.SpeedTrainRate <= 0 {
		lim.SpeedTrainRate = 0.1
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
		lim.DispatchPeriodH = 1
	}
	// ★ 2026-09-25 各项资源的「资源最大值」：入参 > 0 才覆盖，且不超过数据库安全上限。
	//   （1 ≤ 值 ≤ 1 万亿 = ezfyResSafeMax；不给 0 —— 0 会让玩家的入库累加全部失效。）
	checkResMax := func(v *int64, dst *int64, name string) bool {
		if v == nil {
			return true
		}
		if *v < 1 {
			resp.ParamError(c, name+"必须 ≥ 1（0 会让该资源的入库累加失效）")
			return false
		}
		if *v > ezfyResSafeMax {
			resp.ParamError(c, fmt.Sprintf("%s不能超过 %d（数据库安全上限）", name, ezfyResSafeMax))
			return false
		}
		*dst = *v
		return true
	}
	if !checkResMax(in.ResMaxFood, &lim.ResMaxFood, "粮食最大值") {
		return
	}
	if !checkResMax(in.ResMaxSteel, &lim.ResMaxSteel, "钢铁最大值") {
		return
	}
	if !checkResMax(in.ResMaxOil, &lim.ResMaxOil, "石油最大值") {
		return
	}
	if !checkResMax(in.ResMaxRare, &lim.ResMaxRare, "稀有矿最大值") {
		return
	}
	if !checkResMax(in.ResMaxGold, &lim.ResMaxGold, "黄金最大值") {
		return
	}
	// 兜底：老行 / 被存成 0 时回落默认 21 亿
	if lim.ResMaxFood <= 0 {
		lim.ResMaxFood = ezfyResMaxDef
	}
	if lim.ResMaxSteel <= 0 {
		lim.ResMaxSteel = ezfyResMaxDef
	}
	if lim.ResMaxOil <= 0 {
		lim.ResMaxOil = ezfyResMaxDef
	}
	if lim.ResMaxRare <= 0 {
		lim.ResMaxRare = ezfyResMaxDef
	}
	if lim.ResMaxGold <= 0 {
		lim.ResMaxGold = ezfyResMaxDef
	}
	// ★ 三个开关**不兜底**：0 = 关，是合法值，兜底会把它改回开。
	//   （GORM 的 Save 走 UPDATE 全字段，零值会被写进去；下面 Save 后还会再核一遍。）
	// ★ 2026-10-05 防御「Unknown column 'gold_prod_mult'」：seed.skip 的共享库节点
	//   启动不会跑全量 seed，新配置列可能缺失 → 保存前幂等补列（HasColumn 探测，秒回）。
	if !h.DB.Migrator().HasColumn("ezfy_cfg_limit", "gold_prod_mult") {
		h.DB.Exec("ALTER TABLE ezfy_cfg_limit ADD COLUMN gold_prod_mult double DEFAULT 1")
		h.DB.Exec("UPDATE ezfy_cfg_limit SET gold_prod_mult = 1 WHERE gold_prod_mult IS NULL")
	}
	// ★ 2026-10-05 同理防御「Unknown column 'drop_t2' / 'drop_act_pct' 等」
	for _, c := range []struct {
		col string
		def int
	}{
		{"drop_t2", ezfyDropT2Def}, {"drop_t3", ezfyDropT3Def},
		{"drop_t4", ezfyDropT4Def}, {"drop_act_pct", ezfyDropActPctDef},
	} {
		if !h.DB.Migrator().HasColumn("ezfy_cfg_limit", c.col) {
			h.DB.Exec("ALTER TABLE ezfy_cfg_limit ADD COLUMN " + c.col + " int DEFAULT " + strconv.Itoa(c.def))
		}
		h.DB.Exec("UPDATE ezfy_cfg_limit SET " + c.col + " = " + strconv.Itoa(c.def) +
			" WHERE " + c.col + " IS NULL OR " + c.col + " <= 0")
	}
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
		// ★ 2026-09-26：民居容量限制 / 召集人口灵活配置（0 = 关 必须落库）
		"house_pop_limit_on":  lim.HousePopLimitOn,
		"convene_flexible_on": lim.ConveneFlexibleOn,
		"wild_troop_mult":     lim.WildTroopMult,
		"wild_res_mult":       lim.WildResMult,
		"gather_res_mult":     lim.GatherResMult,
		// ★ 2026-09-28 采集后勤加成倍率 / 市长加成倍率
		"officer_gather_mult": lim.OfficerGatherMult,
		"mayor_gain_mult":     lim.MayorGainMult,
		// ★ 2026-09-28 采集等级成长幂次
		"gather_level_pow": lim.GatherLevelPow,
		// ★ 2026-09-28 海野采集系数
		"gather_sea_mult": lim.GatherSeaMult,
		// ★ 2026-09-28 军校刷新周期（1=按天 2=按小时）
		"recruit_cycle_mode": lim.RecruitCycleMode,
		// ★ 2026-09-26 城市资源产量倍率（默认 1，0 = 产量归零）
		"res_prod_mult": lim.ResProdMult,
		// ★ 2026-10-05 黄金产量倍率（与资源倍率拆开，0 = 黄金产量归零）
		"gold_prod_mult": lim.GoldProdMult,
		// ★ 2026-10-05 战斗掉落宝物概率（中级/高级/特殊阈值 + 活动野地掉宝总概率）
		"drop_t2":      lim.DropT2,
		"drop_t3":      lim.DropT3,
		"drop_t4":      lim.DropT4,
		"drop_act_pct": lim.DropActPct,
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
		// ★ 2026-09-25：各项资源的资源最大值（bigint，同样用 map 显式写）
		"res_max_food":  lim.ResMaxFood,
		"res_max_steel": lim.ResMaxSteel,
		"res_max_oil":   lim.ResMaxOil,
		"res_max_rare":  lim.ResMaxRare,
		"res_max_gold":  lim.ResMaxGold,
		// ★ 2026-09-30：向系统出售资源回收比例（每100单位黄金）
		"sys_sell_food":  lim.SysSellFood,
		"sys_sell_steel": lim.SysSellSteel,
		"sys_sell_oil":   lim.SysSellOil,
		"sys_sell_rare":  lim.SysSellRare,
		// ★ 2026-09-26：召集消耗粮食 / 召集获得人口，同样用 map 显式写
		"convene_food_cost": lim.ConveneFoodCost,
		"convene_pop_gain":  lim.ConvenePopGain,
		// ★ 2026-09-26：召集硬性人口上限（0 = 不限，必须显式写否则 0 会被 GORM 吞掉）
		"convene_pop_max": lim.ConvenePopMax,
	})
	// ★ 写完必须重载配置缓存，否则玩家端要重启才生效
	h.ezfyH().cfgsReload()
	resp.OK(c, gin.H{"msg": "系统配置已保存并立即生效", "limit": lim})
}

// ============ 2. 钻石发放 ============

// AdminEzfyDiamondGrant POST /admin/ezfy-players/:id/diamond
//
// ★ 2026-09-23 钻石字段在管理端「发放资源」处应叫**发放**而非「充值」。
// mode = add(默认，可负数扣减) | set(直接设为某值)
func (h *EzfyAdmin) AdminEzfyDiamondGrant(c *gin.Context) {
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
	before := prof.Diamond
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
	// ★ 2026-09-28 钻石流水（管理端调整，set=置为固定值 / add=增减）
	ez.logDiamond(uint(uid), prof.Diamond-before, "管理端调整钻石("+in.Mode+")")
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
func (h *EzfyAdmin) AdminEzfyWordFilters(c *gin.Context) {
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
func (h *EzfyAdmin) AdminEzfyWordFilterCreate(c *gin.Context) {
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
func (h *EzfyAdmin) AdminEzfyWordFilterUpdate(c *gin.Context) {
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
func (h *EzfyAdmin) AdminEzfyWordFilterDelete(c *gin.Context) {
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
func (h *EzfyAdmin) AdminEzfyWordFilterBulk(c *gin.Context) {
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
