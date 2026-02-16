package models

// HostConfig represents a Docker host configuration
type HostConfig struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Address string `json:"address"`
	IsLocal bool   `json:"isLocal"`
}

// ContainerInfo represents a Docker container
type ContainerInfo struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Image    string            `json:"image"`
	Status   string            `json:"status"`
	State    string            `json:"state"`
	Created  int64             `json:"created"`
	HostID   string            `json:"hostId"`
	HostName string            `json:"hostName"`
	Ports    []PortMapping     `json:"ports"`
	Labels   map[string]string `json:"labels"`
	Health   string            `json:"health"`
	Networks map[string]string `json:"networks"`
	Volumes  int               `json:"volumes"`
}

// ContainerStats represents container resource usage statistics
type ContainerStats struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	HostID      string  `json:"hostId"`
	CPUPercent  float64 `json:"cpuPercent"`
	MemoryUsage uint64  `json:"memoryUsage"`
	MemoryLimit uint64  `json:"memoryLimit"`
	MemoryPct   float64 `json:"memoryPercent"`
	NetRx       uint64  `json:"networkRx"`
	NetTx       uint64  `json:"networkTx"`
	BlockRead   uint64  `json:"blockRead"`
	BlockWrite  uint64  `json:"blockWrite"`
}

// PortMapping represents a container port mapping
type PortMapping struct {
	Private uint16 `json:"private"`
	Public  uint16 `json:"public"`
	Type    string `json:"type"`
}

// DiskInfo represents disk usage information
type DiskInfo struct {
	MountPoint string `json:"mountPoint"`
	Device     string `json:"device"`
	TotalBytes uint64 `json:"totalBytes"`
	UsedBytes  uint64 `json:"usedBytes"`
	FreeBytes  uint64 `json:"freeBytes"`
}

// HostFileEntry represents a file or directory on a host
type HostFileEntry struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	ModTime int64  `json:"modTime"`
	IsDir   bool   `json:"isDir"`
}

// HostStats represents host-level statistics
type HostStats struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	ContainerCount int        `json:"containerCount"`
	RunningCount   int        `json:"runningCount"`
	CPUPercent     float64    `json:"cpuPercent"`
	MemoryPercent  float64    `json:"memoryPercent"`
	MemoryUsed     uint64     `json:"memoryUsed"`
	MemoryTotal    uint64     `json:"memoryTotal"`
	Online         bool       `json:"online"`
	SSHHost        string     `json:"sshHost"`
	Disks          []DiskInfo `json:"disks"`
}

// Environment represents a managed Docker host configuration
type Environment struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	ConnectionType string `json:"connectionType"` // "socket" or "tcp"
	Address        string `json:"address"`        // socket path or host:port
	Protocol       string `json:"protocol"`       // "http" or "https"
	IsLocal        bool   `json:"isLocal"`
	Labels         string `json:"labels"`
	Status         string `json:"status"` // "online", "offline", "unknown"
	DockerVersion  string `json:"dockerVersion"`
	// Update settings
	AutoUpdate     bool   `json:"autoUpdate"`
	UpdateSchedule string `json:"updateSchedule"` // cron expression
	ImagePrune     bool   `json:"imagePrune"`
	// Feature flags
	EventTracking bool `json:"eventTracking"`
	VulnScanning  bool `json:"vulnScanning"`
}

// ImageUpdate represents container image update information
type ImageUpdate struct {
	ContainerID   string `json:"containerId"`
	ContainerName string `json:"containerName"`
	Image         string `json:"image"`
	HostID        string `json:"hostId"`
	CurrentDigest string `json:"currentDigest"`
	LatestDigest  string `json:"latestDigest,omitempty"`
	CurrentTag    string `json:"currentTag"`
	LatestTag     string `json:"latestTag,omitempty"`
	HasUpdate     bool   `json:"hasUpdate"`
	CheckedAt     int64  `json:"checkedAt"`
}

// SSEMessage represents a Server-Sent Event message
type SSEMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}
