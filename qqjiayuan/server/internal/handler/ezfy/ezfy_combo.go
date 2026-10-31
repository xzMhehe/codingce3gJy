package ezfy

import (
	"encoding/json"
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
// （ezfy_cfg_general.combo_skill_ids，JSON 数组，★ 2026-10-24 一个名将可配多个）。
// 参考率土之滨的「每回合按概率发动」，
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
	// ★★ 2026-10-31 追击（配合战法类型 4）：每次普攻命中后掷骰，命中则对被攻击目标追加一次 Value% 伤害。
	//   与「连击(8)」的区别：连击是**回合级**（本回合每次攻击都追加），追击是**每次攻击独立掷骰**。
	ezfyComboPursue = 10 // 追击（追加一次攻击，伤害为原来的 X%）
)

// 战法类型（★ 2026-10-31 新增，参考率土之滨的四类战法；列 ezfy_cfg_general_skill.trigger_type）
const (
	ezfyComboTriggerPassive = 1 // 被动：必定发动，整场常驻
	ezfyComboTriggerCommand = 2 // 指挥：战前掷一次骰，命中后持续 Rounds 回合（0=整场）
	ezfyComboTriggerActive  = 3 // 主动：每回合开始掷骰（原有行为）
	ezfyComboTriggerPursuit = 4 // 追击：每次普攻命中后掷骰
)

// ezfyComboWholeBattle 「整场」的剩余回合数哨兵（被动 / Rounds=0 的指挥）。
const ezfyComboWholeBattle = 999

// ezfyComboTriggerName 战法类型名（管理端下拉 / 战报播报共用）
func ezfyComboTriggerName(t int) string {
	switch t {
	case ezfyComboTriggerPassive:
		return "被动"
	case ezfyComboTriggerCommand:
		return "指挥"
	case ezfyComboTriggerPursuit:
		return "追击"
	}
	return "主动"
}

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
	case ezfyComboPursue:
		return "追击"
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
	ProcPct int    `json:"proc_pct"` // 发动概率%（含义随 TriggerType）
	Rounds  int    `json:"rounds"`   // 发动后持续回合
	// ★★ 2026-10-31 战法类型：1被动 2指挥 3主动 4追击（见 ezfyComboTrigger* 常量）
	TriggerType int `json:"trigger_type"`
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

// ezfyComboOfGeneralCfg 由「名将池配置 + 该军官的等级」生成战斗用组合技列表
// （nil / 空 = 没配或已下架）。★ 2026-10-24 支持**一个名将配多个技能**（用户要求），
// 每个技能独立定级、独立掷骰；普通军官（kind=1）即便误配也不生效。
//
// officerLevel 传**军官实例等级**（名将满级 350）；AI 守将没有实例时传池子配置的 Level。
func ezfyComboOfGeneralCfg(g *model.EzfyCfgGeneral, officerLevel int) []*ezfyComboSkill {
	if g == nil || g.Kind != 2 {
		return nil
	}
	out := []*ezfyComboSkill{}
	for _, id := range comboSkillIdsOfCfg(g) {
		s := ezfyCfg.generalSkill(id)
		if s == nil || s.Enabled == 0 {
			continue
		}
		maxLv := s.MaxLevel
		if maxLv < 1 {
			maxLv = 10
		}
		lv := ezfyComboLevelOf(officerLevel, maxLv, s.LevelPerStep)
		trig := s.TriggerType
		if trig < 1 || trig > 4 {
			trig = ezfyComboTriggerActive // 老数据（列默认 0）按主动处理，行为与改动前一致
		}
		rounds := s.Rounds
		// 主动默认持续 1 回合；被动/指挥的 Rounds=0 表示**整场**（不能被打回 1）
		if rounds < 1 && trig == ezfyComboTriggerActive {
			rounds = 1
		}
		proc := s.ProcPct
		if proc < 0 {
			proc = 0
		}
		if proc > 100 {
			proc = 100
		}
		out = append(out, &ezfyComboSkill{
			ID: s.ID, Name: s.Name, Kind: s.Kind, Level: lv,
			Value:   ezfyComboValue(s.BaseValue, s.GrowPerLevel, lv, maxLv),
			ProcPct: proc, Rounds: rounds, TriggerType: trig,
		})
	}
	return out
}

