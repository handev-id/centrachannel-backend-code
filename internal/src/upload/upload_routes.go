package upload

import "github.com/gofiber/fiber/v3"

func RegisterRoutes(group fiber.Router, handler *UploadHandler) {
	group.Post("/", handler.Upload)
}
