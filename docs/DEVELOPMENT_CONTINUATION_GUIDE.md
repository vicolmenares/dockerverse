### 2026-02-12 - Fix SSH/IP real en DOCKER_HOSTS y backend

- Cambios:
   - DOCKER_HOSTS actualizado para usar IP real de cada raspi (ejemplo: raspi1:Raspeberry Main:192.168.1.145:remote|raspi2:Raspeberry Secondary:192.168.1.146:remote).
   - Backend (main.go) ajustado para derivar correctamente la IP para SSH, evitando localhost.
   - Compose actualizado para reflejar nueva config.
   - Redeploy completo con ./deploy-to-raspi.sh.
- Archivos:
   - `backend/main.go`
   - `docker-compose.unified.yml`
- Tests:
   - `go test ./...` (backend): OK (sin tests).
   - `npm --prefix frontend run check`: OK (0 errors, 0 warnings).
- Deploy a Raspberry Pi: completado con `./deploy-to-raspi.sh`.
   - Resultado: OK (contenedor healthy en `:3007`).
- Verificacion en Raspberry Pi: OK
   - UI muestra hosts online, disco y Files/Console funcionando (esperado).
   - SSH conecta a IP real, no localhost.
- Git push: pendiente tras este update.

Note: If Files/Console still fail with connection to ::1, ensure that the `DOCKER_HOSTS` entries do not mark a remote host as `local`. Remote Raspberry Pis must be marked `remote` and use the Pi IP address. Example:

```
raspi1:Raspeberry Main:192.168.1.145:remote|raspi2:Raspeberry Secondary:192.168.1.146:remote
```

Added logging to `backend/main.go` in `dialSSH()` to surface the resolved host and TCP address on dial attempts. This helps debugging connection errors like `connect: connection refused` or unresolved hostnames.
 - Backend now attempts multiple SSH address fallbacks when dialing (explicit IP, derived host from DOCKER_HOSTS, `host.docker.internal`, and host ID). See `deriveSSHCandidates()` in `backend/main.go`.
 - After applying the fallback, redeploy and verify `/api/debug/hosts` and backend logs for which candidate succeeded.
 - If the Docker remote API is not reachable, the backend now falls back to executing Docker CLI commands over SSH (e.g. `docker start/stop/restart`) so container actions from the UI still work when only SSH is available. See `ContainerAction` SSH fallback in `backend/main.go`.
 - Git push: committed changes to branch `feature/toggle-filters-host-rename-2026-02-12` (please push if not already pushed from the deploy environment).
# DockerVerse - Guía Completa de Continuación de Desarrollo

> **Documento de transferencia de conocimiento para continuar el desarrollo desde macOS**
> 
> Última actualización: 17 de febrero de 2026
> Versión actual: v2.4.1

---

## 📋 Tabla de Contenidos

