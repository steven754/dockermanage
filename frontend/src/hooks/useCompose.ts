import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { composeApi } from '@/lib/api'
import type { ComposeProject, ComposeProjectSummary } from '@/lib/types'

/** Compose 项目列表 */
export function useComposeProjects() {
  return useQuery({
    queryKey: ['compose', 'projects'],
    queryFn: () => composeApi.projects(),
    staleTime: 10_000,
  })
}

/** Compose 项目详情（含文件内容与服务列表） */
export function useComposeProject(name: string | undefined) {
  return useQuery({
    queryKey: ['compose', 'project', name],
    queryFn: () => composeApi.project(name as string),
    enabled: Boolean(name),
    staleTime: 5_000,
  })
}

/** Compose 操作：保存文件 / up / down */
export function useComposeActions() {
  const queryClient = useQueryClient()
  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ['compose'] })
    void queryClient.invalidateQueries({ queryKey: ['containers'] })
    void queryClient.invalidateQueries({ queryKey: ['dashboard'] })
  }

  const saveFile = useMutation({
    mutationFn: ({
      name,
      filename,
      content,
    }: {
      name: string
      filename: string
      content: string
    }) => composeApi.saveFile(name, filename, content),
    onSuccess: invalidate,
  })

  const up = useMutation({
    mutationFn: ({ name, services }: { name: string; services?: string[] }) =>
      composeApi.up(name, services),
    onSuccess: invalidate,
  })

  const down = useMutation({
    mutationFn: ({ name, removeVolumes }: { name: string; removeVolumes?: boolean }) =>
      composeApi.down(name, removeVolumes),
    onSuccess: invalidate,
  })

  return { saveFile, up, down }
}

export type { ComposeProject, ComposeProjectSummary }
