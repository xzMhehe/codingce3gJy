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
	resp.OK(c, gin.H{"id": fam.ID, "name": fam.Name, "slogan": fam.Slogan, "role": m.Role, "members": count})
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

	out := gin.H{
		"id": fam.ID, "name": fam.Name, "slogan": fam.Slogan, "description": fam.Description,
		"announcement": fam.Announcement, "owner_id": fam.OwnerID, "owner": fam.Owner,
		"tree_level": fam.TreeLevel, "tree_exp": fam.TreeExp, "battle_score": fam.BattleScore,
		"members": members, "my_role": myRole, "member_count": len(members),
		"signed_today": signed > 0, "tree_today": treeToday, "online": online, "created_at": fam.CreatedAt,
		"forum_board_id": forumBoard.ID,
	}
	resp.OK(c, out)
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
	addExpAndCoins(h.DB, uid, 20, 5, 1)
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
	addExpAndCoins(h.DB, uid, 10, 3, 1)
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
	page, _ := pageOf(c, 10)
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
