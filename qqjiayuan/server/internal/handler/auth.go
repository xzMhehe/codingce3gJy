package handler

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/authutil"
	"qqjiayuan/server/pkg/resp"
)

type AuthHandler struct {
	DB     *gorm.DB
	Secret string
	ExpH   int
}

type regReq struct {
	Nickname string `json:"nickname" binding:"required,min=1,max=20"`
	Password string `json:"password" binding:"required,min=6,max=20"`
	Gender   int    `json:"gender"`
}

// 注册：昵称+性别+密码，系统自动分配家园号码（靓号）
func (h *AuthHandler) Register(c *gin.Context) {
	var req regReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "昵称或密码格式不对：昵称1-20字，密码6-20位")
		return
	}
	if req.Gender != 1 && req.Gender != 2 {
		req.Gender = 1
	}
	var exists int64
	h.DB.Model(&model.User{}).Where("nickname = ?", req.Nickname).Count(&exists)
	if exists > 0 {
		resp.ParamError(c, "这个昵称已经被注册了，换一个吧")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		resp.ServerError(c, err)
		return
	}
	user := model.User{
		Nickname: req.Nickname,
		Password: string(hash),
		Gender:   req.Gender,
		Coins:    100, // 新人礼包
		Level:    1,
	}
	if err := h.DB.Create(&user).Error; err != nil {
		resp.ServerError(c, err)
		return
	}
	h.DB.Model(&user).Update("username", fmt.Sprintf("%d", user.ID))

	var member model.Role
	h.DB.Where("code = ?", "member").First(&member)
	h.DB.Model(&user).Association("Roles").Append(&member)

	h.DB.Create(&model.Notification{
		UserID: user.ID, Type: "system",
		Title:   "欢迎来到家园社区",
		Content: fmt.Sprintf("你的家园号码是 %d，请牢记！新人礼包100金币已到账。多逛论坛多回帖，经验等级蹭蹭涨。", user.ID),
	})
	token, _ := authutil.GenerateToken(user.ID, user.Nickname, h.Secret, h.ExpH)
	resp.OK(c, gin.H{"token": token, "user_id": user.ID, "username": user.Username, "nickname": user.Nickname})
}

type loginReq struct {
	Name     string `json:"name" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请输入账号和密码")
		return
	}
	var user model.User
	if err := h.DB.Preload("Roles").Where("username = ? OR nickname = ?", req.Name, req.Name).First(&user).Error; err != nil {
		resp.ParamError(c, "账号不存在，先免费注册一个吧")
		return
	}
	if user.Status == 0 {
		resp.Forbidden(c, "该账号已被封禁，如有疑问请联系客服")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)) != nil {
		resp.ParamError(c, "密码不对哦，再想想")
		return
	}
	token, err := authutil.GenerateToken(user.ID, user.Nickname, h.Secret, h.ExpH)
	if err != nil {
		resp.ServerError(c, err)
		return
	}
	now := gorm.Expr("NOW()")
	h.DB.Model(&user).Updates(map[string]interface{}{"last_login_at": now, "last_active_at": now})
	resp.OK(c, gin.H{"token": token, "user": h.userBrief(user)})
}

// 找回资料：凭昵称模糊找回家园号码（演示站的情怀功能）
func (h *AuthHandler) FindAccount(c *gin.Context) {
	word := strings.TrimSpace(c.Query("nickname"))
	if word == "" {
		resp.ParamError(c, "请输入你的昵称")
		return
	}
	var users []model.User
	h.DB.Where("nickname LIKE ?", "%"+word+"%").Limit(10).Find(&users)
	out := make([]gin.H, 0)
	for _, u := range users {
		// 昵称脱敏：保留首尾字符
		n := []rune(u.Nickname)
		masked := string(n[0])
		if len(n) > 2 {
			masked += strings.Repeat("*", len(n)-2)
		}
		if len(n) > 1 {
			masked += string(n[len(n)-1])
		}
		out = append(out, gin.H{"id": u.ID, "username": u.Username, "nickname": masked, "created_at": u.CreatedAt.Format("2006-01-02")})
	}
	resp.OK(c, out)
}

func (h *AuthHandler) Me(c *gin.Context) {
	uid := middleware.GetUID(c)
	var user model.User
	if err := h.DB.Preload("Roles").Preload("Badges").First(&user, uid).Error; err != nil {
		resp.NotFound(c, "用户不存在")
		return
	}
	resp.OK(c, gin.H{
		"id": user.ID, "username": user.Username, "nickname": user.Nickname,
		"gender": user.Gender, "signature": user.Signature, "color": user.Color,
		"avatar": user.Avatar, "coins": user.Coins, "exp": user.Exp, "level": user.Level,
		"yuanbao": user.YuanBao, "jinzuan": user.JinZuan, "youquan": user.YouQuan,
		"age": user.Age, "birth_year": user.BirthYear, "birth_month": user.BirthMonth, "birth_day": user.BirthDay,
		"introduction": user.Introduction, "city": user.City,
		"level_icon": user.LevelIcon, "level_title": user.LevelTitle,
		"roles": user.Roles, "badges": user.Badges, "priv": user.Priv,
		"perms": middleware.UserPermissionCodes(h.DB, uid),
	})
}

type pwdReq struct {
	Old string `json:"old" binding:"required"`
	New string `json:"new" binding:"required,min=6,max=20"`
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req pwdReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "新密码需要6-20位")
		return
	}
	var user model.User
	h.DB.First(&user, uid)
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Old)) != nil {
		resp.ParamError(c, "原密码不对哦")
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(req.New), bcrypt.DefaultCost)
	h.DB.Model(&user).Update("password", string(hash))
	resp.OK(c, nil)
}

func (h *AuthHandler) userBrief(u model.User) gin.H {
	return gin.H{"id": u.ID, "username": u.Username, "nickname": u.Nickname,
		"gender": u.Gender, "color": u.Color, "level": u.Level, "coins": u.Coins}
}
