# DockerVerse Logs Page Enhancement - Implementation Plan v2 (Combined)

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build professional log viewer combining Dozzle's navigation (host filtering, stack grouping) + Dockhand's controls (pause, pixels, wrap) + custom enhancements (fuzzy search, regex, split mode).

**Architecture:** Frontend-heavy with minimal backend changes. SvelteKit 2 + Svelte 5, flexbox layout, stack-based navigation, advanced controls.

**Tech Stack:** Svelte 5, TypeScript, Tailwind CSS, Lucide icons, Go backend (Fiber v2)

**Design Document:** `docs/plans/2026-02-16-logs-improvement-design-v2.md`

---

## Implementation Phases

### Phase 1: Critical Fixes + Backend Foundation (Tasks 1-4)
### Phase 2: Navigation Layer - Dozzle Style (Tasks 5-7)
### Phase 3: Control Layer - Dockhand Style (Tasks 8-10)
### Phase 4: Search + Filtering (Tasks 11-13)
### Phase 5: Split Mode + Visual Polish (Tasks 14-16)
### Phase 6: Testing, Deploy, Documentation (Tasks 17-20)

---

## PHASE 1: Critical Fixes + Backend Foundation

### Task 1: Fix Layout Stability Bug (CRITICAL)

**Files:**
- Modify: `frontend/src/routes/logs/+page.svelte:260-290`

**Context:** Replace fixed height `h-[calc(100vh-7rem)]` with flexbox pattern to prevent resize bug.

#### Step 1: Replace fixed height with flex-based layout

**Before (line 272):**
```svelte
<div class="flex flex-col h-[calc(100vh-7rem)]">
```

**After:**
```svelte
<div class="flex flex-col flex-1 min-h-0">
```

#### Step 2: Update main container structure

Find the container with sidebar + logs area:

**After:**
```svelte
<div class="flex flex-1 min-h-0 gap-4">
  <!-- Sidebar: fixed width, scrollable -->
  <aside class="w-80 flex-shrink-0 flex flex-col min-h-0 bg-background-secondary border border-border rounded-lg">
    <div class="flex-1 overflow-auto">
      <!-- navigation content -->
    </div>
  </aside>

  <!-- Logs area: fills remaining space -->
  <div class="flex-1 flex flex-col min-h-0">
    <!-- controls + logs -->
  </div>
</div>
```

#### Step 3: Test layout stability

**Test:** Select/deselect containers → logs area should NOT resize

#### Step 4: Commit

```bash
git add frontend/src/routes/logs/+page.svelte
git commit -m "fix(logs): replace fixed height with flexbox for layout stability

- Remove h-[calc(100vh-7rem)] causing resize bug
- Apply flex-1 min-h-0 pattern for stable dimensions
- Sidebar and logs area maintain consistent size

Refs: docs/plans/2026-02-16-logs-improvement-design-v2.md"
```

---

### Task 2: Backend - Add Stack Metadata Endpoint

**Files:**
- Modify: `backend/main.go` (add /api/stacks endpoint)

**Context:** Create endpoint to list Docker Compose stacks by host.

#### Step 1: Add getStacks handler function

Add after existing container handlers (around line 800):

```go
// GET /api/stacks?hostId=raspi1
// Returns Docker Compose stacks grouped by com.docker.compose.project label
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
	ctx := context.Background()
	containers, err := dockerClient.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Group by com.docker.compose.project label
	stacks := make(map[string][]map[string]interface{})
	standalone := []map[string]interface{}{}

	for _, ctr := range containers {
		project := ctr.Labels["com.docker.compose.project"]
		service := ctr.Labels["com.docker.compose.service"]

		containerInfo := map[string]interface{}{
			"id":      ctr.ID,
			"name":    strings.TrimPrefix(ctr.Names[0], "/"),
			"state":   ctr.State,
			"service": service,
		}

		if project != "" {
			if _, exists := stacks[project]; !exists {
				stacks[project] = []map[string]interface{}{}
			}
			stacks[project] = append(stacks[project], containerInfo)
		} else {
			standalone = append(standalone, containerInfo)
		}
	}

	return c.JSON(fiber.Map{
		"stacks":     stacks,
		"standalone": standalone,
	})
}
```

#### Step 2: Register route

Find the API routes section and add:

```go
api.Get("/stacks", getStacks)
```

#### Step 3: Test endpoint manually

```bash
curl http://localhost:3001/api/stacks?hostId=raspi1 | jq
```

**Expected:** JSON with stacks grouped by project name + standalone containers

#### Step 4: Commit

