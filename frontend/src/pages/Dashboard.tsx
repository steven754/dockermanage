import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  ArrowRight,
  Cpu,
  Download,
  HardDrive,
  MemoryStick,
  MoreHorizontal,
  Network,
  Play,
  Plus,
  RefreshCw,
  Search,
  Square,
  Trash2,
} from 'lucide-react'
import { toast } from 'sonner'
import { PageHeader } from '@/components/layout/PageHeader'
import { StatCard, NetworkStatCard, type StatTrend } from '@/components/dashboard/StatCard'
import { ResourceChart, type ChartPoint, type ChartRange } from '@/components/dashboard/ResourceChart'
import { ContainerCard } from '@/components/dashboard/ContainerCard'
import { EventStream } from '@/components/dashboard/EventStream'
import { QuickActions, type QuickAction } from '@/components/dashboard/QuickActions'
import { SystemStatus, buildSystemStatusItems } from '@/components/dashboard/SystemStatus'
import { EmptyState, ErrorState } from '@/components/common/States'
import { ConfirmDialog } from '@/components/common/ConfirmDialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/input'
import { useDashboard, useContainers, useContainerActions } from '@/hooks/useContainers'
import { useLiveStats } from '@/hooks/useLiveStats'
import { useEventStream } from '@/hooks/useEventStream'
import { useStatsHistory } from '@/hooks/useStats'
import { containerApi } from '@/lib/api'
import {
  formatBytes,
  formatRelativeTime,
  isContainerAbnormal,
  isContainerRunning,
  isContainerStopped,
} from '@/lib/format'
import { cn } from '@/lib/utils'

type BulkAction = 'startAll' | 'stopAll' | 'prune' | 'updateAll'

const BULK_META: Record<BulkAction, { title: string; description: string }> = {
  startAll: { title: '启动全部容器？', description: '将启动所有处于「已停止」或「已创建」状态的容器。' },
  stopAll: { title: '停止全部容器？', description: '将停止所有运行中的容器，业务会短暂中断。' },
  prune: { title: '清理已停止的容器？', description: '将删除所有已停止的容器并回收磁盘空间，此操作不可撤销。' },
  updateAll: { title: '一键更新全部镜像？', description: '将拉取最新镜像并重建所有运行中的容器。' },
}

