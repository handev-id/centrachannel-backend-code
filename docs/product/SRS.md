# SRS — CentraChannel API

## 1. Data Model

### 1.1 tenants

| Column | Type | Constraints |
|--------|------|-------------|
| id | SERIAL | PK |
| name | VARCHAR(255) | NOT NULL |
| domain | VARCHAR(255) | NOT NULL, UNIQUE |
| logo | JSONB | |
| address | VARCHAR(255) | |
| phone | VARCHAR(255) | |
| email | VARCHAR(255) | |
| is_active | BOOLEAN | DEFAULT true |
| settings | JSONB | |
| created_at | TIMESTAMPTZ | NOT NULL |
| updated_at | TIMESTAMPTZ | NOT NULL |

### 1.2 roles

| Column | Type | Constraints |
|--------|------|-------------|
| id | SERIAL | PK |
| tenant_id | INTEGER | NOT NULL, FK → tenants.id |
| name | VARCHAR(255) | NOT NULL |
| created_at | TIMESTAMPTZ | |
| updated_at | TIMESTAMPTZ | |

UNIQUE(tenant_id, name). Seeded at tenant creation: super-admin, admin, agent.

### 1.3 users

| Column | Type | Constraints |
|--------|------|-------------|
| id | SERIAL | PK |
| tenant_id | INTEGER | NOT NULL, FK → tenants.id |
| first_name | VARCHAR(255) | NOT NULL |
| last_name | VARCHAR(255) | |
| username | VARCHAR(255) | NOT NULL |
| email | VARCHAR(255) | NOT NULL |
| phone | VARCHAR(255) | |
| password | VARCHAR(255) | NOT NULL |
| avatar | JSONB | |
| last_login | TIMESTAMPTZ | |
| deleted_at | TIMESTAMPTZ | |
| created_at | TIMESTAMPTZ | NOT NULL |
| updated_at | TIMESTAMPTZ | NOT NULL |

UNIQUE(tenant_id, email). Soft delete.

### 1.4 role_user

| Column | Type | Constraints |
|--------|------|-------------|
| id | SERIAL | PK |
| tenant_id | INTEGER | NOT NULL, FK → tenants.id |
| user_id | INTEGER | FK → users.id ON DELETE CASCADE |
| role_id | INTEGER | FK → roles.id ON DELETE CASCADE |
| created_at | TIMESTAMPTZ | |
| updated_at | TIMESTAMPTZ | |

### 1.5 auth_access_tokens

| Column | Type | Constraints |
|--------|------|-------------|
| id | SERIAL | PK |
| tokenable_id | INTEGER | NOT NULL, FK → users.id ON DELETE CASCADE |
| type | VARCHAR(255) | NOT NULL |
| name | VARCHAR(255) | |
| hash | VARCHAR(255) | NOT NULL |
| abilities | TEXT | NOT NULL |
| created_at | TIMESTAMPTZ | |
| updated_at | TIMESTAMPTZ | |
| last_used_at | TIMESTAMPTZ | |
| expires_at | TIMESTAMPTZ | |

### 1.6 channels (global — no tenant_id)

| Column | Type | Constraints |
|--------|------|-------------|
| id | SERIAL | PK |
| name | VARCHAR(255) | NOT NULL |
| type | VARCHAR(255) | NOT NULL, UNIQUE |
| logo | JSONB | |
| created_at | TIMESTAMPTZ | |
| updated_at | TIMESTAMPTZ | |

Seeded values: `facebook`, `instagram`, `whatsapp_business`, `whatsapp`.

### 1.7 contacts

| Column | Type | Constraints |
|--------|------|-------------|
| id | SERIAL | PK |
| tenant_id | INTEGER | NOT NULL, FK → tenants.id |
| first_name | VARCHAR(255) | NOT NULL |
| last_name | VARCHAR(255) | |
| username | VARCHAR(255) | |
| email | VARCHAR(255) | |
| phone | VARCHAR(255) | |
| avatar | JSONB | |
| country | VARCHAR(255) | |
| bio | TEXT | |
| occupation | VARCHAR(255) | |
| category | VARCHAR(255) | |
| category_description | TEXT | |
| gender | VARCHAR(255) | |
| date_of_birth | DATE | |
| province_of_origin | VARCHAR(255) | |
| facebook | VARCHAR(255) | |
| instagram | VARCHAR(255) | |
| whatsapp | VARCHAR(255) | |
| x | VARCHAR(255) | |
| tiktok | VARCHAR(255) | |
| status | TEXT | NOT NULL DEFAULT 'individual', CHECK(individual, institution) |
| institution_name | VARCHAR(255) | |
| merged_to_id | INTEGER | FK → contacts.id ON DELETE SET NULL |
| deleted_at | TIMESTAMPTZ | |
| created_at | TIMESTAMPTZ | NOT NULL |
| updated_at | TIMESTAMPTZ | NOT NULL |

