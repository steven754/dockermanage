import { useEffect, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { Command } from 'cmdk'
import {
  Box,
  Container,
  Database,
  FileCode2,
  Image as ImageIcon,
  Layers,
  LayoutDashboard,
  Network,
  Play,
  RefreshCcw,
  Search,
  Square,
  Trash2,
} from 'lucide-react'
import { useUiStore } from '@/store/ui'
import { useContainers, useContainerActions } from '@/hooks/useContainers'
import { containerApi } from '@/lib/api'
import { StateBadge } from '@/components/common/StatusBadge'
import { isContainerRunning, isContainerStopped } from '@/lib/format'
import { ContainerIcon } from '@/components/common/ContainerIcon'
import { cn } from '@/lib/utils'

/** ⌘K 命令面板：全局搜索（容器 / 页面）+ 快捷操作 */
export function CommandPalette() {
  const navigate = useNavigate()
  const open = useUiStore((s) => s.commandPaletteOpen)
  const setOpen = useUiStore((s) => s.setCommandPaletteOpen)
  const { data: containersData } = useContainers()
  const { prune } = useContainerActions()

  const containers = useMemo(() => containersData?.items ?? [], [containersData])

  // ⌘K / Ctrl+K 打开，ESC 关闭（cmdk 内置 ESC 关闭，这里只管快捷键唤起）
  useEffect(() => {
    const handler = (event: KeyboardEvent) => {
      if (event.key === 'k' && (event.metaKey || event.ctrlKey)) {
        event.preventDefault()
        setOpen(true)
      }
    }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [setOpen])

  const run = (fn: () => void) => {
    setOpen(false)
    // 等待面板关闭动画后再执行，避免焦点冲突
    window.setTimeout(fn, 60)
  }

  const startAll = () => {
    const stopped = containers.filter((c) => isContainerStopped(c.state, c.exitCode))
    if (stopped.length === 0) return
    void Promise.allSettled(stopped.map((c) => containerApi.action(c.id, 'start')))
  }

  const stopAll = () => {
    const running = containers.filter((c) => c.state === 'running')
    if (running.length === 0) return
    void Promise.allSettled(running.map((c) => containerApi.action(c.id, 'stop')))
  }

  const updateAll = () => {
    const updatable = containers.filter((c) => c.state === 'running')
    if (updatable.length === 0) return
    void Promise.allSettled(updatable.map((c) => containerApi.update(c.id)))
  }

  const pages = [
    { label: 'Dashboard', to: '/', icon: LayoutDashboard, hint: '概览' },
    { label: '容器列表', to: '/containers', icon: Container, hint: '资源' },
    { label: '镜像列表', to: '/images', icon: ImageIcon, hint: '资源' },
    { label: '卷列表', to: '/volumes', icon: Database, hint: '资源' },
    { label: '网络列表', to: '/networks', icon: Network, hint: '资源' },
    { label: 'Compose 项目', to: '/compose', icon: FileCode2, hint: '资源' },
    { label: '事件流', to: '/events', icon: Layers, hint: '概览' },
    { label: '设置', to: '/settings', icon: Box, hint: '系统' },
  ]

  return (
    <div
      className={cn('cmdk-overlay', !open && 'hidden')}
      onClick={(event) => {
        if (event.target === event.currentTarget) setOpen(false)
      }}
    >
      <Command
        label="命令面板"
        className="cmdk-panel"
        onKeyDown={(event) => {
          if (event.key === 'Escape') setOpen(false)
        }}
      >
        <div className="cmdk-input">
          <Search className="h-4 w-4 text-fg-muted" />
          <Command.Input
            autoFocus
            placeholder="输入命令、搜索容器或跳转…"
            className="w-full bg-transparent text-[15px] text-fg-primary outline-none placeholder:text-fg-subtle"
          />
          <span className="kbd">ESC</span>
        </div>

        <Command.List className="cmdk-list">
          <Command.Empty className="px-3 py-6 text-center text-[13px] text-fg-subtle">
            没有匹配的结果
          </Command.Empty>

          <Command.Group
            heading="建议"
            className="[&_[cmdk-group-heading]]:px-3 [&_[cmdk-group-heading]]:pb-1 [&_[cmdk-group-heading]]:pt-3 [&_[cmdk-group-heading]]:text-[11px] [&_[cmdk-group-heading]]:uppercase [&_[cmdk-group-heading]]:tracking-wider [&_[cmdk-group-heading]]:text-fg-subtle"
          >
            <Command.Item
              value="启动全部容器"
              onSelect={() =>
                run(() => {
                  if (window.confirm('确认启动所有已停止的容器？')) startAll()
                })
              }
              className="cmdk-item"
            >
              <div className="ci-icon">
                <Play className="h-4 w-4 text-emerald-400" />
              </div>
              <div className="min-w-0 flex-1">
                <div className="text-sm">启动全部</div>
                <div className="text-[11px] text-fg-subtle">
                  {containers.filter((c) => isContainerStopped(c.state, c.exitCode)).length} 个已停止
                </div>
              </div>
              <div className="cmdk-shortcut">
                <span className="kbd">↵</span>
              </div>
            </Command.Item>

            <Command.Item
              value="停止全部容器"
              onSelect={() =>
                run(() => {
                  if (window.confirm('确认停止所有运行中的容器？')) stopAll()
                })
              }
              className="cmdk-item"
            >
              <div className="ci-icon">
                <Square className="h-4 w-4 text-amber-400" />
              </div>
              <div className="min-w-0 flex-1">
                <div className="text-sm">停止全部</div>
                <div className="text-[11px] text-fg-subtle">谨慎操作</div>
              </div>
            </Command.Item>

            <Command.Item
              value="清理资源 prune"
              onSelect={() =>
                run(() => {
                  if (window.confirm('确认清理所有已停止的容器？')) prune.mutate(false)
                })
              }
              className="cmdk-item"
            >
              <div className="ci-icon">
                <Trash2 className="h-4 w-4 text-red-400" />
              </div>
              <div className="min-w-0 flex-1">
                <div className="text-sm">清理资源</div>
                <div className="text-[11px] text-fg-subtle">删除已停止的容器</div>
              </div>
            </Command.Item>

            <Command.Item
              value="一键更新全部镜像"
              onSelect={() =>
                run(() => {
                  if (window.confirm('确认拉取新镜像并重建所有运行中的容器？')) updateAll()
                })
              }
              className="cmdk-item"
            >
              <div className="ci-icon">
                <RefreshCcw className="h-4 w-4 text-brand" />
              </div>
              <div className="min-w-0 flex-1">
                <div className="text-sm">更新所有镜像</div>
                <div className="text-[11px] text-fg-subtle">
                  {containers.filter((c) => isContainerRunning(c.state)).length} 个容器可更新
                </div>
              </div>
              <div className="cmdk-shortcut">
                <span className="kbd">⌘</span>
                <span className="kbd">U</span>
              </div>
            </Command.Item>
          </Command.Group>

          <Command.Group
            heading="容器"
            className="[&_[cmdk-group-heading]]:px-3 [&_[cmdk-group-heading]]:pb-1 [&_[cmdk-group-heading]]:pt-3 [&_[cmdk-group-heading]]:text-[11px] [&_[cmdk-group-heading]]:uppercase [&_[cmdk-group-heading]]:tracking-wider [&_[cmdk-group-heading]]:text-fg-subtle"
          >
            {containers.slice(0, 50).map((container) => (
              <Command.Item
                key={container.id}
                value={`container ${container.name} ${container.image}`}
                onSelect={() => run(() => navigate(`/containers/${container.id}`))}
                className="cmdk-item"
              >
                <div className="ci-icon">
                  <ContainerIcon
                    image={container.image}
                    name={container.name}
                    className="h-7 w-7"
                    iconClassName="h-3.5 w-3.5"
                  />
                </div>
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <span className="truncate text-sm">{container.name}</span>
                    <StateBadge state={container.state} exitCode={container.exitCode} />
                  </div>
                  <div className="truncate font-mono text-[11px] text-fg-subtle">
                    {container.image}
                  </div>
                </div>
              </Command.Item>
            ))}
          </Command.Group>

          <Command.Group
            heading="页面"
            className="[&_[cmdk-group-heading]]:px-3 [&_[cmdk-group-heading]]:pb-1 [&_[cmdk-group-heading]]:pt-3 [&_[cmdk-group-heading]]:text-[11px] [&_[cmdk-group-heading]]:uppercase [&_[cmdk-group-heading]]:tracking-wider [&_[cmdk-group-heading]]:text-fg-subtle"
          >
            {pages.map((page) => {
              const Icon = page.icon
              return (
                <Command.Item
                  key={page.to}
                  value={`page ${page.label} ${page.hint}`}
                  onSelect={() => run(() => navigate(page.to))}
                  className="cmdk-item"
                >
                  <div className="ci-icon">
                    <Icon className="h-4 w-4" />
                  </div>
                  <div className="min-w-0 flex-1">
                    <div className="text-sm">{page.label}</div>
                    <div className="text-[11px] text-fg-subtle">{page.hint}</div>
                  </div>
                </Command.Item>
              )
            })}
          </Command.Group>
        </Command.List>
      </Command>
    </div>
  )
}
