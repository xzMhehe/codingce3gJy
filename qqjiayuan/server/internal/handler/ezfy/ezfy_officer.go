package ezfy

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"sync"
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
//   - 职位：市长(产量+(10+后勤/20)*3%)、城守(守城防御+10%)

const (
	ezfyRecruitRefreshLimit = 5     // 军校每小时刷新次数上限（2026-09-28 每天5次 → 每1小时5次）
	ezfyRecruitCostPerLevel = 1000  // 招募费用 = 军官等级 × 该值(参考 conquer.html: 26级→26000)
	ezfyGrantCost           = 10000 // 赏赐一次消耗黄金
	ezfyOfficerMaxSkill     = 3     // 军官技能上限
	ezfyOfficerLoyaltyMax   = 100   // 忠诚上限
	ezfyCaptiveMinLoyalty   = 40    // 收编俘虏后的最低忠诚
	ezfyBuildingAcademy     = 9     // 军校
	ezfyBuildingStaff       = 10    // 参谋部
	ezfyPositionNone        = 0
	ezfyPositionMayor       = 1 // 市长
	ezfyPositionGuard       = 2 // 城守
	// ★ 2026-09-29 市长/城守在任期间每分钟获得被动经验（合理成长，避免职位军官等级定格）
	ezfyDutyExpPerMin = 3
)

// recruitCycleKey 军校刷新计数周期 key（**小时窗口**）。
//
// ★ 2026-09-28 军校免费刷新次数从「每天 5 次」改为「每 1 小时 5 次」。
//
//	用 "2006010215"（10 位 YmdH，如 2026092815 = 2026-09-28 第15点）作为周期标识，
//	整点窗口变化即新周期，正好装进 ezfy_recruit.recruit_date 的 varchar(10)。
func recruitCycleKey() string {
	// ★ 2026-09-28 刷新周期按「天 / 小时」可配（二战系统配置默认按小时）。
	//   按小时用 10 位 YmdH（装得进 varchar(10)）；按天用 2006-01-02。
	if ezfyRecruitCycleHourly() {
		return time.Now().Format("2006010215")
	}
	return time.Now().Format("2006-01-02")
}

// ezfyOfficerMaxLevel 普通军官最高等级（用户规则：「普通军官最高等级 150」）
// ezfyGeneralMaxLevel 名将最高等级（2026-09-29 用户规则：「名将最高等级 350」）
//
// 所有会抬高军官等级的地方都要夹对应上限：
//
//	① 战斗加经验升级（addOfficerExp）
//	② 军校招募候选（rollOfficerDrafts）
//	③ 管理端一键生成军官（AdminEzfyGenOfficers）
//	④ 管理端直接编辑军官（AdminEzfyOfficerUpdate）
const ezfyOfficerMaxLevel = 150
const ezfyGeneralMaxLevel = 350

// officerMaxLevelOf 该军官实例的最高等级：名将（general_id>0 且池子 kind=2）350，普通军官 150。
func officerMaxLevelOf(db *gorm.DB, o *model.EzfyOfficer) int {
	if o.GeneralId <= 0 {
		return ezfyOfficerMaxLevel
	}
	var g model.EzfyCfgGeneral
	if err := db.Where("id = ?", o.GeneralId).First(&g).Error; err != nil || g.Kind != 2 {
		return ezfyOfficerMaxLevel
	}
	return ezfyGeneralMaxLevel
}

// （EzfyHandler 便捷封装，供游戏链路调用）
//
// ★ 2026-10-05 性能：改走**进程内配置缓存** ezfyCfg.general()（该表本来就在缓存里），
// 不再每个军官查一次 `SELECT * FROM ezfy_cfg_general WHERE id = ?`。
// 语义不变：名将实例(kind=2) → 名将上限；查不到/非名将 → 普通军官上限。
// 配置改动最多滞后 30 秒（与全站其它取配置的地方同一口径）。
func (h *EzfyHandler) officerMaxLevelOf(o *model.EzfyOfficer) int {
	if o.GeneralId <= 0 {
		return ezfyOfficerMaxLevel
	}
	if g := ezfyCfg.general(int(o.GeneralId)); g != nil && g.Kind == 2 {
		return ezfyGeneralMaxLevel
	}
	return ezfyOfficerMaxLevel
}

// ezfyStarItemID 「星级徽章」的道具 cfg_id（ItemType 19）
const ezfyStarItemID = 23

// ezfySkillBookItemID 「军官技能书」的道具 cfg_id（ItemType 11）
const ezfySkillBookItemID = 15

// ezfyOfficerRenameCardItemID 「军官改名卡」的道具 cfg_id（ItemType 21）
const ezfyOfficerRenameCardItemID = 25

// ============ 基础查询 ============

// officerList 城市军官列表（自愈：出征中但已无对应行军命令的军官解除出征态）
func (h *EzfyHandler) officerList(cityId uint) []model.EzfyOfficer {
	var list []model.EzfyOfficer
	var orders []model.EzfyOrder
	// ★ 2026-10-05 性能：军官表与「命令表（自愈判定用）」**并行**取 —— 原来必须先拿军官列表、
	//   发现有出征态(status=1)的军官才去查命令表，两条**串行**跨 WAN 往返（线上 ~250ms）。
	//   命令表只取 id/officer 两列，代价很小；并行后固定 1 个 RTT。
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); h.DB.Where("city_id = ?", cityId).Order("id ASC").Find(&list) }()
	go func() { defer wg.Done(); orders = h.orderListByCity(cityId) }()
	wg.Wait()
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
	// ★ 2026-10-05 性能：两个调用方都只读 Officer 字段（判军官是否在外），
	//   只取 id/officer，不再把 troops/result/battle_result 这些大字段拉回来。
	h.DB.Select("id, officer").Where("city_id = ? AND status IN (0,1,2)", cityId).Find(&orders)
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

// captiveCountOf 按给定军官列表统计俘虏数（纯内存，不查库）。
func captiveCountOf(list []model.EzfyOfficer) int {
	n := 0
	for i := range list {
		if list[i].IsCaptive == 1 {
			n++
		}
	}
	return n
}

// ezfyCaptivePerStaff 战俘营每级参谋部可关押的俘虏数
//
// ★★ 2026-10-06 用户规则：**参谋部容量 与 战俘营容量 是两套，分开算**——
//   - 参谋部容量（在职军官位） = 参谋部等级           → officerCapacity
//   - 战俘营容量（关押俘虏）   = 参谋部等级 × 4       → captiveCapacity
//   两者各数各的：officerCount 只数在职（不含俘虏），captiveCount 只数俘虏。
const ezfyCaptivePerStaff = 4

// officerCapacity 参谋部容量 = 参谋部等级（在职军官位；没参谋部 = 0）。
//
// ★ 所有「军官位够不够」的判定（军校招募 / 收编俘虏）都必须走这里。
func (h *EzfyHandler) officerCapacity(cityId uint) int {
	return h.buildingLevel(cityId, ezfyBuildingStaff)
}

// captiveCapacity 战俘营容量 = **当前城市**的参谋部等级 × 4。
//
// ★★ 2026-10-06 用户规则：战俘营按城市算、绑在玩家城市上 ——
// 容量看的是「俘虏所在那座城」的参谋部，而不是玩家全服所有城的总和。
func (h *EzfyHandler) captiveCapacity(cityId uint) int {
	return h.buildingLevel(cityId, ezfyBuildingStaff) * ezfyCaptivePerStaff
}

// captiveCount 该城关押的俘虏数（会查库；已有军官列表时用 captiveCountOf）。
func (h *EzfyHandler) captiveCount(cityId uint) int {
	return captiveCountOf(h.officerList(cityId))
}

