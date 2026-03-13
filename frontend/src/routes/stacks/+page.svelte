<script lang="ts">
  import { onMount } from "svelte";
  import {
    Layers, Plus, RefreshCw, ChevronDown, ChevronRight,
    Play, Square, RotateCcw, Download, Trash2, Pencil,
    Loader2, X, Check, AlertCircle
  } from "lucide-svelte";
  import { language } from "$lib/stores/docker";
  import { API_BASE, getAuthHeaders } from "$lib/api/docker";
  import { currentUser, isLoading } from "$lib/stores/auth";
  import { goto } from "$app/navigation";

  // ── Types ──────────────────────────────────────────────────────────────
  interface ServiceInfo {
    id: string;
    name: string;
    state: string;
    service: string;
  }

  interface Stack {
    name: string;
    type: string; // portainer | dockerverse | external | unknown
    hasFile: boolean;
    configFilePath: string;
    workingDir: string;
    status: string; // running | partial | stopped
    services: ServiceInfo[];
  }

  // ── State ───────────────────────────────────────────────────────────────
  let hosts = $state<{ id: string; name: string }[]>([]);
  let selectedHostId = $state("");
  let stacks = $state<Stack[]>([]);
  let loading = $state(true);
  let refreshing = $state(false);
  let expandedStacks = $state<Set<string>>(new Set());

  // Edit modal
  let editingStack = $state<Stack | null>(null);
  let editContent = $state("");
  let editLoading = $state(false);
  let editSaving = $state(false);
  let editError = $state("");
  let actionOutput = $state("");

  // Action loading per stack
  let stackActionLoading = $state<Record<string, string>>({});

  // New stack modal
  let showNewStack = $state(false);
  let newName = $state("");
  let newContent = $state("");
  let newDeploying = $state(false);
  let newError = $state("");

  // ── Translations ────────────────────────────────────────────────────────
  const t = $derived($language === "es" ? {
    title: "Stacks",
    newStack: "Nuevo Stack",
    noStacks: "No hay stacks en este host",
    selectHost: "Selecciona un host",
    services: "servicios",
    edit: "Editar",
    up: "Desplegar",
    down: "Detener",
    pull: "Pull & Redeploy",
    delete: "Eliminar",
    save: "Guardar",
    saveAndDeploy: "Guardar y Desplegar",
    cancel: "Cancelar",
    deploy: "Desplegar",
    stackName: "Nombre del stack",
    composeContent: "Contenido del compose file",
    deleteConfirm: "¿Eliminar este stack? Se ejecutará docker compose down y se borrará el directorio.",
    outputLabel: "Output del comando:",
  } : {
    title: "Stacks",
    newStack: "New Stack",
    noStacks: "No stacks found on this host",
    selectHost: "Select a host",
    services: "services",
    edit: "Edit",
    up: "Deploy",
    down: "Stop",
    pull: "Pull & Redeploy",
    delete: "Delete",
    save: "Save",
    saveAndDeploy: "Save & Deploy",
    cancel: "Cancel",
    deploy: "Deploy",
    stackName: "Stack name",
    composeContent: "Compose file content",
    deleteConfirm: "Delete this stack? This will run docker compose down and remove the directory.",
    outputLabel: "Command output:",
  });

  // ── Auth guard ──────────────────────────────────────────────────────────
  $effect(() => {
    if (!$isLoading && $currentUser && !$currentUser.roles.includes("admin")) {
      goto("/");
    }
  });

  // ── Data fetching ───────────────────────────────────────────────────────
  async function fetchHosts() {
    try {
      const res = await fetch(`${API_BASE}/api/environments`, { headers: getAuthHeaders() });
      if (res.ok) {
        const data = await res.json();
        hosts = data.map((h: { id: string; name: string }) => ({ id: h.id, name: h.name }));
        if (hosts.length > 0 && !selectedHostId) {
          selectedHostId = hosts[0].id;
        }
      }
    } catch (e) { console.error("Failed to fetch hosts:", e); }
  }

  async function fetchStacks() {
    if (!selectedHostId) return;
    refreshing = true;
    try {
      const res = await fetch(`${API_BASE}/api/stacks?hostId=${selectedHostId}`, {
        headers: getAuthHeaders()
      });
      if (res.ok) {
        stacks = await res.json();
      } else {
        console.error("fetchStacks error:", res.status, res.statusText);
        // stacks stays as empty array — user sees "No stacks found"
      }
    } catch (e) { console.error("Failed to fetch stacks:", e); }
    loading = false;
    refreshing = false;
  }

  onMount(async () => {
    await fetchHosts();
  });

  $effect(() => {
    if (selectedHostId) {
      fetchStacks();
    }
  });

  // ── Stack actions ───────────────────────────────────────────────────────
  async function stackAction(stack: Stack, action: "up" | "down" | "pull") {
    stackActionLoading = { ...stackActionLoading, [stack.name]: action };
    actionOutput = "";
    try {
      const res = await fetch(`${API_BASE}/api/stacks/${stack.name}/${action}?hostId=${selectedHostId}`, {
        method: "POST",
        headers: { ...getAuthHeaders(), "Content-Type": "application/json" },
        body: JSON.stringify({})
      });
      const data = await res.json();
      actionOutput = data.output || "";
      await fetchStacks();
    } catch (e) { console.error(e); }
    const next = { ...stackActionLoading };
    delete next[stack.name];
    stackActionLoading = next;
  }

  async function deleteStack(stack: Stack) {
    if (!confirm(t.deleteConfirm)) return;
    stackActionLoading = { ...stackActionLoading, [stack.name]: "delete" };
    try {
      await fetch(`${API_BASE}/api/stacks/${stack.name}?hostId=${selectedHostId}`, {
        method: "DELETE",
        headers: getAuthHeaders()
      });
      await fetchStacks();
    } catch (e) { console.error(e); }
    const next = { ...stackActionLoading };
    delete next[stack.name];
    stackActionLoading = next;
  }

  // ── Edit modal ───────────────────────────────────────────────────────────
  async function openEdit(stack: Stack) {
    editingStack = stack;
    editContent = "";
    editError = "";
    actionOutput = "";
    editLoading = true;
    try {
      const res = await fetch(
        `${API_BASE}/api/stacks/${stack.name}/file?hostId=${selectedHostId}`,
        { headers: getAuthHeaders() }
      );
      if (res.ok) {
        const data = await res.json();
        editContent = data.content;
      } else {
        editError = "Failed to read compose file";
      }
    } catch (e) { editError = String(e); }
    editLoading = false;
  }

  function closeEdit() {
    editingStack = null;
    editContent = "";
    editError = "";
    actionOutput = "";
  }

  async function saveEdit(andDeploy = false) {
    if (!editingStack) return;
    editSaving = true;
    editError = "";
    actionOutput = "";
    try {
      const saveRes = await fetch(
        `${API_BASE}/api/stacks/${editingStack.name}/file?hostId=${selectedHostId}`,
        {
          method: "PUT",
          headers: { ...getAuthHeaders(), "Content-Type": "application/json" },
          body: JSON.stringify({ content: editContent })
        }
      );
      if (!saveRes.ok) {
        const err = await saveRes.json();
        editError = err.error || "Save failed";
        editSaving = false;
        return;
      }
      if (andDeploy) {
        const upRes = await fetch(
          `${API_BASE}/api/stacks/${editingStack.name}/up?hostId=${selectedHostId}`,
          {
            method: "POST",
            headers: { ...getAuthHeaders(), "Content-Type": "application/json" },
            body: JSON.stringify({})
          }
        );
        const upData = await upRes.json();
        actionOutput = upData.output || "";
        if (!upRes.ok) {
          editError = upData.error || "Deploy failed";
          editSaving = false;
          return;
        }
      }
      await fetchStacks();
      if (!andDeploy) closeEdit();
    } catch (e) { editError = String(e); }
    editSaving = false;
  }

  // ── New stack modal ──────────────────────────────────────────────────────
  async function createStack() {
    newDeploying = true;
    newError = "";
    try {
      const res = await fetch(`${API_BASE}/api/stacks/create?hostId=${selectedHostId}`, {
        method: "POST",
        headers: { ...getAuthHeaders(), "Content-Type": "application/json" },
        body: JSON.stringify({ name: newName.trim(), content: newContent })
      });
      const data = await res.json();
      if (!res.ok) {
        newError = data.error || "Create failed";
        newDeploying = false;
        return;
      }
      showNewStack = false;
      newName = "";
      newContent = "";
      await fetchStacks();
    } catch (e) { newError = String(e); }
    newDeploying = false;
  }

  // ── UI helpers ───────────────────────────────────────────────────────────
  function toggleExpand(name: string) {
    const next = new Set(expandedStacks);
    if (next.has(name)) next.delete(name);
    else next.add(name);
    expandedStacks = next;
  }

  function statusDotColor(status: string) {
    if (status === "running") return "text-green-400";
    if (status === "partial") return "text-yellow-400";
    return "text-gray-400";
  }

  function typeBadgeClass(type: string) {
    if (type === "portainer") return "bg-gray-700 text-gray-300";
    if (type === "dockerverse") return "bg-blue-900 text-blue-300";
    if (type === "external") return "bg-yellow-900 text-yellow-300";
    return "bg-gray-800 text-gray-500";
  }

  function typeLabel(type: string) {
    if (type === "portainer") return "Portainer";
    if (type === "dockerverse") return "DockerVerse";
    if (type === "external") return "External";
    return "Unknown";
  }

  function runningCount(services: ServiceInfo[]) {
    return services.filter(s => s.state === "running").length;
  }
