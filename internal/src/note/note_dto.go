package note

type CreateNoteRequest struct {
	Text string `json:"text" validate:"required"`
	Date string `json:"date,omitempty"`
}

type UpdateNoteRequest struct {
	Text *string `json:"text,omitempty"`
	Date *string `json:"date,omitempty"`
}
