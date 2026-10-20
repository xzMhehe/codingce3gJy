package ezfy

import (
	"encoding/json"
	"fmt"
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

// 二战风云管理端（玩家/数据/流水/系统），复刻 stzb-fk GM 能力，表均带 ezfy_ 前缀

// AdminEzfyPlayers 玩家列表（word=昵称/家园号/用户ID 模糊）
func (h *EzfyAdmin) AdminEzfyPlayers(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	word := strings.TrimSpace(c.Query("word"))
	q := h.DB.Model(&model.EzfyProfile{})
	// ★★ 2026-10-17 人机模块：玩家管理**默认只出真人**（is_bot = 0）。
	//   人机的数据只在「人机管理」里出现 —— 运营看到的注册数/在线数/新增玩家不再被几百个人机污染。
	//   ⚠️ 下面 word 的两个 OR 条件必须自己加括号：GORM 串联 Where 只按 AND 拼、不补括号，
	//      否则会变成 `is_bot=0 AND user_id=? OR game_uid=?` → 把人机也带出来。
	q = q.Where("is_bot = 0")
	if word != "" {
		// ★★ 2026-10-15 用户规则：「二战管理系统里所有家园号条件查询，全部换成游戏号查询」。
		//   原来这里拿 word 去**家园库** `users.username`（= 家园号）反查 uid 再匹配 ——
		//   家园号会随「转靓号」变，而玩家在游戏里看到、报给客服的是**游戏号**
		//   （`ezfy_profile.game_uid`：建档时 = 家园ID，之后与 user_id 解耦）。
		//   现在：数字 → 用户ID 或 游戏号；非数字 → 玩家昵称（另兼容家园昵称精确）。
		if uid, err := strconv.Atoi(word); err == nil {
			q = q.Where("(user_id = ? OR game_uid = ?)", uid, uid)
		} else {
			var wu model.User
			h.home().Select("id").Where("nickname = ?", word).First(&wu)
			if wu.ID > 0 {
				q = q.Where("(nickname LIKE ? OR user_id = ?)", "%"+word+"%", wu.ID)
			} else {
				q = q.Where("nickname LIKE ?", "%"+word+"%")
			}
		}
	}
	var total int64
	q.Count(&total)
	var rows []model.EzfyProfile
	q.Order("prestige DESC, id ASC").Offset(offset).Limit(size).Find(&rows)
	type rowOut struct {
		model.EzfyProfile
		HomeNick    string `json:"home_nick"`
		HomeNum     string `json:"home_num"`
		CityCount   int64  `json:"city_count"`
		CampName    string `json:"camp_name"`
		RankName    string `json:"rank_name"`
		LastLoginAt int64  `json:"last_login_at"` // ★ 2026-10-12 最后登陆时间(ms)
	}
	out := []rowOut{}
	// ★★ 2026-10-15 性能修复（用户反馈「玩家信息管理 分页查询严重卡顿，这才 200 多个人，
	//   切分页大小时更严重」）。
	//
	//   原来**每一行**都要打两条库：
	//     ① `h.home().First(&u, p.UserID)` —— 取家园昵称 / 家园号（跨库，可能跨 WAN）
	//     ② `COUNT(*) FROM ezfy_city WHERE user_id = ?` —— 数城池
	//   → size=100 就是 **200 条往返**；线上 1核1G + RDS 必然卡到几十秒，
	//     而且分页越大越慢 —— 正是用户描述的现象（不是数据量大，是查询条数被行数放大）。
	//
	//   现在三样附属数据（最后登陆 / 家园账号 / 城池数）**全部按 `IN ?` 批量取**，
	//   整页固定 4 条 SQL：①分页主查询 ②最后登陆 ③家园账号 ④城池数(GROUP BY user_id)。
	//   ★ 口径不变：家园昵称/家园号仍是「按 user_id 查家园库」，城池数仍是该玩家的城数。
	uids := make([]uint, 0, len(rows))
	for _, p := range rows {
		uids = append(uids, p.UserID)
	}
	homeOf := map[uint]model.User{} // 家园账号（id → 昵称/家园号）
	lastLoginOf := map[uint]int64{} // 最后登陆时间(ms)
	cityCountOf := map[uint]int64{} // 城池数
	if len(uids) > 0 {
		var users []model.User
		h.home().Select("id, username, nickname").Where("id IN ?", uids).Find(&users)
		for _, u := range users {
			homeOf[u.ID] = u
		}
		var accs []model.EzfyAccount
		h.DB.Select("game_id, last_login_at").Where("game_id IN ?", uids).Find(&accs)
		for _, a := range accs {
			lastLoginOf[a.GameID] = a.LastLoginAt
		}
		var sums []struct {
			UserID uint
			Cnt    int64
		}
		h.DB.Model(&model.EzfyCity{}).
			Select("user_id, COUNT(*) AS cnt").
			Where("user_id IN ?", uids).Group("user_id").Scan(&sums)
		for _, s := range sums {
			cityCountOf[s.UserID] = s.Cnt
		}
	}
	for _, p := range rows {
		u := homeOf[p.UserID]
		out = append(out, rowOut{EzfyProfile: p, HomeNick: u.Nickname, HomeNum: u.Username,
			CityCount: cityCountOf[p.UserID], CampName: ezfyCampName(p.Camp),
			RankName:    ezfyRankNameAt(ezfyProfileRank(&p)),
			LastLoginAt: lastLoginOf[p.UserID]})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

func ezfyCampName(camp int) string {
	if camp == 2 {
		return "轴心国"
	}
	return "同盟国"
}

// AdminEzfyPlayerDetail GET /admin/ezfy-players/:id/detail —— 玩家详情（档案/城池/背包/军团/军官/最近出征）
//
// ★★ 2026-10-17 卡顿优化（用户反馈「玩家详情打开有些卡顿」）：
//
//	原来 11 条 SQL **全部串行**，线上二战库是远端 RDS（单次 20~40ms）→ 打开详情固定 400ms+。
//	其中最慢的一条是「按战报反查名将获取时间」：`content LIKE '%俘虏敌将:x%'` 要扫该玩家全部战报
//	（前缀通配符用不了索引）。现在：
//	  ① 独立查询**分波并行**：第一波 6 条（档案/城池/背包/军团/出征/家园昵称）
//	     → 第二波 3 条（兵力求和/军官/军团名）→ 第三波（名将配置，必要时才查战报）；
//	  ② 战报反查只针对 **source 未标注(0) 的名将** —— source 有枚举后，军校招募/野地俘虏/
//	     抢玩家/管理端发放都能直接判定，正常情况下这条最慢的 SQL **根本不执行**。
func (h *EzfyAdmin) AdminEzfyPlayerDetail(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	// ===== 第一波并行：6 条互不依赖的查询（1 RTT 而不是 6 RTT）=====
	var p model.EzfyProfile
	var pErr error
	var cities []model.EzfyCity
	var bag []model.EzfyItem
	var members []model.EzfyCorpsMember
	var orders []model.EzfyOrder
	var u model.User
	homeOK := false
	// ★ 2026-10-18 玩家详情背包 tab 要看「待开宝箱」个数：一并查 ezfy_user_chest
	var chests []model.EzfyUserChest

	var wg sync.WaitGroup
	wg.Add(7)
	go func() { defer wg.Done(); pErr = h.DB.Where("user_id = ?", id).First(&p).Error }()
	go func() { defer wg.Done(); h.DB.Where("user_id = ?", id).Find(&cities) }()
	go func() { defer wg.Done(); h.DB.Where("user_id = ?", id).Find(&bag) }()
	go func() { defer wg.Done(); h.DB.Where("user_id = ?", id).Find(&members) }()
	go func() { defer wg.Done(); h.DB.Where("user_id = ?", id).Order("id DESC").Limit(20).Find(&orders) }()
	go func() { defer wg.Done(); h.DB.Where("user_id = ?", id).Find(&chests) }()
	go func() {
		defer wg.Done()
		if err := h.home().First(&u, id).Error; err == nil {
			homeOK = true
		}
	}()
	wg.Wait()
	if pErr != nil {
		resp.NotFound(c, "玩家不存在")
		return
	}
	homeNick, homeNum := "", ""
	if homeOK {
		homeNick, homeNum = u.Nickname, u.Username
	}
	cityIDs := make([]int64, 0, len(cities))
	for _, ct := range cities {
		cityIDs = append(cityIDs, int64(ct.ID))
	}

	// ===== 第二波并行：依赖城池/军团成员的三条 =====
	var troopOf map[uint]int64
	var officers []model.EzfyOfficer
	corpsNameOf := map[uint]string{}
	var wg2 sync.WaitGroup
	wg2.Add(3)
	go func() { // 城池总兵力：一次 GROUP BY，避免每城一条 SQL（N+1）
		defer wg2.Done()
		if len(cityIDs) == 0 {
			return
		}
		var sums []struct {
			CityID uint
			Total  int64
		}
		h.DB.Model(&model.EzfyCityTroop{}).
			Select("city_id, SUM(count) AS total").
			Where("city_id IN ?", cityIDs).Group("city_id").Scan(&sums)
		m := make(map[uint]int64, len(sums))
		for _, s := range sums {
			m[s.CityID] = s.Total
		}
		troopOf = m
	}()
	go func() { // 军官：玩家名下所有城市的全部军官（含出征中/俘虏）
		defer wg2.Done()
		if len(cityIDs) == 0 {
			return
		}
		h.DB.Where("city_id IN ?", cityIDs).Order("star DESC, level DESC, id ASC").Find(&officers)
	}()
	go func() { // 军团名：批量一次（原来每个成员一条 SQL）
		defer wg2.Done()
		if len(members) == 0 {
			return
		}
		cids := make([]uint, 0, len(members))
		for _, m := range members {
			cids = append(cids, m.CorpsId)
		}
		var cps []model.EzfyCorps
		h.DB.Select("id, name").Where("id IN ?", cids).Find(&cps)
		m := make(map[uint]string, len(cps))
		for _, cp := range cps {
			m[cp.ID] = cp.Name
		}
		corpsNameOf = m
	}()
	wg2.Wait()

	// 城池视图
	type cityOut struct {
		model.EzfyCity
		TroopTotal int64 `json:"troop_total"` // 城池总兵力（ezfy_city_troop.count 求和）
	}
	citiesViews := make([]cityOut, 0, len(cities))
	for _, ct := range cities {
		citiesViews = append(citiesViews, cityOut{EzfyCity: ct, TroopTotal: troopOf[ct.ID]})
	}

	// 背包：道具名走进程内配置缓存（0 条 SQL）
	type bagOut struct {
		model.EzfyItem
		ItemName string `json:"item_name"`
	}
	ezfyCfg.load(h.DB)
	bagViews := make([]bagOut, 0, len(bag))
	for _, it := range bag {
		name := ""
		if cfg := ezfyCfg.item(it.CfgId); cfg != nil {
			name = cfg.Name
		}
		bagViews = append(bagViews, bagOut{EzfyItem: it, ItemName: name})
	}

	// 军团视图
	type corpsOut struct {
		model.EzfyCorpsMember
		CorpsName string `json:"corps_name"`
	}
	corpsViews := make([]corpsOut, 0, len(members))
	for _, m := range members {
		corpsViews = append(corpsViews, corpsOut{EzfyCorpsMember: m, CorpsName: corpsNameOf[m.CorpsId]})
	}

	// ===== 军官视图（名将原名 / 获取方式与时间）=====
	type officerOut struct {
		model.EzfyOfficer
		CityName     string    `json:"city_name"`
		PositionName string    `json:"position_name"`
		StatusName   string    `json:"status_name"`
		TypeName     string    `json:"type_name"`    // ★ 普通 / 名将（只有 kind=2 名将池的才算名将）
		GeneralName  string    `json:"general_name"` // ★ 原名将名称（名将池回填，供管理端点击查看）
		GeneralStar  int       `json:"general_star"` // ★ 原名将星级
		GetWay       string    `json:"get_way"`      // ★ 获取方式（活动野地俘虏/战俘营/管理端发放）
		GetTime      time.Time `json:"get_time"`     // ★ 获取时间（按战报反查；无战报回落到军官更新时间）
	}
	officersViews := make([]officerOut, 0, len(officers))
	if len(officers) > 0 {
		cityNames := make(map[int64]string, len(cities))
		for _, ct := range cities {
			cityNames[int64(ct.ID)] = ct.Name
		}
		// 原名将配置：收集 general_id 一次查配置表，避免 N+1
		gNameOf, gStarOf, gKindOf := map[int]string{}, map[int]int{}, map[int]int{}
		{
			ids := make([]int, 0)
			seen := map[int]bool{}
			for _, o := range officers {
				if o.GeneralId > 0 && !seen[o.GeneralId] {
					seen[o.GeneralId] = true
					ids = append(ids, o.GeneralId)
				}
			}
			if len(ids) > 0 {
				var gs []model.EzfyCfgGeneral
				h.DB.Where("id IN ?", ids).Find(&gs)
				for _, g := range gs {
					gNameOf[g.ID] = g.Name
					gStarOf[g.ID] = g.Star
					gKindOf[g.ID] = g.Kind
				}
			}
		}
		// 获取方式/时间：只有「名将 + source 未标注(0)」才需要按战报反查
		// （source 有枚举后，军校招募/野地俘虏/抢玩家/管理端发放都能直接判定 → 这条慢 SQL 通常不执行）
		getWayOf, getTimeOf := map[uint]string{}, map[uint]time.Time{}
		{
			names := make([]string, 0)
			for _, o := range officers {
				if o.GeneralId > 0 && gKindOf[o.GeneralId] == 2 && o.Source == 0 {
					names = append(names, o.Name)
				}
			}
			if len(names) > 0 {
				conds := make([]string, 0, len(names))
				args := make([]interface{}, 0, len(names)+1)
				args = append(args, p.UserID)
				for _, n := range names {
					conds = append(conds, "content LIKE ?")
					args = append(args, "%俘虏敌将:"+n+"%")
				}
				var rows []struct {
					Content   string
					CreatedAt time.Time
				}
				h.DB.Table("ezfy_report").
					Select("content, created_at").
					Where("user_id = ? AND ("+strings.Join(conds, " OR ")+")", args...).
					Order("created_at ASC").
					Scan(&rows)
				for _, r := range rows {
					for i := range officers {
						o := &officers[i]
						if o.GeneralId <= 0 || gKindOf[o.GeneralId] != 2 {
							continue
						}
						if _, ok := getTimeOf[o.ID]; ok {
							continue
						}
						if strings.Contains(r.Content, "俘虏敌将:"+o.Name) {
							getWayOf[o.ID] = "活动野地俘虏"
							getTimeOf[o.ID] = r.CreatedAt
						}
					}
				}
			}
		}
		for i := range officers {
			o := officers[i]
			isGeneral := o.GeneralId > 0 && gKindOf[o.GeneralId] == 2
			typeName := "普通"
			if isGeneral {
				typeName = "名将"
			}
			// 获取方式/时间（仅名将展示；普通军官留空）
			getWay, getTime := "", o.UpdateTime
			generalName, generalStar := "", 0
			if isGeneral {
				generalName, generalStar = gNameOf[o.GeneralId], gStarOf[o.GeneralId]
				if w, ok := getWayOf[o.ID]; ok {
					getWay, getTime = w, getTimeOf[o.ID]
				} else {
					// ★ 2026-10-06 优先按 source 枚举标注：原来只认「俘虏敌将」战报 +
					//   IsCaptive 兜底，PvP 抢将收编后（source=4, IsCaptive=0）被误判成
					//   「管理端发放」（线上反馈：35806363 抢小号军官却显示管理端发放）。
					switch o.Source {
					case model.EzfyOfficerSourcePvp:
						getWay = "抢玩家获取"
					case model.EzfyOfficerSourceWildland:
						if o.IsCaptive == 1 {
							getWay = "活动野地俘虏(战俘营)"
						} else {
							getWay = "活动野地俘虏"
						}
					case model.EzfyOfficerSourceRecruit:
						getWay = "军校招募"
					case model.EzfyOfficerSourceGrant:
						getWay = "管理端发放"
					default: // source 未标注(历史老数据，10-05 加列前的军官)
						if o.IsCaptive == 1 {
							getWay = "活动野地俘虏(战俘营)"
						} else {
							// ★ 2026-10-06 不能兜底成「管理端发放」——source=0 只是「未标注」，
							//   实质可能是野地任务/打小号抢来的名将（线上全服反馈），
							//   等全局回填 SQL 标注后再看正确来源，这里如实显示未标注。
							getWay = "未知(老数据)"
						}
					}
				}
			}
			officersViews = append(officersViews, officerOut{
				EzfyOfficer:  o,
				CityName:     cityNames[o.CityId],
				PositionName: ezfyPositionName(o.Position),
				StatusName:   ezfyOfficerStatusName(&o),
				TypeName:     typeName,
				GeneralName:  generalName,
				GeneralStar:  generalStar,
				GetWay:       getWay,
				GetTime:      getTime,
			})
		}
	}

	// ===== 待开宝箱视图（ezfy_user_chest：管理端/活动发放，玩家背包→宝箱里自己开）=====
	type chestOut struct {
		model.EzfyUserChest
		ChestName string `json:"chest_name"` // 宝箱配置名（ezfy_cfg_chest）
	}
	chestsViews := make([]chestOut, 0, len(chests))
	if len(chests) > 0 {
		chestNameOf := map[int]string{}
		var cfgChests []model.EzfyCfgChest
		h.DB.Select("id, name").Find(&cfgChests)
		for _, cc := range cfgChests {
			chestNameOf[cc.ID] = cc.Name
		}
		for _, ch := range chests {
			chestsViews = append(chestsViews, chestOut{
				EzfyUserChest: ch,
				ChestName:     chestNameOf[ch.ChestId],
			})
		}
	}

	resp.OK(c, gin.H{"player": p, "home_nick": homeNick, "home_num": homeNum,
		"cities": citiesViews, "bag": bagViews, "chests": chestsViews, "corps": corpsViews, "orders": orders,
		"officers":  officersViews,
		"camp_name": ezfyCampName(p.Camp), "rank_name": ezfyRankNameAt(ezfyProfileRank(&p))})
}

// AdminEzfyPlayerKick POST /admin/ezfy-players/:id/kick —— 踢下线（强制该玩家重新登录二战）
//
// ★ 2026-10-12 用户要求：「二战管理系统 · 玩家信息管理 · 操作加个踢下线功能」。
//
// 二战 token 是**无状态 JWT**，服务端没有会话可删，所以用「被踢时间戳」作废旧 token：
// 写 `ezfy_account.kick_at = now`，中间件按 `token.iat <= kick_at → 该 token 已作废` 拒绝
// （见 middleware.EzfyJWTAuth）。玩家重新登录会签发新 token（iat 晚于 kick_at）→ 自动恢复，
// 不需要额外的「解封」操作。
//
// id = 家园号(game_id)。从未登录过二战的玩家没有 ezfy_account 行 → 直接提示，不算失败。
func (h *EzfyAdmin) AdminEzfyPlayerKick(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		resp.ParamError(c, "玩家不正确")
		return
	}
	now := time.Now().UnixMilli()
	res := h.DB.Model(&model.EzfyAccount{}).Where("game_id = ?", id).Update("kick_at", now)
	if res.Error != nil {
		resp.ServerError(c, res.Error)
		return
	}
	if res.RowsAffected == 0 {
		resp.NotFound(c, "该玩家还没有二战登录账号（从未登录过二战）")
		return
	}
	// 立即刷新进程内缓存 → 本机即时生效（不必等 60s TTL）
	middleware.SetEzfyKickAt(uint(id), now)
	resp.OK(c, gin.H{"msg": "已踢下线，该玩家需要重新登录二战", "kick_at": now})
}

// AdminEzfyPlayerUpdate 修改玩家（昵称/声望/阵营）
func (h *EzfyAdmin) AdminEzfyPlayerUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var p model.EzfyProfile
	if err := h.DB.Where("user_id = ?", id).First(&p).Error; err != nil {
		resp.NotFound(c, "玩家不存在")
		return
	}
	var in struct {
		Nickname *string `json:"nickname"`
		Prestige *int    `json:"prestige"`
		Camp     *int    `json:"camp"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	updates := map[string]interface{}{}
	if in.Nickname != nil && strings.TrimSpace(*in.Nickname) != "" {
		updates["nickname"] = trimStr(strings.TrimSpace(*in.Nickname), 20)
	}
	if in.Prestige != nil && *in.Prestige >= 0 {
		updates["prestige"] = *in.Prestige
	}
	if in.Camp != nil && (*in.Camp == 1 || *in.Camp == 2) {
		updates["camp"] = *in.Camp
	}
	if len(updates) > 0 {
		h.DB.Model(&model.EzfyProfile{}).Where("id = ?", p.ID).Updates(updates)
	}
	resp.OK(c, gin.H{"msg": "修改成功"})
}

// AdminEzfyGrant 发放资源/道具（资源入主城并按仓储上限截断，道具入背包）
// AdminEzfyGrantOfficer POST /admin/ezfy-players/:id/grant-officer  {general_id}
// 名将只能由管理端发放(): 直接把 cfg_general 里的名将变成该玩家的军官
func (h *EzfyAdmin) AdminEzfyGrantOfficer(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var in struct {
		GeneralID int `json:"general_id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.GeneralID <= 0 {
		resp.ParamError(c, "请选择名将")
		return
	}
	msg, errMsg := h.ezfyGrantGeneral(uint(id), in.GeneralID)
	if errMsg != "" {
		resp.ParamError(c, errMsg)
		return
	}
	resp.OK(c, gin.H{"msg": msg})
}

func (h *EzfyAdmin) AdminEzfyGrant(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var p model.EzfyProfile
	if err := h.DB.Where("user_id = ?", id).First(&p).Error; err != nil {
		resp.NotFound(c, "玩家不存在")
		return
	}
	var in struct {
		Gold  int64 `json:"gold"`
		Food  int64 `json:"food"`
		Steel int64 `json:"steel"`
		Oil   int64 `json:"oil"`
		Rare  int64 `json:"rare"`
		Items []struct {
			CfgID int `json:"cfg_id"`
			Count int `json:"count"`
		} `json:"items"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	ez := &EzfyHandler{DB: h.DB, HomeDB: h.HomeDB}
	msg := "已发放"
	if in.Gold != 0 || in.Food != 0 || in.Steel != 0 || in.Oil != 0 || in.Rare != 0 {
		// GM 发放不按仓储上限截断（玩家要多少给多少，可以超上限堆着）
		// ★ 2026-10-09 改为返回「实际入库城市 + 实际到账量」：被资源最大值封顶（或扣到 0）时
		//   管理员能一眼看出「填了 100 万只进去 30 万」，而不是只看到「已发放」。
		city, got := ez.giveResourcesNoCap(p.UserID, in.Food, in.Steel, in.Oil, in.Rare, in.Gold)
		msg += " 入「" + city.Name + "」"
		add := func(label string, want, real int64) {
			if want == 0 {
				return
			}
			msg += fmt.Sprintf(" %s%+d", label, real)
			if real != want {
				msg += fmt.Sprintf("(填%+d, 受资源上限/下限截断)", want)
			}
		}
		add("粮食", in.Food, got[0])
		add("钢铁", in.Steel, got[1])
		add("石油", in.Oil, got[2])
		add("稀矿", in.Rare, got[3])
		add("黄金", in.Gold, got[4])
	}
	for _, it := range in.Items {
		if it.CfgID <= 0 || it.Count <= 0 {
			continue
		}
		var cfg model.EzfyCfgItem
		if err := h.DB.First(&cfg, it.CfgID).Error; err != nil {
			resp.ParamError(c, "道具不存在："+strconv.Itoa(it.CfgID))
			return
		}
		ez.addItem(p.UserID, it.CfgID, it.Count)
		msg += fmt.Sprintf(" 【%s】×%d", cfg.Name, it.Count)
	}
	// 站内通知
	h.DB.Create(&model.EzfyNotice{UserId: p.UserID, Title: "管理员发放",
		Content: strings.TrimSpace(msg) + "，请查收。"})
	resp.OK(c, gin.H{"msg": msg})
}

// AdminEzfyItemGrantPlayers 道具发放目标玩家搜索（昵称 LIKE / 游戏ID 精确，供发放对话框选择）
func (h *EzfyAdmin) AdminEzfyItemGrantPlayers(c *gin.Context) {
	word := strings.TrimSpace(c.Query("word"))
	if word == "" {
		resp.ParamError(c, "请输入玩家昵称或游戏ID")
		return
	}
	q := h.DB.Model(&model.EzfyProfile{})
	if uid, err := strconv.Atoi(word); err == nil {
		// ★ 2026-10-15 弹窗 placeholder 写的是「玩家昵称 / 游戏ID」→ 数字也要匹配**游戏号**
		//   （game_uid 转靓号后与 user_id 解耦，只匹配 user_id 会让管理员按玩家报的游戏号搜不到）。
		q = q.Where("user_id = ? OR game_uid = ?", uid, uid)
	} else {
		q = q.Where("nickname LIKE ?", "%"+word+"%")
	}
	var rows []model.EzfyProfile
	q.Order("prestige DESC, id ASC").Limit(20).Find(&rows)
	type out struct {
		UserID   uint   `json:"user_id"`
		Nickname string `json:"nickname"`
		HomeNum  string `json:"home_num"`
	}
	// ★ 2026-10-15 同 AdminEzfyPlayers：家园号**一次 IN 批量取**，别再每行一次 First
	//   （这个对话框最多 20 行 → 原来 20 条跨库往返，线上明显卡）。
	gids := make([]uint, 0, len(rows))
	for _, p := range rows {
		gids = append(gids, p.UserID)
	}
	homeNumOf := map[uint]string{}
	if len(gids) > 0 {
		var us []model.User
		h.home().Select("id, username").Where("id IN ?", gids).Find(&us)
		for _, u := range us {
			homeNumOf[u.ID] = u.Username
		}
	}
	list := []out{}
	for _, p := range rows {
		list = append(list, out{UserID: p.UserID, Nickname: p.Nickname, HomeNum: homeNumOf[p.UserID]})
	}
	resp.OK(c, gin.H{"list": list})
}

// AdminEzfyItemGrantOptions GET /admin/ezfy-item-grant/options —— 「发放道具」弹窗的道具下拉选项。
// ★ 2026-09-27 修复「发放道具不全 / 检索迁城无匹配」：
//
//	旧实现复用 /admin/ezfy-data/items 分页接口，pageOf() 会把 size>100 钳制回落 10，
//	导致下拉永远只拿到前 10 件道具（迁城计划等 13 号以后的道具选不到）。
//	这里直接全量返回 ezfy_cfg_item（id + name），不走分页钳制。
func (h *EzfyAdmin) AdminEzfyItemGrantOptions(c *gin.Context) {
	var items []model.EzfyCfgItem
	h.DB.Order("id ASC").Find(&items)
	opts := make([]gin.H, 0, len(items))
	for i := range items {
		it := items[i]
		// ★ 2026-09-27 为爱发电卡：只在专属「为爱发电卡维护」页发放，不进通用发放道具下拉
		if isLoveCardItem(it.ItemType) {
			continue
		}
		opts = append(opts, gin.H{"id": it.ID, "name": it.Name})
	}
	resp.OK(c, gin.H{"list": opts, "total": len(opts)})
}

// AdminEzfyItemGrant POST /admin/ezfy-item-grant {player, items[]} —— 数据管理→道具配置的「发放道具」。
// player 支持玩家昵称或游戏ID(user_id)；道具入背包并站内通知。
func (h *EzfyAdmin) AdminEzfyItemGrant(c *gin.Context) {
	var in struct {
		Player string `json:"player"`
		Items  []struct {
			CfgID int `json:"cfg_id"`
			Count int `json:"count"`
		} `json:"items"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	player := strings.TrimSpace(in.Player)
	if player == "" {
		resp.ParamError(c, "请填写玩家昵称或游戏ID")
		return
	}
	if len(in.Items) == 0 {
		resp.ParamError(c, "请添加要发放的道具")
		return
	}
	var p model.EzfyProfile
	if uid, err := strconv.Atoi(player); err == nil {
		if err := h.DB.Where("user_id = ?", uid).First(&p).Error; err != nil {
			resp.NotFound(c, "未找到该游戏ID对应的玩家，请先搜索确认")
			return
		}
	} else {
		if err := h.DB.Where("nickname = ?", player).First(&p).Error; err != nil {
			resp.NotFound(c, "未找到该昵称对应的玩家，请先搜索确认")
			return
		}
	}
	ez := &EzfyHandler{DB: h.DB, HomeDB: h.HomeDB}
	items := ""
	attaches := make([]model.EzfyMailAttach, 0, len(in.Items))
	for _, it := range in.Items {
		if it.CfgID <= 0 || it.Count <= 0 {
			continue
		}
		// ★ 2026-10-10 为爱发电卡：已彻底脱离道具体系（不在 ezfy_cfg_item），
		//   发放即生效（创建激活记录），先按卡片目录识别，命中即不再查道具表。
		//   ★ 2026-10-18 改走邮件后，卡片仍**直接生效**（没有背包概念，走邮件领取反而绕），
		//     邮件正文里注明「已直接激活」。
		if cat := loveCardByID(it.CfgID); cat != nil {
			ez.createLoveCard(p.UserID, cat, it.Count)
			items = items + fmt.Sprintf(" 【%s】×%d（已直接激活）", cat.Name, it.Count)
			continue
		}
		var cfg model.EzfyCfgItem
		if err := h.DB.First(&cfg, it.CfgID).Error; err != nil {
			resp.ParamError(c, "道具不存在："+strconv.Itoa(it.CfgID))
			return
		}
		// ★ 2026-09-27 为爱发电卡：发放即生效，不走背包（创建激活记录），其余道具照常入背包。
		if isLoveCardItem(cfg.ItemType) {
			if cat := loveCardByType(cfg.ItemType); cat != nil {
				ez.createLoveCard(p.UserID, cat, it.Count)
				items += fmt.Sprintf(" 【%s】×%d（已直接激活）", cat.Name, it.Count)
				continue
			}
		}
		// ★★ 2026-10-18 用户需求「数据管理里面发放道具也是走邮件吧，不然用户都不知道获得了什么」：
		//   不再直接 addItem 进背包，改成**系统邮件 + 附件**，玩家在邮箱点「领取」才到账。
		attaches = append(attaches, model.EzfyMailAttach{
			Kind: ezfyAttachItem, CfgID: it.CfgID, Num: int64(it.Count), Name: cfg.Name,
		})
		items += fmt.Sprintf(" 【%s】×%d", cfg.Name, it.Count)
	}
	if items == "" && len(attaches) == 0 {
		resp.ParamError(c, "请添加要发放的道具")
		return
	}
	if len(attaches) > 0 {
		content := "系统发放道具：" + strings.TrimSpace(items) +
			fmt.Sprintf("，请在 %d 天内到邮箱领取，过期作废。", ezfyMailAttachExpireDays)
		if _, err := ezfyMailSendBatch(h.DB, []ezfyMailDraft{{
			ReceiverID: p.UserID, Subject: "系统发放道具", Content: content, Attaches: attaches,
		}}, ezfyMailAttachExpireDays); err != nil {
			resp.ServerError(c, fmt.Errorf("发放失败：%w", err))
			return
		}
	}
	resp.OK(c, gin.H{"msg": fmt.Sprintf("已通过系统邮件发放给玩家「%s」(ID:%d)%s，玩家需在邮箱领取",
		p.Nickname, p.UserID, items)})
}

// ★★ 2026-10-18 「发放道具」改走系统邮件后，原来的 ezfyGrantItemToPlayers（直接写背包）
//   已无调用方，整体删除 —— 避免留一条与邮件发放**口径不同**的旁路，
//   以后有人想「补发」时误用它（那条路玩家收不到任何通知/邮件）。
//   批量写背包的原子 upsert 写法在邮件领取里（ezfy_mail_attach.go::MailClaim）继续沿用。

// AdminEzfyItemGrantTargets GET /admin/ezfy-item-grant/targets —— 全员发放的「目标人数」预览。
//
// 单独做一个小接口而不是复用 /admin/ezfy-players：后者的权限点是 module:ezfyPlayers，
// 而「发放道具」在 module:ezfyData 下 —— 只有数据管理权限的管理员调它会 403。
func (h *EzfyAdmin) AdminEzfyItemGrantTargets(c *gin.Context) {
	var n int64
	h.DB.Model(&model.EzfyProfile{}).Count(&n)
	resp.OK(c, gin.H{"total": n})
}

// AdminEzfyItemGrantAll POST /admin/ezfy-item-grant/all {items[]} —— 「发放道具」的**全员发放**。
//
// ★ 2026-10-13 用户需求：发放道具原来一次只能给一个玩家，补一个全员发放。
//
//	发放对象 = 全部二战玩家档案（ezfy_profile 的所有 user_id）。
//	道具入背包；同时给每人写一条站内通知（与单人发放同一套文案）。
//	⚠️ 为爱发电卡不在「发放道具」下拉里（见 AdminEzfyItemGrantOptions 已过滤），
//	   但为了兼容仍然按「发放即生效」处理，避免有人直接构造 cfg_id 调接口时发错。
func (h *EzfyAdmin) AdminEzfyItemGrantAll(c *gin.Context) {
	var in struct {
		Items []struct {
			CfgID int `json:"cfg_id"`
			Count int `json:"count"`
		} `json:"items"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if len(in.Items) == 0 {
		resp.ParamError(c, "请添加要发放的道具")
		return
	}
	var uids []uint
	if err := h.DB.Model(&model.EzfyProfile{}).Pluck("user_id", &uids).Error; err != nil {
		resp.ServerError(c, fmt.Errorf("读取玩家列表失败：%w", err))
		return
	}
	if len(uids) == 0 {
		resp.NotFound(c, "还没有任何二战玩家档案，无法全员发放")
		return
	}
	ez := &EzfyHandler{DB: h.DB, HomeDB: h.HomeDB}
	items := ""
	attaches := make([]model.EzfyMailAttach, 0, len(in.Items))
	for _, it := range in.Items {
		if it.CfgID <= 0 || it.Count <= 0 {
			continue
		}
		// 为爱发电卡：发放即生效（创建激活记录），不走背包
		if cat := loveCardByID(it.CfgID); cat != nil {
			for _, uid := range uids {
				ez.createLoveCard(uid, cat, it.Count)
			}
			items += fmt.Sprintf(" 【%s】×%d（已直接激活）", cat.Name, it.Count)
			continue
		}
		var cfg model.EzfyCfgItem
		if err := h.DB.First(&cfg, it.CfgID).Error; err != nil {
			resp.ParamError(c, "道具不存在："+strconv.Itoa(it.CfgID))
			return
		}
		if isLoveCardItem(cfg.ItemType) {
			if cat := loveCardByType(cfg.ItemType); cat != nil {
				for _, uid := range uids {
					ez.createLoveCard(uid, cat, it.Count)
				}
				items += fmt.Sprintf(" 【%s】×%d（已直接激活）", cat.Name, it.Count)
				continue
			}
		}
		// ★★ 2026-10-18 全员发放同样改走邮件（不再直接写背包）：每人一封系统邮件 + 同一份附件。
		attaches = append(attaches, model.EzfyMailAttach{
			Kind: ezfyAttachItem, CfgID: it.CfgID, Num: int64(it.Count), Name: cfg.Name,
		})
		items += fmt.Sprintf(" 【%s】×%d", cfg.Name, it.Count)
	}
	if items == "" && len(attaches) == 0 {
		resp.ParamError(c, "请添加要发放的道具")
		return
	}
	if len(attaches) > 0 {
		content := "系统发放道具：" + strings.TrimSpace(items) +
			fmt.Sprintf("，请在 %d 天内到邮箱领取，过期作废。", ezfyMailAttachExpireDays)
		drafts := make([]ezfyMailDraft, 0, len(uids))
		for _, uid := range uids {
			drafts = append(drafts, ezfyMailDraft{
				ReceiverID: uid, Subject: "系统发放道具", Content: content, Attaches: attaches,
			})
		}
		if _, err := ezfyMailSendBatch(h.DB, drafts, ezfyMailAttachExpireDays); err != nil {
			resp.ServerError(c, fmt.Errorf("发放失败：%w", err))
			return
		}
	}
	resp.OK(c, gin.H{"msg": fmt.Sprintf("已给全部 %d 名玩家发送系统邮件%s，玩家需在邮箱领取", len(uids), items), "players": len(uids)})
}

// AdminEzfyPlayerDelete 删除玩家（**物理删除该玩家在二战库的全部数据**）。
//
// ★★ 2026-10-15 用户规则：「操作里面的删除 是物理操作，把该玩家的所有数据都删除掉」。
//
//	老实现只清了 16 张表，登录账号/军官/装备/交易行/邮件/好友/风云榜/宣战/占领/军团
//	等一大批全留着（删完号后台还能看到残留）→ 改成走 `ezfyPurgePlayerData`
//	的**声明式清单 + 事务**（清单见 ezfy_admin_purge.go，漏表护栏见 ezfy_admin_purge_test.go）。
//
//	★ 只删二战库；家园账号（qq_jiayuan.users）与家园库的老 ezfy_* 回滚表一律不动。
func (h *EzfyAdmin) AdminEzfyPlayerDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var p model.EzfyProfile
	if err := h.DB.Where("user_id = ?", id).First(&p).Error; err != nil {
		resp.NotFound(c, "玩家不存在")
		return
	}
	uid := p.UserID
	var cityIds []int64
	h.DB.Model(&model.EzfyCity{}).Where("user_id = ?", uid).Pluck("id", &cityIds)
	rows, err := h.ezfyPurgePlayerData(uid, cityIds)
	if err != nil {
		resp.ServerError(c, err)
		return
	}
	resp.OK(c, gin.H{
		"msg":  fmt.Sprintf("已物理删除「%s」在二战库的全部数据（含登录账号，共 %d 行）", p.Nickname, rows),
		"rows": rows,
	})
}

// ============ 数据管理（配置表 + 城池，复用 xy 数据管理白名单模式） ============

// ezfyTableDef 数据表定义（Fields 为可编辑字段白名单：字段名(json) -> 类型）
type ezfyTableDef struct {
	Model  interface{}
	Fields map[string]string
}

var ezfyTableDefs = map[string]ezfyTableDef{
	"activities": {&model.EzfyActivity{}, map[string]string{
		"name": "string", "type": "int", "param": "int",
		"start_time": "int64", "end_time": "int64", "status": "int", "des": "string",
	}},
	"items": {&model.EzfyCfgItem{}, map[string]string{
		"name": "string", "item_type": "int", "param1": "int64",
		"price_gold": "int64", "icon": "string", "description": "string", "stock": "int",
		// ★ 第九轮：钻石道具（price_diamond > 0 只能用钻石买）+ 商城分类
		"price_diamond": "int64", "category": "string",
	}},
	"taskTypes": {&model.EzfyCfgTaskType{}, map[string]string{
		"name": "string", "code": "string", "reset_type": "int", "sort_no": "int", "status": "int",
	}},
	"tasks": {&model.EzfyCfgTask{}, map[string]string{
		"name": "string", "task_type": "string", "target": "int", "reward_gold": "int64",
		"reward_food": "int64", "reward_steel": "int64", "reward_oil": "int64",
		"reward_rare": "int64", "reward_prestige": "int", "sort_no": "int", "type_id": "int", "status": "int",
	}},
	// ★ 2026-09-27 商城「装备|宝箱」价格定义迁到「数据管理」。
	//   装备（散件+套装件）价格/库存在这维护；**战斗属性/三维属性不在白名单里**，
	//   那些字段由「军官装备管理」菜单（散件装备/套装管理 tab）单独维护，避免两处能改同一字段。
	"equipments": {&model.EzfyCfgEquipment{}, map[string]string{
		"name": "string", "type": "string", "tier": "int", "level": "int",
		"slot": "string", "set_id": "int", "series": "string",
		"price_gold": "int64", "price_diamond": "int64", "stock": "int",
		// ★ 2026-10-14 上下架开关（1上架 0下架；与价格解耦）
		"shelf": "int",
	}},
	// ★ 宝箱（套装装备唯一产出渠道）价格/库存/上架 + 奖池由前端「套装装备配置」tab 的奖池弹窗维护。
	"chests": {&model.EzfyCfgChest{}, map[string]string{
		"name": "string", "price_gold": "int64", "price_diamond": "int64",
		"stock": "int", "open_max": "int", "enabled": "int", "sort_no": "int",
		"des": "string", "effect": "string",
	}},
}

// ezfyDataMoved 已经从「数据管理」迁到专属模块的表 → 提示去哪改
//
// ★ 之前「数据管理」把建筑/兵种/科技/野地/城池也放进来了，与
//
//	建筑管理 / 兵种管理 / 科技管理 / 地图管理 / 城市管理 完全重复，
//	同一个字段两处能改、种子策略还不一样，容易改出不一致。
//	现在数据管理只保留「没有专属模块」的零散配置表。
var ezfyDataMoved = map[string]string{
	"buildings":      "「建筑管理 → 总建筑配置」",
	"buildingLevels": "「建筑管理 → 总建筑配置 → 等级配置」",
	"troops":         "「兵种管理 → 兵种配置」",
	"techs":          "「科技管理 → 科技配置」",
	"techLevels":     "「科技管理 → 科技配置 → 等级配置」",
	"wildlands":      "「地图管理 → 野地类型」",
	"cities":         "「城市管理」",
}

// ezfyAnyStr / ezfyAnyInt64 / ezfyAnyInt 把「Find 到 map」拿到的 interface{} 值转成基础类型。
//
// ★ 数据管理列表走的是 `Find(&[]map[string]interface{})`，MySQL 驱动对 varchar 可能给
// []byte、对 bigint 给 int64，直接断言会 panic —— 统一从这里兜底转换。
func ezfyAnyStr(v interface{}) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case []byte:
		return string(x)
	}
	return fmt.Sprint(v)
}

func ezfyAnyInt64(v interface{}) int64 {
	switch x := v.(type) {
	case nil:
		return 0
	case int64:
		return x
	case int:
		return int64(x)
	case int32:
		return int64(x)
	case uint64:
		return int64(x)
	case float64:
		return int64(x)
	case []byte:
		n, _ := strconv.ParseInt(string(x), 10, 64)
		return n
	case string:
		n, _ := strconv.ParseInt(x, 10, 64)
		return n
	}
	return 0
}

func (h *EzfyAdmin) ezfyTableOf(c *gin.Context) (ezfyTableDef, bool) {
	name := c.Param("table")
	def, ok := ezfyTableDefs[name]
	if !ok {
		if where, moved := ezfyDataMoved[name]; moved {
			resp.ParamError(c, "该表已迁到 "+where+" 维护，请到对应模块操作（避免两处重复配置）")
			return def, false
		}
		resp.ParamError(c, "未知数据表")
	}
	return def, ok
}

// AdminEzfyData 数据分页查询（table=buildings/buildingLevels/troops/techs/techLevels/wildlands/items/taskTypes/tasks/cities）
func (h *EzfyAdmin) AdminEzfyData(c *gin.Context) {
	// ★ 2026-10-02 玩家道具使用流水（数据管理 → 道具使用，只读）：
	//   按「玩家ID / 昵称 / 道具名」搜索，附昵称列，倒序展示最近消耗记录。
	if c.Param("table") == "itemUseLogs" {
		page, offset, size := pageOf(c, 10)
		word := strings.TrimSpace(c.Query("word"))
		base := h.DB.Table("ezfy_item_use_logs").
			Select("ezfy_item_use_logs.id, ezfy_item_use_logs.user_id, ezfy_item_use_logs.cfg_id, " +
				"ezfy_item_use_logs.item_name, ezfy_item_use_logs.item_type, ezfy_item_use_logs.count, " +
				"ezfy_item_use_logs.reason, " +
				"DATE_FORMAT(ezfy_item_use_logs.created_at, '%Y-%m-%d %H:%i:%s') AS created_at, ezfy_profile.nickname").
			Joins("LEFT JOIN ezfy_profile ON ezfy_profile.user_id = ezfy_item_use_logs.user_id")
		if word != "" {
			// 数字既可能是「用户ID」也可能是玩家在游戏里看到的「游戏ID」(game_uid)，或道具配置ID(cfg_id)
			if id, err := strconv.Atoi(word); err == nil && id > 0 {
				base = base.Where("ezfy_item_use_logs.user_id = ? OR ezfy_profile.game_uid = ? OR ezfy_item_use_logs.cfg_id = ?", id, id, id)
			} else {
				base = base.Where("(ezfy_profile.nickname LIKE ? OR ezfy_item_use_logs.item_name LIKE ?)", "%"+word+"%", "%"+word+"%")
			}
		}
		var total int64
		base.Count(&total)
		var rows []map[string]interface{}
		base.Order("ezfy_item_use_logs.id DESC").Offset(offset).Limit(size).Find(&rows)
		resp.OK(c, gin.H{"list": rows, "total": total, "page": page, "size": size})
		return
	}
	// ★ 2026-10-16 玩家开箱流水（数据管理 → 宝箱开启，只读）：
	//   按「玩家ID / 昵称 / 宝箱名」搜索，附昵称列，倒序展示最近开箱记录。
	//   （数据源 ezfy_chest_logs：商城购买即开 + 背包待开宝箱开箱，每次开箱写一条。）
	if c.Param("table") == "chestLogs" {
		page, offset, size := pageOf(c, 10)
		word := strings.TrimSpace(c.Query("word"))
		base := h.DB.Table("ezfy_chest_logs").
			Select("ezfy_chest_logs.id, ezfy_chest_logs.user_id, ezfy_chest_logs.chest_id, " +
				"ezfy_chest_logs.chest_name, ezfy_chest_logs.count, ezfy_chest_logs.result, " +
				"DATE_FORMAT(ezfy_chest_logs.created_at, '%Y-%m-%d %H:%i:%s') AS created_at, ezfy_profile.nickname").
			Joins("LEFT JOIN ezfy_profile ON ezfy_profile.user_id = ezfy_chest_logs.user_id")
		if word != "" {
			if id, err := strconv.Atoi(word); err == nil && id > 0 {
				// 数字既可能是「用户ID」也可能是玩家在游戏里看到的「游戏ID」(game_uid)，或宝箱配置ID(chest_id)
				base = base.Where("ezfy_chest_logs.user_id = ? OR ezfy_profile.game_uid = ? OR ezfy_chest_logs.chest_id = ?", id, id, id)
			} else {
				base = base.Where("(ezfy_profile.nickname LIKE ? OR ezfy_chest_logs.chest_name LIKE ?)", "%"+word+"%", "%"+word+"%")
			}
		}
		var total int64
		base.Count(&total)
		var rows []map[string]interface{}
		base.Order("ezfy_chest_logs.id DESC").Offset(offset).Limit(size).Find(&rows)
		resp.OK(c, gin.H{"list": rows, "total": total, "page": page, "size": size})
		return
	}
	// ★ 2026-09-28 玩家钻石流水（数据管理 → 钻石流水，只读）：
	//   按「玩家ID」或「昵称」搜索，附昵称列，倒序展示最近流水。
	if c.Param("table") == "diamondLogs" {
		page, offset, size := pageOf(c, 10)
		word := strings.TrimSpace(c.Query("word"))
		// ★★ 2026-09-28 修复「钻石流水不展示玩家流水」：
		//   JOIN 条件原来写成 `ezfy_profile.uid` —— 但 ezfy_profile **根本没有 uid 列**
		//   （列名是 user_id，见 model.EzfyProfile.UserID 的 gorm tag），SQL 直接报
		//   `ERROR 1054 Unknown column 'ezfy_profile.uid' in 'on clause'`
		//   → 接口 500 → 前端表格永远是空的（看起来像「没有流水」）。
		//   实测：SHOW COLUMNS FROM ezfy_profile = id/user_id/nickname/prestige/camp/
		//   updated_at/game_uid/current_city_id/rename_used/camp_used/recruit_free_limit/diamond/rank。
		base := h.DB.Table("ezfy_diamond_logs").
			Select("ezfy_diamond_logs.id, ezfy_diamond_logs.user_id, ezfy_diamond_logs.change, " +
				"ezfy_diamond_logs.balance, ezfy_diamond_logs.reason, ezfy_diamond_logs.extra, " +
				"DATE_FORMAT(ezfy_diamond_logs.created_at, '%Y-%m-%d %H:%i:%s') AS created_at, ezfy_profile.nickname").
			Joins("LEFT JOIN ezfy_profile ON ezfy_profile.user_id = ezfy_diamond_logs.user_id")
		if word != "" {
			if id, err := strconv.Atoi(word); err == nil && id > 0 {
				// ★ 数字既可能是「用户ID」也可能是玩家在游戏里看到的「游戏ID」(game_uid) → 两个都匹配，
				//   否则管理员拿玩家报的游戏ID来查会查不到（game_uid 首次=家园ID，之后与 user_id 解耦）。
				base = base.Where("ezfy_diamond_logs.user_id = ? OR ezfy_profile.game_uid = ?", id, id)
			} else {
				base = base.Where("ezfy_profile.nickname LIKE ?", "%"+word+"%")
			}
		}
		var total int64
		base.Count(&total)
		var rows []map[string]interface{}
		base.Order("ezfy_diamond_logs.id DESC").Offset(offset).Limit(size).Find(&rows)
		resp.OK(c, gin.H{"list": rows, "total": total, "page": page, "size": size})
		return
	}
	def, ok := h.ezfyTableOf(c)
	if !ok {
		return
	}
	page, offset, size := pageOf(c, 10)
	word := strings.TrimSpace(c.Query("word"))
	var total int64
	q := h.DB.Model(def.Model)
	lq := h.DB.Model(def.Model)
	if word != "" {
		if id, err := strconv.Atoi(word); err == nil {
			q = q.Where("id = ?", id)
			lq = lq.Where("id = ?", id)
		} else {
			q = q.Where("name LIKE ?", "%"+word+"%")
			lq = lq.Where("name LIKE ?", "%"+word+"%")
		}
	}
	// ★ 2026-09-27 「装备道具配置」能过滤出「用户商城在售」的装备（前端默认开启）。
	//   过滤条件与用户端 equipShopList 保持一致：
	//   ① 有价格（黄金/钻石 > 0）；② 类型为「军官装备」或「套装」。
	//   ★★ 2026-10-14 用户「之前下的装备现在上架吧」→ 套装件（Type=套装）/系列件重新上架，
	//      这里**去掉**原来的「非第一批套装件」限制，并放行 Type="套装"，与 equipShopList 同步。
	//   ★★ 2026-10-14 新增 shelf 上下架开关 → 在售判定追加 `shelf = 1`（与 equipShopList 完全一致）。
	if c.Param("table") == "equipments" && c.Query("mall") == "1" {
		mallCond := "shelf = 1 AND (price_gold > 0 OR price_diamond > 0) AND type IN ('军官装备','套装')"
		q = q.Where(mallCond)
		lq = lq.Where(mallCond)
	}
	// ★ 2026-09-27 为爱发电卡：不进「数据管理→道具配置」通用列表（由专属维护页负责）
	if c.Param("table") == "items" {
		q = q.Where("item_type NOT IN ?", []int{ezfyItemTypeLoveCard, ezfyItemTypeLoveCardPro})
		lq = lq.Where("item_type NOT IN ?", []int{ezfyItemTypeLoveCard, ezfyItemTypeLoveCardPro})
	}
	// ★★ 2026-10-07 「道具配置」可读性改造（分类一栏是空的/是数字，整页很乱）：
	//   库里 category **大部分是空串**，玩家在商城看到的分类其实是算出来的
	//   （显式功能分类 → 按 item_type 归类；「钻石/黄金道具」只锁货币，见 ezfyCategoryOf）。
	//   所以这里做两件事，口径全部复用 ezfyCategoryOf，不写第二套：
	//     ① 支持 `cat` 分类筛选 —— 先按同一口径算出命中的 id 集合，再按 id 过滤 + 分页
	//        （道具表只有几十行，全量扫一遍代价可忽略；也避免把推导规则翻译成 SQL 表达式）；
	//     ② 把「当前数据里出现过的分类」回给前端做筛选下拉（顺序走 ezfyCategoryOrder）。
	var itemCats []string
	if c.Param("table") == "items" {
		var all []model.EzfyCfgItem
		h.DB.Where("item_type NOT IN ?", []int{ezfyItemTypeLoveCard, ezfyItemTypeLoveCardPro}).
			Order("id").Find(&all)
		cat := strings.TrimSpace(c.Query("cat"))
		seen := map[string]bool{}
		ids := make([]int, 0, len(all))
		for i := range all {
			ec := ezfyItemCategory(&all[i])
			if !seen[ec] {
				seen[ec] = true
				itemCats = append(itemCats, ec)
			}
			if cat != "" && ec == cat {
				ids = append(ids, all[i].ID)
			}
		}
		sort.Slice(itemCats, func(i, j int) bool {
			return ezfyCategoryRank(itemCats[i]) < ezfyCategoryRank(itemCats[j])
		})
		if cat != "" {
			// ids 为空时 GORM 生成 `id IN (NULL)` → 命中 0 行，正是想要的结果
			q = q.Where("id IN ?", ids)
			lq = lq.Where("id IN ?", ids)
		}
	}
	q.Count(&total)
	var rows []map[string]interface{}
	lq.Order("id").Offset(offset).Limit(size).Find(&rows)
	// ★ 道具配置：补「生效分类」中文名（原来直接显示 category 列，空串就一片空白 → 看着像没分类）
	if c.Param("table") == "items" {
		for _, r := range rows {
			r["category_name"] = ezfyCategoryOf(
				ezfyAnyStr(r["category"]), int(ezfyAnyInt64(r["item_type"])), ezfyAnyInt64(r["price_diamond"]))
		}
	}
	resp.OK(c, gin.H{"list": rows, "total": total, "page": page, "size": size, "cats": itemCats})
}

// AdminEzfyDataCreate 新增数据
func (h *EzfyAdmin) AdminEzfyDataCreate(c *gin.Context) {
	def, ok := h.ezfyTableOf(c)
	if !ok {
		return
	}
	var in map[string]interface{}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	vals := xyPickVals(in, def.Fields)
	if len(vals) == 0 {
		resp.ParamError(c, "无可写入字段")
		return
	}
	if err := h.DB.Model(def.Model).Create(vals).Error; err != nil {
		resp.ParamError(c, "新增失败："+err.Error())
		return
	}
	// ★ 第九轮：数据管理写的是配置表（道具/任务/活动…），
	//   进程内的 ezfyCfg 缓存必须重载，否则游戏内看不到新增的道具。
	h.ezfyReload()
	resp.OK(c, gin.H{"msg": "新增成功"})
}

// AdminEzfyDataUpdate 修改数据
func (h *EzfyAdmin) AdminEzfyDataUpdate(c *gin.Context) {
	def, ok := h.ezfyTableOf(c)
	if !ok {
		return
	}
	id, _ := strconv.Atoi(c.Param("id"))
	var in map[string]interface{}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	vals := xyPickVals(in, def.Fields)
	if len(vals) == 0 {
		resp.ParamError(c, "无可修改字段")
		return
	}
	if err := h.DB.Model(def.Model).Where("id = ?", id).Updates(vals).Error; err != nil {
		resp.ParamError(c, "修改失败："+err.Error())
		return
	}
	// ★ 第九轮：配置表改完必须重载 ezfyCfg，否则游戏内还是旧值
	h.ezfyReload()
	resp.OK(c, gin.H{"msg": "修改成功"})
}

// AdminEzfyDataDelete 删除数据
func (h *EzfyAdmin) AdminEzfyDataDelete(c *gin.Context) {
	def, ok := h.ezfyTableOf(c)
	if !ok {
		return
	}
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.DB.Delete(def.Model, id).Error; err != nil {
		resp.ParamError(c, "删除失败："+err.Error())
		return
	}
	// ★ 第九轮：配置表删完必须重载 ezfyCfg
	h.ezfyReload()
	resp.OK(c, gin.H{"msg": "删除成功"})
}

// AdminEzfyDataShelf PUT /admin/ezfy-data-shelf/:table
//
// ★ 2026-10-14 上下架批量开关（用户「上下架不好维护，优化下」）。
//
//	Body: { ids: [1,2,3], shelf: 1 }（1=上架 0=下架）。
//	只对**含 shelf 列**的表生效（当前 = equipments）；下架**不清价**（价格留着，重新上架即恢复）。
//	一条 UPDATE ... WHERE id IN (...) 完成，避免 N 次行内点击。
//
//	★ 路由走独立前缀 `/ezfy-data-shelf/:table`：`/ezfy-data/:table/shelf` 会与
//	  `/ezfy-data/:table/:id` 同层冲突（静态段 vs 参数段）→ Gin 把 shelf 当成 id。
func (h *EzfyAdmin) AdminEzfyDataShelf(c *gin.Context) {
	def, ok := h.ezfyTableOf(c)
	if !ok {
		return
	}
	// 仅装备表有上下架（其它表没有 shelf 列）
	if c.Param("table") != "equipments" {
		resp.ParamError(c, "该表不支持上下架")
		return
	}
	var in struct {
		Ids   []int `json:"ids"`
		Shelf int   `json:"shelf"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || len(in.Ids) == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	shelf := 0
	if in.Shelf > 0 {
		shelf = 1
	}
	if err := h.DB.Model(def.Model).Where("id IN ?", in.Ids).
		Updates(map[string]interface{}{"shelf": shelf}).Error; err != nil {
		resp.ParamError(c, "操作失败："+err.Error())
		return
	}
	h.ezfyReload()
	word := "下架"
	if shelf == 1 {
		word = "上架"
	}
	resp.OK(c, gin.H{"msg": fmt.Sprintf("已%s %d 件", word, len(in.Ids)), "shelf": shelf})
}

// ============ 流水管理（出征/世界聊天/交易所） ============

// AdminEzfyOrders 出征订单列表（word=用户ID/玩家昵称，type=出征类型，status=状态）
// ezfyOrderTroopsText 出征部队 → 「步兵×100 卡车×50」
func ezfyOrderTroopsText(raw string) string {
	if raw == "" {
		return ""
	}
	groups := parseGroups(raw)
	parts := make([]string, 0, len(groups))
	for _, g := range groups {
		if g.Count <= 0 {
			continue
		}
		name := ezfyCfg.troopName(g.TroopId, 0)
		if name == "" {
			name = "兵种" + strconv.Itoa(g.TroopId)
		}
		parts = append(parts, name+"×"+strconv.FormatInt(g.Count, 10))
	}
	return strings.Join(parts, " ")
}

// ezfyOrderResText 资源 JSON → 「粮100 钢50」
func ezfyOrderResText(raw string) string {
	if raw == "" || raw == "{}" {
		return ""
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return raw
	}
	order := []struct{ k, n string }{{"food", "粮"}, {"steel", "钢"}, {"oil", "油"}, {"rare", "稀"}, {"gold", "金"}}
	parts := []string{}
	for _, it := range order {
		v, ok := m[it.k]
		if !ok {
			continue
		}
		n, _ := strconv.ParseInt(fmt.Sprintf("%v", v), 10, 64)
		if n > 0 {
			parts = append(parts, it.n+strconv.FormatInt(n, 10))
		}
	}
	return strings.Join(parts, " ")
}

func (h *EzfyAdmin) AdminEzfyOrders(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	word := strings.TrimSpace(c.Query("word"))
	orderType := atoiOr(c.Query("type"), 0)
	status := atoiOr(c.Query("status"), -1)
	q := h.DB.Model(&model.EzfyOrder{})
	if word != "" {
		if uid, err := strconv.Atoi(word); err == nil {
			q = q.Where("user_id = ?", uid)
		} else {
			var ids []uint
			h.DB.Model(&model.EzfyProfile{}).Select("user_id").
				Where("nickname LIKE ?", "%"+word+"%").Scan(&ids)
			if len(ids) > 0 {
				q = q.Where("user_id IN ?", ids)
			} else {
				q = q.Where("1 = 0")
			}
		}
	}
	if orderType > 0 {
		q = q.Where("order_type = ?", orderType)
	}
	if status >= 0 {
		q = q.Where("status = ?", status)
	}
	var total int64
	q.Count(&total)
	var rows []model.EzfyOrder
	q.Order("id DESC").Offset(offset).Limit(size).Find(&rows)
	type rowOut struct {
		model.EzfyOrder
		PlayerName string `json:"player_name"`
		HomeNum    string `json:"home_num"`
		TypeName   string `json:"type_name"`
		// ★ 把部队/资源/待带回整理成可读文字，管理端「出征记录详情」直接用，
		//   免得前端还要自己去拉兵种表（跨模块权限也不一定给）。
		TroopsText string `json:"troops_text"`
		ResText    string `json:"res_text"`
		CarryText  string `json:"carry_text"`
	}
	out := []rowOut{}
	for _, o := range rows {
		pn, hn := h.ezfyAdminName(o.UserID)
		out = append(out, rowOut{EzfyOrder: o, PlayerName: pn, HomeNum: hn,
			TypeName:   ezfyOrderTypeName(o.OrderType),
			TroopsText: ezfyOrderTroopsText(o.Troops),
			ResText:    ezfyOrderResText(o.Resources),
			CarryText:  ezfyOrderResText(o.Carry)})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// AdminEzfyOrderDelete 删除出征订单
func (h *EzfyAdmin) AdminEzfyOrderDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.EzfyOrder{}, id)
	resp.OK(c, gin.H{"msg": "已删除"})
}

// ============ 战报查询（2026-10-06） ============
//
// ★ （点7）：「管理端加个战报查询模块，可以查询玩家战报，战报不是订单没有订单概念」。
// 战报 = ezfy_report（侦察/掠夺/征服/战斗/采集/系统），与出征订单 ezfy_order 是两张表，
// 所以这里的列表独立于「出征记录」模块，按 玩家(user_id/昵称) + 战报类型 过滤。
// 类型中文名复用 ezfyReportTypeName(reportType, title)（优先按标题前缀判定）。

// AdminEzfyReports 战报列表（可按玩家 / 战报类型 / 时间范围过滤）
//
// ★ 2026-10-08（第2轮优化）彻底去掉「捞内存再分页」：
//
//	ezfy_report 已新增持久化列 type_name（写战报时由 EzfyReportTypeName 落库，
//	存量行用 cmd/backfilltype 一次性回填），因此类型过滤也直接走 SQL WHERE，
//	与玩家/时间条件统一进 Count + Offset/Limit 库内分页，任何组合都不再全表读内存。
func (h *EzfyAdmin) AdminEzfyReports(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	word := strings.TrimSpace(c.Query("word"))
	// ★ 2026-10-08 类型过滤改按「持久化 type_name 列」（写时落库），见 cmd/backfilltype。
	typeName := strings.TrimSpace(c.Query("type"))
	// ★ 2026-10-08 新增「时间起止」检索：created_at 落在此区间（end 含整天末）。
	startStr := strings.TrimSpace(c.Query("start"))
	endStr := strings.TrimSpace(c.Query("end"))

	q := h.DB.Model(&model.EzfyReport{})
	if word != "" {
		if uid, err := strconv.Atoi(word); err == nil {
			// ★ 2026-10-07 数字既可能是 user_id，也可能是游戏里看到的游戏ID(game_uid)，两者都匹配。
			// ★★ 2026-10-15 用户规则：**家园号条件查询全部换成游戏号** → 去掉按
			//   `users.username`(家园号) 反查 uid 的那一支（家园号会随转靓号变，游戏号才是玩家报的那个）。
			// ★★ 2026-10-17 卡顿优化：原来用 `OR user_id IN (SELECT ...)` 子查询，每次查询都重跑子查询。
			//   改成**一次性内存查集合**再 IN（ezfy_profile 有 user_id 索引，战报走 idx_user 索引命中），
			//   管理端查人机战报（人机 GameUID==UserID）天然只命中 user_id = ? 一个值。
			var gids []uint
			h.DB.Model(&model.EzfyProfile{}).Where("game_uid = ?", uid).Pluck("user_id", &gids)
			ids := append([]uint{uint(uid)}, gids...)
			if len(ids) > 1 {
				q = q.Where("user_id IN ?", ids)
			} else {
				q = q.Where("user_id = ?", uid)
			}
		} else {
			var ids []uint
			h.DB.Model(&model.EzfyProfile{}).Select("user_id").
				Where("nickname LIKE ?", "%"+word+"%").Scan(&ids)
			// 家园库只按**昵称**模糊（不再按家园号）
			var uids []uint
			h.home().Model(&model.User{}).Select("id").
				Where("nickname LIKE ?", "%"+word+"%").Scan(&uids)
			ids = append(ids, uids...)
			if len(ids) > 0 {
				q = q.Where("user_id IN ?", ids)
			} else {
				q = q.Where("1 = 0")
			}
		}
	}
	// 类型过滤走持久化列（可走索引），不再内存运算。
	if typeName != "" {
		q = q.Where("type_name = ?", typeName)
	}
	// ★ 2026-10-18 支持按用户ID精确过滤（人机出征战报弹窗：uid=人机 user_id，只显示该人机的战报）
	if uidStr := strings.TrimSpace(c.Query("uid")); uidStr != "" {
		if uid, err := strconv.Atoi(uidStr); err == nil && uid > 0 {
			q = q.Where("user_id = ?", uid)
		}
	}
	// 时间范围走 SQL（created_at 起止，配合索引）。
	if startStr != "" {
		if st, err := time.ParseInLocation("2006-01-02", startStr, time.Local); err == nil {
			q = q.Where("created_at >= ?", st)
		}
	}
	if endStr != "" {
		if et, err := time.ParseInLocation("2006-01-02", endStr, time.Local); err == nil {
			// 结束日期按「整天含末」：< 末日+1 天
			q = q.Where("created_at < ?", et.AddDate(0, 0, 1))
		}
	}

	var rows []model.EzfyReport
	var total int64
	q.Count(&total)
	q.Order("id DESC").Offset(offset).Limit(size).Find(&rows)

	// ★ 2026-10-08 类型下拉交给后端给出库里真实的 type_name 去重列表，
	//   前端不再手写枚举（避免出现库里没有的「增援」/漏掉「升级、交易、返航」等）。
	var types []string
	h.DB.Model(&model.EzfyReport{}).Distinct("type_name").
		Where("type_name <> ''").Order("type_name").Pluck("type_name", &types)

	type rowOut struct {
		model.EzfyReport
		// ★ 2026-10-08 覆盖内嵌的 type_name 字段：存量未回填/部署前老代码新写的行(空串)，
		//   用 ezfyTypeNameStored 现算兜底，管理端不再显示空白类型（与玩家端同口径）。
		TypeName   string `json:"type_name"`
		PlayerName string `json:"player_name"`
		HomeNum    string `json:"home_num"`
		Preview    string `json:"preview"`
		// ★ 2026-10-18 目标玩家名字：出征战报（玩家城目标）展示守方玩家名
		TargetName string `json:"target_name"`
	}
	out := []rowOut{}

	// ★★ 2026-10-18 卡顿优化：原来每行 `ezfyAdminName`（h.DB + 家园库各查一次）= N+1 跨库，
	//   一次翻页 20 条 = 40 次往返。改为**批量取一次**（ezfyAdminNamesBatch），
	//   目标玩家名再从订单/城市批量反查后内存映射，整页最多 4~5 条 SQL。
	uidSet := map[uint]bool{}
	orderIds := make([]int64, 0, len(rows))
	for _, r := range rows {
		uidSet[r.UserID] = true
		if r.OrderId > 0 {
			orderIds = append(orderIds, r.OrderId)
		}
	}
	// 订单 → 目标玩家城 ID（只认打玩家城 target_type=3；野地/寇城无目标玩家）
	orderCity := map[int64]int64{}
	if len(orderIds) > 0 {
		var orders []model.EzfyOrder
		h.DB.Select("id, target_type, target_id").Where("id IN ?", orderIds).Find(&orders)
		for _, o := range orders {
			if o.TargetType == 3 && o.TargetId > 0 {
				orderCity[int64(o.ID)] = o.TargetId
			}
		}
	}
	// 玩家城 ID → 守方 uid
	cityUid := map[int64]uint{}
	if len(orderCity) > 0 {
		cids := make([]int64, 0, len(orderCity))
		for _, cid := range orderCity {
			cids = append(cids, cid)
		}
		var cities []model.EzfyCity
		h.DB.Select("id, user_id").Where("id IN ?", cids).Find(&cities)
		for _, ct := range cities {
			cityUid[int64(ct.ID)] = ct.UserID
			uidSet[ct.UserID] = true
		}
	}
	// 订单 → 目标玩家 uid
	targetUid := map[int64]uint{}
	for oid, cid := range orderCity {
		if tu, ok := cityUid[cid]; ok {
			targetUid[oid] = tu
		}
	}
	allUids := make([]uint, 0, len(uidSet))
	for u := range uidSet {
		allUids = append(allUids, u)
	}
	nickOf, _ := h.ezfyAdminNamesBatch(allUids)

	for _, r := range rows {
		preview := r.Content
		if len([]rune(preview)) > 200 {
			preview = string([]rune(preview)[:200])
		}
		tname := ""
		if tu, ok := targetUid[r.OrderId]; ok {
			tname = nickOf[tu]
		}
		out = append(out, rowOut{EzfyReport: r, TypeName: ezfyTypeNameStored(r),
			PlayerName: nickOf[r.UserID], HomeNum: "", Preview: preview, TargetName: tname})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size, "types": types})
}

// AdminEzfyReportDetail 战报详情（content/detail 全文）
func (h *EzfyAdmin) AdminEzfyReportDetail(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var r model.EzfyReport
	if err := h.DB.First(&r, id).Error; err != nil {
		resp.NotFound(c, "战报不存在")
		return
	}
	pn, hn := h.ezfyAdminName(r.UserID)
	resp.OK(c, gin.H{
		"report":      r,
		"player_name": pn, "home_num": hn,
		"type_name": ezfyTypeNameStored(r),
	})
}

// AdminEzfyReportDelete 删除战报
func (h *EzfyAdmin) AdminEzfyReportDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.EzfyReport{}, id)
	resp.OK(c, gin.H{"msg": "已删除"})
}

