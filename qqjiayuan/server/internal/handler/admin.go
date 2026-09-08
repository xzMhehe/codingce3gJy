package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

type AdminHandler struct{ DB *gorm.DB }

// ---- 概览 ----
func (h *AdminHandler) Stats(c *gin.Context) {
	var users, threads, replies, boards, online int64
	h.DB.Model(&model.User{}).Count(&users)
	h.DB.Model(&model.Thread{}).Where("status = 1").Count(&threads)
	h.DB.Model(&model.Reply{}).Where("status = 1").Count(&replies)
	h.DB.Model(&model.Board{}).Count(&boards)
	h.DB.Model(&model.User{}).Where("last_active_at > ?", time.Now().Add(-10*time.Minute)).Count(&online)
	resp.OK(c, gin.H{"users": users, "threads": threads, "replies": replies, "boards": boards, "online": online})
}

// ---- 用户管理 ----
func (h *AdminHandler) Users(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	word := c.Query("word")
	q := h.DB.Model(&model.User{})
	if word != "" {
		q = q.Where("username = ? OR nickname LIKE ?", word, "%"+word+"%")
	}
	var total int64
	q.Count(&total)
	var users []model.User
	q.Preload("Roles").Preload("Badges").Order("id ASC").Offset(offset).Limit(size).Find(&users)
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": users})
}

// UserDetail 用户详情：完整资料 + IP + 地址/证件/密保/联系方式 + 最近日志（对齐诺哈 admin/user）
func (h *AdminHandler) UserDetail(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var u model.User
	if err := h.DB.First(&u, id).Error; err != nil {
		resp.NotFound(c, "用户不存在")
		return
	}
	var addr model.UserAddress
	h.DB.Where("user_id = ?", u.ID).First(&addr)
	var doc model.UserDocument
	h.DB.Where("user_id = ?", u.ID).First(&doc)
	var prot model.UserProtection
	h.DB.Where("user_id = ?", u.ID).First(&prot)
	var ct model.UserContact
	h.DB.Where("user_id = ?", u.ID).First(&ct)
	var logs []model.UserLog
	h.DB.Where("user_id = ?", u.ID).Order("created_at DESC").Limit(20).Find(&logs)
	// 证件号脱敏
	num := doc.Number
	if len(num) > 6 {
		num = num[:3] + "***********" + num[len(num)-3:]
	}
	resp.OK(c, gin.H{
		"user": gin.H{
			"id": u.ID, "username": u.Username, "nickname": u.Nickname, "gender": u.Gender,
			"age": u.Age, "birth_year": u.BirthYear, "birth_month": u.BirthMonth, "birth_day": u.BirthDay,
			"birth_type": u.BirthType, "solar": u.Solar, "lunar": u.Lunar,
			"signature": u.Signature, "introduction": u.Introduction, "city": u.City, "color": u.Color,
			"level": u.Level, "exp": u.Exp, "coins": u.Coins, "yuanbao": u.YuanBao, "jinzuan": u.JinZuan, "youquan": u.YouQuan,
			"active_days": u.ActiveDays, "hours": u.Hours, "achieve": u.Achieve,
			"friend_policy": u.FriendPolicy, "config": u.Config, "noble": u.Noble,
			"status": u.Status, "add_ip": u.AddIP, "last_ip": u.LastIP,
			"created_at": u.CreatedAt, "last_login_at": u.LastLoginAt, "last_active_at": u.LastActiveAt,
			"has_paypass": u.PayPass != "",
		},
		"address":     addr,
		"document":    gin.H{"type": doc.Type, "real_name": doc.RealName, "number": num, "has_doc": doc.ID > 0},
		"protection":  gin.H{"issue": prot.Issue, "has_protection": prot.ID > 0},
		"contact":     gin.H{"qq": ct.QQ, "mail": ct.Mail, "phone": ct.Phone},
		"logs":        logs,
	})
}

// 家族管理：列表（含成员数，分页）
func (h *AdminHandler) Families(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	q := h.DB.Model(&model.Family{}).Where("status = 1")
	var total int64
	q.Count(&total)
	var fams []model.Family
	q.Preload("Owner").Order("id ASC").Offset(offset).Limit(size).Find(&fams)
	out := []gin.H{}
	for _, f := range fams {
		var cnt int64
		h.DB.Model(&model.FamilyMember{}).Where("family_id = ?", f.ID).Count(&cnt)
		owner := ""
		if f.Owner != nil {
			owner = f.Owner.Nickname
		}
		out = append(out, gin.H{"id": f.ID, "name": f.Name, "slogan": f.Slogan, "owner": owner,
			"members": cnt, "announcement": f.Announcement, "battle_score": f.BattleScore,
			"is_feature": f.IsFeature, "category": f.Category})
	}
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": out})
}

