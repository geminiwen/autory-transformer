.PHONY: help test test-quick test-thinking test-stream test-error test-setup verify clean build run docker-build docker-run

# 默认目标
.DEFAULT_GOAL := help

# 颜色定义
BLUE := \033[0;34m
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
NC := \033[0m # No Color

help: ## 显示帮助信息
	@echo "$(BLUE)DeepSeek R1 API 代理 - Makefile 命令$(NC)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-18s$(NC) %s\n", $$1, $$2}'
	@echo ""

test-setup: ## 配置测试环境（创建 .env.test）
	@if [ ! -f .env.test ]; then \
		cp .env.test.example .env.test; \
		echo "$(YELLOW)⚠️  已创建 .env.test 文件，请编辑并填写你的 API Key$(NC)"; \
		echo ""; \
		echo "编辑文件: nano .env.test"; \
		echo "或者:     vim .env.test"; \
		echo ""; \
	else \
		echo "$(GREEN)✓ .env.test 已存在$(NC)"; \
	fi

verify: test-setup ## 验证 ARK Endpoint 配置（直接调用 ARK API）
	@echo "$(BLUE)🔍 验证配置...$(NC)"
	@chmod +x verify_config.sh
	@bash verify_config.sh

test: test-setup ## 运行完整测试套件
	@echo "$(BLUE)🧪 运行完整测试套件...$(NC)"
	@bash test.sh

test-quick: test-setup ## 快速测试（仅基础功能）
	@echo "$(BLUE)⚡ 快速测试模式$(NC)"
	@bash test.sh quick

test-thinking: test-setup ## 测试 Extended Thinking 功能
	@echo "$(BLUE)🧠 Thinking 模式专项测试$(NC)"
	@bash test.sh thinking

test-stream: test-setup ## 测试流式响应
	@echo "$(BLUE)🌊 流式响应专项测试$(NC)"
	@bash test.sh stream

test-error: test-setup ## 测试错误处理
	@echo "$(BLUE)⚠️  错误处理专项测试$(NC)"
	@bash test.sh error

clean: ## 清理构建产物
	@echo "$(YELLOW)🧹 清理构建产物...$(NC)"
	@rm -rf output/
	@rm -f test.log
	@echo "$(GREEN)✓ 清理完成$(NC)"

build: ## 编译项目
	@echo "$(BLUE)🔨 编译项目...$(NC)"
	@mkdir -p output
	@go build -o output/anthropic-to-ark
	@echo "$(GREEN)✓ 编译完成: ./output/anthropic-to-ark$(NC)"

run: build ## 运行服务
	@echo "$(BLUE)🚀 启动服务...$(NC)"
	@./output/anthropic-to-ark

dev: ## 开发模式（自动重载）
	@echo "$(BLUE)💻 开发模式$(NC)"
	@go run main.go

docker-build: ## 构建 Docker 镜像
	@echo "$(BLUE)🐳 构建 Docker 镜像...$(NC)"
	@docker build -t anthropic-to-ark .
	@echo "$(GREEN)✓ Docker 镜像构建完成$(NC)"

docker-run: docker-build ## 运行 Docker 容器
	@echo "$(BLUE)🚀 启动 Docker 容器...$(NC)"
	@docker run -p 3000:3000 \
		-e ARK_BASE_URL=https://ark-ap-southeast.byteintl.net/api/v3 \
		-e ARK_ENDPOINT_THINKING=ep-20250424174745-w6pgh \
		-e ARK_ENDPOINT_DEFAULT=ep-20250424174745-w6pgh \
		anthropic-to-ark

deps: ## 下载依赖
	@echo "$(BLUE)📦 下载依赖...$(NC)"
	@go mod download
	@echo "$(GREEN)✓ 依赖下载完成$(NC)"

tidy: ## 整理依赖
	@echo "$(BLUE)📦 整理依赖...$(NC)"
	@go mod tidy
	@echo "$(GREEN)✓ 依赖整理完成$(NC)"

fmt: ## 格式化代码
	@echo "$(BLUE)✨ 格式化代码...$(NC)"
	@go fmt ./...
	@echo "$(GREEN)✓ 代码格式化完成$(NC)"

lint: ## 代码检查
	@echo "$(BLUE)🔍 代码检查...$(NC)"
	@go vet ./...
	@echo "$(GREEN)✓ 代码检查完成$(NC)"

all: clean deps fmt lint build test ## 执行完整构建流程

install: build ## 安装到系统
	@echo "$(BLUE)📦 安装到系统...$(NC)"
	@sudo cp output/anthropic-to-ark /usr/local/bin/
	@echo "$(GREEN)✓ 安装完成: /usr/local/bin/anthropic-to-ark$(NC)"

uninstall: ## 从系统卸载
	@echo "$(YELLOW)🗑️  从系统卸载...$(NC)"
	@sudo rm -f /usr/local/bin/anthropic-to-ark
	@echo "$(GREEN)✓ 卸载完成$(NC)"
