package middleware

import (
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

// ErrorHandler 错误处理中间件
func ErrorHandler(r *ghttp.Request) {
	r.Middleware.Next()
	
	if err := r.GetError(); err != nil {
		// 记录错误日志
		g.Log().Error(r.Context(), err)
		
		// 返回错误响应
		r.Response.WriteJsonExit(g.Map{
			"code":    gerror.Code(err).Code(),
			"message": err.Error(),
			"data":    nil,
		})
	}
}
