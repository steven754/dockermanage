import type { LucideIcon } from 'lucide-react'
import { AlertTriangle, TrendingDown, TrendingUp } from 'lucide-react'
import { cn } from '@/lib/utils'

export interface StatTrend {
  /** 变化幅度文案，如 "4.1%" */
  value: string
  direction: 'up' | 'down'
  tone: 'success' | 'warning' | 'danger' | 'muted'
}

/**
 * 资源卡片（对应设计稿 .elev-card.stat-card）。
 * 数值使用等宽字体 + count-up 入场动效，下方为 meter 进度条。
 */
export function StatCard({
  label,
  icon: Icon,
  meta,
  value,
  unit = '%',
  meter,
  hint,
  trend,
  className,
}: {
  label: string
  icon: LucideIcon
  meta?: string
  value: number | string | null | undefined
  unit?: string
  /** 进度条百分比（0-100）；不传则不渲染 */
  meter?: number
  /** 底部补充说明 */
  hint?: string
  trend?: StatTrend
  className?: string
}) {
  const display = value === null || value === undefined ? '—' : value
  const meterClass =
    meter === undefined ? '' : meter > 85 ? 'danger' : meter > 60 ? 'warn' : ''

  return (
    <div className={cn('elev-card stat-card p-4', className)}>
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <div className="flex h-8 w-8 items-center justify-center rounded-md bg-brand-soft">
            <Icon className="h-4 w-4 text-brand" />
          </div>
          <span className="text-sm text-fg-muted">{label}</span>
        </div>
        {meta ? <span className="font-mono text-[11px] text-fg-subtle">{meta}</span> : null}
      </div>

      <div className="mt-3 flex items-end gap-2">
        <span className="count-up font-mono text-3xl font-semibold tracking-tight">
          {display}
        </span>
        <span className="mb-1 text-sm text-fg-muted">{unit}</span>
        {trend ? (
          <span
            className={cn(
              'mb-1 ml-auto flex items-center gap-0.5 text-xs',
              trend.tone === 'success' && 'text-emerald-400',
              trend.tone === 'warning' && 'text-amber-400',
              trend.tone === 'danger' && 'text-red-400',
              trend.tone === 'muted' && 'text-fg-subtle',
            )}
          >
            {trend.direction === 'up' ? (
              <TrendingUp className="h-3 w-3" />
            ) : (
              <TrendingDown className="h-3 w-3" />
            )}
            {trend.tone === 'danger' ? <AlertTriangle className="h-3 w-3" /> : null}
            {trend.value}
          </span>
        ) : null}
      </div>

      {meter !== undefined ? (
        <div className="meter mt-3">
          <span className={meterClass} style={{ width: `${Math.min(Math.max(meter, 0), 100)}%` }} />
        </div>
      ) : null}

      {hint ? <div className="mt-1.5 font-mono text-[11px] text-fg-subtle">{hint}</div> : null}
    </div>
  )
}

/** 网络 I/O 卡片（双列 RX / TX 布局） */
export function NetworkStatCard({
  label,
  icon: Icon,
  meta,
  rxBytesPerSec,
  txBytesPerSec,
  className,
}: {
  label: string
  icon: LucideIcon
  meta?: string
  rxBytesPerSec: number | null | undefined
  txBytesPerSec: number | null | undefined
  className?: string
}) {
  const fmt = (value: number | null | undefined) => {
    if (value === null || value === undefined || Number.isNaN(value)) return { n: '—', u: '' }
    const mb = value / 1024 / 1024
    if (mb >= 1) return { n: mb.toFixed(mb >= 100 ? 0 : 1), u: 'MB/s' }
    return { n: (value / 1024).toFixed(0), u: 'KB/s' }
  }
  const rx = fmt(rxBytesPerSec)
  const tx = fmt(txBytesPerSec)

  return (
    <div className={cn('elev-card stat-card p-4', className)}>
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <div className="flex h-8 w-8 items-center justify-center rounded-md bg-brand-soft">
            <Icon className="h-4 w-4 text-brand" />
          </div>
          <span className="text-sm text-fg-muted">{label}</span>
        </div>
        {meta ? <span className="font-mono text-[11px] text-fg-subtle">{meta}</span> : null}
      </div>

      <div className="mt-3 grid grid-cols-2 gap-3">
        <div>
          <div className="flex items-center gap-1 text-[11px] text-fg-subtle">↓ RX</div>
          <div className="mt-0.5 font-mono text-lg font-semibold">
            {rx.n}
            <span className="text-xs text-fg-muted"> {rx.u}</span>
          </div>
        </div>
        <div>
          <div className="flex items-center gap-1 text-[11px] text-fg-subtle">↑ TX</div>
          <div className="mt-0.5 font-mono text-lg font-semibold">
            {tx.n}
            <span className="text-xs text-fg-muted"> {tx.u}</span>
          </div>
        </div>
      </div>
    </div>
  )
}
