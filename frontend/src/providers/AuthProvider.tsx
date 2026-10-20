import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from 'react'
import { useQueryClient } from '@tanstack/react-query'
import {
  ApiError,
  authApi,
  getAccessToken,
  setAccessToken,
  setMustChangePasswordHandler,
  setRefreshToken,
  setUnauthorizedHandler,
} from '@/lib/api'
import type { CurrentUser } from '@/lib/types'

/**
 * 认证 Provider。
 *  - 单用户 JWT 登录（access token 存 localStorage，refresh token 同样存一份以兼容无 cookie 场景）
 *  - 首次启动 mustChangePassword = true 时，强制停留在改密流程
 *  - 注册 401 处理：清 token + 跳登录
 */

interface AuthContextValue {
  /** 是否已完成鉴权（含 token 与用户信息） */
  isAuthenticated: boolean
  /** 首启初始密码未修改 → 需要强制改密 */
  mustChangePassword: boolean
  /** 仍在使用首启初始密码 → 提示尽快修改 */
  defaultPasswordInUse: boolean
  user: CurrentUser | null
  isBootstrapping: boolean
  login: (username: string, password: string) => Promise<{ mustChangePassword: boolean; defaultPasswordInUse: boolean }>
  logout: () => Promise<void>
  changePassword: (oldPassword: string, newPassword: string) => Promise<void>
  refreshUser: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient()
  const [user, setUser] = useState<CurrentUser | null>(null)
  const [mustChangePassword, setMustChangePassword] = useState(false)
  const [defaultPasswordInUse, setDefaultPasswordInUse] = useState(false)
  const [isBootstrapping, setIsBootstrapping] = useState(true)
  const mounted = useRef(true)

  useEffect(() => {
    mounted.current = true
    return () => {
      mounted.current = false
    }
  }, [])

  const clearSession = useCallback(() => {
    setAccessToken(null)
    setRefreshToken(null)
    setUser(null)
    setMustChangePassword(false)
    setDefaultPasswordInUse(false)
    queryClient.clear()
  }, [queryClient])

  // 401 → 清理会话（由 RequireAuth 负责跳转）
  useEffect(() => {
    setUnauthorizedHandler(() => {
      clearSession()
    })
    return () => setUnauthorizedHandler(null)
  }, [clearSession])

  // 403 MUST_CHANGE_PASSWORD → 强制回到改密流程（后端已强制，此处同步前端状态）
  useEffect(() => {
    setMustChangePasswordHandler(() => {
      setMustChangePassword(true)
    })
    return () => setMustChangePasswordHandler(null)
  }, [])

  // 启动时用已有 token 拉一次 /auth/me
  useEffect(() => {
    let cancelled = false
    const bootstrap = async () => {
      if (!getAccessToken()) {
        if (!cancelled) setIsBootstrapping(false)
        return
      }
      try {
        const me = await authApi.me()
        if (cancelled) return
        setUser(me)
        setMustChangePassword(Boolean(me.mustChangePassword))
      } catch (err) {
        if (err instanceof ApiError && err.status === 401) {
          setAccessToken(null)
          setRefreshToken(null)
        }
      } finally {
        if (!cancelled) setIsBootstrapping(false)
      }
    }
    void bootstrap()
    return () => {
      cancelled = true
    }
  }, [])

  const login = useCallback(async (username: string, password: string) => {
    const res = await authApi.login(username, password)
    setAccessToken(res.accessToken)
    setRefreshToken(res.refreshToken)
    const mustChange = Boolean(res.mustChangePassword)
    setMustChangePassword(mustChange)
    setDefaultPasswordInUse(Boolean(res.defaultPasswordInUse))
    try {
      const me = await authApi.me()
      setUser(me)
      setMustChangePassword(mustChange || Boolean(me.mustChangePassword))
    } catch {
      // /auth/me 失败不阻断登录流程，登录响应本身已带 mustChangePassword
      setUser({ id: 0, username, mustChangePassword: mustChange })
    }
    return { mustChangePassword: mustChange, defaultPasswordInUse: Boolean(res.defaultPasswordInUse) }
  }, [])

  const logout = useCallback(async () => {
    try {
      await authApi.logout()
    } catch {
      /* 后端登出失败也要清本地会话 */
    }
    clearSession()
  }, [clearSession])

  const changePassword = useCallback(
    async (oldPassword: string, newPassword: string) => {
      const res = await authApi.changePassword(oldPassword, newPassword)
      // 改密后旧令牌已失效，必须换成新令牌
      setAccessToken(res.accessToken)
      setRefreshToken(res.refreshToken)
      setMustChangePassword(false)
      setDefaultPasswordInUse(false)
      try {
        const me = await authApi.me()
        setUser(me)
      } catch {
        /* ignore */
      }
    },
    [],
  )

  const refreshUser = useCallback(async () => {
    if (!getAccessToken()) return
    try {
      const me = await authApi.me()
      setUser(me)
      setMustChangePassword(Boolean(me.mustChangePassword))
    } catch {
      /* ignore */
    }
  }, [])

  const value = useMemo<AuthContextValue>(
    () => ({
      isAuthenticated: Boolean(user) && Boolean(getAccessToken()),
      mustChangePassword,
      defaultPasswordInUse,
      user,
      isBootstrapping,
      login,
      logout,
      changePassword,
      refreshUser,
    }),
    [user, mustChangePassword, defaultPasswordInUse, isBootstrapping, login, logout, changePassword, refreshUser],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth 必须在 AuthProvider 内使用')
  return ctx
}
