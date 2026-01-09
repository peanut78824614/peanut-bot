package service

import (
	"context"
	"data/internal/model"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gfile"
)

type IKyberSwap interface {
	FetchPools(ctx context.Context, page int) ([]model.Pool, error)
	FetchAllPools(ctx context.Context) ([]model.Pool, error)
	GetStoredPools(ctx context.Context) ([]model.Pool, error)
	SavePools(ctx context.Context, pools []model.Pool) error
	ComparePools(oldPools, newPools []model.Pool) []model.Pool
}

type kyberSwapImpl struct{}

var kyberSwapService = kyberSwapImpl{}

// KyberSwap 获取 KyberSwap 服务实例
func KyberSwap() IKyberSwap {
	return &kyberSwapService
}

// FetchPools 获取指定页面的池子数据
func (s *kyberSwapImpl) FetchPools(ctx context.Context, page int) ([]model.Pool, error) {
	// KyberSwap API 端点
	url := fmt.Sprintf("https://zap-earn-service-v3.kyberengineering.io/api/v1/explorer/pools?chainIds=56%%2C8453&page=%d&limit=10&interval=24h&protocol=&tag=high_apr&sortBy=&orderBy=&q=", page)
	
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json")
	
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	// 先尝试解析为通用结构，查看实际响应格式
	var rawData map[string]interface{}
	if err := json.Unmarshal(body, &rawData); err != nil {
		g.Log().Error(ctx, "JSON解析失败:", err)
		g.Log().Debug(ctx, "响应内容:", string(body))
		return nil, err
	}
	
	// 尝试多种可能的响应格式
	pools := make([]model.Pool, 0)
	parseFailedCount := 0
	
	// 格式1: { data: { pools: [...] } }
	if data, ok := rawData["data"].(map[string]interface{}); ok {
		if poolsData, ok := data["pools"].([]interface{}); ok {
			g.Log().Info(ctx, fmt.Sprintf("格式1: 找到 %d 个池子数据", len(poolsData)))
			for i, p := range poolsData {
				if pool := s.parsePoolFromInterface(p); pool != nil {
					pools = append(pools, *pool)
				} else {
					parseFailedCount++
					if i < 3 { // 只记录前3个失败的，避免日志过多
						g.Log().Debug(ctx, fmt.Sprintf("解析第 %d 个池子失败", i+1))
					}
				}
			}
		}
		// 格式1变体: { data: [...] } 直接是数组
		if len(pools) == 0 {
			if poolsData, ok := data[""].([]interface{}); ok {
				g.Log().Info(ctx, fmt.Sprintf("格式1变体: 找到 %d 个池子数据", len(poolsData)))
				for i, p := range poolsData {
					if pool := s.parsePoolFromInterface(p); pool != nil {
						pools = append(pools, *pool)
					} else {
						parseFailedCount++
						if i < 3 {
							g.Log().Debug(ctx, fmt.Sprintf("解析第 %d 个池子失败", i+1))
						}
					}
				}
			}
		}
	}
	
	// 格式2: { pools: [...] }
	if len(pools) == 0 {
		if poolsData, ok := rawData["pools"].([]interface{}); ok {
			g.Log().Info(ctx, fmt.Sprintf("格式2: 找到 %d 个池子数据", len(poolsData)))
			for i, p := range poolsData {
				if pool := s.parsePoolFromInterface(p); pool != nil {
					pools = append(pools, *pool)
				} else {
					parseFailedCount++
					if i < 3 {
						g.Log().Debug(ctx, fmt.Sprintf("解析第 %d 个池子失败", i+1))
					}
				}
			}
		}
	}
	
	// 格式3: 直接是数组 [...]
	if len(pools) == 0 {
		// 尝试直接解析为数组
		var poolsArray []interface{}
		if err := json.Unmarshal(body, &poolsArray); err == nil && len(poolsArray) > 0 {
			g.Log().Info(ctx, fmt.Sprintf("格式3: 找到 %d 个池子数据", len(poolsArray)))
			for i, p := range poolsArray {
				if pool := s.parsePoolFromInterface(p); pool != nil {
					pools = append(pools, *pool)
				} else {
					parseFailedCount++
					if i < 3 {
						g.Log().Debug(ctx, fmt.Sprintf("解析第 %d 个池子失败", i+1))
					}
				}
			}
		}
	}
	
	if len(pools) == 0 {
		g.Log().Warning(ctx, "未能解析出池子数据，响应格式可能不同")
		bodyLen := len(body)
		previewLen := 500
		if bodyLen < previewLen {
			previewLen = bodyLen
		}
		g.Log().Debug(ctx, "响应内容前500字符:", string(body[:previewLen]))
		// 打印 rawData 的键，帮助调试
		keys := make([]string, 0, len(rawData))
		for k := range rawData {
			keys = append(keys, k)
		}
		g.Log().Debug(ctx, "响应数据键:", keys)
		return []model.Pool{}, nil
	}
	
	if parseFailedCount > 0 {
		g.Log().Warning(ctx, fmt.Sprintf("成功解析 %d 个池子，失败 %d 个", len(pools), parseFailedCount))
	} else {
		g.Log().Info(ctx, fmt.Sprintf("成功解析 %d 个池子", len(pools)))
	}
	
	return pools, nil
}


