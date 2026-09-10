package handler

import (
	"math/rand"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 开心农场（对齐诺哈三代 ASP 版 wap/game/farm 玩法）
// 升级曲线：升级所需经验 = (level+1)*farmPointStep；每 5 级系统赠送 1 块地
const (
	farmPointStep = 10
	farmInitLands = 4
)

type FarmHandler struct{ DB *gorm.DB }

// ---------- 内部工具 ----------

func (h *FarmHandler) farmOf(uid uint) *model.Farm {
	var f model.Farm
	if err := h.DB.Where("user_id = ?", uid).First(&f).Error; err != nil {
		u := h.userBrief(uid)
		f = model.Farm{UserID: uid, Name: u.Nickname + "的农场", Level: 1, Point: 0, Mucks: 3, CSteal: 0,
			LastRefresh: time.Now().Format("2006-01-02")}
		h.DB.Create(&f)
		for i := 1; i <= farmInitLands; i++ {
			h.DB.Create(&model.FarmLand{UserID: uid, Sort: i})
		}
		h.farmSend(0, uid, "欢迎来到开心农场，系统赠送了你 "+strconv.Itoa(farmInitLands)+" 块菜地，快去翻地种植吧！")
	}
	return &f
}

func (h *FarmHandler) userBrief(uid uint) *model.User {
	var u model.User
	h.DB.Select("id,username,nickname,coins").First(&u, uid)
	return &u
}

// nickMap 批量取昵称（管理端列表用）
func (h *FarmHandler) nickMap(uidSet map[uint]bool) map[uint]string {
	m := make(map[uint]string, len(uidSet))
	for uid := range uidSet {
		m[uid] = ""
	}
	if len(uidSet) == 0 {
		return m
	}
	ids := make([]uint, 0, len(uidSet))
	for uid := range uidSet {
		ids = append(ids, uid)
	}
	var users []model.User
	h.DB.Select("id, nickname").Where("id IN ?", ids).Find(&users)
	for _, u := range users {
		m[u.ID] = u.Nickname
	}
	return m
}

func (h *FarmHandler) farmSend(from, to uint, content string) {
	h.DB.Create(&model.FarmMsg{UserID: to, FID: from, Content: content})
}

// farmAddPoint 加经验并处理升级（对齐 FarmAddPoint：每 5 级送地）
func (h *FarmHandler) farmAddPoint(uid uint, pts int) {
	if pts <= 0 {
		return
	}
	f := h.farmOf(uid)
	h.farmSend(0, uid, "您获得"+strconv.Itoa(pts)+"点经验！")
	f.Point += pts
	need := (f.Level + 1) * farmPointStep
	if f.Point > need {
		f.Point -= need
		f.Level++
		if f.Level%5 == 0 {
			var maxSort int
			h.DB.Model(&model.FarmLand{}).Where("user_id = ?", uid).
				Select("COALESCE(MAX(sort), 0)").Scan(&maxSort)
			h.DB.Create(&model.FarmLand{UserID: uid, Sort: maxSort + 1})
			h.farmSend(0, uid, "恭喜您升为"+strconv.Itoa(f.Level)+"级，系统赠送了一块菜地！")
		} else {
			h.farmSend(0, uid, "恭喜您升为"+strconv.Itoa(f.Level)+"级。")
		}
	}
	h.DB.Model(&model.Farm{}).Where("id = ?", f.ID).Updates(map[string]interface{}{
		"level": f.Level, "point": f.Point})
}

// 每日刷新：施肥次数恢复为 3（对齐 Farm_Refresh）
func (h *FarmHandler) farmDailyRefresh(f *model.Farm) {
	today := time.Now().Format("2006-01-02")
	if f.LastRefresh != today {
		h.DB.Model(&model.Farm{}).Where("id = ? AND mucks < 3", f.ID).Update("mucks", 3)
		h.DB.Model(&model.Farm{}).Where("id = ?", f.ID).Update("last_refresh", today)
		f.Mucks = 3
		f.LastRefresh = today
	}
}

// landTime 地块种植/成熟时间（空地为 NULL，回退当前时间，仅对有作物的地块实际参与计算）
func landTime(land *model.FarmLand) (time.Time, time.Time) {
	now := time.Now()
	if land.PlantedAt == nil {
		return now, now
	}
	p := *land.PlantedAt
	m := now
	if land.MatureAt != nil {
		m = *land.MatureAt
	}
	return p, m
}

// 生长阶段（对齐 my_farm.asp：NeedTime/20 分段 2/5/9/15）
// 返回 1发芽 2小叶子 3大叶子 4开花 5将熟 6已成熟 与剩余描述
func farmStage(now, plantedAt, matureAt time.Time) (int, string) {
	if !now.Before(matureAt) {
		return 6, "已成熟"
	}
	totalMin := matureAt.Sub(plantedAt).Minutes()
	if totalMin <= 0 {
		return 6, "已成熟"
	}
	pastMin := now.Sub(plantedAt).Minutes()
	remain := func(min float64) string {
		if min < 1 {
			return "1分钟"
		}
		if min < 60 {
			return strconv.Itoa(int(min)) + "分钟"
		}
		return strconv.Itoa(int(min)/60) + "小时" + strconv.Itoa(int(min)%60) + "分钟"
	}
	seg := totalMin / 20
	switch {
	case pastMin < seg*2:
		return 1, remain(seg*2-pastMin) + "后发芽"
	case pastMin < seg*5:
		return 2, remain(seg*5-pastMin) + "后小叶子"
	case pastMin < seg*9:
		return 3, remain(seg*9-pastMin) + "后大叶子"
	case pastMin < seg*15:
		return 4, remain(seg*15-pastMin) + "后开花"
	default:
		return 5, remain(totalMin-pastMin) + "后成熟"
	}
}

// 随机灾害（对齐 CropStatus：生长过 1/4 后首次查看时随机触发旱/草/虫）
func (h *FarmHandler) farmCropStatus(land *model.FarmLand) {
	if land.Drys == 0 {
		land.Drys = 1 + rand.Intn(2)
	}
	if land.Weed == 0 {
		land.Weed = 1 + rand.Intn(2)
	}
	if land.Pest == 0 {
		land.Pest = 1 + rand.Intn(2)
	}
	h.DB.Model(&model.FarmLand{}).Where("id = ?", land.ID).
		Updates(map[string]interface{}{"drys": land.Drys, "weed": land.Weed, "pest": land.Pest})
}

// landView 地块输出（stage/status 由服务端算好，前端直接渲染）
func landView(land model.FarmLand, now time.Time) gin.H {
	stage, statusTxt := 0, ""
	if land.Type == 1 {
		p, m := landTime(&land)
		stage, statusTxt = farmStage(now, p, m)
	}
	return gin.H{
		"id": land.ID, "sort": land.Sort, "type": land.Type, "plow": land.Plow,
		"seed_id": land.SeedID, "name": land.Name,
		"drys": land.Drys, "weed": land.Weed, "pest": land.Pest, "trap": land.Trap,
		"yield": land.Yield, "cycle": land.Cycle, "period": land.Period,
		"stage": stage, "status_txt": statusTxt,
		"need_water": land.Drys == 1, "need_weed": land.Weed == 1, "need_pest": land.Pest == 1,
		"mature": stage == 6,
	}
}

type plantable struct {
	ID    uint
	Cycle int
	Aging int
	Again int
	Yield int
	Point int
	Level int
	Price int
	Name  string
}

func (h *FarmHandler) seedByID(id uint) *plantable {
	var s model.FarmSeed
	if err := h.DB.First(&s, id).Error; err != nil || s.Status != 1 {
		return nil
	}
	return &plantable{ID: s.ID, Cycle: s.Cycle, Aging: s.Aging, Again: s.Again,
		Yield: s.Yield, Point: s.Point, Level: s.Level, Price: s.Price, Name: s.Name}
}

// bagAdd 背包累加（无则新增）
func (h *FarmHandler) bagAdd(uid, oid uint, name string, dtype, amount int) {
	var bag model.FarmBag
	err := h.DB.Where("user_id = ? AND oid = ? AND dtype = ?", uid, oid, dtype).First(&bag).Error
	if err != nil {
		h.DB.Create(&model.FarmBag{UserID: uid, Oid: oid, Name: name, DType: dtype, Amount: amount})
		return
	}
	h.DB.Model(&model.FarmBag{}).Where("id = ?", bag.ID).Update("amount", gorm.Expr("amount + ?", amount))
}

func (h *FarmHandler) bagConsume(bag *model.FarmBag) {
	if bag.Amount <= 1 {
		h.DB.Delete(model.FarmBag{}, bag.ID)
	} else {
		h.DB.Model(&model.FarmBag{}).Where("id = ?", bag.ID).Update("amount", bag.Amount-1)
	}
}

// ---------- 接口 ----------

type landReq struct {
	LandID uint `json:"land_id" binding:"required"`
}

// View 我的农场主页
func (h *FarmHandler) View(c *gin.Context) {
	uid := middleware.GetUID(c)
	f := h.farmOf(uid)
	h.farmDailyRefresh(f)
	u := h.userBrief(uid)
	now := time.Now()

	var lands []model.FarmLand
	h.DB.Where("user_id = ?", uid).Order("sort ASC, id ASC").Find(&lands)
	landViews := []gin.H{}
	for _, l := range lands {
		if l.Type == 1 {
			var tmp = l
			lp, lm := landTime(&tmp)
			if _, st := farmStage(now, lp, lm); st != "已成熟" && l.Drys == 0 {
				h.farmCropStatus(&tmp)
				l = tmp
			}
		}
		landViews = append(landViews, landView(l, now))
	}

	// 未读消息弹出（对齐 PopUpMessage：查看即置已读）
	var msgs []model.FarmMsg
	h.DB.Where("user_id = ?", uid).Order("id DESC").Limit(10).Find(&msgs)
	h.DB.Model(&model.FarmMsg{}).Where("user_id = ? AND status = 0", uid).Update("status", 1)
	msgViews := []gin.H{}
	for _, m := range msgs {
		nick := "系统消息"
		if m.FID > 0 {
			if mu := h.userBrief(m.FID); mu.ID != 0 {
				nick = mu.Nickname
			}
		}
		msgViews = append(msgViews, gin.H{"id": m.ID, "nick": nick, "fid": m.FID,
			"msg": m.Content, "time_txt": m.CreatedAt.Format("01-02 15:04")})
	}

	// 最新加入农场（对齐 main.asp TOP5）
	var recent []model.Farm
	h.DB.Order("created_at DESC").Limit(5).Find(&recent)
	recentViews := []gin.H{}
	for _, r := range recent {
		ru := h.userBrief(r.UserID)
		recentViews = append(recentViews, gin.H{"uid": r.UserID, "nick": ru.Nickname,
			"level": r.Level, "time_txt": fmtTimeAgo(now.Sub(r.CreatedAt))})
	}

	resp.OK(c, gin.H{
		"farm": gin.H{"uid": uid, "name": f.Name, "level": f.Level, "point": f.Point,
			"need": (f.Level+1)*farmPointStep, "mucks": f.Mucks, "csteal": f.CSteal,
			"coins": u.Coins, "lands": len(landViews), "nick": u.Nickname},
		"lands": landViews, "msgs": msgViews, "recent": recentViews,
	})
}

// Visit 参观他人农场
func (h *FarmHandler) Visit(c *gin.Context) {
	uid := middleware.GetUID(c)
	target, _ := strconv.Atoi(c.Query("uid"))
	if target <= 0 || uint(target) == uid {
		resp.ParamError(c, "农场不存在")
		return
	}
	f := h.farmOf(uid)
	h.farmDailyRefresh(f)
	tf := h.farmOf(uint(target))
	tu := h.userBrief(uint(target))
	now := time.Now()

	var lands []model.FarmLand
	h.DB.Where("user_id = ?", target).Order("sort ASC, id ASC").Find(&lands)
	landViews := []gin.H{}
	for _, l := range lands {
		if l.Type == 1 {
			var tmp = l
			lp, lm := landTime(&tmp)
			if _, st := farmStage(now, lp, lm); st != "已成熟" && l.Drys == 0 {
				h.farmCropStatus(&tmp)
				l = tmp
			}
		}
		landViews = append(landViews, landView(l, now))
	}

	// 我是否是对方的奴隶
	var slaveCnt int64
	h.DB.Model(&model.FarmSlave{}).Where("owner_uid = ? AND fid = ?", target, uid).Count(&slaveCnt)

	resp.OK(c, gin.H{
		"farm": gin.H{"uid": target, "name": tf.Name, "level": tf.Level, "csteal": tf.CSteal,
			"nick": tu.Nickname, "is_slave": slaveCnt > 0},
		"lands": landViews,
	})
}

// Neighbors 邻居（好友中有农场的 + 最新开农场的人）
func (h *FarmHandler) Neighbors(c *gin.Context) {
	uid := middleware.GetUID(c)
	var friendIDs []uint
	h.DB.Model(&model.Friendship{}).Where("user_id = ? AND status = 1", uid).
		Pluck("friend_id", &friendIDs)

	out := []gin.H{}
	if len(friendIDs) > 0 {
		var farms []model.Farm
		h.DB.Where("user_id IN ?", friendIDs).Order("level DESC").Limit(20).Find(&farms)
		for _, f := range farms {
			u := h.userBrief(f.UserID)
			out = append(out, gin.H{"uid": f.UserID, "nick": u.Nickname,
				"name": f.Name, "level": f.Level, "is_friend": true})
		}
	}
	if len(out) == 0 {
		var farms []model.Farm
		h.DB.Where("user_id <> ?", uid).Order("created_at DESC").Limit(10).Find(&farms)
		for _, f := range farms {
			u := h.userBrief(f.UserID)
			out = append(out, gin.H{"uid": f.UserID, "nick": u.Nickname,
				"name": f.Name, "level": f.Level, "is_friend": false})
		}
	}
	resp.OK(c, out)
}

// Shop 商店（种子/化肥/陷阱）
func (h *FarmHandler) Shop(c *gin.Context) {
	var seeds []model.FarmSeed
	h.DB.Where("status = 1").Order("level ASC, id ASC").Find(&seeds)
	seedViews := []gin.H{}
	for _, s := range seeds {
		seedViews = append(seedViews, gin.H{"id": s.ID, "name": s.Name, "cycle": s.Cycle,
			"aging": s.Aging, "again": s.Again, "yield": s.Yield, "price": s.Price,
			"seed_price": s.Price*5*s.Cycle, "point": s.Point, "level": s.Level})
	}
	var mucks []model.FarmMuck
	h.DB.Where("status = 1").Order("id ASC").Find(&mucks)
	var traps []model.FarmTrap
	h.DB.Where("status = 1").Order("id ASC").Find(&traps)
	resp.OK(c, gin.H{"seeds": seedViews, "mucks": mucks, "traps": traps})
}

type farmBuyReq struct {
	Kind   string `json:"kind" binding:"required"` // seed / muck / trap
	ID     uint   `json:"id" binding:"required"`
	Amount int    `json:"amount" binding:"required,min=1,max=99"`
}

// Buy 购买（对齐 buy_seed/shop_muck/shop_trap：种子价=price*5*cycle，需等级）
func (h *FarmHandler) Buy(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req farmBuyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请选择要购买的物品")
		return
	}
	u := h.userBrief(uid)
	switch req.Kind {
	case "seed":
		var s model.FarmSeed
		if err := h.DB.First(&s, req.ID).Error; err != nil || s.Status != 1 {
			resp.NotFound(c, "无此种子")
			return
		}
		f := h.farmOf(uid)
		if s.Level > f.Level {
			resp.ParamError(c, "您的农场等级不够，需要"+strconv.Itoa(s.Level)+"级")
			return
		}
		cost := s.Price * 5 * s.Cycle * req.Amount
		if u.Coins < cost {
			resp.ParamError(c, "G币不足，需要"+strconv.Itoa(cost)+" G币")
			return
		}
		h.DB.Model(&model.User{}).Where("id = ?", uid).Update("coins", gorm.Expr("coins - ?", cost))
		h.bagAdd(uid, s.ID, s.Name, 1, req.Amount)
		h.farmSend(0, uid, "您购买了"+strconv.Itoa(req.Amount)+"个["+s.Name+"]种子，花费["+strconv.Itoa(cost)+"G币]。")
		resp.OK(c, gin.H{"msg": "购买成功", "coins": u.Coins - cost})
	case "muck":
		var m model.FarmMuck
		if err := h.DB.First(&m, req.ID).Error; err != nil || m.Status != 1 {
			resp.NotFound(c, "无此化肥")
			return
		}
		cost := m.Price * req.Amount
		if u.Coins < cost {
			resp.ParamError(c, "G币不足，需要"+strconv.Itoa(cost)+" G币")
			return
		}
		h.DB.Model(&model.User{}).Where("id = ?", uid).Update("coins", gorm.Expr("coins - ?", cost))
		h.bagAdd(uid, m.ID, m.Name, 2, req.Amount)
		h.farmSend(0, uid, "您购买了"+strconv.Itoa(req.Amount)+"个["+m.Name+"]，花费["+strconv.Itoa(cost)+"G币]。")
		resp.OK(c, gin.H{"msg": "购买成功", "coins": u.Coins - cost})
	case "trap":
		var t model.FarmTrap
		if err := h.DB.First(&t, req.ID).Error; err != nil || t.Status != 1 {
			resp.NotFound(c, "无此陷阱")
			return
		}
		cost := t.Price * req.Amount
		if u.Coins < cost {
			resp.ParamError(c, "G币不足，需要"+strconv.Itoa(cost)+" G币")
			return
		}
		h.DB.Model(&model.User{}).Where("id = ?", uid).Update("coins", gorm.Expr("coins - ?", cost))
		h.bagAdd(uid, t.ID, t.Name, 3, req.Amount)
		h.farmSend(0, uid, "您购买了"+strconv.Itoa(req.Amount)+"个["+t.Name+"]，花费["+strconv.Itoa(cost)+"G币]。")
		resp.OK(c, gin.H{"msg": "购买成功", "coins": u.Coins - cost})
	default:
		resp.ParamError(c, "kind 必须是 seed / muck / trap")
	}
}

