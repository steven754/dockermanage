import { Link } from 'react-router-dom'
import { Compass } from 'lucide-react'
import { Button } from '@/components/ui/button'

/** 404 页面 */
export function NotFound() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-4 bg-bg-base text-fg-primary">
      <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-bg-muted text-fg-subtle">
        <Compass className="h-6 w-6" />
      </div>
      <div className="text-center">
        <p className="text-lg font-semibold">页面不存在</p>
        <p className="mt-1 text-[13px] text-fg-muted">你访问的路径没有对应的页面</p>
      </div>
      <Button asChild variant="primary" size="sm">
        <Link to="/">返回 Dashboard</Link>
      </Button>
    </div>
  )
}
