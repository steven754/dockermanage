import { useMemo, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import {
  BarChart3,
  Cable,
  ChevronRight,
  FileCode2,
  FolderTree,
  HardDrive,
  List,
  Network,
  RefreshCw,
  RotateCw,
  Square,
  TerminalSquare,
  Trash2,
} from 'lucide-react'
import { toast } from 'sonner'
import { PageHeader } from '@/components/layout/PageHeader'
import { ContainerIcon } from '@/components/common/ContainerIcon'
import { StateBadge, HealthBadge } from '@/components/common/StatusBadge'
import { ErrorState } from '@/components/common/States'
import { ConfirmDialog } from '@/components/common/ConfirmDialog'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/input'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { StatsTab } from '@/components/container/StatsTab'
import { LogsTab } from '@/components/container/LogsTab'
import { TerminalTab } from '@/components/container/TerminalTab'
import { InspectTab } from '@/components/container/InspectTab'
import { FilesTab } from '@/components/container/FilesTab'
import { EnvTab } from '@/components/container/EnvTab'
import { useContainer, useContainerActions } from '@/hooks/useContainers'
import { useLiveStats } from '@/hooks/useLiveStats'
import { formatBytes, formatBytesPerSec, formatRelativeTime, shortId } from '@/lib/format'
import { cn } from '@/lib/utils'

/** 容器详情页 */
export function ContainerDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const containerId = id ?? ''
  const { data: container, isLoading, error, refetch } = useContainer(containerId)
  const { action, remove, update } = useContainerActions()
  const live = useLiveStats()

  const [tab, setTab] = useState('stats')
  const [confirmDelete, setConfirmDelete] = useState(false)

  const liveStat = live.containers[containerId]
  const cpu = liveStat?.cpuPct ?? container?.cpuPct ?? null
  const mem = liveStat?.memUsageBytes ?? container?.memUsageBytes ?? null
  const memLimit = liveStat?.memLimitBytes ?? container?.memLimitBytes ?? null
  const rx = liveStat?.netRxBytesPerSec ?? container?.netRxBytesPerSec ?? null
  const tx = liveStat?.netTxBytesPerSec ?? container?.netTxBytesPerSec ?? null

  const memPct = useMemo(() => {
    if (mem === null || memLimit === null || !memLimit) return null
    return (mem / memLimit) * 100
  }, [mem, memLimit])

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-5 w-40" />
        <div className="elev-card p-5">
          <div className="flex items-center gap-3">
            <Skeleton className="h-11 w-11 rounded-lg" />
            <Skeleton className="h-6 w-48" />
          </div>
        </div>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
          {Array.from({ length: 4 }).map((_, index) => (
            <Skeleton key={index} className="h-24 w-full rounded-xl" />
          ))}
        </div>
      </div>
    )
  }

  if (error || !container) {
    return (
      <div className="space-y-6">
        <PageHeader title="容器详情" />
        <div className="elev-card p-5">
          <ErrorState
            title="无法加载容器详情"
            message={error instanceof Error ? error.message : '容器不存在或已被删除。'}
            onRetry={() => void refetch()}
          />
        </div>
      </div>
    )
  }

  const isRunning = container.state === 'running'

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-1.5 text-[13px] text-fg-subtle">
        <Link to="/containers" className="hover:text-fg-secondary">
          容器
        </Link>
        <ChevronRight className="h-3.5 w-3.5" />
        <span className="text-fg-secondary">{container.name}</span>
      </div>

      {/* ===== 概览 ===== */}
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="flex min-w-0 items-center gap-3">
          <ContainerIcon
            image={container.image}
            name={container.name}
            className="h-11 w-11 rounded-lg"
            iconClassName="h-5 w-5"
          />
          <div className="min-w-0">
            <div className="flex flex-wrap items-center gap-2">
              <h1 className="truncate text-2xl font-semibold tracking-tight">{container.name}</h1>
              <StateBadge state={container.state} exitCode={container.exitCode} />
              <HealthBadge health={container.health} />
            </div>
            <p className="mt-1 flex flex-wrap items-center gap-2 font-mono text-[11px] text-fg-subtle">
              <span>{container.image}</span>
              <span>·</span>
              <span>ID {shortId(container.id)}</span>
              <span>·</span>
              <span>创建于 {formatRelativeTime(container.createdAt)}</span>
              {container.composeProject ? (
                <>
                  <span>·</span>
                  <span className="text-brand">{container.composeProject}</span>
                </>
              ) : null}
            </p>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <Button
            variant="default"
            onClick={() => action.mutate({ id: container.id, action: 'restart' })}
          >
            <RotateCw className="h-3.5 w-3.5" />
            重启
          </Button>
          <Button
            variant="default"
            onClick={() =>
              action.mutate({
                id: container.id,
                action: isRunning ? 'stop' : 'start',
              })
            }
          >
            {isRunning ? (
              <>
                <Square className="h-3.5 w-3.5" />
                停止
              </>
            ) : (
              <>
                <RefreshCw className="h-3.5 w-3.5" />
                启动
              </>
            )}
          </Button>
          <Button variant="default" onClick={() => update.mutate(container.id)}>
            <RefreshCw className="h-3.5 w-3.5" />
            更新镜像
          </Button>
          <Button variant="danger" onClick={() => setConfirmDelete(true)}>
            <Trash2 className="h-3.5 w-3.5" />
            删除
          </Button>
          <Button variant="primary" onClick={() => setTab('terminal')}>
            <TerminalSquare className="h-3.5 w-3.5" />
            进入终端
          </Button>
        </div>
      </div>

      {/* ===== 指标卡 ===== */}
      <section className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <MetricCard
          label="CPU"
          value={cpu === null ? '—' : `${cpu.toFixed(1)}%`}
          hint={container.status}
          meter={cpu === null ? undefined : cpu}
        />
        <MetricCard
          label="内存"
          value={mem === null ? '—' : formatBytes(mem)}
          hint={memLimit ? `上限 ${formatBytes(memLimit)}` : undefined}
          meter={memPct ?? undefined}
        />
        <MetricCard
          label="网络 I/O"
          value={rx === null ? '—' : formatBytesPerSec(rx)}
          hint={tx === null ? undefined : `TX ${formatBytesPerSec(tx)}`}
        />
        <MetricCard
          label="磁盘 I/O"
          value={liveStat ? formatBytesPerSec(liveStat.blockReadBytesPerSec) : '—'}
          hint={liveStat ? `写 ${formatBytesPerSec(liveStat.blockWriteBytesPerSec)}` : undefined}
        />
      </section>

      {/* ===== Tabs ===== */}
      <Tabs value={tab} onValueChange={setTab}>
        <div className="flex items-center justify-between border-b border-border">
          <TabsList className="border-b-0">
            <TabsTrigger value="stats">
              <BarChart3 className="mr-1 inline h-3.5 w-3.5 -mt-0.5" />
              Stats
            </TabsTrigger>
            <TabsTrigger value="logs">
              <List className="mr-1 inline h-3.5 w-3.5 -mt-0.5" />
              Logs
            </TabsTrigger>
            <TabsTrigger value="terminal">
              <TerminalSquare className="mr-1 inline h-3.5 w-3.5 -mt-0.5" />
              Terminal
            </TabsTrigger>
            <TabsTrigger value="inspect">
              <FileCode2 className="mr-1 inline h-3.5 w-3.5 -mt-0.5" />
              Inspect
            </TabsTrigger>
            <TabsTrigger value="files">
              <FolderTree className="mr-1 inline h-3.5 w-3.5 -mt-0.5" />
              Files
            </TabsTrigger>
            <TabsTrigger value="env">
              <List className="mr-1 inline h-3.5 w-3.5 -mt-0.5" />
              环境变量
            </TabsTrigger>
          </TabsList>
          <span className="hidden items-center gap-1.5 pb-2.5 text-[11px] text-fg-subtle sm:flex">
            <RefreshCw className="h-3 w-3" />
            每 10s 刷新
          </span>
        </div>

        <TabsContent value="stats">
          <StatsTab containerId={container.id} />
          <div className="mt-4 grid grid-cols-1 gap-4 xl:grid-cols-3">
            <InfoCard title="网络" icon={Network}>
              {container.networks.length === 0 ? (
                <p className="text-[13px] text-fg-subtle">未接入任何网络</p>
              ) : (
                <ul className="space-y-2">
                  {container.networks.map((net) => (
                    <li key={net.name} className="flex items-center justify-between text-[13px]">
                      <span className="text-fg-secondary">{net.name}</span>
                      <span className="font-mono text-xs text-fg-subtle">{net.ip}</span>
                    </li>
                  ))}
                </ul>
              )}
            </InfoCard>
            <InfoCard title="挂载" icon={HardDrive}>
              {container.mounts.length === 0 ? (
                <p className="text-[13px] text-fg-subtle">没有挂载</p>
              ) : (
                <ul className="space-y-2">
                  {container.mounts.map((mount) => (
                    <li key={`${mount.type}-${mount.destination}`} className="text-[13px]">
                      <div className="flex items-center gap-2">
                        <span className="rounded-full border border-border bg-bg-muted px-2 py-0.5 text-[11px] text-fg-muted">
                          {mount.type}
                        </span>
                        <span className="font-mono text-xs text-brand">{mount.name}</span>
                      </div>
                      <div className="mt-0.5 truncate font-mono text-[11px] text-fg-subtle">
                        {mount.destination}
                      </div>
                    </li>
                  ))}
                </ul>
              )}
            </InfoCard>
            <InfoCard title="端口映射" icon={Cable}>
              {container.ports.length === 0 ? (
                <p className="text-[13px] text-fg-subtle">没有端口映射</p>
              ) : (
                <ul className="space-y-2">
                  {container.ports.map((port) => (
                    <li key={`${port.privatePort}-${port.publicPort}`} className="text-[13px]">
                      <div className="font-mono">
                        :{port.publicPort || port.privatePort}
                        {port.publicPort ? ` → :${port.privatePort}` : ''}
                      </div>
                      <div className="mt-0.5 flex items-center gap-1.5">
                        <span className="rounded-full border border-info/30 bg-info-soft px-2 py-0.5 text-[11px] text-info">
                          {port.type}
                        </span>
                        <span className="font-mono text-[11px] text-fg-subtle">{port.ip}</span>
                      </div>
                    </li>
                  ))}
                </ul>
              )}
            </InfoCard>
          </div>
        </TabsContent>
        <TabsContent value="logs">
          <LogsTab containerId={container.id} />
        </TabsContent>
        <TabsContent value="terminal">
          <TerminalTab containerId={container.id} />
        </TabsContent>
        <TabsContent value="inspect">
          <InspectTab containerId={container.id} />
        </TabsContent>
        <TabsContent value="files">
          <FilesTab containerId={container.id} />
        </TabsContent>
        <TabsContent value="env">
          <EnvTab containerId={container.id} />
        </TabsContent>
      </Tabs>

      <ConfirmDialog
        open={confirmDelete}
        onOpenChange={setConfirmDelete}
        title={`删除容器 ${container.name}？`}
        description="容器将被停止并删除，挂载的数据卷会保留。此操作不可撤销。"
        onConfirm={() => {
          remove.mutate(
            { id: container.id, force: true },
            {
              onSuccess: () => {
                toast.success('容器已删除')
                navigate('/containers')
              },
              onError: (err) => toast.error(err instanceof Error ? err.message : '删除失败'),
            },
          )
        }}
      />
    </div>
  )
}

function MetricCard({
  label,
  value,
  hint,
  meter,
}: {
  label: string
  value: string
  hint?: string
  meter?: number
}) {
  return (
    <div className="elev-card p-4">
      <div className="text-[11px] text-fg-subtle">{label}</div>
      <div className="mt-1 flex items-end gap-1.5">
        <span className="font-mono text-xl font-semibold tracking-tight">{value}</span>
      </div>
      {meter !== undefined ? (
        <div className="meter mt-2.5">
          <span
            className={cn(
              meter > 85 ? 'danger' : meter > 60 ? 'warn' : '',
            )}
            style={{ width: `${Math.min(Math.max(meter, 0), 100)}%` }}
          />
        </div>
      ) : null}
      {hint ? <div className="mt-1.5 text-[11px] text-fg-subtle">{hint}</div> : null}
    </div>
  )
}

function InfoCard({
  title,
  icon: Icon,
  children,
}: {
  title: string
  icon: typeof Network
  children: React.ReactNode
}) {
  return (
    <div className="elev-card p-5">
      <div className="mb-3 flex items-center gap-2">
        <Icon className="h-4 w-4 text-fg-subtle" />
        <h3 className="text-sm font-semibold">{title}</h3>
      </div>
      {children}
    </div>
  )
}
