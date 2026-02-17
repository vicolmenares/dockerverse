# Shell Page Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a `/shell` page with multi-tab terminal support — each tab is an independent xterm.js session (container exec or host SSH) that stays alive when switching tabs.

**Architecture:** Reuse the existing `Terminal.svelte` engine by adding a `mode="embedded"` prop that removes popup-specific behavior (fixed position, drag, fullscreen, forced X button). The Shell page manages tab state (array of sessions) and renders all Terminal instances simultaneously, showing only the active tab via CSS. Backend WebSocket endpoints already exist.

**Tech Stack:** Svelte 5 runes, xterm.js (already in project via Terminal.svelte), lucide-svelte icons, Tailwind CSS, existing `/ws/terminal/:hostId/:containerId` and `/ws/ssh/:hostId` WebSocket endpoints.

---

## Task 1: Add Shell to sidebar nav

**Files:**
- Modify: `frontend/src/routes/+layout.svelte:6-30` (icon imports)
- Modify: `frontend/src/routes/+layout.svelte:110-115` (sidebarItems array, after "logs" entry)

**Step 1: Add SquareTerminal to the lucide import block**

Find this block (lines 6-30):
```typescript
import {
  Search,
  Settings as SettingsIcon,
  RefreshCw,
  Globe,
  X,
  User,
  LogOut,
  ChevronDown,
  Moon,
  Sun,
  Menu,
  Home,
  Shield,
  Bell,
  Palette,
  Info,
  Database,
  Users,
  ArrowUpCircle,
  ScrollText,
  Server,
  ChevronsLeft,
  ChevronsRight,
} from "lucide-svelte";
```

Add `SquareTerminal,` after `ScrollText,`:
```typescript
import {
  Search,
  Settings as SettingsIcon,
  RefreshCw,
  Globe,
  X,
  User,
  LogOut,
  ChevronDown,
  Moon,
  Sun,
  Menu,
  Home,
  Shield,
  Bell,
  Palette,
  Info,
  Database,
  Users,
  ArrowUpCircle,
  ScrollText,
  SquareTerminal,
  Server,
  ChevronsLeft,
  ChevronsRight,
} from "lucide-svelte";
```

**Step 2: Add Shell entry to sidebarItems after the logs entry**

Find:
```typescript
  {
    id: "logs",
    icon: ScrollText,
    label: "Logs",
    href: "/logs",
  },
  {
    id: "environments",
```

Replace with:
```typescript
  {
    id: "logs",
    icon: ScrollText,
    label: "Logs",
    href: "/logs",
  },
  {
    id: "shell",
    icon: SquareTerminal,
    label: "Shell",
    href: "/shell",
  },
  {
    id: "environments",
```

**Step 3: Verify it renders**

Run: `cd frontend && npm run dev` (if running locally) or just deploy.
Expected: "Shell" appears in sidebar between "Logs" and "Environments" with terminal icon. Clicking navigates to `/shell` (will show 404 until Task 3).

**Step 4: Commit**
```bash
git add frontend/src/routes/+layout.svelte
git commit -m "feat: add Shell nav entry to sidebar"
```

---

## Task 2: Add embedded mode to Terminal.svelte

**Files:**
- Modify: `frontend/src/lib/components/Terminal.svelte`

This task adds `mode="embedded"` support without breaking existing popup usage anywhere.

**Step 1: Update the props destructuring (around line 22)**

Find:
```typescript
  let {
    container,
    host,
    mode = "container",
    onClose,
  }: {
    container?: Container;
    host?: Host;
    mode?: "container" | "host";
    onClose: () => void;
  } = $props();
```

Replace with:
```typescript
  let {
    container,
    host,
    mode = "container",
    terminalMode = "popup",
    active = true,
    onClose,
  }: {
    container?: Container;
    host?: Host;
    mode?: "container" | "host";
    terminalMode?: "popup" | "embedded";
    active?: boolean;
    onClose?: () => void;
  } = $props();

  let isEmbedded = $derived(terminalMode === "embedded");
```

Note: `mode` already exists with a different meaning ("container" | "host"). The new prop is named `terminalMode` to avoid collision.

