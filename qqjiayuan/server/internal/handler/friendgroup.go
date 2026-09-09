package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

type FriendGroupHandler struct{ DB *gorm.DB }

// 我的分组（含各分组好友）
func (h *FriendGroupHandler) List(c *gin.Context) {
	uid := middleware.GetUID(c)
	var groups []model.FriendGroup
	h.DB.Where("user_id = ?", uid).Order("sort ASC, id ASC").Find(&groups)
	out := []gin.H{}
	for _, g := range groups {
		var items []model.FriendGroupItem
		h.DB.Where("group_id = ?", g.ID).Find(&items)
		ids := []uint{}
		for _, it := range items {
			ids = append(ids, it.FriendID)
		}
		members := []gin.H{}
		if len(ids) > 0 {
			var users []model.User
			h.DB.Where("id IN ?", ids).Find(&users)
			for _, u := range users {
				members = append(members, gin.H{"id": u.ID, "nickname": u.Nickname, "color": u.Color, "level": u.Level})
			}
		}
		out = append(out, gin.H{"id": g.ID, "name": g.Name, "sort": g.Sort, "count": g.Amount, "members": members})
	}
	resp.OK(c, out)
}

type groupReq struct {
	Name string `json:"name" binding:"required,min=1,max=12"`
	Sort int    `json:"sort"`
}

func (h *FriendGroupHandler) Create(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req groupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "分组名 1-12 个字")
		return
	}
	if req.Sort < 0 || req.Sort > 99 {
		req.Sort = 0
	}
	// 对齐诺哈：最多 10 个分组
	var count int64
	h.DB.Model(&model.FriendGroup{}).Where("user_id = ?", uid).Count(&count)
	if count >= 10 {
		resp.ParamError(c, "最多只能添加10个分组")
		return
	}
	h.DB.Create(&model.FriendGroup{UserID: uid, Name: req.Name, Sort: req.Sort})
	resp.OK(c, nil)
}

func (h *FriendGroupHandler) Delete(c *gin.Context) {
	uid := middleware.GetUID(c)
	id, _ := strconv.Atoi(c.Param("id"))
	var g model.FriendGroup
	if err := h.DB.Where("id = ? AND user_id = ?", id, uid).First(&g).Error; err != nil {
		resp.NotFound(c, "分组不存在")
		return
	}
	h.DB.Where("group_id = ?", g.ID).Delete(&model.FriendGroupItem{})
	h.DB.Delete(&g)
	// 对齐诺哈 group_del_ok.asp：删除分组后该组好友归回未分组
	h.DB.Model(&model.Friendship{}).Where("user_id = ? AND group_id = ?", uid, id).Update("group_id", 0)
	resp.OK(c, nil)
}

type addGroupFriendReq struct {
	FriendID uint `json:"friend_id" binding:"required"`
}

// 把好友加入分组
func (h *FriendGroupHandler) AddFriend(c *gin.Context) {
	uid := middleware.GetUID(c)
	id, _ := strconv.Atoi(c.Param("id"))
	var g model.FriendGroup
	if err := h.DB.Where("id = ? AND user_id = ?", id, uid).First(&g).Error; err != nil {
		resp.NotFound(c, "分组不存在")
		return
	}
	var req addGroupFriendReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请选择好友")
		return
	}
	var exist int64
	h.DB.Model(&model.FriendGroupItem{}).Where("group_id = ? AND friend_id = ?", g.ID, req.FriendID).Count(&exist)
	if exist > 0 {
		resp.ParamError(c, "该好友已在此分组")
		return
	}
	h.DB.Create(&model.FriendGroupItem{GroupID: g.ID, UserID: uid, FriendID: req.FriendID})
	// 同步到好友关系（对齐诺哈 wap_friend.group）
	h.DB.Model(&model.Friendship{}).Where("user_id = ? AND friend_id = ? AND status = 1", uid, req.FriendID).Update("group_id", g.ID)
	dbUpdateGroupAmount(h.DB, uid, g.ID)
	resp.OK(c, nil)
}

func (h *FriendGroupHandler) RemoveFriend(c *gin.Context) {
	uid := middleware.GetUID(c)
	id, _ := strconv.Atoi(c.Param("id"))
	fid, _ := strconv.Atoi(c.Param("friendId"))
	h.DB.Where("group_id = ? AND user_id = ? AND friend_id = ?", id, uid, fid).Delete(&model.FriendGroupItem{})
	h.DB.Model(&model.Friendship{}).Where("user_id = ? AND friend_id = ? AND status = 1", uid, fid).Update("group_id", 0)
	dbUpdateGroupAmount(h.DB, uid, uint(id))
	resp.OK(c, nil)
}