// comboSkillIdsOfCfg 名将池配置 → 组合技池 ID 列表。
//
// ★ 2026-10-24 新字段 combo_skill_ids（JSON 数组）优先；旧库只有 combo_skill_id
// （单技能时代）→ 回退成 [combo_skill_id]，保证老数据无缝升级、管理端保存前不丢技能。
func comboSkillIdsOfCfg(g *model.EzfyCfgGeneral) []int {
	if g == nil {
		return nil
	}
	ids := []int{}
	if s := strings.TrimSpace(g.ComboSkillIds); s != "" && s != "[]" {
		if err := json.Unmarshal([]byte(s), &ids); err != nil {
			ids = nil // 非法 JSON → 当没配
		}
	}
	if len(ids) == 0 && g.ComboSkillId > 0 {
		ids = []int{g.ComboSkillId}
	}
	return ids
}

// comboSkillIdsTxt 技能 ID 列表 → 落库 JSON 字符串（管理端保存用）。
func comboSkillIdsTxt(ids []int) string {
	if len(ids) == 0 {
		return ""
	}
	b, _ := json.Marshal(ids)
	return string(b)
}

// comboSkillIdsAdd 给列表追加一个技能 ID（去重）。
func comboSkillIdsAdd(ids []int, id int) []int {
	for _, v := range ids {
		if v == id {
			return ids
		}
	}
	return append(ids, id)
}

// comboSkillIdsRemove 从列表移除一个技能 ID。
func comboSkillIdsRemove(ids []int, id int) []int {
	out := []int{}
	for _, v := range ids {
		if v != id {
			out = append(out, v)
		}
	}
	return out
}

// comboSkillsOfOfficer 军官实例（名将）的组合技列表；普通军官 / 非名将一律空。
func (h *EzfyHandler) comboSkillsOfOfficer(o *model.EzfyOfficer) []*ezfyComboSkill {
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
	// ★★ 2026-10-31 战法类型进文案（被动不写发动率）
	switch c.TriggerType {
	case ezfyComboTriggerPassive:
		return fmt.Sprintf("%s %s（被动 %s）", c.Name, ezfyRomanLevel(c.Level), ezfyComboValueTxt(c))
	case ezfyComboTriggerCommand:
		return fmt.Sprintf("%s %s（指挥 %s，战前 %d%%）", c.Name, ezfyRomanLevel(c.Level), ezfyComboValueTxt(c), c.ProcPct)
	case ezfyComboTriggerPursuit:
		return fmt.Sprintf("%s %s（追击 %s，%d%%）", c.Name, ezfyRomanLevel(c.Level), ezfyComboValueTxt(c), c.ProcPct)
	}
	return fmt.Sprintf("%s %s（%s %s，发动率%d%%）", c.Name, ezfyRomanLevel(c.Level), ezfyComboKindName(c.Kind), ezfyComboValueTxt(c), c.ProcPct)
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
	case ezfyComboPursue:
		return fmt.Sprintf("追击伤害%d%%", c.Value)
	case ezfyComboBreak:
		return fmt.Sprintf("无视防御%d%%", c.Value)
	case ezfyComboReduce:
		return fmt.Sprintf("受伤-%d%%", c.Value)
	default:
		return fmt.Sprintf("+%d%%", c.Value)
	}
}

