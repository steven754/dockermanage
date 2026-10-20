import { useEffect, useMemo, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import {
  ArrowLeft,
  CheckCircle2,
  FileCode2,
  FilePlus2,
  FolderOpen,
  History,
  ListChecks,
  Play,
  Rocket,
  Save,
  Square,
  Terminal,
} from 'lucide-react'
import { toast } from 'sonner'
import { PageHeader } from '@/components/layout/PageHeader'
import { EmptyState, ErrorState } from '@/components/common/States'
import { ConfirmDialog } from '@/components/common/ConfirmDialog'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/input'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { YamlEditor, extractServices, validateCompose } from '@/components/compose/YamlEditor'
import { ValidationPanel } from '@/components/compose/ValidationPanel'
import { ServiceList } from '@/components/compose/ServiceList'
import { useComposeActions, useComposeProject, useComposeProjects } from '@/hooks/useCompose'
import { formatRelativeTime } from '@/lib/format'
import { cn } from '@/lib/utils'

/** Compose 项目页：项目列表 + 在线编辑 / 校验 / 部署 */
export function Compose() {
  const { project } = useParams<{ project: string }>()
  return project ? <ComposeEditor project={project} /> : <ComposeProjectList />
}

/* ---------------------------------------------------------------- 项目列表 */

function ComposeProjectList() {
  const { data, isLoading, error, refetch } = useComposeProjects()
  const projects = useMemo(() => data?.items ?? [], [data])

  return (
    <div className="space-y-6">
      <PageHeader
        title="Compose"
        description="管理 /compose 目录下的 Compose 项目，支持在线编辑、实时校验与一键部署"
      />

      <section className="elev-card overflow-hidden">
        {isLoading && projects.length === 0 ? (
          <div className="space-y-2 p-4">
            {Array.from({ length: 4 }).map((_, index) => (
              <Skeleton key={index} className="h-14 w-full" />
            ))}
          </div>
        ) : error && projects.length === 0 ? (
          <ErrorState
            title="无法加载 Compose 项目"
            message={error instanceof Error ? error.message : undefined}
            onRetry={() => void refetch()}
          />
        ) : projects.length === 0 ? (
          <EmptyState
            icon={FolderOpen}
            title="还没有 Compose 项目"
            description="把 docker-compose.yml 放进面板挂载的 /compose 目录，就会出现在这里。"
          />
        ) : (
          <div className="grid grid-cols-1 gap-3 p-4 md:grid-cols-2 xl:grid-cols-3">
            {projects.map((item) => (
              <Link
                key={item.name}
                to={`/compose/${encodeURIComponent(item.name)}`}
                className="surface-card surface-card-hover p-4"
              >
                <div className="flex items-center gap-2">
                  <FileCode2 className="h-4 w-4 text-brand" />
                  <span className="truncate font-medium">{item.name}</span>
                </div>
                <p className="mt-1.5 truncate font-mono text-[11px] text-fg-subtle">{item.path}</p>
                <div className="mt-2.5 flex flex-wrap items-center gap-2 text-[11px]">
                  <span className="rounded-full border border-border bg-bg-muted px-2 py-0.5 text-fg-muted">
                    {item.serviceCount} 个服务
                  </span>
                  <span className="rounded-full border border-success/30 bg-success-soft px-2 py-0.5 text-success">
                    {item.runningCount} 运行中
                  </span>
                  <span className="text-fg-subtle">
                    {formatRelativeTime(item.updatedAt)}更新
                  </span>
                </div>
              </Link>
            ))}
          </div>
        )}
      </section>
    </div>
  )
}

/* ---------------------------------------------------------------- 编辑器 */

function ComposeEditor({ project }: { project: string }) {
  const navigate = useNavigate()
  const { data, isLoading, error, refetch } = useComposeProject(project)
  const { saveFile, up, down } = useComposeActions()

  const [activeFile, setActiveFile] = useState<string | null>(null)
  const [draft, setDraft] = useState<string>('')
  const [dirty, setDirty] = useState(false)
  const [selectedServices, setSelectedServices] = useState<string[]>([])
  const [confirmDown, setConfirmDown] = useState(false)
  const [deployLog, setDeployLog] = useState<string[]>([])

  const currentFile = useMemo(() => {
    const files = data?.files ?? []
    return files.find((file) => file.name === activeFile) ?? files[0] ?? null
  }, [data, activeFile])

  // 切换文件 / 首次加载时把内容载入编辑器
  useEffect(() => {
    if (currentFile) {
      setDraft(currentFile.content)
      setDirty(false)
    }
  }, [currentFile])

  const validation = useMemo(() => validateCompose(draft), [draft])
  const services = useMemo(() => extractServices(validation.document), [validation.document])

  const isComposeFile = (name: string) => /\.ya?ml$/i.test(name)

  const handleSave = () => {
    if (!currentFile) return
    saveFile.mutate(
      { name: project, filename: currentFile.name, content: draft },
      {
        onSuccess: () => {
          toast.success('已保存')
          setDirty(false)
        },
        onError: (err) => toast.error(err instanceof Error ? err.message : '保存失败'),
      },
    )
  }

  const handleUp = () => {
    setDeployLog((prev) => [...prev, `> docker compose -p ${project} up`])
    up.mutate(
      { name: project, services: selectedServices.length > 0 ? selectedServices : undefined },
      {
        onSuccess: (result) => {
          setDeployLog((prev) => [...prev, `✓ 已启动 ${result.started.join('、') || '全部服务'}`])
          toast.success('部署成功')
        },
        onError: (err) => {
          setDeployLog((prev) => [...prev, `✗ ${err instanceof Error ? err.message : '部署失败'}`])
          toast.error(err instanceof Error ? err.message : '部署失败')
        },
      },
    )
  }

  if (isLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-6 w-64" />
        <Skeleton className="h-[520px] w-full rounded-xl" />
      </div>
    )
  }

  if (error || !data) {
    return (
      <div className="space-y-6">
        <PageHeader
          title={
            <span className="flex items-center gap-2">
              <Button variant="ghost" size="icon-sm" onClick={() => navigate('/compose')}>
                <ArrowLeft className="h-4 w-4" />
              </Button>
              {project}
            </span>
          }
        />
        <div className="elev-card p-5">
          <ErrorState
            title="无法加载 Compose 项目"
            message={error instanceof Error ? error.message : '项目不存在。'}
            onRetry={() => void refetch()}
          />
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {/* ===== 项目头 ===== */}
      <div className="flex flex-wrap items-center gap-3">
        <Button variant="ghost" size="icon-sm" onClick={() => navigate('/compose')}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <FileCode2 className="h-4 w-4 text-brand" />
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <h1 className="text-lg font-semibold tracking-tight">
              {currentFile?.name ?? project}
            </h1>
            <span
              className={cn(
                'rounded-full border px-2 py-0.5 text-[11px]',
                dirty
                  ? 'border-warning/30 bg-warning-soft text-warning'
                  : 'border-success/30 bg-success-soft text-success',
              )}
            >
              {dirty ? '已修改' : '已生效'}
            </span>
          </div>
          <p className="truncate font-mono text-[11px] text-fg-subtle">{data.path}</p>
        </div>

        <div className="ml-auto flex flex-wrap items-center gap-2">
          <Button variant="default" onClick={handleSave} disabled={!dirty || !currentFile}>
            <Save className="h-3.5 w-3.5" />
            保存
          </Button>
          <Button
            variant="default"
            onClick={() => setConfirmDown(true)}
            disabled={!data.services.length}
          >
            <Square className="h-3.5 w-3.5" />
            停止
          </Button>
          <Button variant="primary" onClick={handleUp}>
            <Rocket className="h-3.5 w-3.5" />
            部署
          </Button>
        </div>
      </div>

      {/* ===== 文件标签 ===== */}
      <div className="flex items-center gap-1 overflow-x-auto border-b border-border">
        {data.files.map((file) => (
          <button
            key={file.name}
            type="button"
            onClick={() => setActiveFile(file.name)}
            className={cn(
              'flex items-center gap-1.5 whitespace-nowrap border-b-2 px-3 py-2 text-[13px] font-medium transition-colors',
              currentFile?.name === file.name
                ? 'border-brand text-fg-primary'
                : 'border-transparent text-fg-muted hover:text-fg-secondary',
            )}
          >
            <FileCode2 className="h-3.5 w-3.5" />
            {file.name}
          </button>
        ))}
        <Button variant="ghost" size="sm" className="ml-1 shrink-0">
          <FilePlus2 className="h-3.5 w-3.5" />
          新建文件
        </Button>
      </div>

      {/* ===== 主体：编辑器 + 右栏 ===== */}
      <div className="grid grid-cols-1 gap-4 xl:grid-cols-3">
        <div className="xl:col-span-2">
          {currentFile && isComposeFile(currentFile.name) ? (
            <YamlEditor value={draft} onChange={(value) => { setDraft(value); setDirty(true) }} />
          ) : (
            <div className="code-block min-h-[420px] whitespace-pre-wrap">{draft}</div>
          )}
        </div>

        <div className="elev-card overflow-hidden">
          <Tabs defaultValue="validate">
            <div className="border-b border-border px-2">
              <TabsList className="border-b-0">
                <TabsTrigger value="validate" className="px-2 text-xs">
                  <CheckCircle2 className="mr-1 inline h-3.5 w-3.5 -mt-0.5" />
                  校验
                </TabsTrigger>
                <TabsTrigger value="services" className="px-2 text-xs">
                  <ListChecks className="mr-1 inline h-3.5 w-3.5 -mt-0.5" />
                  服务
                </TabsTrigger>
                <TabsTrigger value="logs" className="px-2 text-xs">
                  <Terminal className="mr-1 inline h-3.5 w-3.5 -mt-0.5" />
                  部署日志
                </TabsTrigger>
                <TabsTrigger value="history" className="px-2 text-xs">
                  <History className="mr-1 inline h-3.5 w-3.5 -mt-0.5" />
                  历史
                </TabsTrigger>
              </TabsList>
            </div>

            <TabsContent value="validate" className="p-4">
              <ValidationPanel result={validation} />
            </TabsContent>

            <TabsContent value="services" className="p-0">
              <ServiceList
                services={services}
                runtime={data.services}
                selected={selectedServices}
                onToggle={(name) =>
                  setSelectedServices((prev) =>
                    prev.includes(name)
                      ? prev.filter((item) => item !== name)
                      : [...prev, name],
                  )
                }
                onUp={(name) =>
                  up.mutate(
                    { name: project, services: [name] },
                    {
                      onSuccess: () => toast.success(`已启动 ${name}`),
                      onError: (err) =>
                        toast.error(err instanceof Error ? err.message : '启动失败'),
                    },
                  )
                }
                onDown={(name) =>
                  down.mutate(
                    { name: project },
                    {
                      onSuccess: () => toast.success(`已停止 ${name}`),
                      onError: (err) =>
                        toast.error(err instanceof Error ? err.message : '停止失败'),
                    },
                  )
                }
              />
            </TabsContent>

            <TabsContent value="logs" className="p-0">
              {deployLog.length === 0 ? (
                <EmptyState
                  icon={Terminal}
                  title="暂无部署日志"
                  description="点击右上角「部署」后，这里会显示执行输出。"
                />
              ) : (
                <div className="terminal max-h-[520px] whitespace-pre-wrap">
                  {deployLog.map((line, index) => (
                    <div key={index}>{line}</div>
                  ))}
                </div>
              )}
            </TabsContent>

            <TabsContent value="history" className="p-0">
              <div className="p-4">
                <p className="text-[13px] text-fg-muted">
                  Compose 部署历史由后端持久化，v1.0 契约暂未提供该查询端点。
                </p>
                <p className="mt-2 flex items-center gap-1.5 font-mono text-[11px] text-fg-subtle">
                  <Play className="h-3 w-3" />
                  当前文件更新时间 {formatRelativeTime(currentFile?.updatedAt)}
                </p>
              </div>
            </TabsContent>
          </Tabs>
        </div>
      </div>

      <ConfirmDialog
        open={confirmDown}
        onOpenChange={setConfirmDown}
        title={`停止项目 ${project}？`}
        description="项目下的所有服务容器将被停止并移除，挂载的数据卷会保留。"
        confirmText="确认停止"
        onConfirm={() => {
          down.mutate(
            { name: project, removeVolumes: false },
            {
              onSuccess: () => {
                toast.success('项目已停止')
                setConfirmDown(false)
              },
              onError: (err) => toast.error(err instanceof Error ? err.message : '停止失败'),
            },
          )
        }}
      />
    </div>
  )
}
