# Dockermanage

> 一款通过单个 `docker-compose` 部署的 Docker 可视化管理面板 —— 个人/家庭自用场景下 Portainer 的现代化替代品。

**技术栈**：Go 1.22（chi + docker SDK + SQLite/WAL） · React 18 + TypeScript + Vite · Tailwind CSS + shadcn/ui · xterm.js

---

## 项目状态

| 阶段 | 状态 | 产出 |
|---|---|---|
| 需求与范围梳理 | ✅ 完成 | [SPEC.md](./SPEC.md) §1-§3 |
| 技术方案（架构 / API / 数据模型 / 部署） | ✅ 完成 | [SPEC.md](./SPEC.md) §4-§13 |
| 14 项关键决策拍板 | ✅ 完成 | [SPEC.md](./SPEC.md) §15.2 |
| UI 设计稿（4 屏） | ✅ 完成 | [ui-mocks/](./ui-mocks/) |
| 工程基线（Dockerfile / compose / CI / Makefile） | ✅ 完成 | [Dockerfile](./Dockerfile) · [docker-compose.yml](./docker-compose.yml) |
| 后端实现（Go，81 文件 / 10.6k 行） | ✅ 完成 | [backend/](./backend/) |
| 前端实现（React，68 源文件 / 9.5k 行） | ✅ 完成 | [frontend/](./frontend/) |
| 端到端联调（28 项检查全过） | ✅ 完成 | 静态检查 / 鉴权流程 / 8 个核心接口 / SSE / SPA 回退 |
| 容器化部署验证 | 🔄 验证中 | `make docker-build` |

## 文档

- **[SPEC.md](./SPEC.md)** — 技术规格说明书：功能范围、架构、模块划分、REST/SSE/WebSocket API、SQLite 表结构、安全模型、部署方案、路线图
- **[docs/api-contract.md](./docs/api-contract.md)** — API 契约：各接口请求体/响应体的确切 JSON 形状、流式协议帧格式
- **[ui-mocks/README.md](./ui-mocks/README.md)** — UI 设计说明：设计 token、组件规范、渲染自检记录

## UI 设计稿

4 个核心屏幕的静态设计稿（暗色优先，双击 HTML 即可在浏览器打开，无需装依赖）：

| 页面 | 文件 | 说明 |
|---|---|---|
| Dashboard | `ui-mocks/index.html` | 首屏门面：4 个资源卡 + 实时图表 + 容器卡 + 事件流 + 快捷操作 |
| 容器列表 | `ui-mocks/containers.html` | 多维筛选、批量操作、表格/卡片双视图 |
| 容器详情 | `ui-mocks/container-detail.html` | Stats / Logs / Terminal / Inspect / Files / 环境变量 六 tab |
| Compose | `ui-mocks/compose.html` | YAML 在线编辑 + 实时校验 + 部署 |

渲染截图见 `ui-mocks/screenshots/`。复现方式：

```bash
cd ui-mocks && ./.shoot.sh 1440 900
```

## 快速开始

### 容器部署（推荐）

```bash
git clone https://github.com/steven754/dockermanage.git
cd dockermanage
docker compose up -d
```

访问 `http://<host>:8080`，用默认凭据登录：

| 用户名 | 密码 |
|---|---|
| `admin` | `admin` |

登录后系统会**强制你立即改密**，改完才能进面板。

想要更强的初始密码，在 `docker-compose.yml` 的环境变量里指定（仅首次启动生效）：

```bash
# 方式一：指定你自己的初始密码
PANEL_INITIAL_PASSWORD='Your-Strong-Pass' docker compose up -d

# 方式二：生成 16 位随机密码，从日志里取
PANEL_INITIAL_PASSWORD=random docker compose up -d
docker compose logs panel | grep '初始'
```

初始密码同时写入 `data/.password`（权限 0600），改密后自动删除。

> ⚠️ **首次启动后请立刻改密码。** 面板默认只监听 HTTP，**公网暴露请务必挂在反向代理后面并启用 TLS**。
> 面板以特权模式运行并挂载了 docker.sock，等同于宿主机 root 权限，不要直接暴露到公网。

### 本地开发

```bash
make install          # 安装前后端依赖
make dev-backend      # 后端 → :8080
make dev-frontend     # 前端 → :5173（/api 代理到 :8080）
```

> macOS + Docker Desktop 时后端需要显式指定 socket：
> `export DOCKER_HOST=unix://$(docker context inspect --format '{{.Endpoints.docker.Host}}' | sed 's|^unix://||')`

### 常用命令

```bash
make help          # 列出全部命令
make test          # 后端竞态测试 + 前端构建
make lint          # go vet + gofmt + eslint
make docker-build  # 构建本机架构镜像
make docker-release# 构建并推送 amd64 + arm64 双架构镜像
```

## 范围说明

**v1.0 做**：Dashboard、Container / Image / Volume / Network 管理、Compose 部署、资源监控历史、事件流、⌘K 命令面板、一键更新全部镜像、单用户鉴权（JWT）、amd64 + arm64 双架构镜像。

**v1.0 不做**：多主机、Swarm / K8s、RBAC 多用户、审计日志、容器内文件浏览器、告警 webhook、应用模板市场（后三项排 v1.1）。详见 [SPEC.md §3](./SPEC.md)。

## License

[MIT](./LICENSE) © 2026 steven754
