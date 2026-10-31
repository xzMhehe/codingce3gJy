package seed

import (
	"fmt"
	"log"

	"gorm.io/gorm"

	"qqjiayuan/server/internal/model"
)

// 名将组合技能池（★ 2026-10-23）
//
// 设计要点（详见仓库根目录 `1022名将组合技能.md`）：
//   - **系统专属**：普通军官（kind=1）学不到，只能由管理端把池子里的技能配给名将
//     （`ezfy_cfg_general.combo_skill_ids`，JSON 数组，一个名将可配多个）；
//   - **通用**：一套效果对所有兵种生效（陆/海/空不区分），与战争系统的攻击/防御/射程/
//     速度/反击/治疗/连击 同一套参数，不做第二套数值体系；
//   - **按等级成长**：组合技等级由**军官等级**分档（默认每 35 级 +1，350 级 = 10 级），
//     效果值 = BaseValue + (等级-1)×GrowPerLevel；
//   - **概率发动**：每回合开始按 ProcPct 掷一次，命中后持续 Rounds 回合（1 = 只本回合）；
//   - **数值克制**：满级对齐「普通军官技能满级」的量级（尖兵猛攻/弧形壁垒 Lv.6 = 180%），
//     但组合技是**概率发动 + 只持续 1 回合**，期望值只有标称值的 30~40%，
//     不会像常驻技能那样每回合都吃满。
//
// ★ 2026-10-23 二次调整：等级上限 5 → 10、步长 70 → 35。
// ★ 2026-10-23 三次调整（用户反馈「16% 太少了」）：数值放大到普通技能量级 ——
//
//	攻击/防御/射程/速度/连击 满级 180%（每级 +18），全线总攻 2 回合所以 120%（+12），
//	减伤/破防受「100% 封顶」限制取 80%，治疗 60%，反击回合仍是 +5（次数型）。
var ezfyGeneralSkillSeeds = []model.EzfyCfgGeneralSkill{
	// —— 增益类（本回合按概率发动，直接进战斗加成）——
	{ID: 1, Name: "装甲突击", Kind: 1, BaseValue: 18, GrowPerLevel: 18, MaxLevel: 10, LevelPerStep: 35, ProcPct: 40, Rounds: 1, Quality: 2, SortNo: 1,
		Des: "攻击力+%d%%"},
	{ID: 2, Name: "纵深防御", Kind: 2, BaseValue: 18, GrowPerLevel: 18, MaxLevel: 10, LevelPerStep: 35, ProcPct: 35, Rounds: 1, Quality: 3, SortNo: 2,
		Des: "防御力+%d%%"},
	{ID: 3, Name: "火力延伸", Kind: 3, BaseValue: 18, GrowPerLevel: 18, MaxLevel: 10, LevelPerStep: 35, ProcPct: 30, Rounds: 1, Quality: 2, SortNo: 3,
		Des: "射程+%d%%"},
	{ID: 4, Name: "快速集群", Kind: 4, BaseValue: 18, GrowPerLevel: 18, MaxLevel: 10, LevelPerStep: 35, ProcPct: 35, Rounds: 1, Quality: 2, SortNo: 4,
		Des: "速度+%d%%"},
	// —— 减伤 / 破防（★ 都受 100% 封顶限制，取 80%）——
	{ID: 5, Name: "筑垒地域", Kind: 5, BaseValue: 8, GrowPerLevel: 8, MaxLevel: 10, LevelPerStep: 35, ProcPct: 40, Rounds: 1, Quality: 3, SortNo: 5,
		Des: "受到伤害-%d%%"},
	{ID: 6, Name: "穿甲火力", Kind: 6, BaseValue: 8, GrowPerLevel: 8, MaxLevel: 10, LevelPerStep: 35, ProcPct: 35, Rounds: 1, Quality: 4, SortNo: 6,
		Des: "破防%d%%（发动后本回合攻击无视目标 %d%% 防御）"},
	// —— 治疗 ——
	{ID: 7, Name: "野战医院", Kind: 7, BaseValue: 6, GrowPerLevel: 6, MaxLevel: 10, LevelPerStep: 35, ProcPct: 30, Rounds: 1, Quality: 3, SortNo: 7,
		Des: "回合治疗%d%%（发动回合结束时恢复本回合已损兵力的 %d%%）"},
	// —— 连击（追加一击的伤害为原来的 %d%%）——
	{ID: 8, Name: "波次突击", Kind: 8, BaseValue: 18, GrowPerLevel: 18, MaxLevel: 10, LevelPerStep: 35, ProcPct: 30, Rounds: 1, Quality: 4, SortNo: 8,
		Des: "连击（发动后本回合多打一次普攻，这次普攻伤害为原来的 %d%%）"},
	// —— 反击回合（★ 次数型：上限仍 5 级，10 级就是 +10 回合，太变态）——
	{ID: 9, Name: "死守反击", Kind: 9, BaseValue: 1, GrowPerLevel: 1, MaxLevel: 5, LevelPerStep: 35, ProcPct: 30, Rounds: 1, Quality: 3, SortNo: 9,
		Des: "反击回合+%d（发动后本场战斗的反击回合数增加，与「绝地逆袭」叠加）"},
	// —— 持续型（2 回合 → 数值压一档，满级 120%）——
	{ID: 10, Name: "全线总攻", Kind: 1, BaseValue: 12, GrowPerLevel: 12, MaxLevel: 10, LevelPerStep: 35, ProcPct: 30, Rounds: 2, Quality: 4, SortNo: 10,
		Des: "攻击力+%d%%（发动后持续 2 回合，覆盖面更久）"},

	// ============ 2026-10-31 扩充：按品质各 +6 条 ============
	//
	// 命名口径（用户要求）：「二战战争类、著名战役里面的**修饰词**也行，不能就是著名战役」——
	// 所以不直接用「斯大林格勒/中途岛」这类战役名，而是「钢铁洪流 / 钳形攻势 / 空降突击」这种战争打法词。
	//
	// 战法类型（TriggerType，参考率土之滨的四类）：
	//   1 被动：必定发动、整场常驻（Rounds=0）→ 数值压到主动的 1/4 左右
	//   2 指挥：战前掷一次，命中后整场（Rounds=0）→ 数值约为主动的 2/3
	//   3 主动：每回合掷骰（原 10 条同款）
	//   4 追击：每次普攻命中后掷骰追加一击（Kind=10）→ 数值约为主动的 4 成
	//
	// —— 精良（Quality 2）——
	{ID: 11, Name: "战备动员", Kind: 1, BaseValue: 4, GrowPerLevel: 4, MaxLevel: 10, LevelPerStep: 35, ProcPct: 100, Rounds: 0, Quality: 2, SortNo: 11, TriggerType: 1,
		Des: "攻击力+%d%%（被动：战斗开始即生效，整场常驻）"},
	{ID: 12, Name: "铁壁工事", Kind: 2, BaseValue: 4, GrowPerLevel: 4, MaxLevel: 10, LevelPerStep: 35, ProcPct: 100, Rounds: 0, Quality: 2, SortNo: 12, TriggerType: 1,
		Des: "防御力+%d%%（被动：战斗开始即生效，整场常驻）"},
	{ID: 13, Name: "摩托突进", Kind: 4, BaseValue: 8, GrowPerLevel: 8, MaxLevel: 10, LevelPerStep: 35, ProcPct: 45, Rounds: 0, Quality: 2, SortNo: 13, TriggerType: 2,
		Des: "速度+%d%%（指挥：战前掷骰，命中后整场生效）"},
	{ID: 14, Name: "装甲奔袭", Kind: 4, BaseValue: 18, GrowPerLevel: 18, MaxLevel: 10, LevelPerStep: 35, ProcPct: 35, Rounds: 1, Quality: 2, SortNo: 14, TriggerType: 3,
		Des: "速度+%d%%"},
	{ID: 15, Name: "远程压制", Kind: 3, BaseValue: 18, GrowPerLevel: 18, MaxLevel: 10, LevelPerStep: 35, ProcPct: 30, Rounds: 1, Quality: 2, SortNo: 15, TriggerType: 3,
		Des: "射程+%d%%"},
	{ID: 16, Name: "侧翼伏击", Kind: 10, BaseValue: 7, GrowPerLevel: 7, MaxLevel: 10, LevelPerStep: 35, ProcPct: 30, Rounds: 1, Quality: 2, SortNo: 16, TriggerType: 4,
		Des: "追击（每次普攻命中后概率发动，对被普攻的目标追加一次战法伤害，伤害为普攻的 %d%%）"},

	// —— 稀有（Quality 3）——
	{ID: 17, Name: "要塞防御", Kind: 2, BaseValue: 5, GrowPerLevel: 5, MaxLevel: 10, LevelPerStep: 35, ProcPct: 100, Rounds: 0, Quality: 3, SortNo: 17, TriggerType: 1,
		Des: "防御力+%d%%（被动：战斗开始即生效，整场常驻）"},
	{ID: 18, Name: "钢筋掩体", Kind: 5, BaseValue: 8, GrowPerLevel: 8, MaxLevel: 10, LevelPerStep: 35, ProcPct: 40, Rounds: 0, Quality: 3, SortNo: 18, TriggerType: 2,
		Des: "受到伤害-%d%%（指挥：战前掷骰，命中后整场生效）"},
	{ID: 19, Name: "空降突击", Kind: 1, BaseValue: 12, GrowPerLevel: 12, MaxLevel: 10, LevelPerStep: 35, ProcPct: 40, Rounds: 0, Quality: 3, SortNo: 19, TriggerType: 2,
		Des: "攻击力+%d%%（指挥：战前掷骰，命中后整场生效）"},
	{ID: 20, Name: "破甲突击", Kind: 6, BaseValue: 8, GrowPerLevel: 8, MaxLevel: 10, LevelPerStep: 35, ProcPct: 35, Rounds: 1, Quality: 3, SortNo: 20, TriggerType: 3,
		Des: "破防%d%%（发动后本回合攻击无视目标 %d%% 防御）"},
	{ID: 21, Name: "潜航猎杀", Kind: 10, BaseValue: 8, GrowPerLevel: 8, MaxLevel: 10, LevelPerStep: 35, ProcPct: 30, Rounds: 1, Quality: 3, SortNo: 21, TriggerType: 4,
		Des: "追击（每次普攻命中后概率发动，对被普攻的目标追加一次战法伤害，伤害为普攻的 %d%%）"},
	{ID: 22, Name: "梯次冲锋", Kind: 8, BaseValue: 18, GrowPerLevel: 18, MaxLevel: 10, LevelPerStep: 35, ProcPct: 30, Rounds: 1, Quality: 3, SortNo: 22, TriggerType: 3,
		Des: "连击（发动后本回合多打一次普攻，这次普攻伤害为原来的 %d%%）"},

	// —— 传说（Quality 4）——
	{ID: 23, Name: "钢铁洪流", Kind: 1, BaseValue: 6, GrowPerLevel: 6, MaxLevel: 10, LevelPerStep: 35, ProcPct: 100, Rounds: 0, Quality: 4, SortNo: 23, TriggerType: 1,
		Des: "攻击力+%d%%（被动：战斗开始即生效，整场常驻）"},
	{ID: 24, Name: "钳形攻势", Kind: 1, BaseValue: 18, GrowPerLevel: 18, MaxLevel: 10, LevelPerStep: 35, ProcPct: 45, Rounds: 0, Quality: 4, SortNo: 24, TriggerType: 2,
		Des: "攻击力+%d%%（指挥：战前掷骰，命中后整场生效）"},
	{ID: 25, Name: "饱和突击", Kind: 1, BaseValue: 22, GrowPerLevel: 22, MaxLevel: 10, LevelPerStep: 35, ProcPct: 35, Rounds: 1, Quality: 4, SortNo: 25, TriggerType: 3,
		Des: "攻击力+%d%%"},
	{ID: 26, Name: "纵深突破", Kind: 6, BaseValue: 10, GrowPerLevel: 10, MaxLevel: 10, LevelPerStep: 35, ProcPct: 35, Rounds: 1, Quality: 4, SortNo: 26, TriggerType: 3,
		Des: "破防%d%%（发动后本回合攻击无视目标 %d%% 防御）"},
	{ID: 27, Name: "乘胜追击", Kind: 10, BaseValue: 10, GrowPerLevel: 10, MaxLevel: 10, LevelPerStep: 35, ProcPct: 35, Rounds: 1, Quality: 4, SortNo: 27, TriggerType: 4,
		Des: "追击（每次普攻命中后概率发动，对被普攻的目标追加一次战法伤害，伤害为普攻的 %d%%）"},
	{ID: 28, Name: "闪击突进", Kind: 4, BaseValue: 18, GrowPerLevel: 18, MaxLevel: 10, LevelPerStep: 35, ProcPct: 45, Rounds: 0, Quality: 4, SortNo: 28, TriggerType: 2,
		Des: "速度+%d%%（指挥：战前掷骰，命中后整场生效）"},
}

