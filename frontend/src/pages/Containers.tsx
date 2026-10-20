import { useMemo, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import {
  CircleAlert,
  CircleCheck,
  CircleStop,
  Layers,
  MoreHorizontal,
  Play,
  Plus,
  RefreshCw,
  RotateCw,
  Search,
  Square,
  Trash2,
  Grid3x3,
  List,
  Filter,
  Database,
  Tag,
} from 'lucide-react'
import { toast } from 'sonner'
import { PageHeader } from '@/components/layout/PageHeader'
import { ContainerIcon } from '@/components/common/ContainerIcon'
import { StateBadge } from '@/components/common/StatusBadge'
import { EmptyState, ErrorState } from '@/components/common/States'
import { ConfirmDialog } from '@/components/common/ConfirmDialog'
import { Button } from '@/components/ui/button'
import { Input, Skeleton } from '@/components/ui/input'
import { Checkbox } from '@/components/ui/controls'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useContainers, useContainerActions } from '@/hooks/useContainers'
import { useLiveStats } from '@/hooks/useLiveStats'
import { containerApi } from '@/lib/api'
import {
  formatBytes,
  formatDate,
  formatPorts,
  formatRelativeTime,
  isContainerAbnormal,
  isContainerRunning,
  isContainerStopped,
} from '@/lib/format'
import { cn } from '@/lib/utils'
import type { Container } from '@/lib/types'

type StatusFilter = 'all' | 'running' | 'stopped' | 'error'
type ViewMode = 'table' | 'card'

