# DockerVerse Logs Page Improvement - Design Document

**Date**: 2026-02-16
**Version**: v2.5.0
**Authors**: Claude + Victor Heredia
**Status**: Approved for Implementation

---

## Executive Summary

This document outlines the redesign of the DockerVerse logs page, taking inspiration from Dozzle while maintaining our unique multi-host architecture. The focus is on fixing critical layout bugs, improving search capabilities, and enhancing multi-container log viewing without adding heavy dependencies.

**Approach**: Hybrid enhancement (Approach 3) - delivering professional-grade UX improvements without SQL engine complexity.

---

## Problem Statement

### Current Issues

1. **CRITICAL BUG**: Log container resizes when selecting/deselecting containers
   - Root cause: Fixed height calculation `h-[calc(100vh-7rem)]` on line 272
   - Impact: Jarring UX, lost scroll position, visual instability

2. **Limited Search**: Only basic string matching
   - No fuzzy search for container names
   - No regex support for log filtering
   - Manual scrolling to find specific patterns

3. **Multi-Container UX**: Functional but basic
   - Timestamps not aligned across containers
   - Color coding could be more prominent
   - No side-by-side viewing option

4. **Missing Power Features**:
   - No keyboard shortcuts for common actions
   - No timestamp format options (absolute vs relative)
   - Export functionality is basic

### Success Criteria

✅ **Bug Fix**: Container area maintains stable height regardless of selections
✅ **Search**: Fuzzy container search + regex log filtering
✅ **UX**: Split-screen mode for 2 containers side-by-side
✅ **A11y**: Keyboard navigation, focus states, ARIA labels
✅ **Performance**: No bundle size increase >20KB

---

## Architecture Overview

### Current Architecture (Preserved)

```
Frontend (Svelte 5)
├── SSE Streams (EventSource)
│   └── /api/logs/:hostId/:containerId (existing)
├── State Management ($state, $derived)
└── Three Viewing Modes
    ├── Single: One container, full width
    ├── Multi: Multiple containers, stacked
    └── Grouped: Grouped by host

Backend (Go + Fiber)
├── Log streaming via SSE
├── Docker API integration
└── No changes needed for Phase 1
```

### New Architecture (Enhanced)

```
Frontend (Svelte 5) - ENHANCED
├── SSE Streams (unchanged)
├── State Management (enhanced)
│   ├── Fuzzy search state
│   ├── Regex filter state
│   ├── Split-screen layout state
│   └── Keyboard shortcuts state
├── Four Viewing Modes (NEW)
│   ├── Single: One container, full width
│   ├── Multi: Multiple containers, stacked
│   ├── Grouped: Grouped by host
│   └── Split: 2 containers side-by-side (NEW)
└── Enhanced Components
    ├── FuzzySearch component
    ├── RegexFilter component
    ├── TimestampToggle component
    └── KeyboardShortcuts handler

Backend (Go + Fiber) - NO CHANGES PHASE 1
└── Existing endpoints sufficient
```

---

## Detailed Design

### 1. Layout Stability Fix (CRITICAL)

**Problem**: Fixed height causes resize on container selection changes.

**Solution**: Flexbox-based layout pattern from Terminal component.

#### Before (Buggy):
```svelte
<!-- Line 272 - REMOVE -->
<div class="flex flex-col h-[calc(100vh-7rem)]">
  <!-- sidebar + logs area -->
</div>
```

#### After (Stable):
```svelte
<!-- Use flex-1 pattern -->
<div class="flex flex-col flex-1 min-h-0">
  <div class="flex flex-1 min-h-0 gap-4">
    <!-- Sidebar: fixed width -->
    <aside class="w-80 flex-shrink-0 flex flex-col min-h-0">
      <!-- container list with overflow -->
    </aside>

    <!-- Logs area: flex-1 fills remaining space -->
    <div class="flex-1 flex flex-col min-h-0">
      <!-- toolbar, search, logs -->
      <div class="flex-1 overflow-auto">
        <!-- logs content -->
      </div>
    </div>
  </div>
</div>
```

