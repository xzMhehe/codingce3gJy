package ezfy

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

// MapStarDeleteXY POST /games/ezfy/map/stars/delete-xy  {x,y}
//
// ★★ 2026-10-25 用户「点了[已收藏]没反应，弹正在同步收藏列表」：
// 原来取消收藏必须先从**收藏列表**里 find 到那条记录的 id，列表还没同步时前端只能提示
// 「请稍后再点一次」—— 玩家点了不动，体验很差。
// 这里按**坐标**取消，前端不再依赖列表 id；删 0 行也返回成功，
// 顺带把「按钮显示已收藏、库里其实没有」的脏状态**自愈**（再点一次就回到「收藏」）。
func (h *EzfyHandler) MapStarDeleteXY(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		X int `json:"x"`
		Y int `json:"y"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if req.X < 1 || req.X > 500 || req.Y < 1 || req.Y > 500 {
		resp.ParamError(c, "坐标需在 1~500 之间")
		return
	}
	h.DB.Where("user_id = ? AND x = ? AND y = ?", uid, req.X, req.Y).Delete(&model.EzfyMapStar{})
	resp.OK(c, gin.H{"msg": "已取消收藏"})
}
