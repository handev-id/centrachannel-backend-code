package conversation

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Conversation struct {
	ID           uuid.UUID            `json:"id"`
	TenantID     uuid.UUID            `json:"tenant_id"`
	Status       string               `json:"status"`
	ProfileID    uuid.UUID            `json:"profile_id"`
	AgentID      *uuid.UUID           `json:"agent_id,omitempty"`
	ChannelID    uuid.UUID            `json:"channel_id"`
	LastAgentID  *uuid.UUID           `json:"last_agent_id,omitempty"`
	UnreadCount  int                  `json:"unread_count"`
	LastMessage  json.RawMessage      `json:"last_message,omitempty"`
	LastActivity *time.Time           `json:"last_activity,omitempty"`
	LastSeen     *time.Time           `json:"last_seen,omitempty"`
	CreatedAt    time.Time            `json:"created_at"`
	UpdatedAt    time.Time            `json:"updated_at"`
	Contact      *ConversationContact `json:"contact,omitempty"`
}

type ConversationContact struct {
	ID        uuid.UUID       `json:"id"`
	FirstName string          `json:"first_name"`
	LastName  *string         `json:"last_name,omitempty"`
	Avatar    json.RawMessage `json:"avatar,omitempty"`
}
