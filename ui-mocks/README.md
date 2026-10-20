# Dockermanage · UI 设计稿 v0.1

> 单主机自用的 Docker 管理面板 · 设计稿阶段 · 暗色优先
> 技术栈目标：**Go + React + shadcn/ui + Tailwind CSS**

---

## 📁 文件清单

```
ui-mocks/
├── index.html             # Dashboard（首屏门面，最详细）
├── containers.html        # 容器列表（紧凑卡片 + 筛选 + 批量操作）
├── container-detail.html  # 容器详情（顶部概览 + tabs：Stats/Logs/Terminal/Inspect/Files/Env）
├── compose.html           # Compose 编辑器（左 YAML + 右实时校验 + 部署）
├── README.md              # 本文件
├── .shoot.sh              # 无头 Chrome 批量截图脚本（自检用）
├── assets/
│   └── styles.css         # 共享样式 + CSS 变量（设计 token）
└── screenshots/           # 1440×900 桌面版渲染截图（4 页）
```

双击 `index.html` 即可在浏览器打开。无需 Node 依赖。

---

## ✅ 渲染自检记录（2026-10-20）

用无头 Chrome（`--window-size=1440,900`）对 4 个页面逐页截图验证，另在 640px 窄屏复检响应式。
Console 无 JS 错误（仅 Tailwind CDN 的 production 提示，属预期）。

初稿发现并已修复的 3 个问题：

| # | 问题 | 根因 | 修法 |
|---|---|---|---|
| 1 | ⌘K 命令面板默认展开、裁切在页面顶端 | `.cmdk-overlay { display:flex }` 盖掉了 Tailwind 的 `.hidden { display:none }` | 提高特异性：`.cmdk-overlay.hidden { display:none }` |
| 2 | 顶部搜索框被挤成竖排单字 | flex item 缺 `min-w-0`，文案 span 撑到 min-content | 文案加 `min-w-0 truncate`，窄屏 `hidden sm:block`；按钮改 `sm:flex-1 sm:max-w-xl` |
| 3 | 窄屏侧栏仍占 240px 布局宽度，且无入口 | 只有 index.html 带 `sidebar-fixed` 类；窄屏无汉堡菜单 | 4 页统一加 `sidebar-fixed`；窄屏改 `position:fixed` 抽屉 + 汉堡按钮 + 毛玻璃遮罩 + ESC 关闭 |

> **关键坑位（实现前端时注意）**：Tailwind Play CDN 把样式 `<style>` 注入到 `document.head` **末尾**，
> 晚于 `<link rel=stylesheet>`。因此**同特异性下 Tailwind 工具类会盖掉自定义 CSS**。
> 自定义类需要覆盖 Tailwind 工具类时，必须提高特异性（如 `aside.sidebar-fixed`）或用 `!important`。

复现截图：

```bash
cd ui-mocks && ./.shoot.sh 1440 900   # 桌面版
cd ui-mocks && ./.shoot.sh 640 800    # 窄屏版
```

---

## 🎨 设计 Token

### 配色（暗色 · shadcn/ui 风格 + 微蓝调）

| 用途 | Token | Tailwind | 说明 |
|---|---|---|---|
| 页面底 | `--bg-base` | `bg-bg-base` | `#09090b` zinc-950 |
| 主背景 | `--bg-elev` | `bg-bg-elev` | `#0f0f12` |
| 卡片 | `--bg-card` | `bg-bg-card` | `#18181b` zinc-900 |
| 卡片 hover | `--bg-card-hover` | `bg-bg-cardHover` | `#1f1f23` |
| 分隔/输入 | `--bg-muted` | `bg-bg-muted` | `#27272a` zinc-800 |
| 边框 | `--border` | `border-border` | `#27272a` |
| 边框强 | `--border-strong` | `border-borderStrong` | `#3f3f46` zinc-700 |
| 主文字 | `--fg-primary` | `text-fg-primary` | `#fafafa` zinc-50 |
| 次文字 | `--fg-secondary` | `text-fg-secondary` | `#d4d4d8` zinc-300 |
| 弱文字 | `--fg-muted` | `text-fg-muted` | `#a1a1aa` zinc-400 |
| 提示 | `--fg-subtle` | `text-fg-subtle` | `#71717a` zinc-500 |
| **主色** | `--primary` | `bg-brand` | `#3b82f6` blue-500 |
| 主色 hover | `--primary-hover` | `bg-brand-hover` | `#2563eb` |
| 主色软 | `--primary-soft` | `bg-brand-soft` | `rgba(59,130,246,0.1)` |
| 成功 | `--success` | — | `#10b981` emerald-500 |
| 警告 | `--warning` | — | `#f59e0b` amber-500 |
| 危险 | `--danger` | — | `#ef4444` red-500 |
| 信息 | `--info` | — | `#06b6d4` cyan-500 |

> 全部 CSS 变量定义见 [`assets/styles.css`](assets/styles.css) 的 `:root` 块。

### 字体

- **Sans**：`Inter`（Google Fonts CDN） — UI 正文、按钮、标签
- **Mono**：`JetBrains Mono` — 终端、日志、镜像名、代码块、数字

