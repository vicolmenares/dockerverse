# Databasement Repository Analysis

**Repository:** [David-Crty/databasement](https://github.com/David-Crty/databasement)  
**Purpose:** Self-hosted database backup management application for MySQL, PostgreSQL, and MariaDB  
**Live Demo:** [databasement-demo.crty.dev](https://databasement-demo.crty.dev/)

---

## 1. Tech Stack Overview

### Backend
| Technology | Version/Details |
|------------|----------------|
| **PHP** | 8.4+ |
| **Laravel** | 12.x |
| **Livewire** | 4.x (reactive components) |
| **Laravel Fortify** | Authentication scaffolding |
| **Laravel Sanctum** | API token authentication |
| **Laravel Octane + FrankenPHP** | Performance optimization |
| **Database Support** | SQLite, MySQL, PostgreSQL, MariaDB |

### Frontend
| Technology | Details |
|------------|---------|
| **Blade Templates** | Server-rendered templates |
| **Tailwind CSS** | v4.x |
| **DaisyUI** | UI component library via `@plugin "daisyui"` |
| **Mary UI** | Livewire component library (`robsontenorio/mary`) |
| **Alpine.js** | Lightweight JavaScript framework |
| **Chart.js** | Charting library |
| **Vite** | v7.x - Build tool |

### Key Libraries
```json
{
  "frontend": {
    "@tailwindcss/vite": "^4.1.16",
    "chart.js": "^4.5.1",
    "copy-to-clipboard": "^3.3.3"
  },
  "backend": {
    "livewire/livewire": "^4.0",
    "robsontenorio/mary": "^2.4",
    "laravel/fortify": "^1.30",
    "laravel/sanctum": "^4.2",
    "laravel/octane": "^2.13"
  }
}
```

---

## 2. UI/UX Patterns

### Sidebar Navigation Implementation

The sidebar uses Mary UI's `<x-main>` and `<x-menu>` components with these key features:

#### Layout Structure (`resources/views/layouts/app.blade.php`)

```blade
<x-main>
    {{-- SIDEBAR --}}
    <x-slot:sidebar drawer="main-drawer" collapsible class="bg-base-100 lg:bg-inherit">
        <div class="flex flex-col h-full">
            {{-- BRAND --}}
            <x-app-brand class="px-5 pt-4" />

            {{-- MAIN MENU --}}
            <x-menu activate-by-route class="flex-1">
                <x-menu-separator />
                <x-menu-item title="{{ __('Dashboard') }}" icon="o-home" link="{{ route('dashboard') }}" wire:navigate />
                <x-menu-item title="{{ __('Database Servers') }}" icon="o-server-stack" link="{{ route('database-servers.index') }}" wire:navigate />
                
                {{-- LIVEWIRE COMPONENT FOR DYNAMIC BADGE --}}
                <livewire:menu.jobs-menu-item />
                
                <x-menu-item title="{{ __('Volumes') }}" icon="o-circle-stack" link="{{ route('volumes.index') }}" wire:navigate />
                <x-menu-item title="{{ __('Users') }}" icon="o-users" link="{{ route('users.index') }}" wire:navigate />
                <x-menu-separator />
                <x-menu-item title="{{ __('Configuration') }}" icon="o-cog-6-tooth" link="{{ route('configuration.index') }}" wire:navigate />
                <x-menu-item title="{{ __('API Docs') }}" no-wire-navigate="true" icon="o-document-text" link="{{ route('scramble.docs.ui') }}" />
                <x-menu-item title="{{ __('API Tokens') }}" icon="o-key" link="{{ route('api-tokens.index') }}" wire:navigate />
            </x-menu>

            {{-- USER SECTION AT BOTTOM --}}
            @if($user = auth()->user())
                <x-menu activate-by-route class="mt-auto" title="">
                    <x-menu-sub title="{{ $user->name }}" icon="o-user">
                        <x-menu-item title="{{ __('Appearance') }}" icon="o-paint-brush" link="{{ route('appearance.edit') }}" wire:navigate />
                        {{-- More menu items... --}}
                        <form method="POST" action="{{ route('logout') }}" class="w-full">
                            @csrf
                            <x-button type="submit" class="w-full" icon="o-power">{{ __('Logout') }}</x-button>
                        </form>
                    </x-menu-sub>
                </x-menu>
            @endif
        </div>
    </x-slot:sidebar>

    <x-slot:content>
        {{ $slot }}
    </x-slot:content>
</x-main>
```

#### Key Sidebar Features

1. **Collapsible on Desktop** - `collapsible` attribute
2. **Drawer on Mobile** - `drawer="main-drawer"` with hamburger toggle
3. **Route-based Activation** - `activate-by-route` highlights current page
4. **SPA Navigation** - `wire:navigate` for smooth transitions
5. **Dynamic Badges** - Livewire components with `wire:poll` for real-time updates

#### Dynamic Menu Item with Badge (`app/Livewire/Menu/JobsMenuItem.php`)

```php
class JobsMenuItem extends Component
{
    public function getActiveJobsCountProperty(): int
    {
        return BackupJob::whereIn('status', ['running', 'pending'])->count();
    }

    public function render(): string
    {
        return <<<'HTML'
        <div wire:poll.5s>
            <x-menu-item
                title="{{ __('Jobs') }}"
                icon="o-queue-list"
                link="{{ route('jobs.index') }}"
                wire:navigate
                :badge="$this->activeJobsCount > 0 ? $this->activeJobsCount : null"
                badge-classes="badge-warning badge-soft"
            />
        </div>
        HTML;
    }
}
```

### Color Scheme & Styling

#### CSS Structure (`resources/css/app.css`)

```css
@import 'tailwindcss';

@plugin "daisyui" {
    themes: all;
}

@source '../views';
@source '../../vendor/robsontenorio/mary/src/View/Components/**/*.php';

@theme {
    --font-sans: 'Instrument Sans', ui-sans-serif, system-ui, sans-serif;
}
```

#### Theme Management

```javascript
// In layout head - persisted theme preference
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

#### Color Variables Used

- `--color-success` - Green for completed/success states
- `--color-error` - Red for failed/error states  
- `--color-warning` - Yellow/orange for running/warning states
- `--color-info` - Blue for pending/info states
- `--color-primary`, `--color-secondary`, `--color-accent` - Theme colors

---

## 3. Session Management

### Configuration (`config/session.php`)

```php
return [
    'driver' => env('SESSION_DRIVER', 'database'),  // Database-backed sessions
    'lifetime' => (int) env('SESSION_LIFETIME', 120),  // 2 hours default
    'expire_on_close' => env('SESSION_EXPIRE_ON_CLOSE', false),
    'encrypt' => env('SESSION_ENCRYPT', false),
    'http_only' => env('SESSION_HTTP_ONLY', true),  // Prevents JS access
    'same_site' => env('SESSION_SAME_SITE', 'lax'),
];
```

### Authentication System (`config/fortify.php`)

```php
return [
    'guard' => 'web',
    'home' => '/dashboard',
    'features' => [
        Features::registration(),
        Features::resetPasswords(),
        Features::twoFactorAuthentication([
            'confirm' => true,
            'confirmPassword' => true,
        ]),
    ],
    'limiters' => [
        'login' => 'login',      // Rate limiting on login
        'two-factor' => 'two-factor',
    ],
];
```

### Password Confirmation Timeout (`config/auth.php`)

```php
// Re-authentication required after 3 hours of inactivity
'password_timeout' => env('AUTH_PASSWORD_TIMEOUT', 10800),
```

### API Token Authentication (Sanctum)

```php
// API Token Management Component
#[Title('API Tokens')]
class Index extends Component
{
    public function createToken(): void
    {
        $this->validate();
        $token = Auth::user()->createToken($this->tokenName);
        $this->newToken = $token->plainTextToken;
        $this->tokenName = '';
        $this->showTokenModal = true;
    }

    public function deleteToken(): void
    {
        $token = PersonalAccessToken::findOrFail($this->deleteTokenId);
        if ($this->canDelete($token)) {
            $token->delete();
            $this->success(__('API token revoked successfully.'));
        }
    }
}
```

---

## 4. Log Viewer Implementation

### Log Display Pattern

The logs modal implements a sophisticated expandable log view with:

#### Data Structure

```php
// Logs are stored as JSON in BackupJob model
$logs = $this->selectedJob->getLogs();

// Each log entry structure:
[
    'timestamp' => '2024-02-07T10:30:00Z',
    'type' => 'command|info|error|warning',
    'message' => 'Log message text',
    'level' => 'info|error|warning|success',
    'command' => 'mysqldump --single-transaction...',  // For command type
    'output' => 'Command output text...',
    'exit_code' => 0,
    'duration_ms' => 1500,
    'context' => ['key' => 'value'],  // Optional context data
    'status' => 'running|completed'
]
```

#### Blade View Pattern (`_logs-modal.blade.php`)

```blade
<x-modal wire:model="showLogsModal" 
         @close="$wire.closeLogs()" 
         title="{{ __('Job Logs') }}" 
         class="backdrop-blur" 
         box-class="w-full sm:w-11/12 max-w-6xl max-h-[90vh]">
    
    {{-- Job Info Header --}}
    <div class="p-4 bg-base-200 rounded-lg space-y-2">
        <div class="flex items-center gap-3">
            <x-database-type-icon :type="$job->snapshot->database_type" class="w-6 h-6" />
            <div class="font-semibold">{{ $job->snapshot->databaseServer->name }}</div>
        </div>
        
        {{-- Status Badge --}}
        @if($jobStatus === 'running')
            <div class="badge badge-warning gap-1">
                <x-loading class="loading-spinner loading-xs" />
                {{ __('Running') }}
            </div>
        @else
            <x-badge :value="$jobStatusBadge['label']" :class="$jobStatusBadge['class']" />
        @endif
    </div>

    {{-- Logs Table with Collapsible Details --}}
    <div class="border border-base-300 rounded-lg overflow-hidden">
        <div class="max-h-[60vh] overflow-y-auto divide-y divide-base-300">
            @foreach($logs as $log)
                @php
                    $rowState = match(true) {
                        $isRunning => 'warning',
                        $isCommand && $log['exit_code'] !== 0 => 'error',
                        $isCommand && $log['exit_code'] === 0 => 'success',
                        default => $log['level'] ?? 'info',
                    };
                    
                    $borderClass = match(true) {
                        $isError => 'border-l-error bg-error/5',
                        $isWarning => 'border-l-warning',
                        $isSuccess => 'border-l-success',
                        default => 'border-l-info',
                    };
                @endphp
                
                <div class="flex border-l-4 {{ $borderClass }}">
                    <x-collapse :collapse-plus-minus="$hasDetails">
                        <x-slot:heading>
                            <span class="font-mono text-xs">{{ $timestamp->format('H:i:s M d') }}</span>
                            <x-badge :value="ucfirst($logLevel)" class="{{ $badgeClass }}" />
                            @if($isCommand)
                                <code class="bg-neutral text-neutral-content px-2 py-1 rounded text-xs">
                                    <span class="text-success">$</span> {{ $log['command'] }}
                                </code>
                            @else
                                <span class="text-sm">{{ $log['message'] }}</span>
                            @endif
                        </x-slot:heading>
                        
                        <x-slot:content>
                            @if($isCommand)
                                <div class="mockup-code text-sm max-h-64 overflow-auto">
                                    <pre data-prefix="$"><code>{{ $log['command'] }}</code></pre>
                                    @foreach(explode("\n", $log['output']) as $line)
                                        <pre data-prefix=">"><code>{{ $line }}</code></pre>
                                    @endforeach
                                </div>
                            @endif
                        </x-slot:content>
                    </x-collapse>
                </div>
            @endforeach
        </div>
    </div>
</x-modal>
```

### Filtering Mechanism

```php
class LatestJobs extends Component
{
    public string $statusFilter = 'all';

    public function statusOptions(): array
    {
        return [
            ['id' => 'all', 'name' => __('All')],
            ['id' => 'running', 'name' => __('Running')],
            ['id' => 'failed', 'name' => __('Failed')],
            ['id' => 'completed', 'name' => __('Completed')],
            ['id' => 'pending', 'name' => __('Pending')],
        ];
    }

    public function fetchJobs(): void
    {
        $query = BackupJob::query()
            ->with(['snapshot.databaseServer', 'restore.targetServer'])
            ->orderByRaw("CASE WHEN status = 'running' THEN 0 ELSE 1 END")
            ->orderBy('created_at', 'desc')
            ->limit(12);

        if ($this->statusFilter !== 'all') {
            $query->where('status', $this->statusFilter);
        }

        $this->jobs = $query->get();
    }
}
```

---

## 5. Terminal/SSH Implementation

**Note:** Databasement does not include an interactive terminal/SSH interface (no xterm.js). However, it has **SSH tunnel support** for database connections.

### SSH Tunnel Configuration

```php
// SSH Config Model Fields
class DatabaseServerSshConfig extends Model
{
    const SENSITIVE_FIELDS = ['password', 'private_key', 'key_passphrase'];
    
    protected $fillable = [
        'host',
        'port',
        'username',
        'auth_type',  // 'password' or 'key'
        'password',
        'private_key',
        'key_passphrase',
    ];
}

// Form handling for SSH
class DatabaseServerForm extends Form
{
    public bool $ssh_enabled = false;
    public string $ssh_config_mode = 'create';  // 'existing' or 'create'
    public ?string $ssh_config_id = null;
    public string $ssh_host = '';
    public int $ssh_port = 22;
    public string $ssh_username = '';
    public string $ssh_auth_type = 'password';
    public string $ssh_password = '';
    public string $ssh_private_key = '';
    public string $ssh_key_passphrase = '';

    public function testSshConnection(): void
    {
        $this->validate($this->getSshValidationRules());
        $sshConfig = $this->buildSshConfigForTest();
        $result = SshTunnelService::testConnection($sshConfig);
        $this->sshTestSuccess = $result['success'];
        $this->sshTestMessage = $result['message'];
    }
}
```

### For Dockerverse SvelteKit Application

If you need a terminal feature, consider using **xterm.js** with these patterns:

```typescript
// Recommended approach for SvelteKit
import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import { WebLinksAddon } from 'xterm-addon-web-links';

// Component pattern
<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  
  let terminalElement: HTMLDivElement;
  let terminal: Terminal;
  let fitAddon: FitAddon;

  onMount(() => {
    terminal = new Terminal({
      theme: {
        background: '#1e1e1e',
        foreground: '#ffffff',
      },
      cursorBlink: true,
      fontSize: 14,
    });
    
    fitAddon = new FitAddon();
    terminal.loadAddon(fitAddon);
    terminal.loadAddon(new WebLinksAddon());
    
    terminal.open(terminalElement);
    fitAddon.fit();
    
    // Connect to WebSocket for real-time terminal
    const ws = new WebSocket('ws://your-backend/terminal');
    terminal.onData(data => ws.send(data));
    ws.onmessage = (event) => terminal.write(event.data);
  });
  
  onDestroy(() => {
    terminal?.dispose();
  });
</script>

<div bind:this={terminalElement} class="h-full w-full"></div>
```

---

## 6. Charts and Graphs

### Chart.js Integration

#### Setup (`resources/js/app.js`)

```javascript
import Chart from 'chart.js/auto';

window.Chart = Chart;

document.addEventListener('alpine:init', () => {
    Alpine.data('chart', (config, options = {}) => ({
        init() {
            this.$nextTick(() => {
                const canvas = this.$refs.canvas;
                if (!canvas) return;

                // Resolve CSS custom properties to actual colors
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

                // Add byte formatting for storage charts
                if (options.formatBytes) {
                    const formatBytes = (bytes) => {
                        if (bytes === 0) return '0 B';
                        const k = 1024;
                        const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
                        const i = Math.floor(Math.log(bytes) / Math.log(k));
                        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
                    };

                    config.options.plugins.tooltip.callbacks = {
                        label: (context) => `${context.label}: ${formatBytes(context.raw)}`
                    };
                }

                new Chart(canvas, config);
            });
        }
    }));
});
```

#### Bar Chart - Jobs Activity (`app/Livewire/Dashboard/JobsActivityChart.php`)

```php
class JobsActivityChart extends Component
{
    use WithDeferredLoading;

    public array $chart = [];

    protected function loadContent(): void
    {
        $days = 14;
        $startDate = Carbon::now()->subDays($days - 1)->startOfDay();

        $jobs = BackupJob::where('created_at', '>=', $startDate)
            ->get()
            ->groupBy(fn ($job) => $job->created_at->format('Y-m-d'));

        $labels = [];
        $completed = [];
        $failed = [];
        $running = [];
        $pending = [];

        for ($i = 0; $i < $days; $i++) {
            $date = $startDate->copy()->addDays($i);
            $dateKey = $date->format('Y-m-d');
            $labels[] = $date->format('M j');

            $dayJobs = $jobs->get($dateKey, collect());
            $completed[] = $dayJobs->where('status', 'completed')->count();
            $failed[] = $dayJobs->where('status', 'failed')->count();
            $running[] = $dayJobs->where('status', 'running')->count();
            $pending[] = $dayJobs->where('status', 'pending')->count();
        }

        $this->chart = [
            'type' => 'bar',
            'data' => [
                'labels' => $labels,
                'datasets' => [
                    [
                        'label' => __('Completed'),
                        'data' => $completed,
                        'backgroundColor' => '--color-success',
                        'borderRadius' => 4,
                    ],
                    [
                        'label' => __('Failed'),
                        'data' => $failed,
                        'backgroundColor' => '--color-error',
                        'borderRadius' => 4,
                    ],
                    // ... more datasets
                ],
            ],
            'options' => [
                'responsive' => true,
                'maintainAspectRatio' => false,
                'scales' => [
                    'x' => ['stacked' => true, 'grid' => ['display' => false]],
                    'y' => ['stacked' => true, 'beginAtZero' => true],
                ],
                'plugins' => [
                    'legend' => ['position' => 'bottom'],
                ],
            ],
        ];
    }
}
```

#### Blade Template for Charts

```blade
<div wire:init="load">
    <x-card title="{{ __('Jobs Activity') }}" subtitle="{{ __('Last 14 days') }}" shadow>
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
</div>
```

#### Doughnut Chart - Storage Distribution

```php
// With byte formatting option
<div class="h-48" x-data="chart(@js($chart), { formatBytes: true })">
    <canvas x-ref="canvas"></canvas>
</div>
```

### For SvelteKit Application

Consider using **Chart.js** directly or a Svelte wrapper like `svelte-chartjs`:

```svelte
<script lang="ts">
  import { onMount } from 'svelte';
  import Chart from 'chart.js/auto';

  let canvas: HTMLCanvasElement;
  let chart: Chart;

  export let data: { labels: string[]; datasets: any[] };
  export let type: 'bar' | 'doughnut' | 'line' = 'bar';

  $: if (chart && data) {
    chart.data = data;
    chart.update();
  }

  onMount(() => {
    chart = new Chart(canvas, {
      type,
      data,
      options: {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
          legend: { position: 'bottom' }
        }
      }
    });

    return () => chart?.destroy();
  });
</script>

<canvas bind:this={canvas}></canvas>
```

---

## 7. Performance Optimizations

### Deferred Loading Pattern

```php
// Trait for lazy loading components
trait WithDeferredLoading
{
    public bool $loaded = false;

    public function load(): void
    {
        $this->loadContent();
        $this->loaded = true;
    }

    abstract protected function loadContent(): void;
}

// Usage in component
class StatsCards extends Component
{
    use WithDeferredLoading;

    protected function loadContent(): void
    {
        $this->totalSnapshots = Snapshot::count();
        $this->totalStorage = Formatters::humanFileSize((int) Snapshot::sum('file_size'));
        // ... more expensive queries
    }
}
```

#### Blade Pattern for Deferred Loading

```blade
<div wire:init="load" class="grid gap-4 md:grid-cols-3">
    @if(!$loaded)
        {{-- Skeleton loading state --}}
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
        {{-- Actual content --}}
    @endif
</div>
```

### Polling for Real-time Updates

```blade
{{-- Poll every 5 seconds --}}
<div wire:poll.5s="fetchJobs">
    {{-- Content that auto-refreshes --}}
</div>

{{-- Poll on init --}}
<div wire:init="load" wire:poll.5s="fetchJobs">
```

### Bundle Optimization (`vite.config.js`)

```javascript
import { defineConfig } from 'vite';
import laravel from 'laravel-vite-plugin';
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
    plugins: [
        laravel({
            input: ['resources/css/app.css', 'resources/js/app.js'],
            refresh: true,  // Hot reload during development
        }),
        tailwindcss(),
    ],
    server: {
        cors: true,
    },
});
```

### Caching Strategy

```php
// Using Laravel's cache for expensive operations
public function verifyFiles(): void
{
    $lock = Cache::lock('verify-snapshot-files', 300);  // 5 min lock

    if (! $lock->get()) {
        $this->warning(__('File verification is already running.'));
        return;
    }

    VerifySnapshotFileJob::dispatch();
    $this->success(__('File verification job dispatched.'));
}
```

---

## 8. Recommendations for SvelteKit + Go Application

### Sidebar Navigation (SvelteKit)

```svelte
<!-- components/Sidebar.svelte -->
<script lang="ts">
  import { page } from '$app/stores';
  import { slide } from 'svelte/transition';

  export let collapsed = false;

  const menuItems = [
    { title: 'Dashboard', icon: 'home', href: '/' },
    { title: 'Hosts', icon: 'server', href: '/hosts' },
    { title: 'Containers', icon: 'box', href: '/containers' },
    { title: 'Jobs', icon: 'queue', href: '/jobs', badge: 'runningCount' },
    { title: 'Volumes', icon: 'database', href: '/volumes' },
    { title: 'Users', icon: 'users', href: '/users' },
  ];

  $: currentPath = $page.url.pathname;
