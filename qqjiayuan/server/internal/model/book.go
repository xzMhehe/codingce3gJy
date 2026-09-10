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

// BookChapter 章节（对齐诺哈 wap/book/chapter_list.asp + admin/book/chapter_*）
type BookChapter struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	BookID    uint      `gorm:"index" json:"book_id"`
	Title     string    `gorm:"type:varchar(60)" json:"title"`
	Content   string    `gorm:"type:text" json:"content"`
	VIP       int       `gorm:"default:0" json:"vip"` // 1=VIP章节
	Sort      int       `gorm:"default:0" json:"sort"`
	CreatedAt time.Time `json:"created_at"`
}

func (BookChapter) TableName() string { return "book_chapters" }

// BookComment 书评（对齐诺哈 wap/book/comment_list.asp，1-5星评分）
type BookComment struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	BookID    uint      `gorm:"index" json:"book_id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Score     int       `gorm:"default:5" json:"score"`
	Content   string    `gorm:"type:varchar(300)" json:"content"`
	Status    int       `gorm:"default:1" json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func (BookComment) TableName() string { return "book_comments" }

// BookShelf 书架（对齐诺哈 wap/book/shelf_add_ok.asp）
type BookShelf struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"uniqueIndex:uk_shelf" json:"user_id"`
	BookID    uint      `gorm:"uniqueIndex:uk_shelf" json:"book_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (BookShelf) TableName() string { return "book_shelves" }

// 帖子收藏
type ThreadFavorite struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"uniqueIndex:uk_fav_user" json:"user_id"`
	ThreadID  uint      `gorm:"uniqueIndex:uk_fav_user" json:"thread_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (ThreadFavorite) TableName() string { return "thread_favorites" }
