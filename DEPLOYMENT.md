# 部署指南

## Systemd 服务配置

### 问题排查：status=203/EXEC 错误

`status=203/EXEC` 错误表示 systemd 无法执行服务文件中指定的可执行文件。常见原因：

1. **可执行文件路径不正确**
2. **可执行文件不存在**
3. **可执行文件没有执行权限**
4. **工作目录配置错误**

### 解决步骤

#### 1. 检查可执行文件

```bash
# 在服务器上检查可执行文件是否存在
ls -la /path/to/your/app/main
# 或
ls -la /path/to/your/app/app

# 确保文件有执行权限
chmod +x /path/to/your/app/main
```

#### 2. 检查 systemd 服务文件

```bash
# 查看服务文件内容
cat /etc/systemd/system/App-PeanutBot.service

# 检查 ExecStart 路径是否正确
# 确保路径是绝对路径，不是相对路径
```

#### 3. 测试可执行文件

```bash
# 手动测试可执行文件是否能运行
/path/to/your/app/main

# 或者如果使用编译后的二进制
/path/to/your/app/app
```

#### 4. 检查工作目录

```bash
# 确保 WorkingDirectory 指向项目根目录
# 这样配置文件才能正确加载
```

#### 5. 查看详细错误信息

```bash
# 查看服务状态
systemctl status App-PeanutBot.service

# 查看详细日志
journalctl -u App-PeanutBot.service -n 50 --no-pager

# 查看启动失败的具体原因
systemctl status App-PeanutBot.service -l
```

### 正确的服务文件配置示例

编辑 `/etc/systemd/system/App-PeanutBot.service`：

```ini
[Unit]
Description=Peanut Bot Service
After=network.target

[Service]
Type=simple
User=root
# 修改为实际的可执行文件绝对路径
ExecStart=/opt/peanutbot/main
# 或者如果使用编译后的二进制文件：
# ExecStart=/opt/peanutbot/app

# 工作目录（必须设置为项目根目录）
WorkingDirectory=/opt/peanutbot

# 环境变量（可选，如果需要指定配置文件路径）
Environment="GF_GCFG_FILE=config/config.yaml"

# 重启策略
Restart=always
RestartSec=5

# 日志配置
StandardOutput=journal
StandardError=journal
SyslogIdentifier=peanutbot

[Install]
WantedBy=multi-user.target
```

### 部署步骤

#### 1. 编译项目

```bash
# 在开发机器上编译（Linux 环境）
GOOS=linux GOARCH=amd64 go build -o app main.go

# 或者使用 Makefile
make build
```

#### 2. 上传到服务器

```bash
# 将编译后的二进制文件上传到服务器
scp app user@server:/opt/peanutbot/

# 上传配置文件
scp -r config/ user@server:/opt/peanutbot/
```

#### 3. 设置权限

```bash
# 在服务器上设置执行权限
chmod +x /opt/peanutbot/app

# 确保工作目录存在
mkdir -p /opt/peanutbot
```

#### 4. 创建 systemd 服务文件

```bash
# 复制服务文件到 systemd 目录
sudo cp App-PeanutBot.service /etc/systemd/system/

# 或者直接编辑
sudo nano /etc/systemd/system/App-PeanutBot.service
```

**重要：** 修改服务文件中的路径：
- `ExecStart`: 改为实际的可执行文件路径（如 `/opt/peanutbot/app`）
- `WorkingDirectory`: 改为项目根目录（如 `/opt/peanutbot`）

#### 5. 重新加载 systemd 并启动服务

```bash
# 重新加载 systemd 配置
sudo systemctl daemon-reload

# 启动服务
sudo systemctl start App-PeanutBot.service

# 设置开机自启
sudo systemctl enable App-PeanutBot.service

# 查看服务状态
sudo systemctl status App-PeanutBot.service
```

### 常见问题

#### 问题 1: 可执行文件路径错误

**错误信息：**
```
Main process exited, code=exited, status=203/EXEC
```

**解决方法：**
- 使用 `which` 或 `whereis` 查找可执行文件位置
- 确保使用绝对路径
- 检查路径中是否有拼写错误

#### 问题 2: 可执行文件没有执行权限

**解决方法：**
```bash
chmod +x /path/to/your/app/main
```

#### 问题 3: 配置文件找不到

**错误信息：**
```
配置文件加载失败
```

**解决方法：**
- 确保 `WorkingDirectory` 设置为项目根目录
- 检查配置文件路径是否正确
- 确保配置文件存在且有读取权限

#### 问题 4: 端口被占用

**错误信息：**
```
bind: address already in use
```

**解决方法：**
```bash
# 查找占用端口的进程
lsof -i :8000
# 或
netstat -tulpn | grep 8000

# 停止占用端口的进程或修改配置文件中的端口
```

### 验证部署

```bash
# 检查服务状态
systemctl status App-PeanutBot.service

# 检查服务是否在运行
ps aux | grep app

# 检查端口是否监听
netstat -tulpn | grep 8000

# 测试 API
curl http://localhost:8000/api/v1/health
```

### 日志查看

```bash
# 查看实时日志
journalctl -u App-PeanutBot.service -f

# 查看最近 100 行日志
journalctl -u App-PeanutBot.service -n 100

# 查看今天的日志
journalctl -u App-PeanutBot.service --since today

# 查看应用日志文件（如果配置了文件日志）
tail -f /opt/peanutbot/logs/$(date +%Y-%m-%d).log
```
