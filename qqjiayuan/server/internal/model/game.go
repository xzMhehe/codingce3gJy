package model

import "time"

// 我的游戏（用户添加到"正在玩"，对齐诺哈 wap_game_bag：sort 排序）
type MyGame struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint      `gorm:"uniqueIndex:uk_mygame;comment:用户ID" json:"user_id"`
	GameID    uint      `gorm:"uniqueIndex:uk_mygame;comment:游戏ID" json:"game_id"`
	Sort      int       `gorm:"default:0;comment:排序值" json:"sort"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	Game      *Game     `gorm:"foreignKey:GameID" json:"game,omitempty"`
}

func (MyGame) TableName() string { return "my_games" }

// 游戏大厅条目（对齐诺哈 wap_game：intro 简介 / gpath 游戏路径）
// Logo 为 static/image 下文件名，空则前端渲染文字标；Path 为本站路由（如 /games/garden），Url 为外站地址
type Game struct {
	ID       uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	Name     string `gorm:"type:varchar(30);comment:名称" json:"name"`
	Category string `gorm:"type:varchar(10);comment:net网络游戏 / com社区游戏" json:"category"` // net网络游戏 / com社区游戏
	Logo     string `gorm:"type:varchar(50);comment:Logo" json:"logo"`
	Stars    string `gorm:"type:varchar(10);comment:Stars" json:"stars"`
	Desc     string `gorm:"type:varchar(100);comment:描述" json:"desc"`
	Intro    string `gorm:"type:varchar(200);comment:游戏简介（诺哈 wap_game.intro）" json:"intro"`      // 游戏简介（诺哈 wap_game.intro）
	Path     string `gorm:"type:varchar(100);comment:站内入口（诺哈 wap_game.gpath），空=未开发" json:"path"` // 站内入口（诺哈 wap_game.gpath），空=未开发
	Url      string `gorm:"type:varchar(200);comment:外站游戏网址" json:"url"`                         // 外站游戏网址
	BoardID  uint   `gorm:"comment:对应游戏论坛板块" json:"board_id"`                                    // 对应游戏论坛板块
	Sort     int    `gorm:"comment:排序值" json:"sort"`
	Status   int    `gorm:"default:1;comment:状态" json:"status"`
}

func (Game) TableName() string { return "games" }
