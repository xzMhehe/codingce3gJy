package handler

import (
	"fmt"
	"math/rand"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 二战风云 城市迁移（复刻原版 city/cityHallMove.html）
//
// ★ 第十二轮改造：迁城区域从「10 个人造矩形」改为**按世界地图的七大洲**。
//
//	用户规则：「地图是仿制二战的世界地图，所有的土地、城市都要加个所属洲」，
//	         迁城时「选洲迁城池」——迁城计划选洲，沿海迁城计划选洲。
//
//	旧实现把 50~450 的坐标域硬切成「西欧/东欧/西亚/东亚/南亚/北非/南非/澳洲/北美/南美」
//	10 个矩形，与 ezfy_geo.go 里的七大洲几何**完全对不上**：选了「北美」可能落在
//	南美洲的陆地上，城市列表里显示的所属洲和迁城时选的区域互相矛盾。
//	现在统一以 ezfyContinentOf() 的洲 ID 为准，区域名就是洲名。
//
// 三种迁城方式（与三种道具一一对应）：
//
//	low  迁城计划     —— 选洲，落该洲内的随机空平原（陆地城市）
//	high 高级迁城计划 —— 指定 x/y，必须是未被占领的平原
//	sea  沿海迁城计划 —— 选洲 或 指定 x/y，必须是未被占领的沿海平原
//
// ★ 第十二轮起：迁城**消耗对应道具**，不再扣黄金（道具本身可用黄金或钻石在商城购买）。
const ezfyMoveCityGoldCost = 200000 // 保留常量：道具的默认黄金定价参考值

// 三种迁城道具的 CfgId（见 seed/ezfy.go seedEzfyMoveItems）
const (
	ezfyItemMoveCity    = 20 // 迁城计划     ItemType 16
	ezfyItemMoveCityAdv = 21 // 高级迁城计划 ItemType 17
	ezfyItemMoveCitySea = 22 // 沿海迁城计划 ItemType 18
)

// 道具 ItemType（与 ezfy_cfg_item.item_type 对应）
const (
	ezfyItemTypeMoveCity    = 16
	ezfyItemTypeMoveCityAdv = 17
	ezfyItemTypeMoveCitySea = 18
	// ★ 2026-09-26 三种加速道具（种子 ezfyEzfyCfgItem 的 4~9 号）：
	//   4 建筑加速30分 / 5 建筑加速2小时 → ItemType 3
	//   6 训练加速30分 / 7 训练加速2小时 → ItemType 4
	//   8 科技加速30分 / 9 科技加速2小时 → ItemType 5
	ezfyItemTypeBuildSpeed = 3
	ezfyItemTypeTrainSpeed = 4
	ezfyItemTypeTechSpeed  = 5
)

// ezfyMoveKinds 三种迁城方式 → 所需道具 cfgId + 展示名
type ezfyMoveKind struct {
	Code    string // low / high / sea
	ItemId  int    // 需要的道具 cfgId
	Label   string // 道具名
	PickSea bool   // 落点是否必须是沿海平原（海城）
}

var ezfyMoveKinds = []ezfyMoveKind{
	{"low", ezfyItemMoveCity, "迁城计划", false},
	{"high", ezfyItemMoveCityAdv, "高级迁城计划", false},
	{"sea", ezfyItemMoveCitySea, "沿海迁城计划", true},
}

func ezfyMoveKindOf(code string) *ezfyMoveKind {
	for i := range ezfyMoveKinds {
		if ezfyMoveKinds[i].Code == code {
			return &ezfyMoveKinds[i]
		}
	}
	return nil
}

// ezfyMoveArea 迁城区域 = 一个洲
//
// ★ 第十二轮：ID 就是洲 ID（ezfyContinentOf 的返回值），Name 就是洲名。
// 不再使用人造矩形边界 —— 落点判定直接调 ezfyContinentOf()。
type ezfyMoveArea struct {
	ID   int    // 洲 ID：1欧洲 2亚洲 3非洲 4北美洲 5南美洲 6大洋洲 7南极洲
	Name string // 洲名
}

// ezfyMoveAreas 可迁入的洲列表（顺序按常见认知：欧洲优先，内测都聚在欧洲）
var ezfyMoveAreas = []ezfyMoveArea{
	{1, "欧洲"},
	{2, "亚洲"},
	{3, "非洲"},
	{4, "北美洲"},
	{5, "南美洲"},
	{6, "大洋洲"},
	{7, "南极洲"},
}

// ezfyDefaultMoveContinent 新玩家建城 / 批量迁移的默认洲 = 欧洲
//
// 用户规则：「新玩家 默认 建城市也是默认欧洲城市」「把所有线上、线下玩家的城市
// 都迁移到欧洲坐标 …… 尽量都在附近（内测玩家之间征服、侦查、掠夺）」。
const ezfyDefaultMoveContinent = 1 // 欧洲

func ezfyMoveAreaOf(id int) *ezfyMoveArea {
	for i := range ezfyMoveAreas {
		if ezfyMoveAreas[i].ID == id {
			return &ezfyMoveAreas[i]
		}
	}
	return nil
}

// MoveInfo GET /games/ezfy/city/move —— 迁城页数据（洲列表/三种道具持有量/当前坐标）
func (h *EzfyHandler) MoveInfo(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	city := h.getOrCreateCity(uid)
	h.refreshCity(uid, &city)

	areas := make([]gin.H, 0, len(ezfyMoveAreas))
	for _, a := range ezfyMoveAreas {
		areas = append(areas, gin.H{"id": a.ID, "name": a.Name})
	}
	// 三种道具的持有数量 + 商城定价（前端直接展示「持有 x 个 / 商城 xx 黄金」）
	items := make([]gin.H, 0, len(ezfyMoveKinds))
	for _, k := range ezfyMoveKinds {
		cfg := ezfyCfg.item(k.ItemId)
		it := gin.H{"code": k.Code, "item_id": k.ItemId, "label": k.Label, "count": h.itemCount(uid, k.ItemId)}
		if cfg != nil {
			it["price_gold"] = cfg.PriceGold
			it["price_diamond"] = cfg.PriceDiamond
			it["description"] = cfg.Description
		}
		items = append(items, it)
	}
	resp.OK(c, gin.H{
		"areas": areas, "gold_cost": ezfyMoveCityGoldCost, "gold": city.Gold,
		"items":             items,
		"default_continent": ezfyDefaultMoveContinent,
		"city": gin.H{"id": city.ID, "name": city.Name, "x": city.X, "y": city.Y,
			"continent": ezfyRegionName(city.X, city.Y)},
	})
}

// MoveCity POST /games/ezfy/city/move  {city_id, type: low|high|sea, continent_id, area_id, x, y}
//
// ★ 第十二轮：消耗对应道具（迁城计划 / 高级迁城计划 / 沿海迁城计划），不再扣黄金。
func (h *EzfyHandler) MoveCity(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		CityId int64  `json:"city_id"`
		Type   string `json:"type"`
		// 选洲迁城（low / sea 都支持）
		ContinentId int `json:"continent_id"`
		// 兼容旧前端字段名
		AreaId int `json:"area_id"`
		X      int `json:"x"`
		Y      int `json:"y"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	h.cfgs()
	kind := ezfyMoveKindOf(req.Type)
	if kind == nil {
		resp.ParamError(c, "迁城方式不正确")
		return
	}
	city := h.bodyCity(uid, req.CityId)
	if city == nil {
		resp.ParamError(c, "城市不存在")
		return
	}
	h.refreshCity(uid, city)

	// 先校验道具，避免「扣了资源才发现道具不够」
	if have := h.itemCount(uid, kind.ItemId); have < 1 {
		resp.ParamError(c, fmt.Sprintf("背包里没有【%s】，请先到商城购买", kind.Label))
		return
	}

	// 选洲：continent_id 优先，回落到旧字段 area_id，再回落默认洲（欧洲）
	continent := req.ContinentId
	if continent == 0 {
		continent = req.AreaId
	}

	var tx, ty int
	switch kind.Code {
	case "low", "sea":
		// 选洲迁城：给了坐标就走坐标，否则按洲找空位
		if req.X > 0 && req.Y > 0 {
			if msg := h.checkMoveTarget(req.X, req.Y, kind.PickSea); msg != "" {
				resp.ParamError(c, msg)
				return
			}
			tx, ty = req.X, req.Y
			break
		}
		if continent == 0 {
			continent = ezfyDefaultMoveContinent
		}
		if ezfyMoveAreaOf(continent) == nil {
			resp.ParamError(c, "请选择要迁入的洲")
			return
		}
		x, y, ok := h.findFreePosInContinent(continent, kind.PickSea)
		if !ok {
			resp.ParamError(c, fmt.Sprintf("%s暂时没有可用空地, 请换个洲试试", ezfyContinentNames[continent]))
			return
		}
		tx, ty = x, y
	case "high":
		if req.X <= 0 || req.Y <= 0 {
			resp.ParamError(c, "请输入要迁移的目标坐标")
			return
		}
		if msg := h.checkMoveTarget(req.X, req.Y, false); msg != "" {
			resp.ParamError(c, msg)
			return
		}
		tx, ty = req.X, req.Y
	}

	oldX, oldY := city.X, city.Y
	oldRegion := ezfyRegionName(oldX, oldY)
	// ★ 消耗道具（不是扣黄金）
	h.consumeItem(uid, kind.ItemId)
	city.X, city.Y = tx, ty
	h.saveCityRes(city)
	h.DB.Model(&model.EzfyCity{}).Where("id = ?", city.ID).Updates(map[string]interface{}{"x": tx, "y": ty})
	// 旧坐标上的「玩家城」地图区域记录要清掉，否则地图还显示那里有城
	h.DB.Where("x = ? AND y = ? AND area_type = ?", oldX, oldY, 3).Delete(&model.EzfyMapArea{})

	newRegion := ezfyRegionName(tx, ty)
	h.addReport(uid, 6, "城市迁移完成",
		fmt.Sprintf("消耗【%s】×1, 城市[%s]已从%s(%d,%d)迁移到%s(%d,%d)。\n附属野地不会随城迁移, 请重新占领。",
			kind.Label, city.Name, oldRegion, oldX, oldY, newRegion, tx, ty))
	resp.OK(c, gin.H{
		"msg": fmt.Sprintf("迁城成功, 新坐标(%d,%d) %s", tx, ty, newRegion),
		"x":   tx, "y": ty, "continent": newRegion,
	})
}

// findFreePosInContinent 在指定洲内随机找一个「未被占领的空位」
//
// needCoastal=true → 必须是沿海平原（海城）；false → 平原或沿海平原都可以（陆地城市）。
//
// ★ 用户要求「尽量都在附近」：先做「优先贴着已有城市」的采样 ——
// 内测玩家互相之间要能侦查/掠夺，离太远就没得打。
// 具体做法：前 60% 的尝试在「已有城市周围 ±6 格」的圈里取样，
// 失败再退回全洲均匀采样，保证人多时也一定放得下。
func (h *EzfyHandler) findFreePosInContinent(continent int, needCoastal bool) (int, int, bool) {
	return h.findFreePosInContinentExcept(continent, needCoastal, nil)
}

// findFreePosInContinentExcept 同 findFreePosInContinent，但可排除一批「已被占用」的坐标。
//
// ★ 为什么需要 excluded（踩过的坑，别删）：
//
//	批量迁移时计划是一次性算出来的，但每迁走一座城就多占一个格子，
//	后面那些城「预先算好」的落点很可能正好撞上前面刚挪过去的城，
//	表现为大批 `该坐标已有城市` 失败（线上 147 城实测失败 71 座）。
//	所以执行阶段必须**逐城实时重算**，并把本次已决定的落点排除掉。
func (h *EzfyHandler) findFreePosInContinentExcept(continent int, needCoastal bool, excluded map[[2]int]bool) (int, int, bool) {
	return h.findFreePosInContinentExceptC(continent, needCoastal, excluded, nil)
}

// moveCtx 批量迁城时复用的「全城坐标」上下文。
//
// ★ 2026-09-21 性能修复：findFreePosInContinentExcept 原来每次调用都要
//
//	① `SELECT x,y FROM ezfy_city`（建 occupied）
//	② `SELECT * FROM ezfy_city`（取 anchors，整表全字段）
//	批量迁 147 座城 = 294 次全表扫描 → IO 飙升。
//	现在批量场景把这两份数据在**批次开始时读一次**，逐城复用，
//	每城只额外维护「本批已落点」的 excluded 集合。
type moveCtx struct {
	occupied map[[2]int]bool // 全服已占坐标
	anchors  []model.EzfyCity
}

// newMoveCtx 批次开始时调一次，读齐 occupied 与 anchors。
func (h *EzfyHandler) newMoveCtx() *moveCtx {
	ctx := &moveCtx{occupied: map[[2]int]bool{}}
	var cities []model.EzfyCity
	h.DB.Model(&model.EzfyCity{}).Order("id ASC").Find(&cities)
	for _, c := range cities {
		ctx.occupied[[2]int{c.X, c.Y}] = true
		ctx.anchors = append(ctx.anchors, c)
	}
	return ctx
}

// MoveCtx 导出的批次上下文别名，供 CLI 运维工具（ezfymigrate）使用。
type MoveCtx = moveCtx

// NewMoveCtx 供 CLI 工具调用：批次开始时建一次，逐城传给 EzfyMoveOneCityC /
// EzfyMoveOneCityCoastalC，避免「每城都全表扫一遍城市表」。
func (h *EzfyHandler) NewMoveCtx() *MoveCtx { return h.newMoveCtx() }

// markOccupied 落点后把新坐标记进 occupied，供同批次后续城市避让。
func (ctx *moveCtx) markOccupied(x, y int) {
	if ctx != nil && ctx.occupied != nil {
		ctx.occupied[[2]int{x, y}] = true
	}
}

// findFreePosInContinentExceptC 带上下文的版本：ctx 非 nil 时复用其 occupied/anchors，
// 不再查库；ctx 为 nil 时行为与原来完全一致（自己查库，供单次迁城调用）。
func (h *EzfyHandler) findFreePosInContinentExceptC(continent int, needCoastal bool,
	excluded map[[2]int]bool, ctx *moveCtx) (int, int, bool) {
	// ★ 「已占用坐标」读进内存。
	//
	//	原来每个候选点都发一条 `SELECT COUNT(*) ... WHERE x=? AND y=?`，
	//	在「沿海平原」这种**稀有目标**上要试几万次 → DB 被拖垮。
	//	改成内存判定；批量场景由 ctx 复用同一份快照，避免逐城全表扫。
	occupied := map[[2]int]bool{}
	var anchors []model.EzfyCity
	if ctx != nil {
		occupied, anchors = ctx.occupied, ctx.anchors
	} else {
		var cities []model.EzfyCity
		h.DB.Model(&model.EzfyCity{}).Order("id ASC").Find(&cities)
		for _, c := range cities {
			occupied[[2]int{c.X, c.Y}] = true
			anchors = append(anchors, c)
		}
	}
	ok := func(x, y int) bool {
		if x < 1 || y < 1 || x >= ezfyWorldSize || y >= ezfyWorldSize {
			return false
		}
		if ezfyContinentOf(x, y) != continent {
			return false
		}
		if excluded != nil && excluded[[2]int{x, y}] {
			return false
		}
		if occupied[[2]int{x, y}] {
			return false
		}
		t := ezfyTerrainEx(x, y)
		if needCoastal {
			if t != ezfyTerrainCoastalPlain {
				return false
			}
		} else if t != 1 && t != ezfyTerrainCoastalPlain {
			return false
		}
		return true
	}

	// ① 贴着已有城市采样（同一片区域）
	if len(anchors) > 0 {
		for i := 0; i < 4000; i++ {
			a := anchors[rand.Intn(len(anchors))]
			x := a.X + rand.Intn(13) - 6
			y := a.Y + rand.Intn(13) - 6
			if ok(x, y) {
				return x, y, true
			}
		}
	}
	// ② 需要沿海平原时：走**缓存的沿海平原索引**取点。
	//
	//	★ 2026-09-21 线上性能事故：这里原来是「双 for 枚举整个 500×500 世界」，
	//	  每个格子都要跑 ezfyTerrainEx（内部判洲 + 8 邻域判海洋，合计十几次
	//	  「点在多边形内」判定）→ 单次请求上百万次形状运算，1 核直接打满，
	//	  批量迁城逐城调用更是雪崩。
	//	  现在改为：全图枚举只在进程内跑一次（coastalPlainCandidates 带缓存），
	//	  之后每座城都是「过滤已占用 + 随机取点」，O(n) 且无形状运算。
	if needCoastal {
		if x, y, found := pickCoastalPos(continent, occupied, excluded); found {
			return x, y, true
		}
		return 0, 0, false
	}
	// ③ 目标洲内均匀采样兜底（陆地目标多得多，随机足够）
	for i := 0; i < 20000; i++ {
		x := rand.Intn(ezfyWorldSize)
		y := rand.Intn(ezfyWorldSize)
		if ok(x, y) {
			return x, y, true
		}
	}
	return 0, 0, false
}

// checkMoveTarget 校验指定坐标能否迁城；needCoastal=true 时必须是沿海平原
func (h *EzfyHandler) checkMoveTarget(x, y int, needCoastal bool) string {
	if x < 1 || x > ezfyWorldSize-1 || y < 1 || y > ezfyWorldSize-1 {
		return fmt.Sprintf("坐标需在 1~%d 之间", ezfyWorldSize-1)
	}
	t := ezfyTerrainEx(x, y)
	if t != 1 && t != ezfyTerrainCoastalPlain {
		return "只能迁移到平原或沿海平原"
	}
	var n int64
	h.DB.Model(&model.EzfyCity{}).Where("x = ? AND y = ?", x, y).Count(&n)
	if n > 0 {
		return "该坐标已有城市"
	}
	isCoastal := t == ezfyTerrainCoastalPlain
	if needCoastal && !isCoastal {
		return "沿海迁城只能迁移到沿海平原"
	}
	if !needCoastal && isCoastal {
		return "该坐标是沿海平原, 请使用沿海迁城计划"
	}
	return ""
}

// ============ 批量迁移（CLI 工具 ezfymigrate 与内置迁移共用） ============

// EzfyMovePlanItem 单座城市的迁移计划（预演 / 执行都以它为单位）
type EzfyMovePlanItem struct {
	CityId    uint
	UserID    uint
	Name      string
	OldX      int
	OldY      int
	OldRegion string
	IsSea     bool
	NewX      int
	NewY      int
	NewRegion string
	Reason    string // 无法迁移时的原因
}

// EzfyBuildMovePlan 给一批城市生成迁移计划（不落库）
//
// continent：目标洲 ID（0 → 默认欧洲）
// onlySea / onlyLand：只处理海城 / 只处理陆城（都 false = 全部）
// 返回的计划里 NewX/NewY 为 0 表示找不到落点，Reason 说明原因。
func (h *EzfyHandler) EzfyBuildMovePlan(cities []model.EzfyCity, continent int, onlySea, onlyLand bool) []EzfyMovePlanItem {
	if continent == 0 {
		continent = ezfyDefaultMoveContinent
	}
	out := make([]EzfyMovePlanItem, 0, len(cities))
	for i := range cities {
		ct := cities[i]
		item := EzfyMovePlanItem{
			CityId: ct.ID, UserID: ct.UserID, Name: ct.Name,
			OldX: ct.X, OldY: ct.Y, OldRegion: ezfyRegionName(ct.X, ct.Y),
			IsSea: h.isSeaCity(&ct),
		}
		if onlySea && !item.IsSea {
			item.Reason = "非海城，跳过"
			out = append(out, item)
			continue
		}
		if onlyLand && item.IsSea {
			item.Reason = "非陆城，跳过"
			out = append(out, item)
			continue
		}
		// 已经在目标洲的城不用动
		if ezfyContinentOf(ct.X, ct.Y) == continent {
			item.NewX, item.NewY = ct.X, ct.Y
			item.NewRegion = item.OldRegion
			item.Reason = "已在目标洲，无需迁移"
			out = append(out, item)
			continue
		}
		x, y, ok := h.findFreePosInContinent(continent, item.IsSea)
		if !ok {
			item.Reason = "目标洲内找不到可用空地"
			out = append(out, item)
			continue
		}
		item.NewX, item.NewY = x, y
		item.NewRegion = ezfyRegionName(x, y)
		out = append(out, item)
	}
	return out
}

// EzfyApplyMove 真正执行一次城市迁移（落库）
//
// ★ 与玩家侧 MoveCity 的关键差别（别把两者合并）：
//
//	① **不扣道具** —— 这是运营/迁移脚本专用，不是玩家行为；
//	② **不做「海城 / 内陆」互斥校验** —— 玩家迁城分三种道具，规则必须严；
//	   但批量迁移只求「把城搬进目标洲、互相靠近」，陆城落在沿海平原
//	   完全无害（反而以后能练海军），没必要为此把城卡住不迁。
//	   只保留「必须是无城市的平原/沿海平原、坐标在界内」这两条硬约束，
//	   保证结果和玩家自己迁城一样**合法**。
func (h *EzfyHandler) EzfyApplyMove(cityId uint, x, y int) string {
	var ct model.EzfyCity
	if err := h.DB.First(&ct, cityId).Error; err != nil {
		return "城市不存在"
	}
	if x < 1 || y < 1 || x >= ezfyWorldSize || y >= ezfyWorldSize {
		return fmt.Sprintf("坐标需在 1~%d 之间", ezfyWorldSize-1)
	}
	t := ezfyTerrainEx(x, y)
	if t != 1 && t != ezfyTerrainCoastalPlain {
		return "只能迁移到平原或沿海平原"
	}
	var n int64
	h.DB.Model(&model.EzfyCity{}).Where("x = ? AND y = ?", x, y).Count(&n)
	if n > 0 {
		return "该坐标已有城市"
	}
	oldX, oldY := ct.X, ct.Y
	if err := h.DB.Model(&model.EzfyCity{}).Where("id = ?", ct.ID).
		Updates(map[string]interface{}{"x": x, "y": y}).Error; err != nil {
		return "写入失败: " + err.Error()
	}
	h.DB.Where("x = ? AND y = ? AND area_type = ?", oldX, oldY, 3).Delete(&model.EzfyMapArea{})
	return ""
}

// EzfyMoveOneCity 把一座城迁到目标洲的实时空闲处（**逐城重算落点**）
//
// 这是批量迁移的**唯一正确用法**：不要预先给所有城算好坐标再挨个落库，
// 因为每迁一座城就多占一格，预计算的落点会自相撞（见
// findFreePosInContinentExcept 的注释）。正确姿势是「迁一座 → 重算下一座」。
//
// excluded：本次批次里已经决定/占用的坐标，避免同一批内互相撞车。
//
// 返回实际落点 x, y 与错误信息（"" = 成功）。已在目标洲的城原样返回。
func (h *EzfyHandler) EzfyMoveOneCity(cityId uint, continent int, excluded map[[2]int]bool) (int, int, string) {
	return h.EzfyMoveOneCityC(cityId, continent, excluded, nil)
}

// EzfyMoveOneCityC 带 moveCtx 的版本：批量迁移时请用 newMoveCtx() 建一次 ctx，
// 逐城传入 —— 这样「全城坐标」只读一次，不再每城全表扫描。
func (h *EzfyHandler) EzfyMoveOneCityC(cityId uint, continent int, excluded map[[2]int]bool, ctx *moveCtx) (int, int, string) {
	if continent == 0 {
		continent = ezfyDefaultMoveContinent
	}
	var ct model.EzfyCity
	if err := h.DB.First(&ct, cityId).Error; err != nil {
		return 0, 0, "城市不存在"
	}
	// 已经在目标洲：不折腾，原样保留（但记进 excluded，防止别的城撞上来）
	if ezfyContinentOf(ct.X, ct.Y) == continent {
		if excluded != nil {
			excluded[[2]int{ct.X, ct.Y}] = true
		}
		return ct.X, ct.Y, ""
	}
	isSea := h.isSeaCity(&ct)
	x, y, ok := h.findFreePosInContinentExceptC(continent, isSea, excluded, ctx)
	if !ok {
		return 0, 0, "目标洲内找不到可用空地"
	}
	if msg := h.EzfyApplyMove(ct.ID, x, y); msg != "" {
		return 0, 0, msg
	}
	if excluded != nil {
		excluded[[2]int{x, y}] = true
	}
	ctx.markOccupied(x, y)
	return x, y, ""
}

// ============ 沿海迁城计划（运维用：把「航海协会城」迁回沿海平原） ============
//
// ★ 线上事故复盘（2026-09-20）：
//
//	玩家侧迁城是用 isSeaCity() 判断「海城 / 陆城」的，而 isSeaCity() 看的是
//	**当前坐标的地形**。批量迁城时海城被搬到普通平原后，isSeaCity() 立刻变成
//	false，再迁一次也只会落回普通平原 —— 结果「海城变成了陆地城」，
//	航海协会形同虚设（只有沿海平原上的城市才能建/用航海协会）。
//
// 修法：不看当前地形，直接按「这座城里有没有航海协会」来判定它应该是海城，
// 落点强制为**沿海平原**（即「沿海迁城计划」的规则）。
//
// EzfyTerrainAt 导出地形值，供 CLI 运维工具（ezfymigrate --stats）统计展示用。
//
//	1 = 平原, 2..7 = 其他陆地地形(草原/森林/盆地/丘陵/沼泽/山地),
//	8 = 海洋, 9 = 沿海平原(ezfyTerrainCoastalPlain, 只有它能建航海协会)
func EzfyTerrainAt(x, y int) int { return ezfyTerrainEx(x, y) }

// EzfyCoastalPlainValue 沿海平原的地形值（= ezfyTerrainCoastalPlain），给 CLI 比较用
const EzfyCoastalPlainValue = ezfyTerrainCoastalPlain

// EzfyCoastalNeedMoveAt 该坐标上的城是否需要「沿海迁城计划」重新安置。
//
// ★ 用户明确的条件：**只看「脚下是不是沿海平原」** ——
//
//	有航海协会、但现在站在普通平原（陆地城市）上的城才需要迁；
//	已经在沿海平原上的城保持原位（哪怕它不在目标洲，也不再折腾）。
//
// 导出给 CLI 工具 ezfymigrate 预演时统计用。continent 参数保留但**不再参与判定**。
func EzfyCoastalNeedMoveAt(x, y, continent int) bool {
	_ = continent
	return ezfyTerrainEx(x, y) != ezfyTerrainCoastalPlain
}

// EzfyCitiesWithBuilding 取「建有指定建筑(level>0)」的全部城池，按 id 升序。
func (h *EzfyHandler) EzfyCitiesWithBuilding(buildingId int) []model.EzfyCity {
	var ids []int64
	h.DB.Model(&model.EzfyCityBuilding{}).
		Where("building_id = ? AND level > 0", buildingId).
		Distinct().Pluck("city_id", &ids)
	if len(ids) == 0 {
		return nil
	}
	var cities []model.EzfyCity
	h.DB.Where("id IN ?", ids).Order("id ASC").Find(&cities)
	return cities
}

// EzfyMoveOneCityCoastal 把一座城迁到目标洲的**沿海平原**空位（沿海迁城计划）。
//
// 与 EzfyMoveOneCity 的差别：
//   - 落点强制为沿海平原（needCoastal = true），不看当前地形；
//   - 「已经在沿海平原上」直接跳过 —— 只要它不在沿海平原上就继续迁，
//     否则「海城变陆城」永远修不回来（这正是线上那个 bug）。
//
// ★ 用户规则：「不够就给移动到亚洲的沿海平原」——
//
//	目标洲（默认欧洲）没有空闲沿海平原时，自动退到 fallback 洲（默认亚洲）再找。
//	线上实测：欧洲沿海平原总共只有 55 格，被 50 座城占满后极易「放不下」。
//
// fallback <= 0 表示不启用备用洲。返回实际落点 x, y 与错误信息（"" = 成功）。
func (h *EzfyHandler) EzfyMoveOneCityCoastal(cityId uint, continent, fallback int, excluded map[[2]int]bool) (int, int, string) {
	return h.EzfyMoveOneCityCoastalC(cityId, continent, fallback, excluded, nil)
}

// EzfyMoveOneCityCoastalC 带 moveCtx 的版本（批量场景请复用同一个 ctx）。
func (h *EzfyHandler) EzfyMoveOneCityCoastalC(cityId uint, continent, fallback int,
	excluded map[[2]int]bool, ctx *moveCtx) (int, int, string) {
	if continent == 0 {
		continent = ezfyDefaultMoveContinent
	}
	var ct model.EzfyCity
	if err := h.DB.First(&ct, cityId).Error; err != nil {
		return 0, 0, "城市不存在"
	}
	// 已经在沿海平原上 → 不用动（哪怕不在目标洲，也别折腾玩家的城）
	if ezfyTerrainEx(ct.X, ct.Y) == ezfyTerrainCoastalPlain {
		if excluded != nil {
			excluded[[2]int{ct.X, ct.Y}] = true
		}
		return ct.X, ct.Y, ""
	}
	x, y, ok := h.findFreePosInContinentExceptC(continent, true, excluded, ctx)
	if !ok && fallback > 0 && fallback != continent {
		x, y, ok = h.findFreePosInContinentExceptC(fallback, true, excluded, ctx)
	}
	if !ok {
		return 0, 0, "目标洲(及备用洲)内找不到可用沿海平原"
	}
	if msg := h.EzfyApplyMove(ct.ID, x, y); msg != "" {
		return 0, 0, msg
	}
	if excluded != nil {
		excluded[[2]int{x, y}] = true
	}
	ctx.markOccupied(x, y)
	return x, y, ""
}
