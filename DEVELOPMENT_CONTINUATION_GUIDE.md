# DockerVerse - Guía Completa de Continuación de Desarrollo

> **Documento de transferencia de conocimiento para continuar el desarrollo desde macOS**
> 
> Última actualización: 8 de febrero de 2026
> Versión actual: v2.2.0

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

---

## 🎯 Resumen Ejecutivo

**DockerVerse** es un dashboard de gestión multi-host de Docker, diseñado para administrar contenedores en múltiples Raspberry Pis desde una interfaz web moderna. El proyecto se desarrolló completamente desde cero usando:

- **Backend**: Go 1.22 con Fiber framework
- **Frontend**: SvelteKit 2.x con Svelte 5, TailwindCSS 3.4
- **Deployment**: Docker con arquitectura unificada (single container)
- **Target**: Raspberry Pi 4/5 con Docker instalado

### Características Principales (v2.1.0)

- ✅ Gestión multi-host de contenedores Docker
- ✅ Terminal web con WebSocket (7 temas, búsqueda, reconexión, WebGL, zoom)
- ✅ Visor de logs estilo Databasement con filtros avanzados
- ✅ Gráficos de recursos en tiempo real (CPU, RAM, Red, Disco)
- ✅ Resource Leaderboard con tabs (CPU/Memory/Network/Restarts)
- ✅ Sistema de autenticación con JWT + TOTP/MFA
- ✅ Detección de actualizaciones con indicadores animados
- ✅ Panel de actualizaciones pendientes con dropdown
- ✅ Subida de avatar de usuario
- ✅ Auto-logout configurable (5, 10, 15, 30, 60, 120 min)
- ✅ Command Palette (Ctrl+K)
- ✅ Sidebar con estado activo resaltado
- ✅ Soporte multi-idioma (ES/EN)
- ✅ Tema oscuro nativo
- ✅ Settings con navegación por rutas SvelteKit (v2.2.0)

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
│  │  (Port 80)  │  │ (Port 3001) │  │   (Port 3000)   │    │
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
                              ▼
                    ┌─────────────────┐
                    │  Docker Socket  │
                    │   (Read-Only)   │
                    └─────────────────┘
```

### Diagrama de Red Multi-Host

```
┌─────────────────────────────────────────────────────────────────┐
│                     RED LOCAL (192.168.1.x)                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────────┐                                           │
│  │   Windows Dev    │                                           │
│  │  (Este equipo)   │ ─────────────────────────┐               │
│  │  SSH + SCP       │                          │               │
│  └──────────────────┘                          │               │
│                                                 ▼               │
│  ┌──────────────────┐    ┌──────────────────┐  │               │
│  │   Raspberry Pi   │    │   Raspberry Pi   │  │               │
│  │  192.168.1.145   │    │  192.168.1.146   │  │               │
│  │  (Server Main)   │    │   (Server 2)     │  │               │
│  │  Port: 3007      │    │   Port: 3006     │  │               │
│  │  DockerVerse     │    │   Docker Host    │  │               │
│  └──────────────────┘    └──────────────────┘  │               │
│           │                       │             │               │
│           └───────────────────────┘             │               │
│                     │                           │               │
│                     ▼                           │               │
│          Docker API via SSH                     │               │
│                                                 │               │
└─────────────────────────────────────────────────────────────────┘
```

---

## 🛠️ Stack Tecnológico Completo

### Backend (Go)

| Componente | Versión | Propósito |
|------------|---------|-----------|
| Go | 1.22+ | Lenguaje principal |
| Fiber | v2.52.0 | Framework web HTTP |
| fiber/websocket | v2.2.1 | WebSocket support |
| docker/docker | v27.0.0 | Docker API client |
| golang-jwt/jwt | v5.2.1 | JSON Web Tokens |
| pquerna/otp | v1.4.0 | TOTP/2FA support |
| creack/pty | v1.1.21 | Terminal pseudo-TTY |
| golang.org/x/crypto | v0.25.0 | bcrypt hashing |

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

### Infrastructure

| Componente | Versión | Propósito |
|------------|---------|-----------|
| Docker | 24.x+ | Containerization |
| Docker Compose | 2.x | Orchestration |
| Nginx | 1.25 | Reverse proxy |
| s6-overlay | v3 | Process supervisor |
| Alpine Linux | 3.19 | Base image |

### Herramientas de Desarrollo (Windows)

| Herramienta | Versión | Propósito |
|-------------|---------|-----------|
| VS Code | Latest | IDE principal |
| Node.js | 20.x LTS | Runtime frontend dev |
| npm | 10.x | Package manager |
| Go | 1.22.x | Backend development |
| PowerShell | 7.x | Scripting |
| Posh-SSH | 3.2.7 | SSH/SCP desde PowerShell |
| Git | 2.x | Version control |
| GitHub CLI | 2.x | GitHub operations |

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
│   └── main.go                # Backend principal (~3500 líneas)
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
│   │   │   │   ├── CommandPalette.svelte
│   │   │   │   ├── ContainerCard.svelte
│   │   │   │   ├── HostCard.svelte
│   │   │   │   ├── Login.svelte
│   │   │   │   ├── LogViewer.svelte
│   │   │   │   ├── ResourceChart.svelte
│   │   │   │   ├── Settings.svelte (legacy, no longer imported)
│   │   │   │   └── Terminal.svelte
│   │   │   ├── settings/
│   │   │   │   └── index.ts   # Shared translations & types
│   │   │   └── stores/
│   │   │       ├── auth.ts    # Auth store (~550 líneas)
│   │   │       └── docker.ts  # Docker store (~500 líneas)
│   │   └── routes/
│   │       ├── +layout.svelte # Main layout (~640 líneas)
│   │       ├── +page.svelte   # Dashboard page
│   │       └── settings/      # Settings pages (v2.2.0)
│   │           ├── +layout.svelte     # Settings layout + auth guard
│   │           ├── +page.svelte       # Settings menu
│   │           ├── profile/+page.svelte
│   │           ├── security/+page.svelte
│   │           ├── users/+page.svelte
│   │           ├── notifications/+page.svelte
│   │           ├── appearance/+page.svelte
│   │           ├── data/+page.svelte
│   │           └── about/+page.svelte
│   └── static/
│       ├── robots.txt
│       └── sw.js              # Service worker stub
├── nginx/
│   └── nginx.conf             # Nginx configuration
├── docker-compose.yml         # Multi-container (legacy)
├── docker-compose.unified.yml # Single container
├── Dockerfile.unified         # Unified build
├── transfer.ps1               # Windows deploy script
├── sync.ps1                   # Sync script
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

---

## 💻 Configuración del Entorno de Desarrollo

### Variables de Entorno

```bash
# Backend
PORT=3001
DOCKER_HOST=unix:///var/run/docker.sock
JWT_SECRET=***JWT-SECRET-REMOVED***
DATA_PATH=/data