// ezfyComboLegacyFP 一代「出厂数值」的指纹（base/grow/max/step 全等才算没被人改过）。
type ezfyComboLegacyFP struct {
	Base, Grow, Max, Step int
}

// ezfyGeneralSkillLegacy 历史版本的出厂数值，用于**一次性升级**到当前版本。
//
//	① 首版（2026-10-23 上午）：5 级 / 步长 70 / 数值 8+2 之类；
//	② 二代（2026-10-23 中午）：10 级 / 步长 35 / 数值 7+1（满级值不变那版）；
//
// 只有「完全等于某一代指纹」的行才会被升级 —— 管理端动过任何一个字段就不碰。
var ezfyGeneralSkillLegacy = map[int][]ezfyComboLegacyFP{
	1:  {{8, 2, 5, 70}, {7, 1, 10, 35}},
	2:  {{10, 3, 5, 70}, {13, 1, 10, 35}},
	3:  {{4, 1, 5, 70}, {4, 1, 10, 35}},
	4:  {{6, 2, 5, 70}, {5, 1, 10, 35}},
	5:  {{6, 2, 5, 70}, {5, 1, 10, 35}},
	6:  {{8, 3, 5, 70}, {11, 1, 10, 35}},
	7:  {{4, 2, 5, 70}, {3, 1, 10, 35}},
	8:  {{50, 5, 5, 70}, {52, 2, 10, 35}},
	9:  {{1, 1, 5, 70}},
	10: {{6, 2, 5, 70}, {5, 1, 10, 35}},
}

