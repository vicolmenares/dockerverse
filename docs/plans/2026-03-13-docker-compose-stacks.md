# Docker Compose Stacks Management Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a Stacks page to DockerVerse that lets users view, edit, and deploy Docker Compose stacks across hosts — similar to Portainer.

**Architecture:** SSH-based file access reads compose files from where they live on the Raspberry Pi hosts. Backend adds 8 new endpoints extending the existing `/api/stacks`. Frontend adds a new `/stacks` top-level route with stack list, inline editor modal, and new stack creation modal. No external YAML libraries — plain text editing.

**Tech Stack:** Go 1.23 + Fiber v2, SvelteKit 2.x + Svelte 5 runes ($state/$derived/$effect), TailwindCSS, lucide-svelte, `runSSHCommand` existing SSH helper, `base64` for safe file transfer over SSH.

---

## Task 1: Add stack helper types and functions to backend

**Files:**
- Modify: `backend/main.go` — add after the `ContainerEventBuffer` struct (around line 710)

**Step 1: Add StackInfo and ServiceInfo structs + helper functions**

Find the line after the `ContainerEventBuffer` struct closing brace and add:

```go
// StackInfo represents a Docker Compose stack discovered on a host
type StackInfo struct {
	Name           string        `json:"name"`
	Type           string        `json:"type"` // "portainer", "dockerverse", "external", "unknown"
	HasFile        bool          `json:"hasFile"`
	ConfigFilePath string        `json:"configFilePath"` // host-resolved path
	WorkingDir     string        `json:"workingDir"`     // host-resolved path
	Status         string        `json:"status"`         // "running", "partial", "stopped"
	Services       []ServiceInfo `json:"services"`
}

// ServiceInfo represents a single container within a stack
type ServiceInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	State   string `json:"state"`
	Service string `json:"service"`
}

// detectStackType classifies a stack by its config_files label path
func detectStackType(configFilesPath string) string {
	if configFilesPath == "" {
		return "unknown"
	}
	if strings.HasPrefix(configFilesPath, "/data/compose/") {
		return "portainer"
	}
	if strings.HasPrefix(configFilesPath, "/home/pi/dockerverse-stacks/") {
		return "dockerverse"
	}
	return "external"
}

// translateStackPath converts Portainer-internal paths to host filesystem paths.
// Portainer stores files in /data/compose/... inside its container, which maps
// to /var/lib/docker/volumes/portainer_data/_data/compose/... on the host.
func translateStackPath(path string) string {
	if strings.HasPrefix(path, "/data/compose/") {
		return "/var/lib/docker/volumes/portainer_data/_data" + path
	}
	return path
}

// shellEscape wraps a string in single quotes for safe shell use
func shellEscape(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// readFileSSH reads a file from a remote host via SSH
func readFileSSH(hostID, path string) (string, error) {
	return runSSHCommand(hostID, "cat "+shellEscape(path))
}

// writeFileSSH writes content to a file on a remote host via SSH using base64
// to avoid shell escaping issues with special characters in YAML
func writeFileSSH(hostID, path, content string) error {
	encoded := base64.StdEncoding.EncodeToString([]byte(content))
	dirPath := filepath.Dir(path)
	cmd := fmt.Sprintf("mkdir -p %s && echo %s | base64 -d > %s",
		shellEscape(dirPath),
		shellEscape(encoded),
		shellEscape(path),
	)
	_, err := runSSHCommand(hostID, cmd)
	return err
}

// runComposeCmd runs a docker compose subcommand on a remote host via SSH
func runComposeCmd(hostID, configFilePath, stackName, subCmd string) (string, error) {
	cmd := fmt.Sprintf("docker compose -f %s -p %s %s 2>&1",
		shellEscape(configFilePath),
		shellEscape(stackName),
		subCmd,
	)
	return runSSHCommand(hostID, cmd)
}

// computeStackStatus returns "running", "partial", or "stopped" based on services
func computeStackStatus(services []ServiceInfo) string {
	if len(services) == 0 {
		return "stopped"
	}
	running := 0
	for _, s := range services {
		if s.State == "running" {
			running++
		}
	}
	if running == 0 {
		return "stopped"
	}
	if running == len(services) {
		return "running"
	}
	return "partial"
}
```

**Step 2: Add missing imports if not already present**

Verify these imports exist in the `import` block at the top of `main.go`. Add any missing ones:
```go
"encoding/base64"
"path/filepath"
```

**Step 3: Build to verify**

```bash
cd backend && go build ./... 2>&1
```
Expected: no errors.

**Step 4: Commit**

```bash
git add backend/main.go
git commit -m "feat(stacks): add StackInfo types and SSH file helper functions"
```

---

## Task 2: Replace existing `/api/stacks` endpoint with enriched version

**Files:**
- Modify: `backend/main.go` lines ~5240-5292 (the existing `protected.Get("/stacks", ...)` handler)

**Step 1: Replace the existing stacks handler**

Find and replace the entire block from `// GET /api/stacks?hostId=raspi1` through the closing `})` of the handler (before `// Stats`):

