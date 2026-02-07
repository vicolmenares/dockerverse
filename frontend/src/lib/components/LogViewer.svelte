<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import {
    X,
    Maximize2,
    Minimize2,
    Download,
    Trash2,
    Pause,
    Play,
    GripHorizontal,
    Search,
    Filter,
    Calendar,
    Clock,
    AlertTriangle,
    AlertCircle,
    Info,
    Bug,
    ChevronDown,
  } from "lucide-svelte";
  import type { Container } from "$lib/api/docker";
  import { API_BASE } from "$lib/api/docker";
  import { language } from "$lib/stores/docker";

  let { container, onClose }: { container: Container; onClose: () => void } =
    $props();

  let logs = $state<Array<{ line: string; timestamp?: Date; level?: string }>>(
    [],
  );
  let logsContainer: HTMLDivElement;
  let isFullscreen = $state(false);
  let isPaused = $state(false);
  let autoScroll = $state(true);
  let eventSource: EventSource | null = null;

  // Filter states
  let searchQuery = $state("");
  let showFilters = $state(false);
  let selectedLevels = $state<string[]>(["error", "warning", "info", "debug"]);
  let dateFrom = $state("");
  let dateTo = $state("");
  let timeFrom = $state("");
  let timeTo = $state("");

  // Dragging state
  let isDragging = $state(false);
  let dragOffset = $state({ x: 0, y: 0 });
  let position = $state({ x: 0, y: 0 });
  let windowElement: HTMLDivElement;

  // Translations
  const texts = {
    es: {
      waitingLogs: "Esperando logs...",
      lines: "líneas",
      autoScroll: "Auto-scroll",
      search: "Buscar en logs...",
      filters: "Filtros",
      dateRange: "Rango de fecha",
      from: "Desde",
      to: "Hasta",
      levels: "Niveles",
      error: "Error",
      warning: "Warning",
      info: "Info",
      debug: "Debug",
      all: "Todos",
      none: "Ninguno",
      clearFilters: "Limpiar filtros",
      showing: "Mostrando",
      of: "de",
      matchingFilters: "que coinciden con los filtros",
      connectionLost: "[Conexión perdida - reintentando...]",
    },
    en: {
      waitingLogs: "Waiting for logs...",
      lines: "lines",
      autoScroll: "Auto-scroll",
      search: "Search logs...",
      filters: "Filters",
      dateRange: "Date range",
      from: "From",
      to: "To",
      levels: "Levels",
      error: "Error",
      warning: "Warning",
      info: "Info",
      debug: "Debug",
      all: "All",
      none: "None",
      clearFilters: "Clear filters",
      showing: "Showing",
      of: "of",
      matchingFilters: "matching filters",
      connectionLost: "[Connection lost - retrying...]",
    },
  };

  let t = $derived(texts[$language]);

  // Filtered logs based on search, levels, and date range
  let filteredLogs = $derived(() => {
    let result = logs;

    // Filter by search query
    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      result = result.filter((log) => log.line.toLowerCase().includes(query));
    }

    // Filter by levels
    if (selectedLevels.length < 4) {
      result = result.filter((log) => {
        if (!log.level) return true; // Show logs without detected level
        return selectedLevels.includes(log.level);
      });
    }

    // Filter by date range
    if (dateFrom || dateTo || timeFrom || timeTo) {
      const fromDate = dateFrom
        ? new Date(dateFrom + (timeFrom ? `T${timeFrom}` : "T00:00:00"))
        : null;
      const toDate = dateTo
        ? new Date(dateTo + (timeTo ? `T${timeTo}` : "T23:59:59"))
        : null;

      result = result.filter((log) => {
        if (!log.timestamp) return true;
        if (fromDate && log.timestamp < fromDate) return false;
        if (toDate && log.timestamp > toDate) return false;
        return true;
      });
    }

    return result;
  });

  onMount(() => {
    startLogStream();
  });

  onDestroy(() => {
    eventSource?.close();
  });

  function detectLogLevel(line: string): string | undefined {
    const lowerLine = line.toLowerCase();
    if (
      lowerLine.includes("error") ||
      lowerLine.includes("err]") ||
      lowerLine.includes("[err")
    )
      return "error";
    if (lowerLine.includes("warn") || lowerLine.includes("warning"))
      return "warning";
    if (lowerLine.includes("info") || lowerLine.includes("[inf")) return "info";
    if (lowerLine.includes("debug") || lowerLine.includes("[dbg"))
      return "debug";
    return undefined;
  }

  function extractTimestamp(line: string): Date | undefined {
    // Try common timestamp patterns
    const patterns = [
      /\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}/, // ISO format
      /\d{2}\/\d{2}\/\d{4} \d{2}:\d{2}:\d{2}/, // DD/MM/YYYY HH:MM:SS
      /\w{3} \d{2} \d{2}:\d{2}:\d{2}/, // Mon DD HH:MM:SS (syslog)
    ];

    for (const pattern of patterns) {
      const match = line.match(pattern);
      if (match) {
        const date = new Date(match[0]);
        if (!isNaN(date.getTime())) return date;
      }
    }
    return undefined;
  }

  function startLogStream() {
    const apiUrl = API_BASE || "";
    const token =
      typeof localStorage !== "undefined"
        ? localStorage.getItem("auth_access_token")
        : null;
    const baseUrl = `${apiUrl}/api/logs/${container.hostId}/${container.id}/stream`;
    const url = token
      ? `${baseUrl}?token=${encodeURIComponent(token)}`
      : baseUrl;
    eventSource = new EventSource(url);

    eventSource.onmessage = (event) => {
      if (isPaused) return;

      try {
        const data = JSON.parse(event.data);
        const line = data.line || data.data || event.data;
        if (line) {
          const logEntry = {
            line,
            timestamp: extractTimestamp(line) || new Date(),
            level: detectLogLevel(line),
          };
          logs = [...logs, logEntry];

          // Keep only last 1000 lines
          if (logs.length > 1000) {
            logs = logs.slice(-1000);
          }

          // Auto scroll
          if (autoScroll && logsContainer) {
            requestAnimationFrame(() => {
              logsContainer.scrollTop = logsContainer.scrollHeight;
            });
          }
        }
      } catch (e) {
        // Raw text message
        const logEntry = {
          line: event.data,
          timestamp: new Date(),
          level: detectLogLevel(event.data),
        };
        logs = [...logs, logEntry];
      }
    };

    eventSource.onerror = () => {
      logs = [
        ...logs,
        {
          line: `\x1b[31m${t.connectionLost}\x1b[0m`,
          timestamp: new Date(),
          level: "error",
        },
      ];
    };
  }

  function clearLogs() {
    logs = [];
  }

  function togglePause() {
    isPaused = !isPaused;
  }

  function downloadLogs() {
    const content = filteredLogs()
      .map((l) => l.line)
      .join("\n");
    const blob = new Blob([content], { type: "text/plain" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `${container.name}-logs-${new Date().toISOString()}.txt`;
    a.click();
    URL.revokeObjectURL(url);
  }

  function toggleFullscreen() {
    isFullscreen = !isFullscreen;
    if (isFullscreen) {
      position = { x: 0, y: 0 };
    }
  }

  function clearFilters() {
    searchQuery = "";
    selectedLevels = ["error", "warning", "info", "debug"];
    dateFrom = "";
    dateTo = "";
    timeFrom = "";
    timeTo = "";
  }

  function toggleLevel(level: string) {
    if (selectedLevels.includes(level)) {
      selectedLevels = selectedLevels.filter((l) => l !== level);
    } else {
      selectedLevels = [...selectedLevels, level];
    }
  }

  function startDrag(e: MouseEvent) {
    if (isFullscreen) return;
    isDragging = true;
    const rect = windowElement.getBoundingClientRect();
    dragOffset = {
      x: e.clientX - rect.left,
      y: e.clientY - rect.top,
    };
    document.addEventListener("mousemove", onDrag);
    document.addEventListener("mouseup", stopDrag);
  }

  function onDrag(e: MouseEvent) {
    if (!isDragging) return;
    position = {
      x: e.clientX - dragOffset.x,
      y: e.clientY - dragOffset.y,
    };
  }

  function stopDrag() {
    isDragging = false;
    document.removeEventListener("mousemove", onDrag);
    document.removeEventListener("mouseup", stopDrag);
  }

  // Parse ANSI colors to HTML
  function parseAnsi(text: string): string {
    const ansiMap: Record<string, string> = {
      "30": "#15161e",
      "31": "#f7768e",
      "32": "#9ece6a",
      "33": "#e0af68",
      "34": "#7aa2f7",
      "35": "#bb9af7",
      "36": "#7dcfff",
      "37": "#a9b1d6",
      "90": "#414868",
      "91": "#f7768e",
      "92": "#9ece6a",
      "93": "#e0af68",
      "94": "#7aa2f7",
      "95": "#bb9af7",
      "96": "#7dcfff",
      "97": "#c0caf5",
    };

    return text
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/\x1b\[(\d+)m/g, (_, code) => {
        if (code === "0") return "</span>";
        const color = ansiMap[code];
        return color ? `<span style="color: ${color}">` : "";
      });
  }

  function getLevelIcon(level: string | undefined) {
    switch (level) {
      case "error":
        return AlertCircle;
      case "warning":
        return AlertTriangle;
      case "info":
        return Info;
      case "debug":
        return Bug;
      default:
        return null;
    }
  }

  function getLevelColor(level: string | undefined) {
    switch (level) {
      case "error":
        return "text-stopped";
      case "warning":
        return "text-paused";
      case "info":
        return "text-primary";
      case "debug":
        return "text-foreground-muted";
      default:
        return "text-foreground";
    }
  }
</script>

<div
  bind:this={windowElement}
  class="fixed z-50 bg-background-secondary border border-border rounded-lg shadow-2xl flex flex-col overflow-hidden"
  class:inset-4={isFullscreen}
  class:w-[800px]={!isFullscreen}
  class:h-[500px]={!isFullscreen}
  style={!isFullscreen && (position.x !== 0 || position.y !== 0)
    ? `left: ${position.x}px; top: ${position.y}px;`
    : !isFullscreen
      ? "top: 1rem; left: 1rem;"
      : ""}
>
  <!-- Header (Draggable) -->
  <div
    role="toolbar"
    class="flex items-center justify-between px-4 py-2 bg-background-tertiary border-b border-border select-none"
    class:cursor-grab={!isFullscreen && !isDragging}
    class:cursor-grabbing={isDragging}
    onmousedown={startDrag}
  >
    <div class="flex items-center gap-3">
      {#if !isFullscreen}
        <GripHorizontal class="w-4 h-4 text-foreground-muted" />
      {/if}
      <div class="flex gap-1.5">
        <span class="w-3 h-3 rounded-full bg-accent-red"></span>
        <span class="w-3 h-3 rounded-full bg-paused"></span>
        <span class="w-3 h-3 rounded-full bg-running"></span>
      </div>
      <span class="text-sm font-medium text-foreground">
        Logs: {container.name}
        <span class="text-foreground-muted">@{container.hostId}</span>
      </span>
      {#if isPaused}
        <span class="text-xs px-2 py-0.5 bg-paused/20 text-paused rounded"
          >PAUSED</span
        >
      {/if}
    </div>

    <div class="flex items-center gap-1">
      <button
        class="btn-icon"
        onclick={togglePause}
        title={isPaused ? "Resume" : "Pause"}
      >
        {#if isPaused}
          <Play class="w-4 h-4" />
        {:else}
          <Pause class="w-4 h-4" />
        {/if}
      </button>
      <button class="btn-icon" onclick={clearLogs} title="Clear">
        <Trash2 class="w-4 h-4" />
      </button>
      <button class="btn-icon" onclick={downloadLogs} title="Download">
        <Download class="w-4 h-4" />
      </button>
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
      <button
        class="btn-icon hover:text-accent-red"
        onclick={onClose}
        title="Close"
      >
        <X class="w-4 h-4" />
      </button>
    </div>
  </div>

  <!-- Search and Filters Bar -->
  <div class="px-4 py-2 bg-background border-b border-border space-y-2">
    <div class="flex items-center gap-2">
      <!-- Search Input -->
      <div class="flex-1 relative">
        <Search
          class="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-foreground-muted"
        />
        <input
          type="text"
          bind:value={searchQuery}
          placeholder={t.search}
          class="w-full pl-9 pr-4 py-1.5 text-sm bg-background-secondary border border-border rounded-lg
                 text-foreground placeholder:text-foreground-muted focus:outline-none focus:border-primary/50"
        />
      </div>

      <!-- Filter Toggle -->
      <button
        class="btn btn-ghost btn-sm flex items-center gap-1 {showFilters
          ? 'text-primary'
          : ''}"
        onclick={() => (showFilters = !showFilters)}
      >
        <Filter class="w-4 h-4" />
        {t.filters}
        <ChevronDown
          class="w-3 h-3 transition-transform {showFilters ? 'rotate-180' : ''}"
        />
      </button>
    </div>

    <!-- Expanded Filters -->
    {#if showFilters}
      <div
        class="grid grid-cols-2 lg:grid-cols-4 gap-3 pt-2 border-t border-border"
      >
        <!-- Date From -->
        <div class="space-y-1">
          <label class="text-xs text-foreground-muted flex items-center gap-1">
            <Calendar class="w-3 h-3" />
            {t.from}
          </label>
          <div class="flex gap-1">
            <input
              type="date"
              bind:value={dateFrom}
              class="flex-1 px-2 py-1 text-xs bg-background-secondary border border-border rounded text-foreground"
            />
            <input
              type="time"
              bind:value={timeFrom}
              class="w-20 px-2 py-1 text-xs bg-background-secondary border border-border rounded text-foreground"
            />
          </div>
        </div>

        <!-- Date To -->
        <div class="space-y-1">
          <label class="text-xs text-foreground-muted flex items-center gap-1">
            <Calendar class="w-3 h-3" />
            {t.to}
          </label>
          <div class="flex gap-1">
            <input
              type="date"
              bind:value={dateTo}
              class="flex-1 px-2 py-1 text-xs bg-background-secondary border border-border rounded text-foreground"
            />
            <input
              type="time"
              bind:value={timeTo}
              class="w-20 px-2 py-1 text-xs bg-background-secondary border border-border rounded text-foreground"
            />
          </div>
        </div>

        <!-- Log Levels -->
        <div class="space-y-1">
          <span class="text-xs text-foreground-muted block">{t.levels}</span>
          <div
            class="flex flex-wrap gap-1"
            role="group"
            aria-label="Log level filters"
          >
            <button
              class="px-2 py-0.5 text-xs rounded border flex items-center gap-1 transition-colors
                     {selectedLevels.includes('error')
                ? 'bg-stopped/20 border-stopped text-stopped'
                : 'border-border text-foreground-muted hover:border-stopped'}"
              onclick={() => toggleLevel("error")}
            >
              <AlertCircle class="w-3 h-3" />
              {t.error}
            </button>
            <button
              class="px-2 py-0.5 text-xs rounded border flex items-center gap-1 transition-colors
                     {selectedLevels.includes('warning')
                ? 'bg-paused/20 border-paused text-paused'
                : 'border-border text-foreground-muted hover:border-paused'}"
              onclick={() => toggleLevel("warning")}
            >
              <AlertTriangle class="w-3 h-3" />
              {t.warning}
            </button>
            <button
              class="px-2 py-0.5 text-xs rounded border flex items-center gap-1 transition-colors
                     {selectedLevels.includes('info')
                ? 'bg-primary/20 border-primary text-primary'
                : 'border-border text-foreground-muted hover:border-primary'}"
              onclick={() => toggleLevel("info")}
            >
              <Info class="w-3 h-3" />
              {t.info}
            </button>
            <button
              class="px-2 py-0.5 text-xs rounded border flex items-center gap-1 transition-colors
                     {selectedLevels.includes('debug')
                ? 'bg-foreground/10 border-foreground-muted text-foreground'
                : 'border-border text-foreground-muted hover:border-foreground'}"
              onclick={() => toggleLevel("debug")}
            >
              <Bug class="w-3 h-3" />
              {t.debug}
            </button>
          </div>
        </div>

        <!-- Clear Filters -->
        <div class="flex items-end">
          <button
            class="px-3 py-1 text-xs bg-background-tertiary hover:bg-background-secondary border border-border rounded text-foreground-muted hover:text-foreground transition-colors"
            onclick={clearFilters}
          >
            {t.clearFilters}
          </button>
        </div>
      </div>
    {/if}
  </div>

  <!-- Logs -->
  <div
    bind:this={logsContainer}
    class="flex-1 overflow-auto p-4 bg-[#1a1b26] font-mono text-sm"
    onscroll={() => {
      // Disable auto-scroll if user scrolls up
      const { scrollTop, scrollHeight, clientHeight } = logsContainer;
      autoScroll = scrollHeight - scrollTop - clientHeight < 50;
    }}
  >
    {#if filteredLogs().length === 0}
      <div class="text-foreground-muted text-center py-8">
        {logs.length === 0
          ? t.waitingLogs
          : `${t.showing} 0 ${t.of} ${logs.length} ${t.lines}`}
      </div>
    {:else}
      {#each filteredLogs() as log, i}
        {@const LevelIcon = getLevelIcon(log.level)}
        <div
          class="log-line hover:bg-white/5 -mx-4 px-4 py-0.5 flex items-start gap-2 group"
        >
          <span
            class="text-foreground-muted select-none w-8 text-right shrink-0"
            >{i + 1}</span
          >
          {#if LevelIcon}
            <span class="shrink-0 mt-0.5 {getLevelColor(log.level)}">
              <LevelIcon class="w-3.5 h-3.5" />
            </span>
          {:else}
            <span class="w-3.5 shrink-0"></span>
          {/if}
          <span class="flex-1 {getLevelColor(log.level)}">
            <!-- Highlight search matches -->
            {#if searchQuery}
              {@html parseAnsi(log.line).replace(
                new RegExp(
                  `(${searchQuery.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")})`,
                  "gi",
                ),
                '<mark class="bg-primary/40 text-white rounded px-0.5">$1</mark>',
              )}
            {:else}
              {@html parseAnsi(log.line)}
            {/if}
          </span>
          {#if log.timestamp}
            <span
              class="text-foreground-muted text-xs opacity-0 group-hover:opacity-100 transition-opacity shrink-0"
            >
              {log.timestamp.toLocaleTimeString()}
            </span>
          {/if}
        </div>
      {/each}
    {/if}
  </div>

  <!-- Footer -->
  <div
    class="px-4 py-2 bg-background-tertiary border-t border-border flex items-center justify-between text-xs text-foreground-muted"
  >
    <span>
      {#if filteredLogs().length !== logs.length}
        {t.showing}
        <strong class="text-foreground">{filteredLogs().length}</strong>
        {t.of}
        {logs.length}
        {t.lines}
      {:else}
        {logs.length} {t.lines}
      {/if}
    </span>
    <label class="flex items-center gap-2 cursor-pointer">
      <input
        type="checkbox"
        bind:checked={autoScroll}
        class="w-3 h-3 accent-primary"
      />
      {t.autoScroll}
    </label>
  </div>
</div>

<style>
  .log-line {
    white-space: pre-wrap;
    word-break: break-all;
    line-height: 1.5;
  }

  /* Style for date/time inputs */
  input[type="date"]::-webkit-calendar-picker-indicator,
  input[type="time"]::-webkit-calendar-picker-indicator {
    filter: invert(0.8);
    cursor: pointer;
  }
</style>
