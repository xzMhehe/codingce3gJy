package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// ============ 军衔晋升宝物（2026-09-28 用户要求） ============
//
// 参考原版《晋升军衔宝物参考.xlsx》：军衔晋升不再只看声望 —— 声望达标是前提，
// 还需**提交**该军衔所需的宝物数量才能真正晋升。
// 宝物 = 野地采集掉落的 9 种珠宝（装备配置 ID 27-35，见 ezfyTerrainTreasureNames）。
//
// xlsx 原版只有 10 档（对应城市 7~16），游戏里军衔是 20 级（上等兵~五星上将 全都要宝物）：
//   - 第 2~11 级（上等兵~大尉）直接沿用原版 10 档的**增量需求**数值；
//   - 第 12 级起（少校~五星上将）在原版最高档（4 种×40）之上逐级加码，
//     逐步把 9 种珠宝全部引入，直到五星上将九种各要 100。
var ezfyRankTreasures = map[int][]ezfyRankTreasure{
	2:  {{Name: "蓝宝石戒指", Count: 10}, {Name: "红宝石戒指", Count: 5}},
	3:  {{Name: "红宝石戒指", Count: 15}, {Name: "祖母绿", Count: 10}},
	4:  {{Name: "黑曜石戒指", Count: 30}, {Name: "琥珀项链", Count: 20}},
	5:  {{Name: "铂金戒指", Count: 30}, {Name: "黄金手镯", Count: 20}},
	6:  {{Name: "玛瑙项坠", Count: 30}, {Name: "翡翠项链", Count: 30}},
	7:  {{Name: "蓝宝石戒指", Count: 40}, {Name: "红宝石戒指", Count: 20}},
	8:  {{Name: "黑曜石戒指", Count: 40}, {Name: "祖母绿", Count: 40}},
	9:  {{Name: "黑曜石戒指", Count: 40}, {Name: "琥珀项链", Count: 40}, {Name: "铂金戒指", Count: 40}, {Name: "黄金手镯", Count: 30}},
	10: {{Name: "琥珀项链", Count: 40}, {Name: "铂金戒指", Count: 40}, {Name: "黄金手镯", Count: 40}, {Name: "玛瑙项坠", Count: 40}},
	11: {{Name: "铂金戒指", Count: 40}, {Name: "黄金手镯", Count: 40}, {Name: "玛瑙项坠", Count: 40}, {Name: "翡翠项链", Count: 40}},
	12: {{Name: "蓝宝石戒指", Count: 50}, {Name: "红宝石戒指", Count: 40}, {Name: "祖母绿", Count: 50}},
	13: {{Name: "黑曜石戒指", Count: 70}, {Name: "琥珀项链", Count: 50}, {Name: "铂金戒指", Count: 60}, {Name: "黄金手镯", Count: 40}},
	14: {{Name: "玛瑙项坠", Count: 60}, {Name: "翡翠项链", Count: 60}, {Name: "蓝宝石戒指", Count: 50}, {Name: "红宝石戒指", Count: 40}},
	15: {{Name: "黑曜石戒指", Count: 70}, {Name: "琥珀项链", Count: 60}, {Name: "黄金手镯", Count: 50}, {Name: "祖母绿", Count: 50}},
	16: {{Name: "铂金戒指", Count: 70}, {Name: "玛瑙项坠", Count: 70}, {Name: "翡翠项链", Count: 60}, {Name: "红宝石戒指", Count: 50}},
	17: {{Name: "黑曜石戒指", Count: 80}, {Name: "琥珀项链", Count: 80}, {Name: "铂金戒指", Count: 70}, {Name: "黄金手镯", Count: 60}, {Name: "玛瑙项坠", Count: 60}},
	18: {{Name: "翡翠项链", Count: 80}, {Name: "蓝宝石戒指", Count: 70}, {Name: "红宝石戒指", Count: 70}, {Name: "祖母绿", Count: 70}},
	19: {{Name: "黑曜石戒指", Count: 60}, {Name: "琥珀项链", Count: 60}, {Name: "铂金戒指", Count: 60}, {Name: "黄金手镯", Count: 60}, {Name: "玛瑙项坠", Count: 60}, {Name: "翡翠项链", Count: 60}, {Name: "蓝宝石戒指", Count: 60}, {Name: "红宝石戒指", Count: 60}, {Name: "祖母绿", Count: 60}},
	20: {{Name: "黑曜石戒指", Count: 100}, {Name: "琥珀项链", Count: 100}, {Name: "铂金戒指", Count: 100}, {Name: "黄金手镯", Count: 100}, {Name: "玛瑙项坠", Count: 100}, {Name: "翡翠项链", Count: 100}, {Name: "蓝宝石戒指", Count: 100}, {Name: "红宝石戒指", Count: 100}, {Name: "祖母绿", Count: 100}},
}

