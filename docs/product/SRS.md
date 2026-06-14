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

### 1.11 message_raws

| Column | Type | Constraints |
|--------|------|-------------|
| id | SERIAL | PK |
| tenant_id | INTEGER | NOT NULL, FK → tenants.id |
| webhook_message_id_by_external_id | VARCHAR(255) | NOT NULL, UNIQUE |
| data | JSONB | |
| created_at | TIMESTAMPTZ | |
| updated_at | TIMESTAMPTZ | |

Stores raw webhook payload for deduplication and debugging.

### 1.12 tags

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

Base URL: `https://{tenant-domain}/api/v1`

### 2.1 Tenant Onboarding (no tenant middleware)

| Method | Path | Description |
|--------|------|-------------|
| POST | /api/v1/tenants | Create tenant |
| GET | /api/v1/tenants | List tenants |
| GET | /api/v1/tenants/:id | Get tenant |
| PUT | /api/v1/tenants/:id | Update tenant |
| DELETE | /api/v1/tenants/:id | Delete tenant |

### 2.2 Auth

| Method | Path | Description |
|--------|------|-------------|
| POST | /api/v1/auth/register | Register user (first user→super-admin) |
| POST | /api/v1/auth/login | Login, returns token |
| GET | /api/v1/auth/check-token | Validate current token |
| POST | /api/v1/auth/logout | Revoke token |

### 2.3 Users

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/v1/users | List users |
| GET | /api/v1/users/:id | Get user with roles |
| POST | /api/v1/users | Create user |
| PUT | /api/v1/users/:id | Update user |
| DELETE | /api/v1/users/:id | Soft delete (403 if super admin) |

### 2.4 Dashboard

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/v1/dashboard/stats | Get dashboard stats (total contacts, active conversations, resolved today, unassigned count) |
| GET | /api/v1/dashboard/chart | Get chart data (conversations per day for N days) |

### 2.5 Channels (read-only, seeded)

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/v1/channels | List channels |

### 2.6 Contacts

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/v1/contacts | List contacts (paginated, searchable, filterable) |
| GET | /api/v1/contacts/:id | Get contact with profiles |
| POST | /api/v1/contacts | Create contact |
| PUT | /api/v1/contacts/:id | Update contact |
| DELETE | /api/v1/contacts/:id | Soft delete |
| POST | /api/v1/contacts/:id/merge | Merge into target contact |
| POST | /api/v1/contacts/:id/unmerge | Unmerge from parent |
| GET | /api/v1/contacts/:id/conversations | List contact's conversations |

### 2.7 Profiles

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/v1/profiles | List profiles (filterable by contact_id, channel_id) |
| GET | /api/v1/profiles/:id | Get profile |
| PUT | /api/v1/profiles/:id | Update profile (set is_main, linked_device_whatsapp_id) |

### 2.8 Conversations

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/v1/conversations | List conversations (paginated, filterable by status, channel_id, agent_id, assigned/unassigned) |
| GET | /api/v1/conversations/:id | Get conversation with profile, contact, last messages |
| POST | /api/v1/conversations/:id/assign | Assign to current user |
| POST | /api/v1/conversations/:id/unassign | Unassign (set agent_id to null) |
| POST | /api/v1/conversations/:id/resolve | Mark resolved |
| POST | /api/v1/conversations/:id/reopen | Reopen (set to unassigned) |
| POST | /api/v1/conversations | Create conversation (triggered by incoming message or manually) |
| PUT | /api/v1/conversations/:id/read | Mark read (reset unread_count) |

### 2.9 Messages

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/v1/conversations/:id/messages | List messages (paginated, chronological) |
| POST | /api/v1/conversations/:id/messages | Send message (text or file) |
| PUT | /api/v1/messages/:id | Update message status (delivered, read) |

### 2.10 Upload

| Method | Path | Description |
|--------|------|-------------|
| POST | /api/v1/upload | Upload file → storage.solodevs.my.id, returns public_url + metadata |

### 2.11 Tags

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/v1/tags | List tags |
| POST | /api/v1/tags | Create tag |
| PUT | /api/v1/tags/:id | Update tag |
| DELETE | /api/v1/tags/:id | Delete tag |

### 2.12 Notes

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/v1/conversations/:id/notes | List notes for conversation |
| POST | /api/v1/conversations/:id/notes | Create note |
| PUT | /api/v1/notes/:id | Update note |
| DELETE | /api/v1/notes/:id | Delete note |

### 2.13 WhatsApp Devices

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/v1/whatsapp-devices | List devices |
| POST | /api/v1/whatsapp-devices | Create device config |
| PUT | /api/v1/whatsapp-devices/:id | Update device |
| DELETE | /api/v1/whatsapp-devices/:id | Delete device |
| POST | /api/v1/whatsapp-devices/:id/connect | Mock: update status to CONNECTED |
| POST | /api/v1/whatsapp-devices/:id/disconnect | Mock: update status to DISCONNECTED |
| POST | /api/v1/whatsapp-devices/:id/scan | Mock: return mock QR code |

### 2.14 WebSocket

| Event | Direction | Payload |
|-------|-----------|---------|
| conversation.created | → client | `{ type, data: conversation }` |
| conversation.updated | → client | `{ type, data: conversation }` |
| message.created | → client | `{ type, data: message }` |
| message.updated | → client | `{ type, data: message }` |

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
| 7 | MessageRaws | — | 11 |
| 8 | Tags | tenants | 12 |
| 9 | ConversationTag | conversations, tags | 13 |
| 10 | Notes | conversations, users | 14 |
| 11 | WhatsApp Devices | tenants | 15 |
| 12 | Dashboard | contacts, conversations | — (query-only) |
| 13 | Upload | — | — (external service) |
| 14 | WebSocket | — | — (infrastructure) |
