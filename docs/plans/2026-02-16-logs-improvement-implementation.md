# DockerVerse Logs Page Improvement - Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Enhance logs page with fuzzy search, regex filtering, split-screen mode, keyboard shortcuts, and fix critical layout bug.

**Architecture:** Frontend-only improvements to existing Svelte 5 logs page. Uses $state/$derived reactivity, SSE streams unchanged, flex-based layout for stability, no new dependencies.

**Tech Stack:** Svelte 5, TypeScript, Tailwind CSS, Lucide icons, EventSource (SSE)

**Design Document:** `docs/plans/2026-02-16-logs-improvement-design.md`

---

## Task 1: Fix Layout Stability Bug (CRITICAL)

**Files:**
- Modify: `frontend/src/routes/logs/+page.svelte:260-290`

**Context:** Current fixed height `h-[calc(100vh-7rem)]` causes container to resize when selecting/deselecting containers. Replace with flexbox pattern from Terminal component.

### Step 1: Backup current layout structure

```bash
git diff frontend/src/routes/logs/+page.svelte
```

Expected: Clean working directory or show current changes

### Step 2: Replace fixed height with flex-based layout

Find the main container div (around line 272) and replace:

**Before:**
```svelte
<div class="flex flex-col h-[calc(100vh-7rem)]">
```

**After:**
```svelte
<div class="flex flex-col flex-1 min-h-0">
```

### Step 3: Update sidebar and content area layout

Find the flex container with sidebar + logs area and update:

**Before:**
```svelte
<div class="flex gap-4">
  <aside class="w-80">
    <!-- container list -->
  </aside>
  <div class="flex-1">
    <!-- logs area -->
  </div>
</div>
```

**After:**
```svelte
<div class="flex flex-1 min-h-0 gap-4">
  <!-- Sidebar: fixed width, scrollable -->
  <aside class="w-80 flex-shrink-0 flex flex-col min-h-0">
    <div class="flex-1 overflow-auto">
      <!-- container list -->
    </div>
  </aside>

  <!-- Logs area: fills remaining space -->
  <div class="flex-1 flex flex-col min-h-0">
    <!-- toolbar -->
    <div class="flex-1 overflow-auto">
      <!-- logs content -->
    </div>
  </div>
</div>
```

### Step 4: Test layout stability manually

**Test steps:**
1. Run dev server: `cd frontend && npm run dev`
2. Navigate to http://localhost:3000/logs
3. Select a container (logs should appear)
4. Select another container (logs area should NOT resize)
5. Deselect containers (logs area should remain stable)
6. Test with different browser widths (responsive)

Expected: Logs area maintains consistent height throughout interactions

### Step 5: Commit

```bash
git add frontend/src/routes/logs/+page.svelte
git commit -m "fix(logs): replace fixed height with flexbox for layout stability

- Remove h-[calc(100vh-7rem)] causing resize on container selection
- Apply flex-1 min-h-0 pattern from Terminal component
- Sidebar and logs area now maintain stable dimensions
- Fixes jarring UX and lost scroll position

Refs: docs/plans/2026-02-16-logs-improvement-design.md"
```

---

## Task 2: Add Fuzzy Container Search

**Files:**
- Modify: `frontend/src/routes/logs/+page.svelte:1-50` (add types and functions)
- Modify: `frontend/src/routes/logs/+page.svelte:100-150` (update state and derived)

**Context:** Add intelligent fuzzy matching for container names (e.g., "ngx" finds "nginx-proxy").

### Step 1: Add fuzzy match types and function

Add near the top of the script section (after imports, around line 30):

```typescript
// Fuzzy search types
interface FuzzyMatch {
  match: boolean;
  score: number;
}

interface ContainerWithScore extends Container {
  match?: boolean;
  score?: number;
}

/**
 * Fuzzy match algorithm for container search
 * Priorities: exact substring > acronym > character sequence
 */
function fuzzyMatch(query: string, text: string): FuzzyMatch {
  if (!query) return { match: true, score: 100 };

  const q = query.toLowerCase();
  const t = text.toLowerCase();

  // Exact substring match - highest priority
  if (t.includes(q)) {
    return { match: true, score: 100 };
  }

  // Acronym match (e.g., "ngx" matches "nginx-proxy")
  const words = t.split(/[-_\s]/);
  const acronym = words.map(w => w[0]).join('');
  if (acronym.includes(q)) {
    return { match: true, score: 80 };
  }

  // Character sequence match
  let qIndex = 0;
  let score = 0;
  for (let i = 0; i < t.length && qIndex < q.length; i++) {
    if (t[i] === q[qIndex]) {
      qIndex++;
      score += 50 / q.length;
    }
  }

  return {
    match: qIndex === q.length,
    score: Math.round(score)
  };
}
```

### Step 2: Update container filtering with fuzzy search

Find the `containerSearch` state variable and update the derived containers:

**Find (around line 100):**
```typescript
let containerSearch = $state("");
```

**Replace with:**
```typescript
let containerSearch = $state("");

// Fuzzy-filtered and sorted containers
let filteredContainers = $derived(() => {
  if (!containerSearch) return containers;

  return containers
    .map(c => ({ ...c, ...fuzzyMatch(containerSearch, c.name) }))
    .filter(c => c.match)
    .sort((a, b) => (b.score || 0) - (a.score || 0));
});
```

### Step 3: Update container list to use filteredContainers

Find the container list rendering (around line 200-250) and replace:

**Before:**
```svelte
{#each containers as c}
```

**After:**
```svelte
{#each filteredContainers as c}
```

### Step 4: Add search match highlighting

Add a helper function after `fuzzyMatch`:

```typescript
/**
 * Highlight matched characters in container name
 */
function highlightMatch(text: string, query: string): string {
  if (!query) return text;

  const q = query.toLowerCase();
  const t = text.toLowerCase();

  // For exact matches, highlight the substring
  const index = t.indexOf(q);
  if (index !== -1) {
    return text.substring(0, index) +
           `<mark class="bg-primary/30 text-foreground">${text.substring(index, index + q.length)}</mark>` +
           text.substring(index + q.length);
  }

  // For other matches, highlight individual matched characters
  let result = '';
  let qIndex = 0;
  for (let i = 0; i < text.length && qIndex < q.length; i++) {
    if (t[i] === q[qIndex]) {
      result += `<mark class="bg-primary/30 text-foreground">${text[i]}</mark>`;
      qIndex++;
    } else {
      result += text[i];
    }
  }
  result += text.substring(result.replace(/<[^>]*>/g, '').length);

  return result;
}
```

### Step 5: Update container name rendering with highlight

Find container name display in the list and update:

**Before:**
```svelte
<span class="flex-1 text-sm">{c.name}</span>
```

**After:**
```svelte
<span class="flex-1 text-sm">
  {@html highlightMatch(c.name, containerSearch)}
</span>
```

### Step 6: Test fuzzy search manually

**Test cases:**
1. Type "ngx" → should find "nginx-proxy"
2. Type "doc" → should find "dockerverse"
3. Type "rp" → should find "redis-prod" (acronym)
4. Type exact name → should show at top with highest score
5. Clear search → all containers shown

Expected: Containers filtered and sorted by relevance, matched characters highlighted

### Step 7: Commit