// Bag 背包（种子/化肥/陷阱，种植/施肥/设陷阱时选品用）
func (h *FarmHandler) Bag(c *gin.Context) {
	uid := middleware.GetUID(c)
	dtype, _ := strconv.Atoi(c.DefaultQuery("dtype", "1"))
	if dtype != 1 && dtype != 2 && dtype != 3 {
		resp.ParamError(c, "dtype 必须是 1种子 2化肥 3陷阱")
		return
	}
	var bags []model.FarmBag
	h.DB.Where("user_id = ? AND dtype = ? AND amount > 0", uid, dtype).Order("id ASC").Find(&bags)
	resp.OK(c, bags)
}

// Warehouse 仓库（收获的果实）
func (h *FarmHandler) Warehouse(c *gin.Context) {
	uid := middleware.GetUID(c)
	var bags []model.FarmBag
	h.DB.Where("user_id = ? AND dtype = 11 AND amount > 0", uid).Order("id ASC").Find(&bags)
	out := []gin.H{}
	for _, b := range bags {
		price := 0
		if s := h.seedByID(b.Oid); s != nil {
			price = s.Price
		}
		out = append(out, gin.H{"id": b.ID, "oid": b.Oid, "name": b.Name,
			"amount": b.Amount, "price": price})
	}
	resp.OK(c, out)
}

