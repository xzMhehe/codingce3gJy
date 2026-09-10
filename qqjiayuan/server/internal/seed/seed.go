package seed

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/model"
)

// Run 建表 + 幂等初始化数据
func Run(db *gorm.DB, staticDir string) {
	// user_badges 旧结构兜底：老表为 (user_id,badge_id) 复合主键且无自增 id，
	// AutoMigrate 直接加 id 主键会报 Multiple primary key defined，先升级（幂等）
	if db.Migrator().HasTable("user_badges") && !db.Migrator().HasColumn("user_badges", "id") {
		db.Exec(`ALTER TABLE user_badges
			DROP PRIMARY KEY,
			ADD COLUMN id bigint unsigned NOT NULL AUTO_INCREMENT,
			ADD PRIMARY KEY (id),
			ADD COLUMN sort int DEFAULT 0,
			ADD COLUMN granted_at datetime NULL,
			ADD COLUMN expire_at datetime NULL,
			ADD UNIQUE KEY uk_ub (user_id, badge_id)`)
		db.Exec("UPDATE user_badges SET granted_at = NOW() WHERE granted_at IS NULL")
	}

	err := db.AutoMigrate(
		&model.User{}, &model.Role{}, &model.Permission{},
		&model.Board{}, &model.Thread{}, &model.Reply{},
		&model.Announcement{}, &model.SignIn{},
		&model.Friendship{}, &model.FriendApply{}, &model.FriendBlack{}, &model.PrivateMessage{},
		&model.ChatMessage{}, &model.Notification{},
		&model.Badge{},
		&model.Game{},
		&model.Resource{},
		&model.Space{}, &model.Mood{}, &model.MoodComment{},
		&model.Article{}, &model.Album{}, &model.Photo{},
		&model.SpaceFile{},
		&model.SpaceMessage{}, &model.Visitor{},
		&model.BankAccount{}, &model.WorkRecord{},
		&model.Family{}, &model.FamilyMember{}, &model.FamilySignIn{},
		&model.FamilyActivity{},
		&model.FamilyFavorite{},
		&model.FamilyLdUser{}, &model.FamilyLdLog{},
		&model.FamilyWarBattle{}, &model.FamilyWarLife{}, &model.FamilyWarChat{},
		&model.Book{}, &model.BookChapter{}, &model.BookComment{}, &model.BookShelf{}, &model.ThreadFavorite{},
		&model.FriendGroup{}, &model.FriendGroupItem{},
		&model.GardenPlot{}, &model.MyGame{}, &model.UserFlower{},
		&model.GardenActivity{}, &model.Donation{}, &model.PlazaSection{},
		&model.Garden{}, &model.GardenSeed{}, &model.GardenMap{}, &model.GardenBag{},
		&model.GardenBottle{}, &model.GardenGift{}, &model.GardenMix{},
		&model.GardenMsg{}, &model.GardenMapLog{}, &model.GardenLandLog{},
		&model.GardenElf{}, &model.GardenElfLog{},
		&model.GardenSign{}, &model.GardenSignReward{},
		&model.Farm{}, &model.FarmSeed{}, &model.FarmMuck{}, &model.FarmTrap{},
		&model.FarmLand{}, &model.FarmBag{}, &model.FarmMsg{}, &model.FarmSlave{}, &model.FarmSteal{},
		&model.ParkUser{}, &model.CarShop{}, &model.CarGarage{}, &model.CarStop{}, &model.CarLog{}, &model.CarMsg{},
		&model.NoblePlan{}, &model.NobleLevel{}, &model.Good{}, &model.UserGood{}, &model.Setting{},
		&model.MoneyShop{},
		&model.WalletLog{},
		&model.Marriage{},
		&model.ThreadVote{}, &model.ReplyVote{}, &model.ThreadGift{}, &model.ThreadFlower{},
		&model.Report{},
		&model.TtouApply{}, &model.TtouWorship{},
		&model.Home{}, &model.HomeNews{}, &model.HomeFavorite{}, &model.UserContact{},
		&model.Invite{}, &model.GuestBook{}, &model.GuestReply{},
		&model.PhoneAudit{},
		&model.SiteArticleCategory{}, &model.SiteArticle{}, &model.SiteArticleComment{},
		&model.ShopCategory{}, &model.Shop{}, &model.ShopGoods{}, &model.ShopOrder{}, &model.ShopComment{},
		&model.ArticleComment{},
		&model.UserAddress{}, &model.UserDocument{}, &model.UserProtection{}, &model.UserLog{},
		&model.BoardCategory{}, &model.BoardMember{}, &model.StickyReply{},
		&model.ThreadPoll{}, &model.ThreadPollOption{}, &model.ThreadPollVote{},
		&model.ThreadReward{}, &model.ThreadRewardLog{}, &model.ThreadFloor{},
		&model.ThreadAttachment{}, &model.WordFilter{},
		&model.CityManager{},
		&model.UserBadge{},
	)
	if err != nil {
		log.Fatalf("建表失败: %v", err)
	}
	db.Exec("ALTER TABLE users AUTO_INCREMENT = 10000")
	db.Exec("ALTER TABLE threads AUTO_INCREMENT = 10000")

	// 会员勋章：注册自定义 join model（含排序/授予时间/过期时间）
	db.SetupJoinTable(&model.User{}, "Badges", &model.UserBadge{})

	// 勋章商店/会员勋章列兜底（幂等）
	mig := db.Migrator()
	for _, col := range []string{"price", "period", "sort"} {
		if !mig.HasColumn("badges", col) {
			db.Exec("ALTER TABLE badges ADD COLUMN " + col + " int DEFAULT 0")
		}
	}
	for _, col := range []string{"sort", "granted_at", "expire_at"} {
		if !mig.HasColumn("user_badges", col) {
			db.Exec("ALTER TABLE user_badges ADD COLUMN " + col + " datetime NULL")
		}
	}
	db.Exec("ALTER TABLE user_badges MODIFY COLUMN sort int DEFAULT 0")
	db.Exec("UPDATE user_badges SET granted_at = NOW() WHERE granted_at IS NULL")
	// 清理已过期勋章（复刻诺哈：自动删除过期勋章）
	db.Exec("DELETE FROM user_badges WHERE expire_at IS NOT NULL AND expire_at < NOW()")

	// 兜底：个别环境 AutoMigrate 对存量表可能漏加列，这里显式补齐（幂等）
	m := db.Migrator()
	for _, col := range []string{"yuanbao", "jinzuan", "youquan"} {
		if !m.HasColumn("users", col) {
			db.Exec("ALTER TABLE users ADD COLUMN " + col + " int DEFAULT 0")
		}
	}
	if !m.HasColumn("users", "avatar_base64") {
		db.Exec("ALTER TABLE users ADD COLUMN avatar_base64 longtext")
	}
	// 用户模块（对齐诺哈 wap_user）存量表补列（幂等）
	for _, col := range []string{"hours", "friend_policy", "paid"} {
		if !m.HasColumn("users", col) {
			db.Exec("ALTER TABLE users ADD COLUMN " + col + " int DEFAULT 0")
		}
	}
	if !m.HasColumn("users", "birth_type") {
		db.Exec("ALTER TABLE users ADD COLUMN birth_type int DEFAULT 1")
	}
	for _, col := range []string{"solar", "lunar"} {
		if !m.HasColumn("users", col) {
			db.Exec("ALTER TABLE users ADD COLUMN " + col + " varchar(20) DEFAULT ''")
		}
	}
	if !m.HasColumn("users", "config") {
		db.Exec("ALTER TABLE users ADD COLUMN config varchar(50) DEFAULT ''")
	}
	if !m.HasColumn("users", "add_ip") {
		db.Exec("ALTER TABLE users ADD COLUMN add_ip varchar(45) DEFAULT ''")
	}
	if !m.HasColumn("users", "last_ip") {
		db.Exec("ALTER TABLE users ADD COLUMN last_ip varchar(45) DEFAULT ''")
	}
	if !m.HasColumn("users", "pay_pass") {
		db.Exec("ALTER TABLE users ADD COLUMN pay_pass varchar(100) DEFAULT ''")
	}
	// 老用户补默认个性配置（诺哈 wap_user.config CSV）
	db.Exec("UPDATE users SET config = '10,1200,1500,1200,0' WHERE config IS NULL OR config = ''")
	// 种子演示帖统一设为已发布（audit_status=1），否则论坛/详情/活动专区不可见
	db.Exec("UPDATE threads SET audit_status = 1 WHERE audit_status = 0 AND status = 1")

	// 论坛重构：boards/threads 新增列兜底补齐（幂等）
	bcols := map[string]string{
		"category_id": "ALTER TABLE boards ADD COLUMN category_id bigint DEFAULT 0",
		"notice":      "ALTER TABLE boards ADD COLUMN notice text",
		"tags":        "ALTER TABLE boards ADD COLUMN tags varchar(100) DEFAULT ''",
		"moderator_id": "ALTER TABLE boards ADD COLUMN moderator_id bigint DEFAULT 0",
		"members_only": "ALTER TABLE boards ADD COLUMN members_only int DEFAULT 0",
	}
	for col, sql := range bcols {
		if !m.HasColumn("boards", col) {
			db.Exec(sql)
		}
	}
	tcols := map[string]string{
		"is_head":      "ALTER TABLE threads ADD COLUMN is_head int DEFAULT 0",
		"is_lock":      "ALTER TABLE threads ADD COLUMN is_lock int DEFAULT 0",
		"is_recom":     "ALTER TABLE threads ADD COLUMN is_recom int DEFAULT 0",
		"is_notice":    "ALTER TABLE threads ADD COLUMN is_notice int DEFAULT 0",
		"type":         "ALTER TABLE threads ADD COLUMN type int DEFAULT 0",
		"audit_status": "ALTER TABLE threads ADD COLUMN audit_status int DEFAULT 1",
	}
	for col, sql := range tcols {
		if !m.HasColumn("threads", col) {
			db.Exec(sql)
		}
	}
	if !m.HasColumn("replies", "parent_reply_id") {
		db.Exec("ALTER TABLE replies ADD COLUMN parent_reply_id bigint DEFAULT 0")
	}
	if !m.HasColumn("goods", "youquan_price") {
		db.Exec("ALTER TABLE goods ADD COLUMN youquan_price int DEFAULT 0")
	}
	// 道具商城复刻诺哈商店中心：库存/销量/结束时间补列（幂等；首次加库存列时回填，避免反复覆盖管理员设置）
	if !m.HasColumn("goods", "stock") {
		db.Exec("ALTER TABLE goods ADD COLUMN stock int DEFAULT 0")
		db.Exec("UPDATE goods SET stock = 999")
	}
	if !m.HasColumn("goods", "sales") {
		db.Exec("ALTER TABLE goods ADD COLUMN sales int DEFAULT 0")
	}
	if !m.HasColumn("goods", "end_time") {
		db.Exec("ALTER TABLE goods ADD COLUMN end_time datetime NULL")
	}
	// 存量商品补销售时间（幂等）
	db.Exec("UPDATE goods SET add_time = NOW() WHERE add_time IS NULL")
	// 超Q/蓝钻复刻：每日成长时间 + 方案新字段
	if !m.HasColumn("users", "blue_ptime") {
		db.Exec("ALTER TABLE users ADD COLUMN blue_ptime datetime NULL")
	}
	if !m.HasColumn("users", "qq_ptime") {
		db.Exec("ALTER TABLE users ADD COLUMN qq_ptime datetime NULL")
	}
	for _, col := range []string{"speed", "`limit`", "stock", "sales"} {
		if !m.HasColumn("noble_plans", col) {
			db.Exec("ALTER TABLE noble_plans ADD COLUMN " + col + " int DEFAULT 0")
		}
	}
	if !m.HasColumn("wallet_logs", "remark") {
		db.Exec("ALTER TABLE wallet_logs ADD COLUMN remark varchar(50) DEFAULT ''")
	}
	// 同城客栈复刻：板块补区号/创建人/人气列（参考诺哈 wap_bbs.ccode/uid/click，幂等）
	for _, col := range []string{"city_code", "creator_id", "click"} {
		if !m.HasColumn("boards", col) {
			if col == "city_code" {
				db.Exec("ALTER TABLE boards ADD COLUMN city_code varchar(10) DEFAULT ''")
			} else {
				db.Exec("ALTER TABLE boards ADD COLUMN " + col + " bigint DEFAULT 0")
			}
		}
	}

	seedWordFilters(db)
	seedRBAC(db)
	seedBoards(db)
	seedBoardCategories(db)
	seedAnnouncements(db)
	seedUsersAndContent(db)
	seedTongcheng(db)
	seedBadges(db)
	seedGameBoards(db)
	seedGongtan(db)
	seedMigrate2026(db)
	seedGames(db)
	seedFamilies(db)
	seedFamilyPatch(db)
	seedFamilyBoards(db)
	seedBooks(db)
	seedBookChapters(db)
	seedGardenActivities(db)
	seedGardenData(db)
	seedGardenSignRewards(db)
	seedFarmData(db)
	seedParkData(db)
	seedPlazaSections(db)
	seedNoblePlans(db)
	seedNobleLevels(db)
	seedGoods(db)
	seedMoneyShop(db)
	seedResources(db, staticDir)
	seedGuestbook(db)
	seedSiteArticles(db)
	seedShop(db)
	seedActivities(db)
	seedFriendMigrate(db)
	fmt.Println("数据初始化完成")
}

// seedResources 资源库种子：勋章/头像/logo/特权8级 + 扫描 static 目录登记其余文件
func seedResources(db *gorm.DB, staticDir string) {
	var count int64
	db.Model(&model.Resource{}).Count(&count)
	if count == 0 {
		add := func(sub, file, cat, name string, level int) {
			db.Create(&model.Resource{File: sub + "/" + file, Category: cat, Name: name, Level: level, Status: 1})
		}
		for _, f := range model.BadgeIconPresets {
			add("picture", f, "badge", f, 0)
		}
		for _, f := range model.AvatarPresets {
			add("picture", f, "avatar", f, 0)
		}
		for _, f := range model.GameLogoPresets {
			add("image", f, "game", f, 0)
		}
		for _, pv := range model.PrivSeed {
			add("", pv.File, "priv", pv.Name, pv.Level)
		}
	}
	// 扫描目录登记新文件
	syncDir(db, staticDir)
}

// seedFamilies 示例家族（幂等：family 表为空时才写入）
func seedFamilies(db *gorm.DB) {
	var count int64
	db.Model(&model.Family{}).Count(&count)
	if count > 0 {
		return
	}
	byNick := func(nick string) uint {
		var u model.User
		db.Where("nickname = ?", nick).First(&u)
		return u.ID
	}
	spec := []struct {
		name, slogan, desc, ann, ownerNick, category string
		battle                                        int
		members                                       []string
	}{
		{"清风明月", "轻风徐来，明月入怀", "以文会友，共话家常。", "欢迎回家，常来常往。", "云起", "舞文弄墨", 320, []string{"云起", "咏荷", "闲云野鹤"}},
		{"与世无争", "与世无争，不问西东", "恬淡生活，其乐融融。", "兄弟姐妹们常回家看看。", "安珞", "情感男女", 260, []string{"安珞", "蓝天"}},
		{"断念阁", "聚是一团火，散是满天星", "以舞会友，以歌传情。", "新老朋友皆可入阁。", "　　瞿詺南　", "青春校园", 180, []string{"　　瞿詺南　"}},
	}
	for _, s := range spec {
		ownerID := byNick(s.ownerNick)
		fam := model.Family{Name: s.name, Slogan: s.slogan, Description: s.desc,
			Announcement: s.ann, Category: s.category, OwnerID: ownerID, TreeLevel: 2 + randInt(3), TreeExp: 150 + randInt(200), BattleScore: s.battle}
		db.Create(&fam)
		db.Create(&model.FamilyMember{FamilyID: fam.ID, UserID: ownerID, Role: "owner", Exp: 320})
		for _, nick := range s.members {
			if nick == s.ownerNick {
				continue
			}
			db.Create(&model.FamilyMember{FamilyID: fam.ID, UserID: byNick(nick), Role: "member", Exp: 30 + randInt(80)})
		}
	}
}

