/**
 * 统一 API 客户端。
 * 所有后端请求都必须经由此文件，组件里不允许出现裸 fetch。
 *
 * 职责：
 *  - 自动附带 `Authorization: Bearer <access_token>`
 *  - 错误统一抛出 ApiError（含 HTTP 状态码 + 契约里的 code）
 *  - 401 统一触发未授权处理（跳登录页）
 */

import type {
  ComposeFile,
  ComposeProject,
  ComposeProjectSummary,
  ComposeService,
  Container,
  ContainerStats,
  CurrentUser,
  DashboardResponse,
  DockerEvent,
  Image,
  ItemsResponse,
  LoginResponse,
  Network,
  OkResponse,
  PruneResult,
  Settings,
  StatsHistory,
  StatsRange,
  ThemeName,
  UpdateResult,
  VersionInfo,
  Volume,
} from './types'

/* ---------------------------------------------------------------- 常量 */

const ACCESS_TOKEN_KEY = 'dm.accessToken'
const REFRESH_TOKEN_KEY = 'dm.refreshToken'

/** API 基地址；留空表示同源相对路径（开发环境走 vite proxy，生产由后端托管静态资源） */
export const API_BASE = (import.meta.env.VITE_API_BASE ?? '').replace(/\/+$/, '')

/* ---------------------------------------------------------------- 错误 */

export class ApiError extends Error {
  readonly status: number
  readonly code: string

  constructor(status: number, code: string, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }

  /** 是否为「后端不可达 / 网络错误」 */
  get isNetwork(): boolean {
    return this.status === 0
  }
}

/* ---------------------------------------------------------------- token */

export function getAccessToken(): string | null {
  try {
    return localStorage.getItem(ACCESS_TOKEN_KEY)
  } catch {
    return null
  }
}

export function setAccessToken(token: string | null): void {
  try {
    if (token) localStorage.setItem(ACCESS_TOKEN_KEY, token)
    else localStorage.removeItem(ACCESS_TOKEN_KEY)
  } catch {
    /* localStorage 不可用时忽略（隐私模式） */
  }
}

export function getRefreshToken(): string | null {
  try {
    return localStorage.getItem(REFRESH_TOKEN_KEY)
  } catch {
    return null
  }
}

export function setRefreshToken(token: string | null): void {
  try {
    if (token) localStorage.setItem(REFRESH_TOKEN_KEY, token)
    else localStorage.removeItem(REFRESH_TOKEN_KEY)
  } catch {
    /* ignore */
  }
}

/* -------------------------------------------------- 401 统一处理 */

type UnauthorizedHandler = () => void
let unauthorizedHandler: UnauthorizedHandler | null = null

/** 注册未授权回调（AuthProvider 注入：清 token + 跳 /login） */
export function setUnauthorizedHandler(handler: UnauthorizedHandler | null): void {
  unauthorizedHandler = handler
}

/* -------------------------------------------------- 403 强制改密统一处理 */

type MustChangePasswordHandler = () => void
let mustChangePasswordHandler: MustChangePasswordHandler | null = null

/**
 * 注册「初始密码未修改」回调。
 *
 * 后端在初始密码未修改时会以 403 + MUST_CHANGE_PASSWORD 拒绝除改密类接口外的
 * 全部请求，这里统一转成前端状态，强制用户回到改密流程。
 */
export function setMustChangePasswordHandler(handler: MustChangePasswordHandler | null): void {
  mustChangePasswordHandler = handler
}

/* ---------------------------------------------------------------- 工具 */

export type QueryValue = string | number | boolean | undefined | null

export function buildQuery(params?: Record<string, QueryValue>): string {
  if (!params) return ''
  const sp = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null || value === '') continue
    sp.set(key, String(value))
  }
  const s = sp.toString()
  return s ? `?${s}` : ''
}

/** 拼接为绝对 URL（供 EventSource / WebSocket 使用） */
export function absoluteUrl(path: string): string {
  if (/^https?:\/\//i.test(path)) return path
  const base = API_BASE || (typeof window !== 'undefined' ? window.location.origin : 'http://localhost')
  return `${base}${path.startsWith('/') ? path : `/${path}`}`
}

/** 转换为 ws(s):// 协议的 URL */
export function websocketUrl(path: string, params?: Record<string, QueryValue>): string {
  const url = new URL(absoluteUrl(`${path}${buildQuery(params)}`))
  url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:'
  return url.toString()
}

/**
 * SSE / WebSocket 无法自定义请求头，
 * 因此把 access token 以 `access_token` 查询参数附带（后端需支持该约定）。
 */
export function authQueryParams(): Record<string, QueryValue> {
  const token = getAccessToken()
  return token ? { access_token: token } : {}
}

/* ---------------------------------------------------------------- 请求 */

export interface RequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH'
  body?: unknown
  query?: Record<string, QueryValue>
  signal?: AbortSignal
  /** 跳过 401 跳转（登录接口自身使用） */
  skipAuthRedirect?: boolean
}

