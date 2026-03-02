package models

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Vulnerability represents a single CVE or security finding from a scanner.
type Vulnerability struct {
	ID           string `json:"id"`
	Severity     string `json:"severity"`             // critical, high, medium, low, negligible, unknown
	Package      string `json:"package"`
	Version      string `json:"version"`
	FixedVersion string `json:"fixedVersion,omitempty"`
	Description  string `json:"description,omitempty"`
	Link         string `json:"link,omitempty"`
	Scanner      string `json:"scanner"` // trivy | grype
}

// ScanSummary holds counts of vulnerabilities grouped by severity.
type ScanSummary struct {
	Critical   int `json:"critical"`
	High       int `json:"high"`
	Medium     int `json:"medium"`
	Low        int `json:"low"`
	Negligible int `json:"negligible"`
	Unknown    int `json:"unknown"`
}

// ScanResult is a full record of one vulnerability scan run against a container.
type ScanResult struct {
	ID              string          `json:"id"`
	ContainerID     string          `json:"containerId"`
	ContainerName   string          `json:"containerName"`
	ImageName       string          `json:"imageName"`
	ImageID         string          `json:"imageId"`
	HostID          string          `json:"hostId"`
	Scanner         string          `json:"scanner"`        // trivy | grype | both
	ScannedAt       time.Time       `json:"scannedAt"`
	ScanDurationMs  int64           `json:"scanDurationMs"`
	Summary         ScanSummary     `json:"summary"`
	Vulnerabilities []Vulnerability `json:"vulnerabilities"`
	Blocked         bool            `json:"blocked"`
	BlockReason     string          `json:"blockReason,omitempty"`
	ForceOverride   bool            `json:"forceOverride"`
	TriggeredBy     string          `json:"triggeredBy"` // manual | auto
}

// ScanStore manages persistence of ScanResult records using a JSON file.
type ScanStore struct {
	mu       sync.RWMutex
	dataDir  string
	results  []ScanResult
	maxItems int
}

// NewScanStore creates a ScanStore backed by <dataDir>/scans.json.
// It loads any previously persisted results on startup.
func NewScanStore(dataDir string) (*ScanStore, error) {
	ss := &ScanStore{
		dataDir:  dataDir,
		maxItems: 500, // keep last 500 scans
	}
	if err := ss.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("load scan store: %w", err)
	}
	return ss, nil
}

func (ss *ScanStore) scanFile() string {
	return filepath.Join(ss.dataDir, "scans.json")
}

func (ss *ScanStore) load() error {
	data, err := os.ReadFile(ss.scanFile())
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &ss.results)
}

func (ss *ScanStore) save() error {
	data, err := json.MarshalIndent(ss.results, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ss.scanFile(), data, 0644)
}

// Save persists a new ScanResult, prepending it so the list is newest-first.
// Older entries beyond maxItems are discarded.
func (ss *ScanStore) Save(result *ScanResult) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	// Prepend (newest first)
	ss.results = append([]ScanResult{*result}, ss.results...)

	// Trim to maxItems
	if len(ss.results) > ss.maxItems {
		ss.results = ss.results[:ss.maxItems]
	}

	return ss.save()
}

// List returns up to limit scan results (newest first).
// If limit <= 0 or greater than the number of stored results, all results are returned.
func (ss *ScanStore) List(limit int) []ScanResult {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	if limit <= 0 || limit > len(ss.results) {
		limit = len(ss.results)
	}
	result := make([]ScanResult, limit)
	copy(result, ss.results[:limit])
	return result
}

// GetLatestForImage returns the most recent ScanResult whose ImageName matches,
// or nil if no matching result exists.
func (ss *ScanStore) GetLatestForImage(imageName string) *ScanResult {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	for _, r := range ss.results {
		if r.ImageName == imageName {
			return &r
		}
	}
	return nil
}
