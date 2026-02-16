# Backend Modular Refactoring - Design Document

**Date:** 2026-02-16
**Author:** Claude Sonnet 4.5 & User
**Status:** Approved
**Version:** v2.5.0 (refactoring milestone)

---

## Context

### Current State
- **File:** `backend/main.go` - **4,518 lines** in a single monolithic file
- **Problem:** Adding new features (like container updates) risks breaking existing functionality
- **Need:** Modular architecture where changes to one feature don't affect others

### Sections in Current main.go
1. Configuration (~100 lines)
2. Types (~300 lines)
3. User Store (~200 lines)
4. JWT Functions (~150 lines)
5. Apprise Notification Service (~100 lines)
6. Email Service (~150 lines)
7. Docker Client Manager (~400 lines)
8. WebSocket Hub (~200 lines)
9. HTTP Handlers (~2,500 lines)
10. Image Update Check Functions (~300 lines)
11. Main (~118 lines)

---

## Goals

✅ **Isolation** - Changes in one module don't affect others
✅ **Testability** - Each module can be tested independently
✅ **Readability** - Files of 100-300 lines instead of 4,500
✅ **Maintainability** - Easy to find and modify specific functionality
✅ **Collaboration** - Multiple developers can work without conflicts

---

## Target Architecture

```
backend/
├── main.go                 # Bootstrap + routing only (~200 lines)
├── config/
│   └── config.go          # Environment configuration loading
├── models/
│   ├── user.go            # User, Role, TOTP, Session structs
│   ├── docker.go          # ContainerInfo, HostConfig, Stats structs
│   └── notification.go    # Notification-related structs
├── services/
│   ├── auth_service.go    # JWT generation, TOTP validation
│   ├── docker_manager.go  # Docker client pool management
│   ├── update_service.go  # Image update checks + container recreation
│   ├── email_service.go   # SMTP2Go email sending
│   └── notify_service.go  # Apprise notifications
├── handlers/
│   ├── auth.go            # POST /api/auth/login, /logout, /refresh, etc.
│   ├── containers.go      # GET/POST /api/containers/:hostId/:id/*
│   ├── hosts.go           # GET/POST/PUT/DELETE /api/hosts/*
│   ├── updates.go         # GET/POST /api/updates/*
│   ├── files.go           # GET/POST /api/files/* (SFTP operations)
│   ├── terminal.go        # WebSocket /api/ws/terminal
│   └── logs.go            # GET /api/debug/logs
├── middleware/
│   └── auth.go            # authMiddleware, adminOnly, CORS
├── store/
│   └── user_store.go      # In-memory user/session storage
└── websocket/
    └── hub.go             # WebSocket hub for container stats
```

---

## Refactoring Plan (Incremental & Safe)

### Phase 1: Extract Models (~30 mins)
**Goal:** Move all type definitions to `models/` package

**Steps:**
1. Create `models/user.go` - User, Role, TOTP, Session, RefreshToken
2. Create `models/docker.go` - HostConfig, ContainerInfo, ContainerStats, ImageUpdate
3. Create `models/notification.go` - AppriseConfig, EmailRequest
4. Update `main.go` imports to use `models.*`

**Validation:**
```bash
go build  # Must compile successfully
```

**Commit:** `refactor(models): extract type definitions to models package`

---

### Phase 2: Extract Services (~1-2 hours)
**Goal:** Move business logic to `services/` package

**Steps:**
1. Create `services/docker_manager.go`:
   - `DockerManager` struct
   - `GetClient()`, `AddHost()`, `RemoveHost()`, `TestConnection()`

2. Create `services/auth_service.go`:
   - `generateAccessToken()`, `generateRefreshToken()`, `verifyToken()`
   - `generateTOTPSecret()`, `verifyTOTP()`, `generateRecoveryCodes()`

3. Create `services/email_service.go`:
   - `sendEmail()` function (SMTP2Go API)

4. Create `services/notify_service.go`:
   - `sendAppriseNotification()` function

5. Update `main.go` to use services

**Validation:**
```bash
go build
./backend  # Run and test auth endpoints
curl -X POST http://localhost:3001/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin"}'
```

**Commit:** `refactor(services): extract business logic to services package`

---

### Phase 3: Extract Handlers (~2-3 hours)
**Goal:** Move HTTP handlers to `handlers/` package

**Steps:**
1. Create `handlers/auth.go`:
   - `/api/auth/login`, `/logout`, `/refresh`, `/me`
   - `/api/auth/password`, `/avatar`
   - `/api/auth/totp/*` endpoints

2. Create `handlers/containers.go`:
   - `/api/hosts/:hostId/containers`
   - `/api/containers/:hostId/:id/start|stop|restart|pause|unpause`
   - `/api/containers/:hostId/:id/stats`
   - `/api/containers/:hostId/:id/logs`

3. Create `handlers/hosts.go`:
   - `/api/hosts` (GET, POST)
   - `/api/hosts/:id` (PUT, DELETE)

