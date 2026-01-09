package controller

import (
	"data/internal/service"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

type cHealth struct{}

var Health = cHealth{}

// Check 健康检查
func (c *cHealth) Check(r *ghttp.Request) {
	status := service.Health.Check(r.Context())
	r.Response.WriteJsonExit(g.Map{
		"code":    0,
		"message": "success",
		"data":    status,
	})
}
