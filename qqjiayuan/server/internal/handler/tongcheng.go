package handler

import (
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 同城客栈（复刻诺哈三代 bbs/city 系列）：
// 省份=同城客栈分区子板块，城市=省份板块子板块（带区号 city_code）
// city.asp 城市主页：在线老乡/老乡聊天/议事论坛/同城管理/同城资料
type CityHandler struct{ DB *gorm.DB }

// 人气去重缓存（参考诺哈 Application("BClick_"&bid&"_"&ClientIp) 会话级去重）
var cityClickCache sync.Map

// tongchengChannel 返回同城客栈分区（找不到返回 ID=0）
func (h *CityHandler) tongchengChannel() model.Board {
	var ch model.Board
	h.DB.Where("parent_id = 0 AND name = ?", "同城客栈").First(&ch)
	return ch
}

func (h *CityHandler) gongjianID() uint {
	var b model.Board
	h.DB.Select("id").Where("parent_id = ? AND name = ?", h.tongchengChannel().ID, "共建同城").First(&b)
	return b.ID
}

// Home 同城首页：省份列表（含城市数/在线人数）+ 共建同城入口
func (h *CityHandler) Home(c *gin.Context) {
	ch := h.tongchengChannel()
	if ch.ID == 0 {
		resp.OK(c, gin.H{"provinces": []gin.H{}, "gongjian_id": 0})
		return
	}
	var provinces []model.Board
	h.DB.Where("parent_id = ? AND status = 1 AND id <> ?", ch.ID, h.gongjianID()).Order("sort ASC, id ASC").Find(&provinces)

	tenMinAgo := time.Now().Add(-10 * time.Minute)
	type provItem struct {
		model.Board
		CityCount int `json:"city_count"`
		Online    int `json:"online"`
	}
	list := []provItem{}
	var totalCities int64
	for _, p := range provinces {
		var cityIDs []uint
		h.DB.Model(&model.Board{}).Where("parent_id = ?", p.ID).Pluck("id", &cityIDs)
		online := int64(0)
		if len(cityIDs) > 0 {
			h.DB.Model(&model.User{}).Where("last_board_id IN ? AND last_active_at > ?", cityIDs, tenMinAgo).Count(&online)
		}
		list = append(list, provItem{Board: p, CityCount: len(cityIDs), Online: int(online)})
		totalCities += int64(len(cityIDs))
	}

	// 全国在线老乡（同城所有城市板块）
	var allCityIDs []uint
	h.DB.Model(&model.Board{}).Where("parent_id IN ?", parentIDs(provinces)).Pluck("id", &allCityIDs)
	onlineAll := int64(0)
	if len(allCityIDs) > 0 {
		h.DB.Model(&model.User{}).Where("last_board_id IN ? AND last_active_at > ?", allCityIDs, tenMinAgo).Count(&onlineAll)
	}

	resp.OK(c, gin.H{"provinces": list, "total_provinces": len(list), "total_cities": totalCities,
		"online": onlineAll, "gongjian_id": h.gongjianID(), "channel": ch})
}

func parentIDs(boards []model.Board) []uint {
	ids := make([]uint, 0, len(boards))
	for _, b := range boards {
		ids = append(ids, b.ID)
	}
	return ids
}

// Province 省份城市列表（参考 city_list.asp，分页）
func (h *CityHandler) Province(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var prov model.Board
	if err := h.DB.First(&prov, id).Error; err != nil {
		resp.NotFound(c, "省份不存在")
		return
	}
	page, offset, size := pageOf(c, 10)
	var total int64
	h.DB.Model(&model.Board{}).Where("parent_id = ? AND status = 1", prov.ID).Count(&total)
	var cities []model.Board
	h.DB.Where("parent_id = ? AND status = 1", prov.ID).Order("sort ASC, id ASC").Offset(offset).Limit(size).Find(&cities)

	tenMinAgo := time.Now().Add(-10 * time.Minute)
	items := []gin.H{}
	for _, ct := range cities {
		var online int64
		h.DB.Model(&model.User{}).Where("last_board_id = ? AND last_active_at > ?", ct.ID, tenMinAgo).Count(&online)
		items = append(items, gin.H{"id": ct.ID, "name": ct.Name, "city_code": ct.CityCode,
			"description": ct.Description, "online": online})
	}
	if maxPage := int(total+int64(size)-1) / size; maxPage < 1 {
		page = 1
	} else if page > maxPage {
		page = maxPage
	}
	resp.OK(c, gin.H{"province": prov, "list": items, "total": total, "page": page, "size": size})
}

// cityHeartbeat 登录用户心跳：记录所在板块（参考诺哈 wap_online.bbsid）
func (h *CityHandler) cityHeartbeat(uid, boardID uint) {
	if uid != 0 {
		h.DB.Model(&model.User{}).Where("id = ?", uid).Updates(map[string]interface{}{
			"last_board_id": boardID, "last_active_at": time.Now(),
		})
	}
}

// Detail 城市主页（参考 city.asp）：人气/在线/创建人/议事论坛精华/同城管理
func (h *CityHandler) Detail(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var city model.Board
	if err := h.DB.First(&city, id).Error; err != nil {
		resp.NotFound(c, "该城市不存在")
		return
	}
	uid := middleware.GetUID(c)
	h.cityHeartbeat(uid, city.ID)

	// 人气：参考诺哈按 IP 会话去重，这里按用户/游客+IP 10 分钟去重
	key := "city_click_" + strconv.Itoa(id) + "_" + strconv.Itoa(int(uid)) + "_" + c.ClientIP()
	if v, ok := cityClickCache.Load(key); !ok || time.Since(v.(time.Time)) > 10*time.Minute {
		cityClickCache.Store(key, time.Now())
		h.DB.Model(&city).UpdateColumn("click", gorm.Expr("click + 1"))
		city.Click++
	}

	// 在线老乡：10 分钟内活跃且停留在本城市板块
	tenMinAgo := time.Now().Add(-10 * time.Minute)
	var online int64
	h.DB.Model(&model.User{}).Where("last_board_id = ? AND last_active_at > ?", city.ID, tenMinAgo).Count(&online)

	// 议事论坛精华 TOP4（参考 city.asp TOP 4 fine=1）
	fineThreads := []model.Thread{}
	h.DB.Preload("User").Where("board_id = ? AND is_fine = 1 AND status = 1 AND audit_status = 1", city.ID).
		Order("id DESC").Limit(4).Find(&fineThreads)

	// 同城管理（wap_manage）
	managers := []model.CityManager{}
	h.DB.Preload("User").Where("board_id = ?", city.ID).Order("sort ASC, id ASC").Find(&managers)

	// 创建人
	var creator *model.User
	if city.CreatorID != 0 {
		h.DB.First(&creator, city.CreatorID)
	}
	// 省份（上级板块）
	var prov model.Board
	h.DB.First(&prov, city.ParentID)

	resp.OK(c, gin.H{"city": city, "province": prov, "creator": creator,
		"online": online, "fine_threads": fineThreads, "managers": managers})
}

// Online 在线老乡（参考 online.asp：停留在本城市板块的活跃用户）
func (h *CityHandler) Online(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var city model.Board
	if err := h.DB.First(&city, id).Error; err != nil {
		resp.NotFound(c, "该城市不存在")
		return
	}
	h.cityHeartbeat(middleware.GetUID(c), city.ID)
	page, offset, size := pageOf(c, 10)
	tenMinAgo := time.Now().Add(-10 * time.Minute)
	var total int64
	h.DB.Model(&model.User{}).Where("last_board_id = ? AND last_active_at > ?", city.ID, tenMinAgo).Count(&total)
	type onlineUser struct {
		ID       uint      `json:"id"`
		Username string    `json:"username"`
		Nickname string    `json:"nickname"`
		Color    string    `json:"color"`
		Level    int       `json:"level"`
		LastAt   time.Time `gorm:"column:last_active_at" json:"last_active_at"`
	}
	var users []onlineUser
	h.DB.Model(&model.User{}).Select("id, username, nickname, color, level, last_active_at").
		Where("last_board_id = ? AND last_active_at > ?", city.ID, tenMinAgo).
		Order("last_active_at DESC").Offset(offset).Limit(size).Scan(&users)
	if maxPage := int(total+int64(size)-1) / size; maxPage < 1 {
		page = 1
	} else if page > maxPage {
		page = maxPage
	}
	resp.OK(c, gin.H{"city": gin.H{"id": city.ID, "name": city.Name}, "list": users, "total": total, "page": page, "size": size})
}

// Lookup 按区号/城市名进入（参考 city_in.asp）
func (h *CityHandler) Lookup(c *gin.Context) {
	code := c.Query("code")
	name := c.Query("name")
	if code == "" && name == "" {
		resp.ParamError(c, "请输入区号或城市名")
		return
	}
	var city model.Board
	var err error
	if code != "" {
		err = h.DB.Where("city_code = ? AND status = 1", code).First(&city).Error
	} else {
		err = h.DB.Where("name = ? AND city_code <> '' AND status = 1", name).First(&city).Error
	}
	if err != nil {
		resp.NotFound(c, "没有找到这个城市，快去「共建同城」申请吧！")
		return
	}
	resp.OK(c, gin.H{"id": city.ID, "name": city.Name, "city_code": city.CityCode, "parent_id": city.ParentID})
}

// ---- 管理端 ----

// AdminTree 同城板块树：省份 → 城市
func (h *CityHandler) AdminTree(c *gin.Context) {
	ch := h.tongchengChannel()
	var provinces []model.Board
	if ch.ID > 0 {
		h.DB.Where("parent_id = ?", ch.ID).Order("sort ASC, id ASC").Find(&provinces)
	}
	type cityNode struct {
		model.Board
		ThreadCount2 int64                  `json:"thread_count2"`
		Managers     []model.CityManager    `json:"managers"`
	}
	type provNode struct {
		model.Board
		Cities []cityNode `json:"cities"`
	}
	nodes := []provNode{}
	for _, p := range provinces {
		pn := provNode{Board: p, Cities: []cityNode{}}
		var cities []model.Board
		h.DB.Where("parent_id = ?", p.ID).Order("sort ASC, id ASC").Find(&cities)
		for _, ct := range cities {
			cn := cityNode{Board: ct, Managers: []model.CityManager{}}
			h.DB.Model(&model.Thread{}).Where("board_id = ? AND status = 1", ct.ID).Count(&cn.ThreadCount2)
			h.DB.Preload("User").Where("board_id = ?", ct.ID).Order("sort ASC, id ASC").Find(&cn.Managers)
			pn.Cities = append(pn.Cities, cn)
		}
		nodes = append(nodes, pn)
	}
	// 共建同城（特殊板块，不属于省份）
	var gongjian *model.Board
	if gid := h.gongjianID(); gid > 0 {
		var b model.Board
		if err := h.DB.First(&b, gid).Error; err == nil {
			gongjian = &b
		}
	}
	resp.OK(c, gin.H{"channel": ch, "provinces": nodes, "gongjian": gongjian})
}

// AdminCreateProvince 新增省份
func (h *CityHandler) AdminCreateProvince(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required,min=1,max=30"`
		Description string `json:"description"`
		Sort        int    `json:"sort"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "省份名称1-30字")
		return
	}
	ch := h.tongchengChannel()
	if ch.ID == 0 {
		resp.ParamError(c, "同城客栈分区不存在")
		return
	}
	var n int64
	h.DB.Model(&model.Board{}).Where("parent_id = ? AND name = ?", ch.ID, req.Name).Count(&n)
	if n > 0 {
		resp.ParamError(c, "该省份已存在")
		return
	}
	b := model.Board{Name: req.Name, ParentID: ch.ID, Description: req.Description, Sort: req.Sort, Status: 1}
	h.DB.Create(&b)
	resp.OK(c, b)
}

// AdminUpdateProvince 编辑省份
func (h *CityHandler) AdminUpdateProvince(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Name        string `json:"name" binding:"required,min=1,max=30"`
		Description string `json:"description"`
		Sort        int    `json:"sort"`
		Status      int    `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "省份名称1-30字")
		return
	}
	var b model.Board
	if err := h.DB.First(&b, id).Error; err != nil {
		resp.NotFound(c, "省份不存在")
		return
	}
	h.DB.Model(&b).Updates(map[string]interface{}{
		"name": req.Name, "description": req.Description, "sort": req.Sort,
		"status": boolToInt(req.Status != 0),
	})
	resp.OK(c, b)
}

