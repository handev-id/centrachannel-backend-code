# PRD — CentraChannel API

Multi-tenant omnichannel customer communication platform. Each tenant (business/client) owns its own data, isolated via `tenant_id`.

## Tenant Isolation

```
business-a.centrachannel.com  → tenant_id = 1
business-b.centrachannel.com  → tenant_id = 2
```

Resolution: domain → Redis cache → DB fallback. All tenant-scoped queries use `WHERE tenant_id = $N`.

## MVP Scope

| Feature | Prio | Notes |
|---------|------|-------|
| Tenant resolution | P0 | Domain → Redis → DB middleware |
| Register | P0 | User registers within a tenant, default role=agent |
| Login | P0 | JWT token |
| Check token | P0 | Validate JWT + return user |
| Logout | P0 | Delete token |
| CRUD User | P1 | List, get, create, update, delete (soft) |

## Roles Per Tenant

super-admin (id=1) → full access, cannot be deleted
admin (id=2) → manage users
agent (id=3) → default role

## Constraints

- All tenant-scoped queries MUST filter by `tenant_id`
- Super admin (user id=1) within a tenant cannot be deleted (403)
- Soft delete via `deleted_at`
- No cross-tenant data leak
- Protected routes require JWT Bearer token
