<script lang="ts">
  import {
    X,
    Folder,
    FileText,
    ArrowUp,
    RefreshCw,
    Upload,
    Download,
    FolderPlus,
    Trash2,
  } from "lucide-svelte";
  import type { Host } from "$lib/api/docker";
  import { API_BASE } from "$lib/api/docker";
  import { language } from "$lib/stores/docker";

  type HostFileEntry = {
    name: string;
    path: string;
    size: number;
    modTime: number;
    isDir: boolean;
  };

  let { host, onClose }: { host: Host; onClose: () => void } = $props();

  let currentPath = $state("/home/pi");
  let entries = $state<HostFileEntry[]>([]);
  let isLoading = $state(false);
  let error = $state<string | null>(null);

  const t = {
    en: {
      title: "Files",
      path: "Path",
      refresh: "Refresh",
      upload: "Upload",
      download: "Download",
      mkdir: "New folder",
      delete: "Delete",
      empty: "No files",
      up: "Up",
      close: "Close",
    },
    es: {
      title: "Archivos",
      path: "Ruta",
      refresh: "Actualizar",
      upload: "Subir",
      download: "Descargar",
      mkdir: "Nueva carpeta",
      delete: "Eliminar",
      empty: "Sin archivos",
      up: "Subir",
      close: "Cerrar",
    },
  }[$language] || {
    title: "Files",
    path: "Path",
    refresh: "Refresh",
    upload: "Upload",
    download: "Download",
    mkdir: "New folder",
    delete: "Delete",
    empty: "No files",
    up: "Up",
    close: "Close",
  };

  function authHeaders(): Record<string, string> {
    const token = localStorage.getItem("auth_access_token");
    const headers: Record<string, string> = {};
    if (token) headers.Authorization = `Bearer ${token}`;
    return headers;
  }

  async function loadDir(pathValue = currentPath) {
    isLoading = true;
    error = null;
    try {
      const res = await fetch(
        `${API_BASE}/api/hosts/${host.id}/files?path=${encodeURIComponent(
          pathValue,
        )}`,
        { headers: authHeaders() },
      );
      if (!res.ok) {
        throw new Error(await res.text());
      }
      const data = (await res.json()) as HostFileEntry[];
      entries = data;
      currentPath = pathValue;
    } catch (err) {
      error = err instanceof Error ? err.message : "Error";
    } finally {
      isLoading = false;
    }
  }

  function goUp() {
    if (currentPath === "/") return;
    const parts = currentPath.split("/").filter(Boolean);
    parts.pop();
    const next = "/" + parts.join("/");
    loadDir(next === "" ? "/" : next);
  }

  async function handleUpload(files: FileList | null) {
    if (!files || files.length === 0) return;
    const file = files[0];
    const body = new FormData();
    body.append("path", currentPath);
    body.append("file", file);

    isLoading = true;
    error = null;
    try {
      const res = await fetch(
        `${API_BASE}/api/hosts/${host.id}/files/upload`,
        {
          method: "POST",
          headers: authHeaders(),
          body,
        },
      );
      if (!res.ok) {
        throw new Error(await res.text());
      }
      await loadDir(currentPath);
    } catch (err) {
      error = err instanceof Error ? err.message : "Error";
    } finally {
      isLoading = false;
    }
  }

  async function handleDownload(file: HostFileEntry) {
    try {
      const res = await fetch(
        `${API_BASE}/api/hosts/${host.id}/files/download?path=${encodeURIComponent(
          file.path,
        )}`,
        { headers: authHeaders() },
      );
      if (!res.ok) {
        throw new Error(await res.text());
      }
      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = file.name;
      a.click();
      URL.revokeObjectURL(url);
    } catch (err) {
      error = err instanceof Error ? err.message : "Error";
    }
  }

  async function handleMkdir() {
    const name = prompt($language === "es" ? "Nombre de carpeta" : "Folder name");
    if (!name) return;
    const target = currentPath.endsWith("/")
      ? `${currentPath}${name}`
      : `${currentPath}/${name}`;

    isLoading = true;
    error = null;
    try {
      const res = await fetch(`${API_BASE}/api/hosts/${host.id}/files/mkdir`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          ...authHeaders(),
        },
        body: JSON.stringify({ path: target }),
      });
      if (!res.ok) {
        throw new Error(await res.text());
      }
      await loadDir(currentPath);
    } catch (err) {
      error = err instanceof Error ? err.message : "Error";
    } finally {
      isLoading = false;
    }
  }

  async function handleDelete(file: HostFileEntry) {
    const ok = confirm(
      $language === "es"
        ? `Eliminar ${file.name}?`
        : `Delete ${file.name}?`,
    );
    if (!ok) return;

    isLoading = true;
    error = null;
    try {
      const res = await fetch(
        `${API_BASE}/api/hosts/${host.id}/files?path=${encodeURIComponent(
          file.path,
        )}`,
        { method: "DELETE", headers: authHeaders() },
      );
      if (!res.ok) {
        throw new Error(await res.text());
      }
      await loadDir(currentPath);
    } catch (err) {
      error = err instanceof Error ? err.message : "Error";
    } finally {
      isLoading = false;
    }
  }

  $effect(() => {
    loadDir(currentPath);
  });
