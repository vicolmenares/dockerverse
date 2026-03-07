<script lang="ts">
  import { onMount } from "svelte";
  import {
    Server, Plus, RefreshCw, Wifi, WifiOff, Pencil, Trash2,
    Loader2, Globe, Activity, ShieldCheck, Unplug,
    Check, X
  } from "lucide-svelte";
  import { language } from "$lib/stores/docker";
  import { API_BASE } from "$lib/api/docker";
  import EnvironmentModal from "$lib/components/EnvironmentModal.svelte";
  import type { EnvironmentData } from "$lib/components/EnvironmentModal.svelte";

  interface EnvironmentResponse extends EnvironmentData {
    status: string;
    dockerVersion: string;
  }

  interface TestResult {
    success: boolean;
    error?: string;
    dockerVersion?: string;
    os?: string;
    containers?: number;
  }

  let environments = $state<EnvironmentResponse[]>([]);
  let loading = $state(true);
  let showModal = $state(false);
  let editingEnv = $state<EnvironmentData | null>(null);
  let testResults = $state<Record<string, TestResult | "testing">>({});
  let testingAll = $state(false);
  let pruneStatus = $state<Record<string, "pruning" | "success" | "error" | null>>({});
  let confirmDelete = $state<string | null>(null);
  let confirmPrune = $state<string | null>(null);

  function getAuthHeaders(): Record<string, string> {
    const token = typeof localStorage !== "undefined" ? localStorage.getItem("auth_access_token") : null;
    const headers: Record<string, string> = { "Content-Type": "application/json" };
    if (token) headers["Authorization"] = `Bearer ${token}`;
    return headers;
  }

  async function fetchEnvironments() {
    loading = true;
    try {
      const res = await fetch(`${API_BASE}/api/environments`, { headers: getAuthHeaders() });
      if (res.ok) environments = await res.json();
    } catch (e) { console.error("Failed to fetch environments:", e); }
    loading = false;
  }

  async function testConnection(id: string) {
    testResults[id] = "testing";
    testResults = { ...testResults };
    try {
      const res = await fetch(`${API_BASE}/api/environments/${id}/test`, {
        method: "POST", headers: getAuthHeaders()
      });
      const data = await res.json();
      testResults[id] = {
        success: data.success,
        error: data.error,
        dockerVersion: data.dockerVersion,
        os: data.os,
        containers: data.containers,
      };
    } catch {
      testResults[id] = { success: false, error: "Connection failed" };
    }
    testResults = { ...testResults };
  }

  async function testAll() {
    if (testingAll) return;
    testingAll = true;
    for (const env of environments) {
      await testConnection(env.id);
    }
    testingAll = false;
  }

  async function pruneSystem(id: string) {
    pruneStatus[id] = "pruning";
    pruneStatus = { ...pruneStatus };
    confirmPrune = null;
    try {
      const res = await fetch(`${API_BASE}/api/prune/all?host=${id}`, {
        method: "POST", headers: getAuthHeaders()
      });
      pruneStatus[id] = res.ok ? "success" : "error";
    } catch {
      pruneStatus[id] = "error";
    }
    pruneStatus = { ...pruneStatus };
    setTimeout(() => { pruneStatus[id] = null; pruneStatus = { ...pruneStatus }; }, 3000);
  }

  async function deleteEnvironment(id: string) {
    confirmDelete = null;
    try {
      const res = await fetch(`${API_BASE}/api/environments/${id}`, {
        method: "DELETE", headers: getAuthHeaders()
      });
      if (res.ok) await fetchEnvironments();
    } catch (e) { console.error("Failed to delete environment:", e); }
  }

  async function handleSave(env: EnvironmentData) {
    const isEdit = editingEnv !== null;
    const url = isEdit ? `${API_BASE}/api/environments/${env.id}` : `${API_BASE}/api/environments`;
    try {
      const res = await fetch(url, {
        method: isEdit ? "PUT" : "POST",
        headers: getAuthHeaders(),
        body: JSON.stringify(env)
      });
      if (res.ok) {
        showModal = false;
        editingEnv = null;
        await fetchEnvironments();
        testAll();
      }
    } catch (e) { console.error("Failed to save environment:", e); }
  }

  function getLabels(env: EnvironmentResponse): string[] {
    if (!env.labels) return [];
    return env.labels.split(",").map(l => l.trim()).filter(Boolean);
  }

  onMount(() => {
    fetchEnvironments().then(() => testAll());
  });
</script>

