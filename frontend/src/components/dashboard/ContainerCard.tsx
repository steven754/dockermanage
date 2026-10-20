import { Link } from 'react-router-dom'
import { formatBytes, formatBytesPerSec, formatPorts } from '@/lib/format'
import type { Container } from '@/lib/types'
import { ContainerIcon } from '@/components/common/ContainerIcon'
import { StateBadge } from '@/components/common/StatusBadge'
import { cn } from '@/lib/utils'

/** Dashboard 容器卡片（设计稿 §5） */
export function ContainerCard({
  container,
  live,
  className,
}: {
  container: Container
  /** SSE 推来的实时指标 */
  live?: { cpuPct: number; memUsageBytes: number; netRxBytesPerSec: number } | null
  className?: string
}) {
  const cpu = live?.cpuPct ?? container.cpuPct
  const mem = live?.memUsageBytes ?? container.memUsageBytes
  const net = live?.netRxBytesPerSec ?? container.netRxBytesPerSec
  const ports = formatPorts(container.ports)
  const subtitle = [container.image, ports].filter(Boolean).join(' · ')

  return (
    <Link
      to={`/containers/${container.id}`}
      className={cn('surface-card surface-card-hover group block p-3.5', className)}
    >
      <div className="flex items-start gap-3">
        <ContainerIcon image={container.image} name={container.name} />
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <span className="truncate font-medium">{container.name}</span>
            <StateBadge state={container.state} exitCode={container.exitCode} />
          </div>
          <div className="mt-0.5 truncate font-mono text-[11px] text-fg-subtle">{subtitle}</div>
          <div className="mt-2.5 grid grid-cols-3 gap-2 text-[11px]">
            <div>
              <span className="text-fg-subtle">CPU</span>{' '}
              <span className="font-mono text-fg-secondary">
                {cpu === null || cpu === undefined ? '—' : `${cpu.toFixed(1)}%`}
              </span>
            </div>
            <div>
              <span className="text-fg-subtle">MEM</span>{' '}
              <span className="font-mono text-fg-secondary">
                {mem === null || mem === undefined ? '—' : formatBytes(mem)}
              </span>
            </div>
            <div>
              <span className="text-fg-subtle">NET</span>{' '}
              <span className="font-mono text-fg-secondary">
                {net === null || net === undefined ? '—' : formatBytesPerSec(net, 0)}
              </span>
            </div>
          </div>
        </div>
      </div>
    </Link>
  )
}
