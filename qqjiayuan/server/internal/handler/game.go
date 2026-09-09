package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

type GameHandler struct{ DB *gorm.DB }

// 游戏可用的 logo 素材（static/image 下）
var gameLogoPresets = []string{
	"logo.jpg", "mofahuayuan.gif", "hunli2.jpg", "kaixinnongchang.gif",
	"kuangqiangchewei.gif", "jwt.png", "cwlogo.gif", "shuiguoleyuan.gif",
	"quanminliema.gif", "jiayuangushi.gif", "dahuachuiniu.gif",
}

// 公开：游戏列表（大厅用）
func (h *GameHandler) List(c *gin.Context) {
	var games []model.Game
	h.DB.Where("status = 1").Order("sort ASC, id ASC").Find(&games)
	resp.OK(c, games)
}

// 我的游戏（对齐诺哈 game_list.asp：按 sort 排序，含上移下移）
func (h *GameHandler) MyList(c *gin.Context) {
	uid := middleware.GetUID(c)
	var rows []model.MyGame
	h.DB.Preload("Game").Where("user_id = ?", uid).Order("sort ASC, id ASC").Find(&rows)
	out := []gin.H{}
	for _, r := range rows {
		if r.Game != nil {
			out = append(out, gin.H{"id": r.Game.ID, "name": r.Game.Name, "logo": r.Game.Logo,
				"desc": r.Game.Desc, "stars": r.Game.Stars, "sort": r.Sort})
		}
	}
	resp.OK(c, out)
}

// 我的游戏排序（对齐诺哈 game_move.asp：dir=up 上移 / down 下移）
func (h *GameHandler) MyMove(c *gin.Context) {
	uid := middleware.GetUID(c)
	gid, _ := strconv.Atoi(c.Param("gameId"))
	dir := c.Query("dir")
	var cur model.MyGame
	if err := h.DB.Where("user_id = ? AND game_id = ?", uid, gid).First(&cur).Error; err != nil {
		resp.NotFound(c, "该游戏不在你的游戏中")
		return
	}
	// 先重编号（1..n），保证 sort 互异
	h.renumberMyGames(uid)
	h.DB.First(&cur, cur.ID)
	var curSort, peerSort int
	if dir == "up" {
		var prev model.MyGame
		if err := h.DB.Where("user_id = ? AND sort = ?", uid, cur.Sort-1).First(&prev).Error; err != nil {
			resp.ParamError(c, "已经在最前面啦")
			return
		}
		curSort, peerSort = cur.Sort, prev.Sort
		h.DB.Model(&model.MyGame{}).Where("id = ?", cur.ID).Update("sort", peerSort)
		h.DB.Model(&model.MyGame{}).Where("id = ?", prev.ID).Update("sort", curSort)
	} else if dir == "down" {
		var next model.MyGame
		if err := h.DB.Where("user_id = ? AND sort = ?", uid, cur.Sort+1).First(&next).Error; err != nil {
			resp.ParamError(c, "已经在最后面啦")
			return
		}
		curSort, peerSort = cur.Sort, next.Sort
		h.DB.Model(&model.MyGame{}).Where("id = ?", cur.ID).Update("sort", peerSort)
		h.DB.Model(&model.MyGame{}).Where("id = ?", next.ID).Update("sort", curSort)
	} else {
		resp.ParamError(c, "dir 必须是 up 或 down")
		return
	}
	h.renumberMyGames(uid)
	resp.OK(c, "排序已更新")
}

// renumberMyGames 重新编号（对齐诺哈 sort 连续，保证排序稳定）
func (h *GameHandler) renumberMyGames(uid uint) {
	var rows []model.MyGame
	h.DB.Where("user_id = ?", uid).Order("sort ASC, id ASC").Find(&rows)
	for i, r := range rows {
		if r.Sort != i+1 {
			h.DB.Model(&model.MyGame{}).Where("id = ?", r.ID).Update("sort", i+1)
		}
	}
}

type addMyGameReq struct {
	GameID uint `json:"game_id" binding:"required"`
}

