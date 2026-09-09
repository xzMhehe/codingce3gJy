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

// 婚恋模块（对齐诺哈 bbs/marriage：求婚/同意拒绝/离婚/已婚列表）
const marriageCost = 999 // 求婚/离婚手续费（G币，诺哈 propose.asp / divorce.asp）

type MarriageHandler struct{ DB *gorm.DB }

// 我的婚恋状态：伴侣/宝宝 + 收到/发出的求婚
func (h *MarriageHandler) Status(c *gin.Context) {
	uid := middleware.GetUID(c)
	var u model.User
	h.DB.First(&u, uid)
	partner := gin.H{}
	if u.PartnerID > 0 {
		var p model.User
		if err := h.DB.First(&p, u.PartnerID).Error; err == nil {
			partner = gin.H{"id": p.ID, "nickname": p.Nickname, "color": p.Color, "level": p.Level}
		}
	}
	// 收到的求婚（求婚中）
	var incoming []model.Marriage
	h.DB.Where("bid = ? AND status = 1", uid).Order("id DESC").Find(&incoming)
	inList := []gin.H{}
	for _, m := range incoming {
		var p model.User
		h.DB.First(&p, m.Aid)
		inList = append(inList, gin.H{"id": m.ID, "from_id": m.Aid, "from": p.Nickname, "color": p.Color, "message": m.Message, "created_at": m.CreatedAt})
	}
	// 我发出的求婚（求婚中）
	var outgoing []model.Marriage
	h.DB.Where("aid = ? AND status = 1", uid).Order("id DESC").Find(&outgoing)
	outList := []gin.H{}
	for _, m := range outgoing {
		var p model.User
		h.DB.First(&p, m.Bid)
		outList = append(outList, gin.H{"id": m.ID, "to_id": m.Bid, "to": p.Nickname, "color": p.Color, "message": m.Message, "created_at": m.CreatedAt})
	}
	resp.OK(c, gin.H{
		"married": u.PartnerID > 0, "partner": partner, "baby": u.BabyName,
		"incoming": inList, "outgoing": outList,
	})
}

type proposeReq struct {
	To      uint   `json:"to" binding:"required"`
	Message string `json:"message" binding:"required,max=200"`
}

// 求婚（对齐诺哈 propose.asp：需 999 G币，双方须未婚）
func (h *MarriageHandler) Propose(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req proposeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请填写求婚号码和表白语（200字以内）")
		return
	}
	if req.To == uid {
		resp.ParamError(c, "不能和自己结婚")
		return
	}
	var target model.User
	if err := h.DB.First(&target, req.To).Error; err != nil || target.Status == 0 {
		resp.NotFound(c, "求婚对象不存在")
		return
	}
	var me model.User
	h.DB.First(&me, uid)
	// 双方须未婚且无待处理求婚
	if me.PartnerID > 0 {
		resp.ParamError(c, "您已结婚，先离婚再来吧")
		return
	}
	if target.PartnerID > 0 {
		resp.ParamError(c, "对方已结婚")
		return
	}
	var pending int64
	h.DB.Model(&model.Marriage{}).
		Where("status = 1 AND ((aid = ? AND bid = ?) OR (aid = ? AND bid = ?))", uid, req.To, req.To, uid).
		Count(&pending)
	if pending > 0 {
		resp.ParamError(c, "已有求婚在进行中，等待对方处理")
		return
	}
	// 扣除 G币
	if me.Coins < marriageCost {
		resp.ParamError(c, "G币不足，求婚需要 " + strconv.Itoa(marriageCost) + " G币")
		return
	}
	h.DB.Model(&me).UpdateColumn("coins", gorm.Expr("coins - ?", marriageCost))
	h.DB.Create(&model.WalletLog{UserID: uid, Kind: "marriage", Title: "求婚费用", Currency: "coins", Delta: -marriageCost})
	// 创建求婚证书
	m := model.Marriage{Aid: uid, Bid: req.To, Message: req.Message, Status: 1}
	h.DB.Create(&m)
	// 通知对方
	h.DB.Create(&model.Notification{
		UserID: req.To, Type: "system", RefID: m.ID,
		Title:   me.Nickname + " 向您求婚！",
		Content: "表白：" + left(req.Message, 80) + "（到「婚恋」页处理：同意/拒绝）",
	})
	resp.OK(c, "求婚成功，已扣除 " + strconv.Itoa(marriageCost) + " G币并把求婚信息发给对方")
}

