package handler

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 二战风云 玩家信息页（复刻 PlayerController.infoOther + templates/user/info.html）
//
// 原版行为：`GET /ezfy/infoOther?userId=N` 渲染 `info.html`，显示
// 昵称 / 声望 / 军衔(军衔职) ，并且**不是自己且还不是好友时**给一个「加好友」按钮。
// 本项目补上了城市数/军官数/总兵力/军团等本项目已有的信息。
//
// ★ 游戏是沉浸式的：游戏内点玩家名只能看这一页（二战风云的数据），
//   不允许跳到家园站点的个人主页 /user/:id。

// PlayerInfo GET /games/ezfy/player/:id —— 查看某位统帅的信息
func (h *EzfyHandler) PlayerInfo(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	tid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || tid == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	// ★ id 既接受家园 user_id，也接受「游戏ID」（首次=家园ID，之后可能独立变化）
	target := uint(tid)
	var u model.User
	if err := h.DB.First(&u, target).Error; err != nil {
		var gp model.EzfyProfile
		if e2 := h.DB.Where("game_uid = ?", int64(tid)).First(&gp).Error; e2 == nil {
			target = gp.UserID
			if e3 := h.DB.First(&u, target).Error; e3 != nil {
				resp.NotFound(c, "这位统帅不存在")
				return
			}
		} else {
			resp.NotFound(c, "这位统帅不存在")
			return
		}
	}

	// 档案：只读，不存在就按默认值展示（不给别人凭空建档案）
	var p model.EzfyProfile
	h.DB.Where("user_id = ?", target).First(&p)
	nickname := p.Nickname
	if nickname == "" {
		nickname = u.Nickname
	}

	// 城市 / 军官 / 兵力 / 野地
	var cities []model.EzfyCity
	h.DB.Where("user_id = ?", target).Find(&cities)
	cityIds := make([]uint, 0, len(cities))
	for _, ct := range cities {
		cityIds = append(cityIds, ct.ID)
	}
	var officerCount, wildCount int64
	var troopTotal int64
	if len(cityIds) > 0 {
		h.DB.Model(&model.EzfyOfficer{}).Where("city_id IN ?", cityIds).Count(&officerCount)
		h.DB.Model(&model.EzfyWildland{}).Where("city_id IN ?", cityIds).Count(&wildCount)
		h.DB.Model(&model.EzfyCityTroop{}).Where("city_id IN ?", cityIds).
			Select("COALESCE(SUM(count),0)").Scan(&troopTotal)
	}

	// 军团
	corpsName := ""
	if cp := h.myCorpsOf(target); cp != nil {
		corpsName = cp.Name
	}

	// 好友关系（复刻 infoOther 的 isShowAdd）
	isSelf := target == uid
	isFriend, isApplied := false, false
	if !isSelf {
		var n int64
		h.DB.Model(&model.Friendship{}).Where("user_id = ? AND friend_id = ? AND status = 1", uid, target).Count(&n)
		isFriend = n > 0
		h.DB.Model(&model.FriendApply{}).Where("user_id = ? AND friend_id = ?", uid, target).Count(&n)
		isApplied = n > 0
	}

	resp.OK(c, gin.H{
		"user_id": target, "account": u.Username,
		"game_uid": p.GameUID,
		"home_num": u.Username,
		"name":     u.Nickname, "nickname": nickname, "color": u.Color,
		"camp": p.Camp, "camp_name": ezfyCampName(p.Camp),
		"prestige": p.Prestige, "rank_name": ezfyRankName(p.Prestige), "rank_post": ezfyRankPost(p.Prestige),
		"corps_name":    corpsName,
		"city_count":    len(cities),
		"officer_count": officerCount,
		"troop_total":   troopTotal,
		"wild_count":    wildCount,
		"is_self":       isSelf,
		"is_friend":     isFriend,
		"is_applied":    isApplied,
	})
}

// PlayerSearch 搜索玩家（★ 优先按「游戏ID」，也支持家园号码与昵称）
//
// GET /games/ezfy/player-search?keyword=xxx
func (h *EzfyHandler) PlayerSearch(c *gin.Context) {
	kw := strings.TrimSpace(c.Query("keyword"))
	if kw == "" {
		resp.OK(c, gin.H{"list": []gin.H{}})
		return
	}
	var profs []model.EzfyProfile
	// 纯数字：游戏ID / 家园号码 精确命中；否则按游戏内昵称模糊
	if n, err := strconv.ParseInt(kw, 10, 64); err == nil {
		h.DB.Where("game_uid = ?", n).Limit(20).Find(&profs)
		if len(profs) == 0 {
			// 家园号码（users.username）兜底
			var uids []uint
			h.DB.Model(&model.User{}).Select("id").Where("username = ?", kw).Scan(&uids)
			if len(uids) > 0 {
				h.DB.Where("user_id IN ?", uids).Limit(20).Find(&profs)
			}
		}
	} else {
		h.DB.Where("nickname LIKE ?", "%"+kw+"%").Limit(20).Find(&profs)
	}

	out := make([]gin.H, 0, len(profs))
	for _, p := range profs {
		var u model.User
		h.DB.Select("username, nickname").First(&u, p.UserID)
		nick := p.Nickname
		if nick == "" {
			nick = u.Nickname
		}
		out = append(out, gin.H{
			"user_id": p.UserID, "game_uid": p.GameUID,
			"home_num": u.Username, "nickname": nick,
			"camp": p.Camp, "camp_name": ezfyCampName(p.Camp),
			"prestige": p.Prestige, "rank_name": ezfyRankName(p.Prestige),
		})
	}
	resp.OK(c, gin.H{"list": out, "total": len(out)})
}
