package handler

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 二战风云 军官/学院系统（忠实移植 stzb-fk：AcadeController + OfficerController + GameServiceImpl 军官段）
//
// 玩法要点：
//   - 军校(建筑9)等级 = 每日候选名将数量；参谋部(建筑10)等级 = 军官容量上限
//   - 招募费用 = 名将等级 × 500 黄金；同一名将全局只能拥有一个
//   - 忠诚：出征 -5、赏赐 +10(1万黄金)、归零自动离职
//   - 技能：最多 3 个，学习 1 万黄金/个，遗忘免费
//   - 装备：同部位唯一（珠宝不限），需军官等级 ≥ 装备需求等级
//   - 职位：市长(产量+10%+后勤/20)、城守(守城防御+10%)

const (
	ezfyRecruitRefreshLimit = 5     // 军校每日刷新次数上限
	ezfyRecruitCostPerLevel = 1000  // 招募费用 = 军官等级 × 该值(参考 conquer.html: 26级→26000)
	ezfyGrantCost           = 10000 // 赏赐一次消耗黄金
	ezfyLearnSkillCost      = 10000 // 学习技能消耗黄金
	ezfyOfficerMaxSkill     = 3     // 军官技能上限
	ezfyOfficerLoyaltyMax   = 100   // 忠诚上限
	ezfyCaptiveMinLoyalty   = 40    // 收编俘虏后的最低忠诚
	ezfyBuildingAcademy     = 9     // 军校
	ezfyBuildingStaff       = 10    // 参谋部
	ezfyPositionNone        = 0
	ezfyPositionMayor       = 1 // 市长
	ezfyPositionGuard       = 2 // 城守
)

// ezfyOfficerMaxLevel 军官最高等级（用户规则：「军官最高等级 150」）
//
// 与名将配置 ezfy_cfg_general.level 的上限一致（现有名将就是 110~150 级）。
// 所有会抬高军官等级的地方都要夹这个上限：
//
//	① 战斗加经验升级（addOfficerExp）
//	② 军校招募候选（rollOfficerDrafts）
//	③ 管理端一键生成军官（AdminEzfyGenOfficers）
//	④ 管理端直接编辑军官（AdminEzfyOfficerUpdate）
const ezfyOfficerMaxLevel = 150

// ezfyStarItemID 「军官升星卡」的道具 cfg_id（ItemType 19）
const ezfyStarItemID = 23

// ============ 基础查询 ============

// officerList 城市军官列表（自愈：出征中但已无对应行军命令的军官解除出征态）
func (h *EzfyHandler) officerList(cityId uint) []model.EzfyOfficer {
	var list []model.EzfyOfficer
	h.DB.Where("city_id = ?", cityId).Order("id ASC").Find(&list)
	orders := h.orderListByCity(cityId)
	for i := range list {
		o := &list[i]
		if o.Status != 1 {
			continue
		}
		busy := false
		for j := range orders {
			if orders[j].Officer == o.Name {
				busy = true
				break
			}
		}
		if !busy {
			o.Status = 0
			h.DB.Model(&model.EzfyOfficer{}).Where("id = ?", o.ID).Update("status", 0)
		}
	}
	return list
}

// orderListByCity 该城市尚未结束的行军命令（用于军官出征态自愈）
//
// ★ 必须含 2(返航中)：只写 (0,1) 会让「已踏上归途但还没到家」的军官被误判成空闲，
//
//	状态被自愈回 0 → 同一军官能被二次出征（用户反馈的 bug）。
func (h *EzfyHandler) orderListByCity(cityId uint) []model.EzfyOrder {
	var orders []model.EzfyOrder
	h.DB.Where("city_id = ? AND status IN (0,1,2)", cityId).Find(&orders)
	return orders
}

// officerBusyOrder 军官当前是否还有未结束的命令（0行军中/1驻守中/2返航中）
//
// 与 officer.Status 双保险：officer.Status 可能因历史数据漂移而不准，
// 这里直接按命令表判定，保证「没回来就不能再出征」。
func (h *EzfyHandler) officerBusyOrder(cityId uint, name string) bool {
	if name == "" {
		return false
	}
	for _, o := range h.orderListByCity(cityId) {
		if o.Officer == name {
			return true
		}
	}
	return false
}

func (h *EzfyHandler) officerOf(cityId uint, id int64) *model.EzfyOfficer {
	var o model.EzfyOfficer
	if err := h.DB.First(&o, id).Error; err != nil {
		return nil
	}
	if o.CityId != int64(cityId) {
		return nil
	}
	return &o
}

// officerCount 在职军官数（不含俘虏）
//
// ⚠️ 性能提示：本函数会查库。高频接口（如 View）请改用 officerCountOf，
// 把已经取到的军官列表传进去，避免重复查询。
func (h *EzfyHandler) officerCount(cityId uint) int {
	return officerCountOf(h.officerList(cityId))
}

// officerCountOf 按给定军官列表统计在职数（纯内存，不查库）。
func officerCountOf(list []model.EzfyOfficer) int {
	n := 0
	for i := range list {
		if list[i].IsCaptive != 1 {
			n++
		}
	}
	return n
}

// ownedGeneralIds 玩家已拥有的名将 ID（跨城统计，防重复招募）
func (h *EzfyHandler) ownedGeneralIds(uid uint) map[int]bool {
	owned := map[int]bool{}
	var cityIds []int64
	h.DB.Model(&model.EzfyCity{}).Where("user_id = ?", uid).Pluck("id", &cityIds)
	if len(cityIds) == 0 {
		return owned
	}
	var list []model.EzfyOfficer
	h.DB.Where("city_id IN ?", cityIds).Find(&list)
	for _, o := range list {
		if o.GeneralId > 0 {
			owned[o.GeneralId] = true
		}
	}
	return owned
}

// officerSkills 解析军官已学技能名
func officerSkills(o *model.EzfyOfficer) []string {
	out := []string{}
	if o == nil || o.Skill == "" {
		return out
	}
	_ = json.Unmarshal([]byte(o.Skill), &out)
	return out
}

// officerEquipped 解析军官已穿戴装备
func officerEquipped(o *model.EzfyOfficer) []map[string]interface{} {
	out := []map[string]interface{}{}
	if o == nil || o.Equipment == "" {
		return out
	}
	_ = json.Unmarshal([]byte(o.Equipment), &out)
	return out
}

// ============ 军校招募 ============

// ============ 军校招募：从军官池抽普通军官 ============
//
// ★ 2026-09-22 用户要求：
//   - 军官池 ezfy_cfg_general 里同时维护「普通军官(kind=1)」和「名将(kind=2)」；
//   - 军校招募/刷新**从池子里抽普通军官**（按 Weight 加权、不重复），
//     不再是每次现编随机名字 —— 管理端改了池子，玩家刷新出来的列表就跟着变；
//   - 名将仍然只能由管理端发放，不进招募池。

// ezfyOfficerDraft 军校招募候选（来自军官池的普通军官）
type ezfyOfficerDraft struct {
	Key       string `json:"key"`
	PoolId    int    `json:"pool_id"` // 军官池里的 ID（追溯/对账用）
	Name      string `json:"name"`
	Level     int    `json:"level"`
	Star      int    `json:"star"`
	Logistics int    `json:"logistics"`
	Military  int    `json:"military"`
	Learning  int    `json:"learning"`
	Cost      int64  `json:"cost"`
}

// ezfyOfficerFirstNames / ezfyOfficerLastNames 兜底随机名（军官池被清空时用）
var ezfyOfficerFirstNames = []string{
	"Pater", "Jeremy", "David", "Michael", "John", "Robert", "James", "William", "Charles", "Henry",
	"George", "Edward", "Frank", "Albert", "Arthur", "Walter", "Harold", "Ralph", "Roy", "Earl",
	"Bernard", "Clifford", "Norman", "Stanley", "Leonard", "Herbert", "Frederick", "Raymond", "Ernest", "Douglas",
}
var ezfyOfficerLastNames = []string{
	"Robinson", "Lee", "Smith", "Brown", "Wilson", "Taylor", "Clark", "Hall", "Young", "Wright",
	"King", "Scott", "Green", "Baker", "Adams", "Nelson", "Carter", "Mitchell", "Perez", "Roberts",
	"Turner", "Phillips", "Campbell", "Parker", "Evans", "Edwards", "Collins", "Stewart", "Morris", "Murphy",
}

// ezfyRollStar 星级概率: 5星3% 4星7% 3星20% 2星30% 1星40%
func ezfyRollStar() int {
	r := rand.Intn(100)
	switch {
	case r < 3:
		return 5
	case r < 10:
		return 4
	case r < 30:
		return 3
	case r < 60:
		return 2
	default:
		return 1
	}
}

// rollOfficerDrafts 从军官池抽 n 名普通军官当候选
//
// 抽不到（池子为空/池子被停用）时回落到「现场随机生成」，保证军校永远不空转。
func (h *EzfyHandler) rollOfficerDrafts(academyLevel, n int) []ezfyOfficerDraft {
	h.cfgs()
	if n <= 0 {
		n = 1
	}
	pool := ezfyCfg.poolOfficers()
	if len(pool) == 0 {
		return ezfyRandomDrafts(academyLevel, n)
	}
	// 加权前缀和（权重 <=0 按 1 算）
	cum := make([]int, len(pool))
	sum := 0
	for i, g := range pool {
		w := g.Weight
		if w <= 0 {
			w = 1
		}
		sum += w
		cum[i] = sum
	}
	pickOne := func() int {
		r := rand.Intn(sum)
		return sort.Search(len(cum), func(i int) bool { return cum[i] > r })
	}
	used := map[int]bool{}
	out := make([]ezfyOfficerDraft, 0, n)
	span := maxInt(1, academyLevel*8)
	for i := 0; i < n; i++ {
		idx := -1
		for try := 0; try < 60; try++ {
			cand := pickOne()
			if cand >= 0 && cand < len(pool) && !used[cand] {
				idx = cand
				break
			}
		}
		if idx < 0 {
			// 池子太小抽不出新的了，允许重复
			idx = pickOne()
			if idx < 0 || idx >= len(pool) {
				break
			}
		}
		used[idx] = true
		g := pool[idx]
		// 等级：随军校等级提高，但不超过该军官在池子里配的等级上限
		maxLv := g.Level
		if maxLv <= 0 || maxLv > ezfyOfficerMaxLevel {
			maxLv = ezfyOfficerMaxLevel
		}
		lv := 5 + rand.Intn(span)
		if lv > maxLv {
			lv = maxLv
		}
		if lv < 1 {
			lv = 1
		}
		out = append(out, ezfyOfficerDraft{
			Key:    g.Name + "-" + strconv.Itoa(g.ID) + "-" + strconv.FormatInt(time.Now().UnixNano()+int64(i), 10),
			PoolId: g.ID, Name: g.Name, Level: lv, Star: g.Star,
			Logistics: g.Logistics, Military: g.Military, Learning: g.Learning,
			Cost: int64(lv) * ezfyRecruitCostPerLevel,
		})
	}
	return out
}

// ezfyRandomDrafts 兜底：军官池为空时现场随机生成（老逻辑，保留避免军校空转）
func ezfyRandomDrafts(academyLevel, n int) []ezfyOfficerDraft {
	out := make([]ezfyOfficerDraft, 0, n)
	used := map[string]bool{}
	span := maxInt(1, academyLevel*8)
	for i := 0; i < n; i++ {
		name := ""
		for k := 0; k < 30; k++ {
			name = ezfyOfficerFirstNames[rand.Intn(len(ezfyOfficerFirstNames))] + "·" +
				ezfyOfficerLastNames[rand.Intn(len(ezfyOfficerLastNames))]
			if !used[name] {
				break
			}
		}
		used[name] = true
		lv := 5 + rand.Intn(span)
		if lv > ezfyOfficerMaxLevel {
			lv = ezfyOfficerMaxLevel
		}
		star := ezfyRollStar()
		base := 20 + star*5
		out = append(out, ezfyOfficerDraft{
			Key:  name + "-" + strconv.FormatInt(time.Now().UnixNano()+int64(i), 10),
			Name: name, Level: lv, Star: star,
			Logistics: base + rand.Intn(25),
			Military:  base + rand.Intn(25),
			Learning:  base + rand.Intn(25),
			Cost:      int64(lv) * ezfyRecruitCostPerLevel,
		})
	}
	return out
}

func joinDrafts(list []ezfyOfficerDraft) string {
	b, _ := json.Marshal(list)
	return string(b)
}

func parseDrafts(raw string) []ezfyOfficerDraft {
	out := []ezfyOfficerDraft{}
	if raw == "" {
		return out
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return []ezfyOfficerDraft{}
	}
	return out
}

// recruitInfo 当日候选(首次访问生成并落库)
func (h *EzfyHandler) recruitInfo(uid uint, academyLevel int) ([]ezfyOfficerDraft, int, int) {
	h.cfgs()
	date := time.Now().Format("2006-01-02")
	var rec model.EzfyRecruit
	err := h.DB.Where("user_id = ? AND recruit_date = ?", uid, date).First(&rec).Error
	if err != nil {
		drafts := h.rollOfficerDrafts(academyLevel, maxInt(1, minInt(academyLevel, 10)))
		rec = model.EzfyRecruit{UserId: uid, RecruitDate: date, RefreshCount: 0,
			Candidates: joinDrafts(drafts)}
		h.DB.Create(&rec)
	}
	// ★ 上限支持按玩家覆盖（管理端「军校免费刷次数」维护）
	limit := h.ezfyRecruitFreeLimit(uid)
	return parseDrafts(rec.Candidates), maxInt(0, limit-rec.RefreshCount), limit
}

