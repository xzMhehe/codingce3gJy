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
	err := db.AutoMigrate(
		&model.User{}, &model.Role{}, &model.Permission{},
		&model.Board{}, &model.Thread{}, &model.Reply{},
		&model.Announcement{}, &model.SignIn{},
		&model.Friendship{}, &model.PrivateMessage{},
		&model.ChatMessage{}, &model.Notification{},
		&model.Badge{},
		&model.Game{},
		&model.Resource{},
		&model.Space{}, &model.Mood{}, &model.MoodComment{},
		&model.Article{}, &model.Album{}, &model.Photo{},
		&model.SpaceMessage{}, &model.Visitor{},
		&model.BankAccount{}, &model.WorkRecord{},
		&model.Family{}, &model.FamilyMember{}, &model.FamilySignIn{},
		&model.FamilyActivity{},
		&model.Book{}, &model.ThreadFavorite{},
		&model.FriendGroup{}, &model.FriendGroupItem{},
		&model.GardenPlot{}, &model.MyGame{}, &model.UserFlower{},
		&model.GardenActivity{}, &model.Donation{}, &model.PlazaSection{},
		&model.NoblePlan{}, &model.Good{},
	)
	if err != nil {
		log.Fatalf("建表失败: %v", err)
	}
	db.Exec("ALTER TABLE users AUTO_INCREMENT = 10000")
	db.Exec("ALTER TABLE threads AUTO_INCREMENT = 10000")

	seedRBAC(db)
	seedBoards(db)
	seedAnnouncements(db)
	seedUsersAndContent(db)
	seedBadges(db)
	seedGameBoards(db)
	seedGongtan(db)
	seedMigrate2026(db)
	seedGames(db)
	seedFamilies(db)
	seedFamilyPatch(db)
	seedBooks(db)
	seedGardenActivities(db)
	seedPlazaSections(db)
	seedNoblePlans(db)
	seedGoods(db)
	seedResources(db, staticDir)
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

// seedGardenActivities 花园示例活动（幂等）
func seedGardenActivities(db *gorm.DB) {
	var n int64
	db.Model(&model.GardenActivity{}).Count(&n)
	if n > 0 {
		return
	}
	for _, a := range []model.GardenActivity{
		{Title: "春天的爱恋", Desc: "收集春天花朵，赢取限定花种。"},
		{Title: "小魔女的烦恼", Desc: "帮助小魔女完成任务，获得魔法药水。"},
		{Title: "花仙子的新房子", Desc: "装饰花仙子的小屋，赢取家园装扮。"},
		{Title: "寻找遗失的碎片", Desc: "集齐碎片，兑换稀有花盆。"},
	} {
		db.Create(&a)
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
	if n > 0 {
		return
	}
	for _, p := range model.NoblePlanPresets {
		db.Create(&p)
	}
}

// seedGoods 道具商城示例（幂等）
func seedGoods(db *gorm.DB) {
	var n int64
	db.Model(&model.Good{}).Count(&n)
	if n > 0 {
		return
	}
	for _, g := range model.GoodPresets {
		db.Create(&g)
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
				db.Create(&model.Resource{File: file, Category: "other", Name: strings.TrimSuffix(name, ext), Status: 1})
				added++
			}
		}
	}
	return added
}

// seedMigrate2026 迁移演示站其余友友与帖子，全部时间落在 2026 年
// （逐条幂等：用户按号码去重，帖子按「标题+板块」去重，可安全重复执行）
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

