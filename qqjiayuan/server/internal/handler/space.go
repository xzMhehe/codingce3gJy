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
	h.DB.Model(&model.Home{}).Where("user_id = ?", uid).UpdateColumn("moods", gorm.Expr("moods + 1"))
	addHomeNews(h.DB, uid, 0, 101, m.ID, "发表了心情："+req.Content)
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
	viewer := middleware.GetUID(c)
	q := h.DB.Model(&model.Article{}).
		Where("user_id = ? AND status = 1", userID).
		Where("atype = 0 OR user_id = ? OR ? = 0", viewer, viewer)
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
		CatID   uint   `json:"cat_id"`
		Atype   int    `json:"atype"` // 0公开 1不公开
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "标题和内容不能为空")
		return
	}
	if !verifyTime("article_"+strconv.Itoa(int(uid)), 15) {
		resp.ParamError(c, "发表太频繁了，歇 15 秒再来")
		return
	}
	if req.Atype != 1 {
		req.Atype = 0
	}
	a := model.Article{UserID: uid, Title: req.Title, CatID: req.CatID, Atype: req.Atype, Content: req.Content, Status: 1}
	h.DB.Create(&a)
	if req.Atype == 0 {
		addHomeNews(h.DB, uid, 0, 102, a.ID, "发表了日志《"+a.Title+"》")
	}
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
// SpaceFileList 空间文件（诺哈 blog/file：列表）
func (h *SpaceHandler) SpaceFileList(c *gin.Context) {
	userID, ok := positiveQuery(c, "user_id")
	if !ok {
		return
	}
	var files []model.SpaceFile
	h.DB.Select("id", "user_id", "name", "size", "clicks", "created_at").
		Where("user_id = ?", userID).Order("id DESC").Limit(50).Find(&files)
	resp.OK(c, files)
}

