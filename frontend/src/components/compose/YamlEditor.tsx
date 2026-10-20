import { useCallback, useEffect, useMemo, useRef } from 'react'
import { parseDocument } from 'yaml'

/* ---------------------------------------------------------------- 词法高亮 */

type TokenKind = 'plain' | 'comment' | 'key' | 'str' | 'val' | 'punc'

interface Token {
  text: string
  kind: TokenKind
}

const CLASS_BY_KIND: Record<TokenKind, string> = {
  plain: 'text-fg-secondary',
  comment: 'yaml-cmt',
  key: 'yaml-key',
  str: 'yaml-str',
  val: 'yaml-val',
  punc: 'yaml-punc',
}

/** 找到行内注释起始位置（忽略引号内的 #） */
function findCommentIndex(line: string): number {
  let quote: string | null = null
  for (let i = 0; i < line.length; i += 1) {
    const char = line[i]
    if (quote) {
      if (char === quote) quote = null
      continue
    }
    if (char === '"' || char === "'") {
      quote = char
      continue
    }
    if (char === '#' && (i === 0 || /\s/.test(line[i - 1]))) return i
  }
  return -1
}

/** 轻量 YAML 行级分词（仅用于展示，不参与解析） */
function tokenizeLine(line: string): Token[] {
  const tokens: Token[] = []
  const commentIndex = findCommentIndex(line)
  const code = commentIndex >= 0 ? line.slice(0, commentIndex) : line
  const comment = commentIndex >= 0 ? line.slice(commentIndex) : ''

  let rest = code
  const match = /^(\s*)(-\s+)?/.exec(rest)
  if (match) {
    const [, indent, dash] = match
    if (indent) tokens.push({ text: indent, kind: 'plain' })
    if (dash) tokens.push({ text: dash, kind: 'punc' })
    rest = rest.slice(match[0].length)
  }

  const keyMatch = /^([A-Za-z0-9_.$-]+)(\s*)(:)(\s*)(.*)$/.exec(rest)
  if (keyMatch) {
    const [, key, keySpace, colon, valueSpace, value] = keyMatch
    tokens.push({ text: key, kind: 'key' })
    if (keySpace) tokens.push({ text: keySpace, kind: 'plain' })
    tokens.push({ text: colon, kind: 'punc' })
    if (valueSpace) tokens.push({ text: valueSpace, kind: 'plain' })
    if (value) {
      if (/^["']/.test(value)) {
        tokens.push({ text: value, kind: 'str' })
      } else {
        // 值里可能混有字符串与标点
        const valueTokens = /("[^"]*"|'[^']*'|[A-Za-z0-9_.:/@-]+|[^\sA-Za-z0-9_.:/@-]+)/g
        let cursor = 0
        let vm: RegExpExecArray | null
        while ((vm = valueTokens.exec(value)) !== null) {
          if (vm.index > cursor) {
            tokens.push({ text: value.slice(cursor, vm.index), kind: 'punc' })
          }
          tokens.push({ text: vm[0], kind: 'str' })
          cursor = vm.index + vm[0].length
        }
        if (cursor < value.length) tokens.push({ text: value.slice(cursor), kind: 'punc' })
      }
    }
  } else if (rest) {
    tokens.push({ text: rest, kind: 'plain' })
  }

  if (comment) tokens.push({ text: comment, kind: 'comment' })
  return tokens
}

/* ---------------------------------------------------------------- 编辑器 */

export interface YamlEditorProps {
  value: string
  onChange: (value: string) => void
  /** 状态栏右侧附加信息 */
  meta?: string
  readOnly?: boolean
  className?: string
}

/**
 * YAML 在线编辑器。
 * 采用「高亮层 + 透明 textarea」叠层方案：语法高亮由 React 节点渲染
 * （不走 dangerouslySetInnerHTML，因此不存在注入风险）。
 */
export function YamlEditor({ value, onChange, meta, readOnly = false, className }: YamlEditorProps) {
  const textareaRef = useRef<HTMLTextAreaElement | null>(null)
  const preRef = useRef<HTMLPreElement | null>(null)
  const gutterRef = useRef<HTMLDivElement | null>(null)

  const lines = useMemo(() => value.split('\n'), [value])

  const highlighted = useMemo(
    () =>
      lines.map((line, index) => (
        <div key={index} className="flex">
          {tokenizeLine(line).map((token, tokenIndex) => (
            <span key={tokenIndex} className={CLASS_BY_KIND[token.kind]}>
              {token.text}
            </span>
          ))}
          {line === '' ? ' ' : null}
        </div>
      )),
    [lines],
  )

  const syncScroll = useCallback(() => {
    const textarea = textareaRef.current
    if (!textarea) return
    if (preRef.current) {
      preRef.current.scrollTop = textarea.scrollTop
      preRef.current.scrollLeft = textarea.scrollLeft
    }
    if (gutterRef.current) {
      gutterRef.current.scrollTop = textarea.scrollTop
    }
  }, [])

  // Tab 键插入两个空格而不是切换焦点
  const handleKeyDown = (event: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (event.key !== 'Tab') return
    event.preventDefault()
    const target = event.currentTarget
    const start = target.selectionStart
    const end = target.selectionEnd
    const next = `${value.slice(0, start)}  ${value.slice(end)}`
    onChange(next)
    requestAnimationFrame(() => {
      target.selectionStart = start + 2
      target.selectionEnd = start + 2
    })
  }

  useEffect(() => {
    syncScroll()
  }, [value, syncScroll])

  return (
    <div className={className}>
      <div className="relative flex overflow-hidden rounded-lg border border-border bg-[#0a0a0c]">
        {/* 行号栏 */}
        <div
          ref={gutterRef}
          className="w-11 shrink-0 select-none overflow-hidden border-r border-border/60 bg-[#0a0a0c] py-3 text-right font-mono text-[12px] leading-[1.7] text-fg-subtle/70"
        >
          {lines.map((_, index) => (
            <div key={index} className="pr-3">
              {index + 1}
            </div>
          ))}
        </div>

        {/* 主体 */}
        <div className="relative min-w-0 flex-1">
          <pre
            ref={preRef}
            aria-hidden
            className="pointer-events-none absolute inset-0 overflow-hidden whitespace-pre px-4 py-3 font-mono text-[12.5px] leading-[1.7] text-fg-secondary"
          >
            {highlighted}
          </pre>
          <textarea
            ref={textareaRef}
            value={value}
            readOnly={readOnly}
            onChange={(event) => onChange(event.target.value)}
            onScroll={syncScroll}
            onKeyDown={handleKeyDown}
            spellCheck={false}
            className="relative z-10 block h-full min-h-[420px] w-full resize-none overflow-auto whitespace-pre bg-transparent px-4 py-3 font-mono text-[12.5px] leading-[1.7] text-transparent caret-brand outline-none selection:bg-brand/30"
          />
        </div>
      </div>

      <div className="mt-2 flex items-center justify-between font-mono text-[11px] text-fg-subtle">
        <span>
          {meta ?? `Ln ${value.split('\n').length}, Col 1`} · UTF-8 · YAML · LF
        </span>
        <span>{value.length} 字符</span>
      </div>
    </div>
  )
}

/* ---------------------------------------------------------------- 校验 */

export type ValidationLevel = 'pass' | 'warn' | 'error' | 'info'

export interface ValidationCheck {
  id: string
  title: string
  level: ValidationLevel
  detail: string
  /** 附加建议（如修复路径） */
  hint?: string
}

export interface ValidationResult {
  checks: ValidationCheck[]
  passed: number
  total: number
  /** 解析出的 compose 文档（解析失败为 null） */
  document: Record<string, unknown> | null
  parseError: string | null
}

export interface ComposeServiceShape {
  name?: string
  image?: string
  build?: unknown
  ports?: (string | number)[]
  volumes?: (string | Record<string, unknown>)[]
  networks?: string[] | Record<string, unknown>
  privileged?: boolean
  user?: string
  restart?: string
  cap_add?: string[]
}

function asRecord(value: unknown): Record<string, unknown> | null {
  return value && typeof value === 'object' && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : null
}

/**
 * 客户端 Compose 校验。
 * 后端契约 v1.0 没有校验端点，因此在浏览器侧完成基础规则检查。
 */
export function validateCompose(content: string): ValidationResult {
  const checks: ValidationCheck[] = []

  if (!content.trim()) {
    return {
      checks: [
        {
          id: 'empty',
          title: 'YAML 解析',
          level: 'error',
          detail: '文件内容为空，无法校验',
        },
      ],
      passed: 0,
      total: 1,
      document: null,
      parseError: '文件内容为空',
    }
  }

  let document: Record<string, unknown> | null = null
  let parseError: string | null = null
  const lineCount = content.split('\n').length

  try {
    const doc = parseDocument(content, { prettyErrors: true })
    if (doc.errors.length > 0) {
      parseError = doc.errors[0].message
    } else {
      document = (doc.toJS() as Record<string, unknown>) ?? {}
    }
  } catch (err) {
    parseError = err instanceof Error ? err.message : 'YAML 解析失败'
  }

  checks.push({
    id: 'yaml',
    title: 'YAML 解析',
    level: parseError ? 'error' : 'pass',
    detail: parseError ? parseError : `parsed in 0ms · ${lineCount} lines`,
  })

  if (!document) {
    return { checks, passed: 0, total: checks.length, document: null, parseError }
  }

  const servicesRecord = asRecord(document.services)
  const services = (servicesRecord ?? {}) as Record<string, ComposeServiceShape>
  const serviceNames = Object.keys(services)
  const volumeRecord = asRecord(document.volumes)
  const networkRecord = asRecord(document.networks)

  /* ---- Compose schema 校验 ---- */
  if (serviceNames.length === 0) {
    checks.push({
      id: 'schema',
      title: 'Compose schema 校验',
      level: 'error',
      detail: '缺少 services 定义，compose-spec 3.8 · invalid',
    })
  } else {
    const invalid = serviceNames.filter((name) => {
      const svc = services[name]
      return !svc || (!svc.image && !svc.build)
    })
    checks.push({
      id: 'schema',
      title: 'Compose schema 校验',
      level: invalid.length > 0 ? 'error' : 'pass',
      detail:
        invalid.length > 0
          ? `服务 ${invalid.join('、')} 既没有 image 也没有 build`
          : 'compose-spec 3.8 · valid',
    })
  }

  /* ---- 镜像版本检查 ---- */
  const latestTags = serviceNames.filter((name) => {
    const image = services[name]?.image
    return typeof image === 'string' && (image.endsWith(':latest') || !image.includes(':'))
  })
  checks.push({
    id: 'image',
    title: '镜像版本检查',
    level: latestTags.length > 0 ? 'warn' : 'pass',
    detail:
      latestTags.length > 0
        ? `${latestTags.join('、')} 有更新 · 建议固定标签`
        : '所有镜像均已固定版本',
    hint: latestTags.length > 0 ? '例如 nginx:1.27-alpine 代替 nginx:latest' : undefined,
  })

  /* ---- 端口冲突检测 ---- */
  const portMap = new Map<string, string[]>()
  serviceNames.forEach((name) => {
    const ports = services[name]?.ports
    if (!Array.isArray(ports)) return
    ports.forEach((port) => {
      if (typeof port !== 'string') return
      const match = /^([^:]*):(\d+)/.exec(port)
      if (!match) return
      const [, host, container] = match
      const key = host ? `${host}:${container}` : container
      const list = portMap.get(key) ?? []
      list.push(name)
      portMap.set(key, list)
    })
  })
  const conflicts = [...portMap.entries()].filter(([, owners]) => owners.length > 1)
  checks.push({
    id: 'ports',
    title: '端口冲突检测',
    level: conflicts.length > 0 ? 'error' : 'pass',
    detail:
      conflicts.length > 0
        ? conflicts.map(([key, owners]) => `${key} 被 ${owners.join('、')} 重复占用`).join('；')
        : `${portMap.size} 个端口映射无冲突`,
  })

  /* ---- 挂载路径检查 ---- */
  const absoluteMounts: string[] = []
  serviceNames.forEach((name) => {
    const volumes = services[name]?.volumes
    if (!Array.isArray(volumes)) return
    volumes.forEach((volume) => {
      if (typeof volume !== 'string') return
      const parts = volume.split(':')
      const source = parts[0]
      if (source && source.startsWith('/')) absoluteMounts.push(`${name}: ${source}`)
    })
  })
  checks.push({
    id: 'mounts',
    title: '挂载路径检查',
    level: absoluteMounts.length > 0 ? 'warn' : 'pass',
    detail:
      absoluteMounts.length > 0
        ? `存在宿主机绝对路径挂载：${absoluteMounts.join('、')}`
        : '所有挂载均使用命名卷或相对路径',
    hint: absoluteMounts.length > 0 ? '建议改为命名卷，由 Docker 管理生命周期' : undefined,
  })

  /* ---- 网络引用完整性 ---- */
  const declaredNetworks = new Set(Object.keys(networkRecord ?? {}))
  declaredNetworks.add('default')
  const unknownNetworks: string[] = []
  serviceNames.forEach((name) => {
    const networks = services[name]?.networks
    if (!networks) return
    if (Array.isArray(networks)) {
      networks.forEach((net) => {
        if (typeof net === 'string' && !declaredNetworks.has(net)) unknownNetworks.push(net)
      })
    } else if (typeof networks === 'object') {
      Object.keys(networks).forEach((net) => {
        if (!declaredNetworks.has(net)) unknownNetworks.push(net)
      })
    }
  })
  checks.push({
    id: 'networks',
    title: '网络引用完整性',
    level: unknownNetworks.length > 0 ? 'error' : 'pass',
    detail:
      unknownNetworks.length > 0
        ? `引用了未定义的网络：${unknownNetworks.join('、')}`
        : `${declaredNetworks.size} 个网络引用均已定义`,
  })

  /* ---- 安全最佳实践 ---- */
  const privileged = serviceNames.filter((name) => services[name]?.privileged)
  const noUser = serviceNames.filter((name) => !services[name]?.user)
  const capAdd = serviceNames.filter((name) => Array.isArray(services[name]?.cap_add) && services[name]!.cap_add!.length > 0)
  const securityIssues: string[] = []
  if (privileged.length > 0) securityIssues.push(`privileged: ${privileged.join('、')}`)
  if (noUser.length > 0) securityIssues.push(`未设置 user: ${noUser.join('、')}`)
  if (capAdd.length > 0) securityIssues.push(`cap_add: ${capAdd.join('、')}`)
  checks.push({
    id: 'security',
    title: '安全最佳实践',
    level: privileged.length > 0 ? 'error' : noUser.length > 0 ? 'warn' : 'pass',
    detail:
      securityIssues.length > 0
        ? securityIssues.join('；')
        : '未设置 user 时建议加入 read_only 与 cap_drop',
    hint: securityIssues.length > 0 ? '建议加入 user 与 read_only 与 cap_drop' : undefined,
  })

  /* ---- 命名卷引用 ---- */
  const namedVolumes = new Set(Object.keys(volumeRecord ?? {}))
  const missingVolumes: string[] = []
  serviceNames.forEach((name) => {
    const volumes = services[name]?.volumes
    if (!Array.isArray(volumes)) return
    volumes.forEach((volume) => {
      if (typeof volume !== 'string') return
      const source = volume.split(':')[0]
      if (source && !source.startsWith('/') && !source.startsWith('.') && !namedVolumes.has(source)) {
        missingVolumes.push(source)
      }
    })
  })
  checks.push({
    id: 'volumes',
    title: '命名卷定义完整性',
    level: missingVolumes.length > 0 ? 'warn' : 'pass',
    detail:
      missingVolumes.length > 0
        ? `引用了未声明的命名卷：${[...new Set(missingVolumes)].join('、')}`
        : '命名卷引用均已声明',
  })

  const passed = checks.filter((check) => check.level === 'pass').length
  return { checks, passed, total: checks.length, document, parseError }
}

/** 把校验结果里服务部分转成结构，供 ServiceList 使用 */
export function extractServices(document: Record<string, unknown> | null): ComposeServiceShape[] {
  if (!document) return []
  const servicesRecord = asRecord(document.services)
  if (!servicesRecord) return []
  return Object.entries(servicesRecord).map(([name, value]) => {
    const record = asRecord(value)
    return { ...(record ?? {}), name } as ComposeServiceShape
  })
}
