package middleware

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

// Logging 日志记录中间件
func Logging(r *ghttp.Request) {
	start := time.Now()
	
	r.Middleware.Next()
	
	// 记录请求日志
	g.Log().Info(r.Context(), 
		"method:", r.Method,
		"path:", r.URL.Path,
		"ip:", r.GetClientIp(),
		"status:", r.Response.Status,
		"duration:", time.Since(start).String(),
	)
}
