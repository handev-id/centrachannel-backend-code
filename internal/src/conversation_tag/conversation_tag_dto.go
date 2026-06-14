package conversation_tag

type AttachTagRequest struct {
	TagID int `json:"tag_id" validate:"required"`
}