// seedFamilyPatch 给已有家族补类别、并在家族动态表为空时写入示例动态（幂等）
func seedFamilyPatch(db *gorm.DB) {
	patches := map[string]string{"云起": "舞文弄墨", "安珞": "情感男女", "　　瞿詺南　": "青春校园"}
	for nick, cat := range patches {
		var owner model.User
		db.Where("nickname = ?", nick).First(&owner)
		if owner.ID == 0 {
			continue
		}
		db.Model(&model.Family{}).Where("owner_id = ? AND (category IS NULL OR category = '')", owner.ID).Update("category", cat)
	}

	var n int64
	db.Model(&model.FamilyActivity{}).Count(&n)
	if n > 0 {
		return
	}
	byNick := func(nick string) uint {
		var u model.User
		db.Where("nickname = ?", nick).First(&u)
		return u.ID
	}
	famOwner := func(ownerNick string) uint {
		ownerID := byNick(ownerNick)
		var f model.Family
		db.Where("owner_id = ?", ownerID).First(&f)
		return f.ID
	}
	samples := []string{
		"在家族签到",
		"抚摸/拥抱了守护树",
		"参加了家族乐斗，战胜了对手",
		"分享了家族公告",
	}
	if fid := famOwner("云起"); fid > 0 {
		for _, s := range samples {
			db.Create(&model.FamilyActivity{FamilyID: fid, UserID: byNick("云起"), Content: s})
		}
	}
	if fid := famOwner("安珞"); fid > 0 {
		db.Create(&model.FamilyActivity{FamilyID: fid, UserID: byNick("安珞"), Content: "在家族签到"})
	}
}

// seedFamilyBoards 家族大看台板块 + 特色家族标记 + 示例活动帖（幂等）
func seedFamilyBoards(db *gorm.DB) {
	var root model.Board
	db.Where("name = ?", "家族大厅").First(&root)
	if root.ID == 0 {
		return
	}
	var board model.Board
	db.Where("parent_id = ? AND name = ?", root.ID, "家族大看台").First(&board)
	if board.ID == 0 {
		board = model.Board{ParentID: root.ID, Name: "家族大看台", Description: "家族活动、家族大事一览"}
		db.Create(&board)
	}
	// 特色家族标记
	db.Model(&model.Family{}).Where("name = ? AND status = 1", "清风明月").Update("is_feature", 1)
	db.Model(&model.Family{}).Where("name = ? AND status = 1", "与世无争").Update("is_feature", 1)
	// 示例活动帖（该板块无帖子时写入）
	var n int64
	db.Model(&model.Thread{}).Where("board_id = ?", board.ID).Count(&n)
	if n > 0 {
		return
	}
	byNick := func(nick string) uint {
		var u model.User
		db.Where("nickname = ?", nick).First(&u)
		return u.ID
	}
	spec := []struct {
		title, content, by string
		views              int
	}{
		{"【清风明月·中秋活动】月圆人团圆，回帖赢金币", "中秋佳节，家族全体成员一起赏月吃月饼，回帖即可获得金币奖励！", "云起", 286},
		{"【与世无争·周末聚会】来家族大厅唠唠嗑", "周末啦，兄弟姐妹们快来家族大看台集合，聊聊这一周的趣事～", "安珞", 190},
		{"【断念阁】新成员入阁欢迎帖", "欢迎新伙伴加入断念阁，新人报道帖～", "　　瞿詺南　", 120},
	}
	for _, s := range spec {
		uid := byNick(s.by)
		if uid == 0 {
			continue
		}
		db.Create(&model.Thread{BoardID: board.ID, UserID: uid, Title: s.title,
			Content: s.content, ViewCount: s.views, Status: 1})
	}
}

// seedBooks 书城示例书籍（幂等）
func seedBooks(db *gorm.DB) {
	var count int64
	db.Model(&model.Book{}).Count(&count)
	if count > 0 {
		return
	}
	books := []model.Book{
		{Title: "剑影江湖", Author: "家园侠客", Category: "武侠", Intro: "乱世出英雄，一柄长剑闯天涯。家国恩怨，儿女情长。", Status: "连载", Recommend: 1, Rating: "★★★★★"},
		{Title: "山河故人", Author: "云深不知", Category: "武侠", Intro: "倦鸟归林，故人相逢。旧时刀剑，今朝煮茶。", Status: "连载", Recommend: 0, Rating: "★★★★"},
		{Title: "仲夏绮梦", Author: "木槿昔年", Category: "言情", Intro: "那年仲夏，蝉鸣与少年，都是青春最美的模样。", Status: "完结", Recommend: 1, Rating: "★★★★★"},
		{Title: "碎碎念", Author: "文墨", Category: "都市", Intro: "都市里的烟火气，柴米油盐也动人。", Status: "连载", Recommend: 0, Rating: "★★★"},
		{Title: "盗墓", Author: "夜行人", Category: "灵异", Intro: "地下的秘密，随着灯火一盏盏熄灭。", Status: "连载", Recommend: 1, Rating: "★★★★"},
		{Title: "忘忧奶茶店", Author: "小甜", Category: "都市", Intro: "一杯奶茶，换你一个故事。", Status: "完结", NewBook: 1, Rating: "★★★★"},
		{Title: "六零：城里小白菜回乡当团宠", Author: "阿园", Category: "言情", Intro: "穿越六零，从城里小白菜到乡间团宠。", Status: "连载", NewBook: 1, Recommend: 1, Rating: "★★★★★"},
		{Title: "大汉宏图", Author: "子夜", Category: "武侠", Intro: "铁血王朝，宏图霸业，一将功成万骨枯。", Status: "连载", NewBook: 1, Rating: "★★★★"},
	}
	for _, b := range books {
		db.Create(&b)
	}
}

// seedBookChapters 书城章节+书评种子（幂等：给每本书补齐章节，有评论的书跳过）
func seedBookChapters(db *gorm.DB) {
	var books []model.Book
	db.Order("id ASC").Find(&books)
	if len(books) == 0 {
		return
	}
	adminID := idByUsername(db, "10000")
	for _, b := range books {
		var n int64
		db.Model(&model.BookChapter{}).Where("book_id = ?", b.ID).Count(&n)
		if n == 0 {
			chs := []model.BookChapter{
				{BookID: b.ID, Title: "第一章 缘起", Sort: 1, Content: "故事，要从很久以前说起。\n\n那一年风起云涌，少年背起行囊，独自踏上了远行的路。谁也不知道，这趟旅程将彻底改变他的命运……\n\n（本章为示例章节，可到管理后台-书城管理-章节管理中编辑。）"},
				{BookID: b.ID, Title: "第二章 风波", Sort: 2, Content: "风波骤起。\n\n镇口的茶棚里，一位灰衣人放下茶碗，只说了一句话，满座皆惊。\n\n他说的正是那个名字——那个所有人都以为再也不会出现的名字。"},
				{BookID: b.ID, Title: "第三章 入局", Sort: 3, VIP: 1, Content: "所谓入局，便再无回头路。\n\n少年握紧了手中长剑，望向远方的群山。他知道，从这一刻起，自己不再是旁观者。"},
			}
			for i := range chs {
				db.Create(&chs[i])
			}
		}
		var cn int64
		db.Model(&model.BookComment{}).Where("book_id = ?", b.ID).Count(&cn)
		if cn == 0 && adminID > 0 {
			db.Create(&model.BookComment{BookID: b.ID, UserID: adminID, Score: 5, Content: "家园书友推荐好书，值得一读！"})
		}
	}
}

// seedGardenActivities 花园示例活动（幂等；已存在则补齐 desc/needs/reward）
func seedGardenActivities(db *gorm.DB) {
	acts := []model.GardenActivity{
		{Title: "春天的爱恋", Desc: "春天来了，魔法花园里的花儿沐浴着温馨的春风和绵绵的春雨含苞待放着，想要在春天绽放自己最美的身影。花仙子陶醉在浓浓的春意中，撒下了许多象征着爱情的朝暮盈霄花种子，快去寻找吧！",
			Needs: `[{"flower":"红玫瑰","n":6},{"flower":"红桃花","n":6},{"flower":"红色勿忘我","n":6},{"flower":"红色烈焰焚情","n":6}]`, Reward: "朝暮盈霄花"},
		{Title: "小魔女的烦恼", Desc: "小魔女：“每次聚会都要hold住全场，不够鲜花装扮自己怎么办呀！谁能送我一些鲜花，我会给TA丰厚的回报哟！”",
			Needs: `[{"flower":"红色菊花","n":3},{"flower":"红色野花","n":3},{"flower":"红桃花","n":3},{"flower":"红兰花","n":3}]`, Reward: "夜魔南瓜花"},
		{Title: "花仙子的新房子", Desc: "花仙子：“555，我的花房有些时候没有修整了，天气开始转凉，我都被冻感冒几次了，我急需一些花来重新补整我的花房，请你帮我去采些丁香/樱花/野花/梅花/兰花，我会拿最新出的步步高升花种回报你哦！”",
			Needs: `[{"flower":"红丁香","n":1},{"flower":"红樱花","n":1},{"flower":"红色野花","n":1},{"flower":"红梅花","n":1},{"flower":"红兰花","n":1}]`, Reward: "步步高升"},
		{Title: "寻找遗失的碎片", Desc: "想要开启花园的精灵花册，需要集齐对应的珍惜碎片，快来用你种出来的鲜花和我兑换！",
			Needs: `[{"flower":"红玫瑰","n":33},{"flower":"黄玫瑰","n":33},{"flower":"白玫瑰","n":33},{"flower":"粉玫瑰","n":33},{"flower":"银玫瑰","n":3},{"flower":"金玫瑰","n":1}]`, Reward: "玫瑰金碎片"},
	}
	for _, a := range acts {
		var old model.GardenActivity
		if err := db.Where("title = ?", a.Title).First(&old).Error; err == nil {
			// 老记录补齐字段
			db.Model(&old).Updates(map[string]interface{}{"desc": a.Desc, "needs": a.Needs, "reward": a.Reward})
		} else {
			db.Create(&a)
		}
	}
}

// seedGardenData 魔法花园种子/图鉴/合成配方（幂等：按名称去重，老库已有不覆盖）
func seedGardenData(db *gorm.DB) {
	seeds := []model.GardenSeed{
		{Name: "向日葵", DType: 0, Level: 1, Price: 5, Seed: 1, Ling: 1, Buds: 1, Less: 2, More: 4, Remark: "沉默的爱，勇敢追求幸福。"},
		{Name: "玫瑰花", DType: 0, Level: 1, Price: 10, Seed: 1, Ling: 1, Buds: 2, Less: 3, More: 5, Remark: "爱情与热恋，勇敢表达。"},
		{Name: "郁金香", DType: 0, Level: 2, Price: 20, Seed: 2, Ling: 2, Buds: 2, Less: 3, More: 6, Remark: "博爱、体贴、高雅。"},
		{Name: "月光花", DType: 0, Level: 3, Price: 40, Seed: 2, Ling: 2, Buds: 3, Less: 4, More: 8, Remark: "幸福与美好的憧憬。"},
		{Name: "百合花", DType: 0, Level: 4, Price: 80, Seed: 3, Ling: 3, Buds: 3, Less: 5, More: 10, Remark: "百年好合，纯洁无瑕。"},
		{Name: "牡丹", DType: 0, Level: 5, Price: 150, Seed: 3, Ling: 4, Buds: 4, Less: 6, More: 12, Remark: "圆满、浓情、富贵。"},
		{Name: "蓝色妖姬", DType: 0, Level: 7, Price: 300, Seed: 4, Ling: 5, Buds: 5, Less: 8, More: 16, Remark: "奇迹与不可能的爱。"},
		{Name: "樱花", DType: 0, Level: 9, Price: 500, Seed: 5, Ling: 6, Buds: 6, Less: 10, More: 20, Remark: "生命、幸福、热烈。"},
		{Name: "天山雪莲", DType: 0, Level: 12, Price: 800, Seed: 6, Ling: 8, Buds: 8, Less: 12, More: 24, Remark: "纯洁的爱、坚贞。"},
		// 魔法屋合成产物（特殊种子）
		{Name: "银色菊花", DType: 1, Level: 3, Price: 0, Seed: 2, Ling: 2, Buds: 2, Less: 4, More: 8, Remark: "真诚的思念。"},
		{Name: "银野花", DType: 1, Level: 4, Price: 0, Seed: 2, Ling: 3, Buds: 3, Less: 5, More: 10, Remark: "野性之美。"},
		{Name: "端阳花", DType: 1, Level: 5, Price: 0, Seed: 3, Ling: 3, Buds: 3, Less: 6, More: 12, Remark: "端午安康。"},
		{Name: "银友谊花", DType: 1, Level: 6, Price: 0, Seed: 3, Ling: 4, Buds: 4, Less: 7, More: 14, Remark: "友谊长存。"},
		{Name: "银色烈焰焚情", DType: 1, Level: 8, Price: 0, Seed: 4, Ling: 5, Buds: 5, Less: 8, More: 16, Remark: "炽热的爱。"},
		{Name: "金色烈焰焚情", DType: 1, Level: 10, Price: 0, Seed: 5, Ling: 6, Buds: 6, Less: 10, More: 20, Remark: "永恒的爱。"},
		// 道具（对齐参考站 dz_list：魔力播种机/爱心棒/收割机/花肥/营养液等）
		{Name: "染色药水", DType: 2, Level: 1, Price: 5000, Seed: 0, Ling: 0, Buds: 0, Less: 0, More: 0, Remark: "可以给花朵染色。"},
		{Name: "魔力播种机", DType: 2, Level: 1, Price: 20000, Seed: 0, Ling: 0, Buds: 0, Less: 0, More: 0, Remark: "一键播种所有空花盆。"},
		{Name: "魔力爱心棒", DType: 2, Level: 1, Price: 20000, Seed: 0, Ling: 0, Buds: 0, Less: 0, More: 0, Remark: "一键照料所有花朵。"},
		{Name: "魔力收割机", DType: 2, Level: 1, Price: 20000, Seed: 0, Ling: 0, Buds: 0, Less: 0, More: 0, Remark: "一键收获所有成熟花朵。"},
		{Name: "愿望果实", DType: 2, Level: 1, Price: 5000, Seed: 0, Ling: 0, Buds: 0, Less: 0, More: 0, Remark: "许下一个美好的愿望。"},
		{Name: "小魔法花肥", DType: 2, Level: 1, Price: 10000, Seed: 0, Ling: 0, Buds: 0, Less: 0, More: 0, Remark: "缩短花朵成长时间。"},
		{Name: "魔法营养液", DType: 2, Level: 1, Price: 50000, Seed: 0, Ling: 0, Buds: 0, Less: 0, More: 0, Remark: "大幅缩短花朵成长时间。"},
		{Name: "小魔力营养液", DType: 2, Level: 1, Price: 100000, Seed: 0, Ling: 0, Buds: 0, Less: 0, More: 0, Remark: "让花朵立即成熟。"},
	}
	for _, s := range seeds {
		var n int64
		db.Model(&model.GardenSeed{}).Where("name = ?", s.Name).Count(&n)
		if n == 0 {
			db.Create(&s)
		}
	}

	maps := []model.GardenMap{
		{SeedID: 1, Name: "金色向日葵", DType: 0},
		{SeedID: 1, Name: "七彩向日葵", DType: 1},
		{SeedID: 1, Name: "太阳神花", DType: 2},
		{SeedID: 2, Name: "红玫瑰", DType: 0},
		{SeedID: 2, Name: "蓝玫瑰", DType: 1},
		{SeedID: 2, Name: "黑玫瑰", DType: 2},
		{SeedID: 3, Name: "黄郁金香", DType: 0},
		{SeedID: 3, Name: "粉郁金香", DType: 1},
		{SeedID: 3, Name: "黑郁金香", DType: 2},
		{SeedID: 4, Name: "月光花", DType: 0},
		{SeedID: 4, Name: "星月花", DType: 1},
		{SeedID: 4, Name: "幻月花", DType: 2},
		{SeedID: 5, Name: "白百合", DType: 0},
		{SeedID: 5, Name: "金百合", DType: 1},
		{SeedID: 5, Name: "火百合", DType: 2},
		{SeedID: 6, Name: "粉牡丹", DType: 0},
		{SeedID: 6, Name: "绿牡丹", DType: 1},
		{SeedID: 6, Name: "黑牡丹", DType: 2},
		{SeedID: 7, Name: "蓝色妖姬", DType: 0},
		{SeedID: 7, Name: "冰蓝妖姬", DType: 1},
		{SeedID: 7, Name: "魅蓝妖姬", DType: 2},
		{SeedID: 8, Name: "粉樱花", DType: 0},
		{SeedID: 8, Name: "垂枝樱", DType: 1},
		{SeedID: 8, Name: "夜樱", DType: 2},
		{SeedID: 9, Name: "雪莲花", DType: 0},
		{SeedID: 9, Name: "金雪莲", DType: 1},
		{SeedID: 9, Name: "七彩雪莲", DType: 2},
		{SeedID: 10, Name: "银色菊花", DType: 0},
		{SeedID: 11, Name: "银野花", DType: 0},
		{SeedID: 12, Name: "端阳花", DType: 0},
		{SeedID: 13, Name: "银友谊花", DType: 0},
		{SeedID: 14, Name: "银色烈焰焚情", DType: 1},
		{SeedID: 15, Name: "金色烈焰焚情", DType: 2},
	}
	for _, m := range maps {
		var n int64
		db.Model(&model.GardenMap{}).Where("name = ?", m.Name).Count(&n)
		if n == 0 {
			db.Create(&m)
		}
	}

	mixes := []model.GardenMix{
		{SeedID: 10, Flower: "向日葵", Need: 5},
		{SeedID: 11, Flower: "向日葵", Need: 3},
		{SeedID: 11, Flower: "玫瑰花", Need: 2},
		{SeedID: 12, Flower: "玫瑰花", Need: 4},
		{SeedID: 12, Flower: "郁金香", Need: 2},
		{SeedID: 13, Flower: "向日葵", Need: 2},
		{SeedID: 13, Flower: "玫瑰花", Need: 2},
		{SeedID: 13, Flower: "郁金香", Need: 2},
		{SeedID: 14, Flower: "月光花", Need: 3},
		{SeedID: 15, Flower: "月光花", Need: 6},
	}
	for _, m := range mixes {
		var n int64
		db.Model(&model.GardenMix{}).Where("seed_id = ? AND flower = ?", m.SeedID, m.Flower).Count(&n)
		if n == 0 {
			db.Create(&m)
		}
	}

	// 精灵花册（10 只，按点亮图谱数解锁）
	elves := []model.GardenElf{
		{Name: "绿芽精灵", Desc: "花园的新生，点亮 3 个图谱后觉醒。", NeedMap: 3, Sort: 1, Img: "elf_1.png"},
		{Name: "露珠精灵", Desc: "清晨的第一滴露水，点亮 6 个图谱后觉醒。", NeedMap: 6, Sort: 2, Img: "elf_2.png"},
		{Name: "花粉精灵", Desc: "随风飞舞的花粉，点亮 9 个图谱后觉醒。", NeedMap: 9, Sort: 3, Img: "elf_3.png"},
		{Name: "花苞精灵", Desc: "含苞待放的期待，点亮 12 个图谱后觉醒。", NeedMap: 12, Sort: 4, Img: "elf_4.png"},
		{Name: "月光精灵", Desc: "月下的银色光辉，点亮 15 个图谱后觉醒。", NeedMap: 15, Sort: 5, Img: "elf_5.png"},
		{Name: "彩虹精灵", Desc: "七彩的花之桥，点亮 18 个图谱后觉醒。", NeedMap: 18, Sort: 6, Img: "elf_6.png"},
		{Name: "星光精灵", Desc: "夜空里的花语，点亮 21 个图谱后觉醒。", NeedMap: 21, Sort: 7, Img: "elf_7.png"},
		{Name: "晨露精灵", Desc: "晨曦中的晶莹，点亮 24 个图谱后觉醒。", NeedMap: 24, Sort: 8, Img: "elf_8.png"},
		{Name: "花语精灵", Desc: "读懂每一朵花的低语，点亮 27 个图谱后觉醒。", NeedMap: 27, Sort: 9, Img: "elf_9.png"},
		{Name: "花园精灵王", Desc: "花园世界的守护者，点亮 30 个图谱后降临。", NeedMap: 30, Sort: 10, Img: "elf_10.png"},
	}
	for _, e := range elves {
		var n int64
		db.Model(&model.GardenElf{}).Where("name = ?", e.Name).Count(&n)
		if n == 0 {
			db.Create(&e)
		} else {
			// 补全图片字段（老库已有行）
			db.Model(&model.GardenElf{}).Where("name = ?", e.Name).Update("img", e.Img)
		}
	}
}

