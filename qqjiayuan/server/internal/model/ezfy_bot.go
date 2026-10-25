package model

import "time"

// ============ 人机（AI 玩家）模块（★ 2026-10-17）============
//
// 设计见项目根 `人机模块方案.md`：人机 = 一个**有状态的玩家代理**（会发育、会积累、会宣战），
// 走**和真人完全相同的业务接口**（内部 HTTP 自调用），不写第二套玩法。
//
// 标识口径（三处，写入点唯一 = ezfyBotProvision）：
//   - 家园库 `users.is_bot`      账号级**权威源**
//   - `ezfy_profile.is_bot`      二战侧**权威源**（所有「是不是人机」判定都看它）
//   - `ezfy_city.is_bot`         供地图渲染 / 目标筛选 / 城市统计**直接查 city 表**时过滤的冗余副本
//
// ⚠️ 新增表已同时登记到：
//   `seed/ezfy_run.go::ezfyMigrateModels`（建表）
//   `handler/ezfy/ezfy_admin_purge.go::ezfyPurgeByUid`（删号/清空人机，漏表护栏测试会校验）

// EzfyBot 人机档案（与 ezfy_profile 一对一）
type EzfyBot struct {
	ID      uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID  uint   `gorm:"uniqueIndex:uk_ezfy_bot_uid;comment:玩家ID" json:"user_id"`
	Level   int    `gorm:"default:2;comment:难度档 1新手陪练 2普通 3精英" json:"level"`
	Persona string `gorm:"type:varchar(20);comment:性格(决定节奏/侵略性)" json:"persona"`
	Enabled int    `gorm:"default:1;comment:是否启用(0=暂停行动)" json:"enabled"`

	// 调度：next_act_at <= now 才轮到它行动；每次行动后加随机抖动，模拟真人节奏
	NextActAt  int64  `gorm:"default:0;index:idx_bot_next_act;comment:下次行动时间(ms)" json:"next_act_at"`
	LastActAt  int64  `gorm:"default:0;comment:上次行动时间(ms)" json:"last_act_at"`
	LastAction string `gorm:"type:varchar(30);comment:上次动作" json:"last_action"`

	// 攻击节流：每日出征次数（跨天自动清零）
	AtkCountToday int    `gorm:"default:0;comment:今日出征次数" json:"atk_count_today"`
	AtkDate       string `gorm:"type:varchar(10);comment:今日日期 yyyymmdd" json:"atk_date"`
	// ★ 2026-10-17 出征设置（0=按难度默认）：频率 次/小时（简单6 普通8 精英10）、单次出兵（30/35/40万）
	AtkPerHour int   `gorm:"default:0;comment:攻击频率(次/小时)，0=按难度默认" json:"atk_per_hour"`
	AtkTroop   int64 `gorm:"default:0;comment:单次出征兵力，0=按难度默认" json:"atk_troop"`
	LastAtkAt  int64 `gorm:"default:0;comment:上次出征时间(ms)，按小时频率节流" json:"last_atk_at"`
	// ★ 2026-10-17 出征类型：0=默认征服(3)；1=侦察 2=掠夺 3=征服（管理端可调整）
	AtkType int `gorm:"default:0;comment:出征类型 0=默认征服 1=侦察 2=掠夺 3=征服" json:"atk_type"`
	// ★ 2026-10-17 世界聊天（机器人也能说话）：每天句子不重复，跨天清空
	ChatDate string `gorm:"type:varchar(10);comment:世界发言日期 yyyymmdd" json:"chat_date"`
	ChatUsed string `gorm:"type:text;comment:今日已用发言句下标(逗号分隔)" json:"chat_used"`
	// ★ 2026-10-19 自动发言开关（管理端「编辑人机-基本信息」可调）：0=不自动发言
	ChatOn int `gorm:"default:1;comment:是否自动发言(0=关闭,1=开启)" json:"chat_on"`
	// ★ 2026-10-25 打野地开关（管理端「编辑人机-基本信息」可调）：0=不打玩家附属野地（默认）1=打
	//
	//	开启后：① 宣战等待期先征服目标玩家的附属野地；② 周围没有可打的城市时兜底打玩家野地。
	//	关闭（默认）：人机只打城市，绝不碰玩家野地（用户口径：默认不打玩家野地）。
	AttackWild int `gorm:"default:0;comment:是否攻击玩家附属野地(0=不打,1=打)" json:"attack_wild"`

	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (EzfyBot) TableName() string { return "ezfy_bot" }

// EzfyBotTarget 人机攻击计划
//
// ★ 硬约束：打玩家城必须先宣战、宣战 24h 才生效、有效期 48h（ezfyWarDelayHours / ezfyWarDurationHours）
// → 人机**不能「看到谁就打谁」**，必须提前规划，用这张表驱动时序。
//
// 状态机：0 待宣战 → 1 待生效(已宣战) → 2 可出击(已生效) → 3 冷却
// (bot_uid, target_uid) 唯一：同一人机对同一目标只保留一条计划。
type EzfyBotTarget struct {
	ID           uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	BotUid       uint   `gorm:"uniqueIndex:uk_bot_target;comment:人机玩家ID" json:"bot_uid"`
	TargetUid    uint   `gorm:"uniqueIndex:uk_bot_target;index:idx_bot_target_uid;comment:目标玩家ID" json:"target_uid"`
	TargetCityId int64  `gorm:"default:0;comment:目标城市ID" json:"target_city_id"`
	Status       int    `gorm:"default:0;index:idx_bot_target_status;comment:0待宣战 1待生效 2可出击 3冷却" json:"status"`
	EffectTime   int64  `gorm:"default:0;comment:宣战生效时间(ms)" json:"effect_time"`
	CoolUntil    int64  `gorm:"default:0;comment:冷却截止(ms)" json:"cool_until"`
	Note         string `gorm:"type:varchar(100);comment:备注(目标快照)" json:"note"`

	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (EzfyBotTarget) TableName() string { return "ezfy_bot_target" }

// EzfyBotLog 人机行为日志（排障 + 管理端展示「为什么没动手」）
type EzfyBotLog struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	BotUid    uint      `gorm:"index:idx_bot_log_uid;comment:人机玩家ID" json:"bot_uid"`
	Action    string    `gorm:"type:varchar(30);comment:动作" json:"action"`
	Detail    string    `gorm:"type:varchar(255);comment:详情" json:"detail"`
	Result    int       `gorm:"default:1;comment:1成功 0失败/跳过" json:"result"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (EzfyBotLog) TableName() string { return "ezfy_bot_log" }

// EzfyBotLock 人机 tick 的**租约锁**（多进程互斥）
//
// ★ 项目是「两个服务进程共享一个 RDS」→ 任何 for range time.Tick 都会在两个进程里各跑一遍。
// 不加锁 → 每个动作执行两次（重复造兵、重复出征、资源翻倍）。
// 用**带条件的原子 UPDATE** 抢租约，不引入 Redis：
//
//	UPDATE ezfy_bot_lock SET owner=?, expire_at=? WHERE name=? AND expire_at < ?
//
// RowsAffected = 1 才算抢到；租约到期自动可被别的进程接管（进程崩了不会死锁）。
type EzfyBotLock struct {
	Name      string    `gorm:"primaryKey;type:varchar(30);comment:锁名" json:"name"`
	Owner     string    `gorm:"type:varchar(64);comment:持有者(host:pid)" json:"owner"`
	ExpireAt  int64     `gorm:"default:0;comment:租约到期时间(ms)" json:"expire_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (EzfyBotLock) TableName() string { return "ezfy_bot_lock" }
