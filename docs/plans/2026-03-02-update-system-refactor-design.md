# Design: Update System Refactor + Vulnerability Scanner

**Date:** 2026-03-02
**Status:** Approved
**Version target:** v2.5.0

---

## Problem Statement

1. **False positive updates**: The app shows 2 containers needing updates when all are actually up-to-date.
2. **No vulnerability scanning**: No CVE scanning before applying container updates.
3. **No scan history**: No record of past vulnerability assessments.

---

## Root Cause Analysis

### Bug 1 — Multi-arch digest mismatch (primary bug)

The `crane.Digest()` call returns the **manifest list digest** (multi-platform), but the locally pulled image on arm64 Raspberry Pi stores the **platform-specific digest** in `RepoDigests`. These are different SHA256 values even when the image is fully up-to-date.

```
Local  (arm64): nginx@sha256:PLATFORM_SPECIFIC_ABC123...
Remote (crane): sha256:MANIFEST_LIST_XYZ789...
```

`!strings.Contains(local, remote)` → always `true` → permanent false positive.

**Fix:** Use `crane.WithPlatform(&v1.Platform{OS: "linux", Architecture: "arm64"})` when calling `crane.Digest()`.

### Bug 2 — Cache key tied to container ID

The update cache is keyed by `container.ID` which changes every time a container is recreated during an update. After an update:
- Old ID cache entry is deleted
- New container (new ID) has no cache entry
- Next check creates a fresh entry, but may hit Bug 1 again

**Fix:** Use `image:hostID` as the stable cache key instead of container ID.

### Bug 3 — Race condition in background checker

The 15-minute background checker runs concurrently with manual updates, creating brief windows where old and new containers coexist in Docker's container list.

**Fix:** Add a per-image mutex/lock during update operations to prevent concurrent checks.

---

## Solution Design

### Architecture Overview

```
DockerVerse Backend (Go)
    │
    ├── checkContainerUpdate() — FIXED
    │       ├── Cache key: image:hostID (stable)
    │       ├── crane.Digest with arm64 platform
    │       └── Normalizes digest comparison
    │
    ├── updateContainerImage() — REFACTORED
    │       ├── Pull to temp tag
    │       ├── Run vulnerability scan (Trivy + Grype)
    │       ├── Evaluate blocking criteria
    │       ├── Force override if admin requested
    │       └── Recreate container with full config
    │
    ├── SSE endpoint: /api/updates/:hostId/:containerId/stream
    │       └── Real-time events during update + scan
    │
    └── New endpoints:
            ├── GET  /api/scans                    — scan history
            ├── GET  /api/scans/:imageId            — scan detail
            └── POST /api/containers/:h/:c/scan     — manual scan
```

### Update Process Flow

```
1. Detect update (fixed digest comparison)
2. Pull new image to temp tag (image:dockverse-update-{ts})
3. Run Trivy scan on temp image
   └── (parallel) Run Grype scan on temp image
4. Parse results → aggregate by severity
5. Evaluate blocking criteria:
   - never          → always proceed
   - any            → block if vulnerabilities > 0
   - critical_high  → block if critical ≥ 1 OR high ≥ 1
   - critical        → block if critical ≥ 1
   - more_than_current → block if new > cached baseline
6. If BLOCKED:
   a. Save scan results to SQLite
   b. Remove temp image
   c. Send notification
   d. Return blocked status (with override token for admin)
7. If APPROVED (or force_override=true):
   a. Re-tag temp image to original tag
   b. Stop old container
   c. Remove old container
   d. Create + start new container (full config passthrough)
   e. Save scan results to SQLite
   f. Clear update cache
   g. Send success notification
```

### Vulnerability Scanner

Runs as ephemeral Docker containers — nothing installed on host:

```go
// Trivy
docker run --rm \
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  -v dockverse-trivy-db:/cache/trivy \
  -e TRIVY_CACHE_DIR=/cache/trivy \
  aquasec/trivy:latest \
  image --format json {imageName}

// Grype
docker run --rm \
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  -v dockverse-grype-db:/cache/grype \
  -e GRYPE_DB_CACHE_DIR=/cache/grype \
  anchore/grype:latest \
  -o json -v {imageName}
```

**Platform support:** ARM64 native images available for both Trivy and Grype.

### Blocking Criteria

| Criterion | Description | Block condition |
|-----------|-------------|-----------------|
| `never` | Never block updates | Never |
| `any` | Block on any finding | total > 0 |
| `critical_high` | Block on serious issues | critical ≥ 1 or high ≥ 1 |
| `critical` | Block only on critical | critical ≥ 1 |
| `more_than_current` | Comparative blocking | new_critical > old_critical or new_high > old_high |

