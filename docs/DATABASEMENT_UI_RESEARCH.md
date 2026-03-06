# Databasement UI/UX Analysis — David-Crty/databasement

> **Repository**: https://github.com/David-Crty/databasement  
> **Stack**: Laravel 12 + Livewire 4 + Alpine.js + TailwindCSS v4 + DaisyUI v5 + Mary UI v2.4 + Chart.js  
> **Analysis Date**: February 8, 2026

---

## 1. Technology Stack

### Frontend Framework

| Layer | Package | Version |
|-------|---------|---------|
| CSS Framework | TailwindCSS | ^4.1.16 |
| UI Component Library | **DaisyUI** | ^5.5.5 |
| Blade Component Library | **Mary UI** (robsontenorio/mary) | ^2.4 |
| Reactive UI | **Livewire** | ^4.0 |
| Client-side JS | **Alpine.js** (bundled with Livewire) | — |
| Charts | **Chart.js** | ^4.5.1 |
| Build | Vite 7 + laravel-vite-plugin | — |
| Icons | blade-fontawesome + blade-devicons + blade-icons | — |
| Font | Instrument Sans | — |

### Entry Points (vite.config.js)
```js
input: ['resources/css/app.css', 'resources/js/app.js']
```

### CSS Setup (resources/css/app.css)
```css
@import 'tailwindcss';

@plugin "daisyui" {
    themes: all;     /* ALL DaisyUI themes enabled */
}

@source '../views';
@source '../../vendor/laravel/framework/src/Illuminate/Pagination/resources/views/*.blade.php';
@source '../../vendor/robsontenorio/mary/src/View/Components/**/*.php';
@source '../../vendor/masmerise/livewire-toaster/resources/views/*.blade.php';

@theme {
    --font-sans: 'Instrument Sans', ui-sans-serif, system-ui, sans-serif,
        'Apple Color Emoji', 'Segoe UI Emoji', 'Segoe UI Symbol', 'Noto Color Emoji';
}
```

**Key design decision**: No custom color theme defined — relies entirely on DaisyUI's built-in theme system with `data-theme` attribute on `<html>`. Default theme is `"dark"`.

---

## 2. Dark Theme / Theming System

### How Themes Work
- `<html data-theme="dark">` is the default, hardcoded in both layout files
- Theme is **stored in `localStorage`** and applied via a `<script>` block BEFORE page render (flash-prevention)
- Theme persists across Livewire SPA navigations via `livewire:navigated` event

```js
function applyTheme() {
    const savedTheme = localStorage.getItem('theme');
    if (savedTheme) {
        document.documentElement.setAttribute('data-theme', savedTheme);
    }
}
applyTheme();
document.addEventListener('livewire:navigated', () => {
    applyTheme();
});
```

### Available Themes (Appearance Page)
**All 30 DaisyUI themes** are offered to the user:
```
dark, light, cupcake, bumblebee, emerald, corporate, synthwave, retro,
cyberpunk, valentine, halloween, garden, forest, aqua, lofi, pastel,
fantasy, wireframe, black, luxury, dracula, cmyk, autumn, business,
acid, lemonade, night, coffee, winter, dim, nord, sunset
```

### DaisyUI "dark" Theme Color Palette (CSS Variables)
DaisyUI's `dark` theme automatically provides these CSS variables:
- `--color-base-100` → `#1d232a` (main background)
- `--color-base-200` → `#191e24` (slightly darker - body bg)
- `--color-base-300` → `#15191e` (borders, separators)
- `--color-base-content` → `#a6adba` (default text)
- `--color-primary` → `#7480ff` (primary actions)
- `--color-secondary` → `#ff52d9` (secondary)
- `--color-accent` → `#00cdb8` (accent)
- `--color-neutral` → `#2a323c` (neutral surfaces)
- `--color-info` → `#00b5ff` (info badges)
- `--color-success` → `#00a96e` (success badges)
- `--color-warning` → `#ffbe00` (warning/running)
- `--color-error` → `#ff5861` (error/failed)

