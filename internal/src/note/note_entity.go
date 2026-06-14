package note

import (
	"database/sql"
	"time"
)

type Note struct {
	ID             int          `json:"id"`
	TenantID       int          `json:"tenant_id"`
	Text           string       `json:"text"`
	Date           *time.Time   `json:"date,omitempty"`
	ConversationID int          `json:"conversation_id"`
	UserID         *int         `json:"user_id,omitempty"`
	DeletedAt      sql.NullTime `json:"deleted_at,omitempty"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
}
