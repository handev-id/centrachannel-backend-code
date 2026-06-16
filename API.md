# CentraChannel API Reference

Base URL: `https://{tenant}.centrachannel.com/api`

Auth: `Bearer` JWT token in `Authorization` header.

---

## Response Envelope

```json
{
  "meta": { "code": 200, "message": "Success" },
  "data": { ... },
  "errors": null
}
```

### Offset Pagination (default)

Used by most endpoints (contacts, users, campaigns, etc.).
```json
{
  "meta": { "code": 200, "message": "Success" },
  "data": { "meta": { "total": 50, "per_page": 20, "current_page": 1, "last_page": 3, "from": 1, "to": 20 }, "data": [...] }
}
```

### Cursor Pagination (Conversations & Messages)

Used by `GET /conversations` and `GET /conversations/{id}/messages` when `last_id` is provided.
```json
{
  "meta": { "code": 200, "message": "Success" },
  "data": { "meta_pagination": { "last_id": 42, "has_more": true }, "data": [...] }
}
```

For conversations, `meta_pagination` also includes `last_activity` (ISO 8601 timestamp) for composite cursor positioning.

---

## Observability (Public — No Auth)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health status |
| GET | `/ping` | Returns `pong` |
| GET | `/version` | Version info |
| GET | `/sse-test-ping` | Send test ping via SSE to a tenant. Query: `tenant_id` (required) |

## Webhooks (Public)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/webhook/evolution` | Evolution API webhook |
| GET | `/webhook/meta` | Meta webhook verification |
| POST | `/webhook/meta` | Meta webhook handler |

## SSE (Server-Sent Events)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/event` | SSE event stream (auth required, use `?token=` query param) |

### Events

The SSE endpoint pushes real-time events in standard SSE format. A **heartbeat ping** is sent every 5 seconds so the client can verify connectivity:

```text
event: connected
data: {"user_id":5}

event: ping
data: {}

event: user:online
data: {"id":3}

event: user:offline
data: {"id":3}

event: message:new
data: {"id":1,"text":"Hello","sender_type":"contact",...}

event: conversation:updated
data: {"id":1,"action":"assign","agent_id":2}
```

| Event | Description |
|-------|-------------|
| `connected` | Initial connection confirmation with `user_id` |
| `ping` | Heartbeat every 5s. FE can listen to this to detect stale connections |
| `user:online` | Broadcast to **all** connected clients (including sender) when a user comes online |
| `user:offline` | Broadcast to remaining clients when a user disconnects |
| `message:new` | New message in a conversation |
| `conversation:updated` | Conversation status change (assign/unassign/resolve/reopen) |

**Initial presence state:** Fetch via `GET /api/user` — response includes `is_online` field for each user.

Client usage (JavaScript):
```js
const evtSource = new EventSource('/api/event?token=...');
evtSource.addEventListener('message:new', (e) => {
  const msg = JSON.parse(e.data);
  // update UI
});
```

---

## Auth (Public — No Tenant Required)

### POST /api/auth/register
Register a new user for the current tenant.

**Request:**
```json
{
  "first_name": "John",
  "last_name": "Doe",
  "username": "johndoe",
  "email": "john@example.com",
  "phone": "6281234567890",
  "password": "password123",
  "avatar": { ... }
}
```

### POST /api/auth/login
Login and get JWT token.

**Request:**
```json
{ "username": "johndoe", "password": "password123" }
```

**Response:**
```json
{ "token": "eyJ...", "expires_at": "2025-08-05T20:52:26.949Z" }
```

### GET /api/auth/check-token
Check if current token is valid. Requires auth.

### DELETE /api/auth/logout
Invalidate current token. Requires auth.

---

## Tenant

### POST /api/tenants/onboard
Register a new tenant (public, no auth).

**Request:**
```json
{
  "name": "Acme Corp",
  "domain": "acme.com",
  "admin_email": "admin@acme.com",
  "admin_password": "adminpass123",
  "company_name": "Acme Corporation"
}
```

### GET /api/tenants
List all tenants. Requires super-admin role.

### GET /api/tenants/{id}
Get tenant by ID. Requires admin+ role.

---

## User (Requires Auth)

### GET /api/user
List users. Paginated. Requires super-admin/admin.

**Query params:** `page`, `limit`, `search`, `role_id`, `sort_by`

### POST /api/user
Create user. Requires super-admin/admin.

