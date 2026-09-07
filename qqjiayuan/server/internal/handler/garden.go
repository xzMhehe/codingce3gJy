package handler

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

type GardenHandler struct{ DB *gorm.DB }

// ============ 种子与图鉴数据（数据库驱动，空表回退内置默认；管理端可维护） ============

type seedDef struct {
	ID     uint
	Name   string
	DType  int    // 0普通可购买 1特殊(魔法屋合成)
	Level  int    // 购买所需花园等级
	Price  int    // 金币价格
	Seed   int    // 种子期 分钟
	Ling   int    // 花苗期 分钟
	Buds   int    // 花蕾期 分钟
	Less   int    // 最低产量
	More   int    // 最高产量
	Remark string // 鲜花花语
}

type mapDef struct {
	SeedName string
	Name     string
	DType    int
}

type mixDef struct {
	SeedName string
	Flower   string
	Need     int
}

// 内置默认（seed 幂等写入 garden_seeds / garden_maps / garden_mixes，表空时兜底用）
var defaultSeeds = []seedDef{
	{Name: "向日葵", DType: 0, Level: 1, Price: 5, Seed: 1, Ling: 1, Buds: 1, Less: 2, More: 4, Remark: "阳光,明亮,爱慕"},
	{Name: "玫瑰花", DType: 0, Level: 1, Price: 10, Seed: 1, Ling: 1, Buds: 2, Less: 3, More: 5, Remark: "爱情,美丽,热情"},
	{Name: "郁金香", DType: 0, Level: 2, Price: 20, Seed: 2, Ling: 2, Buds: 2, Less: 3, More: 6, Remark: "爱的表白,荣誉,祝福"},
	{Name: "月光花", DType: 0, Level: 3, Price: 40, Seed: 2, Ling: 2, Buds: 3, Less: 4, More: 8, Remark: "梦幻,纯洁,相思"},
	{Name: "百合花", DType: 0, Level: 4, Price: 80, Seed: 3, Ling: 3, Buds: 3, Less: 5, More: 10, Remark: "纯洁,庄严,心心相印"},
	{Name: "牡丹", DType: 0, Level: 5, Price: 150, Seed: 3, Ling: 4, Buds: 4, Less: 6, More: 12, Remark: "圆满,浓情,富贵"},
	{Name: "蓝色妖姬", DType: 0, Level: 7, Price: 300, Seed: 4, Ling: 5, Buds: 5, Less: 8, More: 16, Remark: "清纯的爱,敦厚善良"},
	{Name: "樱花", DType: 0, Level: 9, Price: 500, Seed: 5, Ling: 6, Buds: 6, Less: 10, More: 20, Remark: "生命,幸福,热烈"},
	{Name: "天山雪莲", DType: 0, Level: 12, Price: 800, Seed: 6, Ling: 8, Buds: 8, Less: 12, More: 24, Remark: "纯白的爱,坚韧,圣洁"},
	// 魔法屋合成产物（特殊种子）
	{Name: "银色菊花", DType: 1, Level: 3, Price: 0, Seed: 2, Ling: 2, Buds: 2, Less: 4, More: 8, Remark: "银色思念,静谧"},
	{Name: "银野花", DType: 1, Level: 4, Price: 0, Seed: 2, Ling: 3, Buds: 3, Less: 5, More: 10, Remark: "银色梦想,自由"},
	{Name: "端阳花", DType: 1, Level: 5, Price: 0, Seed: 3, Ling: 3, Buds: 3, Less: 6, More: 12, Remark: "端午安康,平安"},
	{Name: "银友谊花", DType: 1, Level: 6, Price: 0, Seed: 3, Ling: 4, Buds: 4, Less: 7, More: 14, Remark: "银色友谊,长存"},
	{Name: "银色烈焰焚情", DType: 1, Level: 8, Price: 0, Seed: 4, Ling: 5, Buds: 5, Less: 8, More: 16, Remark: "银色烈焰,炙热"},
	{Name: "金色烈焰焚情", DType: 1, Level: 10, Price: 0, Seed: 5, Ling: 6, Buds: 6, Less: 10, More: 20, Remark: "金色烈焰,永恒"},
}

var defaultMaps = []mapDef{
	{"向日葵", "金色向日葵", 0},
	{"向日葵", "七彩向日葵", 1},
	{"向日葵", "太阳神花", 2},
	{"玫瑰花", "红玫瑰", 0},
	{"玫瑰花", "蓝玫瑰", 1},
	{"玫瑰花", "黑玫瑰", 2},
	{"郁金香", "黄郁金香", 0},
	{"郁金香", "粉郁金香", 1},
	{"郁金香", "黑郁金香", 2},
	{"月光花", "月光花", 0},
	{"月光花", "星月花", 1},
	{"月光花", "幻月花", 2},
	{"百合花", "白百合", 0},
	{"百合花", "金百合", 1},
	{"百合花", "火百合", 2},
	{"牡丹", "粉牡丹", 0},
	{"牡丹", "绿牡丹", 1},
	{"牡丹", "黑牡丹", 2},
	{"蓝色妖姬", "蓝色妖姬", 0},
	{"蓝色妖姬", "冰蓝妖姬", 1},
	{"蓝色妖姬", "魅蓝妖姬", 2},
	{"樱花", "粉樱花", 0},
	{"樱花", "垂枝樱", 1},
	{"樱花", "夜樱", 2},
	{"天山雪莲", "雪莲花", 0},
	{"天山雪莲", "金雪莲", 1},
	{"天山雪莲", "七彩雪莲", 2},
	{"银色菊花", "银色菊花", 0},
	{"银野花", "银野花", 0},
	{"端阳花", "端阳花", 0},
	{"银友谊花", "银友谊花", 0},
	{"银色烈焰焚情", "银色烈焰焚情", 1},
	{"金色烈焰焚情", "金色烈焰焚情", 2},
}

var defaultMixes = []mixDef{
	{"银色菊花", "向日葵", 5},
	{"银野花", "向日葵", 3},
	{"银野花", "玫瑰花", 2},
	{"端阳花", "玫瑰花", 4},
	{"端阳花", "郁金香", 2},
	{"银友谊花", "向日葵", 2},
	{"银友谊花", "玫瑰花", 2},
	{"银友谊花", "郁金香", 2},
	{"银色烈焰焚情", "月光花", 3},
	{"金色烈焰焚情", "月光花", 6},
}

// 从数据库加载种子（表空/无数据时回退内置默认）
func (h *GardenHandler) loadSeeds() []model.GardenSeed {
	var rows []model.GardenSeed
	h.DB.Order("id ASC").Find(&rows)
	if len(rows) == 0 {
		for i, d := range defaultSeeds {
			rows = append(rows, model.GardenSeed{ID: uint(i + 1), Name: d.Name, DType: d.DType, Level: d.Level, Price: d.Price, Seed: d.Seed, Ling: d.Ling, Buds: d.Buds, Less: d.Less, More: d.More, Remark: d.Remark, Status: 1})
		}
	}
	return rows
}

func (h *GardenHandler) loadMaps() []model.GardenMap {
	var rows []model.GardenMap
	h.DB.Order("id ASC").Find(&rows)
	if len(rows) == 0 {
		for i, d := range defaultMaps {
			rows = append(rows, model.GardenMap{ID: uint(i + 1), SeedID: 0, Name: d.Name, DType: d.DType})
		}
	}
	return rows
}

func (h *GardenHandler) loadMixes() []model.GardenMix {
	var rows []model.GardenMix
	h.DB.Order("id ASC").Find(&rows)
	if len(rows) == 0 {
		for i, d := range defaultMixes {
			rows = append(rows, model.GardenMix{ID: uint(i + 1), SeedID: 0, Flower: d.Flower, Need: d.Need})
		}
	}
	return rows
}

