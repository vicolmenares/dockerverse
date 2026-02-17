# DockerVerse Logs Page Enhancement - Design Document v2 (Combined)

**Date**: 2026-02-16
**Version**: v2.5.0 (Combined Dozzle + Dockhand)
**Authors**: Claude + Victor Heredia
**Status**: Approved for Implementation
**Inspiration**: Dozzle + Dockhand best features combined

---

## Executive Summary

This document outlines the **enhanced** redesign of the DockerVerse logs page, combining the best features from **Dozzle** (host filtering, stack grouping) and **Dockhand** (advanced log controls, viewing modes). The result is a professional-grade log viewer that maintains our unique multi-host architecture while delivering best-in-class UX.

**Approach**: Hybrid enhancement combining Dozzle's navigation patterns + Dockhand's log controls + our custom features (fuzzy search, regex, split mode, keyboard shortcuts).

---

## Problem Statement (Updated)

### Current Issues

1. **CRITICAL BUG**: Log container resizes when selecting/deselecting containers
   - Root cause: Fixed height calculation `h-[calc(100vh-7rem)]` on line 272
   - Impact: Jarring UX, lost scroll position, visual instability

2. **Limited Navigation**:
   - No host filtering in sidebar
   - Containers not grouped by Docker Compose stacks
   - Flat list makes navigation difficult with many containers

3. **Missing Dozzle Features**:
   - No breadcrumb navigation (Hosts > Host Name)
   - No stack-based grouping with expand/collapse
   - No visual indication of stack relationships

4. **Missing Dockhand Features**:
   - No advanced log controls (pixels/font size, detailed wrap controls)
   - Viewing modes could be more intuitive
   - Missing visual polish in log display

5. **Limited Search**:
   - No fuzzy search for container names
   - No regex support for log filtering
   - Manual scrolling to find specific patterns

### Success Criteria (Updated)

✅ **Bug Fix**: Container area maintains stable height regardless of selections
✅ **Navigation**: Host filtering + stack-based grouping like Dozzle
✅ **Controls**: Advanced log controls like Dockhand (pause, pixels, wrap, etc.)
✅ **Modes**: Keep Single/Multi/Grouped + add Split mode
✅ **Search**: Fuzzy container search + regex log filtering
✅ **UX**: Professional UI matching both Dozzle and Dockhand quality
✅ **A11y**: Keyboard navigation, focus states, ARIA labels
✅ **Performance**: No bundle size increase >25KB (slightly higher for extra features)

---

## Architecture Overview (Updated)

### Enhanced Architecture

```
Frontend (Svelte 5) - ENHANCED WITH DOZZLE + DOCKHAND PATTERNS
├── Navigation Layer (NEW - Dozzle-inspired)
│   ├── Host Selector (breadcrumb + dropdown)
│   ├── Stack Grouping (expand/collapse)
│   └── Container Hierarchy Display
├── Control Layer (NEW - Dockhand-inspired)
│   ├── Viewing Mode Selector (Single/Multi/Grouped/Split)
│   ├── Log Controls (Pause, Auto-scroll, Pixels, Wrap)
│   └── Advanced Options (Export, Clear, Settings)
├── Search Layer (Enhanced)
│   ├── Fuzzy container search
│   ├── Regex log filtering
│   └── Quick filters
├── Display Layer
│   ├── Single mode (one container)
│   ├── Multi mode (multiple stacked)
│   ├── Grouped mode (by host)
│   └── Split mode (side-by-side) - NEW
└── State Management ($state, $derived)
    ├── Selected host
    ├── Expanded stacks
    ├── Selected containers
    ├── Display preferences
    └── Log streaming (SSE)

Backend (Go + Fiber) - MINIMAL CHANGES
├── Existing SSE log streaming (unchanged)
├── New endpoint: /api/stacks (list Docker Compose stacks)
└── Enhanced: /api/containers (include stack metadata)
```

---

## Detailed Design: New Features

### 1. Host Filtering Sidebar (Dozzle-inspired)