### Theme Selector UI Pattern (appearance.blade.php)
Uses a **color grid preview** for each theme:
```html
<div class="rounded-box grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5">
    @foreach($themes as $themeName)
        <div class="border-base-content/20 hover:border-base-content/40 overflow-hidden rounded-lg border outline-2 outline-offset-2 transition-all"
             :class="isActive('{{ $themeName }}') ? 'outline outline-base-content' : 'outline-transparent'"
             @click="setTheme('{{ $themeName }}')">
            <div data-theme="{{ $themeName }}" class="bg-base-100 text-base-content w-full cursor-pointer font-sans">
                <div class="grid grid-cols-5 grid-rows-3">
                    <div class="bg-base-200 col-start-1 row-span-2 row-start-1"></div>
                    <div class="bg-base-300 col-start-1 row-start-3"></div>
                    <div class="bg-base-100 col-span-4 col-start-2 row-span-3 row-start-1 flex flex-col gap-1 p-2">
                        <div class="font-bold">{{ $themeName }}</div>
                        <div class="flex flex-wrap gap-1">
                            <!-- Color swatches: primary, secondary, accent, neutral -->
                            <div class="bg-primary flex aspect-square w-5 items-center justify-center rounded lg:w-6">
                                <div class="text-primary-content text-sm font-bold">A</div>
                            </div>
                            <!-- ...secondary, accent, neutral swatches... -->
                        </div>
                    </div>
                </div>
            </div>
        </div>
    @endforeach
</div>
```

---

## 3. Sidebar / Navigation Structure

### Layout Architecture (layouts/app.blade.php)
Uses **Mary UI's `<x-main>` layout component** with a collapsible sidebar drawer pattern:

```
┌────────────────────────────────────────────┐
│ <x-nav> (mobile only, lg:hidden)           │
│   Brand + Hamburger menu                   │
├──────┬─────────────────────────────────────┤
│      │                                     │
│ SIDE │  <x-slot:content>                   │
│ BAR  │    Alert banners                    │
│      │    {{ $slot }} (page content)       │
│      │    <footer>                         │
│      │                                     │
├──────┴─────────────────────────────────────┤
│ <x-toast /> (DaisyUI toast notifications)  │
└────────────────────────────────────────────┘
```

### Mobile Navigation
```blade
<x-nav sticky class="lg:hidden">
    <x-slot:brand>
        <x-app-brand />
    </x-slot:brand>
    <x-slot:actions>
        <label for="main-drawer" class="lg:hidden me-3">
            <x-icon name="o-bars-3" class="cursor-pointer" />
        </label>
    </x-slot:actions>
</x-nav>
```

### Sidebar Component
```blade
<x-slot:sidebar drawer="main-drawer" collapsible class="bg-base-100 lg:bg-inherit">
    <div class="flex flex-col h-full">
        <!-- BRAND -->
        <x-app-brand class="px-5 pt-4" />

        <!-- MAIN MENU -->
        <x-menu activate-by-route class="flex-1">
            <x-menu-separator />
            <x-menu-item title="Dashboard"      icon="o-home"          link="{{ route('dashboard') }}"         wire:navigate />
            <x-menu-item title="Database Servers" icon="o-server-stack" link="{{ route('database-servers.index') }}" wire:navigate />
            <livewire:menu.jobs-menu-item />   <!-- Dynamic Livewire component for Jobs -->
            <x-menu-item title="Volumes"         icon="o-circle-stack"  link="{{ route('volumes.index') }}"     wire:navigate />
            <x-menu-item title="Users"           icon="o-users"         link="{{ route('users.index') }}"       wire:navigate />
            <x-menu-separator />
            <x-menu-item title="Configuration"   icon="o-cog-6-tooth"   link="{{ route('configuration.index') }}" wire:navigate />
            <x-menu-item title="API Docs"        icon="o-document-text" link="{{ route('scramble.docs.ui') }}" no-wire-navigate />
            <x-menu-item title="API Tokens"      icon="o-key"           link="{{ route('api-tokens.index') }}" wire:navigate />
        </x-menu>

        <!-- USER SECTION (pinned to bottom) -->
        <x-menu activate-by-route class="mt-auto">
            <x-menu-sub title="{{ $user->name }}" icon="o-user">
                <x-menu-item title="Appearance"    icon="o-paint-brush"  ... />
                <x-menu-item title="Profile"       icon="o-user"         ... />
                <x-menu-item title="Password"      icon="o-key"          ... />
                <x-menu-item title="Two-Factor Auth" icon="o-shield-check" ... />
                <!-- Logout form -->
            </x-menu-sub>
        </x-menu>
    </div>
</x-slot:sidebar>
```

