import { useCallback, useEffect, useRef, useState } from 'react'
import { Terminal } from '@xterm/xterm'
import { Eraser, Radio, Search } from 'lucide-react'
import '@xterm/xterm/css/xterm.css'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { useWebSocket, buildWsUrl } from '@/hooks/useWebSocket'
import { cn } from '@/lib/utils'

const THEME = {
  background: '#050505',
  foreground: '#d4d4d8',
  cursor: '#3b82f6',
  selectionBackground: 'rgba(59,130,246,.3)',
  black: '#18181b',
  red: '#ef4444',
  green: '#10b981',
  yellow: '#f59e0b',
  blue: '#3b82f6',
  magenta: '#a78bfa',
  cyan: '#06b6d4',
  white: '#e4e4e7',
}

/**
 * 容器日志。
 * 使用 xterm 渲染后端下发的 ANSI 转义序列（后端不剥离颜色），
 * 因此颜色被完整保留，且不存在 HTML 注入风险。
 */
export function LogsTab({ containerId }: { containerId: string }) {
  const containerRef = useRef<HTMLDivElement | null>(null)
  const termRef = useRef<Terminal | null>(null)
  const linesRef = useRef<string[]>([])
  const [keyword, setKeyword] = useState('')
  const [tail, setTail] = useState('200')
  const [follow, setFollow] = useState(true)

  const url = buildWsUrl(`/api/v1/containers/${encodeURIComponent(containerId)}/logs`, {
    tail: Number(tail),
    timestamps: 1,
    follow: follow ? 1 : 0,
  })

  const render = useCallback((lines: string[]) => {
    const term = termRef.current
    if (!term) return
    term.reset()
    if (lines.length === 0) return
    term.write(lines.join('\r\n'))
  }, [])

  const handleMessage = useCallback(
    (data: string) => {
      const term = termRef.current
      if (!term) return
      const lines = data.split(/\r?\n/).filter((line) => line.length > 0)
      if (lines.length === 0) return
      const next = [...linesRef.current, ...lines]
      linesRef.current = next.length > 5000 ? next.slice(next.length - 5000) : next
      const kw = keyword.trim().toLowerCase()
      if (kw) {
        render(linesRef.current.filter((line) => line.toLowerCase().includes(kw)))
      } else {
        term.write(lines.join('\r\n'))
      }
    },
    [keyword, render],
  )

  const { status, reconnect } = useWebSocket({
    url,
    enabled: Boolean(containerId),
    onMessage: handleMessage,
  })

  // 初始化 xterm
  useEffect(() => {
    const node = containerRef.current
    if (!node) return
    const term = new Terminal({
      convertEol: true,
      cursorBlink: true,
      disableStdin: true,
      fontSize: 12.5,
      fontFamily: "'JetBrains Mono', ui-monospace, SFMono-Regular, Menlo, monospace",
      lineHeight: 1.45,
      scrollback: 5000,
      theme: THEME,
    })
    term.open(node)
    termRef.current = term
    return () => {
      term.dispose()
      termRef.current = null
    }
  }, [])

  // 关键字变化时重新过滤
  useEffect(() => {
    const kw = keyword.trim().toLowerCase()
    if (!kw) {
      render(linesRef.current)
      return
    }
    render(linesRef.current.filter((line) => line.toLowerCase().includes(kw)))
  }, [keyword, render])

  const clear = () => {
    linesRef.current = []
    termRef.current?.clear()
  }

  return (
    <div className="elev-card overflow-hidden">
      <div className="flex flex-wrap items-center gap-2 border-b border-border p-3">
        <div className="relative min-w-[200px] flex-1">
          <Search className="absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-fg-subtle" />
          <Input
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            placeholder="过滤日志内容…"
            className="h-8 pl-8"
          />
        </div>
        <Select value={tail} onValueChange={setTail}>
          <SelectTrigger className="h-8 w-28">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="100">最近 100 行</SelectItem>
            <SelectItem value="200">最近 200 行</SelectItem>
            <SelectItem value="500">最近 500 行</SelectItem>
            <SelectItem value="1000">最近 1000 行</SelectItem>
          </SelectContent>
        </Select>
        <Button
          variant={follow ? 'default' : 'ghost'}
          size="sm"
          onClick={() => setFollow((value) => !value)}
          title="跟随输出"
        >
          <Radio className="h-3.5 w-3.5" />
          跟随
        </Button>
        <Button variant="ghost" size="sm" onClick={clear}>
          <Eraser className="h-3.5 w-3.5" />
          清屏
        </Button>
        <span
          className={cn(
            'inline-flex items-center gap-1.5 text-[11px]',
            status === 'open' ? 'text-success' : 'text-fg-subtle',
          )}
        >
          <span
            className={cn('h-1.5 w-1.5 rounded-full', status === 'open' ? 'dot-running' : 'dot-stopped')}
          />
          {status === 'open' ? '已连接' : status === 'connecting' ? '连接中' : '未连接'}
        </span>
        {status === 'error' || status === 'closed' ? (
          <Button variant="outline" size="sm" onClick={reconnect}>
            重连
          </Button>
        ) : null}
      </div>
      <div ref={containerRef} className="h-[420px] overflow-auto bg-[#050505] p-3" />
    </div>
  )
}