func (h *EzfyHandler) refreshRecruit(uid uint, academyLevel int) string {
	date := time.Now().Format("2006-01-02")
	var rec model.EzfyRecruit
	err := h.DB.Where("user_id = ? AND recruit_date = ?", uid, date).First(&rec).Error
	used := 0
	if err == nil {
		used = rec.RefreshCount
	}
	limit := h.ezfyRecruitFreeLimit(uid)
	if used >= limit {
		return "今日刷新次数已用完(每天限" + strconv.Itoa(limit) +
			"次, 明天0点重置；也可以在军校直接使用「招生简章」刷新)"
	}
	drafts := h.rollOfficerDrafts(academyLevel, maxInt(1, minInt(academyLevel, 10)))
	if err != nil {
		rec = model.EzfyRecruit{UserId: uid, RecruitDate: date}
	}
	rec.RefreshCount = used + 1
	rec.Candidates = joinDrafts(drafts)
	if rec.ID == 0 {
		h.DB.Create(&rec)
	} else {
		h.DB.Model(&model.EzfyRecruit{}).Where("id = ?", rec.ID).
			Updates(map[string]interface{}{"refresh_count": rec.RefreshCount, "candidates": rec.Candidates})
	}
	return ""
}

// hireOfficerDraft 雇佣候选军官: 从当日候选里按 key 取, 校验容量/黄金后入库并从候选里移除
func (h *EzfyHandler) hireOfficerDraft(city *model.EzfyCity, uid uint, key string) string {
	h.calcResource(city)
	if h.buildingLevel(city.ID, ezfyBuildingAcademy) < 1 {
		return "需要先建造军校"
	}
	staff := h.buildingLevel(city.ID, ezfyBuildingStaff)
	if staff < 1 {
		return "需要先建造参谋部"
	}
	if h.officerCount(city.ID) >= staff {
		return "参谋部容量不足(参谋部" + strconv.Itoa(staff) + "级容纳" + strconv.Itoa(staff) + "名军官)"
	}
	date := time.Now().Format("2006-01-02")
	var rec model.EzfyRecruit
	if err := h.DB.Where("user_id = ? AND recruit_date = ?", uid, date).First(&rec).Error; err != nil {
		return "候选已失效, 请刷新"
	}
	drafts := parseDrafts(rec.Candidates)
	var pick *ezfyOfficerDraft
	kept := []ezfyOfficerDraft{}
	for i := range drafts {
		if drafts[i].Key == key && pick == nil {
			d := drafts[i]
			pick = &d
			continue
		}
		kept = append(kept, drafts[i])
	}
	if pick == nil {
		return "该候选不存在(可能已被雇佣或已刷新)"
	}
	if city.Gold < pick.Cost {
		return "黄金不足(雇佣需要" + strconv.FormatInt(pick.Cost, 10) + "黄金)"
	}
	city.Gold -= pick.Cost
	h.saveCityRes(city)
	// ★ 2026-09-22：原始属性写进 base_*（重修书洗点回退到这个值），
	//   并按「每级 1 点」补上该等级应有的可用属性点。
	//   普通军官的 GeneralId 保持 0（general_id>0 全站都当「名将」用，别混）。
	o := model.EzfyOfficer{
		CityId: int64(city.ID), GeneralId: 0, Name: pick.Name, Star: pick.Star,
		Level: pick.Level, Exp: 0,
		Military: pick.Military, Logistics: pick.Logistics, Learning: pick.Learning,
		BaseMilitary: pick.Military, BaseLogistics: pick.Logistics, BaseLearning: pick.Learning,
		FreePoints: maxInt(0, pick.Level-1),
		Loyalty:    ezfyOfficerLoyaltyMax, Skill: "", Equipment: "",
		Position: ezfyPositionNone, Status: 0, IsCaptive: 0, UpdateTime: time.Now(),
	}
	h.DB.Create(&o)
	h.DB.Model(&model.EzfyRecruit{}).Where("id = ?", rec.ID).Update("candidates", joinDrafts(kept))
	// ★ 五星军官值得全服看一眼（用户要求）
	if pick.Star >= 5 {
		h.ezfySysChat("恭喜玩家 %s 在军校招募到五星军官 %s！", h.ezfyProfileName(uid), o.Name)
	}
	return ""
}

// refreshRecruitFree 免费刷新当日候选名将（招生简章用，不消耗每日刷新次数）
func (h *EzfyHandler) refreshRecruitFree(uid uint) string {
	city := h.getOrCreateCity(uid)
	academy := h.buildingLevel(city.ID, ezfyBuildingAcademy)
	if academy < 1 {
		return "需要先建造军校"
	}
	date := time.Now().Format("2006-01-02")
	var rec model.EzfyRecruit
	err := h.DB.Where("user_id = ? AND recruit_date = ?", uid, date).First(&rec).Error
	drafts := h.rollOfficerDrafts(academy, maxInt(1, minInt(academy, 10)))
	if err != nil {
		rec = model.EzfyRecruit{UserId: uid, RecruitDate: date, RefreshCount: 0}
	}
	rec.Candidates = joinDrafts(drafts)
	if rec.ID == 0 {
		h.DB.Create(&rec)
	} else {
		h.DB.Model(&model.EzfyRecruit{}).Where("id = ?", rec.ID).Update("candidates", rec.Candidates)
	}
	return ""
}

// ============ 军官操作 ============

// grantOfficer 赏赐：1万黄金 → 忠诚 +10
func (h *EzfyHandler) grantOfficer(city *model.EzfyCity, officerId int64) string {
	h.calcResource(city)
	o := h.officerOf(city.ID, officerId)
	if o == nil {
		return "武将不存在"
	}
	if o.Loyalty >= ezfyOfficerLoyaltyMax {
		return "忠诚已满"
	}
	if city.Gold < ezfyGrantCost {
		return "黄金不足(赏赐需要1万黄金)"
	}
	city.Gold -= ezfyGrantCost
	h.saveCityRes(city)
	h.DB.Model(&model.EzfyOfficer{}).Where("id = ?", o.ID).
		Update("loyalty", minInt(ezfyOfficerLoyaltyMax, o.Loyalty+10))
	return ""
}

// learnSkill 学习技能：1万黄金/个，最多 3 个，出征中不可学
func (h *EzfyHandler) learnSkill(city *model.EzfyCity, officerId int64, skillId int) string {
	h.calcResource(city)
	o := h.officerOf(city.ID, officerId)
	if o == nil {
		return "武将不存在"
	}
	if o.Status == 1 {
		return "武将出征中,无法学习"
	}
	cfg := ezfyCfg.skill(skillId)
	if cfg == nil {
		return "技能不存在"
	}
	skills := officerSkills(o)
	if len(skills) >= ezfyOfficerMaxSkill {
		return "技能已满(最多3个)"
	}
	for _, s := range skills {
		if s == cfg.Name {
			return "已学习该技能"
		}
	}
	if city.Gold < ezfyLearnSkillCost {
		return "黄金不足(学习技能需要1万黄金)"
	}
	city.Gold -= ezfyLearnSkillCost
	h.saveCityRes(city)
	skills = append(skills, cfg.Name)
	h.saveOfficerSkills(o, skills)
	return ""
}

// forgetSkill 遗忘技能（免费）
func (h *EzfyHandler) forgetSkill(city *model.EzfyCity, officerId int64, skillId int) string {
	o := h.officerOf(city.ID, officerId)
	if o == nil {
		return "武将不存在"
	}
	cfg := ezfyCfg.skill(skillId)
	if cfg == nil {
		return "技能不存在"
	}
	skills := officerSkills(o)
	kept := []string{}
	found := false
	for _, s := range skills {
		if s == cfg.Name {
			found = true
			continue
		}
		kept = append(kept, s)
	}
	if !found {
		return "未学习该技能"
	}
	h.saveOfficerSkills(o, kept)
	return ""
}

func (h *EzfyHandler) saveOfficerSkills(o *model.EzfyOfficer, skills []string) {
	b, _ := json.Marshal(skills)
	h.DB.Model(&model.EzfyOfficer{}).Where("id = ?", o.ID).Update("skill", string(b))
}

// ============ 装备 ============

func (h *EzfyHandler) equipmentList(uid uint) []model.EzfyEquipment {
	var list []model.EzfyEquipment
	h.DB.Where("user_id = ?", uid).Order("id ASC").Find(&list)
	return list
}

// ezfySlotCanon 部位别名归一（2026-09-23 用户反馈「同部位能穿多件」）
//
// ★ 根本原因：不同的套装对同一个身体部位用了**不同的字符串**——「头盔」和「头部」
//
//	都指头、「胸甲」和「胸部」都指胸、「手套/左手/手部」都指手…… 老代码只做**精确字符串**
//	判重，于是玩家能同时穿「传说英雄[头盔]」和「赤色锤镰[头部]」两件头装 → 同部位穿了两件。
//	这里把所有同名部位的书写统一成一个规范词，判重和落库都走它，才能真正做到「同部位唯一」。
func ezfySlotCanon(s string) string {
	switch s {
	case "头盔":
		return "头部"
	case "护肩":
		return "肩部"
	case "胸甲":
		return "胸部"
	case "手套", "左手":
		return "手部"
	case "战靴":
		return "足部"
	case "腰带":
		return "腰部"
	}
	return s
}

// addEquipment 生成装备实例进背包
func (h *EzfyHandler) addEquipment(city *model.EzfyCity, cfg *model.EzfyCfgEquipment) {
	e := model.EzfyEquipment{
		UserId: city.UserID, CityId: int64(city.ID), CfgId: cfg.ID, Name: cfg.Name,
		Type: cfg.Type, Tier: cfg.Tier, Military: cfg.Military, Logistics: cfg.Logistics,
		Learning: cfg.Learning, Level: cfg.Level, OfficerId: 0, CreatedAt: time.Now(),
		// ★ 部位统一存归一后的规范名，老实例的原始字符串由判重时归一兜底
		Slot: ezfySlotCanon(cfg.EquipSlot()), SetId: cfg.SetId,
		// ★ 六项战斗属性随实例带走（进战斗计算用）
		Series: cfg.Series, Enhance: cfg.Enhance,
		Dmg: cfg.Dmg, Def: cfg.Def, Hp: cfg.Hp, Move: cfg.Move, Crit: cfg.Crit, CritDmg: cfg.CritDmg,
	}
	h.DB.Create(&e)
}

// equipItem 穿戴装备：等级达标 + 同部位唯一（珠宝不限，可以叠）
//
// ★ 2026-09-22：同部位判定改用「Slot（留空回落 Type）」，
// 这样套装里的头/肩/胸/腰/手/足/饰品/挂件/勋章 9 件互不冲突，能整套穿上。
func (h *EzfyHandler) equipItem(city *model.EzfyCity, officerId, equipId int64) string {
	h.calcResource(city)
	o := h.officerOf(city.ID, officerId)
	if o == nil {
		return "武将不存在"
	}
	var e model.EzfyEquipment
	if err := h.DB.First(&e, equipId).Error; err != nil || e.UserId != city.UserID {
		return "装备不存在"
	}
	if e.OfficerId > 0 {
		return "装备已被穿戴"
	}
	if o.Level < e.Level {
		return "武将等级不足(需要" + strconv.Itoa(e.Level) + "级)"
	}
	slot := ezfySlotCanon(e.EquipSlot())
	equipped := officerEquipped(o)
	if slot != "珠宝" {
		for _, m := range equipped {
			// ★ 2026-09-23 修复「同部位能穿多件」：判重前先把**双方**的部位别名归一。
			//   否则「传说英雄[头盔]」(部位'头盔') 和 「赤色锤镰[头部]」(部位'头部')
			//   这种同名部位不同写法会同时通过，导致一个部位穿了两件。
			if t, _ := m["slot"].(string); ezfySlotCanon(t) == slot {
				return "已穿戴同部位装备(" + slot + ")"
			}
			// 老数据没有 slot 字段 → 回落到 type
			if t, _ := m["slot"].(string); t == "" {
				if ot, _ := m["type"].(string); ezfySlotCanon(ot) == slot {
					return "已穿戴同部位装备(" + slot + ")"
				}
			}
		}
	}
	// ★ 顺序很重要：先把军官身上的装备列表写成功，再改装备行的 officer_id。
	//   反过来的话（先改 officer_id 再写 JSON），一旦 JSON 写失败就会留下
	//   「装备显示已穿戴、但军官身上没有」的半截状态 —— 实测踩过（varchar(500) 截断）。
	equipped = append(equipped, map[string]interface{}{
		"id": e.ID, "name": e.Name, "type": e.Type, "slot": slot, "set_id": e.SetId,
		"military": e.Military, "logistics": e.Logistics, "learning": e.Learning,
		// ★ 六项战斗属性（进战斗计算）
		"series": e.Series, "enhance": e.Enhance,
		"dmg": e.Dmg, "def": e.Def, "hp": e.Hp, "move": e.Move, "crit": e.Crit, "crit_dmg": e.CritDmg,
	})
	if msg := h.saveOfficerEquipment(o, equipped); msg != "" {
		return msg
	}
	h.DB.Model(&model.EzfyEquipment{}).Where("id = ?", e.ID).Update("officer_id", int64(o.ID))
	return ""
}

