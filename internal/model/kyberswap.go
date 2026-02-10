package model

// Pool 流动性池信息
type Pool struct {
	ID             string  `json:"id"`               // 池子ID
	Name           string  `json:"name"`             // 池子名称
	APR            float64 `json:"apr"`              // 年化收益率
	TVL            float64 `json:"tvl"`             // 总锁定价值（流动性金额）
	Liquidity      float64 `json:"liquidity"`       // 总流动性（保留两位小数展示）
	ChainID        int     `json:"chain_id"`        // 链ID
	ChainName      string  `json:"chain_name"`      // 链名称，来自 chain.name（如 base, bsc）
	Token0         string  `json:"token0"`         // 代币0 地址
	Token1         string  `json:"token1"`         // 代币1 地址
	Token0Symbol   string  `json:"token0_symbol"`  // 代币0 符号
	Token1Symbol   string  `json:"token1_symbol"`  // 代币1 符号
	Volume24h      float64 `json:"volume_24h"`      // 24小时交易量
	Fees24h        float64 `json:"fees_24h"`       // 24小时手续费 earnFee
	URL            string  `json:"url"`             // 池子链接
	Version        string  `json:"version"`        // 池子版本：v3 或 v4
	FeeTier        float64 `json:"fee_tier"`        // 手续费费率（百分比，如 0.14）
	Protocol       string  `json:"protocol"`        // 协议展示名：univ4, univ3 等
	ContractAddress string `json:"contract_address"` // 非 USDT/USDC 的代币合约地址
}

// PoolList 池子列表
type PoolList struct {
	Pools []Pool `json:"pools"`
}
