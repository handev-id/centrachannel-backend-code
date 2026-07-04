# ARCHITECTURE — CentraChannel API

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.24 |
| HTTP Framework | Fiber v3 |
| Database | PostgreSQL 16 |
| Cache | Redis 7 |
| Migrations | golang-migrate |
| Realtime Events | _(not yet implemented, planned: WebSocket)_ |
| Auth | JWT (golang-jwt) |
| DI | Manual (container pattern) |

## Dual-Layer Architecture

Two access layers based on domain:

| Layer | Domain | Tenant Context | Auth |
|-------|--------|----------------|------|
| **`internal/app`** | `centrachannel.com/api` | No (resolved from payload if needed) | Optional (API key for webhooks) |
| **`internal/src`** | `{tenant}.centrachannel.com/api` | Yes (from domain via TenantMiddleware) | JWT + Role |

## Request Lifecycle

### Main Domain (`internal/app`)

```
centrachannel.com/api/*
       │
  CORS + Logger
       │
  ┌────┴──────────────────────────────┐
  │  Observability: /health, /ping    │
  │  Docs:          /docs, /swagger   │
  │  Webhook:       /webhook/*        │
  │  Registration:  /api/tenants/onboard
  └───────────────────────────────────┘
```

### Subdomain (`internal/src`)

```
{tenant}.centrachannel.com/api/*
       │
  CORS + Logger
       │
  TenantMiddleware (domain → tenant_id from Redis/DB)
       │
  AuthMiddleware (JWT → user_id, role)
       │
  RoleMiddleware (super-admin / admin / agent)
       │
  Handler → Service → Repository → DB/Redis
```

### Tenant Middleware

1. Extract domain from `Host` header
2. Check Redis: `tenant:{domain}` → `tenant_id`
3. Cache miss → query `tenants` table → set Redis (TTL: 1 hour)
4. Set `tenant` in Fiber context (`c.Locals("tenant")`)
5. Applied only to routes after `app.Use(TenantMiddleware)` (subdomain routes)

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
│       └── ...
├── internal/
│   ├── app/
│   │   ├── docs/
│   │   │   └── docs.go
│   │   ├── observability/
│   │   │   ├── observability_handler.go
│   │   │   └── observability_routes.go
│   │   ├── registration/
│   │   │   ├── registration_handler.go
│   │   │   ├── registration_service.go
│   │   │   ├── registration_dto.go
│   │   │   ├── registration_routes.go
│   │   │   └── registration_service_test.go
│   │   └── webhook/
│   │       ├── webhook_handler.go
│   │       ├── webhook_service.go
│   │       ├── webhook_entity.go
│   │       ├── webhook_dto.go
│   │       ├── webhook_routes.go
│   │       ├── webhook_handler_test.go
│   │       └── webhook_service_test.go
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
│   │   ├── campaign/
│   │   ├── tag/
│   │   ├── note/
│   │   ├── conversation_tag/
│   │   └── whatsapp_device/
│   ├── middleware/
│   │   ├── tenant_middleware.go
│   │   └── auth_middleware.go
│   ├── utils/
│   │   ├── exception/
│   │   ├── hash/
│   │   ├── logger/
│   │   ├── response/
│   │   └── avatar/
│   ├── messenger/
│   │   ├── messenger.go         # OutgoingMessage + Messenger interface
│   │   ├── meta.go              # MetaSender (fb/ig direct)
│   │   ├── evolution.go         # EvolutionSender (wa/wa_business via Evolution API)
│   │   ├── mock.go              # MockSender (fallback)
│   │   └── dispatcher.go        # NewSender factory
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

Features in both `internal/app/` and `internal/src/` follow:

```
feature_handler.go      # HTTP layer: parse request, validate DTO, call service, send response
feature_service.go      # Business logic: rules, orchestration, transaction management
feature_entity.go       # DB model struct
feature_dto.go          # Request/response DTOs
feature_routes.go       # Route registration
feature_repository.go   # Interface + DBTX type
feature_repository_impl.go  # SQL implementation
```

**Difference:** `internal/app/` features register routes **before** TenantMiddleware (main domain). `internal/src/` features register routes **after** TenantMiddleware + AuthMiddleware (subdomain). `internal/app/` features may import entities and repositories from `internal/src/` when they need tenant data resolved from payload (e.g., webhook resolves tenant from Meta page ID).

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

| Type | Backend | Description |
|------|---------|-------------|
| facebook | Meta Graph API (direct) | Facebook Page integration |
| instagram | Meta Graph API (direct) | Instagram Professional integration |
| whatsapp_business | Evolution API (→ Meta Graph API) | Official WhatsApp Business API, proxied via Evolution API |
| whatsapp | Evolution API (→ Baileys) | Unofficial WhatsApp Web via Baileys library |

Channels are global records (shared across all tenants) with no `tenant_id`.

### Per-Tenant Channel Credentials

