package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

type SpaceHandler struct{ DB *gorm.DB }

const defaultSpacePageSize = 10

func positiveParam(c *gin.Context, name string) (int, bool) {
	id, err := strconv.Atoi(c.Param(name))
	if err != nil || id <= 0 {
		resp.ParamError(c, name+" 必须是正整数")
		return 0, false
	}
	return id, true
}

func positiveQuery(c *gin.Context, name string) (int, bool) {
	id, err := strconv.Atoi(c.Query(name))
	if err != nil || id <= 0 {
		resp.ParamError(c, name+" 必须是正整数")
		return 0, false
	}
	return id, true
}

func pageParams(c *gin.Context, fallback int) (int, int) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 { page = 1 }
	sizeText := c.Query("size")
	if sizeText == "" { sizeText = c.Query("page_size") }
	size, err := strconv.Atoi(sizeText)
	if err != nil || size < 1 { size = fallback }
	if size > 100 { size = 100 }
	return page, size
}

func (h *SpaceHandler) activeSpace(userID int) (model.Space, error) {
	var space model.Space
	err := h.DB.Where("user_id = ? AND status = 1", userID).First(&space).Error
	return space, err
}

func queryError(c *gin.Context, err error) bool {
	if err == nil { return false }
	resp.ServerError(c, err)
	return true
}

// ---- 空间信息 ----

// OpenSpace 开通空间
func (h *SpaceHandler) OpenSpace(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Name      string `json:"name" binding:"max=50"`
		Signature string `json:"signature" binding:"max=200"`
		Intro     string `json:"intro" binding:"max=500"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数不对")
		return
	}
	var space model.Space
	if err := h.DB.Where("user_id = ?", uid).First(&space).Error; err == nil {
		resp.ParamError(c, "空间已开通")
		return
	}
	space = model.Space{
		UserID:    uid,
		Name:      req.Name,
		Signature: req.Signature,
		Intro:     req.Intro,
		Status:    1,
	}
	if err := h.DB.Create(&space).Error; err != nil {
		resp.ServerError(c, err)
		return
	}
	resp.OK(c, space)
}

// SpaceInfo 获取空间信息（公开）
func (h *SpaceHandler) SpaceInfo(c *gin.Context) {
	userID, ok := positiveParam(c, "userId")
	if !ok { return }
	space, err := h.activeSpace(userID)
	if err != nil {
		resp.NotFound(c, "空间未开通或已关闭")
		return
	}
	// 统计
	var moodCount, articleCount, albumCount, msgCount, visitorCount int64
	h.DB.Model(&model.Mood{}).Where("user_id = ? AND status = 1", userID).Count(&moodCount)
	h.DB.Model(&model.Article{}).Where("user_id = ? AND status = 1", userID).Count(&articleCount)
	h.DB.Model(&model.Album{}).Where("user_id = ?", userID).Count(&albumCount)
	h.DB.Model(&model.SpaceMessage{}).Where("to_user_id = ? AND status = 1", userID).Count(&msgCount)
	h.DB.Model(&model.Visitor{}).Where("owner_id = ?", userID).Count(&visitorCount)

	resp.OK(c, gin.H{
		"space":        space,
		"mood_count":   moodCount,
		"article_count": articleCount,
		"album_count":  albumCount,
		"msg_count":    msgCount,
		"visitor_count": visitorCount,
	})
}

// UpdateSpace 更新空间设置
func (h *SpaceHandler) UpdateSpace(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Name      string `json:"name" binding:"max=50"`
		Signature string `json:"signature" binding:"max=200"`
		Intro     string `json:"intro" binding:"max=500"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数不对")
		return
	}
	h.DB.Model(&model.Space{}).Where("user_id = ?", uid).Updates(map[string]interface{}{
		"name": req.Name, "signature": req.Signature, "intro": req.Intro,
	})
	resp.OK(c, nil)
}

// ---- 心情说说 ----

