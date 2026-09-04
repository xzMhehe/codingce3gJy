package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

type GardenHandler struct{ DB *gorm.DB }

// 花盆上限 / 添置一个花盆的花费
const gardenPotMax = 12
const gardenPotCost = 100

// 作物：名称 / 种子价 / 成熟秒 / 收获价
type cropItem struct {
	Name  string `json:"name"`
	Seed  int    `json:"seed"`
	Secs  int    `json:"secs"`
	Sell  int    `json:"sell"`
}

var gardenCrops = []cropItem{
	{"向日葵", 5, 30, 12},
	{"玫瑰花", 10, 60, 25},
	{"郁金香", 20, 120, 55},
	{"月光花", 40, 240, 115},
}

// 花园状态：4 块地
// 我的花盆数
func userPots(u model.User) int {
	p := u.GardenPots
	if p < 2 {
		p = gardenPotMax
	}
	return p
}

func (h *GardenHandler) View(c *gin.Context) {
	uid := middleware.GetUID(c)
	var u model.User
	h.DB.First(&u, uid)
	pots := userPots(u)

	// 确保每个花盆都有记录（无重复）
	for i := 0; i < pots; i++ {
		var already int64
		h.DB.Model(&model.GardenPlot{}).Where("user_id = ? AND plot = ?", uid, i).Count(&already)
		if already == 0 {
			h.DB.Create(&model.GardenPlot{UserID: uid, Plot: i, Status: 0})
		}
	}
	var plots []model.GardenPlot
	h.DB.Where("user_id = ?", uid).Order("plot ASC").Find(&plots)

	out := []gin.H{}
	for _, p := range plots {
		state := "empty"
		remain := 0
		if p.Status == 1 {
			state = "growing"
			remain = plantRemain(p)
			if remain <= 0 {
				state = "ripe"
			}
		}
		out = append(out, gin.H{"index": p.Plot, "crop": p.Crop, "status": state, "remain": remain})
	}
	resp.OK(c, gin.H{"plots": out, "coins": u.Coins, "crops": gardenCrops, "pots": pots})
}

type plantReq struct {
	Index int    `json:"index"`
	Crop  string `json:"crop"`
}

// 种植
func (h *GardenHandler) Plant(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req plantReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请选择地块和作物")
		return
	}
	if req.Index < 0 {
		resp.ParamError(c, "花盆不存在")
		return
	}
	var crop *cropItem
	for i := range gardenCrops {
		if gardenCrops[i].Name == req.Crop {
			crop = &gardenCrops[i]
			break
		}
	}
	if crop == nil {
		resp.ParamError(c, "作物不存在")
		return
	}
	var u model.User
	h.DB.First(&u, uid)
	if req.Index >= userPots(u) {
		resp.ParamError(c, "花盆不存在")
		return
	}
	if u.Coins < crop.Seed {
		resp.ParamError(c, "金币不足，买不起" + crop.Name + "种子")
		return
	}
	var plot model.GardenPlot
	if err := h.DB.Where("user_id = ? AND plot = ?", uid, req.Index).First(&plot).Error; err != nil {
		resp.ParamError(c, "请先刷新花园")
		return
	}
	if plot.Status == 1 {
		resp.ParamError(c, "这块地已经种上了")
		return
	}
	h.DB.Model(&u).Update("coins", gorm.Expr("coins - ?", crop.Seed))
	now := time.Now()
	h.DB.Model(&plot).Updates(map[string]interface{}{"crop": crop.Name, "status": 1, "seed_at": &now})
	resp.OK(c, nil)
}

// 收获
func (h *GardenHandler) Harvest(c *gin.Context) {
	uid := middleware.GetUID(c)
	idx, _ := strconv.Atoi(c.PostForm("index"))
	if idx < 0 {
		resp.ParamError(c, "花盆不存在")
		return
	}
	var u model.User
	h.DB.First(&u, uid)
	if idx >= userPots(u) {
		resp.ParamError(c, "花盆不存在")
		return
	}
	var plot model.GardenPlot
	if err := h.DB.Where("user_id = ? AND plot = ?", uid, idx).First(&plot).Error; err != nil {
		resp.ParamError(c, "请先刷新花园")
		return
	}
	if plot.Status != 1 || plantRemain(plot) > 0 {
		resp.ParamError(c, "还没成熟，再等等吧")
		return
	}
	sell := 0
	for _, cr := range gardenCrops {
		if cr.Name == plot.Crop {
			sell = cr.Sell
			break
		}
	}
	reward := sell / 2
	if reward < 1 {
		reward = 1
	}
	h.DB.Model(&plot).Updates(map[string]interface{}{"status": 0, "crop": ""})
	h.addFlower(uid, plot.Crop, 1)
	h.DB.Model(&model.User{}).Where("id = ?", uid).Update("coins", gorm.Expr("coins + ?", reward))
	h.DB.Model(&model.User{}).Where("id = ?", uid).Update("exp", gorm.Expr("exp + ?", 5))
	h.DB.First(&u, uid)
	resp.OK(c, gin.H{"gain": reward, "flower": plot.Crop, "coins": u.Coins})
}