**Step 2: Add $effect to fit terminal when activated (after the existing state declarations, around line 75)**

Find the block that ends with:
```typescript
  // Connection status color
  let statusColor = $derived(
```

Add BEFORE that line:
```typescript
  // Re-fit terminal when tab becomes active (embedded mode)
  $effect(() => {
    if (active && fitAddon) {
      setTimeout(() => fitAddon?.fit(), 50);
    }
  });

```

**Step 3: Update the root div class (around line 678)**

Find:
```svelte
<div
  bind:this={windowElement}
  class="fixed z-50 bg-background-secondary border border-border rounded-lg shadow-2xl flex flex-col overflow-hidden"
  class:inset-4={isFullscreen}
  class:w-[800px]={!isFullscreen}
  class:h-[500px]={!isFullscreen}
  style={!isFullscreen && (position.x !== 0 || position.y !== 0)
    ? `left: ${position.x}px; top: ${position.y}px;`
    : !isFullscreen
      ? "bottom: 1rem; right: 1rem;"
      : ""}
>
```

Replace with:
```svelte
<div
  bind:this={windowElement}
  class:fixed={!isEmbedded}
  class:z-50={!isEmbedded}
  class:rounded-lg={!isEmbedded}
  class:shadow-2xl={!isEmbedded}
  class:h-full={isEmbedded}
  class="bg-background-secondary border border-border flex flex-col overflow-hidden"
  class:inset-4={!isEmbedded && isFullscreen}
  class:w-[800px]={!isEmbedded && !isFullscreen}
  class:h-[500px]={!isEmbedded && !isFullscreen}
  style={!isEmbedded && !isFullscreen && (position.x !== 0 || position.y !== 0)
    ? `left: ${position.x}px; top: ${position.y}px;`
    : !isEmbedded && !isFullscreen
      ? "bottom: 1rem; right: 1rem;"
      : ""}
>
```

**Step 4: Hide drag handle, fullscreen, and X buttons in embedded mode**

In the header section, find the drag handle div (around line 700):
```svelte
      {#if !isFullscreen}
        <GripHorizontal class="w-4 h-4 text-foreground-muted" />
      {/if}
```

Change to:
```svelte
      {#if !isFullscreen && !isEmbedded}
        <GripHorizontal class="w-4 h-4 text-foreground-muted" />
      {/if}
```

Find the fullscreen button (around line 805):
```svelte
      <!-- Fullscreen -->
      <button
        class="btn-icon"
        onclick={toggleFullscreen}
        title={isFullscreen ? "Minimize" : "Maximize"}
      >
        {#if isFullscreen}
          <Minimize2 class="w-4 h-4" />
        {:else}
          <Maximize2 class="w-4 h-4" />
        {/if}
      </button>
```

Change to:
```svelte
      <!-- Fullscreen (popup mode only) -->
      {#if !isEmbedded}
      <button
        class="btn-icon"
        onclick={toggleFullscreen}
        title={isFullscreen ? "Minimize" : "Maximize"}
      >
        {#if isFullscreen}
          <Minimize2 class="w-4 h-4" />
        {:else}
          <Maximize2 class="w-4 h-4" />
        {/if}
      </button>
      {/if}
```

Find the X (close) button (around line 816):
```svelte
      <button
        class="btn-icon hover:text-accent-red"
        onclick={onClose}
        title={t.close}
      >
        <X class="w-4 h-4" />
      </button>
```

Change to:
```svelte
      {#if !isEmbedded && onClose}
      <button
        class="btn-icon hover:text-accent-red"
        onclick={onClose}
        title={t.close}
      >
        <X class="w-4 h-4" />
      </button>
      {/if}
```

**Step 5: Prevent drag in embedded mode**

Find the header div with onmousedown (around line 691):
```svelte
  <div
    role="toolbar"
    tabindex="0"
    class="flex items-center justify-between px-4 py-2 bg-background-tertiary border-b border-border select-none"
    class:cursor-grab={!isFullscreen && !isDragging}
    class:cursor-grabbing={isDragging}
    onmousedown={startDrag}
  >
```

