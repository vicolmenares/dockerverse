<script lang="ts">
  import { onMount } from "svelte";
  import { type ScanResult, type Vulnerability } from "$lib/api/docker";
  import { language, scanHistory, refreshScanHistory } from "$lib/stores/docker";
  import { Shield, ShieldAlert, ShieldCheck, X, ChevronDown, ChevronUp, AlertTriangle } from "lucide-svelte";

  let scans = $derived($scanHistory);
  let loading = $state(true);
  let error = $state("");
  let selected = $state<ScanResult | null>(null);
  let sortedVulns = $state<Vulnerability[]>([]);

  onMount(async () => {
    try {
      await refreshScanHistory();
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to load scan history";
    } finally {
      loading = false;
    }
  });

  const severityOrder: Record<string, number> = {
    critical: 0,
    high: 1,
    medium: 2,
    low: 3,
    negligible: 4,
    unknown: 5,
  };

  const severityClass: Record<string, string> = {
    critical: "text-red-400",
    high: "text-orange-400",
    medium: "text-yellow-400",
    low: "text-blue-400",
    negligible: "text-gray-400",
    unknown: "text-gray-500",
  };

  const severityBg: Record<string, string> = {
    critical: "bg-red-500/15 text-red-400 border-red-500/30",
    high: "bg-orange-500/15 text-orange-400 border-orange-500/30",
    medium: "bg-yellow-500/15 text-yellow-400 border-yellow-500/30",
    low: "bg-blue-500/15 text-blue-400 border-blue-500/30",
    negligible: "bg-gray-500/15 text-gray-400 border-gray-500/30",
    unknown: "bg-gray-600/15 text-gray-500 border-gray-600/30",
  };

  // Summary stats
  let totalScans = $derived(scans.length);
  let imagesWithCriticals = $derived(
    new Set(scans.filter((s) => s.summary.critical > 0).map((s) => s.imageName)).size
  );
  let lastScanDate = $derived(
    scans.length > 0 ? new Date(scans[0].scannedAt).toLocaleString() : "-"
  );

  function openDetail(scan: ScanResult) {
    selected = scan;
    sortedVulns = [...(scan.vulnerabilities ?? [])].sort((a, b) => {
      const oa = severityOrder[a.severity.toLowerCase()] ?? 99;
      const ob = severityOrder[b.severity.toLowerCase()] ?? 99;
      return oa - ob;
    });
  }

  function closeDetail() {
    selected = null;
    sortedVulns = [];
  }

  function formatDate(dt: string) {
    try {
      return new Date(dt).toLocaleString();
    } catch {
      return dt;
    }
  }

  function parseImageName(imageName: string) {
    const parts = imageName.split(":");
    return parts[0].split("/").pop() ?? parts[0];
  }
</script>

<svelte:head>
  <title>Security — DockerVerse</title>
</svelte:head>

