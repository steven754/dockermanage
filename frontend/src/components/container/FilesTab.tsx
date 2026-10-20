import { FolderTree, HardDrive, Info } from 'lucide-react'
import { useContainer } from '@/hooks/useContainers'
import { Skeleton } from '@/components/ui/input'
import { ErrorState } from '@/components/common/States'
import { cn } from '@/lib/utils'

/**
 * 容器详情 · Files 标签页。
 *
 * 说明：容器内文件浏览器在 SPEC §3.2 中被拍板推迟到 v1.1（仅预留 API 边界），
 * 因此这里展示挂载点清单与规划说明，而不是一个假的文件树。
 */
export function FilesTab({ containerId }: { containerId: string }) {
  const { data, isLoading, error, refetch } = useContainer(containerId)

  if (isLoading) {
    return (
      <div className="elev-card space-y-2 p-5">
        {Array.from({ length: 4 }).map((_, index) => (
          <Skeleton key={index} className="h-4 w-full" />
        ))}
      </div>
    )
  }

  if (error) {
    return (
      <div className="elev-card p-5">
        <ErrorState
          title="无法加载容器信息"
          message={error instanceof Error ? error.message : undefined}
          onRetry={() => void refetch()}
        />
      </div>
    )
  }

  const mounts = data?.mounts ?? []

  return (
    <div className="space-y-4">
      <div className="elev-card p-5">
        <div className="mb-3 flex items-center gap-2">
          <FolderTree className="h-4 w-4 text-brand" />
          <h3 className="text-sm font-semibold">容器内文件浏览器</h3>
          <span className="rounded-full border border-warning/30 bg-warning-soft px-2 py-0.5 text-[11px] text-warning">
            v1.1 规划中
          </span>
        </div>
        <p className="text-[13px] leading-relaxed text-fg-muted">
          该功能在 SPEC §3.2 中被拍板推迟到 v1.1，v1.0 仅预留 API 边界。
          下面列出当前容器的挂载点，可作为文件浏览的入口参考。
        </p>
      </div>

      <div className="elev-card overflow-hidden">
        <div className="flex items-center gap-2 border-b border-border px-4 py-2.5">
          <HardDrive className="h-4 w-4 text-fg-subtle" />
          <h3 className="text-sm font-semibold">挂载点</h3>
          <span className="font-normal text-fg-subtle">· {mounts.length} 项</span>
        </div>
        {mounts.length === 0 ? (
          <div className="flex flex-col items-center gap-2 py-10 text-center">
            <Info className="h-5 w-5 text-fg-subtle" />
            <p className="text-[13px] text-fg-muted">该容器没有挂载卷或目录</p>
          </div>
        ) : (
          <ul className="divide-y divide-border/60">
            {mounts.map((mount) => (
              <li key={`${mount.type}-${mount.destination}`} className="row-hover px-4 py-2.5">
                <div className="flex items-center gap-2">
                  <span className="rounded-full border border-border bg-bg-muted px-2 py-0.5 text-[11px] text-fg-muted">
                    {mount.type}
                  </span>
                  <span className="font-mono text-[12px] text-fg-secondary">{mount.destination}</span>
                  {mount.readOnly ? (
                    <span className="rounded-full border border-border px-2 py-0.5 text-[11px] text-fg-subtle">
                      只读
                    </span>
                  ) : null}
                </div>
                <div className="mt-1 font-mono text-[11px] text-fg-subtle">
                  {mount.name ? `${mount.name} → ` : ''}
                  {mount.source}
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>

      <p className={cn('text-[11px] text-fg-subtle')}>
        提示：文件浏览功能在 v1.1 提供，届时可直接在挂载点内查看 / 下载文件。
      </p>
    </div>
  )
}
