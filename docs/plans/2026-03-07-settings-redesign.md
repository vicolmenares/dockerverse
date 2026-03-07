# Settings Redesign Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Redesign DockerVerse Settings section and sidebar to match Dockhand's clean, professional layout — collapsible Settings group in sidebar, consolidated Security, renamed General, new Authentication section, and full-width Environments table.

**Architecture:** Sidebar gets a collapsible Settings group (Environments, General, Users, Notifications, Authentication). The duplicate Security entries are resolved: `/security` keeps vulnerability scans, `/settings/security` content moves to `/settings/authentication`. `/settings/appearance` becomes `/settings/general`. Environments table becomes full-width and linear.

**Tech Stack:** SvelteKit 2, Svelte 5 (runes: `$state`, `$derived`, `$effect`, `$props`), Tailwind CSS, Lucide Svelte icons, Go Fiber backend.

---

## Key File Reference

- Sidebar + layout: `frontend/src/routes/+layout.svelte`
- Settings layout: `frontend/src/routes/settings/+layout.svelte`
- Settings hub: `frontend/src/routes/settings/+page.svelte`
- Appearance page: `frontend/src/routes/settings/appearance/+page.svelte`
- Security (auth) page: `frontend/src/routes/settings/security/+page.svelte`
- Environments page: `frontend/src/routes/settings/environments/+page.svelte`
- Environments modal: `frontend/src/lib/components/EnvironmentModal.svelte`
- Vuln scan page: `frontend/src/routes/security/+page.svelte` (stays at `/security`)

---

## Task 1: Add Settings group to sidebar with collapse behavior

**Files:**
- Modify: `frontend/src/routes/+layout.svelte`

**Step 1: Read the current sidebar**

Read `frontend/src/routes/+layout.svelte` to understand current sidebarItems structure (lines 106–177).

**Step 2: Replace sidebarItems with grouped structure**

Replace the flat `sidebarItems` array and nav rendering with a two-group structure:

```typescript
// In <script> block — replace the sidebarItems derived

type NavItem = {
  id: string;
  icon: any;
  label: string;
  href: string;
};

type NavGroup = {
  id: string;
  label: string;
  items: NavItem[];
};

let settingsOpen = $state(browser ? localStorage.getItem('dockerverse-settings-open') !== 'false' : true);

function toggleSettings() {
  settingsOpen = !settingsOpen;
  if (browser) localStorage.setItem('dockerverse-settings-open', String(settingsOpen));
}

const mainItems = $derived<NavItem[]>([
  { id: 'dashboard', icon: Home, label: $language === 'es' ? 'Dashboard' : 'Dashboard', href: '/' },
  { id: 'logs', icon: ScrollText, label: 'Logs', href: '/logs' },
  { id: 'shell', icon: SquareTerminal, label: 'Shell', href: '/shell' },
  { id: 'security-scans', icon: Shield, label: $language === 'es' ? 'Seguridad' : 'Security', href: '/security' },
]);

const settingsItems = $derived<NavItem[]>([
  { id: 'environments', icon: Server, label: $language === 'es' ? 'Entornos' : 'Environments', href: '/settings/environments' },
  ...($currentUser?.roles?.includes('admin') ? [{ id: 'users', icon: Users, label: $language === 'es' ? 'Usuarios' : 'Users', href: '/settings/users' }] : []),
  { id: 'notifications', icon: Bell, label: $language === 'es' ? 'Notificaciones' : 'Notifications', href: '/settings/notifications' },
  { id: 'general', icon: Palette, label: $language === 'es' ? 'General' : 'General', href: '/settings/general' },
  { id: 'authentication', icon: KeyRound, label: $language === 'es' ? 'Autenticación' : 'Authentication', href: '/settings/authentication' },
  { id: 'data', icon: Database, label: $language === 'es' ? 'Datos' : 'Data', href: '/settings/data' },
  { id: 'about', icon: Info, label: $language === 'es' ? 'Acerca de' : 'About', href: '/settings/about' },
]);
```

**Step 3: Add KeyRound to icon imports**

In the imports from `lucide-svelte`, add `KeyRound, ChevronRight` (ChevronRight for the settings group toggle arrow).

**Step 4: Update activeSidebarItem derived**