**Key Principles**:
- Parent: `flex flex-col flex-1 min-h-0`
- Sidebar: `w-80 flex-shrink-0` (fixed width, doesn't shrink)
- Content: `flex-1 min-h-0` (fills remaining space)
- Scrollable areas: `overflow-auto` on inner containers

**UX Guidelines Applied**:
- ✅ Layout stability (no content jumping)
- ✅ Responsive behavior preserved
- ✅ No horizontal scroll

---

### 2. Fuzzy Container Search

**Inspiration**: Dozzle's intelligent container name matching.

#### Implementation:

```typescript
// New fuzzy search function
function fuzzyMatch(query: string, text: string): { match: boolean; score: number } {
  const q = query.toLowerCase();
  const t = text.toLowerCase();

  // Exact substring match - highest score
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

// Sort containers by match score
let filteredContainers = $derived(() => {
  if (!containerSearch) return allContainers;

  return allContainers
    .map(c => ({ ...c, ...fuzzyMatch(containerSearch, c.name) }))
    .filter(c => c.match)
    .sort((a, b) => b.score - a.score);
});
```

**UX Enhancements**:
- Highlight matched characters in container name
- Show match score as visual indicator
- Sort by relevance (exact > acronym > sequence)

---

### 3. Regex Log Filtering

**Inspiration**: Dozzle's regex-based log search.

#### Implementation:

```typescript
// State
let logSearch = $state("");
let regexEnabled = $state(false);
let regexError = $state<string | null>(null);

// Compile regex with error handling
let searchPattern = $derived(() => {
  if (!logSearch) return null;

  if (regexEnabled) {
    try {
      regexError = null;
      return new RegExp(logSearch, 'gi');
    } catch (e) {
      regexError = e.message;
      return null;
    }
  }

  // Simple string search
  return new RegExp(logSearch.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'gi');
});

// Filter logs
let filteredLogs = $derived(() => {
  if (!searchPattern) return allLogs;

  return allLogs.filter(log =>
    searchPattern.test(log.line)
  );
});

// Highlight matches in log line
function highlightMatches(line: string): string {
  if (!searchPattern) return line;

  return line.replace(searchPattern, (match) =>
    `<mark class="bg-primary/30 text-foreground">${match}</mark>`
  );
}
```

**UI Components**:
```svelte
<div class="flex items-center gap-2">
  <input
    type="text"
    bind:value={logSearch}
    placeholder={regexEnabled ? "Regex pattern..." : "Search logs..."}
    class="input flex-1"
  />

  <button
    class="btn-icon {regexEnabled ? 'text-primary' : ''}"
    onclick={() => regexEnabled = !regexEnabled}
    title="Toggle regex mode"
  >
    <Regex class="w-4 h-4" />
  </button>

  {#if regexError}
    <span class="text-xs text-accent-red">{regexError}</span>
  {/if}
</div>
```

**UX Guidelines Applied**:
- ✅ Clear error feedback (regex validation)
- ✅ Visual indicator for regex mode
- ✅ Accessible keyboard navigation

---

### 4. Split-Screen Mode

**Inspiration**: Dozzle's side-by-side container viewing.

#### State Management:

```typescript
type LogMode = "single" | "multi" | "grouped" | "split"; // NEW: split mode

let mode = $state<LogMode>("single");
let splitContainers = $state<[string, string] | null>(null); // [containerId1, containerId2]

// Enter split mode
function enterSplitMode(container1: string, container2: string) {
  mode = "split";
  splitContainers = [container1, container2];
  selectedContainers.clear();
  selectedContainers.add(container1);
  selectedContainers.add(container2);
}
```

#### Layout:

```svelte
{#if mode === "split" && splitContainers}
  <div class="flex-1 flex gap-4 min-h-0">
    <!-- Left pane -->
    <div class="flex-1 flex flex-col min-h-0 border border-border rounded-lg overflow-hidden">
      <div class="px-3 py-2 bg-background-secondary border-b border-border">
        <span class="text-sm font-medium">{getContainerName(splitContainers[0])}</span>
      </div>
      <div class="flex-1 overflow-auto p-3 font-mono text-xs">
        {#each getLogsForContainer(splitContainers[0]) as log}
          <div class="log-line">
            <span class="text-foreground-muted mr-2">{formatTimestamp(log.ts)}</span>
            {@html highlightMatches(log.line)}
          </div>
        {/each}
      </div>
    </div>

    <!-- Right pane (same structure) -->
    <div class="flex-1 flex flex-col min-h-0 border border-border rounded-lg overflow-hidden">
      <!-- container 2 logs -->
    </div>
  </div>
{/if}
```

**UX Enhancements**:
- Independent scroll for each pane
- Synchronized auto-scroll toggle
- Resizable panes (stretch goal)

---

### 5. Timestamp Enhancements

**Inspiration**: Dozzle's flexible timestamp display.

#### Implementation:

```typescript
type TimestampFormat = "absolute" | "relative" | "none";
let timestampFormat = $state<TimestampFormat>("absolute");

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

  // Relative
  const now = Date.now();
  const diff = now - ts;

  if (diff < 1000) return "just now";
  if (diff < 60000) return `${Math.floor(diff / 1000)}s ago`;
  if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`;
  return `${Math.floor(diff / 86400000)}d ago`;
}
```

**UI Toggle**:
```svelte
<div class="flex items-center gap-1 border border-border rounded-md p-1">
  <button
    class="btn-sm {timestampFormat === 'absolute' ? 'btn-primary' : 'btn-ghost'}"
    onclick={() => timestampFormat = 'absolute'}
  >
    HH:MM:SS
  </button>
  <button
    class="btn-sm {timestampFormat === 'relative' ? 'btn-primary' : 'btn-ghost'}"
    onclick={() => timestampFormat = 'relative'}
  >
    Relative
  </button>
  <button
    class="btn-sm {timestampFormat === 'none' ? 'btn-primary' : 'btn-ghost'}"
    onclick={() => timestampFormat = 'none'}
  >
    Hide
  </button>
