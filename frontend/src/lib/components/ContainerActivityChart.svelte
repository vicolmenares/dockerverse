<script lang="ts">
  import { RefreshCw, Activity } from 'lucide-svelte';
  import { language } from '$lib/stores/docker';
  import { API_BASE, getAuthHeaders } from '$lib/api/docker';
  import { onMount, onDestroy } from 'svelte';

  type ContainerEvent = {
    timestamp: string;
    hostId: string;
    name: string;
    action: string;
  };

  type HourBucket = {
    label: string;
    hour: number;
    start: number;
    stop: number;
    restart: number;
    update: number;
    total: number;
  };

  const ACTION_COLORS: Record<string, string> = {
    start: '#22c55e',
    stop: '#ef4444',
    restart: '#f59e0b',
    update: '#3b82f6',
  };

  let events = $state<ContainerEvent[]>([]);
  let loading = $state(true);
  let error = $state<string | null>(null);
  let interval: ReturnType<typeof setInterval> | null = null;

  const WIDTH = 480;
  const HEIGHT = 120;
  const PADDING = { top: 10, right: 8, bottom: 24, left: 28 };

  let buckets = $derived.by((): HourBucket[] => {
    const now = new Date();
    const result: HourBucket[] = [];
    for (let i = 23; i >= 0; i--) {
      const d = new Date(now);
      d.setHours(d.getHours() - i, 0, 0, 0);
      result.push({
        label: `${String(d.getHours()).padStart(2, '0')}:00`,
        hour: d.getHours(),
        start: 0, stop: 0, restart: 0, update: 0, total: 0,
      });
    }
    for (const e of events) {
      const ts = new Date(e.timestamp);
      const hoursAgo = Math.floor((now.getTime() - ts.getTime()) / 3600000);
      if (hoursAgo < 24) {
        const idx = 23 - hoursAgo;
        if (idx >= 0 && idx < result.length) {
          const b = result[idx];
          if (e.action === 'start') b.start++;
          else if (e.action === 'stop') b.stop++;
          else if (e.action === 'restart') b.restart++;
          else if (e.action === 'update') b.update++;
          b.total++;
        }
      }
    }
    return result;
  });

  let maxTotal = $derived(Math.max(...buckets.map(b => b.total), 1));

  const chartW = WIDTH - PADDING.left - PADDING.right;
  const chartH = HEIGHT - PADDING.top - PADDING.bottom;

  let barWidth = $derived(Math.floor(chartW / 24) - 1);

  function getBars(bucket: HourBucket, idx: number) {
    const x = PADDING.left + idx * (chartW / 24);
    const actions: [string, number][] = [
      ['stop', bucket.stop],
      ['restart', bucket.restart],
      ['update', bucket.update],
      ['start', bucket.start],
    ];
    const bars: { x: number; y: number; h: number; color: string }[] = [];
    let stackY = HEIGHT - PADDING.bottom;
    for (const [key, val] of actions) {
      if (val === 0) continue;
      const h = Math.max((val / maxTotal) * chartH, 1);
      stackY -= h;
      bars.push({ x, y: stackY, h, color: ACTION_COLORS[key] });
    }
    return bars;
  }

  let yLabels = $derived([
    { y: PADDING.top, label: String(maxTotal) },
    { y: PADDING.top + chartH / 2, label: String(Math.round(maxTotal / 2)) },
    { y: PADDING.top + chartH, label: '0' },
  ]);

  async function load() {
    try {
      const res = await fetch(`${API_BASE}/api/container-events?hours=24`, {
        headers: getAuthHeaders(),
      });
      if (!res.ok) throw new Error('Failed to load');
      const data = await res.json();
      events = data.events ?? [];
      error = null;
    } catch {
      error = 'Failed to load events';
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    load();
    interval = setInterval(load, 60_000);
  });
  onDestroy(() => { if (interval) clearInterval(interval); });

  let t = $derived({
    title: $language === 'es' ? 'Actividad de contenedores' : 'Container Activity',
    noActivity: $language === 'es' ? 'Sin actividad en las últimas 24h' : 'No activity in the last 24h',
    last24h: $language === 'es' ? 'Últimas 24 horas' : 'Last 24 hours',
    start: $language === 'es' ? 'Inicio' : 'Start',
    stop: $language === 'es' ? 'Detención' : 'Stop',
    restart: $language === 'es' ? 'Reinicio' : 'Restart',
    update: $language === 'es' ? 'Actualización' : 'Update',
  });
</script>

<div class="bg-background-secondary rounded-xl border border-border p-4">
  <div class="flex items-center justify-between mb-3">
    <div class="flex items-center gap-2">
      <Activity class="w-4 h-4 text-primary" />
      <span class="text-sm font-semibold text-foreground">{t.title}</span>
      <span class="text-xs text-foreground-muted">{t.last24h}</span>
    </div>
    {#if loading}
      <RefreshCw class="w-3.5 h-3.5 animate-spin text-foreground-muted" />
    {/if}
  </div>

  {#if error}
    <div class="text-xs text-stopped py-4 text-center">{error}</div>
  {:else if !loading && events.length === 0}
    <div class="text-xs text-foreground-muted py-6 text-center">{t.noActivity}</div>
  {:else}
    <svg width="100%" viewBox="0 0 {WIDTH} {HEIGHT}" class="overflow-visible">
      <!-- Y axis lines and labels -->
      {#each yLabels as yl}
        <text x={PADDING.left - 4} y={yl.y + 3} text-anchor="end" class="fill-foreground-muted" font-size="8">{yl.label}</text>
        <line x1={PADDING.left} y1={yl.y} x2={WIDTH - PADDING.right} y2={yl.y} stroke="currentColor" stroke-width="0.3" class="text-border" />
      {/each}

      <!-- Bars -->
      {#each buckets as bucket, idx}
        {#each getBars(bucket, idx) as bar}
          <rect x={bar.x} y={bar.y} width={barWidth} height={bar.h} fill={bar.color} rx="1" opacity="0.85" />
        {/each}
        <!-- X axis label every 6 hours -->
        {#if idx % 6 === 0}
          <text
            x={PADDING.left + idx * (chartW / 24) + barWidth / 2}
            y={HEIGHT - PADDING.bottom + 10}
            text-anchor="middle"
            class="fill-foreground-muted"
            font-size="7"
          >{bucket.label}</text>
        {/if}
      {/each}
    </svg>

    <!-- Legend -->
    <div class="flex items-center gap-3 mt-2 flex-wrap">
      {#each [['start', t.start], ['stop', t.stop], ['restart', t.restart], ['update', t.update]] as [key, label]}
        <div class="flex items-center gap-1">
          <div class="w-2 h-2 rounded-sm" style="background:{ACTION_COLORS[key]}"></div>
          <span class="text-xs text-foreground-muted">{label}</span>
        </div>
      {/each}
    </div>
  {/if}
</div>
