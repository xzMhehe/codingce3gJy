package model

import "time"

// ============ 家园(Home)扩展 —— 对齐诺哈三代 wap_home / wap_news / wap_bbs_favor / wap_qq / wap_user_contact ============

// Home 家园主页统计（诺哈 wap_home）
type Home struct {
	ID             uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID         uint      `gorm:"uniqueIndex;comment:用户ID" json:"user_id"`
	Name           string    `gorm:"type:varchar(50);comment:家园名" json:"name"`   // 家园名
	Point          int       `gorm:"default:0;comment:活跃点（每日首登+1）" json:"point"` // 活跃点（每日首登+1）
	Moods          int       `gorm:"default:0;comment:心情数" json:"moods"`         // 心情数
	Visitors       int       `gorm:"default:0;comment:访客总数" json:"visitors"`     // 访客总数
	Messages       int       `gorm:"default:0;comment:留言总数" json:"messages"`     // 留言总数
	LastActiveDate string    `gorm:"type:varchar(10);comment:最后活跃日期" json:"last_active_date"`
	CreatedAt      time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt      time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (Home) TableName() string { return "homes" }

// HomeNews 新鲜事（诺哈 wap_news）
// NType: 1=发帖 2=回帖 101=心情 102=日志 103=照片 104=空间留言 105=商品
type HomeNews struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint      `gorm:"index;comment:动作发起者" json:"user_id"`                      // 动作发起者
	NID       uint      `gorm:"default:0;comment:相关人（0=系统）" json:"nid"`                  // 相关人（0=系统）
	NType     int       `gorm:"comment:动作类型" json:"ntype"`                               // 动作类型
	RefID     uint      `gorm:"default:0;comment:关联对象 id（帖子/日志/照片…）" json:"ref_id"`      // 关联对象 id（帖子/日志/照片…）
	Content   string    `gorm:"type:varchar(200);comment:动作文案（如”发表了心情”）" json:"content"` // 动作文案（如"发表了心情"）
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (HomeNews) TableName() string { return "home_news" }

// HomeFavorite 我的收藏（诺哈 wap_bbs_favor 泛化）
// FType: 1=论坛 2=帖子 3=家族 4=日志 5=相册 6=商品
type HomeFavorite struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint      `gorm:"index:idx_fav_uniq;comment:用户ID" json:"user_id"`
	FType     int       `gorm:"index:idx_fav_uniq;comment:F类型" json:"f_type"`
	RefID     uint      `gorm:"comment:关联ID" json:"ref_id"`
	Name      string    `gorm:"type:varchar(100);comment:名称" json:"name"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (HomeFavorite) TableName() string { return "home_favorites" }

// UserContact 通讯录（诺哈 wap_user_contact：QQ/邮箱/手机任一可作联系方式）
type UserContact struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint      `gorm:"uniqueIndex;comment:用户ID" json:"user_id"`
	QQ        string    `gorm:"type:varchar(16);comment:QQ 号（5-11 位数字）" json:"qq"` // QQ 号（5-11 位数字）
	Mail      string    `gorm:"type:varchar(50);comment:邮箱" json:"mail"`           // 邮箱
	Phone     string    `gorm:"type:varchar(16);comment:手机号" json:"phone"`         // 手机号
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (UserContact) TableName() string { return "user_contacts" }

// PhoneAudit 手机号验证（对齐诺哈 wap_phone）：用户提交手机号待审核，管理员通过后写入联系方式
type PhoneAudit struct {
	ID        uint       `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint       `gorm:"index;comment:用户ID" json:"user_id"`
	Phone     string     `gorm:"type:varchar(16);comment:Phone" json:"phone"`
	Status    int        `gorm:"default:0;comment:0待审核 1通过 2拒绝" json:"status"` // 0待审核 1通过 2拒绝
	HandledAt *time.Time `gorm:"comment:已处理时间" json:"handled_at"`
	CreatedAt time.Time  `gorm:"comment:创建时间" json:"created_at"`
	User      *User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (PhoneAudit) TableName() string { return "phone_audits" }

// Invite 邀请开通家园（诺哈 invite.asp + wap_user_promo 推荐奖励）
type Invite struct {
	ID        uint       `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint       `gorm:"index;comment:邀请人" json:"user_id"` // 邀请人
	Code      string     `gorm:"type:varchar(16);uniqueIndex;comment:编码" json:"code"`
	UsedUID   uint       `gorm:"default:0;comment:被邀请人" json:"used_uid"`    // 被邀请人
	Status    int        `gorm:"default:0;comment:0未使用 1已使用" json:"status"` // 0未使用 1已使用
	CreatedAt time.Time  `gorm:"comment:创建时间" json:"created_at"`
	UsedAt    *time.Time `gorm:"comment:已用时间" json:"used_at"`
}

func (Invite) TableName() string { return "invites" }

// ============ 留言本（诺哈 wap_guest / wap_guest_reply，全站留言板+私密留言） ============

type GuestBook struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint      `gorm:"default:0;comment:留言者（0=游客）" json:"user_id"`      // 留言者（0=游客）
	Name      string    `gorm:"type:varchar(20);comment:留言者名（游客手输）" json:"name"` // 留言者名（游客手输）
	Pass      string    `gorm:"type:varchar(32);comment:私密留言密码（空=公开）" json:"-"`  // 私密留言密码（空=公开）
	Content   string    `gorm:"type:varchar(500);comment:内容" json:"content"`
	Status    int       `gorm:"default:1;comment:1正常 0删除" json:"status"` // 1正常 0删除
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (GuestBook) TableName() string { return "guest_books" }

type GuestReply struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	GuestID   uint      `gorm:"index;comment:GuestID" json:"guest_id"`
	UserID    uint      `gorm:"comment:回复人（管理员）" json:"user_id"` // 回复人（管理员）
	Content   string    `gorm:"type:varchar(300);comment:内容" json:"content"`
	Status    int       `gorm:"default:1;comment:状态" json:"status"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (GuestReply) TableName() string { return "guest_replies" }

// ============ 文章（诺哈 wap_article / wap_article_comment，社区专栏） ============

type SiteArticleCategory struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	Name      string    `gorm:"type:varchar(30);uniqueIndex;comment:名称" json:"name"`
	Sort      int       `gorm:"default:0;comment:排序值" json:"sort"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (SiteArticleCategory) TableName() string { return "site_article_categories" }

// SiteArticle 社区文章（区别于空间日志 articles）
type SiteArticle struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint      `gorm:"index;comment:用户ID（作者）" json:"user_id"` // 作者
	Title     string    `gorm:"type:varchar(60);comment:标题" json:"title"`
	CatID     uint      `gorm:"default:0;comment:分类ID" json:"cat_id"`
	Plate     string    `gorm:"type:varchar(30);comment:专栏" json:"plate"` // 专栏
	Image     string    `gorm:"type:varchar(100);comment:图片" json:"image"`
	Click     int       `gorm:"default:0;comment:点击" json:"click"`
	Writer    string    `gorm:"type:varchar(20);comment:原作者" json:"writer"`    // 原作者
	Source    string    `gorm:"type:varchar(50);comment:来源（来源）" json:"source"` // 来源
	Comment   int       `gorm:"default:0;comment:评论数" json:"comment"`          // 评论数
	Content   string    `gorm:"type:text;comment:内容" json:"content"`
	Status    int       `gorm:"default:1;comment:状态" json:"status"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (SiteArticle) TableName() string { return "site_articles" }

type SiteArticleComment struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	ArticleID uint      `gorm:"index;comment:文章ID" json:"article_id"`
	UserID    uint      `gorm:"index;comment:用户ID" json:"user_id"`
	Content   string    `gorm:"type:varchar(300);comment:内容" json:"content"`
	Status    int       `gorm:"default:1;comment:状态" json:"status"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (SiteArticleComment) TableName() string { return "site_article_comments" }

// ============ 商店（诺哈 wap_shop / wap_shop_trade / wap_shop_order / wap_shop_comment，道具买卖） ============

type ShopCategory struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	Name      string    `gorm:"type:varchar(30);uniqueIndex;comment:名称" json:"name"`
	Sort      int       `gorm:"default:0;comment:排序值" json:"sort"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (ShopCategory) TableName() string { return "shop_categories" }

// Shop 店铺（一人一店）
type Shop struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint      `gorm:"uniqueIndex;comment:用户ID" json:"user_id"`
	Name      string    `gorm:"type:varchar(30);comment:名称" json:"name"`
	Status    int       `gorm:"default:1;comment:状态" json:"status"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (Shop) TableName() string { return "shops" }

// ShopGoods 商品（BidMT 交易模式：1=一口价）
type ShopGoods struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint      `gorm:"index;comment:店主=卖家" json:"user_id"` // 店主=卖家
	Name      string    `gorm:"type:varchar(60);comment:名称" json:"name"`
	CatID     uint      `gorm:"default:0;comment:分类ID" json:"cat_id"`
	BidMT     int       `gorm:"default:1;comment:1=一口价 2=求购" json:"bid_mt"` // 1=一口价 2=求购
	Money     int       `gorm:"default:0;comment:币种 0=G币" json:"money"`     // 币种 0=G币
	Price     int       `gorm:"comment:价格" json:"price"`
	Amount    int       `gorm:"default:1;comment:数量（库存）" json:"amount"` // 库存
	Sales     int       `gorm:"default:0;comment:销量（销量）" json:"sales"`  // 销量
	Intro     string    `gorm:"type:varchar(500);comment:简介" json:"intro"`
	Image     string    `gorm:"type:varchar(100);comment:图片" json:"image"`
	Clicks    int       `gorm:"default:0;comment:点击次数" json:"clicks"`
	Comment   int       `gorm:"default:0;comment:评论" json:"comment"`
	Status    int       `gorm:"default:1;comment:1上架 0下架" json:"status"` // 1上架 0下架
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (ShopGoods) TableName() string { return "shop_goods" }

// ShopOrder 订单 Status: 1待付款 2待发货 3待收货 4完成 5取消
type ShopOrder struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	GoodsID   uint      `gorm:"index;comment:商品ID" json:"goods_id"`
	SellerID  uint      `gorm:"index;comment:卖家ID" json:"seller_id"`
	BuyerID   uint      `gorm:"index;comment:买家ID" json:"buyer_id"`
	GoodsName string    `gorm:"type:varchar(60);comment:商品名称" json:"goods_name"`
	Money     int       `gorm:"default:0;comment:银两" json:"money"`
	Price     int       `gorm:"comment:价格" json:"price"`
	Amount    int       `gorm:"default:1;comment:数量" json:"amount"`
	Status    int       `gorm:"default:1;comment:状态" json:"status"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (ShopOrder) TableName() string { return "shop_orders" }

// ShopComment 商品评价 DType: 1=好评 2=中评 3=差评
type ShopComment struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	GoodsID   uint      `gorm:"index;comment:商品ID" json:"goods_id"`
	OrderID   uint      `gorm:"default:0;comment:订单ID" json:"order_id"`
	UserID    uint      `gorm:"index;comment:用户ID" json:"user_id"`
	DType     int       `gorm:"default:1;comment:D类型" json:"d_type"`
	Content   string    `gorm:"type:varchar(300);comment:内容" json:"content"`
	Reply     string    `gorm:"type:varchar(300);comment:店主回复" json:"reply"` // 店主回复
	Status    int       `gorm:"default:1;comment:状态" json:"status"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (ShopComment) TableName() string { return "shop_comments" }

// ArticleComment 空间日志评论（诺哈 wap_blog_article_comment）
type ArticleComment struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	ArticleID uint      `gorm:"index;comment:文章ID" json:"article_id"`
	UserID    uint      `gorm:"index;comment:用户ID" json:"user_id"`
	Content   string    `gorm:"type:varchar(300);comment:内容" json:"content"`
	Status    int       `gorm:"default:1;comment:状态" json:"status"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (ArticleComment) TableName() string { return "article_comments" }
