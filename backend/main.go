package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	fbRecover "github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/websocket/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/pkg/sftp"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/ssh"
)

// =============================================================================
// Configuration
// =============================================================================

type HostConfig struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Address string `json:"address"`
	IsLocal bool   `json:"isLocal"`
}

// App Settings - persisted to file
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

var hosts = parseHostsConfig(getEnvOrDefault("DOCKER_HOSTS",
	"raspi1:Raspeberry Main:unix:///var/run/docker.sock:local"))

// Host health tracking for backoff on unreachable hosts
var (
	hostHealth   = make(map[string]time.Time) // hostID -> last failure time
	hostHealthMu sync.RWMutex
	hostBackoff  = 30 * time.Second // Skip hosts that failed within this window
)

// parseHostsConfig parses DOCKER_HOSTS env var.
// Format: "id:name:address:local/remote" separated by "|"
// Address may contain colons (e.g., unix:///var/run/docker.sock, http://host:port)
// so we split from the end to extract the local/remote flag.
func parseHostsConfig(config string) []HostConfig {
	var result []HostConfig
	entries := strings.Split(config, "|")
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		// Find last colon to extract the local/remote flag
		lastColon := strings.LastIndex(entry, ":")
		if lastColon < 0 {
			log.Printf("Warning: invalid host config entry: %s", entry)
			continue
		}
		isLocalStr := entry[lastColon+1:]
		rest := entry[:lastColon]

		// Now split the rest into id:name:address
		// Find first two colons for id and name, rest is address
		firstColon := strings.Index(rest, ":")
		if firstColon < 0 {
			log.Printf("Warning: invalid host config entry: %s", entry)
			continue
		}
		id := rest[:firstColon]
		afterID := rest[firstColon+1:]

		secondColon := strings.Index(afterID, ":")
		if secondColon < 0 {
			log.Printf("Warning: invalid host config entry: %s", entry)
			continue
		}
		name := afterID[:secondColon]
		address := afterID[secondColon+1:]

		isLocal := strings.EqualFold(isLocalStr, "local") || strings.EqualFold(isLocalStr, "true")
		result = append(result, HostConfig{
			ID:      id,
			Name:    name,
			Address: address,
			IsLocal: isLocal,
		})
	}
	if len(result) == 0 {
		log.Println("Warning: no valid hosts configured, defaulting to local socket")
		result = append(result, HostConfig{
			ID:      "local",
			Name:    "Local",
			Address: "unix:///var/run/docker.sock",
			IsLocal: true,
		})
	}
	return result
}

func deriveSSHHost(h HostConfig) string {
	if h.IsLocal {
		// If Address is an IP, use it; else fallback to host.docker.internal
		// `host.docker.internal` is provided via extra_hosts mapping in compose
		addr := strings.TrimSpace(h.Address)
		if addr != "" && net.ParseIP(addr) != nil {
			return addr
		}
		return "host.docker.internal"
	}
	addr := strings.TrimSpace(h.Address)
	if addr == "" {
		return h.ID
	}
	if parts := strings.SplitN(addr, "://", 2); len(parts) == 2 {
		addr = parts[1]
	}
	if parts := strings.SplitN(addr, "/", 2); len(parts) == 2 {
		addr = parts[0]
	}
	if parts := strings.SplitN(addr, "@", 2); len(parts) == 2 {
		addr = parts[1]
	}
	if parts := strings.SplitN(addr, ":", 2); len(parts) == 2 {
		addr = parts[0]
	}
	if addr == "" {
		return h.ID
	}
	return addr
}

// deriveSSHCandidates returns a list of hostname/IP candidates to try when dialing SSH.
// This allows the backend to attempt multiple fallbacks (explicit IP, host.docker.internal,
// plain id, etc.) which helps in containerized environments where DNS/resolution may differ.
func deriveSSHCandidates(h HostConfig) []string {
	var cands []string
	addr := strings.TrimSpace(h.Address)

	if h.IsLocal {
		// If Address is an IP, try it first
		if addr != "" && net.ParseIP(addr) != nil {
			cands = append(cands, addr)
		}
		// Always try host.docker.internal as a fallback (provided via extra_hosts)
		cands = append(cands, "host.docker.internal")
	} else {
		// Try to extract host from address (strip scheme, path, user)
		if addr != "" {
			if parts := strings.SplitN(addr, "://", 2); len(parts) == 2 {
				addr = parts[1]
			}
			if parts := strings.SplitN(addr, "/", 2); len(parts) == 2 {
				addr = parts[0]
			}
			if parts := strings.SplitN(addr, "@", 2); len(parts) == 2 {
				addr = parts[1]
			}
			if parts := strings.SplitN(addr, ":", 2); len(parts) == 2 {
				addr = parts[0]
			}
			if addr != "" {
				cands = append(cands, addr)
			}
		}
		// Always also try the host ID (may resolve via internal DNS)
		cands = append(cands, h.ID)
	}

	// Ensure uniqueness while preserving order
	seen := make(map[string]struct{})
	uniq := make([]string, 0, len(cands))
	for _, s := range cands {
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		uniq = append(uniq, s)
	}
	return uniq
}

func getHostConfigByID(hostID string) *HostConfig {
	for _, h := range hosts {
		if h.ID == hostID {
			return &h
		}
	}
	return nil
}

func isHostHealthy(hostID string) bool {
	hostHealthMu.RLock()
	defer hostHealthMu.RUnlock()
	lastFail, exists := hostHealth[hostID]
	if !exists {
		return true
	}
	return time.Since(lastFail) > hostBackoff
}

func markHostFailed(hostID string) {
	hostHealthMu.Lock()
	hostHealth[hostID] = time.Now()
	hostHealthMu.Unlock()
}

func markHostHealthy(hostID string) {
	hostHealthMu.Lock()
	delete(hostHealth, hostID)
	hostHealthMu.Unlock()
}

// Watchtower configuration
var (
	watchtowerToken = getEnvOrDefault("WATCHTOWER_TOKEN", "")
	watchtowerURLs  = parseWatchtowerURLs(getEnvOrDefault("WATCHTOWER_URLS", ""))
)

func parseWatchtowerURLs(config string) map[string]string {
	result := make(map[string]string)
	if config == "" {
		return result
	}
	entries := strings.Split(config, "|")
	for _, entry := range entries {
		parts := strings.SplitN(entry, ":", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}

var (
	jwtSecret      = []byte(getEnvOrDefault("JWT_SECRET", generateSecret()))
	dataDir        = getEnvOrDefault("DATA_DIR", "/data")
	defaultAdmin   = getEnvOrDefault("ADMIN_USER", "admin")
	defaultPass    = getEnvOrDefault("ADMIN_PASS", "admin123")
	appriseURL     = getEnvOrDefault("APPRISE_URL", "https://apprise.nerdslabs.com")
	smtp2goAPIKey  = getEnvOrDefault("SMTP2GO_API_KEY", "api-0BCAE7D34EA545FE9041EDA3EEF6C8DD")
	smtpFrom       = getEnvOrDefault("SMTP_FROM", "docker-verse@nerdslabs.com")
	smtpSenderName = getEnvOrDefault("SMTP_SENDER_NAME", "DockerVerse")
	sshUser        = getEnvOrDefault("SSH_USER", "pi")
	sshPort        = getEnvOrDefault("SSH_PORT", "22")
	sshKeyPath     = getEnvOrDefault("SSH_KEY_PATH", "/data/ssh/id_rsa")
	sshKeyPass     = getEnvOrDefault("SSH_KEY_PASSPHRASE", "")
)

var (
	sshAuthOnce   sync.Once
	sshAuthMethod ssh.AuthMethod
	sshAuthErr    error
)

func getSSHAuthMethod() (ssh.AuthMethod, error) {
	sshAuthOnce.Do(func() {
		keyData, err := os.ReadFile(sshKeyPath)
		if err != nil {
			sshAuthErr = fmt.Errorf("read ssh key: %w", err)
			return
		}
		var signer ssh.Signer
		if sshKeyPass != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(keyData, []byte(sshKeyPass))
		} else {
			signer, err = ssh.ParsePrivateKey(keyData)
		}
		if err != nil {
			sshAuthErr = fmt.Errorf("parse ssh key: %w", err)
			return
		}
		sshAuthMethod = ssh.PublicKeys(signer)
	})
	return sshAuthMethod, sshAuthErr
}

func dialSSH(hostID string) (*ssh.Client, error) {
	h := getHostConfigByID(hostID)
	if h == nil {
		return nil, fmt.Errorf("unknown host: %s", hostID)
	}
	candidates := deriveSSHCandidates(*h)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("ssh host not configured for %s", hostID)
	}
	authMethod, err := getSSHAuthMethod()
	if err != nil {
		return nil, err
	}
	config := &ssh.ClientConfig{
		User:            sshUser,
		Auth:            []ssh.AuthMethod{authMethod},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         8 * time.Second,
	}
	var lastErr error
	for _, cand := range candidates {
		addr := net.JoinHostPort(cand, sshPort)
		log.Printf("dialSSH: hostID=%s trying candidate=%s addr=%s", hostID, cand, addr)
		conn, err := net.DialTimeout("tcp", addr, 8*time.Second)
		if err != nil {
			log.Printf("dialSSH: candidate failed hostID=%s addr=%s err=%v", hostID, addr, err)
			lastErr = err
			continue
		}
		clientConn, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
		if err != nil {
			log.Printf("dialSSH: ssh handshake failed hostID=%s addr=%s err=%v", hostID, addr, err)
			lastErr = err
			conn.Close()
			continue
		}
		log.Printf("dialSSH: hostID=%s connected via %s", hostID, addr)
		return ssh.NewClient(clientConn, chans, reqs), nil
	}
	if lastErr != nil {
		return nil, fmt.Errorf("all ssh candidates failed for %s: last error: %w", hostID, lastErr)
	}
	return nil, fmt.Errorf("no ssh candidates available for %s", hostID)
}

func runSSHCommand(hostID, cmd string) (string, error) {
	client, err := dialSSH(hostID)
	if err != nil {
		return "", err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	var buf bytes.Buffer
	session.Stdout = &buf
	session.Stderr = &buf
	if err := session.Run(cmd); err != nil {
		return buf.String(), err
	}
	return buf.String(), nil
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// isAdminRequest returns true if the request is from localhost or contains a valid admin JWT.
func isAdminRequest(c *fiber.Ctx) bool {
	ip := c.IP()
	if ip == "127.0.0.1" || ip == "::1" || strings.HasPrefix(ip, "::ffff:127.0.0.1") {
		return true
	}
	// Check token
	auth := c.Get("Authorization")
	var tokenStr string
	if auth != "" {
		tokenStr = strings.TrimPrefix(auth, "Bearer ")
	} else {
		tokenStr = c.Query("token")
	}
	if tokenStr == "" {
		return false
	}
	claims, err := validateToken(tokenStr)
	if err != nil {
		return false
	}
	return strings.EqualFold(claims.Role, "admin")
}

// tailFileLines returns the last `n` lines from the specified file.
func tailFileLines(path string, n int) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open log file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	buf := make([]string, 0, n)
	for scanner.Scan() {
		line := scanner.Text()
		if len(buf) < n {
			buf = append(buf, line)
		} else {
			// rotate
			copy(buf[0:], buf[1:])
			buf[n-1] = line
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("read log file: %w", err)
	}
	return strings.Join(buf, "\n"), nil
}

func generateSecret() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

func maskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "***@***"
	}

	local := parts[0]
	domain := parts[1]

	// Mask local part: show first 3 and last 2 chars
	var maskedLocal string
	if len(local) <= 3 {
		maskedLocal = local[:1] + "***"
	} else if len(local) <= 5 {
		maskedLocal = local[:2] + "***" + local[len(local)-1:]
	} else {
		maskedLocal = local[:3] + "****" + local[len(local)-2:]
	}

	return maskedLocal + "@" + domain
}

// =============================================================================
// Types
// =============================================================================

type User struct {
	ID             string    `json:"id"`
	Username       string    `json:"username"`
	Email          string    `json:"email"`
	FirstName      string    `json:"firstName"`
	LastName       string    `json:"lastName"`
	Avatar         string    `json:"avatar,omitempty"` // Base64 encoded avatar or URL
	PasswordHash   string    `json:"passwordHash,omitempty"`
	Role           string    `json:"role"` // "admin" or "user"
	EmailConfirmed bool      `json:"emailConfirmed"`
	ConfirmToken   string    `json:"-"`
	TOTPSecret     string    `json:"-"`           // TOTP secret (not exposed to API)
	TOTPEnabled    bool      `json:"totpEnabled"` // Whether 2FA is enabled
	RecoveryCodes  []string  `json:"-"`           // Recovery codes (not exposed to API)
	CreatedAt      time.Time `json:"createdAt"`
	LastLogin      time.Time `json:"lastLogin,omitempty"`
}

type ResetCode struct {
	Code      string
	UserID    string
	ExpiresAt time.Time
}

type UserStore struct {
	Users      map[string]*User `json:"users"`
	Settings   AppSettings      `json:"settings"`
	mu         sync.RWMutex
	resetCodes map[string]*ResetCode // email -> reset code (not persisted)
}

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
	Stack    string            `json:"stack"`
	Service  string            `json:"service"`
}

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

type PortMapping struct {
	Private uint16 `json:"private"`
	Public  uint16 `json:"public"`
	Type    string `json:"type"`
}

type DiskInfo struct {
	MountPoint string `json:"mountPoint"`
	Device     string `json:"device"`
	TotalBytes uint64 `json:"totalBytes"`
	UsedBytes  uint64 `json:"usedBytes"`
	FreeBytes  uint64 `json:"freeBytes"`
}

type HostFileEntry struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	ModTime int64  `json:"modTime"`
	IsDir   bool   `json:"isDir"`
}

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

type SSEMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// Environment configuration for managed Docker hosts
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

// EnvironmentStore manages environment persistence
type EnvironmentStore struct {
	mu           sync.RWMutex
	Environments map[string]*Environment `json:"environments"`
	filePath     string
}

func NewEnvironmentStore(filePath string) *EnvironmentStore {
	store := &EnvironmentStore{
		Environments: make(map[string]*Environment),
		filePath:     filePath,
	}
	store.load()
	return store
}

func (es *EnvironmentStore) load() {
	data, err := os.ReadFile(es.filePath)
	if err != nil {
		// If file doesn't exist, initialize from DOCKER_HOSTS
		es.migrateFromHosts()
		return
	}
	if err := json.Unmarshal(data, &es.Environments); err != nil {
		log.Printf("Warning: failed to parse environments file: %v", err)
		es.migrateFromHosts()
	}
}

func (es *EnvironmentStore) migrateFromHosts() {
	for _, h := range hosts {
		connType := "tcp"
		addr := h.Address
		protocol := "http"
		if h.IsLocal {
			connType = "socket"
			addr = h.Address
		}
		es.Environments[h.ID] = &Environment{
			ID:             h.ID,
			Name:           h.Name,
			ConnectionType: connType,
			Address:        addr,
			Protocol:       protocol,
			IsLocal:        h.IsLocal,
			Status:         "unknown",
			EventTracking:  true,
		}
	}
	es.save()
}