// 待审核家族列表（管理端）
func (h *AdminHandler) PendingFamilies(c *gin.Context) {
	var fams []model.Family
	h.DB.Preload("Owner").Where("status = 2").Order("created_at ASC").Find(&fams)
	out := []gin.H{}
	for _, f := range fams {
		owner := ""
		if f.Owner != nil {
			owner = f.Owner.Nickname
		}
		out = append(out, gin.H{"id": f.ID, "name": f.Name, "slogan": f.Slogan,
			"description": f.Description, "category": f.Category, "owner_id": f.OwnerID,
			"owner": owner, "created_at": f.CreatedAt})
	}
	resp.OK(c, out)
}

// 特色家族设置
func (h *AdminHandler) FamilyFeature(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		IsFeature int `json:"is_feature"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数有误")
		return
	}
	if req.IsFeature != 0 && req.IsFeature != 1 {
		req.IsFeature = 0
	}
	h.DB.Model(&model.Family{}).Where("id = ?", id).Update("is_feature", req.IsFeature)
	resp.OK(c, nil)
}

// 家族审核：通过（扣 500 G币并成立）/ 拒绝（删除并保留申请记录日志）
func (h *AdminHandler) FamilyReview(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Approve bool `json:"approve"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数有误")
		return
	}
	var fam model.Family
	if err := h.DB.First(&fam, id).Error; err != nil || fam.Status != 2 {
		resp.ParamError(c, "家族不存在或不在待审状态")
		return
	}
	if req.Approve {
		var u model.User
		if err := h.DB.First(&u, fam.OwnerID).Error; err != nil {
			resp.ParamError(c, "族长账号不存在")
			return
		}
		if u.Coins < familyCreateCost {
			resp.ParamError(c, "族长G币不足 " + strconv.Itoa(familyCreateCost) + "，无法通过（可先给族长充值）")
			return
		}
		h.DB.Model(&u).Update("coins", gorm.Expr("coins - ?", familyCreateCost))
		h.DB.Model(&fam).Update("status", 1)
		h.DB.Create(&model.FamilyMember{FamilyID: fam.ID, UserID: fam.OwnerID, Role: "owner"})
		h.DB.Create(&model.FamilyActivity{FamilyID: fam.ID, UserID: fam.OwnerID, Content: "家族审核通过，正式成立！"})
		resp.OK(c, gin.H{"approved": true})
		return
	}
	// 拒绝：删除家族及其关联数据
	h.DB.Where("family_id = ?", fam.ID).Delete(&model.FamilyMember{})
	h.DB.Where("family_id = ?", fam.ID).Delete(&model.FamilyActivity{})
	h.DB.Where("family_id = ?", fam.ID).Delete(&model.FamilySignIn{})
	h.DB.Delete(&fam)
	resp.OK(c, gin.H{"approved": false})
}

// 家族解散/恢复
func (h *AdminHandler) FamilyStatus(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Status int `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数有误")
		return
	}
	h.DB.Model(&model.Family{}).Where("id = ?", id).Update("status", req.Status)
	resp.OK(c, nil)
}

// 家族公告修改
func (h *AdminHandler) FamilyAnn(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Announcement string `json:"announcement"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数有误")
		return
	}
	h.DB.Model(&model.Family{}).Where("id = ?", id).Update("announcement", req.Announcement)
	resp.OK(c, nil)
}

// 同城管理：同城客栈下的子板块（城市）
func (h *AdminHandler) Tongcheng(c *gin.Context) {
	var root model.Board
	h.DB.Where("name = ?", "同城客栈").First(&root)
	var subs []model.Board
	if root.ID > 0 {
		h.DB.Where("parent_id = ?", root.ID).Order("sort ASC").Find(&subs)
	}
	resp.OK(c, subs)
}

// T台秀：当前上榜用户（默认经验最高，可被配置覆盖）
func (h *AdminHandler) Ttou(c *gin.Context) {
	var ttouID string
	h.DB.Model(&model.Setting{}).Where("key = ?", "ttou_user_id").First(&struct {
		Key   string `gorm:"primaryKey"`
		Value string
	}{})
	h.DB.Raw("SELECT value FROM settings WHERE `key`='ttou_user_id'").Scan(&ttouID)
	var u model.User
	if ttouID != "" {
		var id uint
		h.DB.Raw("SELECT id FROM users WHERE id = ?", ttouID).Scan(&id)
		if id > 0 {
			h.DB.First(&u, id)
		}
	}
	if u.ID == 0 {
		h.DB.Where("status = 1").Order("exp DESC").First(&u)
	}
	resp.OK(c, gin.H{"id": u.ID, "nickname": u.Nickname, "color": u.Color, "exp": u.Exp})
}