**Layout Pattern**:
```
┌─────────────────────────────┐
│  Hosts >  [raspberry_pi_main]  │  ← Breadcrumb + dropdown
├─────────────────────────────┤
│  📦 dockpeek (2)          [▼]│  ← Stack with count, collapse button
│    ● dockpeek                │  ← Container 1
│    ● dockpeek-socket-proxy   │  ← Container 2
│                               │
│  📦 dozzle (2)            [▼]│
│    ● Dozzle                   │
│    ● Dozzle-Agent             │
│                               │
│  📦 monitoring_services (4) [▶]│  ← Collapsed stack
│                               │
│  📁 Standalone Containers    │  ← Non-stack containers
│    ● redis                    │
│    ● postgres                 │
└─────────────────────────────┘
```

#### Implementation:

```typescript
// Types for stack grouping
interface DockerStack {
  name: string;
  containers: Container[];
  isCompose: boolean; // true if Docker Compose stack
}

interface Container {
  id: string;
  name: string;
  hostId: string;
  stack?: string; // Docker Compose stack name (from labels)
  state: 'running' | 'exited' | 'paused';
}

// State
let selectedHost = $state<string | 'all'>('all');
let expandedStacks = $state<Set<string>>(new Set());
let hosts = $state<Host[]>([]);
let stacks = $state<DockerStack[]>([]);

// Derived: containers grouped by stack
let groupedContainers = $derived(() => {
  // Filter containers by selected host
  const filtered = selectedHost === 'all'
    ? containers
    : containers.filter(c => c.hostId === selectedHost);

  // Group by stack (using com.docker.compose.project label)
  const byStack = new Map<string, Container[]>();
  const standalone: Container[] = [];

  filtered.forEach(c => {
    if (c.stack) {
      if (!byStack.has(c.stack)) {
        byStack.set(c.stack, []);
      }
      byStack.get(c.stack)!.push(c);
    } else {
      standalone.push(c);
    }
  });

  // Convert to DockerStack array
  const result: DockerStack[] = [];

  // Add stacks
  Array.from(byStack.entries())
    .sort(([a], [b]) => a.localeCompare(b))
    .forEach(([name, containers]) => {
      result.push({
        name,
        containers: containers.sort((a, b) => a.name.localeCompare(b.name)),
        isCompose: true
      });
    });

  // Add standalone containers group
  if (standalone.length > 0) {
    result.push({
      name: 'Standalone Containers',
      containers: standalone.sort((a, b) => a.name.localeCompare(b.name)),
      isCompose: false
    });
  }

  return result;
});

// Toggle stack expansion
function toggleStack(stackName: string) {
  if (expandedStacks.has(stackName)) {
    expandedStacks.delete(stackName);
  } else {
    expandedStacks.add(stackName);
  }
  expandedStacks = new Set(expandedStacks); // Trigger reactivity
}
```

#### UI Component:

