# Authentication System Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement a production-grade authentication system for DockerVerse matching the feature set of Dockhand and Portainer — including an auth settings UI, rate limiting, LDAP, OIDC, and API keys.

**Architecture:** The Go backend keeps its JSON-file persistence (UserStore) but gains new top-level structs for auth settings, LDAP config, OIDC config, and API keys, all persisted in separate JSON files alongside the existing `users.json`. The SvelteKit frontend gets a fully functional `settings/authentication` page and an updated login page that supports multiple providers.

**Tech Stack:** Go Fiber (backend), SvelteKit 2 + Svelte 5 runes (frontend), `go-ldap/ldap/v3` (LDAP client), `coreos/go-oidc/v3` + `golang.org/x/oauth2` (OIDC), existing JWT/bcrypt/TOTP stack.

---

## Research Summary

### What Dockhand does (github.com/Finsys/dockhand)
- **Local auth:** argon2id (64MB/3-iter/1-thread), PHC format strings
- **Sessions:** 32-byte random tokens, HttpOnly + SameSite=Strict + Secure flag, configurable timeout (24h default, 30d max)
- **TOTP:** SHA1, 6-digit, 30s, 10 backup codes **hashed with SHA256** before storage — DockerVerse stores them plaintext today
- **Rate limiting:** 5 failed attempts → 15-min lockout, in-memory sliding window, cleared on success
- **LDAP:** service-account bind, user search with injection prevention (`ldap.EscapeFilter`), auto-create user, group DN → admin/role mapping
- **OIDC:** PKCE + state + nonce, configurable claims (username/email/displayName/admin), role mappings from claims, OIDC users bypass MFA
- **Auth settings page:** enable/disable auth, default provider, session timeout, per-provider tabs (local/LDAP/OIDC)
- **Login page UI:** provider selector buttons, OIDC as SSO button above divider, two-step TOTP flow

### What Portainer does (github.com/portainer/portainer)
- **Local auth:** bcrypt
- **LDAP:** `LDAPSettings` struct — URL, TLS/StartTLS, AnonymousMode, ReaderDN, SearchSettings[], GroupSearchSettings[], AutoCreateUsers; timing attack mitigation (fake bind when user not found)
- **OAuth:** generic OAuth2 — ClientID/Secret, AuthorizationURI, AccessTokenURI, ResourceURI, Scopes, UserIdentifier (which field maps to username), OAuthAutoCreateUsers, DefaultTeamID, SSO, LogoutURI
- **API keys:** per-user, description (max 128 chars), password required to create, raw key shown once, encrypted digest stored, no expiry; `APIKey{ID, UserID, Description, Prefix, DateCreated, LastUsed, Digest}`
- **JWT:** contains userId, username, role, passwordChangeFlag; also set as cookie
- **Teams:** LDAP group sync, OAuth default team assignment

### DockerVerse current state
- `User`: bcrypt hash, role (admin/user), TOTPSecret, TOTPEnabled, **RecoveryCodes stored as plaintext hex** (bug)
- `UserStore`: JSON file, in-memory map
- `AppSettings`: only notification settings — **no auth settings at all**
- JWT access + refresh tokens, no cookie
- TOTP implemented, recovery codes implemented but unhashed
- No rate limiting, no LDAP, no OIDC, no API keys
- `settings/authentication` route: empty placeholder

---

## Phase 1 — Auth Settings + Security Hardening

### Task 1: Add `AuthConfig` struct to backend + persist to file

**Files:**
- Modify: `backend/main.go` (add struct + load/save + API endpoints)

**Step 1: Add the struct** (after `AppSettings`):

```go
// Auth configuration - persisted to data/auth-config.json
type AuthConfig struct {
    AuthEnabled         bool   `json:"authEnabled"`
    SessionTimeoutSecs  int    `json:"sessionTimeoutSecs"`  // default 86400 (24h)
    MaxLoginAttempts    int    `json:"maxLoginAttempts"`    // default 5
    LockoutDurationSecs int    `json:"lockoutDurationSecs"` // default 900 (15min)
    DefaultProvider     string `json:"defaultProvider"`     // "local" | "ldap" | "oidc"
}
```

**Step 2: Add load/save functions** (pattern matches UserStore.save()):

```go
var authConfigPath = filepath.Join(dataDir, "auth-config.json")

func loadAuthConfig() AuthConfig {
    cfg := AuthConfig{
        AuthEnabled:         true,
        SessionTimeoutSecs:  86400,
        MaxLoginAttempts:    5,
        LockoutDurationSecs: 900,
        DefaultProvider:     "local",
    }
    data, err := os.ReadFile(authConfigPath)
    if err == nil {
        json.Unmarshal(data, &cfg)
    }
    return cfg
}

func saveAuthConfig(cfg AuthConfig) error {
    data, err := json.MarshalIndent(cfg, "", "  ")
    if err != nil {
        return err
    }
    return os.WriteFile(authConfigPath, data, 0600)
}
```

**Step 3: Add global var** (near the top, alongside `store`):

```go
var authCfg AuthConfig
```

**Step 4: Initialize in `main()`** (right after `store` is loaded):

```go
authCfg = loadAuthConfig()
```

**Step 5: Add API endpoints** (in the protected admin section):

```go
// GET /api/settings/auth
protected.Get("/settings/auth", adminOnly(func(c *fiber.Ctx) error {
    return c.JSON(authCfg)
}))

// PUT /api/settings/auth
protected.Put("/settings/auth", adminOnly(func(c *fiber.Ctx) error {
    var updated AuthConfig
    if err := c.BodyParser(&updated); err != nil {
        return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
    }
    // Clamp values to sane ranges
    if updated.SessionTimeoutSecs < 300 { updated.SessionTimeoutSecs = 300 }
    if updated.MaxLoginAttempts < 1     { updated.MaxLoginAttempts = 1 }
    if updated.LockoutDurationSecs < 60 { updated.LockoutDurationSecs = 60 }
    authCfg = updated
    if err := saveAuthConfig(authCfg); err != nil {
        return c.Status(500).JSON(fiber.Map{"error": "failed to save"})
    }
    return c.JSON(authCfg)
}))
```

**Step 6: Build and run**

```bash
cd backend && go build ./... && echo "OK"
```
Expected: OK, no errors.

**Step 7: Test endpoints**

```bash
# Get auth config
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:3007/api/settings/auth | jq .

# Update
curl -s -X PUT -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"authEnabled":true,"sessionTimeoutSecs":3600,"maxLoginAttempts":5,"lockoutDurationSecs":900,"defaultProvider":"local"}' \
  http://localhost:3007/api/settings/auth | jq .
```
Expected: both return the config object.

**Step 8: Commit**

```bash
git add backend/main.go
git commit -m "feat(auth): add AuthConfig struct with persist + GET/PUT /api/settings/auth"
```

---

### Task 2: In-memory rate limiter (login lockout)

**Files:**
- Modify: `backend/main.go`

**Step 1: Add rate limiter state** (global, near other globals):

```go
type loginAttempt struct {
    count     int
    firstSeen time.Time
    lockedAt  time.Time
}

var (
    loginAttempts   = make(map[string]*loginAttempt) // key: username
    loginAttemptsMu sync.Mutex
)
```

