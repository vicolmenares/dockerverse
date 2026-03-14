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
    ArrowUpCircle,
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
    checkForUpdates,
    pendingUpdatesCount,
    imageUpdates,
  } from "$lib/stores/docker";
  import HostCard from "$lib/components/HostCard.svelte";
  import ContainerCard from "$lib/components/ContainerCard.svelte";
  import ResourceChart from "$lib/components/ResourceChart.svelte";
  import ContainerActivityChart from '$lib/components/ContainerActivityChart.svelte';
  import HostFiles from "$lib/components/HostFiles.svelte";

  // Lazy load heavy components (xterm is ~1MB)
  let Terminal: any = $state(null);
  let LogViewer: any = $state(null);

  let eventSource: EventSource | null = null;
  let terminalContainer = $state<Container | null>(null);
  let logsContainer = $state<Container | null>(null);
  let hostTerminal = $state<Host | null>(null);
  let hostFiles = $state<Host | null>(null);
  let isLoading = $state(true);
  let connectionError = $state<string | null>(null);
  let viewMode = $state<"grid" | "table">("grid");
  let filterState = $state<"all" | "running" | "stopped" | "updates">(
    (typeof localStorage !== "undefined"
      ? localStorage.getItem("dockerverse_filterState") as any
      : null) || "all"
  );
  let resourceMetric = $state<"cpu" | "memory" | "network" | "restarts">("cpu");
  let showResourceLeaderboard = $state(true);
  let expandedHostId = $state<string | null>(null);
  let pageSize = $state(
    typeof localStorage !== "undefined"
      ? parseInt(localStorage.getItem("dockerverse_pageSize") || "12", 10)
      : 12,
  );
  let currentPage = $state(1);
  let topN = $state(
    typeof localStorage !== "undefined"
      ? parseInt(localStorage.getItem("dockerverse_topN") || "10", 10)
      : 10,
  );
  $effect(() => {
    localStorage.setItem("dockerverse_topN", String(topN));
  });
  $effect(() => {
    localStorage.setItem("dockerverse_pageSize", String(pageSize));
  });
  const topNOptions = [5, 10, 15, 20, 30];
  const pageSizeOptions = [9, 12, 18, 24];

  $effect(() => {
    localStorage.setItem("dockerverse_filterState", filterState);
  });

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
  let expandedHost = $derived(
    expandedHostId
      ? $hosts.find((h) => h.id === expandedHostId) || null
      : null,
  );

  function toggleFilter(next: "all" | "running" | "stopped" | "updates") {
    filterState = filterState === next ? "all" : next;
  }

  function toggleHostResources(hostId: string) {
    expandedHostId = expandedHostId === hostId ? null : hostId;
  }

  $effect(() => {
    filterState;
    $selectedHost;
    currentPage = 1;
  });

  // Filtered containers
  let displayContainers = $derived.by(() => {
    let result = $filteredContainers;
    if (filterState === "running") {
      result = result.filter((c) => c.state === "running");
    } else if (filterState === "stopped") {
      result = result.filter((c) => c.state !== "running");
    } else if (filterState === "updates") {
      const updateIds = new Set(
        $imageUpdates.filter((u) => u.hasUpdate).map((u) => u.containerId),
      );
      result = result.filter((c) => updateIds.has(c.id));
    }
    return result;
  });

  let totalPages = $derived(
    Math.max(1, Math.ceil(displayContainers.length / pageSize)),
  );
  $effect(() => {
    if (currentPage > totalPages) currentPage = totalPages;
  });
  let pagedContainers = $derived.by(() => {
    const start = (currentPage - 1) * pageSize;
    return displayContainers.slice(start, start + pageSize);
  });
  let pageNumbers = $derived.by(() => {
    const maxButtons = 5;
    const total = totalPages;
    if (total <= maxButtons) {
      return Array.from({ length: total }, (_, i) => i + 1);
    }
    const half = Math.floor(maxButtons / 2);
    let start = Math.max(1, currentPage - half);
    let end = Math.min(total, start + maxButtons - 1);
    if (end - start + 1 < maxButtons) {
      start = Math.max(1, end - maxButtons + 1);
    }
    return Array.from({ length: end - start + 1 }, (_, i) => start + i);
  });

  function goToPage(page: number) {
    const clamped = Math.min(Math.max(1, page), totalPages);
    currentPage = clamped;
  }

  // Top 14 containers by selected resource metric
  let topContainers = $derived.by(() => {
    const running = $filteredContainers.filter((c) => c.state === "running");
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

    return sorted.slice(0, topN);
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
        return (stats.networkRx + stats.networkTx) / 1048576; // MB, no cap - relative scaling handles proportions
      default:
        return 0;
    }
  }

  function getMetricBarColor(percent: number): string {
    if (resourceMetric === "network") return "bg-accent-purple";
    if (percent >= 80) return "bg-stopped";
    if (percent >= 50) return "bg-paused";
    return "bg-running";
  }

  // Get max metric value for bar chart scaling
  let topMaxValue = $derived.by(() => {
    if (topContainers.length === 0) return 1;
    const values = topContainers.map((item) => {
      const p = getMetricPercent(item.stats);
      return p;
    });
    return Math.max(...values, 1);
  });

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

  onMount(() => {
    let active = true;

    void (async () => {
      await loadData();
      if (!active) return;
      startStatsStream();

      // Check for image updates after initial load
      checkForUpdates();
    })();

    // Listen for refresh events from header
    window.addEventListener("dockerverse:refresh", handleRefresh);

    // Refresh data every 30 seconds, check updates every 15 minutes
    const dataInterval = setInterval(loadData, 30000);
    const updateInterval = setInterval(checkForUpdates, 15 * 60 * 1000);
    return () => {
      active = false;
      clearInterval(dataInterval);
      clearInterval(updateInterval);
      window.removeEventListener("dockerverse:refresh", handleRefresh);
    };
  });

  onDestroy(() => {
    eventSource?.close();
  });

  async function loadData() {
    try {
      const results = await Promise.allSettled([
        fetchContainers(),
        fetchHosts(),
      ]);

      if (results[0].status === "fulfilled") {
        containers.set(results[0].value);
      }
      if (results[1].status === "fulfilled") {
        hosts.set(results[1].value);
      }

      const allFailed = results.every((r) => r.status === "rejected");
      connectionError = allFailed ? "Error connecting to backend" : null;
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
          return new Map(current);
        });
        connectionError = null;
      },
      onContainers: (containersData: Container[]) => {
        containers.set(containersData);
        connectionError = null;
      },
      onHosts: (hostsData: Host[]) => {
        hosts.set(hostsData);
        connectionError = null;
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

  async function openHostTerminal(host: Host) {
    if (!Terminal) {
      Terminal = (await import("$lib/components/Terminal.svelte")).default;
    }
    hostTerminal = host;
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

<div class="min-h-screen">
  <!-- Connection Error Banner -->
  {#if connectionError}
    <div class="border-b border-red-500/30 bg-red-500/10 px-6 py-3 flex items-center gap-2 text-red-400 text-sm">
      <AlertTriangle class="w-4 h-4 shrink-0" />
      {connectionError}
    </div>
  {/if}

  <!-- Stats Bar -->
  <div class="border-b border-zinc-800 bg-zinc-950 px-6 py-3 flex items-center overflow-x-auto">
    <div class="flex items-center gap-3 pr-5">
      <Server class="w-4 h-4 text-zinc-500 shrink-0" />
      <span class="text-zinc-500 uppercase tracking-widest text-[10px] font-semibold">{t.hosts}</span>
      <span class="font-mono text-zinc-200">{onlineHosts}<span class="text-zinc-600">/{$hosts.length}</span></span>
    </div>
    <div class="w-px h-5 bg-zinc-800 shrink-0"></div>
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div
      class="flex items-center gap-3 px-5 cursor-pointer hover:bg-zinc-900 transition-colors"
      onclick={() => toggleFilter('all')}
    >
      <Box class="w-4 h-4 text-zinc-500 shrink-0" />
      <span class="text-zinc-500 uppercase tracking-widest text-[10px] font-semibold">{t.total}</span>
      <span class="font-mono {filterState === 'all' ? 'text-accent-cyan' : 'text-zinc-200'}">{totalContainers}</span>
    </div>
    <div class="w-px h-5 bg-zinc-800 shrink-0"></div>
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div
      class="flex items-center gap-3 px-5 cursor-pointer hover:bg-zinc-900 transition-colors"
      onclick={() => toggleFilter('running')}
    >
      <Wifi class="w-4 h-4 text-zinc-500 shrink-0" />
      <span class="text-zinc-500 uppercase tracking-widest text-[10px] font-semibold">{t.running}</span>
      <span class="font-mono {filterState === 'running' ? 'text-green-400' : 'text-zinc-200'}">{runningContainers}</span>
    </div>
    <div class="w-px h-5 bg-zinc-800 shrink-0"></div>
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div
      class="flex items-center gap-3 px-5 cursor-pointer hover:bg-zinc-900 transition-colors"
      onclick={() => toggleFilter('stopped')}
    >
      <WifiOff class="w-4 h-4 text-zinc-500 shrink-0" />
      <span class="text-zinc-500 uppercase tracking-widest text-[10px] font-semibold">{t.stopped}</span>
      <span class="font-mono {filterState === 'stopped' ? 'text-red-400' : 'text-zinc-200'}">{stoppedContainers}</span>
    </div>
    <div class="w-px h-5 bg-zinc-800 shrink-0"></div>
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div
      class="flex items-center gap-3 px-5 cursor-pointer hover:bg-zinc-900 transition-colors"
      onclick={() => toggleFilter('updates')}
    >
      <ArrowUpCircle class="w-4 h-4 text-zinc-500 shrink-0" />
      <span class="text-zinc-500 uppercase tracking-widest text-[10px] font-semibold">{t.pendingUpdates}</span>
      <span class="font-mono {filterState === 'updates' ? 'text-orange-400' : ($pendingUpdatesCount > 0 ? 'text-orange-400' : 'text-zinc-200')}">{$pendingUpdatesCount}</span>
    </div>
  </div>

  <!-- Top Resources Section -->
  {#if !isLoading && runningContainers > 0}
    <section>
      <div class="sticky top-0 z-10 border-b border-zinc-800 bg-zinc-950 px-6 py-3 flex items-center justify-between">
        <div class="flex items-center gap-3">
          <BarChart3 class="w-4 h-4 text-zinc-500" />
          <span class="text-xs uppercase tracking-widest text-zinc-500 font-semibold">
            {$language === "es" ? "Top Recursos" : "Top Resources"}
          </span>
          {#if $selectedHost}
            <span class="text-xs font-mono px-2 py-0.5 bg-zinc-800 text-zinc-300 border border-zinc-700">{$selectedHost}</span>
          {/if}
        </div>
        <div class="flex items-center gap-3">
          <div class="flex items-center gap-0 border border-zinc-800">
            <button
              class="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium transition-colors {resourceMetric === 'cpu' ? 'bg-zinc-800 text-zinc-100' : 'text-zinc-500 hover:text-zinc-300'}"
              onclick={() => (resourceMetric = "cpu")}
            >
              <Cpu class="w-3 h-3" /> CPU
            </button>
            <button
              class="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium transition-colors {resourceMetric === 'memory' ? 'bg-zinc-800 text-zinc-100' : 'text-zinc-500 hover:text-zinc-300'}"
              onclick={() => (resourceMetric = "memory")}
            >
              <MemoryStick class="w-3 h-3" /> {$language === "es" ? "Memoria" : "Memory"}
            </button>
            <button
              class="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium transition-colors {resourceMetric === 'network' ? 'bg-zinc-800 text-zinc-100' : 'text-zinc-500 hover:text-zinc-300'}"
              onclick={() => (resourceMetric = "network")}
            >
              <ArrowUpDown class="w-3 h-3" /> {$language === "es" ? "Red" : "Net"}
            </button>
            <button
              class="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium transition-colors {resourceMetric === 'restarts' ? 'bg-zinc-800 text-zinc-100' : 'text-zinc-500 hover:text-zinc-300'}"
              onclick={() => (resourceMetric = "restarts")}
            >
              <RotateCcw class="w-3 h-3" /> {$language === "es" ? "Reinicios" : "Restarts"}
            </button>
          </div>
          <div class="flex items-center gap-0 border border-zinc-800">
            {#each topNOptions as n}
              <button
                class="px-2.5 py-1.5 text-xs font-mono transition-colors {topN === n ? 'bg-zinc-800 text-zinc-100' : 'text-zinc-500 hover:text-zinc-300'}"
                onclick={() => (topN = n)}
              >{n}</button>
            {/each}
          </div>
          <button
            class="text-xs text-zinc-600 hover:text-zinc-400 transition-colors"
            onclick={() => (showResourceLeaderboard = !showResourceLeaderboard)}
          >
            {showResourceLeaderboard ? ($language === "es" ? "Ocultar" : "Hide") : ($language === "es" ? "Mostrar" : "Show")}
          </button>
        </div>
      </div>

      {#if showResourceLeaderboard}
        {#if topContainers.length === 0}
          <div class="px-6 py-8 text-center text-zinc-600 text-sm border-b border-zinc-800">
            {$language === "es" ? "No hay datos de recursos disponibles" : "No resource data available"}
          </div>
        {:else}
          <div class="border-b border-zinc-800">
            {#each topContainers as item, i}
              {@const percent = getMetricPercent(item.stats)}
              {@const barWidth = topMaxValue > 0 ? (percent / topMaxValue) * 100 : 0}
              <div class="flex items-center gap-4 px-6 py-2.5 border-b border-zinc-900 hover:bg-zinc-900/50 transition-colors">
                <span class="w-5 text-right text-xs font-mono shrink-0 {i < 3 ? 'text-zinc-300' : 'text-zinc-600'}">{i + 1}</span>
                <div class="w-36 shrink-0 min-w-0">
                  <p class="text-sm font-mono text-zinc-200 truncate">{item.container.name}</p>
                  <p class="text-[10px] text-zinc-600 truncate font-mono">{item.container.hostId}</p>
                </div>
                <div class="flex-1 h-4 bg-zinc-900 overflow-hidden">
                  <div
                    class="h-full transition-all duration-500 ease-out {getMetricBarColor(percent)}"
                    style="width: {Math.max(1, barWidth)}%"
                  ></div>
                </div>
                <span class="w-16 text-right text-sm font-mono tabular-nums text-zinc-300 shrink-0">{getMetricValue(item.stats)}</span>
              </div>
            {/each}
          </div>
        {/if}
      {/if}
    </section>
  {/if}

  <!-- Hosts Section -->
  <section>
    <div class="sticky top-0 z-10 border-b border-zinc-800 bg-zinc-950 px-6 py-3 flex items-center gap-3">
      <Server class="w-4 h-4 text-zinc-500" />
      <span class="text-xs uppercase tracking-widest text-zinc-500 font-semibold">{t.hosts}</span>
      {#if $selectedHost}
        <span class="text-xs font-mono px-2 py-0.5 bg-zinc-800 text-zinc-300 border border-zinc-700">{t.filterByHost}: {$selectedHost}</span>
      {/if}
    </div>

    {#if isLoading}
      {#each [1, 2] as _}
        <div class="border-b border-zinc-800 px-6 py-4 animate-pulse flex items-center gap-6">
          <div class="h-3 bg-zinc-800 rounded w-32"></div>
          <div class="h-3 bg-zinc-800 rounded w-48 flex-1"></div>
        </div>
      {/each}
    {:else}
      {#each $hosts as host}
        <div class="group border-b border-zinc-800 hover:bg-zinc-900/40 transition-colors" style="border-left: 3px solid {host.online ? '#22c55e' : '#52525b'}">
          <HostCard
            {host}
            resourcesOpen={expandedHostId === host.id}
            onToggleResources={toggleHostResources}
            onOpenHostTerminal={openHostTerminal}
            onOpenHostFiles={(target) => (hostFiles = target)}
          />
        </div>
      {/each}

      {#if expandedHost}
        <div class="border-b border-zinc-800 bg-zinc-900/30 px-6 py-4">
          <div class="flex items-center justify-between mb-3">
            <div class="flex items-center gap-3">
              <Server class="w-4 h-4 text-zinc-500" />
              <span class="text-sm font-mono text-zinc-200">{expandedHost.name}</span>
              <span class="text-xs font-mono text-zinc-600">{expandedHost.id}</span>
            </div>
            <span class="text-xs text-zinc-600 uppercase tracking-widest">
              {$language === "es" ? "Recursos en vivo" : "Live resources"}
            </span>
          </div>
          <ResourceChart host={expandedHost} />
        </div>
      {/if}

      <div class="border-b border-zinc-800 px-6 py-4">
        <ContainerActivityChart />
      </div>
    {/if}
  </section>

  <!-- Containers Section -->
  <section>
    <div class="sticky top-0 z-10 border-b border-zinc-800 bg-zinc-950 px-6 py-3 flex items-center justify-between">
      <div class="flex items-center gap-3">
        <Box class="w-4 h-4 text-zinc-500" />
        <span class="text-xs uppercase tracking-widest text-zinc-500 font-semibold">{t.containers}</span>
        <span class="text-xs font-mono text-zinc-600">({displayContainers.length})</span>
      </div>
      <div class="flex items-center gap-0 border border-zinc-800">
        <button
          class="px-3 py-1.5 text-xs transition-colors {filterState === 'all' ? 'bg-zinc-800 text-zinc-100' : 'text-zinc-500 hover:text-zinc-300'}"
          onclick={() => toggleFilter("all")}
        >{t.all}</button>
        <button
          class="px-3 py-1.5 text-xs transition-colors {filterState === 'running' ? 'bg-green-500/20 text-green-400' : 'text-zinc-500 hover:text-zinc-300'}"
          onclick={() => toggleFilter("running")}
        >{t.running}</button>
        <button
          class="px-3 py-1.5 text-xs transition-colors {filterState === 'stopped' ? 'bg-red-500/20 text-red-400' : 'text-zinc-500 hover:text-zinc-300'}"
          onclick={() => toggleFilter("stopped")}
        >{t.stopped}</button>
      </div>
    </div>

    {#if isLoading}
      {#each [1, 2, 3, 4, 5, 6] as _}
        <div class="border-b border-zinc-800 px-6 py-4 animate-pulse flex items-center gap-6">
          <div class="h-3 bg-zinc-800 rounded w-40"></div>
          <div class="h-3 bg-zinc-800 rounded flex-1"></div>
          <div class="h-3 bg-zinc-800 rounded w-20"></div>
        </div>
      {/each}
    {:else if displayContainers.length === 0}
      <div class="px-6 py-16 text-center">
        <Box class="w-10 h-10 text-zinc-700 mx-auto mb-3" />
        <p class="text-zinc-600 text-sm">{t.noContainers}</p>
      </div>
    {:else}
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3">
        {#each pagedContainers as container (container.id)}
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

      {#if displayContainers.length > pageSize}
        <div class="border-t border-zinc-800 px-6 py-3 flex flex-col md:flex-row md:items-center md:justify-between gap-3">
          <div class="text-xs text-zinc-600 font-mono">
            {#if displayContainers.length > 0}
              {#if $language === "es"}
                Mostrando {(currentPage - 1) * pageSize + 1}–{Math.min(currentPage * pageSize, displayContainers.length)} de {displayContainers.length}
              {:else}
                Showing {(currentPage - 1) * pageSize + 1}–{Math.min(currentPage * pageSize, displayContainers.length)} of {displayContainers.length}
              {/if}
            {/if}
          </div>
          <div class="flex items-center justify-center gap-0 border border-zinc-800">
            <button
              class="px-3 py-1.5 text-xs text-zinc-500 hover:text-zinc-300 hover:bg-zinc-900 transition-colors disabled:opacity-30"
              onclick={() => goToPage(currentPage - 1)}
              disabled={currentPage === 1}
            >{$language === "es" ? "Anterior" : "Prev"}</button>
            {#each pageNumbers as page}
              <button
                class="w-8 h-8 text-xs font-mono transition-colors {page === currentPage ? 'bg-zinc-800 text-zinc-100' : 'text-zinc-500 hover:text-zinc-300 hover:bg-zinc-900'}"
                onclick={() => goToPage(page)}
              >{page}</button>
            {/each}
            <button
              class="px-3 py-1.5 text-xs text-zinc-500 hover:text-zinc-300 hover:bg-zinc-900 transition-colors disabled:opacity-30"
              onclick={() => goToPage(currentPage + 1)}
              disabled={currentPage === totalPages}
            >{$language === "es" ? "Siguiente" : "Next"}</button>
          </div>
          <div class="flex items-center justify-end gap-0 border border-zinc-800">
            {#each pageSizeOptions as size}
              <button
                class="px-2.5 py-1.5 text-xs font-mono transition-colors {pageSize === size ? 'bg-zinc-800 text-zinc-100' : 'text-zinc-500 hover:text-zinc-300'}"
                onclick={() => (pageSize = size)}
              >{size}</button>
            {/each}
          </div>
        </div>
      {/if}
    {/if}
  </section>

  <!-- Terminal Modal (Lazy Loaded) -->
  {#if terminalContainer && Terminal}
    {@const TerminalComponent = Terminal}
    <TerminalComponent
      container={terminalContainer}
      onClose={() => (terminalContainer = null)}
    />
  {/if}

  {#if hostTerminal && Terminal}
    {@const TerminalComponent = Terminal}
    <TerminalComponent
      host={hostTerminal}
      mode="host"
      onClose={() => (hostTerminal = null)}
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

  {#if hostFiles}
    <HostFiles host={hostFiles} onClose={() => (hostFiles = null)} />
  {/if}
</div>