// seedGardenSignRewards 魔法花园七日连签奖励配置（幂等：按 day 补齐，已有不覆盖）
// 连续第N天：花种 / G币 / 经验 / 元宝；第7天大奖后进入新一轮
func seedGardenSignRewards(db *gorm.DB) {
	rows := []model.GardenSignReward{
		{Day: 1, SeedN: 1, Coins: 500, Exp: 200, Ingots: 0},
		{Day: 2, SeedN: 2, Coins: 1000, Exp: 300, Ingots: 0},
		{Day: 3, SeedN: 3, Coins: 2000, Exp: 500, Ingots: 0},
		{Day: 4, SeedN: 3, Coins: 4000, Exp: 700, Ingots: 0},
		{Day: 5, SeedN: 4, Coins: 10000, Exp: 1000, Ingots: 0},
		{Day: 6, SeedN: 4, Coins: 20000, Exp: 1500, Ingots: 0},
		{Day: 7, SeedN: 5, Coins: 30000, Exp: 2000, Ingots: 5},
	}
	for _, r := range rows {
		var n int64
		db.Model(&model.GardenSignReward{}).Where("day = ?", r.Day).Count(&n)
		if n == 0 {
			db.Create(&r)
		}
	}
}

// seedFarmData 开心农场种子数据（幂等：按名称逐条补齐）
// 对齐诺哈三代 wap_farm_seed / wap_farm_muck / wap_farm_trap 数值体系
// 种子价 = price*5*cycle（商店购入价），aging/again 为分钟
func seedFarmData(db *gorm.DB) {
	seeds := []model.FarmSeed{
		{Name: "白萝卜", Cycle: 1, Aging: 15, Again: 0, Yield: 8, Price: 2, Point: 4, Level: 1},
		{Name: "胡萝卜", Cycle: 1, Aging: 25, Again: 0, Yield: 8, Price: 4, Point: 5, Level: 1},
		{Name: "玉米", Cycle: 2, Aging: 30, Again: 30, Yield: 9, Price: 6, Point: 6, Level: 2},
		{Name: "土豆", Cycle: 1, Aging: 45, Again: 0, Yield: 10, Price: 8, Point: 7, Level: 3},
		{Name: "番茄", Cycle: 3, Aging: 40, Again: 25, Yield: 10, Price: 10, Point: 8, Level: 4},
		{Name: "草莓", Cycle: 4, Aging: 45, Again: 30, Yield: 12, Price: 14, Point: 10, Level: 5},
		{Name: "西瓜", Cycle: 1, Aging: 90, Again: 0, Yield: 14, Price: 20, Point: 12, Level: 6},
		{Name: "南瓜", Cycle: 2, Aging: 80, Again: 60, Yield: 16, Price: 25, Point: 14, Level: 8},
		{Name: "樱桃", Cycle: 3, Aging: 90, Again: 60, Yield: 18, Price: 32, Point: 16, Level: 10},
		{Name: "榴莲", Cycle: 2, Aging: 120, Again: 90, Yield: 20, Price: 40, Point: 20, Level: 12},
		{Name: "人参", Cycle: 1, Aging: 240, Again: 0, Yield: 24, Price: 60, Point: 30, Level: 15},
	}
	for _, s := range seeds {
		var n int64
		db.Model(&model.FarmSeed{}).Where("name = ?", s.Name).Count(&n)
		if n == 0 {
			db.Create(&s)
		}
	}

	mucks := []model.FarmMuck{
		{Name: "小化肥", Speed: 10, Price: 100},
		{Name: "化肥", Speed: 30, Price: 250},
		{Name: "大化肥", Speed: 60, Price: 450},
		{Name: "神奇化肥", Speed: 120, Price: 800},
	}
	for _, m := range mucks {
		var n int64
		db.Model(&model.FarmMuck{}).Where("name = ?", m.Name).Count(&n)
		if n == 0 {
			db.Create(&m)
		}
	}

	traps := []model.FarmTrap{
		{Name: "烂陷阱", Rate: 30, Price: 100},
		{Name: "普通陷阱", Rate: 50, Price: 200},
		{Name: "高级陷阱", Rate: 70, Price: 350},
		{Name: "神奇陷阱", Rate: 90, Price: 600},
	}
	for _, t := range traps {
		var n int64
		db.Model(&model.FarmTrap{}).Where("name = ?", t.Name).Count(&n)
		if n == 0 {
			db.Create(&t)
		}
	}
}

// seedParkData 抢车位车辆数据（幂等：按名称逐条补齐）
// 对齐诺哈 wap_car_shop：dtype 1普通车 2高级车 3酷族车 4贵族车 5试驾车
// 盈利≈价格的0.5%/小时（停满12小时净收入约价5.4%，符合原版"停车一天回本一成"节奏）
func seedParkData(db *gorm.DB) {
	cars := []model.CarShop{
		// 普通车
		{Name: "奥拓", Icon: "gif", Price: 1200, Money: 6, DType: 1},
		{Name: "奇瑞QQ", Icon: "gif", Price: 2000, Money: 10, DType: 1},
		{Name: "夏利", Icon: "gif", Price: 2800, Money: 14, DType: 1},
		{Name: "五菱之光", Icon: "gif", Price: 3600, Money: 18, DType: 1},
		{Name: "长安之星", Icon: "gif", Price: 4500, Money: 22, DType: 1},
		{Name: "捷达", Icon: "gif", Price: 6000, Money: 30, DType: 1},
		{Name: "富康", Icon: "gif", Price: 6800, Money: 34, DType: 1},
		{Name: "桑塔纳", Icon: "gif", Price: 8000, Money: 40, DType: 1},
		{Name: "比亚迪F3", Icon: "gif", Price: 9000, Money: 45, DType: 1},
		{Name: "爱丽舍", Icon: "gif", Price: 10000, Money: 50, DType: 1},
		// 高级车
		{Name: "伊兰特", Icon: "gif", Price: 15000, Money: 75, DType: 2},
		{Name: "别克凯越", Icon: "gif", Price: 18000, Money: 90, DType: 2},
		{Name: "骐达", Icon: "gif", Price: 22000, Money: 110, DType: 2},
		{Name: "POLO", Icon: "gif", Price: 26000, Money: 130, DType: 2},
		{Name: "卡罗拉", Icon: "gif", Price: 30000, Money: 150, DType: 2},
		{Name: "思域", Icon: "gif", Price: 36000, Money: 180, DType: 2},
		{Name: "福克斯", Icon: "gif", Price: 40000, Money: 200, DType: 2},
		{Name: "速腾", Icon: "gif", Price: 45000, Money: 225, DType: 2},
		{Name: "轩逸", Icon: "gif", Price: 50000, Money: 250, DType: 2},
		{Name: "明锐", Icon: "gif", Price: 55000, Money: 275, DType: 2},
		// 酷族车
		{Name: "马自达3", Icon: "gif", Price: 68000, Money: 340, DType: 3},
		{Name: "甲壳虫", Icon: "gif", Price: 88000, Money: 440, DType: 3},
		{Name: "MINI COOPER", Icon: "gif", Price: 108000, Money: 540, DType: 3},
		{Name: "马自达6", Icon: "gif", Price: 128000, Money: 640, DType: 3},
		{Name: "天籁", Icon: "gif", Price: 158000, Money: 790, DType: 3},
		{Name: "锐志", Icon: "gif", Price: 188000, Money: 940, DType: 3},
		{Name: "君越", Icon: "gif", Price: 218000, Money: 1090, DType: 3},
		{Name: "凯美瑞", Icon: "gif", Price: 248000, Money: 1240, DType: 3},
		{Name: "雅阁", Icon: "gif", Price: 278000, Money: 1390, DType: 3},
		{Name: "帕萨特领驭", Icon: "gif", Price: 308000, Money: 1540, DType: 3},
		// 贵族车
		{Name: "奥迪A4", Icon: "gif", Price: 400000, Money: 2000, DType: 4},
		{Name: "宝马3系", Icon: "gif", Price: 500000, Money: 2500, DType: 4},
		{Name: "奔驰C级", Icon: "gif", Price: 600000, Money: 3000, DType: 4},
		{Name: "凯迪拉克CTS", Icon: "gif", Price: 700000, Money: 3500, DType: 4},
		{Name: "奥迪A6L", Icon: "gif", Price: 800000, Money: 4000, DType: 4},
		{Name: "宝马5系", Icon: "gif", Price: 1000000, Money: 5000, DType: 4},
		{Name: "奔驰E级", Icon: "gif", Price: 1200000, Money: 6000, DType: 4},
		{Name: "奥迪Q7", Icon: "gif", Price: 1600000, Money: 8000, DType: 4},
		{Name: "宝马7系", Icon: "gif", Price: 2000000, Money: 10000, DType: 4},
		{Name: "奔驰S级", Icon: "gif", Price: 2500000, Money: 12500, DType: 4},
		{Name: "保时捷卡宴", Icon: "gif", Price: 3000000, Money: 15000, DType: 4},
		{Name: "法拉利F430", Icon: "gif", Price: 5000000, Money: 25000, DType: 4},
		{Name: "兰博基尼", Icon: "gif", Price: 8000000, Money: 40000, DType: 4},
		{Name: "劳斯莱斯幻影", Icon: "gif", Price: 12000000, Money: 60000, DType: 4},
		// 试驾车（低门槛高盈利彩蛋）
		{Name: "试驾体验车", Icon: "gif", Price: 500, Money: 15, DType: 5},
		{Name: "试驾跑车", Icon: "gif", Price: 2000, Money: 80, DType: 5},
	}
	for _, s := range cars {
		var n int64
		db.Model(&model.CarShop{}).Where("name = ?", s.Name).Count(&n)
		if n == 0 {
			db.Create(&s)
		}
	}
}

// seedPlazaSections 广场板块开关（幂等，默认全显示）
func seedPlazaSections(db *gorm.DB) {
	var n int64
	db.Model(&model.PlazaSection{}).Count(&n)
	if n > 0 {
		return
	}
	for i, p := range model.PlazaSectionPresets {
		db.Create(&model.PlazaSection{Key: p.Key, Name: p.Name, Enabled: 1, Sort: i})
	}
}

// seedNoblePlans 特权开通方案（幂等）
func seedNoblePlans(db *gorm.DB) {
	var n int64
	db.Model(&model.NoblePlan{}).Count(&n)
	if n == 0 {
		for _, p := range model.NoblePlanPresets {
			db.Create(&p)
		}
		return
	}
	// 老库方案补齐 speed/limit/stock（幂等）
	for _, p := range model.NoblePlanPresets {
		db.Model(&model.NoblePlan{}).Where("name = ?", p.Name).Updates(map[string]interface{}{
			"speed": p.Speed, "limit": p.Limit, "stock": p.Stock,
		})
	}
}

