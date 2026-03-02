package models

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

// ScannerConfig holds the scanner selection and blocking criteria.
type ScannerConfig struct {
	Scanner  string // "trivy" | "grype" | "both" | "none"
	Criteria string // "never" | "any" | "critical_high" | "critical" | "more_than_current"
}

// ScanEvent is emitted during a scan run for SSE progress streaming.
type ScanEvent struct {
	Stage    string `json:"stage"`
	Scanner  string `json:"scanner,omitempty"`
	Message  string `json:"message"`
	Progress int    `json:"progress,omitempty"`
	Output   string `json:"output,omitempty"`
}

// DockerClientProvider is satisfied by any type that can return a Docker client
// for a given host ID. The DockerManager in package main implements this.
type DockerClientProvider interface {
	GetClient(hostID string) (*client.Client, error)
}

// ScanEngine executes vulnerability scans using ephemeral Docker containers.
type ScanEngine struct {
	dm DockerClientProvider
}

// NewScanEngine creates a ScanEngine backed by any DockerClientProvider
// (e.g. the *DockerManager from package main).
func NewScanEngine(dm DockerClientProvider) *ScanEngine {
	return &ScanEngine{dm: dm}
}

// scannerSpec holds the static configuration for a single scanner tool.
type scannerSpec struct {
	name       string
	imageName  string
	volumeName string
	cacheDir   string
	envVar     string
	cmdFn      func(imageName string) []string
}

var trivySpec = scannerSpec{
	name:       "trivy",
	imageName:  "aquasec/trivy:latest",
	volumeName: "dockverse-trivy-db",
	cacheDir:   "/cache/trivy",
	envVar:     "TRIVY_CACHE_DIR",
	cmdFn: func(imageName string) []string {
		return []string{"image", "--format", "json", "--quiet", imageName}
	},
}

var grypeSpec = scannerSpec{
	name:       "grype",
	imageName:  "anchore/grype:latest",
	volumeName: "dockverse-grype-db",
	cacheDir:   "/cache/grype",
	envVar:     "GRYPE_DB_CACHE_DIR",
	cmdFn: func(imageName string) []string {
		return []string{"-o", "json", "-q", imageName}
	},
}

// Scan runs the configured scanner(s) on an image and returns the results.
// hostID identifies which Docker host runs the scan containers.
// onEvent, if non-nil, is called with progress updates suitable for SSE.
func (se *ScanEngine) Scan(ctx context.Context, hostID, imageName string, cfg ScannerConfig, onEvent func(ScanEvent)) ([]ScanResult, error) {
	emit := func(ev ScanEvent) {
		if onEvent != nil {
			onEvent(ev)
		}
	}

	var specs []scannerSpec
	switch cfg.Scanner {
	case "trivy":
		specs = []scannerSpec{trivySpec}
	case "grype":
		specs = []scannerSpec{grypeSpec}
	case "both":
		specs = []scannerSpec{trivySpec, grypeSpec}
	case "none", "":
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown scanner: %s", cfg.Scanner)
	}

	var results []ScanResult
	for _, spec := range specs {
		emit(ScanEvent{
			Stage:   "start",
			Scanner: spec.name,
			Message: fmt.Sprintf("Starting %s scan of %s", spec.name, imageName),
		})

		result, err := se.runScanner(ctx, hostID, imageName, spec, emit)
		if err != nil {
			return results, fmt.Errorf("%s scan failed: %w", spec.name, err)
		}
		results = append(results, result)

		emit(ScanEvent{
			Stage:   "done",
			Scanner: spec.name,
			Message: fmt.Sprintf("%s scan complete: %d vulnerabilities found", spec.name, len(result.Vulnerabilities)),
		})
	}

	return results, nil
}

