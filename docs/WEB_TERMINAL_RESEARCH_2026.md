# Web-Based SSH Terminal Best Practices (2025-2026)

## Research for Dockerverse — SvelteKit + xterm.js

> **xterm.js v6.0.0** (released Dec 2025) — latest stable; used by VS Code, Hyper, Tabby, Azure Cloud Shell, Portainer, Proxmox VE, and 100+ production apps.

---

## 1. xterm.js Addons — Complete Reference

All official addons use the `@xterm/addon-*` namespace (v6.x):

| Addon | Package | Purpose | Priority |
|-------|---------|---------|----------|
| **Fit** | `@xterm/addon-fit` | Auto-resize terminal to container | **Essential** |
| **WebGL** | `@xterm/addon-webgl` | GPU-accelerated WebGL2 renderer | **Essential** |
| **Search** | `@xterm/addon-search` | In-buffer search with decorations | **Essential** |
| **Web Links** | `@xterm/addon-web-links` | Clickable URL detection | **Recommended** |
| **Attach** | `@xterm/addon-attach` | WebSocket attachment to backend process | **Recommended** |
| **Clipboard** | `@xterm/addon-clipboard` | Browser clipboard access (OSC 52) | **Recommended** |
| **Web Fonts** | `@xterm/addon-web-fonts` | Easy web font integration | **Recommended** |
| **Unicode11** | `@xterm/addon-unicode11` | Proper CJK/emoji widths | Nice-to-have |
| **Unicode Graphemes** | `@xterm/addon-unicode-graphemes` | Grapheme clustering (experimental) | Nice-to-have |
| **Image** | `@xterm/addon-image` | Inline terminal images (iTerm2/Sixel protocol) | Nice-to-have |
| **Serialize** | `@xterm/addon-serialize` | Export buffer to VT sequences or HTML | Nice-to-have |
| **Ligatures** | `@xterm/addon-ligatures` | Font ligature rendering | Nice-to-have |
| **Progress** | `@xterm/addon-progress` | Progress bar API (OSC 9;4) | Nice-to-have |

### Installation (all recommended addons)

```bash
npm install @xterm/xterm @xterm/addon-fit @xterm/addon-webgl @xterm/addon-search \
  @xterm/addon-web-links @xterm/addon-clipboard @xterm/addon-web-fonts @xterm/addon-attach
```

---

## 2. Terminal Themes — Exact Color Values

### Theme 1: Tokyo Night (Default recommendation for Docker dashboards)

```typescript
const tokyoNight = {
  background: '#1a1b26',
  foreground: '#c0caf5',
  cursor: '#c0caf5',
  cursorAccent: '#1a1b26',
  selectionBackground: '#33467c',
  selectionForeground: '#c0caf5',
  selectionInactiveBackground: '#283457',
  black: '#15161e',
  red: '#f7768e',
  green: '#9ece6a',
  yellow: '#e0af68',
  blue: '#7aa2f7',
  magenta: '#bb9af7',
  cyan: '#7dcfff',
  white: '#a9b1d6',
  brightBlack: '#414868',
  brightRed: '#f7768e',
  brightGreen: '#9ece6a',
  brightYellow: '#e0af68',
  brightBlue: '#7aa2f7',
  brightMagenta: '#bb9af7',
  brightCyan: '#7dcfff',
  brightWhite: '#c0caf5',
};
```

### Theme 2: Dracula

```typescript
const dracula = {
  background: '#282a36',
  foreground: '#f8f8f2',
  cursor: '#f8f8f2',
  cursorAccent: '#282a36',
  selectionBackground: '#44475a',
  selectionForeground: '#f8f8f2',
  selectionInactiveBackground: '#3a3d4b',
  black: '#21222c',
  red: '#ff5555',
  green: '#50fa7b',
  yellow: '#f1fa8c',
  blue: '#bd93f9',
  magenta: '#ff79c6',
  cyan: '#8be9fd',
  white: '#f8f8f2',
  brightBlack: '#6272a4',
  brightRed: '#ff6e6e',
  brightGreen: '#69ff94',
  brightYellow: '#ffffa5',
  brightBlue: '#d6acff',
  brightMagenta: '#ff92df',
  brightCyan: '#a4ffff',
  brightWhite: '#ffffff',
};
```

