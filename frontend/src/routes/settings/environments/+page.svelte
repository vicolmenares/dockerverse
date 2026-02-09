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
  let testResult = $state<Record<string, { success: boolean; message: string }>>({});

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
            ? `Docker ${data.dockerVersion} - ${data.containers} containers`
            : data.error,
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
    if (!confirm($language === "es" ? "¿Eliminar este entorno?" : "Delete this environment?")) return;
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

  onMount(fetchEnvironments);
</script>

<div class="p-4 space-y-6">
  <!-- Header -->
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-3">
      <Server class="w-6 h-6 text-primary" />
      <div>
        <h2 class="text-lg font-semibold text-foreground">{st.environments}</h2>
        <p class="text-sm text-foreground-muted">{st.environmentsDesc}</p>
      </div>
    </div>
    <div class="flex items-center gap-2">
      <button
        class="flex items-center gap-1.5 px-3 py-1.5 text-sm bg-background-tertiary text-foreground rounded-lg hover:bg-background-tertiary/80 transition-colors"
        onclick={testAll}
      >
        <Zap class="w-4 h-4" />
        {st.testAll}
      </button>
      <button
        class="flex items-center gap-1.5 px-3 py-1.5 text-sm bg-background-tertiary text-foreground rounded-lg hover:bg-background-tertiary/80 transition-colors"
        onclick={fetchEnvironments}
      >
        <RefreshCw class="w-4 h-4" />
      </button>
      <button
        class="flex items-center gap-1.5 px-3 py-1.5 text-sm bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors"
        onclick={openAdd}
      >
        <Plus class="w-4 h-4" />
        {st.addEnvironment}
      </button>
    </div>
  </div>

  <!-- Table -->
  {#if loading}
    <div class="flex items-center justify-center py-12">
      <Loader2 class="w-6 h-6 text-primary animate-spin" />
    </div>
  {:else if environments.length === 0}
    <div class="text-center py-12">
      <Server class="w-12 h-12 text-foreground-muted/30 mx-auto mb-3" />
      <p class="text-foreground-muted">{$language === "es" ? "No hay entornos configurados" : "No environments configured"}</p>
    </div>
  {:else}
    <div class="overflow-x-auto">
      <table class="w-full">
        <thead>
          <tr class="border-b border-border">
            <th class="text-left py-3 px-4 text-xs font-semibold text-foreground-muted uppercase tracking-wider">{$language === "es" ? "Nombre" : "Name"}</th>
            <th class="text-left py-3 px-4 text-xs font-semibold text-foreground-muted uppercase tracking-wider">{$language === "es" ? "Conexión" : "Connection"}</th>
            <th class="text-left py-3 px-4 text-xs font-semibold text-foreground-muted uppercase tracking-wider">Labels</th>
            <th class="text-left py-3 px-4 text-xs font-semibold text-foreground-muted uppercase tracking-wider">Status</th>
            <th class="text-left py-3 px-4 text-xs font-semibold text-foreground-muted uppercase tracking-wider">Docker</th>
            <th class="text-right py-3 px-4 text-xs font-semibold text-foreground-muted uppercase tracking-wider">{$language === "es" ? "Acciones" : "Actions"}</th>
          </tr>
        </thead>
        <tbody>
          {#each environments as env}
            <tr class="border-b border-border/50 hover:bg-background-tertiary/30 transition-colors">
              <td class="py-3 px-4">
                <div class="flex items-center gap-2">
                  <Server class="w-4 h-4 text-foreground-muted" />
                  <div>
                    <p class="text-sm font-medium text-foreground">{env.name}</p>
                    <p class="text-xs text-foreground-muted">{env.id}</p>
                  </div>
                </div>
              </td>
              <td class="py-3 px-4">
                <p class="text-xs font-mono text-foreground-muted truncate max-w-[200px]" title={env.address}>{env.address}</p>
                <p class="text-[10px] text-foreground-muted">{env.connectionType === "socket" ? "Unix Socket" : "TCP"}</p>
              </td>
              <td class="py-3 px-4">
                {#if env.labels}
                  <div class="flex flex-wrap gap-1">
                    {#each env.labels.split(",").map(l => l.trim()).filter(Boolean) as label}
                      <span class="text-[10px] px-1.5 py-0.5 bg-primary/10 text-primary rounded-full">{label}</span>
                    {/each}
                  </div>
                {:else}
                  <span class="text-xs text-foreground-muted">-</span>
                {/if}
              </td>
              <td class="py-3 px-4">
                <span class="flex items-center gap-1.5 text-xs {env.status === 'online' ? 'text-running' : 'text-stopped'}">
                  {#if env.status === "online"}
                    <Wifi class="w-3.5 h-3.5" />
                  {:else}
                    <WifiOff class="w-3.5 h-3.5" />
                  {/if}
                  {env.status}
                </span>
                {#if testResult[env.id]}
                  <p class="text-[10px] mt-0.5 {testResult[env.id].success ? 'text-running' : 'text-accent-red'}">
                    {testResult[env.id].message}
                  </p>
                {/if}
              </td>
              <td class="py-3 px-4">
                <span class="text-xs text-foreground-muted">{env.dockerVersion || "-"}</span>
              </td>
              <td class="py-3 px-4">
                <div class="flex items-center justify-end gap-1">
                  <button
                    class="btn-icon hover:bg-primary/20 hover:text-primary"
                    onclick={() => testConnection(env.id)}
                    title={st.testConnection}
                  >
                    {#if testingId === env.id}
                      <Loader2 class="w-4 h-4 animate-spin" />
                    {:else}
                      <Zap class="w-4 h-4" />
                    {/if}
                  </button>
                  <button
                    class="btn-icon hover:bg-background-tertiary"
                    onclick={() => openEdit(env)}
                    title={st.editEnvironment}
                  >
                    <Pencil class="w-4 h-4" />
                  </button>
                  <button
                    class="btn-icon hover:bg-stopped/20 hover:text-stopped"
                    onclick={() => handleDelete(env.id)}
                    title={st.deleteEnvironment}
                  >
                    <Trash2 class="w-4 h-4" />
                  </button>
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
