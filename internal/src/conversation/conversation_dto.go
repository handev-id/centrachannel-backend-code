package conversation

type ListConversationQuery struct {
	Page         int    `query:"page"`
	Limit        int    `query:"limit"`
	Status       string `query:"status"`
	ChannelID    int    `query:"channel_id"`
	AgentID      int    `query:"agent_id"`
	Search       string `query:"search"`
	SortBy       string `query:"sort_by"`
	LastActivity string `query:"last_activity"`
	LastID       int    `query:"last_id"`
}

type CreateConversationRequest struct {
	ProfileID int `json:"profile_id" validate:"required"`
	ChannelID int `json:"channel_id" validate:"required"`
	AgentID   int `json:"agent_id,omitempty"`
}


