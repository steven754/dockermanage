import { type LucideIcon } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

export interface QuickAction {
  id: string
  label: string
  hint: string
  icon: LucideIcon
  iconClass: string
  onSelect: () => void
  disabled?: boolean
  pending?: boolean
}

/**
 * 快捷操作内容（2×2 网格）。
 * 不自带卡片外框 —— 由调用方与「系统状态」放进同一张卡片里，
 * 与设计稿一致。
 */
export function QuickActions({
  actions,
  className,
}: {
  actions: QuickAction[]
  className?: string
}) {
  return (
    <div className={className}>
      <h2 className="mb-4 text-base font-semibold">快捷操作</h2>
      <div className="grid grid-cols-2 gap-2.5">
        {actions.map((action) => {
          const Icon = action.icon
          return (
            <Button
              key={action.id}
              variant="default"
              onClick={action.onSelect}
              disabled={action.disabled || action.pending}
              className="h-auto flex-col items-start gap-1 py-3 hover:border-brand"
            >
              <Icon className={cn('h-4 w-4', action.iconClass)} />
              <span className="text-sm font-medium">
                {action.pending ? '处理中…' : action.label}
              </span>
              <span className="font-normal text-[11px] text-fg-subtle">{action.hint}</span>
            </Button>
          )
        })}
      </div>
    </div>
  )
}