// Plow 翻地（对齐 plow.asp：+2 经验）
func (h *FarmHandler) Plow(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req landReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	var land model.FarmLand
	if err := h.DB.Where("id = ? AND user_id = ?", req.LandID, uid).First(&land).Error; err != nil {
		resp.NotFound(c, "土地不存在")
		return
	}
	if land.Type != 0 || land.Plow == 1 {
		resp.ParamError(c, "这块地不用翻")
		return
	}
	h.DB.Model(&land).Update("plow", 1)
	h.farmAddPoint(uid, 2)
	resp.OK(c, gin.H{"msg": "翻地成功，经验+2"})
}

// Plant 种植（对齐 plant.asp：+3 经验，扣 1 种子）
func (h *FarmHandler) Plant(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		LandID uint `json:"land_id" binding:"required"`
		BagID  uint `json:"bag_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请选择土地和种子")
		return
	}
	var land model.FarmLand
	if err := h.DB.Where("id = ? AND user_id = ? AND type = 0 AND plow = 1", req.LandID, uid).
		First(&land).Error; err != nil {
		resp.ParamError(c, "这块地不能种植，请先翻地")
		return
	}
	var bag model.FarmBag
	if err := h.DB.Where("id = ? AND user_id = ? AND dtype = 1 AND amount > 0", req.BagID, uid).
		First(&bag).Error; err != nil {
		resp.ParamError(c, "没有这种子")
		return
	}
	s := h.seedByID(bag.Oid)
	if s == nil {
		resp.NotFound(c, "种子信息不存在")
		return
	}
	now := time.Now()
	h.DB.Model(&land).Updates(map[string]interface{}{
		"type": 1, "plow": 0, "seed_id": s.ID, "name": s.Name,
		"drys": 0, "weed": 0, "pest": 0, "trap": 0,
		"yield": s.Yield, "cycle": s.Cycle, "period": 1,
		"planted_at": now, "mature_at": now.Add(time.Duration(s.Aging) * time.Minute),
	})
	h.bagConsume(&bag)
	h.farmAddPoint(uid, 3)
	resp.OK(c, gin.H{"msg": "种植成功，经验+3"})
}

// Care 浇水/除草/除虫（对齐 drys/weed/pest.asp：+2 经验，可帮好友操作）
func (h *FarmHandler) Care(c *gin.Context) {
	uid := middleware.GetUID(c)
	kind := c.Param("kind")
	var req struct {
		LandID uint `json:"land_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	var land model.FarmLand
	if err := h.DB.First(&land, req.LandID).Error; err != nil {
		resp.NotFound(c, "土地不存在")
		return
	}
	col, cur, label := "", 0, ""
	switch kind {
	case "water":
		col, cur, label = "drys", land.Drys, "浇了水"
	case "weed":
		col, cur, label = "weed", land.Weed, "除了草"
	case "pest":
		col, cur, label = "pest", land.Pest, "除了虫"
	default:
		resp.ParamError(c, "kind 必须是 water / weed / pest")
		return
	}
	if cur != 1 {
		resp.ParamError(c, "这块地不需要"+map[string]string{"water": "浇水", "weed": "除草", "pest": "除虫"}[kind])
		return
	}
	h.DB.Model(&model.FarmLand{}).Where("id = ?", land.ID).Update(col, 3)
	h.farmAddPoint(uid, 2)
	if land.UserID != uid {
		h.farmSend(0, land.UserID, h.userBrief(uid).Nickname+" 帮你的作物"+label+"。")
	}
	resp.OK(c, gin.H{"msg": "操作成功，经验+2"})
}

