import type { Config } from 'tailwindcss'

/**
 * 把 CSS 变量（RGB 三通道格式）包装成 Tailwind 颜色，
 * 这样同一套类名在 light / dark 两种主题下自动取到不同色值。
 */
const v = (name: string, alpha = '<alpha-value>') => `rgb(var(${name}) / ${alpha})`

export default {
  darkMode: 'class',
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        // —— 设计稿 token：表面 ——
        bg: {
          base: v('--bg-base'),
          elev: v('--bg-elev'),
          card: v('--bg-card'),
          cardHover: v('--bg-card-hover'),
          muted: v('--bg-muted'),
          overlay: 'rgb(var(--overlay) / 0.72)',
        },
        // —— 设计稿 token：文字 ——
        fg: {
          primary: v('--fg-primary'),
          secondary: v('--fg-secondary'),
          muted: v('--fg-muted'),
          subtle: v('--fg-subtle'),
          disabled: v('--fg-disabled'),
        },
        // —— 设计稿 token：边框 ——
        border: {
          DEFAULT: v('--border'),
          strong: v('--border-strong'),
          focus: v('--brand'),
        },
        // —— 设计稿 token：主色 ——
        brand: {
          DEFAULT: v('--brand'),
          hover: v('--brand-hover'),
          soft: 'rgb(var(--brand) / 0.10)',
        },
        // —— 语义色 ——
        success: { DEFAULT: v('--success'), soft: 'rgb(var(--success) / 0.12)' },
        warning: { DEFAULT: v('--warning'), soft: 'rgb(var(--warning) / 0.12)' },
        danger: { DEFAULT: v('--danger'), soft: 'rgb(var(--danger) / 0.12)' },
        info: { DEFAULT: v('--info'), soft: 'rgb(var(--info) / 0.12)' },
        // —— shadcn/ui 语义别名（内部组件使用）——
        background: v('--bg-base'),
        foreground: v('--fg-primary'),
        card: v('--bg-card'),
        popover: v('--bg-elev'),
        'popover-foreground': v('--fg-primary'),
        primary: { DEFAULT: v('--brand'), foreground: '#ffffff' },
        secondary: { DEFAULT: v('--bg-muted'), foreground: v('--fg-secondary') },
        'secondary-foreground': v('--fg-primary'),
        muted: { DEFAULT: v('--bg-muted'), foreground: v('--fg-muted') },
        'muted-foreground': v('--fg-muted'),
        accent: { DEFAULT: v('--bg-card-hover'), foreground: v('--fg-primary') },
        'accent-foreground': v('--fg-primary'),
        destructive: { DEFAULT: v('--danger'), foreground: '#ffffff' },
        'destructive-foreground': '#ffffff',
        input: v('--border'),
        ring: v('--brand'),
        topbar: 'rgb(var(--topbar) / 0.72)',
      },
      fontFamily: {
        sans: ['Inter', '-apple-system', 'BlinkMacSystemFont', 'Segoe UI', 'Roboto', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'Fira Code', 'ui-monospace', 'SFMono-Regular', 'Menlo', 'monospace'],
      },
      borderRadius: {
        lg: 'var(--radius-lg)',
        md: 'var(--radius-md)',
        sm: 'var(--radius-sm)',
        xl: 'var(--radius-xl)',
        full: '9999px',
      },
      boxShadow: {
        sm: '0 1px 2px 0 rgba(0,0,0,.35)',
        md: '0 4px 8px -2px rgba(0,0,0,.5)',
        lg: '0 12px 24px -6px rgba(0,0,0,.55)',
        'glow-blue': '0 0 0 1px rgba(59,130,246,.35), 0 0 18px rgba(59,130,246,.35)',
        'glow-green': '0 0 0 1px rgba(16,185,129,.35), 0 0 18px rgba(16,185,129,.35)',
      },
      keyframes: {
        'count-up-fade': {
          from: { opacity: '0', transform: 'translateY(6px)' },
          to: { opacity: '1', transform: 'translateY(0)' },
        },
        shimmer: {
          '0%': { backgroundPosition: '-400px 0' },
          '100%': { backgroundPosition: '400px 0' },
        },
        'pulse-glow-green': {
          '0%, 100%': { boxShadow: '0 0 0 0 rgba(16,185,129,.55)' },
          '50%': { boxShadow: '0 0 0 6px rgba(16,185,129,0)' },
        },
        'pulse-glow-blue': {
          '0%, 100%': { boxShadow: '0 0 0 0 rgba(59,130,246,.55)' },
          '50%': { boxShadow: '0 0 0 6px rgba(59,130,246,0)' },
        },
      },
      animation: {
        'count-up': 'count-up-fade 600ms cubic-bezier(0.16, 1, 0.3, 1) both',
        shimmer: 'shimmer 1.4s linear infinite',
        'pulse-glow-green': 'pulse-glow-green 2s infinite',
        'pulse-glow-blue': 'pulse-glow-blue 2s infinite',
      },
    },
  },
  plugins: [],
} satisfies Config