// seedNobleLevels 贵宾等级配置（复刻诺哈 wap_vip_config，幂等）
func seedNobleLevels(db *gorm.DB) {
	var n int64
	db.Model(&model.NobleLevel{}).Count(&n)
	if n == 0 {
		for _, l := range model.NobleLevelPresets {
			db.Create(&l)
		}
		return
	}
	// 老库补齐 1-8 级与图标（幂等）
	for _, l := range model.NobleLevelPresets {
		var exist int64
		db.Model(&model.NobleLevel{}).Where("id = ?", l.ID).Count(&exist)
		if exist == 0 {
			db.Create(&l)
		} else {
			db.Model(&model.NobleLevel{}).Where("id = ?", l.ID).Updates(map[string]interface{}{
				"icon_blue": l.IconBlue, "icon_qq": l.IconQQ,
			})
		}
	}
}

// seedGoods 道具商城示例（幂等：按名称补种，老库已有商品也会补齐鲜花等新品）
func seedGoods(db *gorm.DB) {
	for _, g := range model.GoodPresets {
		var exist int64
		db.Model(&model.Good{}).Where("name = ?", g.Name).Count(&exist)
		if exist == 0 {
			db.Create(&g)
		} else {
			// 老库商品补齐友友券价（幂等）；库存仅在为0时回填，不覆盖运营调整
			db.Model(&model.Good{}).Where("name = ?", g.Name).Update("youquan_price", g.YouQuanPrice)
			db.Model(&model.Good{}).Where("name = ? AND stock = 0", g.Name).Update("stock", g.Stock)
		}
	}
}

// seedMoneyShop 货币商店种子（复刻诺哈 wap_money_shop：花一种货币买另一种货币礼包，幂等按名称补种）
func seedMoneyShop(db *gorm.DB) {
	end := time.Date(2027, 12, 31, 23, 59, 59, 0, time.Local)
	presets := []model.MoneyShop{
		{Name: "1000G币礼包", MType: "coins", Money: 1000, PType: "yuanbao", Price: 10, Stock: 100, Status: 1},
		{Name: "10000G币豪华礼包", MType: "coins", Money: 10000, PType: "yuanbao", Price: 88, Stock: 50, Status: 1},
		{Name: "10元宝特惠包", MType: "yuanbao", Money: 10, PType: "coins", Price: 10000, Stock: 200, Status: 1},
		{Name: "100元宝礼包", MType: "yuanbao", Money: 100, PType: "coins", Price: 100000, Stock: 100, Status: 1},
		{Name: "5张友友券礼包", MType: "youquan", Money: 5, PType: "yuanbao", Price: 2, Stock: 80, Status: 1},
		{Name: "1金钻礼包", MType: "jinzuan", Money: 1, PType: "yuanbao", Price: 50, Stock: 30, Status: 1},
	}
	for _, s := range presets {
		var exist int64
		db.Model(&model.MoneyShop{}).Where("name = ?", s.Name).Count(&exist)
		if exist == 0 {
			s.AddTime = time.Now()
			s.EndTime = end
			db.Create(&s)
		}
	}
}

func randInt(n int) int { return int(time.Now().UnixNano())%n + 1 }

// syncDir 扫描 static/picture 与 static/image，把未登记的图片登记为「其他」
func syncDir(db *gorm.DB, staticDir string) int {
	added := 0
	for _, sub := range []string{"picture", "image"} {
		entries, err := os.ReadDir(filepath.Join(staticDir, sub))
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
			db.Model(&model.Resource{}).Where("file = ?", file).Count(&n)
			if n == 0 {
				short := strings.TrimSuffix(name, ext)
				if len(short) > 30 {
					short = short[:30]
				}
				db.Create(&model.Resource{File: file, Category: "other", Name: short, Status: 1})
				added++
			}
		}
	}
	return added
}

// seedMigrate2026 迁移演示站其余友友与帖子，全部时间落在 2026 年
// （逐条幂等：用户按号码去重，帖子按「标题+板块」去重，可安全重复执行）
func seedFriendMigrate(db *gorm.DB) {
	// 老表补齐新列（幂等，对齐诺哈 wap_friend：remark/group_id/degree）
	m := db.Migrator()
	for _, col := range []string{"remark", "group_id", "degree"} {
		if !m.HasColumn("friendships", col) {
			db.Exec("ALTER TABLE friendships ADD COLUMN " + col + " " + map[string]string{
				"remark":   "varchar(30) DEFAULT ''",
				"group_id": "bigint DEFAULT 0",
				"degree":   "int DEFAULT 0",
			}[col])
		}
	}
	for _, col := range []string{"sort", "amount"} {
		if !m.HasColumn("friend_groups", col) {
			db.Exec("ALTER TABLE friend_groups ADD COLUMN " + col + " int DEFAULT 0")
		}
	}

	// 幂等：仅当还未迁移（不存在 friend_applies/friend_blacks 数据）且老库有 status 列时执行
	if !m.HasColumn("friendships", "status") {
		return
	}
	var applyCount int64
	db.Model(&model.FriendApply{}).Count(&applyCount)
	var blackCount int64
	db.Model(&model.FriendBlack{}).Count(&blackCount)
	if applyCount > 0 || blackCount > 0 {
		// 已迁移过，直接同步一次 group 数量后返回
		syncGroupAmounts(db)
		return
	}

	// ① 未确认的申请（老 status=0：user_id 申请 → friend_id）转 friend_applies
	type oldRow struct {
		UserID   uint
		FriendID uint
	}
	var pend []oldRow
	db.Raw("SELECT user_id, friend_id FROM friendships WHERE status = 0 AND user_id != friend_id").Scan(&pend)
	for _, p := range pend {
		var n int64
		db.Model(&model.FriendApply{}).Where("user_id = ? AND friend_id = ?", p.FriendID, p.UserID).Count(&n)
		if n == 0 {
			db.Create(&model.FriendApply{UserID: p.FriendID, FriendID: p.UserID})
		}
	}

	// ② 已确认的好友（老 status=1，无向）转双向 status=1 记录，并合并分组/备注
	var acc []oldRow
	db.Raw("SELECT user_id, friend_id FROM friendships WHERE status = 1 AND user_id != friend_id").Scan(&acc)
	// 收集分组信息（老 friend_group_items：group_id -> user_id, friend_id）
	groupOf := map[string]uint{} // "uid:oid" -> group_id
	type gi struct {
		GroupID  uint
		UserID   uint
		FriendID uint
	}
	var gis []gi
	db.Table("friend_group_items").Scan(&gis)
	for _, g := range gis {
		groupOf[fmt.Sprintf("%d:%d", g.UserID, g.FriendID)] = g.GroupID
	}

	// 清空旧无向记录，改造成纯方向模型
	db.Exec("DELETE FROM friendships")
	created := 0
	for _, a := range acc {
		// 建立 我→TA 与 TA→我 两条
		for _, dir := range []oldRow{{a.UserID, a.FriendID}, {a.FriendID, a.UserID}} {
			var n int64
			db.Model(&model.Friendship{}).Where("user_id = ? AND friend_id = ?", dir.UserID, dir.FriendID).Count(&n)
			if n == 0 {
				db.Create(&model.Friendship{
					UserID: dir.UserID, FriendID: dir.FriendID, Status: 1,
					GroupID: groupOf[fmt.Sprintf("%d:%d", dir.UserID, dir.FriendID)],
				})
				created++
			}
		}
	}

	// ③ 统一 status=1（纯方向模型下 status 仅记录建立状态）
	db.Exec("UPDATE friendships SET status = 1")
	// ④ 追加唯一约束（数据清洗后确保同一 user→friend 唯一）
	if !m.HasIndex("friendships", "uk_uf") {
		db.Exec("ALTER TABLE friendships ADD UNIQUE INDEX uk_uf (user_id, friend_id)")
	}

	syncGroupAmounts(db)
	if created > 0 {
		fmt.Printf("好友模块迁移：建立 %d 条双向好友关系\n", created)
	}
}

// syncGroupAmounts 同步各分组的好友数（诺哈 wap_friend_group.amount）
func syncGroupAmounts(db *gorm.DB) {
	var groups []model.FriendGroup
	db.Find(&groups)
	for _, g := range groups {
		var n int64
		db.Model(&model.Friendship{}).Where("user_id = ? AND group_id = ? AND status = 1", g.UserID, g.ID).Count(&n)
		db.Model(&model.FriendGroup{}).Where("id = ?", g.ID).Update("amount", n)
	}
	// 未分组（group_id=0）好友数不额外记录
}

