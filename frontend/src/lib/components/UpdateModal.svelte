<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { X, Loader2, CheckCircle2, AlertCircle, Download } from "lucide-svelte";
  import type { Container } from "$lib/api/docker";
  import { triggerContainerUpdate } from "$lib/api/docker";
  import { language, checkForUpdates } from "$lib/stores/docker";
  import { API_BASE } from "$lib/api/docker";

  let {
    container,
    onclose,
  }: {
    container: Container;
    onclose: () => void;
  } = $props();

  type UpdateStatus = "idle" | "triggering" | "waiting" | "success" | "error";
  let status = $state<UpdateStatus>("idle");
  let statusMessage = $state("");
  let logLines = $state<string[]>([]);
  let showLogs = $state(true);
  let pollTimer: ReturnType<typeof setInterval> | null = null;
  let autoCloseTimer: ReturnType<typeof setTimeout> | null = null;

  let statusText = $derived({
    idle: $language === "es" ? "Listo para actualizar" : "Ready to update",
    triggering:
      $language === "es"
        ? "Activando Watchtower..."
        : "Triggering Watchtower...",
    waiting:
      $language === "es"
        ? "Esperando reinicio del contenedor..."
        : "Waiting for container restart...",
    success:
      $language === "es"
        ? "Actualización completada"
        : "Update completed",
    error: $language === "es" ? "Error en la actualización" : "Update failed",
  });

  function addLog(msg: string) {
    const ts = new Date().toLocaleTimeString();
    logLines = [...logLines, `[${ts}] ${msg}`];
  }

  async function startUpdate() {
    status = "triggering";
    addLog(
      $language === "es"
        ? `Activando actualización para ${container.name}...`
        : `Triggering update for ${container.name}...`,
    );

    try {
      const result = await triggerContainerUpdate(
        container.hostId,
        container.id,
      );
      addLog(result.message || "Watchtower triggered successfully");
      status = "waiting";
      addLog(
        $language === "es"
          ? "Monitoreando estado del contenedor..."
          : "Monitoring container state...",
      );

      // Poll container state every 3s for up to 120s
      let attempts = 0;
      const maxAttempts = 40;
      pollTimer = setInterval(async () => {
        attempts++;
        try {
          const token =
            typeof localStorage !== "undefined"
              ? localStorage.getItem("auth_access_token")
              : null;
          const headers: Record<string, string> = {
            "Content-Type": "application/json",
          };
          if (token) headers["Authorization"] = `Bearer ${token}`;

          const res = await fetch(`${API_BASE}/api/containers`, { headers });
          if (res.ok) {
            const containers = await res.json();
            const updated = containers.find(
              (c: Container) =>
                c.name === container.name && c.hostId === container.hostId,
            );
            if (updated && updated.state === "running") {
              // Container is running - check if it was recently created (restarted)
              const age = Date.now() / 1000 - updated.created;
              if (age < 120 || attempts > 10) {
                addLog(
                  $language === "es"
                    ? `${container.name} está ejecutándose con la nueva imagen`
                    : `${container.name} is running with updated image`,
                );
                status = "success";
                if (pollTimer) clearInterval(pollTimer);
                checkForUpdates();
                autoCloseTimer = setTimeout(onclose, 5000);
              }
            }
          }
        } catch {
          // Ignore poll errors
        }
        if (attempts >= maxAttempts) {
          addLog(
            $language === "es"
              ? "Tiempo de espera agotado. La actualización puede seguir en progreso."
              : "Polling timed out. Update may still be in progress.",
          );
          status = "success";
          if (pollTimer) clearInterval(pollTimer);
          checkForUpdates();
        }
      }, 3000);
    } catch (e) {
      const msg = e instanceof Error ? e.message : "Unknown error";
      addLog(`Error: ${msg}`);
      status = "error";
      statusMessage = msg;
    }
  }

  function cleanup() {
    if (pollTimer) clearInterval(pollTimer);
    if (autoCloseTimer) clearTimeout(autoCloseTimer);
  }

  function handleClose() {
    cleanup();
    onclose();
  }

  // Auto-start the update once on mount
  onMount(() => {
    startUpdate();
  });

  onDestroy(cleanup);
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
    <div
      class="flex items-center justify-between px-5 py-4 border-b border-border"
    >
      <div class="flex items-center gap-3">
        <div class="p-2 bg-accent-orange/15 rounded-lg">
          <Download class="w-5 h-5 text-accent-orange" />
        </div>
        <div>
          <h3 class="font-semibold text-foreground">
            {$language === "es"
              ? "Actualizando contenedor"
              : "Updating container"}
          </h3>
          <p class="text-xs text-foreground-muted">{container.name}</p>
        </div>
      </div>
      <button
        class="btn-icon hover:bg-background-tertiary"
        onclick={handleClose}
      >
        <X class="w-5 h-5" />
      </button>
    </div>

    <!-- Status -->
    <div class="px-5 py-4">
      <div class="flex items-center gap-3 mb-4">
        {#if status === "triggering" || status === "waiting"}
          <Loader2 class="w-5 h-5 text-primary animate-spin" />
          <span class="text-sm text-foreground">{statusText[status]}</span>
        {:else if status === "success"}
          <CheckCircle2 class="w-5 h-5 text-running" />
          <span class="text-sm text-running">{statusText.success}</span>
        {:else if status === "error"}
          <AlertCircle class="w-5 h-5 text-accent-red" />
          <span class="text-sm text-accent-red"
            >{statusText.error}: {statusMessage}</span
          >
        {:else}
          <span class="text-sm text-foreground-muted">{statusText.idle}</span>
        {/if}
      </div>

      <!-- Progress bar -->
      {#if status === "triggering" || status === "waiting"}
        <div
          class="h-1.5 bg-background-tertiary rounded-full overflow-hidden mb-4"
        >
          <div
            class="h-full bg-primary rounded-full animate-pulse"
            style="width: {status === 'triggering' ? '30%' : '70%'}"
          ></div>
        </div>
      {:else if status === "success"}
        <div
          class="h-1.5 bg-background-tertiary rounded-full overflow-hidden mb-4"
        >
          <div class="h-full bg-running rounded-full" style="width: 100%"></div>
        </div>
      {/if}

      <!-- Log output -->
      {#if showLogs}
        <div
          class="bg-background rounded-lg border border-border p-3 max-h-48 overflow-y-auto font-mono text-xs"
        >
          {#each logLines as line}
            <div class="text-foreground-muted leading-relaxed">{line}</div>
          {/each}
          {#if logLines.length === 0}
            <div class="text-foreground-muted italic">
              {$language === "es"
                ? "Esperando actividad..."
                : "Waiting for activity..."}
            </div>
          {/if}
        </div>
      {/if}
    </div>

    <!-- Footer -->
    <div
      class="flex items-center justify-between px-5 py-3 border-t border-border bg-background-tertiary/30"
    >
      <button
        class="text-xs text-foreground-muted hover:text-foreground"
        onclick={() => (showLogs = !showLogs)}
      >
        {showLogs
          ? $language === "es"
            ? "Ocultar logs"
            : "Hide logs"
          : $language === "es"
            ? "Mostrar logs"
            : "Show logs"}
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
