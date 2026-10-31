package seed

import (
	"log"

	"gorm.io/gorm"

	"qqjiayuan/server/internal/ezfyreq"
	"qqjiayuan/server/internal/model"
)

// ★ 2026-10-31 科技改名（规避原版名称的版权风险）：旧名 → 新名。
//
// 新名统一取「二战时期通用技术名词」，与效果一一对应，不带任何厂商/型号/商标：
//
//	1粮食 2钢铁 3石油 4稀矿 5造兵速度 6攻击 7防御 8射程 9重装备 10速度 11建造
//	12情报 13负重 14容量 15携带 16城防 17掠夺 18耗粮 19空军速度 20城防修复 21伤兵恢复
//
// ★ 权威数据在 seed/ezfy_cfg_gen.go 的 ezfyEzfyCfgTech（科技主表 batch=UpdateAll，
// 每次启动自动对齐，所以**科技名本身不需要迁移**）。这里只补一处迁移：
// ezfy_cfg_troop.require 是 batchKeep（只补缺不覆盖），线上老库的文本仍是旧科技名。
var ezfyTechRenamePairs = [][2]string{
	{"种植技术", "合成氨工业"},
	{"炼钢技术", "合金冶炼"},
	{"勘探技术", "深井钻探"},
	{"冶炼技术", "钨矿精选"},
	{"军训艺术", "动员训练"},
	{"武器科技", "枪械改良"},
	{"装甲科技", "均质装甲"},
	{"弹道学", "炮兵射表"},
	{"重工技术", "重型机床"},
	{"燃烧引擎", "机械增压"},
	{"建筑技术", "预制构件"},
	{"侦察技巧", "无线电侦听"},
	{"装载技术", "标准化装卸"},
	{"储存技术", "野战仓储"},
	{"指挥艺术", "参谋作业"},
	{"掩体防御", "永备工事"},
	{"掠夺技巧", "战地征用"},
	{"补给技巧", "野战口粮"},
	{"喷气引擎", "喷气推进"},
	{"抢修技术", "工程抢修"},
	{"治愈伤兵", "战地医疗"},
}

const ezfyTechRenameKey = "ezfy_tech_rename_wwii_v1"

// alignEzfyTechRename 把 ezfy_cfg_troop.require 里残留的旧科技名换成新名（settings 打标，只跑一次）。
//
// 为什么必须做：改名后 require 文本里还是旧名，而 require 文本是 require_ids 的
// 解析来源（internal/ezfyreq.Parse 按名字查 id）—— 名字对不上就转不出结构化 ID，
// 训练门槛会缺一条。**校验本身已改为只读 require_ids**（见 ezfy.go 的
// ezfyTroopRequireMsg），所以这里纯粹是「把展示文本修对 + 让回填能解析成功」。
//
// 用 SQL REPLACE 而不是整列按内置值对齐：只动含旧名的行、只动旧名那一段，
// 管理员在「兵种配置」里改过的其它要求不会被覆盖。失败不写标记，下次启动重试。
func alignEzfyTechRename(db *gorm.DB) {
	if !db.Migrator().HasTable(&model.EzfyCfgTroop{}) || !db.Migrator().HasTable("settings") {
		return
	}
	var marked int64
	db.Raw("SELECT COUNT(*) FROM settings WHERE `key` = ?", ezfyTechRenameKey).Scan(&marked)
	if marked > 0 {
		return
	}
	var total int64
	for _, p := range ezfyTechRenamePairs {
		res := db.Exec(
			"UPDATE ezfy_cfg_troop SET `require` = REPLACE(`require`, ?, ?) WHERE `require` LIKE ?",
			p[0], p[1], "%"+p[0]+"%")
		if res.Error != nil {
			log.Printf("ezfy 科技改名：require 对齐失败 %s→%s（下次启动重试）: %v", p[0], p[1], res.Error)
			return
		}
		total += res.RowsAffected
	}
	db.Exec("INSERT INTO settings (`key`, `value`) VALUES (?, ?)", ezfyTechRenameKey, "1")
	if total > 0 {
		log.Printf("ezfy 科技改名：兵种 require 已对齐 %d 行", total)
	}
}

// backfillEzfyTroopRequireIDs 把 ezfy_cfg_troop.require 文本解析成结构化 ID 写进 require_ids。
//
// ★★ 2026-10-31 为什么必须做：训练校验现在**只读 require_ids**（见 ezfy.go 的
// ezfyTroopRequireMsg），文本只用来展示。之前是按名字反查 id —— 名字一改/打错/
// 全角半角不一致就查不到，门槛被静默跳过（玩家不研究科技也能造高级兵）。
//
// 每次启动都跑（幂等 + 自愈）：
//   - 老库从没填过 → 补齐；
//   - 管理端改了 require 文本、或从别处导入库 → 重算对齐；
//   - 算出来和库里一致 → 零写入。
//
// 解析不出名字（管理端手写错别字）的行只告警、保留原值，不阻塞启动 ——
// 那种行在校验侧会退回「按名字解析」，等价于改造前的行为，不会变得更糟。
func backfillEzfyTroopRequireIDs(db *gorm.DB) {
	if !db.Migrator().HasTable(&model.EzfyCfgTroop{}) {
		return
	}
	bIDs, tIDs := ezfyRequireNameIDs(db)
	if len(bIDs) == 0 && len(tIDs) == 0 {
		return
	}
	var troops []model.EzfyCfgTroop
	if err := db.Order("id").Find(&troops).Error; err != nil {
		log.Printf("ezfy require_ids 回填：读取兵种配置失败: %v", err)
		return
	}
	fixed := 0
	for _, t := range troops {
		refs, unknown := ezfyreq.Parse(t.Require, bIDs, tIDs)
		if len(unknown) > 0 {
			log.Printf("ezfy require_ids 回填：兵种 id=%d 的 require 有无法识别的名字 %v（保留原值）", t.ID, unknown)
			continue
		}
		want := ezfyreq.Format(refs)
		if want == t.RequireIds {
			continue
		}
		if err := db.Model(&model.EzfyCfgTroop{}).Where("id = ?", t.ID).
			Update("require_ids", want).Error; err != nil {
			log.Printf("ezfy require_ids 回填：兵种 id=%d 写入失败: %v", t.ID, err)
			continue
		}
		fixed++
	}
	if fixed > 0 {
		log.Printf("ezfy require_ids 回填：%d 个兵种已对齐（校验改为按 ID，不再依赖名字）", fixed)
	}
}

// ezfyRequireNameIDs 建筑名→ID、科技名→ID（require 文本解析用）。
// 两张表各自从 1 编号，所以解析结果必须带来源前缀（见 ezfyreq.Ref.Kind）。
func ezfyRequireNameIDs(db *gorm.DB) (map[string]int, map[string]int) {
	bIDs := map[string]int{}
	tIDs := map[string]int{}
	var bs []model.EzfyCfgBuilding
	if err := db.Select("id", "name").Find(&bs).Error; err == nil {
		for _, b := range bs {
			if b.Name != "" {
				bIDs[b.Name] = b.ID
			}
		}
	}
	var ts []model.EzfyCfgTech
	if err := db.Select("id", "name").Find(&ts).Error; err == nil {
		for _, t := range ts {
			if t.Name != "" {
				tIDs[t.Name] = t.ID
			}
		}
	}
	return bIDs, tIDs
}
