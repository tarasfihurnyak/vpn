# Corporate VPN — Control Plane

[![CI](https://github.com/tarasfihurnyak/vpn/actions/workflows/ci.yml/badge.svg)](https://github.com/tarasfihurnyak/vpn/actions/workflows/ci.yml)

> **Status: MVP / Work in Progress.** The control-plane REST API is functional. The WireGuard data-plane server and the CLI client are not yet implemented.

A corporate VPN management system built in Go. The project is split into two logical parts:

| Part | Status |
| --- | --- |
| **Control Plane** — REST admin API (this repo) | ✅ MVP done |
| **Data Plane** — WireGuard server integration | 🔲 Planned |
| **CLI VPN Client** | 🔲 Planned |

---

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Tech Stack](#tech-stack)
- [Features](#features)
- [Prerequisites](#prerequisites)
- [Getting Started](#getting-started)
- [Configuration](#configuration)
- [API Reference](#api-reference)
- [Development](#development)
- [Roadmap](#roadmap)
- [Security](#security)

---

## Overview

The **control plane** is a REST API that acts as an admin panel for the VPN infrastructure. Administrators use it to manage users and WireGuard peers. Each peer record stores a name, public key, assigned IP address, and an enabled flag. The `allowed_ips` table defines which CIDRs are routed through the tunnel for each user. The `server_config` table holds the WireGuard server's public key, listen port, IP pool, and DNS — this is returned to the CLI client as part of the tunnel config so it can bring up its end of the tunnel. Once the data plane is implemented, peer configs will be pushed to the WireGuard server.

---

## Architecture

```
                         ┌──────────────────────────────────┐
  Admin / Browser ──────►│  Traefik (TLS termination, :443)  │
                         └────────────────┬─────────────────┘
                                          │ HTTP (internal)
                         ┌────────────────▼─────────────────┐
                         │   Control Plane API  (:8080)      │
                         │  ┌──────────┐  ┌──────────────┐  │
                         │  │  /auth   │  │ /users /peers│  │
                         │  └──────────┘  └──────────────┘  │
                         └────────────────┬─────────────────┘
                                          │
                         ┌────────────────▼─────────────────┐
                         │        PostgreSQL 16              │
                         └──────────────────────────────────┘

  [Planned]
  CLI VPN Client ──gRPC+TLS──────────► Control Plane (auth + tunnel config)
  Control Plane ──gRPC+TLS───────────► Data Plane (peer sync)
  CLI VPN Client ──WireGuard (UDP)───► Data Plane (VPN tunnel)
```

---

## Tech Stack

| Layer | Technology |
| --- | --- |
| Language | Go 1.26+ |
| HTTP Router | [chi v5](https://github.com/go-chi/chi) |
| Database | PostgreSQL 16 |
| DB Driver | [pgx v5](https://github.com/jackc/pgx) |
| Query Generation | [sqlc](https://sqlc.dev) |
| Migrations | [sql-migrate](https://github.com/rubenv/sql-migrate) |
| Auth | JWT ES256 (ECDSA P-256) via [golang-jwt](https://github.com/golang-jwt/jwt) |
| Password Hashing | bcrypt |
| Logging | [zerolog](https://github.com/rs/zerolog) |
| Rate Limiting | [golang.org/x/time/rate](https://pkg.go.dev/golang.org/x/time/rate) (per-IP token bucket) |
| Reverse Proxy | Traefik v3 |
| TLS (local dev) | [mkcert](https://github.com/FiloSottile/mkcert) |
| Testing | testify + [testcontainers-go](https://golang.testcontainers.org) |

---

## Features

### Implemented

- **User management** — create users, look up by ID; passwords stored as bcrypt hashes
- **Peer management** — register WireGuard peers (name, public key, assigned IP address, enabled flag) per user; per-user `allowed_ips` CIDRs define which traffic is routed through the tunnel
- **Authentication**
  - Login with username or email
  - JWT access tokens (ES256, short-lived, default 15 min)
  - Refresh token rotation with **replay-attack detection** — if a revoked token is reused, all user sessions are immediately invalidated
  - `HttpOnly` + `Secure` cookies for browser-based flows; `Authorization: Bearer` header for non-browser clients
- **Security middleware**
  - Per-IP rate limiting on `POST /api/auth/login` (5 req/min sustained, burst 3)
  - Origin header CSRF check on all cookie-based endpoints (`/refresh`, `/logout`)
  - Request body size cap (1 MB)
  - Panic recovery
- **Database**
  - Migrations run automatically on startup
  - Type-safe queries generated via sqlc
- **Infrastructure**
  - Multi-stage Docker build producing a lean Alpine image
  - Docker Compose stack: API + PostgreSQL + Traefik
  - HTTP → HTTPS redirect enforced at the Traefik level
  - `make tls-setup` generates locally trusted TLS certificates via mkcert
  - Structured JSON logging

### Admin Panel (REST API)

The REST API **is** the admin panel. There is no separate web UI — administrators interact with it over HTTP. All management operations (user creation, peer registration, token lifecycle) go through this API. A Swagger/OpenAPI specification is planned (see [Roadmap](#roadmap)).

---

## Prerequisites

- [Go 1.26+](https://go.dev/dl/)
- [Docker](https://docs.docker.com/get-docker/) & Docker Compose
- [mkcert](https://github.com/FiloSottile/mkcert) (for local TLS)

---

## Getting Started

### 1. Generate the JWT signing key

```bash
make gen-jwt-key
# Outputs: certs/jwt_private.pem  (ES256 / ECDSA P-256)
```

### 2. Create the environment file

```bash
cp .env.example .env
# Set DB_USER, DB_PASSWORD, DB_NAME, and JWT_PRIVATE_KEY_FILE at minimum
# JWT_PRIVATE_KEY_FILE=certs/jwt_private.pem  (already set in .env.example)
```

See [Configuration](#configuration) for all variables.

### 3. Trust the TLS certificate (first time only)

The stack uses mkcert to issue a certificate for `localhost`. Each machine that will send HTTPS requests to the API needs to trust the mkcert root CA.

```bash
# Install the mkcert root CA into your system / browser trust store:
mkcert -install

# On other client machines (no mkcert installed):
# Manually add certs/rootCA.pem to the OS trusted certificate store.
```

### 4. Start the stack

```bash
make up
# Equivalent to: docker compose up --build -d
# Certificates are generated automatically if they do not exist.
```

The API is now available at `https://localhost/api`.

### 5. Health check

```bash
curl https://localhost/health
# HTTP 200 OK
```

---

## Configuration

All configuration is read from environment variables or a `.env` file in the project root.

| Variable | Required | Default | Description |
| --- | :---: | --- | --- |
| `DB_USER` | ✅ | — | PostgreSQL username |
| `DB_PASSWORD` | ✅ | — | PostgreSQL password |
| `DB_NAME` | ✅ | — | PostgreSQL database name |
| `DB_HOST` | | `localhost` | PostgreSQL host |
| `DB_PORT` | | `5432` | PostgreSQL port |
| `DB_SSLMODE` | | `disable` | PostgreSQL SSL mode |
| `JWT_PRIVATE_KEY_FILE` | ✅ | — | Path to the ES256 PEM private key (`certs/jwt_private.pem`) |
| `JWT_ACCESS_TTL` | | `15m` | Access token lifetime |
| `JWT_REFRESH_TTL` | | `168h` | Refresh token lifetime (7 days) |
| `JWT_SECURE_COOKIE` | | `true` | Attach `Secure` flag to cookies |
| `JWT_ALLOWED_ORIGINS` | | `http://localhost` | Comma-separated CSRF-allowed origins |

---

## API Reference

> A Swagger/OpenAPI spec is planned. The table below covers the current endpoints.

Protected routes require `Authorization: Bearer <access_token>`.

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| `GET` | `/health` | — | Health check |
| `POST` | `/api/auth/login` | — | Authenticate; returns access token + sets refresh cookie |
| `POST` | `/api/auth/refresh` | Cookie | Rotate refresh token; returns a new token pair |
| `POST` | `/api/auth/logout` | Cookie | Revoke the current refresh token |
| `POST` | `/api/users` | JWT | Create a user |
| `GET` | `/api/users/{id}` | JWT | Get user by UUID |
| `POST` | `/api/peers` | JWT | Register a WireGuard peer (name, public_key, ip_address) |
| `GET` | `/api/peers/{id}` | JWT | Get peer by UUID |

---

## Development

### Run tests

```bash
make test            # all tests (uses testcontainers — requires Docker)
make test-migration  # migration tests only
```

Tests run automatically on every push and pull request via GitHub Actions (format check → lint → tests).

### Lint

```bash
make lint            # runs golangci-lint via Docker — no local install required
```

### Regenerate SQL queries (sqlc)

```bash
make sqlc
```

### Database migrations

```bash
# Create a new migration file:
make migrate-new NAME=add_some_table

# Apply all pending migrations:
make migrate-up

# Roll back the last migration:
make migrate-down
```

---

## Roadmap

### Control Plane

- [ ] Swagger / OpenAPI documentation
- [ ] Two-factor authentication — TOTP (2FA)
- [ ] Multi-factor authentication — Google OIDC with domain restriction
- [ ] Role-based access control (admin vs. regular user)
- [ ] Peer enable / disable endpoint
- [ ] Client tunnel config endpoint — returns server public key, endpoint, peer IP, and user `allowed_ips` CIDRs for WireGuard config generation

### Data Plane (separate component — not yet started)

- [ ] WireGuard interface management (own and manage `wg0`)
- [ ] Register server public key, listen port, IP pool, and DNS with the control plane on startup
- [ ] gRPC (TLS) interface to receive peer config pushes from the control plane
- [ ] Apply incoming peer changes to the kernel interface in real time

### CLI VPN Client (separate tool — not yet started)

- [ ] Generate a local WireGuard key pair and register the public key with the control plane
- [ ] Authentication over gRPC (TLS)
- [ ] Receive WireGuard tunnel config from the control plane on successful auth
- [ ] Apply config and bring up the tunnel via `wg`/`wg-quick`

---

## Security

Security considerations, threat model, and hardening notes are documented in [SECURITY.md](SECURITY.md).

---

## License

This project is for personal/educational use. No license is currently specified.
