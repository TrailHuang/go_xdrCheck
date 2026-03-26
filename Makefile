.PHONY: all build clean test install-deps

# 项目配置
PROJECT_NAME := xdr_check_optimized
MODULE_NAME := github.com/user/go_xdrCheck
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.0.0-dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")

# 构建参数
GO := go
GOFLAGS := -v
LDFLAGS := -w -s -X "main.version=$(VERSION)" -X "main.buildTime=$(BUILD_TIME)" -X "main.gitCommit=$(GIT_COMMIT)"

# 输出目录
BUILD_DIR := build
BIN_DIR := $(BUILD_DIR)/bin

# 默认目标
all: build-linux build-linux-arm64

# 构建Linux x86_64版本
build-linux:
	@echo "Building $(PROJECT_NAME) for linux/amd64..."
	@mkdir -p $(BIN_DIR)/linux/amd64
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/linux/amd64/$(PROJECT_NAME) .
	@echo "Build completed: $(BIN_DIR)/linux/amd64/$(PROJECT_NAME)"

# 构建Linux ARM64版本
build-linux-arm64:
	@echo "Building $(PROJECT_NAME) for linux/arm64..."
	@mkdir -p $(BIN_DIR)/linux/arm64
	GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/linux/arm64/$(PROJECT_NAME) .
	@echo "Build completed: $(BIN_DIR)/linux/arm64/$(PROJECT_NAME)"

# 构建当前平台版本（用于开发）
build:
	@echo "Building $(PROJECT_NAME) for current platform..."
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(PROJECT_NAME) .
	@echo "Build completed: $(BIN_DIR)/$(PROJECT_NAME)"

# 清理构建文件
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@$(GO) clean
	@echo "Clean completed"

# 运行测试
test:
	@echo "Running tests..."
	$(GO) test -v ./...

# 安装依赖
install-deps:
	@echo "Installing dependencies..."
	$(GO) mod download
	$(GO) mod verify

# 代码格式化
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

# 显示版本信息
version:
	@echo "Project: $(PROJECT_NAME)"
	@echo "Module: $(MODULE_NAME)"
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Go Version: $(shell go version)"

# 显示帮助信息
help:
	@echo "Available targets:"
	@echo "  all               - 构建Linux x86_64和ARM64版本（默认）"
	@echo "  build-linux       - 构建Linux x86_64版本"
	@echo "  build-linux-arm64 - 构建Linux ARM64版本"
	@echo "  build             - 构建当前平台版本"
	@echo "  test              - 运行测试"
	@echo "  clean             - 清理构建文件"
	@echo "  install-deps      - 安装依赖"
	@echo "  fmt               - 格式化代码"
	@echo "  version           - 显示版本信息"
	@echo "  help              - 显示此帮助信息"

# 默认目标
.DEFAULT_GOAL := all