func (es *EnvironmentStore) save() {
	data, err := json.MarshalIndent(es.Environments, "", "  ")
	if err != nil {
		log.Printf("Error marshaling environments: %v", err)
		return
	}
	if err := os.WriteFile(es.filePath, data, 0644); err != nil {
		log.Printf("Error saving environments: %v", err)
	}
}

func (es *EnvironmentStore) GetAll() []*Environment {
	es.mu.RLock()
	defer es.mu.RUnlock()
	result := make([]*Environment, 0, len(es.Environments))
	for _, env := range es.Environments {
		result = append(result, env)
	}
	return result
}

func (es *EnvironmentStore) Get(id string) *Environment {
	es.mu.RLock()
	defer es.mu.RUnlock()
	return es.Environments[id]
}

func (es *EnvironmentStore) Create(env *Environment) {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.Environments[env.ID] = env
	es.save()
}

func (es *EnvironmentStore) Update(env *Environment) {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.Environments[env.ID] = env
	es.save()
}

func (es *EnvironmentStore) Delete(id string) {
	es.mu.Lock()
	defer es.mu.Unlock()
	delete(es.Environments, id)
	es.save()
}

type JWTClaims struct {
	UserID   string `json:"userId"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

type LoginRequest struct {
	Username     string `json:"username"`
	Password     string `json:"password"`
	RememberMe   bool   `json:"rememberMe"`
	TOTPCode     string `json:"totpCode,omitempty"`
	RecoveryCode string `json:"recoveryCode,omitempty"`
}

type LoginResponse struct {
	User         *User      `json:"user"`
	Tokens       AuthTokens `json:"tokens"`
	RequiresTOTP bool       `json:"requiresTOTP,omitempty"`
	TempToken    string     `json:"tempToken,omitempty"` // Temporary token for 2FA verification
}

type AuthTokens struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int    `json:"expiresIn"`
	TokenType    string `json:"tokenType"`
}

type CreateUserRequest struct {
	Username  string `json:"username"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Role      string `json:"role"`
}

type UpdateUserRequest struct {
	Email     string `json:"email,omitempty"`
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
	Password  string `json:"password,omitempty"`
	Role      string `json:"role,omitempty"`
}

type NotifyRequest struct {
	Title   string `json:"title"`
	Body    string `json:"body"`
	Type    string `json:"type"` // info, success, warning, failure
	Tags    string `json:"tags,omitempty"`
	Channel string `json:"channel,omitempty"` // telegram, email, both, or empty for all
}

// Image update tracking
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

// Update check cache
var (
	updateCache   = make(map[string]*ImageUpdate) // containerID -> update info
	updateCacheMu sync.RWMutex
)

// =============================================================================
// User Store
// =============================================================================

func NewUserStore() *UserStore {
	store := &UserStore{
		Users:      make(map[string]*User),
		resetCodes: make(map[string]*ResetCode),
		Settings: AppSettings{
			CPUThreshold:       80.0,
			MemoryThreshold:    80.0,
			AppriseURL:         appriseURL,
			AppriseKey:         "dockerverse",
			NotifyOnStop:       true,
			NotifyOnHighCPU:    false,
			NotifyOnHighMem:    false,
			AlertsBootstrapped: true,
		},
	}
	store.load()

	// Ensure admin user exists with valid password
	needsSave := false
	if len(store.Users) == 0 {
		hash, _ := bcrypt.GenerateFromPassword([]byte(defaultPass), bcrypt.DefaultCost)
		store.Users["admin"] = &User{
			ID:           "admin",
			Username:     defaultAdmin,
			Email:        "admin@dockerverse.local",
			FirstName:    "Admin",
			LastName:     "User",
			PasswordHash: string(hash),
			Role:         "admin",
			CreatedAt:    time.Now(),
		}
		needsSave = true
	} else {
		// Fix any users with missing password hash (recovery from old bug)
		for _, user := range store.Users {
			if user.PasswordHash == "" {
				hash, _ := bcrypt.GenerateFromPassword([]byte(defaultPass), bcrypt.DefaultCost)
				user.PasswordHash = string(hash)
				needsSave = true
				log.Printf("[WARN] Recovered missing password for user: %s (reset to default)", user.Username)
			}
		}
	}

	if needsSave {
		store.save()
	}

	return store
}

func (s *UserStore) load() {
	path := dataDir + "/users.json"
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	json.Unmarshal(data, s)
	if !s.Settings.AlertsBootstrapped {
		s.Settings.NotifyOnHighCPU = false
		s.Settings.NotifyOnHighMem = false
		s.Settings.AlertsBootstrapped = true
		s.save()
	}
}

func (s *UserStore) save() {
	os.MkdirAll(dataDir, 0755)
	path := dataDir + "/users.json"
	data, _ := json.MarshalIndent(s, "", "  ")
	os.WriteFile(path, data, 0644)
}

func (s *UserStore) GetUser(username string) *User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Case-insensitive username lookup
	usernameLower := strings.ToLower(username)
	for _, u := range s.Users {
		if strings.ToLower(u.Username) == usernameLower {
			return u
		}
	}
	return nil
}

// SafeUser returns a copy without sensitive fields for API responses
func (u *User) SafeUser() *User {
	if u == nil {
		return nil
	}
	return &User{
		ID:             u.ID,
		Username:       u.Username,
		Email:          u.Email,
		FirstName:      u.FirstName,
		LastName:       u.LastName,
		Avatar:         u.Avatar,
		PasswordHash:   "", // Never expose in API
		Role:           u.Role,
		EmailConfirmed: u.EmailConfirmed,
		TOTPEnabled:    u.TOTPEnabled,
		CreatedAt:      u.CreatedAt,
		LastLogin:      u.LastLogin,
	}
}

func (s *UserStore) GetAllUsers() []*User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	users := make([]*User, 0, len(s.Users))
	for _, u := range s.Users {
		users = append(users, u)
	}
	return users
}

func (s *UserStore) GetUserByEmail(email string) *User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.Users {
		if u.Email == email {
			return u
		}
	}
	return nil
}

func (s *UserStore) SetResetCode(email, code, userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.resetCodes[email] = &ResetCode{
		Code:      code,
		UserID:    userID,
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}
}

func (s *UserStore) VerifyResetCode(email, code string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rc, exists := s.resetCodes[email]
	if !exists || rc.Code != code || time.Now().After(rc.ExpiresAt) {
		return "", false
	}
	return rc.UserID, true
}

func (s *UserStore) ClearResetCode(email string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.resetCodes, email)
}

func (s *UserStore) UpdatePassword(userID, newPassword string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, exists := s.Users[userID]
	if !exists {
		return fmt.Errorf("user not found")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.PasswordHash = string(hash)
	s.save()
	return nil
}

func (s *UserStore) CreateUser(req CreateUserRequest) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.Users[req.Username]; exists {
		return nil, fmt.Errorf("user already exists")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Generate confirmation token
	tokenBytes := make([]byte, 32)
	rand.Read(tokenBytes)
	confirmToken := base64.URLEncoding.EncodeToString(tokenBytes)

	user := &User{
		ID:             req.Username,
		Username:       req.Username,
		Email:          req.Email,
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		PasswordHash:   string(hash),
		Role:           req.Role,
		EmailConfirmed: false,
		ConfirmToken:   confirmToken,
		CreatedAt:      time.Now(),
	}

	s.Users[req.Username] = user
	s.save()
	return user, nil
}

func (s *UserStore) ConfirmEmail(token string) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, user := range s.Users {
		if user.ConfirmToken == token && !user.EmailConfirmed {
			user.EmailConfirmed = true
			user.ConfirmToken = "" // Clear the token
			s.save()
			return user, nil
		}
	}
	return nil, fmt.Errorf("invalid or expired confirmation token")
}

func (s *UserStore) UpdateUser(username string, req UpdateUserRequest) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.Users[username]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}

	if req.Email != "" {
		user.Email = req.Email
	}
	if req.FirstName != "" {
		user.FirstName = req.FirstName
	}
	if req.LastName != "" {
		user.LastName = req.LastName
	}
	if req.Role != "" {
		user.Role = req.Role
	}
	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		user.PasswordHash = string(hash)
	}

	s.save()
	return user, nil
}

func (s *UserStore) DeleteUser(username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if username == "admin" {
		return fmt.Errorf("cannot delete admin user")
	}

	if _, exists := s.Users[username]; !exists {
		return fmt.Errorf("user not found")
	}

	delete(s.Users, username)
	s.save()
	return nil
}

func (s *UserStore) ValidateLogin(username, password string) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Case-insensitive username lookup: find user by comparing lowercase
	var user *User
	usernameLower := strings.ToLower(username)
	for _, u := range s.Users {
		if strings.ToLower(u.Username) == usernameLower {
			user = u
			break
		}
	}

	if user == nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	user.LastLogin = time.Now()
	s.save()
	return user, nil
}

func (s *UserStore) UpdateSettings(settings AppSettings) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Settings = settings
	s.save()
}

func (s *UserStore) GetSettings() AppSettings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Settings
}

// TOTP Functions
func (s *UserStore) SetupTOTP(username string) (secret string, url string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.Users[username]
	if !exists {
		return "", "", fmt.Errorf("user not found")
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "DockerVerse",
		AccountName: user.Email,
		SecretSize:  32,
		Algorithm:   otp.AlgorithmSHA1,
	})
	if err != nil {
		return "", "", err
	}

	// Store secret (not yet enabled until confirmed)
	user.TOTPSecret = key.Secret()
	s.save()

	return key.Secret(), key.URL(), nil
}

func (s *UserStore) EnableTOTP(username string, code string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.Users[username]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}

	if user.TOTPSecret == "" {
		return nil, fmt.Errorf("TOTP not set up")
	}

	// Validate the code
	if !totp.Validate(code, user.TOTPSecret) {
		return nil, fmt.Errorf("invalid code")
	}

	// Generate recovery codes
	recoveryCodes := make([]string, 10)
	for i := 0; i < 10; i++ {
		code := make([]byte, 8)
		rand.Read(code)
		recoveryCodes[i] = fmt.Sprintf("%x", code)[:16]
	}

	user.TOTPEnabled = true
	user.RecoveryCodes = recoveryCodes
	s.save()

	return recoveryCodes, nil
}

func (s *UserStore) DisableTOTP(username string, password string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.Users[username]
	if !exists {
		return fmt.Errorf("user not found")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return fmt.Errorf("invalid password")
	}

	user.TOTPSecret = ""
	user.TOTPEnabled = false
	user.RecoveryCodes = nil
	s.save()

	return nil
}

func (s *UserStore) UseRecoveryCode(username string, code string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.Users[username]
	if !exists {
		return false, fmt.Errorf("user not found")
	}

	// Find and remove the recovery code
	for i, rc := range user.RecoveryCodes {
		if rc == code {
			// Remove the used code
			user.RecoveryCodes = append(user.RecoveryCodes[:i], user.RecoveryCodes[i+1:]...)
			s.save()
			return true, nil
		}
	}

	return false, nil
}

func (s *UserStore) GetRecoveryCodesCount(username string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.Users[username]
	if !exists {
		return 0
	}
	return len(user.RecoveryCodes)
}

func (s *UserStore) RegenerateRecoveryCodes(username string, password string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.Users[username]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid password")
	}

	if !user.TOTPEnabled {
		return nil, fmt.Errorf("TOTP not enabled")
	}

	// Generate new recovery codes
	recoveryCodes := make([]string, 10)
	for i := 0; i < 10; i++ {
		code := make([]byte, 8)
		rand.Read(code)
		recoveryCodes[i] = fmt.Sprintf("%x", code)[:16]
	}

	user.RecoveryCodes = recoveryCodes
	s.save()

	return recoveryCodes, nil
}

// Generate a temporary token for 2FA verification step
func generateTempToken(username string) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   username,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)), // Short-lived
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Issuer:    "dockerverse-2fa",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// =============================================================================
// JWT Functions
// =============================================================================

func generateTokens(user *User, remember bool) (*AuthTokens, error) {
	expiresIn := 24 * time.Hour
	if remember {
		expiresIn = 7 * 24 * time.Hour
	}

	claims := JWTClaims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "dockerverse",
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessTokenStr, err := accessToken.SignedString(jwtSecret)
	if err != nil {
		return nil, err
	}

	// Refresh token (longer lived)
	refreshClaims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * 24 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Subject:   user.ID,
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenStr, err := refreshToken.SignedString(jwtSecret)
	if err != nil {
		return nil, err
	}

	return &AuthTokens{
		AccessToken:  accessTokenStr,
		RefreshToken: refreshTokenStr,
		ExpiresIn:    int(expiresIn.Seconds()),
		TokenType:    "Bearer",
	}, nil
}

func validateToken(tokenStr string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, fmt.Errorf("invalid token")
}

// JWT Middleware
func jwtMiddleware(store *UserStore) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var tokenStr string

		// First try Authorization header
		auth := c.Get("Authorization")
		if auth != "" {
			tokenStr = strings.TrimPrefix(auth, "Bearer ")
		} else {
			// Fall back to query parameter (for SSE/EventSource which doesn't support headers)
			tokenStr = c.Query("token")
		}

		if tokenStr == "" {
			return c.Status(401).JSON(fiber.Map{"error": "missing authorization"})
		}

		claims, err := validateToken(tokenStr)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{"error": "invalid token"})
		}

		user := store.GetUser(claims.Username)
		if user == nil {
			return c.Status(401).JSON(fiber.Map{"error": "user not found"})
		}

		c.Locals("user", user)
		c.Locals("claims", claims)
		return c.Next()
	}
}

// Admin only middleware
func adminOnly() fiber.Handler {
	return func(c *fiber.Ctx) error {
		user := c.Locals("user").(*User)
		if user.Role != "admin" {
			return c.Status(403).JSON(fiber.Map{"error": "admin access required"})
		}
		return c.Next()
	}
}

// =============================================================================
// Apprise Notification Service
// =============================================================================

type NotificationService struct {
	store        *UserStore
	lastNotified map[string]time.Time
	mu           sync.Mutex
}

func NewNotificationService(store *UserStore) *NotificationService {
	return &NotificationService{
		store:        store,
		lastNotified: make(map[string]time.Time),
	}
}

func (n *NotificationService) Send(title, body, notifyType string) error {
	return n.SendWithChannel(title, body, notifyType, "all")
}