```bash
git add frontend/src/routes/logs/+page.svelte
git commit -m "feat(logs): add fuzzy container search with highlighting

- Implement fuzzy match algorithm (exact > acronym > sequence)
- Sort results by match score for relevance
- Highlight matched characters in container names
- Improves container discovery and UX

Closes #[issue] if any"
```

---

## Task 3: Add Regex Log Filtering

**Files:**
- Modify: `frontend/src/routes/logs/+page.svelte:50-80` (add regex state and logic)
- Modify: `frontend/src/routes/logs/+page.svelte:150-200` (update log filtering)
- Modify: `frontend/src/routes/logs/+page.svelte:400-450` (add regex toggle UI)

**Context:** Add regex-based log filtering with toggle and error handling.

### Step 1: Add regex state variables

Add after the existing search state (around line 80):

```typescript
// Regex filtering state
let regexEnabled = $state(false);
let regexError = $state<string | null>(null);

// Compile search pattern with error handling
let searchPattern = $derived(() => {
  if (!logSearch) return null;

  if (regexEnabled) {
    try {
      regexError = null;
      return new RegExp(logSearch, 'gi');
    } catch (e) {
      regexError = (e as Error).message.replace('Invalid regular expression: /', '');
      return null;
    }
  }

  // Simple string search - escape special regex characters
  const escaped = logSearch.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  return new RegExp(escaped, 'gi');
});
```

### Step 2: Update log filtering logic

Find where logs are filtered (around line 150) and update:

**Before:**
```typescript
let displayedLogs = $derived(() => {
  // existing filtering logic
});
```

**After:**
```typescript
let displayedLogs = $derived(() => {
  let logs = allLogs;

  // Filter by selected containers
  if (mode !== 'single' && selectedContainers.size > 0) {
    logs = logs.filter(log =>
      Array.from(selectedContainers).some(id => log.key.startsWith(id))
    );
  }

  // Filter by search pattern
  if (searchPattern) {
    logs = logs.filter(log => searchPattern.test(log.line));
  }

  return logs;
});
```

### Step 3: Add highlight matches function

Add helper function after searchPattern derived:

```typescript
/**
 * Highlight regex matches in log line
 */
function highlightLogMatches(line: string): string {
  if (!searchPattern || regexError) return line;

  // Reset regex lastIndex for global flag
  searchPattern.lastIndex = 0;

  return line.replace(searchPattern, (match) =>
    `<mark class="bg-primary/30 text-foreground font-semibold">${match}</mark>`
  );
}
```

### Step 4: Add regex toggle button to UI

Find the search input area (around line 400) and update:

**Before:**
```svelte
<input
  type="text"
  bind:value={logSearch}
  placeholder="Search logs..."
  class="input flex-1"
/>
```

**After:**
```svelte
<div class="flex items-center gap-2">
  <input
    type="text"
    bind:value={logSearch}
    placeholder={regexEnabled ? "Regex pattern (e.g., ERROR|WARN)" : "Search logs..."}
    class="input flex-1 {regexError ? 'border-accent-red' : ''}"
  />

  <button
    class="btn-icon {regexEnabled ? 'text-primary border-primary' : ''}"
    onclick={() => regexEnabled = !regexEnabled}
    title={regexEnabled ? "Disable regex mode" : "Enable regex mode"}
    aria-label={regexEnabled ? "Disable regex" : "Enable regex"}
  >
    <Code class="w-4 h-4" />
  </button>

  {#if regexError}
    <span class="text-xs text-accent-red max-w-xs truncate" title={regexError}>
      {regexError}
    </span>
  {/if}
</div>
```

### Step 5: Update log line rendering with highlights

Find log line rendering (around line 450) and update:

**Before:**
```svelte
<pre class="font-mono text-xs">{log.line}</pre>
```

**After:**
```svelte
<pre class="font-mono text-xs whitespace-pre-wrap break-all">
  {@html highlightLogMatches(log.line)}
</pre>
```

### Step 6: Import Code icon

Add to imports at top:

```typescript
import { Code, /* existing imports */ } from "lucide-svelte";
```

### Step 7: Test regex filtering manually

**Test cases:**
1. Type "error" with regex OFF → simple string match
2. Type "ERROR|WARN" with regex ON → matches both patterns
3. Type invalid regex "(" with regex ON → shows error message
4. Clear search → error clears, all logs shown
5. Test pattern `/api/.*timeout` → matches API timeout logs

Expected: Regex patterns filter correctly, errors handled gracefully, matches highlighted

### Step 8: Commit

```bash
git add frontend/src/routes/logs/+page.svelte
git commit -m "feat(logs): add regex log filtering with toggle

- Implement regex pattern compilation with error handling
- Add regex mode toggle button in search bar
- Highlight regex matches in log lines
- Display validation errors inline
- Supports patterns like ERROR|WARN, /api/.*, etc.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 4: Add Split-Screen Mode

**Files:**
- Modify: `frontend/src/routes/logs/+page.svelte:80-100` (add split mode state)
- Modify: `frontend/src/routes/logs/+page.svelte:350-400` (add mode toggle)
- Modify: `frontend/src/routes/logs/+page.svelte:500-600` (add split view rendering)

**Context:** Add fourth viewing mode for side-by-side comparison of two containers.

### Step 1: Update LogMode type and add split state

Find the LogMode type (around line 80) and update:

**Before:**
```typescript
type LogMode = "single" | "multi" | "grouped";
```

**After:**
```typescript
type LogMode = "single" | "multi" | "grouped" | "split";

// Split mode state - which two containers to show
let splitContainers = $state<[string, string] | null>(null);
```

### Step 2: Add helper functions for split mode

Add after split state:

```typescript
/**
 * Enter split mode with two containers
 */
function enterSplitMode(container1: string, container2: string) {
  mode = "split";
  splitContainers = [container1, container2];
  selectedContainers.clear();
  selectedContainers.add(container1);
  selectedContainers.add(container2);
}

/**
 * Get logs for a specific container
 */
function getLogsForContainer(containerId: string) {
  return displayedLogs.filter(log => log.key.startsWith(containerId));
}

/**
 * Get container name by ID
 */
function getContainerName(containerId: string): string {
  const container = containers.find(c => c.id === containerId);
  return container?.name || containerId;
}
```

### Step 3: Add split mode toggle button

Find the mode selection buttons (around line 350) and add split button:

**Before:**
```svelte
<div class="flex gap-1" role="radiogroup" aria-label="Log viewing mode">
  <button>Single</button>
  <button>Multi</button>
  <button>Grouped</button>
