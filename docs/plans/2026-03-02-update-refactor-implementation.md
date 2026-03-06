# Update System Refactor + Vulnerability Scanner — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix false-positive update detection bugs, add Trivy+Grype vulnerability scanning with configurable blocking, SSE streaming, and a Security history page.

**Architecture:** Fix multi-arch digest comparison in Go backend → add ephemeral-container scanner engine → integrate scan into update flow → stream progress via SSE → update Svelte frontend with scan results UI.

**Tech Stack:** Go 1.23, Fiber v2, `go-containerregistry/crane`, Docker SDK, SQLite (via `database/sql`), SvelteKit 5, Svelte 5, TailwindCSS, TypeScript.

---

## Task 1: Fix Multi-arch Digest Bug

**Files:**
- Modify: `backend/main.go:4519-4544` (`checkContainerUpdate` function)

**Context:** `crane.Digest()` returns the manifest-list (multi-arch) SHA, but the locally pulled arm64 image stores a platform-specific SHA in `RepoDigests`. They differ even when the image is current → permanent false positive.

Also need to add the `v1` import from `go-containerregistry`.

**Step 1: Add required import to main.go**

Find the import block at line 35 and add:
```go
"github.com/google/go-containerregistry/pkg/crane"
v1 "github.com/opencontainers/image-spec/specs-go/v1"
```

Wait — `crane.WithPlatform` uses its own platform type. Check:
```bash
grep -r "WithPlatform\|Platform{" $(go env GOPATH)/pkg/mod/github.com/google/go-containerregistry*/pkg/crane/ 2>/dev/null | head -5
```

The crane package exposes `crane.WithPlatform` that takes `*v1.Platform` from `github.com/google/go-containerregistry/pkg/v1`. Add this import:
```go
v1 "github.com/google/go-containerregistry/pkg/v1"
```

**Step 2: Detect host architecture dynamically**

In `checkContainerUpdate`, before the `crane.Digest` call (line 4520), add architecture detection based on the remote Docker host. For now use `runtime.GOARCH` which will be `arm64` when backend runs on Raspberry Pi. Add `"runtime"` to imports.

**Step 3: Replace the digest check (lines 4519-4544)**

Replace this block:
```go
// Compare local digest with remote registry digest using crane
remoteDigest, err := crane.Digest(container.Image)
if err != nil {
    // Registry unreachable, auth failed, or image not found - not an error
    log.Printf("Could not check registry for %s: %v", container.Image, err)
    update.HasUpdate = false
    update.LatestDigest = ""
} else {
    // Compare: currentDigest is "image@sha256:xxx", remoteDigest is "sha256:yyy"
    hasUpdate := currentDigest != "" && !strings.Contains(currentDigest, remoteDigest)
    update.HasUpdate = hasUpdate
    update.LatestDigest = remoteDigest
    if hasUpdate {
        // Show short digest as "latest tag" since we can't resolve the actual tag
        if len(remoteDigest) > 15 {
            update.LatestTag = remoteDigest[:15] + "..."
        } else {
            update.LatestTag = remoteDigest
        }
    }
}

// Cache the result
updateCacheMu.Lock()
updateCache[container.ID] = update
updateCacheMu.Unlock()
```

With this corrected version:
```go
// Detect architecture for platform-specific digest comparison
arch := runtime.GOARCH // "arm64" on Raspberry Pi, "amd64" on x86
platform := &v1.Platform{OS: "linux", Architecture: arch}

// Get platform-specific digest to avoid multi-arch manifest false positives
remoteDigest, err := crane.Digest(container.Image, crane.WithPlatform(platform))
if err != nil {
    log.Printf("Could not check registry for %s (arch=%s): %v", container.Image, arch, err)
    update.HasUpdate = false
    update.LatestDigest = ""
} else {
    // Extract the sha256 portion from currentDigest for fair comparison
    // currentDigest format: "registry/image@sha256:ABCDEF..."
    // remoteDigest format:  "sha256:ABCDEF..."
    localSHA := ""
    if idx := strings.Index(currentDigest, "@sha256:"); idx >= 0 {
        localSHA = currentDigest[idx+1:] // "sha256:ABCDEF..."
    }
    hasUpdate := localSHA != "" && localSHA != remoteDigest
    update.HasUpdate = hasUpdate
    update.LatestDigest = remoteDigest
    log.Printf("[UpdateCheck] %s: localSHA=%s remoteDigest=%s hasUpdate=%v",
        container.Image, localSHA[:min(20, len(localSHA))], remoteDigest[:min(20, len(remoteDigest))], hasUpdate)
    if hasUpdate {
        if len(remoteDigest) > 19 {
            update.LatestTag = remoteDigest[:19] + "..."
        } else {
            update.LatestTag = remoteDigest
        }
    }
}

// Cache the result — use stable key (image:hostID) not container.ID which changes on recreate
cacheKey := container.Image + ":" + container.HostID
updateCacheMu.Lock()
updateCache[cacheKey] = update
updateCacheMu.Unlock()
```

Add `min` helper if not present (Go 1.21+: built-in `min` exists, so should be fine).

**Step 4: Fix all other cache references to use the new key**

Find every place that reads/writes `updateCache[container.ID]` and replace with `updateCache[cacheKey]` where `cacheKey = container.Image + ":" + container.HostID`.

Also fix the `checkSingleImageUpdate` function at line 4435 — the forced-check endpoint bypasses cache but still calls `checkContainerUpdate` which will use the new key correctly. Just make sure the delete in the update endpoint also uses the new key.

At line 3738, change:
```go
// OLD
delete(updateCache, containerID)
```
To:
```go
// NEW - need to get the image name from the container first, or clear all matching
// Simple approach: use hostID:containerID pattern and scan cache
for k := range updateCache {
    if strings.HasSuffix(k, ":"+hostID) {
        // We need the image name - just clear all for this host after update
    }
}
```

Actually, the cleanest approach: after `updateContainerImage` succeeds, pass the image name back so we can clear the right cache key. Alternatively, clear the entire host's cache entries. For simplicity:

```go
// Clear update cache for this host after any container update
updateCacheMu.Lock()
for k := range updateCache {
    if strings.HasSuffix(k, ":"+hostID) {
        delete(updateCache, k)
    }
}
updateCacheMu.Unlock()
```

**Step 5: Add `"runtime"` to imports**

In the import block at line 3-41, add `"runtime"` to the standard library imports.

Also add the v1 platform import:
```go
v1 "github.com/google/go-containerregistry/pkg/v1"
```

**Step 6: Build and verify compilation**

```bash
cd /Users/vcolmenares/Documents/Laboratories/Antigravity/skills/dockerverse-project/dockerverse/backend
go build ./...
```
Expected: no errors.

**Step 7: Verify fix by checking what containers were falsely flagged**

After build:
```bash
# Restart the backend and check logs
docker compose restart backend
docker compose logs -f backend | grep -i "UpdateCheck\|hasUpdate"
```

Expected: containers that were always showing as "needs update" should now show `hasUpdate=false`.

