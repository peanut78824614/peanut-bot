package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// PoolInfo 池子信息
type PoolInfo struct {
	PoolID      common.Hash `json:"poolId"`
	Token0      string      `json:"token0"`
	Token1      string      `json:"token1"`
	Token0Name  string      `json:"token0Name"`
	Token1Name  string      `json:"token1Name"`
	Liquidity   *big.Float  `json:"liquidity"`
	Volume24h   *big.Float  `json:"volume24h"`
	Fees24h     *big.Float  `json:"fees24h"`
	APR         *big.Float  `json:"apr"`
	TVL         *big.Float  `json:"tvl"`
	LastUpdated time.Time   `json:"lastUpdated"`
}

// UniswapV4Monitor Uniswap V4监控器
type UniswapV4Monitor struct {
	client       *ethclient.Client
	poolManager  common.Address
	knownPools   map[common.Hash]*PoolInfo
	updateTicker *time.Ticker

	// Telegram 推送配置
	telegramToken string
	telegramChat  string
}

// BinanceAlphaToken 币安 Alpha 监控对象
type BinanceAlphaToken struct {
	Symbol              string
	LastPrice           float64
	PriceChangePercent  float64
	QuoteVolume         float64
}

// BinanceMonitor 监控币安现货行情
type BinanceMonitor struct {
	client       *http.Client
	knownSymbols map[string]time.Time // 用于识别“上新”（程序启动后首次出现的交易对）
}

// NewBinanceMonitor 创建币安监控器
func NewBinanceMonitor() *BinanceMonitor {
	return &BinanceMonitor{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		knownSymbols: make(map[string]time.Time),
	}
}

// PoolManager ABI片段（用于监听PoolCreated事件）
const poolCreatedEventSignature = "PoolCreated(bytes32,address,address,uint24,int24)"

// NewUniswapV4Monitor 创建新的监控器
func NewUniswapV4Monitor(rpcURL string, poolManagerAddr string) (*UniswapV4Monitor, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("连接BSC节点失败: %v", err)
	}

	return &UniswapV4Monitor{
		client:        client,
		poolManager:   common.HexToAddress(poolManagerAddr),
		knownPools:    make(map[common.Hash]*PoolInfo),
		telegramToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		telegramChat:  os.Getenv("TELEGRAM_CHAT_ID"),
	}, nil
}

