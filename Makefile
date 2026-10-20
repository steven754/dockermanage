.DEFAULT_GOAL := help
SHELL := /bin/bash
BACKEND := backend
FRONTEND := frontend
IMAGE ?= ghcr.io/steven754/dockermanage
TAG ?= latest
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)

.PHONY: help
help: ## 显示所有可用命令
	@grep -hE '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
	  | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

# ---------------------------------------------------------------- 依赖安装
.PHONY: install
install: ## 安装前后端依赖
	cd $(FRONTEND) && npm install
	cd $(BACKEND) && go mod download

# ---------------------------------------------------------------- 本地开发
.PHONY: dev-backend
dev-backend: ## 启动后端（:8080）
	cd $(BACKEND) && go run ./cmd/panel --port 8080 --data ./data --compose-dir ./compose

.PHONY: dev-frontend
dev-frontend: ## 启动前端 dev server（:5173，/api 代理到 :8080）
	cd $(FRONTEND) && npm run dev

.PHONY: dev
dev: ## 同时启动前后端（需另开终端跑 backend）
	@echo "→ 后端: make dev-backend    前端: make dev-frontend"

# ---------------------------------------------------------------- 构建
.PHONY: build
build: build-frontend build-backend ## 构建前后端

.PHONY: build-frontend
build-frontend: ## 构建前端到 frontend/dist
	cd $(FRONTEND) && npm run build

.PHONY: build-backend
build-backend: ## 构建后端二进制到 backend/bin/panel
	cd $(BACKEND) && CGO_ENABLED=0 GOTOOLCHAIN=local go build -trimpath \
	  -ldflags="-s -w \
	    -X github.com/steven754/dockermanage/backend/internal/version.Version=$(VERSION) \
	    -X github.com/steven754/dockermanage/backend/internal/version.Commit=$(COMMIT)" \
	  -o bin/panel ./cmd/panel

# ---------------------------------------------------------------- 质量
.PHONY: test
test: test-backend test-frontend ## 跑全部测试

.PHONY: test-backend
test-backend: ## 后端单测
	cd $(BACKEND) && go test -race -coverprofile=coverage.out ./...

.PHONY: test-frontend
test-frontend: ## 前端类型检查 + 构建
	cd $(FRONTEND) && npm run build

.PHONY: lint
lint: lint-backend lint-frontend ## 跑全部 lint

.PHONY: lint-backend
lint-backend: ## go vet + gofmt 检查
	cd $(BACKEND) && go vet ./... && test -z "$$(gofmt -l .)" || (gofmt -l . && echo "↑ 需 gofmt" && exit 1)

.PHONY: lint-frontend
lint-frontend: ## eslint
	cd $(FRONTEND) && npm run lint

.PHONY: fmt
fmt: ## 格式化 Go 代码
	cd $(BACKEND) && gofmt -w .

# ---------------------------------------------------------------- Docker
.PHONY: docker-build
docker-build: ## 构建单架构镜像（当前平台）
	docker build \
	  --build-arg VERSION=$(VERSION) \
	  --build-arg COMMIT=$(COMMIT) \
	  --build-arg BUILD_TIME=$$(date -u +%Y-%m-%dT%H:%M:%SZ) \
	  -t $(IMAGE):$(TAG) .

.PHONY: docker-release
docker-release: ## 构建并推送 amd64 + arm64 双架构镜像
	docker buildx create --use --name dockermanage-builder 2>/dev/null || true
	docker buildx build \
	  --platform linux/amd64,linux/arm64 \
	  --build-arg VERSION=$(VERSION) \
	  --build-arg COMMIT=$(COMMIT) \
	  --build-arg BUILD_TIME=$$(date -u +%Y-%m-%dT%H:%M:%SZ) \
	  -t $(IMAGE):$(TAG) \
	  $(if $(TAG != latest),-t $(IMAGE):$(TAG),) \
	  --push .

.PHONY: docker-up
docker-up: ## 启动本机 compose 栈
	mkdir -p data compose
	docker compose up -d
	@echo "→ http://localhost:8080"

.PHONY: docker-down
docker-down: ## 停止本机 compose 栈
	docker compose down

.PHONY: docker-logs
docker-logs: ## 查看 compose 日志
	docker compose logs -f --tail=100

.PHONY: docker-clean
docker-clean: ## 清理本机 compose 栈及数据卷（危险：会删 data/ 和 compose/）
	docker compose down -v

# ---------------------------------------------------------------- 清理
.PHONY: clean
clean: ## 清理构建产物
	rm -rf $(FRONTEND)/dist $(BACKEND)/bin $(BACKEND)/coverage.out
	find . -name node_modules -type d -prune -exec rm -rf {} +