### Sidebar Design Patterns
- **`activate-by-route`** — Mary UI auto-highlights the active menu item based on current route
- **`wire:navigate`** — Livewire SPA navigation (no full page reload)
- **`collapsible`** — Sidebar can collapse to icon-only mode on desktop
- **`drawer="main-drawer"`** — On mobile, sidebar becomes a DaisyUI drawer (slide-out panel)
- **`flex flex-col h-full`** — User menu pinned to bottom with `mt-auto`
- Icons use **Heroicons outline style** (`o-` prefix from blade-icons)

### App Brand Component (components/app-brand.blade.php)
```blade
<a href="/" wire:navigate>
    <!-- Visible when sidebar expanded -->
    <div class="hidden-when-collapsed">
        <div class="flex items-center gap-3 w-fit">
            <x-logo-icon class="w-10 h-10" />
            <span class="font-bold me-3 tracking-wider uppercase
                bg-gradient-to-r from-cyan-400 via-purple-500 to-purple-600
                bg-clip-text text-transparent
                drop-shadow-[0_0_10px_rgba(6,182,212,0.3)]">
                Databasement
            </span>
        </div>
    </div>
    <!-- Visible when sidebar collapsed -->
    <div class="display-when-collapsed hidden mx-5 mt-5 mb-1 h-[28px]">
        <x-logo-icon class="w-7 h-7" />
    </div>
</a>
```

**Brand gradient**: `from-cyan-400 via-purple-500 to-purple-600` with a cyan glow drop-shadow.

### Logo Icon (components/logo-icon.blade.php)
A custom SVG of **3 stacked isometric database layers** with gradient fills:
- **Top layer** (brightest): `#00d4ff` → `#a855f7` → `#1a1a2e`
- **Middle layer**: `#007a99` → `#6b21a8` → `#0f0f1a`
- **Bottom layer** (darkest): `#004455` → `#4c1d95` → `#080812`
- **Stroke color**: `#06b6d4` (cyan-500)

---

## 4. Dashboard Layout

### Structure (livewire/dashboard.blade.php)
```blade
<div>
    <x-header title="Dashboard" separator />

    <div class="flex flex-col gap-6">
        <!-- Row 1: Stats Cards (3 columns) -->
        <livewire:dashboard.stats-cards />

        <!-- Row 2: Jobs Activity Chart (full width) -->
        <livewire:dashboard.jobs-activity-chart />

        <!-- Row 3: Latest Jobs (2/3) + Charts (1/3) -->
        <div class="grid gap-6 lg:grid-cols-3 items-start">
            <div class="lg:col-span-2 h-full">
                <livewire:dashboard.latest-jobs />
            </div>
            <div class="grid grid-rows-2 gap-6 h-full">
                <livewire:dashboard.success-rate-chart />
                <livewire:dashboard.storage-distribution-chart />
            </div>
        </div>
    </div>
</div>
```

### Visual Layout
```
┌─────────────────────────────────────────────────────────────┐
│ Dashboard                                                    │
├───────────────────┬──────────────────┬───────────────────────┤
│  📸 Snapshots     │  💾 Total Storage │  📊 Success Rate (30d)│
│  1,234 snapshots  │  24.5 GB          │  97.3%               │
│  [All verified ✓] │                    │  ⟳ 2 running         │
├───────────────────┴──────────────────┴───────────────────────┤
│ Jobs Activity — Last 14 days                                 │
│ ████ ███ ██████ █████ ... (stacked bar chart)                │
│ [Completed] [Failed] [Running] [Pending]                     │
├──────────────────────────────────────┬───────────────────────┤
│ Latest Jobs                          │ Job Status (30d)      │
│ ┌────────────────────────────────┐   │ [Doughnut Chart]      │
│ │ Backup  my-server / mydb  Done │   │                       │
│ │ Restore staging / app   Running│   ├───────────────────────┤
│ │ ...                             │   │ Storage by Volume     │
│ └────────────────────────────────┘   │ [Doughnut Chart]      │
│ → View all jobs                      │                       │
└──────────────────────────────────────┴───────────────────────┘
```

---

## 5. Stats Cards (dashboard/stats-cards.blade.php)

