package handler

import (
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

type ContactHandler struct{ DB *gorm.DB }

// View 通讯录（诺哈 contact.asp：QQ/邮箱/手机 三行）
func (h *ContactHandler) View(c *gin.Context) {
	uid := middleware.GetUID(c)
	var ct model.UserContact
	h.DB.Where("user_id = ?", uid).First(&ct)
	resp.OK(c, gin.H{"qq": ct.QQ, "mail": ct.Mail, "phone": ct.Phone})
}

var (
	qqRe   = regexp.MustCompile(`^[1-9][0-9]{4,10}$`)
	mailRe = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	phoneRe = regexp.MustCompile(`^1[0-9]{10}$`)
)

// Save 设置联系方式
func (h *ContactHandler) Save(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		QQ    string `json:"qq"`
		Mail  string `json:"mail"`
		Phone string `json:"phone"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数不对")
		return
	}
	req.QQ = strings.TrimSpace(req.QQ)
	req.Mail = strings.TrimSpace(req.Mail)
	req.Phone = strings.TrimSpace(req.Phone)
	if req.QQ != "" && !qqRe.MatchString(req.QQ) {
		resp.ParamError(c, "QQ 号码须为 5-11 位数字")
		return
	}
	if req.Mail != "" && !mailRe.MatchString(req.Mail) {
		resp.ParamError(c, "邮箱格式不对")
		return
	}
	if req.Phone != "" && !phoneRe.MatchString(req.Phone) {
		resp.ParamError(c, "手机号须为 1 开头的 11 位数字")
		return
	}
	var ct model.UserContact
	h.DB.Where("user_id = ?", uid).FirstOrCreate(&ct, model.UserContact{UserID: uid})
	h.DB.Model(&ct).Updates(map[string]interface{}{"qq": req.QQ, "mail": req.Mail, "phone": req.Phone})
	resp.OK(c, gin.H{"qq": req.QQ, "mail": req.Mail, "phone": req.Phone})
}

// SaveQQ 绑定 QQ（诺哈 wap_qq：qnum + 校验，这里演示站简化为号+密码确认）
func (h *ContactHandler) SaveQQ(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		QQ     string `json:"qq" binding:"required"`
		QQPass string `json:"qq_pass" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请输入 QQ 号码和密码")
		return
	}
	if !qqRe.MatchString(strings.TrimSpace(req.QQ)) {
		resp.ParamError(c, "QQ 号码须为 5-11 位数字")
		return
	}
	if len(req.QQPass) < 6 {
		resp.ParamError(c, "QQ 密码不少于 6 位")
		return
	}
	var ct model.UserContact
	h.DB.Where("user_id = ?", uid).FirstOrCreate(&ct, model.UserContact{UserID: uid})
	h.DB.Model(&ct).Update("qq", strings.TrimSpace(req.QQ))
	resp.OK(c, gin.H{"qq": strings.TrimSpace(req.QQ)})
}