// 设置 T台秀用户
func (h *AdminHandler) TtouSet(c *gin.Context) {
	var req struct {
		UserID uint `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请填写用户号码")
		return
	}
	var u model.User
	if err := h.DB.First(&u, req.UserID).Error; err != nil {
		resp.NotFound(c, "用户不存在")
		return
	}
	h.DB.Exec("REPLACE INTO settings(`key`,`value`) VALUES ('ttou_user_id', ?)", u.Username)
	resp.OK(c, nil)
}

// 移除 T台秀指定（恢复自动取经验最高）
func (h *AdminHandler) TtouClear(c *gin.Context) {
	h.DB.Exec("DELETE FROM settings WHERE `key` = 'ttou_user_id'")
	resp.OK(c, nil)
}

// T台秀上榜申请列表
func (h *AdminHandler) TtouApplies(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	var total int64
	h.DB.Model(&model.TtouApply{}).Where("status = 0").Count(&total)
	var applies []model.TtouApply
	h.DB.Where("status = 0").Order("id DESC").Offset(offset).Limit(size).Find(&applies)
	list := []gin.H{}
	for _, a := range applies {
		var u model.User
		h.DB.Preload("Priv").First(&u, a.UserID)
		list = append(list, gin.H{
			"id": a.ID, "user_id": a.UserID, "nickname": u.Nickname, "username": u.Username,
			"color": u.Color, "avatar": u.Avatar, "avatar_base64": u.AvatarBase64,
			"level_icon": u.LevelIcon, "slogan": a.Slogan, "created_at": a.CreatedAt,
		})
	}
	resp.OK(c, gin.H{"list": list, "total": total, "page": page, "size": size})
}

// 采纳上榜申请：设为当前秀主
func (h *AdminHandler) TtouApplyAccept(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var a model.TtouApply
	if err := h.DB.First(&a, id).Error; err != nil {
		resp.NotFound(c, "申请不存在")
		return
	}
	var u model.User
	if err := h.DB.First(&u, a.UserID).Error; err != nil {
		resp.NotFound(c, "用户不存在")
		return
	}
	h.DB.Exec("REPLACE INTO settings(`key`,`value`) VALUES ('ttou_user_id', ?)", u.Username)
	h.DB.Model(&model.TtouApply{}).Where("id = ?", id).Update("status", 1)
	resp.OK(c, nil)
}

// 钱包管理：用户G币列表
func (h *AdminHandler) Wallets(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	word := c.Query("word")
	q := h.DB.Model(&model.User{})
	if word != "" {
		q = q.Where("username = ? OR nickname LIKE ?", word, "%"+word+"%")
	}
	var total int64
	q.Count(&total)
	var users []model.User
	q.Order("coins DESC").Offset(offset).Limit(size).Find(&users)
	out := []gin.H{}
	for _, u := range users {
		bank := 0
		var acc model.BankAccount
		if err := h.DB.Where("user_id = ?", u.ID).First(&acc).Error; err == nil {
			bank = acc.Balance
		}
		out = append(out, gin.H{"id": u.ID, "nickname": u.Nickname, "color": u.Color, "coins": u.Coins, "bank": bank,
			"yuanbao": u.YuanBao, "jinzuan": u.JinZuan, "youquan": u.YouQuan})
	}
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": out})
}

// 设置用户货币（G币/元宝/金钻/友友券）
func (h *AdminHandler) WalletSet(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Coins   int `json:"coins"`
		YuanBao int `json:"yuanbao"`
		JinZuan int `json:"jinzuan"`
		YouQuan int `json:"youquan"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Coins < 0 || req.YuanBao < 0 || req.JinZuan < 0 || req.YouQuan < 0 {
		resp.ParamError(c, "货币数值需大于等于 0")
		return
	}
	h.DB.Model(&model.User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"coins": req.Coins, "yuanbao": req.YuanBao, "jinzuan": req.JinZuan, "youquan": req.YouQuan,
	})
	resp.OK(c, nil)
}

