import { useState } from 'react'
import { ChevronDown, ChevronRight } from 'lucide-react'
import { useContainerInspect } from '@/hooks/useContainers'
import { Skeleton } from '@/components/ui/input'
import { ErrorState } from '@/components/common/States'
import { Button } from '@/components/ui/button'

type JsonValue =
  | string
  | number
  | boolean
  | null
  | JsonValue[]
  | { [key: string]: JsonValue }

function isPlainObject(value: JsonValue): value is { [key: string]: JsonValue } {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function valueColor(value: JsonValue): string {
  if (typeof value === 'string') return 'text-emerald-400'
  if (typeof value === 'number') return 'text-amber-400'
  if (typeof value === 'boolean') return 'text-violet-400'
  if (value === null) return 'text-fg-subtle'
  return 'text-fg-secondary'
}

/** 递归 JSON 树节点 */
function JsonNode({
  name,
  value,
  depth = 0,
  defaultOpen = false,
}: {
  name?: string
  value: JsonValue
  depth?: number
  defaultOpen?: boolean
}) {
  const [open, setOpen] = useState(defaultOpen || depth < 1)

  if (isPlainObject(value) || Array.isArray(value)) {
    const entries: [string, JsonValue][] = Array.isArray(value)
      ? value.map((item, index) => [String(index), item])
      : Object.entries(value)
    const isArray = Array.isArray(value)
    return (
      <div style={{ paddingLeft: depth === 0 ? 0 : 12 }}>
        <button
          type="button"
          onClick={() => setOpen((value) => !value)}
          className="flex items-center gap-1 font-mono text-[12px] text-fg-secondary hover:text-fg-primary"
        >
          {open ? (
            <ChevronDown className="h-3 w-3 shrink-0 text-fg-subtle" />
          ) : (
            <ChevronRight className="h-3 w-3 shrink-0 text-fg-subtle" />
          )}
          {name !== undefined ? <span className="text-blue-400">{name}</span> : null}
          <span className="text-fg-subtle">
            {isArray ? `[${entries.length}]` : `{${entries.length}}`}
          </span>
        </button>
        {open ? (
          <div className="border-l border-border/70">
            {entries.map(([key, child]) => (
              <JsonNode key={key} name={key} value={child} depth={depth + 1} />
            ))}
          </div>
        ) : null}
      </div>
    )
  }

  return (
    <div className="flex items-start gap-1 font-mono text-[12px]" style={{ paddingLeft: depth * 12 }}>
      <span className="whitespace-pre">
        {name !== undefined ? <span className="text-blue-400">{name}</span> : null}
        {name !== undefined ? <span className="text-fg-subtle">: </span> : null}
        <span className={valueColor(value)}>
          {typeof value === 'string' ? `"${value}"` : String(value)}
        </span>
      </span>
    </div>
  )
}

/** 容器详情 · Inspect 标签页（docker inspect 原始 JSON 树） */
export function InspectTab({ containerId }: { containerId: string }) {
  const { data, isLoading, error, refetch } = useContainerInspect(containerId)

  if (isLoading) {
    return (
      <div className="elev-card space-y-2 p-5">
        {Array.from({ length: 10 }).map((_, index) => (
          <Skeleton key={index} className="h-4 w-full" />
        ))}
      </div>
    )
  }

  if (error) {
    return (
      <div className="elev-card p-5">
        <ErrorState
          title="无法加载 inspect 数据"
          message={error instanceof Error ? error.message : undefined}
          onRetry={() => void refetch()}
        />
      </div>
    )
  }

  const json = (data ?? {}) as JsonValue

  return (
    <div className="elev-card overflow-hidden">
      <div className="flex items-center justify-between border-b border-border px-4 py-2.5">
        <h3 className="text-sm font-semibold">docker inspect</h3>
        <div className="flex items-center gap-2">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => {
              void navigator.clipboard
                ?.writeText(JSON.stringify(json, null, 2))
                .then(() => undefined)
                .catch(() => undefined)
            }}
          >
            复制 JSON
          </Button>
        </div>
      </div>
      <div className="max-h-[560px] overflow-auto p-4">
        <JsonNode value={json} defaultOpen />
      </div>
    </div>
  )
}
