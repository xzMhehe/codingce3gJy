package handler

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 二战风云管理端 —— 扩展模块（城市/建筑/建筑队列/兵种/征兵/军官/资源/科技/地图/军团/私聊）
// 与 ezfy_admin.go 的 4 个基础模块（玩家/流水/系统/数据）并列，共用 ezfyAdminName 等辅助函数。

// ezfyH 返回一个带 DB 的 EzfyHandler，用于复用游戏内结算逻辑
func (h *AdminHandler) ezfyH() *EzfyHandler {
	ez := &EzfyHandler{DB: h.DB}
	ez.cfgs()
	return ez
}

// ezfyCityOf 取城池，不存在时返回错误信息
func (h *AdminHandler) ezfyCityOf(id int64) (*model.EzfyCity, string) {
	var ct model.EzfyCity
	if err := h.DB.First(&ct, id).Error; err != nil {
		return nil, "城池不存在"
	}
	return &ct, ""
}

// ============ 1. 城市管理 ============

// AdminEzfyCities 城池列表（word=城名/用户ID/玩家昵称）
func (h *AdminHandler) AdminEzfyCities(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	word := strings.TrimSpace(c.Query("word"))
	q := h.DB.Model(&model.EzfyCity{})
	if word != "" {
		if uid, err := strconv.Atoi(word); err == nil {
			q = q.Where("user_id = ? OR id = ?", uid, uid)
		} else {
			var ids []uint
			h.DB.Model(&model.EzfyProfile{}).Select("user_id").
				Where("nickname LIKE ?", "%"+word+"%").Scan(&ids)
			if len(ids) > 0 {
				q = q.Where("name LIKE ? OR user_id IN ?", "%"+word+"%", ids)
			} else {
				q = q.Where("name LIKE ?", "%"+word+"%")
			}
		}
	}
	var total int64
	q.Count(&total)
	var rows []model.EzfyCity
	q.Order("id DESC").Offset(offset).Limit(size).Find(&rows)

	type rowOut struct {
		model.EzfyCity
		PlayerName  string `json:"player_name"`
		HomeNum     string `json:"home_num"`
		CampName    string `json:"camp_name"`
		BuildingNum int64  `json:"building_num"`
		TroopNum    int64  `json:"troop_num"`
		OfficerNum  int64  `json:"officer_num"`
		WildNum     int64  `json:"wild_num"`
		TerrainName string `json:"terrain_name"`
		// ★ 管理端「城市管理」展示列：玩家所在州（大洲/大洋名）
		Continent string `json:"continent"`
	}
	out := []rowOut{}
	for _, ct := range rows {
		pn, hn := h.ezfyAdminName(ct.UserID)
		var bn, tn, on, wn int64
		h.DB.Model(&model.EzfyCityBuilding{}).Where("city_id = ? AND level > 0", ct.ID).Count(&bn)
		h.DB.Model(&model.EzfyCityTroop{}).Where("city_id = ? AND count > 0", ct.ID).Count(&tn)
		h.DB.Model(&model.EzfyOfficer{}).Where("city_id = ?", ct.ID).Count(&on)
		h.DB.Model(&model.EzfyWildland{}).Where("city_id = ?", ct.ID).Count(&wn)
		var p model.EzfyProfile
		camp := ""
		if err := h.DB.Where("user_id = ?", ct.UserID).First(&p).Error; err == nil {
			camp = ezfyCampName(p.Camp)
		}
		out = append(out, rowOut{EzfyCity: ct, PlayerName: pn, HomeNum: hn, CampName: camp,
			BuildingNum: bn, TroopNum: tn, OfficerNum: on, WildNum: wn,
			TerrainName: ezfyTerrainNameEx(ct.X, ct.Y),
			Continent:   ezfyRegionName(ct.X, ct.Y)})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// AdminEzfyCityDetail 城池详情（建筑/部队/科技/军官/野地/出征）
func (h *AdminHandler) AdminEzfyCityDetail(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	ct, msg := h.ezfyCityOf(int64(id))
	if ct == nil {
		resp.NotFound(c, msg)
		return
	}
	pn, hn := h.ezfyAdminName(ct.UserID)
	var buildings []model.EzfyCityBuilding
	h.DB.Where("city_id = ?", ct.ID).Order("building_id").Find(&buildings)
	type bOut struct {
		model.EzfyCityBuilding
		CfgName string `json:"cfg_name"`
		CfgType int    `json:"cfg_type"`
	}
	bViews := []bOut{}
	for _, b := range buildings {
		name, typ := "", 0
		var cfg model.EzfyCfgBuilding
		if err := h.DB.First(&cfg, b.BuildingId).Error; err == nil {
			name, typ = cfg.Name, cfg.Type
		}
		bViews = append(bViews, bOut{EzfyCityBuilding: b, CfgName: name, CfgType: typ})
	}
	var troops []model.EzfyCityTroop
	h.DB.Where("city_id = ?", ct.ID).Find(&troops)
	type tOut struct {
		model.EzfyCityTroop
		CfgName string `json:"cfg_name"`
	}
	tViews := []tOut{}
	for _, t := range troops {
		name := ""
		var cfg model.EzfyCfgTroop
		if err := h.DB.First(&cfg, t.TroopId).Error; err == nil {
			name = cfg.Name
		}
		tViews = append(tViews, tOut{EzfyCityTroop: t, CfgName: name})
	}
	var techs []model.EzfyCityTech
	h.DB.Where("city_id = ?", ct.ID).Find(&techs)
	type cOut struct {
		model.EzfyCityTech
		CfgName string `json:"cfg_name"`
	}
	cViews := []cOut{}
	for _, t := range techs {
		name := ""
		var cfg model.EzfyCfgTech
		if err := h.DB.First(&cfg, t.TechId).Error; err == nil {
			name = cfg.Name
		}
		cViews = append(cViews, cOut{EzfyCityTech: t, CfgName: name})
	}
	var officers []model.EzfyOfficer
	h.DB.Where("city_id = ?", ct.ID).Order("id").Find(&officers)
	var wilds []model.EzfyWildland
	h.DB.Where("city_id = ?", ct.ID).Find(&wilds)
	var orders []model.EzfyOrder
	h.DB.Where("city_id = ?", ct.ID).Order("id DESC").Limit(20).Find(&orders)
	resp.OK(c, gin.H{
		"city": ct, "player_name": pn, "home_num": hn,
		"terrain_name": ezfyTerrainNameEx(ct.X, ct.Y),
		"buildings":    bViews, "troops": tViews, "techs": cViews,
		"officers": officers, "wildlands": wilds, "orders": orders,
	})
}

// AdminEzfyCityUpdate 修改城池（城名/坐标/民心/民怨/税率/人口/资源）
func (h *AdminHandler) AdminEzfyCityUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	ct, msg := h.ezfyCityOf(int64(id))
	if ct == nil {
		resp.NotFound(c, msg)
		return
	}
	var in map[string]interface{}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	updates := map[string]interface{}{}
	if v, ok := in["name"]; ok {
		if s := strings.TrimSpace(fmt.Sprint(v)); s != "" {
			updates["name"] = trimStr(s, 50)
		}
	}
	num := func(key string, min int64) {
		if v, ok := in[key]; ok {
			if f, err := strconv.ParseFloat(fmt.Sprint(v), 64); err == nil {
				n := int64(f)
				if n < min {
					n = min
				}
				updates[key] = n
			}
		}
	}
	for _, k := range []string{"x", "y", "city_level", "feelings", "grievance", "tax_rate",
		"pop", "pop_max", "gold", "food", "steel", "oil", "rare",
		"gold_cap", "food_cap", "steel_cap", "oil_cap", "rare_cap",
		"rate_food", "rate_steel", "rate_oil", "rate_rare",
		"ware_food", "ware_steel", "ware_oil", "ware_rare"} {
		num(k, 0)
	}
	if len(updates) == 0 {
		resp.ParamError(c, "无可修改字段")
		return
	}
	if err := h.DB.Model(&model.EzfyCity{}).Where("id = ?", ct.ID).Updates(updates).Error; err != nil {
		resp.ParamError(c, "修改失败："+err.Error())
		return
	}
	resp.OK(c, gin.H{"msg": "城池已更新"})
}

// AdminEzfyCityReset 重置城池（清空建筑/部队/科技/军官/野地/队列，回到初始状态）
func (h *AdminHandler) AdminEzfyCityReset(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	ct, msg := h.ezfyCityOf(int64(id))
	if ct == nil {
		resp.NotFound(c, msg)
		return
	}
	cid := ct.ID
	h.DB.Where("city_id = ?", cid).Delete(&model.EzfyCityBuilding{})
	h.DB.Where("city_id = ?", cid).Delete(&model.EzfyCityTroop{})
	h.DB.Where("city_id = ?", cid).Delete(&model.EzfyCityTech{})
	h.DB.Where("city_id = ?", cid).Delete(&model.EzfyTrainQueue{})
	h.DB.Where("city_id = ?", cid).Delete(&model.EzfyWildland{})
	h.DB.Where("city_id = ?", cid).Delete(&model.EzfyWounded{})
	h.DB.Where("city_id = ?", cid).Delete(&model.EzfyCityEffect{})
	h.DB.Where("city_id = ?", cid).Delete(&model.EzfyCityTarget{})
	h.DB.Where("city_id = ?", cid).Delete(&model.EzfyOfficer{})
	h.DB.Where("city_id = ?", cid).Delete(&model.EzfyOrder{})
	// 补回 1 级市政厅
	h.DB.Create(&model.EzfyCityBuilding{CityId: int64(cid), BuildingId: 1, Level: 1, Status: 0})
	h.DB.Model(&model.EzfyCity{}).Where("id = ?", cid).Updates(map[string]interface{}{
		"city_level": 1, "pop": 0, "feelings": 100, "grievance": 0,
	})
	resp.OK(c, gin.H{"msg": "城池已重置（建筑/部队/科技/军官/野地全部清空）"})
}

// AdminEzfyCityDelete 拆除城池
func (h *AdminHandler) AdminEzfyCityDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	ct, msg := h.ezfyCityOf(int64(id))
	if ct == nil {
		resp.NotFound(c, msg)
		return
	}
	cid := ct.ID
	h.DB.Where("city_id = ?", cid).Delete(&model.EzfyCityBuilding{})
	h.DB.Where("city_id = ?", cid).Delete(&model.EzfyCityTroop{})
	h.DB.Where("city_id = ?", cid).Delete(&model.EzfyCityTech{})
	h.DB.Where("city_id = ?", cid).Delete(&model.EzfyTrainQueue{})
	h.DB.Where("city_id = ?", cid).Delete(&model.EzfyWildland{})
	h.DB.Where("city_id = ?", cid).Delete(&model.EzfyWounded{})
	h.DB.Where("city_id = ?", cid).Delete(&model.EzfyCityEffect{})
	h.DB.Where("city_id = ?", cid).Delete(&model.EzfyCityTarget{})
	h.DB.Where("city_id = ?", cid).Delete(&model.EzfyOfficer{})
	h.DB.Where("city_id = ?", cid).Delete(&model.EzfyOrder{})
	h.DB.Delete(&model.EzfyCity{}, cid)
	resp.OK(c, gin.H{"msg": "城池「" + ct.Name + "」已拆除"})
}