func seedMigrate2026(db *gorm.DB) {
	d := func(month, day, h, m int) time.Time {
		return time.Date(2026, time.Month(month), day, h, m, 0, 0, time.Local)
	}

	type nu struct {
		id      uint
		nick    string
		color   string
		exp     int
		created time.Time
		badges  []string
	}
	news := []nu{
		{35805083, "李春风", "#ff0000", 44000, d(1, 20, 9, 0), []string{"3.gif", "501.gif"}},
		{35805768, "尊上", "#6495ED", 53300, d(1, 28, 14, 30), []string{"903.gif"}},
		{35806205, "Ruby", "#ff0000", 15600, d(3, 5, 20, 0), nil},
		{35804196, "苍笙踏歌", "#ff0000", 20100, d(2, 8, 11, 0), nil},
		{35805532, "懒织红笺", "#ff0000", 22400, d(2, 16, 15, 0), nil},
		{35803370, "刘乐乐ヾ", "#FF8C00", 30700, d(1, 10, 10, 0), nil},
		{35799015, "麦", "#BC8F8F", 10900, d(4, 2, 16, 0), nil},
		{35792031, "宋", "#808080", 12600, d(3, 22, 19, 0), nil},
		{35806088, "秋水未央", "#ff0000", 38200, d(2, 1, 10, 0), []string{"804.gif"}},
		{35806114, "   ╰┈→cc", "#ff0000", 41500, d(1, 25, 13, 0), []string{"704.gif"}},
		{35806199, "老王", "#4169E1", 30500, d(2, 14, 21, 0), []string{"803.gif"}},
		{35806066, "断念", "#ff0000", 10800, d(4, 20, 22, 0), nil},
		{35806077, "轻轻淡写", "#ff0000", 21500, d(3, 18, 20, 0), nil},
		{35806260, "筱轩么么哒", "#ff0000", 6800, d(5, 12, 18, 0), nil},
		{35806345, "胡", "#004299", 0, d(8, 29, 10, 0), nil},
	}
	uid := map[string]uint{}
	for _, u := range news {
		uid[u.nick] = u.id
		var cnt int64
		db.Model(&model.User{}).Where("username = ?", fmt.Sprint(u.id)).Count(&cnt)
		if cnt > 0 {
			continue
		}
		hash, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
		mk := model.User{
			ID: u.id, Username: fmt.Sprint(u.id), Nickname: u.nick,
			Password: string(hash), Gender: 1, Color: u.color,
			Exp: u.exp, Coins: 200, Achieve: u.exp / 100,
			CreatedAt: u.created, UpdatedAt: u.created,
		}
		mk.Level = model.CalcLevel(u.exp)
		if err := db.Create(&mk).Error; err != nil {
			fmt.Println("migrate user:", u.nick, err)
			continue
		}
		if len(u.badges) > 0 {
			var bs []model.Badge
			db.Where("icon IN ?", u.badges).Find(&bs)
			if len(bs) > 0 {
				db.Model(&mk).Association("Badges").Append(&bs)
			}
		}
		var member model.Role
		db.Where("code = ?", "member").First(&member)
		db.Model(&mk).Association("Roles").Append(&member)
	}

	// 老居民的注册时间也归到 2026 年
	regDates := map[uint]time.Time{
		10000:    d(1, 1, 8, 0),
		10001:    d(1, 15, 9, 0),
		10002:    d(2, 3, 10, 0),
		10003:    d(2, 20, 11, 0),
		10004:    d(3, 15, 14, 0),
		10005:    d(4, 1, 15, 0),
		10006:    d(8, 10, 9, 0),
		10007:    d(8, 20, 9, 0),
		35797804: d(3, 10, 12, 0),
	}
	for idd, t := range regDates {
		db.Model(&model.User{}).Where("id = ?", idd).Updates(map[string]interface{}{"created_at": t, "updated_at": t})
	}

	// 老居民也进署名映射（云起/站长小Q/瞿詺南等是不少帖子和回复的作者）
	for _, name := range []string{"云起", "站长小Q", "安珞", "咏荷", "闲云野鹤", "蓝天", "　　瞿詺南　"} {
		var u model.User
		if err := db.Where("nickname = ?", name).First(&u).Error; err == nil {
			uid[name] = u.ID
		}
	}

	// 板块索引
	var boards []model.Board
	db.Find(&boards)
	bid := func(name string) uint {
		for _, b := range boards {
			if b.Name == name {
				return b.ID
			}
		}
		return 0
	}

	type rp struct {
		by, content string
		at          time.Time
	}
	type th struct {
		board, title, by, content string
		views                     int
		created                   time.Time
		top, fine                 int
		replies                   []rp
	}
	list := []th{
		{board: "新人求助", title: "【小破网】社区论坛公约", by: "李春风", fine: 1, views: 79, created: d(6, 1, 9, 0),
			content: "【小破网】社区论坛公约\n\n第一条 为加强社区网络平台的建设、管理及维护，营造健康和谐的交流环境，特制定本公约。\n第二条 友友发帖回帖应当遵守法律法规，尊重他人，文明用语。\n第三条 禁止发布外网链接、广告及任何形式的商业推广。\n第四条 各板块版主应当及时处理违规内容，公坛协管员负责巡查督导。\n第五条 本公约自发布之日起施行。\n\n　　　　　　　　家园社区 公坛管理组"},
		{board: "新人求助", title: "严厉打击宣传外网公告", by: "李春风", fine: 1, views: 527, created: d(7, 1, 8, 30),
			content: "【公告】严厉打击宣传外网行为\n\n近期发现有部分账号在社区内宣传外部网站、发布拉人广告，严重扰乱社区秩序。\n\n现公告如下：\n一、凡发布外网宣传内容的帖子一律删除，账号视情节封禁3-30天；\n二、屡教不改者永久封号并公示；\n三、欢迎大家向客服中心举报，举报属实奖励金币。\n\n家园是我们共同的家，请大家一起守护！\n\n　　　　　　　　家园社区 客服团",
			replies: []rp{
				{by: "云起", content: "严厉支持！共同维护家园环境。", at: d(7, 1, 10, 0)},
				{by: "苍笙踏歌", content: "支持！见到一个举报一个。", at: d(7, 2, 9, 0)},
			}},
		{board: "新人求助", title: "【UBB代码演示】ubb功能调用代码", by: "秋水未央", fine: 1, views: 8467, created: d(5, 20, 14, 0),
			content: "【UBB代码演示】\n\nUBB 是家园发帖常用的排版代码，掌握以下几条，帖子立刻好看十倍：\n\n[b]加粗文字[/b] —— 加粗\n[i]倾斜文字[/i] —— 倾斜\n[color=red]红字[/color] —— 变色\n[url=http://家园社区]链接[/url] —— 超链接\n\n排版三件套：分割线、颜色代码、居中标签，配合使用效果最佳。\n大家回帖练手，不懂就问！",
			replies: []rp{
				{by: "轻轻淡写", content: "收藏了，排版果然重要。", at: d(5, 21, 9, 0)},
				{by: "Ruby", content: "好用！帖子瞬间变好看了", at: d(5, 21, 20, 0)},
				{by: "刘乐乐ヾ", content: "小白福音，感谢未央大佬。", at: d(5, 22, 12, 0)},
			}},
		{board: "新人求助", title: "密码忘记了", by: "筱轩么么哒", views: 67, created: d(8, 20, 21, 0),
			content: "呜呜呜，密码忘记了怎么办？号码还在的，求管理员帮忙找回！",
			replies: []rp{
				{by: "站长小Q", content: "请到客服中心发帖申诉，管理员核实后帮你重置密码。", at: d(8, 21, 9, 0)},
			}},
		{board: "休闲灌水", title: "【休闲灌水】论坛版规", by: "   ╰┈→cc", top: 1, fine: 1, views: 257, created: d(6, 15, 10, 0),
			content: "【休闲灌水】论坛版规\n\n一、本版为休闲灌水专区，日常聊天、灌水打卡均可；\n二、禁止发布广告、外链及人身攻击内容；\n三、恶意刷屏者版主有权删帖并禁言；\n四、精华帖标准：有内容、有温度、有回复量；\n五、本版规最终解释权归公坛管理组。\n\n祝大家灌水愉快！",
			replies: []rp{
				{by: "麦", content: "版规收到，每天来打卡！", at: d(6, 16, 8, 30)},
			}},
		{board: "休闲灌水", title: "春趣", by: "云起", views: 6, created: d(8, 29, 11, 0),
			content: "春风拂柳绿如烟，\n燕归来时花满天。\n童子放鸢田埂上，\n一竿撑起半边天。",
			replies: []rp{
				{by: "咏荷", content: "好诗！有田园气息。", at: d(8, 29, 11, 30)},
				{by: "安珞", content: "来晚了，赏了~", at: d(8, 29, 12, 0)},
			}},
		{board: "休闲灌水", title: "咏荷", by: "云起", views: 2, created: d(8, 29, 11, 20),
			content: "接天莲叶无穷碧，\n映日荷花别样红。\n咏而归舟惊鹭起，\n一池清香伴晚风。"},
		{board: "休闲灌水", title: "怎知", by: "云起", views: 3, created: d(8, 29, 11, 40),
			content: "怎知相思苦，\n只缘未相逢。\n若得君心意，\n不负此生情。"},
		{board: "休闲灌水", title: "西风", by: "云起", views: 4, created: d(8, 29, 12, 0),
			content: "西风昨夜过园林，\n吹落黄花满地金。\n游子他乡逢秋雨，\n一封家书抵万金。"},
		{board: "休闲灌水", title: "桃花", by: "云起", views: 3, created: d(8, 29, 12, 20),
			content: "桃花坞里桃花庵，\n桃花庵下桃花仙。\n桃花仙人种桃树，\n又摘桃花换酒钱。"},
		{board: "情感天地", title: "终是庄周梦了蝶，你是恩赐也是劫。", by: "断念", views: 89, created: d(8, 25, 22, 0),
			content: "庄生晓梦迷蝴蝶。\n有人说，庄周梦的是蝶，而我梦的是你。\n\n你是恩赐，也是劫。\n恩赐是遇见，劫是离别。\n\n来情感天地，说说你的故事。愿天下有情人都不留遗憾。",
			replies: []rp{
				{by: "尊上", content: "梦醒时分，各自安好。", at: d(8, 26, 9, 0)},
				{by: "懒织红笺", content: "字字扎心，抱一个。", at: d(8, 26, 20, 0)},
			}},
		{board: "情感天地", title: "执子之手 与子成双", by: "咏荷", views: 66, created: d(8, 22, 19, 0),
			content: "执子之手，与子成双。\n征婚台首帖，希望在这里遇到那个愿意陪我赏花看月亮的人。\n要求不高：上线勤、说话暖、会陪聊。",
			replies: []rp{
				{by: "安珞", content: "祝百年好合！家园第一对？", at: d(8, 23, 10, 0)},
			}},
		{board: "情感天地", title: "有没有", by: "Ruby", views: 30, created: d(8, 27, 23, 0),
			content: "有没有那么一个人，你一上线就想看他在不在线。\n有没有？"},
		{board: "家族风云", title: "4月份封控19天总结", by: "尊上", views: 88, created: d(8, 5, 20, 0),
			content: "【与世无争】家族4月份封控19天总结\n\n19天里，家族成员线上陪伴不断：\n- 每晚八点聊天室准时集合，累计发言3000+条；\n- 家族互赠金币486次；\n- 新增成员7人。\n\n隔离病毒不隔离爱，与世无争，与家有约。",
			replies: []rp{
				{by: "宋", content: "家族有你真好。", at: d(8, 6, 9, 0)},
				{by: "老王", content: "那19天是家族最热闹的时候。", at: d(8, 6, 12, 0)},
			}},
		{board: "家族风云", title: "恭喜我大咪", by: "老王", views: 40, created: d(8, 18, 21, 0),
			content: "恭喜家族的大咪当选本周风云人物，撒花！\n大咪上周发帖30篇、回帖200+，勤劳榜第一，实至名归！",
			replies: []rp{
				{by: "刘乐乐ヾ", content: "恭喜恭喜！向大咪学习。", at: d(8, 19, 8, 0)},
				{by: "麦", content: "撒花✿", at: d(8, 19, 10, 0)},
			}},
		{board: "家族申请", title: "我要申家", by: "断念", views: 70, created: d(8, 8, 15, 0),
			content: "ID:35806066+断念+申请创建家族【断念阁】\n\n家族宗旨：聚是一团火，散是满天星。\n现有成员：15人，日活跃8+。\n望管理员批准！",
			replies: []rp{
				{by: "　　瞿詺南　", content: "材料收到，公坛已受理，等待审核。", at: d(8, 9, 9, 0)},
			}},
		{board: "数码动漫", title: "10000毫安电量+加密芯片+3D人脸+128GB", by: "轻轻淡写", views: 60, created: d(8, 26, 12, 0),
			content: "如题，这配置放在当年想都不敢想。\n现在新机内卷到这个程度了，大家换机是选大电池还是选影像？",
			replies: []rp{
				{by: "苍笙踏歌", content: "我选大电池，一天三充受不了。", at: d(8, 26, 14, 0)},
			}},
		{board: "数码动漫", title: "荣耀再次爆发，6000mAh“神机”备货，天玑9000加持", by: "轻轻淡写", views: 54, created: d(8, 24, 10, 0),
			content: "如题，官方公布了首批适配名单，快看看有没有你的机型。",
			replies: []rp{
				{by: "刘乐乐ヾ", content: "等一个真香价。", at: d(8, 24, 11, 0)},
			}},
		{board: "公坛事务", title: "【小破网】社区统筹管理纲要", by: "　　瞿詺南　", fine: 1, views: 308, created: d(7, 15, 16, 0),
			content: "第一章，总则\n\n第一条 为加强社区网络平台的建设、管理及维护（以下简称社区），服务社区工作，服务用户，提升社区知名度、美誉度，特制定本管理规定；\n\n第二条 网络平台建设、运营和管理必须坚持立德树人，以人为本，弘扬社会主义核心价值观，传递网络正能量；\n\n第三条 社区目前所建设和管理的网络平台包括：\n\n1.社区网站，网址：https://家园社区\n\n2.手机QQ群，群号242973387\n\n第四条 社区发布的信息内容须遵循真实性、准确性和及时性相统一的原则；\n\n第五条 信息内容和展现方式须全面、完整、合时宜。\n\n第二章 管理机构及职责\n\n第六条 社区设立  七大管理区域，即〔公坛〕〔同城〕〔客服团〕〔家服团〕〔爱游团〕〔社区传媒〕〔监察局〕\n\n第七条，各区域管理 对所负责的区域进行全面监督及管理。\n\n第八条 专职监察员负责社区的信息收集与汇总、日常管理，负责社区案件审核，处理通告发布以及其他信息材料的管理、录入与发布；\n\n第九条 处理结果发布后的反馈（含反响评论、错误指正、批评建议等）须及时上报，妥善处理。\n\n第三章 规范及要求\n\n第十条 信息发布严格执行\"谁发布、谁负责；谁批准、谁负责\"的原则。如发布不良、有害或反动等内容信息，将对经办人和负责人依据规定追究相关责任；\n\n第十六条 其他事宜参考【小破网】社区论坛公约\n\n第四章，社区的应聘要求及流程\n\n随着社区人员越来越多，管理人员的更替和加入也越来越频繁。社区必须做出一个统一的管理流程，终结各自一摊的局面。\n\n为此做出以下规定，\n\n应聘贴必须在招聘活动进行时有效，拒绝凭空应聘和空降职务。\n\n应聘要求统一为〔一线管理〕\n\n1.家园等级不低于3级\n\n2.论坛等级不低于一级\n\n3.有足够时间上线活跃\n\n〔一级管理〕\n\n不接受直接申请，\n\n应聘帖子统一为\n\n标题：昵称+ID+应聘区域职务\n\n帖子内容为\n\n①.家园ID:\n\n②.家园昵称:\n\n③.家园等级:\n\n④.论坛等级:\n\n⑤.是否已经绑定手机:\n\n⑥.每日在线时长:\n\n⑦.对社区的了解:\n\n⑧.对应聘职位的了解:\n\n社区原则上是不支持社区管理兼职，但是各区域可根据实际情况调整管理是否兼职，最多两个一线管理职务，需要分管仲裁报备到仲裁团，(家族职务除外)\n\n第五章， 管理员的考核和审核\n\n报名成功后：需要统一安排培训，是否被录取，以录取公告为准。\n\n统一为不带权考核时间最多为7天，\n\n带权实习最长时间为一个月，\n\n第六章，管理员实习\n\n社区支持新人培训后上岗，但是各区域可根据实际情况选择是否培训。需要分管仲裁报备到仲裁团。\n\n实习内容包括\n\n1.实习目的\n\n2.实习时间\n\n3.实习单位\n\n4.实习主要内容\n\n5.实习心得\n\n撰写报告后提交各区域内版\n\n管理的晋升参考社区管理统筹大纲\n\n【监察局】\n\n第七章，监察员转正后的工作内容详责\n\n第一条 ，工作内容可参考【小破网】社区论坛公约进行，\n\n二条， 关于签名\n\n1、 不得出现宣扬反动、封建迷信、淫秽、色情、暴力、凶杀、恐怖、教唆犯罪等不符合国家法律规定的以及任何包含种族、性别、宗教歧视性和猥亵性的信息内容；\n\n2、 不得出现有侮辱性言语、挑衅、辱骂其他人以及不健康内容；\n\n3、 不得出现国家明令禁止广告的内容或链接；\n\n4、 不得出现其它违反《小破网社区各区域管理规定》的内容。\n\n　　　　　　　　家园社区 公坛管理组",
			replies: []rp{
				{by: "云起", content: "纲要已读，共同遵守！", at: d(7, 15, 17, 0)},
				{by: "李春风", content: "已转发客服团学习。", at: d(7, 16, 9, 0)},
			}},
		{board: "公坛事务", title: "[重要]家园社区已完成所有法定备案", by: "站长小Q", fine: 1, views: 4384, created: d(6, 10, 10, 0),
			content: "家园社区（家园社区）已取得工信部备案：渝ICP备17001534号-2。\n\n备案信息可在工信部官网查询。家园合法合规运营，请大家放心游玩，也别忘了身边的老朋友。\n\n　　　　　　　　家园社区 站长办",
			replies: []rp{
				{by: "云起", content: "大事务！恭喜家园。", at: d(6, 10, 12, 0)},
				{by: "尊上", content: "普天同庆，家园长长久久。", at: d(6, 10, 14, 0)},
			}},
	}
	for _, x := range list {
		authorID := uid[x.by]
		if authorID == 0 || bid(x.board) == 0 {
			continue
		}
		var cnt int64
		db.Model(&model.Thread{}).Where("title = ? AND board_id = ?", x.title, bid(x.board)).Count(&cnt)
		if cnt > 0 {
			continue
		}
		t := model.Thread{
			BoardID: bid(x.board), UserID: authorID,
			Title: x.title, Content: x.content,
			IsTop: x.top, IsFine: x.fine, ViewCount: x.views,
			CreatedAt: x.created, UpdatedAt: x.created,
		}
		db.Create(&t)
		var last time.Time
		floor := 2
		for _, r := range x.replies {
			rid := uid[r.by]
			if rid == 0 {
				continue
			}
			db.Create(&model.Reply{ThreadID: t.ID, UserID: rid, Content: r.content, Floor: floor, CreatedAt: r.at})
			floor++
			last = r.at
			t.ReplyCount++
		}
		if !last.IsZero() {
			t.LastReplyAt = &last
		}
		db.Model(&t).Updates(map[string]interface{}{"reply_count": t.ReplyCount, "last_reply_at": t.LastReplyAt})
		db.Model(&model.Board{}).Where("id = ?", t.BoardID).UpdateColumn("thread_count", gorm.Expr("thread_count + 1"))
	}
}

func seedRBAC(db *gorm.DB) {
	perms := []model.Permission{
		{Name: "后台访问", Code: "admin:access", Remark: "进入管理后台"},
		{Name: "用户管理", Code: "user:manage", Remark: "封禁/解封/重置密码/分配角色"},
		{Name: "板块管理", Code: "board:manage", Remark: "板块增删改"},
		{Name: "帖子管理", Code: "thread:manage", Remark: "置顶/精华/删帖删回复"},
		{Name: "公告管理", Code: "announcement:manage", Remark: "公告广播增删改"},
		{Name: "角色权限管理", Code: "role:manage", Remark: "角色与权限分配"},
		{Name: "马甲管理", Code: "badge:manage", Remark: "勋章/马甲增删与授予"},
		{Name: "游戏管理", Code: "game:manage", Remark: "游戏大厅增删改"},
	}
	for _, p := range perms {
		db.Where("code = ?", p.Code).FirstOrCreate(&p)
	}

	roles := []model.Role{
		{Name: "超级管理员", Code: "super_admin", Remark: "拥有全部权限"},
		{Name: "管理员", Code: "admin", Remark: "社区日常管理"},
		{Name: "版主", Code: "moderator", Remark: "管理帖子"},
		{Name: "普通会员", Code: "member", Remark: "注册用户默认角色"},
	}
	for _, r := range roles {
		db.Where("code = ?", r.Code).FirstOrCreate(&r)
	}

	// 全部权限 -> 超级管理员；其余按名赋予
	var allPerms []model.Permission
	db.Find(&allPerms)
	super := model.Role{}
	db.Where("code = ?", "super_admin").First(&super)
	db.Model(&super).Association("Permissions").Replace(&allPerms)

	grant := func(roleCode string, codes ...string) {
		var role model.Role
		db.Where("code = ?", roleCode).First(&role)
		var ps []model.Permission
		db.Where("code IN ?", codes).Find(&ps)
		db.Model(&role).Association("Permissions").Replace(&ps)
	}
	grant("admin", "admin:access", "user:manage", "board:manage", "thread:manage", "announcement:manage", "badge:manage", "game:manage")
	grant("moderator", "admin:access", "thread:manage")
	grant("member")
}

func seedBoards(db *gorm.DB) {
	var count int64
	db.Model(&model.Board{}).Count(&count)
	if count > 0 {
		return
	}
	type sub struct {
		parent string
		name   string
		desc   string
	}
	channels := []model.Board{
		{Name: "公共论坛", Description: "最大的公共讨论区，畅所欲言", Sort: 1},
		{Name: "家族大厅", Description: "家族组建、申请、风云榜", Sort: 2},
		{Name: "同城客栈", Description: "累了吗？来同城客栈透个气吧", Sort: 3},
		{Name: "社区服务", Description: "客服、建议、公示", Sort: 4},
	}
	for i := range channels {
		db.Create(&channels[i])
	}
	subs := []sub{
		{"公共论坛", "休闲灌水", "没事灌灌水，聊聊日常"},
		{"公共论坛", "时尚美眉", "美眉们的时尚领地"},
		{"公共论坛", "新人求助", "新手报到、不懂就问"},
		{"公共论坛", "情感天地", "缘分、心情日记、鹊桥相会"},
		{"公共论坛", "数码动漫", "玩转手机、游戏狂潮、动漫天地"},
		{"家族大厅", "家族申请", "申请建立你的家族"},
		{"家族大厅", "家族风云", "家族动态与风云榜"},
		{"同城客栈", "福建", "福建的老乡看过来"},
		{"同城客栈", "北京", "北京的老乡看过来"},
		{"同城客栈", "广东", "广东的老乡看过来"},
		{"同城客栈", "共建同城", "你的城市还没有？来这里申请"},
		{"社区服务", "客服中心", "投诉、找回资料、社区事务"},
		{"社区服务", "意见建议", "为家园建设献言献策"},
		{"社区服务", "社区公示", "封禁公示、人事任免"},
	}
	nameID := map[string]uint{}
	db.Model(&model.Board{}).Where("parent_id = 0").Find(&channels)
	for _, ch := range channels {
		nameID[ch.Name] = ch.ID
	}
	for _, s := range subs {
		db.Create(&model.Board{ParentID: nameID[s.parent], Name: s.name, Description: s.desc})
	}
}

