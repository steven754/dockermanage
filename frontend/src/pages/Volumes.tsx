import { useMemo, useState } from 'react'
import { Database, HardDrive, MoreHorizontal, Search, Trash2, TriangleAlert } from 'lucide-react'
import { toast } from 'sonner'
import { PageHeader } from '@/components/layout/PageHeader'
import { EmptyState, ErrorState } from '@/components/common/States'
import { ConfirmDialog } from '@/components/common/ConfirmDialog'
import { Button } from '@/components/ui/button'
import { Input, Skeleton } from '@/components/ui/input'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useVolumeActions, useVolumes } from '@/hooks/useResources'
import { formatBytes, formatDate } from '@/lib/format'
import { cn } from '@/lib/utils'
import type { Volume } from '@/lib/types'

/** 卷列表页（孤儿卷高亮） */
export function Volumes() {
  const { data, isLoading, error, refetch } = useVolumes()
  const { remove, prune } = useVolumeActions()

  const [keyword, setKeyword] = useState('')
  const [orphanOnly, setOrphanOnly] = useState(false)
  const [pendingDelete, setPendingDelete] = useState<Volume | null>(null)
  const [pendingPrune, setPendingPrune] = useState(false)

  const volumes = useMemo(() => data?.items ?? [], [data])
  const orphanCount = volumes.filter((volume) => volume.orphan).length

  const filtered = useMemo(() => {
    let list = volumes
    if (orphanOnly) list = list.filter((volume) => volume.orphan)
    const kw = keyword.trim().toLowerCase()
    if (kw) list = list.filter((volume) => volume.name.toLowerCase().includes(kw))
    return list
  }, [volumes, keyword, orphanOnly])

  const totalSize = volumes.reduce((sum, volume) => sum + volume.sizeBytes, 0)

  return (
    <div className="space-y-6">
      <PageHeader
        title="卷"
        description={
          <span className="flex flex-wrap items-center gap-1.5">
            共 {volumes.length} 个
            <span className="text-fg-subtle">· 占用 {formatBytes(totalSize)}</span>
            {orphanCount > 0 ? (
              <span className="text-warning">· {orphanCount} 个孤儿卷</span>
            ) : null}
          </span>
        }
        actions={
          <Button variant="default" onClick={() => setPendingPrune(true)}>
            <Trash2 className="h-3.5 w-3.5" />
            清理孤儿卷
          </Button>
        }
      />

      <section className="elev-card overflow-hidden">
        <div className="flex flex-wrap items-center gap-2 border-b border-border p-3">
          <div className="relative min-w-[200px] flex-1">
            <Search className="absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-fg-subtle" />
            <Input
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
              placeholder="按卷名过滤…"
              className="h-8 pl-8"
            />
          </div>
          <Button
            variant={orphanOnly ? 'default' : 'ghost'}
            size="sm"
            onClick={() => setOrphanOnly((value) => !value)}
          >
            <TriangleAlert className="h-3.5 w-3.5" />
            仅孤儿卷 <span className="font-mono text-fg-subtle">{orphanCount}</span>
          </Button>
        </div>

        {isLoading && volumes.length === 0 ? (
          <div className="space-y-2 p-4">
            {Array.from({ length: 5 }).map((_, index) => (
              <Skeleton key={index} className="h-4 w-full" />
            ))}
          </div>
        ) : error && volumes.length === 0 ? (
          <ErrorState
            title="无法加载卷列表"
            message={error instanceof Error ? error.message : undefined}
            onRetry={() => void refetch()}
          />
        ) : filtered.length === 0 ? (
          <EmptyState
            icon={Database}
            title={volumes.length === 0 ? '还没有卷' : '没有匹配的卷'}
            description={
              volumes.length === 0 ? '容器创建卷后会出现在这里。' : '换个关键字试试。'
            }
          />
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full border-collapse">
              <thead className="text-[11px] uppercase tracking-wider text-fg-subtle">
                <tr className="border-b border-border">
                  <th className="px-4 py-2.5 text-left font-medium">名称</th>
                  <th className="px-4 py-2.5 text-left font-medium">驱动</th>
                  <th className="px-4 py-2.5 text-left font-medium">挂载点</th>
                  <th className="px-4 py-2.5 text-left font-medium">大小</th>
                  <th className="px-4 py-2.5 text-left font-medium">引用</th>
                  <th className="px-4 py-2.5 text-left font-medium">创建</th>
                  <th className="w-10 px-4 py-2.5" />
                </tr>
              </thead>
              <tbody>
                {filtered.map((volume) => (
                  <tr
                    key={volume.name}
                    className={cn(
                      'row-hover border-b border-border/60 last:border-b-0',
                      volume.orphan && 'bg-warning-soft/20',
                    )}
                  >
                    <td className="px-4 py-2.5">
                      <div className="flex items-center gap-2">
                        <Database className="h-3.5 w-3.5 text-fg-subtle" />
                        <span className="font-mono text-[12px] text-fg-primary">{volume.name}</span>
                        {volume.orphan ? (
                          <span className="inline-flex items-center gap-1 rounded-full border border-warning/30 bg-warning-soft px-2 py-0.5 text-[11px] text-warning">
                            <TriangleAlert className="h-3 w-3" />
                            孤儿卷
                          </span>
                        ) : null}
                      </div>
                    </td>
                    <td className="px-4 py-2.5 font-mono text-[12px] text-fg-secondary">
                      {volume.driver}
                    </td>
                    <td className="max-w-[280px] truncate px-4 py-2.5 font-mono text-[11px] text-fg-subtle">
                      {volume.mountpoint}
                    </td>
                    <td className="px-4 py-2.5 font-mono text-[12px] text-fg-secondary">
                      {formatBytes(volume.sizeBytes)}
                    </td>
                    <td className="px-4 py-2.5 font-mono text-[12px] text-fg-secondary">
                      {volume.refCount}
                    </td>
                    <td className="px-4 py-2.5 text-xs text-fg-muted">
                      {formatDate(volume.createdAt)}
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
                            className="text-danger"
                            onSelect={() => setPendingDelete(volume)}
                          >
                            <Trash2 className="h-3.5 w-3.5" />
                            删除
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <p className="flex items-center gap-1.5 text-[11px] text-fg-subtle">
        <HardDrive className="h-3 w-3" />
        孤儿卷指未被任何容器引用的卷，清理前请确认其中没有需要保留的数据。
      </p>

      <ConfirmDialog
        open={pendingDelete !== null}
        onOpenChange={(open) => {
          if (!open) setPendingDelete(null)
        }}
        title={`删除卷 ${pendingDelete?.name ?? ''}？`}
        description="卷中的数据将被永久删除，此操作不可撤销。"
        onConfirm={() => {
          if (!pendingDelete) return
          remove.mutate(
            { name: pendingDelete.name, force: false },
            {
              onSuccess: () => {
                toast.success('卷已删除')
                setPendingDelete(null)
              },
              onError: (err) => toast.error(err instanceof Error ? err.message : '删除失败'),
            },
          )
        }}
      />

      <ConfirmDialog
        open={pendingPrune}
        onOpenChange={setPendingPrune}
        title="清理孤儿卷？"
        description={`将删除 ${orphanCount} 个未被容器引用的卷，卷内数据不可恢复。`}
        confirmText="确认清理"
        onConfirm={() => {
          prune.mutate(undefined, {
            onSuccess: (result) => {
              toast.success(
                `已清理 ${result.deleted.length} 个卷，回收 ${formatBytes(result.spaceReclaimedBytes)}`,
              )
              setPendingPrune(false)
            },
            onError: (err) => toast.error(err instanceof Error ? err.message : '清理失败'),
          })
        }}
      />
    </div>
  )
}
