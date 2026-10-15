package ezfy

import (
	"encoding/json"
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
		"enabled":  ezfyPvpRankOn(),
		"excluded": pvpExcludedRows(h.DB),
	})
}

// AdminEzfyPvpRankToggle 开关风云榜统计（POST /admin/ezfy-pvp-rank/toggle，body: {on: 0|1}）。
//
// ★ 2026-10-14 用户需求「风云榜 tab 能够选择开启关闭，开启就是现在这样，关闭就是不进行统计」。
//   关闭后战斗结算不再写榜（历史数据保留）；重新开启即继续累计。
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

// AdminEzfyPvpExclude POST /admin/ezfy-pvp-rank/exclude，body: {user_id, reason}。
//
// ★ 2026-10-14 风云榜「操作列」：[不参与排名] —— 把该玩家加入排除名单，
//   榜单不再展示 TA 的名次（历史战绩数据保留，从名单移除即恢复）。
func (h *EzfyAdmin) AdminEzfyPvpExclude(c *gin.Context) {
	var in struct {
		UserID uint   `json:"user_id"`
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.UserID == 0 {
		resp.ParamError(c, "请选择玩家")
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
	// upsert：重复标记只更新原因/昵称，不报错
	if err := h.DB.Exec(`INSERT INTO ezfy_pvp_exclude (user_id, nickname, reason, created_at, updated_at)
VALUES (?, ?, ?, NOW(), NOW())
ON DUPLICATE KEY UPDATE nickname = VALUES(nickname), reason = VALUES(reason), updated_at = VALUES(updated_at)`,
		in.UserID, nick, reason).Error; err != nil {
		resp.ParamError(c, "操作失败："+err.Error())
		return
	}
	shown := nick
	if shown == "" {
		shown = strconv.Itoa(int(in.UserID))
	}
	resp.OK(c, gin.H{"msg": "已把「" + shown + "」设为不参与排名（榜单不再展示 TA）"})
}

// AdminEzfyPvpExcludeRemove DELETE /admin/ezfy-pvp-rank/exclude/:id —— 从排除名单移除（恢复参与排名）。
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
	resp.OK(c, gin.H{"msg": "已恢复「" + shown + "」参与排名"})
}

// AdminEzfyPowerRank 管理端战力榜（GET /admin/ezfy-power-rank?page=&size=&word=）。
//
// ★ 2026-10-14 用户需求「排行榜下再设战力榜 tab」：**复用玩家端战力口径**
//   （科技 + 建筑 + 兵种，权重读 ezfy_cfg_limit），管理端做**全量分页**查看，
//   便于运营按玩家名/ID 检索。口径与玩家端 /rank 的「战力榜」完全一致。
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

	// ★ 2026-10-15 战力榜「不参与排名」：与风云榜共用同一份排除名单（排除后重新排名，与风云榜口径一致）
	exSet := pvpExcludedSet(h.DB)
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
		"usage": "战力 = 科技 + 建筑 + 兵种（权重在「二战系统配置」可调），与游戏内战力榜同口径"})
}
