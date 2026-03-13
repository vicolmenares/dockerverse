<script lang="ts">
  import { onMount } from "svelte";
  import {
    Layers, Plus, RefreshCw, Play, Square, RotateCcw,
    Download, Trash2, Pencil, Loader2, X, Check,
    AlertCircle, Server, ChevronDown, ChevronRight
  } from "lucide-svelte";
  import { language } from "$lib/stores/docker";
  import { API_BASE, getAuthHeaders } from "$lib/api/docker";
  import { currentUser, isLoading } from "$lib/stores/auth";
  import { goto } from "$app/navigation";

  // ── Types ───────────────────────────────────────────────────────────────
  interface ServiceInfo { id: string; name: string; state: string; service: string; }
  interface Stack {
    name: string; type: string; hasFile: boolean;
    configFilePath: string; workingDir: string;
    status: string; services: ServiceInfo[];
    hostId: string; hostName: string;
  }
  interface HostGroup { id: string; name: string; stacks: Stack[]; loading: boolean; error: string; collapsed: boolean; }

  // ── State ────────────────────────────────────────────────────────────────
  let hostGroups = $state<HostGroup[]>([]);
  let globalLoading = $state(true);

  // Side panel — edit
  let panelStack = $state<Stack | null>(null);
  let panelMode = $state<"edit" | "new">("edit");
  let panelContent = $state("");
  let panelLoading = $state(false);
  let panelSaving = $state(false);
  let panelError = $state("");
  let panelOutput = $state("");

  // New stack form
  let newHostId = $state("");
  let newName = $state("");
  let newContent = $state("");
  let newDeploying = $state(false);
  let newError = $state("");

  // Action loading per stack key = `${hostId}:${name}`
  let stackBusy = $state<Record<string, string>>({});

  // ── Translations ─────────────────────────────────────────────────────────
  const t = $derived($language === "es" ? {
    title: "Stacks", newStack: "Nuevo Stack", refresh: "Actualizar",
    noStacks: "Sin stacks", services: "servicios", edit: "Editar",
    up: "Desplegar", down: "Detener", pull: "Pull & Redeploy",
    delete: "Eliminar", save: "Guardar", saveAndDeploy: "Guardar y Desplegar",
    cancel: "Cancelar", deploy: "Desplegar", stackName: "Nombre del stack",
    composeContent: "Contenido compose", outputLabel: "Output:",
    deleteConfirm: "¿Eliminar este stack? Se ejecutará docker compose down y se borrará el directorio.",
    host: "Host", stack: "Stack", upOf: "Act/Total", type: "Tipo", actions: "Acciones",
    fileNotAccessible: "Archivo no accesible",
  } : {
    title: "Stacks", newStack: "New Stack", refresh: "Refresh",
    noStacks: "No stacks", services: "services", edit: "Edit",
    up: "Deploy", down: "Stop", pull: "Pull & Redeploy",
    delete: "Delete", save: "Save", saveAndDeploy: "Save & Deploy",
    cancel: "Cancel", deploy: "Deploy", stackName: "Stack name",
    composeContent: "Compose content", outputLabel: "Output:",
    deleteConfirm: "Delete this stack? This will run docker compose down and remove the directory.",
    host: "Host", stack: "Stack", upOf: "Up/Total", type: "Type", actions: "Actions",
    fileNotAccessible: "File not accessible",
  });

  // ── Auth guard ───────────────────────────────────────────────────────────
  $effect(() => {
    if (!$isLoading && $currentUser && !$currentUser.roles.includes("admin")) goto("/");
  });

  // ── Data fetching ────────────────────────────────────────────────────────
  async function loadAll() {
    globalLoading = true;
    try {
      const res = await fetch(`${API_BASE}/api/hosts/names`, { headers: getAuthHeaders() });
      if (!res.ok) { globalLoading = false; return; }
      const envs: { id: string; name: string }[] = await res.json();
      hostGroups = envs.map(e => ({ id: e.id, name: e.name, stacks: [], loading: true, error: "", collapsed: false }));
      newHostId = envs[0]?.id ?? "";
      await Promise.allSettled(
        envs.map(async (env, i) => {
          try {
            const r = await fetch(`${API_BASE}/api/stacks?hostId=${env.id}`, { headers: getAuthHeaders() });
            if (r.ok) {
              const raw: Stack[] = await r.json();
              hostGroups[i].stacks = raw.map(s => ({ ...s, hostId: env.id, hostName: env.name }));
            } else {
              hostGroups[i].error = `HTTP ${r.status}`;
            }
          } catch { hostGroups[i].error = "Connection failed"; }
          hostGroups[i].loading = false;
          hostGroups = [...hostGroups];
        })
      );
    } catch { /* ignore */ }
    globalLoading = false;
  }

  onMount(loadAll);

  async function refreshHost(groupIdx: number) {
    const g = hostGroups[groupIdx];
    hostGroups[groupIdx] = { ...g, loading: true, error: "" };
    hostGroups = [...hostGroups];
    try {
      const r = await fetch(`${API_BASE}/api/stacks?hostId=${g.id}`, { headers: getAuthHeaders() });
      if (r.ok) {
        const raw: Stack[] = await r.json();
        hostGroups[groupIdx].stacks = raw.map(s => ({ ...s, hostId: g.id, hostName: g.name }));
      } else {
        hostGroups[groupIdx].error = `HTTP ${r.status}`;
      }
    } catch { hostGroups[groupIdx].error = "Connection failed"; }
    hostGroups[groupIdx].loading = false;
    hostGroups = [...hostGroups];
  }

  // ── Helpers ──────────────────────────────────────────────────────────────
  function busyKey(hostId: string, name: string) { return `${hostId}:${name}`; }
  function runningCount(services: ServiceInfo[]) { return services.filter(s => s.state === "running").length; }

  function statusAccent(status: string) {
    if (status === "running") return "#22c55e";
    if (status === "partial") return "#eab308";
    return "#52525b";
  }

  function typeBadge(type: string) {
    if (type === "portainer") return "bg-zinc-700 text-zinc-300";
    if (type === "dockerverse") return "bg-blue-950 text-blue-300";
    if (type === "external") return "bg-amber-950 text-amber-300";
    return "bg-zinc-800 text-zinc-500";
  }

  function typeLabel(type: string) {
    if (type === "portainer") return "Portainer";
    if (type === "dockerverse") return "DockerVerse";
    if (type === "external") return "External";
    return "Unknown";
  }

  function hostGroupIdx(hostId: string) {
    return hostGroups.findIndex(g => g.id === hostId);
  }

  // ── Stack actions ────────────────────────────────────────────────────────
  async function stackAction(stack: Stack, action: "up" | "down" | "pull") {
    const key = busyKey(stack.hostId, stack.name);
    stackBusy = { ...stackBusy, [key]: action };
    panelOutput = "";
    try {
      const res = await fetch(`${API_BASE}/api/stacks/${stack.name}/${action}?hostId=${stack.hostId}`, {
        method: "POST",
        headers: { ...getAuthHeaders(), "Content-Type": "application/json" },
        body: JSON.stringify({})
      });
      const data = await res.json();
      if (panelStack?.name === stack.name && panelStack?.hostId === stack.hostId) {
        panelOutput = data.output || "";
        if (!res.ok) panelError = data.error || `${action} failed`;
      }
      const gi = hostGroupIdx(stack.hostId);
      if (gi >= 0) await refreshHost(gi);
    } catch (e) { console.error(e); }
    const next = { ...stackBusy };
    delete next[key];
    stackBusy = next;
  }

  async function deleteStack(stack: Stack) {
    if (!confirm(t.deleteConfirm)) return;
    const key = busyKey(stack.hostId, stack.name);
    stackBusy = { ...stackBusy, [key]: "delete" };
    try {
      await fetch(`${API_BASE}/api/stacks/${stack.name}?hostId=${stack.hostId}`, {
        method: "DELETE",
        headers: getAuthHeaders()
      });
      if (panelStack?.name === stack.name && panelStack?.hostId === stack.hostId) closePanel();
      const gi = hostGroupIdx(stack.hostId);
      if (gi >= 0) await refreshHost(gi);
    } catch (e) { console.error(e); }
    const next = { ...stackBusy };
    delete next[key];
    stackBusy = next;
  }

  // ── Edit panel ───────────────────────────────────────────────────────────
  async function openEdit(stack: Stack) {
    panelStack = stack;
    panelMode = "edit";
    panelContent = "";
    panelError = "";
    panelOutput = "";
    panelLoading = true;
    try {
      const res = await fetch(`${API_BASE}/api/stacks/${stack.name}/file?hostId=${stack.hostId}`, {
        headers: getAuthHeaders()
      });
      if (res.ok) {
        const data = await res.json();
        panelContent = data.content;
      } else {
        panelError = t.fileNotAccessible;
      }
    } catch (e) { panelError = String(e); }
    panelLoading = false;
  }

  function closePanel() {
    panelStack = null;
    panelContent = "";
    panelError = "";
    panelOutput = "";
  }

  async function saveEdit(andDeploy = false) {
    if (!panelStack) return;
    panelSaving = true;
    panelError = "";
    panelOutput = "";
    try {
      const saveRes = await fetch(`${API_BASE}/api/stacks/${panelStack.name}/file?hostId=${panelStack.hostId}`, {
        method: "PUT",
        headers: { ...getAuthHeaders(), "Content-Type": "application/json" },
        body: JSON.stringify({ content: panelContent })
      });
      if (!saveRes.ok) {
        const err = await saveRes.json();
        panelError = err.error || "Save failed";
        panelSaving = false;
        return;
      }
      if (andDeploy) {
        const upRes = await fetch(`${API_BASE}/api/stacks/${panelStack.name}/up?hostId=${panelStack.hostId}`, {
          method: "POST",
          headers: { ...getAuthHeaders(), "Content-Type": "application/json" },
          body: JSON.stringify({})
        });
        const upData = await upRes.json();
        panelOutput = upData.output || "";
        if (!upRes.ok) {
          panelError = upData.error || "Deploy failed";
          panelSaving = false;
          return;
        }
      }
      const gi = hostGroupIdx(panelStack.hostId);
      if (gi >= 0) await refreshHost(gi);
      if (!andDeploy) closePanel();
    } catch (e) { panelError = String(e); }
    panelSaving = false;
  }

  // ── New stack panel ──────────────────────────────────────────────────────
  function openNew() {
    panelStack = null;
    panelMode = "new";
    newName = "";
    newContent = "";
    newError = "";
    panelOutput = "";
  }

  async function createStack() {
    newDeploying = true;
    newError = "";
    try {
      const res = await fetch(`${API_BASE}/api/stacks/create?hostId=${newHostId}`, {
        method: "POST",
        headers: { ...getAuthHeaders(), "Content-Type": "application/json" },
        body: JSON.stringify({ name: newName.trim(), content: newContent })
      });
      const data = await res.json();
      if (!res.ok) { newError = data.error || "Create failed"; newDeploying = false; return; }
      closePanel();
      const gi = hostGroupIdx(newHostId);
      if (gi >= 0) await refreshHost(gi);
    } catch (e) { newError = String(e); }
    newDeploying = false;
  }

  const panelOpen = $derived(panelMode === "new" ? true : panelStack !== null);
