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
	flaH := &handler.FlaHandler{DB: db}
	famH := &handler.FamilyHandler{DB: db, Secret: cfg.Jwt.Secret}
	bookH := &handler.BookHandler{DB: db}
	favH := &handler.FavoriteHandler{DB: db}
	fgH := &handler.FriendGroupHandler{DB: db}
	nobleH := &handler.NobleHandler{DB: db}
	nameH := &handler.NameHandler{DB: db}
	goodH := &handler.GoodHandler{DB: db}
	msH := &handler.MoneyShopHandler{DB: db}
	rankH := &handler.RankHandler{DB: db, Secret: cfg.Jwt.Secret}
	hlH := &handler.HomeLevelHandler{DB: db}
	achH := &handler.AchieveHandler{DB: db}
	ttouH := &handler.TtouHandler{DB: db}
	gardenH := &handler.GardenHandler{DB: db}
	farmH := &handler.FarmHandler{DB: db}
	parkH := &handler.ParkHandler{DB: db}
	jwtH := &handler.JwtHandler{DB: db}
	hxH := &handler.HxxyHandler{DB: db}
	ezfyH := &handler.EzfyHandler{DB: db}
	itH := &handler.InteractHandler{DB: db}
	homeH := &handler.HomeHandler{DB: db}
	contactH := &handler.ContactHandler{DB: db}
	guestH := &handler.GuestHandler{DB: db}
	saH := &handler.SiteArticleHandler{DB: db}
	shopH := &handler.ShopHandler{DB: db}
	actH := &handler.ActivityHandler{DB: db}
	yqH := &handler.YouQuanHandler{DB: db}
	welfareH := &handler.WelfareHandler{DB: db}

	jwtM := middleware.JWTAuth(db, cfg.Jwt.Secret)
	optAuth := middleware.OptionalAuth(db, cfg.Jwt.Secret)
	perm := middleware.RequirePerm

	api := r.Group("/api")
	{
		// 公开接口
		api.POST("/auth/register", authH.Register)
		api.POST("/auth/login", authH.Login)
		api.GET("/auth/find", authH.FindAccount)
		api.GET("/auth/find/contact", authH.FindByContact)
		api.GET("/auth/find/document", authH.FindByDocument)
		api.GET("/auth/repass/channels", authH.RepassChannels)
		api.POST("/auth/repass/qq", authH.RepassByQQ)
		api.POST("/auth/repass/mail", authH.RepassByMail)
		api.POST("/auth/repass/protection", authH.RepassByProtection)
		api.GET("/fla", flaH.Index)
		// 福利院·慈善基金（领取/名单公开可见）
		api.GET("/welfare", optAuth, welfareH.Index)
		api.GET("/welfare/claims", welfareH.Claims)
		api.GET("/welfare/donations", welfareH.Donations)
		api.GET("/plaza", plazaH.Index)
		// 在线用户（复刻诺哈 online.html：用户/游客混排，游客按 IP 展示）
		api.GET("/online", optAuth, plazaH.Online)
		api.GET("/announcements", plazaH.Announcements)
		// 站点展示信息（页脚 QQ 群号等，后台站点设置 qq_group 可配）
		api.GET("/site-info", plazaH.SiteInfo)
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
		api.GET("/activities/column", actH.Column)
		// 公告帖（诺哈 topic_notice.asp：notice=1 帖子列表）
		api.GET("/notices/threads", actH.NoticeList)
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

			// 个性昵称（复刻 3GQQ name.html：购买/赠送/颜色设置）
			authed.GET("/name", nameH.NameInfo)
			authed.POST("/name/buy", nameH.NameBuy)
			authed.POST("/name/send", nameH.NameSend)
			authed.POST("/name/set", nameH.NameSet)
			authed.PUT("/auth/password", authH.ChangePassword)
			authed.PUT("/users/me", userH.UpdateMe)
			authed.PUT("/me/city", userH.CitySet)
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
			authed.POST("/games/garden/item-use", gardenH.ItemUse)
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
			authed.POST("/games/park/sell", parkH.Sell)
			authed.POST("/games/park/stop", parkH.Stop)
			authed.POST("/games/park/favor", parkH.Favor)
			authed.POST("/games/park/seal", parkH.Seal)
			authed.GET("/games/park/garage", parkH.Garage)
			authed.GET("/games/park/logs", parkH.Logs)
			authed.GET("/games/park/top", parkH.Top)

			// 精武堂（复刻 wap 精武堂：属性/修炼/技能/比武/锻造/任务/头衔/帮派）
			authed.GET("/games/jwt/view", jwtH.View)
			authed.GET("/games/jwt/shop", jwtH.Shop)
			authed.POST("/games/jwt/buy", jwtH.Buy)
			authed.POST("/games/jwt/energy", jwtH.EnergyAlloc)
			authed.GET("/games/jwt/skills", jwtH.SkillShop)
			authed.POST("/games/jwt/learn", jwtH.LearnSkill)
			authed.POST("/games/jwt/skill-act", jwtH.SkillAct)
			authed.POST("/games/jwt/practice", jwtH.Practice)
			authed.GET("/games/jwt/arena", jwtH.ArenaList)
			authed.POST("/games/jwt/arena", jwtH.Arena)
			authed.GET("/games/jwt/arena-live", jwtH.ArenaLive)
			authed.GET("/games/jwt/records", jwtH.ArenaRecords)
			authed.GET("/games/jwt/records/:id", jwtH.RecordDetail)
			authed.GET("/games/jwt/bag", jwtH.Bag)
			authed.POST("/games/jwt/item-use", jwtH.UseItem)
			authed.POST("/games/jwt/item-drop", jwtH.ItemDrop)
			authed.POST("/games/jwt/equip", jwtH.Equip)
			authed.POST("/games/jwt/unequip", jwtH.Unequip)
			authed.GET("/games/jwt/forge", jwtH.Forge)
			authed.POST("/games/jwt/forge", jwtH.ForgeItem)
			authed.GET("/games/jwt/tasks", jwtH.Tasks)
			authed.POST("/games/jwt/sign", jwtH.Sign)
			authed.POST("/games/jwt/reward", jwtH.Reward)
			authed.POST("/games/jwt/chat", jwtH.Chat)
			authed.GET("/games/jwt/chat", jwtH.ChatList)
			authed.GET("/games/jwt/logs", jwtH.Logs)
			authed.GET("/games/jwt/titles", jwtH.Titles)
			authed.POST("/games/jwt/titles", jwtH.Titles)
			authed.GET("/games/jwt/gangs", jwtH.Gangs)
			authed.POST("/games/jwt/gang/create", jwtH.GangCreate)
			authed.POST("/games/jwt/gang/join", jwtH.GangJoin)
			authed.POST("/games/jwt/gang/leave", jwtH.GangLeave)
			authed.GET("/games/jwt/ranking", jwtH.Ranking)
			authed.GET("/games/jwt/title-ranking", jwtH.TitleRanking)
			authed.POST("/games/jwt/myskill", jwtH.MySkill)
			authed.GET("/games/jwt/friends", jwtH.Friends)
			authed.POST("/games/jwt/profile", jwtH.PlayerProfile)
			authed.POST("/games/jwt/profile-edit", jwtH.ProfileEdit)
			authed.POST("/games/jwt/levelup", jwtH.LevelUp)
			authed.POST("/games/jwt/gangview", jwtH.GangView)
			authed.POST("/games/jwt/gangapply", jwtH.GangApply)

			// 幻想西游（复刻 wap 幻想西游：地图行走/回合战斗/装备/技能/宠物/副本/社交）
			// 整组挂服务器维护中间件：维护中所有游戏接口返回维护公告
			hxxyG := authed.Group("/games/hxxy", hxH.HxxyMaintGate())
			hxxyG.GET("/status", hxH.Status)
			hxxyG.POST("/create", hxH.Create)
			hxxyG.GET("/state", hxH.State)
			hxxyG.GET("/home", hxH.Home)
			hxxyG.GET("/pm/:id", hxH.PMList)
			hxxyG.POST("/pm/send", hxH.PMSend)
			hxxyG.GET("/attrs", hxH.Attrs)
			hxxyG.POST("/move", hxH.Move)
			hxxyG.POST("/jump", hxH.Jump)
			hxxyG.GET("/map/grid", hxH.MapGrid)
			hxxyG.GET("/npc/:id", hxH.NpcView)
			hxxyG.POST("/npc/teleport", hxH.NpcTeleport)
			hxxyG.POST("/rest", hxH.Rest)
			hxxyG.POST("/battle/start", hxH.BattleStart)
			hxxyG.GET("/battle/state", hxH.BattleState)
			hxxyG.POST("/battle/action", hxH.BattleAction)
			hxxyG.GET("/bag", hxH.Bag)
			hxxyG.POST("/bag/use", hxH.BagUse)
			hxxyG.POST("/bag/discard", hxH.BagDiscard)
			hxxyG.GET("/warehouse", hxH.Warehouse)
			hxxyG.POST("/warehouse/deposit", hxH.WhDeposit)
			hxxyG.POST("/warehouse/takeout", hxH.WhTakeout)
			hxxyG.POST("/equip/wear", hxH.EquipWear)
			hxxyG.POST("/equip/takeoff", hxH.EquipTakeoff)
			hxxyG.POST("/equip/upgrade", hxH.EquipUpgrade)
			hxxyG.POST("/equip/hole", hxH.EquipHole)
			hxxyG.POST("/equip/gem", hxH.EquipGem)
			hxxyG.GET("/skills", hxH.Skills)
			hxxyG.POST("/skills/learn", hxH.SkillLearn)
			hxxyG.GET("/pets", hxH.Pets)
			hxxyG.POST("/pets/act", hxH.PetAct)
			hxxyG.GET("/shop/:kind", hxH.Shop)
			hxxyG.POST("/shop/buy", hxH.ShopBuy)
			hxxyG.POST("/shop/sell", hxH.ShopSell)
			hxxyG.GET("/bank", hxH.Bank)
			hxxyG.POST("/bank/deposit", hxH.BankDeposit)
			hxxyG.POST("/bank/withdraw", hxH.BankWithdraw)
			hxxyG.GET("/quests", hxH.Quests)
			hxxyG.POST("/quests/accept", hxH.QuestAccept)
			hxxyG.POST("/quests/submit", hxH.QuestSubmit)
			hxxyG.POST("/quests/abandon", hxH.QuestAbandon)
			hxxyG.GET("/activities", hxH.Activities)
			hxxyG.POST("/activities/claim", hxH.ActivityClaim)
			hxxyG.GET("/dungeons", hxH.Dungeons)
			hxxyG.POST("/dungeons/enter", hxH.DungeonEnter)
			hxxyG.GET("/bosses", hxH.Bosses)
			hxxyG.POST("/bosses/challenge", hxH.BossChallenge)
			hxxyG.GET("/cultivate", hxH.Cultivate)
			hxxyG.POST("/cultivate/toggle", hxH.CultivateToggle)
			hxxyG.POST("/cultivate/upgrade", hxH.CultivateUpgrade)
			hxxyG.POST("/cultivate/exchange-dan", hxH.CultivateExchangeDan)
			hxxyG.GET("/titles", hxH.Titles)
			hxxyG.POST("/titles/activate", hxH.TitleActivate)
			hxxyG.POST("/titles/wear", hxH.TitleWear)
			hxxyG.POST("/signin", hxH.Signin)
			hxxyG.GET("/signin/info", hxH.SigninInfo)
			hxxyG.POST("/signin/claim", hxH.SigninClaim)
			hxxyG.GET("/welfare", hxH.Welfare)
			hxxyG.POST("/welfare/gift", hxH.WelfareGiftClaim)
			hxxyG.POST("/welfare/noble", hxH.WelfareNobleClaim)
			hxxyG.GET("/vip/info", hxH.VipInfo)
			hxxyG.POST("/vip/exchange", hxH.VipExchange)
			hxxyG.POST("/vip/recharge", hxH.VipRecharge)
			hxxyG.GET("/rank/:type", hxH.Rank)
			hxxyG.GET("/chat", hxH.ChatList)
			hxxyG.POST("/chat", hxH.ChatPost)
			hxxyG.GET("/friends", hxH.Friends)
			hxxyG.POST("/friends/add", hxH.FriendAdd)
			hxxyG.POST("/friends/black", hxH.FriendBlack)
			hxxyG.POST("/friends/remove", hxH.FriendRemove)
			hxxyG.GET("/gang", hxH.GangInfo)
			hxxyG.POST("/gang/create", hxH.GangCreate)
			hxxyG.POST("/gang/join", hxH.GangJoin)
			hxxyG.POST("/gang/leave", hxH.GangLeave)
			hxxyG.POST("/gang/donate", hxH.GangDonate)
			hxxyG.POST("/gang/invite", hxH.GangInvite)
			hxxyG.POST("/gang/invite/agree", hxH.GangInviteAgree)
			hxxyG.POST("/gang/invite/refuse", hxH.GangInviteRefuse)
			hxxyG.GET("/gang/mall", hxH.GangMall)
			hxxyG.POST("/gang/mall/buy", hxH.GangMallBuy)
			hxxyG.POST("/gang/appoint", hxH.GangAppoint)
			hxxyG.POST("/gang/dismiss", hxH.GangDismiss)
			hxxyG.POST("/gang/kick", hxH.GangKick)
			hxxyG.POST("/gang/upgrade", hxH.GangUpgrade)
			hxxyG.POST("/gang/dissolve", hxH.GangDissolve)
			hxxyG.GET("/marriage", hxH.MarriageInfo)
			hxxyG.POST("/marriage/propose", hxH.MarriagePropose)
			hxxyG.POST("/marriage/agree", hxH.MarriageAgree)
			hxxyG.POST("/marriage/refuse", hxH.MarriageRefuse)
			hxxyG.POST("/marriage/divorce", hxH.MarriageDivorce)
			hxxyG.GET("/house", hxH.House)
			hxxyG.POST("/house/buy", hxH.HouseBuy)
			hxxyG.POST("/house/invite", hxH.HouseInvite)
			hxxyG.POST("/house/invite/agree", hxH.HouseInviteAgree)
			hxxyG.POST("/house/invite/refuse", hxH.HouseInviteRefuse)
			hxxyG.POST("/house/visit", hxH.HouseVisit)
			hxxyG.GET("/stalls", hxH.Stalls)
			hxxyG.GET("/stalls/mine", hxH.StallsMine)
			hxxyG.GET("/stalls/player/:id", hxH.StallsOf)
			hxxyG.POST("/stall/sell", hxH.StallSell)
			hxxyG.POST("/stall/buy", hxH.StallBuy)
			hxxyG.POST("/stall/cancel", hxH.StallCancel)
			hxxyG.GET("/auction", hxH.AuctionList)
			hxxyG.GET("/auction/mine", hxH.AuctionMine)
			hxxyG.POST("/auction/sell", hxH.AuctionSell)
			hxxyG.POST("/auction/buy", hxH.AuctionBuy)
			hxxyG.POST("/auction/cancel", hxH.AuctionCancel)
			hxxyG.POST("/give/money", hxH.GiveMoney)
			hxxyG.POST("/give/item", hxH.GiveItem)
			hxxyG.GET("/wallet", hxH.WalletLogs)
			hxxyG.GET("/battle-logs", hxH.BattleLogs)
			hxxyG.GET("/arena/info", hxH.ArenaInfo)
			hxxyG.POST("/arena/fight", hxH.ArenaFight)
			hxxyG.GET("/tower/info", hxH.TowerInfo)
			hxxyG.POST("/tower/start", hxH.TowerStart)
			hxxyG.POST("/fun/roll", hxH.FunRoll)
			hxxyG.GET("/teyun/list", hxH.TeyunList)
			hxxyG.POST("/teyun/go", hxH.TeyunGo)
			hxxyG.GET("/player/:id", hxH.PlayerView)
			hxxyG.GET("/gz/info", hxH.GzInfo)
			hxxyG.POST("/gz/signup", hxH.GzSignup)
			hxxyG.POST("/gz/rod", hxH.GzRod)
			hxxyG.POST("/gz/neijian", hxH.GzNeijian)
			hxxyG.GET("/team/info", hxH.TeamInfo)
			hxxyG.POST("/team/create", hxH.TeamCreate)
			hxxyG.POST("/team/invite", hxH.TeamInvite)
			hxxyG.POST("/team/agree", hxH.TeamAgree)
			hxxyG.POST("/team/refuse", hxH.TeamRefuse)
			hxxyG.POST("/team/kick", hxH.TeamKick)
			hxxyG.POST("/team/leave", hxH.TeamLeave)

			// 二战风云（复刻 stzb-fk 二战风云：城池建造/资源结算/造兵科技/地图出征/多回合战斗/军团）
			// 整组挂服务器维护中间件：维护中所有游戏接口返回维护公告
			ezfyG := authed.Group("/games/ezfy", ezfyH.EzfyMaintGate())
			{
				ezfyG.GET("/view", ezfyH.View)
				ezfyG.GET("/city/list", ezfyH.CityList)
				ezfyG.POST("/city/switch", ezfyH.SwitchCity)
				ezfyG.POST("/city/create", ezfyH.CreateCity)
				ezfyG.POST("/city/rename", ezfyH.RenameCity)
				ezfyG.POST("/city/tax", ezfyH.SetTax)
				ezfyG.POST("/city/convene", ezfyH.Convene)
				ezfyG.POST("/city/placate", ezfyH.Placate)
				ezfyG.POST("/city/abandon-wild", ezfyH.AbandonWildland)
				ezfyG.GET("/city/move", ezfyH.MoveInfo)
				ezfyG.POST("/city/move", ezfyH.MoveCity)
				ezfyG.GET("/city/produce", ezfyH.ProduceInfo)
				ezfyG.POST("/city/produce", ezfyH.ProduceSet)
				ezfyG.GET("/resources", ezfyH.Resources)
				ezfyG.GET("/buildings", ezfyH.Buildings)
				ezfyG.POST("/build", ezfyH.Build)
				ezfyG.POST("/building/upgrade", ezfyH.Upgrade)
				ezfyG.POST("/building/max-level", ezfyH.MaxLevel)
				ezfyG.POST("/building/delete", ezfyH.DeleteBuilding)
				ezfyG.POST("/building/speed", ezfyH.SpeedBuilding)
				ezfyG.GET("/troops", ezfyH.Troops)
				ezfyG.POST("/troops/train", ezfyH.Train)
				ezfyG.POST("/troops/speed-all", ezfyH.SpeedTrainAll)
				ezfyG.POST("/troops/recover", ezfyH.RecoverWounded)
				ezfyG.POST("/troops/dismiss", ezfyH.DismissDefence)
				ezfyG.GET("/techs", ezfyH.Techs)
				ezfyG.POST("/techs/research", ezfyH.Research)
				ezfyG.POST("/techs/speed", ezfyH.SpeedTech)
				ezfyG.POST("/techs/cancel", ezfyH.CancelTech)
				ezfyG.GET("/map", ezfyH.MapView)
				ezfyG.GET("/map/wildland", ezfyH.WildlandView)
				ezfyG.GET("/map/stars", ezfyH.MapStars)
				ezfyG.POST("/map/stars", ezfyH.MapStarAdd)
				ezfyG.POST("/map/stars/delete", ezfyH.MapStarDelete)
				ezfyG.POST("/order", ezfyH.CreateOrder)
				ezfyG.POST("/order/preview", ezfyH.OrderPreview)
				ezfyG.GET("/orders", ezfyH.OrderList)
				ezfyG.POST("/order/recall", ezfyH.RecallOrder)
				ezfyG.POST("/wild/collect-all", ezfyH.CollectAll)
				ezfyG.POST("/wild/harvest-all", ezfyH.HarvestAll)
				ezfyG.GET("/reports", ezfyH.Reports)
				ezfyG.GET("/reports/dynamics", ezfyH.ReportDynamics)
				ezfyG.GET("/reports/:id", ezfyH.ReportView)
				ezfyG.GET("/targets", ezfyH.Targets)
				ezfyG.POST("/targets", ezfyH.SaveTarget)
				ezfyG.GET("/corps/list", ezfyH.CorpsList)
				ezfyG.POST("/corps/create", ezfyH.CorpsCreate)
				ezfyG.POST("/corps/join", ezfyH.CorpsJoin)
				ezfyG.POST("/corps/leave", ezfyH.CorpsLeave)
				ezfyG.POST("/corps/kick", ezfyH.CorpsKick)
				ezfyG.POST("/corps/notice", ezfyH.CorpsNotice)
				ezfyG.GET("/corps/chats", ezfyH.CorpsChats)
				ezfyG.POST("/corps/chat", ezfyH.CorpsChat)
				ezfyG.POST("/corps/mail", ezfyH.CorpsMail)
				ezfyG.GET("/corps/members", ezfyH.CorpsMembers)
				ezfyG.GET("/liaison", ezfyH.Liaison)
				ezfyG.GET("/player/:id", ezfyH.PlayerInfo)
				ezfyG.POST("/war/declare", ezfyH.DeclareWar)
				ezfyG.GET("/war/status", ezfyH.WarStatus)
				ezfyG.GET("/rank", ezfyH.Rank)
				ezfyG.GET("/mall", ezfyH.Mall)
				ezfyG.POST("/mall/buy", ezfyH.Buy)
				ezfyG.GET("/bag", ezfyH.Bag)
				ezfyG.POST("/bag/use", ezfyH.UseItem)
				ezfyG.GET("/tasks", ezfyH.Tasks)
				ezfyG.POST("/tasks/award", ezfyH.TaskAward)
				ezfyG.GET("/welfare", ezfyH.Welfare)
				ezfyG.POST("/welfare/sign", ezfyH.Sign)
				ezfyG.POST("/welfare/gift/:type", ezfyH.Gift)
				ezfyG.GET("/notices", ezfyH.Notices)
				ezfyG.GET("/activity", ezfyH.ActivityInfo)
				ezfyG.GET("/chat", ezfyH.ChatList)
				ezfyG.POST("/chat", ezfyH.ChatSend)
				ezfyG.GET("/chat/home", ezfyH.HomeChat)
				ezfyG.GET("/exchange", ezfyH.ExchangeList)
				ezfyG.POST("/exchange/sell", ezfyH.ExchangeSell)
				ezfyG.POST("/exchange/buy", ezfyH.ExchangeBuy)
				ezfyG.POST("/exchange/cancel", ezfyH.ExchangeCancel)
				ezfyG.GET("/city/wildfull", ezfyH.WildlandFull)
				ezfyG.GET("/city/warehouse", ezfyH.Warehouse)
				ezfyG.POST("/city/warehouse", ezfyH.WarehouseSet)
				ezfyG.POST("/city/occupy/:op", ezfyH.OccupyOp)
				ezfyG.GET("/orders/:id", ezfyH.OrderView)

				// ===== 军官/学院系统（复刻 stzb-fk：军校/参谋部/技能/装备/俘虏/任命）=====
				ezfyG.GET("/officers", ezfyH.Officers)
				ezfyG.GET("/officers/onduty", ezfyH.OfficersOnDuty)
				ezfyG.GET("/officers/skills", ezfyH.OfficerSkills)
				ezfyG.GET("/officers/equipments", ezfyH.OfficerEquipments)
				ezfyG.GET("/officers/generals", ezfyH.OfficerGenerals)
				ezfyG.GET("/officers/:id", ezfyH.OfficerDetail)
				ezfyG.POST("/officers/:id/grant", ezfyH.OfficerGrant)
				ezfyG.POST("/officers/:id/skill", ezfyH.OfficerSkill)
				ezfyG.POST("/officers/:id/equip", ezfyH.OfficerEquip)
				ezfyG.POST("/officers/:id/position", ezfyH.OfficerPosition)
				ezfyG.POST("/officers/:id/captive", ezfyH.OfficerCaptive)
				ezfyG.POST("/officers/:id/exile", ezfyH.OfficerExile)
				ezfyG.GET("/acade/recruit", ezfyH.AcadeRecruit)
				ezfyG.POST("/acade/recruit/refresh", ezfyH.AcadeRefresh)
				ezfyG.POST("/acade/recruit/hire", ezfyH.AcadeRecruitDo)
			}

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
			authed.GET("/friends/search", friendH.Search)
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
			authed.POST("/guestbook/:id/reply", perm(db, "module:guestbook"), guestH.Reply)
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

			// 福利院捐款上榜（每日价高者受全社区膜拜）
			authed.POST("/fla/donate", flaH.Donate)
			authed.POST("/fla/worship", flaH.Worship)

			// 福利院·慈善基金（每日领取/捐献）
			authed.POST("/welfare/claim", welfareH.Claim)
			authed.POST("/welfare/donate", welfareH.Donate)

			// 管理后台（RBAC 权限点）
			admin := authed.Group("/admin")
			{
				admin.GET("/stats", perm(db, "module:dashboard"), adminH.Stats)

				// 举报管理
				admin.GET("/reports", perm(db, "module:threads"), adminH.Reports)
				admin.PUT("/reports/:id", perm(db, "module:threads"), adminH.ReportHandle)

				// 广场板块开关
				admin.GET("/plaza-sections", perm(db, "module:articles"), plazaH.AdminSections)
				admin.PUT("/plaza-sections/:id", perm(db, "module:articles"), plazaH.AdminSectionUpdate)

				// 特权管理（蓝钻/超Q，复刻诺哈 vip：等级配置/贵宾会员/贵宾销售）
				admin.GET("/privileges/stats", perm(db, "module:privileges"), nobleH.AdminStats)
				admin.GET("/privileges/levels", perm(db, "module:privileges"), nobleH.AdminLevels)
				admin.POST("/privileges/levels", perm(db, "module:privileges"), nobleH.AdminLevelCreate)
				admin.PUT("/privileges/levels/:id", perm(db, "module:privileges"), nobleH.AdminLevelUpdate)
				admin.DELETE("/privileges/levels/:id", perm(db, "module:privileges"), nobleH.AdminLevelDelete)
				admin.GET("/privileges/plans", perm(db, "module:privileges"), nobleH.AdminPlans)
				admin.POST("/privileges/plans", perm(db, "module:privileges"), nobleH.AdminPlanCreate)
				admin.PUT("/privileges/plans/:id", perm(db, "module:privileges"), nobleH.AdminPlanUpdate)
				admin.DELETE("/privileges/plans/:id", perm(db, "module:privileges"), nobleH.AdminPlanDelete)
				admin.GET("/privileges/users", perm(db, "module:privileges"), nobleH.AdminUsers)
				admin.PUT("/privileges/users/:id", perm(db, "module:privileges"), nobleH.AdminUserUpdate)
				admin.POST("/privileges/users/:id/open", perm(db, "module:privileges"), nobleH.AdminUserOpen)
				admin.POST("/privileges/users/:id/close", perm(db, "module:privileges"), nobleH.AdminUserClose)
				admin.POST("/privileges/expired-clean", perm(db, "module:privileges"), nobleH.AdminExpiredClean)
				admin.POST("/privileges/batch", perm(db, "module:privileges"), nobleH.AdminBatch)

				// 道具商城管理
				admin.GET("/goods", perm(db, "module:goods"), goodH.AdminList)
				admin.POST("/goods", perm(db, "module:goods"), goodH.AdminCreate)
				admin.PUT("/goods/:id", perm(db, "module:goods"), goodH.AdminUpdate)
				admin.DELETE("/goods/:id", perm(db, "module:goods"), goodH.AdminDelete)

				// 货币商店管理（复刻诺哈 wap_money_shop）
				admin.GET("/money-shops", perm(db, "module:moneyShop"), msH.AdminList)
				admin.POST("/money-shops", perm(db, "module:moneyShop"), msH.AdminCreate)
				admin.PUT("/money-shops/:id", perm(db, "module:moneyShop"), msH.AdminUpdate)
				admin.DELETE("/money-shops/:id", perm(db, "module:moneyShop"), msH.AdminDelete)

				// 花园活动管理
				admin.GET("/garden-activities", perm(db, "module:gardenActivities"), gardenH.AdminActivities)
				admin.POST("/garden-activities", perm(db, "module:gardenActivities"), gardenH.AdminActCreate)
				admin.PUT("/garden-activities/:id", perm(db, "module:gardenActivities"), gardenH.AdminActUpdate)
				admin.DELETE("/garden-activities/:id", perm(db, "module:gardenActivities"), gardenH.AdminActDelete)

				// 花园数据维护（种子/图鉴/配方）
				admin.GET("/garden-seeds", perm(db, "module:gardenSeeds"), gardenH.AdminSeeds)
				admin.POST("/garden-seeds", perm(db, "module:gardenSeeds"), gardenH.AdminSeedCreate)
				admin.PUT("/garden-seeds/:id", perm(db, "module:gardenSeeds"), gardenH.AdminSeedUpdate)
				admin.DELETE("/garden-seeds/:id", perm(db, "module:gardenSeeds"), gardenH.AdminSeedDelete)
				admin.GET("/garden-maps", perm(db, "module:gardenMaps"), gardenH.AdminMaps)
				admin.POST("/garden-maps", perm(db, "module:gardenMaps"), gardenH.AdminMapCreate)
				admin.PUT("/garden-maps/:id", perm(db, "module:gardenMaps"), gardenH.AdminMapUpdate)
				admin.DELETE("/garden-maps/:id", perm(db, "module:gardenMaps"), gardenH.AdminMapDelete)
				admin.GET("/garden-mixes", perm(db, "module:gardenMixes"), gardenH.AdminMixes)
				admin.POST("/garden-mixes", perm(db, "module:gardenMixes"), gardenH.AdminMixCreate)
				admin.PUT("/garden-mixes/:id", perm(db, "module:gardenMixes"), gardenH.AdminMixUpdate)
				admin.DELETE("/garden-mixes/:id", perm(db, "module:gardenMixes"), gardenH.AdminMixDelete)
				admin.GET("/garden-elves", perm(db, "module:gardenElves"), gardenH.AdminElves)
				admin.POST("/garden-elves", perm(db, "module:gardenElves"), gardenH.AdminElfCreate)
				admin.PUT("/garden-elves/:id", perm(db, "module:gardenElves"), gardenH.AdminElfUpdate)
				admin.DELETE("/garden-elves/:id", perm(db, "module:gardenElves"), gardenH.AdminElfDelete)

				// 花园游戏数据管理（用户数据/日志流水/排行榜）
				admin.GET("/garden-users", perm(db, "module:gardenData"), gardenH.AdminGardenUsers)
				admin.GET("/garden-users/:uid", perm(db, "module:gardenData"), gardenH.AdminGardenUserDetail)
				admin.PUT("/garden-users/:uid/garden", perm(db, "module:gardenData"), gardenH.AdminGardenEdit)
				admin.PUT("/garden-users/:uid/coins", perm(db, "module:gardenData"), gardenH.AdminGardenCoins)
				admin.PUT("/garden-users/:uid/bag", perm(db, "module:gardenData"), gardenH.AdminGardenBagSet)
				admin.PUT("/garden-users/:uid/flowers/:target", perm(db, "module:gardenData"), gardenH.AdminGardenFlowerSet)
				admin.GET("/garden-logs", perm(db, "module:gardenData"), gardenH.AdminGardenLogs)
				admin.GET("/garden-rank", perm(db, "module:gardenData"), gardenH.AdminGardenRank)
				admin.GET("/garden-sign-rewards", perm(db, "module:gardenSign"), gardenH.AdminSignRewards)
				admin.PUT("/garden-sign-rewards/:day", perm(db, "module:gardenSign"), gardenH.AdminSignRewardUpdate)
				admin.GET("/garden-sign-stats", perm(db, "module:gardenSign"), gardenH.AdminSignStats)
				// 开心农场管理（种子/化肥/陷阱/用户数据/日志/排行）
				admin.GET("/farm-seeds", perm(db, "module:farmSeeds"), farmH.AdminSeeds)
				admin.POST("/farm-seeds", perm(db, "module:farmSeeds"), farmH.AdminSeedCreate)
				admin.PUT("/farm-seeds/:id", perm(db, "module:farmSeeds"), farmH.AdminSeedUpdate)
				admin.DELETE("/farm-seeds/:id", perm(db, "module:farmSeeds"), farmH.AdminSeedDelete)
				admin.GET("/farm-mucks", perm(db, "module:farmItems"), farmH.AdminMucks)
				admin.POST("/farm-mucks", perm(db, "module:farmItems"), farmH.AdminMuckCreate)
				admin.PUT("/farm-mucks/:id", perm(db, "module:farmItems"), farmH.AdminMuckUpdate)
				admin.DELETE("/farm-mucks/:id", perm(db, "module:farmItems"), farmH.AdminMuckDelete)
				admin.GET("/farm-traps", perm(db, "module:farmItems"), farmH.AdminTraps)
				admin.POST("/farm-traps", perm(db, "module:farmItems"), farmH.AdminTrapCreate)
				admin.PUT("/farm-traps/:id", perm(db, "module:farmItems"), farmH.AdminTrapUpdate)
				admin.DELETE("/farm-traps/:id", perm(db, "module:farmItems"), farmH.AdminTrapDelete)
				admin.GET("/farm-users", perm(db, "module:farmData"), farmH.AdminUsers)
				admin.GET("/farm-users/:uid", perm(db, "module:farmData"), farmH.AdminUserDetail)
				admin.PUT("/farm-users/:uid/farm", perm(db, "module:farmData"), farmH.AdminFarmEdit)
				admin.PUT("/farm-users/:uid/coins", perm(db, "module:farmData"), farmH.AdminCoins)
				admin.PUT("/farm-users/:uid/bag/:target", perm(db, "module:farmData"), farmH.AdminBagSet)
				admin.PUT("/farm-users/:uid/land/:id/clear", perm(db, "module:farmData"), farmH.AdminLandClear)
				admin.GET("/farm-logs", perm(db, "module:farmData"), farmH.AdminLogs)
				admin.GET("/farm-rank", perm(db, "module:farmData"), farmH.AdminRank)
				// 抢车位管理（车市/用户数据/日志）
				admin.GET("/park-cars", perm(db, "module:parkCars"), parkH.AdminCars)
				admin.POST("/park-cars", perm(db, "module:parkCars"), parkH.AdminCarCreate)
				admin.PUT("/park-cars/:id", perm(db, "module:parkCars"), parkH.AdminCarUpdate)
				admin.DELETE("/park-cars/:id", perm(db, "module:parkCars"), parkH.AdminCarDelete)
				admin.GET("/park-users", perm(db, "module:parkData"), parkH.AdminUsers)
				admin.PUT("/park-users/:uid", perm(db, "module:parkData"), parkH.AdminUserEdit)
				admin.PUT("/park-users/:uid/spots/:id/clear", perm(db, "module:parkData"), parkH.AdminSpotClear)
				admin.GET("/park-logs", perm(db, "module:parkData"), parkH.AdminLogs)

				// 精武堂管理（玩家/道具/技能/帮派/聊天）
				admin.GET("/jwt-players", perm(db, "module:jwtPlayers"), adminH.AdminJwtPlayers)
				admin.GET("/jwt-players/:uid/detail", perm(db, "module:jwtPlayers"), adminH.AdminJwtPlayerDetail)
				admin.PUT("/jwt-players/:uid", perm(db, "module:jwtPlayers"), adminH.AdminJwtPlayerUpdate)
				admin.GET("/jwt-items", perm(db, "module:jwtItems"), adminH.AdminJwtItems)
				admin.POST("/jwt-items", perm(db, "module:jwtItems"), adminH.AdminJwtItemCreate)
				admin.PUT("/jwt-items/:id", perm(db, "module:jwtItems"), adminH.AdminJwtItemUpdate)
				admin.DELETE("/jwt-items/:id", perm(db, "module:jwtItems"), adminH.AdminJwtItemDelete)
				admin.GET("/jwt-skills", perm(db, "module:jwtSkills"), adminH.AdminJwtSkills)
				admin.POST("/jwt-skills", perm(db, "module:jwtSkills"), adminH.AdminJwtSkillCreate)
				admin.PUT("/jwt-skills/:id", perm(db, "module:jwtSkills"), adminH.AdminJwtSkillUpdate)
				admin.DELETE("/jwt-skills/:id", perm(db, "module:jwtSkills"), adminH.AdminJwtSkillDelete)
				admin.GET("/jwt-gangs", perm(db, "module:jwtPlayers"), adminH.AdminJwtGangs)
				admin.DELETE("/jwt-gangs/:id", perm(db, "module:jwtPlayers"), adminH.AdminJwtGangDelete)
				admin.GET("/jwt-chats", perm(db, "module:jwtPlayers"), adminH.AdminJwtChats)

				// 个性昵称管理（购买状态/颜色调整/重置）
				admin.GET("/name-users", perm(db, "module:nickName"), nameH.AdminNameUsers)
				admin.PUT("/name-users/:uid", perm(db, "module:nickName"), nameH.AdminNameUserEdit)
				admin.DELETE("/jwt-chats/:id", perm(db, "module:jwtPlayers"), adminH.AdminJwtChatDelete)
				admin.GET("/jwt-records", perm(db, "module:jwtRecords"), adminH.AdminJwtRecords)
				admin.GET("/jwt-records/:id/detail", perm(db, "module:jwtRecords"), adminH.AdminJwtRecordDetail)
				admin.DELETE("/jwt-records/:id", perm(db, "module:jwtRecords"), adminH.AdminJwtRecordDelete)

				// 幻想西游管理（玩家/游戏数据/战斗流水/货币流水/系统管理）
				admin.GET("/xy-players", perm(db, "module:xyPlayers"), adminH.AdminXyPlayers)
				admin.GET("/xy-players/:id/detail", perm(db, "module:xyPlayers"), adminH.AdminXyPlayerDetail)
				admin.PUT("/xy-players/:id", perm(db, "module:xyPlayers"), adminH.AdminXyPlayerUpdate)
				admin.POST("/xy-players/:id/grant", perm(db, "module:xyPlayers"), adminH.AdminXyGrant)
				admin.POST("/xy-players/:id/ban", perm(db, "module:xyPlayers"), adminH.AdminXyBan)
				admin.POST("/xy-players/:id/mute", perm(db, "module:xyPlayers"), adminH.AdminXyMute)
				admin.DELETE("/xy-players/:id", perm(db, "module:xyPlayers"), adminH.AdminXyPlayerDelete)
				admin.GET("/xy-data/:table", perm(db, "module:xyData"), adminH.AdminXyData)
				admin.POST("/xy-data/:table", perm(db, "module:xyData"), adminH.AdminXyDataCreate)
				admin.PUT("/xy-data/:table/:id", perm(db, "module:xyData"), adminH.AdminXyDataUpdate)
				admin.DELETE("/xy-data/:table/:id", perm(db, "module:xyData"), adminH.AdminXyDataDelete)
				admin.GET("/xy-server", perm(db, "module:xySystem"), adminH.AdminXyServer)
				admin.POST("/xy-server/maintenance", perm(db, "module:xySystem"), adminH.AdminXyServerSet)
				admin.POST("/xy-server/exp2x", perm(db, "module:xySystem"), adminH.AdminXyExp2xSet)
				admin.GET("/xy-battles", perm(db, "module:xyLogs"), adminH.AdminXyBattles)
				admin.DELETE("/xy-battles/:id", perm(db, "module:xySystem"), adminH.AdminXyBattleDelete)
				admin.GET("/xy-wallet", perm(db, "module:xySystem"), adminH.AdminXyWallet)
				admin.GET("/xy-stats", perm(db, "module:xySystem"), adminH.AdminXyStats)
				admin.POST("/xy-announce", perm(db, "module:xySystem"), adminH.AdminXyAnnounce)
				admin.POST("/xy-grant-all", perm(db, "module:xySystem"), adminH.AdminXyGrantAll)

				// ============ 二战风云管理 ezfy/（玩家/数据/流水/系统） ============
				admin.GET("/ezfy-players", perm(db, "module:ezfyPlayers"), adminH.AdminEzfyPlayers)
				admin.GET("/ezfy-players/:id/detail", perm(db, "module:ezfyPlayers"), adminH.AdminEzfyPlayerDetail)
				admin.PUT("/ezfy-players/:id", perm(db, "module:ezfyPlayers"), adminH.AdminEzfyPlayerUpdate)
				admin.POST("/ezfy-players/:id/grant", perm(db, "module:ezfyPlayers"), adminH.AdminEzfyGrant)
				admin.POST("/ezfy-players/:id/grant-officer", perm(db, "module:ezfyPlayers"), adminH.AdminEzfyGrantOfficer)
				admin.DELETE("/ezfy-players/:id", perm(db, "module:ezfyPlayers"), adminH.AdminEzfyPlayerDelete)
				admin.GET("/ezfy-data/:table", perm(db, "module:ezfyData"), adminH.AdminEzfyData)
				admin.POST("/ezfy-data/:table", perm(db, "module:ezfyData"), adminH.AdminEzfyDataCreate)
				admin.PUT("/ezfy-data/:table/:id", perm(db, "module:ezfyData"), adminH.AdminEzfyDataUpdate)
				admin.DELETE("/ezfy-data/:table/:id", perm(db, "module:ezfyData"), adminH.AdminEzfyDataDelete)
				admin.GET("/ezfy-orders", perm(db, "module:ezfyLogs"), adminH.AdminEzfyOrders)
				admin.DELETE("/ezfy-orders/:id", perm(db, "module:ezfyLogs"), adminH.AdminEzfyOrderDelete)
				admin.GET("/ezfy-chats", perm(db, "module:ezfyLogs"), adminH.AdminEzfyChats)
				admin.DELETE("/ezfy-chats/:id", perm(db, "module:ezfyLogs"), adminH.AdminEzfyChatDelete)
				admin.GET("/ezfy-exchanges", perm(db, "module:ezfyLogs"), adminH.AdminEzfyExchanges)
				admin.DELETE("/ezfy-exchanges/:id", perm(db, "module:ezfyLogs"), adminH.AdminEzfyExchangeDelete)
				admin.GET("/ezfy-corps", perm(db, "module:ezfySystem"), adminH.AdminEzfyCorps)
				admin.DELETE("/ezfy-corps/:id", perm(db, "module:ezfySystem"), adminH.AdminEzfyCorpsDelete)
				admin.GET("/ezfy-notices", perm(db, "module:ezfySystem"), adminH.AdminEzfyNotices)
				admin.POST("/ezfy-announce", perm(db, "module:ezfySystem"), adminH.AdminEzfyAnnounce)
				admin.DELETE("/ezfy-notices/:id", perm(db, "module:ezfySystem"), adminH.AdminEzfyNoticeDelete)
				admin.GET("/ezfy-stats", perm(db, "module:ezfySystem"), adminH.AdminEzfyStats)
				admin.GET("/ezfy-server", perm(db, "module:ezfySystem"), adminH.AdminEzfyServer)
				admin.POST("/ezfy-server/maintenance", perm(db, "module:ezfySystem"), adminH.AdminEzfyServerSet)

				// ============ 会员管理 user/（诺哈：会员列表/证件/联系/地址/密保/日志/财务/推荐） ============
				admin.GET("/users", perm(db, "module:users"), adminH.Users)
				admin.GET("/users/:id/detail", perm(db, "module:users"), adminH.UserDetail)
				admin.GET("/pretty-suggest", perm(db, "module:users"), adminH.PrettySuggest)
				admin.POST("/users/:id/pretty", perm(db, "module:users"), adminH.UserPretty)
				admin.PUT("/users/:id/home", perm(db, "module:home"), adminH.UserHomeSet)
				admin.GET("/invites", perm(db, "module:invites"), adminH.AdminInvites)
				admin.POST("/invites", perm(db, "module:invites"), adminH.AdminInviteCreate)
				admin.DELETE("/invites/:id", perm(db, "module:invites"), adminH.AdminInviteDelete)
				admin.GET("/wallet-logs", perm(db, "module:walletLogs"), adminH.AdminWalletLogs)
				// 会员子页（诺哈：会员证件/联系/地址/密保/日志）
				admin.GET("/user-docu", perm(db, "module:userDocu"), adminH.AdminUserDocu)
				admin.DELETE("/user-docu/:id", perm(db, "module:userDocu"), adminH.AdminUserDocuDelete)
				admin.GET("/user-contacts", perm(db, "module:userContact"), adminH.AdminUserContacts)
				admin.PUT("/user-contacts/:id", perm(db, "module:userContact"), adminH.AdminUserContactUpdate)
				admin.GET("/user-addresses", perm(db, "module:userAddress"), adminH.AdminUserAddresses)
				admin.PUT("/user-addresses/:id", perm(db, "module:userAddress"), adminH.AdminUserAddressUpdate)
				admin.GET("/user-protections", perm(db, "module:userProtec"), adminH.AdminUserProtections)
				admin.DELETE("/user-protections/:uid", perm(db, "module:userProtec"), adminH.AdminUserProtectionDelete)
				admin.GET("/user-logs", perm(db, "module:userLogs"), adminH.AdminUserLogs)

				// ============ 家园管理 home/（诺哈：家园列表/家园访客/游戏管理） ============
				admin.GET("/homes", perm(db, "module:homes"), adminH.AdminHomes)
				admin.GET("/visitors", perm(db, "module:visitors"), adminH.AdminVisitors)

				// ============ 信息管理 message/（诺哈：家信列表） ============
				admin.GET("/home-news", perm(db, "module:messages"), adminH.AdminHomeNews)
				admin.DELETE("/home-news/:id", perm(db, "module:messages"), adminH.AdminHomeNewsDel)
				admin.GET("/messages", perm(db, "module:messages"), adminH.AdminMessages)
				admin.DELETE("/messages/:id", perm(db, "module:messages"), adminH.AdminMessageDel)

				// ============ 书城管理 book/（诺哈：小说/章节/书评） ============
				admin.GET("/books", perm(db, "module:books"), adminH.AdminBooks)
				admin.POST("/books", perm(db, "module:books"), adminH.AdminBookCreate)
				admin.PUT("/books/:id", perm(db, "module:books"), adminH.AdminBookUpdate)
				admin.DELETE("/books/:id", perm(db, "module:books"), adminH.AdminBookDel)
				admin.GET("/books/:id/chapters", perm(db, "module:bookChapters"), adminH.AdminBookChapters)
				admin.POST("/books/:id/chapters", perm(db, "module:bookChapters"), adminH.AdminChapterCreate)
				admin.PUT("/book-chapters/:id", perm(db, "module:bookChapters"), adminH.AdminChapterUpdate)
				admin.DELETE("/book-chapters/:id", perm(db, "module:bookChapters"), adminH.AdminChapterDel)
				admin.GET("/book-comments", perm(db, "module:bookComments"), adminH.AdminBookComments)
				admin.PUT("/book-comments/:id", perm(db, "module:bookComments"), adminH.AdminBookCommentUpdate)
				admin.DELETE("/book-comments/:id", perm(db, "module:bookComments"), adminH.AdminBookCommentDel)

				// ============ 系统配置 config/（站点设置） ============
				admin.GET("/site-config", perm(db, "module:siteConfig"), adminH.AdminSiteConfig)
				admin.PUT("/site-config", perm(db, "module:siteConfig"), adminH.AdminSiteConfigSave)

				// ============ 菜单维护 admin_menus/（管理端菜单覆盖配置） ============
				admin.GET("/menus", adminH.AdminMenus)
				admin.PUT("/menus", perm(db, "module:menus"), adminH.AdminMenuSave)
				admin.DELETE("/menus/:key", perm(db, "module:menus"), adminH.AdminMenuDel)

				// ============ 社区管理 bbs/（诺哈：帖子管理/回复管理/恢复帖子/黑名单） ============
				admin.GET("/threads/recycle", perm(db, "module:recycle"), adminH.AdminThreadRecycle)
				admin.PUT("/threads/:id/restore", perm(db, "module:recycle"), adminH.AdminThreadRestore)

				// ============ 商城管理 shop/（诺哈：商品管理/订单/评论） ============
				admin.GET("/shops", perm(db, "module:shops"), adminH.AdminShops)
				admin.GET("/shop-goods", perm(db, "module:shopGoods"), adminH.AdminShopGoods)
				admin.PUT("/shop-goods/:id/status", perm(db, "module:shopGoods"), adminH.AdminShopGoodsStatus)
				admin.DELETE("/shop-goods/:id", perm(db, "module:shopGoods"), adminH.AdminShopGoodsDel)
				admin.GET("/shop-orders", perm(db, "module:shopOrders"), adminH.AdminShopOrders)
				admin.GET("/shop-comments", perm(db, "module:shopComments"), adminH.AdminShopComments)
				admin.DELETE("/shop-comments/:id", perm(db, "module:shopComments"), adminH.AdminShopCommentDel)

				// ============ 文章管理 article/（诺哈：文章配置/文章分类） ============
				admin.GET("/site-articles", perm(db, "module:articles"), adminH.AdminSiteArticles)
				admin.DELETE("/site-articles/:id", perm(db, "module:articles"), adminH.AdminSiteArticleDel)
				admin.GET("/article-categories", perm(db, "module:articles"), adminH.AdminArticleCategories)
				admin.POST("/article-categories", perm(db, "module:articles"), adminH.AdminArticleCategoryCreate)
				admin.PUT("/article-categories/:id", perm(db, "module:articles"), adminH.AdminArticleCategoryUpdate)
				admin.DELETE("/article-categories/:id", perm(db, "module:articles"), adminH.AdminArticleCategoryDel)

				// ============ 留言管理 guest/ ============
				admin.GET("/guestbook", perm(db, "module:guestbook"), adminH.AdminGuestbook)
				admin.DELETE("/guestbook/:id", perm(db, "module:guestbook"), adminH.AdminGuestbookDel)
				admin.GET("/wallets", perm(db, "module:wallet"), adminH.Wallets)
				admin.PUT("/wallets/:id", perm(db, "module:wallet"), adminH.WalletSet)

				// 家族 / 同城 / T台秀
				admin.GET("/families", perm(db, "module:families"), adminH.Families)
				admin.PUT("/families/:id/status", perm(db, "module:families"), adminH.FamilyStatus)
				admin.PUT("/families/:id/ann", perm(db, "module:families"), adminH.FamilyAnn)
				admin.GET("/families/pending", perm(db, "module:families"), adminH.PendingFamilies)
				admin.PUT("/families/:id/feature", perm(db, "module:families"), adminH.FamilyFeature)
				admin.PUT("/families/:id/review", perm(db, "module:families"), adminH.FamilyReview)
				admin.GET("/tongcheng", perm(db, "module:tongcheng"), cityH.AdminTree)
				admin.POST("/tongcheng/provinces", perm(db, "module:tongcheng"), cityH.AdminCreateProvince)
				admin.PUT("/tongcheng/provinces/:id", perm(db, "module:tongcheng"), cityH.AdminUpdateProvince)
				admin.DELETE("/tongcheng/provinces/:id", perm(db, "module:tongcheng"), cityH.AdminDeleteProvince)
				admin.POST("/tongcheng/cities", perm(db, "module:tongcheng"), cityH.AdminCreateCity)
				admin.PUT("/tongcheng/cities/:id", perm(db, "module:tongcheng"), cityH.AdminUpdateCity)
				admin.DELETE("/tongcheng/cities/:id", perm(db, "module:tongcheng"), cityH.AdminDeleteCity)
				admin.GET("/tongcheng/managers/:id", perm(db, "module:tongcheng"), cityH.AdminManagers)
				admin.POST("/tongcheng/managers", perm(db, "module:tongcheng"), cityH.AdminManagerAdd)
				admin.DELETE("/tongcheng/managers/:id", perm(db, "module:tongcheng"), cityH.AdminManagerDelete)
				admin.GET("/ttou", perm(db, "module:ttou"), adminH.Ttou)
				admin.PUT("/ttou", perm(db, "module:ttou"), adminH.TtouSet)
				admin.DELETE("/ttou", perm(db, "module:ttou"), adminH.TtouClear)
				admin.GET("/ttou/applies", perm(db, "module:ttou"), adminH.TtouApplies)
				admin.POST("/ttou/applies/:id/accept", perm(db, "module:ttou"), adminH.TtouApplyAccept)

				// 福利院捐款上榜记录管理
				admin.GET("/fla-donations", perm(db, "module:fla"), flaH.AdminList)
				admin.POST("/fla-donations", perm(db, "module:fla"), flaH.AdminCreate)
				admin.PUT("/fla-donations/:id", perm(db, "module:fla"), flaH.AdminUpdate)
				admin.DELETE("/fla-donations/:id", perm(db, "module:fla"), flaH.AdminDelete)

				// 福利院·慈善基金管理
				admin.GET("/welfare", perm(db, "module:welfare"), welfareH.AdminStats)
				admin.POST("/welfare/pool", perm(db, "module:welfare"), welfareH.AdminSetPool)
				admin.GET("/welfare/claims", perm(db, "module:welfare"), welfareH.AdminClaims)
				admin.POST("/welfare/claims", perm(db, "module:welfare"), welfareH.AdminCreateClaim)
				admin.DELETE("/welfare/claims/:id", perm(db, "module:welfare"), welfareH.AdminDeleteClaim)
				admin.GET("/welfare/donations", perm(db, "module:welfare"), welfareH.AdminDonations)
				admin.DELETE("/welfare/donations/:id", perm(db, "module:welfare"), welfareH.AdminDeleteDonate)
				admin.PUT("/users/:id/status", perm(db, "module:users"), adminH.UserStatus)
				admin.PUT("/users/:id/password", perm(db, "module:users"), adminH.ResetPassword)
				admin.PUT("/users/:id/roles", perm(db, "module:users"), adminH.UserRoles)
				admin.PUT("/users/:id/badges", perm(db, "module:badges"), badgeH.UserBadges)
				admin.PUT("/users/:id/extras", perm(db, "module:users"), adminH.UserExtras)
				admin.POST("/users/batch-delete", perm(db, "module:users"), adminH.UsersBatchDelete)
				admin.GET("/phones", perm(db, "module:userPhones"), adminH.Phones)
				admin.POST("/phones/:id/audit", perm(db, "module:userPhones"), adminH.PhoneAudit)
				admin.DELETE("/phones/:id", perm(db, "module:userPhones"), adminH.PhoneDelete)

				admin.GET("/badges", perm(db, "module:badges"), badgeH.AdminList)
				admin.POST("/badges", perm(db, "module:badges"), badgeH.Create)
				admin.PUT("/badges/:id", perm(db, "module:badges"), badgeH.Update)
				admin.DELETE("/badges/:id", perm(db, "module:badges"), badgeH.Delete)
				admin.GET("/user-badges", perm(db, "module:badges"), badgeH.AdminUserBadges)
				admin.POST("/user-badges/grant", perm(db, "module:badges"), badgeH.GrantOne)
				admin.PUT("/user-badges/:id", perm(db, "module:badges"), badgeH.AdminUserBadgeUpdate)
				admin.DELETE("/user-badges/:id", perm(db, "module:badges"), badgeH.AdminUserBadgeDel)

				admin.GET("/games", perm(db, "module:games"), gameH.AdminList)
				admin.POST("/games", perm(db, "module:games"), gameH.Create)
				admin.PUT("/games/:id", perm(db, "module:games"), gameH.Update)
				admin.DELETE("/games/:id", perm(db, "module:games"), gameH.Delete)

				admin.GET("/resources", perm(db, "module:resources"), resH.List)
				admin.PUT("/resources/:id", perm(db, "module:resources"), resH.Update)
				admin.POST("/resources/sync", perm(db, "module:resources"), resH.Sync)
				admin.POST("/resources/upload", perm(db, "module:resources"), resH.Upload)
				admin.DELETE("/resources/:id", perm(db, "module:resources"), resH.Delete)

				admin.GET("/boards", perm(db, "module:boards"), adminH.Boards)
				admin.POST("/boards", perm(db, "module:boards"), adminH.CreateBoard)
				admin.PUT("/boards/:id", perm(db, "module:boards"), adminH.UpdateBoard)
				admin.DELETE("/boards/:id", perm(db, "module:boards"), adminH.DeleteBoard)
				admin.GET("/board-categories", perm(db, "module:boardCategories"), adminH.BoardCategories)
				admin.POST("/board-categories", perm(db, "module:boardCategories"), adminH.CreateBoardCategory)
				admin.PUT("/board-categories/:id", perm(db, "module:boardCategories"), adminH.UpdateBoardCategory)
				admin.DELETE("/board-categories/:id", perm(db, "module:boardCategories"), adminH.DeleteBoardCategory)
				admin.GET("/board-members", perm(db, "module:boards"), adminH.BoardMembers)
				admin.POST("/board-members", perm(db, "module:boards"), adminH.BoardMemberAdd)
				admin.DELETE("/board-members/:boardId/:userId", perm(db, "module:boards"), adminH.BoardMemberRemove)
				admin.GET("/word-filters", perm(db, "module:wordFilters"), adminH.WordFilters)
				admin.POST("/word-filters", perm(db, "module:wordFilters"), adminH.CreateWordFilter)
				admin.POST("/word-filters/bulk", perm(db, "module:wordFilters"), adminH.BulkCreateWordFilter)
				admin.PUT("/word-filters/:id", perm(db, "module:wordFilters"), adminH.UpdateWordFilter)
				admin.DELETE("/word-filters/:id", perm(db, "module:wordFilters"), adminH.DeleteWordFilter)

				admin.GET("/threads", perm(db, "module:threads"), adminH.Threads)
				admin.PUT("/threads/:id", perm(db, "module:threads"), adminH.UpdateThread)
				admin.DELETE("/threads/:id", perm(db, "module:threads"), adminH.DeleteThread)
				admin.DELETE("/replies/:id", perm(db, "module:threads"), adminH.DeleteReply)

				admin.GET("/announcements", perm(db, "module:announcements"), adminH.Announcements)
				admin.POST("/announcements", perm(db, "module:announcements"), adminH.CreateAnnouncement)
				admin.PUT("/announcements/:id", perm(db, "module:announcements"), adminH.UpdateAnnouncement)
				admin.DELETE("/announcements/:id", perm(db, "module:announcements"), adminH.DeleteAnnouncement)

				admin.GET("/roles", perm(db, "module:roles"), adminH.Roles)
				admin.GET("/permissions", perm(db, "module:roles"), adminH.Permissions)
				// 当前管理员的权限码（前端菜单按角色过滤）
				admin.GET("/my-perms", adminH.MyPerms)
				admin.POST("/roles", perm(db, "module:roles"), adminH.CreateRole)
				admin.PUT("/roles/:id/perms", perm(db, "module:roles"), adminH.UpdateRolePerms)
				admin.DELETE("/roles/:id", perm(db, "module:roles"), adminH.DeleteRole)

				// 空间管理
				admin.GET("/spaces", perm(db, "module:spaces"), spaceH.AdminSpaces)
				admin.PUT("/spaces/:id/status", perm(db, "module:spaces"), spaceH.AdminSpaceStatus)
				admin.DELETE("/spaces/:id", perm(db, "module:spaces"), spaceH.AdminSpaceDelete)
				admin.GET("/spaces/:userId/moods", perm(db, "module:spaces"), spaceH.AdminSpaceMoods)
				admin.DELETE("/moods/:id", perm(db, "module:spaces"), spaceH.AdminDeleteMood)
				admin.GET("/spaces/:userId/articles", perm(db, "module:spaces"), spaceH.AdminSpaceArticles)
				admin.DELETE("/articles/:id", perm(db, "module:spaces"), spaceH.AdminDeleteArticle)
				admin.GET("/spaces/:userId/albums", perm(db, "module:spaces"), spaceH.AdminSpaceAlbums)
				admin.GET("/spaces/:userId/messages", perm(db, "module:spaces"), spaceH.AdminSpaceMessages)
				admin.GET("/spaces/:userId/visitors", perm(db, "module:spaces"), spaceH.AdminSpaceVisitors)
				admin.GET("/albums/:id/photos", perm(db, "module:spaces"), spaceH.AdminAlbumPhotos)
				admin.DELETE("/photos/:id", perm(db, "module:spaces"), spaceH.AdminDeletePhoto)
				admin.DELETE("/albums/:id", perm(db, "module:spaces"), spaceH.AdminDeleteAlbum)
				admin.DELETE("/space-messages/:id", perm(db, "module:spaces"), spaceH.AdminDeleteSpaceMessage)
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