```html
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500;600&display=swap" rel="stylesheet">
```

### 圆角

| 用途 | Token | 值 |
|---|---|---|
| 标签/按钮 | `--radius-full` | `9999px` |
| 按钮 | `--radius-md` | `8px` |
| 输入 | `--radius-md` | `8px` |
| 卡片 | `--radius-lg` | `12px` |
| 大面板/对话框 | `--radius-xl` | `16px` |

### 间距基准（4px）

`--space-1..10` = `4 / 8 / 12 / 16 / 20 / 24 / 32 / 40`px。Tailwind 默认 `p-1..p-10` 已覆盖大部分场景。

### 阴影（暗色下克制）

| Token | 用途 |
|---|---|
| `--shadow-sm` | 卡片 hover |
| `--shadow-md` | 下拉/小浮层 |
| `--shadow-lg` | ⌘K 命令面板 |
| `--shadow-glow-blue` | 焦点/激活 |
| 关键动效 | 状态指示呼吸感 — `box-shadow: 0 0 0 0→6px` |

---

## 🧩 组件规范

### 1. 状态徽标（badge）

```html
<span class="badge badge-running"><span class="dot-running"></span>running</span>
<span class="badge badge-error"><span class="dot-error"></span>exited</span>
<span class="badge badge-paused"><span class="dot-paused"></span>paused</span>
<span class="badge badge-stopped"><span class="dot-stopped"></span>stopped</span>
```

- **运行中** — `emerald` 软底 + 绿点带 `pulse-glow-green` 呼吸动画
- **异常** — `red` 软底 + 红点
- **暂停** — `amber` 软底 + 黄点
- **停止** — `zinc` 软底 + 灰点

### 2. 资源卡片（stat-card）

```html
<div class="elev-card stat-card p-4">
  <div class="flex items-center justify-between">
    <span class="text-sm text-fg-muted">CPU</span>
    <span class="text-[11px] text-fg-subtle font-mono">16 cores</span>
  </div>
  <div class="mt-3 flex items-end gap-2">
    <span class="text-3xl font-semibold count-up font-mono">23.4</span>
    <span class="text-sm text-fg-muted mb-1">%</span>
  </div>
  <div class="mt-3 meter"><span style="width:23.4%"></span></div>
</div>
```

要点：
- 数字大字号 + `count-up` 入场动效（`@keyframes count-up-fade`）
- 进度条 `meter` 组件：默认蓝，>60% 切 amber，>85% 切 red

### 3. 终端 / 日志（terminal）

```html
<div class="terminal">
  <span class="t-prompt">root@id</span>
  <span class="t-info">[INFO]</span>
  <span class="t-warn">[WARN]</span>
  <span class="t-error">[ERROR]</span>
  <span class="t-key">key</span>
  <span class="t-num">42</span>
  <span class="t-str">"value"</span>
  <span class="t-dim">2026-10-16T07:30:00Z</span>
</div>
```

| 类 | 颜色 | 用途 |
|---|---|---|
| `.t-prompt` | cyan `#06b6d4` | shell prompt |
| `.t-info` | emerald `#10b981` | INFO 日志 / 成功 |
| `.t-warn` | amber `#f59e0b` | WARN 日志 |
| `.t-error` | red `#ef4444` | ERROR 日志 |
| `.t-dim` | zinc-500 `#71717a` | 时间戳、注释 |
| `.t-key` | violet `#a78bfa` | 字段名、关键字 |
| `.t-num` | amber `#fbbf24` | 数字 |
| `.t-str` | emerald `#10b981` | 字符串 |

### 4. ⌘K 命令面板

- 顶部居中（`padding-top: 14vh`），圆角 `xl`
- 背景半透明 + `backdrop-filter: blur(6px)` 蒙层
- 选中项 `.cmdk-item.is-active` 用 `bg-card-hover` 强调
- 右侧用 `.kbd` 显示快捷键提示
- 键盘：⌘K 唤起 / ESC 关闭 / ↑↓ 选择 / ↵ 确认

### 5. 容器卡片（dashboard 列表）

```html
<a class="surface-card surface-card-hover p-3.5 block">
  <div class="flex items-start gap-3">
    <div class="w-9 h-9 rounded-md bg-emerald-500/15 flex items-center justify-center">
      <i data-lucide="globe" class="w-4 h-4 text-emerald-400"></i>
    </div>
    <div class="flex-1">
      <span class="font-medium">nginx-proxy</span>
      <span class="badge badge-running">running</span>
      <div class="text-[11px] text-fg-subtle font-mono">nginx:1.27-alpine · :80→:80</div>
      <div class="grid grid-cols-3 gap-2">CPU / MEM / NET</div>
    </div>
  </div>
</a>
```

每个容器一个**品牌色 icon 块**（不靠 emoji）：
nginx / postgres / redis / qbittorrent / jellyfin / nextcloud / homeassistant / adguard / transmission / vaultwarden / gitea / portainer 全部映射到 lucide 图标。

### 6. 标签页（tabs）

下划线型，激活态 `border-bottom: 2px solid var(--primary)` + `color: var(--fg-primary)`。

---

