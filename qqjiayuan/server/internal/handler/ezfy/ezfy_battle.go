package ezfy

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"strings"
)

// 二战风云 多回合战斗引擎（忠实移植 BattleEngine.java）
// 规则: 回合制, 每回合按速度从高到低行动; 双方相向移动, 进入射程后开火; 一方全灭或回合耗尽结束
//
// ★ 2026-09-22 「实现指挥功能」（入口：军情 → 军队动态 → [指挥]）：
// 引擎从「一次跑完所有回合」拆成**可逐回合推进的状态机**（ezfyBattleState.Step），
// 这样才能支持「每回合 30 秒、前 25 秒下指令、后 5 秒锁定并由服务器结算」的实时指挥。
// ezfySimulate 保留为兼容包装（内部循环 Step），现有调用点零改动。

const (
	// 最大回合数 —— 复刻《战斗机制（家园玩家必看）》§1「战斗最多40回合；达到上限仍未分胜负则按平局处理」
	ezfyBattleMaxRounds = 40
	ezfyBattleStartDist = 6000 // 战场初始距离（攻方 0，守方 6000，相距 6000）
	// ★ 2026-09-23 双方后退不能无限制，最多各退 10000。
	//   战场坐标空间约「10000 / 6000 / 10000」：
	//   攻方起点 0 只能退到 -10000；守方起点 6000 只能退到 16000。超过夹回。
	ezfyBattleRetreatMax = 10000

	// ★ 暴击基础伤害加成%：装备只配了暴击几率、没配暴击伤害（crit_dmg = 0）时的兜底倍率。
	//
	// 2026-09-23 用户反馈「暴击打了 1 个单位」—— 根因就是这类装备：
	// 例「黑色幽灵[徽章]」crit=125 / crit_dmg=0，按老公式 伤害×(100+0)/100 = 伤害不变，
	// 于是「出了【暴击】但一点没多打」，玩家看到的就成了「暴击跟没暴一样」。
	ezfyCritBaseBonusPct = 50

	// ★ 2026-10-02 盟军驻军战「溃败撤退」阈值：守方剩余兵力跌破初始兵力的 50% 即判定战败，
	//   战斗提前结束，剩余部队自动返航回出发城市（不再死战到全灭）。
	//   —— 让「驻军战败 → 剩余部队回到自己城市」有兵可回（引擎里「战败」原本=全灭）。
	//   后续如需可调，可挪进「二战系统配置」。
	ezfyGarrisonBreakPct = 50
)

// 战场指挥指令（玩家每回合可下达）
const (
	ezfyCmdAdvance = "advance" // 前进
	ezfyCmdHold    = "hold"    // 暂停
	ezfyCmdRetreat = "retreat" // 后退
)

type ezfyFightUnit struct {
	id           string
	cfg          *ezfyTroopStats
	count        int64
	initialCount int64
	pos          int
}

type ezfyTroopStats struct {
	ID          int
	Name        string
	Type        int
	Health      int
	AtkSea      int
	AtkGround   int
	AtkAir      int
	Defence     int
	Speed       int
	AttackRange int
}

func ezfyStatsOf(troopId int) *ezfyTroopStats {
	cfg := ezfyCfg.troop(troopId)
	if cfg == nil {
		return nil
	}
	return &ezfyTroopStats{
		ID: cfg.ID, Name: cfg.Name, Type: cfg.Type, Health: cfg.Health,
		AtkSea: cfg.AtkSea, AtkGround: cfg.AtkGround, AtkAir: cfg.AtkAir,
		Defence: cfg.Defence, Speed: cfg.Speed, AttackRange: cfg.AttackRange,
	}
}

func (u *ezfyFightUnit) alive() bool { return u.count > 0 }

type ezfyBattleResult struct {
	AttackerWin    bool
	Draw           bool
	Rounds         int
	Actions        []string
	AttackerLosses []ezfyUnitGroup
	DefenderLosses []ezfyUnitGroup
	AttackerLeft   []ezfyUnitGroup
	DefenderLeft   []ezfyUnitGroup
}

// ============ 战场状态机（实时指挥用） ============

// ezfyBattleState 战场状态：可以**一回合一次**地推进，也可以一次跑完（ezfySimulate）。
//
// 加成字段存的是**不含装备**的基础值，Step 里再叠加装备六项 ——
// 这样快照往返（存库 → 重建）不会把装备加成叠两遍。
type ezfyBattleState struct {
	Attackers []*ezfyFightUnit
	Defenders []*ezfyFightUnit

	AtkBonus int
	DefBonus int
	// ★ 2026-10-06 守方攻击加成（城防/守城部队行动时也吃科技+军官技能加成，
	//   原来守方行动时 unitAtkBonus=0，城防打人完全没加成 —— 用户反馈「科技/技能加成没算入伤害」）
	DefAtkBonus int
	// ★ 2026-10-06 军官「攻击加成」部分（军官技能+属性带来的百分点）：
	//   战报行动日志用它拆解「攻击加成+N%」的来源（军官/科技/装备）。
	//   0 = 该方无军官 / 老战场快照（不显示拆解括号，仍显示合并加成）。
	AtkOfficerBonus int
	DefOfficerBonus int
	// ★ 2026-10-06 拆解展示：军官加成里「技能」占的百分点（已含在 AtkOfficerBonus 内），
	//   单独展示「军官技能+N%」；老快照 0 → 属性部分 = 总加成，回退旧格式。
	AtkOfficerSkill int
	DefOfficerSkill int
	// ★ 2026-10-06 拆解明细：军官技能 / 科技**逐项**（战报展示「军官技能·尖兵突击+N%」「科技·弹道学+N%」，
	//   多个分开展示）。新战场快照里存这些；老快照为 nil → 回退旧格式合并展示。
	AtkSkillBreak []ezfyBonusItem
	DefSkillBreak []ezfyBonusItem
	AtkTechBreak  []ezfyBonusItem
	DefTechBreak  []ezfyBonusItem
	// ★ 2026-10-06 守方「防御加成」**逐项**明细：城墙 / 科技 / 军官属性 / 军官技能 / 装备，
	//   被攻击时在攻击行展示「防御加成+N%(城墙+50% 科技·装甲科技+30% 军官·冥王+20% 军官技能·弧形防御+60% 装备+20%)」。
	//   Name 直接带前缀；老快照为 nil → 不展示防御段。攻方没有防御加成，不需要攻方版。
	DefDefBreak []ezfyBonusItem
	// ★ 2026-10-07 攻方「防御加成」：原先攻方部队被打时防御加成恒为 0 ——
	//   出征军官带的防御技能(弧形防御/弹幕支援)+属性防御+装备防御 从未生效，战报也不展示
	//   （用户反馈「弧形防御 Lv.5 防御力+150% 完全看不到」）。与守方口径对称：
	//   AtkDefBonus = 攻方军官属性+防御技能+装备 Def 的总和，伤害计算/战斗加成行/被打行都用它；
	//   AtkDefBreak = 逐项明细（Name 带前缀），老快照 0/nil → 回退旧行为(攻方无防御加成)。
	AtkDefBonus   int
	AtkDefBreak   []ezfyBonusItem
	AtkSpeedBonus int
	DefSpeedBonus int
	// ★ 2026-10-06 射程加成%（用户要求：射程 = 兵种基础射程 × (1 + 科技加成)）
	AtkRangeBonus int
	DefRangeBonus int
	AtkEquip      ezfyBattleBonus
	DefEquip      ezfyBattleBonus

	AtkTargets map[int]int
	DefTargets map[int]int
	AtkMoves   map[int]int
	DefMoves   map[int]int

	AtkOfficerDesc string
	DefOfficerDesc string

	// 军官技能「绝地反击」：前N回合被攻击后存活可立即反击攻击者（N=技能等级，1级=前1回合…）
	// AtkCounter/DefCounter 是布尔镜像，仅用于老战场快照回退（见 ezfyBattleStateFromSnapshot）
	AtkCounter bool // 攻方带队军官有「绝地反击」
	DefCounter bool // 守方城守军官有「绝地反击」
	// ★ 2026-10-06 技能随军官等级升级：生效回合数（0=没学）。新战场字段，老快照缺省 0 → 用 bool 回退 1 回合。
	AtkCounterRounds int
	DefCounterRounds int

	// ★ 2026-09-23 指挥室/回合日志里攻守双方的兵种名都显示「阵营兵种名」。
	// 攻方=出征方阵营；守方只有玩家城才有值，野地/AI/寇城为 0（通用名）。
	AtkCamp int
	DefCamp int

	Round       int  // 已结算回合数
	Done        bool // 是否已分胜负 / 回合耗尽
	AttackerWin bool
	Draw        bool // ★ 2026-09-24 打到 40 回合未分胜负 = 平局（守方仍算守住）

	// ★ 2026-10-02 盟军驻军战「溃败撤退」：守方剩余兵力跌破 DefBreak% 时判定战败，
	//   战斗提前结束、守方剩余兵力返航回城（而不是死战到最后一兵）。
	DefBreak        int   // 溃败阈值%（0 = 不启用）
	DefInitialTotal int64 // 守方初始总兵力（DefBreak 的判断基准）

	Head    []string // 开局描述（军官/加成/初始距离），只写一次
	Actions []string // 每回合的行动日志
}