// 种子 id → 定义（默认回退：id 即内置下标）
func (h *GardenHandler) seedByID(id uint) *seedDef {
	rows := h.loadSeeds()
	for _, s := range rows {
		if s.ID == id && s.Status == 1 {
			return &seedDef{ID: s.ID, Name: s.Name, DType: s.DType, Level: s.Level, Price: s.Price, Seed: s.Seed, Ling: s.Ling, Buds: s.Buds, Less: s.Less, More: s.More, Remark: s.Remark}
		}
	}
	if id > 0 && uint(id) <= uint(len(defaultSeeds)) {
		d := defaultSeeds[id-1]
		d.ID = id
		return &d
	}
	return nil
}

// 种子名 → 可开出的图鉴花
func (h *GardenHandler) mapsBySeedName(seedName string) []model.GardenMap {
	rows := h.loadMaps()
	// 先按 seed_id 匹配（数据库模式）
	seedID := uint(0)
	for _, s := range h.loadSeeds() {
		if s.Name == seedName {
			seedID = s.ID
			break
		}
	}
	var out []model.GardenMap
	if seedID > 0 {
		for _, m := range rows {
			if m.SeedID == seedID {
				out = append(out, m)
			}
		}
	}
	if len(out) == 0 {
		for _, m := range rows {
			if m.Name == seedName || mapSeedName(m) == seedName {
				out = append(out, m)
			}
		}
	}
	if len(out) == 0 {
		for _, d := range defaultMaps {
			if d.SeedName == seedName {
				out = append(out, model.GardenMap{Name: d.Name, DType: d.DType})
			}
		}
	}
	return out
}

// 花名 → 图鉴定义
func (h *GardenHandler) mapByName(flower string) (mapID uint, dtype int, ok bool) {
	rows := h.loadMaps()
	for _, m := range rows {
		if m.Name == flower {
			return m.ID, m.DType, true
		}
	}
	for i, d := range defaultMaps {
		if d.Name == flower {
			return uint(i + 1), d.DType, true
		}
	}
	return 0, 0, false
}

// 合成配方
func (h *GardenHandler) mixesBySeed(seedName string) []model.GardenMix {
	rows := h.loadMixes()
	var out []model.GardenMix
	seedID := uint(0)
	for _, s := range h.loadSeeds() {
		if s.Name == seedName {
			seedID = s.ID
			break
		}
	}
	for _, m := range rows {
		if m.SeedID == seedID || (seedID == 0 && mixSeedName(m) == seedName) {
			out = append(out, m)
		}
	}
	if len(out) == 0 {
		for _, d := range defaultMixes {
			if d.SeedName == seedName {
				out = append(out, model.GardenMix{Flower: d.Flower, Need: d.Need})
			}
		}
	}
	return out
}

// 图鉴所属种子名（数据库模式按 seed_id；兼容内置）
func mapSeedName(m model.GardenMap) string {
	if m.SeedID > 0 {
		for i, d := range defaultMaps {
			if uint(i+1) == m.SeedID {
				return d.SeedName
			}
		}
	}
	return m.Name
}

func mixSeedName(m model.GardenMix) string {
	for i, d := range defaultMixes {
		if uint(i+1) == m.ID {
			return d.SeedName
		}
	}
	return ""
}

// 播种、浇水、锄草、捉虫、收获的经验值
const (
	gardenExpSow   = 2
	gardenExpCare  = 1
	gardenExpMix   = 20
	gardenExpNew   = 10 // 首次点亮普通图谱
	gardenExpNewRare = 100 // 首次点亮独特/珍稀图谱
	gardenExpKnown = 1  // 已点亮图谱
)

// 添置花圃花费：当前 lands → 所需等级/金币（对齐 ASP）
func gardenLandCost(lands int) (level, money int) {
	switch lands {
	case 2:
		return 2, 50
	case 3:
		return 5, 100
	case 4:
		return 8, 200
	case 5:
		return 12, 400
	case 6:
		return 16, 800
	case 7:
		return 20, 1600
	case 8:
		return 25, 3200
	case 9:
		return 32, 6400
	default:
		return 88, 12800
	}
}

const gardenMaxLands = 10

// 花园等级称号（对齐 ASP LevelName 1-35，>35 传说魔法师）
var gardenLevelNames = []string{
	"", "见习魔法学徒", "初级魔法学徒", "中级魔法学徒", "高级魔法学徒", "特级魔法学徒",
	"一星魔导士", "二星魔导士", "三星魔导士", "四星魔导士", "五星魔导士",
	"一星大魔导士", "二星大魔导士", "三星大魔导士", "四星大魔导士", "五星大魔导士",
	"一星圣魔导士", "二星圣魔导士", "三星圣魔导士", "四星圣魔导士", "五星圣魔导士",
	"一星魔法师", "二星魔法师", "三星魔法师", "四星魔法师", "五星魔法师",
	"一星大魔法师", "二星大魔法师", "三星大魔法师", "四星大魔法师", "五星大魔法师",
	"一星圣魔法师", "二星圣魔法师", "三星圣魔法师", "四星圣魔法师", "五星圣魔法师",
}

func gardenLevelName(level int) string {
	if level >= len(gardenLevelNames) {
		return "传说魔法师"
	}
	return gardenLevelNames[level]
}

// ============ 工具 ============

