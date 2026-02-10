<script lang="ts">
  import { onMount } from "svelte";
  import {
    Server,
    Plus,
    RefreshCw,
    Wifi,
    WifiOff,
    Pencil,
    Trash2,
    Zap,
    Loader2,
    HardDrive,
    Globe,
  } from "lucide-svelte";
  import { language } from "$lib/stores/docker";
  import { settingsText } from "$lib/settings";
  import { API_BASE } from "$lib/api/docker";
  import EnvironmentModal from "$lib/components/EnvironmentModal.svelte";
  import type { EnvironmentData } from "$lib/components/EnvironmentModal.svelte";

  let st = $derived(settingsText[$language]);

  interface EnvironmentResponse extends EnvironmentData {
    status: string;
    dockerVersion: string;
  }

  let environments = $state<EnvironmentResponse[]>([]);
  let loading = $state(true);
  let showModal = $state(false);
  let editingEnv = $state<EnvironmentData | null>(null);
  let testingId = $state<string | null>(null);
  let testResult = $state<Record<string, { success: boolean; message: string; dockerVersion?: string; os?: string; containers?: number }>>({});

  function getAuthHeaders(): Record<string, string> {
    const token = typeof localStorage !== "undefined" ? localStorage.getItem("auth_access_token") : null;
    const headers: Record<string, string> = { "Content-Type": "application/json" };
    if (token) headers["Authorization"] = `Bearer ${token}`;
    return headers;
  }

  async function fetchEnvironments() {
    loading = true;
    try {
      const res = await fetch(`${API_BASE}/api/environments`, {
        headers: getAuthHeaders(),
      });
      if (res.ok) {
        environments = await res.json();
      }
    } catch (e) {
      console.error("Failed to fetch environments:", e);
    }
    loading = false;
  }

  async function testConnection(id: string) {
    testingId = id;
    try {
      const res = await fetch(`${API_BASE}/api/environments/${id}/test`, {
        method: "POST",
        headers: getAuthHeaders(),
      });
      const data = await res.json();
      testResult = {
        ...testResult,
        [id]: {
          success: data.success,
          message: data.success
            ? `Docker ${data.dockerVersion}`
            : data.error,
          dockerVersion: data.dockerVersion,
          os: data.os,
          containers: data.containers,
        },
      };
    } catch (e) {
      testResult = {
        ...testResult,
        [id]: { success: false, message: "Connection failed" },
      };
    }
    testingId = null;
  }

  async function testAll() {
    for (const env of environments) {
      await testConnection(env.id);
    }
  }

  async function handleSave(env: EnvironmentData) {
    const isEdit = editingEnv !== null;
    const url = isEdit
      ? `${API_BASE}/api/environments/${env.id}`
      : `${API_BASE}/api/environments`;
    const method = isEdit ? "PUT" : "POST";

    try {
      const res = await fetch(url, {
        method,
        headers: getAuthHeaders(),
        body: JSON.stringify(env),
      });
      if (res.ok) {
        showModal = false;
        editingEnv = null;
        await fetchEnvironments();
      }
    } catch (e) {
      console.error("Failed to save environment:", e);
    }
  }

  async function handleDelete(id: string) {
    if (!confirm($language === "es" ? "\u00bfEliminar este entorno?" : "Delete this environment?")) return;
    try {
      const res = await fetch(`${API_BASE}/api/environments/${id}`, {
        method: "DELETE",
        headers: getAuthHeaders(),
      });
      if (res.ok) {
        await fetchEnvironments();
      }
    } catch (e) {
      console.error("Failed to delete environment:", e);
    }
  }

  function openEdit(env: EnvironmentResponse) {
    editingEnv = { ...env };
    showModal = true;
  }

  function openAdd() {
    editingEnv = null;
    showModal = true;
  }

  function getDockerVersion(env: EnvironmentResponse): string {
    const test = testResult[env.id];
    if (test?.dockerVersion) return test.dockerVersion;
    return env.dockerVersion || "-";
  }

  onMount(() => {
    fetchEnvironments().then(() => {
      testAll();
    });
  });
</script>