**Step 8: Commit**
```bash
cd /Users/vcolmenares/Documents/Laboratories/Antigravity/skills/dockerverse-project/dockerverse
git add backend/main.go
git commit -m "fix: correct multi-arch digest comparison in update detection

- Use crane.WithPlatform(arm64) to get platform-specific digest
- Extract sha256 portion from local RepoDigests for accurate comparison
- Change cache key from containerID (unstable) to image:hostID (stable)
- Clear host cache entries after successful container update
- Add debug logging for digest comparison

Fixes false-positive updates showing for arm64 containers"
```

---

## Task 2: Add SQLite Scan Results Table

**Files:**
- Create: `backend/models/scan.go`
- Modify: `backend/main.go` (DB init section)

**Context:** Need persistent storage for scan results to support the scan history page and `more_than_current` blocking criterion. Main.go already uses `database/sql` with `modernc.org/sqlite` (check) or file-based store.

**Step 1: Check how main.go stores data**

```bash
grep -n "database\|sqlite\|data/\|\.db\b" backend/main.go | head -20
```

If the project uses a JSON file store or sqlite, check the pattern and follow it.

**Step 2: Create scan.go model**

```go
// backend/models/scan.go
package models

import (
    "database/sql"
    "encoding/json"
    "time"
)

type Vulnerability struct {
    ID           string `json:"id"`
    Severity     string `json:"severity"`   // critical, high, medium, low, negligible, unknown
    Package      string `json:"package"`
    Version      string `json:"version"`
    FixedVersion string `json:"fixedVersion,omitempty"`
    Description  string `json:"description,omitempty"`
    Link         string `json:"link,omitempty"`
    Scanner      string `json:"scanner"`    // trivy | grype
}

type ScanSummary struct {
    Critical   int `json:"critical"`
    High       int `json:"high"`
    Medium     int `json:"medium"`
    Low        int `json:"low"`
    Negligible int `json:"negligible"`
    Unknown    int `json:"unknown"`
}

type ScanResult struct {
    ID              string          `json:"id"`
    ContainerID     string          `json:"containerId"`
    ContainerName   string          `json:"containerName"`
    ImageName       string          `json:"imageName"`
    ImageID         string          `json:"imageId"`
    HostID          string          `json:"hostId"`
    Scanner         string          `json:"scanner"`         // trivy | grype | both
    ScannedAt       time.Time       `json:"scannedAt"`
    ScanDurationMs  int64           `json:"scanDurationMs"`
    Summary         ScanSummary     `json:"summary"`
    Vulnerabilities []Vulnerability `json:"vulnerabilities"`
    Blocked         bool            `json:"blocked"`
    BlockReason     string          `json:"blockReason,omitempty"`
    ForceOverride   bool            `json:"forceOverride"`
    TriggeredBy     string          `json:"triggeredBy"` // manual | auto | schedule
}

type ScanStore struct {
    db *sql.DB
}

func NewScanStore(db *sql.DB) (*ScanStore, error) {
    _, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS scan_results (
            id TEXT PRIMARY KEY,
            container_id TEXT,
            container_name TEXT,
            image_name TEXT NOT NULL,
            image_id TEXT,
            host_id TEXT,
            scanner TEXT NOT NULL,
            scanned_at TEXT NOT NULL,
            scan_duration_ms INTEGER DEFAULT 0,
            critical_count INTEGER DEFAULT 0,
            high_count INTEGER DEFAULT 0,
            medium_count INTEGER DEFAULT 0,
            low_count INTEGER DEFAULT 0,
            negligible_count INTEGER DEFAULT 0,
            unknown_count INTEGER DEFAULT 0,
            vulnerabilities TEXT DEFAULT '[]',
            blocked INTEGER DEFAULT 0,
            block_reason TEXT DEFAULT '',
            force_override INTEGER DEFAULT 0,
            triggered_by TEXT DEFAULT 'manual',
            created_at TEXT NOT NULL
        );
        CREATE INDEX IF NOT EXISTS idx_scan_image ON scan_results(image_name);
        CREATE INDEX IF NOT EXISTS idx_scan_host ON scan_results(host_id);
        CREATE INDEX IF NOT EXISTS idx_scan_date ON scan_results(scanned_at DESC);
    `)
    return &ScanStore{db: db}, err
}

func (s *ScanStore) Save(result *ScanResult) error {
    vulnJSON, _ := json.Marshal(result.Vulnerabilities)
    _, err := s.db.Exec(`
        INSERT OR REPLACE INTO scan_results
        (id, container_id, container_name, image_name, image_id, host_id,
         scanner, scanned_at, scan_duration_ms,
         critical_count, high_count, medium_count, low_count, negligible_count, unknown_count,
         vulnerabilities, blocked, block_reason, force_override, triggered_by, created_at)
        VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
    `,
        result.ID, result.ContainerID, result.ContainerName, result.ImageName, result.ImageID, result.HostID,
        result.Scanner, result.ScannedAt.Format(time.RFC3339), result.ScanDurationMs,
        result.Summary.Critical, result.Summary.High, result.Summary.Medium,
        result.Summary.Low, result.Summary.Negligible, result.Summary.Unknown,
        string(vulnJSON), boolToInt(result.Blocked), result.BlockReason,
        boolToInt(result.ForceOverride), result.TriggeredBy, time.Now().Format(time.RFC3339),
    )
    return err
}

