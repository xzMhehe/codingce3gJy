package handler

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
)

// 二战风云 多回合战斗引擎（忠实移植 BattleEngine.java）
// 规则: 回合制, 每回合按速度从高到低行动; 双方相向移动, 进入射程后开火; 一方全灭或回合耗尽结束

const (
	ezfyBattleMaxRounds = 30   // 最大回合数
	ezfyBattleStartDist = 6000 // 战场初始距离
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

// ezfySimulate 执行战斗
// attackerUnits/defenderUnits: [troopId, count]
// atkBonus: 攻方攻击加成%(科技)  defBonus: 守方防御加成%(城墙+科技)
// atkSpeedBonus/defSpeedBonus: 速度加成%
// atkEquip/defEquip: 装备六项加成（伤害/防御/生命/移动距离/暴击几率/暴击伤害，单位百分点）
// atkTargets/defTargets: 兵种ID->优先攻击兵种ID(0=最近, 司令部配置)
// atkMoves/defMoves: 兵种ID->1前进 0停止
func ezfySimulate(attackerUnits, defenderUnits []ezfyUnitGroup,
	atkBonus, defBonus, atkSpeedBonus, defSpeedBonus int,
	atkEquip, defEquip ezfyBattleBonus,
	atkOfficerDesc, defOfficerDesc string,
	atkTargets, defTargets map[int]int,
	atkMoves, defMoves map[int]int) ezfyBattleResult {

	result := ezfyBattleResult{Actions: []string{}}
	actions := result.Actions
	if atkOfficerDesc != "" {
		actions = append(actions, "【攻方军官】"+atkOfficerDesc)
	}
	if defOfficerDesc != "" {
		actions = append(actions, "【守方军官】"+defOfficerDesc)
	}
	// ★ 装备六项加成并入基础加成：伤害→攻击、防御→防御、移动距离→速度
	//   （生命/暴击几率/暴击伤害在伤害结算里单独算，见 ezfyCalcDamage）
	atkBonus += atkEquip.Dmg
	defBonus += defEquip.Def
	atkSpeedBonus += atkEquip.Move
	defSpeedBonus += defEquip.Move
	actions = append(actions, fmt.Sprintf("战斗加成: 攻方 攻击+%d%% 速度+%d%% | 守方 防御+%d%% 速度+%d%%",
		atkBonus, atkSpeedBonus, defBonus, defSpeedBonus))
	if atkEquip != (ezfyBattleBonus{}) {
		actions = append(actions, "【攻方装备】"+ezfyEquipBonusDesc(atkEquip))
	}
	if defEquip != (ezfyBattleBonus{}) {
		actions = append(actions, "【守方装备】"+ezfyEquipBonusDesc(defEquip))
	}
	actions = append(actions, fmt.Sprintf("战场初始相距%d, 攻守双方相向推进", ezfyBattleStartDist))

	attackers := []*ezfyFightUnit{}
	defenders := []*ezfyFightUnit{}
	idx := 0
	for _, ug := range attackerUnits {
		cfg := ezfyStatsOf(ug.TroopId)
		if cfg != nil && ug.Count > 0 {
			idx++
			attackers = append(attackers, &ezfyFightUnit{id: fmt.Sprintf("A%d", idx), cfg: cfg, count: ug.Count, initialCount: ug.Count, pos: 0})
		}
	}
	idx = 0
	for _, ug := range defenderUnits {
		cfg := ezfyStatsOf(ug.TroopId)
		if cfg != nil && ug.Count > 0 {
			idx++
			defenders = append(defenders, &ezfyFightUnit{id: fmt.Sprintf("D%d", idx), cfg: cfg, count: ug.Count, initialCount: ug.Count, pos: ezfyBattleStartDist})
		}
	}
	if len(attackers) == 0 || len(defenders) == 0 {
		result.AttackerWin = len(attackers) > 0
		result.Rounds = 0
		result.Actions = actions
		result.AttackerLosses = ezfyToGroups(attackers, true)
		result.DefenderLosses = ezfyToGroups(defenders, true)
		result.AttackerLeft = ezfyToGroups(attackers, false)
		return result
	}

	attackerWin := false
	round := 0
	for round = 1; round <= ezfyBattleMaxRounds; round++ {
		actions = append(actions, fmt.Sprintf("第%d回合:", round))
		all := append(append([]*ezfyFightUnit{}, attackers...), defenders...)
		sort.SliceStable(all, func(i, j int) bool { return all[i].cfg.Speed > all[j].cfg.Speed })

		for _, unit := range all {
			if !unit.alive() {
				continue
			}
			isAtk := ezfyContains(attackers, unit)
			side := "【守方】"
			enemySide := "攻方"
			if isAtk {
				side = "【攻方】"
				enemySide = "守方"
			}
			enemies := defenders
			if !isAtk {
				enemies = attackers
			}
			liveEnemies := ezfyAliveList(enemies)
			if len(liveEnemies) == 0 {
				break
			}
			targetMap := defTargets
			if isAtk {
				targetMap = atkTargets
			}
			target := ezfyPickTarget(unit, liveEnemies, targetMap)

			rangeD := unit.cfg.AttackRange
			dist := ezfyAbs(target.pos - unit.pos)
			moveMap := defMoves
			if isAtk {
				moveMap = atkMoves
			}
			// 移动判定: 与原版 BattleEngine 一致 —— 未配置的兵种默认「前进」
			// (Java: moveMap.getOrDefault(unit.cfg.getId(), 1) > 0)
			// 注意不能写成 moveMap[id] > 0: 缺键会取零值 0 变成「停止」,
			// 导致攻方近战兵种原地不动、永远够不到敌人而必败。
			moveForward := true
			if moveMap != nil {
				if v, ok := moveMap[unit.cfg.ID]; ok {
					moveForward = v > 0
				}
			}
			if moveForward && dist > rangeD {
				speedBonus := defSpeedBonus
				if isAtk {
					speedBonus = atkSpeedBonus
				}
				move := unit.cfg.Speed * (100 + speedBonus) / 100
				if move > dist-rangeD {
					move = dist - rangeD
				}
				if isAtk {
					unit.pos += move
				} else {
					unit.pos -= move
				}
				dist = ezfyAbs(target.pos - unit.pos)
				actions = append(actions, fmt.Sprintf("%s%s前进%d, 与%s%s相距%d", side, unit.cfg.Name, move, enemySide, target.cfg.Name, dist))
			}
			if dist <= rangeD {
				baseAtk := ezfyPickAttack(unit.cfg, target.cfg)
				unitAtkBonus := 0
				unitDefBonus := defBonus
				equip := defEquip
				if isAtk {
					unitAtkBonus = atkBonus
					unitDefBonus = 0
					equip = atkEquip
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
					actions = append(actions, fmt.Sprintf("%s%s攻击%s%s%s, 消灭%d个",
						side, unit.cfg.Name, critTxt, enemySide, target.cfg.Name, killed))
				}
			}
			if len(ezfyAliveList(enemies)) == 0 {
				attackerWin = isAtk
				break
			}
		}
		if attackerWin {
			break
		}
		if len(ezfyAliveList(defenders)) == 0 {
			attackerWin = true
			break
		}
		if len(ezfyAliveList(attackers)) == 0 {
			attackerWin = false
			break
		}
	}

	result.AttackerWin = attackerWin
	if round > ezfyBattleMaxRounds {
		round = ezfyBattleMaxRounds
	}
	result.Rounds = round
	result.Actions = actions
	result.AttackerLosses = ezfyToGroups(attackers, true)
	result.DefenderLosses = ezfyToGroups(defenders, true)
	result.AttackerLeft = ezfyToGroups(attackers, false)
	return result
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