**Request:**
```json
{
  "first_name": "Jane",
  "last_name": "Doe",
  "username": "janedoe",
  "email": "jane@example.com",
  "phone": "6281234567890",
  "password": "password123",
  "avatar": { ... },
  "roles": [2, 3]
}
```

### GET /api/user/{id}
Get user details with roles.

### PUT /api/user/{id}
Update user. Super admin (id=1) cannot be deleted.

**Request:** Same fields as create, all optional.

### DELETE /api/user/{id}
Soft-delete user. Returns 403 for super admin.

---

## Campaign (Requires Auth)

### GET /api/campaigns
List campaigns. Paginated. Query: `page`, `limit`, `search`

### POST /api/campaigns
Create campaign. Required: `name`, `message_template`, `channel_id`

**Request fields:** `name`, `type`, `description`, `message_template`, `channel_id`, `sending_option`, `scheduled_at`, `recipient_list_id`, `template_id`

### GET /api/campaigns/{id}
Get campaign details.

### PUT /api/campaigns/{id}
Update campaign. Fields: `name`, `type`, `description`, `message_template`, `status`, `sending_option`, `scheduled_at`, `recipient_list_id`, `template_id`, `agent_id`, `sender_id`

### DELETE /api/campaigns/{id}
Delete campaign.

### POST /api/campaigns/{id}/send
Trigger sending campaign.

### Campaign Templates

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/campaigns/templates` | List templates (paginated) |
| POST | `/api/campaigns/templates` | Create template |
| GET | `/api/campaigns/templates/{id}` | Get template |
| PUT | `/api/campaigns/templates/{id}` | Update template |
| DELETE | `/api/campaigns/templates/{id}` | Delete template |

**Template request:** `name`, `type`, `template_type`, `category`, `language`, `content` (JSON object), `variables`, `quality`, `account_id`

### Recipient Lists

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/campaigns/recipient-lists` | List (paginated) |
| POST | `/api/campaigns/recipient-lists` | Create. Required: `name` |
| GET | `/api/campaigns/recipient-lists/{id}` | Get |
| PUT | `/api/campaigns/recipient-lists/{id}` | Update |
| DELETE | `/api/campaigns/recipient-lists/{id}` | Delete |
| GET | `/api/campaigns/recipient-lists/{id}/contacts` | List contacts in list (paginated) |
| POST | `/api/campaigns/recipient-lists/{id}/contacts` | Add contact. Body: `first_name`, `phone` (required) |
| DELETE | `/api/campaigns/recipient-lists/{id}/contacts/{contactId}` | Remove contact from list |

---

## Channels

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/channels` | List all channels |

---

## Contact (Requires Auth)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/contacts` | List contacts (paginated). Query: `page`, `limit`, `search`, `channel_id`, `status` |
| POST | `/api/contacts` | Create contact |
| GET | `/api/contacts/{id}` | Get contact with profiles |
| PUT | `/api/contacts/{id}` | Update contact |
| DELETE | `/api/contacts/{id}` | Soft-delete contact |
| POST | `/api/contacts/{id}/merge` | Merge duplicate contacts. Body: `{ "target_contact_id": 2 }` |
| POST | `/api/contacts/{id}/unmerge` | Unmerge contacts |
| GET | `/api/contacts/{id}/conversations` | Get conversations for contact |
| GET | `/api/contacts/export` | Export contacts as CSV |
| POST | `/api/contacts/import` | Import contacts via CSV |

---

## Conversation (Requires Auth)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/conversations` | List conversations (paginated). Query: `page`, `limit`, `status`, `channel_id`, `search`, `agent_id`, `tag`. For cursor pagination: `last_activity` (ISO 8601), `last_id` |
| POST | `/api/conversations` | Create conversation |
| GET | `/api/conversations/{id}` | Get conversation with contact, channel, agent, tags, notes, latest messages |
| POST | `/api/conversations/{id}/assign` | Assign to agent. Body: `{ "agent_id": 1 }` |
| POST | `/api/conversations/{id}/unassign` | Unassign conversation |
| POST | `/api/conversations/{id}/resolve` | Mark as resolved |
| POST | `/api/conversations/{id}/reopen` | Reopen conversation |
| GET | `/api/conversations/unread` | Count unread conversations |
| PUT | `/api/conversations/{id}/read` | Mark as read |

---