// unequipItem 卸下装备
func (h *EzfyHandler) unequipItem(city *model.EzfyCity, equipId int64) string {
	var e model.EzfyEquipment
	if err := h.DB.First(&e, equipId).Error; err != nil || e.UserId != city.UserID {
		return "装备不存在"
	}
	if e.OfficerId <= 0 {
		return "装备未穿戴"
	}
	if o := h.officerOf(city.ID, e.OfficerId); o != nil {
		kept := []map[string]interface{}{}
		for _, m := range officerEquipped(o) {
			if id, ok := m["id"].(float64); ok && int64(id) == int64(e.ID) {
				continue
			}
			kept = append(kept, m)
		}
		if msg := h.saveOfficerEquipment(o, kept); msg != "" {
			return msg
		}
	}
	h.DB.Model(&model.EzfyEquipment{}).Where("id = ?", e.ID).Update("officer_id", 0)
	return ""
}

// saveOfficerEquipment 写回军官的已穿戴装备 JSON
//
// ★ 返回错误字符串（而不是静默忽略）：这一列曾经是 varchar(500)，
// 穿到第 5 件就写不进去（MySQL 1406），当时没人看返回值 →
// 装备行已标成「已穿戴」但军官身上没有，玩家看到「穿了没效果」。现在会直接报错。
func (h *EzfyHandler) saveOfficerEquipment(o *model.EzfyOfficer, list []map[string]interface{}) string {
	b, err := json.Marshal(list)
	if err != nil {
		return "装备数据序列化失败：" + err.Error()
	}
	if err := h.DB.Model(&model.EzfyOfficer{}).Where("id = ?", o.ID).
		Update("equipment", string(b)).Error; err != nil {
		return "保存已穿戴装备失败：" + err.Error()
	}
	return ""
}

// equipmentOwnerId 查询某装备穿戴者（卸下后跳转详情用）
func (h *EzfyHandler) equipmentOwnerId(uid uint, equipId int64) int64 {
	var e model.EzfyEquipment
	if err := h.DB.First(&e, equipId).Error; err != nil || e.UserId != uid {
		return 0
	}
	return e.OfficerId
}

// ============ 任命 / 俘虏 / 流放 ============

// setOfficerPosition 任命：1市长 2城守 0卸任（同职位唯一）
func (h *EzfyHandler) setOfficerPosition(city *model.EzfyCity, officerId int64, position int) string {
	o := h.officerOf(city.ID, officerId)
	if o == nil {
		return "武将不存在"
	}
	if position < 0 || position > 2 {
		return "职位错误"
	}
	if o.Position == position && position != ezfyPositionNone {
		return "已是该职位"
	}
	if position != ezfyPositionNone {
		h.DB.Model(&model.EzfyOfficer{}).
			Where("city_id = ? AND position = ? AND id <> ?", city.ID, position, o.ID).
			Update("position", ezfyPositionNone)
	}
	h.DB.Model(&model.EzfyOfficer{}).Where("id = ?", o.ID).Update("position", position)
	return ""
}

// recruitCaptive 收编俘虏（忠诚至少 40，占用参谋部容量）
func (h *EzfyHandler) recruitCaptive(city *model.EzfyCity, officerId int64) string {
	o := h.officerOf(city.ID, officerId)
	if o == nil {
		return "武将不存在"
	}
	if o.IsCaptive != 1 {
		return "该武将不是俘虏"
	}
	if o.Status == 1 {
		return "出征中无法收编"
	}
	staff := h.buildingLevel(city.ID, ezfyBuildingStaff)
	if staff < 1 {
		return "需要先建造参谋部"
	}
	if h.officerCount(city.ID) >= staff {
		return "参谋部容量不足(参谋部" + strconv.Itoa(staff) + "级容纳" + strconv.Itoa(staff) + "名军官)"
	}
	loyalty := o.Loyalty
	if loyalty < ezfyCaptiveMinLoyalty {
		loyalty = ezfyCaptiveMinLoyalty
	}
	h.DB.Model(&model.EzfyOfficer{}).Where("id = ?", o.ID).
		Updates(map[string]interface{}{"is_captive": 0, "loyalty": loyalty, "update_time": time.Now()})
	return ""
}

// freeOfficer 释放俘虏（删除记录）
func (h *EzfyHandler) freeOfficer(city *model.EzfyCity, officerId int64) string {
	o := h.officerOf(city.ID, officerId)
	if o == nil {
		return "武将不存在"
	}
	if o.Status == 1 {
		return "出征中无法遣散"
	}
	h.DB.Delete(&model.EzfyOfficer{}, o.ID)
	return ""
}

// exileOfficer 流放（需先卸任）
func (h *EzfyHandler) exileOfficer(city *model.EzfyCity, officerId int64) string {
	o := h.officerOf(city.ID, officerId)
	if o == nil {
		return "武将不存在"
	}
	if o.Status == 1 {
		return "出征中无法流放"
	}
	if o.Position != ezfyPositionNone {
		return "请先卸任市长/城守再流放"
	}
	h.DB.Delete(&model.EzfyOfficer{}, o.ID)
	return ""
}

// ============ 加成接入 ============

// mayorBonusPct 市长产量加成 %（10 + 后勤/20）
func (h *EzfyHandler) mayorBonusPct(cityId uint) int {
	var o model.EzfyOfficer
	if err := h.DB.Where("city_id = ? AND position = ? AND is_captive = 0", cityId, ezfyPositionMayor).
		First(&o).Error; err != nil {
		return 0
	}
	// ★ 用有效后勤（自身 + 装备），否则给市长穿后勤装备没有任何效果
	_, log, _ := h.officerEffective(&o)
	return 10 + log/20
}

// officerByName 按名字取本城军官
func (h *EzfyHandler) officerByName(cityId uint, name string) *model.EzfyOfficer {
	if name == "" {
		return nil
	}
	var o model.EzfyOfficer
	if err := h.DB.Where("city_id = ? AND name = ?", cityId, name).First(&o).Error; err != nil {
		return nil
	}
	return &o
}

// positionOfficer 取本城某职位的军官（1市长 2城守）
func (h *EzfyHandler) positionOfficer(cityId uint, position int) *model.EzfyOfficer {
	var o model.EzfyOfficer
	if err := h.DB.Where("city_id = ? AND position = ?", cityId, position).First(&o).Error; err != nil {
		return nil
	}
	return &o
}

// officerOnDutyList 可带队的军官（在职、未出征、非俘虏）
func (h *EzfyHandler) officerOnDutyList(cityId uint) []model.EzfyOfficer {
	out := []model.EzfyOfficer{}
	for _, o := range h.officerList(cityId) {
		if o.Status == 0 && o.IsCaptive != 1 {
			out = append(out, o)
		}
	}
	return out
}

// officerHasSkill 军官是否已学某技能
func (h *EzfyHandler) officerHasSkill(o *model.EzfyOfficer, skill string) bool {
	if o == nil {
		return false
	}
	for _, s := range officerSkills(o) {
		if s == skill {
			return true
		}
	}
	return false
}

// officerEquipBonus 汇总军官已穿戴装备的属性加成
//
// ★ 之前装备只算了「军事」一项，后勤/学识的加成**完全没有任何去处**，
//
//	玩家穿上带后勤/学识的装备后数字一动不动，看起来就是「穿装备没效果」。
func (h *EzfyHandler) officerEquipBonus(o *model.EzfyOfficer) (mil, log, lea int) {
	for _, m := range officerEquipped(o) {
		mil += jsonInt(m["military"])
		log += jsonInt(m["logistics"])
		lea += jsonInt(m["learning"])
	}
	return
}

// officerSetBonus 套装加成：统计已穿戴装备里各套装的件数，达到 parts 就触发
//
// ★ 用户规则（2026-09-22）：**套装效果只有穿齐才生效**；只穿一件或几件，
// 只算那几件装备本身的属性加成，不给任何套装加成。所以这里的门槛是 `cnt >= s.Parts`。
//
// 返回 (军事, 后勤, 学识, 已触发套装说明)。装备里的 set_id 由 equipItem 写入；
// 老数据没有 set_id → 计 0，不会误触发。
func (h *EzfyHandler) officerSetBonus(o *model.EzfyOfficer) (mil, log, lea int, active []string) {
	if o == nil {
		return
	}
	for _, s := range h.officerSetProgress(o) {
		if s.Active {
			mil += s.Set.Military
			log += s.Set.Logistics
			lea += s.Set.Learning
			active = append(active, s.Set.Name+"("+strconv.Itoa(s.Worn)+"/"+strconv.Itoa(s.Set.Parts)+"件)")
		}
	}
	return
}

// ezfySetProgress 某套装的穿戴进度（用于界面展示「还差几件」）
type ezfySetProgress struct {
	Set    *model.EzfyCfgEquipSet
	Worn   int  // 已穿件数
	Active bool // 是否已穿齐（套装效果是否生效）
}

// officerSetProgress 军官身上各套装的穿戴进度（只列至少穿了 1 件的套装）
func (h *EzfyHandler) officerSetProgress(o *model.EzfyOfficer) []ezfySetProgress {
	out := []ezfySetProgress{}
	if o == nil {
		return out
	}
	h.cfgs()
	cnt := map[int]int{}
	for _, m := range officerEquipped(o) {
		if sid := jsonInt(m["set_id"]); sid > 0 {
			cnt[sid]++
		}
	}
	ids := make([]int, 0, len(cnt))
	for sid := range cnt {
		ids = append(ids, sid)
	}
	sort.Ints(ids)
	for _, sid := range ids {
		s := ezfyCfg.equipSet(sid)
		if s == nil || s.Parts <= 0 {
			continue
		}
		out = append(out, ezfySetProgress{Set: s, Worn: cnt[sid], Active: cnt[sid] >= s.Parts})
	}
	return out
}

// officerEffective 军官的**有效属性**（自身 + 装备 + 套装）
func (h *EzfyHandler) officerEffective(o *model.EzfyOfficer) (mil, log, lea int) {
	if o == nil {
		return
	}
	em, el, ee := h.officerEquipBonus(o)
	sm, sl, se, _ := h.officerSetBonus(o)
	return o.Military + em + sm, o.Logistics + el + sl, o.Learning + ee + se
}

// ezfyBattleBonus 一方在战斗中的六项加成（单位：百分点，100 = +100%）
//
// ★ 2026-09-22 参照 装备距离伤害表.xlsx：
//
//	基础攻击力 = 基础攻击属性 × (1 + 科技加成) × (1 + 装备伤害加成)
//	总攻击力   = 基础攻击力 × (1 + 暴击伤害加成)
//
// 装备的「伤害/防御/生命/移动距离/暴击几率/暴击伤害」全部是**百分比加成**。
type ezfyBattleBonus struct {
	Dmg     int // 伤害加成%
	Def     int // 防御加成%
	Hp      int // 生命加成%
	Move    int // 移动距离加成%
	Crit    int // 暴击几率加成%
	CritDmg int // 暴击伤害加成%
}

// officerBattleEquipBonus 汇总军官身上装备 + 已触发套装的六项战斗加成
//
// ★ 用户规则：**套装效果只有穿齐才生效**。
//   - 每一件装备自身的六项属性 → **永远生效**（穿一件就加一件）
//   - 套装配置里的额外加成 → 只有穿齐 `parts` 件才叠加
//
// 只算「已经穿在军官身上」的装备（老数据的 JSON 里没有这些字段 → 记 0，不会算错）。
func (h *EzfyHandler) officerBattleEquipBonus(o *model.EzfyOfficer) ezfyBattleBonus {
	b := ezfyBattleBonus{}
	if o == nil {
		return b
	}
	h.cfgs()
	for _, m := range officerEquipped(o) {
		b.Dmg += jsonInt(m["dmg"])
		b.Def += jsonInt(m["def"])
		b.Hp += jsonInt(m["hp"])
		b.Move += jsonInt(m["move"])
		b.Crit += jsonInt(m["crit"])
		b.CritDmg += jsonInt(m["crit_dmg"])
	}
	// 套装额外加成（★ 只有**穿齐**才加；只穿几件只算各件自身属性）
	for _, p := range h.officerSetProgress(o) {
		if !p.Active {
			continue
		}
		b.Dmg += p.Set.Dmg
		b.Def += p.Set.Def
		b.Hp += p.Set.Hp
		b.Move += p.Set.Move
		b.Crit += p.Set.Crit
		b.CritDmg += p.Set.CritDmg
	}
	return b
}

// officerBattleView 六项战斗加成的下发格式（列表/详情共用）
func (h *EzfyHandler) officerBattleView(o *model.EzfyOfficer) gin.H {
	b := h.officerBattleEquipBonus(o)
	return gin.H{
		"dmg": b.Dmg, "def": b.Def, "hp": b.Hp,
		"move": b.Move, "crit": b.Crit, "crit_dmg": b.CritDmg,
	}
}

// officerSetProgressView 套装穿戴进度（给前端展示「还差几件才生效」）
func (h *EzfyHandler) officerSetProgressView(o *model.EzfyOfficer) []gin.H {
	out := []gin.H{}
	for _, p := range h.officerSetProgress(o) {
		out = append(out, gin.H{
			"set_id": p.Set.ID, "name": p.Set.Name, "parts": p.Set.Parts,
			"worn": p.Worn, "active": p.Active, "need": maxInt(0, p.Set.Parts-p.Worn),
			"military": p.Set.Military, "logistics": p.Set.Logistics, "learning": p.Set.Learning,
			"dmg": p.Set.Dmg, "def": p.Set.Def, "hp": p.Set.Hp,
			"move": p.Set.Move, "crit": p.Set.Crit, "crit_dmg": p.Set.CritDmg,
			"effect": p.Set.Effect,
		})
	}
	return out
}