// MoodList 心情列表（自己的或某用户的）
func (h *SpaceHandler) MoodList(c *gin.Context) {
	userID, ok := positiveQuery(c, "user_id")
	if !ok { return }
	if _, err := h.activeSpace(userID); err != nil {
		resp.NotFound(c, "空间未开通或已关闭")
		return
	}
	page, pageSize := pageParams(c, defaultSpacePageSize)
	q := h.DB.Model(&model.Mood{}).Where("user_id = ? AND status = 1", userID)
	var total int64
	q.Count(&total)
	var moods []model.Mood
	q.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&moods)

	// 加载评论数和评论
	type moodCommentExtra struct {
		model.MoodComment
		Nickname string `json:"nickname"`
	}
	type moodWithExtra struct {
		model.Mood
		Nickname     string             `json:"nickname"`
		CommentCount int64              `json:"comment_count"`
		ForwardCount int64              `json:"forward_count"`
		Comments     []moodCommentExtra `json:"comments"`
	}
	var out []moodWithExtra
	for _, m := range moods {
		var u model.User
		h.DB.Select("nickname").First(&u, m.UserID)
		var cc int64
		h.DB.Model(&model.MoodComment{}).Where("mood_id = ?", m.ID).Count(&cc)
		var fc int64
		h.DB.Model(&model.Mood{}).Where("content LIKE ? AND user_id = ?", "转发：%"+truncStr(m.Content, 20)+"%", m.UserID).Count(&fc)
		var rawComments []model.MoodComment
		h.DB.Where("mood_id = ?", m.ID).Order("created_at ASC").Limit(5).Find(&rawComments)
		var comments []moodCommentExtra
		for _, c := range rawComments {
			var cu model.User
			h.DB.Select("nickname").First(&cu, c.UserID)
			comments = append(comments, moodCommentExtra{MoodComment: c, Nickname: cu.Nickname})
		}
		out = append(out, moodWithExtra{Mood: m, Nickname: u.Nickname, CommentCount: cc, ForwardCount: fc, Comments: comments})
	}
	resp.OK(c, gin.H{"total": total, "page": page, "size": pageSize, "list": out})
}

// MoodAdd 发表心情
func (h *SpaceHandler) MoodAdd(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Content string `json:"content" binding:"required,max=500"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "内容不能为空，最多500字")
		return
	}
	m := model.Mood{UserID: uid, Content: req.Content, Status: 1}
	h.DB.Create(&m)
	resp.OK(c, m)
}

// MoodDel 删除心情
func (h *SpaceHandler) MoodDel(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	uid := middleware.GetUID(c)
	h.DB.Model(&model.Mood{}).Where("id = ? AND user_id = ?", id, uid).Update("status", 0)
	resp.OK(c, nil)
}

// MoodCommentAdd 评论心情
func (h *SpaceHandler) MoodCommentAdd(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Content string `json:"content" binding:"required,max=300"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "评论不能为空")
		return
	}
	moodID, _ := strconv.Atoi(c.Param("id"))
	mc := model.MoodComment{MoodID: uint(moodID), UserID: uid, Content: req.Content}
	h.DB.Create(&mc)
	resp.OK(c, mc)
}

// MoodForward 转发心情
func (h *SpaceHandler) MoodForward(c *gin.Context) {
	uid := middleware.GetUID(c)
	srcID, _ := strconv.Atoi(c.Param("id"))
	var src model.Mood
	if err := h.DB.First(&src, srcID).Error; err != nil || src.Status != 1 {
		resp.NotFound(c, "心情不存在")
		return
	}
	m := model.Mood{UserID: uid, Content: "转发：" + src.Content, Status: 1}
	h.DB.Create(&m)
	resp.OK(c, m)
}

// ---- 日志 ----

// ArticleList 日志列表
func (h *SpaceHandler) ArticleList(c *gin.Context) {
	userID, ok := positiveQuery(c, "user_id")
	if !ok { return }
	if _, err := h.activeSpace(userID); err != nil { resp.NotFound(c, "空间未开通或已关闭"); return }
	page, pageSize := pageParams(c, defaultSpacePageSize)
	q := h.DB.Model(&model.Article{}).Where("user_id = ? AND status = 1", userID)
	var total int64
	q.Count(&total)
	var articles []model.Article
	q.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&articles)
	resp.OK(c, gin.H{"total": total, "page": page, "size": pageSize, "list": articles})
}

