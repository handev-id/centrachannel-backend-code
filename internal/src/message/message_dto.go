package message

import "encoding/json"

type ListMessageQuery struct {
	Page   int `query:"page"`
	Limit  int `query:"limit"`
	LastID int `query:"last_id"`
}

type SendMessageRequest struct {
	Text       *string         `json:"text,omitempty"`
	Attachment json.RawMessage `json:"attachment,omitempty"`
	SenderID   int             `json:"sender_id" validate:"required"`
	SenderType string          `json:"sender_type" validate:"required,oneof=contact user ai"`
}

type UpdateMessageStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=sent delivered read unread"`
}