```go
// GET /api/stacks?hostId=raspi1
// Returns Docker Compose stacks with type, file path, and service status
protected.Get("/stacks", func(c *fiber.Ctx) error {
	hostID := c.Query("hostId")
	if hostID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "hostId required"})
	}

	cli, err := dm.GetClient(hostID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Host not found"})
	}

	ctx := context.Background()
	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Group containers by compose project
	stackMap := make(map[string]*StackInfo)

	for _, ctr := range containers {
		project := ctr.Labels["com.docker.compose.project"]
		if project == "" {
			continue
		}

		if _, exists := stackMap[project]; !exists {
			rawPath := ctr.Labels["com.docker.compose.project.config_files"]
			rawDir := ctr.Labels["com.docker.compose.project.working_dir"]
			stackType := detectStackType(rawPath)
			hostPath := translateStackPath(rawPath)
			hostDir := translateStackPath(rawDir)

			stackMap[project] = &StackInfo{
				Name:           project,
				Type:           stackType,
				HasFile:        rawPath != "",
				ConfigFilePath: hostPath,
				WorkingDir:     hostDir,
				Services:       []ServiceInfo{},
			}
		}

		svc := ServiceInfo{
			ID:      ctr.ID[:12],
			Name:    strings.TrimPrefix(ctr.Names[0], "/"),
			State:   ctr.State,
			Service: ctr.Labels["com.docker.compose.service"],
		}
		stackMap[project].Services = append(stackMap[project].Services, svc)
	}

	// Compute status and build result slice
	result := make([]StackInfo, 0, len(stackMap))
	for _, stack := range stackMap {
		stack.Status = computeStackStatus(stack.Services)
		result = append(result, *stack)
	}

	// Sort alphabetically by name
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return c.JSON(result)
})
```

**Step 2: Add `sort` to imports if missing**

Check the import block — `"sort"` should already be present. If not, add it.

**Step 3: Build**

```bash
cd backend && go build ./... 2>&1
```
Expected: no errors.

**Step 4: Manual test**

```bash
curl -s -H "Authorization: Bearer <token>" "http://localhost:3001/api/stacks?hostId=raspi1" | python3 -m json.tool | head -40
```
Expected: JSON array of stacks with `name`, `type`, `hasFile`, `configFilePath`, `status`, `services`.

**Step 5: Commit**

```bash
git add backend/main.go
git commit -m "feat(stacks): enrich /api/stacks response with type, file path, and status"
```

---

## Task 3: Add stack file read/write and compose action endpoints

**Files:**
- Modify: `backend/main.go` — add new endpoints after the `/api/stacks` GET handler (before `// Stats`)

**Step 1: Add all new stack endpoints**

After the closing `})` of the stacks GET handler, add:

```go
// GET /api/stacks/:name/file?hostId=X
// Reads the compose file for a stack via SSH
protected.Get("/stacks/:name/file", func(c *fiber.Ctx) error {
	hostID := c.Query("hostId")
	stackName := c.Params("name")
	if hostID == "" || stackName == "" {
		return c.Status(400).JSON(fiber.Map{"error": "hostId and name required"})
	}

	// Find the stack's config file path via container labels
	cli, err := dm.GetClient(hostID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Host not found"})
	}

	ctx := context.Background()
	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	var configFilePath string
	for _, ctr := range containers {
		if ctr.Labels["com.docker.compose.project"] == stackName {
			raw := ctr.Labels["com.docker.compose.project.config_files"]
			configFilePath = translateStackPath(raw)
			break
		}
	}

	if configFilePath == "" {
		return c.Status(404).JSON(fiber.Map{"error": "Compose file path not found for stack"})
	}

	content, err := readFileSSH(hostID, configFilePath)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to read file: " + err.Error()})
	}

	return c.JSON(fiber.Map{"content": content, "path": configFilePath})
})

// PUT /api/stacks/:name/file?hostId=X
// Writes the compose file for a stack via SSH
// Body: { "content": "...", "configFilePath": "..." }
protected.Put("/stacks/:name/file", func(c *fiber.Ctx) error {
	hostID := c.Query("hostId")
	stackName := c.Params("name")
	if hostID == "" || stackName == "" {
		return c.Status(400).JSON(fiber.Map{"error": "hostId and name required"})
	}

	var body struct {
		Content        string `json:"content"`
		ConfigFilePath string `json:"configFilePath"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid body"})
	}
	if body.Content == "" || body.ConfigFilePath == "" {
		return c.Status(400).JSON(fiber.Map{"error": "content and configFilePath required"})
	}

	if err := writeFileSSH(hostID, body.ConfigFilePath, body.Content); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to write file: " + err.Error()})
	}

	return c.JSON(fiber.Map{"ok": true})
})

