import { useEffect, useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { Container, KeyRound, Loader2, Lock, User } from 'lucide-react'
import { useAuth } from '@/hooks/useAuth'
import { ApiError } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Input, Label } from '@/components/ui/input'
import { useTheme } from '@/hooks/useTheme'

/** 登录页 + 首次启动强制改密 */
export function Login() {
  const navigate = useNavigate()
  const { login, changePassword, mustChangePassword, user, isBootstrapping } = useAuth()
  const { resolvedTheme } = useTheme()

  const [username, setUsername] = useState('admin')
  const [password, setPassword] = useState('')
  const [oldPassword, setOldPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [pending, setPending] = useState(false)
  const [usedDefaultPassword, setUsedDefaultPassword] = useState(false)

  // 已登录则直接进入面板
  useEffect(() => {
    if (!isBootstrapping && user && !mustChangePassword) {
      navigate('/', { replace: true })
    }
  }, [isBootstrapping, user, mustChangePassword, navigate])

  const handleLogin = async (event: FormEvent) => {
    event.preventDefault()
    setError(null)
    setPending(true)
    try {
      const result = await login(username.trim(), password)
      setUsedDefaultPassword(result.defaultPasswordInUse)
      if (result.mustChangePassword) {
        setOldPassword(password)
        setPassword('')
      } else {
        navigate('/', { replace: true })
      }
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '登录失败，请稍后重试')
    } finally {
      setPending(false)
    }
  }

  const handleChangePassword = async (event: FormEvent) => {
    event.preventDefault()
    setError(null)
    if (newPassword.length < 8) {
      setError('新密码至少 8 位')
      return
    }
    if (newPassword !== confirmPassword) {
      setError('两次输入的新密码不一致')
      return
    }
    setPending(true)
    try {
      await changePassword(oldPassword, newPassword)
      navigate('/', { replace: true })
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '修改密码失败，请稍后重试')
    } finally {
      setPending(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-bg-base px-4 py-10">
      <div className="w-full max-w-sm">
        <div className="mb-6 flex flex-col items-center gap-3">
          <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-gradient-to-br from-brand to-cyan-400 shadow-glow-blue">
            <Container className="h-5 w-5 text-white" />
          </div>
          <div className="text-center">
            <h1 className="text-lg font-semibold tracking-tight">Dockermanage</h1>
            <p className="mt-1 text-[13px] text-fg-muted">
              {mustChangePassword
                ? '首次启动，请先修改初始密码'
                : '登录以管理本机 Docker 资源'}
            </p>
          </div>
        </div>

        <div className="surface-card p-5">
          {mustChangePassword ? (
            <form onSubmit={handleChangePassword} className="space-y-4">
              {usedDefaultPassword ? (
                <p className="rounded-md border border-warning/30 bg-warning-soft px-3 py-2 text-[12px] text-warning">
                  当前仍在使用默认初始密码，请立即设置一个仅你知道的强密码。
                </p>
              ) : null}
              <div className="space-y-1.5">
                <Label htmlFor="old-password">当前密码</Label>
                <div className="relative">
                  <KeyRound className="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-fg-subtle" />
                  <Input
                    id="old-password"
                    type="password"
                    autoComplete="current-password"
                    className="pl-8"
                    value={oldPassword}
                    onChange={(e) => setOldPassword(e.target.value)}
                    required
                  />
                </div>
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="new-password">新密码</Label>
                <div className="relative">
                  <Lock className="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-fg-subtle" />
                  <Input
                    id="new-password"
                    type="password"
                    autoComplete="new-password"
                    className="pl-8"
                    value={newPassword}
                    onChange={(e) => setNewPassword(e.target.value)}
                    required
                  />
                </div>
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="confirm-password">确认新密码</Label>
                <div className="relative">
                  <Lock className="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-fg-subtle" />
                  <Input
                    id="confirm-password"
                    type="password"
                    autoComplete="new-password"
                    className="pl-8"
                    value={confirmPassword}
                    onChange={(e) => setConfirmPassword(e.target.value)}
                    required
                  />
                </div>
              </div>
              {error ? <ErrorText message={error} /> : null}
              <Button type="submit" variant="primary" className="w-full" disabled={pending}>
                {pending ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : null}
                {pending ? '提交中…' : '修改密码并进入'}
              </Button>
            </form>
          ) : (
            <form onSubmit={handleLogin} className="space-y-4">
              <div className="rounded-md border border-brand/30 bg-brand-soft px-3 py-2 text-[12px] leading-relaxed text-fg-secondary">
                <div className="mb-1 font-medium text-fg-primary">默认账号</div>
                <div className="flex items-center gap-4">
                  <span>
                    用户名 <code className="rounded bg-bg-muted px-1 py-0.5 font-mono text-fg-primary">admin</code>
                  </span>
                  <span>
                    密码 <code className="rounded bg-bg-muted px-1 py-0.5 font-mono text-fg-primary">admin</code>
                  </span>
                </div>
                <div className="mt-1.5 text-[11px] text-fg-subtle">
                  登录后系统会强制要求你立即修改密码
                </div>
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="username">用户名</Label>
                <div className="relative">
                  <User className="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-fg-subtle" />
                  <Input
                    id="username"
                    autoComplete="username"
                    className="pl-8"
                    value={username}
                    onChange={(e) => setUsername(e.target.value)}
                    required
                  />
                </div>
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="password">密码</Label>
                <div className="relative">
                  <Lock className="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-fg-subtle" />
                  <Input
                    id="password"
                    type="password"
                    autoComplete="current-password"
                    className="pl-8"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    required
                  />
                </div>
              </div>
              {error ? <ErrorText message={error} /> : null}
              <Button type="submit" variant="primary" className="w-full" disabled={pending}>
                {pending ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : null}
                {pending ? '登录中…' : '登录'}
              </Button>
            </form>
          )}
        </div>

        <p className="mt-4 text-center text-[11px] text-fg-subtle">
          初始密码可在后端日志与 <code className="font-mono">data/.password</code> 中查看 · 主题{' '}
          {resolvedTheme === 'dark' ? '暗色' : '亮色'}
        </p>
      </div>
    </div>
  )
}

function ErrorText({ message }: { message: string }) {
  return (
    <p className="rounded-md border border-danger/30 bg-danger-soft px-3 py-2 text-[12px] text-danger">
      {message}
    </p>
  )
}
