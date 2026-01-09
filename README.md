# GoFrame 项目

基于 GoFrame 框架构建的完整项目，包含 API 接口、定时任务、日志配置等功能。

## 项目结构

```
.
├── main.go                 # 主入口文件
├── go.mod                  # Go 模块依赖
├── config/                 # 配置文件目录
│   └── config.yaml        # 应用配置文件
├── manifest/               # 部署清单目录
│   └── config/            # 部署配置文件
├── internal/               # 内部代码目录
│   ├── controller/        # 控制器层
│   │   ├── user.go        # 用户控制器
│   │   ├── router.go      # 路由注册
│   │   └── middleware.go  # 中间件
│   ├── service/           # 服务层
│   │   └── user.go        # 用户服务
│   ├── dao/               # 数据访问层
│   │   └── user.go        # 用户数据访问
│   ├── model/             # 数据模型
│   │   └── user.go        # 用户模型
│   └── task/              # 定时任务
│       ├── task.go        # 任务管理器
│       └── jobs/           # 具体任务
│           └── example.go # 示例任务
└── logs/                  # 日志目录（自动创建）
```

## 功能特性

### 1. API 接口
- RESTful API 设计
- 用户管理接口（CRUD）
- 健康检查接口
- 统一的响应格式
- 请求参数验证

### 2. 定时任务
- 基于 GoFrame gcron 的定时任务系统
- 支持 Cron 表达式
- 示例任务：
  - 每5分钟执行一次的任务
  - 每天凌晨2点执行的任务
  - 每小时执行一次的任务

### 3. 日志系统
- 可配置的日志级别
- 日志文件自动切分
- 日志压缩和保留策略
- 请求日志中间件
- 支持控制台和文件输出

### 4. 中间件
- CORS 跨域支持
- 请求日志记录
- 错误处理

## 快速开始

### 1. 安装依赖

```bash
go mod tidy
```

### 2. 初始化数据库（可选）

项目使用 SQLite 作为默认数据库，首次运行会自动创建数据库文件。

如果需要使用 MySQL 或其他数据库，请修改 `config/config.yaml` 中的数据库配置：

```yaml
database:
  default:
    link: "mysql:user:password@tcp(127.0.0.1:3306)/database"
```

### 3. 创建数据库表

在首次运行前，需要创建用户表。可以执行以下 SQL：

```sql
CREATE TABLE IF NOT EXISTS `users` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `name` VARCHAR(100) NOT NULL,
  `email` VARCHAR(100) NOT NULL,
  `phone` VARCHAR(20) NOT NULL,
  `created_at` DATETIME NOT NULL,
  `updated_at` DATETIME NOT NULL
);
```

### 4. 运行项目

```bash
go run main.go
```

或者编译后运行：

```bash
go build -o app main.go
./app
```

### 5. 访问接口

- 健康检查: `http://localhost:8000/api/v1/health`
- 用户列表: `http://localhost:8000/api/v1/users`
- 用户详情: `http://localhost:8000/api/v1/users/{id}`

## API 接口文档

### 用户管理接口

#### 1. 获取用户列表
```
GET /api/v1/users?page=1&size=10
```

#### 2. 获取用户详情
```
GET /api/v1/users/{id}
```

#### 3. 创建用户
```
POST /api/v1/users
Content-Type: application/json

{
  "name": "张三",
  "email": "zhangsan@example.com",
  "phone": "13800138000"
}
```

#### 4. 更新用户
```
PUT /api/v1/users/{id}
Content-Type: application/json

{
  "name": "李四",
  "email": "lisi@example.com"
}
```

#### 5. 删除用户
```
DELETE /api/v1/users/{id}
```

## 配置说明

### 服务器配置
- `address`: 服务器监听地址
- `readTimeout`: 读取超时时间
- `writeTimeout`: 写入超时时间

### 日志配置
- `path`: 日志文件存储路径
- `level`: 日志级别（DEBUG, INFO, NOTICE, WARNING, ERROR, CRITICAL）
- `format`: 日志格式（json, text）
- `stdout`: 是否输出到控制台
- `fileSize`: 单个日志文件大小限制
- `fileDays`: 日志文件保留天数
- `compress`: 是否压缩旧日志文件

### 定时任务配置
在 `internal/task/task.go` 中配置定时任务，支持标准的 Cron 表达式。

## 开发建议

1. **添加新接口**：在 `internal/controller/` 中创建控制器，在 `router.go` 中注册路由
2. **添加新服务**：在 `internal/service/` 中实现业务逻辑
3. **添加新任务**：在 `internal/task/jobs/` 中创建任务函数，在 `task.go` 中注册
4. **数据库操作**：使用 `internal/dao/` 层进行数据访问

## 注意事项

1. 确保 Go 版本 >= 1.21
2. 首次运行前需要创建数据库表
3. 日志文件会自动创建在 `logs/` 目录
4. 配置文件支持环境变量覆盖

## 许可证

MIT License