func (s *ScanStore) List(limit int) ([]ScanResult, error) {
    rows, err := s.db.Query(`
        SELECT id, container_id, container_name, image_name, image_id, host_id,
               scanner, scanned_at, scan_duration_ms,
               critical_count, high_count, medium_count, low_count, negligible_count, unknown_count,
               vulnerabilities, blocked, block_reason, force_override, triggered_by
        FROM scan_results ORDER BY scanned_at DESC LIMIT ?
    `, limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    return scanRows(rows)
}

func (s *ScanStore) GetLatestForImage(imageName string) (*ScanResult, error) {
    rows, err := s.db.Query(`
        SELECT id, container_id, container_name, image_name, image_id, host_id,
               scanner, scanned_at, scan_duration_ms,
               critical_count, high_count, medium_count, low_count, negligible_count, unknown_count,
               vulnerabilities, blocked, block_reason, force_override, triggered_by
        FROM scan_results WHERE image_name = ? ORDER BY scanned_at DESC LIMIT 1
    `, imageName)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    results, err := scanRows(rows)
    if err != nil || len(results) == 0 {
        return nil, err
    }
    return &results[0], nil
}

func scanRows(rows *sql.Rows) ([]ScanResult, error) {
    var results []ScanResult
    for rows.Next() {
        var r ScanResult
        var scannedAt string
        var vulnJSON string
        var blocked, forceOverride int
        err := rows.Scan(
            &r.ID, &r.ContainerID, &r.ContainerName, &r.ImageName, &r.ImageID, &r.HostID,
            &r.Scanner, &scannedAt, &r.ScanDurationMs,
            &r.Summary.Critical, &r.Summary.High, &r.Summary.Medium,
            &r.Summary.Low, &r.Summary.Negligible, &r.Summary.Unknown,
            &vulnJSON, &blocked, &r.BlockReason, &forceOverride, &r.TriggeredBy,
        )
        if err != nil {
            return nil, err
        }
        r.ScannedAt, _ = time.Parse(time.RFC3339, scannedAt)
        r.Blocked = blocked == 1
        r.ForceOverride = forceOverride == 1
        json.Unmarshal([]byte(vulnJSON), &r.Vulnerabilities)
        results = append(results, r)
    }
    return results, rows.Err()
}

func boolToInt(b bool) int {
    if b { return 1 }
    return 0
}
```

**Step 3: Check how main.go initializes storage**

```bash
grep -n "sql\|sqlite\|\.db\|NewDB\|initDB\|data/" backend/main.go | head -20
```

If there's no SQL database yet, add sqlite initialization. Otherwise extend existing pattern.

**Step 4: Initialize ScanStore in main.go**

Find the main() function startup section and add:
```go
// Near the data directory setup
scanStore, err := models.NewScanStore(db)  // where db is the *sql.DB
if err != nil {
    log.Fatalf("Failed to initialize scan store: %v", err)
}
```

If no SQL database exists yet, add sqlite3 setup:
```go
import "database/sql"
import _ "modernc.org/sqlite"  // pure Go sqlite3

db, err := sql.Open("sqlite", filepath.Join(dataDir, "dockerverse.db"))
if err != nil {
    log.Fatalf("Failed to open database: %v", err)
}
```

And add to go.mod:
```bash
cd backend && go get modernc.org/sqlite
```

**Step 5: Build**
```bash
cd backend && go build ./...
```

**Step 6: Commit**
```bash
git add backend/models/scan.go backend/main.go backend/go.mod backend/go.sum
git commit -m "feat: add scan results SQLite storage model"
```

---

## Task 3: Vulnerability Scanner Engine

**Files:**
- Create: `backend/models/scanner.go`

**Context:** Runs Trivy or Grype as ephemeral Docker containers. Mounts the Docker socket (read-only) and a named volume for the scanner DB cache. Parses JSON output from each scanner.

**Step 1: Create scanner.go**

```go
// backend/models/scanner.go
package models

import (
    "bufio"
    "context"
    "encoding/json"
    "fmt"
    "log"
    "strings"
    "time"

    "github.com/docker/docker/api/types/container"
    "github.com/docker/docker/api/types/filters"
    dockerimage "github.com/docker/docker/api/types/image"
    "github.com/docker/docker/api/types/mount"
    "github.com/docker/docker/client"
    "github.com/docker/docker/pkg/stdcopy"
    "bytes"
    "io"
)

const (
    trivyImage    = "aquasec/trivy:latest"
    grypeImage    = "anchore/grype:latest"
    trivyVolume   = "dockverse-trivy-db"
    grypeVolume   = "dockverse-grype-db"
)

// ScannerConfig holds per-request scan settings
type ScannerConfig struct {
    Scanner  string // "trivy" | "grype" | "both"
    Criteria string // "never" | "any" | "critical_high" | "critical" | "more_than_current"
}

// ScanEvent is emitted during scanning for SSE streaming
type ScanEvent struct {
    Stage    string `json:"stage"`    // checking | pulling | scanning | parsing | complete | error
    Scanner  string `json:"scanner,omitempty"`
    Message  string `json:"message"`
    Progress int    `json:"progress,omitempty"`
    Output   string `json:"output,omitempty"`
}

// ScanEngine runs vulnerability scanners as ephemeral Docker containers
type ScanEngine struct {
    dm        *DockerManager
    scanStore *ScanStore
}

func NewScanEngine(dm *DockerManager, ss *ScanStore) *ScanEngine {
    return &ScanEngine{dm: dm, scanStore: ss}
}

// Scan runs the configured scanner(s) on an image and returns results.
// onEvent is called with progress events (can be nil for silent operation).
func (se *ScanEngine) Scan(ctx context.Context, hostID, imageName string, cfg ScannerConfig, onEvent func(ScanEvent)) ([]ScanResult, error) {
    emit := func(e ScanEvent) {
        if onEvent != nil {
            onEvent(e)
        }
    }

    var results []ScanResult

    runScanner := func(scanner string) error {
        emit(ScanEvent{Stage: "checking", Scanner: scanner,
            Message: fmt.Sprintf("Checking %s scanner image...", scanner)})

        cli, err := se.dm.GetClient(hostID)
        if err != nil {
            return fmt.Errorf("get client: %w", err)
        }

        // Pull scanner image if needed
        scannerImg := trivyImage
        if scanner == "grype" {
            scannerImg = grypeImage
        }
        if err := ensureImage(ctx, cli, scannerImg, emit, scanner); err != nil {
            return fmt.Errorf("scanner image unavailable: %w", err)
        }

        // Ensure cache volume exists
        volName := trivyVolume
        if scanner == "grype" {
            volName = grypeVolume
        }
        if err := ensureVolume(ctx, cli, volName); err != nil {
            log.Printf("[Scanner] Warning: could not ensure volume %s: %v", volName, err)
        }

        emit(ScanEvent{Stage: "scanning", Scanner: scanner, Progress: 30,
            Message: fmt.Sprintf("Scanning %s with %s...", imageName, scanner)})

        // Build command
        var cmd []string
        var envVars []string
        var cacheDir string

        if scanner == "trivy" {
            cacheDir = "/cache/trivy"
            cmd = []string{"image", "--format", "json", "--quiet", imageName}
            envVars = []string{"TRIVY_CACHE_DIR=" + cacheDir}
        } else {
            cacheDir = "/cache/grype"
            cmd = []string{"-o", "json", "-q", imageName}
            envVars = []string{"GRYPE_DB_CACHE_DIR=" + cacheDir}
        }

        // Run scanner container
        startTime := time.Now()
        output, err := runScannerContainer(ctx, cli, scannerImg, cmd, envVars, volName, cacheDir, scanner, emit)
        if err != nil {
            return fmt.Errorf("%s scan failed: %w", scanner, err)
        }

        emit(ScanEvent{Stage: "parsing", Scanner: scanner, Progress: 80,
            Message: "Parsing scan results..."})

        // Parse output
        var vulns []Vulnerability
        var summary ScanSummary

        if scanner == "trivy" {
            vulns, summary, err = parseTrivyOutput(output)
        } else {
            vulns, summary, err = parseGrypeOutput(output)
        }
        if err != nil {
            return fmt.Errorf("parse %s output: %w", scanner, err)
        }

        result := ScanResult{
            ID:              fmt.Sprintf("%s-%d", scanner, time.Now().UnixNano()),
            ImageName:       imageName,
            HostID:          hostID,
            Scanner:         scanner,
            ScannedAt:       time.Now(),
            ScanDurationMs:  time.Since(startTime).Milliseconds(),
            Summary:         summary,
            Vulnerabilities: vulns,
        }

        results = append(results, result)

        emit(ScanEvent{Stage: "complete", Scanner: scanner, Progress: 100,
            Message: fmt.Sprintf("%s scan complete: %dC %dH %dM %dL",
                scanner, summary.Critical, summary.High, summary.Medium, summary.Low)})

        return nil
    }

    if cfg.Scanner == "trivy" || cfg.Scanner == "both" {
        if err := runScanner("trivy"); err != nil {
            if cfg.Scanner == "trivy" {
                return nil, err
            }
            log.Printf("[Scanner] Trivy failed: %v", err)
        }
    }

    if cfg.Scanner == "grype" || cfg.Scanner == "both" {
        if err := runScanner("grype"); err != nil {
            if cfg.Scanner == "grype" {
                return nil, err
            }
            log.Printf("[Scanner] Grype failed: %v", err)
        }
    }

    return results, nil
}

// EvaluateCriteria checks if an update should be blocked based on scan results and criteria.
// Returns (blocked bool, reason string)
func EvaluateCriteria(criteria string, newSummary ScanSummary, baseline *ScanSummary) (bool, string) {
    switch criteria {
    case "never":
        return false, ""
    case "any":
        total := newSummary.Critical + newSummary.High + newSummary.Medium + newSummary.Low + newSummary.Negligible + newSummary.Unknown
        if total > 0 {
            return true, fmt.Sprintf("Found %d vulnerabilities (criteria: block on any)", total)
        }
    case "critical_high":
        if newSummary.Critical > 0 || newSummary.High > 0 {
            return true, fmt.Sprintf("Found %d critical, %d high vulnerabilities", newSummary.Critical, newSummary.High)
        }
    case "critical":
        if newSummary.Critical > 0 {
            return true, fmt.Sprintf("Found %d critical vulnerabilities", newSummary.Critical)
        }
    case "more_than_current":
        if baseline != nil {
            if newSummary.Critical > baseline.Critical {
                return true, fmt.Sprintf("New image has more critical vulnerabilities (%d > %d)", newSummary.Critical, baseline.Critical)
            }
            if newSummary.High > baseline.High {
                return true, fmt.Sprintf("New image has more high vulnerabilities (%d > %d)", newSummary.High, baseline.High)
            }
        }
    }
    return false, ""
}

// AggregateSummary returns the worst-case summary across multiple scan results
func AggregateSummary(results []ScanResult) ScanSummary {
    var agg ScanSummary
    for _, r := range results {
        if r.Summary.Critical > agg.Critical { agg.Critical = r.Summary.Critical }
        if r.Summary.High > agg.High { agg.High = r.Summary.High }
        if r.Summary.Medium > agg.Medium { agg.Medium = r.Summary.Medium }
        if r.Summary.Low > agg.Low { agg.Low = r.Summary.Low }
        if r.Summary.Negligible > agg.Negligible { agg.Negligible = r.Summary.Negligible }
        if r.Summary.Unknown > agg.Unknown { agg.Unknown = r.Summary.Unknown }
    }
    return agg
}

// --- internal helpers ---

func ensureImage(ctx context.Context, cli *client.Client, imgRef string, emit func(ScanEvent), scanner string) error {
    images, err := cli.ImageList(ctx, dockerimage.ListOptions{
        Filters: filters.NewArgs(filters.Arg("reference", imgRef)),
    })
    if err == nil && len(images) > 0 {
        return nil // already present
    }
    emit(ScanEvent{Stage: "pulling", Scanner: scanner,
        Message: fmt.Sprintf("Pulling scanner image %s...", imgRef)})
    r, err := cli.ImagePull(ctx, imgRef, dockerimage.PullOptions{})
    if err != nil {
        return err
    }
    defer r.Close()
    _, err = io.Copy(io.Discard, r)
    return err
}

func ensureVolume(ctx context.Context, cli *client.Client, name string) error {
    _, err := cli.VolumeInspect(ctx, name)
    if err == nil {
        return nil // exists
    }
    _, err = cli.VolumeCreate(ctx, volume.CreateOptions{Name: name})
    return err
}

func runScannerContainer(ctx context.Context, cli *client.Client,
    img string, cmd []string, envVars []string,
    volName, cacheDir string, scanner string,
    emit func(ScanEvent)) (string, error) {

    containerName := fmt.Sprintf("dockverse-%s-%d", scanner, time.Now().UnixNano())

    resp, err := cli.ContainerCreate(ctx,
        &container.Config{
            Image: img,
            Cmd:   cmd,
            Env:   envVars,
        },
        &container.HostConfig{
            Mounts: []mount.Mount{
                {
                    Type:     mount.TypeBind,
                    Source:   "/var/run/docker.sock",
                    Target:   "/var/run/docker.sock",
                    ReadOnly: true,
                },
                {
                    Type:   mount.TypeVolume,
                    Source: volName,
                    Target: cacheDir,
                },
            },
            AutoRemove: false, // we'll remove manually after reading logs
        },
        nil, nil, containerName,
    )
    if err != nil {
        return "", fmt.Errorf("create scanner container: %w", err)
    }

    defer func() {
        rmCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        cli.ContainerRemove(rmCtx, resp.ID, container.RemoveOptions{Force: true})
    }()

    if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
        return "", fmt.Errorf("start scanner container: %w", err)
    }

    // Wait for completion with timeout
    waitCtx, waitCancel := context.WithTimeout(ctx, 10*time.Minute)
    defer waitCancel()

    statusCh, errCh := cli.ContainerWait(waitCtx, resp.ID, container.WaitConditionNotRunning)
    select {
    case err := <-errCh:
        if err != nil {
            return "", fmt.Errorf("scanner container wait error: %w", err)
        }
    case status := <-statusCh:
        if status.StatusCode != 0 {
            log.Printf("[Scanner] %s container exited with code %d", scanner, status.StatusCode)
        }
    }

    // Read stdout
    out, err := cli.ContainerLogs(ctx, resp.ID, container.LogsOptions{
        ShowStdout: true,
        ShowStderr: false,
    })
    if err != nil {
        return "", fmt.Errorf("get scanner logs: %w", err)
    }
    defer out.Close()

    var stdoutBuf bytes.Buffer
    if _, err := stdcopy.StdCopy(&stdoutBuf, io.Discard, out); err != nil {
        // Some scanners don't use multiplexed streams, try direct read
        out.Close()
        out2, _ := cli.ContainerLogs(ctx, resp.ID, container.LogsOptions{ShowStdout: true})
        if out2 != nil {
            io.Copy(&stdoutBuf, out2)
            out2.Close()
        }
    }

    return stdoutBuf.String(), nil
}

