import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import { settingsApi } from '@/lib/api'
import type { ThemeName } from '@/lib/types'

/**
 * 自写主题 Provider（dark / light / system）。
 *  - 在 <html> 上加 / 移除 `dark` class，Tailwind `darkMode: 'class'`
 *  - system 跟随 `prefers-color-scheme` 媒体查询
 *  - 选择结果写入 localStorage，并与后端 settings 同步
 */

const STORAGE_KEY = 'dm.theme'

interface ThemeContextValue {
  /** 用户选择的主题（可能是 system） */
  theme: ThemeName
  /** 实际生效的主题（system 会被解析成 dark / light） */
  resolvedTheme: 'dark' | 'light'
  setTheme: (theme: ThemeName) => void
}

const ThemeContext = createContext<ThemeContextValue | null>(null)

function readStoredTheme(): ThemeName {
  try {
    const value = localStorage.getItem(STORAGE_KEY)
    if (value === 'dark' || value === 'light' || value === 'system') return value
  } catch {
    /* ignore */
  }
  return 'dark'
}

function systemPrefersDark(): boolean {
  if (typeof window === 'undefined' || !window.matchMedia) return true
  return window.matchMedia('(prefers-color-scheme: dark)').matches
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [theme, setThemeState] = useState<ThemeName>(readStoredTheme)
  const [systemDark, setSystemDark] = useState<boolean>(systemPrefersDark)

  // 监听系统主题变化（仅在 system 模式下生效）
  useEffect(() => {
    if (typeof window === 'undefined' || !window.matchMedia) return
    const mq = window.matchMedia('(prefers-color-scheme: dark)')
    const handler = (e: MediaQueryListEvent) => setSystemDark(e.matches)
    mq.addEventListener('change', handler)
    return () => mq.removeEventListener('change', handler)
  }, [])

  const resolvedTheme: 'dark' | 'light' = theme === 'system' ? (systemDark ? 'dark' : 'light') : theme

  // 应用到 <html>
  useEffect(() => {
    const root = document.documentElement
    root.classList.toggle('dark', resolvedTheme === 'dark')
    root.style.colorScheme = resolvedTheme
  }, [resolvedTheme])

  const setTheme = useCallback((next: ThemeName) => {
    setThemeState(next)
    try {
      localStorage.setItem(STORAGE_KEY, next)
    } catch {
      /* ignore */
    }
    // 同步到后端（失败不阻塞本地切换）
    void settingsApi.update({ theme: next }).catch(() => undefined)
  }, [])

  const value = useMemo<ThemeContextValue>(
    () => ({ theme, resolvedTheme, setTheme }),
    [theme, resolvedTheme, setTheme],
  )

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>
}

export function useTheme(): ThemeContextValue {
  const ctx = useContext(ThemeContext)
  if (!ctx) throw new Error('useTheme 必须在 ThemeProvider 内使用')
  return ctx
}
