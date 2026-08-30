package handler

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

type ResourceHandler struct {
	DB        *gorm.DB
	StaticDir string // 如 ../web/dist/static
}

// 管理端：资源分页列表（分类筛选 + 关键字）
func (h *ResourceHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	q := h.DB.Model(&model.Resource{})
	if cat := c.Query("category"); cat != "" {
		q = q.Where("category = ?", cat)
	}
	if word := c.Query("word"); word != "" {
		like := "%" + word + "%"
		q = q.Where("file LIKE ? OR name LIKE ?", like, like)
	}
	var total int64
	q.Count(&total)
	if maxPage := int(total + int64(size) - 1) / size; maxPage < 1 {
		page = 1
	} else if page > maxPage {
		page = maxPage
	}
	var list []model.Resource
	q.Order("category ASC, level ASC, id ASC").Offset((page - 1) * size).Limit(size).Find(&list)
	resp.OK(c, gin.H{"list": list, "total": total, "page": page, "size": size})
}

// 管理端：编辑资源（名称/分类/等级/启停）
func (h *ResourceHandler) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Name     string `json:"name" binding:"max=30"`
		Category string `json:"category" binding:"required,oneof=badge avatar game priv other"`
		Level    int    `json:"level"`
		Status   *int   `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "分类必须是 badge/avatar/game/priv/other")
		return
	}
	var r model.Resource
	if err := h.DB.First(&r, id).Error; err != nil {
		resp.NotFound(c, "资源不存在")
		return
	}
	updates := map[string]interface{}{"name": req.Name, "category": req.Category, "level": req.Level}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	h.DB.Model(&r).Updates(updates)
	resp.OK(c, r)
}

// 管理端：扫描 static 目录，把新文件登记为「其他」类资源
func (h *ResourceHandler) Sync(c *gin.Context) {
	added := 0
	for _, sub := range []string{"picture", "image"} {
		dir := filepath.Join(h.StaticDir, sub)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			ext := strings.ToLower(filepath.Ext(name))
			if ext != ".gif" && ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".bmp" {
				continue
			}
			file := sub + "/" + name
			var n int64
			h.DB.Model(&model.Resource{}).Where("file = ?", file).Count(&n)
			if n == 0 {
				h.DB.Create(&model.Resource{
					File: file, Category: "other",
					Name: strings.TrimSuffix(name, ext), Status: 1,
				})
				added++
			}
		}
	}
	resp.OK(c, gin.H{"added": added})
}

// 公开：特权等级列表（用户页/管理面板选择用）
func (h *ResourceHandler) Privs(c *gin.Context) {
	var list []model.Resource
	h.DB.Where("category = ? AND status = 1", "priv").Order("level ASC").Find(&list)
	resp.OK(c, list)
}
