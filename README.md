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
make run
```

## Commands

| Command | Description |
|---------|-------------|
| `make help` | List all available commands |
| `make migrate` / `make migrate-up` | Apply pending migrations |
| `make rollback n=1` / `make migrate-down n=1` | Rollback N migrations |
| `make migrate-drop` | Drop all tables |
| `make seed` | Run all seed files through `database/seed.sql` in one transaction |
| `make reset` | Drop + migrate + seed |
| `make status` | Current migration version and pending migration files |
| `make run` | Run the backend |
| `make test` / `make test-unit` / `make coverage` | Run all unit tests without cached results, with verbose output and coverage |
| `make test-all` | Run unit and integration tests without cached results, with coverage |
| `make docker-build` | Build the image configured by `IMAGE_NAME` and `IMAGE_TAG` |
| `make docker-run` | Run the image with host networking and the configured environment |

Commands load `.env` using the project's `godotenv` parser. Existing environment variables and Make command-line overrides take precedence. Use `ENV_FILE` to select another file:

```bash
make migrate-up ENV_FILE=.env.local
make rollback n=2
make test
```

Database commands use `DB_URL` when set; otherwise they build it from `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, and `DB_SSL_MODE`. Keep `DB_URL` consistent with those fields when both are set. Optional overrides in `.env` include `MIGRATIONS_PATH` and `SEED_FILE`.

`make test` covers every application package and excludes `test/integration`. It uses `-count=1` to disable Go's test-result cache, prints each test and coverage per function, and writes `coverage/unit.out` plus `coverage/unit.html`. `make test-all` also includes integration tests and writes `coverage/all.out` plus `coverage/all.html`. Failed tests still return a failing exit status; any generated coverage then may be incomplete.

Set `COVERAGE_DIR` or `TEST_TIMEOUT` in `.env` or the command line to override the report directory or per-package timeout (default: `coverage`, `5m`). Migration operations require `golang-migrate` (installed automatically if missing); seed requires `psql`.

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
| DELETE | `/api/user/:id` | JWT | Soft delete (protects the super-admin role) |

## Docker

```bash
make docker-build
make docker-run
```

## License

MIT

## UUID identifiers

All application primary keys and foreign keys use PostgreSQL `UUID`. Existing CREATE migrations generate IDs with `gen_random_uuid()` (PostgreSQL 13+). Go uses `uuid.UUID`; API IDs and JWT `sub`/`tenant` claims are UUID strings. Provider IDs remain strings. Super admin deletion is blocked by the tenant's `super-admin` role.

These patched migrations describe a fresh development database. An already migrated integer-ID database is not converted by `make migrate-up`; recreate it using `make reset` only when its data may be discarded. Existing numeric-ID JWTs must be replaced by logging in again.
