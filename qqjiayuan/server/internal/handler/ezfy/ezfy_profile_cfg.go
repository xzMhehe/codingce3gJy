package ezfy

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 二战风云 —— 统帅页自助改昵称/改阵营 + 军校免费刷新次数配置
//
// 1. 改昵称：**首次免费**，之后每次消耗 1 张「改名卡」(ItemType 13)
// 2. 改阵营：**首次免费**，之后每次消耗 1 个「阵营转换道具」(ItemType 14)
// 3. 军校免费刷新次数：全局默认值存在 settings 表，可按玩家在 ezfy_profile 上覆盖；
//    次数用完时用户端可在军校**直接使用招生简章**刷新（不用跳背包）。
//
// ★★ 道具 cfg_id 一律按名字解析（道具已迁到 1001+，别硬编码数字），见下方 ezfyItemXxx()。

const (
	// 军校每小时免费刷新次数的 settings key（管理端可改；2026-09-28 从每日改每小时）
	ezfySettingRecruitFreeLimit = "ezfy_recruit_free_limit"

	ezfyRecruitFreeLimitDefault = 5
)

// ★★ 2026-10-10 统帅页自助道具一律**按名字解析** cfg_id（道具已统一迁到 1001+，见
// seed/ezfy_cfg_gen.go）。硬编码数字 ID 会在调号段后静默失效（查不到 → 持有量恒 0 →
// 「背包里明明有却提示没有」）。与集结令/信号弹同款，见 ezfy_geo.go 的 itemIDByName。
const (
	ezfyItemNameRenameCard    = "改名卡"    // ItemType 13
	ezfyItemNameCampSwitch    = "阵营转换道具" // ItemType 14
	ezfyItemNameRecruitTicket = "招生简章"   // ItemType 9
)

// ezfyItemRenameCard 改名卡的道具 cfg_id（0 = 配置里没有）
func ezfyItemRenameCard() int { return ezfyCfg.itemIDByName(ezfyItemNameRenameCard) }

// ezfyItemCampSwitch 阵营转换道具的道具 cfg_id（0 = 配置里没有）
func ezfyItemCampSwitch() int { return ezfyCfg.itemIDByName(ezfyItemNameCampSwitch) }

// ezfyItemRecruitTicket 招生简章的道具 cfg_id（0 = 配置里没有）
func ezfyItemRecruitTicket() int { return ezfyCfg.itemIDByName(ezfyItemNameRecruitTicket) }

// ezfyRecruitFreeLimit 取某玩家的军校每小时免费刷新次数（玩家覆盖 > 全局默认）
func (h *EzfyHandler) ezfyRecruitFreeLimit(uid uint) int {
	var p model.EzfyProfile
	if err := h.DB.Where("user_id = ?", uid).First(&p).Error; err == nil && p.RecruitFreeLimit > 0 {
		return p.RecruitFreeLimit
	}
	return ezfyRecruitLimitOf(h.DB)
}

// ezfyRecruitFreeLimitWith 同上，但用**本请求已经取到的档案**（省一次跨 WAN 的 profile 查询）。
// ★ 2026-10-05 性能：/acade/recruit 原来在懒结算里刚查过 profile，这里又查一次。
func (h *EzfyHandler) ezfyRecruitFreeLimitWith(p *model.EzfyProfile) int {
	if p != nil && p.RecruitFreeLimit > 0 {
		return p.RecruitFreeLimit
	}
	return ezfyRecruitLimitOf(h.DB)
}

// ============ 统帅页自助接口 ============