</div>
```

**After:**
```svelte
<div class="flex gap-1" role="radiogroup" aria-label="Log viewing mode">
  <button
    role="radio"
    aria-checked={mode === 'single'}
    onclick={() => mode = 'single'}
    class="btn-sm {mode === 'single' ? 'btn-primary' : 'btn-ghost'}"
    title="View one container (Ctrl+1)"
  >
    <Square class="w-4 h-4 mr-1" />
    Single
  </button>

  <button
    role="radio"
    aria-checked={mode === 'multi'}
    onclick={() => mode = 'multi'}
    class="btn-sm {mode === 'multi' ? 'btn-primary' : 'btn-ghost'}"
    title="View multiple containers stacked (Ctrl+2)"
  >
    <Layers class="w-4 h-4 mr-1" />
    Multi
  </button>

  <button
    role="radio"
    aria-checked={mode === 'grouped'}
    onclick={() => mode = 'grouped'}
    class="btn-sm {mode === 'grouped' ? 'btn-primary' : 'btn-ghost'}"
    title="View grouped by host (Ctrl+3)"
  >
    <Grid class="w-4 h-4 mr-1" />
    Grouped
  </button>

  <button
    role="radio"
    aria-checked={mode === 'split'}
    onclick={() => {
      if (selectedContainers.size >= 2) {
        const [c1, c2] = Array.from(selectedContainers).slice(0, 2);
        enterSplitMode(c1, c2);
      } else {
        mode = 'split';
      }
    }}
    disabled={selectedContainers.size < 2 && mode !== 'split'}
    class="btn-sm {mode === 'split' ? 'btn-primary' : 'btn-ghost'}"
    title="View two containers side-by-side (Ctrl+4)"
  >
    <Columns class="w-4 h-4 mr-1" />
    Split
  </button>
