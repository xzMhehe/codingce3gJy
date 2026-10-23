package seed

import (
	"log"

	"gorm.io/gorm"

	"qqjiayuan/server/internal/model"
)

// 名将组合技能池（★ 2026-10-23）
//
// 设计要点（详见仓库根目录 `1022名将组合技能.md`）：
//   - **系统专属**：普通军官（kind=1）学不到，只能由管理端把池子里的技能配给名将
//     （`ezfy_cfg_general.combo_skill_id`）；
//   - **通用**：一套效果对所有兵种生效（陆/海/空不区分），与战争系统的攻击/防御/射程/
//     速度/反击/治疗/连击 同一套参数，不做第二套数值体系；
//   - **按等级成长**：组合技等级由**军官等级**分档（默认每 35 级 +1，350 级 = 10 级），
//     效果值 = BaseValue + (等级-1)×GrowPerLevel；
//   - **概率发动**：每回合开始按 ProcPct 掷一次，命中后持续 Rounds 回合（1 = 只本回合）；
//   - **数值克制**：单技能满级不超过 ~25%（连击/反击这类按次数的除外），
//     避免「一个组合技顶一套装备」。
//
// ★ 2026-10-23 二次调整：等级上限 5 → 10、步长 70 → 35。
//
//	成长从 5 段切成 10 段，**满级值一分不动** —— 做法是把 BaseValue 下调
//	（满级值 = base + 9×grow，grow 统一 1，连击 2），只有「长弓远略」例外：
//	它满级只有 8%，10 段整数步长下最小能到 13%，仍远低于 25% 的安全线。
var ezfyGeneralSkillSeeds = []model.EzfyCfgGeneralSkill{
	// —— 增益类（本回合按概率发动，直接进战斗加成）——
	{ID: 1, Name: "铁血军魂", Kind: 1, BaseValue: 7, GrowPerLevel: 1, MaxLevel: 10, LevelPerStep: 35, ProcPct: 40, Rounds: 1, Quality: 2, SortNo: 1,
		Des: "攻击力+%d%%（发动后本回合全体部队攻击提升）"},
	{ID: 2, Name: "不动如山", Kind: 2, BaseValue: 13, GrowPerLevel: 1, MaxLevel: 10, LevelPerStep: 35, ProcPct: 35, Rounds: 1, Quality: 3, SortNo: 2,
		Des: "防御力+%d%%（发动后本回合全体部队防御提升）"},
	{ID: 3, Name: "长弓远略", Kind: 3, BaseValue: 4, GrowPerLevel: 1, MaxLevel: 10, LevelPerStep: 35, ProcPct: 30, Rounds: 1, Quality: 2, SortNo: 3,
		Des: "射程+%d%%（发动后本回合全体部队射程提升）"},
	{ID: 4, Name: "疾风迅雷", Kind: 4, BaseValue: 5, GrowPerLevel: 1, MaxLevel: 10, LevelPerStep: 35, ProcPct: 35, Rounds: 1, Quality: 2, SortNo: 4,
		Des: "速度+%d%%（发动后本回合全体部队移动速度提升）"},
	// —— 减伤 / 破防 ——
	{ID: 5, Name: "铜墙铁壁", Kind: 5, BaseValue: 5, GrowPerLevel: 1, MaxLevel: 10, LevelPerStep: 35, ProcPct: 40, Rounds: 1, Quality: 3, SortNo: 5,
		Des: "受到伤害-%d%%（发动后本回合受到的伤害降低）"},
	{ID: 6, Name: "摧锋破甲", Kind: 6, BaseValue: 11, GrowPerLevel: 1, MaxLevel: 10, LevelPerStep: 35, ProcPct: 35, Rounds: 1, Quality: 4, SortNo: 6,
		Des: "破防%d%%（发动后本回合攻击无视目标 %d%% 防御）"},
	// —— 治疗 ——
	{ID: 7, Name: "战地回春", Kind: 7, BaseValue: 3, GrowPerLevel: 1, MaxLevel: 10, LevelPerStep: 35, ProcPct: 30, Rounds: 1, Quality: 3, SortNo: 7,
		Des: "回合治疗%d%%（发动回合结束时恢复本回合已损兵力的 %d%%）"},
	// —— 连击（每级 +2，10 级 52→70%）——
	{ID: 8, Name: "连环突击", Kind: 8, BaseValue: 52, GrowPerLevel: 2, MaxLevel: 10, LevelPerStep: 35, ProcPct: 30, Rounds: 1, Quality: 4, SortNo: 8,
		Des: "连击（发动后本回合每次攻击追加一次，追加伤害为原来的 %d%%）"},
	// —— 反击回合（★ 次数型：上限仍 5 级，10 级就是 +10 回合，太变态）——
	{ID: 9, Name: "血战到底", Kind: 9, BaseValue: 1, GrowPerLevel: 1, MaxLevel: 5, LevelPerStep: 35, ProcPct: 30, Rounds: 1, Quality: 3, SortNo: 9,
		Des: "反击回合+%d（发动后本场战斗的反击回合数增加，与「绝地反击」叠加）"},
	// —— 持续型（多回合）——
	{ID: 10, Name: "万钧之势", Kind: 1, BaseValue: 5, GrowPerLevel: 1, MaxLevel: 10, LevelPerStep: 35, ProcPct: 30, Rounds: 2, Quality: 4, SortNo: 10,
		Des: "攻击力+%d%%（发动后持续 2 回合，覆盖面更久）"},
}

