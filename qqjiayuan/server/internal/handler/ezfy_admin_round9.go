package handler

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 二战风云 管理端（第九轮新增）
//
//	1. 建筑数量上限配置（军事区/资源区各 33，管理端可维护）
//	2. 钻石充值（钻石只能管理端充值，玩家端只读余额）
//	3. 二战聊天敏感词（独立维护页，与社区「黑名单榜」分开）

// ============ 1. 建筑数量上限配置 ============

// AdminEzfyBuildLimitGet GET /admin/ezfy-build-limit
func (h *AdminHandler) AdminEzfyBuildLimitGet(c *gin.Context) {
	lim := model.EzfyCfgLimit{ID: 1, MilitaryMax: 33, ResourceMax: 33, HouseMax: 10, FactoryMax: 0}
	if err := h.DB.First(&lim, 1).Error; err != nil {
		h.DB.Create(&lim)
	}
	resp.OK(c, lim)
}

// AdminEzfyBuildLimitUpdate PUT /admin/ezfy-build-limit
//
// 军事区(type 2/3/4) 与 资源区(type 1) 的数量上限**分开**维护，默认各 33。
// factory_max = 0 表示军工厂不限数量（默认，符合用户规则）。
func (h *AdminHandler) AdminEzfyBuildLimitUpdate(c *gin.Context) {
	var in struct {
		MilitaryMax     *int `json:"military_max"`
		ResourceMax     *int `json:"resource_max"`
		HouseMax        *int `json:"house_max"`
		FactoryMax      *int `json:"factory_max"`
		NoticeHomeCount *int `json:"notice_home_count"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	lim := model.EzfyCfgLimit{ID: 1, MilitaryMax: 33, ResourceMax: 33, HouseMax: 10, FactoryMax: 0}
	h.DB.First(&lim, 1)
	check := func(v *int, name string) (int, bool) {
		if v == nil {
			return 0, true
		}
		if *v < 0 || *v > 999 {
			resp.ParamError(c, name+" 需要在 0~999 之间")
			return 0, false
		}
		return *v, true
	}
	if v, ok := check(in.MilitaryMax, "军事区上限"); !ok {
		return
	} else if in.MilitaryMax != nil {
		lim.MilitaryMax = v
	}
	if v, ok := check(in.ResourceMax, "资源区上限"); !ok {
		return
	} else if in.ResourceMax != nil {
		lim.ResourceMax = v
	}
	if v, ok := check(in.HouseMax, "民居上限"); !ok {
		return
	} else if in.HouseMax != nil {
		lim.HouseMax = v
	}
	if v, ok := check(in.FactoryMax, "军工厂上限"); !ok {
		return
	} else if in.FactoryMax != nil {
		lim.FactoryMax = v
	}
	if v, ok := check(in.NoticeHomeCount, "首页公告条数"); !ok {
		return
	} else if in.NoticeHomeCount != nil {
		lim.NoticeHomeCount = v
	}
	if lim.MilitaryMax <= 0 {
		lim.MilitaryMax = 33
	}
	if lim.ResourceMax <= 0 {
		lim.ResourceMax = 33
	}
	if lim.HouseMax <= 0 {
		lim.HouseMax = 10
	}
	// ★ 首页公告条数允许 0（= 首页不展示公告），但不允许负数；未配过时默认 1
	if lim.NoticeHomeCount < 0 {
		lim.NoticeHomeCount = 1
	}
	lim.ID = 1
	if err := h.DB.Save(&lim).Error; err != nil {
		resp.ParamError(c, "保存失败："+err.Error())
		return
	}
	// ★ 写完必须重载配置缓存，否则玩家端要重启才生效
	h.ezfyH().cfgsReload()
	resp.OK(c, gin.H{"msg": "建筑数量上限已保存并立即生效", "limit": lim})
}

// ============ 2. 钻石充值 ============

// AdminEzfyDiamondRecharge POST /admin/ezfy-players/:id/diamond
//
// 用户规则：钻石**只能管理端充值**（玩家端只读余额）。
// mode = add(默认，可负数扣减) | set(直接设为某值)
func (h *AdminHandler) AdminEzfyDiamondRecharge(c *gin.Context) {
	uid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || uid == 0 {
		resp.ParamError(c, "玩家ID错误")
		return
	}
	var in struct {
		Amount int64  `json:"amount"`
		Mode   string `json:"mode"`
		Remark string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	ez := h.ezfyH()
	prof := ez.ensureProfile(uint(uid))
	if in.Mode == "set" {
		if in.Amount < 0 {
			resp.ParamError(c, "钻石不能为负数")
			return
		}
		prof.Diamond = in.Amount
	} else {
		if in.Amount == 0 {
			resp.ParamError(c, "请填写充值数量")
			return
		}
		prof.Diamond = prof.Diamond + in.Amount
		if prof.Diamond < 0 {
			prof.Diamond = 0
		}
	}
	if err := h.DB.Model(&model.EzfyProfile{}).Where("id = ?", prof.ID).
		Update("diamond", prof.Diamond).Error; err != nil {
		resp.ParamError(c, "充值失败："+err.Error())
		return
	}
	note := fmt.Sprintf("管理员为你充值钻石 %+d，当前余额 %d", in.Amount, prof.Diamond)
	if in.Mode == "set" {
		note = fmt.Sprintf("管理员将你的钻石余额设为 %d", prof.Diamond)
	}
	if strings.TrimSpace(in.Remark) != "" {
		note += "（" + strings.TrimSpace(in.Remark) + "）"
	}
	h.DB.Create(&model.EzfyNotice{UserId: uint(uid), Title: "钻石充值", Content: note})
	resp.OK(c, gin.H{"msg": note, "diamond": prof.Diamond})
}

// ============ 3. 二战聊天敏感词（独立维护页） ============

// AdminEzfyWordFilters GET /admin/ezfy-word-filters
func (h *AdminHandler) AdminEzfyWordFilters(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	word := strings.TrimSpace(c.Query("word"))
	q := h.DB.Model(&model.EzfyWordFilter{})
	if word != "" {
		q = q.Where("word LIKE ?", "%"+word+"%")
	}
	var total int64
	q.Count(&total)
	var rows []model.EzfyWordFilter
	lq := h.DB.Model(&model.EzfyWordFilter{})
	if word != "" {
		lq = lq.Where("word LIKE ?", "%"+word+"%")
	}
	lq.Order("id DESC").Offset(offset).Limit(size).Find(&rows)
	resp.OK(c, gin.H{"list": rows, "total": total, "page": page, "size": size})
}

// AdminEzfyWordFilterCreate POST /admin/ezfy-word-filters
func (h *AdminHandler) AdminEzfyWordFilterCreate(c *gin.Context) {
	var in struct {
		Word    string `json:"word"`
		Replace string `json:"replace"`
		Type    int    `json:"type"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	w := strings.TrimSpace(in.Word)
	if w == "" {
		resp.ParamError(c, "敏感词不能为空")
		return
	}
	if len([]rune(w)) > 50 {
		resp.ParamError(c, "敏感词最长50字")
		return
	}
	if in.Type != 2 {
		in.Type = 1
	}
	row := model.EzfyWordFilter{Word: w, Replace: strings.TrimSpace(in.Replace), Type: in.Type}
	if err := h.DB.Create(&row).Error; err != nil {
		resp.ParamError(c, "新增失败（该词可能已存在）："+err.Error())
		return
	}
	h.ezfyH().cfgsReload()
	resp.OK(c, gin.H{"msg": "已添加敏感词「" + w + "」", "id": row.ID})
}

// AdminEzfyWordFilterUpdate PUT /admin/ezfy-word-filters/:id
func (h *AdminHandler) AdminEzfyWordFilterUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var row model.EzfyWordFilter
	if err := h.DB.First(&row, id).Error; err != nil {
		resp.NotFound(c, "敏感词不存在")
		return
	}
	var in struct {
		Word    *string `json:"word"`
		Replace *string `json:"replace"`
		Type    *int    `json:"type"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if in.Word != nil {
		w := strings.TrimSpace(*in.Word)
		if w == "" {
			resp.ParamError(c, "敏感词不能为空")
			return
		}
		row.Word = w
	}
	if in.Replace != nil {
		row.Replace = strings.TrimSpace(*in.Replace)
	}
	if in.Type != nil {
		if *in.Type == 2 {
			row.Type = 2
		} else {
			row.Type = 1
		}
	}
	if err := h.DB.Save(&row).Error; err != nil {
		resp.ParamError(c, "保存失败："+err.Error())
		return
	}
	h.ezfyH().cfgsReload()
	resp.OK(c, gin.H{"msg": "已保存"})
}

// AdminEzfyWordFilterDelete DELETE /admin/ezfy-word-filters/:id
func (h *AdminHandler) AdminEzfyWordFilterDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.DB.Delete(&model.EzfyWordFilter{}, id).Error; err != nil {
		resp.ParamError(c, "删除失败："+err.Error())
		return
	}
	h.ezfyH().cfgsReload()
	resp.OK(c, gin.H{"msg": "已删除"})
}

// AdminEzfyWordFilterBulk POST /admin/ezfy-word-filters/bulk
//
// 批量导入：每行一个，格式 `词[,替换词[,类型]]`，类型 1=替换(默认) 2=拦截，# 开头忽略。
func (h *AdminHandler) AdminEzfyWordFilterBulk(c *gin.Context) {
	var in struct {
		Text string `json:"text"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	added, skipped := 0, 0
	for _, line := range strings.Split(in.Text, "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, ",")
		word := strings.TrimSpace(parts[0])
		if word == "" {
			continue
		}
		rep := ""
		typ := 1
		if len(parts) > 1 {
			rep = strings.TrimSpace(parts[1])
		}
		if len(parts) > 2 {
			if v, err := strconv.Atoi(strings.TrimSpace(parts[2])); err == nil && v == 2 {
				typ = 2
			}
		}
		row := model.EzfyWordFilter{Word: word, Replace: rep, Type: typ}
		if err := h.DB.Create(&row).Error; err != nil {
			skipped++
			continue
		}
		added++
	}
	h.ezfyH().cfgsReload()
	resp.OK(c, gin.H{"msg": fmt.Sprintf("导入完成：新增 %d 条，跳过（已存在/无效）%d 条", added, skipped),
		"added": added, "skipped": skipped})
}
