package upload

import "mime/multipart"

type UploadRequest struct {
	File *multipart.FileHeader `form:"file" validate:"required"`
}
