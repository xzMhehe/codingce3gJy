package handler

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
// ★ 2026-09-22 用户要求「实现指挥功能」（入口：军情 → 军队动态 → [指挥]）：
// 引擎从「一次跑完所有回合」拆成**可逐回合推进的状态机**（ezfyBattleState.Step），
// 这样才能支持「每回合 30 秒、前 25 秒下指令、后 5 秒锁定并由服务器结算」的实时指挥。
// ezfySimulate 保留为兼容包装（内部循环 Step），现有调用点零改动。

const (
	// 最大回合数 —— 复刻《战斗机制（家园玩家必看）》§1「战斗最多40回合；达到上限仍未分胜负则按平局处理」
	ezfyBattleMaxRounds = 40
	ezfyBattleStartDist = 6000 // 战场初始距离
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
	Rounds         int
	Actions        []string
	AttackerLosses []ezfyUnitGroup
	DefenderLosses []ezfyUnitGroup
	AttackerLeft   []ezfyUnitGroup
}

// ============ 战场状态机（实时指挥用） ============

// ezfyBattleState 战场状态：可以**一回合一次**地推进，也可以一次跑完（ezfySimulate）。
//
// 加成字段存的是**不含装备**的基础值，Step 里再叠加装备六项 ——
// 这样快照往返（存库 → 重建）不会把装备加成叠两遍。
type ezfyBattleState struct {
	Attackers []*ezfyFightUnit
	Defenders []*ezfyFightUnit

	AtkBonus      int
	DefBonus      int
	AtkSpeedBonus int
	DefSpeedBonus int
	AtkEquip      ezfyBattleBonus
	DefEquip      ezfyBattleBonus

	AtkTargets map[int]int
	DefTargets map[int]int
	AtkMoves   map[int]int
	DefMoves   map[int]int

	AtkOfficerDesc string
	DefOfficerDesc string

	Round       int  // 已结算回合数
	Done        bool // 是否已分胜负 / 回合耗尽
	AttackerWin bool

	Head    []string // 开局描述（军官/加成/初始距离），只写一次
	Actions []string // 每回合的行动日志
}

