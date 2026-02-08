<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import {
    Cpu,
    MemoryStick,
    Wifi,
    HardDrive,
    Activity,
    TrendingUp,
    TrendingDown,
    Minus,
    Clock,
  } from "lucide-svelte";
  import type { Host } from "$lib/api/docker";
  import { language } from "$lib/stores/docker";

  let { host }: { host: Host } = $props();

  // Historical data for charts (last 60 data points = 1 minute at 1s intervals)
  const MAX_HISTORY = 60;
  let cpuHistory = $state<number[]>([]);
  let memHistory = $state<number[]>([]);
  let netInHistory = $state<number[]>([]);
  let netOutHistory = $state<number[]>([]);
  let diskReadHistory = $state<number[]>([]);
  let diskWriteHistory = $state<number[]>([]);

  // Current values (simulated network/disk for demo)
  let netIn = $state(0);
  let netOut = $state(0);
  let diskRead = $state(0);
  let diskWrite = $state(0);

  // Refresh interval
  let refreshInterval: ReturnType<typeof setInterval> | null = null;

  // Translations
  const translations = {
    en: {
      resourceMonitor: "Resource Monitor",
      cpu: "CPU Usage",
      memory: "Memory Usage",
      network: "Network I/O",
      disk: "Disk I/O",
      current: "Current",
      average: "Avg",
      peak: "Peak",
      in: "In",
      out: "Out",
      read: "Read",
      write: "Write",
      last60s: "Last 60s",
      mbps: "MB/s",
      percent: "%",
    },
    es: {
      resourceMonitor: "Monitor de Recursos",
      cpu: "Uso de CPU",
      memory: "Uso de Memoria",
      network: "Red I/O",
      disk: "Disco I/O",
      current: "Actual",
      average: "Prom",
      peak: "Pico",
      in: "Entrada",
      out: "Salida",
      read: "Lectura",
      write: "Escritura",
      last60s: "Últimos 60s",
      mbps: "MB/s",
      percent: "%",
    },
  };

  let t = $derived(translations[$language] || translations.en);

  // Calculate stats
  function getStats(history: number[]) {
    if (history.length === 0) return { current: 0, avg: 0, peak: 0 };
    const current = history[history.length - 1] || 0;
    const avg = history.reduce((a, b) => a + b, 0) / history.length;
    const peak = Math.max(...history);
    return { current, avg, peak };
  }

  let cpuStats = $derived(getStats(cpuHistory));
  let memStats = $derived(getStats(memHistory));
  let netInStats = $derived(getStats(netInHistory));
  let netOutStats = $derived(getStats(netOutHistory));

  // Get trend icon
  function getTrend(history: number[]): "up" | "down" | "stable" {
    if (history.length < 5) return "stable";
    const recent = history.slice(-5);
    const older = history.slice(-10, -5);
    if (older.length === 0) return "stable";

    const recentAvg = recent.reduce((a, b) => a + b, 0) / recent.length;
    const olderAvg = older.reduce((a, b) => a + b, 0) / older.length;

    const diff = recentAvg - olderAvg;
    if (diff > 5) return "up";
    if (diff < -5) return "down";
    return "stable";
  }

  // Generate SVG path for sparkline
  function getSparklinePath(
    data: number[],
    width: number,
    height: number,
    maxValue: number = 100,
  ): string {
    if (data.length < 2) return "";

    const padding = 2;
    const effectiveWidth = width - padding * 2;
    const effectiveHeight = height - padding * 2;

    const points = data.map((value, i) => {
      const x = padding + (i / (data.length - 1)) * effectiveWidth;
      const y =
        padding + effectiveHeight - (value / maxValue) * effectiveHeight;
      return `${x},${y}`;
    });

    return `M ${points.join(" L ")}`;
  }

  // Generate area path for filled chart
  function getAreaPath(
    data: number[],
    width: number,
    height: number,
    maxValue: number = 100,
  ): string {
    if (data.length < 2) return "";

    const padding = 2;
    const effectiveWidth = width - padding * 2;
    const effectiveHeight = height - padding * 2;

    const points = data.map((value, i) => {
      const x = padding + (i / (data.length - 1)) * effectiveWidth;
      const y =
        padding + effectiveHeight - (value / maxValue) * effectiveHeight;
      return `${x},${y}`;
    });

    return `M ${padding},${padding + effectiveHeight} L ${points.join(" L ")} L ${padding + effectiveWidth},${padding + effectiveHeight} Z`;
  }

  // Color based on value
  function getValueColor(value: number): string {
    if (value >= 80) return "text-stopped";
    if (value >= 50) return "text-paused";
    return "text-running";
  }

  function getChartColor(value: number): string {
    if (value >= 80) return "#f7768e";
    if (value >= 50) return "#e0af68";
    return "#9ece6a";
  }

  // Update data
  function updateData() {
    // Add current values to history
    cpuHistory = [...cpuHistory.slice(-(MAX_HISTORY - 1)), host.cpuPercent];
    memHistory = [...memHistory.slice(-(MAX_HISTORY - 1)), host.memoryPercent];

    // Simulate network/disk I/O (in real implementation, this would come from the backend)
    const baseNet = Math.random() * 5;
    netIn = baseNet + Math.random() * 2;
    netOut = baseNet * 0.3 + Math.random() * 1;
    netInHistory = [...netInHistory.slice(-(MAX_HISTORY - 1)), netIn];
    netOutHistory = [...netOutHistory.slice(-(MAX_HISTORY - 1)), netOut];

    diskRead = Math.random() * 10;
    diskWrite = Math.random() * 5;
    diskReadHistory = [...diskReadHistory.slice(-(MAX_HISTORY - 1)), diskRead];
    diskWriteHistory = [
      ...diskWriteHistory.slice(-(MAX_HISTORY - 1)),
      diskWrite,
    ];
  }

  onMount(() => {
    // Initialize with current values
    updateData();

    // Update every second
    refreshInterval = setInterval(updateData, 1000);
  });

  onDestroy(() => {
    if (refreshInterval) {
      clearInterval(refreshInterval);
    }
  });
