# ARCHITECTURE — CentraChannel API

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.24 |
| HTTP Framework | Fiber v2 |
| Database | PostgreSQL 16 |
| Cache | Redis 7 |
| Migrations | golang-migrate |
| WebSocket | gorilla/websocket |
| Auth | JWT (golang-jwt) |
| DI | Manual (container pattern) |

## Request Lifecycle

```
Client → Domain DNS → Load Balancer → Fiber App
                                           │
                                    Tenant Middleware
                                      (domain → tenant_id)
                                           │
                                    Auth Middleware
                                    (JWT → user_id, role)
                                           │
                                     Handler
                                   (validate DTO)
                                           │
                                     Service
                                   (business logic)
                                           │
                                    Repository
                                     (SQL query)
                                           │
                                       DB/Redis
```

### Tenant Middleware

1. Extract domain from `Host` header
2. Check Redis: `tenant:{domain}` → `tenant_id`
3. Cache miss → query `tenants` table → set Redis (TTL: 1 hour)
4. Set `tenant_id` in Fiber context (`c.Locals("tenant_id")`)
5. Skip for `/api/v1/tenants` routes (onboarding)

### Auth Middleware

1. Extract `Authorization: Bearer {token}` header
2. Parse JWT → extract `user_id`, `tenant_id`
3. Verify `tenant_id` matches tenant middleware value
4. Set `user_id`, `user_role` in Fiber context

## Project Structure

```
centrachannel/
├── cmd/
│   └── server/
│       └── main.go
├── config/
│   ├── config.go
│   └── config.yaml
├── database/
│   ├── connector.go
│   ├── postgres.go
│   ├── redis.go
│   ├── seed.sql
│   ├── scripts/
│   └── migrations/
│       ├── 000001_create_tenants_table.up.sql
│       ├── 000001_create_tenants_table.down.sql
│       ├── 000002_create_roles_table.up.sql
│       ├── 000002_create_roles_table.down.sql
│       ├── 000003_create_users_table.up.sql
│       ├── 000003_create_users_table.down.sql
│       ├── 000004_create_role_user_table.up.sql
│       ├── 000004_create_role_user_table.down.sql
│       ├── 000005_create_auth_access_tokens_table.up.sql
│       ├── 000005_create_auth_access_tokens_table.down.sql
│       └── ...
├── internal/
│   ├── src/
│   │   ├── auth/
│   │   │   ├── auth_handler.go
│   │   │   ├── auth_service.go
│   │   │   ├── auth_entity.go
│   │   │   ├── auth_dto.go
│   │   │   ├── auth_routes.go
│   │   │   ├── auth_repository.go
│   │   │   └── auth_repository_impl.go
│   │   ├── user/
│   │   │   ├── user_handler.go
│   │   │   ├── user_service.go
│   │   │   ├── user_entity.go
│   │   │   ├── user_dto.go
│   │   │   ├── user_routes.go
│   │   │   ├── user_repository.go
│   │   │   └── user_repository_impl.go
│   │   ├── tenant/
│   │   ├── dashboard/
│   │   ├── channel/
│   │   ├── contact/
│   │   ├── profile/
│   │   ├── conversation/
│   │   ├── message/
│   │   ├── message_raw/
│   │   ├── tag/
│   │   ├── note/
│   │   └── whatsapp_device/
│   ├── middleware/
│   │   ├── tenant_middleware.go
│   │   └── auth_middleware.go
│   ├── utils/
│   │   ├── exception/
│   │   ├── hash/
│   │   ├── logger/
│   │   └── response/
│   └── di/
│       └── container.go
├── test/
├── docs/
│   └── product/
│       ├── PRD.md
│       ├── SRS.md
│       └── ARCHITECTURE.md
├── Dockerfile
├── Makefile
├── go.mod
└── go.sum
```

## Feature Pattern

Each feature in `internal/src/` follows:

```
feature_handler.go      # HTTP layer: parse request, validate DTO, call service, send response
feature_service.go      # Business logic: rules, orchestration, transaction management
feature_entity.go       # DB model struct
feature_dto.go          # Request/response DTOs
feature_routes.go       # Route registration
feature_repository.go   # Interface + DBTX type
feature_repository_impl.go  # SQL implementation
```

## Key Architecture Decisions

### Multi-Tenancy: Domain-Based + Row-Level

One application instance, one database. All tenant-scoped tables have a `tenant_id` column. Queries always include `WHERE tenant_id = $N`. The domain header resolves to `tenant_id` via Redis cache (fallback to DB).

### Dependency Injection

The `di.Container` holds shared dependencies (Config, *sql.DB, *sql.Redis, Logger). Handlers receive the container and extract what they need. Services receive only their specific dependencies (never the container).

### Repository Pattern with DBTX

All repository methods accept `DBTX` (interface matching both `*sql.DB` and `*sql.Tx`) as the first parameter. This allows:
- Direct calls: `repo.GetByEmail(ctx, db, ...)`
- Transactional calls: `repo.Create(ctx, tx, ...)`

### File Upload

Files are uploaded to `https://storage.solodevs.my.id` via multipart POST. The returned `public_url` is stored in the message's `attachment` JSONB field.

### WebSocket

WebSocket only pushes notification events (created/updated). The FE refetches data when notified. No data is transported over WebSocket.

Example events:
```json
{"type": "message.created", "data": {"conversation_id": 1, "message_id": 42}}
{"type": "conversation.updated", "data": {"conversation_id": 1, "status": "assigned"}}
```

### Avatar Generation

User and contact avatars without a custom image get a generated DiceBear URL:
```
https://api.dicebear.com/9.x/initials/svg?seed={name}
```

### Contact Merge

- `contacts.merged_to_id` points to the surviving contact
- Profiles of merged contacts get `merged_from_contact_id` set
- Merged contacts are soft-deleted
- Unmerge restores the original contact and clears profile references

### Channel Types

| Type | Description |
|------|-------------|
| facebook | Facebook Page integration |
| instagram | Instagram Professional integration |
| whatsapp_business | Official WhatsApp Business API (Meta) |
| whatsapp | Unofficial/third-party WhatsApp (mock for MVP) |

Channels are global records (shared across all tenants) with no `tenant_id`.

### WhatsApp Unofficial (MVP)

- Device config stored in `whatsapp_devices` table
- Connect/disconnect/scan endpoints return mock responses
- Actual integration is Phase 2/3

### Campaign (Phase 3)

Campaign tables follow the crm-be schema:
- `campaigns`: name, type, status, sending_option, stats
- `campaign_templates`: name, type, content, variables
- `campaign_recipient_lists`: name, source, status
- `campaign_recipient_contact_lists`: contact info + FK to recipient list
- `campaign_recipients`: per-recipient status, delivery/open/click times

All campaign tables get `tenant_id`.

### Soft Delete

Applied via `deleted_at` column. Queries filter `WHERE deleted_at IS NULL`. The super admin (user with role super-admin and id=1 within tenant) cannot be deleted (returns 403).

### N+1 Prevention

Batch methods exist for list operations. Example:
```go
GetRolesByUserIDs(ctx, db, tenantID, userIDs []int) (map[int][]Role, error)
```
Instead of looping `GetRolesByUserID` per user.

### Seed Data

Seeds run on initial tenant creation:
- Roles: super-admin, admin, agent
- Channels: facebook, instagram, whatsapp_business, whatsapp
