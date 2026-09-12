package service

import (
	"context"
	"data/internal/model"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gfile"
)

// 四条链按群推送：与原 high_apr 单群推送完全独立。
// ETH(1) / Base(8453) / BSC(56) / Robinhood(4663) 各请求一次 tag=high_apr，不过滤池子，按链推送到各自群组。
const farmingServicePoolsURL = "https://earn-service.kyberswap.com/api/v1/explorer/pools?chainIds=%d&page=%d&limit=100&interval=24h&protocol=&tag=high_apr&sortBy=&orderBy=&q="

const (
	farmingEthChainID       = 1
	farmingBaseChainID      = 8453
	farmingBscChainID       = 56
	farmingRobinhoodChainID = robinhoodChainID
)

// FarmingChain 按链拆群推送的配置
type FarmingChain struct {
	ID    int
	Key   string // 对应 telegram.farming.<key>
	Label string
}

// FarmingChains 返回 farming_pool 新推送监控的四条链
func FarmingChains() []FarmingChain {
	return []FarmingChain{
		{ID: farmingEthChainID, Key: "eth", Label: "ETH"},
		{ID: farmingBaseChainID, Key: "base", Label: "Base"},
		{ID: farmingBscChainID, Key: "bsc", Label: "BSC"},
		{ID: farmingRobinhoodChainID, Key: "robinhood", Label: "Robinhood"},
	}
}

func farmingServiceURL(chainID, page int) string {
	return fmt.Sprintf(farmingServicePoolsURL, chainID, page)
}

func farmingSentFilePath(chainID int) string {
	today := time.Now().Format("2006-01-02")
	return fmt.Sprintf("data/farming_sent_pools_%d_%s.json", chainID, today)
}

// FarmingChatID 读取某条链对应的 Telegram 群组 Chat ID
func FarmingChatID(ctx context.Context, chainKey string) string {
	return g.Cfg().MustGet(ctx, "telegram.farming."+chainKey, "").String()
}

// FarmingPushEnabled 新推送总开关
func FarmingPushEnabled(ctx context.Context) bool {
	return g.Cfg().MustGet(ctx, "telegram.farming.enabled", false).Bool()
}

// FetchFarmingPoolsByChain 拉取指定链的 high_apr 池子（不过滤）
func (s *kyberSwapImpl) FetchFarmingPoolsByChain(ctx context.Context, chainID int) ([]model.Pool, error) {
	url := farmingServiceURL(chainID, 1)
	g.Log().Info(ctx, fmt.Sprintf("正在获取 high_apr %s(%d) page=1 的池子数据...", earnServiceChainLabel(chainID), chainID))
	pools, err := s.fetchPoolsFromURL(ctx, url, true)
	if err != nil {
		return nil, err
	}
	g.Log().Info(ctx, fmt.Sprintf("high_apr %s(%d) 解析到 %d 个池子（不过滤）", earnServiceChainLabel(chainID), chainID, len(pools)))
	return pools, nil
}

// GetTodaySentFarmingPoolIDs 获取今天某条链已推送的 farming 池子 ID
func (s *kyberSwapImpl) GetTodaySentFarmingPoolIDs(ctx context.Context, chainID int) (map[string]bool, error) {
	filePath := farmingSentFilePath(chainID)
	if !gfile.Exists(filePath) {
		return make(map[string]bool), nil
	}

	content := gfile.GetContents(filePath)
	if content == "" || content == "[]" {
		return make(map[string]bool), nil
	}

	var poolIDs []string
	if err := json.Unmarshal([]byte(content), &poolIDs); err != nil {
		return nil, err
	}

	poolIDMap := make(map[string]bool, len(poolIDs))
	for _, id := range poolIDs {
		poolIDMap[id] = true
	}
	return poolIDMap, nil
}

// AddSentFarmingPoolIDs 记录今天某条链已推送的 farming 池子 ID（与旧推送文件隔离）
func (s *kyberSwapImpl) AddSentFarmingPoolIDs(ctx context.Context, chainID int, poolIDs []string) error {
	if len(poolIDs) == 0 {
		return nil
	}

	filePath := farmingSentFilePath(chainID)
	existingMap, err := s.GetTodaySentFarmingPoolIDs(ctx, chainID)
	if err != nil {
		return err
	}
	for _, id := range poolIDs {
		existingMap[id] = true
	}

	allIDs := make([]string, 0, len(existingMap))
	for id := range existingMap {
		allIDs = append(allIDs, id)
	}

	dir := gfile.Dir(filePath)
	if !gfile.Exists(dir) {
		if err := gfile.Mkdir(dir); err != nil {
			return err
		}
	}

	data, err := json.MarshalIndent(allIDs, "", "  ")
	if err != nil {
		return err
	}
	return gfile.PutContents(filePath, string(data))
}

// ResetDailySentFarmingPools 重置四条链今天的 farming 已推送记录
func (s *kyberSwapImpl) ResetDailySentFarmingPools(ctx context.Context) error {
	for _, chain := range FarmingChains() {
		filePath := farmingSentFilePath(chain.ID)
		dir := gfile.Dir(filePath)
		if !gfile.Exists(dir) {
			if err := gfile.Mkdir(dir); err != nil {
				return err
			}
		}
		if err := gfile.PutContents(filePath, "[]"); err != nil {
			return err
		}
		g.Log().Info(ctx, fmt.Sprintf("重置 farming 已推送记录: %s", filePath))
	}
	return nil
}