// ezfyDefaultComboOfGeneral 名将 → 组合技的**初始搭配**（只在「全服还没人配过」时灌一次）。
//
// 管理端可以随时改（名将编辑里的「组合技能」下拉），这里只是给线上 31 名名将
// 一个能立刻玩的默认值，避免功能上线后一片空白。
// key = ezfy_cfg_general.id（kind=2 名将），value = 组合技池 ID。
var ezfyDefaultComboOfGeneral = map[int]int{
	1:  1,  // Stalin(斯大林)    → 装甲突击
	18: 6,  // 朱可夫            → 穿甲火力
	9:  8,  // 巴顿              → 波次突击
	29: 3,  // 尼米兹            → 火力延伸
	22: 2,  // 蒙哥马利          → 纵深防御
	11: 4,  // 隆美尔            → 快速集群
	21: 5,  // 麦克阿瑟          → 筑垒地域
	12: 9,  // 崔可夫            → 死守反击
	15: 7,  // 马歇尔            → 野战医院
	25: 10, // 古德里安          → 全线总攻
}

// EnsureEzfyGeneralSkills 幂等补建组合技能池（全量 seed 与 skip 分支共用）。
//
//   - 表不存在 → 建表；
//   - 按 ID 只补「缺的那几条」，**不覆盖**管理端改过的名称/说明/数值/概率；
//   - 名将默认搭配只在「所有名将的 combo_skill_id 都是 0」时灌一次
//     （管理端配过任何一个就不再动，避免每次重启把人工配置改回去）。
func EnsureEzfyGeneralSkills(db *gorm.DB) {
	if !db.Migrator().HasTable("ezfy_cfg_general_skill") {
		if err := db.AutoMigrate(&model.EzfyCfgGeneralSkill{}); err != nil {
			log.Printf("【严重】名将组合技能池建表失败: %v", err)
			return
		}
	}
	for i := range ezfyGeneralSkillSeeds {
		s := ezfyGeneralSkillSeeds[i]
		var cnt int64
		db.Model(&model.EzfyCfgGeneralSkill{}).Where("id = ?", s.ID).Count(&cnt)
		if cnt > 0 {
			continue
		}
		s.Enabled = 1
		if s.LevelPerStep <= 0 {
			s.LevelPerStep = 35 // ★ 名将满级 350 级 → 35 级 1 档，正好 10 档吃满
		}
		if err := db.Create(&s).Error; err != nil {
			log.Printf("【严重】名将组合技能补建失败 id=%d: %v", s.ID, err)
			continue
		}
		// ★★ 2026-10-31 GORM 对「带 default 标签的零值字段」会跳过写入 →
		//   被动/指挥的 rounds=0（整场）会被写成默认值。这里显式补一次。
		if s.TriggerType == 1 || s.TriggerType == 2 {
			db.Model(&model.EzfyCfgGeneralSkill{}).Where("id = ?", s.ID).Update("rounds", 0)
		}
	}
	upgradeLegacyEzfyGeneralSkills(db)
	// ★★ 2026-10-31 组合技改名（率土风 → 二战风）：表是「按 ID 只补缺、不覆盖」，
	//   改种子名字落不到老库 → 用 settings 打标一次性改名（只改仍是出厂名的行）。
	renameEzfyGeneralSkills(db)
	// ★★ 2026-10-31 二次改名：新增的 18 条最初用了战役名（斯大林格勒/中途岛…），
	//   用户要求「不能就是著名战役」→ 改成战争打法词。只对**已插入过**的库生效（线上还没插 → 空跑）。
	renameEzfyComboBatch2(db)
	// ★★ 2026-10-31 修正存量：被动/指挥的「持续」必须是 0（整场），早期插入的写成了 1。
	fixEzfyComboWholeBattleRounds(db)
	// 名将默认搭配：只在「一个都没配」时灌（幂等闸门）
	// ★ 2026-10-24 多技能：combo_skill_ids（JSON 数组）也计入「已配过」闸门
	var assigned int64
	db.Model(&model.EzfyCfgGeneral{}).
		Where("kind = 2 AND (combo_skill_id > 0 OR (combo_skill_ids IS NOT NULL AND combo_skill_ids <> '' AND combo_skill_ids <> '[]'))").
		Count(&assigned)
	if assigned > 0 {
		return
	}
	for gid, sid := range ezfyDefaultComboOfGeneral {
		db.Model(&model.EzfyCfgGeneral{}).
			Where("id = ? AND kind = 2", gid).
			Updates(map[string]interface{}{
				"combo_skill_ids": fmt.Sprintf("[%d]", sid),
				"combo_skill_id":  0,
			})
	}
}

