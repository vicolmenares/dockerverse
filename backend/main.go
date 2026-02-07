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
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/websocket/v2"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
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
	AppriseURL      string   `json:"appriseUrl"`
	AppriseKey      string   `json:"appriseKey"`
	TelegramEnabled bool     `json:"telegramEnabled"`
	TelegramURL     string   `json:"telegramUrl"`
	EmailEnabled    bool     `json:"emailEnabled"`
	NotifyOnStop    bool     `json:"notifyOnStop"`
	NotifyOnStart   bool     `json:"notifyOnStart"`
	NotifyOnHighCPU bool     `json:"notifyOnHighCpu"`
	NotifyOnHighMem bool     `json:"notifyOnHighMem"`
	NotifyTags      []string `json:"notifyTags"`
}

var hosts = []HostConfig{
	{ID: "raspi1", Name: "Raspi Main", Address: "unix:///var/run/docker.sock", IsLocal: true},
	{ID: "raspi2", Name: "Raspi Secondary", Address: "http://192.168.1.146:2375", IsLocal: false},
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
)

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
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
	PasswordHash   string    `json:"passwordHash,omitempty"`
	Role           string    `json:"role"` // "admin" or "user"
	EmailConfirmed bool      `json:"emailConfirmed"`
	ConfirmToken   string    `json:"-"`
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

type HostStats struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	ContainerCount int     `json:"containerCount"`
	RunningCount   int     `json:"runningCount"`
	CPUPercent     float64 `json:"cpuPercent"`
	MemoryPercent  float64 `json:"memoryPercent"`
	Online         bool    `json:"online"`
}

type SSEMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type JWTClaims struct {
	UserID   string `json:"userId"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

type LoginRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	RememberMe bool   `json:"rememberMe"`
}

type LoginResponse struct {
	User   *User      `json:"user"`
	Tokens AuthTokens `json:"tokens"`
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

// =============================================================================
// User Store
// =============================================================================

func NewUserStore() *UserStore {
	store := &UserStore{
		Users:      make(map[string]*User),
		resetCodes: make(map[string]*ResetCode),
		Settings: AppSettings{
			CPUThreshold:    80.0,
			MemoryThreshold: 80.0,
			AppriseURL:      appriseURL,
			AppriseKey:      "dockerverse",
			NotifyOnStop:    true,
			NotifyOnHighCPU: true,
			NotifyOnHighMem: true,
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
	return s.Users[username]
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
		PasswordHash:   "", // Never expose in API
		Role:           u.Role,
		EmailConfirmed: u.EmailConfirmed,
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

	user, exists := s.Users[username]
	if !exists {
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

			cli, err := dm.GetClient(h.ID)
			if err != nil {
				return
			}

			containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
			if err != nil {
				log.Printf("Error listing containers for %s: %v", h.Name, err)
				return
			}

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
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)
	cpuPercent := 0.0
	if systemDelta > 0 && cpuDelta > 0 {
		cpuPercent = (cpuDelta / systemDelta) * float64(len(stats.CPUStats.CPUUsage.PercpuUsage)) * 100.0
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

func (dm *DockerManager) GetAllStats(ctx context.Context) ([]ContainerStats, error) {
	containers, err := dm.GetAllContainers(ctx)
	if err != nil {
		return nil, err
	}

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

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			stats, err := dm.GetContainerStats(ctx, cont.HostID, cont.ID)
			if err != nil {
				return
			}

			stats.Name = cont.Name

			// Check for high resource usage notifications
			go dm.notifySvc.NotifyHighResource(cont.Name, stats.CPUPercent, stats.MemoryPct)

			mu.Lock()
			allStats = append(allStats, *stats)
			mu.Unlock()
		}(c)
	}

	wg.Wait()
	return allStats, nil
}

func (dm *DockerManager) GetHostStats(ctx context.Context) []HostStats {
	hostStats := make([]HostStats, len(hosts))
	var wg sync.WaitGroup

	for i, host := range hosts {
		wg.Add(1)
		go func(idx int, h HostConfig) {
			defer wg.Done()

			hs := HostStats{
				ID:     h.ID,
				Name:   h.Name,
				Online: false,
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
				hostStats[idx] = hs
				return
			}

			hs.Online = true

			// Get containers
			containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
			if err == nil {
				hs.ContainerCount = len(containers)
				var totalCPU, totalMem float64
				var runningCount int

				for _, c := range containers {
					if c.State == "running" {
						runningCount++
						// Get stats for running container with timeout
						statsCtx, statsCancel := context.WithTimeout(ctx, 2*time.Second)
						statsResp, err := cli.ContainerStats(statsCtx, c.ID, false)
						if err == nil {
							var stats types.StatsJSON
							if json.NewDecoder(statsResp.Body).Decode(&stats) == nil {
								// Calculate CPU percentage
								cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
								systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)
								if systemDelta > 0 && cpuDelta > 0 {
									numCPUs := float64(len(stats.CPUStats.CPUUsage.PercpuUsage))
									if numCPUs == 0 {
										numCPUs = 1
									}
									totalCPU += (cpuDelta / systemDelta) * numCPUs * 100.0
								}
								// Calculate memory percentage
								if stats.MemoryStats.Limit > 0 {
									totalMem += float64(stats.MemoryStats.Usage) / float64(stats.MemoryStats.Limit) * 100.0
								}
							}
							statsResp.Body.Close()
						}
						statsCancel()
					}
				}
				hs.RunningCount = runningCount
				hs.CPUPercent = totalCPU
				hs.MemoryPercent = totalMem
			}

			hostStats[idx] = hs
		}(i, host)
	}

	wg.Wait()
	return hostStats
}

func (dm *DockerManager) ContainerAction(ctx context.Context, hostID, containerID, action string) error {
	cli, err := dm.GetClient(hostID)
	if err != nil {
		return err
	}

	switch action {
	case "start":
		return cli.ContainerStart(ctx, containerID, container.StartOptions{})
	case "stop":
		timeout := 10
		return cli.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout})
	case "restart":
		timeout := 10
		return cli.ContainerRestart(ctx, containerID, container.StopOptions{Timeout: &timeout})
	case "pause":
		return cli.ContainerPause(ctx, containerID)
	case "unpause":
		return cli.ContainerUnpause(ctx, containerID)
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
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

// =============================================================================
// WebSocket Hub for Real-time Updates
// =============================================================================

type WSHub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan []byte
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mu         sync.Mutex
}

func NewWSHub() *WSHub {
	return &WSHub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
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

func setupRoutes(app *fiber.App, dm *DockerManager, store *UserStore, notifySvc *NotificationService, hub *WSHub, emailSvc *EmailService) {
	api := app.Group("/api")

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

		tokens, err := generateTokens(user, req.RememberMe)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "failed to generate tokens"})
		}

		return c.JSON(LoginResponse{
			User:   user.SafeUser(),
			Tokens: *tokens,
		})
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

	// Stats
	protected.Get("/stats", func(c *fiber.Ctx) error {
		ctx := context.Background()
		stats, err := dm.GetAllStats(ctx)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(stats)
	})

	// Container actions
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

	// SSE endpoint for real-time stats (legacy, kept for compatibility)
	protected.Get("/events", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/event-stream")
		c.Set("Cache-Control", "no-cache")
		c.Set("Connection", "keep-alive")
		c.Set("Transfer-Encoding", "chunked")

		ctx := c.Context()

		c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					// Get containers
					containers, err := dm.GetAllContainers(context.Background())
					if err == nil {
						msg := SSEMessage{Type: "containers", Data: containers}
						data, _ := json.Marshal(msg)
						fmt.Fprintf(w, "data: %s\n\n", data)
					}

					// Get stats
					stats, err := dm.GetAllStats(context.Background())
					if err == nil {
						msg := SSEMessage{Type: "stats", Data: stats}
						data, _ := json.Marshal(msg)
						fmt.Fprintf(w, "data: %s\n\n", data)
					}

					// Get host stats
					hostStats := dm.GetHostStats(context.Background())
					msg := SSEMessage{Type: "hosts", Data: hostStats}
					data, _ := json.Marshal(msg)
					fmt.Fprintf(w, "data: %s\n\n", data)

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
	app.Get("/ws/terminal/:hostId/:containerId", websocket.New(func(c *websocket.Conn) {
		hostID := c.Params("hostId")
		containerID := c.Params("containerId")

		cli, err := dm.GetClient(hostID)
		if err != nil {
			errMsg, _ := json.Marshal(map[string]string{"type": "error", "data": err.Error()})
			c.WriteMessage(websocket.TextMessage, errMsg)
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

	// Health check (no auth)
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "timestamp": time.Now().Unix()})
	})
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
			ctx := context.Background()

			// Broadcast containers
			containers, err := dm.GetAllContainers(ctx)
			if err == nil {
				hub.Broadcast("containers", containers)
			}

			// Broadcast stats
			stats, err := dm.GetAllStats(ctx)
			if err == nil {
				hub.Broadcast("stats", stats)
			}

			// Broadcast hosts
			hosts := dm.GetHostStats(ctx)
			hub.Broadcast("hosts", hosts)
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
	dm := NewDockerManager(notifySvc)
	hub := NewWSHub()

	go hub.Run()
	startBroadcaster(dm, hub)

	app := fiber.New(fiber.Config{
		AppName:      "DockerVerse API",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	})

	// Middleware
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
	setupRoutes(app, dm, userStore, notifySvc, hub, emailSvc)

	// Get port from env or default
	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}

	log.Printf("🐳 DockerVerse starting on port %s", port)
	log.Printf("📊 Admin user: %s (change via env ADMIN_USER/ADMIN_PASS)", defaultAdmin)
	log.Printf("🔔 Apprise URL: %s", appriseURL)
	log.Printf("📧 SMTP2Go API configured (from: %s)", smtpFrom)
	log.Fatal(app.Listen(":" + port))
}