Change to:
```svelte
  <div
    role="toolbar"
    tabindex="0"
    class="flex items-center justify-between px-4 py-2 bg-background-tertiary border-b border-border select-none"
    class:cursor-grab={!isEmbedded && !isFullscreen && !isDragging}
    class:cursor-grabbing={!isEmbedded && isDragging}
    onmousedown={isEmbedded ? undefined : startDrag}
  >
```

**Step 6: Verify no breakage**

The existing Terminal popup usage (in ContainerCard.svelte) calls:
```svelte
<Terminal container={...} host={...} mode="container" onClose={...} />
```
This still works because `terminalMode` defaults to `"popup"` and `onClose` is now optional (was required).

Check ContainerCard.svelte to confirm it passes `onClose`. If it does, no change needed there.

**Step 7: Commit**
```bash
git add frontend/src/lib/components/Terminal.svelte
git commit -m "feat: add embedded mode to Terminal component"
```

---

## Task 3: Create the Shell page

**Files:**
- Create: `frontend/src/routes/shell/+page.svelte`

This is the main task. Full implementation below.

**Step 1: Create the file**

Create `frontend/src/routes/shell/+page.svelte` with this content:

```svelte
<script lang="ts">
  import { SquareTerminal, Plus, X, Server, Box } from "lucide-svelte";
  import Terminal from "$lib/components/Terminal.svelte";
  import { containers, hosts } from "$lib/stores/docker";
  import type { Container, Host } from "$lib/api/docker";

  // ── Tab state ───────────────────────────────────────────────
  interface Tab {
    id: string;
    type: "container" | "host";
    container?: Container;
    host?: Host;
    label: string;
    hostLabel: string;
  }

  let tabs = $state<Tab[]>([]);
  let activeTabId = $state<string | null>(null);

  // ── Toolbar state ───────────────────────────────────────────
  let selectedHostId = $state<string>("");
  let selectedContainerId = $state<string>("");

  // Running containers, optionally filtered by selected host
  let filteredContainers = $derived(
    $containers.filter(
      (c) =>
        c.state === "running" &&
        (selectedHostId === "" || c.hostId === selectedHostId),
    ),
  );

  // Reset container selection when host changes
  $effect(() => {
    selectedHostId; // track
    selectedContainerId = "";
  });

  // ── Tab actions ─────────────────────────────────────────────
  function openContainerShell() {
    const container = $containers.find((c) => c.id === selectedContainerId);
    if (!container) return;
    const id = crypto.randomUUID();
    tabs = [
      ...tabs,
      {
        id,
        type: "container",
        container,
        label: container.name,
        hostLabel: container.hostId,
      },
    ];
    activeTabId = id;
  }

  function openHostSSH() {
    const host = $hosts.find((h) => h.id === selectedHostId);
    if (!host) return;
    const id = crypto.randomUUID();
    tabs = [
      ...tabs,
      {
        id,
        type: "host",
        host,
        label: host.name,
        hostLabel: host.id,
      },
    ];
    activeTabId = id;
  }

  function closeTab(tabId: string) {
    const idx = tabs.findIndex((t) => t.id === tabId);
    const newTabs = tabs.filter((t) => t.id !== tabId);
    tabs = newTabs;
    if (activeTabId === tabId) {
      activeTabId = newTabs[idx]?.id ?? newTabs[idx - 1]?.id ?? null;
    }
  }

  // Keyboard shortcuts
  function handleKeydown(e: KeyboardEvent) {
    // Ctrl+W = close active tab
    if ((e.ctrlKey || e.metaKey) && e.key === "w" && activeTabId) {
      e.preventDefault();
      closeTab(activeTabId);
    }
  }
</script>

<svelte:window onkeydown={handleKeydown} />

<!-- Page root: fixed, expands when sidebar collapses via --sidebar-w CSS var -->
<div
  class="shell-page-root fixed top-16 left-0 right-0 bottom-0 flex flex-col bg-background z-10 overflow-hidden"
>
  <!-- ── Toolbar ─────────────────────────────────────────── -->
  <div
    class="flex items-center gap-3 px-4 py-2.5 border-b border-border bg-background-secondary flex-shrink-0"
  >
    <!-- Page title -->
    <div class="flex items-center gap-2 mr-2">
      <SquareTerminal class="w-4 h-4 text-primary" />
      <span class="text-sm font-semibold text-foreground">Shell</span>
    </div>

    <div class="w-px h-5 bg-border"></div>

    <!-- Host selector -->
    <div class="flex items-center gap-2">
      <label class="text-xs text-foreground-muted whitespace-nowrap">Host</label>
      <select
        bind:value={selectedHostId}
        class="bg-background border border-border rounded-md px-2 py-1 text-sm text-foreground focus:outline-none focus:border-primary cursor-pointer"
      >
        <option value="">All hosts</option>
        {#each $hosts as host}
          <option value={host.id}>{host.name}</option>
        {/each}
      </select>
    </div>

    <!-- Container selector -->
    <div class="flex items-center gap-2">
      <label class="text-xs text-foreground-muted whitespace-nowrap">Container</label>
      <select
        bind:value={selectedContainerId}
        class="bg-background border border-border rounded-md px-2 py-1 text-sm text-foreground focus:outline-none focus:border-primary cursor-pointer min-w-[180px]"
        disabled={filteredContainers.length === 0}
      >
        <option value="">Select container...</option>
        {#each filteredContainers as c}
          <option value={c.id}>{c.name} ({c.hostId})</option>
        {/each}
      </select>
    </div>

    <!-- Open Shell button -->
    <button
      class="flex items-center gap-1.5 px-3 py-1.5 rounded-md bg-primary text-white text-sm font-medium hover:bg-primary/90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed cursor-pointer"
      disabled={!selectedContainerId}
      onclick={openContainerShell}
    >
      <Box class="w-3.5 h-3.5" />
      Open Shell
    </button>

    <!-- SSH Host button -->
    <button
      class="flex items-center gap-1.5 px-3 py-1.5 rounded-md border border-border text-sm text-foreground hover:bg-background-tertiary transition-colors disabled:opacity-40 disabled:cursor-not-allowed cursor-pointer"
      disabled={!selectedHostId}
      onclick={openHostSSH}
    >
      <Server class="w-3.5 h-3.5" />
      SSH Host
    </button>
  </div>

  <!-- ── Tab bar (only visible when there are tabs) ───────── -->
  {#if tabs.length > 0}
    <div
      class="flex items-center border-b border-border bg-background-tertiary flex-shrink-0 overflow-x-auto"
    >
      {#each tabs as tab (tab.id)}
        <button
          class="group flex items-center gap-1.5 px-3 py-2 text-sm border-r border-border whitespace-nowrap transition-colors cursor-pointer"
          class:bg-background={activeTabId === tab.id}
          class:text-foreground={activeTabId === tab.id}
          class:border-b-2={activeTabId === tab.id}
          class:border-b-primary={activeTabId === tab.id}
          class:text-foreground-muted={activeTabId !== tab.id}
          class:hover:bg-background={activeTabId !== tab.id}
          class:hover:text-foreground={activeTabId !== tab.id}
          onclick={() => (activeTabId = tab.id)}
        >
          {#if tab.type === "host"}
            <Server class="w-3.5 h-3.5 flex-shrink-0 text-amber-400" />
          {:else}
            <Box class="w-3.5 h-3.5 flex-shrink-0 text-sky-400" />
          {/if}
          <span>{tab.label}</span>
          <span class="text-foreground-muted/60 text-xs">@{tab.hostLabel}</span>
          <!-- Close tab button -->
          <span
            role="button"
            tabindex="0"
            class="ml-1 p-0.5 rounded hover:bg-accent-red/20 hover:text-accent-red transition-colors opacity-0 group-hover:opacity-100 cursor-pointer"
            onclick={(e) => {
              e.stopPropagation();
              closeTab(tab.id);
            }}
            onkeydown={(e) => e.key === 'Enter' && closeTab(tab.id)}
            aria-label="Close tab"
          >
            <X class="w-3 h-3" />
          </span>
        </button>
      {/each}

      <!-- New tab button -->
      <button
        class="p-2 text-foreground-muted hover:text-foreground hover:bg-background transition-colors cursor-pointer"
        onclick={openContainerShell}
        disabled={!selectedContainerId}
        title="Open new shell (select container above)"
      >
        <Plus class="w-4 h-4" />
      </button>
    </div>
  {/if}

  <!-- ── Terminal area ─────────────────────────────────────── -->
  <div class="flex-1 relative overflow-hidden">
    {#if tabs.length === 0}
      <!-- Empty state -->
      <div
        class="absolute inset-0 flex flex-col items-center justify-center gap-4 text-foreground-muted"
      >
        <SquareTerminal class="w-12 h-12 opacity-20" />
        <div class="text-center">
          <p class="text-base font-medium text-foreground-muted">
            Select a container or host to open a shell
          </p>
          <p class="text-sm mt-1 opacity-60">
            Choose from the toolbar above and click Open Shell or SSH Host
          </p>
        </div>
        <div class="flex items-center gap-4 text-xs opacity-40 mt-2">
          <span><kbd class="bg-background-tertiary border border-border rounded px-1.5 py-0.5">Ctrl+W</kbd> Close tab</span>
        </div>
      </div>
    {:else}
      <!-- Render ALL terminal instances, show only the active one -->
      {#each tabs as tab (tab.id)}
        <div
          class="absolute inset-0"
          class:hidden={activeTabId !== tab.id}
        >
          <Terminal
            terminalMode="embedded"
            active={activeTabId === tab.id}
            mode={tab.type}
            container={tab.container}
            host={tab.host}
          />
        </div>
      {/each}
    {/if}
  </div>
</div>

<style>
  @media (min-width: 1024px) {
    .shell-page-root {
      left: var(--sidebar-w, 16rem);
      transition: left 300ms ease;
    }
  }
</style>
```