// 相对时间（对齐 ASP TimeDiff：刚刚 / N分钟前 / N小时前 / N天前）
func timeAgo(t time.Time) string {
	d := time.Since(t)
	if d < time.Minute {
		return "刚刚"
	}
	if d < time.Hour {
		return fmt.Sprintf("%d分钟前", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%d小时前", int(d.Hours()))
	}
	return fmt.Sprintf("%d天前", int(d.Hours()/24))
}

// 懒创建花园（每个用户一座）
func (h *GardenHandler) ensureGarden(uid uint) model.Garden {
	var g model.Garden
	if err := h.DB.Where("user_id = ?", uid).First(&g).Error; err != nil {
		var u model.User
		h.DB.First(&u, uid)
		name := "我的花园"
		if u.Nickname != "" {
			name = u.Nickname + "的花园"
		}
		g = model.Garden{UserID: uid, Name: name, Level: 1, Point: 0, Lands: 2, Notice: "欢迎光临我的花园！", Config: 0}
		h.DB.Create(&g)
	}
	// 补齐花圃记录，并清理超出 lands 的旧记录（兼容旧版 4 块地数据）
	for i := 0; i < g.Lands; i++ {
		var c int64
		h.DB.Model(&model.GardenPlot{}).Where("user_id = ? AND plot = ?", uid, i).Count(&c)
		if c == 0 {
			h.DB.Create(&model.GardenPlot{UserID: uid, Plot: i, Status: 0})
		}
	}
	h.DB.Where("user_id = ? AND plot >= ?", uid, g.Lands).Delete(&model.GardenPlot{})
	// 兼容旧版遗留数据：seed_id=0 却标记生长中/成熟的，重置为空花圃
	h.DB.Model(&model.GardenPlot{}).Where("user_id = ? AND seed_id = 0 AND status <> 0", uid).
		Updates(map[string]interface{}{"status": 0, "name": "", "yield": 0, "amount": 0})
	return g
}

// 给花园加经验（自动升级，每级 200 经验）
func (h *GardenHandler) addGardenPoint(uid uint, point int) model.Garden {
	var g model.Garden
	h.DB.Where("user_id = ?", uid).First(&g)
	g.Point += point
	for g.Point >= g.Level*200 {
		g.Point -= g.Level * 200
		g.Level++
	}
	h.DB.Model(&model.Garden{}).Where("id = ?", g.ID).Updates(map[string]interface{}{"point": g.Point, "level": g.Level})
	return g
}

// 花园消息：uid=操作者 fid=花园主
func (h *GardenHandler) addGardenMsg(uid, fid uint, remark string) {
	if fid == 0 {
		fid = uid
	}
	h.DB.Create(&model.GardenMsg{UID: uid, FID: fid, Remark: remark})
}

// 花进花篮
func (h *GardenHandler) addFlower(uid uint, flower string, n int) {
	var uf model.UserFlower
	if err := h.DB.Where("user_id = ? AND flower = ?", uid, flower).First(&uf).Error; err != nil {
		h.DB.Create(&model.UserFlower{UserID: uid, Flower: flower, Count: n})
		return
	}
	h.DB.Model(&uf).Update("count", gorm.Expr("count + ?", n))
}

// 花进花瓶
func (h *GardenHandler) addBottle(uid uint, flower string, n int) {
	var b model.GardenBottle
	if err := h.DB.Where("user_id = ? AND flower = ?", uid, flower).First(&b).Error; err != nil {
		h.DB.Create(&model.GardenBottle{UserID: uid, Flower: flower, Count: n})
		return
	}
	h.DB.Model(&b).Update("count", gorm.Expr("count + ?", n))
}

// 判断成熟/推进生长：返回花圃当前状态信息
// status: 0空 1生长中 2成熟
type plotView struct {
	ID     uint   `json:"id"`
	Plot   int    `json:"plot"`
	SeedID uint   `json:"seed_id"`
	Seed   string `json:"seed"`    // 种子名
	Name   string `json:"name"`    // 开花后的花名
	MapID  uint   `json:"map_id"`  // 图鉴 id（开花后）
	Stage  int    `json:"stage"`   // 0空 1种子期 2花苗期 3花蕾期 4成熟
	StageName string `json:"stage_name"`
	Remain int    `json:"remain"`  // 剩余秒
	RemainTxt string `json:"remain_txt"`
	Drys   int    `json:"drys"`
	Weed   int    `json:"weed"`
	Pest   int    `json:"pest"`
	Yield  int    `json:"yield"`
	Amount int    `json:"amount"`
	NeedWater bool `json:"need_water"`
	NeedWeed  bool `json:"need_weed"`
	NeedPest  bool `json:"need_pest"`
	CanPick   bool `json:"can_pick"` // 好友可采摘
}

// 根据种子 id 查定义（数据库，空表回退内置）
func (h *GardenHandler) plotSeed(id uint) *seedDef {
	return h.seedByID(id)
}

// 推进花圃生长并返回视图（ASP：时间到了随机变一朵图鉴花，产量=随机±状态修正）
func (h *GardenHandler) plotView(p model.GardenPlot, uid uint, viewerIsOwner bool) plotView {
	v := plotView{
		ID: p.ID, Plot: p.Plot, SeedID: p.SeedID, Name: p.Name,
		Drys: p.Drys, Weed: p.Weed, Pest: p.Pest, Yield: p.Yield, Amount: p.Amount,
	}
	if p.Status == 0 || p.SeedID == 0 {
		v.Stage = 0
		v.StageName = "空花圃"
		return v
	}
	sd := h.seedByID(p.SeedID)
	if sd == nil {
		v.Stage = 0
		v.StageName = "空花圃"
		return v
	}
	v.Seed = sd.Name
	if p.Status == 2 {
		v.Stage = 4
		v.StageName = "成熟"
		if mid, _, ok := h.mapByName(p.Name); ok {
			v.MapID = mid
		}
		v.CanPick = !viewerIsOwner && p.Amount >= 2
		return v
	}
	// 计算生长阶段（分钟），操作判定用秒（对齐 ASP 后半段规则）
	elapsedSec := 0
	if p.SeedAt != nil {
		elapsedSec = int(time.Since(*p.SeedAt).Seconds())
	}
	elapsed := elapsedSec / 60
	total := sd.Seed + sd.Ling + sd.Buds
	if elapsed >= total {
		// 开花：随机变一朵图鉴花，产量=随机(less,more)±状态修正（最低1）
		yield := sd.Less + rand.Intn(sd.More-sd.Less+1)
		if p.Drys == 1 {
			yield++
		} else {
			yield--
		}
		if p.Weed == 1 {
			yield++
		} else {
			yield--
		}
		if p.Pest == 1 {
			yield++
		} else {
			yield--
		}
		if yield < 1 {
			yield = 1
		}
		// 随机一朵该种子的图鉴花
		maps := h.mapsBySeedName(sd.Name)
		var m model.GardenMap
		if len(maps) > 0 {
			m = maps[rand.Intn(len(maps))]
		} else {
			m = model.GardenMap{Name: sd.Name}
		}
		h.DB.Model(&p).Updates(map[string]interface{}{
			"name": m.Name, "yield": yield, "amount": yield, "status": 2,
		})
		p.Name = m.Name
		p.Yield = yield
		p.Amount = yield
		p.Status = 2
		v.Name = m.Name
		v.MapID = m.ID
		v.Yield = yield
		v.Amount = yield
		v.Stage = 4
		v.StageName = "成熟"
		v.CanPick = !viewerIsOwner && p.Amount >= 2
		return v
	}
	// 阶段
	var stage int
	var stageName string
	var remainSec int
	switch {
	case elapsed < sd.Seed:
		stage, stageName = 1, "种子期"
		remainSec = (sd.Seed - elapsed) * 60
	case elapsed < sd.Seed+sd.Ling:
		stage, stageName = 2, "花苗期"
		remainSec = (sd.Seed + sd.Ling - elapsed) * 60
	default:
		stage, stageName = 3, "花蕾期"
		remainSec = (sd.Seed + sd.Ling + sd.Buds - elapsed) * 60
	}
	v.Stage = stage
	v.StageName = stageName
	v.Remain = remainSec
	v.RemainTxt = fmtRemain(remainSec)
	// ASP：各阶段后半段才需要操作（秒级精确判定）
	if stage == 1 && elapsedSec > sd.Seed*60/2 && p.Drys == 0 {
		v.NeedWater = true
	}
	if stage == 2 && elapsedSec > (sd.Seed+sd.Ling/2)*60 && p.Weed == 0 {
		v.NeedWeed = true
	}
	if stage == 3 && elapsedSec > (sd.Seed+sd.Ling+sd.Buds/2)*60 && p.Pest == 0 {
		v.NeedPest = true
	}
	return v
}

func fmtRemain(sec int) string {
	if sec < 60 {
		return strconv.Itoa(sec) + "秒"
	}
	if sec < 3600 {
		return strconv.Itoa(sec/60) + "分钟"
	}
	return strconv.Itoa(sec/3600) + "小时"
}

// ============ 视图 ============

// 我的花园
func (h *GardenHandler) View(c *gin.Context) {
	uid := middleware.GetUID(c)
	g := h.ensureGarden(uid)
	var u model.User
	h.DB.First(&u, uid)

	var plots []model.GardenPlot
	h.DB.Where("user_id = ?", uid).Order("plot ASC").Find(&plots)
	out := make([]plotView, 0, len(plots))
	for _, p := range plots {
		out = append(out, h.plotView(p, uid, true))
	}
	// 背包
	var bags []model.GardenBag
	h.DB.Where("user_id = ?", uid).Find(&bags)
	// 图鉴进度
	total := len(h.loadMaps())
	var logged int64
	h.DB.Model(&model.GardenMapLog{}).Where("user_id = ?", uid).Count(&logged)
	// 最近点亮的花（TOP5，对齐 ASP my_garden 顶部展示）
	var logs []model.GardenMapLog
	h.DB.Where("user_id = ?", uid).Order("id DESC").Limit(5).Find(&logs)
	recent := []gin.H{}
	for _, l := range logs {
		for _, md := range h.loadMaps() {
			if md.ID == l.MapID {
				recent = append(recent, gin.H{"id": md.ID, "name": md.Name, "img": md.Img})
				break
			}
		}
	}
	// 消息（对齐参考站：显示发送者昵称 + 相对时间）
	var msgs []model.GardenMsg
	h.DB.Where("fid = ?", uid).Order("id DESC").Limit(10).Find(&msgs)
	msgOut := make([]gin.H, 0, len(msgs))
	for _, m := range msgs {
		nick := ""
		var su model.User
		if err := h.DB.First(&su, m.UID).Error; err == nil {
			nick = su.Nickname
		}
		msgOut = append(msgOut, gin.H{
			"id": m.ID, "nick": nick, "msg": m.Remark,
			"time_txt": timeAgo(m.CreatedAt),
		})
	}

	resp.OK(c, gin.H{
		"garden": gin.H{
			"name": g.Name, "level": g.Level, "level_name": gardenLevelName(g.Level),
			"point": g.Point, "need": g.Level * 200,
			"lands": g.Lands, "notice": g.Notice, "config": g.Config,
			"common": g.Common, "festival": g.Festival, "scarce": g.Scarce,
			"map_total": total, "map_got": logged,
		},
		"coins":      u.Coins,
		"plots":      out,
		"bag":        bags,
		"msgs":       msgOut,
		"recent_maps": recent,
	})
}

// 好友花园（可浇水/锄草/捉虫/采摘由园主状态决定，这里只读展示）
func (h *GardenHandler) Visit(c *gin.Context) {
	uid := middleware.GetUID(c)
	tid, _ := strconv.Atoi(c.Query("uid"))
	if tid == 0 {
		tid = int(uid)
	}
	var u model.User
	if err := h.DB.First(&u, tid).Error; err != nil {
		resp.NotFound(c, "无此用户")
		return
	}
	var g model.Garden
	if err := h.DB.Where("user_id = ?", tid).First(&g).Error; err != nil {
		resp.OK(c, gin.H{"garden": nil, "nickname": u.Nickname, "msg": "对方还没有开通花园"})
		return
	}
	var plots []model.GardenPlot
	h.DB.Where("user_id = ?", tid).Order("plot ASC").Find(&plots)
	out := make([]plotView, 0, len(plots))
	for _, p := range plots {
		v := h.plotView(p, uint(tid), false)
		// 已采摘过的不能摘
		if v.Stage == 4 {
			var c1 int64
			h.DB.Model(&model.GardenLandLog{}).Where("land_id = ? AND user_id = ?", p.ID, uid).Count(&c1)
			if c1 > 0 {
				v.CanPick = false
			}
		}
		out = append(out, v)
	}
	resp.OK(c, gin.H{
		"garden":   gin.H{"user_id": uint(tid), "name": g.Name, "level": g.Level, "level_name": gardenLevelName(g.Level), "point": g.Point, "need": g.Level * 200, "lands": g.Lands, "notice": g.Notice, "config": g.Config},
		"nickname": u.Nickname,
		"plots":    out,
		"is_owner": uint(tid) == uid,
	})
}

// ============ 操作 ============

// 商店（种子列表）
func (h *GardenHandler) Shop(c *gin.Context) {
	g := h.ensureGarden(middleware.GetUID(c))
	out := make([]gin.H, 0)
	for _, sd := range h.loadSeeds() {
		if sd.DType != 0 || sd.Status == 0 {
			continue
		}
		out = append(out, gin.H{"id": sd.ID, "name": sd.Name, "level": sd.Level, "level_name": gardenLevelName(sd.Level), "price": sd.Price, "vip_price": sd.Price * 8 / 10,
			"seed": sd.Seed, "ling": sd.Ling, "buds": sd.Buds, "less": sd.Less, "more": sd.More,
			"yield_avg": (sd.Less + sd.More) / 2,
			"hours": float64(int(float64(sd.Seed+sd.Ling+sd.Buds)/60*10)) / 10,
			"remark": sd.Remark, "garden_level": g.Level})
	}
	resp.OK(c, out)
}

// 购买种子
type buyReq struct {
	ID     int `json:"id" binding:"required"`
	Amount int `json:"amount"`
}

func (h *GardenHandler) Buy(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req buyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请选择要购买的花种")
		return
	}
	sd := h.seedByID(uint(req.ID))
	if sd == nil || sd.DType != 0 {
		resp.ParamError(c, "无此花种")
		return
	}
	amount := req.Amount
	if amount < 1 || amount > 99 {
		amount = 1
	}
	g := h.ensureGarden(uid)
	if g.Level < sd.Level {
		resp.ParamError(c, "购买此花种需花园达到"+strconv.Itoa(sd.Level)+"级")
		return
	}
	var u model.User
	h.DB.First(&u, uid)
	cost := sd.Price * amount
	if u.Coins < cost {
		resp.ParamError(c, "G币不足，买不起"+strconv.Itoa(amount)+"颗"+sd.Name+"种子")
		return
	}
	h.DB.Model(&model.User{}).Where("id = ?", uid).Update("coins", gorm.Expr("coins - ?", cost))
	// 入背包
	var bag model.GardenBag
	if err := h.DB.Where("user_id = ? AND seed_id = ?", uid, req.ID).First(&bag).Error; err != nil {
		h.DB.Create(&model.GardenBag{UserID: uid, SeedID: uint(req.ID), Name: sd.Name, Amount: amount})
	} else {
		h.DB.Model(&bag).Update("amount", gorm.Expr("amount + ?", amount))
	}
	h.DB.First(&u, uid)
	resp.OK(c, gin.H{"coins": u.Coins, "name": sd.Name, "amount": amount})
}