func (n *NotificationService) SendWithChannel(title, body, notifyType, channel string) error {
	settings := n.store.GetSettings()
	if settings.AppriseURL == "" {
		return fmt.Errorf("apprise not configured")
	}

	// Use stateless API if we have direct URLs configured
	urls := []string{}

	if (channel == "telegram" || channel == "all" || channel == "both") && settings.TelegramEnabled && settings.TelegramURL != "" {
		urls = append(urls, settings.TelegramURL)
	}

	if (channel == "email" || channel == "all" || channel == "both") && settings.EmailEnabled {
		// Email is handled separately via SMTP2Go - we just mark it
	}

	// Build formatted body for Telegram (Markdown)
	formattedBody := n.formatTelegramMessage(title, body, notifyType)

	// If we have direct URLs, use stateless API
	if len(urls) > 0 {
		url := fmt.Sprintf("%s/notify", settings.AppriseURL)

		payload := map[string]interface{}{
			"urls":   urls,
			"title":  title,
			"body":   formattedBody,
			"type":   notifyType,
			"format": "markdown",
		}

		jsonData, _ := json.Marshal(payload)
		log.Printf("Sending notification via stateless API: %s", url)

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 15 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		if resp.StatusCode >= 400 {
			return fmt.Errorf("apprise error (%d): %s", resp.StatusCode, string(respBody))
		}

		log.Printf("Notification sent successfully via %v", urls)
		return nil
	}

	// Fallback to stateful API using tags
	url := fmt.Sprintf("%s/notify/%s", settings.AppriseURL, settings.AppriseKey)

	tags := []string{}
	if channel == "telegram" || (channel == "all" && settings.TelegramEnabled) {
		tags = append(tags, "telegram")
	}
	if channel == "email" || (channel == "all" && settings.EmailEnabled) {
		tags = append(tags, "email")
	}
	if len(tags) == 0 {
		tags = append(tags, "all")
	}
	if len(settings.NotifyTags) > 0 && channel == "all" {
		tags = settings.NotifyTags
	}

	payload := map[string]interface{}{
		"title":  title,
		"body":   formattedBody,
		"type":   notifyType,
		"format": "markdown",
		"tag":    strings.Join(tags, ","),
	}

	jsonData, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("apprise error: %s", string(body))
	}

	return nil
}

func (n *NotificationService) formatTelegramMessage(title, body, notifyType string) string {
	// Modern Telegram message template with Markdown
	icon := "ℹ️"
	switch notifyType {
	case "success":
		icon = "✅"
	case "warning":
		icon = "⚠️"
	case "failure", "error":
		icon = "🚨"
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")

	template := `%s *%s*

%s

━━━━━━━━━━━━━━━
🕐 %s
📡 _DockerVerse_`

	return fmt.Sprintf(template, icon, title, body, timestamp)
}

func (n *NotificationService) ConfigureApprise() error {
	settings := n.store.GetSettings()
	if settings.AppriseURL == "" {
		return fmt.Errorf("apprise URL not configured")
	}

	// Add URLs to Apprise stateful storage
	url := fmt.Sprintf("%s/add/%s", settings.AppriseURL, settings.AppriseKey)

	urls := []string{}

	// Add Telegram URL if configured
	if settings.TelegramEnabled && settings.TelegramURL != "" {
		urls = append(urls, settings.TelegramURL)
	}

	if len(urls) == 0 {
		return nil // Nothing to configure
	}

	payload := map[string]interface{}{
		"urls": urls,
	}

	jsonData, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("apprise config error: %s", string(body))
	}

	return nil
}

func (n *NotificationService) shouldNotify(key string) bool {
	n.mu.Lock()
	defer n.mu.Unlock()

	lastTime, exists := n.lastNotified[key]
	if !exists || time.Since(lastTime) > 5*time.Minute {
		n.lastNotified[key] = time.Now()
		return true
	}
	return false
}

func (n *NotificationService) NotifyContainerState(containerName, state string) {
	settings := n.store.GetSettings()

	if state == "stopped" && settings.NotifyOnStop {
		key := "stop:" + containerName
		if n.shouldNotify(key) {
			n.Send(
				"🛑 Container Stopped",
				fmt.Sprintf("Container '%s' has stopped", containerName),
				"warning",
			)
		}
	} else if state == "started" && settings.NotifyOnStart {
		key := "start:" + containerName
		if n.shouldNotify(key) {
			n.Send(
				"🟢 Container Started",
				fmt.Sprintf("Container '%s' has started", containerName),
				"success",
			)
		}
	}
}

func (n *NotificationService) NotifyHighResource(containerName string, cpu, memory float64) {
	settings := n.store.GetSettings()

	if cpu > settings.CPUThreshold && settings.NotifyOnHighCPU {
		key := "cpu:" + containerName
		if n.shouldNotify(key) {
			n.Send(
				"⚠️ High CPU Usage",
				fmt.Sprintf("Container '%s' CPU at %.1f%% (threshold: %.0f%%)", containerName, cpu, settings.CPUThreshold),
				"warning",
			)
		}
	}

	if memory > settings.MemoryThreshold && settings.NotifyOnHighMem {
		key := "mem:" + containerName
		if n.shouldNotify(key) {
			n.Send(
				"⚠️ High Memory Usage",
				fmt.Sprintf("Container '%s' Memory at %.1f%% (threshold: %.0f%%)", containerName, memory, settings.MemoryThreshold),
				"warning",
			)
		}
	}
}

// =============================================================================
// Email Service (SMTP2Go API)
// =============================================================================

type EmailService struct {
	apiKey     string
	from       string
	senderName string
}

type SMTP2GoRequest struct {
	APIKey   string   `json:"api_key"`
	To       []string `json:"to"`
	Sender   string   `json:"sender"`
	Subject  string   `json:"subject"`
	HTMLBody string   `json:"html_body"`
}

type SMTP2GoResponse struct {
	RequestID string `json:"request_id"`
	Data      struct {
		Succeeded int      `json:"succeeded"`
		Failed    int      `json:"failed"`
		Failures  []string `json:"failures"`
		EmailID   string   `json:"email_id"`
	} `json:"data"`
}

func NewEmailService() *EmailService {
	return &EmailService{
		apiKey:     smtp2goAPIKey,
		from:       smtpFrom,
		senderName: smtpSenderName,
	}
}

func (e *EmailService) Send(to, subject, htmlBody string) error {
	url := "https://api.smtp2go.com/v3/email/send"

	reqBody := SMTP2GoRequest{
		APIKey:   e.apiKey,
		To:       []string{to},
		Sender:   fmt.Sprintf("%s <%s>", e.senderName, e.from),
		Subject:  subject,
		HTMLBody: htmlBody,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		log.Printf("Email JSON marshal error: %v", err)
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Email send HTTP error: %v", err)
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var apiResp SMTP2GoResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		log.Printf("Email API response parse error: %v, body: %s", err, string(body))
		return err
	}

	if resp.StatusCode != 200 || apiResp.Data.Failed > 0 {
		log.Printf("Email send failed: status=%d, response=%s", resp.StatusCode, string(body))
		return fmt.Errorf("email send failed: %s", string(body))
	}

	log.Printf("Email sent to %s: %s (email_id: %s)", to, subject, apiResp.Data.EmailID)
	return nil
}