// AdminDeleteProvince 删除省份（需先清空城市）
func (h *CityHandler) AdminDeleteProvince(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var n int64
	h.DB.Model(&model.Board{}).Where("parent_id = ?", id).Count(&n)
	if n > 0 {
		resp.ParamError(c, "请先删除该省份下的城市")
		return
	}
	h.DB.Delete(&model.Board{}, id)
	resp.OK(c, nil)
}

// AdminCreateCity 新增城市
func (h *CityHandler) AdminCreateCity(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		ProvinceID  uint   `json:"province_id" binding:"required"`
		Name        string `json:"name" binding:"required,min=1,max=30"`
		CityCode    string `json:"city_code"`
		Description string `json:"description"`
		Sort        int    `json:"sort"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "城市名称1-30字")
		return
	}
	var prov model.Board
	if err := h.DB.First(&prov, req.ProvinceID).Error; err != nil || prov.ParentID != h.tongchengChannel().ID {
		resp.ParamError(c, "省份不存在")
		return
	}
	if req.CityCode != "" {
		var n int64
		h.DB.Model(&model.Board{}).Where("city_code = ?", req.CityCode).Count(&n)
		if n > 0 {
			resp.ParamError(c, "该区号已被其他城市使用")
			return
		}
	}
	b := model.Board{Name: req.Name, ParentID: prov.ID, CityCode: req.CityCode,
		Description: req.Description, Sort: req.Sort, Status: 1, CreatorID: uid}
	h.DB.Create(&b)
	resp.OK(c, b)
}

// AdminUpdateCity 编辑城市
func (h *CityHandler) AdminUpdateCity(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Name        string `json:"name" binding:"required,min=1,max=30"`
		CityCode    string `json:"city_code"`
		Description string `json:"description"`
		Sort        int    `json:"sort"`
		Status      int    `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "城市名称1-30字")
		return
	}
	var b model.Board
	if err := h.DB.First(&b, id).Error; err != nil {
		resp.NotFound(c, "城市不存在")
		return
	}
	if req.CityCode != "" && req.CityCode != b.CityCode {
		var n int64
		h.DB.Model(&model.Board{}).Where("city_code = ? AND id <> ?", req.CityCode, b.ID).Count(&n)
		if n > 0 {
			resp.ParamError(c, "该区号已被其他城市使用")
			return
		}
	}
	h.DB.Model(&b).Updates(map[string]interface{}{
		"name": req.Name, "city_code": req.CityCode, "description": req.Description,
		"sort": req.Sort, "status": boolToInt(req.Status != 0),
	})
	resp.OK(c, b)
}