// ezfyNewBattleState 初始化战场
// attackerUnits/defenderUnits: [troopId, count]
// atkBonus: 攻方攻击加成%(军官+科技)  defBonus: 守方防御加成%(城墙+科技+城守)
// defAtkBonus: 守方攻击加成%(科技+军官技能，城防/守城部队行动时用)
// atkSpeedBonus/defSpeedBonus: 速度加成%
// atkRangeBonus/defRangeBonus: 射程加成%（射程 = 基础射程 × (1+加成%)）
// atkEquip/defEquip: 装备六项加成（伤害/防御/生命/移动距离/暴击几率/暴击伤害，单位百分点）
// atkOfficerBonus/defOfficerBonus: 攻/守方攻击加成中「军官」占的百分点（0=无军官；
//
//	仅供战报日志拆解「攻击加成+N%」来源，不改伤害计算）
//
// atkOfficerSkill/defOfficerSkill: 军官加成中「技能」占的百分点（已含在 atkOfficerBonus 内，
//
//	供拆解单独展示「军官技能+N%」；0 → 属性部分=总加成，回退旧格式）
//
// atkSkills/defSkills: 技能**逐项**明细（战报展示「军官技能·尖兵突击+N%」；nil → 回退合并展示）
// atkTechs/defTechs: 攻击加成里科技**逐项**明细（战报展示「科技·弹道学+N%」；nil → 回退合并展示）
// defDefBreak: 守方「防御加成」逐项明细（Name 带前缀：城墙/科技·/军官·/军官技能·/装备；nil → 不展示防御段）
// atkDefBonus: 攻方「防御加成」总和（军官属性+防御技能+装备 Def；0 = 无防御来源，被打时不吃防御减伤）
// atkDefBreak: 攻方「防御加成」逐项明细（Name 带前缀：军官·/军官技能·/装备；nil → 不展示攻方防御段）
// atkTargets/defTargets: 兵种ID->优先攻击兵种ID(0=最近, 司令部配置)
// atkMoves/defMoves: 兵种ID->1前进 0停止（玩家不下指令时的默认行为）
// atkCounterRounds/defCounterRounds: 绝地反击生效回合数（0=没学；1/2/3 = 前N回合）
func ezfyNewBattleState(attackerUnits, defenderUnits []ezfyUnitGroup,
	atkBonus, defBonus, defAtkBonus, atkSpeedBonus, defSpeedBonus int,
	atkRangeBonus, defRangeBonus int,
	atkEquip, defEquip ezfyBattleBonus,
	atkOfficerDesc, defOfficerDesc string,
	atkOfficerBonus, defOfficerBonus int,
	atkOfficerSkill, defOfficerSkill int,
	atkSkills, defSkills, atkTechs, defTechs, defDefBreak []ezfyBonusItem,
	atkDefBonus int, atkDefBreak []ezfyBonusItem,
	atkTargets, defTargets map[int]int,
	atkMoves, defMoves map[int]int,
	atkCounterRounds, defCounterRounds int, atkCamp, defCamp int) *ezfyBattleState {

	st := &ezfyBattleState{
		AtkBonus: atkBonus, DefBonus: defBonus,
		DefAtkBonus:     defAtkBonus,
		AtkOfficerBonus: atkOfficerBonus, DefOfficerBonus: defOfficerBonus,
		AtkOfficerSkill: atkOfficerSkill, DefOfficerSkill: defOfficerSkill,
		AtkSkillBreak: atkSkills, DefSkillBreak: defSkills,
		AtkTechBreak: atkTechs, DefTechBreak: defTechs,
		DefDefBreak: defDefBreak,
		AtkDefBonus: atkDefBonus, AtkDefBreak: atkDefBreak,
		AtkSpeedBonus: atkSpeedBonus, DefSpeedBonus: defSpeedBonus,
		AtkRangeBonus: atkRangeBonus, DefRangeBonus: defRangeBonus,
		AtkEquip: atkEquip, DefEquip: defEquip,
		AtkTargets: atkTargets, DefTargets: defTargets,
		AtkMoves: atkMoves, DefMoves: defMoves,
		AtkOfficerDesc: atkOfficerDesc, DefOfficerDesc: defOfficerDesc,
		AtkCounter: atkCounterRounds > 0, DefCounter: defCounterRounds > 0,
		AtkCounterRounds: atkCounterRounds, DefCounterRounds: defCounterRounds,
		AtkCamp: atkCamp, DefCamp: defCamp,
		Head: []string{}, Actions: []string{},
	}

	// —— 开局描述（与拆分层之前完全一致，保证老战报格式不变）——
	if atkOfficerDesc != "" {
		st.Head = append(st.Head, "【攻方军官】"+atkOfficerDesc)
	}
	if defOfficerDesc != "" {
		st.Head = append(st.Head, "【守方军官】"+defOfficerDesc)
	}
	// ★ 装备六项加成并入基础加成：伤害→攻击、防御→防御、移动距离→速度
	//   （生命/暴击几率/暴击伤害在伤害结算里单独算，见 ezfyCalcDamage）
	effAtk := atkBonus + atkEquip.Dmg
	effDef := defBonus + defEquip.Def
	effAtkSpeed := atkSpeedBonus + atkEquip.Move
	effDefSpeed := defSpeedBonus + defEquip.Move
	// ★ 2026-10-06 守方攻击加成单独列出：城防/守城部队行动时也吃科技+军官技能加成，
	//   攻守双方的「加成伤害」都能在开场加成行里一眼看出来
	// ★ 2026-10-06 射程加成也显示在加成行（用户要求「回合射程加成也要体现出来」，
	//   弹道学=射程）；为 0 时不显示，老战报格式不变
	atkRangeTxt, defRangeTxt := "", ""
	if atkRangeBonus > 0 {
		atkRangeTxt = fmt.Sprintf(" 射程+%d%%", atkRangeBonus)
	}
	if defRangeBonus > 0 {
		defRangeTxt = fmt.Sprintf(" 射程+%d%%", defRangeBonus)
	}
	// ★ 2026-10-07 攻方防御加成（AtkDefBonus 军官部分 + 装备 Def）也显示在加成行：
	//   出征军官带弧形防御/弹幕支援 + 装备防御 → 攻方被打时减伤（原来攻方防御恒 0，看不出带了防御技能）
	st.Head = append(st.Head, fmt.Sprintf("战斗加成: 攻方 攻击+%d%% 防御+%d%% 速度+%d%%%s | 守方 攻击+%d%% 防御+%d%% 速度+%d%%%s",
		effAtk, atkDefBonus+atkEquip.Def, effAtkSpeed, atkRangeTxt, st.DefAtkBonus, effDef, effDefSpeed, defRangeTxt))
	// ★ 2026-10-08 攻/守方科技逐项单列（用户要求罗列）：如「科技·弹道学+30% 科技·装甲科技+15%」
	if aTech := ezfyBonusItemsDesc(atkTechs); aTech != "" {
		st.Head = append(st.Head, "【攻方科技】"+aTech)
	}
	if dTech := ezfyBonusItemsDesc(defTechs); dTech != "" {
		st.Head = append(st.Head, "【守方科技】"+dTech)
	}
	// ★ 2026-10-08 装备加成恒定罗列（不再"非零才显示"；这里的 desc 空值时已返回"无"），
	//   避免玩家反馈的"装备加成看不到"。装备伤害已并入上方「攻击+%d%%」总数值。
	st.Head = append(st.Head, "【攻方装备】"+ezfyEquipBonusDesc(atkEquip))
	st.Head = append(st.Head, "【守方装备】"+ezfyEquipBonusDesc(defEquip))
	st.Head = append(st.Head, fmt.Sprintf("战场初始相距%d, 攻守双方相向推进", ezfyBattleStartDist))

	idx := 0
	for _, ug := range attackerUnits {
		cfg := ezfyStatsOf(ug.TroopId)
		if cfg != nil && ug.Count > 0 {
			idx++
			st.Attackers = append(st.Attackers, &ezfyFightUnit{
				id: fmt.Sprintf("A%d", idx), cfg: cfg, count: ug.Count,
				initialCount: ug.Count, pos: 0})
		}
	}
	idx = 0
	for _, ug := range defenderUnits {
		cfg := ezfyStatsOf(ug.TroopId)
		if cfg != nil && ug.Count > 0 {
			idx++
			st.Defenders = append(st.Defenders, &ezfyFightUnit{
				id: fmt.Sprintf("D%d", idx), cfg: cfg, count: ug.Count,
				initialCount: ug.Count, pos: ezfyBattleStartDist})
		}
	}
	// 一方无兵 → 直接结束（与原实现一致：Rounds = 0）
	if len(st.Attackers) == 0 || len(st.Defenders) == 0 {
		st.Done = true
		st.AttackerWin = len(st.Attackers) > 0
	}
	return st
}