func (e *EmailService) SendWelcome(user *User) error {
	subject := "Welcome to DockerVerse! 🐳"
	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; padding: 20px; }
        .container { max-width: 600px; margin: 0 auto; background: white; border-radius: 12px; padding: 40px; box-shadow: 0 4px 6px rgba(0,0,0,0.1); }
        h1 { color: #3b82f6; margin-bottom: 20px; }
        p { color: #333; line-height: 1.6; }
        .highlight { background: #e0f2fe; padding: 15px; border-radius: 8px; margin: 20px 0; }
        .footer { color: #666; font-size: 12px; margin-top: 30px; border-top: 1px solid #eee; padding-top: 20px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🐳 Welcome to DockerVerse!</h1>
        <p>Hi <strong>%s</strong>,</p>
        <p>Your account has been created successfully. You can now access the DockerVerse dashboard to manage all your Docker containers across multiple hosts.</p>
        <div class="highlight">
            <strong>Your username:</strong> %s<br>
            <strong>Your role:</strong> %s
        </div>
        <p>Features available to you:</p>
        <ul>
            <li>View and manage containers</li>
            <li>Monitor CPU and memory usage</li>
            <li>Access container logs and terminal</li>
            <li>Configure notification alerts</li>
        </ul>
        <p>If you have any questions, contact your administrator.</p>
        <div class="footer">
            <p>This email was sent from DockerVerse - Multi-Host Docker Management</p>
        </div>
    </div>
</body>
</html>
`, user.FirstName, user.Username, user.Role)

	return e.Send(user.Email, subject, body)
}

func (e *EmailService) SendPasswordReset(user *User, resetToken string) error {
	subject := "Password Reset - DockerVerse 🔐"
	// In real implementation, this would link to a reset page
	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; padding: 20px; }
        .container { max-width: 600px; margin: 0 auto; background: white; border-radius: 12px; padding: 40px; box-shadow: 0 4px 6px rgba(0,0,0,0.1); }
        h1 { color: #3b82f6; margin-bottom: 20px; }
        p { color: #333; line-height: 1.6; }
        .code { background: #1e293b; color: #10b981; padding: 15px 25px; border-radius: 8px; font-family: monospace; font-size: 24px; display: inline-block; margin: 20px 0; letter-spacing: 3px; }
        .warning { background: #fef3c7; color: #92400e; padding: 15px; border-radius: 8px; margin: 20px 0; }
        .footer { color: #666; font-size: 12px; margin-top: 30px; border-top: 1px solid #eee; padding-top: 20px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🔐 Password Reset Request</h1>
        <p>Hi <strong>%s</strong>,</p>
        <p>We received a request to reset your password. Use the code below to complete the reset:</p>
        <div style="text-align: center;">
            <div class="code">%s</div>
        </div>
        <div class="warning">
            <strong>⚠️ Important:</strong> This code expires in 15 minutes. If you didn't request this reset, please ignore this email.
        </div>
        <p>For security, never share this code with anyone.</p>
        <div class="footer">
            <p>This email was sent from DockerVerse - Multi-Host Docker Management</p>
        </div>
    </div>
</body>
</html>
`, user.FirstName, resetToken)

	return e.Send(user.Email, subject, body)
}

func (e *EmailService) SendPasswordChangeNotification(user *User, ipAddress, userAgent string) error {
	subject := "Password Changed - DockerVerse 🔐"
	timestamp := time.Now().Format("January 2, 2006 at 3:04 PM MST")

	// Try to get location from IP (simplified - in production use GeoIP service)
	location := "Unknown location"
	if ipAddress != "" {
		location = fmt.Sprintf("IP: %s", ipAddress)
	}

	// Parse user agent for device info
	device := "Unknown device"
	if userAgent != "" {
		if len(userAgent) > 100 {
			userAgent = userAgent[:100] + "..."
		}
		device = userAgent
	}

	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; padding: 20px; }
        .container { max-width: 600px; margin: 0 auto; background: white; border-radius: 12px; padding: 40px; box-shadow: 0 4px 6px rgba(0,0,0,0.1); }
        h1 { color: #3b82f6; margin-bottom: 20px; }
        p { color: #333; line-height: 1.6; }
        .info-box { background: #f0f9ff; padding: 15px 20px; border-radius: 8px; margin: 20px 0; border-left: 4px solid #3b82f6; }
        .info-row { display: flex; margin: 8px 0; }
        .info-label { color: #666; width: 100px; font-weight: 500; }
        .info-value { color: #333; }
        .warning { background: #fef3c7; color: #92400e; padding: 15px; border-radius: 8px; margin: 20px 0; }
        .footer { color: #666; font-size: 12px; margin-top: 30px; border-top: 1px solid #eee; padding-top: 20px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🔐 Password Changed Successfully</h1>
        <p>Hi <strong>%s</strong>,</p>
        <p>Your DockerVerse password was changed successfully. Here are the details:</p>
        
        <div class="info-box">
            <div class="info-row">
                <span class="info-label">📅 When:</span>
                <span class="info-value">%s</span>
            </div>
            <div class="info-row">
                <span class="info-label">📍 Location:</span>
                <span class="info-value">%s</span>
            </div>
            <div class="info-row">
                <span class="info-label">💻 Device:</span>
                <span class="info-value">%s</span>
            </div>
        </div>
        
        <div class="warning">
            <strong>⚠️ Wasn't you?</strong><br>
            If you didn't change your password, please contact your administrator immediately and consider:
            <ul style="margin: 10px 0 0 20px; padding: 0;">
                <li>Resetting your password</li>
                <li>Checking your account activity</li>
                <li>Enabling additional security measures</li>
            </ul>
        </div>
        
        <div class="footer">
            <p>This email was sent from DockerVerse - Multi-Host Docker Management</p>
            <p style="font-size: 11px; color: #999;">This is an automated security notification.</p>
        </div>
    </div>
</body>
</html>
`, user.FirstName, timestamp, location, device)

	return e.Send(user.Email, subject, body)
}

func (e *EmailService) SendNotification(to, title, body, notifyType string) error {
	icon := "ℹ️"
	bgColor := "#3b82f6"
	switch notifyType {
	case "success":
		icon = "✅"
		bgColor = "#22c55e"
	case "warning":
		icon = "⚠️"
		bgColor = "#f59e0b"
	case "failure", "error":
		icon = "🚨"
		bgColor = "#ef4444"
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")

	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; padding: 20px; margin: 0; }
        .container { max-width: 600px; margin: 0 auto; background: white; border-radius: 12px; overflow: hidden; box-shadow: 0 4px 6px rgba(0,0,0,0.1); }
        .header { background: %s; color: white; padding: 20px 30px; }
        .header h1 { margin: 0; font-size: 24px; }
        .content { padding: 30px; }
        .message { color: #333; line-height: 1.6; font-size: 16px; }
        .footer { background: #f8fafc; padding: 20px 30px; color: #64748b; font-size: 12px; border-top: 1px solid #e2e8f0; }
        .timestamp { color: #94a3b8; font-size: 13px; margin-top: 20px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>%s %s</h1>
        </div>
        <div class="content">
            <p class="message">%s</p>
            <p class="timestamp">🕐 %s</p>
        </div>
        <div class="footer">
            <p>📡 DockerVerse - Multi-Host Docker Management</p>
        </div>
    </div>
</body>
</html>
`, bgColor, icon, title, body, timestamp)

	return e.Send(to, fmt.Sprintf("%s %s", icon, title), htmlBody)
}

func (e *EmailService) SendEmailConfirmation(user *User, confirmToken, baseURL string) error {
	subject := "Confirm Your Email - DockerVerse ✉️"
	confirmLink := fmt.Sprintf("%s/confirm-email?token=%s", baseURL, confirmToken)

	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; padding: 20px; }
        .container { max-width: 600px; margin: 0 auto; background: white; border-radius: 12px; padding: 40px; box-shadow: 0 4px 6px rgba(0,0,0,0.1); }
        h1 { color: #3b82f6; margin-bottom: 20px; }
        p { color: #333; line-height: 1.6; }
        .button { display: inline-block; background: #3b82f6; color: white; padding: 14px 32px; border-radius: 8px; text-decoration: none; font-weight: bold; margin: 20px 0; }
        .link { color: #3b82f6; word-break: break-all; }
        .footer { color: #666; font-size: 12px; margin-top: 30px; border-top: 1px solid #eee; padding-top: 20px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>✉️ Confirm Your Email</h1>
        <p>Hi %s! 👋</p>
        <p>Welcome to DockerVerse! Please confirm your email address to activate your account.</p>
        
        <a href="%s" class="button">Confirm Email</a>
        
        <p style="font-size: 14px; color: #666;">Or copy this link to your browser:</p>
        <p class="link">%s</p>
        
        <p style="font-size: 14px; color: #888;">This link will expire in 24 hours.</p>
        
        <div class="footer">
            <p>This email was sent from DockerVerse - Multi-Host Docker Management</p>
        </div>
    </div>
</body>
</html>
`, user.FirstName, confirmLink, confirmLink)

	return e.Send(user.Email, subject, body)
}

// =============================================================================
// Docker Client Manager
// =============================================================================

type DockerManager struct {
	clients    map[string]*client.Client
	mu         sync.RWMutex
	lastStates map[string]string
	stateMu    sync.RWMutex
	notifySvc  *NotificationService
}

func NewDockerManager(notifySvc *NotificationService) *DockerManager {
	dm := &DockerManager{
		clients:    make(map[string]*client.Client),
		lastStates: make(map[string]string),
		notifySvc:  notifySvc,
	}
	dm.initClients()
	return dm
}

func (dm *DockerManager) initClients() {
	for _, host := range hosts {
		var cli *client.Client
		var err error

		if host.IsLocal {
			cli, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		} else {
			cli, err = client.NewClientWithOpts(
				client.WithHost(host.Address),
				client.WithAPIVersionNegotiation(),
			)
		}

		if err != nil {
			log.Printf("Warning: Could not connect to host %s: %v", host.Name, err)
			continue
		}

		dm.clients[host.ID] = cli
		log.Printf("Connected to Docker host: %s", host.Name)
	}
}

func (dm *DockerManager) GetClient(hostID string) (*client.Client, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	cli, ok := dm.clients[hostID]
	if !ok {
		return nil, fmt.Errorf("no client for host: %s", hostID)
	}
	return cli, nil
}

func (dm *DockerManager) GetAllContainers(ctx context.Context) ([]ContainerInfo, error) {
	var allContainers []ContainerInfo
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, host := range hosts {
		wg.Add(1)
		go func(h HostConfig) {
			defer wg.Done()

			// Skip hosts that recently failed (backoff)
			if !isHostHealthy(h.ID) {
				return
			}

			cli, err := dm.GetClient(h.ID)
			if err != nil {
				return
			}

			containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
			if err != nil {
				log.Printf("Error listing containers for %s: %v", h.Name, err)
				markHostFailed(h.ID)
				return
			}
			markHostHealthy(h.ID)

			for _, c := range containers {
				name := strings.TrimPrefix(c.Names[0], "/")

				// Get health status
				health := ""
				if c.State == "running" {
					inspect, err := cli.ContainerInspect(ctx, c.ID)
					if err == nil && inspect.State.Health != nil {
						health = inspect.State.Health.Status
					}
				}

				var ports []PortMapping
				for _, p := range c.Ports {
					ports = append(ports, PortMapping{
						Private: p.PrivatePort,
						Public:  p.PublicPort,
						Type:    p.Type,
					})
				}

				// Extract network IPs
				networks := make(map[string]string)
				if c.NetworkSettings != nil {
					for netName, net := range c.NetworkSettings.Networks {
						networks[netName] = net.IPAddress
					}
				}

				info := ContainerInfo{
					ID:       c.ID[:12],
					Name:     name,
					Image:    c.Image,
					Status:   c.Status,
					State:    c.State,
					Created:  c.Created,
					HostID:   h.ID,
					HostName: h.Name,
					Ports:    ports,
					Labels:   c.Labels,
					Health:   health,
					Networks: networks,
					Volumes:  len(c.Mounts),
					Stack:    c.Labels["com.docker.compose.project"],
					Service:  c.Labels["com.docker.compose.service"],
				}

				// Check state changes for notifications
				key := h.ID + ":" + c.ID[:12]
				dm.stateMu.Lock()
				lastState, exists := dm.lastStates[key]
				if exists && lastState != c.State {
					if c.State == "running" && lastState == "exited" {
						go dm.notifySvc.NotifyContainerState(name, "started")
					} else if c.State == "exited" && lastState == "running" {
						go dm.notifySvc.NotifyContainerState(name, "stopped")
					}
				}
				dm.lastStates[key] = c.State
				dm.stateMu.Unlock()

				mu.Lock()
				allContainers = append(allContainers, info)
				mu.Unlock()
			}
		}(host)
	}

	wg.Wait()

	// Sort alphabetically by name
	sort.Slice(allContainers, func(i, j int) bool {
		return strings.ToLower(allContainers[i].Name) < strings.ToLower(allContainers[j].Name)
	})

	return allContainers, nil
}

func (dm *DockerManager) GetContainerStats(ctx context.Context, hostID, containerID string) (*ContainerStats, error) {
	cli, err := dm.GetClient(hostID)
	if err != nil {
		return nil, err
	}

	statsResp, err := cli.ContainerStats(ctx, containerID, false)
	if err != nil {
		return nil, err
	}
	defer statsResp.Body.Close()

	var stats types.StatsJSON
	if err := json.NewDecoder(statsResp.Body).Decode(&stats); err != nil {
		return nil, err
	}

	// Calculate CPU percentage
	// Note: PercpuUsage may be empty on ARM/aarch64 with cgroups v2
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)
	numCPUs := float64(stats.CPUStats.OnlineCPUs)
	if numCPUs == 0 {
		numCPUs = float64(len(stats.CPUStats.CPUUsage.PercpuUsage))
	}
	if numCPUs == 0 {
		numCPUs = 1.0
	}
	cpuPercent := 0.0
	if systemDelta > 0 && cpuDelta > 0 {
		cpuPercent = (cpuDelta / systemDelta) * numCPUs * 100.0
	}

	// Calculate memory percentage
	memPercent := 0.0
	if stats.MemoryStats.Limit > 0 {
		memPercent = float64(stats.MemoryStats.Usage) / float64(stats.MemoryStats.Limit) * 100.0
	}

	// Calculate network I/O
	var netRx, netTx uint64
	for _, net := range stats.Networks {
		netRx += net.RxBytes
		netTx += net.TxBytes
	}

	// Calculate block I/O
	var blockRead, blockWrite uint64
	for _, bio := range stats.BlkioStats.IoServiceBytesRecursive {
		if bio.Op == "read" {
			blockRead += bio.Value
		} else if bio.Op == "write" {
			blockWrite += bio.Value
		}
	}

	return &ContainerStats{
		ID:          containerID,
		Name:        stats.Name,
		HostID:      hostID,
		CPUPercent:  cpuPercent,
		MemoryUsage: stats.MemoryStats.Usage,
		MemoryLimit: stats.MemoryStats.Limit,
		MemoryPct:   memPercent,
		NetRx:       netRx,
		NetTx:       netTx,
		BlockRead:   blockRead,
		BlockWrite:  blockWrite,
	}, nil
}

func (dm *DockerManager) GetStatsForContainers(ctx context.Context, containers []ContainerInfo) []ContainerStats {
	var allStats []ContainerStats
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, c := range containers {
		if c.State != "running" {
			continue
		}

		wg.Add(1)
		go func(cont ContainerInfo) {
			defer wg.Done()

			sCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			stats, err := dm.GetContainerStats(sCtx, cont.HostID, cont.ID)
			if err != nil {
				return
			}

			stats.Name = cont.Name

			go dm.notifySvc.NotifyHighResource(cont.Name, stats.CPUPercent, stats.MemoryPct)

			mu.Lock()
			allStats = append(allStats, *stats)
			mu.Unlock()
		}(c)
	}

	wg.Wait()
	return allStats
}

// Disk info cache per host (30s TTL)
var (
	diskCache   = make(map[string][]DiskInfo)
	diskCacheMu sync.RWMutex
	diskCacheAt = make(map[string]time.Time)
)

func getDiskInfo(ctx context.Context, hostID string) []DiskInfo {
	// Check cache
	diskCacheMu.RLock()
	if cached, ok := diskCache[hostID]; ok {
		if time.Since(diskCacheAt[hostID]) < 30*time.Second {
			diskCacheMu.RUnlock()
			return cached
		}
	}
	diskCacheMu.RUnlock()

	output, err := runSSHCommand(hostID, "df -B1 / /mnt /media /run/media 2>/dev/null")
	// Some systems return a non-zero exit code for `df` when some mountpoints
	// do not exist or other non-fatal conditions occur. The SSH session will
	// return an error in that case but may still produce valid output. Treat
	// the output as usable if it's non-empty; only fail when there is no
	// output to parse.
	if err != nil {
		if strings.TrimSpace(output) == "" {
			log.Printf("getDiskInfo(%s): ssh df failed and produced no output: %v", hostID, err)
			return nil
		}
		log.Printf("getDiskInfo(%s): ssh df returned error (non-fatal), will parse output: %v", hostID, err)
	}

	// Parse df output
	var disks []DiskInfo
	seen := make(map[string]bool)
	scanner := bufio.NewScanner(strings.NewReader(output))
	first := true
	for scanner.Scan() {
		line := scanner.Text()
		if first {
			first = false
			continue // skip header
		}
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		device := fields[0]
		// Filter to real filesystems only
		if strings.HasPrefix(device, "tmpfs") || strings.HasPrefix(device, "devtmpfs") ||
			strings.HasPrefix(device, "overlay") || strings.HasPrefix(device, "shm") ||
			strings.HasPrefix(device, "proc") || strings.HasPrefix(device, "sysfs") ||
			strings.HasPrefix(device, "none") || strings.HasPrefix(device, "cgroup") {
			continue
		}
		total, _ := strconv.ParseUint(fields[1], 10, 64)
		used, _ := strconv.ParseUint(fields[2], 10, 64)
		free, _ := strconv.ParseUint(fields[3], 10, 64)
		mountPoint := fields[5]
		if total == 0 {
			continue
		}
		key := device + "|" + mountPoint
		if seen[key] {
			continue
		}
		seen[key] = true
		disks = append(disks, DiskInfo{
			MountPoint: mountPoint,
			Device:     device,
			TotalBytes: total,
			UsedBytes:  used,
			FreeBytes:  free,
		})
	}

	// Cache result
	diskCacheMu.Lock()
	diskCache[hostID] = disks
	diskCacheAt[hostID] = time.Now()
	diskCacheMu.Unlock()

	return disks
}

func (dm *DockerManager) GetAllStats(ctx context.Context) ([]ContainerStats, error) {
	containers, err := dm.GetAllContainers(ctx)
	if err != nil {
		return nil, err
	}
	return dm.GetStatsForContainers(ctx, containers), nil
}

func (dm *DockerManager) GetHostStats(ctx context.Context) []HostStats {
	hostStats := make([]HostStats, len(hosts))
	var wg sync.WaitGroup

	for i, host := range hosts {
		wg.Add(1)
		go func(idx int, h HostConfig) {
			defer wg.Done()

			hs := HostStats{
				ID:      h.ID,
				Name:    h.Name,
				Online:  false,
				SSHHost: deriveSSHHost(h),
			}

			// Skip hosts in backoff period
			if !isHostHealthy(h.ID) {
				hostStats[idx] = hs
				return
			}

			cli, err := dm.GetClient(h.ID)
			if err != nil {
				hostStats[idx] = hs
				return
			}

			// Check if host is online with short timeout
			pingCtx, pingCancel := context.WithTimeout(ctx, 3*time.Second)
			_, err = cli.Ping(pingCtx)
			pingCancel()
			if err != nil {
				log.Printf("Host %s offline: %v", h.Name, err)
				markHostFailed(h.ID)
				hostStats[idx] = hs
				return
			}
			markHostHealthy(h.ID)

			hs.Online = true

			// Get host info for real memory total and CPU count
			info, infoErr := cli.Info(ctx)
			var memTotal uint64
			var hostNCPU float64
			if infoErr == nil {
				memTotal = uint64(info.MemTotal)
				hostNCPU = float64(info.NCPU)
			}

			// Get disk info (cached, 30s TTL)
			hs.Disks = getDiskInfo(ctx, h.ID)

			// Get containers
			containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
			if err == nil {
				hs.ContainerCount = len(containers)
				var totalCPU float64
				var totalMemUsage uint64
				var maxMemLimit uint64
				var runningCount int
				var statsWg sync.WaitGroup
				var statsMu sync.Mutex

				for _, c := range containers {
					if c.State == "running" {
						runningCount++
						statsWg.Add(1)
						go func(containerID string) {
							defer statsWg.Done()
							statsCtx, statsCancel := context.WithTimeout(ctx, 4*time.Second)
							defer statsCancel()
							statsResp, err := cli.ContainerStats(statsCtx, containerID, false)
							if err != nil {
								return
							}
							var stats types.StatsJSON
							if json.NewDecoder(statsResp.Body).Decode(&stats) == nil {
								cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
								systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)
								var cpu float64
								if systemDelta > 0 && cpuDelta > 0 {
									numCPUs := float64(stats.CPUStats.OnlineCPUs)
									if numCPUs == 0 {
										numCPUs = float64(len(stats.CPUStats.CPUUsage.PercpuUsage))
									}
									if numCPUs == 0 {
										numCPUs = hostNCPU
									}
									if numCPUs == 0 {
										numCPUs = 1
									}
									cpu = (cpuDelta / systemDelta) * numCPUs * 100.0
								}
								statsMu.Lock()
								totalCPU += cpu
								totalMemUsage += stats.MemoryStats.Usage
								if stats.MemoryStats.Limit > maxMemLimit {
									maxMemLimit = stats.MemoryStats.Limit
								}
								statsMu.Unlock()
							}
							statsResp.Body.Close()
						}(c.ID)
					}
				}
				statsWg.Wait()
				hs.RunningCount = runningCount
				// Divide total CPU by host core count to get 0-100% range
				if hostNCPU > 0 {
					totalCPU = totalCPU / hostNCPU
				}
				hs.CPUPercent = totalCPU
				hs.MemoryUsed = totalMemUsage
				if memTotal == 0 && maxMemLimit > 0 {
					memTotal = maxMemLimit
				}
				hs.MemoryTotal = memTotal
				if memTotal > 0 {
					hs.MemoryPercent = float64(totalMemUsage) / float64(memTotal) * 100.0
				}
			}

			hostStats[idx] = hs
		}(i, host)
	}

	wg.Wait()
	return hostStats
}

func (dm *DockerManager) ContainerAction(ctx context.Context, hostID, containerID, action string) error {
	cli, err := dm.GetClient(hostID)
	if err == nil {
		// Try Docker API first
		switch action {
		case "start":
			if err := cli.ContainerStart(ctx, containerID, container.StartOptions{}); err == nil {
				return nil
			} else {
				log.Printf("ContainerAction: docker API start failed for %s on %s: %v", containerID, hostID, err)
			}
		case "stop":
			timeout := 10
			if err := cli.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout}); err == nil {
				return nil
			} else {
				log.Printf("ContainerAction: docker API stop failed for %s on %s: %v", containerID, hostID, err)
			}
		case "restart":
			timeout := 10
			if err := cli.ContainerRestart(ctx, containerID, container.StopOptions{Timeout: &timeout}); err == nil {
				return nil
			} else {
				log.Printf("ContainerAction: docker API restart failed for %s on %s: %v", containerID, hostID, err)
			}
		case "pause":
			if err := cli.ContainerPause(ctx, containerID); err == nil {
				return nil
			} else {
				log.Printf("ContainerAction: docker API pause failed for %s on %s: %v", containerID, hostID, err)
			}
		case "unpause":
			if err := cli.ContainerUnpause(ctx, containerID); err == nil {
				return nil
			} else {
				log.Printf("ContainerAction: docker API unpause failed for %s on %s: %v", containerID, hostID, err)
			}
		default:
			return fmt.Errorf("unknown action: %s", action)
		}
	} else {
		log.Printf("ContainerAction: docker client unavailable for host %s: %v", hostID, err)
	}

	// Fallback: try to perform action over SSH by running the docker CLI remotely.
	// This helps when the Docker remote API is not exposed but SSH access is available.
	var sshCmd string
	switch action {
	case "start":
		sshCmd = fmt.Sprintf("docker start %s", containerID)
	case "stop":
		sshCmd = fmt.Sprintf("docker stop %s", containerID)
	case "restart":
		sshCmd = fmt.Sprintf("docker restart %s", containerID)
	case "pause":
		sshCmd = fmt.Sprintf("docker pause %s", containerID)
	case "unpause":
		sshCmd = fmt.Sprintf("docker unpause %s", containerID)
	default:
		return fmt.Errorf("unknown action: %s", action)
	}

	out, err := runSSHCommand(hostID, sshCmd)
	if err != nil {
		log.Printf("ContainerAction: ssh fallback failed for host=%s cmd=%q out=%q err=%v", hostID, sshCmd, out, err)
		return fmt.Errorf("action failed: %v (ssh fallback output: %s)", err, out)
	}
	log.Printf("ContainerAction: ssh fallback succeeded for host=%s cmd=%q output=%s", hostID, sshCmd, out)
	return nil
}

func (dm *DockerManager) GetContainerLogs(ctx context.Context, hostID, containerID string, tail int, follow bool) (io.ReadCloser, error) {
	cli, err := dm.GetClient(hostID)
	if err != nil {
		return nil, err
	}

	tailStr := fmt.Sprintf("%d", tail)
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     follow,
		Tail:       tailStr,
		Timestamps: true,
	}

	return cli.ContainerLogs(ctx, containerID, options)
}

// updateContainerImage updates a container using health check validation approach.
// This implements a safe update strategy with minimal downtime:
// 1. Pull new image
// 2. Create temporary container (no ports) to validate image
// 3. Health check validation (10-30s timeout)
// 4. If healthy: remove temp → stop old → create new → start
// 5. If unhealthy: remove temp → keep old running
func (dm *DockerManager) updateContainerImage(ctx context.Context, hostID, containerID string) error {
	cli, err := dm.GetClient(hostID)
	if err != nil {
		return fmt.Errorf("get docker client: %w", err)
	}

	// Step 1: Inspect current container to get configuration
	log.Printf("[Update] Step 1: Inspecting container %s on host %s", containerID, hostID)
	inspect, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return fmt.Errorf("inspect container: %w", err)
	}

	imageName := inspect.Config.Image
	oldContainerName := strings.TrimPrefix(inspect.Name, "/")
	log.Printf("[Update] Container: %s, Image: %s", oldContainerName, imageName)

	// Step 2: Pull latest image
	log.Printf("[Update] Step 2: Pulling image %s", imageName)
	pullCtx, pullCancel := context.WithTimeout(ctx, 5*time.Minute)
	defer pullCancel()

	pullResp, err := cli.ImagePull(pullCtx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("pull image: %w", err)
	}
	// Must read the pull response to ensure completion
	_, _ = io.Copy(io.Discard, pullResp)
	pullResp.Close()
	log.Printf("[Update] Image pulled successfully")

	// Step 3: Create temporary validation container (no ports, no name conflicts)
	log.Printf("[Update] Step 3: Creating temporary validation container")
	tempName := fmt.Sprintf("%s-validate-%d", oldContainerName, time.Now().Unix())

	// Create config for temp container (similar to original but no ports)
	tempConfig := &container.Config{
		Image:        imageName,
		Env:          inspect.Config.Env,
		Cmd:          inspect.Config.Cmd,
		Entrypoint:   inspect.Config.Entrypoint,
		WorkingDir:   inspect.Config.WorkingDir,
		User:         inspect.Config.User,
		Labels:       inspect.Config.Labels,
		Healthcheck:  inspect.Config.Healthcheck,
	}

	// Host config without port mappings
	tempHostConfig := &container.HostConfig{
		Binds:       inspect.HostConfig.Binds,
		RestartPolicy: container.RestartPolicy{Name: "no"}, // No restart for temp
		// Explicitly NO port bindings for temp container
	}

	tempCreateResp, err := cli.ContainerCreate(ctx, tempConfig, tempHostConfig, nil, nil, tempName)
	if err != nil {
		return fmt.Errorf("create temp container: %w", err)
	}
	tempID := tempCreateResp.ID
	log.Printf("[Update] Temp container created: %s", tempID[:12])

	// Cleanup function for temp container
	cleanupTemp := func() {
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanCancel()
		cli.ContainerRemove(cleanCtx, tempID, container.RemoveOptions{Force: true})
		log.Printf("[Update] Temp container removed: %s", tempID[:12])
	}

	// Step 4: Start temp container
	log.Printf("[Update] Step 4: Starting temp container for validation")
	if err := cli.ContainerStart(ctx, tempID, container.StartOptions{}); err != nil {
		cleanupTemp()
		return fmt.Errorf("start temp container: %w", err)
	}

	// Step 5: Health check validation (wait for healthy status or timeout)
	log.Printf("[Update] Step 5: Validating image health (30s timeout)")
	healthy := false
	healthTimeout := time.After(30 * time.Second)
	healthTicker := time.NewTicker(1 * time.Second)
	defer healthTicker.Stop()

healthCheck:
	for {
		select {
		case <-healthTimeout:
			log.Printf("[Update] Health check timeout - image validation failed")
			break healthCheck
		case <-healthTicker.C:
			tempInspect, err := cli.ContainerInspect(ctx, tempID)
			if err != nil {
				log.Printf("[Update] Failed to inspect temp container: %v", err)
				break healthCheck
			}

			// Check if container exited
			if !tempInspect.State.Running {
				if tempInspect.State.ExitCode == 0 {
					// Container ran and exited successfully (some containers do this)
					log.Printf("[Update] Container exited successfully (exit code 0)")
					healthy = true
					break healthCheck
				}
				log.Printf("[Update] Container exited with error (exit code %d)", tempInspect.State.ExitCode)
				break healthCheck
			}

			// If container has health check defined, wait for it
			if tempInspect.State.Health != nil {
				status := tempInspect.State.Health.Status
				log.Printf("[Update] Health status: %s", status)
				if status == "healthy" {
					healthy = true
					break healthCheck
				}
				if status == "unhealthy" {
					break healthCheck
				}
			} else {
				// No health check defined, if running for 5s assume healthy
				startedAt, err := time.Parse(time.RFC3339Nano, tempInspect.State.StartedAt)
				if err == nil && time.Since(startedAt) > 5*time.Second {
					log.Printf("[Update] No healthcheck defined, container running for 5s - assuming healthy")
					healthy = true
					break healthCheck
				}
			}
		}
	}

	// Step 6: Decision based on health check
	if !healthy {
		cleanupTemp()
		return fmt.Errorf("image validation failed: container unhealthy or crashed")
	}

	log.Printf("[Update] ✅ Image validated successfully")
	cleanupTemp()

	// Step 7: Stop old container
	log.Printf("[Update] Step 7: Stopping old container %s", containerID[:12])
	stopTimeout := 10
	if err := cli.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &stopTimeout}); err != nil {
		return fmt.Errorf("stop old container: %w", err)
	}

	// Step 8: Remove old container
	log.Printf("[Update] Step 8: Removing old container")
	if err := cli.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true}); err != nil {
		return fmt.Errorf("remove old container: %w", err)
	}

	// Step 9: Create new container with original configuration
	log.Printf("[Update] Step 9: Creating new container with updated image")
	newCreateResp, err := cli.ContainerCreate(
		ctx,
		inspect.Config,              // Original config
		inspect.HostConfig,          // Original host config (with ports!)
		nil,                         // Network config (will use default from HostConfig)
		nil,                         // Platform
		oldContainerName,            // Same name as before
	)
	if err != nil {
		return fmt.Errorf("create new container: %w", err)
	}
	newID := newCreateResp.ID
	log.Printf("[Update] New container created: %s", newID[:12])

	// Step 10: Start new container
	log.Printf("[Update] Step 10: Starting new container")
	if err := cli.ContainerStart(ctx, newID, container.StartOptions{}); err != nil {
		return fmt.Errorf("start new container: %w", err)
	}

	log.Printf("[Update] ✅ Container updated successfully: %s → %s", containerID[:12], newID[:12])
	return nil
}

// =============================================================================
// WebSocket Hub for Real-time Updates
// =============================================================================

type WSHub struct {
	clients    map[*websocket.Conn]bool
	sseClients map[chan []byte]bool
	broadcast  chan []byte
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mu         sync.Mutex
	sseMu      sync.Mutex
}

func NewWSHub() *WSHub {
	return &WSHub{
		clients:    make(map[*websocket.Conn]bool),
		sseClients: make(map[chan []byte]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

func (h *WSHub) RegisterSSE(ch chan []byte) {
	h.sseMu.Lock()
	h.sseClients[ch] = true
	h.sseMu.Unlock()
}

func (h *WSHub) UnregisterSSE(ch chan []byte) {
	h.sseMu.Lock()
	delete(h.sseClients, ch)
	h.sseMu.Unlock()
}

func (h *WSHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
			}
			h.mu.Unlock()
		case message := <-h.broadcast:
			h.mu.Lock()
			for client := range h.clients {
				if err := client.WriteMessage(websocket.TextMessage, message); err != nil {
					client.Close()
					delete(h.clients, client)
				}
			}
			h.mu.Unlock()
			// Fan out to SSE clients
			h.sseMu.Lock()
			for ch := range h.sseClients {
				select {
				case ch <- message:
				default:
					// Drop message if SSE client is slow
				}
			}
			h.sseMu.Unlock()
		}
	}
}

func (h *WSHub) Broadcast(msgType string, data interface{}) {
	msg := SSEMessage{Type: msgType, Data: data}
	jsonData, _ := json.Marshal(msg)
	h.broadcast <- jsonData
}

// =============================================================================
// HTTP Handlers
// =============================================================================

func setupRoutes(app *fiber.App, dm *DockerManager, store *UserStore, notifySvc *NotificationService, hub *WSHub, emailSvc *EmailService, envStore *EnvironmentStore) {
	// Health check endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	api := app.Group("/api")

	// Debug endpoint to inspect parsed hosts (temporary)
	api.Get("/debug/hosts", func(c *fiber.Ctx) error {
		return c.JSON(hosts)
	})

	// =========================
	// Auth routes (no auth required)
	// =========================

	api.Post("/auth/login", func(c *fiber.Ctx) error {
		var req LoginRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
		}

		user, err := store.ValidateLogin(req.Username, req.Password)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{"error": "invalid credentials"})
		}

		// Check if user has TOTP enabled
		if user.TOTPEnabled && user.TOTPSecret != "" {
			// If no TOTP code provided, return that 2FA is required
			if req.TOTPCode == "" && req.RecoveryCode == "" {
				// Generate a temporary token for the 2FA step
				tempToken, err := generateTempToken(user.Username)
				if err != nil {
					return c.Status(500).JSON(fiber.Map{"error": "failed to generate temp token"})
				}
				return c.JSON(LoginResponse{
					RequiresTOTP: true,
					TempToken:    tempToken,
				})
			}

			// Verify TOTP code or recovery code
			if req.TOTPCode != "" {
				// Use wider time window (±2 periods = ±60s) for RPi clock drift
				valid, validErr := totp.ValidateCustom(req.TOTPCode, user.TOTPSecret, time.Now(), totp.ValidateOpts{
					Period:    30,
					Skew:      2,
					Digits:    otp.DigitsSix,
					Algorithm: otp.AlgorithmSHA1,
				})
				log.Printf("2FA attempt for %s: valid=%v err=%v", user.Username, valid, validErr)
				if !valid || validErr != nil {
					return c.Status(401).JSON(fiber.Map{"error": "invalid 2FA code"})
				}
			} else if req.RecoveryCode != "" {
				// Check recovery code
				valid, err := store.UseRecoveryCode(user.Username, req.RecoveryCode)
				if err != nil || !valid {
					return c.Status(401).JSON(fiber.Map{"error": "invalid recovery code"})
				}
			}
		}

		tokens, err := generateTokens(user, req.RememberMe)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "failed to generate tokens"})
		}

		return c.JSON(LoginResponse{
			User:   user.SafeUser(),
			Tokens: *tokens,
		})
	})

	// 2FA Recovery - disable 2FA using username+password (localhost only)
	api.Post("/auth/disable-2fa", func(c *fiber.Ctx) error {
		// Only allow from localhost for security
		ip := c.IP()
		if ip != "127.0.0.1" && ip != "::1" && ip != "localhost" {
			return c.Status(403).JSON(fiber.Map{"error": "only accessible from localhost"})
		}

		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
		}

		user, err := store.ValidateLogin(req.Username, req.Password)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{"error": "invalid credentials"})
		}

		store.mu.Lock()
		if u, ok := store.Users[user.Username]; ok {
			u.TOTPEnabled = false
			u.TOTPSecret = ""
			u.RecoveryCodes = nil
		}
		store.mu.Unlock()
		store.save()

		log.Printf("2FA disabled for user %s via recovery endpoint", user.Username)
		return c.JSON(fiber.Map{"success": true, "message": "2FA disabled for " + user.Username})
	})

	api.Post("/auth/refresh", func(c *fiber.Ctx) error {
		var req struct {
			RefreshToken string `json:"refreshToken"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
		}

		token, err := jwt.Parse(req.RefreshToken, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})
		if err != nil || !token.Valid {
			return c.Status(401).JSON(fiber.Map{"error": "invalid refresh token"})
		}

		claims, _ := token.Claims.(jwt.MapClaims)
		username := claims["sub"].(string)

		user := store.GetUser(username)
		if user == nil {
			return c.Status(401).JSON(fiber.Map{"error": "user not found"})
		}

		tokens, err := generateTokens(user, true)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "failed to generate tokens"})
		}

		return c.JSON(fiber.Map{"tokens": tokens})
	})

	// Password Recovery (no auth required)
	api.Post("/auth/forgot-password", func(c *fiber.Ctx) error {
		var req struct {
			Email    string `json:"email"`
			Username string `json:"username"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
		}

		// Find user by email or username
		users := store.GetAllUsers()
		var foundUser *User
		for _, u := range users {
			if (req.Email != "" && u.Email == req.Email) || (req.Username != "" && u.Username == req.Username) {
				foundUser = u
				break
			}
		}

		// Always return success (don't reveal if email exists)
		if foundUser == nil || foundUser.Email == "" {
			return c.JSON(fiber.Map{"success": true, "message": "If the user exists, a reset code will be sent"})
		}

		// Generate masked email for display
		maskedEmail := maskEmail(foundUser.Email)

		// Generate 6-digit reset code
		b := make([]byte, 3)
		rand.Read(b)
		resetCode := fmt.Sprintf("%06d", int(b[0])%10*100000+int(b[1])%10*10000+int(b[2])%10*1000+int(b[0]^b[1])%10*100+int(b[1]^b[2])%10*10+int(b[0]^b[2])%10)

		// DEBUG LOG - Remove in production
		log.Printf("[DEBUG] Reset code for %s: %s", foundUser.Email, resetCode)

		// Store reset code (in production, use Redis with TTL)
		// For now, we'll use a simple in-memory store
		store.SetResetCode(foundUser.Email, resetCode, foundUser.ID)

		// Send email
		go func() {
			if err := emailSvc.SendPasswordReset(foundUser, resetCode); err != nil {
				log.Printf("Failed to send reset email to %s: %v", foundUser.Email, err)
			}
		}()

		return c.JSON(fiber.Map{
			"success":     true,
			"message":     "Reset code sent",
			"maskedEmail": maskedEmail,
			"email":       foundUser.Email, // Full email needed for reset-password endpoint
		})
	})

	// Verify reset code (without resetting password)
	api.Post("/auth/verify-code", func(c *fiber.Ctx) error {
		var req struct {
			Email string `json:"email"`
			Code  string `json:"code"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
		}

		// Verify code
		_, valid := store.VerifyResetCode(req.Email, req.Code)
		if !valid {
			return c.Status(400).JSON(fiber.Map{"error": "invalid or expired code", "valid": false})
		}

		return c.JSON(fiber.Map{"valid": true, "message": "Code verified successfully"})
	})

	api.Post("/auth/reset-password", func(c *fiber.Ctx) error {
		var req struct {
			Email       string `json:"email"`
			Code        string `json:"code"`
			NewPassword string `json:"newPassword"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
		}

		// Find user by email
		users := store.GetAllUsers()
		var foundUser *User
		for _, u := range users {
			if u.Email == req.Email {
				foundUser = u
				break
			}
		}

		if foundUser == nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid reset request"})
		}

		// Verify code
		_, valid := store.VerifyResetCode(req.Email, req.Code)
		if !valid {
			return c.Status(400).JSON(fiber.Map{"error": "invalid or expired code"})
		}

		// Update password
		_, err := store.UpdateUser(foundUser.Username, UpdateUserRequest{Password: req.NewPassword})
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "failed to update password"})
		}

		// Clear reset code
		store.ClearResetCode(req.Email)

		return c.JSON(fiber.Map{"success": true, "message": "Password updated successfully"})
	})

	// Email confirmation endpoint (no auth required)
	api.Get("/auth/confirm-email", func(c *fiber.Ctx) error {
		token := c.Query("token")
		if token == "" {
			return c.Status(400).JSON(fiber.Map{"error": "missing token"})
		}

		user, err := store.ConfirmEmail(token)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}

		// Send welcome email now that email is confirmed
		go func() {
			if err := emailSvc.SendWelcome(user); err != nil {
				log.Printf("Failed to send welcome email to %s: %v", user.Email, err)
			}
		}()

		return c.JSON(fiber.Map{
			"success":  true,
			"message":  "Email confirmed successfully",
			"username": user.Username,
		})
	})

	// =========================
	// Protected routes
	// =========================

	protected := api.Group("", jwtMiddleware(store))

	// =========================
	// TOTP/2FA routes (require auth)
	// =========================

	// Setup TOTP - Generate secret and QR code URL
	protected.Post("/auth/totp/setup", func(c *fiber.Ctx) error {
		username := c.Locals("user").(*User).Username

		secret, url, err := store.SetupTOTP(username)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{
			"secret": secret,
			"url":    url,
		})
	})

	// Enable TOTP - Verify code and activate 2FA
	protected.Post("/auth/totp/enable", func(c *fiber.Ctx) error {
		username := c.Locals("user").(*User).Username

		var req struct {
			Code string `json:"code"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
		}

		recoveryCodes, err := store.EnableTOTP(username, req.Code)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{
			"success":       true,
			"recoveryCodes": recoveryCodes,
		})
	})

	// Disable TOTP - Requires password confirmation
	protected.Post("/auth/totp/disable", func(c *fiber.Ctx) error {
		username := c.Locals("user").(*User).Username

		var req struct {
			Password string `json:"password"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
		}

		err := store.DisableTOTP(username, req.Password)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"success": true})
	})

	// Get TOTP status
	protected.Get("/auth/totp/status", func(c *fiber.Ctx) error {
		username := c.Locals("user").(*User).Username
		user := store.GetUser(username)
		if user == nil {
			return c.Status(404).JSON(fiber.Map{"error": "user not found"})
		}

		return c.JSON(fiber.Map{
			"enabled":       user.TOTPEnabled,
			"recoveryCount": store.GetRecoveryCodesCount(username),
		})
	})

	// Regenerate recovery codes
	protected.Post("/auth/totp/regenerate-recovery", func(c *fiber.Ctx) error {
		username := c.Locals("user").(*User).Username

		var req struct {
			Password string `json:"password"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
		}

		codes, err := store.RegenerateRecoveryCodes(username, req.Password)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"recoveryCodes": codes})
	})

	// Auth
	protected.Post("/auth/logout", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"success": true})
	})

	protected.Get("/auth/me", func(c *fiber.Ctx) error {
		user := c.Locals("user").(*User)
		return c.JSON(user.SafeUser())
	})

	protected.Patch("/auth/profile", func(c *fiber.Ctx) error {
		user := c.Locals("user").(*User)
		var req UpdateUserRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
		}

		// Users can only update their own profile (not role)
		req.Role = ""
		updated, err := store.UpdateUser(user.Username, req)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"user": updated.SafeUser()})
	})

	// Avatar upload endpoint
	protected.Post("/auth/avatar", func(c *fiber.Ctx) error {
		user := c.Locals("user").(*User)
		log.Printf("Avatar upload request from %s, body size: %d bytes", user.Username, len(c.Body()))

		var req struct {
			Avatar string `json:"avatar"` // Base64 encoded image data URI
		}
		if err := c.BodyParser(&req); err != nil {
			log.Printf("Avatar upload parse error for %s: %v", user.Username, err)
			return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
		}

		// Validate base64 image (should start with data:image/)
		if req.Avatar != "" && !strings.HasPrefix(req.Avatar, "data:image/") {
			return c.Status(400).JSON(fiber.Map{"error": "invalid image format"})
		}

		// Limit size (roughly 500KB base64 ~ 375KB image)
		if len(req.Avatar) > 500000 {
			return c.Status(400).JSON(fiber.Map{"error": "image too large (max 500KB)"})
		}

		// Update user's avatar
		store.mu.Lock()
		if u, ok := store.Users[user.Username]; ok {
			u.Avatar = req.Avatar
		}
		store.mu.Unlock()
		store.save()

		// Return updated user
		return c.JSON(fiber.Map{"success": true, "avatar": req.Avatar})
	})

	// Delete avatar endpoint
	protected.Delete("/auth/avatar", func(c *fiber.Ctx) error {
		user := c.Locals("user").(*User)

		store.mu.Lock()
		if u, ok := store.Users[user.Username]; ok {
			u.Avatar = ""
		}
		store.mu.Unlock()
		store.save()

		return c.JSON(fiber.Map{"success": true})
	})

	protected.Post("/auth/password", func(c *fiber.Ctx) error {
		user := c.Locals("user").(*User)
		var req struct {
			CurrentPassword string `json:"currentPassword"`
			NewPassword     string `json:"newPassword"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
		}

		// Verify current password
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
			return c.Status(401).JSON(fiber.Map{"error": "current password is incorrect"})
		}

		// Update password
		_, err := store.UpdateUser(user.Username, UpdateUserRequest{Password: req.NewPassword})
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		// Send password change notification email (async)
		if user.Email != "" {
			go func() {
				ipAddress := c.IP()
				userAgent := c.Get("User-Agent")
				if err := emailSvc.SendPasswordChangeNotification(user, ipAddress, userAgent); err != nil {
					log.Printf("Failed to send password change notification to %s: %v", user.Email, err)
				} else {
					log.Printf("Password change notification sent to %s", user.Email)
				}
			}()
		}

		return c.JSON(fiber.Map{"success": true})
	})

	// =========================
	// User Management (Admin only)
	// =========================

	admin := protected.Group("/users", adminOnly())

	admin.Get("/", func(c *fiber.Ctx) error {
		users := store.GetAllUsers()
		safeUsers := make([]*User, len(users))
		for i, u := range users {
			safeUsers[i] = u.SafeUser()
		}
		return c.JSON(safeUsers)
	})

	admin.Post("/", func(c *fiber.Ctx) error {
		var req CreateUserRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
		}

		if req.Role == "" {
			req.Role = "user"
		}

		user, err := store.CreateUser(req)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}

		// Send email confirmation (async, don't block on errors)
		if user.Email != "" {
			go func() {
				// Get base URL from request
				baseURL := fmt.Sprintf("%s://%s", c.Protocol(), c.Hostname())
				if err := emailSvc.SendEmailConfirmation(user, user.ConfirmToken, baseURL); err != nil {
					log.Printf("Failed to send confirmation email to %s: %v", user.Email, err)
				} else {
					log.Printf("Confirmation email sent to %s", user.Email)
				}
			}()
		}

		return c.Status(201).JSON(user.SafeUser())
	})

	admin.Patch("/:username", func(c *fiber.Ctx) error {
		username := c.Params("username")
		var req UpdateUserRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
		}

		user, err := store.UpdateUser(username, req)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(user.SafeUser())
	})

	admin.Delete("/:username", func(c *fiber.Ctx) error {
		username := c.Params("username")
		if err := store.DeleteUser(username); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"success": true})
	})

	// =========================
	// Settings
	// =========================

	protected.Get("/settings", func(c *fiber.Ctx) error {
		return c.JSON(store.GetSettings())
	})

	protected.Put("/settings", adminOnly(), func(c *fiber.Ctx) error {
		var settings AppSettings
		if err := c.BodyParser(&settings); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
		}
		store.UpdateSettings(settings)
		return c.JSON(settings)
	})

	// =========================
	// Notifications
	// =========================

	protected.Post("/notify/test", adminOnly(), func(c *fiber.Ctx) error {
		var req NotifyRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
		}

		if req.Title == "" {
			req.Title = "🧪 DockerVerse Test"
		}
		if req.Body == "" {
			req.Body = "This is a test notification from DockerVerse - If you see this, notifications are working!"
		}
		if req.Type == "" {
			req.Type = "info"
		}

		channel := req.Channel
		if channel == "" {
			channel = "both"
		}

		results := fiber.Map{"channel": channel}
		var errors []string

		// Send Telegram notification
		if channel == "telegram" || channel == "both" || channel == "all" {
			if err := notifySvc.SendWithChannel(req.Title, req.Body, req.Type, "telegram"); err != nil {
				errors = append(errors, fmt.Sprintf("telegram: %s", err.Error()))
				results["telegram"] = "failed"
			} else {
				results["telegram"] = "sent"
			}
		}

		// Send Email notification to the current admin user
		if channel == "email" || channel == "both" || channel == "all" {
			// Get user from context (set by jwtMiddleware)
			user := c.Locals("user").(*User)
			if user != nil && user.Email != "" {
				if err := emailSvc.SendNotification(user.Email, req.Title, req.Body, req.Type); err != nil {
					errors = append(errors, fmt.Sprintf("email: %s", err.Error()))
					results["email"] = "failed"
				} else {
					results["email"] = "sent"
				}
			} else {
				errors = append(errors, "email: no email configured for user")
				results["email"] = "no_email"
			}
		}

		results["success"] = len(errors) == 0
		if len(errors) > 0 {
			results["errors"] = errors
		}

		return c.JSON(results)
	})

	// Configure Apprise URLs
	protected.Post("/notify/configure", adminOnly(), func(c *fiber.Ctx) error {
		if err := notifySvc.ConfigureApprise(); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"success": true})
	})

	// =========================
	// Docker routes (protected)
	// =========================

	// Hosts - with timeout to prevent slow hosts from blocking
	protected.Get("/hosts", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		stats := dm.GetHostStats(ctx)
		return c.JSON(stats)
	})

	protected.Get("/hosts/:hostId/files", func(c *fiber.Ctx) error {
		hostID := c.Params("hostId")
		dir := c.Query("path", "/")
		if !strings.HasPrefix(dir, "/") {
			return c.Status(400).JSON(fiber.Map{"error": "path must be absolute"})
		}
		sshClient, err := dialSSH(hostID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		defer sshClient.Close()
		sftpClient, err := sftp.NewClient(sshClient)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		defer sftpClient.Close()
		entries, err := sftpClient.ReadDir(dir)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		results := make([]HostFileEntry, 0, len(entries))
		for _, entry := range entries {
			name := entry.Name()
			results = append(results, HostFileEntry{
				Name:    name,
				Path:    path.Join(dir, name),
				Size:    entry.Size(),
				ModTime: entry.ModTime().Unix(),
				IsDir:   entry.IsDir(),
			})
		}
		return c.JSON(results)
	})

	protected.Get("/hosts/:hostId/files/download", func(c *fiber.Ctx) error {
		hostID := c.Params("hostId")
		filePath := c.Query("path", "")
		if filePath == "" || !strings.HasPrefix(filePath, "/") {
			return c.Status(400).JSON(fiber.Map{"error": "path must be absolute"})
		}
		sshClient, err := dialSSH(hostID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		defer sshClient.Close()
		sftpClient, err := sftp.NewClient(sshClient)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		defer sftpClient.Close()
		file, err := sftpClient.Open(filePath)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		defer file.Close()
		c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", path.Base(filePath)))
		return c.SendStream(file)
	})

	protected.Post("/hosts/:hostId/files/upload", func(c *fiber.Ctx) error {
		hostID := c.Params("hostId")
		dir := c.FormValue("path", "/")
		if !strings.HasPrefix(dir, "/") {
			return c.Status(400).JSON(fiber.Map{"error": "path must be absolute"})
		}
		fileHeader, err := c.FormFile("file")
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "missing file"})
		}
		src, err := fileHeader.Open()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		defer src.Close()
		sshClient, err := dialSSH(hostID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		defer sshClient.Close()
		sftpClient, err := sftp.NewClient(sshClient)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		defer sftpClient.Close()
		target := path.Join(dir, fileHeader.Filename)
		dst, err := sftpClient.Create(target)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		defer dst.Close()
		if _, err := io.Copy(dst, src); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"success": true, "path": target})
	})

	protected.Post("/hosts/:hostId/files/mkdir", func(c *fiber.Ctx) error {
		hostID := c.Params("hostId")
		var req struct {
			Path string `json:"path"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
		}
		if req.Path == "" || !strings.HasPrefix(req.Path, "/") {
			return c.Status(400).JSON(fiber.Map{"error": "path must be absolute"})
		}
		sshClient, err := dialSSH(hostID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		defer sshClient.Close()
		sftpClient, err := sftp.NewClient(sshClient)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		defer sftpClient.Close()
		if err := sftpClient.MkdirAll(req.Path); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"success": true})
	})

	protected.Delete("/hosts/:hostId/files", func(c *fiber.Ctx) error {
		hostID := c.Params("hostId")
		filePath := c.Query("path", "")
		if filePath == "" || !strings.HasPrefix(filePath, "/") {
			return c.Status(400).JSON(fiber.Map{"error": "path must be absolute"})
		}
		sshClient, err := dialSSH(hostID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		defer sshClient.Close()
		sftpClient, err := sftp.NewClient(sshClient)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		defer sftpClient.Close()
		stat, err := sftpClient.Stat(filePath)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		if stat.IsDir() {
			if err := sftpClient.RemoveDirectory(filePath); err != nil {
				return c.Status(500).JSON(fiber.Map{"error": err.Error()})
			}
		} else {
			if err := sftpClient.Remove(filePath); err != nil {
				return c.Status(500).JSON(fiber.Map{"error": err.Error()})
			}
		}
		return c.JSON(fiber.Map{"success": true})
	})

	// Containers - with timeout
	protected.Get("/containers", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		containers, err := dm.GetAllContainers(ctx)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(containers)
	})

	// GET /api/stacks?hostId=raspi1
	// Returns Docker Compose stacks grouped by com.docker.compose.project label
	protected.Get("/stacks", func(c *fiber.Ctx) error {
		hostID := c.Query("hostId")
		if hostID == "" {
			return c.Status(400).JSON(fiber.Map{"error": "hostId required"})
		}

		cli, err := dm.GetClient(hostID)
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "Host not found"})
		}

		// List all containers
		ctx := context.Background()
		containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
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
	})

	// Stats
	protected.Get("/stats", func(c *fiber.Ctx) error {
		ctx := context.Background()
		stats, err := dm.GetAllStats(ctx)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(stats)
	})

	// Image updates check
	protected.Get("/updates", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		containers, err := dm.GetAllContainers(ctx)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		updates := checkImageUpdates(ctx, containers, dm)
		return c.JSON(updates)
	})

	// Force update check for a specific container
	protected.Post("/updates/:hostId/:containerId/check", func(c *fiber.Ctx) error {
		hostID := c.Params("hostId")
		containerID := c.Params("containerId")

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		update := checkSingleImageUpdate(ctx, hostID, containerID, dm)
		return c.JSON(update)
	})

	// Container actions
	// Trigger Watchtower update for a specific container
	// IMPORTANT: This must come BEFORE the generic /:action route to avoid conflicts
	protected.Post("/containers/:hostId/:containerId/update", func(c *fiber.Ctx) error {
		hostID := c.Params("hostId")
		containerID := c.Params("containerId")

		// Create context with 10-minute timeout for the entire update process
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		// Execute the update using the new health-check based update logic
		if err := dm.updateContainerImage(ctx, hostID, containerID); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		// Clear update cache for this container
		updateCacheMu.Lock()
		delete(updateCache, containerID)
		updateCacheMu.Unlock()

		return c.JSON(fiber.Map{
			"success": true,
			"message": "Container updated successfully with health validation",
		})
	})

	// Generic container actions (start, stop, restart, pause, unpause)
	// Note: 'update' is NOT handled here - use the specific /update endpoint above
	protected.Post("/containers/:hostId/:containerId/:action", func(c *fiber.Ctx) error {
		hostID := c.Params("hostId")
		containerID := c.Params("containerId")
		action := c.Params("action")

		ctx := context.Background()
		if err := dm.ContainerAction(ctx, hostID, containerID, action); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		// Broadcast the state change immediately
		go func() {
			time.Sleep(500 * time.Millisecond)
			containers, _ := dm.GetAllContainers(context.Background())
			hub.Broadcast("containers", containers)
		}()

		return c.JSON(fiber.Map{"success": true})
	})

	// Logs endpoint
	protected.Get("/logs/:hostId/:containerId", func(c *fiber.Ctx) error {
		hostID := c.Params("hostId")
		containerID := c.Params("containerId")
		tail := c.QueryInt("tail", 100)

		ctx := context.Background()
		logs, err := dm.GetContainerLogs(ctx, hostID, containerID, tail, false)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		defer logs.Close()

		// Read all logs
		data, err := io.ReadAll(logs)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		// Parse docker log format (8-byte header per line)
		lines := parseDockerLogs(data)

		return c.JSON(fiber.Map{"logs": lines})
	})

	// Logs SSE for streaming
	protected.Get("/logs/:hostId/:containerId/stream", func(c *fiber.Ctx) error {
		hostID := c.Params("hostId")
		containerID := c.Params("containerId")

		c.Set("Content-Type", "text/event-stream")
		c.Set("Cache-Control", "no-cache")
		c.Set("Connection", "keep-alive")

		ctx := c.Context()

		c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
			logCtx, cancel := context.WithCancel(context.Background())
			defer cancel()

			logs, err := dm.GetContainerLogs(logCtx, hostID, containerID, 50, true)
			if err != nil {
				return
			}
			defer logs.Close()

			reader := bufio.NewReader(logs)
			buf := make([]byte, 8192)

			for {
				select {
				case <-ctx.Done():
					return
				default:
					n, err := reader.Read(buf)
					if err != nil {
						return
					}
					if n > 0 {
						lines := parseDockerLogs(buf[:n])
						for _, line := range lines {
							msg := SSEMessage{Type: "log", Data: line}
							data, _ := json.Marshal(msg)
							fmt.Fprintf(w, "data: %s\n\n", data)
						}
						w.Flush()
					}
				}
			}
		})

		return nil
	})

	// Search
	protected.Get("/search", func(c *fiber.Ctx) error {
		query := strings.ToLower(c.Query("q", ""))
		if query == "" {
			return c.JSON([]ContainerInfo{})
		}

		ctx := context.Background()
		containers, err := dm.GetAllContainers(ctx)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		var results []ContainerInfo
		for _, cont := range containers {
			if strings.Contains(strings.ToLower(cont.Name), query) ||
				strings.Contains(strings.ToLower(cont.Image), query) ||
				strings.Contains(strings.ToLower(cont.HostName), query) {
				results = append(results, cont)
			}
		}

		return c.JSON(results)
	})

	// SSE endpoint - subscribes to hub broadcasts instead of polling Docker API
	protected.Get("/events", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/event-stream")
		c.Set("Cache-Control", "no-cache")
		c.Set("Connection", "keep-alive")
		c.Set("Transfer-Encoding", "chunked")

		ctx := c.Context()
		msgCh := make(chan []byte, 64)
		hub.RegisterSSE(msgCh)

		c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
			defer hub.UnregisterSSE(msgCh)

			for {
				select {
				case <-ctx.Done():
					return
				case msg := <-msgCh:
					fmt.Fprintf(w, "data: %s\n\n", msg)
					w.Flush()
				}
			}
		})

		return nil
	})

	// WebSocket for real-time updates
	app.Get("/ws/events", websocket.New(func(c *websocket.Conn) {
		hub.register <- c
		defer func() {
			hub.unregister <- c
			c.Close()
		}()

		// Keep connection alive
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				break
			}
		}
	}))

	// WebSocket for terminal
	// Helper function: SSH fallback for container terminal when Docker API exec is blocked
	handleContainerTerminalSSH := func(c *websocket.Conn, hostID, containerID string) {
		client, err := dialSSH(hostID)
		if err != nil {
			errMsg, _ := json.Marshal(map[string]string{"type": "error", "data": "SSH connection failed: " + err.Error()})
			c.WriteMessage(websocket.TextMessage, errMsg)
			return
		}
		defer client.Close()

		session, err := client.NewSession()
		if err != nil {
			errMsg, _ := json.Marshal(map[string]string{"type": "error", "data": "SSH session failed: " + err.Error()})
			c.WriteMessage(websocket.TextMessage, errMsg)
			return
		}
		defer session.Close()

		modes := ssh.TerminalModes{
			ssh.ECHO:          1,
			ssh.TTY_OP_ISPEED: 14400,
			ssh.TTY_OP_OSPEED: 14400,
		}
		if err := session.RequestPty("xterm-256color", 40, 120, modes); err != nil {
			errMsg, _ := json.Marshal(map[string]string{"type": "error", "data": "PTY request failed: " + err.Error()})
			c.WriteMessage(websocket.TextMessage, errMsg)
			return
		}

		stdout, err := session.StdoutPipe()
		if err != nil {
			errMsg, _ := json.Marshal(map[string]string{"type": "error", "data": err.Error()})
			c.WriteMessage(websocket.TextMessage, errMsg)
			return
		}
		stderr, err := session.StderrPipe()
		if err != nil {
			errMsg, _ := json.Marshal(map[string]string{"type": "error", "data": err.Error()})
			c.WriteMessage(websocket.TextMessage, errMsg)
			return
		}
		stdin, err := session.StdinPipe()
		if err != nil {
			errMsg, _ := json.Marshal(map[string]string{"type": "error", "data": err.Error()})
			c.WriteMessage(websocket.TextMessage, errMsg)
			return
		}

		// Execute docker exec via SSH
		// Try bash first, fallback to sh
		cmd := fmt.Sprintf("docker exec -it %s sh -c 'command -v bash >/dev/null && exec bash || exec sh'", containerID)
		if err := session.Start(cmd); err != nil {
			errMsg, _ := json.Marshal(map[string]string{"type": "error", "data": "Failed to start docker exec: " + err.Error()})
			c.WriteMessage(websocket.TextMessage, errMsg)
			return
		}

		// Send connection success message
		connMsg, _ := json.Marshal(map[string]string{"type": "info", "data": "Connected via SSH fallback"})
		c.WriteMessage(websocket.TextMessage, connMsg)

		output := func(reader io.Reader) {
			buf := make([]byte, 4096)
			for {
				n, err := reader.Read(buf)
				if err != nil {
					return
				}
				if n > 0 {
					msg, _ := json.Marshal(map[string]string{"type": "output", "data": string(buf[:n])})
					c.WriteMessage(websocket.TextMessage, msg)
				}
			}
		}
		go output(stdout)
		go output(stderr)

		for {
			_, msg, err := c.ReadMessage()
			if err != nil {
				return
			}
			var wsMsg struct {
				Type string `json:"type"`
				Data string `json:"data"`
				Cols int    `json:"cols"`
				Rows int    `json:"rows"`
			}
			if json.Unmarshal(msg, &wsMsg) == nil {
				if wsMsg.Type == "input" {
					stdin.Write([]byte(wsMsg.Data))
				} else if wsMsg.Type == "resize" {
					if wsMsg.Cols > 0 && wsMsg.Rows > 0 {
						session.WindowChange(wsMsg.Rows, wsMsg.Cols)
					}
				}
			}
		}
	}

	app.Get("/ws/terminal/:hostId/:containerId", websocket.New(func(c *websocket.Conn) {
		hostID := c.Params("hostId")
		containerID := c.Params("containerId")

		cli, err := dm.GetClient(hostID)
		if err != nil {
			// No Docker client available, try SSH fallback directly
			log.Printf("Terminal: no docker client for host=%s, attempting SSH fallback", hostID)
			handleContainerTerminalSSH(c, hostID, containerID)
			return
		}

		ctx := context.Background()

		// Create exec
		execConfig := container.ExecOptions{
			AttachStdin:  true,
			AttachStdout: true,
			AttachStderr: true,
			Tty:          true,
			Cmd:          []string{"/bin/sh", "-c", "command -v bash >/dev/null && exec bash || exec sh"},
		}

		execIDResp, err := cli.ContainerExecCreate(ctx, containerID, execConfig)
		if err != nil {
			// Check if it's a 403 or other Docker API error - try SSH fallback
			errStr := err.Error()
			if strings.Contains(errStr, "403") || strings.Contains(errStr, "Forbidden") {
				log.Printf("Terminal: docker exec blocked (403) for container=%s on host=%s, attempting SSH fallback", containerID, hostID)
				handleContainerTerminalSSH(c, hostID, containerID)
				return
			}
			// Other errors, report and exit
			errMsg, _ := json.Marshal(map[string]string{"type": "error", "data": "Error creating exec: " + err.Error()})
			c.WriteMessage(websocket.TextMessage, errMsg)
			return
		}

		// Attach to exec
		attachResp, err := cli.ContainerExecAttach(ctx, execIDResp.ID, container.ExecStartOptions{Tty: true})
		if err != nil {
			errMsg, _ := json.Marshal(map[string]string{"type": "error", "data": "Error attaching: " + err.Error()})
			c.WriteMessage(websocket.TextMessage, errMsg)
			return
		}
		defer attachResp.Close()

		// Read from container -> WebSocket (send as JSON)
		go func() {
			buf := make([]byte, 4096)
			for {
				n, err := attachResp.Reader.Read(buf)
				if err != nil {
					return
				}
				if n > 0 {
					msg, _ := json.Marshal(map[string]string{"type": "output", "data": string(buf[:n])})
					c.WriteMessage(websocket.TextMessage, msg)
				}
			}
		}()

		// Read from WebSocket -> container (parse JSON input)
		for {
			_, msg, err := c.ReadMessage()
			if err != nil {
				return
			}
			var wsMsg struct {
				Type string `json:"type"`
				Data string `json:"data"`
				Cols int    `json:"cols"`
				Rows int    `json:"rows"`
			}
			if json.Unmarshal(msg, &wsMsg) == nil {
				if wsMsg.Type == "input" {
					attachResp.Conn.Write([]byte(wsMsg.Data))
				} else if wsMsg.Type == "resize" {
					// Resize terminal
					cli.ContainerExecResize(ctx, execIDResp.ID, container.ResizeOptions{
						Height: uint(wsMsg.Rows),
						Width:  uint(wsMsg.Cols),
					})
				}
			}
		}
	}))

	// WebSocket for SSH host terminal
	app.Get("/ws/ssh/:hostId", websocket.New(func(c *websocket.Conn) {
		hostID := c.Params("hostId")

		client, err := dialSSH(hostID)
		if err != nil {
			errMsg, _ := json.Marshal(map[string]string{"type": "error", "data": err.Error()})
			c.WriteMessage(websocket.TextMessage, errMsg)
			return
		}
		defer client.Close()

		session, err := client.NewSession()
		if err != nil {
			errMsg, _ := json.Marshal(map[string]string{"type": "error", "data": err.Error()})
			c.WriteMessage(websocket.TextMessage, errMsg)
			return
		}
		defer session.Close()

		modes := ssh.TerminalModes{
			ssh.ECHO:          1,
			ssh.TTY_OP_ISPEED: 14400,
			ssh.TTY_OP_OSPEED: 14400,
		}
		if err := session.RequestPty("xterm-256color", 24, 80, modes); err != nil {
			errMsg, _ := json.Marshal(map[string]string{"type": "error", "data": err.Error()})
			c.WriteMessage(websocket.TextMessage, errMsg)
			return
		}

		stdout, err := session.StdoutPipe()
		if err != nil {
			errMsg, _ := json.Marshal(map[string]string{"type": "error", "data": err.Error()})
			c.WriteMessage(websocket.TextMessage, errMsg)
			return
		}
		stderr, err := session.StderrPipe()
		if err != nil {
			errMsg, _ := json.Marshal(map[string]string{"type": "error", "data": err.Error()})
			c.WriteMessage(websocket.TextMessage, errMsg)
			return
		}
		stdin, err := session.StdinPipe()
		if err != nil {
			errMsg, _ := json.Marshal(map[string]string{"type": "error", "data": err.Error()})
			c.WriteMessage(websocket.TextMessage, errMsg)
			return
		}

		if err := session.Shell(); err != nil {
			errMsg, _ := json.Marshal(map[string]string{"type": "error", "data": err.Error()})
			c.WriteMessage(websocket.TextMessage, errMsg)
			return
		}

		output := func(reader io.Reader) {
			buf := make([]byte, 4096)
			for {
				n, err := reader.Read(buf)
				if err != nil {
					return
				}
				if n > 0 {
					msg, _ := json.Marshal(map[string]string{"type": "output", "data": string(buf[:n])})
					c.WriteMessage(websocket.TextMessage, msg)
				}
			}
		}
		go output(stdout)
		go output(stderr)

		for {
			_, msg, err := c.ReadMessage()
			if err != nil {
				return
			}
			var wsMsg struct {
				Type string `json:"type"`
				Data string `json:"data"`
				Cols int    `json:"cols"`
				Rows int    `json:"rows"`
			}
			if json.Unmarshal(msg, &wsMsg) == nil {
				if wsMsg.Type == "input" {
					stdin.Write([]byte(wsMsg.Data))
				} else if wsMsg.Type == "resize" {
					if wsMsg.Cols > 0 && wsMsg.Rows > 0 {
						session.WindowChange(wsMsg.Rows, wsMsg.Cols)
					}
				}
			}
		}
	}))

	// =========================
	// Environment CRUD (admin-only)
	// =========================

	protected.Get("/environments", func(c *fiber.Ctx) error {
		envs := envStore.GetAll()
		// Test connection status for each
		for _, env := range envs {
			cli, err := dm.GetClient(env.ID)
			if err != nil {
				env.Status = "offline"
				continue
			}
			pingCtx, pingCancel := context.WithTimeout(context.Background(), 3*time.Second)
			pingResp, err := cli.Ping(pingCtx)
			pingCancel()
			if err != nil {
				env.Status = "offline"
			} else {
				env.Status = "online"
				// Try Info first, fallback to Ping response for version
				info, infoErr := cli.Info(context.Background())
				if infoErr == nil && info.ServerVersion != "" {
					env.DockerVersion = info.ServerVersion
				} else {
					// Fallback: try ServerVersion API
					sv, svErr := cli.ServerVersion(context.Background())
					if svErr == nil && sv.Version != "" {
						env.DockerVersion = sv.Version
					} else if pingResp.APIVersion != "" {
						env.DockerVersion = "API " + pingResp.APIVersion
					}
				}
			}
		}
		return c.JSON(envs)
	})

	protected.Post("/environments", func(c *fiber.Ctx) error {
		user := c.Locals("user").(*User)
		if user.Role != "admin" {
			return c.Status(403).JSON(fiber.Map{"error": "admin required"})
		}

		var env Environment
		if err := c.BodyParser(&env); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
		}
		if env.ID == "" || env.Name == "" {
			return c.Status(400).JSON(fiber.Map{"error": "id and name are required"})
		}

		envStore.Create(&env)

		// Hot-reload: add to hosts and create Docker client
		isLocal := env.ConnectionType == "socket"
		newHost := HostConfig{
			ID:      env.ID,
			Name:    env.Name,
			Address: env.Address,
			IsLocal: isLocal,
		}
		// Check if host already exists
		found := false
		for i, h := range hosts {
			if h.ID == env.ID {
				hosts[i] = newHost
				found = true
				break
			}
		}
		if !found {
			hosts = append(hosts, newHost)
		}
		// Force client recreation
		dm.mu.Lock()
		delete(dm.clients, env.ID)
		dm.mu.Unlock()

		return c.Status(201).JSON(env)
	})

	protected.Put("/environments/:id", func(c *fiber.Ctx) error {
		user := c.Locals("user").(*User)
		if user.Role != "admin" {
			return c.Status(403).JSON(fiber.Map{"error": "admin required"})
		}

		id := c.Params("id")
		existing := envStore.Get(id)
		if existing == nil {
			return c.Status(404).JSON(fiber.Map{"error": "environment not found"})
		}

		var env Environment
		if err := c.BodyParser(&env); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
		}
		env.ID = id
		envStore.Update(&env)

		// Hot-reload host config
		isLocal := env.ConnectionType == "socket"
		for i, h := range hosts {
			if h.ID == id {
				hosts[i] = HostConfig{
					ID:      id,
					Name:    env.Name,
					Address: env.Address,
					IsLocal: isLocal,
				}
				break
			}
		}
		dm.mu.Lock()
		delete(dm.clients, id)
		dm.mu.Unlock()

		return c.JSON(env)
	})

	protected.Delete("/environments/:id", func(c *fiber.Ctx) error {
		user := c.Locals("user").(*User)
		if user.Role != "admin" {
			return c.Status(403).JSON(fiber.Map{"error": "admin required"})
		}

		id := c.Params("id")
		if envStore.Get(id) == nil {
			return c.Status(404).JSON(fiber.Map{"error": "environment not found"})
		}

		envStore.Delete(id)

		// Remove from hosts
		for i, h := range hosts {
			if h.ID == id {
				hosts = append(hosts[:i], hosts[i+1:]...)
				break
			}
		}
		dm.mu.Lock()
		delete(dm.clients, id)
		dm.mu.Unlock()

		return c.JSON(fiber.Map{"success": true})
	})

	protected.Post("/environments/:id/test", func(c *fiber.Ctx) error {
		id := c.Params("id")
		env := envStore.Get(id)
		if env == nil {
			return c.Status(404).JSON(fiber.Map{"error": "environment not found"})
		}

		cli, err := dm.GetClient(id)
		if err != nil {
			return c.JSON(fiber.Map{"success": false, "error": err.Error()})
		}
		pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
		pingResp, err := cli.Ping(pingCtx)
		pingCancel()
		if err != nil {
			return c.JSON(fiber.Map{"success": false, "error": err.Error()})
		}
		info, infoErr := cli.Info(context.Background())
		version := info.ServerVersion
		os := info.OperatingSystem
		containers := info.Containers
		if infoErr != nil || version == "" {
			// Fallback: use ServerVersion from Ping or use version endpoint
			sv, svErr := cli.ServerVersion(context.Background())
			if svErr == nil {
				version = sv.Version
				os = sv.Os + "/" + sv.Arch
			} else if pingResp.APIVersion != "" {
				version = "API " + pingResp.APIVersion
			}
		}
		return c.JSON(fiber.Map{
			"success":       true,
			"dockerVersion": version,
			"os":            os,
			"containers":    containers,
		})
	})

	// Health check (no auth)
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "timestamp": time.Now().Unix()})
	})
}

