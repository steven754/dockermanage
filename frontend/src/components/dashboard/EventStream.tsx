import {
  AlertTriangle,
  Box,
  CheckCircle2,
  Download,
  GitPullRequestArrow,
  Play,
  Settings,
  Square,
  XOctagon,
  type LucideIcon,
} from 'lucide-react'
import { formatRelativeTime } from '@/lib/format'
import type { DockerEvent } from '@/lib/types'
import { cn } from '@/lib/utils'

interface EventVisual {
  icon: LucideIcon
  bg: string
  fg: string
  /** 事件主体文字颜色 */
  nameCls: string
  /** 事件描述 */
  describe: (event: DockerEvent) => string
  /** 次级补充信息 */
  detail?: (event: DockerEvent) => string | null
}

const VISUALS: Record<string, EventVisual> = {
  'container.die': {
    icon: XOctagon,
    bg: 'bg-red-500/15',
    fg: 'text-red-400',
    nameCls: 'text-red-400',
    describe: () => 'exited',
    detail: (event) =>
      event.exitCode ? `code=${event.exitCode} · 容器异常退出` : '容器异常退出',
  },
  'container.start': {
    icon: Play,
    bg: 'bg-emerald-500/15',
    fg: 'text-emerald-400',
    nameCls: '',
    describe: () => 'started',
  },
  'container.stop': {
    icon: Square,
    bg: 'bg-slate-500/15',
    fg: 'text-slate-400',
    nameCls: '',
    describe: () => 'stopped',
  },
  'container.kill': {
    icon: XOctagon,
    bg: 'bg-red-500/15',
    fg: 'text-red-400',
    nameCls: 'text-red-400',
    describe: () => 'killed',
  },
  'container.restart': {
    icon: Play,
    bg: 'bg-emerald-500/15',
    fg: 'text-emerald-400',
    nameCls: '',
    describe: () => 'restarted',
  },
  'container.create': {
    icon: Box,
    bg: 'bg-blue-500/15',
    fg: 'text-blue-400',
    nameCls: '',
    describe: () => 'created',
  },
  'container.destroy': {
    icon: Square,
    bg: 'bg-slate-500/15',
    fg: 'text-slate-400',
    nameCls: '',
    describe: () => 'removed',
  },
  'container.health_status: healthy': {
    icon: CheckCircle2,
    bg: 'bg-emerald-500/15',
    fg: 'text-emerald-400',
    nameCls: '',
    describe: () => '健康检查通过',
    detail: () => 'healthy',
  },
  'image.pull': {
    icon: GitPullRequestArrow,
    bg: 'bg-blue-500/15',
    fg: 'text-blue-400',
    nameCls: '',
    describe: () => 'pulled',
    detail: (event) => {
      const image = event.details?.image
      return typeof image === 'string' ? image : null
    },
  },
  'image.remove': {
    icon: Download,
    bg: 'bg-slate-500/15',
    fg: 'text-slate-400',
    nameCls: '',
    describe: () => 'removed',
  },
  'daemon.reload': {
    icon: Settings,
    bg: 'bg-brand-soft',
    fg: 'text-brand',
    nameCls: '',
    describe: () => '配置重载',
  },
}

const DEFAULT_VISUAL: EventVisual = {
  icon: AlertTriangle,
  bg: 'bg-amber-500/15',
  fg: 'text-amber-400',
  nameCls: '',
  describe: (event) => event.action,
}

function visualFor(event: DockerEvent): EventVisual {
  return VISUALS[`${event.type}.${event.action}`] ?? DEFAULT_VISUAL
}

/** 事件流时间线 */
export function EventStream({
  events,
  className,
  maxHeightClass = 'max-h-[640px]',
}: {
  events: DockerEvent[]
  className?: string
  maxHeightClass?: string
}) {
  return (
    <div className={cn('space-y-3 overflow-auto px-5 pb-5', maxHeightClass, className)}>
      {events.map((event) => {
        const visual = visualFor(event)
        const Icon = visual.icon
        const detail = visual.detail?.(event) ?? null
        return (
          <div key={`${event.id}-${event.ts}`} className="group flex gap-3">
            <div className="flex flex-col items-center">
              <div
                className={cn(
                  'flex h-7 w-7 shrink-0 items-center justify-center rounded-md',
                  visual.bg,
                )}
              >
                <Icon className={cn('h-3.5 w-3.5', visual.fg)} />
              </div>
              <div className="mt-1.5 w-px flex-1 bg-border" />
            </div>
            <div className="flex-1 pb-3">
              <div className="text-sm">
                <span className={cn('font-medium', visual.nameCls)}>
                  {event.resourceName || event.actorName || '—'}
                </span>{' '}
                <span className="text-fg-muted">{visual.describe(event)}</span>
                {detail ? (
                  <span className="font-mono text-xs text-fg-secondary"> {detail}</span>
                ) : null}
              </div>
              <div className="mt-1 text-[11px] text-fg-subtle">
                {formatRelativeTime(event.ts)}
              </div>
            </div>
          </div>
        )
      })}
    </div>
  )
}
