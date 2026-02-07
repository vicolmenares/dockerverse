# DockerVerse 🐳

A modern, real-time Docker monitoring and management portal built with **Svelte 5 + SvelteKit** frontend and **Go + Fiber** backend.

![DockerVerse](https://img.shields.io/badge/DockerVerse-v1.0-blue)
![Svelte](https://img.shields.io/badge/Svelte-5.0-orange)
![Go](https://img.shields.io/badge/Go-1.21+-cyan)

## ✨ Features

- **Real-time Dashboard**: Live metrics (CPU, Memory, Network) via SSE streaming
- **Multi-host Support**: Monitor containers across multiple Docker hosts
- **Web Terminal**: Interactive terminal (exec) powered by xterm.js
- **Log Viewer**: Stream and download container logs with ANSI color support
- **Container Management**: Start, stop, restart containers with one click
- **Command Palette**: Quick search (⌘K / Ctrl+K) for instant container access
- **Modern UI**: Tokyo Night dark theme with smooth animations

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                       DockerVerse                           │
├─────────────────────────────────────────────────────────────┤
│  Frontend (Svelte 5 + SvelteKit)                           │
│  ├── Dashboard with real-time metrics                       │
│  ├── Terminal (xterm.js + WebSocket)                       │
│  └── Logs viewer (SSE streaming)                           │
├─────────────────────────────────────────────────────────────┤
│  Backend (Go + Fiber)                                       │
│  ├── REST API for container management                      │
│  ├── SSE for metrics streaming                              │
│  ├── WebSocket for terminal sessions                        │
│  └── Docker SDK for multi-host management                   │
├─────────────────────────────────────────────────────────────┤
│  Docker Hosts                                               │
│  ├── raspi1 (local via unix socket)                        │
│  └── raspi2 (remote via TCP 2375)                          │
└─────────────────────────────────────────────────────────────┘
```

## 🚀 Quick Start

### Prerequisites

- Docker & Docker Compose
- Node.js 20+ (for local development)
- Go 1.21+ (for local development)

### Deploy with Docker Compose

```bash
# Clone and navigate to the project
cd dockerverse

# Build and start all services
docker-compose up -d --build

# Access the portal
open http://localhost:3000
```

### Local Development

**Backend:**
```bash
cd backend
go mod download
go run main.go
# API runs on http://localhost:3001
```

**Frontend:**
```bash
cd frontend
npm install
npm run dev
# Dev server runs on http://localhost:5173
```

## 🔧 Configuration

### Environment Variables

**Backend (.env):**
```env
PORT=3001
DOCKER_HOST=unix:///var/run/docker.sock
```

**Frontend (.env):**
```env
PUBLIC_API_URL=http://localhost:3001
```

### Adding Remote Docker Hosts

Edit `backend/main.go` and add hosts to the `dockerConfigs` map:

```go
dockerConfigs := map[string]string{
    "raspi1": "unix:///var/run/docker.sock",
    "raspi2": "tcp://192.168.1.146:2375",
    "server": "tcp://10.0.0.50:2375",
}
```

> **Note:** For remote hosts, ensure Docker daemon is configured to accept TCP connections.

## 📁 Project Structure

```
dockerverse/
├── backend/
│   ├── main.go           # Go backend with Fiber
│   ├── Dockerfile        # Multi-stage Go build
│   └── go.mod
├── frontend/
│   ├── src/
│   │   ├── routes/
│   │   │   ├── +layout.svelte
│   │   │   └── +page.svelte
│   │   └── lib/
│   │       ├── api/docker.ts       # API client
│   │       ├── stores/docker.ts    # Svelte stores
│   │       └── components/
│   │           ├── HostCard.svelte
│   │           ├── ContainerCard.svelte
│   │           ├── Terminal.svelte
│   │           ├── LogViewer.svelte
│   │           └── CommandPalette.svelte
│   ├── Dockerfile        # Multi-stage Node build
│   └── package.json
└── docker-compose.yml
```

## 🌐 DNS & Proxy Setup

### AdGuard Home DNS Entry
Add a rewrite rule pointing to your Docker host:
```
docker-connect.nerdslabs.com → 192.168.1.145
```

### Nginx Proxy Manager
Create a Proxy Host:
- **Domain:** docker-connect.nerdslabs.com
- **Forward Host:** 192.168.1.145
- **Forward Port:** 3000
- **SSL:** Request Let's Encrypt certificate

## 🎨 Design System

The UI uses a Tokyo Night-inspired color palette:

| Color | Hex | Usage |
|-------|-----|-------|
| Background | `#1a1b26` | Main background |
| Secondary | `#24283b` | Cards, panels |
| Primary | `#7aa2f7` | Accents, buttons |
| Running | `#9ece6a` | Running state |
| Stopped | `#f7768e` | Stopped state |
| Paused | `#e0af68` | Paused state |

## 📝 API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/containers` | List all containers |
| GET | `/api/hosts` | List all hosts with stats |
| GET | `/api/search?q=` | Search containers |
| GET | `/api/stats/stream` | SSE stats stream |
| POST | `/api/hosts/:id/containers/:cid/:action` | Container action |
| GET | `/api/hosts/:id/containers/:cid/logs` | Get logs |
| WS | `/ws/terminal/:hostId/:containerId` | Terminal WebSocket |

## 🛠️ Tech Stack

- **Frontend:** Svelte 5, SvelteKit, TypeScript, Tailwind CSS, xterm.js
- **Backend:** Go, Fiber, Docker SDK
- **Real-time:** Server-Sent Events, WebSockets
- **Deployment:** Docker, Docker Compose

## 📄 License

MIT License - feel free to use and modify!

---

Built with 💙 for the homelab community