// parseTrivyOutput parses Trivy JSON output
func parseTrivyOutput(output string) ([]Vulnerability, ScanSummary, error) {
    jsonStr := extractJSON(output)
    if jsonStr == "" {
        return nil, ScanSummary{}, fmt.Errorf("no JSON found in trivy output (len=%d)", len(output))
    }

    var raw struct {
        Results []struct {
            Vulnerabilities []struct {
                VulnerabilityID  string `json:"VulnerabilityID"`
                PkgName          string `json:"PkgName"`
                InstalledVersion string `json:"InstalledVersion"`
                FixedVersion     string `json:"FixedVersion"`
                Severity         string `json:"Severity"`
                Description      string `json:"Description"`
                PrimaryURL       string `json:"PrimaryURL"`
            } `json:"Vulnerabilities"`
        } `json:"Results"`
    }

    if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
        return nil, ScanSummary{}, fmt.Errorf("parse trivy JSON: %w", err)
    }

    var vulns []Vulnerability
    var summary ScanSummary
    for _, result := range raw.Results {
        for _, v := range result.Vulnerabilities {
            sev := strings.ToLower(v.Severity)
            vuln := Vulnerability{
                ID:           v.VulnerabilityID,
                Severity:     sev,
                Package:      v.PkgName,
                Version:      v.InstalledVersion,
                FixedVersion: v.FixedVersion,
                Description:  v.Description,
                Link:         v.PrimaryURL,
                Scanner:      "trivy",
            }
            vulns = append(vulns, vuln)
            countSeverity(&summary, sev)
        }
    }
    return vulns, summary, nil
}

