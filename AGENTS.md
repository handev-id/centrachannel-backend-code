# AGENTS.md

## Purpose

This document defines the mandatory coding standards, architecture rules, and implementation guidelines for all AI coding agents working on this repository.

Agents MUST follow all rules in this document.

---

# Architecture

This project follows a Dual-Layer Feature-Based Modular Architecture.

Two layers exist under `internal/`:

- **`internal/app`** — HTTP handlers registered on the **main domain** (`centrachannel.com/api`).
  Routes here are registered **before** `TenantMiddleware`. No tenant context is resolved from the request.
  Examples: documentation, health checks, webhooks (tenant resolved from payload), tenant registration.

- **`internal/src`** — Business features accessed via **subdomain** (`{tenant}.centrachannel.com/api`).
  Routes here are registered **after** `TenantMiddleware` + `AuthMiddleware`.
  Tenant context is resolved from the domain via `TenantMiddleware`.
  Examples: auth, campaign, contact, conversation, message.

Shared components are in `internal/middleware`, `internal/utils`, `config`, `database`.

A lightweight Dependency Injection container lives in `internal/di` and provides `Config`, `*sql.DB`, and `Logger` instances.

Agents must preserve this architecture.

---

# Project Structure

```text
centrachannel/
│
├── cmd/
├── config/
├── database/
│   ├── connector.go
│   ├── postgres.go
│   ├── seed.sql
│   ├── scripts/
│   └── migrations/
├── internal/
│   ├── app/
│   │   ├── docs/
│   │   ├── observability/
│   │   ├── registration/
│   │   └── webhook/
│   ├── src/
│   │   ├── auth/
│   │   ├── campaign/
│   │   ├── channel/
│   │   ├── contact/
│   │   ├── conversation/
│   │   ├── conversation_tag/
│   │   ├── dashboard/
│   │   ├── message/
│   │   ├── note/
│   │   ├── tag/
│   │   ├── tenant/
│   │   ├── upload/
│   │   ├── user/
│   │   └── whatsapp_device/
│   ├── messenger/
│   │   ├── messenger.go
│   │   ├── meta.go
│   │   ├── mock.go
│   │   └── dispatcher.go
│   ├── middleware/
│   ├── utils/
│   │   ├── exception/
│   │   ├── hash/
│   │   ├── logger/
│   │   └── response/
│   └── di/
│
├── test/
├── Dockerfile
├── Makefile
└── ...
```

---

# Mandatory Rules

## Rule 1 — Feature Location

Features are split into two layers:

- **`internal/app`** — Main domain routes (no tenant context from request).
  Examples: `internal/app/docs`, `internal/app/observability`, `internal/app/webhook`, `internal/app/registration`.

- **`internal/src`** — Tenant-scoped routes (accessed via subdomain with TenantMiddleware).
  Examples: `internal/src/auth`, `internal/src/campaign`, `internal/src/contact`.

Never create business features outside these directories.

Allowed: `internal/app/docs`, `internal/app/webhook`, `internal/src/auth`, `internal/src/user`

Forbidden: `internal/auth`, `internal/user`, `pkg/auth`, `pkg/user`

## Rule 2 — One Folder = One Feature

Everything related to a feature must remain inside its folder.

## Rule 3 — No Shared Business Logic

Business logic belongs to its feature. Forbidden: `internal/services`, `internal/business`, `internal/managers`

---

# Feature Structure

Every feature must follow this structure (flat — no `repository/` sub-package):

```text
feature/
├── feature_handler.go
├── feature_service.go
├── feature_entity.go
├── feature_dto.go
├── feature_routes.go
├── feature_repository.go
└── feature_repository_impl.go
```

Example:

```text
auth/
├── auth_handler.go
├── auth_service.go
├── auth_entity.go
├── auth_dto.go
├── auth_routes.go
├── auth_repository.go
└── auth_repository_impl.go
```

> **Important**: Repository files live in the feature root, not in a `repository/` sub-package. A sub-package that imports its parent creates an import cycle in Go.

Agents must never deviate from this structure.

---

# File Naming Convention

Mandatory format: `<feature>_<type>.go`

Examples: `auth_handler.go`, `auth_service.go`, `auth_entity.go`, `auth_dto.go`, `auth_routes.go`

Forbidden: `handler.go`, `service.go`, `entity.go`, `dto.go`, `routes.go`

---

# Repository Rules

Each feature contains only two repository files at the feature root:

```text
auth_repository.go        # Interface only
auth_repository_impl.go   # Implementation only
```

## auth_repository.go

Contains the repository interface and the `DBTX` type for transaction support:

```go
type DBTX interface {
    ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
    QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
    QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

type AuthRepository interface {
    GetByEmail(ctx context.Context, q DBTX, tenantID int, email string) (*User, error)
    GetByID(ctx context.Context, q DBTX, tenantID int, id int) (*User, error)
    Create(ctx context.Context, q DBTX, user *User) (int, error)
    GetRolesByUserID(ctx context.Context, q DBTX, tenantID int, userID int) ([]Role, error)
}
```