Soft delete. Merge: `merged_to_id` points to the surviving contact.

### 1.8 profiles

| Column | Type | Constraints |
|--------|------|-------------|
| id | SERIAL | PK |
| external_id | VARCHAR(255) | NOT NULL |
| username | VARCHAR(255) | |
| display_name | VARCHAR(255) | |
| is_main | BOOLEAN | DEFAULT true |
| linked_device_whatsapp_id | VARCHAR(255) | |
| merged_from_contact_id | INTEGER | FK → contacts.id ON DELETE SET NULL |
| contact_id | INTEGER | NOT NULL, FK → contacts.id ON DELETE CASCADE |
| channel_id | INTEGER | NOT NULL, FK → channels.id ON DELETE CASCADE |
| deleted_at | TIMESTAMPTZ | |
| created_at | TIMESTAMPTZ | NOT NULL |
| updated_at | TIMESTAMPTZ | NOT NULL |

UNIQUE(channel_id, external_id) WHERE deleted_at IS NULL. A contact can have multiple profiles (one per channel). `is_main` indicates the primary profile.

### 1.9 conversations

| Column | Type | Constraints |
|--------|------|-------------|
| id | SERIAL | PK |
| tenant_id | INTEGER | NOT NULL, FK → tenants.id |
| status | TEXT | NOT NULL, CHECK(unassigned, assigned, resolved) |
| profile_id | INTEGER | NOT NULL, FK → profiles.id ON DELETE CASCADE |
| agent_id | INTEGER | FK → users.id ON DELETE SET NULL |
| channel_id | INTEGER | NOT NULL, FK → channels.id ON DELETE CASCADE |
| last_agent_id | INTEGER | FK → users.id ON DELETE SET NULL |
| unread_count | INTEGER | NOT NULL DEFAULT 0 |
| last_message | JSONB | |
| last_activity | TIMESTAMPTZ | |
| last_seen | TIMESTAMPTZ | |
| created_at | TIMESTAMPTZ | NOT NULL |
| updated_at | TIMESTAMPTZ | NOT NULL |

### 1.10 messages

| Column | Type | Constraints |
|--------|------|-------------|
| id | SERIAL | PK |
| tenant_id | INTEGER | NOT NULL, FK → tenants.id |
| text | TEXT | |
| attachment | JSONB | |
| status | TEXT | NOT NULL, CHECK(sent, delivered, read, unread) |
| sender_id | INTEGER | NOT NULL |
| sender_type | TEXT | NOT NULL, CHECK(contact, user, ai) |
| webhook_message_id | VARCHAR(255) | |
| webhook_message_reply_id | VARCHAR(255) | |
| conversation_id | INTEGER | NOT NULL, FK → conversations.id ON DELETE CASCADE |
| created_at | TIMESTAMPTZ | NOT NULL |
| updated_at | TIMESTAMPTZ | NOT NULL |

Attachment JSONB structure:
```json
{
  "type": "image|file",
  "url": "https://storage.solodevs.my.id/...",
  "name": "filename.pdf",
  "size": 12345,
  "mime": "application/pdf"
}
```

### 1.11 tags

| Column | Type | Constraints |
|--------|------|-------------|
| id | SERIAL | PK |
| tenant_id | INTEGER | NOT NULL, FK → tenants.id |
| name | VARCHAR(255) | NOT NULL |
| color | VARCHAR(255) | |
| created_at | TIMESTAMPTZ | NOT NULL |
| updated_at | TIMESTAMPTZ | NOT NULL |

UNIQUE(tenant_id, name).

### 1.13 conversation_tag

| Column | Type | Constraints |
|--------|------|-------------|
| id | SERIAL | PK |
| tenant_id | INTEGER | NOT NULL, FK → tenants.id |
| conversation_id | INTEGER | FK → conversations.id ON DELETE CASCADE |
| tag_id | INTEGER | FK → tags.id ON DELETE CASCADE |
| created_at | TIMESTAMPTZ | NOT NULL |
| updated_at | TIMESTAMPTZ | NOT NULL |

