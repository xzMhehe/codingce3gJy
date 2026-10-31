package ezfy

import "fmt"

// ============ 普通军官技能「间隔发动」（2026-10-31） ============
//
// 配置表（ezfy_cfg_skill）的文案本来就是「每3回合提升XX，持续1回合」，
// 但代码一直把它当成**常驻加成**（开战即生效、整场不减），
// 玩家实测「弧形壁垒 / 火炮压制 一直在生效」，与文案不符。用户 2026-10-31 要求按文案改。
//
// 口径（用户确认）：
//   · 弧形壁垒：每 3 回合提升防御力，持续 1 回合（第 1、4、7 … 回合发动）
//   · 火炮压制：每 3 回合提升**陆军装甲部队**攻击，持续 1 回合
//               「攻击力」= 本回合 对地/对空/对海/对防 各 +等级数值（10×技能等级）
//               装甲部队 = 装甲车(4)/轻型坦克(5)/重型坦克(6)/突击炮(7)/火箭(8)
//
// 实现方式：技能值仍然算进战场快照（= 「发动回合」的满额加成），
// 战斗引擎在**非发动回合**把这部分扣掉 —— 这样：
//   ① 老快照（没有 AtkTimed/DefTimed）行为完全不变；
//   ② 不需要改出征/活动两条链路里几十处加成计算；
//   ③ 战报里「发动/未发动」一目了然（每回合开头有播报行）。

// 陆军装甲部队（火炮压制作用范围）
var ezfyArmorTroopIds = []int{4, 5, 6, 7, 8}

// 坦克冲锋作用范围（用户 2026-10-31：只生效轻型坦克、重型坦克）
var ezfyTankRushTroopIds = []int{5, 6}

// ezfyTimedSkill 一个「每 Interval 回合发动、持续 Duration 回合」的技能运行时。
//
// 只存**效果值**（已按技能等级算好），引擎按回合判断是否生效。
// 字段全部导出 + json tag：战场快照要能原样往返（指挥室暂停再打不能丢状态）。
type ezfyTimedSkill struct {
	Name     string `json:"name"`
	Level    int    `json:"level"`
	Interval int    `json:"interval"` // 每 N 回合发动一次（0 → 默认 3）
	Duration int    `json:"duration"` // 每次持续 M 回合（0 → 默认 1）
	Left     int    `json:"left"`     // 剩余生效回合（>0 = 本回合生效）

	// 效果：只填本技能用到的那一项
	Def        int         `json:"def,omitempty"`         // 防御加成%（弧形壁垒）
	AtkFlat    map[int]int `json:"atk_flat,omitempty"`    // 兵种 ID → 攻击属性加成（火炮压制）
	Range      map[int]int `json:"range,omitempty"`       // 兵种 type → 射程%（保留扩展位）
	SpeedTroop map[int]int `json:"speed_troop,omitempty"` // 兵种 ID → 速度%（保留扩展位）
}

// ezfyTimedSkillInterval 默认间隔：每 3 回合发动 1 回合
const ezfyTimedSkillInterval = 3

// procOn 第 round 回合是否**是发动回合**：第 1、1+N、1+2N … 回合
// （战斗开始第一回合就能吃到，和「绝地逆袭第1回合」的节奏一致）
func (s *ezfyTimedSkill) procOn(round int) bool {
	if s == nil || round < 1 {
		return false
	}
	interval := s.Interval
	if interval <= 0 {
		interval = ezfyTimedSkillInterval
	}
	return (round-1)%interval == 0
}

// durationOf 本次发动的持续回合（默认 1）
func (s *ezfyTimedSkill) durationOf() int {
	if s == nil || s.Duration <= 0 {
		return 1
	}
	return s.Duration
}

// ezfyTimedRoll 回合开始刷新每个技能的 Left（发动回合 → Duration；其余 → 递减）。
//
// 返回**本回合生效**的技能列表（给调用方做播报/扣减）。
func ezfyTimedRoll(list []*ezfyTimedSkill, round int) []*ezfyTimedSkill {
	active := []*ezfyTimedSkill{}
	for _, s := range list {
		if s == nil {
			continue
		}
		if s.procOn(round) {
			s.Left = s.durationOf()
		}
		if s.Left > 0 {
			active = append(active, s)
		}
	}
	return active
}

// ezfyTimedEndRound 回合结束：生效回合数递减（与组合技 comboEndRound 同口径）
func ezfyTimedEndRound(list []*ezfyTimedSkill) {
	for _, s := range list {
		if s != nil && s.Left > 0 {
			s.Left--
		}
	}
}

// ezfyTimedDefOff 本回合**没发动**的技能的防御加成合计（要从静态加成里扣掉）
func ezfyTimedDefOff(list []*ezfyTimedSkill, round int) int {
	off := 0
	for _, s := range list {
		if s == nil || s.procOn(round) {
			continue
		}
		off += s.Def
	}
	return off
}