All methods accept `DBTX` as first param — allows passing `*sql.DB` (direct) or `*sql.Tx` (transactional).

## auth_repository_impl.go

Contains struct and methods only:

```go
type authRepository struct{}
func NewAuthRepository() AuthRepository { return &authRepository{} }
```

---

# Handler Rules

Handlers are responsible for:
- HTTP Request parsing
- DTO validation
- Service invocation
- HTTP response

Handlers must NOT:
- Execute SQL
- Access database directly
- Contain business rules
- Perform complex calculations

Handler receives `*di.Container` and injects dependencies into service:

```go
func NewAuthHandler(c *di.Container) *AuthHandler {
    repo    := NewAuthRepository()
    service := NewAuthService(repo, c.DB, c.Config, c.Logger)
    return &AuthHandler{service: service}
}
```

---

# Service Rules

Services are responsible for:
- Business rules
- Domain logic
- Validation rules
- Repository orchestration
- Transaction management

Services must NOT:
- Parse HTTP requests
- Return Fiber responses
- Execute raw SQL outside repository calls

Service receives its dependencies explicitly (not the container):

```go
type authService struct {
    repo   AuthRepository
    db     *sql.DB
    cfg    *config.Config
    logger *logger.Logger
}
```

For transactional operations, use `db.BeginTx()` and pass `*sql.Tx` to repo methods:

```go
tx, _ := s.db.BeginTx(ctx, nil)
userID, _ := s.repo.Create(ctx, tx, user)
tx.ExecContext(ctx, `INSERT INTO role_user ...`, userID, 3)
tx.Commit()
```

---

# Entity Rules

Entities represent database models.

```go
type Tenant struct {
    ID        int             `json:"id"`
    Name      string          `json:"name"`
    Domain    string          `json:"domain"`
    Logo      json.RawMessage `json:"logo,omitempty"`
    Address   *string         `json:"address,omitempty"`
    Phone     *string         `json:"phone,omitempty"`
    Email     *string         `json:"email,omitempty"`
    IsActive  bool            `json:"is_active"`
    Settings  json.RawMessage `json:"settings,omitempty"`
    CreatedAt time.Time       `json:"created_at"`
    UpdatedAt time.Time       `json:"updated_at"`
}

type User struct {
    ID        int              `json:"id"`
    TenantID  int              `json:"tenant_id"`
    FirstName string           `json:"first_name"`
    LastName  *string          `json:"last_name,omitempty"`
    Username  string           `json:"username"`
    Email     string           `json:"email"`
    Phone     *string          `json:"phone,omitempty"`
    Password  string           `json:"-"`
    Avatar    json.RawMessage  `json:"avatar,omitempty"`
    LastLogin *time.Time       `json:"last_login,omitempty"`
    DeletedAt sql.NullTime     `json:"deleted_at,omitempty"`
    CreatedAt time.Time        `json:"created_at"`
    UpdatedAt time.Time        `json:"updated_at"`
    Roles     []Role           `json:"roles,omitempty"`
}

type Role struct {
    ID        int       `json:"id"`
    TenantID  int       `json:"tenant_id"`
    Name      string    `json:"name"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

Entities must NOT:
- Contain business logic
- Contain validation logic
- Contain HTTP logic

---

# DTO Rules

DTOs are required for every request body.

```go
type RegisterRequest struct {
    FirstName string          `json:"first_name" validate:"required,max=255"`
    LastName  *string         `json:"last_name,omitempty"`
    Username  string          `json:"username" validate:"required,min=3,max=50"`
    Email     string          `json:"email" validate:"required,email"`
    Phone     *string         `json:"phone,omitempty"`
    Avatar    json.RawMessage `json:"avatar,omitempty"`
    Password  *string         `json:"password,omitempty" validate:"omitempty,min=8"`
}
```

Do not bind request bodies directly into entities.

Forbidden:
```go
var user User
ctx.BodyParser(&user)
```

Always use DTOs.

---

# Route Rules

Routes must be registered inside `feature_routes.go`.

```go
func RegisterRoutes(router fiber.Router, handler *HandlerType)
```

Do not register feature routes inside main.go.

---

# Database Rules

Location: `database/`

```text
database/
├── connector.go       # DBConnector interface
├── postgres.go        # PostgreSQL connection
├── seed.sql           # Seed data (roles)
├── scripts/           # Shell helpers for migrate/seed/reset
└── migrations/        # golang-migrate format
    ├── {version}_{title}.up.sql
    └── {version}_{title}.down.sql
```

Migrations use `golang-migrate` format — each has a paired up/down file.

Seed data goes in `database/seed.sql` and runs via `make seed`.

Agents must not create database code outside this location.

---

# Migration Rules

- Every schema change needs an up/down pair
- Up file: `{timestamp}_{action}_{table}.up.sql`
- Down file: `{timestamp}_reverse-action_{table}.down.sql`
- Use `CREATE` for up titles, `DROP` for down titles (ensures correct `ls` ordering)
- Version must be a unique timestamp

## Migration Ordering (CRITICAL)