```typescript
let activeSidebarItem = $derived.by(() => {
  const pathname = $page.url.pathname;
  if (pathname.startsWith('/logs')) return 'logs';
  if (pathname.startsWith('/shell')) return 'shell';
  if (pathname.startsWith('/security') && !pathname.startsWith('/settings')) return 'security-scans';
  if (pathname.startsWith('/settings/users')) return 'users';
  if (pathname.startsWith('/settings/notifications')) return 'notifications';
  if (pathname.startsWith('/settings/general')) return 'general';
  if (pathname.startsWith('/settings/authentication')) return 'authentication';
  if (pathname.startsWith('/settings/environments')) return 'environments';
  if (pathname.startsWith('/settings/data')) return 'data';
  if (pathname.startsWith('/settings/about')) return 'about';
  if (pathname.startsWith('/settings')) return 'settings';
  return 'dashboard';
});

// Detect if any settings item is active (for auto-expanding group)
let isSettingsActive = $derived(
  activeSidebarItem === 'environments' ||
  activeSidebarItem === 'users' ||
  activeSidebarItem === 'notifications' ||
  activeSidebarItem === 'general' ||
  activeSidebarItem === 'authentication' ||
  activeSidebarItem === 'data' ||
  activeSidebarItem === 'about' ||
  activeSidebarItem === 'settings'
);
```

**Step 5: Replace nav rendering**

Replace the `<nav>` block (`{#each sidebarItems as item}`) with:

```svelte
<nav class="flex-1 px-2 py-3 space-y-0.5 overflow-y-auto">
  <!-- Main items -->
  {#each mainItems as item}
    {@const Icon = item.icon}
    {@const isActive = activeSidebarItem === item.id}
    <a
      href={item.href}
      class="flex items-center {navCollapsed ? 'justify-center px-2 py-2.5' : 'gap-3 px-3 py-2.5'} rounded-lg transition-colors {isActive
        ? 'bg-primary/15 text-primary border-l-2 border-primary'
        : 'text-foreground-muted hover:text-foreground hover:bg-background-tertiary'}"
      onclick={() => { showSidebar = false; }}
      title={navCollapsed ? item.label : undefined}
    >
      <Icon class="w-5 h-5 flex-shrink-0" />
      {#if !navCollapsed}
        <span class="text-sm font-medium">{item.label}</span>
      {/if}
    </a>
  {/each}

  <!-- Settings group -->
  <div class="pt-2">
    <button
      class="w-full flex items-center {navCollapsed ? 'justify-center px-2 py-2' : 'gap-3 px-3 py-2'} rounded-lg transition-colors text-foreground-muted hover:text-foreground hover:bg-background-tertiary"
      onclick={toggleSettings}
      title={navCollapsed ? 'Settings' : undefined}
    >
      <SettingsIcon class="w-5 h-5 flex-shrink-0 {isSettingsActive ? 'text-primary' : ''}" />
      {#if !navCollapsed}
        <span class="text-sm font-medium flex-1 text-left {isSettingsActive ? 'text-primary' : ''}">
          {$language === 'es' ? 'Configuración' : 'Settings'}
        </span>
        <ChevronRight class="w-4 h-4 transition-transform {settingsOpen ? 'rotate-90' : ''}" />
      {/if}
    </button>

    {#if (settingsOpen || isSettingsActive) && !navCollapsed}
      <div class="mt-0.5 ml-2 pl-3 border-l border-border space-y-0.5">
        {#each settingsItems as item}
          {@const Icon = item.icon}
          {@const isActive = activeSidebarItem === item.id}
          <a
            href={item.href}
            class="flex items-center gap-3 px-3 py-2 rounded-lg transition-colors text-sm {isActive
              ? 'bg-primary/15 text-primary'
              : 'text-foreground-muted hover:text-foreground hover:bg-background-tertiary'}"
            onclick={() => { showSidebar = false; }}
          >
            <Icon class="w-4 h-4 flex-shrink-0" />
            <span class="font-medium">{item.label}</span>
          </a>
        {/each}
      </div>
    {:else if navCollapsed}
      {#each settingsItems as item}
        {@const Icon = item.icon}
        {@const isActive = activeSidebarItem === item.id}
        <a
          href={item.href}
          class="flex items-center justify-center px-2 py-2.5 rounded-lg transition-colors {isActive
            ? 'bg-primary/15 text-primary'
            : 'text-foreground-muted hover:text-foreground hover:bg-background-tertiary'}"
          onclick={() => { showSidebar = false; }}
          title={item.label}
        >
          <Icon class="w-5 h-5 flex-shrink-0" />
        </a>
      {/each}
    {/if}
  </div>
</nav>
```

