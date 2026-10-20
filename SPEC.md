# Docker 管理面板 · 技术规格说明书（SPEC）

> 版本：v1.0
> 最后更新：2026-10-20
> 状态：**已定稿**（功能范围 / 技术栈 / 14 项关键决策已拍板；待 UI 稿与全文验收）
> 作者：Mavis（root session） + worker（UI 稿并行产出）
> 决策记录：§15.2 · UI 设计稿：`ui-mocks/`

---

## 1. 项目概览

### 1.1 一句话定位
一款**通过单个 docker-compose 部署**的 Docker 可视化管理面板，定位为个人/家庭自用场景下 Portainer 的现代化替代品。

### 1.2 目标用户
- **唯一目标用户**：个人/家庭用户，单台 Linux 服务器 / NAS / 小主机（x86_64、arm64）
- **协作场景**：单机单用户，不做团队协作与多主机（见 §1.4 非目标）

### 1.3 核心价值
1. **现代感** — Portainer 的 Bootstrap 3 视觉已 9-10 年没大改，本项目用 shadcn/ui + Tailwind 把"第一印象"拉到 2025 标准
2. **实时性** — Dashboard 实时图表、容器终端/日志用 WebSocket 流式，无明显延迟
3. **可发现性** — ⌘K 命令面板覆盖所有功能，Power User 友好
4. **一键常用操作** — 一键更新全部镜像、清理 dangling、清理孤儿卷（应用模板市场 v1.1 起追加）
5. **轻部署** — 一个 docker-compose up 就跑起来，< 50MB 镜像，单 SQLite 文件持久化

### 1.4 非目标（v1.0 不做）
- ❌ 多主机管理
- ❌ Swarm / Kubernetes 编排
- ❌ RBAC 多用户体系
- ❌ 审计日志（个人单用户不需要）
- ❌ 商业化（计费、团队空间、SaaS 化）
- ❌ Windows 容器

---

## 2. 术语表

| 术语 | 含义 |
|---|---|
| **Panel** | 本项目本体，对外提供的 Web 管理面板 |
| **Backend** | Go 单二进制服务，对接 Docker Engine API |
| **Frontend** | React SPA，Vite 构建 |
| **Host** | Panel 所在主机（即 Docker daemon 所在主机，v1.0 一致） |
| **Compose** | `docker-compose.yml` 描述的多容器应用 |
| **Stats** | 容器运行时指标（CPU/内存/网络/磁盘 IO） |
| **Event** | Docker daemon 产生的事件流（容器启停、镜像拉取等） |

---

## 3. 功能范围

### 3.1 P0（v1.0 必须交付）

| 模块 | 子功能 | UI 重点 |
|---|---|---|
| **Dashboard** | 主机资源总览（CPU/内存/磁盘/网络）+ 实时折线图 + 容器卡片列表 + 事件流 + 快捷操作 | 首屏门面，重点打磨 |
| **Container** | 列表（多维筛选）/ 启停/重启/删除 / 详情 / **实时日志 tail** / **exec 终端** / **stats 实时曲线** / Inspect 查看 | xterm.js 风格终端，ANSI 颜色 |
| **Image** | 列表（含 size/tag/创建时间）/ pull / 推送 / 删除 / tag / 清理 dangling / 镜像大小可视化 | 大小饼图，悬停看占比 |
| **Volume** | 列表 / 查看被哪个容器用 / 删除 / 孤儿卷高亮 | — |
| **Network** | 列表 / 详情 / 删除 | — |
| **Compose** | 上传 / 在线编辑 `/docker-compose.yml` / up/down/ps / 日志聚合查看 | YAML 在线编辑 + 实时校验 + diff |
| **设置** | 基础认证密码修改 / 主题（暗/亮） / 端口 / 镜像加速器 | — |

### 3.2 P1 功能（v1.0 范围已拍板，2026-10-20）