// ProfileSelfInfo GET /games/ezfy/profile/self —— 统帅页自助信息
func (h *EzfyHandler) ProfileSelfInfo(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs() // ★ 下面按名字解析改名卡/阵营转换道具的 cfg_id，必须先加载配置缓存
	p := h.ensureProfile(uid)
	var u model.User
	h.home().First(&u, uid)

	renameFree := p.RenameUsed == 0
	campFree := p.CampUsed == 0
	resp.OK(c, gin.H{
		"game_uid":  p.GameUID,
		"home_num":  u.Username,
		"nickname":  ezfyNickOf(p, &u),
		"camp":      p.Camp,
		"camp_name": ezfyCampName(p.Camp),

		"rename_free":       renameFree,
		"rename_card_count": h.itemCount(uid, ezfyItemRenameCard()),
		"rename_card_id":    ezfyItemRenameCard(),

		"camp_free":       campFree,
		"camp_item_count": h.itemCount(uid, ezfyItemCampSwitch()),
		"camp_item_id":    ezfyItemCampSwitch(),

		// ★ 2026-10-10 首页布局（1=新布局 2=老布局；0=未设置按新布局展示），统帅页单选切换
		"home_layout": p.HomeLayout,

		// ★ 2026-10-12 访问记录：别人看我统帅页的总量（信息页「占领野地」下方可点开访问列表）
		"visit_count": h.visitCount(uid),
	})
}

// visitCount 该玩家统帅页被访问的历史总量（含记录在案的所有访问）。
func (h *EzfyHandler) visitCount(uid uint) int64 {
	var n int64
	h.DB.Model(&model.EzfyVisitLog{}).Where("target_uid = ?", uid).Count(&n)
	return n
}

func ezfyNickOf(p model.EzfyProfile, u *model.User) string {
	if p.Nickname != "" {
		return p.Nickname
	}
	return u.Nickname
}

// ezfyHasEmoji 名字里是否含 emoji（表情符号）。
//
// ★ 2026-10-11 用户要求「[修改昵称] 不能用表情」：emoji 在复古 WAP 文字页上渲染不一致
// （同一个 emoji 在 Windows/安卓/iOS 上宽度能差一倍）还容易撑破行宽，所以昵称一律不许带。
// 覆盖范围 = Unicode 里「表情符号」所在的主要区块（含 emoji 序列的组成部分）：
//
//	1F000–1FAFF  表情与象形（😀🎉🔥🚀🀄🃏…，含扩展 A；国旗 1F1E6–1F1FF 也在内）
//	2600–27BF    杂项符号 + 装饰符号（☀❤✅⚡✂✈ 等）——**但放行 ★(2605) ☆(2606)**：
//	             它们是常用装饰符号、不算表情，游戏里也常见
//	2B00–2BFF    杂项符号与箭头（⭐⬛⬆ 等）
//	FE00–FE0F    变体选择符（emoji 呈现选择符「️」就落在这）
//	200D         零宽连接符（👨‍👩‍👦 这类组合 emoji 的粘合符）
//	20E3         组合键帽（1️⃣ 的 ️⃣）
//	00A9 00AE 203C 2049 2122 2139 3030 303D 3297 3299（©®‼⁉™ℹ〰〽㊗㊙ —— 会被当 emoji 渲染）
//
// ⚠️ 故意**不**包含 2190–21FF（← → ↑ ↓ 等箭头）：那是常用符号，不算表情。
func ezfyHasEmoji(s string) bool {
	for _, r := range s {
		switch {
		case r >= 0x1F000 && r <= 0x1FAFF,
			r >= 0x2600 && r <= 0x27BF && r != 0x2605 && r != 0x2606,
			r >= 0x2B00 && r <= 0x2BFF,
			r >= 0xFE00 && r <= 0xFE0F,
			r == 0x200D, r == 0x20E3,
			r == 0x00A9, r == 0x00AE, r == 0x203C, r == 0x2049,
			r == 0x2122, r == 0x2139, r == 0x3030, r == 0x303D,
			r == 0x3297, r == 0x3299:
			return true
		}
	}
	return false
}

