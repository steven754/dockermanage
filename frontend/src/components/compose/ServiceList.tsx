import { Boxes, Play, Square } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { StateBadge } from '@/components/common/StatusBadge'
import { EmptyState } from '@/components/common/States'
import { cn } from '@/lib/utils'
import type { ComposeService } from '@/lib/types'
import type { ComposeServiceShape } from './YamlEditor'

/** 服务列表（来自 compose 文件或后端 ps 状态） */
export function ServiceList({
  services,
  runtime,
  selected,
  onToggle,
  onUp,
  onDown,
  className,
}: {
  /** compose 文件里解析出的服务定义 */
  services: ComposeServiceShape[]
  /** 后端返回的运行时状态 */
  runtime?: ComposeService[]
  /** 当前勾选的服务名 */
  selected?: string[]
  onToggle?: (name: string) => void
  onUp?: (name: string) => void
  onDown?: (name: string) => void
  className?: string
}) {
  const runtimeMap = new Map((runtime ?? []).map((item) => [item.name, item]))

  if (services.length === 0) {
    return (
      <div className={className}>
        <EmptyState
          icon={Boxes}
          title="没有服务定义"
          description="在左侧编辑器中编写 services 后，服务会显示在这里。"
        />
      </div>
    )
  }

  return (
    <ul className={cn('divide-y divide-border/60', className)}>
      {services.map((service) => {
        const state = runtimeMap.get(service.name ?? '')
        return (
          <li key={service.name} className="row-hover px-4 py-3">
            <div className="flex items-center gap-2">
              {onToggle ? (
                <button
                  type="button"
                  onClick={() => onToggle(service.name ?? '')}
                  className={cn(
                    'flex h-5 w-5 items-center justify-center rounded border text-[10px] transition',
                    selected?.includes(service.name ?? '')
                      ? 'border-brand bg-brand text-white'
                      : 'border-border-strong bg-bg-base text-transparent',
                  )}
                  aria-label={`选择 ${service.name}`}
                >
                  ✓
                </button>
              ) : null}
              <span className="font-medium">{service.name}</span>
              {state ? (
                <StateBadge state={state.state} />
              ) : (
                <span className="rounded-full border border-border bg-bg-muted px-2 py-0.5 text-[11px] text-fg-subtle">
                  未部署
                </span>
              )}
            </div>

            <div className="mt-1 truncate font-mono text-[11px] text-fg-subtle">
              {service.image ?? (service.build ? 'build' : '—')}
            </div>

            {state?.ports && state.ports.length > 0 ? (
              <div className="mt-1 flex flex-wrap gap-1">
                {state.ports.map((port) => (
                  <span
                    key={`${port.privatePort}-${port.publicPort}`}
                    className="rounded-full border border-info/30 bg-info-soft px-2 py-0.5 font-mono text-[10px] text-info"
                  >
                    {port.publicPort ? `${port.publicPort}→${port.privatePort}` : port.privatePort}
                  </span>
                ))}
              </div>
            ) : null}

            {onUp || onDown ? (
              <div className="mt-2 flex items-center gap-2">
                {onUp ? (
                  <Button variant="outline" size="sm" onClick={() => onUp(service.name ?? '')}>
                    <Play className="h-3 w-3" />
                    启动
                  </Button>
                ) : null}
                {onDown ? (
                  <Button variant="outline" size="sm" onClick={() => onDown(service.name ?? '')}>
                    <Square className="h-3 w-3" />
                    停止
                  </Button>
                ) : null}
                {state?.status ? (
                  <span className="font-mono text-[11px] text-fg-subtle">{state.status}</span>
                ) : null}
              </div>
            ) : null}
          </li>
        )
      })}
    </ul>
  )
}
