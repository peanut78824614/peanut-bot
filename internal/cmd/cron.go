package cmd

import (
	"context"
	"data/internal/cron"
	"os"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gcmd"
	"github.com/gogf/gf/v2/os/gctx"
)

var (
	Cron = gcmd.Command{
		Name:  "cron",
		Usage: "cron",
		Brief: "启动定时任务",
		Func: func(ctx context.Context, parser *gcmd.Parser) (err error) {
			// 初始化定时任务
			cron.InitCron(ctx)
			
			// 保持运行
			select {}
		},
	}
)