**Step 6: Commit**

```bash
cd /Users/vcolmenares/Documents/Laboratories/Antigravity/skills/dockerverse-project/dockerverse
git add frontend/src/routes/+layout.svelte
git commit -m "feat: add collapsible Settings group to sidebar navigation"
```

---

## Task 2: Create /settings/authentication route (move from /settings/security)

**Files:**
- Create: `frontend/src/routes/settings/authentication/+page.svelte`
- Modify: `frontend/src/routes/settings/security/+page.svelte` (redirect to /settings/authentication)

**Step 1: Read current /settings/security page**

Read `frontend/src/routes/settings/security/+page.svelte` to get full content.

**Step 2: Create authentication page**

Copy the entire content of `/settings/security/+page.svelte` into `/settings/authentication/+page.svelte`.

Update the page title/heading from "Security" to "Authentication" and update any breadcrumb text.

**Step 3: Replace /settings/security with redirect**

```svelte
<!-- frontend/src/routes/settings/security/+page.svelte -->
<script lang="ts">
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte';
  onMount(() => goto('/settings/authentication', { replaceState: true }));
</script>
```

**Step 4: Commit**

```bash
git add frontend/src/routes/settings/authentication/+page.svelte frontend/src/routes/settings/security/+page.svelte
git commit -m "feat: add /settings/authentication route, redirect /settings/security"
```

---

## Task 3: Rename /settings/appearance to /settings/general

**Files:**
- Create: `frontend/src/routes/settings/general/+page.svelte`
- Modify: `frontend/src/routes/settings/appearance/+page.svelte` (redirect)

**Step 1: Read current appearance page**

Read `frontend/src/routes/settings/appearance/+page.svelte`.

**Step 2: Create /settings/general**

Copy the entire content into `frontend/src/routes/settings/general/+page.svelte`.

Update the page heading from "Appearance" to "General" (or "General Settings").

**Step 3: Redirect /settings/appearance**

```svelte
<!-- frontend/src/routes/settings/appearance/+page.svelte -->
<script lang="ts">
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte';
  onMount(() => goto('/settings/general', { replaceState: true }));
</script>
```

**Step 4: Commit**

```bash
git add frontend/src/routes/settings/general/+page.svelte frontend/src/routes/settings/appearance/+page.svelte
git commit -m "feat: add /settings/general (renamed from /settings/appearance)"
```

---

## Task 4: Update settings hub page

**Files:**
- Modify: `frontend/src/routes/settings/+page.svelte`

**Step 1: Read current settings hub**

Read `frontend/src/routes/settings/+page.svelte`.

**Step 2: Update hub links and labels**

Replace:
- "Appearance" card → "General" (href `/settings/general`)
- "Security" (password/2FA) card → "Authentication" (href `/settings/authentication`)
- Remove duplicate security reference, confirm only one Security card exists for vuln scans (which links to `/security` — or remove it since Security is already a main nav item)

Example updated card structure:
```svelte
<!-- General settings card -->
<a href="/settings/general" class="settings-card-class">
  <Palette class="w-6 h-6" />
  <div>
    <h3>General</h3>
    <p>Theme, language, display preferences</p>
  </div>
</a>

<!-- Authentication card -->
<a href="/settings/authentication" class="settings-card-class">
  <KeyRound class="w-6 h-6" />
  <div>
    <h3>Authentication</h3>
    <p>Password, two-factor authentication</p>
  </div>
</a>
```

**Step 3: Commit**

```bash
git add frontend/src/routes/settings/+page.svelte
git commit -m "feat: update settings hub with General and Authentication links"
```

---

## Task 5: Redesign Environments table to full-width linear layout

**Files:**
- Modify: `frontend/src/routes/settings/environments/+page.svelte`
- Modify: `frontend/src/routes/settings/+layout.svelte`

