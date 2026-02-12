<script lang="ts">
  import {
    Cpu,
    HardDrive,
    Server,
    Database,
    ChevronDown,
    ChevronUp,
    Terminal as TerminalIcon,
  } from "lucide-svelte";
  import type { Host } from "$lib/api/docker";
  import { selectedHost, language, translations } from "$lib/stores/docker";

  let {
    host,
    onclick,
    resourcesOpen = false,
    onToggleResources,
  }: {
    host: Host;
    onclick?: () => void;
    resourcesOpen?: boolean;
    onToggleResources?: (hostId: string) => void;
  } = $props();

  let isSelected = $derived($selectedHost === host.id);
  let t = $derived(translations[$language]);

  function handleClick() {
    if ($selectedHost === host.id) {
      selectedHost.set(null);
    } else {
      selectedHost.set(host.id);
    }
    onclick?.();
  }

  function getStatusColor(online: boolean) {
    return online ? "bg-running" : "bg-stopped";
  }

  function getCpuColor(percent: number) {
    if (percent >= 80) return "text-accent-red";
    if (percent >= 50) return "text-accent-orange";
    return "text-running";
  }

  function getMemColor(percent: number) {
    if (percent >= 80) return "text-accent-red";
    if (percent >= 50) return "text-accent-orange";
    return "text-accent-cyan";
  }

  function getDiskColor(percent: number) {
    if (percent >= 90) return "text-accent-red";
    if (percent >= 70) return "text-accent-orange";
    return "text-accent-purple";
  }

  function formatSize(bytes: number): string {
    if (bytes < 1073741824) return (bytes / 1048576).toFixed(0) + " MB";
    return (bytes / 1073741824).toFixed(1) + " GB";
  }

  let diskTotalUsed = $derived(
    (host.disks || []).reduce((sum, d) => sum + d.usedBytes, 0),
  );
  let diskTotalFree = $derived(
    (host.disks || []).reduce((sum, d) => sum + d.freeBytes, 0),
  );
  let diskTotalSize = $derived(
    (host.disks || []).reduce((sum, d) => sum + d.totalBytes, 0),
  );
  let diskPercent = $derived(
    diskTotalSize > 0 ? (diskTotalUsed / diskTotalSize) * 100 : 0,
  );
  let diskPercentText = $derived(
    diskTotalSize > 0 ? `${diskPercent.toFixed(1)}%` : "—",
  );
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
  class="card card-hover p-5 cursor-pointer transition-all bg-gradient-to-br from-background-secondary/70 via-background-secondary/40 to-background-tertiary/10 {isSelected
    ? 'ring-2 ring-primary'
    : ''}"
  onclick={handleClick}
