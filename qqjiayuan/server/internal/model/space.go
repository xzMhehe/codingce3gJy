package model

import "time"

// Space 用户空间
type Space struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint      `gorm:"uniqueIndex;comment:用户ID" json:"user_id"`
	Name      string    `gorm:"type:varchar(50);comment:空间名称" json:"name"`       // 空间名称
	Signature string    `gorm:"type:varchar(200);comment:空间签名" json:"signature"` // 空间签名
	Intro     string    `gorm:"type:varchar(500);comment:空间介绍" json:"intro"`     // 空间介绍
	Status    int       `gorm:"default:1;comment:1正常 0关闭" json:"status"`         // 1正常 0关闭
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (Space) TableName() string { return "spaces" }

// Mood 心情说说
type Mood struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint      `gorm:"index;comment:用户ID" json:"user_id"`
	Content   string    `gorm:"type:varchar(500);comment:内容" json:"content"`
	Status    int       `gorm:"default:1;comment:1正常 0删除" json:"status"` // 1正常 0删除
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (Mood) TableName() string { return "moods" }

// MoodComment 心情评论
type MoodComment struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	MoodID    uint      `gorm:"index;comment:心情ID" json:"mood_id"`
	UserID    uint      `gorm:"index;comment:用户ID" json:"user_id"`
	Content   string    `gorm:"type:varchar(300);comment:内容" json:"content"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (MoodComment) TableName() string { return "mood_comments" }

// Article 日志
type Article struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint      `gorm:"index;comment:用户ID" json:"user_id"`
	Title     string    `gorm:"type:varchar(100);comment:标题" json:"title"`
	CatID     uint      `gorm:"default:0;comment:日志分类（诺哈 wap_blog_article_category）" json:"cat_id"` // 日志分类（诺哈 wap_blog_article_category）
	Atype     int       `gorm:"default:0;comment:0公开 1不公开（仅自己可见）" json:"atype"`                     // 0公开 1不公开（仅自己可见）
	Content   string    `gorm:"type:text;comment:内容" json:"content"`
	Status    int       `gorm:"default:1;comment:1正常 0删除" json:"status"` // 1正常 0删除
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (Article) TableName() string { return "articles" }

// Album 相册
type Album struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint      `gorm:"index;comment:用户ID" json:"user_id"`
	Name      string    `gorm:"type:varchar(50);comment:名称" json:"name"`
	Cover     string    `gorm:"type:varchar(100);comment:封面图" json:"cover"` // 封面图
	Count     int       `gorm:"default:0;comment:数量" json:"count"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (Album) TableName() string { return "albums" }

// Photo 相片
type Photo struct {
	ID          uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	AlbumID     uint      `gorm:"index;comment:相册ID" json:"album_id"`
	UserID      uint      `gorm:"index;comment:用户ID" json:"user_id"`
	File        string    `gorm:"type:varchar(100);comment:文件" json:"file"`
	Format      string    `gorm:"type:varchar(10);comment:jpg/gif/png" json:"format"`                      // jpg/gif/png
	SPath       string    `gorm:"type:varchar(200);comment:存储路径（诺哈 wap_blog_album.spath）" json:"s_path"`   // 存储路径（诺哈 wap_blog_album.spath）
	PhotoBase64 string    `gorm:"type:longtext;comment:base64 存储（data URI）" json:"photo_base64,omitempty"` // base64 存储（data URI）
	Caption     string    `gorm:"type:varchar(100);comment:Caption" json:"caption"`
	Sizes       int       `gorm:"default:0;comment:字节数" json:"sizes"` // 字节数
	Clicks      int       `gorm:"default:0;comment:点击次数" json:"clicks"`
	CreatedAt   time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (Photo) TableName() string { return "photos" }

// SpaceFile 空间文件（诺哈 blog/file：传文件）
type SpaceFile struct {
	ID         uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID     uint      `gorm:"index;comment:用户ID" json:"user_id"`
	Name       string    `gorm:"type:varchar(100);comment:名称" json:"name"`
	FileBase64 string    `gorm:"type:longtext;comment:base64 data URI" json:"file_base64,omitempty"` // base64 data URI
	Size       int       `gorm:"default:0;comment:大小" json:"size"`
	Clicks     int       `gorm:"default:0;comment:点击次数" json:"clicks"`
	CreatedAt  time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (SpaceFile) TableName() string { return "space_files" }

// SpaceMessage 空间留言
type SpaceMessage struct {
	ID         uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	ToUserID   uint      `gorm:"index;comment:接收用户ID" json:"to_user_id"`
	FromUserID uint      `gorm:"index;comment:来源用户ID" json:"from_user_id"`
	Mtype      int       `gorm:"default:0;comment:0公开 1悄悄话（仅主人和留言者可见）" json:"mtype"` // 0公开 1悄悄话（仅主人和留言者可见）
	Content    string    `gorm:"type:varchar(300);comment:内容" json:"content"`
	Status     int       `gorm:"default:1;comment:状态" json:"status"`
	CreatedAt  time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (SpaceMessage) TableName() string { return "space_messages" }

// Visitor 访客记录
type Visitor struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	OwnerID   uint      `gorm:"index;comment:空间主人" json:"owner_id"`    // 空间主人
	UserID    uint      `gorm:"index;comment:用户ID（访客）" json:"user_id"` // 访客
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (Visitor) TableName() string { return "visitors" }