</script>

<div
  class="mt-4 pt-4 border-t border-border space-y-4 animate-in fade-in slide-in-from-top-2 duration-300"
>
  <div class="flex items-center gap-2 text-foreground-muted">
    <Activity class="w-4 h-4" />
    <span class="text-xs font-medium uppercase tracking-wider"
      >{t.resourceMonitor}</span
    >
    <span class="text-xs">({t.last60s})</span>
  </div>

  <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
    <!-- CPU Chart -->
    <div class="bg-background-tertiary/30 rounded-lg p-3">
      <div class="flex items-center justify-between mb-2">
        <div class="flex items-center gap-2">
          <Cpu class="w-4 h-4 text-primary" />
          <span class="text-sm font-medium text-foreground">{t.cpu}</span>
        </div>
        <div class="flex items-center gap-1">
          {#if getTrend(cpuHistory) === "up"}
            <TrendingUp class="w-3 h-3 text-stopped" />
          {:else if getTrend(cpuHistory) === "down"}
            <TrendingDown class="w-3 h-3 text-running" />
          {:else}
            <Minus class="w-3 h-3 text-foreground-muted" />
          {/if}
          <span class="text-lg font-bold {getValueColor(cpuStats.current)}"
            >{cpuStats.current.toFixed(1)}%</span
          >
        </div>
      </div>

      <!-- Sparkline -->
      <div class="h-16 relative">
        <svg
          class="w-full h-full"
          viewBox="0 0 200 60"
          preserveAspectRatio="none"
        >
          <!-- Grid lines -->
          <line
            x1="0"
            y1="15"
            x2="200"
            y2="15"
            stroke="currentColor"
            class="text-border"
            stroke-width="0.5"
            stroke-dasharray="2,2"
          />
          <line
            x1="0"
            y1="30"
            x2="200"
            y2="30"
            stroke="currentColor"
            class="text-border"
            stroke-width="0.5"
            stroke-dasharray="2,2"
          />
          <line
            x1="0"
            y1="45"
            x2="200"
            y2="45"
            stroke="currentColor"
            class="text-border"
            stroke-width="0.5"
            stroke-dasharray="2,2"
          />

          <!-- Area fill -->
          <path
            d={getAreaPath(cpuHistory, 200, 60)}
            fill={getChartColor(cpuStats.current)}
            fill-opacity="0.2"
          />

          <!-- Line -->
          <path
            d={getSparklinePath(cpuHistory, 200, 60)}
            fill="none"
            stroke={getChartColor(cpuStats.current)}
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </svg>

        <!-- Y-axis labels -->
        <div
          class="absolute left-0 top-0 h-full flex flex-col justify-between text-[10px] text-foreground-muted pointer-events-none"
        >
          <span>100%</span>
          <span>50%</span>
          <span>0%</span>
        </div>
      </div>

      <!-- Stats -->
      <div class="flex justify-between mt-2 text-xs text-foreground-muted">
        <span
          >{t.average}:
          <strong class="text-foreground">{cpuStats.avg.toFixed(1)}%</strong
          ></span
        >
        <span
          >{t.peak}:
          <strong class="text-foreground">{cpuStats.peak.toFixed(1)}%</strong
          ></span
        >
      </div>
    </div>

    <!-- Memory Chart -->
    <div class="bg-background-tertiary/30 rounded-lg p-3">
      <div class="flex items-center justify-between mb-2">
        <div class="flex items-center gap-2">
          <MemoryStick class="w-4 h-4 text-accent-cyan" />
          <span class="text-sm font-medium text-foreground">{t.memory}</span>
        </div>
        <div class="flex items-center gap-1">
          {#if getTrend(memHistory) === "up"}
            <TrendingUp class="w-3 h-3 text-stopped" />
          {:else if getTrend(memHistory) === "down"}
            <TrendingDown class="w-3 h-3 text-running" />
          {:else}
            <Minus class="w-3 h-3 text-foreground-muted" />
          {/if}
          <span class="text-lg font-bold {getValueColor(memStats.current)}"
            >{memStats.current.toFixed(1)}%</span
          >
        </div>
      </div>

      <!-- Sparkline -->
      <div class="h-16 relative">
        <svg
          class="w-full h-full"
          viewBox="0 0 200 60"
          preserveAspectRatio="none"
        >
          <!-- Grid lines -->
          <line
            x1="0"
            y1="15"
            x2="200"
            y2="15"
            stroke="currentColor"
            class="text-border"
            stroke-width="0.5"
            stroke-dasharray="2,2"
          />
          <line
            x1="0"
            y1="30"
            x2="200"
            y2="30"
            stroke="currentColor"
            class="text-border"
            stroke-width="0.5"
            stroke-dasharray="2,2"
          />
          <line
            x1="0"
            y1="45"
            x2="200"
            y2="45"
            stroke="currentColor"
            class="text-border"
            stroke-width="0.5"
            stroke-dasharray="2,2"
          />

          <!-- Area fill -->
          <path
            d={getAreaPath(memHistory, 200, 60)}
            fill="#7dcfff"
            fill-opacity="0.2"
          />

          <!-- Line -->
          <path
            d={getSparklinePath(memHistory, 200, 60)}
            fill="none"
            stroke="#7dcfff"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </svg>

        <!-- Y-axis labels -->
        <div
          class="absolute left-0 top-0 h-full flex flex-col justify-between text-[10px] text-foreground-muted pointer-events-none"
        >
          <span>100%</span>
          <span>50%</span>
          <span>0%</span>
        </div>
      </div>

      <!-- Stats -->
      <div class="flex justify-between mt-2 text-xs text-foreground-muted">
        <span
          >{t.average}:
          <strong class="text-foreground">{memStats.avg.toFixed(1)}%</strong
          ></span
        >
        <span
          >{t.peak}:
          <strong class="text-foreground">{memStats.peak.toFixed(1)}%</strong
          ></span
        >
      </div>
    </div>

    <!-- Network I/O -->
    <div class="bg-background-tertiary/30 rounded-lg p-3">
      <div class="flex items-center justify-between mb-2">
        <div class="flex items-center gap-2">
          <Wifi class="w-4 h-4 text-accent-purple" />
          <span class="text-sm font-medium text-foreground">{t.network}</span>
        </div>
        <div class="flex items-center gap-2 text-xs">
          <span class="text-running">↓ {netIn.toFixed(2)} MB/s</span>
          <span class="text-primary">↑ {netOut.toFixed(2)} MB/s</span>
        </div>
      </div>

      <!-- Dual sparkline -->
      <div class="h-16 relative">
        <svg
          class="w-full h-full"
          viewBox="0 0 200 60"
          preserveAspectRatio="none"
        >
          <!-- Grid lines -->
          <line
            x1="0"
            y1="30"
            x2="200"
            y2="30"
            stroke="currentColor"
            class="text-border"
            stroke-width="0.5"
            stroke-dasharray="2,2"
          />

          <!-- In (download) -->
          <path
            d={getAreaPath(netInHistory, 200, 60, 10)}
            fill="#9ece6a"
            fill-opacity="0.2"
          />
          <path
            d={getSparklinePath(netInHistory, 200, 60, 10)}
            fill="none"
            stroke="#9ece6a"
            stroke-width="1.5"
            stroke-linecap="round"
          />

          <!-- Out (upload) -->
          <path
            d={getAreaPath(netOutHistory, 200, 60, 10)}
            fill="#7aa2f7"
            fill-opacity="0.2"
          />
          <path
            d={getSparklinePath(netOutHistory, 200, 60, 10)}
            fill="none"
            stroke="#7aa2f7"
            stroke-width="1.5"
            stroke-linecap="round"
          />
        </svg>
      </div>

      <!-- Legend -->
      <div class="flex justify-center gap-4 mt-2 text-xs text-foreground-muted">
        <span class="flex items-center gap-1"
          ><span class="w-2 h-2 rounded bg-running"></span> {t.in}</span
        >
        <span class="flex items-center gap-1"
          ><span class="w-2 h-2 rounded bg-primary"></span> {t.out}</span
        >
      </div>
    </div>

    <!-- Disk I/O -->
    <div class="bg-background-tertiary/30 rounded-lg p-3">
      <div class="flex items-center justify-between mb-2">
        <div class="flex items-center gap-2">
          <HardDrive class="w-4 h-4 text-paused" />
          <span class="text-sm font-medium text-foreground">{t.disk}</span>
        </div>
        <div class="flex items-center gap-2 text-xs">
          <span class="text-accent-cyan">R: {diskRead.toFixed(2)} MB/s</span>
          <span class="text-paused">W: {diskWrite.toFixed(2)} MB/s</span>
        </div>
      </div>

      <!-- Dual sparkline -->
      <div class="h-16 relative">
        <svg
          class="w-full h-full"
          viewBox="0 0 200 60"
          preserveAspectRatio="none"
        >
          <!-- Grid lines -->
          <line
            x1="0"
            y1="30"
            x2="200"
            y2="30"
            stroke="currentColor"
            class="text-border"
            stroke-width="0.5"
            stroke-dasharray="2,2"
          />

          <!-- Read -->
          <path
            d={getAreaPath(diskReadHistory, 200, 60, 15)}
            fill="#7dcfff"
            fill-opacity="0.2"
          />
          <path
            d={getSparklinePath(diskReadHistory, 200, 60, 15)}
            fill="none"
            stroke="#7dcfff"
            stroke-width="1.5"
            stroke-linecap="round"
          />

          <!-- Write -->
          <path
            d={getAreaPath(diskWriteHistory, 200, 60, 15)}
            fill="#e0af68"
            fill-opacity="0.2"
          />
          <path
            d={getSparklinePath(diskWriteHistory, 200, 60, 15)}
            fill="none"
            stroke="#e0af68"
            stroke-width="1.5"
            stroke-linecap="round"
          />
        </svg>
      </div>

      <!-- Legend -->
      <div class="flex justify-center gap-4 mt-2 text-xs text-foreground-muted">
        <span class="flex items-center gap-1"
          ><span class="w-2 h-2 rounded bg-accent-cyan"></span> {t.read}</span
        >
        <span class="flex items-center gap-1"
          ><span class="w-2 h-2 rounded bg-paused"></span> {t.write}</span
        >
      </div>
    </div>
  </div>
</div>