// =============================================================================
// Image Update Check Functions
// =============================================================================

// checkImageUpdates checks all containers for available image updates
func checkImageUpdates(ctx context.Context, containers []ContainerInfo, dm *DockerManager) []ImageUpdate {
	var updates []ImageUpdate
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Limit concurrency to avoid overwhelming Docker hosts
	sem := make(chan struct{}, 5)

	for _, c := range containers {
		// Skip containers that are not running
		if c.State != "running" {
			continue
		}

		wg.Add(1)
		go func(container ContainerInfo) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			select {
			case <-ctx.Done():
				return
			default:
			}

			update := checkContainerUpdate(ctx, container, dm)
			if update != nil {
				mu.Lock()
				updates = append(updates, *update)
				mu.Unlock()
			}
		}(c)
	}

	wg.Wait()
	return updates
}

// checkSingleImageUpdate forces an update check for a specific container
func checkSingleImageUpdate(ctx context.Context, hostID, containerID string, dm *DockerManager) *ImageUpdate {
	dm.mu.RLock()
	cli, ok := dm.clients[hostID]
	dm.mu.RUnlock()
	if !ok {
		return nil
	}

	inspect, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil
	}

	container := ContainerInfo{
		ID:     containerID,
		Name:   strings.TrimPrefix(inspect.Name, "/"),
		Image:  inspect.Config.Image,
		HostID: hostID,
	}

	return checkContainerUpdate(ctx, container, dm)
}