## Message (Requires Auth)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/conversations/{conversationId}/messages` | List messages in conversation (paginated). Query: `page`, `limit`. For cursor pagination: `last_id` |
| POST | `/api/conversations/{conversationId}/messages` | Send message |
| PUT | `/api/messages/{id}` | Update message status. Body: `{ "status": "read" }` |

---

## Tags (Requires Auth)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/tags` | List tags. Query: `search` |
| POST | `/api/tags` | Create tag. Body: `{ "name": "...", "color": "#FF5733" }` |
| PUT | `/api/tags/{id}` | Update tag |
| DELETE | `/api/tags/{id}` | Delete tag |

### Conversation Tags

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/conversations/{id}/tags` | List tags on a conversation |
| POST | `/api/conversations/{id}/tags` | Attach tag. Body: `{ "tag_id": 1 }` |
| DELETE | `/api/conversations/{id}/tags/{tagId}` | Detach tag from conversation |

---

## Notes (Requires Auth)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/conversations/{conversationId}/notes` | List notes. Query: `page`, `limit` |
| POST | `/api/conversations/{conversationId}/notes` | Create note. Body: `{ "text": "...", "date": "2025-07-29" }` |
| PUT | `/api/notes/{id}` | Update note |
| DELETE | `/api/notes/{id}` | Delete note |

---

## WhatsApp Device (Requires Auth)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/whatsapp-devices` | List devices. Query: `status` |
| POST | `/api/whatsapp-devices` | Create device. Body: `{ "name": "...", "phone": "...", "country_code": "62" }` |
| GET | `/api/whatsapp-devices/{id}` | Get device |
| PUT | `/api/whatsapp-devices/{id}` | Update device |
| DELETE | `/api/whatsapp-devices/{id}` | Delete device |
| POST | `/api/whatsapp-devices/{id}/connect` | Connect device (returns pairing code) |
| POST | `/api/whatsapp-devices/{id}/disconnect` | Disconnect device |
| POST | `/api/whatsapp-devices/{id}/scan` | Get QR code for scanning |

---

## Upload (Requires Auth)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/upload` | Upload file. Multipart form: `file` field |

---

## Dashboard (Requires Auth)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/dashboard/` | Comprehensive analytics: summary, conversations, contacts, messages, campaigns, agents |
| GET | `/api/dashboard/chart` | Conversation trend chart |

**Query params:** `days` (integer, 1-365, default 30) — rolling window for new/new metrics, trends, and message history.

### Response Structure

```json
{
  "summary": {
    "total_contacts": 3210,
    "new_contacts": 54,
    "total_conversations": 1276,
    "active_conversations": 60,
    "new_conversations": 121,
    "resolved_today": 15,
    "total_unread_conversations": 37,
    "total_unread_messages": 90,
    "total_messages": 15420,
    "messages_today": 235,
    "total_campaigns": 42,
    "connected_whatsapp_devices": 3
  },
  "conversations": {
    "by_status": { "unassigned": 19, "assigned": 41, "resolved": 1216 },
    "by_channel": [
      { "channel_id": 1, "channel_name": "whatsapp", "total": 978, "unread": 65 }
    ],
    "recent": [
      {
        "id": 1001, "status": "assigned", "unread_count": 2,
        "last_activity": "2026-06-15T13:40:00Z",
        "last_message": { "text": "Hello!" },
        "channel": { "id": 1, "name": "whatsapp", "logo": null },
        "agent": { "id": 9, "first_name": "Alya", "last_name": "Putri", "avatar": null },
        "contact": { "id": 44, "first_name": "Rafi", "last_name": "Saputra", "avatar": null }
      }
    ],
    "trend": [
      { "date": "2026-06-15", "count": 15 }
    ]
  },
  "contacts": {
    "by_status": { "individual": 3021, "institution": 189 },
    "new_contacts": 54
  },
  "messages": {
    "total": 15420,
    "today": 235,
    "by_sender_type": { "contact": 10200, "user": 5220 },
    "by_date": [
      { "date": "2026-06-15", "count": 235 }
    ]
  },
  "campaigns": {
    "total": 42,
    "by_status": { "draft": 3, "inprogress": 2, "completed": 7 }
  },
  "agents": [
    { "id": 9, "first_name": "Alya", "last_name": "Putri", "avatar": null, "ongoing_conversations": 16 }
  ]
}
```