### Theme 3: Catppuccin Mocha

```typescript
const catppuccinMocha = {
  background: '#1e1e2e',
  foreground: '#cdd6f4',
  cursor: '#f5e0dc',
  cursorAccent: '#1e1e2e',
  selectionBackground: '#585b70',
  selectionForeground: '#cdd6f4',
  selectionInactiveBackground: '#45475a',
  black: '#45475a',
  red: '#f38ba8',
  green: '#a6e3a1',
  yellow: '#f9e2af',
  blue: '#89b4fa',
  magenta: '#f5c2e7',
  cyan: '#94e2d5',
  white: '#bac2de',
  brightBlack: '#585b70',
  brightRed: '#f38ba8',
  brightGreen: '#a6e3a1',
  brightYellow: '#f9e2af',
  brightBlue: '#89b4fa',
  brightMagenta: '#f5c2e7',
  brightCyan: '#94e2d5',
  brightWhite: '#a6adc8',
};
```

### Theme 4: Nord

```typescript
const nord = {
  background: '#2e3440',
  foreground: '#d8dee9',
  cursor: '#d8dee9',
  cursorAccent: '#2e3440',
  selectionBackground: '#434c5e',
  selectionForeground: '#d8dee9',
  selectionInactiveBackground: '#3b4252',
  black: '#3b4252',
  red: '#bf616a',
  green: '#a3be8c',
  yellow: '#ebcb8b',
  blue: '#81a1c1',
  magenta: '#b48ead',
  cyan: '#88c0d0',
  white: '#e5e9f0',
  brightBlack: '#4c566a',
  brightRed: '#bf616a',
  brightGreen: '#a3be8c',
  brightYellow: '#ebcb8b',
  brightBlue: '#81a1c1',
  brightMagenta: '#b48ead',
  brightCyan: '#8fbcbb',
  brightWhite: '#eceff4',
};
```

### Theme 5: GitHub Dark

```typescript
const githubDark = {
  background: '#0d1117',
  foreground: '#c9d1d9',
  cursor: '#c9d1d9',
  cursorAccent: '#0d1117',
  selectionBackground: '#264f78',
  selectionForeground: '#ffffff',
  selectionInactiveBackground: '#1c3a5c',
  black: '#161b22',
  red: '#ff7b72',
  green: '#7ee787',
  yellow: '#d29922',
  blue: '#79c0ff',
  magenta: '#d2a8ff',
  cyan: '#a5d6ff',
  white: '#f0f6fc',
  brightBlack: '#484f58',
  brightRed: '#ffa198',
  brightGreen: '#7ee787',
  brightYellow: '#e3b341',
  brightBlue: '#a5d6ff',
  brightMagenta: '#d2a8ff',
  brightCyan: '#a5d6ff',
  brightWhite: '#f0f6fc',
};
```

### Theme 6: One Dark Pro

```typescript
const oneDarkPro = {
  background: '#282c34',
  foreground: '#abb2bf',
  cursor: '#528bff',
  cursorAccent: '#282c34',
  selectionBackground: '#3e4452',
  selectionForeground: '#abb2bf',
  selectionInactiveBackground: '#353b45',
  black: '#282c34',
  red: '#e06c75',
  green: '#98c379',
  yellow: '#e5c07b',
  blue: '#61afef',
  magenta: '#c678dd',
  cyan: '#56b6c2',
  white: '#abb2bf',
  brightBlack: '#5c6370',
  brightRed: '#e06c75',
  brightGreen: '#98c379',
  brightYellow: '#e5c07b',
  brightBlue: '#61afef',
  brightMagenta: '#c678dd',
  brightCyan: '#56b6c2',
  brightWhite: '#ffffff',
};
```

---

## 3. Optimal Terminal Configuration Object

