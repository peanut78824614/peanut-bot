# Go 安装和项目部署详细步骤

## 一、安装 Go 语言环境

### 1. 下载 Go（推荐使用最新稳定版）

```bash
# 进入临时目录
cd /tmp

# 下载 Go（根据你的系统架构选择，x86_64 使用 amd64）
# 查看系统架构
uname -m

# 如果是 x86_64，下载 amd64 版本
wget https://go.dev/dl/go1.21.6.linux-amd64.tar.gz

# 如果是 ARM64，下载 arm64 版本
# wget https://go.dev/dl/go1.21.6.linux-arm64.tar.gz
```

### 2. 删除旧版本（如果存在）

```bash
# 删除旧版本 Go（如果已安装）
sudo rm -rf /usr/local/go
```

### 3. 解压并安装

```bash
# 解压到 /usr/local 目录
sudo tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz
```

### 4. 配置环境变量

```bash
# 编辑环境变量文件
sudo vim /etc/profile
# 或者使用 nano
# sudo nano /etc/profile

# 在文件末尾添加以下内容：
export PATH=$PATH:/usr/local/go/bin
export GOPATH=$HOME/go
export GOPROXY=https://goproxy.cn,direct  # 使用国内镜像加速

# 保存并退出（vim: 按 ESC，输入 :wq，回车）
# nano: Ctrl+X，然后 Y，回车
```

### 5. 使环境变量生效

```bash
# 重新加载环境变量
source /etc/profile

# 或者重新登录
# 退出 SSH 重新登录
```

### 6. 验证安装

```bash
# 检查 Go 版本
go version

# 应该显示类似：go version go1.21.6 linux/amd64

# 检查 Go 环境
go env
```

## 二、准备项目文件

### 1. 创建项目目录

```bash
# 创建项目目录（根据你的实际情况修改路径）
sudo mkdir -p /opt/peanut-bot
cd /opt/peanut-bot
```

### 2. 上传项目文件

**方式一：使用 git（推荐）**

```bash
# 如果项目在 git 仓库中
git clone <你的仓库地址> /opt/peanut-bot
cd /opt/peanut-bot
```

**方式二：使用 scp 上传**

在本地机器执行：
```bash
# 上传整个项目目录
scp -r /path/to/data root@your-server-ip:/opt/peanut-bot/

# 或者只上传必要文件
scp -r main.go go.mod go.sum internal/ config/ root@your-server-ip:/opt/peanut-bot/
```

### 3. 在服务器上检查文件

```bash
cd /opt/peanut-bot
ls -la
# 应该看到：main.go, go.mod, go.sum, internal/, config/ 等文件
```

## 三、编译项目

### 1. 下载依赖

```bash
cd /opt/peanut-bot

# 设置 Go 代理（加速下载）
go env -w GOPROXY=https://goproxy.cn,direct

# 下载依赖
go mod download

# 整理依赖
go mod tidy
```

### 2. 编译项目

```bash
# 编译为 Linux 可执行文件
go build -o app main.go

# 或者使用 Makefile（如果已上传）
make build

# 检查编译结果
ls -la app
```

### 3. 设置执行权限

```bash
chmod +x app
```

## 四、配置项目

### 1. 检查配置文件

```bash
# 查看配置文件
cat config/config.yaml

# 确保配置文件中的路径和配置正确
# 特别是 Telegram 的 botToken 和 chatId
```

### 2. 创建必要的目录

```bash
# 创建数据目录
mkdir -p data
mkdir -p logs

# 设置权限
chmod 755 data logs
```

## 五、后台运行项目

### 方式一：使用 systemd（推荐，支持开机自启）

#### 1. 创建 systemd 服务文件

```bash
sudo vim /etc/systemd/system/peanut-bot.service
```

#### 2. 添加以下内容：

```ini
[Unit]
Description=Peanut Bot Service
After=network.target

[Service]
Type=simple
User=root
# 可执行文件路径
ExecStart=/opt/peanut-bot/app
# 工作目录（必须设置为项目根目录）
WorkingDirectory=/opt/peanut-bot
# 环境变量（可选）
Environment="GF_GCFG_FILE=config/config.yaml"

# 重启策略
Restart=always
RestartSec=5

# 日志配置
StandardOutput=journal
StandardError=journal
SyslogIdentifier=peanut-bot

[Install]
WantedBy=multi-user.target
```

#### 3. 保存并退出

```bash
# vim: 按 ESC，输入 :wq，回车
```

#### 4. 启动服务

```bash
# 重新加载 systemd 配置
sudo systemctl daemon-reload

# 启动服务
sudo systemctl start peanut-bot.service

# 设置开机自启
sudo systemctl enable peanut-bot.service

# 查看服务状态
sudo systemctl status peanut-bot.service
```

#### 5. 常用管理命令

