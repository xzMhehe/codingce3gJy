package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type ServerConfig struct {
	Port        int    `yaml:"port"`
	WebDir      string `yaml:"web_dir"`
	AdminWebDir string `yaml:"admin_web_dir"`
}

type MysqlConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
}

type JwtConfig struct {
	Secret      string `yaml:"secret"`
	ExpireHours int    `yaml:"expire_hours"`
}

// SeedConfig：Seed.Skip 用于「这台节点连接的是已被别的实例灌好数据的共享库」时，
// 跳过启动时的全量初始化(seed)，避免对已填充的表重复 INSERT / 跨 WAN 挂起。
type SeedConfig struct {
	Skip bool `yaml:"skip"`
}

// Config 服务端配置。
//
// ★★ 2026-10-10 双数据源（多数据源）：
//
//	Mysql     = 家园库（qq_jiayuan）：账号/角色/权限/私信/系统设置 + 其它小游戏
//	EzfyMysql = 二战征途库（qq_ezzt）：全部 ezfy_* 表
//
// 二战只通过「家园号」(users.id) 与家园关联：账号、昵称、私信、系统设置仍由家园库
// 唯一持有（二战经 EzfyHandler.HomeDB 读），不复制、不需要同步。
// EzfyMysql 留空 = 退化为单库模式（二战表与家园同库），老部署不受影响。
type Config struct {
	Server    ServerConfig `yaml:"server"`
	Mysql     MysqlConfig  `yaml:"mysql"`
	EzfyMysql MysqlConfig  `yaml:"ezfy_mysql"`
	Jwt       JwtConfig    `yaml:"jwt"`
	Seed      SeedConfig   `yaml:"seed"`
}

// EzfyDBConfig 返回二战库配置；未配置（DBName 为空）时回落家园库（单库模式）。
func (c *Config) EzfyDBConfig() MysqlConfig {
	if c.EzfyMysql.DBName == "" {
		return c.Mysql
	}
	m := c.EzfyMysql
	// 只写 dbname、其余留空时，沿用家园库的 host/port/user/password（同实例换库）
	if m.Host == "" {
		m.Host = c.Mysql.Host
	}
	if m.Port == 0 {
		m.Port = c.Mysql.Port
	}
	if m.User == "" {
		m.User = c.Mysql.User
	}
	if m.Password == "" {
		m.Password = c.Mysql.Password
	}
	return m
}

// EzfySplit 二战是否使用独立库。
func (c *Config) EzfySplit() bool {
	return c.EzfyMysql.DBName != "" && c.EzfyMysql.DBName != c.Mysql.DBName
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Jwt.ExpireHours == 0 {
		cfg.Jwt.ExpireHours = 168
	}
	return cfg, nil
}

func (m *MysqlConfig) DSN() string {
	return m.User + ":" + m.Password + "@tcp(" + m.Host + ":" + itoa(m.Port) + ")/" +
		m.DBName + "?charset=utf8mb4&parseTime=True&loc=Local&timeout=10s&readTimeout=30s&writeTimeout=30s"
}

// DSNWithoutDB 不带库名的连接串。
// 建库前目标库还不存在，必须用这个连到 MySQL 实例本身。
func (m *MysqlConfig) DSNWithoutDB() string {
	return m.User + ":" + m.Password + "@tcp(" + m.Host + ":" + itoa(m.Port) + ")/" +
		"?charset=utf8mb4&parseTime=True&loc=Local&timeout=10s&readTimeout=30s&writeTimeout=30s"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