**Step 1: Fix settings layout width constraint**

Read `frontend/src/routes/settings/+layout.svelte`. Change `max-w-2xl mx-auto` to `max-w-6xl mx-auto` or remove entirely for environments (may need a per-page override).

Actually: the settings layout applies to ALL settings pages. Better to change it to `max-w-5xl mx-auto` globally to give more breathing room.

**Step 2: Read current environments page**

Read `frontend/src/routes/settings/environments/+page.svelte`.

**Step 3: Redesign the environments table**

Replace the cramped table with a clean card/row-based design inspired by Dockhand:

Design principles:
- Each environment gets its own card row (like Dockhand's linear list)
- Show: colored status dot, name (bold), connection type badge, host/socket path, labels as tags, feature flags as small icons
- Actions (Edit/Delete/Test) on the right side
- "Add Environment" button top-right, prominent
- Wide layout using full available width

Example structure:
```svelte
<!-- Page header -->
<div class="flex items-center justify-between mb-6">
  <div>
    <h1 class="text-2xl font-bold text-foreground">Environments</h1>
    <p class="text-sm text-foreground-muted mt-1">Manage Docker host connections</p>
  </div>
  <button class="btn btn-primary" onclick={openAddModal}>
    <Plus class="w-4 h-4" />
    Add Environment
  </button>
</div>

<!-- Environment list -->
<div class="space-y-2">
  {#each environments as env}
    <div class="bg-background-secondary border border-border rounded-xl px-5 py-4 flex items-center gap-4 hover:border-primary/30 transition-colors">
      <!-- Status indicator -->
      <div class="w-2.5 h-2.5 rounded-full flex-shrink-0 {env.isConnected ? 'bg-green-500' : 'bg-foreground-muted/30'}"></div>

      <!-- Name + connection info -->
      <div class="flex-1 min-w-0">
        <div class="flex items-center gap-3">
          <span class="font-semibold text-foreground">{env.name}</span>
          <!-- Connection type badge -->
          <span class="text-xs px-2 py-0.5 rounded-full bg-background-tertiary text-foreground-muted border border-border">
            {env.connectionType === 'socket' ? 'Socket' : 'TCP'}
          </span>
          {#if env.isLocal}
            <span class="text-xs px-2 py-0.5 rounded-full bg-primary/10 text-primary border border-primary/20">Local</span>
          {/if}
        </div>
        <p class="text-sm text-foreground-muted mt-0.5 truncate">
          {env.connectionType === 'socket' ? env.socketPath || '/var/run/docker.sock' : `${env.host}:${env.port}`}
        </p>
        <!-- Labels -->
        {#if env.labels?.length}
          <div class="flex gap-1 mt-1.5 flex-wrap">
            {#each env.labels as label}
              <span class="text-xs px-2 py-0.5 bg-background-tertiary text-foreground-muted rounded">{label}</span>
            {/each}
          </div>
        {/if}
      </div>

      <!-- Feature flags -->
      <div class="flex items-center gap-2 text-foreground-muted">
        {#if env.monitoring?.enabled}
          <span title="Monitoring" class="text-green-500"><Activity class="w-4 h-4" /></span>
        {/if}
        {#if env.tls?.enabled}
          <span title="TLS" class="text-blue-500"><Lock class="w-4 h-4" /></span>
        {/if}
      </div>

      <!-- Actions -->
      <div class="flex items-center gap-2 flex-shrink-0">
        <button class="btn btn-ghost btn-sm" onclick={() => testConnection(env)} title="Test connection">
          <Zap class="w-4 h-4" />
        </button>
        <button class="btn btn-ghost btn-sm" onclick={() => openEditModal(env)}>
          <Pencil class="w-4 h-4" />
          <span class="hidden sm:inline ml-1.5">Edit</span>
        </button>
        <button class="btn btn-ghost btn-sm text-red-400 hover:text-red-300" onclick={() => confirmDelete(env)}>
          <Trash2 class="w-4 h-4" />
        </button>
      </div>
    </div>
  {/each}

  {#if environments.length === 0}
    <div class="text-center py-16 text-foreground-muted">
      <Server class="w-12 h-12 mx-auto mb-3 opacity-30" />
      <p class="text-lg font-medium">No environments configured</p>
      <p class="text-sm mt-1">Add a Docker host to get started</p>
    </div>
  {/if}
</div>
```

Icons needed: `Plus, Pencil, Trash2, Activity, Lock, Zap` from lucide-svelte.

**Step 4: Update settings layout max-width**

In `frontend/src/routes/settings/+layout.svelte`, change the width constraint:
```svelte
<!-- Before -->
<div class="max-w-2xl mx-auto">

<!-- After -->
<div class="max-w-5xl mx-auto">
```

**Step 5: Run dev build to verify no compile errors**

```bash
cd /Users/vcolmenares/Documents/Laboratories/Antigravity/skills/dockerverse-project/dockerverse/frontend
npm run check
```

Expected: no TypeScript errors.

**Step 6: Commit**

```bash
cd ..
git add frontend/src/routes/settings/environments/+page.svelte frontend/src/routes/settings/+layout.svelte
git commit -m "feat: redesign environments table to full-width linear card layout"
```

---

## Task 6: Playwright end-to-end tests

**Files:**
- Read: existing test files to understand patterns
- Create/modify: `frontend/tests/settings-redesign.spec.ts` (or `e2e/` folder equivalent)

**Step 1: Find existing test location**

```bash
find /Users/vcolmenares/Documents/Laboratories/Antigravity/skills/dockerverse-project/dockerverse/frontend -name "*.spec.*" -o -name "playwright.config.*"
```

**Step 2: Launch Playwright session**

Use the `playwright` skill if available. Connect to DockerVerse at `http://raspi-main:3007` (or locally rebuilt dev server).

**Step 3: Test checklist via Playwright**

Run through manually or with test script:

1. Login with admin credentials
2. Verify sidebar shows "Settings" group with toggle button
3. Click Settings toggle → group expands/collapses
4. Verify "Security" appears once in main nav (links to `/security`)
5. Verify Settings group has: Environments, Users, Notifications, General, Authentication, Data, About
6. Navigate to `/settings/environments` → verify full-width card layout, each environment readable
7. Click Edit on Raspberry Main → modal opens with populated fields → save → 200 OK
8. Navigate to `/settings/general` → verify full appearance/theme page loads
9. Navigate to `/settings/authentication` → verify password/2FA page loads
10. Navigate to `/settings/appearance` → verify redirects to `/settings/general`
11. Navigate to `/settings/security` → verify redirects to `/settings/authentication`
12. Verify settings hub at `/settings` shows updated card links

**Step 4: Fix any failures before continuing**

---

## Task 7: Build frontend Docker image and deploy to raspi-main

**Step 1: Verify build succeeds locally**

```bash
cd /Users/vcolmenares/Documents/Laboratories/Antigravity/skills/dockerverse-project/dockerverse/frontend
npm run build
```

Expected: build completes with no errors.

**Step 2: Git push**

```bash
cd /Users/vcolmenares/Documents/Laboratories/Antigravity/skills/dockerverse-project/dockerverse
git push origin main
```

**Step 3: Build and deploy Docker image on raspi-main**

```bash
ssh raspi-main "cd ~/portainer/Files/AppData/Config/dockerverse && docker compose pull && docker compose up -d --build"
```

Or if DockerVerse is managed differently:

```bash
ssh raspi-main "cd /path/to/dockerverse && docker compose up -d --build frontend"
```

**Step 4: Verify deployment**

Open browser to `http://raspi-main:3007` and run through the test checklist from Task 6 manually.

---

## Summary of Route Changes

| Old Route | New Route | Action |
|-----------|-----------|--------|
| `/settings/appearance` | `/settings/general` | Renamed, old redirects |
| `/settings/security` | `/settings/authentication` | Renamed, old redirects |
| `/security` | `/security` | Unchanged (vuln scans) |
| `/settings/environments` | `/settings/environments` | Unchanged, redesigned UI |

## Summary of Sidebar Changes

| Before | After |
|--------|-------|
| Flat list with all items | Main items + collapsible Settings group |
| Two "Security" entries | One "Security" in main nav (vuln scans) |
| "Appearance" | "General" in Settings group |
| No "Authentication" | "Authentication" in Settings group |
| Environments at top level | Environments in Settings group |