| 模块 | 说明 | 版本 | 决策 |
|---|---|---|---|
| 资源监控历史（1h/24h/7d） | SQLite 持久化 stats 数据，可回看历史曲线 | **v1.0** | ✅ 必做（差异化关键） |
| 事件流持久化 | events 表 + 滚动保留 7 天 | **v1.0** | ✅ 必做 |
| 一键更新全部镜像 | 拉取新镜像 + 重启容器 | **v1.0** | ✅ 必做 |
| ⌘K 命令面板 | 覆盖所有功能的快速搜索 | **v1.0** | ✅ 必做（差异化关键） |
| 容器内文件浏览器 | 挂载卷文件树查看/下载/上传 | v1.1 | 决策：v1.0 不做，仅预留 API 边界 |
| 告警通知（webhook） | 容器异常退出/资源超阈值 → 飞书/钉钉/Telegram/Slack | v1.1 | 决策：v1.0 不做，notify 表结构已预留（§9.1） |
| 应用模板市场 | 一键部署 Nextcloud/Jellyfin/AdGuard 等 | v1.1 | 决策：v1.0 不做，templates 表已预留（§9.1） |
| 备份/恢复（容器配置 + 卷） | 导出/导入 JSON + tar 快照 | v1.1+ | 决策：v1.0 不做 |

### 3.3 P2（不做）
- 多主机管理 / Swarm / K8s / RBAC / 审计 / 计费

---

## 4. 系统架构

### 4.1 总体架构

```
┌──────────────────────────────────────────────────────────────┐
│                  Docker Host（同机部署）                      │
│                                                              │
│   ┌─────────────────────────────────────────────────────┐    │
│   │  Panel Container（特权模式）                          │    │
│   │                                                     │    │
│   │  ┌────────────────┐    ┌─────────────────────────┐   │    │
│   │  │  Frontend      │    │  Backend (单进程)         │   │    │
│   │  │  React + Vite  │◄──►│  Go + chi + docker SDK  │   │    │
│   │  │  静态资源       │    │  + SQLite (WAL)         │   │    │
│   │  └────────────────┘    └─────────────────────────┘   │    │
│   │           ▲                     ▲                    │    │
│   │           │                     │                    │    │
│   │           │ HTTP/SSE/WS        │ unix socket         │    │
│   └───────────┼─────────────────────┼────────────────────┘    │
│              │                     │                          │
│   ┌──────────┴──────┐    ┌────────┴───────────────┐          │
│   │  User Browser   │    │  Docker Daemon         │          │
│   │                 │    │  /var/run/docker.sock  │          │
│   └─────────────────┘    └────────────────────────┘          │
│                                                              │
│   ┌─────────────────────────────────────────────────────┐    │
│   │  Persistent Volumes                                │    │
│   │  - /data (SQLite, settings, templates)             │    │
│   │  - /compose (compose stack 部署目录)               │    │
│   └─────────────────────────────────────────────────────┘    │
└──────────────────────────────────────────────────────────────┘
```

### 4.2 技术栈

| 层 | 选型 | 理由 |
|---|---|---|
| **后端语言** | Go 1.22+ | 单二进制、内存低、Docker 官方 SDK 成熟（`github.com/docker/docker/client`） |
| **HTTP 框架** | chi v5 | 轻量、middleware 链清晰、对比 gin 更接近 net/http 风格 |
| **WebSocket** | nhooyr.io/websocket 或 coder/websocket | 比 gorilla 更现代 |
| **SSE** | 标准库即可 | Dashboard 实时图表用 SSE 足够，复杂度低于 WS |
| **数据库** | SQLite + WAL | 单文件、零运维、性能足够；WAL 模式提升并发 |
| **ORM** | sqlx + 手工 SQL | 简单查询可控，避免 GORM 反射开销 |
| **前端框架** | React 18 + TypeScript | 生态成熟、shadcn/ui 基于此 |
| **构建** | Vite 5 | 启动快、HMR 好 |
| **样式** | Tailwind CSS 3 + shadcn/ui | 高度可定制、符合"高 UI 要求" |
| **图表** | Recharts 或 chart.js | Recharts 基于 SVG、易定制 |
| **终端** | xterm.js + xterm-addon-fit + xterm-addon-web-links | 业界标准 |
| **图标** | lucide-react | 一致，纯净 |
| **状态管理** | Zustand + TanStack Query | 轻量，server state 分离 |
| **路由** | React Router v6 | 标准选择 |
| **容器化** | 多阶段 Dockerfile | 后端 + 前端静态资源合并到一个镜像 |

### 4.3 关键决策记录