</script>

<div class="fixed inset-0 z-50 bg-black/50 backdrop-blur-sm flex items-center justify-center p-4">
  <div class="bg-background-secondary border border-border rounded-xl shadow-2xl w-full max-w-4xl max-h-[85vh] flex flex-col overflow-hidden">
    <div class="flex items-center justify-between px-4 py-3 bg-background-tertiary border-b border-border">
      <div class="flex items-center gap-2">
        <Folder class="w-4 h-4 text-primary" />
        <div>
          <p class="text-sm font-semibold text-foreground">{host.name}</p>
          <p class="text-xs text-foreground-muted">{t.title}</p>
        </div>
      </div>
      <button class="btn-icon hover:text-accent-red" onclick={onClose} title={t.close}>
        <X class="w-4 h-4" />
      </button>
    </div>

    <div class="px-4 py-3 border-b border-border flex flex-wrap items-center gap-2">
      <div class="flex items-center gap-2 flex-1 min-w-[260px]">
        <span class="text-xs text-foreground-muted">{t.path}</span>
        <input
          class="flex-1 bg-background rounded-md border border-border px-2 py-1 text-xs text-foreground"
          bind:value={currentPath}
          onkeydown={(e) => e.key === "Enter" && loadDir(currentPath)}
        />
      </div>
      <div class="flex items-center gap-2">
        <button class="btn-icon" onclick={goUp} title={t.up}>
          <ArrowUp class="w-4 h-4" />
        </button>
        <button class="btn-icon" onclick={() => loadDir(currentPath)} title={t.refresh}>
          <RefreshCw class="w-4 h-4" />
        </button>
        <label class="btn btn-ghost text-xs flex items-center gap-2">
          <Upload class="w-3.5 h-3.5" />
          {t.upload}
          <input type="file" class="hidden" onchange={(e) => handleUpload((e.target as HTMLInputElement).files)} />
        </label>
        <button class="btn btn-ghost text-xs flex items-center gap-2" onclick={handleMkdir}>
          <FolderPlus class="w-3.5 h-3.5" />
          {t.mkdir}
        </button>
      </div>
    </div>

    <div class="flex-1 overflow-auto">
      {#if error}
        <div class="p-4 text-sm text-accent-red">{error}</div>
      {/if}
      {#if isLoading}
        <div class="p-4 text-sm text-foreground-muted">Loading...</div>
      {:else if entries.length === 0}
        <div class="p-4 text-sm text-foreground-muted">{t.empty}</div>
      {:else}
        <div class="divide-y divide-border">
          {#each entries as entry}
            <div class="flex items-center justify-between px-4 py-2 hover:bg-background-tertiary/50">
              <button
                class="flex items-center gap-2 text-left text-sm text-foreground"
                onclick={() => entry.isDir && loadDir(entry.path)}
              >
                {#if entry.isDir}
                  <Folder class="w-4 h-4 text-primary" />
                {:else}
                  <FileText class="w-4 h-4 text-foreground-muted" />
                {/if}
                <span class="truncate max-w-[240px]">{entry.name}</span>
              </button>
              <div class="flex items-center gap-2">
                {#if !entry.isDir}
                  <button class="btn-icon" onclick={() => handleDownload(entry)} title={t.download}>
                    <Download class="w-4 h-4" />
                  </button>
                {/if}
                <button class="btn-icon hover:text-accent-red" onclick={() => handleDelete(entry)} title={t.delete}>
                  <Trash2 class="w-4 h-4" />
                </button>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  </div>
</div>
