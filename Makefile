.PHONY: run build clean test help

# 默认目标
.DEFAULT_GOAL := help

# 运行项目
run:
	@echo "启动项目..."
	@go run main.go

# 构建项目
build:
	@echo "构建项目..."
	@go build -o app main.go
	@echo "构建完成: ./app"

# 清理构建文件
clean:
	@echo "清理构建文件..."
	@rm -f app
	@rm -rf logs/*
	@echo "清理完成"

# 运行测试
test:
	@echo "运行测试..."
	@go test -v ./...

# 格式化代码
fmt:
	@echo "格式化代码..."
	@go fmt ./...

# 代码检查
lint:
	@echo "代码检查..."
	@golangci-lint run

# 安装依赖
deps:
	@echo "安装依赖..."
	@go mod download
	@go mod tidy

# 初始化数据库（SQLite）
init-db:
	@echo "初始化数据库..."
	@sqlite3 data.db < scripts/init_db.sql
	@echo "数据库初始化完成"

# 帮助信息
help:
	@echo "可用命令:"
	@echo "  make run       - 运行项目"
	@echo "  make build     - 构建项目"
	@echo "  make clean     - 清理构建文件"
	@echo "  make test      - 运行测试"
	@echo "  make fmt       - 格式化代码"
	@echo "  make lint      - 代码检查"
	@echo "  make deps      - 安装依赖"
	@echo "  make init-db   - 初始化数据库"
	@echo "  make help      - 显示帮助信息"