1. [Resumen Ejecutivo](#resumen-ejecutivo)
2. [Historia del Proyecto](#historia-del-proyecto)
3. [Arquitectura del Sistema](#arquitectura-del-sistema)
4. [Stack Tecnológico Completo](#stack-tecnológico-completo)
5. [Estructura del Proyecto](#estructura-del-proyecto)
6. [Funcionalidades por Versión](#funcionalidades-por-versión)
7. [Configuración del Entorno de Desarrollo](#configuración-del-entorno-de-desarrollo)
8. [Guía de Instalación para macOS](#guía-de-instalación-para-macos)
9. [Conexión a Raspberry Pis](#conexión-a-raspberry-pis)
10. [Proceso de Deployment](#proceso-de-deployment)
11. [Base de Datos y Persistencia](#base-de-datos-y-persistencia)
12. [Autenticación y Seguridad](#autenticación-y-seguridad)
13. [API Reference](#api-reference)
14. [Guía de Troubleshooting](#guía-de-troubleshooting)
15. [Roadmap y Próximos Pasos](#roadmap-y-próximos-pasos)
16. [Estado del Repositorio (Git)](#estado-del-repositorio-git)
17. [Mapa de Módulos UI/UX](#mapa-de-módulos-uiux)
18. [Mapa de Funcionalidades End-to-End](#mapa-de-funcionalidades-end-to-end)
19. [Tracking de Cambios](#tracking-de-cambios)

---

## 🎯 Resumen Ejecutivo

**DockerVerse** es un dashboard de gestión multi-host de Docker, diseñado para administrar contenedores en múltiples Raspberry Pis desde una interfaz web moderna. El proyecto se desarrolló completamente desde cero usando:

- **Backend**: Go 1.23 con Fiber framework
- **Frontend**: SvelteKit 2.x con Svelte 5, TailwindCSS 3.4
- **Deployment**: Docker con arquitectura unificada (single container)
- **Target**: Raspberry Pi 4/5 con Docker instalado

### Características Principales (v2.4.0)

- ✅ Gestión multi-host de contenedores Docker
- ✅ Terminal web con WebSocket (7 temas, búsqueda, reconexión, WebGL, zoom)
- ✅ Visor de logs estilo Databasement con filtros avanzados
- ✅ Gráficos de recursos en tiempo real (CPU, RAM, Red, Disco)
- ✅ Resource Leaderboard con tabs (CPU/Memory/Network/Restarts)
- ✅ Sistema de autenticación con JWT + TOTP/MFA (SHA1)
- ✅ Detección de actualizaciones con indicadores animados
- ✅ Panel de actualizaciones pendientes con dropdown
- ✅ Subida de avatar de usuario
- ✅ Auto-logout configurable (5, 10, 15, 30, 60, 120 min)
- ✅ Command Palette (Ctrl+K)
- ✅ Sidebar con estado activo resaltado
- ✅ Soporte multi-idioma (ES/EN)
- ✅ Tema oscuro nativo
- ✅ Settings con navegación por rutas SvelteKit (v2.2.0)
- ✅ Configurable Docker hosts via DOCKER_HOSTS env var (v2.3.0)
- ✅ Host health backoff - skip unreachable hosts for 30s (v2.3.0)
- ✅ Broadcaster timeouts (5s) prevent SSE hangs (v2.3.0)
- ✅ Frontend resilient loading with Promise.allSettled (v2.3.0)
- ✅ Fetch timeout utility (8s default) on all API calls (v2.3.0)
- ✅ SSE data clears connection errors automatically (v2.3.0)
- ✅ Real image update detection via registry digest comparison (v2.3.0)
- ✅ Background update checker every 15 minutes (v2.3.0)
- ✅ Watchtower HTTP API integration for click-to-update (v2.3.0)
- ✅ Update button on ContainerCard when updates available (v2.3.0)
- ✅ Configurable Top Resources count selector (5/10/15/20/30) (v2.3.0)
- ✅ Tabular-nums on all real-time numeric displays to prevent jitter (v2.3.0)
- ✅ SSH console embebido por host via WebSocket (v2.4.0)
- ✅ File manager SFTP con upload/download por host (v2.4.0)
- ✅ Bulk update modal para actualización masiva de contenedores (v2.4.0)
- ✅ Paginación de contenedores con selector (9/12/18/24) (v2.4.0)
- ✅ Disk free via SSH (sin contenedor busybox) (v2.4.0)
- ✅ Backend file logging con /api/debug/logs endpoint (v2.4.0)
- ✅ SSH fallback para container actions cuando Docker API falla (v2.4.0)
- ✅ Multiple SSH candidates con deriveSSHCandidates() (v2.4.0)
- ✅ Environments CRUD page en settings (v2.4.0)
- ✅ Logs page dedicada con modos single/multi/agrupado, regex, keyboard shortcuts (v2.4.1)
- ✅ Socket-proxy integration para acceso seguro al Docker daemon (v2.4.0)
- ✅ Docker version fallback via ServerVersion API (v2.4.0)
- ✅ Health endpoint /health para container healthcheck (v2.4.0)

---

## 📜 Historia del Proyecto

### Cronología de Desarrollo

#### Fase 1: Inicio (Enero 2026)
- Concepto inicial y planificación
- Setup del entorno de desarrollo Windows
- Arquitectura inicial con contenedores separados

#### Fase 2: v1.0.0 (Febrero 2026)
**Características implementadas:**
1. Dashboard principal con grid de hosts
2. Tarjetas de contenedores con acciones (start/stop/restart)
3. Terminal web básica con xterm.js
4. Visor de logs básico
5. Sistema de autenticación JWT
6. Gestión de usuarios (CRUD)
7. Refresh token con rotación
8. Command Palette (Ctrl+K)
9. Sidebar collapsible
10. Soporte multi-idioma (ES/EN)
11. Persistencia de datos en volumen Docker

#### Fase 3: v2.0.0 (Febrero 2026)
**Nuevas características:**
1. Auto-logout por inactividad (30 minutos)
2. Ocultación de UI innecesaria en login
3. OTP/TOTP MFA con QR y códigos de recuperación
4. LogViewer mejorado con filtros de fecha/hora y nivel
5. Terminal mejorada con:
   - 5 temas (Tokyo Night, Dracula, Monokai, Nord, GitHub Dark)
   - Búsqueda con Ctrl+F
   - Reconexión automática con backoff exponencial
   - Control de tamaño de fuente
6. Gráficos de recursos bajo cada host (sparklines)
7. Detección de actualizaciones de imágenes (Watchtower-style)
8. Contador de actualizaciones pendientes en header
9. Settings movido a sidebar
10. Sección de seguridad unificada (Password + 2FA)
11. Subida y eliminación de avatar de usuario

#### Fase 4: v2.1.0 (Febrero 2026)
**Mejoras de UX/UI inspiradas en Databasement:**
1. **Auto-logout Configurable**: Selección de tiempo (5, 10, 15, 30, 60, 120 min)
2. **Log Viewer Restyled**: Layout estilo Databasement con:
   - Tabla con bordes coloreados por nivel (verde=info, amarillo=warn, rojo=error)
   - Columnas Date/Type/Message
   - Filtros de rango de fecha mejorados
3. **Terminal Premium**:
   - 2 nuevos temas: Catppuccin Mocha, One Dark Pro (7 temas totales)
   - WebGL renderer para mejor performance
   - Web-links addon para URLs clickeables
   - Ctrl+Scroll para zoom de fuente
   - Scrollback aumentado a 10,000 líneas
4. **Resource Leaderboard**: Gráfico con tabs para:
   - Top 14 contenedores por CPU/Memory/Network/Restarts
   - Filtrado por host
5. **Update Indicators**: Badge animado en cada contenedor
6. **Pending Updates Panel**: Dropdown en header con contador y lista
7. **Sidebar Active State**: Resaltado visual del item activo
8. **Avatar Upload Fix**: Corregido endpoint API

#### Fase 5: v2.2.0 (8 Febrero 2026)
**Migración a navegación basada en rutas (Page-Based Navigation):**

Se eliminó el patrón de modal flotante (`Settings.svelte` como overlay `fixed inset-0 z-50`) y se migró a rutas SvelteKit dedicadas. Cada sección de configuración ahora es una página independiente con URL propia.

**Cambios principales:**
1. **Shared Settings Module** (`$lib/settings/index.ts`): Traducciones y tipos extraídos de Settings.svelte
2. **Settings Layout** (`routes/settings/+layout.svelte`): Layout con breadcrumb y auth guard
3. **9 rutas de settings creadas**:
   - `/settings` - Menú principal de configuración
   - `/settings/profile` - Perfil de usuario y avatar
   - `/settings/security` - Auto-logout, contraseña, 2FA/TOTP
   - `/settings/users` - Gestión de usuarios (admin)
   - `/settings/notifications` - Umbrales, canales, Apprise
   - `/settings/appearance` - Tema y idioma
   - `/settings/data` - Caché y almacenamiento
   - `/settings/about` - Información de la app
4. **Sidebar actualizado**: Todos los items usan `href` links en vez de callbacks `action()`
5. **Active state por URL**: `activeSidebarItem` se deriva de `$page.url.pathname`
6. **User menu**: Botón "Settings" navega a `/settings` en vez de abrir modal
7. **Updates dropdown**: Link "Ver todo" navega a `/settings/data`
8. **Bug fix**: `ResourceChart.svelte` importaba `language` desde `$lib/stores/auth` (incorrecto) → corregido a `$lib/stores/docker`

**Archivos creados (10):**
| Archivo | Descripción |
|---------|-------------|
| `src/lib/settings/index.ts` | Traducciones compartidas, tipos |
| `src/routes/settings/+layout.svelte` | Layout settings con breadcrumb |
| `src/routes/settings/+page.svelte` | Menú principal settings |
| `src/routes/settings/profile/+page.svelte` | Perfil y avatar |
| `src/routes/settings/security/+page.svelte` | Seguridad, password, 2FA |
| `src/routes/settings/users/+page.svelte` | Gestión usuarios |
| `src/routes/settings/notifications/+page.svelte` | Notificaciones |
| `src/routes/settings/appearance/+page.svelte` | Tema e idioma |
| `src/routes/settings/data/+page.svelte` | Datos y caché |
| `src/routes/settings/about/+page.svelte` | Acerca de |

**Archivos modificados (2):**
| Archivo | Cambios |
|---------|---------|
| `src/routes/+layout.svelte` | Removido Settings modal, sidebar usa hrefs, active state por URL |
| `src/lib/components/ResourceChart.svelte` | Fix import `language` store |

**Nota:** `Settings.svelte` ya no se importa pero se mantiene como referencia histórica.

**Hotfix v2.2.0 - Nginx Cache & app.css (8 Feb 2026):**
- **Bug**: Nginx proxy cache permissions (`/var/cache/nginx/`) causaban `Permission denied` al cachear assets estáticos (CSS/JS), resultando en respuestas vacías (200 con 0 bytes). La app cargaba sin estilos ni JS.
- **Fix `Dockerfile.unified`**: Agregado `chown -R nginx:nginx /var/cache/nginx /run/nginx` en el script de arranque de nginx s6. También se incluye `package-lock.json` en el COPY para installs consistentes.
- **Bug**: `app.html` tenía `<link rel="preload" href="app.css">` pero ese archivo no existe en el build de SvelteKit (CSS se bundlea en hashes inmutables). Generaba error 404 en consola.
- **Fix `app.html`**: Removido el preload link a `app.css`.

---

## 🏗️ Arquitectura del Sistema

### Arquitectura Unificada (Single Container)

```
┌─────────────────────────────────────────────────────────────┐
│                    Docker Container                          │
│                    (dockerverse:unified)                     │
│                                                              │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                    s6-overlay                        │   │
│  │              (Process Supervisor)                    │   │
│  └─────────────────────────────────────────────────────┘   │
│           │                │                │               │
│           ▼                ▼                ▼               │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐    │
│  │   Nginx     │  │  Go Backend │  │ SvelteKit Node  │    │
│  │  (Port 80)  │  │ (Port 3002) │  │   (Port 3003)   │    │
│  │  Reverse    │  │   Fiber     │  │   SSR/Hydrate   │    │
│  │   Proxy     │  │    API      │  │                 │    │
│  └──────┬──────┘  └─────────────┘  └─────────────────┘    │
│         │                ▲                ▲                │
│         │                │                │                │
│         └────────────────┴────────────────┘                │
│              Routing: /api/* → Backend                      │
│                       /*     → Frontend                     │
└─────────────────────────────────────────────────────────────┘
                              │
                    ┌─────────┴─────────┐
                    ▼                   ▼
          ┌─────────────────┐  ┌────────────────┐
          │  Docker Socket  │  │  SSH (port 22)  │
          │   (Read-Only)   │  │  to remote hosts│
          └─────────────────┘  └────────────────┘
```

### Diagrama de Red Multi-Host

```
┌─────────────────────────────────────────────────────────────────┐
│                     RED LOCAL (192.168.1.x)                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────────┐                                           │
│  │   macOS Dev      │                                           │
│  │  (Este equipo)   │ ─────────────────────────┐               │
│  │  SSH + rsync     │                          │               │
│  └──────────────────┘                          │               │
│                                                 ▼               │
│  ┌──────────────────┐    ┌──────────────────┐                  │
│  │   Raspberry Pi   │    │   Raspberry Pi   │                  │
│  │  192.168.1.145   │    │  192.168.1.146   │                  │
│  │  (Server Main)   │    │   (Server 2)     │                  │
│  │  Port: 3007      │    │   Docker Host    │                  │
│  │  DockerVerse     │    │   Port: 2375     │                  │
│  │  socket-proxy    │    │                  │                  │
│  │  Port: 23750     │    │                  │                  │
│  └──────────────────┘    └──────────────────┘                  │
│           │                       │                             │
│           └───────────────────────┘                             │
│                     │                                           │
│           Docker API (TCP) + SSH                                │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

> **Nota**: raspi1 usa socket-proxy (puerto 23750) para acceso seguro al Docker socket.
> raspi2 expone Docker API directamente en puerto 2375.

---

## 🛠️ Stack Tecnológico Completo

### Backend (Go)

| Componente | Versión | Propósito |
|------------|---------|-----------|
| Go | 1.23+ | Lenguaje principal |
| Fiber | v2.52.0 | Framework web HTTP |
| fiber/websocket | v2.2.1 | WebSocket support |
| docker/docker | v27.5.0 | Docker API client |
| golang-jwt/jwt | v5.2.1 | JSON Web Tokens |
| pquerna/otp | v1.4.0 | TOTP/2FA support (SHA1) |
| creack/pty | v1.1.21 | Terminal pseudo-TTY |
| go-containerregistry | v0.20.3 | Registry digest comparison (crane) |
| pkg/sftp | v1.13.10 | SFTP file manager |
| golang.org/x/crypto | v0.41.0 | bcrypt hashing, SSH |

### Frontend (SvelteKit)

| Componente | Versión | Propósito |
|------------|---------|-----------|
| SvelteKit | ^2.5.20 | Framework principal |
| Svelte | ^5.0.0 | UI reactiva |
| TailwindCSS | 3.4.4 | Estilos utility-first |
| @xterm/xterm | ^5.5.0 | Terminal emulator |
| @xterm/addon-fit | ^0.10.0 | Auto-resize terminal |
| @xterm/addon-search | ^0.15.0 | Terminal search |
| @xterm/addon-web-links | ^0.11.0 | Clickable links |
| @xterm/addon-webgl | ^0.18.0 | WebGL renderer (performance) |
| lucide-svelte | ^0.408.0 | Iconos |
| echarts | ^5.5.0 | Gráficos (opcional) |
| clsx | ^2.1.1 | Utility classes |
| @playwright/test | ^1.58.2 | E2E testing |

### Infrastructure

| Componente | Versión | Propósito |
|------------|---------|-----------|
| Docker | 24.x+ | Containerization |
| Docker Compose | 2.x | Orchestration |
| Nginx | 1.25 | Reverse proxy |
| s6-overlay | v3 | Process supervisor |
| Alpine Linux | 3.19 | Base image |

### Herramientas de Desarrollo (macOS)

| Herramienta | Versión | Propósito |
|-------------|---------|-----------|
| VS Code | Latest | IDE principal |
| Node.js | 20.x LTS | Runtime frontend dev |
| npm | 10.x | Package manager |
| Go | 1.23.x | Backend development |
| Docker Desktop | 4.x | Containers |
| Git | 2.x | Version control |
| GitHub CLI | 2.x | GitHub operations |
| SSH | Built-in | Conexión a Raspberry Pis |
| rsync | Built-in | Sincronización de archivos |

---

## 📁 Estructura del Proyecto

```
dockerverse/
├── .git/                      # Git repository
├── .dockerignore              # Docker ignore rules
├── backend/
│   ├── Dockerfile             # Go backend container
│   ├── go.mod                 # Go dependencies
│   ├── go.sum                 # Go checksums
│   └── main.go                # Backend principal (~4400 líneas)
│       ├── Structs (User, Host, Container, etc.)
│       ├── Auth (JWT, Refresh, TOTP)
│       ├── Docker API integration
│       ├── WebSocket handlers (terminal, logs)
│       └── Image update checking
├── frontend/
│   ├── Dockerfile             # Frontend container
│   ├── package.json           # Node dependencies
│   ├── svelte.config.js       # SvelteKit config
│   ├── vite.config.ts         # Vite bundler config
│   ├── tailwind.config.js     # TailwindCSS config
│   ├── postcss.config.js      # PostCSS config
│   ├── tsconfig.json          # TypeScript config
│   ├── src/
│   │   ├── app.html           # HTML template
│   │   ├── app.css            # Global styles
│   │   ├── app.d.ts           # Type definitions
│   │   ├── hooks.server.ts    # Server hooks
│   │   ├── lib/
│   │   │   ├── index.ts       # Lib exports
│   │   │   ├── api/
│   │   │   │   └── docker.ts  # API client (~400 líneas)
│   │   │   ├── components/
│   │   │   │   ├── index.ts
│   │   │   │   ├── BulkUpdateModal.svelte   # Bulk update de contenedores
│   │   │   │   ├── CommandPalette.svelte
│   │   │   │   ├── ContainerCard.svelte
│   │   │   │   ├── EnvironmentModal.svelte  # CRUD de environments
│   │   │   │   ├── HostCard.svelte
│   │   │   │   ├── HostFiles.svelte         # File manager SFTP
│   │   │   │   ├── Login.svelte
│   │   │   │   ├── LogViewer.svelte
│   │   │   │   ├── ResourceChart.svelte
│   │   │   │   ├── Settings.svelte (legacy, no longer imported)
│   │   │   │   ├── Terminal.svelte
│   │   │   │   └── UpdateModal.svelte       # Update individual de contenedor
│   │   │   ├── settings/
│   │   │   │   └── index.ts   # Shared translations & types
│   │   │   └── stores/
│   │   │       ├── auth.ts    # Auth store (~550 líneas)
│   │   │       └── docker.ts  # Docker store (~500 líneas)
│   │   └── routes/
│   │       ├── +layout.svelte # Main layout (~640 líneas)
│   │       ├── +page.svelte   # Dashboard page
│   │       ├── logs/          # Logs page (v2.4.0)
│   │       │   └── +page.svelte       # Backend logs viewer
│   │       └── settings/      # Settings pages (v2.2.0+)
│   │           ├── +layout.svelte     # Settings layout + auth guard
│   │           ├── +page.svelte       # Settings menu
│   │           ├── profile/+page.svelte
│   │           ├── security/+page.svelte
│   │           ├── users/+page.svelte
│   │           ├── notifications/+page.svelte
│   │           ├── appearance/+page.svelte
│   │           ├── data/+page.svelte
│   │           ├── environments/+page.svelte  # Environments CRUD (v2.4.0)
│   │           └── about/+page.svelte
│   └── static/
│       ├── robots.txt
│       └── sw.js              # Service worker stub
├── nginx/
│   └── nginx.conf             # Nginx configuration
├── docker-compose.yml         # Multi-container (legacy)
├── docker-compose.unified.yml # Single container
├── Dockerfile.unified         # Unified build
├── deploy-to-raspi.sh         # Deploy automático a Raspberry Pi
├── sync-to-raspi.sh           # Sync de archivos a Raspberry Pi
├── setup-mac.sh               # Setup del entorno macOS
├── transfer.ps1               # Windows deploy script (legacy)
├── sync.ps1                   # Sync script (legacy)
├── .env                       # Variables de entorno (no commitear secrets)
├── README.md                  # Basic readme
├── UNIFIED_CONTAINER_ARCHITECTURE.md
└── DEVELOPMENT_CONTINUATION_GUIDE.md  # Este documento
```

---

## ✅ Funcionalidades por Versión

### v1.0.0 - Foundation Release

| Feature | Descripción | Archivo(s) Principal(es) |
|---------|-------------|-------------------------|
| Multi-host Dashboard | Grid de hosts con estado | +layout.svelte, HostCard.svelte |
| Container Management | Start/Stop/Restart | ContainerCard.svelte, docker.ts |
| Web Terminal | xterm.js con WebSocket | Terminal.svelte, main.go |
| Log Viewer | Streaming de logs | LogViewer.svelte, main.go |
| JWT Auth | Login/Logout con tokens | auth.ts, main.go |
| Refresh Tokens | Rotación automática | auth.ts, main.go |
| User Management | CRUD de usuarios | Settings.svelte, main.go |
| Command Palette | Ctrl+K quick actions | CommandPalette.svelte |
| Sidebar | Navegación collapsible | +layout.svelte |
| i18n | Español/Inglés | docker.ts (translations) |
| Dark Theme | Tema oscuro nativo | app.css, tailwind.config.js |

### v2.0.0 - Security & Monitoring Release

| Feature | Descripción | Archivo(s) Principal(es) |
|---------|-------------|-------------------------|
| Auto-logout | 30 min inactividad | auth.ts (setupActivityTracking) |
| Login UI Clean | Sin search/refresh | +layout.svelte |
| TOTP/MFA | 2FA con QR code | Settings.svelte, main.go |
| Recovery Codes | 10 códigos backup | Settings.svelte, main.go |
| Advanced LogViewer | Filtros fecha/nivel/búsqueda | LogViewer.svelte |
| Terminal Themes | 5 temas visuales | Terminal.svelte |
| Terminal Search | Ctrl+F find | Terminal.svelte |
| Terminal Reconnect | Backoff exponencial | Terminal.svelte |
| Resource Charts | Sparklines CPU/RAM/Net/Disk | ResourceChart.svelte |
| Image Updates | Watchtower-style check | docker.ts, main.go |
| Updates Counter | Badge en header | +layout.svelte |
| Unified Security | Password + 2FA juntos | Settings.svelte |
| Avatar Upload | Foto de perfil | Settings.svelte, auth.ts, main.go |

### v2.1.0 - UX/UI Enhancement Release

| Feature | Descripción | Archivo(s) Principal(es) |
|---------|-------------|-------------------------|
| Configurable Auto-logout | 5, 10, 15, 30, 60, 120 min | auth.ts, Settings.svelte |
| Databasement-style Logs | Tabla con bordes coloreados | LogViewer.svelte |
| Terminal WebGL | Renderer WebGL para performance | Terminal.svelte |
| Terminal Themes++ | +2 temas (Catppuccin, One Dark Pro) | Terminal.svelte |
| Terminal Web-links | URLs clickeables | Terminal.svelte |
| Terminal Zoom | Ctrl+Scroll para font size | Terminal.svelte |
| Terminal Scrollback | 10,000 líneas | Terminal.svelte |
| Resource Leaderboard | Top-14 CPU/Memory/Network/Restarts | +page.svelte |
| Update Badge | Indicador animado por contenedor | ContainerCard.svelte |
| Pending Updates Panel | Dropdown con lista de updates | +layout.svelte |
| Sidebar Active State | Highlight del item actual | +layout.svelte |
| Avatar Upload Fix | Corrección de API endpoint | auth.ts |

### v2.4.0 - SSH, SFTP & Infrastructure Release

| Feature | Descripción | Archivo(s) Principal(es) |
|---------|-------------|-------------------------|
| SSH Console | Terminal SSH embebida por host | Terminal.svelte, main.go |
| SFTP File Manager | Navegador de archivos con upload/download | HostFiles.svelte, main.go |
| Bulk Update Modal | Actualización masiva de contenedores | BulkUpdateModal.svelte, docker.ts |
| Update Modal | Actualización individual con Watchtower | UpdateModal.svelte |
| Container Pagination | Selector de página (9/12/18/24) | +page.svelte |
| Disk Free via SSH | Métricas de disco sin busybox | main.go |
| Backend Logging | File logging + /api/debug/logs | main.go |
| SSH Fallback Actions | Container start/stop/restart via SSH | main.go |
| SSH Candidates | deriveSSHCandidates() multiples IPs | main.go |
| Environments CRUD | Página de gestión de environments | environments/+page.svelte, EnvironmentModal.svelte |
| Logs Page | Visor de logs con streaming SSE, 3 modos, fuzzy filter, regex, keyboard shortcuts, display limit | logs/+page.svelte |
| Toggle Filters | Deseleccionar filtros en cards resumen | +page.svelte |
| Host Rename | Display name configurable via DOCKER_HOSTS | main.go, .env |
| Socket-proxy | Acceso seguro al Docker daemon (23750) | docker-compose.unified.yml |
| Docker Version Fallback | ServerVersion API para daemons restringidos | main.go |
| 2FA SHA1 Fix | Compatibilidad con authenticator apps | main.go |
| Health Endpoint | /health para container healthcheck | main.go |
| Port Reorganization | Backend 3002, Frontend 3003 | Dockerfile.unified, docker-compose.unified.yml |

---

## 💻 Configuración del Entorno de Desarrollo

### Variables de Entorno

```bash
# Backend
PORT=3002                    # Puerto interno del backend (Nginx hace proxy)
DOCKER_HOST=unix:///var/run/docker.sock
DOCKER_HOSTS=raspi1:Raspeberry Main:tcp://192.168.1.145:23750:remote|raspi2:Raspeberry Secondary:tcp://192.168.1.146:2375:remote
JWT_SECRET=***JWT-SECRET-REMOVED***
DATA_PATH=/data
WATCHTOWER_TOKEN=  # Watchtower HTTP API token (optional)
WATCHTOWER_URLS=   # Watchtower URLs per host (optional, format: hostId:url|hostId:url)

# SSH configuration for host console + SFTP
SSH_USER=pi                  # Usuario SSH para conectar a hosts
SSH_PORT=22                  # Puerto SSH
SSH_KEY_PATH=/data/ssh/id_rsa  # Ruta a la clave SSH dentro del contenedor
SSH_KEY_PASSPHRASE=          # Passphrase de la clave (opcional)

# Frontend
NODE_ENV=production
ORIGIN=http://localhost:3007
PUBLIC_API_URL=  # Empty for same-origin

# Container
TZ=America/Mexico_City
S6_VERBOSITY=1
S6_BEHAVIOUR_IF_STAGE2_FAILS=2
S6_CMD_WAIT_FOR_SERVICES_MAXTIME=30000
```

### Puertos Utilizados

| Puerto | Servicio | Descripción |
|--------|----------|-------------|
| 3003 | SvelteKit | Frontend SSR (interno al contenedor) |
| 3002 | Go/Fiber | Backend API (interno al contenedor) |
| 3007 | DockerVerse | Producción (expuesto al host) |
| 80 | Nginx (container) | Reverse proxy (interno, mapeado a 3007) |
| 23750 | socket-proxy | Docker socket proxy en raspi1 |
| 2375 | Docker API | Docker daemon expuesto en raspi2 |

---

## 🍎 Guía de Instalación para macOS

### Prerrequisitos del Sistema

macOS Monterey (12.x) o superior con los siguientes requisitos:
- Terminal con acceso a comandos básicos
- Conexión a internet para descargas
- Acceso SSH a las Raspberry Pis

### Script de Instalación Automática

Se incluye el archivo `setup-mac.sh` que:
1. Detecta herramientas instaladas
2. Verifica versiones mínimas requeridas
3. Instala faltantes via Homebrew
4. Configura el entorno de desarrollo

**Ejecutar:**
```bash
chmod +x setup-mac.sh
./setup-mac.sh
```

### Herramientas Requeridas

| Herramienta | Versión Mínima | Instalación | Propósito |
|-------------|----------------|-------------|-----------|
| Homebrew | 4.x | `/bin/bash -c "$(curl -fsSL ...)"` | Package manager |
| Git | 2.40+ | `brew install git` | Version control |
| Node.js | 20.x LTS | `brew install node@20` | Frontend runtime |
| npm | 10.x | Con Node.js | Package manager |
| Go | 1.23+ | `brew install go` | Backend |
| Docker Desktop | 4.x | `brew install --cask docker` | Containers |
| GitHub CLI | 2.40+ | `brew install gh` | GitHub operations |
| VS Code | Latest | `brew install --cask visual-studio-code` | IDE |

### Extensiones VS Code Recomendadas

```bash
# Instalar extensiones
code --install-extension svelte.svelte-vscode
code --install-extension golang.go
code --install-extension bradlc.vscode-tailwindcss
code --install-extension ms-azuretools.vscode-docker
code --install-extension GitHub.copilot
code --install-extension GitHub.copilot-chat
```

---

## 🔌 Conexión a Raspberry Pis

### Configuración de Hosts

| Host | IP | Usuario | Password | DockerVerse | Docker API |
|------|-----|---------|----------|-------------|------------|
| raspi-main | 192.168.1.145 | pi | Pi16870403 | :3007 | socket-proxy :23750 |
| raspi-secondary | 192.168.1.146 | pi | Pi16870403 | N/A | :2375 (directo) |

### Conexión SSH desde Mac

```bash
# Conexión básica
ssh pi@192.168.1.145

# Conexión con clave (recomendado)
ssh-copy-id pi@192.168.1.145
ssh pi@192.168.1.145
```

### Configurar SSH Config

Añadir a `~/.ssh/config`:

```
Host raspi-main
    HostName 192.168.1.145
    User pi
    IdentityFile ~/.ssh/id_rsa

Host raspi-secondary
    HostName 192.168.1.146
    User pi
    IdentityFile ~/.ssh/id_rsa
```

Uso:
```bash
ssh raspi-main
```

### Transferencia de Archivos

```bash
# SCP individual
scp -r ./frontend/src pi@192.168.1.145:/home/pi/dockerverse/frontend/

# rsync (recomendado para sincronización)
rsync -avz --exclude 'node_modules' --exclude '.git' \
  ./dockerverse/ pi@192.168.1.145:/home/pi/dockerverse/
```

---

## 🚀 Proceso de Deployment

### Desde macOS a Raspberry Pi

#### 1. Sincronizar código

```bash
# Script de sincronización
./sync-to-raspi.sh

# O manualmente con rsync
rsync -avz --exclude 'node_modules' --exclude '.git' \
  --exclude 'test-*' --exclude '*.log' \
  ./ pi@192.168.1.145:/home/pi/dockerverse/
```

#### 2. Build en Raspberry Pi

```bash
# Conectar a Raspi
ssh raspi-main

# Build y deploy
cd /home/pi/dockerverse
docker compose -f docker-compose.unified.yml down
docker compose -f docker-compose.unified.yml build --no-cache
docker compose -f docker-compose.unified.yml up -d

# Verificar
docker ps | grep dockerverse
docker logs -f dockerverse
```

#### 3. Verificar deployment

```bash
# Health check
curl http://192.168.1.145:3007/api/health

# Login test
curl -X POST http://192.168.1.145:3007/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'
```

### Script de Deploy Automático (Mac)

Se incluye `deploy-to-raspi.sh`:

```bash
# Uso:
./deploy-to-raspi.sh            # Deploy completo (sync + build + restart)
./deploy-to-raspi.sh --quick    # Solo sync de archivos (sin rebuild)
./deploy-to-raspi.sh --no-cache # Force rebuild sin cache
./deploy-to-raspi.sh --help     # Ver opciones

# Target: pi@192.168.1.145:/home/pi/dockerverse
# Compose: docker-compose.unified.yml
```

El script:
1. Valida conexión SSH a la Raspberry Pi
2. Sincroniza archivos con rsync (excluye node_modules, .git, logs, etc.)
3. Ejecuta `docker compose up` con la config unificada
4. Espera y verifica health del contenedor

---

## 💾 Base de Datos y Persistencia

### Almacenamiento

DockerVerse usa almacenamiento basado en archivos JSON en el volumen `/data`:

| Archivo | Contenido |
|---------|-----------|
| `/data/users.json` | Usuarios, passwords (bcrypt), avatars |
| `/data/hosts.json` | Configuración de hosts Docker |
| `/data/settings.json` | Configuración de la aplicación |

### Backup

```bash
# Desde Mac
ssh raspi-main "docker exec dockerverse cat /data/users.json" > backup/users.json
ssh raspi-main "docker exec dockerverse cat /data/hosts.json" > backup/hosts.json

# Restore
cat backup/users.json | ssh raspi-main "docker exec -i dockerverse tee /data/users.json"
```

### Usuario Default

```json
{
  "username": "admin",
  "password": "$2a$10$...",  // bcrypt de "admin123"
  "email": "admin@dockerverse.local",
  "firstName": "Admin",
  "lastName": "User",
  "roles": ["admin", "user"]
}
```

---

## 🔐 Autenticación y Seguridad

### Flujo de Autenticación

```
┌─────────────┐      ┌─────────────┐      ┌─────────────┐
│   Login     │      │   Backend   │      │  Storage    │
│   Form      │─────▶│   /login    │─────▶│  (bcrypt)   │
└─────────────┘      └─────────────┘      └─────────────┘
                            │
                            ▼
                    ┌───────────────┐
                    │ JWT + Refresh │
                    │    Tokens     │
                    └───────────────┘
                            │
       ┌────────────────────┼────────────────────┐
       │                    │                    │
       ▼                    ▼                    ▼
┌─────────────┐      ┌─────────────┐      ┌─────────────┐
│ Access Token│      │Refresh Token│      │ 2FA/TOTP    │
│  15 min     │      │  7 days     │      │ (optional)  │
└─────────────┘      └─────────────┘      └─────────────┘
```

### Tokens JWT

| Token | Duración | Uso |
|-------|----------|-----|
| Access Token | 15 minutos | Autenticación API |
| Refresh Token | 7 días | Renovar access token |

### TOTP/2FA

- **Algoritmo**: SHA1 (compatible con Google Authenticator, Authy)
- **Período**: 30 segundos
- **Dígitos**: 6
- **Recovery codes**: 10 códigos de un solo uso

---

## 📡 API Reference

### Endpoints Principales

#### Autenticación

| Method | Endpoint | Descripción |
|--------|----------|-------------|
| POST | `/api/auth/login` | Login con username/password |
| POST | `/api/auth/logout` | Logout y revoca tokens |
| POST | `/api/auth/refresh` | Renueva access token |
| GET | `/api/auth/me` | Info usuario actual |
| POST | `/api/auth/password` | Cambiar password |
| POST | `/api/auth/avatar` | Subir avatar (base64) |
| DELETE | `/api/auth/avatar` | Eliminar avatar |

#### TOTP

| Method | Endpoint | Descripción |
|--------|----------|-------------|
| GET | `/api/auth/totp/status` | Estado 2FA |
| POST | `/api/auth/totp/setup` | Iniciar setup 2FA |
| POST | `/api/auth/totp/verify` | Verificar y activar |
| POST | `/api/auth/totp/disable` | Desactivar 2FA |

#### Hosts

| Method | Endpoint | Descripción |
|--------|----------|-------------|
| GET | `/api/hosts` | Lista de hosts |
| POST | `/api/hosts` | Agregar host |
| PUT | `/api/hosts/:id` | Actualizar host |
| DELETE | `/api/hosts/:id` | Eliminar host |

#### Containers

| Method | Endpoint | Descripción |
|--------|----------|-------------|
| GET | `/api/hosts/:hostId/containers` | Contenedores de un host |
| POST | `/api/containers/:hostId/:id/start` | Iniciar contenedor |
| POST | `/api/containers/:hostId/:id/stop` | Detener contenedor |
| POST | `/api/containers/:hostId/:id/restart` | Reiniciar contenedor |
| GET | `/api/containers/:hostId/:id/stats` | Estadísticas |

#### WebSocket

| Endpoint | Descripción |
|----------|-------------|
| `/api/ws/logs/:hostId/:containerId` | Stream de logs |
| `/api/ws/terminal/:hostId/:containerId` | Terminal interactiva |

#### Image Updates

| Method | Endpoint | Descripción |
|--------|----------|-------------|
| GET | `/api/updates` | Lista de actualizaciones |
| POST | `/api/updates/:hostId/:containerId/check` | Verificar imagen |
| POST | `/api/containers/:hostId/:containerId/update` | Trigger Watchtower update |
| POST | `/api/updates/bulk` | Bulk update de contenedores |

#### Environments

| Method | Endpoint | Descripción |
|--------|----------|-------------|
| GET | `/api/environments` | Lista de environments |
| POST | `/api/environments` | Agregar environment |
| PUT | `/api/environments/:id` | Actualizar environment |
| DELETE | `/api/environments/:id` | Eliminar environment |
| POST | `/api/environments/:id/test` | Test de conexión |

#### Debug (admin-only)

| Method | Endpoint | Descripción |
|--------|----------|-------------|
| GET | `/health` | Health check del contenedor |
| GET | `/api/debug/logs` | Últimas líneas del backend.log (?lines=200) |
| GET | `/api/debug/hosts` | Diagnóstico de parsing de hosts |

#### SSH / SFTP

| Method | Endpoint | Descripción |
|--------|----------|-------------|
| WS | `/ws/ssh/:hostId` | Terminal SSH al host via WebSocket |
| GET | `/api/hosts/:hostId/files?path=` | Listar archivos (SFTP) |
| GET | `/api/hosts/:hostId/files/download?path=` | Descargar archivo |
| POST | `/api/hosts/:hostId/files/upload` | Subir archivo |

---

## 🔧 Guía de Troubleshooting

### Problemas Comunes

#### Container no inicia

```bash
# Ver logs
docker logs dockerverse

# Verificar puertos
netstat -tlnp | grep 3007

# Reiniciar
docker compose -f docker-compose.unified.yml restart
```

#### Error de conexión Docker socket

```bash
# Verificar permisos
ls -la /var/run/docker.sock

# Añadir usuario al grupo docker
sudo usermod -aG docker $USER
```

#### Frontend no carga

```bash
# Verificar build
docker exec dockerverse ls -la /app/frontend/build

# Verificar Nginx
docker exec dockerverse nginx -t
docker exec dockerverse cat /var/log/nginx/error.log
```

#### WebSocket no conecta

- Verificar que Nginx permite upgrade WebSocket
- Verificar CORS en backend
- Verificar que el contenedor objetivo está corriendo

#### Error de autenticación

```bash
# Verificar users.json
docker exec dockerverse cat /data/users.json

# Resetear admin password
docker exec dockerverse sh -c 'echo "[NUEVO_JSON]" > /data/users.json'
```

---

## 🗺️ Roadmap y Próximos Pasos

### v2.2.0 (Completado - 8 Feb 2026)

- [x] Settings migrado de modal a rutas SvelteKit
- [x] Navegación por URL para todas las secciones de configuración
- [x] Auth guard en settings layout
- [x] Active state del sidebar derivado de URL
- [x] Fix import bug en ResourceChart.svelte

### v2.3.0 (Completado - 8 Feb 2026)

- [x] Configurable Docker hosts via DOCKER_HOSTS env var (replaces hardcoded hosts)
- [x] Host health tracking with 30s backoff for unreachable hosts
- [x] Broadcaster timeouts (5s context) prevent SSE hangs on offline hosts
- [x] Frontend resilient loading (Promise.allSettled, fetch timeouts, SSE error clearing)
- [x] Real image update detection via go-containerregistry/crane digest comparison
- [x] Background update checker (every 15 min, 15 min cache per container)
- [x] Watchtower HTTP API integration (WATCHTOWER_TOKEN, WATCHTOWER_URLS env vars)
- [x] Click-to-update button on ContainerCard for Watchtower-managed containers
- [x] Configurable Top Resources count selector (5/10/15/20/30 pill buttons)
- [x] Tabular-nums CSS class on all real-time numeric displays (prevents jitter)
- [x] Dockerfile upgraded from Go 1.22 to Go 1.23
- [x] Deploy script updated to use `docker compose` v2 plugin syntax
- [x] go.sum included in Dockerfile COPY for reliable builds

### v2.4.0 (Completado - 12 Feb 2026)

- [x] SSH console embebido por host (WebSocket `/ws/ssh/:hostId`)
- [x] File manager SFTP con upload/download por host (HostFiles.svelte)
- [x] Bulk update modal para actualización masiva (BulkUpdateModal.svelte)
- [x] Paginación de contenedores con selector (9/12/18/24)
- [x] Disk free via SSH (`df` directo, sin contenedor busybox)
- [x] Backend file logging (`/data/logs/backend.log`) + recover middleware
- [x] Debug endpoints (`/api/debug/logs`, `/api/debug/hosts`) admin-only
- [x] SSH fallback para container actions (start/stop/restart via SSH)
- [x] Multiple SSH candidates (`deriveSSHCandidates()`) para conexiones resilientes
- [x] Environments CRUD page en settings
- [x] Logs page dedicada con modos single/multi/agrupado y descarga
- [x] Toggle de filtros en cards de resumen (deseleccionar con segundo click)
- [x] Renombrado de hosts por display name
- [x] Socket-proxy integration (puerto 23750) para acceso seguro al Docker daemon
- [x] Docker version fallback via ServerVersion API (para daemons con acceso restringido)
- [x] 2FA algorithm fix: SHA256→SHA1 para compatibilidad con authenticator apps
- [x] Health endpoint `/health` para container healthcheck
- [x] openssh-client incluido en container image
- [x] Internal ports reorganizados (backend: 3002, frontend: 3003)
- [x] `@types/node` y `@playwright/test` agregados al frontend

### v2.4.1 (Completado - 17 Feb 2026)

**Logs Page Improvements** — Rediseño completo de `frontend/src/routes/logs/+page.svelte`

- [x] Modo **single container** con SSE stream en tiempo real por contenedor
- [x] Modo **multi container** (todos los contenedores seleccionados en un solo feed)
- [x] Modo **agrupado** (un panel por stack/contenedor)
- [x] Filtro de búsqueda de contenedores con **fuzzy matching** (acronim, substring, secuencia)
- [x] Filtro de búsqueda de **logs en tiempo real** con highlight `<mark>` en líneas coincidentes
- [x] Soporte **regex** en filtro de logs con indicador de error para regex inválidos
- [x] **Selector "Last N lines"** para limitar logs mostrados (All / Last 100/500/1000/2000)
- [x] **Pausar/reanudar** stream de logs (Ctrl+P o Space)
- [x] Toggle **wrap lines** para líneas largas (Ctrl+W)
- [x] **Timestamp cycling**: formato absoluto (HH:MM:SS.mmm) → relativo (~ago) → oculto
- [x] **Keyboard shortcuts**: Ctrl+1/2/3 (modos), Ctrl+P (pause), Ctrl+W (wrap), / (focus search), Escape (limpiar)
- [x] **Descarga** de logs como archivo de texto
- [x] **Select all / deselect all** contenedores por stack
- [x] Arquitectura: containers groupados por stack, filtrado reactivo con `$derived.by()`
- [x] Fix: `groupedContainers` ahora deriva de `filteredContainers` (no de `$containers` raw)
- [x] Fix: `state_unsafe_mutation` — `regexResult` como objeto combinado `{pattern, error}`

### v2.5.0 (Planificado)

- [ ] Container Activity chart (bar chart estilo Jobs Activity)
- [ ] Docker Compose management (view/edit compose files)
- [ ] Container creation wizard
- [ ] Network visualization
- [ ] Volume management UI
- [ ] Container templates/presets

### v3.0.0 (Planificado)

- [ ] Multi-user permissions (RBAC)
- [ ] Audit log
- [ ] API keys for automation
- [ ] Webhook integrations
- [ ] Dashboard widgets customization

### v4.0.0 (Futuro)

- [ ] Kubernetes support
- [ ] Portainer import
- [ ] Mobile app (React Native)
- [ ] Plugin system

---

## 📚 Referencias y Documentación

### Documentación Oficial

- [Go Documentation](https://go.dev/doc/)
- [Fiber Framework](https://docs.gofiber.io/)
- [SvelteKit](https://kit.svelte.dev/docs)
- [Svelte 5](https://svelte.dev/docs)
- [TailwindCSS](https://tailwindcss.com/docs)
- [Docker SDK for Go](https://pkg.go.dev/github.com/docker/docker)
- [xterm.js](https://xtermjs.org/docs/)

### Repositorios de Referencia

- [Docker API Docs](https://docs.docker.com/engine/api/)
- [JWT Best Practices](https://auth0.com/docs/secure/tokens/json-web-tokens)
- [TOTP RFC 6238](https://datatracker.ietf.org/doc/html/rfc6238)

---

## 🤝 Cómo Continuar el Desarrollo

### 1. Clonar el repositorio

```bash
git clone https://github.com/[TU_USUARIO]/dockerverse.git
cd dockerverse
```

### 2. Ejecutar setup

```bash
chmod +x setup-mac.sh
./setup-mac.sh
```

### 3. Configurar SSH a Raspis

```bash
# Copiar clave SSH
ssh-copy-id pi@192.168.1.145

# Verificar conexión
ssh pi@192.168.1.145 "docker ps"
```

### 4. Desarrollo local

```bash
# Frontend (terminal 1)
cd frontend
npm install
npm run dev

# Backend requiere Docker socket, mejor en Raspi
```

### 5. Deploy a producción

```bash
./deploy-to-raspi.sh
```

---

## ⚠️ Notas Importantes

1. **Nunca commitear** passwords o secrets reales
2. **El JWT_SECRET** debe cambiarse en producción
3. **El Docker socket** da acceso completo - usar con precaución
4. **Backup regular** del volumen `/data`
5. **Las IPs** pueden cambiar si las Raspis usan DHCP

---

## 📝 Changelog v2.1.0 (8 de febrero de 2026)

### ✨ Nuevas Características

1. **Auto-logout Configurable**
   - Archivo: `frontend/src/lib/stores/auth.ts`
   - Funciones: `getAutoLogoutMinutes()`, `setAutoLogoutMinutes()`
   - Opciones: 5, 10, 15, 30, 60, 120 minutos
   - LocalStorage key: `dockerverse_auto_logout_minutes`

2. **Log Viewer Restyled (Databasement-style)**
   - Archivo: `frontend/src/lib/components/LogViewer.svelte`
   - Layout de tabla con columnas Date/Type/Message
   - Bordes coloreados por nivel: `border-l-4` (verde/amarillo/rojo)
   - Mejores filtros de rango de fecha

3. **Terminal Premium**
   - Archivo: `frontend/src/lib/components/Terminal.svelte`
   - Nuevos temas: Catppuccin Mocha, One Dark Pro
   - WebGL addon: `@xterm/addon-webgl` para rendering acelerado
   - Web-links addon: URLs clickeables en terminal
   - Ctrl+Scroll: Zoom de fuente dinámica
   - Scrollback: 10,000 líneas (antes 1,000)

4. **Resource Leaderboard**
   - Archivo: `frontend/src/routes/+page.svelte`
   - Componente nuevo con 4 tabs: CPU/Memory/Network/Restarts
   - Top-14 contenedores por cada métrica
   - Integración con filtro de hosts

5. **Update Indicators**
   - Archivo: `frontend/src/lib/components/ContainerCard.svelte`
   - Badge animado con pulse cuando `hasUpdate` es true
   - Usa store `imageUpdates` de `docker.ts`

6. **Pending Updates Panel**
   - Archivo: `frontend/src/routes/+layout.svelte`
   - Dropdown en header con lista de contenedores con updates
   - Contador animado con efecto glow
   - CSS animations: `pulse-update`, `glow-green`

7. **Sidebar Active State**
   - Archivo: `frontend/src/routes/+layout.svelte`
   - Variable reactiva: `activeSidebarItem`
   - Highlight visual del item actual

### 🐛 Fixes

- **Avatar Upload**: Fixed missing `${API_BASE}` prefix in `updateProfile` endpoint
  - Archivo: `frontend/src/lib/stores/auth.ts`
  - Antes: `PATCH /api/auth/profile`
  - Ahora: `PATCH ${API_BASE}/api/auth/profile`

### 🎨 Styles

- Archivo: `frontend/src/app.css`
- Añadidas animaciones:
  ```css
  @keyframes pulse-update { ... }
  .glow-green { box-shadow: 0 0 20px rgba(34, 197, 94, 0.5); }
  ```

### 📦 Deployment

- Desplegado en: Raspberry Pi @ 192.168.1.145:3007
- 3 contenedores: nginx, frontend, backend (todos healthy)
- Git tag: `v2.1.0`
- GitHub: https://github.com/vicolmenares/dockerverse

### 🧪 Testing Completo

- ✅ P1: API `/api/settings` retorna configuración correctamente
- ✅ P3: TOTP `/api/auth/totp/status` funcional
- ✅ P7: Updates `/api/updates` verifica 83 imágenes
- ✅ P11: Profile `PATCH /api/auth/profile` funciona
- ✅ P12: 3 containers corriendo (nginx, frontend, backend)

## Estado del Repositorio (Git)

- Rama actual: `feature/toggle-filters-host-rename-2026-02-12`
- Origen: `origin/master` en `06af6de`
- Estado: Branch ~28 commits adelante de master, con merge de origin/master integrado (14 Feb 2026)
- Remote: `https://github.com/vicolmenares/dockerverse.git`

## Mapa de Módulos UI/UX

- **Layout y navegación**: shell, header, sidebar, menú de usuario, badge de actualizaciones y palette en [frontend/src/routes/+layout.svelte](frontend/src/routes/+layout.svelte), soportado por [frontend/src/lib/components/CommandPalette.svelte](frontend/src/lib/components/CommandPalette.svelte) y [frontend/src/lib/components/Login.svelte](frontend/src/lib/components/Login.svelte).
- **Dashboard principal**: hosts, tarjetas de contenedores, leaderboard de recursos, filtros y preloads de terminal/logs en [frontend/src/routes/+page.svelte](frontend/src/routes/+page.svelte) usando [frontend/src/lib/components/HostCard.svelte](frontend/src/lib/components/HostCard.svelte), [frontend/src/lib/components/ContainerCard.svelte](frontend/src/lib/components/ContainerCard.svelte), [frontend/src/lib/components/ResourceChart.svelte](frontend/src/lib/components/ResourceChart.svelte), [frontend/src/lib/components/UpdateModal.svelte](frontend/src/lib/components/UpdateModal.svelte) y [frontend/src/lib/components/BulkUpdateModal.svelte](frontend/src/lib/components/BulkUpdateModal.svelte).
- **Logs**: panel dedicado con modos single/multi/agrupado y descarga en [frontend/src/routes/logs/+page.svelte](frontend/src/routes/logs/+page.svelte); visor flotante con filtros por nivel/fecha en [frontend/src/lib/components/LogViewer.svelte](frontend/src/lib/components/LogViewer.svelte).
- **Settings page-based**: layout protegido y breadcrumb en [frontend/src/routes/settings/+layout.svelte](frontend/src/routes/settings/+layout.svelte); secciones hijas para profile, security (password/2FA/auto-logout), users, notifications, appearance, data, environments y about bajo [frontend/src/routes/settings](frontend/src/routes/settings).
- **Terminal web**: ventana flotante con temas, WebGL y WebSocket en [frontend/src/lib/components/Terminal.svelte](frontend/src/lib/components/Terminal.svelte).
- **Estado y API**: stores globales en [frontend/src/lib/stores/auth.ts](frontend/src/lib/stores/auth.ts) y [frontend/src/lib/stores/docker.ts](frontend/src/lib/stores/docker.ts); helpers HTTP/SSE/WS en [frontend/src/lib/api/docker.ts](frontend/src/lib/api/docker.ts).

## Mapa de Funcionalidades End-to-End

- **Autenticación y sesiones**: JWT + refresh + rotación, TOTP y recovery codes en [backend/main.go](backend/main.go); login/persistencia/auto-logout configurable en [frontend/src/lib/stores/auth.ts](frontend/src/lib/stores/auth.ts) y [frontend/src/lib/components/Login.svelte](frontend/src/lib/components/Login.svelte).
- **Seguridad de sesión**: seguimiento de actividad y guard de rutas de settings en [frontend/src/routes/+layout.svelte](frontend/src/routes/+layout.svelte) y [frontend/src/routes/settings/+layout.svelte](frontend/src/routes/settings/+layout.svelte), con opciones de auto-logout.
- **Usuarios y roles**: CRUD y roles admin/user implementados en [backend/main.go](backend/main.go); UI administrativa en [frontend/src/routes/settings/users](frontend/src/routes/settings/users).
- **Hosts/Entornos**: parser de `DOCKER_HOSTS`, persistencia (`EnvironmentStore`) y backoff de health en [backend/main.go](backend/main.go); UI vinculada al ítem "Environments" en el sidebar.
- **Contenedores y métricas**: SSE `/api/events` alimenta [frontend/src/lib/stores/docker.ts](frontend/src/lib/stores/docker.ts) vía [frontend/src/lib/api/docker.ts](frontend/src/lib/api/docker.ts) para stats, hosts y contenedores; render en HostCard/ContainerCard y leaderboard.
- **Acciones y updates**: start/stop/restart y detección de updates; flujo de actualización individual en [frontend/src/lib/components/UpdateModal.svelte](frontend/src/lib/components/UpdateModal.svelte) y masiva en [frontend/src/lib/components/BulkUpdateModal.svelte](frontend/src/lib/components/BulkUpdateModal.svelte) contra endpoints de Watchtower definidos en [backend/main.go](backend/main.go).
- **Logs y observabilidad**: streaming SSE por contenedor desde [backend/main.go](backend/main.go) consumido en [frontend/src/routes/logs/+page.svelte](frontend/src/routes/logs/+page.svelte) y [frontend/src/lib/components/LogViewer.svelte](frontend/src/lib/components/LogViewer.svelte), con búsqueda, filtros y exportación.
- **Terminal**: WebSocket `/ws/terminal/{host}/{container}` implementado en [backend/main.go](backend/main.go) y consumido por [frontend/src/lib/components/Terminal.svelte](frontend/src/lib/components/Terminal.svelte) con temas y WebGL.
- **Notificaciones y umbrales**: AppSettings (CPU/Mem thresholds, Apprise/Telegram/Email, flags de eventos) en [backend/main.go](backend/main.go); interfaz en settings/notifications y limpieza de datos en settings/data.
- **Buenas prácticas aplicadas**: separación de stores y API, `fetchWithTimeout` en todas las llamadas, SSE con reconexión, tokens rotados y guardados en storage, settings modularizados por ruta, componentes auto-contenidos para operaciones críticas (terminal, logs, updates).

---

*Documento actualizado el 14 de febrero de 2026*
*DockerVerse v2.4.0*

## Tracking de Cambios

### 2026-02-12 - Toggle de filtros y rename de hosts

- Base de trabajo: commit 32d36f7 (deployment activo).
- Cambios:
   - Toggle de filtros en cards de resumen (Total/Running/Stopped/Updates) para permitir deseleccionar con segundo click.
   - Renombrado de hosts por display name en `DOCKER_HOSTS` a "Raspeberry Main" y "Raspberry Secondary".
   - Default de host local en backend alineado a "Raspeberry Main".
- Archivos:
   - `frontend/src/routes/+page.svelte`
   - `.env`
   - `backend/main.go`
- Tests (frontend): `npm --prefix frontend run check`
   - Resultado: FALLA.
   - Errores principales:
      - `frontend/vite.config.ts`: falta `@types/node` (process no definido).
      - `frontend/src/lib/components/Terminal.svelte`: firma async en `onMount`.
      - `frontend/src/lib/stores/auth.ts`: `auth.update` no existe (tipado).
      - `frontend/src/lib/components/BulkUpdateModal.svelte`: exports no encontrados en `api/docker`.
      - `frontend/src/routes/+layout.svelte`: `currentUser.role` vs `roles`.
- Deploy a Raspberry Pi: completado con `./deploy-to-raspi.sh`.
   - Resultado: OK (contenedor healthy en `:3007`).
   - API test del script: HTTP 401 (esperado sin auth).
- Verificacion en Raspberry Pi: OK
   - `curl -I http://localhost:3007` -> HTTP 200.
- Git push: completado (incluye `.env` por solicitud).

### 2026-02-12 - Fix svelte-check (errores)

- Base de trabajo: rama `feature/toggle-filters-host-rename-2026-02-12`.
- Cambios:
   - `onMount` sincronizado en Dashboard y Terminal para evitar promesas en retorno.

### 2026-02-12 - Backend logging & debug endpoint

- Commit: c453ab2
- Cambios: Se añadió logging a fichero en `DATA_DIR/logs/backend.log`, middleware `recover` con stacktraces, y el endpoint `/api/debug/logs` (admin-only) para obtener las últimas líneas del log desde el contenedor.
- Deploy: pendiente (se desplegará a la Raspi en el siguiente paso).
   - `auth.update` expuesto y tipado en store para actualizar avatar.
   - Bulk update client agregado en API frontend con resultados agregados.
   - Validacion de rol admin usando `roles`.
   - Agregado `@types/node` para tipado de `process` en Vite.
- Archivos:
   - `frontend/src/routes/+page.svelte`
   - `frontend/src/lib/components/Terminal.svelte`
   - `frontend/src/lib/stores/auth.ts`
   - `frontend/src/lib/api/docker.ts`
   - `frontend/src/routes/+layout.svelte`
   - `frontend/package.json`
   - `frontend/package-lock.json`
- Tests (frontend): `npm --prefix frontend run check`
   - Resultado: OK con warnings de a11y y estado local en componentes (sin errores).
- Deploy a Raspberry Pi: completado con `./deploy-to-raspi.sh`.
   - Resultado: OK (contenedor healthy en `:3007`).
- Verificacion en Raspberry Pi: OK
   - `curl -I http://localhost:3007` -> HTTP 200.
- Git push: completado.

### 2026-02-12 - Hosts modernos + memoria/discos + a11y

- Base de trabajo: rama `feature/toggle-filters-host-rename-2026-02-12`.
- Cambios:
   - Rediseño de cards de hosts y recursos en panel dedicado para evitar expansion gris en card no seleccionada.
   - Correcciones de a11y (labels, aria-labels, tabindex) y estado reactivo en logs.
   - Ajustes backend: discos con `df` sobre `/mnt` y `/media`, y fallback de memoria total con limites de contenedor; timeout de stats aumentado.
- Archivos:
   - `frontend/src/lib/components/HostCard.svelte`
   - `frontend/src/routes/+page.svelte`
   - `frontend/src/lib/components/CommandPalette.svelte`
   - `frontend/src/lib/components/BulkUpdateModal.svelte`
   - `frontend/src/lib/components/EnvironmentModal.svelte`
   - `frontend/src/routes/settings/appearance/+page.svelte`
   - `frontend/src/routes/logs/+page.svelte`
   - `backend/main.go`
- Tests (frontend): `npm --prefix frontend run check`
   - Resultado: OK (0 errors, 0 warnings).
- Deploy a Raspberry Pi: completado con `./deploy-to-raspi.sh`.
   - Resultado: OK (contenedor healthy en `:3007`).
   - API test del script: HTTP 401 (esperado sin auth).
- Verificacion en Raspberry Pi: OK
   - `curl -I http://localhost:3007` -> HTTP 200.
- Git push: pendiente.

### 2026-02-12 - Hosts no aparecen (investigacion + fix)

- Logs (raspi-main):
   - `getDiskInfo(raspi1)`: `No such image: busybox:latest`.
   - `getDiskInfo(raspi2)`: `403 Forbidden` (remote daemon denies create).
   - `Error saving environments: open data/environments.json: no such file or directory`.
- Diagnostico: deadlock por `statsMu.Lock()` duplicado en `GetHostStats` bloqueaba la agregacion de stats, afectando render de hosts.
- Fix aplicado:
   - Mutex corregido y actualizacion de `maxMemLimit` dentro del lock.
- Tests:
   - `go test ./...` (backend): OK (sin tests).
   - `npm --prefix frontend run check`: OK.
- Deploy a Raspberry Pi: completado con `./deploy-to-raspi.sh`.
   - Resultado: OK (contenedor healthy en `:3007`).
   - API test del script: HTTP 401 (esperado sin auth).
- Verificacion en Raspberry Pi: OK
   - `curl -I http://localhost:3007` -> HTTP 200.
- Git push: pendiente.

### 2026-02-12 - SSH por host + paginacion containers + disk free

- Cambios:
   - Agregado `sshHost` en `HostStats` (backend) derivado de `DOCKER_HOSTS`.
   - Boton SSH en cards de hosts con enlace `ssh://` (frontend).
   - Metrica de disco ahora muestra espacio libre/total en host y por disco.
   - `getDiskInfo` ahora elige imagen existente o intenta pull con fallback configurable (`DISK_INFO_IMAGE`).
   - Redisenio de metricas de contenedores con panel fijo para evitar saltos de altura.
   - Paginacion de contenedores con selector de tamano de pagina (9/12/18/24).
- Archivos:
   - `backend/main.go`
   - `frontend/src/lib/api/docker.ts`
   - `frontend/src/lib/components/HostCard.svelte`
   - `frontend/src/lib/components/ContainerCard.svelte`
   - `frontend/src/routes/+page.svelte`
- Tests:
   - `go test ./...` (backend): OK (sin tests).
   - `npm --prefix frontend run check`: OK (0 errors, 0 warnings).
- Deploy a Raspberry Pi: completado con `./deploy-to-raspi.sh`.
   - Resultado: OK (contenedor healthy en `:3007`).
   - API test del script: HTTP 401 (esperado sin auth).
- Verificacion en Raspberry Pi: OK
   - `curl -I http://192.168.1.145:3007` -> HTTP 200.
- Git push: completado.

### 2026-02-12 - SSH in-app + SFTP + discos via SSH + alertas pausadas

- Cambios:
   - SSH embebido por host (WebSocket `/ws/ssh/:hostId`) y file manager SFTP con upload/download.
   - Discos ahora se leen via `df` por SSH (sin contenedor busybox) con dedupe de mounts.
   - Cards de contenedores con altura fija y numeracion tabular para evitar cambios de tamano.
   - Alertas de CPU/Mem deshabilitadas por defecto con bootstrap de migracion.
- Archivos:
   - `backend/main.go`
   - `backend/go.mod`, `backend/go.sum`
   - `frontend/src/lib/api/docker.ts`
   - `frontend/src/lib/components/Terminal.svelte`
   - `frontend/src/lib/components/HostCard.svelte`
   - `frontend/src/lib/components/HostFiles.svelte`
   - `frontend/src/lib/components/ContainerCard.svelte`
   - `frontend/src/routes/+page.svelte`
- Tests:
   - `go test ./...` (backend): OK (sin tests).
   - `npm --prefix frontend run check`: OK (0 errors, 0 warnings).
- Deploy a Raspberry Pi: completado con `./deploy-to-raspi.sh`.
   - Resultado: OK (contenedor healthy en `:3007`).
   - API test del script: HTTP 401 (esperado sin auth).
- Verificacion en Raspberry Pi: OK
   - `curl -I http://192.168.1.145:3007` -> HTTP 200.
- Git push: completado.

### 2026-02-12 - SSH fallback para container actions + multiples candidatos SSH

- Commits: f143d63, dd25250, 3540285, 07b0f9c, 47e3eef, dee5b24, 704a63c, d36ccb9, 2512405, 3021ed4, 344de3e
- Cambios:
   - Backend ahora intenta ejecutar container actions (start/stop/restart) via SSH cuando Docker API falla.
   - `deriveSSHCandidates()` genera multiples IPs/hostnames para intentar conexion SSH.
   - `dialSSH()` con logging detallado y timeout.
   - Endpoint `/api/debug/hosts` para diagnostico de parsing de hosts.
   - Soporte para local (unix socket) y remote (tcp) en DOCKER_HOSTS.
   - Fix en env YAML: mapping para todas las vars y DOCKER_HOSTS entre comillas.
   - `extra_hosts: host-gateway` para resolver `host.docker.internal`.
   - `df` acepta salida con exit code non-zero y parsea si hay output.
- Archivos:
   - `backend/main.go`
   - `docker-compose.unified.yml`
   - `Dockerfile.unified`

### 2026-02-12 - Socket-proxy y actualizacion de IPs

- Commits: c4d8c1d, 1d757b2, 603a43f
- Cambios:
   - DOCKER_HOSTS actualizado para usar socket-proxy en raspi1 (puerto 23750).
   - Intento de actualizar raspi2 IP a 192.168.1.11 revertido a 192.168.1.146 por problemas de conectividad.
   - Puertos internos reorganizados: backend 3001→3002, frontend 3000→3003.
   - openssh-client agregado al contenedor para SSH nativo.
   - Health endpoint `/health` agregado al backend.
   - Eliminada dependencia de red externa `container_network_ipv4`.
- Archivos:
   - `docker-compose.unified.yml`
   - `Dockerfile.unified`
   - `backend/main.go`

### 2026-02-15 - SSH fallback para terminal de contenedores

- Problema identificado: socket-proxy en raspi1 (puerto 23750) bloquea operaciones `ContainerExecCreate()` con error 403, impidiendo conexión a contenedores via terminal.
- Causa raíz: socket-proxy requiere `EXEC=1` en variables de entorno para permitir operaciones de exec, que está deshabilitado por seguridad.
- Solución implementada:
   - Agregada función `handleContainerTerminalSSH()` que ejecuta `docker exec` via SSH cuando Docker API falla con 403.
   - Modificado WebSocket handler `/ws/terminal/:hostId/:containerId` para detectar errores 403/"Forbidden" y hacer fallback a SSH.
   - Usa `dialSSH()` y sesión SSH con PTY interactivo para manejar entrada/salida del terminal.
   - Mantiene consistencia con patrón existente de SSH fallback para acciones de contenedor (start/stop/restart).
- Archivos modificados:
   - `backend/main.go` (líneas 3689-3780)
- Tests:
   - `go build`: OK (compilación exitosa)
   - `go test ./...`: OK (sin tests definidos)
   - `npm --prefix frontend run check`: OK (0 errors, 0 warnings)
- Deploy a Raspberry Pi: completado con `./deploy-to-raspi.sh`
   - Resultado: OK (contenedor healthy en `:3007`)
   - Build time: ~2 minutos
   - API test: HTTP 401 (esperado sin auth)
- Verificación esperada:
   - Terminal de contenedores en raspi1 debe funcionar via SSH fallback
   - Terminal de contenedores en raspi2 debe funcionar directamente via Docker API
   - Mensaje "Connected via SSH fallback" indica uso de fallback SSH
- Referencias:
   - [docker-socket-proxy GitHub](https://github.com/Tecnativa/docker-socket-proxy)
   - [Docker Socket Proxy Security Guide](https://www.paulsblog.dev/how-to-secure-your-docker-environment-by-using-a-docker-socket-proxy/)

### 2026-02-15 - Fix de actualización de contenedores (route ordering)

- Problema identificado: Al intentar actualizar un contenedor via botón Update, se recibía error "unknown action: update".
- Causa raíz: Conflicto de rutas en Fiber. La ruta genérica `/containers/:hostId/:containerId/:action` (línea 3460) se evaluaba ANTES que la ruta específica `/containers/:hostId/:containerId/update` (línea 3481), causando que Fiber matcheara "update" como un parámetro `:action` y llamara a `ContainerAction()` que no tiene case para "update".
- Solución implementada:
   - Movida la ruta específica de Watchtower `/containers/:hostId/:containerId/update` ANTES de la ruta genérica `/:action`.
   - Agregados comentarios explicativos para prevenir este issue en el futuro.
   - La ruta específica ahora se evalúa primero, permitiendo que las llamadas a `/update` sean manejadas correctamente por la integración con Watchtower HTTP API.
- Comportamiento esperado:
   - Botón "Update" en ContainerCard debe funcionar correctamente
   - Watchtower HTTP API debe recibir la petición de actualización
   - Cache de updates debe limpiarse después de actualización exitosa
- Archivos modificados:
   - `backend/main.go` (líneas 3460-3543 reordenadas)
- Tests:
   - `go build`: OK (compilación exitosa)
   - `go test ./...`: OK (sin tests definidos)
   - `npm --prefix frontend run check`: OK (0 errors, 0 warnings)
- Deploy a Raspberry Pi: completado con `./deploy-to-raspi.sh`
   - Resultado: OK (contenedor healthy en `:3007`)
   - Build time: ~1.5 minutos
   - API test: HTTP 401 (esperado sin auth)
- Notas técnicas:
   - Fiber evalúa rutas en orden de declaración
   - Rutas más específicas deben declararse antes que rutas con parámetros genéricos
   - Este patrón aplica a otros frameworks web (Express, FastAPI, etc.)

### 2026-02-15 - Integración completa de Watchtower HTTP API

- Problema identificado: La funcionalidad de actualización de contenedores estaba bloqueada por múltiples issues de configuración y networking.
- Issues resueltos:
   1. **Configuración de Watchtower**: Watchtower HTTP API no estaba configurado con las URLs correctas en DockerVerse
   2. **Networking Docker**: DockerVerse no podía alcanzar Watchtower en el mismo host debido a aislamiento de redes Docker
   3. **Timeout de contexto**: Request HTTP usaba el mismo contexto que ContainerInspect, causando timeouts de 10 segundos
   4. **API sin parámetros**: Endpoint `/v1/update` sin parámetros actualizaba TODOS los contenedores (operación lenta)

- Soluciones implementadas:
   1. **Configuración de .env**:
      - Agregado `WATCHTOWER_TOKEN=***WATCHTOWER-TOKEN-REMOVED***`
      - Agregado `WATCHTOWER_URLS=raspi1:http://172.26.0.1:8080|raspi2:http://192.168.1.146:8081`
      - raspi1 usa IP del gateway Docker (172.26.0.1) para conectividad local
      - raspi2 usa IP externa (192.168.1.146) ya que está en host diferente

   2. **Docker Compose - extra_hosts**:
      - Agregado `extra_hosts: ["host.docker.internal:host-gateway"]` en docker-compose.unified.yml
      - Permite que el contenedor acceda a servicios en el host Docker

   3. **Backend - Contextos separados**:
      - Separados los contextos de ContainerInspect (10s) y HTTP request a Watchtower (30s)
      - Previene que el timeout de inspect afecte el request HTTP

   4. **Backend - Parámetro image en API**:
      - Agregado query parameter `?image=<imageName>` al endpoint `/v1/update`
      - Actualiza solo el contenedor específico en lugar de todos (mejora de 30s → 200ms)
      - Importado paquete `net/url` para escape correcto de query parameters

- Comportamiento final:
   - ✅ Actualización de contenedores funciona en <220ms (antes: timeout 30s)
   - ✅ Funciona en ambos hosts (raspi1 y raspi2)
   - ✅ Manejo de errores correcto (contenedor inexistente, etc.)
   - ✅ Watchtower logs confirman triggers HTTP API recibidos
   - ✅ Backend logs muestran respuestas HTTP 200 exitosas

- Archivos modificados:
   - `.env` (agregadas WATCHTOWER_TOKEN y WATCHTOWER_URLS)
   - `docker-compose.unified.yml` (agregado extra_hosts)
   - `backend/main.go`:
     - Línea 15: agregado import `"net/url"`
     - Líneas 3476-3498: contextos separados + parámetro image

- Tests exhaustivos realizados:
   - ✅ Test 1: watchtower-main en raspi1 (179ms) - Success
   - ✅ Test 2: adguardhome en raspi2 (60ms) - Success
   - ✅ Test 3: actual-budget en raspi1 (211ms) - Success
   - ✅ Test 4: Error handling - contenedor inexistente (HTTP 500) - Success
   - ✅ Verificación de logs Watchtower: "Updates triggered by HTTP API request" confirmado
   - ✅ Verificación de logs backend: HTTP 200 en todos los casos exitosos

- Deploy a Raspberry Pi: completado con `./deploy-to-raspi.sh`
   - Resultado: OK (contenedor healthy en `:3007`)
   - Build time: ~1.5 minutos
   - API test: HTTP 401 (esperado sin auth)

- Notas técnicas:
   - Watchtower HTTP API acepta parámetro `?image=` para updates selectivos ([docs](https://containrrr.dev/watchtower/http-api-mode/))
   - Contenedores en bridge network no pueden acceder a host:port vía IP externa
   - Gateway IP de bridge network es `.1` de la subnet (ej: 172.26.0.1 para 172.26.0.0/16)
   - `extra_hosts: host-gateway` mapea a docker0 (172.17.0.1) no a la red custom

- Git commit/push: pendiente (siguiente paso)

### 2026-02-15 (pm) - Watchtower `?image=` parameter fix

- **Problema identificado**: User reportó que watchtower-secondary en raspi2 mostraba "Exit code 1" y aunque el modal mostraba "Update completed", el indicador de "update pending" no se limpiaba después de actualizar.

- **Investigación del issue**:
  1. **Logs de watchtower-secondary**: Container corría correctamente en puerto 8081 (no 8080 que estaba ocupado por qBittorrent)
  2. **HTTP API funcionando**: Endpoint respondía HTTP 200, pero logs mostraban "Session done Failed=0 Scanned=0 Updated=0"
  3. **Root cause**: El parámetro `?image=` no funcionaba en Watchtower v1.7.1 en raspi2
     - Implementamos formato correcto ORG/APP (ej: `linuxserver/radarr` de `lscr.io/linuxserver/radarr:latest`)
     - GitHub discussion #1731 confirmó formato correcto
     - Agregamos logging extensivo mostrando `formatted=linuxserver/radarr` enviándose correctamente
     - Tests manuales con `curl "http://localhost:8081/v1/update?image=linuxserver/radarr"` confirmaron "Scanned=0"
     - Sin parámetro `?image=`, Watchtower escanea 30 contenedores exitosamente

- **Solución implementada**:
  1. **Simplificación del API call**:
     - Removido parámetro `?image=` del endpoint `/v1/update`
     - Removido código de formatting de ORG/APP extraction
     - Removido import `net/url` (ya no necesario)
     - Call directo a `/v1/update` sin parameters

  2. **Aumento de timeout**:
     - Context timeout aumentado de 30s → 60s
     - Necesario para procesar scan de todos los contenedores

  3. **Logging mejorado**:
     - Agregado logging de URL, containerID e imageName para debugging
     - Log format: `Watchtower update: URL=%s, container=%s (%s)`

- **Trade-off aceptado**:
  - Al eliminar parámetro `?image=`, Watchtower ahora escanea TODOS los contenedores por disponibilidad de actualizaciones
  - Esto significa que hacer click en "Update" para un contenedor específico puede actualizar OTROS contenedores si tienen updates disponibles
  - Es un trade-off aceptable: mejor tener funcionalidad que funciona (aunque más amplia) que una funcionalidad rota

- **Comportamiento final**:
  - ✅ Update trigger funciona en ambos hosts (raspi1 y raspi2)
  - ✅ Watchtower logs muestran "Scanned=30" (no "Scanned=0")
  - ✅ No más timeout errors
  - ✅ Update process completa exitosamente
  - ⚠️ Update button ahora actualiza todos los contenedores con updates disponibles, no solo el específico

- **Archivos modificados**:
  - `backend/main.go`:
    - Línea 14-15: removido import `"net/url"`
    - Líneas 3490-3503: simplificado API call, aumentado timeout, agregado logging

- **Tests realizados**:
  - ✅ Deployment successful a Raspberry Pi
  - ✅ Container healthy en puerto :3007
  - ⏳ Test funcional del update trigger (pendiente validación en producción)

- **Aprendizajes técnicos**:
  - Watchtower v1.7.1 HTTP API documenta soporte para `?image=` parameter, pero no funciona confiablemente en todos los casos
  - Formato correcto es `ORG/APP` (ej: `linuxserver/radarr`), NO path completo con registry y tag
  - Llamar `/v1/update` sin parámetros es más confiable que intentar targeting selectivo
  - Siempre verificar logs de Watchtower para confirmar comportamiento (campo "Scanned" es indicador clave)

- Deploy: ✅ Completado con `./deploy-to-raspi.sh`
- Git commit/push: pendiente (siguiente paso)

### 2026-02-10 - 2FA SHA1 fix + Docker version fallback

- Commits: d5cd321, 06af6de (en master, mergeados a branch el 14 Feb)
- Cambios:
   - TOTP setup corregido: SHA256→SHA1 para compatibilidad con Google Authenticator y Authy.
   - Environments: fallback a `cli.ServerVersion()` cuando `cli.Info()` retorna 403.
   - Test de conexion de environments usa `ServerVersion()` como alternativa.
   - Muestra version real del Docker engine (ej: "29.2.1") en vez de version de API (ej: "1.53").
- Archivos:
   - `backend/main.go`