// upgradeLegacyEzfyGeneralSkills 把历史版本（5 级/70、10 级/35 两代）的出厂数值
// 一次性升级到当前版本（10 级 / 步长 35 / 数值放大到普通技能量级）。
//
// 幂等：升级后数值就不再等于任何一代指纹，下次启动匹配不到 → 什么都不做。
// 安全：管理端改过数值/步长/上限的行（指纹不匹配）**一律不碰**，
// 避免把人调过的平衡改回去（多机共享库同理，先跑到的机器升级，其余机器匹配不到）。
func upgradeLegacyEzfyGeneralSkills(db *gorm.DB) {
	for i := range ezfyGeneralSkillSeeds {
		n := ezfyGeneralSkillSeeds[i]
		fps, ok := ezfyGeneralSkillLegacy[n.ID]
		if !ok {
			continue
		}
		for _, fp := range fps {
			// 已经是当前版本（四元组全等）→ 跳过
			if fp.Base == n.BaseValue && fp.Grow == n.GrowPerLevel &&
				fp.Max == n.MaxLevel && fp.Step == n.LevelPerStep {
				continue
			}
			res := db.Model(&model.EzfyCfgGeneralSkill{}).
				Where("id = ? AND base_value = ? AND grow_per_level = ? AND max_level = ? AND level_per_step = ?",
					n.ID, fp.Base, fp.Grow, fp.Max, fp.Step).
				Updates(map[string]interface{}{
					"base_value":     n.BaseValue,
					"grow_per_level": n.GrowPerLevel,
					"max_level":      n.MaxLevel,
					"level_per_step": n.LevelPerStep,
				})
			if res.Error != nil {
				log.Printf("【严重】名将组合技能升级失败 id=%d: %v", n.ID, res.Error)
				break
			}
			if res.RowsAffected > 0 {
				log.Printf("名将组合技能「%s」已升级：%d/%+d 上限 %d 级 步长 %d（旧版 %d/%+d 上限 %d 步长 %d）",
					n.Name, n.BaseValue, n.GrowPerLevel, n.MaxLevel, n.LevelPerStep,
					fp.Base, fp.Grow, fp.Max, fp.Step)
				break // 升完就不用再匹配其它代指纹
			}
		}
	}
}