## ✨ 视觉重点实现说明

| 重点 | 实现 |
|---|---|
| 暗色统一 | 全部 token 在 `:root` 集中维护，Tailwind config 同步注入 |
| 呼吸感指示 | `.dot-running` `animation: pulse-glow-green 2s infinite` |
| 数字滚动入场 | `.count-up` 类 + JS `requestAnimationFrame` 渐增 |
| 骨架屏 | `.skeleton` 类（shimmer 渐变动画） |
| 终端 ANSI 颜色 | `.terminal` + 7 个语义色类 |
| 实时折线图 | Chart.js（CDN），蓝色渐变填充 |
| ⌘K 命令面板 | 顶部居中 + 蒙层 + 快捷键提示 |
| 焦点态 | `:focus-visible` 蓝色 outline + 偏移 |
| 响应式 | `xl:col-span-*` 三列 → 单列；侧栏在 `<1024px` 隐藏 |

---

## 🖼️ 截图说明（Mac）

设计稿交付后，可以用 `Cmd+Shift+4` 截图。建议截取以下画面：

| # | 页面 | 路径 | 重点截取 |
|---|---|---|---|
| 1 | Dashboard 完整 | `index.html` | 顶部到事件流 |
| 2 | Dashboard 资源卡片 | `index.html` | 顶部 4 个 stat-card |
| 3 | Dashboard 实时图表 | `index.html` | 中段 chart 区 |
| 4 | ⌘K 命令面板 | `index.html`（按 ⌘K 唤起） | 顶部居中面板 |
| 5 | 容器列表 | `containers.html` | 表头 + 第一屏 |
| 6 | 容器列表选中态 | `containers.html` | 蓝色批量操作栏 |
| 7 | 容器详情 Stats | `container-detail.html` | 4 个 mini chart |
| 8 | 容器详情 Terminal | `container-detail.html` | 切换到 Terminal tab |
| 9 | 容器详情 Logs | `container-detail.html` | 切换到 Logs tab |
| 10 | 容器详情 Inspect | `container-detail.html` | 切换到 Inspect tab |
| 11 | Compose 编辑器 | `compose.html` | YAML + 校验面板 |

**截图步骤（Mac）**：

1. 在 Safari / Chrome 中打开目标 HTML
2. 浏览器建议用暗色模式：`系统设置 → 外观 → 深色`
3. 窗口分辨率建议 1440 × 900 或更大
4. 按 `Cmd+Shift+4` → 按 `空格` 切窗口截图模式 → 点击窗口
5. 默认保存到桌面，文件名为「屏幕截图 …」，可改名为 `01-dashboard-full.png` 等

可选：在 Chrome DevTools → Rendering → Emulate CSS media `prefers-color-scheme: dark`，强制暗色预览。

---

## 🔧 技术细节

### 引入方式（每个 HTML 头部）

```html
<!-- Tailwind CDN（最快，免安装） -->
<script src="https://cdn.tailwindcss.com"></script>
<script>
  tailwind.config = { darkMode:'class', theme:{ extend:{ colors:{...}, fontFamily:{...} }}}
</script>

<!-- 图标（lucide） -->
<script src="https://unpkg.com/lucide@latest"></script>
<!-- 在 body 末尾调用 -->
<script>lucide.createIcons();</script>

<!-- 图表（Chart.js，仅 Dashboard / 详情页需要） -->
<script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.1/dist/chart.umd.min.js"></script>

<!-- 共享样式 -->
<link rel="stylesheet" href="assets/styles.css" />
```

### 暗色强制

```html
<html lang="zh-CN" class="dark">
```

设计稿默认 dark，后续如果要切 light，需在 `:root` 加 `[data-theme="light"]` override。

### 真实感 mock data

容器名（不出现 lorem ipsum）：

```
nginx-proxy / postgres-main / redis-cache / qbittorrent / jellyfin /
nextcloud / homeassistant / adguard-home / transmission / vaultwarden /
gitea / portainer-agent
```

镜像：

```
nginx:1.27-alpine
postgres:16
redis:7-alpine
ghcr.io/linuxserver/qbittorrent:latest
ghcr.io/linuxserver/jellyfin:latest
nextcloud:29-apache
ghcr.io/home-assistant/home-assistant:stable
ghcr.io/adguard/adguardhome:latest
transmission:4.0
vaultwarden/server:1.32
gitea/gitea:1.22
portainer/agent:2.21
```

---

## ⚠️ 已知约束 / 后续 TODO

- 设计稿未做 Light 主题（如需，告诉我加 `[data-theme="light"]`）
- Tailwind CDN 模式（生产应改 JIT 构建）
- 仅做 4 个核心屏幕：P1 功能（监控历史 / 事件流 / 文件浏览器 / 一键更新 / 告警 / ⌘K / 应用模板）部分入口已露出，详稿按需展开
- Compose 编辑器未做 Monaco / CodeMirror 集成，仅静态展示
- ⌘K 命令面板 mock 数据，后续接入真实索引
- 暂未考虑国际化（i18n），文案暂用中文

---

**v0.1 · 2026-10-16 · 设计稿就绪，等 owner 验收**