// ezfyMoveDir 本回合的移动方向：1 前进 / 0 暂停 / -1 后退
//
// 优先级：玩家指令 > 司令部「兵种战斗配置」(moveMap) > 默认前进。
// 指令为空串表示「这一方没有下指令」，沿用 moveMap 的原始语义。
func ezfyMoveDir(unit *ezfyFightUnit, moveMap map[int]int, cmd string) int {
	switch cmd {
	case ezfyCmdHold:
		return 0
	case ezfyCmdRetreat:
		return -1
	case ezfyCmdAdvance:
		return 1
	}
	// 与原版 BattleEngine 一致 —— 未配置的兵种默认「前进」
	// (Java: moveMap.getOrDefault(unit.cfg.getId(), 1) > 0)
	// 注意不能写成 moveMap[id] > 0: 缺键会取零值 0 变成「停止」,
	// 导致攻方近战兵种原地不动、永远够不到敌人而必败。
	if moveMap != nil {
		if v, ok := moveMap[unit.cfg.ID]; ok {
			if v > 0 {
				return 1
			}
			return 0
		}
	}
	return 1
}

// ezfyOfficerShortName 从「军官战斗描述」里取军官名
// （desc 形如「名 Lv.3 攻击加成+10%…」或「将名 守军防御+12%…」）
func ezfyOfficerShortName(desc string) string {
	if desc == "" {
		return ""
	}
	if i := strings.Index(desc, " Lv."); i > 0 {
		return desc[:i]
	}
	if i := strings.IndexByte(desc, ' '); i > 0 {
		return desc[:i]
	}
	return desc
}

// ezfyTechItem 单条科技明细（加成 0 → 返回 nil，不展示）
func ezfyTechItem(name string, value int) []ezfyBonusItem {
	if value == 0 {
		return nil
	}
	return []ezfyBonusItem{{Name: name, Value: value}}
}

// ezfyBonusItems 把多条明细拼成一条（全 nil / 零条 → 返回零值 nil）
func ezfyBonusItems(groups ...[]ezfyBonusItem) []ezfyBonusItem {
	out := []ezfyBonusItem{}
	for _, g := range groups {
		out = append(out, g...)
	}
	return out
}

// ezfyDefBonusTxt 被攻击方的「防御加成」来源明细 → 「守方防御加成+N%(…)」/「攻方防御加成+N%(…)」。
// label 传「守方防御加成」/「攻方防御加成」指明归属 —— 不然被打行里光写「防御加成」没标明归属，
// 玩家会把明细里的军官名误读成「给对方加 buff」（用户反馈 2026-10-07：Stalin 的弧形防御看着像给敌人加成）。
// Name 已带前缀（城墙/科技·/军官·/军官技能·/装备）。全零/无明细 → 返回空串（不展示防御段）。
func ezfyDefBonusTxt(label string, items []ezfyBonusItem) string {
	total := 0
	parts := []string{}
	for _, it := range items {
		if it.Value == 0 {
			continue
		}
		total += it.Value
		parts = append(parts, fmt.Sprintf("%s+%d%%", it.Name, it.Value))
	}
	if total == 0 || len(parts) == 0 {
		return ""
	}
	return fmt.Sprintf("%s+%d%%(%s)", label, total, strings.Join(parts, " "))
}

// ezfyBonusBreakdown 「攻击加成+N%」的来源拆解（军官属性/军官技能/科技/装备），写进战报行动日志。
// ★ 2026-10-06 支持**逐项展开**：给了 skills/techs 明细就走逐项展示
//
//	（「军官技能·尖兵突击+90%」「科技·弹道学+54%」，多个分开展示）；
//	明细为 nil（老战场快照）回退旧格式合并展示（军官技能+N% / 科技+N%）。
//
// 只要任一来源非零就展示（PVP 里一方军官没攻击技能/没设城守时，科技加成也照常列出，
// 不会像以前一样整括号消失）；全零返回空串，不改动原样式的合并加成显示。
func ezfyBonusBreakdown(officerBonus, skillBonus, techBase, equipBonus int, name string,
	skills, techs []ezfyBonusItem) string {

	attr := officerBonus - skillBonus
	// 科技合计：有明细用明细之和；老快照回退原口径（科技 = 总加成 - 军官加成）
	techTotal := 0
	for _, t := range techs {
		techTotal += t.Value
	}
	if len(techs) == 0 {
		techTotal = techBase - officerBonus
	}
	if attr == 0 && skillBonus == 0 && techTotal == 0 && equipBonus == 0 {
		return ""
	}
	parts := []string{}
	if name != "" && attr != 0 {
		parts = append(parts, fmt.Sprintf("军官·%s+%d%%", name, attr))
	}
	if len(skills) > 0 {
		for _, s := range skills {
			if s.Value != 0 {
				parts = append(parts, fmt.Sprintf("军官技能·%s+%d%%", s.Name, s.Value))
			}
		}
	} else if skillBonus != 0 {
		parts = append(parts, fmt.Sprintf("军官技能+%d%%", skillBonus))
	}
	if len(techs) > 0 {
		for _, t := range techs {
			if t.Value != 0 {
				parts = append(parts, fmt.Sprintf("科技·%s+%d%%", t.Name, t.Value))
			}
		}
	} else if techTotal != 0 {
		parts = append(parts, fmt.Sprintf("科技+%d%%", techTotal))
	}
	if equipBonus != 0 {
		parts = append(parts, fmt.Sprintf("装备+%d%%", equipBonus))
	}
	if len(parts) == 0 {
		return ""
	}
	return "(" + strings.Join(parts, " ") + ")"
}

// Step 结算**一个回合**。返回 true 表示战斗已结束。
//
// atkCmds: 攻方**逐兵种**的指令（troopId → advance|hold|retreat）。
// defCmds: 守方**逐兵种**的指令（2026-09-23 「敌人打自己，自己也能指挥」——
//
//	玩家守城时与攻方一样逐兵种指挥，AI/野地不下指令时传 nil）。
//
// ★ （2026-09-22）：「指挥不是指挥全部，自己带的兵种都能指挥，就是单独指挥」
// —— 所以指令是按兵种存的，没给的兵种回落到司令部兵种配置。
// ★ 2026-10-08 技能发动播报用的「带感口号」池（克制版，不浮夸）。
//   轮流取 + 攻/守、攻击/防御、回合 三重混合偏移，避免重复枯燥。
var ezfySlogans = []string{
	"战意沸腾", "气势如虹", "士气大振", "杀声震天", "锋芒毕露", "势不可挡",
	"铁血激荡", "锐不可当", "一往无前", "战鼓擂动", "陷阵之志", "气吞万里",
	"剑锋所指", "热血方刚", "战意高昂", "孤掷一注",
}

