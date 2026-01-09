package task

import (
	"context"
	"data/internal/task/jobs"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gcron"
)

var cron *gcron.Cron

// Start 启动定时任务
func Start(ctx context.Context) {
	cron = gcron.New()
	
	// 注册定时任务
	registerTasks(ctx)
	
	// 启动定时任务管理器
	cron.Start()
	
	g.Log().Info(ctx, "定时任务已启动")
}

// Stop 停止定时任务
func Stop(ctx context.Context) {
	if cron != nil {
		cron.Stop()
		g.Log().Info(ctx, "定时任务已停止")
	}
}

// registerTasks 注册所有定时任务
func registerTasks(ctx context.Context) {
	// KyberSwap 监控任务：每30秒执行一次
	// 注意：标准 cron 不支持秒级，使用 GoFrame 的 @every 语法
	cron.Add(ctx, "@every 30s", jobs.KyberSwapMonitorJob, "kyberswap_monitor")
	
	// 示例任务：每5分钟执行一次
	cron.Add(ctx, "*/5 * * * *", jobs.ExampleJob, "example_task")
	
	// 示例任务：每天凌晨2点执行
	cron.Add(ctx, "0 2 * * *", jobs.DailyJob, "daily_task")
	
	// 示例任务：每小时执行一次
	cron.Add(ctx, "0 * * * *", jobs.HourlyJob, "hourly_task")
	
	g.Log().Info(ctx, "定时任务注册完成")
}
