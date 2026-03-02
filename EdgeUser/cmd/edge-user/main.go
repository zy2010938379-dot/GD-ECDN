package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/TeaOSLab/EdgeUser/internal/api"
	"github.com/gin-gonic/gin"
)

func main() {
	// 创建用户控制器
	userController, err := api.NewUserController()
	if err != nil {
		log.Fatalf("Failed to create user controller: %v", err)
	}
	defer userController.Close()

	// 设置Gin模式
	gin.SetMode(gin.ReleaseMode)

	// 创建Gin路由
	router := gin.Default()

	// 添加中间件
	router.Use(corsMiddleware())
	router.Use(loggingMiddleware())

	// API路由组
	apiGroup := router.Group("/api/v1")
	{
		// 用户管理路由
		userGroup := apiGroup.Group("/users")
		{
			userGroup.POST("/login", userController.Login)
			userGroup.POST("/register", userController.Register)
			userGroup.GET("/:userId", userController.GetUserInfo)
			userGroup.PUT("/info", userController.UpdateUserInfo)
			userGroup.GET("/dashboard/:userId", userController.GetUserDashboard)
			userGroup.GET("/features/:userId", userController.GetUserFeatures)
			userGroup.PUT("/servers-state/:userId", userController.RenewUserServersState)

			// 访问密钥管理路由
			accessKeyGroup := userGroup.Group("/access-keys")
			{
				accessKeyGroup.POST("", userController.CreateAccessKey)
				accessKeyGroup.GET("", userController.ListAccessKeys)
				accessKeyGroup.DELETE("/:accessKeyId", userController.DeleteAccessKey)
				accessKeyGroup.PUT("/:accessKeyId/status", userController.UpdateAccessKeyStatus)
			}

			// 身份认证路由
			identityGroup := userGroup.Group("/identity")
			{
				identityGroup.POST("", userController.CreateUserIdentity)
				identityGroup.GET("/:userId", userController.GetUserIdentity)
				identityGroup.PUT("/:identityId", userController.UpdateUserIdentity)
				identityGroup.POST("/:identityId/submit", userController.SubmitUserIdentity)
			}
		}

		// 服务器管理路由
		serverGroup := apiGroup.Group("/servers")
		{
			serverGroup.GET("/user/:userId", userController.ListUserServers)
			serverGroup.GET("/:serverId", userController.GetServerInfo)
			serverGroup.POST("", userController.CreateServer)
			serverGroup.PUT("/:serverId", userController.UpdateServer)
			serverGroup.DELETE("/:serverId", userController.DeleteServer)
		}

		// 健康检查路由
		apiGroup.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":    "healthy",
				"module":    "EdgeUser",
				"timestamp": "2024-01-01T00:00:00Z",
				"timestamp": time.Now().Format(time.RFC3339),
			})
		})
	}

	// 启动HTTP服务器
	server := &http.Server{
		Addr:    ":8081",
		Handler: router,
	}

	// 优雅关闭
	go func() {
		log.Println("EdgeUser服务启动，监听端口: 8081")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("正在关闭EdgeUser服务...")

	// 优雅关闭服务器
	if err := server.Shutdown(nil); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("EdgeUser服务已关闭")
}

// corsMiddleware CORS中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// loggingMiddleware 日志中间件
func loggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf("[%s] %s %s", c.Request.Method, c.Request.URL.Path, c.ClientIP())
		c.Next()
	}
}
