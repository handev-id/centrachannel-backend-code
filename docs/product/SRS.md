# SRS — Software Requirements Specification

## 1. Data Model

### tenants

```sql
CREATE TABLE tenants (
    id         SERIAL PRIMARY KEY,
    name       VARCHAR(255) NOT NULL,
    domain     VARCHAR(255) NOT NULL UNIQUE,
    logo       JSONB,
    address    TEXT,
    phone      VARCHAR(20),
    email      VARCHAR(255),
    is_active  BOOLEAN NOT NULL DEFAULT TRUE,
    settings   JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_tenants_domain ON tenants(domain);
```

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
```

### users (tenant-scoped)

```sql
CREATE TABLE users (
    id         SERIAL PRIMARY KEY,
    tenant_id  INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    first_name VARCHAR(255) NOT NULL,
    last_name  VARCHAR(255),
    username   VARCHAR(50)  NOT NULL,
    email      VARCHAR(255) NOT NULL,
    phone      VARCHAR(20),
    password   VARCHAR(255) NOT NULL,
    avatar     JSONB,
    last_login TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, username),
    UNIQUE(tenant_id, email)
);
```

```go
type User struct {
    ID        int             `json:"id"`
    TenantID  int             `json:"tenant_id"`
    FirstName string          `json:"first_name"`
    LastName  *string         `json:"last_name,omitempty"`
    Username  string          `json:"username"`
    Email     string          `json:"email"`
    Phone     *string         `json:"phone,omitempty"`
    Password  string          `json:"-"`
    Avatar    json.RawMessage `json:"avatar,omitempty"`
    LastLogin *time.Time      `json:"last_login,omitempty"`
    DeletedAt sql.NullTime    `json:"deleted_at,omitempty"`
    CreatedAt time.Time       `json:"created_at"`
    UpdatedAt time.Time       `json:"updated_at"`
    Roles     []Role          `json:"roles,omitempty"`
}
```

### roles (tenant-scoped)

```sql
CREATE TABLE roles (
    id         SERIAL PRIMARY KEY,
    tenant_id  INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name       VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

-- Default: (1,super-admin), (2,admin), (3,agent)
```

### role_user (pivot)

```sql
CREATE TABLE role_user (
    id         SERIAL PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id    INTEGER NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, role_id)
);
```

### auth_access_tokens

```sql
CREATE TABLE auth_access_tokens (
    id            SERIAL PRIMARY KEY,
    tokenable_id  INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type          VARCHAR(255) NOT NULL,
    name          VARCHAR(255),
    hash          VARCHAR(255) NOT NULL,
    abilities     TEXT NOT NULL,
    created_at    TIMESTAMPTZ,
    updated_at    TIMESTAMPTZ,
    last_used_at  TIMESTAMPTZ,
    expires_at    TIMESTAMPTZ
);
```

## 2. API Endpoints

All endpoints are tenant-scoped via middleware. Path prefix: `/api`

### Auth (no auth required)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/auth/register` | Register user (default role=agent) |
| POST | `/api/auth/login` | Login, return JWT |

**POST /api/auth/register**
```
Body: { first_name, last_name?, username, email, phone?, password }
Resp 201: { meta: {code, message}, data: User }
Errors: 400 (email/username taken, invalid payload)
```

**POST /api/auth/login**
```
Body: { username, password }
Resp 200: { meta: {code, message}, data: { type: "bearer", token } }
Errors: 401 (invalid credentials)
```

### Auth (token required)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/auth/check-token` | Validate JWT, return user |
| DELETE | `/api/auth/logout` | Logout |

**GET /api/auth/check-token**
```
Header: Authorization: Bearer {token}
Resp 200: { meta: {code, message}, data: User }
Errors: 401
```

### Users (token required + role: super-admin/admin/agent)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/user` | Paginated user list (query: page, limit, search, role_id) |
| GET | `/api/user/:id` | User detail |
| POST | `/api/user` | Create user |
| PUT | `/api/user/:id` | Update user |
| DELETE | `/api/user/:id` | Soft delete user |

**POST /api/user**
```
Body: { first_name, last_name?, username, email, phone?, password?, roles: [int] }
Resp 201: { meta, data: User }
```

**PUT /api/user/:id**
```
Body: { first_name, last_name?, username, email, phone?, password?, roles: [int] }
Resp 200: { meta, data: User }
```

**DELETE /api/user/:id**
```
Resp 200: { meta: {code:200, message: "User deleted succesfuly"} }
Resp 403: { meta: {code:403, message: "You can not delete super admin"} } (id=1)
```

## 3. Tenant Cache Contract

```go
type TenantCache interface {
    Get(ctx context.Context, domain string) (*Tenant, error)
    Set(ctx context.Context, domain string, tenant *Tenant) error
    Clear(ctx context.Context, domain string) error
}
```

- Key: `tenant:{domain}`
- Value: JSON tenant
- TTL: 1 hour

## 4. Migration Files (in order)

| # | File | Description |
|---|------|-------------|
| 05 | `create_tenants_table` | tenants table |
| 06 | `create_roles_table` | roles WITH tenant_id + seed defaults |
| 07 | `create_users_table` | users WITH tenant_id + composite unique |
| 08 | `create_role_user_table` | pivot |
| 09 | `create_auth_access_tokens_table` | tokens |

Seed: default roles (super-admin, admin, agent) per tenant + dummy tenant.

## 5. Implementation Order

1. **Foundation** — Add Redis dependency, update config + DI, drop old tables, create fresh migrations, run
2. **Tenant feature** — entity, repo, cache (`internal/src/tenant/`)
3. **Tenant middleware** — read Host header, look up tenant by domain via Redis → DB fallback, set locals
4. **Update Auth** — add `tenant_id` to entity/repo/service/handler
5. **Update User** — add `tenant_id` to entity/repo/service/handler
6. **Wire up** — tenant middleware globally, auth middleware for user routes
7. **Verify** — `go build ./...`, test isolation between tenants