func (h *GameHandler) MyAdd(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req addMyGameReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请选择游戏")
		return
	}
	var g model.Game
	if err := h.DB.First(&g, req.GameID).Error; err != nil {
		resp.NotFound(c, "游戏不存在")
		return
	}
	var exist int64
	h.DB.Model(&model.MyGame{}).Where("user_id = ? AND game_id = ?", uid, req.GameID).Count(&exist)
	if exist > 0 {
		resp.ParamError(c, "该游戏已在你的游戏中")
		return
	}
	// 追加到末尾（对齐诺哈 wap_game_bag：sort 递增）
	var maxSort int
	h.DB.Model(&model.MyGame{}).Where("user_id = ?", uid).Select("COALESCE(MAX(sort), 0)").Scan(&maxSort)
	h.DB.Create(&model.MyGame{UserID: uid, GameID: req.GameID, Sort: maxSort + 1})
	resp.OK(c, nil)
}

func (h *GameHandler) MyRemove(c *gin.Context) {
	uid := middleware.GetUID(c)
	gid, _ := strconv.Atoi(c.Param("gameId"))
	h.DB.Where("user_id = ? AND game_id = ?", uid, gid).Delete(&model.MyGame{})
	resp.OK(c, nil)
}

// 后台：分页列表
func (h *GameHandler) AdminList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 10
	}
	q := h.DB.Model(&model.Game{})
	var total int64
	q.Count(&total)
	if maxPage := int(total+int64(size)-1) / size; maxPage < 1 {
		page = 1
	} else if page > maxPage {
		page = maxPage
	}
	var games []model.Game
	q.Order("sort ASC, id ASC").Offset((page - 1) * size).Limit(size).Find(&games)
	resp.OK(c, gin.H{"list": games, "total": total, "page": page, "size": size})
}

type gameReq struct {
	Name     string `json:"name" binding:"required,min=1,max=30"`
	Category string `json:"category" binding:"required,oneof=net com"`
	Logo     string `json:"logo" binding:"max=50"`
	Stars    string `json:"stars" binding:"max=10"`
	Desc     string `json:"desc" binding:"max=100"`
	Intro    string `json:"intro" binding:"max=200"`
	Path     string `json:"path" binding:"max=100"`
	Url      string `json:"url" binding:"max=200"`
	BoardID  uint   `json:"board_id"`
	Sort     int    `json:"sort"`
	Status   int    `json:"status"`
}

func (h *GameHandler) Create(c *gin.Context) {
	var req gameReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "游戏名必填，类型必须是 net(网络游戏) 或 com(社区游戏)")
		return
	}
	if req.Logo != "" && !h.validLogo(req.Logo) {
		resp.ParamError(c, "logo 不在素材库中")
		return
	}
	if req.Status == 0 {
		req.Status = 1
	}
	if req.Stars == "" {
		req.Stars = "★★★"
	}
	g := model.Game{Name: req.Name, Category: req.Category, Logo: req.Logo, Stars: req.Stars,
		Desc: req.Desc, Intro: req.Intro, Path: req.Path, Url: req.Url, BoardID: req.BoardID, Sort: req.Sort, Status: req.Status}
	h.DB.Create(&g)
	resp.OK(c, g)
}

func (h *GameHandler) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req gameReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "游戏名必填，类型必须是 net 或 com")
		return
	}
	if req.Logo != "" && !h.validLogo(req.Logo) {
		resp.ParamError(c, "logo 不在素材库中")
		return
	}
	var g model.Game
	if err := h.DB.First(&g, id).Error; err != nil {
		resp.NotFound(c, "游戏不存在")
		return
	}
	h.DB.Model(&g).Updates(map[string]interface{}{
		"name": req.Name, "category": req.Category, "logo": req.Logo,
		"stars": req.Stars, "desc": req.Desc, "intro": req.Intro, "path": req.Path, "url": req.Url,
		"board_id": req.BoardID, "sort": req.Sort, "status": req.Status,
	})
	resp.OK(c, g)
}

// validLogo 校验 logo 是否在资源库（game 类，启用中）或预置清单
func (h *GameHandler) validLogo(logo string) bool {
	var n int64
	h.DB.Model(&model.Resource{}).
		Where("category = ? AND status = 1 AND file LIKE ?", "game", "%/"+logo).
		Count(&n)
	if n > 0 {
		return true
	}
	for _, p := range model.GameLogoPresets {
		if p == logo {
			return true
		}
	}
	return false
}

func (h *GameHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.Game{}, id)
	resp.OK(c, nil)
}