```svelte
<!-- Host selector breadcrumb -->
<div class="flex items-center gap-2 px-4 py-3 border-b border-border">
  <span class="text-sm text-foreground-muted">Hosts</span>
  <ChevronRight class="w-4 h-4 text-foreground-muted" />

  <select
    bind:value={selectedHost}
    class="text-sm font-medium bg-transparent border-none focus:ring-0"
  >
    <option value="all">All Hosts</option>
    {#each hosts as host}
      <option value={host.id}>{host.name}</option>
    {/each}
  </select>
</div>

<!-- Stacks and containers -->
<div class="flex-1 overflow-auto px-2 py-2 space-y-1">
  {#each groupedContainers as stack}
    <div class="stack-group">
      <!-- Stack header -->
      <button
        onclick={() => toggleStack(stack.name)}
        class="w-full flex items-center gap-2 px-3 py-2 rounded hover:bg-background-secondary transition-colors group"
      >
        <!-- Expand/collapse icon -->
        {#if expandedStacks.has(stack.name)}
          <ChevronDown class="w-4 h-4 text-foreground-muted flex-shrink-0" />
        {:else}
          <ChevronRight class="w-4 h-4 text-foreground-muted flex-shrink-0" />
        {/if}

        <!-- Stack icon -->
        {#if stack.isCompose}
          <Layers class="w-4 h-4 text-primary flex-shrink-0" />
        {:else}
          <Folder class="w-4 h-4 text-foreground-muted flex-shrink-0" />
        {/if}

        <!-- Stack name with count -->
        <span class="flex-1 text-left text-sm font-medium group-hover:text-primary transition-colors">
          {stack.name}
          <span class="text-foreground-muted ml-1">({stack.containers.length})</span>
        </span>

        <!-- Select all in stack button -->
        <button
          onclick={(e) => {
            e.stopPropagation();
            selectAllInStack(stack.name);
          }}
          class="opacity-0 group-hover:opacity-100 transition-opacity p-1 hover:bg-primary/20 rounded"
          title="Select all containers in this stack"
        >
          <CheckSquare class="w-3.5 h-3.5" />
        </button>
      </button>

      <!-- Containers (shown when expanded) -->
      {#if expandedStacks.has(stack.name)}
        <div class="ml-6 mt-1 space-y-1">
          {#each stack.containers as container}
            <label
              class="flex items-center gap-3 px-3 py-2 rounded hover:bg-background-secondary cursor-pointer group transition-all"
              class:bg-primary/10={selectedContainers.has(container.id)}
            >
              <input
                type="checkbox"
                checked={selectedContainers.has(container.id)}
                onchange={() => toggleContainer(container.id)}
                class="w-3.5 h-3.5 accent-primary focus:ring-2 focus:ring-primary focus:ring-offset-2"
              />

              <!-- Status indicator -->
              <div
                class="w-2 h-2 rounded-full flex-shrink-0"
                class:bg-running={container.state === 'running'}
                class:bg-stopped={container.state === 'exited'}
                class:bg-paused={container.state === 'paused'}
              ></div>

              <!-- Container name -->
              <span class="flex-1 text-sm group-hover:text-primary transition-colors">
                {container.name}
              </span>
            </label>
          {/each}
        </div>
      {/if}
    </div>
  {/each}
</div>
```

---

### 2. Advanced Log Controls (Dockhand-inspired)

**Control Bar Layout**:
```
[Single] [Multi] [Grouped] [Split]  |  [Pause] [Auto-scroll] [Wrap] [Pixels: 14] [Clear] [Export]
```

#### Implementation:

```typescript
// Log display preferences (Dockhand-style)
let isPaused = $state(false);
let autoScroll = $state(true);
let lineWrap = $state(true);
let fontSize = $state(14); // "pixels" in Dockhand
let maxLines = $state(1000); // Buffer limit
let showTimestamps = $state(true);
let timestampFormat = $state<'absolute' | 'relative'>('absolute');

// Advanced controls state
interface LogPreferences {
  fontSize: number;
  lineWrap: boolean;
  autoScroll: boolean;
  showTimestamps: boolean;
  timestampFormat: 'absolute' | 'relative' | 'none';
  theme: 'auto' | 'light' | 'dark';
  maxLines: number;
}

let preferences = $state<LogPreferences>({
  fontSize: 14,
  lineWrap: true,
  autoScroll: true,
  showTimestamps: true,
  timestampFormat: 'absolute',
  theme: 'auto',
  maxLines: 1000
});
```

#### UI Component:

```svelte
<!-- Enhanced control bar -->
<div class="flex items-center justify-between px-4 py-2 border-b border-border bg-background-secondary">
  <!-- Left: Viewing modes -->
  <div class="flex items-center gap-1" role="radiogroup" aria-label="Log viewing mode">
    <button
      role="radio"
      aria-checked={mode === 'single'}
      onclick={() => mode = 'single'}
      class="btn-sm {mode === 'single' ? 'btn-primary' : 'btn-ghost'}"
      title="Single container (Ctrl+1)"
    >
      <Square class="w-4 h-4 mr-1" />
      Single
    </button>

    <button
      role="radio"
      aria-checked={mode === 'multi'}
      onclick={() => mode = 'multi'}
      class="btn-sm {mode === 'multi' ? 'btn-primary' : 'btn-ghost'}"
      title="Multiple containers (Ctrl+2)"
    >
      <Layers class="w-4 h-4 mr-1" />
      Multi
    </button>

    <button
      role="radio"
      aria-checked={mode === 'grouped'}
      onclick={() => mode = 'grouped'}
      class="btn-sm {mode === 'grouped' ? 'btn-primary' : 'btn-ghost'}"
      title="Grouped by host (Ctrl+3)"
    >
      <Grid class="w-4 h-4 mr-1" />
      Grouped
    </button>

    <button
      role="radio"
      aria-checked={mode === 'split'}
      onclick={() => handleSplitMode()}
      disabled={selectedContainers.size < 2 && mode !== 'split'}
      class="btn-sm {mode === 'split' ? 'btn-primary' : 'btn-ghost'}"
      title="Split view (Ctrl+4) - requires 2 containers"
    >
      <Columns class="w-4 h-4 mr-1" />
      Split
    </button>
  </div>

  <!-- Center: Log controls (Dockhand-inspired) -->
  <div class="flex items-center gap-2 border-l border-border pl-4 ml-4">
    <!-- Pause/Play -->
    <button
      onclick={() => isPaused = !isPaused}
      class="btn-icon {isPaused ? 'text-accent-red' : 'text-running'}"
      title={isPaused ? 'Resume streaming (Ctrl+P)' : 'Pause streaming (Ctrl+P)'}
    >
      {#if isPaused}
        <Play class="w-4 h-4" />
      {:else}
        <Pause class="w-4 h-4" />
      {/if}
    </button>

    <!-- Auto-scroll -->
    <button
      onclick={() => autoScroll = !autoScroll}
      class="btn-icon {autoScroll ? 'text-primary' : ''}"
      title="Toggle auto-scroll"
    >
      <ArrowDown class="w-4 h-4" />
    </button>

    <!-- Line wrap -->
    <button
      onclick={() => lineWrap = !lineWrap}
      class="btn-icon {lineWrap ? 'text-primary' : ''}"
      title="Toggle line wrap (Ctrl+W)"
    >
      <WrapText class="w-4 h-4" />
    </button>

    <!-- Font size (pixels) -->
    <div class="flex items-center gap-1 border border-border rounded-md px-2 py-1">
      <button
        onclick={() => fontSize = Math.max(10, fontSize - 1)}
        class="btn-icon-sm"
        title="Decrease font size"
      >
        <Minus class="w-3 h-3" />
      </button>

      <span class="text-xs font-mono w-8 text-center">
        {fontSize}px
      </span>

      <button
        onclick={() => fontSize = Math.min(24, fontSize + 1)}
        class="btn-icon-sm"
        title="Increase font size"
      >
        <Plus class="w-3 h-3" />
      </button>
    </div>

    <!-- Timestamp format -->
    <select
      bind:value={timestampFormat}
      class="text-xs border border-border rounded px-2 py-1 bg-background"
      title="Timestamp format"
    >
      <option value="absolute">HH:MM:SS</option>
      <option value="relative">Relative</option>
      <option value="none">Hide</option>
    </select>
  </div>

  <!-- Right: Actions -->
  <div class="flex items-center gap-2 border-l border-border pl-4 ml-4">
    <!-- Clear logs -->
    <button
      onclick={() => clearLogs()}
      class="btn-sm btn-ghost"
      title="Clear all logs (Ctrl+K)"
    >
      <Trash class="w-4 h-4 mr-1" />
      Clear
    </button>

    <!-- Export -->
    <button
      onclick={() => showExportMenu = !showExportMenu}
      class="btn-sm btn-ghost"
      title="Export logs (Ctrl+E)"
    >
      <Download class="w-4 h-4 mr-1" />
      Export
    </button>

    <!-- Settings -->
    <button
      onclick={() => showSettingsMenu = !showSettingsMenu}
      class="btn-icon"
      title="Display settings"
    >
      <Settings class="w-4 h-4" />
    </button>
  </div>
</div>
```

