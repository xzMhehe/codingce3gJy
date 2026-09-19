package handler

import (
	"strconv"

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
	var u model.User
	if err := h.DB.First(&u, tid).Error; err != nil {
		resp.NotFound(c, "这位统帅不存在")
		return
	}
	target := uint(tid)

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
		"user_id": target, "account": strconv.FormatUint(tid, 10),
		"name": u.Nickname, "nickname": nickname, "color": u.Color,
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
