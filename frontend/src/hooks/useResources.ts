import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { imageApi, networkApi, volumeApi } from '@/lib/api'
import type { PruneResult } from '@/lib/types'

/* ---------------------------------------------------------------- 镜像 */

export function useImages() {
  return useQuery({
    queryKey: ['images'],
    queryFn: () => imageApi.list(),
    staleTime: 10_000,
  })
}

export function useImageActions() {
  const queryClient = useQueryClient()
  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ['images'] })
    void queryClient.invalidateQueries({ queryKey: ['dashboard'] })
  }

  const pull = useMutation({
    mutationFn: (image: string) => imageApi.pull(image),
    onSuccess: invalidate,
  })
  const remove = useMutation({
    mutationFn: ({ id, force }: { id: string; force?: boolean }) => imageApi.remove(id, force),
    onSuccess: invalidate,
  })
  const prune = useMutation<PruneResult, Error, boolean>({
    mutationFn: (dangling) => imageApi.prune(dangling ?? true),
    onSuccess: invalidate,
  })
  const tag = useMutation({
    mutationFn: ({ id, repo, tag }: { id: string; repo: string; tag: string }) =>
      imageApi.tag(id, repo, tag),
    onSuccess: invalidate,
  })

  return { pull, remove, prune, tag }
}

/* ---------------------------------------------------------------- 卷 */

export function useVolumes() {
  return useQuery({
    queryKey: ['volumes'],
    queryFn: () => volumeApi.list(),
    staleTime: 15_000,
  })
}

export function useVolumeActions() {
  const queryClient = useQueryClient()
  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ['volumes'] })
    void queryClient.invalidateQueries({ queryKey: ['dashboard'] })
  }
  const remove = useMutation({
    mutationFn: ({ name, force }: { name: string; force?: boolean }) => volumeApi.remove(name, force),
    onSuccess: invalidate,
  })
  const prune = useMutation({
    mutationFn: () => volumeApi.prune(),
    onSuccess: invalidate,
  })
  return { remove, prune }
}

/* ---------------------------------------------------------------- 网络 */

export function useNetworks() {
  return useQuery({
    queryKey: ['networks'],
    queryFn: () => networkApi.list(),
    staleTime: 15_000,
  })
}

export function useNetworkActions() {
  const queryClient = useQueryClient()
  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ['networks'] })
    void queryClient.invalidateQueries({ queryKey: ['dashboard'] })
  }
  const remove = useMutation({
    mutationFn: (id: string) => networkApi.remove(id),
    onSuccess: invalidate,
  })
  return { remove }
}
