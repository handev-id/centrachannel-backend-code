package conversation_tag

import "time"

type ConversationTag struct {
	ID             int       `json:"id"`
	TenantID       int       `json:"tenant_id"`
	ConversationID int       `json:"conversation_id"`
	TagID          int       `json:"tag_id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
