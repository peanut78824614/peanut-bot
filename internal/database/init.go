package database

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
)

// Init 初始化数据库（当前仅 Kyber + Telegram 功能，无需业务表）
func Init(ctx context.Context) error {
	_ = g.DB()
	g.Log().Info(ctx, "数据库初始化完成")
	return nil
}
