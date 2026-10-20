import { useMemo, useState } from 'react'
import { Network as NetworkIcon, MoreHorizontal, Search, Trash2 } from 'lucide-react'
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
import { useNetworkActions, useNetworks } from '@/hooks/useResources'
import { formatDate } from '@/lib/format'
import type { Network } from '@/lib/types'

/** 网络列表页 */
export function Networks() {
  const { data, isLoading, error, refetch } = useNetworks()
  const { remove } = useNetworkActions()

  const [keyword, setKeyword] = useState('')
  const [pendingDelete, setPendingDelete] = useState<Network | null>(null)

  const networks = useMemo(() => data?.items ?? [], [data])

  const filtered = useMemo(() => {
    const kw = keyword.trim().toLowerCase()
    if (!kw) return networks
    return networks.filter(
      (network) =>
        network.name.toLowerCase().includes(kw) || network.driver.toLowerCase().includes(kw),
    )
  }, [networks, keyword])

  return (
    <div className="space-y-6">
      <PageHeader title="网络" description={`共 ${networks.length} 个网络`} />

      <section className="elev-card overflow-hidden">
        <div className="flex flex-wrap items-center gap-2 border-b border-border p-3">
          <div className="relative min-w-[200px] flex-1">
            <Search className="absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-fg-subtle" />
            <Input
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
              placeholder="按网络名或驱动过滤…"
              className="h-8 pl-8"
            />
          </div>
        </div>

        {isLoading && networks.length === 0 ? (
          <div className="space-y-2 p-4">
            {Array.from({ length: 4 }).map((_, index) => (
              <Skeleton key={index} className="h-4 w-full" />
            ))}
          </div>
        ) : error && networks.length === 0 ? (
          <ErrorState
            title="无法加载网络列表"
            message={error instanceof Error ? error.message : undefined}
            onRetry={() => void refetch()}
          />
        ) : filtered.length === 0 ? (
          <EmptyState
            icon={NetworkIcon}
            title={networks.length === 0 ? '还没有网络' : '没有匹配的网络'}
            description={
              networks.length === 0
                ? '创建容器时接入网络会自动生成 bridge 网络。'
                : '换个关键字试试。'
            }
          />
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full border-collapse">
              <thead className="text-[11px] uppercase tracking-wider text-fg-subtle">
                <tr className="border-b border-border">
                  <th className="px-4 py-2.5 text-left font-medium">名称</th>
                  <th className="px-4 py-2.5 text-left font-medium">ID</th>
                  <th className="px-4 py-2.5 text-left font-medium">驱动</th>
                  <th className="px-4 py-2.5 text-left font-medium">子网 / 网关</th>
                  <th className="px-4 py-2.5 text-left font-medium">属性</th>
                  <th className="px-4 py-2.5 text-left font-medium">容器</th>
                  <th className="px-4 py-2.5 text-left font-medium">创建</th>
                  <th className="w-10 px-4 py-2.5" />
                </tr>
              </thead>
              <tbody>
                {filtered.map((network) => (
                  <tr
                    key={network.id}
                    className="row-hover border-b border-border/60 last:border-b-0"
                  >
                    <td className="px-4 py-2.5">
                      <div className="flex items-center gap-2">
                        <NetworkIcon className="h-3.5 w-3.5 text-fg-subtle" />
                        <span className="font-medium">{network.name}</span>
                      </div>
                    </td>
                    <td className="px-4 py-2.5 font-mono text-[11px] text-fg-subtle">
                      {network.id.slice(0, 12)}
                    </td>
                    <td className="px-4 py-2.5 font-mono text-[12px] text-fg-secondary">
                      {network.driver}
                    </td>
                    <td className="px-4 py-2.5 font-mono text-[11px] text-fg-subtle">
                      {network.ipam?.subnet ?? '—'}
                      {network.ipam?.gateway ? (
                        <div className="text-fg-muted">网关 {network.ipam.gateway}</div>
                      ) : null}
                    </td>
                    <td className="px-4 py-2.5">
                      <div className="flex flex-wrap gap-1">
                        {network.internal ? (
                          <span className="rounded-full border border-border bg-bg-muted px-2 py-0.5 text-[11px] text-fg-muted">
                            internal
                          </span>
                        ) : null}
                        {network.attachable ? (
                          <span className="rounded-full border border-info/30 bg-info-soft px-2 py-0.5 text-[11px] text-info">
                            attachable
                          </span>
                        ) : null}
                      </div>
                    </td>
                    <td className="px-4 py-2.5 font-mono text-[12px] text-fg-secondary">
                      {network.containerCount}
                    </td>
                    <td className="px-4 py-2.5 text-xs text-fg-muted">
                      {formatDate(network.createdAt)}
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
                            onSelect={() => setPendingDelete(network)}
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

      <ConfirmDialog
        open={pendingDelete !== null}
        onOpenChange={(open) => {
          if (!open) setPendingDelete(null)
        }}
        title={`删除网络 ${pendingDelete?.name ?? ''}？`}
        description="接入该网络的容器会被断开连接，此操作不可撤销。"
        onConfirm={() => {
          if (!pendingDelete) return
          remove.mutate(pendingDelete.id, {
            onSuccess: () => {
              toast.success('网络已删除')
              setPendingDelete(null)
            },
            onError: (err) => toast.error(err instanceof Error ? err.message : '删除失败'),
          })
        }}
      />
    </div>
  )
}