/**
 * 是否携带 Cookie。
 *  - 同源（生产环境由面板托管前端）：携带，HttpOnly refresh cookie 生效
 *  - 跨域（开发期 vite proxy 之外的直连）：不携带，避免触发
 *    `Access-Control-Allow-Origin: *` 与 credentials 互斥的 CORS 限制
 *    鉴权完全依赖 Authorization: Bearer 头
 */
export function shouldUseCredentials(): boolean {
  if (typeof window === 'undefined') return false
  if (!API_BASE) return true
  try {
    return new URL(API_BASE, window.location.origin).origin === window.location.origin
  } catch {
    return false
  }
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { method = 'GET', body, query, signal, skipAuthRedirect } = options
  const url = `${absoluteUrl(path)}${buildQuery(query)}`

  const headers: Record<string, string> = { Accept: 'application/json' }
  const token = getAccessToken()
  if (token) headers.Authorization = `Bearer ${token}`
  if (body !== undefined) headers['Content-Type'] = 'application/json'

  let response: Response
  try {
    response = await fetch(url, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
      signal,
      credentials: shouldUseCredentials() ? 'include' : 'omit',
    })
  } catch (err) {
    if (err instanceof DOMException && err.name === 'AbortError') throw err
    throw new ApiError(0, 'NETWORK_ERROR', '无法连接到后端服务，请确认面板已启动')
  }

  if (response.status === 204) return undefined as T

  const raw = await response.text()
  let parsed: unknown = undefined
  if (raw) {
    try {
      parsed = JSON.parse(raw)
    } catch {
      parsed = undefined
    }
  }

  if (!response.ok) {
    const errBody = parsed as { error?: { code?: string; message?: string } } | undefined
    const code = errBody?.error?.code ?? `HTTP_${response.status}`
    const message = errBody?.error?.message ?? response.statusText ?? '请求失败'
    if (response.status === 401 && !skipAuthRedirect) {
      unauthorizedHandler?.()
    }
    if (response.status === 403 && code === 'MUST_CHANGE_PASSWORD') {
      mustChangePasswordHandler?.()
    }
    throw new ApiError(response.status, code, message)
  }

  return parsed as T
}

/** 供 Mutation 使用的包装：把成功/失败 toast 收敛到一处 */
export const api = {
  get: <T>(path: string, query?: Record<string, QueryValue>, signal?: AbortSignal) =>
    request<T>(path, { method: 'GET', query, signal }),
  post: <T>(path: string, body?: unknown, query?: Record<string, QueryValue>) =>
    request<T>(path, { method: 'POST', body, query }),
  put: <T>(path: string, body?: unknown, query?: Record<string, QueryValue>) =>
    request<T>(path, { method: 'PUT', body, query }),
  del: <T>(path: string, query?: Record<string, QueryValue>) =>
    request<T>(path, { method: 'DELETE', query }),
  raw: request,
}

/* ---------------------------------------------------------------- 业务接口 */

/** 认证 */
export const authApi = {
  login: (username: string, password: string) =>
    request<LoginResponse>('/api/v1/auth/login', {
      method: 'POST',
      body: { username, password },
      skipAuthRedirect: true,
    }),
  logout: () => request<OkResponse>('/api/v1/auth/logout', { method: 'POST', body: {} }),
  /** 改密；响应带新的令牌对（改密前签发的令牌已失效） */
  changePassword: (oldPassword: string, newPassword: string) =>
    request<LoginResponse>('/api/v1/auth/password', {
      method: 'POST',
      body: { oldPassword, newPassword },
    }),
  me: () => request<CurrentUser>('/api/v1/auth/me'),
}

/** 总览 */
export const dashboardApi = {
  get: () => request<DashboardResponse>('/api/v1/dashboard'),
}