// ezfyComboEffectTxt 玩家端「特殊技能：」一行的效果文案。
//
// ★ 2026-10-25 用户要求格式统一成「加号 + 属性名 + 数值」：
//
//	特殊技能：[装甲突击]：+攻击力180%
//
// （原来是 `+180%` 再单起一行灰色括号说明「攻击力+180%（发动后…）」，信息重复又占行。）
// 与 ezfyComboShortEffect 的区别只在前缀/单位位置，别混用：
//   - ezfyComboShortEffect：`攻击力+180%`（战报军官行内联，句子中间读）
//   - ezfyComboEffectTxt  ：`+攻击力180%`（详情页独立一行，视觉上以 + 开头）
func ezfyComboEffectTxt(c *ezfyComboSkill) string {
	if c == nil {
		return ""
	}
	switch c.Kind {
	case ezfyComboAtk:
		return fmt.Sprintf("+攻击力%d%%", c.Value)
	case ezfyComboDef:
		return fmt.Sprintf("+防御力%d%%", c.Value)
	case ezfyComboRange:
		return fmt.Sprintf("+射程%d%%", c.Value)
	case ezfyComboSpeed:
		return fmt.Sprintf("+速度%d%%", c.Value)
	case ezfyComboReduce:
		return fmt.Sprintf("+受伤减免%d%%", c.Value)
	case ezfyComboBreak:
		return fmt.Sprintf("+破防%d%%", c.Value)
	case ezfyComboHeal:
		return fmt.Sprintf("+治疗%d%%", c.Value)
	case ezfyComboDouble:
		return fmt.Sprintf("+连击伤害%d%%", c.Value)
	case ezfyComboCounter:
		return fmt.Sprintf("+反击%d回合", c.Value)
	case ezfyComboPursue:
		return fmt.Sprintf("+追击伤害%d%%", c.Value)
	}
	return fmt.Sprintf("+%d%%", c.Value)
}

// ezfyComboShortTxt 短效果文案（准备回合「军官行」内联用，格式对齐军官技能）
//
//	例：尖兵猛攻(VI 攻击力+180%) 装甲突击(X 攻击力+180%，发动40%)
func ezfyComboShortTxt(c *ezfyComboSkill) string {
	if c == nil {
		return ""
	}
	// ★★ 2026-10-31 统一成「名称(N 效果，概率%)」，不再塞「被动/指挥/追击」这类说明文字
	//   （用户要求回合相关描述简约）。被动必定发动 → 不写概率。
	//   ★ 同日用户要求：指挥模块 / 战报回合里也去掉「Lv.」修饰，只留罗马数字等级。
	if c.TriggerType == ezfyComboTriggerPassive {
		return fmt.Sprintf("%s(%s %s)", c.Name, ezfyRomanLevel(c.Level), ezfyComboShortEffect(c))
	}
	return fmt.Sprintf("%s(%s %s，%d%%)", c.Name, ezfyRomanLevel(c.Level), ezfyComboShortEffect(c), c.ProcPct)
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
	case ezfyComboPursue:
		return fmt.Sprintf("追击伤害%d%%", c.Value)
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
		return fmt.Sprintf("全军攻击+%d%%", c.Value)
	case ezfyComboDef:
		return fmt.Sprintf("全军防御+%d%%", c.Value)
	case ezfyComboRange:
		return fmt.Sprintf("全军射程+%d%%", c.Value)
	case ezfyComboSpeed:
		return fmt.Sprintf("全军速度+%d%%", c.Value)
	case ezfyComboReduce:
		return fmt.Sprintf("受到伤害-%d%%", c.Value)
	case ezfyComboBreak:
		return fmt.Sprintf("攻击无视目标%d%%防御", c.Value)
	case ezfyComboHeal:
		return fmt.Sprintf("回合结束恢复已损兵力%d%%", c.Value)
	case ezfyComboDouble:
		return fmt.Sprintf("本回合追加一次普攻（伤害%d%%）", c.Value)
	case ezfyComboCounter:
		return fmt.Sprintf("反击回合+%d", c.Value)
	case ezfyComboPursue:
		return fmt.Sprintf("追击（对本次普攻目标追加%d%%伤害）", c.Value)
	}
	return ezfyComboValueTxt(c)
}

