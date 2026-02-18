<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { SquareTerminal, Plus, X, Server, Box, ChevronDown } from "lucide-svelte";
  import Terminal from "$lib/components/Terminal.svelte";
  import { containers, hosts, language, translations } from "$lib/stores/docker";
  import type { Container, Host } from "$lib/api/docker";

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
  let selectedHostId = $state<string>("");
  let selectedContainerId = $state<string>("");
  let tabStatuses = $state<Map<string, "connecting" | "connected" | "disconnected" | "error">>(new Map());
  let lastOpenedType = $state<"container" | "host">("container");

  let t = $derived($translations[$language] || $translations.en);

  let hostList = $derived($hosts);
  let runningContainers = $derived(
    $containers.filter(
      (c) => c.state === "running" && (!selectedHostId || c.hostId === selectedHostId)
    )
  );

  // Auto-select first host
  $effect(() => {
    if (hostList.length > 0 && !selectedHostId) {
      selectedHostId = hostList[0].id;
    }
  });

  // Auto-select first container when host changes OR when selected container disappears.
  // Guard prevents SSE updates (which recompute runningContainers) from resetting the
  // user's selection every 2 seconds.
  $effect(() => {
    const ids = new Set(runningContainers.map((c) => c.id));
    if (!selectedContainerId || !ids.has(selectedContainerId)) {
      selectedContainerId = runningContainers[0]?.id ?? "";
    }
  });

  function generateId(): string {
    return Math.random().toString(36).slice(2, 9);
  }

  function setTabStatus(tabId: string, status: "connecting" | "connected" | "disconnected" | "error") {
    tabStatuses = new Map(tabStatuses).set(tabId, status);
  }

  function openContainerShell() {
    const container = $containers.find((c) => c.id === selectedContainerId);
    if (!container) return;
    const host = $hosts.find((h) => h.id === container.hostId);
    const id = generateId();
    tabs = [
      ...tabs,
      {
        id,
        type: "container",
        container,
        label: container.name,
        hostLabel: host?.name ?? container.hostId,
      },
    ];
    activeTabId = id;
    lastOpenedType = "container";
  }

  function openHostSSH() {
    const host = $hosts.find((h) => h.id === selectedHostId);
    if (!host) return;
    const id = generateId();
    tabs = [
      ...tabs,
      {
        id,
        type: "host",
        host,
        label: `${host.name} SSH`,
        hostLabel: host.name,
      },
    ];
    activeTabId = id;
    lastOpenedType = "host";
  }

  function closeTab(tabId: string) {
    const idx = tabs.findIndex((t) => t.id === tabId);
    tabs = tabs.filter((t) => t.id !== tabId);
    if (activeTabId === tabId) {
      if (tabs.length > 0) {
        activeTabId = tabs[Math.max(0, idx - 1)].id;
      } else {
        activeTabId = null;
      }
    }
    const newStatuses = new Map(tabStatuses);
    newStatuses.delete(tabId);
    tabStatuses = newStatuses;
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.ctrlKey && e.key === "t") {
      e.preventDefault();
      lastOpenedType === "host" ? openHostSSH() : openContainerShell();
    }
    if (e.ctrlKey && e.key === "w" && activeTabId) {
      e.preventDefault();
      closeTab(activeTabId);
    }
  }

  onMount(() => {
    document.addEventListener("keydown", handleKeydown);
    return () => document.removeEventListener("keydown", handleKeydown);
  });
</script>

