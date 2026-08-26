package note

type CreateNoteRequest struct {
	Text           string `json:"text" validate:"required"`
	Date           string `json:"date,omitempty"`
	ConversationID int    `json:"conversation_id" validate:"required"`
}

type UpdateNoteRequest struct {
	Text *string `json:"text,omitempty"`
	Date *string `json:"date,omitempty"`
}