// 设置用户家园资料（等级/活跃天数/成就点/城市）
func (h *AdminHandler) UserHomeSet(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Level      int     `json:"level"`
		ActiveDays float64 `json:"active_days"`
		Achieve    int     `json:"achieve"`
		City       string  `json:"city"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数有误")
		return
	}
	if req.Level < 1 {
		req.Level = 1
	}
	if req.ActiveDays < 0 {
		req.ActiveDays = 0
	}
	if req.Achieve < 0 {
		req.Achieve = 0
	}
	h.DB.Model(&model.User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"level": req.Level, "active_days": req.ActiveDays, "achieve": req.Achieve, "city": req.City,
	})
	resp.OK(c, nil)
}

func (h *AdminHandler) UserStatus(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Status int `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || (req.Status != 0 && req.Status != 1) {
		resp.ParamError(c, "status 只能是 0（封禁）或 1（正常）")
		return
	}
	uid := middleware.GetUID(c)
	if uint(id) == uid {
		resp.ParamError(c, "不能封禁自己")
		return
	}
	if err := h.DB.Model(&model.User{}).Where("id = ?", id).Update("status", req.Status).Error; err != nil {
		resp.NotFound(c, "用户不存在")
		return
	}
	resp.OK(c, nil)
}

func (h *AdminHandler) ResetPassword(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Password string `json:"password" binding:"required,min=6,max=20"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "新密码6-20位")
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	h.DB.Model(&model.User{}).Where("id = ?", id).Update("password", string(hash))
	resp.OK(c, "密码已重置")
}

func (h *AdminHandler) UserRoles(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		RoleIDs []uint `json:"role_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请提交 role_ids 数组")
		return
	}
	var user model.User
	if err := h.DB.First(&user, id).Error; err != nil {
		resp.NotFound(c, "用户不存在")
		return
	}
	var roles []model.Role
	h.DB.Where("id IN ?", req.RoleIDs).Find(&roles)
	h.DB.Model(&user).Association("Roles").Replace(&roles)
	resp.OK(c, nil)
}

