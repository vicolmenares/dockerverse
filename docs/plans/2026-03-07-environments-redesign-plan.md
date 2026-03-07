# Environments Redesign — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the card-grid environments list and basic modal with a dockhand-inspired flat table + full-featured 4-tab modal, adding TLS, per-env settings, detect sockets, colored labels, and pre-save connection test.

**Architecture:** Backend (Go 1.23, Fiber) extends the `Environment` struct with new fields and adds two new endpoints. Frontend (SvelteKit 2 + Svelte 5) rewrites the list page as a `<table>` and rewrites `EnvironmentModal.svelte` into a 4-tab panel. All persistence is JSON files (no database migrations needed — just backward-compatible struct changes).

**Tech Stack:** Go 1.23, Fiber v2, Docker SDK, SvelteKit 2, Svelte 5 ($state/$derived/$effect), Tailwind CSS, lucide-svelte

---

## Task 1: Extend Environment struct + migrate labels

**Files:**
- Modify: `backend/main.go` (Environment struct, ~line 594)

**Step 1: Read the current struct**

Open `backend/main.go`, find the `Environment` struct at line ~594. Read lines 594–618.

**Step 2: Replace the struct with the extended version**

Replace the entire `Environment` struct with:

```go
type Environment struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	ConnectionType string   `json:"connectionType"` // "socket", "tcp", "tcp+tls"
	Address        string   `json:"address"`        // kept for migration compat
	SocketPath     string   `json:"socketPath"`     // configurable socket path
	Host           string   `json:"host"`           // hostname/IP for tcp modes
	Port           int      `json:"port"`           // port for tcp modes
	Protocol       string   `json:"protocol"`       // "http" or "https"
	IsLocal        bool     `json:"isLocal"`
	// TLS
	TlsCa         string `json:"tlsCa"`
	TlsCert       string `json:"tlsCert"`
	TlsKey        string `json:"tlsKey"`
	TlsSkipVerify bool   `json:"tlsSkipVerify"`
	// Metadata
	Labels    []string `json:"labels"`
	PublicIP  string   `json:"publicIp"`
	Timezone  string   `json:"timezone"`
	Status    string   `json:"status"`
	DockerVersion string `json:"dockerVersion"`
	// Update settings
	AutoUpdate     bool   `json:"autoUpdate"`
	UpdateSchedule string `json:"updateSchedule"`
	ImagePrune     bool   `json:"imagePrune"`
	ImagePruneMode string `json:"imagePruneMode"` // "dangling" | "all"
	ImagePruneCron string `json:"imagePruneCron"`
	// Monitoring flags
	EventTracking    bool `json:"eventTracking"`
	VulnScanning     bool `json:"vulnScanning"`
	CollectMetrics   bool `json:"collectMetrics"`
	HighlightChanges bool `json:"highlightChanges"`
	// Disk warning
	DiskWarningEnabled   bool    `json:"diskWarningEnabled"`
	DiskWarningMode      string  `json:"diskWarningMode"`      // "percentage" | "absolute"
	DiskWarningThreshold float64 `json:"diskWarningThreshold"` // % or GB
}
```

**Step 3: Add labels migration in `EnvironmentStore.load()`**

After `json.Unmarshal(data, &es.Environments)` succeeds, add migration to handle old comma-string labels:

```go
// Migrate old data: populate SocketPath/Host/Port from Address if empty
for _, env := range es.Environments {
    if env.SocketPath == "" && env.Host == "" {
        if env.ConnectionType == "socket" || env.IsLocal {
            env.SocketPath = env.Address
            if env.SocketPath == "" {
                env.SocketPath = "/var/run/docker.sock"
            }
        } else {
            // Parse "host:port" or "http://host:port"
            addr := strings.TrimPrefix(env.Address, "http://")
            addr = strings.TrimPrefix(addr, "https://")
            if h, p, err := net.SplitHostPort(addr); err == nil {
                env.Host = h
                if port, err := strconv.Atoi(p); err == nil {
                    env.Port = port
                }
            } else {
                env.Host = addr
                env.Port = 2375
            }
        }
    }
    if env.Port == 0 {
        env.Port = 2375
    }
    if env.SocketPath == "" && (env.ConnectionType == "socket" || env.IsLocal) {
        env.SocketPath = "/var/run/docker.sock"
    }
}
```

Note: `net` and `strconv` are already imported in most Go files; add them to imports if missing.

**Step 4: Update `migrateFromHosts()` to set SocketPath/Host/Port**

In the migration loop, set `SocketPath` when local, `Host`+`Port` when remote:

```go
env := &Environment{
    ID:            h.ID,
    Name:          h.Name,
    ConnectionType: connType,
    Address:       addr,
    Protocol:      protocol,
    IsLocal:       h.IsLocal,
    Status:        "unknown",
    EventTracking: true,
    CollectMetrics: true,
}
if h.IsLocal {
    env.SocketPath = h.Address
    if env.SocketPath == "" {
        env.SocketPath = "/var/run/docker.sock"
    }
} else {
    // parse addr into Host + Port
    a := strings.TrimPrefix(addr, "http://")
    a = strings.TrimPrefix(a, "https://")
    if host, port, err := net.SplitHostPort(a); err == nil {
        env.Host = host
        if p, err := strconv.Atoi(port); err == nil {
            env.Port = p
        }
    } else {
        env.Host = a
        env.Port = 2375
    }
}
```