UNIQUE(conversation_id, tag_id).

### 1.14 notes

| Column | Type | Constraints |
|--------|------|-------------|
| id | SERIAL | PK |
| tenant_id | INTEGER | NOT NULL, FK → tenants.id |
| text | TEXT | NOT NULL |
| date | DATE | |
| conversation_id | INTEGER | FK → conversations.id ON DELETE CASCADE |
| user_id | INTEGER | FK → users.id ON DELETE SET NULL |
| created_at | TIMESTAMPTZ | NOT NULL |
| updated_at | TIMESTAMPTZ | NOT NULL |

### 1.15 whatsapp_devices

| Column | Type | Constraints |
|--------|------|-------------|
| id | SERIAL | PK |
| tenant_id | INTEGER | NOT NULL, FK → tenants.id |
| name | VARCHAR(255) | NOT NULL |
| country_code | VARCHAR(255) | NOT NULL DEFAULT '62' |
| phone | VARCHAR(255) | NOT NULL |
| whatsapp_id | VARCHAR(255) | NOT NULL |
| status | TEXT | NOT NULL DEFAULT 'DISCONNECTED', CHECK(CONNECTED, DISCONNECTED) |
| created_at | TIMESTAMPTZ | |
| updated_at | TIMESTAMPTZ | |

UNIQUE(tenant_id, whatsapp_id).

---

## 2. API Endpoints

Two access layers based on domain:

| Layer | Base URL | Tenant Context | Auth |
|-------|----------|----------------|------|
| Main domain | `https://centrachannel.com/api` | No (resolved from payload) | API key (webhook) |
| Subdomain | `https://{tenant}.centrachannel.com/api` | Yes (from Host header) | JWT + Role |

### 2.1 Tenant Registration (main domain, no tenant middleware)

| Method | Path | Description |
|--------|------|-------------|
| POST | /api/tenants/onboard | Register new tenant + admin user |

### 2.2 Tenant Management (subdomain, admin-only)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /api/tenants | JWT, admin | List tenants |
| GET | /api/tenants/:id | JWT, admin | Get tenant detail |

### 2.3 Auth (subdomain)

| Method | Path | Description |
|--------|------|-------------|
| POST | /api/auth/register | Register user (default role=agent) |
| POST | /api/auth/login | Login, returns JWT |
| GET | /api/auth/check-token | Validate current token |
| DELETE | /api/auth/logout | Revoke token |

### 2.4 Users (subdomain)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /api/user | JWT, admin | List users |
| GET | /api/user/:id | JWT, admin | Get user with roles |
| POST | /api/user | JWT, admin | Create user |
| PUT | /api/user/:id | JWT, admin | Update user |
| DELETE | /api/user/:id | JWT, admin | Soft delete (403 if super admin) |

### 2.5 Dashboard (subdomain)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /api/dashboard/stats | JWT, agent+ | Get dashboard stats |
| GET | /api/dashboard/chart | JWT, agent+ | Get chart data |

### 2.6 Channels (subdomain, read-only, seeded)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /api/channels | JWT, agent+ | List channels |

### 2.7 Contacts (subdomain)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /api/contacts | JWT, agent+ | List contacts (paginated, searchable) |
| GET | /api/contacts/:id | JWT, agent+ | Get contact with profiles |
| POST | /api/contacts | JWT, agent+ | Create contact |
| PUT | /api/contacts/:id | JWT, agent+ | Update contact |
| DELETE | /api/contacts/:id | JWT, agent+ | Soft delete |
| POST | /api/contacts/:id/merge | JWT, agent+ | Merge into target contact |
| POST | /api/contacts/:id/unmerge | JWT, agent+ | Unmerge from parent |
| GET | /api/contacts/:id/conversations | JWT, agent+ | List contact's conversations |
| GET | /api/contacts/export | JWT, agent+ | Export contacts |
| POST | /api/contacts/import | JWT, agent+ | Import contacts |

