import { useCallback, useEffect, useRef, useState } from 'react'
import { useSSE } from './useSSE'
import type { StreamedContainerStats, SystemStats } from '@/lib/types'

/**
 * 主机 / 容器实时指标（SSE `/api/v1/stats/stream`）。
 *
 * 后端每 10 秒推一条 `system` 事件与若干条 `container` 事件；
 * 这里在客户端保留一个滚动缓冲区，供 Dashboard 实时折线图使用。
 */

export interface LiveStats {
  system: SystemStats | null
  containers: Record<string, StreamedContainerStats>
  /** 每个容器的指标时间序列（时间升序） */
  containerBuffer: Record<string, StreamedContainerStats[]>
  /** 最近的系统指标（时间升序） */
  buffer: SystemStats[]
  connected: boolean
  lastEventAt: Date | null
}

const MAX_BUFFER = 400 // 约 66 分钟（10 秒采样）
const MAX_CONTAINER_BUFFER = 200

export function useLiveStats(enabled = true): LiveStats {
  const [system, setSystem] = useState<SystemStats | null>(null)
  const [containers, setContainers] = useState<Record<string, StreamedContainerStats>>({})
  const [containerBuffer, setContainerBuffer] = useState<Record<string, StreamedContainerStats[]>>({})
  const [buffer, setBuffer] = useState<SystemStats[]>([])
  const seenRef = useRef<Set<string>>(new Set())

  const handleMessage = useCallback((payload: unknown, eventName: string) => {
    if (eventName === 'system') {
      const stat = payload as SystemStats
      if (!stat || typeof stat.cpuPct !== 'number') return
      setSystem(stat)
      setBuffer((prev) => {
        const next = [...prev, stat]
        return next.length > MAX_BUFFER ? next.slice(next.length - MAX_BUFFER) : next
      })
      return
    }
    if (eventName === 'container') {
      const stat = payload as StreamedContainerStats
      if (!stat || !stat.id) return
      setContainers((prev) => ({ ...prev, [stat.id]: stat }))
      setContainerBuffer((prev) => {
        const list = prev[stat.id] ?? []
        const next = [...list, stat]
        return {
          ...prev,
          [stat.id]:
            next.length > MAX_CONTAINER_BUFFER
              ? next.slice(next.length - MAX_CONTAINER_BUFFER)
              : next,
        }
      })
    }
  }, [])

  const { status, lastEventAt } = useSSE<SystemStats | StreamedContainerStats>(
    '/api/v1/stats/stream',
    {
      events: ['system', 'container'],
      enabled,
      onMessage: handleMessage,
    },
  )

  // 容器停止后从实时表里移除，避免列表里残留
  useEffect(() => {
    const timer = window.setInterval(() => {
      if (seenRef.current.size === 0) return
      setContainers((prev) => {
        const next: Record<string, StreamedContainerStats> = {}
        for (const [id, stat] of Object.entries(prev)) {
          if (seenRef.current.has(id)) next[id] = stat
        }
        return next
      })
    }, 30_000)
    return () => window.clearInterval(timer)
  }, [])

  return {
    system,
    containers,
    containerBuffer,
    buffer,
    connected: status === 'open',
    lastEventAt,
  }
}