**Step 5: Verify the backend compiles**

```bash
cd /home/pi/dockerverse/backend
# or locally:
cd /path/to/dockerverse/backend
go build ./...
```

Expected: no errors (or only "declared and not used" for new fields, not fatal)

**Step 6: Commit locally**

```bash
git add backend/main.go
git commit -m "feat(backend): extend Environment struct with TLS, labels[]string, socketPath, monitoring fields"
```

---

## Task 2: Update Docker client creation to support TLS and new fields

**Files:**
- Modify: `backend/main.go` — `DockerManager` init / `GetClient()` / `connectDockerHost()` function

**Step 1: Find where Docker clients are created**

Search for `client.NewClientWithOpts` or `client.FromEnv` in `main.go`. This is likely in a function that iterates over hosts/environments to create Docker clients.

**Step 2: Update client creation to use SocketPath / Host+Port**

When creating a client for an environment:

```go
func createDockerClient(env *Environment) (*client.Client, error) {
    var opts []client.Opt
    opts = append(opts, client.WithAPIVersionNegotiation())

    switch env.ConnectionType {
    case "socket", "":
        socketPath := env.SocketPath
        if socketPath == "" {
            socketPath = env.Address // backward compat
        }
        if socketPath == "" {
            socketPath = "/var/run/docker.sock"
        }
        opts = append(opts, client.WithHost("unix://"+socketPath))

    case "tcp+tls":
        host := fmt.Sprintf("tcp://%s:%d", env.Host, env.Port)
        opts = append(opts, client.WithHost(host))
        if env.TlsCa != "" || env.TlsCert != "" {
            tlsCfg, err := buildTLSConfig(env)
            if err != nil {
                return nil, fmt.Errorf("TLS config: %w", err)
            }
            opts = append(opts, client.WithHTTPClient(&http.Client{
                Transport: &http.Transport{TLSClientConfig: tlsCfg},
            }))
        }

    default: // "tcp"
        host := fmt.Sprintf("tcp://%s:%d", env.Host, env.Port)
        if env.Host == "" {
            // fallback to old Address field
            host = "tcp://" + env.Address
        }
        opts = append(opts, client.WithHost(host))
    }

    return client.NewClientWithOpts(opts...)
}

func buildTLSConfig(env *Environment) (*tls.Config, error) {
    tlsCfg := &tls.Config{
        InsecureSkipVerify: env.TlsSkipVerify,
    }
    if env.TlsCa != "" {
        pool := x509.NewCertPool()
        if !pool.AppendCertsFromPEM([]byte(env.TlsCa)) {
            return nil, fmt.Errorf("invalid CA certificate")
        }
        tlsCfg.RootCAs = pool
    }
    if env.TlsCert != "" && env.TlsKey != "" {
        cert, err := tls.X509KeyPair([]byte(env.TlsCert), []byte(env.TlsKey))
        if err != nil {
            return nil, fmt.Errorf("invalid client cert/key: %w", err)
        }
        tlsCfg.Certificates = []tls.Certificate{cert}
    }
    return tlsCfg, nil
}
```

Add imports at top if missing: `"crypto/tls"`, `"crypto/x509"`, `"net/http"`

**Step 3: Update DockerManager initialization**

Find where environments are iterated to build `dm.clients`. Replace the client-creation call with `createDockerClient(env)`.

**Step 4: Compile**

```bash
go build ./...
```

Expected: compiles. If import errors, add missing imports.

**Step 5: Commit**

```bash
git add backend/main.go
git commit -m "feat(backend): TLS Docker client support for tcp+tls connection type"
```

---

## Task 3: Update environment CRUD endpoints for new fields

**Files:**
- Modify: `backend/main.go` — POST /api/environments (~line 4648) and PUT /api/environments/:id (~line 4692)

**Step 1: Find the POST /api/environments handler**

Search for `4648` or `"POST"` + `"/environments"`. Read the current handler to see what fields it parses.

**Step 2: Update the POST handler to parse all new fields**

Replace the body parsing struct with:

```go
var body struct {
    ID               string   `json:"id"`
    Name             string   `json:"name"`
    ConnectionType   string   `json:"connectionType"`
    SocketPath       string   `json:"socketPath"`
    Host             string   `json:"host"`
    Port             int      `json:"port"`
    Protocol         string   `json:"protocol"`
    TlsCa            string   `json:"tlsCa"`
    TlsCert          string   `json:"tlsCert"`
    TlsKey           string   `json:"tlsKey"`
    TlsSkipVerify    bool     `json:"tlsSkipVerify"`
    Labels           []string `json:"labels"`
    PublicIP         string   `json:"publicIp"`
    Timezone         string   `json:"timezone"`
    AutoUpdate       bool     `json:"autoUpdate"`
    UpdateSchedule   string   `json:"updateSchedule"`
    ImagePrune       bool     `json:"imagePrune"`
    ImagePruneMode   string   `json:"imagePruneMode"`
    ImagePruneCron   string   `json:"imagePruneCron"`
    EventTracking    bool     `json:"eventTracking"`
    VulnScanning     bool     `json:"vulnScanning"`
    CollectMetrics   bool     `json:"collectMetrics"`
    HighlightChanges bool     `json:"highlightChanges"`
    DiskWarningEnabled   bool    `json:"diskWarningEnabled"`
    DiskWarningMode      string  `json:"diskWarningMode"`
    DiskWarningThreshold float64 `json:"diskWarningThreshold"`
}
```

Then build the `Environment` from body, setting `IsLocal` based on ConnectionType:

```go
isLocal := body.ConnectionType == "socket" || body.ConnectionType == ""
socketPath := body.SocketPath
if socketPath == "" && isLocal {
    socketPath = "/var/run/docker.sock"
}
port := body.Port
if port == 0 {
    port = 2375
}
// Build backward-compat Address
address := socketPath
if !isLocal {
    address = fmt.Sprintf("%s:%d", body.Host, port)
}

env := &Environment{
    ID:             body.ID,
    Name:           body.Name,
    ConnectionType: body.ConnectionType,
    Address:        address,
    SocketPath:     socketPath,
    Host:           body.Host,
    Port:           port,
    Protocol:       body.Protocol,
    IsLocal:        isLocal,
    TlsCa:          body.TlsCa,
    TlsCert:        body.TlsCert,
    TlsKey:         body.TlsKey,
    TlsSkipVerify:  body.TlsSkipVerify,
    Labels:         body.Labels,
    PublicIP:       body.PublicIP,
    Timezone:       body.Timezone,
    Status:         "unknown",
    AutoUpdate:     body.AutoUpdate,
    UpdateSchedule: body.UpdateSchedule,
    ImagePrune:     body.ImagePrune,
    ImagePruneMode: body.ImagePruneMode,
    ImagePruneCron: body.ImagePruneCron,
    EventTracking:  body.EventTracking,
    VulnScanning:   body.VulnScanning,
    CollectMetrics: body.CollectMetrics,
    HighlightChanges: body.HighlightChanges,
    DiskWarningEnabled:   body.DiskWarningEnabled,
    DiskWarningMode:      body.DiskWarningMode,
    DiskWarningThreshold: body.DiskWarningThreshold,
}
```

**Step 3: Apply same changes to PUT /api/environments/:id handler**

Same body struct, same field mapping. Keep the existing `env.ID` from the URL param, not from body.

**Step 4: Update GET /api/environments to return all new fields**

The GET handler returns `envStore.GetAll()` — since the struct now has all fields, they'll be included automatically in `json.Marshal`. No changes needed unless the handler manually builds a response struct.

**Step 5: Compile and verify**

```bash
go build ./...
```

**Step 6: Commit**

```bash
git add backend/main.go
git commit -m "feat(backend): environment CRUD endpoints handle all new fields"
```

---

## Task 4: Add pre-save connection test endpoint

**Files:**
- Modify: `backend/main.go` — add route near line 4758

**Step 1: Add the route before the existing `/environments/:id/test`**

```go
// POST /api/environments/test — test connection WITHOUT saving (pre-save)
protected.Post("/environments/test", func(c *fiber.Ctx) error {
    var body struct {
        ConnectionType string `json:"connectionType"`
        SocketPath     string `json:"socketPath"`
        Host           string `json:"host"`
        Port           int    `json:"port"`
        Protocol       string `json:"protocol"`
        TlsCa          string `json:"tlsCa"`
        TlsCert        string `json:"tlsCert"`
        TlsKey         string `json:"tlsKey"`
        TlsSkipVerify  bool   `json:"tlsSkipVerify"`
    }
    if err := c.BodyParser(&body); err != nil {
        return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
    }
    if body.Port == 0 {
        body.Port = 2375
    }
    if body.SocketPath == "" && (body.ConnectionType == "socket" || body.ConnectionType == "") {
        body.SocketPath = "/var/run/docker.sock"
    }

    env := &Environment{
        ConnectionType: body.ConnectionType,
        SocketPath:     body.SocketPath,
        Host:           body.Host,
        Port:           body.Port,
        Protocol:       body.Protocol,
        TlsCa:          body.TlsCa,
        TlsCert:        body.TlsCert,
        TlsKey:         body.TlsKey,
        TlsSkipVerify:  body.TlsSkipVerify,
    }

    cli, err := createDockerClient(env)
    if err != nil {
        return c.JSON(fiber.Map{"success": false, "error": err.Error()})
    }
    defer cli.Close()

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    info, err := cli.Info(ctx)
    if err != nil {
        return c.JSON(fiber.Map{"success": false, "error": err.Error()})
    }

    version, err := cli.ServerVersion(ctx)
    serverVersion := ""
    if err == nil {
        serverVersion = version.Version
    }

    return c.JSON(fiber.Map{
        "success": true,
        "info": fiber.Map{
            "serverVersion": serverVersion,
            "containers":    info.Containers,
            "images":        info.Images,
            "name":          info.Name,
            "os":            info.OSType + " " + info.Architecture,
        },
    })
})
```

