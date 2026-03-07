<script lang="ts">
  import { onMount } from "svelte";
  import {
    Server, Plus, RefreshCw, Wifi, WifiOff, Pencil, Trash2,
    Loader2, Globe, Activity, ShieldCheck, Unplug,
    Check, X, Zap
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
    if (Array.isArray(env.labels)) return env.labels;
    return (env.labels as unknown as string).split(",").map((l: string) => l.trim()).filter(Boolean);
  }

  function getConnectionLabel(env: EnvironmentResponse): string {
    if (!env.connectionType || env.connectionType === "socket") {
      return env.socketPath || "/var/run/docker.sock";
    }
    const host = env.host || "";
    const port = env.port || 2375;
    return host ? `${host}:${port}` : "";
  }

  function getTestResult(id: string): TestResult | "testing" | null {
    return testResults[id] ?? null;
  }

  onMount(() => {
    fetchEnvironments().then(() => testAll());
  });
</script>

<div class="space-y-5">
  <!-- Page header -->
  <div class="flex items-center justify-between">
    <div>
      <h1 class="text-xl font-semibold text-foreground">
        {$language === "es" ? "Entornos" : "Environments"}
      </h1>
      <p class="text-sm text-foreground-muted mt-0.5">
        {$language === "es"
          ? `${environments.length} entorno${environments.length !== 1 ? 's' : ''} configurado${environments.length !== 1 ? 's' : ''}`
          : `${environments.length} environment${environments.length !== 1 ? 's' : ''} configured`}
      </p>
    </div>
    <div class="flex items-center gap-2">
      <button
        class="flex items-center gap-1.5 px-3 py-1.5 text-sm bg-background-secondary border border-border text-foreground rounded-lg hover:bg-background-tertiary transition-colors disabled:opacity-50"
        onclick={testAll}
        disabled={testingAll || environments.length === 0}
        title={$language === "es" ? "Probar todas las conexiones" : "Test all connections"}
      >
        {#if testingAll}
          <RefreshCw class="w-3.5 h-3.5 animate-spin" />
        {:else}
          <Zap class="w-3.5 h-3.5" />
        {/if}
        {$language === "es" ? "Probar todos" : "Test all"}
      </button>
      <button
        class="flex items-center gap-1.5 px-3 py-1.5 text-sm bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors"
        onclick={() => { editingEnv = null; showModal = true; }}
      >
        <Plus class="w-4 h-4" />
        {$language === "es" ? "Agregar entorno" : "Add environment"}
      </button>
    </div>
  </div>

  <!-- Environment cards -->
  {#if loading && environments.length === 0}
    <div class="flex items-center justify-center py-20">
      <Loader2 class="w-6 h-6 text-primary animate-spin" />
    </div>
  {:else if environments.length === 0}
    <div class="text-center py-20 border border-dashed border-border rounded-xl">
      <Server class="w-10 h-10 text-foreground-muted/30 mx-auto mb-3" />
      <p class="text-sm font-medium text-foreground-muted">
        {$language === "es" ? "Sin entornos configurados" : "No environments configured"}
      </p>
      <p class="text-xs text-foreground-muted/60 mt-1">
        {$language === "es" ? "Agrega un host Docker para comenzar" : "Add a Docker host to get started"}
      </p>
    </div>
  {:else}
    <div class="space-y-2">
      {#each environments as env (env.id)}
        {@const result = getTestResult(env.id)}
        {@const isTesting = result === "testing"}
        {@const testResult = result !== null && result !== "testing" ? result as TestResult : null}
        {@const ps = pruneStatus[env.id]}
        {@const labels = getLabels(env)}
        {@const connectionLabel = getConnectionLabel(env)}
        {@const isSocket = !env.connectionType || env.connectionType === "socket"}

        <div class="bg-background-secondary border border-border rounded-xl px-5 py-4 hover:border-primary/20 transition-colors">
          <div class="flex items-start gap-4">
            <!-- Connection type icon -->
            <div class="mt-0.5 p-2 rounded-lg bg-background-tertiary/60 flex-shrink-0">
              {#if isSocket}
                <Unplug class="w-4 h-4 text-cyan-400" />
              {:else}
                <Globe class="w-4 h-4 text-primary" />
              {/if}
            </div>

            <!-- Main info -->
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2 flex-wrap">
                <span class="font-semibold text-foreground">{env.name}</span>
                <span class="text-xs px-2 py-0.5 rounded-full bg-background-tertiary text-foreground-muted border border-border">
                  {isSocket
                    ? ($language === "es" ? "Socket Unix" : "Unix Socket")
                    : "TCP"}
                </span>
                {#if isSocket}
                  <span class="text-xs px-2 py-0.5 rounded-full bg-background-tertiary text-foreground-muted/60 border border-border/50">Local</span>
                {/if}
              </div>

              <p class="text-sm text-foreground-muted mt-1 font-mono truncate">{connectionLabel}</p>

              {#if labels.length > 0}
                <div class="flex gap-1 mt-2 flex-wrap">
                  {#each labels as label}
                    <span class="text-xs px-2 py-0.5 bg-primary/10 text-primary rounded-full font-medium">{label}</span>
                  {/each}
                </div>
              {/if}

              {#if env.autoUpdate || env.vulnScanning || env.eventTracking || env.imagePrune}
                <div class="flex items-center gap-3 mt-2">
                  {#if env.autoUpdate}
                    <span class="flex items-center gap-1 text-xs text-green-400">
                      <RefreshCw class="w-3 h-3" />
                      Auto-update
                    </span>
                  {/if}
                  {#if env.vulnScanning}
                    <span class="flex items-center gap-1 text-xs text-green-400">
                      <ShieldCheck class="w-3 h-3" />
                      CVE scan
                    </span>
                  {/if}
                  {#if env.eventTracking}
                    <span class="flex items-center gap-1 text-xs text-amber-400">
                      <Activity class="w-3 h-3" />
                      {$language === "es" ? "Eventos" : "Events"}
                    </span>
                  {/if}
                  {#if env.imagePrune}
                    <span class="flex items-center gap-1 text-xs text-amber-400">
                      <Trash2 class="w-3 h-3" />
                      Prune
                    </span>
                  {/if}
                </div>
              {/if}
            </div>

            <!-- Status -->
            <div class="text-right min-w-[130px] flex-shrink-0">
              {#if isTesting}
                <div class="flex items-center gap-1.5 text-foreground-muted text-xs justify-end">
                  <RefreshCw class="w-3.5 h-3.5 animate-spin" />
                  Testing...
                </div>
              {:else if testResult}
                {#if testResult.success}
                  <div class="flex items-center gap-1.5 text-green-400 text-xs justify-end">
                    <Wifi class="w-3.5 h-3.5" />
                    Connected
                  </div>
                  {#if testResult.dockerVersion}
                    <p class="text-[11px] text-foreground-muted mt-0.5">
                      Docker {testResult.dockerVersion}{testResult.containers !== undefined ? ` · ${testResult.containers} containers` : ''}
                    </p>
                  {/if}
                {:else}
                  <div class="flex items-center gap-1.5 text-red-400 text-xs justify-end" title={testResult.error}>
                    <WifiOff class="w-3.5 h-3.5" />
                    Failed
                  </div>
                {/if}
              {:else}
                <span class="text-xs text-foreground-muted/40">Not tested</span>
              {/if}
            </div>

            <!-- Actions -->
            <div class="flex items-center gap-1 flex-shrink-0">
              <button
                class="p-2 rounded-lg hover:bg-background-tertiary transition-colors text-foreground-muted hover:text-foreground"
                onclick={() => testConnection(env.id)}
                disabled={isTesting}
                title={$language === "es" ? "Probar conexión" : "Test connection"}
              >
                {#if isTesting}
                  <RefreshCw class="w-4 h-4 animate-spin" />
                {:else}
                  <Zap class="w-4 h-4" />
                {/if}
              </button>

              <button
                class="p-2 rounded-lg hover:bg-background-tertiary transition-colors text-foreground-muted hover:text-foreground"
                onclick={() => { editingEnv = { ...env }; showModal = true; }}
                title={$language === "es" ? "Editar" : "Edit"}
              >
                <Pencil class="w-4 h-4" />
              </button>

              {#if confirmPrune === env.id}
                <button class="p-2 rounded-lg hover:bg-green-500/10 transition-colors" onclick={() => pruneSystem(env.id)} disabled={ps === "pruning"}>
                  <Check class="w-4 h-4 text-green-400" />
                </button>
                <button class="p-2 rounded-lg hover:bg-background-tertiary transition-colors" onclick={() => (confirmPrune = null)}>
                  <X class="w-4 h-4 text-foreground-muted" />
                </button>
              {:else}
                <button
                  class="p-2 rounded-lg hover:bg-background-tertiary transition-colors text-foreground-muted hover:text-amber-400"
                  onclick={() => (confirmPrune = env.id)}
                  title={$language === "es" ? "Limpiar sistema" : "Prune system"}
                  disabled={ps === "pruning"}
                >
                  {#if ps === "pruning"}
                    <RefreshCw class="w-4 h-4 animate-spin" />
                  {:else if ps === "success"}
                    <Check class="w-4 h-4 text-green-400" />
                  {:else}
                    <Trash2 class="w-4 h-4" />
                  {/if}
                </button>
              {/if}

              {#if confirmDelete === env.id}
                <button class="p-2 rounded-lg hover:bg-red-500/10 transition-colors" onclick={() => deleteEnvironment(env.id)}>
                  <Check class="w-4 h-4 text-red-400" />
                </button>
                <button class="p-2 rounded-lg hover:bg-background-tertiary transition-colors" onclick={() => (confirmDelete = null)}>
                  <X class="w-4 h-4 text-foreground-muted" />
                </button>
              {:else}
                <button
                  class="p-2 rounded-lg hover:bg-red-500/10 transition-colors text-foreground-muted hover:text-red-400"
                  onclick={() => (confirmDelete = env.id)}
                  title={$language === "es" ? "Eliminar" : "Delete"}
                >
                  <Trash2 class="w-4 h-4" />
                </button>
              {/if}
            </div>
          </div>
        </div>
      {/each}
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