// POST /api/stacks/:name/up?hostId=X
// Runs docker compose up -d for an existing stack
// Body: { "configFilePath": "..." }
protected.Post("/stacks/:name/up", func(c *fiber.Ctx) error {
	hostID := c.Query("hostId")
	stackName := c.Params("name")

	var body struct {
		ConfigFilePath string `json:"configFilePath"`
	}
	if err := c.BodyParser(&body); err != nil || body.ConfigFilePath == "" {
		return c.Status(400).JSON(fiber.Map{"error": "configFilePath required"})
	}

	output, err := runComposeCmd(hostID, body.ConfigFilePath, stackName, "up -d")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error(), "output": output})
	}
	return c.JSON(fiber.Map{"ok": true, "output": output})
})

// POST /api/stacks/:name/down?hostId=X
// Runs docker compose down for an existing stack
// Body: { "configFilePath": "..." }
protected.Post("/stacks/:name/down", func(c *fiber.Ctx) error {
	hostID := c.Query("hostId")
	stackName := c.Params("name")

	var body struct {
		ConfigFilePath string `json:"configFilePath"`
	}
	if err := c.BodyParser(&body); err != nil || body.ConfigFilePath == "" {
		return c.Status(400).JSON(fiber.Map{"error": "configFilePath required"})
	}

	output, err := runComposeCmd(hostID, body.ConfigFilePath, stackName, "down")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error(), "output": output})
	}
	return c.JSON(fiber.Map{"ok": true, "output": output})
})

// POST /api/stacks/:name/pull?hostId=X
// Runs docker compose pull then up -d for an existing stack
// Body: { "configFilePath": "..." }
protected.Post("/stacks/:name/pull", func(c *fiber.Ctx) error {
	hostID := c.Query("hostId")
	stackName := c.Params("name")

	var body struct {
		ConfigFilePath string `json:"configFilePath"`
	}
	if err := c.BodyParser(&body); err != nil || body.ConfigFilePath == "" {
		return c.Status(400).JSON(fiber.Map{"error": "configFilePath required"})
	}

	// Pull first, then redeploy
	pullOut, _ := runComposeCmd(hostID, body.ConfigFilePath, stackName, "pull")
	upOut, err := runComposeCmd(hostID, body.ConfigFilePath, stackName, "up -d")
	output := pullOut + "\n" + upOut
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error(), "output": output})
	}
	return c.JSON(fiber.Map{"ok": true, "output": output})
})