</script>

<!-- Page -->
<div class="p-6 space-y-6 min-h-screen">
  <!-- Header -->
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-3">
      <Layers class="w-6 h-6 text-blue-400" />
      <h1 class="text-xl font-semibold text-white">{t.title}</h1>
    </div>
    <div class="flex items-center gap-3">
      <select
        bind:value={selectedHostId}
        class="select select-sm bg-gray-800 border border-gray-700 text-white rounded-lg px-3 py-1.5 text-sm"
      >
        {#each hosts as host}
          <option value={host.id}>{host.name}</option>
        {/each}
      </select>
      <button
        onclick={() => fetchStacks()}
        class="btn btn-ghost btn-icon"
        disabled={refreshing}
      >
        <RefreshCw class="w-4 h-4 {refreshing ? 'animate-spin' : ''}" />
      </button>
      <button
        onclick={() => { showNewStack = true; newError = ""; }}
        class="btn btn-primary btn-sm flex items-center gap-2"
      >
        <Plus class="w-4 h-4" />
        {t.newStack}
      </button>
    </div>
  </div>

  <!-- Loading -->
  {#if loading}
    <div class="flex items-center justify-center py-20">
      <Loader2 class="w-8 h-8 animate-spin text-blue-400" />
    </div>
  {:else if stacks.length === 0}
    <div class="text-center py-20 text-gray-500">
      <Layers class="w-12 h-12 mx-auto mb-3 opacity-30" />
      <p>{selectedHostId ? t.noStacks : t.selectHost}</p>
    </div>
  {:else}
    <div class="space-y-2">
      {#each stacks as stack}
        {@const expanded = expandedStacks.has(stack.name)}
        {@const busy = stackActionLoading[stack.name]}
        <div class="bg-gray-800 rounded-lg border border-gray-700 overflow-hidden">
          <!-- Stack header -->
          <button
            onclick={() => toggleExpand(stack.name)}
            class="w-full flex items-center gap-3 px-4 py-3 hover:bg-gray-750 text-left"
          >
            {#if expanded}
              <ChevronDown class="w-4 h-4 text-gray-400 flex-shrink-0" />
            {:else}
              <ChevronRight class="w-4 h-4 text-gray-400 flex-shrink-0" />
            {/if}
            <span class="font-medium text-white flex-1">{stack.name}</span>
            <span class="text-sm {statusDotColor(stack.status)}">
              ● {runningCount(stack.services)}/{stack.services.length} {t.services}
            </span>
            <span class="text-xs px-2 py-0.5 rounded {typeBadgeClass(stack.type)}">
              {typeLabel(stack.type)}
            </span>
          </button>

          <!-- Expanded body -->
          {#if expanded}
            <div class="border-t border-gray-700 px-4 pb-3">
              {#if stack.services.length > 0}
                <div class="py-2 space-y-1">
                  {#each stack.services as svc}
                    <div class="flex items-center gap-2 text-sm">
                      <span class="w-2 h-2 rounded-full flex-shrink-0 {svc.state === 'running' ? 'bg-green-400' : 'bg-gray-500'}"></span>
                      <span class="text-gray-300">{svc.name}</span>
                      <span class="text-gray-500 text-xs">{svc.state}</span>
                    </div>
                  {/each}
                </div>
              {/if}

              <div class="flex items-center gap-2 mt-2 flex-wrap">
                {#if stack.hasFile}
                  <button
                    onclick={() => openEdit(stack)}
                    class="btn btn-ghost btn-sm flex items-center gap-1"
                    disabled={!!busy}
                  >
                    <Pencil class="w-3.5 h-3.5" />
                    {t.edit}
                  </button>
                {/if}

                <button
                  onclick={() => stackAction(stack, "up")}
                  class="btn btn-ghost btn-sm flex items-center gap-1 text-green-400 hover:text-green-300"
                  disabled={!!busy}
                >
                  {#if busy === "up"}
                    <Loader2 class="w-3.5 h-3.5 animate-spin" />
                  {:else}
                    <Play class="w-3.5 h-3.5" />
                  {/if}
                  {t.up}
                </button>

                <button
                  onclick={() => stackAction(stack, "down")}
                  class="btn btn-ghost btn-sm flex items-center gap-1 text-red-400 hover:text-red-300"
                  disabled={!!busy}
                >
                  {#if busy === "down"}
                    <Loader2 class="w-3.5 h-3.5 animate-spin" />
                  {:else}
                    <Square class="w-3.5 h-3.5" />
                  {/if}
                  {t.down}
                </button>

                <button
                  onclick={() => stackAction(stack, "pull")}
                  class="btn btn-ghost btn-sm flex items-center gap-1 text-blue-400 hover:text-blue-300"
                  disabled={!!busy}
                >
                  {#if busy === "pull"}
                    <Loader2 class="w-3.5 h-3.5 animate-spin" />
                  {:else}
                    <Download class="w-3.5 h-3.5" />
                  {/if}
                  {t.pull}
                </button>

                {#if stack.type === "dockerverse"}
                  <button
                    onclick={() => deleteStack(stack)}
                    class="btn btn-ghost btn-sm flex items-center gap-1 text-red-500 hover:text-red-400 ml-auto"
                    disabled={!!busy}
                  >
                    {#if busy === "delete"}
                      <Loader2 class="w-3.5 h-3.5 animate-spin" />
                    {:else}
                      <Trash2 class="w-3.5 h-3.5" />
                    {/if}
                    {t.delete}
                  </button>
                {/if}
              </div>
            </div>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>

<!-- Edit Modal -->
{#if editingStack}
  <div class="fixed inset-0 bg-black/60 z-50 flex items-center justify-center p-4">
    <div class="bg-gray-900 rounded-xl border border-gray-700 w-full max-w-3xl max-h-[90vh] flex flex-col">
      <div class="flex items-center justify-between px-5 py-4 border-b border-gray-700">
        <div>
          <h2 class="font-semibold text-white">{editingStack.name}</h2>
          <p class="text-xs text-gray-500 mt-0.5">{editingStack.configFilePath}</p>
        </div>
        <button onclick={closeEdit} class="btn btn-ghost btn-icon">
          <X class="w-5 h-5" />
        </button>
      </div>

      <div class="flex-1 overflow-auto p-5 flex flex-col gap-4">
        {#if editLoading}
          <div class="flex items-center justify-center py-10">
            <Loader2 class="w-6 h-6 animate-spin text-blue-400" />
          </div>
        {:else if editError && !editContent}
          <div class="flex items-center gap-2 text-red-400 text-sm">
            <AlertCircle class="w-4 h-4" />
            {editError}
          </div>
        {:else}
          <textarea
            bind:value={editContent}
            class="w-full bg-gray-800 text-gray-100 font-mono text-sm p-4 rounded-lg border border-gray-700 focus:outline-none focus:border-blue-500 resize-none"
            style="min-height: 400px;"
            spellcheck="false"
          ></textarea>

          {#if editError}
            <div class="flex items-center gap-2 text-red-400 text-sm">
              <AlertCircle class="w-4 h-4" />
              {editError}
            </div>
          {/if}

          {#if actionOutput}
            <div class="bg-gray-950 rounded-lg p-3">
              <p class="text-xs text-gray-500 mb-1">{t.outputLabel}</p>
              <pre class="text-xs text-gray-300 whitespace-pre-wrap overflow-auto max-h-40">{actionOutput}</pre>
            </div>
          {/if}
        {/if}
      </div>

      {#if !editLoading && (editContent || !editError)}
        <div class="flex items-center justify-end gap-3 px-5 py-4 border-t border-gray-700">
          <button onclick={closeEdit} class="btn btn-ghost btn-sm">{t.cancel}</button>
          <button
            onclick={() => saveEdit(false)}
            class="btn btn-secondary btn-sm flex items-center gap-2"
            disabled={editSaving}
          >
            {#if editSaving}
              <Loader2 class="w-4 h-4 animate-spin" />
            {:else}
              <Check class="w-4 h-4" />
            {/if}
            {t.save}
          </button>
          <button
            onclick={() => saveEdit(true)}
            class="btn btn-primary btn-sm flex items-center gap-2"
            disabled={editSaving}
          >
            {#if editSaving}
              <Loader2 class="w-4 h-4 animate-spin" />
            {:else}
              <RotateCcw class="w-4 h-4" />
            {/if}
            {t.saveAndDeploy}
          </button>
        </div>
      {/if}
    </div>
  </div>
{/if}

<!-- New Stack Modal -->
{#if showNewStack}
  <div class="fixed inset-0 bg-black/60 z-50 flex items-center justify-center p-4">
    <div class="bg-gray-900 rounded-xl border border-gray-700 w-full max-w-2xl flex flex-col">
      <div class="flex items-center justify-between px-5 py-4 border-b border-gray-700">
        <h2 class="font-semibold text-white">{t.newStack}</h2>
        <button onclick={() => { showNewStack = false; newError = ""; }} class="btn btn-ghost btn-icon">
          <X class="w-5 h-5" />
        </button>
      </div>

      <div class="p-5 space-y-4">
        <div>
          <label class="block text-sm text-gray-400 mb-1">{t.stackName}</label>
          <input
            bind:value={newName}
            type="text"
            placeholder="my-stack"
            class="w-full bg-gray-800 border border-gray-700 text-white font-mono text-sm rounded-lg px-3 py-2 focus:outline-none focus:border-blue-500"
          />
        </div>

        <div>
          <label class="block text-sm text-gray-400 mb-1">{t.composeContent}</label>
          <textarea
            bind:value={newContent}
            class="w-full bg-gray-800 text-gray-100 font-mono text-sm p-4 rounded-lg border border-gray-700 focus:outline-none focus:border-blue-500 resize-none"
            style="min-height: 300px;"
            placeholder="services:&#10;  myapp:&#10;    image: nginx:latest&#10;    ports:&#10;      - '8080:80'"
            spellcheck="false"
          ></textarea>
        </div>

        {#if newError}
          <div class="flex items-center gap-2 text-red-400 text-sm">
            <AlertCircle class="w-4 h-4" />
            {newError}
          </div>
        {/if}
      </div>

      <div class="flex items-center justify-end gap-3 px-5 py-4 border-t border-gray-700">
        <button onclick={() => { showNewStack = false; newError = ""; }} class="btn btn-ghost btn-sm">
          {t.cancel}
        </button>
        <button
          onclick={createStack}
          class="btn btn-primary btn-sm flex items-center gap-2"
          disabled={newDeploying || !newName.trim() || !newContent.trim()}
        >
          {#if newDeploying}
            <Loader2 class="w-4 h-4 animate-spin" />
          {:else}
            <Play class="w-4 h-4" />
          {/if}
          {t.deploy}
        </button>
      </div>
    </div>
  </div>
{/if}
