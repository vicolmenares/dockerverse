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
    Download,
    Loader2,
    CheckCircle2,
    RefreshCw,
    Network,
    Box,
    ExternalLink,
  } from "lucide-svelte";
  import type { Container, ContainerStats } from "$lib/api/docker";
  import UpdateModal from "./UpdateModal.svelte";
  import {
    containerAction,
    triggerContainerUpdate,
    checkImageUpdate,
    formatBytes,
    formatUptime,
  } from "$lib/api/docker";
  import {
    language,
    translations,
    imageUpdates,
    checkForUpdates,
    lastUpdateCheck,
  } from "$lib/stores/docker";

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
  let updating = $state(false);
  let checking = $state(false);
  let showUpdateModal = $state(false);
  let t = $derived(translations[$language]);

  // Get the update info for this container
  let updateInfo = $derived.by(() => {
    const updates = $imageUpdates;
    return updates.find(
      (u) =>
        u.containerId === container.id &&
        u.hostId === container.hostId,
    ) ?? null;
  });

  // Check if container has a pending update
  let hasUpdate = $derived(updateInfo?.hasUpdate ?? false);

  // Check if update check has been performed for this container
  let hasBeenChecked = $derived(updateInfo !== null);

  // Check if container has Watchtower enabled
  let hasWatchtower = $derived(
    container.labels?.["com.centurylinklabs.watchtower.enable"] === "true" ||
      container.labels?.["com.centurylinklabs.watchtower"] === "true",
  );

  // Build tooltip text with version info
  let updateTooltip = $derived.by(() => {
    if (!updateInfo) return $language === "es" ? "No verificado" : "Not checked";
    const img = updateInfo.image || container.image;
    const currentShort = updateInfo.currentDigest
      ? updateInfo.currentDigest.split("@").pop()?.substring(0, 19) + "..."
      : "unknown";
    if (updateInfo.hasUpdate) {
      const latestShort = updateInfo.latestDigest
        ? updateInfo.latestDigest.substring(0, 19) + "..."
        : "unknown";
      return $language === "es"
        ? `Actual: ${img} (${currentShort})\nNuevo: ${latestShort}\nClick para actualizar con Watchtower`
        : `Current: ${img} (${currentShort})\nLatest: ${latestShort}\nClick to update via Watchtower`;
    }
    return $language === "es"
      ? `${img} (${currentShort})\nImagen actualizada`
      : `${img} (${currentShort})\nUp to date`;
  });

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

  function handleUpdate() {
    showUpdateModal = true;
  }

  async function handleCheckUpdate() {
    checking = true;
    try {
      const result = await checkImageUpdate(container.hostId, container.id);
      // Update the store with this single result
      imageUpdates.update((current) => {
        const idx = current.findIndex(
          (u) => u.containerId === container.id && u.hostId === container.hostId,
        );
        if (idx >= 0) {
          current[idx] = result;
        } else {
          current = [...current, result];
        }
        return current;
      });
    } catch (e) {
      console.error("Check update failed:", e);
    }
    checking = false;
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
          class="w-2.5 h-2.5 rounded-full block {getStateColor(
            container.state,
          )}"
        ></span>
        {#if hasUpdate}
          <span
            class="absolute -top-1 -right-1 w-2 h-2 rounded-full bg-accent-orange"
          ></span>
        {/if}
      </div>
      <div class="min-w-0 flex-1">
        <div class="flex items-center gap-1.5">
          <h4
            class="font-medium text-foreground truncate"
            title={container.name}
          >
            {container.name}
          </h4>
          {#if hasUpdate}
            <span
              class="flex items-center gap-1 text-[10px] font-semibold text-accent-orange bg-accent-orange/15 border border-accent-orange/30 px-1.5 py-0.5 rounded-full flex-shrink-0 update-badge cursor-help"
              title={updateTooltip}
            >
              <ArrowUpCircle class="w-3 h-3" />
              UPDATE
            </span>
          {:else if hasBeenChecked}
            <span
              class="flex items-center gap-1 text-[10px] font-medium text-running bg-running/15 border border-running/30 px-1.5 py-0.5 rounded-full flex-shrink-0 cursor-help"
              title={updateTooltip}
            >
              <CheckCircle2 class="w-2.5 h-2.5" />
            </span>
          {:else if hasWatchtower}
            <span
              class="flex items-center gap-1 text-[10px] font-medium text-running/70 bg-running/10 px-1.5 py-0.5 rounded-full flex-shrink-0 cursor-help"
              title={$language === "es" ? "Watchtower monitoreando" : "Watchtower monitoring"}
            >
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

  <!-- Stats Grid (fixed height to avoid layout shift) -->
  <div class="border-y border-border rounded-lg bg-background-tertiary/40 px-2 py-2 min-h-[72px]">
    <div class="grid grid-cols-4 gap-2">
      <div class="text-center">
        <p
          class="text-xs text-foreground-muted flex items-center justify-center gap-1 mb-1"
        >
          <Cpu class="w-3 h-3" />
        </p>
        <p
          class="text-sm font-mono {stats && isRunning && stats.cpuPercent > 50
            ? 'text-accent-orange'
            : 'text-foreground'}"
        >
          {#if isRunning && stats}
            {stats.cpuPercent.toFixed(1)}%
          {:else}
            —
          {/if}
        </p>
      </div>
      <div class="text-center">
        <p
          class="text-xs text-foreground-muted flex items-center justify-center gap-1 mb-1"
        >
          <HardDrive class="w-3 h-3" />
        </p>
        <p
          class="text-sm font-mono {stats && isRunning && stats.memoryPercent > 70
            ? 'text-accent-orange'
            : 'text-foreground'}"
        >
          {#if isRunning && stats}
            {stats.memoryPercent.toFixed(1)}%
          {:else}
            —
          {/if}
        </p>
      </div>
      <div class="text-center">
        <p
          class="text-xs text-foreground-muted flex items-center justify-center gap-1 mb-1"
        >
          <ArrowUpDown class="w-3 h-3" />
        </p>
        <p class="text-sm font-mono text-foreground">
          {#if isRunning && stats}
            {formatBytes(stats.networkRx)}/{formatBytes(stats.networkTx)}
          {:else}
            —
          {/if}
        </p>
      </div>
      <div class="text-center">
        <p class="text-xs text-foreground-muted mb-1">{t.uptime}</p>
        <p class="text-sm font-mono text-running">
          {#if isRunning}
            {formatUptime(container.status)}
          {:else}
            —
          {/if}
        </p>
      </div>
    </div>
    <p class="mt-1 text-[10px] text-center text-foreground-muted">
      {#if !isRunning}
        {t.stoppedContainer}
      {:else if !stats}
        {$language === "es" ? "Esperando metricas" : "Waiting for metrics"}
      {:else}
        &nbsp;
      {/if}
    </p>
  </div>

  <!-- Container Details: Ports, IP, Volumes -->
  {#if container.ports?.length > 0 || Object.keys(container.networks || {}).length > 0 || container.volumes > 0}
    <div class="flex flex-wrap gap-1.5">
      {#each (container.ports || []).filter(p => p.public > 0) as port}
        <span class="inline-flex items-center gap-1 text-[10px] px-1.5 py-0.5 bg-primary/10 text-primary rounded-full border border-primary/20">
          <ExternalLink class="w-2.5 h-2.5" />
          :{port.public}
        </span>
      {/each}
      {#each Object.entries(container.networks || {}) as [netName, ip]}
        {#if ip}
          <span class="inline-flex items-center gap-1 text-[10px] px-1.5 py-0.5 bg-accent-cyan/10 text-accent-cyan rounded-full border border-accent-cyan/20" title="{netName}: {ip}">
            <Network class="w-2.5 h-2.5" />
            {ip}
          </span>
        {/if}
      {/each}
      {#if container.volumes > 0}
        <span class="inline-flex items-center gap-1 text-[10px] px-1.5 py-0.5 bg-accent-purple/10 text-accent-purple rounded-full border border-accent-purple/20">
          <Box class="w-2.5 h-2.5" />
          {container.volumes} vol{container.volumes > 1 ? 's' : ''}
        </span>
      {/if}
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
      {#if hasUpdate}
        <button
          class="flex items-center gap-1.5 px-2.5 py-1.5 bg-accent-orange/20 text-accent-orange rounded-lg hover:bg-accent-orange/30 transition-colors"
          onclick={handleUpdate}
          disabled={updating}
          title={updateTooltip}
        >
          {#if updating}
            <Loader2 class="w-4 h-4 animate-spin" />
          {:else}
            <Download class="w-4 h-4" />
          {/if}
          <span class="text-xs font-medium"
            >{$language === "es" ? "Actualizar" : "Update"}</span
          >
        </button>
      {:else if isRunning}
        <button
          class="btn-icon hover:bg-primary/20 hover:text-primary"
          onclick={handleCheckUpdate}
          disabled={checking}
          title={$language === "es" ? "Verificar actualización" : "Check for update"}
        >
          {#if checking}
            <Loader2 class="w-4 h-4 animate-spin" />
          {:else}
            <RefreshCw class="w-4 h-4" />
          {/if}
        </button>
      {/if}
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

{#if showUpdateModal}
  <UpdateModal {container} onclose={() => (showUpdateModal = false)} />
{/if}

<style>
  .watchtower-spin {
    animation: watchtower-rotate 8s linear infinite;
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
