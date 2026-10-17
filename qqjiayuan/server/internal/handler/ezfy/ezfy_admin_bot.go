// 管理端 · 人机（AI 玩家）模块（★ 2026-10-17）
//
// 用户要求（人机模块方案 7.1）：人机管理要和玩家管理**彻底分开** ——
// 前端菜单/组件、后端路由/handler、权限点、数据（is_bot）五层全部独立，
// 日后维护人机模块不会碰到任何玩家管理代码，出问题能整块回滚。
//
// 路由前缀：/api/admin/ezfy-bot/*      权限点：module:ezfyBot
// 页面：admin-web/src/components/admin/AdminEzfyBot.vue
package ezfy

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// ezfyBotRow 人机列表一行（一次性批量补齐，不逐行查）
type ezfyBotRow struct {
	model.EzfyBot
	Nickname   string `json:"nickname"`
	Prestige   int    `json:"prestige"`
	Camp       int    `json:"camp"`
	GameUID    int64  `json:"game_uid"`
	CityID     uint   `json:"city_id"`
	CityName   string `json:"city_name"`
	X          int    `json:"x"`
	Y          int    `json:"y"`
	CityLevel  int    `json:"city_level"`
	TroopTotal int64  `json:"troop_total"`
}

// AdminEzfyBotList GET /admin/ezfy-bot/list —— 人机列表（一次全量，≤50 条）
//
// 与「玩家信息管理」完全独立：这里只出 is_bot=1 的人，那边只出 is_bot=0 的人。
func (h *EzfyAdmin) AdminEzfyBotList(c *gin.Context) {
	word := strings.TrimSpace(c.Query("word"))
	q := h.DB.Model(&model.EzfyBot{})
	if word != "" {
		// 昵称/用户ID 过滤（昵称在 profile，先拿 uid 再 IN，避免逐行子查询）
		if uid, err := strconv.Atoi(word); err == nil {
			q = q.Where("user_id = ?", uid)
		} else {
			var uids []uint
			h.DB.Model(&model.EzfyProfile{}).Where("nickname LIKE ? AND is_bot = 1", "%"+word+"%").
				Limit(200).Pluck("user_id", &uids)
			if len(uids) == 0 {
				resp.OK(c, gin.H{"list": []gin.H{}, "total": 0, "max": ezfyBotMaxCount})
				return
			}
			q = q.Where("user_id IN ?", uids)
		}
	}
	var bots []model.EzfyBot
	q.Order("id ASC").Limit(ezfyBotMaxCount + 10).Find(&bots)

	uids := make([]uint, 0, len(bots))
	for _, b := range bots {
		uids = append(uids, b.UserID)
	}
	profOf := map[uint]model.EzfyProfile{}
	cityOf := map[uint]model.EzfyCity{}
	troopOf := map[uint]int64{}
	if len(uids) > 0 {
		var profs []model.EzfyProfile
		h.DB.Select("user_id, nickname, prestige, camp, game_uid, current_city_id").
			Where("user_id IN ?", uids).Find(&profs)
		for _, p := range profs {
			profOf[p.UserID] = p
		}
		var cities []model.EzfyCity
		h.DB.Select("id, user_id, name, x, y, city_level").Where("user_id IN ?", uids).Find(&cities)
		for _, ct := range cities {
			// 优先 current_city_id，否则取第一座
			if p, ok := profOf[ct.UserID]; ok && int64(ct.ID) == p.CurrentCityId {
				cityOf[ct.UserID] = ct
			} else if _, exists := cityOf[ct.UserID]; !exists {
				cityOf[ct.UserID] = ct
			}
		}
		ids := make([]int64, 0, len(cityOf))
		for _, ct := range cityOf {
			ids = append(ids, int64(ct.ID))
		}
		if len(ids) > 0 {
			var sums []struct {
				CityID uint
				Total  int64
			}
			h.DB.Model(&model.EzfyCityTroop{}).Select("city_id, SUM(count) AS total").
				Where("city_id IN ?", ids).Group("city_id").Scan(&sums)
			cityIDOf := map[uint]uint{}
			for uid, ct := range cityOf {
				cityIDOf[ct.ID] = uid
			}
			for _, s := range sums {
				if uid, ok := cityIDOf[s.CityID]; ok {
					troopOf[uid] = s.Total
				}
			}
		}
	}

	now := time.Now().UnixMilli()
	out := make([]gin.H, 0, len(bots))
	for _, b := range bots {
		p := profOf[b.UserID]
		ct := cityOf[b.UserID]
		out = append(out, gin.H{
			"user_id": b.UserID, "nickname": p.Nickname, "game_uid": p.GameUID,
			"level": b.Level, "level_text": ezfyBotLevelText(b.Level),
			"persona": b.Persona, "enabled": b.Enabled,
			"prestige": p.Prestige, "camp": p.Camp, "camp_name": ezfyCampName(p.Camp),
			"city_id": ct.ID, "city_name": ct.Name, "x": ct.X, "y": ct.Y, "city_level": ct.CityLevel,
			"troop_total": troopOf[b.UserID],
			"last_act_at": b.LastActAt, "last_action": b.LastAction,
			"next_act_at":     b.NextActAt,
			"status_text":     ezfyBotStatusText(b, now),
			"atk_count_today": b.AtkCountToday, "atk_date": b.AtkDate,
		})
	}
	resp.OK(c, gin.H{
		"list": out, "total": len(out), "max": ezfyBotMaxCount,
		"attack_on": ezfyBotAttackOn(), "atk_daily_cap": ezfyBotDailyAtkCap(),
	})
}

