<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { X, Loader2, CheckCircle2, AlertCircle, Download, Shield, ScanSearch } from "lucide-svelte";
  import type { Container } from "$lib/api/docker";
  import { API_BASE } from "$lib/api/docker";
  import { language, checkForUpdates } from "$lib/stores/docker";

  let {
    container,
    onclose,
  }: {
    container: Container;
    onclose: () => void;
  } = $props();

  let started = $state(false);

  // Stream state
  type StreamStatus = "idle" | "streaming" | "success" | "scanning" | "error";
  let status = $state<StreamStatus>("idle");
  let statusMessage = $state("");
  let logLines = $state<string[]>([]);
  let showLogs = $state(true);
  let autoCloseTimer: ReturnType<typeof setTimeout> | null = null;
  let eventSource: EventSource | null = null;

  // Scan result state — always informational, never blocks the update
  let scanSummary = $state<{ critical: number; high: number; medium: number; low: number } | null>(null);
  let currentStage = $state("");

  function addLog(msg: string) {
    const ts = new Date().toLocaleTimeString();
    logLines = [...logLines, `[${ts}] ${msg}`];
  }

  function buildSSEUrl(): string {
    const token = typeof localStorage !== "undefined"
      ? localStorage.getItem("auth_access_token")
      : null;
    const params = new URLSearchParams();
    if (token) params.set("token", token);
    return `${API_BASE}/api/containers/${container.hostId}/${container.id}/update-stream?${params.toString()}`;
  }

  function openStream() {
    if (eventSource) {
      eventSource.close();
      eventSource = null;
    }

    status = "streaming";
    started = true;
    scanSummary = null;
    logLines = [];
    currentStage = "";

    addLog(
      $language === "es"
        ? `Iniciando actualización para ${container.name}...`
        : `Starting update for ${container.name}...`
    );

    eventSource = new EventSource(buildSSEUrl());

    eventSource.addEventListener("progress", (e) => {
      try {
        const data = JSON.parse(e.data) as { stage: string; message: string };
        currentStage = data.stage;
        addLog(data.message);
        if (data.stage === "scan_start") status = "scanning";
      } catch {}
    });

    eventSource.addEventListener("scan_progress", (e) => {
      try {
        const data = JSON.parse(e.data) as { scanner: string; message: string };
        addLog(`[${data.scanner}] ${data.message}`);
      } catch {}
    });

    eventSource.addEventListener("scan_result", (e) => {
      try {
        const data = JSON.parse(e.data) as {
          summary: { critical: number; high: number; medium: number; low: number };
          scanner: string;
        };
        scanSummary = data.summary;
        const s = data.summary;
        addLog(`Scan complete (${data.scanner}): ${s.critical}C ${s.high}H ${s.medium}M ${s.low}L`);
      } catch {}
    });

    eventSource.addEventListener("updated", (e) => {
      try {
        const data = JSON.parse((e as MessageEvent).data) as { message: string };
        addLog(data.message ?? ($language === "es"
          ? `${container.name} actualizado exitosamente`
          : `${container.name} updated successfully`));
      } catch {}
      status = "success";
      checkForUpdates();
      // SSE stays open — background scan results still incoming
    });

    eventSource.addEventListener("error", (e) => {
      if (status !== "success" && status !== "scanning") {
        try {
          const data = JSON.parse((e as MessageEvent).data) as { message: string };
          statusMessage = data.message;
          addLog(`Error: ${data.message}`);
        } catch {
          statusMessage = "Stream error";
          addLog("Stream error occurred");
        }
        status = "error";
        cleanup();
      }
    });

    eventSource.addEventListener("close", () => {
      cleanup();
      if (status === "streaming") status = "success";
      if (status === "success" || status === "scanning") {
        autoCloseTimer = setTimeout(onclose, 5000);
      }
    });

    eventSource.onerror = () => {
      if (status === "streaming") {
        status = "error";
        statusMessage = "Connection lost";
        addLog("Connection to server lost");
        cleanup();
      }
    };
  }

  function cleanup() {
    if (eventSource) {
      eventSource.close();
      eventSource = null;
    }
  }

  function handleClose() {
    cleanup();
    if (autoCloseTimer) clearTimeout(autoCloseTimer);
    onclose();
  }

  onMount(() => {
    openStream();
  });

  onDestroy(() => {
    cleanup();
    if (autoCloseTimer) clearTimeout(autoCloseTimer);
  });
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
  class="fixed inset-0 bg-black/60 z-[100] flex items-center justify-center p-4"
  onclick={handleClose}
