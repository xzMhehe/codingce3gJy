package model

import "time"

// 社区银行存款账户
type BankAccount struct {
	ID             uint       `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID         uint       `gorm:"uniqueIndex;comment:用户ID" json:"user_id"`
	Balance        int        `gorm:"default:0;comment:余额" json:"balance"`
	LastInterestAt *time.Time `gorm:"comment:最后兴趣时间" json:"last_interest_at"`
	CreatedAt      time.Time  `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"comment:更新时间" json:"updated_at"`
}

func (BankAccount) TableName() string { return "bank_accounts" }

// 打工记录（用于统计每日打工次数）
type WorkRecord struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint      `gorm:"index;comment:用户ID" json:"user_id"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (WorkRecord) TableName() string { return "work_records" }

// 慈善基金捐款记录
type Donation struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint      `gorm:"index;comment:用户ID" json:"user_id"`
	Amount    int       `gorm:"comment:数量" json:"amount"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (Donation) TableName() string { return "donations" }

// 钱包收支流水（钱包页明细）
type WalletLog struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint      `gorm:"index;comment:用户ID" json:"user_id"`
	Kind      string    `gorm:"type:varchar(20);comment:post/reply/sign/work/buy/tip/dig/bank/charity/lottery" json:"kind"` // post/reply/sign/work/buy/tip/dig/bank/charity/lottery
	Title     string    `gorm:"type:varchar(60);comment:标题" json:"title"`
	Currency  string    `gorm:"type:varchar(10);default:coins;comment:coins/yuanbao/jinzuan/youquan" json:"currency"` // coins/yuanbao/jinzuan/youquan
	Delta     int       `gorm:"comment:增量" json:"delta"`
	Remark    string    `gorm:"type:varchar(50);default:'';comment:备注" json:"remark"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (WalletLog) TableName() string { return "wallet_logs" }

// 货币商店（复刻诺哈 wap_money_shop）：花 ptype 货币买 mtype 货币礼包
type MoneyShop struct {
	ID      uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	Name    string    `gorm:"type:varchar(40);comment:名称" json:"name"`
	MType   string    `gorm:"column:mtype;type:varchar(10);comment:卖出货币 coins/yuanbao/jinzuan/youquan" json:"mtype"` // 卖出货币 coins/yuanbao/jinzuan/youquan
	Money   int       `gorm:"default:0;comment:每份卖出数量" json:"money"`                                                 // 每份卖出数量
	PType   string    `gorm:"column:ptype;type:varchar(10);comment:支付货币" json:"ptype"`                               // 支付货币
	Price   int       `gorm:"default:0;comment:每份单价" json:"price"`                                                   // 每份单价
	Stock   int       `gorm:"default:0;comment:库存数量" json:"stock"`                                                   // 库存数量
	Sales   int       `gorm:"default:0;comment:销售数量" json:"sales"`                                                   // 销售数量
	Status  int       `gorm:"default:1;comment:1上架 0下架" json:"status"`                                               // 1上架 0下架
	AddTime time.Time `gorm:"comment:销售时间" json:"add_time"`                                                          // 销售时间
	EndTime time.Time `gorm:"comment:结束时间" json:"end_time"`                                                          // 结束时间
}

func (MoneyShop) TableName() string { return "money_shop" }

// 婚恋：婚姻证书（对齐诺哈 wap_marriage_marry：aid 求婚方 / bid 被求婚方 / status 1求婚中 0已婚）
type Marriage struct {
	ID        uint       `gorm:"primaryKey;comment:主键ID" json:"id"`
	Aid       uint       `gorm:"index;comment:求婚方" json:"aid"`                 // 求婚方
	Bid       uint       `gorm:"index;comment:被求婚方" json:"bid"`                // 被求婚方
	Status    int        `gorm:"default:1;comment:1求婚中 0已婚" json:"status"`     // 1求婚中 0已婚
	Message   string     `gorm:"type:varchar(200);comment:表白语" json:"message"` // 表白语
	CreatedAt time.Time  `gorm:"comment:创建时间" json:"created_at"`
	EndTime   *time.Time `gorm:"comment:结婚时间" json:"end_time"` // 结婚时间
}

func (Marriage) TableName() string { return "marriages" }
