package wire

import (
	"horserun/config"
	"horserun/internal/biz"
	"horserun/internal/di"
)

// Initialize 初始化应用
func Initialize() (*biz.Handler, error) {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, err
	}

	// 初始化数据库
	err = di.InitDB(&cfg.Database)
	if err != nil {
		return nil, err
	}

	// 自动迁移数据库模型
	err = di.AutoMigrate()
	if err != nil {
		return nil, err
	}

	// 创建授权码管理器
	manager := biz.NewManager()

	// 创建处理器
	handler := biz.NewHandler(manager)

	return handler, nil
}

// NewApp 创建应用
func NewApp() (*biz.Handler, error) {
	return Initialize()
}

