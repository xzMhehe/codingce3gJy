package model

import "time"

// 我的游戏（用户添加到"正在玩"）
type MyGame struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"uniqueIndex:uk_mygame" json:"user_id"`
	GameID    uint      `gorm:"uniqueIndex:uk_mygame" json:"game_id"`
	CreatedAt time.Time `json:"created_at"`
	Game      *Game     `gorm:"foreignKey:GameID" json:"game,omitempty"`
}

func (MyGame) TableName() string { return "my_games" }

// 游戏大厅条目：Logo 为 static/image 下文件名，空则前端渲染文字标
type Game struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Name     string `gorm:"type:varchar(30)" json:"name"`
	Category string `gorm:"type:varchar(10)" json:"category"` // net网络游戏 / com社区游戏
	Logo     string `gorm:"type:varchar(50)" json:"logo"`
	Stars    string `gorm:"type:varchar(10)" json:"stars"`
	Desc     string `gorm:"type:varchar(100)" json:"desc"`
	Url      string `gorm:"type:varchar(200)" json:"url"` // 游戏网址，未开发为空
	BoardID  uint   `json:"board_id"` // 对应游戏论坛板块
	Sort     int    `json:"sort"`
	Status   int    `gorm:"default:1" json:"status"`
}

func (Game) TableName() string { return "games" }
