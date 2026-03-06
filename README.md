# DockerVerse 🐳

> **Multi-Host Docker Management Dashboard**
> 
> A modern, real-time Docker monitoring and management portal built with **Svelte 5 + SvelteKit** frontend and **Go + Fiber** backend.

![DockerVerse](https://img.shields.io/badge/DockerVerse-v2.5.0-blue)
![Svelte](https://img.shields.io/badge/Svelte-5.0-orange)
![Go](https://img.shields.io/badge/Go-1.22+-cyan)
![License](https://img.shields.io/badge/License-MIT-green)

## ✨ Features

### Core Features
- **🖥️ Multi-host Dashboard**: Monitor containers across multiple Docker hosts
- **📊 Real-time Metrics**: Live CPU, Memory, Network, Disk stats with sparkline charts
- **🔲 Web Terminal**: Interactive terminal with 5 themes, search, and auto-reconnection
- **📋 Log Viewer**: Advanced filtering by date, time, and log level
- **🎛️ Container Management**: Start, stop, restart containers with one click
- **⌨️ Command Palette**: Quick search (⌘K / Ctrl+K) for instant access

### Security (v2.0.0)
- **🔐 JWT Authentication**: Access + Refresh tokens with rotation
- **📱 2FA/TOTP**: Two-factor authentication with QR code setup
- **🔑 Recovery Codes**: 10 backup codes for 2FA recovery
- **⏰ Auto-logout**: Automatic logout after 30 min of inactivity
- **👤 Avatar Upload**: Personalized user profiles

### Monitoring (v2.0.0)
- **📈 Resource Charts**: Expandable sparkline graphs under each host
- **🔄 Image Updates**: Watchtower-style update detection
- **🔔 Updates Counter**: Badge showing pending image updates
- **🌐 Multi-language**: Full Spanish/English support

### Security Scanning (v2.5.0)
- **🛡️ Vulnerability Scanning**: Trivy and Grype integration via ephemeral Docker containers
- **⚡ Real-time Progress**: SSE streaming for live update + scan feedback
- **🚫 Configurable Blocking**: 5 criteria: never / any / critical_high / critical / more_than_current
- **🔓 Admin Override**: Force update despite vulnerabilities, with full audit trail
- **📜 Scan History**: Persistent history of all scans at `/security`
- **🏷️ CVE Badges**: Critical/High counts shown on container cards
- **🔧 Multi-arch Fix**: Correct digest comparison for ARM64 (Raspberry Pi) hosts

## 🏗️ Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                  Docker Compose Stack                         │
│                                                               │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐      │
│  │   Nginx     │  │  Go Backend │  │ SvelteKit Node  │      │
│  │  :3007→:80  │──│  :3006→3001 │  │   (Port 3000)   │      │
│  └──────┬──────┘  └──────┬──────┘  └────────┬────────┘      │
│         │                │                   │               │
│         └─── /api/* ─────┘                   │               │
│         └─── /* ─────────────────────────────┘               │
│                                                               │
│  Volume: backend_data:/data (users, settings persistence)    │
│  Volume: nginx_cache (static asset caching)                  │
└──────────────────────────────────────────────────────────────┘
```

## 🚀 Quick Start

### Prerequisites

- Docker & Docker Compose v2
- For development: Node.js 20+, Go 1.22+

### Deploy with Docker Compose

```bash
# Clone repository
git clone https://github.com/vicolmenares/dockerverse.git
cd dockerverse

# Deploy
docker compose up -d

# Access at http://localhost:3007
# Default credentials: admin / admin123
```

### Development Setup

#### macOS
```bash
chmod +x setup-mac.sh
./setup-mac.sh
```

See [DEVELOPMENT_CONTINUATION_GUIDE.md](./DEVELOPMENT_CONTINUATION_GUIDE.md) for complete setup instructions.

## 🔧 Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | 3001 | Backend API port |
| `JWT_SECRET` | dockerverse-secret | JWT signing key |
| `TZ` | America/Mexico_City | Timezone |

### Ports

| Port | Service |
|------|---------|
| 3007 | Production (Nginx) |
| 3000 | Frontend dev server |
| 3001 | Backend API |

## 📦 Tech Stack

| Component | Technology |
|-----------|------------|
| Backend | Go 1.22, Fiber v2, Docker SDK |
| Frontend | SvelteKit 2.x, Svelte 5, TailwindCSS 3.4 |
| Terminal | xterm.js with search addon |
| Icons | Lucide Svelte |
| Container | Alpine Linux, s6-overlay, Nginx |

## 📋 Version History

### v2.1.0 (February 2026)
- ⏰ Configurable auto-logout timer (5, 10, 15, 30, 60, 120 min)
- 🎨 Log viewer restyled with Databasement-style table layout
- 🖥️ Terminal improvements: Catppuccin Mocha & One Dark Pro themes, WebGL renderer, web-links addon, Ctrl+Scroll zoom
- 📊 Resource leaderboard chart with CPU/Memory/Network/Restarts tabs (top-14 display)
- 🔄 Update indicator badge per container with pulse animation
- 🔔 Pending updates dropdown panel with animated counter
- ✨ Settings sidebar active-state highlighting
- 🔧 Fixed avatar upload API endpoint prefix

### v2.0.1 (February 2026)
- 🐛 Fix TOTP endpoint panics (missing import + wrong Locals key)
- 🐛 Fix AUTH_STORAGE_KEY undefined causing avatar persistence failure
- 🐛 Fix profile save using wrong HTTP method (PUT → PATCH)
- 🐛 Fix version strings (1.0.0 → 2.0.0) in Settings and Login
- 💾 Add backend_data volume for user/settings persistence

### v2.0.0 (February 2026)
- 🔐 TOTP/2FA authentication
- 📈 Resource sparkline charts
- 🔄 Image update detection
- 🎨 Terminal themes (5 options)
- 🔍 Terminal search (Ctrl+F)
- 📋 Advanced log filtering
- ⏰ Auto-logout (30 min)
- 👤 Avatar upload

### v1.0.0 (January 2026)
- Initial release
- Multi-host dashboard
- Container management
- Web terminal
- Log viewer
- JWT authentication

## 📄 Documentation

- [Development Guide](./DEVELOPMENT_CONTINUATION_GUIDE.md) - Complete setup and continuation guide
- [Architecture](./UNIFIED_CONTAINER_ARCHITECTURE.md) - Container architecture details

## 📄 License

MIT License - see [LICENSE](LICENSE) for details.

## 👤 Author

**Victor Heredia**

---

*Built with ❤️ for Docker enthusiasts*