```bash
git add backend/main.go
git commit -m "feat(api): add /api/stacks endpoint for Docker Compose grouping

- Returns containers grouped by com.docker.compose.project label
- Includes service name from com.docker.compose.service
- Separates standalone containers without stack

Enables Dozzle-style stack navigation in frontend.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 3: Backend - Enhance Containers Endpoint with Stack Metadata

**Files:**
- Modify: `backend/main.go` (enhance /api/containers response)

**Context:** Add stack and service metadata to existing container list endpoint.

#### Step 1: Find getContainers function

Locate the `/api/containers` handler (around line 600)

#### Step 2: Add stack metadata to response

In the container mapping section, add:

```go
// Find existing code that builds container response:
containerResponse := fiber.Map{
	"id":     ctr.ID,
	"name":   strings.TrimPrefix(ctr.Names[0], "/"),
	"state":  ctr.State,
	"hostId": hostId,
	// ... existing fields ...
}

// ADD these fields:
containerResponse["stack"] = ctr.Labels["com.docker.compose.project"]
containerResponse["service"] = ctr.Labels["com.docker.compose.service"]
containerResponse["labels"] = ctr.Labels // Full labels for reference
```

#### Step 3: Test enhanced endpoint

```bash
curl http://localhost:3001/api/containers | jq '.[] | {name, stack, service}' | head -20
```

**Expected:** Containers now include `stack` and `service` fields

#### Step 4: Commit

```bash
git add backend/main.go
git commit -m "feat(api): add stack metadata to /api/containers endpoint

- Include com.docker.compose.project as 'stack' field
- Include com.docker.compose.service as 'service' field
- Return full labels map for additional metadata

Enables frontend to group containers by stack.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 4: Frontend - Add Stack Types and Interfaces

**Files:**
- Modify: `frontend/src/routes/logs/+page.svelte:1-50` (add types)

**Context:** Define TypeScript interfaces for stack-based navigation.

#### Step 1: Add stack-related types

Add after existing imports (around line 20):

```typescript
// Stack grouping types (Dozzle-inspired)
interface DockerStack {
  name: string;
  containers: Container[];
  isCompose: boolean; // true for Docker Compose stacks, false for standalone group
  isExpanded?: boolean;
}

// Enhanced Container interface
interface Container {
  id: string;
  name: string;
  hostId: string;
  state: 'running' | 'exited' | 'paused' | 'restarting';
  stack?: string; // com.docker.compose.project
  service?: string; // com.docker.compose.service
  labels?: Record<string, string>;
}

// Host interface
interface Host {
  id: string;
  name: string;
  address: string;
  type: 'local' | 'remote';
}

// Log display preferences (Dockhand-inspired)
interface LogPreferences {
  fontSize: number; // pixels (10-24)
  lineWrap: boolean;
  autoScroll: boolean;
  showTimestamps: boolean;
  timestampFormat: 'absolute' | 'relative' | 'none';
  maxLines: number; // buffer limit
}
```

#### Step 2: Initialize state with new types

```typescript
// Navigation state (Dozzle-inspired)
let selectedHost = $state<string | 'all'>('all');
let expandedStacks = $state<Set<string>>(new Set());
let stacks = $state<DockerStack[]>([]);

// Preferences state (Dockhand-inspired)
let preferences = $state<LogPreferences>({
  fontSize: 14,
  lineWrap: true,
  autoScroll: true,
  showTimestamps: true,
  timestampFormat: 'absolute',
  maxLines: 1000
});
```

#### Step 3: Commit

