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
//   状态被自愈回 0 → 同一军官能被二次出征（用户反馈的 bug）。
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
func (h *EzfyHandler) officerCount(cityId uint) int {
	n := 0
	for _, o := range h.officerList(cityId) {
		if o.IsCaptive != 1 {
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

// ============ 军校招募：随机普通军官 ============
//
// 按用户要求: 军校招募给的是**随机生成的普通军官**(参考 conquer.html 的 Jeremy·Lee 26级 1星 31/52/38 26000),
// 名将(cfg_general 的 31 位)**只能由管理端发放**, 不再出现在招募池里。

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

// ezfyOfficerDraft 军校招募候选(随机普通军官)
type ezfyOfficerDraft struct {
	Key       string `json:"key"`
	Name      string `json:"name"`
	Level     int    `json:"level"`
	Star      int    `json:"star"`
	Logistics int    `json:"logistics"`
	Military  int    `json:"military"`
	Learning  int    `json:"learning"`
	Cost      int64  `json:"cost"`
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

// rollOfficerDrafts 生成 n 个随机军官候选; 等级/属性随军校等级提高
func rollOfficerDrafts(academyLevel, n int) []ezfyOfficerDraft {
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
		drafts := rollOfficerDrafts(academyLevel, maxInt(1, minInt(academyLevel, 10)))
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
	drafts := rollOfficerDrafts(academyLevel, maxInt(1, minInt(academyLevel, 10)))
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
	o := model.EzfyOfficer{
		CityId: int64(city.ID), GeneralId: 0, Name: pick.Name, Star: pick.Star,
		Level: pick.Level, Exp: 0,
		Military: pick.Military, Logistics: pick.Logistics, Learning: pick.Learning,
		Loyalty: ezfyOfficerLoyaltyMax, Skill: "", Equipment: "",
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
	drafts := rollOfficerDrafts(academy, maxInt(1, minInt(academy, 10)))
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

func (h *EzfyHandler) addEquipment(city *model.EzfyCity, cfg *model.EzfyCfgEquipment) {
	e := model.EzfyEquipment{
		UserId: city.UserID, CityId: int64(city.ID), CfgId: cfg.ID, Name: cfg.Name,
		Type: cfg.Type, Tier: cfg.Tier, Military: cfg.Military, Logistics: cfg.Logistics,
		Learning: cfg.Learning, Level: cfg.Level, OfficerId: 0, CreatedAt: time.Now(),
	}
	h.DB.Create(&e)
}

// equipItem 穿戴装备：等级达标 + 同部位唯一（珠宝不限）
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
	equipped := officerEquipped(o)
	if e.Type != "珠宝" {
		for _, m := range equipped {
			if t, _ := m["type"].(string); t == e.Type {
				return "已穿戴同类型装备"
			}
		}
	}
	h.DB.Model(&model.EzfyEquipment{}).Where("id = ?", e.ID).Update("officer_id", int64(o.ID))
	equipped = append(equipped, map[string]interface{}{
		"id": e.ID, "name": e.Name, "type": e.Type,
		"military": e.Military, "logistics": e.Logistics, "learning": e.Learning,
	})
	h.saveOfficerEquipment(o, equipped)
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
		h.saveOfficerEquipment(o, kept)
	}
	h.DB.Model(&model.EzfyEquipment{}).Where("id = ?", e.ID).Update("officer_id", 0)
	return ""
}

func (h *EzfyHandler) saveOfficerEquipment(o *model.EzfyOfficer, list []map[string]interface{}) {
	b, _ := json.Marshal(list)
	h.DB.Model(&model.EzfyOfficer{}).Where("id = ?", o.ID).Update("equipment", string(b))
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
	_, log, _ := officerEffective(&o)
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
//   玩家穿上带后勤/学识的装备后数字一动不动，看起来就是「穿装备没效果」。
func officerEquipBonus(o *model.EzfyOfficer) (mil, log, lea int) {
	for _, m := range officerEquipped(o) {
		mil += jsonInt(m["military"])
		log += jsonInt(m["logistics"])
		lea += jsonInt(m["learning"])
	}
	return
}

// officerEffective 军官的**有效属性**（自身 + 装备）
func officerEffective(o *model.EzfyOfficer) (mil, log, lea int) {
	if o == nil {
		return
	}
	em, el, ee := officerEquipBonus(o)
	return o.Military + em, o.Logistics + el, o.Learning + ee
}

// officerBaseBonus 军官基础战斗加成（军事属性 + 装备军事加成）
func (h *EzfyHandler) officerBaseBonus(o *model.EzfyOfficer) int {
	if o == nil {
		return 0
	}
	em, _, _ := officerEquipBonus(o)
	return o.Military + em
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

// officerGuardBonus 城守守城防御加成（+10 及 防御/掩体+10、生命/鼓舞+5）
// officerGuardBonus 军官防御加成（基础 10 + 学识/20 + 技能）
//
// ★ 学识原来只展示、不参与任何计算（原版也是这样），加上装备的学识加成也没去处。
//   这里把学识接到「防御」上，让三项属性各有用途：
//   军事→攻击、后勤→市长产量、学识→防御。
func (h *EzfyHandler) officerGuardBonus(o *model.EzfyOfficer) int {
	if o == nil {
		return 0
	}
	_, _, lea := officerEffective(o)
	bonus := 10 + lea/20
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

// addOfficerExp 军官获得经验（升级经验 = 等级×200，每级随机 +2 属性）
func (h *EzfyHandler) addOfficerExp(city *model.EzfyCity, officerId uint, exp int64) {
	var o model.EzfyOfficer
	if err := h.DB.First(&o, officerId).Error; err != nil {
		return
	}
	o.Exp += exp
	leveled := false
	for o.Exp >= int64(o.Level)*200 {
		o.Exp -= int64(o.Level) * 200
		o.Level++
		switch rand.Intn(3) {
		case 0:
			o.Military += 2
		case 1:
			o.Logistics += 2
		default:
			o.Learning += 2
		}
		leveled = true
	}
	h.DB.Model(&model.EzfyOfficer{}).Where("id = ?", o.ID).
		Updates(map[string]interface{}{"exp": o.Exp, "level": o.Level,
			"military": o.Military, "logistics": o.Logistics, "learning": o.Learning})
	if leveled {
		h.addReport(city.UserID, 6, "将领升级: "+o.Name,
			o.Name+"在战斗中成长, 升到了"+strconv.Itoa(o.Level)+"级!", "")
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

// randomEquipment 随机取指定品质的非珠宝装备
func (h *EzfyHandler) randomEquipment(tier int) *model.EzfyCfgEquipment {
	pool := []model.EzfyCfgEquipment{}
	for _, e := range ezfyCfg.equipments {
		if e.Type != "珠宝" && e.Tier == tier {
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
	o := model.EzfyOfficer{
		CityId: int64(city.ID), GeneralId: g.ID, Name: g.Name, Star: star,
		Level: maxInt(1, level), Exp: 0,
		Military: g.Military, Logistics: g.Logistics, Learning: g.Learning,
		Loyalty: 30, Skill: "", Equipment: "",
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
				Loyalty: 30, Skill: o.Skill, Equipment: o.Equipment,
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
		em, el, ee := officerEffective(o)
		views = append(views, gin.H{
			"id": o.ID, "name": o.Name, "star": o.Star, "level": o.Level, "exp": o.Exp,
			"military": o.Military, "logistics": o.Logistics, "learning": o.Learning,
			// ★ 含装备加成的有效属性（前端展示「基础(+装备)」）
			"military_total": em, "logistics_total": el, "learning_total": ee,
			"equip_military": em - o.Military, "equip_logistics": el - o.Logistics,
			"equip_learning": ee - o.Learning,
			"loyalty":        o.Loyalty, "position": o.Position, "position_name": ezfyPositionName(o.Position),
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
		"used":          h.officerCount(city.ID),
		"gold":          city.Gold,
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
	em, el, ee := officerEffective(o)
	bag := []gin.H{}
	for _, e := range h.equipmentList(uid) {
		bag = append(bag, gin.H{
			"id": e.ID, "name": e.Name, "type": e.Type, "tier": e.Tier,
			"tier_name": ezfyTierName(e.Tier),
			"military":  e.Military, "logistics": e.Logistics, "learning": e.Learning,
			"level": e.Level, "officer_id": e.OfficerId, "worn": e.OfficerId > 0,
		})
	}
	resp.OK(c, gin.H{
		"officer": gin.H{
			"id": o.ID, "name": o.Name, "star": o.Star, "level": o.Level, "exp": o.Exp,
			"military": o.Military, "logistics": o.Logistics, "learning": o.Learning,
			// ★ 有效属性（基础 + 装备），前端展示成「33 (+5) = 38」
			"military_total": em, "logistics_total": el, "learning_total": ee,
			"equip_military": em - o.Military, "equip_logistics": el - o.Logistics,
			"equip_learning": ee - o.Learning,
			"attack":         h.officerBattleBonus(o), "defence": h.officerGuardBonus(o),
			"loyalty": o.Loyalty, "position": o.Position, "position_name": ezfyPositionName(o.Position),
			"status": o.Status, "status_name": ezfyOfficerStatusName(o), "is_captive": o.IsCaptive,
			"exp_need": o.Level * 200,
		},
		"skills": skillViews, "all_skills": allSkills,
		"equipped": officerEquipped(o), "bag": bag, "gold": city.Gold,
	})
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
	h.fail(c, h.refreshRecruit(uid, academy))
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
	h.fail(c, h.hireOfficerDraft(&city, uid, req.Key))
}

// OfficerGrant POST /games/ezfy/officers/:id/grant
func (h *EzfyHandler) OfficerGrant(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	city := h.getOrCreateCity(uid)
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	h.fail(c, h.grantOfficer(&city, id))
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
		h.fail(c, h.forgetSkill(&city, id, req.SkillId))
		return
	}
	h.fail(c, h.learnSkill(&city, id, req.SkillId))
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
		h.fail(c, h.unequipItem(&city, req.EquipId))
		return
	}
	h.fail(c, h.equipItem(&city, id, req.EquipId))
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
	h.fail(c, h.setOfficerPosition(&city, id, req.Position))
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
		h.fail(c, h.recruitCaptive(&city, id))
		return
	}
	h.fail(c, h.freeOfficer(&city, id))
}

// OfficerExile POST /games/ezfy/officers/:id/exile
func (h *EzfyHandler) OfficerExile(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	city := h.getOrCreateCity(uid)
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	h.fail(c, h.exileOfficer(&city, id))
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
		})
	}
	cfgList := []gin.H{}
	for _, e := range ezfyCfg.equipments {
		cfgList = append(cfgList, gin.H{"id": e.ID, "name": e.Name, "type": e.Type, "tier": e.Tier,
			"tier_name": ezfyTierName(e.Tier), "military": e.Military, "logistics": e.Logistics,
			"learning": e.Learning, "level": e.Level, "des": e.Des})
	}
	sort.Slice(cfgList, func(i, j int) bool { return cfgList[i]["id"].(int) < cfgList[j]["id"].(int) })
	resp.OK(c, gin.H{"bag": bag, "all": cfgList})
}

// OfficerGenerals GET /games/ezfy/officers/generals —— 名将图鉴（按等级倒序）
func (h *EzfyHandler) OfficerGenerals(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	owned := h.ownedGeneralIds(uid)
	list := []gin.H{}
	for _, g := range ezfyCfg.generals {
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