// ============ 2. 建筑管理 ============

// AdminEzfyBuildings 玩家建筑列表（word=城名/玩家，city_id 精确筛选，status 状态筛选）
func (h *AdminHandler) AdminEzfyBuildings(c *gin.Context) {
	page, offset, size := pageOf(c, 15)
	word := strings.TrimSpace(c.Query("word"))
	cityId := int64(atoiOr(c.Query("city_id"), 0))
	status := atoiOr(c.Query("status"), -1)
	q := h.DB.Model(&model.EzfyCityBuilding{})
	if cityId > 0 {
		q = q.Where("city_id = ?", cityId)
	} else if word != "" {
		if id, err := strconv.Atoi(word); err == nil {
			q = q.Where("city_id = ?", id)
		} else {
			var cids []uint
			h.DB.Model(&model.EzfyCity{}).Select("id").
				Where("name LIKE ?", "%"+word+"%").Scan(&cids)
			if len(cids) > 0 {
				q = q.Where("city_id IN ?", cids)
			} else {
				q = q.Where("1 = 0")
			}
		}
	}
	if status >= 0 {
		q = q.Where("status = ?", status)
	}
	var total int64
	q.Count(&total)
	var rows []model.EzfyCityBuilding
	q.Order("id DESC").Offset(offset).Limit(size).Find(&rows)

	type rowOut struct {
		model.EzfyCityBuilding
		CfgName   string `json:"cfg_name"`
		CfgType   int    `json:"cfg_type"`
		MaxLevel  int    `json:"max_level"`
		CityName  string `json:"city_name"`
		OwnerName string `json:"owner_name"`
		HomeNum   string `json:"home_num"`
		StatusTxt string `json:"status_txt"`
	}
	out := []rowOut{}
	for _, b := range rows {
		name, typ, maxLv := "", 0, 0
		var cfg model.EzfyCfgBuilding
		if err := h.DB.First(&cfg, b.BuildingId).Error; err == nil {
			name, typ, maxLv = cfg.Name, cfg.Type, cfg.MaxLevel
		}
		cityName, owner, home := "", "", ""
		var ct model.EzfyCity
		if err := h.DB.First(&ct, b.CityId).Error; err == nil {
			cityName = ct.Name
			owner, home = h.ezfyAdminName(ct.UserID)
		}
		out = append(out, rowOut{EzfyCityBuilding: b, CfgName: name, CfgType: typ, MaxLevel: maxLv,
			CityName: cityName, OwnerName: owner, HomeNum: home,
			StatusTxt: ezfyBuildStatusName(b.Status)})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

func ezfyBuildStatusName(s int) string {
	switch s {
	case 1:
		return "建造中"
	case 2:
		return "升级中"
	}
	return "空闲"
}

// AdminEzfyBuildingCreate 给城池添加建筑
func (h *AdminHandler) AdminEzfyBuildingCreate(c *gin.Context) {
	var in struct {
		CityId     int64 `json:"city_id"`
		BuildingId int   `json:"building_id"`
		Level      int   `json:"level"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.CityId <= 0 || in.BuildingId <= 0 {
		resp.ParamError(c, "请填写城池与建筑")
		return
	}
	var cfg model.EzfyCfgBuilding
	if err := h.DB.First(&cfg, in.BuildingId).Error; err != nil {
		resp.ParamError(c, "建筑配置不存在")
		return
	}
	if in.Level <= 0 {
		in.Level = 1
	}
	if cfg.MaxLevel > 0 && in.Level > cfg.MaxLevel {
		in.Level = cfg.MaxLevel
	}
	var dup int64
	h.DB.Model(&model.EzfyCityBuilding{}).
		Where("city_id = ? AND building_id = ?", in.CityId, in.BuildingId).Count(&dup)
	if dup > 0 {
		resp.ParamError(c, "该城池已有此建筑，请直接修改等级")
		return
	}
	// ★ 用户规则：军事区 / 资源区各有数量上限（默认各 33，见 ezfy_cfg_limit）。
	//   管理端「添加建筑」原来完全绕过这个校验，是玩家「军事区 36 个」超限的来源。
	//   这里按 cfg.Type 分区计数后再拦一道，玩家端与管理端口径一致。
	lim := ezfyLimit()
	var mil, res int64
	var exist []model.EzfyCityBuilding
	h.DB.Where("city_id = ?", in.CityId).Find(&exist)
	for _, b := range exist {
		c := ezfyCfg.building(b.BuildingId)
		if c == nil {
			continue
		}
		switch c.Type {
		case 1:
			res++
		case 2, 3, 4:
			mil++
		}
	}
	switch cfg.Type {
	case 1:
		if res >= int64(lim.ResourceMax) {
			resp.ParamError(c, fmt.Sprintf("资源区建筑数量已达上限(%d/%d)，无法再添加", res, lim.ResourceMax))
			return
		}
	case 2, 3, 4:
		if mil >= int64(lim.MilitaryMax) {
			resp.ParamError(c, fmt.Sprintf("军事区建筑数量已达上限(%d/%d)，无法再添加", mil, lim.MilitaryMax))
			return
		}
	}
	h.DB.Create(&model.EzfyCityBuilding{CityId: in.CityId, BuildingId: in.BuildingId,
		Level: in.Level, Status: 0})
	resp.OK(c, gin.H{"msg": "已添加建筑：" + cfg.Name + " Lv." + strconv.Itoa(in.Level)})
}

// AdminEzfyBuildingUpdate 修改建筑（等级/状态/结束时间）
func (h *AdminHandler) AdminEzfyBuildingUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var b model.EzfyCityBuilding
	if err := h.DB.First(&b, id).Error; err != nil {
		resp.NotFound(c, "建筑不存在")
		return
	}
	var in map[string]interface{}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	updates := map[string]interface{}{}
	for _, k := range []string{"level", "status", "end_time"} {
		if v, ok := in[k]; ok {
			if f, err := strconv.ParseFloat(fmt.Sprint(v), 64); err == nil {
				n := int64(f)
				if n < 0 {
					n = 0
				}
				updates[k] = n
			}
		}
	}
	if len(updates) == 0 {
		resp.ParamError(c, "无可修改字段")
		return
	}
	h.DB.Model(&model.EzfyCityBuilding{}).Where("id = ?", b.ID).Updates(updates)
	// 市政厅等级同步
	if b.BuildingId == 1 {
		if lv, ok := updates["level"]; ok {
			h.DB.Model(&model.EzfyCity{}).Where("id = ?", b.CityId).Update("city_level", lv)
		}
	}
	resp.OK(c, gin.H{"msg": "建筑已更新"})
}

// AdminEzfyBuildingFinish 立即完成建筑建造/升级（单步）
func (h *AdminHandler) AdminEzfyBuildingFinish(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var b model.EzfyCityBuilding
	if err := h.DB.First(&b, id).Error; err != nil {
		resp.NotFound(c, "建筑不存在")
		return
	}
	if b.Status == 0 {
		resp.ParamError(c, "该建筑不在建造/升级中")
		return
	}
	h.DB.Model(&model.EzfyCityBuilding{}).Where("id = ?", b.ID).
		Update("end_time", time.Now().UnixMilli()-1)
	var ct model.EzfyCity
	if err := h.DB.First(&ct, b.CityId).Error; err == nil {
		ez := h.ezfyH()
		ez.checkBuildingDone(&ct)
	}
	var nb model.EzfyCityBuilding
	h.DB.First(&nb, b.ID)
	resp.OK(c, gin.H{"msg": "已立即完成，当前等级 Lv." + strconv.Itoa(nb.Level)})
}

// AdminEzfyBuildingDelete 删除建筑
func (h *AdminHandler) AdminEzfyBuildingDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var b model.EzfyCityBuilding
	if err := h.DB.First(&b, id).Error; err != nil {
		resp.NotFound(c, "建筑不存在")
		return
	}
	h.DB.Delete(&model.EzfyCityBuilding{}, id)
	resp.OK(c, gin.H{"msg": "建筑已删除"})
}

// ============ 3. 建筑队列管理 ============

// AdminEzfyBuildQueue 建造/升级中的建筑队列
func (h *AdminHandler) AdminEzfyBuildQueue(c *gin.Context) {
	page, offset, size := pageOf(c, 15)
	word := strings.TrimSpace(c.Query("word"))
	q := h.DB.Model(&model.EzfyCityBuilding{}).Where("status IN ?", []int{1, 2})
	if word != "" {
		if id, err := strconv.Atoi(word); err == nil {
			q = q.Where("city_id = ?", id)
		} else {
			var cids []uint
			h.DB.Model(&model.EzfyCity{}).Select("id").
				Where("name LIKE ?", "%"+word+"%").Scan(&cids)
			if len(cids) > 0 {
				q = q.Where("city_id IN ?", cids)
			} else {
				q = q.Where("1 = 0")
			}
		}
	}
	var total int64
	q.Count(&total)
	var rows []model.EzfyCityBuilding
	q.Order("end_time ASC").Offset(offset).Limit(size).Find(&rows)
	now := time.Now().UnixMilli()
	type rowOut struct {
		model.EzfyCityBuilding
		CfgName    string `json:"cfg_name"`
		MaxLevel   int    `json:"max_level"`
		CityName   string `json:"city_name"`
		OwnerName  string `json:"owner_name"`
		HomeNum    string `json:"home_num"`
		StatusTxt  string `json:"status_txt"`
		RemainSec  int64  `json:"remain_sec"`
		ChainBuild int    `json:"chain_build"` // 1=一键满级连锁
	}
	out := []rowOut{}
	for _, b := range rows {
		name, maxLv := "", 0
		var cfg model.EzfyCfgBuilding
		if err := h.DB.First(&cfg, b.BuildingId).Error; err == nil {
			name, maxLv = cfg.Name, cfg.MaxLevel
		}
		cityName, owner, home := "", "", ""
		var ct model.EzfyCity
		if err := h.DB.First(&ct, b.CityId).Error; err == nil {
			cityName = ct.Name
			owner, home = h.ezfyAdminName(ct.UserID)
		}
		remain := (b.EndTime - now) / 1000
		if remain < 0 {
			remain = 0
		}
		chain := 0
		if b.StartTime == 0 {
			chain = 1
		}
		out = append(out, rowOut{EzfyCityBuilding: b, CfgName: name, MaxLevel: maxLv,
			CityName: cityName, OwnerName: owner, HomeNum: home,
			StatusTxt: ezfyBuildStatusName(b.Status), RemainSec: remain, ChainBuild: chain})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// AdminEzfyBuildQueueFinish 队列项立即完成
func (h *AdminHandler) AdminEzfyBuildQueueFinish(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var b model.EzfyCityBuilding
	if err := h.DB.First(&b, id).Error; err != nil {
		resp.NotFound(c, "队列项不存在")
		return
	}
	if b.Status == 0 {
		resp.ParamError(c, "该建筑已不在队列中")
		return
	}
	h.DB.Model(&model.EzfyCityBuilding{}).Where("id = ?", b.ID).
		Update("end_time", time.Now().UnixMilli()-1)
	var ct model.EzfyCity
	if err := h.DB.First(&ct, b.CityId).Error; err == nil {
		h.ezfyH().checkBuildingDone(&ct)
	}
	resp.OK(c, gin.H{"msg": "已立即完成"})
}

// AdminEzfyBuildQueueSpeed 队列项加速 N 分钟
func (h *AdminHandler) AdminEzfyBuildQueueSpeed(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var in struct {
		Minutes int64 `json:"minutes"`
	}
	c.ShouldBindJSON(&in)
	if in.Minutes <= 0 {
		in.Minutes = 10
	}
	var b model.EzfyCityBuilding
	if err := h.DB.First(&b, id).Error; err != nil {
		resp.NotFound(c, "队列项不存在")
		return
	}
	if b.Status == 0 {
		resp.ParamError(c, "该建筑已不在队列中")
		return
	}
	end := b.EndTime - in.Minutes*60000
	now := time.Now().UnixMilli()
	if end < now {
		end = now - 1
	}
	h.DB.Model(&model.EzfyCityBuilding{}).Where("id = ?", b.ID).Update("end_time", end)
	var ct model.EzfyCity
	if err := h.DB.First(&ct, b.CityId).Error; err == nil {
		h.ezfyH().checkBuildingDone(&ct)
	}
	resp.OK(c, gin.H{"msg": fmt.Sprintf("已加速 %d 分钟", in.Minutes)})
}

// AdminEzfyBuildQueueCancel 取消队列项（回退为空闲，等级不变）
func (h *AdminHandler) AdminEzfyBuildQueueCancel(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var b model.EzfyCityBuilding
	if err := h.DB.First(&b, id).Error; err != nil {
		resp.NotFound(c, "队列项不存在")
		return
	}
	if b.Status == 0 {
		resp.ParamError(c, "该建筑已不在队列中")
		return
	}
	h.DB.Model(&model.EzfyCityBuilding{}).Where("id = ?", b.ID).
		Updates(map[string]interface{}{"status": 0, "end_time": 0})
	resp.OK(c, gin.H{"msg": "已取消该建筑的建造/升级"})
}

// AdminEzfyBuildQueueFinishAll 一键完成全部队列（每个建筑完成一步）
func (h *AdminHandler) AdminEzfyBuildQueueFinishAll(c *gin.Context) {
	var rows []model.EzfyCityBuilding
	h.DB.Where("status IN ?", []int{1, 2}).Find(&rows)
	if len(rows) == 0 {
		resp.OK(c, gin.H{"msg": "当前没有建造中的建筑"})
		return
	}
	ez := h.ezfyH()
	now := time.Now().UnixMilli() - 1
	cityIds := map[uint]bool{}
	for _, b := range rows {
		h.DB.Model(&model.EzfyCityBuilding{}).Where("id = ?", b.ID).Update("end_time", now)
		cityIds[uint(b.CityId)] = true
	}
	for cid := range cityIds {
		var ct model.EzfyCity
		if err := h.DB.First(&ct, cid).Error; err == nil {
			ez.checkBuildingDone(&ct)
		}
	}
	resp.OK(c, gin.H{"msg": fmt.Sprintf("已立即完成 %d 个建筑", len(rows))})
}

// ============ 4. 兵种管理 ============

// AdminEzfyTroops 玩家部队列表（city_id 精确筛选，word=城名/玩家）
func (h *AdminHandler) AdminEzfyTroops(c *gin.Context) {
	page, offset, size := pageOf(c, 15)
	word := strings.TrimSpace(c.Query("word"))
	cityId := int64(atoiOr(c.Query("city_id"), 0))
	q := h.DB.Model(&model.EzfyCityTroop{}).Where("count > 0")
	if cityId > 0 {
		q = q.Where("city_id = ?", cityId)
	} else if word != "" {
		if id, err := strconv.Atoi(word); err == nil {
			q = q.Where("city_id = ?", id)
		} else {
			var cids []uint
			h.DB.Model(&model.EzfyCity{}).Select("id").
				Where("name LIKE ?", "%"+word+"%").Scan(&cids)
			if len(cids) > 0 {
				q = q.Where("city_id IN ?", cids)
			} else {
				q = q.Where("1 = 0")
			}
		}
	}
	var total int64
	q.Count(&total)
	var rows []model.EzfyCityTroop
	q.Order("id DESC").Offset(offset).Limit(size).Find(&rows)

	type rowOut struct {
		model.EzfyCityTroop
		CfgName   string `json:"cfg_name"`
		TroopType int    `json:"troop_type"`
		Pop       int    `json:"pop"`
		TotalPop  int64  `json:"total_pop"`
		CityName  string `json:"city_name"`
		OwnerName string `json:"owner_name"`
		HomeNum   string `json:"home_num"`
		TypeName  string `json:"type_name"`
	}
	out := []rowOut{}
	for _, t := range rows {
		name, typ, pop := "", 0, 0
		var cfg model.EzfyCfgTroop
		if err := h.DB.First(&cfg, t.TroopId).Error; err == nil {
			name, typ, pop = cfg.Name, cfg.Type, cfg.Pop
		}
		cityName, owner, home := "", "", ""
		var ct model.EzfyCity
		if err := h.DB.First(&ct, t.CityId).Error; err == nil {
			cityName = ct.Name
			owner, home = h.ezfyAdminName(ct.UserID)
		}
		out = append(out, rowOut{EzfyCityTroop: t, CfgName: name, TroopType: typ, Pop: pop,
			TotalPop: t.Count * int64(pop), CityName: cityName, OwnerName: owner, HomeNum: home,
			TypeName: ezfyTroopTypeName(typ)})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

func ezfyTroopTypeName(t int) string {
	switch t {
	case 1:
		return "海军"
	case 2:
		return "陆军"
	case 3:
		return "空军"
	case 4:
		return "城防"
	}
	return "其他"
}

// AdminEzfyTroopsCfg 兵种配置列表（只读参考，含造价与属性）
func (h *AdminHandler) AdminEzfyTroopsCfg(c *gin.Context) {
	var rows []model.EzfyCfgTroop
	h.DB.Order("type, id").Find(&rows)
	type rowOut struct {
		model.EzfyCfgTroop
		TypeName string `json:"type_name"`
	}
	out := []rowOut{}
	for _, r := range rows {
		out = append(out, rowOut{EzfyCfgTroop: r, TypeName: ezfyTroopTypeName(r.Type)})
	}
	resp.OK(c, gin.H{"list": out, "total": len(out)})
}

// AdminEzfyTroopGrant 增/减兵力（count 为负表示扣减）
//
// ★ 数量上限 ezfyAdminTroopMax（1 亿）：不是游戏规则，是**防呆**。
//   线上踩过：管理端输入框里手滑敲成 99999999999，城1 就多了 1000 亿航母，
//   军队耗粮直接变成 4.4 万亿，玩家以为「补给技巧科技坏了」。
//   真正的病因是脏数据，不是公式 —— 所以在这里卡一道，别再让别人踩。
func (h *AdminHandler) AdminEzfyTroopGrant(c *gin.Context) {
	var in struct {
		CityId  int64 `json:"city_id"`
		TroopId int   `json:"troop_id"`
		Count   int64 `json:"count"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.CityId <= 0 || in.TroopId <= 0 || in.Count == 0 {
		resp.ParamError(c, "请填写城池、兵种与数量")
		return
	}
	if in.Count > ezfyAdminTroopMax || in.Count < -ezfyAdminTroopMax {
		resp.ParamError(c, fmt.Sprintf("单次数量需在 ±%s 之间（防误输入）", ezfyFmtBig(ezfyAdminTroopMax)))
		return
	}
	var ct model.EzfyCity
	if err := h.DB.First(&ct, in.CityId).Error; err != nil {
		resp.NotFound(c, "城池不存在")
		return
	}
	var cfg model.EzfyCfgTroop
	if err := h.DB.First(&cfg, in.TroopId).Error; err != nil {
		resp.ParamError(c, "兵种配置不存在")
		return
	}
	ez := h.ezfyH()
	if in.Count > 0 {
		// 增加时要检查**结果**是否越界（已有 9000 万 + 再加 9000 万 = 1.8 亿）
		var cur int64
		h.DB.Model(&model.EzfyCityTroop{}).
			Where("city_id = ? AND troop_id = ?", in.CityId, in.TroopId).
			Pluck("count", &cur)
		if cur+in.Count > ezfyAdminTroopMax {
			resp.ParamError(c, fmt.Sprintf("该城【%s】现有 %s，再加会超过上限 %s",
				cfg.Name, ezfyFmtBig(cur), ezfyFmtBig(ezfyAdminTroopMax)))
			return
		}
		ez.addTroop(uint(in.CityId), in.TroopId, in.Count)
	} else {
		var t model.EzfyCityTroop
		if err := h.DB.Where("city_id = ? AND troop_id = ?", in.CityId, in.TroopId).First(&t).Error; err != nil {
			resp.NotFound(c, "该城池没有此兵种")
			return
		}
		left := t.Count + in.Count
		if left < 0 {
			left = 0
		}
		h.DB.Model(&model.EzfyCityTroop{}).Where("id = ?", t.ID).Update("count", left)
	}
	resp.OK(c, gin.H{"msg": fmt.Sprintf("【%s】%+d", cfg.Name, in.Count)})
}

// ezfyAdminTroopMax 单城单兵种数量上限（防呆用，1 亿足够任何正常玩法）
const ezfyAdminTroopMax int64 = 100000000

// ezfyFmtBig 把大数格式化成带万/亿单位的可读串（管理端提示用，与前端 fmtBig 口径一致）
func ezfyFmtBig(n int64) string {
	neg := n < 0
	if neg {
		n = -n
	}
	var s string
	switch {
	case n >= 100000000:
		s = fmt.Sprintf("%.2f亿", float64(n)/100000000)
	case n >= 10000:
		s = fmt.Sprintf("%.2f万", float64(n)/10000)
	default:
		s = fmt.Sprintf("%d", n)
	}
	if neg {
		return "-" + s
	}
	return s
}

// AdminEzfyTroopUpdate 直接设置某城某兵种数量
func (h *AdminHandler) AdminEzfyTroopUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var t model.EzfyCityTroop
	if err := h.DB.First(&t, id).Error; err != nil {
		resp.NotFound(c, "记录不存在")
		return
	}
	var in struct {
		Count int64 `json:"count"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if in.Count < 0 {
		in.Count = 0
	}
	// ★ 同上：挡住手滑输入的天文数字（线上就是这么被写脏的）
	if in.Count > ezfyAdminTroopMax {
		resp.ParamError(c, fmt.Sprintf("数量不能超过 %s（防误输入）", ezfyFmtBig(ezfyAdminTroopMax)))
		return
	}
	h.DB.Model(&model.EzfyCityTroop{}).Where("id = ?", t.ID).Update("count", in.Count)
	resp.OK(c, gin.H{"msg": "兵力已更新"})
}

// AdminEzfyTroopDelete 清除某城某兵种
func (h *AdminHandler) AdminEzfyTroopDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.EzfyCityTroop{}, id)
	resp.OK(c, gin.H{"msg": "已清除"})
}

// AdminEzfyWounded 伤兵/逃兵列表
func (h *AdminHandler) AdminEzfyWounded(c *gin.Context) {
	page, offset, size := pageOf(c, 15)
	cityId := int64(atoiOr(c.Query("city_id"), 0))
	typ := atoiOr(c.Query("type"), -1)
	q := h.DB.Model(&model.EzfyWounded{})
	if cityId > 0 {
		q = q.Where("city_id = ?", cityId)
	}
	if typ >= 0 {
		q = q.Where("type = ?", typ)
	}
	var total int64
	q.Count(&total)
	var rows []model.EzfyWounded
	q.Order("id DESC").Offset(offset).Limit(size).Find(&rows)
	type rowOut struct {
		model.EzfyWounded
		CfgName   string `json:"cfg_name"`
		CityName  string `json:"city_name"`
		OwnerName string `json:"owner_name"`
		TypeName  string `json:"type_name"`
	}
	out := []rowOut{}
	for _, w := range rows {
		name, cityName, owner := "", "", ""
		var cfg model.EzfyCfgTroop
		if err := h.DB.First(&cfg, w.TroopId).Error; err == nil {
			name = cfg.Name
		}
		var ct model.EzfyCity
		if err := h.DB.First(&ct, w.CityId).Error; err == nil {
			cityName = ct.Name
			owner, _ = h.ezfyAdminName(ct.UserID)
		}
		tn := "伤兵"
		if w.Type == 1 {
			tn = "逃兵"
		}
		out = append(out, rowOut{EzfyWounded: w, CfgName: name, CityName: cityName,
			OwnerName: owner, TypeName: tn})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// AdminEzfyWoundedDelete 清除伤兵/逃兵记录
func (h *AdminHandler) AdminEzfyWoundedDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.EzfyWounded{}, id)
	resp.OK(c, gin.H{"msg": "已清除"})
}

// ============ 5. 队伍征兵（训练队列） ============

// AdminEzfyTrainQueue 征兵/训练队列列表
func (h *AdminHandler) AdminEzfyTrainQueue(c *gin.Context) {
	page, offset, size := pageOf(c, 15)
	word := strings.TrimSpace(c.Query("word"))
	cityId := int64(atoiOr(c.Query("city_id"), 0))
	status := atoiOr(c.Query("status"), -1)
	q := h.DB.Model(&model.EzfyTrainQueue{})
	if cityId > 0 {
		q = q.Where("city_id = ?", cityId)
	} else if word != "" {
		if id, err := strconv.Atoi(word); err == nil {
			q = q.Where("city_id = ?", id)
		} else {
			var cids []uint
			h.DB.Model(&model.EzfyCity{}).Select("id").
				Where("name LIKE ?", "%"+word+"%").Scan(&cids)
			if len(cids) > 0 {
				q = q.Where("city_id IN ?", cids)
			} else {
				q = q.Where("1 = 0")
			}
		}
	}
	if status >= 0 {
		q = q.Where("status = ?", status)
	}
	var total int64
	q.Count(&total)
	var rows []model.EzfyTrainQueue
	q.Order("id DESC").Offset(offset).Limit(size).Find(&rows)
	now := time.Now().UnixMilli()
	statusNames := map[int]string{0: "训练中", 1: "待领取", 2: "已完成"}
	type rowOut struct {
		model.EzfyTrainQueue
		CfgName   string `json:"cfg_name"`
		CityName  string `json:"city_name"`
		OwnerName string `json:"owner_name"`
		HomeNum   string `json:"home_num"`
		StatusTxt string `json:"status_txt"`
		RemainSec int64  `json:"remain_sec"`
		PopNeed   int64  `json:"pop_need"`
	}
	out := []rowOut{}
	for _, t := range rows {
		name, pop := "", 0
		var cfg model.EzfyCfgTroop
		if err := h.DB.First(&cfg, t.TroopId).Error; err == nil {
			name, pop = cfg.Name, cfg.Pop
		}
		cityName, owner, home := "", "", ""
		var ct model.EzfyCity
		if err := h.DB.First(&ct, t.CityId).Error; err == nil {
			cityName = ct.Name
			owner, home = h.ezfyAdminName(ct.UserID)
		}
		remain := (t.EndTime - now) / 1000
		if remain < 0 {
			remain = 0
		}
		out = append(out, rowOut{EzfyTrainQueue: t, CfgName: name, CityName: cityName,
			OwnerName: owner, HomeNum: home, StatusTxt: statusNames[t.Status],
			RemainSec: remain, PopNeed: t.Count * int64(pop)})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// AdminEzfyTrainCreate 新增征兵任务（直接入队，不计费）
func (h *AdminHandler) AdminEzfyTrainCreate(c *gin.Context) {
	var in struct {
		CityId  int64 `json:"city_id"`
		TroopId int   `json:"troop_id"`
		Count   int64 `json:"count"`
		Seconds int64 `json:"seconds"` // 0=用兵种配置耗时
		Instant bool  `json:"instant"` // true=立即完成入库
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.CityId <= 0 || in.TroopId <= 0 || in.Count <= 0 {
		resp.ParamError(c, "请填写城池、兵种与数量")
		return
	}
	var cfg model.EzfyCfgTroop
	if err := h.DB.First(&cfg, in.TroopId).Error; err != nil {
		resp.ParamError(c, "兵种配置不存在")
		return
	}
	var ct model.EzfyCity
	if err := h.DB.First(&ct, in.CityId).Error; err != nil {
		resp.NotFound(c, "城池不存在")
		return
	}
	now := time.Now().UnixMilli()
	sec := in.Seconds
	if sec <= 0 {
		sec = int64(cfg.TrainTime) * in.Count
		if sec <= 0 {
			sec = 60
		}
	}
	ez := h.ezfyH()
	if in.Instant {
		ez.addTroop(uint(in.CityId), in.TroopId, in.Count)
		resp.OK(c, gin.H{"msg": fmt.Sprintf("已立即征召【%s】×%d", cfg.Name, in.Count)})
		return
	}
	h.DB.Create(&model.EzfyTrainQueue{CityId: in.CityId, TroopId: in.TroopId, Count: in.Count,
		Status: 0, StartTime: now, EndTime: now + sec*1000})
	resp.OK(c, gin.H{"msg": fmt.Sprintf("已入队【%s】×%d，耗时 %d 秒", cfg.Name, in.Count, sec)})
}

// AdminEzfyTrainFinish 立即完成征兵（兵力入库）
func (h *AdminHandler) AdminEzfyTrainFinish(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var q model.EzfyTrainQueue
	if err := h.DB.First(&q, id).Error; err != nil {
		resp.NotFound(c, "队列项不存在")
		return
	}
	if q.Status == 2 {
		resp.ParamError(c, "该队列已完成")
		return
	}
	h.DB.Model(&model.EzfyTrainQueue{}).Where("id = ?", q.ID).
		Updates(map[string]interface{}{"end_time": time.Now().UnixMilli() - 1, "status": 0})
	var ct model.EzfyCity
	if err := h.DB.First(&ct, q.CityId).Error; err == nil {
		h.ezfyH().collectTrainQueue(&ct)
	}
	resp.OK(c, gin.H{"msg": "已立即完成，兵力已入库"})
}

// AdminEzfyTrainSpeed 征兵加速 N 分钟
func (h *AdminHandler) AdminEzfyTrainSpeed(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var in struct {
		Minutes int64 `json:"minutes"`
	}
	c.ShouldBindJSON(&in)
	if in.Minutes <= 0 {
		in.Minutes = 10
	}
	var q model.EzfyTrainQueue
	if err := h.DB.First(&q, id).Error; err != nil {
		resp.NotFound(c, "队列项不存在")
		return
	}
	if q.Status == 2 {
		resp.ParamError(c, "该队列已完成")
		return
	}
	end := q.EndTime - in.Minutes*60000
	now := time.Now().UnixMilli()
	if end < now {
		end = now - 1
	}
	h.DB.Model(&model.EzfyTrainQueue{}).Where("id = ?", q.ID).Update("end_time", end)
	var ct model.EzfyCity
	if err := h.DB.First(&ct, q.CityId).Error; err == nil {
		h.ezfyH().collectTrainQueue(&ct)
	}
	resp.OK(c, gin.H{"msg": fmt.Sprintf("已加速 %d 分钟", in.Minutes)})
}

// AdminEzfyTrainDelete 取消征兵队列
func (h *AdminHandler) AdminEzfyTrainDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.EzfyTrainQueue{}, id)
	resp.OK(c, gin.H{"msg": "已取消该征兵队列"})
}

// ============ 6. 军官管理 ============

// AdminEzfyOfficers 军官列表（word=军官名/城名，city_id 筛选，captive 只看俘虏）
func (h *AdminHandler) AdminEzfyOfficers(c *gin.Context) {
	page, offset, size := pageOf(c, 15)
	word := strings.TrimSpace(c.Query("word"))
	cityId := int64(atoiOr(c.Query("city_id"), 0))
	captive := atoiOr(c.Query("captive"), -1)
	q := h.DB.Model(&model.EzfyOfficer{})
	if cityId > 0 {
		q = q.Where("city_id = ?", cityId)
	}
	if captive >= 0 {
		q = q.Where("is_captive = ?", captive)
	}
	if word != "" {
		if id, err := strconv.Atoi(word); err == nil {
			q = q.Where("city_id = ? OR id = ?", id, id)
		} else {
			var cids []uint
			h.DB.Model(&model.EzfyCity{}).Select("id").
				Where("name LIKE ?", "%"+word+"%").Scan(&cids)
			if len(cids) > 0 {
				q = q.Where("name LIKE ? OR city_id IN ?", "%"+word+"%", cids)
			} else {
				q = q.Where("name LIKE ?", "%"+word+"%")
			}
		}
	}
	var total int64
	q.Count(&total)
	var rows []model.EzfyOfficer
	q.Order("id DESC").Offset(offset).Limit(size).Find(&rows)
	posNames := map[int]string{0: "无", 1: "市长", 2: "城守"}
	statusNames := map[int]string{0: "在职", 1: "出征中", 2: "被俘"}
	type rowOut struct {
		model.EzfyOfficer
		CityName   string `json:"city_name"`
		OwnerName  string `json:"owner_name"`
		HomeNum    string `json:"home_num"`
		PosName    string `json:"pos_name"`
		StatusName string `json:"status_name"`
	}
	out := []rowOut{}
	for _, o := range rows {
		cityName, owner, home := "", "", ""
		var ct model.EzfyCity
		if err := h.DB.First(&ct, o.CityId).Error; err == nil {
			cityName = ct.Name
			owner, home = h.ezfyAdminName(ct.UserID)
		}
		out = append(out, rowOut{EzfyOfficer: o, CityName: cityName, OwnerName: owner,
			HomeNum: home, PosName: posNames[o.Position], StatusName: statusNames[o.Status]})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// AdminEzfyOfficerCfg 名将配置列表
func (h *AdminHandler) AdminEzfyOfficerCfg(c *gin.Context) {
	var rows []model.EzfyCfgGeneral
	h.DB.Order("id").Find(&rows)
	resp.OK(c, gin.H{"list": rows, "total": len(rows)})
}

// ezfyGrantGeneral 把名将配置中的某位名将发放给玩家（返回 msg / errMsg，errMsg 非空表示失败）
func (h *AdminHandler) ezfyGrantGeneral(uid uint, generalID int) (string, string) {
	var p model.EzfyProfile
	if err := h.DB.Where("user_id = ?", uid).First(&p).Error; err != nil {
		return "", "玩家不存在"
	}
	ez := h.ezfyH()
	g := ezfyCfg.general(generalID)
	if g == nil {
		return "", "名将不存在"
	}
	city := ez.getOrCreateCity(p.UserID)
	var dup int64
	h.DB.Model(&model.EzfyOfficer{}).Where("city_id = ? AND general_id = ?", city.ID, g.ID).Count(&dup)
	if dup > 0 {
		return "", "该玩家已拥有" + g.Name
	}
	star := g.Star
	if star <= 0 {
		star = 5
	}
	o := model.EzfyOfficer{
		CityId: int64(city.ID), GeneralId: g.ID, Name: g.Name, Star: star,
		Level: 1, Exp: 0,
		Military: g.Military, Logistics: g.Logistics, Learning: g.Learning,
		Loyalty: 100, Skill: "", Equipment: "",
		Position: 0, Status: 0, IsCaptive: 0, UpdateTime: time.Now(),
	}
	h.DB.Create(&o)
	return "已发放名将: " + g.Name, ""
}

// AdminEzfyOfficerGrant 发放名将（军官管理模块入口，参数在 body：user_id + general_id）
func (h *AdminHandler) AdminEzfyOfficerGrant(c *gin.Context) {
	var in struct {
		UserID    uint `json:"user_id"`
		GeneralID int  `json:"general_id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.UserID == 0 || in.GeneralID <= 0 {
		resp.ParamError(c, "请选择玩家与名将")
		return
	}
	msg, errMsg := h.ezfyGrantGeneral(in.UserID, in.GeneralID)
	if errMsg != "" {
		resp.ParamError(c, errMsg)
		return
	}
	resp.OK(c, gin.H{"msg": msg})
}

// AdminEzfyOfficerUpdate 修改军官属性
func (h *AdminHandler) AdminEzfyOfficerUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var o model.EzfyOfficer
	if err := h.DB.First(&o, id).Error; err != nil {
		resp.NotFound(c, "军官不存在")
		return
	}
	var in map[string]interface{}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	updates := map[string]interface{}{}
	if v, ok := in["name"]; ok {
		if s := strings.TrimSpace(fmt.Sprint(v)); s != "" {
			updates["name"] = trimStr(s, 100)
		}
	}
	for _, k := range []string{"level", "star", "exp", "military", "logistics", "learning",
		"loyalty", "position", "status", "is_captive"} {
		if v, ok := in[k]; ok {
			if f, err := strconv.ParseFloat(fmt.Sprint(v), 64); err == nil {
				n := int64(f)
				if n < 0 {
					n = 0
				}
				updates[k] = n
			}
		}
	}
	if len(updates) == 0 {
		resp.ParamError(c, "无可修改字段")
		return
	}
	updates["update_time"] = time.Now()
	h.DB.Model(&model.EzfyOfficer{}).Where("id = ?", o.ID).Updates(updates)
	resp.OK(c, gin.H{"msg": "军官已更新"})
}

// AdminEzfyOfficerDelete 解雇军官
func (h *AdminHandler) AdminEzfyOfficerDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var o model.EzfyOfficer
	if err := h.DB.First(&o, id).Error; err != nil {
		resp.NotFound(c, "军官不存在")
		return
	}
	h.DB.Delete(&model.EzfyOfficer{}, id)
	resp.OK(c, gin.H{"msg": "军官「" + o.Name + "」已解雇"})
}