// parseGrypeOutput parses Grype JSON output
func parseGrypeOutput(output string) ([]Vulnerability, ScanSummary, error) {
    jsonStr := extractJSON(output)
    if jsonStr == "" {
        return nil, ScanSummary{}, fmt.Errorf("no JSON found in grype output (len=%d)", len(output))
    }

    var raw struct {
        Matches []struct {
            Vulnerability struct {
                ID          string   `json:"id"`
                Severity    string   `json:"severity"`
                Description string   `json:"description"`
                DataSource  string   `json:"dataSource"`
                Fix         struct {
                    Versions []string `json:"versions"`
                } `json:"fix"`
            } `json:"vulnerability"`
            Artifact struct {
                Name    string `json:"name"`
                Version string `json:"version"`
            } `json:"artifact"`
        } `json:"matches"`
    }

    if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
        return nil, ScanSummary{}, fmt.Errorf("parse grype JSON: %w", err)
    }

    var vulns []Vulnerability
    var summary ScanSummary
    for _, m := range raw.Matches {
        sev := strings.ToLower(m.Vulnerability.Severity)
        fixedVer := ""
        if len(m.Vulnerability.Fix.Versions) > 0 {
            fixedVer = m.Vulnerability.Fix.Versions[0]
        }
        vuln := Vulnerability{
            ID:           m.Vulnerability.ID,
            Severity:     sev,
            Package:      m.Artifact.Name,
            Version:      m.Artifact.Version,
            FixedVersion: fixedVer,
            Description:  m.Vulnerability.Description,
            Link:         m.Vulnerability.DataSource,
            Scanner:      "grype",
        }
        vulns = append(vulns, vuln)
        countSeverity(&summary, sev)
    }
    return vulns, summary, nil
}

func extractJSON(output string) string {
    start := strings.Index(output, "{")
    end := strings.LastIndex(output, "}")
    if start == -1 || end == -1 || end <= start {
        return ""
    }
    return output[start : end+1]
}

func countSeverity(s *ScanSummary, sev string) {
    switch sev {
    case "critical":
        s.Critical++
    case "high":
        s.High++
    case "medium":
        s.Medium++
    case "low":
        s.Low++
    case "negligible":
        s.Negligible++
    default:
        s.Unknown++
    }
}
```

Add missing import for volume:
```go
"github.com/docker/docker/api/types/volume"
```

**Step 2: Add volume package to go.mod if needed**
```bash
cd backend && go get github.com/docker/docker/api/types/volume
go build ./...
```

**Step 3: Commit**
```bash
git add backend/models/scanner.go backend/go.mod backend/go.sum
git commit -m "feat: add vulnerability scanner engine (Trivy + Grype)"
```

---

## Task 4: Integrate Scanner into Update Flow

**Files:**
- Modify: `backend/main.go` (update endpoint, `updateContainerImage` function)

**Context:** The existing `updateContainerImage` at line 2543 does health-check validation but no security scanning. We need to add scan + blocking criteria + force override support.

**Step 1: Add scan parameters to update request body**

At the update endpoint (line 3724), parse a JSON body:
```go
type UpdateRequest struct {
    Scanner   string `json:"scanner"`   // "trivy" | "grype" | "both" | "" (defaults to "trivy")
    Criteria  string `json:"criteria"`  // "never" | "any" | "critical_high" | "critical" | "more_than_current"
    Force     bool   `json:"force"`     // admin override to ignore blocking
}
```

**Step 2: Add SSE endpoint for streaming updates**

Before the existing update endpoint, add a SSE endpoint:
```go
// SSE: stream real-time update + scan progress
protected.Get("/containers/:hostId/:containerId/update-stream", func(c *fiber.Ctx) error {
    hostID := c.Params("hostId")
    containerID := c.Params("containerId")

    c.Set("Content-Type", "text/event-stream")
    c.Set("Cache-Control", "no-cache")
    c.Set("Connection", "keep-alive")
    c.Set("X-Accel-Buffering", "no")

    scanner := c.Query("scanner", "trivy")
    criteria := c.Query("criteria", "never")
    force := c.Query("force", "false") == "true"

    c.Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
        emit := func(event string, data interface{}) {
            b, _ := json.Marshal(data)
            fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, string(b))
            w.Flush()
        }

        ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
        defer cancel()

        err := dm.updateContainerWithScan(ctx, hostID, containerID,
            models.ScannerConfig{Scanner: scanner, Criteria: criteria},
            force, scanEngine, scanStore, emit)

        if err != nil {
            emit("error", fiber.Map{"message": err.Error()})
        } else {
            emit("done", fiber.Map{"success": true})
        }
    }))
    return nil
})
```

**Step 3: Create `updateContainerWithScan` on DockerManager**

This new method wraps the existing `updateContainerImage` and adds scan logic:
```go
func (dm *DockerManager) updateContainerWithScan(
    ctx context.Context,
    hostID, containerID string,
    cfg models.ScannerConfig,
    forceOverride bool,
    engine *models.ScanEngine,
    store *models.ScanStore,
    emit func(event string, data interface{}),
) error {
    // Step 1: Pull new image to temp tag
    emit("progress", map[string]interface{}{"stage": "pulling", "message": "Pulling new image..."})
    // ... (pull logic from existing updateContainerImage)

    // Step 2: If scanner configured, run scan
    if cfg.Scanner != "" && cfg.Scanner != "none" {
        emit("progress", map[string]interface{}{"stage": "scanning", "message": "Starting vulnerability scan..."})

        results, err := engine.Scan(ctx, hostID, imageName, cfg, func(e models.ScanEvent) {
            emit("scan_progress", e)
        })
        if err != nil {
            return fmt.Errorf("vulnerability scan failed: %w", err)
        }

        // Save scan results
        for i := range results {
            results[i].ContainerID = containerID
            results[i].TriggeredBy = "manual"
            store.Save(&results[i])
        }

        // Evaluate blocking criteria
        if cfg.Criteria != "never" && !forceOverride {
            summary := models.AggregateSummary(results)
            var baseline *models.ScanSummary
            if cfg.Criteria == "more_than_current" {
                if prev, err := store.GetLatestForImage(imageName); err == nil && prev != nil {
                    baseline = &prev.Summary
                }
            }
            blocked, reason := models.EvaluateCriteria(cfg.Criteria, summary, baseline)
            if blocked {
                // Emit block event with CVE details
                emit("blocked", map[string]interface{}{
                    "reason":  reason,
                    "summary": summary,
                    "results": results,
                })
                return fmt.Errorf("update blocked: %s", reason)
            }
        }

        if forceOverride {
            emit("progress", map[string]interface{}{"stage": "override",
                "message": "Force override active - proceeding despite vulnerabilities"})
        }

        emit("progress", map[string]interface{}{"stage": "scan_passed",
            "message": "Scan complete, proceeding with update..."})
    }

    // Step 3: Proceed with actual update (existing health-check logic)
    emit("progress", map[string]interface{}{"stage": "updating", "message": "Recreating container..."})
    return dm.updateContainerImage(ctx, hostID, containerID)
}
```

**Step 4: Add scan API endpoints**

```go
// GET /api/scans — list recent scans
protected.Get("/scans", func(c *fiber.Ctx) error {
    limit := 100
    results, err := scanStore.List(limit)
    if err != nil {
        return c.Status(500).JSON(fiber.Map{"error": err.Error()})
    }
    return c.JSON(results)
})

