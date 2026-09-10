package model

import "time"

// 社区银行存款账户
type BankAccount struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	UserID         uint       `gorm:"uniqueIndex" json:"user_id"`
	Balance        int        `gorm:"default:0" json:"balance"`
	LastInterestAt *time.Time `json:"last_interest_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (BankAccount) TableName() string { return "bank_accounts" }

// 打工记录（用于统计每日打工次数）
type WorkRecord struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (WorkRecord) TableName() string { return "work_records" }

// 慈善基金捐款记录
type Donation struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Amount    int       `json:"amount"`
	CreatedAt time.Time `json:"created_at"`
}

func (Donation) TableName() string { return "donations" }

// 钱包收支流水（钱包页明细）
type WalletLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Kind      string    `gorm:"type:varchar(20)" json:"kind"` // post/reply/sign/work/buy/tip/dig/bank/charity/lottery
	Title     string    `gorm:"type:varchar(60)" json:"title"`
	Currency  string    `gorm:"type:varchar(10);default:coins" json:"currency"` // coins/yuanbao/jinzuan/youquan
	Delta     int       `json:"delta"`
	Remark    string    `gorm:"type:varchar(50);default:''" json:"remark"`
	CreatedAt time.Time `json:"created_at"`
}

func (WalletLog) TableName() string { return "wallet_logs" }

// 货币商店（复刻诺哈 wap_money_shop）：花 ptype 货币买 mtype 货币礼包
type MoneyShop struct {
	ID      uint      `gorm:"primaryKey" json:"id"`
	Name    string    `gorm:"type:varchar(40)" json:"name"`
	MType   string    `gorm:"column:mtype;type:varchar(10)" json:"mtype"` // 卖出货币 coins/yuanbao/jinzuan/youquan
	Money   int       `gorm:"default:0" json:"money"`                     // 每份卖出数量
	PType   string    `gorm:"column:ptype;type:varchar(10)" json:"ptype"` // 支付货币
	Price   int       `gorm:"default:0" json:"price"`                     // 每份单价
	Stock   int       `gorm:"default:0" json:"stock"`                     // 库存数量
	Sales   int       `gorm:"default:0" json:"sales"`                     // 销售数量
	Status  int       `gorm:"default:1" json:"status"`                    // 1上架 0下架
	AddTime time.Time `json:"add_time"`                                   // 销售时间
	EndTime time.Time `json:"end_time"`                                   // 结束时间
}

func (MoneyShop) TableName() string { return "money_shop" }

// 婚恋：婚姻证书（对齐诺哈 wap_marriage_marry：aid 求婚方 / bid 被求婚方 / status 1求婚中 0已婚）
type Marriage struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	Aid       uint       `gorm:"index" json:"aid"`         // 求婚方
	Bid       uint       `gorm:"index" json:"bid"`         // 被求婚方
	Status    int        `gorm:"default:1" json:"status"`  // 1求婚中 0已婚
	Message   string     `gorm:"type:varchar(200)" json:"message"` // 表白语
	CreatedAt time.Time  `json:"created_at"`
	EndTime   *time.Time `json:"end_time"` // 结婚时间
}

func (Marriage) TableName() string { return "marriages" }