// AdminEzfyOfficerCaptive 设为俘虏 / 释放
func (h *AdminHandler) AdminEzfyOfficerCaptive(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var in struct {
		Captive bool `json:"captive"`
	}
	c.ShouldBindJSON(&in)
	var o model.EzfyOfficer
	if err := h.DB.First(&o, id).Error; err != nil {
		resp.NotFound(c, "军官不存在")
		return
	}
	updates := map[string]interface{}{"is_captive": 0, "status": 0, "update_time": time.Now()}
	msg := "已释放"
	if in.Captive {
		updates["is_captive"] = 1
		updates["status"] = 2
		updates["position"] = 0
		msg = "已设为俘虏"
	}
	h.DB.Model(&model.EzfyOfficer{}).Where("id = ?", o.ID).Updates(updates)
	resp.OK(c, gin.H{"msg": msg})
}

// ============ 7. 资源管理 ============

// AdminEzfyResources 各城资源总览
func (h *AdminHandler) AdminEzfyResources(c *gin.Context) {
	page, offset, size := pageOf(c, 15)
	word := strings.TrimSpace(c.Query("word"))
	q := h.DB.Model(&model.EzfyCity{})
	if word != "" {
		if uid, err := strconv.Atoi(word); err == nil {
			q = q.Where("user_id = ? OR id = ?", uid, uid)
		} else {
			var ids []uint
			h.DB.Model(&model.EzfyProfile{}).Select("user_id").
				Where("nickname LIKE ?", "%"+word+"%").Scan(&ids)
			if len(ids) > 0 {
				q = q.Where("name LIKE ? OR user_id IN ?", "%"+word+"%", ids)
			} else {
				q = q.Where("name LIKE ?", "%"+word+"%")
			}
		}
	}
	var total int64
	q.Count(&total)
	var rows []model.EzfyCity
	q.Order("id DESC").Offset(offset).Limit(size).Find(&rows)
	type rowOut struct {
		model.EzfyCity
		PlayerName string `json:"player_name"`
		HomeNum    string `json:"home_num"`
		TotalRes   int64  `json:"total_res"`
	}
	out := []rowOut{}
	for _, ct := range rows {
		pn, hn := h.ezfyAdminName(ct.UserID)
		out = append(out, rowOut{EzfyCity: ct, PlayerName: pn, HomeNum: hn,
			TotalRes: ct.Food + ct.Steel + ct.Oil + ct.Rare})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// AdminEzfyResourceSet 直接设置城池资源（不截断，GM 专用）
func (h *AdminHandler) AdminEzfyResourceSet(c *gin.Context) {
	var in struct {
		CityId int64  `json:"city_id"`
		Field  string `json:"field"` // gold/food/steel/oil/rare
		Value  int64  `json:"value"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.CityId <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	allow := map[string]bool{"gold": true, "food": true, "steel": true, "oil": true, "rare": true}
	if !allow[in.Field] {
		resp.ParamError(c, "不支持的资源类型")
		return
	}
	if in.Value < 0 {
		in.Value = 0
	}
	var ct model.EzfyCity
	if err := h.DB.First(&ct, in.CityId).Error; err != nil {
		resp.NotFound(c, "城池不存在")
		return
	}
	h.DB.Model(&model.EzfyCity{}).Where("id = ?", ct.ID).Update(in.Field, in.Value)
	resp.OK(c, gin.H{"msg": fmt.Sprintf("已将城池「%s」的%s设为 %d", ct.Name, ezfyResNameOf(in.Field), in.Value)})
}

func ezfyResNameOf(field string) string {
	switch field {
	case "gold":
		return "黄金"
	case "food":
		return "粮食"
	case "steel":
		return "钢铁"
	case "oil":
		return "石油"
	case "rare":
		return "稀矿"
	}
	return field
}

// AdminEzfyResourceGrant 批量发放资源（按 user_id 发给其主城，**不按仓储上限截断**）
func (h *AdminHandler) AdminEzfyResourceGrant(c *gin.Context) {
	var in struct {
		UserIds []uint `json:"user_ids"`
		Gold    int64  `json:"gold"`
		Food    int64  `json:"food"`
		Steel   int64  `json:"steel"`
		Oil     int64  `json:"oil"`
		Rare    int64  `json:"rare"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || len(in.UserIds) == 0 {
		resp.ParamError(c, "请选择要发放的玩家")
		return
	}
	if in.Gold == 0 && in.Food == 0 && in.Steel == 0 && in.Oil == 0 && in.Rare == 0 {
		resp.ParamError(c, "请填写发放数量")
		return
	}
	ez := h.ezfyH()
	ok := 0
	for _, uid := range in.UserIds {
		var p model.EzfyProfile
		if err := h.DB.Where("user_id = ?", uid).First(&p).Error; err != nil {
			continue
		}
		ez.giveResourcesNoCap(uid, in.Food, in.Steel, in.Oil, in.Rare, in.Gold)
		h.DB.Create(&model.EzfyNotice{UserId: uid, Title: "管理员发放",
			Content: "管理员为你发放了资源，请查收。"})
		ok++
	}
	resp.OK(c, gin.H{"msg": fmt.Sprintf("已向 %d 名玩家发放资源（不受仓储上限限制）", ok)})
}

// AdminEzfyResourceSummary 全服资源统计
func (h *AdminHandler) AdminEzfyResourceSummary(c *gin.Context) {
	type agg struct{ Sum int64 }
	var gold, food, steel, oil, rare agg
	h.DB.Model(&model.EzfyCity{}).Select("COALESCE(SUM(gold),0) as sum").Scan(&gold)
	h.DB.Model(&model.EzfyCity{}).Select("COALESCE(SUM(food),0) as sum").Scan(&food)
	h.DB.Model(&model.EzfyCity{}).Select("COALESCE(SUM(steel),0) as sum").Scan(&steel)
	h.DB.Model(&model.EzfyCity{}).Select("COALESCE(SUM(oil),0) as sum").Scan(&oil)
	h.DB.Model(&model.EzfyCity{}).Select("COALESCE(SUM(rare),0) as sum").Scan(&rare)
	var cities int64
	h.DB.Model(&model.EzfyCity{}).Count(&cities)
	// 资源 TOP10
	type topRow struct {
		ID     uint   `json:"id"`
		Name   string `json:"name"`
		UserID uint   `json:"user_id"`
		Total  int64  `json:"total"`
		Gold   int64  `json:"gold"`
	}
	tops := []topRow{}
	h.DB.Model(&model.EzfyCity{}).
		Select("id, name, user_id, (gold+food+steel+oil+rare) as total, gold").
		Order("total DESC").Limit(10).Scan(&tops)
	resp.OK(c, gin.H{"gold": gold.Sum, "food": food.Sum, "steel": steel.Sum,
		"oil": oil.Sum, "rare": rare.Sum, "cities": cities, "tops": tops})
}

// ============ 8. 科技管理 ============

// AdminEzfyTechs 玩家科技列表
func (h *AdminHandler) AdminEzfyTechs(c *gin.Context) {
	page, offset, size := pageOf(c, 15)
	word := strings.TrimSpace(c.Query("word"))
	cityId := int64(atoiOr(c.Query("city_id"), 0))
	q := h.DB.Model(&model.EzfyCityTech{})
	if cityId > 0 {
		q = q.Where("city_id = ?", cityId)
	} else if word != "" {
		if id, err := strconv.Atoi(word); err == nil {
			q = q.Where("city_id = ?", id)
		} else {
			var cids []uint
			h.DB.Model(&model.EzfyCity{}).Select("id").
				Where("name LIKE ?", "%"+word+"%").Scan(&cids)
			if len(cids) > 0 {
				q = q.Where("city_id IN ?", cids)
			} else {
				q = q.Where("1 = 0")
			}
		}
	}
	var total int64
	q.Count(&total)
	var rows []model.EzfyCityTech
	q.Order("id DESC").Offset(offset).Limit(size).Find(&rows)
	now := time.Now().UnixMilli()
	type rowOut struct {
		model.EzfyCityTech
		CfgName   string `json:"cfg_name"`
		MaxLevel  int    `json:"max_level"`
		CityName  string `json:"city_name"`
		OwnerName string `json:"owner_name"`
		HomeNum   string `json:"home_num"`
		StatusTxt string `json:"status_txt"`
		RemainSec int64  `json:"remain_sec"`
	}
	out := []rowOut{}
	for _, t := range rows {
		name, maxLv := "", 0
		var cfg model.EzfyCfgTech
		if err := h.DB.First(&cfg, t.TechId).Error; err == nil {
			name, maxLv = cfg.Name, cfg.MaxLevel
		}
		cityName, owner, home := "", "", ""
		var ct model.EzfyCity
		if err := h.DB.First(&ct, t.CityId).Error; err == nil {
			cityName = ct.Name
			owner, home = h.ezfyAdminName(ct.UserID)
		}
		remain := (t.EndTime - now) / 1000
		if remain < 0 {
			remain = 0
		}
		st := "已掌握"
		if t.Status == 1 {
			st = "研究中"
		}
		out = append(out, rowOut{EzfyCityTech: t, CfgName: name, MaxLevel: maxLv,
			CityName: cityName, OwnerName: owner, HomeNum: home, StatusTxt: st, RemainSec: remain})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// AdminEzfyTechsCfg 科技配置列表
func (h *AdminHandler) AdminEzfyTechsCfg(c *gin.Context) {
	var rows []model.EzfyCfgTech
	h.DB.Order("type, id").Find(&rows)
	resp.OK(c, gin.H{"list": rows, "total": len(rows)})
}

// AdminEzfyTechSet 设置城池科技等级（存在则改，不存在则建）
func (h *AdminHandler) AdminEzfyTechSet(c *gin.Context) {
	var in struct {
		CityId int64 `json:"city_id"`
		TechId int   `json:"tech_id"`
		Level  int   `json:"level"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.CityId <= 0 || in.TechId <= 0 {
		resp.ParamError(c, "请填写城池与科技")
		return
	}
	if in.Level < 0 {
		in.Level = 0
	}
	var cfg model.EzfyCfgTech
	if err := h.DB.First(&cfg, in.TechId).Error; err != nil {
		resp.ParamError(c, "科技配置不存在")
		return
	}
	if cfg.MaxLevel > 0 && in.Level > cfg.MaxLevel {
		in.Level = cfg.MaxLevel
	}
	var ct model.EzfyCity
	if err := h.DB.First(&ct, in.CityId).Error; err != nil {
		resp.NotFound(c, "城池不存在")
		return
	}
	var t model.EzfyCityTech
	if err := h.DB.Where("city_id = ? AND tech_id = ?", in.CityId, in.TechId).First(&t).Error; err != nil {
		h.DB.Create(&model.EzfyCityTech{CityId: in.CityId, TechId: in.TechId, Level: in.Level, Status: 0})
	} else {
		h.DB.Model(&model.EzfyCityTech{}).Where("id = ?", t.ID).
			Updates(map[string]interface{}{"level": in.Level, "status": 0, "end_time": 0})
	}
	resp.OK(c, gin.H{"msg": fmt.Sprintf("【%s】已设为 Lv.%d", cfg.Name, in.Level)})
}

// AdminEzfyTechUpdate 修改科技记录（等级/状态）
func (h *AdminHandler) AdminEzfyTechUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var t model.EzfyCityTech
	if err := h.DB.First(&t, id).Error; err != nil {
		resp.NotFound(c, "记录不存在")
		return
	}
	var in map[string]interface{}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	updates := map[string]interface{}{}
	for _, k := range []string{"level", "status", "end_time"} {
		if v, ok := in[k]; ok {
			if f, err := strconv.ParseFloat(fmt.Sprint(v), 64); err == nil {
				n := int64(f)
				if n < 0 {
					n = 0
				}
				updates[k] = n
			}
		}
	}
	if len(updates) == 0 {
		resp.ParamError(c, "无可修改字段")
		return
	}
	h.DB.Model(&model.EzfyCityTech{}).Where("id = ?", t.ID).Updates(updates)
	resp.OK(c, gin.H{"msg": "科技记录已更新"})
}

// AdminEzfyTechFinish 立即完成科技研究
func (h *AdminHandler) AdminEzfyTechFinish(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var t model.EzfyCityTech
	if err := h.DB.First(&t, id).Error; err != nil {
		resp.NotFound(c, "记录不存在")
		return
	}
	if t.Status != 1 {
		resp.ParamError(c, "该科技不在研究中")
		return
	}
	h.DB.Model(&model.EzfyCityTech{}).Where("id = ?", t.ID).
		Update("end_time", time.Now().UnixMilli()-1)
	var ct model.EzfyCity
	if err := h.DB.First(&ct, t.CityId).Error; err == nil {
		h.ezfyH().checkTechDone(&ct)
	}
	resp.OK(c, gin.H{"msg": "已立即完成研究"})
}

// AdminEzfyTechDelete 删除科技记录（重置该科技）
func (h *AdminHandler) AdminEzfyTechDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.EzfyCityTech{}, id)
	resp.OK(c, gin.H{"msg": "已重置该科技"})
}

// ============ 9. 地图管理 ============

// AdminEzfyMapCities 城市坐标分布（按坐标排序，便于地图查看）
func (h *AdminHandler) AdminEzfyMapCities(c *gin.Context) {
	page, offset, size := pageOf(c, 20)
	word := strings.TrimSpace(c.Query("word"))
	q := h.DB.Model(&model.EzfyCity{})
	if word != "" {
		if uid, err := strconv.Atoi(word); err == nil {
			q = q.Where("user_id = ? OR id = ?", uid, uid)
		} else {
			q = q.Where("name LIKE ?", "%"+word+"%")
		}
	}
	var total int64
	q.Count(&total)
	var rows []model.EzfyCity
	q.Order("x, y").Offset(offset).Limit(size).Find(&rows)
	type rowOut struct {
		ID          uint   `json:"id"`
		Name        string `json:"name"`
		UserID      uint   `json:"user_id"`
		X           int    `json:"x"`
		Y           int    `json:"y"`
		CityLevel   int    `json:"city_level"`
		PlayerName  string `json:"player_name"`
		HomeNum     string `json:"home_num"`
		TerrainName string `json:"terrain_name"`
		Continent   int    `json:"continent"`
	}
	out := []rowOut{}
	for _, ct := range rows {
		pn, hn := h.ezfyAdminName(ct.UserID)
		out = append(out, rowOut{ID: ct.ID, Name: ct.Name, UserID: ct.UserID, X: ct.X, Y: ct.Y,
			CityLevel: ct.CityLevel, PlayerName: pn, HomeNum: hn,
			TerrainName: ezfyTerrainNameEx(ct.X, ct.Y),
			Continent:   ezfyContinentOf(ct.X, ct.Y)})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// AdminEzfyMapWildlands 已占野地列表
func (h *AdminHandler) AdminEzfyMapWildlands(c *gin.Context) {
	page, offset, size := pageOf(c, 20)
	word := strings.TrimSpace(c.Query("word"))
	q := h.DB.Model(&model.EzfyWildland{})
	if word != "" {
		if id, err := strconv.Atoi(word); err == nil {
			q = q.Where("city_id = ?", id)
		} else {
			var cids []uint
			h.DB.Model(&model.EzfyCity{}).Select("id").
				Where("name LIKE ?", "%"+word+"%").Scan(&cids)
			if len(cids) > 0 {
				q = q.Where("city_id IN ?", cids)
			} else {
				q = q.Where("1 = 0")
			}
		}
	}
	var total int64
	q.Count(&total)
	var rows []model.EzfyWildland
	q.Order("id DESC").Offset(offset).Limit(size).Find(&rows)
	type rowOut struct {
		model.EzfyWildland
		CityName    string `json:"city_name"`
		OwnerName   string `json:"owner_name"`
		HomeNum     string `json:"home_num"`
		TerrainName string `json:"terrain_name"`
		StatusTxt   string `json:"status_txt"`
	}
	out := []rowOut{}
	for _, w := range rows {
		cityName, owner, home := "", "", ""
		var ct model.EzfyCity
		if err := h.DB.First(&ct, w.CityId).Error; err == nil {
			cityName = ct.Name
			owner, home = h.ezfyAdminName(ct.UserID)
		}
		st := "空闲"
		if w.Status == 1 {
			st = "采集中"
		}
		out = append(out, rowOut{EzfyWildland: w, CityName: cityName, OwnerName: owner,
			HomeNum: home, TerrainName: ezfyTerrainNameEx(w.X, w.Y), StatusTxt: st})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// AdminEzfyMapOccupy 占领记录
func (h *AdminHandler) AdminEzfyMapOccupy(c *gin.Context) {
	page, offset, size := pageOf(c, 15)
	status := atoiOr(c.Query("status"), -1)
	q := h.DB.Model(&model.EzfyOccupy{})
	if status >= 0 {
		q = q.Where("status = ?", status)
	}
	var total int64
	q.Count(&total)
	var rows []model.EzfyOccupy
	q.Order("id DESC").Offset(offset).Limit(size).Find(&rows)
	type rowOut struct {
		model.EzfyOccupy
		AtkName  string `json:"atk_name"`
		DefName  string `json:"def_name"`
		StatusTx string `json:"status_txt"`
	}
	out := []rowOut{}
	for _, o := range rows {
		an, _ := h.ezfyAdminName(o.AtkUserId)
		dn, _ := h.ezfyAdminName(o.DefUserId)
		st := "占领中"
		if o.Status == 2 {
			st = "已归还"
		}
		out = append(out, rowOut{EzfyOccupy: o, AtkName: an, DefName: dn, StatusTx: st})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// AdminEzfyMapOccupyRelease 解除占领（归还城市）
func (h *AdminHandler) AdminEzfyMapOccupyRelease(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var o model.EzfyOccupy
	if err := h.DB.First(&o, id).Error; err != nil {
		resp.NotFound(c, "占领记录不存在")
		return
	}
	h.DB.Model(&model.EzfyOccupy{}).Where("id = ?", o.ID).Update("status", 2)
	resp.OK(c, gin.H{"msg": "已解除占领，「" + o.CityName + "」已归还"})
}

// AdminEzfyMapOccupyDelete 删除占领记录
func (h *AdminHandler) AdminEzfyMapOccupyDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.EzfyOccupy{}, id)
	resp.OK(c, gin.H{"msg": "已删除"})
}

// AdminEzfyMapAreas 地图区域记录（ezfy_map_area，落库的野地/寇城快照）
func (h *AdminHandler) AdminEzfyMapAreas(c *gin.Context) {
	page, offset, size := pageOf(c, 20)
	areaType := atoiOr(c.Query("area_type"), -1)
	q := h.DB.Model(&model.EzfyMapArea{})
	if areaType >= 0 {
		q = q.Where("area_type = ?", areaType)
	}
	var total int64
	q.Count(&total)
	var rows []model.EzfyMapArea
	q.Order("id DESC").Offset(offset).Limit(size).Find(&rows)
	type rowOut struct {
		model.EzfyMapArea
		TerrainName string `json:"terrain_name"`
		TypeName    string `json:"type_name"`
	}
	out := []rowOut{}
	typeNames := map[int]string{0: "空地", 1: "野地(已占)", 2: "寇城", 3: "玩家城", 4: "资源田"}
	for _, a := range rows {
		out = append(out, rowOut{EzfyMapArea: a,
			TerrainName: ezfyTerrainNameEx(a.X, a.Y),
			TypeName:    typeNames[a.AreaType]})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// AdminEzfyMapAreaDelete 删除地图区域记录
func (h *AdminHandler) AdminEzfyMapAreaDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.EzfyMapArea{}, id)
	resp.OK(c, gin.H{"msg": "已删除"})
}

// AdminEzfyMapStars 坐标收藏列表
func (h *AdminHandler) AdminEzfyMapStars(c *gin.Context) {
	page, offset, size := pageOf(c, 20)
	q := h.DB.Model(&model.EzfyMapStar{})
	var total int64
	q.Count(&total)
	var rows []model.EzfyMapStar
	q.Order("id DESC").Offset(offset).Limit(size).Find(&rows)
	type rowOut struct {
		model.EzfyMapStar
		OwnerName   string `json:"owner_name"`
		HomeNum     string `json:"home_num"`
		TerrainName string `json:"terrain_name"`
	}
	out := []rowOut{}
	for _, s := range rows {
		on, hn := h.ezfyAdminName(s.UserID)
		out = append(out, rowOut{EzfyMapStar: s, OwnerName: on, HomeNum: hn,
			TerrainName: ezfyTerrainNameEx(s.X, s.Y)})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// AdminEzfyMapStarsDelete 删除坐标收藏
func (h *AdminHandler) AdminEzfyMapStarsDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.EzfyMapStar{}, id)
	resp.OK(c, gin.H{"msg": "已删除"})
}

// AdminEzfyMapLookup 坐标查询（给定 x,y 返回地形/大陆/野地等级/归属）
func (h *AdminHandler) AdminEzfyMapLookup(c *gin.Context) {
	x := atoiOr(c.Query("x"), -1)
	y := atoiOr(c.Query("y"), -1)
	if x < 0 || y < 0 {
		resp.ParamError(c, "请提供坐标 x / y")
		return
	}
	terrain := ezfyTerrainEx(x, y)
	out := gin.H{
		"x": x, "y": y,
		"terrain":      terrain,
		"terrain_name": ezfyTerrainName(terrain),
		"continent":    ezfyContinentOf(x, y),
		"wild_level":   ezfyWildlandLevel(x, y),
	}
	var ct model.EzfyCity
	if err := h.DB.Where("x = ? AND y = ?", x, y).First(&ct).Error; err == nil {
		pn, hn := h.ezfyAdminName(ct.UserID)
		out["city"] = gin.H{"id": ct.ID, "name": ct.Name, "user_id": ct.UserID,
			"player_name": pn, "home_num": hn, "city_level": ct.CityLevel}
	}
	var area model.EzfyMapArea
	if err := h.DB.Where("x = ? AND y = ?", x, y).First(&area).Error; err == nil {
		out["area"] = area
	}
	resp.OK(c, out)
}

// ============ 10. 军团管理 ============

// AdminEzfyCorpsMembers 军团成员列表
func (h *AdminHandler) AdminEzfyCorpsMembers(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var cp model.EzfyCorps
	if err := h.DB.First(&cp, id).Error; err != nil {
		resp.NotFound(c, "军团不存在")
		return
	}
	var rows []model.EzfyCorpsMember
	h.DB.Where("corps_id = ?", cp.ID).Order("is_leader DESC, id").Find(&rows)
	type rowOut struct {
		model.EzfyCorpsMember
		PlayerName string `json:"player_name"`
		HomeNum    string `json:"home_num"`
		Prestige   int    `json:"prestige"`
		RankName   string `json:"rank_name"`
		RoleName   string `json:"role_name"`
	}
	out := []rowOut{}
	for _, m := range rows {
		pn, hn := h.ezfyAdminName(m.UserId)
		prestige := 0
		var p model.EzfyProfile
		if err := h.DB.Where("user_id = ?", m.UserId).First(&p).Error; err == nil {
			prestige = p.Prestige
		}
		role := "成员"
		if m.IsLeader == 1 {
			role = "军团长"
		}
		out = append(out, rowOut{EzfyCorpsMember: m, PlayerName: pn, HomeNum: hn,
			Prestige: prestige, RankName: ezfyRankName(prestige), RoleName: role})
	}
	resp.OK(c, gin.H{"corps": cp, "list": out, "total": len(out)})
}

// AdminEzfyCorpsUpdate 修改军团（名称/公告/团长）
func (h *AdminHandler) AdminEzfyCorpsUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var cp model.EzfyCorps
	if err := h.DB.First(&cp, id).Error; err != nil {
		resp.NotFound(c, "军团不存在")
		return
	}
	var in struct {
		Name         *string `json:"name"`
		Notice       *string `json:"notice"`
		LeaderUserId *uint   `json:"leader_user_id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	updates := map[string]interface{}{}
	if in.Name != nil {
		if s := strings.TrimSpace(*in.Name); s != "" {
			var dup int64
			h.DB.Model(&model.EzfyCorps{}).Where("name = ? AND id <> ?", s, cp.ID).Count(&dup)
			if dup > 0 {
				resp.ParamError(c, "军团名已被占用")
				return
			}
			updates["name"] = trimStr(s, 20)
		}
	}
	if in.Notice != nil {
		updates["notice"] = trimStr(strings.TrimSpace(*in.Notice), 200)
	}
	if in.LeaderUserId != nil && *in.LeaderUserId > 0 {
		var m model.EzfyCorpsMember
		if err := h.DB.Where("corps_id = ? AND user_id = ?", cp.ID, *in.LeaderUserId).First(&m).Error; err != nil {
			resp.ParamError(c, "该玩家不是本军团成员")
			return
		}
		h.DB.Model(&model.EzfyCorpsMember{}).Where("corps_id = ?", cp.ID).Update("is_leader", 0)
		h.DB.Model(&model.EzfyCorpsMember{}).Where("id = ?", m.ID).
			Updates(map[string]interface{}{"is_leader": 1, "title": "军团长"})
		updates["leader_user_id"] = *in.LeaderUserId
	}
	if len(updates) == 0 {
		resp.ParamError(c, "无可修改字段")
		return
	}
	h.DB.Model(&model.EzfyCorps{}).Where("id = ?", cp.ID).Updates(updates)
	resp.OK(c, gin.H{"msg": "军团已更新"})
}

// AdminEzfyCorpsKick 移出军团成员
func (h *AdminHandler) AdminEzfyCorpsKick(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	uid, _ := strconv.Atoi(c.Param("uid"))
	var cp model.EzfyCorps
	if err := h.DB.First(&cp, id).Error; err != nil {
		resp.NotFound(c, "军团不存在")
		return
	}
	if int(cp.LeaderUserId) == uid {
		resp.ParamError(c, "军团长不能被移出，请先转让团长")
		return
	}
	var m model.EzfyCorpsMember
	if err := h.DB.Where("corps_id = ? AND user_id = ?", cp.ID, uid).First(&m).Error; err != nil {
		resp.NotFound(c, "该玩家不在本军团")
		return
	}
	h.DB.Delete(&model.EzfyCorpsMember{}, m.ID)
	n := cp.MemberCount - 1
	if n < 0 {
		n = 0
	}
	h.DB.Model(&model.EzfyCorps{}).Where("id = ?", cp.ID).Update("member_count", n)
	resp.OK(c, gin.H{"msg": "已移出军团"})
}

// AdminEzfyCorpsChats 军团聊天记录
func (h *AdminHandler) AdminEzfyCorpsChats(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	page, offset, size := pageOf(c, 20)
	q := h.DB.Model(&model.EzfyCorpsChat{}).Where("corps_id = ?", id)
	var total int64
	q.Count(&total)
	var rows []model.EzfyCorpsChat
	q.Order("id DESC").Offset(offset).Limit(size).Find(&rows)
	resp.OK(c, gin.H{"list": rows, "total": total, "page": page, "size": size})
}

// AdminEzfyCorpsChatDelete 删除军团聊天
func (h *AdminHandler) AdminEzfyCorpsChatDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.EzfyCorpsChat{}, id)
	resp.OK(c, gin.H{"msg": "已删除"})
}

// ============ 11. 私聊管理 ============

// AdminEzfyPrivchats 玩家私聊记录（word 匹配内容或双方昵称）
func (h *AdminHandler) AdminEzfyPrivchats(c *gin.Context) {
	page, offset, size := pageOf(c, 20)
	word := strings.TrimSpace(c.Query("word"))
	sender := atoiOr(c.Query("sender"), 0)
	receiver := atoiOr(c.Query("receiver"), 0)
	isRead := atoiOr(c.Query("is_read"), -1)
	q := h.DB.Model(&model.PrivateMessage{})
	if sender > 0 {
		q = q.Where("sender_id = ?", sender)
	}
	if receiver > 0 {
		q = q.Where("receiver_id = ?", receiver)
	}
	if isRead >= 0 {
		q = q.Where("is_read = ?", isRead)
	}
	if word != "" {
		if uid, err := strconv.Atoi(word); err == nil {
			q = q.Where("sender_id = ? OR receiver_id = ?", uid, uid)
		} else {
			var ids []uint
			h.DB.Model(&model.User{}).Select("id").
				Where("nickname LIKE ? OR username LIKE ?", "%"+word+"%", "%"+word+"%").Scan(&ids)
			if len(ids) > 0 {
				q = q.Where("content LIKE ? OR sender_id IN ? OR receiver_id IN ?", "%"+word+"%", ids, ids)
			} else {
				q = q.Where("content LIKE ?", "%"+word+"%")
			}
		}
	}
	var total int64
	q.Count(&total)
	var rows []model.PrivateMessage
	q.Order("id DESC").Offset(offset).Limit(size).Find(&rows)

	names := map[uint]string{}
	nums := map[uint]string{}
	nameOf := func(uid uint) (string, string) {
		if n, ok := names[uid]; ok {
			return n, nums[uid]
		}
		n, num := h.ezfyAdminName(uid)
		if n == "" {
			var u model.User
			if err := h.DB.First(&u, uid).Error; err == nil {
				n, num = u.Nickname, u.Username
			}
		}
		names[uid] = n
		nums[uid] = num
		return n, num
	}
	type rowOut struct {
		model.PrivateMessage
		SenderName   string `json:"sender_name"`
		SenderNum    string `json:"sender_num"`
		ReceiverName string `json:"receiver_name"`
		ReceiverNum  string `json:"receiver_num"`
	}
	out := []rowOut{}
	for _, m := range rows {
		sn, snum := nameOf(m.SenderID)
		rn, rnum := nameOf(m.ReceiverID)
		out = append(out, rowOut{PrivateMessage: m, SenderName: sn, SenderNum: snum,
			ReceiverName: rn, ReceiverNum: rnum})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// AdminEzfyPrivchatDelete 删除单条私聊
func (h *AdminHandler) AdminEzfyPrivchatDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.PrivateMessage{}, id)
	resp.OK(c, gin.H{"msg": "已删除"})
}

// AdminEzfyPrivchatClear 清空某玩家的私聊（dir=from 发出的 / to 收到的 / all 全部）
func (h *AdminHandler) AdminEzfyPrivchatClear(c *gin.Context) {
	var in struct {
		UserId uint   `json:"user_id"`
		Dir    string `json:"dir"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.UserId <= 0 {
		resp.ParamError(c, "请指定玩家")
		return
	}
	var n int64
	switch in.Dir {
	case "from":
		h.DB.Model(&model.PrivateMessage{}).Where("sender_id = ?", in.UserId).Count(&n)
		h.DB.Where("sender_id = ?", in.UserId).Delete(&model.PrivateMessage{})
	case "to":
		h.DB.Model(&model.PrivateMessage{}).Where("receiver_id = ?", in.UserId).Count(&n)
		h.DB.Where("receiver_id = ?", in.UserId).Delete(&model.PrivateMessage{})
	default:
		h.DB.Model(&model.PrivateMessage{}).
			Where("sender_id = ? OR receiver_id = ?", in.UserId, in.UserId).Count(&n)
		h.DB.Where("sender_id = ?", in.UserId).Delete(&model.PrivateMessage{})
		h.DB.Where("receiver_id = ?", in.UserId).Delete(&model.PrivateMessage{})
	}
	resp.OK(c, gin.H{"msg": fmt.Sprintf("已清空 %d 条私聊记录", n)})
}
