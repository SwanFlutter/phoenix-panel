# 🔥 PHOENIX PANEL

A multi-protocol proxy subscription management panel. Manage users, nodes, and
inbounds across **Xray-core** and **sing-box** from one API, and issue
per-user subscription links (VLESS, VMess, Trojan, Shadowsocks, Hysteria2, TUIC).

> **Status: foundation.** This repository currently contains the production-grade
> backend foundation — config, database, auth, security middleware, the proxy-core
> abstraction, link/subscription generation, and the REST API. The proxy-core
> adapters (`internal/core/xray.go`, `internal/core/singbox.go`) are wired as
> stubs at the gRPC/HTTP boundary, and the React frontend + OpenAPI spec are the
> next phases. See [Roadmap](#roadmap).

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

## Quick start

### With Docker (recommended — no Go toolchain needed)

```bash
cp .env.example .env
# REQUIRED: set a strong secret and admin password
#   PHOENIX_JWT_SECRET   (>= 32 chars; generate: openssl rand -hex 32)
#   PHOENIX_ADMIN_PASSWORD
docker compose up -d --build
```

The panel comes up on `http://localhost:8080`. On first boot it creates a sudo
admin from `PHOENIX_ADMIN_USERNAME` / `PHOENIX_ADMIN_PASSWORD` and a `local` node.

### Locally (requires Go ≥ 1.22)

```bash
cp .env.example .env   # edit secrets
go mod tidy
make run
```

## Configuration

All configuration is via environment variables (see `.env.example` for the full
list). Highlights:

| Variable | Default | Notes |
|---|---|---|
| `PHOENIX_DB_DRIVER` | `sqlite` | `sqlite` or `postgres` |
| `PHOENIX_JWT_SECRET` | — | **required**, ≥ 32 chars |
| `PHOENIX_ADMIN_PASSWORD` | — | required on first boot |
| `PHOENIX_RATE_RPS` | `20` | per-IP API rate limit |
| `PHOENIX_LOGIN_RATE_RPS` | `1` | stricter limit on `/login` |
| `PHOENIX_DEFAULT_CORE` | `xray` | `xray` or `sing-box` |

## API overview

| Method | Path | Auth | Description |
|---|---|---|---|
| `POST` | `/api/admin/login` | — | Obtain a JWT |
| `GET`  | `/api/admin/me` | Bearer | Current admin |
| `POST` | `/api/admin/change-password` | Bearer | Rotate password |
| `GET`/`POST` | `/api/admin/users` | Bearer | List / create users |
| `GET`/`PATCH`/`DELETE` | `/api/admin/users/:id` | Bearer | Manage a user |
| `POST` | `/api/admin/users/:id/reset` | Bearer | Reset traffic |
| `POST` | `/api/admin/users/:id/regenerate-sub` | Bearer | Rotate sub token |
| `GET`/`POST` | `/api/admin/nodes` | Bearer (sudo to mutate) | Nodes |
| `POST`/`DELETE` | `/api/admin/inbounds[/:id]` | Bearer (sudo) | Inbounds |
| `GET` | `/sub/:token` | token | Public subscription document |
| `GET` | `/healthz`, `/readyz` | — | Health checks |

### Example

```bash
# Login
TOKEN=$(curl -s localhost:8080/api/admin/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"<your-pw>"}' | jq -r .token)

# Create a user with a 50 GiB cap
curl -s localhost:8080/api/admin/users \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"username":"alice","data_limit":53687091200}' | jq

# Fetch the subscription (base64 of the share links)
curl -s localhost:8080/sub/<sub_token>
```

## Security

- **Passwords:** Argon2id (64 MiB, t=3, p=2), constant-time verification.
- **Tokens:** HS256 JWT with alg pinning; unguessable 24-byte subscription tokens.
- **Brute force:** dedicated strict rate limit on login; generic error + timing
  equalization to prevent username enumeration.
- **Transport hardening:** CSP, HSTS, `X-Frame-Options`, `nosniff`.
- **Audit trail:** every privileged action is recorded in `audit_logs`.
- **Least privilege:** sudo vs admin roles; node/inbound mutation is sudo-only.
- Follows OWASP Top 10 guidance throughout.

## Roadmap

- [ ] Real Xray gRPC + sing-box management wiring in `internal/core`
- [ ] Traffic-collection scheduler + per-user usage reconciliation
- [ ] React + TypeScript admin dashboard & user panel (`/web`)
- [ ] OpenAPI 3.1 spec + generated docs
- [ ] Installer script, CI/CD pipeline
- [ ] Admin & user guides

## License

To be determined by the project owner.