>
  <div class="flex items-center justify-between mb-4">
    <div class="flex items-center gap-3">
      <div class="p-2.5 bg-background-tertiary/50 rounded-lg">
        <Server class="w-5 h-5 text-primary" />
      </div>
      <div>
        <h3 class="font-semibold text-foreground">{host.name}</h3>
        <p class="text-sm text-foreground-muted">{host.id}</p>
      </div>
    </div>
    <div class="flex items-center gap-2 text-sm">
      {#if host.sshHost}
        <a
          class="inline-flex items-center gap-1.5 px-2.5 py-1 bg-background-tertiary/70 text-foreground-muted rounded-lg hover:text-foreground hover:bg-background-tertiary transition-colors"
          href={`ssh://${host.sshHost}`}
          target="_blank"
          rel="noreferrer"
          onclick={(e) => e.stopPropagation()}
        >
          <TerminalIcon class="w-3.5 h-3.5" />
          <span class="text-xs font-medium">SSH</span>
        </a>
      {/if}
      <span class="flex items-center gap-2 text-sm">
        <span class="w-2 h-2 rounded-full {getStatusColor(host.online)}"></span>
        {host.online ? t.online : t.offline}
      </span>
    </div>
  </div>

  <div class="grid grid-cols-4 gap-3">
    <!-- Containers -->
    <div class="text-center">
      <p class="metric-value text-foreground">
        {host.runningCount}<span class="text-foreground-muted"
          >/{host.containerCount}</span
        >
      </p>
      <p class="metric-label">{t.containers}</p>
    </div>

    <!-- CPU -->
    <div class="text-center">
      <p class="metric-value tabular-nums {getCpuColor(host.cpuPercent)}">
        {host.cpuPercent.toFixed(1)}%
      </p>
      <p class="metric-label flex items-center justify-center gap-1">
        <Cpu class="w-3 h-3" /> CPU
      </p>
    </div>

    <!-- Memory -->
    <div class="text-center">
      <p class="metric-value tabular-nums {getMemColor(host.memoryPercent)}">
        {host.memoryPercent.toFixed(1)}%
      </p>
      <p class="metric-label flex items-center justify-center gap-1">
        <HardDrive class="w-3 h-3" /> RAM
      </p>
      {#if host.memoryTotal > 0}
        <p class="text-[10px] text-foreground-muted">
          {formatSize(host.memoryUsed)} / {formatSize(host.memoryTotal)}
        </p>
      {/if}
    </div>

    <!-- Disk -->
    <div class="text-center">
      <p class="metric-value tabular-nums {getDiskColor(diskPercent)}">
        {diskPercentText}
      </p>
      <p class="metric-label flex items-center justify-center gap-1">
        <Database class="w-3 h-3" /> Disk
      </p>
      {#if diskTotalSize > 0}
        <p class="text-[10px] text-foreground-muted">
          {formatSize(diskTotalFree)} {$language === "es" ? "libre" : "free"} / {formatSize(diskTotalSize)}
        </p>
      {/if}
    </div>
  </div>

  <!-- Progress bars -->
  <div class="mt-4 space-y-2">
    <div class="h-1.5 bg-background-tertiary rounded-full overflow-hidden">
      <div
        class="h-full transition-all duration-500 {host.cpuPercent >= 80
          ? 'bg-accent-red'
          : host.cpuPercent >= 50
            ? 'bg-accent-orange'
            : 'bg-running'}"
        style="width: {Math.min(host.cpuPercent, 100)}%"
      ></div>
    </div>
    <div class="h-1.5 bg-background-tertiary rounded-full overflow-hidden">
      <div
        class="h-full transition-all duration-500 {host.memoryPercent >= 80
          ? 'bg-accent-red'
          : host.memoryPercent >= 50
            ? 'bg-accent-orange'
            : 'bg-accent-cyan'}"
        style="width: {Math.min(host.memoryPercent, 100)}%"
      ></div>
    </div>
    <div class="h-1.5 bg-background-tertiary rounded-full overflow-hidden">
      <div
        class="h-full transition-all duration-500 {diskPercent >= 90
          ? 'bg-accent-red'
          : diskPercent >= 70
            ? 'bg-accent-orange'
            : 'bg-accent-purple'}"
        style="width: {Math.min(diskPercent, 100)}%"
      ></div>
    </div>
    {#if host.disks && host.disks.length > 1}
      <div class="mt-1 space-y-1">
        {#each host.disks as disk}
          {@const pct = disk.totalBytes > 0 ? (disk.usedBytes / disk.totalBytes) * 100 : 0}
          <div class="flex items-center gap-2 text-[10px] text-foreground-muted">
            <span class="w-16 truncate" title={disk.mountPoint}>{disk.mountPoint}</span>
            <div class="flex-1 h-1 bg-background-tertiary rounded-full overflow-hidden">
              <div
                class="h-full transition-all duration-500 {pct >= 90
                  ? 'bg-accent-red'
                  : pct >= 70
                    ? 'bg-accent-orange'
                    : 'bg-accent-purple'}"
                style="width: {Math.min(pct, 100)}%"
              ></div>
            </div>
            <span class="tabular-nums">{formatSize(disk.freeBytes)}/{formatSize(disk.totalBytes)}</span>
          </div>
        {/each}
      </div>
    {/if}
  </div>

  <!-- Expand/Collapse Resource Charts -->
  {#if host.online}
    <button
      class="mt-3 w-full flex items-center justify-center gap-1 py-1.5 text-xs text-foreground-muted hover:text-foreground hover:bg-background-tertiary/50 rounded-lg transition-colors"
      onclick={(e) => {
        e.stopPropagation();
        onToggleResources?.(host.id);
      }}
      aria-pressed={resourcesOpen}
    >
      {#if resourcesOpen}
        <ChevronUp class="w-4 h-4" />
        <span>{t.hideResources || "Hide resources"}</span>
      {:else}
        <ChevronDown class="w-4 h-4" />
        <span>{t.showResources || "Show resources"}</span>
      {/if}
    </button>
  {/if}
</div>
