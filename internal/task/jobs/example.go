package jobs

import (
	"context"
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

// ExampleJob 示例定时任务
func ExampleJob(ctx context.Context) {
	g.Log().Info(ctx, "执行示例定时任务，当前时间:", time.Now().Format("2006-01-02 15:04:05"))
	
	// 这里可以添加具体的业务逻辑
	// 例如：数据清理、数据同步、报表生成等
	
	// 模拟业务处理
	time.Sleep(1 * time.Second)
	
	g.Log().Info(ctx, "示例定时任务执行完成")
}

// DailyJob 每日定时任务
func DailyJob(ctx context.Context) {
	g.Log().Info(ctx, "执行每日定时任务，当前时间:", time.Now().Format("2006-01-02 15:04:05"))
	
	// 每日任务示例：数据备份、日志清理等
	// 可以在这里添加具体的业务逻辑
	
	g.Log().Info(ctx, "每日定时任务执行完成")
}

// HourlyJob 每小时定时任务
func HourlyJob(ctx context.Context) {
	g.Log().Info(ctx, "执行每小时定时任务，当前时间:", time.Now().Format("2006-01-02 15:04:05"))
	
	// 每小时任务示例：数据统计、缓存刷新等
	// 可以在这里添加具体的业务逻辑
	
	g.Log().Info(ctx, "每小时定时任务执行完成")
}