// POST /api/containers/:hostId/:containerId/scan — manual scan
protected.Post("/containers/:hostId/:containerId/scan", func(c *fiber.Ctx) error {
    hostID := c.Params("hostId")
    containerID := c.Params("containerId")

    var req struct {
        Scanner string `json:"scanner"`
    }
    c.BodyParser(&req)
    if req.Scanner == "" { req.Scanner = "trivy" }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
    defer cancel()

    // Get image name from container
    cli, err := dm.GetClient(hostID)
    if err != nil {
        return c.Status(500).JSON(fiber.Map{"error": err.Error()})
    }
    inspect, err := cli.ContainerInspect(ctx, containerID)
    if err != nil {
        return c.Status(404).JSON(fiber.Map{"error": "container not found"})
    }

    results, err := scanEngine.Scan(ctx, hostID, inspect.Config.Image,
        models.ScannerConfig{Scanner: req.Scanner, Criteria: "never"}, nil)
    if err != nil {
        return c.Status(500).JSON(fiber.Map{"error": err.Error()})
    }

    for i := range results {
        results[i].ContainerID = containerID
        results[i].ContainerName = strings.TrimPrefix(inspect.Name, "/")
        results[i].TriggeredBy = "manual"
        scanStore.Save(&results[i])
    }

    return c.JSON(results)
})
```

**Step 5: Wire up scanEngine and scanStore in main()**

Find where `dm` is used and add:
```go
scanStore, _ := models.NewScanStore(db)
scanEngine := models.NewScanEngine(dm, scanStore)
```

Pass these to the route closures (or make them package-level vars alongside `dm`).

**Step 6: Build**
```bash
cd backend && go build ./...
```

**Step 7: Commit**
```bash
git add backend/main.go backend/models/
git commit -m "feat: integrate vulnerability scanning into container update flow

- SSE endpoint for real-time update+scan progress
- Trivy and Grype scanner integration
- Configurable blocking criteria (never/any/critical_high/critical/more_than_current)
- Force override support for admin
- Scan results persisted to SQLite
- Manual scan endpoint POST /api/containers/:h/:c/scan
- Scan history endpoint GET /api/scans"
```

---

## Task 5: Frontend — Update Modal with SSE Streaming

**Files:**
- Modify: `frontend/src/lib/components/UpdateModal.svelte`
- Modify: `frontend/src/lib/api/docker.ts`

**Context:** Replace the simple polling-based update flow with SSE streaming. Add scanner selection and vulnerability criteria dropdowns. Show real-time scan progress and CVE list.

**Step 1: Add SSE update function to docker.ts**

```typescript
// In frontend/src/lib/api/docker.ts

export interface ScanEvent {
    stage: string;
    scanner?: string;
    message: string;
    progress?: number;
    output?: string;
}

export interface ScanResult {
    id: string;
    imageName: string;
    scanner: string;
    scannedAt: string;
    summary: { critical: number; high: number; medium: number; low: number; negligible: number; unknown: number };
    vulnerabilities: Vulnerability[];
}

export interface Vulnerability {
    id: string;
    severity: string;
    package: string;
    version: string;
    fixedVersion?: string;
    description?: string;
    link?: string;
    scanner: string;
}

export function streamContainerUpdate(
    hostId: string,
    containerId: string,
    options: { scanner: string; criteria: string; force?: boolean },
    onEvent: (event: string, data: unknown) => void
): EventSource {
    const params = new URLSearchParams({
        scanner: options.scanner,
        criteria: options.criteria,
        force: options.force ? 'true' : 'false'
    });
    const url = `${API_BASE}/api/containers/${hostId}/${containerId}/update-stream?${params}`;
    const es = new EventSource(url);

    ['progress', 'scan_progress', 'blocked', 'done', 'error'].forEach(eventType => {
        es.addEventListener(eventType, (e: MessageEvent) => {
            try {
                onEvent(eventType, JSON.parse(e.data));
            } catch {
                onEvent(eventType, e.data);
            }
        });
    });

    return es;
}

export async function scanContainer(hostId: string, containerId: string, scanner = 'trivy'): Promise<ScanResult[]> {
    const res = await fetchWithTimeout(
        `${API_BASE}/api/containers/${hostId}/${containerId}/scan`,
        { method: 'POST', headers: { ...getAuthHeaders(), 'Content-Type': 'application/json' },
          body: JSON.stringify({ scanner }) },
        120000
    );
    if (!res.ok) throw new Error('Scan failed');
    return res.json();
}