func (st *ezfyBattleState) Step(atkCmds, defCmds map[int]string) bool {
	if st.Done {
		return true
	}
	st.Round++

	// ★ 2026-10-05 用户要求：战报里的兵种名**统一展示基础兵种名**（不带阵营前缀）。
	//   本函数生成的日志就是战报「战斗过程」正文，故不再按攻守阵营取名。
	stName := func(u *ezfyFightUnit) string {
		if n := ezfyCfg.troopName(u.cfg.ID, 0); n != "" {
			return n
		}
		return u.cfg.Name
	}

	// 装备加成在这里叠加（快照里存的是不含装备的基础值）
	atkBonus := st.AtkBonus + st.AtkEquip.Dmg
	defBonus := st.DefBonus + st.DefEquip.Def
	atkSpeedBonus := st.AtkSpeedBonus + st.AtkEquip.Move
	defSpeedBonus := st.DefSpeedBonus + st.DefEquip.Move
	// ★ 2026-10-06 「攻击加成」来源拆解（军官属性/军官技能/科技/装备），写进每行行动日志，
	//   玩家一眼能看到加成是谁给的 —— 攻方=军官技能+科技+装备，守方=城守技能+科技。
	atkBonusBreak := ezfyBonusBreakdown(st.AtkOfficerBonus, st.AtkOfficerSkill, st.AtkBonus, st.AtkEquip.Dmg,
		ezfyOfficerShortName(st.AtkOfficerDesc), st.AtkSkillBreak, st.AtkTechBreak)
	defBonusBreak := ezfyBonusBreakdown(st.DefOfficerBonus, st.DefOfficerSkill, st.DefAtkBonus, 0,
		ezfyOfficerShortName(st.DefOfficerDesc), st.DefSkillBreak, st.DefTechBreak)
	// ★ 2026-10-06 守方「防御加成」来源拆解：城防爪到谁被打就展示在谁的行动行
	//   （城墙+科技+城守属性+城守技能+装备 → 伤害变少的来源）。攻方没有防御加成 → 只在守方被攻击的行上展示；
	//   老快照无明细 → 不展示防御段。
	defBonusTxt := ezfyDefBonusTxt("守方防御加成", st.DefDefBreak)
	// ★ 2026-10-07 攻方「防御加成」来源拆解：攻方被打（守方行动、守方绝地反击还击）时展示，
	//   来源 = 攻方军官属性+防御技能(弧形防御/弹幕支援)+装备 Def。攻方无明细 → 不展示防御段（回退老行为）。
	atkDefBonusTxt := ezfyDefBonusTxt("攻方防御加成", st.AtkDefBreak)

	st.Actions = append(st.Actions, fmt.Sprintf("第%d回合:", st.Round))

	// ★ 2026-10-08 技能发动播报：每个生效回合都提示。
	//   攻/守方的攻击类技能（AtkSkillBreak/DefSkillBreak）逐个写一行带感口号，再带出防御类军官技能
	//   （弧形防御等，明细 Name 带「军官技能·」前缀 → 去掉前缀后发动，说明全体防御力+N%）。
	//   纯配置/科技/装备来源不播报（无「军官技能·」前缀不碰）。「攻击加成+N%」拆解保持原样。
	skillAnno := func(side, offName string, off int, wording string, skills []ezfyBonusItem) {
		for i, s := range skills {
			if s.Value == 0 {
				continue
			}
			idx := (st.Round + i*3 + off*7) % len(ezfySlogans)
			st.Actions = append(st.Actions,
				fmt.Sprintf("%s%s发动【%s】！%s，全体%s+%d%%", side, offName, s.Name, ezfySlogans[idx], wording, s.Value))
		}
	}
	// 只挑明细分里确实是「军官技能」的来源（防御/科技/城防/装备等不算技能发动）
	officerSkillOnly := func(items []ezfyBonusItem) []ezfyBonusItem {
		out := []ezfyBonusItem{}
		for _, it := range items {
			if strings.HasPrefix(it.Name, "军官技能·") {
				it.Name = strings.TrimPrefix(it.Name, "军官技能·")
				out = append(out, it)
			}
		}
		return out
	}
	skillAnno("【攻方】", ezfyOfficerShortName(st.AtkOfficerDesc), 0, "攻击力", st.AtkSkillBreak)
	skillAnno("【守方】", ezfyOfficerShortName(st.DefOfficerDesc), 1, "攻击力", st.DefSkillBreak)
	skillAnno("【攻方】", ezfyOfficerShortName(st.AtkOfficerDesc), 2, "防御力", officerSkillOnly(st.AtkDefBreak))
	skillAnno("【守方】", ezfyOfficerShortName(st.DefOfficerDesc), 3, "防御力", officerSkillOnly(st.DefDefBreak))
	all := append(append([]*ezfyFightUnit{}, st.Attackers...), st.Defenders...)
	sort.SliceStable(all, func(i, j int) bool { return all[i].cfg.Speed > all[j].cfg.Speed })

	for _, unit := range all {
		if !unit.alive() {
			continue
		}
		isAtk := ezfyContains(st.Attackers, unit)
		side := "【守方】"
		enemySide := "攻方"
		if isAtk {
			side = "【攻方】"
			enemySide = "守方"
		}
		enemies := st.Defenders
		if !isAtk {
			enemies = st.Attackers
		}
		liveEnemies := ezfyAliveList(enemies)
		if len(liveEnemies) == 0 {
			break
		}
		targetMap := st.DefTargets
		if isAtk {
			targetMap = st.AtkTargets
		}
		target := ezfyPickTarget(unit, liveEnemies, targetMap)

		// ★ 2026-10-06 射程 = 兵种基础射程 × (1 + 射程科技加成%)（用户要求）
		rangeD := unit.cfg.AttackRange
		if isAtk {
			rangeD = unit.cfg.AttackRange * (100 + st.AtkRangeBonus) / 100
		} else {
			rangeD = unit.cfg.AttackRange * (100 + st.DefRangeBonus) / 100
		}
		dist := ezfyAbs(target.pos - unit.pos)

		moveMap := st.DefMoves
		cmd := ""
		speedBonus := defSpeedBonus
		cmds := defCmds
		if isAtk {
			moveMap = st.AtkMoves
			speedBonus = atkSpeedBonus
			cmds = atkCmds
		}
		// ★ 按兵种取指令（缺省 = 空串 → 回落司令部兵种配置）
		//   0 号键是旧格式遗留的「全军统一指令」，老战场记录也能正常打。
		//   攻守双方同口径：玩家守城时守军也能逐兵种指挥。
		if cmds != nil {
			if v, ok := cmds[unit.cfg.ID]; ok {
				cmd = v
			} else if v, ok := cmds[0]; ok {
				cmd = v
			}
		}

		dir := ezfyMoveDir(unit, moveMap, cmd)
		// ★ 2026-10-06 冲锋到贴脸（用户确认的前进语义）：
		//   前进不再「进射程就停」，而是**一路冲到与最近敌人贴身**为止（沿途进射程照常开火）。
		//   移动量截断到「最近敌人当前距离 - 1」：单位按速度从高到低**逐个行动**，
		//   后行动的单位看到的是最新位置，截断到 dist-1 天然不会互相穿过（修复用户反馈的
		//   「穿过对面无限前进」bug —— 原实现双方各自停在射程边缘，既不贴脸又相互够不到）。
		nearestDist := int(^uint(0) >> 1)
		for _, e := range liveEnemies {
			if d := ezfyAbs(e.pos - unit.pos); d < nearestDist {
				nearestDist = d
			}
		}
		// 后退随时可执行（2026-09-24 用户反馈「点后退没反应」：接战后已在射程内，
		// 老条件 dist > rangeD 会跳过后退）。移动量 = 兵种速度 × (1+速度加成)。
		moved := false
		if dir != 0 && (nearestDist > 1 || dir < 0) {
			move := unit.cfg.Speed * (100 + speedBonus) / 100
			if dir > 0 {
				// 前进最多推到「与最近敌人相距 1」（贴身），不越过敌人
				if limit := nearestDist - 1; move > limit {
					move = limit
				}
				if move < 1 {
					move = 0
				}
			}
			if move != 0 {
				move *= dir
				if isAtk {
					unit.pos += move
				} else {
					unit.pos -= move
				}
				// ★ 2026-09-23 后退最多 10000，不能无限制后退。
				//   攻方起点 0 → 最低 -10000；守方起点 6000 → 最高 16000。超出夹回。
				if isAtk {
					if unit.pos < -ezfyBattleRetreatMax {
						unit.pos = -ezfyBattleRetreatMax
					}
				} else {
					if unit.pos > ezfyBattleStartDist+ezfyBattleRetreatMax {
						unit.pos = ezfyBattleStartDist + ezfyBattleRetreatMax
					}
				}
				dist = ezfyAbs(target.pos - unit.pos)
				// 移动后重算最近距离（后方开火切换目标用）
				nearestDist = int(^uint(0) >> 1)
				for _, e := range liveEnemies {
					if d := ezfyAbs(e.pos - unit.pos); d < nearestDist {
						nearestDist = d
					}
				}
				verb := "前进"
				if dir < 0 {
					verb = "后撤"
				}
				st.Actions = append(st.Actions, fmt.Sprintf("%s%s%s%d, 与%s%s相距%d",
					side, stName(unit), verb, ezfyAbs(move), enemySide, stName(target), dist))
				moved = true
			}
		}
		if !moved && dist > rangeD {
			// ★ 2026-09-29 用户反馈「选了攻击目标但对方火箭不在射程内，就啥也没操作，玩家不知道咋回事」：
			//   本回合单位既没移动（指令 hold / 已贴身停止）也没开火 → 完全静默。
			//   追加一条行动日志说明「够不到目标、无法攻击」，让玩家明白不是 bug。
			st.Actions = append(st.Actions, fmt.Sprintf("%s%s 距目标%s%s%d，超出射程%d，无法攻击",
				side, stName(unit), enemySide, stName(target), dist, rangeD))
		}
		// ★ 2026-10-04 优先目标被挡在射程外、但最近敌人在射程内 → 改打最近的（避免干站）。
		//   配合上方「前进不越过最近敌人」的修复：停在最近敌人射程边缘后能立刻开火。
		if dist > rangeD && nearestDist <= rangeD {
			target = ezfyPickTarget(unit, liveEnemies, nil)
			dist = ezfyAbs(target.pos - unit.pos)
		}
		if dist <= rangeD {
			baseAtk := ezfyPickAttack(unit.cfg, target.cfg)
			// ★ 攻守参数取值（2026-09-23 修复「守方防御加成完全不生效」）：
			//   ezfyCalcDamage 的第 5 个参数是**被攻击方(target)的防御加成**，
			//   而 target 是谁取决于本回合哪一方在行动：
			//     · 攻方行动 → target 是守方 → 该传守方防御加成 defBonus
			//     · 守方行动 → target 是攻方 → 攻方没有防御加成，传 0
			//   老实现恰好写反了（攻方传 0、守方传 defBonus），后果是
			//   **城墙 / 城守 / 防御装备在被打时完全不起作用**，反而在守方反击时
			//   把守方自己的防御加成算到了攻方身上。
			//   回归测试：ezfy_defbonus_test.go（修复前 +100% 防御与 +0% 损失完全相同）。
			unitAtkBonus := 0
			unitDefBonus := 0
			equip := st.DefEquip
			if isAtk {
				unitAtkBonus = atkBonus
				// ★ 2026-10-07 守方防御加成 = 基础(城墙+科技+城守) + 装备 Def（原来漏了装备 Def，
				//   战斗加成行显示「防御+X%」与伤害减伤不一致）
				unitDefBonus = defBonus + st.DefEquip.Def
				equip = st.AtkEquip
			} else {
				// ★ 2026-10-06 守方行动时也要吃科技+军官技能的攻击加成
				//   （原实现 unitAtkBonus=0，城防/守城部队打人完全没加成 —— 用户反馈
				//   「科技加成、技能加成没有算入伤害当中」的根因之一）
				unitAtkBonus = st.DefAtkBonus
				// ★ 2026-10-07 攻方「防御加成」= 出征军官属性+防御技能(弧形防御/弹幕支援)+装备 Def。
				//   原来攻方被打时防御恒 0 —— 攻方军官带弧形防御 Lv.5「防御力+150%」完全看不见也不生效（用户反馈）。
				unitDefBonus = st.AtkDefBonus + st.AtkEquip.Def
			}
			bonusBreak := atkBonusBreak
			if !isAtk {
				bonusBreak = defBonusBreak
			}
			// ★ 生命加成：守方装备的生命%让同一发伤害打掉的兵更少
			//   （等价于「有效生命 = 兵种生命 × (1 + 生命加成%)」）
			hpMul := 100
			if equip.Hp != 0 {
				hpMul = 100 + equip.Hp
				if hpMul < 1 {
					hpMul = 1
				}
			}
			damage := ezfyCalcDamage(baseAtk, target.cfg.Defence, unit.count, unitAtkBonus, unitDefBonus)
			// ★ 暴击：按暴击几率 roll（几率封顶 100%），命中则乘 (1 + 暴击伤害加成)
			//
			// ⚠️ 2026-09-23 用户反馈「暴击打了 1 个单位」→ 老实现有两个坑：
			//   ① 暴击几率是装备累加值，可以超过 100（如「革命者[折扇]」crit=125），
			//      老写法 `rand.Intn(100) < 125` 恒真 = 必定暴击，几率形同虚设；
			//   ② 暴击伤害加成可能是 0（如「黑色幽灵[徽章]」crit=125 / crit_dmg=0），
			//      老写法伤害 ×(100+0)/100 = 原样不变 → 出了【暴击】却一点没多打，
			//      看起来就是「暴击只打了 1 个」。
			//   现在：几率封顶 100%；暴击伤害加成 <= 0 时按基础倍率兜底，
			//   保证「只要出了暴击，就一定要比不暴击打得更疼」。
			crit := false
			critBonus := 0
			if equip.Crit > 0 {
				chance := equip.Crit
				if chance > 100 {
					chance = 100
				}
				if rand.Intn(100) < chance {
					crit = true
					critBonus = equip.CritDmg
					if critBonus <= 0 {
						critBonus = ezfyCritBaseBonusPct
					}
					damage = damage * int64(100+critBonus) / 100
				}
			}
			// ★ 伤害按「剩余伤害」逐目标结算：先把本次全部伤害打在首选目标上；
			//   若全歼且伤害还有溢出 → 触发【势不可挡】，溢出伤害继续打下一个存活目标，
			//   若仍未全歼且还有溢出则继续级联，直到伤害耗尽或对方全灭。
			//   ★ 2026-09-23 溢出打新目标时要**按新目标重新算伤害** ——
			//   兵种类型不同要重选攻击属性(对海/对陆/对空)，防御不同要重算减免，
			//   「B 比 A 防御高，对 B 的伤害要按 B 的防御重新折算」，以此类推。
			//   实现：剩余伤害按「对原目标满额伤害 / 对新目标满额伤害」等比折算。
			critTxt := ""
			if crit {
				// 把实际生效的暴击倍率写进战报，玩家一眼能看出暴击有没有生效
				critTxt = fmt.Sprintf("【暴击+%d%%】", critBonus)
			}
			remaining := damage
			first := true
			lastTarget := target // 当前 remaining 是按 lastTarget 的防御/兵种类型算出的
			for remaining > 0 {
				live := ezfyAliveList(enemies)
				if len(live) == 0 {
					break
				}
				cur := target
				if !first || !cur.alive() {
					// ★ 2026-10-04 修复「指挥模块溢出伤害有 bug」：势不可挡的溢出伤害只能继续打
					//   **射程内**的敌人。原实现从全部存活敌人里挑下一个目标、无视射程，
					//   短射程兵种能隔着半个战场溅射（用户反馈）。
					inRange := make([]*ezfyFightUnit, 0, len(live))
					for _, e := range live {
						if ezfyAbs(e.pos-unit.pos) <= rangeD {
							inRange = append(inRange, e)
						}
					}
					if len(inRange) == 0 {
						break // 射程内没有敌人了，溢出伤害到此为止
					}
					cur = ezfyPickTarget(unit, inRange, targetMap)
					if cur == nil {
						break
					}
				}
				// ★ 命中新目标：按新目标重选攻击属性、重算防御减免，折算剩余伤害
				if cur != lastTarget {
					dFrom := ezfyCalcDamage(ezfyPickAttack(unit.cfg, lastTarget.cfg), lastTarget.cfg.Defence, unit.count, unitAtkBonus, unitDefBonus)
					dTo := ezfyCalcDamage(ezfyPickAttack(unit.cfg, cur.cfg), cur.cfg.Defence, unit.count, unitAtkBonus, unitDefBonus)
					if dFrom > 0 {
						remaining = int64(float64(remaining) * float64(dTo) / float64(dFrom))
					}
					lastTarget = cur
				}
				chp := cur.cfg.Health * hpMul / 100
				if chp < 1 {
					chp = 1
				}
				killed := remaining / int64(chp)
				if first {
					// ★ 暴击向上取整（有余数就多杀 1 个）：
					//   暴击后的伤害往往不是目标血量的整数倍，老写法直接向下取整会把暴击的增益
					//   整个抹掉 —— 伤害明明放大到 1.5 倍，killed 还是 1，玩家看到的就是
					//   「暴击打了 1 个单位」（2026-09-23 用户反馈）。这里把零头补成 1 个。
					if crit && remaining%int64(chp) != 0 {
						killed++
					}
					if killed < 1 {
						killed = 1
					}
				} else {
					// 溢出伤害不足以再消灭一个单位 → 停，不再多打
					if killed < 1 {
						break
					}
				}
				overflow := int64(0)
				if killed >= cur.count {
					killed = cur.count
					overflow = remaining - cur.count*int64(chp)
					if overflow < 0 {
						overflow = 0
					}
				}
				cur.count -= killed
				if first {
					// ★ 2026-10-06 攻击行写明「攻击加成+N%」与来源拆解(军官·名+/科技+/装备+)
					//   「造成D伤害」改为**本目标实际承受的伤害**（消灭数×有效生命），
					//   与下方【势不可挡】各行的伤害加总=这次攻击的总伤害，方便玩家核算。
					// ★ 2026-10-06 守方被攻击时追加「防御加成+N%(来源)」—— 让玩家看出伤害为啥少打了。
					line := fmt.Sprintf("%s%s攻击%s%s%s, 攻击加成+%d%%%s",
						side, stName(unit), critTxt, enemySide, stName(cur), unitAtkBonus, bonusBreak)
					if isAtk {
						// 攻方打守方 → 展示守方防御加成
						if defBonusTxt != "" {
							line += ", " + defBonusTxt
						}
					} else if atkDefBonusTxt != "" {
						// 守方打攻方 → 展示攻方防御加成（弧形防御/弹幕支援/装备）
						line += ", " + atkDefBonusTxt
					}
					line += fmt.Sprintf(", 造成%d伤害, 消灭%d个", killed*int64(chp), killed)
					st.Actions = append(st.Actions, line)
				} else {
					line := fmt.Sprintf("%s%s【势不可挡】溢出伤害继续攻击%s%s, 攻击加成+%d%%%s",
						side, stName(unit), enemySide, stName(cur), unitAtkBonus, bonusBreak)
					if isAtk {
						// 攻方打守方 → 展示守方防御加成
						if defBonusTxt != "" {
							line += ", " + defBonusTxt
						}
					} else if atkDefBonusTxt != "" {
						// 守方打攻方 → 展示攻方防御加成（弧形防御/弹幕支援/装备）
						line += ", " + atkDefBonusTxt
					}
					line += fmt.Sprintf(", 造成%d伤害, 消灭%d个", killed*int64(chp), killed)
					st.Actions = append(st.Actions, line)
				}
				remaining = overflow
				first = false
			}
		}
		// ★ 军官技能「绝地反击」：**前N回合**被打且存活 → 立即反击本次攻击者。
		//   2026-10-06 技能随军官等级自动升级：1级=前1回合 / 2级=前2回合 / 3级=前3回合。
		if dist <= rangeD && target.alive() {
			counterRounds := st.AtkCounterRounds // target 是攻方时用攻方回合数
			if isAtk {
				counterRounds = st.DefCounterRounds // 攻方在打 → 被打方是守方，用守方回合数
			}
			if st.Round <= counterRounds && counterRounds > 0 {
				// 反击方(target)攻击加成 / 被反击方(unit)防御加成 —— 与常规攻击同一口径。
				// ★ 2026-10-06 修复：守方反击时原来 cbAtk 恒为 0（「绝地反击」技能没算进伤害），
				//   现在守方反击也吃守方攻击加成(科技+军官技能)。
				cbAtk, cbDef := 0, 0
				if isAtk { // 攻方在打 → 被打的是守方 → 守方发动反击（被还击方=攻方）
					cbAtk = st.DefAtkBonus
					// ★ 2026-10-07 攻方被守方反击 → 攻方防御加成减伤（原来恒 0）
					cbDef = st.AtkDefBonus + st.AtkEquip.Def
				} else { // 守方在打 → 被打的是攻方 → 攻方发动反击（被还击方=守方）
					cbAtk = atkBonus
					// ★ 2026-10-07 守方被攻方反击 → 守方防御加成（基础+装备）减伤
					cbDef = defBonus + st.DefEquip.Def
				}
				dmg := ezfyCalcDamage(ezfyPickAttack(target.cfg, unit.cfg), unit.cfg.Defence, target.count, cbAtk, cbDef)
				kCnt := dmg / int64(unit.cfg.Health)
				if kCnt < 1 {
					kCnt = 1
				}
				if kCnt > unit.count {
					kCnt = unit.count
				}
				unit.count -= kCnt
				// ★ 2026-10-06 反击行标明「军官技能·绝地反击」+ 加成来源/伤害，攻守的技能触发一眼可见
				//   ★ 2026-10-06 v2: 反击方侧标也带【】括号（与攻击行口径一致：/【攻方】【守方】对齐）
				counterBreak := defBonusBreak
				if !isAtk {
					counterBreak = atkBonusBreak
				}
				// ★ 2026-10-08 绝地反击显示方向修正：
				//   还击方是**被攻击方 target**（拥有绝地反击），发起攻击的 unit 是被还击方。
				//   侧标/军官/兵种一律取「还击方」，并带带感口号，攻守视角与颜色随还击方阵营对齐
				//   （原来误用 actor(side/unit)，玩家是攻方时却显示「守方…」，描述、颜色都反了）。
				actorBare := strings.Trim(side, "【】")
				counterSide := enemySide // 还击方 = 被攻击方 = 发起方的对手
				counterOfficer := ezfyOfficerShortName(st.DefOfficerDesc)
				if counterSide == "攻方" {
					counterOfficer = ezfyOfficerShortName(st.AtkOfficerDesc)
				}
				counterSlogan := ezfySlogans[(st.Round*5+2)%len(ezfySlogans)]
				line := fmt.Sprintf("【%s】%s发动【绝地反击】%s还击%s%s！%s，攻击加成+%d%%%s",
					counterSide, counterOfficer, stName(target), actorBare, stName(unit), counterSlogan, cbAtk, counterBreak)
				// 反击行（方向与常规攻击相反：cur 在行动、target 还击）：
				//   攻方在打(isAtk=true)、守方反击 → 被还击方是攻方 → 展示攻方防御加成；
				//   守方在打(isAtk=false)、攻方反击 → 被还击方是守方 → 展示守方防御加成。
				if isAtk {
					if atkDefBonusTxt != "" {
						line += ", " + atkDefBonusTxt
					}
				} else if defBonusTxt != "" {
					line += ", " + defBonusTxt
				}
				line += fmt.Sprintf(", 造成%d伤害, 消灭%d个", dmg, kCnt)
				st.Actions = append(st.Actions, line)
			}
		}
		if len(ezfyAliveList(enemies)) == 0 {
			st.Done = true
			st.AttackerWin = isAtk
			break
		}
	}

	if !st.Done {
		if len(ezfyAliveList(st.Defenders)) == 0 {
			st.Done, st.AttackerWin = true, true
		} else if len(ezfyAliveList(st.Attackers)) == 0 {
			st.Done, st.AttackerWin = true, false
		} else if st.DefBreak > 0 && st.DefInitialTotal > 0 {
			// ★ 2026-10-02 盟军驻军战：守方剩余兵力跌破阈值 → 溃败撤退（攻方突破、战斗提前结束，
			//   守方剩余兵力返航回城）。放在 40 回合平局**之前**：被打到阈值以下算战败，
			//   而不是拖到回合上限按平局处理。
			defLeft := int64(0)
			for _, d := range st.Defenders {
				defLeft += d.count
			}
			if defLeft*100 < int64(st.DefBreak)*st.DefInitialTotal {
				st.Done, st.AttackerWin = true, true
			}
		} else if st.Round >= ezfyBattleMaxRounds {
			// ★ 2026-09-24 40 回合未分胜负按**平局**描述（守方视为守住，攻方无胜果）
			st.Done, st.Draw = true, true
		}
	}
	return st.Done
}