// runScanner executes a single scanner tool as an ephemeral container.
func (se *ScanEngine) runScanner(ctx context.Context, hostID, imageName string, spec scannerSpec, emit func(ScanEvent)) (ScanResult, error) {
	start := time.Now()
	result := ScanResult{
		ImageName: imageName,
		HostID:    hostID,
		Scanner:   spec.name,
		ScannedAt: start,
	}

	cli, err := se.dm.GetClient(hostID)
	if err != nil {
		return result, fmt.Errorf("get docker client: %w", err)
	}

	// Step 1: Ensure scanner image is present locally.
	emit(ScanEvent{Stage: "check_image", Scanner: spec.name, Message: fmt.Sprintf("Checking for %s image", spec.imageName)})

	imgFilter := filters.NewArgs(filters.Arg("reference", spec.imageName))
	images, err := cli.ImageList(ctx, image.ListOptions{Filters: imgFilter})
	if err != nil {
		return result, fmt.Errorf("image list: %w", err)
	}

	if len(images) == 0 {
		emit(ScanEvent{Stage: "pull", Scanner: spec.name, Message: fmt.Sprintf("Pulling %s …", spec.imageName)})

		pullCtx, pullCancel := context.WithTimeout(ctx, 10*time.Minute)
		defer pullCancel()

		pullResp, err := cli.ImagePull(pullCtx, spec.imageName, image.PullOptions{})
		if err != nil {
			return result, fmt.Errorf("pull scanner image %s: %w", spec.imageName, err)
		}
		_, _ = io.Copy(io.Discard, pullResp)
		pullResp.Close()

		emit(ScanEvent{Stage: "pull_done", Scanner: spec.name, Message: fmt.Sprintf("Pulled %s", spec.imageName)})
	}

	// Step 2: Ensure cache volume exists.
	_, err = cli.VolumeInspect(ctx, spec.volumeName)
	if err != nil {
		emit(ScanEvent{Stage: "create_volume", Scanner: spec.name, Message: fmt.Sprintf("Creating cache volume %s", spec.volumeName)})
		_, err = cli.VolumeCreate(ctx, volume.CreateOptions{Name: spec.volumeName})
		if err != nil {
			return result, fmt.Errorf("create cache volume %s: %w", spec.volumeName, err)
		}
	}

	// Step 3: Create scanner container.
	emit(ScanEvent{Stage: "create_container", Scanner: spec.name, Message: fmt.Sprintf("Creating %s container", spec.name)})

	containerCfg := &container.Config{
		Image: spec.imageName,
		Cmd:   spec.cmdFn(imageName),
		Env:   []string{fmt.Sprintf("%s=%s", spec.envVar, spec.cacheDir)},
	}

	hostCfg := &container.HostConfig{
		Binds: []string{
			"/var/run/docker.sock:/var/run/docker.sock:ro",
			fmt.Sprintf("%s:%s", spec.volumeName, spec.cacheDir),
		},
		AutoRemove: false, // we need to read logs before removal
	}

	createResp, err := cli.ContainerCreate(ctx, containerCfg, hostCfg, nil, nil, "")
	if err != nil {
		return result, fmt.Errorf("create scanner container: %w", err)
	}
	containerID := createResp.ID

	// Always attempt cleanup.
	defer func() {
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanCancel()
		_ = cli.ContainerRemove(cleanCtx, containerID, container.RemoveOptions{Force: true})
	}()

	// Step 4: Start the container.
	emit(ScanEvent{Stage: "running", Scanner: spec.name, Message: fmt.Sprintf("Running %s …", spec.name), Progress: 25})

	if err := cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return result, fmt.Errorf("start scanner container: %w", err)
	}

	// Step 5: Wait for the container to finish (10-minute timeout).
	waitCtx, waitCancel := context.WithTimeout(ctx, 10*time.Minute)
	defer waitCancel()

	statusCh, errCh := cli.ContainerWait(waitCtx, containerID, container.WaitConditionNotRunning)
	select {
	case waitErr := <-errCh:
		if waitErr != nil {
			return result, fmt.Errorf("wait for scanner container: %w", waitErr)
		}
	case waitResp := <-statusCh:
		if waitResp.Error != nil {
			return result, fmt.Errorf("scanner container error: %s", waitResp.Error.Message)
		}
	}

	emit(ScanEvent{Stage: "parsing", Scanner: spec.name, Message: "Parsing scan output …", Progress: 75})

	// Step 6: Collect container logs (stdout only for JSON).
	logReader, err := cli.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: false,
	})
	if err != nil {
		return result, fmt.Errorf("get container logs: %w", err)
	}
	defer logReader.Close()

	var stdoutBuf bytes.Buffer
	_, _ = stdcopy.StdCopy(&stdoutBuf, io.Discard, logReader)
	rawOutput := stdoutBuf.String()

	// Step 7: Parse JSON output.
	vulns, err := parseOutput(spec.name, rawOutput)
	if err != nil {
		return result, fmt.Errorf("parse %s output: %w", spec.name, err)
	}

	result.Vulnerabilities = vulns
	result.Summary = buildSummary(vulns)
	result.ScanDurationMs = time.Since(start).Milliseconds()

	return result, nil
}

// extractJSON finds the outermost JSON object in a string that may contain
// non-JSON preamble lines (common with both Trivy and Grype).
func extractJSON(s string) string {
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	return s[start : end+1]
}

// parseOutput dispatches to the appropriate parser based on the scanner name.
func parseOutput(scannerName, raw string) ([]Vulnerability, error) {
	jsonStr := extractJSON(raw)
	if jsonStr == "" {
		return nil, fmt.Errorf("no JSON found in %s output", scannerName)
	}

	switch scannerName {
	case "trivy":
		return parseTrivyOutput(jsonStr)
	case "grype":
		return parseGrypeOutput(jsonStr)
	default:
		return nil, fmt.Errorf("unknown scanner: %s", scannerName)
	}
}