**Step 2: Verify the file was created**

```bash
ls frontend/src/routes/shell/
```
Expected: `+page.svelte`

**Step 3: Deploy to raspi and test**

```bash
./deploy-to-raspi.sh
```

Then navigate to `http://192.168.1.145:3007/shell` and verify:
- "Shell" appears in sidebar with terminal icon
- Toolbar shows host dropdown and container dropdown
- Selecting a running container and clicking "Open Shell" opens a new tab with a working terminal
- Switching between tabs keeps sessions alive
- Ctrl+W closes the active tab
- Clicking × on a tab closes it
- Collapsing sidebar expands the shell area (--sidebar-w CSS var)

**Step 4: Commit**
```bash
git add frontend/src/routes/shell/+page.svelte
git commit -m "feat: add Shell page with multi-tab terminal support"
```

---

## Task 4: Push and verify

**Step 1: Push all commits**
```bash
git push
```

**Step 2: Open the shell page in the browser**

Navigate to `http://192.168.1.145:3007/shell`

Checklist:
- [ ] Sidebar shows "Shell" with SquareTerminal icon between Logs and Environments
- [ ] Toolbar: host dropdown populates with raspi1/raspi2
- [ ] Container dropdown: shows running containers filtered by selected host
- [ ] "Open Shell" button disabled when no container selected
- [ ] "SSH Host" button disabled when no host selected
- [ ] Clicking "Open Shell" creates a tab → terminal connects → prompt appears
- [ ] Clicking "SSH Host" creates a tab → SSH terminal connects → host prompt appears
- [ ] Tab switching: sessions stay alive (terminal output preserved)
- [ ] Tab close (× button): disconnects WebSocket and removes tab
- [ ] Ctrl+W: closes active tab
- [ ] Sidebar collapse: shell area expands (same as logs page)
- [ ] Empty state shows correctly when all tabs closed

---

## Notes

- **No backend changes needed.** Both WS endpoints are already deployed and working.
- **Backwards compatibility:** `Terminal.svelte` in popup mode is unchanged. `ContainerCard.svelte` and any other existing popup usage is unaffected because `terminalMode` defaults to `"popup"`.
- **Tab persistence:** Tabs are in-memory only. Refreshing the page closes all sessions. This is expected behavior for a terminal.
- **Performance:** Each open tab maintains an active WebSocket and an xterm.js instance. The xterm.js rendering is paused for hidden tabs (no repaints) but the connection stays alive.
