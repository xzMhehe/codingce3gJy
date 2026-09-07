package model

import "time"

// ============ 家园(Home)扩展 —— 对齐诺哈三代 wap_home / wap_news / wap_bbs_favor / wap_qq / wap_user_contact ============

// Home 家园主页统计（诺哈 wap_home）
type Home struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `gorm:"uniqueIndex" json:"user_id"`
	Name           string    `gorm:"type:varchar(50)" json:"name"`  // 家园名
	Point          int       `gorm:"default:0" json:"point"`        // 活跃点（每日首登+1）
	Moods          int       `gorm:"default:0" json:"moods"`        // 心情数
	Visitors       int       `gorm:"default:0" json:"visitors"`     // 访客总数
	Messages       int       `gorm:"default:0" json:"messages"`     // 留言总数
	LastActiveDate string    `gorm:"type:varchar(10)" json:"last_active_date"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (Home) TableName() string { return "homes" }

// HomeNews 新鲜事（诺哈 wap_news）
// NType: 1=发帖 2=回帖 101=心情 102=日志 103=照片 104=空间留言 105=商品
type HomeNews struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`  // 动作发起者
	NID       uint      `gorm:"default:0" json:"nid"`  // 相关人（0=系统）
	NType     int       `json:"ntype"`                 // 动作类型
	RefID     uint      `gorm:"default:0" json:"ref_id"` // 关联对象 id（帖子/日志/照片…）
	Content   string    `gorm:"type:varchar(200)" json:"content"` // 动作文案（如"发表了心情"）
	CreatedAt time.Time `json:"created_at"`
}

func (HomeNews) TableName() string { return "home_news" }

// HomeFavorite 我的收藏（诺哈 wap_bbs_favor 泛化）
// FType: 1=论坛 2=帖子 3=家族 4=日志 5=相册 6=商品
type HomeFavorite struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index:idx_fav_uniq" json:"user_id"`
	FType     int       `gorm:"index:idx_fav_uniq" json:"f_type"`
	RefID     uint      `json:"ref_id"`
	Name      string    `gorm:"type:varchar(100)" json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

func (HomeFavorite) TableName() string { return "home_favorites" }

// UserContact 通讯录（诺哈 wap_user_contact：QQ/邮箱/手机任一可作联系方式）
type UserContact struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"uniqueIndex" json:"user_id"`
	QQ        string    `gorm:"type:varchar(16)" json:"qq"`   // QQ 号（5-11 位数字）
	Mail      string    `gorm:"type:varchar(50)" json:"mail"` // 邮箱
	Phone     string    `gorm:"type:varchar(16)" json:"phone"` // 手机号
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (UserContact) TableName() string { return "user_contacts" }

// Invite 邀请开通家园（诺哈 invite.asp + wap_user_promo 推荐奖励）
type Invite struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	UserID    uint       `gorm:"index" json:"user_id"` // 邀请人
	Code      string     `gorm:"type:varchar(16);uniqueIndex" json:"code"`
	UsedUID   uint       `gorm:"default:0" json:"used_uid"` // 被邀请人
	Status    int        `gorm:"default:0" json:"status"`   // 0未使用 1已使用
	CreatedAt time.Time  `json:"created_at"`
	UsedAt    *time.Time `json:"used_at"`
}

func (Invite) TableName() string { return "invites" }

// ============ 留言本（诺哈 wap_guest / wap_guest_reply，全站留言板+私密留言） ============

type GuestBook struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"default:0" json:"user_id"` // 留言者（0=游客）
	Name      string    `gorm:"type:varchar(20)" json:"name"` // 留言者名（游客手输）
	Pass      string    `gorm:"type:varchar(32)" json:"-"` // 私密留言密码（空=公开）
	Content   string    `gorm:"type:varchar(500)" json:"content"`
	Status    int       `gorm:"default:1" json:"status"` // 1正常 0删除
	CreatedAt time.Time `json:"created_at"`
}

func (GuestBook) TableName() string { return "guest_books" }

type GuestReply struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	GuestID   uint      `gorm:"index" json:"guest_id"`
	UserID    uint      `json:"user_id"` // 回复人（管理员）
	Content   string    `gorm:"type:varchar(300)" json:"content"`
	Status    int       `gorm:"default:1" json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func (GuestReply) TableName() string { return "guest_replies" }

// ============ 文章（诺哈 wap_article / wap_article_comment，社区专栏） ============