// POST /api/stacks/create?hostId=X
// Creates a new DockerVerse-managed stack: writes file via SSH and runs up -d
// Body: { "name": "mystack", "content": "services:\n  ..." }
protected.Post("/stacks/create", func(c *fiber.Ctx) error {
	hostID := c.Query("hostId")
	if hostID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "hostId required"})
	}

	var body struct {
		Name    string `json:"name"`
		Content string `json:"content"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid body"})
	}
	if body.Name == "" || body.Content == "" {
		return c.Status(400).JSON(fiber.Map{"error": "name and content required"})
	}

	// Validate name: alphanumeric + hyphens only
	for _, ch := range body.Name {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_') {
			return c.Status(400).JSON(fiber.Map{"error": "Stack name must be alphanumeric (hyphens and underscores allowed)"})
		}
	}

	configFilePath := "/home/pi/dockerverse-stacks/" + body.Name + "/docker-compose.yml"

	// Check if directory already exists
	checkOut, _ := runSSHCommand(hostID, "test -d "+shellEscape("/home/pi/dockerverse-stacks/"+body.Name)+" && echo exists || echo notfound")
	if strings.TrimSpace(checkOut) == "exists" {
		return c.Status(409).JSON(fiber.Map{"error": "Stack '" + body.Name + "' already exists"})
	}

	// Write the compose file
	if err := writeFileSSH(hostID, configFilePath, body.Content); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to write compose file: " + err.Error()})
	}

	// Deploy
	output, err := runComposeCmd(hostID, configFilePath, body.Name, "up -d")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error(), "output": output})
	}

	return c.JSON(fiber.Map{"ok": true, "output": output})
})

// DELETE /api/stacks/:name?hostId=X
// Stops and removes a DockerVerse-managed stack (down + rm directory)
protected.Delete("/stacks/:name", func(c *fiber.Ctx) error {
	hostID := c.Query("hostId")
	stackName := c.Params("name")
	if hostID == "" || stackName == "" {
		return c.Status(400).JSON(fiber.Map{"error": "hostId and name required"})
	}

	configFilePath := "/home/pi/dockerverse-stacks/" + stackName + "/docker-compose.yml"

	// Run compose down first
	downOut, _ := runComposeCmd(hostID, configFilePath, stackName, "down")

	// Remove directory
	rmOut, err := runSSHCommand(hostID, "rm -rf "+shellEscape("/home/pi/dockerverse-stacks/"+stackName))
	output := downOut + "\n" + rmOut
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to remove stack directory: " + err.Error(), "output": output})
	}

	return c.JSON(fiber.Map{"ok": true, "output": output})
})
```

**Step 2: Build**

```bash
cd backend && go build ./... 2>&1
```
Expected: no errors.

**Step 3: Commit**

```bash
git add backend/main.go
git commit -m "feat(stacks): add stack file read/write and compose action endpoints"
```

---

## Task 4: Create the Stacks frontend page

**Files:**
- Create: `frontend/src/routes/stacks/+page.svelte`

**Step 1: Create the file with this content**

```svelte
<script lang="ts">
  import { onMount } from "svelte";
  import {
    Layers, Plus, RefreshCw, ChevronDown, ChevronRight,
    Play, Square, RotateCcw, Download, Trash2, Pencil,
    Loader2, X, Check, AlertCircle
  } from "lucide-svelte";
  import { language } from "$lib/stores/docker";
  import { API_BASE, getAuthHeaders } from "$lib/api/docker";
  import { currentUser } from "$lib/stores/auth";
  import { goto } from "$app/navigation";

  // ── Types ──────────────────────────────────────────────────────────────
  interface ServiceInfo {
    id: string;
    name: string;
    state: string;
    service: string;
  }

  interface Stack {
    name: string;
    type: string; // portainer | dockerverse | external | unknown
    hasFile: boolean;
    configFilePath: string;
    workingDir: string;
    status: string; // running | partial | stopped
    services: ServiceInfo[];
  }

  // ── State ───────────────────────────────────────────────────────────────
  let hosts = $state<{ id: string; name: string }[]>([]);
  let selectedHostId = $state("");
  let stacks = $state<Stack[]>([]);
  let loading = $state(true);
  let refreshing = $state(false);
  let expandedStacks = $state<Set<string>>(new Set());

  // Edit modal
  let editingStack = $state<Stack | null>(null);
  let editContent = $state("");
  let editLoading = $state(false);
  let editSaving = $state(false);
  let editError = $state("");
  let actionOutput = $state("");

  // Action loading per stack
  let actionLoading = $state<Record<string, string>>({});

  // New stack modal
  let showNewStack = $state(false);
  let newName = $state("");
  let newContent = $state("");
  let newDeploying = $state(false);
  let newError = $state("");

  // ── Translations ────────────────────────────────────────────────────────
  const t = $derived($language === "es" ? {
    title: "Stacks",
    newStack: "Nuevo Stack",
    noStacks: "No hay stacks en este host",
    selectHost: "Selecciona un host",
    running: "corriendo",
    partial: "parcial",
    stopped: "detenido",
    edit: "Editar",
    up: "Desplegar",
    down: "Detener",
    pull: "Pull & Redeploy",
    delete: "Eliminar",
    save: "Guardar",
    saveAndDeploy: "Guardar y Desplegar",
    cancel: "Cancelar",
    deploy: "Desplegar",
    stackName: "Nombre del stack",
    composeContent: "Contenido del compose file",
    deleteConfirm: "¿Eliminar este stack? Se ejecutará docker compose down y se borrará el directorio.",
    outputLabel: "Output del comando:",
    services: "servicios",
  } : {
    title: "Stacks",
    newStack: "New Stack",
    noStacks: "No stacks found on this host",
    selectHost: "Select a host",
    running: "running",
    partial: "partial",
    stopped: "stopped",
    edit: "Edit",
    up: "Deploy",
    down: "Stop",
    pull: "Pull & Redeploy",
    delete: "Delete",
    save: "Save",
    saveAndDeploy: "Save & Deploy",
    cancel: "Cancel",
    deploy: "Deploy",
    stackName: "Stack name",
    composeContent: "Compose file content",
    deleteConfirm: "Delete this stack? This will run docker compose down and remove the directory.",
    outputLabel: "Command output:",
    services: "services",
  });

  // ── Auth guard ──────────────────────────────────────────────────────────
  $effect(() => {
    if ($currentUser && $currentUser.role !== "admin") {
      goto("/");
    }
  });

  // ── Data fetching ───────────────────────────────────────────────────────
  async function fetchHosts() {
    try {
      const res = await fetch(`${API_BASE}/api/environments`, { headers: getAuthHeaders() });
      if (res.ok) {
        const data = await res.json();
        hosts = data.map((h: { id: string; name: string }) => ({ id: h.id, name: h.name }));
        if (hosts.length > 0 && !selectedHostId) {
          selectedHostId = hosts[0].id;
        }
      }
    } catch (e) { console.error("Failed to fetch hosts:", e); }
  }

  async function fetchStacks() {
    if (!selectedHostId) return;
    refreshing = true;
    try {
      const res = await fetch(`${API_BASE}/api/stacks?hostId=${selectedHostId}`, {
        headers: getAuthHeaders()
      });
      if (res.ok) {
        stacks = await res.json();
      }
    } catch (e) { console.error("Failed to fetch stacks:", e); }
    loading = false;
    refreshing = false;
  }

  onMount(async () => {
    await fetchHosts();
    await fetchStacks();
  });

  $effect(() => {
    selectedHostId;
    fetchStacks();
  });

  // ── Stack actions ───────────────────────────────────────────────────────
  function setActionLoading(name: string, action: string) {
    actionLoading = { ...actionLoading, [name]: action };
  }
  function clearActionLoading(name: string) {
    const next = { ...actionLoading };
    delete next[name];
    actionLoading = next;
  }

  async function stackAction(stack: Stack, action: "up" | "down" | "pull") {
    setActionLoading(stack.name, action);
    actionOutput = "";
    try {
      const res = await fetch(`${API_BASE}/api/stacks/${stack.name}/${action}?hostId=${selectedHostId}`, {
        method: "POST",
        headers: { ...getAuthHeaders(), "Content-Type": "application/json" },
        body: JSON.stringify({ configFilePath: stack.configFilePath })
      });
      const data = await res.json();
      actionOutput = data.output || "";
      await fetchStacks();
    } catch (e) { console.error(e); }
    clearActionLoading(stack.name);
  }

  async function deleteStack(stack: Stack) {
    if (!confirm(t.deleteConfirm)) return;
    setActionLoading(stack.name, "delete");
    try {
      await fetch(`${API_BASE}/api/stacks/${stack.name}?hostId=${selectedHostId}`, {
        method: "DELETE",
        headers: getAuthHeaders()
      });
      await fetchStacks();
    } catch (e) { console.error(e); }
    clearActionLoading(stack.name);
  }

  // ── Edit modal ───────────────────────────────────────────────────────────
  async function openEdit(stack: Stack) {
    editingStack = stack;
    editContent = "";
    editError = "";
    actionOutput = "";
    editLoading = true;
    try {
      const res = await fetch(
        `${API_BASE}/api/stacks/${stack.name}/file?hostId=${selectedHostId}`,
        { headers: getAuthHeaders() }
      );
      if (res.ok) {
        const data = await res.json();
        editContent = data.content;
      } else {
        editError = "Failed to read compose file";
      }
    } catch (e) { editError = String(e); }
    editLoading = false;
  }

  function closeEdit() {
    editingStack = null;
    editContent = "";
    editError = "";
    actionOutput = "";
  }

  async function saveEdit(andDeploy = false) {
    if (!editingStack) return;
    editSaving = true;
    editError = "";
    actionOutput = "";
    try {
      // Save file
      const saveRes = await fetch(
        `${API_BASE}/api/stacks/${editingStack.name}/file?hostId=${selectedHostId}`,
        {
          method: "PUT",
          headers: { ...getAuthHeaders(), "Content-Type": "application/json" },
          body: JSON.stringify({ content: editContent, configFilePath: editingStack.configFilePath })
        }
      );
      if (!saveRes.ok) {
        const err = await saveRes.json();
        editError = err.error || "Save failed";
        editSaving = false;
        return;
      }
      // Optionally deploy
      if (andDeploy) {
        const upRes = await fetch(
          `${API_BASE}/api/stacks/${editingStack.name}/up?hostId=${selectedHostId}`,
          {
            method: "POST",
            headers: { ...getAuthHeaders(), "Content-Type": "application/json" },
            body: JSON.stringify({ configFilePath: editingStack.configFilePath })
          }
        );
        const upData = await upRes.json();
        actionOutput = upData.output || "";
        if (!upRes.ok) {
          editError = upData.error || "Deploy failed";
          editSaving = false;
          return;
        }
      }
      await fetchStacks();
    } catch (e) { editError = String(e); }
    editSaving = false;
    if (!editError && !andDeploy) closeEdit();
  }

  // ── New stack modal ──────────────────────────────────────────────────────
  async function createStack() {
    newDeploying = true;
    newError = "";
    try {
      const res = await fetch(`${API_BASE}/api/stacks/create?hostId=${selectedHostId}`, {
        method: "POST",
        headers: { ...getAuthHeaders(), "Content-Type": "application/json" },
        body: JSON.stringify({ name: newName.trim(), content: newContent })
      });
      const data = await res.json();
      if (!res.ok) {
        newError = data.error || "Create failed";
        newDeploying = false;
        return;
      }
      showNewStack = false;
      newName = "";
      newContent = "";
      await fetchStacks();
    } catch (e) { newError = String(e); }
    newDeploying = false;
  }

  // ── UI helpers ───────────────────────────────────────────────────────────
  function toggleExpand(name: string) {
    const next = new Set(expandedStacks);
    if (next.has(name)) next.delete(name);
    else next.add(name);
    expandedStacks = next;
  }

  function statusColor(status: string) {
    if (status === "running") return "text-green-400";
    if (status === "partial") return "text-yellow-400";
    return "text-gray-400";
  }

  function typeBadge(type: string) {
    if (type === "portainer") return "bg-gray-700 text-gray-300";
    if (type === "dockerverse") return "bg-blue-900 text-blue-300";
    if (type === "external") return "bg-yellow-900 text-yellow-300";
    return "bg-gray-800 text-gray-500";
  }

  function typeLabel(type: string) {
    if (type === "portainer") return "Portainer";
    if (type === "dockerverse") return "DockerVerse";
    if (type === "external") return "External";
    return "Unknown";
  }

  function runningCount(services: ServiceInfo[]) {
    return services.filter(s => s.State === "running").length;
  }
</script>

<!-- Page -->
<div class="p-6 space-y-6 min-h-screen">
  <!-- Header -->
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-3">
      <Layers class="w-6 h-6 text-blue-400" />
      <h1 class="text-xl font-semibold text-white">{t.title}</h1>
    </div>
    <div class="flex items-center gap-3">
      <!-- Host selector -->
      <select
        bind:value={selectedHostId}
        class="select select-sm bg-gray-800 border-gray-700 text-white"
      >
        {#each hosts as host}
          <option value={host.id}>{host.name}</option>
        {/each}
      </select>

      <!-- Refresh -->
      <button
        onclick={() => fetchStacks()}
        class="btn btn-ghost btn-icon"
        disabled={refreshing}
      >
        <RefreshCw class="w-4 h-4 {refreshing ? 'animate-spin' : ''}" />
      </button>

      <!-- New Stack -->
      <button
        onclick={() => { showNewStack = true; newError = ""; }}
        class="btn btn-primary btn-sm flex items-center gap-2"
      >
        <Plus class="w-4 h-4" />
        {t.newStack}
      </button>
    </div>
  </div>

  <!-- Loading -->
  {#if loading}
    <div class="flex items-center justify-center py-20">
      <Loader2 class="w-8 h-8 animate-spin text-blue-400" />
    </div>
  {:else if stacks.length === 0}
    <div class="text-center py-20 text-gray-500">
      <Layers class="w-12 h-12 mx-auto mb-3 opacity-30" />
      <p>{selectedHostId ? t.noStacks : t.selectHost}</p>
    </div>
  {:else}
    <!-- Stack list -->
    <div class="space-y-2">
      {#each stacks as stack}
        {@const expanded = expandedStacks.has(stack.name)}
        {@const busy = actionLoading[stack.name]}
        <div class="bg-gray-800 rounded-lg border border-gray-700 overflow-hidden">
          <!-- Stack header row -->
          <button
            onclick={() => toggleExpand(stack.name)}
            class="w-full flex items-center gap-3 px-4 py-3 hover:bg-gray-750 text-left"
          >
            {#if expanded}
              <ChevronDown class="w-4 h-4 text-gray-400 flex-shrink-0" />
            {:else}
              <ChevronRight class="w-4 h-4 text-gray-400 flex-shrink-0" />
            {/if}

            <span class="font-medium text-white flex-1">{stack.name}</span>

            <!-- Status dot + count -->
            <span class="text-sm {statusColor(stack.status)}">
              ● {runningCount(stack.services)}/{stack.services.length} {t.services}
            </span>

            <!-- Type badge -->
            <span class="text-xs px-2 py-0.5 rounded {typeBadge(stack.type)}">
              {typeLabel(stack.type)}
            </span>
          </button>

          <!-- Expanded: services + actions -->
          {#if expanded}
            <div class="border-t border-gray-700 px-4 pb-3">
              <!-- Service list -->
              {#if stack.services.length > 0}
                <div class="py-2 space-y-1">
                  {#each stack.services as svc}
                    <div class="flex items-center gap-2 text-sm">
                      <span class="w-2 h-2 rounded-full flex-shrink-0 {svc.state === 'running' ? 'bg-green-400' : 'bg-gray-500'}"></span>
                      <span class="text-gray-300">{svc.name}</span>
                      <span class="text-gray-500 text-xs">{svc.state}</span>
                    </div>
                  {/each}
                </div>
              {/if}

              <!-- Action output (shown inline when action runs from this stack) -->

              <!-- Actions row -->
              <div class="flex items-center gap-2 mt-2 flex-wrap">
                {#if stack.hasFile}
                  <button
                    onclick={() => openEdit(stack)}
                    class="btn btn-ghost btn-sm flex items-center gap-1"
                    disabled={!!busy}
                  >
                    <Pencil class="w-3.5 h-3.5" />
                    {t.edit}
                  </button>
                {/if}

                <button
                  onclick={() => stackAction(stack, "up")}
                  class="btn btn-ghost btn-sm flex items-center gap-1 text-green-400 hover:text-green-300"
                  disabled={!!busy}
                >
                  {#if busy === "up"}
                    <Loader2 class="w-3.5 h-3.5 animate-spin" />
                  {:else}
                    <Play class="w-3.5 h-3.5" />
                  {/if}
                  {t.up}
                </button>

                <button
                  onclick={() => stackAction(stack, "down")}
                  class="btn btn-ghost btn-sm flex items-center gap-1 text-red-400 hover:text-red-300"
                  disabled={!!busy}
                >
                  {#if busy === "down"}
                    <Loader2 class="w-3.5 h-3.5 animate-spin" />
                  {:else}
                    <Square class="w-3.5 h-3.5" />
                  {/if}
                  {t.down}
                </button>

                <button
                  onclick={() => stackAction(stack, "pull")}
                  class="btn btn-ghost btn-sm flex items-center gap-1 text-blue-400 hover:text-blue-300"
                  disabled={!!busy}
                >
                  {#if busy === "pull"}
                    <Loader2 class="w-3.5 h-3.5 animate-spin" />
                  {:else}
                    <Download class="w-3.5 h-3.5" />
                  {/if}
                  {t.pull}
                </button>

                {#if stack.type === "dockerverse"}
                  <button
                    onclick={() => deleteStack(stack)}
                    class="btn btn-ghost btn-sm flex items-center gap-1 text-red-500 hover:text-red-400 ml-auto"
                    disabled={!!busy}
                  >
                    {#if busy === "delete"}
                      <Loader2 class="w-3.5 h-3.5 animate-spin" />
                    {:else}
                      <Trash2 class="w-3.5 h-3.5" />
                    {/if}
                    {t.delete}
                  </button>
                {/if}
              </div>
            </div>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>

<!-- Edit Modal -->
{#if editingStack}
  <div class="fixed inset-0 bg-black/60 z-50 flex items-center justify-center p-4">
    <div class="bg-gray-900 rounded-xl border border-gray-700 w-full max-w-3xl max-h-[90vh] flex flex-col">
      <!-- Modal header -->
      <div class="flex items-center justify-between px-5 py-4 border-b border-gray-700">
        <div>
          <h2 class="font-semibold text-white">{editingStack.name}</h2>
          <p class="text-xs text-gray-500 mt-0.5">{editingStack.configFilePath}</p>
        </div>
        <button onclick={closeEdit} class="btn btn-ghost btn-icon">
          <X class="w-5 h-5" />
        </button>
      </div>

      <!-- Modal body -->
      <div class="flex-1 overflow-auto p-5 flex flex-col gap-4">
        {#if editLoading}
          <div class="flex items-center justify-center py-10">
            <Loader2 class="w-6 h-6 animate-spin text-blue-400" />
          </div>
        {:else if editError && !editContent}
          <div class="flex items-center gap-2 text-red-400 text-sm">
            <AlertCircle class="w-4 h-4" />
            {editError}
          </div>
        {:else}
          <textarea
            bind:value={editContent}
            class="w-full flex-1 bg-gray-800 text-gray-100 font-mono text-sm p-4 rounded-lg border border-gray-700 focus:outline-none focus:border-blue-500 resize-none"
            style="min-height: 400px;"
            spellcheck="false"
          ></textarea>

          {#if editError}
            <div class="flex items-center gap-2 text-red-400 text-sm">
              <AlertCircle class="w-4 h-4" />
              {editError}
            </div>
          {/if}

          {#if actionOutput}
            <div class="bg-gray-950 rounded-lg p-3">
              <p class="text-xs text-gray-500 mb-1">{t.outputLabel}</p>
              <pre class="text-xs text-gray-300 whitespace-pre-wrap overflow-auto max-h-40">{actionOutput}</pre>
            </div>
          {/if}
        {/if}
      </div>

      <!-- Modal footer -->
      {#if !editLoading && editContent}
        <div class="flex items-center justify-end gap-3 px-5 py-4 border-t border-gray-700">
          <button onclick={closeEdit} class="btn btn-ghost btn-sm">{t.cancel}</button>
          <button
            onclick={() => saveEdit(false)}
            class="btn btn-secondary btn-sm"
            disabled={editSaving}
          >
            {#if editSaving}
              <Loader2 class="w-4 h-4 animate-spin" />
            {:else}
              <Check class="w-4 h-4" />
            {/if}
            {t.save}
          </button>
          <button
            onclick={() => saveEdit(true)}
            class="btn btn-primary btn-sm"
            disabled={editSaving}
          >
            {#if editSaving}
              <Loader2 class="w-4 h-4 animate-spin" />
            {:else}
              <RotateCcw class="w-4 h-4" />
            {/if}
            {t.saveAndDeploy}
          </button>
        </div>
      {/if}
    </div>
  </div>
{/if}

<!-- New Stack Modal -->
{#if showNewStack}
  <div class="fixed inset-0 bg-black/60 z-50 flex items-center justify-center p-4">
    <div class="bg-gray-900 rounded-xl border border-gray-700 w-full max-w-2xl flex flex-col">
      <div class="flex items-center justify-between px-5 py-4 border-b border-gray-700">
        <h2 class="font-semibold text-white">{t.newStack}</h2>
        <button onclick={() => { showNewStack = false; newError = ""; }} class="btn btn-ghost btn-icon">
          <X class="w-5 h-5" />
        </button>
      </div>

      <div class="p-5 space-y-4">
        <div>
          <label class="block text-sm text-gray-400 mb-1">{t.stackName}</label>
          <input
            bind:value={newName}
            type="text"
            placeholder="my-stack"
            class="input input-sm w-full bg-gray-800 border-gray-700 text-white font-mono"
          />
        </div>

        <div>
          <label class="block text-sm text-gray-400 mb-1">{t.composeContent}</label>
          <textarea
            bind:value={newContent}
            class="w-full bg-gray-800 text-gray-100 font-mono text-sm p-4 rounded-lg border border-gray-700 focus:outline-none focus:border-blue-500 resize-none"
            style="min-height: 300px;"
            placeholder="services:&#10;  myapp:&#10;    image: nginx:latest&#10;    ports:&#10;      - '8080:80'"
            spellcheck="false"
          ></textarea>
        </div>

        {#if newError}
          <div class="flex items-center gap-2 text-red-400 text-sm">
            <AlertCircle class="w-4 h-4" />
            {newError}
          </div>
        {/if}
      </div>

      <div class="flex items-center justify-end gap-3 px-5 py-4 border-t border-gray-700">
        <button onclick={() => { showNewStack = false; newError = ""; }} class="btn btn-ghost btn-sm">
          {t.cancel}
        </button>
        <button
          onclick={createStack}
          class="btn btn-primary btn-sm flex items-center gap-2"
          disabled={newDeploying || !newName.trim() || !newContent.trim()}
        >
          {#if newDeploying}
            <Loader2 class="w-4 h-4 animate-spin" />
          {:else}
            <Play class="w-4 h-4" />
          {/if}
          {t.deploy}
        </button>
      </div>
    </div>
  </div>
{/if}
```

**Step 2: Check TypeScript**

```bash
cd frontend && npm run check 2>&1 | tail -20
```
Expected: no errors (or only pre-existing warnings unrelated to stacks).

**Step 3: Commit**

```bash
git add frontend/src/routes/stacks/+page.svelte
git commit -m "feat(stacks): add Stacks page with view, edit, and deploy UI"
```

---

## Task 5: Add Stacks entry to sidebar

**Files:**
- Modify: `frontend/src/routes/+layout.svelte` lines ~92-99

**Step 1: Add Layers icon import**

Find the line with the existing lucide imports (line ~8). Add `Layers` to the import:
```typescript
import {
  Home, ScrollText, SquareTerminal, Shield,
  Settings as SettingsIcon, Layers
} from "lucide-svelte";
```

**Step 2: Add stacks item to mainItems**

Find the `mainItems` derived array (around line 94) and add the stacks entry after `shell`:

```typescript
const mainItems = $derived([
  { id: "dashboard", icon: Home, label: "Dashboard", href: "/" },
  { id: "logs", icon: ScrollText, label: "Logs", href: "/logs" },
  { id: "shell", icon: SquareTerminal, label: "Shell", href: "/shell" },
  { id: "stacks", icon: Layers, label: $language === "es" ? "Stacks" : "Stacks", href: "/stacks" },
  { id: "security-scans", icon: Shield, label: $language === "es" ? "Seguridad" : "Security", href: "/security" },
  { id: "settings", icon: SettingsIcon, label: $language === "es" ? "Configuración" : "Settings", href: "/settings" },
]);
```

**Step 3: Check TypeScript**

```bash
cd frontend && npm run check 2>&1 | tail -10
```
Expected: no errors.

**Step 4: Commit**

```bash
git add frontend/src/routes/+layout.svelte
git commit -m "feat(stacks): add Stacks entry to main sidebar navigation"
```

---

## Task 6: Build, test, and deploy

**Step 1: Full backend build**

```bash
cd backend && go build ./... 2>&1
```
Expected: no errors.

**Step 2: Full frontend check**

```bash
cd frontend && npm run check 2>&1
```
Expected: no new errors.

**Step 3: Deploy to Raspi**

```bash
./deploy-to-raspi.sh
```
Expected: build succeeds, container restarts healthy.

**Step 4: Manual browser testing checklist**

Open `http://192.168.1.145:3007/stacks` and verify:

- [ ] Sidebar shows "Stacks" entry with Layers icon
- [ ] Page loads and shows stacks for the selected host
- [ ] Each stack shows correct type badge (Portainer/External/DockerVerse)
- [ ] Status shows correct running count (e.g., `● 1/1 services`)
- [ ] Expanding a stack shows service list with state dots
- [ ] Click "Edit" on a Portainer stack → modal opens with compose file content
- [ ] Edit content and click "Save" → success (no deploy)
- [ ] Click "Save & Deploy" → modal shows command output
- [ ] Click "Deploy" (up) on a stopped stack → stack comes back up
- [ ] Click "Stop" (down) on a running stack → stack stops
- [ ] Click "+ New Stack" → modal opens
- [ ] Create a test stack with a simple nginx compose → deploys successfully
- [ ] New stack appears in list with `[DockerVerse]` badge
- [ ] Delete button visible only on DockerVerse stacks
- [ ] Delete a DockerVerse stack → confirms, stack disappears from list

**Step 5: Update DEVELOPMENT_CONTINUATION_GUIDE.md**

In the guide, update the v2.7.0 section to mark it complete and add v2.8.0 entry:

```markdown
### v2.7.0 ✅ COMPLETADO (2026-03-13)
- [x] Audit log
- [x] Container Activity chart
- [x] Docker Compose management (Stacks page)
- [x] Profile page enhancements

### v2.8.0 (Planificado)
- [ ] QR code offline (reemplazar api.qrserver.com con qrcode npm)
- [ ] Container Activity Chart: filtro por host
- [ ] Container creation wizard
```

**Step 6: Commit everything and push**

```bash
git add docs/DEVELOPMENT_CONTINUATION_GUIDE.md
git commit -m "docs: update roadmap to reflect v2.7.0 completion with Stacks feature"
git push origin master
```