// ArticleDetail 日志详情
func (h *SpaceHandler) ArticleDetail(c *gin.Context) {
	id, ok := positiveParam(c, "id")
	if !ok { return }
	var a model.Article
	if err := h.DB.Where("id = ? AND status = 1", id).First(&a).Error; err != nil {
		resp.NotFound(c, "日志不存在")
		return
	}
	if _, err := h.activeSpace(int(a.UserID)); err != nil {
		resp.NotFound(c, "空间未开通或已关闭")
		return
	}
	resp.OK(c, a)
}

// ArticleAdd 发表日志
func (h *SpaceHandler) ArticleAdd(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Title   string `json:"title" binding:"required,max=100"`
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "标题和内容不能为空")
		return
	}
	a := model.Article{UserID: uid, Title: req.Title, Content: req.Content, Status: 1}
	h.DB.Create(&a)
	resp.OK(c, a)
}

// ArticleDel 删除日志
func (h *SpaceHandler) ArticleDel(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	uid := middleware.GetUID(c)
	h.DB.Model(&model.Article{}).Where("id = ? AND user_id = ?", id, uid).Update("status", 0)
	resp.OK(c, nil)
}

// ---- 相册 ----

// AlbumList 相册列表
func (h *SpaceHandler) AlbumList(c *gin.Context) {
	userID, ok := positiveQuery(c, "user_id")
	if !ok { return }
	if _, err := h.activeSpace(userID); err != nil { resp.NotFound(c, "空间未开通或已关闭"); return }
	var albums []model.Album
	h.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&albums)
	resp.OK(c, albums)
}

// AlbumCreate 创建相册
func (h *SpaceHandler) AlbumCreate(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Name string `json:"name" binding:"required,max=50"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "相册名称不能为空")
		return
	}
	a := model.Album{UserID: uid, Name: req.Name}
	h.DB.Create(&a)
	resp.OK(c, a)
}

// ---- 留言 ----

// SpaceMsgList 留言列表
func (h *SpaceHandler) SpaceMsgList(c *gin.Context) {
	userID, ok := positiveQuery(c, "user_id")
	if !ok { return }
	if _, err := h.activeSpace(userID); err != nil { resp.NotFound(c, "空间未开通或已关闭"); return }
	page, pageSize := pageParams(c, 20)
	q := h.DB.Model(&model.SpaceMessage{}).Where("to_user_id = ? AND status = 1", userID)
	var total int64
	q.Count(&total)
	var msgs []model.SpaceMessage
	q.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&msgs)

	type msgWithUser struct {
		model.SpaceMessage
		FromNickname string `json:"from_nickname"`
	}
	var out []msgWithUser
	for _, m := range msgs {
		var u model.User
		h.DB.Select("nickname").First(&u, m.FromUserID)
		out = append(out, msgWithUser{SpaceMessage: m, FromNickname: u.Nickname})
	}
	resp.OK(c, gin.H{"total": total, "page": page, "size": pageSize, "list": out})
}

// SpaceMsgAdd 留言
func (h *SpaceHandler) SpaceMsgAdd(c *gin.Context) {
	uid := middleware.GetUID(c)
	toUserID, _ := strconv.Atoi(c.Param("userId"))
	var req struct {
		Content string `json:"content" binding:"required,max=300"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "留言不能为空")
		return
	}
	msg := model.SpaceMessage{ToUserID: uint(toUserID), FromUserID: uid, Content: req.Content, Status: 1}
	h.DB.Create(&msg)
	resp.OK(c, msg)
}

// SpaceMsgDel 删除留言
func (h *SpaceHandler) SpaceMsgDel(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	uid := middleware.GetUID(c)
	// 只能删自己的或空间主人删收到的
	h.DB.Model(&model.SpaceMessage{}).Where("id = ? AND (from_user_id = ? OR to_user_id = ?)", id, uid, uid).Update("status", 0)
	resp.OK(c, nil)
}