// seedBoardCategories 分区下分类分组（诺哈 wap_bbs_category），幂等
func seedBoardCategories(db *gorm.DB) {
	var count int64
	db.Model(&model.BoardCategory{}).Count(&count)
	if count > 0 {
		return
	}
	channelID := func(name string) uint {
		var b model.Board
		db.Where("name = ? AND parent_id = 0", name).First(&b)
		return b.ID
	}
	gs, fs, zc := channelID("公共论坛"), channelID("家族大厅"), channelID("同城客栈")
	if gs > 0 {
		db.Create(&[]model.BoardCategory{
			{ParentID: gs, Name: "灌水闲聊", Sort: 1},
			{ParentID: gs, Name: "兴趣圈子", Sort: 2},
			{ParentID: gs, Name: "互助问答", Sort: 3},
		})
	}
	if zc > 0 {
		db.Create(&[]model.BoardCategory{
			{ParentID: zc, Name: "沿海城市", Sort: 1},
			{ParentID: zc, Name: "内陆地区", Sort: 2},
		})
	}
	if fs > 0 {
		db.Create(&model.BoardCategory{ParentID: fs, Name: "家族专区", Sort: 1})
	}
}

// seedTongcheng 同城客栈：全国 32 省份 + 各省城市（带区号），参考诺哈三代
// 同城模块 wap_bbs(pid=9) 结构与 city.txt 区号表（幂等：已存在的省份/城市按名称跳过）
func seedTongcheng(db *gorm.DB) {
	var ch model.Board
	db.Where("parent_id = 0 AND name = ?", "同城客栈").First(&ch)
	if ch.ID == 0 {
		return
	}
	type city struct {
		name string
		code string
	}
	provinces := []struct {
		name   string
		cities []city
	}{
		{"安徽", []city{{"合肥", "0551"}, {"芜湖", "0552"}, {"蚌埠", "0553"}, {"淮南", "0554"}, {"马鞍山", "0555"}, {"安庆", "0556"}}},
		{"北京", []city{{"北京", "010"}}},
		{"福建", []city{{"福州", "0591"}, {"厦门", "0592"}, {"宁德", "0593"}, {"莆田", "0594"}, {"泉州", "0595"}, {"漳州", "0596"}, {"龙岩", "0597"}, {"三明", "0598"}, {"南平", "0599"}}},
		{"甘肃", []city{{"兰州", "0931"}, {"金昌", "0935"}, {"白银", "0943"}, {"天水", "0938"}}},
		{"港澳台", []city{{"香港", "00852"}, {"澳门", "00853"}, {"台北", "00886"}}},
		{"广东", []city{{"广州", "020"}, {"韶关", "0751"}, {"深圳", "0755"}, {"珠海", "0756"}, {"汕头", "0754"}, {"佛山", "0757"}, {"湛江", "0759"}, {"茂名", "0668"}, {"东莞", "0769"}, {"中山", "0760"}}},
		{"广西", []city{{"南宁", "0771"}, {"柳州", "0772"}, {"桂林", "0773"}, {"梧州", "0774"}, {"北海", "0779"}, {"玉林", "0775"}}},
		{"贵州", []city{{"贵阳", "0851"}, {"遵义", "0852"}, {"安顺", "0853"}, {"六盘水", "0858"}}},
		{"海南", []city{{"海口", "0898"}, {"三亚", "0899"}}},
		{"河北", []city{{"石家庄", "0311"}, {"邯郸", "0310"}, {"保定", "0312"}, {"张家口", "0313"}, {"承德", "0314"}, {"唐山", "0315"}, {"廊坊", "0316"}, {"沧州", "0317"}, {"衡水", "0318"}, {"邢台", "0319"}, {"秦皇岛", "0335"}}},
		{"河南", []city{{"郑州", "0371"}, {"开封", "0378"}, {"洛阳", "0379"}, {"平顶山", "0375"}, {"安阳", "0372"}, {"鹤壁", "0392"}, {"新乡", "0373"}, {"焦作", "0391"}, {"许昌", "0374"}, {"漯河", "0395"}, {"南阳", "0377"}, {"商丘", "0370"}, {"信阳", "0376"}, {"周口", "0394"}, {"驻马店", "0396"}}},
		{"黑龙江", []city{{"哈尔滨", "0451"}, {"齐齐哈尔", "0452"}, {"牡丹江", "0453"}, {"佳木斯", "0454"}, {"绥化", "0455"}, {"黑河", "0456"}, {"大庆", "0459"}}},
		{"湖北", []city{{"武汉", "027"}, {"十堰", "0719"}, {"宜昌", "0717"}, {"襄阳", "0710"}, {"鄂州", "0711"}, {"荆门", "0724"}, {"孝感", "0712"}, {"荆州", "0716"}, {"黄冈", "0713"}, {"咸宁", "0715"}, {"随州", "0722"}}},
		{"湖南", []city{{"长沙", "0731"}, {"株洲", "0732"}, {"湘潭", "0733"}, {"衡阳", "0734"}, {"岳阳", "0730"}, {"常德", "0736"}, {"张家界", "0744"}, {"益阳", "0737"}, {"郴州", "0735"}, {"永州", "0746"}, {"怀化", "0745"}, {"娄底", "0738"}}},
		{"吉林", []city{{"长春", "0431"}, {"吉林", "0432"}, {"延吉", "0433"}, {"四平", "0434"}, {"通化", "0435"}, {"松原", "0438"}, {"白山", "0439"}}},
		{"江苏", []city{{"南京", "025"}, {"无锡", "0510"}, {"徐州", "0516"}, {"常州", "0519"}, {"苏州", "0512"}, {"南通", "0513"}, {"连云港", "0518"}, {"淮安", "0517"}, {"盐城", "0515"}, {"扬州", "0514"}, {"镇江", "0511"}, {"泰州", "0523"}, {"宿迁", "0527"}}},
		{"江西", []city{{"南昌", "0791"}, {"景德镇", "0798"}, {"萍乡", "0799"}, {"九江", "0792"}, {"新余", "0790"}, {"鹰潭", "0701"}, {"赣州", "0797"}, {"吉安", "0796"}, {"宜春", "0795"}, {"抚州", "0794"}, {"上饶", "0793"}}},
		{"辽宁", []city{{"沈阳", "024"}, {"大连", "0411"}, {"鞍山", "0412"}, {"抚顺", "0413"}, {"本溪", "0414"}, {"丹东", "0415"}, {"锦州", "0416"}, {"营口", "0417"}, {"阜新", "0418"}, {"盘锦", "0427"}, {"葫芦岛", "0429"}}},
		{"内蒙", []city{{"呼和浩特", "0471"}, {"包头", "0472"}, {"乌海", "0473"}, {"赤峰", "0476"}, {"通辽", "0475"}, {"鄂尔多斯", "0477"}, {"呼伦贝尔", "0470"}}},
		{"宁夏", []city{{"银川", "0951"}, {"石嘴山", "0952"}, {"吴忠", "0953"}, {"固原", "0954"}}},
		{"青海", []city{{"西宁", "0971"}, {"海东", "0972"}, {"德令哈", "0977"}, {"格尔木", "0979"}}},
		{"山东", []city{{"济南", "0531"}, {"青岛", "0532"}, {"淄博", "0533"}, {"枣庄", "0632"}, {"东营", "0546"}, {"烟台", "0535"}, {"潍坊", "0536"}, {"济宁", "0537"}, {"泰安", "0538"}, {"威海", "0631"}, {"日照", "0633"}, {"莱芜", "0634"}, {"临沂", "0539"}, {"德州", "0534"}, {"聊城", "0635"}, {"滨州", "0543"}}},
		{"山西", []city{{"太原", "0351"}, {"大同", "0352"}, {"阳泉", "0353"}, {"长治", "0355"}, {"晋城", "0356"}, {"朔州", "0349"}, {"晋中", "0354"}, {"运城", "0359"}, {"忻州", "0350"}, {"临汾", "0357"}, {"吕梁", "0358"}}},
		{"陕西", []city{{"西安", "029"}, {"咸阳", "0910"}, {"铜川", "0919"}, {"渭南", "0913"}, {"延安", "0911"}, {"榆林", "0912"}, {"汉中", "0916"}, {"安康", "0915"}, {"商洛", "0914"}, {"宝鸡", "0917"}}},
		{"上海", []city{{"上海", "021"}}},
		{"四川", []city{{"成都", "028"}, {"攀枝花", "0812"}, {"自贡", "0813"}, {"泸州", "0830"}, {"德阳", "0838"}, {"绵阳", "0816"}, {"广元", "0839"}, {"遂宁", "0825"}, {"内江", "0832"}, {"乐山", "0833"}, {"宜宾", "0831"}, {"南充", "0817"}, {"达州", "0818"}, {"雅安", "0835"}, {"广安", "0826"}, {"巴中", "0827"}, {"西昌", "0834"}}},
		{"天津", []city{{"天津", "022"}}},
		{"西藏", []city{{"拉萨", "0891"}, {"日喀则", "0892"}, {"山南", "0893"}, {"林芝", "0894"}, {"昌都", "0895"}}},
		{"新疆", []city{{"乌鲁木齐", "0991"}, {"克拉玛依", "0990"}, {"石河子", "0993"}, {"吐鲁番", "0995"}, {"哈密", "0902"}, {"喀什", "0998"}, {"伊宁", "0999"}, {"库尔勒", "0996"}, {"阿克苏", "0997"}}},
		{"云南", []city{{"昆明", "0871"}, {"大理", "0872"}, {"曲靖", "0874"}, {"玉溪", "0877"}, {"保山", "0875"}, {"丽江", "0888"}, {"临沧", "0883"}}},
		{"浙江", []city{{"杭州", "0571"}, {"宁波", "0574"}, {"温州", "0577"}, {"嘉兴", "0573"}, {"湖州", "0572"}, {"绍兴", "0575"}, {"金华", "0579"}, {"衢州", "0570"}, {"丽水", "0578"}, {"台州", "0576"}, {"舟山", "0580"}}},
		{"重庆", []city{{"重庆", "023"}}},
	}

	// 已有的同城客栈子板块：视为省份（福建/北京/广东），共建同城保持独立
	existing := map[string]model.Board{}
	var subs []model.Board
	db.Where("parent_id = ?", ch.ID).Find(&subs)
	for _, s := range subs {
		existing[s.Name] = s
	}

	adminID := idByUsername(db, "10000")
	for i, p := range provinces {
		prov, ok := existing[p.name]
		if !ok {
			prov = model.Board{Name: p.name, ParentID: ch.ID, Description: "欢迎" + p.name + "的老乡",
				Sort: i + 1, Status: 1, CreatorID: adminID}
			db.Create(&prov)
		} else {
			db.Model(&prov).UpdateColumn("sort", i+1)
			if prov.CreatorID == 0 {
				db.Model(&prov).UpdateColumn("creator_id", adminID)
			}
		}
		// 省份下的城市（区号唯一，幂等）
		for j, c := range p.cities {
			var ct model.Board
			db.Where("parent_id = ? AND name = ?", prov.ID, c.name).First(&ct)
			if ct.ID == 0 {
				ct = model.Board{Name: c.name, ParentID: prov.ID, CityCode: c.code,
					Description: c.name + "的老乡看过来", Sort: j + 1, Status: 1, CreatorID: adminID}
				db.Create(&ct)
			} else if ct.CityCode == "" {
				db.Model(&ct).UpdateColumn("city_code", c.code)
			}
		}
	}

	// 同城管理（参考 wap_manage）：各省首个城市默认任命超级管理员为「同城管理员」
	var mn int64
	db.Model(&model.CityManager{}).Count(&mn)
	if mn == 0 {
		for _, p := range provinces {
			var prov model.Board
			db.Where("parent_id = ? AND name = ?", ch.ID, p.name).First(&prov)
			if prov.ID == 0 || len(p.cities) == 0 {
				continue
			}
			var ct model.Board
			db.Where("parent_id = ? AND name = ?", prov.ID, p.cities[0].name).First(&ct)
			if ct.ID == 0 {
				continue
			}
			db.Create(&model.CityManager{BoardID: ct.ID, UserID: adminID, Title: "同城管理员"})
		}
	}
}

// seedWordFilters 敏感词库（幂等）
func seedWordFilters(db *gorm.DB) {
	var count int64
	db.Model(&model.WordFilter{}).Count(&count)
	if count > 0 {
		return
	}
	db.Create(&[]model.WordFilter{
		{Word: "垃圾", Replace: "***", Type: 1},
		{Word: "废柴", Replace: "***", Type: 1},
		{Word: "去死", Replace: "***", Type: 1},
		{Word: "傻逼", Replace: "***", Type: 1},
		{Word: "脑残", Replace: "***", Type: 1},
		{Word: "色情", Replace: "***", Type: 1},
		{Word: "赌博", Replace: "***", Type: 2},
		{Word: "代练", Replace: "***", Type: 2},
	})
}

func seedAnnouncements(db *gorm.DB) {
	var count int64
	db.Model(&model.Announcement{}).Count(&count)
	if count > 0 {
		return
	}
	db.Create(&[]model.Announcement{
		{Type: "notice", Title: "欢迎来到家园社区", Content: "在这里，玩家可以随时随地的和好友进行互动，一起玩游戏。社区常年招募管理员与版主，有意者到客服中心申请。"},
		{Type: "broadcast", Title: "行百里者，半于九十", Content: "小Q广播：走一百里路，走了九十里才算走了一半。越接近成功越要认真对待！"},
		{Type: "activity", Title: "五一活动之《歌王就是你》第二届举办帖", Content: "活动时间：即日起至月底。参与方式：在休闲灌水板块发布你的拿手歌曲翻唱帖，回帖数前三名获得社区勋章与金币奖励！"},
	})
}

