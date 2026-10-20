# API 契约 v1.0

> 本文件是前后端**唯一权威的接口形状来源**。SPEC.md §6 定义端点与语义，本文件定义请求体与响应体的确切 JSON 形状。
> 前后端并行开发时以本文件为准；如需变更，先改本文件再改代码。
> 基准日期：2026-10-20 · 对应 SPEC.md v1.0

---

## 0. 通用约定

### 0.1 前缀与鉴权

- 所有接口前缀 `/api/v1`
- 鉴权：`Authorization: Bearer <access_token>`（JWT）
- 除 `/api/v1/auth/login`、`/api/v1/version`、`/healthz`、`/readyz` 外全部需要鉴权
- 401 一律返回 `{"error":{"code":"UNAUTHORIZED","message":"..."}}`

**流式端点的鉴权例外（SSE / WebSocket）**

浏览器的 `EventSource` 与 `WebSocket` API **无法设置自定义请求头**，因此流式端点额外接受查询参数携带令牌：

```
GET /api/v1/stats/stream?access_token=<JWT>
GET /api/v1/events/stream?access_token=<JWT>
WS  /api/v1/containers/:id/logs?access_token=<JWT>
WS  /api/v1/containers/:id/exec?access_token=<JWT>
```

- 参数名 **`access_token`**（前端约定），后端同时兼容别名 `token`
- 优先级：`Authorization` 头 > `?access_token` > `?token`
- 非流式 REST 接口一律走 `Authorization` 头，不接受查询参数

### 0.2 错误格式

```json
{ "error": { "code": "CONTAINER_NOT_FOUND", "message": "Container abc not found" } }
```

| HTTP | code | 场景 |
|---|---|---|
| 400 | `VALIDATION_ERROR` | 参数不合法 |
| 401 | `UNAUTHORIZED` | 未登录 / token 过期 |
| 403 | `FORBIDDEN` | 无权限 |
| 404 | `NOT_FOUND` / `CONTAINER_NOT_FOUND` / `IMAGE_NOT_FOUND` | 资源不存在 |
| 409 | `CONFLICT` | 状态冲突（如容器已在运行） |
| 502 | `DOCKER_API_ERROR` | 上游 docker daemon 错误，message 透传 |
| 500 | `INTERNAL_ERROR` | 服务端异常 |

> HTTP 恒定返回对应状态码，错误信息一律在 `error.message`（中文，面向用户）。

### 0.3 命名约定

- 时间统一 **RFC3339 UTC 字符串**，如 `"2026-10-20T15:30:00Z"`
- 字节数统一 `*Bytes` 后缀的 **number**（JS 侧用 `Number()` 包装，注意 int64 精度）
- 百分比统一 `*Pct` 后缀的 **number**（0-100，非 0-1）
- 速率统一 `*BytesPerSec`
- 布尔字段用 `is` / `has` / `*Enabled` 前缀，避免 JS 关键字

### 0.4 分页

列表接口 v1.0 一次性返回全量（个人单机场景数据量小），不实现分页。但统一用 `{"items": [...]}` 包裹，前端按数组处理。

---

## 1. 认证

### POST /api/v1/auth/login

```jsonc
// 请求
{ "username": "admin", "password": "xxxxxxxx" }

// 200
{
  "accessToken": "eyJhbGciOi...",
  "refreshToken": "eyJhbGciOi...",
  "expiresIn": 3600,               // access token 有效秒数
  "mustChangePassword": true,      // 初始密码未改过 → true（强制改密）
  "defaultPasswordInUse": true     // 同上，前端据此提示尽快修改
}
```

> **初始凭据**：用户名固定 `admin`，初始密码默认 `admin`（与 Portainer 一致），
> 首次登录后系统强制要求改密。可用环境变量 `PANEL_INITIAL_PASSWORD` 覆盖，
> 设为 `random` 则生成 16 位随机密码。

### POST /api/v1/auth/password

```jsonc
// 请求
{ "oldPassword": "xxx", "newPassword": "yyy" }

// 200
{ "ok": true }
```