// AdminDeleteCity 删除城市（需先无帖子/无管理）
func (h *CityHandler) AdminDeleteCity(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var n int64
	h.DB.Model(&model.Thread{}).Where("board_id = ? AND status = 1", id).Count(&n)
	if n > 0 {
		resp.ParamError(c, "该城市还有帖子，不能删除")
		return
	}
	h.DB.Where("board_id = ?", id).Delete(&model.CityManager{})
	h.DB.Delete(&model.Board{}, id)
	resp.OK(c, nil)
}

// AdminManagers 某城市的同城管理列表
func (h *CityHandler) AdminManagers(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var list []model.CityManager
	h.DB.Preload("User").Where("board_id = ?", id).Order("sort ASC, id ASC").Find(&list)
	resp.OK(c, list)
}

// AdminManagerAdd 任命同城管理
func (h *CityHandler) AdminManagerAdd(c *gin.Context) {
	var req struct {
		BoardID uint   `json:"board_id" binding:"required"`
		UserID  uint   `json:"user_id" binding:"required"`
		Title   string `json:"title" binding:"required,min=1,max=30"`
		Sort    int    `json:"sort"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请填写城市、用户ID和职务名称")
		return
	}
	var u model.User
	if err := h.DB.First(&u, req.UserID).Error; err != nil {
		resp.NotFound(c, "用户不存在")
		return
	}
	m := model.CityManager{BoardID: req.BoardID, UserID: req.UserID, Title: req.Title, Sort: req.Sort}
	h.DB.Create(&m)
	resp.OK(c, m)
}

// AdminManagerDelete 免除同城管理
func (h *CityHandler) AdminManagerDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.CityManager{}, id)
	resp.OK(c, nil)
}