// 播种：从背包扣 1 颗 → 种到第一个空花圃
type sowReq struct {
	SeedID int `json:"seed_id" binding:"required"`
}

func (h *GardenHandler) Sow(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req sowReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请选择要播种的种子")
		return
	}
	sd := h.seedByID(uint(req.SeedID))
	if sd == nil {
		resp.ParamError(c, "无此种子")
		return
	}
	var bag model.GardenBag
	if err := h.DB.Where("user_id = ? AND seed_id = ?", uid, req.SeedID).First(&bag).Error; err != nil || bag.Amount < 1 {
		resp.ParamError(c, "您没有该种子了")
		return
	}
	h.ensureGarden(uid)
	var plot model.GardenPlot
	if err := h.DB.Where("user_id = ? AND status = 0", uid).Order("plot ASC").First(&plot).Error; err != nil {
		resp.ParamError(c, "您没有空花圃，先去添置或收获吧")
		return
	}
	// 扣种子
	if bag.Amount > 1 {
		h.DB.Model(&bag).Update("amount", gorm.Expr("amount - 1"))
	} else {
		h.DB.Delete(&bag)
	}
	now := time.Now()
	h.DB.Model(&plot).Updates(map[string]interface{}{
		"seed_id": req.SeedID, "name": "", "drys": 0, "weed": 0, "pest": 0,
		"yield": 0, "amount": 0, "status": 1, "seed_at": &now,
	})
	h.addGardenPoint(uid, gardenExpSow)
	resp.OK(c, gin.H{"msg": "播种成功，经验值+" + strconv.Itoa(gardenExpSow), "seed": sd.Name})
}

// 浇水/锄草/捉虫（ASP：各+1经验，给园主发消息）
func (h *GardenHandler) Care(c *gin.Context) {
	uid := middleware.GetUID(c)
	kind := c.Param("kind")
	pid, _ := strconv.Atoi(c.PostForm("id"))
	if pid == 0 {
		resp.ParamError(c, "请选择花圃")
		return
	}
	var plot model.GardenPlot
	if err := h.DB.First(&plot, pid).Error; err != nil || plot.Status != 1 {
		resp.ParamError(c, "无此花朵")
		return
	}
	// 只能操作自己的花圃（ASP water.asp: WHERE uid=MyId）
	if plot.UserID != uid {
		resp.Forbidden(c, "只能打理自己的花园")
		return
	}
	var field string
	var doneMsg string
	switch kind {
	case "water":
		if plot.Drys == 1 {
			resp.ParamError(c, "已经浇过水了")
			return
		}
		field, doneMsg = "drys", "浇水"
	case "weed":
		if plot.Weed == 1 {
			resp.ParamError(c, "已经锄过草了")
			return
		}
		field, doneMsg = "weed", "锄草"
	case "pest":
		if plot.Pest == 1 {
			resp.ParamError(c, "已经捉过虫了")
			return
		}
		field, doneMsg = "pest", "捉虫"
	default:
		resp.ParamError(c, "未知操作")
		return
	}
	h.DB.Model(&plot).Update(field, 1)
	h.addGardenPoint(uid, gardenExpCare)
	h.addGardenMsg(uid, plot.UserID, "来花园帮忙"+doneMsg+"。")
	resp.OK(c, gin.H{"msg": doneMsg + "成功！经验值+" + strconv.Itoa(gardenExpCare)})
}