export async function fetchScanHistory(): Promise<ScanResult[]> {
    const res = await fetchWithTimeout(`${API_BASE}/api/scans`, { headers: getAuthHeaders() }, 15000);
    if (!res.ok) throw new Error('Failed to fetch scan history');
    return res.json();
}
```

**Step 2: Rewrite UpdateModal.svelte**

The key changes to the existing UpdateModal:
1. Add scanner + criteria selector dropdowns
2. Replace polling with SSE EventSource
3. Show real-time progress steps
4. Display CVE list when scan completes
5. Show "Force Update" button when blocked (if user is admin)

Full component structure (adapt existing styles):
```svelte
<script lang="ts">
  import { streamContainerUpdate, type ScanResult } from '$lib/api/docker';

  export let hostId: string;
  export let containerId: string;
  export let containerName: string;
  export let onClose: () => void;
  export let onSuccess: () => void;
  export let isAdmin = false;

  type Stage = 'idle' | 'running' | 'blocked' | 'done' | 'error';
  let stage: Stage = 'idle';
  let scanner = 'trivy';
  let criteria = 'never';
  let steps: { label: string; status: 'pending' | 'running' | 'done' | 'error' }[] = [];
  let scanResults: ScanResult[] = [];
  let blockReason = '';
  let errorMsg = '';
  let eventSource: EventSource | null = null;

  function startUpdate(force = false) {
    stage = 'running';
    steps = [
      { label: 'Pulling new image', status: 'pending' },
      { label: `Scanning with ${scanner}`, status: 'pending' },
      { label: 'Recreating container', status: 'pending' },
    ];
    scanResults = [];
    blockReason = '';

    eventSource = streamContainerUpdate(hostId, containerId, { scanner, criteria, force }, handleEvent);
    eventSource.onerror = () => {
      stage = 'error';
      errorMsg = 'Connection to update stream lost';
      eventSource?.close();
    };
  }

  function handleEvent(event: string, data: unknown) {
    const d = data as Record<string, unknown>;
    switch (event) {
      case 'progress':
        updateStep(d.stage as string, d.message as string);
        break;
      case 'scan_progress':
        // Show scanner sub-progress
        updateStep('scanning', d.message as string);
        break;
      case 'blocked':
        stage = 'blocked';
        blockReason = d.reason as string;
        scanResults = (d.results as ScanResult[]) || [];
        eventSource?.close();
        break;
      case 'done':
        stage = 'done';
        eventSource?.close();
        setTimeout(onSuccess, 1500);
        break;
      case 'error':
        stage = 'error';
        errorMsg = d.message as string;
        eventSource?.close();
        break;
    }
  }

  function updateStep(stageName: string, message: string) {
    const stageMap: Record<string, number> = {
      pulling: 0, scanning: 1, scan_passed: 1, updating: 2, done: 2
    };
    const idx = stageMap[stageName] ?? -1;
    if (idx >= 0) {
      steps = steps.map((s, i) => ({
        ...s,
        status: i < idx ? 'done' : i === idx ? 'running' : 'pending'
      }));
    }
  }
</script>

