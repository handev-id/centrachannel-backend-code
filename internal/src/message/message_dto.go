package message

import "encoding/json"

type ListMessageQuery struct {
	Limit  int `query:"limit"`
	LastID int `query:"last_id"`
}

type SendMessageRequest struct {
	Text           *string         `json:"text,omitempty"`
	Attachment     json.RawMessage `json:"attachment,omitempty"`
	SenderID       int             `json:"sender_id" validate:"required"`
	SenderType     string          `json:"sender_type" validate:"required,oneof=contact user ai"`
	ConversationID int             `json:"conversation_id" validate:"required"`
}

type UpdateMessageStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=sent delivered read unread"`
}