type handleProposeReq struct {
	Action string `json:"action" binding:"required,oneof=accept reject"`
}

// 处理求婚（对齐诺哈 propose_audit.asp：act=a 同意 / act=r 拒绝）
func (h *MarriageHandler) Handle(c *gin.Context) {
	uid := middleware.GetUID(c)
	id, _ := strconv.Atoi(c.Param("id"))
	var req handleProposeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "action 必须是 accept 或 reject")
		return
	}
	var m model.Marriage
	if err := h.DB.First(&m, id).Error; err != nil || m.Status != 1 {
		resp.NotFound(c, "求婚不存在或已处理")
		return
	}
	if m.Bid != uid {
		resp.Forbidden(c, "这不是发给您的求婚")
		return
	}
	var me model.User
	h.DB.First(&me, uid)
	now := time.Now()
	if req.Action == "accept" {
		if me.PartnerID > 0 {
			resp.ParamError(c, "您已结婚")
			return
		}
		// 结婚：双方互设伴侣
		h.DB.Model(&m).Updates(map[string]interface{}{"status": 0, "end_time": now})
		h.DB.Model(&model.User{}).Where("id = ?", me.ID).Update("partner_id", m.Aid)
		h.DB.Model(&model.User{}).Where("id = ?", m.Aid).Update("partner_id", me.ID)
		h.DB.Create(&model.Notification{UserID: m.Aid, Type: "system", RefID: m.ID,
			Title: "恭喜！" + me.Nickname + " 同意了您的求婚！", Content: "你们已成为夫妻！"})
		resp.OK(c, "恭喜你们成为夫妻！")
		return
	}
	// 拒绝：删除求婚记录
	h.DB.Delete(&m)
	h.DB.Create(&model.Notification{UserID: m.Aid, Type: "system", RefID: m.ID,
		Title: me.Nickname + " 拒绝了您的求婚", Content: "要多努力哦！"})
	resp.OK(c, "已拒绝对方的求婚")
}

// 离婚（对齐诺哈 divorce.asp：需 999 G币）
func (h *MarriageHandler) Divorce(c *gin.Context) {
	uid := middleware.GetUID(c)
	var me model.User
	h.DB.First(&me, uid)
	if me.PartnerID == 0 {
		resp.ParamError(c, "您还没有结婚")
		return
	}
	if me.Coins < marriageCost {
		resp.ParamError(c, "G币不足，离婚需要 " + strconv.Itoa(marriageCost) + " G币")
		return
	}
	var m model.Marriage
	if err := h.DB.Where("status = 0 AND ((aid = ? AND bid = ?) OR (aid = ? AND bid = ?))", uid, me.PartnerID, me.PartnerID, uid).First(&m).Error; err != nil {
		resp.NotFound(c, "婚姻证书不存在")
		return
	}
	partnerID := m.Aid
	if m.Aid == uid {
		partnerID = m.Bid
	}
	h.DB.Model(&me).UpdateColumn("coins", gorm.Expr("coins - ?", marriageCost))
	h.DB.Create(&model.WalletLog{UserID: uid, Kind: "marriage", Title: "离婚手续费", Currency: "coins", Delta: -marriageCost})
	h.DB.Delete(&m)
	h.DB.Model(&model.User{}).Where("id = ?", uid).Update("partner_id", 0)
	h.DB.Model(&model.User{}).Where("id = ?", partnerID).Update("partner_id", 0)
	h.DB.Create(&model.Notification{UserID: partnerID, Type: "system", RefID: m.ID,
		Title: me.Nickname + " 已经与您离婚了", Content: "祝各自安好"})
	resp.OK(c, "离婚成功，您又恢复到单身了")
}

// 已婚列表（对齐诺哈 marry_list.asp）
func (h *MarriageHandler) List(c *gin.Context) {
	var married []model.Marriage
	h.DB.Where("status = 0").Order("id DESC").Limit(50).Find(&married)
	out := []gin.H{}
	for _, m := range married {
		var a, b model.User
		h.DB.First(&a, m.Aid)
		h.DB.First(&b, m.Bid)
		out = append(out, gin.H{
			"id": m.ID,
			"a":  gin.H{"id": a.ID, "nickname": a.Nickname, "color": a.Color, "level": a.Level},
			"b":  gin.H{"id": b.ID, "nickname": b.Nickname, "color": b.Color, "level": b.Level},
			"created_at": m.EndTime,
		})
	}
	resp.OK(c, out)
}