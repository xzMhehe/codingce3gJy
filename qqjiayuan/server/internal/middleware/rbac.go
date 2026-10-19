package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/pkg/resp"
)

// RBAC 鉴权：校验当前用户是否拥有指定权限码
// 用法：router.DELETE("/x", JWTAuth(...), RequirePerm(db, "thread:manage"), handler)
func RequirePerm(db *gorm.DB, permCode string) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := GetUID(c)
		var count int64
		db.Raw(`
SELECT COUNT(DISTINCT p.id)
FROM permissions p
JOIN role_permissions rp ON rp.permission_id = p.id
JOIN user_roles ur ON ur.role_id = rp.role_id
WHERE ur.user_id = ? AND p.code = ?`, uid, permCode).Scan(&count)
		if count == 0 {
			resp.Forbidden(c, "你没有权限进入！")
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireAnyPerm 校验当前用户拥有任一指定权限码（多个菜单共用同一接口时用）
// 用法：router.GET("/x", JWTAuth(...), RequireAnyPerm(db, "module:a", "module:b"), handler)
func RequireAnyPerm(db *gorm.DB, codes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := GetUID(c)
		var count int64
		db.Raw(`
SELECT COUNT(DISTINCT p.id)
FROM permissions p
JOIN role_permissions rp ON rp.permission_id = p.id
JOIN user_roles ur ON ur.role_id = rp.role_id
WHERE ur.user_id = ? AND p.code IN ?`, uid, codes).Scan(&count)
		if count == 0 {
			resp.Forbidden(c, "你没有权限进入！")
			c.Abort()
			return
		}
		c.Next()
	}
}

// UserPermissionCodes 返回用户全部权限码
func UserPermissionCodes(db *gorm.DB, uid uint) []string {
	var codes []string
	db.Raw(`
SELECT DISTINCT p.code
FROM permissions p
JOIN role_permissions rp ON rp.permission_id = p.id
JOIN user_roles ur ON ur.role_id = rp.role_id
WHERE ur.user_id = ?`, uid).Scan(&codes)
	return codes
}

// HasAdminPerm 该账号是否拥有任一「管理菜单权限」（module:*）——即能进管理端的人。
// 与 admin-web 登录后的判断同口径（Login.vue: perms 里存在 module:* 前缀才算管理端可进）。
// ★★ 2026-10-19 管理端与用户端 token 分离：登录接口 `scope=admin` 时用它卡资格 ——
//
//	没有 module:* 权限的玩家**不能**换到管理端 token（否则管理端会话作废线会被玩家顶）。
//	注意：会话隔离本身按「登录端」分流（见 authutil.AdminIssuer / users.admin_session_at），
//	不再用「这个账号是不是管理员」来判断要不要写 session_at。
func HasAdminPerm(db *gorm.DB, uid uint) bool {
	for _, code := range UserPermissionCodes(db, uid) {
		if strings.HasPrefix(code, "module:") {
			return true
		}
	}
	return false
}
