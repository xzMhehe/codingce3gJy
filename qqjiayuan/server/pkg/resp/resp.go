package resp

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// 统一响应结构 {code, msg, data}
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok", "data": data})
}

func Fail(c *gin.Context, httpStatus, code int, msg string) {
	c.JSON(httpStatus, gin.H{"code": code, "msg": msg, "data": nil})
}

func ParamError(c *gin.Context, msg string) {
	Fail(c, http.StatusOK, 400, msg)
}

func Unauthorized(c *gin.Context, msg string) {
	Fail(c, http.StatusOK, 401, msg)
}

func Forbidden(c *gin.Context, msg string) {
	Fail(c, http.StatusOK, 403, msg)
}

func NotFound(c *gin.Context, msg string) {
	Fail(c, http.StatusOK, 404, msg)
}

func ServerError(c *gin.Context, err error) {
	Fail(c, http.StatusOK, 500, "服务器开小差了: "+err.Error())
}
