package model

// Pool 流动性池信息
type Pool struct {
	ID          string  `json:"id"`           // 池子ID
	Name        string  `json:"name"`         // 池子名称
	APR         float64 `json:"apr"`          // 年化收益率
	TVL         float64 `json:"tvl"`          // 总锁定价值
	ChainID     int     `json:"chain_id"`     // 链ID
	Token0      string  `json:"token0"`       // 代币0
	Token1      string  `json:"token1"`       // 代币1
	Token0Symbol string `json:"token0_symbol"` // 代币0符号
	Token1Symbol string `json:"token1_symbol"` // 代币1符号
	Volume24h   float64 `json:"volume_24h"`   // 24小时交易量
	Fees24h     float64 `json:"fees_24h"`     // 24小时手续费
	URL         string  `json:"url"`          // 池子链接
	Version     string  `json:"version"`       // 池子版本：v3 或 v4
	FeeTier     int     `json:"fee_tier"`     // 费率等级：1 或 3
	Protocol    string  `json:"protocol"`     // 协议类型：Uniswap, Pancake, KyberSwap 等
}

// PoolList 池子列表
type PoolList struct {
	Pools []Pool `json:"pools"`
}