---

### 3. Combined Search Features

**Search Bar with Fuzzy + Regex**:

```svelte
<div class="flex items-center gap-4 px-4 py-3 border-b border-border">
  <!-- Container search (fuzzy) -->
  <div class="flex-1 flex items-center gap-2 border border-border rounded-lg px-3 py-2">
    <Search class="w-4 h-4 text-foreground-muted" />
    <input
      bind:this={containerSearchInput}
      bind:value={containerSearch}
      placeholder="Search containers (fuzzy: 'ngx' finds 'nginx')..."
      class="flex-1 bg-transparent text-sm focus:outline-none"
    />
    {#if containerSearch}
      <button
        onclick={() => containerSearch = ''}
        class="btn-icon-sm"
      >
        <X class="w-3 h-3" />
      </button>
    {/if}
  </div>

  <!-- Log search (with regex toggle) -->
  <div class="flex-1 flex items-center gap-2 border border-border rounded-lg px-3 py-2 {regexError ? 'border-accent-red' : ''}">
    <Search class="w-4 h-4 text-foreground-muted" />
    <input
      bind:this={logSearchInput}
      bind:value={logSearch}
      placeholder={regexEnabled ? "Regex: ERROR|WARN..." : "Search logs..."}
      class="flex-1 bg-transparent text-sm focus:outline-none"
    />

    <button
      onclick={() => regexEnabled = !regexEnabled}
      class="btn-icon-sm {regexEnabled ? 'text-primary' : ''}"
      title="Toggle regex mode"
    >
      <Code class="w-4 h-4" />
    </button>

    {#if logSearch}
      <button
        onclick={() => logSearch = ''}
        class="btn-icon-sm"
      >
        <X class="w-3 h-3" />
      </button>
    {/if}
  </div>
</div>

{#if regexError}
  <div class="px-4 py-2 bg-accent-red/10 border-b border-accent-red text-xs text-accent-red">
    <AlertCircle class="w-3 h-3 inline mr-1" />
    Invalid regex: {regexError}
  </div>
{/if}
```

---

### 4. Backend API Changes (Minimal)

#### New Endpoint: List Docker Compose Stacks

```go
// GET /api/stacks?hostId=raspi1
func getStacks(c *fiber.Ctx) error {
    hostId := c.Query("hostId")
    if hostId == "" {
        return c.Status(400).JSON(fiber.Map{"error": "hostId required"})
    }

    dockerClient := getDockerClient(hostId)
    if dockerClient == nil {
        return c.Status(404).JSON(fiber.Map{"error": "Host not found"})
    }

    // List all containers
    containers, err := dockerClient.ContainerList(context.Background(), container.ListOptions{All: true})
    if err != nil {
        return c.Status(500).JSON(fiber.Map{"error": err.Error()})
    }

    // Group by com.docker.compose.project label
    stacks := make(map[string][]string)
    standalone := []string{}

    for _, ctr := range containers {
        project := ctr.Labels["com.docker.compose.project"]
        if project != "" {
            if _, exists := stacks[project]; !exists {
                stacks[project] = []string{}
            }
            stacks[project] = append(stacks[project], ctr.ID)
        } else {
            standalone = append(standalone, ctr.ID)
        }
    }

    return c.JSON(fiber.Map{
        "stacks": stacks,
        "standalone": standalone,
    })
}
```

#### Enhanced: Include Stack Metadata in Container List

```go
// Enhance existing /api/containers endpoint
type ContainerResponse struct {
    ID       string            `json:"id"`
    Name     string            `json:"name"`
    State    string            `json:"state"`
    HostId   string            `json:"hostId"`
    Stack    string            `json:"stack"` // NEW: from com.docker.compose.project
    Service  string            `json:"service"` // NEW: from com.docker.compose.service
    Labels   map[string]string `json:"labels"`
}

// In getContainers function, extract stack info:
stack := ctr.Labels["com.docker.compose.project"]
service := ctr.Labels["com.docker.compose.service"]
```

