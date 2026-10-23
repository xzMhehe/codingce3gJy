package ezfy

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/model"
)

// ============ 名将组合技能（2026-10-23） ============
//
// 「组合技能」是**系统专属**的一类战法：普通军官（kind=1）学不到，
// 只能由管理端把池子（ezfy_cfg_general_skill）里的技能配给名将
// （ezfy_cfg_general.combo_skill_id）。参考率土之滨的「每回合按概率发动」，
// 但只取这一层，不引入拆解/连携，避免玩法过重。
//
// 三条硬规则（详见仓库根目录 `1022名将组合技能.md`）：
//  1. **通用**：效果对全体兵种生效（陆/海/空不区分），与战争系统的
//     攻击/防御/射程/速度/反击/治疗/连击 共用同一套参数，不做第二套数值体系；
//  2. **按等级成长**：组合技等级由**军官等级**分档（默认每 35 级 +1，
//     350 级满级 = 10 级），效果值 = BaseValue + (等级-1) × GrowPerLevel；
//     ★ 2026-10-23 用户纠正：名将满级 350 级、所有名将都是 5 星，
//       拿星级当等级没有成长空间，改按军官等级；
//     ★ 同日二次调整：等级上限 5 → **10**（步长 70 → 35），成长更细，
//       满级数值保持不变（下调 BaseValue 抵消，见 seed 表）。
//  3. **概率发动**：每回合开始按 ProcPct 掷一次，命中后持续 Rounds 回合（1 = 只本回合）。
//
// ⚠️ 数值克制：单技能满级不超过 ~25%（连击/反击按次数），
// 避免「一个组合技顶一套装备」。所有数值都走配置表，管理端可调，代码里不写死。

const (
	ezfyComboAtk     = 1 // 攻击加成%
	ezfyComboDef     = 2 // 防御加成%
	ezfyComboRange   = 3 // 射程加成%
	ezfyComboSpeed   = 4 // 速度加成%
	ezfyComboReduce  = 5 // 受到伤害减免%
	ezfyComboBreak   = 6 // 破防%（无视目标 X% 防御）
	ezfyComboHeal    = 7 // 回合治疗%（恢复已损兵力 X%）
	ezfyComboDouble  = 8 // 连击（追加一次攻击，伤害为原来的 X%）
	ezfyComboCounter = 9 // 反击回合 +N
)

// ezfyComboKindName 效果类型名（管理端下拉 / 玩家端展示共用）
func ezfyComboKindName(kind int) string {
	switch kind {
	case ezfyComboAtk:
		return "攻击加成"
	case ezfyComboDef:
		return "防御加成"
	case ezfyComboRange:
		return "射程加成"
	case ezfyComboSpeed:
		return "速度加成"
	case ezfyComboReduce:
		return "受伤减免"
	case ezfyComboBreak:
		return "破防"
	case ezfyComboHeal:
		return "回合治疗"
	case ezfyComboDouble:
		return "连击"
	case ezfyComboCounter:
		return "反击回合"
	}
	return "未知"
}

// ezfyComboSkill 战斗里用的「已定级」组合技（战场快照要能序列化，故全部导出+json tag）
type ezfyComboSkill struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Kind    int    `json:"kind"`
	Level   int    `json:"level"`
	Value   int    `json:"value"`    // 当前等级效果值（含义随 Kind）
	ProcPct int    `json:"proc_pct"` // 每回合发动概率%
	Rounds  int    `json:"rounds"`   // 发动后持续回合
}

// ezfyComboValue 组合技某等级的效果值 = base + (lv-1)×grow，夹在 [1, 上限] 内。
//
// ★ 连击/反击这类「次数」型效果也用同一公式，只是 base/grow 的量级不同（1~5）。
func ezfyComboValue(base, grow, level, maxLevel int) int {
	if level < 1 {
		level = 1
	}
	if maxLevel < 1 {
		maxLevel = 10
	}
	if level > maxLevel {
		level = maxLevel
	}
	v := base + (level-1)*grow
	if v < 1 {
		v = 1
	}
	return v
}