### POST /api/v1/auth/logout
```jsonc
{ "ok": true }
```

### GET /api/v1/auth/me
```jsonc
{
  "id": 1,
  "username": "admin",
  "mustChangePassword": false
}
```

---

## 2. Dashboard

### GET /api/v1/dashboard

```jsonc
{
  "host": {
    "name": "homeserver.local",
    "dockerVersion": "27.3.1",
    "composeVersion": "v2.29.0",
    "os": "Linux 6.8.0",
    "arch": "arm64",
    "cpuCores": 16,
    "uptimeSeconds": 2016000
  },
  "resources": {
    "cpu":    { "usedPct": 23.4, "cores": 16 },
    "memory": { "usedBytes": 28600000000, "totalBytes": 68719476736, "usedPct": 41.8 },
    "disk":   { "usedBytes": 1450000000, "totalBytes": 2147483648, "usedPct": 67.2, "mountpoint": "/var/lib/docker" },
    "network":{ "rxBytesPerSec": 1280000, "txBytesPerSec": 42000 }
  },
  "containers": { "total": 12, "running": 9, "stopped": 2, "error": 1 },
  "images":    { "total": 38, "totalSizeBytes": 12345678900 },
  "volumes":   { "total": 9, "orphan": 2 },
  "networks":  { "total": 4 },
  "buildCache":{ "sizeBytes": 2100000000 },
  "recentEvents": [ /* Event[]，见 §6 */ ]
}
```

---

## 3. 容器

### GET /api/v1/containers

Query：`status`（逗号分隔多值）、`search`（名称/镜像模糊）、`label`（`k=v`）、`limit`、`offset`

```jsonc
{
  "items": [
    {
      "id": "a3f3c2e8b1d4",
      "name": "nginx-proxy",
      "image": "nginx:1.27-alpine",
      "imageId": "sha256:abc123...",
      "state": "running",        // running|exited|paused|created|restarting|dead|removing
      "status": "Up 3 days",     // 人类可读状态
      "health": "healthy",       // healthy|unhealthy|starting|none
      "exitCode": 0,
      "createdAt": "2026-10-17T08:00:00Z",
      "startedAt": "2026-10-17T08:00:01Z",
      "finishedAt": null,
      "restartCount": 0,
      "ports": [
        { "privatePort": 80, "publicPort": 8080, "type": "tcp", "ip": "0.0.0.0" }
      ],
      "networks": [
        { "name": "proxy", "ip": "172.18.0.3", "aliases": ["nginx"] }
      ],
      "mounts": [
        { "type": "volume", "name": "nginx_data", "source": "/var/lib/docker/volumes/nginx_data/_data",
          "destination": "/data", "readOnly": false }
      ],
      "labels": { "com.docker.compose.project": "media-server" },
      "composeProject": "media-server",
      "cpuPct": 1.2,
      "memUsageBytes": 38000000,
      "memLimitBytes": 2147483648,
      "netRxBytesPerSec": 1200000,
      "netTxBytesPerSec": 45000
    }
  ]
}
```

> `cpuPct` / `memUsageBytes` 等实时字段在列表接口里为**可选**（可能为 null），因为采样子进程有 1 秒延迟。

### GET /api/v1/containers/:id
返回单个容器对象（形状同上）。

### GET /api/v1/containers/:id/json
返回 docker inspect 的**原始 JSON**（透传，前端 JSON 树展示）。

### POST /api/v1/containers/:id/{start|stop|restart|kill|pause|unpause}
```jsonc
{ "ok": true }
```

### DELETE /api/v1/containers/:id?force=true
```jsonc
{ "ok": true }
```

### POST /api/v1/containers/prune
```jsonc
// 请求
{ "all": false }
// 200
{ "deleted": [ "old1", "old2" ], "spaceReclaimedBytes": 1234567 }
```

### POST /api/v1/containers/:id/update
拉取新镜像并重建容器（保留原配置）。
```jsonc
{ "ok": true, "oldImage": "nginx:1.26", "newImage": "nginx:1.27" }
```

