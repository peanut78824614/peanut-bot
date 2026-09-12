package jobs

import (
	"context"
	"data/internal/model"
	"data/internal/service"
	"fmt"
	"sync"
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

var (
	farmingMonitorMu        sync.Mutex
	farmingUnconfiguredOnce sync.Once
)

const (
	farmingButtonText = "联系作者 VX : love-home8"
	farmingButtonURL  = "https://www.baidu.com"
	telegramMaxLength = 4096
)

// KyberSwapFarmingMonitorJob farming_pool 按链拆群推送。
// 与原有 high_apr 单群推送并行，互不影响；确认跑通后再关闭旧任务。
func KyberSwapFarmingMonitorJob(ctx context.Context) {
	if !service.FarmingPushEnabled(ctx) {
		return
	}
	if !farmingMonitorMu.TryLock() {
		g.Log().Warning(ctx, "上一轮 farming_pool 监控仍在执行，跳过本轮")
		return
	}
	defer farmingMonitorMu.Unlock()

	configured := false
	for _, chain := range service.FarmingChains() {
		if service.FarmingChatID(ctx, chain.Key) != "" {
			configured = true
			break
		}
	}
	if !configured {
		farmingUnconfiguredOnce.Do(func() {
			g.Log().Warning(ctx, "farming 已启用，但四条链的群组 Chat ID 都未配置，跳过推送。请填写 telegram.farming.eth/base/bsc/robinhood")
		})
		return
	}

	g.Log().Info(ctx, "开始执行 KyberSwap farming_pool 按链推送任务...")

	kyberSwap := service.KyberSwap()
	telegram := service.Telegram()

	for _, chain := range service.FarmingChains() {
		notifyFarmingChain(ctx, kyberSwap, telegram, chain)
	}

	g.Log().Info(ctx, "KyberSwap farming_pool 按链推送任务执行完成")
}

func notifyFarmingChain(ctx context.Context, kyberSwap service.IKyberSwap, telegram service.ITelegram, chain service.FarmingChain) {
	chatID := service.FarmingChatID(ctx, chain.Key)
	if chatID == "" {
		g.Log().Warning(ctx, fmt.Sprintf("farming %s(%d) 群组 Chat ID 未配置（telegram.farming.%s），跳过推送", chain.Label, chain.ID, chain.Key))
		return
	}

	sentPoolIDs, err := kyberSwap.GetTodaySentFarmingPoolIDs(ctx, chain.ID)
	if err != nil {
		g.Log().Error(ctx, fmt.Sprintf("获取 %s farming 今日已推送列表失败:", chain.Label), err)
		sentPoolIDs = make(map[string]bool)
	}
	g.Log().Info(ctx, fmt.Sprintf("farming %s 今天已推送 %d 个池子", chain.Label, len(sentPoolIDs)))

	newPools, err := kyberSwap.FetchFarmingPoolsByChain(ctx, chain.ID)
	if err != nil {
		g.Log().Error(ctx, fmt.Sprintf("获取 %s high_apr 数据失败:", chain.Label), err)
		return
	}
	g.Log().Info(ctx, fmt.Sprintf("farming %s 获取到 %d 个池子（不过滤）", chain.Label, len(newPools)))

	poolsToNotify := make([]model.Pool, 0)
	poolIDsToAdd := make([]string, 0)
	for _, pool := range newPools {
		if !sentPoolIDs[pool.ID] {
			poolsToNotify = append(poolsToNotify, pool)
			poolIDsToAdd = append(poolIDsToAdd, pool.ID)
		}
	}

	if len(poolsToNotify) == 0 {
		g.Log().Info(ctx, fmt.Sprintf("farming %s 今天所有池子都已推送过，没有新池子", chain.Label))
	} else {
		g.Log().Info(ctx, fmt.Sprintf("farming %s 发现 %d 个新池子，准备发送到群组 %s", chain.Label, len(poolsToNotify), chatID))

		message := service.FormatPoolsMessage(poolsToNotify, false)
		if err := sendFarmingTelegram(ctx, telegram, chatID, message); err != nil {
			g.Log().Error(ctx, fmt.Sprintf("farming %s 发送 Telegram 失败:", chain.Label), err)
		} else {
			g.Log().Info(ctx, fmt.Sprintf("farming %s Telegram 消息发送成功", chain.Label))
		}

		if err := kyberSwap.AddSentFarmingPoolIDs(ctx, chain.ID, poolIDsToAdd); err != nil {
			g.Log().Error(ctx, fmt.Sprintf("保存 %s farming 已推送池子 ID 失败:", chain.Label), err)
		} else {
			g.Log().Info(ctx, fmt.Sprintf("farming %s 已记录 %d 个池子 ID", chain.Label, len(poolIDsToAdd)))
		}
	}

	notifyFarmingEarnFeeSurge(ctx, kyberSwap, telegram, chain, chatID, newPools)
}

func notifyFarmingEarnFeeSurge(ctx context.Context, kyberSwap service.IKyberSwap, telegram service.ITelegram, chain service.FarmingChain, chatID string, pools []model.Pool) {
	history, err := kyberSwap.GetFarmingEarnFeeHistoryWithTime(ctx, chain.ID)
	if err != nil {
		g.Log().Warning(ctx, fmt.Sprintf("farming %s 获取 earnFee 历史失败，将按空记录继续:", chain.Label), err)
		history = make(map[string]service.EarnFeeHistory)
	}

	poolsToNotify, poolsHistory, updates := service.DetectEarnFeeSurges(pools, history)
	if err := kyberSwap.UpdateFarmingEarnFeeHistories(ctx, chain.ID, updates); err != nil {
		g.Log().Error(ctx, fmt.Sprintf("farming %s 批量更新 earnFee 历史失败:", chain.Label), err)
	}

	if len(poolsToNotify) == 0 {
		g.Log().Info(ctx, fmt.Sprintf("farming %s 没有发现交易额暴增的池子", chain.Label))
		return
	}

	for _, pool := range poolsToNotify {
		historyItem := poolsHistory[pool.ID]
		oldEarnFee := historyItem.Value
		increaseRatio := 0.0
		if oldEarnFee > 0 {
			increaseRatio = (pool.Fees24h - oldEarnFee) / oldEarnFee
		}
		g.Log().Info(ctx, fmt.Sprintf("farming %s 池子 %s earnFee 从 %.2f 增加到 %.2f，增长 %.2f%%",
			chain.Label, pool.ID, oldEarnFee, pool.Fees24h, increaseRatio*100))
	}
	g.Log().Info(ctx, fmt.Sprintf("farming %s 发现 %d 个交易额暴增的池子，准备发送到群组 %s", chain.Label, len(poolsToNotify), chatID))

	message := service.FormatEarnFeeSurgeMessage(poolsToNotify, poolsHistory)
	if err := sendFarmingTelegram(ctx, telegram, chatID, message); err != nil {
		g.Log().Error(ctx, fmt.Sprintf("farming %s 发送交易额暴增消息失败:", chain.Label), err)
	} else {
		g.Log().Info(ctx, fmt.Sprintf("farming %s 交易额暴增消息发送成功", chain.Label))
	}
}

func sendFarmingTelegram(ctx context.Context, telegram service.ITelegram, chatID, message string) error {
	if len(message) <= telegramMaxLength {
		return telegram.SendMessageWithMarkdownAndButton(ctx, chatID, message, farmingButtonText, farmingButtonURL)
	}

	messages := splitMessage(message, telegramMaxLength)
	var firstErr error
	for i, msg := range messages {
		var err error
		if i == 0 {
			err = telegram.SendMessageWithMarkdownAndButton(ctx, chatID, msg, farmingButtonText, farmingButtonURL)
		} else {
			err = telegram.SendMessageWithMarkdown(ctx, chatID, msg)
		}
		if err != nil {
			g.Log().Error(ctx, "发送 Telegram 消息失败:", err)
			if firstErr == nil {
				firstErr = err
			}
		} else {
			g.Log().Info(ctx, "Telegram 消息发送成功")
		}
		time.Sleep(1 * time.Second)
	}
	return firstErr
}
