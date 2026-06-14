package upload

import (
	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/di"
	"centrachannel/internal/utils/response"
)

type UploadHandler struct {
	service UploadService
}

func NewUploadHandler(c *di.Container) *UploadHandler {
	service := NewUploadService(c.Config.StorageURL)
	return &UploadHandler{service: service}
}

func (h *UploadHandler) Upload(c fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return response.BadRequest(c, "File is required", nil)
	}

	result, err := h.service.Upload(file)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	return response.Created(c, "File uploaded", result)
}