// checkContainerUpdate checks if a container's image has an update available
// by comparing the local image digest with the remote registry digest.
func checkContainerUpdate(ctx context.Context, container ContainerInfo, dm *DockerManager) *ImageUpdate {
	// Check cache first (valid for 15 minutes)
	updateCacheMu.RLock()
	if cached, ok := updateCache[container.ID]; ok {
		if time.Now().Unix()-cached.CheckedAt < 900 {
			updateCacheMu.RUnlock()
			return cached
		}
	}
	updateCacheMu.RUnlock()

	dm.mu.RLock()
	cli, ok := dm.clients[container.HostID]
	dm.mu.RUnlock()
	if !ok {
		return nil
	}

	// Get current image info
	imageInspect, _, err := cli.ImageInspectWithRaw(ctx, container.Image)
	if err != nil {
		return nil
	}

	currentDigest := ""
	if len(imageInspect.RepoDigests) > 0 {
		currentDigest = imageInspect.RepoDigests[0]
	}

	// Extract current tag from image name (e.g., "nginx:1.25" -> "1.25")
	currentTag := "latest"
	if parts := strings.SplitN(container.Image, ":", 2); len(parts) == 2 {
		currentTag = parts[1]
	}

	update := &ImageUpdate{
		ContainerID:   container.ID,
		ContainerName: container.Name,
		Image:         container.Image,
		HostID:        container.HostID,
		CurrentDigest: currentDigest,
		CurrentTag:    currentTag,
		HasUpdate:     false,
		CheckedAt:     time.Now().Unix(),
	}

	// Check for watchtower exclusion label
	inspect, err := cli.ContainerInspect(ctx, container.ID)
	if err == nil {
		labels := inspect.Config.Labels
		if val, ok := labels["com.centurylinklabs.watchtower.enable"]; ok && val == "false" {
			// Container is explicitly excluded from updates
			updateCacheMu.Lock()
			updateCache[container.ID] = update
			updateCacheMu.Unlock()
			return update
		}
	}

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

	return update
}