<div class="fixed top-16 left-0 right-0 bottom-0 flex flex-col overflow-hidden bg-background z-10 shell-page-root">
  <!-- Toolbar -->
  <div class="flex items-center gap-2 px-4 py-2 bg-background-secondary border-b border-border flex-shrink-0">

    <!-- Primary group: host + container + open shell as one visual unit -->
    <div class="flex items-center rounded-lg border border-border bg-background overflow-hidden">
      <!-- Host selector -->
      <div class="relative border-r border-border">
        <Server class="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-foreground-muted pointer-events-none" />
        <select
          bind:value={selectedHostId}
          class="appearance-none bg-transparent pl-8 pr-7 py-1.5 text-sm text-foreground focus:outline-none cursor-pointer"
          style="appearance:none; -webkit-appearance:none; -moz-appearance:none;"
        >
          {#each hostList as host}
            <option value={host.id}>{host.name}</option>
          {/each}
        </select>
        <ChevronDown class="absolute right-2 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-foreground-muted pointer-events-none" />
      </div>

      <!-- Container selector -->
      <div class="relative border-r border-border">
        <Box class="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-foreground-muted pointer-events-none" />
        <select
          bind:value={selectedContainerId}
          class="appearance-none bg-transparent pl-8 pr-7 py-1.5 text-sm text-foreground focus:outline-none cursor-pointer min-w-[160px]"
          style="appearance:none; -webkit-appearance:none; -moz-appearance:none;"
        >
          {#if runningContainers.length === 0}
            <option value="">No containers</option>
          {:else}
            {#each runningContainers as c}
              <option value={c.id}>{c.name}</option>
            {/each}
          {/if}
        </select>
        <ChevronDown class="absolute right-2 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-foreground-muted pointer-events-none" />
      </div>

      <!-- Open Shell button — flush inside the group -->
      <button
        class="flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium text-primary hover:bg-primary hover:text-primary-content transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
        onclick={openContainerShell}
        disabled={!selectedContainerId}
      >
        <SquareTerminal class="w-4 h-4" />
        Open Shell
      </button>
    </div>

    <!-- Secondary action: SSH Host, separated visually -->
    <div class="ml-auto">
      <button
        class="btn btn-ghost btn-sm flex items-center gap-1.5 text-foreground-muted hover:text-foreground"
        onclick={openHostSSH}
        disabled={!selectedHostId}
      >
        <Server class="w-4 h-4" />
        SSH Host
      </button>
    </div>
  </div>

  <!-- Tab bar -->
  {#if tabs.length > 0}
    <div class="flex items-center bg-background-secondary border-b border-border overflow-x-auto flex-shrink-0 scrollbar-hide">
      {#each tabs as tab}
        {@const tabStatus = tabStatuses.get(tab.id) ?? "connecting"}
        <div
          role="tab"
          tabindex="0"
          class="flex items-center gap-2 px-4 py-2 text-sm border-b-2 whitespace-nowrap transition-colors cursor-pointer {activeTabId === tab.id
            ? 'border-primary text-foreground bg-background'
            : 'border-transparent text-foreground-muted hover:text-foreground'}"
          onclick={() => (activeTabId = tab.id)}
          onkeydown={(e) => e.key === "Enter" && (activeTabId = tab.id)}
          aria-selected={activeTabId === tab.id}
        >
          {#if tab.type === "host"}
            <Server class="w-3.5 h-3.5 flex-shrink-0" />
          {:else}
            <Box class="w-3.5 h-3.5 flex-shrink-0" />
          {/if}
          <span class="w-2 h-2 rounded-full flex-shrink-0 {tabStatus === 'connected' ? 'bg-running' : tabStatus === 'connecting' ? 'bg-primary animate-pulse' : tabStatus === 'error' ? 'bg-stopped' : 'bg-paused'}"></span>
          <span>{tab.label}</span>
          <span class="text-foreground-muted text-xs">@{tab.hostLabel}</span>
          <button
            class="ml-1 rounded hover:bg-background-tertiary p-0.5 text-foreground-muted hover:text-foreground"
            onclick={(e) => { e.stopPropagation(); closeTab(tab.id); }}
            aria-label="Close tab"
          >
            <X class="w-3 h-3" />
          </button>
        </div>
      {/each}
      <button
        class="flex items-center px-3 py-2 text-foreground-muted hover:text-foreground"
        onclick={() => lastOpenedType === "host" ? openHostSSH() : openContainerShell()}
        title="New tab"
        aria-label="New terminal tab"
      >
        <Plus class="w-4 h-4" />
      </button>
    </div>
  {/if}

  <!-- Terminal area -->
  <div class="flex-1 overflow-hidden relative">
    {#if tabs.length === 0}
      <!-- Empty state -->
      <div class="absolute inset-0 flex flex-col items-center justify-center gap-6 text-foreground-muted">
        <SquareTerminal class="w-16 h-16 opacity-20" />
        <div class="text-center">
          <p class="text-lg font-medium text-foreground-muted">Select a container or host above to open a shell</p>
        </div>
        <div class="flex gap-3">
          <button class="btn btn-primary flex items-center gap-2" onclick={openContainerShell} disabled={!selectedContainerId}>
            <SquareTerminal class="w-4 h-4" />
            Open Shell
          </button>
          <button class="btn btn-ghost flex items-center gap-2" onclick={openHostSSH} disabled={!selectedHostId}>
            <Server class="w-4 h-4" />
            SSH Host
          </button>
        </div>
      </div>
    {:else}
      {#each tabs as tab}
        <div class="absolute inset-0" style="display: {activeTabId === tab.id ? 'block' : 'none'};">
          {#if tab.type === "container"}
            <Terminal
              container={tab.container}
              mode="container"
              terminalMode="embedded"
              active={activeTabId === tab.id}
              onStatusChange={(s) => setTabStatus(tab.id, s)}
            />
          {:else}
            <Terminal
              host={tab.host}
              mode="host"
              terminalMode="embedded"
              active={activeTabId === tab.id}
              onStatusChange={(s) => setTabStatus(tab.id, s)}
            />
          {/if}
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

  .scrollbar-hide {
    scrollbar-width: none;
    -ms-overflow-style: none;
  }
  .scrollbar-hide::-webkit-scrollbar {
    display: none;
  }
</style>