**Step 2: Add helper functions** (before the login route handler):

```go
func isLockedOut(username string) bool {
    loginAttemptsMu.Lock()
    defer loginAttemptsMu.Unlock()
    a, ok := loginAttempts[username]
    if !ok { return false }
    if a.lockedAt.IsZero() { return false }
    return time.Since(a.lockedAt) < time.Duration(authCfg.LockoutDurationSecs)*time.Second
}

func recordFailedLogin(username string) {
    loginAttemptsMu.Lock()
    defer loginAttemptsMu.Unlock()
    a, ok := loginAttempts[username]
    if !ok {
        a = &loginAttempt{}
        loginAttempts[username] = a
    }
    // Reset window if older than lockout duration
    window := time.Duration(authCfg.LockoutDurationSecs) * time.Second
    if time.Since(a.firstSeen) > window {
        a.count = 0
        a.firstSeen = time.Now()
        a.lockedAt = time.Time{}
    }
    a.count++
    if a.count >= authCfg.MaxLoginAttempts {
        a.lockedAt = time.Now()
    }
}

func clearFailedLogins(username string) {
    loginAttemptsMu.Lock()
    defer loginAttemptsMu.Unlock()
    delete(loginAttempts, username)
}
```

**Step 3: Add cleanup goroutine** (in `main()`, after `authCfg` init):

```go
go func() {
    ticker := time.NewTicker(5 * time.Minute)
    for range ticker.C {
        loginAttemptsMu.Lock()
        cutoff := time.Duration(authCfg.LockoutDurationSecs) * time.Second
        for k, a := range loginAttempts {
            if time.Since(a.firstSeen) > cutoff*2 {
                delete(loginAttempts, k)
            }
        }
        loginAttemptsMu.Unlock()
    }
}()
```

**Step 4: Wire into login handler** — in the existing POST `/api/auth/login` handler, add at the top (before password check):

```go
// Rate limiting check
if isLockedOut(req.Username) {
    return c.Status(429).JSON(fiber.Map{
        "error":   "account temporarily locked",
        "message": fmt.Sprintf("Too many failed attempts. Try again in %d minutes.", authCfg.LockoutDurationSecs/60),
    })
}
```

And after `ValidateLogin` fails:

```go
recordFailedLogin(req.Username)
return c.Status(401).JSON(fiber.Map{"error": "invalid credentials"})
```

And after successful login (before token generation):

```go
clearFailedLogins(req.Username)
```

**Step 5: Build**

```bash
cd backend && go build ./... && echo "OK"
```

**Step 6: Manual test** — make 5 bad login attempts:

```bash
for i in {1..6}; do
  curl -s -X POST http://localhost:3007/api/auth/login \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"wrong"}' | jq .error
done
```
Expected: first 5 return "invalid credentials", 6th returns "account temporarily locked".

**Step 7: Commit**

```bash
git add backend/main.go
git commit -m "feat(auth): add in-memory rate limiter with configurable lockout"
```

---

### Task 3: Hash recovery codes before storage (security fix)

**Files:**
- Modify: `backend/main.go`

**Step 1: Add a helper to hash a recovery code** (near the TOTP functions):

```go
func hashRecoveryCode(code string) string {
    h := sha256.Sum256([]byte(code))
    return fmt.Sprintf("%x", h)
}
```

Add `"crypto/sha256"` to imports if not present.

**Step 2: Update `EnableTOTP`** — replace the loop that generates recovery codes:

```go
rawCodes := make([]string, 10)
hashedCodes := make([]string, 10)
for i := 0; i < 10; i++ {
    b := make([]byte, 8)
    rand.Read(b)
    raw := fmt.Sprintf("%x", b)[:16]
    rawCodes[i] = raw
    hashedCodes[i] = hashRecoveryCode(raw)
}
user.TOTPEnabled = true
user.RecoveryCodes = hashedCodes  // store hashed
s.save()
return rawCodes, nil  // return raw (shown once)
```

**Step 3: Update `RegenerateRecoveryCodes`** — same pattern as Step 2.

**Step 4: Update `UseRecoveryCode`** — compare hash of submitted code to stored hash:

```go
func (s *UserStore) UseRecoveryCode(username, code string) (bool, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    user, ok := s.Users[username]
    if !ok { return false, fmt.Errorf("user not found") }
    hashed := hashRecoveryCode(code)
    for i, rc := range user.RecoveryCodes {
        if rc == hashed {
            user.RecoveryCodes = append(user.RecoveryCodes[:i], user.RecoveryCodes[i+1:]...)
            s.save()
            return true, nil
        }
    }
    return false, nil
}
```

**Step 5: Build**

```bash
cd backend && go build ./... && echo "OK"
```

**Step 6: Test TOTP flow** (manual):
- Call `POST /api/auth/totp/setup` → get secret
- Call `POST /api/auth/totp/enable` with valid TOTP code → get recovery codes (raw)
- Check `users.json` — recovery codes should now look like `"64char-hex-sha256-hash"` not raw 16-char hex

**Step 7: Commit**

```bash
git add backend/main.go
git commit -m "fix(auth): hash recovery codes with SHA256 before storage"
```

---

### Task 4: Auth Settings UI — `settings/authentication` page (General tab)

**Files:**
- Modify: `frontend/src/routes/settings/authentication/+page.svelte`

**Step 1: Write the page** (replaces empty placeholder):

```svelte
<script lang="ts">
  import { onMount } from 'svelte';

  let config = $state({
    authEnabled: true,
    sessionTimeoutSecs: 86400,
    maxLoginAttempts: 5,
    lockoutDurationSecs: 900,
    defaultProvider: 'local',
  });
  let saving = $state(false);
  let saved = $state(false);
  let error = $state('');

  const sessionHours = $derived(Math.round(config.sessionTimeoutSecs / 3600 * 10) / 10);
  const lockoutMins = $derived(Math.round(config.lockoutDurationSecs / 60));

  onMount(async () => {
    const res = await fetch('/api/settings/auth', {
      headers: { Authorization: `Bearer ${localStorage.getItem('accessToken')}` },
    });
    if (res.ok) config = await res.json();
  });

  async function save() {
    saving = true; error = '';
    const res = await fetch('/api/settings/auth', {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${localStorage.getItem('accessToken')}`,
      },
      body: JSON.stringify(config),
    });
    saving = false;
    if (res.ok) { saved = true; setTimeout(() => saved = false, 2000); }
    else { const d = await res.json(); error = d.error || 'Failed to save'; }
  }
</script>

