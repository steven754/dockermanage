import { Link, useNavigate } from 'react-router-dom'
import { Bell, Container, Download, Menu, Search, Settings as SettingsIcon } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useUiStore } from '@/store/ui'
import { useAuth } from '@/hooks/useAuth'

/** 顶栏：品牌 + ⌘K 触发器 + 快捷入口 + 用户 */
export function Topbar() {
  const navigate = useNavigate()
  const { user } = useAuth()
  const setCommandPaletteOpen = useUiStore((s) => s.setCommandPaletteOpen)
  const toggleSidebar = useUiStore((s) => s.toggleSidebar)

  return (
    <header className="topbar flex items-center gap-4 px-4">
      <Button
        variant="ghost"
        size="icon-sm"
        className="-ml-1 shrink-0 lg:hidden"
        onClick={toggleSidebar}
        aria-label="打开菜单"
      >
        <Menu className="h-4 w-4" />
      </Button>

      <Link to="/" className="flex items-center gap-2.5">
        <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-gradient-to-br from-brand to-cyan-400 shadow-glow-blue">
          <Container className="h-[18px] w-[18px] text-white" />
        </div>
        <span className="font-semibold tracking-tight">Dockermanage</span>
        <span className="hidden text-xs text-fg-subtle sm:inline">v0.1 · single-host</span>
      </Link>

      <button
        type="button"
        onClick={() => setCommandPaletteOpen(true)}
        className="ml-4 flex items-center gap-2 rounded-md border border-border bg-bg-card px-3 py-1.5 text-sm text-fg-muted transition hover:border-border-strong sm:max-w-xl sm:flex-1"
      >
        <Search className="h-3.5 w-3.5 shrink-0" />
        <span className="hidden min-w-0 flex-1 truncate text-left sm:block">
          搜索容器、镜像、卷、网络，或执行命令…
        </span>
        <span className="kbd shrink-0">⌘</span>
        <span className="kbd shrink-0">K</span>
      </button>

      <div className="ml-auto flex items-center gap-1">
        <Button
          variant="ghost"
          size="icon-sm"
          title="Pull 镜像"
          onClick={() => navigate('/images?action=pull')}
        >
          <Download className="h-4 w-4" />
        </Button>
        <Button variant="ghost" size="icon-sm" title="通知" className="relative">
          <Bell className="h-4 w-4" />
          <span className="absolute right-1.5 top-1.5 h-1.5 w-1.5 rounded-full bg-red-500" />
        </Button>
        <Button
          variant="ghost"
          size="icon-sm"
          title="设置"
          onClick={() => navigate('/settings')}
        >
          <SettingsIcon className="h-4 w-4" />
        </Button>
        <div className="mx-1 h-5 w-px bg-border" />
        <div className="flex cursor-pointer items-center gap-2 rounded-md py-1 pl-1 pr-2 hover:bg-bg-card">
          <div className="flex h-6 w-6 items-center justify-center rounded-full bg-gradient-to-br from-emerald-400 to-blue-500 text-[11px] font-semibold text-white">
            {(user?.username ?? 'U').slice(0, 1).toUpperCase()}
          </div>
          <span className="hidden text-sm sm:inline">{user?.username ?? '未登录'}</span>
        </div>
      </div>
    </header>
  )
}