**决策 D1：后端用 Go 而非 Node**
- 理由：单二进制部署、内存占用低（~20MB 运行时）、Docker SDK 官方支持、编译期类型检查

**决策 D2：单进程单镜像部署**
- 理由：个人场景，避免 docker-compose 多服务编排的运维负担；前端静态资源由 Go embed 包打包进二进制

**决策 D3：SQLite 而非 Postgres**
- 理由：单文件随 docker volume 持久化，零运维；WAL 模式下并发读性能足够；如未来需要切换，repository 层抽象已隔离

**决策 D4：容器内部署时使用 `--pid host` 但不挂载 docker.sock**
- 理由：见 §8.1 安全考虑 — 优先使用 Docker Proxy 而非 socket 挂载（v1.0 暂用 socket，v1.1 引入 Proxy）

**决策 D5：WebSocket 用于终端/日志流，SSE 用于 stats/events 广播**
- 理由：终端/日志需要双向（resize、键盘输入），用 WS；stats/events 只需服务端→客户端，SSE 更简单且自动重连

---

## 5. 模块设计

### 5.1 后端模块（Go）

```
backend/
├── cmd/panel/main.go               # 入口
├── internal/
│   ├── server/                    # HTTP server 启动 + 路由
│   │   ├── router.go              # chi 路由
│   │   ├── middleware.go          # auth / logging / recover
│   │   └── sse.go                 # SSE helper
│   ├── handler/                   # HTTP handler
│   │   ├── auth.go                # /api/v1/auth/*
│   │   ├── container.go           # /api/v1/containers/*
│   │   ├── image.go               # /api/v1/images/*
│   │   ├── volume.go              # /api/v1/volumes/*
│   │   ├── network.go             # /api/v1/networks/*
│   │   ├── compose.go             # /api/v1/compose/*
│   │   ├── stats.go               # /api/v1/stats/*
│   │   ├── events.go              # /api/v1/events (SSE)
│   │   ├── dashboard.go           # /api/v1/dashboard
│   │   ├── notify.go              # /api/v1/notify/*
│   │   ├── template.go            # /api/v1/templates
│   │   └── settings.go            # /api/v1/settings
│   ├── service/                   # 业务服务层
│   │   ├── docker/                # Docker SDK 封装
│   │   │   ├── client.go          # Docker client 单例
│   │   │   ├── container.go
│   │   │   ├── image.go
│   │   │   ├── volume.go
│   │   │   ├── network.go
│   │   │   ├── stats.go          # 系统级 stats（CPU/内存/磁盘）
│   │   │   └── events.go
│   │   ├── stats/                 # stats 历史持久化 + 查询
│   │   ├── events/                # events 持久化 + 查询
│   │   ├── auth/                  # 单用户认证（bcrypt）
│   │   ├── settings/              # 设置持久化
│   │   ├── notify/                # webhook 通知
│   │   └── update/                # 一键更新镜像逻辑
│   ├── model/                     # 数据模型（DTO + DB struct）
│   ├── repo/                      # 数据库 repository
│   ├── config/                    # 配置加载（环境变量 + 配置文件）
│   └── version/                   # 版本信息
├── migrations/                    # SQLite migration
└── go.mod
```

### 5.2 前端路由

```
/                              # Dashboard
/login                         # 登录页（首次访问）
/containers                    # 容器列表
/containers/:id                # 容器详情（tabs: Overview/Logs/Terminal/Stats/Inspect）
/images                        # 镜像列表
/volumes                       # 卷列表
/networks                      # 网络列表
/compose                       # Compose 项目列表
/compose/:project              # Compose 项目详情
/settings                      # 设置（认证 / 主题 / 端口 / 镜像加速器）
/templates                    # 应用模板（一键部署）
```

### 5.3 关键数据流

**实时数据**：
- `GET /api/v1/events`（SSE）— 推送所有 docker events，前端订阅
- `GET /api/v1/stats/stream`（SSE）— 推送容器 stats（CPU/内存），每秒一次
- `GET /api/v1/system/stats`（SSE）— 推送主机系统 stats
- `WS /api/v1/containers/:id/logs`（WebSocket）— 容器日志流（带 since/timestamps）
- `WS /api/v1/containers/:id/exec`（WebSocket）— exec 终端（双向）

