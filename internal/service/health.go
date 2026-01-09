package service

import (
	"context"
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

type sHealth struct{}

var Health = sHealth{}

// Check 健康检查
func (s *sHealth) Check(ctx context.Context) map[string]interface{} {
	return map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
		"version":   g.Cfg().MustGet(ctx, "app.version", "1.0.0"),
	}
}
