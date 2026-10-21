# Dockermanage

> 一款通过单个 `docker-compose` 部署的 Docker 可视化管理面板 —— 个人/家庭自用场景下 Portainer 的现代化替代品。

**技术栈**：Go 1.24（chi v5 + docker SDK v27 + modernc SQLite/WAL 纯 Go 驱动） · React 18 + TypeScript + Vite 5 · Tailwind CSS 3.4 + shadcn/ui · xterm.js

---

## 项目状态

| 阶段 | 状态 | 产出 |
|---|---|---|
| 需求与范围梳理 | ✅ 完成 | [SPEC.md](./SPEC.md) §1-§3 |
| 技术方案（架构 / API / 数据模型 / 部署） | ✅ 完成 | [SPEC.md](./SPEC.md) §4-§13 |
| 27 项关键决策拍板 | ✅ 完成 | [SPEC.md](./SPEC.md) §15.2 |
| UI 设计稿（4 屏） | ✅ 完成 | [ui-mocks/](./ui-mocks/) |
| 工程基线（Dockerfile / compose / CI / Makefile） | ✅ 完成 | [Dockerfile](./Dockerfile) · [docker-compose.yml](./docker-compose.yml) |
| 后端实现（Go，81 文件 / 11.2k 行） | ✅ 完成 | [backend/](./backend/) |
| 前端实现（React，68 源文件 / 9.6k 行） | ✅ 完成 | [frontend/](./frontend/) |
| 端到端联调（31 项检查全过） | ✅ 完成 | 静态检查 / 鉴权流程 / 8 个核心接口 / SSE / SPA 回退 |
| 安全增量扫描与加固（5 条 finding 全修） | ✅ 完成 | 强制改密后端强制 / 令牌版本失效 / 代理头收紧 / WS Origin 校验 |
| 容器化部署验证 | ✅ 完成 | 镜像实拉实跑，docker compose 一键起 |
| 镜像发布（amd64 + arm64 双架构） | ✅ 完成 | `stevensign/dockermanage` · `ghcr.io/steven754/dockermanage` |
| 持续集成 | ✅ 全绿 | 前端 / 后端 / 双架构镜像构建 三项通过 |

### 当前版本

`v0.1.0` · 2026-10-20 · 面板默认凭据 `admin` / `admin`（登录后强制改密）

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

或直接用镜像（amd64 / arm64 双架构）：

```bash
docker run -d --name dockermanage -p 8080:8080 \
  --privileged \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v $(pwd)/data:/data -v $(pwd)/compose:/compose \
  stevensign/dockermanage:latest
```

> 镜像同时发布在 Docker Hub（`stevensign/dockermanage`）与 GitHub Container Registry
> （`ghcr.io/steven754/dockermanage`），两者内容一致。

访问 `http://<host>:8080`，用默认凭据登录：

| 用户名 | 密码 |
|---|---|
| `admin` | `admin` |

登录后系统会**强制你立即改密**，改完才能进面板。

### 初始密码：默认值只是一个兜底

`admin` / `admin` 是**首次启动**时的兜底默认值（编排文件里写的是 `${PANEL_INITIAL_PASSWORD:-admin}`），
你可以**在第一次启动前**就把它换掉：

```bash
# 方式一：第一次启动前就指定强密码，全程不出现弱默认
PANEL_INITIAL_PASSWORD='你的强密码' docker compose up -d

# 方式二：让它生成 16 位随机密码，从日志里取
PANEL_INITIAL_PASSWORD=random docker compose up -d
docker compose logs panel | grep '初始'
```

**为什么值得多这一步**：面板以特权模式运行并挂载了 docker.sock，权限等同宿主机 root。
从容器启动到你改完密码这段时间里，默认凭据是全网已知的 —— 公网机器上这个窗口是真实风险。
指定初始密码能让这个窗口直接不存在。

**注意语义**：`PANEL_INITIAL_PASSWORD` **只在首次启动（数据库里还没有用户）时生效一次**。
之后再改这个变量、哪怕删了 `data/` 之外的任何配置，都不会影响已有部署 —— 想改密码只能登录后去设置页改。

初始密码同时写入 `data/.password`（权限 0600），**改密后该文件自动删除**。

### 可调环境变量

| 变量 | 默认 | 生效时机 | 说明 |
|---|---|---|---|
| `PANEL_INITIAL_PASSWORD` | `admin` | **仅首次启动一次** | 首启初始密码；`random` 表示生成 16 位随机密码 |
| `PANEL_TRUST_PROXY` | `false` | 重启即生效 | 是否采信 `X-Forwarded-For`；**仅在面板确实部署在反向代理之后时开启**，否则登录限流可被伪造头绕过 |
| `PANEL_LOGIN_RATE_LIMIT` | `10` | 重启即生效 | 每分钟允许的登录尝试次数。容器化部署时所有请求来源地址相同，该阈值实际是全局的 |

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
make help           # 列出全部命令
make test           # 后端竞态测试 + 前端构建
make lint           # go vet + gofmt + eslint
make tidy           # 整理后端依赖（固定 Go 1.24 基线，不会顶高 go 指令）
make docker-build   # 构建本机架构镜像
make docker-release # 构建并推送 amd64 + arm64 双架构镜像
make docker-mirror  # 把已构建好的多架构镜像从一个仓库搬运到另一个
```

> **交叉构建限制**：在 Apple Silicon 上构建 amd64 依赖 QEMU 模拟，Docker Desktop 自带的
> QEMU 版本较旧时 Go 工具链会在 netpoll 处崩溃（`go mod download` 退出码 2）。
> 此时用 `make docker-mirror` 搬运 CI 已构建好的多架构清单，或升级 QEMU。

## 范围说明

**v1.0 做**：Dashboard、Container / Image / Volume / Network 管理、Compose 部署、资源监控历史、事件流、⌘K 命令面板、一键更新全部镜像、单用户鉴权（JWT）、amd64 + arm64 双架构镜像。

**v1.0 不做**：多主机、Swarm / K8s、RBAC 多用户、审计日志、容器内文件浏览器、告警 webhook、应用模板市场（后三项排 v1.1）。详见 [SPEC.md §3](./SPEC.md)。

**v1.1 计划**：容器内文件浏览器 · 告警 webhook（飞书/钉钉/Telegram/Slack）· 应用模板市场 · 备份恢复。相关表结构已在 v1.0 预留。

**已知限制**：
- 容器内看不到宿主机 `/var/lib/docker`，故卷的 `sizeBytes` 恒为 0、磁盘挂载点按 `/` 统计
- WebSocket 终端与日志的 Origin 校验要求面板与前端同源，跨域反代需自行放行
- 面板以特权模式 + docker.sock 运行，权限等同宿主机 root，仅限可信网络使用

## License

[MIT](./LICENSE) © 2026 steven754
