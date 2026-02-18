<script lang="ts">
  import { onMount } from "svelte";
  import { SquareTerminal, Plus, X, Server, Box, ChevronDown, Check, HardDrive, Info } from "lucide-svelte";
  import SplitPane, { type SplitNode, type SplitTab } from "$lib/components/SplitPane.svelte";
  import { containers, hosts } from "$lib/stores/docker";
  import type { Container, Host } from "$lib/api/docker";

  interface Tab {
    id: string;
    type: "container" | "host";
    container?: Container;
    host?: Host;
    label: string;
    hostLabel: string;
  }

  // ─── Tab bar state ─────────────────────────────────────────────────────────
  let tabs = $state<Tab[]>([]);
  let activeTabId = $state<string | null>(null);
  let selectedHostId = $state<string>("");
  let selectedContainerId = $state<string>("");
  let tabStatuses = $state<Map<string, "connecting" | "connected" | "disconnected" | "error">>(new Map());
  let lastOpenedType = $state<"container" | "host">("container");

  // ─── Dropdown UI state ─────────────────────────────────────────────────────
  let hostOpen = $state(false);
  let containerOpen = $state(false);
  let hostDropdownEl: HTMLDivElement | null = $state(null);
  let containerDropdownEl: HTMLDivElement | null = $state(null);
  let stripHighlight = $state(false);
  let stripEl: HTMLDivElement | null = $state(null);
  let sftpToast = $state(false);

  // ─── Split pane state ──────────────────────────────────────────────────────
  let splitTree = $state<SplitNode | null>(null);
  let activePaneId = $state<string>("");
  let tabsMap = $state<Map<string, SplitTab>>(new Map());

  // ─── Derived ───────────────────────────────────────────────────────────────
  let hostList = $derived($hosts);
  let runningContainers = $derived(
    $containers.filter(
      (c) => c.state === "running" && (!selectedHostId || c.hostId === selectedHostId)
    )
  );
  let selectedHostName = $derived(
    hostList.find((h) => h.id === selectedHostId)?.name ?? "Host"
  );
  let selectedContainerName = $derived(
    runningContainers.find((c) => c.id === selectedContainerId)?.name ?? "Container"
  );

  // ─── Auto-select first host ────────────────────────────────────────────────
  $effect(() => {
    if (hostList.length > 0 && !selectedHostId) {
      selectedHostId = hostList[0].id;
    }
  });

  // ─── Auto-select first container (guard against SSE reset) ────────────────
  $effect(() => {
    const ids = new Set(runningContainers.map((c) => c.id));
    if (!selectedContainerId || !ids.has(selectedContainerId)) {
      selectedContainerId = runningContainers[0]?.id ?? "";
    }
  });

  // ─── Strip highlight flash ─────────────────────────────────────────────────
  function flashStrip() {
    stripHighlight = true;
    setTimeout(() => (stripHighlight = false), 600);
  }

  // ─── ID generators ─────────────────────────────────────────────────────────
  function generateId(): string {
    return Math.random().toString(36).slice(2, 9);
  }

  function generatePaneId(): string {
    return "pane-" + Math.random().toString(36).slice(2, 9);
  }

  // ─── Split tree helpers ────────────────────────────────────────────────────

  function updateRatio(node: SplitNode, targetId: string, ratio: number): SplitNode {
    if (node.kind === "leaf") return node;
    if (node.paneId === targetId) return { ...node, ratio };
    return {
      ...node,
      first: updateRatio(node.first, targetId, ratio),
      second: updateRatio(node.second, targetId, ratio),
    };
  }

  function splitLeafH(node: SplitNode, targetPaneId: string, newLeaf: SplitNode): SplitNode {
    if (node.kind === "leaf") {
      if (node.paneId !== targetPaneId) return node;
      return { kind: "hsplit", paneId: generatePaneId(), first: node, second: newLeaf, ratio: 0.5 };
    }
    return {
      ...node,
      first: splitLeafH(node.first, targetPaneId, newLeaf),
      second: splitLeafH(node.second, targetPaneId, newLeaf),
    };
  }

  function splitLeafV(node: SplitNode, targetPaneId: string, newLeaf: SplitNode): SplitNode {
    if (node.kind === "leaf") {
      if (node.paneId !== targetPaneId) return node;
      return { kind: "vsplit", paneId: generatePaneId(), first: node, second: newLeaf, ratio: 0.5 };
    }
    return {
      ...node,
      first: splitLeafV(node.first, targetPaneId, newLeaf),
      second: splitLeafV(node.second, targetPaneId, newLeaf),
    };
  }

  function assignTabToPane(node: SplitNode, targetPaneId: string, tabId: string): SplitNode {
    if (node.kind === "leaf") return node.paneId === targetPaneId ? { ...node, tabId } : node;
    return {
      ...node,
      first: assignTabToPane(node.first, targetPaneId, tabId),
      second: assignTabToPane(node.second, targetPaneId, tabId),
    };
  }

  function removePane(node: SplitNode, targetPaneId: string): SplitNode | null {
    if (node.kind === "leaf") return node.paneId === targetPaneId ? null : node;
    const newFirst = removePane(node.first, targetPaneId);
    const newSecond = removePane(node.second, targetPaneId);
    if (newFirst === null) return newSecond;
    if (newSecond === null) return newFirst;
    return { ...node, first: newFirst, second: newSecond };
  }

  function firstLeafPaneId(node: SplitNode): string {
    if (node.kind === "leaf") return node.paneId;
    return firstLeafPaneId(node.first);
  }

  /** Returns tabId for a paneId. `null` = empty pane, `undefined` = pane not found. */
  function getTabIdForPane(node: SplitNode, targetPaneId: string): string | null | undefined {
    if (node.kind === "leaf") return node.paneId === targetPaneId ? node.tabId : undefined;
    const fromFirst = getTabIdForPane(node.first, targetPaneId);
    if (fromFirst !== undefined) return fromFirst;
    return getTabIdForPane(node.second, targetPaneId);
  }

  /** Returns the paneId holding a given tabId, or null. */
  function findPaneForTab(node: SplitNode, tabId: string): string | null {
    if (node.kind === "leaf") return node.tabId === tabId ? node.paneId : null;
    return findPaneForTab(node.first, tabId) ?? findPaneForTab(node.second, tabId);
  }

  // ─── Add tab to split tree ─────────────────────────────────────────────────
  function addTabToTree(tab: Tab) {
    const newPaneId = generatePaneId();
    const newLeaf: SplitNode = { kind: "leaf", paneId: newPaneId, tabId: tab.id };

    if (!splitTree) {
      splitTree = newLeaf;
      activePaneId = newPaneId;
    } else {
      const currentTabId = activePaneId ? getTabIdForPane(splitTree, activePaneId) : undefined;
      if (currentTabId === null) {
        // Active pane is empty — assign this tab there
        splitTree = assignTabToPane(splitTree, activePaneId, tab.id);
      } else {
        // Active pane is occupied — split it horizontally
        splitTree = splitLeafH(splitTree, activePaneId, newLeaf);
        activePaneId = newPaneId;
      }
    }
  }

  // ─── Open shells ───────────────────────────────────────────────────────────
  function openContainerShell() {
    const container = $containers.find((c) => c.id === selectedContainerId);
    if (!container) return;
    const host = $hosts.find((h) => h.id === container.hostId);
    const id = generateId();
    const tab: Tab = {
      id,
      type: "container",
      container,
      label: container.name,
      hostLabel: host?.name ?? container.hostId,
    };
    tabs = [...tabs, tab];
    activeTabId = id;
    lastOpenedType = "container";
    tabsMap = new Map(tabsMap).set(id, tab);
    addTabToTree(tab);
  }

  function openHostSSH() {
    const host = $hosts.find((h) => h.id === selectedHostId);
    if (!host) return;
    const id = generateId();
    const tab: Tab = {
      id,
      type: "host",
      host,
      label: `${host.name} SSH`,
      hostLabel: host.name,
    };
    tabs = [...tabs, tab];
    activeTabId = id;
    lastOpenedType = "host";
    tabsMap = new Map(tabsMap).set(id, tab);
    addTabToTree(tab);
  }

  // ─── Close tab from tab bar X button ──────────────────────────────────────
  function closeTab(tabId: string) {
    const idx = tabs.findIndex((t) => t.id === tabId);
    tabs = tabs.filter((t) => t.id !== tabId);
    if (activeTabId === tabId) {
      activeTabId = tabs.length > 0 ? tabs[Math.max(0, idx - 1)].id : null;
    }

    const newStatuses = new Map(tabStatuses);
    newStatuses.delete(tabId);
    tabStatuses = newStatuses;

    const newTabsMap = new Map(tabsMap);
    newTabsMap.delete(tabId);
    tabsMap = newTabsMap;

    if (splitTree) {
      const paneId = findPaneForTab(splitTree, tabId);
      if (paneId) {
        const wasActive = activePaneId === paneId;
        splitTree = removePane(splitTree, paneId);
        if (splitTree) {
          if (wasActive) {
            activePaneId = firstLeafPaneId(splitTree);
            const newTabId = getTabIdForPane(splitTree, activePaneId);
            if (newTabId) activeTabId = newTabId;
          }
        } else {
          activePaneId = "";
        }
      }
    }
  }

  // ─── Activate tab from tab bar click ──────────────────────────────────────
  function activateTab(tabId: string) {
    activeTabId = tabId;
    if (splitTree) {
      const paneId = findPaneForTab(splitTree, tabId);
      if (paneId) activePaneId = paneId;
    }
  }

  // ─── SplitPane event handlers ──────────────────────────────────────────────
  function handleRatioChange(splitNodeId: string, ratio: number) {
    if (!splitTree) return;
    splitTree = updateRatio(splitTree, splitNodeId, ratio);
  }

  function handleSplitH(paneId: string) {
    if (!splitTree) return;
    const newPaneId = generatePaneId();
    splitTree = splitLeafH(splitTree, paneId, { kind: "leaf", paneId: newPaneId, tabId: null });
    activePaneId = newPaneId;
    flashStrip();
  }

  function handleSplitV(paneId: string) {
    if (!splitTree) return;
    const newPaneId = generatePaneId();
    splitTree = splitLeafV(splitTree, paneId, { kind: "leaf", paneId: newPaneId, tabId: null });
    activePaneId = newPaneId;
    flashStrip();
  }

  function handlePaneFocus(paneId: string) {
    activePaneId = paneId;
    if (splitTree) {
      const tabId = getTabIdForPane(splitTree, paneId);
      if (tabId) activeTabId = tabId;
    }
  }

  function handleClosePane(paneId: string) {
    if (!splitTree) return;
    const tabId = getTabIdForPane(splitTree, paneId);
    const wasActive = activePaneId === paneId;
    splitTree = removePane(splitTree, paneId);

    if (tabId) {
      const idx = tabs.findIndex((t) => t.id === tabId);
      tabs = tabs.filter((t) => t.id !== tabId);
      if (activeTabId === tabId) {
        activeTabId = tabs.length > 0 ? tabs[Math.max(0, idx - 1)].id : null;
      }
      const newStatuses = new Map(tabStatuses);
      newStatuses.delete(tabId);
      tabStatuses = newStatuses;
      const newTabsMap = new Map(tabsMap);
      newTabsMap.delete(tabId);
      tabsMap = newTabsMap;
    }

    if (splitTree) {
      if (wasActive) {
        activePaneId = firstLeafPaneId(splitTree);
        const newTabId = getTabIdForPane(splitTree, activePaneId);
        if (newTabId) activeTabId = newTabId;
      }
    } else {
      activePaneId = "";
    }
  }

  function handleStatusChange(tabId: string, status: "connecting" | "connected" | "disconnected" | "error") {
    tabStatuses = new Map(tabStatuses).set(tabId, status);
  }

  // ─── Keyboard shortcuts ────────────────────────────────────────────────────
  function handleKeydown(e: KeyboardEvent) {
    if (e.key === "Escape") { hostOpen = false; containerOpen = false; }
    if (e.ctrlKey && e.key === "t") {
      e.preventDefault();
      lastOpenedType === "host" ? openHostSSH() : openContainerShell();
    }
    if (e.ctrlKey && e.key === "w" && activeTabId) {
      e.preventDefault();
      closeTab(activeTabId);
    }
  }

  onMount(() => {
    function handleOutside(e: MouseEvent) {
      if (hostOpen && hostDropdownEl && !hostDropdownEl.contains(e.target as Node)) hostOpen = false;
      if (containerOpen && containerDropdownEl && !containerDropdownEl.contains(e.target as Node)) containerOpen = false;
    }
    document.addEventListener("keydown", handleKeydown);
    document.addEventListener("mousedown", handleOutside);
    return () => {
      document.removeEventListener("keydown", handleKeydown);
      document.removeEventListener("mousedown", handleOutside);
    };
  });
