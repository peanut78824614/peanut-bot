package controller

import (
	"data/internal/model"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

// MiddlewareCORS 跨域中间件
func MiddlewareCORS(r *ghttp.Request) {
	r.Response.CORSDefault()
	r.Middleware.Next()
}

// MiddlewareLog 日志中间件
func MiddlewareLog(r *ghttp.Request) {
	start := time.Now()
	
	// 继续处理请求
	r.Middleware.Next()

	// 记录请求日志
	duration := time.Since(start)
	
	g.Log().Info(r.Context(), g.Map{
		"method":   r.Method,
		"path":     r.URL.Path,
		"ip":       r.GetClientIp(),
		"status":   r.Response.Status,
		"duration": duration.String(),
		"size":     r.Response.BufferLength(),
	})
}

// MiddlewareError 错误处理中间件
func MiddlewareError(r *ghttp.Request) {
	r.Middleware.Next()

	if err := r.GetError(); err != nil {
		g.Log().Error(r.Context(), "请求处理错误:", err)
		r.Response.WriteJson(model.Response{
			Code:    500,
			Message: "服务器内部错误",
			Data:    nil,
		})
	}
}