**Important:** This route MUST be registered BEFORE `protected.Post("/environments/:id/test", ...)` to avoid the `:id` wildcard matching the literal "test".

**Step 2: Compile**

```bash
go build ./...
```

**Step 3: Quick manual test**

```bash
curl -s -X POST http://localhost:3001/api/environments/test \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"connectionType":"socket","socketPath":"/var/run/docker.sock"}' | python3 -m json.tool
```

Expected: `{"success": true, "info": {...}}`

**Step 4: Commit**

```bash
git add backend/main.go
git commit -m "feat(backend): add pre-save connection test endpoint POST /environments/test"
```

---

## Task 5: Add detect-sockets endpoint

**Files:**
- Modify: `backend/main.go` — add route in protected routes section

**Step 1: Add the route**

```go
// GET /api/environments/detect-sockets — find available Docker sockets on the host
protected.Get("/environments/detect-sockets", func(c *fiber.Ctx) error {
    candidates := []struct {
        path string
        name string
    }{
        {"/var/run/docker.sock", "Docker (default)"},
        {"/run/docker.sock", "Docker (alt)"},
        {"/var/run/podman/podman.sock", "Podman"},
        {"/run/podman/podman.sock", "Podman (alt)"},
    }

    var found []fiber.Map
    for _, candidate := range candidates {
        if _, err := os.Stat(candidate.path); err == nil {
            found = append(found, fiber.Map{
                "path": candidate.path,
                "name": candidate.name,
            })
        }
    }

    return c.JSON(fiber.Map{"sockets": found})
})
```

**Step 2: Compile**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add backend/main.go
git commit -m "feat(backend): add detect-sockets endpoint"
```

---

## Task 6: Copy files to raspi-main and rebuild backend

**Step 1: Copy and rebuild**

```bash
cd /path/to/dockerverse-project/dockerverse
scp backend/main.go raspi-main:/tmp/main.go
ssh raspi-main "cp /tmp/main.go /home/pi/dockerverse/backend/main.go && cd /home/pi/dockerverse && docker compose build --no-cache backend 2>&1 | tail -5"
```

**Step 2: Restart and verify**

```bash
ssh raspi-main "cd /home/pi/dockerverse && docker compose up -d backend"
ssh raspi-main "docker logs dockerverse-backend --tail 10"
```

Expected: backend starts, no panic, logs show "Fiber v2..." ready message.

---

## Task 7: Rewrite environments list page as table

**Files:**
- Modify: `frontend/src/routes/settings/environments/+page.svelte`

**Step 1: Read the current file**

Read the full current file (already done in analysis). Note it uses cards with `bg-background-secondary border border-border rounded-xl`.

**Step 2: Replace with table layout**

The new page structure:

```svelte
<script lang="ts">
  import { onMount } from "svelte";
  import {
    Server, Plus, RefreshCw, Wifi, WifiOff, Pencil, Trash2,
    Loader2, Globe, Activity, ShieldCheck, Unplug, Zap,
    Check, XCircle, ImageMinus
  } from "lucide-svelte";
  import { language } from "$lib/stores/docker";
  import { API_BASE } from "$lib/api/docker";
  import EnvironmentModal from "$lib/components/EnvironmentModal.svelte";

  // Types
  interface Environment {
    id: string;
    name: string;
    connectionType: string;
    socketPath: string;
    host: string;
    port: number;
    protocol: string;
    labels: string[];
    status: string;
    dockerVersion: string;
    autoUpdate: boolean;
    vulnScanning: boolean;
    eventTracking: boolean;
    imagePrune: boolean;
    collectMetrics: boolean;
  }

  interface TestResult {
    success: boolean;
    error?: string;
    info?: { serverVersion: string; containers: number; images: number; name: string };
  }

  let environments = $state<Environment[]>([]);
  let loading = $state(true);
  let showModal = $state(false);
  let editingEnv = $state<Environment | null>(null);
  let testResults = $state<Record<string, TestResult | "testing">>({});
  let testingAll = $state(false);
  let pruneStatus = $state<Record<string, "pruning" | "success" | "error" | null>>({});
  let confirmDelete = $state<string | null>(null);
  let confirmPrune = $state<string | null>(null);

  function authHeaders(): Record<string, string> {
    const token = typeof localStorage !== "undefined" ? localStorage.getItem("auth_access_token") : null;
    const h: Record<string, string> = { "Content-Type": "application/json" };
    if (token) h["Authorization"] = `Bearer ${token}`;
    return h;
  }

  async function fetchEnvironments() {
    loading = true;
    try {
      const res = await fetch(`${API_BASE}/api/environments`, { headers: authHeaders() });
      if (res.ok) environments = await res.json();
    } catch (e) { console.error(e); }
    loading = false;
  }

  async function testConnection(id: string) {
    testResults[id] = "testing";
    testResults = { ...testResults };
    try {
      const res = await fetch(`${API_BASE}/api/environments/${id}/test`, {
        method: "POST", headers: authHeaders()
      });
      testResults[id] = await res.json();
    } catch {
      testResults[id] = { success: false, error: "Connection failed" };
    }
    testResults = { ...testResults };
  }

  async function testAll() {
    if (testingAll) return;
    testingAll = true;
    for (const env of environments) {
      await testConnection(env.id);
    }
    testingAll = false;
  }

  async function pruneSystem(id: string) {
    pruneStatus[id] = "pruning";
    pruneStatus = { ...pruneStatus };
    confirmPrune = null;
    try {
      const res = await fetch(`${API_BASE}/api/containers/${id}/prune`, {
        method: "POST", headers: authHeaders()
      });
      pruneStatus[id] = res.ok ? "success" : "error";
    } catch {
      pruneStatus[id] = "error";
    }
    pruneStatus = { ...pruneStatus };
    setTimeout(() => { pruneStatus[id] = null; pruneStatus = { ...pruneStatus }; }, 3000);
  }

  async function deleteEnvironment(id: string) {
    confirmDelete = null;
    try {
      const res = await fetch(`${API_BASE}/api/environments/${id}`, {
        method: "DELETE", headers: authHeaders()
      });
      if (res.ok) await fetchEnvironments();
    } catch (e) { console.error(e); }
  }

  async function handleSave(env: any) {
    const isEdit = editingEnv !== null;
    const url = isEdit ? `${API_BASE}/api/environments/${env.id}` : `${API_BASE}/api/environments`;
    try {
      const res = await fetch(url, {
        method: isEdit ? "PUT" : "POST",
        headers: authHeaders(),
        body: JSON.stringify(env)
      });
      if (res.ok) {
        showModal = false;
        editingEnv = null;
        await fetchEnvironments();
        testAll();
      }
    } catch (e) { console.error(e); }
  }

  function getConnectionLabel(env: Environment): string {
    if (env.connectionType === "socket" || !env.connectionType) {
      return env.socketPath || "/var/run/docker.sock";
    }
    return `${env.host}:${env.port}`;
  }

  onMount(() => {
    fetchEnvironments().then(() => testAll());
  });
