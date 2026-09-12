# Telegram Bot 群组发送消息指南

## 创建 4 个链群组 + 获取 Bot ID（farming 按链推送）

旧的 `high_apr` 单群推送继续走 `telegram.chatId`，不要改。下面只给 **按链拆群的新推送** 建 4 个群。

可以直接复用现在配置里的同一个 Bot，不必再申请新 Bot。

### A. 查看 Bot ID 和用户名

项目跑起来后访问：

```
http://localhost:8000/api/v1/telegram/bot
```

返回里的：

- `id`：Bot 数字 ID（例如 `7195943995`）
- `username`：Bot 用户名（添加群成员时搜索 `@username`）

也可以在 Telegram 搜 `@BotFather`，发 `/mybots`，选中你的 Bot 查看。

Token 格式是 `BotID:密钥`，冒号前面那串就是 Bot ID。当前项目已配置 Token，一般不用重建 Bot。

如果还没有 Bot：

1. Telegram 搜索 `@BotFather`
2. 发送 `/newbot`
3. 按提示设置名称和用户名（用户名必须以 `bot` 结尾）
4. 复制 Token，填到 `config/config.yaml` 的 `telegram.botToken`

### B. 创建 4 个群组

在 Telegram 里各建一个群，建议命名：

1. `ETH Farming`
2. `Base Farming`
3. `BSC Farming`
4. `Robinhood Farming`

步骤（每个群做一遍）：

1. 打开 Telegram → 左上菜单 / 铅笔 → **新建群组**
2. 先把自己加进去，填群名
3. 创建后点群名称 → **添加成员** → 搜索 `@你的Bot用户名` → 添加
4. 群设置 → **管理员** → 把 Bot 设为管理员，至少打开 **发送消息**
5. 在群里随便发一条消息（例如 `test`），让 Bot 能看到这个群

### C. 获取每个群的 Chat ID

Bot 被加进群并收到一条消息后，访问：

```
http://localhost:8000/api/v1/telegram/updates
```

在返回的 `chats` 里找对应群名，`id` 就是 Chat ID（超级群通常是 `-100` 开头的负数）。

也可以浏览器打开（把 Token 换成你的）：

```
https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getUpdates
```

找到 `"chat":{"id":-100xxxxxxxxxx,"title":"ETH Farming",...}`。

填到配置（旧的 `chatId` 不要动）：

```yaml
telegram:
  botToken: "你的Bot Token"
  chatId: "-1003643474581"   # 旧推送，保持不变
  farming:
    enabled: true
    eth: "-100..."          # ETH 群
    base: "-100..."         # Base 群
    bsc: "-100..."          # BSC 群
    robinhood: "-100..."    # Robinhood 群
```

改完配置后重启项目。然后可以：

- 预览四条链池子（不推送）：`http://localhost:8000/api/v1/telegram/farming-preview`
- 给已配置的链群发测试消息：`http://localhost:8000/api/v1/telegram/test-farming`
- 只测一条链：`http://localhost:8000/api/v1/telegram/test-farming?chain=eth`

确认 4 个群都能收到后，新推送就算跑通。之后再把旧的 `kyberswap_monitor` 关掉。

---

## 完整步骤（旧单群推送，逻辑不变）

### 1. 将 Bot 添加到群组

1. **打开你的 Telegram 群组**
2. **点击群组名称**（在聊天界面顶部）
3. **点击"添加成员"或"Add members"**
4. **搜索你的 Bot 用户名**（例如：`@your_bot_name`）
5. **选择并添加 Bot 到群组**

### 2. 给 Bot 发送消息权限（重要）

1. **在群组中，点击群组名称**
2. **进入"管理员"或"Administrators"**
3. **点击"添加管理员"或"Add Administrator"**
4. **选择你的 Bot**
5. **至少勾选"发送消息"权限**
6. **保存设置**

**注意**：如果不想给 Bot 管理员权限，至少确保 Bot 在群组中，并且群组允许所有成员发送消息。

### 3. 在群组中发送测试消息

在群组中发送任意消息给 Bot，例如：
- `/start`
- `test`
- `hello`