// Ppest 放虫（对齐 ppest.asp：恶作剧，无经验）
func (h *FarmHandler) Ppest(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req landReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	var land model.FarmLand
	if err := h.DB.Where("id = ? AND user_id <> ?", req.LandID, uid).First(&land).Error; err != nil {
		resp.NotFound(c, "土地不存在")
		return
	}
	if land.Type != 1 || land.Pest != 0 {
		resp.ParamError(c, "这块地放不了虫")
		return
	}
	h.DB.Model(&land).Update("pest", 1)
	h.farmSend(0, land.UserID, h.userBrief(uid).Nickname+" 在你的作物里放了一条虫。")
	resp.OK(c, gin.H{"msg": "放虫成功"})
}

// Muck 施肥（对齐 muck.asp：每日 3 次，endtime 提前）
func (h *FarmHandler) Muck(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		LandID uint `json:"land_id" binding:"required"`
		BagID  uint `json:"bag_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请选择土地和化肥")
		return
	}
	f := h.farmOf(uid)
	h.farmDailyRefresh(f)
	if f.Mucks < 1 {
		resp.ParamError(c, "您的施肥次数已经达到上限（每日3次）")
		return
	}
	var land model.FarmLand
	if err := h.DB.Where("id = ? AND user_id = ? AND type = 1", req.LandID, uid).
		First(&land).Error; err != nil {
		resp.ParamError(c, "这块地没有作物")
		return
	}
	if land.MatureAt == nil || land.MatureAt.Before(time.Now()) {
		resp.ParamError(c, "作物已成熟，不用施肥")
		return
	}
	var bag model.FarmBag
	if err := h.DB.Where("id = ? AND user_id = ? AND dtype = 2 AND amount > 0", req.BagID, uid).
		First(&bag).Error; err != nil {
		resp.ParamError(c, "没有这种化肥")
		return
	}
	var m model.FarmMuck
	if err := h.DB.First(&m, bag.Oid).Error; err != nil {
		resp.NotFound(c, "化肥信息不存在")
		return
	}
	h.DB.Model(&land).Update("mature_at", (*land.MatureAt).Add(-time.Duration(m.Speed)*time.Minute))
	h.bagConsume(&bag)
	h.DB.Model(&model.Farm{}).Where("id = ?", f.ID).Update("mucks", f.Mucks-1)
	resp.OK(c, gin.H{"msg": "施肥成功，成熟时间提前" + strconv.Itoa(m.Speed) + "分钟，今日剩余施肥次数" + strconv.Itoa(f.Mucks-1)})
}

// Trap 设陷阱（对齐 trap.asp：覆盖设置几率）
func (h *FarmHandler) Trap(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		LandID uint `json:"land_id" binding:"required"`
		BagID  uint `json:"bag_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请选择土地和陷阱")
		return
	}
	var land model.FarmLand
	if err := h.DB.Where("id = ? AND user_id = ? AND type = 1", req.LandID, uid).
		First(&land).Error; err != nil {
		resp.ParamError(c, "这块地没有作物")
		return
	}
	var bag model.FarmBag
	if err := h.DB.Where("id = ? AND user_id = ? AND dtype = 3 AND amount > 0", req.BagID, uid).
		First(&bag).Error; err != nil {
		resp.ParamError(c, "没有这种陷阱")
		return
	}
	var t model.FarmTrap
	if err := h.DB.First(&t, bag.Oid).Error; err != nil {
		resp.NotFound(c, "陷阱信息不存在")
		return
	}
	h.DB.Model(&land).Update("trap", t.Rate)
	h.bagConsume(&bag)
	resp.OK(c, gin.H{"msg": "设陷阱成功，触发几率" + strconv.Itoa(t.Rate) + "%"})
}

// Pick 收获（对齐 pick.asp：未处理的旱/草/虫各扣 1 产量；多季作物进入下一季）
func (h *FarmHandler) Pick(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req landReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	var land model.FarmLand
	if err := h.DB.Where("id = ? AND user_id = ? AND type = 1", req.LandID, uid).
		First(&land).Error; err != nil {
		resp.ParamError(c, "这块地没有作物")
		return
	}
	if land.MatureAt == nil || land.MatureAt.After(time.Now()) {
		resp.ParamError(c, "作物还没成熟")
		return
	}
	s := h.seedByID(land.SeedID)
	yield := land.Yield
	if yield >= 3 {
		if land.Drys == 1 {
			yield--
		}
		if land.Weed == 1 {
			yield--
		}
		if land.Pest == 1 {
			yield--
		}
	}
	resetLand := func() {
		now := time.Now()
		h.DB.Model(&land).Updates(map[string]interface{}{
			"type": 0, "plow": 0, "seed_id": 0, "name": "",
			"drys": 0, "weed": 0, "pest": 0, "trap": 0,
			"yield": 0, "cycle": 0, "period": 0,
			"planted_at": now, "mature_at": now,
		})
	}
	if s == nil {
		resetLand()
		resp.OK(c, gin.H{"msg": "收获完成", "amount": 0})
		return
	}
	if land.Cycle <= land.Period {
		resetLand()
	} else {
		now := time.Now()
		h.DB.Model(&land).Updates(map[string]interface{}{
			"drys": 0, "weed": 0, "pest": 0, "trap": 0,
			"yield": s.Yield, "period": land.Period + 1,
			"planted_at": now, "mature_at": now.Add(time.Duration(s.Again) * time.Minute),
		})
	}
	h.DB.Where("owner_uid = ? AND land_id = ?", uid, land.ID).Delete(&model.FarmSteal{})
	h.farmAddPoint(uid, s.Point)
	if yield > 0 {
		h.bagAdd(uid, s.ID, s.Name, 11, yield)
	}
	h.farmSend(0, uid, "您收获了蔬菜["+s.Name+"]，共["+strconv.Itoa(yield)+"个]。")
	resp.OK(c, gin.H{"msg": "收获[" + s.Name + "] " + strconv.Itoa(yield) + "个，经验+" + strconv.Itoa(s.Point), "amount": yield})
}

// Steal 偷菜（对齐 steal.asp：1~2 个，陷阱可抓奴隶，最后 1 个不能偷）
func (h *FarmHandler) Steal(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req landReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	var land model.FarmLand
	if err := h.DB.Where("id = ? AND user_id <> ?", req.LandID, uid).First(&land).Error; err != nil {
		resp.NotFound(c, "土地不存在")
		return
	}
	owner := land.UserID
	if land.Type != 1 || land.MatureAt == nil || land.MatureAt.After(time.Now()) {
		resp.ParamError(c, "动作太慢，场主已经收菜啦！")
		return
	}
	var tf model.Farm
	if err := h.DB.Where("user_id = ?", owner).First(&tf).Error; err != nil {
		resp.ParamError(c, "农场不存在")
		return
	}
	// 摘取权限 0所有人 1仅好友 4禁止
	if tf.CSteal == 4 {
		resp.Forbidden(c, "本农场禁止摘取！")
		return
	}
	if tf.CSteal == 1 {
		var cnt int64
		h.DB.Model(&model.Friendship{}).Where("user_id = ? AND friend_id = ? AND status = 1", owner, uid).Count(&cnt)
		if cnt == 0 {
			resp.Forbidden(c, "本农场只有好友才可以摘取！")
			return
		}
	}
	// 奴隶两天内不能偷
	var slaveCnt int64
	h.DB.Model(&model.FarmSlave{}).Where("owner_uid = ? AND fid = ? AND created_at > ?",
		owner, uid, time.Now().AddDate(0, 0, -2)).Count(&slaveCnt)
	if slaveCnt > 0 {
		resp.ParamError(c, "您已经成为了对方的奴隶，两天内不能在此农场偷菜哦！")
		return
	}
	// 陷阱判定
	if land.Trap > 0 && land.Trap > rand.Intn(100) {
		var bonus = 1 + rand.Intn(5)
		h.DB.Model(&land).Updates(map[string]interface{}{"trap": 0, "yield": gorm.Expr("yield + ?", bonus)})
		u := h.userBrief(uid)
		var exist int64
		h.DB.Model(&model.FarmSlave{}).Where("owner_uid = ? AND fid = ?", owner, uid).Count(&exist)
		if exist == 0 {
			h.DB.Create(&model.FarmSlave{OwnerUID: owner, FID: uid, Name: u.Nickname})
		}
		h.farmSend(0, owner, u.Nickname+" 掉入了您的陷阱，成为了您的奴隶，同时您的作物获得增产"+strconv.Itoa(bonus)+"。")
		resp.OK(c, gin.H{"msg": "您掉入陷阱，成为了奴隶，两天内不能在此农场偷菜哦！", "trapped": true})
		return
	}
	// 一块地只能偷一次
	var stealCnt int64
	h.DB.Model(&model.FarmSteal{}).Where("owner_uid = ? AND fid = ? AND land_id = ?", owner, uid, land.ID).Count(&stealCnt)
	if stealCnt > 0 {
		resp.ParamError(c, "不要太贪心哦！已经不多啦！")
		return
	}
	if land.Yield <= 1 {
		resp.ParamError(c, "最后一个不能偷哦！")
		return
	}
	stealAmount := 1 + rand.Intn(2)
	if land.Yield == 2 {
		stealAmount = 1
	}
	h.DB.Model(&land).Update("yield", gorm.Expr("yield - ?", stealAmount))
	h.DB.Create(&model.FarmSteal{OwnerUID: owner, FID: uid, LandID: land.ID})
	h.bagAdd(uid, land.SeedID, land.Name, 11, stealAmount)
	u := h.userBrief(uid)
	h.farmAddPoint(uid, stealAmount)
	h.farmSend(0, owner, u.Nickname+" 在你家摘了["+strconv.Itoa(stealAmount)+"个"+land.Name+"]。")
	resp.OK(c, gin.H{"msg": "您成功偷取了[" + land.Name + "]，共[" + strconv.Itoa(stealAmount) + "个]！"})
}