// officerBaseAttr 军官的**原始属性**（重修书洗点回退的目标）
//
// 优先用实例上的 base_*（招募时的快照，管理端改池子也不会影响已发出的军官）；
// 老数据 base_* 全 0 时，能对上军官池就用池子里的值，否则回落到当前属性。
func officerBaseAttr(o *model.EzfyOfficer) (int, int, int) {
	if o == nil {
		return 0, 0, 0
	}
	if o.BaseMilitary > 0 || o.BaseLogistics > 0 || o.BaseLearning > 0 {
		return o.BaseMilitary, o.BaseLogistics, o.BaseLearning
	}
	if g := ezfyCfg.general(o.GeneralId); g != nil && g.Military+g.Logistics+g.Learning > 0 {
		return g.Military, g.Logistics, g.Learning
	}
	return o.Military, o.Logistics, o.Learning
}

// officerAllocatedPoints 已经分配到三属性上的点数（当前 − 原始）
func officerAllocatedPoints(o *model.EzfyOfficer) int {
	bm, bl, be := officerBaseAttr(o)
	n := (o.Military - bm) + (o.Logistics - bl) + (o.Learning - be)
	if n < 0 {
		return 0
	}
	return n
}

// officerAddAttr 分配属性点：只写玩家自己的军官实例，**绝不回写军官池**
//
// attr: military / logistics / learning（也接受中文「军事/后勤/学识」）
func (h *EzfyHandler) officerAddAttr(city *model.EzfyCity, officerId int64, attr string, count int) string {
	o := h.officerOf(city.ID, officerId)
	if o == nil {
		return "军官不存在"
	}
	if o.IsCaptive == 1 {
		return "俘虏不能加点, 请先在军校收编"
	}
	if count <= 0 {
		count = 1
	}
	if count > 1000 {
		return "单次最多分配 1000 点"
	}
	if o.FreePoints <= 0 {
		return "没有可用属性点(每升 1 级得 1 点, 可用「重修书」重置已分配的点)"
	}
	if count > o.FreePoints {
		return "可用属性点不足(剩余" + strconv.Itoa(o.FreePoints) + "点)"
	}
	col := ""
	switch attr {
	case "military", "军事", "军":
		col = "military"
	case "logistics", "后勤", "后":
		col = "logistics"
	// ★ 第三项统一叫「学识」（别再叫「学习」—— 学习是「学技能」那个动作，容易混）
	case "learning", "学识", "学":
		col = "learning"
	default:
		return "属性类型错误(可选 军事/后勤/学识)"
	}
	if err := h.DB.Model(&model.EzfyOfficer{}).Where("id = ?", o.ID).
		Updates(map[string]interface{}{
			"free_points": o.FreePoints - count,
			col:           gorm.Expr(col+" + ?", count),
			"update_time": time.Now(),
		}).Error; err != nil {
		return "加点失败：" + err.Error()
	}
	return ""
}

// ezfyAttrToBonus 属性 → 加成百分点：floor((属性 + 1) / 2)
//
// ★ 复刻《战斗机制（家园玩家必看）》§6「军事和学识怎样变成攻防加成」：
//
//	攻击加成百分点 = floor(有效军事 + 1) ÷ 2
//	防御加成百分点 = floor(有效学识 + 1) ÷ 2
//
// 即**每 2 点属性 = 1 个百分点**（有效军事 300 → 150% 攻击加成）。
// 注意口径是「有效属性」（自身 + 装备 + 套装），见 officerEffective。
func ezfyAttrToBonus(attr int) int {
	if attr <= 0 {
		return 0
	}
	return (attr + 1) / 2
}

// officerBaseBonus 军官基础攻击加成（有效军事 ÷ 2，装备/套装军事已含在有效属性里）
//
// ★ 用户反馈（2026-09-22）：原来直接把军事值当百分点（军事 655 → 攻击+655%），
//
//	比参考文档高了一倍；现按 §6 改为 floor((有效军事+1)/2)。
func (h *EzfyHandler) officerBaseBonus(o *model.EzfyOfficer) int {
	if o == nil {
		return 0
	}
	mil, _, _ := h.officerEffective(o)
	return ezfyAttrToBonus(mil)
}

// officerSkillBattleBonus 军官技能带来的攻击加成（复刻原版 getOfficerBattleBonus 的技能段）
// 突击+10 鼓舞+5 爆破+8 空袭+8 海战+8 装甲突击+8
func (h *EzfyHandler) officerSkillBattleBonus(o *model.EzfyOfficer) int {
	if o == nil {
		return 0
	}
	bonus := 0
	for _, s := range officerSkills(o) {
		switch s {
		case "尖兵突击":
			bonus += 30
		case "火炮控制":
			bonus += 10
		case "四指编队", "狼群战术":
			bonus += 15
		}
	}
	return bonus
}

// officerBattleBonus 带队军官总攻击加成（军事 + 装备 + 技能）
func (h *EzfyHandler) officerBattleBonus(o *model.EzfyOfficer) int {
	return h.officerBaseBonus(o) + h.officerSkillBattleBonus(o)
}

// officerSpeedSkill 是否带行军/战斗速度类技能(坦克突袭/闪电袭击/越岛战术 任一)
func (h *EzfyHandler) officerSpeedSkill(o *model.EzfyOfficer) bool {
	return h.officerHasSkill(o, "坦克突袭") || h.officerHasSkill(o, "闪电袭击") || h.officerHasSkill(o, "越岛战术")
}

// officerGuardAttrBonus 军官**属性部分**的防御加成（有效学识 ÷ 2）
//
// ★ 复刻《战斗机制（家园玩家必看）》§6：防御加成百分点 = floor(有效学识 + 1) ÷ 2。
// 单独拆出来是给战报用的 —— officerBattleDesc 会自己再列技能，避免技能被算两遍。
func (h *EzfyHandler) officerGuardAttrBonus(o *model.EzfyOfficer) int {
	if o == nil {
		return 0
	}
	_, _, lea := h.officerEffective(o)
	return ezfyAttrToBonus(lea)
}

// officerGuardBonus 军官防御加成（有效学识 ÷ 2 + 技能）
//
// ★ 三项属性各有用途：军事→攻击、后勤→市长产量、学识→防御。
//
// ★ 用户反馈（2026-09-22）：原来是 `10 + 学识/20`（学识 376 → 防御+58），
//
//	比参考文档 §6 的 `floor((学识+1)/2)`（学识 376 → 防御+188）低了 3 倍多，
//	已按文档重写；固定 +10 基础值一并去掉（文档里没有这一项）。
func (h *EzfyHandler) officerGuardBonus(o *model.EzfyOfficer) int {
	if o == nil {
		return 0
	}
	bonus := h.officerGuardAttrBonus(o)
	for _, s := range officerSkills(o) {
		switch s {
		case "弧形防御":
			bonus += 30
		case "弹幕支援":
			bonus += 10
		}
	}
	return bonus
}

// officerReportDesc 战报中的军官行：名字(N级)
func officerReportDesc(o *model.EzfyOfficer) string {
	if o == nil {
		return ""
	}
	return o.Name + "(" + strconv.Itoa(o.Level) + "级)"
}

// officerBattleDesc 军官战斗作用描述（战报展示用）
func (h *EzfyHandler) officerBattleDesc(o *model.EzfyOfficer, baseBonus int, label string) string {
	if o == nil {
		return ""
	}
	sb := o.Name + " Lv." + strconv.Itoa(o.Level)
	parts := []string{}
	if baseBonus > 0 {
		parts = append(parts, label+"+"+strconv.Itoa(baseBonus)+"%")
	}
	for _, s := range officerSkills(o) {
		if eff := ezfySkillEffectText(s); eff != "" {
			parts = append(parts, s+"("+eff+")")
		}
	}
	if len(parts) > 0 {
		sb += " " + strings.Join(parts, " ")
	}
	return sb
}

// ezfySkillEffectText 技能在战斗/后勤中的作用文本
func ezfySkillEffectText(skill string) string {
	switch skill {
	case "尖兵突击":
		return "攻击力+30%"
	case "弧形防御":
		return "防御力+30%"
	case "绝地反击":
		return "第1回合反击"
	case "火炮控制":
		return "陆军装甲攻击+10"
	case "坦克突袭":
		return "陆军速度+10%"
	case "四指编队":
		return "空军对空攻击+15%"
	case "闪电袭击":
		return "空军速度+10%"
	case "狼群战术":
		return "海军对海攻击+15%"
	case "越岛战术":
		return "海军速度+10%"
	case "弹幕支援":
		return "城防攻击范围+10%"
	case "黄金眼":
		return "侦查等级+1"
	case "机械改造":
		return "回收率+10%, 出征油耗-10%"
	default:
		return ""
	}
}

func jsonInt(v interface{}) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	}
	return 0
}

// officerGoOut 军官出征/归来：出征置状态=1、归来置 0。
//
// ★ 第九轮用户规则：出征/派遣**不再扣忠诚**（旧版每次 -5，归零即离职，玩家反感）；
// 只有打败仗才扣，见 ezfy_order.go 的败仗结算 → officerLoseLoyalty。
func (h *EzfyHandler) officerGoOut(city *model.EzfyCity, name string, goOut bool) {
	o := h.officerByName(city.ID, name)
	if o == nil {
		return
	}
	if goOut {
		// ★ 第九轮用户规则：**派遣/出征不掉忠心**（原来每次 -5，归零就离职，玩家很反感）。
		//   只有打了败仗才掉，且掉的量按战损合理计算（见 officerLoseLoyalty / 战斗结算）。
		h.DB.Model(&model.EzfyOfficer{}).Where("id = ?", o.ID).Update("status", 1)
		return
	}
	h.DB.Model(&model.EzfyOfficer{}).Where("id = ?", o.ID).Update("status", 0)
}

// officerLoseLoyalty 扣军官忠心（归零自动离职）。
//
// 只在**打败仗**时调用；delta 由战损程度算出来（见 ezfy_order.go 的败仗结算）。
func (h *EzfyHandler) officerLoseLoyalty(uid uint, cityId uint, name string, delta int, reason string) {
	if name == "" || delta <= 0 {
		return
	}
	o := h.officerByName(cityId, name)
	if o == nil {
		return
	}
	loyalty := o.Loyalty - delta
	if loyalty < 0 {
		loyalty = 0
	}
	h.DB.Model(&model.EzfyOfficer{}).Where("id = ?", o.ID).Update("loyalty", loyalty)
	if loyalty <= 0 {
		h.DB.Delete(&model.EzfyOfficer{}, o.ID)
		h.addReport(uid, 6, "将领离职: "+o.Name,
			o.Name+"因连番战败、忠诚度归零而离职, 离开了你的城市。"+reason, "")
	}
}

// moveOfficerTo 军官随军调任到目标城市（职位清空）
func (h *EzfyHandler) moveOfficerTo(city *model.EzfyCity, name string, targetCityId uint) {
	o := h.officerByName(city.ID, name)
	if o == nil || o.Status == 2 {
		return
	}
	h.DB.Model(&model.EzfyOfficer{}).Where("id = ?", o.ID).
		Updates(map[string]interface{}{"city_id": int64(targetCityId), "position": 0, "status": 0, "update_time": time.Now()})
}

// addOfficerExp 军官获得经验（升级经验 = 等级×200）
//
// ★ 2026-09-22 用户规则：**每升 1 级给 1 点可用属性点，由玩家自己分配**
// （原来每级随机 +2 属性 → 玩家没得选，洗点后还会「都堆到学识上」，已废）。
// 加点只写玩家自己的军官实例，绝不回写军官池。
//
// ★ 用户规则「军官最高等级 150」：到 150 级后不再升级，多余经验直接丢弃
// （不丢的话经验会无限累积，将来放开上限会一次性跳很多级）。
func (h *EzfyHandler) addOfficerExp(city *model.EzfyCity, officerId uint, exp int64) {
	var o model.EzfyOfficer
	if err := h.DB.First(&o, officerId).Error; err != nil {
		return
	}
	o.Exp += exp
	gained := 0
	for o.Level < ezfyOfficerMaxLevel && o.Exp >= int64(o.Level)*200 {
		o.Exp -= int64(o.Level) * 200
		o.Level++
		gained++
	}
	// 满级后不保留经验
	if o.Level >= ezfyOfficerMaxLevel {
		o.Exp = 0
	}
	o.FreePoints += gained
	h.DB.Model(&model.EzfyOfficer{}).Where("id = ?", o.ID).
		Updates(map[string]interface{}{"exp": o.Exp, "level": o.Level, "free_points": o.FreePoints})
	if gained > 0 {
		h.addReport(city.UserID, 6, "将领升级: "+o.Name,
			o.Name+"在战斗中成长, 升到了"+strconv.Itoa(o.Level)+"级, 获得"+strconv.Itoa(gained)+
				"点属性点(可前往 [军官] 详情页分配)!", "")
	}
}