// 收获（园主自己，成熟花圃全部收获）
func (h *GardenHandler) Harvest(c *gin.Context) {
	uid := middleware.GetUID(c)
	pid, _ := strconv.Atoi(c.PostForm("id"))
	if pid == 0 {
		resp.ParamError(c, "请选择花圃")
		return
	}
	var plot model.GardenPlot
	if err := h.DB.First(&plot, pid).Error; err != nil || plot.UserID != uid {
		resp.ParamError(c, "无此花朵")
		return
	}
	if plot.Status != 2 {
		resp.ParamError(c, "还没成熟，再等等吧")
		return
	}
	amount := plot.Amount
	if amount < 1 {
		amount = 1
	}
	flower := plot.Name
	if flower == "" {
		flower = "未知花"
	}
	// 花进花篮 + 花园累计
	h.addFlower(uid, flower, amount)
	h.DB.Model(&model.Garden{}).Where("user_id = ?", uid).Update("basket_cnt", gorm.Expr("basket_cnt + ?", amount))
	// 金币 random(0,amount)
	money := rand.Intn(amount + 1)
	if money > 0 {
		h.DB.Model(&model.User{}).Where("id = ?", uid).Update("coins", gorm.Expr("coins + ?", money))
	}
	// 图鉴点亮 & 经验
	exp := gardenExpKnown
	if mapID, dtype, ok := h.mapByName(flower); ok {
		var c1 int64
		h.DB.Model(&model.GardenMapLog{}).Where("user_id = ? AND map_id = ?", uid, mapID).Count(&c1)
		if c1 == 0 {
			h.DB.Create(&model.GardenMapLog{UserID: uid, MapID: mapID})
			if dtype == 0 {
				exp = gardenExpNew
				h.DB.Model(&model.Garden{}).Where("user_id = ?", uid).Update("common", gorm.Expr("common + 1"))
				h.addGardenMsg(uid, uid, "种出了"+flower+"花朵")
			} else {
				exp = gardenExpNewRare
				field := "festival"
				if dtype == 2 {
					field = "scarce"
				}
				h.DB.Model(&model.Garden{}).Where("user_id = ?", uid).Update(field, gorm.Expr(field+" + 1"))
				h.addGardenMsg(uid, uid, "点亮了"+flower+"图谱")
			}
		}
	}
	h.addGardenPoint(uid, exp)
	// 清空花圃 + 清除采摘记录
	h.DB.Model(&plot).Updates(map[string]interface{}{
		"seed_id": 0, "name": "", "drys": 0, "weed": 0, "pest": 0,
		"yield": 0, "amount": 0, "status": 0, "seed_at": nil,
	})
	h.DB.Where("land_id = ?", plot.ID).Delete(&model.GardenLandLog{})
	var u model.User
	h.DB.First(&u, uid)
	resp.OK(c, gin.H{"flower": flower, "amount": amount, "exp": exp, "money": money, "coins": u.Coins, "msg": "收获成功！经验值+" + strconv.Itoa(exp) + "，金币+" + strconv.Itoa(money)})
}

// 铲除花朵（对齐 plant_del：清空地里的花，返还地块）
func (h *GardenHandler) DelPlot(c *gin.Context) {
	uid := middleware.GetUID(c)
	pid, _ := strconv.Atoi(c.PostForm("id"))
	if pid == 0 {
		resp.ParamError(c, "请选择花圃")
		return
	}
	var plot model.GardenPlot
	if err := h.DB.First(&plot, pid).Error; err != nil || plot.UserID != uid {
		resp.ParamError(c, "无此花朵")
		return
	}
	if plot.Status == 0 || plot.SeedID == 0 {
		resp.ParamError(c, "这个花盆是空的")
		return
	}
	h.DB.Model(&plot).Updates(map[string]interface{}{
		"seed_id": 0, "name": "", "drys": 0, "weed": 0, "pest": 0,
		"yield": 0, "amount": 0, "status": 0, "seed_at": nil,
	})
	h.DB.Where("land_id = ?", plot.ID).Delete(&model.GardenLandLog{})
	resp.OK(c, gin.H{"msg": "铲除花朵成功"})
}

// 采摘（好友）：每花圃每人限 1 次，花圃剩>=2 才能摘
func (h *GardenHandler) Pick(c *gin.Context) {
	uid := middleware.GetUID(c)
	pid, _ := strconv.Atoi(c.PostForm("id"))
	if pid == 0 {
		resp.ParamError(c, "请选择花圃")
		return
	}
	var plot model.GardenPlot
	if err := h.DB.First(&plot, pid).Error; err != nil || plot.Status != 2 {
		resp.ParamError(c, "无此花朵")
		return
	}
	if plot.UserID == uid {
		resp.ParamError(c, "这是您自己的花园，请用收获")
		return
	}
	// 权限：0全部 1仅好友 2禁止
	var g model.Garden
	h.DB.Where("user_id = ?", plot.UserID).First(&g)
	if g.Config == 2 {
		resp.ParamError(c, "园主设置任何人都不能采摘哦")
		return
	}
	if g.Config == 1 {
		var c1 int64
		h.DB.Model(&model.Friendship{}).Where("(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)", plot.UserID, uid, uid, plot.UserID).Count(&c1)
		if c1 == 0 {
			resp.ParamError(c, "园主设置只能好友才能采摘哦")
			return
		}
	}
	// 每人每花圃限一次
	var c1 int64
	h.DB.Model(&model.GardenLandLog{}).Where("land_id = ? AND user_id = ?", pid, uid).Count(&c1)
	if c1 > 0 {
		resp.ParamError(c, "已经摘过啦，做人不要太贪心哦")
		return
	}
	if plot.Amount < 2 {
		resp.ParamError(c, "花圃只剩1朵"+plot.Name+"，不能采摘")
		return
	}
	h.DB.Model(&plot).Update("amount", gorm.Expr("amount - 1"))
	h.DB.Create(&model.GardenLandLog{LandID: plot.ID, UserID: uid})
	h.addFlower(uid, plot.Name, 1)
	h.addGardenMsg(uid, plot.UserID, "来花园摘走了1朵"+plot.Name+"。")
	resp.OK(c, gin.H{"flower": plot.Name, "msg": "您成功采摘了好友1朵" + plot.Name + "，花朵已进入您的花篮"})
}

// 添置新花圃（等级+金币）
func (h *GardenHandler) AddLand(c *gin.Context) {
	uid := middleware.GetUID(c)
	g := h.ensureGarden(uid)
	if g.Lands >= gardenMaxLands {
		resp.ParamError(c, "花圃已到上限（"+strconv.Itoa(gardenMaxLands)+"块）")
		return
	}
	needLevel, needMoney := gardenLandCost(g.Lands)
	if g.Level < needLevel {
		resp.ParamError(c, "您的花园等级不足，需要达到"+strconv.Itoa(needLevel)+"级")
		return
	}
	var u model.User
	h.DB.First(&u, uid)
	if u.Coins < needMoney {
		resp.ParamError(c, "G币不足，添置花圃需要"+strconv.Itoa(needMoney)+" G币")
		return
	}
	h.DB.Model(&model.User{}).Where("id = ?", uid).Update("coins", gorm.Expr("coins - ?", needMoney))
	h.DB.Model(&model.Garden{}).Where("id = ?", g.ID).Update("lands", g.Lands+1)
	h.DB.Create(&model.GardenPlot{UserID: uid, Plot: g.Lands, Status: 0})
	h.DB.First(&u, uid)
	resp.OK(c, gin.H{"lands": g.Lands + 1, "coins": u.Coins, "msg": "成功添置新花圃！"})
}

// 花篮（我的花库存）
func (h *GardenHandler) Basket(c *gin.Context) {
	uid := middleware.GetUID(c)
	var rows []model.UserFlower
	h.DB.Where("user_id = ? AND count > 0", uid).Order("flower ASC").Find(&rows)
	out := []gin.H{}
	for _, r := range rows {
		out = append(out, gin.H{"flower": r.Flower, "count": r.Count})
	}
	resp.OK(c, out)
}

