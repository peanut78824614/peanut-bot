package jobs

import (
	"context"
	"data/internal/model"
	"data/internal/service"
	"fmt"
	"strings"
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

// KyberSwapMonitorJob KyberSwap 监控任务
func KyberSwapMonitorJob(ctx context.Context) {
	g.Log().Info(ctx, "开始执行 KyberSwap 监控任务...")
	
	kyberSwap := service.KyberSwap()
	telegram := service.Telegram()
	
	// 获取今天已推送的池子ID列表
	sentPoolIDs, err := kyberSwap.GetTodaySentPoolIDs(ctx)
	if err != nil {
		g.Log().Error(ctx, "获取今天已推送池子列表失败:", err)
		sentPoolIDs = make(map[string]bool) // 使用空map继续执行
	}
	
	g.Log().Info(ctx, fmt.Sprintf("今天已推送 %d 个池子", len(sentPoolIDs)))
	
	// 获取新数据
	newPools, err := kyberSwap.FetchAllPools(ctx)
	if err != nil {
		g.Log().Error(ctx, "获取 KyberSwap 数据失败:", err)
		return
	}
	
	g.Log().Info(ctx, fmt.Sprintf("获取到 %d 个池子数据", len(newPools)))
	
	// 找出今天未推送的池子（新增的池子）
	poolsToNotify := make([]model.Pool, 0)
	poolIDsToAdd := make([]string, 0)
	
	for _, pool := range newPools {
		// 如果这个池子今天还没有推送过，则加入推送列表
		if !sentPoolIDs[pool.ID] {
			poolsToNotify = append(poolsToNotify, pool)
			poolIDsToAdd = append(poolIDsToAdd, pool.ID)
		}
	}
	
	if len(poolsToNotify) > 0 {
		g.Log().Info(ctx, fmt.Sprintf("发现 %d 个新池子（今天首次出现），准备发送通知", len(poolsToNotify)))
		
		// 发送到 Telegram
		telegramChatId := g.Cfg().MustGet(ctx, "telegram.chatId", "").String()
		if telegramChatId != "" {
			message := service.FormatPoolsMessage(poolsToNotify, false)
			buttonText := "联系作者 VX : love-home8"
			buttonURL := "https://www.baidu.com"
			maxLength := 4096
			if len(message) > maxLength {
				messages := splitMessage(message, maxLength)
				for i, msg := range messages {
					if i == 0 {
						if err := telegram.SendMessageWithMarkdownAndButton(ctx, telegramChatId, msg, buttonText, buttonURL); err != nil {
							g.Log().Error(ctx, "发送 Telegram 消息失败:", err)
						} else {
							g.Log().Info(ctx, "Telegram 消息发送成功")
						}
					} else {
						if err := telegram.SendMessageWithMarkdown(ctx, telegramChatId, msg); err != nil {
							g.Log().Error(ctx, "发送 Telegram 消息失败:", err)
						} else {
							g.Log().Info(ctx, "Telegram 消息发送成功")
						}
					}
					time.Sleep(1 * time.Second)
				}
			} else {
				if err := telegram.SendMessageWithMarkdownAndButton(ctx, telegramChatId, message, buttonText, buttonURL); err != nil {
					g.Log().Error(ctx, "发送 Telegram 消息失败:", err)
				} else {
					g.Log().Info(ctx, "Telegram 消息发送成功")
				}
			}
			
			// 记录已推送的池子ID到今天的记录中
			if err := kyberSwap.AddSentPoolIDs(ctx, poolIDsToAdd); err != nil {
				g.Log().Error(ctx, "保存已推送池子ID失败:", err)
			} else {
				g.Log().Info(ctx, fmt.Sprintf("已记录 %d 个池子ID到今天推送历史", len(poolIDsToAdd)))
			}
		} else {
			g.Log().Warning(ctx, "Telegram Chat ID 未配置，跳过 Telegram 通知")
		}
	} else {
		g.Log().Info(ctx, "今天所有池子都已推送过，没有新池子")
	}
	
	// 保存最新的池子数据（用于其他用途，不用于推送判断）
	if err := kyberSwap.SavePools(ctx, newPools); err != nil {
		g.Log().Error(ctx, "保存池子数据失败:", err)
	} else {
		g.Log().Info(ctx, "池子数据保存成功")
	}
	
	g.Log().Info(ctx, "KyberSwap 监控任务执行完成")
}

// splitMessage 分割长消息
func splitMessage(text string, maxLength int) []string {
	if len(text) <= maxLength {
		return []string{text}
	}
	
	var messages []string
	lines := strings.Split(text, "\n")
	currentMessage := ""
	
	for _, line := range lines {
		if len(currentMessage)+len(line)+1 > maxLength {
			if currentMessage != "" {
				messages = append(messages, currentMessage)
				currentMessage = line + "\n"
			} else {
				// 单行太长，强制分割
				for len(line) > maxLength {
					messages = append(messages, line[:maxLength])
					line = line[maxLength:]
				}
				currentMessage = line + "\n"
			}
		} else {
			currentMessage += line + "\n"
		}
	}
	
	if currentMessage != "" {
		messages = append(messages, currentMessage)
	}
	
	return messages
}

// ResetDailySentPoolsJob 每天0点重置已推送池子记录
func ResetDailySentPoolsJob(ctx context.Context) {
	g.Log().Info(ctx, "开始执行每日重置已推送池子记录任务...")
	
	kyberSwap := service.KyberSwap()
	
	if err := kyberSwap.ResetDailySentPools(ctx); err != nil {
		g.Log().Error(ctx, "重置每日已推送池子记录失败:", err)
	} else {
		g.Log().Info(ctx, "每日已推送池子记录重置成功")
	}
}