// fetchBinanceTickers 从币安获取24小时行情数据
func (b *BinanceMonitor) fetchBinanceTickers(ctx context.Context) ([]BinanceAlphaToken, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.binance.com/api/v3/ticker/24hr", nil)
	if err != nil {
		return nil, err
	}

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求币安行情失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("币安行情接口返回状态码 %d, body=%s", resp.StatusCode, string(body))
	}

	var raw []struct {
		Symbol              string `json:"symbol"`
		LastPrice           string `json:"lastPrice"`
		PriceChangePercent  string `json:"priceChangePercent"`
		QuoteVolume         string `json:"quoteVolume"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("解析币安行情响应失败: %v", err)
	}

	var result []BinanceAlphaToken
	for _, t := range raw {
		// 只关注 USDT 计价的现货交易对，过滤掉BUSD/TRY/FDUSD等
		if !strings.HasSuffix(t.Symbol, "USDT") {
			continue
		}

		lp, err1 := strconv.ParseFloat(t.LastPrice, 64)
		pct, err2 := strconv.ParseFloat(t.PriceChangePercent, 64)
		qv, err3 := strconv.ParseFloat(t.QuoteVolume, 64)
		if err1 != nil || err2 != nil || err3 != nil {
			continue
		}

		result = append(result, BinanceAlphaToken{
			Symbol:             t.Symbol,
			LastPrice:          lp,
			PriceChangePercent: pct,
			QuoteVolume:        qv,
		})
	}

	return result, nil
}

// DetectAlphaTokens 识别“上新”和大幅波动的币种
// minChange: 价格24h涨跌幅阈值（绝对值，单位%）
// minQuoteVol: 24h 报价币种成交额阈值（单位：USDT）
func (b *BinanceMonitor) DetectAlphaTokens(ctx context.Context, minChange float64, minQuoteVol float64) (newTokens, bigMovers []BinanceAlphaToken, err error) {
	tickers, err := b.fetchBinanceTickers(ctx)
	if err != nil {
		return nil, nil, err
	}

	now := time.Now()

	for _, t := range tickers {
		// 记录“上新”——程序启动后首次出现的交易对
		if _, exists := b.knownSymbols[t.Symbol]; !exists {
			b.knownSymbols[t.Symbol] = now
			newTokens = append(newTokens, t)
		}

		// 大幅波动币：涨跌幅绝对值 >= minChange 且 成交额 >= minQuoteVol
		if math.Abs(t.PriceChangePercent) >= minChange && t.QuoteVolume >= minQuoteVol {
			bigMovers = append(bigMovers, t)
		}
	}

	return newTokens, bigMovers, nil
}

// PrintBinanceAlphaTokens 打印币安 Alpha 代币信息
func (b *BinanceMonitor) PrintBinanceAlphaTokens(newTokens, bigMovers []BinanceAlphaToken, minChange float64, minQuoteVol float64) {
	if len(newTokens) == 0 && len(bigMovers) == 0 {
		fmt.Println("\n[Binance] 暂无新的上新交易对或大幅波动币。")
		return
	}

	fmt.Println("\n================ Binance Alpha 监控 ================")
	if len(newTokens) > 0 {
		fmt.Println("🆕 新上架交易对（程序启动后首次发现）：")
		for _, t := range newTokens {
			fmt.Printf("- %s  当前价: %.6f USDT  24h 成交额: %.0f USDT\n",
				t.Symbol, t.LastPrice, t.QuoteVolume)
		}
		fmt.Println()
	}

	if len(bigMovers) > 0 {
		fmt.Printf("📈 大幅波动交易对（|24h%%| ≥ %.2f%% 且 24h 成交额 ≥ %.0f USDT）：\n", minChange, minQuoteVol)
		for _, t := range bigMovers {
			dir := "涨"
			if t.PriceChangePercent < 0 {
				dir = "跌"
			}
			fmt.Printf("- %s  方向: %s  幅度: %.2f%%  价: %.6f USDT  24h 成交额: %.0f USDT\n",
				t.Symbol, dir, t.PriceChangePercent, t.LastPrice, t.QuoteVolume)
		}
	}
	fmt.Println("===================================================\n")
}

// StartAlphaMonitor 启动币安 Alpha 监控
func (b *BinanceMonitor) StartAlphaMonitor(ctx context.Context, interval time.Duration, minChange float64, minQuoteVol float64) {
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				newTokens, bigMovers, err := b.DetectAlphaTokens(ctx, minChange, minQuoteVol)
				if err != nil {
					log.Printf("[Binance] 监控失败: %v", err)
					continue
				}
				b.PrintBinanceAlphaTokens(newTokens, bigMovers, minChange, minQuoteVol)
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}

// GetPoolsFromTheGraph 从The Graph获取池子列表（如果可用）
func (m *UniswapV4Monitor) GetPoolsFromTheGraph() ([]*PoolInfo, error) {
	// The Graph API endpoint for Uniswap V4 on BSC (需要根据实际情况调整)
	// 注意：BSC上的Uniswap V4子图可能尚未部署，这里使用通用端点
	graphURL := "https://api.thegraph.com/subgraphs/name/uniswap/uniswap-v4-bsc"
	
	query := `{
		"query": "{
			pools(first: 100, orderBy: totalValueLockedUSD, orderDirection: desc) {
				id
				token0 {
					symbol
					id
				}
				token1 {
					symbol
					id
				}
				totalValueLockedUSD
				volumeUSD
				feesUSD
			}
		}"
	}`

	req, err := http.NewRequest("POST", graphURL, strings.NewReader(query))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		// The Graph可能不可用，返回空列表
		log.Printf("警告: 无法从The Graph获取数据: %v", err)
		return []*PoolInfo{}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data struct {
			Pools []struct {
				ID                string `json:"id"`
				Token0            struct {
					Symbol string `json:"symbol"`
					ID     string `json:"id"`
				} `json:"token0"`
				Token1            struct {
					Symbol string `json:"symbol"`
					ID     string `json:"id"`
				} `json:"token1"`
				TotalValueLockedUSD string `json:"totalValueLockedUSD"`
				VolumeUSD           string `json:"volumeUSD"`
				FeesUSD             string `json:"feesUSD"`
			} `json:"pools"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	if len(result.Errors) > 0 {
		log.Printf("GraphQL错误: %v", result.Errors)
		return []*PoolInfo{}, nil
	}

	var pools []*PoolInfo
	for _, p := range result.Data.Pools {
		tvl, _ := new(big.Float).SetString(p.TotalValueLockedUSD)
		volume, _ := new(big.Float).SetString(p.VolumeUSD)
		fees, _ := new(big.Float).SetString(p.FeesUSD)
		
		// 计算APR: (fees24h / tvl) * 365 * 100
		apr := new(big.Float)
		if tvl != nil && tvl.Sign() > 0 {
			apr.Quo(fees, tvl)
			apr.Mul(apr, big.NewFloat(365))
			apr.Mul(apr, big.NewFloat(100))
		}

		poolID := common.HexToHash(p.ID)
		pool := &PoolInfo{
			PoolID:      poolID,
			Token0:      p.Token0.ID,
			Token1:      p.Token1.ID,
			Token0Name:  p.Token0.Symbol,
			Token1Name:  p.Token1.Symbol,
			TVL:         tvl,
			Volume24h:   volume,
			Fees24h:     fees,
			APR:         apr,
			LastUpdated: time.Now(),
		}
		pools = append(pools, pool)
		m.knownPools[poolID] = pool
	}

	return pools, nil
}

