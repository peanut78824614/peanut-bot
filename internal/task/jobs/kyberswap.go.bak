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
	
	// 获取存储的旧数据
	oldPools, err := kyberSwap.GetStoredPools(ctx)
	if err != nil {
		g.Log().Error(ctx, "获取存储的池子数据失败:", err)
		oldPools = []model.Pool{} // 使用空列表继续执行
	}
	
	// 获取新数据
	newPools, err := kyberSwap.FetchAllPools(ctx)
	if err != nil {
		g.Log().Error(ctx, "获取 KyberSwap 数据失败:", err)
		return
	}
	
	g.Log().Info(ctx, fmt.Sprintf("获取到 %d 个池子数据", len(newPools)))
	
	// 判断是否是首次运行（没有旧数据）
	isFirstRun := len(oldPools) == 0
	
	// 比较数据，找出新增的池子
	newPoolsList := kyberSwap.ComparePools(oldPools, newPools)
	
	// 如果是首次运行，推送所有池子；否则只推送新增的池子
	poolsToNotify := newPoolsList
	if isFirstRun && len(newPools) > 0 {
		g.Log().Info(ctx, "首次运行，将推送所有池子数据")
		poolsToNotify = newPools
	}
	
	if len(poolsToNotify) > 0 {
		messagePrefix := "新池子"
		if isFirstRun {
			messagePrefix = "池子"
		}
		g.Log().Info(ctx, fmt.Sprintf("发现 %d 个%s，准备发送通知", len(poolsToNotify), messagePrefix))
		
		// 发送到 Telegram
		telegramChatId := g.Cfg().MustGet(ctx, "telegram.chatId", "").String()
		if telegramChatId != "" {
			message := service.FormatPoolsMessage(poolsToNotify, isFirstRun)
			
			// 如果消息太长，分批发送
			maxLength := 4096 // Telegram 消息最大长度
			if len(message) > maxLength {
				// 分批发送
				messages := splitMessage(message, maxLength)
				for _, msg := range messages {
					if err := telegram.SendMessageWithMarkdown(ctx, telegramChatId, msg); err != nil {
						g.Log().Error(ctx, "发送 Telegram 消息失败:", err)
					} else {
						g.Log().Info(ctx, "Telegram 消息发送成功")
					}
					// 避免发送过快
					time.Sleep(1 * time.Second)
				}
			} else {
				// 发送单个消息
				if err := telegram.SendMessageWithMarkdown(ctx, telegramChatId, message); err != nil {
					g.Log().Error(ctx, "发送 Telegram 消息失败:", err)
				} else {
					g.Log().Info(ctx, "Telegram 消息发送成功")
				}
			}
		} else {
			g.Log().Warning(ctx, "Telegram Chat ID 未配置，跳过 Telegram 通知")
		}
	} else {
		g.Log().Info(ctx, "没有发现新池子")
	}
	
	// 保存新数据
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
