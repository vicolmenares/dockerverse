<script lang="ts">
  import {
    Play,
    Square,
    RotateCcw,
    Terminal as TermIcon,
    ScrollText,
    Cpu,
    HardDrive,
    ArrowUpDown,
    ArrowUpCircle,
  } from "lucide-svelte";
  import type { Container, ContainerStats, ImageUpdate } from "$lib/api/docker";
  import { containerAction, formatBytes, formatUptime } from "$lib/api/docker";
  import { language, translations, imageUpdates } from "$lib/stores/docker";

  let {
    container,
    stats,
    onTerminal,
    onLogs,
  }: {
    container: Container;
    stats?: ContainerStats;
    onTerminal: () => void;
    onLogs: () => void;
  } = $props();

  let loading = $state(false);
  let t = $derived(translations[$language]);

  // Check if container has a pending update
  let hasUpdate = $derived.by(() => {
    const updates = $imageUpdates;
    return updates.some(u => u.containerId === container.id && u.hostId === container.hostId && u.hasUpdate);
  });

  // Check if container has Watchtower enabled
  let hasWatchtower = $derived(
    container.labels?.['com.centurylinklabs.watchtower.enable'] === 'true' ||
    container.labels?.['com.centurylinklabs.watchtower'] === 'true'
  );

  function getStateColor(state: string) {
    switch (state) {
      case "running":
        return "bg-running";
      case "exited":
        return "bg-stopped";
      case "paused":
        return "bg-paused";
      default:
        return "bg-foreground-muted";
    }
  }

  function getStateBg(state: string) {
    switch (state) {
      case "running":
        return "bg-running/10 text-running";
      case "exited":
        return "bg-stopped/10 text-stopped";
      case "paused":
        return "bg-paused/10 text-paused";
      default:
        return "bg-foreground-muted/10 text-foreground-muted";
    }
  }

  async function handleAction(action: "start" | "stop" | "restart") {
    loading = true;
    try {
      await containerAction(container.hostId, container.id, action);
    } catch (e) {
      console.error("Action failed:", e);
    }
    loading = false;
  }

  function parseImage(image: string) {
    const parts = image.split(":");
    return {
      name: parts[0].split("/").pop() || parts[0],
      tag: parts[1] || "latest",
    };
  }

  // Use derived values to track container state changes
  const image = $derived(parseImage(container.image));
  const isRunning = $derived(container.state === "running");
</script>

