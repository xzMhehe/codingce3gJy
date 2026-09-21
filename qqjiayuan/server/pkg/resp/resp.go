package resp

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// 统一响应结构 {code, msg, data}
//
// ★ 用户反馈「用户端消息提示还都是 ok」：
// 原来 OK() 的 msg 恒为字符串 "ok"，凡是没在 data 里带 msg 的接口，
// 前端弹出来的就是「ok」，玩家看不懂发生了什么。
//
// 现在的口径：
//  1. data 是 gin.H 且带 msg → **提升**到顶层 msg（前端读 r.msg 就能拿到具体文案）；
//  2. data 没带 msg → 顶层回落「操作成功」（而不是 "ok"）；
//  3. 需要更具体的文案时，用 OKMsg 显式指定。
//
// 这样前端无论读 r.msg 还是 r.data.msg，都不会再出现「ok」。
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": okMsgOf(data), "data": data})
}

// OKMsg 成功响应 + 显式顶层文案（data 里没有 msg 时用这个）
func OKMsg(c *gin.Context, msg string, data interface{}) {
	if msg == "" {
		msg = "操作成功"
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": msg, "data": data})
}

// okMsgOf 从 data 里提取可展示的成功文案
func okMsgOf(data interface{}) string {
	if m, ok := data.(gin.H); ok {
		if s, ok2 := m["msg"].(string); ok2 && s != "" {
			return s
		}
	}
	return "操作成功"
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