**Admin override:** Admin users can force an update even when blocked. The override is logged to the audit trail with timestamp, user, and reason.

### Data Model (SQLite)

```sql
CREATE TABLE scan_results (
  id TEXT PRIMARY KEY,
  container_id TEXT,
  container_name TEXT,
  image_name TEXT,
  image_id TEXT,
  host_id TEXT,
  scanner TEXT,           -- 'trivy' | 'grype' | 'both'
  scanned_at TEXT,        -- ISO8601
  scan_duration_ms INTEGER,
  critical_count INTEGER DEFAULT 0,
  high_count INTEGER DEFAULT 0,
  medium_count INTEGER DEFAULT 0,
  low_count INTEGER DEFAULT 0,
  negligible_count INTEGER DEFAULT 0,
  unknown_count INTEGER DEFAULT 0,
  vulnerabilities TEXT,   -- JSON array
  blocked INTEGER DEFAULT 0,
  block_reason TEXT,
  force_override INTEGER DEFAULT 0,
  triggered_by TEXT,      -- 'manual' | 'auto' | 'schedule'
  created_at TEXT
);
```

### API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/updates` | All containers with update status (fixed) |
| POST | `/api/updates/:hostId/:containerId/check` | Force re-check single container |
| POST | `/api/containers/:hostId/:containerId/update` | Trigger update (+ scan) |
| POST | `/api/containers/:hostId/:containerId/update?force=true` | Force update (admin override) |
| GET | `/api/containers/:hostId/:containerId/scan` | Get latest scan for container |
| POST | `/api/containers/:hostId/:containerId/scan` | Trigger manual scan |
| GET | `/api/scans` | Scan history (all containers) |
| GET | `/api/scans/:id` | Scan detail with full CVE list |
| GET | `/api/scans/summary` | Aggregated severity summary |
| SSE | `/api/updates/:hostId/:containerId/stream` | Real-time update/scan progress |

### SSE Event Format

```json
{"event": "pulling", "data": {"stage": "pulling", "message": "Pulling nginx:latest...", "progress": 20}}
{"event": "scanning", "data": {"stage": "scanning", "scanner": "trivy", "message": "Scanning with Trivy...", "progress": 45}}
{"event": "scan_output", "data": {"scanner": "trivy", "output": "[trivy] Found 2 critical..."}}
{"event": "result", "data": {"stage": "blocked", "critical": 2, "high": 3, "reason": "2 critical CVEs found"}}
{"event": "result", "data": {"stage": "updated", "newDigest": "sha256:abc...", "scanSummary": {...}}}
{"event": "error", "data": {"message": "Scanner pull failed", "error": "..."}}
```

---

## UI/UX Changes

### Container Cards

- Show vulnerability badge if last scan has findings: `🛡️ 2C 3H` (C=critical, H=high)
- Update badge becomes orange/red if vulnerable update available
- Quick "Scan" button in card dropdown

### Update Modal

- Scanner selection: Trivy | Grype | Both
- Vulnerability criteria selection (per-update)
- Real-time streaming progress via SSE
- Full CVE list displayed when scan completes
- "Force Update" button visible to admins when blocked

### New Security Page (`/security`)

- Table of all past scans
- Filter by: host | image | scanner | severity | date range
- Click row → full CVE detail modal
- Summary cards: total scans, images with criticals, last scan date

---

## Implementation Sequence

1. **Fix update detection bugs** (backend)
   - Multi-arch digest comparison
   - Cache key refactor
   - Concurrent check protection

2. **Vulnerability scanner backend** (backend)
   - Scanner execution engine (Trivy + Grype)
   - Result parsing and storage (SQLite migration)
   - Scan-integrated update flow

3. **SSE streaming** (backend + frontend)
   - SSE endpoint in Go Fiber
   - Frontend client with EventSource

4. **Frontend updates** (frontend)
   - Update Modal with scan progress
   - Container card vulnerability badges
   - Security/scan history page

5. **Documentation consolidation**
   - Move all docs to `docs/` directory
   - Update README

6. **Deploy + test on raspi-main**

7. **Git commit** (no credentials, no keys)

---

## Security Constraints

- No credentials committed to git
- Scanner containers have read-only Docker socket access
- Force override logged with user + timestamp + reason
- Scan results stored locally (SQLite), never sent externally
- Scanner images pulled from official registries only