// Result 汇总战场结果（格式与拆分层之前完全一致）
func (st *ezfyBattleState) Result() ezfyBattleResult {
	actions := append([]string{}, st.Head...)
	actions = append(actions, st.Actions...)
	return ezfyBattleResult{
		AttackerWin:    st.AttackerWin,
		Draw:           st.Draw,
		Rounds:         st.Round,
		Actions:        actions,
		AttackerLosses: ezfyToGroups(st.Attackers, true),
		DefenderLosses: ezfyDefLossGroups(st.Defenders),
		AttackerLeft:   ezfyToGroups(st.Attackers, false),
		DefenderLeft:   ezfyToGroups(st.Defenders, false),
	}
}

// ezfySimulate 执行战斗（兼容包装：一次跑完）
// ★ 2026-10-06 新加成参数（守方攻击/射程）兼容包装一律传 0，行为不变；
//
//	军官加成拆解参数（atkOfficerBonus/defOfficerBonus）仅供战报日志展示
func ezfySimulate(attackerUnits, defenderUnits []ezfyUnitGroup,
	atkBonus, defBonus, atkSpeedBonus, defSpeedBonus int,
	atkRangeBonus, defRangeBonus, defAtkBonus int,
	atkEquip, defEquip ezfyBattleBonus,
	atkOfficerDesc, defOfficerDesc string,
	atkOfficerBonus, defOfficerBonus int,
	atkTargets, defTargets map[int]int,
	atkMoves, defMoves map[int]int) ezfyBattleResult {

	st := ezfyNewBattleState(attackerUnits, defenderUnits,
		atkBonus, defBonus, defAtkBonus, atkSpeedBonus, defSpeedBonus,
		atkRangeBonus, defRangeBonus,
		atkEquip, defEquip, atkOfficerDesc, defOfficerDesc,
		atkOfficerBonus, defOfficerBonus,
		0, 0, // 技能拆解：simulate 包装无军官技能拆分，攻方/守方技能算在 officerBonus 内（回退合并展示）
		nil, nil, nil, nil, nil, // 技能/科技/防御逐项明细：simulate 无军官/无科技上下文（回退展示）
		0, nil, // 攻方防御加成：simulate 无军官上下文（攻方无防御加成，回退老行为）
		atkTargets, defTargets, atkMoves, defMoves, 0, 0, 0, 0)
	for !st.Done {
		// nil = 沿用司令部的兵种战斗配置，与原实现行为一致
		st.Step(nil, nil)
	}
	return st.Result()
}

