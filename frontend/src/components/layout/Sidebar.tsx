import { NavLink } from 'react-router-dom'
import {
  Activity,
  BellRing,
  Container,
  Database,
  FileCode2,
  FolderTree,
  Layers,
  LayoutDashboard,
  LayoutTemplate,
  Network,
  RefreshCcw,
  Server,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { useUiStore } from '@/store/ui'
import { useDashboard } from '@/hooks/useContainers'
import { useEventStream } from '@/hooks/useEventStream'
import { formatDuration } from '@/lib/format'

interface NavEntry {
  to: string
  label: string
  icon: typeof Activity
  /** 右侧计数 */
  count?: number
  /** 未来版本标记（v1.1 预留功能） */
  future?: boolean
  end?: boolean
}

/**
 * 侧栏导航。结构与设计稿 1:1：
 * 概览 / 资源 / 工具 三组 + 底部主机信息卡。
 */
export function Sidebar() {
  const { data: dashboard } = useDashboard()
  const { events } = useEventStream(50)
  const sidebarOpen = useUiStore((s) => s.sidebarOpen)
  const setSidebarOpen = useUiStore((s) => s.setSidebarOpen)

  const containerCount = dashboard?.containers.total
  const imageCount = dashboard?.images.total
  const volumeCount = dashboard?.volumes.total
  const networkCount = dashboard?.networks.total
  const eventCount = events.length

  const overview: NavEntry[] = [
    { to: '/', label: 'Dashboard', icon: LayoutDashboard, end: true },
    { to: '/events', label: '事件流', icon: Activity, count: eventCount },
  ]

  const resources: NavEntry[] = [
    { to: '/containers', label: '容器', icon: Container, count: containerCount },
    { to: '/images', label: '镜像', icon: Layers, count: imageCount },
    { to: '/volumes', label: '卷', icon: Database, count: volumeCount },
    { to: '/networks', label: '网络', icon: Network, count: networkCount },
    { to: '/compose', label: 'Compose', icon: FileCode2 },
  ]

  const tools: NavEntry[] = [
    { to: '/containers?view=tools', label: '文件浏览器', icon: FolderTree, future: true },
    { to: '/containers?view=update', label: '一键更新', icon: RefreshCcw },
    { to: '/settings?tab=alerts', label: '告警', icon: BellRing, future: true },
    { to: '/settings?tab=templates', label: '应用模板', icon: LayoutTemplate, future: true },
  ]

  const renderGroup = (title: string, entries: NavEntry[]) => (
    <div className="flex flex-col gap-1">
      <div className="px-3 pb-1 pt-4 text-[11px] uppercase tracking-wider text-fg-subtle first:pt-2">
        {title}
      </div>
      {entries.map((entry) => {
        const Icon = entry.icon
        if (entry.future) {
          return (
            <div
              key={entry.label}
              className="nav-item cursor-not-allowed opacity-50"
              title="v1.1 规划中"
            >
              <Icon className="nav-icon" />
              {entry.label}
              <span className="ml-auto rounded-full border border-border px-1.5 text-[10px] text-fg-subtle">
                v1.1
              </span>
            </div>
          )
        }
        return (
          <NavLink
            key={entry.to}
            to={entry.to}
            end={entry.end}
            onClick={() => setSidebarOpen(false)}
            className={({ isActive }) => cn('nav-item', isActive && 'active')}
          >
            <Icon className="nav-icon" />
            {entry.label}
            {entry.count !== undefined ? (
              <span className="ml-auto text-xs text-fg-subtle">{entry.count}</span>
            ) : null}
          </NavLink>
        )
      })}
    </div>
  )

  return (
    <aside
      className={cn(
        'sidebar-fixed flex w-60 shrink-0 flex-col gap-1 border-r border-border bg-bg-elev p-3',
        'h-[calc(100vh-56px)] sticky top-14',
        sidebarOpen && 'open',
      )}
    >
      {renderGroup('概览', overview)}
      {renderGroup('资源', resources)}
      {renderGroup('工具', tools)}

      <div className="mt-auto rounded-lg border border-border bg-bg-card/60 p-3">
        <div className="mb-1.5 flex items-center gap-2">
          <Server className="h-3.5 w-3.5 text-fg-muted" />
          <span className="text-xs font-medium">{dashboard?.host.name ?? '—'}</span>
        </div>
        <div className="text-[11px] leading-relaxed text-fg-subtle">
          <div>
            Docker {dashboard?.host.dockerVersion ?? '—'} · {dashboard?.host.os ?? '—'}
          </div>
          <div>
            Uptime{' '}
            <span className="font-mono text-fg-secondary">
              {formatDuration(dashboard?.host.uptimeSeconds)}
            </span>
          </div>
        </div>
        <div className="mt-2 flex items-center gap-1.5">
          <span className={cn('h-2 w-2 rounded-full', dashboard ? 'dot-running' : 'dot-error')} />
          <span className="text-[11px] text-fg-muted">
            {dashboard ? '引擎在线' : '引擎离线'}
          </span>
        </div>
      </div>
    </aside>
  )
}
