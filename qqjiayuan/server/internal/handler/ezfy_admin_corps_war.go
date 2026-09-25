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

// 二战风云 管理端 —— 军团宣战 / 军团商城维护（★ 2026-09-25 用户要求）
//
// 与「宣战管理（ezfy_admin_war.go）」并列：
//
//	1. 军团宣战：列表（状态筛选/军团名关键字/分页）、后台代宣战、强制结束、全部结束、删除
//	2. 军团商城：列表（类型/关键字/分页）、新增或更新、上下架、删除、道具池下拉
//
// ★ 状态约定（与 model.EzfyCorpsWar 严格一致）：1 = 待生效  2 = 交战中  3 = 已结束

// ezfyCorpsWarStatusText 军团宣战状态中文名
func ezfyCorpsWarStatusText(s int) string {
	switch s {
	case 1:
		return "待生效"
	case 2:
		return "交战中"
	case 3:
		return "已结束"
	}
	return "未知"
}

// ezfyCorpsWarLiveStatus 结合时间戳算「当前实际状态」（库里 status 可能是陈旧的待生效）
func ezfyCorpsWarLiveStatus(w *model.EzfyCorpsWar) int {
	if w.Status == 3 {
		return 3
	}
	now := time.Now().UnixMilli()
	if w.ExpireTime > 0 && now >= w.ExpireTime {
		return 3
	}
	if w.EffectTime > 0 && now >= w.EffectTime {
		return 2
	}
	return 1
}

// ============ 1. 军团宣战维护 ============

// AdminEzfyCorpsWars GET /admin/ezfy-corps-wars
//
// 查询参数：keyword（军团名关键字，兼容 word）、status（0 全部 / 1 待生效 / 2 交战中 / 3 已结束）
func (h *AdminHandler) AdminEzfyCorpsWars(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	word := strings.TrimSpace(c.Query("keyword"))
	if word == "" {
		word = strings.TrimSpace(c.Query("word"))
	}
	status := atoiOr(c.Query("status"), 0)

	q := h.DB.Model(&model.EzfyCorpsWar{})
	if status >= 1 && status <= 3 {
		// 按「真实状态」筛选：先按时间维度收窄，再在内存里精筛
		now := time.Now().UnixMilli()
		switch status {
		case 1:
			q = q.Where("status = 1 AND effect_time > ?", now)
		case 2:
			q = q.Where("status IN (1,2) AND effect_time <= ? AND expire_time > ?", now, now)
		case 3:
			q = q.Where("status = 3 OR expire_time <= ?", now)
		}
	}
	if word != "" {
		like := "%" + word + "%"
		var ids []uint
		h.DB.Model(&model.EzfyCorps{}).Where("name LIKE ?", like).Pluck("id", &ids)
		if len(ids) == 0 {
			resp.OK(c, gin.H{"list": []interface{}{}, "total": 0, "page": page, "size": size})
			return
		}
		q = q.Where("atk_corps_id IN ? OR def_corps_id IN ?", ids, ids)
	}

	var total int64
	q.Count(&total)
	var rows []model.EzfyCorpsWar
	q.Order("id DESC").Offset(offset).Limit(size).Find(&rows)

	type rowOut struct {
		model.EzfyCorpsWar
		LiveState  int    `json:"live_status"`
		StateText  string `json:"state_text"`
		StatusName string `json:"status_name"`
		LeftText   string `json:"left_text"`
		RemainHour int64  `json:"remaining_h"`
		DeclareAt  string `json:"declare_at_text"`
		EffectAt   string `json:"effect_at_text"`
		ExpireAt   string `json:"expire_at_text"`
		EndAt      string `json:"end_at_text"`
	}
	out := []rowOut{}
	now := time.Now().UnixMilli()
	for i := range rows {
		w := rows[i]
		live := ezfyCorpsWarLiveStatus(&w)
		w.Status = live // 展示用真实状态
		left := int64(0)
		if live != 3 {
			target := w.ExpireTime
			if live == 1 {
				target = w.EffectTime
			}
			if target > now {
				left = (target - now + 3599999) / 3600000
			}
		}
		out = append(out, rowOut{
			EzfyCorpsWar: w,
			LiveState:    live, StateText: ezfyCorpsWarStatusText(live),
			StatusName: ezfyCorpsWarStatusText(live),
			LeftText:   ezfyCorpsWarAdminLeftText(&w, live),
			RemainHour: left,
			DeclareAt:  ezfyFmtMs(w.DeclareTime), EffectAt: ezfyFmtMs(w.EffectTime),
			ExpireAt: ezfyFmtMs(w.ExpireTime), EndAt: ezfyFmtMs(w.EndTime),
		})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size,
		"delay_hours": ezfyCorpsWarDelayHours, "duration_hours": ezfyCorpsWarDurationHours,
		"total_hours": ezfyCorpsWarTotalHours})
}

