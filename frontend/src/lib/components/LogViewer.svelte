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
  } from "lucide-svelte";
  import type { Container } from "$lib/api/docker";
  import { API_BASE } from "$lib/api/docker";

  let { container, onClose }: { container: Container; onClose: () => void } =
    $props();

  let logs = $state<string[]>([]);
  let logsContainer: HTMLDivElement;
  let isFullscreen = $state(false);
  let isPaused = $state(false);
  let autoScroll = $state(true);
  let eventSource: EventSource | null = null;

  // Dragging state
  let isDragging = $state(false);
  let dragOffset = $state({ x: 0, y: 0 });
  let position = $state({ x: 0, y: 0 });
  let windowElement: HTMLDivElement;

  onMount(() => {
    startLogStream();
  });

  onDestroy(() => {
    eventSource?.close();
  });

  function startLogStream() {
    // Use SSE for streaming logs with authentication token
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
          logs = [...logs, line];

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
        logs = [...logs, event.data];
      }
    };

    eventSource.onerror = () => {
      logs = [...logs, "\x1b[31m[Connection lost - retrying...]\x1b[0m"];
    };
  }

  function clearLogs() {
    logs = [];
  }

  function togglePause() {
    isPaused = !isPaused;
  }

  function downloadLogs() {
    const content = logs.join("\n");
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
</script>

<div
  bind:this={windowElement}
  class="fixed z-50 bg-background-secondary border border-border rounded-lg shadow-2xl flex flex-col overflow-hidden"
  class:inset-4={isFullscreen}
  class:w-[700px]={!isFullscreen}
  class:h-[400px]={!isFullscreen}
  style={!isFullscreen && (position.x !== 0 || position.y !== 0)
    ? `left: ${position.x}px; top: ${position.y}px;`
    : !isFullscreen
      ? "top: 1rem; left: 1rem;"
      : ""}
>
  <!-- Header (Draggable) -->
  <div
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
    {#if logs.length === 0}
      <div class="text-foreground-muted text-center py-8">
        Waiting for logs...
      </div>
    {:else}
      {#each logs as line, i}
        <div class="log-line hover:bg-white/5 -mx-4 px-4 py-0.5">
          <span class="text-foreground-muted mr-3 select-none">{i + 1}</span>
          {@html parseAnsi(line)}
        </div>
      {/each}
    {/if}
  </div>

  <!-- Footer -->
  <div
    class="px-4 py-2 bg-background-tertiary border-t border-border flex items-center justify-between text-xs text-foreground-muted"
  >
    <span>{logs.length} líneas</span>
    <label class="flex items-center gap-2 cursor-pointer">
      <input
        type="checkbox"
        bind:checked={autoScroll}
        class="w-3 h-3 accent-primary"
      />
      Auto-scroll
    </label>
  </div>
</div>

<style>
  .log-line {
    white-space: pre-wrap;
    word-break: break-all;
    line-height: 1.5;
  }
</style>