// 设身份：贵族等级 / 婚恋伴侣 / 宝宝
func (h *AdminHandler) UserExtras(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Noble     int    `json:"noble"`
		PartnerID uint   `json:"partner_id"`
		BabyName  string `json:"baby_name" binding:"max=20"`
		PrivID    uint   `json:"priv_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数不对：noble 0-2，宝宝名20字以内")
		return
	}
	if req.Noble < 0 || req.Noble > 2 {
		resp.ParamError(c, "贵族等级只能是 0无 1一级 2二级")
		return
	}
	var user model.User
	if err := h.DB.First(&user, id).Error; err != nil {
		resp.NotFound(c, "用户不存在")
		return
	}
	if req.PartnerID > 0 {
		var p model.User
		if err := h.DB.First(&p, req.PartnerID).Error; err != nil {
			resp.NotFound(c, "伴侣号码不存在")
			return
		}
		if req.PartnerID == user.ID {
			resp.ParamError(c, "不能和自己结成城堡")
			return
		}
	}
	updates := map[string]interface{}{
		"noble": req.Noble, "partner_id": req.PartnerID, "baby_name": req.BabyName,
		"priv_id": req.PrivID,
	}
	h.DB.Model(&user).Updates(updates)
	resp.OK(c, nil)
}

// ---- 板块管理 ----
func (h *AdminHandler) Boards(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "15"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 15
	}
	q := h.DB.Model(&model.Board{})
	if word := c.Query("word"); word != "" {
		q = q.Where("name LIKE ?", "%"+word+"%")
	}
	var total int64
	q.Count(&total)
	if maxPage := int(total+int64(size)-1) / size; maxPage < 1 {
		page = 1
	} else if page > maxPage {
		page = maxPage
	}
	var boards []model.Board
	q.Order("parent_id ASC, sort ASC, id ASC").Offset((page - 1) * size).Limit(size).Find(&boards)
	resp.OK(c, gin.H{"list": boards, "total": total, "page": page, "size": size})
}

type boardReq struct {
	Name        string `json:"name" binding:"required,min=1,max=30"`
	ParentID    uint   `json:"parent_id"`
	CategoryID  uint   `json:"category_id"`
	Description string `json:"description" binding:"max=500"`
	Notice      string `json:"notice"`
	Tags        string `json:"tags" binding:"max=100"`
	ModeratorID uint   `json:"moderator_id"`
	MembersOnly int    `json:"members_only"`
	Sort        int    `json:"sort"`
	Status      int    `json:"status"`
}

func (h *AdminHandler) CreateBoard(c *gin.Context) {
	var req boardReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "板块名称1-30字")
		return
	}
	if req.Status == 0 && req.ParentID == 0 {
		req.Status = 1
	}
	board := model.Board{Name: req.Name, ParentID: req.ParentID, CategoryID: req.CategoryID,
		Description: req.Description, Notice: req.Notice, Tags: req.Tags,
		ModeratorID: req.ModeratorID, MembersOnly: req.MembersOnly, Sort: req.Sort, Status: req.Status}
	if req.ParentID != 0 {
		var parent model.Board
		if err := h.DB.First(&parent, req.ParentID).Error; err != nil || parent.ParentID != 0 {
			resp.ParamError(c, "上级分区不存在（只支持两级）")
			return
		}
		if req.CategoryID != 0 {
			var ct model.BoardCategory
			if err := h.DB.First(&ct, req.CategoryID).Error; err != nil || ct.ParentID != req.ParentID {
				resp.ParamError(c, "分类不属于该分区")
				return
			}
		}
	} else {
		req.CategoryID = 0
		board.CategoryID = 0
	}
	h.DB.Create(&board)
	resp.OK(c, board)
}

func (h *AdminHandler) UpdateBoard(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req boardReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "板块名称1-30字")
		return
	}
	var board model.Board
	if err := h.DB.First(&board, id).Error; err != nil {
		resp.NotFound(c, "板块不存在")
		return
	}
	if board.ParentID == 0 {
		req.CategoryID = 0
		req.ModeratorID = 0
		req.MembersOnly = 0
	}
	updates := map[string]interface{}{
		"name": req.Name, "description": req.Description, "notice": req.Notice, "tags": req.Tags,
		"sort": req.Sort, "status": boolToInt(req.Status != 0),
		"category_id": req.CategoryID, "moderator_id": req.ModeratorID, "members_only": req.MembersOnly,
	}
	if board.ParentID == 0 {
		updates["moderator_id"] = board.ModeratorID
		updates["members_only"] = board.MembersOnly
	}
	h.DB.Model(&board).Updates(updates)
	resp.OK(c, board)
}

func (h *AdminHandler) DeleteBoard(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var cnt int64
	h.DB.Model(&model.Board{}).Where("parent_id = ?", id).Count(&cnt)
	if cnt > 0 {
		resp.ParamError(c, "请先删除该分区下的子板块")
		return
	}
	var cnt2 int64
	h.DB.Model(&model.Thread{}).Where("board_id = ? AND status = 1", id).Count(&cnt2)
	if cnt2 > 0 {
		resp.ParamError(c, "该板块还有帖子，不能删除")
		return
	}
	h.DB.Delete(&model.Board{}, id)
	resp.OK(c, nil)
}

// ---- 板块分类 ----
func (h *AdminHandler) BoardCategories(c *gin.Context) {
	var list []model.BoardCategory
	h.DB.Order("parent_id ASC, sort ASC, id ASC").Find(&list)
	resp.OK(c, list)
}

func (h *AdminHandler) CreateBoardCategory(c *gin.Context) {
	var req struct {
		ParentID uint   `json:"parent_id" binding:"required"`
		Name     string `json:"name" binding:"required,min=1,max=30"`
		Sort     int    `json:"sort"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "分类名称1-30字")
		return
	}
	var parent model.Board
	if err := h.DB.First(&parent, req.ParentID).Error; err != nil || parent.ParentID != 0 {
		resp.ParamError(c, "分区不存在")
		return
	}
	ct := model.BoardCategory{ParentID: req.ParentID, Name: req.Name, Sort: req.Sort}
	h.DB.Create(&ct)
	resp.OK(c, ct)
}

