# Environments Redesign — Design Document
Date: 2026-03-07

## Overview

Redesign the DockerVerse Environments section to match dockhand's linear table-based UI and feature set. Replaces the current card-grid layout with a flat table (rows, no boxes) and expands the modal from a basic 4-tab form into a full-featured environment configuration panel.

## Reference

- Dockhand source: https://github.com/Finsys/dockhand
- Key files analyzed: `src/routes/settings/environments/EnvironmentsTab.svelte`, `EnvironmentModal.svelte` (2596 lines)
- DockerVerse current: `frontend/src/routes/settings/environments/+page.svelte`, `frontend/src/lib/components/EnvironmentModal.svelte`

---

## Part 1 — List UI (Table rows, no cards)

### Layout

Header: title + badge "N total" + buttons [Test all] [Add environment]

Table columns:
| Column | Content |
|--------|---------|
| Name | Connection type icon + environment name (bold) |
| Connection | Socket path or host:port |
| Labels | Colored badge tags |
| Features | Icon row: AutoUpdate, VulnScanning, EventTracking, ImagePrune |
| Status | Wifi connected / WifiOff failed + Docker version inline |
| Actions | Test, Edit, Prune system (with confirm), Delete (with confirm) |

### Connection type icons
- Socket → `Unplug` icon (cyan)
- TCP/TCP+TLS → `Globe` icon (blue)

### Feature icons (with tooltips)
- AutoUpdate → `RefreshCw` (green)
- VulnScanning → `ShieldCheck` (green)
- EventTracking → `Activity` (amber)
- ImagePrune → `Trash2` (amber)

### Status behavior
- Auto-tests all environments on mount (sequential, not parallel)
- Per-row test button with spinner while testing
- Connected: green Wifi + "Docker X.Y.Z · N containers"
- Failed: red WifiOff + error text on hover

### Actions
- Test: individual connection test
- Edit: opens modal in edit mode
- Prune system: confirm popover → POST /api/prune/all?host={id}
- Delete: confirm popover → DELETE /api/environments/{id}

---

## Part 2 — Environment Modal (4 tabs)

### Tab 1 — General

Fields:
- **ID** (text, disabled when editing) — unique machine-readable key
- **Name** (text, required) — display name
- **Labels** (tag input) — colored tags, add by typing + Enter, click to remove; stored as `[]string` in backend
- **Connection type** (3-button selector):
  - `socket` — Unix socket
  - `tcp` — Direct TCP (HTTP)
  - `tcp+tls` — Direct TCP with TLS (HTTPS)
- **Socket path** (shown when socket): text input, default `/var/run/docker.sock` + "Detect" button
- **Host** (shown when tcp/tls): hostname/IP input
- **Port** (shown when tcp/tls): number input, default 2375 (tcp) / 2376 (tls)
- **TLS fields** (shown when tcp+tls):
  - CA Certificate (textarea)
  - Client Certificate (textarea)
  - Client Key (textarea)
  - Skip verify (toggle)
- **Test connection** button (pre-save, uses POST /api/environments/test)
- **Public IP** (optional text, auto-filled from host)

### Tab 2 — Updates

- **Update check** toggle
  - When enabled: Cron expression input (text, with human-readable hint)
  - **Auto-update** toggle (sub-option when update check enabled)
- **Image prune** toggle
  - When enabled: Mode selector (dangling / all) + Cron expression input

### Tab 3 — Monitoring

- **Event tracking** toggle — log container start/stop/die events
- **Collect metrics** toggle (new field) — CPU/memory polling
- **Vulnerability scanning** toggle — Trivy scan on image pull/update
- **Highlight changes** toggle (new field) — visual indicator on changed containers

### Tab 4 — Advanced

- **Timezone** — searchable dropdown (IANA timezone names)
- **Disk warning** toggle
  - When enabled: Mode (percentage / absolute GB) + threshold input

---

## Part 3 — Backend Changes (Go)

### Environment struct additions

