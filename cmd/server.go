package main

import (
	"context"
	"github.com/gin-gonic/gin"
	"horserun/config"
	"horserun/wire"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("加载配置失败：%v", err)
	}

	// 使用Wire进行依赖注入
	handler, err := wire.NewApp()
	if err != nil {
		log.Fatalf("初始化应用失败：%v", err)
	}

	// 公开接口路由器 - 监听公网端口
	publicRouter := gin.Default()
	publicRouter.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 仅暴露验证接口
	publicGroup := publicRouter.Group("/api/v1/authcode")
	{
		publicGroup.GET("/validate/:code", handler.Validate)
	}

	// 内部接口路由器 - 只监听本地
	internalRouter := gin.Default()
	internalGroup := internalRouter.Group("/api/v1/internal/authcode")
	{
		internalGroup.POST("/generate", handler.Generate)
		internalGroup.POST("/activate", handler.Activate)
		internalGroup.GET("/:code", handler.GetCode)
		internalGroup.GET("/list", handler.ListCodes)
		internalGroup.DELETE("/:code", handler.DeleteCode)
	}

	// 启动公开服务（监听所有网卡）
	publicAddr := cfg.Server.PublicAddr
	publicServer := &http.Server{
		Addr:         publicAddr,
		Handler:      publicRouter,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// 启动内部服务（仅监听本地回环地址）
	internalAddr := cfg.Server.InternalAddr
	internalServer := &http.Server{
		Addr:         internalAddr,
		Handler:      internalRouter,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Starting PUBLIC server on %s", publicAddr)
		log.Println("Public API: /api/v1/authcode/validate/:code")
		if err := publicServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Public server failed: %v", err)
		}
	}()

	go func() {
		log.Printf("Starting INTERNAL server on %s", internalAddr)
		log.Println("Internal API: Only accessible from localhost")
		if err := internalServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Internal server failed: %v", err)
		}
	}()

	log.Println("Authorization code server started")
	<-quit
	log.Println("Shutting down servers...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := publicServer.Shutdown(ctx); err != nil {
		log.Printf("Public server shutdown error: %v", err)
	}
	if err := internalServer.Shutdown(ctx); err != nil {
		log.Printf("Internal server shutdown error: %v", err)
	}

	log.Println("Servers stopped")
}