// ezfyRankTreasure 单个宝物的晋升需求
type ezfyRankTreasure struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// ezfyParseRankTreasures 解析军衔配置里的宝物 JSON（[{"name","count"}]）
func ezfyParseRankTreasures(raw string) []ezfyRankTreasure {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var reqs []ezfyRankTreasure
	if err := json.Unmarshal([]byte(raw), &reqs); err != nil {
		return nil
	}
	return reqs
}

// ezfyRankTreasureReqs 某军衔的宝物需求（★ 数据库配置优先，管理端军衔配置可维护；
// 配置为空/解析失败时回落内置默认 ezfyRankTreasures，保证逻辑永远可用；
// 管理端显式清空（存 []）则视为该军衔无需宝物）
func ezfyRankTreasureReqs(rankID int) []ezfyRankTreasure {
	for _, r := range ezfyCfg.rankList() {
		if r.ID == rankID {
			if reqs := ezfyParseRankTreasures(r.Treasures); reqs != nil {
				return reqs
			}
			break
		}
	}
	return ezfyRankTreasures[rankID]
}

// ============ 一次性迁移：已晋升的不补宝物 + 宝物需求落表（2026-09-28） ============

var ezfyRankInitOnce sync.Once

// ezfyMigrateRankInit 幂等迁移，进程内只跑一次：
//
//  1. 老玩家（rank=0，还没写过晋升等级）按当前声望推导的军衔**直接写入 rank** ——
//     已晋升的就晋升了，部署后不用补宝物，从下一级开始才需要提交宝物晋升；
//  2. 军衔宝物配置（ezfy_cfg_rank.treasures）空行回填内置默认，
//     之后管理端在「军衔配置」里可以直接维护。
func ezfyMigrateRankInit(db *gorm.DB) {
	// 1) 老玩家 rank 落位（保证展示与旧声望规则一致，不缩水、不补宝物）
	// ★ `rank` 是 MySQL 保留字（窗口函数 RANK），WHERE 里必须用反引号包列名，否则启动报
	//  "Error 1064 ... near '= 0'"，这条迁移永远跑不成功。
	var profs []model.EzfyProfile
	db.Select("id", "prestige", "rank").Where("`rank` = 0").Find(&profs)
	done := 0
	for _, p := range profs {
		lv := ezfyRankIndex(p.Prestige) + 1
		if err := db.Model(&model.EzfyProfile{}).Where("id = ?", p.ID).Update("rank", lv).Error; err != nil {
			log.Printf("ezfy 军衔迁移失败 uid=%d: %v", p.UserID, err)
			continue
		}
		done++
	}
	if done > 0 {
		log.Printf("ezfy 军衔迁移: %d 名老玩家已晋升等级落位（无需补宝物）", done)
	}
	// 2) 宝物需求回填（只补空行，管理端改过的值不动）
	backfill := 0
	for id, reqs := range ezfyRankTreasures {
		if len(reqs) == 0 {
			continue
		}
		b, err := json.Marshal(reqs)
		if err != nil {
			continue
		}
		res := db.Model(&model.EzfyCfgRank{}).
			Where("id = ? AND (treasures IS NULL OR treasures = '')", id).
			Update("treasures", string(b))
		if res.Error != nil {
			log.Printf("ezfy 军衔宝物回填失败 rank=%d: %v", id, res.Error)
			continue
		}
		backfill += int(res.RowsAffected)
	}
	if backfill > 0 {
		log.Printf("ezfy 军衔迁移: %d 档军衔宝物需求已落表", backfill)
	}
}

// ezfyTreasureBagOnce 一次性迁移：历史「宝物签到」误发到道具表(ezfy_item)的宝物 → 装备表(ezfy_equipment)
//
// ★ 2026-09-28 修复「签到宝物没到账」：签到原来用 addItem 发进道具表，
//   而宝物配置 27-35 是装备配置（采集掉宝进的是装备表）——道具表里既没名字、
//   军衔晋升也统计不到，玩家自然「感觉没到账」。这里把存量一次性转正，
//   之后签到直接走 addEquipment，不会再产生这类脏数据。
var ezfyTreasureBagOnce sync.Once

func ezfyMigrateTreasureBag(db *gorm.DB) {
	var items []model.EzfyItem
	db.Where("cfg_id >= ? AND cfg_id <= ? AND count > 0", 27, 35).Find(&items)
	moved := 0
	for _, it := range items {
		cfg, ok := ezfyCfg.equipments[it.CfgId]
		if !ok {
			continue
		}
		for i := 0; i < it.Count; i++ {
			eq := model.EzfyEquipment{
				UserId: it.UserId, CfgId: cfg.ID, Name: cfg.Name,
				Type: cfg.Type, Tier: cfg.Tier, Military: cfg.Military, Logistics: cfg.Logistics,
				Learning: cfg.Learning, Level: cfg.Level, OfficerId: 0, CreatedAt: time.Now(),
				Slot: model.EzfySlotCanon(cfg.EquipSlot()), SetId: cfg.SetId,
				Series: cfg.Series, Enhance: cfg.Enhance,
				Dmg: cfg.Dmg, Def: cfg.Def, Hp: cfg.Hp, Move: cfg.Move, Crit: cfg.Crit, CritDmg: cfg.CritDmg,
			}
			if err := db.Create(&eq).Error; err == nil {
				moved++
			}
		}
		db.Where("id = ?", it.ID).Delete(&model.EzfyItem{})
	}
	if moved > 0 {
		log.Printf("ezfy 宝物迁移: %d 件历史签到宝物已转入装备表(背包可见/可晋升)", moved)
	}
}