// ezfyComboLevelOf 组合技等级：按**军官等级**分档。
//
//	默认 step=35 / 上限 10：1~35 级=1 级 … 316~350 级=10 级。
//	名将满级 350 → 正好吃满 10 级（与「满级 350」的成长线对齐）。
//	step / maxLevel 都走配置，管理端可把 step 调大让成长更慢（70 → 5 级封顶）。
func ezfyComboLevelOf(officerLevel, maxLevel, step int) int {
	if maxLevel < 1 {
		maxLevel = 10
	}
	if step < 1 {
		step = 35
	}
	if officerLevel < 1 {
		officerLevel = 1
	}
	lv := 1 + (officerLevel-1)/step
	if lv > maxLevel {
		lv = maxLevel
	}
	return lv
}

// ezfyComboOfGeneralCfg 由「名将池配置 + 该军官的等级」生成战斗用组合技（nil = 没配 / 已下架）
//
// officerLevel 传**军官实例等级**（名将满级 350）；AI 守将没有实例时传池子配置的 Level。
func ezfyComboOfGeneralCfg(g *model.EzfyCfgGeneral, officerLevel int) *ezfyComboSkill {
	if g == nil || g.Kind != 2 || g.ComboSkillId <= 0 {
		return nil
	}
	s := ezfyCfg.generalSkill(g.ComboSkillId)
	if s == nil || s.Enabled == 0 {
		return nil
	}
	maxLv := s.MaxLevel
	if maxLv < 1 {
		maxLv = 10
	}
	lv := ezfyComboLevelOf(officerLevel, maxLv, s.LevelPerStep)
	rounds := s.Rounds
	if rounds < 1 {
		rounds = 1
	}
	proc := s.ProcPct
	if proc < 0 {
		proc = 0
	}
	if proc > 100 {
		proc = 100
	}
	return &ezfyComboSkill{
		ID: s.ID, Name: s.Name, Kind: s.Kind, Level: lv,
		Value:   ezfyComboValue(s.BaseValue, s.GrowPerLevel, lv, maxLv),
		ProcPct: proc, Rounds: rounds,
	}
}

// comboSkillOfOfficer 军官实例（名将）的组合技；普通军官 / 非名将一律 nil。
func (h *EzfyHandler) comboSkillOfOfficer(o *model.EzfyOfficer) *ezfyComboSkill {
	if o == nil || o.GeneralId <= 0 {
		return nil
	}
	g := ezfyCfg.general(o.GeneralId)
	return ezfyComboOfGeneralCfg(g, o.Level) // ★ 按军官等级分档，不看星级
}

// ezfyComboTxt 组合技一行文案（战报开局行 / 玩家端详情共用）
func ezfyComboTxt(c *ezfyComboSkill) string {
	if c == nil {
		return ""
	}
	return fmt.Sprintf("%s Lv.%d（%s %s，发动率%d%%）", c.Name, c.Level, ezfyComboKindName(c.Kind), ezfyComboValueTxt(c), c.ProcPct)
}

// ezfyComboValueTxt 效果值的可读文案（按类型区分单位）
func ezfyComboValueTxt(c *ezfyComboSkill) string {
	if c == nil {
		return ""
	}
	switch c.Kind {
	case ezfyComboCounter:
		return fmt.Sprintf("反击回合+%d", c.Value)
	case ezfyComboHeal:
		return fmt.Sprintf("恢复已损兵力%d%%", c.Value)
	case ezfyComboDouble:
		return fmt.Sprintf("追加攻击(伤害%d%%)", c.Value)
	case ezfyComboBreak:
		return fmt.Sprintf("无视防御%d%%", c.Value)
	case ezfyComboReduce:
		return fmt.Sprintf("受伤-%d%%", c.Value)
	default:
		return fmt.Sprintf("+%d%%", c.Value)
	}
}

// ezfyComboShortTxt 短效果文案（准备回合「军官行」内联用，格式对齐军官技能）
//
//	例：尖兵突击(Lv.6 攻击力+180%) 铁血军魂(Lv.10 攻击力+180%，发动40%)
func ezfyComboShortTxt(c *ezfyComboSkill) string {
	if c == nil {
		return ""
	}
	return fmt.Sprintf("%s(Lv.%d %s，发动%d%%)", c.Name, c.Level, ezfyComboShortEffect(c), c.ProcPct)
}

