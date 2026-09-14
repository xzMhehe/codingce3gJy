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

type Config struct {
	Server ServerConfig `yaml:"server"`
	Mysql  MysqlConfig  `yaml:"mysql"`
	Jwt    JwtConfig    `yaml:"jwt"`
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
		m.DBName + "?charset=utf8mb4&parseTime=True&loc=Local"
}

// DSNWithoutDB 不带库名的连接串。
// 建库前目标库还不存在，必须用这个连到 MySQL 实例本身。
func (m *MysqlConfig) DSNWithoutDB() string {
	return m.User + ":" + m.Password + "@tcp(" + m.Host + ":" + itoa(m.Port) + ")/" +
		"?charset=utf8mb4&parseTime=True&loc=Local"
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