# Frontend
NODE_ENV=production
ORIGIN=http://localhost:3007
PUBLIC_API_URL=  # Empty for same-origin

# Container
TZ=America/Mexico_City
S6_VERBOSITY=1
```

### Puertos Utilizados

| Puerto | Servicio | Descripción |
|--------|----------|-------------|
| 3000 | SvelteKit | Frontend SSR |
| 3001 | Go/Fiber | Backend API |
| 3006 | DockerVerse Prev | Versión anterior |
| 3007 | DockerVerse | Producción |
| 80 | Nginx (container) | Reverse proxy |

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
| Go | 1.22+ | `brew install go` | Backend |
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

| Host | IP | Usuario | Password | Puerto DockerVerse |
|------|-----|---------|----------|-------------------|
| raspi-main | 192.168.1.145 | pi | Pi16870403 | 3007 |
| raspi-secondary | 192.168.1.146 | pi | Pi16870403 | N/A |

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
docker-compose -f docker-compose.unified.yml down
docker-compose -f docker-compose.unified.yml build --no-cache
docker-compose -f docker-compose.unified.yml up -d

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
#!/bin/bash
# Uso: ./deploy-to-raspi.sh [--no-cache]

RASPI_HOST="pi@192.168.1.145"
RASPI_PATH="/home/pi/dockerverse"
NO_CACHE=${1:-""}

echo "📦 Syncing files..."
rsync -avz --exclude 'node_modules' --exclude '.git' \
  --exclude 'test-*' ./ $RASPI_HOST:$RASPI_PATH/

echo "🔨 Building on Raspberry Pi..."
ssh $RASPI_HOST "cd $RASPI_PATH && \
  docker-compose -f docker-compose.unified.yml down && \
  docker-compose -f docker-compose.unified.yml build $NO_CACHE && \
  docker-compose -f docker-compose.unified.yml up -d"

echo "✅ Waiting for container..."
sleep 10

echo "🔍 Checking status..."
ssh $RASPI_HOST "docker ps | grep dockerverse"

echo "🎉 Deploy complete!"
```

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
docker-compose -f docker-compose.unified.yml restart
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

### v2.3.0 (Planificado)

- [ ] Container Activity chart (bar chart estilo Jobs Activity)
- [ ] Docker Compose management (view/edit compose files)
- [ ] Container creation wizard
- [ ] Network visualization
- [ ] Volume management UI
- [ ] Container templates/presets

### v2.4.0 (Planificado)

- [ ] Multi-user permissions (RBAC)
- [ ] Audit log
- [ ] API keys for automation
- [ ] Webhook integrations
- [ ] Dashboard widgets customization

### v3.0.0 (Futuro)

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

---

*Documento generado el 8 de febrero de 2026*
*DockerVerse v2.1.0*
