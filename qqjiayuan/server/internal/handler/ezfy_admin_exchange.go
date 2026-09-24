package handler

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 二战风云管理端 —— 资源交易行维护
//
// 用户要求：「资源交易行 管理要有对应的维护页面，也能后端新增卖，只不过卖方是系统，
// 定价黄金/钻石，玩家卖只能按黄金买卖」。
//
// 所以这里的接口只做三件事：
//  1. 列出全部挂单（含玩家单与系统单，可按状态/关键字过滤）
//  2. 新增**系统挂单**（IsSystem=1, SellerId=0），计价货币可选黄金或钻石
//  3. 下架 / 删除挂单（系统单不涉及退款；玩家单在售时会把资源退回卖家主城）

// ezfyExchangeRefund 把在售玩家挂单的资源退回卖家主城（系统单不用退）
func (h *AdminHandler) ezfyExchangeRefund(e *model.EzfyExchange) {
	if e.IsSystem == 1 {
		return
	}
	var city model.EzfyCity
	if err := h.DB.Where("user_id = ?", e.SellerId).Order("id ASC").First(&city).Error; err != nil {
		return
	}
	switch e.EsType {
	case 1:
		city.Food += e.EsCount
	case 2:
		city.Steel += e.EsCount
	case 3:
		city.Oil += e.EsCount
	case 4:
		city.Rare += e.EsCount
	}
	h.ezfyH().saveCityRes(&city)
}

// AdminEzfyExchangeList GET /admin/ezfy-exchange?page=&size=&status=&word=
func (h *AdminHandler) AdminEzfyExchangeList(c *gin.Context) {
	page, offset, size := pageOf(c, 15)
	word := strings.TrimSpace(c.Query("word"))
	status := atoiOr(c.Query("status"), -1) // -1 全部

	q := h.DB.Model(&model.EzfyExchange{})
	if status >= 0 {
		q = q.Where("status = ?", status)
	}
	if word != "" {
		if id, err := strconv.Atoi(word); err == nil {
			q = q.Where("id = ? OR seller_id = ?", id, id)
		} else {
			q = q.Where("seller_name LIKE ?", "%"+word+"%")
		}
	}
	var total int64
	q.Count(&total)
	var rows []model.EzfyExchange
	q.Order("is_system DESC, id DESC").Offset(offset).Limit(size).Find(&rows)

	// 统计（在售系统单 / 在售玩家单）
	var sysOn, playerOn int64
	h.DB.Model(&model.EzfyExchange{}).Where("status = 0 AND is_system = 1").Count(&sysOn)
	h.DB.Model(&model.EzfyExchange{}).Where("status = 0 AND is_system = 0").Count(&playerOn)

	out := make([]gin.H, 0, len(rows))
	for _, e := range rows {
		seller := e.SellerName
		if e.IsSystem == 1 {
			seller = "系统"
		}
		out = append(out, gin.H{
			"id": e.ID, "seller_id": e.SellerId, "seller_name": seller,
			"is_system": e.IsSystem,
			"es_type":   e.EsType, "type_name": ezfyResNames[e.EsType],
			"es_count": e.EsCount, "total_price": e.TotalPrice,
			"unit_price": e.TotalPrice / maxInt64(1, e.EsCount),
			"currency":   e.Currency, "currency_name": ezfyMoneyName(e.Currency),
			"status": e.Status, "status_name": ezfyExchangeStatusName(e.Status),
			"buyer_id":   e.BuyerId,
			"created_at": e.CreatedAt, "updated_at": e.UpdatedAt,
		})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size,
		"system_on": sysOn, "player_on": playerOn})
}

func ezfyExchangeStatusName(s int) string {
	switch s {
	case 0:
		return "在售"
	case 1:
		return "已成交"
	case 2:
		return "已下架"
	}
	return "未知"
}