<div class="p-6 space-y-6 max-w-4xl mx-auto">
  <!-- Header -->
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-3">
      <div class="p-2.5 bg-primary/15 rounded-xl">
        <Server class="w-6 h-6 text-primary" />
      </div>
      <div>
        <h2 class="text-lg font-semibold text-foreground">{st.environments}</h2>
        <p class="text-sm text-foreground-muted">{st.environmentsDesc}</p>
      </div>
    </div>
    <div class="flex items-center gap-2">
      <button
        class="flex items-center gap-1.5 px-3 py-2 text-sm bg-background-tertiary text-foreground rounded-lg hover:bg-background-tertiary/80 transition-colors"
        onclick={testAll}
      >
        <Zap class="w-4 h-4" />
        {st.testAll}
      </button>
      <button
        class="flex items-center gap-1.5 px-3 py-2 text-sm bg-background-tertiary text-foreground rounded-lg hover:bg-background-tertiary/80 transition-colors"
        onclick={fetchEnvironments}
      >
        <RefreshCw class="w-4 h-4" />
      </button>
      <button
        class="flex items-center gap-1.5 px-3 py-2 text-sm bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors"
        onclick={openAdd}
      >
        <Plus class="w-4 h-4" />
        {st.addEnvironment}
      </button>
    </div>
  </div>

  <!-- Cards -->
  {#if loading}
    <div class="flex items-center justify-center py-16">
      <Loader2 class="w-6 h-6 text-primary animate-spin" />
    </div>
  {:else if environments.length === 0}
    <div class="text-center py-16">
      <Server class="w-12 h-12 text-foreground-muted/30 mx-auto mb-3" />
      <p class="text-foreground-muted">{$language === "es" ? "No hay entornos configurados" : "No environments configured"}</p>
    </div>
  {:else}
    <div class="grid gap-4">
      {#each environments as env}
        <div class="bg-background-secondary border border-border rounded-xl p-5 hover:border-primary/30 transition-colors">
          <!-- Top Row: Name, Status, Actions -->
          <div class="flex items-start justify-between mb-4">
            <div class="flex items-center gap-3">
              <div class="p-2 rounded-lg {env.status === 'online' ? 'bg-running/15' : 'bg-stopped/15'}">
                <Server class="w-5 h-5 {env.status === 'online' ? 'text-running' : 'text-stopped'}" />
              </div>
              <div>
                <div class="flex items-center gap-2">
                  <h3 class="font-semibold text-foreground">{env.name}</h3>
                  <span class="text-xs px-2 py-0.5 rounded-full {env.status === 'online' ? 'bg-running/15 text-running' : 'bg-stopped/15 text-stopped'} flex items-center gap-1">
                    {#if env.status === "online"}
                      <Wifi class="w-3 h-3" />
                    {:else}
                      <WifiOff class="w-3 h-3" />
                    {/if}
                    {env.status}
                  </span>
                </div>
                <p class="text-xs text-foreground-muted mt-0.5">ID: {env.id}</p>
              </div>
            </div>
            <div class="flex items-center gap-1">
              <button
                class="p-2 rounded-lg hover:bg-primary/15 hover:text-primary transition-colors"
                onclick={() => testConnection(env.id)}
                title={st.testConnection}
              >
                {#if testingId === env.id}
                  <Loader2 class="w-4 h-4 animate-spin" />
                {:else}
                  <Zap class="w-4 h-4 text-foreground-muted" />
                {/if}
              </button>
              <button
                class="p-2 rounded-lg hover:bg-background-tertiary transition-colors"
                onclick={() => openEdit(env)}
                title={st.editEnvironment}
              >
                <Pencil class="w-4 h-4 text-foreground-muted" />
              </button>
              <button
                class="p-2 rounded-lg hover:bg-stopped/15 hover:text-stopped transition-colors"
                onclick={() => handleDelete(env.id)}
                title={st.deleteEnvironment}
              >
                <Trash2 class="w-4 h-4 text-foreground-muted" />
              </button>
            </div>
          </div>

          <!-- Info Grid -->
          <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
            <!-- Connection -->
            <div class="space-y-1">
              <p class="text-[10px] uppercase tracking-wider text-foreground-muted font-semibold">
                {$language === "es" ? "Conexi\u00f3n" : "Connection"}
              </p>
              <div class="flex items-center gap-1.5">
                <Globe class="w-3.5 h-3.5 text-foreground-muted" />
                <p class="text-xs text-foreground font-mono truncate" title={env.address}>{env.address}</p>
              </div>
              <p class="text-[10px] text-foreground-muted">{env.connectionType === "socket" ? "Unix Socket" : `TCP (${env.protocol?.toUpperCase() || "HTTP"})`}</p>
            </div>

            <!-- Docker Version -->
            <div class="space-y-1">
              <p class="text-[10px] uppercase tracking-wider text-foreground-muted font-semibold">Docker</p>
              <div class="flex items-center gap-1.5">
                <HardDrive class="w-3.5 h-3.5 text-foreground-muted" />
                <p class="text-xs text-foreground font-mono">{getDockerVersion(env)}</p>
              </div>
              {#if testResult[env.id]?.containers !== undefined}
                <p class="text-[10px] text-foreground-muted">{testResult[env.id].containers} containers</p>
              {/if}
            </div>

            <!-- Labels -->
            <div class="space-y-1">
              <p class="text-[10px] uppercase tracking-wider text-foreground-muted font-semibold">Labels</p>
              {#if env.labels}
                <div class="flex flex-wrap gap-1">
                  {#each env.labels.split(",").map(l => l.trim()).filter(Boolean) as label}
                    <span class="text-[10px] px-1.5 py-0.5 bg-primary/10 text-primary rounded-full">{label}</span>
                  {/each}
                </div>
              {:else}
                <p class="text-xs text-foreground-muted">-</p>
              {/if}
            </div>

            <!-- Features -->
            <div class="space-y-1">
              <p class="text-[10px] uppercase tracking-wider text-foreground-muted font-semibold">
                {$language === "es" ? "Caracter\u00edsticas" : "Features"}
              </p>
              <div class="flex flex-wrap gap-1">
                {#if env.autoUpdate}
                  <span class="text-[10px] px-1.5 py-0.5 bg-accent-cyan/10 text-accent-cyan rounded-full">Auto-update</span>
                {/if}
                {#if env.eventTracking}
                  <span class="text-[10px] px-1.5 py-0.5 bg-accent-purple/10 text-accent-purple rounded-full">Events</span>
                {/if}
                {#if env.vulnScanning}
                  <span class="text-[10px] px-1.5 py-0.5 bg-accent-orange/10 text-accent-orange rounded-full">Scan</span>
                {/if}
                {#if !env.autoUpdate && !env.eventTracking && !env.vulnScanning}
                  <p class="text-xs text-foreground-muted">-</p>
                {/if}
              </div>
            </div>
          </div>

          <!-- Test Result -->
          {#if testResult[env.id]}
            <div class="mt-3 pt-3 border-t border-border/50">
              <p class="text-xs {testResult[env.id].success ? 'text-running' : 'text-accent-red'}">
                {testResult[env.id].message}
                {#if testResult[env.id].os}
                  <span class="text-foreground-muted ml-2">({testResult[env.id].os})</span>
                {/if}
              </p>
            </div>
          {/if}
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