// OfficersOnDuty GET /games/ezfy/officers/onduty —— 出征界面可选的带队军官
func (h *EzfyHandler) OfficersOnDuty(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	city := h.getOrCreateCity(uid)
	h.refreshCity(uid, &city)
	list := []gin.H{}
	for _, o := range h.officerOnDutyList(city.ID) {
		list = append(list, gin.H{
			"id": o.ID, "name": o.Name, "level": o.Level, "star": o.Star,
			"military": o.Military, "logistics": o.Logistics, "learning": o.Learning,
			"loyalty":      o.Loyalty,
			"battle_bonus": h.officerBattleBonus(&o), "position_name": ezfyPositionName(o.Position),
			"skills": officerSkills(&o),
		})
	}
	resp.OK(c, gin.H{"officers": list, "hq_level": h.buildingLevel(city.ID, 13)})
}

// OfficerDispatch POST /games/ezfy/officers/:id/dispatch
//
// 城市列表的 [派遣]：把当前城市的某名军官调往自己的另一座城市。
// 规则：军官必须属于当前城、不在出征中、不是俘虏；目标城必须是自己的城且不是本城。
func (h *EzfyHandler) OfficerDispatch(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		CityId   int64 `json:"city_id"`   // 军官当前所在城（出发城）
		TargetId int64 `json:"target_id"` // 目标城
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	src := h.cityOf(uid, req.CityId)
	if src == nil {
		c2 := h.currentCity(uid)
		src = &c2
	}
	if src == nil || src.ID == 0 {
		resp.NotFound(c, "城市不存在")
		return
	}
	dst := h.cityOf(uid, req.TargetId)
	if dst == nil {
		resp.ParamError(c, "目标城市不存在或不属于你")
		return
	}
	if dst.ID == src.ID {
		resp.ParamError(c, "目标城市不能是当前城市")
		return
	}
	o := h.officerOf(src.ID, id)
	if o == nil {
		resp.NotFound(c, "军官不在该城市")
		return
	}
	if o.IsCaptive == 1 {
		resp.ParamError(c, "俘虏不能派遣, 请先在军校收编")
		return
	}
	if o.Status == 1 || h.officerBusyOrder(src.ID, o.Name) {
		resp.ParamError(c, "军官"+o.Name+"正在出征中, 未归队前不能派遣")
		return
	}
	// 带职位的军官（市长/城守）离开会让职位悬空，要求先卸任
	if o.Position != 0 {
		resp.ParamError(c, "请先卸任「"+ezfyPositionName(o.Position)+"」再派遣")
		return
	}
	if err := h.DB.Model(&model.EzfyOfficer{}).Where("id = ?", o.ID).
		Update("city_id", int64(dst.ID)).Error; err != nil {
		resp.ParamError(c, "派遣失败："+err.Error())
		return
	}
	h.addReport(uid, 5, "军官调遣",
		fmt.Sprintf("军官%s已从[%s]调往[%s](%d,%d)。", o.Name, src.Name, dst.Name, dst.X, dst.Y), "", 0)
	resp.OK(c, gin.H{"msg": fmt.Sprintf("%s 已调往 %s", o.Name, dst.Name)})
}

// ============ 野地掉宝 / 俘虏守将 ============

// wildlandLoot 战胜野地/寇城掉宝：按等级概率掉装备 + 按地形概率掉珠宝
func (h *EzfyHandler) wildlandLoot(city *model.EzfyCity, level, terrain int, special bool) string {
	h.cfgs()
	desc := ""
	roll := rand.Intn(100)
	dropChance, tier := 80, 1
	if level >= 3 && roll < 50 {
		tier = 2
	}
	if level >= 6 && roll < 20 {
		tier = 3
	}
	if level >= 9 && roll < 6 {
		tier = 4
	}
	if special {
		if tier < 3 {
			tier = 3
		}
		dropChance = 100
	}
	if roll < dropChance {
		if cfg := h.randomEquipment(tier); cfg != nil {
			h.addEquipment(city, cfg)
			desc += " 宝物[" + ezfyTierName(tier) + "]:" + cfg.Name
			// ★ 系统消息（用户要求：战斗掉落的装备要能看到）
			h.ezfySysChat("恭喜玩家 %s 战斗掉落%s宝物：%s", h.ezfyProfileName(city.UserID), ezfyTierName(tier), cfg.Name)
		}
	}
	if rand.Intn(100) < 40 {
		if jewel := h.randomJewel(terrain); jewel != nil {
			h.addEquipment(city, jewel)
			desc += " 珠宝:" + jewel.Name
			h.ezfySysChat("恭喜玩家 %s 缴获地形珠宝：%s", h.ezfyProfileName(city.UserID), jewel.Name)
		}
	}
	return desc
}

// ezfyProfileName 取玩家昵称（发系统消息用），拿不到时给个兜底，避免出现「恭喜玩家  晋升」
func (h *EzfyHandler) ezfyProfileName(uid uint) string {
	if uid == 0 {
		return "某玩家"
	}
	if n := h.ensureProfile(uid).Nickname; trimSpace(n) != "" {
		return n
	}
	return "某玩家"
}

func ezfyTierName(tier int) string {
	switch tier {
	case 1:
		return "初级"
	case 2:
		return "中级"
	case 3:
		return "高级"
	default:
		return "特殊"
	}
}

// randomEquipment 随机取指定品质的**非套装、非珠宝**装备
//
// ★ 用户规则（2026-09-22）：**套装军官装备只能通过宝箱开启**。
//
//	战斗掉落（活动目标/野地）只出普通装备（武器/防具/饰品）与地形珠宝，
//	套装件（set_id > 0）在这里被排除 —— 想让某套装能掉落，必须从这条规则外另开口子。
func (h *EzfyHandler) randomEquipment(tier int) *model.EzfyCfgEquipment {
	pool := []model.EzfyCfgEquipment{}
	for _, e := range ezfyCfg.equipments {
		if e.Type == "珠宝" || e.SetId > 0 {
			continue
		}
		if e.Tier == tier {
			pool = append(pool, e)
		}
	}
	if len(pool) == 0 {
		return nil
	}
	e := pool[rand.Intn(len(pool))]
	return &e
}

// randomJewel 按地形取珠宝（地形 1-8 对应珠宝 id 19-26）
func (h *EzfyHandler) randomJewel(terrain int) *model.EzfyCfgEquipment {
	jewels := []model.EzfyCfgEquipment{}
	for _, e := range ezfyCfg.equipments {
		if e.Type == "珠宝" {
			jewels = append(jewels, e)
		}
	}
	if len(jewels) == 0 {
		return nil
	}
	sort.Slice(jewels, func(i, j int) bool { return jewels[i].ID < jewels[j].ID })
	idx := maxInt(1, minInt(8, terrain)) - 1
	if idx >= len(jewels) {
		idx = len(jewels) - 1
	}
	j := jewels[idx]
	return &j
}

// captureWildlandOfficer 战胜野地/寇城后俘虏守将
//
// 规则(用户明确):
//   - 该野地/寇城必须在「野地类型」里配了**守军军官**（cfg.OfficerId > 0，最多 1 个，
//     且只能从军官池 ezfy_cfg_general 里选）；没配就俘不到军官
//   - 星级越高（军官池里的名将越强），俘虏概率越高
//   - 需要参谋部有空位
//
// wildType: 1陆地野地 2海野 3寇城;  level: 野地等级
func (h *EzfyHandler) captureWildlandOfficer(city *model.EzfyCity, wildType, level int, special bool) string {
	h.cfgs()
	cfg := ezfyCfg.wildland(wildType, level)
	if cfg == nil || cfg.OfficerId <= 0 {
		return "" // 该目标没有守将, 不产生战俘
	}
	g := ezfyCfg.general(cfg.OfficerId)
	if g == nil {
		return "" // 军官池里已没有这个军官（被删了）
	}
	star := g.Star
	if star <= 0 {
		star = 1
	}
	// 概率: 基础 20%, 名将星级每星 +5%, 特殊目标翻倍, 上限 60%
	chance := 20 + star*5
	if special {
		chance *= 2
	}
	if chance > 60 {
		chance = 60
	}
	if rand.Intn(100) >= chance {
		return ""
	}
	if h.buildingLevel(city.ID, ezfyBuildingStaff) < 1 {
		return "" // 没有参谋部, 无法收押
	}
	// 参谋部容量
	if h.officerCount(city.ID) >= h.buildingLevel(city.ID, ezfyBuildingStaff) {
		return ""
	}
	// ★ 俘虏到的就是配置里那位**军官池军官**（属性/星级取自军官池）
	//   等级同样夹在「军官最高等级 150」以内
	captiveLv := maxInt(1, minInt(level, ezfyOfficerMaxLevel))
	o := model.EzfyOfficer{
		CityId: int64(city.ID), GeneralId: g.ID, Name: g.Name, Star: star,
		Level: captiveLv, Exp: 0,
		Military: g.Military, Logistics: g.Logistics, Learning: g.Learning,
		// ★ 原始属性 = 军官池里的值
		BaseMilitary: g.Military, BaseLogistics: g.Logistics, BaseLearning: g.Learning,
		FreePoints: maxInt(0, captiveLv-1),
		Loyalty:    30, Skill: "", Equipment: "",
		Position: ezfyPositionNone, Status: 0, IsCaptive: 1, UpdateTime: time.Now(),
	}
	h.DB.Create(&o)
	return "俘虏敌将:" + o.Name + "(" + strconv.Itoa(star) + "星, 忠诚30) 可前往军校收编"
}

// defectDefenderOfficers 攻打玩家城市后, 目标城军官忠诚下降;
// 忠诚归零的军官会弃城投敌, 成为攻方的战俘(复刻用户描述的 PvP 战俘来源)。
//
// 返回写进攻方战报的文本片段。
func (h *EzfyHandler) defectDefenderOfficers(atkCity *model.EzfyCity, target *model.EzfyCity, atkUid uint) string {
	if target == nil {
		return ""
	}
	officers := h.officerList(target.ID)
	if len(officers) == 0 {
		return ""
	}
	// 参谋部有空位才收得下战俘
	room := h.buildingLevel(atkCity.ID, ezfyBuildingStaff) - h.officerCount(atkCity.ID)

	var defected []model.EzfyOfficer
	var stayed []string
	for i := range officers {
		o := &officers[i]
		drop := 10 + rand.Intn(11) // 每次被攻打 忠诚 -10~-20
		loyalty := o.Loyalty - drop
		if loyalty <= 0 {
			defected = append(defected, *o)
			continue
		}
		h.DB.Model(&model.EzfyOfficer{}).Where("id = ?", o.ID).
			Update("loyalty", loyalty)
		stayed = append(stayed, o.Name+"("+strconv.Itoa(loyalty)+")")
	}

	var b strings.Builder
	if len(stayed) > 0 {
		b.WriteString("\n敌方军官忠诚下降: " + strings.Join(stayed, " "))
	}
	for i := range defected {
		o := &defected[i]
		// 从原城移除
		h.DB.Delete(&model.EzfyOfficer{}, o.ID)
		if room > 0 {
			room--
			// 收编为攻方战俘(等级/属性保留, 忠诚重置为 30 待收编)
			cap := model.EzfyOfficer{
				CityId: int64(atkCity.ID), GeneralId: o.GeneralId, Name: o.Name, Star: o.Star,
				Level: o.Level, Exp: o.Exp,
				Military: o.Military, Logistics: o.Logistics, Learning: o.Learning,
				// ★ 原始属性与可用点数一起带走，别让被俘/归降把玩家点过的点吞掉
				BaseMilitary: o.BaseMilitary, BaseLogistics: o.BaseLogistics, BaseLearning: o.BaseLearning,
				FreePoints: o.FreePoints,
				Loyalty:    30, Skill: o.Skill, Equipment: o.Equipment,
				Position: ezfyPositionNone, Status: 0, IsCaptive: 1, UpdateTime: time.Now(),
			}
			h.DB.Create(&cap)
			b.WriteString("\n敌方军官 " + o.Name + " 忠诚归零, 弃城归降, 已收入我方战俘营")
			h.addReport(target.UserID, 6, "将领叛离: "+o.Name,
				o.Name+"因忠诚度归零, 弃城投敌, 加入了对"+atkCity.Name+"的阵营。\n请及时赏赐军官以维持忠诚。", "")
		} else {
			b.WriteString("\n敌方军官 " + o.Name + " 忠诚归零离去(我方参谋部已满, 未能收押)")
			h.addReport(target.UserID, 6, "将领叛离: "+o.Name,
				o.Name+"因忠诚度归零而离开了你的城市。", "")
		}
	}
	return b.String()
}

// ============ HTTP 接口 ============

