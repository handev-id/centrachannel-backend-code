# CentraChannel API

Multi-tenant REST API built with Go, Fiber, PostgreSQL, and Redis.

## Tech Stack

- **Backend:** Go + Fiber v3
- **Database:** PostgreSQL
- **Cache:** Redis
- **Auth:** JWT (HS256)
- **Password:** bcrypt
- **Migration:** golang-migrate

## Prerequisites

- Go 1.26+
- PostgreSQL
- Redis

## Architecture Overview

```
FB / IG ─────────► Meta Graph API (direct)
WA Business ─────► Evolution API ──► Meta Cloud API
WA (unofficial) ──► Evolution API ──► Baileys (WhatsApp Web)
```

## Quick Start

```bash
cp .env.example .env
make migrate-up
make seed
go run ./cmd/server/main.go
```

## Commands

| Command | Description |
|---------|-------------|
| `make migrate-up` | Apply pending migrations |
| `make migrate-down n=1` | Rollback N migrations |
| `make migrate-drop` | Drop all tables |
| `make seed` | Seed tenant + roles |
| `make reset` | Drop + migrate + seed |
| `make status` | Migration status |

## Project Structure

```text
internal/
├── src/
│   ├── tenant/          # Tenant resolution (domain → Redis → DB)
│   ├── auth/            # Register, login, check-token, logout
│   └── user/            # CRUD users within tenant
├── messenger/
│   ├── messenger.go     # Messenger interface
│   ├── meta.go          # MetaSender — FB/IG direct
│   ├── evolution.go     # EvolutionSender — WA via Evolution API
│   ├── mock.go          # MockSender — fallback
│   └── dispatcher.go    # NewSender factory
├── middleware/
│   ├── tenant_middleware.go  # Domain → Tenant resolution
│   └── auth_middleware.go    # JWT verification
├── di/container.go      # Config, DB, Redis, Logger
└── utils/
    ├── hash/            # bcrypt
    ├── logger/          # Centralized logger
    └── response/        # Standardized JSON responses
```

## Database Schema

```sql
-- tenants: multi-tenant isolation
tenants (id, name, domain UNIQUE, logo, address, phone, email, is_active, settings, timestamps)

-- roles: per-tenant roles (super-admin, admin, agent)
roles (id, tenant_id → tenants, name, UNIQUE(tenant_id, name), timestamps)

-- users: per-tenant users
users (id, tenant_id → tenants, first_name, last_name, username, email, phone,
       password, avatar, last_login, deleted_at, created_at, updated_at,
       UNIQUE(tenant_id, username), UNIQUE(tenant_id, email))

-- role_user: many-to-many pivot
role_user (user_id → users, role_id → roles, UNIQUE(user_id, role_id))

-- auth_access_tokens: token tracking
auth_access_tokens (id, tokenable_id → users, type, name, hash, abilities, ...)
```

## Migration Order

```
01. tenants          ← MUST be first
02. roles            ← REFERENCES tenants(tenant_id)
03. users            ← REFERENCES tenants(tenant_id)
04. role_user        ← pivot
05. auth_access_tokens
```

## API

All endpoints prefixed with `/api`. Tenant resolved from `Host` header automatically. For local dev, use `X-Tenant-Slug` header or add entry in `/etc/hosts`.

### Auth

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/auth/register` | No | Register (default role=agent) |
| POST | `/api/auth/login` | No | Login |
| GET | `/api/auth/check-token` | JWT | Validate token |
| DELETE | `/api/auth/logout` | JWT | Logout |

### Users

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/user` | JWT | Paginated list |
| POST | `/api/user` | JWT | Create user with roles |
| GET | `/api/user/:id` | JWT | User detail |
| PUT | `/api/user/:id` | JWT | Update user + sync roles |
| DELETE | `/api/user/:id` | JWT | Soft delete (protects id=1) |

## Docker

```bash
make docker-build
make docker-run
```

## License

MIT