// seedBadges 勋章/马甲种子：图标均来自演示站 static/picture 素材
func seedBadges(db *gorm.DB) {
	var count int64
	db.Model(&model.Badge{}).Count(&count)
	if count == 0 {
		badges := []model.Badge{
			{Name: "公坛协管员", Icon: "706.jpg", Remark: "社区职务徽章"},
			{Name: "公坛管理", Icon: "704.gif", Remark: "公坛管理组"},
			{Name: "客服专员", Icon: "3.gif", Remark: "客服团专属"},
			{Name: "社区传媒", Icon: "501.gif", Remark: "时报记者专属"},
			{Name: "老友归来", Icon: "803.gif", Remark: "回归纪念"},
			{Name: "原创写手", Icon: "804.gif", Remark: "文学贡献"},
			{Name: "贵族一级", Icon: "103.gif", Remark: "贵族身份"},
			{Name: "贵族二级", Icon: "15.gif", Remark: "贵族身份"},
			{Name: "尊上", Icon: "903.gif", Remark: "传说中的马甲"},
			{Name: "表情大师", Icon: "45.gif", Remark: "斗图冠军"},
		}
		for i := range badges {
			db.Create(&badges[i])
		}
	}
	// 给示例用户发马甲与头像（已发过则跳过）
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
	grant := func(nickname string, avatar string, names ...string) {
		var u model.User
		if err := db.Where("nickname = ?", nickname).First(&u).Error; err != nil {
			return
		}
		if avatar != "" && u.Avatar == "" {
			db.Model(&u).Update("avatar", avatar)
		}
		var bs []model.Badge
		for _, n := range names {
			if id := idOf(n); id > 0 {
				bs = append(bs, model.Badge{ID: id})
			}
		}
		if len(bs) > 0 {
			db.Model(&u).Association("Badges").Append(&bs)
		}
	}
	// 与演示站一致：马甲一串挂在昵称前，等级图标 v{N}.gif 由等级自动生成
	grant("站长小Q", "1985acg.jpg", "公坛协管员", "公坛管理", "贵族一级")
	grant("云起", "131851611.jpg", "公坛协管员", "老友归来", "贵族一级")
	grant("安珞", "1031047330.png", "社区传媒", "贵族二级")
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
	var count int64
	db.Model(&model.Game{}).Count(&count)
	if count > 0 {
		return
	}
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
	games := []model.Game{
		{Name: "幻想西游", Category: "net", Logo: "", Stars: "★★★★★", Desc: "经典wap游戏，古典神话网游，再梦西游。持神兵利器，降五爪金龙，携爱行走西游", BoardID: bid("幻想西游"), Sort: 1},
		{Name: "永恒修仙", Category: "net", Logo: "logo.jpg", Stars: "★★★★★", Desc: "经典wap游戏，永恒修仙。欢迎体验", BoardID: bid("永恒修仙"), Sort: 2},
		{Name: "魔法花园", Category: "com", Logo: "mofahuayuan.gif", Stars: "★★★★★", Desc: "花的世界，花的海洋，花的物语", BoardID: bid("魔法花园"), Sort: 1},
		{Name: "婚礼殿堂", Category: "com", Logo: "hunli2.jpg", Stars: "★★★★★", Desc: "闯荡社区快来: 婚姻礼堂 寻找爱的另一半！", BoardID: bid("婚礼殿堂"), Sort: 2},
		{Name: "开心农场", Category: "com", Logo: "kaixinnongchang.gif", Stars: "★★★★☆", Desc: "开心农场，播种开心，收获快乐", BoardID: bid("开心农场"), Sort: 3},
		{Name: "狂抢车位", Category: "com", Logo: "kuangqiangchewei.gif", Stars: "★★★☆☆", Desc: "停放车辆，展现身价，乐趣无穷", BoardID: bid("狂抢车位"), Sort: 4},
		{Name: "精武堂", Category: "com", Logo: "jwt.png", Stars: "★★★★★", Desc: "江湖格斗，残酷厮杀，随死即生", BoardID: bid("精武堂"), Sort: 5},
		{Name: "家园宠物", Category: "com", Logo: "cwlogo.gif", Stars: "★★", Desc: "家园宠物，内测中", BoardID: bid("家园宠物"), Sort: 6},
		{Name: "水果乐园", Category: "com", Logo: "shuiguoleyuan.gif", Stars: "★★☆☆☆", Desc: "轻松娱乐，点缀生活，水果乐园", BoardID: bid("水果乐园"), Sort: 7},
		{Name: "全民猎马", Category: "com", Logo: "quanminliema.gif", Stars: "★★★★☆", Desc: "周二四六，包你赢够，尽在猎马", BoardID: bid("全民猎马"), Sort: 8},
		{Name: "家园股市", Category: "com", Logo: "jiayuangushi.gif", Stars: "★☆☆☆☆", Desc: "家园股市，一夜成名，瞬间暴富", BoardID: bid("家园股市"), Sort: 9},
		{Name: "大话吹牛", Category: "com", Logo: "dahuachuiniu.gif", Stars: "★★★☆☆", Desc: "大话吹牛，打打闹闹，更是乐哉", BoardID: bid("大话吹牛"), Sort: 10},
	}
	for i := range games {
		db.Create(&games[i])
	}
}