// ezfySimulateBreak 盟军驻军战专用：与 ezfySimulate 相同，但守方剩余兵力跌破 defBreakPct%
// 时判定「溃败撤退」—— 战斗提前结束、守方剩余兵力返航回城（而非死战到最后一兵）。
// 仅驻军战调用，其它战斗路径（指挥室/野地/活动）不受影响。
func ezfySimulateBreak(attackerUnits, defenderUnits []ezfyUnitGroup,
	atkBonus, defBonus, atkSpeedBonus, defSpeedBonus int,
	atkRangeBonus, defRangeBonus, defAtkBonus int,
	atkEquip, defEquip ezfyBattleBonus,
	atkOfficerDesc, defOfficerDesc string,
	atkOfficerBonus, defOfficerBonus int,
	atkTargets, defTargets map[int]int,
	atkMoves, defMoves map[int]int,
	defBreakPct int) ezfyBattleResult {

	st := ezfyNewBattleState(attackerUnits, defenderUnits,
		atkBonus, defBonus, defAtkBonus, atkSpeedBonus, defSpeedBonus,
		atkRangeBonus, defRangeBonus,
		atkEquip, defEquip, atkOfficerDesc, defOfficerDesc,
		atkOfficerBonus, defOfficerBonus,
		0, 0, // 技能拆解：驻军战守方无军官；攻方技能算在 officerBonus 内（回退合并展示）
		nil, nil, nil, nil, nil, // 技能/科技/防御逐项明细：simulateBreak 无军官/无科技上下文（回退展示）
		0, nil, // 攻方防御加成：simulateBreak 无军官上下文（攻方无防御加成，回退老行为）
		atkTargets, defTargets, atkMoves, defMoves, 0, 0, 0, 0)
	if defBreakPct > 0 {
		for _, d := range defenderUnits {
			if d.Count > 0 {
				st.DefInitialTotal += d.Count
			}
		}
		st.DefBreak = defBreakPct
	}
	for !st.Done {
		st.Step(nil, nil)
	}
	return st.Result()
}

// ============ 战场快照（存库 / 重建） ============

// ezfyBattleUnitSnap 战场单位快照
//
// ★ 除了 troop_id，还把**兵种战斗属性**一起存下来：
// 指挥室是懒结算（一场可能横跨几十分钟），中途管理员改兵种配置/删兵种
// 都不该影响已经开打的这一场 —— 存了属性就不依赖配置表，重建必定无损。
type ezfyBattleUnitSnap struct {
	ID           string `json:"id"`
	TroopId      int    `json:"troop_id"`
	Count        int64  `json:"count"`
	InitialCount int64  `json:"initial_count"`
	Pos          int    `json:"pos"`

	Type        int    `json:"type"`
	Name        string `json:"name"`
	Health      int    `json:"health"`
	AtkSea      int    `json:"atk_sea"`
	AtkGround   int    `json:"atk_ground"`
	AtkAir      int    `json:"atk_air"`
	Defence     int    `json:"defence"`
	Speed       int    `json:"speed"`
	AttackRange int    `json:"attack_range"`
}