func (h *AdminHandler) UpdateBoardCategory(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Name string `json:"name" binding:"required,min=1,max=30"`
		Sort int    `json:"sort"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "分类名称1-30字")
		return
	}
	var ct model.BoardCategory
	if err := h.DB.First(&ct, id).Error; err != nil {
		resp.NotFound(c, "分类不存在")
		return
	}
	h.DB.Model(&ct).Updates(map[string]interface{}{"name": req.Name, "sort": req.Sort})
	resp.OK(c, ct)
}

func (h *AdminHandler) DeleteBoardCategory(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var cnt int64
	h.DB.Model(&model.Board{}).Where("category_id = ?", id).Count(&cnt)
	if cnt > 0 {
		resp.ParamError(c, "该分类下还有板块，请先移出")
		return
	}
	h.DB.Delete(&model.BoardCategory{}, id)
	resp.OK(c, nil)
}

// ---- 会员制版块成员 ----
func (h *AdminHandler) BoardMembers(c *gin.Context) {
	bid, _ := strconv.Atoi(c.DefaultQuery("board_id", "0"))
	q := h.DB.Model(&model.BoardMember{})
	if bid > 0 {
		q = q.Where("board_id = ?", bid)
	}
	var list []model.BoardMember
	q.Preload("User").Order("id DESC").Limit(200).Find(&list)
	resp.OK(c, list)
}

func (h *AdminHandler) BoardMemberAdd(c *gin.Context) {
	var req struct {
		BoardID uint `json:"board_id" binding:"required"`
		UserID  uint `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数不对")
		return
	}
	var cnt int64
	h.DB.Model(&model.BoardMember{}).Where("board_id = ? AND user_id = ?", req.BoardID, req.UserID).Count(&cnt)
	if cnt > 0 {
		h.DB.Model(&model.BoardMember{}).Where("board_id = ? AND user_id = ?", req.BoardID, req.UserID).Update("status", 1)
		resp.OK(c, nil)
		return
	}
	h.DB.Create(&model.BoardMember{BoardID: req.BoardID, UserID: req.UserID, Status: 1})
	resp.OK(c, nil)
}

func (h *AdminHandler) BoardMemberRemove(c *gin.Context) {
	bid, _ := strconv.Atoi(c.Param("boardId"))
	uid, _ := strconv.Atoi(c.Param("userId"))
	h.DB.Where("board_id = ? AND user_id = ?", bid, uid).Delete(&model.BoardMember{})
	resp.OK(c, nil)
}

// ---- 敏感词管理 ----
func (h *AdminHandler) WordFilters(c *gin.Context) {
	var list []model.WordFilter
	h.DB.Order("id ASC").Find(&list)
	resp.OK(c, list)
}

func (h *AdminHandler) CreateWordFilter(c *gin.Context) {
	var req model.WordFilter
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数不对")
		return
	}
	if req.Word == "" {
		resp.ParamError(c, "敏感词不能为空")
		return
	}
	if req.Replace == "" {
		req.Replace = "***"
	}
	h.DB.Create(&req)
	resp.OK(c, req)
}