/** 容器 */
export const containerApi = {
  list: (query?: Record<string, QueryValue>) =>
    request<ItemsResponse<Container>>('/api/v1/containers', { query }),
  get: (id: string) => request<Container>(`/api/v1/containers/${encodeURIComponent(id)}`),
  inspect: (id: string) => request<unknown>(`/api/v1/containers/${encodeURIComponent(id)}/json`),
  stats: (id: string) => request<ContainerStats>(`/api/v1/containers/${encodeURIComponent(id)}/stats`),
  action: (id: string, action: 'start' | 'stop' | 'restart' | 'kill' | 'pause' | 'unpause') =>
    request<OkResponse>(`/api/v1/containers/${encodeURIComponent(id)}/${action}`, {
      method: 'POST',
      body: {},
    }),
  remove: (id: string, force = true) =>
    request<OkResponse>(`/api/v1/containers/${encodeURIComponent(id)}`, {
      method: 'DELETE',
      query: { force },
    }),
  prune: (all = false) =>
    request<PruneResult>('/api/v1/containers/prune', { method: 'POST', body: { all } }),
  update: (id: string) =>
    request<UpdateResult>(`/api/v1/containers/${encodeURIComponent(id)}/update`, {
      method: 'POST',
      body: {},
    }),
}

/** 镜像 */
export const imageApi = {
  list: () => request<ItemsResponse<Image>>('/api/v1/images'),
  pull: (image: string) =>
    request<{ ok: boolean; imageId: string }>('/api/v1/images/pull', {
      method: 'POST',
      body: { image },
    }),
  tag: (id: string, repo: string, tag: string) =>
    request<OkResponse>(`/api/v1/images/${encodeURIComponent(id)}/tag`, {
      method: 'POST',
      body: { repo, tag },
    }),
  remove: (id: string, force = false) =>
    request<OkResponse>(`/api/v1/images/${encodeURIComponent(id)}`, {
      method: 'DELETE',
      query: { force },
    }),
  prune: (dangling = true) =>
    request<PruneResult>('/api/v1/images/prune', { method: 'POST', body: { dangling } }),
}

/** 卷 */
export const volumeApi = {
  list: () => request<ItemsResponse<Volume>>('/api/v1/volumes'),
  remove: (name: string, force = false) =>
    request<OkResponse>(`/api/v1/volumes/${encodeURIComponent(name)}`, {
      method: 'DELETE',
      query: { force },
    }),
  prune: () =>
    request<PruneResult>('/api/v1/volumes/prune', { method: 'POST', body: {} }),
}

/** 网络 */
export const networkApi = {
  list: () => request<ItemsResponse<Network>>('/api/v1/networks'),
  remove: (id: string) =>
    request<OkResponse>(`/api/v1/networks/${encodeURIComponent(id)}`, { method: 'DELETE' }),
}

/** 事件 */
export const eventApi = {
  history: (query?: Record<string, QueryValue>) =>
    request<ItemsResponse<DockerEvent>>('/api/v1/events', { query }),
}

/** 指标 */
export const statsApi = {
  history: (range: StatsRange, containerId?: string) =>
    request<StatsHistory>('/api/v1/stats/history', {
      query: { range, containerId },
    }),
}

/** Compose */
export const composeApi = {
  projects: () => request<ItemsResponse<ComposeProjectSummary>>('/api/v1/compose/projects'),
  project: (name: string) =>
    request<ComposeProject>(`/api/v1/compose/projects/${encodeURIComponent(name)}`),
  saveFile: (name: string, filename: string, content: string) =>
    request<{ ok: boolean; sizeBytes: number }>(
      `/api/v1/compose/projects/${encodeURIComponent(name)}/files/${encodeURIComponent(filename)}`,
      { method: 'PUT', body: { content } },
    ),
  up: (name: string, services?: string[]) =>
    request<{ ok: boolean; started: string[] }>(
      `/api/v1/compose/projects/${encodeURIComponent(name)}/up`,
      { method: 'POST', body: services ? { services } : {} },
    ),
  down: (name: string, removeVolumes = false) =>
    request<OkResponse>(`/api/v1/compose/projects/${encodeURIComponent(name)}/down`, {
      method: 'POST',
      body: { removeVolumes },
    }),
  ps: (name: string) =>
    request<ItemsResponse<ComposeService>>(
      `/api/v1/compose/projects/${encodeURIComponent(name)}/ps`,
    ),
}

/** 设置 */
export const settingsApi = {
  get: () => request<Settings>('/api/v1/settings'),
  update: (payload: Partial<Settings>) =>
    request<OkResponse>('/api/v1/settings', { method: 'PUT', body: payload }),
}

/** 版本 */
export const versionApi = {
  get: () => request<VersionInfo>('/api/v1/version'),
}

/** 主题类型再导出，方便设置页引用 */
export type { ThemeName }
export type { ComposeFile }
