<script lang="ts">
  import {
    Cpu,
    HardDrive,
    Server,
    ChevronDown,
    ChevronUp,
  } from "lucide-svelte";
  import type { Host } from "$lib/api/docker";
  import { selectedHost, language, translations } from "$lib/stores/docker";
  import ResourceChart from "./ResourceChart.svelte";

  let { host, onclick }: { host: Host; onclick?: () => void } = $props();

  let isSelected = $derived($selectedHost === host.id);
  let showResources = $state(false);
  let t = $derived(translations[$language]);

  function handleClick() {
    if ($selectedHost === host.id) {
      selectedHost.set(null); // Deselect
    } else {
      selectedHost.set(host.id); // Select
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
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
  class="card card-hover p-5 cursor-pointer transition-all {isSelected
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
    <span class="flex items-center gap-2 text-sm">
      <span class="w-2 h-2 rounded-full {getStatusColor(host.online)}"></span>
      {host.online ? t.online : t.offline}
    </span>
  </div>

  <div class="grid grid-cols-3 gap-4">
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
  </div>

  <!-- Expand/Collapse Resource Charts -->
  {#if host.online}
    <button
      class="mt-3 w-full flex items-center justify-center gap-1 py-1.5 text-xs text-foreground-muted hover:text-foreground hover:bg-background-tertiary/50 rounded-lg transition-colors"
      onclick={(e) => {
        e.stopPropagation();
        showResources = !showResources;
      }}
    >
      {#if showResources}
        <ChevronUp class="w-4 h-4" />
        <span>{t.hideResources || "Hide resources"}</span>
      {:else}
        <ChevronDown class="w-4 h-4" />
        <span>{t.showResources || "Show resources"}</span>
      {/if}
    </button>
  {/if}

  <!-- Resource Charts -->
  {#if showResources && host.online}
    <ResourceChart {host} />
  {/if}
</div>