**鉴权**（单用户）：
- 首次启动生成随机密码，写入 `/data/.password`，提示用户修改
- `POST /api/v1/auth/login` 返回 HttpOnly cookie（JWT 或 session ID，v1.0 用 JWT 简化）
- 后续请求带 `Authorization: Bearer <token>` 或 cookie

---

## 6. API 设计

### 6.1 REST 端点

```
# 认证
POST   /api/v1/auth/login            # 登录
POST   /api/v1/auth/logout           # 登出
POST   /api/v1/auth/password         # 修改密码（首次登录强制修改）

# Dashboard
GET    /api/v1/dashboard              # 总览数据（合并主机 stats + 容器摘要 + 事件）

# Container
GET    /api/v1/containers             # 列表（支持 ?status=running&label=...&search=...）
POST   /api/v1/containers/:id/start
POST   /api/v1/containers/:id/stop
POST   /api/v1/containers/:id/restart
POST   /api/v1/containers/:id/kill
DELETE /api/v1/containers/:id?force=true
GET    /api/v1/containers/:id         # 详情
GET    /api/v1/containers/:id/json    # docker inspect
WS     /api/v1/containers/:id/logs?tail=200&timestamps=true&follow=true
WS     /api/v1/containers/:id/exec?cmd=sh  # interactive
GET    /api/v1/containers/:id/stats   # 当前快照（一次）
POST   /api/v1/containers/:id/update  # 一键更新（pull + recreate）
POST   /api/v1/containers/prune       # 清理已停止容器

# Image
GET    /api/v1/images                 # 列表
POST   /api/v1/images/pull            # { image: "nginx:1.27" }
POST   /api/v1/images/:id/tag         # { repo, tag }
DELETE /api/v1/images/:id?force=true
POST   /api/v1/images/prune           # { dangling: true }

# Volume
GET    /api/v1/volumes
DELETE /api/v1/volumes/:name?force
POST   /api/v1/volumes/prune

# Network
GET    /api/v1/networks
DELETE /api/v1/networks/:id

# Compose
GET    /api/v1/compose/projects       # 列出 /compose 目录下所有项目
GET    /api/v1/compose/projects/:name
PUT    /api/v1/compose/projects/:name/file  # 保存 compose.yml
POST   /api/v1/compose/projects/:name/up    # { services?: ["web"] }
POST   /api/v1/compose/projects/:name/down
GET    /api/v1/compose/projects/:name/ps

# Stats 历史
GET    /api/v1/stats/history?range=1h|24h|7d&container=...

# Events（事件流 + 历史）
GET    /api/v1/events                 # SSE 实时流
GET    /api/v1/events/history?limit=100&since=...

# Settings
GET    /api/v1/settings
PUT    /api/v1/settings

# Notify（v1.1+）
POST   /api/v1/notify/test
GET    /api/v1/notify/channels
PUT    /api/v1/notify/channels

# Templates（v1.1+）
GET    /api/v1/templates
POST   /api/v1/templates/:id/deploy
```

### 6.2 错误响应格式
```json
{ "error": { "code": "CONTAINER_NOT_FOUND", "message": "Container xyz not found" } }
```

错误码：
- `UNAUTHORIZED` (401) / `FORBIDDEN` (403) / `NOT_FOUND` (404) / `CONFLICT` (409)
- `DOCKER_API_ERROR` (502) — 上游 docker daemon 错误透传
- `VALIDATION_ERROR` (400) / `INTERNAL_ERROR` (500)

### 6.3 SSE 消息格式
```
event: container.start
data: {"id":"abc123","name":"nginx-proxy","time":"2026-10-16T15:30:00Z"}

event: container.stats
data: {"id":"abc123","cpu":"15.3","mem":"128MB","net_rx":"1.2MB","net_tx":"0.3MB"}
```

---

## 7. Docker 集成

### 7.1 Docker Client 配置
```go
// 默认通过 unix socket 连接
cli, _ := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
// 实际：client.NewClientWithOpts(client.WithHost("unix:///var/run/docker.sock"))
```

### 7.2 Privileged Container 要求
`docker-compose.yml` 中需要：
```yaml
services:
  panel:
    image: <registry>/<repo>/<image>:latest
    privileged: true          # exec 需要
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /data:/data
      - /compose:/compose
    ports:
      - "8080:8080"
    restart: unless-stopped
```

