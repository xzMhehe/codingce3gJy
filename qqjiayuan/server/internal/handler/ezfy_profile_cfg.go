package handler

import (
	"strconv"
	"strings"
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
// 1. 改昵称：**首次免费**，之后每次消耗 1 张「改名卡」(道具 17 / ItemType 13)
// 2. 改阵营：**首次免费**，之后每次消耗 1 个「阵营转换道具」(道具 18 / ItemType 14)
// 3. 军校免费刷新次数：全局默认值存在 settings 表，可按玩家在 ezfy_profile 上覆盖；
//    次数用完时用户端可在军校**直接使用招生简章**刷新（不用跳背包）。

const (
	ezfyItemRenameCard = 17 // 改名卡
	ezfyItemCampSwitch = 18 // 阵营转换道具

	// 军校每日免费刷新次数的 settings key（管理端可改）
	ezfySettingRecruitFreeLimit = "ezfy_recruit_free_limit"

	ezfyRecruitFreeLimitDefault = 5
)

// ezfyRecruitFreeLimit 取某玩家的军校每日免费刷新次数（玩家覆盖 > 全局默认）
func (h *EzfyHandler) ezfyRecruitFreeLimit(uid uint) int {
	var p model.EzfyProfile
	if err := h.DB.Where("user_id = ?", uid).First(&p).Error; err == nil && p.RecruitFreeLimit > 0 {
		return p.RecruitFreeLimit
	}
	return ezfyRecruitLimitOf(h.DB)
}

// ============ 统帅页自助接口 ============

// ProfileSelfInfo GET /games/ezfy/profile/self —— 统帅页自助信息
func (h *EzfyHandler) ProfileSelfInfo(c *gin.Context) {
	uid := middleware.GetUID(c)
	p := h.ensureProfile(uid)
	var u model.User
	h.DB.First(&u, uid)

	renameFree := p.RenameUsed == 0
	campFree := p.CampUsed == 0
	resp.OK(c, gin.H{
		"game_uid":  p.GameUID,
		"home_num":  u.Username,
		"nickname":  ezfyNickOf(p, &u),
		"camp":      p.Camp,
		"camp_name": ezfyCampName(p.Camp),

		"rename_free":       renameFree,
		"rename_card_count": h.itemCount(uid, ezfyItemRenameCard),
		"rename_card_id":    ezfyItemRenameCard,

		"camp_free":       campFree,
		"camp_item_count": h.itemCount(uid, ezfyItemCampSwitch),
		"camp_item_id":    ezfyItemCampSwitch,
	})
}

func ezfyNickOf(p model.EzfyProfile, u *model.User) string {
	if p.Nickname != "" {
		return p.Nickname
	}
	return u.Nickname
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
	p := h.ensureProfile(uid)
	var u model.User
	h.DB.First(&u, uid)
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
		if h.itemCount(uid, ezfyItemRenameCard) <= 0 {
			resp.ParamError(c, "首次免费改名已用掉，需要消耗 1 张「改名卡」（当前没有，可到商城购买）")
			return
		}
		h.consumeItem(uid, ezfyItemRenameCard)
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
		if h.itemCount(uid, ezfyItemCampSwitch) <= 0 {
			resp.ParamError(c, "首次免费转换阵营已用掉，需要消耗 1 个「阵营转换道具」（当前没有，可到商城购买）")
			return
		}
		h.consumeItem(uid, ezfyItemCampSwitch)
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

// ============ 军校免费刷新次数 ============

// ezfyRecruitLimitOf 读全局默认次数（settings 表，管理端可改）
//
// ★ `key` 是 MySQL 保留字，条件必须走结构体/Map 形式让 GORM 加反引号，
//   直接写 Where("key = ?") 会报语法错。
func ezfyRecruitLimitOf(db *gorm.DB) int {
	var st model.Setting
	if err := db.Where(&model.Setting{Key: ezfySettingRecruitFreeLimit}).First(&st).Error; err == nil {
		if n, e := strconv.Atoi(strings.TrimSpace(st.Value)); e == nil && n >= 0 {
			return n
		}
	}
	return ezfyRecruitFreeLimitDefault
}

// RecruitUseTicket POST /games/ezfy/acade/recruit/ticket
//
// ★ 用户规则：军校刷新次数用完后，**直接在军校使用招生简章**（不用先去背包用）。
func (h *EzfyHandler) RecruitUseTicket(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	city := h.getOrCreateCity(uid)
	if h.buildingLevel(city.ID, ezfyBuildingAcademy) < 1 {
		resp.ParamError(c, "需要先建造军校")
		return
	}
	if h.itemCount(uid, ezfyItemRecruitTicket) <= 0 {
		resp.ParamError(c, "没有「招生简章」（可到商城购买）")
		return
	}
	if msg := h.refreshRecruitFree(uid); msg != "" {
		resp.ParamError(c, msg)
		return
	}
	h.consumeItem(uid, ezfyItemRecruitTicket)
	resp.OK(c, gin.H{"msg": "已使用招生简章 ×1，军校候选名将已刷新（不占每日次数）"})
}

const ezfyItemRecruitTicket = 13 // 招生简章

// ============ 管理端：军校免费刷新次数维护 ============

// AdminEzfyRecruitLimitGet 读全局默认 + 玩家覆盖列表
func (h *AdminHandler) AdminEzfyRecruitLimitGet(c *gin.Context) {
	resp.OK(c, gin.H{
		"global": ezfyRecruitLimitOf(h.DB),
		"key":    ezfySettingRecruitFreeLimit,
		"usage":  "玩家覆盖为 0 表示跟随全局默认；玩家次数用完可在军校直接使用招生简章刷新",
	})
}

// AdminEzfyRecruitLimitSet 设置全局默认次数
func (h *AdminHandler) AdminEzfyRecruitLimitSet(c *gin.Context) {
	var in struct {
		Value int `json:"value"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.Value < 0 {
		resp.ParamError(c, "请填写不小于 0 的次数")
		return
	}
	h.DB.Clauses(clause.OnConflict{UpdateAll: true}).
		Create(&model.Setting{Key: ezfySettingRecruitFreeLimit, Value: strconv.Itoa(in.Value)})
	resp.OK(c, gin.H{"msg": "全局默认军校刷新次数已设为 " + strconv.Itoa(in.Value) + " 次/天"})
}

// AdminEzfyRecruitLimitList 玩家覆盖列表（含今日已用次数）
func (h *AdminHandler) AdminEzfyRecruitLimitList(c *gin.Context) {
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

	today := time.Now().Format("2006-01-02")
	global := ezfyRecruitLimitOf(h.DB)
	out := make([]gin.H, 0, len(rows))
	for _, p := range rows {
		var rec model.EzfyRecruit
		used, left := 0, global
		limit := global
		if p.RecruitFreeLimit > 0 {
			limit = p.RecruitFreeLimit
		}
		if err := h.DB.Where("user_id = ? AND recruit_date = ?", p.UserID, today).First(&rec).Error; err == nil {
			used = rec.RefreshCount
			left = limit - used
			if left < 0 {
				left = 0
			}
		}
		name, homeNum := h.ezfyAdminName(p.UserID)
		var ticket int64
		h.DB.Model(&model.EzfyItem{}).Where("user_id = ? AND cfg_id = ?", p.UserID, ezfyItemRecruitTicket).
			Select("COALESCE(SUM(`count`),0)").Scan(&ticket)
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
func (h *AdminHandler) AdminEzfyRecruitLimitSetUser(c *gin.Context) {
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
	resp.OK(c, gin.H{"msg": "已将该玩家的军校刷新次数设为 " + strconv.Itoa(in.Value) + " 次/天"})
}

// AdminEzfyRecruitLimitReset 重置某玩家今日已用次数
func (h *AdminHandler) AdminEzfyRecruitLimitReset(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("id"))
	today := time.Now().Format("2006-01-02")
	h.DB.Model(&model.EzfyRecruit{}).Where("user_id = ? AND recruit_date = ?", uid, today).
		Update("refresh_count", 0)
	resp.OK(c, gin.H{"msg": "已重置该玩家今日的军校刷新次数"})
}