// 花瓶（收到的花）
func (h *GardenHandler) Bottle(c *gin.Context) {
	uid := middleware.GetUID(c)
	var rows []model.GardenBottle
	h.DB.Where("user_id = ? AND count > 0", uid).Order("flower ASC").Find(&rows)
	out := []gin.H{}
	for _, r := range rows {
		out = append(out, gin.H{"flower": r.Flower, "count": r.Count})
	}
	resp.OK(c, out)
}

// 送花：花篮 → 对方花瓶
type giftReq struct {
	ToUID  uint   `json:"to_uid" binding:"required"`
	Flower string `json:"flower" binding:"required"`
	Amount int    `json:"amount"`
	Remark string `json:"remark"`
}

func (h *GardenHandler) Gift(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req giftReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请选择赠送的花朵")
		return
	}
	if req.ToUID == uid {
		resp.ParamError(c, "不能送给自己")
		return
	}
	var tu model.User
	if err := h.DB.First(&tu, req.ToUID).Error; err != nil {
		resp.NotFound(c, "无此会员")
		return
	}
	amount := req.Amount
	if amount < 1 {
		amount = 1
	}
	var uf model.UserFlower
	if err := h.DB.Where("user_id = ? AND flower = ?", uid, req.Flower).First(&uf).Error; err != nil || uf.Count < amount {
		resp.ParamError(c, "您没有那么多鲜花")
		return
	}
	if uf.Count > amount {
		h.DB.Model(&uf).Update("count", gorm.Expr("count - ?", amount))
	} else {
		h.DB.Delete(&uf)
	}
	h.addBottle(req.ToUID, req.Flower, amount)
	h.DB.Model(&model.Garden{}).Where("user_id = ?", uid).Update("basket_cnt", gorm.Expr("basket_cnt - ?", amount))
	h.DB.Model(&model.Garden{}).Where("user_id = ?", req.ToUID).Update("bottle_cnt", gorm.Expr("bottle_cnt + ?", amount))
	remark := req.Remark
	if remark == "" {
		remark = "希望你开心快乐！"
	}
	h.DB.Create(&model.GardenGift{FromUID: uid, ToUID: req.ToUID, Flower: req.Flower, Amount: amount, Remark: remark})
	h.addGardenMsg(uid, req.ToUID, "送了您"+strconv.Itoa(amount)+"朵"+req.Flower)
	resp.OK(c, gin.H{"msg": "赠送成功！"})
}

// 送花记录
func (h *GardenHandler) GiftLog(c *gin.Context) {
	uid := middleware.GetUID(c)
	var rows []model.GardenGift
	h.DB.Where("from_uid = ? OR to_uid = ?", uid, uid).Order("id DESC").Limit(50).Find(&rows)
	out := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		var fu, tu model.User
		fromNick, toNick := "?", "?"
		if err := h.DB.First(&fu, r.FromUID).Error; err == nil {
			fromNick = fu.Nickname
		}
		if err := h.DB.First(&tu, r.ToUID).Error; err == nil {
			toNick = tu.Nickname
		}
		out = append(out, gin.H{
			"id": r.ID, "from_uid": r.FromUID, "to_uid": r.ToUID,
			"from_nick": fromNick, "to_nick": toNick,
			"flower": r.Flower, "amount": r.Amount, "remark": r.Remark,
			"time_txt": timeAgo(r.CreatedAt),
		})
	}
	resp.OK(c, out)
}

// 魔法屋：合成列表
func (h *GardenHandler) Room(c *gin.Context) {
	uid := middleware.GetUID(c)
	var uf []model.UserFlower
	h.DB.Where("user_id = ?", uid).Find(&uf)
	have := map[string]int{}
	for _, r := range uf {
		have[r.Flower] = r.Count
	}
	// 按产物种子聚合材料
	type mixItem struct {
		SeedID uint        `json:"seed_id"`
		Name   string      `json:"name"`
		Level  int         `json:"level"`
		Mats   []gin.H     `json:"mats"`
		Can    bool        `json:"can"`
	}
	out := []mixItem{}
	seen := map[uint]int{}
	for _, sd := range h.loadSeeds() {
		if sd.DType == 0 || sd.Status == 0 {
			continue
		}
		idx, ok := seen[sd.ID]
		if !ok {
			idx = len(out)
			seen[sd.ID] = idx
			out = append(out, mixItem{SeedID: sd.ID, Name: sd.Name, Level: sd.Level, Mats: []gin.H{}, Can: true})
		}
	}
	for _, mix := range h.loadMixes() {
		for _, sd := range h.loadSeeds() {
			if sd.ID == mix.SeedID {
				idx, ok := seen[sd.ID]
				if !ok {
					idx = len(out)
					seen[sd.ID] = idx
					out = append(out, mixItem{SeedID: sd.ID, Name: sd.Name, Level: sd.Level, Mats: []gin.H{}, Can: true})
				}
				hv := have[mix.Flower]
				out[idx].Mats = append(out[idx].Mats, gin.H{"flower": mix.Flower, "need": mix.Need, "have": hv})
				if hv < mix.Need {
					out[idx].Can = false
				}
				break
			}
		}
	}
	resp.OK(c, out)
}

// 合成：消耗花朵 → 获得特殊种子 1 颗
type mixReq struct {
	SeedID int `json:"seed_id" binding:"required"`
}

func (h *GardenHandler) Mix(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req mixReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请选择要合成的花种")
		return
	}
	sd := h.seedByID(uint(req.SeedID))
	if sd == nil || sd.DType == 0 {
		resp.ParamError(c, "无此花种")
		return
	}
	var uf []model.UserFlower
	h.DB.Where("user_id = ?", uid).Find(&uf)
	have := map[string]int{}
	for _, r := range uf {
		have[r.Flower] = r.Count
	}
	mixes := h.mixesBySeed(sd.Name)
	for _, mix := range mixes {
		if have[mix.Flower] < mix.Need {
			resp.ParamError(c, "所需花朵不足，您目前还不能合成"+sd.Name)
			return
		}
	}
	// 扣材料
	for _, mix := range mixes {
		h.DB.Model(&model.UserFlower{}).Where("user_id = ? AND flower = ?", uid, mix.Flower).
			Update("count", gorm.Expr("count - ?", mix.Need))
	}
	// 获得种子 1 颗
	var bag model.GardenBag
	if err := h.DB.Where("user_id = ? AND seed_id = ?", uid, req.SeedID).First(&bag).Error; err != nil {
		h.DB.Create(&model.GardenBag{UserID: uid, SeedID: uint(req.SeedID), Name: sd.Name, Amount: 1})
	} else {
		h.DB.Model(&bag).Update("amount", gorm.Expr("amount + 1"))
	}
	h.addGardenPoint(uid, gardenExpMix)
	resp.OK(c, gin.H{"msg": "合成成功！经验值+" + strconv.Itoa(gardenExpMix) + "，获得" + sd.Name + "种子一颗"})
}

// 花之图谱（分类：0普通 1独特 2珍稀）
func (h *GardenHandler) MapList(c *gin.Context) {
	uid := middleware.GetUID(c)
	ty, _ := strconv.Atoi(c.Query("ty"))
	var logged []model.GardenMapLog
	h.DB.Where("user_id = ?", uid).Find(&logged)
	got := map[uint]bool{}
	for _, l := range logged {
		got[l.MapID] = true
	}
	out := make([]gin.H, 0, len(h.loadMaps()))
	for _, md := range h.loadMaps() {
		if md.DType != ty {
			continue
		}
		img := md.Img
		if img == "" {
			img = "m_s_" + strconv.Itoa(int(md.ID)) + ".gif"
		}
		item := gin.H{"id": md.ID, "name": md.Name, "dtype": md.DType, "img": img, "got": got[md.ID]}
		// 图谱详情（对齐 map.asp：花种等级/单价/预计成花/成花时间/花语）
		if sd := h.seedByID(md.SeedID); sd != nil {
			item["seed_id"] = sd.ID
			item["seed_name"] = sd.Name
			item["level_name"] = gardenLevelName(sd.Level)
			item["price"] = sd.Price
			item["yield_avg"] = (sd.Less + sd.More) / 2
			item["hours"] = float64(int(float64(sd.Seed+sd.Ling+sd.Buds)/60*10)) / 10
			item["remark"] = sd.Remark
		}
		out = append(out, item)
	}
	var g model.Garden
	h.DB.Where("user_id = ?", uid).First(&g)
	resp.OK(c, gin.H{"ty": ty, "list": out, "common": g.Common, "festival": g.Festival, "scarce": g.Scarce})
}

