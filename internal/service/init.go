package service

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
)

// Init 初始化服务
func Init(ctx context.Context) {
	g.Log().Info(ctx, "服务初始化完成")
}
