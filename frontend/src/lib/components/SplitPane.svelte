<script lang="ts" module>
  // ─── Type Definitions (exported for use by the shell page) ────────────────

  export type SplitNode =
    | { kind: 'leaf'; paneId: string; tabId: string | null }
    | { kind: 'hsplit'; paneId: string; first: SplitNode; second: SplitNode; ratio: number }
    | { kind: 'vsplit'; paneId: string; first: SplitNode; second: SplitNode; ratio: number };

  export interface SplitTab {
    id: string;
    type: 'container' | 'host';
    container?: import('$lib/api/docker').Container;
    host?: import('$lib/api/docker').Host;
    label: string;
    hostLabel: string;
  }
</script>

<script lang="ts">
  import { Columns2, Rows2, X } from 'lucide-svelte';
  import Terminal from '$lib/components/Terminal.svelte';

  // ─── Props ────────────────────────────────────────────────────────────────

  interface Props {
    /** The split tree node this component renders. */
    node: SplitNode;
    /** All open terminal tabs, keyed by tab.id. */
    tabs: Map<string, SplitTab>;
    /** Terminal connection statuses, keyed by tab.id. */
    tabStatuses: Map<string, 'connecting' | 'connected' | 'disconnected' | 'error'>;
    /** paneId of the currently focused pane. */
    activePaneId: string;
    /** Called when the user clicks a pane to focus it. */
    onPaneFocus: (paneId: string) => void;
    /** Called when the user drags the resize handle. */
    onRatioChange: (splitPaneId: string, ratio: number) => void;
    /** Called when the user clicks the "split right" button on a leaf. */
    onSplitH: (paneId: string) => void;
    /** Called when the user clicks the "split down" button on a leaf. */
    onSplitV: (paneId: string) => void;
    /** Called when the user clicks the "close pane" button on a leaf. */
    onClose: (paneId: string) => void;
    /** Called when a terminal's connection status changes. */
    onStatusChange: (tabId: string, status: 'connecting' | 'connected' | 'disconnected' | 'error') => void;
  }

  let {
    node,
    tabs,
    tabStatuses,
    activePaneId,
    onPaneFocus,
    onRatioChange,
    onSplitH,
    onSplitV,
    onClose,
    onStatusChange,
  }: Props = $props();

  // ─── Drag-to-resize ───────────────────────────────────────────────────────

  /**
   * Begin a horizontal resize drag (for hsplit nodes).
   * Tracks mouse movement across the full document until mouseup.
   */
  function startHResize(e: MouseEvent) {
    if (node.kind !== 'hsplit') return;
    e.preventDefault();

    const startX = e.clientX;
    const container = (e.currentTarget as HTMLElement).parentElement!;
    const startRatio = node.ratio;

    function onMove(ev: MouseEvent) {
      if (node.kind !== 'hsplit') return;
      const dx = ev.clientX - startX;
      const totalWidth = container.offsetWidth;
      const newRatio = Math.max(0.1, Math.min(0.9, startRatio + dx / totalWidth));
      onRatioChange(node.paneId, newRatio);
    }

    function onUp() {
      document.removeEventListener('mousemove', onMove);
      document.removeEventListener('mouseup', onUp);
    }

    document.addEventListener('mousemove', onMove);
    document.addEventListener('mouseup', onUp);
  }

  /**
   * Begin a vertical resize drag (for vsplit nodes).
   */
  function startVResize(e: MouseEvent) {
    if (node.kind !== 'vsplit') return;
    e.preventDefault();

    const startY = e.clientY;
    const container = (e.currentTarget as HTMLElement).parentElement!;
    const startRatio = node.ratio;

    function onMove(ev: MouseEvent) {
      if (node.kind !== 'vsplit') return;
      const dy = ev.clientY - startY;
      const totalHeight = container.offsetHeight;
      const newRatio = Math.max(0.1, Math.min(0.9, startRatio + dy / totalHeight));
      onRatioChange(node.paneId, newRatio);
    }

    function onUp() {
      document.removeEventListener('mousemove', onMove);
      document.removeEventListener('mouseup', onUp);
    }

    document.addEventListener('mousemove', onMove);
    document.addEventListener('mouseup', onUp);
  }