// GetPoolsFromDexScreener 从DexScreener获取Uniswap池子数据（备选方案）
func (m *UniswapV4Monitor) GetPoolsFromDexScreener() ([]*PoolInfo, error) {
	// DexScreener API - 获取BSC链上的Uniswap池子
	// 使用搜索端点，这是更可靠的方法
	// 注意：DexScreener的搜索API会返回所有链的数据，我们需要通过chainId过滤
	endpoints := []string{
		"https://api.dexscreener.com/latest/dex/search?q=uniswap",
		"https://api.dexscreener.com/latest/dex/tokens/0x55d398326f99059fF775485246999027B3197955", // BSC USDT
		"https://api.dexscreener.com/latest/dex/tokens/0xe9e7CEA3DedcA5984780Bafc599bD69ADd087D56", // BSC BUSD
		"https://api.dexscreener.com/latest/dex/tokens/0xbb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c", // BSC WBNB
	}
	
	var lastErr error
	for _, endpoint := range endpoints {
		pools, err := m.tryDexScreenerEndpoint(endpoint)
		if err == nil && len(pools) > 0 {
			return pools, nil
		}
		lastErr = err
		log.Printf("尝试端点 %s 失败: %v", endpoint, err)
		// 在重试前等待一下
		time.Sleep(1 * time.Second)
	}
	
	return nil, fmt.Errorf("所有DexScreener端点都失败，最后错误: %v", lastErr)
}

