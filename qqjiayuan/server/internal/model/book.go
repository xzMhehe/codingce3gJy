package model

import "time"

// 书城书籍
type Book struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Title     string    `gorm:"type:varchar(60)" json:"title"`
	Author    string    `gorm:"type:varchar(30)" json:"author"`
	Category  string    `gorm:"type:varchar(20)" json:"category"` // 武侠/言情/都市/灵异...
	Intro     string    `gorm:"type:varchar(300)" json:"intro"`
	Status    string    `gorm:"type:varchar(10);default:连载" json:"status"` // 连载/完结
	Recommend int       `gorm:"default:0" json:"recommend"`                 // 1=强力推荐
	NewBook   int       `gorm:"default:0" json:"new_book"`                  // 1=新书上架
	Rating    string    `gorm:"type:varchar(10);default:★★★★★" json:"rating"`
	Views     int       `gorm:"default:0" json:"views"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Book) TableName() string { return "books" }

// 帖子收藏
type ThreadFavorite struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"uniqueIndex:uk_fav_user" json:"user_id"`
	ThreadID  uint      `gorm:"uniqueIndex:uk_fav_user" json:"thread_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (ThreadFavorite) TableName() string { return "thread_favorites" }