```typescript
import { Terminal } from '@xterm/xterm';

const terminal = new Terminal({
  // === Core ===
  cursorBlink: true,
  cursorStyle: 'bar',              // Modern look: 'block' | 'underline' | 'bar'
  cursorWidth: 2,                  // Pixels when cursorStyle='bar'
  
  // === Font ===
  fontSize: 14,
  fontFamily: "'JetBrains Mono', 'Fira Code', 'Cascadia Code', Menlo, Monaco, 'Courier New', monospace",
  fontWeight: '400',
  fontWeightBold: '700',
  letterSpacing: 0,
  lineHeight: 1.2,
  
  // === Buffer ===
  scrollback: 10000,               // Lines kept in scrollback (default 1000)
  
  // === Behavior ===
  allowProposedApi: true,           // Required for some addons
  allowTransparency: false,         // Keep false for perf; true only if you need transparent bg
  convertEol: false,
  altClickMovesCursor: true,        // Alt+click moves cursor in editors like vim
  rightClickSelectsWord: true,
  
  // === Accessibility ===
  screenReaderMode: false,          // Enable for a11y (perf cost)
  minimumContrastRatio: 4.5,        // WCAG AA compliance
  
  // === Theme ===
  theme: tokyoNight,                // See themes above
  
  // === Performance ===
  // fastScrollModifier: 'alt',     // Hold alt for fast scroll (deprecated in v6)
  // fastScrollSensitivity: 5,      // Multiplier for fast scroll (deprecated in v6)
  scrollSensitivity: 1,
  
  // === Advanced ===
  logLevel: 'off',                  // 'trace' | 'debug' | 'info' | 'warn' | 'error' | 'off'
  drawBoldTextInBrightColors: true,
  macOptionIsMeta: false,           // Set true on macOS for proper Meta key
  macOptionClickForcesSelection: false,
});
```

---

## 4. Performance Best Practices

### WebGL Renderer (Critical for Docker dashboards)

The **WebGL addon** renders via a `<canvas>` with WebGL2 context. This is orders of magnitude faster than the DOM renderer for high-throughput outputs (logs, builds, etc.).

```typescript
import { WebglAddon } from '@xterm/addon-webgl';

// Load AFTER terminal.open()
const webglAddon = new WebglAddon();

// Handle context loss gracefully
webglAddon.onContextLoss(() => {
  console.warn('WebGL context lost, falling back to DOM renderer');
  webglAddon.dispose();
  // Optionally reload WebGL after a delay
  setTimeout(() => {
    try {
      const newWebgl = new WebglAddon();
      newWebgl.onContextLoss(() => newWebgl.dispose());
      terminal.loadAddon(newWebgl);
    } catch (e) {
      console.warn('WebGL not available, using DOM renderer');
    }
  }, 1000);
});

terminal.loadAddon(webglAddon);
```

### Performance Checklist

| Practice | Impact | Details |
|----------|--------|---------|
| **Use WebGL renderer** | 10-50x faster | Falls back to DOM if unsupported |
| **Set `allowTransparency: false`** | Significant | Transparent backgrounds disable renderer optimizations |
| **Limit `scrollback`** | Memory | 10,000 lines ≈ 20-40 MB; don't use 100k+ |
| **Batch writes** | Reduces reflows | Combine multiple `write()` calls into one |
| **Debounce `fit()`** | Reduces layout thrash | Don't call on every pixel of resize |
| **Use `requestAnimationFrame`** | Smooth renders | Wrap fit() in rAF during animations |
| **Dispose properly** | Memory leaks | Always call `terminal.dispose()` in `onDestroy` |
| **Avoid `screenReaderMode`** | CPU overhead | Only enable when needed |
| **Set `logLevel: 'off'`** | Micro-optimization | Disables internal console logging |

### Buffer Management

```typescript
// Efficient way to limit memory for long-running sessions
const terminal = new Terminal({ scrollback: 10000 }); // 10k lines

// Clear buffer periodically for log viewers
function clearOldBuffer() {
  // terminal.clear() keeps current viewport, clears scrollback
  terminal.clear();
}

// Download buffer before clearing
function exportBuffer(): string {
  const buffer = terminal.buffer.active;
  let content = '';
  for (let i = 0; i < buffer.length; i++) {
    const line = buffer.getLine(i);
    if (line) content += line.translateToString(true) + '\n';
  }
  return content;
}
```

---

## 5. Essential Terminal Features

