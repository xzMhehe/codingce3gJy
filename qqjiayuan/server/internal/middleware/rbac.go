package middleware

import (
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
