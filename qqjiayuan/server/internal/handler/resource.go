package handler

import (
	"encoding/base64"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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
	// 输出 has_data 标记（Data 本身不外发，前端用 /api/res/ 取图）
	out := make([]gin.H, 0, len(list))
	for _, r := range list {
		out = append(out, gin.H{
			"id": r.ID, "file": r.File, "category": r.Category, "name": r.Name,
			"level": r.Level, "status": r.Status, "has_data": r.Data != "",
		})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// 管理端：上传图片入库（base64 data URI 存储，对齐"图片资源由数据库维护"）
func (h *ResourceHandler) Upload(c *gin.Context) {
	var req struct {
		Name     string `json:"name" binding:"required,max=30"`
		Category string `json:"category" binding:"required,oneof=badge avatar game priv other"`
		Level    int    `json:"level"`
		Data     string `json:"data" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请填写名称/分类并选择图片")
		return
	}
	if !strings.HasPrefix(req.Data, "data:image/") || !strings.Contains(req.Data, ";base64,") {
		resp.ParamError(c, "仅支持 JPG/PNG/GIF/WEBP/BMP 图片")
		return
	}
	if len(req.Data) > 1400*1024 {
		resp.ParamError(c, "图片不能超过 1MB，请压缩后上传")
		return
	}
	mime := req.Data[len("data:"):strings.Index(req.Data, ";")]
	ext := map[string]string{"image/gif": ".gif", "image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp", "image/bmp": ".bmp"}[mime]
	if ext == "" {
		resp.ParamError(c, "不支持的图片格式")
		return
	}
	file := fmt.Sprintf("db/%d%d%s", time.Now().UnixNano(), rand.Intn(900)+100, ext)
	r := model.Resource{File: file, Category: req.Category, Name: req.Name, Level: req.Level, Status: 1, Data: req.Data}
	if err := h.DB.Create(&r).Error; err != nil {
		resp.ServerError(c, err)
		return
	}
	resp.OK(c, gin.H{"id": r.ID, "file": r.File, "name": r.Name, "category": r.Category})
}

// 管理端：删除资源（库存图片连同数据删除）
func (h *ResourceHandler) Delete(c *gin.Context) {
	id, ok := positiveParam(c, "id")
	if !ok {
		return
	}
	var r model.Resource
	if err := h.DB.First(&r, id).Error; err != nil {
		resp.NotFound(c, "资源不存在")
		return
	}
	h.DB.Delete(&r)
	resp.OK(c, nil)
}

// 公开：库存图片取图（/api/res/db/xxx.gif）—— 解码 base64 直接输出，带客户端缓存
func (h *ResourceHandler) Serve(c *gin.Context) {
	path := strings.TrimPrefix(c.Param("path"), "/")
	var r model.Resource
	if err := h.DB.Where("file = ?", path).First(&r).Error; err != nil || r.Data == "" {
		c.String(404, "not found")
		return
	}
	idx := strings.Index(r.Data, ";base64,")
	if idx < 0 {
		c.String(404, "not found")
		return
	}
	mime := r.Data[len("data:"):idx]
	payload := r.Data[idx+len(";base64,"):]
	raw, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		c.String(404, "not found")
		return
	}
	c.Header("Cache-Control", "public, max-age=86400")
	c.Data(200, mime, raw)
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
