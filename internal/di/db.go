package di

import (
	"horserun/config"
	"horserun/internal/model"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB 全局数据库连接
var DB *gorm.DB

// InitDB 初始化数据库连接
func InitDB(cfg *config.DatabaseConfig) error {
	// 解析DSN，提取数据库类型和连接信息
	dsn := cfg.DSN

	// 初始化数据库连接
	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return err
	}

	// 配置连接池
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}

	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	log.Println("数据库连接初始化成功")
	return nil
}

// AutoMigrate 自动迁移数据库模型
func AutoMigrate() error {
	// 在这里添加需要迁移的模型
	return DB.AutoMigrate(&model.AuthCode{})
}