<!-- Modal content — adapt to existing modal styles in codebase -->
<div class="modal">
  {#if stage === 'idle'}
    <h2>Update {containerName}</h2>
    <label>Scanner
      <select bind:value={scanner}>
        <option value="trivy">Trivy</option>
        <option value="grype">Grype</option>
        <option value="both">Both</option>
        <option value="none">None (skip scan)</option>
      </select>
    </label>
    <label>Block criteria
      <select bind:value={criteria}>
        <option value="never">Never block</option>
        <option value="critical">Critical only</option>
        <option value="critical_high">Critical or High</option>
        <option value="any">Any vulnerability</option>
        <option value="more_than_current">More than current</option>
      </select>
    </label>
    <button on:click={() => startUpdate(false)}>Update</button>
    <button on:click={onClose}>Cancel</button>

  {:else if stage === 'running'}
    <h2>Updating {containerName}...</h2>
    {#each steps as step}
      <div class="step step--{step.status}">{step.label}</div>
    {/each}

  {:else if stage === 'blocked'}
    <h2>⚠️ Update Blocked</h2>
    <p>{blockReason}</p>
    <!-- Show top CVEs -->
    {#each scanResults as result}
      <div>
        <strong>{result.scanner}</strong>:
        {result.summary.critical}C {result.summary.high}H {result.summary.medium}M
      </div>
    {/each}
    <button on:click={onClose}>Cancel</button>
    {#if isAdmin}
      <button class="danger" on:click={() => startUpdate(true)}>Force Update (Override)</button>
    {/if}

  {:else if stage === 'done'}
    <h2>✅ Updated Successfully</h2>
    <button on:click={onClose}>Close</button>

  {:else if stage === 'error'}
    <h2>❌ Update Failed</h2>
    <p>{errorMsg}</p>
    <button on:click={onClose}>Close</button>
  {/if}
</div>
```

**Step 3: Build frontend**
```bash
cd frontend && npm run build
```
Expected: successful build, no TypeScript errors.

**Step 4: Commit**
```bash
git add frontend/src/lib/components/UpdateModal.svelte frontend/src/lib/api/docker.ts
git commit -m "feat: update modal with SSE streaming and vulnerability scan UI"
```

---

## Task 6: Frontend — Container Card Vulnerability Badge

**Files:**
- Modify: `frontend/src/lib/components/ContainerCard.svelte`
- Modify: `frontend/src/lib/stores/docker.ts`

**Context:** Show vulnerability summary badge on container cards when a scan result exists for the container's image.

**Step 1: Add scan store**

In `frontend/src/lib/stores/docker.ts`, add:
```typescript
import { fetchScanHistory } from '$lib/api/docker';

export const scanResults = writable<ScanResult[]>([]);

export async function refreshScans() {
    try {
        const results = await fetchScanHistory();
        scanResults.set(results);
    } catch (e) {
        console.error('Failed to fetch scan history:', e);
    }
}
```

**Step 2: Add vulnerability badge to ContainerCard**

Find where the update badge is shown in `ContainerCard.svelte` and add alongside it:
```svelte
<script lang="ts">
  import { scanResults } from '$lib/stores/docker';
  // ...existing imports

  $: latestScan = $scanResults.find(s => s.imageName === container.image);
</script>

<!-- In the card template, near the update badge: -->
{#if latestScan && (latestScan.summary.critical > 0 || latestScan.summary.high > 0)}
  <span class="badge badge--vuln" title="Vulnerability scan results">
    🛡️
    {#if latestScan.summary.critical > 0}
      <span class="critical">{latestScan.summary.critical}C</span>
    {/if}
    {#if latestScan.summary.high > 0}
      <span class="high">{latestScan.summary.high}H</span>
    {/if}
  </span>
{/if}
```

**Step 3: Build**
```bash
cd frontend && npm run build
```

**Step 4: Commit**
```bash
git add frontend/src/lib/components/ContainerCard.svelte frontend/src/lib/stores/docker.ts
git commit -m "feat: show vulnerability severity badge on container cards"
```

---

## Task 7: Frontend — Security / Scan History Page

**Files:**
- Create: `frontend/src/routes/security/+page.svelte`
- Create: `frontend/src/routes/security/+page.ts` (optional, for preloading)

**Step 1: Create the security page**

```svelte
<!-- frontend/src/routes/security/+page.svelte -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { fetchScanHistory } from '$lib/api/docker';
  import type { ScanResult, Vulnerability } from '$lib/api/docker';

  let scans: ScanResult[] = [];
  let loading = true;
  let selectedScan: ScanResult | null = null;
  let filterSeverity = 'all';

  onMount(async () => {
    try {
      scans = await fetchScanHistory();
    } finally {
      loading = false;
    }
  });

  function severityColor(sev: string) {
    return { critical: 'text-red-600', high: 'text-orange-500',
              medium: 'text-yellow-500', low: 'text-blue-400' }[sev] ?? 'text-gray-400';
  }
</script>

<div class="p-6">
  <h1 class="text-2xl font-bold mb-4">Security — Scan History</h1>

  {#if loading}
    <p>Loading...</p>
  {:else if scans.length === 0}
    <p class="text-muted">No scans yet. Trigger an update with scanning enabled.</p>
  {:else}
    <table class="w-full text-sm">
      <thead>
        <tr>
          <th>Image</th>
          <th>Scanner</th>
          <th>Critical</th>
          <th>High</th>
          <th>Medium</th>
          <th>Low</th>
          <th>Scanned At</th>
          <th>Status</th>
        </tr>
      </thead>
      <tbody>
        {#each scans as scan}
          <tr class="cursor-pointer hover:bg-muted" on:click={() => selectedScan = scan}>
            <td>{scan.imageName}</td>
            <td>{scan.scanner}</td>
            <td class="text-red-600">{scan.summary.critical}</td>
            <td class="text-orange-500">{scan.summary.high}</td>
            <td class="text-yellow-500">{scan.summary.medium}</td>
            <td class="text-blue-400">{scan.summary.low}</td>
            <td>{new Date(scan.scannedAt).toLocaleString()}</td>
            <td>
              {#if scan.blocked}
                <span class="badge badge--blocked">Blocked</span>
              {:else}
                <span class="badge badge--ok">Passed</span>
              {/if}
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  {/if}
</div>

<!-- CVE detail drawer/modal -->
{#if selectedScan}
  <div class="modal-overlay" on:click={() => selectedScan = null}>
    <div class="modal" on:click|stopPropagation>
      <h2>{selectedScan.imageName} — {selectedScan.scanner} results</h2>
      <p class="text-sm text-muted">Scanned {new Date(selectedScan.scannedAt).toLocaleString()}</p>
      <table class="w-full text-sm mt-4">
        <thead>
          <tr><th>CVE</th><th>Severity</th><th>Package</th><th>Version</th><th>Fix</th></tr>
        </thead>
        <tbody>
          {#each selectedScan.vulnerabilities.sort((a, b) => {
            const order = ['critical','high','medium','low','negligible','unknown'];
            return order.indexOf(a.severity) - order.indexOf(b.severity);
          }) as v}
            <tr>
              <td><a href={v.link} target="_blank" class="text-primary hover:underline">{v.id}</a></td>
              <td class={severityColor(v.severity)}>{v.severity}</td>
              <td>{v.package}</td>
              <td>{v.version}</td>
              <td>{v.fixedVersion || '—'}</td>
            </tr>
          {/each}
        </tbody>
      </table>
      <button on:click={() => selectedScan = null}>Close</button>
    </div>
  </div>
{/if}
```

**Step 2: Add Security link to navigation**

Find the navigation component (likely in `+layout.svelte` or a nav component) and add:
```svelte
<a href="/security">Security</a>
```

**Step 3: Build**
```bash
cd frontend && npm run build
```

**Step 4: Commit**
```bash
git add frontend/src/routes/security/
git commit -m "feat: add Security page with scan history and CVE detail"
```

---

## Task 8: Consolidate Documentation

**Files:**
- Review: all `*.md` files in project root
- Move: scattered docs to `docs/` directory
- Update: `README.md` with new features

**Step 1: List all markdown files**
```bash
find /Users/vcolmenares/Documents/Laboratories/Antigravity/skills/dockerverse-project/dockerverse -name "*.md" | grep -v node_modules | grep -v .git
```

**Step 2: Move large docs to docs/**
```bash
cd dockerverse-project/dockerverse
mkdir -p docs
# Move large files that are not README/CHANGELOG
mv DEVELOPMENT_CONTINUATION_GUIDE.md docs/
mv SECURITY_REMEDIATION.md docs/
mv DATABASEMENT_ANALYSIS.md docs/
mv WEB_TERMINAL_RESEARCH_2026.md docs/
mv UNIFIED_CONTAINER_ARCHITECTURE.md docs/
```

**Step 3: Update README.md**

Add a "Security Scanning" section describing:
- Vulnerability scanning with Trivy and Grype
- Configurable blocking criteria
- Security history page at `/security`
- How to configure scanner (Settings page)

**Step 4: Commit**
```bash
git add docs/ README.md
git commit -m "docs: consolidate documentation into docs/ directory, update README"
```

---

## Task 9: Deploy to raspi-main and Test

**Step 1: Copy build to raspi-main**
```bash
# If using docker build + deploy script:
cd /Users/vcolmenares/Documents/Laboratories/Antigravity/skills/dockerverse-project/dockerverse
ssh raspi-main "cd /path/to/dockerverse && git pull"
# OR use the deploy script if one exists:
ls *.sh deploy* 2>/dev/null
```

**Step 2: Build and deploy**
```bash
ssh raspi-main "cd /path/to/dockerverse && docker compose build && docker compose up -d"
```

**Step 3: Verify update detection fix**
```bash
# Check backend logs for correct digest comparison
ssh raspi-main "docker compose logs backend | grep -i 'UpdateCheck\|hasUpdate\|false positive' | tail -20"
```

Expected: Containers that were falsely showing as "needs update" should now show `hasUpdate=false`.

**Step 4: Verify scanner works**
```bash
# Trigger a manual scan via API
curl -X POST "http://192.168.1.145:3007/api/containers/raspi1/CONTAINER_ID/scan" \
  -H "Authorization: Bearer TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"scanner":"trivy"}'
```

Expected: JSON response with vulnerability summary.

**Step 5: Verify SSE stream works**
```bash
curl -N "http://192.168.1.145:3007/api/containers/raspi1/CONTAINER_ID/update-stream?scanner=trivy&criteria=never" \
  -H "Authorization: Bearer TOKEN"
```

Expected: SSE events streamed in real-time.

**Step 6: Check Security page in browser**

Open `http://192.168.1.145:3007/security` — should show scan history table.

---

## Task 10: Final Git Push

**Security checklist before push:**
```bash
# Verify no secrets in tracked files
git diff HEAD --name-only | xargs grep -l "password\|secret\|api_key\|API_KEY" 2>/dev/null
grep -r "JWT_SECRET\|WATCHTOWER_TOKEN\|SMTP2GO" --include="*.go" --include="*.ts" backend/ frontend/src/

# Verify .env is not tracked
git status | grep ".env"

# Verify .gitignore covers sensitive files
cat .gitignore | grep -E "\.env|\.key|\.pem"
```

**Step 1: Final review**
```bash
git log --oneline -10
git diff origin/main..HEAD --stat
```

**Step 2: Push**
```bash
git push origin main
```

---

## Implementation Checklist

- [ ] Task 1: Fix multi-arch digest bug + cache key refactor
- [ ] Task 2: Add SQLite scan results model
- [ ] Task 3: Vulnerability scanner engine (Trivy + Grype)
- [ ] Task 4: Integrate scanner into update flow + SSE + API endpoints
- [ ] Task 5: UpdateModal with SSE streaming + scan UI
- [ ] Task 6: ContainerCard vulnerability badge
- [ ] Task 7: Security/scan history page
- [ ] Task 8: Consolidate documentation
- [ ] Task 9: Deploy and test on raspi-main
- [ ] Task 10: Security check + git push
