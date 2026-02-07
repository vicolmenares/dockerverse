<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { Box, Server, AlertTriangle, Wifi, WifiOff } from "lucide-svelte";
  import {
    fetchContainers,
    fetchHosts,
    createEventStream,
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