</script>

<!--
  ─── Recursive Render ────────────────────────────────────────────────────────

  HSPLIT: two children side-by-side with a vertical drag handle.
  VSPLIT: two children stacked with a horizontal drag handle.
  LEAF:   one terminal (or an empty placeholder) with a hover overlay.
-->

{#if node.kind === 'hsplit'}
  <!-- ── Horizontal split (left | right) ─────────────────────────────────── -->
  <div class="flex flex-row h-full w-full overflow-hidden">
    <!-- First child: takes `ratio` of the total width -->
    <div
      class="flex-none overflow-hidden"
      style="width: {node.ratio * 100}%; min-width: 80px;"
    >
      <svelte:self
        node={node.first}
        {tabs}
        {tabStatuses}
        {activePaneId}
        {onPaneFocus}
        {onRatioChange}
        {onSplitH}
        {onSplitV}
        {onClose}
        {onStatusChange}
      />
    </div>

    <!-- Vertical drag handle -->
    <div
      class="w-[3px] flex-none cursor-col-resize bg-border hover:bg-primary/60 active:bg-primary transition-colors duration-100 z-10"
      title="Drag to resize"
      onmousedown={startHResize}
    ></div>

    <!-- Second child: takes the remaining width -->
    <div class="flex-1 overflow-hidden min-w-[80px]">
      <svelte:self
        node={node.second}
        {tabs}
        {tabStatuses}
        {activePaneId}
        {onPaneFocus}
        {onRatioChange}
        {onSplitH}
        {onSplitV}
        {onClose}
        {onStatusChange}
      />
    </div>
  </div>

{:else if node.kind === 'vsplit'}
  <!-- ── Vertical split (top / bottom) ───────────────────────────────────── -->
  <div class="flex flex-col h-full w-full overflow-hidden">
    <!-- First child: takes `ratio` of the total height -->
    <div
      class="flex-none overflow-hidden"
      style="height: {node.ratio * 100}%; min-height: 60px;"
    >
      <svelte:self
        node={node.first}
        {tabs}
        {tabStatuses}
        {activePaneId}
        {onPaneFocus}
        {onRatioChange}
        {onSplitH}
        {onSplitV}
        {onClose}
        {onStatusChange}
      />
    </div>

    <!-- Horizontal drag handle -->
    <div
      class="h-[3px] flex-none cursor-row-resize bg-border hover:bg-primary/60 active:bg-primary transition-colors duration-100 z-10"
      title="Drag to resize"
      onmousedown={startVResize}
    ></div>

    <!-- Second child: takes the remaining height -->
    <div class="flex-1 overflow-hidden min-h-[60px]">
      <svelte:self
        node={node.second}
        {tabs}
        {tabStatuses}
        {activePaneId}
        {onPaneFocus}
        {onRatioChange}
        {onSplitH}
        {onSplitV}
        {onClose}
        {onStatusChange}
      />
    </div>
  </div>

{:else}
  <!-- ── Leaf pane ────────────────────────────────────────────────────────── -->
  {@const tab = node.tabId != null ? (tabs.get(node.tabId) ?? null) : null}
  {@const isActive = activePaneId === node.paneId}
  {@const statusKey = tab?.id ?? ''}
  {@const connStatus = tabStatuses.get(statusKey) ?? 'connecting'}

  <!-- Outer wrapper: relative so we can absolutely-position the hover overlay -->
  <div
    class="relative h-full w-full overflow-hidden {isActive
      ? 'ring-1 ring-inset ring-primary/50'
      : 'ring-1 ring-inset ring-transparent'} transition-shadow duration-150"
    role="button"
    tabindex="0"
    aria-label="Terminal pane{tab ? ': ' + tab.label : ''}"
    onclick={() => onPaneFocus(node.paneId)}
    onkeydown={(e) => e.key === 'Enter' && onPaneFocus(node.paneId)}
  >
    {#if tab}
      <!-- ── Terminal component ────────────────────────────────────────── -->
      <div class="h-full w-full">
        {#if tab.type === 'container'}
          <Terminal
            container={tab.container}
            mode="container"
            terminalMode="embedded"
            active={isActive}
            onStatusChange={(s) => onStatusChange(tab.id, s)}
          />
        {:else}
          <Terminal
            host={tab.host}
            mode="host"
            terminalMode="embedded"
            active={isActive}
            onStatusChange={(s) => onStatusChange(tab.id, s)}
          />
        {/if}
      </div>
    {:else}
      <!-- ── Empty pane placeholder ─────────────────────────────────────── -->
      <div class="h-full w-full flex items-center justify-center bg-background-secondary">
        <p class="text-sm text-foreground-muted select-none">Empty pane</p>
      </div>
    {/if}

    <!-- ── Hover control overlay ──────────────────────────────────────────
         Appears at the top of the pane when hovered or when the pane
         is focused. Contains: label, status dot, split-right, split-down,
         and close buttons.
    ─────────────────────────────────────────────────────────────────────── -->
    <div
      class="pane-overlay absolute top-0 left-0 right-0 h-7 flex items-center gap-1 px-2
             bg-black/70 backdrop-blur-sm z-20 select-none
             opacity-0 group-hover:opacity-100 transition-opacity duration-150"
    >
      <!-- Status dot -->
      <span
        class="w-1.5 h-1.5 rounded-full flex-shrink-0 {connStatus === 'connected'
          ? 'bg-running'
          : connStatus === 'connecting'
            ? 'bg-primary animate-pulse'
            : connStatus === 'error'
              ? 'bg-stopped'
              : 'bg-paused'}"
      ></span>

      <!-- Tab label -->
      <span class="text-xs text-white/80 font-mono truncate flex-1 min-w-0">
        {tab ? tab.label : 'empty'}
        {#if tab?.hostLabel}
          <span class="text-white/40">@{tab.hostLabel}</span>
        {/if}
      </span>

      <!-- Split right (hsplit) -->
      <button
        class="pane-ctrl-btn"
        onclick={(e) => { e.stopPropagation(); onSplitH(node.paneId); }}
        title="Split right"
        aria-label="Split pane right"
      >
        <Columns2 class="w-3.5 h-3.5" />
      </button>

      <!-- Split down (vsplit) -->
      <button
        class="pane-ctrl-btn"
        onclick={(e) => { e.stopPropagation(); onSplitV(node.paneId); }}
        title="Split down"
        aria-label="Split pane down"
      >
        <Rows2 class="w-3.5 h-3.5" />
      </button>

      <!-- Close pane -->
      <button
        class="pane-ctrl-btn hover:text-accent-red"
        onclick={(e) => { e.stopPropagation(); onClose(node.paneId); }}
        title="Close pane"
        aria-label="Close pane"
      >
        <X class="w-3.5 h-3.5" />
      </button>
    </div>
  </div>
{/if}

<style>
  /*
   * The hover overlay uses a CSS :hover approach rather than a Svelte
   * $state so it's zero-overhead and works even when the pane is obscured
   * by the terminal canvas. The .pane-overlay is revealed whenever its
   * parent wrapper is hovered.
   */
  div:hover > .pane-overlay,
  div:focus-within > .pane-overlay {
    opacity: 1;
  }

  /* Small icon button inside the overlay */
  :global(.pane-ctrl-btn) {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 1.375rem;   /* 22px */
    height: 1.375rem;
    border-radius: 0.25rem;
    color: rgba(255, 255, 255, 0.65);
    background: transparent;
    border: none;
    cursor: pointer;
    padding: 0;
    flex-shrink: 0;
    transition: background-color 100ms, color 100ms;
  }

  :global(.pane-ctrl-btn:hover) {
    background-color: rgba(255, 255, 255, 0.12);
    color: rgba(255, 255, 255, 0.9);
  }
</style>
