package main

import (
	"context"
	"data/internal/controller"
	"data/internal/service"
	"data/internal/task"
	"os"
	"os/signal"
	"syscall"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gctx"
)

func main() {
	ctx := gctx.New()

	// 初始化服务
	service.Init(ctx)

	// 启动定时任务
	task.Start(ctx)

	// 启动 HTTP 服务器
	s := g.Server()
	s.Group("/api/v1", func(group *ghttp.RouterGroup) {
		group.Middleware(
			controller.MiddlewareCORS,
			controller.MiddlewareLog,
		)
		controller.RegisterRoutes(group)
	})

	// 优雅关闭
	handleGracefulShutdown(ctx, s)
}

// handleGracefulShutdown 处理优雅关闭
func handleGracefulShutdown(ctx context.Context, s *ghttp.Server) {
	// 监听系统信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// 启动服务器
	go func() {
		s.Run()
	}()

	// 等待关闭信号
	<-sigChan
	g.Log().Info(ctx, "收到关闭信号，开始优雅关闭...")

	// 停止定时任务
	task.Stop(ctx)

	// 关闭服务器
	if err := s.Shutdown(); err != nil {
		g.Log().Error(ctx, "服务器关闭失败:", err)
	}

	g.Log().Info(ctx, "服务器已关闭")
}