<div class="space-y-6 px-4 sm:px-6 lg:px-8 py-6">
  <!-- Page header -->
  <div class="flex items-center gap-3">
    <div class="p-2 bg-accent-red/15 rounded-lg">
      <Shield class="w-6 h-6 text-accent-red" />
    </div>
    <div>
      <h1 class="text-2xl font-bold text-foreground">
        {$language === "es" ? "Seguridad" : "Security"}
      </h1>
      <p class="text-sm text-foreground-muted">
        {$language === "es" ? "Historial de escaneos de vulnerabilidades" : "Vulnerability scan history"}
      </p>
    </div>
  </div>

  {#if loading}
    <div class="flex items-center justify-center py-20">
      <div class="w-10 h-10 border-4 border-primary/30 border-t-primary rounded-full animate-spin"></div>
    </div>
  {:else if error}
    <div class="p-4 bg-accent-red/10 border border-accent-red/30 rounded-xl text-accent-red text-sm">
      {error}
    </div>
  {:else}
    <!-- Summary cards -->
    <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
      <div class="card p-4 flex items-center gap-4">
        <div class="p-3 bg-primary/15 rounded-lg">
          <Shield class="w-5 h-5 text-primary" />
        </div>
        <div>
          <p class="text-2xl font-bold text-foreground">{totalScans}</p>
          <p class="text-xs text-foreground-muted">
            {$language === "es" ? "Escaneos totales" : "Total scans"}
          </p>
        </div>
      </div>
      <div class="card p-4 flex items-center gap-4">
        <div class="p-3 bg-accent-red/15 rounded-lg">
          <ShieldAlert class="w-5 h-5 text-accent-red" />
        </div>
        <div>
          <p class="text-2xl font-bold text-foreground">{imagesWithCriticals}</p>
          <p class="text-xs text-foreground-muted">
            {$language === "es" ? "Imágenes con críticos" : "Images with criticals"}
          </p>
        </div>
      </div>
      <div class="card p-4 flex items-center gap-4">
        <div class="p-3 bg-running/15 rounded-lg">
          <ShieldCheck class="w-5 h-5 text-running" />
        </div>
        <div>
          <p class="text-sm font-semibold text-foreground truncate">{lastScanDate}</p>
          <p class="text-xs text-foreground-muted">
            {$language === "es" ? "Último escaneo" : "Last scan"}
          </p>
        </div>
      </div>
    </div>

    <!-- Scans table -->
    {#if scans.length === 0}
      <div class="card p-12 text-center">
        <Shield class="w-12 h-12 text-foreground-muted mx-auto mb-3" />
        <p class="text-foreground-muted text-sm">
          {$language === "es" ? "No hay escaneos registrados aún." : "No scans recorded yet."}
        </p>
        <p class="text-foreground-muted text-xs mt-1">
          {$language === "es"
            ? "Actualiza un contenedor con un escáner habilitado para registrar resultados."
            : "Update a container with a scanner enabled to record results."}
        </p>
      </div>
    {:else}
      <div class="card overflow-hidden">
        <div class="overflow-x-auto">
          <table class="w-full text-sm">
            <thead>
              <tr class="border-b border-border bg-background-tertiary/40">
                <th class="text-left px-4 py-3 text-xs font-semibold text-foreground-muted uppercase tracking-wide">
                  {$language === "es" ? "Imagen" : "Image"}
                </th>
                <th class="text-left px-4 py-3 text-xs font-semibold text-foreground-muted uppercase tracking-wide">
                  {$language === "es" ? "Escáner" : "Scanner"}
                </th>
                <th class="text-center px-3 py-3 text-xs font-semibold text-red-400 uppercase tracking-wide">C</th>
                <th class="text-center px-3 py-3 text-xs font-semibold text-orange-400 uppercase tracking-wide">H</th>
                <th class="text-center px-3 py-3 text-xs font-semibold text-yellow-400 uppercase tracking-wide">M</th>
                <th class="text-center px-3 py-3 text-xs font-semibold text-blue-400 uppercase tracking-wide">L</th>
                <th class="text-left px-4 py-3 text-xs font-semibold text-foreground-muted uppercase tracking-wide">
                  {$language === "es" ? "Fecha" : "Scanned at"}
                </th>
                <th class="text-left px-4 py-3 text-xs font-semibold text-foreground-muted uppercase tracking-wide">
                  {$language === "es" ? "Estado" : "Status"}
                </th>
              </tr>
            </thead>
            <tbody>
              {#each scans as scan}
                <tr
                  class="border-b border-border/50 hover:bg-background-tertiary/30 cursor-pointer transition-colors"
                  onclick={() => openDetail(scan)}
                >
                  <td class="px-4 py-3">
                    <div class="font-medium text-foreground truncate max-w-[180px]" title={scan.imageName}>
                      {parseImageName(scan.imageName)}
                    </div>
                    <div class="text-xs text-foreground-muted truncate max-w-[180px]">
                      {scan.containerName}
                    </div>
                  </td>
                  <td class="px-4 py-3 text-foreground-muted">{scan.scanner}</td>
                  <td class="px-3 py-3 text-center">
                    <span class="font-semibold {scan.summary.critical > 0 ? 'text-red-400' : 'text-foreground-muted/40'}">
                      {scan.summary.critical}
                    </span>
                  </td>
                  <td class="px-3 py-3 text-center">
                    <span class="font-semibold {scan.summary.high > 0 ? 'text-orange-400' : 'text-foreground-muted/40'}">
                      {scan.summary.high}
                    </span>
                  </td>
                  <td class="px-3 py-3 text-center">
                    <span class="font-semibold {scan.summary.medium > 0 ? 'text-yellow-400' : 'text-foreground-muted/40'}">
                      {scan.summary.medium}
                    </span>
                  </td>
                  <td class="px-3 py-3 text-center">
                    <span class="font-semibold {scan.summary.low > 0 ? 'text-blue-400' : 'text-foreground-muted/40'}">
                      {scan.summary.low}
                    </span>
                  </td>
                  <td class="px-4 py-3 text-xs text-foreground-muted whitespace-nowrap">
                    {formatDate(scan.scannedAt)}
                  </td>
                  <td class="px-4 py-3">
                    {#if scan.blocked}
                      <span class="px-2 py-0.5 text-xs font-semibold bg-accent-red/15 text-accent-red rounded-full border border-accent-red/30">
                        {$language === "es" ? "Bloqueado" : "Blocked"}
                      </span>
                    {:else}
                      <span class="px-2 py-0.5 text-xs font-semibold bg-running/15 text-running rounded-full border border-running/30">
                        {$language === "es" ? "Pasado" : "Passed"}
                      </span>
                    {/if}
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </div>
    {/if}
  {/if}
</div>

<!-- CVE Detail Drawer/Modal -->
{#if selected}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    class="fixed inset-0 bg-black/60 z-[100] flex items-end sm:items-center justify-center p-0 sm:p-4"
    onclick={closeDetail}
  >
    <div
      class="bg-background-secondary border border-border rounded-t-xl sm:rounded-xl shadow-2xl w-full sm:max-w-2xl max-h-[85vh] flex flex-col overflow-hidden"
      onclick={(e) => e.stopPropagation()}
    >
      <!-- Drawer header -->
      <div class="flex items-start justify-between px-5 py-4 border-b border-border flex-shrink-0">
        <div>
          <h3 class="font-semibold text-foreground">{selected.containerName}</h3>
          <p class="text-xs text-foreground-muted mt-0.5 truncate max-w-[360px]">{selected.imageName}</p>
          <div class="flex items-center gap-2 mt-2 flex-wrap">
            <span class="text-xs text-foreground-muted">
              {selected.scanner} &middot; {formatDate(selected.scannedAt)}
            </span>
            {#if selected.blocked}
              <span class="px-1.5 py-0.5 text-[10px] font-semibold bg-accent-red/15 text-accent-red rounded-full border border-accent-red/30">
                {$language === "es" ? "Bloqueado" : "Blocked"}
              </span>
            {:else}
              <span class="px-1.5 py-0.5 text-[10px] font-semibold bg-running/15 text-running rounded-full border border-running/30">
                {$language === "es" ? "Pasado" : "Passed"}
              </span>
            {/if}
          </div>
        </div>
        <button class="btn-icon hover:bg-background-tertiary flex-shrink-0" onclick={closeDetail}>
          <X class="w-5 h-5" />
        </button>
      </div>

      <!-- Summary row -->
      <div class="px-5 py-3 flex items-center gap-3 border-b border-border/50 flex-shrink-0 flex-wrap">
        {#each [
          { label: "Critical", count: selected.summary.critical, cls: "bg-red-500/15 text-red-400 border-red-500/30" },
          { label: "High", count: selected.summary.high, cls: "bg-orange-500/15 text-orange-400 border-orange-500/30" },
          { label: "Medium", count: selected.summary.medium, cls: "bg-yellow-500/15 text-yellow-400 border-yellow-500/30" },
          { label: "Low", count: selected.summary.low, cls: "bg-blue-500/15 text-blue-400 border-blue-500/30" },
          { label: "Negligible", count: selected.summary.negligible, cls: "bg-gray-500/15 text-gray-400 border-gray-500/30" },
        ] as item}
          <span class="px-2.5 py-1 text-xs font-semibold rounded-full border {item.cls}">
            {item.count} {item.label}
          </span>
        {/each}
      </div>

      <!-- Block reason -->
      {#if selected.blocked && selected.blockReason}
        <div class="px-5 py-3 border-b border-border/50 flex-shrink-0">
          <div class="flex items-start gap-2 p-3 bg-accent-red/10 border border-accent-red/30 rounded-lg">
            <AlertTriangle class="w-4 h-4 text-accent-red flex-shrink-0 mt-0.5" />
            <p class="text-xs text-accent-red leading-relaxed">{selected.blockReason}</p>
          </div>
        </div>
      {/if}

      <!-- CVE table -->
      <div class="flex-1 overflow-y-auto">
        {#if sortedVulns.length === 0}
          <div class="flex flex-col items-center justify-center py-16 text-foreground-muted">
            <ShieldCheck class="w-10 h-10 mb-2 text-running" />
            <p class="text-sm">
              {$language === "es" ? "Sin vulnerabilidades encontradas" : "No vulnerabilities found"}
            </p>
          </div>
        {:else}
          <table class="w-full text-sm">
            <thead class="sticky top-0 bg-background-secondary">
              <tr class="border-b border-border bg-background-tertiary/40">
                <th class="text-left px-4 py-2 text-xs font-semibold text-foreground-muted uppercase tracking-wide">ID</th>
                <th class="text-left px-4 py-2 text-xs font-semibold text-foreground-muted uppercase tracking-wide">
                  {$language === "es" ? "Severidad" : "Severity"}
                </th>
                <th class="text-left px-4 py-2 text-xs font-semibold text-foreground-muted uppercase tracking-wide">
                  {$language === "es" ? "Paquete" : "Package"}
                </th>
                <th class="text-left px-4 py-2 text-xs font-semibold text-foreground-muted uppercase tracking-wide">
                  {$language === "es" ? "Versión" : "Version"}
                </th>
                <th class="text-left px-4 py-2 text-xs font-semibold text-foreground-muted uppercase tracking-wide">
                  {$language === "es" ? "Corrección" : "Fixed"}
                </th>
              </tr>
            </thead>
            <tbody>
              {#each sortedVulns as vuln}
                <tr class="border-b border-border/50 hover:bg-background-tertiary/30">
                  <td class="px-4 py-2">
                    {#if vuln.link}
                      <a
                        href={vuln.link}
                        target="_blank"
                        rel="noopener noreferrer"
                        class="text-primary hover:underline text-xs font-mono"
                      >
                        {vuln.id}
                      </a>
                    {:else}
                      <span class="text-xs font-mono text-foreground-muted">{vuln.id}</span>
                    {/if}
                  </td>
                  <td class="px-4 py-2">
                    <span class="px-1.5 py-0.5 text-[10px] font-semibold rounded-full border {severityBg[vuln.severity.toLowerCase()] ?? 'bg-gray-500/15 text-gray-400 border-gray-500/30'}">
                      {vuln.severity}
                    </span>
                  </td>
                  <td class="px-4 py-2 text-xs text-foreground font-mono">{vuln.package}</td>
                  <td class="px-4 py-2 text-xs text-foreground-muted font-mono">{vuln.version}</td>
                  <td class="px-4 py-2 text-xs text-running font-mono">
                    {vuln.fixedVersion ?? "—"}
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        {/if}
      </div>

      <!-- Footer -->
      <div class="px-5 py-3 border-t border-border flex-shrink-0 flex items-center justify-between">
        <span class="text-xs text-foreground-muted">
          {sortedVulns.length} {$language === "es" ? "vulnerabilidades" : "vulnerabilities"}
        </span>
        <button
          class="px-4 py-1.5 text-sm bg-background-tertiary hover:bg-background-tertiary/80 text-foreground rounded-lg transition-colors"
          onclick={closeDetail}
        >
          {$language === "es" ? "Cerrar" : "Close"}
        </button>
      </div>
    </div>
  </div>
{/if}