> 注：`privileged: true` 是 exec/inspect 容器内进程所需，是个人场景的合理妥协。如未来需要更严格隔离，引入 Docker Proxy（tecnativa/docker-socket-proxy）。

### 7.3 镜像加速器支持
- `~/.docker/daemon.json` 由用户自己配（Panel 不动 host docker daemon 配置）
- Panel 自身拉镜像时通过 `--registry-mirror` 透传（v1.0 不实现，v1.1+）

---

## 8. 安全考虑

### 8.1 容器逃逸防护
| 风险 | 缓解 |
|---|---|
| Panel 容器被攻破→直接控制 host docker | 1. **强密码 + JWT**；2. 不暴露 docker socket 到容器外；3. v1.1+ 引入 Docker Proxy 替代 socket 直挂 |
| exec 容器内 shell → 利用特权逃逸 | 个人自用场景已知风险，文档明示，不做过度防护 |
| Compose 编辑器任意文件读写 | `/compose` 目录限定，不允许 `..`、不允许绝对路径 |

### 8.2 Web 安全
- HTTPS：v1.0 不内置 TLS，依赖反向代理（Caddy / Nginx）— 文档明确说明
- CSRF：双 token（access token + refresh token），access 走 Authorization header，refresh 走 HttpOnly cookie
- CORS：默认同源，可配置允许的 origin
- XSS：React 默认转义；不直接渲染用户输入 HTML
- Rate Limit：登录端点 5 次/分钟

### 8.3 敏感数据
- 容器环境变量（特别是密码）默认**脱敏**显示，前端 hover 才显示明文
- Settings 中的 webhook URL / 密码 bcrypt 存储或 AES 加密
- 日志中禁止打印环境变量值

---

## 9. 数据模型

### 9.1 SQLite 表

```sql
-- 用户（v1.0 仅单用户）
CREATE TABLE users (
    id              INTEGER PRIMARY KEY,
    username        TEXT NOT NULL UNIQUE,
    password_hash   TEXT NOT NULL,                -- bcrypt
    created_at      INTEGER NOT NULL,
    last_login_at   INTEGER
);

-- 设置（kv 存储）
CREATE TABLE settings (
    key             TEXT PRIMARY KEY,
    value           TEXT NOT NULL,                -- JSON
    updated_at      INTEGER NOT NULL
);

-- 系统 stats 历史（采样频率：10 秒）
CREATE TABLE system_stats (
    ts              INTEGER NOT NULL,                -- unix timestamp
    cpu_pct          REAL NOT NULL,
    mem_used_mb       INTEGER NOT NULL,
    mem_total_mb       INTEGER NOT NULL,
    disk_used_gb       INTEGER NOT NULL,
    disk_total_gb      INTEGER NOT NULL,
    net_rx_kbps       INTEGER NOT NULL,
    net_tx_kbps       INTEGER NOT NULL,
    PRIMARY KEY (ts)
);
CREATE INDEX idx_system_stats_ts ON system_stats(ts);

-- 容器 stats 历史
CREATE TABLE container_stats (
    ts              INTEGER NOT NULL,
    container_id    TEXT NOT NULL,
    container_name  TEXT NOT NULL,
    cpu_pct         REAL NOT NULL,
    mem_used_mb       INTEGER NOT NULL,
    mem_limit_mb      INTEGER NOT NULL,
    net_rx_kbps       INTEGER NOT NULL,
    net_tx_kbps       INTEGER NOT NULL,
    block_read_mb    INTEGER NOT NULL,
    block_write_mb   INTEGER NOT NULL,
    PRIMARY KEY (ts, container_id)
);
CREATE INDEX idx_container_stats_ts ON container_stats(ts);
CREATE INDEX idx_container_stats_container ON container_stats(container_id, ts);

-- 事件历史（保留 7 天，自动清理）
CREATE TABLE events (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    ts              INTEGER NOT NULL,
    type            TEXT NOT NULL,                  -- container.start / image.pull / etc
    action          TEXT NOT NULL,
    actor_id        TEXT,
    actor_name       TEXT,
    resource_kind   TEXT NOT NULL,
    resource_id     TEXT,
    resource_name   TEXT,
    details         TEXT                            -- JSON
);
CREATE INDEX idx_events_ts ON events(ts);
CREATE INDEX idx_events_type ON events(type);

-- 通知通道（v1.1）
CREATE TABLE notify_channels (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    type            TEXT NOT NULL,                  -- webhook / telegram / slack
    name            TEXT NOT NULL,
    config          TEXT NOT NULL,                  -- JSON, 加密存储
    enabled         INTEGER NOT NULL DEFAULT 1,
    created_at      INTEGER NOT NULL
);

-- 通知规则（v1.1）
CREATE TABLE notify_rules (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    channel_id      INTEGER NOT NULL REFERENCES notify_channels(id),
    trigger         TEXT NOT NULL,                  -- container.died / cpu.exceeded / etc
    threshold       TEXT,                           -- JSON: {cpu: 80, duration: 5m}
    enabled         INTEGER NOT NULL DEFAULT 1,
    created_at      INTEGER NOT NULL
);

-- 应用模板（v1.1）
CREATE TABLE templates (
    id              TEXT PRIMARY KEY,                -- slug
    name            TEXT NOT NULL,
    description     TEXT,
    category        TEXT,                           -- media / network / productivity / dev
    icon            TEXT,
    compose_yaml    TEXT NOT NULL,
    env_schema      TEXT,                           -- 用户需要填的变量
    version         INT NOT NULL DEFAULT 1,
    updated_at      INTEGER NOT NULL
);
```

