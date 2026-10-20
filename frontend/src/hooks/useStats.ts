import { useQuery } from '@tanstack/react-query'
import { statsApi } from '@/lib/api'
import type { StatsRange } from '@/lib/types'

/** 指标历史（1h / 24h / 7d） */
export function useStatsHistory(range: StatsRange, containerId?: string) {
  return useQuery({
    queryKey: ['stats', 'history', range, containerId ?? 'system'],
    queryFn: () => statsApi.history(range, containerId),
    staleTime: 30_000,
    refetchInterval: 60_000,
  })
}