// FetchAllPools 获取所有页面的池子数据（page 1-10）
func (s *kyberSwapImpl) FetchAllPools(ctx context.Context) ([]model.Pool, error) {
	allPools := make([]model.Pool, 0)
	poolMap := make(map[string]bool) // 用于去重
	
	for page := 1; page <= 10; page++ {
		g.Log().Info(ctx, fmt.Sprintf("正在获取第 %d 页数据...", page))
		
		pools, err := s.FetchPools(ctx, page)
		if err != nil {
			g.Log().Error(ctx, fmt.Sprintf("获取第 %d 页数据失败:", page), err)
			continue
		}
		
		// 去重
		for _, pool := range pools {
			if !poolMap[pool.ID] {
				poolMap[pool.ID] = true
				allPools = append(allPools, pool)
			}
		}
		
		// 避免请求过快
		time.Sleep(500 * time.Millisecond)
	}
	
	return allPools, nil
}

// GetStoredPools 获取存储的池子数据
func (s *kyberSwapImpl) GetStoredPools(ctx context.Context) ([]model.Pool, error) {
	filePath := "data/kyberswap_pools.json"
	
	if !gfile.Exists(filePath) {
		return []model.Pool{}, nil
	}
	
	content := gfile.GetContents(filePath)
	
	var pools []model.Pool
	if err := json.Unmarshal([]byte(content), &pools); err != nil {
		return nil, err
	}
	
	return pools, nil
}

// SavePools 保存池子数据
func (s *kyberSwapImpl) SavePools(ctx context.Context, pools []model.Pool) error {
	filePath := "data/kyberswap_pools.json"
	
	// 确保目录存在
	dir := gfile.Dir(filePath)
	if !gfile.Exists(dir) {
		if err := gfile.Mkdir(dir); err != nil {
			return err
		}
	}
	
	data, err := json.MarshalIndent(pools, "", "  ")
	if err != nil {
		return err
	}
	
	return gfile.PutContents(filePath, string(data))
}

// ComparePools 比较新旧池子数据，返回新增的池子
func (s *kyberSwapImpl) ComparePools(oldPools, newPools []model.Pool) []model.Pool {
	oldMap := make(map[string]bool)
	for _, pool := range oldPools {
		oldMap[pool.ID] = true
	}
	
	newPoolsList := make([]model.Pool, 0)
	for _, pool := range newPools {
		if !oldMap[pool.ID] {
			newPoolsList = append(newPoolsList, pool)
		}
	}
	
	return newPoolsList
}

// formatAPR 格式化 APR
func formatAPR(apr float64) string {
	if apr >= 1000 {
		return fmt.Sprintf("%.2f%%", apr)
	} else if apr >= 100 {
		return fmt.Sprintf("%.1f%%", apr)
	} else {
		return fmt.Sprintf("%.2f%%", apr)
	}
}

// formatTVL 格式化 TVL
func formatTVL(tvl float64) string {
	if tvl >= 1000000 {
		return fmt.Sprintf("$%.2fM", tvl/1000000)
	} else if tvl >= 1000 {
		return fmt.Sprintf("$%.2fK", tvl/1000)
	} else {
		return fmt.Sprintf("$%.2f", tvl)
	}
}