</script>

<div class="fixed top-16 left-0 right-0 bottom-0 flex flex-col overflow-hidden bg-background z-10 shell-page-root">
  <!-- Toolbar -->
  <div class="flex items-center gap-3 px-4 py-2 bg-background-secondary border-b border-border flex-shrink-0">

    <!-- Fused command strip: host | container | open-shell -->
    <div
      bind:this={stripEl}
      class="flex items-stretch divide-x divide-border rounded-lg border border-border bg-background transition-shadow duration-150 {stripHighlight ? 'ring-2 ring-primary/50' : ''}"
    >

      <!-- Host dropdown -->
      <div class="relative" bind:this={hostDropdownEl}>
        <button
          class="flex items-center gap-2 pl-3 pr-2.5 py-1.5 text-sm text-foreground hover:bg-background-tertiary transition-colors duration-150 h-full rounded-l-lg"
          onclick={() => { hostOpen = !hostOpen; containerOpen = false; }}
          aria-expanded={hostOpen}
          aria-haspopup="listbox"
        >
          <Server class="w-3.5 h-3.5 text-foreground-muted flex-shrink-0" />
          <span class="font-medium whitespace-nowrap">{selectedHostName}</span>
          <ChevronDown class="w-3.5 h-3.5 text-foreground-muted transition-transform duration-200 {hostOpen ? 'rotate-180' : ''}" />
        </button>
        {#if hostOpen}
          <ul
            class="absolute top-full left-0 mt-1.5 bg-background-secondary border border-border rounded-lg shadow-xl shadow-black/50 z-50 min-w-[180px] py-1 overflow-hidden"
            role="listbox"
          >
            {#each hostList as host}
              <li role="option" aria-selected={selectedHostId === host.id}>
                <button
                  class="w-full flex items-center gap-2.5 px-3 py-1.5 text-sm hover:bg-background-tertiary transition-colors duration-100 text-left {selectedHostId === host.id ? 'text-primary font-medium' : 'text-foreground'}"
                  onclick={() => { selectedHostId = host.id; hostOpen = false; }}
                >
                  <Server class="w-3.5 h-3.5 flex-shrink-0 {selectedHostId === host.id ? 'text-primary' : 'text-foreground-muted'}" />
                  <span class="flex-1">{host.name}</span>
                  {#if selectedHostId === host.id}
                    <Check class="w-3.5 h-3.5 text-primary flex-shrink-0" />
                  {/if}
                </button>
              </li>
            {/each}
          </ul>
        {/if}
      </div>

      <!-- Container dropdown -->
      <div class="relative" bind:this={containerDropdownEl}>
        <button
          class="flex items-center gap-2 pl-3 pr-2.5 py-1.5 text-sm text-foreground hover:bg-background-tertiary transition-colors duration-150 h-full min-w-[170px] disabled:opacity-50 disabled:cursor-not-allowed"
          onclick={() => { containerOpen = !containerOpen; hostOpen = false; }}
          disabled={runningContainers.length === 0}
          aria-expanded={containerOpen}
          aria-haspopup="listbox"
        >
          <Box class="w-3.5 h-3.5 text-foreground-muted flex-shrink-0" />
          <span class="font-mono text-xs flex-1 text-left truncate">
            {runningContainers.length === 0 ? 'no containers' : selectedContainerName}
          </span>
          <ChevronDown class="w-3.5 h-3.5 text-foreground-muted transition-transform duration-200 {containerOpen ? 'rotate-180' : ''}" />
        </button>
        {#if containerOpen && runningContainers.length > 0}
          <ul
            class="absolute top-full left-0 mt-1.5 bg-background-secondary border border-border rounded-lg shadow-xl shadow-black/50 z-50 min-w-[240px] py-1 overflow-hidden max-h-72 overflow-y-auto"
            role="listbox"
          >
            {#each runningContainers as c}
              <li role="option" aria-selected={selectedContainerId === c.id}>
                <button
                  class="w-full flex items-center gap-2.5 px-3 py-1.5 text-sm hover:bg-background-tertiary transition-colors duration-100 text-left {selectedContainerId === c.id ? 'text-primary' : 'text-foreground'}"
                  onclick={() => { selectedContainerId = c.id; containerOpen = false; }}
                >
                  <Box class="w-3 h-3 flex-shrink-0 {selectedContainerId === c.id ? 'text-primary' : 'text-foreground-muted'}" />
                  <span class="font-mono text-xs flex-1 truncate">{c.name}</span>
                  {#if selectedContainerId === c.id}
                    <Check class="w-3.5 h-3.5 text-primary flex-shrink-0" />
                  {/if}
                </button>
              </li>
            {/each}
          </ul>
        {/if}
      </div>

      <!-- Open Shell — fused end of group -->
      <button
        class="flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium text-primary hover:bg-primary/10 transition-colors duration-150 disabled:opacity-40 disabled:cursor-not-allowed whitespace-nowrap rounded-r-lg"
        onclick={openContainerShell}
        disabled={!selectedContainerId}
      >
        <SquareTerminal class="w-4 h-4" />
        Open Shell
      </button>
    </div>

    <!-- SSH Host + SFTP — secondary, far right -->
    <div class="ml-auto flex items-center gap-2">
      <button
        class="flex items-center gap-1.5 px-3 py-1.5 text-sm text-foreground-muted hover:text-foreground border border-border hover:border-primary/50 rounded-lg transition-colors duration-150 disabled:opacity-40 disabled:cursor-not-allowed"
        onclick={openHostSSH}
        disabled={!selectedHostId}
      >
        <Server class="w-4 h-4" />
        SSH Host
      </button>
      <button
        class="flex items-center gap-1.5 px-3 py-1.5 text-sm text-foreground-muted hover:text-foreground border border-border hover:border-primary/50 rounded-lg transition-colors duration-150 disabled:opacity-40 disabled:cursor-not-allowed"
        onclick={() => { sftpToast = true; setTimeout(() => sftpToast = false, 3000); }}
        title="SFTP File Browser"
      >
        <HardDrive class="w-4 h-4" />
        SFTP
      </button>
    </div>
  </div>

  {#if sftpToast}
    <div class="fixed bottom-4 right-4 bg-background-secondary border border-border rounded-lg px-4 py-2 text-sm text-foreground shadow-lg z-50 flex items-center gap-2">
      <Info class="w-4 h-4 text-primary" />
      SFTP support coming soon
    </div>
  {/if}

  <!-- Tab bar -->
  {#if tabs.length > 0}
    <div class="flex items-center bg-background-secondary border-b border-border overflow-x-auto flex-shrink-0 scrollbar-hide">
      {#each tabs as tab}
        {@const tabStatus = tabStatuses.get(tab.id) ?? "connecting"}
        <div
          role="tab"
          tabindex="0"
          class="flex items-center gap-2 px-4 py-2 text-sm border-b-2 whitespace-nowrap transition-colors cursor-pointer {activeTabId === tab.id
            ? 'border-primary text-foreground bg-background'
            : 'border-transparent text-foreground-muted hover:text-foreground'}"
          onclick={() => activateTab(tab.id)}
          onkeydown={(e) => e.key === "Enter" && activateTab(tab.id)}
          aria-selected={activeTabId === tab.id}
        >
          {#if tab.type === "host"}
            <Server class="w-3.5 h-3.5 flex-shrink-0" />
          {:else}
            <Box class="w-3.5 h-3.5 flex-shrink-0" />
          {/if}
          <span class="w-2 h-2 rounded-full flex-shrink-0 {tabStatus === 'connected' ? 'bg-running' : tabStatus === 'connecting' ? 'bg-primary animate-pulse' : tabStatus === 'error' ? 'bg-stopped' : 'bg-paused'}"></span>
          <span>{tab.label}</span>
          <span class="text-foreground-muted text-xs">@{tab.hostLabel}</span>
          <button
            class="ml-1 rounded hover:bg-background-tertiary p-0.5 text-foreground-muted hover:text-foreground"
            onclick={(e) => { e.stopPropagation(); closeTab(tab.id); }}
            aria-label="Close tab"
          >
            <X class="w-3 h-3" />
          </button>
        </div>
      {/each}
      <button
        class="flex items-center px-3 py-2 text-foreground-muted hover:text-foreground"
        onclick={() => { hostOpen = false; containerOpen = false; flashStrip(); }}
        title="New tab — pick a connection in the toolbar above"
        aria-label="New terminal tab"
      >
        <Plus class="w-4 h-4" />
      </button>
    </div>
  {/if}

  <!-- Terminal area -->
  <div class="flex-1 overflow-hidden relative">
    {#if !splitTree}
      <!-- Empty state -->
      <div class="absolute inset-0 flex flex-col items-center justify-center gap-6 text-foreground-muted">
        <SquareTerminal class="w-16 h-16 opacity-20" />
        <div class="text-center">
          <p class="text-lg font-medium text-foreground-muted">Select a container or host above to open a shell</p>
          <p class="text-sm mt-1 text-foreground-muted/60">Hover over an open pane to split it horizontally or vertically</p>
        </div>
        <div class="flex gap-3">
          <button class="btn btn-primary flex items-center gap-2" onclick={openContainerShell} disabled={!selectedContainerId}>
            <SquareTerminal class="w-4 h-4" />
            Open Shell
          </button>
          <button class="btn btn-ghost flex items-center gap-2" onclick={openHostSSH} disabled={!selectedHostId}>
            <Server class="w-4 h-4" />
            SSH Host
          </button>
        </div>
      </div>
    {:else}
      <div class="absolute inset-0">
        <SplitPane
          node={splitTree}
          tabs={tabsMap}
          {tabStatuses}
          {activePaneId}
          onPaneFocus={handlePaneFocus}
          onRatioChange={handleRatioChange}
          onSplitH={handleSplitH}
          onSplitV={handleSplitV}
          onClose={handleClosePane}
          onStatusChange={handleStatusChange}
        />
      </div>
    {/if}
  </div>
</div>

<style>
  @media (min-width: 1024px) {
    .shell-page-root {
      left: var(--sidebar-w, 16rem);
      transition: left 300ms ease;
    }
  }

  .scrollbar-hide {
    scrollbar-width: none;
    -ms-overflow-style: none;
  }
  .scrollbar-hide::-webkit-scrollbar {
    display: none;
  }
</style>