// ProfileRename POST /games/ezfy/profile/rename {nickname}
//
// 首次免费；之后消耗 1 张改名卡。
func (h *EzfyHandler) ProfileRename(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Nickname string `json:"nickname"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	name := strings.TrimSpace(req.Nickname)
	if name == "" {
		resp.ParamError(c, "请填写新的统帅昵称")
		return
	}
	if len([]rune(name)) < 2 || len([]rune(name)) > 12 {
		resp.ParamError(c, "昵称长度需在 2~12 个字符之间")
		return
	}
	// ★ 2026-10-11 用户要求「[修改昵称] 不能用表情」：昵称不许带 emoji（前端也有同样一版即时提示）
	if ezfyHasEmoji(name) {
		resp.ParamError(c, "昵称不能包含表情符号")
		return
	}
	h.cfgs() // 按名字解析「改名卡」cfg_id 前必须先加载配置缓存
	p := h.ensureProfile(uid)
	var u model.User
	h.home().First(&u, uid)
	if name == ezfyNickOf(p, &u) {
		resp.ParamError(c, "新昵称与当前昵称相同")
		return
	}

	usedCard := false
	if p.RenameUsed == 0 {
		// 首次免费
		h.DB.Model(&model.EzfyProfile{}).Where("id = ?", p.ID).
			Updates(map[string]interface{}{"nickname": name, "rename_used": 1})
	} else {
		if h.itemCount(uid, ezfyItemRenameCard()) <= 0 {
			resp.ParamError(c, "首次免费改名已用掉，需要消耗 1 张「改名卡」（当前没有，可到商城购买）")
			return
		}
		h.consumeItem(uid, ezfyItemRenameCard(), "玩家改名")
		usedCard = true
		h.DB.Model(&model.EzfyProfile{}).Where("id = ?", p.ID).Update("nickname", name)
	}
	msg := "改名成功：「" + name + "」"
	if usedCard {
		msg += "（消耗改名卡 ×1）"
	} else {
		msg += "（首次免费）"
	}
	resp.OK(c, gin.H{"msg": msg, "nickname": name})
}

// ProfileChangeCamp POST /games/ezfy/profile/camp {camp}
//
// 首次免费；之后消耗 1 个阵营转换道具。
func (h *EzfyHandler) ProfileChangeCamp(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Camp int `json:"camp"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if req.Camp != 1 && req.Camp != 2 {
		resp.ParamError(c, "阵营只能是 1(同盟国) 或 2(轴心国)")
		return
	}
	h.cfgs() // 按名字解析「阵营转换道具」cfg_id 前必须先加载配置缓存
	p := h.ensureProfile(uid)
	if p.Camp == req.Camp {
		resp.ParamError(c, "当前已经是「"+ezfyCampName(req.Camp)+"」")
		return
	}

	usedItem := false
	if p.CampUsed == 0 {
		h.DB.Model(&model.EzfyProfile{}).Where("id = ?", p.ID).
			Updates(map[string]interface{}{"camp": req.Camp, "camp_used": 1})
	} else {
		if h.itemCount(uid, ezfyItemCampSwitch()) <= 0 {
			resp.ParamError(c, "首次免费转换阵营已用掉，需要消耗 1 个「阵营转换道具」（当前没有，可到商城购买）")
			return
		}
		h.consumeItem(uid, ezfyItemCampSwitch(), "阵营转换")
		usedItem = true
		h.DB.Model(&model.EzfyProfile{}).Where("id = ?", p.ID).Update("camp", req.Camp)
	}
	msg := "阵营已转换为「" + ezfyCampName(req.Camp) + "」"
	if usedItem {
		msg += "（消耗阵营转换道具 ×1）"
	} else {
		msg += "（首次免费）"
	}
	resp.OK(c, gin.H{"msg": msg, "camp": req.Camp, "camp_name": ezfyCampName(req.Camp)})
}