// ezfyComboRuntime 战斗内**单个**组合技的运行时状态（快照要能序列化，全部导出+json tag）。
//
// ★ 2026-10-24 名将可学多个组合技：每个技能一组独立运行时，各自掷骰、各自倒计时。
type ezfyComboRuntime struct {
	Skill *ezfyComboSkill `json:"skill"`
	Left  int             `json:"left"` // 剩余生效回合（0 = 本回合未生效）
	Used  bool            `json:"used"` // 「反击回合+」是否已加过（一次性）
}

// comboRuntimes 把技能列表包成运行时（Left=0 未生效，等每回合掷骰）。
func comboRuntimes(skills []*ezfyComboSkill) []*ezfyComboRuntime {
	out := []*ezfyComboRuntime{}
	for _, s := range skills {
		if s == nil {
			continue
		}
		out = append(out, &ezfyComboRuntime{Skill: s})
	}
	return out
}

// activeCombos 本回合已生效（Left>0）的技能列表。
func activeCombos(rts []*ezfyComboRuntime) []*ezfyComboSkill {
	out := []*ezfyComboSkill{}
	for _, rt := range rts {
		if rt != nil && rt.Left > 0 && rt.Skill != nil {
			out = append(out, rt.Skill)
		}
	}
	return out
}

// comboRoll 每回合开始掷骰：到期的效果自然失效，没生效的按概率尝试发动。
//
// ★ 2026-10-24 多技能：每个技能**独立掷骰**，多个技能可同时发动（播报各一行），
// 让玩家一眼看到「这回合为什么多了加成」。
//
// ★★ 2026-10-31 四类战法（用户需求，参考率土之滨）：
//
//	被动(1)：第 1 回合直接生效、整场常驻，不掷骰（播报一次「常驻生效」）
//	指挥(2)：只在第 1 回合掷一次骰，命中后 Rounds 回合（0 → 整场）
//	主动(3)：每回合开始掷骰（原行为）
//	追击(4)：不在这里处理 —— 每次普攻命中后单独掷骰（见 comboPursuitOf / Step 里的追击段）
func (st *ezfyBattleState) comboRoll() {
	// 反击回合是「一次性永久加成」，整场只加一次（被动/指挥/主动共用）
	applyCounterOnce := func(rt *ezfyComboRuntime, c *ezfyComboSkill, side string) {
		if c.Kind != ezfyComboCounter || rt.Used {
			return
		}
		rt.Used = true
		if side == "【攻方】" {
			st.AtkCounterRounds += c.Value
			st.AtkCounter = true
		} else {
			st.DefCounterRounds += c.Value
			st.DefCounter = true
		}
	}
	anno := func(c *ezfyComboSkill, side, offName string) {
		// ★★ 2026-10-31 用户要求「简约、官方、二战主题」：去口号与感叹号，
		//   统一成 `【攻方】某某发动组合技【装甲突击】，全军攻击+180%`。
		//   ★ 同日再简化：**不再追加括号说明**（「（被动·常驻）」「（指挥·战前生效）」等），
		//     回合内只报「谁发动了什么、加多少」，战法类型看准备回合/详情即可。
		st.Actions = append(st.Actions, fmt.Sprintf("%s%s发动组合技【%s】，%s",
			side, offName, c.Name, ezfyComboProcTxt(c)))
	}
	roll := func(rts []*ezfyComboRuntime, side, offName string) {
		for _, rt := range rts {
			if rt == nil || rt.Skill == nil {
				continue
			}
			c := rt.Skill
			switch c.TriggerType {
			case ezfyComboTriggerPursuit:
				continue // 追击按「每次攻击」掷骰，不占回合层
			case ezfyComboTriggerPassive:
				// 必定发动：第 1 回合起整场常驻，只播报一次
				if rt.Left > 0 {
					continue
				}
				rt.Left = ezfyComboWholeBattle
				applyCounterOnce(rt, c, side)
				anno(c, side, offName)
				continue
			case ezfyComboTriggerCommand:
				// 战前（第 1 回合）掷一次，之后不再掷
				if st.Round > 1 {
					continue
				}
				if c.ProcPct <= 0 || rand.Intn(100) >= c.ProcPct {
					continue // 指挥没生效
				}
				// ★ 率土口径：指挥战法**命中后整场生效**（Rounds 只对主动类有意义）
				rt.Left = ezfyComboWholeBattle
				applyCounterOnce(rt, c, side)
				anno(c, side, offName)
				continue
			}
			// 主动：每回合开始掷骰（原行为）
			if rt.Left > 0 {
				continue // 还在持续期内，不重复掷
			}
			if c.ProcPct <= 0 || rand.Intn(100) >= c.ProcPct {
				continue
			}
			rt.Left = c.Rounds
			if rt.Left < 1 {
				rt.Left = 1
			}
			applyCounterOnce(rt, c, side)
			anno(c, side, offName)
		}
	}
	roll(st.AtkCombos, "【攻方】", ezfyOfficerShortName(st.AtkOfficerDesc))
	roll(st.DefCombos, "【守方】", ezfyOfficerShortName(st.DefOfficerDesc))
}

