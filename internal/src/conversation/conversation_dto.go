package conversation

import "github.com/google/uuid"

type ListConversationQuery struct {
	Limit        int       `query:"limit"`
	Status       string    `query:"status"`
	ChannelID    uuid.UUID `query:"channel_id"`
	AgentID      uuid.UUID `query:"agent_id"`
	Search       string    `query:"search"`
	SortBy       string    `query:"sort_by"`
	LastActivity string    `query:"last_activity"`
	LastID       uuid.UUID `query:"last_id"`
}

type CreateConversationRequest struct {
	ProfileID uuid.UUID `json:"profile_id" validate:"required,uuid"`
	ChannelID uuid.UUID `json:"channel_id" validate:"required,uuid"`
	AgentID   uuid.UUID `json:"agent_id,omitempty"`
}