```bash
# 查看服务状态
systemctl status peanut-bot.service

# 启动服务
systemctl start peanut-bot.service

# 停止服务
systemctl stop peanut-bot.service

# 重启服务
systemctl restart peanut-bot.service

# 查看实时日志
journalctl -u peanut-bot.service -f

# 查看最近 100 行日志
journalctl -u peanut-bot.service -n 100

# 查看今天的日志
journalctl -u peanut-bot.service --since today
```

### 方式二：使用 nohup（简单方式）

```bash
cd /opt/peanut-bot

# 后台运行，输出到 nohup.out
nohup ./app > nohup.out 2>&1 &

# 查看进程
ps aux | grep app

# 查看日志
tail -f nohup.out
```

### 方式三：使用 screen（适合临时运行）

```bash
# 安装 screen（如果没有）
yum install screen -y  # CentOS/RHEL
# 或
apt-get install screen -y  # Ubuntu/Debian

# 创建新的 screen 会话
screen -S peanut-bot

# 在 screen 中运行
cd /opt/peanut-bot
./app

# 按 Ctrl+A 然后按 D 退出 screen（程序继续运行）

# 重新连接 screen
screen -r peanut-bot

# 查看所有 screen 会话
screen -ls
```

## 六、验证部署

### 1. 检查进程

```bash
# 查看进程是否运行
ps aux | grep app

# 应该看到类似：
# root  12345  0.1  0.5  /opt/peanut-bot/app
```

### 2. 检查端口（如果项目有 HTTP 服务）

```bash
# 查看端口监听（根据你的配置修改端口）
netstat -tulpn | grep 8000
# 或
ss -tulpn | grep 8000
```

### 3. 测试 API（如果有 HTTP 服务）

```bash
# 测试健康检查接口
curl http://localhost:8000/api/v1/health
```

### 4. 查看日志

```bash
# systemd 方式
journalctl -u peanut-bot.service -f

# nohup 方式
tail -f /opt/peanut-bot/nohup.out

# 应用日志文件
tail -f /opt/peanut-bot/logs/$(date +%Y-%m-%d).log
```

## 七、常见问题排查

### 问题 1: go: command not found

**原因：** Go 未正确安装或环境变量未生效

**解决：**
```bash
# 检查 Go 是否安装
ls /usr/local/go/bin/go

# 检查环境变量
echo $PATH

# 重新加载环境变量
source /etc/profile

# 或者手动添加
export PATH=$PATH:/usr/local/go/bin
```

### 问题 2: 编译失败

**原因：** 依赖下载失败或网络问题

**解决：**
```bash
# 设置 Go 代理
go env -w GOPROXY=https://goproxy.cn,direct

# 清理并重新下载
go clean -modcache
go mod download
```

### 问题 3: 配置文件找不到

**原因：** 工作目录不正确

**解决：**
```bash
# 确保在项目根目录运行
cd /opt/peanut-bot

# 检查配置文件是否存在
ls -la config/config.yaml

# 如果使用 systemd，确保 WorkingDirectory 设置正确
```

### 问题 4: 端口被占用

**解决：**
```bash
# 查找占用端口的进程
lsof -i :8000
# 或
netstat -tulpn | grep 8000

# 停止占用端口的进程
kill -9 <PID>
```

### 问题 5: 权限问题

**解决：**
```bash
# 确保可执行文件有执行权限
chmod +x /opt/peanut-bot/app

# 确保目录有读写权限
chmod -R 755 /opt/peanut-bot
```

## 八、更新项目

### 1. 停止服务

```bash
systemctl stop peanut-bot.service
```

### 2. 更新代码

```bash
cd /opt/peanut-bot
git pull  # 如果使用 git
# 或重新上传文件
```

### 3. 重新编译

```bash
go build -o app main.go
```

### 4. 启动服务

```bash
systemctl start peanut-bot.service
```

## 九、卸载

### 1. 停止并禁用服务

```bash
systemctl stop peanut-bot.service
systemctl disable peanut-bot.service
```

### 2. 删除服务文件

```bash
sudo rm /etc/systemd/system/peanut-bot.service
sudo systemctl daemon-reload
```

### 3. 删除项目文件

```bash
rm -rf /opt/peanut-bot
```

### 4. 卸载 Go（可选）

```bash
sudo rm -rf /usr/local/go
# 编辑 /etc/profile，删除 Go 相关环境变量
```

---

## 快速命令总结

```bash
# 1. 安装 Go
cd /tmp
wget https://go.dev/dl/go1.21.6.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' | sudo tee -a /etc/profile
source /etc/profile
go version

# 2. 编译项目
cd /opt/peanut-bot
go env -w GOPROXY=https://goproxy.cn,direct
go mod download
go build -o app main.go
chmod +x app

# 3. 创建 systemd 服务
sudo vim /etc/systemd/system/peanut-bot.service
# 复制上面的服务配置内容

# 4. 启动服务
sudo systemctl daemon-reload
sudo systemctl start peanut-bot.service
sudo systemctl enable peanut-bot.service
sudo systemctl status peanut-bot.service
```
