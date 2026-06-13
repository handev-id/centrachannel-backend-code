# AGENTS.md

## Purpose

This document defines the mandatory coding standards, architecture rules, and implementation guidelines for all AI coding agents working on this repository.

Examples:

- OpenAI Codex
- Claude Code
- Cursor Agent
- OpenCode
- Aider
- Gemini CLI
- Any autonomous coding assistant

Agents MUST follow all rules in this document.

---

# Architecture

This project follows a Feature-Based Modular Architecture.

Business features are located in:

```text
internal/src
```

Shared application components are located in:

```text
internal/middleware
internal/utils
config
database
```

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
│   └── migrations/
├── internal/
│   ├── src/
│   │   └── auth/
│   │       └── repository/
│   ├── middleware/
│   ├── utils/
│   │   ├── exception/
│   │   │   └── exception.go
│   │   ├── hash/
│   │   │   └── hash.go
│   │   ├── logger/
│   │   │   └── logger.go
│   │   └── response/
│   │       └── response.go
│   └── di/
│
├── test/
│   ├── unit/
│   └── integration/
└── ...
```


---

# Mandatory Rules

## Rule 1

Never create business features outside:

```text
internal/src
```

Allowed:

```text
internal/src/auth
internal/src/user
internal/src/product
```

Forbidden:

```text
internal/auth
internal/user
pkg/auth
pkg/user
```

---

## Rule 2

One folder equals one feature.

Example:

```text
internal/src/auth
```

Everything related to authentication must remain inside that folder.

---

## Rule 3

Do not create shared business logic.

Business logic belongs to its feature.

Forbidden:

```text
internal/services
internal/business
internal/managers
```

---

# Feature Structure

Every feature must follow this structure.

```text
feature/
├── feature_handler.go
├── feature_service.go
├── feature_entity.go
├── feature_dto.go
├── feature_routes.go
│
└── repository/
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
│
└── repository/
    ├── auth_repository.go
    └── auth_repository_impl.go
```

Agents must never deviate from this structure.

---

# File Naming Convention

Mandatory format:

```text
<feature>_<type>.go
```

Examples:

```text
auth_handler.go
auth_service.go
auth_entity.go
auth_dto.go
auth_routes.go
```

Forbidden:

```text
handler.go
service.go
entity.go
dto.go
routes.go
```

---

# Repository Rules

Each repository contains only:

```text
repository/
├── auth_repository.go
└── auth_repository_impl.go
```

---

## auth_repository.go

Contains interfaces only.

Example:

```go
type AuthRepository interface {
    FindByID(ctx context.Context, id string) (*User, error)
}
```

---

## auth_repository_impl.go

Contains implementation only.

Example:

```go
type authRepository struct {
    db *sql.DB
}
```

---

# Handler Rules

Handlers are responsible for:

- Request parsing
- DTO validation
- Service invocation
- HTTP response

Handlers must NOT:

- Execute SQL
- Access database directly
- Contain business rules
- Perform complex calculations

---

# Service Rules

Services are responsible for:

- Business rules
- Domain logic
- Validation rules
- Repository orchestration

Services must NOT:

- Parse HTTP requests
- Return Fiber responses
- Execute raw SQL

---

# Entity Rules

Entities represent domain/database models.

Example:

```go
type User struct {
    ID        uuid.UUID
    Name      string
    Email     string
    CreatedAt time.Time
}
```

Entities must NOT:

- Contain business logic
- Contain validation logic
- Contain HTTP logic

---

# DTO Rules

DTOs are required for every request body.

Example:

```go
type LoginRequest struct {
    Email string `json:"email" validate:"required,email"`
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

Routes must be registered inside:

```text
feature_routes.go
```

Example:

```go
func RegisterRoutes(router fiber.Router)
```

Do not register feature routes inside main.go.

main.go should only call route registration functions.

---

# Middleware Rules

Location:

```text
internal/middleware
```

Naming:

```text
auth_middleware.go
cors_middleware.go
logger_middleware.go
recovery_middleware.go
```

Middleware must NOT:

- Access repositories
- Execute business logic
- Execute SQL queries

Middleware should only handle cross-cutting concerns.

---

# Utility Rules

Location:

```text
internal/utils
```

Allowed:

- JWT helpers
- Hash helpers
- Validation helpers
- Response helpers
- Pagination helpers

Forbidden:

- Business logic
- Feature-specific logic

---

# Database Rules

Location:

```text
database
```

Contains:

```text
postgres.go
migrations/
```

Agents must not create database code outside this location.

---

# Dependency Injection

Always inject dependencies.

Allowed:

```go
func NewAuthService(
    repo repository.AuthRepository,
) AuthService
```

Forbidden:

```go
var authRepo = repository.NewAuthRepository()
```

No hidden global dependencies.

---

# Error Handling

Always return errors.

Never panic inside business code.

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

Use centralized logging.

Do not use:

```go
fmt.Println()
```

for application logging.

Use the project's logger utility.

---

# Validation

Validate DTOs before entering service layer.

Preferred flow:

```text
Request
   ↓
DTO Validation
   ↓
Service
   ↓
Repository
```

---

# Testing Rules

Tests should mirror project structure.

Example:

```text
test/
├── integration/
└── e2e/
```

Agents should generate tests for:

- Services
- Repositories
- Critical business flows

---

# Code Style

Priorities:

1. Readability
2. Maintainability
3. Simplicity
4. Performance

Never sacrifice readability for clever code.

---

# Forbidden Patterns

Do not introduce:

- God Objects
- Global State
- Circular Dependencies
- Shared Business Services
- Massive Utility Packages
- Hidden Side Effects

---

# AI Agent Checklist

Before submitting code, verify:

- Feature is inside internal/src
- Naming conventions are correct
- DTO exists
- Entity exists
- Repository interface exists
- Repository implementation exists
- Handler contains no business logic
- Service contains business logic
- No direct SQL outside repositories
- No global state introduced
- Dependency injection used
- Error handling implemented
- Code follows Go conventions

If any item fails, revise the implementation before completion.