// ProfileSetLayout POST /games/ezfy/profile/layout {layout} —— 首页布局切换（1=新布局 2=老布局）
//
// ★ 2026-10-10 统帅页「首页布局」单选：即改即生效（老布局/新布局），无额外确认弹窗。
func (h *EzfyHandler) ProfileSetLayout(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Layout int `json:"layout"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if req.Layout != 1 && req.Layout != 2 {
		resp.ParamError(c, "布局只能是 1(新布局) 或 2(老布局)")
		return
	}
	p := h.ensureProfile(uid)
	h.DB.Model(&model.EzfyProfile{}).Where("id = ?", p.ID).Update("home_layout", req.Layout)
	name := "新布局"
	if req.Layout == 2 {
		name = "老布局"
	}
	resp.OK(c, gin.H{"msg": "首页已切换为" + name, "layout": req.Layout})
}

// ============ 军校免费刷新次数 ============

// ezfyRecruitLimitOf 读全局默认次数（settings 表，管理端可改）
//
// ★ `key` 是 MySQL 保留字，条件必须走结构体/Map 形式让 GORM 加反引号，
//
//	直接写 Where("key = ?") 会报语法错。
// ★ 2026-10-05 性能：全局默认值进程内缓存 30 秒。
//
//	这条 `SELECT * FROM settings WHERE key='ezfy_recruit_free_limit'` 是**每个**军校接口的
//	必经一步（跨 WAN ~120ms），而它是管理端低频改动的一个全局开关 —— 30 秒收敛完全够。
var (
	ezfyRecruitLimitMu  sync.Mutex
	ezfyRecruitLimitVal int
	ezfyRecruitLimitAt  int64
)

const ezfyRecruitLimitTTLMs = 30000

func ezfyRecruitLimitOf(db *gorm.DB) int {
	now := time.Now().UnixMilli()
	ezfyRecruitLimitMu.Lock()
	if ezfyRecruitLimitAt > 0 && now-ezfyRecruitLimitAt < ezfyRecruitLimitTTLMs {
		v := ezfyRecruitLimitVal
		ezfyRecruitLimitMu.Unlock()
		return v
	}
	ezfyRecruitLimitMu.Unlock()

	v := ezfyRecruitFreeLimitDefault
	var st model.Setting
	if err := db.Where(&model.Setting{Key: ezfySettingRecruitFreeLimit}).First(&st).Error; err == nil {
		if n, e := strconv.Atoi(strings.TrimSpace(st.Value)); e == nil && n >= 0 {
			v = n
		}
	}
	ezfyRecruitLimitMu.Lock()
	ezfyRecruitLimitVal, ezfyRecruitLimitAt = v, now
	ezfyRecruitLimitMu.Unlock()
	return v
}

// RecruitUseTicket POST /games/ezfy/acade/recruit/ticket
//
// ★ 用户规则：军校刷新次数用完后，**直接在军校使用招生简章**（不用先去背包用）。
func (h *EzfyHandler) RecruitUseTicket(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	ezfyPageCacheDel(uid) // 候选已重刷, 失效玩家页缓存（与 AcadeRefresh 同口径）
	city := h.getOrCreateCity(uid)
	if h.buildingLevel(city.ID, ezfyBuildingAcademy) < 1 {
		resp.ParamError(c, "需要先建造军校")
		return
	}
	if h.itemCount(uid, ezfyItemRecruitTicket()) <= 0 {
		resp.ParamError(c, "没有「招生简章」（可到商城购买）")
		return
	}
	if msg := h.refreshRecruitFree(uid); msg != "" {
		resp.ParamError(c, msg)
		return
	}
	h.consumeItem(uid, ezfyItemRecruitTicket(), "军校招募")
	resp.OK(c, gin.H{"msg": "已使用招生简章 ×1，军校候选名将已刷新（不占每小时次数）"})
}

// ============ 管理端：军校免费刷新次数维护 ============

// AdminEzfyRecruitLimitGet 读全局默认 + 玩家覆盖列表
func (h *EzfyAdmin) AdminEzfyRecruitLimitGet(c *gin.Context) {
	// 周期模式（1=按天 2=按小时）来自「二战系统配置」，前端据此显示「每日 / 每小时」
	cycle := 2
	if ezfyCfg.ready() {
		cycle = ezfyCfg.limit.RecruitCycleMode
	}
	resp.OK(c, gin.H{
		"global":     ezfyRecruitLimitOf(h.DB),
		"key":        ezfySettingRecruitFreeLimit,
		"cycle_mode": cycle,
		"usage":      "玩家覆盖为 0 表示跟随全局默认；次数按周期重置；玩家次数用完可在军校直接使用招生简章刷新",
	})
}

// AdminEzfyRecruitLimitSet 设置全局默认次数
func (h *EzfyAdmin) AdminEzfyRecruitLimitSet(c *gin.Context) {
	var in struct {
		Value int `json:"value"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.Value < 0 {
		resp.ParamError(c, "请填写不小于 0 的次数")
		return
	}
	h.DB.Clauses(clause.OnConflict{UpdateAll: true}).
		Create(&model.Setting{Key: ezfySettingRecruitFreeLimit, Value: strconv.Itoa(in.Value)})
	resp.OK(c, gin.H{"msg": "全局默认军校刷新次数已设为 " + strconv.Itoa(in.Value) + " 次/小时"})
}