>
  <div
    class="bg-background-secondary border border-border rounded-xl shadow-2xl w-full max-w-lg overflow-hidden"
    onclick={(e) => e.stopPropagation()}
  >
    <!-- Header -->
    <div class="flex items-center justify-between px-5 py-4 border-b border-border">
      <div class="flex items-center gap-3">
        <div class="p-2 bg-accent-orange/15 rounded-lg">
          <Download class="w-5 h-5 text-accent-orange" />
        </div>
        <div>
          <h3 class="font-semibold text-foreground">
            {$language === "es" ? "Actualizando contenedor" : "Updating container"}
          </h3>
          <p class="text-xs text-foreground-muted">{container.name}</p>
        </div>
      </div>
      <button class="btn-icon hover:bg-background-tertiary" onclick={handleClose}>
        <X class="w-5 h-5" />
      </button>
    </div>

    <div class="px-5 py-4">
      <!-- Status row -->
      <div class="flex items-center gap-3 mb-4">
        {#if status === "streaming"}
          <Loader2 class="w-5 h-5 text-primary animate-spin flex-shrink-0" />
          <span class="text-sm text-foreground">
            {$language === "es" ? "Actualizando..." : "Updating..."}
          </span>
        {:else if status === "success"}
          <CheckCircle2 class="w-5 h-5 text-running flex-shrink-0" />
          <span class="text-sm text-running">
            {$language === "es" ? "Actualización completada" : "Update completed"}
          </span>
        {:else if status === "scanning"}
          <CheckCircle2 class="w-5 h-5 text-running flex-shrink-0" />
          <span class="text-sm text-running">
            {$language === "es" ? "Actualizado" : "Updated"}
          </span>
        {:else if status === "error"}
          <AlertCircle class="w-5 h-5 text-accent-red flex-shrink-0" />
          <span class="text-sm text-accent-red">
            {$language === "es" ? "Error" : "Error"}: {statusMessage}
          </span>
        {/if}
      </div>

      <!-- Progress bar -->
      {#if status === "streaming"}
        <div class="h-1.5 bg-background-tertiary rounded-full overflow-hidden mb-4">
          <div class="h-full bg-primary rounded-full animate-pulse" style="width: 60%"></div>
        </div>
      {:else if status === "success" || status === "scanning"}
        <div class="h-1.5 bg-background-tertiary rounded-full overflow-hidden mb-4">
          <div class="h-full bg-running rounded-full" style="width: 100%"></div>
        </div>
      {:else if status === "error"}
        <div class="h-1.5 bg-background-tertiary rounded-full overflow-hidden mb-4">
          <div class="h-full bg-accent-red rounded-full" style="width: 100%"></div>
        </div>
      {/if}

      <!-- Background scan indicator -->
      {#if status === "scanning" && !scanSummary}
        <div class="flex items-center gap-2 mb-4 text-xs text-foreground-muted">
          <ScanSearch class="w-3.5 h-3.5 animate-pulse flex-shrink-0" />
          <span>{$language === "es" ? "Escaneando vulnerabilidades en segundo plano..." : "Scanning for vulnerabilities in background..."}</span>
        </div>
      {/if}

      <!-- Scan summary badges (informational) -->
      {#if scanSummary}
        <div class="flex items-center gap-2 mb-4 flex-wrap">
          <Shield class="w-4 h-4 text-foreground-muted flex-shrink-0" />
          <span class="text-xs text-foreground-muted">
            {$language === "es" ? "Resultado del escaneo:" : "Scan result:"}
          </span>
          {#if scanSummary.critical > 0}
            <span class="px-2 py-0.5 text-xs font-semibold bg-red-500/15 text-red-400 rounded-full border border-red-500/30">
              {scanSummary.critical}C
            </span>
          {/if}
          {#if scanSummary.high > 0}
            <span class="px-2 py-0.5 text-xs font-semibold bg-orange-500/15 text-orange-400 rounded-full border border-orange-500/30">
              {scanSummary.high}H
            </span>
          {/if}
          {#if scanSummary.medium > 0}
            <span class="px-2 py-0.5 text-xs font-semibold bg-yellow-500/15 text-yellow-400 rounded-full border border-yellow-500/30">
              {scanSummary.medium}M
            </span>
          {/if}
          {#if scanSummary.low > 0}
            <span class="px-2 py-0.5 text-xs font-semibold bg-blue-500/15 text-blue-400 rounded-full border border-blue-500/30">
              {scanSummary.low}L
            </span>
          {/if}
          {#if scanSummary.critical === 0 && scanSummary.high === 0 && scanSummary.medium === 0 && scanSummary.low === 0}
            <span class="px-2 py-0.5 text-xs font-semibold bg-running/15 text-running rounded-full border border-running/30">
              {$language === "es" ? "Sin vulnerabilidades" : "Clean"}
            </span>
          {/if}
        </div>
      {/if}

      <!-- Log output -->
      {#if showLogs}
        <div class="bg-background rounded-lg border border-border p-3 max-h-48 overflow-y-auto font-mono text-xs">
          {#each logLines as line}
            <div class="text-foreground-muted leading-relaxed">{line}</div>
          {/each}
          {#if logLines.length === 0}
            <div class="text-foreground-muted italic">
              {$language === "es" ? "Esperando actividad..." : "Waiting for activity..."}
            </div>
          {/if}
        </div>
      {/if}
    </div>

    <!-- Footer -->
    <div class="flex items-center justify-between px-5 py-3 border-t border-border bg-background-tertiary/30">
      <button
        class="text-xs text-foreground-muted hover:text-foreground"
        onclick={() => (showLogs = !showLogs)}
      >
        {showLogs
          ? $language === "es" ? "Ocultar logs" : "Hide logs"
          : $language === "es" ? "Mostrar logs" : "Show logs"}
      </button>
      <button
        class="px-4 py-1.5 text-sm bg-background-tertiary hover:bg-background-tertiary/80 text-foreground rounded-lg transition-colors"
        onclick={handleClose}
      >
        {$language === "es" ? "Cerrar" : "Close"}
      </button>
    </div>
  </div>
</div>
