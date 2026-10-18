package ezfy

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 二战风云管理端 —— 军衔配置维护 + 玩家军衔维护
//
// 军衔表复刻原版 rank/rankIndex.html：军衔等级 / 职位 / 需要声望 / **可建城数**。
// ★ 可建城数就是「玩家能拥有的城市数量上限」（军衔限制分城数量）。

// ============ 军衔配置 ============

// AdminEzfyRanks 军衔配置列表（含该军衔玩家数与晋升宝物）
func (h *EzfyAdmin) AdminEzfyRanks(c *gin.Context) {
	ezfyCfg.load(h.DB) // 晋升宝物从配置缓存读（玩家端口径一致；改过配置的 Update 会 reload）
	var rows []model.EzfyCfgRank
	h.DB.Order("id").Find(&rows)
	if len(rows) == 0 {
		rows = ezfyDefaultRanks()
	}
	// 每个军衔当前有多少玩家
	type cntAgg struct {
		RankIdx int
		Cnt     int64
	}
	var profs []model.EzfyProfile
	h.DB.Select("prestige", "rank").Find(&profs)
	cnt := map[int]int64{}
	for _, p := range profs {
		cnt[ezfyProfileRank(&p)]++
	}
	out := make([]gin.H, 0, len(rows))
	for i, r := range rows {
		out = append(out, gin.H{
			"id": r.ID, "level": i + 1, "name": r.Name, "post": r.Post,
			"need_prestige": r.NeedPrestige, "city_max": r.CityMax, "des": r.Des,
			"treasures":    ezfyRankTreasureReqs(r.ID), // 该军衔晋升所需宝物（管理端可维护）
			"player_count": cnt[i],
		})
	}
	resp.OK(c, gin.H{"list": out, "total": len(out),
		"usage": "「可建城数」即该军衔下玩家能拥有的城市数量上限；「晋升宝物」为声望达标后需提交的宝物（野地采集掉落，管理端可维护）"})
}