// ezfyComboShortEffect 只有效果本身，不带技能名（军官行 / 播报共用）
func ezfyComboShortEffect(c *ezfyComboSkill) string {
	if c == nil {
		return ""
	}
	switch c.Kind {
	case ezfyComboAtk:
		return fmt.Sprintf("攻击力+%d%%", c.Value)
	case ezfyComboDef:
		return fmt.Sprintf("防御力+%d%%", c.Value)
	case ezfyComboRange:
		return fmt.Sprintf("射程+%d%%", c.Value)
	case ezfyComboSpeed:
		return fmt.Sprintf("移动速度+%d%%", c.Value)
	case ezfyComboReduce:
		return fmt.Sprintf("受到伤害-%d%%", c.Value)
	case ezfyComboBreak:
		return fmt.Sprintf("破防%d%%", c.Value)
	case ezfyComboHeal:
		return fmt.Sprintf("回合治疗%d%%", c.Value)
	case ezfyComboDouble:
		return fmt.Sprintf("连击伤害%d%%", c.Value)
	case ezfyComboCounter:
		return fmt.Sprintf("反击回合+%d", c.Value)
	}
	return fmt.Sprintf("+%d%%", c.Value)
}

// ezfyComboProcTxt 发动播报用的效果描述（自带主语，玩家不看类型名也懂）
func ezfyComboProcTxt(c *ezfyComboSkill) string {
	if c == nil {
		return ""
	}
	switch c.Kind {
	case ezfyComboAtk:
		return fmt.Sprintf("全体攻击力+%d%%", c.Value)
	case ezfyComboDef:
		return fmt.Sprintf("全体防御力+%d%%", c.Value)
	case ezfyComboRange:
		return fmt.Sprintf("全体射程+%d%%", c.Value)
	case ezfyComboSpeed:
		return fmt.Sprintf("全体移动速度+%d%%", c.Value)
	case ezfyComboReduce:
		return fmt.Sprintf("受到伤害-%d%%", c.Value)
	case ezfyComboBreak:
		return fmt.Sprintf("攻击无视目标%d%%防御", c.Value)
	case ezfyComboHeal:
		return fmt.Sprintf("回合结束恢复已损兵力%d%%", c.Value)
	case ezfyComboDouble:
		return fmt.Sprintf("追加一次攻击（伤害%d%%）", c.Value)
	case ezfyComboCounter:
		return fmt.Sprintf("反击回合+%d", c.Value)
	}
	return ezfyComboValueTxt(c)
}

// comboRoll 每回合开始掷骰：到期的效果自然失效，没生效的按概率尝试发动。
//
// 返回本回合是否生效（供加成计算使用）。发动时写一行战报播报，
// 让玩家一眼看到「这回合为什么多了加成」。
func (st *ezfyBattleState) comboRoll() {
	roll := func(c *ezfyComboSkill, left *int, side, offName string, used *bool) {
		if c == nil {
			return
		}
		if *left > 0 {
			return // 还在持续期内，不重复掷
		}
		if c.ProcPct <= 0 || rand.Intn(100) >= c.ProcPct {
			return
		}
		*left = c.Rounds
		if *left < 1 {
			*left = 1
		}
		// 反击回合是「一次性永久加成」，整场只加一次，避免每回合叠上去
		if c.Kind == ezfyComboCounter && used != nil && !*used {
			*used = true
			if side == "【攻方】" {
				st.AtkCounterRounds += c.Value
				st.AtkCounter = true
			} else {
				st.DefCounterRounds += c.Value
				st.DefCounter = true
			}
		}
		idx := (st.Round*3 + len(st.Actions)) % len(ezfySlogans)
		st.Actions = append(st.Actions, fmt.Sprintf("%s%s发动组合技【%s】！%s，%s",
			side, offName, c.Name, ezfySlogans[idx], ezfyComboProcTxt(c)))
	}
	roll(st.AtkCombo, &st.AtkComboLeft, "【攻方】", ezfyOfficerShortName(st.AtkOfficerDesc), &st.AtkComboUsed)
	roll(st.DefCombo, &st.DefComboLeft, "【守方】", ezfyOfficerShortName(st.DefOfficerDesc), &st.DefComboUsed)
}

// comboActive 某一方本回合的组合技是否生效，以及生效的技能（nil = 没生效）
func comboActive(c *ezfyComboSkill, left int) *ezfyComboSkill {
	if c == nil || left <= 0 {
		return nil
	}
	return c
}