// AdminEzfyRecruitLimitList 玩家覆盖列表（含本小时已用次数）
func (h *EzfyAdmin) AdminEzfyRecruitLimitList(c *gin.Context) {
	h.cfgs() // 按名字解析「招生简章」cfg_id 前先加载配置缓存
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
	q.Order("id").Offset(offset).Limit(size).Find(&rows)

	cycle := recruitCycleKey()
	global := ezfyRecruitLimitOf(h.DB)
	out := make([]gin.H, 0, len(rows))
	for _, p := range rows {
		var rec model.EzfyRecruit
		used, left := 0, global
		limit := global
		if p.RecruitFreeLimit > 0 {
			limit = p.RecruitFreeLimit
		}
		if err := h.DB.Where("user_id = ? AND recruit_date = ?", p.UserID, cycle).First(&rec).Error; err == nil {
			used = rec.RefreshCount
			left = limit - used
			if left < 0 {
				left = 0
			}
		}
		name, homeNum := h.ezfyAdminName(p.UserID)
		var ticket int64
		h.DB.Model(&model.EzfyItem{}).Where("user_id = ? AND cfg_id = ?", p.UserID, ezfyItemRecruitTicket()).
			Select("COALESCE(SUM(`count`),0)").Scan(&ticket)
		// 字段名沿用 used_today/left_today 以兼容管理端前端，语义已是「本小时」
		out = append(out, gin.H{
			"user_id": p.UserID, "game_uid": p.GameUID, "nickname": p.Nickname,
			"player_name": name, "home_num": homeNum,
			"override": p.RecruitFreeLimit, "limit": limit, "used_today": used, "left_today": left,
			"ticket_count": ticket,
		})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size, "global": global})
}

// AdminEzfyRecruitLimitSetUser 设置某玩家的覆盖次数（0 = 跟随全局）
func (h *EzfyAdmin) AdminEzfyRecruitLimitSetUser(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("id"))
	var in struct {
		Value int `json:"value"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.Value < 0 {
		resp.ParamError(c, "请填写不小于 0 的次数")
		return
	}
	var p model.EzfyProfile
	if err := h.DB.Where("user_id = ?", uid).First(&p).Error; err != nil {
		resp.NotFound(c, "玩家不存在")
		return
	}
	h.DB.Model(&model.EzfyProfile{}).Where("id = ?", p.ID).Update("recruit_free_limit", in.Value)
	if in.Value == 0 {
		resp.OK(c, gin.H{"msg": "已改为跟随全局默认次数"})
		return
	}
	resp.OK(c, gin.H{"msg": "已将该玩家的军校刷新次数设为 " + strconv.Itoa(in.Value) + " 次/小时"})
}

// AdminEzfyRecruitLimitReset 重置某玩家本小时已用次数
func (h *EzfyAdmin) AdminEzfyRecruitLimitReset(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("id"))
	cycle := recruitCycleKey()
	h.DB.Model(&model.EzfyRecruit{}).Where("user_id = ? AND recruit_date = ?", uid, cycle).
		Update("refresh_count", 0)
	resp.OK(c, gin.H{"msg": "已重置该玩家本小时的军校刷新次数"})
}
