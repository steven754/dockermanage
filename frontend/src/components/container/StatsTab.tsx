import { useMemo } from 'react'
import { Area, AreaChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts'
import { useStatsHistory } from '@/hooks/useStats'
import { useLiveStats } from '@/hooks/useLiveStats'
import { formatBytes, formatBytesPerSec } from '@/lib/format'
import { cn } from '@/lib/utils'

interface Series {
  ts: number
  label: string
  value: number
}

interface MiniChartProps {
  title: string
  hint?: string
  color: string
  data: Series[]
  unit: string
  /** 数值格式化 */
  format?: (value: number) => string
  domain?: [number | 'auto' | 'dataMin', number | 'auto' | 'dataMax']
  /** 是否渐变填充 */
  fill?: boolean
}

function MiniChart({
  title,
  hint,
  color,
  data,
  unit,
  format,
  domain = ['auto', 'auto'],
  fill = true,
}: MiniChartProps) {
  const gradientId = useMemo(
    () => `fill-${title.replace(/\s+/g, '-')}-${color.replace('#', '')}`,
    [title, color],
  )
  const fmt = format ?? ((value: number) => value.toFixed(1))

  return (
    <div className="elev-card p-5">
      <div className="mb-3 flex items-center justify-between">
        <h3 className="text-sm font-semibold">{title}</h3>
        <span className="text-[11px] text-fg-subtle">{hint ?? '最近 5 分钟'}</span>
      </div>
      <div className="h-40">
        {data.length === 0 ? (
          <div className="flex h-full items-center justify-center text-[11px] text-fg-subtle">
            暂无采样数据
          </div>
        ) : (
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={data} margin={{ top: 4, right: 4, left: 0, bottom: 0 }}>
              <defs>
                <linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor={color} stopOpacity={0.4} />
                  <stop offset="100%" stopColor={color} stopOpacity={0} />
                </linearGradient>
              </defs>
              <XAxis dataKey="label" hide />
              <YAxis domain={domain} hide />
              <Tooltip
                contentStyle={{
                  background: 'rgb(var(--bg-card))',
                  border: '1px solid rgb(var(--border))',
                  borderRadius: 8,
                  fontSize: 11,
                  color: 'rgb(var(--fg-primary))',
                }}
                labelStyle={{ color: 'rgb(var(--fg-subtle))' }}
                formatter={(value: number) => [`${fmt(value)} ${unit}`, title]}
              />
              <Area
                type="monotone"
                dataKey="value"
                stroke={color}
                strokeWidth={2}
                dot={false}
                fill={fill ? `url(#${gradientId})` : 'transparent'}
                isAnimationActive={false}
              />
            </AreaChart>
          </ResponsiveContainer>
        )}
      </div>
    </div>
  )
}

/** 容器详情 · Stats 标签页 */
export function StatsTab({ containerId }: { containerId: string }) {
  const history = useStatsHistory('1h', containerId)
  const live = useLiveStats()

  const liveSeries = useMemo(() => live.containerBuffer[containerId] ?? [], [live.containerBuffer, containerId])
  const historyPoints = useMemo(() => history.data?.points ?? [], [history.data])

  const cpuData = useMemo<Series[]>(
    () =>
      historyPoints.map((point) => {
        const ts = Date.parse(point.ts)
        return {
          ts,
          label: new Date(ts).toLocaleTimeString('zh-CN', {
            hour: '2-digit',
            minute: '2-digit',
          }),
          value: point.cpuPct,
        }
      }),
    [historyPoints],
  )

  const memData = useMemo<Series[]>(
    () =>
      historyPoints.map((point) => {
        const ts = Date.parse(point.ts)
        return {
          ts,
          label: new Date(ts).toLocaleTimeString('zh-CN', {
            hour: '2-digit',
            minute: '2-digit',
          }),
          value: point.memUsageBytes / 1024 / 1024,
        }
      }),
    [historyPoints],
  )

  const netData = useMemo<Series[]>(
    () =>
      historyPoints.map((point) => {
        const ts = Date.parse(point.ts)
        return {
          ts,
          label: new Date(ts).toLocaleTimeString('zh-CN', {
            hour: '2-digit',
            minute: '2-digit',
          }),
          value: point.netRxBytesPerSec / 1024 / 1024,
        }
      }),
    [historyPoints],
  )

  // 块 I/O 历史接口未提供，退化为使用实时推送序列
  const blockData = useMemo<Series[]>(
    () =>
      liveSeries.map((point) => ({
        ts: Date.parse(new Date().toISOString()),
        label: new Date().toLocaleTimeString('zh-CN', {
          hour: '2-digit',
          minute: '2-digit',
          second: '2-digit',
        }),
        value: point.blockReadBytesPerSec / 1024 / 1024,
      })),
    [liveSeries],
  )

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-1 gap-4 xl:grid-cols-2">
        <MiniChart
          title="CPU 使用率"
          color="#3b82f6"
          data={cpuData}
          unit="%"
          domain={[0, 100]}
        />
        <MiniChart
          title="内存使用"
          color="#06b6d4"
          data={memData}
          unit="MB"
          format={(value) => value.toFixed(0)}
        />
        <MiniChart
          title="网络吞吐"
          color="#10b981"
          data={netData}
          unit="MB/s"
          format={(value) => value.toFixed(2)}
        />
        <MiniChart
          title="块 I/O"
          color="#a78bfa"
          data={blockData}
          unit="MB/s"
          hint="实时推送"
          format={(value) => value.toFixed(2)}
        />
      </div>

      {liveSeries.length > 0 ? (
        <div className="elev-card p-5">
          <h3 className="mb-3 text-sm font-semibold">实时快照</h3>
          <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
            <Snapshot label="CPU" value={`${liveSeries[liveSeries.length - 1].cpuPct.toFixed(1)}%`} />
            <Snapshot
              label="内存"
              value={formatBytes(liveSeries[liveSeries.length - 1].memUsageBytes)}
            />
            <Snapshot
              label="网络 RX"
              value={formatBytesPerSec(liveSeries[liveSeries.length - 1].netRxBytesPerSec)}
            />
            <Snapshot
              label="网络 TX"
              value={formatBytesPerSec(liveSeries[liveSeries.length - 1].netTxBytesPerSec)}
            />
          </div>
        </div>
      ) : null}
    </div>
  )
}

function Snapshot({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="text-[11px] text-fg-subtle">{label}</div>
      <div className={cn('mt-0.5 font-mono text-base font-semibold')}>{value}</div>
    </div>
  )
}
