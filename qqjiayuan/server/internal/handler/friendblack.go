package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 黑名单（对齐诺哈 black_list.asp / black_add_ok.asp / black_del_ok.asp）
type BlackHandler struct{ DB *gorm.DB }

// 我的黑名单
func (h *BlackHandler) List(c *gin.Context) {
	uid := middleware.GetUID(c)
	var rows []model.FriendBlack
	h.DB.Where("user_id = ? AND end_time > ?", uid, time.Now()).Order("id DESC").Find(&rows)
	out := []gin.H{}
	for _, b := range rows {
		var u model.User
		if err := h.DB.First(&u, b.FriendID).Error; err != nil {
			continue
		}
		name := b.Name
		if name == "" {
			name = u.Nickname
		}
		out = append(out, gin.H{"id": b.ID, "friend_id": u.ID, "nickname": name, "color": u.Color,
			"level": u.Level, "add_time": b.AddTime, "end_time": b.EndTime})
	}
	resp.OK(c, out)
}

// 拉黑（对齐诺哈 black_add_ok.asp）：移除双方好友关系，最多100位，有效期1年
func (h *BlackHandler) Add(c *gin.Context) {
	uid := middleware.GetUID(c)
	id, _ := strconv.Atoi(c.Param("id"))
	var target model.User
	if err := h.DB.First(&target, id).Error; err != nil || target.Status == 0 {
		resp.NotFound(c, "无此会员")
		return
	}
	if id == int(uid) {
		resp.ParamError(c, "不能拉黑自己")
		return
	}
	// 清理过期
	h.DB.Where("user_id = ? AND end_time <= ?", uid, time.Now()).Delete(&model.FriendBlack{})
	var count int64
	h.DB.Model(&model.FriendBlack{}).Where("user_id = ?", uid).Count(&count)
	if count >= 100 {
		resp.ParamError(c, "最多只能添加100位黑名单")
		return
	}
	var exist model.FriendBlack
	if err := h.DB.Where("user_id = ? AND friend_id = ?", uid, id).First(&exist).Error; err == nil {
		resp.ParamError(c, "请不要重复添加")
		return
	}
	now := time.Now()
	h.DB.Create(&model.FriendBlack{
		UserID: uid, FriendID: uint(id), Name: target.Nickname,
		AddTime: now, EndTime: now.AddDate(1, 0, 0),
	})
	// 移除双方好友关系（对齐诺哈：双向删除）
	h.DB.Where("user_id = ? AND friend_id = ?", uid, id).Delete(&model.Friendship{})
	h.DB.Where("user_id = ? AND friend_id = ?", id, uid).Delete(&model.Friendship{})
	// 清理未决申请
	h.DB.Where("(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)", uid, id, id, uid).Delete(&model.FriendApply{})
	resp.OK(c, "已加入黑名单")
}

// 解除黑名单（对齐诺哈 black_del_ok.asp）
func (h *BlackHandler) Remove(c *gin.Context) {
	uid := middleware.GetUID(c)
	id, _ := strconv.Atoi(c.Param("id"))
	res := h.DB.Where("id = ? AND user_id = ?", id, uid).Delete(&model.FriendBlack{})
	if res.RowsAffected == 0 {
		resp.NotFound(c, "无此黑名单")
		return
	}
	resp.OK(c, "已解除拉黑")
}