### 9.2 采样与清理策略
- 系统 stats：10 秒采样一次，保留 7 天（约 60K 行/主机，< 5MB）
- 容器 stats：10 秒采样一次，只采样 running 容器，保留 7 天
- events：实时写入，保留 7 天，凌晨定时清理
- 通过 cron-like goroutine + context 控制生命周期

---

## 10. 性能与可观测性

### 10.1 性能目标
- 首屏 LCP < 1s（本地网络）
- API 95 分位延迟 < 200ms（不含 SSE/WS）
- SSE/WS 推送延迟 < 100ms
- 内存占用：空闲 < 80MB，负载 < 200MB

### 10.2 可观测性
- 内部指标：pprof endpoint（`/debug/pprof`，仅 localhost 访问或鉴权）
- 结构化日志：zap，JSON 格式，按需采样
- 健康检查：`GET /healthz`（liveness）、`GET /readyz`（包含 docker daemon 连通性）
- 版本信息：`GET /api/v1/version` 返回 git commit / build time

---

## 11. 部署

### 11.1 项目结构
```
dockermanage/                    # 本仓库
├── SPEC.md                      # 本文档
├── README.md                    # 用户使用文档
├── docker-compose.yml           # 部署入口
├── Dockerfile                   # 多阶段构建
├── backend/                     # Go 后端
├── frontend/                    # React 前端
├── ui-mocks/                    # UI 设计稿（仅开发期）
```

### 11.2 Dockerfile（多阶段）
```dockerfile
# Stage 1: Frontend build
FROM node:20-alpine AS frontend
WORKDIR /app
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 2: Backend build
FROM golang:1.22-alpine AS backend
WORKDIR /app
COPY backend/go.* ./
RUN go mod download
COPY backend/ ./
COPY --from=frontend /app/dist /app/web
RUN CGO_ENABLED=1 go build -ldflags="-s -w -X main.Version=$(VERSION)" -o /out/panel ./cmd/panel

# Stage 3: Runtime
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=backend /out/panel /usr/local/bin/panel
VOLUME ["/data", "/compose"]
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/panel"]
```

**多架构构建（决策 D-A8：amd64 + arm64 双架构）**

```bash
docker buildx create --use --name dockermanage-builder
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t ghcr.io/<user>/dockermanage:latest \
  -t ghcr.io/<user>/dockermanage:v1.0.0 \
  --push .
```

- Go 静态编译无 CGO 依赖，交叉编译零成本；alpine base 天然支持两架构
- CI 用 matrix（`linux/amd64` + `linux/arm64`）跑 buildx，产出 manifest list
- 镜像体积目标 < 50MB（前端 dist 约 1.5MB + Go 二进制约 20MB + alpine 基础层）

