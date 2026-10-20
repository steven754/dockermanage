import {
  Box,
  Cloud,
  Database,
  Download,
  DownloadCloud,
  Film,
  GitBranch,
  Globe,
  Home,
  Lock,
  Shield,
  Wrench,
  type LucideIcon,
} from 'lucide-react'
import { cn } from '@/lib/utils'

interface IconRule {
  /** 镜像名 / 容器名匹配关键字（小写包含匹配） */
  match: string[]
  icon: LucideIcon
  /** 图标块背景色 */
  bg: string
  /** 图标颜色 */
  fg: string
}

/** 常见镜像 → lucide 图标 + 品牌色（设计稿 §5 组件规范） */
const RULES: IconRule[] = [
  { match: ['nginx', 'traefik', 'caddy', 'haproxy'], icon: Globe, bg: 'bg-emerald-500/15', fg: 'text-emerald-400' },
  { match: ['postgres', 'mysql', 'mariadb', 'clickhouse'], icon: Database, bg: 'bg-blue-500/15', fg: 'text-blue-400' },
  { match: ['redis', 'valkey', 'memcached'], icon: Database, bg: 'bg-red-500/15', fg: 'text-red-400' },
  { match: ['qbittorrent', 'deluge'], icon: DownloadCloud, bg: 'bg-purple-500/15', fg: 'text-purple-400' },
  { match: ['jellyfin', 'plex', 'emby'], icon: Film, bg: 'bg-pink-500/15', fg: 'text-pink-400' },
  { match: ['nextcloud', 'owncloud', 'seafile'], icon: Cloud, bg: 'bg-cyan-500/15', fg: 'text-cyan-400' },
  { match: ['home-assistant', 'homeassistant', 'hass'], icon: Home, bg: 'bg-orange-500/15', fg: 'text-orange-400' },
  { match: ['adguard', 'pi-hole', 'pihole'], icon: Shield, bg: 'bg-indigo-500/15', fg: 'text-indigo-400' },
  { match: ['transmission', 'aria2', 'sabnzbd'], icon: Download, bg: 'bg-slate-500/15', fg: 'text-slate-400' },
  { match: ['vaultwarden', 'bitwarden'], icon: Lock, bg: 'bg-yellow-500/15', fg: 'text-yellow-400' },
  { match: ['gitea', 'gitlab', 'forgejo'], icon: GitBranch, bg: 'bg-violet-500/15', fg: 'text-violet-400' },
  { match: ['portainer'], icon: Wrench, bg: 'bg-yellow-500/15', fg: 'text-yellow-400' },
]

/** 根据镜像名推断图标与配色 */
export function resolveContainerIcon(image: string, name?: string): IconRule {
  const haystack = `${image} ${name ?? ''}`.toLowerCase()
  for (const rule of RULES) {
    if (rule.match.some((keyword) => haystack.includes(keyword))) return rule
  }
  return { match: [], icon: Box, bg: 'bg-bg-muted', fg: 'text-fg-muted' }
}

export function ContainerIcon({
  image,
  name,
  className,
  iconClassName,
}: {
  image: string
  name?: string
  className?: string
  iconClassName?: string
}) {
  const rule = resolveContainerIcon(image, name)
  const Icon = rule.icon
  return (
    <div
      className={cn(
        'flex h-9 w-9 shrink-0 items-center justify-center rounded-md',
        rule.bg,
        className,
      )}
    >
      <Icon className={cn('h-4 w-4', rule.fg, iconClassName)} />
    </div>
  )
}
