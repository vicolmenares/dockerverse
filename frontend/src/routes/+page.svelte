<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import {
    Box,
    Server,
    AlertTriangle,
    Wifi,
    WifiOff,
    Cpu,
    MemoryStick,
    ArrowUpDown,
    RotateCcw,
    BarChart3,
  } from "lucide-svelte";
  import {
    fetchContainers,
    fetchHosts,
    createEventStream,
    formatBytes,
    type Container,
    type Host,
    type ContainerStats,
  } from "$lib/api/docker";
  import {
    containers,
    containerStats,
    hosts,
    filteredContainers,
    language,
    translations,
    selectedHost,
  } from "$lib/stores/docker";
  import HostCard from "$lib/components/HostCard.svelte";
  import ContainerCard from "$lib/components/ContainerCard.svelte";

  // Lazy load heavy components (xterm is ~1MB)
  let Terminal: any = $state(null);
  let LogViewer: any = $state(null);

  let eventSource: EventSource | null = null;
  let terminalContainer = $state<Container | null>(null);
  let logsContainer = $state<Container | null>(null);
  let isLoading = $state(true);
  let connectionError = $state<string | null>(null);
  let viewMode = $state<"grid" | "table">("grid");
  let filterState = $state<"all" | "running" | "stopped">("all");
  let resourceMetric = $state<"cpu" | "memory" | "network" | "restarts">("cpu");
  let showResourceLeaderboard = $state(true);

  // Get current translations
  let t = $derived(translations[$language]);

  // Computed stats
  let totalContainers = $derived($containers.length);
  let runningContainers = $derived(
    $containers.filter((c) => c.state === "running").length,
  );
  let stoppedContainers = $derived(
    $containers.filter((c) => c.state !== "running").length,
  );
  let onlineHosts = $derived($hosts.filter((h) => h.online).length);

  // Filtered containers
  let displayContainers = $derived.by(() => {
    let result = $filteredContainers;
    if (filterState === "running") {
      result = result.filter((c) => c.state === "running");
    } else if (filterState === "stopped") {
      result = result.filter((c) => c.state !== "running");
    }
    return result;
  });

  // Top 14 containers by selected resource metric
  let topContainers = $derived.by(() => {
    const running = $containers.filter((c) => c.state === "running");
    const withStats = running
      .map((c) => ({
        container: c,
        stats: $containerStats.get(`${c.id}@${c.hostId}`),
      }))
      .filter((item) => item.stats);

    const sorted = [...withStats].sort((a, b) => {
      if (!a.stats || !b.stats) return 0;
      switch (resourceMetric) {
        case "cpu":
          return b.stats.cpuPercent - a.stats.cpuPercent;
        case "memory":
          return b.stats.memoryPercent - a.stats.memoryPercent;
        case "network":
          return (
            b.stats.networkRx +
            b.stats.networkTx -
            (a.stats.networkRx + a.stats.networkTx)
          );
        case "restarts":
          return 0; // We don't track restarts yet
        default:
          return 0;
      }
    });

    return sorted.slice(0, 14);
  });

  function getMetricValue(stats: ContainerStats | undefined): string {
    if (!stats) return "—";
    switch (resourceMetric) {
      case "cpu":
        return `${stats.cpuPercent.toFixed(1)}%`;
      case "memory":
        return `${stats.memoryPercent.toFixed(1)}%`;
      case "network":
        return `${formatBytes(stats.networkRx + stats.networkTx)}`;
      case "restarts":
        return "0";
      default:
        return "—";
    }
  }

  function getMetricPercent(stats: ContainerStats | undefined): number {
    if (!stats) return 0;
    switch (resourceMetric) {
      case "cpu":
        return Math.min(100, stats.cpuPercent);
      case "memory":
        return Math.min(100, stats.memoryPercent);
      case "network":
        return Math.min(100, (stats.networkRx + stats.networkTx) / 1048576); // rough scale
      default:
        return 0;
    }
  }

  function getMetricBarColor(percent: number): string {
    if (percent >= 80) return "bg-stopped";
    if (percent >= 50) return "bg-paused";
    return "bg-running";
  }

  // Preload Terminal/LogViewer on hover
  function preloadComponents() {
    if (!Terminal) {
      import("$lib/components/Terminal.svelte").then(
        (m) => (Terminal = m.default),
      );
    }
    if (!LogViewer) {
      import("$lib/components/LogViewer.svelte").then(
        (m) => (LogViewer = m.default),
      );
    }
  }

  function handleRefresh() {
    loadData();
  }

  onMount(async () => {
    await loadData();
    startStatsStream();

    // Listen for refresh events from header
    window.addEventListener("dockerverse:refresh", handleRefresh);

    // Refresh every 30 seconds
    const interval = setInterval(loadData, 30000);
    return () => {
      clearInterval(interval);
      window.removeEventListener("dockerverse:refresh", handleRefresh);
    };
  });

  onDestroy(() => {
    eventSource?.close();
  });

  async function loadData() {
    try {
      const [containersData, hostsData] = await Promise.all([
        fetchContainers(),
        fetchHosts(),
      ]);
      containers.set(containersData);
      hosts.set(hostsData);
      connectionError = null;
    } catch (e) {
      connectionError = "Error connecting to backend";
      console.error(e);
    } finally {
      isLoading = false;
    }
  }

  function startStatsStream() {
    eventSource = createEventStream({
      onStats: (statsArray: ContainerStats[]) => {
        containerStats.update((current) => {
          for (const stat of statsArray) {
            current.set(`${stat.id}@${stat.hostId}`, stat);
          }
          return current;
        });
      },
      onContainers: (containersData: Container[]) => {
        containers.set(containersData);
      },
      onHosts: (hostsData: Host[]) => {
        hosts.set(hostsData);
      },
    });
  }

  function getStats(container: Container): ContainerStats | undefined {
    return $containerStats.get(`${container.id}@${container.hostId}`);
  }

  async function openTerminal(container: Container) {
    if (!Terminal) {
      Terminal = (await import("$lib/components/Terminal.svelte")).default;
    }
    terminalContainer = container;
  }

  async function openLogs(container: Container) {
    if (!LogViewer) {
      LogViewer = (await import("$lib/components/LogViewer.svelte")).default;
    }
    logsContainer = container;
  }