### 11.3 docker-compose.yml
```yaml
version: "3.9"
services:
  panel:
    image: ghcr.io/<user>/dockermanage:latest
    container_name: dockermanage
    restart: unless-stopped
    privileged: true
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./data:/data
      - ./compose:/compose
    ports:
      - "8080:8080"
    environment:
      - PANEL_PORT=8080
      - PANEL_DATA=/data
      - PANEL_COMPOSE_DIR=/compose
      - PANEL_LOG_LEVEL=info
      - TZ=${TZ:-Asia/Shanghai}
```

### 11.4 升级
```bash
docker compose pull
docker compose up -d
# /data 持久化，无需迁移
```

### 11.5 备份
```bash
# 数据目录
tar czf backup-$(date +%Y%m%d).tar.gz data/ compose/
```

---

## 12. 开发流程

### 12.1 本地开发（无 Docker）
```bash
# Backend
cd backend
go run ./cmd/panel --dev --port 8080

# Frontend
cd frontend
npm install
npm run dev  # 默认 http://localhost:5173，proxy /api 到 :8080
```

### 12.2 Docker 内开发
```bash
docker compose -f docker-compose.dev.yml up
```

### 12.3 首次启动
1. 用户 `docker compose up -d`
2. 容器生成随机密码写入 `/data/.password`，日志打印
3. 用户访问 `http://host:8080`，强制修改密码
4. 进入 Dashboard

---

## 13. 测试策略

### 13.1 单元测试
- Go：`go test -race`，覆盖率 ≥ 70%（service + model 层）
- TS：Vitest，关键 hooks + utilities

### 13.2 集成测试
- 用 testcontainers-go 起一个真实 docker daemon，跑端到端 API
- CI 中跑（GitHub Actions + DinD）

### 13.3 E2E（v1.1+）
- Playwright 跑关键场景：登录、Dashboard 加载、容器启停、看日志

### 13.4 手动测试 checklist
- [ ] Dashboard 实时数据刷新（间隔 10s）
- [ ] 容器 exec 终端交互正常（resize、回车）
- [ ] 日志流支持 ANSI 颜色 + 搜索
- [ ] ⌘K 命令面板能搜到所有页面
- [ ] 暗/亮主题切换无残留
- [ ] Compose up/down 后服务正常
- [ ] 升级镜像不丢数据

---

## 14. 路线图

### v1.0（首个发布，目标 4-6 周）
- P0 全功能
- P1 必做项：stats 历史、events 历史、⌘K 命令面板、一键更新全部镜像
- 交付形态：单镜像（**amd64 + arm64 双架构**），docker-compose 一键部署，**需登录**，UI **仅中文**
- 安全基线：`privileged` + docker.sock（v1.0 接受，v1.2 升级 Docker Proxy）
- 存储：SQLite 单文件（仅预留 repository 抽象，不实装 Postgres）

### v1.1（+2-4 周）
- 容器内文件浏览器
- 告警通知（webhook，飞书/钉钉/Telegram/Slack）
- 应用模板市场（首批 5-10 个常用模板）
- 备份/恢复（容器配置 + 卷）

### v1.2（+4-6 周）
- Docker Proxy 替代 socket（安全强化）
- HTTPS 内置（自签证书 / ACME）
- 多用户基础（仍无 RBAC，仅分主路由可见性）

### v2.0（远期）
- 轻量 RBAC（admin / viewer）
- 镜像仓库管理（私有 registry）
- v1.3+ 评估是否做多主机

---

## 15. 风险与决策记录

### 15.1 风险
| 风险 | 影响 | 缓解 |
|---|---|---|
| Docker SDK 版本不兼容（不同 docker engine 版本） | API 报错 | 用 `client.WithAPIVersionNegotiation()`，CI 多版本测试 |
| SQLite WAL 模式下多进程并发写 | 数据库锁 | 单进程保证（一个 Panel 实例），未来如需多实例换 Postgres |
| WebSocket 容器日志断连后无法续传 | 体验差 | 实现重连 + 重连后传 `since` 时间戳 |
| `privileged: true` 安全争议 | 文档明示 + 推荐反向代理 + 内置 auth | 文档明确强调"个人场景可接受，企业部署需额外加固" |

### 15.2 决策记录（2026-10-20 全部拍板）