4. Create `handlers/files.go`:
   - `/api/files/:hostId/list`
   - `/api/files/:hostId/upload`, `/download`, `/delete`

5. Create `handlers/terminal.go`:
   - `/api/ws/terminal/:hostId/:containerId`

6. Create `handlers/logs.go`:
   - `/api/debug/logs`

7. Update `main.go` routing to use handlers

**Validation:**
```bash
go build
./backend
# Test key endpoints
curl http://localhost:3001/api/hosts
curl http://localhost:3001/api/hosts/raspi1/containers
```

**Commit:** `refactor(handlers): extract HTTP handlers to handlers package`

---

### Phase 4: Extract Config & Middleware (~30 mins)
**Goal:** Move configuration and middleware to dedicated packages

**Steps:**
1. Create `config/config.go`:
   - `LoadConfig()` function
   - `getEnvOrDefault()` helper

2. Create `middleware/auth.go`:
   - `AuthMiddleware()` middleware
   - `AdminOnly()` middleware
   - `CORS()` middleware

3. Create `store/user_store.go`:
   - `UserStore` struct
   - User/session management functions

4. Create `websocket/hub.go`:
   - `Hub` struct
   - WebSocket broadcast logic

**Validation:**
```bash
go build
./backend
# Test protected endpoints
curl http://localhost:3001/api/hosts  # Should return 401
```

**Commit:** `refactor(config): extract configuration, middleware, and store packages`

---

### Phase 5: Implement Update Service (~1 hour)
**Goal:** Add new update service with health check validation

**Steps:**
1. Create `services/update_service.go`:
   - `UpdateContainer(ctx, cli, containerID)` - Implements Enfoque A
   - `validateImageHealth(ctx, cli, containerID)` - Health check logic

2. Create `handlers/updates.go`:
   - `POST /api/containers/:hostId/:containerId/update` - Uses UpdateService

3. Remove old Watchtower HTTP API code from main.go

**Validation:**
```bash
go build
./backend
# Test update endpoint
curl -X POST http://localhost:3001/api/containers/raspi1/<container-id>/update \
  -H "Authorization: Bearer <token>"
```

**Commit:** `feat(updates): implement container update with health check validation`

---

## Final main.go Structure

```go
package main

import (
    "dockerverse/config"
    "dockerverse/handlers"
    "dockerverse/middleware"
    "dockerverse/services"
    "dockerverse/store"
    "dockerverse/websocket"
    "github.com/gofiber/fiber/v2"
)

func main() {
    // Load configuration
    cfg := config.LoadConfig()

    // Initialize services
    dm := services.NewDockerManager(cfg.Hosts)
    authService := services.NewAuthService(cfg.JWTSecret)

    // Initialize stores
    userStore := store.NewUserStore()
    wsHub := websocket.NewHub()

    // Initialize Fiber app
    app := fiber.New()

    // Middleware
    app.Use(middleware.CORS())

    // Public routes
    app.Post("/api/auth/login", handlers.Login(userStore, authService))

    // Protected routes
    protected := app.Group("/api", middleware.AuthMiddleware(userStore, authService))
    protected.Get("/hosts", handlers.GetHosts(dm))
    protected.Post("/containers/:hostId/:id/update", handlers.UpdateContainer(dm))
    // ... more routes

    // Start server
    app.Listen(":3001")
}
```

**Estimated lines:** ~200 (vs current 4,518)

---

## Safety Measures

### After Each Phase:
1. ✅ **Compile check:** `go build` must succeed
2. ✅ **Manual testing:** Test key endpoints
3. ✅ **Git commit:** Incremental commits allow easy rollback
4. ✅ **Tag backup:** If needed, `git tag v2.4.x-backup` before phase

### Rollback Strategy:
```bash
# If something breaks during a phase:
git reset --hard HEAD~1  # Undo last commit
# Or return to pre-refactoring:
git reset --hard v2.4.0-watchtower-api
```

---

## Success Criteria

- [ ] All phases completed
- [ ] `go build` succeeds
- [ ] All API endpoints work as before
- [ ] Auth, containers, hosts, files, terminal functional
- [ ] New update service implemented and tested
- [ ] Code organized into logical modules
- [ ] Each file < 500 lines
- [ ] Documentation updated

---

## Trade-offs

### Benefits
✅ Maintainability (easier to find and modify code)
✅ Testability (isolated modules)
✅ Collaboration (no merge conflicts)
✅ Feature development (add new features without risk)

### Costs
⚠️ Initial effort (~5-7 hours total)
⚠️ Testing required after each phase
⚠️ Import statements in files (minor verbosity)

---

## Notes

- Each phase is independent and can be completed separately
- Phases 1-4 are pure refactoring (no behavior changes)
- Phase 5 adds new functionality (update service)
- All phases tested and committed incrementally
- Rollback possible at any point

---

## References

- Current codebase: `backend/main.go` (4,518 lines)
- Backup version: `v2.4.0-watchtower-api`
- Target version: `v2.5.0` (modular architecture)