</script>

<div class="p-6 space-y-4 max-w-6xl mx-auto">
  <!-- Header -->
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-3">
      <div class="p-2 bg-primary/15 rounded-lg">
        <Server class="w-5 h-5 text-primary" />
      </div>
      <div>
        <h2 class="text-base font-semibold text-foreground">
          {$language === "es" ? "Entornos" : "Environments"}
        </h2>
        <p class="text-xs text-foreground-muted">{environments.length} total</p>
      </div>
    </div>
    <div class="flex items-center gap-2">
      <button
        class="flex items-center gap-1.5 px-3 py-1.5 text-sm bg-background-tertiary text-foreground rounded-lg hover:bg-background-tertiary/80 transition-colors disabled:opacity-50"
        onclick={testAll}
        disabled={testingAll || environments.length === 0}
      >
        {#if testingAll}
          <RefreshCw class="w-4 h-4 animate-spin" />
        {:else}
          <Wifi class="w-4 h-4" />
        {/if}
        {$language === "es" ? "Probar todos" : "Test all"}
      </button>
      <button
        class="flex items-center gap-1.5 px-3 py-1.5 text-sm bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors"
        onclick={() => { editingEnv = null; showModal = true; }}
      >
        <Plus class="w-4 h-4" />
        {$language === "es" ? "Agregar" : "Add environment"}
      </button>
    </div>
  </div>

  <!-- Table -->
  {#if loading && environments.length === 0}
    <div class="flex items-center justify-center py-16">
      <Loader2 class="w-6 h-6 text-primary animate-spin" />
    </div>
  {:else if environments.length === 0}
    <div class="text-center py-16">
      <Server class="w-10 h-10 text-foreground-muted/30 mx-auto mb-3" />
      <p class="text-sm text-foreground-muted">
        {$language === "es" ? "Sin entornos configurados" : "No environments configured"}
      </p>
    </div>
  {:else}
    <div class="border border-border rounded-lg overflow-hidden">
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b border-border bg-background-secondary/50">
            <th class="px-4 py-2.5 text-left text-xs font-semibold text-foreground-muted uppercase tracking-wider w-48">
              {$language === "es" ? "Nombre" : "Name"}
            </th>
            <th class="px-4 py-2.5 text-left text-xs font-semibold text-foreground-muted uppercase tracking-wider">
              {$language === "es" ? "Conexión" : "Connection"}
            </th>
            <th class="px-4 py-2.5 text-left text-xs font-semibold text-foreground-muted uppercase tracking-wider w-32">
              Labels
            </th>
            <th class="px-4 py-2.5 text-left text-xs font-semibold text-foreground-muted uppercase tracking-wider w-28">
              {$language === "es" ? "Funciones" : "Features"}
            </th>
            <th class="px-4 py-2.5 text-left text-xs font-semibold text-foreground-muted uppercase tracking-wider w-36">
              Status
            </th>
            <th class="px-4 py-2.5 text-right text-xs font-semibold text-foreground-muted uppercase tracking-wider w-32">
              {$language === "es" ? "Acciones" : "Actions"}
            </th>
          </tr>
        </thead>
        <tbody>
          {#each environments as env (env.id)}
            {@const result = testResults[env.id]}
            {@const isTesting = result === "testing"}
            {@const ps = pruneStatus[env.id]}
            <tr class="border-b border-border/50 last:border-0 hover:bg-background-secondary/30 transition-colors">
              <!-- Name -->
              <td class="px-4 py-3">
                <div class="flex items-center gap-2">
                  {#if env.connectionType === "socket" || !env.connectionType}
                    <Unplug class="w-3.5 h-3.5 text-accent-cyan shrink-0" />
                  {:else}
                    <Globe class="w-3.5 h-3.5 text-primary shrink-0" />
                  {/if}
                  <span class="font-medium text-foreground truncate">{env.name}</span>
                </div>
              </td>

              <!-- Connection -->
              <td class="px-4 py-3">
                <span class="text-xs text-foreground-muted font-mono">{getConnectionLabel(env)}</span>
              </td>

              <!-- Labels -->
              <td class="px-4 py-3">
                <div class="flex flex-wrap gap-1">
                  {#if env.labels && env.labels.length > 0}
                    {#each env.labels as label}
                      <span class="text-[10px] px-1.5 py-0.5 bg-primary/10 text-primary rounded-full font-medium">
                        {label}
                      </span>
                    {/each}
                  {:else}
                    <span class="text-foreground-muted text-xs">—</span>
                  {/if}
                </div>
              </td>

              <!-- Features -->
              <td class="px-4 py-3">
                <div class="flex items-center gap-1.5">
                  {#if env.autoUpdate}
                    <span title={$language === "es" ? "Auto-actualización" : "Auto-update"}>
                      <RefreshCw class="w-3.5 h-3.5 text-running" />
                    </span>
                  {/if}
                  {#if env.vulnScanning}
                    <span title={$language === "es" ? "Escaneo de vulnerabilidades" : "Vulnerability scanning"}>
                      <ShieldCheck class="w-3.5 h-3.5 text-running" />
                    </span>
                  {/if}
                  {#if env.eventTracking}
                    <span title={$language === "es" ? "Seguimiento de eventos" : "Event tracking"}>
                      <Activity class="w-3.5 h-3.5 text-accent-orange" />
                    </span>
                  {/if}
                  {#if env.imagePrune}
                    <span title={$language === "es" ? "Limpieza de imágenes" : "Image prune"}>
                      <ImageMinus class="w-3.5 h-3.5 text-accent-orange" />
                    </span>
                  {/if}
                  {#if !env.autoUpdate && !env.vulnScanning && !env.eventTracking && !env.imagePrune}
                    <span class="text-foreground-muted text-xs">—</span>
                  {/if}
                </div>
              </td>

              <!-- Status -->
              <td class="px-4 py-3">
                {#if isTesting}
                  <div class="flex items-center gap-1.5 text-foreground-muted text-xs">
                    <RefreshCw class="w-3.5 h-3.5 animate-spin" />
                    <span>Testing...</span>
                  </div>
                {:else if result && result !== "testing"}
                  {#if result.success}
                    <div class="flex items-center gap-1.5 text-running text-xs">
                      <Wifi class="w-3.5 h-3.5" />
                      <span>Connected</span>
                    </div>
                    {#if result.info?.serverVersion}
                      <p class="text-[10px] text-foreground-muted mt-0.5 ml-5">
                        Docker {result.info.serverVersion} · {result.info.containers} containers
                      </p>
                    {/if}
                  {:else}
                    <div class="flex items-center gap-1.5 text-accent-red text-xs" title={result.error}>
                      <WifiOff class="w-3.5 h-3.5" />
                      <span>Failed</span>
                    </div>
                  {/if}
                {:else}
                  <span class="text-xs text-foreground-muted">Not tested</span>
                {/if}
              </td>

              <!-- Actions -->
              <td class="px-4 py-3">
                <div class="flex items-center justify-end gap-1">
                  <!-- Test -->
                  <button
                    class="p-1.5 rounded hover:bg-background-tertiary transition-colors"
                    onclick={() => testConnection(env.id)}
                    disabled={isTesting}
                    title="Test connection"
                  >
                    {#if isTesting}
                      <RefreshCw class="w-3.5 h-3.5 animate-spin text-foreground-muted" />
                    {:else}
                      <Wifi class="w-3.5 h-3.5 text-foreground-muted hover:text-primary" />
                    {/if}
                  </button>

                  <!-- Edit -->
                  <button
                    class="p-1.5 rounded hover:bg-background-tertiary transition-colors"
                    onclick={() => { editingEnv = { ...env }; showModal = true; }}
                    title="Edit"
                  >
                    <Pencil class="w-3.5 h-3.5 text-foreground-muted" />
                  </button>

                  <!-- Prune system (with inline confirm) -->
                  {#if confirmPrune === env.id}
                    <div class="flex items-center gap-1">
                      <button
                        class="p-1.5 rounded hover:bg-running/15 transition-colors"
                        onclick={() => pruneSystem(env.id)}
                        title="Confirm prune"
                        disabled={ps === "pruning"}
                      >
                        <Check class="w-3.5 h-3.5 text-running" />
                      </button>
                      <button
                        class="p-1.5 rounded hover:bg-background-tertiary transition-colors"
                        onclick={() => (confirmPrune = null)}
                      >
                        <XCircle class="w-3.5 h-3.5 text-foreground-muted" />
                      </button>
                    </div>
                  {:else}
                    <button
                      class="p-1.5 rounded hover:bg-background-tertiary transition-colors"
                      onclick={() => (confirmPrune = env.id)}
                      title="Prune system"
                      disabled={ps === "pruning"}
                    >
                      {#if ps === "pruning"}
                        <RefreshCw class="w-3.5 h-3.5 animate-spin text-foreground-muted" />
                      {:else if ps === "success"}
                        <Check class="w-3.5 h-3.5 text-running" />
                      {:else}
                        <ImageMinus class="w-3.5 h-3.5 text-foreground-muted" />
                      {/if}
                    </button>
                  {/if}

                  <!-- Delete (with inline confirm) -->
                  {#if confirmDelete === env.id}
                    <div class="flex items-center gap-1">
                      <button
                        class="p-1.5 rounded hover:bg-accent-red/15 transition-colors"
                        onclick={() => deleteEnvironment(env.id)}
                        title="Confirm delete"
                      >
                        <Check class="w-3.5 h-3.5 text-accent-red" />
                      </button>
                      <button
                        class="p-1.5 rounded hover:bg-background-tertiary transition-colors"
                        onclick={() => (confirmDelete = null)}
                      >
                        <XCircle class="w-3.5 h-3.5 text-foreground-muted" />
                      </button>
                    </div>
                  {:else}
                    <button
                      class="p-1.5 rounded hover:bg-accent-red/15 transition-colors"
                      onclick={() => (confirmDelete = env.id)}
                      title="Delete"
                    >
                      <Trash2 class="w-3.5 h-3.5 text-foreground-muted hover:text-accent-red" />
                    </button>
                  {/if}
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>

{#if showModal}
  <EnvironmentModal
    environment={editingEnv}
    onclose={() => { showModal = false; editingEnv = null; }}
    onsave={handleSave}
  />
{/if}
```

**Step 3: Deploy frontend to raspi-main to verify visually**

```bash
scp frontend/src/routes/settings/environments/+page.svelte raspi-main:/tmp/env-page.svelte
ssh raspi-main "cp /tmp/env-page.svelte /home/pi/dockerverse/frontend/src/routes/settings/environments/+page.svelte"
ssh raspi-main "cd /home/pi/dockerverse && docker compose build --no-cache frontend 2>&1 | tail -5 && docker compose up -d frontend"
```

**Step 4: Check visually at https://docker-connect.nerdslabs.com/settings/environments**

Verify: table renders, rows show correctly, no console errors.

**Step 5: Commit**

```bash
git add frontend/src/routes/settings/environments/+page.svelte
git commit -m "feat(frontend): environments list as flat table rows (no cards)"
```

---

## Task 8: Rewrite EnvironmentModal with 4 tabs

**Files:**
- Modify: `frontend/src/lib/components/EnvironmentModal.svelte`

This is the largest task. The new modal has 4 tabs with full functionality.

**Step 1: Read the current EnvironmentModal.svelte** (already done)

**Step 2: Write the new full modal**

New `EnvironmentData` interface at top of `<script module>`:

```typescript
export interface EnvironmentData {
  id: string;
  name: string;
  connectionType: string;   // "socket" | "tcp" | "tcp+tls"
  socketPath: string;
  host: string;
  port: number;
  protocol: string;
  tlsCa: string;
  tlsCert: string;
  tlsKey: string;
  tlsSkipVerify: boolean;
  labels: string[];
  publicIp: string;
  timezone: string;
  // Updates
  autoUpdate: boolean;
  updateSchedule: string;
  imagePrune: boolean;
  imagePruneMode: string;
  imagePruneCron: string;
  // Monitoring
  eventTracking: boolean;
  vulnScanning: boolean;
  collectMetrics: boolean;
  highlightChanges: boolean;
  // Advanced
  diskWarningEnabled: boolean;
  diskWarningMode: string;
  diskWarningThreshold: number;
}
```

New script section features:
- `activeTab` state: `"general" | "updates" | "monitoring" | "advanced"`
- `testResult` state for pre-save connection test
- `testing` state (spinner on Test button)
- `detectedSockets` state for socket detection
- `labelInput` state for tag input
- `saving` state

General tab logic:
- Connection type: 3 buttons (socket / tcp / tcp+tls)
- When socket: show socketPath + Detect button (calls `GET /api/environments/detect-sockets`)
- When tcp/tcp+tls: show host + port fields
- When tcp+tls: show CA/cert/key textareas + skip-verify toggle
- Label tag input: type + Enter to add, click badge to remove
- Test button: calls `POST /api/environments/test` with current form values

Updates tab:
- Auto-update toggle → when on: show cron input
- Image prune toggle → when on: show mode radio (dangling/all) + cron input

Monitoring tab:
- Event tracking toggle
- Collect metrics toggle
- Vulnerability scanning toggle
- Highlight changes toggle

Advanced tab:
- Timezone: searchable `<select>` (populate with a curated list of ~30 common IANA timezones)
- Disk warning toggle → when on: mode radio + threshold number input

**Step 3: Key TIMEZONES list to include in the component**

```typescript
const TIMEZONES = [
  "UTC", "America/New_York", "America/Chicago", "America/Denver",
  "America/Los_Angeles", "America/Sao_Paulo", "Europe/London",
  "Europe/Paris", "Europe/Berlin", "Europe/Madrid", "Europe/Rome",
  "Europe/Amsterdam", "Europe/Moscow", "Asia/Tokyo", "Asia/Shanghai",
  "Asia/Singapore", "Asia/Dubai", "Asia/Kolkata", "Australia/Sydney",
  "Pacific/Auckland"
];
```

**Step 4: The complete new EnvironmentModal.svelte**

Write the full file to `frontend/src/lib/components/EnvironmentModal.svelte`. The modal should:
1. Open at `activeTab = "general"`
2. On save: call `onsave(form)` — the parent handles the API call
3. Show `testResult` inline below the Test button
4. Disable the ID field when editing

Full implementation (~300 lines) — write in the actual file, not in this plan.

**Step 5: Deploy and test manually**

```bash
scp frontend/src/lib/components/EnvironmentModal.svelte raspi-main:/tmp/EnvironmentModal.svelte
ssh raspi-main "cp /tmp/EnvironmentModal.svelte /home/pi/dockerverse/frontend/src/lib/components/EnvironmentModal.svelte"
ssh raspi-main "cd /home/pi/dockerverse && docker compose build --no-cache frontend 2>&1 | tail -5 && docker compose up -d frontend"
```

Manual test checklist:
- [ ] Add new environment with socket type → saves and appears in table
- [ ] Add new environment with TCP type → saves with host:port
- [ ] Add new environment with TCP+TLS → TLS fields appear
- [ ] Test connection button works before saving
- [ ] Detect sockets button shows available sockets
- [ ] Labels: type + Enter adds tag, click removes tag
- [ ] Updates tab: toggles save correctly
- [ ] Monitoring tab: toggles save correctly
- [ ] Advanced tab: timezone and disk warning save correctly
- [ ] Edit existing environment → form pre-fills all fields correctly
- [ ] Delete environment → confirm inline, then removes

**Step 6: Commit**

```bash
git add frontend/src/lib/components/EnvironmentModal.svelte
git commit -m "feat(frontend): full 4-tab EnvironmentModal with TLS, labels, socket detect, pre-save test"
```

---

## Task 9: End-to-end test and bug fixes

**Step 1: Test full add/edit flow**

Navigate to https://docker-connect.nerdslabs.com/settings/environments

Test scenarios:
1. Add environment (socket) → verify in table
2. Add environment (TCP) → verify connection test works
3. Edit environment → change labels, timezone, update settings
4. Delete environment → confirm works
5. Test All → verify all rows update sequentially

**Step 2: Check browser console for errors**

Open DevTools → Console. No errors should appear.

**Step 3: Check backend logs**

```bash
ssh raspi-main "docker logs dockerverse-backend --tail 30"
```

No panics or 500 errors.

**Step 4: Fix any bugs found**

For each bug: identify root cause, fix, deploy, re-test.

**Step 5: Final commit**

```bash
git add -A
git commit -m "fix(environments): post-QA bug fixes and adjustments"
```

---

## Task 10: Push to git

**Step 1: Ensure all commits are clean**

```bash
git log --oneline -10
```

**Step 2: Push**

```bash
git push origin main
```

---

## Deferred Plan Summary (to be planned separately)

### Plan A: Hawser Agent (edge mode)
Requirements: agent binary on remote host, bidirectional WebSocket tunnel, token-based auth, new connection type "hawser-edge" in DockerVerse.

### Plan B: Notifications per-environment
Requirements: notification channels model (SMTP/webhook), per-env subscriptions with event type arrays, dispatch hook in Docker event watcher, UI for managing channels + per-env subscriptions.

### Plan C: Scanner management UI
Requirements: pull scanner image on demand with SSE progress, check scanner version vs registry, remove scanner image, support both Grype and Trivy per-environment.