| # | 议题 | 决策 | 影响章节 |
|---|---|---|---|
| D-A1 | P1 必做 4 项 | **全做进 v1.0**（stats 历史 / events 历史 / ⌘K / 一键更新） | §3.2 / §14 |
| D-A2 | 应用模板市场 | v1.0 **不做**，推 v1.1；templates 表结构预留 | §3.2 / §9.1 |
| D-A3 | 容器内文件浏览器 | v1.0 **不做**，推 v1.1 | §3.2 |
| D-A4 | 告警通知 webhook | v1.0 **不做**，推 v1.1；notify 表结构预留 | §3.2 / §9.1 |
| D-A5 | TLS | v1.0 **不内置**，依赖反向代理（Caddy / Nginx） | §8.2 / §11 |
| D-A6 | 首屏鉴权 | v1.0 **需要登录**，首启生成随机密码 + 强制改密 | §5.3 / §12.3 |
| D-A7 | i18n | v1.0 **仅中文** | §11.2 前端 |
| D-A8 | 多架构镜像 | **amd64 + arm64 双架构**，CI matrix build | §11.1 / §11.2 |
| D-B1 | 容器运行模式 | 接受 `privileged: true` + 挂载 docker.sock；v1.2 升级 Docker Proxy | §8.1 / §11.3 |
| D-B2 | 鉴权实现 | **JWT 双 token**（access 走 header，refresh 走 HttpOnly cookie） | §5.3 / §8.2 |
| D-B3 | 数据库演进 | SQLite 为主，**仅预留 repository 抽象**，v1.0 不实装 Postgres | §4.2 / §4.3 D3 |
| D-C1 | stats 采样 | 系统与容器均 **10 秒一次**，保留 **7 天** | §9.2 |
| D-C2 | 性能目标 | LCP < 1s / API P95 < 200ms / 空闲内存 < 80MB | §10.1 |
| D-C3 | 事件保留期 | **7 天**，凌晨定时清理 | §9.2 |
| D-C4 | 默认端口与目录 | **8080**；`/data` + `/compose` 相对路径挂载 | §11.3 |

> 以上 14 项为 2026-10-20 用户拍板结论，后续实现以本表为准。如需变更，请在附录 B 记录并同步修改相关章节。

---

## 16. 验收清单

- [x] 用户拍板第 3 节功能范围（2026-10-20）
- [x] 用户拍板第 4.2 节技术栈 — Go + React + shadcn/ui（2026-10-16）
- [x] 用户拍板第 15.2 节 14 项决策（2026-10-20）
- [ ] 用户验收 UI 稿（4 个屏幕视觉与设计方向）— `ui-mocks/index.html`
- [ ] 用户验收本文档全文

### 16.1 开工前置检查（开发启动前逐条确认）

- [ ] Docker Engine 版本 ≥ 20.10 且 `docker.sock` 可访问
- [ ] 本机 Node ≥ 20 / Go ≥ 1.22（开发环境）
- [ ] `ui-mocks/` 4 屏设计稿已被前端实现直接采纳，不重新设计
- [ ] §9.1 migration 已就绪（users / settings / system_stats / container_stats / events 五张表先行，notify / templates 预留）
- [ ] 鉴权模块先落地（首启随机密码 → 强制改密 → JWT 双 token），再开其他接口

---

## 附录 A：参考对照

| 我们的方案 | 参考实现 | 借鉴点 |
|---|---|---|
| Dashboard 设计 | Grafana / Netdata | 卡片+图表布局 |
| 终端体验 | LazyDocker / VSCode | xterm.js + 快捷键 |
| 整体风格 | shadcn/ui Dashboard 示例 | 现代暗色 |
| 命令面板 | Linear / Raycast | ⌘K 全功能搜索 |

## 附录 B：变更记录

| 日期 | 版本 | 变更人 | 变更内容 |
|---|---|---|---|
| 2026-10-16 | v1.0-draft | Mavis | 初版草案 |
| 2026-10-20 | v1.0 | Mavis | 用户拍板 A/B/C 三组共 14 项决策（§15.2），文档定稿；P1 范围与 v1.0/v1.1 路线图按决策重排；修正 §1.2 目标用户表述与 §16 章节引用笔误；新增 §16.1 开工前置检查；UI 稿完成 4 屏无头 Chrome 渲染自检，修复 ⌘K 遮罩/搜索框竖排/窄屏侧栏 3 个布局 bug（详见 `ui-mocks/README.md`） |