// comboOfKind 本回合生效的技能是否是指定类型（不是该类型返回 0）
func comboOfKind(c *ezfyComboSkill, kind int) int {
	if c == nil || c.Kind != kind {
		return 0
	}
	return c.Value
}

// comboEndRound 回合结束：治疗结算 + 生效回合数递减。
//
// 治疗放在回合结束，用「本回合损失」为基数（回合中单位可能已被打光，
// 只要还有存活单位就按各自已损比例回补），符合「回合治疗」的直觉。
func (st *ezfyBattleState) comboEndRound(atkActive, defActive *ezfyComboSkill) {
	if pct := comboOfKind(atkActive, ezfyComboHeal); pct > 0 {
		if n := ezfyComboHealSide(st.Attackers, pct); n > 0 {
			st.Actions = append(st.Actions, fmt.Sprintf("【攻方】%s组合技【%s】生效，治疗兵力%d",
				ezfyOfficerShortName(st.AtkOfficerDesc), atkActive.Name, n))
		}
	}
	if pct := comboOfKind(defActive, ezfyComboHeal); pct > 0 {
		if n := ezfyComboHealSide(st.Defenders, pct); n > 0 {
			st.Actions = append(st.Actions, fmt.Sprintf("【守方】%s组合技【%s】生效，治疗兵力%d",
				ezfyOfficerShortName(st.DefOfficerDesc), defActive.Name, n))
		}
	}
	if st.AtkComboLeft > 0 {
		st.AtkComboLeft--
	}
	if st.DefComboLeft > 0 {
		st.DefComboLeft--
	}
}

// ezfyComboHealSide 按「已损兵力的 pct%」治疗本方全部存活单位，返回总治疗量。
func ezfyComboHealSide(units []*ezfyFightUnit, pct int) int64 {
	if pct <= 0 {
		return 0
	}
	var total int64
	for _, u := range units {
		if u == nil || !u.alive() || u.initialCount <= u.count {
			continue
		}
		lost := u.initialCount - u.count
		heal := lost * int64(pct) / 100
		if heal < 1 {
			heal = 1 // 有损失至少回 1 个，避免「发动了但什么都没发生」
		}
		if heal > lost {
			heal = lost
		}
		u.count += heal
		total += heal
	}
	return total
}

// comboFollowUp 连击：主攻击结束后追加一次攻击（伤害 = 主攻击的 Value%）。
//
// 目标沿用本次主攻击的目标；若它已被打光，改打射程内最近的敌人。
// 只做单体追加（不做「势不可挡」式溢出级联），避免一次连击把对面全清。
func (st *ezfyBattleState) comboFollowUp(unit, target *ezfyFightUnit, enemies []*ezfyFightUnit,
	rangeD, atkFlat, atkBonus, defBonus, defBreakPct int, side, enemySide string,
	nameOf func(*ezfyFightUnit) string, skillName string, dmgPct int) {
	if unit == nil || !unit.alive() || dmgPct <= 0 {
		return
	}
	cur := target
	if cur == nil || !cur.alive() {
		inRange := make([]*ezfyFightUnit, 0, len(enemies))
		for _, e := range ezfyAliveList(enemies) {
			if ezfyAbs(e.pos-unit.pos) <= rangeD {
				inRange = append(inRange, e)
			}
		}
		if len(inRange) == 0 {
			return
		}
		cur = ezfyPickTarget(unit, inRange, nil)
		if cur == nil {
			return
		}
	}
	def := cur.cfg.Defence
	if defBreakPct > 0 {
		def = def * (100 - defBreakPct) / 100
		if def < 1 {
			def = 1
		}
	}
	damage := ezfyCalcDamage(ezfyPickAttack(unit.cfg, cur.cfg)+atkFlat, def, unit.count, atkBonus, defBonus)
	damage = damage * int64(dmgPct) / 100
	if damage < 1 {
		damage = 1
	}
	chp := cur.hp
	if chp < 1 {
		chp = 1
	}
	killed := damage / int64(chp)
	if killed < 1 {
		killed = 1
	}
	if killed > cur.count {
		killed = cur.count
	}
	cur.count -= killed
	st.Actions = append(st.Actions, fmt.Sprintf("%s%s发动组合技【%s】连击%s%s！%s损失%d",
		side, nameOf(unit), skillName, enemySide, nameOf(cur), enemySide, killed))
}