/** Dashboard —— 首屏门面 */
export function Dashboard() {
  const navigate = useNavigate()
  const { data: dashboard, isLoading, error, refetch } = useDashboard()
  const { data: containersData } = useContainers()
  const { prune } = useContainerActions()
  const live = useLiveStats()
  const { events } = useEventStream(50)

  const [range, setRange] = useState<ChartRange>('30m')
  const [filter, setFilter] = useState<'all' | 'running' | 'stopped' | 'error'>('all')
  const [keyword, setKeyword] = useState('')
  const [bulkAction, setBulkAction] = useState<BulkAction | null>(null)

  const containers = useMemo(() => containersData?.items ?? [], [containersData])

  const running = containers.filter((c) => isContainerRunning(c.state))
  const stopped = containers.filter((c) => isContainerStopped(c.state, c.exitCode))
  const errored = containers.filter((c) => isContainerAbnormal(c.state, c.exitCode))

  /* ---------------------------------------------------------- 资源数值 */

  const cpuPct = live.system?.cpuPct ?? dashboard?.resources.cpu.usedPct ?? 0
  const memUsed = live.system?.memUsedBytes ?? dashboard?.resources.memory.usedBytes ?? 0
  const memTotal =
    live.system?.memTotalBytes ?? dashboard?.resources.memory.totalBytes ?? 0
  const memPct =
    live.system && live.system.memTotalBytes > 0
      ? (live.system.memUsedBytes / live.system.memTotalBytes) * 100
      : (dashboard?.resources.memory.usedPct ?? 0)
  const diskPct = dashboard?.resources.disk.usedPct ?? 0
  const rxBytes = live.system?.netRxBytesPerSec ?? dashboard?.resources.network.rxBytesPerSec ?? 0
  const txBytes = live.system?.netTxBytesPerSec ?? dashboard?.resources.network.txBytesPerSec ?? 0

  // 与上一次采样值比较，得到趋势
  const prevRef = useRef<{ cpu: number; mem: number } | null>(null)
  const trendOf = useCallback((current: number, key: 'cpu' | 'mem'): StatTrend | undefined => {
    const prev = prevRef.current
    if (!prev) return undefined
    const before = prev[key]
    const delta = current - before
    if (Math.abs(delta) < 0.05) return undefined
    const tone = key === 'mem' ? (delta > 0 ? 'warning' : 'success') : delta > 0 ? 'warning' : 'success'
    return {
      value: `${Math.abs(delta).toFixed(1)}%`,
      direction: delta > 0 ? 'up' : 'down',
      tone,
    }
  }, [])

  // 记录上一次采样值，供趋势计算使用
  useEffect(() => {
    prevRef.current = { cpu: cpuPct, mem: memPct }
  }, [cpuPct, memPct])

  const cpuTrend = trendOf(cpuPct, 'cpu')
  const memTrend = trendOf(memPct, 'mem')

  /* ---------------------------------------------------------- 图表数据 */

  const history = useStatsHistory(range === '1h' || range === '24h' ? range : '1h')

  const chartData = useMemo<ChartPoint[]>(() => {
    const now = Date.now()
    const windowMs = range === '1m' ? 60_000 : range === '30m' ? 30 * 60_000 : 0
    const total = memTotal || dashboard?.resources.memory.totalBytes || 0
    const relativeLabel = (ts: number) =>
      range === '1h' || range === '24h'
        ? new Date(ts).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
        : `${Math.max(0, Math.round((now - ts) / 1000))}s`

    // 历史点（REST）+ 实时点（SSE）合并，按时间升序，同一时刻以实时点为准
    const merged = new Map<number, ChartPoint>()

    for (const point of history.data?.points ?? []) {
      const ts = Date.parse(point.ts)
      if (!Number.isFinite(ts)) continue
      if (windowMs > 0 && ts < now - windowMs) continue
      merged.set(ts, {
        ts,
        label: relativeLabel(ts),
        cpu: point.cpuPct,
        mem: total > 0 ? (point.memUsageBytes / total) * 100 : 0,
        rx: point.netRxBytesPerSec / 1024 / 1024,
        tx: point.netTxBytesPerSec / 1024 / 1024,
      })
    }

    for (const point of live.buffer) {
      const ts = Date.parse(point.ts)
      if (!Number.isFinite(ts)) continue
      if (windowMs > 0 && ts < now - windowMs) continue
      merged.set(ts, {
        ts,
        label: relativeLabel(ts),
        cpu: point.cpuPct,
        mem:
          point.memTotalBytes > 0 ? (point.memUsedBytes / point.memTotalBytes) * 100 : 0,
        rx: point.netRxBytesPerSec / 1024 / 1024,
        tx: point.netTxBytesPerSec / 1024 / 1024,
      })
    }

    return [...merged.values()].sort((a, b) => a.ts - b.ts)
  }, [range, live.buffer, history.data, memTotal, dashboard])

  /* ---------------------------------------------------------- 容器列表 */

  const filteredContainers = useMemo(() => {
    let list = containers
    if (filter === 'running') list = list.filter((c) => isContainerRunning(c.state))
    if (filter === 'stopped') list = list.filter((c) => isContainerStopped(c.state, c.exitCode))
    if (filter === 'error') list = list.filter((c) => isContainerAbnormal(c.state, c.exitCode))
    const kw = keyword.trim().toLowerCase()
    if (kw) {
      list = list.filter(
        (c) => c.name.toLowerCase().includes(kw) || c.image.toLowerCase().includes(kw),
      )
    }
    return list
  }, [containers, filter, keyword])

  /* ---------------------------------------------------------- 快捷操作 */

  const executeBulk = async (action: BulkAction) => {
    setBulkAction(null)
    try {
      if (action === 'startAll') {
        const targets = containers.filter((c) => isContainerStopped(c.state, c.exitCode))
        await Promise.allSettled(targets.map((c) => containerApi.action(c.id, 'start')))
        toast.success(`已启动 ${targets.length} 个容器`)
      } else if (action === 'stopAll') {
        const targets = containers.filter((c) => c.state === 'running')
        await Promise.allSettled(targets.map((c) => containerApi.action(c.id, 'stop')))
        toast.success(`已停止 ${targets.length} 个容器`)
      } else if (action === 'prune') {
        const result = await prune.mutateAsync(false)
        toast.success(
          `已清理 ${result.deleted.length} 个容器，回收 ${formatBytes(result.spaceReclaimedBytes)}`,
        )
      } else if (action === 'updateAll') {
        const targets = containers.filter((c) => c.state === 'running')
        await Promise.allSettled(targets.map((c) => containerApi.update(c.id)))
        toast.success(`已更新 ${targets.length} 个容器`)
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '操作失败')
    }
  }

  const quickActions: QuickAction[] = [
    {
      id: 'startAll',
      label: '启动全部',
      hint: `${stopped.length} 个已停止`,
      icon: Play,
      iconClass: 'text-emerald-400',
      onSelect: () => setBulkAction('startAll'),
      disabled: stopped.length === 0,
    },
    {
      id: 'stopAll',
      label: '停止全部',
      hint: '谨慎操作',
      icon: Square,
      iconClass: 'text-amber-400',
      onSelect: () => setBulkAction('stopAll'),
      disabled: running.length === 0,
    },
    {
      id: 'prune',
      label: '清理资源',
      hint: `${stopped.length} 个可回收`,
      icon: Trash2,
      iconClass: 'text-red-400',
      onSelect: () => setBulkAction('prune'),
      disabled: stopped.length === 0,
    },
    {
      id: 'updateAll',
      label: '一键更新',
      hint: `${running.length} 个可更新`,
      icon: RefreshCw,
      iconClass: 'text-brand',
      onSelect: () => setBulkAction('updateAll'),
      disabled: running.length === 0,
    },
  ]

  /* ---------------------------------------------------------- 渲染 */

  if (error && !dashboard) {
    return (
      <div className="space-y-6">
        <PageHeader title="Dashboard" description="实时概览" />
        <div className="elev-card p-5">
          <ErrorState
            title="无法加载面板数据"
            message={
              error instanceof Error ? error.message : '后端服务暂时不可用，请确认面板已启动后重试。'
            }
            onRetry={() => void refetch()}
          />
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Dashboard"
        description={
          <span className="flex flex-wrap items-center gap-1.5">
            实时概览 · 上次刷新
            <span className="text-fg-secondary">
              {live.lastEventAt ? formatRelativeTime(live.lastEventAt.toISOString()) : '刚刚'}
            </span>
            <span className="text-fg-subtle">·</span>
            <span className="text-fg-muted">
              数据每 <span className="font-mono text-fg-secondary">10s</span> 推送
            </span>
            <span
              className={cn(
                'ml-1 inline-flex items-center gap-1 text-[11px]',
                live.connected ? 'text-success' : 'text-fg-subtle',
              )}
            >
              <span className={cn('h-1.5 w-1.5 rounded-full', live.connected ? 'dot-running' : 'dot-stopped')} />
              {live.connected ? '实时已连接' : '实时未连接'}
            </span>
          </span>
        }
        actions={
          <>
            <Button variant="default" onClick={() => void refetch()}>
              <RefreshCw className="h-3.5 w-3.5" />
              刷新
            </Button>
            <Button variant="default" onClick={() => navigate('/images?action=pull')}>
              <Download className="h-3.5 w-3.5" />
              Pull 镜像
            </Button>
            <Button variant="primary" onClick={() => navigate('/containers?action=create')}>
              <Plus className="h-3.5 w-3.5" />
              新建容器
            </Button>
          </>
        }
      />

      {/* ===== 资源卡片 4 张 ===== */}
      <section className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {isLoading && !dashboard ? (
          <>
            <StatCardSkeleton />
            <StatCardSkeleton />
            <StatCardSkeleton />
            <StatCardSkeleton />
          </>
        ) : (
          <>
            <StatCard
              label="CPU"
              icon={Cpu}
              meta={`${dashboard?.resources.cpu.cores ?? dashboard?.host.cpuCores ?? '—'} cores`}
              value={cpuPct.toFixed(1)}
              meter={cpuPct}
              trend={cpuTrend}
            />
            <StatCard
              label="内存"
              icon={MemoryStick}
              meta={memTotal > 0 ? formatBytes(memTotal) : '—'}
              value={memPct.toFixed(1)}
              meter={memPct}
              trend={memTrend}
              hint={
                memTotal > 0
                  ? `${formatBytes(memUsed)} / ${formatBytes(memTotal)}`
                  : undefined
              }
            />
            <StatCard
              label="磁盘"
              icon={HardDrive}
              meta={dashboard?.resources.disk.mountpoint ?? '—'}
              value={diskPct.toFixed(1)}
              meter={diskPct}
              trend={
                diskPct > 85
                  ? { value: '注意', direction: 'up', tone: 'danger' }
                  : undefined
              }
              hint={
                dashboard
                  ? `${formatBytes(dashboard.resources.disk.usedBytes)} / ${formatBytes(
                      dashboard.resources.disk.totalBytes,
                    )}`
                  : undefined
              }
            />
            <NetworkStatCard
              label="网络 I/O"
              icon={Network}
              meta={dashboard?.host.arch ?? '—'}
              rxBytesPerSec={rxBytes}
              txBytesPerSec={txBytes}
            />
          </>
        )}
      </section>

      {/* ===== 实时图表 + 快捷操作 ===== */}
      <section className="grid grid-cols-1 gap-4 xl:grid-cols-3">
        <ResourceChart
          className="xl:col-span-2"
          data={chartData}
          range={range}
          onRangeChange={setRange}
          subtitle={
            range === '1h' || range === '24h'
              ? `最近 ${range} · 10 秒粒度`
              : `最近 ${range} · 实时推送`
          }
          lastLabel={
            live.lastEventAt ? formatRelativeTime(live.lastEventAt.toISOString()) : '—'
          }
        />

        <div className="elev-card p-5">
          <QuickActions actions={quickActions} />
          <SystemStatus items={buildSystemStatusItems(dashboard)} />
        </div>
      </section>

      {/* ===== 容器列表 + 事件流 ===== */}
      <section className="grid grid-cols-1 gap-4 xl:grid-cols-3">
        <div className="elev-card overflow-hidden xl:col-span-2">
          <div className="flex items-center justify-between p-5 pb-3">
            <div>
              <h2 className="text-base font-semibold">
                容器 · <span className="font-normal text-fg-muted">{containers.length} 个</span>
              </h2>
              <p className="mt-0.5 text-xs text-fg-muted">
                <span className="text-emerald-400">{running.length} 运行中</span> ·{' '}
                <span className="text-fg-muted">{stopped.length} 停止</span> ·{' '}
                <span className="text-red-400">{errored.length} 异常</span>
              </p>
            </div>
            <button
              type="button"
              onClick={() => navigate('/containers')}
              className="flex items-center gap-1 text-sm text-brand hover:underline"
            >
              查看全部
              <ArrowRight className="h-3.5 w-3.5" />
            </button>
          </div>

          <div className="flex flex-wrap items-center gap-2 px-5 pb-3">
            <FilterChip
              active={filter === 'all'}
              onClick={() => setFilter('all')}
            >
              全部 <span className="font-mono text-fg-subtle">{containers.length}</span>
            </FilterChip>
            <FilterChip active={filter === 'running'} onClick={() => setFilter('running')}>
              运行 <span className="font-mono text-fg-subtle">{running.length}</span>
            </FilterChip>
            <FilterChip active={filter === 'stopped'} onClick={() => setFilter('stopped')}>
              停止 <span className="font-mono text-fg-subtle">{stopped.length}</span>
            </FilterChip>
            <FilterChip active={filter === 'error'} onClick={() => setFilter('error')}>
              异常 <span className="font-mono text-fg-subtle text-red-400">{errored.length}</span>
            </FilterChip>
            <div className="relative ml-auto">
              <Search className="absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-fg-subtle" />
              <Input
                value={keyword}
                onChange={(e) => setKeyword(e.target.value)}
                placeholder="过滤容器名…"
                className="h-7 w-48 pl-8 text-xs"
              />
            </div>
          </div>

          <div className="grid grid-cols-1 gap-3 px-5 pb-5 md:grid-cols-2">
            {filteredContainers.length === 0 ? (
              <div className="col-span-full">
                <EmptyState
                  title={containers.length === 0 ? '还没有容器' : '没有匹配的容器'}
                  description={
                    containers.length === 0
                      ? '部署第一个服务后，它会出现在这里。'
                      : '换个关键字或筛选条件试试。'
                  }
                />
              </div>
            ) : (
              filteredContainers.slice(0, 8).map((container) => (
                <ContainerCard
                  key={container.id}
                  container={container}
                  live={live.containers[container.id]}
                />
              ))
            )}
          </div>
        </div>

        <div className="elev-card flex flex-col">
          <div className="flex items-center justify-between p-5 pb-3">
            <div>
              <h2 className="text-base font-semibold">事件流</h2>
              <p className="mt-0.5 text-xs text-fg-muted">最近 1 小时 · 实时</p>
            </div>
            <Button variant="ghost" size="icon-sm" title="更多">
              <MoreHorizontal className="h-4 w-4" />
            </Button>
          </div>
          {events.length === 0 ? (
            <EmptyState title="暂无事件" description="Docker 发生动作时会实时出现在这里。" />
          ) : (
            <EventStream events={events} />
          )}
        </div>
      </section>

      <footer className="flex items-center justify-between border-t border-border pb-2 pt-6 text-[11px] text-fg-subtle">
        <div className="flex items-center gap-3">
          <span className="flex items-center gap-1.5">
            <span className={cn('h-2 w-2 rounded-full', live.connected ? 'dot-running' : 'dot-stopped')} />
            {live.connected ? 'SSE 已连接' : 'SSE 未连接'}
          </span>
          <span>·</span>
          <span className="font-mono">api/v1</span>
          <span>·</span>
          <span>UI v0.1.0</span>
        </div>
        <div className="flex items-center gap-3">
          <button type="button" onClick={() => navigate('/containers')} className="hover:text-fg-secondary">
            容器
          </button>
          <button type="button" onClick={() => navigate('/compose')} className="hover:text-fg-secondary">
            Compose
          </button>
        </div>
      </footer>

      <ConfirmDialog
        open={bulkAction !== null}
        onOpenChange={(open) => {
          if (!open) setBulkAction(null)
        }}
        title={bulkAction ? BULK_META[bulkAction].title : ''}
        description={bulkAction ? BULK_META[bulkAction].description : ''}
        confirmText="确认执行"
        onConfirm={() => {
          if (bulkAction) void executeBulk(bulkAction)
        }}
      />
    </div>
  )
}

function StatCardSkeleton() {
  return (
    <div className="elev-card stat-card p-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Skeleton className="h-8 w-8 rounded-md" />
          <Skeleton className="h-4 w-12" />
        </div>
        <Skeleton className="h-3 w-16" />
      </div>
      <Skeleton className="mt-3 h-8 w-24" />
      <Skeleton className="mt-3 h-1.5 w-full" />
    </div>
  )
}

function FilterChip({
  active,
  onClick,
  children,
}: {
  active: boolean
  onClick: () => void
  children: React.ReactNode
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'inline-flex h-7 items-center gap-1 rounded-md border px-2.5 text-xs transition-colors',
        active
          ? 'border-brand/40 bg-brand-soft text-brand'
          : 'border-border bg-transparent text-fg-muted hover:border-border-strong hover:text-fg-primary',
      )}
    >
      {children}
    </button>
  )
}

// —— 快捷操作图标 ——