// AdminEzfyBotCreate POST /admin/ezfy-bot/create {count, level} —— 批量生成人机
//
// count 单次最多 10（避免一次建太多把 1 核打满）；level 1新手/2普通/3精英。
func (h *EzfyAdmin) AdminEzfyBotCreate(c *gin.Context) {
	var in struct {
		Count int  `json:"count"`
		Level int  `json:"level"`
		Boost bool `json:"boost"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if in.Count < 1 || in.Count > 10 {
		resp.ParamError(c, "单次生成 1~10 个")
		return
	}
	if in.Level < 1 || in.Level > 3 {
		in.Level = 2
	}
	// 生成走 EzfyHandler（建城/建档案的懒结算都在那边）
	ez := &EzfyHandler{DB: h.DB, HomeDB: h.HomeDB}
	ez.cfgs()
	created, failed := 0, 0
	var names []string
	for i := 0; i < in.Count; i++ {
		uid, err := ez.ezfyBotProvision(in.Level, in.Boost)
		if err != nil {
			failed++
			continue
		}
		created++
		names = append(names, ez.ezfyBotNickOf(uid))
	}
	msg := "已生成 " + strconv.Itoa(created) + " 个人机"
	if len(names) > 0 {
		msg += "：" + strings.Join(names, "、")
	}
	if failed > 0 {
		msg += "（" + strconv.Itoa(failed) + " 个失败，可能已达数量上限）"
	}
	resp.OK(c, gin.H{"msg": msg, "created": created, "failed": failed})
}

// AdminEzfyBotToggle POST /admin/ezfy-bot/:uid/toggle {enabled} —— 单个启停
func (h *EzfyAdmin) AdminEzfyBotToggle(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("uid"))
	var in struct {
		Enabled int `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || uid <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	if in.Enabled != 0 {
		in.Enabled = 1
	}
	res := h.DB.Model(&model.EzfyBot{}).Where("user_id = ?", uid).Update("enabled", in.Enabled)
	if res.RowsAffected == 0 {
		resp.NotFound(c, "人机不存在")
		return
	}
	if in.Enabled == 1 {
		// 重新启用时立刻可行动，不用等下一轮
		h.DB.Model(&model.EzfyBot{}).Where("user_id = ?", uid).Update("next_act_at", time.Now().UnixMilli())
		resp.OK(c, gin.H{"msg": "已启用"})
		return
	}
	resp.OK(c, gin.H{"msg": "已暂停"})
}

// AdminEzfyBotLevel POST /admin/ezfy-bot/:uid/level {level} —— 调整难度档
func (h *EzfyAdmin) AdminEzfyBotLevel(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("uid"))
	var in struct {
		Level int `json:"level"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || uid <= 0 || in.Level < 1 || in.Level > 3 {
		resp.ParamError(c, "参数错误（难度 1~3）")
		return
	}
	res := h.DB.Model(&model.EzfyBot{}).Where("user_id = ?", uid).Updates(map[string]interface{}{
		"level": in.Level, "persona": ezfyBotPersona(in.Level),
	})
	if res.RowsAffected == 0 {
		resp.NotFound(c, "人机不存在")
		return
	}
	resp.OK(c, gin.H{"msg": "难度已调整为「" + ezfyBotLevelText(in.Level) + "」"})
}

// AdminEzfyBotAct POST /admin/ezfy-bot/:uid/act —— 立即行动一次（排障：看它这轮做了什么）
func (h *EzfyAdmin) AdminEzfyBotAct(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("uid"))
	if uid <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var b model.EzfyBot
	if err := h.DB.Where("user_id = ?", uid).First(&b).Error; err != nil {
		resp.NotFound(c, "人机不存在")
		return
	}
	ez := &EzfyHandler{DB: h.DB, HomeDB: h.HomeDB, Secret: ezfyBotAdminSecret, ExpH: 1}
	ez.cfgs()
	ez.botTickOne(b)
	// 回读最新动作
	h.DB.Where("user_id = ?", uid).First(&b)
	resp.OK(c, gin.H{"msg": "已执行一轮：" + b.LastAction, "last_action": b.LastAction})
}

// ezfyBotAdminSecret 管理端「立即行动」用的 JWT 密钥。
//
// ★ 由 router.Setup 注入（与 cfg.Jwt.Secret 一致）—— 管理端 handler 本身不持有 Secret，
//
//	但「立即行动」要替人机签 token 打内部接口，必须能拿到。
var ezfyBotAdminSecret = ""

// SetEzfyBotAdminSecret 由 router.Setup 调用。
func SetEzfyBotAdminSecret(secret string) { ezfyBotAdminSecret = secret }

// AdminEzfyBotDelete DELETE /admin/ezfy-bot/:uid —— 删除单个人机（连根拔掉）
func (h *EzfyAdmin) AdminEzfyBotDelete(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("uid"))
	if uid <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var b model.EzfyBot
	if err := h.DB.Where("user_id = ?", uid).First(&b).Error; err != nil {
		resp.NotFound(c, "人机不存在")
		return
	}
	nick := ""
	{
		var p model.EzfyProfile
		h.DB.Select("nickname").Where("user_id = ?", uid).First(&p)
		nick = p.Nickname
	}
	if err := h.ezfyBotPurgeOne(uint(uid)); err != nil {
		resp.ServerError(c, err)
		return
	}
	resp.OK(c, gin.H{"msg": "已删除人机「" + nick + "」及其全部数据"})
}

// AdminEzfyBotClear POST /admin/ezfy-bot/clear —— 一键清空全部人机
//
// 走删号框架（ezfyPurgePlayerData）：二战库全清 + 家园账号物理删除。
func (h *EzfyAdmin) AdminEzfyBotClear(c *gin.Context) {
	var bots []model.EzfyBot
	h.DB.Select("user_id").Find(&bots)
	okCnt, failCnt := 0, 0
	for _, b := range bots {
		if err := h.ezfyBotPurgeOne(b.UserID); err != nil {
			failCnt++
			continue
		}
		okCnt++
	}
	// 计划表/日志表里可能残留孤儿行（人机已删但计划还在）
	h.DB.Exec("DELETE FROM ezfy_bot_target")
	h.DB.Exec("DELETE FROM ezfy_bot_log")
	resp.OK(c, gin.H{
		"msg":     "已清空 " + strconv.Itoa(okCnt) + " 个人机" + map[bool]string{true: "（" + strconv.Itoa(failCnt) + " 个失败）"}[failCnt > 0],
		"removed": okCnt, "failed": failCnt,
	})
}

// ezfyBotPurgeOne 删一个人机的全部数据：二战库（purge 框架）+ 家园 users 行。
func (h *EzfyAdmin) ezfyBotPurgeOne(uid uint) error {
	var cities []model.EzfyCity
	h.DB.Select("id").Where("user_id = ?", uid).Find(&cities)
	ids := make([]int64, 0, len(cities))
	for _, ct := range cities {
		ids = append(ids, int64(ct.ID))
	}
	if _, err := h.ezfyPurgePlayerData(uid, ids); err != nil {
		return err
	}
	// 家园账号（只删带 is_bot=1 的，防误删真人）
	h.home().Exec("DELETE FROM users WHERE id = ? AND is_bot = 1", uid)
	return nil
}

// AdminEzfyBotLogs GET /admin/ezfy-bot/logs?uid=&page=&size= —— 行为日志
//
// 排障利器：能看到「它上一次做了什么、为什么没动手」。
func (h *EzfyAdmin) AdminEzfyBotLogs(c *gin.Context) {
	page, offset, size := pageOf(c, 20)
	uid, _ := strconv.Atoi(c.Query("uid"))
	q := h.DB.Model(&model.EzfyBotLog{})
	if uid > 0 {
		q = q.Where("bot_uid = ?", uid)
	}
	var total int64
	q.Count(&total)
	var rows []model.EzfyBotLog
	q.Order("id DESC").Offset(offset).Limit(size).Find(&rows)

	// 批量补昵称
	uids := make([]uint, 0, len(rows))
	for _, r := range rows {
		uids = append(uids, r.BotUid)
	}
	nickOf := map[uint]string{}
	if len(uids) > 0 {
		var profs []model.EzfyProfile
		h.DB.Select("user_id, nickname").Where("user_id IN ?", uids).Find(&profs)
		for _, p := range profs {
			nickOf[p.UserID] = p.Nickname
		}
	}
	out := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		out = append(out, gin.H{
			"id": r.ID, "bot_uid": r.BotUid, "nickname": nickOf[r.BotUid],
			"action": r.Action, "detail": r.Detail, "result": r.Result,
			"created_at": r.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// AdminEzfyBotConfig GET /admin/ezfy-bot/config —— 风控配置
func (h *EzfyAdmin) AdminEzfyBotConfig(c *gin.Context) {
	resp.OK(c, gin.H{
		"attack_on":     ezfyBotAttackOn(),
		"atk_daily_cap": ezfyBotDailyAtkCap(),
		"max_count":     ezfyBotMaxCount,
		"tick_seconds":  int(ezfyBotTickInterval.Seconds()),
		"batch_size":    ezfyBotBatchSize,
	})
}

// AdminEzfyBotConfigSet POST /admin/ezfy-bot/config {attack_on, atk_daily_cap}
//
// ★ 改完必须 ezfyCfg.reload：开关读的是进程内配置缓存（不 reload 要等 30s 周期刷新）。
func (h *EzfyAdmin) AdminEzfyBotConfigSet(c *gin.Context) {
	var in struct {
		AttackOn    *int `json:"attack_on"`
		AtkDailyCap *int `json:"atk_daily_cap"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	up := map[string]interface{}{}
	if in.AttackOn != nil {
		v := 0
		if *in.AttackOn != 0 {
			v = 1
		}
		up["bot_atk_on"] = v
	}
	if in.AtkDailyCap != nil {
		v := *in.AtkDailyCap
		if v < 0 {
			v = 0
		}
		if v > 100 {
			v = 100
		}
		up["bot_atk_daily"] = v
	}
	if len(up) == 0 {
		resp.ParamError(c, "没有要修改的配置")
		return
	}
	if err := h.DB.Model(&model.EzfyCfgLimit{}).Where("id = 1").Updates(up).Error; err != nil {
		resp.ServerError(c, err)
		return
	}
	ezfyCfg.reload(h.DB, true)
	resp.OK(c, gin.H{"msg": "已保存"})
}
