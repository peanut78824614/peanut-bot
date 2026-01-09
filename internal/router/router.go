package router

import (
	"context"
	"data/internal/controller"
	"data/internal/middleware"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

func InitRouter(ctx context.Context) {
	s := g.Server()

	// 全局中间件
	s.Use(middleware.CORS, middleware.Logging, middleware.ErrorHandler)

	// API 路由组
	apiGroup := s.Group("/api/v1")
	{
		// 用户相关接口
		userGroup := apiGroup.Group("/user")
		{
			userGroup.POST("/create", controller.User.Create)
			userGroup.GET("/list", controller.User.List)
			userGroup.GET("/:id", controller.User.Get)
			userGroup.PUT("/:id", controller.User.Update)
			userGroup.DELETE("/:id", controller.User.Delete)
		}

		// 健康检查
		apiGroup.GET("/health", controller.Health.Check)
	}

	// 静态文件服务
	s.SetServerRoot("public")
}