// SpaceFileAdd 传文件（base64 上传）
func (h *SpaceHandler) SpaceFileAdd(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Name       string `json:"name" binding:"required,max=100"`
		FileBase64 string `json:"file_base64" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请选择要上传的文件")
		return
	}
	if len(req.FileBase64) > 5*1024*1024 {
		resp.ParamError(c, "文件太大，请压缩到 5MB 以内")
		return
	}
	f := model.SpaceFile{UserID: uid, Name: req.Name, FileBase64: req.FileBase64, Size: len(req.FileBase64)}
	h.DB.Create(&f)
	resp.OK(c, gin.H{"id": f.ID, "name": f.Name})
}

// SpaceFileDel 删除空间文件
func (h *SpaceHandler) SpaceFileDel(c *gin.Context) {
	uid := middleware.GetUID(c)
	id, _ := strconv.Atoi(c.Param("id"))
	res := h.DB.Where("id = ? AND user_id = ?", id, uid).Delete(&model.SpaceFile{})
	if res.RowsAffected == 0 {
		resp.NotFound(c, "文件不存在")
		return
	}
	resp.OK(c, "已删除")
}

// SpaceFileDownload 下载空间文件（记点击）
func (h *SpaceHandler) SpaceFileDownload(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var f model.SpaceFile
	if err := h.DB.First(&f, id).Error; err != nil {
		resp.NotFound(c, "文件不存在")
		return
	}
	h.DB.Model(&f).UpdateColumn("clicks", gorm.Expr("clicks + 1"))
	resp.OK(c, gin.H{"name": f.Name, "base64": f.FileBase64})
}

func (h *SpaceHandler) SpaceFriends(c *gin.Context) {	userID, ok := positiveQuery(c, "user_id")
	if !ok {
		return
	}
	var ids []uint
	h.DB.Model(&model.Friendship{}).Where("user_id = ? AND status = 1", userID).
		Order("id DESC").Limit(50).Pluck("friend_id", &ids)
	out := []gin.H{}
	if len(ids) > 0 {
		var friends []model.User
		h.DB.Select("id", "nickname", "color", "level", "signature").Where("id IN ?", ids).Find(&friends)
		for _, u := range friends {
			out = append(out, gin.H{"id": u.ID, "nickname": u.Nickname, "color": u.Color, "level": u.Level, "signature": u.Signature})
		}
	}
	resp.OK(c, out)
}

func (h *SpaceHandler) AlbumList(c *gin.Context) {	userID, ok := positiveQuery(c, "user_id")
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
	viewer := middleware.GetUID(c)
	q := h.DB.Model(&model.SpaceMessage{}).
		Where("to_user_id = ? AND status = 1", userID).
		Where("(mtype = 0 OR from_user_id = ? OR ? = 0)", viewer, viewer)
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
		Mtype   int    `json:"mtype"` // 0公开 1悄悄话
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "留言不能为空")
		return
	}
	if req.Mtype != 1 {
		req.Mtype = 0
	}
	if !verifyTime("spacemsg_"+strconv.Itoa(int(uid)), 5) {
		resp.ParamError(c, "留言太频繁了，歇 5 秒再来")
		return
	}
	msg := model.SpaceMessage{ToUserID: uint(toUserID), FromUserID: uid, Mtype: req.Mtype, Content: req.Content, Status: 1}
	h.DB.Create(&msg)
	h.DB.Model(&model.Home{}).Where("user_id = ?", toUserID).UpdateColumn("messages", gorm.Expr("messages + 1"))
	addHomeNews(h.DB, uid, uint(toUserID), 104, msg.ID, "给你留言："+req.Content)
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

// ---- 相册照片（用户端，诺哈 wap_blog_album） ----

// PhotoList 相册照片列表（未开通/关闭返回空）
func (h *SpaceHandler) PhotoList(c *gin.Context) {
	albumID, ok := positiveParam(c, "albumId")
	if !ok {
		return
	}
	var album model.Album
	if err := h.DB.First(&album, albumID).Error; err != nil {
		resp.NotFound(c, "相册不存在")
		return
	}
	var photos []model.Photo
	h.DB.Where("album_id = ?", albumID).Order("created_at DESC").Limit(200).Find(&photos)
	out := make([]gin.H, 0, len(photos))
	for _, p := range photos {
		out = append(out, gin.H{"id": p.ID, "caption": p.Caption, "format": p.Format, "clicks": p.Clicks, "photo_base64": p.PhotoBase64, "created_at": p.CreatedAt})
	}
	resp.OK(c, gin.H{"album_id": album.ID, "album_name": album.Name, "count": album.Count, "list": out})
}

// PhotoAdd 上传照片（相册主人；base64 data URI 存储，参照头像方案）
func (h *SpaceHandler) PhotoAdd(c *gin.Context) {
	uid := middleware.GetUID(c)
	albumID, ok := positiveParam(c, "albumId")
	if !ok {
		return
	}
	var album model.Album
	if err := h.DB.Where("id = ? AND user_id = ?", albumID, uid).First(&album).Error; err != nil {
		resp.Forbidden(c, "只能往自己的相册传照片")
		return
	}
	var req struct {
		Caption    string `json:"caption" binding:"max=100"`
		PhotoBase64 string `json:"photo_base64" binding:"required"`
		Format     string `json:"format"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请选择要上传的相片")
		return
	}
	if len(req.PhotoBase64) > 3*1024*1024 {
		resp.ParamError(c, "图片太大，请压缩到 3MB 以内")
		return
	}
	if req.Format == "" {
		req.Format = "jpg"
	}
	p := model.Photo{AlbumID: album.ID, UserID: uid, Caption: req.Caption, Format: req.Format, PhotoBase64: req.PhotoBase64, Sizes: len(req.PhotoBase64)}
	h.DB.Create(&p)
	h.DB.Model(&album).UpdateColumn("count", gorm.Expr("count + 1"))
	h.DB.Model(&model.Home{}).Where("user_id = ?", uid).UpdateColumn("moods", gorm.Expr("moods + 1"))
	addHomeNews(h.DB, uid, 0, 103, p.ID, "上传了新照片"+req.Caption)
	resp.OK(c, gin.H{"id": p.ID})
}

// PhotoDel 删除照片（主人或管理员）
func (h *SpaceHandler) PhotoDel(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	uid := middleware.GetUID(c)
	var p model.Photo
	if err := h.DB.First(&p, id).Error; err != nil {
		resp.NotFound(c, "照片不存在")
		return
	}
	if p.UserID != uid && !spaceIsAdmin(h.DB, uid) {
		resp.Forbidden(c, "只能删除自己的照片")
		return
	}
	h.DB.Delete(&model.Photo{}, id)
	h.DB.Model(&model.Album{}).Where("id = ?", p.AlbumID).UpdateColumn("count", gorm.Expr("GREATEST(count - 1, 0)"))
	resp.OK(c, nil)
}

