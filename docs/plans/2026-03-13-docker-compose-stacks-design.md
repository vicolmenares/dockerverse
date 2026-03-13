# Docker Compose Stacks Management — Design

**Version:** v2.8.0
**Date:** 2026-03-13
**Status:** Approved

## Goal

Add a Stacks page to DockerVerse that allows viewing, editing, and deploying Docker Compose stacks across all configured hosts — similar to Portainer's stacks feature.

## Context

- DockerVerse already has `GET /api/stacks` that groups containers by `com.docker.compose.project` label
- Docker Compose v2 automatically sets `com.docker.compose.project.config_files` and `com.docker.compose.project.working_dir` labels on all managed containers
- DockerVerse already has SSH infrastructure (`runSSHCommand`, `dialSSH`) for remote execution
- Portainer stores its compose files inside its own data volume (`/data/compose/{id}/v{N}/docker-compose.yml`), so paths in labels for Portainer-managed stacks must be translated to the host path

## Architecture

### Stack Types

| Type | `config_files` path pattern | Edit | Delete |
|------|----------------------------|------|--------|
| `portainer` | starts with `/data/compose/` | ✅ | ❌ |
| `dockerverse` | starts with `/home/pi/dockerverse-stacks/` | ✅ | ✅ |
| `external` | any other path | ✅ | ❌ |
| `unknown` | no `config_files` label | ❌ | ❌ |

**Path translation for Portainer stacks:**
`/data/compose/X/vY/docker-compose.yml` → `/var/lib/docker/volumes/portainer_data/_data/compose/X/vY/docker-compose.yml`

### New DockerVerse stacks directory on host

`/home/pi/dockerverse-stacks/{name}/docker-compose.yml`

## Backend API

All endpoints are protected (JWT/API key required).

```
GET    /api/stacks?hostId=X                → list all stacks with metadata
GET    /api/stacks/:name/file?hostId=X     → read compose file via SSH
PUT    /api/stacks/:name/file?hostId=X     → write compose file via SSH
POST   /api/stacks/:name/up?hostId=X       → docker compose up -d via SSH
POST   /api/stacks/:name/down?hostId=X     → docker compose down via SSH
POST   /api/stacks/:name/pull?hostId=X     → docker compose pull via SSH
POST   /api/stacks/create?hostId=X         → create new stack (mkdir + write + up)
DELETE /api/stacks/:name?hostId=X          → down + rm directory (DockerVerse stacks only)
```

### Stack response shape

```json
{
  "name": "adguardhome",
  "type": "portainer",
  "hasFile": true,
  "configFilePath": "/var/lib/docker/volumes/portainer_data/_data/compose/1/v14/docker-compose.yml",
  "workingDir": "/var/lib/docker/volumes/portainer_data/_data/compose/1/v14",
  "status": "running",
  "services": [
    { "id": "abc123", "name": "adguardhome", "state": "running", "service": "adguardhome" }
  ]
}
```

### Compose commands via SSH

```bash
# Up
docker compose -f <configFilePath> -p <name> up -d

# Down
docker compose -f <configFilePath> -p <name> down

# Pull
docker compose -f <configFilePath> -p <name> pull

# Create new stack
mkdir -p /home/pi/dockerverse-stacks/<name>
cat > /home/pi/dockerverse-stacks/<name>/docker-compose.yml << 'EOF'
<content>
EOF
docker compose -f /home/pi/dockerverse-stacks/<name>/docker-compose.yml -p <name> up -d
```

## Frontend

### New route

`frontend/src/routes/stacks/+page.svelte`

### Sidebar entry

New item in sidebar between Dashboard and Settings (admin-only visibility).

### Page layout

```
┌─────────────────────────────────────────────────────┐
│ Stacks          [Host selector ▼]    [+ New Stack]  │
├─────────────────────────────────────────────────────┤
│ ▼ adguardhome   ● 1/1 running  [Portainer]          │
│   adguardhome   running                              │
│   [Edit] [↑ Up] [↓ Down] [↻ Pull & Redeploy]        │
├─────────────────────────────────────────────────────┤
│ ▼ mi-stack      ● 2/2 running  [DockerVerse]        │
│   servicio-a    running                              │
│   servicio-b    running                              │
│   [Edit] [↑ Up] [↓ Down] [↻ Pull & Redeploy] [🗑]   │
└─────────────────────────────────────────────────────┘
```

### Edit modal

- Monospace textarea with the compose file content
- **Save** button → PUT file only
- **Save & Deploy** button → PUT file + POST up
- Shows stderr output on failure

### New Stack modal

- Stack name input
- Host selector
- Compose content textarea
- **Deploy** button → POST /api/stacks/create

### Type badges

- `Portainer` — gray
- `DockerVerse` — blue
- `External` — yellow
- Delete button only shown for `DockerVerse` type

## Error handling

| Scenario | Behavior |
|----------|----------|
| SSH unavailable | Stack shown read-only, action buttons disabled with tooltip |
| File not accessible | Editor shows "File not accessible" message |
| `docker compose up` fails | Modal shows stderr from command |
| Duplicate stack name on create | Validated before mkdir, returns 409 |
| Host not found | 404 with error message |

## Out of scope

- YAML validation / linting
- Environment variables management (separate from compose file)
- Git-based stacks
- Stack templates
- Logs streaming per stack (containers already have individual log viewer)
