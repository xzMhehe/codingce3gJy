package model

import "time"

// 书城书籍
type Book struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	Title     string    `gorm:"type:varchar(60);comment:标题" json:"title"`
	Author    string    `gorm:"type:varchar(30);comment:作者" json:"author"`
	Category  string    `gorm:"type:varchar(20);comment:武侠/言情/都市/灵异..." json:"category"` // 武侠/言情/都市/灵异...
	Intro     string    `gorm:"type:varchar(300);comment:简介" json:"intro"`
	Status    string    `gorm:"type:varchar(10);default:连载;comment:连载/完结" json:"status"` // 连载/完结
	Recommend int       `gorm:"default:0;comment:1=强力推荐" json:"recommend"`               // 1=强力推荐
	NewBook   int       `gorm:"default:0;comment:1=新书上架" json:"new_book"`                // 1=新书上架
	Rating    string    `gorm:"type:varchar(10);default:★★★★★;comment:评分" json:"rating"`
	Views     int       `gorm:"default:0;comment:浏览数" json:"views"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (Book) TableName() string { return "books" }

// BookChapter 章节（对齐诺哈 wap/book/chapter_list.asp + admin/book/chapter_*）
type BookChapter struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	BookID    uint      `gorm:"index;comment:书籍ID" json:"book_id"`
	Title     string    `gorm:"type:varchar(60);comment:标题" json:"title"`
	Content   string    `gorm:"type:text;comment:内容" json:"content"`
	VIP       int       `gorm:"default:0;comment:1=VIP章节" json:"vip"` // 1=VIP章节
	Sort      int       `gorm:"default:0;comment:排序值" json:"sort"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (BookChapter) TableName() string { return "book_chapters" }

// BookComment 书评（对齐诺哈 wap/book/comment_list.asp，1-5星评分）
type BookComment struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	BookID    uint      `gorm:"index;comment:书籍ID" json:"book_id"`
	UserID    uint      `gorm:"index;comment:用户ID" json:"user_id"`
	Score     int       `gorm:"default:5;comment:积分" json:"score"`
	Content   string    `gorm:"type:varchar(300);comment:内容" json:"content"`
	Status    int       `gorm:"default:1;comment:状态" json:"status"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (BookComment) TableName() string { return "book_comments" }

// BookShelf 书架（对齐诺哈 wap/book/shelf_add_ok.asp）
type BookShelf struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint      `gorm:"uniqueIndex:uk_shelf;comment:用户ID" json:"user_id"`
	BookID    uint      `gorm:"uniqueIndex:uk_shelf;comment:书籍ID" json:"book_id"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (BookShelf) TableName() string { return "book_shelves" }

// 帖子收藏
type ThreadFavorite struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint      `gorm:"uniqueIndex:uk_fav_user;comment:用户ID" json:"user_id"`
	ThreadID  uint      `gorm:"uniqueIndex:uk_fav_user;comment:帖子ID" json:"thread_id"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (ThreadFavorite) TableName() string { return "thread_favorites" }