// Officers GET /games/ezfy/officers —— 军官列表 + 军校/参谋部等级
func (h *EzfyHandler) Officers(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	city := h.getOrCreateCity(uid)
	h.refreshCity(uid, &city)
	list := h.officerList(city.ID)
	views := []gin.H{}
	for i := range list {
		o := &list[i]
		skills := officerSkills(o)
		em, el, ee := h.officerEffective(o)
		sm, sl, se, activeSets := h.officerSetBonus(o)
		bm, bl, be := officerBaseAttr(o)
		views = append(views, gin.H{
			"id": o.ID, "name": o.Name, "star": o.Star, "level": o.Level, "exp": o.Exp,
			"military": o.Military, "logistics": o.Logistics, "learning": o.Learning,
			// ★ 含装备/套装加成的有效属性（前端展示「基础(+装备)」）
			"military_total": em, "logistics_total": el, "learning_total": ee,
			"equip_military": em - o.Military, "equip_logistics": el - o.Logistics,
			"equip_learning": ee - o.Learning,
			// ★ 2026-09-22：原始属性 / 可用属性点 / 已分配点数（前端加点用）
			"base_military": bm, "base_logistics": bl, "base_learning": be,
			"free_points": o.FreePoints, "used_points": officerAllocatedPoints(o),
			"set_military": sm, "set_logistics": sl, "set_learning": se,
			"active_sets":  activeSets,
			"set_progress": h.officerSetProgressView(o),
			// ★ 军官装备的六项战斗加成（伤害/防御/生命/移动距离/暴击）——
			//   列表里也要下发，否则玩家会以为「穿了装备没加属性」
			"battle":  h.officerBattleView(o),
			"loyalty": o.Loyalty, "position": o.Position, "position_name": ezfyPositionName(o.Position),
			"status": o.Status, "status_name": ezfyOfficerStatusName(o),
			"is_captive": o.IsCaptive, "skills": skills,
			"equip_count": len(officerEquipped(o)),
			// 攻/防(复刻原版军官卡片上的 攻/防 两项, 含技能与装备加成)
			"attack":  h.officerBattleBonus(o),
			"defence": h.officerGuardBonus(o),
		})
	}
	resp.OK(c, gin.H{
		"officers":      views,
		"academy_level": h.buildingLevel(city.ID, ezfyBuildingAcademy),
		"staff_level":   h.buildingLevel(city.ID, ezfyBuildingStaff),
		"capacity":      h.buildingLevel(city.ID, ezfyBuildingStaff),
		// ★ 用上面已取到的 list（勿改回 h.officerCount，那会再查一次库）
		"used": officerCountOf(list),
		"gold": city.Gold,
		// ★ 用户反馈「军官是消耗黄金的，黄金现在消耗 0」→ 军官工资（黄金/小时）。
		//   随 calcResource 懒结算一起扣，这里只负责让玩家看得见。
		//   ★ 用上面已取到的 list 做纯内存计算（勿改回 officerSalaryPerHour，那会再查一次库）
		"salary":           officerSalaryOf(list),
		"salary_per_level": ezfyOfficerSalaryPerLvCfg(),
		// ★ 用户规则「军官最高等级 150」：前端据此显示「满级」
		"max_level": ezfyOfficerMaxLevel,
		// ★ 升星配置（前端据此显示星级上限/成功率/每星加点）
		"star_up_on": ezfyStarUpOn(), "star_chance_on": ezfyStarChanceOn(),
		"star_max": ezfyStarMax(), "star_attr_gain": ezfyStarAttrGain(),
		"star_card": h.itemCount(uid, ezfyStarItemID),
	})
}

func ezfyPositionName(p int) string {
	switch p {
	case ezfyPositionMayor:
		return "市长"
	case ezfyPositionGuard:
		return "城守"
	default:
		return "无"
	}
}

func ezfyOfficerStatusName(o *model.EzfyOfficer) string {
	if o.Status == 1 {
		return "出征中"
	}
	if o.IsCaptive == 1 {
		return "俘虏"
	}
	return "在职"
}

// OfficerDetail GET /games/ezfy/officers/:id —— 军官详情（技能/装备/可学技能/背包装备）
func (h *EzfyHandler) OfficerDetail(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	city := h.getOrCreateCity(uid)
	h.refreshCity(uid, &city)
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	o := h.officerOf(city.ID, id)
	if o == nil {
		resp.ParamError(c, "武将不存在")
		return
	}
	skillViews := []gin.H{}
	for _, s := range officerSkills(o) {
		eff := ""
		if sid, ok := ezfyCfg.skillByName[s]; ok {
			if cfg := ezfyCfg.skill(sid); cfg != nil {
				eff = cfg.Effect
			}
		}
		skillViews = append(skillViews, gin.H{"name": s, "effect": eff})
	}
	allSkills := []gin.H{}
	for _, s := range ezfyCfg.skills {
		allSkills = append(allSkills, gin.H{"id": s.ID, "name": s.Name, "effect": s.Effect, "type": s.Type})
	}
	sort.Slice(allSkills, func(i, j int) bool {
		return allSkills[i]["id"].(int) < allSkills[j]["id"].(int)
	})
	em, el, ee := h.officerEffective(o)
	bag := []gin.H{}
	for _, e := range h.equipmentList(uid) {
		bag = append(bag, gin.H{
			"id": e.ID, "name": e.Name, "type": e.Type, "tier": e.Tier,
			"tier_name": ezfyTierName(e.Tier),
			"military":  e.Military, "logistics": e.Logistics, "learning": e.Learning,
			"level": e.Level, "officer_id": e.OfficerId, "worn": e.OfficerId > 0,
			"slot": e.EquipSlot(), "set_id": e.SetId, "set_name": h.ezfySetName(e.SetId),
			"series": e.Series, "enhance": e.Enhance,
			"dmg": e.Dmg, "def": e.Def, "hp": e.Hp, "move": e.Move, "crit": e.Crit, "crit_dmg": e.CritDmg,
		})
	}
	sm, sl, se, activeSets := h.officerSetBonus(o)
	bm, bl, be := officerBaseAttr(o)
	resp.OK(c, gin.H{
		"officer": gin.H{
			"id": o.ID, "name": o.Name, "star": o.Star, "level": o.Level, "exp": o.Exp,
			"military": o.Military, "logistics": o.Logistics, "learning": o.Learning,
			// ★ 有效属性（基础 + 装备 + 套装），前端展示成「33 (+5) = 38」
			"military_total": em, "logistics_total": el, "learning_total": ee,
			"equip_military": em - o.Military, "equip_logistics": el - o.Logistics,
			"equip_learning": ee - o.Learning,
			// ★ 2026-09-22：加点用
			"base_military": bm, "base_logistics": bl, "base_learning": be,
			"free_points": o.FreePoints, "used_points": officerAllocatedPoints(o),
			"set_military": sm, "set_logistics": sl, "set_learning": se,
			"active_sets": activeSets,
			// ★ 套装穿戴进度（穿齐才生效；这里让前端能显示「还差 N 件」）
			"set_progress": h.officerSetProgressView(o),
			// ★ 装备六项战斗加成（直接进战斗计算）
			"battle": h.officerBattleView(o),
			// ★ 升星：星级上限 / 当前成功率 / 每星加多少 / 持有升星卡数
			"star_max": ezfyStarMax(), "star_up_on": ezfyStarUpOn(),
			"star_chance_on": ezfyStarChanceOn(), "star_rate": ezfyStarSuccessRate(o.Star),
			"star_attr_gain": ezfyStarAttrGain(), "star_card": h.itemCount(uid, ezfyStarItemID),
			"attack": h.officerBattleBonus(o), "defence": h.officerGuardBonus(o),
			"loyalty": o.Loyalty, "position": o.Position, "position_name": ezfyPositionName(o.Position),
			"status": o.Status, "status_name": ezfyOfficerStatusName(o), "is_captive": o.IsCaptive,
			"exp_need": o.Level * 200,
		},
		"skills": skillViews, "all_skills": allSkills,
		// ★ 已穿戴装备补上套装名（老数据里只存了 set_id，前端不该显示「套装21」这种内部 ID）
		"equipped": h.officerEquippedView(o), "bag": bag, "gold": city.Gold,
	})
}

// officerEquippedView 已穿戴装备的下发格式（补套装名，前端直接用）
func (h *EzfyHandler) officerEquippedView(o *model.EzfyOfficer) []gin.H {
	out := []gin.H{}
	for _, m := range officerEquipped(o) {
		sid := jsonInt(m["set_id"])
		slot, _ := m["slot"].(string)
		typ, _ := m["type"].(string)
		if slot == "" {
			slot = typ
		}
		out = append(out, gin.H{
			"id": jsonInt(m["id"]), "name": m["name"], "type": typ, "slot": slot,
			"set_id": sid, "set_name": h.ezfySetName(sid),
			"military": jsonInt(m["military"]), "logistics": jsonInt(m["logistics"]),
			"learning": jsonInt(m["learning"]),
			"series":   m["series"], "enhance": jsonInt(m["enhance"]),
			"dmg": jsonInt(m["dmg"]), "def": jsonInt(m["def"]), "hp": jsonInt(m["hp"]),
			"move": jsonInt(m["move"]), "crit": jsonInt(m["crit"]), "crit_dmg": jsonInt(m["crit_dmg"]),
		})
	}
	return out
}

// ezfySetName 套装名（0 / 查不到返回空串）
func (h *EzfyHandler) ezfySetName(setId int) string {
	if s := ezfyCfg.equipSet(setId); s != nil {
		return s.Name
	}
	return ""
}

// AcadeRecruit GET /games/ezfy/acade/recruit —— 军校候选名将
func (h *EzfyHandler) AcadeRecruit(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	city := h.getOrCreateCity(uid)
	h.refreshCity(uid, &city)
	academy := h.buildingLevel(city.ID, ezfyBuildingAcademy)
	out := gin.H{
		"academy_level": academy, "staff_level": h.buildingLevel(city.ID, ezfyBuildingStaff),
		"capacity": h.buildingLevel(city.ID, ezfyBuildingStaff), "used": h.officerCount(city.ID),
		"gold": city.Gold, "candidates": []gin.H{},
	}
	if academy >= 1 {
		drafts, left, limit := h.recruitInfo(uid, academy)
		views := []gin.H{}
		for _, d := range drafts {
			views = append(views, gin.H{
				"key": d.Key, "name": d.Name, "level": d.Level, "star": d.Star,
				"military": d.Military, "logistics": d.Logistics, "learning": d.Learning,
				"cost": d.Cost,
			})
		}
		out["candidates"] = views
		out["refresh_left"] = left
		out["refresh_limit"] = limit
	}
	resp.OK(c, out)
}

// AcadeRefresh POST /games/ezfy/acade/recruit/refresh
func (h *EzfyHandler) AcadeRefresh(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	city := h.getOrCreateCity(uid)
	academy := h.buildingLevel(city.ID, ezfyBuildingAcademy)
	if academy < 1 {
		h.fail(c, "需要先建造军校")
		return
	}
	h.done(c, h.refreshRecruit(uid, academy), "候选已刷新")
}

// AcadeRecruitDo POST /games/ezfy/acade/recruit/hire  {key}
func (h *EzfyHandler) AcadeRecruitDo(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	city := h.getOrCreateCity(uid)
	var req struct {
		Key string `json:"key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Key == "" {
		resp.ParamError(c, "参数错误")
		return
	}
	h.done(c, h.hireOfficerDraft(&city, uid, req.Key), "军官雇佣成功")
}

// OfficerGrant POST /games/ezfy/officers/:id/grant
func (h *EzfyHandler) OfficerGrant(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	city := h.getOrCreateCity(uid)
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	h.done(c, h.grantOfficer(&city, id), "赏赐成功, 忠诚已提升")
}

// OfficerAttr POST /games/ezfy/officers/:id/attr  {attr: military|logistics|learning, count}
//
// ★ 2026-09-22 用户要求：每升 1 级得 1 点可用属性点，玩家自己分配；
// 只写玩家自己的军官实例，**绝不回写军官池**。
func (h *EzfyHandler) OfficerAttr(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	city := h.getOrCreateCity(uid)
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		Attr  string `json:"attr"`
		Count int    `json:"count"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	o := h.officerOf(city.ID, id)
	if o == nil {
		resp.NotFound(c, "军官不存在")
		return
	}
	if msg := h.officerAddAttr(&city, id, req.Attr, req.Count); msg != "" {
		h.fail(c, msg)
		return
	}
	// 回读最新状态，省掉前端一次请求
	var now model.EzfyOfficer
	h.DB.First(&now, o.ID)
	sm, sl, se, activeSets := h.officerSetBonus(&now)
	em, el, ee := h.officerEffective(&now)
	resp.OK(c, gin.H{
		"msg": "加点成功",
		"officer": gin.H{
			"id": now.ID, "military": now.Military, "logistics": now.Logistics, "learning": now.Learning,
			"base_military": now.BaseMilitary, "base_logistics": now.BaseLogistics, "base_learning": now.BaseLearning,
			"free_points": now.FreePoints, "used_points": officerAllocatedPoints(&now),
			"military_total": em, "logistics_total": el, "learning_total": ee,
			"set_military": sm, "set_logistics": sl, "set_learning": se, "active_sets": activeSets,
		},
	})
}

// OfficerAttrAll POST /games/ezfy/officers/:id/attr/all  {attr}
//
// 把当前**全部**可用属性点一次性加到某一项（懒人按钮）。
func (h *EzfyHandler) OfficerAttrAll(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	city := h.getOrCreateCity(uid)
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		Attr string `json:"attr"`
	}
	_ = c.ShouldBindJSON(&req)
	o := h.officerOf(city.ID, id)
	if o == nil {
		resp.NotFound(c, "军官不存在")
		return
	}
	if o.FreePoints <= 0 {
		h.fail(c, "没有可用属性点")
		return
	}
	h.done(c, h.officerAddAttr(&city, id, req.Attr, o.FreePoints), "加点成功")
}