// Sell 卖果实（对齐 sell.asp：单价×数量）
func (h *FarmHandler) Sell(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		BagID uint `json:"bag_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	var bag model.FarmBag
	if err := h.DB.Where("id = ? AND user_id = ? AND dtype = 11", req.BagID, uid).
		First(&bag).Error; err != nil {
		resp.NotFound(c, "仓库里没有这个果实")
		return
	}
	total := 0
	if s := h.seedByID(bag.Oid); s != nil {
		total = s.Price * bag.Amount
	}
	h.DB.Delete(&bag)
	if total > 0 {
		h.DB.Model(&model.User{}).Where("id = ?", uid).Update("coins", gorm.Expr("coins + ?", total))
	}
	h.farmSend(0, uid, "您卖出了["+bag.Name+"]，获得"+strconv.Itoa(total)+"G币。")
	u := h.userBrief(uid)
	resp.OK(c, gin.H{"msg": "卖出[" + bag.Name + "] " + strconv.Itoa(bag.Amount) + "个，获得" + strconv.Itoa(total) + " G币", "coins": u.Coins})
}

// Rank 等级排名（对齐 level.asp：按经验取前 100）
func (h *FarmHandler) Rank(c *gin.Context) {
	var farms []model.Farm
	h.DB.Order("point DESC, level DESC").Limit(100).Find(&farms)
	out := []gin.H{}
	for i, f := range farms {
		u := h.userBrief(f.UserID)
		out = append(out, gin.H{"rank": i + 1, "uid": f.UserID, "nick": u.Nickname,
			"name": f.Name, "level": f.Level, "point": f.Point})
	}
	resp.OK(c, out)
}

// Slaves 我的奴隶列表
func (h *FarmHandler) Slaves(c *gin.Context) {
	uid := middleware.GetUID(c)
	var slaves []model.FarmSlave
	h.DB.Where("owner_uid = ?", uid).Order("id ASC").Find(&slaves)
	out := []gin.H{}
	for _, s := range slaves {
		out = append(out, gin.H{"id": s.ID, "fid": s.FID, "name": s.Name,
			"punish": s.Punish, "appease": s.Appease,
			"time_txt": fmtTimeAgo(time.Since(s.CreatedAt))})
	}
	resp.OK(c, out)
}

type slaveActReq struct {
	Act string `json:"act"` // punish: 0扫大街 1守菜地 2关黑屋 3发神经 / appease: 0逛商场 1住酒店 2泡温泉 3神秘事件
}

// SlaveAct 惩罚/安抚奴隶（对齐 punish/appease.asp：各限 6 次，奖励 1~20 随机金币或经验）
func (h *FarmHandler) SlaveAct(c *gin.Context) {
	uid := middleware.GetUID(c)
	kind := c.Param("kind")
	id, _ := strconv.Atoi(c.Param("id"))
	var slave model.FarmSlave
	if err := h.DB.Where("id = ? AND owner_uid = ?", id, uid).First(&slave).Error; err != nil {
		resp.NotFound(c, "无此奴隶！")
		return
	}
	var acts []string
	cur := 0
	if kind == "punish" {
		acts = []string{"扫大街", "守菜地", "关黑屋", "发神经"}
		cur = slave.Punish
	} else if kind == "appease" {
		acts = []string{"逛商场", "住酒店", "泡温泉", "神秘事件"}
		cur = slave.Appease
	} else {
		resp.ParamError(c, "kind 必须是 punish / appease")
		return
	}
	if cur > 5 {
		resp.ParamError(c, "一个奴隶只能"+map[string]string{"punish": "惩罚", "appease": "安抚"}[kind]+"6次！")
		return
	}
	var req slaveActReq
	_ = c.ShouldBindJSON(&req)
	actIdx := 3
	switch req.Act {
	case "0", "1", "2", "3":
		actIdx, _ = strconv.Atoi(req.Act)
	}
	actName := acts[actIdx]
	col := "punish"
	if kind == "appease" {
		col = "appease"
	}
	h.DB.Model(&slave).Update(col, cur+1)

	award := 1 + rand.Intn(20)
	if kind == "punish" {
		if rand.Intn(2) == 1 {
			h.DB.Model(&model.User{}).Where("id = ?", uid).Update("coins", gorm.Expr("coins + ?", award))
			h.farmSend(0, uid, "您惩罚奴隶["+slave.Name+"]去"+actName+"，获得"+strconv.Itoa(award)+"G币。")
			resp.OK(c, gin.H{"msg": "惩罚[" + slave.Name + "]去" + actName + "，获得" + strconv.Itoa(award) + " G币"})
		} else {
			h.farmAddPoint(uid, award)
			h.farmSend(0, uid, "您惩罚奴隶["+slave.Name+"]去"+actName+"，获得"+strconv.Itoa(award)+"点经验。")
			resp.OK(c, gin.H{"msg": "惩罚[" + slave.Name + "]去" + actName + "，获得" + strconv.Itoa(award) + " 点经验"})
		}
	} else {
		h.farmAddPoint(slave.FID, award)
		h.farmSend(0, uid, "您安抚奴隶["+slave.Name+"]去"+actName+"，TA获得了"+strconv.Itoa(award)+"点经验。")
		h.farmSend(uid, slave.FID, "农场主["+h.userBrief(uid).Nickname+"]安抚您去"+actName+"，您获得"+strconv.Itoa(award)+"点经验。")
		resp.OK(c, gin.H{"msg": "安抚[" + slave.Name + "]去" + actName + "，TA获得" + strconv.Itoa(award) + " 点经验"})
	}
}

// Setting 农场设置（改名 + 摘取权限）
func (h *FarmHandler) Setting(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Name   string `json:"name" binding:"max=30"`
		CSteal *int   `json:"csteal"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	f := h.farmOf(uid)
	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.CSteal != nil {
		if *req.CSteal != 0 && *req.CSteal != 1 && *req.CSteal != 4 {
			resp.ParamError(c, "csteal 必须是 0所有人 1仅好友 4禁止")
			return
		}
		updates["csteal"] = *req.CSteal
	}
	if len(updates) == 0 {
		resp.ParamError(c, "没有要修改的内容")
		return
	}
	h.DB.Model(&model.Farm{}).Where("id = ?", f.ID).Updates(updates)
	resp.OK(c, gin.H{"msg": "设置已保存"})
}

