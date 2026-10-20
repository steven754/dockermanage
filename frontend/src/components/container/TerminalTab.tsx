import { useEffect, useRef, useState } from 'react'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import { Maximize2, TerminalSquare } from 'lucide-react'
import '@xterm/xterm/css/xterm.css'
import { Button } from '@/components/ui/button'
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

/** 可选 shell */
const SHELLS = [
  { value: '/bin/sh', label: '/bin/sh' },
  { value: '/bin/bash', label: '/bin/bash' },
  { value: '/bin/zsh', label: '/bin/zsh' },
  { value: '/bin/ash', label: '/bin/ash' },
]

/** 容器详情 · Terminal 标签页（xterm.js + WebSocket exec） */
export function TerminalTab({ containerId }: { containerId: string }) {
  const containerRef = useRef<HTMLDivElement | null>(null)
  const termRef = useRef<Terminal | null>(null)
  const fitRef = useRef<FitAddon | null>(null)
  const [cmd, setCmd] = useState('/bin/sh')
  const [fullscreen, setFullscreen] = useState(false)

  const url = buildWsUrl(`/api/v1/containers/${encodeURIComponent(containerId)}/exec`, {
    cmd,
    tty: 1,
  })

  const { status, send, reconnect } = useWebSocket({
    url,
    enabled: Boolean(containerId),
    onMessage: (data) => termRef.current?.write(data),
  })

  // 初始化 xterm + Fit + WebLinks
  useEffect(() => {
    const node = containerRef.current
    if (!node) return
    const term = new Terminal({
      cursorBlink: true,
      fontSize: 12.5,
      fontFamily: "'JetBrains Mono', ui-monospace, SFMono-Regular, Menlo, monospace",
      lineHeight: 1.45,
      scrollback: 5000,
      theme: THEME,
    })
    const fit = new FitAddon()
    term.loadAddon(fit)
    term.loadAddon(new WebLinksAddon())
    term.open(node)

    const inputDisposable = term.onData((data) => send(data))
    const resizeDisposable = term.onResize(({ cols, rows }) => {
      send(JSON.stringify({ type: 'resize', cols, rows }))
    })

    try {
      fit.fit()
    } catch {
      /* 节点尚未布局完成时忽略 */
    }

    termRef.current = term
    fitRef.current = fit

    return () => {
      inputDisposable.dispose()
      resizeDisposable.dispose()
      term.dispose()
      termRef.current = null
      fitRef.current = null
    }
  }, [send])

  // 容器尺寸变化时自适应
  useEffect(() => {
    const node = containerRef.current
    if (!node || typeof ResizeObserver === 'undefined') return
    const observer = new ResizeObserver(() => {
      try {
        fitRef.current?.fit()
      } catch {
        /* ignore */
      }
    })
    observer.observe(node)
    return () => observer.disconnect()
  }, [])

  return (
    <div
      className={cn(
        'elev-card overflow-hidden',
        fullscreen && 'fixed inset-0 z-50 rounded-none',
      )}
    >
      <div className="flex flex-wrap items-center gap-2 border-b border-border p-3">
        <TerminalSquare className="h-4 w-4 text-fg-subtle" />
        <Select value={cmd} onValueChange={setCmd}>
          <SelectTrigger className="h-8 w-40">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {SHELLS.map((shell) => (
              <SelectItem key={shell.value} value={shell.value}>
                {shell.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <span
          className={cn(
            'inline-flex items-center gap-1.5 text-[11px]',
            status === 'open' ? 'text-success' : 'text-fg-subtle',
          )}
        >
          <span
            className={cn('h-1.5 w-1.5 rounded-full', status === 'open' ? 'dot-running' : 'dot-stopped')}
          />
          {status === 'open' ? '会话已建立' : status === 'connecting' ? '连接中…' : '未连接'}
        </span>
        {status === 'error' || status === 'closed' ? (
          <Button variant="outline" size="sm" onClick={reconnect}>
            重连
          </Button>
        ) : null}
        <Button
          variant="ghost"
          size="sm"
          className="ml-auto"
          onClick={() => setFullscreen((value) => !value)}
        >
          <Maximize2 className="h-3.5 w-3.5" />
          {fullscreen ? '退出全屏' : '全屏'}
        </Button>
      </div>
      <div
        ref={containerRef}
        className={cn('overflow-auto bg-[#050505] p-3', fullscreen ? 'h-[calc(100vh-90px)]' : 'h-[460px]')}
      />
    </div>
  )
}
