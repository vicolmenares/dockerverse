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

  // Severity left-border colors
  const severityBorder: Record<string, string> = {
    critical: "#ef4444",
    high: "#f97316",
    medium: "#eab308",
    low: "#3b82f6",
    negligible: "#52525b",
    unknown: "#52525b",
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

<div class="min-h-screen">
  <!-- Sticky page header -->
  <div class="sticky top-0 z-10 border-b border-zinc-800 bg-zinc-950 px-6 py-3 flex items-center gap-3">
    <Shield class="w-4 h-4 text-zinc-500" />
    <span class="text-xs uppercase tracking-widest text-zinc-500 font-semibold">
      {$language === "es" ? "Seguridad" : "Security"}
    </span>
    <span class="text-zinc-700 text-xs">—</span>
    <span class="text-xs text-zinc-600">
      {$language === "es" ? "Historial de escaneos de vulnerabilidades" : "Vulnerability scan history"}
    </span>
  </div>

  {#if loading}
    <div class="flex items-center justify-center py-20">
      <div class="w-8 h-8 border-2 border-zinc-700 border-t-zinc-400 rounded-full animate-spin"></div>
    </div>
  {:else if error}
    <div class="mx-6 mt-6 border-l-4 border-red-500 bg-red-500/10 px-4 py-3 text-red-400 text-sm">
      {error}
    </div>
  {:else}
    <!-- Summary stats bar -->
    <div class="border-b border-zinc-800 bg-zinc-950 px-6 py-3 flex items-center overflow-x-auto">
      <div class="flex items-center gap-3 pr-5">
        <Shield class="w-4 h-4 text-zinc-500 shrink-0" />
        <span class="text-zinc-500 uppercase tracking-widest text-[10px] font-semibold">
          {$language === "es" ? "Escaneos" : "Scans"}
        </span>
        <span class="font-mono text-zinc-200">{totalScans}</span>
      </div>
      <div class="w-px h-5 bg-zinc-800 shrink-0"></div>
      <div class="flex items-center gap-3 px-5">
        <ShieldAlert class="w-4 h-4 text-red-500 shrink-0" />
        <span class="text-zinc-500 uppercase tracking-widest text-[10px] font-semibold">
          {$language === "es" ? "Con críticos" : "With criticals"}
        </span>
        <span class="font-mono {imagesWithCriticals > 0 ? 'text-red-400' : 'text-zinc-200'}">{imagesWithCriticals}</span>
      </div>
      <div class="w-px h-5 bg-zinc-800 shrink-0"></div>
      <div class="flex items-center gap-3 px-5">
        <ShieldCheck class="w-4 h-4 text-zinc-500 shrink-0" />
        <span class="text-zinc-500 uppercase tracking-widest text-[10px] font-semibold">
          {$language === "es" ? "Último escaneo" : "Last scan"}
        </span>
        <span class="font-mono text-zinc-400 text-xs">{lastScanDate}</span>
      </div>
    </div>

    <!-- Scans table -->
    {#if scans.length === 0}
      <div class="px-6 py-16 text-center">
        <Shield class="w-10 h-10 text-zinc-700 mx-auto mb-3" />
        <p class="text-zinc-600 text-sm">
          {$language === "es" ? "No hay escaneos registrados aún." : "No scans recorded yet."}
        </p>
        <p class="text-zinc-700 text-xs mt-1">
          {$language === "es"
            ? "Actualiza un contenedor con un escáner habilitado para registrar resultados."
            : "Update a container with a scanner enabled to record results."}
        </p>
      </div>
    {:else}
      <!-- Column headers -->
      <div class="border-b border-zinc-800 bg-zinc-900 px-6 py-2 grid grid-cols-[1fr_100px_32px_32px_32px_32px_160px_90px] gap-4 items-center">
        <span class="text-[10px] uppercase tracking-widest text-zinc-600 font-semibold">
          {$language === "es" ? "Imagen" : "Image"}
        </span>
        <span class="text-[10px] uppercase tracking-widest text-zinc-600 font-semibold">
          {$language === "es" ? "Escáner" : "Scanner"}
        </span>
        <span class="text-[10px] uppercase tracking-widest text-red-500/70 font-semibold text-center">C</span>
        <span class="text-[10px] uppercase tracking-widest text-orange-500/70 font-semibold text-center">H</span>
        <span class="text-[10px] uppercase tracking-widest text-yellow-500/70 font-semibold text-center">M</span>
        <span class="text-[10px] uppercase tracking-widest text-blue-500/70 font-semibold text-center">L</span>
        <span class="text-[10px] uppercase tracking-widest text-zinc-600 font-semibold">
          {$language === "es" ? "Fecha" : "Scanned at"}
        </span>
        <span class="text-[10px] uppercase tracking-widest text-zinc-600 font-semibold">
          {$language === "es" ? "Estado" : "Status"}
        </span>
      </div>

      {#each scans as scan}
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <div
          class="group border-b border-zinc-800/60 hover:bg-zinc-900/50 transition-colors cursor-pointer px-6 py-3 grid grid-cols-[1fr_100px_32px_32px_32px_32px_160px_90px] gap-4 items-center"
          style="border-left: 3px solid {scan.summary.critical > 0 ? '#ef4444' : scan.summary.high > 0 ? '#f97316' : scan.summary.medium > 0 ? '#eab308' : '#22c55e'}"
          onclick={() => openDetail(scan)}
        >
          <div class="min-w-0">
            <p class="text-sm font-mono text-zinc-200 truncate">{parseImageName(scan.imageName)}</p>
            <p class="text-[10px] font-mono text-zinc-600 truncate">{scan.containerName}</p>
          </div>
          <span class="text-xs font-mono text-zinc-500 truncate">{scan.scanner}</span>
          <span class="text-xs font-mono text-center {scan.summary.critical > 0 ? 'text-red-400 font-semibold' : 'text-zinc-700'}">{scan.summary.critical}</span>
          <span class="text-xs font-mono text-center {scan.summary.high > 0 ? 'text-orange-400 font-semibold' : 'text-zinc-700'}">{scan.summary.high}</span>
          <span class="text-xs font-mono text-center {scan.summary.medium > 0 ? 'text-yellow-400 font-semibold' : 'text-zinc-700'}">{scan.summary.medium}</span>
          <span class="text-xs font-mono text-center {scan.summary.low > 0 ? 'text-blue-400 font-semibold' : 'text-zinc-700'}">{scan.summary.low}</span>
          <span class="text-xs font-mono text-zinc-500 whitespace-nowrap">{formatDate(scan.scannedAt)}</span>
          <span>
            {#if scan.blocked}
              <span class="text-xs font-mono px-2 py-0.5 bg-red-500/10 text-red-400 border border-red-500/20">
                {$language === "es" ? "Bloqueado" : "Blocked"}
              </span>
            {:else}
              <span class="text-xs font-mono px-2 py-0.5 bg-green-500/10 text-green-400 border border-green-500/20">
                {$language === "es" ? "Pasado" : "Passed"}
              </span>
            {/if}
          </span>
        </div>
      {/each}
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
      class="bg-zinc-950 border border-zinc-800 w-full sm:max-w-2xl max-h-[85vh] flex flex-col overflow-hidden"
      onclick={(e) => e.stopPropagation()}
    >
      <!-- Drawer header -->
      <div class="flex items-start justify-between px-5 py-4 border-b border-zinc-800 flex-shrink-0">
        <div>
          <h3 class="font-mono text-zinc-200">{selected.containerName}</h3>
          <p class="text-xs font-mono text-zinc-600 mt-0.5 truncate max-w-[360px]">{selected.imageName}</p>
          <div class="flex items-center gap-2 mt-2 flex-wrap">
            <span class="text-xs font-mono text-zinc-600">
              {selected.scanner} &middot; {formatDate(selected.scannedAt)}
            </span>
            {#if selected.blocked}
              <span class="text-xs font-mono px-2 py-0.5 bg-red-500/10 text-red-400 border border-red-500/20">
                {$language === "es" ? "Bloqueado" : "Blocked"}
              </span>
            {:else}
              <span class="text-xs font-mono px-2 py-0.5 bg-green-500/10 text-green-400 border border-green-500/20">
                {$language === "es" ? "Pasado" : "Passed"}
              </span>
            {/if}
          </div>
        </div>
        <button class="p-1.5 hover:bg-zinc-800 transition-colors text-zinc-500 hover:text-zinc-300 flex-shrink-0" onclick={closeDetail}>
          <X class="w-4 h-4" />
        </button>
      </div>

      <!-- Summary row -->
      <div class="px-5 py-3 flex items-center gap-2 border-b border-zinc-800/50 flex-shrink-0 flex-wrap">
        {#each [
          { label: "Critical", count: selected.summary.critical, cls: "bg-red-500/15 text-red-400 border-red-500/30" },
          { label: "High", count: selected.summary.high, cls: "bg-orange-500/15 text-orange-400 border-orange-500/30" },
          { label: "Medium", count: selected.summary.medium, cls: "bg-yellow-500/15 text-yellow-400 border-yellow-500/30" },
          { label: "Low", count: selected.summary.low, cls: "bg-blue-500/15 text-blue-400 border-blue-500/30" },
          { label: "Negligible", count: selected.summary.negligible, cls: "bg-gray-500/15 text-gray-400 border-gray-500/30" },
        ] as item}
          <span class="px-2 py-0.5 text-xs font-mono border {item.cls}">
            {item.count} {item.label}
          </span>
        {/each}
      </div>

      <!-- Block reason -->
      {#if selected.blocked && selected.blockReason}
        <div class="px-5 py-3 border-b border-zinc-800/50 flex-shrink-0">
          <div class="flex items-start gap-2 border-l-4 border-red-500 bg-red-500/10 px-3 py-2">
            <AlertTriangle class="w-4 h-4 text-red-400 flex-shrink-0 mt-0.5" />
            <p class="text-xs text-red-400 leading-relaxed">{selected.blockReason}</p>
          </div>
        </div>
      {/if}

      <!-- CVE table -->
      <div class="flex-1 overflow-y-auto">
        {#if sortedVulns.length === 0}
          <div class="flex flex-col items-center justify-center py-16 text-zinc-600">
            <ShieldCheck class="w-10 h-10 mb-2 text-green-500/50" />
            <p class="text-sm">
              {$language === "es" ? "Sin vulnerabilidades encontradas" : "No vulnerabilities found"}
            </p>
          </div>
        {:else}
          <!-- CVE column headers -->
          <div class="sticky top-0 border-b border-zinc-800 bg-zinc-900 px-4 py-2 grid grid-cols-[140px_90px_1fr_100px_100px] gap-3 items-center">
            <span class="text-[10px] uppercase tracking-widest text-zinc-600 font-semibold">ID</span>
            <span class="text-[10px] uppercase tracking-widest text-zinc-600 font-semibold">
              {$language === "es" ? "Severidad" : "Severity"}
            </span>
            <span class="text-[10px] uppercase tracking-widest text-zinc-600 font-semibold">
              {$language === "es" ? "Paquete" : "Package"}
            </span>
            <span class="text-[10px] uppercase tracking-widest text-zinc-600 font-semibold">
              {$language === "es" ? "Versión" : "Version"}
            </span>
            <span class="text-[10px] uppercase tracking-widest text-zinc-600 font-semibold">
              {$language === "es" ? "Corrección" : "Fixed"}
            </span>
          </div>
          {#each sortedVulns as vuln}
            <div
              class="border-b border-zinc-800/50 hover:bg-zinc-900/50 px-4 py-2 grid grid-cols-[140px_90px_1fr_100px_100px] gap-3 items-center"
              style="border-left: 3px solid {severityBorder[vuln.severity.toLowerCase()] ?? '#52525b'}"
            >
              <div>
                {#if vuln.link}
                  <a
                    href={vuln.link}
                    target="_blank"
                    rel="noopener noreferrer"
                    class="text-xs font-mono text-blue-400 hover:underline truncate block"
                  >
                    {vuln.id}
                  </a>
                {:else}
                  <span class="text-xs font-mono text-zinc-500 truncate block">{vuln.id}</span>
                {/if}
              </div>
              <div>
                <span class="text-xs font-mono px-1.5 py-0.5 border {severityBg[vuln.severity.toLowerCase()] ?? 'bg-gray-500/15 text-gray-400 border-gray-500/30'}">
                  {vuln.severity}
                </span>
              </div>
              <span class="text-xs font-mono text-zinc-300 truncate">{vuln.package}</span>
              <span class="text-xs font-mono text-zinc-500 truncate">{vuln.version}</span>
              <span class="text-xs font-mono text-green-400 truncate">{vuln.fixedVersion ?? "—"}</span>
            </div>
          {/each}
        {/if}
      </div>

      <!-- Footer -->
      <div class="px-5 py-3 border-t border-zinc-800 flex-shrink-0 flex items-center justify-between">
        <span class="text-xs font-mono text-zinc-600">
          {sortedVulns.length} {$language === "es" ? "vulnerabilidades" : "vulnerabilities"}
        </span>
        <button
          class="px-4 py-1.5 text-xs font-mono bg-zinc-800 hover:bg-zinc-700 text-zinc-300 transition-colors"
          onclick={closeDetail}
        >
          {$language === "es" ? "Cerrar" : "Close"}
        </button>
      </div>
    </div>
  </div>
{/if}