// renameEzfyGeneralSkills 把 10 条名将组合技从「率土风」改成「二战/战争风」名字（只跑一次）。
//
// ★★ 2026-10-31 用户需求：「名将组合技能 名字都不像是二战的技能，优化下，要战争类型的名字，
// 之前是模仿率土」。
//
// 旧名 → 新名（效果与 ID 不变，只有名字）：
//
//	1 铁血军魂→装甲突击（攻击）   2 不动如山→纵深防御（防御）   3 长弓远略→火力延伸（射程）
//	4 疾风迅雷→快速集群（速度）   5 铜墙铁壁→筑垒地域（减伤）   6 摧锋破甲→穿甲火力（破防）
//	7 战地回春→野战医院（治疗）   8 连环突击→波次突击（连击）   9 血战到底→死守反击（反击回合）
//	10 万钧之势→全线总攻（攻击·2 回合）
//
// ⚠️ 只改**仍然是出厂旧名**的行（`WHERE id=? AND name=?`）：
// 管理端自己改过名字的行一律不碰，避免把人工配置冲掉。名将搭配存的是 ID（combo_skill_ids），
// 改名不影响任何已配好的组合。
func renameEzfyGeneralSkills(db *gorm.DB) {
	if !db.Migrator().HasTable("ezfy_cfg_general_skill") || !db.Migrator().HasTable("settings") {
		return
	}
	const key = "ezfy_general_skill_rename_20261031"
	var marked int64
	db.Raw("SELECT COUNT(*) FROM settings WHERE `key` = ?", key).Scan(&marked)
	if marked > 0 {
		return
	}
	renames := []struct {
		id       int
		from, to string
	}{
		{1, "铁血军魂", "装甲突击"},
		{2, "不动如山", "纵深防御"},
		{3, "长弓远略", "火力延伸"},
		{4, "疾风迅雷", "快速集群"},
		{5, "铜墙铁壁", "筑垒地域"},
		{6, "摧锋破甲", "穿甲火力"},
		{7, "战地回春", "野战医院"},
		{8, "连环突击", "波次突击"},
		{9, "血战到底", "死守反击"},
		{10, "万钧之势", "全线总攻"},
	}
	done := 0
	for _, it := range renames {
		res := db.Exec("UPDATE ezfy_cfg_general_skill SET name = ? WHERE id = ? AND name = ?",
			it.to, it.id, it.from)
		if res.Error != nil {
			log.Printf("ezfy 组合技改名失败 id=%d %s→%s: %v（下次启动重试）", it.id, it.from, it.to, res.Error)
			return
		}
		done += int(res.RowsAffected)
	}
	db.Exec("INSERT INTO settings (`key`, `value`) VALUES (?, ?)", key, "1")
	log.Printf("ezfy 名将组合技改名：%d 条已换成二战风名字", done)
}