// comboPursuitOf 该方所有「追击」类技能（**不看 Left**：追击是每次普攻独立掷骰，不按回合常驻）。
//
// 返回按效果值从高到低排序 —— 每次攻击只让**最强的一条**掷骰（避免一次普攻触发好几条追击）。
func comboPursuitOf(rts []*ezfyComboRuntime) []*ezfyComboSkill {
	out := []*ezfyComboSkill{}
	for _, rt := range rts {
		if rt != nil && rt.Skill != nil && rt.Skill.TriggerType == ezfyComboTriggerPursuit {
			out = append(out, rt.Skill)
		}
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].Value > out[i].Value {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

// comboPursuitRoll 追击掷骰：返回本次攻击要触发的技能（没触发 → nil）。
//
// 多个追击技能按效果值从高到低逐个掷，命中即止（同一次普攻最多追加一击）。
func comboPursuitRoll(list []*ezfyComboSkill) *ezfyComboSkill {
	for _, c := range list {
		if c == nil || c.ProcPct <= 0 {
			continue
		}
		if rand.Intn(100) < c.ProcPct {
			return c
		}
	}
	return nil
}

// comboOfKind 本回合生效的技能里，指定类型的效果值合计（多个同类技能**叠加**）。
func comboOfKind(active []*ezfyComboSkill, kind int) int {
	if len(active) == 0 {
		return 0
	}
	sum := 0
	for _, c := range active {
		if c != nil && c.Kind == kind {
			sum += c.Value
		}
	}
	return sum
}

// comboBreakTxt 来源拆解文本：指定类型的生效技能逐项列出（空格分隔）。
//
//	例：组合技·装甲突击+180% 组合技·全线总攻+120%
func comboBreakTxt(active []*ezfyComboSkill, kind int, prefix string) string {
	out := ""
	for _, c := range active {
		if c == nil || c.Kind != kind {
			continue
		}
		out += fmt.Sprintf("%s·%s+%d%% ", prefix, c.Name, c.Value)
	}
	return strings.TrimSpace(out)
}

// comboDoubleBest 连击：多个连击技同时生效时，只取最强的一个触发**一次**追加攻击
// （避免一次主攻击连击好几次把对面清场）。
func comboDoubleBest(active []*ezfyComboSkill) (*ezfyComboSkill, bool) {
	var best *ezfyComboSkill
	for _, c := range active {
		if c == nil || c.Kind != ezfyComboDouble {
			continue
		}
		if best == nil || c.Value > best.Value {
			best = c
		}
	}
	return best, best != nil
}

// comboEndRound 回合结束：治疗结算 + 每个技能的生效回合数递减。
//
// 治疗放在回合结束，用「本回合损失」为基数（回合中单位可能已被打光，
// 只要还有存活单位就按各自已损比例回补），符合「回合治疗」的直觉。
// ★ 2026-10-24 多技能：每个生效中的治疗技独立结算、各播报一行。
func (st *ezfyBattleState) comboEndRound() {
	heal := func(rts []*ezfyComboRuntime, side string, units []*ezfyFightUnit, offDesc string) {
		for _, rt := range rts {
			if rt == nil || rt.Left <= 0 || rt.Skill == nil || rt.Skill.Kind != ezfyComboHeal {
				continue
			}
			if n := ezfyComboHealSide(units, rt.Skill.Value); n > 0 {
				st.Actions = append(st.Actions, fmt.Sprintf("%s%s组合技【%s】生效，治疗兵力%d",
					side, ezfyOfficerShortName(offDesc), rt.Skill.Name, n))
			}
		}
	}
	heal(st.AtkCombos, "【攻方】", st.Attackers, st.AtkOfficerDesc)
	heal(st.DefCombos, "【守方】", st.Defenders, st.DefOfficerDesc)
	for _, rt := range st.AtkCombos {
		if rt != nil && rt.Left > 0 {
			rt.Left--
		}
	}
	for _, rt := range st.DefCombos {
		if rt != nil && rt.Left > 0 {
			rt.Left--
		}
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
//
// ★★ 2026-10-31 用户口径：连击相当于**两次攻击**，第二次攻击要**单独计算暴击**
// （主攻击那次在 Step 里已经各自掷过骰，这里必须再掷一次，不能共用主攻击的结果）。
// critChance / critDmg 由调用方传**攻击方自己的装备暴击属性**；追击是战法伤害，传 0 不暴击。
//
// ★★ 2026-10-31 狼群围猎修复：atkBonusAt/defBonusAt 传**按目标取值**的闭包，
// 不能像原来那样传主目标算好的固定数值 —— 连击把原目标打死后换目标时，
// 「海军对海+75%」之类按目标类型生效的加成必须按**新目标**重新计算（否则打陆军也带 75%）。
func (st *ezfyBattleState) comboFollowUp(unit, target *ezfyFightUnit, enemies []*ezfyFightUnit,
	rangeD, atkFlat int, atkBonusAt, defBonusAt func(*ezfyFightUnit) int, defBreakPct int, side, enemySide string,
	nameOf func(*ezfyFightUnit) string, skillName string, dmgPct int, label string, strictTarget bool,
	critChance, critDmg int) {
	if label == "" {
		label = "连击"
	}
	if unit == nil || !unit.alive() || dmgPct <= 0 {
		return
	}
	cur := target
	if cur == nil || !cur.alive() {
		// ★★ 2026-10-31 用户纠正「追击、连击不是一个意思」：
		//   **连击** = 本回合多打一次普攻（可换目标、可溢出）；
		//   **追击** = 普攻命中后的战法伤害，**只打被普攻的那个目标**，目标已死就不再追。
		if strictTarget {
			return
		}
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
	// ★★ 2026-10-31 按**实际出手目标**取兵种专属攻/防加成（狼群围猎/四指阵列按目标类型生效）。
	atkBonus, defBonus := 0, 0
	if atkBonusAt != nil {
		atkBonus = atkBonusAt(cur)
	}
	if defBonusAt != nil {
		defBonus = defBonusAt(cur)
	}
	damage := ezfyCalcDamage(ezfyPickAttack(unit.cfg, cur.cfg)+atkFlat, def, unit.count, atkBonus, defBonus)
	damage = damage * int64(dmgPct) / 100
	if damage < 1 {
		damage = 1
	}
	// ★★ 2026-10-31 连击的第二次攻击**单独掷暴击**（与主攻击、与追击都互不共用）。
	crit, critBonus := ezfyRollCrit(critChance, critDmg)
	if crit {
		damage = damage * int64(100+critBonus) / 100
		st.Actions = append(st.Actions, fmt.Sprintf("%s%s发动【暴击】，本次%s伤害+%d%%",
			side, nameOf(unit), label, critBonus))
	}
	critTxt := ""
	if crit {
		critTxt = fmt.Sprintf("【暴击+%d%%】", critBonus)
	}
	chp := cur.hp
	if chp < 1 {
		chp = 1
	}
	killed := damage / int64(chp)
	// 与主攻击同口径：暴击的零头向上取整，否则 1.5 倍伤害会被向下取整抹掉。
	if crit && damage%int64(chp) != 0 {
		killed++
	}
	if killed < 1 {
		killed = 1
	}
	if killed > cur.count {
		killed = cur.count
	}
	cur.count -= killed
	// ★★ 2026-10-31 陈述式（去感叹号）：
	//   `【攻方】轰炸机发动组合技【梯次冲锋】连击，守方轰炸机损失7个`
	//   暴击时行内带【暴击+N%】，与主攻击的标记口径一致。
	st.Actions = append(st.Actions, fmt.Sprintf("%s%s发动组合技【%s】%s%s，%s%s损失%d个",
		side, nameOf(unit), skillName, label, critTxt, enemySide, nameOf(cur), killed))
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
		// ★ 2026-10-25 详情页「特殊技能：」一行用的 `+攻击力180%` 文案（见 ezfyComboEffectTxt）
		"effect_text": ezfyComboEffectTxt(c),
		"proc_pct":    c.ProcPct, "rounds": c.Rounds, "des": des,
	}
}

// ezfyComboBriefs 组合技列表的对外结构（空 → nil，前端据此不渲染）。
//
// ★ 2026-10-24 名将可学多个组合技：玩家端下发**数组**（普通军官恒 nil）。
func ezfyComboBriefs(skills []*ezfyComboSkill) interface{} {
	if len(skills) == 0 {
		return nil
	}
	out := make([]gin.H, 0, len(skills))
	for _, c := range skills {
		if b := ezfyComboBrief(c); b != nil {
			out = append(out, b.(gin.H))
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// ezfyComboBriefAtMax 组合技的**满级**展示（名将图鉴用）。
//
// 图鉴是「这个技能满级多强」的预览，不该拿池子配置的 level 去定级
// （池里 level 是「等级上限」，本地库甚至可能填 1 → 图鉴会显示成 Lv.1 很误导）。
func ezfyComboBriefAtMax(g *model.EzfyCfgGeneral) interface{} {
	if g == nil || g.Kind != 2 {
		return nil
	}
	// 传一个足够大的等级 → 必被夹到 max_level，即「满级」
	return ezfyComboBriefs(ezfyComboOfGeneralCfg(g, 1<<20))
}

// officerComboView 军官详情里的组合技列表（普通军官恒 nil）
func (h *EzfyHandler) officerComboView(o *model.EzfyOfficer) interface{} {
	return ezfyComboBriefs(h.comboSkillsOfOfficer(o))
}

// comboShortTxts 运行时列表 → 短效果文案列表（准备回合军官行内联用）
func comboShortTxts(rts []*ezfyComboRuntime) []string {
	out := []string{}
	for _, rt := range rts {
		if rt == nil || rt.Skill == nil {
			continue
		}
		if t := ezfyComboShortTxt(rt.Skill); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// appendComboHead 战报「准备回合」展示组合技。
//
// ★ 2026-10-23 用户反馈：原来单独一行「【攻方组合技】…」被追加在最后（排在
// 「战场初始相距6000」之后），既像补充说明、又和军官技能割裂。改成**并进军官行**，
// 格式对齐军官技能：
//
//	【攻方军官】斯大林 Lv.350 攻击加成+477% 防御加成+253% 尖兵猛攻(VI 攻击力+180%) …
//	              └→ 追加：装甲突击(X 攻击力+180%，发动40%) 全线总攻(VIII 攻击力+120%，发动30%)
//
// ★ 2026-10-24 名将可学多个组合技：多个技能文案用空格接在军官行后面。
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
	atkTxt := strings.Join(comboShortTxts(st.AtkCombos), " ")
	defTxt := strings.Join(comboShortTxts(st.DefCombos), " ")
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