// AdminEzfyExchangeCreate POST /admin/ezfy-exchange
// 新增**系统挂单**：卖方固定为系统，货币可选 1 黄金 / 2 钻石。
// ★ 2026-09-24 用户要求：数量(es_count)是「资源数量」，选完模板后仍可二次修改；
//
//	新增 repeat(挂单数量) 支持一次上架多单，不用一单一单调。
func (h *AdminHandler) AdminEzfyExchangeCreate(c *gin.Context) {
	var in struct {
		EsType     int   `json:"es_type"`
		EsCount    int64 `json:"es_count"`
		TotalPrice int64 `json:"total_price"`
		Currency   int   `json:"currency"`
		Repeat     int   `json:"repeat"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if _, ok := ezfyResNames[in.EsType]; !ok {
		resp.ParamError(c, "资源类型错误")
		return
	}
	if in.EsCount <= 0 {
		resp.ParamError(c, "数量必须大于 0")
		return
	}
	if in.TotalPrice <= 0 {
		resp.ParamError(c, "总价必须大于 0")
		return
	}
	if in.Currency != ezfyMoneyDiamond {
		in.Currency = ezfyMoneyGold
	}
	repeat := in.Repeat
	if repeat <= 0 {
		repeat = 1
	}
	if repeat > 500 {
		resp.ParamError(c, "单次最多挂 500 单")
		return
	}
	// 系统挂单**不扣任何人的资源**（卖方是系统），只是往交易所里挂一批货
	var creates []model.EzfyExchange
	for i := 0; i < repeat; i++ {
		creates = append(creates, model.EzfyExchange{
			SellerId: 0, SellerName: "系统",
			EsType: in.EsType, EsCount: in.EsCount, TotalPrice: in.TotalPrice,
			Status: 0, IsSystem: 1, Currency: in.Currency,
		})
	}
	if err := h.DB.CreateInBatches(creates, 200).Error; err != nil {
		resp.ParamError(c, "上架失败："+err.Error())
		return
	}
	resp.OK(c, gin.H{"msg": fmt.Sprintf("已上架 %d 单系统挂单: %s×%d 售%d%s/单",
		repeat, ezfyResNames[in.EsType], in.EsCount, in.TotalPrice, ezfyMoneyName(in.Currency))})
}

// ============ 挂单模板维护（2026-09-24 用户要求） ============

// AdminEzfyExchangeTplList GET /admin/ezfy-exchange-tpls
func (h *AdminHandler) AdminEzfyExchangeTplList(c *gin.Context) {
	var rows []model.EzfyExchangeTemplate
	h.DB.Order("sort_no, id").Find(&rows)
	out := make([]gin.H, 0, len(rows))
	for _, t := range rows {
		out = append(out, gin.H{
			"id": t.ID, "name": t.Name,
			"es_type": t.EsType, "type_name": ezfyResNames[t.EsType],
			"es_count": t.EsCount, "total_price": t.TotalPrice,
			"currency": t.Currency, "currency_name": ezfyMoneyName(t.Currency),
			"sort_no": t.SortNo,
		})
	}
	resp.OK(c, gin.H{"list": out})
}

// AdminEzfyExchangeTplCreate POST /admin/ezfy-exchange-tpls
func (h *AdminHandler) AdminEzfyExchangeTplCreate(c *gin.Context) {
	var in struct {
		Name       string `json:"name"`
		EsType     int    `json:"es_type"`
		EsCount    int64  `json:"es_count"`
		TotalPrice int64  `json:"total_price"`
		Currency   int    `json:"currency"`
		SortNo     int    `json:"sort_no"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if msg := ezfyCheckTpl(&in.Name, in.EsType, in.EsCount, in.TotalPrice); msg != "" {
		resp.ParamError(c, msg)
		return
	}
	if in.Currency != ezfyMoneyDiamond {
		in.Currency = ezfyMoneyGold
	}
	if err := h.DB.Create(&model.EzfyExchangeTemplate{
		Name: in.Name, EsType: in.EsType, EsCount: in.EsCount,
		TotalPrice: in.TotalPrice, Currency: in.Currency, SortNo: in.SortNo,
	}).Error; err != nil {
		resp.ParamError(c, "新增失败："+err.Error())
		return
	}
	resp.OK(c, gin.H{"msg": "模板已新增"})
}

// AdminEzfyExchangeTplUpdate PUT /admin/ezfy-exchange-tpls/:id
func (h *AdminHandler) AdminEzfyExchangeTplUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var t model.EzfyExchangeTemplate
	if err := h.DB.First(&t, id).Error; err != nil {
		resp.NotFound(c, "模板不存在")
		return
	}
	var in struct {
		Name       string `json:"name"`
		EsType     int    `json:"es_type"`
		EsCount    int64  `json:"es_count"`
		TotalPrice int64  `json:"total_price"`
		Currency   int    `json:"currency"`
		SortNo     int    `json:"sort_no"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if msg := ezfyCheckTpl(&in.Name, in.EsType, in.EsCount, in.TotalPrice); msg != "" {
		resp.ParamError(c, msg)
		return
	}
	if in.Currency != ezfyMoneyDiamond {
		in.Currency = ezfyMoneyGold
	}
	if err := h.DB.Model(&model.EzfyExchangeTemplate{}).Where("id = ?", t.ID).
		Updates(map[string]interface{}{
			"name": in.Name, "es_type": in.EsType, "es_count": in.EsCount,
			"total_price": in.TotalPrice, "currency": in.Currency, "sort_no": in.SortNo,
		}).Error; err != nil {
		resp.ParamError(c, "修改失败："+err.Error())
		return
	}
	resp.OK(c, gin.H{"msg": "模板已保存"})
}

// AdminEzfyExchangeTplDelete DELETE /admin/ezfy-exchange-tpls/:id
func (h *AdminHandler) AdminEzfyExchangeTplDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var t model.EzfyExchangeTemplate
	if err := h.DB.First(&t, id).Error; err != nil {
		resp.NotFound(c, "模板不存在")
		return
	}
	h.DB.Delete(&model.EzfyExchangeTemplate{}, id)
	resp.OK(c, gin.H{"msg": "模板已删除"})
}

// ezfyCheckTpl 校验模板字段
func ezfyCheckTpl(name *string, esType int, esCount, totalPrice int64) string {
	if strings.TrimSpace(*name) == "" {
		return "请填写模板名"
	}
	*name = strings.TrimSpace(*name)
	if _, ok := ezfyResNames[esType]; !ok {
		return "资源类型错误"
	}
	if esCount <= 0 || totalPrice <= 0 {
		return "数量与总价必须大于 0"
	}
	return ""
}

// AdminEzfyExchangeOff POST /admin/ezfy-exchange/:id/off
func (h *AdminHandler) AdminEzfyExchangeOff(c *gin.Context) {
	id := int64(atoiOr(c.Param("id"), 0))
	var e model.EzfyExchange
	if err := h.DB.First(&e, id).Error; err != nil {
		resp.NotFound(c, "挂单不存在")
		return
	}
	if e.Status != 0 {
		resp.ParamError(c, "该挂单已经不在售")
		return
	}
	h.ezfyExchangeRefund(&e)
	h.DB.Model(&model.EzfyExchange{}).Where("id = ?", e.ID).Update("status", 2)
	msg := "已下架"
	if e.IsSystem != 1 {
		msg += "，资源已退回卖家城市"
	}
	resp.OK(c, gin.H{"msg": msg})
}

// AdminEzfyExchangeRemove DELETE /admin/ezfy-exchange/:id
// 直接删除记录（历史流水清理用；在售的玩家单会先把资源退回）。
//
// 注意：不能叫 AdminEzfyExchangeDelete —— 那个名字被 ezfy_admin.go 里「流水管理」的
// 旧接口占了（它只删记录、不退资源）。
func (h *AdminHandler) AdminEzfyExchangeRemove(c *gin.Context) {
	id := int64(atoiOr(c.Param("id"), 0))
	var e model.EzfyExchange
	if err := h.DB.First(&e, id).Error; err != nil {
		resp.NotFound(c, "挂单不存在")
		return
	}
	if e.Status == 0 {
		h.ezfyExchangeRefund(&e)
	}
	h.DB.Delete(&model.EzfyExchange{}, e.ID)
	resp.OK(c, gin.H{"msg": "已删除"})
}
