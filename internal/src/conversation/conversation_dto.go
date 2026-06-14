package conversation

type ListConversationQuery struct {
	Page      int    `query:"page"`
	Limit     int    `query:"limit"`
	Status    string `query:"status"`
	ChannelID int    `query:"channel_id"`
	AgentID   int    `query:"agent_id"`
	Search    string `query:"search"`
	SortBy    string `query:"sort_by"`
}

type CreateConversationRequest struct {
	ProfileID int `json:"profile_id" validate:"required"`
	ChannelID int `json:"channel_id" validate:"required"`
	AgentID   int `json:"agent_id,omitempty"`
}

type PaginatedResponse struct {
	Meta PaginationMeta `json:"meta"`
	Data interface{}    `json:"data"`
}

type PaginationMeta struct {
	Total       int `json:"total"`
	PerPage     int `json:"per_page"`
	CurrentPage int `json:"current_page"`
	LastPage    int `json:"last_page"`
	From        int `json:"from"`
	To          int `json:"to"`
}