// officerStarUp 执行一次升星判定（**不扣升星卡**，扣卡由调用方按「失败是否保留」决定）
//
// ★ 2026-09-22 用户要求：「玩家自己的军官可以用升星卡升级星级，属性增加；
// 概率的最好也能有个开关控制，属性加多少也要可配。」
//
//   - 概率开关 `officer_star_chance_on`：关 = 必成功
//   - 成功率 = 基础 − (当前星级−1)×递减，夹在 [下限, 100]
//   - 每升 1 星三维各 +`officer_star_attr_gain`（**base_* 一起加**，重修书洗点不会把它洗掉）
//   - 星级上限 `officer_star_max`
//
// 只写玩家自己的军官实例，绝不回写军官池。
func (h *EzfyHandler) officerStarUp(city *model.EzfyCity, officerId int64) (string, bool) {
	o := h.officerOf(city.ID, officerId)
	if o == nil {
		return "军官不存在", false
	}
	if o.IsCaptive == 1 {
		return "俘虏不能升星, 请先在军校收编", false
	}
	max := ezfyStarMax()
	if max < 1 {
		max = 1
	}
	if o.Star >= max {
		return fmt.Sprintf("星级已达上限(%d星)", max), false
	}
	rate := ezfyStarSuccessRate(o.Star)
	if rand.Intn(100) >= rate {
		if ezfyStarChanceOn() {
			return fmt.Sprintf("升星失败(成功率%d%%，星级不变)", rate), false
		}
		return "升星失败", false
	}
	gain := ezfyStarAttrGain()
	bm, bl, be := officerBaseAttr(o)
	if bm <= 0 {
		bm = o.Military
	}
	if bl <= 0 {
		bl = o.Logistics
	}
	if be <= 0 {
		be = o.Learning
	}
	h.DB.Model(&model.EzfyOfficer{}).Where("id = ?", o.ID).Updates(map[string]interface{}{
		"star":     o.Star + 1,
		"military": o.Military + gain, "logistics": o.Logistics + gain, "learning": o.Learning + gain,
		"base_military": bm + gain, "base_logistics": bl + gain, "base_learning": be + gain,
		"update_time": time.Now(),
	})
	h.addReport(city.UserID, 6, "军官升星: "+o.Name,
		fmt.Sprintf("%s 升星成功: %d星→%d星, 军事/后勤/学识各+%d。", o.Name, o.Star, o.Star+1, gain), "")
	return fmt.Sprintf("升星成功: %s %d星→%d星, 三维各+%d", o.Name, o.Star, o.Star+1, gain), true
}

// OfficerStarUp POST /games/ezfy/officers/:id/starup
//
// 在军官详情页直接点「升星」：消耗背包里的 1 张「军官升星卡」。
// 失败是否退卡由 `officer_star_keep_on_fail` 控制。
func (h *EzfyHandler) OfficerStarUp(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	city := h.getOrCreateCity(uid)
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if !ezfyStarUpOn() {
		h.fail(c, "升星功能已关闭")
		return
	}
	o := h.officerOf(city.ID, id)
	if o == nil {
		resp.NotFound(c, "军官不存在")
		return
	}
	if h.itemCount(uid, ezfyStarItemID) <= 0 {
		h.fail(c, "没有「军官升星卡」，可在商城购买或开宝箱获得")
		return
	}
	msg, ok := h.officerStarUp(&city, id)
	// ★ 失败时按配置决定要不要退卡（keep=1 则本次不消耗）
	if ok || !ezfyStarKeepOnFail() {
		h.consumeItem(uid, ezfyStarItemID)
	}
	if !ok {
		h.fail(c, msg)
		return
	}
	h.done(c, "", msg)
}

// ============ 宝箱（钻石/黄金购买，开箱按权重出套装件） ============
//
// ★ 2026-09-22 用户要求：「有的套装是开宝箱概率得到的，看看怎么引入宝箱，宝箱一般用钻石买。」
//
// 奖池 ezfy_cfg_chest_item：kind 1=装备（进 ezfy_equipment 背包，可就地穿）、2=道具（进道具背包）。

// ezfyChestPool 取某宝箱的奖池（按 id 升序，保证抽奖顺序稳定）
func (h *EzfyHandler) ezfyChestPool(chestId int) []model.EzfyCfgChestItem {
	var list []model.EzfyCfgChestItem
	h.DB.Where("chest_id = ?", chestId).Order("id").Find(&list)
	return list
}

// ezfyDrawChest 按权重从奖池抽一条（全部权重为 0 时等概率）
func ezfyDrawChest(pool []model.EzfyCfgChestItem) *model.EzfyCfgChestItem {
	if len(pool) == 0 {
		return nil
	}
	sum := 0
	for i := range pool {
		w := pool[i].Weight
		if w < 0 {
			w = 0
		}
		sum += w
	}
	if sum <= 0 {
		p := pool[rand.Intn(len(pool))]
		return &p
	}
	r := rand.Intn(sum)
	acc := 0
	for i := range pool {
		w := pool[i].Weight
		if w < 0 {
			w = 0
		}
		acc += w
		if r < acc {
			return &pool[i]
		}
	}
	p := pool[len(pool)-1]
	return &p
}

// ezfyGrantChestPrize 发放一件奖品，返回展示文案
func (h *EzfyHandler) ezfyGrantChestPrize(city *model.EzfyCity, it *model.EzfyCfgChestItem) string {
	if it == nil {
		return "空箱"
	}
	n := it.Count
	if n <= 0 {
		n = 1
	}
	switch it.Kind {
	case 1: // 装备 → 玩家装备背包
		cfg := ezfyCfg.equipment(it.RefId)
		if cfg == nil {
			return ""
		}
		for i := 0; i < n; i++ {
			h.addEquipment(city, cfg)
		}
		return cfg.Name + "×" + strconv.Itoa(n)
	case 2: // 道具 → 道具背包
		cfg := ezfyCfg.item(it.RefId)
		if cfg == nil {
			return ""
		}
		h.addItem(city.UserID, it.RefId, n)
		return cfg.Name + "×" + strconv.Itoa(n)
	case 3: // ★ 整套（RefId = 套装 id）：把该套装的**全部件**一次性发给玩家
		//
		// ★ 用户要求（2026-09-22）：「套装宝箱……开出来还是按套来吧」
		// —— 免得玩家花 300~800 钻石开出一件，还得凑 10 次。
		s := ezfyCfg.equipSet(it.RefId)
		if s == nil {
			return ""
		}
		cnt := 0
		// 按 ID 升序发放，保证「同一次开箱给的件顺序稳定」（equipments 是 map，遍历顺序随机）
		for _, id := range ezfyEquipIDsOfSet(it.RefId) {
			e := ezfyCfg.equipments[id]
			ee := e
			h.addEquipment(city, &ee)
			cnt++
		}
		if cnt == 0 {
			return ""
		}
		return s.Name + " 整套（" + strconv.Itoa(cnt) + "件）"
	default:
		return ""
	}
}

// ChestList GET /games/ezfy/chest —— 宝箱列表（含奖池展示）
func (h *EzfyHandler) ChestList(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	city := h.getOrCreateCity(uid)
	h.refreshCity(uid, &city)
	var chests []model.EzfyCfgChest
	h.DB.Where("enabled <> 0").Order("sort_no, id").Find(&chests)
	out := []gin.H{}
	for _, c2 := range chests {
		pool := []gin.H{}
		for _, p := range h.ezfyChestPool(c2.ID) {
			name, quality := "", p.Quality
			switch p.Kind {
			case 1:
				if cfg := ezfyCfg.equipment(p.RefId); cfg != nil {
					name = cfg.Name
					if quality == "" {
						quality = ezfyTierName(cfg.Tier)
					}
				}
			case 2:
				if cfg := ezfyCfg.item(p.RefId); cfg != nil {
					name = cfg.Name
				}
			case 3: // 整套（RefId = 套装 id）
				if s := ezfyCfg.equipSet(p.RefId); s != nil {
					name = s.Name + " 整套"
				}
			}
			if name == "" {
				continue
			}
			pool = append(pool, gin.H{
				"kind": p.Kind, "ref_id": p.RefId, "name": name,
				"count": p.Count, "weight": p.Weight, "quality": quality,
			})
		}
		out = append(out, gin.H{
			"id": c2.ID, "name": c2.Name,
			"price_gold": c2.PriceGold, "price_diamond": c2.PriceDiamond,
			"stock": c2.Stock, "sold_out": c2.Stock == 0, "open_max": c2.OpenMax,
			"des": c2.Des, "effect": c2.Effect, "pool": pool,
		})
	}
	resp.OK(c, gin.H{"chests": out, "gold": city.Gold, "diamond": h.ensureProfile(uid).Diamond})
}

// ChestOpen POST /games/ezfy/chest/open  {chest_id, count, currency: gold|diamond}
func (h *EzfyHandler) ChestOpen(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	city := h.getOrCreateCity(uid)
	h.calcResource(&city)
	var req struct {
		ChestId  int    `json:"chest_id"`
		Count    int    `json:"count"`
		Currency string `json:"currency"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if req.Count <= 0 {
		req.Count = 1
	}
	var chest model.EzfyCfgChest
	if err := h.DB.First(&chest, req.ChestId).Error; err != nil || chest.Enabled == 0 {
		h.fail(c, "宝箱不存在或已下架")
		return
	}
	maxOpen := chest.OpenMax
	if maxOpen <= 0 {
		maxOpen = 1
	}
	if req.Count > maxOpen {
		h.fail(c, fmt.Sprintf("单次最多开%d个", maxOpen))
		return
	}
	if chest.Stock == 0 {
		h.fail(c, "该宝箱已售罄")
		return
	}
	if chest.Stock > 0 && req.Count > chest.Stock {
		h.fail(c, fmt.Sprintf("库存不足(剩余%d个)", chest.Stock))
		return
	}
	useDiamond := req.Currency == "diamond"
	price := chest.PriceGold
	unit := "黄金"
	if useDiamond {
		price = chest.PriceDiamond
		unit = "钻石"
	}
	if price <= 0 {
		h.fail(c, "该宝箱不支持用"+unit+"购买")
		return
	}
	total := price * int64(req.Count)
	if useDiamond {
		prof := h.ensureProfile(uid)
		if prof.Diamond < total {
			h.fail(c, fmt.Sprintf("钻石不足(需要%d钻石, 当前%d)", total, prof.Diamond))
			return
		}
		if err := h.DB.Model(&model.EzfyProfile{}).Where("id = ?", prof.ID).
			Update("diamond", prof.Diamond-total).Error; err != nil {
			h.fail(c, "扣钻石失败："+err.Error())
			return
		}
	} else {
		if city.Gold < total {
			h.fail(c, fmt.Sprintf("黄金不足(需要%d黄金, 当前%d)", total, city.Gold))
			return
		}
		city.Gold -= total
		h.saveCityRes(&city)
	}
	pool := h.ezfyChestPool(chest.ID)
	if len(pool) == 0 {
		h.fail(c, "该宝箱还没配置奖池，请联系管理员")
		return
	}
	results := []gin.H{}
	for i := 0; i < req.Count; i++ {
		prize := ezfyDrawChest(pool)
		desc := h.ezfyGrantChestPrize(&city, prize)
		if desc == "" {
			continue
		}
		results = append(results, gin.H{"name": desc, "quality": prize.Quality})
	}
	if chest.Stock > 0 {
		h.DB.Model(&model.EzfyCfgChest{}).Where("id = ?", chest.ID).
			Update("stock", gorm.Expr("stock - ?", req.Count))
		h.cfgsReload()
	}
	names := []string{}
	for _, r := range results {
		names = append(names, fmt.Sprint(r["name"]))
	}
	// resp.OK 会把 data 里的 msg 提升到顶层（前端读 r.msg）
	resp.OK(c, gin.H{
		"msg":     fmt.Sprintf("开箱成功: 花费%d%s, 获得 %s", total, unit, strings.Join(names, "、")),
		"results": results, "gold": city.Gold, "diamond": h.ensureProfile(uid).Diamond,
	})
}

// OfficerSkill POST /games/ezfy/officers/:id/skill  {op: learn|forget, skill_id}
func (h *EzfyHandler) OfficerSkill(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	city := h.getOrCreateCity(uid)
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		Op      string `json:"op"`
		SkillId int    `json:"skill_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if req.Op == "forget" {
		h.done(c, h.forgetSkill(&city, id, req.SkillId), "技能已遗忘")
		return
	}
	h.done(c, h.learnSkill(&city, id, req.SkillId), "技能学习成功")
}

// OfficerEquip POST /games/ezfy/officers/:id/equip  {equip_id, op: on|off}
func (h *EzfyHandler) OfficerEquip(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	city := h.getOrCreateCity(uid)
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		EquipId int64  `json:"equip_id"`
		Op      string `json:"op"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if req.Op == "off" {
		h.done(c, h.unequipItem(&city, req.EquipId), "装备已卸下")
		return
	}
	h.done(c, h.equipItem(&city, id, req.EquipId), "装备已穿上")
}

// OfficerPosition POST /games/ezfy/officers/:id/position  {position}
func (h *EzfyHandler) OfficerPosition(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	city := h.getOrCreateCity(uid)
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		Position int `json:"position"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	h.done(c, h.setOfficerPosition(&city, id, req.Position), "任命成功")
}

// OfficerCaptive POST /games/ezfy/officers/:id/captive  {op: free|recruit}
func (h *EzfyHandler) OfficerCaptive(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	city := h.getOrCreateCity(uid)
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		Op string `json:"op"`
	}
	_ = c.ShouldBindJSON(&req)
	if req.Op == "recruit" {
		h.done(c, h.recruitCaptive(&city, id), "收编成功, 军官已入列")
		return
	}
	h.done(c, h.freeOfficer(&city, id), "已释放该武将")
}