type SiteArticleCategory struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"type:varchar(30);uniqueIndex" json:"name"`
	Sort      int       `gorm:"default:0" json:"sort"`
	CreatedAt time.Time `json:"created_at"`
}

func (SiteArticleCategory) TableName() string { return "site_article_categories" }

// SiteArticle 社区文章（区别于空间日志 articles）
type SiteArticle struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"` // 作者
	Title     string    `gorm:"type:varchar(60)" json:"title"`
	CatID     uint      `gorm:"default:0" json:"cat_id"`
	Plate     string    `gorm:"type:varchar(30)" json:"plate"` // 专栏
	Image     string    `gorm:"type:varchar(100)" json:"image"`
	Click     int       `gorm:"default:0" json:"click"`
	Writer    string    `gorm:"type:varchar(20)" json:"writer"`  // 原作者
	Source    string    `gorm:"type:varchar(50)" json:"source"`  // 来源
	Comment   int       `gorm:"default:0" json:"comment"`        // 评论数
	Content   string    `gorm:"type:text" json:"content"`
	Status    int       `gorm:"default:1" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (SiteArticle) TableName() string { return "site_articles" }

type SiteArticleComment struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ArticleID uint      `gorm:"index" json:"article_id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Content   string    `gorm:"type:varchar(300)" json:"content"`
	Status    int       `gorm:"default:1" json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func (SiteArticleComment) TableName() string { return "site_article_comments" }

// ============ 商店（诺哈 wap_shop / wap_shop_trade / wap_shop_order / wap_shop_comment，道具买卖） ============

type ShopCategory struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"type:varchar(30);uniqueIndex" json:"name"`
	Sort      int       `gorm:"default:0" json:"sort"`
	CreatedAt time.Time `json:"created_at"`
}

func (ShopCategory) TableName() string { return "shop_categories" }

// Shop 店铺（一人一店）
type Shop struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"uniqueIndex" json:"user_id"`
	Name      string    `gorm:"type:varchar(30)" json:"name"`
	Status    int       `gorm:"default:1" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Shop) TableName() string { return "shops" }

// ShopGoods 商品（BidMT 交易模式：1=一口价）
type ShopGoods struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"` // 店主=卖家
	Name      string    `gorm:"type:varchar(60)" json:"name"`
	CatID     uint      `gorm:"default:0" json:"cat_id"`
	BidMT     int       `gorm:"default:1" json:"bid_mt"` // 1=一口价 2=求购
	Money     int       `gorm:"default:0" json:"money"`  // 币种 0=G币
	Price     int       `json:"price"`
	Amount    int       `gorm:"default:1" json:"amount"` // 库存
	Sales     int       `gorm:"default:0" json:"sales"`  // 销量
	Intro     string    `gorm:"type:varchar(500)" json:"intro"`
	Image     string    `gorm:"type:varchar(100)" json:"image"`
	Clicks    int       `gorm:"default:0" json:"clicks"`
	Comment   int       `gorm:"default:0" json:"comment"`
	Status    int       `gorm:"default:1" json:"status"` // 1上架 0下架
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (ShopGoods) TableName() string { return "shop_goods" }

// ShopOrder 订单 Status: 1待付款 2待发货 3待收货 4完成 5取消
type ShopOrder struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	GoodsID   uint      `gorm:"index" json:"goods_id"`
	SellerID  uint      `gorm:"index" json:"seller_id"`
	BuyerID   uint      `gorm:"index" json:"buyer_id"`
	GoodsName string    `gorm:"type:varchar(60)" json:"goods_name"`
	Money     int       `gorm:"default:0" json:"money"`
	Price     int       `json:"price"`
	Amount    int       `gorm:"default:1" json:"amount"`
	Status    int       `gorm:"default:1" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (ShopOrder) TableName() string { return "shop_orders" }

// ShopComment 商品评价 DType: 1=好评 2=中评 3=差评
type ShopComment struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	GoodsID   uint      `gorm:"index" json:"goods_id"`
	OrderID   uint      `gorm:"default:0" json:"order_id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	DType     int       `gorm:"default:1" json:"d_type"`
	Content   string    `gorm:"type:varchar(300)" json:"content"`
	Reply     string    `gorm:"type:varchar(300)" json:"reply"` // 店主回复
	Status    int       `gorm:"default:1" json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func (ShopComment) TableName() string { return "shop_comments" }

// ArticleComment 空间日志评论（诺哈 wap_blog_article_comment）
type ArticleComment struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ArticleID uint      `gorm:"index" json:"article_id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Content   string    `gorm:"type:varchar(300)" json:"content"`
	Status    int       `gorm:"default:1" json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func (ArticleComment) TableName() string { return "article_comments" }
