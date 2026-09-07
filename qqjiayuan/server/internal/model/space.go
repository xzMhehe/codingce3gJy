package model

import "time"

// Space 用户空间
type Space struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"uniqueIndex" json:"user_id"`
	Name      string    `gorm:"type:varchar(50)" json:"name"`       // 空间名称
	Signature string    `gorm:"type:varchar(200)" json:"signature"` // 空间签名
	Intro     string    `gorm:"type:varchar(500)" json:"intro"`     // 空间介绍
	Status    int       `gorm:"default:1" json:"status"`            // 1正常 0关闭
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Space) TableName() string { return "spaces" }

// Mood 心情说说
type Mood struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Content   string    `gorm:"type:varchar(500)" json:"content"`
	Status    int       `gorm:"default:1" json:"status"` // 1正常 0删除
	CreatedAt time.Time `json:"created_at"`
}

func (Mood) TableName() string { return "moods" }

// MoodComment 心情评论
type MoodComment struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	MoodID    uint      `gorm:"index" json:"mood_id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Content   string    `gorm:"type:varchar(300)" json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

func (MoodComment) TableName() string { return "mood_comments" }

// Article 日志
type Article struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Title     string    `gorm:"type:varchar(100)" json:"title"`
	CatID     uint      `gorm:"default:0" json:"cat_id"` // 日志分类（诺哈 wap_blog_article_category）
	Atype     int       `gorm:"default:0" json:"atype"`  // 0公开 1不公开（仅自己可见）
	Content   string    `gorm:"type:text" json:"content"`
	Status    int       `gorm:"default:1" json:"status"` // 1正常 0删除
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Article) TableName() string { return "articles" }

// Album 相册
type Album struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Name      string    `gorm:"type:varchar(50)" json:"name"`
	Cover     string    `gorm:"type:varchar(100)" json:"cover"` // 封面图
	Count     int       `gorm:"default:0" json:"count"`
	CreatedAt time.Time `json:"created_at"`
}

func (Album) TableName() string { return "albums" }

// Photo 相片
type Photo struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	AlbumID     uint      `gorm:"index" json:"album_id"`
	UserID      uint      `gorm:"index" json:"user_id"`
	File        string    `gorm:"type:varchar(100)" json:"file"`
	Format      string    `gorm:"type:varchar(10)" json:"format"` // jpg/gif/png
	SPath       string    `gorm:"type:varchar(200)" json:"s_path"` // 存储路径（诺哈 wap_blog_album.spath）
	PhotoBase64 string    `gorm:"type:longtext" json:"photo_base64,omitempty"` // base64 存储（data URI）
	Caption     string    `gorm:"type:varchar(100)" json:"caption"`
	Sizes       int       `gorm:"default:0" json:"sizes"` // 字节数
	Clicks      int       `gorm:"default:0" json:"clicks"`
	CreatedAt   time.Time `json:"created_at"`
}

func (Photo) TableName() string { return "photos" }

// SpaceMessage 空间留言
type SpaceMessage struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ToUserID   uint      `gorm:"index" json:"to_user_id"`
	FromUserID uint      `gorm:"index" json:"from_user_id"`
	Mtype      int       `gorm:"default:0" json:"mtype"` // 0公开 1悄悄话（仅主人和留言者可见）
	Content    string    `gorm:"type:varchar(300)" json:"content"`
	Status     int       `gorm:"default:1" json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

func (SpaceMessage) TableName() string { return "space_messages" }

// Visitor 访客记录
type Visitor struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	OwnerID   uint      `gorm:"index" json:"owner_id"`   // 空间主人
	UserID    uint      `gorm:"index" json:"user_id"`     // 访客
	CreatedAt time.Time `json:"created_at"`
}

func (Visitor) TableName() string { return "visitors" }
