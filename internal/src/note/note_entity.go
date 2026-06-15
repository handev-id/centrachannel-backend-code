package note

import (
	"time"

	"centrachannel/internal/utils"
)

type Note struct {
	ID             int          `json:"id"`
	TenantID       int          `json:"tenant_id"`
	Text           string       `json:"text"`
	Date           *time.Time   `json:"date,omitempty"`
	ConversationID int          `json:"conversation_id"`
	UserID         *int         `json:"user_id,omitempty"`
	DeletedAt      utils.NullableTime `json:"deleted_at,omitempty"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
}
