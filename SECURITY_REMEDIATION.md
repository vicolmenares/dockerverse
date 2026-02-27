# Security Remediation Log — DockerVerse

**Date:** 2026-02-27
**Scope:** Full credentials audit + remediation of `dockerverse` repository
**Repo:** https://github.com/vicolmenares/dockerverse

---

## Executive Summary

A security audit identified hardcoded credentials committed to the public GitHub
repository. All secrets have been removed from source code, docker-compose files,
and the complete git history. The `.env` file is now strictly local-only.

---

## Findings Fixed

| # | Severity | Finding | Status |
|---|----------|---------|--------|
| 1 | 🔴 CRITICAL | SMTP2Go API key hardcoded in `backend/main.go` | ✅ Removed |
| 2 | 🔴 CRITICAL | `.env` committed to git (JWT secret + Watchtower token exposed) | ✅ Removed from history |
| 3 | 🔴 HIGH | JWT secret hardcoded in `docker-compose.yml` | ✅ Removed |
| 4 | 🔴 HIGH | JWT secret hardcoded fallback in `docker-compose.unified.yml` | ✅ Removed |
| 5 | 🟠 HIGH | Admin password `admin123` hardcoded as default in `main.go` | ✅ Removed |
| 6 | 🟡 MEDIUM | Docker daemon URL using port 2375 (no TLS) in `.env` | ✅ Fixed to 2376 |
| 7 | 🟡 MEDIUM | Raspi IPs hardcoded in `docker-compose.unified.yml` | ✅ Moved to `.env` |

---

## Changes Made

### `backend/main.go`

**Before:**
```go
defaultPass   = getEnvOrDefault("ADMIN_PASS", "admin123")
smtp2goAPIKey = getEnvOrDefault("SMTP2GO_API_KEY", "api-0BCAE7D34EA545FE9041EDA3EEF6C8DD")
```

**After:**
```go
// No hardcoded fallback — random secret generated on first run if ADMIN_PASS not set
defaultPass   = getEnvOrDefault("ADMIN_PASS", generateSecret())
// Email disabled if SMTP2GO_API_KEY not set
smtp2goAPIKey = getEnvOrDefault("SMTP2GO_API_KEY", "")
```

Added startup warnings when `ADMIN_PASS` or `SMTP2GO_API_KEY` are not configured.

### `docker-compose.yml`

**Before:**
```yaml
- JWT_SECRET=dockerverse-super-secret-key-2026
- DOCKER_HOSTS=${DOCKER_HOSTS:-raspi1:...:http://192.168.1.146:2375:remote}
```

**After:**
```yaml
# Requires JWT_SECRET to be set in .env — no fallback
- JWT_SECRET=${JWT_SECRET}
- DOCKER_HOSTS=${DOCKER_HOSTS}
```

### `docker-compose.unified.yml`

**Before:**
```yaml
JWT_SECRET: "${JWT_SECRET:-dockerverse-super-secret-key-2026}"
DOCKER_HOSTS: "raspi1:...:tcp://192.168.1.146:2376:remote"
```

**After:**
```yaml
JWT_SECRET: "${JWT_SECRET}"
DOCKER_HOSTS: "${DOCKER_HOSTS}"
```

### `.env` (local file — never committed)

- Rotated `JWT_SECRET` to a strong random 48-byte base64 value (via `openssl rand -base64 48`)
- Rotated `WATCHTOWER_TOKEN` to a strong random 32-byte base64 value
- Fixed `DOCKER_HOSTS` raspi2 URL: `http://...:2375` → `tcp://...:2376` (TLS)
- Added `ADMIN_PASS=` and `SMTP2GO_API_KEY=` (must be filled in)

### `.env.example` (new file — safe to commit)

Created a template file with placeholder values and comments. Committed to git
so other contributors know what variables are required without exposing real values.

---

## Git History Cleanup

The following secrets were present in historical commits and have been
**permanently purged** from the entire git history:

| Secret | Replaced with |
|--------|--------------|
| `api-0BCAE7D34EA545FE9041EDA3EEF6C8DD` (SMTP2Go key) | `***SMTP2GO-KEY-REMOVED***` |
| `dockerverse-super-secret-key-2026` (JWT secret) | `***JWT-SECRET-REMOVED***` |
| `dockerverse-watchtower-2026` (Watchtower token) | `***WATCHTOWER-TOKEN-REMOVED***` |
| `.env` file | Removed from all commits |

**Tool used:** `git-filter-repo` (https://github.com/newren/git-filter-repo)

**Commands run:**
```bash
# Install
pip3 install git-filter-repo

# Purge .env file + replace hardcoded secrets in ALL history
git filter-repo \
  --path .env --invert-paths \
  --replace-text /tmp/secrets-to-purge.txt \
  --force

# Force push all branches
git push origin master --force
git push origin --force --all
```

**Verification after cleanup:**
```bash
# Verified 0 occurrences of each secret in full git history
git log --all -p | grep "api-0BCAE7D34EA545FE9041EDA3EEF6C8DD"  # → 0
git log --all -p | grep "dockerverse-super-secret-key-2026"       # → 0
git log --all -- ".env"                                            # → (no commits)
```

---

## Action Required — Post-Remediation

### Rotate the SMTP2Go API key (CRITICAL)
Even though the key is purged from git history, it was publicly visible in GitHub
before the force push. **You must rotate it immediately:**

1. Go to https://app.smtp2go.com/settings/apikeys
2. Delete or revoke key `api-0BCAE7D34EA545FE9041EDA3EEF6C8DD`
3. Generate a new key
4. Add the new key to your `.env` file: `SMTP2GO_API_KEY=your-new-key`

### Set a strong admin password
Before running the app for the first time with the new config:
```bash
# Add to .env
ADMIN_PASS=your-strong-password-here
```
If the user data store already exists with the old `admin123` password hash,
log in and change it via the Settings → Profile page, or delete
`/data/dockerverse-data.json` to start fresh.

### GitHub Cache Warning
GitHub may cache the old content for a short time after a force push.
The cached content typically expires within 24 hours. GitHub's UI will show
the new (clean) history immediately.

---

## Security Architecture — Going Forward

### Rule: No secrets in source code ever
All secrets must live exclusively in:
- `.env` (local, in `.gitignore`, never committed)
- Server environment variables (set directly on Raspberry Pi)
- A secrets manager (Vault, Doppler, etc.) for production use

### Rule: `.env` stays local
The `.gitignore` already includes `.env`. Never use `git add -f .env`.

### Rule: Rotate exposed secrets immediately
If a secret is ever accidentally committed:
1. Rotate the credential at the service provider first
2. Remove from code
3. Run `git filter-repo` to purge history
4. Force push

### Secrets inventory (local only, in `.env`)
| Variable | Description | Where to get |
|----------|-------------|--------------|
| `JWT_SECRET` | App auth signing key | `openssl rand -base64 48` |
| `ADMIN_PASS` | Dashboard admin password | Choose a strong password |
| `SMTP2GO_API_KEY` | Email sending | smtp2go.com dashboard |
| `WATCHTOWER_TOKEN` | Container update auth | `openssl rand -base64 32` |
| `DOCKER_HOSTS` | Raspi connection strings | Your network config |