// ezfyGeneralSkillLegacy 首版（2026-10-23 上午）的「5 级 / 步长 70」出厂数值。
//
// 只用于**一次性升级**：库里还是这套原封不动的数值（说明管理端没改过）就自动
// 改成上面的 10 级配置；管理端动过任何一个字段（数值/步长/上限）就不碰。
// key = 技能 ID，value = {BaseValue, GrowPerLevel, MaxLevel}。
var ezfyGeneralSkillLegacy = map[int][3]int{
	1:  {8, 2, 5},
	2:  {10, 3, 5},
	3:  {4, 1, 5},
	4:  {6, 2, 5},
	5:  {6, 2, 5},
	6:  {8, 3, 5},
	7:  {4, 2, 5},
	8:  {50, 5, 5},
	9:  {1, 1, 5},
	10: {6, 2, 5},
}

// ezfyDefaultComboOfGeneral 名将 → 组合技的**初始搭配**（只在「全服还没人配过」时灌一次）。
//
// 管理端可以随时改（名将编辑里的「组合技能」下拉），这里只是给线上 31 名名将
// 一个能立刻玩的默认值，避免功能上线后一片空白。
// key = ezfy_cfg_general.id（kind=2 名将），value = 组合技池 ID。
var ezfyDefaultComboOfGeneral = map[int]int{
	1:  1,  // Stalin(斯大林)    → 铁血军魂
	18: 6,  // 朱可夫            → 摧锋破甲
	9:  8,  // 巴顿              → 连环突击
	29: 3,  // 尼米兹            → 长弓远略
	22: 2,  // 蒙哥马利          → 不动如山
	11: 4,  // 隆美尔            → 疾风迅雷
	21: 5,  // 麦克阿瑟          → 铜墙铁壁
	12: 9,  // 崔可夫            → 血战到底
	15: 7,  // 马歇尔            → 战地回春
	25: 10, // 古德里安          → 万钧之势
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
		}
	}
	upgradeLegacyEzfyGeneralSkills(db)
	// 名将默认搭配：只在「一个都没配」时灌（幂等闸门）
	var assigned int64
	db.Model(&model.EzfyCfgGeneral{}).Where("kind = 2 AND combo_skill_id > 0").Count(&assigned)
	if assigned > 0 {
		return
	}
	for gid, sid := range ezfyDefaultComboOfGeneral {
		db.Model(&model.EzfyCfgGeneral{}).
			Where("id = ? AND kind = 2", gid).
			Update("combo_skill_id", sid)
	}
}

// upgradeLegacyEzfyGeneralSkills 把首版「5 级 / 步长 70」的出厂数值一次性升到 10 级。
//
// 幂等：升级后数值就不再等于 legacy 指纹，下次启动匹配不到 → 什么都不做。
// 安全：管理端改过数值/步长/上限的行（指纹不匹配）**一律不碰**，
// 避免把人调过的平衡改回去（多机共享库同理，先跑到的机器升级，其余机器匹配不到）。
func upgradeLegacyEzfyGeneralSkills(db *gorm.DB) {
	for i := range ezfyGeneralSkillSeeds {
		n := ezfyGeneralSkillSeeds[i]
		old, ok := ezfyGeneralSkillLegacy[n.ID]
		if !ok {
			continue
		}
		// 指纹 = 老的 base/grow/max + 步长 70。命中才动（连「数值没变、只是步长要从 70 改 35」
		// 的「血战到底」也会被带上，保证 10 个技能步长一致）。
		res := db.Model(&model.EzfyCfgGeneralSkill{}).
			Where("id = ? AND base_value = ? AND grow_per_level = ? AND max_level = ? AND level_per_step = ?",
				n.ID, old[0], old[1], old[2], 70).
			Updates(map[string]interface{}{
				"base_value":     n.BaseValue,
				"grow_per_level": n.GrowPerLevel,
				"max_level":      n.MaxLevel,
				"level_per_step": 35,
			})
		if res.Error != nil {
			log.Printf("【严重】名将组合技能升级 10 级失败 id=%d: %v", n.ID, res.Error)
			continue
		}
		if res.RowsAffected > 0 {
			log.Printf("名将组合技能「%s」已升级（上限 %d 级 / 步长 35，满级值不变）", n.Name, n.MaxLevel)
		}
	}
}
