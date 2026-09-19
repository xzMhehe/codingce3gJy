package handler

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 二战风云 —— 游戏内好友（★ 与家园好友完全分开）
//
// 用户要求:「游戏里面加好友只是游戏里面的好友, 非家园分割开」。
// 所以这里**不碰**家园的 Friendship / FriendApply 表，另起 ezfy_friend / ezfy_friend_apply。
//
// 接口：
//   GET  /games/ezfy/friends             我的游戏好友
//   GET  /games/ezfy/friends/applies     收到的申请 + 我发出的申请
//   GET  /games/ezfy/friends/search      搜玩家（按游戏ID / 家园号码 / 昵称）
//   POST /games/ezfy/friends/apply       发申请
//   POST /games/ezfy/friends/handle      同意/拒绝
//   POST /games/ezfy/friends/delete      删好友

// ezfyIsFriend 是否已是游戏内好友
func (h *EzfyHandler) ezfyIsFriend(uid, other uint) bool {
	var n int64
	h.DB.Model(&model.EzfyFriend{}).Where("user_id = ? AND friend_id = ?", uid, other).Count(&n)
	return n > 0
}

// ezfyFriendBrief 好友/申请行里展示的玩家信息
func (h *EzfyHandler) ezfyFriendBrief(uid uint) gin.H {
	var u model.User
	h.DB.First(&u, uid)
	var p model.EzfyProfile
	h.DB.Where("user_id = ?", uid).First(&p)
	nick := p.Nickname
	if nick == "" {
		nick = u.Nickname
	}
	// 游戏ID：首次 = 家园ID；老数据兜底
	gu := p.GameUID
	if gu == 0 {
		gu = int64(uid)
	}
	return gin.H{
		"user_id": uid, "game_uid": gu,
		"home_num": u.Username, "nickname": nick,
		"camp": p.Camp, "camp_name": ezfyCampName(p.Camp),
		"prestige": p.Prestige, "rank_name": ezfyRankName(p.Prestige),
	}
}

// Friends GET /games/ezfy/friends —— 我的游戏好友
func (h *EzfyHandler) Friends(c *gin.Context) {
	uid := middleware.GetUID(c)
	var rows []model.EzfyFriend
	h.DB.Where("user_id = ?", uid).Order("id DESC").Find(&rows)
	out := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		b := h.ezfyFriendBrief(r.FriendId)
		b["remark"] = r.Remark
		b["created_at"] = r.CreatedAt
		out = append(out, b)
	}
	resp.OK(c, gin.H{"list": out, "total": len(out)})
}

// FriendsApplies GET /games/ezfy/friends/applies
func (h *EzfyHandler) FriendsApplies(c *gin.Context) {
	uid := middleware.GetUID(c)
	// 收到的（待处理）
	var inbox []model.EzfyFriendApply
	h.DB.Where("target_id = ? AND status = 0", uid).Order("id DESC").Find(&inbox)
	in := make([]gin.H, 0, len(inbox))
	for _, a := range inbox {
		b := h.ezfyFriendBrief(a.UserId)
		b["apply_id"] = a.ID
		b["remark"] = a.Remark
		b["created_at"] = a.CreatedAt
		in = append(in, b)
	}
	// 我发出的
	var outbox []model.EzfyFriendApply
	h.DB.Where("user_id = ?", uid).Order("id DESC").Limit(50).Find(&outbox)
	out := make([]gin.H, 0, len(outbox))
	for _, a := range outbox {
		b := h.ezfyFriendBrief(a.TargetId)
		b["apply_id"] = a.ID
		b["status"] = a.Status
		b["status_txt"] = map[int]string{0: "待处理", 1: "已同意", 2: "已拒绝"}[a.Status]
		out = append(out, b)
	}
	resp.OK(c, gin.H{"inbox": in, "outbox": out,
		"inbox_count": len(in)})
}

