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
	GetTodaySentPoolIDs(ctx context.Context) (map[string]bool, error)
	AddSentPoolIDs(ctx context.Context, poolIDs []string) error
	ResetDailySentPools(ctx context.Context) error
}
type kyberSwapImpl struct{}
var kyberSwapService = kyberSwapImpl{}
// KyberSwap 获取 KyberSwap 服务实例
func KyberSwap() IKyberSwap {
	return &kyberSwapService
}
// earnServicePoolsURL Kyber Earn 池子列表 API（接口可能较慢，超时时间较长）
const earnServicePoolsURL = "https://earn-service.kyberswap.com/api/v1/explorer/pools?chainIds=8453%%2C56&page=%d&limit=100&interval=24h&protocol=&tag=high_apr&sortBy=&orderBy=&q="

// FetchPools 获取指定页面的池子数据（仅保留 tokens 中 symbol 不包含 WETH 的池子）
func (s *kyberSwapImpl) FetchPools(ctx context.Context, page int) ([]model.Pool, error) {
	url := fmt.Sprintf(earnServicePoolsURL, page)
	client := &http.Client{
		Timeout: 90 * time.Second, // 接口可能较慢，延长等待
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
// FetchAllPools 获取池子数据（仅拉取 page=1）
func (s *kyberSwapImpl) FetchAllPools(ctx context.Context) ([]model.Pool, error) {
	return s.FetchPools(ctx, 1)
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
// exchangeToShort 将 API 的 exchange 转为短名（如 univ4）
func exchangeToShort(exchange string) string {
	ex := strings.ToLower(exchange)
	switch {
	case strings.Contains(ex, "uniswap-v4"), strings.Contains(ex, "uniswapv4"):
		return "univ4"
	case strings.Contains(ex, "uniswap-v3"), strings.Contains(ex, "uniswapv3"):
		return "univ3"
	case strings.Contains(ex, "uniswapv2"):
		return "univ2"
	case strings.Contains(ex, "pancake-infinity"):
		return "pancake-infinity"
	case strings.Contains(ex, "pancake-v3"), strings.Contains(ex, "pancakev3"):
		return "pv3"
	case strings.Contains(ex, "pancake"):
		return "pancake"
	case strings.Contains(ex, "kyber"):
		return "kyber"
	default:
		if exchange != "" {
			return exchange
		}
		return "未知"
	}
}

// chainNameDisplay 将 chain.name 转为展示用（base -> Base, bsc -> BNB）
func chainNameDisplay(name string) string {
	switch strings.ToLower(name) {
	case "bsc":
		return "BNB"
	case "base":
		return "Base"
	case "":
		return "未知"
	default:
		if len(name) > 0 {
			return strings.ToUpper(name[:1]) + strings.ToLower(name[1:])
		}
		return name
	}
}

// FormatPoolMessage 格式化池子消息用于 Telegram（按约定格式）
func FormatPoolMessage(pool model.Pool) string {
	tokenPair := fmt.Sprintf("%s / %s", pool.Token0Symbol, pool.Token1Symbol)
	if tokenPair == " / " && pool.Name != "" {
		tokenPair = strings.Replace(pool.Name, "/", " / ", 1)
	}
	protocolDisplay := pool.Protocol
	if protocolDisplay == "" {
		protocolDisplay = "-"
	}
	feeText := fmt.Sprintf("%.2f%%", pool.FeeTier)
	chainDisplay := chainNameDisplay(pool.ChainName)
	volText := fmt.Sprintf("$%.2f", pool.Volume24h)
	feesText := fmt.Sprintf("$%.2f", pool.Fees24h)
	var b strings.Builder
	// 重点字段用 *粗体* 高亮
	b.WriteString(fmt.Sprintf("🌐 *代币名称*：*%s*\n\n", tokenPair))
	b.WriteString(fmt.Sprintf("🌉 来源: %s\n\n", chainDisplay))
	b.WriteString(fmt.Sprintf("📈 APR: 🔥 %s\n\n", formatAPR(pool.APR)))
	b.WriteString(fmt.Sprintf("📈 协议: %s\n\n", protocolDisplay))
	b.WriteString(fmt.Sprintf("💰 费率: %s\n\n", feeText))
	b.WriteString(fmt.Sprintf("💎 TVL: %s\n\n", formatTVL(pool.TVL)))
	b.WriteString(fmt.Sprintf("📊 *24h交易量*：*%s*\n\n", volText))
	b.WriteString(fmt.Sprintf("💵 *24h手续费*：*%s*\n", feesText))
	if pool.ContractAddress != "" {
		b.WriteString(fmt.Sprintf("\n📋 合约地址（长按复制）：\n\n`%s`\n", pool.ContractAddress))
	}
	return b.String()
}
// FormatPoolsMessage 格式化多个池子消息（带序号与分隔）
// 发现 1 个新池子时采用 PUMP 金狗提醒格式标题
func FormatPoolsMessage(pools []model.Pool, isFirstRun bool) string {
	if len(pools) == 0 {
		return ""
	}
	var builder strings.Builder
		// 用全角空格使标题视觉居中（Telegram 无原生居中）
		builder.WriteString("　　　　🔴🔴  高收益流动性提醒 🔴🔴\n\n")
		if isFirstRun && len(pools) != 1 {
			builder.WriteString(fmt.Sprintf("🎉 *首次运行 | %d 个池子*\n\n", len(pools)))
		} else if !isFirstRun && len(pools) != 1 {
			builder.WriteString(fmt.Sprintf("✨ *发现 %d 个新池子*\n\n", len(pools)))
		}
	for i, pool := range pools {
		builder.WriteString(fmt.Sprintf("▸ *【%d】*\n\n", i+1))
		builder.WriteString(FormatPoolMessage(pool))
		if i < len(pools)-1 {
			builder.WriteString("\n\n━━━━━━━━━━━━━━━━━━━━\n\n")
		}
	}
	return builder.String()
}
// hasWETH 判断 tokens 数组中是否包含 symbol 为 WETH 的代币
func hasWETH(tokens []interface{}) bool {
	for _, t := range tokens {
		m, ok := t.(map[string]interface{})
		if !ok {
			continue
		}
		sym, _ := m["symbol"].(string)
		if strings.EqualFold(sym, "WETH") {
			return true
		}
	}
	return false
}

// hasUSDTOrUSDC 判断 tokens 中是否包含 USDT 或 USDC（至少一个即可）
func hasUSDTOrUSDC(tokens []interface{}) bool {
	for _, t := range tokens {
		m, ok := t.(map[string]interface{})
		if !ok {
			continue
		}
		sym, _ := m["symbol"].(string)
		lower := strings.ToLower(sym)
		if lower == "usdt" || lower == "usdc" {
			return true
		}
	}
	return false
}

// parsePoolFromInterface 从 interface{} 解析池子数据（仅保留 tokens 中 symbol 不包含 WETH 的池子）
// 字段映射：tvl->TVL, earnFee->Fees24h, feeTier->费率%, liquidity->总流动性, exchange->协议, apr->APR
// 合约地址取 tokens 中 symbol 不为 USDT/USDC 的 address；代币名称取 tokens 的 symbol
func (s *kyberSwapImpl) parsePoolFromInterface(data interface{}) *model.Pool {
	poolMap, ok := data.(map[string]interface{})
	if !ok {
		return nil
	}

	tokens, _ := poolMap["tokens"].([]interface{})
	if len(tokens) < 2 {
		return nil
	}
	if hasWETH(tokens) {
		return nil // 过滤：排除含 WETH 的池子
	}
	if !hasUSDTOrUSDC(tokens) {
		return nil // 过滤：只推送 tokens 中包含 USDT 或 USDC（至少一个）的池子
	}

	pool := &model.Pool{}

	if id, ok := poolMap["address"].(string); ok && id != "" {
		pool.ID = id
	} else if id, ok := poolMap["id"].(string); ok && id != "" {
		pool.ID = id
	} else if id, ok := poolMap["poolId"].(string); ok && id != "" {
		pool.ID = id
	} else {
		return nil
	}

	if apr, ok := poolMap["apr"].(float64); ok {
		pool.APR = apr
	} else if apr, ok := poolMap["apy"].(float64); ok {
		pool.APR = apr
	}
	if tvl, ok := poolMap["tvl"].(float64); ok {
		pool.TVL = tvl
	}
	if liq, ok := poolMap["liquidity"].(float64); ok {
		pool.Liquidity = liq
	}
	if vol, ok := poolMap["volume"].(float64); ok {
		pool.Volume24h = vol
	}
	if earnFee, ok := poolMap["earnFee"].(float64); ok {
		pool.Fees24h = earnFee
	}
	if feeTier, ok := poolMap["feeTier"].(float64); ok {
		pool.FeeTier = feeTier
	} else if feeTier, ok := poolMap["feeTier"].(int); ok {
		pool.FeeTier = float64(feeTier)
	}
	if exchange, ok := poolMap["exchange"].(string); ok {
		pool.Protocol = exchange
	}
	if chain, ok := poolMap["chain"].(map[string]interface{}); ok {
		if id, ok := chain["id"].(float64); ok {
			pool.ChainID = int(id)
		}
		if name, ok := chain["name"].(string); ok {
			pool.ChainName = name
		}
	} else if chainId, ok := poolMap["chainId"].(float64); ok {
		pool.ChainID = int(chainId)
	}

	// tokens: 代币名称用 symbol，合约地址取 symbol 不为 USDT/USDC 的 address
	var syms []string
	for i, t := range tokens {
		m, ok := t.(map[string]interface{})
		if !ok {
			continue
		}
		addr, _ := m["address"].(string)
		sym, _ := m["symbol"].(string)
		syms = append(syms, sym)
		symLower := strings.ToLower(sym)
		if symLower != "usdt" && symLower != "usdc" {
			pool.ContractAddress = addr
		}
		if i == 0 {
			pool.Token0 = addr
			pool.Token0Symbol = sym
		} else if i == 1 {
			pool.Token1 = addr
			pool.Token1Symbol = sym
		}
	}
	if len(syms) >= 2 {
		pool.Name = fmt.Sprintf("%s/%s", syms[0], syms[1])
	}

	if pool.ID != "" {
		pool.URL = fmt.Sprintf("https://kyberswap.com/earn/pools/%s", pool.ID)
	}
	return pool
}
// GetTodaySentPoolIDs 获取今天已推送的池子ID列表
func (s *kyberSwapImpl) GetTodaySentPoolIDs(ctx context.Context) (map[string]bool, error) {
	today := time.Now().Format("2006-01-02")
	filePath := fmt.Sprintf("data/sent_pools_%s.json", today)
	
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
	
	poolIDMap := make(map[string]bool)
	for _, id := range poolIDs {
		poolIDMap[id] = true
	}
	
	return poolIDMap, nil
}
// AddSentPoolIDs 添加已推送的池子ID到今天的记录中
func (s *kyberSwapImpl) AddSentPoolIDs(ctx context.Context, poolIDs []string) error {
	if len(poolIDs) == 0 {
		return nil
	}
	
	today := time.Now().Format("2006-01-02")
	filePath := fmt.Sprintf("data/sent_pools_%s.json", today)
	
	// 获取今天已有的池子ID
	existingMap, err := s.GetTodaySentPoolIDs(ctx)
	if err != nil {
		return err
	}
	
	// 添加新的池子ID（去重）
	for _, id := range poolIDs {
		existingMap[id] = true
	}
	
	// 转换为数组
	allIDs := make([]string, 0, len(existingMap))
	for id := range existingMap {
		allIDs = append(allIDs, id)
	}
	
	// 确保目录存在
	dir := gfile.Dir(filePath)
	if !gfile.Exists(dir) {
		if err := gfile.Mkdir(dir); err != nil {
			return err
		}
	}
	
	// 保存到文件
	data, err := json.MarshalIndent(allIDs, "", "  ")
	if err != nil {
		return err
	}
	
	return gfile.PutContents(filePath, string(data))
}
// ResetDailySentPools 重置每天的已推送记录（在每天0点执行）
func (s *kyberSwapImpl) ResetDailySentPools(ctx context.Context) error {
	// 获取今天的日期，清空今天的已推送记录
	today := time.Now().Format("2006-01-02")
	filePath := fmt.Sprintf("data/sent_pools_%s.json", today)
	
	// 确保目录存在
	dir := gfile.Dir(filePath)
	if !gfile.Exists(dir) {
		if err := gfile.Mkdir(dir); err != nil {
			return err
		}
	}
	
	// 重置文件为空数组
	emptyData := "[]"
	if err := gfile.PutContents(filePath, emptyData); err != nil {
		return err
	}
	
	g.Log().Info(ctx, fmt.Sprintf("重置今天的已推送记录: %s", filePath))
	return nil
}