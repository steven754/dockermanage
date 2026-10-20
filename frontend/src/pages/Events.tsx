import { useMemo, useState } from 'react'
import { Activity, RefreshCw } from 'lucide-react'
import { PageHeader } from '@/components/layout/PageHeader'
import { EventStream } from '@/components/dashboard/EventStream'
import { EmptyState, ErrorState } from '@/components/common/States'
import { Button } from '@/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { useEventStream } from '@/hooks/useEventStream'
import { cn } from '@/lib/utils'

const TYPE_OPTIONS = [
  { value: 'all', label: '全部类型' },
  { value: 'container', label: '容器' },
  { value: 'image', label: '镜像' },
  { value: 'volume', label: '卷' },
  { value: 'network', label: '网络' },
  { value: 'daemon', label: '守护进程' },
]

/** 事件流页 */
export function Events() {
  const [type, setType] = useState('all')
  const { events, isLoading, error, refetch, connected } = useEventStream(200)

  const filtered = useMemo(
    () => (type === 'all' ? events : events.filter((event) => event.type === type)),
    [events, type],
  )

  return (
    <div className="space-y-6">
      <PageHeader
        title="事件流"
        description={
          <span className="flex flex-wrap items-center gap-1.5">
            Docker 引擎事件实时推送
            <span
              className={cn(
                'inline-flex items-center gap-1 text-[11px]',
                connected ? 'text-success' : 'text-fg-subtle',
              )}
            >
              <span
                className={cn(
                  'h-1.5 w-1.5 rounded-full',
                  connected ? 'dot-running' : 'dot-stopped',
                )}
              />
              {connected ? '实时已连接' : '实时未连接'}
            </span>
          </span>
        }
        actions={
          <>
            <Select value={type} onValueChange={setType}>
              <SelectTrigger className="w-32">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {TYPE_OPTIONS.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Button variant="default" onClick={() => void refetch()}>
              <RefreshCw className="h-3.5 w-3.5" />
              刷新
            </Button>
          </>
        }
      />

      <section className="elev-card overflow-hidden">
        {isLoading && events.length === 0 ? (
          <div className="space-y-3 p-5">
            {Array.from({ length: 8 }).map((_, index) => (
              <div key={index} className="flex gap-3">
                <div className="skeleton h-7 w-7 rounded-md" />
                <div className="flex-1 space-y-2">
                  <div className="skeleton h-4 w-1/3" />
                  <div className="skeleton h-3 w-1/2" />
                </div>
              </div>
            ))}
          </div>
        ) : error && events.length === 0 ? (
          <ErrorState
            title="无法加载事件流"
            message={error instanceof Error ? error.message : undefined}
            onRetry={() => void refetch()}
          />
        ) : filtered.length === 0 ? (
          <EmptyState
            icon={Activity}
            title="暂无事件"
            description="Docker 引擎产生动作时，事件会实时出现在这里。"
          />
        ) : (
          <EventStream events={filtered} maxHeightClass="max-h-[calc(100vh-260px)]" />
        )}
      </section>
    </div>
  )
}
