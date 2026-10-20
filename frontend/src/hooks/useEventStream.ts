import { useCallback, useEffect, useMemo, useState } from 'react'
import { useSSE } from './useSSE'
import { useEvents } from './useEvents'
import type { DockerEvent } from '@/lib/types'

/**
 * 事件流：历史（REST）+ 实时（SSE `/api/v1/events/stream`）合并。
 * 返回按时间倒序排列的事件列表。
 */
export function useEventStream(limit = 100) {
  const { data, isLoading, error, refetch } = useEvents(limit)
  const [live, setLive] = useState<DockerEvent[]>([])

  const handleMessage = useCallback((payload: unknown) => {
    const evt = payload as DockerEvent
    if (!evt || typeof evt !== 'object' || !evt.type) return
    setLive((prev) => [evt, ...prev].slice(0, limit))
  }, [limit])

  const { status } = useSSE<DockerEvent>('/api/v1/events/stream', {
    events: ['event'],
    onMessage: handleMessage,
  })

  // 历史刷新时丢弃已入库的实时事件，避免重复
  useEffect(() => {
    if (!data) return
    setLive((prev) => {
      if (prev.length === 0) return prev
      const known = new Set(data.items.map((item) => item.id))
      return prev.filter((item) => !known.has(item.id))
    })
  }, [data])

  /**
   * 合并实时与历史事件：
   *  - 按 id 去重（id 缺失时不做去重，避免丢事件）
   *  - 实时事件在前（按时间倒序）
   */
  const events = useMemo(() => {
    const seen = new Set<number>()
    const merged: DockerEvent[] = []
    const push = (event: DockerEvent) => {
      if (typeof event.id === 'number' && event.id > 0) {
        if (seen.has(event.id)) return
        seen.add(event.id)
      }
      merged.push(event)
    }
    live.forEach(push)
    ;(data?.items ?? []).forEach(push)
    return merged.slice(0, limit)
  }, [live, data, limit])

  return {
    events,
    isLoading,
    error,
    refetch,
    connected: status === 'open',
  }
}