// startUpdateChecker runs periodic update checks every 15 minutes
func startUpdateChecker(dm *DockerManager) {
	go func() {
		// Initial check after 30 seconds (let system stabilize)
		time.Sleep(30 * time.Second)
		runUpdateCheck(dm)

		ticker := time.NewTicker(15 * time.Minute)
		for range ticker.C {
			runUpdateCheck(dm)
		}
	}()
}

func runUpdateCheck(dm *DockerManager) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	containers, err := dm.GetAllContainers(ctx)
	if err != nil {
		log.Printf("Update checker: failed to get containers: %v", err)
		return
	}

	updates := checkImageUpdates(ctx, containers, dm)
	updateCount := 0
	for _, u := range updates {
		if u.HasUpdate {
			updateCount++
		}
	}
	if updateCount > 0 {
		log.Printf("Update checker: %d containers have updates available", updateCount)
	}
}

func parseDockerLogs(data []byte) []string {
	var lines []string
	for len(data) > 8 {
		// Docker log format: 8-byte header + message
		size := int(data[4])<<24 | int(data[5])<<16 | int(data[6])<<8 | int(data[7])
		if size <= 0 || 8+size > len(data) {
			break
		}
		line := string(data[8 : 8+size])
		lines = append(lines, strings.TrimSpace(line))
		data = data[8+size:]
	}
	return lines
}

