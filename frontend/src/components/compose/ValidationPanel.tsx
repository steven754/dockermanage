import { AlertTriangle, CheckCircle2, Info, ShieldAlert, XCircle } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { ValidationCheck, ValidationLevel, ValidationResult } from './YamlEditor'

const META: Record<
  ValidationLevel,
  { label: string; icon: LucideIcon; cls: string; badge: string }
> = {
  pass: {
    label: '通过',
    icon: CheckCircle2,
    cls: 'text-success',
    badge: 'border-success/30 bg-success-soft text-success',
  },
  warn: {
    label: '警告',
    icon: AlertTriangle,
    cls: 'text-warning',
    badge: 'border-warning/30 bg-warning-soft text-warning',
  },
  error: {
    label: '错误',
    icon: XCircle,
    cls: 'text-danger',
    badge: 'border-danger/30 bg-danger-soft text-danger',
  },
  info: {
    label: '建议',
    icon: Info,
    cls: 'text-info',
    badge: 'border-info/30 bg-info-soft text-info',
  },
}

/** 校验结果总览 */
export function ValidationOverview({ result }: { result: ValidationResult }) {
  const hasError = result.checks.some((check) => check.level === 'error')
  const hasWarn = result.checks.some((check) => check.level === 'warn')
  const level: ValidationLevel = hasError ? 'error' : hasWarn ? 'warn' : 'pass'
  const meta = META[level]
  const Icon = meta.icon

  const serviceCount = Object.keys(
    (result.document?.services as Record<string, unknown> | undefined) ?? {},
  ).length
  const volumeCount = Object.keys(
    (result.document?.volumes as Record<string, unknown> | undefined) ?? {},
  ).length
  const networkCount = Object.keys(
    (result.document?.networks as Record<string, unknown> | undefined) ?? {},
  ).length

  return (
    <div className="rounded-lg border border-border bg-bg-base p-4">
      <div className="flex items-start gap-3">
        <div
          className={cn(
            'flex h-8 w-8 shrink-0 items-center justify-center rounded-lg',
            level === 'pass' && 'bg-success-soft',
            level === 'warn' && 'bg-warning-soft',
            level === 'error' && 'bg-danger-soft',
          )}
        >
          <Icon className={cn('h-4 w-4', meta.cls)} />
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <span className="text-sm font-semibold">
              {level === 'pass'
                ? 'YAML 语法有效'
                : level === 'warn'
                  ? 'YAML 语法有效，存在警告'
                  : 'YAML 存在错误'}
            </span>
            <span
              className={cn(
                'inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-[11px]',
                meta.badge,
              )}
            >
              <span className="h-1.5 w-1.5 rounded-full bg-current" />
              {level === 'pass' ? '就绪' : level === 'warn' ? '需确认' : '不可部署'}
            </span>
          </div>
          <p className="mt-0.5 text-xs text-fg-muted">
            {result.parseError
              ? '解析失败，无法进行后续检查'
              : level === 'pass'
                ? '通过 schema 校验 · 可安全部署'
                : level === 'warn'
                  ? '通过 schema 校验 · 建议处理警告后再部署'
                  : '存在阻断性问题 · 修正后方可部署'}
          </p>
        </div>
      </div>

      <div className="mt-4 grid grid-cols-3 gap-2">
        <StatBox label="服务数" value={serviceCount} />
        <StatBox label="卷数" value={volumeCount} />
        <StatBox label="网络数" value={networkCount} />
      </div>
    </div>
  )
}

function StatBox({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-md border border-border bg-bg-card p-3">
      <div className="text-[11px] text-fg-subtle">{label}</div>
      <div className="mt-0.5 font-mono text-xl font-semibold">{value}</div>
    </div>
  )
}

/** 校验项列表 */
export function ValidationPanel({ result }: { result: ValidationResult }) {
  return (
    <div className="space-y-3">
      <ValidationOverview result={result} />

      <div className="text-[11px] text-fg-subtle">
        检查项 · {result.passed} / {result.total} 通过
      </div>

      <div className="space-y-2.5">
        {result.checks.map((check) => (
          <CheckItem key={check.id} check={check} />
        ))}
      </div>
    </div>
  )
}

function CheckItem({ check }: { check: ValidationCheck }) {
  const meta = META[check.level]
  const Icon = check.level === 'error' && check.id === 'security' ? ShieldAlert : meta.icon
  return (
    <div className="rounded-lg border border-border bg-bg-card p-3.5">
      <div className="flex items-center gap-2">
        <Icon className={cn('h-4 w-4 shrink-0', meta.cls)} />
        <span className="text-sm font-medium">{check.title}</span>
        <span
          className={cn(
            'ml-auto inline-flex items-center rounded-full border px-2 py-0.5 text-[11px]',
            meta.badge,
          )}
        >
          {meta.label}
        </span>
      </div>
      <p className="mt-1.5 font-mono text-[11px] leading-relaxed text-fg-subtle">
        {check.detail}
      </p>
      {check.hint ? (
        <p className="mt-1 text-[11px] text-fg-muted">建议：{check.hint}</p>
      ) : null}
    </div>
  )
}