// PhotoClick 照片浏览+1（返回原图）
func (h *SpaceHandler) PhotoClick(c *gin.Context) {
	id, ok := positiveParam(c, "id")
	if !ok {
		return
	}
	var p model.Photo
	if err := h.DB.First(&p, id).Error; err != nil {
		resp.NotFound(c, "照片不存在")
		return
	}
	h.DB.Model(&p).UpdateColumn("clicks", gorm.Expr("clicks + 1"))
	resp.OK(c, gin.H{"id": p.ID, "caption": p.Caption, "photo_base64": p.PhotoBase64, "clicks": p.Clicks + 1, "created_at": p.CreatedAt})
}

// ---- 空间日志分类（诺哈 wap_blog_article_category） ----

// ArticleCatList 我的日志分类
func (h *SpaceHandler) ArticleCatList(c *gin.Context) {
	uid := middleware.GetUID(c)
	var cats []gin.H
	h.DB.Model(&model.Article{}).Where("user_id = ?", uid).
		Select("cat_id, COUNT(*) as cnt").
		Group("cat_id").Order("cat_id ASC").Scan(&cats)
	resp.OK(c, cats)
}

// ArticleCatAdd 新增日志分类（以文章分类 id 形式：article 表 cat_id 复用，这里用 cat_id 作为分类，默认 0=未分类）
// 诺哈为独立分类表；此处简化为前端输入的任意分类名 → 记录到 settings？改为直接由 cat_id 对应日志分类名。
func (h *SpaceHandler) ArticleCatAdd(c *gin.Context) {
	// 预留：分类名存入 settings 表（key=blog_cat_{uid}_{name}），返回 id
	uid := middleware.GetUID(c)
	var req struct {
		Name string `json:"name" binding:"required,max=20"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "分类名不能为空")
		return
	}
	key := "blog_cat_" + strconv.Itoa(int(uid)) + "_" + req.Name
	h.DB.Where(&model.Setting{Key: key}).FirstOrCreate(&model.Setting{Key: key, Value: req.Name})
	resp.OK(c, gin.H{"name": req.Name})
}

// ArticleCatName 按 (user_id, cat_id) 反查分类名
func articleCatName(db *gorm.DB, userID, catID uint) string {
	if catID == 0 {
		return "未分类"
	}
	var setting model.Setting
	if err := db.Where("`key` = ?", "blog_cat_"+strconv.FormatUint(uint64(userID), 10)+"_"+strconv.FormatUint(uint64(catID), 10)).First(&setting).Error; err == nil {
		return setting.Value
	}
	return "分类" + strconv.FormatUint(uint64(catID), 10)
}

// ---- 空间日志评论（诺哈 wap_blog_article_comment） ----

// ArticleCommentList 日志评论列表
func (h *SpaceHandler) ArticleCommentList(c *gin.Context) {
	id, ok := positiveParam(c, "id")
	if !ok {
		return
	}
	var list []model.ArticleComment
	h.DB.Where("article_id = ? AND status = 1", id).Order("created_at ASC").Limit(100).Find(&list)
	type row struct {
		model.ArticleComment
		Nickname string `json:"nickname"`
		Color    string `json:"color"`
	}
	out := make([]row, 0, len(list))
	for _, cm := range list {
		r := row{ArticleComment: cm}
		var u model.User
		h.DB.Select("nickname,color").First(&u, cm.UserID)
		r.Nickname = u.Nickname
		r.Color = u.Color
		out = append(out, r)
	}
	resp.OK(c, out)
}

// ArticleCommentAdd 日志评论
func (h *SpaceHandler) ArticleCommentAdd(c *gin.Context) {
	uid := middleware.GetUID(c)
	id, ok := positiveParam(c, "id")
	if !ok {
		return
	}
	var req struct {
		Content string `json:"content" binding:"required,max=300"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "评论不能为空")
		return
	}
	var a model.Article
	if err := h.DB.Where("id = ? AND status = 1", id).First(&a).Error; err != nil {
		resp.NotFound(c, "日志不存在")
		return
	}
	if a.Atype == 1 && a.UserID != uid {
		resp.Forbidden(c, "该日志不公开，无法评论")
		return
	}
	h.DB.Create(&model.ArticleComment{ArticleID: a.ID, UserID: uid, Content: req.Content, Status: 1})
	resp.OK(c, nil)
}

func spaceIsAdmin(db *gorm.DB, uid uint) bool {
	for _, code := range middleware.UserPermissionCodes(db, uid) {
		if code == "admin:access" {
			return true
		}
	}
	return false
}