</script>

<!-- Page: full-screen flex with optional side panel -->
<div class="flex h-full min-h-screen bg-zinc-950 text-zinc-200">

  <!-- Main column -->
  <div class="flex-1 flex flex-col overflow-auto" class:mr-[460px]={panelOpen}>

    <!-- Header -->
    <div class="flex items-center justify-between px-6 py-4 border-b border-zinc-800 sticky top-0 bg-zinc-950 z-10">
      <div class="flex items-center gap-2">
        <Layers class="w-5 h-5 text-zinc-400" />
        <h1 class="text-base font-semibold tracking-wide">{t.title}</h1>
      </div>
      <div class="flex items-center gap-2">
        <button
          onclick={loadAll}
          class="flex items-center gap-1.5 px-3 py-1.5 text-xs text-zinc-400 hover:text-zinc-200 border border-zinc-800 hover:border-zinc-600 rounded transition-colors"
          disabled={globalLoading}
        >
          <RefreshCw class="w-3.5 h-3.5 {globalLoading ? 'animate-spin' : ''}" />
          {t.refresh}
        </button>
        <button
          onclick={openNew}
          class="flex items-center gap-1.5 px-3 py-1.5 text-xs text-white bg-blue-600 hover:bg-blue-500 rounded transition-colors"
        >
          <Plus class="w-3.5 h-3.5" />
          {t.newStack}
        </button>
      </div>
    </div>

    <!-- Table header -->
    <div class="grid grid-cols-[1fr_80px_100px_160px] gap-0 px-6 py-2 text-xs text-zinc-500 uppercase tracking-wider border-b border-zinc-800/50">
      <span>{t.stack}</span>
      <span class="text-right">{t.upOf}</span>
      <span class="text-center">{t.type}</span>
      <span class="text-right">{t.actions}</span>
    </div>

    <!-- Global loading -->
    {#if globalLoading}
      <div class="flex items-center justify-center py-24">
        <Loader2 class="w-6 h-6 animate-spin text-zinc-500" />
      </div>
    {:else}
      {#each hostGroups as group, gi}
        <!-- Host separator row -->
        <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
        <div
          onclick={() => { hostGroups[gi].collapsed = !hostGroups[gi].collapsed; hostGroups = [...hostGroups]; }}
          class="group w-full flex items-center gap-2 px-6 py-2 bg-zinc-900 border-y border-zinc-800 text-xs text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800/60 transition-colors cursor-pointer"
        >
          {#if group.collapsed}
            <ChevronRight class="w-3.5 h-3.5" />
          {:else}
            <ChevronDown class="w-3.5 h-3.5" />
          {/if}
          <Server class="w-3.5 h-3.5" />
          <span class="font-medium uppercase tracking-widest">{group.name}</span>
          {#if group.loading}
            <Loader2 class="w-3 h-3 animate-spin ml-1" />
          {:else if group.error}
            <span class="text-red-500 ml-1">{group.error}</span>
          {:else}
            <span class="text-zinc-600 ml-1">({group.stacks.length})</span>
          {/if}
          <div class="flex-1"></div>
          <button
            onclick={(e) => { e.stopPropagation(); refreshHost(gi); }}
            class="opacity-0 group-hover:opacity-100 p-0.5 hover:text-white transition-opacity"
            title={t.refresh}
          >
            <RefreshCw class="w-3 h-3" />
          </button>
        </div>

        {#if !group.collapsed}
          {#if group.stacks.length === 0 && !group.loading}
            <div class="px-6 py-4 text-sm text-zinc-600 border-b border-zinc-800/40">
              {t.noStacks}
            </div>
          {:else}
            {#each group.stacks as stack}
              {@const key = busyKey(stack.hostId, stack.name)}
              {@const busy = stackBusy[key]}
              {@const rc = runningCount(stack.services)}
              {@const total = stack.services.length}
              {@const isActive = panelStack?.name === stack.name && panelStack?.hostId === stack.hostId}
              <div
                class="group grid grid-cols-[1fr_80px_100px_160px] gap-0 px-6 py-3 border-b border-zinc-800/40 hover:bg-zinc-900/60 transition-colors relative"
                style="border-left: 3px solid {statusAccent(stack.status)}"
                class:bg-zinc-900={isActive}
              >
                <!-- Stack name -->
                <div class="flex flex-col justify-center min-w-0">
                  <span class="font-mono text-sm text-zinc-100 truncate">{stack.name}</span>
                  <span class="font-mono text-xs text-zinc-600 truncate">{stack.hostName}</span>
                </div>

                <!-- Running count -->
                <div class="flex items-center justify-end">
                  <span class="font-mono text-sm" style="color: {statusAccent(stack.status)}">{rc}/{total}</span>
                </div>

                <!-- Type badge -->
                <div class="flex items-center justify-center">
                  <span class="text-xs px-2 py-0.5 rounded font-mono {typeBadge(stack.type)}">
                    {typeLabel(stack.type)}
                  </span>
                </div>

                <!-- Actions (visible on hover or when panel open for this stack) -->
                <div class="flex items-center justify-end gap-1 opacity-0 group-hover:opacity-100 transition-opacity"
                     class:opacity-100={isActive || !!busy}>
                  {#if stack.hasFile}
                    <button
                      onclick={() => openEdit(stack)}
                      class="p-1.5 rounded text-zinc-400 hover:text-zinc-100 hover:bg-zinc-700 transition-colors"
                      title={t.edit}
                      disabled={!!busy}
                    >
                      <Pencil class="w-3.5 h-3.5" />
                    </button>
                  {/if}

                  <button
                    onclick={() => stackAction(stack, "up")}
                    class="p-1.5 rounded text-green-500 hover:text-green-300 hover:bg-zinc-700 transition-colors"
                    title={t.up}
                    disabled={!!busy}
                  >
                    {#if busy === "up"}
                      <Loader2 class="w-3.5 h-3.5 animate-spin" />
                    {:else}
                      <Play class="w-3.5 h-3.5" />
                    {/if}
                  </button>

                  <button
                    onclick={() => stackAction(stack, "down")}
                    class="p-1.5 rounded text-red-500 hover:text-red-300 hover:bg-zinc-700 transition-colors"
                    title={t.down}
                    disabled={!!busy}
                  >
                    {#if busy === "down"}
                      <Loader2 class="w-3.5 h-3.5 animate-spin" />
                    {:else}
                      <Square class="w-3.5 h-3.5" />
                    {/if}
                  </button>

                  <button
                    onclick={() => stackAction(stack, "pull")}
                    class="p-1.5 rounded text-blue-400 hover:text-blue-200 hover:bg-zinc-700 transition-colors"
                    title={t.pull}
                    disabled={!!busy}
                  >
                    {#if busy === "pull"}
                      <Loader2 class="w-3.5 h-3.5 animate-spin" />
                    {:else}
                      <Download class="w-3.5 h-3.5" />
                    {/if}
                  </button>

                  {#if stack.type === "dockerverse"}
                    <button
                      onclick={() => deleteStack(stack)}
                      class="p-1.5 rounded text-red-600 hover:text-red-400 hover:bg-zinc-700 transition-colors"
                      title={t.delete}
                      disabled={!!busy}
                    >
                      {#if busy === "delete"}
                        <Loader2 class="w-3.5 h-3.5 animate-spin" />
                      {:else}
                        <Trash2 class="w-3.5 h-3.5" />
                      {/if}
                    </button>
                  {/if}
                </div>
              </div>
            {/each}
          {/if}
        {/if}
      {/each}
    {/if}
  </div>

  <!-- Side panel -->
  {#if panelOpen}
    <div class="fixed top-0 right-0 w-[460px] h-full bg-zinc-900 border-l border-zinc-800 flex flex-col z-20">

      <!-- Panel header -->
      <div class="flex items-center justify-between px-5 py-4 border-b border-zinc-800">
        {#if panelMode === "edit" && panelStack}
          <div class="min-w-0">
            <p class="font-mono text-sm font-medium text-zinc-100 truncate">{panelStack.name}</p>
            <p class="font-mono text-xs text-zinc-500 truncate mt-0.5">{panelStack.configFilePath}</p>
          </div>
        {:else}
          <p class="font-semibold text-sm text-zinc-100">{t.newStack}</p>
        {/if}
        <button onclick={closePanel} class="p-1.5 rounded text-zinc-500 hover:text-zinc-200 hover:bg-zinc-800 transition-colors ml-2 flex-shrink-0">
          <X class="w-4 h-4" />
        </button>
      </div>

      <!-- Panel body -->
      <div class="flex-1 overflow-auto flex flex-col p-5 gap-4">

        {#if panelMode === "edit"}
          {#if panelLoading}
            <div class="flex items-center justify-center py-12">
              <Loader2 class="w-5 h-5 animate-spin text-zinc-500" />
            </div>
          {:else if panelError && !panelContent}
            <div class="flex items-center gap-2 text-red-400 text-sm">
              <AlertCircle class="w-4 h-4" />
              {panelError}
            </div>
          {:else}
            <textarea
              bind:value={panelContent}
              class="flex-1 w-full bg-zinc-950 text-zinc-100 font-mono text-xs p-4 rounded border border-zinc-800 focus:outline-none focus:border-zinc-600 resize-none"
              style="min-height: 340px;"
              spellcheck="false"
            ></textarea>

            {#if panelError}
              <div class="flex items-center gap-2 text-red-400 text-xs">
                <AlertCircle class="w-3.5 h-3.5" />
                {panelError}
              </div>
            {/if}

            {#if panelOutput}
              <div class="bg-zinc-950 rounded border border-zinc-800 p-3">
                <p class="text-xs text-zinc-500 mb-1">{t.outputLabel}</p>
                <pre class="text-xs text-zinc-300 whitespace-pre-wrap overflow-auto max-h-32">{panelOutput}</pre>
              </div>
            {/if}
          {/if}

        {:else}
          <!-- New stack form -->
          <div class="space-y-4">
            <div>
              <label class="block text-xs text-zinc-500 mb-1.5">Host</label>
              <select
                bind:value={newHostId}
                class="w-full bg-zinc-950 border border-zinc-800 text-zinc-200 text-sm font-mono rounded px-3 py-2 focus:outline-none focus:border-zinc-600"
              >
                {#each hostGroups as g}
                  <option value={g.id}>{g.name}</option>
                {/each}
              </select>
            </div>

            <div>
              <label class="block text-xs text-zinc-500 mb-1.5">{t.stackName}</label>
              <input
                bind:value={newName}
                type="text"
                placeholder="my-stack"
                class="w-full bg-zinc-950 border border-zinc-800 text-zinc-200 font-mono text-sm rounded px-3 py-2 focus:outline-none focus:border-zinc-600"
              />
            </div>

            <div class="flex-1">
              <label class="block text-xs text-zinc-500 mb-1.5">{t.composeContent}</label>
              <textarea
                bind:value={newContent}
                class="w-full bg-zinc-950 text-zinc-100 font-mono text-xs p-4 rounded border border-zinc-800 focus:outline-none focus:border-zinc-600 resize-none"
                style="min-height: 280px;"
                placeholder="services:&#10;  myapp:&#10;    image: nginx:latest&#10;    ports:&#10;      - '8080:80'"
                spellcheck="false"
              ></textarea>
            </div>

            {#if newError}
              <div class="flex items-center gap-2 text-red-400 text-xs">
                <AlertCircle class="w-3.5 h-3.5" />
                {newError}
              </div>
            {/if}
          </div>
        {/if}
      </div>

      <!-- Panel footer -->
      <div class="flex items-center justify-end gap-2 px-5 py-4 border-t border-zinc-800">
        {#if panelMode === "edit"}
          {#if !panelLoading && (panelContent || !panelError)}
            <button onclick={closePanel} class="px-3 py-1.5 text-xs text-zinc-400 hover:text-zinc-200 border border-zinc-800 hover:border-zinc-600 rounded transition-colors">
              {t.cancel}
            </button>
            <button
              onclick={() => saveEdit(false)}
              class="flex items-center gap-1.5 px-3 py-1.5 text-xs text-zinc-200 border border-zinc-700 hover:border-zinc-500 rounded transition-colors"
              disabled={panelSaving}
            >
              {#if panelSaving}
                <Loader2 class="w-3.5 h-3.5 animate-spin" />
              {:else}
                <Check class="w-3.5 h-3.5" />
              {/if}
              {t.save}
            </button>
            <button
              onclick={() => saveEdit(true)}
              class="flex items-center gap-1.5 px-3 py-1.5 text-xs text-white bg-blue-600 hover:bg-blue-500 rounded transition-colors"
              disabled={panelSaving}
            >
              {#if panelSaving}
                <Loader2 class="w-3.5 h-3.5 animate-spin" />
              {:else}
                <RotateCcw class="w-3.5 h-3.5" />
              {/if}
              {t.saveAndDeploy}
            </button>
          {/if}
        {:else}
          <button
            onclick={closePanel}
            class="px-3 py-1.5 text-xs text-zinc-400 hover:text-zinc-200 border border-zinc-800 hover:border-zinc-600 rounded transition-colors"
          >
            {t.cancel}
          </button>
          <button
            onclick={createStack}
            class="flex items-center gap-1.5 px-3 py-1.5 text-xs text-white bg-blue-600 hover:bg-blue-500 rounded transition-colors"
            disabled={newDeploying || !newName.trim() || !newContent.trim()}
          >
            {#if newDeploying}
              <Loader2 class="w-3.5 h-3.5 animate-spin" />
            {:else}
              <Play class="w-3.5 h-3.5" />
            {/if}
            {t.deploy}
          </button>
        {/if}
      </div>
    </div>
  {/if}
</div>