</script>

<svelte:head>
  <title>DockerVerse - Dashboard</title>
</svelte:head>

<div class="min-h-screen bg-background">
  <!-- Connection Error Banner -->
  {#if connectionError}
    <div
      class="bg-accent-red/10 border-b border-accent-red/30 px-4 py-3 flex items-center justify-center gap-2 text-accent-red"
    >
      <AlertTriangle class="w-4 h-4" />
      {connectionError}
    </div>
  {/if}

  <main class="container mx-auto px-4 py-6 max-w-7xl">
    <!-- Stats Overview -->
    <div class="grid grid-cols-2 md:grid-cols-4 gap-4 mb-8">
      <div class="card p-4 flex items-center gap-4">
        <div class="p-3 bg-primary/10 rounded-xl">
          <Server class="w-6 h-6 text-primary" />
        </div>
        <div>
          <p class="metric-label">{t.hosts}</p>
          <p class="metric-value text-foreground">
            {onlineHosts}<span class="text-foreground-muted"
              >/{$hosts.length}</span
            >
          </p>
        </div>
      </div>

      <div class="card p-4 flex items-center gap-4">
        <div class="p-3 bg-accent-cyan/10 rounded-xl">
          <Box class="w-6 h-6 text-accent-cyan" />
        </div>
        <div>
          <p class="metric-label">{t.total}</p>
          <p class="metric-value text-accent-cyan">{totalContainers}</p>
        </div>
      </div>

      <div class="card p-4 flex items-center gap-4">
        <div class="p-3 bg-running/10 rounded-xl">
          <Wifi class="w-6 h-6 text-running" />
        </div>
        <div>
          <p class="metric-label">{t.running}</p>
          <p class="metric-value text-running">{runningContainers}</p>
        </div>
      </div>

      <div class="card p-4 flex items-center gap-4">
        <div class="p-3 bg-stopped/10 rounded-xl">
          <WifiOff class="w-6 h-6 text-stopped" />
        </div>
        <div>
          <p class="metric-label">{t.stopped}</p>
          <p class="metric-value text-stopped">{stoppedContainers}</p>
        </div>
      </div>
    </div>

    <!-- Hosts Section -->
    <section class="mb-8">
      <h2
        class="text-lg font-semibold text-foreground mb-4 flex items-center gap-2"
      >
        <Server class="w-5 h-5 text-primary" />
        {t.hosts}
        {#if $selectedHost}
          <span
            class="text-xs font-normal text-primary bg-primary/10 px-2 py-0.5 rounded"
          >
            {t.filterByHost}: {$selectedHost}
          </span>
        {/if}
      </h2>

      {#if isLoading}
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          {#each [1, 2] as _}
            <div class="card p-5 animate-pulse">
              <div class="h-4 bg-background-tertiary rounded w-1/3 mb-4"></div>
              <div class="h-8 bg-background-tertiary rounded w-full"></div>
            </div>
          {/each}
        </div>
      {:else}
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          {#each $hosts as host}
            <HostCard {host} />
          {/each}
        </div>
      {/if}
    </section>

    <!-- Resource Leaderboard Section -->
    {#if !isLoading && runningContainers > 0}
      <section class="mb-8">
        <div class="flex items-center justify-between mb-4">
          <h2
            class="text-lg font-semibold text-foreground flex items-center gap-2"
          >
            <BarChart3 class="w-5 h-5 text-accent-purple" />
            {$language === "es" ? "Top Recursos" : "Top Resources"}
            <span class="text-sm font-normal text-foreground-muted"
              >(Top {Math.min(14, topContainers.length)})</span
            >
          </h2>
          <button
            class="text-sm text-foreground-muted hover:text-foreground transition-colors"
            onclick={() => (showResourceLeaderboard = !showResourceLeaderboard)}
          >
            {showResourceLeaderboard
              ? $language === "es"
                ? "Ocultar"
                : "Hide"
              : $language === "es"
                ? "Mostrar"
                : "Show"}
          </button>
        </div>

        {#if showResourceLeaderboard}
          <!-- Metric tabs -->
          <div
            class="flex gap-1 mb-4 p-1 bg-background-secondary rounded-lg border border-border w-fit"
          >
            <button
              class="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm font-medium transition-all {resourceMetric ===
              'cpu'
                ? 'bg-primary text-white shadow-sm'
                : 'text-foreground-muted hover:text-foreground'}"
              onclick={() => (resourceMetric = "cpu")}
            >
              <Cpu class="w-3.5 h-3.5" />
              CPU
            </button>
            <button
              class="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm font-medium transition-all {resourceMetric ===
              'memory'
                ? 'bg-accent-cyan text-white shadow-sm'
                : 'text-foreground-muted hover:text-foreground'}"
              onclick={() => (resourceMetric = "memory")}
            >
              <MemoryStick class="w-3.5 h-3.5" />
              {$language === "es" ? "Memoria" : "Memory"}
            </button>
            <button
              class="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm font-medium transition-all {resourceMetric ===
              'network'
                ? 'bg-accent-purple text-white shadow-sm'
                : 'text-foreground-muted hover:text-foreground'}"
              onclick={() => (resourceMetric = "network")}
            >
              <ArrowUpDown class="w-3.5 h-3.5" />
              {$language === "es" ? "Red" : "Network"}
            </button>
            <button
              class="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm font-medium transition-all {resourceMetric ===
              'restarts'
                ? 'bg-paused text-white shadow-sm'
                : 'text-foreground-muted hover:text-foreground'}"
              onclick={() => (resourceMetric = "restarts")}
            >
              <RotateCcw class="w-3.5 h-3.5" />
              {$language === "es" ? "Reinicios" : "Restarts"}
            </button>
          </div>

          <!-- Leaderboard table -->
          <div class="card overflow-hidden">
            <!-- Table header -->
            <div
              class="grid grid-cols-[auto_1fr_120px_200px] gap-4 px-4 py-2 border-b border-border text-xs font-medium text-foreground-muted uppercase tracking-wider bg-background-tertiary/30"
            >
              <span class="w-8 text-center">#</span>
              <span>{$language === "es" ? "Contenedor" : "Container"}</span>
              <span class="text-right"
                >{$language === "es" ? "Valor" : "Value"}</span
              >
              <span>{$language === "es" ? "Uso" : "Usage"}</span>
            </div>

            {#if topContainers.length === 0}
              <div class="px-4 py-8 text-center text-foreground-muted text-sm">
                {$language === "es"
                  ? "No hay datos de recursos disponibles"
                  : "No resource data available"}
              </div>
            {:else}
              {#each topContainers as item, i}
                {@const percent = getMetricPercent(item.stats)}
                <div
                  class="grid grid-cols-[auto_1fr_120px_200px] gap-4 px-4 py-2.5 items-center border-b border-border/50 hover:bg-background-tertiary/20 transition-colors"
                >
                  <span
                    class="w-8 text-center text-sm font-bold {i < 3
                      ? 'text-primary'
                      : 'text-foreground-muted'}">{i + 1}</span
                  >
                  <div class="min-w-0">
                    <p class="text-sm font-medium text-foreground truncate">
                      {item.container.name}
                    </p>
                    <p class="text-xs text-foreground-muted truncate">
                      {item.container.hostId}
                    </p>
                  </div>
                  <span
                    class="text-sm font-mono text-foreground text-right tabular-nums"
                    >{getMetricValue(item.stats)}</span
                  >
                  <div class="flex items-center gap-2">
                    <div
                      class="flex-1 h-2 bg-background-tertiary rounded-full overflow-hidden"
                    >
                      <div
                        class="h-full rounded-full transition-all duration-500 {getMetricBarColor(
                          percent,
                        )}"
                        style="width: {Math.max(2, percent)}%"
                      ></div>
                    </div>
                  </div>
                </div>
              {/each}
            {/if}
          </div>
        {/if}
      </section>
    {/if}

    <!-- Containers Section -->
    <section>
      <div class="flex items-center justify-between mb-4">
        <h2
          class="text-lg font-semibold text-foreground flex items-center gap-2"
        >
          <Box class="w-5 h-5 text-accent-cyan" />
          {t.containers}
          <span class="text-sm font-normal text-foreground-muted"
            >({displayContainers.length})</span
          >
        </h2>

        <div class="flex items-center gap-2">
          <!-- Filter buttons -->
          <div class="flex rounded-lg overflow-hidden border border-border">
            <button
              class="px-3 py-1.5 text-sm transition-colors {filterState ===
              'all'
                ? 'bg-primary text-white'
                : 'bg-background-secondary text-foreground-muted hover:text-foreground'}"
              onclick={() => (filterState = "all")}
            >
              {t.all}
            </button>
            <button
              class="px-3 py-1.5 text-sm transition-colors {filterState ===
              'running'
                ? 'bg-running text-white'
                : 'bg-background-secondary text-foreground-muted hover:text-foreground'}"
              onclick={() => (filterState = "running")}
            >
              {t.running}
            </button>
            <button
              class="px-3 py-1.5 text-sm transition-colors {filterState ===
              'stopped'
                ? 'bg-stopped text-white'
                : 'bg-background-secondary text-foreground-muted hover:text-foreground'}"
              onclick={() => (filterState = "stopped")}
            >
              {t.stopped}
            </button>
          </div>
        </div>
      </div>

      {#if isLoading}
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {#each [1, 2, 3, 4, 5, 6] as _}
            <div class="card p-4 animate-pulse">
              <div class="h-4 bg-background-tertiary rounded w-2/3 mb-3"></div>
              <div class="h-3 bg-background-tertiary rounded w-1/2 mb-4"></div>
              <div class="h-12 bg-background-tertiary rounded w-full"></div>
            </div>
          {/each}
        </div>
      {:else if displayContainers.length === 0}
        <div class="card p-12 text-center">
          <Box class="w-12 h-12 text-foreground-muted mx-auto mb-4" />
          <p class="text-foreground-muted">{t.noContainers}</p>
        </div>
      {:else}
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {#each displayContainers as container (container.id)}
            <!-- svelte-ignore a11y_no_static_element_interactions -->
            <div onmouseenter={preloadComponents}>
              <ContainerCard
                {container}
                stats={getStats(container)}
                onTerminal={() => openTerminal(container)}
                onLogs={() => openLogs(container)}
              />
            </div>
          {/each}
        </div>
      {/if}
    </section>
  </main>

  <!-- Terminal Modal (Lazy Loaded) -->
  {#if terminalContainer && Terminal}
    {@const TerminalComponent = Terminal}
    <TerminalComponent
      container={terminalContainer}
      onClose={() => (terminalContainer = null)}
    />
  {/if}

  <!-- Logs Modal (Lazy Loaded) -->
  {#if logsContainer && LogViewer}
    {@const LogViewerComponent = LogViewer}
    <LogViewerComponent
      container={logsContainer}
      onClose={() => (logsContainer = null)}
    />
  {/if}
</div>
