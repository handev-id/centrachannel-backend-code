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

type CursorPaginationMeta struct {
	LastID  int  `json:"last_id"`
	HasMore bool `json:"has_more"`
}

type CursorPaginatedResponse struct {
	Meta CursorPaginationMeta `json:"meta_pagination"`
	Data interface{}         `json:"data"`
}