<div class="space-y-6 max-w-2xl">
  <div>
    <h2 class="text-lg font-semibold">Authentication Settings</h2>
    <p class="text-sm text-muted-foreground">Configure how users authenticate to DockerVerse.</p>
  </div>

  <!-- Enable auth toggle -->
  <div class="flex items-center justify-between rounded-lg border p-4">
    <div>
      <p class="font-medium">Require Authentication</p>
      <p class="text-sm text-muted-foreground">When disabled, anyone with network access can use DockerVerse.</p>
    </div>
    <input type="checkbox" bind:checked={config.authEnabled} class="toggle" />
  </div>

  <!-- Session timeout -->
  <div class="space-y-2">
    <label class="text-sm font-medium">Session Timeout</label>
    <div class="flex items-center gap-3">
      <input type="range" min="300" max="2592000" step="300"
             bind:value={config.sessionTimeoutSecs} class="flex-1" />
      <span class="text-sm w-24 text-right">{sessionHours}h</span>
    </div>
    <p class="text-xs text-muted-foreground">How long a session stays valid (5 min – 30 days).</p>
  </div>

  <!-- Rate limiting -->
  <div class="grid grid-cols-2 gap-4">
    <div class="space-y-2">
      <label class="text-sm font-medium">Max Login Attempts</label>
      <input type="number" min="1" max="20" bind:value={config.maxLoginAttempts}
             class="input w-full" />
    </div>
    <div class="space-y-2">
      <label class="text-sm font-medium">Lockout Duration (minutes)</label>
      <input type="number" min="1" max="1440" bind:value={lockoutMins}
             onchange={() => config.lockoutDurationSecs = lockoutMins * 60}
             class="input w-full" />
    </div>
  </div>

  {#if error}
    <p class="text-sm text-destructive">{error}</p>
  {/if}

  <button onclick={save} disabled={saving} class="btn btn-primary">
    {saving ? 'Saving…' : saved ? 'Saved!' : 'Save Changes'}
  </button>
</div>
```

**Step 2: Run frontend type check**

```bash
cd frontend && npm run check 2>&1 | grep -E "error|warning" | head -20
```
Expected: 0 errors.

**Step 3: Commit**

```bash
git add frontend/src/routes/settings/authentication/+page.svelte
git commit -m "feat(ui): implement auth settings page (general tab)"
```

---

## Phase 2 — LDAP Integration

### Task 5: Add LDAP dependency to backend

**Files:**
- Modify: `backend/go.mod`, `backend/go.sum`

**Step 1: Add go-ldap**

```bash
cd backend && go get github.com/go-ldap/ldap/v3
```

Expected: go.mod and go.sum updated.

**Step 2: Verify build**

```bash
go build ./... && echo "OK"
```

**Step 3: Commit**

```bash
git add backend/go.mod backend/go.sum
git commit -m "chore(deps): add go-ldap/ldap/v3"
```

---

### Task 6: Add `LdapConfig` struct + persistence + API

**Files:**
- Modify: `backend/main.go`

**Step 1: Add struct** (after `AuthConfig`):

```go
type LdapConfig struct {
    Enabled            bool   `json:"enabled"`
    ServerURL          string `json:"serverUrl"`          // e.g. "ldap://dc.example.com:389"
    BindDN             string `json:"bindDn"`             // service account DN
    BindPassword       string `json:"bindPassword"`
    BaseDN             string `json:"baseDn"`             // search base
    UserFilter         string `json:"userFilter"`         // "(uid=%s)" or "(&(objectClass=user)(sAMAccountName=%s))"
    UsernameAttr       string `json:"usernameAttr"`       // "uid" or "sAMAccountName"
    EmailAttr          string `json:"emailAttr"`          // "mail"
    DisplayNameAttr    string `json:"displayNameAttr"`    // "cn"
    GroupBaseDN        string `json:"groupBaseDn"`
    GroupFilter        string `json:"groupFilter"`        // "(&(objectClass=groupOfNames)(member=%s))"
    AdminGroup         string `json:"adminGroup"`         // DN of admin group
    AutoCreateUsers    bool   `json:"autoCreateUsers"`
    TLSEnabled         bool   `json:"tlsEnabled"`
    StartTLS           bool   `json:"startTls"`
}
```

**Step 2: Add load/save + global var** (same pattern as Task 1):

```go
var ldapConfigPath = filepath.Join(dataDir, "ldap-config.json")
var ldapCfg LdapConfig

func loadLdapConfig() LdapConfig {
    cfg := LdapConfig{UserFilter: "(uid=%s)", UsernameAttr: "uid", EmailAttr: "mail", DisplayNameAttr: "cn"}
    data, _ := os.ReadFile(ldapConfigPath)
    json.Unmarshal(data, &cfg)
    return cfg
}

func saveLdapConfig(cfg LdapConfig) error {
    data, err := json.MarshalIndent(cfg, "", "  ")
    if err != nil { return err }
    return os.WriteFile(ldapConfigPath, data, 0600)
}
```

In `main()`: `ldapCfg = loadLdapConfig()`

**Step 3: Add API endpoints** (admin-protected):

```go
protected.Get("/settings/ldap", adminOnly(func(c *fiber.Ctx) error {
    safe := ldapCfg
    safe.BindPassword = "" // never expose
    return c.JSON(safe)
}))

protected.Put("/settings/ldap", adminOnly(func(c *fiber.Ctx) error {
    var updated LdapConfig
    if err := c.BodyParser(&updated); err != nil {
        return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
    }
    // Preserve password if not sent
    if updated.BindPassword == "" { updated.BindPassword = ldapCfg.BindPassword }
    ldapCfg = updated
    return c.JSON(fiber.Map{"success": true, "error": saveLdapConfig(ldapCfg)})
}))

// Test LDAP connection
protected.Post("/settings/ldap/test", adminOnly(func(c *fiber.Ctx) error {
    err := testLdapConnection(ldapCfg)
    if err != nil { return c.Status(400).JSON(fiber.Map{"error": err.Error()}) }
    return c.JSON(fiber.Map{"success": true, "message": "LDAP connection successful"})
}))
```

**Step 4: Build**

```bash
go build ./... && echo "OK"
```

**Step 5: Commit**

```bash
git add backend/main.go
git commit -m "feat(auth): add LdapConfig struct + persist + admin CRUD endpoints"
```

---

### Task 7: Implement LDAP authentication logic

**Files:**
- Modify: `backend/main.go`

**Step 1: Add `testLdapConnection` function**:

```go
import ldap "github.com/go-ldap/ldap/v3"

func dialLdap(cfg LdapConfig) (*ldap.Conn, error) {
    var conn *ldap.Conn
    var err error
    if cfg.TLSEnabled {
        conn, err = ldap.DialURL(cfg.ServerURL, ldap.DialWithTLSConfig(&tls.Config{InsecureSkipVerify: false}))
    } else {
        conn, err = ldap.DialURL(cfg.ServerURL)
    }
    if err != nil { return nil, fmt.Errorf("dial: %w", err) }
    if cfg.StartTLS {
        if err = conn.StartTLS(&tls.Config{InsecureSkipVerify: false}); err != nil {
            conn.Close()
            return nil, fmt.Errorf("starttls: %w", err)
        }
    }
    return conn, nil
}

func testLdapConnection(cfg LdapConfig) error {
    conn, err := dialLdap(cfg)
    if err != nil { return err }
    defer conn.Close()
    if cfg.BindDN != "" {
        return conn.Bind(cfg.BindDN, cfg.BindPassword)
    }
    return nil // anonymous bind OK
}
```

**Step 2: Add `authenticateLdap` function**:

```go
type ldapUserInfo struct {
    DN          string
    Username    string
    Email       string
    DisplayName string
    IsAdmin     bool
}

func authenticateLdap(cfg LdapConfig, username, password string) (*ldapUserInfo, error) {
    conn, err := dialLdap(cfg)
    if err != nil { return nil, err }
    defer conn.Close()

    // Bind with service account
    if cfg.BindDN != "" {
        if err = conn.Bind(cfg.BindDN, cfg.BindPassword); err != nil {
            return nil, fmt.Errorf("service bind failed: %w", err)
        }
    }

    // Search for user
    escapedUser := ldap.EscapeFilter(username)
    filter := fmt.Sprintf(cfg.UserFilter, escapedUser)
    req := ldap.NewSearchRequest(
        cfg.BaseDN, ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 1, 10, false,
        filter,
        []string{"dn", cfg.UsernameAttr, cfg.EmailAttr, cfg.DisplayNameAttr},
        nil,
    )
    sr, err := conn.Search(req)
    if err != nil || len(sr.Entries) == 0 {
        // Timing attack mitigation: attempt fake bind
        conn.Bind("cn=notexist,dc=example,dc=com", "fakepassword")
        return nil, fmt.Errorf("user not found")
    }

    entry := sr.Entries[0]

    // Bind as user to verify password
    if err = conn.Bind(entry.DN, password); err != nil {
        return nil, fmt.Errorf("invalid credentials")
    }

    info := &ldapUserInfo{
        DN:          entry.DN,
        Username:    entry.GetAttributeValue(cfg.UsernameAttr),
        Email:       entry.GetAttributeValue(cfg.EmailAttr),
        DisplayName: entry.GetAttributeValue(cfg.DisplayNameAttr),
    }
    if info.Username == "" { info.Username = username }

    // Check admin group membership
    if cfg.AdminGroup != "" && cfg.GroupBaseDN != "" {
        groupFilter := fmt.Sprintf(cfg.GroupFilter, ldap.EscapeFilter(entry.DN))
        groupReq := ldap.NewSearchRequest(
            cfg.GroupBaseDN, ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 5, false,
            groupFilter, []string{"dn"}, nil,
        )
        groupSr, err := conn.Search(groupReq)
        if err == nil {
            for _, g := range groupSr.Entries {
                if strings.EqualFold(g.DN, cfg.AdminGroup) {
                    info.IsAdmin = true
                    break
                }
            }
        }
    }

    return info, nil
}
```

**Step 3: Extend login handler** — in the POST `/api/auth/login` handler, after local auth fails or as alternative path when `authCfg.DefaultProvider == "ldap"`:

```go
// If LDAP enabled and no local user found, try LDAP
if ldapCfg.Enabled && user == nil {
    ldapInfo, ldapErr := authenticateLdap(ldapCfg, req.Username, req.Password)
    if ldapErr != nil {
        recordFailedLogin(req.Username)
        return c.Status(401).JSON(fiber.Map{"error": "invalid credentials"})
    }
    // Auto-create local user linked to LDAP
    if ldapCfg.AutoCreateUsers {
        role := "user"
        if ldapInfo.IsAdmin { role = "admin" }
        newUser := &User{
            ID:             uuid.New().String(),
            Username:       ldapInfo.Username,
            Email:          ldapInfo.Email,
            Role:           role,
            AuthProvider:   "ldap",
            EmailConfirmed: true,
            CreatedAt:      time.Now(),
            LastLogin:      time.Now(),
        }
        store.mu.Lock()
        store.Users[newUser.Username] = newUser
        store.mu.Unlock()
        store.save()
        user = newUser
    } else {
        return c.Status(401).JSON(fiber.Map{"error": "account not created in DockerVerse"})
    }
}
```

Add `AuthProvider string \`json:"authProvider"\`` field to the `User` struct.

**Step 4: Build**

```bash
go build ./... && echo "OK"
```

**Step 5: Test with a mock LDAP** (optional — use `ldap3-server` docker image):

```bash
docker run -d --name test-ldap -p 389:389 rroemhild/test-openldap
curl -X POST http://localhost:3007/api/settings/ldap/test \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"serverUrl":"ldap://localhost:389","baseDn":"dc=planetexpress,dc=com"}'
```

**Step 6: Commit**

```bash
git add backend/main.go
git commit -m "feat(auth): implement LDAP authentication with auto-create and admin group mapping"
```

---

### Task 8: LDAP Settings UI

**Files:**
- Modify: `frontend/src/routes/settings/authentication/+page.svelte`

**Step 1: Add LDAP tab to the settings page** — restructure the page into tabs: "General" | "LDAP" | "OIDC" | "API Keys".

The general tab content from Task 4 stays. Add LDAP tab:

```svelte
<!-- LDAP tab -->
{#if activeTab === 'ldap'}
<div class="space-y-4 max-w-2xl">
  <div class="flex items-center justify-between">
    <div>
      <p class="font-medium">Enable LDAP / Active Directory</p>
      <p class="text-sm text-muted-foreground">Authenticate users against your directory server.</p>
    </div>
    <input type="checkbox" bind:checked={ldap.enabled} class="toggle" />
  </div>

  <div class="grid grid-cols-1 gap-4">
    <div>
      <label class="label">Server URL</label>
      <input class="input" placeholder="ldap://dc.example.com:389" bind:value={ldap.serverUrl} />
    </div>
    <div class="grid grid-cols-2 gap-4">
      <label class="flex items-center gap-2">
        <input type="checkbox" bind:checked={ldap.tlsEnabled} /> Use TLS (LDAPS)
      </label>
      <label class="flex items-center gap-2">
        <input type="checkbox" bind:checked={ldap.startTls} /> StartTLS
      </label>
    </div>
    <div class="grid grid-cols-2 gap-4">
      <div>
        <label class="label">Bind DN (service account)</label>
        <input class="input" placeholder="cn=svc,dc=example,dc=com" bind:value={ldap.bindDn} />
      </div>
      <div>
        <label class="label">Bind Password</label>
        <input type="password" class="input" placeholder="••••••" bind:value={ldap.bindPassword} />
      </div>
    </div>
    <div>
      <label class="label">Base DN</label>
      <input class="input" placeholder="dc=example,dc=com" bind:value={ldap.baseDn} />
    </div>
    <div>
      <label class="label">User Search Filter</label>
      <input class="input" placeholder="(uid=%s)" bind:value={ldap.userFilter} />
      <p class="text-xs text-muted-foreground mt-1">Use %s as username placeholder. AD: (&(objectClass=user)(sAMAccountName=%s))</p>
    </div>
    <div class="grid grid-cols-3 gap-4">
      <div>
        <label class="label">Username Attribute</label>
        <input class="input" placeholder="uid" bind:value={ldap.usernameAttr} />
      </div>
      <div>
        <label class="label">Email Attribute</label>
        <input class="input" placeholder="mail" bind:value={ldap.emailAttr} />
      </div>
      <div>
        <label class="label">Display Name Attribute</label>
        <input class="input" placeholder="cn" bind:value={ldap.displayNameAttr} />
      </div>
    </div>

    <details class="border rounded-lg p-4">
      <summary class="cursor-pointer font-medium">Group / Role Mapping (optional)</summary>
      <div class="mt-4 space-y-3">
        <div>
          <label class="label">Group Base DN</label>
          <input class="input" bind:value={ldap.groupBaseDn} />
        </div>
        <div>
          <label class="label">Group Filter</label>
          <input class="input" placeholder="(&(objectClass=groupOfNames)(member=%s))" bind:value={ldap.groupFilter} />
        </div>
        <div>
          <label class="label">Admin Group DN</label>
          <input class="input" placeholder="cn=docker-admins,ou=groups,dc=example,dc=com" bind:value={ldap.adminGroup} />
        </div>
      </div>
    </details>

    <label class="flex items-center gap-2">
      <input type="checkbox" bind:checked={ldap.autoCreateUsers} />
      Auto-create users on first login
    </label>
  </div>

  <div class="flex gap-3">
    <button onclick={testLdap} disabled={testingLdap} class="btn btn-secondary">
      {testingLdap ? 'Testing…' : 'Test Connection'}
    </button>
    <button onclick={saveLdap} disabled={savingLdap} class="btn btn-primary">
      {savingLdap ? 'Saving…' : 'Save'}
    </button>
  </div>
  {#if ldapTestResult}
    <p class="text-sm" class:text-green-500={ldapTestResult.ok} class:text-destructive={!ldapTestResult.ok}>
      {ldapTestResult.msg}
    </p>
  {/if}
</div>
{/if}
```

**Step 2: Add fetch logic** in `<script>`:

```typescript
let ldap = $state({ enabled: false, serverUrl: '', bindDn: '', bindPassword: '',
  baseDn: '', userFilter: '(uid=%s)', usernameAttr: 'uid', emailAttr: 'mail',
  displayNameAttr: 'cn', groupBaseDn: '', groupFilter: '', adminGroup: '',
  autoCreateUsers: false, tlsEnabled: false, startTls: false });
let savingLdap = $state(false);
let testingLdap = $state(false);
let ldapTestResult = $state<{ok:boolean, msg:string} | null>(null);

onMount(async () => {
  const [authRes, ldapRes] = await Promise.all([
    fetch('/api/settings/auth', { headers: authHeaders() }),
    fetch('/api/settings/ldap', { headers: authHeaders() }),
  ]);
  if (authRes.ok) config = await authRes.json();
  if (ldapRes.ok) ldap = await ldapRes.json();
});

async function saveLdap() {
  savingLdap = true;
  const res = await fetch('/api/settings/ldap', {
    method: 'PUT', headers: { ...authHeaders(), 'Content-Type': 'application/json' },
    body: JSON.stringify(ldap),
  });
  savingLdap = false;
}

async function testLdap() {
  testingLdap = true; ldapTestResult = null;
  const res = await fetch('/api/settings/ldap/test', {
    method: 'POST', headers: authHeaders(),
  });
  const d = await res.json();
  ldapTestResult = res.ok ? { ok: true, msg: d.message } : { ok: false, msg: d.error };
  testingLdap = false;
}
```

**Step 3: Type check**

```bash
cd frontend && npm run check 2>&1 | grep error
```

**Step 4: Commit**

```bash
git add frontend/src/routes/settings/authentication/+page.svelte
git commit -m "feat(ui): add LDAP configuration tab to auth settings page"
```

---

## Phase 3 — OIDC Integration

### Task 9: Add OIDC dependencies + `OidcConfig` struct + API

**Files:**
- Modify: `backend/go.mod`, `backend/main.go`

**Step 1: Add dependencies**

```bash
cd backend
go get github.com/coreos/go-oidc/v3/oidc
go get golang.org/x/oauth2
```

**Step 2: Add struct** (after `LdapConfig`):

```go
type OidcConfig struct {
    Enabled          bool   `json:"enabled"`
    Name             string `json:"name"`             // Display name e.g. "Google" or "Keycloak"
    IssuerURL        string `json:"issuerUrl"`         // OIDC discovery URL
    ClientID         string `json:"clientId"`
    ClientSecret     string `json:"clientSecret"`
    RedirectURI      string `json:"redirectUri"`
    Scopes           string `json:"scopes"`            // "openid email profile"
    UsernameClaim    string `json:"usernameClaim"`     // "preferred_username" or "email"
    EmailClaim       string `json:"emailClaim"`        // "email"
    DisplayNameClaim string `json:"displayNameClaim"`  // "name"
    AdminClaim       string `json:"adminClaim"`        // claim that indicates admin (optional)
    AdminValue       string `json:"adminValue"`        // value of admin claim that grants admin role
    AutoCreateUsers  bool   `json:"autoCreateUsers"`
    PKCEEnabled      bool   `json:"pkceEnabled"`       // always true for public clients
}
```

**Step 3: Add PKCE state store** (in-memory, 10-minute TTL):

```go
type oidcState struct {
    State        string
    Nonce        string
    CodeVerifier string
    CreatedAt    time.Time
}

var (
    oidcStates   = make(map[string]*oidcState)
    oidcStatesMu sync.Mutex
)
```

**Step 4: Add load/save + API endpoints** (same pattern as previous tasks):

```go
// GET/PUT /api/settings/oidc
// POST /api/auth/oidc/start  → returns {authorizationUrl}
// GET  /api/auth/oidc/callback?code=...&state=... → validates + logs user in
```

**Step 5: Build**

```bash
go build ./... && echo "OK"
```

**Step 6: Commit**

```bash
git add backend/go.mod backend/go.sum backend/main.go
git commit -m "feat(auth): add OidcConfig struct + dependencies"
```

---

### Task 10: Implement OIDC auth flow (backend)

**Files:**
- Modify: `backend/main.go`

**Step 1: Add `startOidcFlow` handler** (POST `/api/auth/oidc/start`):

```go
app.Post("/api/auth/oidc/start", func(c *fiber.Ctx) error {
    cfg := oidcCfg
    if !cfg.Enabled { return c.Status(400).JSON(fiber.Map{"error": "OIDC not configured"}) }

    // Generate PKCE challenge
    verifier := make([]byte, 32)
    rand.Read(verifier)
    codeVerifier := base64.RawURLEncoding.EncodeToString(verifier)
    h := sha256.Sum256([]byte(codeVerifier))
    codeChallenge := base64.RawURLEncoding.EncodeToString(h[:])

    // Generate state + nonce
    stateBytes := make([]byte, 16)
    rand.Read(stateBytes)
    state := base64.RawURLEncoding.EncodeToString(stateBytes)
    nonceBytes := make([]byte, 16)
    rand.Read(nonceBytes)
    nonce := base64.RawURLEncoding.EncodeToString(nonceBytes)

    oidcStatesMu.Lock()
    oidcStates[state] = &oidcState{State: state, Nonce: nonce, CodeVerifier: codeVerifier, CreatedAt: time.Now()}
    oidcStatesMu.Unlock()

    // Build authorization URL
    provider, err := oidc.NewProvider(context.Background(), cfg.IssuerURL)
    if err != nil { return c.Status(500).JSON(fiber.Map{"error": "OIDC provider error"}) }

    scopes := strings.Fields(cfg.Scopes)
    if len(scopes) == 0 { scopes = []string{"openid", "email", "profile"} }

    oauth2Cfg := &oauth2.Config{
        ClientID:     cfg.ClientID,
        ClientSecret: cfg.ClientSecret,
        RedirectURL:  cfg.RedirectURI,
        Endpoint:     provider.Endpoint(),
        Scopes:       scopes,
    }

    authURL := oauth2Cfg.AuthCodeURL(state,
        oauth2.SetAuthURLParam("nonce", nonce),
        oauth2.SetAuthURLParam("code_challenge", codeChallenge),
        oauth2.SetAuthURLParam("code_challenge_method", "S256"),
    )

    return c.JSON(fiber.Map{"authorizationUrl": authURL})
})
```

**Step 2: Add `oidcCallback` handler** (GET `/api/auth/oidc/callback`):

```go
app.Get("/api/auth/oidc/callback", func(c *fiber.Ctx) error {
    state := c.Query("state")
    code := c.Query("code")
    if state == "" || code == "" {
        return c.Redirect("/login?error=invalid_callback")
    }

    oidcStatesMu.Lock()
    storedState, ok := oidcStates[state]
    if ok { delete(oidcStates, state) }
    oidcStatesMu.Unlock()

    if !ok || time.Since(storedState.CreatedAt) > 10*time.Minute {
        return c.Redirect("/login?error=state_expired")
    }

    cfg := oidcCfg
    provider, err := oidc.NewProvider(context.Background(), cfg.IssuerURL)
    if err != nil { return c.Redirect("/login?error=provider_error") }

    oauth2Cfg := &oauth2.Config{
        ClientID: cfg.ClientID, ClientSecret: cfg.ClientSecret,
        RedirectURL: cfg.RedirectURI, Endpoint: provider.Endpoint(),
    }

    token, err := oauth2Cfg.Exchange(context.Background(), code,
        oauth2.SetAuthURLParam("code_verifier", storedState.CodeVerifier))
    if err != nil { return c.Redirect("/login?error=exchange_failed") }

    // Verify ID token
    verifier := provider.Verifier(&oidc.Config{ClientID: cfg.ClientID})
    rawIDToken, ok := token.Extra("id_token").(string)
    if !ok { return c.Redirect("/login?error=no_id_token") }
    idToken, err := verifier.Verify(context.Background(), rawIDToken)
    if err != nil { return c.Redirect("/login?error=id_token_invalid") }

    var claims map[string]interface{}
    idToken.Claims(&claims)

    username := fmt.Sprint(claims[cfg.UsernameClaim])
    email := fmt.Sprint(claims[cfg.EmailClaim])

    // Find or create user
    user := store.GetUser(username)
    if user == nil {
        if !cfg.AutoCreateUsers { return c.Redirect("/login?error=user_not_provisioned") }
        role := "user"
        if cfg.AdminClaim != "" {
            if v, ok := claims[cfg.AdminClaim]; ok && fmt.Sprint(v) == cfg.AdminValue {
                role = "admin"
            }
        }
        user = &User{
            ID: uuid.New().String(), Username: username, Email: email,
            Role: role, AuthProvider: "oidc", EmailConfirmed: true, CreatedAt: time.Now(),
        }
        store.mu.Lock()
        store.Users[username] = user
        store.mu.Unlock()
        store.save()
    }

    user.LastLogin = time.Now()
    store.save()

    tokens, _ := generateTokens(user, false)
    // Redirect to frontend with tokens in fragment (or set cookie)
    return c.Redirect(fmt.Sprintf("/?token=%s", tokens.AccessToken))
})
```

**Step 3: Add state cleanup goroutine** (in `main()`):

```go
go func() {
    ticker := time.NewTicker(10 * time.Minute)
    for range ticker.C {
        oidcStatesMu.Lock()
        for k, s := range oidcStates {
            if time.Since(s.CreatedAt) > 15*time.Minute { delete(oidcStates, k) }
        }
        oidcStatesMu.Unlock()
    }
}()
```

**Step 4: Build**

```bash
go build ./... && echo "OK"
```

**Step 5: Commit**

```bash
git add backend/main.go
git commit -m "feat(auth): implement OIDC authentication flow with PKCE"
```

---

### Task 11: OIDC Settings UI + Login page provider selector

**Files:**
- Modify: `frontend/src/routes/settings/authentication/+page.svelte`
- Modify: `frontend/src/routes/login/+page.svelte`

**Step 1: Add OIDC tab to auth settings** (same structure as LDAP tab):

```svelte
<!-- OIDC tab -->
{#if activeTab === 'oidc'}
<div class="space-y-4 max-w-2xl">
  <div class="flex items-center justify-between">
    <div>
      <p class="font-medium">Enable OIDC / SSO</p>
      <p class="text-sm text-muted-foreground">Let users sign in with an external identity provider.</p>
    </div>
    <input type="checkbox" bind:checked={oidc.enabled} class="toggle" />
  </div>

  <div class="grid gap-4">
    <div>
      <label class="label">Provider Name</label>
      <input class="input" placeholder="Google / Keycloak / Azure AD" bind:value={oidc.name} />
    </div>
    <div>
      <label class="label">Issuer URL (OIDC Discovery)</label>
      <input class="input" placeholder="https://accounts.google.com" bind:value={oidc.issuerUrl} />
    </div>
    <div class="grid grid-cols-2 gap-4">
      <div>
        <label class="label">Client ID</label>
        <input class="input" bind:value={oidc.clientId} />
      </div>
      <div>
        <label class="label">Client Secret</label>
        <input type="password" class="input" placeholder="••••••" bind:value={oidc.clientSecret} />
      </div>
    </div>
    <div>
      <label class="label">Redirect URI</label>
      <input class="input" placeholder="https://dockerverse.yourdomain.com/api/auth/oidc/callback"
             bind:value={oidc.redirectUri} />
    </div>
    <div>
      <label class="label">Scopes</label>
      <input class="input" placeholder="openid email profile" bind:value={oidc.scopes} />
    </div>
    <div class="grid grid-cols-3 gap-4">
      <div>
        <label class="label">Username Claim</label>
        <input class="input" placeholder="preferred_username" bind:value={oidc.usernameClaim} />
      </div>
      <div>
        <label class="label">Email Claim</label>
        <input class="input" placeholder="email" bind:value={oidc.emailClaim} />
      </div>
      <div>
        <label class="label">Display Name Claim</label>
        <input class="input" placeholder="name" bind:value={oidc.displayNameClaim} />
      </div>
    </div>
    <details class="border rounded-lg p-4">
      <summary class="cursor-pointer font-medium">Admin Claim (optional)</summary>
      <div class="mt-4 grid grid-cols-2 gap-4">
        <div>
          <label class="label">Admin Claim Name</label>
          <input class="input" placeholder="roles" bind:value={oidc.adminClaim} />
        </div>
        <div>
          <label class="label">Admin Claim Value</label>
          <input class="input" placeholder="docker_admin" bind:value={oidc.adminValue} />
        </div>
      </div>
    </details>
    <label class="flex items-center gap-2">
      <input type="checkbox" bind:checked={oidc.autoCreateUsers} />
      Auto-create users on first login
    </label>
  </div>

  <button onclick={saveOidc} class="btn btn-primary">Save</button>
</div>
{/if}
```

**Step 2: Update login page** — when LDAP and/or OIDC are enabled, show provider selector:

Key additions to `login/+page.svelte`:

```svelte
<!-- Above the form, if OIDC enabled -->
{#if providers.oidc?.enabled}
  <button onclick={startOidc} class="btn btn-outline w-full flex items-center gap-2">
    <span>Continue with {providers.oidc.name}</span>
  </button>
  <div class="relative my-4">
    <div class="absolute inset-0 flex items-center"><div class="w-full border-t"></div></div>
    <div class="relative flex justify-center text-xs text-muted-foreground">
      <span class="bg-background px-2">or continue with username</span>
    </div>
  </div>
{/if}

<!-- Existing username/password form stays -->
```

And if LDAP is enabled, add a note below the form:

```svelte
{#if providers.ldap?.enabled}
  <p class="text-xs text-center text-muted-foreground mt-2">
    LDAP users: use your directory username and password
  </p>
{/if}
```

Fetch providers on mount:

```typescript
onMount(async () => {
  const res = await fetch('/api/auth/providers');
  if (res.ok) providers = await res.json();
});

async function startOidc() {
  const res = await fetch('/api/auth/oidc/start', { method: 'POST' });
  const d = await res.json();
  window.location.href = d.authorizationUrl;
}
```

Add `GET /api/auth/providers` endpoint to backend (public, no auth):

```go
app.Get("/api/auth/providers", func(c *fiber.Ctx) error {
    return c.JSON(fiber.Map{
        "ldap": fiber.Map{"enabled": ldapCfg.Enabled},
        "oidc": fiber.Map{"enabled": oidcCfg.Enabled, "name": oidcCfg.Name},
    })
})
```

**Step 3: Type check**

```bash
cd frontend && npm run check 2>&1 | grep error
```

**Step 4: Commit**

```bash
git add frontend/src/routes/settings/authentication/+page.svelte \
         frontend/src/routes/login/+page.svelte \
         backend/main.go
git commit -m "feat(ui): OIDC settings tab + login page provider selector"
```

---

## Phase 4 — API Keys

### Task 12: API key backend (model + endpoints)

**Files:**
- Modify: `backend/main.go`

**Step 1: Add `APIKey` struct + extend `UserStore`**:

```go
type APIKey struct {
    ID          string    `json:"id"`
    UserID      string    `json:"userId"`
    Description string    `json:"description"`
    Prefix      string    `json:"prefix"`      // first 8 chars of raw key (for display)
    Digest      string    `json:"-"`           // SHA256 of full raw key (for verification)
    CreatedAt   time.Time `json:"createdAt"`
    LastUsed    time.Time `json:"lastUsed,omitempty"`
}
```

Add to `UserStore`:

```go
type UserStore struct {
    Users   map[string]*User `json:"users"`
    APIKeys []APIKey         `json:"apiKeys"`
    Settings AppSettings     `json:"settings"`
    mu      sync.RWMutex
}
```

**Step 2: Add `createAPIKey` helper**:

```go
func (s *UserStore) CreateAPIKey(userID, description string) (string, *APIKey, error) {
    raw := make([]byte, 32)
    rand.Read(raw)
    rawKey := "dv_" + base64.RawURLEncoding.EncodeToString(raw) // prefix "dv_" for identification

    h := sha256.Sum256([]byte(rawKey))
    digest := fmt.Sprintf("%x", h)

    key := APIKey{
        ID:          uuid.New().String(),
        UserID:      userID,
        Description: description,
        Prefix:      rawKey[:10], // "dv_" + 7 chars
        Digest:      digest,
        CreatedAt:   time.Now(),
    }

    s.mu.Lock()
    s.APIKeys = append(s.APIKeys, key)
    s.mu.Unlock()
    s.save()

    return rawKey, &key, nil
}

func (s *UserStore) ValidateAPIKey(rawKey string) (*User, error) {
    h := sha256.Sum256([]byte(rawKey))
    digest := fmt.Sprintf("%x", h)
    s.mu.RLock()
    defer s.mu.RUnlock()
    for i, k := range s.APIKeys {
        if k.Digest == digest {
            user := s.Users[k.UserID] // userID is username here
            if user == nil { return nil, fmt.Errorf("user not found") }
            s.APIKeys[i].LastUsed = time.Now()
            return user, nil
        }
    }
    return nil, fmt.Errorf("invalid api key")
}
```

**Step 3: Add `X-API-Key` middleware** — extend `jwtMiddleware` or add parallel middleware:

```go
func apiKeyMiddleware(store *UserStore) fiber.Handler {
    return func(c *fiber.Ctx) error {
        key := c.Get("X-API-Key")
        if key == "" { return c.Next() } // let JWT middleware handle
        user, err := store.ValidateAPIKey(key)
        if err != nil { return c.Status(401).JSON(fiber.Map{"error": "invalid api key"}) }
        store.save() // persist LastUsed update
        c.Locals("user", user)
        return c.Next()
    }
}
```

Wire in before JWT middleware on protected routes.

**Step 4: Add API endpoints**:

```go
// List user's own API keys
protected.Get("/auth/api-keys", func(c *fiber.Ctx) error {
    user := c.Locals("user").(*User)
    var keys []APIKey
    store.mu.RLock()
    for _, k := range store.APIKeys {
        if k.UserID == user.Username {
            k.Digest = "" // never expose digest
            keys = append(keys, k)
        }
    }
    store.mu.RUnlock()
    return c.JSON(keys)
})

// Create API key (requires password confirmation)
protected.Post("/auth/api-keys", func(c *fiber.Ctx) error {
    user := c.Locals("user").(*User)
    var req struct {
        Description string `json:"description"`
        Password    string `json:"password"`
    }
    c.BodyParser(&req)
    if len(req.Description) == 0 || len(req.Description) > 128 {
        return c.Status(400).JSON(fiber.Map{"error": "description must be 1-128 characters"})
    }
    // Verify password (skip for LDAP/OIDC users)
    if user.AuthProvider == "local" || user.AuthProvider == "" {
        if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
            return c.Status(401).JSON(fiber.Map{"error": "invalid password"})
        }
    }
    rawKey, key, err := store.CreateAPIKey(user.Username, req.Description)
    if err != nil { return c.Status(500).JSON(fiber.Map{"error": "failed to create key"}) }
    return c.JSON(fiber.Map{"rawKey": rawKey, "apiKey": key})
})

// Delete API key
protected.Delete("/auth/api-keys/:id", func(c *fiber.Ctx) error {
    user := c.Locals("user").(*User)
    id := c.Params("id")
    store.mu.Lock()
    newKeys := store.APIKeys[:0]
    found := false
    for _, k := range store.APIKeys {
        if k.ID == id && k.UserID == user.Username { found = true; continue }
        newKeys = append(newKeys, k)
    }
    store.APIKeys = newKeys
    store.mu.Unlock()
    if !found { return c.Status(404).JSON(fiber.Map{"error": "not found"}) }
    store.save()
    return c.JSON(fiber.Map{"success": true})
})
```

**Step 5: Build**

```bash
go build ./... && echo "OK"
```

**Step 6: Test**

```bash
# Create a key
KEY_RESP=$(curl -s -X POST http://localhost:3007/api/auth/api-keys \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"description":"CLI access","password":"yourpassword"}')
echo $KEY_RESP | jq .rawKey

RAW_KEY=$(echo $KEY_RESP | jq -r .rawKey)

# Use the key
curl -s -H "X-API-Key: $RAW_KEY" http://localhost:3007/api/containers | jq .[0].name
```
Expected: first returns `rawKey`, second returns a container name.

**Step 7: Commit**

```bash
git add backend/main.go
git commit -m "feat(auth): add API key model, CRUD endpoints, and X-API-Key middleware"
```

---

### Task 13: API Keys UI (Profile Settings)

**Files:**
- Modify: `frontend/src/routes/settings/authentication/+page.svelte` (add "API Keys" tab)

**Step 1: Add API Keys tab content**:

```svelte
<!-- API Keys tab -->
{#if activeTab === 'apikeys'}
<div class="space-y-4">
  <div class="flex items-center justify-between">
    <div>
      <p class="font-medium">API Keys</p>
      <p class="text-sm text-muted-foreground">Use API keys to authenticate scripts and CLI tools.</p>
    </div>
    <button onclick={() => showCreateKey = true} class="btn btn-primary btn-sm">
      + New API Key
    </button>
  </div>

  {#if newKeyValue}
    <div class="rounded-lg border border-yellow-500 bg-yellow-500/10 p-4">
      <p class="text-sm font-medium text-yellow-600">Save this key now — it won't be shown again.</p>
      <code class="mt-2 block break-all text-sm font-mono bg-muted rounded p-2">{newKeyValue}</code>
      <button onclick={() => navigator.clipboard.writeText(newKeyValue)} class="btn btn-sm mt-2">
        Copy
      </button>
    </div>
  {/if}

  <!-- Create dialog -->
  {#if showCreateKey}
    <div class="rounded-lg border p-4 space-y-3 bg-muted/50">
      <input class="input w-full" placeholder="Description (e.g. Home server CI)" bind:value={newKeyDesc} />
      <input type="password" class="input w-full" placeholder="Your password (to confirm)" bind:value={newKeyPassword} />
      <div class="flex gap-2">
        <button onclick={createKey} class="btn btn-primary btn-sm">Create</button>
        <button onclick={() => showCreateKey = false} class="btn btn-secondary btn-sm">Cancel</button>
      </div>
    </div>
  {/if}

  <!-- Key list -->
  <div class="divide-y rounded-lg border">
    {#each apiKeys as key}
      <div class="flex items-center justify-between px-4 py-3">
        <div>
          <p class="font-medium text-sm">{key.description}</p>
          <p class="text-xs text-muted-foreground font-mono">{key.prefix}…</p>
          <p class="text-xs text-muted-foreground">
            Created {new Date(key.createdAt).toLocaleDateString()}
            {key.lastUsed ? ` · Last used ${new Date(key.lastUsed).toLocaleDateString()}` : ''}
          </p>
        </div>
        <button onclick={() => deleteKey(key.id)} class="btn btn-ghost btn-sm text-destructive">
          Revoke
        </button>
      </div>
    {:else}
      <p class="p-4 text-sm text-muted-foreground">No API keys yet.</p>
    {/each}
  </div>
</div>
{/if}
```

**Step 2: Add fetch logic**:

```typescript
let apiKeys = $state<any[]>([]);
let showCreateKey = $state(false);
let newKeyDesc = $state('');
let newKeyPassword = $state('');
let newKeyValue = $state('');

async function loadApiKeys() {
  const res = await fetch('/api/auth/api-keys', { headers: authHeaders() });
  if (res.ok) apiKeys = await res.json();
}

async function createKey() {
  const res = await fetch('/api/auth/api-keys', {
    method: 'POST',
    headers: { ...authHeaders(), 'Content-Type': 'application/json' },
    body: JSON.stringify({ description: newKeyDesc, password: newKeyPassword }),
  });
  if (res.ok) {
    const d = await res.json();
    newKeyValue = d.rawKey;
    showCreateKey = false;
    newKeyDesc = ''; newKeyPassword = '';
    loadApiKeys();
  }
}

async function deleteKey(id: string) {
  await fetch(`/api/auth/api-keys/${id}`, { method: 'DELETE', headers: authHeaders() });
  loadApiKeys();
}
```

Also add `loadApiKeys()` call in `onMount`.

**Step 3: Type check**

```bash
cd frontend && npm run check 2>&1 | grep error
```

**Step 4: Commit**

```bash
git add frontend/src/routes/settings/authentication/+page.svelte
git commit -m "feat(ui): API keys management tab in auth settings"
```

---

## Phase 5 — Deploy & Verify

### Task 14: Deploy and end-to-end verification

**Step 1: Build backend**

```bash
cd backend && go build ./... && echo "OK"
```

**Step 2: Build frontend**

```bash
cd frontend && npm run build 2>&1 | tail -5
```

**Step 3: Deploy to Raspberry Pi**

```bash
cd /path/to/dockerverse && ./deploy-to-raspi.sh
```

**Step 4: Verify auth settings page loads**

Open `http://192.168.1.145:3007/settings/authentication` — should show General | LDAP | OIDC | API Keys tabs.

**Step 5: Verify rate limiting**

```bash
for i in {1..6}; do
  curl -s -X POST http://192.168.1.145:3007/api/auth/login \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"wrong"}' | jq -r .error
done
```
Expected: 5× "invalid credentials", then "account temporarily locked".

**Step 6: Verify API keys**

- Log in to UI → Settings → Authentication → API Keys tab
- Create a key (enter password), copy it
- Use it in a curl: `curl -H "X-API-Key: dv_..." http://192.168.1.145:3007/api/containers`
- Expected: returns container list

**Step 7: Final commit if adjustments made**

```bash
git add -A && git commit -m "fix: post-deploy adjustments"
```

**Step 8: Git push**

```bash
git push origin master
```

---

## Summary

| Phase | Feature | Priority |
|-------|---------|----------|
| 1 | Auth settings backend + UI (enable/disable, session timeout) | HIGH |
| 1 | Rate limiting + lockout | HIGH |
| 1 | Recovery codes security (SHA256 hashing) | HIGH (bug fix) |
| 2 | LDAP integration + settings UI | MEDIUM |
| 3 | OIDC integration + settings UI + login provider selector | MEDIUM |
| 4 | API keys (create/revoke + X-API-Key middleware) | MEDIUM |
| 5 | Deploy + E2E verification | HIGH |

**Estimated complexity:** Each task 30-90 min. Full system ~1-2 days.

**Notes:**
- LDAP and OIDC require real infrastructure to test end-to-end. Use `rroemhild/test-openldap` Docker image for LDAP testing. Use an existing Keycloak instance (already running in infrastructure) for OIDC.
- The `UserStore` JSON persistence works fine for this scale. No need to migrate to SQL.
- OIDC callback token delivery via URL fragment (`/?token=...`) is a simplification — a more secure approach uses `httpOnly` cookies or short-lived authorization codes.