// ezfyBattleSnapshot 战场快照：指挥室把整场状态存进 ezfy_battle.state，
// 每次请求重建 → 推进 → 再存回（懒结算，不需要常驻定时器）。
type ezfyBattleSnapshot struct {
	Attackers []ezfyBattleUnitSnap `json:"attackers"`
	Defenders []ezfyBattleUnitSnap `json:"defenders"`

	AtkBonus        int             `json:"atk_bonus"`
	DefBonus        int             `json:"def_bonus"`
	DefAtkBonus     int             `json:"def_atk_bonus"`
	AtkSpeedBonus   int             `json:"atk_speed_bonus"`
	DefSpeedBonus   int             `json:"def_speed_bonus"`
	AtkRangeBonus   int             `json:"atk_range_bonus"`
	DefRangeBonus   int             `json:"def_range_bonus"`
	AtkEquip        ezfyBattleBonus `json:"atk_equip"`
	DefEquip        ezfyBattleBonus `json:"def_equip"`
	AtkTargets      map[int]int     `json:"atk_targets"`
	DefTargets      map[int]int     `json:"def_targets"`
	AtkMoves        map[int]int     `json:"atk_moves"`
	DefMoves        map[int]int     `json:"def_moves"`
	AtkOfficerDesc  string          `json:"atk_officer_desc"`
	DefOfficerDesc  string          `json:"def_officer_desc"`
	AtkOfficerBonus int             `json:"atk_officer_bonus"`
	DefOfficerBonus int             `json:"def_officer_bonus"`
	// ★ 2026-10-06 军官加成中「技能」占的百分点（老快照没有 → 0，回退旧格式）
	AtkOfficerSkill int `json:"atk_officer_skill"`
	DefOfficerSkill int `json:"def_officer_skill"`
	// ★ 2026-10-06 加成拆解**逐项**明细（老快照没有 → nil，回退旧格式合并展示）
	AtkSkillBreak []ezfyBonusItem `json:"atk_skill_break,omitempty"`
	DefSkillBreak []ezfyBonusItem `json:"def_skill_break,omitempty"`
	AtkTechBreak  []ezfyBonusItem `json:"atk_tech_break,omitempty"`
	DefTechBreak  []ezfyBonusItem `json:"def_tech_break,omitempty"`
	// ★ 2026-10-06 守方「防御加成」逐项明细（老快照没有 → nil → 不展示防御段）
	DefDefBreak []ezfyBonusItem `json:"def_def_break,omitempty"`
	// ★ 2026-10-07 攻方「防御加成」：总值 + 逐项明细（老快照没有 → 0/nil → 攻方无防御加成，回退老行为）
	AtkDefBonus int             `json:"atk_def_bonus"`
	AtkDefBreak []ezfyBonusItem `json:"atk_def_break,omitempty"`

	AtkCounter bool `json:"atk_counter"`
	DefCounter bool `json:"def_counter"`
	// ★ 2026-10-06 绝地反击生效回合数（新字段；老快照没有 → 0，由 FromSnapshot 用 bool 回退成 1 回合）
	AtkCounterRounds int `json:"atk_counter_rounds"`
	DefCounterRounds int `json:"def_counter_rounds"`

	AtkCamp int `json:"atk_camp"`
	DefCamp int `json:"def_camp"`

	Round       int      `json:"round"`
	Done        bool     `json:"done"`
	AttackerWin bool     `json:"attacker_win"`
	Draw        bool     `json:"draw"`
	Head        []string `json:"head"`
	Actions     []string `json:"actions"`
}

func ezfyUnitSnapOf(list []*ezfyFightUnit) []ezfyBattleUnitSnap {
	out := []ezfyBattleUnitSnap{}
	for _, u := range list {
		out = append(out, ezfyBattleUnitSnap{
			ID: u.id, TroopId: u.cfg.ID, Count: u.count,
			InitialCount: u.initialCount, Pos: u.pos,
			Type: u.cfg.Type, Name: u.cfg.Name, Health: u.cfg.Health,
			AtkSea: u.cfg.AtkSea, AtkGround: u.cfg.AtkGround, AtkAir: u.cfg.AtkAir,
			Defence: u.cfg.Defence, Speed: u.cfg.Speed, AttackRange: u.cfg.AttackRange,
		})
	}
	return out
}

// Snapshot 导出快照（存库用）
func (st *ezfyBattleState) Snapshot() ezfyBattleSnapshot {
	// ★ 性能：快照整段 JSON 存库，每次推进都要「反序列化 → 序列化」，
	//   日志无限增长会让开销线性变大 —— 这里截掉最早的日志（正常 40 回合远达不到上限）。
	actions := st.Actions
	if len(actions) > ezfyBattleLogMax {
		actions = actions[len(actions)-ezfyBattleLogMax:]
	}
	return ezfyBattleSnapshot{
		Attackers: ezfyUnitSnapOf(st.Attackers),
		Defenders: ezfyUnitSnapOf(st.Defenders),

		AtkBonus: st.AtkBonus, DefBonus: st.DefBonus,
		DefAtkBonus:     st.DefAtkBonus,
		AtkOfficerBonus: st.AtkOfficerBonus, DefOfficerBonus: st.DefOfficerBonus,
		AtkOfficerSkill: st.AtkOfficerSkill, DefOfficerSkill: st.DefOfficerSkill,
		AtkSkillBreak: st.AtkSkillBreak, DefSkillBreak: st.DefSkillBreak,
		AtkTechBreak: st.AtkTechBreak, DefTechBreak: st.DefTechBreak,
		DefDefBreak:   st.DefDefBreak,
		AtkDefBonus:   st.AtkDefBonus,
		AtkDefBreak:   st.AtkDefBreak,
		AtkSpeedBonus: st.AtkSpeedBonus, DefSpeedBonus: st.DefSpeedBonus,
		AtkRangeBonus: st.AtkRangeBonus, DefRangeBonus: st.DefRangeBonus,
		AtkEquip: st.AtkEquip, DefEquip: st.DefEquip,
		AtkTargets: st.AtkTargets, DefTargets: st.DefTargets,
		AtkMoves: st.AtkMoves, DefMoves: st.DefMoves,
		AtkOfficerDesc: st.AtkOfficerDesc, DefOfficerDesc: st.DefOfficerDesc,
		AtkCounter: st.AtkCounter, DefCounter: st.DefCounter,
		AtkCounterRounds: st.AtkCounterRounds, DefCounterRounds: st.DefCounterRounds,
		AtkCamp: st.AtkCamp, DefCamp: st.DefCamp,

		Round: st.Round, Done: st.Done, AttackerWin: st.AttackerWin, Draw: st.Draw,
		Head: st.Head, Actions: actions,
	}
}

// ezfyBattleStateFromSnapshot 从快照重建战场（兵种配置按 troop_id 重新查表）
func ezfyBattleStateFromSnapshot(snap ezfyBattleSnapshot) *ezfyBattleState {
	// ★ 2026-10-06 老快照（存的是 atk_counter:true 布尔）没有 round 字段 → 回退成前 1 回合
	atkRounds, defRounds := snap.AtkCounterRounds, snap.DefCounterRounds
	if atkRounds == 0 && snap.AtkCounter {
		atkRounds = 1
	}
	if defRounds == 0 && snap.DefCounter {
		defRounds = 1
	}
	st := &ezfyBattleState{
		AtkBonus: snap.AtkBonus, DefBonus: snap.DefBonus,
		DefAtkBonus:     snap.DefAtkBonus,
		AtkOfficerBonus: snap.AtkOfficerBonus, DefOfficerBonus: snap.DefOfficerBonus,
		AtkOfficerSkill: snap.AtkOfficerSkill, DefOfficerSkill: snap.DefOfficerSkill,
		AtkSkillBreak: snap.AtkSkillBreak, DefSkillBreak: snap.DefSkillBreak,
		AtkTechBreak: snap.AtkTechBreak, DefTechBreak: snap.DefTechBreak,
		DefDefBreak:   snap.DefDefBreak,
		AtkDefBonus:   snap.AtkDefBonus,
		AtkDefBreak:   snap.AtkDefBreak,
		AtkSpeedBonus: snap.AtkSpeedBonus, DefSpeedBonus: snap.DefSpeedBonus,
		AtkRangeBonus: snap.AtkRangeBonus, DefRangeBonus: snap.DefRangeBonus,
		AtkEquip: snap.AtkEquip, DefEquip: snap.DefEquip,
		AtkTargets: snap.AtkTargets, DefTargets: snap.DefTargets,
		AtkMoves: snap.AtkMoves, DefMoves: snap.DefMoves,
		AtkOfficerDesc: snap.AtkOfficerDesc, DefOfficerDesc: snap.DefOfficerDesc,
		AtkCounter: snap.AtkCounter, DefCounter: snap.DefCounter,
		AtkCounterRounds: atkRounds, DefCounterRounds: defRounds,
		AtkCamp: snap.AtkCamp, DefCamp: snap.DefCamp,
		Round: snap.Round, Done: snap.Done, AttackerWin: snap.AttackerWin, Draw: snap.Draw,
		Head: snap.Head, Actions: snap.Actions,
	}
	rebuild := func(list []ezfyBattleUnitSnap) []*ezfyFightUnit {
		out := []*ezfyFightUnit{}
		for _, s := range list {
			// 兵种属性直接取快照（不查配置表），保证中途改配置也不影响本场
			out = append(out, &ezfyFightUnit{
				id: s.ID, count: s.Count, initialCount: s.InitialCount, pos: s.Pos,
				cfg: &ezfyTroopStats{
					ID: s.TroopId, Name: s.Name, Type: s.Type, Health: s.Health,
					AtkSea: s.AtkSea, AtkGround: s.AtkGround, AtkAir: s.AtkAir,
					Defence: s.Defence, Speed: s.Speed, AttackRange: s.AttackRange,
				},
			})
		}
		return out
	}
	st.Attackers = rebuild(snap.Attackers)
	st.Defenders = rebuild(snap.Defenders)
	if st.Head == nil {
		st.Head = []string{}
	}
	if st.Actions == nil {
		st.Actions = []string{}
	}
	return st
}