// AdminEzfyRankUpdate 修改军衔配置
func (h *EzfyAdmin) AdminEzfyRankUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var r model.EzfyCfgRank
	if err := h.DB.First(&r, id).Error; err != nil {
		resp.NotFound(c, "军衔不存在")
		return
	}
	var in struct {
		Name         string             `json:"name"`
		Post         string             `json:"post"`
		NeedPrestige *int               `json:"need_prestige"`
		CityMax      *int               `json:"city_max"`
		Des          string             `json:"des"`
		Treasures    []ezfyRankTreasure `json:"treasures"` // 晋升所需宝物（空数组=该军衔无需宝物）
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	updates := map[string]interface{}{}
	if s := strings.TrimSpace(in.Name); s != "" {
		updates["name"] = trimStr(s, 20)
	}
	if s := strings.TrimSpace(in.Post); s != "" {
		updates["post"] = trimStr(s, 20)
	}
	if in.NeedPrestige != nil {
		if *in.NeedPrestige < 0 {
			resp.ParamError(c, "需要声望不能为负")
			return
		}
		updates["need_prestige"] = *in.NeedPrestige
	}
	if in.CityMax != nil {
		if *in.CityMax < 1 {
			resp.ParamError(c, "可建城数至少为 1")
			return
		}
		updates["city_max"] = *in.CityMax
	}
	if in.Des != "" {
		updates["des"] = trimStr(in.Des, 200)
	}
	// 晋升宝物（前端提交数组，空数组 = 该军衔晋升无需宝物）
	if in.Treasures != nil {
		if b, err := json.Marshal(in.Treasures); err == nil {
			updates["treasures"] = string(b)
		}
	}
	if len(updates) == 0 {
		resp.ParamError(c, "无可修改字段")
		return
	}
	if err := h.DB.Model(&model.EzfyCfgRank{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		resp.ParamError(c, "修改失败："+err.Error())
		return
	}
	h.ezfyReload()
	resp.OK(c, gin.H{"msg": "军衔「" + r.Name + "」已保存"})
}

// AdminEzfyRankReset 恢复内置默认军衔表（含晋升宝物需求）
func (h *EzfyAdmin) AdminEzfyRankReset(c *gin.Context) {
	for _, d := range ezfyDefaultRanks() {
		updates := map[string]interface{}{
			"name": d.Name, "post": d.Post,
			"need_prestige": d.NeedPrestige, "city_max": d.CityMax,
		}
		// 宝物需求一并恢复内置默认（无内置默认的军衔清空，如列兵无需宝物）
		if reqs, ok := ezfyRankTreasures[d.ID]; ok && len(reqs) > 0 {
			if b, err := json.Marshal(reqs); err == nil {
				updates["treasures"] = string(b)
			}
		} else {
			updates["treasures"] = ""
		}
		h.DB.Model(&model.EzfyCfgRank{}).Where("id = ?", d.ID).Updates(updates)
	}
	h.ezfyReload()
	resp.OK(c, gin.H{"msg": "军衔表已恢复默认（含晋升宝物）"})
}

// ============ 玩家军衔 ============

// AdminEzfyRankPlayers 玩家军衔列表（含可建城数 / 实际城数）
func (h *EzfyAdmin) AdminEzfyRankPlayers(c *gin.Context) {
	page, offset, size := pageOf(c, 20)
	word := strings.TrimSpace(c.Query("word"))
	q := h.DB.Model(&model.EzfyProfile{})
	if word != "" {
		if n, err := strconv.Atoi(word); err == nil {
			q = q.Where("user_id = ? OR game_uid = ?", n, n)
		} else {
			q = q.Where("nickname LIKE ?", "%"+word+"%")
		}
	}
	var total int64
	q.Count(&total)
	var rows []model.EzfyProfile
	q.Order("prestige DESC").Offset(offset).Limit(size).Find(&rows)

	out := make([]gin.H, 0, len(rows))
	for _, p := range rows {
		var cityCount int64
		h.DB.Model(&model.EzfyCity{}).Where("user_id = ?", p.UserID).Count(&cityCount)
		name, homeNum := h.ezfyAdminName(p.UserID)
		lv := ezfyProfileRank(&p)
		rank := ezfyRankAt(lv)
		out = append(out, gin.H{
			"user_id": p.UserID, "game_uid": p.GameUID,
			"nickname": p.Nickname, "player_name": name, "home_num": homeNum,
			"prestige": p.Prestige,
			"rank_id":  rank.ID, "rank_name": rank.Name, "rank_post": rank.Post,
			"city_max": ezfyRankCityMaxAt(lv), "city_count": cityCount,
			"over_limit": cityCount > int64(ezfyRankCityMaxAt(lv)),
		})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// AdminEzfyRankSetPlayer 把玩家设为指定军衔（写声望为该军衔门槛）
func (h *EzfyAdmin) AdminEzfyRankSetPlayer(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("id"))
	var in struct {
		RankID int `json:"rank_id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.RankID <= 0 {
		resp.ParamError(c, "请选择军衔")
		return
	}
	var p model.EzfyProfile
	if err := h.DB.Where("user_id = ?", uid).First(&p).Error; err != nil {
		resp.NotFound(c, "玩家不存在")
		return
	}
	rank := ezfyRankOf(0)
	found := false
	for _, r := range ezfyCfg.rankList() {
		if r.ID == in.RankID {
			rank, found = r, true
			break
		}
	}
	if !found {
		resp.ParamError(c, "军衔不存在")
		return
	}
	// ★ 2026-09-28 军衔改为声望+宝物手动晋升：管理端直接设军衔时同步写 rank，
	//   否则玩家展示仍停在旧军衔（prestige 只是前提不再自动晋升）。
	h.DB.Model(&model.EzfyProfile{}).Where("id = ?", p.ID).
		Updates(map[string]interface{}{"prestige": rank.NeedPrestige, "rank": rank.ID})
	resp.OK(c, gin.H{"msg": "已将军衔设为「" + rank.Name + "」（声望 " +
		strconv.Itoa(rank.NeedPrestige) + "，可建城 " + strconv.Itoa(rank.CityMax) + " 座）"})
}

// AdminEzfyRankSetPrestige 直接设置玩家声望
func (h *EzfyAdmin) AdminEzfyRankSetPrestige(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("id"))
	var in struct {
		Prestige int `json:"prestige"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.Prestige < 0 {
		resp.ParamError(c, "请填写不小于 0 的声望")
		return
	}
	var p model.EzfyProfile
	if err := h.DB.Where("user_id = ?", uid).First(&p).Error; err != nil {
		resp.NotFound(c, "玩家不存在")
		return
	}
	h.DB.Model(&model.EzfyProfile{}).Where("id = ?", p.ID).Update("prestige", in.Prestige)
	p.Prestige = in.Prestige // 同步内存值，方便下面按新声望推导军衔名（老玩家 Rank=0 回落声望）
	resp.OK(c, gin.H{"msg": "声望已设为 " + strconv.Itoa(in.Prestige) +
		"，当前军衔「" + ezfyRankNameAt(ezfyProfileRank(&p)) + "」"})
}

// AdminEzfyPvpRank 管理端风云榜查询：?week=自然周周串（不传 = 本周）。
//
// ★ 2026-10-14 改为「排行榜 → 风云榜」tab：
//   - enabled：风云榜统计开关（管理端可开关）；
//   - ranks：指定周全量榜单（**积分降序**，已剔除「不参与排名」名单）；
//   - excluded：当前排除名单；
//   - weeks：最近 12 个有数据的周。
func (h *EzfyAdmin) AdminEzfyPvpRank(c *gin.Context) {
	week := c.Query("week")
	if week == "" {
		week = ezfyPvpWeekOf(time.Now())
	}
	// ★ 与玩家端同源：管理端查询也走缓存里的开关（先确保配置已加载）
	h.ezfyH().cfgs()
	var weeks []string
	h.DB.Model(&model.EzfyPvpWeek{}).Distinct("week").Order("week DESC").Limit(12).Pluck("week", &weeks)

	rows := ezfyPvpRankRows(h.DB, week, 0)
	ranks := make([]gin.H, 0, len(rows))
	for i, r := range rows {
		ranks = append(ranks, gin.H{
			"rank": i + 1, "user_id": r.UserID, "name": r.Nickname, "corps": r.Corps,
			"points": r.Points, "battles": r.Battles, "wins": r.Wins, "kills": r.Kills, "losses": r.Losses,
			"win_rate": pvpWinRate(r.Wins, r.Battles),
		})
	}
	resp.OK(c, gin.H{
		"week": week, "weeks": weeks, "ranks": ranks,
		"enabled":   ezfyPvpRankOn(),
		"lock_week": ezfyPvpRankLockWeek(),
		"excluded":  pvpExcludedRows(h.DB, ezfyBoardPvp),
	})
}

// AdminEzfyPvpRankLock 锁分 / 解锁风云榜（POST /admin/ezfy-pvp-rank/lock，body: {week, lock}）。
//
// ★ 2026-10-18 用户需求「风云榜能锁分，锁住了就相当于终止了不会再加」：
//
//	锁住某周 = 该周风云榜终止竞争，战绩不再累计积分（榜单冻结在锁定前）。
//	lock=1 且带 week → 锁定该周；lock=0 → 解除锁定。解锁后该周恢复累计。
func (h *EzfyAdmin) AdminEzfyPvpRankLock(c *gin.Context) {
	var in struct {
		Week string `json:"week"`
		Lock int    `json:"lock"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || (in.Lock != 0 && in.Lock != 1) {
		resp.ParamError(c, "参数错误（lock 只能是 0 或 1）")
		return
	}
	lockWeek := ""
	if in.Lock == 1 {
		lockWeek = strings.TrimSpace(in.Week)
		if lockWeek == "" {
			lockWeek = ezfyPvpWeekOf(time.Now())
		}
	}
	if err := h.DB.Model(&model.EzfyCfgLimit{}).Where("id = 1").Update("pvp_rank_lock_week", lockWeek).Error; err != nil {
		resp.ParamError(c, "保存失败："+err.Error())
		return
	}
	h.ezfyReload()
	if in.Lock == 1 {
		resp.OK(c, gin.H{"msg": "已锁分【" + lockWeek + "】：该周战绩不再累计积分，榜单冻结在锁定前", "week": lockWeek})
		return
	}
	resp.OK(c, gin.H{"msg": "已解除锁分，该周恢复累计积分", "week": ""})
}

// AdminEzfyPvpRankToggle 开关风云榜统计（POST /admin/ezfy-pvp-rank/toggle，body: {on: 0|1}）。
//
// ★ 2026-10-14 用户需求「风云榜 tab 能够选择开启关闭，开启就是现在这样，关闭就是不进行统计」。
//
//	关闭后战斗结算不再写榜（历史数据保留）；重新开启即继续累计。
func (h *EzfyAdmin) AdminEzfyPvpRankToggle(c *gin.Context) {
	var in struct {
		On *int `json:"on"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.On == nil || (*in.On != 0 && *in.On != 1) {
		resp.ParamError(c, "参数错误（on 只能是 0 或 1）")
		return
	}
	if err := h.DB.Model(&model.EzfyCfgLimit{}).Where("id = 1").Update("pvp_rank_on", *in.On).Error; err != nil {
		resp.ParamError(c, "保存失败："+err.Error())
		return
	}
	// 立即生效（重载配置缓存，别的节点靠周期刷新收敛）
	h.ezfyReload()
	if *in.On == 1 {
		resp.OK(c, gin.H{"msg": "风云榜统计已开启", "on": 1})
		return
	}
	resp.OK(c, gin.H{"msg": "风云榜统计已关闭（不再统计新的玩家对战战绩）", "on": 0})
}

// AdminEzfyPvpExclude POST /admin/ezfy-pvp-rank/exclude，body: {user_id, reason, board}。
//
// ★ 2026-10-14 风云榜「操作列」：[不参与排名] —— 把该玩家加入排除名单，
//
//	榜单不再展示 TA 的名次（历史战绩数据保留，从名单移除即恢复）。
//
// ★ 2026-10-16 用户需求「风云榜和战力榜、军衔声望榜都可以单独不参与排名」：
//
//	加 board（pvp/power/prestige）→ 三个榜各存一份**互不影响**的名单；
//	不传 board 时按 pvp 处理（兼容老调用）。
func (h *EzfyAdmin) AdminEzfyPvpExclude(c *gin.Context) {
	var in struct {
		UserID uint   `json:"user_id"`
		Reason string `json:"reason"`
		Board  string `json:"board"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.UserID == 0 {
		resp.ParamError(c, "请选择玩家")
		return
	}
	board := strings.TrimSpace(in.Board)
	if board == "" {
		board = ezfyBoardPvp
	}
	if board != ezfyBoardPvp && board != ezfyBoardPower && board != ezfyBoardPrestige {
		resp.ParamError(c, "榜单类型不正确")
		return
	}
	// 昵称：优先取二战档案（写榜用的就是它），没有则从家园库补
	nick := ""
	var p model.EzfyProfile
	if err := h.DB.Where("user_id = ?", in.UserID).First(&p).Error; err == nil {
		nick = p.Nickname
	}
	if nick == "" {
		nick, _ = h.ezfyAdminName(in.UserID)
	}
	reason := trimStr(strings.TrimSpace(in.Reason), 100)
	// upsert：重复标记只更新原因/昵称，不报错（唯一键 = board + user_id）
	if err := h.DB.Exec(`INSERT INTO ezfy_pvp_exclude (user_id, board, nickname, reason, created_at, updated_at)
VALUES (?, ?, ?, ?, NOW(), NOW())
ON DUPLICATE KEY UPDATE nickname = VALUES(nickname), reason = VALUES(reason), updated_at = VALUES(updated_at)`,
		in.UserID, board, nick, reason).Error; err != nil {
		resp.ParamError(c, "操作失败："+err.Error())
		return
	}
	shown := nick
	if shown == "" {
		shown = strconv.Itoa(int(in.UserID))
	}
	resp.OK(c, gin.H{"msg": "已把「" + shown + "」设为「" + ezfyBoardName(board) + "」不参与排名（该榜不再展示 TA）"})
}

// AdminEzfyPvpExcludeRemove DELETE /admin/ezfy-pvp-rank/exclude/:id —— 从排除名单移除（该榜恢复参与排名）。
//
// ★ 2026-10-16 三榜名单独立：删的是**这一条**（board+user_id），只让该玩家在**这一榜**恢复，
//
//	另外两个榜若也有记录则不受影响（前端按当前 tab 的名单展示，不会误删别的榜）。
func (h *EzfyAdmin) AdminEzfyPvpExcludeRemove(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var row model.EzfyPvpExclude
	if err := h.DB.First(&row, id).Error; err != nil {
		resp.NotFound(c, "该玩家不在排除名单里")
		return
	}
	if err := h.DB.Delete(&model.EzfyPvpExclude{}, id).Error; err != nil {
		resp.ParamError(c, "操作失败："+err.Error())
		return
	}
	shown := row.Nickname
	if shown == "" {
		shown = strconv.Itoa(int(row.UserID))
	}
	resp.OK(c, gin.H{"msg": "已恢复「" + shown + "」参与「" + ezfyBoardName(row.Board) + "」排名"})
}

// AdminEzfyPowerRank 管理端战力榜（GET /admin/ezfy-power-rank?page=&size=&word=）。
//
// ★ 2026-10-14 用户需求「排行榜下再设战力榜 tab」：**复用玩家端战力口径**
//
//	（科技 + 建筑 + 兵种，权重读 ezfy_cfg_limit），管理端做**全量分页**查看，
//	便于运营按玩家名/ID 检索。口径与玩家端 /rank 的「战力榜」完全一致。
func (h *EzfyAdmin) AdminEzfyPowerRank(c *gin.Context) {
	h.ezfyH().cfgs() // 战力权重在配置缓存里
	page, offset, size := pageOf(c, 20)
	word := strings.TrimSpace(c.Query("word"))

	lim := ezfyCfg.limit
	pow := lim.PowerTroopPow
	if pow <= 0 {
		pow = 0.8
	}
	cp := lim.PowerCompressPow
	if cp <= 0 {
		cp = 0.5
	}
	troopQuality := func(id int) float64 {
		if t := ezfyCfg.troop(id); t != nil {
			return float64(t.Health+t.Defence+t.AtkDef+t.AtkGround+t.AtkAir+t.Speed+t.AttackRange) / 100
		}
		return 10
	}
	// 城市 → 归属玩家 / 城市名
	var cities []model.EzfyCity
	h.DB.Find(&cities)
	cityUser := map[int64]int64{}
	cityName := map[int64]string{}
	for _, ct := range cities {
		cityUser[int64(ct.ID)] = int64(ct.UserID)
		cityName[int64(ct.ID)] = ct.Name
	}
	// 兵种战力（按玩家所有城市聚合，含城防 type=4）
	var troops []model.EzfyCityTroop
	h.DB.Find(&troops)
	userTroopScore := map[int64]float64{}
	userTroopTypes := map[int64]map[int]bool{}
	for _, t := range troops {
		if t.Count <= 0 {
			continue
		}
		uid := cityUser[t.CityId]
		if uid == 0 {
			continue
		}
		if userTroopTypes[uid] == nil {
			userTroopTypes[uid] = map[int]bool{}
		}
		if !userTroopTypes[uid][t.TroopId] {
			userTroopTypes[uid][t.TroopId] = true
			userTroopScore[uid] += float64(lim.PowerTroopType)
		}
		userTroopScore[uid] += math.Pow(float64(t.Count), pow) * troopQuality(t.TroopId)
	}
	// 建筑战力
	var builds []model.EzfyCityBuilding
	h.DB.Find(&builds)
	userBuildScore := map[int64]float64{}
	cityBuildSum := map[int64]int{}
	for _, b := range builds {
		uid := cityUser[b.CityId]
		if uid == 0 {
			continue
		}
		userBuildScore[uid] += float64(b.Level * lim.PowerBuildPerLevel)
		cityBuildSum[b.CityId] += b.Level
	}
	// 科技战力（用户级）
	var userTechs []model.EzfyUserTech
	h.DB.Find(&userTechs)
	userTechScore := map[int64]float64{}
	for _, ut := range userTechs {
		if ut.Level <= 0 {
			continue
		}
		userTechScore[int64(ut.UserId)] += float64(ut.Level*lim.PowerTechPerLevel) + float64(lim.PowerTechPerTech)
	}
	// 展示主城：建筑等级最高的城；没有建筑但有兵的玩家兜底取任一有兵城
	bestCity := map[int64]int64{}
	bestLv := map[int64]int{}
	for cid, lv := range cityBuildSum {
		uid := cityUser[cid]
		if lv > bestLv[uid] {
			bestLv[uid] = lv
			bestCity[uid] = cid
		}
	}
	for _, t := range troops {
		if t.Count <= 0 {
			continue
		}
		uid := cityUser[t.CityId]
		if uid == 0 {
			continue
		}
		if _, ok := bestCity[uid]; !ok {
			bestCity[uid] = t.CityId
		}
	}
	type pw struct {
		uid   int64
		power float64
		tech  float64
		build float64
		troop float64
	}
	all := map[int64]*pw{}
	uidSet := map[int64]bool{}
	for uid := range userTechScore {
		uidSet[uid] = true
	}
	for uid := range userBuildScore {
		uidSet[uid] = true
	}
	for uid := range userTroopScore {
		uidSet[uid] = true
	}
	for uid := range uidSet {
		techRaw, buildRaw, troopRaw := userTechScore[uid], userBuildScore[uid], userTroopScore[uid]
		raw := techRaw + buildRaw + troopRaw
		e := &pw{uid: uid, power: math.Pow(raw, cp)}
		if raw > 0 {
			t2 := math.Pow(techRaw, cp)
			b2 := math.Pow(buildRaw, cp)
			o2 := math.Pow(troopRaw, cp)
			sum2 := t2 + b2 + o2
			if sum2 > 0 {
				e.tech = e.power * t2 / sum2
				e.build = e.power * b2 / sum2
				e.troop = e.power * o2 / sum2
			}
		}
		all[uid] = e
	}
	arr := make([]*pw, 0, len(all))
	for _, v := range all {
		arr = append(arr, v)
	}
	sort.Slice(arr, func(i, j int) bool { return arr[i].power > arr[j].power })

	// ★ 2026-10-15 战力榜「不参与排名」；★ 2026-10-16 改为**只读战力榜自己那一份**名单
	//   （风云榜/军衔声望榜各有独立名单，互不影响）
	exSet := pvpExcludedSet(h.DB, ezfyBoardPower)
	ranked := make([]*pw, 0, len(arr))
	for _, e := range arr {
		if exSet[uint(e.uid)] {
			continue
		}
		ranked = append(ranked, e)
	}

	// 昵称：一次批量查档案（避免逐条 ensureProfile 的 N 次跨库往返）
	uidList := make([]uint, 0, len(ranked))
	for _, e := range ranked {
		uidList = append(uidList, uint(e.uid))
	}
	nickByUID := map[uint]string{}
	if len(uidList) > 0 {
		var ps []model.EzfyProfile
		h.DB.Where("user_id IN ?", uidList).Find(&ps)
		for _, p := range ps {
			nickByUID[p.UserID] = p.Nickname
		}
	}
	// 关键词过滤（按昵称或 user_id）—— 在排序后过滤，名次保持全服口径
	hit := func(e *pw) bool {
		if word == "" {
			return true
		}
		if n, err := strconv.Atoi(word); err == nil {
			return e.uid == int64(n)
		}
		return strings.Contains(nickByUID[uint(e.uid)], word)
	}
	filtered := make([]*pw, 0, len(ranked))
	for _, e := range ranked {
		if hit(e) {
			filtered = append(filtered, e)
		}
	}
	total := int64(len(filtered))
	start := offset
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + size
	if end > len(filtered) {
		end = len(filtered)
	}
	// 名次 = 在全服排序（已剔除「不参与排名」玩家）里的位置（过滤关键词也不改变原始名次口径）
	rankOf := map[int64]int{}
	for i, e := range ranked {
		rankOf[e.uid] = i + 1
	}
	out := make([]gin.H, 0, end-start)
	for _, e := range filtered[start:end] {
		out = append(out, gin.H{
			"rank": rankOf[e.uid], "user_id": uint(e.uid), "name": nickByUID[uint(e.uid)],
			"city_name": cityName[bestCity[e.uid]],
			"power":     int64(e.power), "tech_power": int64(e.tech),
			"build_power": int64(e.build), "troop_power": int64(e.troop),
		})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size,
		"excluded": pvpExcludedRows(h.DB, ezfyBoardPower),
		"usage":    "战力 = 科技 + 建筑 + 兵种（权重在「二战系统配置」可调），与游戏内战力榜同口径"})
}

// AdminEzfyPrestigeRank 管理端军衔声望榜（GET /admin/ezfy-prestige-rank?page=&size=&word=）。
//
// ★ 2026-10-15 用户需求「管理系统 → 排行榜：把军衔声望榜也放到这里面，然后也可以不参与排名」：
//   - 口径与玩家端 /rank 的「军衔声望榜」一致 —— 按声望(prestige)降序，展示当前军衔名；
//   - 剔除「不参与排名」名单；★ 2026-10-16 起只读**军衔声望榜自己那一份**（三榜名单互相独立）；
//   - word 支持昵称模糊 / 用户ID 精确（过滤后名次仍保持全服口径，与战力榜一致）。
func (h *EzfyAdmin) AdminEzfyPrestigeRank(c *gin.Context) {
	ezfyCfg.load(h.DB) // 军衔名从配置缓存读（与玩家端口径一致）
	page, offset, size := pageOf(c, 20)
	word := strings.TrimSpace(c.Query("word"))

	// 全量声望排序（玩家端同样按 prestige DESC）；剔除「不参与排名」名单后重排
	var all []model.EzfyProfile
	h.DB.Order("prestige DESC").Find(&all)
	exSet := pvpExcludedSet(h.DB, ezfyBoardPrestige)
	ranked := make([]model.EzfyProfile, 0, len(all))
	for _, p := range all {
		if exSet[p.UserID] {
			continue
		}
		ranked = append(ranked, p)
	}
	// 关键词过滤（昵称模糊 / 用户ID 精确）—— 不改变全服名次口径
	hit := func(p model.EzfyProfile) bool {
		if word == "" {
			return true
		}
		if p.Nickname != "" {
			return strings.Contains(p.Nickname, word)
		}
		// 无昵称的玩家：按用户ID（user_id / game_uid）精确匹配
		return strconv.FormatUint(uint64(p.UserID), 10) == word ||
			strconv.FormatInt(p.GameUID, 10) == word
	}
	// ★ word 为纯数字时按 user_id 精确命中（与战力榜一致）
	if n, err := strconv.Atoi(word); err == nil && word != "" {
		filtered := make([]model.EzfyProfile, 0, len(ranked))
		for _, p := range ranked {
			if p.UserID == uint(n) {
				filtered = append(filtered, p)
			}
		}
		ranked = filtered
	} else {
		filtered := make([]model.EzfyProfile, 0, len(ranked))
		for _, p := range ranked {
			if hit(p) {
				filtered = append(filtered, p)
			}
		}
		ranked = filtered
	}
	total := int64(len(ranked))
	start := offset
	if start > len(ranked) {
		start = len(ranked)
	}
	end := start + size
	if end > len(ranked) {
		end = len(ranked)
	}
	out := make([]gin.H, 0, end-start)
	for i, p := range ranked[start:end] {
		lv := ezfyProfileRank(&p)
		r := ezfyRankAt(lv)
		var cityCount int64
		h.DB.Model(&model.EzfyCity{}).Where("user_id = ?", p.UserID).Count(&cityCount)
		out = append(out, gin.H{
			"rank": start + i + 1, "user_id": p.UserID,
			"name":        p.Nickname,
			"prestige":    p.Prestige,
			"rank_id":     r.ID,
			"rank_name":   r.Name,
			"rank_post":   r.Post,
			"city_max":    ezfyRankCityMaxAt(lv),
			"city_count":  cityCount,
			"is_excluded": exSet[p.UserID],
		})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size,
		"excluded": pvpExcludedRows(h.DB, ezfyBoardPrestige),
		"usage":    "与游戏内「军衔声望榜」同口径（按声望降序）；已标记「不参与排名」的玩家不参与名次，名次为剔除后的全服排名"})
}

// ============ 风云榜奖励发放（★ 2026-10-18） ============
//
// 用户需求（原话）：
//
//	「风云榜 开发一个发放宝箱功能：给第一名/第二名/第三名发宝箱；给参与者发宝箱；
//	 给第一二三名发道具；给参与者发道具；给第一二三名发钻石；给参与者发钻石。
//	 发放不是直接进玩家背包，相当于系统给玩家发邮件，玩家领取，7 天不领取默认无效。
//	 风云榜发东西会加修饰就是『参与风云榜获得的』，普通数据管理的发放就是『系统发放的』」
//
// 实现要点：
//   - 奖励**只写邮件**（ezfy_mail + ezfy_mail_attach），玩家在邮箱点「领取」才到账；
//   - 档位四档：rank1 / rank2 / rank3 / join（参与奖）；
//   - 参与奖范围可设：join_limit=N 表示**第 4 ~ 第 N 名**（截止名次）；join_limit=0 表示第 4 名及以后**全部参战玩家**；
//   - **同一个人同一周只发一次**（ezfy_pvp_reward 唯一索引），重复点只补发还没发的人。

// ezfyRewardTier 一档奖励配置（前端弹窗填什么，这里就收什么）。
type ezfyRewardTier struct {
	Diamond int64 `json:"diamond"`
	Chests  []struct {
		ChestID int `json:"chest_id"`
		Count   int `json:"count"`
	} `json:"chests"`
	Items []struct {
		CfgID int `json:"cfg_id"`
		Count int `json:"count"`
	} `json:"items"`
}

// ezfyRewardTierEmpty 该档是否什么都没配（空档跳过，不发空邮件）。
func ezfyRewardTierEmpty(t ezfyRewardTier) bool {
	if t.Diamond > 0 {
		return false
	}
	for _, c := range t.Chests {
		if c.ChestID > 0 && c.Count > 0 {
			return false
		}
	}
	for _, it := range t.Items {
		if it.CfgID > 0 && it.Count > 0 {
			return false
		}
	}
	return true
}

// ezfyRewardAttaches 把一档配置解析成附件行（同时校验配置是否存在，返回错误文案）。
func (h *EzfyAdmin) ezfyRewardAttaches(t ezfyRewardTier) ([]model.EzfyMailAttach, error) {
	out := make([]model.EzfyMailAttach, 0, len(t.Chests)+len(t.Items)+1)
	if t.Diamond > 0 {
		out = append(out, model.EzfyMailAttach{Kind: ezfyAttachDiamond, Num: t.Diamond, Name: "钻石"})
	}
	for _, c := range t.Chests {
		if c.ChestID <= 0 || c.Count <= 0 {
			continue
		}
		var cfg model.EzfyCfgChest
		if err := h.DB.First(&cfg, c.ChestID).Error; err != nil {
			return nil, fmt.Errorf("宝箱不存在：%d", c.ChestID)
		}
		out = append(out, model.EzfyMailAttach{Kind: ezfyAttachChest, CfgID: c.ChestID, Num: int64(c.Count), Name: cfg.Name})
	}
	for _, it := range t.Items {
		if it.CfgID <= 0 || it.Count <= 0 {
			continue
		}
		var cfg model.EzfyCfgItem
		if err := h.DB.First(&cfg, it.CfgID).Error; err != nil {
			return nil, fmt.Errorf("道具不存在：%d", it.CfgID)
		}
		out = append(out, model.EzfyMailAttach{Kind: ezfyAttachItem, CfgID: it.CfgID, Num: int64(it.Count), Name: cfg.Name})
	}
	return out, nil
}

// AdminEzfyPvpRewardStatus GET /admin/ezfy-pvp-rank/reward?week= —— 该周已发放情况（弹窗打开时预览）
func (h *EzfyAdmin) AdminEzfyPvpRewardStatus(c *gin.Context) {
	week := strings.TrimSpace(c.Query("week"))
	if week == "" {
		week = ezfyPvpWeekOf(time.Now())
	}
	rows := ezfyPvpRankRows(h.DB, week, 0)
	var sent []model.EzfyPvpReward
	h.DB.Where("week = ?", week).Order("rank_no ASC, id ASC").Find(&sent)
	sentSet := make(map[uint]bool, len(sent))
	// ★ 2026-10-18 「本期已发放奖励」明细：已发玩家昵称 + 档位 + 时间，页面下方展示
	type sentOne struct {
		UserID    uint      `json:"user_id"`
		Name      string    `json:"name"`
		RankNo    int       `json:"rank_no"`
		Tier      string    `json:"tier"`
		TierName  string    `json:"tier_name"`
		CreatedAt time.Time `json:"created_at"`
	}
	records := make([]sentOne, 0, len(sent))
	nickOf := map[uint]string{}
	for _, s := range sent {
		sentSet[s.UserID] = true
		nickOf[s.UserID] = ""
	}
	if len(nickOf) > 0 {
		uids := make([]uint, 0, len(nickOf))
		for uid := range nickOf {
			uids = append(uids, uid)
		}
		var profs []model.EzfyProfile
		h.DB.Select("user_id, nickname").Where("user_id IN ?", uids).Find(&profs)
		for _, p := range profs {
			nickOf[p.UserID] = p.Nickname
		}
		for _, s := range sent {
			records = append(records, sentOne{
				UserID: s.UserID, Name: nickOf[s.UserID], RankNo: s.RankNo,
				Tier: s.Tier, TierName: ezfyPvpRewardTierName(s.Tier), CreatedAt: s.CreatedAt,
			})
		}
	}
	type one struct {
		UserID   uint   `json:"user_id"`
		Name     string `json:"name"`
		Points   int64  `json:"points"`
		Battles  int64  `json:"battles"`
		Sent     bool   `json:"sent"`
		Eligible bool   `json:"eligible"` // 积分 > 0 = 有效参与，才发奖
	}
	list := make([]one, 0, len(rows))
	eligible := 0
	for _, r := range rows {
		ok := r.Points > 0
		if ok {
			eligible++
		}
		list = append(list, one{UserID: r.UserID, Name: r.Nickname, Points: r.Points,
			Battles: r.Battles, Sent: sentSet[r.UserID], Eligible: ok})
	}
	resp.OK(c, gin.H{"week": week, "total": len(rows), "eligible": eligible,
		"sent": len(sent), "list": list, "records": records})
}

// ezfyPvpRewardTierName 发放档位中文名（补发 / 已发明细展示用）。
func ezfyPvpRewardTierName(tier string) string {
	switch tier {
	case "rank1":
		return "第 1 名"
	case "rank2":
		return "第 2 名"
	case "rank3":
		return "第 3 名"
	case "join":
		return "参与奖"
	}
	return tier
}

// AdminEzfyPvpRewardRecords GET /admin/ezfy-pvp-rank/reward/records?week=&page=&size=
// —— 「本期已发放奖励」分页列表（★ 2026-10-18 局部刷新，含奖励明细 detail）
func (h *EzfyAdmin) AdminEzfyPvpRewardRecords(c *gin.Context) {
	week := strings.TrimSpace(c.Query("week"))
	if week == "" {
		week = ezfyPvpWeekOf(time.Now())
	}
	page, offset, size := pageOf(c, 10)
	q := h.DB.Model(&model.EzfyPvpReward{}).Where("week = ?", week)
	var total int64
	q.Count(&total)
	var rows []model.EzfyPvpReward
	q.Order("rank_no ASC, id ASC").Offset(offset).Limit(size).Find(&rows)
	nickOf := map[uint]string{}
	if len(rows) > 0 {
		uids := make([]uint, 0, len(rows))
		for _, r := range rows {
			uids = append(uids, r.UserID)
		}
		var profs []model.EzfyProfile
		h.DB.Select("user_id, nickname").Where("user_id IN ?", uids).Find(&profs)
		for _, p := range profs {
			nickOf[p.UserID] = p.Nickname
		}
	}
	out := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		out = append(out, gin.H{
			"user_id": r.UserID, "name": nickOf[r.UserID],
			"rank_no": r.RankNo, "tier": r.Tier, "tier_name": ezfyPvpRewardTierName(r.Tier),
			"detail": r.Detail, "created_at": r.CreatedAt,
		})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// ezfyRewardDetail 一档附件 → 明细摘要（管理端「已发放奖励」可见发了什么）。
func ezfyRewardDetail(as []model.EzfyMailAttach) string {
	parts := make([]string, 0, len(as))
	for _, a := range as {
		switch a.Kind {
		case ezfyAttachDiamond:
			parts = append(parts, fmt.Sprintf("钻石×%d", a.Num))
		case ezfyAttachChest:
			parts = append(parts, fmt.Sprintf("宝箱[%s]×%d", a.Name, a.Num))
		case ezfyAttachItem:
			parts = append(parts, fmt.Sprintf("道具[%s]×%d", a.Name, a.Num))
		}
	}
	return strings.Join(parts, "、")
}

// AdminEzfyPvpReward POST /admin/ezfy-pvp-rank/reward —— 一键发放本周风云榜奖励
//
// body:
//
//	{
//	  "week": "20261012-20261018",
//	  "expire_days": 7,
//	  "join_limit": 20,                       // 0 = 全部参战玩家（第4名及以后）
//	  "tiers": {"rank1": {...}, "rank2": {...}, "rank3": {...}, "join": {...}}
//	}
func (h *EzfyAdmin) AdminEzfyPvpReward(c *gin.Context) {
	var in struct {
		Week       string `json:"week"`
		ExpireDays int    `json:"expire_days"`
		JoinLimit  int    `json:"join_limit"`
		// ★ 2026-10-18 用户「发放的时候加个条件吧 积分大于多少」：
		//   最低积分门槛，当周积分 < MinPoints 的玩家不发奖（0 = 只要积分 > 0 就算有效参与）。
		MinPoints int64 `json:"min_points"`
		// ★ 2026-10-18 补发（用户「发过一次了，还想给参与者再发宝箱」）：
		//   resend=true 时**忽略已发幂等**，符合条件的玩家全部再发一次邮件；
		//   发放记录 upsert 更新为最新邮件（唯一键 week+user_id 不新增行）。
		Resend bool `json:"resend"`
		Tiers  struct {
			Rank1 ezfyRewardTier `json:"rank1"`
			Rank2 ezfyRewardTier `json:"rank2"`
			Rank3 ezfyRewardTier `json:"rank3"`
			Join  ezfyRewardTier `json:"join"`
		} `json:"tiers"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	week := strings.TrimSpace(in.Week)
	if week == "" {
		week = ezfyPvpWeekOf(time.Now())
	}
	expireDays := in.ExpireDays
	if expireDays <= 0 {
		expireDays = 7
	}
	if expireDays > 90 {
		resp.ParamError(c, "领取期限最多 90 天")
		return
	}
	if in.JoinLimit < 0 {
		in.JoinLimit = 0
	}
	if in.JoinLimit > 0 && in.JoinLimit < 4 {
		in.JoinLimit = 4 // 参与奖从第 4 名开始，截止名次不可能小于 4
	}
	if in.MinPoints < 0 {
		in.MinPoints = 0
	}

	// 每档先解析成附件（顺便校验配置是否存在）；空档不参与发放
	plans := map[string][]model.EzfyMailAttach{}
	for _, p := range []struct {
		key string
		cfg ezfyRewardTier
	}{
		{"rank1", in.Tiers.Rank1}, {"rank2", in.Tiers.Rank2},
		{"rank3", in.Tiers.Rank3}, {"join", in.Tiers.Join},
	} {
		if ezfyRewardTierEmpty(p.cfg) {
			continue
		}
		as, err := h.ezfyRewardAttaches(p.cfg)
		if err != nil {
			resp.ParamError(c, err.Error())
			return
		}
		if len(as) > 0 {
			plans[p.key] = as
		}
	}
	if len(plans) == 0 {
		resp.ParamError(c, "请至少给一个档位配置奖励（宝箱 / 道具 / 钻石）")
		return
	}

	rows := ezfyPvpRankRows(h.DB, week, 0)
	if len(rows) == 0 {
		resp.ParamError(c, "该周没有任何战绩，无法发放")
		return
	}
	// 已发名单（幂等）。★ 2026-10-18 补发模式（resend=true）跳过幂等：
	//   已发过的人也会再发一次（玩家邮件里多一封可领取的奖励）。
	var done []model.EzfyPvpReward
	h.DB.Where("week = ?", week).Find(&done)
	doneSet := make(map[uint]bool, len(done))
	for _, d := range done {
		doneSet[d.UserID] = true
	}

	drafts := make([]ezfyMailDraft, 0, len(rows))
	meta := make([]struct {
		uid    uint
		rank   int
		tier   string
		detail string
	}, 0, len(rows))
	skipped := 0
	zeroSkipped := 0
	for i, r := range rows {
		rankNo := i + 1
		// ★★ 2026-10-18 用户规则「积分 0 的不发奖，这是没有效参与」+「发放的时候加个条件 积分大于多少」：
		//   当周积分 <= 0 或低于最低积分门槛的一律跳过（连第 1/2/3 名也不发）——
		//   榜单里可能因为没人打而出现 0 分占前几名的情况，那种名次没有意义。
		//   ★ 名次仍按**原榜单名次**（不重排）：奖励邮件里写的「第 N 名」必须和榜单显示一致。
		if r.Points <= 0 || r.Points < in.MinPoints {
			zeroSkipped++
			continue
		}
		tier := ""
		switch {
		case rankNo == 1:
			tier = "rank1"
		case rankNo == 2:
			tier = "rank2"
		case rankNo == 3:
			tier = "rank3"
		default:
			// 参与奖：第 4 名及以后；join_limit>0 时它是**截止名次**（如 20 = 第 4~20 名）
			if in.JoinLimit > 0 && rankNo > in.JoinLimit {
				continue
			}
			tier = "join"
		}
		as, ok := plans[tier]
		if !ok {
			continue // 该档没配奖励
		}
		if doneSet[r.UserID] && !in.Resend {
			skipped++
			continue
		}
		subject := "风云榜奖励"
		var content string
		if tier == "join" {
			content = fmt.Sprintf("你在【%s】风云榜中累计参战 %d 场，这是你参与风云榜获得的参与奖，请在 %d 天内领取，过期作废。",
				week, r.Battles, expireDays)
		} else {
			content = fmt.Sprintf("恭喜你在【%s】风云榜中获得第 %d 名！这是你参与风云榜获得的奖励，请在 %d 天内领取，过期作废。",
				week, rankNo, expireDays)
		}
		drafts = append(drafts, ezfyMailDraft{
			ReceiverID: r.UserID, Subject: subject, Content: content, Attaches: as,
		})
		meta = append(meta, struct {
			uid    uint
			rank   int
			tier   string
			detail string
		}{r.UserID, rankNo, tier, ezfyRewardDetail(as)})
	}
	if len(drafts) == 0 {
		msg := fmt.Sprintf("本次没有可发放的玩家（已发过 %d 人，积分不达标跳过 %d 人）", skipped, zeroSkipped)
		if skipped == 0 {
			msg = fmt.Sprintf("本周没有达到积分门槛的玩家（跳过 %d 人），未发放", zeroSkipped)
		}
		resp.OK(c, gin.H{"msg": msg, "sent": 0, "skipped": skipped, "zero_skipped": zeroSkipped})
		return
	}

	batchNo, err := ezfyMailSendBatch(h.DB, drafts, expireDays)
	if err != nil {
		resp.ServerError(c, err)
		return
	}
	// 回查邮件 id，写发放记录（唯一索引兜底，重复点也插不进去）
	var saved []struct {
		ID         uint
		ReceiverId uint
	}
	h.DB.Model(&model.EzfyMail{}).Select("id, receiver_id").Where("batch_no = ?", batchNo).Scan(&saved)
	mailOf := make(map[uint]uint, len(saved))
	for _, s := range saved {
		mailOf[s.ReceiverId] = s.ID
	}
	now := time.Now()
	for _, m := range meta {
		// ★ 2026-10-18 补发：唯一键 week+user_id 已存在时更新为**最新一封**邮件
		//   （首次发放 INSERT；补发 UPDATE mail_id/rank_no/tier/detail/created_at，不新增行）
		h.DB.Exec(`INSERT INTO ezfy_pvp_reward (week, user_id, rank_no, tier, mail_id, detail, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE
rank_no = VALUES(rank_no), tier = VALUES(tier), mail_id = VALUES(mail_id), detail = VALUES(detail), created_at = VALUES(created_at)`,
			week, m.uid, m.rank, m.tier, mailOf[m.uid], m.detail, now)
	}
	if in.Resend {
		resp.OK(c, gin.H{
			"msg": fmt.Sprintf("风云榜奖励已补发：%d 人（补发模式已发过的人也再发了一次，跳过积分不达标的 %d 人），奖励已通过系统邮件送达，%d 天内未领取作废",
				len(drafts), zeroSkipped, expireDays),
			"sent": len(drafts), "skipped": skipped, "zero_skipped": zeroSkipped,
			"batch_no": batchNo, "week": week, "resend": true,
		})
		return
	}
	resp.OK(c, gin.H{
		"msg": fmt.Sprintf("风云榜奖励已发放：%d 人（跳过已发 %d 人、积分不达标的 %d 人），奖励已通过系统邮件送达，%d 天内未领取作废",
			len(drafts), skipped, zeroSkipped, expireDays),
		"sent": len(drafts), "skipped": skipped, "zero_skipped": zeroSkipped,
		"batch_no": batchNo, "week": week,
	})
}
