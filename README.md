# Uniswap V4 高收益LP池子监控器（BSC链）

这是一个用Go语言实现的实时监控工具，用于获取BSC（Binance Smart Chain）链上Uniswap V4中流动性高收益的LP池子数据。

## 功能特性

- ✅ 实时获取Uniswap V4池子数据
- ✅ 自动计算池子收益率(APR)
- ✅ 支持多种数据源（The Graph、DexScreener）
- ✅ 监听链上池子创建事件
- ✅ 按收益率排序筛选高收益池子
- ✅ 定时自动更新数据

## 安装依赖

```bash
go mod tidy
```

## 配置

### 1. RPC节点配置

在 `data.go` 的 `main()` 函数中配置您的BSC RPC节点：

```go
// 使用BSC公共节点（推荐）
rpcURL := "https://bsc-dataseed1.binance.org"

// 其他BSC公共节点选项：
// rpcURL := "https://bsc-dataseed2.binance.org"
// rpcURL := "https://rpc.ankr.com/bsc"

// 或使用付费服务（推荐用于生产环境）
// rpcURL := "https://bsc-mainnet.g.alchemy.com/v2/YOUR_API_KEY"
// rpcURL := "https://bsc-mainnet.infura.io/v3/YOUR_INFURA_PROJECT_ID"
```

### 2. Uniswap V4合约地址

如果已知BSC链上Uniswap V4 PoolManager合约地址，请在代码中更新：

```go
poolManagerAddr := "0x..." // 替换为BSC链上的实际地址
```

### 3. Telegram 推送配置（可选）

程序支持把高收益池子列表推送到 Telegram 群，需创建 Telegram Bot 并获取：

- `TELEGRAM_BOT_TOKEN`：Bot 的 Token
- `TELEGRAM_CHAT_ID`：群聊 ID 或个人 ID

在运行前设置环境变量，例如：

```bash
export TELEGRAM_BOT_TOKEN="你的 Bot Token"
export TELEGRAM_CHAT_ID="-100xxxxxxxxxx"  # 群 ID，注意很多群是负数
```

如果未设置这两个环境变量，程序会自动跳过 Telegram 推送，仅在控制台打印。

## 运行

```bash
go run data.go
```

## 输出说明

程序会显示：
- 池子ID
- 代币对（Token0/Token1）
- APR（年化收益率）
- TVL（总锁仓价值）
- 24小时交易量
- 24小时手续费
- 最后更新时间

## 数据源

程序按以下顺序尝试获取数据：

1. **The Graph API** - Uniswap V4 BSC子图（如果可用）
2. **DexScreener API** - BSC链数据源
3. **链上直接查询** - 通过BSC节点查询（需要配置RPC）

## 注意事项

1. Uniswap V4在BSC链上的部署情况需要确认
2. 如果The Graph子图不可用，程序会自动切换到DexScreener API
3. BSC公共节点可能有速率限制，建议使用付费RPC服务以获得更好的性能和稳定性
4. 程序默认每5分钟更新一次数据，可在代码中调整
5. 本程序专门针对BSC链，如需其他链请修改配置

## 自定义配置

### 修改更新频率

```go
monitor.Start(ctx, 5*time.Minute) // 改为您需要的间隔
```

### 修改筛选条件

```go
monitor.PrintHighYieldPools(5.0, 20) // APR >= 5%, 显示前20个
```

## 依赖库

- `github.com/ethereum/go-ethereum` - 以太坊客户端（兼容BSC）
- 标准库：`net/http`, `encoding/json`, `context`, `time` 等

## 许可证

MIT
