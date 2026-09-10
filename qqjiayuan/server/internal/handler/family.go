package handler

import (
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/authutil"
	"qqjiayuan/server/pkg/resp"
)

// 家族分类（参考站家族类别）
var familyCategories = []string{"同城同乡", "青春校园", "打工生涯", "明星粉丝", "情感男女", "娱乐八卦", "军人风采", "舞文弄墨", "游戏动漫", "科技数码", "时尚生活", "其他"}

type FamilyHandler struct {
	DB     *gorm.DB
	Secret string // JWT 密钥，用于公开详情识别登录用户
}

// 创建家族的花费
const familyCreateCost = 500

// 家族列表：公开接口，附带当前用户的角色与成员数
func (h *FamilyHandler) List(c *gin.Context) {
	var users []model.Family
	q := h.DB.Preload("Owner").Where("status = 1")
	if cat := c.Query("category"); cat != "" {
		q = q.Where("category = ?", cat)
	}
	q.Order("battle_score DESC, id ASC").Find(&users)

	uid := uint(0)
	if v, ok := c.Get(middleware.CtxUID); ok {
		uid, _ = v.(uint)
	}
	out := make([]gin.H, 0, len(users))
	for _, f := range users {
		var count int64
		h.DB.Model(&model.FamilyMember{}).Where("family_id = ?", f.ID).Count(&count)
		f.Members = count
		if uid > 0 {
			var m model.FamilyMember
			if err := h.DB.Where("family_id = ? AND user_id = ?", f.ID, uid).First(&m).Error; err == nil {
				f.Role = m.Role
			}
		}
		out = append(out, gin.H{
			"id": f.ID, "name": f.Name, "slogan": f.Slogan, "description": f.Description,
			"category": f.Category, "owner_id": f.OwnerID, "owner": f.Owner, "members": f.Members,
			"tree_level": f.TreeLevel, "battle_score": f.BattleScore, "is_feature": f.IsFeature,
			"role": f.Role, "created_at": f.CreatedAt,
		})
	}
	resp.OK(c, out)
}