<div class="p-6 space-y-4 max-w-6xl mx-auto">
  <!-- Header -->
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-3">
      <div class="p-2 bg-primary/15 rounded-lg">
        <Server class="w-5 h-5 text-primary" />
      </div>
      <div>
        <h2 class="text-base font-semibold text-foreground">
          {$language === "es" ? "Entornos" : "Environments"}
        </h2>
        <p class="text-xs text-foreground-muted">{environments.length} total</p>
      </div>
    </div>
    <div class="flex items-center gap-2">
      <button
        class="flex items-center gap-1.5 px-3 py-1.5 text-sm bg-background-tertiary text-foreground rounded-lg hover:bg-background-tertiary/80 transition-colors disabled:opacity-50"
        onclick={testAll}
        disabled={testingAll || environments.length === 0}
      >
        {#if testingAll}
          <RefreshCw class="w-4 h-4 animate-spin" />
        {:else}
          <Wifi class="w-4 h-4" />
        {/if}
        {$language === "es" ? "Probar todos" : "Test all"}
      </button>
      <button
        class="flex items-center gap-1.5 px-3 py-1.5 text-sm bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors"
        onclick={() => { editingEnv = null; showModal = true; }}
      >
        <Plus class="w-4 h-4" />
        {$language === "es" ? "Agregar" : "Add environment"}
      </button>
    </div>
  </div>

  <!-- Table -->
  {#if loading && environments.length === 0}
    <div class="flex items-center justify-center py-16">
      <Loader2 class="w-6 h-6 text-primary animate-spin" />
    </div>
  {:else if environments.length === 0}
    <div class="text-center py-16">
      <Server class="w-10 h-10 text-foreground-muted/30 mx-auto mb-3" />
      <p class="text-sm text-foreground-muted">
        {$language === "es" ? "Sin entornos configurados" : "No environments configured"}
      </p>
    </div>
  {:else}
    <div class="border border-border rounded-lg overflow-hidden">
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b border-border bg-background-secondary/50">
            <th class="px-4 py-2.5 text-left text-xs font-semibold text-foreground-muted uppercase tracking-wider w-48">
              {$language === "es" ? "Nombre" : "Name"}
            </th>
            <th class="px-4 py-2.5 text-left text-xs font-semibold text-foreground-muted uppercase tracking-wider">
              {$language === "es" ? "Conexión" : "Connection"}
            </th>
            <th class="px-4 py-2.5 text-left text-xs font-semibold text-foreground-muted uppercase tracking-wider w-32">
              Labels
            </th>
            <th class="px-4 py-2.5 text-left text-xs font-semibold text-foreground-muted uppercase tracking-wider w-28">
              {$language === "es" ? "Funciones" : "Features"}
            </th>
            <th class="px-4 py-2.5 text-left text-xs font-semibold text-foreground-muted uppercase tracking-wider w-44">
              Status
            </th>
            <th class="px-4 py-2.5 text-right text-xs font-semibold text-foreground-muted uppercase tracking-wider w-36">
              {$language === "es" ? "Acciones" : "Actions"}
            </th>
          </tr>
        </thead>
        <tbody>
          {#each environments as env (env.id)}
            {@const result = testResults[env.id]}
            {@const isTesting = result === "testing"}
            {@const ps = pruneStatus[env.id]}
            {@const labels = getLabels(env)}
            <tr class="border-b border-border/50 last:border-0 hover:bg-background-secondary/30 transition-colors">
              <!-- Name -->
              <td class="px-4 py-3">
                <div class="flex items-center gap-2">
                  {#if env.connectionType === "socket"}
                    <Unplug class="w-3.5 h-3.5 text-cyan-400 shrink-0" />
                  {:else}
                    <Globe class="w-3.5 h-3.5 text-primary shrink-0" />
                  {/if}
                  <span class="font-medium text-foreground truncate">{env.name}</span>
                </div>
              </td>

              <!-- Connection -->
              <td class="px-4 py-3">
                <span class="text-xs text-foreground-muted font-mono">{env.address}</span>
              </td>

              <!-- Labels -->
              <td class="px-4 py-3">
                <div class="flex flex-wrap gap-1">
                  {#if labels.length > 0}
                    {#each labels as label}
                      <span class="text-[10px] px-1.5 py-0.5 bg-primary/10 text-primary rounded-full font-medium">
                        {label}
                      </span>
                    {/each}
                  {:else}
                    <span class="text-foreground-muted text-xs">—</span>
                  {/if}
                </div>
              </td>

              <!-- Features -->
              <td class="px-4 py-3">
                <div class="flex items-center gap-1.5">
                  {#if env.autoUpdate}
                    <span title={$language === "es" ? "Auto-actualización" : "Auto-update"}>
                      <RefreshCw class="w-3.5 h-3.5 text-green-400" />
                    </span>
                  {/if}
                  {#if env.vulnScanning}
                    <span title={$language === "es" ? "Escaneo de vulnerabilidades" : "Vulnerability scanning"}>
                      <ShieldCheck class="w-3.5 h-3.5 text-green-400" />
                    </span>
                  {/if}
                  {#if env.eventTracking}
                    <span title={$language === "es" ? "Seguimiento de eventos" : "Event tracking"}>
                      <Activity class="w-3.5 h-3.5 text-amber-400" />
                    </span>
                  {/if}
                  {#if env.imagePrune}
                    <span title={$language === "es" ? "Limpieza de imágenes" : "Image prune"}>
                      <Trash2 class="w-3.5 h-3.5 text-amber-400" />
                    </span>
                  {/if}
                  {#if !env.autoUpdate && !env.vulnScanning && !env.eventTracking && !env.imagePrune}
                    <span class="text-foreground-muted text-xs">—</span>
                  {/if}
                </div>
              </td>

              <!-- Status -->
              <td class="px-4 py-3">
                {#if isTesting}
                  <div class="flex items-center gap-1.5 text-foreground-muted text-xs">
                    <RefreshCw class="w-3.5 h-3.5 animate-spin" />
                    <span>Testing...</span>
                  </div>
                {:else if result && result !== "testing"}
                  {#if (result as TestResult).success}
                    <div class="flex items-center gap-1.5 text-green-400 text-xs">
                      <Wifi class="w-3.5 h-3.5" />
                      <span>Connected</span>
                    </div>
                    {#if (result as TestResult).dockerVersion}
                      <p class="text-[10px] text-foreground-muted mt-0.5 ml-5">
                        Docker {(result as TestResult).dockerVersion}
                        {#if (result as TestResult).containers !== undefined}
                          · {(result as TestResult).containers} containers
                        {/if}
                      </p>
                    {/if}
                  {:else}
                    <div class="flex items-center gap-1.5 text-red-400 text-xs" title={(result as TestResult).error}>
                      <WifiOff class="w-3.5 h-3.5" />
                      <span class="truncate max-w-[120px]">Failed</span>
                    </div>
                  {/if}
                {:else}
                  <span class="text-xs text-foreground-muted">Not tested</span>
                {/if}
              </td>

              <!-- Actions -->
              <td class="px-4 py-3">
                <div class="flex items-center justify-end gap-1">
                  <!-- Test -->
                  <button
                    class="p-1.5 rounded hover:bg-background-tertiary transition-colors"
                    onclick={() => testConnection(env.id)}
                    disabled={isTesting}
                    title={$language === "es" ? "Probar conexión" : "Test connection"}
                  >
                    {#if isTesting}
                      <RefreshCw class="w-3.5 h-3.5 animate-spin text-foreground-muted" />
                    {:else}
                      <Wifi class="w-3.5 h-3.5 text-foreground-muted" />
                    {/if}
                  </button>

                  <!-- Edit -->
                  <button
                    class="p-1.5 rounded hover:bg-background-tertiary transition-colors"
                    onclick={() => { editingEnv = { ...env }; showModal = true; }}
                    title={$language === "es" ? "Editar" : "Edit"}
                  >
                    <Pencil class="w-3.5 h-3.5 text-foreground-muted" />
                  </button>

                  <!-- Prune system (with inline confirm) -->
                  {#if confirmPrune === env.id}
                    <div class="flex items-center gap-1">
                      <button
                        class="p-1.5 rounded hover:bg-green-500/15 transition-colors"
                        onclick={() => pruneSystem(env.id)}
                        title={$language === "es" ? "Confirmar limpieza" : "Confirm prune"}
                        disabled={ps === "pruning"}
                      >
                        <Check class="w-3.5 h-3.5 text-green-400" />
                      </button>
                      <button
                        class="p-1.5 rounded hover:bg-background-tertiary transition-colors"
                        onclick={() => (confirmPrune = null)}
                      >
                        <X class="w-3.5 h-3.5 text-foreground-muted" />
                      </button>
                    </div>
                  {:else}
                    <button
                      class="p-1.5 rounded hover:bg-background-tertiary transition-colors"
                      onclick={() => (confirmPrune = env.id)}
                      title={$language === "es" ? "Limpiar sistema" : "Prune system"}
                      disabled={ps === "pruning"}
                    >
                      {#if ps === "pruning"}
                        <RefreshCw class="w-3.5 h-3.5 animate-spin text-foreground-muted" />
                      {:else if ps === "success"}
                        <Check class="w-3.5 h-3.5 text-green-400" />
                      {:else}
                        <Trash2 class="w-3.5 h-3.5 text-foreground-muted" />
                      {/if}
                    </button>
                  {/if}

                  <!-- Delete (with inline confirm) -->
                  {#if confirmDelete === env.id}
                    <div class="flex items-center gap-1">
                      <button
                        class="p-1.5 rounded hover:bg-red-500/15 transition-colors"
                        onclick={() => deleteEnvironment(env.id)}
                        title={$language === "es" ? "Confirmar eliminación" : "Confirm delete"}
                      >
                        <Check class="w-3.5 h-3.5 text-red-400" />
                      </button>
                      <button
                        class="p-1.5 rounded hover:bg-background-tertiary transition-colors"
                        onclick={() => (confirmDelete = null)}
                      >
                        <X class="w-3.5 h-3.5 text-foreground-muted" />
                      </button>
                    </div>
                  {:else}
                    <button
                      class="p-1.5 rounded hover:bg-red-500/15 transition-colors"
                      onclick={() => (confirmDelete = env.id)}
                      title={$language === "es" ? "Eliminar" : "Delete"}
                    >
                      <Trash2 class="w-3.5 h-3.5 text-foreground-muted" />
                    </button>
                  {/if}
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>

{#if showModal}
  <EnvironmentModal
    environment={editingEnv}
    onclose={() => { showModal = false; editingEnv = null; }}
    onsave={handleSave}
  />
{/if}
