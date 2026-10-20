import { cn } from '@/lib/utils'
import type { ContainerHealth, ContainerState } from '@/lib/types'

type BadgeKind = 'running' | 'stopped' | 'error' | 'paused' | 'info' | 'primary'

const KIND_BY_STATE: Record<ContainerState, BadgeKind> = {
  running: 'running',
  exited: 'stopped',
  created: 'stopped',
  restarting: 'info',
  paused: 'paused',
  dead: 'error',
  removing: 'error',
}

const DOT_BY_KIND: Record<BadgeKind, string> = {
  running: 'dot-running',
  stopped: 'dot-stopped',
  error: 'dot-error',
  paused: 'dot-paused',
  info: 'dot-paused',
  primary: 'dot-running',
}

const CLASS_BY_KIND: Record<BadgeKind, string> = {
  running: 'border-success/30 bg-success-soft text-success',
  stopped: 'border-fg-subtle/30 bg-fg-subtle/15 text-fg-muted',
  error: 'border-danger/30 bg-danger-soft text-danger',
  paused: 'border-warning/30 bg-warning-soft text-warning',
  info: 'border-info/30 bg-info-soft text-info',
  primary: 'border-brand/30 bg-brand-soft text-brand',
}

const LABEL_BY_KIND: Record<BadgeKind, string> = {
  running: '运行中',
  stopped: '已停止',
  error: '异常',
  paused: '已暂停',
  info: '重启中',
  primary: '正常',
}

/** 容器状态徽标（对应设计稿 .badge + .dot-*） */
export function StateBadge({
  state,
  label,
  exitCode,
  className,
}: {
  state: ContainerState
  label?: string
  /** 退出码：非 0 的 exited 显示为「异常」（设计稿里 redis-cache exit=137 是红色） */
  exitCode?: number
  className?: string
}) {
  const kind =
    state === 'exited' && exitCode !== undefined && exitCode !== 0
      ? 'error'
      : (KIND_BY_STATE[state] ?? 'stopped')
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-[11px] font-medium',
        CLASS_BY_KIND[kind],
        className,
      )}
    >
      <span className={cn('h-2 w-2 rounded-full', DOT_BY_KIND[kind])} />
      {label ?? LABEL_BY_KIND[kind]}
    </span>
  )
}

/** 健康检查徽标 */
export function HealthBadge({ health, className }: { health: ContainerHealth; className?: string }) {
  if (!health || health === 'none') return null
  const map: Record<Exclude<ContainerHealth, 'none'>, { cls: string; text: string }> = {
    healthy: { cls: 'border-success/30 bg-success-soft text-success', text: '健康' },
    unhealthy: { cls: 'border-danger/30 bg-danger-soft text-danger', text: '不健康' },
    starting: { cls: 'border-warning/30 bg-warning-soft text-warning', text: '检查中' },
  }
  const item = map[health]
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-[11px] font-medium',
        item.cls,
        className,
      )}
    >
      <span className="h-2 w-2 rounded-full bg-current" />
      {item.text}
    </span>
  )
}
