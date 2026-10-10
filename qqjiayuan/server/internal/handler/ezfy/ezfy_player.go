package ezfy

import (
	"strconv"
	"strings"
	"sync"

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
	if err := h.home().First(&u, target).Error; err != nil {
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
	// ★ 2026-10-05 性能（用户反馈「/player/:id 2s+」）：原来 ~10 条查询全串行。
	//   改为两波并行：第一波 档案+城市（1 RTT）→ 第二波 军官/野地/兵力/军团/好友（1 RTT）。
	var p model.EzfyProfile
	var cities []model.EzfyCity
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); h.DB.Where("user_id = ?", target).First(&p) }()
	go func() { defer wg.Done(); h.DB.Where("user_id = ?", target).Find(&cities) }()
	wg.Wait()
	nickname := p.Nickname
	if nickname == "" {
		nickname = u.Nickname
	}
	cityIds := make([]uint, 0, len(cities))
	for _, ct := range cities {
		cityIds = append(cityIds, ct.ID)
	}
	// ★ 2026-10-05 第二波并行（1 RTT）：军官/野地数、单城最高兵力、军团、好友关系
	var officerCount, wildCount, maxCityTroop int64
	corpsName, corpsTitle := "", ""
	isSelf := target == uid // 好友关系判定前置（复刻 infoOther 的 isShowAdd）
	isFriend, isApplied := false, false
	var wg2 sync.WaitGroup
	wg2.Add(4)
	go func() { // 军官数 / 野地数（跨城）
		defer wg2.Done()
		if len(cityIds) > 0 {
			h.DB.Model(&model.EzfyOfficer{}).Where("city_id IN ?", cityIds).Count(&officerCount)
			h.DB.Model(&model.EzfyWildland{}).Where("city_id IN ?", cityIds).Count(&wildCount)
		}
	}()
	go func() { // 单城最高兵力（★ 2026-09-23 不跨城累加，防 int64 溢出）
		defer wg2.Done()
		if len(cityIds) > 0 {
			h.DB.Raw(`SELECT COALESCE(MAX(tot),0) FROM (
					SELECT city_id, SUM(count) tot FROM ezfy_city_troop
					WHERE city_id IN ? GROUP BY city_id) t`, cityIds).Scan(&maxCityTroop)
		}
	}()
	go func() { // 军团名 / 军团职务（★ 2026-09-29 他人统帅页也展示军团职务）
		defer wg2.Done()
		if cp := h.myCorpsOf(target); cp != nil {
			corpsName = cp.Name
			var mb model.EzfyCorpsMember
			if err := h.DB.Where("user_id = ?", target).First(&mb).Error; err == nil {
				corpsTitle = mb.Title
			}
		}
	}()
	go func() { // 好友关系（游戏内好友表，与家园好友分开）
		defer wg2.Done()
		if isSelf {
			return
		}
		var n int64
		h.DB.Model(&model.EzfyFriend{}).Where("user_id = ? AND friend_id = ?", uid, target).Count(&n)
		isFriend = n > 0
		h.DB.Model(&model.EzfyFriendApply{}).
			Where("user_id = ? AND target_id = ? AND status = 0", uid, target).Count(&n)
		isApplied = n > 0
	}()
	wg2.Wait()

	resp.OK(c, gin.H{
		"user_id": target, "account": u.Username,
		"game_uid": p.GameUID,
		"home_num": u.Username,
		"name":     u.Nickname, "nickname": nickname, "color": u.Color,
		"camp": p.Camp, "camp_name": ezfyCampName(p.Camp),
		"prestige": p.Prestige, "rank_name": ezfyRankNameAt(ezfyProfileRank(&p)), "rank_post": ezfyRankPostAt(ezfyProfileRank(&p)),
		"corps_name":    corpsName,
		"corps_title":   corpsTitle,
		"city_count":    len(cities),
		"officer_count": officerCount,
		"troop_max":     maxCityTroop,
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
			h.home().Model(&model.User{}).Select("id").Where("username = ?", kw).Scan(&uids)
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
		h.home().Select("username, nickname").First(&u, p.UserID)
		nick := p.Nickname
		if nick == "" {
			nick = u.Nickname
		}
		out = append(out, gin.H{
			"user_id": p.UserID, "game_uid": p.GameUID,
			"home_num": u.Username, "nickname": nick,
			"camp": p.Camp, "camp_name": ezfyCampName(p.Camp),
			"prestige": p.Prestige, "rank_name": ezfyRankNameAt(ezfyProfileRank(&p)),
		})
	}
	resp.OK(c, gin.H{"list": out, "total": len(out)})
}