// officerOfMine 取「该玩家名下任意城池」的军官（含跨城），并返回该军官真实所在的城。
//
// ★★ 2026-10-06 修「战俘不能释放」：原来 freeOfficer / recruitCaptive 用
// `officerOf(当前城.ID, id)` 查，玩家切城 / 列表刷新与点击之间切换城市时会直接
// 返回 nil → 报「武将不存在」，按钮点了没反应。
// 这里改成按玩家找，并把军官真实所在城一起返回（容量按那座城算）。
func (h *EzfyHandler) officerOfMine(uid uint, id int64) (*model.EzfyOfficer, *model.EzfyCity) {
	var o model.EzfyOfficer
	if err := h.DB.First(&o, id).Error; err != nil {
		return nil, nil
	}
	var city model.EzfyCity
	if err := h.DB.Where("id = ? AND user_id = ?", o.CityId, uid).First(&city).Error; err != nil {
		return nil, nil
	}
	return &o, &city
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

// officerEquipped 解析军官已穿戴装备（按部位去重：同部位只保留**最后**穿上的那件）
//
// ★ 2026-09-24 用户反馈「同一个部位能穿戴多个」：老数据里已经存在同部位多件
//
//	（管理端改「穿戴军官」不校验 + 老代码别名不归一），解析时统一兜底去重，
//	保证展示/属性/套装进度计算都不会把重复件再算进去。
func officerEquipped(o *model.EzfyOfficer) []map[string]interface{} {
	out := []map[string]interface{}{}
	if o == nil || o.Equipment == "" {
		return out
	}
	_ = json.Unmarshal([]byte(o.Equipment), &out)
	if len(out) < 2 {
		return out
	}
	// 从后往前扫：每个部位第一次碰到的一定是最后穿上的那件，前面重复的直接剔除
	seen := map[string]bool{}
	kept := make([]map[string]interface{}, 0, len(out))
	for i := len(out) - 1; i >= 0; i-- {
		m := out[i]
		slot, _ := m["slot"].(string)
		if slot == "" {
			slot, _ = m["type"].(string)
		}
		if seen[model.EzfySlotCanon(slot)] {
			continue
		}
		seen[model.EzfySlotCanon(slot)] = true
		kept = append(kept, m)
	}
	// 倒回来，保持原来「先穿的在前」的相对顺序
	for l, r := 0, len(kept)-1; l < r; l, r = l+1, r-1 {
		kept[l], kept[r] = kept[r], kept[l]
	}
	return kept
}

// ============ 军校招募 ============

// ============ 军校招募：从军官池抽普通军官 ============
//
// ★ 2026-09-22 
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
	// ★ 2026-09-26：等级改成「直接沿用池子里该军官的原始等级」后，
	//   原来按军校等级算的随机跨度 `span` 就用不到了（academyLevel 参数保留给调用方，不再参与取值）。
	_ = academyLevel
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
		// ★ 2026-09-26 「军官池普通军官等级要有差距」：
		//   池子里的 `level` 现在是该军官的**原始等级**（种子按星级分层随机 1~150，
		//   属性也按这个等级同比缩放过了），招募时**直接沿用** ——
		//   不再按「5 + 随机(军校等级×8)」现算（那样池子的等级就没意义了）。
		lv := g.Level
		if lv <= 0 || lv > ezfyOfficerMaxLevel {
			lv = ezfyOfficerMaxLevel
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

// recruitInfo 当周期(小时)候选(首次访问生成并落库)
//
// p 可选：调用方（/acade/recruit）已取到玩家档案时传入 → 省一次 profile 查询
// （军校刷新次数上限要读 profile.recruit_free_limit，原来是再查一次 profile）。
func (h *EzfyHandler) recruitInfo(uid uint, academyLevel int, p ...*model.EzfyProfile) ([]ezfyOfficerDraft, int, int) {
	h.cfgs()
	date := recruitCycleKey()
	var rec model.EzfyRecruit
	err := h.DB.Where("user_id = ? AND recruit_date = ?", uid, date).First(&rec).Error
	if err != nil {
		drafts := h.rollOfficerDrafts(academyLevel, maxInt(1, minInt(academyLevel, 10)))
		rec = model.EzfyRecruit{UserId: uid, RecruitDate: date, RefreshCount: 0,
			Candidates: joinDrafts(drafts)}
		h.DB.Create(&rec)
	}
	// ★ 上限支持按玩家覆盖（管理端「军校免费刷次数」维护）
	var prof *model.EzfyProfile
	if len(p) > 0 {
		prof = p[0]
	}
	limit := h.ezfyRecruitFreeLimitWith(prof)
	return parseDrafts(rec.Candidates), maxInt(0, limit-rec.RefreshCount), limit
}

func (h *EzfyHandler) refreshRecruit(uid uint, academyLevel int) string {
	date := recruitCycleKey()
	var rec model.EzfyRecruit
	err := h.DB.Where("user_id = ? AND recruit_date = ?", uid, date).First(&rec).Error
	used := 0
	if err == nil {
		used = rec.RefreshCount
	}
	limit := h.ezfyRecruitFreeLimit(uid)
	if used >= limit {
		return "本小时刷新次数已用完(每小时限" + strconv.Itoa(limit) +
			"次, 下一个整点重置；也可以在军校直接使用「招生简章」刷新)"
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
	if cap := h.officerCapacity(city.ID); h.officerCount(city.ID) >= cap {
		return "参谋部容量不足(参谋部" + strconv.Itoa(staff) + "级容纳" + strconv.Itoa(cap) + "名军官)"
	}
	date := recruitCycleKey()
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
	// ★ 2026-09-22：原始属性写进 base_*（重修书洗点回退到这个值）。
	// ★★ 2026-09-27 用户规则修正：军官池里的初始属性已包含该等级的全部加点，
	//   招募时**不再**按「每级 1 点」发可用属性点（旧逻辑 99 级会多给 98 点），
	//   只有招募后打架升级（每升 1 级 +1 点，见 L1404）才积累可用点。
	// ★★ 2026-10-06 用户规则「招募的军官也要把军官池 id 维护到库里」：
	//   原来这里写死 `GeneralId: 0`（怕和名将混），后果是普通军官**无法回查军官池**
	//   —— 改名后按名字也找不回来（`officerPoolAttr` 的兜底就是按名字查），
	//   丢官/被俘后连"原本是池子里哪一条"都不知道，没法恢复。
	//   现在直接落 `pick.PoolId`（候选就是从军官池抽的，见 rollOfficerDrafts）。
	//   ⚠️ 「是不是名将」一律用 `ezfyCfg.isGeneral(id)`（池子 kind==2）判定，
	//   别再用 `GeneralId > 0` 当名将标志 —— 普通军官现在也有 id 了。
	//   （兜底随机生成的候选 PoolId=0，仍是 0，没有池子条目可对。）
	o := model.EzfyOfficer{
		CityId: int64(city.ID), GeneralId: pick.PoolId, Name: pick.Name, Star: pick.Star,
		Level: pick.Level, Exp: 0,
		Military: pick.Military, Logistics: pick.Logistics, Learning: pick.Learning,
		BaseMilitary: pick.Military, BaseLogistics: pick.Logistics, BaseLearning: pick.Learning,
		FreePoints: 0,
		Loyalty:    ezfyOfficerLoyaltyMax, Skill: "", Equipment: "",
		Position: ezfyPositionNone, Status: 0, IsCaptive: 0, Source: model.EzfyOfficerSourceRecruit, UpdateTime: time.Now(),
	}
	// ★ 2026-10-06 事故加固：原来忽略 Create 的 error，写库失败会「扣了黄金却没军官」。
	//   （线上 ezfy_officer.source 列缺失时就是这个表现。）失败要退款并如实告知。
	if err := h.DB.Create(&o).Error; err != nil {
		log.Printf("ezfy 军校招募写入失败 city=%d name=%s: %v", city.ID, pick.Name, err)
		city.Gold += pick.Cost
		h.saveCityRes(city)
		return "招募失败, 请稍后重试"
	}
	h.DB.Model(&model.EzfyRecruit{}).Where("id = ?", rec.ID).Update("candidates", joinDrafts(kept))
	// ★ 五星军官值得全服看一眼（）
	if pick.Star >= 5 {
		h.ezfySysChat("恭喜玩家 %s 在军校招募到五星军官 %s！", h.ezfyProfileName(uid), o.Name)
	}
	return ""
}

// refreshRecruitFree 免费刷新当周期候选名将（招生简章用，不消耗每小时刷新次数）
func (h *EzfyHandler) refreshRecruitFree(uid uint) string {
	city := h.getOrCreateCity(uid)
	academy := h.buildingLevel(city.ID, ezfyBuildingAcademy)
	if academy < 1 {
		return "需要先建造军校"
	}
	date := recruitCycleKey()
	var rec model.EzfyRecruit
	err := h.DB.Where("user_id = ? AND recruit_date = ?", uid, date).First(&rec).Error
	drafts := h.rollOfficerDrafts(academy, maxInt(1, minInt(academy, 10)))
	// ★ 2026-09-30 「使用招生简章出五星军官的概率」：
	//   按配置概率把候选中的 1 名置为 5 星（100 = 必出），提高招生简章刷出 5 星的几率。
	drafts = h.boostFiveStarDraft(drafts)
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

// boostFiveStarDraft 招生简章刷新时，按配置概率把候选中的 1 名提升为 5 星。
//
// 概率 = ezfyRecruitFiveStarRate()（1~100，默认 1 = 1%）；100 = 100% 必出 5 星。
// 提升前会先看已有候选是否含 5 星：含就不重复提升；不然挑最高的非 5 星置为 5 星。
func (h *EzfyHandler) boostFiveStarDraft(drafts []ezfyOfficerDraft) []ezfyOfficerDraft {
	rate := ezfyRecruitFiveStarRate()
	if rate >= 100 || (len(drafts) > 0 && rand.Intn(100) < rate) {
		for i := range drafts {
			if drafts[i].Star >= 5 {
				return drafts
			}
		}
		for i := range drafts {
			if drafts[i].Star < 5 {
				drafts[i].Star = 5
				break
			}
		}
	}
	return drafts
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

// ezfyTreasureLoyalty 赏赐宝物加的忠诚：按品质档位（tier 1~4：初级/中级/高级/特殊），最高 +50
var ezfyTreasureLoyalty = []int{0, 10, 20, 35, 50}

// treasureGrantOfficer 赏赐宝物：消耗 1 件背包未穿戴的宝物，按品质加忠诚（最高 +50）
func (h *EzfyHandler) treasureGrantOfficer(city *model.EzfyCity, officerId int64, equipId uint) string {
	h.calcResource(city)
	o := h.officerOf(city.ID, officerId)
	if o == nil {
		return "武将不存在"
	}
	if o.Loyalty >= ezfyOfficerLoyaltyMax {
		return "忠诚已满"
	}
	var e model.EzfyEquipment
	if err := h.DB.First(&e, equipId).Error; err != nil {
		return "宝物不存在"
	}
	if e.UserId != city.UserID {
		return "宝物不属于你"
	}
	if e.OfficerId != 0 {
		return "这件宝物已穿戴, 请先卸下"
	}
	// ★ 2026-09-28 只有「采集宝物」能赏赐 ——
	//   装备表里还混着步枪/钢盔/合金装甲这类普通装备，它们不是宝物，不能拿来换忠诚。
	//   宝物签到抽的也是同一池（9 种珠宝），所以签到领的宝物天然可赏赐。
	if !ezfyCollectibleTreasureNames()[e.Name] {
		return fmt.Sprintf("「%s」不是采集宝物, 不可用于赏赐", e.Name)
	}
	gain := 10
	if e.Tier > 0 && e.Tier < len(ezfyTreasureLoyalty) {
		gain = ezfyTreasureLoyalty[e.Tier]
	}
	// 消耗宝物 + 加忠诚（上限 100）
	h.DB.Delete(&model.EzfyEquipment{}, e.ID)
	h.DB.Model(&model.EzfyOfficer{}).Where("id = ?", o.ID).
		Update("loyalty", minInt(ezfyOfficerLoyaltyMax, o.Loyalty+gain))
	// ★ 文案里不写机器码（原来写「品质1」玩家看不懂）→ 用中文档位名
	return fmt.Sprintf("赏赐【%s】(品质%s), 忠诚 +%d", e.Name, ezfyTierName(e.Tier), gain)
}

// learnSkill 学习技能：消耗 1 本「军官技能书」（道具 15），最多 3 个，出征中不可学
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
	if h.itemCount(city.UserID, ezfySkillBookItemID) <= 0 {
		return "没有「军官技能书」，可在商城购买"
	}
	h.consumeItem(city.UserID, ezfySkillBookItemID, "军官学技能")
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

// addEquipment 生成装备实例进背包
func (h *EzfyHandler) addEquipment(city *model.EzfyCity, cfg *model.EzfyCfgEquipment) {
	e := model.EzfyEquipment{
		UserId: city.UserID, CityId: int64(city.ID), CfgId: cfg.ID, Name: cfg.Name,
		Type: cfg.Type, Tier: cfg.Tier, Military: cfg.Military, Logistics: cfg.Logistics,
		Learning: cfg.Learning, Level: cfg.Level, OfficerId: 0, CreatedAt: time.Now(),
		// ★ 部位统一存归一后的规范名，老实例的原始字符串由判重时归一兜底
		Slot: model.EzfySlotCanon(cfg.EquipSlot()), SetId: cfg.SetId,
		// ★ 六项战斗属性随实例带走（进战斗计算用）
		Series: cfg.Series, Enhance: cfg.Enhance,
		Dmg: cfg.Dmg, Def: cfg.Def, Hp: cfg.Hp, Move: cfg.Move, Crit: cfg.Crit, CritDmg: cfg.CritDmg,
	}
	h.DB.Create(&e)
}

// equipItem 穿戴装备：等级达标 + 同部位唯一（含珠宝，任何部位都只能穿一件）
//
// ★ 2026-09-22：同部位判定改用「Slot（留空回落 Type）」，
//
//	这样套装里的头/肩/胸/腰/手/足/饰品/挂件/勋章 9 件互不冲突，能整套穿上。
//
// ★ 2026-09-24：「同一个部位只能穿戴一个」，取消珠宝的叠穿例外。
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
	slot := model.EzfySlotCanon(e.EquipSlot())
	equipped := officerEquipped(o)
	for _, m := range equipped {
		// ★ 2026-09-23 修复「同部位能穿多件」：判重前先把**双方**的部位别名归一。
		//   否则「传说英雄[头盔]」(部位'头盔') 和 「赤色锤镰[头部]」(部位'头部')
		//   这种同名部位不同写法会同时通过，导致一个部位穿了两件。
		if t, _ := m["slot"].(string); model.EzfySlotCanon(t) == slot {
			return "已穿戴同部位装备(" + slot + ")"
		}
		// 老数据没有 slot 字段 → 回落到 type
		if t, _ := m["slot"].(string); t == "" {
			if ot, _ := m["type"].(string); model.EzfySlotCanon(ot) == slot {
				return "已穿戴同部位装备(" + slot + ")"
			}
		}
	}
	// ★ 顺序很重要：先把军官身上的装备列表写成功，再改装备行的 officer_id。
	//   反过来的话（先改 officer_id 再写 JSON），一旦 JSON 写失败就会留下
	//   「装备显示已穿戴、但军官身上没有」的半截状态 —— 实测踩过（varchar(500) 截断）。
	equipped = append(equipped, map[string]interface{}{
		"id": e.ID, "name": e.Name, "type": e.Type, "slot": slot, "set_id": e.SetId,
		"tier":     e.Tier,
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

// rebuildOfficerEquipJSON 按装备行重建某军官的已穿戴装备 JSON（同部位去重，重复件放回背包）
//
// ★ 2026-09-24 配套修复「同部位能穿戴多件」：管理端改「穿戴军官」后装备行与
//
//	军官 JSON 会脱节（更别说可能直接穿出重复部位），统一用这个函数把两边状态拉齐：
//	同一部位只留 id 最大（最后穿上）的那件，其余 officer_id 置 0 放回背包。
func (h *EzfyHandler) rebuildOfficerEquipJSON(officerId int64) {
	if officerId <= 0 {
		return
	}
	var items []model.EzfyEquipment
	h.DB.Where("officer_id = ?", officerId).Order("id").Find(&items)
	seen := map[string]bool{}
	list := []map[string]interface{}{}
	drops := []int64{}
	for i := len(items) - 1; i >= 0; i-- {
		e := items[i]
		if seen[model.EzfySlotCanon(e.EquipSlot())] {
			drops = append(drops, int64(e.ID))
			continue
		}
		seen[model.EzfySlotCanon(e.EquipSlot())] = true
		list = append(list, map[string]interface{}{
			"id": e.ID, "name": e.Name, "type": e.Type, "slot": model.EzfySlotCanon(e.EquipSlot()), "set_id": e.SetId,
			"tier":     e.Tier,
			"military": e.Military, "logistics": e.Logistics, "learning": e.Learning,
			"series": e.Series, "enhance": e.Enhance,
			"dmg": e.Dmg, "def": e.Def, "hp": e.Hp, "move": e.Move, "crit": e.Crit, "crit_dmg": e.CritDmg,
		})
	}
	for l, r := 0, len(list)-1; l < r; l, r = l+1, r-1 {
		list[l], list[r] = list[r], list[l]
	}
	b, err := json.Marshal(list)
	if err != nil {
		return
	}
	h.DB.Model(&model.EzfyOfficer{}).Where("id = ?", officerId).Update("equipment", string(b))
	if len(drops) > 0 {
		h.DB.Model(&model.EzfyEquipment{}).Where("id IN ?", drops).Update("officer_id", 0)
	}
}

// equipmentOwnerId 查询某装备穿戴者（卸下后跳转详情用）
func (h *EzfyHandler) equipmentOwnerId(uid uint, equipId int64) int64 {
	var e model.EzfyEquipment
	if err := h.DB.First(&e, equipId).Error; err != nil || e.UserId != uid {
		return 0
	}
	return e.OfficerId
}

// transferOfficerEquipsToCaptive 被俘军官的随身装备随俘虏转移（2026-09-29 用户规则）：
//
//	把原玩家军官身上的装备行解绑并挂到俘虏名下：
//	  - user_id 改为攻方玩家（原玩家装备对应的减少，无法再卸下/查看）
//	  - officer_id 改为俘虏 id（「装上俘虏官」）
//	  - original_user_id 记下原归属玩家，供【收编归新玩家 / 释放返还旧玩家】
func (h *EzfyHandler) transferOfficerEquipsToCaptive(captiveID int64, oldOfficerID int64, atkUid, origUid uint) {
	var rows []model.EzfyEquipment
	h.DB.Where("officer_id = ? AND user_id = ?", oldOfficerID, origUid).Find(&rows)
	for i := range rows {
		r := rows[i]
		h.DB.Model(&model.EzfyEquipment{}).Where("id = ?", r.ID).
			Updates(map[string]interface{}{
				"officer_id":       captiveID,
				"user_id":          atkUid,
				"original_user_id": origUid,
			})
	}
}

// returnCaptiveEquips 释放俘虏时，把随俘装备返还给原玩家（2026-09-29 用户规则）：
//
//	按 original_user_id 还原 user_id，并解绑 officer_id（回到原玩家背包）。
func (h *EzfyHandler) returnCaptiveEquips(captiveID int64) {
	var rows []model.EzfyEquipment
	h.DB.Where("officer_id = ? AND original_user_id > 0", captiveID).Find(&rows)
	for i := range rows {
		r := rows[i]
		h.DB.Model(&model.EzfyEquipment{}).Where("id = ?", r.ID).
			Updates(map[string]interface{}{
				"officer_id":       0,
				"user_id":          r.OriginalUserId,
				"original_user_id": 0,
			})
	}
}

// equipIsCaptiveWorn 判断某装备是否挂在「未收编的俘虏」身上
func (h *EzfyHandler) equipIsCaptiveWorn(cityId uint, equipId int64) bool {
	var e model.EzfyEquipment
	if err := h.DB.First(&e, equipId).Error; err != nil || e.OfficerId <= 0 {
		return false
	}
	if o := h.officerOf(cityId, e.OfficerId); o != nil && o.IsCaptive == 1 {
		return true
	}
	return false
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
			Updates(map[string]interface{}{"position": ezfyPositionNone, "duty_exp_at": nil})
	}
	// ★ 2026-09-29 市长/城守在任期间按时间结算被动经验：
	//   任命时把结算基准时间设为当前，卸任(0)时清空基准(NULL) = 停止领取在职经验。
	v := map[string]interface{}{"position": position}
	if position != ezfyPositionNone {
		v["duty_exp_at"] = time.Now()
	} else {
		v["duty_exp_at"] = nil // 清空为 NULL
	}
	h.DB.Model(&model.EzfyOfficer{}).Where("id = ?", o.ID).Updates(v)
	return ""
}

// recruitCaptive 收编俘虏（忠诚至少 40，占用参谋部容量）
//
// ★ 2026-10-06 改成按 uid 找军官（跨城）—— 见 officerOfMine 的说明。
func (h *EzfyHandler) recruitCaptive(uid uint, officerId int64) string {
	o, city := h.officerOfMine(uid, officerId)
	if o == nil || city == nil {
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
	if cap := h.officerCapacity(city.ID); h.officerCount(city.ID) >= cap {
		return "参谋部容量不足(参谋部" + strconv.Itoa(staff) + "级容纳" + strconv.Itoa(cap) + "名军官)"
	}
	loyalty := o.Loyalty
	if loyalty < ezfyCaptiveMinLoyalty {
		loyalty = ezfyCaptiveMinLoyalty
	}
	h.DB.Model(&model.EzfyOfficer{}).Where("id = ?", o.ID).
		Updates(map[string]interface{}{"is_captive": 0, "loyalty": loyalty, "update_time": time.Now()})
	return ""
}

// freeOfficer 释放俘虏（★ 2026-10-06 起是**逻辑删除**，行还在库里可追溯）；
// ★ 2026-09-29 释放时把随俘装备返还给原玩家。
//
// ★ 2026-10-06 改成按 uid 找军官（跨城）—— 见 officerOfMine 的说明。
func (h *EzfyHandler) freeOfficer(uid uint, officerId int64) string {
	o, _ := h.officerOfMine(uid, officerId)
	if o == nil {
		return "武将不存在"
	}
	if o.Status == 1 {
		return "出征中无法遣散"
	}
	// 释放俘虏：随身装备按 original_user_id 返还给原玩家
	if o.IsCaptive == 1 {
		h.returnCaptiveEquips(int64(o.ID))
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

// mayorBonusPct 市长产量加成 %（★ 2026-09-27 「太少，在现有基础上翻三倍」：
//
//	(10 + 后勤/20) × 3 —— 实际产量与详情页展示都走本函数，改一处即全生效）
func (h *EzfyHandler) mayorBonusPct(cityId uint) int {
	var o model.EzfyOfficer
	if err := h.DB.Where("city_id = ? AND position = ? AND is_captive = 0", cityId, ezfyPositionMayor).
		First(&o).Error; err != nil {
		return 0
	}
	// ★ 用有效后勤（自身 + 装备），否则给市长穿后勤装备没有任何效果
	_, log, _ := h.officerEffective(&o)
	// ★ 2026-09-28 「市长加成整体可调」→ 结果 × ezfyMayorGainMult（默认 1）；0 = 关闭。
	return int(float64((10+log/20)*3) * ezfyMayorGainMult())
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

// officerOnDutyList 可带队的军官（在职、未出征、非俘虏、没有带未结束的命令）
func (h *EzfyHandler) officerOnDutyList(cityId uint) []model.EzfyOfficer {
	out := []model.EzfyOfficer{}
	for _, o := range h.officerList(cityId) {
		// ★ 2026-09-24 俘虏出征 bug 加固：与出征入口 createOrder 同一套口径，
		//   Status 必须为 0(在职) 且不是俘虏、也没有带未归队的命令。
		//   ★ 市长/城守有城务在身，不出现在可选出征军官列表。
		if o.Status == 0 && o.IsCaptive != 1 && o.Position == 0 && !h.officerBusyOrder(cityId, o.Name) {
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

// officerBaseAttr 军官的**原始属性** —— 即洗点卡的重置目标
//
// ★ 用户规则（2026-09-26）：「原始属性 = 军官池里那名武将的属性」，
//
//	升星加成同样算在「现代属性 − 原始属性」的差额里（洗点时一并退回），
//	所以这里**优先回查军官池**；池子里查不到的（后台手工生成 / 历史随机生成的军官）
//	才用实例上的 base_* 快照，最后兜底当前属性。
func officerBaseAttr(o *model.EzfyOfficer) (int, int, int) {
	if o == nil {
		return 0, 0, 0
	}
	if bm, bl, be, ok := officerPoolAttr(o); ok {
		return bm, bl, be
	}
	if o.BaseMilitary > 0 || o.BaseLogistics > 0 || o.BaseLearning > 0 {
		return o.BaseMilitary, o.BaseLogistics, o.BaseLearning
	}
	return o.Military, o.Logistics, o.Learning
}

// officerPoolAttr 该军官对应的**军官池武将属性**（原始属性基准 / 洗点重置目标）
//
// ★ 用户规则：洗点 = 洗成「军官池里那名武将的属性」，而不是实例上的 base_* 快照 ——
// 旧代码升星会把 base_* 一起加高，按快照洗点会洗不回池子初始值、退回的点数也少一截。
// 所以这里直接回查池子：
//   - 名将 / 后台发放 / **军校招募**的军官 general_id > 0，按 id 查；
//   - 只有历史老数据（2026-10-06 前招的、general_id=0）才按名字回查。
//
// 查不到（后台手工生成、历史随机生成的军官）返回 ok=false，调用方回落到 base_*。
func officerPoolAttr(o *model.EzfyOfficer) (int, int, int, bool) {
	if o == nil {
		return 0, 0, 0, false
	}
	if o.GeneralId > 0 {
		if g := ezfyCfg.general(o.GeneralId); g != nil && g.Military+g.Logistics+g.Learning > 0 {
			return g.Military, g.Logistics, g.Learning, true
		}
	}
	if g := ezfyCfg.generalByName(o.Name); g != nil {
		return g.Military, g.Logistics, g.Learning, true
	}
	return 0, 0, 0, false
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
	// ★ 2026-09-26 用户规则：普通军官的**每一项**都不得超过同星级名将（且星级递减）。
	//   光修池子和存量数据不够 —— 玩家还能用 free_points 把属性加回去，这里必须卡住。
	//   名将实例（general_id>0）本身就是基准，不限制。
	if o.GeneralId == 0 {
		if c, ok := ezfyGeneralCapByStarDB(h.DB)[o.Star]; ok {
			idx := map[string]int{"military": 0, "logistics": 1, "learning": 2}[col]
			cur := map[string]int{"military": o.Military, "logistics": o.Logistics, "learning": o.Learning}[col]
			if cur >= c[idx] {
				return fmt.Sprintf("%s 已达%d星上限(%d)，无法再加", attr, o.Star, c[idx])
			}
			if cur+count > c[idx] {
				count = c[idx] - cur // 只加得起的那部分，剩余点数留在账上
			}
		}
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
	h.officerGoOutByID(o.ID, goOut)
}

// officerGoOutByID 同上，但调用方**已经有军官对象**时用它 —— 省掉一次
// `SELECT * FROM ezfy_officer WHERE city_id = ? AND name = ?`（跨 WAN ~120ms）。
func (h *EzfyHandler) officerGoOutByID(officerID uint, goOut bool) {
	if officerID == 0 {
		return
	}
	if goOut {
		// ★ 第九轮用户规则：**派遣/出征不掉忠心**（原来每次 -5，归零就离职，玩家很反感）。
		//   只有打了败仗才掉，且掉的量按战损合理计算（见 officerLoseLoyalty / 战斗结算）。
		h.DB.Model(&model.EzfyOfficer{}).Where("id = ?", officerID).Update("status", 1)
		return
	}
	h.DB.Model(&model.EzfyOfficer{}).Where("id = ?", officerID).Update("status", 0)
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
// ★ 用户规则「军官最高等级 150 / 名将最高等级 350」：满级后不再升级，多余经验直接丢弃
// （不丢的话经验会无限累积，将来放开上限会一次性跳很多级）。
func (h *EzfyHandler) addOfficerExp(city *model.EzfyCity, officerId uint, exp int64) {
	var o model.EzfyOfficer
	if err := h.DB.First(&o, officerId).Error; err != nil {
		return
	}
	// ★ 2026-09-30 修复「野地军官/俘虏能升级」：俘虏(IsCaptive=1)未收编不能升级，
	//   否则经验书/战斗经验会作用到野地守将身上。
	if o.IsCaptive == 1 {
		return
	}
	maxLv := h.officerMaxLevelOf(&o)
	o.Exp += exp
	gained := 0
	for o.Level < maxLv && o.Exp >= int64(o.Level)*200 {
		o.Exp -= int64(o.Level) * 200
		o.Level++
		gained++
	}
	// 满级后不保留经验
	if o.Level >= maxLv {
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

// accrueDutyExp 市长/城守在任期间按时间结算被动经验（★ 2026-09-29 
// 「市长当久了等级一直不变」→ 让带职位的军官也能合理成长）。
//
// calcResource 懒结算时随军官列表一起结算：按自上次结算以来的分钟数 × 每分钟经验，
// 通过 addOfficerExp 走统一的升级/加点逻辑。
//
// ⚡ 并发安全：结算基准时间只有「最先写进去的请求」能推前（条件更新抢占），
// 后续并发请求 WHERE 命中不了旧基准 → 不计，避免重复发经验（与 checkBuildingDone 同款手法）。
// ★★ 2026-10-05 性能（线上实测：这是 /buildings、/view 里**最大的单笔开销**）：
//
//	改造前**每个**在任军官串行跑 4 条查询 ——
//	  UPDATE duty_exp_at(写 240ms) → SELECT officer(读 120ms) → SELECT cfg_general(读 120ms)
//	  → UPDATE officer exp(写 240ms) ≈ 720ms/人；市长 + 城守就是 **~1.5s**，
//	  占 /buildings 总耗时（线上实测 2.6s）的一半以上。
//
//	现在改成「按旧基准分组 → 一组一条 CAS UPDATE → 一条批量 CASE UPDATE 写经验/等级」：
//	  N 个军官从 4N 条降到 **2 条**（且 2 条里 1 条是写、1 条是写）。
//	  CAS 语义完全保留（只有把基准推前的请求才结算经验，不会并发重复发经验）。
func (h *EzfyHandler) accrueDutyExp(city *model.EzfyCity, officers []model.EzfyOfficer) {
	var list []model.EzfyOfficer
	if len(officers) > 0 {
		list = officers
	} else {
		list = h.officerList(city.ID)
	}
	now := time.Now()

	// ① 分组：键 = 该军官当前的 duty_exp_at（同一批里通常只有一组）
	type dutyGroup struct {
		base int64
		idx  []int
	}
	var groups []*dutyGroup
	byBase := map[int64]*dutyGroup{}
	for i := range list {
		o := &list[i]
		if o.Position != ezfyPositionMayor && o.Position != ezfyPositionGuard {
			continue
		}
		if o.Status == 1 || o.IsCaptive == 1 {
			continue // 出征中/俘虏不领在职经验
		}
		if o.DutyExpAt == nil {
			// 老数据/未初始化：只打一次基准，不一次性补一大堆经验
			h.DB.Model(&model.EzfyOfficer{}).Where("id = ? AND duty_exp_at IS NULL", o.ID).
				Update("duty_exp_at", now)
			continue
		}
		if now.Sub(*o.DutyExpAt) < time.Minute {
			continue
		}
		b := o.DutyExpAt.UnixMilli()
		g := byBase[b]
		if g == nil {
			g = &dutyGroup{base: b}
			byBase[b] = g
			groups = append(groups, g)
		}
		g.idx = append(g.idx, i)
	}
	for _, g := range groups {
		ids := make([]uint, 0, len(g.idx))
		for _, i := range g.idx {
			ids = append(ids, list[i].ID)
		}
		// ② 一组一条 CAS：把基准从旧值推到 now，只有抢到的请求继续结算经验
		res := h.DB.Model(&model.EzfyOfficer{}).
			Where("id IN ? AND duty_exp_at <= ?", ids, time.UnixMilli(g.base)).
			Update("duty_exp_at", now)
		if res.Error != nil || res.RowsAffected == 0 {
			continue
		}
		exp := int64(now.Sub(time.UnixMilli(g.base)).Minutes()) * ezfyDutyExpPerMin
		if exp <= 0 {
			continue
		}
		if int(res.RowsAffected) != len(ids) {
			// 罕见：同组里有行被并发请求先推走了 → 退回逐行 CAS 兜底，绝不重复发经验
			for _, i := range g.idx {
				o := list[i]
				r := h.DB.Model(&model.EzfyOfficer{}).
					Where("id = ? AND duty_exp_at <= ?", o.ID, time.UnixMilli(g.base)).
					Update("duty_exp_at", now)
				if r.Error == nil && r.RowsAffected > 0 {
					h.addOfficerExp(city, o.ID, exp)
				}
			}
			continue
		}
		h.accrueDutyExpBatch(city, list, g.idx, exp)
	}
}

// accrueDutyExpBatch 把一组军官的 exp/level/free_points 用**一条 CASE UPDATE** 落库。
//
// 等级/加点在内存里算（与 addOfficerExp 同一套公式），升级播报只给真正升级的人发。
func (h *EzfyHandler) accrueDutyExpBatch(city *model.EzfyCity, list []model.EzfyOfficer, idx []int, exp int64) {
	type rowRes struct {
		id     uint
		exp    int64
		lv     int
		free   int
		gained int
		name   string
	}
	rows := make([]rowRes, 0, len(idx))
	for _, i := range idx {
		o := &list[i]
		if o.IsCaptive == 1 {
			continue
		}
		maxLv := h.officerMaxLevelOf(o)
		ne, lv, gained := o.Exp+exp, o.Level, 0
		for lv < maxLv && ne >= int64(lv)*200 {
			ne -= int64(lv) * 200
			lv++
			gained++
		}
		if lv >= maxLv {
			ne = 0 // 满级后不保留经验
		}
		rows = append(rows, rowRes{id: o.ID, exp: ne, lv: lv, free: o.FreePoints + gained,
			gained: gained, name: o.Name})
		// ★ 2026-10-05：把新值同步回内存行 —— 调用方（/officers/skills、/acade/recruit 等）
		//   可以直接复用这份列表展示，不必为了「展示最新等级」再查一次军官表。
		o.Exp, o.Level, o.FreePoints = ne, lv, o.FreePoints+gained
	}
	if len(rows) == 0 {
		return
	}
	ids := make([]uint, 0, len(rows))
	var expCase, lvCase, freeCase strings.Builder
	expCase.WriteString("CASE id")
	lvCase.WriteString("CASE id")
	freeCase.WriteString("CASE id")
	expArgs := make([]interface{}, 0, len(rows)*2)
	lvArgs := make([]interface{}, 0, len(rows)*2)
	freeArgs := make([]interface{}, 0, len(rows)*2)
	for _, r := range rows {
		ids = append(ids, r.id)
		expCase.WriteString(" WHEN ? THEN ?")
		lvCase.WriteString(" WHEN ? THEN ?")
		freeCase.WriteString(" WHEN ? THEN ?")
		expArgs = append(expArgs, r.id, r.exp)
		lvArgs = append(lvArgs, r.id, r.lv)
		freeArgs = append(freeArgs, r.id, r.free)
	}
	expCase.WriteString(" END")
	lvCase.WriteString(" END")
	freeCase.WriteString(" END")
	h.DB.Model(&model.EzfyOfficer{}).Where("id IN ?", ids).Updates(map[string]interface{}{
		"exp":         gorm.Expr(expCase.String(), expArgs...),
		"level":       gorm.Expr(lvCase.String(), lvArgs...),
		"free_points": gorm.Expr(freeCase.String(), freeArgs...),
	})
	for _, r := range rows {
		if r.gained <= 0 {
			continue
		}
		h.addReport(city.UserID, 6, "将领升级: "+r.name,
			r.name+"在战斗中成长, 升到了"+strconv.Itoa(r.lv)+"级, 获得"+
				strconv.Itoa(r.gained)+"点属性点(可前往 [军官] 详情页分配)!", "")
	}
}

// OfficersOnDuty GET /games/ezfy/officers/onduty —— 出征界面可选的带队军官
func (h *EzfyHandler) OfficersOnDuty(c *gin.Context) {
	uid := middleware.GetUID(c)
	// ★ 2026-10-05 性能：ezfyPageSettle = 档案+城市列表 + 一次并行取数 + 快照懒结算（~3 个 RTT 封顶）
	city, snap := h.ezfyPageSettle(uid)
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
	resp.OK(c, gin.H{"officers": list, "hq_level": snap.buildingLevelsOf(h, city.ID)[13]})
}

// OfficerDispatch POST /games/ezfy/officers/:id/dispatch
//
// 城市列表的 [派遣]：把当前城市的某名军官调往自己的另一座城市。
// 规则：军官必须属于当前城、不在出征中、不是俘虏；目标城必须是自己的城且不是本城。
func (h *EzfyHandler) OfficerDispatch(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	ezfyPageCacheDel(uid)
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
	// ★ 2026-09-29 先前 中级/高级/特殊 散件掉率太高（30%/14%/6%），
	//   统一调低 → 中级17% (roll<25) / 高级6% (roll<8) / 特殊2% (roll<2)，
	//   省出的概率全部归到 初级(初级散件变多)。
	// ★ 2026-09-30 玩家反馈高级地掉装备略多：中级 12%(roll<18) / 高级 4%(roll<4) / 特殊 1%(roll<1)
	// ★ 2026-10-05 「战斗掉落高级宝物（狙击步枪）概率可配」→ 三个阈值改读「二战系统配置」
	//   （中级 drop_t2 / 高级 drop_t3 / 特殊 drop_t4，默认 18/4/1；0/负 → 回落默认）。
	t2 := ezfyLimitOr(ezfyCfg.limit.DropT2, 18)
	t3 := ezfyLimitOr(ezfyCfg.limit.DropT3, 4)
	t4 := ezfyLimitOr(ezfyCfg.limit.DropT4, 1)
	if level >= 3 && roll < t2 {
		tier = 2
	}
	if level >= 6 && roll < t3 {
		tier = 3
	}
	if level >= 9 && roll < t4 {
		tier = 4
	}
	if special {
		if tier < 3 {
			tier = 3
		}
		// ★ 2026-09-30 玩家反馈高级地掉装备偏多：活动野地不再必定掉，降为 85%
		// ★ 2026-10-05 该概率改读配置（drop_act_pct，默认 85；0/负 → 回落 85）
		dropChance = ezfyLimitOr(ezfyCfg.limit.DropActPct, 85)
	} else if m := maxInt(maxInt(t2, t3), t4); m > dropChance {
		// 非活动野地：阈值本身即掉率，避免把阈值配得 >80 时被 dropChance 卡掉
		dropChance = m
	}
	if dropChance > 100 {
		dropChance = 100
	}
	if roll < dropChance {
		if cfg := h.randomEquipment(tier); cfg != nil {
			h.addEquipment(city, cfg)
			desc += " 宝物[" + ezfyTierName(tier) + "]:" + cfg.Name
			// ★ 系统消息（战斗掉落的装备要能看到）
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

// randomTerrainTreasure 从地形对应的「采集宝物池」随机取一件(用户规范, 2026-09-23)。
//
// 平原/沿海平原没有珠宝 → 返回 nil, 采集不掉宝。
// 宝物只在采集中掉落; 战斗不再掉装备/珠宝。
func (h *EzfyHandler) randomTerrainTreasure(terrain int) *model.EzfyCfgEquipment {
	names := ezfyTerrainTreasureNames[terrain]
	if len(names) == 0 {
		return nil
	}
	name := names[rand.Intn(len(names))]
	for _, e := range ezfyCfg.equipments {
		if e.Name == name {
			return &e
		}
	}
	return nil
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
	return h.createCaptiveOfficer(city, g, level, special, 0)
}

// createCaptiveOfficer 把指定军官池军官作为战俘抓到攻方城（概率/参谋部容量判定 + 写库）
//
// ★ 2026-09-29 活动野地也能配守将（普通军官/名将都可选），胜利后复用同一套俘虏逻辑。
// 　 g 为军官池条目；level 决定俘虏等级（夹在名将350/普通150）；special 为特殊目标时概率翻倍；
// 　 rateOverride 为显式概率%（0=按星级默认；1~100 直接覆盖，不受默认 60% 上限限制）。
func (h *EzfyHandler) createCaptiveOfficer(city *model.EzfyCity, g *model.EzfyCfgGeneral, level int, special bool, rateOverride int) string {
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
	// 活动野地显式配置了被俘虏概率 → 覆盖默认（0~100，不受 60% 上限限制）
	if rateOverride > 0 {
		if rateOverride > 100 {
			rateOverride = 100
		}
		chance = rateOverride
	}
	if rand.Intn(100) >= chance {
		return ""
	}
	if h.buildingLevel(city.ID, ezfyBuildingStaff) < 1 {
		return "" // 没有参谋部, 无法收押
	}
	// 战俘营容量（★ 2026-10-06 用户规则：战俘营容量 = 参谋部等级 × 4，与在职军官位分开算）
	if h.captiveCount(city.ID) >= h.captiveCapacity(city.ID) {
		return ""
	}
	// ★ 俘虏到的就是配置里那位**军官池军官**（属性/星级/等级取自军官池）
	//   等级同样夹在对应上限以内（名将 350 / 普通 150）
	// ★★ 2026-09-30 用户反馈「野地军官为什么还有等级」：原来用**野地等级**当俘虏等级
	//   （野地10级→俘到10级守将），而军官池现在统一 1 级 → 战俘营出现 10级/6级 俘虏。
	//   改为跟随军官池该军官的等级（g.Level，池子统一 1 级），不再跟野地等级。
	maxLv := ezfyOfficerMaxLevel
	if g.Kind == 2 {
		maxLv = ezfyGeneralMaxLevel
	}
	captiveLv := g.Level
	if captiveLv < 1 {
		captiveLv = 1
	}
	if captiveLv > maxLv {
		captiveLv = maxLv
	}
	o := model.EzfyOfficer{
		CityId: int64(city.ID), GeneralId: g.ID, Name: g.Name, Star: star,
		Level: captiveLv, Exp: 0,
		Military: g.Military, Logistics: g.Logistics, Learning: g.Learning,
		// ★ 原始属性 = 军官池里的值
		BaseMilitary: g.Military, BaseLogistics: g.Logistics, BaseLearning: g.Learning,
		// ★★ 2026-09-27 用户规则：军官池属性已含该等级加点，俘虏时不再按「每级 1 点」发可用点，
		//   只有后续升级（每升 1 级 +1 点）才积累
		FreePoints: 0,
		Loyalty:    30, Skill: "", Equipment: "",
		Position: ezfyPositionNone, Status: 0, IsCaptive: 1, Source: model.EzfyOfficerSourceWildland, UpdateTime: time.Now(),
	}
	// ★★ 2026-10-06 事故加固：原来这里忽略 Create 的 error。
	//   线上 `ezfy_officer.source` 列缺失时 INSERT 报 1054，战俘被静默丢弃、
	//   战报却照样写「已收入我方战俘营」→ 玩家反馈「将领没进战俘营」。
	//   写入失败必须留下日志，且不要谎报成功。
	if err := h.DB.Create(&o).Error; err != nil {
		log.Printf("ezfy 野地战俘写入失败 city=%d general=%d name=%s: %v", city.ID, g.ID, g.Name, err)
		return ""
	}
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
	// 战俘营还有空位才收得下战俘（★ 2026-10-06：战俘营容量 = 参谋部等级 × 4，只数俘虏）
	room := h.captiveCapacity(atkCity.ID) - h.captiveCount(atkCity.ID)
	// ★★ 2026-10-05 用户规则「玩家抢玩家的名将不受重复卡控」：
	//   攻击方**已经拥有**该名将时，也不拦 —— 忠诚归零照样叛逃成俘（管理端标注「抢玩家获取」）。
	//   （2026-10-04 的「PvP 也要卡控」规则已被推翻；系统发放 / PvP 抢将均不再卡控同名将数量。）

	var defected []model.EzfyOfficer
	var stayed []string
	for i := range officers {
		o := &officers[i]
		// ★ 2026-09-30 修复「出征后自己城市的军官忠诚全掉」：
		//   本函数只惩罚**正在城内防守**的军官。出征中(status=1)/被俘(is_captive)的军官
		//   人根本不在城里、也不参与守城，不该被这次攻打扣忠诚甚至当战俘拉走。
		//   （原来 officerList 未过滤 status=1，城被攻打时在外出征的军官也一起-10~-20，
		//     归零还会被误当叛将收编 —— 玩家反馈的 bug。）
		if o.Status == 1 || o.IsCaptive == 1 {
			continue
		}
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
		if room > 0 {
			// 收编为攻方战俘(等级/属性保留, 忠诚重置为 30 待收编)
			// ★ 2026-10-05 Source=抢玩家获取：标记来源，管理端军官列表据此标注
			cap := model.EzfyOfficer{
				CityId: int64(atkCity.ID), GeneralId: o.GeneralId, Name: o.Name, Star: o.Star,
				Level: o.Level, Exp: o.Exp,
				Military: o.Military, Logistics: o.Logistics, Learning: o.Learning,
				// ★ 原始属性与可用点数一起带走，别让被俘/归降把玩家点过的点吞掉
				BaseMilitary: o.BaseMilitary, BaseLogistics: o.BaseLogistics, BaseLearning: o.BaseLearning,
				FreePoints: o.FreePoints,
				Loyalty:    30, Skill: o.Skill, Equipment: o.Equipment,
				Position: ezfyPositionNone, Status: 0, IsCaptive: 1, Source: model.EzfyOfficerSourcePvp, UpdateTime: time.Now(),
			}
			// ★★ 2026-10-06 线上事故加固：**先建俘虏，建成功了再删原军官**。
			//   原实现是「先 Delete 防守方军官、再 Create 俘虏」，且 Create 的 error 被忽略 ——
			//   一旦 INSERT 失败（线上 `ezfy_officer.source` 列缺失报 ERROR 1054），
			//   防守方军官已被删、攻方又没拿到 → 军官凭空消失（玩家反馈「将领没进战俘营」）。
			if err := h.DB.Create(&cap).Error; err != nil {
				log.Printf("ezfy PvP 战俘写入失败 atkCity=%d defCity=%d name=%s: %v",
					atkCity.ID, target.ID, o.Name, err)
				b.WriteString("\n敌方军官 " + o.Name + " 忠诚归零离去(收押失败, 请联系管理员)")
				continue
			}
			room--
			// 建俘虏成功后才把军官从原城移除
			h.DB.Delete(&model.EzfyOfficer{}, o.ID)
			// ★ 2026-09-29 用户规则：被俘军官的随身装备随俘虏转移——
			//   原玩家装备行解绑挂到俘虏名下并记录原归属（收编归新玩家 / 释放返还旧玩家）
			h.transferOfficerEquipsToCaptive(int64(cap.ID), int64(o.ID), atkCity.UserID, target.UserID)
			b.WriteString("\n敌方军官 " + o.Name + " 忠诚归零, 弃城归降, 已收入我方战俘营(随身装备随俘转移)")
			h.addReport(target.UserID, 6, "将领叛离: "+o.Name,
				o.Name+"因忠诚度归零, 弃城投敌, 加入了对"+atkCity.Name+"的阵营。\n请及时赏赐军官以维持忠诚。", "", 0, target.ID)
		} else {
			// 参谋部已满：按原规则军官忠诚归零后仍然离开原城
			h.DB.Delete(&model.EzfyOfficer{}, o.ID)
			b.WriteString("\n敌方军官 " + o.Name + " 忠诚归零离去(我方参谋部已满, 未能收押)")
			h.addReport(target.UserID, 6, "将领叛离: "+o.Name,
				o.Name+"因忠诚度归零而离开了你的城市。", "", 0, target.ID)
		}
	}
	return b.String()
}

// ============ HTTP 接口 ============

// Officers GET /games/ezfy/officers —— 军官列表 + 军校/参谋部等级
//
// ★ 2026-10-04 性能（用户反馈「/officers cache 未命中仍 3s+」）：miss 路径改两波并行。
//
//	第一波 档案+城市列表（1 RTT）定当前城；第二波 军官/俘虏/建筑/队列/科技/升星卡 并行（1 RTT）；
//	懒结算全部复用已取数据（零额外查询），总串行 RTT 从 ~12 降到 ~2。
func (h *EzfyHandler) Officers(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	// ★ 2026-10-04 性能：3s 玩家级缓存，命中零 SQL（军官相关写操作已在入口统一 Del）
	if it, ok := ezfyPageCacheGet(uid, "officers"); ok {
		resp.OK(c, it)
		return
	}
	_, city, _ := h.ezfyPageCity(uid)
	var (
		list        []model.EzfyOfficer
		capList     []model.EzfyOfficer
		buildings   []model.EzfyCityBuilding
		trainQueues []model.EzfyTrainQueue
		starCard    int
	)
	var wg sync.WaitGroup
	wg.Add(4)
	go func() { defer wg.Done(); list = h.officerList(city.ID) }()
	go func() { defer wg.Done(); buildings = h.buildingList(city.ID) }()
	go func() {
		defer wg.Done()
		h.DB.Where("city_id = ? AND status = 0", city.ID).Order("start_time ASC").Find(&trainQueues)
	}()
	go func() { defer wg.Done(); starCard = h.itemCount(uid, ezfyStarItemID) }()
	wg.Wait()
	// ★★ 2026-10-06 用户规则：**战俘营按城市分**（绑在玩家城市上）——
	//   只列**当前城**的俘虏，容量也只看当前城的参谋部等级 × 4。
	//   （2026-09-29 那版「跨城汇总」已取消。）
	//   当前城军官列表 list 本来就含俘虏，纯内存筛出来即可 —— 比原来还省一条 SQL。
	for i := range list {
		if list[i].IsCaptive == 1 {
			capList = append(capList, list[i])
		}
	}
	// 新的排前面（原实现是 `ORDER BY id DESC`）
	sort.Slice(capList, func(a, b int) bool { return capList[a].ID > capList[b].ID })
	// 懒结算复用已取数据（建筑/城市ID/训练队列/军官全在手上，零额外查询）
	// ★ 2026-10-05 性能（用户反馈「/officers 还是 1s+」）：原来这里还跑 checkTechDone +
	//   calcResource（各 2~4 条串行跨 WAN 查询）。军官页不展示科技，资源/工资结算交给
	//   /view 轮询照常推进（最多滞后一轮轮询）；这里只保留 建筑完工 + 训练队列出厂 两个
	//   **纯内存、空闲零写**的结算。
	h.checkBuildingDone(&city, buildings)
	h.collectTrainQueue(&city, trainQueues)
	// ★★ 2026-10-06 用户规则：战俘营**按城市分**（绑在玩家城市上），容量 = 当前城参谋部等级 × 4。
	//   （2026-09-29 那版「俘虏跨城汇总」已取消：多城玩家请切到俘虏所在的那座城查看。）
	// 把一批军官构造成展示视图（当前城军官 + 当前城俘虏共用同一套字段）
	build := func(rows []model.EzfyOfficer) []gin.H {
		out := []gin.H{}
		for i := range rows {
			o := &rows[i]
			skills := officerSkills(o)
			em, el, ee := h.officerEffective(o)
			sm, sl, se, activeSets := h.officerSetBonus(o)
			bm, bl, be := officerBaseAttr(o)
			// ★ 2026-10-04 名将标识 + 原名（玩家改名后仍认得出来）
			isGen := o.GeneralId > 0 && ezfyCfg.isGeneral(o.GeneralId)
			genName, genDes := "", ""
			genStar, genLevel := 0, 0
			if isGen {
				if g := ezfyCfg.general(o.GeneralId); g != nil {
					genName, genStar, genLevel = g.Name, g.Star, g.Level
					// ★ 2026-10-04 功勋介绍用内置二战文案（未收录回落 des）
					genDes = ezfyGeneralLoreOf(o.GeneralId, g.Des)
				}
			}
			out = append(out, gin.H{
				"id": o.ID, "name": o.Name, "star": o.Star, "level": o.Level, "exp": o.Exp,
				// ★ 2026-10-04 名将标识（原名 + 二战功勋背景）
				"is_general":    isGen,
				"general_name":  genName,
				"general_des":   genDes,
				"general_star":  genStar,
				"general_level": genLevel,
				"military":      o.Military, "logistics": o.Logistics, "learning": o.Learning,
				// ★ 含装备/套装加成的有效属性（前端展示「基础(+装备)」）
				"military_total": em, "logistics_total": el, "learning_total": ee,
				"equip_military": em - o.Military, "equip_logistics": el - o.Logistics,
				"equip_learning": ee - o.Learning,
				// ★ 2026-09-22：原始属性 / 可用属性点 / 已分配点数（前端加点用）
				"base_military": bm, "base_logistics": bl, "base_learning": be,
				"free_points": o.FreePoints, "used_points": officerAllocatedPoints(o),
				// ★ 2026-09-30 升星累计加点（洗点前玩家能看懂多少点是升星来的）
				"star_points":  o.StarPoints,
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
		return out
	}
	views := build(list)
	capViews := build(capList)
	// ★ 2026-10-04 性能：军校/参谋部等级用第二波已取建筑列表内存取值（零额外查询）
	blv := buildingLevelsOf(buildings)
	data := gin.H{
		"officers":      views,
		"captives":      capViews, // ★ 2026-10-06 起只含**当前城**的俘虏（战俘营按城市分）
		"academy_level": blv[ezfyBuildingAcademy],
		"staff_level":   blv[ezfyBuildingStaff],
		// ★★ 2026-10-06 用户规则：**参谋部容量 与 战俘营容量是两套，分开算**
		//   capacity         = 参谋部容量（在职军官位）= 参谋部等级
		//   captive_capacity = 战俘营容量 = 参谋部等级 × 4（只数俘虏）
		"capacity":         blv[ezfyBuildingStaff],
		"captive_capacity": blv[ezfyBuildingStaff] * ezfyCaptivePerStaff,
		"captive_used":     captiveCountOf(capList),
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
		"star_up_on": ezfyStarUpOn(), "star_rate": ezfyStarSuccessRate(),
		"star_max": ezfyStarMax(), "star_attr_gain": ezfyStarAttrGain(),
		"star_card": starCard, // ★ 2026-10-04 第二波已并行查好
	}
	ezfyPageCacheSet(uid, "officers", data)
	resp.OK(c, data)
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
//
// ★ 2026-10-04 性能（用户反馈「/officers/:id 3s+」）：miss 路径改两波并行，
//
//	背包/本城军官/物品数/建筑/训练队列 第二波一次打齐，懒结算零额外查询。
func (h *EzfyHandler) OfficerDetail(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	_, city, cities := h.ezfyPageCity(uid)
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	o := h.officerOf(city.ID, id)
	if o == nil {
		// ★ 2026-10-01 修复「点击军官有时候空白」：军官在玩家**其他城市**
		//   （派遣/增援订单结算后随军调任）时，提示准确原因，别让前端摸黑。
		var any model.EzfyOfficer
		myCityIDs := make([]int64, 0, len(cities))
		for _, c := range cities {
			myCityIDs = append(myCityIDs, int64(c.ID))
		}
		if len(myCityIDs) > 0 && h.DB.Where("id = ? AND city_id IN ?", id, myCityIDs).First(&any).Error == nil {
			resp.ParamError(c, "该军官已调往其他城市, 请到对应城市查看")
			return
		}
		resp.ParamError(c, "武将不存在")
		return
	}
	// 第二波并行：背包装备 / 本城军官 / 物品持有数 / 建筑 / 训练队列
	var (
		items        []model.EzfyEquipment
		cityOfficers []model.EzfyOfficer
		cnts         map[int]int
		buildings    []model.EzfyCityBuilding
		trainQueues  []model.EzfyTrainQueue
	)
	var wg sync.WaitGroup
	wg.Add(5)
	go func() { defer wg.Done(); items = h.equipmentList(uid) }()
	go func() { defer wg.Done(); cityOfficers = h.officerList(city.ID) }()
	go func() {
		defer wg.Done()
		cnts = h.itemCounts(uid, ezfyStarItemID, ezfyOfficerRenameCardItemID, ezfySkillBookItemID)
	}()
	go func() { defer wg.Done(); buildings = h.buildingList(city.ID) }()
	go func() {
		defer wg.Done()
		h.DB.Where("city_id = ? AND status = 0", city.ID).Order("start_time ASC").Find(&trainQueues)
	}()
	wg.Wait()
	// 懒结算复用已取数据（零额外查询）
	h.checkBuildingDone(&city, buildings)
	h.checkTechDone(&city, cityIdsOf(cities))
	h.collectTrainQueue(&city, trainQueues)
	h.calcResource(&city, cityOfficers)
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
	// ★ 2026-09-28 赏赐宝物只认「采集宝物」：背包条目带上标记，前端据此过滤可选列表
	treasureSet := ezfyCollectibleTreasureNames()
	// ★ 2026-10-04 性能（用户反馈「军官详情一直加载中」）：原实现逐件装备调
	//   equipIsCaptiveWorn（内部 2 次查库）—— 背包几百件装备就是上千次 SQL 往返，
	//   双机共 RDS 时单次详情能卡到秒级。现在当前城军官只查一次，被俘判定走内存 map。
	//   ★ 2026-10-04 items/cityOfficers 已在第二波并行取好，这里直接用。
	captive := map[int64]bool{}
	for i := range cityOfficers {
		if cityOfficers[i].IsCaptive == 1 {
			captive[int64(cityOfficers[i].ID)] = true
		}
	}
	// ★ 一键穿套装：背包里每个套装分别有件未穿戴的（officer_id=0 才在背包）
	bagSetCnt := map[int]int{}
	for i := range items {
		e := &items[i]
		// ★ 2026-09-29：挂在「未收编俘虏」身上的装备不进背包（展示在俘虏的已穿戴里）
		if e.OfficerId > 0 && captive[e.OfficerId] {
			continue
		}
		if e.OfficerId == 0 && e.SetId > 0 {
			bagSetCnt[e.SetId]++
		}
		bag = append(bag, gin.H{
			"id": e.ID, "cfg_id": e.CfgId, "name": e.Name, "type": e.Type, "tier": e.Tier,
			"tier_name": ezfyTierName(e.Tier),
			"military":  e.Military, "logistics": e.Logistics, "learning": e.Learning,
			"level": e.Level, "officer_id": e.OfficerId, "worn": e.OfficerId > 0,
			"treasure": treasureSet[e.Name],
			"slot":     e.EquipSlot(), "set_id": e.SetId, "set_name": h.ezfySetName(e.SetId),
			"series": e.Series, "enhance": e.Enhance,
			"dmg": e.Dmg, "def": e.Def, "hp": e.Hp, "move": e.Move, "crit": e.Crit, "crit_dmg": e.CritDmg,
		})
	}
	sm, sl, se, activeSets := h.officerSetBonus(o)
	bm, bl, be := officerBaseAttr(o)
	// 背包里有哪些套装可以一键穿戴（给前端「一键穿戴套装」按钮用）
	// ★ 2026-09-24 「套装差了差多少生效看不出来」→ 每个套装带上
	//   parts(总件数)/worn(该军官已穿件数)/need(还差几件生效)，前端直接展示进度。
	wornBySet := map[int]int{}
	for _, m := range officerEquipped(o) {
		if sid := jsonInt(m["set_id"]); sid > 0 {
			wornBySet[sid]++
		}
	}
	bagSets := []gin.H{}
	bSetIds := make([]int, 0, len(bagSetCnt))
	for sid := range bagSetCnt {
		bSetIds = append(bSetIds, sid)
	}
	sort.Ints(bSetIds)
	for _, sid := range bSetIds {
		name := h.ezfySetName(sid)
		if name == "" {
			continue
		}
		parts := 0
		if s := ezfyCfg.equipSet(sid); s != nil {
			parts = s.Parts
		}
		// ★ 2026-09-29 一键穿戴套装表要显示「等级」列：取该套装各件的穿戴等级需求最大值。
		setLevel := 0
		for _, e := range ezfyCfg.equipments {
			if e.SetId == sid && e.Level > setLevel {
				setLevel = e.Level
			}
		}
		bagSets = append(bagSets, gin.H{
			"set_id": sid, "name": name, "bag_count": bagSetCnt[sid],
			"parts": parts, "worn": wornBySet[sid], "need": maxInt(0, parts-wornBySet[sid]),
			"level": setLevel,
		})
	}
	// ★ 2026-10-04 升星卡/改名卡/技能书 已在第二波并行取好（原 3 条 itemCount SQL）
	// ★ 2026-10-04 名将标识 + 二战功勋背景（玩家改名后仍能认出原名与身份）
	isGen := o.GeneralId > 0 && ezfyCfg.isGeneral(o.GeneralId)
	genName, genDes := "", ""
	genStar, genLevel := 0, 0
	if isGen {
		if g := ezfyCfg.general(o.GeneralId); g != nil {
			genName, genStar, genLevel = g.Name, g.Star, g.Level
			// ★ 2026-10-04 功勋介绍用内置二战文案（未收录回落 des）
			genDes = ezfyGeneralLoreOf(o.GeneralId, g.Des)
		}
	}
	resp.OK(c, gin.H{
		"officer": gin.H{
			"id": o.ID, "name": o.Name, "star": o.Star, "level": o.Level, "exp": o.Exp,
			// ★ 2026-10-04 名将标识（原名 + 二战功勋背景，前端据此加「名将背景」tab）
			"is_general":    isGen,
			"general_name":  genName,
			"general_des":   genDes,
			"general_star":  genStar,
			"general_level": genLevel,
			"military":      o.Military, "logistics": o.Logistics, "learning": o.Learning,
			// ★ 有效属性（基础 + 装备 + 套装），前端展示成「33 (+5) = 38」
			"military_total": em, "logistics_total": el, "learning_total": ee,
			"equip_military": em - o.Military, "equip_logistics": el - o.Logistics,
			"equip_learning": ee - o.Learning,
			// ★ 2026-09-22：加点用
			"base_military": bm, "base_logistics": bl, "base_learning": be,
			"free_points": o.FreePoints, "used_points": officerAllocatedPoints(o),
			// ★ 2026-09-30 升星累计加点（洗点前玩家能看懂多少点是升星来的）
			"star_points":  o.StarPoints,
			"set_military": sm, "set_logistics": sl, "set_learning": se,
			"active_sets": activeSets,
			// ★ 套装穿戴进度（穿齐才生效；这里让前端能显示「还差 N 件」）
			"set_progress": h.officerSetProgressView(o),
			// ★ 装备六项战斗加成（直接进战斗计算）
			"battle": h.officerBattleView(o),
			// ★ 升星：星级上限 / 固定成功率 / 每星加多少 / 持有升星卡数
			"star_max": ezfyStarMax(), "star_up_on": ezfyStarUpOn(),
			"star_rate":      ezfyStarSuccessRate(),
			"star_attr_gain": ezfyStarAttrGain(), "star_card": cnts[ezfyStarItemID],
			// ★ 军官改名卡 / 军官技能书 持有数（前端改名按钮与可学技能表头展示）
			"rename_card": cnts[ezfyOfficerRenameCardItemID],
			"skill_book":  cnts[ezfySkillBookItemID],
			"attack":      h.officerBattleBonus(o), "defence": h.officerGuardBonus(o),
			"loyalty": o.Loyalty, "position": o.Position, "position_name": ezfyPositionName(o.Position),
			"status": o.Status, "status_name": ezfyOfficerStatusName(o), "is_captive": o.IsCaptive,
			"exp_need": o.Level * 200,
			// ★ 2026-09-29 上限分档：名将 350 / 普通军官 150，前端按此显示「满级」
			"max_level": h.officerMaxLevelOf(o),
		},
		"skills": skillViews, "all_skills": allSkills,
		// ★ 已穿戴装备补上套装名（老数据里只存了 set_id，前端不该显示「套装21」这种内部 ID）
		"equipped": h.officerEquippedView(o), "bag": bag, "bag_sets": bagSets, "gold": city.Gold,
	})
}

// officerEquippedView 已穿戴装备的下发格式（补套装名，前端直接用）
func (h *EzfyHandler) officerEquippedView(o *model.EzfyOfficer) []gin.H {
	out := []gin.H{}
	// ★ 老快照里没有 tier（上次重构前存的），回落到背包行取品质，保证详情页品质列不为空
	tierOf := map[int64]int{}
	if o != nil && o.ID > 0 {
		var rows []model.EzfyEquipment
		h.DB.Where("officer_id = ?", o.ID).Find(&rows)
		for _, r := range rows {
			tierOf[int64(r.ID)] = r.Tier
		}
	}
	for _, m := range officerEquipped(o) {
		sid := jsonInt(m["set_id"])
		slot, _ := m["slot"].(string)
		typ, _ := m["type"].(string)
		if slot == "" {
			slot = typ
		}
		id := int64(jsonInt(m["id"]))
		tier := jsonInt(m["tier"])
		if tier == 0 {
			tier = tierOf[id]
		}
		out = append(out, gin.H{
			"id": jsonInt(m["id"]), "name": m["name"], "type": typ, "slot": slot,
			"tier": tier, "tier_name": ezfyTierName(tier),
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

// ezfySetSlots 各套装的部位清单（该套装下的装备配置聚合出来的，用于「这套都包含哪些部位」）
func ezfySetSlots() map[int][]string {
	out := map[int][]string{}
	for _, id := range ezfyEquipIDsAsc() {
		e := ezfyCfg.equipments[id]
		if e.SetId <= 0 {
			continue
		}
		slot := e.EquipSlot()
		dup := false
		for _, x := range out[e.SetId] {
			if x == slot {
				dup = true
				break
			}
		}
		if !dup {
			out[e.SetId] = append(out[e.SetId], slot)
		}
	}
	return out
}

// ezfyAllSetsView 全部套装配置（玩家端「套装加成」展示用）
//
// ★ 2026-09-25 用户反馈「有些套装的加成玩家看不到、不容易看到，导致不知道买完套装给军官用哪个」：
// 原来只有 /officers/equipments 的 sets 字段，而且**只含我至少有一件的套装** ——
// 商城里看一件套装件，只知道名字和品质，根本不知道这套穿齐给什么，没法对比。
// 这里给一份**全量**的（含还没拥有的），前端按 set_id 查表即可。
//
// owned = 我拥有该套装的件数（背包 + 已穿戴都算）；slots = 该套装包含哪些部位。
// tier_name 取该套装各件的**最高品质**（套装表本身没有 tier 列）。
func (h *EzfyHandler) ezfyAllSetsView(owned map[int]int, slotsOf map[int][]string) []gin.H {
	pieceTier := map[int]int{}
	for _, e := range ezfyCfg.equipments {
		if e.SetId > 0 && e.Tier > pieceTier[e.SetId] {
			pieceTier[e.SetId] = e.Tier
		}
	}
	out := []gin.H{}
	for _, s := range ezfyCfg.equipSets() {
		slots := slotsOf[s.ID]
		if slots == nil {
			slots = []string{}
		}
		out = append(out, gin.H{
			"id": s.ID, "name": s.Name, "parts": s.Parts, "series": s.Series,
			"tier_name": ezfyTierName(pieceTier[s.ID]),
			"military":  s.Military, "logistics": s.Logistics, "learning": s.Learning,
			"dmg": s.Dmg, "def": s.Def, "hp": s.Hp,
			"move": s.Move, "crit": s.Crit, "crit_dmg": s.CritDmg,
			"effect": s.Effect, "des": s.Des,
			"slots": slots, "owned": owned[s.ID],
		})
	}
	return out
}

// EquipSets GET /games/ezfy/equipsets —— 全部套装配置（含加成、部位、我拥有几件）
//
// ★ 2026-09-25：装备页 / 商城页 / 军官装备页都要「set_id → 这套穿齐给什么」的映射。
// 原来只有 /officers/equipments 里的 sets（且只含我有的），商城里根本没有加成数据。
// 这里单独给一份全量的，前端拉一次缓存住、各页共用（体积很小）。
func (h *EzfyHandler) EquipSets(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	// 我拥有各套装几件（背包 + 已穿戴都算，与 /officers/equipments 的统计口径一致）
	owned := map[int]int{}
	for _, e := range h.equipmentList(uid) {
		if e.SetId > 0 {
			owned[e.SetId]++
		}
	}
	resp.OK(c, gin.H{"sets": h.ezfyAllSetsView(owned, ezfySetSlots())})
}

// AcadeRecruit GET /games/ezfy/acade/recruit —— 军校候选名将
func (h *EzfyHandler) AcadeRecruit(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	// ★ 2026-10-04 性能：3s 玩家级缓存（刷新/雇佣操作已统一 Del）
	if it, ok := ezfyPageCacheGet(uid, "recruit"); ok {
		resp.OK(c, it)
		return
	}
	// ★★ 2026-10-05 性能（用户反馈「/acade/recruit 3s」）：ezfyPageSettle 一次并行取数
	//   （建筑/科技/野地/部队/训练队列/科技行/增产令/市长）+ 快照懒结算，全部零额外读；
	//   建筑等级直接复用快照，不再单独 buildingList 一次。
	city, snap := h.ezfyPageSettle(uid)
	blv := snap.buildingLevelsOf(h, city.ID)
	academy := blv[ezfyBuildingAcademy]
	out := gin.H{
		"academy_level": academy, "staff_level": blv[ezfyBuildingStaff],
		// ★ 2026-10-06 这里是**军校招募页**，用的是「参谋部容量（在职军官位）= 参谋部等级」，
		//   不是战俘营容量（战俘营容量 = ×4，只数俘虏，见 Officers 的 captive_capacity）。
		"capacity": blv[ezfyBuildingStaff],
		// ★ 2026-10-05 性能：复用懒结算已查到的军官列表统计在职人数（原 officerCount 内部又查一次军官表）
		"used": officerCountOf(snap.officersOf(h, city.ID)),
		"gold": city.Gold, "candidates": []gin.H{},
	}
	if academy >= 1 {
		drafts, left, limit := h.recruitInfo(uid, academy, snap.profile)
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
	ezfyPageCacheSet(uid, "recruit", out)
	resp.OK(c, out)
}

// AcadeRefresh POST /games/ezfy/acade/recruit/refresh
func (h *EzfyHandler) AcadeRefresh(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	ezfyPageCacheDel(uid)
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
	ezfyPageCacheDel(uid)
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
	ezfyPageCacheDel(uid)
	city := h.getOrCreateCity(uid)
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	h.done(c, h.grantOfficer(&city, id), "赏赐成功, 忠诚已提升")
}

// OfficerTreasureGrant POST /games/ezfy/officers/:id/treasure-grant  {equip_id}
// 赏赐宝物：消耗 1 件背包未穿戴的宝物，按品质加忠诚（最高 +50）
func (h *EzfyHandler) OfficerTreasureGrant(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	ezfyPageCacheDel(uid)
	city := h.getOrCreateCity(uid)
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var in struct {
		EquipId uint `json:"equip_id"`
	}
	c.ShouldBindJSON(&in)
	h.done(c, h.treasureGrantOfficer(&city, id, in.EquipId), "赏赐成功, 忠诚已提升")
}

// OfficerAttr POST /games/ezfy/officers/:id/attr  {attr: military|logistics|learning, count}
//
// ★ 2026-09-22 每升 1 级得 1 点可用属性点，玩家自己分配；
// 只写玩家自己的军官实例，**绝不回写军官池**。
func (h *EzfyHandler) OfficerAttr(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	ezfyPageCacheDel(uid)
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
			"star_points":    now.StarPoints,
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
	ezfyPageCacheDel(uid)
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

// officerStarUp 执行一次升星判定（**不扣升星卡**，扣卡由调用方决定）
//
// ★ 2026-09-23 「军官升星做得太复杂，优化简约点」→ 简化后规则：
//   - 成功率 = 管理端配置的固定值（officer_star_chance，默认 20%），不再有按星级递减/下限
//   - 失败星级不变、消耗 1 枚星级徽章（调用方统一扣）
//   - 每升 1 星三维各 +`officer_star_attr_gain`，**只加当前属性、不动 base_***
//     ★ 用户规则（2026-09-26）：「升星也是 现属性 − 原池子军官属性」——
//     升星加成属于差额的一部分，洗点时和玩家手动加的点一起退回成待分配点数。
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
	rate := ezfyStarSuccessRate()
	if rand.Intn(100) >= rate {
		return fmt.Sprintf("升星失败(成功率%d%%，星级不变)", rate), false
	}
	// ★ 2026-09-26 「每星三维加成 随机 1-配置的属性」：
	//   三维**各自**随机 +[1, 配置值]（配置 = 管理端 `officer_star_attr_gain`，默认 10），
	//   而不是固定加配置值 —— 这样每颗星涨多少有差异，不再千篇一律。
	//   并且夹到「升星后那个星级」的属性上限（普通军官逐项不得超同星级名将；
	//   名将实例 general_id>0 是基准，不夹）。
	gainMax := ezfyStarAttrGain()
	if gainMax < 1 {
		gainMax = 1
	}
	newStar := o.Star + 1
	newMil := o.Military + 1 + rand.Intn(gainMax)
	newLog := o.Logistics + 1 + rand.Intn(gainMax)
	newLea := o.Learning + 1 + rand.Intn(gainMax)
	if o.GeneralId == 0 {
		if c, ok := ezfyGeneralCapByStarDB(h.DB)[newStar]; ok {
			if newMil > c[0] {
				newMil = c[0]
			}
			if newLog > c[1] {
				newLog = c[1]
			}
			if newLea > c[2] {
				newLea = c[2]
			}
		}
	}
	dMil, dLog, dLea := newMil-o.Military, newLog-o.Logistics, newLea-o.Learning
	// ★ 用户规则（2026-09-26）：「升星也是 现属性 − 原池子军官属性」——
	//   升星加成只加**当前属性**，**不动 base_***（原始属性恒等于军官池武将属性）。
	//   这样「现代属性 − 原始属性」的差额里自然包含了升星加成，
	//   洗点时会和玩家手动加的点一起退回成待分配点数。
	h.DB.Model(&model.EzfyOfficer{}).Where("id = ?", o.ID).Updates(map[string]interface{}{
		"star":     newStar,
		"military": newMil, "logistics": newLog, "learning": newLea,
		// ★ 2026-09-30 升星累计加点（三维之和累加，洗点前玩家能看懂多少点是升星来的）
		"star_points": o.StarPoints + dMil + dLog + dLea,
		"update_time": time.Now(),
	})
	h.addReport(city.UserID, 6, "军官升星: "+o.Name,
		fmt.Sprintf("%s 升星成功: %d星→%d星, 军事+%d 后勤+%d 学识+%d。", o.Name, o.Star, newStar, dMil, dLog, dLea), "")
	return fmt.Sprintf("升星成功: %s %d星→%d星, 军事+%d 后勤+%d 学识+%d", o.Name, o.Star, newStar, dMil, dLog, dLea), true
}

// OfficerStarUp POST /games/ezfy/officers/:id/starup
//
// 在军官详情页直接点「升星」：消耗背包里的 1 张「军官升星卡」。
// 失败是否退卡由 `officer_star_keep_on_fail` 控制。
func (h *EzfyHandler) OfficerStarUp(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	ezfyPageCacheDel(uid)
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
		h.fail(c, "没有「星级徽章」，可在商城购买或开宝箱获得")
		return
	}
	// ★ 简化后：无论成功失败都消耗 1 枚徽章；已达上限则提前拦住，不白白扣卡
	if o.Star >= ezfyStarMax() {
		h.fail(c, fmt.Sprintf("星级已达上限(%d星)", ezfyStarMax()))
		return
	}
	msg, ok := h.officerStarUp(&city, id)
	h.consumeItem(uid, ezfyStarItemID, "军官升星")
	if !ok {
		h.fail(c, msg)
		return
	}
	h.done(c, "", msg)
}

// OfficerRename POST /games/ezfy/officers/:id/rename  {name}
//
// ★ 2026-09-23 「玩家自己的军官也能改名」：消耗 1 张「军官改名卡」。
//
//	只改玩家自己的军官实例（ezfy_officer.name），绝不回写军官池（ezfy_cfg_general）。
func (h *EzfyHandler) OfficerRename(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	ezfyPageCacheDel(uid)
	city := h.getOrCreateCity(uid)
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	o := h.officerOf(city.ID, id)
	if o == nil {
		resp.NotFound(c, "军官不存在")
		return
	}
	if o.IsCaptive == 1 {
		resp.ParamError(c, "俘虏不能改名, 请先在军校收编")
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		resp.ParamError(c, "请填写新的军官名")
		return
	}
	if len([]rune(name)) < 2 || len([]rune(name)) > 12 {
		resp.ParamError(c, "军官名长度需在 2~12 个字符之间")
		return
	}
	if name == o.Name {
		resp.ParamError(c, "新名字与当前名字相同")
		return
	}
	if h.itemCount(uid, ezfyOfficerRenameCardItemID) <= 0 {
		resp.ParamError(c, "没有「军官改名卡」，可在商城购买")
		return
	}
	h.consumeItem(uid, ezfyOfficerRenameCardItemID, "军官改名")
	h.DB.Model(&model.EzfyOfficer{}).Where("id = ?", o.ID).Updates(map[string]interface{}{
		"name": name, "update_time": time.Now(),
	})
	resp.OK(c, gin.H{"msg": "改名成功：「" + name + "」（消耗军官改名卡 ×1）", "name": name})
}

// ============ 宝箱（钻石/黄金购买，开箱按权重出套装件） ============
//
// ★ 2026-09-22 「有的套装是开宝箱概率得到的，看看怎么引入宝箱，宝箱一般用钻石买。」
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

// ezfyEquipAttrText 装备六项战斗属性说明（伤害/防御/生命/移动/暴击几率/暴击伤害 + 老三维军事/后勤/学识）
func ezfyEquipAttrText(e *model.EzfyCfgEquipment) string {
	var parts []string
	if e.Dmg != 0 {
		parts = append(parts, "伤害+"+strconv.Itoa(e.Dmg)+"%")
	}
	if e.Def != 0 {
		parts = append(parts, "防御+"+strconv.Itoa(e.Def)+"%")
	}
	if e.Hp != 0 {
		parts = append(parts, "生命+"+strconv.Itoa(e.Hp)+"%")
	}
	if e.Move != 0 {
		parts = append(parts, "移动距离+"+strconv.Itoa(e.Move)+"%")
	}
	if e.Crit != 0 {
		parts = append(parts, "暴击几率+"+strconv.Itoa(e.Crit)+"%")
	}
	if e.CritDmg != 0 {
		parts = append(parts, "暴击伤害+"+strconv.Itoa(e.CritDmg)+"%")
	}
	if e.Military != 0 {
		parts = append(parts, "军事+"+strconv.Itoa(e.Military))
	}
	if e.Logistics != 0 {
		parts = append(parts, "后勤+"+strconv.Itoa(e.Logistics))
	}
	if e.Learning != 0 {
		parts = append(parts, "学识+"+strconv.Itoa(e.Learning))
	}
	return strings.Join(parts, " ")
}

// ezfyChestPrizeDetail 生成宝箱奖品详情说明（供玩家点奖品名查看具体内容）
func (h *EzfyHandler) ezfyChestPrizeDetail(p *model.EzfyCfgChestItem) string {
	switch p.Kind {
	case 1: // 装备
		cfg := ezfyCfg.equipment(p.RefId)
		if cfg == nil {
			return ""
		}
		var parts []string
		if cfg.EquipSlot() != "" {
			parts = append(parts, "部位:"+cfg.EquipSlot())
		}
		if cfg.Tier > 0 {
			parts = append(parts, "品质:"+ezfyTierName(cfg.Tier))
		}
		if cfg.Level > 0 {
			parts = append(parts, "等级"+strconv.Itoa(cfg.Level))
		}
		if a := ezfyEquipAttrText(cfg); a != "" {
			parts = append(parts, a)
		}
		if cfg.Effect != "" {
			parts = append(parts, "额外效果:"+cfg.Effect)
		}
		if cfg.Des != "" {
			parts = append(parts, cfg.Des)
		}
		return strings.Join(parts, " · ")
	case 2: // 道具
		cfg := ezfyCfg.item(p.RefId)
		if cfg == nil {
			return ""
		}
		if cfg.Description != "" {
			return cfg.Description
		}
		return ""
	case 3: // 整套（RefId = 套装 id）
		s := ezfyCfg.equipSet(p.RefId)
		if s == nil {
			return ""
		}
		var parts []string
		if s.Effect != "" {
			parts = append(parts, "套装效果:"+s.Effect)
		}
		if s.Des != "" {
			parts = append(parts, s.Des)
		}
		ids := ezfyEquipIDsOfSet(p.RefId)
		if len(ids) > 0 {
			var names []string
			for _, id := range ids {
				if e := ezfyCfg.equipments[id]; e.Name != "" {
					names = append(names, e.Name)
				}
			}
			parts = append(parts, "包含"+strconv.Itoa(len(names))+"件："+strings.Join(names, "、"))
		}
		return strings.Join(parts, " · ")
	default:
		return ""
	}
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
		// ★ （2026-09-22）：「套装宝箱……开出来还是按套来吧」
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
	// ★ 2026-10-04 性能（用户反馈「/chest 线上 3s」）：展示页加 3s TTL 玩家级缓存
	//   （开箱后失效），命中零 SQL；未命中走配置缓存 —— chests/pools 已并入 ezfyCfg，
	//   原「查宝箱表 + 逐箱查奖池」的 N+1 全消。
	if it, ok := ezfyChestCacheGet(uid); ok {
		resp.OK(c, it)
		return
	}
	// ★ 2026-10-05 性能：ezfyPageSettle（并行取数 + 快照懒结算）
	city, _ := h.ezfyPageSettle(uid)
	out := []gin.H{}
	for _, c2 := range ezfyCfg.chests {
		pool := []gin.H{}
		for _, p := range ezfyCfg.chestPool(c2.ID) {
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
				"detail": h.ezfyChestPrizeDetail(&p),
			})
		}
		out = append(out, gin.H{
			"id": c2.ID, "name": c2.Name,
			"price_gold": c2.PriceGold, "price_diamond": c2.PriceDiamond,
			"stock": c2.Stock, "sold_out": c2.Stock == 0, "open_max": c2.OpenMax,
			"des": c2.Des, "effect": c2.Effect, "pool": pool,
		})
	}
	data := gin.H{"chests": out, "gold": city.Gold, "diamond": h.ensureProfile(uid).Diamond}
	ezfyChestCacheSet(uid, data)
	resp.OK(c, data)
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
		// ★ 2026-09-28 钻石流水
		h.logDiamond(uid, -total, "购买宝箱: "+chest.Name)
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
	// ★ 2026-10-04 库存/黄金/钻石都变了 → 失效宝箱缓存
	ezfyChestCacheDel(uid)
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
	ezfyPageCacheDel(uid)
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
	ezfyPageCacheDel(uid)
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

// OfficerUnequipAll POST /games/ezfy/officers/:id/unequip-all —— 一键卸下该军官全部装备
func (h *EzfyHandler) OfficerUnequipAll(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	ezfyPageCacheDel(uid)
	city := h.getOrCreateCity(uid)
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	o := h.officerOf(city.ID, id)
	if o == nil {
		resp.ParamError(c, "武将不存在")
		return
	}
	equipped := officerEquipped(o)
	if len(equipped) == 0 {
		h.done(c, "", "该军官未穿戴装备")
		return
	}
	ids := make([]int64, 0, len(equipped))
	for _, m := range equipped {
		if i := jsonInt(m["id"]); i > 0 {
			ids = append(ids, int64(i))
		}
	}
	// ★ 顺序与 equipItem 一致：先把军官身上的装备列表写空，再把装备行 officer_id 置 0
	if msg := h.saveOfficerEquipment(o, []map[string]interface{}{}); msg != "" {
		resp.ParamError(c, msg)
		return
	}
	h.DB.Model(&model.EzfyEquipment{}).Where("id IN ?", ids).Update("officer_id", 0)
	h.done(c, "", fmt.Sprintf("已卸下 %d 件装备到背包", len(ids)))
}

// OfficerEquipSet POST /games/ezfy/officers/:id/equip-set  {set_id} —— 一键穿整套
//
// ★ 把背包里该套装所有件穿上；目标部位被其他套装的件占用时先卸下让位（珠宝可叠不受限）。
func (h *EzfyHandler) OfficerEquipSet(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	ezfyPageCacheDel(uid)
	city := h.getOrCreateCity(uid)
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		SetId int `json:"set_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	o := h.officerOf(city.ID, id)
	if o == nil {
		resp.ParamError(c, "武将不存在")
		return
	}
	worn, swapped, skipped, emsg := h.equipSet(&city, o, req.SetId)
	if emsg != "" {
		resp.ParamError(c, emsg)
		return
	}
	if worn == 0 {
		if skipped > 0 {
			resp.ParamError(c, "背包里该套装装备的佩戴等级都不够")
		} else {
			resp.ParamError(c, "背包里没有该套装的装备")
		}
		return
	}
	txt := fmt.Sprintf("已一键穿上整套 %d 件", worn)
	if swapped > 0 {
		txt += fmt.Sprintf("，另卸下 %d 件同部位装备让位", swapped)
	}
	if skipped > 0 {
		txt += fmt.Sprintf("，%d 件等级不足已跳过", skipped)
	}
	h.done(c, "", txt)
}

// equipSet 一键穿套装的内部实现：返回 (穿上件数, 让位卸下件数, 等级不足跳过件数, 错误信息)
func (h *EzfyHandler) equipSet(city *model.EzfyCity, o *model.EzfyOfficer, setId int) (int, int, int, string) {
	h.calcResource(city)
	s := ezfyCfg.equipSet(setId)
	if s == nil {
		return 0, 0, 0, "该套装不存在"
	}
	var items []model.EzfyEquipment
	h.DB.Where("user_id = ? AND set_id = ? AND officer_id = 0", city.UserID, setId).Order("id ASC").Find(&items)
	if len(items) == 0 {
		return 0, 0, 0, ""
	}
	equipped := officerEquipped(o)
	// 已穿戴集合：slot -> id，让位与同件判定用
	bySlot := map[string]int64{}
	for _, m := range equipped {
		slot, _ := m["slot"].(string)
		if slot == "" {
			slot, _ = m["type"].(string)
		}
		bySlot[model.EzfySlotCanon(slot)] = int64(jsonInt(m["id"]))
	}
	worn, swapped, skipped := 0, 0, 0
	wornIds := []int64{}
	dismountIds := []int64{}
	for _, e := range items {
		if o.Level < e.Level {
			skipped++
			continue
		}
		slot := model.EzfySlotCanon(e.EquipSlot())
		// ★ 同部位让位：先卸下占位的其他装备（2026-09-24 起珠宝同样只留一件，不叠穿）
		if oldId, ok := bySlot[slot]; ok && oldId > 0 {
			kept := []map[string]interface{}{}
			for _, m := range equipped {
				if int64(jsonInt(m["id"])) == oldId {
					continue
				}
				kept = append(kept, m)
			}
			equipped = kept
			dismountIds = append(dismountIds, oldId)
			swapped++
		}
		equipped = append(equipped, map[string]interface{}{
			"id": e.ID, "name": e.Name, "type": e.Type, "slot": slot, "set_id": e.SetId,
			"tier":     e.Tier,
			"military": e.Military, "logistics": e.Logistics, "learning": e.Learning,
			"series": e.Series, "enhance": e.Enhance,
			"dmg": e.Dmg, "def": e.Def, "hp": e.Hp, "move": e.Move, "crit": e.Crit, "crit_dmg": e.CritDmg,
		})
		bySlot[slot] = int64(e.ID)
		wornIds = append(wornIds, int64(e.ID))
		worn++
	}
	if worn == 0 {
		return 0, 0, skipped, ""
	}
	// ★ 顺序同 equipItem：先写军官身上装备 JSON 成功，再改装备行的 officer_id
	if msg := h.saveOfficerEquipment(o, equipped); msg != "" {
		return 0, 0, 0, msg
	}
	if len(wornIds) > 0 {
		h.DB.Model(&model.EzfyEquipment{}).Where("id IN ?", wornIds).Update("officer_id", int64(o.ID))
	}
	if len(dismountIds) > 0 {
		h.DB.Model(&model.EzfyEquipment{}).Where("id IN ?", dismountIds).Update("officer_id", 0)
	}
	return worn, swapped, skipped, ""
}

// OfficerPosition POST /games/ezfy/officers/:id/position  {position}
func (h *EzfyHandler) OfficerPosition(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	ezfyPageCacheDel(uid)
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
//
// ★ 2026-10-06：战俘营是跨城汇总列表，所以这里**不能只按当前城找军官**，
// 一律按 uid 找（见 officerOfMine）——修掉「俘虏在别的城时释放/收编报武将不存在」。
func (h *EzfyHandler) OfficerCaptive(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	ezfyPageCacheDel(uid)
	h.getOrCreateCity(uid)
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		Op string `json:"op"`
	}
	_ = c.ShouldBindJSON(&req)
	if req.Op == "recruit" {
		h.done(c, h.recruitCaptive(uid, id), "收编成功, 军官已入列")
		return
	}
	h.done(c, h.freeOfficer(uid, id), "已释放该武将")
}

// OfficerExile POST /games/ezfy/officers/:id/exile
func (h *EzfyHandler) OfficerExile(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	ezfyPageCacheDel(uid)
	city := h.getOrCreateCity(uid)
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	h.done(c, h.exileOfficer(&city, id), "已流放该军官")
}

// OfficerSkills GET /games/ezfy/officers/skills —— 技能总览 + 我的军官
func (h *EzfyHandler) OfficerSkills(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	// ★ 2026-10-04 性能：3s 玩家级缓存（学习/遗忘技能已统一 Del）
	if it, ok := ezfyPageCacheGet(uid, "skills"); ok {
		resp.OK(c, it)
		return
	}
	// ★★ 2026-10-05 性能（用户反馈「/officers/skills 2s」）：ezfyPageSettle 一次并行取数 + 快照懒结算；
	//   军官列表直接复用懒结算已经查过的那份（exp/level 已同步为结算后），不再查第二次军官表。
	city, snap := h.ezfyPageSettle(uid)
	skills := []gin.H{}
	for _, s := range ezfyCfg.skills {
		skills = append(skills, gin.H{"id": s.ID, "name": s.Name, "effect": s.Effect, "type": s.Type, "des": s.Des})
	}
	sort.Slice(skills, func(i, j int) bool { return skills[i]["id"].(int) < skills[j]["id"].(int) })
	list := []gin.H{}
	for _, o := range snap.officersOf(h, city.ID) {
		skills := officerSkills(&o)
		list = append(list, gin.H{"id": o.ID, "name": o.Name, "level": o.Level,
			"skills": skills, "skill_count": len(skills)})
	}
	data := gin.H{"skills": skills, "officers": list, "gold": city.Gold}
	ezfyPageCacheSet(uid, "skills", data)
	resp.OK(c, data)
}

// OfficerEquipments GET /games/ezfy/officers/equipments —— 装备图鉴 + 我的背包
//
// ★ 2026-10-04 性能优化：原实现逐件装备查 `equipIsCaptiveWorn`（2 次库）+
//
//	`officerOf`（1 次库），背包几百件装备就是上千次 SQL 往返（双机共 RDS 时更明显，
//	装备页因此卡到 1s+）。现在装备只查一次、当前城军官只查一次，被俘穿戴判定/穿戴者
//	名字全部走内存 map —— 总 SQL 从 O(3N) 降到常数。
//
// ★ 2026-10-04 再优化（用户反馈「/equipments 3s+」）：miss 路径改两波并行，
//
//	懒结算复用已取数据，总串行 RTT 从 ~10 降到 ~2。
func (h *EzfyHandler) OfficerEquipments(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	// ★ 2026-10-04 性能：3s 玩家级缓存（穿/脱/套装操作已统一 Del）
	if it, ok := ezfyPageCacheGet(uid, "equips"); ok {
		resp.OK(c, it)
		return
	}
	_, city, cities := h.ezfyPageCity(uid)
	var (
		items        []model.EzfyEquipment
		cityOfficers []model.EzfyOfficer
		buildings    []model.EzfyCityBuilding
		trainQueues  []model.EzfyTrainQueue
	)
	var wg sync.WaitGroup
	wg.Add(4)
	go func() { defer wg.Done(); items = h.equipmentList(uid) }()
	// 当前城军官一次拉全：①被俘军官身上挂的装备不进背包 ②已穿戴装备显示穿戴者名字
	go func() { defer wg.Done(); cityOfficers = h.officerList(city.ID) }()
	go func() { defer wg.Done(); buildings = h.buildingList(city.ID) }()
	go func() {
		defer wg.Done()
		h.DB.Where("city_id = ? AND status = 0", city.ID).Order("start_time ASC").Find(&trainQueues)
	}()
	wg.Wait()
	// 懒结算复用已取数据（零额外查询）
	h.checkBuildingDone(&city, buildings)
	h.checkTechDone(&city, cityIdsOf(cities))
	h.collectTrainQueue(&city, trainQueues)
	h.calcResource(&city, cityOfficers)
	ofByName := map[int64]*model.EzfyOfficer{}
	captive := map[int64]bool{}
	for i := range cityOfficers {
		o := &cityOfficers[i]
		ofByName[int64(o.ID)] = o
		if o.IsCaptive == 1 {
			captive[int64(o.ID)] = true
		}
	}
	// ★ 语义与旧 equipIsCaptiveWorn 完全一致：装备挂在本城某名「未收编俘虏」身上才跳过
	isCaptiveWorn := func(e *model.EzfyEquipment) bool {
		return e.OfficerId > 0 && captive[e.OfficerId]
	}
	bag := []gin.H{}
	// ★ 2026-09-28 赏赐宝物只认「采集宝物」（与军官详情接口口径一致）
	treasureSet := ezfyCollectibleTreasureNames()
	for i := range items {
		e := &items[i]
		// ★ 2026-09-29：挂在「未收编俘虏」身上的装备不进入背包列表——只有收编后才归属本玩家
		if isCaptiveWorn(e) {
			continue
		}
		wornBy := ""
		if e.OfficerId > 0 {
			if o := ofByName[e.OfficerId]; o != nil {
				wornBy = o.Name
			}
		}
		bag = append(bag, gin.H{
			"id": e.ID, "name": e.Name, "type": e.Type, "tier": e.Tier,
			"tier_name": ezfyTierName(e.Tier),
			"military":  e.Military, "logistics": e.Logistics, "learning": e.Learning,
			"level": e.Level, "officer_id": e.OfficerId, "worn": e.OfficerId > 0, "worn_by": wornBy,
			"treasure": treasureSet[e.Name],
			"slot":     e.EquipSlot(), "set_id": e.SetId, "set_name": h.ezfySetName(e.SetId),
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
	//   ★ 2026-10-04 直接复用上面已查的 items，不再二次全表查库。
	owned := map[int]int{}
	for i := range items {
		if items[i].SetId > 0 {
			owned[items[i].SetId]++
		}
	}
	// ★ 套装品质：取该套装各件的最高 Tier（系列 21~26 为 Tier 3/4，第一批套装 Tier 1~4）
	pieceTier := map[int]int{}
	for _, e := range ezfyCfg.equipments {
		if e.SetId > 0 && e.Tier > pieceTier[e.SetId] {
			pieceTier[e.SetId] = e.Tier
		}
	}
	setList := []gin.H{}
	for _, s := range ezfyCfg.equipSets() {
		have := owned[s.ID]
		if have == 0 {
			continue // 一件都没有的套装不展示（图鉴在下面「装备图鉴」里）
		}
		setList = append(setList, gin.H{
			"id": s.ID, "name": s.Name, "parts": s.Parts, "series": s.Series, "tier_name": ezfyTierName(pieceTier[s.ID]),
			"have": have, "complete": have >= s.Parts,
			"military": s.Military, "logistics": s.Logistics, "learning": s.Learning,
			"dmg": s.Dmg, "def": s.Def, "hp": s.Hp, "move": s.Move, "crit": s.Crit, "crit_dmg": s.CritDmg,
			"effect": s.Effect, "des": s.Des,
		})
	}
	data := gin.H{"bag": bag, "all": cfgList, "sets": setList}
	ezfyPageCacheSet(uid, "equips", data)
	resp.OK(c, data)
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
// ★ 2026-09-22 军官穿的装备有套装，玩家自己用黄金或钻石买。
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
// ★ （2026-09-22）：
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
//
// ★ 2026-10-04 性能（用户反馈「/equipshop 线上 2s+」）：展示页 + 3s 玩家级缓存
//
//	（买装备在 EquipShopBuy 已统一失效），缓存命中零 SQL。
func (h *EzfyHandler) EquipShop(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	if it, ok := ezfyPageCacheGet(uid, "equipshop"); ok {
		resp.OK(c, it)
		return
	}
	// ★ 2026-10-05 性能：ezfyPageSettle（并行取数 + 快照懒结算）
	city, _ := h.ezfyPageSettle(uid)
	slots, items := h.equipShopList()
	data := gin.H{
		"slots": slots, "items": items,
		"gold": city.Gold, "diamond": h.ensureProfile(uid).Diamond,
	}
	ezfyPageCacheSet(uid, "equipshop", data)
	resp.OK(c, data)
}

// EquipShopBuy POST /games/ezfy/equipshop/buy  {cfg_id, count, currency: gold|diamond}
//
// 用黄金或钻石买装备（套装件），买入直接进玩家背包，可就地穿到军官身上。
func (h *EzfyHandler) EquipShopBuy(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	ezfyPageCacheDel(uid) // 买装备 → 背包变化，装备页缓存失效
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
		// ★ 2026-09-28 钻石流水
		h.logDiamond(uid, -total, "商城购买装备")
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