### User
```json
{
  "id": 1, "tenant_id": 1,
  "first_name": "John", "last_name": "Doe",
  "username": "johndoe", "email": "john@example.com",
  "phone": "6281234567890",
  "avatar": { "name": "avatar.png", "url": "..." },
  "last_login": "2025-07-29T15:00:00.000Z",
  "deleted_at": null,
  "is_online": true,
  "created_at": "...", "updated_at": "...",
  "roles": [{ "id": 1, "name": "agent" }]
}
```

### Role
`id`, `tenant_id`, `name`, `created_at`, `updated_at`

### Tenant
`id`, `name`, `domain`, `logo`, `address`, `phone`, `email`, `is_active`, `settings` (JSON — contains `channel_configuration` with `meta_access_token` & `whatsapp_phone_id`), `created_at`, `updated_at`

### Channel
`id`, `name`, `type` (facebook|instagram|whatsapp_business|whatsapp), `logo`, `created_at`, `updated_at`

### Profile
`id`, `external_id`, `username`, `display_name`, `is_main`, `linked_device_whatsapp_id`, `contact_id`, `channel_id`, `deleted_at`, `created_at`, `updated_at`

### Contact
```json
{
  "id": 1, "tenant_id": 1,
  "first_name": "Jane", "last_name": "Smith",
  "username": "janesmith", "email": "jane@example.com",
  "phone": "6281234567890",
  "avatar": {}, "country": "Indonesia", "bio": "...",
  "occupation": "Marketing Manager",
  "category": "business", "gender": "female",
  "date_of_birth": "1990-05-15",
  "province_of_origin": "Jawa Barat",
  "facebook": "...", "instagram": "...",
  "whatsapp": "...", "x": "...", "tiktok": "...",
  "status": "individual|institution",
  "institution_name": "Tech Corp",
  "merged_to_id": null, "deleted_at": null,
  "created_at": "...", "updated_at": "...",
  "profiles": [{ ... }]
}
```

### Conversation
```json
{
  "id": 1, "tenant_id": 1,
  "status": "unassigned|assigned|resolved",
  "profile_id": 1, "agent_id": 1,
  "channel_id": 1, "last_agent_id": 1,
  "unread_count": 3,
  "last_message": { "text": "Hello!", "sender_type": "contact" },
  "contact": { ... }, "channel": { ... },
  "agent": { ... },
  "tags": [{ ... }], "notes": [{ ... }]
}
```

### Message
```json
{
  "id": 1, "tenant_id": 1,
  "conversation_id": 1,
  "sender_id": 1, "sender_type": "contact|user|ai",
  "text": "Hello, I need help",
  "attachment": { "name": "file.pdf", "url": "...", "type": "application/pdf" },
  "status": "sent|delivered|read|failed",
  "webhook_message_id": "wamid.123456789",
  "sender": { "id": 1, "first_name": "Jane", "avatar": {} }
}
```

### Tag
`id`, `tenant_id`, `name`, `color` (hex), `created_at`, `updated_at`

### Note
`id`, `tenant_id`, `conversation_id`, `user_id`, `text`, `date`, `deleted_at`, `created_at`, `updated_at`, `user`: { ... }

### Campaign
```json
{
  "id": 1, "tenant_id": 1,
  "name": "Welcome Campaign", "type": "broadcast",
  "description": "...", "message_template": "Hello {{name}}...",
  "channel_id": 1,
  "status": "draft|scheduled|sending|sent|failed",
  "sending_option": {}, "stats": { "sent": 45, "failed": 2 },
  "scheduled_at": null,
  "sent_count": 0, "total_count": 100,
  "agent_id": 1, "sender_id": 1, "recipient_list_id": 1,
  "template_id": 1, "created_by": 1
}
```

### CampaignTemplate
`id`, `tenant_id`, `name`, `type`, `template_type`, `category`, `language`, `content` (JSON), `variables` (JSON), `quality`, `account_id`, `created_at`, `updated_at`

### CampaignRecipientList
`id`, `tenant_id`, `name`, `source`, `status`, `channel_id`, `created_at`, `updated_at`

### WhatsAppDevice
`id`, `tenant_id`, `name`, `country_code`, `phone`, `whatsapp_id`, `status` (CONNECTED|DISCONNECTED), `deleted_at`, `created_at`, `updated_at`

### DashboardStats
See [Dashboard Response Structure](#dashboard-requires-auth) above.

---

## Role & Access Matrix

| Role | Permissions |
|------|-------------|
| **super-admin** | Full access, including user management and tenant management |
| **admin** | All features except user/tenant management |
| **agent** | Contacts, conversations, messages, notes, tags, campaigns |
