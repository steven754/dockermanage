import { useMemo, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import {
  Download,
  Layers,
  MoreHorizontal,
  Package,
  Search,
  Tag,
  Trash2,
  TrendingUp,
} from 'lucide-react'
import { toast } from 'sonner'
import { Cell, Pie, PieChart, ResponsiveContainer, Tooltip } from 'recharts'
import { PageHeader } from '@/components/layout/PageHeader'
import { EmptyState, ErrorState } from '@/components/common/States'
import { ConfirmDialog } from '@/components/common/ConfirmDialog'
import { Button } from '@/components/ui/button'
import { Input, Skeleton } from '@/components/ui/input'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useImages, useImageActions } from '@/hooks/useResources'
import { formatBytes, formatDate, formatRelativeTime, shortId } from '@/lib/format'
import { cn } from '@/lib/utils'
import type { Image } from '@/lib/types'

const PIE_COLORS = [
  '#3b82f6',
  '#06b6d4',
  '#10b981',
  '#f59e0b',
  '#a78bfa',
  '#ef4444',
  '#ec4899',
  '#14b8a6',
  '#f97316',
  '#6366f1',
]

/** 镜像列表页 */
export function Images() {
  const [searchParams, setSearchParams] = useSearchParams()
  const { data, isLoading, error, refetch } = useImages()
  const { pull, remove, prune, tag } = useImageActions()

  const [keyword, setKeyword] = useState('')
  const [showDanglingOnly, setShowDanglingOnly] = useState(false)
  const [pullOpen, setPullOpen] = useState(searchParams.get('action') === 'pull')
  const [imageName, setImageName] = useState('')
  const [pulling, setPulling] = useState(false)
  const [pendingDelete, setPendingDelete] = useState<Image | null>(null)
  const [pendingPrune, setPendingPrune] = useState(false)
  const [tagTarget, setTagTarget] = useState<Image | null>(null)
  const [tagRepo, setTagRepo] = useState('')
  const [tagName, setTagName] = useState('')

  const images = useMemo(() => data?.items ?? [], [data])

  const filtered = useMemo(() => {
    let list = images
    if (showDanglingOnly) list = list.filter((image) => image.dangling)
    const kw = keyword.trim().toLowerCase()
    if (kw) {
      list = list.filter(
        (image) =>
          image.repoTags.some((tagName) => tagName.toLowerCase().includes(kw)) ||
          image.id.toLowerCase().includes(kw),
      )
    }
    return list
  }, [images, keyword, showDanglingOnly])

  const totalSize = images.reduce((sum, image) => sum + image.sizeBytes, 0)
  const danglingCount = images.filter((image) => image.dangling).length

  const pieData = useMemo(
    () =>
      images
        .slice()
        .sort((a, b) => b.sizeBytes - a.sizeBytes)
        .slice(0, 8)
        .map((image, index) => ({
          name: image.repoTags[0] ?? shortId(image.id),
          value: image.sizeBytes,
          color: PIE_COLORS[index % PIE_COLORS.length],
        })),
    [images],
  )

  const handlePull = async () => {
    const name = imageName.trim()
    if (!name) return
    setPulling(true)
    try {
      await pull.mutateAsync(name)
      toast.success(`已拉取 ${name}`)
      setPullOpen(false)
      setImageName('')
      setSearchParams({})
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '拉取失败')
    } finally {
      setPulling(false)
    }
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="镜像"
        description={
          <span className="flex flex-wrap items-center gap-1.5">
            共 {images.length} 个
            <span className="text-fg-subtle">· 占用 {formatBytes(totalSize)}</span>
            {danglingCount > 0 ? (
              <span className="text-warning">· {danglingCount} 个悬空</span>
            ) : null}
          </span>
        }
        actions={
          <>
            <Button variant="default" onClick={() => setPendingPrune(true)}>
              <Trash2 className="h-3.5 w-3.5" />
              清理 dangling
            </Button>
            <Button variant="primary" onClick={() => setPullOpen(true)}>
              <Download className="h-3.5 w-3.5" />
              Pull 镜像
            </Button>
          </>
        }
      />

      {/* ===== 大小可视化 ===== */}
      {images.length > 0 ? (
        <section className="grid grid-cols-1 gap-4 xl:grid-cols-3">
          <div className="elev-card p-5 xl:col-span-1">
            <h2 className="text-base font-semibold">大小分布</h2>
            <p className="mt-0.5 text-xs text-fg-muted">前 {pieData.length} 个镜像占用的空间</p>
            <div className="mt-2 h-48">
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie
                    data={pieData}
                    dataKey="value"
                    nameKey="name"
                    innerRadius={44}
                    outerRadius={72}
                    paddingAngle={2}
                    stroke="none"
                  >
                    {pieData.map((entry) => (
                      <Cell key={entry.name} fill={entry.color} />
                    ))}
                  </Pie>
                  <Tooltip
                    contentStyle={{
                      background: 'rgb(var(--bg-card))',
                      border: '1px solid rgb(var(--border))',
                      borderRadius: 8,
                      fontSize: 11,
                      color: 'rgb(var(--fg-primary))',
                    }}
                    formatter={(value: number) => formatBytes(value)}
                  />
                </PieChart>
              </ResponsiveContainer>
            </div>
            <ul className="mt-2 space-y-1.5">
              {pieData.slice(0, 5).map((entry) => (
                <li key={entry.name} className="flex items-center gap-2 text-[11px]">
                  <span className="h-2 w-2 shrink-0 rounded-full" style={{ background: entry.color }} />
                  <span className="min-w-0 flex-1 truncate font-mono text-fg-secondary">
                    {entry.name}
                  </span>
                  <span className="font-mono text-fg-subtle">{formatBytes(entry.value)}</span>
                </li>
              ))}
            </ul>
          </div>

          <div className="elev-card overflow-hidden xl:col-span-2">
            <div className="flex flex-wrap items-center gap-2 border-b border-border p-3">
              <div className="relative min-w-[200px] flex-1">
                <Search className="absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-fg-subtle" />
                <Input
                  value={keyword}
                  onChange={(e) => setKeyword(e.target.value)}
                  placeholder="按仓库名或标签过滤…"
                  className="h-8 pl-8"
                />
              </div>
              <Button
                variant={showDanglingOnly ? 'default' : 'ghost'}
                size="sm"
                onClick={() => setShowDanglingOnly((value) => !value)}
              >
                <Package className="h-3.5 w-3.5" />
                仅悬空 <span className="font-mono text-fg-subtle">{danglingCount}</span>
              </Button>
            </div>

            {isLoading && images.length === 0 ? (
              <div className="space-y-2 p-4">
                {Array.from({ length: 6 }).map((_, index) => (
                  <Skeleton key={index} className="h-4 w-full" />
                ))}
              </div>
            ) : error && images.length === 0 ? (
              <ErrorState
                title="无法加载镜像列表"
                message={error instanceof Error ? error.message : undefined}
                onRetry={() => void refetch()}
              />
            ) : filtered.length === 0 ? (
              <EmptyState
                icon={Layers}
                title={images.length === 0 ? '还没有镜像' : '没有匹配的镜像'}
                description={
                  images.length === 0 ? '拉取第一个镜像开始使用。' : '换个关键字试试。'
                }
                action={
                  images.length === 0 ? (
                    <Button variant="primary" size="sm" onClick={() => setPullOpen(true)}>
                      <Download className="h-3.5 w-3.5" />
                      Pull 镜像
                    </Button>
                  ) : null
                }
              />
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full border-collapse">
                  <thead className="text-[11px] uppercase tracking-wider text-fg-subtle">
                    <tr className="border-b border-border">
                      <th className="px-4 py-2.5 text-left font-medium">仓库 / 标签</th>
                      <th className="px-4 py-2.5 text-left font-medium">ID</th>
                      <th className="w-40 px-4 py-2.5 text-left font-medium">大小</th>
                      <th className="px-4 py-2.5 text-left font-medium">容器</th>
                      <th className="px-4 py-2.5 text-left font-medium">创建</th>
                      <th className="w-10 px-4 py-2.5" />
                    </tr>
                  </thead>
                  <tbody>
                    {filtered.map((image) => (
                      <tr
                        key={image.id}
                        className="row-hover border-b border-border/60 last:border-b-0"
                      >
                        <td className="px-4 py-2.5">
                          {image.repoTags.length > 0 ? (
                            <div className="flex flex-col">
                              {image.repoTags.map((tagName) => (
                                <span key={tagName} className="font-mono text-[12px] text-fg-primary">
                                  {tagName}
                                </span>
                              ))}
                            </div>
                          ) : (
                            <span className="inline-flex items-center gap-1.5 rounded-full border border-warning/30 bg-warning-soft px-2 py-0.5 text-[11px] text-warning">
                              <Package className="h-3 w-3" />
                              &lt;none&gt;:&lt;none&gt;
                            </span>
                          )}
                        </td>
                        <td className="px-4 py-2.5 font-mono text-[11px] text-fg-subtle">
                          {shortId(image.id)}
                        </td>
                        <td className="whitespace-nowrap px-4 py-2.5">
                          <div className="flex items-center gap-2">
                            <div className="meter h-1 w-20">
                              <span
                                style={{
                                  width: `${Math.min((image.sizeBytes / (totalSize || 1)) * 100, 100)}%`,
                                }}
                              />
                            </div>
                            <span className="font-mono text-[11px] text-fg-secondary">
                              {formatBytes(image.sizeBytes)}
                            </span>
                          </div>
                          {image.sharedSizeBytes > 0 ? (
                            <div className="mt-0.5 font-mono text-[10px] text-fg-subtle">
                              共享 {formatBytes(image.sharedSizeBytes)}
                            </div>
                          ) : null}
                        </td>
                        <td className="px-4 py-2.5 font-mono text-xs text-fg-secondary">
                          {image.containerCount}
                        </td>
                        <td className="px-4 py-2.5 text-xs text-fg-muted">
                          {formatRelativeTime(image.createdAt)}
                          <div className="font-mono text-[10px] text-fg-subtle">
                            {formatDate(image.createdAt)}
                          </div>
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
                                onSelect={() => {
                                  setTagTarget(image)
                                  setTagRepo(image.repoTags[0]?.split(':')[0] ?? '')
                                  setTagName(image.repoTags[0]?.split(':')[1] ?? 'latest')
                                }}
                              >
                                <Tag className="h-3.5 w-3.5" />
                                打标签
                              </DropdownMenuItem>
                              <DropdownMenuItem
                                className="text-danger"
                                onSelect={() => setPendingDelete(image)}
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
          </div>
        </section>
      ) : (
        <section className="elev-card p-5">
          {isLoading ? (
            <div className="space-y-2">
              {Array.from({ length: 5 }).map((_, index) => (
                <Skeleton key={index} className="h-4 w-full" />
              ))}
            </div>
          ) : error ? (
            <ErrorState
              title="无法加载镜像列表"
              message={error instanceof Error ? error.message : undefined}
              onRetry={() => void refetch()}
            />
          ) : (
            <EmptyState
              icon={Layers}
              title="还没有镜像"
              description="拉取第一个镜像开始使用。"
              action={
                <Button variant="primary" size="sm" onClick={() => setPullOpen(true)}>
                  <Download className="h-3.5 w-3.5" />
                  Pull 镜像
                </Button>
              }
            />
          )}
        </section>
      )}

      {/* ===== Pull 弹窗 ===== */}
      <Dialog
        open={pullOpen}
        onOpenChange={(open) => {
          setPullOpen(open)
          if (!open) setSearchParams({})
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Pull 镜像</DialogTitle>
            <DialogDescription>输入要拉取的镜像名，例如 nginx:1.27-alpine</DialogDescription>
          </DialogHeader>
          <div className="mt-4">
            <Input
              value={imageName}
              onChange={(e) => setImageName(e.target.value)}
              placeholder="nginx:1.27-alpine"
              onKeyDown={(e) => {
                if (e.key === 'Enter') void handlePull()
              }}
            />
          </div>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setPullOpen(false)}>
              取消
            </Button>
            <Button variant="primary" onClick={() => void handlePull()} disabled={pulling}>
              <Download className="h-3.5 w-3.5" />
              {pulling ? '拉取中…' : '拉取'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ===== 打标签弹窗 ===== */}
      <Dialog
        open={tagTarget !== null}
        onOpenChange={(open) => {
          if (!open) setTagTarget(null)
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>打标签</DialogTitle>
            <DialogDescription>为镜像 {tagTarget ? shortId(tagTarget.id) : ''} 添加新的标签</DialogDescription>
          </DialogHeader>
          <div className="mt-4 grid grid-cols-3 gap-2">
            <Input
              className="col-span-2"
              value={tagRepo}
              onChange={(e) => setTagRepo(e.target.value)}
              placeholder="仓库名 myrepo/nginx"
            />
            <Input
              value={tagName}
              onChange={(e) => setTagName(e.target.value)}
              placeholder="标签 v1"
            />
          </div>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setTagTarget(null)}>
              取消
            </Button>
            <Button
              variant="primary"
              onClick={() => {
                if (!tagTarget) return
                tag.mutate(
                  { id: tagTarget.id, repo: tagRepo.trim(), tag: tagName.trim() },
                  {
                    onSuccess: () => {
                      toast.success('标签已更新')
                      setTagTarget(null)
                    },
                    onError: (err) => toast.error(err instanceof Error ? err.message : '打标签失败'),
                  },
                )
              }}
            >
              确认
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ===== 删除确认 ===== */}
      <ConfirmDialog
        open={pendingDelete !== null}
        onOpenChange={(open) => {
          if (!open) setPendingDelete(null)
        }}
        title="删除镜像？"
        description={
          pendingDelete
            ? `镜像 ${pendingDelete.repoTags[0] ?? shortId(pendingDelete.id)} 将从本地删除，被 ${pendingDelete.containerCount} 个容器使用时可能失败。`
            : ''
        }
        onConfirm={() => {
          if (!pendingDelete) return
          remove.mutate(
            { id: pendingDelete.id, force: false },
            {
              onSuccess: () => {
                toast.success('镜像已删除')
                setPendingDelete(null)
              },
              onError: (err) => toast.error(err instanceof Error ? err.message : '删除失败'),
            },
          )
        }}
      />

      {/* ===== 清理 dangling 确认 ===== */}
      <ConfirmDialog
        open={pendingPrune}
        onOpenChange={setPendingPrune}
        title="清理悬空镜像？"
        description={`将删除 ${danglingCount} 个没有标签的悬空镜像并回收磁盘空间。`}
        confirmText="确认清理"
        onConfirm={() => {
          prune.mutate(true, {
            onSuccess: (result) => {
              toast.success(
                `已清理 ${result.deleted.length} 个镜像，回收 ${formatBytes(result.spaceReclaimedBytes)}`,
              )
              setPendingPrune(false)
            },
            onError: (err) => toast.error(err instanceof Error ? err.message : '清理失败'),
          })
        }}
      />

      <p className={cn('flex items-center gap-1.5 text-[11px] text-fg-subtle')}>
        <TrendingUp className="h-3 w-3" />
        镜像大小含共享层，实际占用以 Docker 引擎统计为准。
      </p>
    </div>
  )
}
