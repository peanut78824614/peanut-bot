package cmd

import (
	"context"
	"data/internal/router"
	"os"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gcmd"
	"github.com/gogf/gf/v2/os/gctx"
)

var (
	Http = gcmd.Command{
		Name:  "http",
		Usage: "http",
		Brief: "启动 HTTP 服务器",
		Func: func(ctx context.Context, parser *gcmd.Parser) (err error) {
			s := g.Server()
			
			// 注册路由
			router.InitRouter(ctx)
			
			// 启动服务器
			s.Run()
			return nil
		},
	}
)