// ezfyCorpsWarAdminLeftText 剩余时间文案（管理端列表）
func ezfyCorpsWarAdminLeftText(w *model.EzfyCorpsWar, live int) string {
	now := time.Now().UnixMilli()
	switch live {
	case 1:
		hs := (w.EffectTime - now + 3599999) / 3600000
		if hs < 1 {
			hs = 1
		}
		return fmt.Sprintf("约 %d 小时后生效", hs)
	case 2:
		hs := (w.ExpireTime - now) / 3600000
		if hs < 0 {
			hs = 0
		}
		return fmt.Sprintf("剩余约 %d 小时", hs)
	}
	return "—"
}

// AdminEzfyCorpsWarCreate POST /admin/ezfy-corps-wars {atk_corps_id, def_corps_id}
//
// 后台代军团长宣战，规则同用户端（12 小时生效 / 48 小时整场结束）。
func (h *AdminHandler) AdminEzfyCorpsWarCreate(c *gin.Context) {
	var in struct {
		AtkCorpsId uint `json:"atk_corps_id"`
		DefCorpsId uint `json:"def_corps_id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if in.AtkCorpsId == 0 || in.DefCorpsId == 0 {
		resp.ParamError(c, "请选择宣战方与被宣战方军团")
		return
	}
	if in.AtkCorpsId == in.DefCorpsId {
		resp.ParamError(c, "不能对自己军团宣战")
		return
	}
	var atk, def model.EzfyCorps
	if err := h.DB.First(&atk, in.AtkCorpsId).Error; err != nil {
		resp.ParamError(c, "宣战方军团不存在")
		return
	}
	if err := h.DB.First(&def, in.DefCorpsId).Error; err != nil {
		resp.ParamError(c, "被宣战方军团不存在")
		return
	}
	var n int64
	h.DB.Model(&model.EzfyCorpsWar{}).
		Where("status IN (1,2) AND ((atk_corps_id = ? AND def_corps_id = ?) OR (atk_corps_id = ? AND def_corps_id = ?))",
			in.AtkCorpsId, in.DefCorpsId, in.DefCorpsId, in.AtkCorpsId).Count(&n)
	if n > 0 {
		resp.ParamError(c, "双方已存在进行中的军团宣战")
		return
	}
	now := time.Now().UnixMilli()
	w := model.EzfyCorpsWar{
		AtkCorpsId: atk.ID, DefCorpsId: def.ID,
		AtkCorpsName: atk.Name, DefCorpsName: def.Name,
		AtkUserId: atk.LeaderUserId, Status: 1,
		DeclareTime: now,
		EffectTime:  now + int64(ezfyCorpsWarDelayHours)*3600000,
		ExpireTime:  now + int64(ezfyCorpsWarTotalHours)*3600000,
	}
	if err := h.DB.Create(&w).Error; err != nil {
		resp.ParamError(c, "创建失败："+err.Error())
		return
	}
	eh := h.ezfyH()
	eh.ezfyCorpsWarNotify(&w)
	eh.ezfySysChat("【军团宣战】%s 军团向 %s 军团宣战了，%d 小时后生效，%d 小时后整场结束！",
		atk.Name, def.Name, ezfyCorpsWarDelayHours, ezfyCorpsWarTotalHours)
	resp.OK(c, gin.H{"msg": fmt.Sprintf("已创建军团宣战：%s → %s（%s）",
		atk.Name, def.Name, ezfyCorpsWarStatusText(1)), "id": w.ID})
}

// AdminEzfyCorpsWarFinish POST /admin/ezfy-corps-wars/:id/finish
func (h *AdminHandler) AdminEzfyCorpsWarFinish(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var w model.EzfyCorpsWar
	if err := h.DB.First(&w, id).Error; err != nil {
		resp.NotFound(c, "军团宣战记录不存在")
		return
	}
	if w.Status == 3 {
		resp.ParamError(c, "该军团宣战已经结束")
		return
	}
	now := time.Now().UnixMilli()
	// 双保险：status 置 3 且 ExpireTime 拉到当下（与个人宣战一键结束同口径）
	if err := h.DB.Model(&model.EzfyCorpsWar{}).Where("id = ?", w.ID).
		Updates(map[string]interface{}{"status": 3, "expire_time": now, "end_time": now}).Error; err != nil {
		resp.ParamError(c, "操作失败："+err.Error())
		return
	}
	msg := fmt.Sprintf("【军团宣战结束】%s 军团 与 %s 军团 的战争状态已由管理员结束。", w.AtkCorpsName, w.DefCorpsName)
	var members []model.EzfyCorpsMember
	h.DB.Where("corps_id IN ?", []uint{w.AtkCorpsId, w.DefCorpsId}).Find(&members)
	seen := map[uint]bool{}
	for _, m := range members {
		if m.UserId == 0 || seen[m.UserId] {
			continue
		}
		seen[m.UserId] = true
		h.DB.Create(&model.EzfyNotice{UserId: m.UserId, Title: "军团宣战结束", Content: msg})
	}
	resp.OK(c, gin.H{"msg": "已强制结束该军团宣战（双方成员恢复和平）"})
}

// AdminEzfyCorpsWarFinishAll POST /admin/ezfy-corps-wars/finish-all
func (h *AdminHandler) AdminEzfyCorpsWarFinishAll(c *gin.Context) {
	now := time.Now().UnixMilli()
	var rows []model.EzfyCorpsWar
	h.DB.Where("status IN (1,2)").Find(&rows)
	if len(rows) == 0 {
		resp.OK(c, gin.H{"msg": "当前没有进行中的军团宣战", "count": 0})
		return
	}
	if err := h.DB.Model(&model.EzfyCorpsWar{}).Where("status IN (1,2)").
		Updates(map[string]interface{}{"status": 3, "end_time": now}).Error; err != nil {
		resp.ParamError(c, "操作失败："+err.Error())
		return
	}
	body := "【军团宣战结束】服务器已结束当前全部军团战争状态，所有军团恢复和平。"
	seen := map[uint]bool{}
	for _, w := range rows {
		var members []model.EzfyCorpsMember
		h.DB.Where("corps_id IN ?", []uint{w.AtkCorpsId, w.DefCorpsId}).Find(&members)
		for _, m := range members {
			if m.UserId == 0 || seen[m.UserId] {
				continue
			}
			seen[m.UserId] = true
			h.DB.Create(&model.EzfyNotice{UserId: m.UserId, Title: "军团宣战结束", Content: body})
		}
	}
	resp.OK(c, gin.H{"msg": fmt.Sprintf("已结束 %d 条军团宣战记录，涉及 %d 名成员", len(rows), len(seen)),
		"count": len(rows)})
}

// AdminEzfyCorpsWarDelete DELETE /admin/ezfy-corps-wars/:id
func (h *AdminHandler) AdminEzfyCorpsWarDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var w model.EzfyCorpsWar
	if err := h.DB.First(&w, id).Error; err != nil {
		resp.NotFound(c, "军团宣战记录不存在")
		return
	}
	if err := h.DB.Delete(&model.EzfyCorpsWar{}, id).Error; err != nil {
		resp.ParamError(c, "删除失败："+err.Error())
		return
	}
	resp.OK(c, gin.H{"msg": "已删除军团宣战记录"})
}

// ============ 2. 军团商城维护 ============

// AdminEzfyCorpsMallList GET /admin/ezfy-corps-mall?kind=&keyword=&page=&size=
//
// 查询参数：keyword（商品名关键字，兼容 word）、kind（0 全部 / 1 资源包 / 2 道具）
func (h *AdminHandler) AdminEzfyCorpsMallList(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	kind := atoiOr(c.Query("kind"), 0)
	word := strings.TrimSpace(c.Query("keyword"))
	if word == "" {
		word = strings.TrimSpace(c.Query("word"))
	}

	q := h.DB.Model(&model.EzfyCorpsMall{})
	if kind == 1 || kind == 2 {
		q = q.Where("kind = ?", kind)
	}
	if word != "" {
		q = q.Where("name LIKE ?", "%"+word+"%")
	}
	var total int64
	q.Count(&total)
	var rows []model.EzfyCorpsMall
	q.Order("sort ASC, id DESC").Offset(offset).Limit(size).Find(&rows)
	out := make([]gin.H, 0, len(rows))
	for i := range rows {
		it := rows[i]
		out = append(out, gin.H{
			"id": it.ID, "kind": it.Kind, "kind_name": ezfyCorpsMallKindName(it.Kind), "name": it.Name,
			"food": it.Food, "steel": it.Steel, "oil": it.Oil, "rare": it.Rare, "gold": it.Gold,
			"res_text": ezfyCorpsMallResText(&it),
			"item_id":  it.ItemId, "item_count": it.ItemCount,
			// limit 为前端口径（admin 商城维护页用 row.limit），limit_count 为兼容字段
			"price": it.Price, "limit": it.LimitCount, "limit_count": it.LimitCount,
			"stock": it.Stock, "sold": it.Sold,
			"sort": it.Sort, "enabled": it.Enabled,
		})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// AdminEzfyCorpsMallSave POST /admin/ezfy-corps-mall
//
// 新增（id=0）或按 id 更新（整表字段提交，前端表单口径）。
func (h *AdminHandler) AdminEzfyCorpsMallSave(c *gin.Context) {
	var in struct {
		ID        uint   `json:"id"`
		Kind      int    `json:"kind"`
		Name      string `json:"name"`
		Food      int64  `json:"food"`
		Steel     int64  `json:"steel"`
		Oil       int64  `json:"oil"`
		Rare      int64  `json:"rare"`
		Gold      int64  `json:"gold"`
		ItemId    int    `json:"item_id"`
		ItemCount int    `json:"item_count"`
		Price     int64  `json:"price"`
		// 「每人限购」前端口径为 limit（admin 商城维护页 POST limit），limit_count 为兼容字段
		Limit      *int `json:"limit"`
		LimitCount *int `json:"limit_count"`
		Stock      int  `json:"stock"`
		Sort       int  `json:"sort"`
		Enabled    int  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	// 取限购值：优先 limit，其次 limit_count，都没有则 0（不限购）
	limitCount := 0
	if in.Limit != nil {
		limitCount = *in.Limit
	} else if in.LimitCount != nil {
		limitCount = *in.LimitCount
	}
	if in.Kind != 1 && in.Kind != 2 {
		resp.ParamError(c, "商品类型只能是 1资源包 或 2道具")
		return
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		resp.ParamError(c, "请填写商品名称")
		return
	}
	if len([]rune(name)) > 40 {
		resp.ParamError(c, "商品名称过长（限 40 字）")
		return
	}
	if in.Price < 0 {
		resp.ParamError(c, "价格不能为负")
		return
	}
	if limitCount < 0 {
		resp.ParamError(c, "每人限购不能为负（0 = 不限）")
		return
	}
	if in.Stock < -1 {
		resp.ParamError(c, "总库存不能小于 -1（-1 = 不限）")
		return
	}
	if in.Kind == 1 {
		if in.Food < 0 || in.Steel < 0 || in.Oil < 0 || in.Rare < 0 || in.Gold < 0 {
			resp.ParamError(c, "资源数量不能为负")
			return
		}
		if in.Food+in.Steel+in.Oil+in.Rare+in.Gold <= 0 {
			resp.ParamError(c, "资源包至少要填写一种资源数量")
			return
		}
	} else {
		if in.ItemId <= 0 || in.ItemCount <= 0 {
			resp.ParamError(c, "道具商品需要选择道具并填写数量")
			return
		}
		var cfg model.EzfyCfgItem
		if err := h.DB.First(&cfg, in.ItemId).Error; err != nil {
			resp.ParamError(c, "所选道具不存在")
			return
		}
	}
	if in.Enabled != 0 && in.Enabled != 1 {
		in.Enabled = 1
	}
	// 道具商品不存资源字段；资源包不存道具字段（避免脏数据）
	set := model.EzfyCorpsMall{
		Kind: in.Kind, Name: name, Price: in.Price, LimitCount: limitCount,
		Stock: in.Stock, Sort: in.Sort, Enabled: in.Enabled,
	}
	if in.Kind == 1 {
		set.Food, set.Steel, set.Oil, set.Rare, set.Gold = in.Food, in.Steel, in.Oil, in.Rare, in.Gold
	} else {
		set.ItemId, set.ItemCount = in.ItemId, in.ItemCount
	}
	if in.ID > 0 {
		var old model.EzfyCorpsMall
		if err := h.DB.First(&old, in.ID).Error; err != nil {
			resp.NotFound(c, "商品不存在")
			return
		}
		set.ID = old.ID
		set.Sold = old.Sold
		if err := h.DB.Model(&model.EzfyCorpsMall{}).Where("id = ?", old.ID).Updates(map[string]interface{}{
			"kind": set.Kind, "name": set.Name, "food": set.Food, "steel": set.Steel,
			"oil": set.Oil, "rare": set.Rare, "gold": set.Gold,
			"item_id": set.ItemId, "item_count": set.ItemCount, "price": set.Price,
			"limit_count": set.LimitCount, "stock": set.Stock, "sort": set.Sort, "enabled": set.Enabled,
		}).Error; err != nil {
			resp.ParamError(c, "保存失败："+err.Error())
			return
		}
		resp.OK(c, gin.H{"msg": "已保存商品「" + set.Name + "」", "id": old.ID})
		return
	}
	if err := h.DB.Create(&set).Error; err != nil {
		resp.ParamError(c, "创建失败："+err.Error())
		return
	}
	resp.OK(c, gin.H{"msg": "已新增商品「" + set.Name + "」", "id": set.ID})
}

// AdminEzfyCorpsMallToggle POST /admin/ezfy-corps-mall/:id/toggle
func (h *AdminHandler) AdminEzfyCorpsMallToggle(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var it model.EzfyCorpsMall
	if err := h.DB.First(&it, id).Error; err != nil {
		resp.NotFound(c, "商品不存在")
		return
	}
	next := 1
	if it.Enabled == 1 {
		next = 0
	}
	if err := h.DB.Model(&model.EzfyCorpsMall{}).Where("id = ?", it.ID).Update("enabled", next).Error; err != nil {
		resp.ParamError(c, "操作失败："+err.Error())
		return
	}
	word := "已上架"
	if next == 0 {
		word = "已下架"
	}
	resp.OK(c, gin.H{"msg": word + "「" + it.Name + "」", "enabled": next})
}

// AdminEzfyCorpsMallDelete DELETE /admin/ezfy-corps-mall/:id
func (h *AdminHandler) AdminEzfyCorpsMallDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var it model.EzfyCorpsMall
	if err := h.DB.First(&it, id).Error; err != nil {
		resp.NotFound(c, "商品不存在")
		return
	}
	if err := h.DB.Delete(&model.EzfyCorpsMall{}, id).Error; err != nil {
		resp.ParamError(c, "删除失败："+err.Error())
		return
	}
	resp.OK(c, gin.H{"msg": "已删除商品「" + it.Name + "」"})
}

// AdminEzfyCorpsMallItems GET /admin/ezfy-corps-mall/items
//
// 道具池 = 现有游戏道具配置 EzfyCfgItem（供前端下拉选择）
func (h *AdminHandler) AdminEzfyCorpsMallItems(c *gin.Context) {
	var rows []model.EzfyCfgItem
	h.DB.Order("id ASC").Find(&rows)
	out := make([]gin.H, 0, len(rows))
	for i := range rows {
		it := rows[i]
		out = append(out, gin.H{
			"id": it.ID, "name": it.Name, "item_type": it.ItemType,
			// category_name 为前端口径（admin 商城维护页下拉用它拼标签），category 为兼容字段
			"category": ezfyItemCategory(&it), "category_name": ezfyItemCategory(&it),
			"description": it.Description,
		})
	}
	resp.OK(c, gin.H{"list": out, "total": len(out)})
}
