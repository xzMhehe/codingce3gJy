package handler

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 二战风云 地图坐标收藏（复刻原版地图页的「收藏列表」/home/map/indexstar.html）
// 原版为静态列表页，本项目做成真实可用的坐标收藏：存坐标 + 备注名，一键跳转。

// MapStars GET /games/ezfy/map/stars
func (h *EzfyHandler) MapStars(c *gin.Context) {
	uid := middleware.GetUID(c)
	list := []model.EzfyMapStar{}
	h.DB.Where("user_id = ?", uid).Order("id DESC").Find(&list)
	resp.OK(c, gin.H{"stars": list})
}

// MapStarAdd POST /games/ezfy/map/stars  {x,y,name}
func (h *EzfyHandler) MapStarAdd(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		X    int    `json:"x"`
		Y    int    `json:"y"`
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if req.X < 1 || req.X > 500 || req.Y < 1 || req.Y > 500 {
		resp.ParamError(c, "坐标需在 1~500 之间")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = "(" + strconv.Itoa(req.X) + "," + strconv.Itoa(req.Y) + ")"
	}
	if len([]rune(name)) > 16 {
		resp.ParamError(c, "备注名最多16字")
		return
	}
	var cnt int64
	h.DB.Model(&model.EzfyMapStar{}).Where("user_id = ? AND x = ? AND y = ?", uid, req.X, req.Y).Count(&cnt)
	if cnt > 0 {
		resp.ParamError(c, "该坐标已收藏")
		return
	}
	h.DB.Create(&model.EzfyMapStar{UserID: uid, X: req.X, Y: req.Y, Name: name})
	resp.OK(c, gin.H{"msg": "已收藏 (" + strconv.Itoa(req.X) + "," + strconv.Itoa(req.Y) + ")"})
}

// MapStarDelete POST /games/ezfy/map/stars/delete  {id}
func (h *EzfyHandler) MapStarDelete(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Id int64 `json:"id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	h.DB.Where("id = ? AND user_id = ?", req.Id, uid).Delete(&model.EzfyMapStar{})
	resp.OK(c, gin.H{"msg": "已取消收藏"})
}