// FormatPoolMessage 格式化池子消息用于 Telegram
func FormatPoolMessage(pool model.Pool) string {
	var chainName string
	var chainColor string
	switch pool.ChainID {
	case 56:
		chainName = "BSC"
		chainColor = "🟡"
	case 8453:
		chainName = "Base"
		chainColor = "🔵"
	default:
		chainName = fmt.Sprintf("Chain %d", pool.ChainID)
		chainColor = "⚪"
	}
	
	// 协议版本标签 - 显示完整协议名称
	var versionLabel string
	protocol := strings.ToLower(pool.Protocol)
	version := strings.ToLower(pool.Version)
	
	if strings.Contains(protocol, "uniswap") {
		if version == "v4" || strings.Contains(version, "4") {
			versionLabel = "🟢 Uniswap V4"
		} else {
			versionLabel = "🟠 Uniswap V3"
		}
	} else if strings.Contains(protocol, "pancake") {
		if version == "v4" || strings.Contains(version, "4") {
			versionLabel = "🟣 Pancake V4"
		} else {
			versionLabel = "🟡 Pancake V3"
		}
	} else if strings.Contains(protocol, "kyber") {
		if version == "v4" || strings.Contains(version, "4") {
			versionLabel = "🔵 KyberSwap V4"
		} else {
			versionLabel = "🟠 KyberSwap V3"
		}
	} else {
		// 默认根据版本显示
		if version == "v4" || strings.Contains(version, "4") {
			versionLabel = "🟢 " + pool.Protocol + " V4"
		} else {
			versionLabel = "🟠 " + pool.Protocol + " V3"
		}
		if pool.Protocol == "" {
			if version == "v4" {
				versionLabel = "🟢 V4"
			} else {
				versionLabel = "🟠 V3"
			}
		}
	}
	
	// 费率标签
	var feeLabel string
	if pool.FeeTier == 1 {
		feeLabel = "🔵 Fee: 0.01%"
	} else if pool.FeeTier == 3 {
		feeLabel = "🟢 Fee: 1%"
	} else if pool.FeeTier > 0 {
		feeLabel = fmt.Sprintf("⚪ Fee: %d", pool.FeeTier)
	} else {
		feeLabel = ""
	}
	
	// APR 颜色标签
	var aprColor string
	if pool.APR >= 200 {
		aprColor = "🔥" // 超高
	} else if pool.APR >= 100 {
		aprColor = "🟢" // 高
	} else if pool.APR >= 50 {
		aprColor = "🟡" // 中等
	} else {
		aprColor = "⚪" // 普通
	}
	
	var builder strings.Builder
	// 标题行 - 使用颜色标签和粗体增大字体
	builder.WriteString(fmt.Sprintf("%s *%s*  %s %s\n", aprColor, pool.Name, chainColor, chainName))
	
	// 第二行：版本、费率（增加间距）
	infoLine := versionLabel
	if feeLabel != "" {
		infoLine += "    " + feeLabel // 增加间距
	}
	builder.WriteString(infoLine + "\n")
	
	// 第三行：交易对（单独一行，更清晰）
	tokenPair := fmt.Sprintf("%s/%s", pool.Token0Symbol, pool.Token1Symbol)
	builder.WriteString(fmt.Sprintf("💱 *%s*\n\n", tokenPair))
	
	// 核心数据 - 使用粗体增大字体，不使用代码块
	builder.WriteString(fmt.Sprintf("💰 *APR:*     %s %s\n", aprColor, formatAPR(pool.APR)))
	builder.WriteString(fmt.Sprintf("💎 *TVL:*     %s\n", formatTVL(pool.TVL)))
	if pool.Volume24h > 0 {
		builder.WriteString(fmt.Sprintf("📈 *Volume:*  %s\n", formatTVL(pool.Volume24h)))
	}
	if pool.Fees24h > 0 {
		builder.WriteString(fmt.Sprintf("💵 *Fees:*    %s\n", formatTVL(pool.Fees24h)))
	}
	
	return builder.String()
}

// FormatPoolsMessage 格式化多个池子消息
func FormatPoolsMessage(pools []model.Pool, isFirstRun bool) string {
	if len(pools) == 0 {
		return ""
	}
	
	var builder strings.Builder
	
	// 简洁标题
	if isFirstRun {
		builder.WriteString(fmt.Sprintf("🎉 *首次运行 | %d 个池子*\n\n", len(pools)))
	} else {
		builder.WriteString(fmt.Sprintf("✨ *发现 %d 个新池子*\n\n", len(pools)))
	}
	
	// 池子列表 - 用横线分隔
	for i, pool := range pools {
		builder.WriteString(fmt.Sprintf("*[%d]* ", i+1))
		builder.WriteString(FormatPoolMessage(pool))
		// 在池子之间添加横线分隔（最后一个不添加，使用更粗的横线）
		if i < len(pools)-1 {
			builder.WriteString("━━━━━━━━━━━━━━━━━━━━\n\n")
		}
	}
	
	return builder.String()
}

