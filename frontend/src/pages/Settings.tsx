import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Check,
  KeyRound,
  LogOut,
  Monitor,
  Moon,
  Palette,
  Server,
  Sun,
} from 'lucide-react'
import { toast } from 'sonner'
import { PageHeader } from '@/components/layout/PageHeader'
import { Button } from '@/components/ui/button'
import { Input, Label } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/input'
import { ErrorState } from '@/components/common/States'
import { useAuth } from '@/hooks/useAuth'
import { useTheme } from '@/hooks/useTheme'
import { useSettings, useUpdateSettings } from '@/hooks/useSettings'
import { ApiError } from '@/lib/api'
import { cn } from '@/lib/utils'
import type { ThemeName } from '@/lib/types'

const THEME_OPTIONS: { value: ThemeName; label: string; icon: typeof Moon; hint: string }[] = [
  { value: 'dark', label: '暗色', icon: Moon, hint: '默认，深色护眼' },
  { value: 'light', label: '亮色', icon: Sun, hint: '明亮环境' },
  { value: 'system', label: '跟随系统', icon: Monitor, hint: '与操作系统一致' },
]

/** 设置页 */
export function Settings() {
  const navigate = useNavigate()
  const { theme, setTheme } = useTheme()
  const { user, logout, changePassword } = useAuth()
  const { data: settings, isLoading, error, refetch } = useSettings()
  const updateSettings = useUpdateSettings()

  const [registryMirror, setRegistryMirror] = useState('')
  const [sampleInterval, setSampleInterval] = useState('10')
  const [retentionDays, setRetentionDays] = useState('7')

  const [oldPassword, setOldPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [pwdPending, setPwdPending] = useState(false)
  const [pwdError, setPwdError] = useState<string | null>(null)

  useEffect(() => {
    if (!settings) return
    setRegistryMirror(settings.registryMirror ?? '')
    setSampleInterval(String(settings.sampleIntervalSeconds ?? 10))
    setRetentionDays(String(settings.retentionDays ?? 7))
  }, [settings])

  const handleSave = () => {
    updateSettings.mutate(
      {
        registryMirror: registryMirror.trim(),
        sampleIntervalSeconds: Number(sampleInterval) || 10,
        retentionDays: Number(retentionDays) || 7,
      },
      {
        onSuccess: () => toast.success('设置已保存'),
        onError: (err) => toast.error(err instanceof Error ? err.message : '保存失败'),
      },
    )
  }

  const handleChangePassword = async () => {
    setPwdError(null)
    if (newPassword.length < 8) {
      setPwdError('新密码至少 8 位')
      return
    }
    if (newPassword !== confirmPassword) {
      setPwdError('两次输入的新密码不一致')
      return
    }
    setPwdPending(true)
    try {
      await changePassword(oldPassword, newPassword)
      toast.success('密码已修改')
      setOldPassword('')
      setNewPassword('')
      setConfirmPassword('')
    } catch (err) {
      setPwdError(err instanceof ApiError ? err.message : '修改失败，请稍后重试')
    } finally {
      setPwdPending(false)
    }
  }

  return (
    <div className="space-y-6">
      <PageHeader title="设置" description="面板外观、采集策略与账户安全" />

      <div className="grid grid-cols-1 gap-4 xl:grid-cols-2">
        {/* ===== 主题 ===== */}
        <section className="elev-card p-5">
          <div className="mb-4 flex items-center gap-2">
            <Palette className="h-4 w-4 text-brand" />
            <h2 className="text-base font-semibold">外观</h2>
          </div>
          <div className="grid grid-cols-3 gap-3">
            {THEME_OPTIONS.map((option) => {
              const Icon = option.icon
              const active = theme === option.value
              return (
                <button
                  key={option.value}
                  type="button"
                  onClick={() => setTheme(option.value)}
                  className={cn(
                    'relative rounded-lg border p-3 text-left transition-colors',
                    active
                      ? 'border-brand bg-brand-soft'
                      : 'border-border bg-bg-base hover:border-border-strong',
                  )}
                >
                  <Icon
                    className={cn('h-4 w-4', active ? 'text-brand' : 'text-fg-subtle')}
                  />
                  <div className="mt-2 text-[13px] font-medium">{option.label}</div>
                  <div className="mt-0.5 text-[11px] text-fg-subtle">{option.hint}</div>
                  {active ? (
                    <span className="absolute right-2 top-2 flex h-4 w-4 items-center justify-center rounded-full bg-brand text-white">
                      <Check className="h-3 w-3" />
                    </span>
                  ) : null}
                </button>
              )
            })}
          </div>
        </section>

        {/* ===== 采集与镜像 ===== */}
        <section className="elev-card p-5">
          <div className="mb-4 flex items-center gap-2">
            <Server className="h-4 w-4 text-brand" />
            <h2 className="text-base font-semibold">采集与镜像</h2>
          </div>

          {isLoading ? (
            <div className="space-y-3">
              <Skeleton className="h-9 w-full" />
              <Skeleton className="h-9 w-full" />
            </div>
          ) : error ? (
            <ErrorState
              title="无法加载设置"
              message={error instanceof Error ? error.message : undefined}
              onRetry={() => void refetch()}
            />
          ) : (
            <div className="space-y-4">
              <div className="space-y-1.5">
                <Label htmlFor="registry-mirror">镜像加速器</Label>
                <Input
                  id="registry-mirror"
                  value={registryMirror}
                  onChange={(e) => setRegistryMirror(e.target.value)}
                  placeholder="https://docker.1ms.run"
                />
                <p className="text-[11px] text-fg-subtle">
                  拉取镜像时使用的 registry mirror，留空表示直连官方源。
                </p>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1.5">
                  <Label htmlFor="sample-interval">采样间隔（秒）</Label>
                  <Input
                    id="sample-interval"
                    type="number"
                    min={1}
                    value={sampleInterval}
                    onChange={(e) => setSampleInterval(e.target.value)}
                  />
                </div>
                <div className="space-y-1.5">
                  <Label htmlFor="retention-days">数据保留（天）</Label>
                  <Input
                    id="retention-days"
                    type="number"
                    min={1}
                    value={retentionDays}
                    onChange={(e) => setRetentionDays(e.target.value)}
                  />
                </div>
              </div>

              <Button variant="primary" onClick={handleSave} disabled={updateSettings.isPending}>
                {updateSettings.isPending ? '保存中…' : '保存设置'}
              </Button>
            </div>
          )}
        </section>

        {/* ===== 修改密码 ===== */}
        <section className="elev-card p-5">
          <div className="mb-4 flex items-center gap-2">
            <KeyRound className="h-4 w-4 text-brand" />
            <h2 className="text-base font-semibold">修改密码</h2>
          </div>
          <div className="space-y-3.5">
            <div className="space-y-1.5">
              <Label htmlFor="old-password">当前密码</Label>
              <Input
                id="old-password"
                type="password"
                autoComplete="current-password"
                value={oldPassword}
                onChange={(e) => setOldPassword(e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="new-password">新密码</Label>
              <Input
                id="new-password"
                type="password"
                autoComplete="new-password"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="confirm-password">确认新密码</Label>
              <Input
                id="confirm-password"
                type="password"
                autoComplete="new-password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
              />
            </div>
            {pwdError ? (
              <p className="rounded-md border border-danger/30 bg-danger-soft px-3 py-2 text-[12px] text-danger">
                {pwdError}
              </p>
            ) : null}
            <Button variant="primary" onClick={() => void handleChangePassword()} disabled={pwdPending}>
              {pwdPending ? '提交中…' : '更新密码'}
            </Button>
          </div>
        </section>

        {/* ===== 账户 ===== */}
        <section className="elev-card p-5">
          <div className="mb-4 flex items-center gap-2">
            <Server className="h-4 w-4 text-brand" />
            <h2 className="text-base font-semibold">账户</h2>
          </div>
          <div className="space-y-4">
            <div className="flex items-center justify-between rounded-lg border border-border bg-bg-base px-3 py-2.5">
              <div>
                <div className="text-[13px] font-medium">{user?.username ?? 'admin'}</div>
                <div className="text-[11px] text-fg-subtle">单用户模式 · 本机自用</div>
              </div>
              <span className="rounded-full border border-brand/30 bg-brand-soft px-2 py-0.5 text-[11px] text-brand">
                管理员
              </span>
            </div>
            <Button
              variant="danger"
              onClick={() => {
                void logout().then(() => navigate('/login', { replace: true }))
              }}
            >
              <LogOut className="h-3.5 w-3.5" />
              退出登录
            </Button>
          </div>
        </section>
      </div>
    </div>
  )
}
