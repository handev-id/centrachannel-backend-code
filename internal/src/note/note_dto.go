package note

import "github.com/google/uuid"

type CreateNoteRequest struct {
	Text           string    `json:"text" validate:"required"`
	Date           string    `json:"date,omitempty"`
	ConversationID uuid.UUID `json:"conversation_id" validate:"required,uuid"`
}

type UpdateNoteRequest struct {
	Text *string `json:"text,omitempty"`
	Date *string `json:"date,omitempty"`
}
