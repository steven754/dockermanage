import { useEffect, useRef, useState } from 'react'
import { absoluteUrl, authQueryParams, buildQuery, shouldUseCredentials } from '@/lib/api'

/**
 * 通用 SSE hook：EventSource + 自动重连 + 组件卸载时清理。
 *
 * 浏览器 EventSource 无法自定义请求头，因此 token 以 `access_token`
 * 查询参数附带（后端需支持该约定），详见报告中的「偏差」一节。
 */

export type SseStatus = 'idle' | 'connecting' | 'open' | 'error'

export interface UseSSEOptions<T> {
  /** 订阅的事件名列表；缺省订阅默认 `message` 事件 */
  events?: string[]
  /** 是否启用；false 时不建立连接 */
  enabled?: boolean
  /** 收到消息回调（已 JSON.parse 失败时降级为原始字符串） */
  onMessage: (data: T, eventName: string) => void
  /** 连接错误回调 */
  onError?: (message: string) => void
  /** 连接成功回调 */
  onOpen?: () => void
}

export interface UseSSEResult {
  status: SseStatus
  /** 最近一次收到消息的时间 */
  lastEventAt: Date | null
  error: string | null
}

export function useSSE<T>(path: string, options: UseSSEOptions<T>): UseSSEResult {
  const { events = ['message'], enabled = true, onMessage, onError, onOpen } = options

  const [status, setStatus] = useState<SseStatus>('idle')
  const [lastEventAt, setLastEventAt] = useState<Date | null>(null)
  const [error, setError] = useState<string | null>(null)

  // 用 ref 固定回调，避免因回调变化重建连接
  const onMessageRef = useRef(onMessage)
  const onErrorRef = useRef(onError)
  const onOpenRef = useRef(onOpen)
  onMessageRef.current = onMessage
  onErrorRef.current = onError
  onOpenRef.current = onOpen

  useEffect(() => {
    if (!enabled || !path) {
      setStatus('idle')
      return
    }

    const url = `${absoluteUrl(path)}${buildQuery(authQueryParams())}`
    let source: EventSource | null = null
    let disposed = false

    const handle = (raw: string, eventName: string) => {
      if (disposed) return
      let payload: T
      try {
        payload = JSON.parse(raw) as T
      } catch {
        payload = raw as unknown as T
      }
      setLastEventAt(new Date())
      onMessageRef.current(payload, eventName)
    }

    try {
      // 跨域时不带 credentials，避免与 `Access-Control-Allow-Origin: *` 冲突
      source = new EventSource(url, { withCredentials: shouldUseCredentials() })
    } catch (err) {
      const message = err instanceof Error ? err.message : '无法建立 SSE 连接'
      setStatus('error')
      setError(message)
      onErrorRef.current?.(message)
      return
    }

    setStatus('connecting')

    source.onopen = () => {
      if (disposed) return
      setStatus('open')
      setError(null)
      onOpenRef.current?.()
    }

    source.onerror = () => {
      if (disposed) return
      // EventSource 自带重连机制，这里只更新状态
      setStatus('error')
      setError('实时连接已断开，正在重连…')
      onErrorRef.current?.('实时连接已断开，正在重连…')
    }

    const listeners = events.map((name) => {
      const listener = (event: MessageEvent) => handle(event.data as string, name)
      source?.addEventListener(name, listener as EventListener)
      return { name, listener }
    })

    return () => {
      disposed = true
      listeners.forEach(({ name, listener }) => {
        source?.removeEventListener(name, listener as EventListener)
      })
      source?.close()
      source = null
    }
  }, [path, enabled, events.join(',')]) // eslint-disable-line react-hooks/exhaustive-deps

  return { status, lastEventAt, error }
}