</script>

<aside class="flex flex-col h-full bg-base-100" class:w-64={!collapsed} class:w-16={collapsed}>
  <div class="p-4">
    <slot name="brand" />
  </div>

  <nav class="flex-1 overflow-y-auto">
    <ul class="menu">
      {#each menuItems as item}
        <li>
          <a 
            href={item.href}
            class:active={currentPath === item.href || currentPath.startsWith(item.href + '/')}
            class="flex items-center gap-3"
          >
            <Icon name={item.icon} class="w-5 h-5" />
            {#if !collapsed}
              <span>{item.title}</span>
              {#if item.badge}
                <span class="badge badge-warning badge-sm ml-auto">{$store[item.badge]}</span>
              {/if}
            {/if}
          </a>
        </li>
      {/each}
    </ul>
  </nav>

  <div class="mt-auto p-4">
    <slot name="user" />
  </div>
</aside>
```

### Deferred Loading (SvelteKit)

```svelte
<script lang="ts">
  import { onMount } from 'svelte';

  let loaded = false;
  let data: any = null;

  onMount(async () => {
    // Defer loading until component is visible
    data = await fetchData();
    loaded = true;
  });
</script>

{#if !loaded}
  <div class="animate-pulse">
    <div class="h-48 bg-base-300 rounded"></div>
  </div>
{:else}
  <div>
    <!-- Actual content -->
  </div>
{/if}
```

### Session Management (Go Backend)

```go
// middleware/auth.go
type SessionConfig struct {
    Lifetime       time.Duration `env:"SESSION_LIFETIME" default:"7200"` // 2 hours
    RefreshWindow  time.Duration `env:"SESSION_REFRESH_WINDOW" default:"1800"` // 30 min
    SecureCookie   bool          `env:"SESSION_SECURE" default:"true"`
}

func SessionMiddleware(config SessionConfig) gin.HandlerFunc {
    return func(c *gin.Context) {
        session := getSession(c)
        
        // Check session expiry
        if session.ExpiresAt.Before(time.Now()) {
            c.JSON(401, gin.H{"error": "Session expired"})
            c.Abort()
            return
        }
        
        // Refresh session if within refresh window
        if time.Until(session.ExpiresAt) < config.RefreshWindow {
            session.ExpiresAt = time.Now().Add(config.Lifetime)
            saveSession(session)
            
            // Set new token in response header
            c.Header("X-New-Token", generateToken(session))
        }
        
        c.Next()
    }
}
```

### Log Viewer (SvelteKit)

```svelte
<script lang="ts">
  export let logs: LogEntry[];

  interface LogEntry {
    timestamp: string;
    type: 'info' | 'error' | 'warning' | 'command';
    message: string;
    output?: string;
    exitCode?: number;
    duration?: number;
  }

  const levelStyles = {
    info: 'border-l-info',
    error: 'border-l-error bg-error/5',
    warning: 'border-l-warning',
    command: 'border-l-neutral',
  };

  const badgeStyles = {
    info: 'badge-info',
    error: 'badge-error',
    warning: 'badge-warning',
    command: 'badge-neutral',
  };
</script>

<div class="divide-y divide-base-300 max-h-96 overflow-y-auto">
  {#each logs as log}
    <details class="border-l-4 {levelStyles[log.type]}">
      <summary class="px-4 py-2 cursor-pointer flex items-center gap-3">
        <span class="font-mono text-xs text-base-content/70">
          {new Date(log.timestamp).toLocaleTimeString()}
        </span>
        <span class="badge {badgeStyles[log.type]} badge-sm">
          {log.type}
        </span>
        {#if log.type === 'command'}
          <code class="bg-neutral text-neutral-content px-2 py-0.5 rounded text-xs">
            $ {log.message}
          </code>
        {:else}
          <span class="text-sm truncate">{log.message}</span>
        {/if}
      </summary>
      
      {#if log.output}
        <div class="px-4 py-2 bg-base-200">
          <pre class="text-xs overflow-x-auto">{log.output}</pre>
          {#if log.exitCode !== undefined}
            <span class="badge {log.exitCode === 0 ? 'badge-success' : 'badge-error'} badge-sm mt-2">
              Exit: {log.exitCode}
            </span>
          {/if}
        </div>
      {/if}
    </details>
  {/each}
</div>
```

---

## 9. Key Takeaways

| Pattern | Databasement Approach | SvelteKit Equivalent |
|---------|----------------------|---------------------|
| **Component Library** | Mary UI (Livewire) | DaisyUI + custom Svelte components |
| **Styling** | Tailwind + DaisyUI | Same (Tailwind + DaisyUI) |
| **State Management** | Livewire properties | Svelte stores |
| **Real-time Updates** | `wire:poll` | WebSocket or `setInterval` |
| **Deferred Loading** | `wire:init="load"` | `onMount` + loading state |
| **Charts** | Chart.js via Alpine.js | Chart.js or svelte-chartjs |
| **Session** | Laravel Fortify | JWT with refresh tokens |
| **Navigation** | `wire:navigate` | SvelteKit router |
| **Icons** | Heroicons (blade-icons) | Heroicons or Lucide |
| **Toast Notifications** | Mary Toast | svelte-french-toast or custom |
| **Modals** | Mary Modal | Custom or svelte-headless-ui |

### Implementation Priority for Dockerverse

1. **Sidebar** - Implement collapsible sidebar with route-based highlighting
2. **Theme System** - Add localStorage-based theme persistence
3. **Deferred Loading** - Add skeleton loading states
4. **Log Viewer** - Expandable log entries with filtering
5. **Charts** - Dashboard stats with Chart.js
6. **Session Refresh** - Implement token refresh before expiry