func seedUsersAndContent(db *gorm.DB) {
	var count int64
	db.Model(&model.User{}).Count(&count)
	if count > 0 {
		return
	}

	mk := func(nickname, signature, color string, gender int, coins, exp int) model.User {
		hash, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
		return model.User{
			Username:  nickname, // 先占位，落库后回填号码
			Nickname:  nickname,
			Password:  string(hash),
			Gender:    gender,
			Signature: signature,
			Color:     color,
			Coins:     coins,
			Exp:       exp,
			Level:     model.CalcLevel(exp),
		}
	}
	users := []model.User{
		mk("站长小Q", "家园振兴，人人有责", "#DAA520", 1, 9999, 9000),
		mk("云起", "云起时，风也温柔", "#DAA520", 2, 560, 3200),
		mk("安珞", "　 　静待花开　ˇ　", "#ff0000", 2, 320, 1800),
		mk("咏荷", "咏而归，荷风送香", "#008000", 2, 210, 900),
		mk("闲云野鹤", "宠辱不惊，看庭前花开花落", "#004299", 1, 150, 600),
		mk("蓝天", "面朝大海，春暖花开", "#800080", 1, 98, 400),
	}
	// 站长密码单独设置（admin123），其余示例用户统一 123456
	adminHash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	users[0].Password = string(adminHash)
	for i := range users {
		if err := db.Create(&users[i]).Error; err != nil {
			log.Println("seed user:", err)
		}
	}
	for i := range users {
		db.Model(&users[i]).Update("username", fmt.Sprintf("%d", users[i].ID))
	}

	// 角色绑定：站长小Q -> 超级管理员
	super := model.Role{}
	db.Where("code = ?", "super_admin").First(&super)
	db.Model(&users[0]).Association("Roles").Replace(&model.Role{ID: super.ID})
	member := model.Role{}
	db.Where("code = ?", "member").First(&member)
	for _, u := range users[1:] {
		db.Model(&u).Association("Roles").Replace(&model.Role{ID: member.ID})
	}

	// 身份与成就（婚恋/贵族/社区成就点）
	db.Model(&users[0]).Updates(map[string]interface{}{"noble": 1, "achieve": 900})
	db.Model(&users[1]).Updates(map[string]interface{}{"noble": 2, "achieve": 320, "partner_id": users[2].ID, "baby_name": "小云朵"})
	db.Model(&users[2]).Updates(map[string]interface{}{"noble": 1, "achieve": 180, "partner_id": users[1].ID})
	db.Model(&users[3]).Updates(map[string]interface{}{"achieve": 90})
	db.Model(&users[4]).Updates(map[string]interface{}{"achieve": 60})
	db.Model(&users[5]).Updates(map[string]interface{}{"achieve": 40})

	// 板块索引
	var boards []model.Board
	db.Find(&boards)
	bid := func(name string) uint {
		for _, b := range boards {
			if b.Name == name {
				return b.ID
			}
		}
		return 0
	}

	now := time.Now()
	type t struct {
		board, title, content string
		user                  int
		top, fine             int
		views                 int
		replies               []struct {
			user  int
			floor string
		}
	}
	threads := []t{
		{"新人求助", "【新手大全实用手册】", "欢迎加入社区。\n幸甚曾拥有三猪，拥有曾几何时青春澎湃的你们。有人在这寻得单纯的友情、青涩的爱情，亦有人寻得逛街的水友，皆由缘起。\n\n【论坛经验等级对照表】发帖+10经验，回帖+5经验，签到+20经验。\n【日常签到】每天签到可得金币，连续7天有惊喜！\n【新人礼包】注册即送100金币，签到还能翻倍哦。\n\n　　　　　　　　家园社区欢迎你！", 0, 1, 1, 1153, []struct {
			user  int
			floor string
		}{{1, "感谢站长整理，收藏了！"}, {2, "新人报到，学习学习~"}, {5, "手册很实用，赞一个"}}},
		{"休闲灌水", "又一个多月没来了", "工作太忙，好不容易摸鱼上来看看，家人们都还好吗？还记得当年的抢车位和好友买卖吗，感慨啊……", 4, 0, 0, 29, []struct {
			user  int
			floor string
		}{{1, "欢迎回来！老友"}, {2, "抢车位！当年我买了十辆劳斯莱斯"}, {3, "好友买卖才是永远的神"}, {0, "魔法花园还有人记得吗，天天浇水"}}},
		{"休闲灌水", "【唱响论坛业务】你点我唱音乐业务正式开业", "规则：点歌请按格式回复——【点歌】歌曲名+赠言。楼主翻唱后@你。\n今日曲目单：孤勇者、晴天、七里香、突然的自我。", 1, 0, 1, 227, []struct {
			user  int
			floor string
		}{{2, "【点歌】晴天+送给三年前的自己"}, {5, "【点歌】七里香+祝家园越来越好"}, {1, "收到，晚上开唱"}}},
		{"时尚美眉", "漂亮的姐妹头", "分享几张自己画的头像，姐妹们喜欢可以自取，记得回帖告诉我用了哪张哦~", 2, 0, 0, 156, []struct {
			user  int
			floor string
		}{{1, "第三张绝了"}, {4, "已取图，谢谢姐妹"}}},
		{"福建", "睡觉睡觉", "夜深了，福建的老乡们都在吗？报个到一起聊聊天。", 1, 0, 0, 65, []struct {
			user  int
			floor string
		}{{3, "厦门报到"}, {0, "福州报到，老乡好"}}},
		{"北京", "孤独的人无所谓", "一个人在北京漂着，习惯了。你们呢？", 5, 0, 0, 50, []struct {
			user  int
			floor string
		}{{4, "抱一个，都不容易"}}},
		{"家族申请", "id10272+霓裳羽衣+申请皇家贵族", "家族名：霓裳羽衣\n家族宗旨：以舞会友，以歌传情\n申请类型：皇家贵族\n家族成员：28人，日活跃15+，望管理员批准！", 2, 0, 0, 89, []struct {
			user  int
			floor string
		}{{0, "材料齐全，予以通过，欢迎霓裳羽衣家族入驻！"}}},
		{"意见建议", "建议增加勋章系统", "建议社区增加勋章系统：签到达人、灌水狂人、人气之星……帖子被加精华也有勋章，这样大家更有动力。", 3, 0, 0, 42, []struct {
			user  int
			floor string
		}{{0, "建议已收录，感谢你对家园的建议！"}}},
		{"数码动漫", "鸿蒙3.0发布日期确认，首批华为、荣耀适配机型已公布", "如题，官方公布了首批适配名单，快看看有没有你的机型。", 5, 0, 0, 78, []struct {
			user  int
			floor string
		}{{5, "华为老机型也适配，良心"}}},
	}
	for i := range threads {
		tt := threads[i]
		th := model.Thread{
			BoardID: bid(tt.board), UserID: users[tt.user].ID,
			Title: tt.title, Content: tt.content,
			IsTop: tt.top, IsFine: tt.fine, ViewCount: tt.views,
		}
		db.Create(&th)
		var last time.Time
		for _, r := range tt.replies {
			last = now.Add(-time.Duration(len(r.floor)) * time.Hour)
			db.Create(&model.Reply{ThreadID: th.ID, UserID: users[r.user].ID, Content: r.floor, Floor: th.ReplyCount + 2, CreatedAt: last})
			th.ReplyCount++
		}
		if !last.IsZero() {
			th.LastReplyAt = &last
		}
		db.Model(&th).Updates(map[string]interface{}{"reply_count": th.ReplyCount, "last_reply_at": th.LastReplyAt})
		db.Model(&model.Board{}).Where("id = ?", th.BoardID).UpdateColumn("thread_count", gorm.Expr("thread_count + 1"))
	}
}

// seedBadges 勋章商店 + 会员勋章种子（复刻诺哈 wap_medal_shop / wap_medal，不含贵族勋章——贵族为身份非勋章）
func seedBadges(db *gorm.DB) {
	var count int64
	db.Model(&model.Badge{}).Count(&count)
	if count == 0 {
		badges := []model.Badge{
			{Name: "公坛协管员", Icon: "706.jpg", Remark: "社区职务徽章", Price: 0, Period: 0, Sort: 1},
			{Name: "公坛管理", Icon: "704.gif", Remark: "公坛管理组", Price: 0, Period: 0, Sort: 2},
			{Name: "客服专员", Icon: "3.gif", Remark: "客服团专属", Price: 0, Period: 0, Sort: 3},
			{Name: "社区传媒", Icon: "501.gif", Remark: "时报记者专属", Price: 0, Period: 0, Sort: 4},
			{Name: "老友归来", Icon: "803.gif", Remark: "回归纪念", Price: 50, Period: 30, Sort: 5},
			{Name: "原创写手", Icon: "804.gif", Remark: "文学贡献", Price: 100, Period: 30, Sort: 6},
			{Name: "尊上", Icon: "903.gif", Remark: "传说中的勋章", Price: 500, Period: 7, Sort: 7},
			{Name: "表情大师", Icon: "45.gif", Remark: "斗图冠军", Price: 200, Period: 30, Sort: 8},
		}
		for i := range badges {
			db.Create(&badges[i])
		}
	}
	// 老库清除「贵族一级/二级」勋章（贵族改为身份展示，非勋章）及其授予记录（幂等）
	var nobleBadges []model.Badge
	db.Where("name IN (?)", []string{"贵族一级", "贵族二级"}).Find(&nobleBadges)
	if len(nobleBadges) > 0 {
		for _, b := range nobleBadges {
			db.Exec("DELETE FROM user_badges WHERE badge_id = ?", b.ID)
		}
		db.Where("name IN (?)", []string{"贵族一级", "贵族二级"}).Delete(&model.Badge{})
	}
	// 老库勋章补齐排序/价格/有效期（幂等）
	existing := []model.Badge{}
	db.Find(&existing)
	meta := map[string]struct{ sort, price, period int }{
		"公坛协管员": {1, 0, 0}, "公坛管理": {2, 0, 0}, "客服专员": {3, 0, 0}, "社区传媒": {4, 0, 0},
		"老友归来": {5, 50, 30}, "原创写手": {6, 100, 30}, "尊上": {7, 500, 7}, "表情大师": {8, 200, 30},
	}
	for _, b := range existing {
		if m, ok := meta[b.Name]; ok {
			db.Model(&model.Badge{}).Where("id = ?", b.ID).Updates(map[string]interface{}{"sort": m.sort, "price": m.price, "period": m.period})
		}
	}
	// 给示例用户发会员勋章（已发过则跳过）
	var ubCount int64
	db.Raw("SELECT COUNT(*) FROM user_badges").Scan(&ubCount)
	if ubCount > 0 {
		return
	}
	type bd struct{ name string }
	all := []model.Badge{}
	db.Find(&all)
	idOf := func(name string) uint {
		for _, b := range all {
			if b.Name == name {
				return b.ID
			}
		}
		return 0
	}
	now := time.Now()
	grant := func(nickname string, avatar string, names ...string) {
		var u model.User
		if err := db.Where("nickname = ?", nickname).First(&u).Error; err != nil {
			return
		}
		if avatar != "" && u.Avatar == "" {
			db.Model(&u).Update("avatar", avatar)
		}
		for i, n := range names {
			if id := idOf(n); id > 0 {
				db.Create(&model.UserBadge{UserID: u.ID, BadgeID: id, Sort: i + 1, GrantedAt: now})
			}
		}
	}
	// 与演示站一致：勋章一串挂在昵称前，等级图标 v{N}.gif 由等级自动生成（贵族为身份，不占勋章位）
	grant("站长小Q", "1985acg.jpg", "公坛协管员", "公坛管理")
	grant("云起", "131851611.jpg", "公坛协管员", "老友归来")
	grant("安珞", "1031047330.png", "社区传媒")
	grant("咏荷", "1031047330.png", "原创写手")
	grant("闲云野鹤", "125703412.png", "老友归来")
	grant("蓝天", "104039478.jpg", "表情大师")
}

// seedGameBoards 游戏论坛分区 + 各游戏子板块（幂等，已存在则跳过）
func seedGameBoards(db *gorm.DB) {
	var ch model.Board
	if err := db.Where("name = ? AND parent_id = 0", "游戏论坛").First(&ch).Error; err != nil {
		ch = model.Board{Name: "游戏论坛", Description: "各游戏交流区：攻略、晒图、组队、交易", Sort: 5}
		db.Create(&ch)
	}
	games := []struct{ name, desc string }{
		{"幻想西游", "经典wap游戏，古典神话网游，再梦西游"},
		{"永恒修仙", "经典wap游戏，永恒修仙，欢迎体验"},
		{"魔法花园", "花的世界，花的海洋，花的物语"},
		{"婚礼殿堂", "闯荡社区快来：婚姻礼堂，寻找爱的另一半！"},
		{"开心农场", "开心农场，播种开心，收获快乐"},
		{"狂抢车位", "停放车辆，展现身价，乐趣无穷"},
		{"精武堂", "江湖格斗，残酷厮杀，随死即生"},
		{"家园宠物", "家园宠物，内测中"},
		{"水果乐园", "轻松娱乐，点缀生活，水果乐园"},
		{"全民猎马", "周二四六，包你赢够，尽在猎马"},
		{"家园股市", "家园股市，一夜成名，瞬间暴富"},
		{"大话吹牛", "大话吹牛，打打闹闹，更是乐哉"},
		// 复刻诺哈 game 目录：slave/arena/apple/ball/guess/marksix/nabob
		{"好友买卖", "买下好友，打工赚钱，奴隶翻身当主人"},
		{"竞技场", "擂台争霸，比武切磋，胜者为王"},
		{"砸金蛋", "金蛋一砸，好运连连"},
		{"台球", "一杆进洞，桌上争雄"},
		{"猜数", "猜数字赢大奖，试试你的运气"},
		{"六合彩", "买马投注，一夜暴富"},
		{"大富翁", "掷骰子走格子，买地收租当富豪"},
	}
	ids := map[string]uint{}
	for _, g := range games {
		var b model.Board
		if err := db.Where("name = ? AND parent_id = ?", g.name, ch.ID).First(&b).Error; err != nil {
			b = model.Board{ParentID: ch.ID, Name: g.name, Description: g.desc}
			db.Create(&b)
		}
		ids[g.name] = b.ID
	}

	// 每个板块补一条示例帖（只在板块还没有帖子时）
	var host model.User
	db.Where("nickname = ?", "站长小Q").First(&host)
	var yun model.User
	db.Where("nickname = ?", "云起").First(&yun)
	samples := []struct {
		game, title, content string
		replies              []string
	}{
		{"精武堂", "【精武堂】第二届武林大会报名帖", "第二届武林大会即日起开放报名！\n赛制：32进16单败淘汰，每天3场，周日决赛。\n奖励：冠军专属马甲+500金币，亚军300金币。\n回帖格式：【报名】游戏ID+常用武器。", []string{"【报名】云起，常用长枪！", "已报名，求虐"}},
		{"魔法花园", "晒花大赛第3期：谁的花最惊艳", "本周主题：玫瑰！\n把你的花园截图发上来，点赞最高的送高级花种×10。", []string{"我的蓝玫瑰呢，先占楼"}},
		{"狂抢车位", "车神争霸赛：谁的车最贵", "晒出你的座驾！劳斯莱斯幻影镇楼，不服来战。", []string{"楼主的幻影被我贴条了哈哈"}},
		{"幻想西游", "【新区】虎年新区开服公告", "虎年新区正式开服！\n开服前3天经验翻倍，冲级榜前10名送神兵利器。", []string{"新区见！老玩家回归"}},
	}
	for _, sp := range samples {
		bid := ids[sp.game]
		if bid == 0 {
			continue
		}
		var cnt int64
		db.Model(&model.Thread{}).Where("board_id = ?", bid).Count(&cnt)
		if cnt > 0 {
			continue
		}
		th := model.Thread{BoardID: bid, UserID: host.ID, Title: sp.title, Content: sp.content, IsFine: 1, ViewCount: 66}
		db.Create(&th)
		var last time.Time
		for i, r := range sp.replies {
			last = time.Now().Add(-time.Duration(i+1) * time.Hour)
			db.Create(&model.Reply{ThreadID: th.ID, UserID: yun.ID, Content: r, Floor: th.ReplyCount + 2, CreatedAt: last})
			th.ReplyCount++
		}
		if !last.IsZero() {
			th.LastReplyAt = &last
		}
		db.Model(&th).Updates(map[string]interface{}{"reply_count": th.ReplyCount, "last_reply_at": th.LastReplyAt})
		db.Model(&model.Board{}).Where("id = ?", bid).UpdateColumn("thread_count", gorm.Expr("thread_count + 1"))
	}
}

