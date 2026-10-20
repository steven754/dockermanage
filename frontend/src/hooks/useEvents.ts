import { useQuery } from '@tanstack/react-query'
import { eventApi } from '@/lib/api'
import type { DockerEvent } from '@/lib/types'

/** 事件历史（最近 100 条） */
export function useEvents(limit = 100) {
  return useQuery({
    queryKey: ['events'],
    queryFn: () => eventApi.history({ limit }),
    staleTime: 5_000,
    refetchInterval: 60_000,
  })
}

/** 事件流 SSE 消息 */
export type EventStreamMessage = DockerEvent