```go
type Environment struct {
    // existing fields ...
    ID             string   `json:"id"`
    Name           string   `json:"name"`
    ConnectionType string   `json:"connectionType"` // "socket", "tcp", "tcp+tls"
    Address        string   `json:"address"`
    Protocol       string   `json:"protocol"`
    IsLocal        bool     `json:"isLocal"`
    Status         string   `json:"status"`
    DockerVersion  string   `json:"dockerVersion"`

    // new: split socket path from address
    SocketPath     string   `json:"socketPath"`

    // new: TLS
    TlsCa          string   `json:"tlsCa"`
    TlsCert        string   `json:"tlsCert"`
    TlsKey         string   `json:"tlsKey"`
    TlsSkipVerify  bool     `json:"tlsSkipVerify"`

    // changed: labels from string to []string
    Labels         []string `json:"labels"`

    // new: metadata
    PublicIP       string   `json:"publicIp"`
    Timezone       string   `json:"timezone"`

    // existing update settings (already present)
    AutoUpdate     bool     `json:"autoUpdate"`
    UpdateSchedule string   `json:"updateSchedule"`
    ImagePrune     bool     `json:"imagePrune"`
    ImagePruneMode string   `json:"imagePruneMode"` // "dangling" | "all"
    ImagePruneCron string   `json:"imagePruneCron"`

    // existing feature flags
    EventTracking  bool     `json:"eventTracking"`
    VulnScanning   bool     `json:"vulnScanning"`

    // new feature flags
    CollectMetrics    bool `json:"collectMetrics"`
    HighlightChanges  bool `json:"highlightChanges"`

    // new disk warning
    DiskWarningEnabled   bool    `json:"diskWarningEnabled"`
    DiskWarningMode      string  `json:"diskWarningMode"`      // "percentage" | "absolute"
    DiskWarningThreshold float64 `json:"diskWarningThreshold"` // % or GB
}
```

### New API endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | /api/environments/test | Test connection pre-save (body: same fields as create) |
| GET | /api/environments/detect-sockets | Find available Docker sockets on host |

### Docker client creation

Update `GetClient()` to use TLS transport when `TlsCa`/`TlsCert`/`TlsKey` are set:
- Use `tls.X509KeyPair` + `x509.CertPool`
- `TlsSkipVerify` → `InsecureSkipVerify: true`

### Labels migration

`EnvironmentStore.load()` must migrate old `string` labels field to `[]string`:
- If raw JSON has `"labels": "prod, arm64"` → split on `,` and trim
- Write back as array on next save

---

## Part 4 — Settings Section (same session)

After environments, redesign the Settings main page and sub-pages to use the same linear, non-card approach. Specific sections to redesign:

- `/settings/+page.svelte` — already linear (links list), keep as-is
- `/settings/notifications/+page.svelte` — replace card layout with flat rows
- `/settings/appearance/+page.svelte` — replace card layout with flat rows
- `/settings/security/+page.svelte` — flat rows

---

## Deferred Features (separate plan)

### A. Hawser agent (edge mode)
Connection type for outbound agents (agent runs on remote host, connects back to DockerVerse). Requires:
- Hawser token generation (crypto-secure, store in backend)
- WebSocket or SSE tunnel between agent and DockerVerse
- New connection type in Docker client: route through local WebSocket proxy

### B. Notifications per-environment
Per-environment notification channels with configurable event types. Requires:
- Notification channels data model (SMTP / webhook)
- Per-env notification subscriptions (envId + channelId + event types[])
- Event type constants (container_started, container_stopped, update_success, vuln_critical, etc.)
- Notification dispatch in Docker event watcher

### C. Scanner management UI
Per-environment Grype/Trivy management panel. Requires:
- Pull scanner image on demand with progress stream
- Check scanner version vs latest tag
- Remove scanner image (free disk space)
- Support both Grype and Trivy (currently only Trivy in DockerVerse)

---

## Success Criteria

- [ ] Environments page shows flat table (no cards)
- [ ] Each row shows: name, connection, labels (colored), features (icons), status, actions
- [ ] Test All works sequentially with per-row spinners
- [ ] Add/Edit modal has 4 tabs with all fields functional
- [ ] Socket path configurable, Detect button finds available sockets
- [ ] TCP+TLS connection type works end-to-end (backend creates TLS client)
- [ ] Labels stored as array, colored tags in UI
- [ ] Pre-save connection test works
- [ ] Update check cron, auto-update, image prune with mode+cron all save/load correctly
- [ ] Monitoring toggles (event tracking, metrics, vuln scanning, highlight) all persist
- [ ] Advanced: timezone + disk warning save/load correctly
- [ ] Prune system action with confirm works
- [ ] No regression on existing environments (migration handles old labels format)