// seedGongtan 公坛事务板块 + 演示站公坛协管员账号 35797804（瞿詺南）+ 管理须知两帖
func seedGongtan(db *gorm.DB) {
	var gongtan model.Board
	if err := db.Where("name = ? AND parent_id = 0", "公共论坛").First(&gongtan).Error; err != nil {
		return
	}

	// 1. 公坛事务板块（幂等），并把演示站排序搬过来：时尚 | 休闲 | 公坛
	var gt model.Board
	if err := db.Where("name = ? AND parent_id = ?", "公坛事务", gongtan.ID).First(&gt).Error; err != nil {
		gt = model.Board{ParentID: gongtan.ID, Name: "公坛事务", Description: "公坛区域管理：管理须知、花名册、公示", Sort: 3}
		db.Create(&gt)
	}
	// 演示站广场快捷链顺序：时尚 | 休闲 | 公坛
	boardSorts := map[string]int{"时尚美眉": 1, "休闲灌水": 2, "公坛事务": 3, "新人求助": 4, "情感天地": 5, "数码动漫": 6}
	for name, sort := range boardSorts {
		db.Model(&model.Board{}).Where("name = ? AND parent_id = ?", name, gongtan.ID).Update("sort", sort)
	}

	// 2. 注册公坛协管员账号 35797804（资料照搬演示站 35797804.html）
	var qun model.User
	if err := db.Where("username = ?", "35797804").First(&qun).Error; err != nil {
		hash, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
		qun = model.User{
			ID: 35797804, Username: "35797804",
			Nickname: "　　瞿詺南　",
			Password: string(hash),
			Gender:   1,
			Color:    "#ff0000",
			Avatar:   "104039478.jpg",
			Exp:      79888,
			Coins:    2000,
			Achieve:  378,
			Noble:    2,
		}
		qun.Level = model.CalcLevel(qun.Exp)
		if err := db.Create(&qun).Error; err != nil {
			fmt.Println("seed 35797804:", err)
			return
		}
		// 角色：版主（公坛协管员）
		var mod model.Role
		db.Where("code = ?", "moderator").First(&mod)
		db.Model(&qun).Association("Roles").Append(&mod)
		// 马甲：公坛协管员(706.jpg) + 原创写手(804.gif)
		var bs []model.Badge
		db.Where("icon IN ?", []string{"706.jpg", "804.gif"}).Find(&bs)
		if len(bs) > 0 {
			db.Model(&qun).Association("Badges").Append(&bs)
		}
	}

	// 3. 公坛事务两根官方帖（板块为空时才发）
	var cnt int64
	db.Model(&model.Thread{}).Where("board_id = ?", gt.ID).Count(&cnt)
	if cnt > 0 {
		return
	}
	th1 := model.Thread{
		BoardID: gt.ID, UserID: qun.ID,
		Title: "【公坛区域】管理须知",
		IsTop: 1, IsFine: 1, ViewCount: 520,
		Content: "【公坛区域管理须知】\n\n一、公坛区域范围\n公共论坛下设各板块（休闲灌水、时尚美眉、新人求助、情感天地、数码动漫、公坛事务）均属公坛区域管理范围。\n\n二、管理员职责\n1. 公坛协管员负责日常巡查，及时处理违规帖；\n2. 各板块版主负责本版加精、置顶、删帖；\n3. 发现宣传外网、广告、辱骂等行为，第一时间删帖并上报客服中心。\n\n三、发帖规范\n1. 禁止发布外网链接与广告；\n2. 禁止人身攻击、地域攻击；\n3. 水贴适度，共同维护社区环境。\n\n四、本须知自发布之日起施行，解释权归公坛管理组。\n\n　　　　　　　　家园社区 公坛管理组",
	}
	db.Create(&th1)
	var yun model.User
	db.Where("nickname = ?", "云起").First(&yun)
	r1 := model.Reply{ThreadID: th1.ID, UserID: yun.ID, Content: "收到，各版版主学习一下！", Floor: 2}
	db.Create(&r1)
	th1.ReplyCount = 1
	now := time.Now()
	th1.LastReplyAt = &now
	db.Model(&th1).Updates(map[string]interface{}{"reply_count": 1, "last_reply_at": now})

	th2 := model.Thread{
		BoardID: gt.ID, UserID: qun.ID,
		Title: "【公坛区域管理】在职管理员花名册",
		IsTop: 1, ViewCount: 310,
		Content: "【公坛区域管理】在职管理员花名册\n\n公坛组长：站长小Q（10000）\n公坛协管员：　　瞿詺南　（35797804）\n\n各板块版主：\n休闲灌水：云起（10001）\n时尚美眉：安珞（10002）\n\n（本花名册由公坛管理组维护，任免实时更新，如有变动请回帖公示）",
	}
	db.Create(&th2)
	db.Model(&model.Board{}).Where("id = ?", gt.ID).UpdateColumn("thread_count", gorm.Expr("thread_count + 2"))
}

// seedGames 游戏大厅种子（与演示站 index3.html 一致，幂等）
func seedGames(db *gorm.DB) {
	var boards []model.Board
	db.Find(&boards)
	bid := func(name string) uint {
		for _, b := range boards {
			if b.Name == name {
				return b.ID
			}
		}
		return 0
	}
	// 复刻诺哈 wap_game：net 外站游戏 / com 社区游戏（path 为本站路由入口，空=未开发）
	// 排序对齐诺哈游戏大厅：四大社交游戏(魔法花园/开心农场/狂抢车位/好友买卖)在前，
	// 其余诺哈游戏(竞技场/台球/猜数/六合彩/大富翁/大话吹牛/砸金蛋)居中，非诺哈扩展游戏殿后
	games := []model.Game{
		{Name: "魔法花园", Category: "com", Logo: "mofahuayuan.gif", Stars: "★★★★★", Desc: "花的世界，花的海洋，花的物语", Intro: "播种·浇灌·收获，收集图谱点亮精灵，还可到好友花园采摘！", Path: "/games/garden", BoardID: bid("魔法花园"), Sort: 1},
		{Name: "开心农场", Category: "com", Logo: "kaixinnongchang.gif", Stars: "★★★★★", Desc: "开心农场，播种开心，收获快乐", Intro: "翻地播种浇水施肥，偷菜设陷阱，还能卖果实赚G币！", Path: "/games/farm", BoardID: bid("开心农场"), Sort: 2},
		{Name: "狂抢车位", Category: "com", Logo: "kuangqiangchewei.gif", Stars: "★★★☆☆", Desc: "停放车辆，展现身价，乐趣无穷", Intro: "买车停车抢车位，贴条没收罚金，超时收入入国库，还能赠好友豪车！", Path: "/games/park", BoardID: bid("狂抢车位"), Sort: 3},
		{Name: "好友买卖", Category: "com", Logo: "", Stars: "★★★★★", Desc: "买下好友，打工赚钱，奴隶翻身当主人", Intro: "把好友买来做奴隶，让他打工赚钱，还可以身价翻倍转卖", BoardID: bid("好友买卖"), Sort: 4},
		{Name: "竞技场", Category: "com", Logo: "", Stars: "★★★★☆", Desc: "热血江湖，擂台争霸，胜者为王", Intro: "挑战好友擂台，胜场提升段位，冲击竞技之巅", BoardID: bid("竞技场"), Sort: 5},
		{Name: "台球", Category: "com", Logo: "", Stars: "★★★☆☆", Desc: "一杆进洞，桌上争雄", Intro: "好友对战台球，展示你的杆法与技巧", BoardID: bid("台球"), Sort: 6},
		{Name: "猜数", Category: "com", Logo: "", Stars: "★★★☆☆", Desc: "猜数字赢大奖，试试你的运气", Intro: "参与竞猜，猜中大奖抱回家", BoardID: bid("猜数"), Sort: 7},
		{Name: "六合彩", Category: "com", Logo: "", Stars: "★★★☆☆", Desc: "买马投注，一夜暴富", Intro: "六合彩开奖，买中即赚", BoardID: bid("六合彩"), Sort: 8},
		{Name: "大富翁", Category: "com", Logo: "", Stars: "★★★★☆", Desc: "超级富翁，掷骰子走格子，买地收租", Intro: "掷骰前进，买地建屋，收租致富", BoardID: bid("大富翁"), Sort: 9},
		{Name: "大话吹牛", Category: "com", Logo: "dahuachuiniu.gif", Stars: "★★★☆☆", Desc: "大话吹牛，打打闹闹，更是乐哉", Intro: "吹牛打闹，好友互喷，乐在其中", BoardID: bid("大话吹牛"), Sort: 10},
		{Name: "砸金蛋", Category: "com", Logo: "", Stars: "★★★☆☆", Desc: "金蛋一砸，好运连连", Intro: "花G币砸金蛋，砸出金币元宝惊喜不断", BoardID: bid("砸金蛋"), Sort: 11},
		{Name: "婚礼殿堂", Category: "com", Logo: "hunli2.jpg", Stars: "★★★★★", Desc: "闯荡社区快来: 婚姻礼堂 寻找爱的另一半！", BoardID: bid("婚礼殿堂"), Sort: 12},
		{Name: "精武堂", Category: "com", Logo: "jwt.png", Stars: "★★★★★", Desc: "江湖格斗，残酷厮杀，随死即生", BoardID: bid("精武堂"), Sort: 13},
		{Name: "家园宠物", Category: "com", Logo: "cwlogo.gif", Stars: "★★", Desc: "家园宠物，内测中", BoardID: bid("家园宠物"), Sort: 14},
		{Name: "水果乐园", Category: "com", Logo: "shuiguoleyuan.gif", Stars: "★★☆☆☆", Desc: "轻松娱乐，点缀生活，水果乐园", BoardID: bid("水果乐园"), Sort: 15},
		{Name: "全民猎马", Category: "com", Logo: "quanminliema.gif", Stars: "★★★★☆", Desc: "周二四六，包你赢够，尽在猎马", BoardID: bid("全民猎马"), Sort: 16},
		{Name: "家园股市", Category: "com", Logo: "jiayuangushi.gif", Stars: "★☆☆☆☆", Desc: "家园股市，一夜成名，瞬间暴富", BoardID: bid("家园股市"), Sort: 17},
		{Name: "幻想西游", Category: "net", Logo: "", Stars: "★★★★★", Desc: "经典wap游戏，古典神话网游，再梦西游。持神兵利器，降五爪金龙，携爱行走西游", BoardID: bid("幻想西游"), Sort: 1},
		{Name: "永恒修仙", Category: "net", Logo: "logo.jpg", Stars: "★★★★★", Desc: "经典wap游戏，永恒修仙。欢迎体验", BoardID: bid("永恒修仙"), Sort: 2},
	}
	for i := range games {
		var exist int64
		db.Model(&model.Game{}).Where("name = ?", games[i].Name).Count(&exist)
		if exist == 0 {
			db.Create(&games[i])
		} else {
			// 老库补齐新字段（幂等）
			db.Model(&model.Game{}).Where("name = ?", games[i].Name).Updates(map[string]interface{}{
				"intro": games[i].Intro, "path": games[i].Path, "sort": games[i].Sort,
			})
		}
	}
}

// seedGuestbook 站长留言本样例（幂等：表为空才写入）
func seedGuestbook(db *gorm.DB) {
	var count int64
	db.Model(&model.GuestBook{}).Count(&count)
	if count > 0 {
		return
	}
	adminID := idByUsername(db, "10000")
	rows := []struct {
		name, content string
		private       bool
	}{
		{"游客", "家园社区重新上线啦，欢迎新老朋友回家！", false},
		{"站长小Q", "签到、盖楼、灌水、聊天交友，当年的快乐都回来了。", false},
		{"游客", "找回当年的家园号码，满满都是回忆。", true},
	}
	for _, r := range rows {
		g := model.GuestBook{Name: r.name, Content: r.content, Status: 1}
		if r.private {
			g.Pass = "seed" // 私密样例，密码可输入"seed"查看
		}
		if r.name == "站长小Q" {
			g.UserID = adminID
		}
		db.Create(&g)
	}
	if adminID > 0 {
		var first model.GuestBook
		db.Where("name = ?", "游客").First(&first)
		db.Create(&model.GuestReply{GuestID: first.ID, UserID: adminID, Content: "欢迎常来！", Status: 1})
	}
}

func idByUsername(db *gorm.DB, username string) uint {
	var u model.User
	db.Select("id").Where("username = ?", username).First(&u)
	return u.ID
}

// seedActivities 活动专区演示帖（幂等：无活动帖时，挑若干既有帖标为活动帖）
func seedActivities(db *gorm.DB) {
	var n int64
	db.Model(&model.Thread{}).Where("is_active = 1 AND status = 1").Count(&n)
	if n > 0 {
		return
	}
	var ids []uint
	db.Model(&model.Thread{}).Where("status = 1 AND type = 0").
		Order("id DESC").Limit(4).Pluck("id", &ids)
	for _, id := range ids {
		db.Model(&model.Thread{}).Where("id = ?", id).Update("is_active", 1)
	}
}

// seedSiteArticles 文章专栏样例（分类 + 文章，幂等）
func seedSiteArticles(db *gorm.DB) {
	var n int64
	db.Model(&model.SiteArticleCategory{}).Count(&n)
	if n == 0 {
		for i, name := range []string{"社区公告", "心情随笔", "攻略分享"} {
			db.Create(&model.SiteArticleCategory{Name: name, Sort: i})
		}
	}
	db.Model(&model.SiteArticle{}).Count(&n)
	if n == 0 {
		var adminID = idByUsername(db, "10000")
		arts := []struct {
			title, cat, content string
		}{
			{"家园社区开放公测", "社区公告", "家园社区以 QQ家园为蓝本复刻上线，欢迎大家来盖楼、灌水、交朋友！找回当年的家园号码，续写你的青春记忆。"},
			{"写在重开的第一天", "心情随笔", "还记得吗？那年我们用 3GQQ 登上家园，签到、浇水、抢车位……如今一切都回来了。"},
			{"新手指南：如何快速升级", "攻略分享", "每天登录+1活跃天，多回帖、多签到、去打工，金币经验涨得飞快。记得开通个人空间，写写心情记录生活。"},
		}
		for _, a := range arts {
			var catID uint
			db.Model(&model.SiteArticleCategory{}).Where("name = ?", a.cat).Select("id").Scan(&catID)
			db.Create(&model.SiteArticle{
				UserID: adminID, Title: a.title, CatID: catID, Content: a.content,
				Writer: "站长小Q", Source: "家园社区", Status: 1,
			})
		}
	}
}

// seedShop 商店样例（分类 + 站长店铺 + 商品，幂等）
func seedShop(db *gorm.DB) {
	var n int64
	db.Model(&model.ShopCategory{}).Count(&n)
	if n == 0 {
		for i, name := range []string{"装扮道具", "功能道具", "互动道具", "稀有道具"} {
			db.Create(&model.ShopCategory{Name: name, Sort: i})
		}
	}
	db.Model(&model.ShopGoods{}).Count(&n)
	if n == 0 {
		adminID := idByUsername(db, "10000")
		if adminID > 0 {
			db.Where(&model.Shop{UserID: adminID}).FirstOrCreate(&model.Shop{UserID: adminID, Name: "站长杂货铺", Status: 1})
			catID := func(name string) uint {
				var id uint
				db.Model(&model.ShopCategory{}).Where("name = ?", name).Select("id").Scan(&id)
				return id
			}
			goods := []struct {
				name, cat, intro string
				price, amount    int
			}{
				{"家园记忆相册", "装扮道具", "把在家园的点滴装订成册，放在个人空间展示。", 120, 50},
				{"超值新手礼包", "功能道具", "内含金币与经验加成，助力快速升级。", 88, 100},
				{"交友名片卡", "互动道具", "在广场展示自己的名片，更容易被好友发现。", 66, 80},
				{"限量纪念勋章", "稀有道具", "复刻当年家园的限量勋章，戴上它做最靓的仔。", 520, 20},
			}
			for _, g := range goods {
				db.Create(&model.ShopGoods{
					UserID: adminID, Name: g.name, CatID: catID(g.cat), BidMT: 1, Money: 0,
					Price: g.price, Amount: g.amount, Intro: g.intro, Status: 1,
				})
			}
		}
	}
}