Each tenant stores its own channel credentials in `tenants.settings` JSONB:

```json
{
  "channel_configuration": {
    "meta_access_token": "...",
    "whatsapp_phone_id": "...",
    "evolution_business_instance": "t1-waba"
  }
}
```

- `meta_access_token` / `whatsapp_phone_id` — used by direct Meta sender (fallback for WhatsApp Business, primary for Facebook/Instagram)
- `evolution_business_instance` — Evolution API instance name for WhatsApp Business (WHATSAPP-BUSINESS integration)

These are set during tenant onboarding and can be updated via tenant settings API.

### WhatsApp Device Management

Each tenant can register multiple WhatsApp devices (unofficial) in `whatsapp_devices`:

| Field | Description |
|-------|-------------|
| `whatsapp_id` | Evolution API instance name (matches the instance created on Evolution API server) |
| `status` | `CONNECTED` / `DISCONNECTED` |
| `phone` | Phone number associated |

Device CRUD plus `connect`, `disconnect`, `scan` (QR code) endpoints interact with the Evolution API instance management endpoints.

### Profile-to-Device Linking

When a profile is created via webhook from an Evolution API instance, the profile's `linked_device_whatsapp_id` is set to the sending device's `whatsapp_id`. This links incoming messages to the correct outbound device for replies.

### Messenger Architecture

```
                        ┌──────────────────┐
                        │   CentraChannel   │
                        │  (message_service)│
                        └───────┬──────────┘
                                │
                    ┌───────────┴───────────┐
                    │   channel type switch  │
                    └───┬───────┬───────┬───┘
                        │       │       │
                   ┌────┘   ┌───┘   ┌───┘
                   ▼        ▼       ▼
             ┌─────────┐ ┌─────┐ ┌──────┐
             │ FB / IG │ │ WA  │ │ WA   │
             │         │ │Biz  │ │Unoff │
             └────┬────┘ └──┬──┘ └──┬───┘
                  │         │       │
                  ▼         ▼       ▼
          ┌──────────┐ ┌──────────────────┐
          │Meta Graph│ │  Evolution API   │
          │ API v22  │ │ (REST + apikey)  │
          │ (direct) │ ├────────┬────────┤
          └──────────┘ │Baileys │ Meta   │
                       │(Web)   │(Cloud) │
                       └────────┴────────┘
```

### Messenger Package

`internal/messenger/` handles outbound messages to external platforms:

| Sender | Channel Types | Backend |
|--------|--------------|---------|
| **`MetaSender`** | `facebook`, `instagram` | Meta Graph API v22.0 (direct) |
| **`EvolutionSender`** | `whatsapp`, `whatsapp_business` | Evolution API (proxied) |
| **`MockSender`** | fallback | Returns mock message ID |

#### Sender Selection Logic

```
ch.Type = "facebook" / "instagram"
  → MetaSender (direct to Meta Graph API)

ch.Type = "whatsapp_business"
  → EvolutionSender if tenant has evolution_business_instance configured
  → MetaSender (direct fallback)

ch.Type = "whatsapp"
  → EvolutionSender if profile has linked_device_whatsapp_id
  → MockSender (fallback)
```

#### EvolutionSender (`internal/messenger/evolution.go`)

- Sends to `POST /message/sendText/{instanceName}` and `POST /message/sendMedia/{instanceName}`
- Authenticated via `apikey` header
- Returns Evolution API message key as `webhook_message_id`
- Instance name is determined by the channel type:
  - `whatsapp`: `profile.linked_device_whatsapp_id`
  - `whatsapp_business`: `tenant.settings.channel_configuration.evolution_business_instance`

#### Message Delivery Flow

When a user sends a message:
1. Message saved to DB in a transaction (with `status = "sent"`)
2. `deliverToExternal()` runs in a goroutine (non-blocking)
3. Lookup: conversation → profile (external_id) → channel (type)
4. Load tenant settings → parse `channel_configuration`
5. Select sender based on channel type (see selection logic above)
6. Call `Send()` with recipient's external ID and message content
7. On success: store external message ID in `messages.webhook_message_id`
8. On failure: update message status to `"failed"`

### WhatsApp Device Client (`internal/src/whatsapp_device/`)

| Client | Description |
|--------|-------------|
| **`EvolutionClient`** | Real implementation using Evolution API REST endpoints |
| **`MockClient`** | Mock for testing (returns success without API calls) |

`EvolutionClient` implements `WhatsAppClient` interface:

| Method | Evolution API Endpoint |
|--------|----------------------|
| `GetQR()` | `GET /instance/connect/{whatsapp_id}` |
| `CheckConnection()` | `GET /instance/connectionState/{whatsapp_id}` |
| `Disconnect()` | `DELETE /instance/delete/{whatsapp_id}` |
| `SendMessage()` | `POST /message/sendText/{whatsapp_id}` |

Used by device management endpoints (`/api/whatsapp-devices/:id/scan|connect|disconnect`).

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
