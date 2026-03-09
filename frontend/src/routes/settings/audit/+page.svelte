<script lang="ts">
  import { Shield, RefreshCw, ChevronLeft, ChevronRight } from 'lucide-svelte';
  import { language } from '$lib/stores/docker';
  import { API_BASE, getAuthHeaders } from '$lib/api/docker';
  import { goto } from '$app/navigation';
  import { currentUser } from '$lib/stores/auth';

  type AuditEntry = {
    id: string;
    timestamp: string;
    username: string;
    action: string;
    resourceType: string;
    resourceId: string;
    details: string;
    ip: string;
    success: boolean;
  };

  $effect(() => {
    if ($currentUser && !$currentUser.roles?.includes('admin')) {
      goto('/settings');
    }
  });

  let entries = $state<AuditEntry[]>([]);
  let total = $state(0);
  let loading = $state(true);
  let error = $state<string | null>(null);
  let limit = $state(50);
  let offset = $state(0);

  async function loadEntries() {
    loading = true;
    error = null;
    try {
      const res = await fetch(`${API_BASE}/api/audit?limit=${limit}&offset=${offset}`, {
        headers: getAuthHeaders(),
      });
      if (!res.ok) throw new Error('Failed to load audit log');
      const data = await res.json();
      entries = data.entries ?? [];
      total = data.total ?? 0;
    } catch (e) {
      error = e instanceof Error ? e.message : 'Error loading audit log';
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    // reactive to offset changes
    const _offset = offset;
    loadEntries();
  });

  function prevPage() {
    if (offset >= limit) offset -= limit;
  }
  function nextPage() {
    if (offset + limit < total) offset += limit;
  }

  function actionColor(action: string, success: boolean): string {
    if (!success) return 'text-stopped';
    if (action.startsWith('container.stop') || action.startsWith('user.delete')) return 'text-stopped';
    if (action.startsWith('container.start') || action.startsWith('user.create')) return 'text-running';
    if (action === 'login') return 'text-primary';
    return 'text-foreground-muted';
  }

  function formatDate(ts: string): string {
    return new Date(ts).toLocaleString($language === 'es' ? 'es-ES' : 'en-US', {
      month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit', second: '2-digit'
    });
  }
</script>

<div class="p-4 space-y-4">
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-2">
      <Shield class="w-4 h-4 text-primary" />
      <h3 class="text-sm font-semibold text-foreground">
        {$language === 'es' ? 'Registro de auditoría' : 'Audit Log'}
      </h3>
      <span class="text-xs text-foreground-muted">({total} {$language === 'es' ? 'entradas' : 'entries'})</span>
    </div>
    <button
      onclick={loadEntries}
      disabled={loading}
      class="flex items-center gap-1.5 text-xs text-foreground-muted hover:text-foreground"
    >
      <RefreshCw class="w-3.5 h-3.5 {loading ? 'animate-spin' : ''}" />
      {$language === 'es' ? 'Actualizar' : 'Refresh'}
    </button>
  </div>

  {#if error}
    <div class="p-3 bg-stopped/10 border border-stopped/30 rounded-lg text-stopped text-sm">{error}</div>
  {/if}

  {#if loading && entries.length === 0}
    <div class="flex items-center justify-center py-12">
      <RefreshCw class="w-5 h-5 animate-spin text-primary" />
    </div>
  {:else if entries.length === 0}
    <div class="text-center py-12 text-foreground-muted text-sm">
      {$language === 'es' ? 'No hay entradas de auditoría' : 'No audit entries yet'}
    </div>
  {:else}
    <div class="overflow-x-auto rounded-lg border border-border">
      <table class="w-full text-xs">
        <thead>
          <tr class="border-b border-border bg-background-secondary">
            <th class="text-left px-3 py-2 text-foreground-muted font-medium">
              {$language === 'es' ? 'Hora' : 'Time'}
            </th>
            <th class="text-left px-3 py-2 text-foreground-muted font-medium">
              {$language === 'es' ? 'Usuario' : 'User'}
            </th>
            <th class="text-left px-3 py-2 text-foreground-muted font-medium">
              {$language === 'es' ? 'Acción' : 'Action'}
            </th>
            <th class="text-left px-3 py-2 text-foreground-muted font-medium">
              {$language === 'es' ? 'Recurso' : 'Resource'}
            </th>
            <th class="text-left px-3 py-2 text-foreground-muted font-medium">IP</th>
            <th class="text-left px-3 py-2 text-foreground-muted font-medium">
              {$language === 'es' ? 'Estado' : 'Status'}
            </th>
          </tr>
        </thead>
        <tbody>
          {#each entries as entry}
            <tr class="border-b border-border/50 hover:bg-background-secondary/50">
              <td class="px-3 py-2 text-foreground-muted whitespace-nowrap">{formatDate(entry.timestamp)}</td>
              <td class="px-3 py-2 font-medium text-foreground">{entry.username}</td>
              <td class="px-3 py-2 font-mono {actionColor(entry.action, entry.success)}">{entry.action}</td>
              <td class="px-3 py-2 text-foreground-muted truncate max-w-32" title={entry.resourceId}>{entry.resourceId}</td>
              <td class="px-3 py-2 text-foreground-muted font-mono">{entry.ip}</td>
              <td class="px-3 py-2">
                <span class="px-1.5 py-0.5 rounded text-xs font-medium {entry.success ? 'bg-running/10 text-running' : 'bg-stopped/10 text-stopped'}">
                  {entry.success ? 'OK' : ($language === 'es' ? 'Error' : 'Fail')}
                </span>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>

    <!-- Pagination -->
    {#if total > limit}
      <div class="flex items-center justify-between text-xs text-foreground-muted">
        <span>{offset + 1}–{Math.min(offset + limit, total)} / {total}</span>
        <div class="flex gap-1">
          <button
            onclick={prevPage}
            disabled={offset === 0}
            class="p-1.5 rounded border border-border hover:bg-background-secondary disabled:opacity-40"
          >
            <ChevronLeft class="w-3.5 h-3.5" />
          </button>
          <button
            onclick={nextPage}
            disabled={offset + limit >= total}
            class="p-1.5 rounded border border-border hover:bg-background-secondary disabled:opacity-40"
          >
            <ChevronRight class="w-3.5 h-3.5" />
          </button>
        </div>
      </div>
    {/if}
  {/if}
</div>
