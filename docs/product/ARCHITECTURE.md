# Architecture — CentraChannel API

## Request Lifecycle

```
business-a.centrachannel.com/api/auth/login
  │
  ▼
[Tenant Middleware]
  ├─ read full domain "business-a.centrachannel.com" from Host header
  ├─ Redis GET tenant:business-a.centrachannel.com
  │    ├─ HIT  → use cached Tenant
  │    └─ MISS → DB: SELECT * FROM tenants WHERE domain=$1
  │              → Redis SETEX tenant:{domain} 3600
  └─ store Tenant in c.Locals("tenant")
  │
  ▼
[Auth Middleware] (optional, per route group)
  ├─ extract & verify JWT
  └─ c.Locals("user")
  │
  ▼
[Handler] → get tenant from c.Locals("tenant")
  │
  ▼
[Service] → receive *Tenant as parameter
  │
  ▼
[Repository] → all queries: WHERE tenant_id = $N
```

## Tenant Propagation

```
Handler:    tenant := c.Locals("tenant").(*entity.Tenant)
            h.service.Register(c.Context(), req, tenant)

Service:    func (s *authService) Register(ctx, req, tenant) {
                user.TenantID = tenant.ID
                s.repo.Create(ctx, tx, user)  // INSERT includes tenant_id
            }

Repository: SELECT ... FROM users WHERE email=$1 AND tenant_id=$2
```

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Language | Go 1.26 |
| Framework | Fiber v3 |
| Database | PostgreSQL (lib/pq) |
| Cache | Redis (go-redis/v9) |
| Auth | JWT HS256 |
| Migration | golang-migrate |
| Validator | go-playground/validator |

## Relevant Folder Structure

```text
internal/
├── src/
│   ├── tenant/            # new feature
│   │   ├── tenant_entity.go
│   │   ├── tenant_repository.go
│   │   ├── tenant_repository_impl.go
│   │   └── tenant_cache.go
│   ├── auth/              # existing + add tenant_id
│   └── user/              # existing + add tenant_id
├── middleware/
│   └── tenant_middleware.go  # new
└── di/container.go        # add *redis.Client
```

## DI Container (update)

```go
type Container struct {
    Config *config.Config
    DB     *sql.DB
    Redis  *redis.Client
    Logger *logger.Logger
}
```
