package models

// AppSettings represents application-wide settings persisted to file
type AppSettings struct {
	// Notification thresholds
	CPUThreshold    float64 `json:"cpuThreshold"`
	MemoryThreshold float64 `json:"memoryThreshold"`
	// Apprise configuration
	AppriseURL         string   `json:"appriseUrl"`
	AppriseKey         string   `json:"appriseKey"`
	TelegramEnabled    bool     `json:"telegramEnabled"`
	TelegramURL        string   `json:"telegramUrl"`
	EmailEnabled       bool     `json:"emailEnabled"`
	NotifyOnStop       bool     `json:"notifyOnStop"`
	NotifyOnStart      bool     `json:"notifyOnStart"`
	NotifyOnHighCPU    bool     `json:"notifyOnHighCpu"`
	NotifyOnHighMem    bool     `json:"notifyOnHighMem"`
	AlertsBootstrapped bool     `json:"alertsBootstrapped,omitempty"`
	NotifyTags         []string `json:"notifyTags"`
}

// NotifyRequest represents a notification request
type NotifyRequest struct {
	Title   string `json:"title"`
	Body    string `json:"body"`
	Type    string `json:"type"` // info, success, warning, failure
	Tags    string `json:"tags,omitempty"`
	Channel string `json:"channel,omitempty"` // telegram, email, both, or empty for all
}