这一步很重要，因为只有 Bot 收到消息后，才能获取到群组的 Chat ID。

### 4. 获取群组 Chat ID

#### 方法一：使用项目 API（推荐）

1. **确保项目已运行**
   ```bash
   go run main.go
   ```

2. **访问 API 接口**
   ```
   http://localhost:8000/api/v1/telegram/updates
   ```

3. **查看返回的 JSON**
   - 找到 `"type": "group"` 或 `"type": "supergroup"` 的聊天
   - 记录 `"id"` 字段的值（通常是负数，如 `-1001234567890`）

#### 方法二：直接访问 Telegram API

1. **在浏览器中访问**（替换为你的 Bot Token）：
   ```
   https://api.telegram.org/bot7195943995:AAGkxidmBEBY2GzrdhYb2OPEpRiuxY_LtbQ/getUpdates
   ```

2. **查找群组信息**
   - 在返回的 JSON 中搜索 `"type": "group"` 或 `"type": "supergroup"`
   - 找到对应的 `"chat": {"id": -1001234567890}`

### 5. 配置 Chat ID

编辑 `config/config.yaml` 文件：

```yaml
telegram:
  botToken: "7195943995:AAGkxidmBEBY2GzrdhYb2OPEpRiuxY_LtbQ"
  chatId: "-1001234567890"  # 替换为你获取的群组 Chat ID
```

### 6. 测试发送消息

#### 方法一：使用项目 API 测试

访问：
```
http://localhost:8000/api/v1/telegram/chat/-1001234567890
```
（替换为你的 Chat ID）

如果返回成功，说明配置正确。

#### 方法二：重启项目

重启项目后，当监控任务发现新池子时，会自动发送消息到群组。

### 7. 验证消息发送

检查群组中是否收到了 Bot 发送的消息。如果没有收到：

1. **检查 Bot 是否在群组中**
   - 在群组成员列表中查看是否有你的 Bot

2. **检查 Bot 权限**
   - 确保 Bot 有发送消息的权限

3. **检查 Chat ID**
   - 确认 Chat ID 是否正确（群组通常是负数）

4. **查看日志**
   - 查看 `logs/` 目录下的日志文件
   - 查找错误信息

## 常见问题

### Q: Bot 无法发送消息到群组

**A: 可能的原因：**

1. **Bot 没有发送消息权限**
   - 解决：给 Bot 管理员权限，或确保群组允许所有成员发送消息

2. **Chat ID 错误**
   - 解决：重新获取 Chat ID，确保是负数格式

3. **Bot 被移出群组**
   - 解决：重新添加 Bot 到群组

### Q: 如何确认 Chat ID 是否正确？

**A: 使用验证接口：**

访问：`http://localhost:8000/api/v1/telegram/chat/<YOUR_CHAT_ID>`

如果返回成功，说明 Chat ID 正确。

### Q: 群组 Chat ID 是正数还是负数？

**A: 通常是负数**

- 个人聊天：正数（如 `123456789`）
- 群组（group）：负数（如 `-123456789`）
- 超级群组（supergroup）：负数，以 `-100` 开头（如 `-1001234567890`）
- 频道（channel）：负数，以 `-100` 开头

### Q: 如何让 Bot 自动发送消息？

**A: 配置完成后，监控任务会自动发送**

1. 确保 `config/config.yaml` 中配置了正确的 `chatId`
2. 运行项目：`go run main.go`
3. 当监控任务发现新池子时，会自动发送消息到群组

## 快速检查清单

- [ ] Bot 已添加到群组
- [ ] Bot 有发送消息权限
- [ ] 在群组中发送了测试消息给 Bot
- [ ] 获取到了群组 Chat ID（负数）
- [ ] 在 `config/config.yaml` 中配置了 `chatId`
- [ ] 重启了项目
- [ ] 检查日志确认没有错误

## 示例配置

```yaml
telegram:
  botToken: "7195943995:AAGkxidmBEBY2GzrdhYb2OPEpRiuxY_LtbQ"
  chatId: "-1001234567890"  # 你的群组 Chat ID
```

配置完成后，监控任务每30秒检查一次，发现新池子时会自动发送到群组！