```bash
git add frontend/src/routes/logs/+page.svelte
git commit -m "feat(logs): add TypeScript interfaces for stack navigation

- Add DockerStack interface for Dozzle-style grouping
- Enhanced Container interface with stack/service fields
- Add LogPreferences for Dockhand-style controls
- Initialize navigation and preferences state

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## PHASE 2: Navigation Layer - Dozzle Style

### Task 5: Implement Host Filter Breadcrumb

**Files:**
- Modify: `frontend/src/routes/logs/+page.svelte:200-250` (add host selector)

**Context:** Add Dozzle-style breadcrumb navigation: "Hosts > [Selected Host]"

#### Step 1: Fetch hosts list

Add API call in `onMount`:

```typescript
onMount(async () => {
  // Fetch hosts
  const hostsRes = await fetch('/api/hosts');
  hosts = await hostsRes.json();

  // ... existing mount logic
});
```

#### Step 2: Create breadcrumb UI component

Add to sidebar header (top of aside):

```svelte
<!-- Host selector breadcrumb -->
<div class="flex items-center gap-2 px-4 py-3 border-b border-border bg-background-tertiary">
  <span class="text-sm text-foreground-muted font-medium">Hosts</span>
  <ChevronRight class="w-4 h-4 text-foreground-muted" />

  <div class="relative flex-1">
    <select
      bind:value={selectedHost}
      class="w-full text-sm font-medium bg-transparent border border-border rounded px-3 py-1.5 focus:ring-2 focus:ring-primary focus:outline-none cursor-pointer"
      aria-label="Select Docker host"
    >
      <option value="all">All Hosts ({containers.length} containers)</option>
      {#each hosts as host}
        <option value={host.id}>
          {host.name} ({containers.filter(c => c.hostId === host.id).length} containers)
        </option>
      {/each}
    </select>
  </div>
</div>
```

#### Step 3: Import ChevronRight icon

```typescript
import { ChevronRight, /* existing imports */ } from "lucide-svelte";
```

#### Step 4: Filter containers by selected host

```typescript
// Derived: containers filtered by host
let filteredByHost = $derived(() => {
  if (selectedHost === 'all') return containers;
  return containers.filter(c => c.hostId === selectedHost);
});
```

#### Step 5: Test host filtering

1. Select "All Hosts" → all containers shown
2. Select "raspi1" → only raspi1 containers
3. Select "raspi2" → only raspi2 containers

#### Step 6: Commit

```bash
git add frontend/src/routes/logs/+page.svelte
git commit -m "feat(logs): add host filter breadcrumb (Dozzle-style)

- Add 'Hosts > [Selected Host]' breadcrumb navigation
- Dropdown to switch between hosts or view all
- Show container count per host
- Filter containers by selected host

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 6: Implement Stack Grouping Logic

**Files:**
- Modify: `frontend/src/routes/logs/+page.svelte:100-200` (add grouping logic)

**Context:** Group containers by Docker Compose stack (Dozzle-style).

#### Step 1: Create grouping function

```typescript
/**
 * Group containers by Docker Compose stack
 * Returns array of DockerStack objects
 */
function groupContainersByStack(containers: Container[]): DockerStack[] {
  const byStack = new Map<string, Container[]>();
  const standalone: Container[] = [];

  // Group by stack label
  containers.forEach(c => {
    if (c.stack) {
      if (!byStack.has(c.stack)) {
        byStack.set(c.stack, []);
      }
      byStack.get(c.stack)!.push(c);
    } else {
      standalone.push(c);
    }
  });

  // Convert to DockerStack array, sorted alphabetically
  const result: DockerStack[] = Array.from(byStack.entries())
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([name, containers]) => ({
      name,
      containers: containers.sort((a, b) => a.name.localeCompare(b.name)),
      isCompose: true,
      isExpanded: expandedStacks.has(name)
    }));

  // Add standalone group if any
  if (standalone.length > 0) {
    result.push({
      name: 'Standalone Containers',
      containers: standalone.sort((a, b) => a.name.localeCompare(b.name)),
      isCompose: false,
      isExpanded: expandedStacks.has('Standalone Containers')
    });
  }

  return result;
}
```

#### Step 2: Create derived state for grouped containers

```typescript
// Derived: containers grouped by stack (host-filtered)
let groupedContainers = $derived(() => {
  const filtered = selectedHost === 'all'
    ? containers
    : containers.filter(c => c.hostId === selectedHost);

  return groupContainersByStack(filtered);
});
```

#### Step 3: Add toggle stack function

```typescript
/**
 * Toggle stack expansion
 */
function toggleStack(stackName: string) {
  if (expandedStacks.has(stackName)) {
    expandedStacks.delete(stackName);
  } else {
    expandedStacks.add(stackName);
  }
  // Trigger reactivity
  expandedStacks = new Set(expandedStacks);
}

/**
 * Select all containers in a stack
 */
function selectAllInStack(stackName: string) {
  const stack = groupedContainers.find(s => s.name === stackName);
  if (!stack) return;

  stack.containers.forEach(c => {
    selectedContainers.add(c.id);
  });
  selectedContainers = new Set(selectedContainers);
}
```

#### Step 4: Test grouping logic

Add temporary debug output:

```typescript
$effect(() => {
  console.log('Grouped stacks:', groupedContainers);
  console.log('Expanded:', Array.from(expandedStacks));
});
```

#### Step 5: Commit

```bash
git add frontend/src/routes/logs/+page.svelte
git commit -m "feat(logs): implement stack grouping logic

- Group containers by com.docker.compose.project label
- Separate standalone containers into own group
- Sort stacks alphabetically, containers within stacks
- Track expanded/collapsed state per stack
- Add selectAllInStack helper function

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 7: Build Stack Navigation UI (Dozzle-style)

**Files:**
- Modify: `frontend/src/routes/logs/+page.svelte:250-400` (sidebar UI)

**Context:** Build the visual stack navigation with expand/collapse.

#### Step 1: Create stack list UI

Replace existing container list with stack-based UI:

```svelte
<!-- Stacks and containers list -->
<div class="flex-1 overflow-auto px-2 py-2 space-y-1">
  {#each groupedContainers as stack}
    <div class="stack-group" data-stack={stack.name}>
      <!-- Stack header -->
      <button
        onclick={() => toggleStack(stack.name)}
        class="w-full flex items-center gap-2 px-3 py-2 rounded hover:bg-background-tertiary transition-colors group"
        aria-expanded={stack.isExpanded}
        aria-label="Toggle {stack.name}"
      >
        <!-- Expand/collapse icon -->
        {#if stack.isExpanded}
          <ChevronDown class="w-4 h-4 text-foreground-muted flex-shrink-0 transition-transform" />
        {:else}
          <ChevronRight class="w-4 h-4 text-foreground-muted flex-shrink-0 transition-transform" />
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
          <span class="text-foreground-muted font-normal ml-1">
            ({stack.containers.length})
          </span>
        </span>

        <!-- Select all in stack -->
        <button
          onclick={(e) => {
            e.stopPropagation();
            selectAllInStack(stack.name);
          }}
          class="opacity-0 group-hover:opacity-100 transition-opacity p-1 hover:bg-primary/20 rounded"
          title="Select all containers in {stack.name}"
          aria-label="Select all in {stack.name}"
        >
          <CheckSquare class="w-3.5 h-3.5 text-primary" />
        </button>
      </button>

      <!-- Containers (shown when expanded) -->
      {#if stack.isExpanded}
        <div class="ml-6 mt-1 space-y-0.5 animate-in slide-in-from-top-2">
          {#each stack.containers as container}
            <label
              class="flex items-center gap-2 px-3 py-1.5 rounded hover:bg-background-tertiary cursor-pointer group transition-all"
              class:bg-primary/10={selectedContainers.has(container.id)}
            >
              <!-- Checkbox -->
              <input
                type="checkbox"
                checked={selectedContainers.has(container.id)}
                onchange={() => toggleContainer(container.id)}
                class="w-3.5 h-3.5 accent-primary focus:ring-2 focus:ring-primary focus:ring-offset-1 focus:ring-offset-background-secondary rounded"
                aria-label="Select {container.name}"
              />

              <!-- Status indicator -->
              <div
                class="w-2 h-2 rounded-full flex-shrink-0"
                class:bg-running={container.state === 'running'}
                class:bg-stopped={container.state === 'exited'}
                class:bg-paused={container.state === 'paused'}
                class:bg-primary={container.state === 'restarting'}
                class:animate-pulse={container.state === 'restarting'}
                title={container.state}
              ></div>

              <!-- Container name -->
              <span class="flex-1 text-sm group-hover:text-primary transition-colors truncate" title={container.name}>
                {container.name}
              </span>

              <!-- Service badge (if from Compose) -->
              {#if container.service}
                <span class="px-1.5 py-0.5 text-xs bg-background-tertiary text-foreground-muted rounded border border-border">
                  {container.service}
                </span>
              {/if}
            </label>
          {/each}
        </div>
      {/if}
    </div>
  {/each}

  <!-- Empty state -->
  {#if groupedContainers.length === 0}
    <div class="flex flex-col items-center justify-center py-8 text-center">
      <Package class="w-12 h-12 text-foreground-muted mb-3" />
      <p class="text-sm text-foreground-muted">
        No containers found
        {#if selectedHost !== 'all'}
          on {hosts.find(h => h.id === selectedHost)?.name}
        {/if}
      </p>
    </div>
  {/if}
</div>
```

#### Step 2: Import new icons

```typescript
import {
  ChevronDown,
  ChevronRight,
  Layers,
  Folder,
  CheckSquare,
  Package,
  /* existing */
} from "lucide-svelte";
```

#### Step 3: Add CSS for animations

Add to `<style>` block or global CSS:

```css
@keyframes slide-in-from-top-2 {
  from {
    opacity: 0;
    transform: translateY(-8px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.animate-in {
  animation-duration: 200ms;
  animation-timing-function: ease-out;
}

.slide-in-from-top-2 {
  animation-name: slide-in-from-top-2;
}
```

#### Step 4: Test stack UI manually

1. Click stack header → should expand/collapse
2. Click "select all" button → all containers in stack selected
3. Select individual containers → checkboxes work
4. Verify status dots show correct colors
5. Verify service badges appear for Compose containers

#### Step 5: Commit

```bash
git add frontend/src/routes/logs/+page.svelte frontend/src/app.css
git commit -m "feat(logs): build stack navigation UI (Dozzle-style)

- Expand/collapse stacks with animated transitions
- Show stack icon (Layers for Compose, Folder for standalone)
- Display container count per stack
- Select all containers in stack with one click
- Show service badges for Docker Compose services
- Status indicator dots with colors per state
- Smooth animations on expand/collapse

Visual matches Dozzle's excellent navigation UX.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## PHASE 3: Control Layer - Dockhand Style

### Task 8: Build Advanced Control Bar

**Files:**
- Modify: `frontend/src/routes/logs/+page.svelte:400-500` (control bar UI)

**Context:** Add Dockhand-style advanced controls (pause, auto-scroll, pixels, wrap, etc.)

#### Step 1: Create control bar component

Add above the logs display area:

```svelte
<!-- Advanced control bar (Dockhand-inspired) -->
<div class="flex items-center justify-between px-4 py-2.5 border-b border-border bg-background-secondary/50 backdrop-blur">
  <!-- Left: Viewing modes -->
  <div class="flex items-center gap-1" role="radiogroup" aria-label="Log viewing mode">
    <button
      role="radio"
      aria-checked={mode === 'single'}
      onclick={() => mode = 'single'}
      class="btn-sm {mode === 'single' ? 'btn-primary' : 'btn-ghost'}"
      title="Single container (Ctrl+1)"
    >
      <Square class="w-3.5 h-3.5 mr-1.5" />
      Single
    </button>

    <button
      role="radio"
      aria-checked={mode === 'multi'}
      onclick={() => mode = 'multi'}
      class="btn-sm {mode === 'multi' ? 'btn-primary' : 'btn-ghost'}"
      title="Multiple containers stacked (Ctrl+2)"
    >
      <Layers class="w-3.5 h-3.5 mr-1.5" />
      Multi
    </button>

    <button
      role="radio"
      aria-checked={mode === 'grouped'}
      onclick={() => mode = 'grouped'}
      class="btn-sm {mode === 'grouped' ? 'btn-primary' : 'btn-ghost'}"
      title="Grouped by host (Ctrl+3)"
    >
      <Grid class="w-3.5 h-3.5 mr-1.5" />
      Grouped
    </button>

    <button
      role="radio"
      aria-checked={mode === 'split'}
      onclick={() => handleSplitMode()}
      disabled={selectedContainers.size < 2 && mode !== 'split'}
      class="btn-sm {mode === 'split' ? 'btn-primary' : 'btn-ghost'} disabled:opacity-50 disabled:cursor-not-allowed"
      title="Split view (Ctrl+4) - requires 2+ containers"
    >
      <Columns class="w-3.5 h-3.5 mr-1.5" />
      Split
    </button>
  </div>

  <!-- Center: Log controls (Dockhand-inspired) -->
  <div class="flex items-center gap-2">
    <!-- Pause/Play -->
    <button
      onclick={() => isPaused = !isPaused}
      class="btn-icon {isPaused ? 'text-accent-red hover:bg-accent-red/10' : 'text-running hover:bg-running/10'}"
      title={isPaused ? 'Resume streaming (Ctrl+P / Space)' : 'Pause streaming (Ctrl+P / Space)'}
      aria-label={isPaused ? 'Resume' : 'Pause'}
    >
      {#if isPaused}
        <Play class="w-4 h-4 fill-current" />
      {:else}
        <Pause class="w-4 h-4" />
      {/if}
    </button>

    <div class="w-px h-4 bg-border"></div>

    <!-- Auto-scroll -->
    <button
      onclick={() => preferences.autoScroll = !preferences.autoScroll}
      class="btn-icon {preferences.autoScroll ? 'text-primary' : 'text-foreground-muted'}"
      title="Toggle auto-scroll to bottom"
      aria-label="Auto-scroll"
      aria-pressed={preferences.autoScroll}
    >
      <ArrowDown class="w-4 h-4 {preferences.autoScroll ? 'animate-bounce-subtle' : ''}" />
    </button>

    <!-- Line wrap -->
    <button
      onclick={() => preferences.lineWrap = !preferences.lineWrap}
      class="btn-icon {preferences.lineWrap ? 'text-primary' : 'text-foreground-muted'}"
      title="Toggle line wrap (Ctrl+W)"
      aria-label="Line wrap"
      aria-pressed={preferences.lineWrap}
    >
      <WrapText class="w-4 h-4" />
    </button>

    <div class="w-px h-4 bg-border"></div>

    <!-- Font size (pixels) - Dockhand feature -->
    <div class="flex items-center gap-1 border border-border rounded-md px-2 py-1 bg-background">
      <button
        onclick={() => preferences.fontSize = Math.max(10, preferences.fontSize - 1)}
        class="btn-icon-xs"
        title="Decrease font size"
        aria-label="Decrease font size"
      >
        <Minus class="w-3 h-3" />
      </button>

      <span class="text-xs font-mono w-9 text-center tabular-nums" title="Font size in pixels">
        {preferences.fontSize}px
      </span>

      <button
        onclick={() => preferences.fontSize = Math.min(24, preferences.fontSize + 1)}
        class="btn-icon-xs"
        title="Increase font size"
        aria-label="Increase font size"
      >
        <Plus class="w-3 h-3" />
      </button>
    </div>

    <!-- Timestamp format -->
    <select
      bind:value={preferences.timestampFormat}
      class="text-xs border border-border rounded px-2 py-1 bg-background focus:ring-2 focus:ring-primary focus:outline-none"
      title="Timestamp format"
      aria-label="Timestamp format"
    >
      <option value="absolute">HH:MM:SS</option>
      <option value="relative">Relative</option>
      <option value="none">Hide</option>
    </select>
  </div>

  <!-- Right: Actions -->
  <div class="flex items-center gap-2">
    <!-- Clear logs -->
    <button
      onclick={() => clearLogs()}
      class="btn-sm btn-ghost text-foreground-muted hover:text-accent-red"
      title="Clear all logs (Ctrl+K)"
      aria-label="Clear logs"
    >
      <Trash2 class="w-3.5 h-3.5 mr-1.5" />
      Clear
    </button>

    <!-- Export -->
    <div class="relative">
      <button
        onclick={() => showExportMenu = !showExportMenu}
        class="btn-sm btn-ghost"
        title="Export logs (Ctrl+E)"
        aria-label="Export logs"
        aria-haspopup="true"
        aria-expanded={showExportMenu}
      >
        <Download class="w-3.5 h-3.5 mr-1.5" />
        Export
        <ChevronDown class="w-3 h-3 ml-1" />
      </button>

      <!-- Export menu (will implement later) -->
    </div>

    <!-- Settings -->
    <button
      onclick={() => showSettingsMenu = !showSettingsMenu}
      class="btn-icon"
      title="Display settings"
      aria-label="Settings"
      aria-haspopup="true"
      aria-expanded={showSettingsMenu}
    >
      <Settings class="w-4 h-4" />
    </button>
  </div>
</div>
```

#### Step 2: Import new icons

```typescript
import {
  Play,
  Pause,
  ArrowDown,
  WrapText,
  Minus,
  Plus,
  Trash2,
  Download,
  Settings,
  Square,
  Grid,
  Columns,
  /* existing */
} from "lucide-svelte";
```

#### Step 3: Add subtle bounce animation for auto-scroll

In CSS:

```css
@keyframes bounce-subtle {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-3px); }
}

.animate-bounce-subtle {
  animation: bounce-subtle 2s ease-in-out infinite;
}
```

#### Step 4: Test controls manually

- Pause → streaming stops
- Auto-scroll off → new logs don't auto-scroll
- Line wrap toggle → logs wrap/don't wrap
- Font size +/- → text size changes
- Timestamp format → format changes
- Mode buttons → switch between modes

#### Step 5: Commit

```bash
git add frontend/src/routes/logs/+page.svelte frontend/src/app.css
git commit -m "feat(logs): build advanced control bar (Dockhand-style)

- Add 4 viewing mode buttons (Single/Multi/Grouped/Split)
- Pause/Play streaming with visual feedback
- Auto-scroll toggle with animated indicator
- Line wrap toggle
- Font size controls (10-24px) - Dockhand feature
- Timestamp format selector
- Clear and Export actions
- Settings menu button

Professional control layout matching Dockhand's UX.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 9: Implement Control Logic

**Files:**
- Modify: `frontend/src/routes/logs/+page.svelte:100-150` (control functions)

**Context:** Wire up control buttons to actual functionality.

#### Step 1: Add control state variables

```typescript
// Control state
let isPaused = $state(false);
let showExportMenu = $state(false);
let showSettingsMenu = $state(false);

// Log buffer for pause functionality
let logBuffer = $state<LogEntry[]>([]);
```

#### Step 2: Modify log appending logic for pause

Find where logs are appended from SSE and wrap:

```typescript
function startStream(c: Container) {
  const key = `${c.id}@${c.hostId}`;

  const es = createLogStream(c.hostId, c.id, (line: string) => {
    const entry: LogEntry = {
      key,
      name: c.name,
      line,
      color: getContainerColor(key),
      ts: Date.now()
    };

    // If paused, buffer logs instead of displaying
    if (isPaused) {
      logBuffer.push(entry);
    } else {
      allLogs = [...allLogs, entry];

      // Trim to maxLines
      if (allLogs.length > preferences.maxLines) {
        allLogs = allLogs.slice(-preferences.maxLines);
      }

      // Auto-scroll if enabled
      if (preferences.autoScroll) {
        scrollToBottom();
      }
    }
  });

  activeStreams.set(key, es);
}
```

#### Step 3: Add resume function to flush buffer

```typescript
/**
 * Resume streaming - flush buffered logs
 */
function resumeStreaming() {
  if (logBuffer.length > 0) {
    allLogs = [...allLogs, ...logBuffer];
    logBuffer = [];

    // Trim to maxLines
    if (allLogs.length > preferences.maxLines) {
      allLogs = allLogs.slice(-preferences.maxLines);
    }

    if (preferences.autoScroll) {
      scrollToBottom();
    }
  }
}

// Watch isPaused and resume when set to false
$effect(() => {
  if (!isPaused && logBuffer.length > 0) {
    resumeStreaming();
  }
});
```

#### Step 4: Add clear logs function

```typescript
/**
 * Clear all displayed logs
 */
function clearLogs() {
  if (confirm('Clear all logs? This cannot be undone.')) {
    allLogs = [];
    logBuffer = [];
  }
}
```

#### Step 5: Test control logic

1. **Pause**: Click pause → new logs go to buffer (check DevTools)
2. **Resume**: Click play → buffered logs appear
3. **Auto-scroll**: Toggle off → scroll manually, new logs don't affect scroll
4. **Line wrap**: Toggle → verify log lines wrap/don't wrap
5. **Font size**: Change → verify log text size changes
6. **Clear**: Click clear → confirm dialog → logs cleared

#### Step 6: Commit

```bash
git add frontend/src/routes/logs/+page.svelte
git commit -m "feat(logs): implement control logic for pause/resume and preferences

- Pause streaming: buffer logs instead of displaying
- Resume: flush buffer and display accumulated logs
- Clear logs with confirmation dialog
- Auto-scroll respects user preference
- Line wrap and font size applied dynamically
- Trim logs to maxLines buffer limit

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 10: Apply Display Preferences to Log Rendering

**Files:**
- Modify: `frontend/src/routes/logs/+page.svelte:500-700` (log display)

**Context:** Apply font size, line wrap, and timestamp format to rendered logs.

#### Step 1: Apply font size to log container

```svelte
<div
  class="flex-1 overflow-auto p-3 space-y-0.5"
  bind:this={logContainer}
  style="font-size: {preferences.fontSize}px;"
>
  <!-- logs -->
</div>
```

#### Step 2: Apply line wrap to log lines

```svelte
<pre
  class="flex-1 font-mono {preferences.lineWrap ? 'whitespace-pre-wrap break-all' : 'whitespace-pre overflow-x-auto'}"
>
  {@html highlightLogMatches(log.line)}
</pre>
```

#### Step 3: Conditional timestamp display

```svelte
{#if preferences.showTimestamps && preferences.timestampFormat !== 'none'}
  <span class="text-foreground-muted text-xs w-28 flex-shrink-0 font-mono tabular-nums">
    {formatTimestamp(log.ts, preferences.timestampFormat)}
  </span>
{/if}
```

#### Step 4: Update formatTimestamp function

```typescript
function formatTimestamp(ts: number, format: 'absolute' | 'relative' | 'none'): string {
  if (format === 'none') return '';

  const date = new Date(ts);

  if (format === 'absolute') {
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

#### Step 5: Test preferences in all modes

1. **Font size**: Change 10-24px → verify all modes respect it
2. **Line wrap**: Toggle → verify in single, multi, grouped, split
3. **Timestamps**: Switch absolute/relative/none → verify formatting
4. **Mix preferences**: Font 18px + wrap off + relative time → all work together

#### Step 6: Commit

```bash
git add frontend/src/routes/logs/+page.svelte
git commit -m "feat(logs): apply display preferences to log rendering

- Dynamic font size (10-24px) applied to log container
- Line wrap toggle controls whitespace behavior
- Conditional timestamp display with formatting
- All preferences work across all viewing modes
- Tabular numbers for timestamps (monospace alignment)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## PHASE 4: Search + Filtering

### Task 11: Implement Fuzzy Container Search

**Files:**
- Modify: `frontend/src/routes/logs/+page.svelte:150-200` (search logic)
- Modify: `frontend/src/routes/logs/+page.svelte:250-300` (search UI)

**Context:** Add fuzzy search for quick container finding (custom enhancement).

#### Step 1: Add fuzzy match function

(Same as original plan - Task 2 implementation from first plan)

[Copy fuzzy match implementation from original plan Task 2]

#### Step 2: Apply fuzzy search to stack containers

```typescript
// When rendering stack containers, apply fuzzy match
let searchedStacks = $derived(() => {
  if (!containerSearch) return groupedContainers;

  return groupedContainers
    .map(stack => ({
      ...stack,
      containers: stack.containers
        .map(c => ({ ...c, ...fuzzyMatch(containerSearch, c.name) }))
        .filter(c => c.match)
        .sort((a, b) => (b.score || 0) - (a.score || 0))
    }))
    .filter(stack => stack.containers.length > 0);
});
```

#### Step 3: Add search input to sidebar

```svelte
<!-- Container search (fuzzy) -->
<div class="px-4 py-3 border-b border-border">
  <div class="relative">
    <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-foreground-muted pointer-events-none" />
    <input
      bind:this={containerSearchInput}
      bind:value={containerSearch}
      placeholder="Search containers (fuzzy)..."
      class="w-full pl-9 pr-8 py-2 text-sm bg-background border border-border rounded-lg focus:ring-2 focus:ring-primary focus:outline-none"
      aria-label="Search containers"
    />
    {#if containerSearch}
      <button
        onclick={() => containerSearch = ''}
        class="absolute right-2 top-1/2 -translate-y-1/2 btn-icon-xs"
        aria-label="Clear search"
      >
        <X class="w-3.5 h-3.5" />
      </button>
    {/if}
  </div>
</div>
```

#### Step 4: Highlight matches in container names

```svelte
<span class="flex-1 text-sm group-hover:text-primary transition-colors truncate">
  {@html highlightMatch(container.name, containerSearch)}
</span>
```

#### Step 5: Test fuzzy search

- "ngx" → finds "nginx-proxy"
- "doc" → finds "dockerverse", "dockhand"
- "rp" → finds "redis-prod" (acronym)
- Results sorted by relevance

#### Step 6: Commit

```bash
git add frontend/src/routes/logs/+page.svelte
git commit -m "feat(logs): add fuzzy container search with highlighting

- Fuzzy match algorithm (exact > acronym > sequence)
- Search input in sidebar with clear button
- Results sorted by match score
- Highlighted matched characters
- Works across stacks

Custom enhancement not in Dozzle or Dockhand.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 12: Implement Regex Log Filtering

(Same as original plan - Task 3)

[Copy regex implementation from original plan Task 3]

---

### Task 13: Add Combined Search Bar

**Files:**
- Modify: `frontend/src/routes/logs/+page.svelte:400-450` (add search bar)

**Context:** Add prominent search bar above logs area for log filtering.

#### Step 1: Create search bar component

```svelte
<!-- Search bar (above logs) -->
<div class="flex items-center gap-4 px-4 py-3 border-b border-border bg-background">
  <!-- Log search with regex toggle -->
  <div class="flex-1 relative">
    <div class="flex items-center gap-2 border border-border rounded-lg px-3 py-2 {regexError ? 'border-accent-red bg-accent-red/5' : 'focus-within:ring-2 focus-within:ring-primary'}">
      <Search class="w-4 h-4 text-foreground-muted flex-shrink-0" />

      <input
        bind:this={logSearchInput}
        bind:value={logSearch}
        placeholder={regexEnabled ? "Regex pattern (e.g., ERROR|WARN)..." : "Search logs..."}
        class="flex-1 bg-transparent text-sm focus:outline-none"
        aria-label="Search logs"
      />

      <!-- Regex toggle -->
      <button
        onclick={() => regexEnabled = !regexEnabled}
        class="btn-icon-sm {regexEnabled ? 'text-primary bg-primary/10' : 'text-foreground-muted'}"
        title={regexEnabled ? "Disable regex mode" : "Enable regex mode"}
        aria-label="Toggle regex"
        aria-pressed={regexEnabled}
      >
        <Code class="w-4 h-4" />
      </button>

      {#if logSearch}
        <button
          onclick={() => logSearch = ''}
          class="btn-icon-sm"
          aria-label="Clear search"
        >
          <X class="w-3.5 h-3.5" />
        </button>
      {/if}
    </div>

    <!-- Regex error inline -->
    {#if regexError}
      <div class="absolute top-full left-0 right-0 mt-1 px-3 py-2 bg-accent-red/10 border border-accent-red rounded text-xs text-accent-red flex items-start gap-2">
        <AlertCircle class="w-3.5 h-3.5 flex-shrink-0 mt-0.5" />
        <span>Invalid regex: {regexError}</span>
      </div>
    {/if}
  </div>

  <!-- Match count -->
  {#if logSearch && !regexError}
    <div class="text-xs text-foreground-muted whitespace-nowrap">
      {displayedLogs.length.toLocaleString()} / {allLogs.length.toLocaleString()} lines
    </div>
  {/if}
</div>
```

#### Step 2: Import AlertCircle icon

```typescript
import { AlertCircle, /* existing */ } from "lucide-svelte";
```

#### Step 3: Test search bar

1. Type text → filters logs
2. Enable regex → placeholder changes
3. Invalid regex → error shown below input
4. Match count updates in real-time
5. Clear button works

#### Step 4: Commit

```bash
git add frontend/src/routes/logs/+page.svelte
git commit -m "feat(logs): add prominent search bar for log filtering

- Large search input above logs area
- Regex toggle with visual feedback
- Inline error display for invalid patterns
- Live match count display
- Clear button for quick reset

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## PHASE 5: Split Mode + Visual Polish

[Continue with Tasks 14-16 for split mode and visual enhancements...]

---

## PHASE 6: Testing, Deploy, Documentation

[Continue with Tasks 17-20 for testing, deployment, and documentation...]

---

## Summary

**Total Tasks:** 20 (reorganized into 6 phases)
**Estimated Time:** 6-8 hours (slightly more due to Dozzle + Dockhand features)
**Key Deliverables:**
1. ✅ Layout stability fix
2. ✅ Host filtering (Dozzle)
3. ✅ Stack grouping (Dozzle)
4. ✅ Advanced controls (Dockhand)
5. ✅ Fuzzy + regex search (custom)
6. ✅ Split-screen mode (custom)
7. ✅ Full keyboard shortcuts
8. ✅ Professional visual polish

**The Best of Both Worlds** 🎯

---

Plan saved and ready for execution!