### GET /api/v1/containers/:id/stats
```jsonc
{
  "cpuPct": 1.2,
  "memUsageBytes": 38000000,
  "memLimitBytes": 2147483648,
  "netRxBytesPerSec": 1200000,
  "netTxBytesPerSec": 45000,
  "blockReadBytesPerSec": 0,
  "blockWriteBytesPerSec": 2048
}
```

### WS /api/v1/containers/:id/logs

Query：`tail`（默认 200）、`timestamps`（默认 true）、`follow`（默认 true）

- 服务端 → 客户端：**纯文本帧**，每行一条日志，已带 ANSI 颜色转义（后端不剥离）
- 客户端 → 服务端：仅支持 `{"type":"resize"}`（忽略）

### WS /api/v1/containers/:id/exec

Query：`cmd`（默认 `/bin/sh`）、`tty`（默认 true）、`user`（可选）

- 双向：终端输入输出走**文本帧**
- 后端透传 docker exec 的 TTY 输出

---

## 4. 镜像

### GET /api/v1/images
```jsonc
{
  "items": [
    {
      "id": "sha256:abc123...",
      "repoTags": ["nginx:1.27-alpine"],
      "repoDigests": ["nginx@sha256:def456..."],
      "sizeBytes": 142000000,
      "sharedSizeBytes": 12000000,
      "createdAt": "2026-10-01T00:00:00Z",
      "dangling": false,
      "containerCount": 2
    }
  ]
}
```

### POST /api/v1/images/pull
```jsonc
// 请求
{ "image": "nginx:1.27-alpine" }
// 200
{ "ok": true, "imageId": "sha256:abc123..." }
```

### POST /api/v1/images/:id/tag
```jsonc
{ "repo": "myrepo/nginx", "tag": "v1" }
→ { "ok": true }
```

### DELETE /api/v1/images/:id?force=false
```jsonc
{ "ok": true }
```

### POST /api/v1/images/prune
```jsonc
{ "dangling": true }
→ { "deleted": [ "sha256:abc" ], "spaceReclaimedBytes": 1234567 }
```

---

## 5. 卷与网络

### GET /api/v1/volumes
```jsonc
{
  "items": [
    {
      "name": "postgres_data",
      "driver": "local",
      "scope": "local",
      "mountpoint": "/var/lib/docker/volumes/postgres_data/_data",
      "sizeBytes": 123456789,
      "refCount": 2,          // 被几个容器引用
      "orphan": false,        // refCount == 0
      "createdAt": "2026-09-01T00:00:00Z",
      "labels": {}
    }
  ]
}
```

### GET /api/v1/networks
```jsonc
{
  "items": [
    {
      "id": "net123",
      "name": "proxy",
      "driver": "bridge",
      "scope": "local",
      "internal": false,
      "attachable": false,
      "ipam": { "subnet": "172.18.0.0/16", "gateway": "172.18.0.1" },
      "containerCount": 5,
      "createdAt": "2026-09-01T00:00:00Z",
      "labels": {}
    }
  ]
}
```

### DELETE /api/v1/volumes/:name?force=false / DELETE /api/v1/networks/:id
```jsonc
{ "ok": true }
```

---

## 6. 事件

### Event 对象

```jsonc
{
  "id": 123,
  "ts": "2026-10-20T15:30:00Z",
  "type": "container",        // container|image|volume|network|daemon
  "action": "die",            // create|start|die|stop|kill|restart|pause|unpause|destroy|
                               // pull|push|tag|remove|prune|connect|disconnect|...
  "actorId": "a3f3c2e8b1d4",
  "actorName": "nginx-proxy",
  "resourceId": "a3f3c2e8b1d4",
  "resourceName": "nginx-proxy",
  "exitCode": 137,
  "details": { "signal": 9, "image": "nginx:1.27-alpine" }
}
```

### GET /api/v1/events?limit=100&since=<ts>&type=container
```jsonc
{ "items": [ /* Event[] */ ] }
```

