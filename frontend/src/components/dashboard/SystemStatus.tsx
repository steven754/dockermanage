import { formatBytes } from '@/lib/format'
import { cn } from '@/lib/utils'

export interface SystemStatusItem {
  label: string
  value: string
  /** 指示点类型 */
  dot: 'running' | 'stopped' | 'error' | 'paused'
}

/** 系统状态列表（设计稿 Dashboard 右下） */
export function SystemStatus({
  items,
  className,
}: {
  items: SystemStatusItem[]
  className?: string
}) {
  return (
    <div className={cn('mt-5 border-t border-border pt-5', className)}>
      <h3 className="mb-3 text-xs uppercase tracking-wider text-fg-subtle">系统状态</h3>
      <ul className="space-y-2 text-sm">
        {items.map((item) => (
          <li key={item.label} className="flex items-center justify-between">
            <span className="flex items-center gap-2">
              <span className={cn('h-2 w-2 rounded-full', `dot-${item.dot}`)} />
              {item.label}
            </span>
            <span className="font-mono text-xs text-fg-muted">{item.value}</span>
          </li>
        ))}
      </ul>
    </div>
  )
}

/** 把 dashboard 数据转成系统状态条目 */
export function buildSystemStatusItems(dashboard: {
  host?: { dockerVersion: string; composeVersion: string }
  buildCache?: { sizeBytes: number }
  images?: { total: number }
  volumes?: { total: number; orphan: number }
  networks?: { total: number }
} | null | undefined): SystemStatusItem[] {
  return [
    {
      label: 'Docker 引擎',
      value: dashboard?.host?.dockerVersion ?? '—',
      dot: dashboard ? 'running' : 'error',
    },
    {
      label: 'Compose',
      value: dashboard?.host?.composeVersion ?? '—',
      dot: dashboard ? 'running' : 'error',
    },
    {
      label: '构建缓存',
      value: dashboard?.buildCache ? formatBytes(dashboard.buildCache.sizeBytes) : '—',
      dot: 'running',
    },
    {
      label: '镜像备份',
      value: dashboard?.images ? `${dashboard.images.total} 个镜像` : '—',
      dot: 'paused',
    },
  ]
}