<div class="card card-hover p-4 flex flex-col gap-3">
  <!-- Header -->
  <div class="flex items-start justify-between">
    <div class="flex items-center gap-3 min-w-0">
      <div class="relative flex-shrink-0">
        <span
          class="w-2.5 h-2.5 rounded-full block {getStateColor(container.state)}"
        ></span>
        {#if hasUpdate}
          <span class="absolute -top-1 -right-1 w-2 h-2 rounded-full bg-accent-orange update-ping"></span>
        {/if}
      </div>
      <div class="min-w-0 flex-1">
        <div class="flex items-center gap-1.5">
          <h4 class="font-medium text-foreground truncate" title={container.name}>
            {container.name}
          </h4>
          {#if hasUpdate}
            <span class="flex items-center gap-1 text-[10px] font-semibold text-accent-orange bg-accent-orange/15 border border-accent-orange/30 px-1.5 py-0.5 rounded-full flex-shrink-0 update-badge" title="Update available">
              <ArrowUpCircle class="w-3 h-3" />
              UPDATE
            </span>
          {:else if hasWatchtower}
            <span class="flex items-center gap-1 text-[10px] font-medium text-running/70 bg-running/10 px-1.5 py-0.5 rounded-full flex-shrink-0" title="Watchtower monitoring">
              <RotateCcw class="w-2.5 h-2.5 watchtower-spin" />
            </span>
          {/if}
        </div>
        <p
          class="text-xs text-foreground-muted truncate"
          title={container.image}
        >
          {image.name}<span class="text-primary">:{image.tag}</span>
        </p>
      </div>
    </div>
    <span
      class="text-xs px-2 py-1 rounded-full flex-shrink-0 {getStateBg(
        container.state,
      )}"
    >
      {container.state}
    </span>
  </div>

  <!-- Stats Grid (only when running) -->
  {#if isRunning && stats}
    <div class="grid grid-cols-4 gap-2 py-2 border-y border-border">
      <div class="text-center">
        <p
          class="text-xs text-foreground-muted flex items-center justify-center gap-1 mb-1"
        >
          <Cpu class="w-3 h-3" />
        </p>
        <p
          class="text-sm font-mono {stats.cpuPercent > 50
            ? 'text-accent-orange'
            : 'text-foreground'}"
        >
          {stats.cpuPercent.toFixed(1)}%
        </p>
      </div>
      <div class="text-center">
        <p
          class="text-xs text-foreground-muted flex items-center justify-center gap-1 mb-1"
        >
          <HardDrive class="w-3 h-3" />
        </p>
        <p
          class="text-sm font-mono {stats.memoryPercent > 70
            ? 'text-accent-orange'
            : 'text-foreground'}"
        >
          {stats.memoryPercent.toFixed(1)}%
        </p>
      </div>
      <div class="text-center">
        <p
          class="text-xs text-foreground-muted flex items-center justify-center gap-1 mb-1"
        >
          <ArrowUpDown class="w-3 h-3" />
        </p>
        <p class="text-sm font-mono text-foreground">
          {formatBytes(stats.networkRx)}/{formatBytes(stats.networkTx)}
        </p>
      </div>
      <div class="text-center">
        <p class="text-xs text-foreground-muted mb-1">{t.uptime}</p>
        <p class="text-sm font-mono text-running">
          {formatUptime(container.status)}
        </p>
      </div>
    </div>
  {:else if !isRunning}
    <div class="py-2 border-y border-border text-center">
      <p class="text-sm text-foreground-muted">{t.stoppedContainer}</p>
    </div>
  {/if}

  <!-- Actions -->
  <div class="flex items-center justify-between gap-2">
    <div class="flex gap-1">
      {#if isRunning}
        <button
          class="btn-icon hover:bg-background-tertiary"
          onclick={onTerminal}
          title={t.terminal}
        >
          <TermIcon class="w-4 h-4" />
        </button>
        <button
          class="btn-icon hover:bg-background-tertiary"
          onclick={onLogs}
          title={t.logs}
        >
          <ScrollText class="w-4 h-4" />
        </button>
      {:else}
        <!-- Show logs even for stopped containers -->
        <button
          class="btn-icon hover:bg-background-tertiary"
          onclick={onLogs}
          title={t.logs}
        >
          <ScrollText class="w-4 h-4" />
        </button>
      {/if}
    </div>

    <div class="flex gap-1">
      {#if isRunning}
        <button
          class="btn-icon hover:bg-stopped/20 hover:text-stopped"
          onclick={() => handleAction("stop")}
          disabled={loading}
          title={t.stop}
        >
          <Square class="w-4 h-4" />
        </button>
        <button
          class="btn-icon hover:bg-primary/20 hover:text-primary"
          onclick={() => handleAction("restart")}
          disabled={loading}
          title={t.restart}
        >
          <RotateCcw class="w-4 h-4" />
        </button>
      {:else}
        <!-- Prominent Start button for stopped containers -->
        <button
          class="flex items-center gap-1.5 px-3 py-1.5 bg-running/20 text-running rounded-lg hover:bg-running/30 transition-colors"
          onclick={() => handleAction("start")}
          disabled={loading}
          title={t.start}
        >
          <Play class="w-4 h-4" />
          <span class="text-xs font-medium">{t.start}</span>
        </button>
      {/if}
    </div>
  </div>

  <!-- Host tag -->
  <div class="text-xs text-foreground-muted">
    <span class="px-1.5 py-0.5 bg-background-tertiary rounded"
      >{container.hostId}</span
    >
  </div>
</div>

<style>
  .update-ping {
    animation: ping 1.5s cubic-bezier(0, 0, 0.2, 1) infinite;
  }

  .update-badge {
    animation: badge-pulse 2s ease-in-out infinite;
  }

  .watchtower-spin {
    animation: watchtower-rotate 8s linear infinite;
  }

  @keyframes ping {
    0% {
      transform: scale(1);
      opacity: 1;
    }
    75%, 100% {
      transform: scale(2.5);
      opacity: 0;
    }
  }

  @keyframes badge-pulse {
    0%, 100% {
      opacity: 1;
    }
    50% {
      opacity: 0.7;
    }
  }

  @keyframes watchtower-rotate {
    from {
      transform: rotate(0deg);
    }
    to {
      transform: rotate(360deg);
    }
  }
</style>