# Docker Compose 部署教程

面向 **Dockermanage v0.1.0**，覆盖从零启动到日常运维、定制、故障排查的完整流程。

---

## 目录

- [1. 前置要求](#1-前置要求)
- [2. 三分钟快速开始](#2-三分钟快速开始)
- [3. 编排文件逐段解读](#3-编排文件逐段解读)
- [4. 首次登录与改密](#4-首次登录与改密)
- [5. 常用运维命令](#5-常用运维命令)
- [6. 数据与持久化](#6-数据与持久化)
- [7. 常见定制](#7-常见定制)
- [8. 反向代理示例](#8-反向代理示例)
- [9. 故障排查](#9-故障排查)
- [10. 升级与回滚](#10-升级与回滚)
- [11. 安全清单](#11-安全清单)

---

## 1. 前置要求

| 项 | 要求 | 检查命令 |
|---|---|---|
| Docker Engine | ≥ 20.10 | `docker version` |
| Docker Compose | v2（`docker compose` 子命令形式） | `docker compose version` |
| 端口 | 8080 空闲（可改） | `lsof -i :8080` |
| 架构 | amd64 或 arm64（镜像双架构都有） | `uname -m` |

> **Windows 用户**：把命令里的 `/var/run/docker.sock` 换成
> `//./pipe/docker_engine`（PowerShell 路径），或直接用 WSL2。

---

## 2. 三分钟快速开始

```bash
# 1. 拉代码
git clone https://github.com/steven754/dockermanage.git
cd dockermanage

# 2. 起服务
docker compose up -d

# 3. 看状态（应显示 healthy）
docker compose ps

# 4. 打开面板
open http://localhost:8080
```

**默认凭据**：`admin` / `admin` —— 登录后会**强制你立即改密**，改完才能进面板。

### 换成强密码启动（推荐，尤其公网机器）

弱默认凭据在「容器启动」到「你改完密码」之间是全网已知的。面板权限等同宿主机 root，
所以**第一次启动前**就定好密码，这个窗口就不存在：

```bash
# 方式一：指定你自己的密码
PANEL_INITIAL_PASSWORD='你的强密码' docker compose up -d

# 方式二：生成 16 位随机密码，从日志里取
PANEL_INITIAL_PASSWORD=random docker compose up -d
docker compose logs panel | grep '初始'
```

> `PANEL_INITIAL_PASSWORD` **只在首次启动（数据库里还没有用户）时生效一次**。
> 之后再改这个变量不会影响已有部署 —— 想改密码只能登录后去设置页。

---

## 3. 编排文件逐段解读

仓库里的 `docker-compose.yml` 全文如下（附逐段说明）：

```yaml
services:
  panel:
    image: stevensign/dockermanage:latest
    container_name: dockermanage
    restart: unless-stopped

    privileged: true

    ports:
      - "8080:8080"

    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./data:/data
      - ./compose:/compose

    environment:
      PANEL_PORT: "8080"
      PANEL_DATA: "/data"
      PANEL_COMPOSE_DIR: "/compose"
      PANEL_LOG_LEVEL: "info"
      PANEL_INITIAL_PASSWORD: "${PANEL_INITIAL_PASSWORD:-admin}"
      PANEL_TRUST_PROXY: "${PANEL_TRUST_PROXY:-false}"
      PANEL_LOGIN_RATE_LIMIT: "${PANEL_LOGIN_RATE_LIMIT:-10}"
      TZ: "${TZ:-Asia/Shanghai}"

    healthcheck:
      test: ["CMD", "curl", "-fsS", "http://127.0.0.1:8080/healthz"]
      interval: 30s
      timeout: 3s
      retries: 3
      start_period: 10s

    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"
```

### `privileged: true` — 为什么必须有

面板的核心能力是**管理本机 Docker**：启停容器、exec 进容器看终端、读日志、编排部署。
这些操作需要读写 Docker daemon 的 socket，容器内还需要足够的权限进入别的容器的
namespace。

这是 Portainer / Yacht 等同类产品的**共同做法**。代价是：**拿到面板的人 = 拿到宿主机 root**。
所以它只能跑在你信任的网络里。

> 想收敛权限？见 [§8 反向代理](#8-反向代理示例) 与 SPEC §8.1 的 Docker Socket Proxy 方案（v1.2 规划）。

### 三个挂载点

| 宿主路径 | 容器内 | 作用 | 能不能删 |
|---|---|---|---|
| `/var/run/docker.sock` | 同路径 | Docker daemon socket，**面板的全部能力来源** | ❌ 不能 |
| `./data` | `/data` | SQLite 数据库、设置、初始密码文件、JWT 密钥 | ❌ 不能（否则丢设置和历史） |
| `./compose` | `/compose` | 你在面板里创建的编排项目目录 | ❌ 不能（否则 compose 功能不可用） |

### 环境变量

| 变量 | 默认 | 生效时机 | 说明 |
|---|---|---|---|
| `PANEL_INITIAL_PASSWORD` | `admin` | **仅首次启动一次** | 首启初始密码；`random` = 生成 16 位随机密码 |
| `PANEL_TRUST_PROXY` | `false` | 重启即生效 | 是否采信 `X-Forwarded-For` / `X-Real-IP`。**仅在确实部署在反代之后时开启** |
| `PANEL_LOGIN_RATE_LIMIT` | `10` | 重启即生效 | 每分钟允许的登录尝试次数（容器化部署时该阈值是全局的） |
| `PANEL_LOG_LEVEL` | `info` | 重启即生效 | `debug` / `info` / `warn` / `error` |
| `TZ` | `Asia/Shanghai` | 重启即生效 | 容器时区，影响日志时间戳与界面显示 |

### 健康检查

每 30 秒打一次 `/healthz`，连续 3 次失败才算 unhealthy。`docker compose ps` 里看到
`(healthy)` 说明服务真的能响应，不只是「容器在跑」。

---

## 4. 首次登录与改密

```
用户名：admin
密码  ：admin
```

登录后系统会**在服务端强制**要求改密：未改密前访问任何业务接口都返回
`403 MUST_CHANGE_PASSWORD`（不依赖前端跳转，绕过前端也拿不到权限）。

改密成功后：
- 旧令牌立即失效（令牌版本自增，旧 JWT 直接 401）
- `data/.password` 文件被自动删除
- 改密接口会返回新令牌对，前端自动替换

> 忘了密码且没改过？删掉数据目录重来（会丢设置与指标历史）：
> ```bash
> docker compose down && rm -rf data && docker compose up -d
> ```
> 忘了密码**且已改过**：没有找回入口，只能删库重来。

---

## 5. 常用运维命令

```bash
# 状态（看 health）
docker compose ps

# 实时日志
docker compose logs -f panel

# 只看最近 100 行
docker compose logs --tail=100 panel

# 重启
docker compose restart panel

# 停止（保留数据）
docker compose down

# 停止并删除容器与网络（保留 ./data ./compose）
docker compose down

# 拉最新镜像并重建
docker compose pull && docker compose up -d

# 进去看容器内部
docker compose exec panel sh

# 查看健康检查详情
docker inspect dockermanage --format '{{json .State.Health}}'
```

---

## 6. 数据与持久化

```
dockermanage/
├── docker-compose.yml
├── data/              # 运行时数据（已 gitignore）
│   ├── dockermanage.db        # SQLite 主库（设置、指标历史、事件）
│   ├── dockermanage.db-wal    # WAL 日志
│   ├── dockermanage.db-shm    # 共享内存索引
│   ├── .jwt_secret            # JWT 签名密钥（0600）
│   └── .password              # 初始密码（0600，改密后自动删除）
└── compose/           # 面板里创建的编排项目（已 gitignore）
    └── <项目名>/
        └── docker-compose.yml
```

### 备份

> ⚠️ **别直接拷 `dockermanage.db`**。数据库跑在 WAL 模式下，数据大部分还在 `-wal` 文件里：
> ```
> dockermanage.db        4 KB   ← 只拷这个 = 近乎空库
> dockermanage.db-wal  259 KB   ← 真实数据在这
> backup.db (.backup)   77 KB   ← 一致性快照
> ```
> 要么停机整目录拷，要么用 SQLite 的在线备份命令。

```bash
# 方式一：停机整目录备份（最稳，连 .wal 一起带走）
docker compose down
tar czf dockermanage-backup-$(date +%Y%m%d).tar.gz data/ compose/
docker compose up -d

# 方式二：不停机在线备份（镜像内置 sqlite3 CLI）
docker exec dockermanage sqlite3 /data/dockermanage.db ".backup '/data/backup.db'"
docker cp dockermanage:/data/backup.db ./backup.db
docker exec dockermanage rm /data/backup.db   # 拷完清掉，别在数据目录里留垃圾

# 验证备份可用
sqlite3 ./backup.db "PRAGMA integrity_check;"   # 应输出 ok
sqlite3 ./backup.db ".tables"
```

> 方式二依赖运行时镜像里的 `sqlite3`（v0.1.1 起内置）。若你的镜像是更早的版本，
> 用方式一，或在宿主机装 `sqlite3` 后对拷出来的文件执行 `.backup`。

### 迁移到新机器

```bash
# 旧机器
docker compose down
tar czf dmbak.tar.gz data/ compose/

# 新机器
tar xzf dmbak.tar.gz          # 解到 dockermanage/ 目录下
docker compose up -d
```

`data/` 里的 JWT 密钥和密码哈希一起带走，换机后登录态不变。

---

## 7. 常见定制

### 改端口

```yaml
ports:
  - "9090:8080"     # 宿主 9090 → 容器 8080
```

### 改数据目录到别处

```yaml
volumes:
  - /srv/dockermanage/data:/data
  - /srv/dockermanage/compose:/compose
```

### 只监听本机（不暴露到局域网）

```yaml
ports:
  - "127.0.0.1:8080:8080"
```

配反向代理时用这个更安全。

### 局域网访问

面板默认只监听容器内的 8080，通过 `ports` 映射到宿主。局域网其他机器直接访问
`http://<宿主IP>:8080` 即可，无需额外配置。

### 改初始密码（仅首次）

见 [§2](#2-三分钟快速开始)。或在 `docker-compose.yml` 里写死：

```yaml
environment:
  PANEL_INITIAL_PASSWORD: "你的强密码"
```

### 开放更多日志

```yaml
logging:
  driver: json-file
  options:
    max-size: "50m"
    max-file: "5"
```

### 调试模式

```yaml
environment:
  PANEL_LOG_LEVEL: "debug"
```

```bash
docker compose up -d && docker compose logs -f panel
```

---

## 8. 反向代理示例

面板默认只提供 HTTP。**公网暴露务必挂反代 + TLS**。

先在编排文件里把端口改成只监听本机：

```yaml
ports:
  - "127.0.0.1:8080:8080"
environment:
  PANEL_TRUST_PROXY: "true"     # 让限流能拿到真实客户端 IP
```

### Caddy

```caddyfile
panel.example.com {
    reverse_proxy 127.0.0.1:8080
}
```

### Nginx

```nginx
server {
    listen 443 ssl http2;
    server_name panel.example.com;

    ssl_certificate     /etc/letsencrypt/live/panel.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/panel.example.com/privkey.pem;

    # WebSocket 升级（终端 / 日志流需要）
    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # SSE 实时流：关闭缓冲，否则事件会被攒着一起发
        proxy_buffering off;
        proxy_read_timeout 3600s;
    }
}
```

> `proxy_buffering off` 很关键 —— 不关的话 Dashboard 的实时图表和事件流不会逐条推送。

---

## 9. 故障排查

### 容器起不来 / 反复重启

```bash
docker compose logs --tail=50 panel
```

| 日志关键字 | 原因 | 处理 |
|---|---|---|
| `bind: address already in use` | 8080 被占 | 改 `ports` 里的宿主端口 |
| `unable to open database file` | `./data` 不可写 | `mkdir -p data && chmod 755 data` |
| `Docker daemon 不可达` | socket 路径不对 | 确认 `/var/run/docker.sock` 存在 |

`unable to open database file` 那条其实是 **Docker Desktop 上 bind mount 瞬时未就绪**，
程序会重试 10 次（每次间隔 300ms），通常第二遍就自愈了。日志里看到一串
`数据库暂时不可用，稍后重试` 是正常现象。

### 面板能打开但容器列表是空的

面板需要挂载 docker.sock。检查：

```bash
docker compose exec panel docker version     # 应能打印服务端版本
docker compose exec panel ls -la /var/run/docker.sock
```

### 登录一直失败

1. **触发了限流**：默认 10 次/分钟（全局）。等一分钟，或临时调大：
   ```bash
   PANEL_LOGIN_RATE_LIMIT=30 docker compose up -d
   ```
2. **密码不对**：试 `cat data/.password`（文件还在说明没改过初始密码）

### 改了密码后所有接口报 401

正常 —— 改密会让旧令牌失效。**刷新页面重新登录**即可。

### 拉不到镜像

```bash
docker compose pull          # 手动拉一次看报错
docker login                  # Docker Hub 限流时可能需要登录
```

### 端口冲突

```bash
lsof -i :8080                # 看谁占着
docker compose ps            # 看是不是上一个实例没停
```

---

## 10. 升级与回滚

```bash
# 升级
docker compose pull
docker compose up -d
docker compose logs --tail=20 panel   # 确认版本号
```

数据库迁移在启动时自动执行且**幂等**，升级不会丢数据。

### 回滚

```bash
docker compose down
# 把 docker-compose.yml 里的 image 改成旧 tag
docker compose up -d
```

旧 tag 在 Docker Hub 上都保留着，例如：

```bash
PANEL_IMAGE=stevensign/dockermanage:v0.1.0 docker compose up -d
```

> 镜像 tag：`latest`（滚动）、`v0.1.0`（固定版本）。生产环境建议**钉死版本号**，
> 别用 `latest`，否则某次 `pull` 会静默换掉你的版本。

---

## 11. 安全清单

上线前逐条确认：

- [ ] **首次启动前**设了 `PANEL_INITIAL_PASSWORD`（或用 `random`），没让弱默认暴露过
- [ ] 登录后**立刻改密**（服务端强制，但别拖）
- [ ] 端口没直接暴露到公网，或已挂反代 + TLS
- [ ] 若挂反代，`PANEL_TRUST_PROXY=true`（否则限流拿不到真实 IP）
- [ ] `data/` 目录权限收紧（`chmod 700 data`）
- [ ] 镜像钉死版本号，不用 `latest`
- [ ] 知道这面板权限等同宿主机 root —— 不要放在不可信的网络里
- [ ] 重要环境已备份 `data/`（见 [§6](#6-数据与持久化)）

---

## 相关文档

- [SPEC.md](../SPEC.md) — 技术规格说明书
- [docs/api-contract.md](./api-contract.md) — API 契约
- [ui-mocks/README.md](../ui-mocks/README.md) — UI 设计说明
