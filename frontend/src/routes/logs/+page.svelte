<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import {
    ScrollText,
    Search,
    Pause,
    Play,
    ArrowDown,
    Copy,
    Download,
    X,
    Check,
    Loader2,
    Type,
    WrapText,
    Wifi,
    WifiOff,
    ChevronRight,
  } from "lucide-svelte";
  import { containers } from "$lib/stores/docker";
  import { language } from "$lib/stores/docker";
  import { createLogStream, API_BASE } from "$lib/api/docker";
  import type { Container } from "$lib/api/docker";

  type LogMode = "single" | "multi" | "grouped";

  // Stack grouping types (Dozzle-inspired)
  interface DockerStack {
    name: string;
    containers: Container[];
    isCompose: boolean; // true for Docker Compose stacks, false for standalone group
    isExpanded?: boolean;
  }

  // Enhanced Container interface
  interface Container {
    id: string;
    name: string;
    hostId: string;
    state: 'running' | 'exited' | 'paused' | 'restarting';
    stack?: string; // com.docker.compose.project
    service?: string; // com.docker.compose.service
    labels?: Record<string, string>;
  }

  // Host interface
  interface Host {
    id: string;
    name: string;
    address: string;
    type: 'local' | 'remote';
  }

  // Log display preferences (Dockhand-inspired)
  interface LogPreferences {
    fontSize: number; // pixels (10-24)
    lineWrap: boolean;
    autoScroll: boolean;
    showTimestamps: boolean;
    timestampFormat: 'absolute' | 'relative' | 'none';
    maxLines: number; // buffer limit
  }

  // State
  let mode = $state<LogMode>("single");
  let selectedContainers = $state<Set<string>>(new Set());
  let searchFilter = $state("");
  let logSearch = $state("");
  let isPaused = $state(false);
  let autoScroll = $state(true);
  let wrapLines = $state(true);
  let isLive = $state(false);

  // Navigation state (Dozzle-inspired)
  let selectedHost = $state<string | 'all'>('all');
  let expandedStacks = $state<Set<string>>(new Set());
  let stacks = $state<DockerStack[]>([]);
  let hosts = $state<Host[]>([]);

  // Preferences state (Dockhand-inspired)
  let preferences = $state<LogPreferences>({
    fontSize: 14,
    lineWrap: true,
    autoScroll: true,
    showTimestamps: true,
    timestampFormat: 'absolute',
    maxLines: 1000
  });

  // Log entries: { containerKey, containerName, line, color }
  type LogEntry = {
    key: string;
    name: string;
    line: string;
    color: string;
    ts: number;
  };
  let allLogs = $state<LogEntry[]>([]);
  let logsByContainer = $state<Map<string, LogEntry[]>>(new Map());

  // Active streams
  let activeStreams = new Map<string, EventSource>();

  // Container colors for multi-mode
  const colorPalette = [
    "text-primary",
    "text-accent-cyan",
    "text-accent-orange",
    "text-accent-purple",
    "text-running",
    "text-accent-red",
  ];
  let containerColors = new Map<string, string>();

  function getContainerColor(key: string): string {
    if (!containerColors.has(key)) {
      const idx = containerColors.size % colorPalette.length;
      containerColors.set(key, colorPalette[idx]);
    }
    return containerColors.get(key)!;
  }

  // Filter containers by host first, then by search
  let filteredByHost = $derived(
    selectedHost === 'all'
      ? $containers
      : $containers.filter(c => c.hostId === selectedHost)
  );

  let filteredContainers = $derived(
    filteredByHost.filter(
      (c) =>
        c.name.toLowerCase().includes(searchFilter.toLowerCase()) ||
        c.image.toLowerCase().includes(searchFilter.toLowerCase()),
    ),
  );

  // Filtered logs by search
  let filteredLogs = $derived(
    logSearch
      ? allLogs.filter((l) =>
          l.line.toLowerCase().includes(logSearch.toLowerCase()),
        )
      : allLogs,
  );

  let logAreaEl = $state<HTMLDivElement | null>(null);

  function containerKey(c: Container): string {
    return `${c.id}@${c.hostId}`;
  }

  function toggleContainer(c: Container) {
    const key = containerKey(c);
    const newSet = new Set(selectedContainers);
    if (newSet.has(key)) {
      newSet.delete(key);
      stopStream(key);
    } else {
      if (mode === "single") {
        // In single mode, deselect all others
        for (const k of newSet) stopStream(k);
        newSet.clear();
      }
      newSet.add(key);
      startStream(c);
    }
    selectedContainers = newSet;
  }

  function selectAll() {
    const newSet = new Set(selectedContainers);
    for (const c of filteredContainers) {
      const key = containerKey(c);
      if (!newSet.has(key)) {
        newSet.add(key);
        startStream(c);
      }
    }
    selectedContainers = newSet;
  }

  function clearAll() {
    for (const key of selectedContainers) stopStream(key);
    selectedContainers = new Set();
    allLogs = [];
    logsByContainer = new Map();
  }

  function startStream(c: Container) {
    const key = containerKey(c);
    if (activeStreams.has(key)) return;

    const color = getContainerColor(key);
    isLive = true;

    // First fetch initial logs
    fetchInitialLogs(c, key, color);

    // Then start SSE stream
    const es = createLogStream(c.hostId, c.id, (line: string) => {
      if (isPaused) return;
      const entry: LogEntry = {
        key,
        name: c.name,
        line,
        color,
        ts: Date.now(),
      };
      allLogs = [...allLogs, entry];

      const existing = logsByContainer.get(key) || [];
      logsByContainer = new Map(logsByContainer).set(key, [...existing, entry]);

      if (autoScroll) scrollToBottom();
    });
    activeStreams.set(key, es);
  }

  async function fetchInitialLogs(
    c: Container,
    key: string,
    color: string,
  ) {
    try {
      const token =
        typeof localStorage !== "undefined"
          ? localStorage.getItem("auth_access_token")
          : null;
      const headers: Record<string, string> = {
        "Content-Type": "application/json",
      };
      if (token) headers["Authorization"] = `Bearer ${token}`;

      const res = await fetch(
        `${API_BASE}/api/logs/${c.hostId}/${c.id}?tail=200`,
        { headers },
      );
      if (res.ok) {
        const data = await res.json();
        const lines = Array.isArray(data) ? data : (data.logs || []);
        if (lines.length > 0) {
          const entries: LogEntry[] = lines.map((line: string) => ({
            key,
            name: c.name,
            line,
            color,
            ts: Date.now(),
          }));
          allLogs = [...allLogs, ...entries];

          const existingLogs = logsByContainer.get(key) || [];
          logsByContainer = new Map(logsByContainer).set(key, [...existingLogs, ...entries]);

          if (autoScroll) scrollToBottom();
        }
      }
    } catch {
      // Silently fail initial fetch
    }
  }

  function stopStream(key: string) {
    const es = activeStreams.get(key);
    if (es) {
      es.close();
      activeStreams.delete(key);
    }
    if (activeStreams.size === 0) isLive = false;
  }

  function scrollToBottom() {
    requestAnimationFrame(() => {
      if (logAreaEl) {
        logAreaEl.scrollTop = logAreaEl.scrollHeight;
      }
    });
  }

  function copyLogs() {
    const text = filteredLogs.map((l) => `[${l.name}] ${l.line}`).join("\n");
    navigator.clipboard.writeText(text);
  }

  function downloadLogs() {
    const text = filteredLogs.map((l) => `[${l.name}] ${l.line}`).join("\n");
    const blob = new Blob([text], { type: "text/plain" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `dockerverse-logs-${new Date().toISOString().slice(0, 10)}.txt`;
    a.click();
    URL.revokeObjectURL(url);
  }

  function setMode(m: LogMode) {
    mode = m;
    if (m === "single" && selectedContainers.size > 1) {
      // Keep only the first selected
      const firstKey = [...selectedContainers][0];
      for (const key of selectedContainers) {
        if (key !== firstKey) stopStream(key);
      }
      selectedContainers = new Set([firstKey]);
    }
  }

  // Cleanup on destroy
  onDestroy(() => {
    for (const [key] of activeStreams) stopStream(key);
  });

  // Fetch hosts on mount
  onMount(async () => {
    try {
      const hostsRes = await fetch(`${API_BASE}/api/hosts`);
      if (hostsRes.ok) {
        hosts = await hostsRes.json();
      }
    } catch (err) {
      console.error('Failed to fetch hosts:', err);
    }
  });

  let t = $derived({
    title: $language === "es" ? "Logs" : "Logs",
    single: $language === "es" ? "Individual" : "Single",
    multi: $language === "es" ? "Múltiple" : "Multi",
    grouped: $language === "es" ? "Agrupado" : "Grouped",
    searchContainers: $language === "es" ? "Buscar contenedores..." : "Search containers...",
    searchLogs: $language === "es" ? "Buscar en logs..." : "Search logs...",
    selectAll: $language === "es" ? "Seleccionar todos" : "Select all",
    clear: $language === "es" ? "Limpiar" : "Clear",
    live: "Live",
    paused: $language === "es" ? "Pausado" : "Paused",
    copy: $language === "es" ? "Copiar" : "Copy",
    download: $language === "es" ? "Descargar" : "Download",
    noSelection: $language === "es" ? "Selecciona un contenedor para ver logs" : "Select a container to view logs",
    autoScroll: "Auto-scroll",
    wrap: "Wrap",
  });
</script>

<div class="flex flex-col h-[calc(100vh-7rem)]">
  <!-- Header Bar -->
  <div class="flex items-center justify-between mb-4">
    <div class="flex items-center gap-3">
      <ScrollText class="w-6 h-6 text-primary" />
      <h2 class="text-xl font-bold text-foreground">{t.title}</h2>
    </div>

    <!-- Mode Switcher -->
    <div class="flex items-center gap-2">
      <div class="flex bg-background-tertiary/50 rounded-lg p-0.5">
        {#each [{ id: "single", label: t.single }, { id: "multi", label: t.multi }, { id: "grouped", label: t.grouped }] as m}
          <button
            class="px-3 py-1.5 text-xs font-medium rounded-md transition-colors {mode === m.id ? 'bg-primary text-white' : 'text-foreground-muted hover:text-foreground'}"
            onclick={() => setMode(m.id as LogMode)}
          >
            {m.label}
          </button>
        {/each}
      </div>

      <!-- Connection status -->
      <span class="flex items-center gap-1.5 px-2 py-1 text-xs rounded-full {isLive ? 'bg-running/15 text-running' : 'bg-foreground-muted/15 text-foreground-muted'}">
        {#if isLive}
          <Wifi class="w-3 h-3" />
          {isPaused ? t.paused : t.live}
        {:else}
          <WifiOff class="w-3 h-3" />
          {$language === "es" ? "Desconectado" : "Disconnected"}
        {/if}
      </span>
    </div>
  </div>

  <!-- Main Content -->
  <div class="flex gap-4 flex-1 min-h-0">
    <!-- Left Sidebar: Container Selection -->
    <div class="w-64 flex-shrink-0 flex flex-col bg-background-secondary border border-border rounded-xl overflow-hidden">
      <!-- Host selector breadcrumb -->
      <div class="flex items-center gap-2 px-4 py-3 border-b border-border bg-background-tertiary">
        <span class="text-sm text-foreground-muted font-medium">Hosts</span>
        <ChevronRight class="w-4 h-4 text-foreground-muted" />

        <div class="relative flex-1">
          <select
            bind:value={selectedHost}
            class="w-full text-sm font-medium bg-transparent border border-border rounded px-3 py-1.5 focus:ring-2 focus:ring-primary focus:outline-none cursor-pointer"
            aria-label="Select Docker host"
          >
            <option value="all">All Hosts ({$containers.length} containers)</option>
            {#each hosts as host}
              <option value={host.id}>
                {host.name} ({$containers.filter(c => c.hostId === host.id).length} containers)
              </option>
            {/each}
          </select>
        </div>
      </div>

      <!-- Search -->
      <div class="p-3 border-b border-border">
        <div class="relative">
          <Search class="absolute left-2.5 top-2 w-4 h-4 text-foreground-muted" />
          <input
            type="text"
            bind:value={searchFilter}
            placeholder={t.searchContainers}
            class="w-full pl-8 pr-3 py-1.5 text-sm bg-background border border-border rounded-lg text-foreground placeholder:text-foreground-muted focus:border-primary focus:outline-none"
          />
        </div>
        <div class="flex gap-2 mt-2">
          <button
            class="text-[10px] text-primary hover:text-primary/80"
            onclick={selectAll}
          >
            {t.selectAll}
          </button>
          <button
            class="text-[10px] text-foreground-muted hover:text-foreground"
            onclick={clearAll}
          >
            {t.clear}
          </button>
        </div>
      </div>

      <!-- Container List -->
      <div class="flex-1 overflow-y-auto p-2 space-y-0.5">
        {#each filteredContainers as c}
          {@const key = containerKey(c)}
          {@const isSelected = selectedContainers.has(key)}
          <button
            class="w-full flex items-center gap-2 px-2.5 py-2 rounded-lg text-left transition-colors {isSelected ? 'bg-primary/10 border border-primary/30' : 'hover:bg-background-tertiary'}"
            onclick={() => toggleContainer(c)}
          >
            <span class="w-2 h-2 rounded-full flex-shrink-0 {c.state === 'running' ? 'bg-running' : 'bg-stopped'}"></span>
            <div class="min-w-0 flex-1">
              <p class="text-xs font-medium text-foreground truncate">{c.name}</p>
              <p class="text-[10px] text-foreground-muted truncate">{c.image}</p>
            </div>
            {#if isSelected}
              <Check class="w-3.5 h-3.5 text-primary flex-shrink-0" />
            {/if}
          </button>
        {/each}
      </div>
    </div>

    <!-- Log Area -->
    <div class="flex-1 flex flex-col bg-background-secondary border border-border rounded-xl overflow-hidden">
      <!-- Toolbar -->
      <div class="flex items-center gap-2 px-3 py-2 border-b border-border bg-background-tertiary/30">
        <!-- Pause/Play -->
        <button
          class="btn-icon hover:bg-background-tertiary"
          onclick={() => (isPaused = !isPaused)}
          title={isPaused ? "Resume" : "Pause"}
        >
          {#if isPaused}
            <Play class="w-4 h-4 text-running" />
          {:else}
            <Pause class="w-4 h-4" />
          {/if}
        </button>

        <!-- Auto-scroll -->
        <button
          class="flex items-center gap-1 px-2 py-1 text-xs rounded-md transition-colors {autoScroll ? 'bg-primary/15 text-primary' : 'text-foreground-muted hover:text-foreground'}"
          onclick={() => { autoScroll = !autoScroll; if (autoScroll) scrollToBottom(); }}
        >
          <ArrowDown class="w-3 h-3" />
          {t.autoScroll}
        </button>

        <!-- Wrap -->
        <button
          class="flex items-center gap-1 px-2 py-1 text-xs rounded-md transition-colors {wrapLines ? 'bg-primary/15 text-primary' : 'text-foreground-muted hover:text-foreground'}"
          onclick={() => (wrapLines = !wrapLines)}
        >
          <WrapText class="w-3 h-3" />
          {t.wrap}
        </button>

        <div class="flex-1"></div>

        <!-- Search logs -->
        <div class="relative">
          <Search class="absolute left-2 top-1.5 w-3.5 h-3.5 text-foreground-muted" />
          <input
            type="text"
            bind:value={logSearch}
            placeholder={t.searchLogs}
            class="pl-7 pr-3 py-1 text-xs bg-background border border-border rounded-md text-foreground placeholder:text-foreground-muted focus:border-primary focus:outline-none w-40"
          />
        </div>

        <!-- Copy -->
        <button
          class="btn-icon hover:bg-background-tertiary"
          onclick={copyLogs}
          title={t.copy}
        >
          <Copy class="w-4 h-4" />
        </button>

        <!-- Download -->
        <button
          class="btn-icon hover:bg-background-tertiary"
          onclick={downloadLogs}
          title={t.download}
        >
          <Download class="w-4 h-4" />
        </button>
      </div>

      <!-- Log Content -->
      {#if selectedContainers.size === 0}
        <div class="flex-1 flex items-center justify-center">
          <div class="text-center">
            <ScrollText class="w-12 h-12 text-foreground-muted/30 mx-auto mb-3" />
            <p class="text-sm text-foreground-muted">{t.noSelection}</p>
          </div>
        </div>
      {:else if mode === "grouped"}
        <!-- Grouped mode: side by side -->
        <div class="flex-1 flex min-h-0 divide-x divide-border overflow-x-auto">
          {#each [...selectedContainers] as key}
            {@const containerLogs = logsByContainer.get(key) || []}
            {@const name = containerLogs[0]?.name || key.split("@")[0]}
            {@const color = getContainerColor(key)}
            <div class="flex-1 min-w-[300px] flex flex-col">
              <div class="px-3 py-1.5 border-b border-border bg-background-tertiary/20">
                <span class="text-xs font-medium {color}">{name}</span>
              </div>
              <div
                class="flex-1 overflow-y-auto p-3 font-mono text-xs leading-relaxed {wrapLines ? 'whitespace-pre-wrap break-all' : 'whitespace-pre overflow-x-auto'}"
              >
                {#each containerLogs as entry}
                  <div class="text-foreground-muted hover:text-foreground hover:bg-background-tertiary/30 px-1 -mx-1 rounded">{entry.line}</div>
                {/each}
              </div>
            </div>
          {/each}
        </div>
      {:else}
        <!-- Single and Multi mode: unified log view -->
        <div
          bind:this={logAreaEl}
          class="flex-1 overflow-y-auto p-3 font-mono text-xs leading-relaxed {wrapLines ? 'whitespace-pre-wrap break-all' : 'whitespace-pre overflow-x-auto'}"
        >
          {#each filteredLogs as entry}
            <div class="hover:bg-background-tertiary/30 px-1 -mx-1 rounded">
              {#if mode === "multi"}
                <span class="{entry.color} font-semibold">[{entry.name}]</span>{" "}
              {/if}
              <span class="text-foreground-muted">{entry.line}</span>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  </div>
</div>
