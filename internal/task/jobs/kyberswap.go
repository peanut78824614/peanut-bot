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

// KyberSwapEarnFeeMonitorJob 监控 earnFee 变化的任务
func KyberSwapEarnFeeMonitorJob(ctx context.Context) {
	g.Log().Info(ctx, "开始执行 KyberSwap EarnFee 监控任务...")
	
	kyberSwap := service.KyberSwap()
	telegram := service.Telegram()
	
	// 获取当前的池子数据
	newPools, err := kyberSwap.FetchAllPools(ctx)
	if err != nil {
		g.Log().Error(ctx, "获取 KyberSwap 数据失败:", err)
		return
	}
	
	g.Log().Info(ctx, fmt.Sprintf("获取到 %d 个池子数据", len(newPools)))
	
	// 获取历史 earnFee 值
	history, err := kyberSwap.GetPoolEarnFeeHistory(ctx)
	if err != nil {
		g.Log().Error(ctx, "获取 earnFee 历史值失败:", err)
		return
	}
	
	// 找出 earnFee 有明显增加的池子
	poolsToNotify := make([]model.Pool, 0)
	// 保存需要推送的池子的历史值，用于显示原值和现值
	poolsHistory := make(map[string]float64)
	
	for _, pool := range newPools {
		oldEarnFee, exists := history[pool.ID]
		
		// 如果历史值不存在，记录当前值但不推送
		if !exists {
			// 更新历史值
			if err := kyberSwap.UpdatePoolEarnFeeHistory(ctx, pool.ID, pool.Fees24h); err != nil {
				g.Log().Error(ctx, fmt.Sprintf("更新池子 %s 的 earnFee 历史值失败: %v", pool.ID, err))
			}
			continue
		}
		
		// 判断是否有明显增加（例如从100增加到105，即增加5%）
		// 如果旧值为0，跳过
		if oldEarnFee <= 0 {
			// 更新历史值
			if err := kyberSwap.UpdatePoolEarnFeeHistory(ctx, pool.ID, pool.Fees24h); err != nil {
				g.Log().Error(ctx, fmt.Sprintf("更新池子 %s 的 earnFee 历史值失败: %v", pool.ID, err))
			}
			continue
		}
		
		// 计算增长比例
		increaseRatio := (pool.Fees24h - oldEarnFee) / oldEarnFee
		
		// 如果增长超过5%（0.05），则推送
		if increaseRatio >= 0.05 {
			g.Log().Info(ctx, fmt.Sprintf("池子 %s earnFee 从 %.2f 增加到 %.2f，增长 %.2f%%", 
				pool.ID, oldEarnFee, pool.Fees24h, increaseRatio*100))
			poolsToNotify = append(poolsToNotify, pool)
			// 保存历史值，用于显示
			poolsHistory[pool.ID] = oldEarnFee
		}
		
		// 更新历史值
		if err := kyberSwap.UpdatePoolEarnFeeHistory(ctx, pool.ID, pool.Fees24h); err != nil {
			g.Log().Error(ctx, fmt.Sprintf("更新池子 %s 的 earnFee 历史值失败: %v", pool.ID, err))
		}
	}
	
	// 如果有需要推送的池子，发送通知
	if len(poolsToNotify) > 0 {
		g.Log().Info(ctx, fmt.Sprintf("发现 %d 个交易额暴增的池子，准备发送通知", len(poolsToNotify)))
		
		// 发送到 Telegram
		telegramChatId := g.Cfg().MustGet(ctx, "telegram.chatId", "").String()
		if telegramChatId != "" {
			message := service.FormatEarnFeeSurgeMessage(poolsToNotify, poolsHistory)
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
		} else {
			g.Log().Warning(ctx, "Telegram Chat ID 未配置，跳过 Telegram 通知")
		}
	} else {
		g.Log().Info(ctx, "没有发现交易额暴增的池子")
	}
	
	g.Log().Info(ctx, "KyberSwap EarnFee 监控任务执行完成")
}