// AdminEzfyReportsDeleteRange 时间范围内**物理删除**战报（POST /admin/ezfy-reports/delete-range，body: {start, end}）
//
// ★ 2026-10-18 用户需求「战报查询 加个 时间范围内删除（物理删除历史数据，别展示出来）」：
//
//	按 created_at 落在 [start, end] 的战报**物理删除**（Unscoped().Delete，连软删标记一起清掉），
//	删除后列表不再展示；管理员操作前由前端二次确认，接口返回实际删除条数便于提示。
func (h *EzfyAdmin) AdminEzfyReportsDeleteRange(c *gin.Context) {
	var in struct {
		Start string `json:"start"`
		End   string `json:"end"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	startStr := strings.TrimSpace(in.Start)
	endStr := strings.TrimSpace(in.End)
	if startStr == "" && endStr == "" {
		resp.ParamError(c, "请选择要删除的时间范围")
		return
	}
	q := h.DB.Unscoped().Model(&model.EzfyReport{})
	if startStr != "" {
		st, ok := ezfyAdminParseTime(startStr)
		if !ok {
			resp.ParamError(c, "开始时间格式应为 2006-01-02 或 2006-01-02 15:04:05")
			return
		}
		q = q.Where("created_at >= ?", st)
	}
	if endStr != "" {
		et, ok := ezfyAdminParseTime(endStr)
		if !ok {
			resp.ParamError(c, "结束时间格式应为 2006-01-02 或 2006-01-02 15:04:05")
			return
		}
		// 纯日期（无时间）→ 整天含末（< 次日 0 点）；带具体时间 → 精确到该时刻之前
		if strings.Contains(endStr, ":") {
			q = q.Where("created_at < ?", et)
		} else {
			q = q.Where("created_at < ?", et.AddDate(0, 0, 1))
		}
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		resp.ServerError(c, err)
		return
	}
	if total <= 0 {
		resp.OK(c, gin.H{"msg": "该时间范围内没有战报，无需删除", "deleted": 0})
		return
	}
	if err := q.Delete(&model.EzfyReport{}).Error; err != nil {
		resp.ServerError(c, err)
		return
	}
	resp.OK(c, gin.H{
		"msg":     fmt.Sprintf("已物理删除 %d 条战报（%s ~ %s），删除后不可恢复", total, startStr, endStr),
		"deleted": total,
	})
}

// ezfyAdminParseTime 管理端时间范围参数解析：兼容「2006-01-02」与「2006-01-02 15:04:05」两种写法。
func ezfyAdminParseTime(s string) (time.Time, bool) {
	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02"} {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// AdminEzfyChats 聊天记录列表（频道过滤 + 关键词 + 时间范围）
//
// ★ 2026-10-17 由「世界聊天」升级为多频道：channel 1公共 2军团 4系统，
//
//	配合管理端「聊天管理」的 世界/军团/系统 三个 tab。
func (h *EzfyAdmin) AdminEzfyChats(c *gin.Context) {
	page, offset, size := pageOf(c, 20)
	word := strings.TrimSpace(c.Query("word"))
	channel := atoiOr(c.Query("channel"), 0)
	q := h.DB.Model(&model.EzfyChat{})
	if channel > 0 {
		q = q.Where("channel = ?", channel)
	}
	if word != "" {
		q = q.Where("user_name LIKE ? OR content LIKE ?", "%"+word+"%", "%"+word+"%")
	}
	if startStr := strings.TrimSpace(c.Query("start")); startStr != "" {
		if st, ok := ezfyAdminParseTime(startStr); ok {
			q = q.Where("created_at >= ?", st)
		}
	}
	if endStr := strings.TrimSpace(c.Query("end")); endStr != "" {
		if et, ok := ezfyAdminParseTime(endStr); ok {
			// 结束时间按「整天含末」：纯日期时 < 末日+1 天；带时分秒则 < 下一秒
			if strings.Contains(endStr, ":") {
				q = q.Where("created_at < ?", et.Add(time.Second))
			} else {
				q = q.Where("created_at < ?", et.AddDate(0, 0, 1))
			}
		}
	}
	var total int64
	q.Count(&total)
	var rows []model.EzfyChat
	q.Order("id DESC").Offset(offset).Limit(size).Find(&rows)
	resp.OK(c, gin.H{"list": rows, "total": total, "page": page, "size": size})
}

// AdminEzfyChatBatchDelete 按频道+时间范围批量物理删除聊天记录
//
// ★ 2026-10-17 管理端「聊天管理」世界/军团/系统 tab 的「删除所选时间段」：
//
//	channel + start/end 均必填，防止误删整频道数据。
func (h *EzfyAdmin) AdminEzfyChatBatchDelete(c *gin.Context) {
	channel := atoiOr(c.Query("channel"), 0)
	startStr := strings.TrimSpace(c.Query("start"))
	endStr := strings.TrimSpace(c.Query("end"))
	if channel <= 0 || startStr == "" || endStr == "" {
		resp.ParamError(c, "请选择频道并填写起止时间")
		return
	}
	st, ok1 := ezfyAdminParseTime(startStr)
	et, ok2 := ezfyAdminParseTime(endStr)
	if !ok1 || !ok2 {
		resp.ParamError(c, "时间格式错误，应为 2006-01-02 或 2006-01-02 15:04:05")
		return
	}
	q := h.DB.Model(&model.EzfyChat{}).Where("channel = ? AND created_at >= ?", channel, st)
	if strings.Contains(endStr, ":") {
		q = q.Where("created_at < ?", et.Add(time.Second))
	} else {
		q = q.Where("created_at < ?", et.AddDate(0, 0, 1))
	}
	// ezfy_chat 无软删字段，Delete 即物理删除
	res := q.Delete(&model.EzfyChat{})
	resp.OK(c, gin.H{"msg": "已删除", "count": res.RowsAffected})
}

// AdminEzfyChatDelete 删除世界聊天
func (h *EzfyAdmin) AdminEzfyChatDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.EzfyChat{}, id)
	resp.OK(c, gin.H{"msg": "已删除"})
}

// AdminEzfyExchangeDealtDelete 一键删除全部已成交记录（status = 1）
//
// ★ 2026-10-17 管理端「交易行维护 → 挂单维护」的一键清理按钮；
//
//	ezfy_exchange 无软删字段，Delete 即物理删除。
func (h *EzfyAdmin) AdminEzfyExchangeDealtDelete(c *gin.Context) {
	res := h.DB.Where("status = 1").Delete(&model.EzfyExchange{})
	resp.OK(c, gin.H{"msg": "已删除", "count": res.RowsAffected})
}

func (h *EzfyAdmin) ezfyAdminName(uid uint) (string, string) {
	var p model.EzfyProfile
	nick := ""
	if err := h.DB.Where("user_id = ?", uid).First(&p).Error; err == nil {
		nick = p.Nickname
	}
	var u model.User
	num := ""
	if err := h.home().First(&u, uid).Error; err == nil {
		num = u.Username
	}
	return nick, num
}

// atoiOr 解析整数，空串/非法时返回默认值
func atoiOr(s string, def int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

// ============ 系统管理（军团/公告/统计/维护） ============

// AdminEzfyCorps 军团列表
func (h *EzfyAdmin) AdminEzfyCorps(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	word := strings.TrimSpace(c.Query("word"))
	q := h.DB.Model(&model.EzfyCorps{})
	if word != "" {
		q = q.Where("name LIKE ?", "%"+word+"%")
	}
	var total int64
	q.Count(&total)
	var rows []model.EzfyCorps
	q.Order("member_count DESC, id ASC").Offset(offset).Limit(size).Find(&rows)
	type rowOut struct {
		model.EzfyCorps
		LeaderName string `json:"leader_name"`
	}
	out := []rowOut{}
	for _, r := range rows {
		ln, _ := h.ezfyAdminName(r.LeaderUserId)
		out = append(out, rowOut{EzfyCorps: r, LeaderName: ln})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// AdminEzfyCorpsDelete 解散军团（清除成员与军团聊天）
func (h *EzfyAdmin) AdminEzfyCorpsDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var cp model.EzfyCorps
	if err := h.DB.First(&cp, id).Error; err != nil {
		resp.NotFound(c, "军团不存在")
		return
	}
	h.DB.Where("corps_id = ?", id).Delete(&model.EzfyCorpsMember{})
	h.DB.Where("corps_id = ?", id).Delete(&model.EzfyCorpsChat{})
	h.DB.Delete(&model.EzfyCorps{}, id)
	resp.OK(c, gin.H{"msg": "已解散军团「" + cp.Name + "」"})
}

// AdminEzfyNotices 公告列表（user_id=0 全员公告）
func (h *EzfyAdmin) AdminEzfyNotices(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	q := h.DB.Model(&model.EzfyNotice{}).Where("user_id = 0")
	var total int64
	q.Count(&total)
	var rows []model.EzfyNotice
	q.Order("is_top DESC, id DESC").Offset(offset).Limit(size).Find(&rows)
	resp.OK(c, gin.H{"list": rows, "total": total, "page": page, "size": size})
}

// AdminEzfyAnnounce 发布游戏公告（全员可见，玩家游戏内公告栏展示）
func (h *EzfyAdmin) AdminEzfyAnnounce(c *gin.Context) {
	var in struct {
		Title   string `json:"title"`
		Content string `json:"content"`
		IsTop   int    `json:"is_top"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || strings.TrimSpace(in.Content) == "" {
		resp.ParamError(c, "请输入公告内容")
		return
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = "系统公告"
	}
	h.DB.Create(&model.EzfyNotice{UserId: 0, Title: trimStr(title, 100),
		Content: trimStr(strings.TrimSpace(in.Content), 2000), IsTop: in.IsTop})
	// 同步到世界聊天频道（公共频道 + 玩家样式消息）。
	// ⚠️ 走 map 显式给 channel/talk_type：Channel/TalkType 都带 `default:1` 标签，
	//    用结构体建且值为零时会被 GORM 从 INSERT 剔除、退回数据库默认值，语义全靠"碰巧对"。
	h.DB.Model(&model.EzfyChat{}).Create(map[string]interface{}{
		"user_id": 0, "user_name": "系统",
		"content":    trimStr("【公告】"+in.Content, 200),
		"channel":    1, // 公共频道
		"talk_type":  1, // 玩家样式（公告是"系统发的普通消息"，不是只读的系统消息）
		"created_at": time.Now(),
	})
	resp.OK(c, gin.H{"msg": "公告已发布"})
}

// AdminEzfyNoticeUpdate 编辑已发布的公告（标题/内容/置顶）
//
// ★ 已发布的公告要能编辑（原来只能删了重发）。
func (h *EzfyAdmin) AdminEzfyNoticeUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var n model.EzfyNotice
	if err := h.DB.Where("user_id = 0").First(&n, id).Error; err != nil {
		resp.NotFound(c, "公告不存在")
		return
	}
	var in struct {
		Title   *string `json:"title"`
		Content *string `json:"content"`
		IsTop   *int    `json:"is_top"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	updates := map[string]interface{}{}
	if in.Title != nil {
		t := strings.TrimSpace(*in.Title)
		if t == "" {
			t = "游戏公告"
		}
		updates["title"] = trimStr(t, 100)
	}
	if in.Content != nil {
		ct := strings.TrimSpace(*in.Content)
		if ct == "" {
			resp.ParamError(c, "公告内容不能为空")
			return
		}
		updates["content"] = trimStr(ct, 2000)
	}
	if in.IsTop != nil {
		updates["is_top"] = *in.IsTop
	}
	if len(updates) == 0 {
		resp.ParamError(c, "无可修改字段")
		return
	}
	if err := h.DB.Model(&model.EzfyNotice{}).Where("id = ?", n.ID).Updates(updates).Error; err != nil {
		resp.ParamError(c, "保存失败："+err.Error())
		return
	}
	resp.OK(c, gin.H{"msg": "公告已更新"})
}

// AdminEzfyNoticeDelete 删除公告
func (h *EzfyAdmin) AdminEzfyNoticeDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Where("user_id = 0").Delete(&model.EzfyNotice{}, id)
	resp.OK(c, gin.H{"msg": "已删除"})
}

// AdminEzfyStats 游戏统计
func (h *EzfyAdmin) AdminEzfyStats(c *gin.Context) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	var players, todayPlayers, cities, orders, todayOrders, corps, exchanges int64
	h.DB.Model(&model.EzfyProfile{}).Count(&players)
	// EzfyProfile 无 created_at，用当日新建城池数代替「今日新增」
	h.DB.Model(&model.EzfyCity{}).Where("created_at >= ?", today).Count(&todayPlayers)
	h.DB.Model(&model.EzfyCity{}).Count(&cities)
	h.DB.Model(&model.EzfyOrder{}).Count(&orders)
	h.DB.Model(&model.EzfyOrder{}).Where("created_at >= ?", today).Count(&todayOrders)
	h.DB.Model(&model.EzfyCorps{}).Count(&corps)
	h.DB.Model(&model.EzfyExchange{}).Where("status = 0").Count(&exchanges)
	type agg struct{ Sum int64 }
	var gold, food, steel, oil, rare, prestige agg
	h.DB.Model(&model.EzfyCity{}).Select("COALESCE(SUM(gold),0) as sum").Scan(&gold)
	h.DB.Model(&model.EzfyCity{}).Select("COALESCE(SUM(food),0) as sum").Scan(&food)
	h.DB.Model(&model.EzfyCity{}).Select("COALESCE(SUM(steel),0) as sum").Scan(&steel)
	h.DB.Model(&model.EzfyCity{}).Select("COALESCE(SUM(oil),0) as sum").Scan(&oil)
	h.DB.Model(&model.EzfyCity{}).Select("COALESCE(SUM(rare),0) as sum").Scan(&rare)
	h.DB.Model(&model.EzfyProfile{}).Select("COALESCE(SUM(prestige),0) as sum").Scan(&prestige)
	// 阵营分布
	type campRow struct {
		Camp int   `json:"camp"`
		Cnt  int64 `json:"cnt"`
	}
	camps := []campRow{}
	h.DB.Model(&model.EzfyProfile{}).Select("camp, COUNT(*) as cnt").Group("camp").Scan(&camps)
	// 声望排行 TOP10
	type rankRow struct {
		Nickname string `json:"nickname"`
		Prestige int    `json:"prestige"`
		Camp     int    `json:"camp"`
	}
	tops := []rankRow{}
	h.DB.Model(&model.EzfyProfile{}).Select("nickname, prestige, camp").
		Order("prestige DESC").Limit(10).Scan(&tops)
	resp.OK(c, gin.H{
		"players": players, "today_players": todayPlayers,
		"cities": cities, "orders": orders, "today_orders": todayOrders,
		"corps": corps, "exchanges": exchanges,
		"gold": gold.Sum, "food": food.Sum, "steel": steel.Sum, "oil": oil.Sum,
		"rare": rare.Sum, "prestige": prestige.Sum,
		"camps": camps, "tops": tops,
	})
}

// 维护开关内存缓存（5 秒）。
// ★ 2026-10-02 线上 1核1G CPU 100% 优化：EzfyMaintGate 中间件挂在所有 /games/ezfy/* 路由上，
//
//	原本每个请求都查 settings 表（~8 次/秒），与战力榜全表扫描叠加打满 MySQL 单核。
//	维护开关极少变化，缓存 5 秒对「开关维护」的响应速度几乎无感知；AdminEzfyServerSet 写入后
//	主动失效，下次请求立即重读。
var (
	maintCacheMu     sync.Mutex
	maintCacheAt     int64
	maintCacheOn     string
	maintCacheNotice string
)

func ezfyMaintStatus(db *gorm.DB) (on, notice string) {
	now := time.Now().UnixMilli()
	maintCacheMu.Lock()
	defer maintCacheMu.Unlock()
	// ★ 2026-10-05 性能：TTL 5s → 30s。
	//   这两条 `SELECT value FROM settings WHERE key=...` 是**每个 ezfy 请求的第一步**（串行），
	//   线上跨 WAN 单条 ~55ms → 5 秒 TTL 下几乎每个请求都要白付 ~110ms。
	//   维护开关是管理端低频操作，30 秒收敛完全够（与二战配置 30s 周期刷新同一量级）。
	if maintCacheAt > 0 && now-maintCacheAt < 30000 {
		return maintCacheOn, maintCacheNotice
	}
	maintCacheAt = now
	db.Model(&model.Setting{}).Select("`value`").Where("`key` = 'ezfy_maintenance'").Scan(&maintCacheOn)
	db.Model(&model.Setting{}).Select("`value`").Where("`key` = 'ezfy_maintenance_notice'").Scan(&maintCacheNotice)
	return maintCacheOn, maintCacheNotice
}

func ezfyMaintInvalidate() {
	maintCacheMu.Lock()
	maintCacheAt = 0
	maintCacheMu.Unlock()
}

// AdminEzfyServer 服务器维护状态
func (h *EzfyAdmin) AdminEzfyServer(c *gin.Context) {
	on, notice := ezfyMaintStatus(h.DB)
	var players, cities int64
	h.DB.Model(&model.EzfyProfile{}).Count(&players)
	h.DB.Model(&model.EzfyCity{}).Count(&cities)
	resp.OK(c, gin.H{"maintenance": on == "1", "notice": notice, "players": players, "cities": cities})
}

// AdminEzfyServerSet 设置维护模式（on=true 维护中，游戏接口统一拦截）
func (h *EzfyAdmin) AdminEzfyServerSet(c *gin.Context) {
	var in struct {
		On     bool   `json:"on"`
		Notice string `json:"notice"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	val := "0"
	if in.On {
		val = "1"
	}
	xySettingSet(h.DB, "ezfy_maintenance", val)
	xySettingSet(h.DB, "ezfy_maintenance_notice", strings.TrimSpace(in.Notice))
	ezfyMaintInvalidate() // 失效缓存，下次请求立即重读新值
	if in.On {
		resp.OK(c, gin.H{"msg": "二战风云已进入维护模式，玩家将无法进行游戏操作"})
	} else {
		resp.OK(c, gin.H{"msg": "二战风云已开放，玩家可正常游戏"})
	}
}

// EzfyMaintGate 服务器维护拦截（维护中所有游戏接口统一返回维护公告）
func (h *EzfyHandler) EzfyMaintGate() gin.HandlerFunc {
	return func(c *gin.Context) {
		on, notice := ezfyMaintStatus(h.DB)
		if on == "1" {
			if strings.TrimSpace(notice) == "" {
				notice = "服务器维护中，请稍后再来"
			}
			resp.ParamError(c, "【服务器维护中】"+notice)
			c.Abort()
			return
		}
		c.Next()
	}
}
