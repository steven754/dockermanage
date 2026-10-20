import { useCallback, useEffect, useRef, useState } from 'react'
import { websocketUrl } from '@/lib/api'

/**
 * 通用 WebSocket hook：自动重连（指数退避）+ 组件卸载时清理。
 *
 * 浏览器 WebSocket 同样无法自定义请求头，token 以 `access_token`
 * 查询参数附带（后端需支持该约定）。
 */

export type WsStatus = 'idle' | 'connecting' | 'open' | 'closed' | 'error'

export interface UseWebSocketOptions {
  /** 完整 ws:// URL；null 表示不连接 */
  url: string | null
  /** 是否启用 */
  enabled?: boolean
  /** 收到文本帧 */
  onMessage?: (data: string) => void
  /** 收到二进制帧 */
  onBinary?: (data: ArrayBuffer) => void
  onOpen?: () => void
  onClose?: () => void
  /** 最大重连间隔（毫秒） */
  maxRetryDelay?: number
}

export interface UseWebSocketResult {
  status: WsStatus
  send: (data: string) => void
  close: () => void
  reconnect: () => void
}

export function useWebSocket(options: UseWebSocketOptions): UseWebSocketResult {
  const { url, enabled = true, onMessage, onBinary, onOpen, onClose, maxRetryDelay = 15000 } = options

  const [status, setStatus] = useState<WsStatus>('idle')

  const socketRef = useRef<WebSocket | null>(null)
  const retryRef = useRef(0)
  const timerRef = useRef<number | null>(null)
  const disposedRef = useRef(false)
  const manualCloseRef = useRef(false)

  const onMessageRef = useRef(onMessage)
  const onBinaryRef = useRef(onBinary)
  const onOpenRef = useRef(onOpen)
  const onCloseRef = useRef(onClose)
  onMessageRef.current = onMessage
  onBinaryRef.current = onBinary
  onOpenRef.current = onOpen
  onCloseRef.current = onClose

  const clearTimer = useCallback(() => {
    if (timerRef.current !== null) {
      window.clearTimeout(timerRef.current)
      timerRef.current = null
    }
  }, [])

  const connect = useCallback(() => {
    if (!url || !enabled || disposedRef.current) return
    manualCloseRef.current = false
    setStatus('connecting')

    let socket: WebSocket
    try {
      socket = new WebSocket(url)
    } catch {
      setStatus('error')
      scheduleRetry()
      return
    }
    socketRef.current = socket

    socket.onopen = () => {
      retryRef.current = 0
      setStatus('open')
      onOpenRef.current?.()
    }

    socket.onmessage = (event: MessageEvent) => {
      if (typeof event.data === 'string') {
        onMessageRef.current?.(event.data)
      } else if (event.data instanceof Blob) {
        void event.data.arrayBuffer().then((buf) => onBinaryRef.current?.(buf))
      } else if (event.data instanceof ArrayBuffer) {
        onBinaryRef.current?.(event.data)
      }
    }

    socket.onerror = () => {
      setStatus('error')
    }

    socket.onclose = () => {
      setStatus('closed')
      onCloseRef.current?.()
      if (!manualCloseRef.current && !disposedRef.current) scheduleRetry()
    }

    function scheduleRetry() {
      if (disposedRef.current || manualCloseRef.current) return
      clearTimer()
      const delay = Math.min(1000 * 2 ** retryRef.current, maxRetryDelay)
      retryRef.current += 1
      timerRef.current = window.setTimeout(() => connect(), delay)
    }
  }, [url, enabled, clearTimer, maxRetryDelay])

  useEffect(() => {
    disposedRef.current = false
    if (!enabled || !url) {
      setStatus('idle')
      return
    }
    connect()
    return () => {
      disposedRef.current = true
      manualCloseRef.current = true
      clearTimer()
      const socket = socketRef.current
      socketRef.current = null
      if (socket) {
        socket.onclose = null
        socket.close()
      }
    }
  }, [url, enabled, connect, clearTimer])

  const send = useCallback((data: string) => {
    const socket = socketRef.current
    if (socket && socket.readyState === WebSocket.OPEN) socket.send(data)
  }, [])

  const close = useCallback(() => {
    manualCloseRef.current = true
    clearTimer()
    const socket = socketRef.current
    socketRef.current = null
    if (socket) {
      socket.onclose = null
      socket.close()
    }
    setStatus('closed')
  }, [clearTimer])

  const reconnect = useCallback(() => {
    close()
    disposedRef.current = false
    manualCloseRef.current = false
    retryRef.current = 0
    connect()
  }, [close, connect])

  return { status, send, close, reconnect }
}

/** 构造带鉴权参数的 WebSocket URL */
export function buildWsUrl(path: string, params?: Record<string, string | number | undefined>): string {
  const merged: Record<string, string | number | undefined> = { ...params }
  const token = (() => {
    try {
      return localStorage.getItem('dm.accessToken')
    } catch {
      return null
    }
  })()
  if (token) merged.access_token = token
  return websocketUrl(path, merged)
}
