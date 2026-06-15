# 🔥 PHOENIX PANEL

A multi-protocol proxy subscription management panel. Manage users, nodes, and
inbounds across **Xray-core** and **sing-box** from one API, and issue
per-user subscription links (VLESS, VMess, Trojan, Shadowsocks, Hysteria2, TUIC).

**Repository:** [github.com/SwanFlutter/phoenix-panel](https://github.com/SwanFlutter/phoenix-panel)

> **Status: foundation.** This repository currently contains the production-grade
> backend foundation — config, database, auth, security middleware, the proxy-core
> abstraction, link/subscription generation, and the REST API. The proxy-core
> adapters (`internal/core/xray.go`, `internal/core/singbox.go`) are wired as
> stubs at the gRPC/HTTP boundary, and the React frontend + OpenAPI spec are the
> next phases. See [Roadmap](#roadmap).

---

## Table of Contents

- [Architecture](#architecture)
- [Installation](#installation)
  - [Prerequisites](#prerequisites)
  - [Method 1: Docker (Recommended)](#method-1-docker-recommended)
  - [Method 2: Native Installation](#method-2-native-installation)
  - [Method 3: GitHub Clone](#method-3-github-clone)
  - [Automated Installation Script](#automated-installation-script)
- [Configuration](#configuration)
- [API Overview](#api-overview)
- [Security](#security)
- [Troubleshooting](#troubleshooting)
- [Roadmap](#roadmap)
- [License](#license)

---

## Architecture

```
cmd/phoenix            — server entrypoint (config → db → router → http)
internal/
  config              — env-driven configuration + validation
  models              — GORM models + domain types (User, Node, Inbound, …)
  database            — connection (SQLite/Postgres), migrate, seed
  security            — argon2id passwords, JWT, token/uuid generation
  middleware          — auth, rate limiting, CORS, security headers, logging
  audit               — security audit logging
  core                — ProxyCore interface + Xray/sing-box adapters
  links               — share-URL + subscription document generation
  service             — business logic (users, auth, nodes, subscriptions)
  api                 — Gin handlers, DTOs, router
migrations            — canonical SQL schema (Postgres dialect)
```

The layering is strict: `api` → `service` → `models`/`database`. Handlers never
touch the DB directly; services own all invariants and transactions.

---

## Installation

### Prerequisites

Choose your installation method based on your needs:

| Method | Requirements | Difficulty | Speed |
|--------|--------------|-----------|-------|
| **Docker** | Docker + Docker Compose | Easy | Fast |
| **Native** | Linux/Ubuntu, Go 1.22+, systemd | Medium | Normal |
| **GitHub** | Git, bash | Easy | Flexible |

#### System Requirements

- **OS:** Linux (Ubuntu 20.04+ recommended)
- **Memory:** Minimum 512 MB RAM
- **Storage:** Minimum 2 GB free disk space
- **Network:** Port 8080 (configurable via `.env`)

---

### Method 1: Docker (Recommended)

Docker is the easiest way to get started. No compilation needed.

#### Step 1: Clone the Repository

```bash
git clone https://github.com/SwanFlutter/phoenix-panel.git
cd phoenix-panel
```

**Repository:** [SwanFlutter/phoenix-panel](https://github.com/SwanFlutter/phoenix-panel)

#### Step 2: Configure Environment

```bash
cp .env.example .env
```

Edit `.env` with your settings:

```bash
# Required: set a strong JWT secret (32+ characters)
PHOENIX_JWT_SECRET=$(openssl rand -hex 32)

# Required: set admin password
PHOENIX_ADMIN_PASSWORD=your-strong-password

# Optional: set public base URL for subscriptions
PHOENIX_BASE_URL=https://panel.your-domain.com

# Optional: choose database (sqlite or postgres)
PHOENIX_DB_DRIVER=sqlite
```

#### Step 3: Start with Docker Compose

```bash
docker compose up -d --build
```

#### Step 4: Verify Installation

```bash
# Check if container is running
docker ps | grep phoenix-panel

# Check logs
docker compose logs -f panel

# Test health endpoint
curl http://localhost:8080/healthz
```

The panel will be available at: **`http://localhost:8080`**

#### Stop the Service

```bash
docker compose down
```

---

### Method 2: Native Installation

For production deployments or when Docker is not available.

#### Step 1: Install Go

Requires **Go 1.22 or higher**:

```bash
# Check if Go is installed
go version

# If not installed, download from: https://go.dev/dl
```

#### Step 2: Clone the Repository

```bash
git clone https://github.com/SwanFlutter/phoenix-panel.git
cd phoenix-panel
```

#### Step 3: Configure Environment

```bash
cp .env.example .env

# Edit the file with your settings
nano .env
```

#### Step 4: Build the Binary

```bash
make build
# Binary will be created at: ./bin/phoenix
```

#### Step 5: Run Locally (Development)

For testing purposes:

```bash
make run
# Reads configuration from .env
# Panel starts on http://localhost:8080
```

#### Step 6: Install as Systemd Service (Production)

```bash
sudo make install
# or manually:
sudo mkdir -p /opt/phoenix
sudo cp bin/phoenix /opt/phoenix/
sudo cp .env /opt/phoenix/
sudo useradd --system --no-create-home phoenix || true
sudo chown -R phoenix:phoenix /opt/phoenix
```

Create systemd service file at `/etc/systemd/system/phoenix-panel.service`:

```ini
[Unit]
Description=PHOENIX PANEL
After=network.target postgresql.service

[Service]
Type=simple
User=phoenix
Group=phoenix
WorkingDirectory=/opt/phoenix
EnvironmentFile=/opt/phoenix/.env
ExecStart=/opt/phoenix/phoenix
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable phoenix-panel
sudo systemctl start phoenix-panel
sudo systemctl status phoenix-panel
```

View logs:

```bash
sudo journalctl -u phoenix-panel -f
```

---

### Method 3: GitHub Clone

Use the provided installation script for automated setup from GitHub.

#### Quick Install via Script

```bash
# Download and run installation script
bash <(curl -fsSL https://raw.githubusercontent.com/SwanFlutter/phoenix-panel/main/install.sh) docker
```

#### Using install.sh Locally

```bash
# Clone the repository
git clone https://github.com/SwanFlutter/phoenix-panel.git
cd phoenix-panel

# Make script executable
chmod +x install.sh

# Run installation (choose one):
sudo ./install.sh docker    # Install with Docker Compose
sudo ./install.sh native    # Install as native systemd service
sudo ./install.sh github    # Just clone the repository
```

**Script Location:** [install.sh](https://github.com/SwanFlutter/phoenix-panel/blob/main/install.sh)

---

### Automated Installation Script

The `install.sh` script automates the entire installation process:

#### Features

- ✅ Automatic Git installation
- ✅ Automatic repository cloning/updating from GitHub
- ✅ Docker engine installation (if using docker method)
- ✅ Go installation (if using native method)
- ✅ Automatic environment configuration
- ✅ Systemd service setup
- ✅ Health checks and verification

#### Usage

```bash
# Docker installation (recommended)
sudo ./install.sh docker

# Native installation with systemd
sudo ./install.sh native

# Just clone from GitHub
sudo ./install.sh github
```

#### What the Script Does

1. Checks for root privileges
2. Installs Git if needed
3. Clones/updates repository from [GitHub](https://github.com/SwanFlutter/phoenix-panel)
4. Sets up required dependencies
5. Generates secure secrets
6. Configures environment
7. Starts services
8. Performs health checks

---

## Configuration

All configuration is via environment variables. See `.env.example` for the full list.

### Essential Configuration

| Variable | Default | Required | Description |
|---|---|---|---|
| `PHOENIX_JWT_SECRET` | — | ✅ | Secret for JWT signing (≥32 chars). Generate: `openssl rand -hex 32` |
| `PHOENIX_ADMIN_PASSWORD` | — | ✅ | Admin password (set on first boot) |
| `PHOENIX_BASE_URL` | `http://localhost:8080` | ⚠️ | Public URL for subscription links |
| `PHOENIX_DB_DRIVER` | `sqlite` | — | `sqlite` or `postgres` |
| `PHOENIX_PORT` | `8080` | — | Listen port |
| `PHOENIX_MODE` | `release` | — | `debug` or `release` |

### Database Configuration

#### SQLite (Default)

```bash
PHOENIX_DB_DRIVER=sqlite
PHOENIX_DB_SQLITE_PATH=./data/phoenix.db
```

#### PostgreSQL

```bash
PHOENIX_DB_DRIVER=postgres
PHOENIX_DB_HOST=localhost
PHOENIX_DB_PORT=5432
PHOENIX_DB_USER=phoenix
PHOENIX_DB_PASSWORD=secure-password
PHOENIX_DB_NAME=phoenix
```

### Security Configuration

```bash
# JWT Secret (generate with: openssl rand -hex 32)
PHOENIX_JWT_SECRET=your-32-char-secret-here

# JWT Token lifetime
PHOENIX_JWT_TTL=24h

# Rate limiting
PHOENIX_RATE_RPS=20          # API requests per second per IP
PHOENIX_LOGIN_RATE_RPS=1     # Login attempts per second per IP

# CORS origins
PHOENIX_CORS_ORIGINS=*       # or specific domains comma-separated
```

### Proxy Core Configuration

```bash
# Default core for new inbounds
PHOENIX_DEFAULT_CORE=xray    # or sing-box
```

---

## API Overview

The API requires JWT authentication for admin endpoints. Public endpoints (health, subscriptions) are accessible without authentication.

### Authentication Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/api/admin/login` | — | Obtain a JWT token |
| `GET` | `/api/admin/me` | Bearer | Get current admin info |
| `POST` | `/api/admin/change-password` | Bearer | Change admin password |

### User Management

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/api/admin/users` | Bearer | List all users |
| `POST` | `/api/admin/users` | Bearer | Create a new user |
| `GET` | `/api/admin/users/:id` | Bearer | Get user details |
| `PATCH` | `/api/admin/users/:id` | Bearer | Update user |
| `DELETE` | `/api/admin/users/:id` | Bearer | Delete user |
| `POST` | `/api/admin/users/:id/reset` | Bearer | Reset user traffic |
| `POST` | `/api/admin/users/:id/regenerate-sub` | Bearer | Regenerate subscription token |

### Node & Inbound Management

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/api/admin/nodes` | Bearer | List all nodes |
| `POST` | `/api/admin/nodes` | Bearer (sudo) | Create a node |
| `GET`/`POST` | `/api/admin/inbounds` | Bearer (sudo) | Manage inbounds |

### Public Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/sub/:token` | token | Get subscription document |
| `GET` | `/healthz` | — | Health check |
| `GET` | `/readyz` | — | Readiness check |

### Example Usage

#### Login

```bash
TOKEN=$(curl -s http://localhost:8080/api/admin/login \
  -H 'Content-Type: application/json' \
  -d '{
    "username": "admin",
    "password": "your-admin-password"
  }' | jq -r .token)

echo "Token: $TOKEN"
```

#### Create a User

```bash
curl -s http://localhost:8080/api/admin/users \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "username": "alice",
    "data_limit": 53687091200
  }' | jq
```

#### Get Subscription

```bash
curl -s http://localhost:8080/sub/YOUR_SUB_TOKEN
```

---

## Security

PHOENIX PANEL implements security best practices:

- **Passwords:** Argon2id hashing (64 MiB, t=3, p=2), constant-time verification
- **Tokens:** HS256 JWT with algorithm pinning; 24-byte unguessable subscription tokens
- **Brute Force Protection:** Dedicated strict rate limit on `/api/admin/login`
- **Transport Security:** CSP, HSTS, X-Frame-Options, nosniff headers
- **Audit Trail:** All privileged actions logged in `audit_logs`
- **Least Privilege:** Sudo vs admin roles; node/inbound mutations require sudo
- **OWASP Compliance:** Follows OWASP Top 10 guidance throughout

---

## Troubleshooting

### Container won't start (Docker)

```bash
# Check logs
docker compose logs panel

# Common issues:
# 1. Port already in use
#    Solution: Change PHOENIX_PORT in .env

# 2. Missing .env file
#    Solution: cp .env.example .env && edit .env

# 3. Permission denied
#    Solution: sudo docker compose up -d
```

### Panel not accessible

```bash
# Check if service is running
docker ps | grep phoenix-panel

# Check if port is listening
netstat -tlnp | grep 8080

# Test health endpoint
curl http://localhost:8080/healthz

# If unreachable from another machine:
# Edit .env: PHOENIX_BASE_URL=http://your-server-ip:8080
```

### Database errors

```bash
# For SQLite: check file permissions
ls -la ./data/phoenix.db

# For PostgreSQL: verify connection
psql -h localhost -U phoenix -d phoenix

# Reset database (CAUTION: deletes all data)
rm ./data/phoenix.db  # SQLite only
```

### Can't login to admin panel

```bash
# Verify admin password is set in .env
grep PHOENIX_ADMIN_PASSWORD .env

# Check application logs
docker compose logs panel | grep -i login

# The admin user is created on first boot
# if no admins exist
```

### High CPU/Memory usage

```bash
# Check Go goroutines
curl http://localhost:8080/debug/pprof/goroutine | head -5

# Check memory
curl http://localhost:8080/debug/pprof/heap | head -5

# Monitor container resources
docker stats phoenix-panel
```

---

## Development

### Local Development

```bash
# Install dependencies
make tidy

# Run tests
make test

# Run linter
make lint

# Format code
make fmt

# Run locally
make run
```

### Project Structure

- `cmd/phoenix/` — Main application entry point
- `internal/` — Core business logic and services
- `migrations/` — Database migration files
- `docker-compose.yml` — Docker Compose configuration
- `Dockerfile` — Docker image definition
- `install.sh` — Automated installation script

### Makefile Targets

```bash
make help        # Show all available commands
make build       # Build binary
make run         # Run locally
make test        # Run tests
make vet         # Run go vet
make fmt         # Format code
make lint        # Run linter
make docker-build # Build Docker image
make up          # Start docker compose
make down        # Stop docker compose
make logs        # View container logs
```

---

## Roadmap

- [ ] Real Xray gRPC + sing-box management wiring in `internal/core`
- [ ] Traffic-collection scheduler + per-user usage reconciliation
- [ ] React + TypeScript admin dashboard & user panel (`/web`)
- [ ] OpenAPI 3.1 spec + generated docs
- [ ] Enhanced installer script with CI/CD pipeline
- [ ] Admin & user guides
- [ ] Kubernetes deployment manifests
- [ ] Helm charts

---

## Repository Links

- **Main Repository:** [github.com/SwanFlutter/phoenix-panel](https://github.com/SwanFlutter/phoenix-panel)
- **Installation Script:** [install.sh on GitHub](https://github.com/SwanFlutter/phoenix-panel/blob/main/install.sh)
- **Docker Image:** [Docker Compose Config](https://github.com/SwanFlutter/phoenix-panel/blob/main/docker-compose.yml)
- **Issues & Bug Reports:** [GitHub Issues](https://github.com/SwanFlutter/phoenix-panel/issues)
- **Discussions:** [GitHub Discussions](https://github.com/SwanFlutter/phoenix-panel/discussions)

---

## License

To be determined by the project owner.

---

**Last Updated:** 2026-06-15  
**Repository:** [SwanFlutter/phoenix-panel on GitHub](https://github.com/SwanFlutter/phoenix-panel)