// ezfyComboDes 把池子里的说明模板套上当前等级的效果值。
//
// 种子里的 Des 用 `%d` 占位（可能一处也可能两处，如「破防%d%%（…无视 %d%% 防御）」），
// 管理端自己改文案时**不带占位符也照样能用**（原样返回）。
func ezfyComboDes(tpl string, value int) string {
	if tpl == "" {
		return ""
	}
	n := strings.Count(tpl, "%d")
	switch n {
	case 0:
		return tpl
	case 1:
		return fmt.Sprintf(tpl, value)
	default:
		return fmt.Sprintf(tpl, value, value)
	}
}

// ezfyComboBrief 组合技的对外结构（nil → nil，前端据此不渲染）
func ezfyComboBrief(c *ezfyComboSkill) interface{} {
	if c == nil {
		return nil
	}
	s := ezfyCfg.generalSkill(c.ID)
	des := ""
	if s != nil {
		des = ezfyComboDes(s.Des, c.Value)
	}
	return gin.H{
		"id": c.ID, "name": c.Name, "kind": c.Kind, "kind_name": ezfyComboKindName(c.Kind),
		"level": c.Level, "value": c.Value, "value_text": ezfyComboValueTxt(c),
		"proc_pct": c.ProcPct, "rounds": c.Rounds, "des": des,
	}
}

// ezfyComboBriefAtMax 组合技的**满级**展示（名将图鉴用）。
//
// 图鉴是「这个技能满级多强」的预览，不该拿池子配置的 level 去定级
// （池里 level 是「等级上限」，本地库甚至可能填 1 → 图鉴会显示成 Lv.1 很误导）。
func ezfyComboBriefAtMax(g *model.EzfyCfgGeneral) interface{} {
	if g == nil || g.Kind != 2 || g.ComboSkillId <= 0 {
		return nil
	}
	// 传一个足够大的等级 → 必被夹到 max_level，即「满级」
	return ezfyComboBrief(ezfyComboOfGeneralCfg(g, 1<<20))
}

// officerComboView 军官详情里的组合技（普通军官恒 nil）
func (h *EzfyHandler) officerComboView(o *model.EzfyOfficer) interface{} {
	return ezfyComboBrief(h.comboSkillOfOfficer(o))
}

// appendComboHead 战报「准备回合」展示组合技。
//
// ★ 2026-10-23 用户反馈：原来单独一行「【攻方组合技】…」被追加在最后（排在
// 「战场初始相距6000」之后），既像补充说明、又和军官技能割裂。改成**并进军官行**，
// 格式对齐军官技能：
//
//	【攻方军官】斯大林 Lv.350 攻击加成+477% 防御加成+253% 尖兵突击(Lv.6 攻击力+180%) …
//	              └→ 追加：铁血军魂(Lv.10 攻击力+180%，发动40%)
//
// 没有军官行（AI 守将 / 老战场）时退回单独一行，插在「准备回合:」之后。
func (st *ezfyBattleState) appendComboHead() {
	merge := func(side, txt string) bool {
		if txt == "" {
			return true
		}
		prefix := "【" + side + "军官】"
		for i, l := range st.Head {
			if strings.HasPrefix(l, prefix) {
				st.Head[i] = l + " " + txt
				return true
			}
		}
		return false
	}
	atkTxt, defTxt := ezfyComboShortTxt(st.AtkCombo), ezfyComboShortTxt(st.DefCombo)
	okA, okD := merge("攻方", atkTxt), merge("守方", defTxt)
	// 没有军官行 → 退回单独一行，按「攻方 → 守方」顺序插到「准备回合:」后面
	extra := []string{}
	if !okA && atkTxt != "" {
		extra = append(extra, "【攻方组合技】"+atkTxt)
	}
	if !okD && defTxt != "" {
		extra = append(extra, "【守方组合技】"+defTxt)
	}
	if len(extra) == 0 {
		return
	}
	idx := 1
	if len(st.Head) == 0 {
		idx = 0
	}
	out := make([]string, 0, len(st.Head)+len(extra))
	out = append(out, st.Head[:idx]...)
	out = append(out, extra...)
	out = append(out, st.Head[idx:]...)
	st.Head = out
}
