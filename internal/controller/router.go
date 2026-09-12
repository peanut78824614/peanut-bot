package controller

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

// RegisterRoutes 注册所有路由
func RegisterRoutes(group *ghttp.RouterGroup) {
	// 健康检查
	group.GET("/health", HealthCheck)

	// Telegram 辅助接口
	telegramCtrl := NewTelegramController()
	group.Group("/telegram", func(group *ghttp.RouterGroup) {
		group.GET("/updates", telegramCtrl.GetUpdates)                  // 获取更新，用于查找 Chat ID
		group.GET("/bot", telegramCtrl.GetBotInfo)                      // 获取 Bot ID / 用户名
		group.GET("/chat/:chatId", telegramCtrl.GetChatInfo)            // 获取指定 Chat ID 的信息
		group.POST("/test", telegramCtrl.SendTestMessage)               // 发送测试消息，查看格式效果（POST）
		group.GET("/test", telegramCtrl.SendTestMessage)                // 发送测试消息，查看格式效果（GET，方便浏览器测试）
		group.GET("/test-farming", telegramCtrl.SendFarmingTestMessage) // 向各链 farming 群组发送测试消息
		group.GET("/farming-preview", telegramCtrl.FarmingPreview)      // 预览四条链 farming_pool 拉取结果（不推送）
	})
}

// HealthCheck 健康检查接口
func HealthCheck(r *ghttp.Request) {
	r.Response.WriteJson(g.Map{
		"status":  "ok",
		"message": "服务运行正常",
	})
}
