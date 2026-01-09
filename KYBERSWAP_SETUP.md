# KyberSwap 监控功能使用说明

## 功能说明

本功能会每30秒自动监控 KyberSwap 的高 APR 池子（BSC 和 Base 链），当发现新池子时，会自动发送美观的通知到 Telegram。

## 配置步骤

### 1. 创建 Telegram Bot

1. 在 Telegram 中搜索 `@BotFather`
2. 发送 `/newbot` 命令
3. 按照提示设置 Bot 名称和用户名
4. 获取 Bot Token（格式类似：`123456789:ABCdefGHIjklMNOpqrsTUVwxyz`）

### 2. 获取 Chat ID

#### 方式一：个人聊天（私聊）

1. 在 Telegram 中搜索你创建的 Bot
2. 发送任意消息给 Bot（如 `/start`）
3. 访问：`https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getUpdates`
4. 在返回的 JSON 中找到 `"chat":{"id":123456789}`，这个数字就是你的 Chat ID（通常是正数）

#### 方式二：群组聊天（推荐）

1. **将 Bot 添加到群组**
   - 在群组中，点击群组名称 → 添加成员 → 搜索你的 Bot 用户名 → 添加

2. **给 Bot 管理员权限（可选但推荐）**
   - 群组设置 → 管理员 → 添加管理员 → 选择你的 Bot
   - 至少需要"发送消息"权限

3. **在群组中发送一条消息**
   - 可以是任意消息，比如 `/start` 或 `test`

4. **获取群组 Chat ID**

   **方法一：使用项目提供的 API 接口（推荐）**
   
   1. 确保项目已运行，并且已在 `config/config.yaml` 中配置了 `telegram.botToken`
   2. 在群组中发送任意消息给 Bot（如 `/start` 或 `test`）
   3. 访问：`http://localhost:8000/api/v1/telegram/updates`
   4. 在返回的 JSON 中查找群组信息，`id` 字段就是群组的 Chat ID
   
   **方法二：直接访问 Telegram API**
   
   1. 在群组中发送任意消息给 Bot
   2. 访问：`https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getUpdates`
      - 将 `<YOUR_BOT_TOKEN>` 替换为你的实际 Bot Token
      - 注意：URL 中不要包含 `<` 和 `>` 符号
   3. 在返回的 JSON 中找到 `"chat":{"id":-1001234567890}`，这个数字就是群组的 Chat ID
   4. **注意**：群组的 Chat ID 通常是负数（如 `-1001234567890`）
   
   **方法三：使用第三方 Bot**
   
   - 在群组中发送 `/start@getidsbot`（使用 @getidsbot 这个 Bot）
   - 或者使用 @RawDataBot，它会显示群组的完整信息

5. **验证 Chat ID**
   
   访问：`http://localhost:8000/api/v1/telegram/chat/<CHAT_ID>`
   - 将 `<CHAT_ID>` 替换为你获取的 Chat ID
   - 如果返回成功，说明 Chat ID 正确

### 3. 配置项目

编辑 `config/config.yaml` 文件，填写 Telegram 配置：

```yaml
telegram:
  botToken: "你的Bot Token"
  chatId: "你的Chat ID或群组ID"  # 个人聊天用正数，群组用负数（如 -1001234567890）
```

**重要提示**：
- 如果发送到**个人聊天**：Chat ID 是正数（如 `123456789`）
- 如果发送到**群组**：Chat ID 是负数（如 `-1001234567890`）
- 群组 Chat ID 必须以 `-100` 开头
- 确保 Bot 已添加到群组中

### 4. 运行项目

```bash
go run main.go
```

## 功能特性

- ✅ 每30秒自动监控（从 page=1 到 page=10）
- ✅ 自动检测新池子
- ✅ 美观的 Telegram 消息格式（支持 Markdown）
- ✅ 自动保存历史数据
- ✅ 消息过长时自动分批发送

## 消息格式示例

```
🎉 发现 2 个新池子

1. 🎯 新发现高 APR 池子

📊 USDC/USDT
💰 APR: 125.50%
💎 TVL: $2.50M
🔄 交易对: USDC / USDT
⛓️ 链: BSC
📈 24h 交易量: $1.20M
💵 24h 手续费: $500.00K

🔗 查看详情

---

2. 🎯 新发现高 APR 池子

📊 ETH/BTC
💰 APR: 98.75%
💎 TVL: $5.00M
🔄 交易对: ETH / BTC
⛓️ 链: Base
📈 24h 交易量: $2.50M
💵 24h 手续费: $1.00M

🔗 查看详情
```