// 花园设置
type gardenSettingReq struct {
	Name   string `json:"name"`
	Notice string `json:"notice"`
	Config int    `json:"config"`
}

func (h *GardenHandler) Setting(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req gardenSettingReq
	_ = c.ShouldBindJSON(&req)
	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Notice != "" {
		updates["notice"] = req.Notice
	}
	if req.Config >= 0 && req.Config <= 2 {
		updates["config"] = req.Config
	}
	if len(updates) == 0 {
		resp.ParamError(c, "没有需要修改的内容")
		return
	}
	h.DB.Model(&model.Garden{}).Where("user_id = ?", uid).Updates(updates)
	resp.OK(c, gin.H{"msg": "修改成功！"})
}

// 花园消息
func (h *GardenHandler) Msgs(c *gin.Context) {
	uid := middleware.GetUID(c)
	var rows []model.GardenMsg
	h.DB.Where("fid = ?", uid).Order("id DESC").Limit(50).Find(&rows)
	// 7天前消息清理
	h.DB.Where("fid = ? AND created_at < ?", uid, time.Now().AddDate(0, 0, -7)).Delete(&model.GardenMsg{})
	out := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		var u model.User
		nick := "?"
		if err := h.DB.First(&u, r.UID).Error; err == nil {
			nick = u.Nickname
		}
		out = append(out, gin.H{"uid": r.UID, "nickname": nick, "remark": r.Remark, "created_at": r.CreatedAt, "time_txt": timeAgo(r.CreatedAt)})
	}
	resp.OK(c, out)
}

// ============ 活动（保留） ============

// 提交活动任务：用指定花朵兑换奖励花（对齐参考站「完成任务需要/奖励」）
var actTaskNeeds = []struct {
	Flower string
	N      int
}{{"玫瑰花", 6}, {"郁金香", 6}}

const actTaskReward = "朝暮盈霄花"

type actSubmitReq struct {
	ID     uint `json:"id" binding:"required"`
	Amount int  `json:"amount"`
}

func (h *GardenHandler) SubmitActivity(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req actSubmitReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请选择活动")
		return
	}
	var act model.GardenActivity
	if err := h.DB.First(&act, req.ID).Error; err != nil {
		resp.NotFound(c, "活动不存在")
		return
	}
	amount := req.Amount
	if amount < 1 {
		amount = 1
	}
	if amount > 9 {
		amount = 9
	}
	var uf []model.UserFlower
	h.DB.Where("user_id = ?", uid).Find(&uf)
	have := map[string]int{}
	for _, r := range uf {
		have[r.Flower] = r.Count
	}
	for _, need := range actTaskNeeds {
		if have[need.Flower] < need.N*amount {
			resp.ParamError(c, "所需花朵不足，您目前还不能完成任务")
			return
		}
	}
	for _, need := range actTaskNeeds {
		h.DB.Model(&model.UserFlower{}).Where("user_id = ? AND flower = ?", uid, need.Flower).
			Update("count", gorm.Expr("count - ?", need.N*amount))
	}
	h.addFlower(uid, actTaskReward, amount)
	h.DB.Model(&model.User{}).Where("id = ?", uid).Update("exp", gorm.Expr("exp + ?", 20*amount))
	resp.OK(c, gin.H{"reward": actTaskReward, "amount": amount})
}

// 公开：花园活动列表
func (h *GardenHandler) ActivityList(c *gin.Context) {
	var list []model.GardenActivity
	h.DB.Where("status = 1").Order("created_at DESC").Limit(30).Find(&list)
	resp.OK(c, list)
}

// 后台：活动分页
func (h *GardenHandler) AdminActivities(c *gin.Context) {
	var list []model.GardenActivity
	h.DB.Order("created_at DESC").Find(&list)
	resp.OK(c, list)
}

type actReq struct {
	Title  string `json:"title" binding:"required,min=1,max=60"`
	Desc   string `json:"desc" binding:"max=200"`
	Status int    `json:"status"`
}

func (h *GardenHandler) AdminActCreate(c *gin.Context) {
	var req actReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "活动标题必填")
		return
	}
	if req.Status == 0 {
		req.Status = 1
	}
	h.DB.Create(&model.GardenActivity{Title: req.Title, Desc: req.Desc, Status: req.Status})
	resp.OK(c, nil)
}

func (h *GardenHandler) AdminActUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req actReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "活动标题必填")
		return
	}
	h.DB.Model(&model.GardenActivity{}).Where("id = ?", id).Updates(map[string]interface{}{
		"title": req.Title, "desc": req.Desc, "status": req.Status,
	})
	resp.OK(c, nil)
}

func (h *GardenHandler) AdminActDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.GardenActivity{}, id)
	resp.OK(c, nil)
}

// ============ 后台：花园数据维护（种子/图鉴/配方） ============

// 种子列表（含停用）
func (h *GardenHandler) AdminSeeds(c *gin.Context) {
	var rows []model.GardenSeed
	h.DB.Order("d_type ASC, id ASC").Find(&rows)
	resp.OK(c, rows)
}

type adminSeedReq struct {
	Name   string `json:"name" binding:"required"`
	DType  int    `json:"dtype"`
	Level  int    `json:"level"`
	Price  int    `json:"price"`
	Seed   int    `json:"seed"`
	Ling   int    `json:"ling"`
	Buds   int    `json:"buds"`
	Less   int    `json:"less"`
	More   int    `json:"more"`
	Remark string `json:"remark"`
	Status int    `json:"status"`
}

func (h *GardenHandler) AdminSeedCreate(c *gin.Context) {
	var req adminSeedReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请填写种子名称")
		return
	}
	var n int64
	h.DB.Model(&model.GardenSeed{}).Where("name = ?", req.Name).Count(&n)
	if n > 0 {
		resp.ParamError(c, "同名种子已存在")
		return
	}
	if req.Status == 0 {
		req.Status = 1
	}
	h.DB.Create(&model.GardenSeed{Name: req.Name, DType: req.DType, Level: req.Level, Price: req.Price,
		Seed: req.Seed, Ling: req.Ling, Buds: req.Buds, Less: req.Less, More: req.More, Remark: req.Remark, Status: req.Status})
	resp.OK(c, nil)
}

func (h *GardenHandler) AdminSeedUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req adminSeedReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请填写种子名称")
		return
	}
	h.DB.Model(&model.GardenSeed{}).Where("id = ?", id).Updates(map[string]interface{}{
		"name": req.Name, "dtype": req.DType, "level": req.Level, "price": req.Price,
		"seed": req.Seed, "ling": req.Ling, "buds": req.Buds, "less": req.Less, "more": req.More,
		"remark": req.Remark, "status": req.Status,
	})
	resp.OK(c, nil)
}

func (h *GardenHandler) AdminSeedDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.GardenSeed{}, id)
	h.DB.Model(&model.GardenBag{}).Where("seed_id = ?", id).Delete(&model.GardenBag{})
	resp.OK(c, nil)
}

// 图鉴列表
func (h *GardenHandler) AdminMaps(c *gin.Context) {
	rows := h.loadMaps()
	seeds := h.loadSeeds()
	seedName := map[uint]string{}
	for _, s := range seeds {
		seedName[s.ID] = s.Name
	}
	out := make([]gin.H, 0, len(rows))
	for _, m := range rows {
		out = append(out, gin.H{"id": m.ID, "seed_id": m.SeedID, "seed_name": seedName[m.SeedID], "name": m.Name, "dtype": m.DType, "img": m.Img})
	}
	resp.OK(c, out)
}

