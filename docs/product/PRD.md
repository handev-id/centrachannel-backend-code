# PRD — CentraChannel API

Multi-tenant omnichannel customer communication platform. Each tenant (business) manages contacts, conversations, and users across multiple channels (WhatsApp, Facebook, Instagram).

## Tenant Isolation

```
business-a.centrachannel.com  → tenant_id = 1
business-b.centrachannel.com  → tenant_id = 2
```

Resolution: domain header → Redis cache → DB fallback (full domain stored in `tenants.domain`). All tenant-scoped queries use `WHERE tenant_id = $N`.

## Channels (Global)

Channels are global records shared across all tenants (no `tenant_id`):

| Type | Description |
|------|-------------|
| facebook | Facebook Page integration |
| instagram | Instagram Professional integration |
| whatsapp_business | Official WhatsApp Business API via Evolution API (Meta Cloud API) |
| whatsapp | Unofficial WhatsApp Web via Evolution API (Baileys) |

## Roles

Roles are per-tenant, seeded on tenant creation as regular DB records:

| Role | Description |
|------|-------------|
| super-admin | Full access within tenant |
| admin | Manage users, view all conversations |
| agent | Default role, can handle conversations |

Super admin is a role like admin/agent — not a separate admin panel.

## Conversations

- Only `customer (contact from channel) ↔ user (agent/admin/super-admin)` communication
- Statuses: `unassigned` | `assigned` | `resolved`
- Assignment is manual pick (not auto-routed)
- Messages include text, image, file attachments (stored as JSONB)

## Contact Merge

When contacts are merged:
- `contacts.merged_to_id` points to the surviving contact
- Profiles of merged contacts get `merged_from_contact_id` set
- Merged contacts are soft-deleted
- Unmerge restores original contact and clears profile references

## Features by Phase

### Phase 1 — Foundation (Done)

- Tenant resolution (domain → Redis → DB)
- Tenant registration via main domain (`internal/app/registration`)
- Tenant CRUD via subdomain (`internal/src/tenant`, admin-only)
- Auth: register, login, check-token, logout
- User CRUD: list, show, create, update, soft delete (super admin id=1 protected)
- Channel seed: facebook, instagram, whatsapp_business, whatsapp
- Dashboard stats: total contacts, active conversations, resolved today, unassigned count
- Avatar generation via DiceBear initials

### Phase 2 — Conversations

- Contact CRUD + merge/unmerge
- Profile management per channel (external_id, is_main, linked_device)
- Conversation: list, assign, unassign, resolve, reopen, read
- Message: send (text/file), receive, list (paginated), update status
- File upload to `https://storage.solodevs.my.id` (multipart → public_url)
- Tags CRUD (per-tenant, name + color)
- Notes on conversations (text + date)
- WhatsApp device management (connect/disconnect/scan via Evolution API, real QR codes)

### Phase 3 — Campaigns (future)

- Campaign/broadcast (WhatsApp only)
- Template management (name, type, content, variables)
- Recipient list management (manual upload, source tracking)
- Contact import/export
- All campaign tables get `tenant_id`

## Constraints

- All tenant-scoped queries MUST filter by `tenant_id`
- Contact has its own `tenant_id` (not inherited from profile)
- Super admin (user id=1 within tenant) not deletable (403)
- Soft delete via `deleted_at` (contacts, users, profiles)
- Soft delete queries filter `WHERE deleted_at IS NULL`
- No cross-tenant data leak
- Protected routes require JWT Bearer token
- Channels are global (shared across all tenants)
