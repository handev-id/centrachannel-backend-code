package note

import (
	"time"

	"github.com/google/uuid"

	"centrachannel/internal/utils"
)

type Note struct {
	ID             uuid.UUID          `json:"id"`
	TenantID       uuid.UUID          `json:"tenant_id"`
	Text           string             `json:"text"`
	Date           *time.Time         `json:"date,omitempty"`
	ConversationID uuid.UUID          `json:"conversation_id"`
	UserID         *uuid.UUID         `json:"user_id,omitempty"`
	DeletedAt      utils.NullableTime `json:"deleted_at,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
}