### Loading State (Skeleton)
```blade
<div wire:init="load" class="grid gap-4 md:grid-cols-3">
    @if(!$loaded)
        @for($i = 0; $i < 3; $i++)
            <x-card class="animate-pulse">
                <div class="flex items-center gap-4">
                    <div class="w-12 h-12 rounded-lg bg-base-300"></div>
                    <div class="flex-1">
                        <div class="h-4 w-20 bg-base-300 rounded mb-2"></div>
                        <div class="h-6 w-16 bg-base-300 rounded"></div>
                    </div>
                </div>
            </x-card>
        @endfor
    @else
```

### Card 1: Snapshots
- Header: `text-xs font-semibold uppercase tracking-wider text-base-content/50`
- Count: `text-3xl font-bold tabular-nums`
- Unit label: `text-sm text-base-content/40`
- Badges:
  - Missing: `badge badge-warning badge-sm` with link to filtered view
  - Verified: `badge badge-success badge-sm` with check icon
  - Partial: `badge badge-ghost badge-sm` with clock icon
- Verify button: `btn-ghost btn-xs text-base-content/60`

### Card 2: Total Storage
- Icon container: `w-12 h-12 rounded-lg bg-secondary/10 flex items-center justify-center`
- Icon: `w-6 h-6 text-secondary` (o-circle-stack)
- Label: `text-sm text-base-content/70`
- Value: `text-2xl font-bold`

### Card 3: Success Rate
- Dynamic icon background:
  - ≥90%: `bg-success/10`, icon `text-success`
  - ≥70%: `bg-warning/10`, icon `text-warning`
  - <70%: `bg-error/10`, icon `text-error`
- Running indicator: `text-sm font-normal text-warning` with `<x-loading class="loading-xs" />`

---

## 6. Jobs Activity Chart (Bar Chart)

### Blade Template (dashboard/jobs-activity-chart.blade.php)
```blade
<x-card title="Jobs Activity" subtitle="Last 14 days" shadow>
    @if(!$loaded)
        <div class="h-48 flex items-center justify-center">
            <x-loading class="loading-lg" />
        </div>
    @else
        <div class="h-48" x-data="chart(@js($chart))">
            <canvas x-ref="canvas"></canvas>
        </div>
    @endif
</x-card>
```

### PHP Chart Data (app/Livewire/Dashboard/JobsActivityChart.php)
```php
$this->chart = [
    'type' => 'bar',
    'data' => [
        'labels' => $labels,  // ['Jan 25', 'Jan 26', ...]
        'datasets' => [
            [
                'label' => 'Completed',
                'data' => $completed,
                'backgroundColor' => '--color-success',  // DaisyUI CSS var
                'borderRadius' => 4,
            ],
            [
                'label' => 'Failed',
                'data' => $failed,
                'backgroundColor' => '--color-error',
                'borderRadius' => 4,
            ],
            [
                'label' => 'Running',
                'data' => $running,
                'backgroundColor' => '--color-warning',
                'borderRadius' => 4,
            ],
            [
                'label' => 'Pending',
                'data' => $pending,
                'backgroundColor' => '--color-info',
                'borderRadius' => 4,
            ],
        ],
    ],
    'options' => [
        'responsive' => true,
        'maintainAspectRatio' => false,
        'scales' => [
            'x' => ['stacked' => true, 'grid' => ['display' => false]],
            'y' => ['stacked' => true, 'beginAtZero' => true, 'ticks' => ['stepSize' => 1]],
        ],
        'plugins' => [
            'legend' => ['position' => 'bottom'],
        ],
    ],
];
```

### CSS Variable Resolution (app.js)
Colors like `'--color-success'` are **resolved at runtime** to actual color values:
```js
Alpine.data('chart', (config, options = {}) => ({
    init() {
        this.$nextTick(() => {
            const resolveColor = (color) => {
                if (color && color.startsWith('--')) {
                    return getComputedStyle(document.documentElement)
                        .getPropertyValue(color).trim();
                }
                return color;
            };
            config.data.datasets.forEach(dataset => {
                if (Array.isArray(dataset.backgroundColor)) {
                    dataset.backgroundColor = dataset.backgroundColor.map(resolveColor);
                } else {
                    dataset.backgroundColor = resolveColor(dataset.backgroundColor);
                }
            });
            new Chart(canvas, config);
        });
    }
}));
```