// renameEzfyComboBatch2 把「2026-10-31 新增的 18 条组合技」从战役名改成战争打法词（只跑一次）。
//
// 用户原话：「命名属于二战战争类、著名战役里面的修饰词也行，不能就是著名战役啊」。
//
//	11 巴巴罗萨→战备动员   12 铁砧行动→铁壁工事   13 火炬行动→摩托突进   14 沙漠之狐→装甲奔袭
//	15 中途岛→远程压制     16 瓜岛伏击→侧翼伏击   17 库尔斯克→要塞防御   18 大西洋壁垒→钢筋掩体
//	19 市场花园→空降突击   20 斯大林格勒→破甲突击 21 潜艇伏击→潜航猎杀   22 诺曼底登陆→梯次冲锋
//	23 台风行动→钢铁洪流   24 天王星行动→钳形攻势 25 堡垒行动→饱和突击   26 巴格拉季昂→纵深突破
//	27 硫磺岛→乘胜追击     28 霸王行动→闪击突进
//
// ⚠️ 只改「仍然是那批战役名」的行（WHERE id=? AND name=?），管理端改过名字的不碰。
// 线上库还没插过这批（insert-only 只在缺行时补），所以这条迁移在线上是空跑 —— 留着是为了
// 开发/测试机（已经按战役名插过一次）也能自愈到最终命名。
func renameEzfyComboBatch2(db *gorm.DB) {
	if !db.Migrator().HasTable("ezfy_cfg_general_skill") || !db.Migrator().HasTable("settings") {
		return
	}
	const key = "ezfy_general_skill_rename_batch2_20261031"
	var marked int64
	db.Raw("SELECT COUNT(*) FROM settings WHERE `key` = ?", key).Scan(&marked)
	if marked > 0 {
		return
	}
	renames := []struct {
		id       int
		from, to string
	}{
		{11, "巴巴罗萨", "战备动员"},
		{12, "铁砧行动", "铁壁工事"},
		{13, "火炬行动", "摩托突进"},
		{14, "沙漠之狐", "装甲奔袭"},
		{15, "中途岛", "远程压制"},
		{16, "瓜岛伏击", "侧翼伏击"},
		{17, "库尔斯克", "要塞防御"},
		{18, "大西洋壁垒", "钢筋掩体"},
		{19, "市场花园", "空降突击"},
		{20, "斯大林格勒", "破甲突击"},
		{21, "潜艇伏击", "潜航猎杀"},
		{22, "诺曼底登陆", "梯次冲锋"},
		{23, "台风行动", "钢铁洪流"},
		{24, "天王星行动", "钳形攻势"},
		{25, "堡垒行动", "饱和突击"},
		{26, "巴格拉季昂", "纵深突破"},
		{27, "硫磺岛", "乘胜追击"},
		{28, "霸王行动", "闪击突进"},
	}
	done := 0
	for _, it := range renames {
		res := db.Exec("UPDATE ezfy_cfg_general_skill SET name = ? WHERE id = ? AND name = ?",
			it.to, it.id, it.from)
		if res.Error != nil {
			log.Printf("ezfy 组合技二次改名失败 id=%d %s→%s: %v（下次启动重试）", it.id, it.from, it.to, res.Error)
			return
		}
		done += int(res.RowsAffected)
	}
	db.Exec("INSERT INTO settings (`key`, `value`) VALUES (?, ?)", key, "1")
	log.Printf("ezfy 名将组合技二次改名：%d 条战役名已换成战争打法词", done)
}