// fmtTimeAgo 简单的"x前"文案（对齐 ComputationTime）
func fmtTimeAgo(d time.Duration) string {
	sec := int(d.Seconds())
	if sec < 0 {
		sec = 0
	}
	if sec < 60 {
		return strconv.Itoa(sec) + "秒"
	}
	if sec < 3600 {
		return strconv.Itoa(sec/60) + "分钟" + strconv.Itoa(sec%60) + "秒"
	}
	return strconv.Itoa(sec/3600) + "小时" + strconv.Itoa((sec/60)%60) + "分钟"
}

// ==================== 管理端：开心农场管理 ====================

// ---------- 种子管理 ----------

func (h *FarmHandler) AdminSeeds(c *gin.Context) {
	var rows []model.FarmSeed
	h.DB.Order("level ASC, id ASC").Find(&rows)
	out := make([]gin.H, 0, len(rows))
	for _, s := range rows {
		out = append(out, gin.H{"id": s.ID, "name": s.Name, "cycle": s.Cycle, "aging": s.Aging,
			"again": s.Again, "yield": s.Yield, "price": s.Price, "seed_price": s.Price * 5 * s.Cycle,
			"point": s.Point, "level": s.Level})
	}
	resp.OK(c, out)
}

type adminFarmSeedReq struct {
	Name  string `json:"name" binding:"required"`
	Cycle int    `json:"cycle"`
	Aging int    `json:"aging"`
	Again int    `json:"again"`
	Yield int    `json:"yield"`
	Price int    `json:"price"`
	Point int    `json:"point"`
	Level int    `json:"level"`
}

func (h *FarmHandler) AdminSeedCreate(c *gin.Context) {
	var req adminFarmSeedReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请填写种子名称")
		return
	}
	var n int64
	h.DB.Model(&model.FarmSeed{}).Where("name = ?", req.Name).Count(&n)
	if n > 0 {
		resp.ParamError(c, "同名种子已存在")
		return
	}
	h.DB.Create(&model.FarmSeed{Name: req.Name, Cycle: iif(req.Cycle < 1, 1, req.Cycle),
		Aging: iif(req.Aging < 1, 15, req.Aging), Again: req.Again,
		Yield: iif(req.Yield < 1, 8, req.Yield), Price: iif(req.Price < 1, 2, req.Price),
		Point: req.Point, Level: iif(req.Level < 1, 1, req.Level)})
	resp.OK(c, nil)
}

func (h *FarmHandler) AdminSeedUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req adminFarmSeedReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请填写种子名称")
		return
	}
	h.DB.Model(&model.FarmSeed{}).Where("id = ?", id).Updates(map[string]interface{}{
		"name": req.Name, "cycle": iif(req.Cycle < 1, 1, req.Cycle), "aging": iif(req.Aging < 1, 15, req.Aging),
		"again": req.Again, "yield": iif(req.Yield < 1, 8, req.Yield), "price": iif(req.Price < 1, 2, req.Price),
		"point": req.Point, "level": iif(req.Level < 1, 1, req.Level)})
	// 同步背包里的种子名
	h.DB.Model(&model.FarmBag{}).Where("oid = ? AND dtype = 1", id).Update("name", req.Name)
	resp.OK(c, nil)
}

func (h *FarmHandler) AdminSeedDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.FarmSeed{}, id)
	h.DB.Where("oid = ? AND dtype = 1", id).Delete(&model.FarmBag{})
	resp.OK(c, gin.H{"msg": "已删除，并清除了用户背包中的该种子"})
}

// ---------- 化肥管理 ----------

func (h *FarmHandler) AdminMucks(c *gin.Context) {
	var rows []model.FarmMuck
	h.DB.Order("speed ASC, id ASC").Find(&rows)
	resp.OK(c, rows)
}

type adminFarmMuckReq struct {
	Name  string `json:"name" binding:"required"`
	Speed int    `json:"speed"`
	Price int    `json:"price"`
}

func (h *FarmHandler) AdminMuckCreate(c *gin.Context) {
	var req adminFarmMuckReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请填写化肥名称")
		return
	}
	var n int64
	h.DB.Model(&model.FarmMuck{}).Where("name = ?", req.Name).Count(&n)
	if n > 0 {
		resp.ParamError(c, "同名化肥已存在")
		return
	}
	h.DB.Create(&model.FarmMuck{Name: req.Name, Speed: iif(req.Speed < 1, 10, req.Speed), Price: iif(req.Price < 0, 0, req.Price)})
	resp.OK(c, nil)
}

func (h *FarmHandler) AdminMuckUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req adminFarmMuckReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请填写化肥名称")
		return
	}
	h.DB.Model(&model.FarmMuck{}).Where("id = ?", id).Updates(map[string]interface{}{
		"name": req.Name, "speed": iif(req.Speed < 1, 10, req.Speed), "price": iif(req.Price < 0, 0, req.Price)})
	h.DB.Model(&model.FarmBag{}).Where("oid = ? AND dtype = 2", id).Update("name", req.Name)
	resp.OK(c, nil)
}

func (h *FarmHandler) AdminMuckDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.FarmMuck{}, id)
	h.DB.Where("oid = ? AND dtype = 2", id).Delete(&model.FarmBag{})
	resp.OK(c, gin.H{"msg": "已删除，并清除了用户背包中的该化肥"})
}

// ---------- 陷阱管理 ----------

func (h *FarmHandler) AdminTraps(c *gin.Context) {
	var rows []model.FarmTrap
	h.DB.Order("rate ASC, id ASC").Find(&rows)
	resp.OK(c, rows)
}

type adminFarmTrapReq struct {
	Name  string `json:"name" binding:"required"`
	Rate  int    `json:"rate"`
	Price int    `json:"price"`
}

func (h *FarmHandler) AdminTrapCreate(c *gin.Context) {
	var req adminFarmTrapReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请填写陷阱名称")
		return
	}
	var n int64
	h.DB.Model(&model.FarmTrap{}).Where("name = ?", req.Name).Count(&n)
	if n > 0 {
		resp.ParamError(c, "同名陷阱已存在")
		return
	}
	h.DB.Create(&model.FarmTrap{Name: req.Name, Rate: clampInt(req.Rate, 1, 100), Price: iif(req.Price < 0, 0, req.Price)})
	resp.OK(c, nil)
}

func (h *FarmHandler) AdminTrapUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req adminFarmTrapReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请填写陷阱名称")
		return
	}
	h.DB.Model(&model.FarmTrap{}).Where("id = ?", id).Updates(map[string]interface{}{
		"name": req.Name, "rate": clampInt(req.Rate, 1, 100), "price": iif(req.Price < 0, 0, req.Price)})
	h.DB.Model(&model.FarmBag{}).Where("oid = ? AND dtype = 3", id).Update("name", req.Name)
	resp.OK(c, nil)
}

func (h *FarmHandler) AdminTrapDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.FarmTrap{}, id)
	h.DB.Where("oid = ? AND dtype = 3", id).Delete(&model.FarmBag{})
	resp.OK(c, gin.H{"msg": "已删除，并清除了用户背包中的该陷阱"})
}

// ---------- 用户农场数据 ----------