type createFamilyReq struct {
	Name        string `json:"name" binding:"required,min=2,max=30"`
	Slogan      string `json:"slogan"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

// 创建家族（提交申请，需管理员审核；通过后扣 500 金币正式成立）
func (h *FamilyHandler) Create(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req createFamilyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "家族名称需 2-30 个字")
		return
	}
	var exists int64
	h.DB.Model(&model.Family{}).Where("name = ? AND status IN (1,2)", req.Name).Count(&exists)
	if exists > 0 {
		resp.ParamError(c, "这个家族名已被占用（含待审核），换一个吧")
		return
	}
	var memberCount int64
	h.DB.Model(&model.FamilyMember{}).Where("user_id = ?", uid).Count(&memberCount)
	if memberCount > 0 {
		resp.ParamError(c, "你已经加入家族，先退出再建新的吧")
		return
	}
	var pending int64
	h.DB.Model(&model.Family{}).Where("owner_id = ? AND status = 2", uid).Count(&pending)
	if pending > 0 {
		resp.ParamError(c, "你已有一个家族正在审核中，请耐心等待")
		return
	}
	fam := model.Family{Name: req.Name, Slogan: req.Slogan, Description: req.Description,
		Category: req.Category, OwnerID: uid, TreeLevel: 1, Status: 2}
	if err := h.DB.Create(&fam).Error; err != nil {
		resp.ServerError(c, err)
		return
	}
	resp.OK(c, gin.H{"id": fam.ID, "pending": true, "msg": "申请已提交，等待管理员审核（通过后扣 500 金币）"})
}

// 我的家族
func (h *FamilyHandler) Mine(c *gin.Context) {
	uid := middleware.GetUID(c)
	var m model.FamilyMember
	if err := h.DB.Where("user_id = ?", uid).First(&m).Error; err != nil {
		resp.OK(c, nil)
		return
	}
	var fam model.Family
	if err := h.DB.Preload("Owner").First(&fam, m.FamilyID).Error; err != nil || fam.Status == 0 {
		resp.OK(c, nil)
		return
	}
	var count int64
	h.DB.Model(&model.FamilyMember{}).Where("family_id = ?", fam.ID).Count(&count)
	fam.Members = count
	fam.Role = m.Role
	resp.OK(c, gin.H{"id": fam.ID, "name": fam.Name, "slogan": fam.Slogan, "role": m.Role, "members": count,
		"exp": m.Exp, "title": familyTitle(m.Role, m.Exp)})
}

// 家族详情：公告 / 成员 / 我的角色 / 今日已签到（公开可看，登录可识别角色）
func (h *FamilyHandler) Detail(c *gin.Context) {
	fam, ok := h.familyOf(c)
	if !ok {
		return
	}
	uid := h.optUID(c)
	var members []model.FamilyMember
	h.DB.Preload("User").Where("family_id = ?", fam.ID).Order("CASE role WHEN 'owner' THEN 0 WHEN 'admin' THEN 1 ELSE 2 END, created_at ASC").Find(&members)

	// 我是否是成员 / 角色
	myRole := ""
	for _, m := range members {
		if m.UserID == uid {
			myRole = m.Role
			break
		}
	}

	// 今日家族签到状态
	today := todayStr()
	var signed int64
	h.DB.Model(&model.FamilySignIn{}).Where("family_id = ? AND user_id = ? AND sign_date = ? AND type = 'sign'", fam.ID, uid, today).Count(&signed)

	// 今日已签到人数（守护树热度参考）
	var treeToday int64
	h.DB.Model(&model.FamilySignIn{}).Where("family_id = ? AND sign_date = ? AND type = 'sign'", fam.ID, today).Count(&treeToday)

	// 家族在线家人（10 分钟内活跃）
	var online int64
	h.DB.Model(&model.FamilyMember{}).
		Joins("JOIN users ON users.id = family_members.user_id").
		Where("family_members.family_id = ? AND users.last_active_at > ?", fam.ID, time.Now().Add(-10*time.Minute)).
		Count(&online)

	// 家族专属论坛板块（不存在则幂等创建）
	forumBoard := ensureFamilyForumBoard(h.DB, &fam)

	// 我是否收藏了本家族
	var favored bool
	if uid > 0 {
		var n int64
		h.DB.Model(&model.FamilyFavorite{}).Where("user_id = ? AND family_id = ?", uid, fam.ID).Count(&n)
		favored = n > 0
	}

	// 我的贡献值与职称
	myExp, myTitle := 0, ""
	if myRole != "" {
		for _, m := range members {
			if m.UserID == uid {
				myExp = m.Exp
				break
			}
		}
		myTitle = familyTitle(myRole, myExp)
	}

	// 家族访客：登录用户访问即记录（每人每日一条），统计今日访客数（对齐诺哈「访客：今天N人」）
	visitsToday := int64(0)
	if uid > 0 {
		var exist model.FamilyVisit
		if err := h.DB.Where("family_id = ? AND user_id = ? AND day = ?", fam.ID, uid, today).First(&exist).Error; err != nil {
			h.DB.Create(&model.FamilyVisit{FamilyID: fam.ID, UserID: uid, Day: today})
		}
		h.DB.Model(&model.FamilyVisit{}).Where("family_id = ? AND day = ?", fam.ID, today).Count(&visitsToday)
	}

	out := gin.H{
		"id": fam.ID, "name": fam.Name, "slogan": fam.Slogan, "description": fam.Description,
		"announcement": fam.Announcement, "owner_id": fam.OwnerID, "owner": fam.Owner,
		"tree_level": fam.TreeLevel, "tree_exp": fam.TreeExp, "battle_score": fam.BattleScore,
		"war_points": fam.WarPoints,
		"members": members, "my_role": myRole, "member_count": len(members),
		"my_exp": myExp, "my_title": myTitle,
		"signed_today": signed > 0, "tree_today": treeToday, "online": online, "created_at": fam.CreatedAt,
		"forum_board_id": forumBoard.ID, "favored": favored, "visits_today": visitsToday,
	}
	resp.OK(c, out)
}

// 收藏/取消收藏家族（对齐诺哈 family_favor.asp）
func (h *FamilyHandler) Favorite(c *gin.Context) {
	uid := middleware.GetUID(c)
	fam, ok := h.familyOf(c)
	if !ok {
		return
	}
	var exist model.FamilyFavorite
	if err := h.DB.Where("user_id = ? AND family_id = ?", uid, fam.ID).First(&exist).Error; err == nil {
		h.DB.Delete(&exist)
		resp.OK(c, gin.H{"favored": false, "msg": "已取消收藏"})
		return
	}
	h.DB.Create(&model.FamilyFavorite{UserID: uid, FamilyID: fam.ID})
	resp.OK(c, gin.H{"favored": true, "msg": "已收藏该家族"})
}

// 我的收藏家族列表
func (h *FamilyHandler) MyFavorites(c *gin.Context) {
	uid := middleware.GetUID(c)
	var rows []model.FamilyFavorite
	h.DB.Where("user_id = ?", uid).Order("id DESC").Find(&rows)
	ids := []uint{}
	for _, r := range rows {
		ids = append(ids, r.FamilyID)
	}
	out := []gin.H{}
	if len(ids) > 0 {
		var fams []model.Family
		h.DB.Preload("Owner").Where("id IN ?", ids).Find(&fams)
		for _, f := range fams {
			out = append(out, gin.H{"id": f.ID, "name": f.Name, "slogan": f.Slogan, "category": f.Category,
				"owner": f.Owner.Nickname, "tree_level": f.TreeLevel, "battle_score": f.BattleScore})
		}
	}
	resp.OK(c, out)
}

// 族长移除成员（对齐诺哈 family_func_member_remove.asp）
func (h *FamilyHandler) RemoveMember(c *gin.Context) {
	uid := middleware.GetUID(c)
	fam, ok := h.familyOf(c)
	if !ok {
		return
	}
	if fam.OwnerID != uid {
		resp.Forbidden(c, "只有族长才能管理成员")
		return
	}
	mid, _ := strconv.Atoi(c.Param("userId"))
	if mid == int(uid) {
		resp.ParamError(c, "族长不能移除自己，请解散家族")
		return
	}
	res := h.DB.Where("family_id = ? AND user_id = ?", fam.ID, mid).Delete(&model.FamilyMember{})
	if res.RowsAffected == 0 {
		resp.NotFound(c, "该成员不在本家族")
		return
	}
	h.act(fam.ID, uid, "将 %s 移出了家族", func() string {
		var u model.User
		h.DB.First(&u, mid)
		return u.Nickname
	}())
	resp.OK(c, "已移除该成员")
}

// 加入家族
func (h *FamilyHandler) Join(c *gin.Context) {
	uid := middleware.GetUID(c)
	fam, ok := h.familyOf(c)
	if !ok {
		return
	}
	var count int64
	h.DB.Model(&model.FamilyMember{}).Where("user_id = ?", uid).Count(&count)
	if count > 0 {
		resp.ParamError(c, "你已经加入家族了，先退出再加入本家族")
		return
	}
	var inThis int64
	h.DB.Model(&model.FamilyMember{}).Where("family_id = ? AND user_id = ?", fam.ID, uid).Count(&inThis)
	if inThis > 0 {
		resp.OK(c, gin.H{"joined": true})
		return
	}
	h.DB.Create(&model.FamilyMember{FamilyID: fam.ID, UserID: uid})
	ensureFamilyForumBoard(h.DB, &fam)
	h.act(fam.ID, uid, "加入了家族《%s》", fam.Name)
	resp.OK(c, gin.H{"joined": true})
}

// 退出家族
func (h *FamilyHandler) Leave(c *gin.Context) {
	uid := middleware.GetUID(c)
	fam, ok := h.familyOf(c)
	if !ok {
		return
	}
	var m model.FamilyMember
	if err := h.DB.Where("family_id = ? AND user_id = ?", fam.ID, uid).First(&m).Error; err != nil {
		resp.ParamError(c, "你不在这个家族")
		return
	}
	if m.Role == "owner" {
		resp.ParamError(c, "族长不能退出，可解散家族或移交族长")
		return
	}
	h.DB.Delete(&m)
	h.act(fam.ID, uid, "退出了家族")
	resp.OK(c, nil)
}

type familyAnnReq struct {
	Announcement string `json:"announcement" binding:"max=500"`
}

// 更新家族公告（族长/管理）
func (h *FamilyHandler) UpdateAnn(c *gin.Context) {
	uid := middleware.GetUID(c)
	fam, ok := h.familyOf(c)
	if !ok {
		return
	}
	if fam.OwnerID != uid {
		resp.Forbidden(c, "只有族长能修改公告")
		return
	}
	var req familyAnnReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "公告内容过长（500字以内）")
		return
	}
	h.DB.Model(&fam).Update("announcement", req.Announcement)
	h.act(fam.ID, uid, "更新了家族公告")
	resp.OK(c, nil)
}

// 家族签到：每日一次，+20 经验 +5 金币 + 家族贡献
func (h *FamilyHandler) SignIn(c *gin.Context) {
	uid := middleware.GetUID(c)
	fam, ok := h.familyOf(c)
	if !ok {
		return
	}
	if !h.isMember(fam.ID, uid) {
		resp.Forbidden(c, "你还不是本家族成员")
		return
	}
	today := todayStr()
	var exist int64
	h.DB.Model(&model.FamilySignIn{}).Where("family_id = ? AND user_id = ? AND sign_date = ? AND type = 'sign'", fam.ID, uid, today).Count(&exist)
	if exist > 0 {
		resp.ParamError(c, "今天已在家族签到过了")
		return
	}
	if err := h.DB.Create(&model.FamilySignIn{FamilyID: fam.ID, UserID: uid, SignDate: today, Type: "sign"}).Error; err != nil {
		resp.ServerError(c, err)
		return
	}
	addExpAndCoins(h.DB, uid, 20, 5, 1, "fsign", "家族签到")
	h.DB.Model(&model.FamilyMember{}).Where("family_id = ? AND user_id = ?", fam.ID, uid).
		Update("exp", gorm.Expr("exp + ?", 20))
	h.act(fam.ID, uid, "在家族签到")
	resp.OK(c, gin.H{"exp": 20, "coins": 5})
}

// 守护树：抚摸/拥抱（每日一次），+30 成长值，每满 100 升 1 级
func (h *FamilyHandler) Tree(c *gin.Context) {
	uid := middleware.GetUID(c)
	fam, ok := h.familyOf(c)
	if !ok {
		return
	}
	if !h.isMember(fam.ID, uid) {
		resp.Forbidden(c, "你还不是本家族成员")
		return
	}
	today := todayStr()
	var exist int64
	h.DB.Model(&model.FamilySignIn{}).Where("family_id = ? AND user_id = ? AND sign_date = ? AND type = 'tree'", fam.ID, uid, today).Count(&exist)
	if exist > 0 {
		resp.ParamError(c, "今天已经抚摸过守护树啦，明天再来吧")
		return
	}
	h.DB.Create(&model.FamilySignIn{FamilyID: fam.ID, UserID: uid, SignDate: today, Type: "tree"})
	newExp := fam.TreeExp + 30
	newLevel := newExp/100 + 1
	leveled := newLevel > fam.TreeLevel
	h.DB.Model(&fam).Updates(map[string]interface{}{"tree_exp": newExp, "tree_level": newLevel})
	addExpAndCoins(h.DB, uid, 10, 3, 1, "ftree", "家族守护树")
	h.act(fam.ID, uid, "抚摸/拥抱了守护树")
	resp.OK(c, gin.H{"tree_exp": newExp, "tree_level": newLevel, "leveled": leveled})
}

// 家族乐斗：随机匹配一支家族比拼，胜 +15 积分，负 +5 积分（参与奖）
func (h *FamilyHandler) Battle(c *gin.Context) {
	uid := middleware.GetUID(c)
	fam, ok := h.familyOf(c)
	if !ok {
		return
	}
	if !h.isMember(fam.ID, uid) {
		resp.Forbidden(c, "你还不是本家族成员")
		return
	}
	opp := model.Family{}
	h.DB.Where("status = 1 AND id <> ?", fam.ID).
		Order("RAND()").First(&opp)
	oppName := "神秘家族"
	if opp.ID > 0 {
		oppName = opp.Name
	}
	win := rand.Intn(100) < 55
	gain := 5
	if win {
		gain = 15
	}
	newScore := fam.BattleScore + gain
	h.DB.Model(&fam).Update("battle_score", newScore)
	h.act(fam.ID, uid, "参加家族乐斗，%s对手《%s》", familyBattleTitle(win), oppName)
	resp.OK(c, gin.H{"win": win, "opponent": oppName, "gain": gain, "score": newScore})
}

// ---- 家族乐斗（参考站 /bbs/ld/index：战斗力/功勋值/体力值/菜鸟高手乱斗）----

// familyTitle 按贡献值算职称（参考站「初级家人」等）
func familyTitle(role string, exp int) string {
	if role == "owner" {
		return "族长"
	}
	switch {
	case exp >= 1000:
		return "元老家人"
	case exp >= 500:
		return "骨干家人"
	case exp >= 200:
		return "高级家人"
	case exp >= 50:
		return "中级家人"
	}
	return "初级家人"
}

// ldUserOf 获取或创建乐斗个人数据；跨天重置今日次数与体力
func (h *FamilyHandler) ldUserOf(uid uint) model.FamilyLdUser {
	var ld model.FamilyLdUser
	if err := h.DB.Where("user_id = ?", uid).First(&ld).Error; err != nil {
		ld = model.FamilyLdUser{UserID: uid, Fight: 10, Merit: 0, Stamina: 10}
		h.DB.Create(&ld)
	}
	today := todayStr()
	if ld.LdDate != today {
		ld.LdDate = today
		ld.LdCount = 0
		ld.Stamina = 10
		h.DB.Model(&ld).Updates(map[string]interface{}{"ld_date": today, "ld_count": 0, "stamina": 10})
	}
	return ld
}

// 乐斗首页：我的战斗数据 + 三区对手 + 乐斗动态
func (h *FamilyHandler) Ld(c *gin.Context) {
	uid := middleware.GetUID(c)
	fam, ok := h.familyOf(c)
	if !ok {
		return
	}
	if !h.isMember(fam.ID, uid) {
		resp.Forbidden(c, "你还不是本家族成员")
		return
	}
	ld := h.ldUserOf(uid)
	bt, _ := strconv.Atoi(c.Query("bt"))
	if bt < 1 || bt > 3 {
		bt = 1
	}
	// 菜鸟=本族低贡献家人；高手=本族高贡献家人；乱斗=全站随机友友
	type oppRow struct {
		UserID   uint   `json:"user_id"`
		Nickname string `json:"nickname"`
		Color    string `json:"color"`
		Fight    int    `json:"fight"`
		Level    int    `json:"level"`
	}
	opps := []oppRow{}
	if bt <= 2 {
		var members []model.FamilyMember
		q := h.DB.Preload("User").Where("family_id = ? AND user_id <> ?", fam.ID, uid)
		if bt == 1 {
			q = q.Where("exp < 50")
		} else {
			q = q.Where("exp >= 50")
		}
		q.Order("RAND()").Limit(5).Find(&members)
		for _, m := range members {
			if m.User == nil {
				continue
			}
			old := h.ldUserOf(m.UserID)
			opps = append(opps, oppRow{UserID: m.UserID, Nickname: m.User.Nickname, Color: m.User.Color, Fight: old.Fight, Level: m.User.Level})
		}
	} else {
		var users []model.User
		h.DB.Where("id <> ?", uid).Order("RAND()").Limit(5).Find(&users)
		for _, u := range users {
			old := h.ldUserOf(u.ID)
			opps = append(opps, oppRow{UserID: u.ID, Nickname: u.Nickname, Color: u.Color, Fight: old.Fight, Level: u.Level})
		}
	}
	var logs []model.FamilyLdLog
	h.DB.Preload("User").Where("family_id = ?", fam.ID).Order("created_at DESC").Limit(5).Find(&logs)
	remain := 20 - ld.LdCount
	if remain < 0 {
		remain = 0
	}
	resp.OK(c, gin.H{
		"fight": ld.Fight, "merit": ld.Merit, "stamina": ld.Stamina,
		"ld_count": ld.LdCount, "ld_limit": 20, "remain": remain,
		"win_count": ld.WinCount, "lose_count": ld.LoseCount,
		"bt": bt, "opponents": opps, "logs": logs,
	})
}

// 斗一斗：与指定家人乐斗，消耗 1 体力，每日上限 20 次
func (h *FamilyHandler) LdPk(c *gin.Context) {
	uid := middleware.GetUID(c)
	fam, ok := h.familyOf(c)
	if !ok {
		return
	}
	if !h.isMember(fam.ID, uid) {
		resp.Forbidden(c, "你还不是本家族成员")
		return
	}
	oid, _ := strconv.Atoi(c.Param("userId"))
	if oid <= 0 || oid == int(uid) {
		resp.ParamError(c, "对手不合法")
		return
	}
	ld := h.ldUserOf(uid)
	if ld.Stamina <= 0 {
		resp.ParamError(c, "体力值不足，先吃个果实补充体力吧")
		return
	}
	if ld.LdCount >= 20 {
		resp.ParamError(c, "今日乐斗已达 20 次上限，明天再来吧")
		return
	}
	var ou model.User
	if err := h.DB.First(&ou, oid).Error; err != nil {
		resp.NotFound(c, "对手不存在")
		return
	}
	oldd := h.ldUserOf(uint(oid))
	// 战斗力 + 临时手气决定胜负（高手区对手更强）
	win := ld.Fight+rand.Intn(30) > oldd.Fight+rand.Intn(30)
	ld.Stamina--
	ld.LdCount++
	gain := gin.H{}
	if win {
		ld.Fight++
		ld.Merit += 2
		ld.WinCount++
		h.DB.Model(&model.FamilyMember{}).Where("family_id = ? AND user_id = ?", fam.ID, uid).
			Update("exp", gorm.Expr("exp + ?", 5))
		addExpAndCoins(h.DB, uid, 5, 3, 1, "ldwin", "家族乐斗获胜")
		h.ldLog(fam.ID, uid, "在家族乐斗中战胜了【%s】", ou.Nickname)
		h.act(fam.ID, uid, "在家族乐斗中战胜了【%s】", ou.Nickname)
		gain = gin.H{"fight": ld.Fight, "merit": ld.Merit, "exp": 5, "coins": 3}
	} else {
		ld.Merit--
		ld.LoseCount++
		addExpAndCoins(h.DB, uid, 2, 1, 0, "ldlose", "家族乐斗参与奖")
		h.ldLog(fam.ID, uid, "挑战%s惜败，再接再厉！", ou.Nickname)
		gain = gin.H{"merit": ld.Merit, "exp": 2, "coins": 1}
	}
	h.DB.Model(&ld).Updates(map[string]interface{}{
		"fight": ld.Fight, "merit": ld.Merit, "stamina": ld.Stamina,
		"ld_count": ld.LdCount, "win_count": ld.WinCount, "lose_count": ld.LoseCount, "ld_date": ld.LdDate,
	})
	resp.OK(c, gin.H{"win": win, "opponent": ou.Nickname, "stamina": ld.Stamina, "count": ld.LdCount, "gain": gain})
}

// 吃果实：每日一次，体力 +5
func (h *FamilyHandler) LdFruit(c *gin.Context) {
	uid := middleware.GetUID(c)
	ld := h.ldUserOf(uid)
	if ld.Stamina >= 10 {
		resp.ParamError(c, "体力满满，不用吃果实啦")
		return
	}
	newStamina := ld.Stamina + 5
	if newStamina > 10 {
		newStamina = 10
	}
	h.DB.Model(&ld).Update("stamina", newStamina)
	resp.OK(c, gin.H{"stamina": newStamina, "msg": "吃完果实体力恢复了"})
}

func (h *FamilyHandler) ldLog(familyID, userID uint, format string, args ...interface{}) {
	h.DB.Create(&model.FamilyLdLog{FamilyID: familyID, UserID: userID, Content: fmt.Sprintf(format, args...)})
}

// ---- 家族族斗（参考站 /bbs/zd/index：今日战局/生命力/攻击对象/世界喊话/荣誉榜）----

// warLifeOf 获取或创建今日生命力（每日重置 5 点）
func (h *FamilyHandler) warLifeOf(uid uint) model.FamilyWarLife {
	var wl model.FamilyWarLife
	if err := h.DB.Where("user_id = ?", uid).First(&wl).Error; err != nil {
		wl = model.FamilyWarLife{UserID: uid, Life: 5, WarDate: todayStr()}
		h.DB.Create(&wl)
	}
	today := todayStr()
	if wl.WarDate != today {
		wl.Life = 5
		wl.WarDate = today
		wl.Bought = 0
		h.DB.Model(&wl).Updates(map[string]interface{}{"life": 5, "war_date": today, "bought": 0})
	}
	return wl
}

// todayBattleOf 获取或创建今日战局（随机匹配一支敌对家族）
func (h *FamilyHandler) todayBattleOf(fam *model.Family) model.FamilyWarBattle {
	today := todayStr()
	var b model.FamilyWarBattle
	if err := h.DB.Where("war_date = ? AND family_id = ?", today, fam.ID).First(&b).Error; err == nil {
		return b
	}
	enemy := model.Family{}
	h.DB.Where("status = 1 AND id <> ?", fam.ID).Order("RAND()").First(&enemy)
	if enemy.ID == 0 {
		enemy.Name = "神秘家族"
	}
	b = model.FamilyWarBattle{WarDate: today, FamilyID: fam.ID, EnemyID: enemy.ID,
		EnemyName: enemy.Name, MyScore: fam.WarPoints, EnemyScore: enemy.WarPoints}
	h.DB.Create(&b)
	return b
}

// 族斗首页：今日战局 + 生命力 + 攻击对象 + 喊话 + 荣誉榜
func (h *FamilyHandler) War(c *gin.Context) {
	uid := middleware.GetUID(c)
	fam, ok := h.familyOf(c)
	if !ok {
		return
	}
	if !h.isMember(fam.ID, uid) {
		resp.Forbidden(c, "你还不是本家族成员")
		return
	}
	battle := h.todayBattleOf(&fam)
	wl := h.warLifeOf(uid)
	ld := h.ldUserOf(uid)

	// 攻击对象：敌方家族成员随机 3 名
	var enemies []model.FamilyMember
	h.DB.Preload("User").Where("family_id = ?", battle.EnemyID).Order("RAND()").Limit(3).Find(&enemies)
	opps := []gin.H{}
	for _, m := range enemies {
		if m.User == nil {
			continue
		}
		opps = append(opps, gin.H{"user_id": m.UserID, "nickname": m.User.Nickname, "color": m.User.Color})
	}

	// 族斗喊话（yid=1 家族对话 / 默认全部含私聊）
	yid := c.Query("yid")
	cq := h.DB.Preload("User").Where("family_id = ?", fam.ID)
	if yid == "1" {
		cq = cq.Where("type = 'chat'")
	}
	var chats []model.FamilyWarChat
	cq.Order("created_at DESC").Limit(10).Find(&chats)

	// 族斗荣誉榜（前3）+ 我的排名
	var tops []model.Family
	h.DB.Where("status = 1").Order("war_points DESC, id ASC").Limit(3).Find(&tops)
	var moreCount int64
	h.DB.Model(&model.Family{}).Where("status = 1 AND war_points > ?", fam.WarPoints).Count(&moreCount)

	// 昨日战况
	yest := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	var yb model.FamilyWarBattle
	yesterday := gin.H{}
	if err := h.DB.Where("war_date = ? AND family_id = ?", yest, fam.ID).First(&yb).Error; err == nil {
		yesterday = gin.H{"enemy": yb.EnemyName, "win": yb.MyScore >= yb.EnemyScore}
	}

	// 个人功勋榜（本族前5）
	var merits []model.FamilyMember
	h.DB.Preload("User").Where("family_id = ?", fam.ID).Order("exp DESC").Limit(5).Find(&merits)

	// 今日我的攻击次数（动态条数近似）与生命力展示
	myName := fam.Name
	resp.OK(c, gin.H{
		"my_name": myName, "enemy_id": battle.EnemyID, "enemy_name": battle.EnemyName,
		"my_score": battle.MyScore, "enemy_score": battle.EnemyScore,
		"fight": ld.Fight, "merit": ld.Merit, "life": wl.Life,
		"opponents": opps, "chats": chats,
		"tops": tops, "my_rank": int(moreCount) + 1,
		"yesterday": yesterday,
		"merit_top": merits,
	})
}

// 族斗攻击：生命力 -1，胜负影响族斗荣誉点与功勋值
func (h *FamilyHandler) WarPk(c *gin.Context) {
	uid := middleware.GetUID(c)
	fam, ok := h.familyOf(c)
	if !ok {
		return
	}
	if !h.isMember(fam.ID, uid) {
		resp.Forbidden(c, "你还不是本家族成员")
		return
	}
	oid, _ := strconv.Atoi(c.Param("userId"))
	if oid < 0 || oid == int(uid) {
		resp.ParamError(c, "攻击对象不合法")
		return
	}
	// oid=0 表示随机挑选一名敌方家族成员
	if oid == 0 {
		var em model.FamilyMember
		b0 := h.todayBattleOf(&fam)
		if err := h.DB.Where("family_id = ?", b0.EnemyID).Order("RAND()").First(&em).Error; err != nil {
			resp.ParamError(c, "敌方家族暂无成员可攻击")
			return
		}
		oid = int(em.UserID)
	}
	wl := h.warLifeOf(uid)
	if wl.Life <= 0 {
		resp.ParamError(c, "今日生命力已耗尽，可购买生命力继续战斗")
		return
	}
	var ou model.User
	if err := h.DB.First(&ou, oid).Error; err != nil {
		resp.NotFound(c, "攻击对象不存在")
		return
	}
	ld := h.ldUserOf(uid)
	win := rand.Intn(100) < 55+ld.Fight/10
	wl.Life--
	h.DB.Model(&wl).Update("life", wl.Life)
	gain := gin.H{}
	newScore := fam.WarPoints
	if win {
		points := 1 + rand.Intn(5)
		ld.Merit += 2
		newScore = fam.WarPoints + points
		h.DB.Model(&fam).Update("war_points", newScore)
		h.DB.Model(&model.FamilyWarBattle{}).Where("war_date = ? AND family_id = ?", todayStr(), fam.ID).
			Update("my_score", newScore)
		addExpAndCoins(h.DB, uid, 5, 2, 1, "warwin", "族斗获胜")
		h.ldLog(fam.ID, uid, "挑战%s大获全胜，高奏凯歌！", ou.Nickname)
		h.act(fam.ID, uid, "挑战%s大获全胜，高奏凯歌！", ou.Nickname)
		gain = gin.H{"points": points, "merit": ld.Merit, "exp": 5, "coins": 2}
	} else {
		ld.Merit--
		h.ldLog(fam.ID, uid, "挑战%s，惜败而归！", ou.Nickname)
		gain = gin.H{"merit": ld.Merit}
	}
	h.DB.Model(&ld).Updates(map[string]interface{}{"merit": ld.Merit})
	resp.OK(c, gin.H{"win": win, "opponent": ou.Nickname, "life": wl.Life,
		"war_points": newScore, "gain": gain})
}

// 购买生命力：100 G币 = 1 点，每日限购 3 次
func (h *FamilyHandler) WarLife(c *gin.Context) {
	uid := middleware.GetUID(c)
	wl := h.warLifeOf(uid)
	if wl.Life >= 5 {
		resp.ParamError(c, "生命力满格，无需购买")
		return
	}
	if wl.Bought >= 3 {
		resp.ParamError(c, "今日已限购 3 次生命力")
		return
	}
	var u model.User
	h.DB.First(&u, uid)
	if u.Coins < 100 {
		resp.ParamError(c, "G币不足（需 100 G币）")
		return
	}
	addExpAndCoins(h.DB, uid, 0, -100, 0, "warlife", "购买族斗生命力")
	newLife := wl.Life + 1
	h.DB.Model(&wl).Updates(map[string]interface{}{"life": newLife, "bought": wl.Bought + 1})
	resp.OK(c, gin.H{"life": newLife, "msg": "购买成功，生命力 +1"})
}

type warChatReq struct {
	Content string `json:"content" binding:"required,max=120"`
	Type    string `json:"type"`
}

// 族斗喊话：私聊/对话，每次扣 1000 G币
func (h *FamilyHandler) WarChat(c *gin.Context) {
	uid := middleware.GetUID(c)
	fam, ok := h.familyOf(c)
	if !ok {
		return
	}
	if !h.isMember(fam.ID, uid) {
		resp.Forbidden(c, "你还不是本家族成员")
		return
	}
	var req warChatReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "说点什么吧（120字以内）")
		return
	}
	ctype := "chat"
	if req.Type == "private" {
		ctype = "private"
	}
	var u model.User
	h.DB.First(&u, uid)
	if u.Coins < 1000 {
		resp.ParamError(c, "G币不足（每次发言需 1000 G币）")
		return
	}
	addExpAndCoins(h.DB, uid, 1, -1000, 0, "warchat", "族斗喊话")
	h.DB.Create(&model.FamilyWarChat{FamilyID: fam.ID, UserID: uid, Type: ctype, Content: req.Content})
	resp.OK(c, gin.H{"msg": "喊话成功"})
}

// 家族类别（含各类别家族数，参考站「家族类别」）
func (h *FamilyHandler) Categories(c *gin.Context) {
	type catRow struct {
		Name  string `json:"name"`
		Count int64  `json:"count"`
	}
	out := make([]catRow, 0, len(familyCategories))
	for _, ct := range familyCategories {
		var n int64
		h.DB.Model(&model.Family{}).Where("status = 1 AND category = ?", ct).Count(&n)
		out = append(out, catRow{Name: ct, Count: n})
	}
	resp.OK(c, out)
}

// 搜索家族（按名称模糊 / 家族ID精确），参考站 family_search.html
func (h *FamilyHandler) Search(c *gin.Context) {
	wd := strings.TrimSpace(c.Query("wd"))
	out := []gin.H{}
	if wd != "" {
		q := h.DB.Preload("Owner").Where("status = 1").Where("name LIKE ?", "%"+wd+"%")
		if id, err := strconv.Atoi(wd); err == nil {
			q = h.DB.Preload("Owner").Where("status = 1").Where("name LIKE ? OR id = ?", "%"+wd+"%", id)
		}
		var fams []model.Family
		q.Order("battle_score DESC").Limit(50).Find(&fams)
		for _, f := range fams {
			var count int64
			h.DB.Model(&model.FamilyMember{}).Where("family_id = ?", f.ID).Count(&count)
			out = append(out, gin.H{"id": f.ID, "name": f.Name, "category": f.Category,
				"members": count, "owner": f.Owner, "owner_id": f.OwnerID})
		}
	}
	resp.OK(c, out)
}

// 家族排行榜：bt=1规模(成员) 2活跃(今日签到) 3人气(乐斗积分) 4等级(守护树)
func (h *FamilyHandler) Top(c *gin.Context) {
	bt, _ := strconv.Atoi(c.Query("bt"))
	if bt < 1 || bt > 4 {
		bt = 1
	}
	var fams []model.Family
	h.DB.Where("status = 1").Find(&fams)
	today := todayStr()
	out := []gin.H{}
	for _, f := range fams {
		var mc, ac int64
		h.DB.Model(&model.FamilyMember{}).Where("family_id = ?", f.ID).Count(&mc)
		h.DB.Model(&model.FamilySignIn{}).Where("family_id = ? AND sign_date = ? AND type = 'sign'", f.ID, today).Count(&ac)
		value := mc
		switch bt {
		case 2:
			value = ac
		case 3:
			value = int64(f.BattleScore)
		case 4:
			value = int64(f.TreeLevel)*1000 + int64(f.TreeExp)
		}
		out = append(out, gin.H{"id": f.ID, "name": f.Name, "members": mc, "active": ac,
			"score": f.BattleScore, "level": f.TreeLevel, "value": value})
	}
	sort.SliceStable(out, func(i, j int) bool {
		vi, _ := out[i]["value"].(int64)
		vj, _ := out[j]["value"].(int64)
		return vi > vj
	})
	if len(out) > 50 {
		out = out[:50]
	}
	resp.OK(c, out)
}

// ---- 家族论坛（家族专属板块，仅成员可发帖回帖，浏览公开）----

// familyForumBoard 家族专属论坛板块：家族大厅分区下「家族·家族名」
func familyForumBoard(db *gorm.DB, famName string) model.Board {
	var b model.Board
	var hall model.Board
	db.Where("parent_id = 0 AND name = ?", "家族大厅").First(&hall)
	if hall.ID == 0 {
		return b
	}
	db.Where("parent_id = ? AND name = ?", hall.ID, "家族·"+famName).First(&b)
	return b
}

// ensureFamilyForumBoard 幂等创建家族论坛板块
func ensureFamilyForumBoard(db *gorm.DB, fam *model.Family) model.Board {
	var hall model.Board
	db.Where("parent_id = 0 AND name = ?", "家族大厅").First(&hall)
	if hall.ID == 0 {
		return model.Board{}
	}
	var b model.Board
	db.Where("parent_id = ? AND name = ?", hall.ID, "家族·"+fam.Name).First(&b)
	if b.ID == 0 {
		b = model.Board{ParentID: hall.ID, Name: "家族·" + fam.Name, Description: fam.Name + " 家族论坛"}
		db.Create(&b)
	}
	return b
}

// isFamilyMember（包级）判断用户是否家族成员
func isFamilyMember(db *gorm.DB, familyID, userID uint) bool {
	var n int64
	db.Model(&model.FamilyMember{}).Where("family_id = ? AND user_id = ?", familyID, userID).Count(&n)
	return n > 0
}

// familyBoardOwner 判断板块是否家族专属论坛（「家族·xxx」），返回家族ID
func familyBoardOwner(db *gorm.DB, board model.Board) (uint, bool) {
	name := board.Name
	if !strings.HasPrefix(name, "家族·") {
		return 0, false
	}
	famName := strings.TrimPrefix(name, "家族·")
	var fam model.Family
	if err := db.Where("name = ? AND status = 1", famName).First(&fam).Error; err != nil {
		return 0, false
	}
	return fam.ID, true
}

// 家族论坛帖子列表（支持 filter=fine sort=new 分页）
func (h *FamilyHandler) Forum(c *gin.Context) {
	fam, ok := h.familyOf(c)
	if !ok {
		return
	}
	page, _, _ := pageOf(c, 10)
	board := familyForumBoard(h.DB, fam.Name)
	if board.ID == 0 {
		resp.OK(c, gin.H{"board": nil, "total": 0, "page": 1, "size": 10, "list": []gin.H{}})
		return
	}
	q := h.DB.Model(&model.Thread{}).Where("board_id = ? AND status = 1", board.ID)
	if c.Query("filter") == "fine" {
		q = q.Where("is_fine = 1")
	}
	order := "is_top DESC, IFNULL(last_reply_at, created_at) DESC"
	if c.Query("sort") == "new" {
		order = "is_top DESC, created_at DESC"
	}
	var total int64
	q.Count(&total)
	if maxPage := int(total+9) / 10; maxPage < 1 {
		page = 1
	} else if page > maxPage {
		page = maxPage
	}
	var threads []model.Thread
	q.Preload("User").Preload("User.Badges").
		Order(order).Offset((page - 1) * 10).Limit(10).Find(&threads)
	resp.OK(c, gin.H{"board": board, "total": total, "page": page, "size": 10, "list": threads})
}

// 家族热点：家族论坛最新 5 帖（家族主页展示）
func (h *FamilyHandler) Hot(c *gin.Context) {
	fam, ok := h.familyOf(c)
	if !ok {
		return
	}
	board := familyForumBoard(h.DB, fam.Name)
	out := []gin.H{}
	if board.ID > 0 {
		var threads []model.Thread
		h.DB.Preload("User").Where("board_id = ? AND status = 1", board.ID).
			Order("is_top DESC, IFNULL(last_reply_at, created_at) DESC").Limit(5).Find(&threads)
		for _, t := range threads {
			tag := "分享"
			if t.IsTop == 1 {
				tag = "公告"
			} else if t.IsFine == 1 {
				tag = "精华"
			}
			out = append(out, gin.H{"id": t.ID, "title": t.Title, "tag": tag, "view_count": t.ViewCount})
		}
	}
	resp.OK(c, out)
}

// ---- 内部工具 ----

func (h *FamilyHandler) familyOf(c *gin.Context) (model.Family, bool) {
	id, _ := strconv.Atoi(c.Param("id"))
	var fam model.Family
	if err := h.DB.Preload("Owner").First(&fam, id).Error; err != nil || fam.Status != 1 {
		resp.NotFound(c, "家族不存在或未通过审核")
		return fam, false
	}
	return fam, true
}

func (h *FamilyHandler) isMember(familyID, userID uint) bool {
	var count int64
	h.DB.Model(&model.FamilyMember{}).Where("family_id = ? AND user_id = ?", familyID, userID).Count(&count)
	return count > 0
}

func familyBattleTitle(win bool) string {
	if win {
		return "战胜了"
	}
	return "惜败于"
}

// optUID 在公开接口里可选解析登录用户（未带 token 或无效则返回 0）
func (h *FamilyHandler) optUID(c *gin.Context) uint {
	auth := c.GetHeader("Authorization")
	if auth == "" {
		return 0
	}
	claims, err := authutil.ParseToken(strings.TrimPrefix(auth, "Bearer "), h.Secret)
	if err != nil {
		return 0
	}
	return claims.UserID
}

// 记录家族区动态
func (h *FamilyHandler) act(familyID, userID uint, format string, args ...interface{}) {
	h.DB.Create(&model.FamilyActivity{FamilyID: familyID, UserID: userID, Content: fmt.Sprintf(format, args...)})
}

// 家族区动态（全站）
func (h *FamilyHandler) Activities(c *gin.Context) {
	var acts []model.FamilyActivity
	h.DB.Preload("User").Order("created_at DESC").Limit(10).Find(&acts)
	resp.OK(c, acts)
}

// 本家族动态
func (h *FamilyHandler) FamilyActivities(c *gin.Context) {
	fam, ok := h.familyOf(c)
	if !ok {
		return
	}
	var acts []model.FamilyActivity
	h.DB.Preload("User").Where("family_id = ?", fam.ID).Order("created_at DESC").Limit(10).Find(&acts)
	resp.OK(c, acts)
}

// 特色家族列表（is_feature=1 且已审核通过）
func (h *FamilyHandler) FeatureList(c *gin.Context) {
	var fams []model.Family
	h.DB.Preload("Owner").Where("status = 1 AND is_feature = 1").Order("battle_score DESC, id ASC").Find(&fams)
	out := make([]gin.H, 0, len(fams))
	for _, f := range fams {
		var count int64
		h.DB.Model(&model.FamilyMember{}).Where("family_id = ?", f.ID).Count(&count)
		out = append(out, gin.H{
			"id": f.ID, "name": f.Name, "slogan": f.Slogan, "description": f.Description,
			"category": f.Category, "owner_id": f.OwnerID, "owner": f.Owner, "members": count,
			"tree_level": f.TreeLevel, "battle_score": f.BattleScore, "is_feature": f.IsFeature,
		})
	}
	resp.OK(c, out)
}

// 待审核家族列表（公开）
func (h *FamilyHandler) Pending(c *gin.Context) {
	var fams []model.Family
	h.DB.Preload("Owner").Where("status = 2").Order("created_at ASC").Find(&fams)
	out := make([]gin.H, 0, len(fams))
	for _, f := range fams {
		owner := ""
		if f.Owner != nil {
			owner = f.Owner.Nickname
		}
		out = append(out, gin.H{"id": f.ID, "name": f.Name, "slogan": f.Slogan,
			"description": f.Description, "category": f.Category, "owner_id": f.OwnerID, "owner": owner,
			"created_at": f.CreatedAt})
	}
	resp.OK(c, out)
}

// 家族大看台：家族活动板块最新帖子
func (h *FamilyHandler) ActivityThreads(c *gin.Context) {
	var root model.Board
	h.DB.Where("name = ?", "家族大厅").First(&root)
	var board model.Board
	h.DB.Where("parent_id = ? AND name = ?", root.ID, "家族大看台").First(&board)
	if board.ID == 0 {
		resp.OK(c, []gin.H{})
		return
	}
	var threads []model.Thread
	h.DB.Preload("User").Where("board_id = ? AND status = 1", board.ID).
		Order("is_top DESC, last_reply_at DESC, id DESC").Limit(10).Find(&threads)
	out := make([]gin.H, 0, len(threads))
	for _, t := range threads {
		out = append(out, gin.H{"id": t.ID, "title": t.Title, "view_count": t.ViewCount,
			"reply_count": t.ReplyCount, "created_at": t.CreatedAt, "user": t.User})
	}
	resp.OK(c, out)
}