## 数据存储

池子数据会保存在 `data/kyberswap_pools.json` 文件中，用于比较新旧数据。

## 注意事项

1. **API 端点**：代码中使用的 API 端点可能需要根据 KyberSwap 的实际 API 调整
2. **请求频率**：每30秒执行一次，每次获取10页数据，请确保不会触发 API 限流
3. **网络连接**：需要能够访问 KyberSwap 网站和 Telegram API
4. **首次运行**：首次运行时会获取所有数据作为基准，不会发送通知

## 故障排查

### 问题1：无法获取数据

- 检查网络连接
- 查看日志文件 `logs/` 目录
- 确认 KyberSwap API 是否可访问

### 问题2：Telegram 消息发送失败

- 检查 Bot Token 是否正确
- 检查 Chat ID 是否正确
  - 个人聊天：Chat ID 是正数
  - 群组聊天：Chat ID 是负数（如 `-1001234567890`）
- 确认 Bot 已启动（发送 `/start` 给 Bot）
- **如果是群组**：
  - 确认 Bot 已添加到群组中
  - 确认 Bot 有发送消息的权限
  - 在群组中先发送一条消息给 Bot，确保 Bot 能收到消息
- 查看日志中的错误信息


### 问题3：没有检测到新池子

- 这是正常的，只有当有新池子出现时才会发送通知
- 可以查看 `data/kyberswap_pools.json` 确认数据是否正常获取

### 问题4：如何测试消息格式？

如果你想测试消息格式效果，可以使用测试接口：

1. **确保已配置 Chat ID**
   - 在 `config/config.yaml` 中设置 `telegram.chatId`
   - 确保项目正在运行

2. **发送测试消息**

   **方法一：浏览器访问（最简单）**
   - 直接在浏览器中访问：`http://localhost:8000/api/v1/telegram/test`
   - 会立即发送测试消息到 Telegram 群组

   **方法二：使用 curl**
   ```bash
   curl http://localhost:8000/api/v1/telegram/test
   # 或使用 POST
   curl -X POST http://localhost:8000/api/v1/telegram/test
   ```

   **方法三：使用测试脚本**
   ```bash
   ./scripts/test_message.sh
   ```

3. **查看效果**
   - 测试消息会发送到配置的 Telegram 群组
   - 消息包含 3 个示例池子，展示了完整的消息格式
   - 如果返回成功，说明配置正确，请查看 Telegram 群组

## API 端点说明

代码中使用以下 API 端点：
- `https://zap-earn-service-v3.kyberengineering.io/api/v1/explorer/pools`

API 参数：
- `chainIds=56%2C8453` (BSC 和 Base 链)
- `page=1-10` (页码)
- `limit=10` (每页数量)
- `tag=high_apr` (高 APR 标签)

如果该端点不可用或格式不同，代码会自动尝试从 HTML 页面解析数据。

## 群组配置示例

### 完整配置示例

```yaml
telegram:
  botToken: "123456789:ABCdefGHIjklMNOpqrsTUVwxyz"
  chatId: "-1001234567890"  # 群组 Chat ID（负数）
```

### 验证配置

运行项目后，如果配置正确，Bot 会向群组发送消息。如果出现错误，检查：

1. **Bot 未添加到群组**
   - 错误信息：`Telegram API 错误: chat not found`
   - 解决：将 Bot 添加到群组

2. **Bot 没有发送消息权限**
   - 错误信息：`Telegram API 错误: bot is not a member of the group chat`
   - 解决：给 Bot 管理员权限，或至少确保 Bot 可以发送消息

3. **Chat ID 格式错误**
   - 错误信息：`Telegram API 错误: bad request: chat not found`
   - 解决：确认 Chat ID 是正确的负数格式（群组）或正数格式（个人）

## 自定义配置

可以在 `config/config.yaml` 中调整：
- 监控间隔（修改任务调度）
- 监控的链（修改 API 请求参数）
- 监控的页数（修改 `FetchAllPools` 中的循环次数）