### Feature Matrix — What the best web terminals implement

| Feature | Portainer | VS Code | Dockerverse (Current) | Recommended |
|---------|-----------|---------|----------------------|-------------|
| WebGL renderer | ✅ | ✅ | ❌ | **Add** |
| Search (Ctrl+F) | ❌ | ✅ | ✅ | ✅ |
| Fit to container | ✅ | ✅ | ✅ | ✅ |
| Clickable URLs | ✅ | ✅ | ❌ | **Add** |
| Copy/paste | ✅ | ✅ | ✅ (selection) | ✅ |
| Right-click context menu | ❌ | ✅ | ❌ | Optional |
| Font size controls | ❌ | ✅ (Ctrl+/-) | ✅ (buttons) | **Add Ctrl+/-** |
| Theme picker | ❌ | ✅ | ✅ | ✅ |
| Fullscreen toggle | ❌ | ✅ | ✅ | ✅ |
| Download output | ❌ | ❌ | ✅ | ✅ |
| Auto-reconnect | ✅ | ✅ | ✅ | ✅ |
| Connection status | ✅ | ✅ | ✅ | ✅ |
| Clipboard addon (OSC 52) | ❌ | ✅ | ❌ | **Add** |
| Unicode/emoji support | ❌ | ✅ | ❌ | Optional |
| Split panes | ❌ | ✅ | ❌ | V2 |
| Tabs (multiple terminals) | ❌ | ✅ | ❌ | V2 |
| Keyboard shortcuts shown | ❌ | ✅ | ✅ (status bar) | ✅ |

---

## 6. UX Patterns from Docker Management Tools

### How Portainer handles terminals
- **Simple approach**: xterm.js with fit addon, WebSocket to container exec
- Reconnect on disconnect with exponential backoff
- Minimal chrome — just terminal + close button
- Uses `convertEol: true` for Windows containers
- Focus auto-moves to terminal on open

### How Proxmox VE handles terminals
- xterm.js with WebSocket to SPICE/VNC or shell
- Separate terminal windows for each container/VM
- Connection status indicator (colored dot)
- Clipboard integration via browser APIs

### How VS Code handles its terminal (gold standard)
- WebGL renderer by default
- Ctrl+Shift+F for search within terminal
- Multiple profiles (bash, zsh, PowerShell)
- Split terminal panes
- Drag-and-drop files into terminal
- Shell integration for command detection
- Link detection for file paths and URLs
- Unicode 11 support
- GPU-accelerated rendering with fallback

### Best UX Patterns for Dockerverse

1. **Auto-focus on open** — Terminal grabs focus immediately
2. **Ctrl+Shift+C / Ctrl+Shift+V** — Copy/paste without conflicting with shell Ctrl+C
3. **Ctrl+`+`/`-`** or **Ctrl+Scroll** — Font size zoom
4. **Double-click** — Select word
5. **Triple-click** — Select line  
6. **Drag to select** — Natural text selection
7. **Right-click** — Paste (or context menu)
8. **Escape** — Close search bar, deselect
9. **Status bar** — Show theme, shell, dimensions (cols×rows), connection status

---

## 7. Implementation Pattern for Svelte 5

### Complete Enhanced Terminal Component Setup