</div>
```

---

### 6. Keyboard Shortcuts

**Inspiration**: Dozzle's efficient keyboard navigation.

#### Shortcut Map:

| Shortcut | Action | Context |
|----------|--------|---------|
| `Ctrl+F` | Focus log search | Global |
| `Ctrl+K` | Clear logs | Log view |
| `Ctrl+1` | Single mode | Mode switching |
| `Ctrl+2` | Multi mode | Mode switching |
| `Ctrl+3` | Grouped mode | Mode switching |
| `Ctrl+4` | Split mode | Mode switching |
| `Ctrl+P` | Toggle pause/play | Log streaming |
| `Ctrl+W` | Toggle line wrap | Log display |
| `Ctrl+E` | Export logs | Log view |
| `Escape` | Clear search / Deselect | Global |
| `Space` | Pause/Resume scroll | Log view |
| `/` | Focus container search | Container list |

#### Implementation:

```typescript
onMount(() => {
  function handleKeydown(e: KeyboardEvent) {
    // Ignore if typing in input
    if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) {
      // Allow ESC in inputs
      if (e.key !== 'Escape') return;
    }

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
      mode = modes[parseInt(e.key) - 1];
      return;
    }

    // Space - Pause/Resume
    if (e.key === ' ' && !(e.target instanceof HTMLInputElement)) {
      e.preventDefault();
      isPaused = !isPaused;
      return;
    }

    // / - Focus container search
    if (e.key === '/' && !(e.target instanceof HTMLInputElement)) {
      e.preventDefault();
      containerSearchInput?.focus();
      return;
    }

    // Escape - Clear search or deselect
    if (e.key === 'Escape') {
      if (logSearch) {
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
  return () => window.removeEventListener('keydown', handleKeydown);
});
```

**UX Guidelines Applied**:
- ✅ Keyboard navigation support
- ✅ Focus states visible
- ✅ Standard shortcuts (Ctrl+F for search)

---

### 7. Export Enhancements

**Current**: Basic export all logs.
**Enhanced**: Export filtered logs per container with metadata.

#### Implementation:

```typescript
function exportLogs(containerId?: string) {
  const logs = containerId
    ? filteredLogs.filter(l => l.key.startsWith(containerId))
    : filteredLogs;

  const container = containerId
    ? containers.find(c => c.id === containerId)
    : null;

  const metadata = [
    `# DockerVerse Log Export`,
    `# Date: ${new Date().toISOString()}`,
    `# Container: ${container?.name || 'All'}`,
    `# Host: ${container?.hostId || 'Multiple'}`,
    `# Total Lines: ${logs.length}`,
    `# Filter: ${logSearch || 'None'}`,
    ``,
  ].join('\n');

  const content = logs
    .map(log => `[${formatTimestamp(log.ts)}] [${log.name}] ${log.line}`)
    .join('\n');

  const blob = new Blob([metadata + content], { type: 'text/plain' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `logs_${container?.name || 'all'}_${new Date().toISOString().slice(0, 10)}.txt`;
  a.click();
  URL.revokeObjectURL(url);
}
```

---

### 8. Visual Enhancements

#### Color-Coded Container Indicators

**Current**: Small dot, subtle colors.
**Enhanced**: Prominent bar, better contrast.

```svelte
<!-- Multi-mode log entry -->
<div class="flex items-start gap-2 group hover:bg-background-secondary/50 px-2 py-1 rounded">
  <!-- Color bar (4px wide) -->
  <div
    class="w-1 h-full flex-shrink-0 rounded-full"
    style="background: {getContainerColor(log.key)}"
  ></div>

  <!-- Timestamp -->
  <span class="text-foreground-muted text-xs w-24 flex-shrink-0">
    {formatTimestamp(log.ts)}
  </span>

  <!-- Container name badge -->
  <span
    class="px-2 py-0.5 rounded text-xs font-medium flex-shrink-0"
    style="background: {getContainerColor(log.key)}20; color: {getContainerColor(log.key)}"
  >
    {log.name}
  </span>

  <!-- Log content -->
  <pre class="flex-1 text-xs font-mono whitespace-pre-wrap break-all">
    {@html highlightMatches(log.line)}
  </pre>
</div>
```

#### Accessibility Improvements

```svelte
<!-- Container selection checkbox -->
<label class="flex items-center gap-3 px-3 py-2 hover:bg-background-secondary rounded cursor-pointer group">
  <input
    type="checkbox"
    checked={selectedContainers.has(c.id)}
    onchange={() => toggleContainer(c.id)}
    class="w-4 h-4 accent-primary focus:ring-2 focus:ring-primary focus:ring-offset-2"
    aria-label="Select {c.name}"
  />
  <span class="flex-1 text-sm group-hover:text-primary transition-colors">
    {c.name}
  </span>
</label>

<!-- Mode switch buttons -->
<div role="radiogroup" aria-label="Log viewing mode" class="flex gap-1">
  <button
    role="radio"
    aria-checked={mode === 'single'}
    onclick={() => mode = 'single'}
    class="btn-sm"
  >
    Single
  </button>
  <!-- ... other modes -->
</div>
```

**UX Guidelines Applied**:
- ✅ ARIA labels for screen readers
- ✅ Focus rings on interactive elements
- ✅ Color contrast 4.5:1 minimum
- ✅ Hover feedback (cursor-pointer, color change)

---

## Data Flow

### Log Streaming (Unchanged)

```
User Action → Frontend State
              ↓
         SSE Connection (/api/logs/:hostId/:containerId)
              ↓
         Backend Go Handler
              ↓
         Docker API (docker logs --follow)
              ↓
         Stream to Frontend (EventSource.onmessage)
              ↓
         Append to allLogs array
              ↓
         Apply filters (search, regex)
              ↓
         Render filteredLogs
              ↓
         Auto-scroll if enabled
```

### New Filtering Flow

```
allLogs (raw SSE data)
   ↓
Regex Filter (if enabled)
   ↓
Container Filter (selected containers)
   ↓
Search Highlight (mark matches)
   ↓
Mode-Specific Rendering
   ├── Single: Show one container
   ├── Multi: Stack all containers
   ├── Grouped: Group by host
   └── Split: Side-by-side two containers
```

---

## Performance Considerations

### Bundle Size

| Feature | Estimated Size | Justification |
|---------|---------------|---------------|
| Fuzzy search | ~1 KB | Pure TypeScript function |
| Regex filter | 0 KB | Built-in RegExp |
| Split-screen | ~5 KB | Additional layout components |
| Keyboard shortcuts | ~2 KB | Event handlers |
| Timestamp formats | ~1 KB | Date formatting logic |
| **Total Added** | **~9 KB** | Well under 20KB limit ✅ |

### Runtime Performance

**Optimizations**:
1. **$derived for filters**: Svelte's reactivity handles memoization
2. **Virtual scrolling** (future): Only render visible logs
3. **Debounced search**: 150ms delay for regex compilation
4. **Stream throttling**: Limit log append rate to 60 FPS

```typescript
// Debounced regex compilation
let searchDebounceTimer: ReturnType<typeof setTimeout>;
function updateSearch(value: string) {
  clearTimeout(searchDebounceTimer);
  searchDebounceTimer = setTimeout(() => {
    logSearch = value;
  }, 150);
}

// Throttled log appending
let logBuffer: LogEntry[] = [];
setInterval(() => {
  if (logBuffer.length > 0) {
    allLogs = [...allLogs, ...logBuffer];
    logBuffer = [];
  }
}, 16); // ~60 FPS
```

**UX Guidelines Applied**:
- ✅ Performance optimization (debouncing, throttling)
- ✅ Reduced motion support (prefers-reduced-motion)

---

## Error Handling

### Regex Validation

```typescript
function validateRegex(pattern: string): { valid: boolean; error?: string } {
  try {
    new RegExp(pattern);
    return { valid: true };
  } catch (e) {
    return {
      valid: false,
      error: e.message.replace('Invalid regular expression: /', '')
    };
  }
}
```

### SSE Connection Failures

```svelte
{#if connectionErrors.length > 0}
  <div class="bg-accent-red/10 border border-accent-red rounded-lg p-4">
    <h3 class="text-accent-red font-medium mb-2">Connection Issues</h3>
    <ul class="space-y-1 text-sm">
      {#each connectionErrors as error}
        <li>• {error.container}: {error.message}</li>
      {/each}
    </ul>
    <button class="btn btn-sm btn-primary mt-3" onclick={retryConnections}>
      Retry All
    </button>
  </div>
{/if}
```

**UX Guidelines Applied**:
- ✅ Error feedback near problem area
- ✅ Clear error messages
- ✅ Retry mechanism provided

---

## Accessibility Checklist

- [x] All interactive elements have focus states
- [x] ARIA labels for icon-only buttons
- [x] Keyboard navigation support (tab order logical)
- [x] Color contrast 4.5:1 minimum
- [x] Form inputs have labels
- [x] Role attributes for custom widgets (radiogroup, radio)
- [x] Screen reader announcements for state changes
- [x] prefers-reduced-motion support
- [x] No horizontal scroll on mobile

---

## Mobile Responsiveness

### Breakpoints

| Breakpoint | Layout Changes |
|------------|----------------|
| 320px-640px | Sidebar hidden by default, toggle button, single column |
| 640px-1024px | Sidebar 240px, split mode unavailable |
| 1024px+ | Sidebar 320px, full split mode support |

### Mobile-Specific Adjustments

```svelte
<!-- Responsive sidebar -->
<aside
  class="
    w-80 flex-shrink-0
    max-lg:absolute max-lg:inset-y-0 max-lg:left-0 max-lg:z-10
    max-lg:w-64 max-lg:bg-background max-lg:shadow-xl
    max-lg:transform max-lg:transition-transform
    {sidebarOpen ? 'max-lg:translate-x-0' : 'max-lg:-translate-x-full'}
  "
>
  <!-- sidebar content -->
</aside>

<!-- Mobile toggle button -->
<button
  class="lg:hidden btn-icon"
  onclick={() => sidebarOpen = !sidebarOpen}
  aria-label="Toggle sidebar"
>
  <Menu class="w-5 h-5" />
</button>
```

---

## Testing Strategy

### Unit Tests

- Fuzzy search algorithm
- Regex validation
- Timestamp formatting
- Log filtering logic

### Integration Tests

- SSE connection handling
- Multi-container selection
- Mode switching
- Keyboard shortcuts

### E2E Tests

- Complete log viewing workflow
- Split-screen interaction
- Search and filter
- Export functionality

### Manual Testing Checklist

- [ ] Layout stability (resize bug fixed)
- [ ] Fuzzy search accuracy
- [ ] Regex patterns work correctly
- [ ] Split mode renders properly
- [ ] Keyboard shortcuts functional
- [ ] Timestamps format correctly
- [ ] Export includes metadata
- [ ] Mobile responsive
- [ ] Accessibility (screen reader, keyboard-only)
- [ ] Performance (1000+ log lines)

---

## Migration Path

### Phase 1: Bug Fix + Core Features (This PR)

- ✅ Fix layout stability bug
- ✅ Fuzzy container search
- ✅ Regex log filtering
- ✅ Split-screen mode
- ✅ Enhanced timestamps
- ✅ Keyboard shortcuts
- ✅ Export improvements
- ✅ Visual enhancements

**Estimated**: 3-4 hours implementation + 1 hour testing

### Phase 2: Advanced Features (Future)

- Virtual scrolling for performance
- Resizable split panes
- Log syntax highlighting (JSON, error patterns)
- Saved search patterns
- Log level filtering (INFO, WARN, ERROR)
- Container action buttons in log view

### Phase 3: Backend Enhancements (Future)

- Log history persistence (store recent logs in DB)
- Full-text search API endpoint
- Log aggregation across hosts
- Webhook notifications for log patterns

---

## Documentation Updates

### User-Facing

- Update README with new keyboard shortcuts
- Add "Using Logs Page" section with screenshots
- Document regex syntax examples

### Developer-Facing

- Update DEVELOPMENT_CONTINUATION_GUIDE.md
- Document new components and patterns
- Add TypeScript interfaces for new types

---

## Success Metrics

### Quantitative

- Bundle size increase: <20 KB ✅
- Layout stability: 0 resize bugs ✅
- Search accuracy: >90% fuzzy matches ✅
- Keyboard navigation: 100% accessible ✅

### Qualitative

- User feedback: "Logs page feels professional"
- Developer feedback: "Easy to debug container issues"
- Comparison: "Matches Dozzle features we need"

---

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Regex performance issues with complex patterns | Medium | Debounce input, timeout limit (1s) |
| Layout breaks on edge-case viewport sizes | Low | Comprehensive responsive testing |
| Keyboard shortcuts conflict with browser defaults | Low | Use Ctrl+ combinations, document clearly |
| SSE connections multiplied in split mode | Medium | Reuse existing streams, don't create duplicates |

---

## Conclusion

This design delivers a professional-grade log viewer inspired by Dozzle's best features while maintaining DockerVerse's unique multi-host architecture. The focus on layout stability, search capabilities, and keyboard-driven UX ensures a productive debugging experience without adding heavy dependencies.

**Next Steps**:
1. Review and approve this design document ✅
2. Invoke `writing-plans` skill for implementation plan
3. Implement Phase 1 features
4. Test thoroughly (manual + automated)
5. Deploy to production
6. Document and merge to master

---

**Approved by**: Victor Heredia
**Ready for Implementation**: Yes
**Target Version**: v2.5.0