// ---- 访客 ----

// VisitorList 访客列表
func (h *SpaceHandler) VisitorList(c *gin.Context) {
	userID, _ := strconv.Atoi(c.Query("user_id"))
	var visitors []model.Visitor
	h.DB.Where("owner_id = ?", userID).Order("created_at DESC").Limit(30).Find(&visitors)

	type visWithUser struct {
		model.Visitor
		Nickname string `json:"nickname"`
		Avatar   string `json:"avatar"`
	}
	var out []visWithUser
	for _, v := range visitors {
		var u model.User
		h.DB.Select("nickname, avatar").First(&u, v.UserID)
		out = append(out, visWithUser{Visitor: v, Nickname: u.Nickname, Avatar: u.Avatar})
	}
	resp.OK(c, out)
}

// VisitSpace 记录访问
func (h *SpaceHandler) VisitSpace(c *gin.Context) {
	uid := middleware.GetUID(c)
	toUserID, _ := strconv.Atoi(c.Param("userId"))
	if uint(toUserID) == uid {
		return // 访问自己不记录
	}
	// 同一天只记录一次
	var cnt int64
	today := time.Now().Format("2006-01-02")
	h.DB.Model(&model.Visitor{}).Where("owner_id = ? AND user_id = ? AND DATE(created_at) = ?", toUserID, uid, today).Count(&cnt)
	if cnt == 0 {
		h.DB.Create(&model.Visitor{OwnerID: uint(toUserID), UserID: uid})
	}
	resp.OK(c, nil)
}

// ---- 管理端 ----

func truncStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// AdminSpaces 管理端：空间列表
func (h *SpaceHandler) AdminSpaces(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 10
	}
	word := c.Query("word")
	q := h.DB.Model(&model.Space{}).Joins("LEFT JOIN users ON spaces.user_id = users.id")
	if word != "" {
		q = q.Where("users.nickname LIKE ? OR users.username = ?", "%"+word+"%", word)
	}
	var total int64
	q.Count(&total)
	var spaces []model.Space
	q.Order("spaces.id DESC").Offset((page - 1) * size).Limit(size).Find(&spaces)

	type spaceWithUser struct {
		model.Space
		Nickname string `json:"nickname"`
		Username string `json:"username"`
	}
	var out []spaceWithUser
	for _, s := range spaces {
		var u model.User
		h.DB.Select("nickname, username").First(&u, s.UserID)
		out = append(out, spaceWithUser{Space: s, Nickname: u.Nickname, Username: u.Username})
	}
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": out})
}

