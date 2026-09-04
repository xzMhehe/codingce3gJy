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