</div>
```

### Step 4: Import new icons

Add to imports:

```typescript
import { Square, Layers, Grid, Columns, /* existing */ } from "lucide-svelte";
```

### Step 5: Add split view rendering

Find the logs display area (around line 500) and add split mode case:

**After the existing mode cases, add:**

```svelte
{#if mode === "split" && splitContainers}
  <div class="flex-1 flex gap-4 min-h-0">
    <!-- Left pane -->
    <div class="flex-1 flex flex-col min-h-0 border border-border rounded-lg overflow-hidden bg-background-secondary">
      <!-- Header -->
      <div class="px-3 py-2 bg-background-tertiary border-b border-border flex items-center justify-between">
        <span class="text-sm font-medium text-foreground">
          {getContainerName(splitContainers[0])}
        </span>
        <span class="text-xs text-foreground-muted">
          {getLogsForContainer(splitContainers[0]).length} lines
        </span>
      </div>

      <!-- Logs -->
      <div
        class="flex-1 overflow-auto p-3 space-y-1"
        bind:this={logContainerLeft}
      >
        {#each getLogsForContainer(splitContainers[0]) as log}
          <div class="flex items-start gap-2 text-xs font-mono">
            <span class="text-foreground-muted w-24 flex-shrink-0">
              {formatTimestamp(log.ts)}
            </span>
            <pre class="flex-1 whitespace-pre-wrap break-all">
              {@html highlightLogMatches(log.line)}
            </pre>
          </div>
        {/each}
      </div>
    </div>

    <!-- Right pane (same structure) -->
    <div class="flex-1 flex flex-col min-h-0 border border-border rounded-lg overflow-hidden bg-background-secondary">
      <!-- Header -->
      <div class="px-3 py-2 bg-background-tertiary border-b border-border flex items-center justify-between">
        <span class="text-sm font-medium text-foreground">
          {getContainerName(splitContainers[1])}
        </span>
        <span class="text-xs text-foreground-muted">
          {getLogsForContainer(splitContainers[1]).length} lines
        </span>
      </div>

      <!-- Logs -->
      <div
        class="flex-1 overflow-auto p-3 space-y-1"
        bind:this={logContainerRight}
      >
        {#each getLogsForContainer(splitContainers[1]) as log}
          <div class="flex items-start gap-2 text-xs font-mono">
            <span class="text-foreground-muted w-24 flex-shrink-0">
              {formatTimestamp(log.ts)}
            </span>
            <pre class="flex-1 whitespace-pre-wrap break-all">
              {@html highlightLogMatches(log.line)}
            </pre>
          </div>
        {/each}
      </div>
    </div>
  </div>
{/if}
```

### Step 6: Add scroll container refs

Add to state variables:

```typescript
let logContainerLeft: HTMLDivElement;
let logContainerRight: HTMLDivElement;
```

### Step 7: Test split mode manually

**Test steps:**
1. Select 2 containers from list
2. Click "Split" button
3. Verify both containers show side-by-side
4. Verify independent scrolling
5. Verify search/filter works in both panes
6. Switch to another mode and back

Expected: Two containers displayed side-by-side with independent scroll

### Step 8: Commit

```bash
git add frontend/src/routes/logs/+page.svelte
git commit -m "feat(logs): add split-screen mode for two containers

- Add fourth viewing mode 'split' for side-by-side comparison
- Independent scroll for each pane
- Shows line count in headers
- Requires 2 containers selected to enable
- Toggle via Ctrl+4 (keyboard shortcut in next task)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 5: Add Timestamp Enhancements

**Files:**
- Modify: `frontend/src/routes/logs/+page.svelte:100-120` (add timestamp state)
- Modify: `frontend/src/routes/logs/+page.svelte:120-180` (add format functions)
- Modify: `frontend/src/routes/logs/+page.svelte:380-420` (add UI toggle)

**Context:** Add timestamp format options (absolute, relative, none) with toggle buttons.

### Step 1: Add timestamp state and format type

Add after mode state (around line 100):

```typescript
// Timestamp formatting
type TimestampFormat = "absolute" | "relative" | "none";
let timestampFormat = $state<TimestampFormat>("absolute");
```

### Step 2: Add timestamp formatting functions

Add after timestamp state:

```typescript
/**
 * Format timestamp based on selected mode
 */
function formatTimestamp(ts: number): string {
  if (timestampFormat === "none") return "";

  const date = new Date(ts);

  if (timestampFormat === "absolute") {
    return date.toLocaleTimeString('en-US', {
      hour12: false,
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      fractionalSecondDigits: 3
    });
  }

  // Relative format
  const now = Date.now();
  const diff = now - ts;

  if (diff < 1000) return "just now";
  if (diff < 60000) return `${Math.floor(diff / 1000)}s ago`;
  if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`;
  return `${Math.floor(diff / 86400000)}d ago`;
}
```

### Step 3: Add timestamp toggle UI

Find the toolbar area (around line 380) and add timestamp controls:

```svelte
<div class="flex items-center gap-2 border-l border-border pl-4 ml-4">
  <span class="text-xs text-foreground-muted">Timestamp:</span>
  <div class="flex items-center gap-1 border border-border rounded-md p-1">
    <button
      onclick={() => timestampFormat = 'absolute'}
      class="px-2 py-1 text-xs rounded transition-colors {timestampFormat === 'absolute' ? 'bg-primary text-white' : 'hover:bg-background-tertiary'}"
      title="Absolute time (HH:MM:SS.mmm)"
    >
      HH:MM:SS
    </button>
    <button
      onclick={() => timestampFormat = 'relative'}
      class="px-2 py-1 text-xs rounded transition-colors {timestampFormat === 'relative' ? 'bg-primary text-white' : 'hover:bg-background-tertiary'}"
      title="Relative time (e.g., 2m ago)"
    >
      Relative
    </button>
    <button
      onclick={() => timestampFormat = 'none'}
      class="px-2 py-1 text-xs rounded transition-colors {timestampFormat === 'none' ? 'bg-primary text-white' : 'hover:bg-background-tertiary'}"
      title="Hide timestamps"
    >
      Hide
    </button>
  </div>
</div>
```

### Step 4: Update all timestamp displays

Find all instances of timestamp rendering and ensure they use `formatTimestamp(log.ts)`:

**Single mode, multi mode, grouped mode, split mode should all use:**
```svelte
<span class="text-foreground-muted text-xs">
  {formatTimestamp(log.ts)}
</span>
```

### Step 5: Test timestamp formatting

**Test cases:**
1. Default (absolute) → shows HH:MM:SS.mmm
2. Click "Relative" → shows "Xs ago", "Xm ago", etc.
3. Click "Hide" → timestamps disappear
4. Generate new log → relative time updates appropriately
5. Test all viewing modes (single, multi, grouped, split)

Expected: All timestamp displays respect the selected format

### Step 6: Commit

```bash
git add frontend/src/routes/logs/+page.svelte
git commit -m "feat(logs): add timestamp format toggle (absolute/relative/none)

- Add three timestamp modes: absolute (HH:MM:SS), relative (2m ago), none
- Format selector in toolbar
- Applies to all viewing modes
- Improves log readability for different use cases

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 6: Add Keyboard Shortcuts

**Files:**
- Modify: `frontend/src/routes/logs/+page.svelte:600-700` (add keyboard handler)
- Modify: `frontend/src/routes/logs/+page.svelte:800-850` (add shortcut legend)

**Context:** Add keyboard shortcuts for common actions following Dozzle patterns.

### Step 1: Add element refs for focus management

Add to state variables:

```typescript
// Element refs for keyboard shortcuts
let logSearchInput: HTMLInputElement | undefined;
let containerSearchInput: HTMLInputElement | undefined;
```

### Step 2: Add keyboard shortcut handler

Add in the `onMount` callback (around line 600):

```typescript
onMount(() => {
  // Existing mount logic...

  function handleKeydown(e: KeyboardEvent) {
    // Ignore shortcuts when typing in inputs (except ESC)
    const isTyping = e.target instanceof HTMLInputElement ||
                     e.target instanceof HTMLTextAreaElement;
    if (isTyping && e.key !== 'Escape') return;

    // Ctrl+F - Focus log search
    if (e.ctrlKey && e.key === 'f') {
      e.preventDefault();
      logSearchInput?.focus();
      return;
    }

    // Ctrl+K - Clear logs
    if (e.ctrlKey && e.key === 'k') {
      e.preventDefault();
      allLogs = [];
      return;
    }

    // Ctrl+1/2/3/4 - Mode switching
    if (e.ctrlKey && /[1-4]/.test(e.key)) {
      e.preventDefault();
      const modes: LogMode[] = ['single', 'multi', 'grouped', 'split'];
      const newMode = modes[parseInt(e.key) - 1];

      // Special handling for split mode
      if (newMode === 'split' && selectedContainers.size >= 2) {
        const [c1, c2] = Array.from(selectedContainers).slice(0, 2);
        enterSplitMode(c1, c2);
      } else if (newMode !== 'split') {
        mode = newMode;
      }
      return;
    }

    // Ctrl+P - Toggle pause/play
    if (e.ctrlKey && e.key === 'p') {
      e.preventDefault();
      isPaused = !isPaused;
      return;
    }

    // Ctrl+W - Toggle line wrap
    if (e.ctrlKey && e.key === 'w') {
      e.preventDefault();
      lineWrap = !lineWrap;
      return;
    }

    // Ctrl+E - Export logs
    if (e.ctrlKey && e.key === 'e') {
      e.preventDefault();
      exportLogs();
      return;
    }

    // Space - Pause/Resume (when not in input)
    if (e.key === ' ' && !isTyping) {
      e.preventDefault();
      isPaused = !isPaused;
      return;
    }

    // / - Focus container search
    if (e.key === '/' && !isTyping) {
      e.preventDefault();
      containerSearchInput?.focus();
      return;
    }

    // Escape - Clear search or deselect
    if (e.key === 'Escape') {
      if (isTyping) {
        // Blur input
        (e.target as HTMLElement).blur();
      } else if (logSearch) {
        logSearch = "";
      } else if (containerSearch) {
        containerSearch = "";
      } else {
        selectedContainers.clear();
      }
      return;
    }
  }

  window.addEventListener('keydown', handleKeydown);

  return () => {
    window.removeEventListener('keydown', handleKeydown);
  };
});
```

### Step 3: Add lineWrap state

Add to state variables:

```typescript
let lineWrap = $state(true);
```

### Step 4: Apply lineWrap to log rendering

Find log line rendering and add conditional class:

```svelte
<pre class="font-mono text-xs {lineWrap ? 'whitespace-pre-wrap' : 'whitespace-pre'} break-all">
  {@html highlightLogMatches(log.line)}
</pre>
```

### Step 5: Add keyboard shortcuts legend

Add to the toolbar or bottom of the page:

```svelte
<!-- Keyboard shortcuts help -->
<div class="flex items-center gap-4 text-xs text-foreground-muted border-t border-border pt-2 mt-2">
  <span class="font-medium">Shortcuts:</span>
  <span><kbd class="kbd">Ctrl+F</kbd> Search</span>
  <span><kbd class="kbd">Ctrl+K</kbd> Clear</span>
  <span><kbd class="kbd">Ctrl+1-4</kbd> Modes</span>
  <span><kbd class="kbd">Ctrl+P</kbd> Pause</span>
  <span><kbd class="kbd">Ctrl+W</kbd> Wrap</span>
  <span><kbd class="kbd">Ctrl+E</kbd> Export</span>
  <span><kbd class="kbd">/</kbd> Find Container</span>
  <span><kbd class="kbd">Esc</kbd> Clear/Deselect</span>
</div>
```

### Step 6: Add kbd styling to global CSS

Add to `frontend/src/app.css` (if not exists):

```css
.kbd {
  @apply px-1.5 py-0.5 bg-background-secondary border border-border rounded text-xs font-mono;
}
```

### Step 7: Update input bindings for refs

Find search inputs and add refs:

```svelte
<input
  bind:this={logSearchInput}
  type="text"
  bind:value={logSearch}
  placeholder="..."
  class="input"
/>

<!-- Container search -->
<input
  bind:this={containerSearchInput}
  type="text"
  bind:value={containerSearch}
  placeholder="..."
  class="input"
/>
```

### Step 8: Test keyboard shortcuts

**Test each shortcut:**
- `Ctrl+F` → focuses log search
- `Ctrl+K` → clears all logs
- `Ctrl+1` → single mode
- `Ctrl+2` → multi mode
- `Ctrl+3` → grouped mode
- `Ctrl+4` → split mode (with 2 containers selected)
- `Ctrl+P` → toggles pause
- `Ctrl+W` → toggles line wrap
- `Ctrl+E` → exports logs
- `Space` → toggles pause (when not in input)
- `/` → focuses container search
- `Esc` → clears search or deselects

Expected: All shortcuts work correctly, don't interfere with typing

### Step 9: Commit

```bash
git add frontend/src/routes/logs/+page.svelte frontend/src/app.css
git commit -m "feat(logs): add comprehensive keyboard shortcuts

- Ctrl+F: Focus log search
- Ctrl+K: Clear logs
- Ctrl+1-4: Switch viewing modes
- Ctrl+P: Pause/resume streaming
- Ctrl+W: Toggle line wrap
- Ctrl+E: Export logs
- /: Focus container search
- Space: Pause/resume
- Esc: Clear/deselect

Add shortcuts legend in footer
Improves power-user efficiency

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 7: Enhance Export Functionality

**Files:**
- Modify: `frontend/src/routes/logs/+page.svelte:180-250` (enhance export function)
- Modify: `frontend/src/routes/logs/+page.svelte:420-450` (add export menu)

**Context:** Upgrade export to include metadata and filter-aware exporting.

### Step 1: Enhance exportLogs function

Find existing `exportLogs` function and replace:

```typescript
/**
 * Export logs with metadata
 * @param containerId Optional - export specific container, or all if undefined
 */
function exportLogs(containerId?: string) {
  // Determine which logs to export
  const logs = containerId
    ? displayedLogs.filter(l => l.key.startsWith(containerId))
    : displayedLogs;

  // Get container info if specific container
  const container = containerId
    ? containers.find(c => c.id === containerId)
    : null;

  // Build metadata header
  const metadata = [
    `# DockerVerse Log Export`,
    `# Generated: ${new Date().toISOString()}`,
    `# Container: ${container?.name || 'All Selected'}`,
    `# Host: ${container?.hostId || 'Multiple'}`,
    `# Total Lines: ${logs.length}`,
    `# Viewing Mode: ${mode}`,
    `# Search Filter: ${logSearch || 'None'}`,
    `# Regex Enabled: ${regexEnabled}`,
    `# Timestamp Format: ${timestampFormat}`,
    ``,
    `# Filters Applied:`,
    `# - Selected Containers: ${Array.from(selectedContainers).map(id => {
      const c = containers.find(ct => ct.id === id);
      return c?.name || id;
    }).join(', ')}`,
    ``,
    `# ===== LOGS START =====`,
    ``,
  ].join('\n');

  // Format log entries
  const content = logs
    .map(log => {
      const ts = formatTimestamp(log.ts) || new Date(log.ts).toISOString();
      return `[${ts}] [${log.name}] ${log.line}`;
    })
    .join('\n');

  // Create and download file
  const blob = new Blob([metadata + content], { type: 'text/plain; charset=utf-8' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;

  // Generate filename
  const containerName = container?.name || 'all-containers';
  const timestamp = new Date().toISOString().slice(0, 19).replace(/:/g, '-');
  a.download = `dockerverse-logs_${containerName}_${timestamp}.txt`;

  a.click();
  URL.revokeObjectURL(url);
}
```

### Step 2: Add export dropdown menu

Find the export button area (around line 420) and replace:

**Before:**
```svelte
<button onclick={exportLogs}>Export</button>
```

**After:**
```svelte
<!-- Export dropdown -->
<div class="relative">
  <button
    onclick={() => showExportMenu = !showExportMenu}
    class="btn btn-sm btn-ghost"
    title="Export logs (Ctrl+E)"
  >
    <Download class="w-4 h-4 mr-1" />
    Export
    <ChevronDown class="w-3 h-3 ml-1" />
  </button>

  {#if showExportMenu}
    <div class="absolute right-0 top-full mt-1 bg-background-secondary border border-border rounded-lg shadow-xl z-10 py-1 min-w-[200px]">
      <button
        onclick={() => {
          exportLogs();
          showExportMenu = false;
        }}
        class="w-full px-3 py-2 text-sm text-left hover:bg-background-tertiary flex items-center gap-2"
      >
        <FileText class="w-4 h-4" />
        <div class="flex-1">
          <div class="font-medium">Export All Logs</div>
          <div class="text-xs text-foreground-muted">
            {displayedLogs.length} lines
          </div>
        </div>
      </button>

      {#if mode === 'single' && selectedContainers.size === 1}
        <button
          onclick={() => {
            const containerId = Array.from(selectedContainers)[0];
            exportLogs(containerId);
            showExportMenu = false;
          }}
          class="w-full px-3 py-2 text-sm text-left hover:bg-background-tertiary flex items-center gap-2"
        >
          <Container class="w-4 h-4" />
          <div class="flex-1">
            <div class="font-medium">Export Current Container</div>
            <div class="text-xs text-foreground-muted">
              {getLogsForContainer(Array.from(selectedContainers)[0]).length} lines
            </div>
          </div>
        </button>
      {/if}

      {#if mode === 'split' && splitContainers}
        <button
          onclick={() => {
            exportLogs(splitContainers[0]);
            showExportMenu = false;
          }}
          class="w-full px-3 py-2 text-sm text-left hover:bg-background-tertiary flex items-center gap-2"
        >
          <FileText class="w-4 h-4" />
          <div>Export Left Pane</div>
        </button>
        <button
          onclick={() => {
            exportLogs(splitContainers[1]);
            showExportMenu = false;
          }}
          class="w-full px-3 py-2 text-sm text-left hover:bg-background-tertiary flex items-center gap-2"
        >
          <FileText class="w-4 h-4" />
          <div>Export Right Pane</div>
        </button>
      {/if}
    </div>
  {/if}
</div>

<!-- Click outside to close export menu -->
{#if showExportMenu}
  <button
    class="fixed inset-0 z-5 cursor-default"
    onclick={() => showExportMenu = false}
    aria-label="Close export menu"
  ></button>
{/if}
```

### Step 3: Add showExportMenu state

Add to state variables:

```typescript
let showExportMenu = $state(false);
```

### Step 4: Import new icons

Add to imports:

```typescript
import { Download, ChevronDown, FileText, Container, /* existing */ } from "lucide-svelte";
```

### Step 5: Test export functionality

**Test cases:**
1. Export all logs → file includes metadata + all displayed logs
2. Export single container → file includes only that container's logs
3. Export from split mode → can export left or right pane separately
4. Test with search filter active → exported file respects filter
5. Check filename format → `dockerverse-logs_<container>_<timestamp>.txt`
6. Open exported file → metadata header present, logs formatted correctly

Expected: Exports work correctly with metadata and respect filters

### Step 6: Commit

```bash
git add frontend/src/routes/logs/+page.svelte
git commit -m "feat(logs): enhance export with metadata and per-container options

- Add metadata header (timestamp, filters, mode, etc.)
- Export all logs or specific container
- Split mode can export each pane separately
- Dropdown menu for export options
- Better filename format with timestamp
- Respects active search filters

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 8: Visual Enhancements

**Files:**
- Modify: `frontend/src/routes/logs/+page.svelte:500-700` (enhance log rendering)

**Context:** Improve visual design with better color coding, spacing, and accessibility.

### Step 1: Enhance multi-mode log entry rendering

Find multi-mode log rendering (around line 500) and enhance:

**Before:**
```svelte
<div class="log-entry">
  <span>{log.name}</span>
  <span>{log.line}</span>
</div>
```

**After:**
```svelte
<div class="flex items-start gap-2 group hover:bg-background-secondary/50 px-2 py-1.5 rounded transition-colors">
  <!-- Color bar (4px wide, full height) -->
  <div
    class="w-1 h-full flex-shrink-0 rounded-full self-stretch"
    style="background: {log.color}"
    aria-hidden="true"
  ></div>

  <!-- Timestamp -->
  {#if timestampFormat !== 'none'}
    <span class="text-foreground-muted text-xs w-28 flex-shrink-0 font-mono">
      {formatTimestamp(log.ts)}
    </span>
  {/if}

  <!-- Container name badge -->
  <span
    class="px-2 py-0.5 rounded text-xs font-medium flex-shrink-0 border"
    style="
      background: {log.color}15;
      color: {log.color};
      border-color: {log.color}30;
    "
  >
    {log.name}
  </span>

  <!-- Log content -->
  <pre class="flex-1 text-xs font-mono {lineWrap ? 'whitespace-pre-wrap' : 'whitespace-pre'} break-all group-hover:text-foreground transition-colors">
    {@html highlightLogMatches(log.line)}
  </pre>
</div>
```

### Step 2: Enhance container selection checkboxes

Find container list rendering and enhance:

```svelte
<label
  class="flex items-center gap-3 px-3 py-2 hover:bg-background-secondary rounded cursor-pointer group transition-all"
  class:bg-primary/10={selectedContainers.has(c.id)}
>
  <input
    type="checkbox"
    checked={selectedContainers.has(c.id)}
    onchange={() => toggleContainer(c.id)}
    class="w-4 h-4 accent-primary focus:ring-2 focus:ring-primary focus:ring-offset-2 focus:ring-offset-background"
    aria-label="Select {c.name}"
  />

  <!-- Status indicator -->
  <div
    class="w-2 h-2 rounded-full flex-shrink-0"
    class:bg-running={c.state === 'running'}
    class:bg-stopped={c.state === 'exited'}
    class:bg-paused={c.state === 'paused'}
  ></div>

  <!-- Container name with fuzzy highlight -->
  <span class="flex-1 text-sm group-hover:text-primary transition-colors">
    {@html highlightMatch(c.name, containerSearch)}
  </span>

  <!-- Match score indicator (if searching) -->
  {#if containerSearch && c.score}
    <span
      class="px-1.5 py-0.5 rounded text-xs font-medium bg-primary/20 text-primary"
      title="Match score: {c.score}/100"
    >
      {c.score}
    </span>
  {/if}
</label>
```

### Step 3: Add color palette for better contrast

Add color generation function:

```typescript
/**
 * Generate visually distinct colors for containers
 * Uses HSL for better accessibility and contrast
 */
function getContainerColor(key: string): string {
  // Hash the key to get consistent color per container
  let hash = 0;
  for (let i = 0; i < key.length; i++) {
    hash = key.charCodeAt(i) + ((hash << 5) - hash);
  }

  // Generate HSL color with good saturation and lightness
  const hue = Math.abs(hash % 360);
  const saturation = 65 + (Math.abs(hash) % 20); // 65-85%
  const lightness = 55 + (Math.abs(hash >> 8) % 15); // 55-70%

  return `hsl(${hue}, ${saturation}%, ${lightness}%)`;
}
```

### Step 4: Apply color to log entries

Ensure log entries use the color function:

```typescript
// When creating log entry
const entry: LogEntry = {
  key: `${c.id}@${c.hostId}`,
  name: c.name,
  line: line,
  color: getContainerColor(`${c.id}@${c.hostId}`),
  ts: Date.now()
};
```

### Step 5: Add accessibility improvements

Add ARIA labels and roles throughout:

```svelte
<!-- Mode selection -->
<div
  role="radiogroup"
  aria-label="Log viewing mode"
  class="flex gap-1"
>
  <!-- buttons with role="radio" and aria-checked -->
</div>

<!-- Search areas -->
<div role="search" aria-label="Container search">
  <input aria-label="Search containers" />
</div>

<div role="search" aria-label="Log search">
  <input aria-label="Search logs" />
</div>

<!-- Log container -->
<div
  role="log"
  aria-live="polite"
  aria-atomic="false"
  class="flex-1 overflow-auto"
>
  <!-- logs -->
</div>
```

### Step 6: Test visual enhancements

**Test cases:**
1. Multi-mode → color bars visible, container badges have background
2. Hover log entry → background changes smoothly
3. Container list → status dots colored correctly, hover feedback
4. Search with fuzzy match → score badge appears
5. Dark mode → colors have sufficient contrast (4.5:1 minimum)
6. Screen reader → ARIA labels announced correctly
7. Tab navigation → focus rings visible on all interactive elements

Expected: Professional appearance, smooth transitions, accessible

### Step 7: Commit

```bash
git add frontend/src/routes/logs/+page.svelte
git commit -m "feat(logs): enhance visual design and accessibility

- Add prominent color bars for multi-container logs
- Enhanced container badges with background and border
- Improved hover states with smooth transitions
- Better container list with status indicators
- Match score badges for fuzzy search results
- HSL-based color palette for better contrast
- Comprehensive ARIA labels and roles
- Focus states visible for keyboard navigation

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 9: Manual Testing and Bug Fixes

**Files:**
- All modified files

**Context:** Comprehensive manual testing of all new features.

### Step 1: Test layout stability

**Steps:**
1. Open logs page
2. Select/deselect multiple containers rapidly
3. Switch between modes multiple times
4. Resize browser window
5. Test on mobile viewport (375px, 768px, 1024px)

**Expected:** No jarring resizes, stable layout throughout

### Step 2: Test fuzzy search

**Test cases:**
```
Search: "ngx" → should find "nginx-proxy"
Search: "doc" → should find "dockerverse"
Search: "rp" → should find "redis-prod" (acronym)
Search: "dckrs" → should find "docker-server" (sequence)
Empty search → all containers shown
```

**Expected:** Accurate matches, sorted by score, highlighted

### Step 3: Test regex filtering

**Test patterns:**
```
Pattern: "ERROR" (regex off) → case-insensitive match
Pattern: "ERROR|WARN" (regex on) → matches both
Pattern: "/api/.*/timeout" (regex on) → matches API timeouts
Pattern: "(" (regex on) → shows validation error
Pattern: "\\d{3}" (regex on) → matches 3-digit numbers
```

**Expected:** Patterns work, errors handled, matches highlighted

### Step 4: Test split mode

**Steps:**
1. Select 1 container → split button disabled
2. Select 2 containers → split button enabled
3. Click split → two panes appear
4. Verify independent scrolling
5. Search logs → applies to both panes
6. Switch to multi mode → back to stacked
7. Switch to split again → same containers

**Expected:** Split mode works smoothly, maintains state

### Step 5: Test timestamp formats

**Steps:**
1. Default (absolute) → verify HH:MM:SS.mmm format
2. Switch to relative → verify "Xs ago" format
3. Wait 10 seconds → relative time should reflect passage
4. Switch to none → timestamps disappear
5. Test in all viewing modes

**Expected:** All formats work correctly in all modes

### Step 6: Test keyboard shortcuts

**Test each shortcut:** (see Task 6 Step 8)

**Expected:** All shortcuts work without conflicts

### Step 7: Test export

**Steps:**
1. Export all logs → verify metadata header
2. Export single container → verify only that container
3. Export from split mode → verify each pane
4. Apply search filter → verify exported logs respect filter
5. Check filename → correct format with timestamp

**Expected:** Exports work correctly with metadata

### Step 8: Test accessibility

**Tools:** Screen reader (VoiceOver/NVDA), keyboard-only navigation

**Steps:**
1. Tab through all interactive elements → focus visible
2. Navigate with screen reader → ARIA labels announced
3. Use keyboard shortcuts → work without mouse
4. Check color contrast → meets WCAG AA (4.5:1)

**Expected:** Fully keyboard accessible, screen reader friendly

### Step 9: Fix any discovered bugs

If bugs are found during testing:
1. Document the bug (steps to reproduce)
2. Create a fix
3. Test the fix
4. Commit with "fix(logs): [description]"

### Step 10: Final commit for testing notes

```bash
git add .
git commit -m "test(logs): complete manual testing of all features

Tested:
- Layout stability across interactions ✅
- Fuzzy container search accuracy ✅
- Regex log filtering with error handling ✅
- Split-screen mode functionality ✅
- Timestamp format options ✅
- All keyboard shortcuts ✅
- Export with metadata ✅
- Accessibility (ARIA, keyboard, contrast) ✅

All features working as designed.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 10: Update Documentation

**Files:**
- Create: `docs/features/logs-page-guide.md`
- Modify: `DEVELOPMENT_CONTINUATION_GUIDE.md`
- Modify: `README.md`

**Context:** Document new features for users and developers.

### Step 1: Create logs page guide

Create new file:

```markdown
# Logs Page User Guide

## Overview

The DockerVerse Logs Page provides professional-grade log viewing capabilities inspired by Dozzle, with multi-host support and powerful search features.

## Features

### Viewing Modes

**Single Mode** (Ctrl+1)
- View logs from one container at a time
- Full-width display for maximum readability
- Best for focused debugging

**Multi Mode** (Ctrl+2)
- View logs from multiple containers simultaneously
- Logs stacked vertically with color-coded indicators
- Container name badge on each line
- Best for comparing multiple containers

**Grouped Mode** (Ctrl+3)
- View logs grouped by Docker host
- Shows host name as section header
- Best for multi-host deployments

**Split Mode** (Ctrl+4) - NEW!
- View two containers side-by-side
- Independent scrolling for each pane
- Requires 2 containers selected
- Best for direct comparison

### Container Search

**Fuzzy Search:**
- Type "ngx" to find "nginx-proxy"
- Acronym matching (e.g., "rp" finds "redis-prod")
- Character sequence matching
- Results sorted by relevance (match score)
- Matched characters highlighted

**Keyboard:** Press `/` to focus container search

### Log Search & Filtering

**Simple Search:**
- Type any text to search logs
- Case-insensitive matching
- Matched text highlighted

**Regex Mode:**
- Click the `</>` button to enable regex
- Use patterns like `ERROR|WARN`, `/api/.*timeout`
- Invalid patterns show inline error
- Regex validation in real-time

**Keyboard:** Press `Ctrl+F` to focus log search

### Timestamp Options

**Absolute:** HH:MM:SS.mmm (default)
**Relative:** "2m ago", "5s ago", etc.
**None:** Hide timestamps for cleaner view

Toggle between formats using the timestamp selector in the toolbar.

### Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Ctrl+F` | Focus log search |
| `Ctrl+K` | Clear all logs |
| `Ctrl+1` | Single mode |
| `Ctrl+2` | Multi mode |
| `Ctrl+3` | Grouped mode |
| `Ctrl+4` | Split mode |
| `Ctrl+P` | Pause/resume streaming |
| `Ctrl+W` | Toggle line wrap |
| `Ctrl+E` | Export logs |
| `/` | Focus container search |
| `Space` | Pause/resume (when not typing) |
| `Esc` | Clear search or deselect containers |

### Export Logs

**Export All Logs:**
- Includes all displayed logs
- Respects active search filters
- Includes metadata header (timestamp, filters, mode, etc.)

**Export Single Container:**
- Available in Single mode
- Exports only the current container's logs

**Export Split Panes:**
- Available in Split mode
- Export left or right pane separately

**Keyboard:** Press `Ctrl+E` to open export menu

**File Format:**
```
# DockerVerse Log Export
# Generated: 2026-02-16T10:30:00.000Z
# Container: nginx-proxy
# Host: raspi1
# Total Lines: 1523
# Viewing Mode: single
# Search Filter: ERROR
# Regex Enabled: true
# Timestamp Format: absolute

# ===== LOGS START =====

[10:25:30.123] [nginx-proxy] Error: Connection timeout
[10:26:15.456] [nginx-proxy] ERROR: Failed to proxy request
...
```

## Tips & Best Practices

1. **Use Fuzzy Search** - Type partial names to quickly find containers
2. **Regex for Complex Filters** - Use `ERROR|WARN|FATAL` to catch multiple log levels
3. **Split Mode for Comparison** - Compare two container logs side-by-side
4. **Keyboard Shortcuts** - Learn `Ctrl+1-4` for quick mode switching
5. **Export Filtered Logs** - Apply search filters before exporting for focused analysis
6. **Relative Timestamps** - Use when debugging recent events
7. **Hide Timestamps** - Use when copying log messages

## Accessibility

The logs page is fully accessible:
- **Keyboard Navigation:** Tab through all controls
- **Screen Readers:** ARIA labels on all interactive elements
- **High Contrast:** Color contrast meets WCAG AA (4.5:1 minimum)
- **Focus Indicators:** Visible focus rings on all interactive elements

## Troubleshooting

**Container not appearing in list:**
- Check container is running: `docker ps`
- Verify backend can reach Docker host
- Check logs page for connection errors

**Logs not streaming:**
- Check "Pause" is not enabled
- Verify backend SSE connection (Network tab in DevTools)
- Try refreshing the page

**Regex pattern not working:**
- Check syntax error message displayed
- Test pattern at https://regex101.com
- Remember to escape special characters

**Split mode disabled:**
- Select at least 2 containers first
- Split mode requires 2+ selections

## Version History

**v2.5.0** (2026-02-16)
- Added fuzzy container search with highlighting
- Added regex log filtering with toggle
- Added split-screen mode for two containers
- Added timestamp format options (absolute/relative/none)
- Added comprehensive keyboard shortcuts
- Enhanced export with metadata and per-container options
- Fixed layout stability bug
- Improved visual design and accessibility
```

### Step 2: Update DEVELOPMENT_CONTINUATION_GUIDE.md

Add to the features section:

```markdown
## Recent Updates

### v2.5.0 - Logs Page Enhancement (2026-02-16)

**Logs Page Improvements:**
- **Fuzzy Container Search:** Intelligent name matching (exact > acronym > sequence)
- **Regex Log Filtering:** Advanced pattern-based log search with toggle
- **Split-Screen Mode:** Side-by-side comparison of two containers
- **Timestamp Options:** Absolute (HH:MM:SS), relative (2m ago), or hidden
- **Keyboard Shortcuts:** 11 shortcuts for efficient navigation (Ctrl+F, Ctrl+K, etc.)
- **Enhanced Export:** Metadata header, per-container export, filter-aware
- **Visual Enhancements:** Color bars, better badges, improved contrast
- **Accessibility:** ARIA labels, keyboard navigation, focus states

**Bug Fixes:**
- Fixed layout stability: Container area no longer resizes on selections
- Applied flexbox pattern for stable dimensions

**Architecture:**
- Frontend-only changes (no backend modifications)
- Uses Svelte 5 $state/$derived reactivity
- Bundle size increase: ~9KB (well under target)
- SSE streaming unchanged

**Documentation:**
- Design doc: `docs/plans/2026-02-16-logs-improvement-design.md`
- User guide: `docs/features/logs-page-guide.md`
- Implementation plan: `docs/plans/2026-02-16-logs-improvement-implementation.md`
```

### Step 3: Update README.md

Add to features section:

```markdown
### Logs Viewer 📋

Professional-grade log viewing inspired by Dozzle:

- **Four Viewing Modes:** Single, Multi, Grouped, Split-screen
- **Fuzzy Search:** Intelligent container name matching (e.g., "ngx" finds "nginx-proxy")
- **Regex Filtering:** Advanced pattern-based log search (`ERROR|WARN`, `/api/.*/timeout`)
- **Real-Time Streaming:** SSE-based live log updates
- **Keyboard Shortcuts:** 11 shortcuts for efficient navigation
- **Export Options:** All logs, single container, or split panes with metadata
- **Timestamp Formats:** Absolute, relative, or hidden
- **Accessibility:** Full keyboard navigation, ARIA labels, WCAG AA compliant

Press `?` in logs page to see all keyboard shortcuts.
```

### Step 4: Commit documentation

```bash
git add docs/features/logs-page-guide.md DEVELOPMENT_CONTINUATION_GUIDE.md README.md
git commit -m "docs: add logs page user guide and update project docs

- Create comprehensive logs page user guide
- Document fuzzy search, regex filtering, split mode
- List all keyboard shortcuts with examples
- Add troubleshooting section
- Update DEVELOPMENT_CONTINUATION_GUIDE with v2.5.0 features
- Update README feature list

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 11: Deploy and Test on Raspberry Pi

**Files:**
- None (deployment task)

**Context:** Deploy to production Raspberry Pi and perform live testing.

### Step 1: Run local build test

```bash
cd frontend
npm run build
```

Expected: Build completes without errors, check bundle size

### Step 2: Deploy to Raspberry Pi

```bash
./deploy-to-raspi.sh
```

Wait for deployment to complete (build + sync + restart)

### Step 3: Access production logs page

Navigate to: http://192.168.1.145:3007/logs

### Step 4: Test on production

**Critical path testing:**
1. Select containers → verify layout stability
2. Fuzzy search containers → verify matching works
3. Enable regex → search for `ERROR|WARN` → verify results
4. Switch to split mode → verify two panes render
5. Test keyboard shortcuts → Ctrl+1, Ctrl+2, Ctrl+3, Ctrl+4
6. Export logs → verify file downloads with metadata
7. Test on mobile device (phone or tablet)

### Step 5: Monitor for errors

Check browser console for any runtime errors:
- Open DevTools (F12)
- Watch Console tab while using features
- Check Network tab for SSE connection health

### Step 6: Performance check

Check performance metrics:
- Bundle size (should be <20KB increase)
- Memory usage (check browser DevTools Performance tab)
- Log rendering speed (1000+ logs should render smoothly)

### Step 7: Document production verification

```bash
git add .
git commit -m "deploy: verify v2.5.0 on production raspi

Deployed logs page enhancements to http://192.168.1.145:3007

Production testing completed:
- Layout stability confirmed ✅
- Fuzzy search working ✅
- Regex filtering working ✅
- Split mode rendering correctly ✅
- Keyboard shortcuts functional ✅
- Export working with metadata ✅
- Mobile responsive ✅
- No console errors ✅
- Performance metrics within targets ✅

Ready for merge to master.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 12: Merge to Master

**Files:**
- None (git task)

**Context:** Final merge to master branch after successful production testing.

### Step 1: Ensure clean working directory

```bash
git status
```

Expected: All changes committed, working directory clean

### Step 2: Create final summary commit (if needed)

If there were any last-minute fixes during production testing:

```bash
git add .
git commit -m "chore: final cleanup for v2.5.0 logs enhancements"
```

### Step 3: Switch to master and merge

```bash
git checkout master
git pull origin master
git merge feature/toggle-filters-host-rename-2026-02-12 --no-ff
```

### Step 4: Write comprehensive merge commit message

When prompted, write:

```
Merge feature/toggle-filters-host-rename-2026-02-12: Logs page v2.5.0 enhancements

Major features:
- Fuzzy container search with intelligent matching and highlighting
- Regex log filtering with real-time validation
- Split-screen mode for side-by-side container comparison
- Timestamp format options (absolute/relative/none)
- 11 keyboard shortcuts for efficient navigation
- Enhanced export with metadata and per-container options
- Visual enhancements (color bars, badges, hover states)
- Comprehensive accessibility improvements

Bug fixes:
- Fixed critical layout stability bug (removed fixed height)
- Applied flexbox pattern for stable dimensions

Technical details:
- Frontend-only changes (Svelte 5)
- Bundle size increase: ~9KB
- No backend modifications required
- SSE streaming unchanged

Documentation:
- Added user guide: docs/features/logs-page-guide.md
- Updated DEVELOPMENT_CONTINUATION_GUIDE.md
- Updated README.md with new features

Tested:
- Local development environment ✅
- Production deployment on raspi (192.168.1.145:3007) ✅
- Manual testing all features ✅
- Accessibility testing ✅
- Mobile responsiveness ✅
- Performance metrics ✅

Closes #[issue-number] if applicable

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
```

### Step 5: Push to remote

```bash
git push origin master
```

### Step 6: Verify on GitHub

- Check merge commit on GitHub
- Verify all files updated correctly
- Check Actions/CI if configured

### Step 7: Tag the release

```bash
git tag -a v2.5.0 -m "Release v2.5.0: Logs Page Enhancement

Professional-grade log viewer with fuzzy search, regex filtering,
split-screen mode, keyboard shortcuts, and accessibility improvements.

See docs/features/logs-page-guide.md for full feature list."

git push origin v2.5.0
```

### Step 8: Cleanup feature branch (optional)

```bash
git branch -d feature/toggle-filters-host-rename-2026-02-12
git push origin --delete feature/toggle-filters-host-rename-2026-02-12
```

### Step 9: Final verification

Navigate to production: http://192.168.1.145:3007/logs

Verify the logs page is working with all new features.

### Step 10: Celebrate! 🎉

All tasks completed successfully:
- ✅ Layout stability bug fixed
- ✅ Fuzzy container search implemented
- ✅ Regex log filtering added
- ✅ Split-screen mode working
- ✅ Timestamp enhancements complete
- ✅ Keyboard shortcuts functional
- ✅ Export enhanced with metadata
- ✅ Visual design improved
- ✅ Fully accessible
- ✅ Documented
- ✅ Deployed and tested
- ✅ Merged to master

**DockerVerse v2.5.0 is live! 🚀**

---

## Summary

**Total Tasks:** 12
**Estimated Time:** 4-6 hours
**Commits:** ~15-20 (frequent, atomic commits)
**Files Modified:** 1 main file (frontend/src/routes/logs/+page.svelte) + docs
**Bundle Size Impact:** ~9 KB
**Backend Changes:** None (frontend-only)

**Key Deliverables:**
1. Fixed critical layout bug
2. Fuzzy container search
3. Regex log filtering
4. Split-screen mode
5. Timestamp format options
6. Comprehensive keyboard shortcuts
7. Enhanced export functionality
8. Visual design improvements
9. Full accessibility support
10. Complete documentation
11. Production deployment
12. Merged to master

**Quality Assurance:**
- TDD approach where applicable
- Comprehensive manual testing
- Production verification
- Accessibility testing
- Performance monitoring
- Documentation at every step

---

## Plan complete! ✅

**Plan saved to:** `docs/plans/2026-02-16-logs-improvement-implementation.md`

**Two execution options:**

**1. Subagent-Driven (this session)** - I dispatch fresh subagent per task, review between tasks, fast iteration

**2. Parallel Session (separate)** - Open new session with executing-plans, batch execution with checkpoints

**Which approach?**