// AdminUsers 农场用户列表（分页 + 家园号/昵称搜索）
func (h *FarmHandler) AdminUsers(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	word := c.Query("word")
	sub := h.DB.Model(&model.User{}).Select("id")
	if word != "" {
		sub = sub.Where("username = ? OR nickname LIKE ?", word, "%"+word+"%")
	}
	var total int64
	h.DB.Model(&model.Farm{}).Where("user_id IN (?)", sub).Count(&total)
	var rows []model.Farm
	h.DB.Where("user_id IN (?)", sub).Order("level DESC, point DESC, id ASC").Offset(offset).Limit(size).Find(&rows)
	uidSet := map[uint]bool{}
	for _, f := range rows {
		uidSet[f.UserID] = true
	}
	nick := h.nickMap(uidSet)
	out := make([]gin.H, 0, len(rows))
	for _, f := range rows {
		var landN, bagN, whN, slaveN int64
		h.DB.Model(&model.FarmLand{}).Where("user_id = ?", f.UserID).Count(&landN)
		h.DB.Model(&model.FarmBag{}).Where("user_id = ? AND dtype IN (1,2,3)", f.UserID).Count(&bagN)
		h.DB.Model(&model.FarmBag{}).Where("user_id = ? AND dtype = 11 AND amount > 0", f.UserID).Count(&whN)
		h.DB.Model(&model.FarmSlave{}).Where("owner_uid = ?", f.UserID).Count(&slaveN)
		out = append(out, gin.H{
			"user_id": f.UserID, "nickname": nick[f.UserID], "name": f.Name,
			"level": f.Level, "point": f.Point, "need": (f.Level + 1) * farmPointStep,
			"lands": landN, "mucks": f.Mucks, "bag_kinds": bagN, "wh_kinds": whN, "slave_n": slaveN,
		})
	}
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": out})
}

// AdminUserDetail 用户农场详情：农场 + 菜地 + 背包 + 仓库 + 奴隶
func (h *FarmHandler) AdminUserDetail(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("uid"))
	var u model.User
	if err := h.DB.First(&u, uid).Error; err != nil {
		resp.NotFound(c, "无此用户")
		return
	}
	var f model.Farm
	hasFarm := h.DB.Where("user_id = ?", uid).First(&f).Error == nil
	detail := gin.H{"user_id": uint(uid), "nickname": u.Nickname, "coins": u.Coins, "has_farm": hasFarm}
	if hasFarm {
		detail["farm"] = gin.H{"id": f.ID, "name": f.Name, "level": f.Level,
			"point": f.Point, "need": (f.Level + 1) * farmPointStep, "mucks": f.Mucks, "csteal": f.CSteal}
		// 菜地
		var lands []model.FarmLand
		h.DB.Where("user_id = ?", uid).Order("sort ASC").Find(&lands)
		landOut := make([]gin.H, 0, len(lands))
		now := time.Now()
		for _, l := range lands {
			if l.Type == 1 {
				var tmp = l
				lp, lm := landTime(&tmp)
				if _, st := farmStage(now, lp, lm); st != "已成熟" && l.Drys == 0 {
					h.farmCropStatus(&tmp)
					l = tmp
				}
			}
			landOut = append(landOut, landView(l, now))
		}
		detail["lands"] = landOut
		// 背包（种子/化肥/陷阱）
		var bags []model.FarmBag
		h.DB.Where("user_id = ? AND dtype IN (1,2,3) AND amount > 0", uid).
			Order("dtype ASC, id ASC").Find(&bags)
		bagOut := make([]gin.H, 0, len(bags))
		for _, b := range bags {
			bagOut = append(bagOut, gin.H{"id": b.ID, "oid": b.Oid, "dtype": b.DType, "name": b.Name, "amount": b.Amount})
		}
		detail["bag"] = bagOut
		// 仓库（果实，单价由对应种子决定）
		var whs []model.FarmBag
		h.DB.Where("user_id = ? AND dtype = 11 AND amount > 0", uid).Order("id ASC").Find(&whs)
		whOut := make([]gin.H, 0, len(whs))
		for _, w := range whs {
			price := 0
			if s := h.seedByID(w.Oid); s != nil {
				price = s.Price
			}
			whOut = append(whOut, gin.H{"id": w.ID, "oid": w.Oid, "name": w.Name, "amount": w.Amount, "price": price})
		}
		detail["warehouse"] = whOut
		// 奴隶
		var slaves []model.FarmSlave
		h.DB.Where("owner_uid = ?", uid).Order("id DESC").Find(&slaves)
		detail["slaves"] = slaves
	}
	resp.OK(c, detail)
}

// AdminFarmEdit 编辑农场基础信息（名字/等级/经验/施肥次数/摘取权限/菜地数）
func (h *FarmHandler) AdminFarmEdit(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("uid"))
	var req struct {
		Name   string `json:"name" binding:"max=30"`
		Level  *int   `json:"level"`
		Point  *int   `json:"point"`
		Mucks  *int   `json:"mucks"`
		CSteal *int   `json:"csteal"`
		Lands  *int   `json:"lands"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	var f model.Farm
	if err := h.DB.Where("user_id = ?", uid).First(&f).Error; err != nil {
		resp.ParamError(c, "该用户还未开通农场")
		return
	}
	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Level != nil && *req.Level >= 1 {
		updates["level"] = *req.Level
	}
	if req.Point != nil && *req.Point >= 0 {
		updates["point"] = *req.Point
	}
	if req.Mucks != nil {
		updates["mucks"] = clampInt(*req.Mucks, 0, 99)
	}
	if req.CSteal != nil {
		if *req.CSteal != 0 && *req.CSteal != 1 && *req.CSteal != 4 {
			resp.ParamError(c, "csteal 必须是 0所有人 1仅好友 4禁止")
			return
		}
		updates["csteal"] = *req.CSteal
	}
	if len(updates) > 0 {
		h.DB.Model(&model.Farm{}).Where("id = ?", f.ID).Updates(updates)
	}
	// 调整菜地数（补地或回收空地）
	if req.Lands != nil && *req.Lands >= 1 {
		var lands []model.FarmLand
		h.DB.Where("user_id = ?", uid).Order("sort ASC").Find(&lands)
		cur := len(lands)
		target := clampInt(*req.Lands, 1, 20)
		if target > cur {
			maxSort := 0
			for _, l := range lands {
				if l.Sort > maxSort {
					maxSort = l.Sort
				}
			}
			for i := cur + 1; i <= target; i++ {
				h.DB.Create(&model.FarmLand{UserID: uint(uid), Sort: maxSort + i - cur})
			}
		} else if target < cur {
			// 从后往前回收：只能收回空地
			removed := 0
			for i := len(lands) - 1; i >= 0 && removed < cur-target; i-- {
				if lands[i].Type == 0 {
					h.DB.Delete(&model.FarmLand{}, lands[i].ID)
					removed++
				}
			}
			if removed < cur-target {
				resp.OK(c, gin.H{"msg": "部分菜地有作物无法回收，已回收 " + strconv.Itoa(removed) + " 块"})
				return
			}
		}
	}
	resp.OK(c, gin.H{"msg": "农场信息已更新"})
}

// AdminCoins 调整用户 G币
func (h *FarmHandler) AdminCoins(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("uid"))
	var req struct {
		Coins int `json:"coins"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if req.Coins < 0 {
		req.Coins = 0
	}
	h.DB.Model(&model.User{}).Where("id = ?", uid).Update("coins", req.Coins)
	resp.OK(c, gin.H{"msg": "G币已更新", "coins": req.Coins})
}