```typescript
// terminal-setup.ts — Extracted terminal initialization logic

import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { SearchAddon } from '@xterm/addon-search';
import { WebglAddon } from '@xterm/addon-webgl';
import { WebLinksAddon } from '@xterm/addon-web-links';
import { ClipboardAddon } from '@xterm/addon-clipboard';
import '@xterm/xterm/css/xterm.css';

export interface TerminalInstance {
  terminal: Terminal;
  fitAddon: FitAddon;
  searchAddon: SearchAddon;
  webglAddon: WebglAddon | null;
  dispose: () => void;
}

export function createTerminalInstance(
  element: HTMLElement,
  theme: Record<string, string>,
  fontSize: number = 14
): TerminalInstance {
  const terminal = new Terminal({
    cursorBlink: true,
    cursorStyle: 'bar',
    cursorWidth: 2,
    fontSize,
    fontFamily: "'JetBrains Mono', 'Fira Code', Menlo, Monaco, 'Courier New', monospace",
    lineHeight: 1.2,
    scrollback: 10000,
    allowProposedApi: true,
    allowTransparency: false,
    theme,
    minimumContrastRatio: 4.5,
    drawBoldTextInBrightColors: true,
    rightClickSelectsWord: true,
    altClickMovesCursor: true,
    logLevel: 'off',
  });

  // Essential addons
  const fitAddon = new FitAddon();
  const searchAddon = new SearchAddon();
  const webLinksAddon = new WebLinksAddon();
  const clipboardAddon = new ClipboardAddon();

  terminal.loadAddon(fitAddon);
  terminal.loadAddon(searchAddon);
  terminal.loadAddon(webLinksAddon);
  terminal.loadAddon(clipboardAddon);

  // Open terminal
  terminal.open(element);

  // WebGL renderer with fallback
  let webglAddon: WebglAddon | null = null;
  try {
    webglAddon = new WebglAddon();
    webglAddon.onContextLoss(() => {
      console.warn('WebGL context lost');
      webglAddon?.dispose();
      webglAddon = null;
    });
    terminal.loadAddon(webglAddon);
  } catch (e) {
    console.warn('WebGL not available, using DOM renderer');
    webglAddon = null;
  }

  // Fit after open
  fitAddon.fit();

  return {
    terminal,
    fitAddon,
    searchAddon,
    webglAddon,
    dispose: () => {
      webglAddon?.dispose();
      terminal.dispose();
    },
  };
}
```

### Keyboard Shortcuts Pattern

```typescript
// Custom key handler for the terminal
terminal.attachCustomKeyEventHandler((e: KeyboardEvent) => {
  // Ctrl+F — Search
  if (e.ctrlKey && e.key === 'f' && e.type === 'keydown') {
    e.preventDefault();
    toggleSearch();
    return false;
  }
  
  // Ctrl+L — Clear
  if (e.ctrlKey && e.key === 'l' && e.type === 'keydown') {
    e.preventDefault();
    terminal.clear();
    return false;
  }
  
  // Ctrl+= / Ctrl+- — Zoom
  if (e.ctrlKey && e.key === '=' && e.type === 'keydown') {
    e.preventDefault();
    changeFontSize(+1);
    return false;
  }
  if (e.ctrlKey && e.key === '-' && e.type === 'keydown') {
    e.preventDefault();
    changeFontSize(-1);
    return false;
  }
  
  // Ctrl+Shift+C — Copy
  if (e.ctrlKey && e.shiftKey && e.key === 'C' && e.type === 'keydown') {
    e.preventDefault();
    const sel = terminal.getSelection();
    if (sel) navigator.clipboard.writeText(sel);
    return false;
  }
  
  // Ctrl+Shift+V — Paste
  if (e.ctrlKey && e.shiftKey && e.key === 'V' && e.type === 'keydown') {
    e.preventDefault();
    navigator.clipboard.readText().then(text => terminal.paste(text));
    return false;
  }
  
  return true;
});

// Mouse wheel zoom (Ctrl + Scroll)
terminal.attachCustomWheelEventHandler((e: WheelEvent) => {
  if (e.ctrlKey) {
    e.preventDefault();
    changeFontSize(e.deltaY < 0 ? 1 : -1);
    return false;
  }
  return true;
});
```

### ResizeObserver Pattern (Debounced)

```typescript
import { onMount, onDestroy } from 'svelte';

let resizeObserver: ResizeObserver;
let fitTimeout: ReturnType<typeof setTimeout>;

onMount(() => {
  resizeObserver = new ResizeObserver(() => {
    // Debounce fit calls
    clearTimeout(fitTimeout);
    fitTimeout = setTimeout(() => {
      requestAnimationFrame(() => {
        fitAddon.fit();
      });
    }, 50);
  });
  resizeObserver.observe(terminalElement);
});

onDestroy(() => {
  clearTimeout(fitTimeout);
  resizeObserver?.disconnect();
  instance?.dispose();
});
```

---

## 8. Improvements for Current Dockerverse Terminal

Based on the analysis of [Terminal.svelte](dockerverse/frontend/src/lib/components/Terminal.svelte), here are the specific improvements needed:

### Missing (High Priority)
1. **WebGL renderer** — Not loaded; will dramatically improve scroll/render perf for container logs
2. **Web Links addon** — URLs in terminal output should be clickable
3. **Clipboard addon** — OSC 52 support for proper copy/paste from remote shells
4. **Ctrl+Scroll zoom** — Standard UX pattern for font size, currently only button-based
5. **Ctrl+Shift+C/V** — Keyboard copy/paste without conflicting with Ctrl+C (SIGINT)
6. **Debounced ResizeObserver** — Current impl calls `fitAddon.fit()` directly without debounce

### Missing (Medium Priority)
7. **Catppuccin Mocha theme** — Most popular terminal theme in 2025, not included
8. **One Dark Pro theme** — Very popular, not included
9. **`cursorStyle: 'bar'`** — Modern look vs the default block cursor
10. **`minimumContrastRatio: 4.5`** — WCAG accessibility compliance
11. **`lineHeight: 1.2`** — Slightly more spacing for readability
12. **Terminal dimensions in status bar** — Show `cols×rows` (useful for debugging)
13. **Web Fonts addon** — Ensure JetBrains Mono loads correctly

### Missing (Nice-to-Have / V2)
14. **Serialize addon** — For session recording/replay
15. **Unicode11 addon** — Proper emoji rendering
16. **Split pane terminals** — Multiple shells side by side
17. **Tab support** — Multiple terminal tabs per container
18. **Right-click context menu** — Copy, Paste, Clear, Select All

### Package.json additions needed

```json
{
  "dependencies": {
    "@xterm/xterm": "^6.0.0",
    "@xterm/addon-fit": "^0.11.0",
    "@xterm/addon-search": "^0.16.0",
    "@xterm/addon-webgl": "^0.19.0",
    "@xterm/addon-web-links": "^0.12.0",
    "@xterm/addon-clipboard": "^0.2.0",
    "@xterm/addon-web-fonts": "^0.2.0"
  }
}
```

---

## 9. WebSocket Architecture for Docker Exec

### Recommended message protocol

```typescript
// Client → Server
type ClientMessage =
  | { type: 'input'; data: string }
  | { type: 'resize'; cols: number; rows: number }
  | { type: 'ping' };

// Server → Client
type ServerMessage =
  | { type: 'output'; data: string }
  | { type: 'error'; data: string }
  | { type: 'exit'; code: number }
  | { type: 'pong' };
```

### Heartbeat / Keep-alive Pattern

```typescript
let pingInterval: ReturnType<typeof setInterval>;

ws.onopen = () => {
  // Send ping every 30s to keep connection alive
  pingInterval = setInterval(() => {
    if (ws?.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: 'ping' }));
    }
  }, 30000);
};

ws.onclose = () => {
  clearInterval(pingInterval);
};
```

---

## 10. Summary of Recommended Stack

| Component | Choice | Reason |
|-----------|--------|--------|
| **Terminal emulator** | `@xterm/xterm` v6.0.0 | Industry standard, used by VS Code |
| **Renderer** | `@xterm/addon-webgl` | GPU-accelerated, 10-50x faster |
| **Layout** | `@xterm/addon-fit` + debounced ResizeObserver | Responsive sizing |
| **Search** | `@xterm/addon-search` | In-buffer search with highlighting |
| **Links** | `@xterm/addon-web-links` | Clickable URLs |
| **Clipboard** | `@xterm/addon-clipboard` | OSC 52 protocol |
| **Font** | JetBrains Mono → Fira Code → Cascadia Code | Monospace with ligatures |
| **Theme** | Tokyo Night (default), + 5 alternatives | Dark UI optimized |
| **Scrollback** | 10,000 lines | Good balance of memory vs history |
| **Transport** | WebSocket with JSON messages | Real-time bidirectional |
| **Reconnect** | Exponential backoff (max 5 attempts) | Already implemented |

---

*Research compiled February 2026. Sources: xtermjs.org, github.com/xtermjs/xterm.js (v6.0.0), npmjs.com, Portainer source code, VS Code terminal implementation patterns.*