// tryDexScreenerEndpoint 尝试从指定端点获取数据，带重试机制
func (m *UniswapV4Monitor) tryDexScreenerEndpoint(url string) ([]*PoolInfo, error) {
	maxRetries := 3
	retryDelay := 2 * time.Second
	
	for attempt := 1; attempt <= maxRetries; attempt++ {
		client := &http.Client{
			Timeout: 20 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				IdleConnTimeout:     30 * time.Second,
				DisableKeepAlives:   false,
				MaxIdleConnsPerHost: 2,
			},
		}
		
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; UniswapV4Monitor/1.0)")
		
		resp, err := client.Do(req)
		if err != nil {
			if attempt < maxRetries {
				log.Printf("请求失败 (尝试 %d/%d)，%v 秒后重试...", attempt, maxRetries, retryDelay.Seconds())
				time.Sleep(retryDelay)
				retryDelay *= 2 // 指数退避
				continue
			}
			return nil, fmt.Errorf("请求DexScreener API失败: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			if attempt < maxRetries && resp.StatusCode >= 500 {
				// 服务器错误，可以重试
				log.Printf("服务器错误 %d (尝试 %d/%d)，%v 秒后重试...", resp.StatusCode, attempt, maxRetries, retryDelay.Seconds())
				time.Sleep(retryDelay)
				retryDelay *= 2
				continue
			}
			return nil, fmt.Errorf("DexScreener API返回错误状态码: %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			if attempt < maxRetries {
				log.Printf("读取响应失败 (尝试 %d/%d)，%v 秒后重试...", attempt, maxRetries, retryDelay.Seconds())
				time.Sleep(retryDelay)
				retryDelay *= 2
				continue
			}
			return nil, err
		}

		var result struct {
			Pairs []struct {
				ChainID     string  `json:"chainId"`
				DexID       string  `json:"dexId"`
				PairAddress string  `json:"pairAddress"`
				BaseToken   struct {
					Address string `json:"address"`
					Name    string `json:"name"`
					Symbol  string `json:"symbol"`
				} `json:"baseToken"`
				QuoteToken struct {
					Address string `json:"address"`
					Name    string `json:"name"`
					Symbol  string `json:"symbol"`
				} `json:"quoteToken"`
				Liquidity struct {
					Usd float64 `json:"usd"`
				} `json:"liquidity"`
				Volume struct {
					H24 float64 `json:"h24"`
				} `json:"volume"`
				PriceChange struct {
					H24 float64 `json:"h24"`
				} `json:"priceChange"`
				Fdv         float64 `json:"fdv"`
				PairCreatedAt int64 `json:"pairCreatedAt"`
			} `json:"pairs"`
		}

		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("解析DexScreener响应失败: %v", err)
		}

		var pools []*PoolInfo
		for _, p := range result.Pairs {
			// 过滤：只处理BSC链上的Uniswap池子
			// 注意：如果DexScreener支持V4，可以添加更精确的过滤
			// BSC的ChainID可能是"bsc"、"bsc-mainnet"或"56"（BSC链ID）
			chainIDLower := strings.ToLower(p.ChainID)
			if chainIDLower != "bsc" && chainIDLower != "bsc-mainnet" && p.ChainID != "56" {
				continue
			}
			
			dexIDLower := strings.ToLower(p.DexID)
			if !strings.Contains(dexIDLower, "uniswap") {
				continue
			}

			// 过滤掉流动性太低的池子（小于$1000）
			if p.Liquidity.Usd < 1000 {
				continue
			}

			tvl := big.NewFloat(p.Liquidity.Usd)
			volume := big.NewFloat(p.Volume.H24)
			// 估算24h手续费（Uniswap通常为交易量的0.3%）
			fees := new(big.Float).Mul(volume, big.NewFloat(0.003))
			
			// 计算APR: (fees24h / tvl) * 365 * 100
			apr := new(big.Float)
			if tvl.Sign() > 0 {
				apr.Quo(fees, tvl)
				apr.Mul(apr, big.NewFloat(365))
				apr.Mul(apr, big.NewFloat(100))
			}

			poolID := common.HexToHash(p.PairAddress)
			pool := &PoolInfo{
				PoolID:      poolID,
				Token0:      p.BaseToken.Address,
				Token1:      p.QuoteToken.Address,
				Token0Name:  p.BaseToken.Symbol,
				Token1Name:  p.QuoteToken.Symbol,
				TVL:         tvl,
				Volume24h:   volume,
				Fees24h:     fees,
				APR:         apr,
				LastUpdated: time.Now(),
			}
			pools = append(pools, pool)
			m.knownPools[poolID] = pool
		}

		if len(pools) > 0 {
			log.Printf("从DexScreener获取到 %d 个池子", len(pools))
			return pools, nil
		}
		
		// 如果没有找到池子，返回空列表而不是错误
		return pools, nil
	}
	
	return nil, fmt.Errorf("经过 %d 次重试后仍然失败", maxRetries)
}

