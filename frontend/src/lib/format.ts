import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import duration from 'dayjs/plugin/duration'
import 'dayjs/locale/zh-cn'

dayjs.extend(relativeTime)
dayjs.extend(duration)
dayjs.locale('zh-cn')

/**
 * 字节数格式化。
 * 单位基数 1024；`precision` 为小数位。
 * 负数按 0 处理，避免进度条出现负宽度。
 */
export function formatBytes(bytes: number | null | undefined, precision = 1): string {
  if (bytes === null || bytes === undefined || Number.isNaN(bytes)) return '—'
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
  const negative = bytes < 0
  const abs = Math.abs(bytes)
  const i = Math.min(Math.floor(Math.log(abs) / Math.log(1024)), units.length - 1)
  const value = abs / Math.pow(1024, i)
  const text = i === 0 ? `${Math.round(value)} ${units[i]}` : `${value.toFixed(precision)} ${units[i]}`
  return negative ? `-${text}` : text
}

/** 字节数 → 紧凑形式（用于图表轴 / 表格窄列） */
export function formatBytesCompact(bytes: number | null | undefined): string {
  if (bytes === null || bytes === undefined || Number.isNaN(bytes)) return '—'
  if (bytes === 0) return '0 B'
  const units = ['B', 'K', 'M', 'G', 'T']
  const i = Math.min(Math.floor(Math.log(Math.abs(bytes)) / Math.log(1024)), units.length - 1)
  const value = bytes / Math.pow(1024, i)
  return `${i === 0 ? Math.round(value) : value.toFixed(value >= 100 ? 0 : 1)}${units[i]}`
}

/** 速率格式化：bytes/s → "12.3 MB/s" */
export function formatBytesPerSec(bytes: number | null | undefined, precision = 1): string {
  if (bytes === null || bytes === undefined || Number.isNaN(bytes)) return '—'
  return `${formatBytes(bytes, precision)}/s`
}

/** 百分比：0-100 → "23.4%" */
export function formatPercent(pct: number | null | undefined, precision = 1): string {
  if (pct === null || pct === undefined || Number.isNaN(pct)) return '—'
  return `${pct.toFixed(precision)}%`
}

/** 相对时间："3 分钟前" */
export function formatRelativeTime(ts: string | null | undefined): string {
  if (!ts) return '—'
  const d = dayjs(ts)
  if (!d.isValid()) return '—'
  return d.fromNow()
}

/** 绝对时间："2026-10-20 15:30" */
export function formatDateTime(ts: string | null | undefined): string {
  if (!ts) return '—'
  const d = dayjs(ts)
  if (!d.isValid()) return '—'
  return d.format('YYYY-MM-DD HH:mm')
}

/** 仅日期："2026-10-20" */
export function formatDate(ts: string | null | undefined): string {
  if (!ts) return '—'
  const d = dayjs(ts)
  if (!d.isValid()) return '—'
  return d.format('YYYY-MM-DD')
}

/** 持续时长（秒 → "23d 7h" / "2h 15m" / "45s"） */
export function formatDuration(seconds: number | null | undefined): string {
  if (seconds === null || seconds === undefined || Number.isNaN(seconds) || seconds < 0) return '—'
  const s = Math.floor(seconds)
  const d = Math.floor(s / 86400)
  const h = Math.floor((s % 86400) / 3600)
  const m = Math.floor((s % 3600) / 60)
  const sec = s % 60
  if (d > 0) return `${d}d ${h}h`
  if (h > 0) return `${h}h ${m}m`
  if (m > 0) return `${m}m ${sec}s`
  return `${sec}s`
}

/** 容器状态的中文文案 */
export function formatContainerState(state: string): string {
  switch (state) {
    case 'running':
      return '运行中'
    case 'exited':
      return '已停止'
    case 'paused':
      return '已暂停'
    case 'created':
      return '已创建'
    case 'restarting':
      return '重启中'
    case 'dead':
      return '异常'
    case 'removing':
      return '删除中'
    default:
      return state
  }
}

/** 健康检查文案 */
export function formatHealth(health: string): string {
  switch (health) {
    case 'healthy':
      return '健康'
    case 'unhealthy':
      return '不健康'
    case 'starting':
      return '检查中'
    default:
      return '无'
  }
}

/** 端口展示：":80" / ":8080→:80" */
export function formatPorts(
  ports: { privatePort: number; publicPort: number; type: string }[] | undefined,
): string {
  if (!ports || ports.length === 0) return ''
  return ports
    .map((p) => (p.publicPort ? `:${p.publicPort}→:${p.privatePort}` : `:${p.privatePort}`))
    .join(' ')
}

/** 截断长文本，保留首尾 */
export function truncate(text: string, max = 60): string {
  if (text.length <= max) return text
  return `${text.slice(0, max - 1)}…`
}

/** 短 ID（容器 / 镜像 ID 前 12 位） */
export function shortId(id: string): string {
  return id.replace(/^sha256:/, '').slice(0, 12)
}

/** 容器「运行中」判定：paused / restarting 也算存活（与 docker ps 语义一致） */
export function isContainerRunning(state: string): boolean {
  return state === 'running' || state === 'paused' || state === 'restarting'
}

/** 容器「异常」判定：dead，或 exited 且退出码非 0（如 OOM killed = 137） */
export function isContainerAbnormal(state: string, exitCode?: number | null): boolean {
  if (state === 'dead') return true
  if (state === 'exited' && exitCode !== undefined && exitCode !== null && exitCode !== 0) {
    return true
  }
  return false
}

/** 容器「已停止」判定：exited 且退出码为 0，或仅创建未启动 */
export function isContainerStopped(state: string, exitCode?: number | null): boolean {
  if (state === 'created') return true
  return state === 'exited' && !isContainerAbnormal(state, exitCode)
}