// parsePoolFromInterface 从 interface{} 解析池子数据
func (s *kyberSwapImpl) parsePoolFromInterface(data interface{}) *model.Pool {
	poolMap, ok := data.(map[string]interface{})
	if !ok {
		return nil
	}
	
	pool := &model.Pool{}
	
	// 解析 ID - 实际API使用 "address" 字段
	if id, ok := poolMap["address"].(string); ok && id != "" {
		pool.ID = id
	} else if id, ok := poolMap["id"].(string); ok && id != "" {
		pool.ID = id
	} else if id, ok := poolMap["poolId"].(string); ok && id != "" {
		pool.ID = id
	} else if id, ok := poolMap["pool_id"].(string); ok && id != "" {
		pool.ID = id
	} else if id, ok := poolMap["id"].(float64); ok {
		pool.ID = fmt.Sprintf("%.0f", id)
	} else if id, ok := poolMap["poolId"].(float64); ok {
		pool.ID = fmt.Sprintf("%.0f", id)
	} else {
		// ID 是必需的
		return nil
	}
	
	// 解析名称
	if name, ok := poolMap["name"].(string); ok {
		pool.Name = name
	} else if token0, ok := poolMap["token0"].(map[string]interface{}); ok {
		if token1, ok := poolMap["token1"].(map[string]interface{}); ok {
			symbol0, _ := token0["symbol"].(string)
			symbol1, _ := token1["symbol"].(string)
			pool.Name = fmt.Sprintf("%s/%s", symbol0, symbol1)
		}
	}
	
	// 解析 APR
	if apr, ok := poolMap["apr"].(float64); ok {
		pool.APR = apr
	} else if apr, ok := poolMap["apy"].(float64); ok {
		pool.APR = apr
	} else if aprStr, ok := poolMap["apr"].(string); ok {
		fmt.Sscanf(aprStr, "%f", &pool.APR)
	}
	
	// 解析 TVL
	if tvl, ok := poolMap["tvl"].(float64); ok {
		pool.TVL = tvl
	} else if tvl, ok := poolMap["totalValueLocked"].(float64); ok {
		pool.TVL = tvl
	} else if tvlStr, ok := poolMap["tvl"].(string); ok {
		fmt.Sscanf(tvlStr, "%f", &pool.TVL)
	}
	
	// 解析 ChainID
	if chainId, ok := poolMap["chainId"].(float64); ok {
		pool.ChainID = int(chainId)
	} else if chainId, ok := poolMap["chainId"].(int); ok {
		pool.ChainID = chainId
	} else if chainId, ok := poolMap["chain_id"].(float64); ok {
		pool.ChainID = int(chainId)
	}
	
	// 解析 Token0 和 Token1 - 实际API使用 "tokens" 数组
	if tokens, ok := poolMap["tokens"].([]interface{}); ok && len(tokens) >= 2 {
		// Token0
		if token0, ok := tokens[0].(map[string]interface{}); ok {
			if addr, ok := token0["address"].(string); ok {
				pool.Token0 = addr
			}
			if symbol, ok := token0["symbol"].(string); ok {
				pool.Token0Symbol = symbol
			}
		}
		// Token1
		if token1, ok := tokens[1].(map[string]interface{}); ok {
			if addr, ok := token1["address"].(string); ok {
				pool.Token1 = addr
			}
			if symbol, ok := token1["symbol"].(string); ok {
				pool.Token1Symbol = symbol
			}
		}
		// 生成名称
		if pool.Name == "" && pool.Token0Symbol != "" && pool.Token1Symbol != "" {
			pool.Name = fmt.Sprintf("%s/%s", pool.Token0Symbol, pool.Token1Symbol)
		}
	} else {
		// 兼容旧格式：token0 和 token1 对象
		if token0, ok := poolMap["token0"].(map[string]interface{}); ok {
			if addr, ok := token0["address"].(string); ok {
				pool.Token0 = addr
			}
			if symbol, ok := token0["symbol"].(string); ok {
				pool.Token0Symbol = symbol
			}
		}
		if token1, ok := poolMap["token1"].(map[string]interface{}); ok {
			if addr, ok := token1["address"].(string); ok {
				pool.Token1 = addr
			}
			if symbol, ok := token1["symbol"].(string); ok {
				pool.Token1Symbol = symbol
			}
		}
	}
	
	// 解析 Volume24h
	if volume, ok := poolMap["volume24h"].(float64); ok {
		pool.Volume24h = volume
	} else if volume, ok := poolMap["volume24H"].(float64); ok {
		pool.Volume24h = volume
	} else if volume, ok := poolMap["volume"].(float64); ok {
		pool.Volume24h = volume
	}
	
	// 解析 Fees24h
	if fees, ok := poolMap["fees24h"].(float64); ok {
		pool.Fees24h = fees
	} else if fees, ok := poolMap["fees24H"].(float64); ok {
		pool.Fees24h = fees
	} else if fees, ok := poolMap["fees"].(float64); ok {
		pool.Fees24h = fees
	}
	
	// 解析协议信息 - 实际API使用 "exchange" 字段
	if exchange, ok := poolMap["exchange"].(string); ok {
		// 标准化协议名称
		exchangeLower := strings.ToLower(exchange)
		if strings.Contains(exchangeLower, "uniswap-v4") || strings.Contains(exchangeLower, "uniswapv4") {
			pool.Protocol = "Uniswap"
			pool.Version = "v4"
		} else if strings.Contains(exchangeLower, "uniswap-v3") || strings.Contains(exchangeLower, "uniswapv3") {
			pool.Protocol = "Uniswap"
			pool.Version = "v3"
		} else if strings.Contains(exchangeLower, "pancake-v3") || strings.Contains(exchangeLower, "pancakev3") {
			pool.Protocol = "Pancake"
			pool.Version = "v3"
		} else if strings.Contains(exchangeLower, "pancake-infinity") || strings.Contains(exchangeLower, "pancake-infinity-cl") {
			pool.Protocol = "Pancake"
			pool.Version = "v3" // Pancake Infinity 通常视为 v3
		} else if strings.Contains(exchangeLower, "kyber") {
			pool.Protocol = "KyberSwap"
		} else {
			pool.Protocol = exchange // 使用原始值
		}
	} else if protocol, ok := poolMap["protocol"].(string); ok {
		pool.Protocol = protocol
	} else if protocol, ok := poolMap["protocolName"].(string); ok {
		pool.Protocol = protocol
	} else {
		pool.Protocol = "" // 未知协议
	}
	
	// 解析版本信息
	if version, ok := poolMap["version"].(string); ok {
		pool.Version = version
	} else if version, ok := poolMap["poolVersion"].(string); ok {
		pool.Version = version
	} else if version, ok := poolMap["v"].(string); ok {
		pool.Version = version
	} else {
		// 尝试从 ID 或名称中提取版本信息
		idLower := strings.ToLower(pool.ID)
		nameLower := strings.ToLower(pool.Name)
		if strings.Contains(idLower, "v4") || strings.Contains(nameLower, "v4") {
			pool.Version = "v4"
		} else if strings.Contains(idLower, "v3") || strings.Contains(nameLower, "v3") {
			pool.Version = "v3"
		} else {
			pool.Version = "v3" // 默认 v3
		}
	}
	
	// 解析费率等级 - 实际API的 feeTier 可能是小数，需要映射
	if feeTier, ok := poolMap["feeTier"].(float64); ok {
		// 根据费率值映射到标准费率等级
		// 0.01% -> 1, 1% -> 3, 其他值保持原值或映射
		if feeTier >= 0.009 && feeTier <= 0.011 {
			pool.FeeTier = 1 // 0.01%
		} else if feeTier >= 0.99 && feeTier <= 1.01 {
			pool.FeeTier = 3 // 1%
		} else {
			pool.FeeTier = int(feeTier) // 其他值直接转换
		}
	} else if feeTier, ok := poolMap["feeTier"].(int); ok {
		pool.FeeTier = feeTier
	} else if feeTier, ok := poolMap["fee_tier"].(float64); ok {
		pool.FeeTier = int(feeTier)
	} else if fee, ok := poolMap["fee"].(float64); ok {
		// 如果提供的是费率百分比，转换为费率等级
		if fee == 0.01 {
			pool.FeeTier = 1
		} else if fee == 1.0 {
			pool.FeeTier = 3
		} else {
			pool.FeeTier = int(fee)
		}
	} else {
		pool.FeeTier = 0 // 未知
	}
	
	// 生成 URL
	if pool.ID != "" {
		pool.URL = fmt.Sprintf("https://kyberswap.com/earn/pools/%s", pool.ID)
	}
	
	return pool
}
