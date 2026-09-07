package router

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/config"
	"qqjiayuan/server/internal/handler"
	"qqjiayuan/server/internal/middleware"
)

func Setup(db *gorm.DB, cfg *config.Config) *gin.Engine {
	r := gin.Default()
	r.Use(middleware.CORS())

	authH := &handler.AuthHandler{DB: db, Secret: cfg.Jwt.Secret, ExpH: cfg.Jwt.ExpireHours}
	userH := &handler.UserHandler{DB: db, StaticDir: cfg.Server.WebDir + "/static"}
	boardH := &handler.BoardHandler{DB: db}
	threadH := &handler.ThreadHandler{DB: db}
	plazaH := &handler.PlazaHandler{DB: db}
	signH := &handler.SignHandler{DB: db}
	friendH := &handler.FriendHandler{DB: db}
	msgH := &handler.MessageHandler{DB: db}
	chatH := &handler.ChatHandler{DB: db}
	notifyH := &handler.NotifyHandler{DB: db}
	adminH := &handler.AdminHandler{DB: db}
	badgeH := &handler.BadgeHandler{DB: db}
	gameH := &handler.GameHandler{DB: db}
	resH := &handler.ResourceHandler{DB: db, StaticDir: cfg.Server.WebDir + "/static"}
	spaceH := &handler.SpaceHandler{DB: db}
	moodH := &handler.MoodHandler{DB: db}
	ecoH := &handler.EconomyHandler{DB: db}
	famH := &handler.FamilyHandler{DB: db, Secret: cfg.Jwt.Secret}
	bookH := &handler.BookHandler{DB: db}
	favH := &handler.FavoriteHandler{DB: db}
	fgH := &handler.FriendGroupHandler{DB: db}
	nobleH := &handler.NobleHandler{DB: db}
	goodH := &handler.GoodHandler{DB: db}
	rankH := &handler.RankHandler{DB: db, Secret: cfg.Jwt.Secret}
	hlH := &handler.HomeLevelHandler{DB: db}
	achH := &handler.AchieveHandler{DB: db}
	ttouH := &handler.TtouHandler{DB: db}
	gardenH := &handler.GardenHandler{DB: db}
	itH := &handler.InteractHandler{DB: db}
	homeH := &handler.HomeHandler{DB: db}
	contactH := &handler.ContactHandler{DB: db}
	guestH := &handler.GuestHandler{DB: db}
	saH := &handler.SiteArticleHandler{DB: db}
	shopH := &handler.ShopHandler{DB: db}

	jwtM := middleware.JWTAuth(db, cfg.Jwt.Secret)
	perm := middleware.RequirePerm

	api := r.Group("/api")
	{
		// 公开接口
		api.POST("/auth/register", authH.Register)
		api.POST("/auth/login", authH.Login)
		api.GET("/auth/find", authH.FindAccount)
		api.GET("/plaza", plazaH.Index)
		api.GET("/search", plazaH.Search)
		api.GET("/boards", boardH.Tree)
		api.GET("/boards/:id", boardH.Info)
		api.GET("/boards/:id/threads", boardH.Threads)
		api.GET("/threads/:id", threadH.Detail)
		api.GET("/users/:id", userH.Profile)
		api.GET("/badges", badgeH.List)
		api.GET("/badge-presets", badgeH.Presets)
		api.GET("/games", gameH.List)
		api.GET("/privs", resH.Privs)
		// 家族列表/详情/动态公开（未登录仅浏览，无角色）
		api.GET("/families", famH.List)
		api.GET("/families/categories", famH.Categories)
		api.GET("/families/search", famH.Search)
		api.GET("/families/top", famH.Top)
		api.GET("/families/:id", famH.Detail)
		api.GET("/families/:id/activities", famH.FamilyActivities)
		api.GET("/families/:id/forum", famH.Forum)
		api.GET("/families/:id/hot", famH.Hot)
		api.GET("/families/activities", famH.Activities)
		api.GET("/families/featured", famH.FeatureList)
		api.GET("/families/pending", famH.Pending)
		api.GET("/families/activity-threads", famH.ActivityThreads)

		// 花园活动公开列表
		api.GET("/garden-activities", gardenH.ActivityList)
		api.GET("/plaza-sections", plazaH.Sections)
		api.GET("/goods", goodH.List)
		api.GET("/rank", rankH.Top)
		// 书城公开
		api.GET("/books", bookH.Index)
		api.GET("/books/list", bookH.List)
		api.GET("/books/categories", bookH.Categories)
		api.GET("/books/:id", bookH.Detail)

		// 空间公开接口
		api.GET("/space/:userId", spaceH.SpaceInfo)
		api.GET("/space/:userId/moods", spaceH.MoodList)
		api.GET("/space/:userId/articles", spaceH.ArticleList)
		api.GET("/space/:userId/albums", spaceH.AlbumList)
		api.GET("/space/:userId/messages", spaceH.SpaceMsgList)
		api.GET("/space/:userId/visitors", spaceH.VisitorList)
		api.GET("/space/article/:id", spaceH.ArticleDetail)
		api.GET("/space/article/:id/comments", spaceH.ArticleCommentList)
		api.GET("/space/albums/:albumId/photos", spaceH.PhotoList)
		api.GET("/space/photos/:id", spaceH.PhotoClick)

		// 家园（诺哈：他人家园 / 串门）
		api.GET("/home/other/:userId", homeH.Other)
		api.GET("/home/visit", homeH.Visit)

		// 留言本（全站）
		api.GET("/guestbook", guestH.List)

		// 文章（社区专栏）
		api.GET("/articles", saH.List)
		api.GET("/articles/categories", saH.Categories)
		api.GET("/articles/:id", saH.Detail)
		api.GET("/articles/:id/comments", saH.Comments)

		// 商店市场（公开浏览）
		api.GET("/market", shopH.GoodsList)
		api.GET("/market/categories", shopH.Categories)
		api.GET("/market/:id", shopH.GoodsDetail)

		authed := api.Group("/", jwtM)
		{
			authed.GET("/auth/me", authH.Me)
			authed.PUT("/auth/password", authH.ChangePassword)
			authed.PUT("/users/me", userH.UpdateMe)
			authed.POST("/boards/:id/threads", boardH.CreateThread)
			authed.PUT("/threads/:id", threadH.Update)
			authed.POST("/threads/:id/replies", threadH.Reply)
			authed.DELETE("/threads/:id", threadH.DeleteThread)
			authed.DELETE("/replies/:id", threadH.DeleteReply)
			authed.POST("/threads/:id/favorite", favH.Toggle)
			authed.GET("/threads/:id/favorite-status", favH.Status)
			authed.GET("/favorite-threads", favH.MyFavorites)
			authed.GET("/my-replies", favH.MyReplies)

			// 帖子互动：赞/踩/打赏/送花/分享/举报
			authed.POST("/threads/:id/vote", itH.Vote)
			authed.GET("/threads/:id/interact-status", itH.Status)
			authed.POST("/replies/:id/like", itH.ReplyLike)
			authed.POST("/threads/:id/gift", itH.Gift)
			authed.POST("/threads/:id/flower", itH.Flower)
			authed.POST("/threads/:id/share", itH.Share)
			authed.POST("/reports", itH.CreateReport)

			// 好友分组
			authed.GET("/friend-groups", fgH.List)
			authed.POST("/friend-groups", fgH.Create)
			authed.DELETE("/friend-groups/:id", fgH.Delete)
			authed.POST("/friend-groups/:id/friends", fgH.AddFriend)
			authed.DELETE("/friend-groups/:id/friends/:friendId", fgH.RemoveFriend)

			// 游戏：魔法花园（对齐诺哈三代 ASP 版玩法）
			authed.GET("/games/garden/view", gardenH.View)
			authed.GET("/games/garden/visit", gardenH.Visit)
			authed.GET("/games/garden/shop", gardenH.Shop)
			authed.POST("/games/garden/buy", gardenH.Buy)
			authed.POST("/games/garden/sow", gardenH.Sow)
			authed.POST("/games/garden/care/:kind", gardenH.Care)
			authed.POST("/games/garden/harvest", gardenH.Harvest)
			authed.POST("/games/garden/delplot", gardenH.DelPlot)
			authed.POST("/games/garden/pick", gardenH.Pick)
			authed.POST("/games/garden/addland", gardenH.AddLand)
			authed.GET("/games/garden/basket", gardenH.Basket)
			authed.GET("/games/garden/bottle", gardenH.Bottle)
			authed.POST("/games/garden/gift", gardenH.Gift)
			authed.GET("/games/garden/giftlog", gardenH.GiftLog)
			authed.GET("/games/garden/room", gardenH.Room)
			authed.POST("/games/garden/mix", gardenH.Mix)
			authed.GET("/games/garden/map", gardenH.MapList)
			authed.POST("/games/garden/setting", gardenH.Setting)
			authed.GET("/games/garden/msgs", gardenH.Msgs)
			authed.GET("/games/garden/elves", gardenH.Elves)
			authed.GET("/games/garden/rank", gardenH.Rank)
			authed.GET("/games/garden/sign-status", gardenH.SignStatus)
			authed.POST("/games/garden/sign", gardenH.Sign)
			authed.POST("/games/garden/activity-submit", gardenH.SubmitActivity)

			// 我的游戏
			authed.GET("/my-games", gameH.MyList)
			authed.POST("/my-games", gameH.MyAdd)
			authed.DELETE("/my-games/:gameId", gameH.MyRemove)

			authed.POST("/signin", signH.Do)
			authed.GET("/signin/info", signH.Info)
			authed.GET("/friends", friendH.List)
			authed.POST("/friends", friendH.Add)
			authed.POST("/friends/:id/handle", friendH.Handle)
			authed.DELETE("/friends/:id", friendH.Remove)

			authed.GET("/messages/conversations", msgH.Conversations)
			authed.GET("/messages/with/:id", msgH.With)
			authed.POST("/messages", msgH.Send)
			authed.GET("/chat", chatH.List)
			authed.POST("/chat", chatH.Send)
			authed.GET("/notifications", notifyH.List)
			authed.POST("/notifications/read", notifyH.ReadAll)

			// 空间认证接口
			authed.POST("/space", spaceH.OpenSpace)
			authed.PUT("/space", spaceH.UpdateSpace)
			authed.POST("/space/mood", spaceH.MoodAdd)
			authed.DELETE("/space/mood/:id", spaceH.MoodDel)
			authed.POST("/space/mood/:id/comment", spaceH.MoodCommentAdd)
			authed.POST("/space/mood/:id/forward", spaceH.MoodForward)
			authed.POST("/space/article", spaceH.ArticleAdd)
			authed.DELETE("/space/article/:id", spaceH.ArticleDel)
			authed.POST("/space/album", spaceH.AlbumCreate)
			authed.POST("/space/message/:userId", spaceH.SpaceMsgAdd)
			authed.DELETE("/space/message/:id", spaceH.SpaceMsgDel)
			authed.POST("/space/visit/:userId", spaceH.VisitSpace)

			// 家园聚合（诺哈 my_home.asp）与新鲜事
			authed.GET("/home", homeH.View)
			authed.GET("/home/news", homeH.NewsList)

			// 我的收藏（诺哈 wap_bbs_favor 泛化）
			authed.GET("/favorites", homeH.FavList)
			authed.POST("/favorites", homeH.FavAdd)
			authed.DELETE("/favorites/:id", homeH.FavDel)

			// 邀请开通家园（诺哈 invite.asp）
			authed.GET("/invite", homeH.InviteInfo)

			// 通讯录（诺哈 contact.asp）+ QQ 绑定
			authed.GET("/contact", contactH.View)
			authed.POST("/contact", contactH.Save)
			authed.POST("/contact/qq", contactH.SaveQQ)

			// 空间日志分类 / 相册照片 / 日志评论（诺哈 blog 子模块）
			authed.GET("/space/article-categories", spaceH.ArticleCatList)
			authed.POST("/space/article-categories", spaceH.ArticleCatAdd)
			authed.POST("/space/albums/:albumId/photos", spaceH.PhotoAdd)
			authed.DELETE("/space/photos/:id", spaceH.PhotoDel)
			authed.POST("/space/article/:id/comment", spaceH.ArticleCommentAdd)

			// 留言本（诺哈 guest.asp）
			authed.POST("/guestbook", guestH.Add)
			authed.POST("/guestbook/:id/unlock", guestH.Unlock)
			authed.POST("/guestbook/:id/reply", perm(db, "admin:access"), guestH.Reply)
			authed.DELETE("/guestbook/:id", guestH.Del)

			// 文章（诺哈 article.asp 社区专栏）
			authed.POST("/articles", saH.Add)
			authed.POST("/articles/:id/comments", saH.CommentAdd)
			authed.DELETE("/articles/:id", saH.Del)

			// 商店（诺哈 shop.asp C2C 道具买卖）
			authed.GET("/shop/mine", shopH.MyShop)
			authed.POST("/shop", shopH.OpenShop)
			authed.POST("/market", shopH.GoodsAdd)
			authed.PUT("/market/:id", shopH.GoodsUpdate)
			authed.DELETE("/market/:id", shopH.GoodsDelete)
			authed.POST("/market/:id/order", shopH.OrderCreate)
			authed.POST("/shop/orders/:id/pay", shopH.OrderPay)
			authed.POST("/shop/orders/:id/ship", shopH.OrderShip)
			authed.POST("/shop/orders/:id/receive", shopH.OrderReceive)
			authed.POST("/shop/orders/:id/cancel", shopH.OrderCancel)
			authed.GET("/shop/orders", shopH.MyOrders)
			authed.POST("/market/:id/comments", shopH.CommentAdd)
			authed.POST("/shop/comments/:id/reply", shopH.CommentReply)

			// 我的心情（家园个人动态，不依赖空间）
			authed.GET("/moods", moodH.List)
			authed.POST("/moods", moodH.Add)
			authed.DELETE("/moods/:id", moodH.Del)
			authed.GET("/moods/latest", moodH.Latest)

			// 社区经济小助手：银行 / 打工 / 每日星运 / 幸运猜数字
			// 家族系统
			authed.GET("/families/mine", famH.Mine)
			authed.POST("/families", famH.Create)
			authed.POST("/families/:id/join", famH.Join)
			authed.POST("/families/:id/leave", famH.Leave)
			authed.PUT("/families/:id/ann", famH.UpdateAnn)
			authed.POST("/families/:id/signin", famH.SignIn)
			authed.POST("/families/:id/tree", famH.Tree)
			authed.POST("/families/:id/battle", famH.Battle)

			authed.GET("/home-level", hlH.View)
			authed.GET("/achieve", achH.View)
			authed.GET("/me/avatar", userH.MyAvatar)
			authed.POST("/me/avatar", userH.UploadAvatar)
			authed.POST("/me/avatar/qq", userH.QqAvatar)
			authed.POST("/ttou/apply", ttouH.Apply)
			authed.POST("/ttou/worship", ttouH.Worship)
			authed.GET("/avatar/presets", userH.AvatarPresets)
			authed.POST("/avatar/presets", userH.SetPresetAvatar)
			authed.GET("/wallet", ecoH.Wallet)
			authed.POST("/wallet/exchange", ecoH.Exchange)
			authed.POST("/wallet/transfer", ecoH.Transfer)
			authed.GET("/bank/view", ecoH.BankView)
			authed.POST("/bank/deposit", ecoH.BankDeposit)
			authed.POST("/bank/withdraw", ecoH.BankWithdraw)
			authed.POST("/bank/interest", ecoH.BankInterest)
			authed.GET("/work/status", ecoH.WorkStatus)
			authed.POST("/work", ecoH.WorkDo)
			authed.GET("/fortune", ecoH.Fortune)
			authed.POST("/lottery", ecoH.Lottery)
			authed.GET("/noble", nobleH.View)
			authed.POST("/noble/activate", nobleH.Activate)
			authed.POST("/goods/:id/buy", goodH.Buy)
			authed.GET("/bag", goodH.Bag)
			authed.POST("/bag/:id/use", goodH.BagUse)

			authed.POST("/dig", ecoH.Dig)
			authed.POST("/charity", ecoH.Charity)
			authed.GET("/charity/rank", ecoH.CharityRank)

			// 管理后台（RBAC 权限点）
			admin := authed.Group("/admin")
			{
				admin.GET("/stats", perm(db, "admin:access"), adminH.Stats)

				// 举报管理
				admin.GET("/reports", perm(db, "thread:manage"), adminH.Reports)
				admin.PUT("/reports/:id", perm(db, "thread:manage"), adminH.ReportHandle)

				// 广场板块开关
				admin.GET("/plaza-sections", perm(db, "admin:access"), plazaH.AdminSections)
				admin.PUT("/plaza-sections/:id", perm(db, "admin:access"), plazaH.AdminSectionUpdate)

				// 特权管理（蓝钻/超Q）
				admin.GET("/privileges/plans", perm(db, "admin:access"), nobleH.AdminPlans)
				admin.POST("/privileges/plans", perm(db, "admin:access"), nobleH.AdminPlanCreate)
				admin.PUT("/privileges/plans/:id", perm(db, "admin:access"), nobleH.AdminPlanUpdate)
				admin.DELETE("/privileges/plans/:id", perm(db, "admin:access"), nobleH.AdminPlanDelete)
				admin.GET("/privileges/users", perm(db, "admin:access"), nobleH.AdminUsers)
				admin.PUT("/privileges/users/:id", perm(db, "admin:access"), nobleH.AdminUserUpdate)   
				admin.POST("/privileges/users/:id/open", perm(db, "admin:access"), nobleH.AdminUserOpen)
				admin.POST("/privileges/batch", perm(db, "admin:access"), nobleH.AdminBatch)

				// 道具商城管理
				admin.GET("/goods", perm(db, "admin:access"), goodH.AdminList)
				admin.POST("/goods", perm(db, "admin:access"), goodH.AdminCreate)
				admin.PUT("/goods/:id", perm(db, "admin:access"), goodH.AdminUpdate)
				admin.DELETE("/goods/:id", perm(db, "admin:access"), goodH.AdminDelete)

				// 花园活动管理
				admin.GET("/garden-activities", perm(db, "admin:access"), gardenH.AdminActivities)
				admin.POST("/garden-activities", perm(db, "admin:access"), gardenH.AdminActCreate)
				admin.PUT("/garden-activities/:id", perm(db, "admin:access"), gardenH.AdminActUpdate)
				admin.DELETE("/garden-activities/:id", perm(db, "admin:access"), gardenH.AdminActDelete)

				// 花园数据维护（种子/图鉴/配方）
				admin.GET("/garden-seeds", perm(db, "admin:access"), gardenH.AdminSeeds)
				admin.POST("/garden-seeds", perm(db, "admin:access"), gardenH.AdminSeedCreate)
				admin.PUT("/garden-seeds/:id", perm(db, "admin:access"), gardenH.AdminSeedUpdate)
				admin.DELETE("/garden-seeds/:id", perm(db, "admin:access"), gardenH.AdminSeedDelete)
				admin.GET("/garden-maps", perm(db, "admin:access"), gardenH.AdminMaps)
				admin.POST("/garden-maps", perm(db, "admin:access"), gardenH.AdminMapCreate)
				admin.PUT("/garden-maps/:id", perm(db, "admin:access"), gardenH.AdminMapUpdate)
				admin.DELETE("/garden-maps/:id", perm(db, "admin:access"), gardenH.AdminMapDelete)
				admin.GET("/garden-mixes", perm(db, "admin:access"), gardenH.AdminMixes)
				admin.POST("/garden-mixes", perm(db, "admin:access"), gardenH.AdminMixCreate)
				admin.PUT("/garden-mixes/:id", perm(db, "admin:access"), gardenH.AdminMixUpdate)
				admin.DELETE("/garden-mixes/:id", perm(db, "admin:access"), gardenH.AdminMixDelete)
				admin.GET("/garden-elves", perm(db, "admin:access"), gardenH.AdminElves)
				admin.POST("/garden-elves", perm(db, "admin:access"), gardenH.AdminElfCreate)
				admin.PUT("/garden-elves/:id", perm(db, "admin:access"), gardenH.AdminElfUpdate)
				admin.DELETE("/garden-elves/:id", perm(db, "admin:access"), gardenH.AdminElfDelete)

				// 花园游戏数据管理（用户数据/日志流水/排行榜）
				admin.GET("/garden-users", perm(db, "admin:access"), gardenH.AdminGardenUsers)
				admin.GET("/garden-users/:uid", perm(db, "admin:access"), gardenH.AdminGardenUserDetail)
				admin.PUT("/garden-users/:uid/garden", perm(db, "admin:access"), gardenH.AdminGardenEdit)
				admin.PUT("/garden-users/:uid/coins", perm(db, "admin:access"), gardenH.AdminGardenCoins)
				admin.PUT("/garden-users/:uid/bag", perm(db, "admin:access"), gardenH.AdminGardenBagSet)
				admin.PUT("/garden-users/:uid/flowers/:target", perm(db, "admin:access"), gardenH.AdminGardenFlowerSet)
				admin.GET("/garden-logs", perm(db, "admin:access"), gardenH.AdminGardenLogs)
				admin.GET("/garden-rank", perm(db, "admin:access"), gardenH.AdminGardenRank)

				admin.GET("/users", perm(db, "user:manage"), adminH.Users)
				admin.PUT("/users/:id/home", perm(db, "user:manage"), adminH.UserHomeSet)
				admin.GET("/wallets", perm(db, "user:manage"), adminH.Wallets)
				admin.PUT("/wallets/:id", perm(db, "user:manage"), adminH.WalletSet)

				// 家族 / 同城 / T台秀
				admin.GET("/families", perm(db, "admin:access"), adminH.Families)
				admin.PUT("/families/:id/status", perm(db, "admin:access"), adminH.FamilyStatus)
				admin.PUT("/families/:id/ann", perm(db, "admin:access"), adminH.FamilyAnn)
				admin.GET("/families/pending", perm(db, "admin:access"), adminH.PendingFamilies)
				admin.PUT("/families/:id/feature", perm(db, "admin:access"), adminH.FamilyFeature)
				admin.PUT("/families/:id/review", perm(db, "admin:access"), adminH.FamilyReview)
				admin.GET("/tongcheng", perm(db, "admin:access"), adminH.Tongcheng)
				admin.GET("/ttou", perm(db, "admin:access"), adminH.Ttou)
				admin.PUT("/ttou", perm(db, "admin:access"), adminH.TtouSet)
				admin.DELETE("/ttou", perm(db, "admin:access"), adminH.TtouClear)
				admin.GET("/ttou/applies", perm(db, "admin:access"), adminH.TtouApplies)
				admin.POST("/ttou/applies/:id/accept", perm(db, "admin:access"), adminH.TtouApplyAccept)
				admin.PUT("/users/:id/status", perm(db, "user:manage"), adminH.UserStatus)
				admin.PUT("/users/:id/password", perm(db, "user:manage"), adminH.ResetPassword)
				admin.PUT("/users/:id/roles", perm(db, "user:manage"), adminH.UserRoles)
				admin.PUT("/users/:id/badges", perm(db, "badge:manage"), badgeH.UserBadges)
				admin.PUT("/users/:id/extras", perm(db, "user:manage"), adminH.UserExtras)

				admin.GET("/badges", perm(db, "badge:manage"), badgeH.AdminList)
				admin.POST("/badges", perm(db, "badge:manage"), badgeH.Create)
				admin.PUT("/badges/:id", perm(db, "badge:manage"), badgeH.Update)
				admin.DELETE("/badges/:id", perm(db, "badge:manage"), badgeH.Delete)

				admin.GET("/games", perm(db, "game:manage"), gameH.AdminList)
				admin.POST("/games", perm(db, "game:manage"), gameH.Create)
				admin.PUT("/games/:id", perm(db, "game:manage"), gameH.Update)
				admin.DELETE("/games/:id", perm(db, "game:manage"), gameH.Delete)

				admin.GET("/resources", perm(db, "admin:access"), resH.List)
				admin.PUT("/resources/:id", perm(db, "admin:access"), resH.Update)
				admin.POST("/resources/sync", perm(db, "admin:access"), resH.Sync)

				admin.GET("/boards", perm(db, "board:manage"), adminH.Boards)
				admin.POST("/boards", perm(db, "board:manage"), adminH.CreateBoard)
				admin.PUT("/boards/:id", perm(db, "board:manage"), adminH.UpdateBoard)
				admin.DELETE("/boards/:id", perm(db, "board:manage"), adminH.DeleteBoard)

				admin.GET("/threads", perm(db, "thread:manage"), adminH.Threads)
				admin.PUT("/threads/:id", perm(db, "thread:manage"), adminH.UpdateThread)
				admin.DELETE("/threads/:id", perm(db, "thread:manage"), adminH.DeleteThread)
				admin.DELETE("/replies/:id", perm(db, "thread:manage"), adminH.DeleteReply)

				admin.GET("/announcements", perm(db, "announcement:manage"), adminH.Announcements)
				admin.POST("/announcements", perm(db, "announcement:manage"), adminH.CreateAnnouncement)
				admin.PUT("/announcements/:id", perm(db, "announcement:manage"), adminH.UpdateAnnouncement)
				admin.DELETE("/announcements/:id", perm(db, "announcement:manage"), adminH.DeleteAnnouncement)

				admin.GET("/roles", perm(db, "role:manage"), adminH.Roles)
				admin.GET("/permissions", perm(db, "role:manage"), adminH.Permissions)
				admin.POST("/roles", perm(db, "role:manage"), adminH.CreateRole)
				admin.PUT("/roles/:id/perms", perm(db, "role:manage"), adminH.UpdateRolePerms)
				admin.DELETE("/roles/:id", perm(db, "role:manage"), adminH.DeleteRole)

				// 空间管理
				admin.GET("/spaces", perm(db, "user:manage"), spaceH.AdminSpaces)
				admin.PUT("/spaces/:id/status", perm(db, "user:manage"), spaceH.AdminSpaceStatus)
				admin.DELETE("/spaces/:id", perm(db, "user:manage"), spaceH.AdminSpaceDelete)
				admin.GET("/spaces/:userId/moods", perm(db, "user:manage"), spaceH.AdminSpaceMoods)
				admin.DELETE("/moods/:id", perm(db, "user:manage"), spaceH.AdminDeleteMood)
				admin.GET("/spaces/:userId/articles", perm(db, "user:manage"), spaceH.AdminSpaceArticles)
				admin.DELETE("/articles/:id", perm(db, "user:manage"), spaceH.AdminDeleteArticle)
				admin.GET("/spaces/:userId/albums", perm(db, "user:manage"), spaceH.AdminSpaceAlbums)
				admin.GET("/spaces/:userId/messages", perm(db, "user:manage"), spaceH.AdminSpaceMessages)
				admin.GET("/spaces/:userId/visitors", perm(db, "user:manage"), spaceH.AdminSpaceVisitors)
				admin.GET("/albums/:id/photos", perm(db, "user:manage"), spaceH.AdminAlbumPhotos)
				admin.DELETE("/photos/:id", perm(db, "user:manage"), spaceH.AdminDeletePhoto)
				admin.DELETE("/albums/:id", perm(db, "user:manage"), spaceH.AdminDeleteAlbum)
				admin.DELETE("/space-messages/:id", perm(db, "user:manage"), spaceH.AdminDeleteSpaceMessage)
			}
		}
	}

	// 生产模式：托管前端构建产物
	if cfg.Server.WebDir != "" {
		if st, err := os.Stat(cfg.Server.WebDir); err == nil && st.IsDir() {
			r.Static("/assets", cfg.Server.WebDir+"/assets")
			r.NoRoute(func(c *gin.Context) {
				path := c.Request.URL.Path
				if path == "/" || !strings.Contains(path, ".") {
					c.File(cfg.Server.WebDir + "/index.html")
					return
				}
				if _, err := os.Stat(cfg.Server.WebDir + path); err == nil {
					c.File(cfg.Server.WebDir + path)
					return
				}
				c.File(cfg.Server.WebDir + "/index.html")
			})
		}
	}
	// 管理系统前端（/admin-ui/，FileServer 对目录根自动回 index.html）
	if cfg.Server.AdminWebDir != "" {
		if st, err := os.Stat(cfg.Server.AdminWebDir); err == nil && st.IsDir() {
			r.Static("/admin-ui", cfg.Server.AdminWebDir)
		}
	}
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok", "data": "家园社区 API 运行中"})
	})
	return r
}