func (h *AdminHandler) UpdateWordFilter(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Word    string `json:"word"`
		Replace string `json:"replace"`
		Type    int    `json:"type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数不对")
		return
	}
	var wf model.WordFilter
	if err := h.DB.First(&wf, id).Error; err != nil {
		resp.NotFound(c, "敏感词不存在")
		return
	}
	updates := map[string]interface{}{"type": req.Type}
	if req.Word != "" {
		updates["word"] = req.Word
	}
	if req.Replace != "" {
		updates["replace"] = req.Replace
	}
	h.DB.Model(&wf).Updates(updates)
	resp.OK(c, wf)
}

func (h *AdminHandler) DeleteWordFilter(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.WordFilter{}, id)
	resp.OK(c, nil)
}

// ---- 帖子管理 ----
func (h *AdminHandler) UpdateThread(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		IsTop    *int `json:"is_top"`
		IsFine   *int `json:"is_fine"`
		IsHead   *int `json:"is_head"`
		IsLock   *int `json:"is_lock"`
		IsRecom  *int `json:"is_recom"`
		IsNotice *int `json:"is_notice"`
		IsActive *int `json:"is_active"`
		Audit    *int `json:"audit_status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数不对")
		return
	}
	updates := map[string]interface{}{}
	if req.IsTop != nil {
		updates["is_top"] = *req.IsTop
	}
	if req.IsFine != nil {
		updates["is_fine"] = *req.IsFine
	}
	if req.IsHead != nil {
		updates["is_head"] = *req.IsHead
	}
	if req.IsLock != nil {
		updates["is_lock"] = *req.IsLock
	}
	if req.IsRecom != nil {
		updates["is_recom"] = *req.IsRecom
	}
	if req.IsNotice != nil {
		updates["is_notice"] = *req.IsNotice
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if req.Audit != nil {
		updates["audit_status"] = *req.Audit
	}
	if len(updates) == 0 {
		resp.ParamError(c, "没有需要更新的字段")
		return
	}
	if err := h.DB.Model(&model.Thread{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		resp.NotFound(c, "帖子不存在")
		return
	}
	resp.OK(c, nil)
}

func (h *AdminHandler) DeleteThread(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.DB.Model(&model.Thread{}).Where("id = ?", id).Update("status", 0).Error; err != nil {
		resp.NotFound(c, "帖子不存在")
		return
	}
	resp.OK(c, nil)
}

func (h *AdminHandler) DeleteReply(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.DB.Model(&model.Reply{}).Where("id = ?", id).Update("status", 0).Error; err != nil {
		resp.NotFound(c, "回复不存在")
		return
	}
	resp.OK(c, nil)
}

// ---- 举报管理 ----

// Reports 举报列表（status: 全部/-1，0待处理，1已忽略，2已删内容，3已封人）
func (h *AdminHandler) Reports(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 50 {
		size = 10
	}
	q := h.DB.Model(&model.Report{})
	if s := c.Query("status"); s != "" && s != "-1" {
		q = q.Where("status = ?", s)
	}
	if t := c.Query("target_type"); t != "" && t != "all" {
		q = q.Where("target_type = ?", t)
	}
	var total int64
	q.Count(&total)
	var list []model.Report
	q.Preload("Reporter").Order("status ASC, id DESC").Offset((page - 1) * size).Limit(size).Find(&list)

	out := []gin.H{}
	for _, r := range list {
		item := gin.H{
			"id": r.ID, "reporter_id": r.ReporterID, "reporter": r.Reporter,
			"target_type": r.TargetType, "target_id": r.TargetID,
			"reason": r.Reason, "status": r.Status, "result": r.Result,
			"handled_at": r.HandledAt, "created_at": r.CreatedAt,
		}
		// 附带被举报内容快照与作者
		if r.TargetType == "thread" {
			var th model.Thread
			if h.DB.Preload("User").First(&th, r.TargetID).Error == nil {
				item["content"] = th.Title
				item["author"] = th.User
				item["content_status"] = th.Status
			}
		} else {
			var rp model.Reply
			if h.DB.Preload("User").First(&rp, r.TargetID).Error == nil {
				item["content"] = rp.Content
				item["author"] = rp.User
				item["content_status"] = rp.Status
				item["thread_id"] = rp.ThreadID
			}
		}
		out = append(out, item)
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// ReportHandle 处理举报：ignore忽略 / delete删内容 / ban封作者
func (h *AdminHandler) ReportHandle(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	uid := middleware.GetUID(c)
	var req struct {
		Action string `json:"action" binding:"required,oneof=ignore delete ban"`
		Result string `json:"result"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "处理动作有误")
		return
	}
	var r model.Report
	if err := h.DB.First(&r, id).Error; err != nil {
		resp.NotFound(c, "举报记录不存在")
		return
	}
	if r.Status != 0 {
		resp.ParamError(c, "该举报已处理过")
		return
	}
	status := 1
	result := req.Result
	switch req.Action {
	case "ignore":
		if result == "" {
			result = "核实后未发现违规，予以忽略"
		}
	case "delete":
		if r.TargetType == "thread" {
			h.DB.Model(&model.Thread{}).Where("id = ?", r.TargetID).Update("status", 0)
		} else {
			h.DB.Model(&model.Reply{}).Where("id = ?", r.TargetID).Update("status", 0)
		}
		status = 2
		if result == "" {
			result = "内容违规，已删除"
		}
	case "ban":
		var authorID uint
		if r.TargetType == "thread" {
			var th model.Thread
			if h.DB.First(&th, r.TargetID).Error == nil {
				authorID = th.UserID
			}
		} else {
			var rp model.Reply
			if h.DB.First(&rp, r.TargetID).Error == nil {
				authorID = rp.UserID
			}
		}
		if authorID == 0 {
			resp.ParamError(c, "被举报内容不存在，无法封禁")
			return
		}
		h.DB.Model(&model.User{}).Where("id = ?", authorID).Update("status", 0)
		status = 3
		if result == "" {
			result = "情节严重，已封禁发布者"
		}
	}
	now := time.Now()
	h.DB.Model(&r).Updates(map[string]interface{}{"status": status, "result": result, "handler_id": uid, "handled_at": &now})
	// 通知举报人处理结果
	if r.ReporterID > 0 {
		h.DB.Create(&model.Notification{UserID: r.ReporterID, Type: "system",
			Title: "举报处理结果", Content: "你的举报（编号" + strconv.FormatUint(uint64(r.ID), 10) + "）已处理：" + result})
	}
	resp.OK(c, gin.H{"status": status, "result": result})
}

// ---- 公告管理 ----
func (h *AdminHandler) Announcements(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 10
	}
	q := h.DB.Model(&model.Announcement{})
	var total int64
	q.Count(&total)
	if maxPage := int(total+int64(size)-1) / size; maxPage < 1 {
		page = 1
	} else if page > maxPage {
		page = maxPage
	}
	var list []model.Announcement
	q.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&list)
	resp.OK(c, gin.H{"list": list, "total": total, "page": page, "size": size})
}

type annReq struct {
	Type    string `json:"type" binding:"required,oneof=notice broadcast activity"`
	Title   string `json:"title" binding:"required,min=1,max=100"`
	Content string `json:"content" binding:"max=2000"`
	Status  *int   `json:"status"`
}

func (h *AdminHandler) CreateAnnouncement(c *gin.Context) {
	var req annReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "类型必须是 notice/broadcast/activity，标题必填")
		return
	}
	ann := model.Announcement{Type: req.Type, Title: req.Title, Content: req.Content}
	h.DB.Create(&ann)
	resp.OK(c, ann)
}

func (h *AdminHandler) UpdateAnnouncement(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req annReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "类型必须是 notice/broadcast/activity，标题必填")
		return
	}
	var ann model.Announcement
	if err := h.DB.First(&ann, id).Error; err != nil {
		resp.NotFound(c, "公告不存在")
		return
	}
	updates := map[string]interface{}{"type": req.Type, "title": req.Title, "content": req.Content}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	h.DB.Model(&ann).Updates(updates)
	resp.OK(c, ann)
}

func (h *AdminHandler) DeleteAnnouncement(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.Announcement{}, id)
	resp.OK(c, nil)
}

// ---- 角色权限管理 ----
func (h *AdminHandler) Roles(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 10
	}
	q := h.DB.Model(&model.Role{})
	var total int64
	q.Count(&total)
	if maxPage := int(total+int64(size)-1) / size; maxPage < 1 {
		page = 1
	} else if page > maxPage {
		page = maxPage
	}
	var roles []model.Role
	q.Preload("Permissions").Order("id ASC").Offset((page - 1) * size).Limit(size).Find(&roles)
	resp.OK(c, gin.H{"list": roles, "total": total, "page": page, "size": size})
}

func (h *AdminHandler) Permissions(c *gin.Context) {
	var perms []model.Permission
	h.DB.Find(&perms)
	resp.OK(c, perms)
}

type roleReq struct {
	Name   string `json:"name" binding:"required,min=1,max=30"`
	Code   string `json:"code" binding:"required,min=2,max=30"`
	Remark string `json:"remark" binding:"max=100"`
}

func (h *AdminHandler) CreateRole(c *gin.Context) {
	var req roleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "角色名/编码必填")
		return
	}
	var n int64
	h.DB.Model(&model.Role{}).Where("code = ?", req.Code).Count(&n)
	if n > 0 {
		resp.ParamError(c, "角色编码已存在")
		return
	}
	role := model.Role{Name: req.Name, Code: req.Code, Remark: req.Remark}
	h.DB.Create(&role)
	resp.OK(c, role)
}

func (h *AdminHandler) UpdateRolePerms(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		PermIDs []uint `json:"perm_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请提交 perm_ids 数组")
		return
	}
	var role model.Role
	if err := h.DB.First(&role, id).Error; err != nil {
		resp.NotFound(c, "角色不存在")
		return
	}
	var perms []model.Permission
	if len(req.PermIDs) > 0 {
		h.DB.Where("id IN ?", req.PermIDs).Find(&perms)
	}
	h.DB.Model(&role).Association("Permissions").Replace(&perms)
	resp.OK(c, nil)
}

func (h *AdminHandler) DeleteRole(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var role model.Role
	if err := h.DB.First(&role, id).Error; err != nil {
		resp.NotFound(c, "角色不存在")
		return
	}
	if role.Code == "super_admin" || role.Code == "member" {
		resp.ParamError(c, "内置角色不能删除")
		return
	}
	h.DB.Model(&role).Association("Permissions").Clear()
	h.DB.Model(&role).Association("Users").Clear()
	h.DB.Delete(&role)
	resp.OK(c, nil)
}

// ---- 帖子管理列表 ----
func (h *AdminHandler) Threads(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	word := c.Query("word")
	q := h.DB.Model(&model.Thread{}).Where("status = 1")
	if word != "" {
		q = q.Where("title LIKE ?", "%"+word+"%")
	}
	var total int64
	q.Count(&total)
	var list []model.Thread
	q.Preload("User").Preload("Board").Order("created_at DESC").Offset(offset).Limit(size).Find(&list)
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": list})
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