This means charts **automatically adapt to any DaisyUI theme** — switching to `light` or `nord` theme changes chart colors too.

---

## 7. Other Dashboard Charts

### Success Rate Chart (Doughnut)
```php
'type' => 'doughnut',
'data' => [
    'labels' => ['Completed', 'Failed', 'Running', 'Pending'],
    'datasets' => [[
        'data' => [$completed, $failed, $running, $pending],
        'backgroundColor' => ['--color-success', '--color-error', '--color-warning', '--color-info'],
        'borderWidth' => 0,
    ]],
],
'options' => [
    'cutout' => '60%',
    'plugins' => ['legend' => ['position' => 'bottom', 'labels' => ['usePointStyle' => true, 'padding' => 16]]],
],
```

### Storage Distribution Chart (Doughnut)
```php
'type' => 'doughnut',
'backgroundColor' => ['--color-primary', '--color-secondary', '--color-accent',
                       '--color-info', '--color-success', '--color-warning', '--color-error'],
// Special: formatBytes option enables byte-formatting tooltips
```

The `formatBytes` option triggers custom tooltip callbacks in the Alpine `chart` component.

---

## 8. Log Viewer (backup-job/_logs-modal.blade.php)

### Modal Container
```blade
<x-modal wire:model="showLogsModal"
         title="Job Logs"
         class="backdrop-blur"
         box-class="w-full sm:w-11/12 max-w-6xl max-h-[90vh]">
```

### Job Info Header
- Background: `bg-base-200 rounded-lg`
- Database icon + server/db name
- Status badge using `match()`:
  ```php
  $jobStatusBadge = match($jobStatus) {
      'completed' => ['label' => 'Completed', 'class' => 'badge-success'],
      'failed'    => ['label' => 'Failed',    'class' => 'badge-error'],
      'running'   => ['label' => 'Running',   'class' => 'badge-warning'],
      default     => ['label' => ucfirst($jobStatus), 'class' => 'badge-info'],
  };
  ```
- Compression & Volume type badges: `badge badge-outline gap-1.5`
- Timestamps: `text-xs sm:text-sm text-base-content/70`

### Log Table Structure
```
┌──────────────────────────────────────────────────────────────────────┐
│ Date (w-36)    │ Type (w-24, center) │ Message (flex-1)             │
├──────────────────────────────────────────────────────────────────────┤
│ 14:32:05 Jan 25│ [command]           │ $ mysqldump --databases...   │
│ 14:32:06 Feb 1 │ [info]              │ Starting backup process      │
│ 14:32:10 Feb 1 │ [success]           │ Backup completed successfully│
│ 14:32:10 Feb 1 │ [error]             │ Connection refused           │
└──────────────────────────────────────────────────────────────────────┘
```

### Table Header (Desktop)
```blade
<div class="hidden sm:flex bg-base-200 text-sm font-semibold border-b border-base-300">
    <div class="w-36 flex-shrink-0 px-4 py-3">Date</div>
    <div class="w-24 flex-shrink-0 py-3 text-center">Type</div>
    <div class="flex-1 px-4 py-3">Message</div>
</div>
```

### Log Row Pattern
Each log row uses:
- **Left border color** based on state (4px `border-l-4`):
  ```php
  $borderClass = match(true) {
      $isError   => 'border-l-error bg-error/5',
      $isWarning => 'border-l-warning',
      $isSuccess => 'border-l-success',
      default    => 'border-l-info',
  };
  ```

- **Type badges** with DaisyUI styling:
  ```php
  $badgeClass = match($logLevel) {
      'error'   => 'badge-error badge-sm',
      'warning' => 'badge-warning badge-sm',
      'success' => 'badge-success badge-sm',
      'command' => 'badge-neutral badge-sm',
      default   => 'badge-info badge-sm',
  };
  ```

- **Timestamp**: `font-mono text-xs sm:text-sm text-base-content/70 whitespace-nowrap`

- **Command display**: 
  ```blade
  <code class="bg-neutral text-neutral-content px-2 py-1 rounded text-xs font-mono truncate block">
      <span class="text-success">$</span> {{ $log['command'] }}
  </code>
  ```