/** 容器列表页 */
export function Containers() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const { data, isLoading, error, refetch } = useContainers()
  const { action, update } = useContainerActions()
  const live = useLiveStats()

  const [status, setStatus] = useState<StatusFilter>('all')
  const [keyword, setKeyword] = useState('')
  const [view, setView] = useState<ViewMode>('table')
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [pendingDelete, setPendingDelete] = useState<Container[] | null>(null)

  const containers = useMemo(() => data?.items ?? [], [data])

  const running = containers.filter((c) => isContainerRunning(c.state))
  const stopped = containers.filter((c) => isContainerStopped(c.state, c.exitCode))
  const errored = containers.filter((c) => isContainerAbnormal(c.state, c.exitCode))

  const filtered = useMemo(() => {
    let list = containers
    if (status === 'running') list = list.filter((c) => isContainerRunning(c.state))
    if (status === 'stopped')
      list = list.filter((c) => isContainerStopped(c.state, c.exitCode))
    if (status === 'error') list = list.filter((c) => isContainerAbnormal(c.state, c.exitCode))
    const kw = keyword.trim().toLowerCase()
    if (kw) {
      list = list.filter(
        (c) =>
          c.name.toLowerCase().includes(kw) ||
          c.image.toLowerCase().includes(kw) ||
          (c.composeProject ?? '').toLowerCase().includes(kw),
      )
    }
    return list
  }, [containers, status, keyword])

  const totalCpu = running.reduce((sum, c) => sum + (live.containers[c.id]?.cpuPct ?? c.cpuPct ?? 0), 0)
  const totalMem = running.reduce(
    (sum, c) => sum + (live.containers[c.id]?.memUsageBytes ?? c.memUsageBytes ?? 0),
    0,
  )
  const totalMemPct =
    running.length > 0
      ? totalMem / running.reduce((sum, c) => sum + (c.memLimitBytes ?? 0), 0) * 100
      : 0

  const allSelected = filtered.length > 0 && filtered.every((c) => selected.has(c.id))

  const toggleAll = () => {
    if (allSelected) {
      setSelected(new Set())
    } else {
      setSelected(new Set(filtered.map((c) => c.id)))
    }
  }

  const toggleOne = (id: string) => {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const selectedContainers = containers.filter((c) => selected.has(c.id))

  const runBatch = async (kind: 'start' | 'stop' | 'restart' | 'update') => {
    const targets = selectedContainers
    if (targets.length === 0) return
    try {
      if (kind === 'update') {
        await Promise.allSettled(targets.map((c) => containerApi.update(c.id)))
      } else {
        await Promise.allSettled(targets.map((c) => containerApi.action(c.id, kind)))
      }
      toast.success(`已对 ${targets.length} 个容器执行操作`)
      setSelected(new Set())
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '批量操作失败')
    }
  }

  const confirmDelete = async () => {
    const targets = pendingDelete
    if (!targets) return
    try {
      await Promise.allSettled(targets.map((c) => containerApi.remove(c.id, true)))
      toast.success(`已删除 ${targets.length} 个容器`)
      setSelected(new Set())
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '删除失败')
    } finally {
      setPendingDelete(null)
    }
  }

  const handleCreate = () => {
    toast.info('当前版本通过 Compose 部署新容器')
    navigate('/compose')
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="容器"
        description={
          <span className="flex flex-wrap items-center gap-1.5">
            共 {containers.length} 个
            <span className="text-success">· {running.length} 运行</span>
            <span className="text-fg-subtle">· {stopped.length} 停止</span>
            <span className="text-danger">· {errored.length} 异常</span>
          </span>
        }
        actions={
          <>
            <Button variant="default" onClick={() => navigate('/images?action=pull')}>
              <Layers className="h-3.5 w-3.5" />
              Pull 镜像
            </Button>
            <Button variant="default" onClick={() => navigate('/compose')}>
              <Plus className="h-3.5 w-3.5" />
              从 Compose 部署
            </Button>
            <Button variant="primary" onClick={handleCreate}>
              <Plus className="h-3.5 w-3.5" />
              新建容器
            </Button>
          </>
        }
      />

      {/* ===== 统计条 ===== */}
      <section className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <SummaryCard
          icon={CircleCheck}
          iconClass="text-success"
          iconBg="bg-success-soft"
          label="运行中"
          value={running.length}
        />
        <SummaryCard
          icon={CircleStop}
          iconClass="text-fg-muted"
          iconBg="bg-bg-muted"
          label="已停止"
          value={stopped.length}
        />
        <SummaryCard
          icon={CircleAlert}
          iconClass="text-danger"
          iconBg="bg-danger-soft"
          label="异常"
          value={errored.length}
        />
        <div className="elev-card flex items-center gap-3 p-4">
          <div className="flex h-8 w-8 items-center justify-center rounded-md bg-brand-soft">
            <Layers className="h-4 w-4 text-brand" />
          </div>
          <div>
            <div className="text-[11px] text-fg-subtle">总 CPU/MEM</div>
            <div className="mt-0.5 font-mono text-base font-semibold">
              {totalCpu.toFixed(1)}% / {totalMemPct.toFixed(1)}%
            </div>
          </div>
        </div>
      </section>

      {/* ===== 筛选栏 ===== */}
      <section className="elev-card overflow-hidden">
        <div className="flex flex-wrap items-center gap-2 p-3">
          <div className="relative min-w-[220px] flex-1">
            <Search className="absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-fg-subtle" />
            <Input
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
              placeholder="按名称、镜像、标签过滤…"
              className="h-8 pl-8"
            />
          </div>
          <FilterChip active={status === 'all'} onClick={() => setStatus('all')}>
            全部 <span className="font-mono text-fg-subtle">{containers.length}</span>
          </FilterChip>
          <FilterChip active={status === 'running'} onClick={() => setStatus('running')}>
            运行 <span className="font-mono text-fg-subtle">{running.length}</span>
          </FilterChip>
          <FilterChip active={status === 'stopped'} onClick={() => setStatus('stopped')}>
            停止 <span className="font-mono text-fg-subtle">{stopped.length}</span>
          </FilterChip>
          <FilterChip active={status === 'error'} onClick={() => setStatus('error')}>
            异常 <span className="font-mono text-fg-subtle text-danger">{errored.length}</span>
          </FilterChip>
          <button
            type="button"
            className="inline-flex h-8 items-center gap-1.5 rounded-md border border-border px-2.5 text-xs text-fg-muted transition hover:border-border-strong hover:text-fg-primary"
          >
            <Filter className="h-3.5 w-3.5" />
            网络
          </button>
          <button
            type="button"
            className="inline-flex h-8 items-center gap-1.5 rounded-md border border-border px-2.5 text-xs text-fg-muted transition hover:border-border-strong hover:text-fg-primary"
          >
            <Database className="h-3.5 w-3.5" />
            卷
          </button>
          <button
            type="button"
            className="inline-flex h-8 items-center gap-1.5 rounded-md border border-border px-2.5 text-xs text-fg-muted transition hover:border-border-strong hover:text-fg-primary"
          >
            <Tag className="h-3.5 w-3.5" />
            标签
          </button>
          <div className="ml-auto flex items-center gap-1">
            <span className="mr-1 text-[11px] text-fg-subtle">视图</span>
            <Button
              variant={view === 'table' ? 'default' : 'ghost'}
              size="icon-sm"
              onClick={() => setView('table')}
              title="表格视图"
            >
              <List className="h-3.5 w-3.5" />
            </Button>
            <Button
              variant={view === 'card' ? 'default' : 'ghost'}
              size="icon-sm"
              onClick={() => setView('card')}
              title="卡片视图"
            >
              <Grid3x3 className="h-3.5 w-3.5" />
            </Button>
          </div>
        </div>

        {/* ===== 批量操作栏 ===== */}
        {selected.size > 0 ? (
          <div className="flex flex-wrap items-center gap-3 border-t border-border bg-brand-soft px-4 py-2.5">
            <span className="text-sm font-medium text-brand">已选 {selected.size} 项</span>
            <div className="flex flex-wrap items-center gap-2">
              <Button variant="ghost" size="sm" onClick={() => void runBatch('start')}>
                <Play className="h-3.5 w-3.5" />
                启动
              </Button>
              <Button variant="ghost" size="sm" onClick={() => void runBatch('stop')}>
                <Square className="h-3.5 w-3.5" />
                停止
              </Button>
              <Button variant="ghost" size="sm" onClick={() => void runBatch('restart')}>
                <RotateCw className="h-3.5 w-3.5" />
                重启
              </Button>
              <Button variant="ghost" size="sm" onClick={() => void runBatch('update')}>
                <RefreshCw className="h-3.5 w-3.5" />
                更新镜像
              </Button>
            </div>
            <div className="ml-auto flex items-center gap-2">
              <Button
                variant="danger"
                size="sm"
                onClick={() => setPendingDelete(selectedContainers)}
              >
                <Trash2 className="h-3.5 w-3.5" />
                删除
              </Button>
              <Button variant="ghost" size="sm" onClick={() => setSelected(new Set())}>
                取消
              </Button>
            </div>
          </div>
        ) : null}

        {/* ===== 列表 ===== */}
        {isLoading && containers.length === 0 ? (
          <ContainerTableSkeleton />
        ) : error && containers.length === 0 ? (
          <ErrorState
            title="无法加载容器列表"
            message={error instanceof Error ? error.message : undefined}
            onRetry={() => void refetch()}
          />
        ) : filtered.length === 0 ? (
          <EmptyState
            title={containers.length === 0 ? '还没有容器' : '没有匹配的容器'}
            description={
              containers.length === 0
                ? '通过 Compose 部署第一个服务，它会出现在这里。'
                : '换个关键字或筛选条件试试。'
            }
            action={
              containers.length === 0 ? (
                <Button variant="primary" size="sm" onClick={handleCreate}>
                  <Plus className="h-3.5 w-3.5" />
                  新建容器
                </Button>
              ) : null
            }
          />
        ) : view === 'table' ? (
          <div className="overflow-x-auto">
            <table className="w-full border-collapse">
              <thead className="text-[11px] uppercase tracking-wider text-fg-subtle">
                <tr className="border-y border-border">
                  <th className="w-8 px-4 py-2.5">
                    <Checkbox
                      checked={allSelected}
                      onCheckedChange={toggleAll}
                      aria-label="全选"
                    />
                  </th>
                  <th className="px-4 py-2.5 text-left font-medium">名称</th>
                  <th className="px-4 py-2.5 text-left font-medium">状态</th>
                  <th className="px-4 py-2.5 text-left font-medium">镜像</th>
                  <th className="px-4 py-2.5 text-left font-medium">端口</th>
                  <th className="px-4 py-2.5 text-left font-medium">网络</th>
                  <th className="w-40 px-4 py-2.5 text-left font-medium">CPU / MEM</th>
                  <th className="px-4 py-2.5 text-left font-medium">创建</th>
                  <th className="w-10 px-4 py-2.5" />
                </tr>
              </thead>
              <tbody>
                {filtered.map((container) => {
                  const liveStat = live.containers[container.id]
                  const cpu = liveStat?.cpuPct ?? container.cpuPct
                  const mem = liveStat?.memUsageBytes ?? container.memUsageBytes
                  const memLimit = liveStat?.memLimitBytes ?? container.memLimitBytes
                  const memPct = mem && memLimit ? (mem / memLimit) * 100 : null
                  return (
                    <tr
                      key={container.id}
                      className="row-hover border-b border-border/60 last:border-b-0"
                    >
                      <td className="px-4 py-2.5">
                        <Checkbox
                          checked={selected.has(container.id)}
                          onCheckedChange={() => toggleOne(container.id)}
                          aria-label={`选择 ${container.name}`}
                        />
                      </td>
                      <td className="px-4 py-2.5">
                        <button
                          type="button"
                          onClick={() => navigate(`/containers/${container.id}`)}
                          className="flex items-center gap-2.5 text-left hover:underline"
                        >
                          <ContainerIcon
                            image={container.image}
                            name={container.name}
                            className="h-7 w-7 rounded"
                            iconClassName="h-3.5 w-3.5"
                          />
                          <span className="font-medium">{container.name}</span>
                        </button>
                      </td>
                      <td className="px-4 py-2.5">
                        <StateBadge state={container.state} exitCode={container.exitCode} />
                      </td>
                      <td className="px-4 py-2.5 font-mono text-xs text-fg-secondary">
                        {container.image}
                      </td>
                      <td className="px-4 py-2.5 font-mono text-xs text-fg-secondary">
                        {formatPorts(container.ports) || '—'}
                      </td>
                      <td className="px-4 py-2.5">
                        <div className="flex flex-wrap gap-1">
                          {container.networks.length === 0 ? (
                            <span className="text-xs text-fg-subtle">—</span>
                          ) : (
                            container.networks.map((net) => (
                              <span
                                key={net.name}
                                className="rounded-full border border-brand/30 bg-brand-soft px-2 py-0.5 text-[11px] text-brand"
                              >
                                {net.name}
                              </span>
                            ))
                          )}
                        </div>
                      </td>
                      <td className="px-4 py-2.5">
                        {cpu === null || cpu === undefined ? (
                          <span className="font-mono text-xs text-fg-subtle">—</span>
                        ) : (
                          <div className="space-y-1">
                            <div className="flex items-center gap-2">
                              <div className="meter h-1 w-16">
                                <span style={{ width: `${Math.min(cpu, 100)}%` }} />
                              </div>
                              <span className="font-mono text-xs text-fg-secondary">
                                {cpu.toFixed(1)}%
                              </span>
                            </div>
                            <div className="flex items-center gap-2">
                              <div className="meter h-1 w-16">
                                <span
                                  className={cn(
                                    memPct !== null && memPct > 85 ? 'danger' : memPct !== null && memPct > 60 ? 'warn' : '',
                                  )}
                                  style={{ width: `${Math.min(memPct ?? 0, 100)}%` }}
                                />
                              </div>
                              <span className="font-mono text-xs text-fg-secondary">
                                {mem ? formatBytes(mem) : '—'}
                              </span>
                            </div>
                          </div>
                        )}
                      </td>
                      <td className="px-4 py-2.5 text-xs text-fg-muted">
                        {formatRelativeTime(container.createdAt)}
                      </td>
                      <td className="px-4 py-2.5">
                        <DropdownMenu>
                          <DropdownMenuTrigger asChild>
                            <Button variant="ghost" size="icon-sm">
                              <MoreHorizontal className="h-4 w-4" />
                            </Button>
                          </DropdownMenuTrigger>
                          <DropdownMenuContent align="end">
                            <DropdownMenuItem
                              onSelect={() => navigate(`/containers/${container.id}`)}
                            >
                              查看详情
                            </DropdownMenuItem>
                            <DropdownMenuItem
                              onSelect={() => action.mutate({ id: container.id, action: 'start' })}
                            >
                              启动
                            </DropdownMenuItem>
                            <DropdownMenuItem
                              onSelect={() => action.mutate({ id: container.id, action: 'stop' })}
                            >
                              停止
                            </DropdownMenuItem>
                            <DropdownMenuItem
                              onSelect={() => action.mutate({ id: container.id, action: 'restart' })}
                            >
                              重启
                            </DropdownMenuItem>
                            <DropdownMenuItem
                              onSelect={() => update.mutate(container.id)}
                            >
                              更新镜像
                            </DropdownMenuItem>
                            <DropdownMenuItem
                              className="text-danger"
                              onSelect={() => setPendingDelete([container])}
                            >
                              删除
                            </DropdownMenuItem>
                          </DropdownMenuContent>
                        </DropdownMenu>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="grid grid-cols-1 gap-3 p-4 md:grid-cols-2 xl:grid-cols-3">
            {filtered.map((container) => (
              <button
                key={container.id}
                type="button"
                onClick={() => navigate(`/containers/${container.id}`)}
                className="surface-card surface-card-hover p-3.5 text-left"
              >
                <div className="flex items-start gap-3">
                  <ContainerIcon image={container.image} name={container.name} />
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <span className="truncate font-medium">{container.name}</span>
                      <StateBadge state={container.state} exitCode={container.exitCode} />
                    </div>
                    <div className="mt-0.5 truncate font-mono text-[11px] text-fg-subtle">
                      {container.image}
                    </div>
                    <div className="mt-2 grid grid-cols-3 gap-2 text-[11px]">
                      <div>
                        <span className="text-fg-subtle">CPU</span>{' '}
                        <span className="font-mono text-fg-secondary">
                          {container.cpuPct === null ? '—' : `${container.cpuPct.toFixed(1)}%`}
                        </span>
                      </div>
                      <div>
                        <span className="text-fg-subtle">MEM</span>{' '}
                        <span className="font-mono text-fg-secondary">
                          {container.memUsageBytes === null
                            ? '—'
                            : formatBytes(container.memUsageBytes)}
                        </span>
                      </div>
                      <div>
                        <span className="text-fg-subtle">创建</span>{' '}
                        <span className="font-mono text-fg-secondary">
                          {formatDate(container.createdAt)}
                        </span>
                      </div>
                    </div>
                  </div>
                </div>
              </button>
            ))}
          </div>
        )}
      </section>

      {searchParams.get('action') === 'create' ? null : null}

      <ConfirmDialog
        open={pendingDelete !== null}
        onOpenChange={(open) => {
          if (!open) setPendingDelete(null)
        }}
        title={`删除 ${pendingDelete?.length ?? 0} 个容器？`}
        description="容器将被强制停止并删除，其数据卷会保留。此操作不可撤销。"
        onConfirm={() => void confirmDelete()}
      />
    </div>
  )
}

function SummaryCard({
  icon: Icon,
  iconClass,
  iconBg,
  label,
  value,
}: {
  icon: typeof CircleCheck
  iconClass: string
  iconBg: string
  label: string
  value: number
}) {
  return (
    <div className="elev-card flex items-center gap-3 p-4">
      <div className={cn('flex h-8 w-8 items-center justify-center rounded-md', iconBg)}>
        <Icon className={cn('h-4 w-4', iconClass)} />
      </div>
      <div>
        <div className="text-[11px] text-fg-subtle">{label}</div>
        <div className="mt-0.5 font-mono text-lg font-semibold">{value}</div>
      </div>
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
        'inline-flex h-8 items-center gap-1 rounded-md border px-2.5 text-xs transition-colors',
        active
          ? 'border-brand bg-brand-soft text-brand'
          : 'border-border bg-transparent text-fg-muted hover:border-border-strong hover:text-fg-primary',
      )}
    >
      {children}
    </button>
  )
}

function ContainerTableSkeleton() {
  return (
    <div className="space-y-2 p-4">
      {Array.from({ length: 6 }).map((_, index) => (
        <div key={index} className="flex items-center gap-4">
          <Skeleton className="h-4 w-4" />
          <Skeleton className="h-7 w-40" />
          <Skeleton className="h-5 w-16" />
          <Skeleton className="h-4 w-32" />
          <Skeleton className="h-4 w-20" />
          <Skeleton className="ml-auto h-4 w-16" />
        </div>
      ))}
    </div>
  )
}