---

## Updated Implementation Priority

### Phase 1: Critical Fixes + Navigation (Week 1)

1. **Fix layout stability bug** (same as before)
2. **Add host filtering** (Dozzle-inspired breadcrumb)
3. **Add stack grouping** (Dozzle-inspired expand/collapse)
4. **Backend: /api/stacks endpoint**
5. **Backend: enhance /api/containers with stack metadata**

### Phase 2: Search + Controls (Week 2)

6. **Fuzzy container search** (enhanced for stack-aware search)
7. **Regex log filtering** (same as before)
8. **Advanced log controls** (Dockhand-inspired: pause, pixels, wrap, etc.)
9. **Timestamp enhancements** (absolute/relative/none)

### Phase 3: Advanced Features (Week 3)

10. **Split-screen mode** (side-by-side comparison)
11. **Keyboard shortcuts** (11+ shortcuts)
12. **Enhanced export** (with metadata, per-container/per-stack)
13. **Visual enhancements** (color bars, badges, animations)

### Phase 4: Polish + Deploy (Week 4)

14. **Accessibility improvements** (ARIA, keyboard nav, contrast)
15. **Manual testing** (all features, all modes)
16. **Documentation** (user guide, developer docs)
17. **Deploy to raspi** (production testing)
18. **Merge to master**

---

## Bundle Size Estimate (Updated)

| Feature | Estimated Size |
|---------|---------------|
| Fuzzy search | ~1 KB |
| Regex filter | 0 KB (built-in) |
| Stack grouping UI | ~3 KB |
| Host filtering | ~2 KB |
| Advanced controls | ~4 KB |
| Split-screen | ~5 KB |
| Keyboard shortcuts | ~2 KB |
| Timestamp formats | ~1 KB |
| Visual enhancements | ~3 KB |
| **Total Added** | **~21 KB** |

Still well under 25KB target ✅

---

## Design Principles

### From Dozzle:
- Clean, minimalist UI
- Excellent navigation (host → stack → container hierarchy)
- Visual grouping that matches Docker Compose mental model
- Efficient use of space

### From Dockhand:
- Advanced but intuitive controls
- Professional polish in log display
- Flexible display preferences
- Power-user features accessible but not overwhelming

### Our Custom Additions:
- Fuzzy search (neither Dozzle nor Dockhand have this)
- Regex filtering (Dozzle has it, we enhance it)
- Split-screen mode (unique feature)
- Comprehensive keyboard shortcuts
- Multi-host awareness (unique to DockerVerse)

---

## Success Metrics

### Quantitative

- Bundle size increase: <25 KB ✅
- Layout stability: 0 resize bugs ✅
- Search accuracy: >90% fuzzy matches ✅
- Keyboard navigation: 100% accessible ✅
- Navigation depth: Host → Stack → Container (3 levels) ✅
- Control response time: <100ms for all toggles ✅

### Qualitative

- User feedback: "Best of Dozzle + Dockhand combined"
- Developer feedback: "Easy to navigate with many containers"
- Comparison: "Exceeds both Dozzle and Dockhand individually"
- Power users: "Keyboard shortcuts + advanced controls are amazing"

---

## Next Steps

1. ✅ **Design document approved** (this document)
2. 📝 **Update implementation plan** (break down into tasks)
3. 🛠️ **Implement Phase 1** (navigation + host filtering)
4. 🧪 **Test thoroughly** (all features, all modes)
5. 🚀 **Deploy to production**
6. 📚 **Document everything**
7. 🔀 **Merge to master**

---

**Approved by**: Victor Heredia
**Ready for Implementation**: Yes
**Target Version**: v2.5.0 (Combined Edition)

---

## Sources

- [GitHub - Finsys/dockhand](https://github.com/Finsys/dockhand)
- [Dockhand - Modern Docker Management](https://dockhand.pro/)
- [GitHub - amir20/dozzle](https://github.com/amir20/dozzle)
- [Dozzle Official Site](https://dozzle.dev/)