// trivyResult mirrors the relevant parts of Trivy's JSON output.
type trivyResult struct {
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

func parseTrivyOutput(jsonStr string) ([]Vulnerability, error) {
	var tr trivyResult
	if err := json.Unmarshal([]byte(jsonStr), &tr); err != nil {
		return nil, fmt.Errorf("unmarshal trivy json: %w", err)
	}

	var vulns []Vulnerability
	for _, res := range tr.Results {
		for _, v := range res.Vulnerabilities {
			vulns = append(vulns, Vulnerability{
				ID:           v.VulnerabilityID,
				Severity:     strings.ToLower(v.Severity),
				Package:      v.PkgName,
				Version:      v.InstalledVersion,
				FixedVersion: v.FixedVersion,
				Description:  v.Description,
				Link:         v.PrimaryURL,
				Scanner:      "trivy",
			})
		}
	}
	return vulns, nil
}

// grypeResult mirrors the relevant parts of Grype's JSON output.
type grypeResult struct {
	Matches []struct {
		Vulnerability struct {
			ID          string `json:"id"`
			Severity    string `json:"severity"`
			Description string `json:"description"`
			DataSource  string `json:"dataSource"`
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

func parseGrypeOutput(jsonStr string) ([]Vulnerability, error) {
	var gr grypeResult
	if err := json.Unmarshal([]byte(jsonStr), &gr); err != nil {
		return nil, fmt.Errorf("unmarshal grype json: %w", err)
	}

	var vulns []Vulnerability
	for _, m := range gr.Matches {
		fixedVersion := ""
		if len(m.Vulnerability.Fix.Versions) > 0 {
			fixedVersion = strings.Join(m.Vulnerability.Fix.Versions, ", ")
		}
		vulns = append(vulns, Vulnerability{
			ID:           m.Vulnerability.ID,
			Severity:     strings.ToLower(m.Vulnerability.Severity),
			Package:      m.Artifact.Name,
			Version:      m.Artifact.Version,
			FixedVersion: fixedVersion,
			Description:  m.Vulnerability.Description,
			Link:         m.Vulnerability.DataSource,
			Scanner:      "grype",
		})
	}
	return vulns, nil
}

// buildSummary counts vulnerabilities by severity.
func buildSummary(vulns []Vulnerability) ScanSummary {
	var s ScanSummary
	for _, v := range vulns {
		switch v.Severity {
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
	return s
}

// AggregateSummary returns a worst-case (maximum) summary across multiple scan results.
func AggregateSummary(results []ScanResult) ScanSummary {
	var agg ScanSummary
	for _, r := range results {
		if r.Summary.Critical > agg.Critical {
			agg.Critical = r.Summary.Critical
		}
		if r.Summary.High > agg.High {
			agg.High = r.Summary.High
		}
		if r.Summary.Medium > agg.Medium {
			agg.Medium = r.Summary.Medium
		}
		if r.Summary.Low > agg.Low {
			agg.Low = r.Summary.Low
		}
		if r.Summary.Negligible > agg.Negligible {
			agg.Negligible = r.Summary.Negligible
		}
		if r.Summary.Unknown > agg.Unknown {
			agg.Unknown = r.Summary.Unknown
		}
	}
	return agg
}

// EvaluateCriteria checks whether an update should be blocked based on
// the scan results and the chosen blocking policy.
// Returns (blocked bool, reason string).
func EvaluateCriteria(criteria string, newSummary ScanSummary, baseline *ScanSummary) (bool, string) {
	total := newSummary.Critical + newSummary.High + newSummary.Medium +
		newSummary.Low + newSummary.Negligible + newSummary.Unknown

	switch criteria {
	case "never", "":
		return false, ""

	case "any":
		if total > 0 {
			return true, fmt.Sprintf("blocked: %d total vulnerabilities found", total)
		}
		return false, ""

	case "critical_high":
		if newSummary.Critical >= 1 || newSummary.High >= 1 {
			return true, fmt.Sprintf("blocked: %d critical, %d high vulnerabilities found", newSummary.Critical, newSummary.High)
		}
		return false, ""

	case "critical":
		if newSummary.Critical >= 1 {
			return true, fmt.Sprintf("blocked: %d critical vulnerabilities found", newSummary.Critical)
		}
		return false, ""

	case "more_than_current":
		if baseline == nil {
			// No baseline to compare against; allow the update.
			return false, ""
		}
		if newSummary.Critical > baseline.Critical {
			return true, fmt.Sprintf("blocked: critical vulnerabilities increased from %d to %d", baseline.Critical, newSummary.Critical)
		}
		if newSummary.High > baseline.High {
			return true, fmt.Sprintf("blocked: high vulnerabilities increased from %d to %d", baseline.High, newSummary.High)
		}
		return false, ""

	default:
		// Unknown criteria — don't block.
		return false, ""
	}
}
