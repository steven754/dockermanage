import { Outlet } from 'react-router-dom'
import { Topbar } from './Topbar'
import { Sidebar } from './Sidebar'
import { CommandPalette } from './CommandPalette'
import { useUiStore } from '@/store/ui'
import { cn } from '@/lib/utils'

/** 应用骨架：顶栏 + 侧栏 + 主内容 */
export function AppLayout() {
  const sidebarOpen = useUiStore((s) => s.sidebarOpen)
  const setSidebarOpen = useUiStore((s) => s.setSidebarOpen)

  return (
    <div className="min-h-screen bg-bg-base text-fg-primary">
      <Topbar />

      <div
        className={cn('sidebar-backdrop', !sidebarOpen && 'hidden')}
        onClick={() => setSidebarOpen(false)}
      />

      <div className="flex">
        <Sidebar />
        <main className="min-w-0 flex-1 p-6 lg:p-8">
          <Outlet />
        </main>
      </div>

      <CommandPalette />
    </div>
  )
}
