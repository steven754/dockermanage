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
| 开发脚手架 | ⬜ 未开始 | — |
| 前端实现 | ⬜ 未开始 | — |
| 后端实现 | ⬜ 未开始 | — |

## 文档

- **[SPEC.md](./SPEC.md)** — 技术规格说明书（751 行）：功能范围、架构、模块划分、REST/SSE/WebSocket API、SQLite 表结构、安全模型、部署方案、路线图
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

## 快速开始（规划中）

```bash
git clone https://github.com/steven754/dockermanage.git
cd dockermanage
docker compose up -d
```

访问 `http://<host>:8080`，首次启动会在日志中打印随机初始密码。

> 该流程待实现，当前仓库仅含规格与设计稿。

## 范围说明

**v1.0 做**：Dashboard、Container / Image / Volume / Network 管理、Compose 部署、资源监控历史、事件流、⌘K 命令面板、一键更新全部镜像、单用户鉴权（JWT）、amd64 + arm64 双架构镜像。

**v1.0 不做**：多主机、Swarm / K8s、RBAC 多用户、审计日志、容器内文件浏览器、告警 webhook、应用模板市场（后三项排 v1.1）。详见 [SPEC.md §3](./SPEC.md)。

## License

[MIT](./LICENSE) © 2026 steven754
