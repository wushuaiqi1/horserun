package config

import (
	"log"
	"time"

	"github.com/spf13/viper"
)

// Config 应用配置结构体
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	AuthCode AuthCodeConfig
}

// ServerConfig 服务器配置
type ServerConfig struct {
	PublicAddr   string
	InternalAddr string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	DSN             string
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
}

// AuthCodeConfig 授权码配置
type AuthCodeConfig struct {
	CodeLength int
}

// LoadConfig 加载配置
func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yml")
	viper.AddConfigPath("./")
	viper.AddConfigPath("./config")

	// 设置默认值
	viper.SetDefault("server.publicAddr", ":19001")
	viper.SetDefault("server.internalAddr", "127.0.0.1:19002")
	viper.SetDefault("server.readTimeout", "10s")
	viper.SetDefault("server.writeTimeout", "10s")

	viper.SetDefault("database.dsn", "root:password@tcp(127.0.0.1:3306)/horserun?charset=utf8mb4&parseTime=True&loc=Local")
	viper.SetDefault("database.maxIdleConns", 10)
	viper.SetDefault("database.maxOpenConns", 100)
	viper.SetDefault("database.connMaxLifetime", "1h")

	viper.SetDefault("authCode.codeLength", 16)

	// 读取配置文件
	err := viper.ReadInConfig()
	if err != nil {
		log.Printf("警告：无法读取配置文件，使用默认值: %v", err)
	}

	// 解析配置
	var config Config
	err = viper.Unmarshal(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