// AdminBagSet 修改背包/仓库条目：target=bag(背包) warehouse(仓库)
// 背包按 dtype(1种子/2化肥/3陷阱)+oid 定位；仓库按 id 定位，新增时 name 须为有效果实名
func (h *FarmHandler) AdminBagSet(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("uid"))
	target := c.Param("target")
	if target != "bag" && target != "warehouse" {
		resp.ParamError(c, "未知目标")
		return
	}
	var req struct {
		DType  int    `json:"dtype"`
		Oid    uint   `json:"oid"`
		ID     uint   `json:"id"`
		Name   string `json:"name" binding:"required"`
		Amount int    `json:"amount"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if req.Amount < 0 {
		req.Amount = 0
	}
	if target == "bag" {
		if req.DType != 1 && req.DType != 2 && req.DType != 3 {
			resp.ParamError(c, "dtype 必须是 1种子 2化肥 3陷阱")
			return
		}
		if req.Oid == 0 {
			resp.ParamError(c, "缺少道具 id")
			return
		}
		var bag model.FarmBag
		found := h.DB.Where("user_id = ? AND dtype = ? AND oid = ?", uid, req.DType, req.Oid).
			First(&bag).Error == nil
		if req.Amount == 0 {
			h.DB.Where("user_id = ? AND dtype = ? AND oid = ?", uid, req.DType, req.Oid).Delete(&model.FarmBag{})
		} else if found {
			h.DB.Model(&bag).Update("amount", req.Amount)
		} else {
			h.DB.Create(&model.FarmBag{UserID: uint(uid), Oid: req.Oid, Name: req.Name, DType: req.DType, Amount: req.Amount})
		}
		resp.OK(c, gin.H{"msg": "背包已更新"})
		return
	}
	// 仓库
	if req.ID != 0 {
		var bag model.FarmBag
		if err := h.DB.Where("id = ? AND user_id = ? AND dtype = 11", req.ID, uid).First(&bag).Error; err != nil {
			resp.ParamError(c, "仓库里没有这个果实")
			return
		}
		if req.Amount == 0 {
			h.DB.Delete(&bag)
		} else {
			h.DB.Model(&bag).Update("amount", req.Amount)
		}
		resp.OK(c, gin.H{"msg": "仓库已更新"})
		return
	}
	// 新增仓库条目：名称须匹配种子，单价由种子决定
	var s model.FarmSeed
	if err := h.DB.Where("name = ?", req.Name).First(&s).Error; err != nil {
		resp.ParamError(c, "未找到该果实对应的种子，请核对名称")
		return
	}
	var bag model.FarmBag
	found := h.DB.Where("user_id = ? AND dtype = 11 AND oid = ?", uid, s.ID).First(&bag).Error == nil
	if req.Amount == 0 {
		h.DB.Where("user_id = ? AND dtype = 11 AND oid = ?", uid, s.ID).Delete(&model.FarmBag{})
	} else if found {
		h.DB.Model(&bag).Update("amount", req.Amount)
	} else {
		h.DB.Create(&model.FarmBag{UserID: uint(uid), Oid: s.ID, Name: s.Name, DType: 11, Amount: req.Amount})
	}
	resp.OK(c, gin.H{"msg": "仓库已更新"})
}

// AdminLandClear 收回菜地上的作物（重置为已翻空地）
func (h *FarmHandler) AdminLandClear(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("uid"))
	id, _ := strconv.Atoi(c.Param("id"))
	var land model.FarmLand
	if err := h.DB.Where("id = ? AND user_id = ?", id, uid).First(&land).Error; err != nil {
		resp.NotFound(c, "无此菜地")
		return
	}
	if land.Type == 0 {
		resp.ParamError(c, "该菜地没有作物")
		return
	}
	h.DB.Model(&land).Updates(map[string]interface{}{"type": 0, "plow": 1, "seed_id": 0, "name": "",
		"drys": 0, "weed": 0, "pest": 0, "trap": 0, "yield": 0, "cycle": 0, "period": 0,
		"planted_at": nil, "mature_at": nil})
	resp.OK(c, gin.H{"msg": "菜地已收回为空地"})
}

// ---------- 日志与排行 ----------

// AdminLogs 农场日志：type=msg(消息) steal(偷菜) slave(奴隶)
func (h *FarmHandler) AdminLogs(c *gin.Context) {
	page, offset, size := pageOf(c, 15)
	ty := c.DefaultQuery("type", "msg")
	word := c.Query("word")
	var total int64
	switch ty {
	case "steal":
		q := h.DB.Model(&model.FarmSteal{})
		if word != "" {
			q = q.Where("owner_uid IN (SELECT id FROM users WHERE username = ? OR nickname LIKE ?)", word, "%"+word+"%")
		}
		q.Count(&total)
		var rows []model.FarmSteal
		h.DB.Order("id DESC").Offset(offset).Limit(size).Find(&rows)
		uidSet := map[uint]bool{}
		for _, r := range rows {
			uidSet[r.OwnerUID] = true
			uidSet[r.FID] = true
		}
		nick := h.nickMap(uidSet)
		out := make([]gin.H, 0, len(rows))
		for _, r := range rows {
			out = append(out, gin.H{"id": r.ID, "owner": nick[r.OwnerUID], "thief": nick[r.FID],
				"land_id": r.LandID, "created_at": r.CreatedAt.Format("2006-01-02 15:04:05")})
		}
		resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": out})
	case "slave":
		q := h.DB.Model(&model.FarmSlave{})
		if word != "" {
			q = q.Where("owner_uid IN (SELECT id FROM users WHERE username = ? OR nickname LIKE ?)", word, "%"+word+"%")
		}
		q.Count(&total)
		var rows []model.FarmSlave
		h.DB.Order("id DESC").Offset(offset).Limit(size).Find(&rows)
		uidSet := map[uint]bool{}
		for _, r := range rows {
			uidSet[r.OwnerUID] = true
			uidSet[r.FID] = true
		}
		nick := h.nickMap(uidSet)
		out := make([]gin.H, 0, len(rows))
		for _, r := range rows {
			out = append(out, gin.H{"id": r.ID, "owner": nick[r.OwnerUID], "slave": nick[r.FID], "name": r.Name,
				"punish": r.Punish, "appease": r.Appease, "created_at": r.CreatedAt.Format("2006-01-02 15:04:05")})
		}
		resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": out})
	default:
		q := h.DB.Model(&model.FarmMsg{})
		if word != "" {
			q = q.Where("user_id IN (SELECT id FROM users WHERE username = ? OR nickname LIKE ?) OR content LIKE ?", word, "%"+word+"%", "%"+word+"%")
		}
		q.Count(&total)
		var rows []model.FarmMsg
		h.DB.Order("id DESC").Offset(offset).Limit(size).Find(&rows)
		uidSet := map[uint]bool{}
		for _, r := range rows {
			if r.FID > 0 {
				uidSet[r.FID] = true
			}
		}
		nick := h.nickMap(uidSet)
		out := make([]gin.H, 0, len(rows))
		for _, r := range rows {
			from := "系统"
			if r.FID > 0 {
				from = nick[r.FID]
			}
			out = append(out, gin.H{"id": r.ID, "to_uid": r.UserID, "from": from,
				"content": r.Content, "status": r.Status, "created_at": r.CreatedAt.Format("2006-01-02 15:04:05")})
		}
		resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": out})
	}
}

// AdminRank 农场等级排行
func (h *FarmHandler) AdminRank(c *gin.Context) {
	page, offset, size := pageOf(c, 20)
	var total int64
	h.DB.Model(&model.Farm{}).Count(&total)
	var rows []model.Farm
	h.DB.Order("level DESC, point DESC, id ASC").Offset(offset).Limit(size).Find(&rows)
	uidSet := map[uint]bool{}
	for _, f := range rows {
		uidSet[f.UserID] = true
	}
	nick := h.nickMap(uidSet)
	out := make([]gin.H, 0, len(rows))
	for i, f := range rows {
		var landN int64
		h.DB.Model(&model.FarmLand{}).Where("user_id = ?", f.UserID).Count(&landN)
		out = append(out, gin.H{"rank": offset + i + 1, "user_id": f.UserID, "nickname": nick[f.UserID],
			"name": f.Name, "level": f.Level, "point": f.Point, "lands": landN})
	}
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": out})
}

// ---------- 通用小工具 ----------

func iif(cond bool, a, b int) int {
	if cond {
		return a
	}
	return b
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