// OfficerExile POST /games/ezfy/officers/:id/exile
func (h *EzfyHandler) OfficerExile(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	city := h.getOrCreateCity(uid)
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	h.done(c, h.exileOfficer(&city, id), "已流放该军官")
}

// OfficerSkills GET /games/ezfy/officers/skills —— 技能总览 + 我的军官
func (h *EzfyHandler) OfficerSkills(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	city := h.getOrCreateCity(uid)
	h.refreshCity(uid, &city)
	skills := []gin.H{}
	for _, s := range ezfyCfg.skills {
		skills = append(skills, gin.H{"id": s.ID, "name": s.Name, "effect": s.Effect, "type": s.Type, "des": s.Des})
	}
	sort.Slice(skills, func(i, j int) bool { return skills[i]["id"].(int) < skills[j]["id"].(int) })
	list := []gin.H{}
	for _, o := range h.officerList(city.ID) {
		list = append(list, gin.H{"id": o.ID, "name": o.Name, "level": o.Level,
			"skills": officerSkills(&o), "skill_count": len(officerSkills(&o))})
	}
	resp.OK(c, gin.H{"skills": skills, "officers": list, "gold": city.Gold})
}

// OfficerEquipments GET /games/ezfy/officers/equipments —— 装备图鉴 + 我的背包
func (h *EzfyHandler) OfficerEquipments(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	city := h.getOrCreateCity(uid)
	h.refreshCity(uid, &city)
	bag := []gin.H{}
	for _, e := range h.equipmentList(uid) {
		wornBy := ""
		if e.OfficerId > 0 {
			if o := h.officerOf(city.ID, e.OfficerId); o != nil {
				wornBy = o.Name
			}
		}
		bag = append(bag, gin.H{
			"id": e.ID, "name": e.Name, "type": e.Type, "tier": e.Tier,
			"tier_name": ezfyTierName(e.Tier),
			"military":  e.Military, "logistics": e.Logistics, "learning": e.Learning,
			"level": e.Level, "officer_id": e.OfficerId, "worn": e.OfficerId > 0, "worn_by": wornBy,
			"slot": e.EquipSlot(), "set_id": e.SetId, "set_name": h.ezfySetName(e.SetId),
			"series": e.Series, "enhance": e.Enhance,
			"dmg": e.Dmg, "def": e.Def, "hp": e.Hp, "move": e.Move, "crit": e.Crit, "crit_dmg": e.CritDmg,
		})
	}
	cfgList := []gin.H{}
	for _, e := range ezfyCfg.equipments {
		cfgList = append(cfgList, gin.H{"id": e.ID, "name": e.Name, "type": e.Type, "tier": e.Tier,
			"tier_name": ezfyTierName(e.Tier), "military": e.Military, "logistics": e.Logistics,
			"learning": e.Learning, "level": e.Level, "des": e.Des,
			"slot": e.EquipSlot(), "set_id": e.SetId, "set_name": h.ezfySetName(e.SetId),
			"price_gold": e.PriceGold, "price_diamond": e.PriceDiamond, "stock": e.Stock,
			"effect": e.Effect, "series": e.Series,
			"dmg": e.Dmg, "def": e.Def, "hp": e.Hp, "move": e.Move, "crit": e.Crit, "crit_dmg": e.CritDmg})
	}
	sort.Slice(cfgList, func(i, j int) bool { return cfgList[i]["id"].(int) < cfgList[j]["id"].(int) })
	// ★ 套装总览 = **我拥有的**套装（用户反馈：原来列的是全部套装配置，玩家以为是自己有的）
	//   统计口径：背包 + 已穿戴的装备里出现过的 set_id，按套计数。
	owned := map[int]int{}
	for _, e := range h.equipmentList(uid) {
		if e.SetId > 0 {
			owned[e.SetId]++
		}
	}
	setList := []gin.H{}
	for _, s := range ezfyCfg.equipSets() {
		have := owned[s.ID]
		if have == 0 {
			continue // 一件都没有的套装不展示（图鉴在下面「装备图鉴」里）
		}
		setList = append(setList, gin.H{
			"id": s.ID, "name": s.Name, "parts": s.Parts, "series": s.Series,
			"have": have, "complete": have >= s.Parts,
			"military": s.Military, "logistics": s.Logistics, "learning": s.Learning,
			"dmg": s.Dmg, "def": s.Def, "hp": s.Hp, "move": s.Move, "crit": s.Crit, "crit_dmg": s.CritDmg,
			"effect": s.Effect, "des": s.Des,
		})
	}
	resp.OK(c, gin.H{"bag": bag, "all": cfgList, "sets": setList})
}

// OfficerGenerals GET /games/ezfy/officers/generals —— 名将图鉴（按等级倒序）
//
// ★ 2026-09-22：军官池里现在还有 1000 名普通军官，图鉴**只列名将（kind=2）**，
// 否则玩家会看到一千多条。
func (h *EzfyHandler) OfficerGenerals(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	owned := h.ownedGeneralIds(uid)
	list := []gin.H{}
	for _, g := range ezfyCfg.generals {
		if g.Kind == 1 {
			continue
		}
		// 名将只由管理端发放, 获取渠道统一显示为「管理端发放」
		list = append(list, gin.H{
			"id": g.ID, "name": g.Name, "level": g.Level, "star": g.Star,
			"military": g.Military, "logistics": g.Logistics, "learning": g.Learning,
			"source": "管理端发放", "skill": g.Skill, "des": g.Des, "owned": owned[g.ID],
		})
	}
	sort.Slice(list, func(i, j int) bool { return list[i]["level"].(int) > list[j]["level"].(int) })
	resp.OK(c, gin.H{"generals": list})
}

// ============ 装备商城（套装用黄金/钻石购买） ============
//
// ★ 2026-09-22 用户要求：军官穿的装备有套装，玩家自己用黄金或钻石买。
// 只卖「上架」的（price_gold>0 或 price_diamond>0），库存 -1 = 无上限。

// ezfyEquipIDsOfSet 某套装的全部件 ID（升序）
//
// ★ ezfyCfg.equipments 是 map，遍历顺序随机；凡是要「稳定顺序」的地方都走这里
// （开整套的发放顺序、商城的列顺序等）。
func ezfyEquipIDsOfSet(setId int) []int {
	ids := []int{}
	for id, e := range ezfyCfg.equipments {
		if e.SetId == setId {
			ids = append(ids, id)
		}
	}
	sort.Ints(ids)
	return ids
}

// ezfyEquipIDsAsc 全部装备 ID（升序）—— 商城按部位铺表格时保证顺序稳定
func ezfyEquipIDsAsc() []int {
	ids := make([]int, 0, len(ezfyCfg.equipments))
	for id := range ezfyCfg.equipments {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}

// equipShopList 商城在售装备（★ 按**部位**分组）
//
// ★ 用户要求（2026-09-22）：
//   - 「散件也上吧，价格按加成 10 钻石到 50 钻石不等」→ 系列单件（可单穿）+ 纯散件都上架；
//   - 「商城前端做好看点」→ 按 11 个部位分组，前端直接铺成表格。
//
// 上架范围 = **散件**（有系列名的单件 或 不属于任何套装的纯散件）。
// ★ 第一批套装（set_id 1~17，Series 为空）仍**不上架** —— 它们只能开宝箱，
//
//	否则「套装装备只能通过宝箱开启」这条规则就形同虚设。
func (h *EzfyHandler) equipShopList() ([]gin.H, []gin.H) {
	slots := []string{}
	bySlot := map[string][]gin.H{}
	items := []gin.H{}
	// 按 ID 升序遍历（equipments 是 map，直接 range 顺序随机 → 部位分组顺序会乱跳）
	for _, id := range ezfyEquipIDsAsc() {
		e := ezfyCfg.equipments[id]
		if e.PriceGold <= 0 && e.PriceDiamond <= 0 {
			continue
		}
		// ★ 只上架「军官装备」类的散件 —— 也就是《装备距离伤害表》里的 11 个部位。
		//   老版的 武器/防具/饰品/珠宝（Type 是它们自己）不属于那张表，别混进来。
		if e.Type != "军官装备" {
			continue
		}
		if e.SetId > 0 && e.Series == "" {
			continue // 第一批套装：只能开宝箱
		}
		slot := e.EquipSlot()
		if _, ok := bySlot[slot]; !ok {
			slots = append(slots, slot)
		}
		ee := e
		it := h.equipShopItem(&ee)
		bySlot[slot] = append(bySlot[slot], it)
		items = append(items, it)
	}
	out := []gin.H{}
	for _, slot := range slots {
		pieces := bySlot[slot]
		sort.Slice(pieces, func(a, b int) bool {
			return pieces[a]["id"].(int) < pieces[b]["id"].(int)
		})
		out = append(out, gin.H{"slot": slot, "count": len(pieces), "pieces": pieces})
	}
	return out, items
}

func (h *EzfyHandler) equipShopItem(e *model.EzfyCfgEquipment) gin.H {
	sold := e.Stock == 0
	return gin.H{
		"id": e.ID, "name": e.Name, "type": e.Type, "slot": e.EquipSlot(), "tier": e.Tier,
		"tier_name": ezfyTierName(e.Tier), "level": e.Level,
		"military": e.Military, "logistics": e.Logistics, "learning": e.Learning,
		"set_id": e.SetId, "set_name": h.ezfySetName(e.SetId),
		"price_gold": e.PriceGold, "price_diamond": e.PriceDiamond,
		"stock": e.Stock, "sold_out": sold, "effect": e.Effect, "des": e.Des,
		// ★ 六项战斗属性（军官装备）
		"series": e.Series, "enhance": e.Enhance, "enhance_max": e.EnhanceMax,
		"dmg": e.Dmg, "def": e.Def, "hp": e.Hp, "move": e.Move, "crit": e.Crit, "crit_dmg": e.CritDmg,
	}
}

// EquipShop GET /games/ezfy/equipshop —— 装备商城（套装分组）
func (h *EzfyHandler) EquipShop(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	city := h.getOrCreateCity(uid)
	h.refreshCity(uid, &city)
	slots, items := h.equipShopList()
	resp.OK(c, gin.H{
		// ★ slots = 按部位分组（前端铺表格）；items = 扁平列表（检索/兼容用）
		"slots": slots, "items": items,
		"gold": city.Gold, "diamond": h.ensureProfile(uid).Diamond,
	})
}

// EquipShopBuy POST /games/ezfy/equipshop/buy  {cfg_id, count, currency: gold|diamond}
//
// 用黄金或钻石买装备（套装件），买入直接进玩家背包，可就地穿到军官身上。
func (h *EzfyHandler) EquipShopBuy(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	city := h.getOrCreateCity(uid)
	h.calcResource(&city)
	var req struct {
		CfgId    int    `json:"cfg_id"`
		Count    int    `json:"count"`
		Currency string `json:"currency"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if req.Count <= 0 {
		req.Count = 1
	}
	if req.Count > 999 {
		resp.ParamError(c, "单次最多购买 999 件")
		return
	}
	cfg := ezfyCfg.equipment(req.CfgId)
	if cfg == nil {
		h.fail(c, "装备不存在")
		return
	}
	// ★ 商城只卖**散件**：系列单件（Series != ""，可单穿）与纯散件（SetId == 0）都能买；
	//   第一批套装（set_id 1~17、Series 为空）只能开宝箱 —— 这里拦住被绕过前端直接传 cfg_id。
	if cfg.SetId > 0 && cfg.Series == "" {
		h.fail(c, "该套装装备只能通过宝箱开启，商城不出售")
		return
	}
	useDiamond := req.Currency == "diamond"
	price := cfg.PriceGold
	unit := "黄金"
	if useDiamond {
		price = cfg.PriceDiamond
		unit = "钻石"
	}
	if price <= 0 {
		h.fail(c, "该装备不支持用"+unit+"购买")
		return
	}
	if cfg.Stock == 0 {
		h.fail(c, "该装备已售罄")
		return
	}
	if cfg.Stock > 0 && req.Count > cfg.Stock {
		h.fail(c, fmt.Sprintf("库存不足(剩余%d件)", cfg.Stock))
		return
	}
	total := price * int64(req.Count)
	if useDiamond {
		prof := h.ensureProfile(uid)
		if prof.Diamond < total {
			h.fail(c, fmt.Sprintf("钻石不足(需要%d钻石, 当前%d)", total, prof.Diamond))
			return
		}
		if err := h.DB.Model(&model.EzfyProfile{}).Where("id = ?", prof.ID).
			Update("diamond", prof.Diamond-total).Error; err != nil {
			h.fail(c, "扣钻石失败："+err.Error())
			return
		}
	} else {
		if city.Gold < total {
			h.fail(c, fmt.Sprintf("黄金不足(需要%d黄金, 当前%d)", total, city.Gold))
			return
		}
		city.Gold -= total
		h.saveCityRes(&city)
	}
	for i := 0; i < req.Count; i++ {
		h.addEquipment(&city, cfg)
	}
	if cfg.Stock > 0 {
		h.DB.Model(&model.EzfyCfgEquipment{}).Where("id = ?", cfg.ID).
			Update("stock", gorm.Expr("stock - ?", req.Count))
		h.cfgsReload()
	}
	h.done(c, "", fmt.Sprintf("购买成功: %s×%d，花费%d%s", cfg.Name, req.Count, total, unit))
}