// fixEzfyComboWholeBattleRounds 把被动/指挥类组合技的「持续」修正为 0（整场）。
//
// ★★ 2026-10-31 踩坑：`rounds` 原来带 `default:1` 标签，GORM 对零值会跳过写入 →
// 被动/指挥（设计上整场）插进库变成「持续 1 回合」，管理端还显示「1 回合」。
// 现在模型默认值改成 0 + 插入后显式补写；这条迁移负责把**已经插过的行**改回来。
func fixEzfyComboWholeBattleRounds(db *gorm.DB) {
	if !db.Migrator().HasTable("ezfy_cfg_general_skill") || !db.Migrator().HasTable("settings") {
		return
	}
	const key = "ezfy_combo_whole_battle_rounds_20261031"
	var marked int64
	db.Raw("SELECT COUNT(*) FROM settings WHERE `key` = ?", key).Scan(&marked)
	if marked > 0 {
		return
	}
	res := db.Exec("UPDATE ezfy_cfg_general_skill SET rounds = 0 WHERE trigger_type IN (1,2) AND rounds <> 0")
	if res.Error != nil {
		log.Printf("ezfy 组合技持续回合修正失败：%v（下次启动重试）", res.Error)
		return
	}
	db.Exec("INSERT INTO settings (`key`, `value`) VALUES (?, ?)", key, "1")
	log.Printf("ezfy 组合技：%d 条被动/指挥的持续回合已改为 0（整场）", res.RowsAffected)
}
