package model

import "time"

// 我的游戏（用户添加到"正在玩"，对齐诺哈 wap_game_bag：sort 排序）
type MyGame struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"uniqueIndex:uk_mygame" json:"user_id"`
	GameID    uint      `gorm:"uniqueIndex:uk_mygame" json:"game_id"`
	Sort      int       `gorm:"default:0" json:"sort"`
	CreatedAt time.Time `json:"created_at"`
	Game      *Game     `gorm:"foreignKey:GameID" json:"game,omitempty"`
}

func (MyGame) TableName() string { return "my_games" }

// 游戏大厅条目（对齐诺哈 wap_game：intro 简介 / gpath 游戏路径）
// Logo 为 static/image 下文件名，空则前端渲染文字标；Path 为本站路由（如 /games/garden），Url 为外站地址
type Game struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Name     string `gorm:"type:varchar(30)" json:"name"`
	Category string `gorm:"type:varchar(10)" json:"category"` // net网络游戏 / com社区游戏
	Logo     string `gorm:"type:varchar(50)" json:"logo"`
	Stars    string `gorm:"type:varchar(10)" json:"stars"`
	Desc     string `gorm:"type:varchar(100)" json:"desc"`
	Intro    string `gorm:"type:varchar(200)" json:"intro"`   // 游戏简介（诺哈 wap_game.intro）
	Path     string `gorm:"type:varchar(100)" json:"path"`    // 站内入口（诺哈 wap_game.gpath），空=未开发
	Url      string `gorm:"type:varchar(200)" json:"url"`     // 外站游戏网址
	BoardID  uint   `json:"board_id"` // 对应游戏论坛板块
	Sort     int    `json:"sort"`
	Status   int    `gorm:"default:1" json:"status"`
}

func (Game) TableName() string { return "games" }
