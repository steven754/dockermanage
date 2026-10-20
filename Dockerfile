# syntax=docker/dockerfile:1.7
# ==============================================================================
# Dockermanage — 单镜像构建
#   Stage 1  前端构建   node:22-alpine
#   Stage 2  后端构建   golang:1.24-alpine（CGO_ENABLED=0 静态编译）
#   Stage 3  运行时     alpine:3.20（含 docker-cli + compose 插件）
# ==============================================================================

# ------------------------------------------------------------------------------
# Stage 1: Frontend
# ------------------------------------------------------------------------------
FROM node:22-alpine AS frontend

WORKDIR /app

# 依赖层单独缓存：package.json 不变就不重装
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm ci || npm install

COPY frontend/ ./
RUN npm run build

# ------------------------------------------------------------------------------
# Stage 2: Backend
# ------------------------------------------------------------------------------
FROM golang:1.24-alpine AS backend

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

# 禁止 toolchain 自动下载：go.mod 声明的版本必须与本镜像的 Go 一致，
# 否则构建时会静默拉取新版工具链，导致依赖树被意外升级
ENV GOTOOLCHAIN=local

WORKDIR /app

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./

# 前端产物注入后端 embed 目录（backend/internal/server/web/dist）
COPY --from=frontend /app/dist ./internal/server/web/dist

# CGO_ENABLED=0：使用 modernc.org/sqlite（纯 Go 驱动），产出静态二进制
# 注意：-X 必须写完整的包路径，写错不会报错、只会静默失效
RUN CGO_ENABLED=0 GOOS=linux go build \
      -trimpath \
      -ldflags="-s -w \
        -X github.com/steven754/dockermanage/backend/internal/version.Version=${VERSION} \
        -X github.com/steven754/dockermanage/backend/internal/version.Commit=${COMMIT} \
        -X github.com/steven754/dockermanage/backend/internal/version.BuildTime=${BUILD_TIME}" \
      -o /out/panel ./cmd/panel

# ------------------------------------------------------------------------------
# Stage 3: Runtime
# ------------------------------------------------------------------------------
FROM alpine:3.20

# ca-certificates：拉取镜像 / webhook 通知需要
# tzdata：容器时区
# docker-cli + docker-cli-compose：Compose 项目 up/down/ps 需要调用 docker compose CLI
RUN apk add --no-cache \
      ca-certificates \
      tzdata \
      docker-cli \
      docker-cli-compose \
      curl \
      && rm -rf /var/cache/apk/*

# 说明：容器以 root 运行。原因是特权模式下的容器 exec（终端 tab）与
# /var/run/docker.sock 访问都需要较高权限，这与 Portainer / Yacht 的做法一致。
# 若要收敛权限，v1.2 改用 Docker Socket Proxy + 非 root 运行（见 SPEC §8.1）。

COPY --from=backend /out/panel /usr/local/bin/panel

ENV PANEL_PORT=8080 \
    PANEL_DATA=/data \
    PANEL_COMPOSE_DIR=/compose \
    PANEL_LOG_LEVEL=info \
    TZ=Asia/Shanghai

VOLUME ["/data", "/compose"]
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD curl -fsS "http://127.0.0.1:${PANEL_PORT}/healthz" || exit 1

ENTRYPOINT ["/usr/local/bin/panel"]
