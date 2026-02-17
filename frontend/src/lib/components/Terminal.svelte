<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import {
    X,
    Maximize2,
    Minimize2,
    GripHorizontal,
    Search,
    RefreshCw,
    Palette,
    ChevronUp,
    ChevronDown,
    Copy,
    Download,
    Settings,
    Zap,
  } from "lucide-svelte";
  import type { Container, Host } from "$lib/api/docker";
  import { createHostTerminalConnection, createTerminalWebSocket } from "$lib/api/docker";
  import { language } from "$lib/stores/docker";

  let {
    container,
    host,
    mode = "container",
    terminalMode = "popup",
    active = true,
    onClose,
    onStatusChange,
  }: {
    container?: Container;
    host?: Host;
    mode?: "container" | "host";
    terminalMode?: "popup" | "embedded";
    active?: boolean;
    onClose?: () => void;
    onStatusChange?: (status: "connecting" | "connected" | "disconnected" | "error") => void;
  } = $props();

  let isEmbedded = $derived(terminalMode === "embedded");
  let isHostMode = $derived(mode === "host");
  const targetName = $derived(
    isHostMode ? host?.name || "Host" : container?.name || "Container",
  );
  const targetHostId = $derived(
    isHostMode ? host?.id || "" : container?.hostId || "",
  );

  let terminalElement: HTMLDivElement;
  let terminal: any;
  let fitAddon: any;
  let searchAddon: any;
  let webglAddon: any;
  let webLinksAddon: any;
  let ws: WebSocket | null = null;
  let isFullscreen = $state(false);
  let isConnecting = $state(true);
  let error = $state<string | null>(null);
  let connectionStatus = $state<
    "connecting" | "connected" | "disconnected" | "error"
  >("connecting");

  // Search state
  let showSearch = $state(false);
  let searchQuery = $state("");
  let searchResults = $state({ current: 0, total: 0 });
  let searchInput: HTMLInputElement | undefined = $state();

  // Theme state
  let showThemePicker = $state(false);
  let currentTheme = $state("tokyo-night");

  // Dragging state
  let isDragging = $state(false);
  let dragOffset = $state({ x: 0, y: 0 });
  let position = $state({ x: 0, y: 0 });
  let windowElement: HTMLDivElement;

  // Reconnection state
  let reconnectAttempts = $state(0);
  const maxReconnectAttempts = 5;
  let reconnectTimeout: ReturnType<typeof setTimeout> | null = null;

  // Translations
  const translations = {
    en: {
      connecting: "Connecting...",
      connected: "Connected",
      disconnected: "Disconnected",
      reconnecting: "Reconnecting",
      error: "Error",
      search: "Search",
      searchPlaceholder: "Search in terminal...",
      noResults: "No results",
      theme: "Theme",
      reconnect: "Reconnect",
      copyOutput: "Copy output",
      downloadOutput: "Download output",
      close: "Close",
      clearTerminal: "Clear terminal",
      fontSize: "Font size",
      connectionClosed: "Connection closed",
      reconnectIn: "Reconnecting in",
      seconds: "s",
      attempt: "Attempt",
    },
    es: {
      connecting: "Conectando...",
      connected: "Conectado",
      disconnected: "Desconectado",
      reconnecting: "Reconectando",
      error: "Error",
      search: "Buscar",
      searchPlaceholder: "Buscar en terminal...",
      noResults: "Sin resultados",
      theme: "Tema",
      reconnect: "Reconectar",
      copyOutput: "Copiar salida",
      downloadOutput: "Descargar salida",
      close: "Cerrar",
      clearTerminal: "Limpiar terminal",
      fontSize: "Tamaño de fuente",
      connectionClosed: "Conexión cerrada",
      reconnectIn: "Reconectando en",
      seconds: "s",
      attempt: "Intento",
    },
  };

  let t = $derived(translations[$language] || translations.en);

  // Terminal themes
  const themes: Record<
    string,
    { name: string; theme: any; background: string }
  > = {
    "tokyo-night": {
      name: "Tokyo Night",
      background: "#1a1b26",
      theme: {
        background: "#1a1b26",
        foreground: "#c0caf5",
        cursor: "#c0caf5",
        cursorAccent: "#1a1b26",
        selectionBackground: "#33467c",
        black: "#15161e",
        red: "#f7768e",
        green: "#9ece6a",
        yellow: "#e0af68",
        blue: "#7aa2f7",
        magenta: "#bb9af7",
        cyan: "#7dcfff",
        white: "#a9b1d6",
        brightBlack: "#414868",
        brightRed: "#f7768e",
        brightGreen: "#9ece6a",
        brightYellow: "#e0af68",
        brightBlue: "#7aa2f7",
        brightMagenta: "#bb9af7",
        brightCyan: "#7dcfff",
        brightWhite: "#c0caf5",
      },
    },
    dracula: {
      name: "Dracula",
      background: "#282a36",
      theme: {
        background: "#282a36",
        foreground: "#f8f8f2",
        cursor: "#f8f8f2",
        cursorAccent: "#282a36",
        selectionBackground: "#44475a",
        black: "#21222c",
        red: "#ff5555",
        green: "#50fa7b",
        yellow: "#f1fa8c",
        blue: "#bd93f9",
        magenta: "#ff79c6",
        cyan: "#8be9fd",
        white: "#f8f8f2",
        brightBlack: "#6272a4",
        brightRed: "#ff6e6e",
        brightGreen: "#69ff94",
        brightYellow: "#ffffa5",
        brightBlue: "#d6acff",
        brightMagenta: "#ff92df",
        brightCyan: "#a4ffff",
        brightWhite: "#ffffff",
      },
    },
    monokai: {
      name: "Monokai",
      background: "#272822",
      theme: {
        background: "#272822",
        foreground: "#f8f8f2",
        cursor: "#f8f8f0",
        cursorAccent: "#272822",
        selectionBackground: "#49483e",
        black: "#272822",
        red: "#f92672",
        green: "#a6e22e",
        yellow: "#f4bf75",
        blue: "#66d9ef",
        magenta: "#ae81ff",
        cyan: "#a1efe4",
        white: "#f8f8f2",
        brightBlack: "#75715e",
        brightRed: "#f92672",
        brightGreen: "#a6e22e",
        brightYellow: "#f4bf75",
        brightBlue: "#66d9ef",
        brightMagenta: "#ae81ff",
        brightCyan: "#a1efe4",
        brightWhite: "#f9f8f5",
      },
    },
    nord: {
      name: "Nord",
      background: "#2e3440",
      theme: {
        background: "#2e3440",
        foreground: "#d8dee9",
        cursor: "#d8dee9",
        cursorAccent: "#2e3440",
        selectionBackground: "#434c5e",
        black: "#3b4252",
        red: "#bf616a",
        green: "#a3be8c",
        yellow: "#ebcb8b",
        blue: "#81a1c1",
        magenta: "#b48ead",
        cyan: "#88c0d0",
        white: "#e5e9f0",
        brightBlack: "#4c566a",
        brightRed: "#bf616a",
        brightGreen: "#a3be8c",
        brightYellow: "#ebcb8b",
        brightBlue: "#81a1c1",
        brightMagenta: "#b48ead",
        brightCyan: "#8fbcbb",
        brightWhite: "#eceff4",
      },
    },
    "github-dark": {
      name: "GitHub Dark",
      background: "#0d1117",
      theme: {
        background: "#0d1117",
        foreground: "#c9d1d9",
        cursor: "#c9d1d9",
        cursorAccent: "#0d1117",
        selectionBackground: "#264f78",
        black: "#161b22",
        red: "#ff7b72",
        green: "#7ee787",
        yellow: "#d29922",
        blue: "#79c0ff",
        magenta: "#d2a8ff",
        cyan: "#a5d6ff",
        white: "#f0f6fc",
        brightBlack: "#484f58",
        brightRed: "#ffa198",
        brightGreen: "#7ee787",
        brightYellow: "#e3b341",
        brightBlue: "#a5d6ff",
        brightMagenta: "#d2a8ff",
        brightCyan: "#a5d6ff",
        brightWhite: "#f0f6fc",
      },
    },
    "catppuccin-mocha": {
      name: "Catppuccin Mocha",
      background: "#1e1e2e",
      theme: {
        background: "#1e1e2e",
        foreground: "#cdd6f4",
        cursor: "#f5e0dc",
        cursorAccent: "#1e1e2e",
        selectionBackground: "#585b70",
        black: "#45475a",
        red: "#f38ba8",
        green: "#a6e3a1",
        yellow: "#f9e2af",
        blue: "#89b4fa",
        magenta: "#f5c2e7",
        cyan: "#94e2d5",
        white: "#bac2de",
        brightBlack: "#585b70",
        brightRed: "#f38ba8",
        brightGreen: "#a6e3a1",
        brightYellow: "#f9e2af",
        brightBlue: "#89b4fa",
        brightMagenta: "#f5c2e7",
        brightCyan: "#94e2d5",
        brightWhite: "#a6adc8",
      },
    },
    "one-dark-pro": {
      name: "One Dark Pro",
      background: "#282c34",
      theme: {
        background: "#282c34",
        foreground: "#abb2bf",
        cursor: "#528bff",
        cursorAccent: "#282c34",
        selectionBackground: "#3e4451",
        black: "#282c34",
        red: "#e06c75",
        green: "#98c379",
        yellow: "#e5c07b",
        blue: "#61afef",
        magenta: "#c678dd",
        cyan: "#56b6c2",
        white: "#abb2bf",
        brightBlack: "#5c6370",
        brightRed: "#e06c75",
        brightGreen: "#98c379",
        brightYellow: "#e5c07b",
        brightBlue: "#61afef",
        brightMagenta: "#c678dd",
        brightCyan: "#56b6c2",
        brightWhite: "#ffffff",
      },
    },
  };

  // Font size state
  let fontSize = $state(14);

  async function initTerminal() {
    // Dynamic import for xterm
    const { Terminal } = await import("@xterm/xterm");
    const { FitAddon } = await import("@xterm/addon-fit");
    const { SearchAddon } = await import("@xterm/addon-search");

    // Import CSS
    await import("@xterm/xterm/css/xterm.css");

    // Create terminal with selected theme
    terminal = new Terminal({
      cursorBlink: true,
      fontSize: fontSize,
      fontFamily: "JetBrains Mono, Menlo, Monaco, Consolas, monospace",
      theme: themes[currentTheme].theme,
      scrollback: 10000,
      minimumContrastRatio: 4.5,
      lineHeight: 1.2,
      allowProposedApi: true,
    });

    fitAddon = new FitAddon();
    searchAddon = new SearchAddon();
    terminal.loadAddon(fitAddon);
    terminal.loadAddon(searchAddon);

    // Try to load WebGL renderer for 10-50x performance boost
    try {
      const { WebglAddon } = await import("@xterm/addon-webgl");
      webglAddon = new WebglAddon();
      webglAddon.onContextLoss(() => {
        webglAddon?.dispose();
      });
      terminal.open(terminalElement);
      terminal.loadAddon(webglAddon);
    } catch (e) {
      // WebGL not available, fallback to canvas renderer
      console.warn("WebGL addon not available, using canvas renderer");
      if (!terminal.element) {
        terminal.open(terminalElement);
      }
    }

    if (!terminal.element) {
      terminal.open(terminalElement);
    }

    // Try to load web-links addon for clickable URLs
    try {
      const { WebLinksAddon } = await import("@xterm/addon-web-links");
      webLinksAddon = new WebLinksAddon();
      terminal.loadAddon(webLinksAddon);
    } catch (e) {
      console.warn("Web links addon not available");
    }

    fitAddon.fit();

    // Send input to WebSocket (registered once here, not on every reconnect)
    terminal.onData((data: string) => {
      if (ws?.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: "input", data }));
      }
    });

    // Handle resize (registered once here, not on every reconnect)
    terminal.onResize(({ cols, rows }: { cols: number; rows: number }) => {
      if (ws?.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: "resize", cols, rows }));
      }
    });

    // Keyboard shortcuts
    terminal.attachCustomKeyEventHandler((e: KeyboardEvent) => {
      // Ctrl+F for search
      if (e.ctrlKey && e.key === "f") {
        e.preventDefault();
        toggleSearch();
        return false;
      }
      // Ctrl+L to clear
      if (e.ctrlKey && e.key === "l") {
        e.preventDefault();
        clearTerminal();
        return false;
      }
      // Ctrl+Shift+C for copy
      if (e.ctrlKey && e.shiftKey && e.key === "C") {
        e.preventDefault();
        copyOutput();
        return false;
      }
      // Ctrl+Shift+V for paste
      if (e.ctrlKey && e.shiftKey && e.key === "V") {
        e.preventDefault();
        navigator.clipboard.readText().then((text) => {
          if (ws?.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({ type: "input", data: text }));
          }
        });
        return false;
      }
      return true;
    });

    // Ctrl+Scroll to zoom font size
    terminalElement.addEventListener(
      "wheel",
      (e: WheelEvent) => {
        if (e.ctrlKey) {
          e.preventDefault();
          changeFontSize(e.deltaY < 0 ? 1 : -1);
        }
      },
      { passive: false },
    );
  }

  function connectWebSocket() {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.close();
    }

    connectionStatus = "connecting";
    isConnecting = true;
    error = null;

    try {
      if (isHostMode && host) {
        ws = createHostTerminalConnection(host.id);
      } else if (container) {
        ws = createTerminalWebSocket(container.hostId, container.id);
      } else {
        throw new Error("Missing terminal target");
      }

      ws.onopen = () => {
        isConnecting = false;
        connectionStatus = "connected";
        reconnectAttempts = 0;
        terminal.writeln(
          `\x1b[32m● ${t.connected} to ${targetName}\x1b[0m\r\n`,
        );

        // Send initial resize
        const dims = { cols: terminal.cols, rows: terminal.rows };
        ws?.send(JSON.stringify({ type: "resize", ...dims }));
      };

      ws.onmessage = (event) => {
        const data = JSON.parse(event.data);
        if (data.type === "output") {
          terminal.write(data.data);
        } else if (data.type === "error") {
          terminal.writeln(`\x1b[31mError: ${data.data}\x1b[0m`);
        }
      };

      ws.onerror = () => {
        connectionStatus = "error";
        error = "Connection error";
        isConnecting = false;
      };

      ws.onclose = () => {
        connectionStatus = "disconnected";
        terminal.writeln(`\r\n\x1b[33m● ${t.connectionClosed}\x1b[0m`);

        // Auto-reconnect logic
        if (reconnectAttempts < maxReconnectAttempts) {
          const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 10000);
          reconnectAttempts++;
          terminal.writeln(
            `\x1b[33m  ${t.reconnectIn} ${delay / 1000}${t.seconds} (${t.attempt} ${reconnectAttempts}/${maxReconnectAttempts})\x1b[0m`,
          );

          reconnectTimeout = setTimeout(() => {
            terminal.writeln(`\x1b[36m● ${t.reconnecting}...\x1b[0m`);
            connectWebSocket();
          }, delay);
        }
      };

    } catch (e) {
      error = "Failed to connect";
      connectionStatus = "error";
      isConnecting = false;
    }
  }

  onMount(() => {
    let isMounted = true;

    void (async () => {
      await initTerminal();
      if (!isMounted) return;
      connectWebSocket();
    })();

    // Window resize handler
    const resizeObserver = new ResizeObserver(() => {
      fitAddon?.fit();
    });
    resizeObserver.observe(terminalElement);

    return () => {
      isMounted = false;
      resizeObserver.disconnect();
    };
  });

  onDestroy(() => {
    if (reconnectTimeout) {
      clearTimeout(reconnectTimeout);
    }
    ws?.close();
    webglAddon?.dispose();
    webLinksAddon?.dispose();
    terminal?.dispose();
  });

  function toggleFullscreen() {
    isFullscreen = !isFullscreen;
    if (isFullscreen) {
      position = { x: 0, y: 0 };
    }
    setTimeout(() => fitAddon?.fit(), 100);
  }

  function toggleSearch() {
    showSearch = !showSearch;
    if (showSearch) {
      setTimeout(() => searchInput?.focus(), 50);
    } else {
      searchAddon?.clearDecorations();
    }
  }

  function doSearch() {
    if (!searchQuery) {
      searchResults = { current: 0, total: 0 };
      searchAddon?.clearDecorations();
      return;
    }
    searchAddon?.findNext(searchQuery, {
      caseSensitive: false,
      incremental: true,
      decorations: { matchOverviewRuler: "#7aa2f7" },
    });
    // Note: xterm search addon doesn't provide count, we track via find operations
  }

  function searchNext() {
    searchAddon?.findNext(searchQuery);
  }

  function searchPrev() {
    searchAddon?.findPrevious(searchQuery);
  }

  function applyTheme(themeId: string) {
    currentTheme = themeId;
    terminal?.options && (terminal.options.theme = themes[themeId].theme);
    showThemePicker = false;
  }

  function changeFontSize(delta: number) {
    fontSize = Math.max(10, Math.min(24, fontSize + delta));
    if (terminal) {
      terminal.options.fontSize = fontSize;
      fitAddon?.fit();
    }
  }

  function clearTerminal() {
    terminal?.clear();
  }

  function copyOutput() {
    const selection = terminal?.getSelection() || "";
    if (selection) {
      navigator.clipboard.writeText(selection);
    }
  }

  function downloadOutput() {
    // Get terminal buffer content
    const buffer = terminal?.buffer?.active;
    if (!buffer) return;

    let content = "";
    for (let i = 0; i < buffer.length; i++) {
      const line = buffer.getLine(i);
      if (line) {
        content += line.translateToString(true) + "\n";
      }
    }

    const blob = new Blob([content], { type: "text/plain" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `terminal_${targetName}_${new Date().toISOString().slice(0, 10)}.txt`;
    a.click();
    URL.revokeObjectURL(url);
  }

  function manualReconnect() {
    reconnectAttempts = 0;
    if (reconnectTimeout) {
      clearTimeout(reconnectTimeout);
      reconnectTimeout = null;
    }
    terminal.writeln(`\x1b[36m● ${t.reconnecting}...\x1b[0m`);
    connectWebSocket();
  }

  function startDrag(e: MouseEvent) {
    if (isFullscreen) return;
    isDragging = true;
    const rect = windowElement.getBoundingClientRect();
    dragOffset = {
      x: e.clientX - rect.left,
      y: e.clientY - rect.top,
    };
    document.addEventListener("mousemove", onDrag);
    document.addEventListener("mouseup", stopDrag);
  }

  function onDrag(e: MouseEvent) {
    if (!isDragging) return;
    position = {
      x: e.clientX - dragOffset.x,
      y: e.clientY - dragOffset.y,
    };
  }

  function stopDrag() {
    isDragging = false;
    document.removeEventListener("mousemove", onDrag);
    document.removeEventListener("mouseup", stopDrag);
  }

  // Re-fit terminal when tab becomes active (embedded mode)
  $effect(() => {
    if (active && fitAddon) {
      setTimeout(() => fitAddon?.fit(), 50);
    }
  });

  // Notify parent when connection status changes
  $effect(() => {
    onStatusChange?.(connectionStatus);
  });

  // Connection status color
  let statusColor = $derived(
    connectionStatus === "connected"
      ? "text-running"
      : connectionStatus === "connecting"
        ? "text-primary"
        : connectionStatus === "disconnected"
          ? "text-paused"
          : "text-stopped",
  );
</script>

<div
  bind:this={windowElement}
  class:fixed={!isEmbedded}
  class:z-50={!isEmbedded}
  class:rounded-lg={!isEmbedded}
  class:shadow-2xl={!isEmbedded}
  class:h-full={isEmbedded}
  class="bg-background-secondary border border-border flex flex-col overflow-hidden"
  class:inset-4={!isEmbedded && isFullscreen}
  class:w-[800px]={!isEmbedded && !isFullscreen}
  class:h-[500px]={!isEmbedded && !isFullscreen}
  style={!isEmbedded && !isFullscreen && (position.x !== 0 || position.y !== 0)
    ? `left: ${position.x}px; top: ${position.y}px;`
    : !isEmbedded && !isFullscreen
      ? "bottom: 1rem; right: 1rem;"
      : ""}
>
  <!-- Header (Draggable) -->
  <div
    role="toolbar"
    tabindex="0"
    class="flex items-center justify-between px-4 py-2 bg-background-tertiary border-b border-border select-none"
    class:cursor-grab={!isEmbedded && !isFullscreen && !isDragging}
    class:cursor-grabbing={!isEmbedded && isDragging}
    onmousedown={isEmbedded ? undefined : startDrag}
  >
    <div class="flex items-center gap-3">
      {#if !isFullscreen && !isEmbedded}
        <GripHorizontal class="w-4 h-4 text-foreground-muted" />
      {/if}
      <div class="flex gap-1.5">
        <span class="w-3 h-3 rounded-full bg-accent-red"></span>
        <span class="w-3 h-3 rounded-full bg-paused"></span>
        <span class="w-3 h-3 rounded-full bg-running"></span>
      </div>
      <span class="text-sm font-medium text-foreground flex items-center gap-2">
        <Zap class="w-3.5 h-3.5 {statusColor}" />
        {targetName}
        {#if targetHostId}
          <span class="text-foreground-muted">@{targetHostId}</span>
        {/if}
      </span>
    </div>

    <div class="flex items-center gap-1">
      <!-- Search button -->
      <button
        class="btn-icon {showSearch ? 'text-primary' : ''}"
        onclick={toggleSearch}
        title="{t.search} (Ctrl+F)"
      >
        <Search class="w-4 h-4" />
      </button>

      <!-- Theme picker -->
      <div class="relative">
        <button
          class="btn-icon {showThemePicker ? 'text-primary' : ''}"
          onclick={() => (showThemePicker = !showThemePicker)}
          title={t.theme}
        >
          <Palette class="w-4 h-4" />
        </button>

        {#if showThemePicker}
          <div
            class="absolute right-0 top-full mt-1 bg-background-secondary border border-border rounded-lg shadow-xl z-10 py-1 min-w-[140px]"
          >
            {#each Object.entries(themes) as [id, theme]}
              <button
                class="w-full px-3 py-1.5 text-sm text-left hover:bg-background-tertiary flex items-center gap-2 {currentTheme ===
                id
                  ? 'text-primary'
                  : 'text-foreground'}"
                onclick={() => applyTheme(id)}
              >
                <span
                  class="w-3 h-3 rounded-full"
                  style="background: {theme.background}; border: 1px solid var(--border);"
                ></span>
                {theme.name}
              </button>
            {/each}
          </div>
        {/if}
      </div>

      <!-- Font size controls -->
      <div class="flex items-center border-l border-border pl-1 ml-1">
        <button
          class="btn-icon"
          onclick={() => changeFontSize(-1)}
          title={t.fontSize}
        >
          <span class="text-xs font-bold">A-</span>
        </button>
        <span class="text-xs text-foreground-muted w-6 text-center"
          >{fontSize}</span
        >
        <button
          class="btn-icon"
          onclick={() => changeFontSize(1)}
          title={t.fontSize}
        >
          <span class="text-xs font-bold">A+</span>
        </button>
      </div>

      <!-- Copy & Download -->
      <button class="btn-icon" onclick={copyOutput} title={t.copyOutput}>
        <Copy class="w-4 h-4" />
      </button>
      <button
        class="btn-icon"
        onclick={downloadOutput}
        title={t.downloadOutput}
      >
        <Download class="w-4 h-4" />
      </button>

      <!-- Reconnect -->
      {#if connectionStatus === "disconnected" || connectionStatus === "error"}
        <button
          class="btn-icon text-primary"
          onclick={manualReconnect}
          title={t.reconnect}
        >
          <RefreshCw class="w-4 h-4" />
        </button>
      {/if}

      <!-- Fullscreen (popup mode only) -->
      {#if !isEmbedded}
      <button
        class="btn-icon"
        onclick={toggleFullscreen}
        title={isFullscreen ? "Minimize" : "Maximize"}
      >
        {#if isFullscreen}
          <Minimize2 class="w-4 h-4" />
        {:else}
          <Maximize2 class="w-4 h-4" />
        {/if}
      </button>
      {/if}
      {#if !isEmbedded && onClose}
      <button
        class="btn-icon hover:text-accent-red"
        onclick={onClose}
        title={t.close}
      >
        <X class="w-4 h-4" />
      </button>
      {/if}
    </div>
  </div>

  <!-- Search bar -->
  {#if showSearch}
    <div
      class="flex items-center gap-2 px-4 py-2 bg-background border-b border-border"
    >
      <Search class="w-4 h-4 text-foreground-muted" />
      <input
        bind:this={searchInput}
        type="text"
        bind:value={searchQuery}
        oninput={doSearch}
        onkeydown={(e) => {
          if (e.key === "Enter") {
            e.shiftKey ? searchPrev() : searchNext();
          } else if (e.key === "Escape") {
            toggleSearch();
          }
        }}
        placeholder={t.searchPlaceholder}
        class="flex-1 bg-transparent text-sm text-foreground placeholder:text-foreground-muted focus:outline-none"
      />
      <div class="flex items-center gap-1">
        <button class="btn-icon" onclick={searchPrev} title="Previous">
          <ChevronUp class="w-4 h-4" />
        </button>
        <button class="btn-icon" onclick={searchNext} title="Next">
          <ChevronDown class="w-4 h-4" />
        </button>
      </div>
      <button class="btn-icon hover:text-accent-red" onclick={toggleSearch}>
        <X class="w-4 h-4" />
      </button>
    </div>
  {/if}

  <!-- Terminal -->
  <div
    class="flex-1 p-2 relative"
    style="background: {themes[currentTheme].background};"
  >
    {#if isConnecting}
      <div
        class="absolute inset-0 flex items-center justify-center"
        style="background: {themes[currentTheme].background};"
      >
        <div class="flex items-center gap-3 text-foreground-muted">
          <div
            class="animate-spin w-5 h-5 border-2 border-primary border-t-transparent rounded-full"
          ></div>
          <span>{t.connecting}</span>
        </div>
      </div>
    {/if}

    {#if error}
      <div
        class="absolute inset-0 flex items-center justify-center"
        style="background: {themes[currentTheme].background};"
      >
        <div class="text-center">
          <p class="text-accent-red mb-2">{error}</p>
          <div class="flex gap-2 justify-center">
            <button class="btn btn-primary text-sm" onclick={manualReconnect}>
              <RefreshCw class="w-4 h-4 mr-1" />
              {t.reconnect}
            </button>
            {#if onClose}
            <button class="btn btn-ghost text-sm" onclick={onClose}>
              {t.close}
            </button>
            {/if}
          </div>
        </div>
      </div>
    {/if}

    <div bind:this={terminalElement} class="h-full"></div>
  </div>

  <!-- Status bar -->
  <div
    class="px-4 py-1.5 bg-background-tertiary border-t border-border flex items-center justify-between text-xs"
  >
    <div class="flex items-center gap-3">
      <span class="flex items-center gap-1.5 {statusColor}">
        <span
          class="w-2 h-2 rounded-full {connectionStatus === 'connected'
            ? 'bg-running'
            : connectionStatus === 'connecting'
              ? 'bg-primary animate-pulse'
              : connectionStatus === 'disconnected'
                ? 'bg-paused'
                : 'bg-stopped'}"
        ></span>
        {connectionStatus === "connected"
          ? t.connected
          : connectionStatus === "connecting"
            ? t.connecting
            : connectionStatus === "disconnected"
              ? t.disconnected
              : t.error}
      </span>
      <span class="text-foreground-muted">|</span>
      <span class="text-foreground-muted">{themes[currentTheme].name}</span>
    </div>
    <div class="flex items-center gap-2 text-foreground-muted">
      <span>Ctrl+F: {t.search}</span>
      <span>|</span>
      <span>Ctrl+L: {t.clearTerminal}</span>
    </div>
  </div>
</div>

<!-- Click outside to close theme picker -->
{#if showThemePicker}
  <button
    class="fixed inset-0 z-40 cursor-default"
    onclick={() => (showThemePicker = false)}
    aria-label="Close theme picker"
  ></button>
{/if}