type adminMapReq struct {
	SeedID uint   `json:"seed_id"`
	Name   string `json:"name" binding:"required"`
	DType  int    `json:"dtype"`
	Img    string `json:"img"`
}

func (h *GardenHandler) AdminMapCreate(c *gin.Context) {
	var req adminMapReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请填写花名")
		return
	}
	var n int64
	h.DB.Model(&model.GardenMap{}).Where("name = ?", req.Name).Count(&n)
	if n > 0 {
		resp.ParamError(c, "同名花已存在")
		return
	}
	h.DB.Create(&model.GardenMap{SeedID: req.SeedID, Name: req.Name, DType: req.DType, Img: req.Img})
	resp.OK(c, nil)
}

func (h *GardenHandler) AdminMapUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req adminMapReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请填写花名")
		return
	}
	h.DB.Model(&model.GardenMap{}).Where("id = ?", id).Updates(map[string]interface{}{
		"seed_id": req.SeedID, "name": req.Name, "dtype": req.DType, "img": req.Img,
	})
	resp.OK(c, nil)
}

func (h *GardenHandler) AdminMapDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.GardenMap{}, id)
	h.DB.Model(&model.GardenMapLog{}).Where("map_id = ?", id).Delete(&model.GardenMapLog{})
	resp.OK(c, nil)
}

// 配方列表（带产物种子名）
func (h *GardenHandler) AdminMixes(c *gin.Context) {
	rows := h.loadMixes()
	seeds := h.loadSeeds()
	seedInfo := map[uint]model.GardenSeed{}
	for _, s := range seeds {
		seedInfo[s.ID] = s
	}
	out := make([]gin.H, 0, len(rows))
	for _, m := range rows {
		s := seedInfo[m.SeedID]
		out = append(out, gin.H{"id": m.ID, "seed_id": m.SeedID, "seed_name": s.Name,
			"seed_img": "s_s_" + strconv.Itoa(int(m.SeedID)) + ".gif", "flower": m.Flower, "need": m.Need})
	}
	resp.OK(c, out)
}

type adminMixReq struct {
	SeedID uint   `json:"seed_id" binding:"required"`
	Flower string `json:"flower" binding:"required"`
	Need   int    `json:"need"`
}

func (h *GardenHandler) AdminMixCreate(c *gin.Context) {
	var req adminMixReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请选择产物种子和材料花")
		return
	}
	var n int64
	h.DB.Model(&model.GardenMix{}).Where("seed_id = ? AND flower = ?", req.SeedID, req.Flower).Count(&n)
	if n > 0 {
		resp.ParamError(c, "该配方已存在")
		return
	}
	if req.Need < 1 {
		req.Need = 1
	}
	h.DB.Create(&model.GardenMix{SeedID: req.SeedID, Flower: req.Flower, Need: req.Need})
	resp.OK(c, nil)
}

func (h *GardenHandler) AdminMixUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req adminMixReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请选择产物种子和材料花")
		return
	}
	if req.Need < 1 {
		req.Need = 1
	}
	h.DB.Model(&model.GardenMix{}).Where("id = ?", id).Updates(map[string]interface{}{
		"seed_id": req.SeedID, "flower": req.Flower, "need": req.Need,
	})
	resp.OK(c, nil)
}

func (h *GardenHandler) AdminMixDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.GardenMix{}, id)
	resp.OK(c, nil)
}

// ============ 精灵花册（用户端） ============

// 精灵花册列表：返回全部精灵 + 解锁状态（点亮图谱数达标即解锁）
func (h *GardenHandler) Elves(c *gin.Context) {
	uid := middleware.GetUID(c)
	var elves []model.GardenElf
	h.DB.Where("status = 1").Order("sort ASC, id ASC").Find(&elves)
	var g model.Garden
	if err := h.DB.Where("user_id = ?", uid).First(&g).Error; err != nil {
		g.Lands = 2
		g.Level = 1
	}
	var logs []model.GardenElfLog
	h.DB.Where("user_id = ?", uid).Find(&logs)
	got := map[uint]bool{}
	for _, l := range logs {
		got[l.ElfID] = true
	}
	// 点亮图谱数
	var mapCnt int64
	h.DB.Model(&model.GardenMapLog{}).Where("user_id = ?", uid).Count(&mapCnt)
	unlock := 0
	out := make([]gin.H, 0, len(elves))
	for _, e := range elves {
		ok := int(mapCnt) >= e.NeedMap
		if ok {
			unlock++
		}
		// 首次解锁记录
		if ok && !got[e.ID] {
			h.DB.Create(&model.GardenElfLog{UserID: uid, ElfID: e.ID})
		}
		img := e.Img
		if img == "" {
			img = "elf_" + strconv.Itoa(int(e.ID)) + ".png"
		}
		out = append(out, gin.H{"id": e.ID, "name": e.Name, "desc": e.Desc, "img": img,
			"need_map": e.NeedMap, "unlocked": ok})
	}
	resp.OK(c, gin.H{"list": out, "unlocked": unlock, "total": len(elves), "map_got": mapCnt})
}

// 花园排行：按花园等级经验降序 TOP20（对齐 ASP LevelName 体系）
func (h *GardenHandler) Rank(c *gin.Context) {
	var rows []model.Garden
	h.DB.Order("level DESC, point DESC, id ASC").Limit(20).Find(&rows)
	out := make([]gin.H, 0, len(rows))
	for _, g := range rows {
		var u model.User
		nick := ""
		if err := h.DB.First(&u, g.UserID).Error; err == nil {
			nick = u.Nickname
		}
		out = append(out, gin.H{"rank": len(out) + 1, "user_id": g.UserID, "nickname": nick,
			"name": g.Name, "level": g.Level, "level_name": gardenLevelName(g.Level), "point": g.Point,
			"lands": g.Lands, "map_got": g.Common + g.Festival + g.Scarce})
	}
	resp.OK(c, out)
}

// ============ 精灵花册（管理端） ============

func (h *GardenHandler) AdminElves(c *gin.Context) {
	var rows []model.GardenElf
	h.DB.Order("sort ASC, id ASC").Find(&rows)
	resp.OK(c, rows)
}

type adminElfReq struct {
	Name    string `json:"name" binding:"required"`
	Desc    string `json:"desc"`
	Img     string `json:"img"`
	NeedMap int    `json:"need_map"`
	Sort    int    `json:"sort"`
	Status  int    `json:"status"`
}

func (h *GardenHandler) AdminElfCreate(c *gin.Context) {
	var req adminElfReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请填写精灵名称")
		return
	}
	var n int64
	h.DB.Model(&model.GardenElf{}).Where("name = ?", req.Name).Count(&n)
	if n > 0 {
		resp.ParamError(c, "同名精灵已存在")
		return
	}
	if req.Status == 0 {
		req.Status = 1
	}
	if req.NeedMap < 1 {
		req.NeedMap = 1
	}
	h.DB.Create(&model.GardenElf{Name: req.Name, Desc: req.Desc, Img: req.Img, NeedMap: req.NeedMap, Sort: req.Sort, Status: req.Status})
	resp.OK(c, nil)
}

func (h *GardenHandler) AdminElfUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req adminElfReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请填写精灵名称")
		return
	}
	if req.NeedMap < 1 {
		req.NeedMap = 1
	}
	h.DB.Model(&model.GardenElf{}).Where("id = ?", id).Updates(map[string]interface{}{
		"name": req.Name, "desc": req.Desc, "img": req.Img, "need_map": req.NeedMap, "sort": req.Sort, "status": req.Status,
	})
	resp.OK(c, nil)
}

func (h *GardenHandler) AdminElfDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.GardenElf{}, id)
	h.DB.Model(&model.GardenElfLog{}).Where("elf_id = ?", id).Delete(&model.GardenElfLog{})
	resp.OK(c, nil)
}
