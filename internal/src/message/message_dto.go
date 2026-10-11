package message

import (
	"encoding/json"

	"github.com/google/uuid"
)

type ListMessageQuery struct {
	Limit  int       `query:"limit"`
	LastID uuid.UUID `query:"last_id"`
}

type SendMessageRequest struct {
	Text           *string         `json:"text,omitempty"`
	Attachment     json.RawMessage `json:"attachment,omitempty"`
	SenderID       uuid.UUID       `json:"sender_id" validate:"required,uuid"`
	SenderType     string          `json:"sender_type" validate:"required,oneof=contact user ai"`
	ConversationID uuid.UUID       `json:"conversation_id" validate:"required,uuid"`
}

type UpdateMessageStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=sent delivered read unread failed"`
}