// ezfyEquipCfgByName 装备配置名 → 配置（宝物按名字引用，不硬编码 ID，避免与种子两处不一致）
func ezfyEquipCfgByName(name string) *model.EzfyCfgEquipment {
	for _, e := range ezfyCfg.equipments {
		if e.Name == name {
			return &e
		}
	}
	return nil
}

// ezfyTreasureOwned 玩家背包里某宝物的数量（只统计**未穿戴**的：已装备在军官身上的宝物不能提交）
func (h *EzfyHandler) ezfyTreasureOwned(uid uint, cfgID int) int64 {
	var n int64
	h.DB.Model(&model.EzfyEquipment{}).
		Where("user_id = ? AND cfg_id = ? AND officer_id = 0", uid, cfgID).Count(&n)
	return n
}

// ezfyTreasureListText 宝物需求可读文本：黑曜石戒指×40、琥珀项链×20
func ezfyTreasureListText(reqs []ezfyRankTreasure) string {
	parts := make([]string, 0, len(reqs))
	for _, r := range reqs {
		parts = append(parts, fmt.Sprintf("%s×%d", r.Name, r.Count))
	}
	return strings.Join(parts, "、")
}

// Promote POST /games/ezfy/promote —— 晋升军衔
//
// 规则（2026-09-28 用户要求）：声望达标只是前提，还要**提交**所晋升军衔的宝物。
// 宝物从背包（未穿戴）扣除，成功后军衔 +1 并广播全服。
func (h *EzfyHandler) Promote(c *gin.Context) {
	h.cfgs()
	uid := middleware.GetUID(c)
	// 晋升连点可能重复扣宝物：复用 addPrestige 的玩家锁串行化
	mu := &ezfyPrestigeLocks[uid%64]
	mu.Lock()
	defer mu.Unlock()

	profile := h.ensureProfile(uid)
	ranks := ezfyCfg.rankList()
	curLv := ezfyProfileRank(&profile)
	if curLv >= len(ranks) {
		resp.ParamError(c, fmt.Sprintf("已晋升至最高军衔「%s」，无法再晋升", ranks[len(ranks)-1].Name))
		return
	}
	// 军衔等级 = 列表下标 + 1（列表按 id 升序），下标 curLv 即下一级
	next := ranks[curLv]
	if profile.Prestige < next.NeedPrestige {
		resp.ParamError(c, fmt.Sprintf("声望不足：晋升「%s」需要 %d 声望（当前 %d）",
			next.Name, next.NeedPrestige, profile.Prestige))
		return
	}
	reqs := ezfyRankTreasureReqs(next.ID)
	treasureSet := ezfyCollectibleTreasureNames() // ★ 只有能采集的宝物（9 种珠宝）可以用于军衔晋升
	// 先整体校验，任一宝物不足都不扣减（避免扣一半失败）
	for _, r := range reqs {
		cfg := ezfyEquipCfgByName(r.Name)
		if cfg == nil {
			resp.ServerError(c, fmt.Errorf("宝物配置缺失:%s", r.Name))
			return
		}
		if !treasureSet[cfg.Name] {
			resp.ParamError(c, fmt.Sprintf("「%s」不是采集宝物，不可用于军衔晋升", r.Name))
			return
		}
		if h.ezfyTreasureOwned(uid, cfg.ID) < int64(r.Count) {
			resp.ParamError(c, fmt.Sprintf("宝物不足：晋升「%s」需要 %s×%d（背包现有 %d），宝物通过野地采集获得",
				next.Name, r.Name, r.Count, h.ezfyTreasureOwned(uid, cfg.ID)))
			return
		}
	}
	// 逐件扣减宝物（只扣未穿戴的）
	for _, r := range reqs {
		cfg := ezfyEquipCfgByName(r.Name)
		var ids []uint
		h.DB.Model(&model.EzfyEquipment{}).
			Where("user_id = ? AND cfg_id = ? AND officer_id = 0", uid, cfg.ID).
			Order("id ASC").Limit(r.Count).Pluck("id", &ids)
		if len(ids) > 0 {
			h.DB.Delete(&model.EzfyEquipment{}, ids)
		}
	}
	h.DB.Model(&model.EzfyProfile{}).Where("id = ?", profile.ID).Update("rank", next.ID)
	// 晋升全服播报（原 addPrestige 里的自动播报不再触发，见 ezfy.go addPrestige）
	h.ezfySysChat("恭喜玩家 %s 军衔晋升至 %s！", h.ezfyProfileName(uid), next.Name)
	resp.OK(c, gin.H{"rank_id": next.ID, "rank_name": next.Name, "rank_post": next.Post})
}
