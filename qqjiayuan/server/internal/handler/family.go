package handler

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"

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
			"tree_level": f.TreeLevel, "battle_score": f.BattleScore,
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

// 创建家族（需消耗金币）
func (h *FamilyHandler) Create(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req createFamilyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "家族名称需 2-30 个字")
		return
	}
	var exists int64
	h.DB.Model(&model.Family{}).Where("name = ? AND status = 1", req.Name).Count(&exists)
	if exists > 0 {
		resp.ParamError(c, "这个家族名已被占用，换一个吧")
		return
	}
	var memberCount int64
	h.DB.Model(&model.FamilyMember{}).Where("user_id = ?", uid).Count(&memberCount)
	if memberCount > 0 {
		resp.ParamError(c, "你已经加入家族，先退出再建新的吧")
		return
	}
	var u model.User
	if err := h.DB.First(&u, uid).Error; err != nil {
		resp.ParamError(c, "用户不存在")
		return
	}
	if u.Coins < familyCreateCost {
		resp.ParamError(c, "创建家族需要 " + strconv.Itoa(familyCreateCost) + " 金币，你的金币不足")
		return
	}
	fam := model.Family{Name: req.Name, Slogan: req.Slogan, Description: req.Description,
		Category: req.Category, OwnerID: uid, TreeLevel: 1}
	if err := h.DB.Create(&fam).Error; err != nil {
		resp.ServerError(c, err)
		return
	}
	h.DB.Create(&model.FamilyMember{FamilyID: fam.ID, UserID: uid, Role: "owner"})
	h.DB.Model(&u).Update("coins", gorm.Expr("coins - ?", familyCreateCost))
	h.act(fam.ID, uid, "创建了家族《%s》", fam.Name)
	resp.OK(c, gin.H{"id": fam.ID})
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

	out := gin.H{
		"id": fam.ID, "name": fam.Name, "slogan": fam.Slogan, "description": fam.Description,
		"announcement": fam.Announcement, "owner_id": fam.OwnerID, "owner": fam.Owner,
		"tree_level": fam.TreeLevel, "tree_exp": fam.TreeExp, "battle_score": fam.BattleScore,
		"members": members, "my_role": myRole, "member_count": len(members),
		"signed_today": signed > 0, "tree_today": treeToday, "created_at": fam.CreatedAt,
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

// ---- 内部工具 ----

func (h *FamilyHandler) familyOf(c *gin.Context) (model.Family, bool) {
	id, _ := strconv.Atoi(c.Param("id"))
	var fam model.Family
	if err := h.DB.Preload("Owner").First(&fam, id).Error; err != nil || fam.Status == 0 {
		resp.NotFound(c, "家族不存在或已解散")
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
	h.DB.Preload("User").Order("created_at DESC").Limit(30).Find(&acts)
	resp.OK(c, acts)
}

// 本家族动态
func (h *FamilyHandler) FamilyActivities(c *gin.Context) {
	fam, ok := h.familyOf(c)
	if !ok {
		return
	}
	var acts []model.FamilyActivity
	h.DB.Preload("User").Where("family_id = ?", fam.ID).Order("created_at DESC").Limit(30).Find(&acts)
	resp.OK(c, acts)
}
