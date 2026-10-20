import { useMemo } from 'react'
import {
  Area,
  CartesianGrid,
  ComposedChart,
  Line,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { cn } from '@/lib/utils'

export interface ChartPoint {
  ts: number
  label: string
  /** CPU 百分比 */
  cpu: number
  /** 内存百分比 */
  mem: number
  /** 网络接收 MB/s */
  rx: number
  /** 网络发送 MB/s */
  tx: number
}

export type ChartRange = '1m' | '30m' | '1h' | '24h'

const RANGE_OPTIONS: { value: ChartRange; label: string; windowMs: number }[] = [
  { value: '1m', label: '1m', windowMs: 60_000 },
  { value: '30m', label: '30m', windowMs: 30 * 60_000 },
  { value: '1h', label: '1h', windowMs: 60 * 60_000 },
  { value: '24h', label: '24h', windowMs: 24 * 60 * 60_000 },
]

/** 主机资源实时折线图（recharts AreaChart + 渐变填充） */
export function ResourceChart({
  data,
  range,
  onRangeChange,
  subtitle,
  lastLabel,
  className,
}: {
  data: ChartPoint[]
  range: ChartRange
  onRangeChange: (range: ChartRange) => void
  subtitle?: string
  lastLabel?: string
  className?: string
}) {
  const current = RANGE_OPTIONS.find((option) => option.value === range) ?? RANGE_OPTIONS[1]

  const axisTicks = useMemo(
    () => [0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100],
    [],
  )

  return (
    <div className={cn('elev-card p-5', className)}>
      <div className="mb-4 flex items-center justify-between">
        <div>
          <h2 className="text-base font-semibold">主机资源 · 实时</h2>
          <p className="mt-0.5 text-xs text-fg-muted">{subtitle ?? `最近 ${current.label} · 实时推送`}</p>
        </div>
        <div className="flex items-center gap-0.5 rounded-md border border-border bg-bg-base p-0.5">
          {RANGE_OPTIONS.map((option) => (
            <button
              key={option.value}
              type="button"
              onClick={() => onRangeChange(option.value)}
              className={cn(
                'rounded px-2.5 py-1 text-xs transition-colors',
                option.value === range
                  ? 'bg-bg-card font-medium text-fg-primary'
                  : 'text-fg-muted hover:text-fg-primary',
              )}
            >
              {option.label}
            </button>
          ))}
        </div>
      </div>

      <div className="h-64">
        <ResponsiveContainer width="100%" height="100%">
          <ComposedChart data={data} margin={{ top: 8, right: 8, left: -18, bottom: 0 }}>
            <defs>
              <linearGradient id="dm-cpu-fill" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor="#3b82f6" stopOpacity={0.4} />
                <stop offset="100%" stopColor="#3b82f6" stopOpacity={0} />
              </linearGradient>
            </defs>
            <CartesianGrid stroke="rgb(var(--border))" strokeDasharray="3 3" vertical={false} />
            <XAxis
              dataKey="label"
              tick={{ fontSize: 11, fill: 'rgb(var(--fg-subtle))' }}
              axisLine={false}
              tickLine={false}
              minTickGap={24}
            />
            <YAxis
              yAxisId="pct"
              domain={[0, 100]}
              ticks={axisTicks}
              tick={{ fontSize: 11, fill: 'rgb(var(--fg-subtle))' }}
              axisLine={false}
              tickLine={false}
              width={48}
            />
            <YAxis
              yAxisId="net"
              orientation="right"
              tick={{ fontSize: 11, fill: 'rgb(var(--fg-subtle))' }}
              axisLine={false}
              tickLine={false}
              width={44}
              tickFormatter={(value: number) => `${value}`}
            />
            <Tooltip content={<ChartTooltip />} cursor={{ stroke: 'rgb(var(--border-strong))' }} />
            <Area
              yAxisId="pct"
              type="monotone"
              dataKey="cpu"
              stroke="#3b82f6"
              strokeWidth={2}
              dot={false}
              fill="url(#dm-cpu-fill)"
              isAnimationActive={false}
              name="CPU %"
            />
            <Line
              yAxisId="pct"
              type="monotone"
              dataKey="mem"
              stroke="#06b6d4"
              strokeWidth={2}
              dot={false}
              isAnimationActive={false}
              name="内存 %"
            />
            <Line
              yAxisId="net"
              type="monotone"
              dataKey="rx"
              stroke="#10b981"
              strokeWidth={2}
              strokeDasharray="4 4"
              dot={false}
              isAnimationActive={false}
              name="网络 RX MB/s"
            />
          </ComposedChart>
        </ResponsiveContainer>
      </div>

      <div className="mt-3 flex items-center gap-4 text-xs text-fg-muted">
        <span className="flex items-center gap-1.5">
          <span className="h-2 w-2 rounded-full bg-blue-500" />CPU %
        </span>
        <span className="flex items-center gap-1.5">
          <span className="h-2 w-2 rounded-full bg-cyan-400" />内存 %
        </span>
        <span className="flex items-center gap-1.5">
          <span className="h-2 w-2 rounded-full bg-emerald-500" />网络 RX MB/s
        </span>
        {lastLabel ? (
          <span className="ml-auto font-mono text-fg-subtle">
            last: <span className="text-fg-secondary">{lastLabel}</span>
          </span>
        ) : null}
      </div>
    </div>
  )
}

interface TooltipPayloadEntry {
  dataKey?: string | number
  value?: number | string
  color?: string
  name?: string
}

function ChartTooltip({
  active,
  payload,
  label,
}: {
  active?: boolean
  payload?: TooltipPayloadEntry[]
  label?: string | number
}) {
  if (!active || !payload || payload.length === 0) return null
  return (
    <div className="rounded-lg border border-border bg-bg-card px-3 py-2 text-[11px] shadow-md">
      <div className="mb-1 font-mono text-fg-subtle">{label}</div>
      {payload.map((entry) => (
        <div key={String(entry.dataKey)} className="flex items-center gap-2">
          <span className="h-2 w-2 rounded-full" style={{ background: entry.color }} />
          <span className="text-fg-muted">{entry.name}</span>
          <span className="ml-auto font-mono text-fg-primary">{entry.value}</span>
        </div>
      ))}
    </div>
  )
}
