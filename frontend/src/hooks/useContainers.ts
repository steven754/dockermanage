import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { containerApi, dashboardApi } from '@/lib/api'
import type { Container, ContainerStats, PruneResult } from '@/lib/types'

/* ---------------------------------------------------------------- 概览 */

export function useDashboard() {
  return useQuery({
    queryKey: ['dashboard'],
    queryFn: () => dashboardApi.get(),
    refetchInterval: 30_000,
    staleTime: 5_000,
  })
}

/* ---------------------------------------------------------------- 容器 */

export interface ContainerFilters {
  status?: string[]
  search?: string
  label?: string
}

export function useContainers(filters: ContainerFilters = {}) {
  const { status, search, label } = filters
  return useQuery({
    queryKey: ['containers', { status, search, label }],
    queryFn: () =>
      containerApi.list({
        status: status && status.length > 0 ? status.join(',') : undefined,
        search: search || undefined,
        label: label || undefined,
      }),
    refetchInterval: 15_000,
    staleTime: 3_000,
  })
}

export function useContainer(id: string | undefined) {
  return useQuery({
    queryKey: ['container', id],
    queryFn: () => containerApi.get(id as string),
    enabled: Boolean(id),
    refetchInterval: 15_000,
  })
}

export function useContainerInspect(id: string | undefined) {
  return useQuery({
    queryKey: ['container', id, 'inspect'],
    queryFn: () => containerApi.inspect(id as string),
    enabled: Boolean(id),
    staleTime: 10_000,
  })
}

export function useContainerStats(id: string | undefined) {
  return useQuery({
    queryKey: ['container', id, 'stats'],
    queryFn: () => containerApi.stats(id as string),
    enabled: Boolean(id),
    refetchInterval: 5_000,
  })
}

/** 容器生命周期操作（启停/重启/删除/更新） */
export function useContainerActions() {
  const queryClient = useQueryClient()

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ['containers'] })
    void queryClient.invalidateQueries({ queryKey: ['dashboard'] })
    void queryClient.invalidateQueries({ queryKey: ['container'] })
  }

  const action = useMutation({
    mutationFn: ({ id, action }: { id: string; action: 'start' | 'stop' | 'restart' | 'kill' | 'pause' | 'unpause' }) =>
      containerApi.action(id, action),
    onSuccess: invalidate,
  })

  const remove = useMutation({
    mutationFn: ({ id, force }: { id: string; force?: boolean }) =>
      containerApi.remove(id, force ?? true),
    onSuccess: invalidate,
  })

  const prune = useMutation<PruneResult, Error, boolean>({
    mutationFn: (all) => containerApi.prune(all ?? false),
    onSuccess: invalidate,
  })

  const update = useMutation({
    mutationFn: (id: string) => containerApi.update(id),
    onSuccess: invalidate,
  })

  return { action, remove, prune, update, invalidate }
}

export type { Container, ContainerStats, PruneResult }