// AdminSpaceStatus 管理端：启用/关闭空间
func (h *SpaceHandler) AdminSpaceStatus(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Status int `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || (req.Status != 0 && req.Status != 1) {
		resp.ParamError(c, "status 只能是 0（关闭）或 1（正常）")
		return
	}
	h.DB.Model(&model.Space{}).Where("id = ?", id).Update("status", req.Status)
	resp.OK(c, nil)
}

// AdminSpaceDelete 管理端：删除空间
func (h *SpaceHandler) AdminSpaceDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Model(&model.Space{}).Where("id = ?", id).Update("status", 0)
	resp.OK(c, nil)
}

// AdminSpaceMoods 管理端：查看某空间的心情
func (h *SpaceHandler) AdminSpaceMoods(c *gin.Context) {
	userID, _ := strconv.Atoi(c.Param("userId"))
	var moods []model.Mood
	h.DB.Where("user_id = ?", userID).Order("created_at DESC").Limit(50).Find(&moods)
	resp.OK(c, moods)
}

// AdminDeleteMood 管理端：删除心情
func (h *SpaceHandler) AdminDeleteMood(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Model(&model.Mood{}).Where("id = ?", id).Update("status", 0)
	resp.OK(c, nil)
}

// AdminSpaceArticles 管理端：查看某空间的日志
func (h *SpaceHandler) AdminSpaceArticles(c *gin.Context) {
	userID, _ := strconv.Atoi(c.Param("userId"))
	var articles []model.Article
	h.DB.Where("user_id = ?", userID).Order("created_at DESC").Limit(50).Find(&articles)
	resp.OK(c, articles)
}

// AdminDeleteArticle 管理端：删除日志
func (h *SpaceHandler) AdminDeleteArticle(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Model(&model.Article{}).Where("id = ?", id).Update("status", 0)
	resp.OK(c, nil)
}

// AdminSpaceAlbums 管理端：查看某空间的相册（不看空间状态）
func (h *SpaceHandler) AdminSpaceAlbums(c *gin.Context) {
	userID, ok := positiveParam(c, "userId")
	if !ok { return }
	var albums []model.Album
	h.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&albums)
	resp.OK(c, albums)
}

// AdminSpaceMessages 管理端：查看某空间的留言（不看空间状态）
func (h *SpaceHandler) AdminSpaceMessages(c *gin.Context) {
	userID, ok := positiveParam(c, "userId")
	if !ok { return }
	page, pageSize := pageParams(c, 20)
	q := h.DB.Model(&model.SpaceMessage{}).Where("to_user_id = ? AND status = 1", userID)
	var total int64
	q.Count(&total)
	var msgs []model.SpaceMessage
	q.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&msgs)

	type msgWithUser struct {
		model.SpaceMessage
		FromNickname string `json:"from_nickname"`
	}
	var out []msgWithUser
	for _, m := range msgs {
		var u model.User
		h.DB.Select("nickname").First(&u, m.FromUserID)
		out = append(out, msgWithUser{SpaceMessage: m, FromNickname: u.Nickname})
	}
	resp.OK(c, gin.H{"total": total, "page": page, "size": pageSize, "list": out})
}

// AdminSpaceVisitors 管理端：查看某空间的访客（不看空间状态）
func (h *SpaceHandler) AdminSpaceVisitors(c *gin.Context) {
	userID, ok := positiveParam(c, "userId")
	if !ok { return }
	page, pageSize := pageParams(c, 10)
	q := h.DB.Model(&model.Visitor{}).Where("owner_id = ?", userID)
	var total int64
	q.Count(&total)
	var visitors []model.Visitor
	q.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&visitors)

	type visitorWithUser struct {
		model.Visitor
		Nickname string `json:"nickname"`
	}
	var out []visitorWithUser
	for _, v := range visitors {
		var u model.User
		h.DB.Select("nickname").First(&u, v.UserID)
		out = append(out, visitorWithUser{Visitor: v, Nickname: u.Nickname})
	}
	resp.OK(c, gin.H{"total": total, "page": page, "size": pageSize, "list": out})
}

// AdminAlbumPhotos 管理端：按相册查看照片（管理端用）
func (h *SpaceHandler) AdminAlbumPhotos(c *gin.Context) {
	albumID, ok := positiveParam(c, "id")
	if !ok { return }
	page, pageSize := pageParams(c, 10)
	q := h.DB.Model(&model.Photo{}).Where("album_id = ?", albumID)
	var total int64
	q.Count(&total)
	var photos []model.Photo
	q.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&photos)
	resp.OK(c, gin.H{"total": total, "page": page, "size": pageSize, "list": photos})
}

// AdminDeletePhoto 管理端：删除照片（软删）
func (h *SpaceHandler) AdminDeletePhoto(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.Photo{}, id)
	resp.OK(c, nil)
}

// AdminDeleteAlbum 管理端：删除相册（软删相册及照片）
func (h *SpaceHandler) AdminDeleteAlbum(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.Album{}, id)
	h.DB.Where("album_id = ?", id).Delete(&model.Photo{})
	resp.OK(c, nil)
}

// AdminDeleteSpaceMessage 管理端：删除留言
func (h *SpaceHandler) AdminDeleteSpaceMessage(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Model(&model.SpaceMessage{}).Where("id = ?", id).Update("status", 0)
	resp.OK(c, nil)
}