// 添置新花盆（花费金币，+1 花盆）
func (h *GardenHandler) AddPot(c *gin.Context) {
	uid := middleware.GetUID(c)
	var u model.User
	h.DB.First(&u, uid)
	pots := userPots(u)
	if pots >= gardenPotMax {
		resp.ParamError(c, "花盆已到上限")
		return
	}
	if u.Coins < gardenPotCost {
		resp.ParamError(c, "金币不足，添置花盆需要 " + strconv.Itoa(gardenPotCost) + " 金币")
		return
	}
	h.DB.Model(&u).Update("coins", gorm.Expr("coins - ?", gardenPotCost))
	h.DB.Model(&u).Update("garden_pots", pots+1)
	var already int64
	h.DB.Model(&model.GardenPlot{}).Where("user_id = ? AND plot = ?", uid, pots).Count(&already)
	if already == 0 {
		h.DB.Create(&model.GardenPlot{UserID: uid, Plot: pots, Status: 0})
	}
	resp.OK(c, gin.H{"pots": pots + 1, "coins": u.Coins - gardenPotCost})
}

// 收获：花入库存 + 少量金币/经验
func (h *GardenHandler) addFlower(uid uint, flower string, n int) {
	var uf model.UserFlower
	if err := h.DB.Where("user_id = ? AND flower = ?", uid, flower).First(&uf).Error; err != nil {
		h.DB.Create(&model.UserFlower{UserID: uid, Flower: flower, Count: n})
		return
	}
	h.DB.Model(&uf).Update("count", gorm.Expr("count + ?", n))
}

// 花篮（花库存）
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

// 合成表（魔法屋）
type synMat struct {
	Flower string `json:"flower"`
	Need   int    `json:"need"`
	Have   int    `json:"have"`
}
type synItem struct {
	Name string   `json:"name"`
	Mats []synMat `json:"mats"`
	Can  bool     `json:"can"`
}

var synTable = []struct {
	name string
	mats []synMat
}{
	{"银色菊花", []synMat{{"向日葵", 5, 0}}},
	{"银野花", []synMat{{"向日葵", 3, 0}, {"玫瑰花", 2, 0}}},
	{"端阳花", []synMat{{"玫瑰花", 4, 0}, {"郁金香", 2, 0}}},
	{"银友谊花", []synMat{{"向日葵", 2, 0}, {"玫瑰花", 2, 0}, {"郁金香", 2, 0}}},
	{"银色烈焰焚情", []synMat{{"月光花", 3, 0}}},
	{"金色烈焰焚情", []synMat{{"月光花", 6, 0}}},
}

// 魔法屋：合成列表（含每项所需材料与是否有）
func (h *GardenHandler) SynList(c *gin.Context) {
	uid := middleware.GetUID(c)
	var uf []model.UserFlower
	h.DB.Where("user_id = ?", uid).Find(&uf)
	have := map[string]int{}
	for _, r := range uf {
		have[r.Flower] = r.Count
	}
	out := []gin.H{}
	for i, it := range synTable {
		can := true
		mats := make([]synMat, len(it.mats))
		for j, m := range it.mats {
			h := have[m.Flower]
			mats[j] = synMat{Flower: m.Flower, Need: m.Need, Have: h}
			if h < m.Need {
				can = false
			}
		}
		out = append(out, gin.H{"index": i + 1, "name": it.name, "mats": mats, "can": can})
	}
	resp.OK(c, out)
}

type synReq struct {
	Flower string `json:"flower" binding:"required"`
}

// 合成花朵
func (h *GardenHandler) Synthesize(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req synReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请选择要合成的花")
		return
	}
	var target *struct {
		name string
		mats []synMat
	}
	for i := range synTable {
		if synTable[i].name == req.Flower {
			target = &synTable[i]
			break
		}
	}
	if target == nil {
		resp.ParamError(c, "合成目标不存在")
		return
	}
	var uf []model.UserFlower
	h.DB.Where("user_id = ?", uid).Find(&uf)
	have := map[string]int{}
	for _, r := range uf {
		have[r.Flower] = r.Count
	}
	for _, m := range target.mats {
		if have[m.Flower] < m.Need {
			resp.ParamError(c, "所需花朵不足，您目前还不能合成")
			return
		}
	}
	// 扣材料
	for _, m := range target.mats {
		h.DB.Model(&model.UserFlower{}).Where("user_id = ? AND flower = ?", uid, m.Flower).
			Update("count", gorm.Expr("count - ?", m.Need))
	}
	h.addFlower(uid, target.name, 1)
	resp.OK(c, nil)
}

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

// 剩余成熟秒
func plantRemain(p model.GardenPlot) int {
	if p.Status != 1 {
		return 0
	}
	secs := 0
	for _, cr := range gardenCrops {
		if cr.Name == p.Crop {
			secs = cr.Secs
			break
		}
	}
	if p.SeedAt == nil {
		return 0
	}
	elapsed := int(time.Since(*p.SeedAt).Seconds())
	remain := secs - elapsed
	if remain < 0 {
		return 0
	}
	return remain
}
