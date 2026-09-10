package router

import (
	"net/http"
	"os"
	"path/filepath"
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
	marryH := &handler.MarriageHandler{DB: db}
	blackH := &handler.BlackHandler{DB: db}
	fnewsH := &handler.FriendNewsHandler{DB: db}
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
	msH := &handler.MoneyShopHandler{DB: db}
	rankH := &handler.RankHandler{DB: db, Secret: cfg.Jwt.Secret}
	hlH := &handler.HomeLevelHandler{DB: db}
	achH := &handler.AchieveHandler{DB: db}
	ttouH := &handler.TtouHandler{DB: db}
	gardenH := &handler.GardenHandler{DB: db}
	farmH := &handler.FarmHandler{DB: db}
	parkH := &handler.ParkHandler{DB: db}
	itH := &handler.InteractHandler{DB: db}
	homeH := &handler.HomeHandler{DB: db}
	contactH := &handler.ContactHandler{DB: db}
	guestH := &handler.GuestHandler{DB: db}
	saH := &handler.SiteArticleHandler{DB: db}
	shopH := &handler.ShopHandler{DB: db}
	actH := &handler.ActivityHandler{DB: db}
	yqH := &handler.YouQuanHandler{DB: db}

	jwtM := middleware.JWTAuth(db, cfg.Jwt.Secret)
	optAuth := middleware.OptionalAuth(db, cfg.Jwt.Secret)
	perm := middleware.RequirePerm

	api := r.Group("/api")
	{
		// 公开接口
		api.POST("/auth/register", authH.Register)
		api.POST("/auth/login", authH.Login)
		api.GET("/auth/find", authH.FindAccount)
		api.GET("/plaza", plazaH.Index)
		api.GET("/announcements", plazaH.Announcements)
		api.GET("/search", plazaH.Search)
		api.GET("/boards", boardH.Tree)
		api.GET("/boards/:id", boardH.Info)
		api.GET("/boards/:id/threads", boardH.Threads)
		// 同城客栈（复刻诺哈三代同城模块）
		cityH := &handler.CityHandler{DB: db}
		api.GET("/tongcheng", cityH.Home)
		api.GET("/tongcheng/lookup", cityH.Lookup)
		api.GET("/tongcheng/province/:id", cityH.Province)
		api.GET("/tongcheng/city/:id", optAuth, cityH.Detail)
		api.GET("/tongcheng/city/:id/online", optAuth, cityH.Online)
		api.GET("/threads/hot", threadH.Hot)
		// 社区新帖/社区动态（复刻诺哈 topic_new.asp / topic_reply.asp）
		api.GET("/threads/new", threadH.NewList)
		api.GET("/threads/active", threadH.ActiveList)
		// 频道论坛页（复刻诺哈论坛天地：各子版块最新帖）
		api.GET("/boards/:id/forum", boardH.Forum)
		api.GET("/threads/:id", optAuth, threadH.Detail)
		api.GET("/threads/:id/replies", optAuth, threadH.Replies)
		api.GET("/users/:id", userH.Profile)
		api.GET("/badges", badgeH.List)
		api.GET("/badge-presets", badgeH.Presets)
		api.GET("/games", gameH.List)
		api.GET("/privs", resH.Privs)
		// 库存图片取图（管理端上传的 base64 图片）
		api.GET("/res/*path", resH.Serve)
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

		// 店铺商城公开（店铺街/商品详情/评价）
		storeH := &handler.StoreHandler{DB: db}
		api.GET("/stores", storeH.Shops)
		api.GET("/store-goods/:id", storeH.GoodsDetail)
		api.GET("/store-goods/:id/comments", storeH.GoodsComments)

		// 花园活动公开列表
		api.GET("/garden-activities", gardenH.ActivityList)
		api.GET("/plaza-sections", plazaH.Sections)
		// 便民服务（复刻诺哈三代 wap/tool 便民中心）
		toolH := &handler.ToolHandler{DB: db}
		api.GET("/tool/client", toolH.Client)
		api.GET("/tool/ip", toolH.IP)
		api.GET("/tool/weather", toolH.Weather)
		api.GET("/tool/phone", toolH.Phone)
		api.GET("/tool/translate", toolH.Translate)
		api.GET("/tool/name", toolH.Name)
		api.GET("/goods", goodH.List)
		api.GET("/goods/:id", goodH.Detail)
		api.GET("/money-shop", msH.List)
		api.GET("/money-shop/:id", msH.Detail)
		api.GET("/rank", rankH.Top)
		// 活动专区（诺哈 topic_active.asp：活动帖列表）
		api.GET("/activities", actH.List)
		// 书城公开
		api.GET("/books", bookH.Index)
		api.GET("/books/list", bookH.List)
		api.GET("/books/categories", bookH.Categories)
		api.GET("/books/:id", bookH.Detail)
		api.GET("/books/:id/chapters", bookH.Chapters)
		api.GET("/books/:id/comments", bookH.Comments)
		api.GET("/chapters/:id", bookH.Chapter)

		// 空间公开接口
		api.GET("/space/:userId", spaceH.SpaceInfo)
		api.GET("/space/:userId/moods", spaceH.MoodList)
		api.GET("/space/:userId/articles", spaceH.ArticleList)
		api.GET("/space/:userId/albums", spaceH.AlbumList)
		api.GET("/space/:userId/messages", spaceH.SpaceMsgList)
		api.GET("/space/:userId/visitors", spaceH.VisitorList)
		api.GET("/space/:userId/friends", spaceH.SpaceFriends)
		api.GET("/marriage/list", marryH.List)
		api.GET("/space/article/:id", spaceH.ArticleDetail)
		api.GET("/space/article/:id/comments", spaceH.ArticleCommentList)
		api.GET("/space/albums/:albumId/photos", spaceH.PhotoList)
		api.GET("/space/photos/:id", spaceH.PhotoClick)
		api.GET("/space/:userId/files", spaceH.SpaceFileList)
		api.GET("/space/files/:id/download", spaceH.SpaceFileDownload)

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
			// 用户附属信息（诺哈 address/docu/protec/pass/log）
			authed.GET("/me/address", userH.AddressView)
			authed.PUT("/me/address", userH.AddressSave)
			authed.GET("/me/document", userH.DocumentView)
			authed.POST("/me/document", userH.DocumentSave)
			authed.GET("/me/protection", userH.ProtectionView)
			authed.POST("/me/protection", userH.ProtectionSave)
			authed.GET("/me/paypass", userH.PayPassView)
			authed.POST("/me/paypass", userH.PayPassSet)
			authed.GET("/me/logs", userH.MyLogs)
			authed.POST("/boards/:id/threads", boardH.CreateThread)
			authed.PUT("/threads/:id", threadH.Update)
			authed.POST("/threads/:id/replies", threadH.Reply)
			authed.DELETE("/threads/:id", threadH.DeleteThread)
			authed.DELETE("/replies/:id", threadH.DeleteReply)
			authed.POST("/threads/:id/sticky-reply", threadH.StickyReply)
			authed.DELETE("/threads/:id/sticky-reply", threadH.UnstickyReply)
			authed.POST("/threads/:id/poll-vote", threadH.VotePoll)
			authed.POST("/threads/:id/manage", threadH.Manage)
			authed.POST("/threads/:id/audit", threadH.Audit)
			authed.POST("/threads/:id/move", threadH.Move)
			authed.GET("/my-threads", threadH.MyThreads)
			authed.GET("/audit-threads", threadH.AuditList)
			authed.GET("/my-medals", badgeH.MyMedals)
			authed.GET("/attachments/:id/download", threadH.Download)
			authed.POST("/threads/:id/favorite", favH.Toggle)
			authed.GET("/threads/:id/favorite-status", favH.Status)
			authed.GET("/favorite-threads", favH.MyFavorites)
			authed.GET("/my-replies", favH.MyReplies)

			// 店铺商城（登录态：开店/上货/交易/评价，诺哈 wap/shop）
			authed.GET("/stores/my", storeH.MyShop)
			authed.POST("/stores", storeH.ShopCreate)
			authed.GET("/stores/:id", storeH.ShopDetail)
			authed.POST("/store-goods", storeH.GoodsAdd)
			authed.PUT("/store-goods/:id", storeH.GoodsEdit)
			authed.DELETE("/store-goods/:id", storeH.GoodsDel)
			authed.GET("/store-goods-mine", storeH.MyGoods)
			authed.POST("/store-orders", storeH.OrderBuy)
			authed.GET("/store-orders", storeH.MyOrders)
			authed.POST("/store-orders/:id/deliver", storeH.OrderDeliver)
			authed.POST("/store-orders/:id/receive", storeH.OrderReceive)
			authed.POST("/store-orders/:id/cancel", storeH.OrderCancel)
			authed.POST("/store-comments", storeH.CommentAdd)
			authed.POST("/store-comments/:id/reply", storeH.ReplyComment)
			// 书城（登录态：书评/书架）
			authed.POST("/books/:id/comments", bookH.CommentAdd)
			authed.POST("/books/:id/shelf", bookH.ShelfAdd)
			authed.DELETE("/books/:id/shelf", bookH.ShelfDel)
			authed.GET("/me/shelf", bookH.ShelfList)

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
			// 开心农场（对齐诺哈三代 wap/game/farm 玩法）
			authed.GET("/games/farm/view", farmH.View)
			authed.GET("/games/farm/visit", farmH.Visit)
			authed.GET("/games/farm/neighbors", farmH.Neighbors)
			authed.GET("/games/farm/shop", farmH.Shop)
			authed.POST("/games/farm/buy", farmH.Buy)
			authed.GET("/games/farm/bag", farmH.Bag)
			authed.GET("/games/farm/warehouse", farmH.Warehouse)
			authed.POST("/games/farm/plow", farmH.Plow)
			authed.POST("/games/farm/plant", farmH.Plant)
			authed.POST("/games/farm/care/:kind", farmH.Care)
			authed.POST("/games/farm/ppest", farmH.Ppest)
			authed.POST("/games/farm/muck", farmH.Muck)
			authed.POST("/games/farm/trap", farmH.Trap)
			authed.POST("/games/farm/pick", farmH.Pick)
			authed.POST("/games/farm/steal", farmH.Steal)
			authed.POST("/games/farm/sell", farmH.Sell)
			authed.GET("/games/farm/rank", farmH.Rank)
			authed.GET("/games/farm/slaves", farmH.Slaves)
			authed.POST("/games/farm/slave/:kind/:id", farmH.SlaveAct)
			authed.POST("/games/farm/setting", farmH.Setting)
			// 抢车位（对齐诺哈三代 wap/game/car 玩法）
			authed.GET("/games/park/view", parkH.View)
			authed.GET("/games/park/owner", parkH.Owner)
			authed.GET("/games/park/friends", parkH.Friends)
			authed.GET("/games/park/shop", parkH.Shop)
			authed.POST("/games/park/buy", parkH.Buy)
			authed.POST("/games/park/send", parkH.Send)
			authed.POST("/games/park/stop", parkH.Stop)
			authed.POST("/games/park/favor", parkH.Favor)
			authed.POST("/games/park/seal", parkH.Seal)
			authed.GET("/games/park/garage", parkH.Garage)
			authed.GET("/games/park/top", parkH.Top)

			// 我的游戏
			authed.GET("/my-games", gameH.MyList)
			authed.POST("/my-games", gameH.MyAdd)
			authed.DELETE("/my-games/:gameId", gameH.MyRemove)
			authed.POST("/my-games/:gameId/move", gameH.MyMove)

			authed.POST("/signin", signH.Do)
			authed.GET("/signin/info", signH.Info)
			authed.GET("/friends", friendH.List)
			authed.POST("/friends", friendH.Add)
			authed.GET("/friends/status/:id", friendH.Status)
			authed.GET("/friends/news", fnewsH.List)
			authed.POST("/friends/:id/remark", friendH.SetRemark)
			authed.POST("/friends/:id/group", friendH.MoveGroup)
			authed.POST("/friends/apply/:id/handle", friendH.Handle)
			authed.POST("/friends/policy", friendH.SetPolicy)
			authed.DELETE("/friends/:id", friendH.Remove)

			// 黑名单
			authed.GET("/friends/black", blackH.List)
			authed.POST("/friends/black/:id", blackH.Add)
			authed.DELETE("/friends/black/:id", blackH.Remove)

			authed.GET("/messages/conversations", msgH.Conversations)
			authed.GET("/messages/inbox", msgH.Inbox)
			authed.GET("/messages/outbox", msgH.Outbox)
			authed.GET("/messages/with/:id", msgH.With)
			authed.POST("/messages/clear", msgH.Clear)
			authed.DELETE("/messages/:id", msgH.Del)
			authed.GET("/messages/unread", msgH.Unread)
			authed.POST("/messages/read-all", msgH.ReadAll)
			authed.POST("/messages", msgH.Send)
			authed.GET("/chat", chatH.List)
			authed.POST("/chat", chatH.Send)
			authed.GET("/notifications", notifyH.List)
			authed.POST("/notifications/read", notifyH.ReadAll)
			// 婚恋（对齐诺哈 bbs/marriage）
			authed.GET("/marriage/status", marryH.Status)
			authed.POST("/marriage/propose", marryH.Propose)
			authed.POST("/marriage/:id/handle", marryH.Handle)
			authed.POST("/marriage/divorce", marryH.Divorce)

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
			authed.POST("/space/file", spaceH.SpaceFileAdd)
			authed.DELETE("/space/file/:id", spaceH.SpaceFileDel)
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
			authed.GET("/families/mine/favorites", famH.MyFavorites)
			authed.POST("/families", famH.Create)
			authed.POST("/families/:id/join", famH.Join)
			authed.POST("/families/:id/leave", famH.Leave)
			authed.POST("/families/:id/favorite", famH.Favorite)
			authed.POST("/families/:id/members/:userId/remove", famH.RemoveMember)
			authed.PUT("/families/:id/ann", famH.UpdateAnn)
			authed.POST("/families/:id/signin", famH.SignIn)
			authed.POST("/families/:id/tree", famH.Tree)
			authed.POST("/families/:id/battle", famH.Battle)
			authed.GET("/families/:id/ld", famH.Ld)
			authed.POST("/families/:id/ld/fruit", famH.LdFruit)
			authed.POST("/families/:id/ld/pk/:userId", famH.LdPk)
			authed.GET("/families/:id/war", famH.War)
			authed.POST("/families/:id/war/life", famH.WarLife)
			authed.POST("/families/:id/war/pk/:userId", famH.WarPk)
			authed.POST("/families/:id/war/chat", famH.WarChat)

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
			// 友友券中心
			authed.GET("/youquan", yqH.View)
			authed.POST("/youquan/daily", yqH.Daily)
			authed.POST("/youquan/exchange", yqH.Exchange)
			authed.POST("/youquan/transfer", yqH.Transfer)
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
			authed.POST("/noble/gift", nobleH.Gift)
			authed.POST("/goods/:id/buy", goodH.Buy)
			authed.GET("/goods/:id/send-preview", goodH.SendPreview)
			authed.POST("/goods/:id/send", goodH.Send)
			authed.GET("/bag", goodH.Bag)
			authed.POST("/bag/:id/use", goodH.BagUse)
			authed.POST("/money-shop/:id/buy", msH.Buy)
			authed.GET("/money-shop/:id/send-preview", msH.SendPreview)
			authed.POST("/money-shop/:id/send", msH.Send)

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

				// 特权管理（蓝钻/超Q，复刻诺哈 vip：等级配置/贵宾会员/贵宾销售）
				admin.GET("/privileges/stats", perm(db, "admin:access"), nobleH.AdminStats)
				admin.GET("/privileges/levels", perm(db, "admin:access"), nobleH.AdminLevels)
				admin.POST("/privileges/levels", perm(db, "admin:access"), nobleH.AdminLevelCreate)
				admin.PUT("/privileges/levels/:id", perm(db, "admin:access"), nobleH.AdminLevelUpdate)
				admin.DELETE("/privileges/levels/:id", perm(db, "admin:access"), nobleH.AdminLevelDelete)
				admin.GET("/privileges/plans", perm(db, "admin:access"), nobleH.AdminPlans)
				admin.POST("/privileges/plans", perm(db, "admin:access"), nobleH.AdminPlanCreate)
				admin.PUT("/privileges/plans/:id", perm(db, "admin:access"), nobleH.AdminPlanUpdate)
				admin.DELETE("/privileges/plans/:id", perm(db, "admin:access"), nobleH.AdminPlanDelete)
				admin.GET("/privileges/users", perm(db, "admin:access"), nobleH.AdminUsers)
				admin.PUT("/privileges/users/:id", perm(db, "admin:access"), nobleH.AdminUserUpdate)
				admin.POST("/privileges/users/:id/open", perm(db, "admin:access"), nobleH.AdminUserOpen)
				admin.POST("/privileges/users/:id/close", perm(db, "admin:access"), nobleH.AdminUserClose)
				admin.POST("/privileges/expired-clean", perm(db, "admin:access"), nobleH.AdminExpiredClean)
				admin.POST("/privileges/batch", perm(db, "admin:access"), nobleH.AdminBatch)

				// 道具商城管理
				admin.GET("/goods", perm(db, "admin:access"), goodH.AdminList)
				admin.POST("/goods", perm(db, "admin:access"), goodH.AdminCreate)
				admin.PUT("/goods/:id", perm(db, "admin:access"), goodH.AdminUpdate)
				admin.DELETE("/goods/:id", perm(db, "admin:access"), goodH.AdminDelete)

				// 货币商店管理（复刻诺哈 wap_money_shop）
				admin.GET("/money-shops", perm(db, "admin:access"), msH.AdminList)
				admin.POST("/money-shops", perm(db, "admin:access"), msH.AdminCreate)
				admin.PUT("/money-shops/:id", perm(db, "admin:access"), msH.AdminUpdate)
				admin.DELETE("/money-shops/:id", perm(db, "admin:access"), msH.AdminDelete)

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
			admin.GET("/garden-sign-rewards", perm(db, "admin:access"), gardenH.AdminSignRewards)
			admin.PUT("/garden-sign-rewards/:day", perm(db, "admin:access"), gardenH.AdminSignRewardUpdate)
			admin.GET("/garden-sign-stats", perm(db, "admin:access"), gardenH.AdminSignStats)
				// 开心农场管理（种子/化肥/陷阱/用户数据/日志/排行）
				admin.GET("/farm-seeds", perm(db, "admin:access"), farmH.AdminSeeds)
				admin.POST("/farm-seeds", perm(db, "admin:access"), farmH.AdminSeedCreate)
				admin.PUT("/farm-seeds/:id", perm(db, "admin:access"), farmH.AdminSeedUpdate)
				admin.DELETE("/farm-seeds/:id", perm(db, "admin:access"), farmH.AdminSeedDelete)
				admin.GET("/farm-mucks", perm(db, "admin:access"), farmH.AdminMucks)
				admin.POST("/farm-mucks", perm(db, "admin:access"), farmH.AdminMuckCreate)
				admin.PUT("/farm-mucks/:id", perm(db, "admin:access"), farmH.AdminMuckUpdate)
				admin.DELETE("/farm-mucks/:id", perm(db, "admin:access"), farmH.AdminMuckDelete)
				admin.GET("/farm-traps", perm(db, "admin:access"), farmH.AdminTraps)
				admin.POST("/farm-traps", perm(db, "admin:access"), farmH.AdminTrapCreate)
				admin.PUT("/farm-traps/:id", perm(db, "admin:access"), farmH.AdminTrapUpdate)
				admin.DELETE("/farm-traps/:id", perm(db, "admin:access"), farmH.AdminTrapDelete)
				admin.GET("/farm-users", perm(db, "admin:access"), farmH.AdminUsers)
				admin.GET("/farm-users/:uid", perm(db, "admin:access"), farmH.AdminUserDetail)
				admin.PUT("/farm-users/:uid/farm", perm(db, "admin:access"), farmH.AdminFarmEdit)
				admin.PUT("/farm-users/:uid/coins", perm(db, "admin:access"), farmH.AdminCoins)
				admin.PUT("/farm-users/:uid/bag/:target", perm(db, "admin:access"), farmH.AdminBagSet)
				admin.PUT("/farm-users/:uid/land/:id/clear", perm(db, "admin:access"), farmH.AdminLandClear)
				admin.GET("/farm-logs", perm(db, "admin:access"), farmH.AdminLogs)
				admin.GET("/farm-rank", perm(db, "admin:access"), farmH.AdminRank)
				// 抢车位管理（车市/用户数据/日志）
				admin.GET("/park-cars", perm(db, "admin:access"), parkH.AdminCars)
				admin.POST("/park-cars", perm(db, "admin:access"), parkH.AdminCarCreate)
				admin.PUT("/park-cars/:id", perm(db, "admin:access"), parkH.AdminCarUpdate)
				admin.DELETE("/park-cars/:id", perm(db, "admin:access"), parkH.AdminCarDelete)
				admin.GET("/park-users", perm(db, "admin:access"), parkH.AdminUsers)
				admin.PUT("/park-users/:uid", perm(db, "admin:access"), parkH.AdminUserEdit)
				admin.PUT("/park-users/:uid/spots/:id/clear", perm(db, "admin:access"), parkH.AdminSpotClear)
				admin.GET("/park-logs", perm(db, "admin:access"), parkH.AdminLogs)

				// ============ 会员管理 user/（诺哈：会员列表/证件/联系/地址/密保/日志/财务/推荐） ============
				admin.GET("/users", perm(db, "user:manage"), adminH.Users)
				admin.GET("/users/:id/detail", perm(db, "user:manage"), adminH.UserDetail)
				admin.PUT("/users/:id/home", perm(db, "user:manage"), adminH.UserHomeSet)
				admin.GET("/invites", perm(db, "user:manage"), adminH.AdminInvites)
				admin.POST("/invites", perm(db, "user:manage"), adminH.AdminInviteCreate)
				admin.DELETE("/invites/:id", perm(db, "user:manage"), adminH.AdminInviteDelete)
				admin.GET("/wallet-logs", perm(db, "user:manage"), adminH.AdminWalletLogs)
				// 会员子页（诺哈：会员证件/联系/地址/密保/日志）
				admin.GET("/user-docu", perm(db, "user:manage"), adminH.AdminUserDocu)
				admin.DELETE("/user-docu/:id", perm(db, "user:manage"), adminH.AdminUserDocuDelete)
				admin.GET("/user-contacts", perm(db, "user:manage"), adminH.AdminUserContacts)
				admin.PUT("/user-contacts/:id", perm(db, "user:manage"), adminH.AdminUserContactUpdate)
				admin.GET("/user-addresses", perm(db, "user:manage"), adminH.AdminUserAddresses)
				admin.PUT("/user-addresses/:id", perm(db, "user:manage"), adminH.AdminUserAddressUpdate)
				admin.GET("/user-protections", perm(db, "user:manage"), adminH.AdminUserProtections)
				admin.DELETE("/user-protections/:uid", perm(db, "user:manage"), adminH.AdminUserProtectionDelete)
				admin.GET("/user-logs", perm(db, "user:manage"), adminH.AdminUserLogs)

				// ============ 家园管理 home/（诺哈：家园列表/家园访客/游戏管理） ============
				admin.GET("/homes", perm(db, "admin:access"), adminH.AdminHomes)
				admin.GET("/visitors", perm(db, "admin:access"), adminH.AdminVisitors)

				// ============ 信息管理 message/（诺哈：家信列表） ============
				admin.GET("/home-news", perm(db, "admin:access"), adminH.AdminHomeNews)
				admin.DELETE("/home-news/:id", perm(db, "admin:access"), adminH.AdminHomeNewsDel)
				admin.GET("/messages", perm(db, "admin:access"), adminH.AdminMessages)
				admin.DELETE("/messages/:id", perm(db, "admin:access"), adminH.AdminMessageDel)

				// ============ 书城管理 book/（诺哈：小说/章节/书评） ============
				admin.GET("/books", perm(db, "admin:access"), adminH.AdminBooks)
				admin.POST("/books", perm(db, "admin:access"), adminH.AdminBookCreate)
				admin.PUT("/books/:id", perm(db, "admin:access"), adminH.AdminBookUpdate)
				admin.DELETE("/books/:id", perm(db, "admin:access"), adminH.AdminBookDel)
				admin.GET("/books/:id/chapters", perm(db, "admin:access"), adminH.AdminBookChapters)
				admin.POST("/books/:id/chapters", perm(db, "admin:access"), adminH.AdminChapterCreate)
				admin.PUT("/book-chapters/:id", perm(db, "admin:access"), adminH.AdminChapterUpdate)
				admin.DELETE("/book-chapters/:id", perm(db, "admin:access"), adminH.AdminChapterDel)
				admin.GET("/book-comments", perm(db, "admin:access"), adminH.AdminBookComments)
				admin.PUT("/book-comments/:id", perm(db, "admin:access"), adminH.AdminBookCommentUpdate)
				admin.DELETE("/book-comments/:id", perm(db, "admin:access"), adminH.AdminBookCommentDel)

				// ============ 系统配置 config/（站点设置） ============
				admin.GET("/site-config", perm(db, "admin:access"), adminH.AdminSiteConfig)
				admin.PUT("/site-config", perm(db, "admin:access"), adminH.AdminSiteConfigSave)

				// ============ 社区管理 bbs/（诺哈：帖子管理/回复管理/恢复帖子/黑名单） ============
				admin.GET("/threads/recycle", perm(db, "thread:manage"), adminH.AdminThreadRecycle)
				admin.PUT("/threads/:id/restore", perm(db, "thread:manage"), adminH.AdminThreadRestore)

				// ============ 商城管理 shop/（诺哈：商品管理/订单/评论） ============
				admin.GET("/shops", perm(db, "admin:access"), adminH.AdminShops)
				admin.GET("/shop-goods", perm(db, "admin:access"), adminH.AdminShopGoods)
				admin.PUT("/shop-goods/:id/status", perm(db, "admin:access"), adminH.AdminShopGoodsStatus)
				admin.DELETE("/shop-goods/:id", perm(db, "admin:access"), adminH.AdminShopGoodsDel)
				admin.GET("/shop-orders", perm(db, "admin:access"), adminH.AdminShopOrders)
				admin.GET("/shop-comments", perm(db, "admin:access"), adminH.AdminShopComments)
				admin.DELETE("/shop-comments/:id", perm(db, "admin:access"), adminH.AdminShopCommentDel)

				// ============ 文章管理 article/（诺哈：文章配置/文章分类） ============
				admin.GET("/site-articles", perm(db, "admin:access"), adminH.AdminSiteArticles)
				admin.DELETE("/site-articles/:id", perm(db, "admin:access"), adminH.AdminSiteArticleDel)
				admin.GET("/article-categories", perm(db, "admin:access"), adminH.AdminArticleCategories)
				admin.POST("/article-categories", perm(db, "admin:access"), adminH.AdminArticleCategoryCreate)
				admin.PUT("/article-categories/:id", perm(db, "admin:access"), adminH.AdminArticleCategoryUpdate)
				admin.DELETE("/article-categories/:id", perm(db, "admin:access"), adminH.AdminArticleCategoryDel)

				// ============ 留言管理 guest/ ============
				admin.GET("/guestbook", perm(db, "admin:access"), adminH.AdminGuestbook)
				admin.DELETE("/guestbook/:id", perm(db, "admin:access"), adminH.AdminGuestbookDel)
				admin.GET("/wallets", perm(db, "user:manage"), adminH.Wallets)
				admin.PUT("/wallets/:id", perm(db, "user:manage"), adminH.WalletSet)

				// 家族 / 同城 / T台秀
				admin.GET("/families", perm(db, "admin:access"), adminH.Families)
				admin.PUT("/families/:id/status", perm(db, "admin:access"), adminH.FamilyStatus)
				admin.PUT("/families/:id/ann", perm(db, "admin:access"), adminH.FamilyAnn)
				admin.GET("/families/pending", perm(db, "admin:access"), adminH.PendingFamilies)
				admin.PUT("/families/:id/feature", perm(db, "admin:access"), adminH.FamilyFeature)
				admin.PUT("/families/:id/review", perm(db, "admin:access"), adminH.FamilyReview)
				admin.GET("/tongcheng", perm(db, "admin:access"), cityH.AdminTree)
				admin.POST("/tongcheng/provinces", perm(db, "board:manage"), cityH.AdminCreateProvince)
				admin.PUT("/tongcheng/provinces/:id", perm(db, "board:manage"), cityH.AdminUpdateProvince)
				admin.DELETE("/tongcheng/provinces/:id", perm(db, "board:manage"), cityH.AdminDeleteProvince)
				admin.POST("/tongcheng/cities", perm(db, "board:manage"), cityH.AdminCreateCity)
				admin.PUT("/tongcheng/cities/:id", perm(db, "board:manage"), cityH.AdminUpdateCity)
				admin.DELETE("/tongcheng/cities/:id", perm(db, "board:manage"), cityH.AdminDeleteCity)
				admin.GET("/tongcheng/managers/:id", perm(db, "admin:access"), cityH.AdminManagers)
				admin.POST("/tongcheng/managers", perm(db, "board:manage"), cityH.AdminManagerAdd)
				admin.DELETE("/tongcheng/managers/:id", perm(db, "board:manage"), cityH.AdminManagerDelete)
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
				admin.POST("/users/batch-delete", perm(db, "user:manage"), adminH.UsersBatchDelete)
				admin.GET("/phones", perm(db, "user:manage"), adminH.Phones)
				admin.POST("/phones/:id/audit", perm(db, "user:manage"), adminH.PhoneAudit)
				admin.DELETE("/phones/:id", perm(db, "user:manage"), adminH.PhoneDelete)

				admin.GET("/badges", perm(db, "badge:manage"), badgeH.AdminList)
				admin.POST("/badges", perm(db, "badge:manage"), badgeH.Create)
				admin.PUT("/badges/:id", perm(db, "badge:manage"), badgeH.Update)
				admin.DELETE("/badges/:id", perm(db, "badge:manage"), badgeH.Delete)
				admin.GET("/user-badges", perm(db, "badge:manage"), badgeH.AdminUserBadges)
				admin.POST("/user-badges/grant", perm(db, "badge:manage"), badgeH.GrantOne)
				admin.DELETE("/user-badges/:id", perm(db, "badge:manage"), badgeH.AdminUserBadgeDel)

				admin.GET("/games", perm(db, "game:manage"), gameH.AdminList)
				admin.POST("/games", perm(db, "game:manage"), gameH.Create)
				admin.PUT("/games/:id", perm(db, "game:manage"), gameH.Update)
				admin.DELETE("/games/:id", perm(db, "game:manage"), gameH.Delete)

				admin.GET("/resources", perm(db, "admin:access"), resH.List)
				admin.PUT("/resources/:id", perm(db, "admin:access"), resH.Update)
				admin.POST("/resources/sync", perm(db, "admin:access"), resH.Sync)
				admin.POST("/resources/upload", perm(db, "admin:access"), resH.Upload)
				admin.DELETE("/resources/:id", perm(db, "admin:access"), resH.Delete)

				admin.GET("/boards", perm(db, "board:manage"), adminH.Boards)
				admin.POST("/boards", perm(db, "board:manage"), adminH.CreateBoard)
				admin.PUT("/boards/:id", perm(db, "board:manage"), adminH.UpdateBoard)
				admin.DELETE("/boards/:id", perm(db, "board:manage"), adminH.DeleteBoard)
				admin.GET("/board-categories", perm(db, "board:manage"), adminH.BoardCategories)
				admin.POST("/board-categories", perm(db, "board:manage"), adminH.CreateBoardCategory)
				admin.PUT("/board-categories/:id", perm(db, "board:manage"), adminH.UpdateBoardCategory)
				admin.DELETE("/board-categories/:id", perm(db, "board:manage"), adminH.DeleteBoardCategory)
				admin.GET("/board-members", perm(db, "board:manage"), adminH.BoardMembers)
				admin.POST("/board-members", perm(db, "board:manage"), adminH.BoardMemberAdd)
				admin.DELETE("/board-members/:boardId/:userId", perm(db, "board:manage"), adminH.BoardMemberRemove)
				admin.GET("/word-filters", perm(db, "board:manage"), adminH.WordFilters)
				admin.POST("/word-filters", perm(db, "board:manage"), adminH.CreateWordFilter)
				admin.PUT("/word-filters/:id", perm(db, "board:manage"), adminH.UpdateWordFilter)
				admin.DELETE("/word-filters/:id", perm(db, "board:manage"), adminH.DeleteWordFilter)

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
	// 静态图片素材：生产托管 web/dist/static；开发模式回退到 web/public/static（源码目录）
	// 管理端 dev server 将 /static 代理到 8080，此路由保证任意模式下 /static 均可用
	if cfg.Server.WebDir != "" {
		staticDir := filepath.Join(cfg.Server.WebDir, "static")
		if _, err := os.Stat(staticDir); err != nil {
			alt := filepath.Clean(filepath.Join(cfg.Server.WebDir, "..", "public", "static"))
			if st, err2 := os.Stat(alt); err2 == nil && st.IsDir() {
				staticDir = alt
			}
		}
		if st, err := os.Stat(staticDir); err == nil && st.IsDir() {
			r.Static("/static", staticDir)
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