// GetPoolsFromKyberZap 从Kyber Zap API获取高APR池子（包含Uniswap V4）
func (m *UniswapV4Monitor) GetPoolsFromKyberZap() ([]*PoolInfo, error) {
	// Kyber Zap explorer API，按高APR筛选；chainIds=56(BSC),8453(Base)
	// 如果只想要BSC，可将chainIds修改为"56"
	url := "https://zap-earn-service-v3.kyberengineering.io/api/v1/explorer/pools?chainIds=56,8453&page=1&limit=50&interval=24h&protocol=&tag=high_apr&sortBy=&orderBy=&q="

	client := &http.Client{Timeout: 20 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; UniswapV4Monitor/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求Kyber Zap API失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Kyber Zap API返回错误状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 解析响应结构
	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Pools []struct {
				Address  string  `json:"address"`
				Apr      float64 `json:"apr"`
				AllApr   float64 `json:"allApr"`
				LpApr    float64 `json:"lpApr"`
				EarnFee  float64 `json:"earnFee"`
				Volume   float64 `json:"volume"`
				Liquidity float64 `json:"liquidity"`
				TVL      float64 `json:"tvl"`
				Exchange string  `json:"exchange"`
				FeeTier  float64 `json:"feeTier"`
				Tokens   []struct {
					Address string `json:"address"`
					Symbol  string `json:"symbol"`
				} `json:"tokens"`
				Chain struct {
					ID   int    `json:"id"`
					Name string `json:"name"`
				} `json:"chain"`
			} `json:"pools"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析Kyber Zap响应失败: %v", err)
	}
	if result.Code != 0 {
		return nil, fmt.Errorf("Kyber Zap API返回错误: code=%d message=%s", result.Code, result.Message)
	}

	var pools []*PoolInfo
	for _, p := range result.Data.Pools {
		// 只取BSC链 (56) 且交易所是Uniswap V4
		if p.Chain.ID != 56 {
			continue
		}
		if !strings.Contains(strings.ToLower(p.Exchange), "uniswap") {
			continue
		}

		tvl := big.NewFloat(p.TVL)
		volume := big.NewFloat(p.Volume)
		fees := big.NewFloat(p.EarnFee) // Kyber返回的earnFee近似手续费收入
		apr := big.NewFloat(p.Apr)      // 直接使用返回的apr（百分比）

		poolID := common.HexToHash(p.Address)
		token0Name, token1Name := "", ""
		token0Addr, token1Addr := "", ""
		if len(p.Tokens) > 0 {
			token0Name = p.Tokens[0].Symbol
			token0Addr = p.Tokens[0].Address
		}
		if len(p.Tokens) > 1 {
			token1Name = p.Tokens[1].Symbol
			token1Addr = p.Tokens[1].Address
		}

		pool := &PoolInfo{
			PoolID:      poolID,
			Token0:      token0Addr,
			Token1:      token1Addr,
			Token0Name:  token0Name,
			Token1Name:  token1Name,
			TVL:         tvl,
			Volume24h:   volume,
			Fees24h:     fees,
			APR:         apr,
			LastUpdated: time.Now(),
		}
		pools = append(pools, pool)
		m.knownPools[poolID] = pool
	}

	if len(pools) > 0 {
		log.Printf("从Kyber Zap获取到 %d 个BSC池子", len(pools))
	}

	return pools, nil
}

// GetPoolDataFromChain 从链上获取池子数据
func (m *UniswapV4Monitor) GetPoolDataFromChain(poolID common.Hash) (*PoolInfo, error) {
	// 这里需要根据Uniswap V4的实际合约接口来实现
	// 由于V4的架构，可能需要调用PoolManager合约的方法
	// 示例：获取池子的流动性、交易量等数据
	
	// 尝试从已知池子获取数据
	if pool, exists := m.knownPools[poolID]; exists {
		return pool, nil
	}

	// 这里可以添加直接调用合约获取数据的逻辑
	// 由于Uniswap V4的具体实现可能不同，这里提供一个框架
	// 需要使用context时，可以添加: ctx := context.Background()
	
	return nil, fmt.Errorf("无法从链上获取池子数据")
}

// ListenToPoolCreatedEvents 监听新池子创建事件
func (m *UniswapV4Monitor) ListenToPoolCreatedEvents(ctx context.Context) error {
	// 创建事件过滤器
	query := ethereum.FilterQuery{
		Addresses: []common.Address{m.poolManager},
		Topics: [][]common.Hash{
			{common.HexToHash("0x783cca1c0412dd0d695e784568c96da2e9c22ff989357a2e8b1d9b2b4e6b7118")}, // PoolCreated事件签名
		},
	}

	logs := make(chan types.Log)
	sub, err := m.client.SubscribeFilterLogs(ctx, query, logs)
	if err != nil {
		return fmt.Errorf("订阅事件失败: %v", err)
	}

	go func() {
		for {
			select {
			case err := <-sub.Err():
				log.Printf("订阅错误: %v", err)
				return
			case vLog := <-logs:
				m.handlePoolCreatedEvent(vLog)
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

// handlePoolCreatedEvent 处理池子创建事件
func (m *UniswapV4Monitor) handlePoolCreatedEvent(vLog types.Log) {
	// 解析事件数据
	// 这里需要根据Uniswap V4的实际事件结构来解析
	log.Printf("检测到新池子创建: %x", vLog.Topics[1])
	
	// 创建新的池子信息
	poolID := vLog.Topics[1]
	pool := &PoolInfo{
		PoolID:      poolID,
		LastUpdated: time.Now(),
	}
	
	m.knownPools[poolID] = pool
}

// UpdatePoolData 更新池子数据
func (m *UniswapV4Monitor) UpdatePoolData() error {
	// 首先尝试从The Graph获取数据
	pools, err := m.GetPoolsFromTheGraph()
	if err != nil {
		log.Printf("从The Graph获取数据失败: %v", err)
	}

	// 如果The Graph不可用或返回空数据，尝试从DexScreener获取
	if len(pools) == 0 {
		log.Println("The Graph无数据，尝试从DexScreener获取...")
		dexPools, dexErr := m.GetPoolsFromDexScreener()
		if dexErr != nil {
			log.Printf("从DexScreener获取数据失败: %v，尝试Kyber Zap...", dexErr)
			// DexScreener失败后尝试Kyber Zap
			kyberPools, kyberErr := m.GetPoolsFromKyberZap()
			if kyberErr != nil {
				log.Printf("从Kyber Zap获取数据也失败: %v (将继续使用已有数据)", kyberErr)
			} else {
				pools = kyberPools
			}
		} else {
			pools = dexPools
		}
	}

	if len(pools) > 0 {
		log.Printf("成功获取 %d 个池子数据", len(pools))
	} else {
		// 如果所有数据源都失败，但已有已知池子，继续使用它们
		if len(m.knownPools) > 0 {
			log.Printf("警告: 未能从数据源获取新数据，使用已有的 %d 个池子", len(m.knownPools))
		} else {
			log.Println("警告: 未能从任何数据源获取池子数据")
		}
	}

	return nil
}

// GetHighYieldPools 获取高收益池子
func (m *UniswapV4Monitor) GetHighYieldPools(minAPR float64, limit int) []*PoolInfo {
	var pools []*PoolInfo
	
	for _, pool := range m.knownPools {
		if pool.APR != nil {
			apr, _ := pool.APR.Float64()
			if apr >= minAPR {
				pools = append(pools, pool)
			}
		}
	}

	// 按APR排序
	sort.Slice(pools, func(i, j int) bool {
		aprI, _ := pools[i].APR.Float64()
		aprJ, _ := pools[j].APR.Float64()
		return aprI > aprJ
	})

	if limit > 0 && limit < len(pools) {
		pools = pools[:limit]
	}

	return pools
}

// Start 启动监控
func (m *UniswapV4Monitor) Start(ctx context.Context, updateInterval time.Duration) {
	// 初始更新
	if err := m.UpdatePoolData(); err != nil {
		log.Printf("初始数据更新失败: %v", err)
	}

	// 启动定时更新
	m.updateTicker = time.NewTicker(updateInterval)
	go func() {
		for {
			select {
			case <-m.updateTicker.C:
				if err := m.UpdatePoolData(); err != nil {
					log.Printf("数据更新失败: %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// 启动事件监听
	if err := m.ListenToPoolCreatedEvents(ctx); err != nil {
		log.Printf("启动事件监听失败: %v", err)
	}
}

// Stop 停止监控
func (m *UniswapV4Monitor) Stop() {
	if m.updateTicker != nil {
		m.updateTicker.Stop()
	}
	m.client.Close()
}

// PrintHighYieldPools 打印高收益池子
func (m *UniswapV4Monitor) PrintHighYieldPools(minAPR float64, limit int) {
	pools := m.GetHighYieldPools(minAPR, limit)
	
	fmt.Println("\n=== 高收益LP池子 ===")
	fmt.Printf("找到 %d 个池子 (最低APR: %.2f%%)\n\n", len(pools), minAPR)
	
	for i, pool := range pools {
		apr, _ := pool.APR.Float64()
		tvl, _ := pool.TVL.Float64()
		volume, _ := pool.Volume24h.Float64()
		fees, _ := pool.Fees24h.Float64()
		
		fmt.Printf("%d. %s/%s\n", i+1, pool.Token0Name, pool.Token1Name)
		fmt.Printf("   池子ID: %s\n", pool.PoolID.Hex())
		fmt.Printf("   APR: %.2f%%\n", apr)
		fmt.Printf("   TVL: $%.2f\n", tvl)
		fmt.Printf("   24h交易量: $%.2f\n", volume)
		fmt.Printf("   24h手续费: $%.2f\n", fees)
		fmt.Printf("   最后更新: %s\n", pool.LastUpdated.Format("2006-01-02 15:04:05"))
		fmt.Println()
	}
}

// buildTelegramMessage 构建推送到 Telegram 的文案
func (m *UniswapV4Monitor) buildTelegramMessage(minAPR float64, limit int) string {
	pools := m.GetHighYieldPools(minAPR, limit)
	if len(pools) == 0 {
		return fmt.Sprintf("*Uniswap V4 BSC 高收益 LP 池子*\n\n当前未找到 APR ≥ %.2f%% 的池子。", minAPR)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("*Uniswap V4 BSC 高收益 LP 池子*\nAPR ≥ %.2f%%，前 %d 个\n\n", minAPR, limit))

	for i, pool := range pools {
		apr, _ := pool.APR.Float64()
		tvl, _ := pool.TVL.Float64()
		volume, _ := pool.Volume24h.Float64()
		fees, _ := pool.Fees24h.Float64()

		sb.WriteString(fmt.Sprintf("%d\\. *%s / %s*\n", i+1, pool.Token0Name, pool.Token1Name))
		sb.WriteString(fmt.Sprintf("• *APR*: `%.2f%%%%`\n", apr))
		sb.WriteString(fmt.Sprintf("• *TVL*: `$%.0f`\n", tvl))
		sb.WriteString(fmt.Sprintf("• *24h 交易量*: `$%.0f`\n", volume))
		sb.WriteString(fmt.Sprintf("• *24h 手续费*: `$%.0f`\n", fees))
		sb.WriteString(fmt.Sprintf("• *Pool*: `%s`\n\n", pool.PoolID.Hex()))
	}

	sb.WriteString("_数据来源: Uniswap V4 + Kyber Zap 高 APR 池接口_")
	return sb.String()
}

// sendTelegramMessage 发送消息到 Telegram 群
func (m *UniswapV4Monitor) sendTelegramMessage(text string) error {
	if m.telegramToken == "" || m.telegramChat == "" {
		// 未配置 Telegram，则直接跳过，不报致命错误
		log.Println("未配置 TELEGRAM_BOT_TOKEN 或 TELEGRAM_CHAT_ID，跳过 Telegram 推送")
		return nil
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", m.telegramToken)

	form := url.Values{}
	form.Set("chat_id", m.telegramChat)
	form.Set("text", text)
	form.Set("parse_mode", "Markdown")

	resp, err := http.PostForm(apiURL, form)
	if err != nil {
		return fmt.Errorf("发送 Telegram 消息失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Telegram API 返回错误状态码: %d, body=%s", resp.StatusCode, string(body))
	}

	return nil
}

func main() {
	// 配置BSC RPC节点（可以使用公共节点或付费服务）
	// 请替换为您的RPC URL，或者使用公共节点
	rpcURL := "https://bsc-dataseed1.binance.org"
	
	// 其他可用的BSC公共RPC节点：
	// rpcURL := "https://bsc-dataseed2.binance.org"
	// rpcURL := "https://bsc-dataseed3.binance.org"
	// rpcURL := "https://bsc-dataseed4.binance.org"
	// rpcURL := "https://rpc.ankr.com/bsc"
	
	// 如果需要使用付费服务，可以使用：
	// rpcURL := "https://bsc-mainnet.g.alchemy.com/v2/YOUR_API_KEY"
	// rpcURL := "https://bsc-mainnet.infura.io/v3/YOUR_INFURA_PROJECT_ID"
	
	// Uniswap V4 PoolManager合约地址（BSC链上，需要根据实际部署地址调整）
	// 注意：Uniswap V4在BSC上的部署情况需要确认
	// 如果地址未知，可以使用零地址，程序仍可通过API获取数据
	poolManagerAddr := "0x0000000000000000000000000000000000000000"
	
	fmt.Println("正在初始化Uniswap V4监控器（BSC链）...")
	monitor, err := NewUniswapV4Monitor(rpcURL, poolManagerAddr)
	if err != nil {
		log.Printf("警告: 连接BSC节点失败，将仅使用API数据源: %v", err)
		// 即使连接失败，仍可以使用API数据源
		monitor = &UniswapV4Monitor{
			knownPools: make(map[common.Hash]*PoolInfo),
		}
	}
	defer func() {
		if monitor.client != nil {
			monitor.Stop()
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动币安 Alpha 代币监控（上新 + 大幅波动）
	binanceMonitor := NewBinanceMonitor()
	// 例如：监控 24h 涨跌幅绝对值 ≥ 10%，且 24h 成交额 ≥ 1000 万 USDT 的交易对
	binanceMinChange := 10.0        // 10%
	binanceMinQuoteVol := 10_000_000.0 // 1000万 USDT
	binanceMonitor.StartAlphaMonitor(ctx, 1*time.Minute, binanceMinChange, binanceMinQuoteVol)
	
	// 启动监控（每5分钟更新一次）
	if monitor.client != nil {
		monitor.Start(ctx, 5*time.Minute)
	} else {
		// 如果没有连接，只使用定时更新
		monitor.updateTicker = time.NewTicker(5 * time.Minute)
		go func() {
			for {
				select {
				case <-monitor.updateTicker.C:
					if err := monitor.UpdatePoolData(); err != nil {
						log.Printf("数据更新失败: %v", err)
					}
				case <-ctx.Done():
					return
				}
			}
		}()
	}
	
	// 初始显示高收益池子并推送到 Telegram
	fmt.Println("正在获取池子数据...")
	if err := monitor.UpdatePoolData(); err != nil {
		log.Printf("初始数据获取失败: %v", err)
	}
	
	time.Sleep(2 * time.Second) // 等待数据加载
	minAPR := 5.0
	limit := 20
	monitor.PrintHighYieldPools(minAPR, limit) // 显示APR >= 5%的前20个池子

	// Telegram 推送
	msg := monitor.buildTelegramMessage(minAPR, limit)
	if err := monitor.sendTelegramMessage(msg); err != nil {
		log.Printf("Telegram 推送失败: %v", err)
	}
	
	// 定期显示更新
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
		fmt.Println("\n监控已启动，每5分钟自动更新并可选推送到 Telegram...")
	fmt.Println("按 Ctrl+C 退出\n")
	
	for {
		select {
		case <-ticker.C:
			fmt.Println("\n" + strings.Repeat("=", 60))
			fmt.Println("数据更新中...")
			if err := monitor.UpdatePoolData(); err != nil {
				log.Printf("数据更新失败: %v", err)
			}
			monitor.PrintHighYieldPools(5.0, 20)
		case <-ctx.Done():
			return
		}
	}
}
