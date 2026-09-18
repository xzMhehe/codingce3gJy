package handler

import (
	"encoding/json"
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
	ezfyRecruitCostPerLevel = 500   // 招募费用 = 名将等级 × 该值
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
func (h *EzfyHandler) orderListByCity(cityId uint) []model.EzfyOrder {
	var orders []model.EzfyOrder
	h.DB.Where("city_id = ? AND status IN (0,1)", cityId).Find(&orders)
	return orders
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

// recruitCandidates 生成候选名将（仅 recruit=1，剔除已拥有，随机取 军校等级 个）
func (h *EzfyHandler) recruitCandidates(uid uint, academyLevel int) []model.EzfyCfgGeneral {
	h.cfgs()
	owned := h.ownedGeneralIds(uid)
	pool := []model.EzfyCfgGeneral{}
	for _, g := range ezfyCfg.generals {
		if g.Recruit != 1 || owned[g.ID] {
			continue
		}
		pool = append(pool, g)
	}
	if len(pool) == 0 {
		return pool
	}
	randShuffleGenerals(pool)
	n := maxInt(1, minInt(academyLevel, 10))
	if n > len(pool) {
		n = len(pool)
	}
	return pool[:n]
}

func randShuffleGenerals(list []model.EzfyCfgGeneral) {
	for i := len(list) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		list[i], list[j] = list[j], list[i]
	}
}

// recruitInfo 当日候选（首次访问生成并落库；展示时再次剔除已拥有/已停用）
func (h *EzfyHandler) recruitInfo(uid uint, academyLevel int) ([]model.EzfyCfgGeneral, int, int) {
	h.cfgs()
	date := time.Now().Format("2006-01-02")
	var rec model.EzfyRecruit
	err := h.DB.Where("user_id = ? AND recruit_date = ?", uid, date).First(&rec).Error
	if err != nil {
		cands := h.recruitCandidates(uid, academyLevel)
		rec = model.EzfyRecruit{UserId: uid, RecruitDate: date, RefreshCount: 0,
			Candidates: joinGeneralIds(cands)}
		h.DB.Create(&rec)
	}
	cands := h.parseGeneralIds(rec.Candidates)
	owned := h.ownedGeneralIds(uid)
	out := []model.EzfyCfgGeneral{}
	for _, g := range cands {
		if g.Recruit != 1 || owned[g.ID] {
			continue
		}
		out = append(out, g)
	}
	return out, maxInt(0, ezfyRecruitRefreshLimit-rec.RefreshCount), ezfyRecruitRefreshLimit
}

func joinGeneralIds(list []model.EzfyCfgGeneral) string {
	s := ""
	for i, g := range list {
		if i > 0 {
			s += ","
		}
		s += strconv.Itoa(g.ID)
	}
	return s
}

func (h *EzfyHandler) parseGeneralIds(ids string) []model.EzfyCfgGeneral {
	out := []model.EzfyCfgGeneral{}
	if ids == "" {
		return out
	}
	for _, part := range splitComma(ids) {
		id, err := strconv.Atoi(part)
		if err != nil {
			continue
		}
		if g := ezfyCfg.general(id); g != nil {
			out = append(out, *g)
		}
	}
	return out
}

func splitComma(s string) []string {
	out := []string{}
	cur := ""
	for _, ch := range s {
		if ch == ',' {
			if cur != "" {
				out = append(out, cur)
			}
			cur = ""
			continue
		}
		cur += string(ch)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

// refreshRecruit 刷新当日候选（每日限 5 次）
func (h *EzfyHandler) refreshRecruit(uid uint, academyLevel int) string {
	date := time.Now().Format("2006-01-02")
	var rec model.EzfyRecruit
	err := h.DB.Where("user_id = ? AND recruit_date = ?", uid, date).First(&rec).Error
	used := 0
	if err == nil {
		used = rec.RefreshCount
	}
	if used >= ezfyRecruitRefreshLimit {
		return "今日刷新次数已用完(每天限" + strconv.Itoa(ezfyRecruitRefreshLimit) + "次, 明天0点重置)"
	}
	cands := h.recruitCandidates(uid, academyLevel)
	if err != nil {
		rec = model.EzfyRecruit{UserId: uid, RecruitDate: date}
	}
	rec.RefreshCount = used + 1
	rec.Candidates = joinGeneralIds(cands)
	if rec.ID == 0 {
		h.DB.Create(&rec)
	} else {
		h.DB.Model(&model.EzfyRecruit{}).Where("id = ?", rec.ID).
			Updates(map[string]interface{}{"refresh_count": rec.RefreshCount, "candidates": rec.Candidates})
	}
	return ""
}

// recruitOfficer 招募名将：需军校≥1、参谋部≥1、容量未满、未拥有、黄金=等级×500
func (h *EzfyHandler) recruitOfficer(city *model.EzfyCity, generalId int) string {
	h.calcResource(city)
	cfg := ezfyCfg.general(generalId)
	if cfg == nil {
		return "名将不存在"
	}
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
	if h.ownedGeneralIds(city.UserID)[generalId] {
		return "已拥有该名将, 无法重复招募"
	}
	cost := int64(cfg.Level) * ezfyRecruitCostPerLevel
	if city.Gold < cost {
		return "黄金不足(招募需要" + strconv.FormatInt(cost, 10) + "黄金)"
	}
	city.Gold -= cost
	h.saveCityRes(city)
	star := cfg.Star
	if star <= 0 {
		star = 5
	}
	o := model.EzfyOfficer{
		CityId: int64(city.ID), GeneralId: cfg.ID, Name: cfg.Name, Star: star,
		Level: 1, Exp: 0, Military: cfg.Military, Logistics: cfg.Logistics, Learning: cfg.Learning,
		Loyalty: ezfyOfficerLoyaltyMax, Skill: "", Equipment: "",
		Position: ezfyPositionNone, Status: 0, IsCaptive: 0, UpdateTime: time.Now(),
	}
	h.DB.Create(&o)
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
	cands := h.recruitCandidates(uid, academy)
	if err != nil {
		rec = model.EzfyRecruit{UserId: uid, RecruitDate: date, RefreshCount: 0}
	}
	rec.Candidates = joinGeneralIds(cands)
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
	return 10 + o.Logistics/20
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

// officerBaseBonus 军官基础战斗加成（军事属性 + 装备军事加成）
func (h *EzfyHandler) officerBaseBonus(o *model.EzfyOfficer) int {
	if o == nil {
		return 0
	}
	bonus := o.Military
	for _, m := range officerEquipped(o) {
		bonus += jsonInt(m["military"])
	}
	return bonus
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
		case "突击":
			bonus += 10
		case "鼓舞":
			bonus += 5
		case "爆破", "空袭", "海战", "装甲突击":
			bonus += 8
		}
	}
	return bonus
}

// officerBattleBonus 带队军官总攻击加成（军事 + 装备 + 技能）
func (h *EzfyHandler) officerBattleBonus(o *model.EzfyOfficer) int {
	return h.officerBaseBonus(o) + h.officerSkillBattleBonus(o)
}

// officerGuardBonus 城守守城防御加成（+10 及 防御/掩体+10、生命/鼓舞+5）
func (h *EzfyHandler) officerGuardBonus(o *model.EzfyOfficer) int {
	if o == nil {
		return 0
	}
	bonus := 10
	for _, s := range officerSkills(o) {
		switch s {
		case "防御", "掩体":
			bonus += 10
		case "生命", "鼓舞":
			bonus += 5
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
	case "突击":
		return "攻击+10%"
	case "鼓舞":
		return "攻击+5%"
	case "爆破", "空袭", "海战", "装甲突击":
		return "攻击+8%"
	case "防御", "掩体":
		return "守军防御+10%"
	case "生命":
		return "守军防御+5%"
	case "掠夺":
		return "掠夺资源+10%"
	case "移速":
		return "行军移速+10%"
	case "攻速":
		return "战斗速度+10%"
	case "修养":
		return "伤兵恢复+10%"
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

// officerGoOut 军官出征/归来：出征时状态=1 且忠诚-5，归零自动离职
func (h *EzfyHandler) officerGoOut(city *model.EzfyCity, name string, goOut bool) {
	o := h.officerByName(city.ID, name)
	if o == nil {
		return
	}
	if goOut {
		loyalty := o.Loyalty - 5
		if loyalty < 0 {
			loyalty = 0
		}
		h.DB.Model(&model.EzfyOfficer{}).Where("id = ?", o.ID).
			Updates(map[string]interface{}{"status": 1, "loyalty": loyalty})
		if loyalty <= 0 {
			h.DB.Delete(&model.EzfyOfficer{}, o.ID)
			h.addReport(city.UserID, 6, "将领离职: "+o.Name,
				o.Name+"因忠诚度归零而离职, 离开了你的城市。", "")
		}
		return
	}
	h.DB.Model(&model.EzfyOfficer{}).Where("id = ?", o.ID).Update("status", 0)
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
		}
	}
	if rand.Intn(100) < 40 {
		if jewel := h.randomJewel(terrain); jewel != nil {
			h.addEquipment(city, jewel)
			desc += " 珠宝:" + jewel.Name
		}
	}
	return desc
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

// captureWildlandOfficer 战胜寇城/特殊野地按概率俘虏守将
func (h *EzfyHandler) captureWildlandOfficer(city *model.EzfyCity, special bool) string {
	h.cfgs()
	chance := 20
	if special {
		chance = 40
	}
	if rand.Intn(100) >= chance {
		return ""
	}
	staff := h.buildingLevel(city.ID, ezfyBuildingStaff)
	if staff < 1 {
		return ""
	}
	owned := h.ownedGeneralIds(city.UserID)
	pool := []model.EzfyCfgGeneral{}
	for _, g := range ezfyCfg.generals {
		if g.Recruit == 1 && !owned[g.ID] {
			pool = append(pool, g)
		}
	}
	if len(pool) == 0 {
		return ""
	}
	g := pool[rand.Intn(len(pool))]
	star := 1 + rand.Intn(4) // 俘虏星级 1-4
	o := model.EzfyOfficer{
		CityId: int64(city.ID), GeneralId: g.ID, Name: g.Name, Star: star,
		Level: 1, Exp: 0,
		Military:  g.Military * star / 5,
		Logistics: g.Logistics * star / 5,
		Learning:  g.Learning * star / 5,
		Loyalty:   30, Skill: "", Equipment: "",
		Position: ezfyPositionNone, Status: 0, IsCaptive: 1, UpdateTime: time.Now(),
	}
	h.DB.Create(&o)
	return " 俘虏敌将:" + o.Name + "(" + strconv.Itoa(star) + "星, 忠诚30) 可前往军校收编"
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
		views = append(views, gin.H{
			"id": o.ID, "name": o.Name, "star": o.Star, "level": o.Level, "exp": o.Exp,
			"military": o.Military, "logistics": o.Logistics, "learning": o.Learning,
			"loyalty": o.Loyalty, "position": o.Position, "position_name": ezfyPositionName(o.Position),
			"status": o.Status, "status_name": ezfyOfficerStatusName(o),
			"is_captive": o.IsCaptive, "skills": skills,
			"equip_count": len(officerEquipped(o)),
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
		cands, left, limit := h.recruitInfo(uid, academy)
		views := []gin.H{}
		for _, g := range cands {
			views = append(views, gin.H{
				"id": g.ID, "name": g.Name, "level": g.Level, "star": g.Star,
				"military": g.Military, "logistics": g.Logistics, "learning": g.Learning,
				"cost": g.Level * ezfyRecruitCostPerLevel, "des": g.Des, "skill": g.Skill,
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

// AcadeRecruitDo POST /games/ezfy/acade/recruit/:id
func (h *EzfyHandler) AcadeRecruitDo(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	city := h.getOrCreateCity(uid)
	gid, _ := strconv.Atoi(c.Param("id"))
	h.fail(c, h.recruitOfficer(&city, gid))
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
		list = append(list, gin.H{
			"id": g.ID, "name": g.Name, "level": g.Level, "star": g.Star,
			"military": g.Military, "logistics": g.Logistics, "learning": g.Learning,
			"source": g.Source, "skill": g.Skill, "des": g.Des, "owned": owned[g.ID],
		})
	}
	sort.Slice(list, func(i, j int) bool { return list[i]["level"].(int) > list[j]["level"].(int) })
	resp.OK(c, gin.H{"generals": list})
}