// FriendsSearch GET /games/ezfy/friends/search?keyword=
func (h *EzfyHandler) FriendsSearch(c *gin.Context) {
	uid := middleware.GetUID(c)
	kw := strings.TrimSpace(c.Query("keyword"))
	if kw == "" {
		resp.OK(c, gin.H{"list": []gin.H{}})
		return
	}
	var profs []model.EzfyProfile
	if n, err := strconv.ParseInt(kw, 10, 64); err == nil {
		h.DB.Where("game_uid = ?", n).Limit(20).Find(&profs)
		if len(profs) == 0 {
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
		if p.UserID == uid {
			continue // 不显示自己
		}
		b := h.ezfyFriendBrief(p.UserID)
		b["is_friend"] = h.ezfyIsFriend(uid, p.UserID)
		var ap int64
		h.DB.Model(&model.EzfyFriendApply{}).
			Where("user_id = ? AND target_id = ? AND status = 0", uid, p.UserID).Count(&ap)
		b["applied"] = ap > 0
		out = append(out, b)
	}
	resp.OK(c, gin.H{"list": out, "total": len(out)})
}

// FriendsApply POST /games/ezfy/friends/apply {target_id, remark}
func (h *EzfyHandler) FriendsApply(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		TargetID uint   `json:"target_id"`
		Remark   string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.TargetID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	if req.TargetID == uid {
		resp.ParamError(c, "不能加自己为好友")
		return
	}
	var u model.User
	if err := h.DB.First(&u, req.TargetID).Error; err != nil {
		resp.NotFound(c, "该玩家不存在")
		return
	}
	if h.ezfyIsFriend(uid, req.TargetID) {
		resp.ParamError(c, "你们已经是游戏好友了")
		return
	}
	// 对方已经申请过我 → 直接互相成为好友
	var reverse model.EzfyFriendApply
	if h.DB.Where("user_id = ? AND target_id = ? AND status = 0", req.TargetID, uid).
		First(&reverse).Error == nil {
		h.ezfyMakeFriends(uid, req.TargetID)
		h.DB.Model(&model.EzfyFriendApply{}).Where("id = ?", reverse.ID).Update("status", 1)
		resp.OK(c, gin.H{"msg": "对方也在等你，已互相成为游戏好友"})
		return
	}
	var dup int64
	h.DB.Model(&model.EzfyFriendApply{}).
		Where("user_id = ? AND target_id = ? AND status = 0", uid, req.TargetID).Count(&dup)
	if dup > 0 {
		resp.ParamError(c, "已发过申请，等对方处理")
		return
	}
	ap := model.EzfyFriendApply{UserId: uid, TargetId: req.TargetID,
		Remark: trimStr(strings.TrimSpace(req.Remark), 100), Status: 0, CreatedAt: time.Now()}
	h.DB.Create(&ap)
	resp.OK(c, gin.H{"msg": "好友申请已发送"})
}

// FriendsHandle POST /games/ezfy/friends/handle {apply_id, agree}
func (h *EzfyHandler) FriendsHandle(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		ApplyID int64 `json:"apply_id"`
		Agree   bool  `json:"agree"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.ApplyID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var ap model.EzfyFriendApply
	if err := h.DB.First(&ap, req.ApplyID).Error; err != nil {
		resp.NotFound(c, "申请不存在")
		return
	}
	if ap.TargetId != uid {
		resp.ParamError(c, "这条申请不是发给你的")
		return
	}
	if ap.Status != 0 {
		resp.ParamError(c, "该申请已处理过")
		return
	}
	if !req.Agree {
		h.DB.Model(&model.EzfyFriendApply{}).Where("id = ?", ap.ID).Update("status", 2)
		resp.OK(c, gin.H{"msg": "已拒绝该好友申请"})
		return
	}
	h.ezfyMakeFriends(ap.UserId, ap.TargetId)
	h.DB.Model(&model.EzfyFriendApply{}).Where("id = ?", ap.ID).Update("status", 1)
	resp.OK(c, gin.H{"msg": "已同意，你们现在是游戏好友了"})
}

// ezfyMakeFriends 双向写入好友关系（幂等）
func (h *EzfyHandler) ezfyMakeFriends(a, b uint) {
	pairs := [][2]uint{{a, b}, {b, a}}
	for _, pr := range pairs {
		var n int64
		h.DB.Model(&model.EzfyFriend{}).Where("user_id = ? AND friend_id = ?", pr[0], pr[1]).Count(&n)
		if n == 0 {
			h.DB.Create(&model.EzfyFriend{UserId: pr[0], FriendId: pr[1], CreatedAt: time.Now()})
		}
	}
}

// FriendsDelete POST /games/ezfy/friends/delete {friend_id}
func (h *EzfyHandler) FriendsDelete(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		FriendID uint `json:"friend_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.FriendID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	h.DB.Where("(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)",
		uid, req.FriendID, req.FriendID, uid).Delete(&model.EzfyFriend{})
	resp.OK(c, gin.H{"msg": "已解除游戏好友关系"})
}
