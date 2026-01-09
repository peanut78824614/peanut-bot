package controller

import (
	"data/internal/model"
	"data/internal/service"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

// TelegramController Telegram 控制器
type TelegramController struct{}

// NewTelegramController 创建 Telegram 控制器
func NewTelegramController() *TelegramController {
	return &TelegramController{}
}

// GetUpdates 获取 Telegram 更新（用于查找 Chat ID）
func (c *TelegramController) GetUpdates(r *ghttp.Request) {
	ctx := r.Context()
	telegram := service.Telegram()

	updates, err := telegram.GetUpdates(ctx)
	if err != nil {
		r.Response.WriteJson(g.Map{
			"code":    500,
			"message": "获取更新失败: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// 提取所有聊天信息
	chats := make([]g.Map, 0)
	chatMap := make(map[int64]bool) // 用于去重

	for _, update := range updates {
		if update.Message != nil && update.Message.Chat != nil {
			chatID := update.Message.Chat.ID
			if !chatMap[chatID] {
				chatMap[chatID] = true
				chatInfo := g.Map{
					"id":        update.Message.Chat.ID,
					"type":      update.Message.Chat.Type,
					"title":     update.Message.Chat.Title,
					"username":  update.Message.Chat.Username,
					"firstName": update.Message.Chat.FirstName,
					"lastName":  update.Message.Chat.LastName,
				}
				chats = append(chats, chatInfo)
			}
		}
	}

	r.Response.WriteJson(g.Map{
		"code":    200,
		"message": "success",
		"data": g.Map{
			"chats":  chats,
			"count":  len(chats),
			"total":  len(updates),
			"tip":    "在群组或个人聊天中发送任意消息给 Bot，然后刷新此接口即可看到 Chat ID",
		},
	})
}

// GetChatInfo 获取指定 Chat ID 的信息
func (c *TelegramController) GetChatInfo(r *ghttp.Request) {
	ctx := r.Context()
	chatID := r.Get("chatId").String()

	if chatID == "" {
		r.Response.WriteJson(g.Map{
			"code":    400,
			"message": "请提供 chatId 参数",
			"data":    nil,
		})
		return
	}

	telegram := service.Telegram()
	chatInfo, err := telegram.GetChatInfo(ctx, chatID)
	if err != nil {
		r.Response.WriteJson(g.Map{
			"code":    500,
			"message": "获取聊天信息失败: " + err.Error(),
			"data":    nil,
		})
		return
	}

	r.Response.WriteJson(g.Map{
		"code":    200,
		"message": "success",
		"data": g.Map{
			"id":        chatInfo.ID,
			"type":      chatInfo.Type,
			"title":     chatInfo.Title,
			"username":  chatInfo.Username,
			"firstName": chatInfo.FirstName,
			"lastName":  chatInfo.LastName,
		},
	})
}

// SendTestMessage 发送测试消息（用于查看消息格式）
func (c *TelegramController) SendTestMessage(r *ghttp.Request) {
	ctx := r.Context()
	telegram := service.Telegram()
	
	// 获取配置的 Chat ID
	chatID := g.Cfg().MustGet(ctx, "telegram.chatId", "").String()
	if chatID == "" {
		r.Response.WriteJson(g.Map{
			"code":    400,
			"message": "请先在配置文件中设置 telegram.chatId",
			"data":    nil,
		})
		return
	}
	
	// 创建示例池子数据
	testPools := []model.Pool{
		{
			ID:          "test-pool-1",
			Name:        "USDC/USDT",
			APR:         125.50,
			TVL:         2500000,
			ChainID:     56,
			Token0:      "0x123...",
			Token1:      "0x456...",
			Token0Symbol: "USDC",
			Token1Symbol: "USDT",
			Volume24h:   1200000,
			Fees24h:     500000,
			URL:         "https://kyberswap.com/earn/pools/test-pool-1",
			Version:     "v4",
			FeeTier:     1,
			Protocol:    "Uniswap",
		},
		{
			ID:          "test-pool-2",
			Name:        "ETH/BTC",
			APR:         98.75,
			TVL:         5000000,
			ChainID:     8453,
			Token0:      "0x789...",
			Token1:      "0xabc...",
			Token0Symbol: "ETH",
			Token1Symbol: "BTC",
			Volume24h:   2500000,
			Fees24h:     1000000,
			URL:         "https://kyberswap.com/earn/pools/test-pool-2",
			Version:     "v3",
			FeeTier:     3,
			Protocol:    "Pancake",
		},
		{
			ID:          "test-pool-3",
			Name:        "BNB/CAKE",
			APR:         156.80,
			TVL:         1800000,
			ChainID:     56,
			Token0:      "0xdef...",
			Token1:      "0xghi...",
			Token0Symbol: "BNB",
			Token1Symbol: "CAKE",
			Volume24h:   800000,
			Fees24h:     320000,
			URL:         "https://kyberswap.com/earn/pools/test-pool-3",
			Version:     "v4",
			FeeTier:     1,
			Protocol:    "Uniswap",
		},
	}
	
	// 格式化消息
	message := service.FormatPoolsMessage(testPools, false)
	
	// 发送消息
	if err := telegram.SendMessageWithMarkdown(ctx, chatID, message); err != nil {
		r.Response.WriteJson(g.Map{
			"code":    500,
			"message": "发送测试消息失败: " + err.Error(),
			"data":    nil,
		})
		return
	}
	
	r.Response.WriteJson(g.Map{
		"code":    200,
		"message": "测试消息发送成功！请查看 Telegram 群组",
		"data": g.Map{
			"chatId": chatID,
			"pools":  len(testPools),
			"tip":    "这是测试消息，展示了消息格式效果",
		},
	})
}