// Background task to broadcast updates
func startBroadcaster(dm *DockerManager, hub *WSHub) {
	ticker := time.NewTicker(2 * time.Second)
	go func() {
		for range ticker.C {
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)

			// Get containers ONCE (shared by stats and hosts)
			containers, err := dm.GetAllContainers(ctx)
			if err != nil {
				cancel()
				continue
			}
			hub.Broadcast("containers", containers)

			// Run stats and hosts in parallel
			var bcastWg sync.WaitGroup

			bcastWg.Add(1)
			go func() {
				defer bcastWg.Done()
				stats := dm.GetStatsForContainers(ctx, containers)
				hub.Broadcast("stats", stats)
			}()

			bcastWg.Add(1)
			go func() {
				defer bcastWg.Done()
				hosts := dm.GetHostStats(ctx)
				hub.Broadcast("hosts", hosts)
			}()

			bcastWg.Wait()
			cancel()
		}
	}()
}

// =============================================================================
// Main
// =============================================================================

func main() {
	// Initialize stores
	userStore := NewUserStore()
	notifySvc := NewNotificationService(userStore)
	emailSvc := NewEmailService()
	envStore := NewEnvironmentStore("data/environments.json")
	dm := NewDockerManager(notifySvc)
	hub := NewWSHub()

	go hub.Run()
	startBroadcaster(dm, hub)
	startUpdateChecker(dm)

	// Initialize logging to both stdout and a file under `DATA_DIR/logs/backend.log`.
	// This makes it easier to fetch logs from the container filesystem.
	logDir := filepath.Join(dataDir, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("warning: could not create log dir %s: %v", logDir, err)
	}
	logPath := filepath.Join(logDir, "backend.log")
	lf, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("warning: could not open log file %s: %v", logPath, err)
	} else {
		mw := io.MultiWriter(os.Stdout, lf)
		log.SetOutput(mw)
	}

	app := fiber.New(fiber.Config{
		AppName:      "DockerVerse API",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	})

	// Middleware
	app.Use(fbRecover.New(fbRecover.Config{EnableStackTrace: true}))
	app.Use(logger.New())

	// Enable compression (gzip, brotli, deflate)
	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed, // Fast compression
	}))

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	// Serve static files (frontend)
	app.Static("/", "./public")

	// Setup API routes
	setupRoutes(app, dm, userStore, notifySvc, hub, emailSvc, envStore)

	// Debug: expose recent backend logs for easier collection from the container.
	// Query params: ?lines=200 (default 200)
	app.Get("/api/debug/logs", func(c *fiber.Ctx) error {
		if !isAdminRequest(c) {
			return c.Status(401).SendString("unauthorized")
		}
		lines := 200
		if s := c.Query("lines"); s != "" {
			if v, err := strconv.Atoi(s); err == nil && v > 0 && v <= 5000 {
				lines = v
			}
		}
		logPath := filepath.Join(dataDir, "logs", "backend.log")
		out, err := tailFileLines(logPath, lines)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.SendString(out)
	})

	// Get port from env or default
	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}

	log.Printf("🐳 DockerVerse starting on port %s", port)
	log.Printf("📊 Admin user: %s (change via env ADMIN_USER/ADMIN_PASS)", defaultAdmin)
	log.Printf("🔌 Docker hosts configured: %d", len(hosts))
	for _, h := range hosts {
		log.Printf("   - %s (%s) [%s] local=%v", h.ID, h.Name, h.Address, h.IsLocal)
	}
	if watchtowerToken != "" {
		log.Printf("🔄 Watchtower integration enabled (%d hosts)", len(watchtowerURLs))
	}
	log.Printf("🔔 Apprise URL: %s", appriseURL)
	log.Printf("📧 SMTP2Go API configured (from: %s)", smtpFrom)
	log.Fatal(app.Listen(":" + port))
}
