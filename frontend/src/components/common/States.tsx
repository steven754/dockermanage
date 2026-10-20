import { AlertCircle, Inbox, RefreshCw, type LucideIcon } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

/** 空状态：图标 + 文案 + 可选操作 */
export function EmptyState({
  icon: Icon = Inbox,
  title,
  description,
  action,
  className,
}: {
  icon?: LucideIcon
  title: string
  description?: string
  action?: React.ReactNode
  className?: string
}) {
  return (
    <div className={cn('empty-state', className)}>
      <div className="mb-3 flex h-11 w-11 items-center justify-center rounded-xl bg-bg-muted text-fg-subtle">
        <Icon className="h-5 w-5" />
      </div>
      <p className="text-sm font-medium text-fg-secondary">{title}</p>
      {description ? <p className="mt-1 max-w-sm text-xs text-fg-subtle">{description}</p> : null}
      {action ? <div className="mt-4">{action}</div> : null}
    </div>
  )
}

/** 错误态：友好提示 + 重试 */
export function ErrorState({
  title = '数据加载失败',
  message,
  onRetry,
  className,
}: {
  title?: string
  message?: string
  onRetry?: () => void
  className?: string
}) {
  return (
    <div className={cn('empty-state', className)}>
      <div className="mb-3 flex h-11 w-11 items-center justify-center rounded-xl bg-danger-soft text-danger">
        <AlertCircle className="h-5 w-5" />
      </div>
      <p className="text-sm font-medium text-fg-secondary">{title}</p>
      <p className="mt-1 max-w-md text-xs text-fg-subtle">
        {message ?? '后端服务暂时不可用，请确认面板已启动后重试。'}
      </p>
      {onRetry ? (
        <Button variant="outline" size="sm" className="mt-4" onClick={onRetry}>
          <RefreshCw className="h-3.5 w-3.5" />
          重新加载
        </Button>
      ) : null}
    </div>
  )
}