### GET /api/v1/events/stream（SSE）

```
event: event
data: {"id":123,"ts":"2026-10-20T15:30:00Z","type":"container","action":"die",...}

```

---

## 7. 指标流与历史

### GET /api/v1/stats/stream（SSE）

系统级与容器级共用一个连接，事件名区分：

```
event: system
data: {"ts":"2026-10-20T15:30:00Z","cpuPct":23.4,"memUsedBytes":28600000000,
       "memTotalBytes":68719476736,"diskUsedBytes":1450000000,"diskTotalBytes":2147483648,
       "netRxBytesPerSec":1280000,"netTxBytesPerSec":42000}

event: container
data: {"id":"a3f3c2e8b1d4","name":"nginx-proxy","cpuPct":1.2,"memUsageBytes":38000000,
       "memLimitBytes":2147483648,"netRxBytesPerSec":1200000,"netTxBytesPerSec":45000,
       "blockReadBytesPerSec":0,"blockWriteBytesPerSec":2048}
```

采样间隔：**10 秒**（决策 D-C1）。心跳：每 15 秒发一条 `: keepalive` 注释行。

### GET /api/v1/stats/history

Query：`range`（`1h` | `24h` | `7d`，默认 `1h`）、`containerId`（可选，省略则返回系统级）

```jsonc
{
  "range": "1h",
  "stepSeconds": 10,
  "points": [
    { "ts": "2026-10-20T14:30:00Z", "cpuPct": 23.4, "memUsageBytes": 28600000000,
      "netRxBytesPerSec": 1280000, "netTxBytesPerSec": 42000 }
  ]
}
```

---

## 8. Compose

### GET /api/v1/compose/projects
```jsonc
{
  "items": [
    {
      "name": "media-server",
      "path": "/compose/media-server",
      "fileCount": 3,
      "files": ["docker-compose.yml", ".env", "README.md"],
      "serviceCount": 4,
      "runningCount": 2,
      "updatedAt": "2026-10-19T22:10:00Z"
    }
  ]
}
```

### GET /api/v1/compose/projects/:name
```jsonc
{
  "name": "media-server",
  "path": "/compose/media-server",
  "files": [
    { "name": "docker-compose.yml", "content": "services:\n  ...", "sizeBytes": 1234, "updatedAt": "..." }
  ],
  "services": [
    { "name": "qbittorrent", "image": "ghcr.io/linuxserver/qbittorrent:latest", "state": "running",
      "status": "Up 2 hours", "containerId": "abc", "ports": [{ "privatePort": 8080, "publicPort": 9091, "type": "tcp" }] }
  ]
}
```

### PUT /api/v1/compose/projects/:name/files/:filename
```jsonc
{ "content": "services:\n  ..." }
→ { "ok": true, "sizeBytes": 1234 }
```

### POST /api/v1/compose/projects/:name/up
```jsonc
{ "services": ["web"] }     // 省略 = 全部
→ { "ok": true, "started": ["web", "db"] }
```

### POST /api/v1/compose/projects/:name/down
```jsonc
{ "removeVolumes": false }
→ { "ok": true }
```

### GET /api/v1/compose/projects/:name/ps
```jsonc
{ "items": [ /* 服务状态，形状同 §8 services */ ] }
```

---

## 9. 设置

### GET /api/v1/settings
```jsonc
{
  "theme": "dark",                  // dark | light | system
  "registryMirror": "",
  "sampleIntervalSeconds": 10,
  "retentionDays": 7,
  "defaultPageSize": 50
}
```

### PUT /api/v1/settings
```jsonc
{ "theme": "light", "registryMirror": "https://docker.1ms.run" }
→ { "ok": true }
```

---

## 10. 其它

### GET /api/v1/version
```jsonc
{ "version": "v0.1.0", "commit": "50e8447", "buildTime": "2026-10-20T15:00:00Z" }
```

### GET /healthz
```jsonc
{ "ok": true }
```

### GET /readyz
```jsonc
{ "ok": true, "docker": true, "database": true }
```
