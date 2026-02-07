<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { X, Maximize2, Minimize2, GripHorizontal } from "lucide-svelte";
  import type { Container } from "$lib/api/docker";
  import { createTerminalWebSocket } from "$lib/api/docker";

  let { container, onClose }: { container: Container; onClose: () => void } =
    $props();

  let terminalElement: HTMLDivElement;
  let terminal: any;
  let fitAddon: any;
  let ws: WebSocket | null = null;
  let isFullscreen = $state(false);
  let isConnecting = $state(true);
  let error = $state<string | null>(null);

  // Dragging state
  let isDragging = $state(false);
  let dragOffset = $state({ x: 0, y: 0 });
  let position = $state({ x: 0, y: 0 });
  let windowElement: HTMLDivElement;

  onMount(async () => {
    // Dynamic import for xterm
    const { Terminal } = await import("@xterm/xterm");
    const { FitAddon } = await import("@xterm/addon-fit");

    // Import CSS
    await import("@xterm/xterm/css/xterm.css");

    // Create terminal with Tokyo Night theme
    terminal = new Terminal({
      cursorBlink: true,
      fontSize: 14,
      fontFamily: "JetBrains Mono, Menlo, Monaco, Consolas, monospace",
      theme: {
        background: "#1a1b26",
        foreground: "#c0caf5",
        cursor: "#c0caf5",
        cursorAccent: "#1a1b26",
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
      scrollback: 1000,
      allowProposedApi: true,
    });

    fitAddon = new FitAddon();
    terminal.loadAddon(fitAddon);

    terminal.open(terminalElement);
    fitAddon.fit();

    // Connect WebSocket
    try {
      ws = createTerminalWebSocket(container.hostId, container.id);

      ws.onopen = () => {
        isConnecting = false;
        terminal.writeln(`\x1b[32mConnected to ${container.name}\x1b[0m\r\n`);

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
        error = "Connection error";
        isConnecting = false;
      };

      ws.onclose = () => {
        terminal.writeln("\r\n\x1b[33mConnection closed\x1b[0m");
      };

      // Send input to WebSocket
      terminal.onData((data: string) => {
        if (ws?.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ type: "input", data }));
        }
      });

      // Handle resize
      terminal.onResize(({ cols, rows }: { cols: number; rows: number }) => {
        if (ws?.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ type: "resize", cols, rows }));
        }
      });
    } catch (e) {
      error = "Failed to connect";
      isConnecting = false;
    }

    // Window resize handler
    const resizeObserver = new ResizeObserver(() => {
      fitAddon?.fit();
    });
    resizeObserver.observe(terminalElement);

    return () => {
      resizeObserver.disconnect();
    };
  });

  onDestroy(() => {
    ws?.close();
    terminal?.dispose();
  });

  function toggleFullscreen() {
    isFullscreen = !isFullscreen;
    if (isFullscreen) {
      position = { x: 0, y: 0 };
    }
    setTimeout(() => fitAddon?.fit(), 100);
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
</script>

<div
  bind:this={windowElement}
  class="fixed z-50 bg-background-secondary border border-border rounded-lg shadow-2xl flex flex-col overflow-hidden"
  class:inset-4={isFullscreen}
  class:w-[700px]={!isFullscreen}
  class:h-[400px]={!isFullscreen}
  style={!isFullscreen && (position.x !== 0 || position.y !== 0)
    ? `left: ${position.x}px; top: ${position.y}px;`
    : !isFullscreen
      ? "bottom: 1rem; right: 1rem;"
      : ""}
>
  <!-- Header (Draggable) -->
  <div
    class="flex items-center justify-between px-4 py-2 bg-background-tertiary border-b border-border select-none"
    class:cursor-grab={!isFullscreen && !isDragging}
    class:cursor-grabbing={isDragging}
    onmousedown={startDrag}
  >
    <div class="flex items-center gap-3">
      {#if !isFullscreen}
        <GripHorizontal class="w-4 h-4 text-foreground-muted" />
      {/if}
      <div class="flex gap-1.5">
        <span class="w-3 h-3 rounded-full bg-accent-red"></span>
        <span class="w-3 h-3 rounded-full bg-paused"></span>
        <span class="w-3 h-3 rounded-full bg-running"></span>
      </div>
      <span class="text-sm font-medium text-foreground">
        {container.name}
        <span class="text-foreground-muted">@{container.hostId}</span>
      </span>
    </div>

    <div class="flex items-center gap-1">
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
      <button
        class="btn-icon hover:text-accent-red"
        onclick={onClose}
        title="Close"
      >
        <X class="w-4 h-4" />
      </button>
    </div>
  </div>

  <!-- Terminal -->
  <div class="flex-1 p-2 bg-[#1a1b26] relative">
    {#if isConnecting}
      <div
        class="absolute inset-0 flex items-center justify-center bg-[#1a1b26]"
      >
        <div class="flex items-center gap-3 text-foreground-muted">
          <div
            class="animate-spin w-5 h-5 border-2 border-primary border-t-transparent rounded-full"
          ></div>
          <span>Connecting...</span>
        </div>
      </div>
    {/if}

    {#if error}
      <div
        class="absolute inset-0 flex items-center justify-center bg-[#1a1b26]"
      >
        <div class="text-center">
          <p class="text-accent-red mb-2">{error}</p>
          <button class="btn btn-primary text-sm" onclick={onClose}
            >Close</button
          >
        </div>
      </div>
    {/if}

    <div bind:this={terminalElement} class="h-full"></div>
  </div>
</div>