- **Expandable details** using Mary UI's `<x-collapse>`:
  - Command output: `<div class="mockup-code text-sm max-h-64 overflow-auto">` (DaisyUI mockup-code)
  - Exit code: `badge-success badge-sm` (0) or `badge-error badge-sm` (non-zero)
  - Context: `bg-base-300 p-3 rounded font-mono text-xs overflow-x-auto`

### Deferred Loading Pattern
All dashboard components use a `WithDeferredLoading` trait:
```blade
<div wire:init="load">
    @if(!$loaded)
        <!-- skeleton / loading spinner -->
    @else
        <!-- actual content -->
    @endif
</div>
```

---

## 9. Jobs Table (backup-job/index.blade.php)

### Structure
```blade
<x-card shadow>
    <x-table :headers="$headers" :rows="$jobs" :sort-by="$sortBy"
             with-pagination
             :row-decoration="['bg-warning/5' => fn($job) => $job->snapshot && !$job->snapshot->file_exists]">
```

### Table Cell Patterns
- **Type column**: `badge-primary` (Backup) or `badge-secondary` (Restore)
- **Date column**: Primary text + `text-sm text-base-content/70` for relative time
- **Server column**: Database type icon + Server name + DB name
- **Status column**: Same badge pattern as dashboard (success/error/warning/info)
- **Duration column**: `font-mono text-sm` (running shows `text-warning`)
- **Size column**: `font-mono text-sm`
- **Actions**: Ghost buttons for Download (`text-info`), View Logs, Delete (`text-error`)

### Row Decoration
Rows with missing backup files get `bg-warning/5` background tint.

---

## 10. Key Design Patterns Summary

### Component Hierarchy
```
Mary UI Components (Blade)
├── <x-main>           → Main layout with sidebar
├── <x-nav>            → Top navigation bar  
├── <x-menu>           → Sidebar menu
├── <x-menu-item>      → Menu entries with icons
├── <x-menu-sub>       → Submenu (user dropdown)
├── <x-card>           → Card container with shadow
├── <x-header>         → Page headers with separator  
├── <x-table>          → Data tables with sorting/pagination
├── <x-badge>          → Status badges
├── <x-button>         → Buttons with icons/spinners
├── <x-modal>          → Modal dialogs
├── <x-alert>          → Alert banners
├── <x-collapse>       → Expandable sections
├── <x-toast>          → Toast notifications
├── <x-loading>        → Loading spinners
├── <x-icon>           → Heroicons
├── <x-select>         → Select dropdowns
└── <x-popover>        → Popover tooltips
```

### Color Convention
| Purpose | DaisyUI Class | In Dark Theme |
|---------|---------------|---------------|
| Page background | `bg-base-200` | `#191e24` |
| Card/sidebar background | `bg-base-100` | `#1d232a` |
| Borders/separators | `border-base-300` | `#15191e` |
| Primary text | `text-base-content` | `#a6adba` |
| Muted text | `text-base-content/70` | 70% opacity |
| Very muted text | `text-base-content/50` | 50% opacity |
| Faintest text | `text-base-content/40` | 40% opacity |
| Backup badges | `badge-primary` | purple-blue |
| Restore badges | `badge-secondary` | pink |
| Success/Completed | `badge-success` / `text-success` | green |
| Error/Failed | `badge-error` / `text-error` | red |
| Warning/Running | `badge-warning` / `text-warning` | amber |
| Info/Pending | `badge-info` / `text-info` | blue |
| Commands | `bg-neutral text-neutral-content` | dark gray |

### Loading/Skeleton Pattern
```blade
<!-- Skeleton cards -->
<div class="animate-pulse">
    <div class="h-4 w-20 bg-base-300 rounded mb-2"></div>
    <div class="h-6 w-16 bg-base-300 rounded"></div>
</div>

<!-- Spinner -->
<x-loading class="loading-lg" />

<!-- Inline running indicator -->
<x-loading class="loading-spinner loading-xs" />
```

### Real-time Updates
- `wire:poll.5s` on jobs table and latest jobs
- `wire:init="load"` for deferred loading
- Status badges dynamically show spinners for "running" state

### Responsive Breakpoints
- Mobile-first approach
- `sm:` — Small tablets (640px+)
- `lg:` — Desktop (1024px+)
- Mobile: stacked 2-line layout, hidden columns
- Desktop: horizontal 1-line layout, all columns visible
