/**
 * API 类型定义 —— 与 docs/api-contract.md v1.0 严格一一对应。
 * 命名约定（契约 §0.3）：
 *  - 时间统一 RFC3339 UTC 字符串
 *  - 字节数为 number（*Bytes）
 *  - 百分比为 0-100 的 number（*Pct）
 *  - 速率 *BytesPerSec
 */

/* ---------------------------------------------------------------- 认证 */

/** POST /auth/login 响应 */
export interface LoginResponse {
  accessToken: string
  refreshToken: string
  /** access token 有效秒数 */
  expiresIn: number
  /** 仍在使用首启初始密码 → true，需强制改密 */
  mustChangePassword: boolean
  /** 仍在使用首启初始密码 → true，前端据此提示尽快修改 */
  defaultPasswordInUse: boolean
}

/** GET /auth/me 响应 */
export interface CurrentUser {
  id: number
  username: string
  mustChangePassword: boolean
}

/* ---------------------------------------------------------------- 通用 */

export interface ItemsResponse<T> {
  items: T[]
}

export interface OkResponse {
  ok: boolean
}

/** 错误响应：{ "error": { "code", "message" } } */
export interface ApiErrorBody {
  error: {
    code: string
    message: string
  }
}

/* ---------------------------------------------------------------- Dashboard */

export interface DashboardHost {
  name: string
  dockerVersion: string
  composeVersion: string
  os: string
  arch: string
  cpuCores: number
  uptimeSeconds: number
}

export interface DashboardResources {
  cpu: { usedPct: number; cores: number }
  memory: { usedBytes: number; totalBytes: number; usedPct: number }
  disk: { usedBytes: number; totalBytes: number; usedPct: number; mountpoint: string }
  network: { rxBytesPerSec: number; txBytesPerSec: number }
}

export interface DashboardContainers {
  total: number
  running: number
  stopped: number
  error: number
}

export interface DashboardImages {
  total: number
  totalSizeBytes: number
}

export interface DashboardVolumes {
  total: number
  orphan: number
}

export interface DashboardNetworks {
  total: number
}

export interface DashboardBuildCache {
  sizeBytes: number
}

/** GET /dashboard 响应 */
export interface DashboardResponse {
  host: DashboardHost
  resources: DashboardResources
  containers: DashboardContainers
  images: DashboardImages
  volumes: DashboardVolumes
  networks: DashboardNetworks
  buildCache: DashboardBuildCache
  recentEvents: DockerEvent[]
}

/* ---------------------------------------------------------------- 容器 */

export type ContainerState =
  | 'running'
  | 'exited'
  | 'paused'
  | 'created'
  | 'restarting'
  | 'dead'
  | 'removing'

export type ContainerHealth = 'healthy' | 'unhealthy' | 'starting' | 'none'

export interface ContainerPort {
  privatePort: number
  publicPort: number
  type: string
  ip: string
}

export interface ContainerNetwork {
  name: string
  ip: string
  aliases: string[]
}

export interface ContainerMount {
  type: string
  name: string
  source: string
  destination: string
  readOnly: boolean
}

export interface Container {
  id: string
  name: string
  image: string
  imageId: string
  state: ContainerState
  /** 人类可读状态，如 "Up 3 days" */
  status: string
  health: ContainerHealth
  exitCode: number
  createdAt: string
  startedAt: string
  finishedAt: string | null
  restartCount: number
  ports: ContainerPort[]
  networks: ContainerNetwork[]
  mounts: ContainerMount[]
  labels: Record<string, string>
  composeProject: string
  /** 实时字段可能为 null（采样有 1 秒延迟） */
  cpuPct: number | null
  memUsageBytes: number | null
  memLimitBytes: number | null
  netRxBytesPerSec: number | null
  netTxBytesPerSec: number | null
}

/** GET /containers/:id/stats 响应 */
export interface ContainerStats {
  cpuPct: number
  memUsageBytes: number
  memLimitBytes: number
  netRxBytesPerSec: number
  netTxBytesPerSec: number
  blockReadBytesPerSec: number
  blockWriteBytesPerSec: number
}

/** POST /containers/prune 响应 */
export interface PruneResult {
  deleted: string[]
  spaceReclaimedBytes: number
}

/** POST /containers/:id/update 响应 */
export interface UpdateResult {
  ok: boolean
  oldImage: string
  newImage: string
}

/* ---------------------------------------------------------------- 镜像 */

export interface Image {
  id: string
  repoTags: string[]
  repoDigests: string[]
  sizeBytes: number
  sharedSizeBytes: number
  createdAt: string
  dangling: boolean
  containerCount: number
}

/* ---------------------------------------------------------------- 卷 */

export interface Volume {
  name: string
  driver: string
  scope: string
  mountpoint: string
  sizeBytes: number
  /** 被几个容器引用 */
  refCount: number
  /** refCount === 0 */
  orphan: boolean
  createdAt: string
  labels: Record<string, string>
}

/* ---------------------------------------------------------------- 网络 */

export interface NetworkIpam {
  subnet: string
  gateway: string
}

export interface Network {
  id: string
  name: string
  driver: string
  scope: string
  internal: boolean
  attachable: boolean
  ipam: NetworkIpam
  containerCount: number
  createdAt: string
  labels: Record<string, string>
}

/* ---------------------------------------------------------------- 事件 */

export type EventType = 'container' | 'image' | 'volume' | 'network' | 'daemon'

export interface DockerEvent {
  id: number
  ts: string
  type: EventType
  action: string
  actorId: string
  actorName: string
  resourceId: string
  resourceName: string
  exitCode: number
  details: Record<string, unknown>
}

/* ---------------------------------------------------------------- 指标流 */

export interface SystemStats {
  ts: string
  cpuPct: number
  memUsedBytes: number
  memTotalBytes: number
  diskUsedBytes: number
  diskTotalBytes: number
  netRxBytesPerSec: number
  netTxBytesPerSec: number
}

export interface StreamedContainerStats {
  id: string
  name: string
  cpuPct: number
  memUsageBytes: number
  memLimitBytes: number
  netRxBytesPerSec: number
  netTxBytesPerSec: number
  blockReadBytesPerSec: number
  blockWriteBytesPerSec: number
}

export type StatsRange = '1h' | '24h' | '7d'

/** GET /stats/history 响应 */
export interface StatsHistory {
  range: StatsRange
  stepSeconds: number
  points: {
    ts: string
    cpuPct: number
    memUsageBytes: number
    netRxBytesPerSec: number
    netTxBytesPerSec: number
  }[]
}

/* ---------------------------------------------------------------- Compose */

export interface ComposeProjectSummary {
  name: string
  path: string
  fileCount: number
  files: string[]
  serviceCount: number
  runningCount: number
  updatedAt: string
}

export interface ComposeFile {
  name: string
  content: string
  sizeBytes: number
  updatedAt: string
}

export interface ComposeService {
  name: string
  image: string
  state: ContainerState
  status: string
  containerId: string
  ports: ContainerPort[]
}

export interface ComposeProject {
  name: string
  path: string
  files: ComposeFile[]
  services: ComposeService[]
}

/* ---------------------------------------------------------------- 设置 */

export type ThemeName = 'dark' | 'light' | 'system'

export interface Settings {
  theme: ThemeName
  registryMirror: string
  sampleIntervalSeconds: number
  retentionDays: number
  defaultPageSize: number
}

/* ---------------------------------------------------------------- 版本 */

export interface VersionInfo {
  version: string
  commit: string
  buildTime: string
}
