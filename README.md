# CentraChannel API

Modern REST API built with Go, Fiber, and PostgreSQL.

A feature-based architecture designed for maintainability, scalability, readability, and AI-assisted development.

---

# Overview

CentraChannel API follows a Feature-Based Modular Architecture where every business capability is grouped into its own module.

Goals:

- Easy onboarding
- Clear ownership
- Easy refactoring
- Predictable code organization
- AI-friendly code generation
- Scalable project structure

---

# Technology Stack

## Backend

- Go
- Fiber

## Database

- PostgreSQL

## Authentication

- JWT

## Containerization

- Docker
- Docker Compose

## Testing

- Go Testing Package

---

# Project Structure

```text
centrachannel/
│
├── cmd/
│   └── server/
│       └── main.go
│
├── config/
│   └── config.go
│
├── database/
│   ├── postgres.go
│   └── migrations/
│       └── 001_init.sql
│
├── internal/
│   ├── src/
│   │   └── auth/
│   │       └── repository/
│   ├── middleware/
│   │   ├── cors_middleware.go
│   │   └── logger_middleware.go
│   │
│   ├── utils/
│   │   ├── exception/
│   │   │   └── exception.go
│   │   ├── hash/
│   │   │   └── hash.go
│   │   ├── logger/
│   │   │   └── logger.go
│   │   └── response/
│   │       └── response.go
│
├── test/
│   ├── unit/
│   └── integration/
│
├── .env
├── .env.example
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── go.sum
├── README.md
└── AGENTS.md
```

---

# Architecture Principles

## Feature First

Every business capability must live inside:

```text
internal/src
```

Examples:

```text
internal/src/auth
internal/src/user
internal/src/product
internal/src/order
```

Each feature owns:

- Handler
- Service
- Entity
- DTO
- Routes
- Repository

---

## Clear Layer Separation

```text
Handler
   ↓
Service
   ↓
Repository
   ↓
Database
```

Responsibilities never overlap.

---

# Feature Structure

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

## auth_handler.go

Responsible for:

- HTTP Request Parsing
- DTO Validation
- Service Invocation
- HTTP Response

Must not contain business logic.

---

## auth_service.go

Responsible for:

- Business Rules
- Domain Logic
- Data Processing
- Repository Coordination

Must not directly query database.

---

## auth_entity.go

Represents domain/database entities.

---

## auth_dto.go

Represents request and response payloads.

---

## auth_routes.go

Registers feature routes.

---

# Repository Convention

Only two repository files are allowed.

```text
repository/
├── auth_repository.go
└── auth_repository_impl.go
```

## auth_repository.go

Contains repository contract.

## auth_repository_impl.go

Contains repository implementation.

---

# Middleware

Location:

```text
internal/middleware
```

Convention:

```text
auth_middleware.go
cors_middleware.go
logger_middleware.go
recovery_middleware.go
request_id_middleware.go
rate_limit_middleware.go
```

Middleware responsibilities:

- Authentication
- Authorization
- Logging
- Request Tracking
- Recovery
- Rate Limiting

Business logic is prohibited.

---

# Configuration

Location:

```text
config/
```

Contains application configuration loading and environment management.

---

# Database

Location:

```text
database/
```

Structure:

```text
database/
├── postgres.go
└── migrations/
```

Responsibilities:

- Connection Pooling
- Health Checks
- Transaction Management
- Database Migrations

---

# Utilities

Location:

```text
internal/utils
```

Contains reusable helper packages.

Examples:

```text
exception/
hash/
logger/
response/
```

Rules:

- No business logic
- Reusable only
- Feature agnostic

---

# API Response Standard

Success:

```json
{
  "meta": {
    "code": 200,
    "message": "Success"
  },
  "data": {}
}
```

Error:

```json
{
  "meta": {
    "code": 400,
    "message": "Bad Request"
  },
  "errors": {}
}
```

---

# Development Rules

1. All business features must live inside `internal/src`.
2. One folder represents one business feature.
3. Feature files must use feature prefixes.
4. Middleware files must use middleware suffixes.
5. Handler must not contain business logic.
6. Service must not access database directly.
7. Repository handles persistence only.
8. DTO is mandatory for request validation.
9. Entity represents domain/database objects.
10. Repository contains only interface and implementation.
11. Utilities must be reusable and feature agnostic.
12. Prefer readability over clever code.
13. Use dependency injection.
14. Keep functions small and focused.
15. Follow Go idioms whenever possible.

---

# License

MIT License
