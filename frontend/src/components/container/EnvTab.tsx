import { useMemo } from 'react'
import { Eye, ShieldAlert } from 'lucide-react'
import { useContainerInspect } from '@/hooks/useContainers'
import { Skeleton } from '@/components/ui/input'
import { ErrorState } from '@/components/common/States'
import { cn } from '@/lib/utils'

/** 敏感变量名匹配（SPEC §8.3：默认脱敏，hover 显示明文） */
const SENSITIVE = /pass(word|wd)?|secret|token|api[_-]?key|access[_-]?key|credential|private[_-]?key|auth/i

interface EnvEntry {
  key: string
  value: string
  sensitive: boolean
}

function extractEnv(inspect: unknown): EnvEntry[] {
  if (!inspect || typeof inspect !== 'object') return []
  const config = (inspect as { Config?: { Env?: unknown } }).Config
  const env = config?.Env
  if (!Array.isArray(env)) return []
  return env
    .filter((item): item is string => typeof item === 'string')
    .map((item) => {
      const index = item.indexOf('=')
      const key = index >= 0 ? item.slice(0, index) : item
      const value = index >= 0 ? item.slice(index + 1) : ''
      return { key, value, sensitive: SENSITIVE.test(key) }
    })
    .sort((a, b) => a.key.localeCompare(b.key))
}

/** 容器详情 · 环境变量标签页 */
export function EnvTab({ containerId }: { containerId: string }) {
  const { data, isLoading, error, refetch } = useContainerInspect(containerId)
  const env = useMemo(() => extractEnv(data), [data])

  if (isLoading) {
    return (
      <div className="elev-card space-y-2 p-5">
        {Array.from({ length: 6 }).map((_, index) => (
          <Skeleton key={index} className="h-4 w-full" />
        ))}
      </div>
    )
  }

  if (error) {
    return (
      <div className="elev-card p-5">
        <ErrorState
          title="无法加载环境变量"
          message={error instanceof Error ? error.message : undefined}
          onRetry={() => void refetch()}
        />
      </div>
    )
  }

  if (env.length === 0) {
    return (
      <div className="elev-card p-5">
        <ErrorState title="该容器没有环境变量" message="Docker inspect 返回的 Config.Env 为空。" />
      </div>
    )
  }

  return (
    <div className="elev-card overflow-hidden">
      <div className="flex items-center justify-between border-b border-border px-4 py-2.5">
        <h3 className="text-sm font-semibold">
          环境变量 <span className="font-normal text-fg-subtle">· {env.length} 项</span>
        </h3>
        <span className="flex items-center gap-1.5 text-[11px] text-fg-subtle">
          <ShieldAlert className="h-3.5 w-3.5" />
          敏感值已脱敏，悬停查看明文
        </span>
      </div>
      <div className="max-h-[560px] overflow-auto">
        <table className="w-full border-collapse">
          <tbody>
            {env.map((entry) => (
              <EnvRow key={entry.key} entry={entry} />
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function EnvRow({ entry }: { entry: EnvEntry }) {
  return (
    <tr className="row-hover border-b border-border/60 last:border-b-0">
      <td className="w-64 px-4 py-2 font-mono text-[12px] text-blue-400">{entry.key}</td>
      <td className="px-4 py-2 font-mono text-[12px]">
        {entry.sensitive ? (
          <MaskedValue value={entry.value} />
        ) : (
          <span className="text-emerald-400">{entry.value || '—'}</span>
        )}
      </td>
    </tr>
  )
}

/** 脱敏值：默认打码，hover / 点击显示明文 */
function MaskedValue({ value }: { value: string }) {
  return (
    <span
      className={cn('group relative inline-flex items-center gap-1.5 text-fg-subtle')}
      title="悬停查看明文"
    >
      <span className="select-none tracking-widest">••••••••</span>
      <span className="pointer-events-none absolute left-0 z-10 hidden whitespace-nowrap rounded-md border border-border bg-bg-card px-2 py-1 text-emerald-400 shadow-md group-hover:block">
        {value || '—'}
      </span>
      <Eye className="h-3 w-3 opacity-0 transition group-hover:opacity-100" />
    </span>
  )
}
