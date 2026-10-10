package ezfy

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/model"
)

// EzfyAdmin 二战风云 GM 管理端处理器。
//
// ★ 与论坛/西游共用的 handler.AdminHandler 无关：
//   Go 规定方法必须与接收者类型同包，而 AdminHandler 还挂着论坛/西游的 GM 方法，
//   因此二战把 GM 方法统一挂到这个独立类型上，整个二战（游戏 + GM）代码都收敛在 ezfy 包，
//   为将来二战独立部署预留 —— 不动论坛/西游的 AdminHandler。
type EzfyAdmin struct {
	// DB 二战库（独立库模式下 = qq_ezzt；单库模式 = 家园库）。
	DB *gorm.DB
	// HomeDB 家园库：账号/昵称等**仍由家园持有**的数据经它只读（见 EzfyHandler.HomeDB）。
	HomeDB *gorm.DB
}

// home 家园库句柄：未配 HomeDB（单库模式）时回落 DB。
func (h *EzfyAdmin) home() *gorm.DB {
	if h.HomeDB != nil {
		return h.HomeDB
	}
	return h.DB
}

// pageOf 通用分页参数：page 从 1 起，size 默认 defSize（1~100），返回 page、offset、size。
// （handler 包 user.go 同款，这里复制一份，让 ezfy 包不依赖 handler 包。）
func pageOf(c *gin.Context, defSize int) (int, int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", strconv.Itoa(defSize)))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = defSize
	}
	return page, (page - 1) * size, size
}

// xySettingSet 写 settings 键值（存在则更新，否则创建）。
// （handler 包 hxxy_admin.go 同款，这里复制一份，让 ezfy 包不依赖 handler 包。）
func xySettingSet(db *gorm.DB, key, val string) {
	var cnt int64
	db.Model(&model.Setting{}).Where("`key` = ?", key).Count(&cnt)
	if cnt > 0 {
		db.Model(&model.Setting{}).Where("`key` = ?", key).Update("value", val)
	} else {
		db.Create(&model.Setting{Key: key, Value: val})
	}
}

// trimStr 截断到 n 个字符（按 rune 数，避免截断中文）。
// （handler 包 hxxy_sys.go 同款，这里复制一份，让 ezfy 包不依赖 handler 包。）
func trimStr(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}

// xyConvVal 按字段类型转换 JSON 数值。
// （handler 包 hxxy_admin.go 同款，这里复制一份，让 ezfy 包不依赖 handler 包。）
func xyConvVal(v interface{}, typ string) (interface{}, bool) {
	switch typ {
	case "string":
		s, ok := v.(string)
		return s, ok
	default: // int / int64（JSON 数字均为 float64）
		f, ok := v.(float64)
		if !ok {
			return nil, false
		}
		if typ == "int64" {
			return int64(f), true
		}
		return int(f), true
	}
}

// xyPickVals 从输入 map 里按字段白名单挑出可写值。
// （handler 包 hxxy_admin.go 同款，这里复制一份，让 ezfy 包不依赖 handler 包。）
func xyPickVals(in map[string]interface{}, fields map[string]string) map[string]interface{} {
	vals := map[string]interface{}{}
	for k, typ := range fields {
		v, ok := in[k]
		if !ok {
			continue
		}
		if cv, ok2 := xyConvVal(v, typ); ok2 {
			vals[k] = cv
		}
	}
	return vals
}

// atoiDefault 解析整数，失败时返回默认值。
// （handler 包 timelimit.go 同款，这里复制一份，让 ezfy 包不依赖 handler 包。）
func atoiDefault(s string, d int) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return d
	}
	return n
}

// clampInt 把 v 钳制在 [lo, hi]。
// （handler 包 farm.go 同款，这里复制一份，让 ezfy 包不依赖 handler 包。）
func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