// ezfyNewBattleState 初始化战场
// attackerUnits/defenderUnits: [troopId, count]
// atkBonus: 攻方攻击加成%(军官+科技)  defBonus: 守方防御加成%(城墙+科技+城守)
// atkSpeedBonus/defSpeedBonus: 速度加成%
// atkEquip/defEquip: 装备六项加成（伤害/防御/生命/移动距离/暴击几率/暴击伤害，单位百分点）
// atkTargets/defTargets: 兵种ID->优先攻击兵种ID(0=最近, 司令部配置)
// atkMoves/defMoves: 兵种ID->1前进 0停止（玩家不下指令时的默认行为）
func ezfyNewBattleState(attackerUnits, defenderUnits []ezfyUnitGroup,
	atkBonus, defBonus, atkSpeedBonus, defSpeedBonus int,
	atkEquip, defEquip ezfyBattleBonus,
	atkOfficerDesc, defOfficerDesc string,
	atkTargets, defTargets map[int]int,
	atkMoves, defMoves map[int]int) *ezfyBattleState {

	st := &ezfyBattleState{
		AtkBonus: atkBonus, DefBonus: defBonus,
		AtkSpeedBonus: atkSpeedBonus, DefSpeedBonus: defSpeedBonus,
		AtkEquip: atkEquip, DefEquip: defEquip,
		AtkTargets: atkTargets, DefTargets: defTargets,
		AtkMoves: atkMoves, DefMoves: defMoves,
		AtkOfficerDesc: atkOfficerDesc, DefOfficerDesc: defOfficerDesc,
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
	st.Head = append(st.Head, fmt.Sprintf("战斗加成: 攻方 攻击+%d%% 速度+%d%% | 守方 防御+%d%% 速度+%d%%",
		effAtk, effAtkSpeed, effDef, effDefSpeed))
	if atkEquip != (ezfyBattleBonus{}) {
		st.Head = append(st.Head, "【攻方装备】"+ezfyEquipBonusDesc(atkEquip))
	}
	if defEquip != (ezfyBattleBonus{}) {
		st.Head = append(st.Head, "【守方装备】"+ezfyEquipBonusDesc(defEquip))
	}
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

// Step 结算**一个回合**。返回 true 表示战斗已结束。
//
// atkCmds: 攻方**逐兵种**的指令（troopId → advance|hold|retreat）。
// ★ 用户要求（2026-09-22）：「指挥不是指挥全部，自己带的兵种都能指挥，就是单独指挥」
// —— 所以指令是按兵种存的，没给的兵种回落到司令部兵种配置。
// defCmd: 守方统一指令（AI 不下指令，一般传空串）。
func (st *ezfyBattleState) Step(atkCmds map[int]string, defCmd string) bool {
	if st.Done {
		return true
	}
	st.Round++

	// 装备加成在这里叠加（快照里存的是不含装备的基础值）
	atkBonus := st.AtkBonus + st.AtkEquip.Dmg
	defBonus := st.DefBonus + st.DefEquip.Def
	atkSpeedBonus := st.AtkSpeedBonus + st.AtkEquip.Move
	defSpeedBonus := st.DefSpeedBonus + st.DefEquip.Move

	st.Actions = append(st.Actions, fmt.Sprintf("第%d回合:", st.Round))
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

		rangeD := unit.cfg.AttackRange
		dist := ezfyAbs(target.pos - unit.pos)

		moveMap := st.DefMoves
		cmd := defCmd
		speedBonus := defSpeedBonus
		if isAtk {
			moveMap = st.AtkMoves
			speedBonus = atkSpeedBonus
			// ★ 按兵种取指令（缺省 = 空串 → 回落司令部兵种配置）
			//   0 号键是旧格式遗留的「全军统一指令」，老战场记录也能正常打
			cmd = ""
			if atkCmds != nil {
				if v, ok := atkCmds[unit.cfg.ID]; ok {
					cmd = v
				} else if v, ok := atkCmds[0]; ok {
					cmd = v
				}
			}
		}

		dir := ezfyMoveDir(unit, moveMap, cmd)
		if dir != 0 && dist > rangeD {
			move := unit.cfg.Speed * (100 + speedBonus) / 100
			if move > dist-rangeD {
				move = dist - rangeD
			}
			move *= dir
			if isAtk {
				unit.pos += move
			} else {
				unit.pos -= move
			}
			dist = ezfyAbs(target.pos - unit.pos)
			verb := "前进"
			if dir < 0 {
				verb = "后撤"
			}
			st.Actions = append(st.Actions, fmt.Sprintf("%s%s%s%d, 与%s%s相距%d",
				side, unit.cfg.Name, verb, ezfyAbs(move), enemySide, target.cfg.Name, dist))
		}
		if dist <= rangeD {
			baseAtk := ezfyPickAttack(unit.cfg, target.cfg)
			unitAtkBonus := 0
			unitDefBonus := defBonus
			equip := st.DefEquip
			if isAtk {
				unitAtkBonus = atkBonus
				unitDefBonus = 0
				equip = st.AtkEquip
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
			// ★ 暴击：按暴击几率 roll，命中则乘 (1 + 暴击伤害加成)
			crit := false
			if equip.Crit > 0 && rand.Intn(100) < equip.Crit {
				crit = true
				damage = damage * int64(100+equip.CritDmg) / 100
			}
			effHealth := target.cfg.Health * hpMul / 100
			if effHealth < 1 {
				effHealth = 1
			}
			killed := damage / int64(effHealth)
			if killed < 1 {
				killed = 1
			}
			if killed > target.count {
				killed = target.count
			}
			if killed > 0 {
				target.count -= killed
				critTxt := ""
				if crit {
					critTxt = "【暴击】"
				}
				st.Actions = append(st.Actions, fmt.Sprintf("%s%s攻击%s%s%s, 消灭%d个",
					side, unit.cfg.Name, critTxt, enemySide, target.cfg.Name, killed))
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
		} else if st.Round >= ezfyBattleMaxRounds {
			// 回合耗尽按平局处理（复刻文档：平局时守方视为守住）
			st.Done, st.AttackerWin = true, false
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
		Rounds:         st.Round,
		Actions:        actions,
		AttackerLosses: ezfyToGroups(st.Attackers, true),
		DefenderLosses: ezfyToGroups(st.Defenders, true),
		AttackerLeft:   ezfyToGroups(st.Attackers, false),
	}
}

// ezfySimulate 执行战斗（兼容包装：一次跑完）
func ezfySimulate(attackerUnits, defenderUnits []ezfyUnitGroup,
	atkBonus, defBonus, atkSpeedBonus, defSpeedBonus int,
	atkEquip, defEquip ezfyBattleBonus,
	atkOfficerDesc, defOfficerDesc string,
	atkTargets, defTargets map[int]int,
	atkMoves, defMoves map[int]int) ezfyBattleResult {

	st := ezfyNewBattleState(attackerUnits, defenderUnits,
		atkBonus, defBonus, atkSpeedBonus, defSpeedBonus,
		atkEquip, defEquip, atkOfficerDesc, defOfficerDesc,
		atkTargets, defTargets, atkMoves, defMoves)
	for !st.Done {
		// nil = 沿用司令部的兵种战斗配置，与原实现行为一致
		st.Step(nil, "")
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

	AtkBonus       int             `json:"atk_bonus"`
	DefBonus       int             `json:"def_bonus"`
	AtkSpeedBonus  int             `json:"atk_speed_bonus"`
	DefSpeedBonus  int             `json:"def_speed_bonus"`
	AtkEquip       ezfyBattleBonus `json:"atk_equip"`
	DefEquip       ezfyBattleBonus `json:"def_equip"`
	AtkTargets     map[int]int     `json:"atk_targets"`
	DefTargets     map[int]int     `json:"def_targets"`
	AtkMoves       map[int]int     `json:"atk_moves"`
	DefMoves       map[int]int     `json:"def_moves"`
	AtkOfficerDesc string          `json:"atk_officer_desc"`
	DefOfficerDesc string          `json:"def_officer_desc"`

	Round       int      `json:"round"`
	Done        bool     `json:"done"`
	AttackerWin bool     `json:"attacker_win"`
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
		AtkSpeedBonus: st.AtkSpeedBonus, DefSpeedBonus: st.DefSpeedBonus,
		AtkEquip: st.AtkEquip, DefEquip: st.DefEquip,
		AtkTargets: st.AtkTargets, DefTargets: st.DefTargets,
		AtkMoves: st.AtkMoves, DefMoves: st.DefMoves,
		AtkOfficerDesc: st.AtkOfficerDesc, DefOfficerDesc: st.DefOfficerDesc,

		Round: st.Round, Done: st.Done, AttackerWin: st.AttackerWin,
		Head: st.Head, Actions: actions,
	}
}

// ezfyBattleStateFromSnapshot 从快照重建战场（兵种配置按 troop_id 重新查表）
func ezfyBattleStateFromSnapshot(snap ezfyBattleSnapshot) *ezfyBattleState {
	st := &ezfyBattleState{
		AtkBonus: snap.AtkBonus, DefBonus: snap.DefBonus,
		AtkSpeedBonus: snap.AtkSpeedBonus, DefSpeedBonus: snap.DefSpeedBonus,
		AtkEquip: snap.AtkEquip, DefEquip: snap.DefEquip,
		AtkTargets: snap.AtkTargets, DefTargets: snap.DefTargets,
		AtkMoves: snap.AtkMoves, DefMoves: snap.DefMoves,
		AtkOfficerDesc: snap.AtkOfficerDesc, DefOfficerDesc: snap.DefOfficerDesc,
		Round: snap.Round, Done: snap.Done, AttackerWin: snap.AttackerWin,
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

// ezfyPickTarget 选择攻击目标: 司令部配置的优先兵种(取最近), 否则最近目标
func ezfyPickTarget(unit *ezfyFightUnit, enemies []*ezfyFightUnit, targetMap map[int]int) *ezfyFightUnit {
	targetTroop := 0
	if targetMap != nil {
		targetTroop = targetMap[unit.cfg.ID]
	}
	best := (*ezfyFightUnit)(nil)
	minDist := int(^uint(0) >> 1)
	for _, e := range enemies {
		d := ezfyAbs(e.pos - unit.pos)
		if d < minDist {
			minDist = d
			best = e
		}
	}
	if targetTroop > 0 {
		for _, e := range enemies {
			if e.cfg.ID == targetTroop {
				d := ezfyAbs(e.pos - unit.pos)
				if d < minDist {
					minDist = d
					best = e
				}
			}
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
