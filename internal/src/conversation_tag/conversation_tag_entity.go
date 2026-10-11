package conversation_tag

import (
	"time"

	"github.com/google/uuid"
)

type ConversationTag struct {
	ID             uuid.UUID `json:"id"`
	TenantID       uuid.UUID `json:"tenant_id"`
	ConversationID uuid.UUID `json:"conversation_id"`
	TagID          uuid.UUID `json:"tag_id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