### 2.8 Conversations (subdomain)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /api/conversations | JWT, agent+ | List conversations (paginated, filterable) |
| GET | /api/conversations/:id | JWT, agent+ | Get conversation detail |
| POST | /api/conversations | JWT, agent+ | Create conversation |
| POST | /api/conversations/:id/assign | JWT, agent+ | Assign to current user |
| POST | /api/conversations/:id/unassign | JWT, agent+ | Unassign |
| POST | /api/conversations/:id/resolve | JWT, agent+ | Mark resolved |
| POST | /api/conversations/:id/reopen | JWT, agent+ | Reopen |
| PUT | /api/conversations/:id/read | JWT, agent+ | Mark read |
| GET | /api/conversations/unread | JWT, agent+ | Get unread count |

### 2.9 Messages (subdomain)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /api/conversations/:id/messages | JWT, agent+ | List messages (paginated) |
| POST | /api/conversations/:id/messages | JWT, agent+ | Send message (text or file) |
| PUT | /api/messages/:id | JWT, agent+ | Update message status |

### 2.10 Tags (subdomain)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /api/tags | JWT, agent+ | List tags |
| POST | /api/tags | JWT, agent+ | Create tag |
| PUT | /api/tags/:id | JWT, agent+ | Update tag |
| DELETE | /api/tags/:id | JWT, agent+ | Delete tag |

### 2.11 Notes (subdomain)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /api/conversations/:id/notes | JWT, agent+ | List notes for conversation |
| POST | /api/conversations/:id/notes | JWT, agent+ | Create note |
| PUT | /api/notes/:id | JWT, agent+ | Update note |
| DELETE | /api/notes/:id | JWT, agent+ | Delete note |

### 2.12 WhatsApp Devices (subdomain)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /api/whatsapp-devices | JWT, agent+ | List devices |
| POST | /api/whatsapp-devices | JWT, agent+ | Create device |
| GET | /api/whatsapp-devices/:id | JWT, agent+ | Get device |
| PUT | /api/whatsapp-devices/:id | JWT, agent+ | Update device |
| DELETE | /api/whatsapp-devices/:id | JWT, agent+ | Delete device |
| POST | /api/whatsapp-devices/:id/connect | JWT, agent+ | Connect (via Evolution API) |
| POST | /api/whatsapp-devices/:id/disconnect | JWT, agent+ | Disconnect |
| POST | /api/whatsapp-devices/:id/scan | JWT, agent+ | Get QR code |

### 2.13 Upload (subdomain)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | /api/upload | JWT, agent+ | Upload file |

### 2.14 SSE (Server-Sent Events)

| Event | Direction | Description |
|-------|-----------|-------------|
| `connected` | → client | Initial connection confirmation with `user_id` |
| `message:new` | → client | New message (from webhook or user send) |
| `conversation:updated` | → client | Conversation status change (assign/unassign/resolve/reopen) |
| `device:updated` | → client | WhatsApp device status change |
| `user:online` | → client | User came online (first connection) |
| `user:offline` | → client | User went offline (last disconnection) |

**Endpoint:** `GET /event` (auth required, tenant-scoped)

**Format:** Standard SSE (`text/event-stream`), each event includes `event:` name and `data:` JSON payload.

---

## 3. Error Response Format

```json
{
  "success": false,
  "message": "Error description",
  "errors": {
    "field": ["validation error 1", "validation error 2"]
  }
}
```

## 4. Success Response Format

```json
{
  "success": true,
  "message": "Success message",
  "data": { ... }
}
```

Paginated:
```json
{
  "success": true,
  "message": "Success",
  "data": [...],
  "meta": {
    "page": 1,
    "per_page": 10,
    "total": 100,
    "total_pages": 10
  }
}
```

---

## 5. Implementation Order

| # | Feature | Dependencies | Est. Migrations |
|---|---------|-------------|-----------------|
| 1 | Tenants, Roles, Users, Auth, RoleUser | — | 01–05 (done) |
| 2 | Channels seed | — | 06 |
| 3 | Contacts | tenants | 07 |
| 4 | Profiles | contacts, channels | 08 |
| 5 | Conversations | profiles, channels, users | 09 |
| 6 | Messages | conversations | 10 |
| 7 | Tags | tenants | 12 |
| 8 | ConversationTag | conversations, tags | 13 |
| 9 | Notes | conversations, users | 14 |
| 10 | WhatsApp Devices | tenants | 15 |
| 11 | Campaigns | tenants | 16–17 |
| 12 | Dashboard | contacts, conversations | — (query-only) |
| 13 | Upload | — | — (external service) |
| 14 | SSE | — | — (real-time events) |
