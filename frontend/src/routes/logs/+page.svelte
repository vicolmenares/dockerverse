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
    ChevronDown,
    Layers,
    Folder,
    CheckSquare,
    Package,
  } from "lucide-svelte";
  import { containers } from "$lib/stores/docker";
  import { language } from "$lib/stores/docker";
  import { createLogStream, API_BASE, getAuthHeaders } from "$lib/api/docker";
  import type { Container } from "$lib/api/docker";

  type LogMode = "single" | "multi" | "grouped";

  // Stack grouping types (Dozzle-inspired)
  interface DockerStack {
    name: string;
    containers: Container[];
    isCompose: boolean; // true for Docker Compose stacks, false for standalone group
    isExpanded?: boolean;
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
  let regexEnabled = $state(false);
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

  /**
   * Group containers by Docker Compose stack
   * Returns array of DockerStack objects
   */
  function groupContainersByStack(containers: Container[]): DockerStack[] {
    const byStack = new Map<string, Container[]>();
    const standalone: Container[] = [];

    // Group by stack label
    containers.forEach(c => {
      const stackLabel = c.labels?.['com.docker.compose.project'];
      if (stackLabel) {
        if (!byStack.has(stackLabel)) {
          byStack.set(stackLabel, []);
        }
        byStack.get(stackLabel)!.push(c);
      } else {
        standalone.push(c);
      }
    });

    // Convert to DockerStack array, sorted alphabetically
    const result: DockerStack[] = Array.from(byStack.entries())
      .sort(([a], [b]) => a.localeCompare(b))
      .map(([name, containers]) => ({
        name,
        containers: containers.sort((a, b) => a.name.localeCompare(b.name)),
        isCompose: true,
        isExpanded: expandedStacks.has(name)
      }));

    // Add standalone group if any
    if (standalone.length > 0) {
      result.push({
        name: 'Standalone Containers',
        containers: standalone.sort((a, b) => a.name.localeCompare(b.name)),
        isCompose: false,
        isExpanded: expandedStacks.has('Standalone Containers')
      });
    }

    return result;
  }

  /**
   * Toggle stack expansion
   */
  function toggleStack(stackName: string) {
    if (expandedStacks.has(stackName)) {
      expandedStacks.delete(stackName);
    } else {
      expandedStacks.add(stackName);
    }
    // Trigger reactivity
    expandedStacks = new Set(expandedStacks);
  }

  /**
   * Select all containers in a stack
   */
  function selectAllInStack(stackName: string) {
    const stack = groupedContainers.find((s: DockerStack) => s.name === stackName);
    if (!stack) return;

    stack.containers.forEach((c: Container) => {
      const key = containerKey(c);
      if (!selectedContainers.has(key)) {
        selectedContainers.add(key);
        startStream(c);
      }
    });
    selectedContainers = new Set(selectedContainers);
  }

  // Filter containers by host first, then by search
  let filteredByHost = $derived(
    selectedHost === 'all'
      ? $containers
      : $containers.filter(c => c.hostId === selectedHost)
  );

  // Filter containers by search (fuzzy match on name + image)
  let filteredContainers = $derived.by(() => {
    if (!searchFilter) return filteredByHost;

    return filteredByHost
      .map(c => {
        const nameMatch = fuzzyMatch(searchFilter, c.name);
        const imageMatch = fuzzyMatch(searchFilter, c.image);
        const bestMatch = nameMatch.score > imageMatch.score ? nameMatch : imageMatch;
        return { ...c, match: bestMatch.match, score: bestMatch.score };
      })
      .filter(c => c.match)
      .sort((a, b) => b.score - a.score);
  });

  // Derived: containers grouped by stack, respects search filter
  let groupedContainers = $derived.by(() => {
    return groupContainersByStack(filteredContainers);
  });

  // Compile regex pattern and error together (no state mutations inside derived)
  let regexResult = $derived.by(() => {
    if (!logSearch) return { pattern: null, error: null };

    if (regexEnabled) {
      try {
        return { pattern: new RegExp(logSearch, 'gi'), error: null };
      } catch (e) {
        return { pattern: null, error: e instanceof Error ? e.message : 'Invalid regex' };
      }
    }

    // Simple string search - escape special chars
    const escaped = logSearch.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    return { pattern: new RegExp(escaped, 'gi'), error: null };
  });

  let searchPattern = $derived(regexResult.pattern);
  let regexError = $derived(regexResult.error);

  // How many of the last lines to display (null = all)
  let displayLimit = $state<number | null>(null);
  const displayLimitOptions = [
    { label: 'All', value: null },
    { label: 'Last 100', value: 100 },
    { label: 'Last 500', value: 500 },
    { label: 'Last 1000', value: 1000 },
    { label: 'Last 2000', value: 2000 },
  ];

  // Filtered logs by search with regex support, then trimmed to displayLimit
  let filteredLogs = $derived.by(() => {
    const matched = searchPattern
      ? allLogs.filter(l => searchPattern.test(l.line))
      : allLogs;
    if (displayLimit === null) return matched;
    return matched.slice(-displayLimit);
  });

  let logAreaEl = $state<HTMLDivElement | null>(null);
  let logSearchInputEl = $state<HTMLInputElement | null>(null);

  function containerKey(c: Container): string {
    return `${c.id}@${c.hostId}`;
  }

  // Fuzzy matching for container search
  function fuzzyMatch(query: string, text: string): { match: boolean; score: number } {
    if (!query) return { match: true, score: 100 };

    const q = query.toLowerCase();
    const t = text.toLowerCase();

    // Exact substring match - highest score
    if (t.includes(q)) {
      return { match: true, score: 100 };
    }

    // Acronym match (e.g., "ngx" matches "nginx-proxy")
    const words = t.split(/[-_\s]/);
    const acronym = words.map(w => w[0]).join('');
    if (acronym.includes(q)) {
      return { match: true, score: 80 };
    }

    // Character sequence match
    let qIndex = 0;
    let score = 0;
    for (let i = 0; i < t.length && qIndex < q.length; i++) {
      if (t[i] === q[qIndex]) {
        qIndex++;
        score += 50 / q.length;
      }
    }

    return {
      match: qIndex === q.length,
      score: Math.round(score)
    };
  }

  function formatTimestamp(ts: number): string {
    if (preferences.timestampFormat === 'none') return '';
    if (preferences.timestampFormat === 'relative') {
      const diff = Date.now() - ts;
      if (diff < 60000) return `${Math.floor(diff / 1000)}s ago`;
      if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
      return `${Math.floor(diff / 3600000)}h ago`;
    }
    // absolute
    const date = new Date(ts);
    const hours = String(date.getHours()).padStart(2, '0');
    const minutes = String(date.getMinutes()).padStart(2, '0');
    const seconds = String(date.getSeconds()).padStart(2, '0');
    const ms = String(date.getMilliseconds()).padStart(3, '0');
    return `${hours}:${minutes}:${seconds}.${ms}`;
  }

  function cycleTimestampFormat() {
    const formats: Array<'absolute' | 'relative' | 'none'> = ['absolute', 'relative', 'none'];
    const idx = formats.indexOf(preferences.timestampFormat);
    preferences.timestampFormat = formats[(idx + 1) % formats.length];
    preferences.showTimestamps = preferences.timestampFormat !== 'none';
  }

  function handleKeydown(e: KeyboardEvent) {
    // Don't intercept when user is typing in an input/textarea
    if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) {
      if (e.key === 'Escape') {
        (e.target as HTMLInputElement).blur();
        if (logSearch) logSearch = '';
      }
      return;
    }
    if (e.ctrlKey || e.metaKey) {
      switch (e.key) {
        case '1': e.preventDefault(); setMode('single'); break;
        case '2': e.preventDefault(); setMode('multi'); break;
        case '3': e.preventDefault(); setMode('grouped'); break;
        case 'p': case 'P': e.preventDefault(); isPaused = !isPaused; break;
        case 'w': case 'W': e.preventDefault(); wrapLines = !wrapLines; break;
      }
    } else {
      switch (e.key) {
        case '/':
          e.preventDefault();
          logSearchInputEl?.focus();
          break;
        case 'Escape':
          if (logSearch) logSearch = '';
          else if (searchFilter) searchFilter = '';
          break;
        case ' ':
          e.preventDefault();
          isPaused = !isPaused;
          break;
      }
    }
  }

  // Highlight regex matches in log line
  function highlightMatches(line: string): string {
    if (!logSearch || !searchPattern) return line;

    return line.replace(searchPattern, (match) =>
      `<mark class="bg-primary/30 text-foreground rounded px-0.5">${match}</mark>`
    );
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
      const hostsRes = await fetch(`${API_BASE}/api/hosts`, {
        headers: getAuthHeaders()
      });
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

<svelte:window onkeydown={handleKeydown} />

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
        {#each [{ id: "single", label: t.single, key: "⌘1" }, { id: "multi", label: t.multi, key: "⌘2" }, { id: "grouped", label: t.grouped, key: "⌘3" }] as m}
          <button
            class="px-3 py-1.5 text-xs font-medium rounded-md transition-colors {mode === m.id ? 'bg-primary text-white' : 'text-foreground-muted hover:text-foreground'}"
            onclick={() => setMode(m.id as LogMode)}
            title="{m.label} ({m.key})"
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

      <!-- Stacks and containers list -->
      <div class="flex-1 overflow-auto px-2 py-2 space-y-1">
        {#each groupedContainers as stack}
          <div class="stack-group" data-stack={stack.name}>
            <!-- Stack header -->
            <div class="w-full flex items-center gap-2 px-3 py-2 rounded hover:bg-background-tertiary transition-colors group">
              <button
                onclick={() => toggleStack(stack.name)}
                class="flex items-center gap-2 flex-1"
                aria-expanded={stack.isExpanded}
                aria-label="Toggle {stack.name}"
              >
                <!-- Expand/collapse icon -->
                {#if stack.isExpanded}
                  <ChevronDown class="w-4 h-4 text-foreground-muted flex-shrink-0 transition-transform" />
                {:else}
                  <ChevronRight class="w-4 h-4 text-foreground-muted flex-shrink-0 transition-transform" />
                {/if}

                <!-- Stack icon -->
                {#if stack.isCompose}
                  <Layers class="w-4 h-4 text-primary flex-shrink-0" />
                {:else}
                  <Folder class="w-4 h-4 text-foreground-muted flex-shrink-0" />
                {/if}

                <!-- Stack name with count -->
                <span class="flex-1 text-left text-sm font-medium group-hover:text-primary transition-colors">
                  {stack.name}
                  <span class="text-foreground-muted font-normal ml-1">
                    ({stack.containers.length})
                  </span>
                </span>
              </button>

              <!-- Select all in stack -->
              <button
                onclick={(e) => {
                  e.stopPropagation();
                  selectAllInStack(stack.name);
                }}
                class="opacity-0 group-hover:opacity-100 transition-opacity p-1 hover:bg-primary/20 rounded"
                title="Select all containers in {stack.name}"
                aria-label="Select all in {stack.name}"
              >
                <CheckSquare class="w-3.5 h-3.5 text-primary" />
              </button>
            </div>

            <!-- Containers (shown when expanded) -->
            {#if stack.isExpanded}
              <div class="ml-6 mt-1 space-y-0.5 animate-in slide-in-from-top-2">
                {#each stack.containers as container}
                  {@const key = containerKey(container)}
                  {@const serviceLabel = container.labels?.['com.docker.compose.service']}
                  <label
                    class="flex items-center gap-2 px-3 py-1.5 rounded hover:bg-background-tertiary cursor-pointer group transition-all {selectedContainers.has(key) ? 'bg-primary/10' : ''}"
                  >
                    <!-- Checkbox -->
                    <input
                      type="checkbox"
                      checked={selectedContainers.has(key)}
                      onchange={() => toggleContainer(container)}
                      class="w-3.5 h-3.5 accent-primary focus:ring-2 focus:ring-primary focus:ring-offset-1 focus:ring-offset-background-secondary rounded"
                      aria-label="Select {container.name}"
                    />

                    <!-- Status indicator -->
                    <div
                      class="w-2 h-2 rounded-full flex-shrink-0"
                      class:bg-running={container.state === 'running'}
                      class:bg-stopped={container.state === 'exited'}
                      class:bg-paused={container.state === 'paused'}
                      class:bg-primary={container.state === 'restarting'}
                      class:animate-pulse={container.state === 'restarting'}
                      title={container.state}
                    ></div>

                    <!-- Container name -->
                    <span class="flex-1 text-sm group-hover:text-primary transition-colors truncate" title={container.name}>
                      {container.name}
                    </span>

                    <!-- Service badge (if from Compose) -->
                    {#if serviceLabel}
                      <span class="px-1.5 py-0.5 text-xs bg-background-tertiary text-foreground-muted rounded border border-border">
                        {serviceLabel}
                      </span>
                    {/if}
                  </label>
                {/each}
              </div>
            {/if}
          </div>
        {/each}

        <!-- Empty state -->
        {#if groupedContainers.length === 0}
          <div class="flex flex-col items-center justify-center py-8 text-center">
            <Package class="w-12 h-12 text-foreground-muted mb-3" />
            <p class="text-sm text-foreground-muted">
              No containers found
              {#if selectedHost !== 'all'}
                on {hosts.find(h => h.id === selectedHost)?.name}
              {/if}
            </p>
          </div>
        {/if}
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
          title={isPaused ? "Resume (⌘P or Space)" : "Pause (⌘P or Space)"}
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
          title="Toggle line wrap (⌘W)"
        >
          <WrapText class="w-3 h-3" />
          {t.wrap}
        </button>

        <!-- Font Size -->
        <div class="flex items-center gap-0.5">
          <button
            class="btn-icon hover:bg-background-tertiary disabled:opacity-30"
            onclick={() => preferences.fontSize = Math.max(10, preferences.fontSize - 1)}
            disabled={preferences.fontSize <= 10}
            title="Decrease font size"
          >
            <Type class="w-3 h-3" />
          </button>
          <span class="text-[10px] text-foreground-muted px-1 min-w-[2rem] text-center">{preferences.fontSize}px</span>
          <button
            class="btn-icon hover:bg-background-tertiary disabled:opacity-30"
            onclick={() => preferences.fontSize = Math.min(24, preferences.fontSize + 1)}
            disabled={preferences.fontSize >= 24}
            title="Increase font size"
          >
            <Type class="w-4 h-4" />
          </button>
        </div>

        <!-- Timestamps (cycle: absolute → relative → none) -->
        <button
          class="flex items-center gap-1 px-2 py-1 text-xs rounded-md transition-colors {preferences.timestampFormat !== 'none' ? 'bg-primary/15 text-primary' : 'text-foreground-muted hover:text-foreground'}"
          onclick={cycleTimestampFormat}
          title="Timestamp format: {preferences.timestampFormat} (click to cycle)"
        >
          {preferences.timestampFormat === 'absolute' ? '12:34' : preferences.timestampFormat === 'relative' ? '~ago' : '···'}
        </button>

        <div class="flex-1"></div>

        <!-- Last N lines selector -->
        <select
          bind:value={displayLimit}
          class="text-xs bg-background border border-border rounded-md px-2 py-1 text-foreground focus:border-primary focus:outline-none"
          title="Number of lines to display"
        >
          {#each displayLimitOptions as opt}
            <option value={opt.value}>{opt.label}</option>
          {/each}
        </select>

        <!-- Search logs with regex toggle -->
        <div class="flex items-center gap-1">
          <div class="relative">
            <Search class="absolute left-2 top-1.5 w-3.5 h-3.5 text-foreground-muted" />
            <input
              type="text"
              bind:this={logSearchInputEl}
              bind:value={logSearch}
              placeholder={regexEnabled ? "Regex pattern..." : t.searchLogs}
              title="Search logs (press / to focus)"
              class="pl-7 pr-3 py-1 text-xs bg-background border {regexError ? 'border-destructive' : 'border-border'} rounded-md text-foreground placeholder:text-foreground-muted focus:border-primary focus:outline-none w-40"
            />
            {#if regexError}
              <div class="absolute top-full left-0 mt-1 text-xs text-destructive bg-background border border-destructive rounded px-2 py-1 whitespace-nowrap z-10">
                {regexError}
              </div>
            {/if}
          </div>
          <button
            class="btn-icon hover:bg-background-tertiary {regexEnabled ? 'text-primary bg-primary/15' : 'text-foreground-muted'}"
            onclick={() => (regexEnabled = !regexEnabled)}
            title={regexEnabled ? "Disable regex" : "Enable regex"}
          >
            <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
            </svg>
          </button>
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
                class="flex-1 overflow-y-auto p-3 font-mono leading-relaxed {wrapLines ? 'whitespace-pre-wrap break-all' : 'whitespace-pre overflow-x-auto'}"
                style="font-size: {preferences.fontSize}px"
              >
                {#each containerLogs as entry}
                  <div class="text-foreground-muted hover:text-foreground hover:bg-background-tertiary/30 px-1 -mx-1 rounded">
                    {#if preferences.showTimestamps}
                      <span class="text-foreground-muted/50 select-none">{formatTimestamp(entry.ts)} </span>
                    {/if}
                    {@html highlightMatches(entry.line)}
                  </div>
                {/each}
              </div>
            </div>
          {/each}
        </div>
      {:else}
        <!-- Single and Multi mode: unified log view -->
        <div
          bind:this={logAreaEl}
          class="flex-1 overflow-y-auto p-3 font-mono leading-relaxed {wrapLines ? 'whitespace-pre-wrap break-all' : 'whitespace-pre overflow-x-auto'}"
          style="font-size: {preferences.fontSize}px"
        >
          {#each filteredLogs as entry}
            <div class="hover:bg-background-tertiary/30 px-1 -mx-1 rounded">
              {#if preferences.showTimestamps}
                <span class="text-foreground-muted/50 select-none">{formatTimestamp(entry.ts)} </span>
              {/if}
              {#if mode === "multi"}
                <span class="{entry.color} font-semibold">[{entry.name}]</span>{" "}
              {/if}
              <span class="text-foreground-muted">{@html highlightMatches(entry.line)}</span>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  </div>
</div>

<style>
  @keyframes slide-in-from-top-2 {
    from {
      opacity: 0;
      transform: translateY(-8px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  .animate-in {
    animation-duration: 200ms;
    animation-timing-function: ease-out;
  }

  .slide-in-from-top-2 {
    animation-name: slide-in-from-top-2;
  }
</style>
