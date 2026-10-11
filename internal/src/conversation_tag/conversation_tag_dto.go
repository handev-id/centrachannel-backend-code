package conversation_tag

import "github.com/google/uuid"

type AttachTagRequest struct {
	TagID uuid.UUID `json:"tag_id" validate:"required,uuid"`
}