// ezfyTimedAtkFlatOff 本回合没发动的技能里，按兵种 ID 的攻击属性加成合计（扣减用）
func ezfyTimedAtkFlatOff(list []*ezfyTimedSkill, round int) map[int]int {
	out := map[int]int{}
	for _, s := range list {
		if s == nil || s.procOn(round) {
			continue
		}
		for id, v := range s.AtkFlat {
			out[id] += v
		}
	}
	return out
}

// ezfyTimedRangeOff 本回合没发动的技能里，按兵种 type 的射程加成合计（扣减用）
func ezfyTimedRangeOff(list []*ezfyTimedSkill, round int) map[int]int {
	out := map[int]int{}
	for _, s := range list {
		if s == nil || s.procOn(round) {
			continue
		}
		for t, v := range s.Range {
			out[t] += v
		}
	}
	return out
}

// ezfyTimedSpeedTroopOff 本回合没发动的技能里，按兵种 ID 的速度加成合计（扣减用）
func ezfyTimedSpeedTroopOff(list []*ezfyTimedSkill, round int) map[int]int {
	out := map[int]int{}
	for _, s := range list {
		if s == nil || s.procOn(round) {
			continue
		}
		for id, v := range s.SpeedTroop {
			out[id] += v
		}
	}
	return out
}

// ezfyTimedBreakFilter 明细过滤：把**本回合没发动**的技能对应的明细项去掉。
//
// 战报的「防御加成+N%(军官技能·弧形壁垒+150%)」用的是静态明细（快照里恒含技能），
// 非发动回合如果照原样展示，玩家会以为这回合也吃了 150% —— 所以按回合过滤。
func ezfyTimedBreakFilter(items []ezfyBonusItem, list []*ezfyTimedSkill, round int) []ezfyBonusItem {
	if len(list) == 0 || len(items) == 0 {
		return items
	}
	off := map[string]bool{}
	for _, s := range list {
		if s == nil || s.procOn(round) {
			continue
		}
		off["军官技能·"+s.Name] = true
		off[s.Name] = true
	}
	if len(off) == 0 {
		return items
	}
	out := make([]ezfyBonusItem, 0, len(items))
	for _, it := range items {
		if off[it.Name] {
			continue
		}
		out = append(out, it)
	}
	return out
}

// ezfyTimedHas 该技能名是否属于「每 N 回合发动」类（弧形壁垒 / 火炮压制）。
//
// ★★ 2026-10-31 用户反馈「同回合出现两行发动播报」：这类技能由 timedAnno **统一播报一次**
// （只在发动回合），skillAnno 必须跳过同名技能，否则会和防御明细里的那行重复。
func ezfyTimedHas(list []*ezfyTimedSkill, name string) bool {
	for _, s := range list {
		if s != nil && s.Name == name {
			return true
		}
	}
	return false
}

// timedAnno 技能发动播报：只在**发动回合**写一行，未发动不刷屏。
//
// 形如：`【攻方】Stalin发动【弧形壁垒】，全军防御+150%`
//
// ★★ 2026-10-31 用户要求：「展示一次就行，去掉『（持续1回合，每3回合发动1次）』，
//
//	无论攻守方一次就行，玩家知道每3回合发动一次，不用提示」→ 去掉回合/间隔后缀。
//	同一回合里同一个技能**只出一行**（skillAnno 会跳过本表里的技能名，见 ezfyTimedHas）。
func (st *ezfyBattleState) timedAnno(atkOn, defOn []*ezfyTimedSkill) {
	anno := func(on []*ezfyTimedSkill, side, offDesc string, off int) {
		for _, s := range on {
			if s == nil {
				continue
			}
			eff := ""
			switch {
			case s.Def > 0:
				eff = fmt.Sprintf("全军防御+%d%%", s.Def)
			case len(s.AtkFlat) > 0:
				v := 0
				for _, x := range s.AtkFlat {
					v = x
					break
				}
				eff = fmt.Sprintf("陆军装甲部队攻击+%d", v)
			case len(s.Range) > 0:
				v := 0
				for _, x := range s.Range {
					v = x
					break
				}
				eff = fmt.Sprintf("城防攻击范围+%d%%", v)
			default:
				continue
			}
			// ★★ 2026-10-31 与常驻技能同口径：`【守方】某某发动【弧形壁垒】，全军防御+150%`
			//   （去掉口号与感叹号 —— 简约、官方、二战题材）
			st.Actions = append(st.Actions, fmt.Sprintf("%s%s发动【%s】，%s",
				side, ezfyOfficerShortName(offDesc), s.Name, eff))
		}
	}
	anno(atkOn, "【攻方】", st.AtkOfficerDesc, 0)
	anno(defOn, "【守方】", st.DefOfficerDesc, 1)
}
