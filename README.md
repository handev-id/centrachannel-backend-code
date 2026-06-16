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

## Architecture

Two access layers based on domain:

| Domain | Tenant Context | Auth |
|--------|----------------|------|
| `centrachannel.com/api` | No (resolved from payload) | Optional (API key) |
| `{tenant}.centrachannel.com/api` | Yes (from Host header) | JWT + Role |

### Main Domain (`internal/app`)

```
centrachannel.com/api/*
  ├── Observability:  /health, /ping, /version
  ├── Documentation:  /docs (Redoc), /swagger (Swagger UI)
  ├── Webhook:        /webhook/evolution, /webhook/meta
  └── Registration:   POST /api/tenants/onboard
```

### Subdomain (`internal/src`)

```
{tenant}.centrachannel.com/api/*
  ├── TenantMiddleware  → resolve tenant from domain
  ├── AuthMiddleware    → JWT verification
  ├── RoleMiddleware    → super-admin / admin / agent
  └── Business features → auth, campaign, contact, conversation, etc.
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
├── app/                     # Main domain (before TenantMiddleware)
│   ├── docs/                #   Redoc + Swagger UI
│   ├── observability/       #   /health, /ping, /version
│   ├── registration/        #   POST /api/tenants/onboard
│   └── webhook/             #   /webhook/evolution, /webhook/meta
├── src/                     # Subdomain (after TenantMiddleware + auth)
│   ├── tenant/              #   Tenant CRUD (admin only)
│   ├── auth/                #   Register, login, check-token, logout
│   ├── user/                #   CRUD users within tenant
│   ├── campaign/            #   Campaigns, templates, recipient lists
│   ├── channel/             #   Channel listing
│   ├── contact/             #   Contact CRUD, merge, import/export
│   ├── conversation/        #   Conversations, assign, resolve
│   ├── message/             #   Send/receive messages
│   ├── tag/                 #   Tag CRUD
│   ├── note/                #   Notes on conversations
│   ├── dashboard/           #   Stats & charts
│   ├── upload/              #   File upload
│   └── whatsapp_device/     #   WhatsApp device management
├── messenger/               # Outbound message senders
├── middleware/               # Tenant, Auth, CORS, Logger
├── ws/                      # SSE broker & handler (real-time events)
├── di/                      # Dependency injection container
└── utils/                   # hash, logger, response helpers
```

## API

### Main Domain (`centrachannel.com/api`)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/tenants/onboard` | Register new tenant + admin user |

### Subdomain (`{tenant}.centrachannel.com/api`)

Tenant resolved from `Host` header automatically. For local dev, use `X-Tenant-Domain` header.

#### Auth

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/auth/register` | No | Register (default role=agent) |
| POST | `/api/auth/login` | No | Login |
| GET | `/api/auth/check-token` | JWT | Validate token |
| DELETE | `/api/auth/logout` | JWT | Logout |

#### Users

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
