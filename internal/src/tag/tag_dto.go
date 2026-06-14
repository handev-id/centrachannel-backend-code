package tag

type CreateTagRequest struct {
	Name  string  `json:"name" validate:"required,max=255"`
	Color *string `json:"color,omitempty"`
}

type UpdateTagRequest struct {
	Name  *string `json:"name,omitempty"`
	Color *string `json:"color,omitempty"`
}