// ezfyBattleSnapshotEncode / Decode 快照与 JSON 的互转（存 ezfy_battle.state）
func ezfyBattleSnapshotEncode(snap ezfyBattleSnapshot) string {
	b, err := json.Marshal(snap)
	if err != nil {
		return ""
	}
	return string(b)
}

func ezfyBattleSnapshotDecode(s string) (ezfyBattleSnapshot, bool) {
	snap := ezfyBattleSnapshot{}
	if strings.TrimSpace(s) == "" {
		return snap, false
	}
	if err := json.Unmarshal([]byte(s), &snap); err != nil {
		return snap, false
	}
	return snap, true
}

func ezfyAliveList(list []*ezfyFightUnit) []*ezfyFightUnit {
	alive := []*ezfyFightUnit{}
	for _, u := range list {
		if u.alive() {
			alive = append(alive, u)
		}
	}
	return alive
}

func ezfyContains(list []*ezfyFightUnit, u *ezfyFightUnit) bool {
	for _, x := range list {
		if x == u {
			return true
		}
	}
	return false
}

// ezfyPickTarget 选择攻击目标：**优先兵种**（取该兵种里最近的），没有则取最近目标。
//
// ★★ 2026-09-23 修复（「指挥战场时兵种目标带过来…敌对没有目标则默认攻击距离最近的」）：
//
//	老实现把「全局最近距离」先算进 minDist，再拿配置兵种的距离去比 `d < minDist` ——
//	配置兵种的距离**永远不可能小于全局最小值**（顶多相等，而相等也不满足 `<`），
//	所以「司令部 → 兵种战斗配置 → 优先攻击目标」**实际上完全不生效**，永远是打最近的。
//	现在改成：配了优先兵种且敌方**还有该兵种存活** → 取该兵种里最近的；
//	否则（没配 / 该兵种已全灭）→ 回落全局最近，正是用户要的语义。
//
// ⚠️ 这是**平衡性改动**：修好后「优先攻击目标」会真的生效 ——
// 配了某兵种的单位会去追那个兵种，哪怕近处还有别的敌人（打不打得到看射程）。
func ezfyPickTarget(unit *ezfyFightUnit, enemies []*ezfyFightUnit, targetMap map[int]int) *ezfyFightUnit {
	targetTroop := 0
	if targetMap != nil {
		targetTroop = targetMap[unit.cfg.ID]
	}

	// ① 配了优先兵种 → 在该兵种里取最近的（找不到 = 该兵种已全灭 → 走 ②）
	if targetTroop > 0 {
		pref := (*ezfyFightUnit)(nil)
		prefDist := int(^uint(0) >> 1)
		for _, e := range enemies {
			if e.cfg.ID != targetTroop {
				continue
			}
			if d := ezfyAbs(e.pos - unit.pos); d < prefDist {
				prefDist = d
				pref = e
			}
		}
		if pref != nil {
			return pref
		}
	}

	// ② 默认：攻击距离最近的
	best := (*ezfyFightUnit)(nil)
	minDist := int(^uint(0) >> 1)
	for _, e := range enemies {
		if d := ezfyAbs(e.pos - unit.pos); d < minDist {
			minDist = d
			best = e
		}
	}
	return best
}

// ezfyPickAttack 按守方兵种类型选攻击属性: 1海军 2陆军 3空军, 城防取对地/对海较大值
func ezfyPickAttack(atk, def *ezfyTroopStats) int {
	switch def.Type {
	case 1:
		return atk.AtkSea
	case 2:
		return atk.AtkGround
	case 3:
		return atk.AtkAir
	default:
		return maxInt(atk.AtkGround, atk.AtkSea)
	}
}

// ezfyCalcDamage 伤害 = (攻*5+1000)/(守防*(100+防加成%)*3+10)*数量, 再乘攻加成; 最小伤害=数量*5
func ezfyCalcDamage(baseAtk, def int, count int64, atkBonus, defBonus int) int64 {
	bonusDef := def * (100 + defBonus) / 100
	damage := int64(baseAtk*5+1000) / int64(bonusDef*3+10) * count
	damage = damage * int64(100+atkBonus) / 100
	minimum := count * 5
	if damage < minimum {
		return minimum
	}
	return damage
}

func ezfyToGroups(units []*ezfyFightUnit, losses bool) []ezfyUnitGroup {
	groups := []ezfyUnitGroup{}
	for _, u := range units {
		count := u.count
		if losses {
			count = u.initialCount - u.count
		}
		if count > 0 {
			groups = append(groups, ezfyUnitGroup{TroopId: u.cfg.ID, Count: count})
		}
	}
	return groups
}

// ezfyDefLossGroups 守方损失**全量保序**输出（与传入守方部队逐项对齐，可为 0）。
//
// ★ 2026-10-02 盟军驻军改造：
//
//	守方部队 = 盟友驻军队列(先) + 友军自身部队(后)。战后要把损失按来源分扣
//	（先扣驻军订单、再扣城市），必须保证 br.DefenderLosses[i] 与构建守方部队时的
//	defender[i] 严格一一对应。ezfyToGroups(losses=true) 会跳过无损失的 group，
//	顺序对不上；这里不过滤 count==0，让调用方按下标消费（消费处均已有 count<=0 跳过）。
func ezfyDefLossGroups(units []*ezfyFightUnit) []ezfyUnitGroup {
	groups := make([]ezfyUnitGroup, 0, len(units))
	for _, u := range units {
		groups = append(groups, ezfyUnitGroup{TroopId: u.cfg.ID, Count: u.initialCount - u.count})
	}
	return groups
}

// ezfyEquipBonusDesc 把装备六项加成写成战报里的一行
func ezfyEquipBonusDesc(b ezfyBattleBonus) string {
	parts := []string{}
	if b.Dmg != 0 {
		parts = append(parts, fmt.Sprintf("伤害+%d%%", b.Dmg))
	}
	if b.Def != 0 {
		parts = append(parts, fmt.Sprintf("防御+%d%%", b.Def))
	}
	if b.Hp != 0 {
		parts = append(parts, fmt.Sprintf("生命+%d%%", b.Hp))
	}
	if b.Move != 0 {
		parts = append(parts, fmt.Sprintf("移动距离+%d%%", b.Move))
	}
	if b.Crit != 0 {
		parts = append(parts, fmt.Sprintf("暴击几率+%d%%", b.Crit))
	}
	if b.CritDmg != 0 {
		parts = append(parts, fmt.Sprintf("暴击伤害+%d%%", b.CritDmg))
	}
	if len(parts) == 0 {
		return "无"
	}
	return strings.Join(parts, "，")
}

// ezfyBonusItemsDesc 把逐项明细（科技/技能）拼成「名+X% 名+Y%」；跳过非正值。
// 对科技额外补齐「（效果描述）」，让玩家一眼看出该科技起什么用（对应 ezfy_cfg 科技表 Effect）。
func ezfyBonusItemsDesc(items []ezfyBonusItem) string {
	parts := []string{}
	for _, it := range items {
		if it.Value <= 0 {
			continue
		}
		s := fmt.Sprintf("%s+%d%%", it.Name, it.Value)
		// 防御科技逐项名可能带「科技·」前缀，剥离后再查效果
		key := strings.TrimPrefix(it.Name, "科技·")
		if eff, ok := ezfyTechEffectDesc[key]; ok {
			s += "（" + eff + "）"
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, " ")
}

// ezfyTechEffectDesc 科技名 → 作用说明（取自科技配置表 Effect 的「功能」部分，去掉每级百分比）。
// 只为战报科技行展示用；未收录的科技不追加描述。
var ezfyTechEffectDesc = map[string]string{
	"军训艺术": "部队攻击", "武器科技": "部队攻击", "装甲科技": "部队防御",
	"弹道学": "射程加成", "重工技术": "重装备攻防", "燃烧引擎": "部队速度",
	"掩体防御": "城防攻防", "指挥艺术": "携带上限", "侦察技巧": "情报",
	"喷气引擎": "空军速度", "装载技术": "部队负重", "补给技巧": "军队耗粮",
	"掠夺技巧": "掠夺资源", "治愈伤兵": "伤兵恢复", "储存技术": "资源容量",
}
