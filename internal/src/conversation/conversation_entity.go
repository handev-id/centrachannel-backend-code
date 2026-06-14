package conversation

import (
	"encoding/json"
	"time"
)

type Conversation struct {
	ID          int              `json:"id"`
	TenantID    int              `json:"tenant_id"`
	Status      string           `json:"status"`
	ProfileID   int              `json:"profile_id"`
	AgentID     *int             `json:"agent_id,omitempty"`
	ChannelID   int              `json:"channel_id"`
	LastAgentID *int             `json:"last_agent_id,omitempty"`
	UnreadCount int              `json:"unread_count"`
	LastMessage json.RawMessage  `json:"last_message,omitempty"`
	LastActivity *time.Time      `json:"last_activity,omitempty"`
	LastSeen    *time.Time       `json:"last_seen,omitempty"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}