The `tenants` table MUST be created BEFORE any table that references it via foreign key. The correct migration order is:

```
01. create_tenants_table        ← MUST be first (all tables depend on it)
02. create_roles_table          ← references tenants.id via tenant_id
03. create_users_table          ← references tenants.id via tenant_id
04. create_role_user_table      ← pivot between users and roles
05. create_auth_access_tokens   ← references users.id
```

Agents MUST NOT create migrations out of this order. A migration creating a table with `REFERENCES tenants(id)` MUST have a version number higher than the tenants table migration.

---

# N+1 Prevention

Batch-load related data in a single query instead of looping:

```go
// BAD — N+1:
for _, u := range users {
    roles, _ := repo.GetRolesByUserID(ctx, db, u.ID)
    u.Roles = roles
}

// GOOD — 1 query:
rolesMap, _ := repo.GetRolesByUserIDs(ctx, db, userIDs)
for _, u := range users {
    u.Roles = rolesMap[u.ID]
}
```

Always provide a batch variant for list operations (`GetRolesByUserIDs`, etc.).

---

# Middleware Rules

Location: `internal/middleware`

Middleware must NOT:
- Access repositories
- Execute business logic
- Execute SQL queries

Middleware should only handle cross-cutting concerns (auth, CORS, logging).

---

# Utility Rules

Location: `internal/utils`

Allowed: JWT helpers, Hash helpers, Validation helpers, Response helpers, Pagination helpers

Forbidden: Business logic, Feature-specific logic

---

# Dependency Injection

Always inject dependencies. No hidden global dependencies.

Allowed:
```go
func NewAuthService(repo AuthRepository, db *sql.DB, cfg *config.Config, logger *logger.Logger) AuthService
```

Forbidden:
```go
var authRepo = NewAuthRepository()
```

Handler receives the container and distributes dependencies.

Service receives only what it needs — NOT the container.

---

# Soft Delete & Super Admin Protection

- Users use soft delete via `deleted_at` column
- User with `id=1` (super admin) must NOT be deletable
- Return 403 with message `"You can not delete super admin"`

---

# Error Handling

Always return errors. Never panic inside business code.

Preferred:
```go
if err != nil {
    return err
}
```

Forbidden:
```go
panic(err)
```

---

# Logging

Use centralized logger. Do not use `fmt.Println()` for application logging.

---

# Code Style

Priorities:
1. Readability
2. Maintainability
3. Simplicity
4. Performance

Never sacrifice readability for clever code.

---

---

# Channel Credentials (Multi-Tenant)

Every tenant has its own Meta API credentials stored in the `tenants.settings` JSONB column:

```json
{
  "channel_configuration": {
    "meta_access_token": "...",
    "whatsapp_phone_id": "..."
  }
}
```

- These credentials are **per-tenant**, not global env vars
- The message service loads them from the database via `tenantRepo.GetByID()` on every send
- Meta credentials are used for `facebook`, `instagram`, and `whatsapp_business` channels
- `whatsapp` (unofficial) uses `MockSender` — no credentials needed

---

# Messenger Package

`internal/messenger/` handles sending messages to external platforms:

- `Messenger` interface with `Send(msg) (string, error)`
- `MetaSender` — sends to Facebook/Instagram/WhatsApp Business via Meta Graph API v22.0
- `MockSender` — mock for WhatsApp unofficial channel
- `NewSender(channelType, MetaConfig)` — factory that creates the right sender

The message service calls `deliverToExternal()` in a goroutine after saving the message to DB:

1. Lookup conversation → profile (external_id) → channel (type)
2. Load tenant settings → parse `channel_configuration`
3. Create sender and call `Send()`
4. On success: save Meta `message_id` to `messages.webhook_message_id`
5. On failure: update message status to `"failed"`

---

# Forbidden Patterns

- God Objects
- Global State
- Circular Dependencies
- Shared Business Services
- Massive Utility Packages
- Hidden Side Effects

---

# AI Agent Checklist

Before submitting code, verify:

- Feature is in the correct layer (`internal/app` for main domain, `internal/src` for subdomain)
- Naming conventions are correct (`<feature>_<type>.go`)
- DTO exists for every request body
- Entity exists with correct fields (including `TenantID` where applicable)
- Repository interface exists with DBTX support
- Repository implementation exists
- Handler contains no business logic
- Service contains business logic
- No direct SQL outside repositories
- N+1 queries are avoided (batch methods provided)
- Transaction used for multi-table writes (create user + attach role)
- No global state introduced
- Dependency injection used
- Soft delete is respected (queries filter `deleted_at IS NULL`)
- Every tenant-scoped query includes `tenant_id = $N` filter
- Super admin (id=1) is protected from deletion
- Migration files have up/down pairs
- Migration order: tenants → roles → users → role_user → tokens
- Seed data in `database/seed.sql`
- Code follows Go conventions

If any item fails, revise the implementation before completion.

## Commit Rule

Do NOT commit, amend, or push any changes unless the user explicitly asks for it with a command like "commit", "push", "commit dan push", or similar. Pushing or committing without explicit user instruction is forbidden.
